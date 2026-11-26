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
			return compareDates(a, b) == 0
		case tagFloat:
			return a.Float() == b.Float()
		case tagString, tagSymbol:
			// Both tags already prove that ptr/aux encode a plain Go string.
			// Keep this dominant comparison path out of AppendString: that
			// general converter carries a large type switch and stack frame for
			// compressed strings, BSON, lists, and numeric formatting.
			return unsafe.String(a.ptr, int(auxVal(a.aux))) == unsafe.String(b.ptr, int(auxVal(b.aux)))
		case tagCString, tagConcat:
			if ta == tagCString {
				return cstringEqual(a, b)
			}
			return equalBufferText(a, b, false)
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
			if value, ok := ParseDateValue(b.String()); ok {
				return compareDates(a, value) == 0
			}
		}
		if tb == tagInt {
			return a.DateNanosecond() == 0 && TagDateDecodeUnix(auxVal(a.aux)) == b.Int()
		}
		return a.Float() == b.Float()
	case tagInt:
		if tb == tagFloat {
			return float64(a.Int()) == b.Float()
		}
		if tb == tagString || tb == tagSymbol {
			return a.Int() == b.Int()
		}
		if tb == tagDate {
			return b.DateNanosecond() == 0 && a.Int() == TagDateDecodeUnix(auxVal(b.aux))
		}
		if tb == tagBool {
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
			if value, ok := ParseDateValue(a.String()); ok {
				return compareDates(value, b) == 0
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
	case tagCString, tagConcat:
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
			return NewBool(compareDates(a, b) == 0)
		case tagFloat:
			return NewBool(a.Float() == b.Float())
		case tagString, tagSymbol:
			return NewBool(equalStringValues(a, b, true))
		case tagCString, tagConcat:
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
			if value, ok := ParseDateValue(b.String()); ok {
				return NewBool(compareDates(a, value) == 0)
			}
		}
		if tb == tagInt {
			return NewBool(a.DateNanosecond() == 0 && TagDateDecodeUnix(auxVal(a.aux)) == b.Int())
		}
		return NewBool(a.Float() == b.Float())
	case tagInt:
		if tb == tagDate {
			return NewBool(b.DateNanosecond() == 0 && a.Int() == TagDateDecodeUnix(auxVal(b.aux)))
		}
		if tb == tagFloat {
			return NewBool(float64(a.Int()) == b.Float())
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString || tb == tagConcat {
			return NewBool(numericTextEqualsInt(b, a.Int()))
		}
		return NewBool(a.Int() == b.Int())
	case tagFloat:
		if tb == tagInt {
			return NewBool(a.Float() == float64(b.Int()))
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString || tb == tagConcat {
			return NewBool(a.Float() == b.Float())
		}
		return NewBool(a.Float() == b.Float())
	case tagString, tagSymbol, tagCString, tagConcat:
		if tb == tagDate {
			if value, ok := ParseDateValue(a.String()); ok {
				return NewBool(compareDates(value, b) == 0)
			}
		}
		if tb == tagInt {
			return NewBool(numericTextEqualsInt(a, b.Int()))
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
			return NewBool(numericTextEqualsInt(a, b.Int()))
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
func EqualCollated(a, b Scmer, collation string) Scmer {
	if (a.IsString() || a.IsSymbol() || a.IsBufferString()) && (b.IsString() || b.IsSymbol() || b.IsBufferString()) {
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
		if tb == tagInt {
			return TagDateDecodeUnix(auxVal(a.aux)) < b.Int()
		}
		if tb == tagDate {
			return compareDates(a, b) < 0
		}
		if tb == tagString || tb == tagSymbol {
			if value, ok := ParseDateValue(b.String()); ok {
				return compareDates(a, value) < 0
			}
		}
		return a.Float() < b.Float()
	case tagInt:
		if tb == tagDate {
			seconds := TagDateDecodeUnix(auxVal(b.aux))
			integer := a.Int()
			return integer < seconds || integer == seconds && b.DateNanosecond() > 0
		}
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
	if (a.IsBufferString() || b.IsBufferString()) && (a.IsString() || a.IsSymbol()) && (b.IsString() || b.IsSymbol()) {
		return compareBufferText(a, b, false) < 0
	}
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
	case tagString, tagSymbol, tagCString, tagBString, tagConcat:
		switch tb {
		case tagDate:
			if value, ok := ParseDateValue(a.String()); ok {
				return compareDates(value, b) < 0
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
		case tagCString, tagBString, tagConcat, tagBSON:
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
	var d45 JITValueDesc
	_ = d45
	var d46 JITValueDesc
	_ = d46
	var d65 JITValueDesc
	_ = d65
	var d66 JITValueDesc
	_ = d66
	var d67 JITValueDesc
	_ = d67
	var d89 JITValueDesc
	_ = d89
	var d90 JITValueDesc
	_ = d90
	var d91 JITValueDesc
	_ = d91
	var d116 JITValueDesc
	_ = d116
	var d117 JITValueDesc
	_ = d117
	var d144 JITValueDesc
	_ = d144
	var d145 JITValueDesc
	_ = d145
	var d174 JITValueDesc
	_ = d174
	var d175 JITValueDesc
	_ = d175
	var d206 JITValueDesc
	_ = d206
	var d207 JITValueDesc
	_ = d207
	var d208 JITValueDesc
	_ = d208
	var d209 JITValueDesc
	_ = d209
	var d210 JITValueDesc
	_ = d210
	var d211 JITValueDesc
	_ = d211
	var d212 JITValueDesc
	_ = d212
	var d213 JITValueDesc
	_ = d213
	var d214 JITValueDesc
	_ = d214
	var d215 JITValueDesc
	_ = d215
	var d217 JITValueDesc
	_ = d217
	var d218 JITValueDesc
	_ = d218
	var d219 JITValueDesc
	_ = d219
	var d263 JITValueDesc
	_ = d263
	var d264 JITValueDesc
	_ = d264
	var d265 JITValueDesc
	_ = d265
	var d267 JITValueDesc
	_ = d267
	var d268 JITValueDesc
	_ = d268
	var d269 JITValueDesc
	_ = d269
	var d319 JITValueDesc
	_ = d319
	var d320 JITValueDesc
	_ = d320
	var d321 JITValueDesc
	_ = d321
	var d323 JITValueDesc
	_ = d323
	var d324 JITValueDesc
	_ = d324
	var d325 JITValueDesc
	_ = d325
	var d381 JITValueDesc
	_ = d381
	var d382 JITValueDesc
	_ = d382
	var d383 JITValueDesc
	_ = d383
	var d384 JITValueDesc
	_ = d384
	var d386 JITValueDesc
	_ = d386
	var d387 JITValueDesc
	_ = d387
	var d388 JITValueDesc
	_ = d388
	var d451 JITValueDesc
	_ = d451
	var d452 JITValueDesc
	_ = d452
	var d454 JITValueDesc
	_ = d454
	var d455 JITValueDesc
	_ = d455
	var d456 JITValueDesc
	_ = d456
	var d457 JITValueDesc
	_ = d457
	var d458 JITValueDesc
	_ = d458
	var d460 JITValueDesc
	_ = d460
	var d461 JITValueDesc
	_ = d461
	var d462 JITValueDesc
	_ = d462
	var d535 JITValueDesc
	_ = d535
	var d536 JITValueDesc
	_ = d536
	var d537 JITValueDesc
	_ = d537
	var d538 JITValueDesc
	_ = d538
	var d539 JITValueDesc
	_ = d539
	var d540 JITValueDesc
	_ = d540
	var d541 JITValueDesc
	_ = d541
	var d542 JITValueDesc
	_ = d542
	var d543 JITValueDesc
	_ = d543
	var d544 JITValueDesc
	_ = d544
	var d545 JITValueDesc
	_ = d545
	var d546 JITValueDesc
	_ = d546
	var d715 JITValueDesc
	_ = d715
	var d716 JITValueDesc
	_ = d716
	var d803 JITValueDesc
	_ = d803
	var d804 JITValueDesc
	_ = d804
	var d981 JITValueDesc
	_ = d981
	var d982 JITValueDesc
	_ = d982
	var d983 JITValueDesc
	_ = d983
	var d984 JITValueDesc
	_ = d984
	var d985 JITValueDesc
	_ = d985
	var d986 JITValueDesc
	_ = d986
	var d987 JITValueDesc
	_ = d987
	var d988 JITValueDesc
	_ = d988
	var d990 JITValueDesc
	_ = d990
	var d991 JITValueDesc
	_ = d991
	var d992 JITValueDesc
	_ = d992
	var d993 JITValueDesc
	_ = d993
	var d994 JITValueDesc
	_ = d994
	var d995 JITValueDesc
	_ = d995
	var d997 JITValueDesc
	_ = d997
	var d998 JITValueDesc
	_ = d998
	var d999 JITValueDesc
	_ = d999
	var d1000 JITValueDesc
	_ = d1000
	var d1001 JITValueDesc
	_ = d1001
	var d1003 JITValueDesc
	_ = d1003
	var d1004 JITValueDesc
	_ = d1004
	var d1005 JITValueDesc
	_ = d1005
	var d1116 JITValueDesc
	_ = d1116
	var d1117 JITValueDesc
	_ = d1117
	var d1118 JITValueDesc
	_ = d1118
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
	phiBase0 := ctx.AllocStack(int32(32))
	branchSerial := ctx.branchSerial
	_ = branchSerial
	var bbs [32]BBDescriptor
	bbs[24].PhiBase = int32(phiBase0) + int32(0)
	bbs[26].PhiBase = int32(phiBase0) + int32(16)
	d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
	_ = d1
	d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
	_ = d2
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
	bbpos_0_22 := int32(-1)
	_ = bbpos_0_22
	lbl23 := ctx.ReserveLabel()
	_ = lbl23
	bbpos_0_23 := int32(-1)
	_ = bbpos_0_23
	lbl24 := ctx.ReserveLabel()
	_ = lbl24
	bbpos_0_24 := int32(-1)
	_ = bbpos_0_24
	lbl25 := ctx.ReserveLabel()
	_ = lbl25
	bbpos_0_25 := int32(-1)
	_ = bbpos_0_25
	lbl26 := ctx.ReserveLabel()
	_ = lbl26
	bbpos_0_26 := int32(-1)
	_ = bbpos_0_26
	lbl27 := ctx.ReserveLabel()
	_ = lbl27
	bbpos_0_27 := int32(-1)
	_ = bbpos_0_27
	lbl28 := ctx.ReserveLabel()
	_ = lbl28
	bbpos_0_28 := int32(-1)
	_ = bbpos_0_28
	lbl29 := ctx.ReserveLabel()
	_ = lbl29
	bbpos_0_29 := int32(-1)
	_ = bbpos_0_29
	lbl30 := ctx.ReserveLabel()
	_ = lbl30
	bbpos_0_30 := int32(-1)
	_ = bbpos_0_30
	lbl31 := ctx.ReserveLabel()
	_ = lbl31
	bbpos_0_31 := int32(-1)
	_ = bbpos_0_31
	lbl32 := ctx.ReserveLabel()
	_ = lbl32
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		r0 := ctx.AllocReg()
		r1 := ctx.AllocRegExcept(r0)
		ctx.EmitMovRegImm64(r0, 0)
		ctx.EmitMovRegImm64(r1, 0)
		d3 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d3)
		ctx.BindReg(r1, &d3)
		ctx.StabilizeDescForControlFlow(&d3)
		ctx.SyncDesc(&args[0])
		ctx.FreeDesc(&args[0])
		r2 := ctx.AllocReg()
		r3 := ctx.AllocRegExcept(r2)
		ctx.EmitMovRegImm64(r2, 0)
		ctx.EmitMovRegImm64(r3, 0)
		d4 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
		ctx.BindReg(r2, &d4)
		ctx.BindReg(r3, &d4)
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.SyncDesc(&args[1])
		ctx.FreeDesc(&args[1])
		d5 = args[0]
		_ = d5
		d6 = d5
		d6.ID = 0
		d7 = ctx.EmitGetTagDesc(&d6, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d7)
		ctx.FreeDesc(&d5)
		d8 = args[1]
		_ = d8
		d9 = d8
		d9.ID = 0
		d10 = ctx.EmitGetTagDesc(&d9, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d10)
		ctx.FreeDesc(&d8)
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x0))}
		} else {
			r4 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 0)
			d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
			ctx.BindReg(r4, &d11)
		}
		d12 = d11
		ctx.SyncDesc(&d12)
		if d12.Loc != LocImm && d12.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d12.Loc == LocImm {
			if d12.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d12.Condition, lbl4)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d11)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap13 := d1
			snap14 := d2
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
			d1 = snap13
			d2 = snap14
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d26 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d26) {
			return d26
		}
		ctx.EnsureDesc(&d26)
		ctx.EmitMovToReg(result.Reg, d26)
		result.Type = d26.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x0))}
		} else {
			r5 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 0)
			d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
			ctx.BindReg(r5, &d27)
		}
		d28 = d27
		ctx.SyncDesc(&d28)
		if d28.Loc != LocImm && d28.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d28.Loc == LocImm {
			if d28.Imm.Bool() {
				return bbs[4].Render()
			}
			return bbs[5].Render()
		}
		ctx.EmitJump(d28.Condition, lbl5)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d27)
		ctx.FlushRegisterMoves()
		if !bbs[5].Rendered {
			snap29 := d1
			snap30 := d2
			snap31 := d3
			snap32 := d4
			snap33 := d5
			snap34 := d6
			snap35 := d7
			snap36 := d8
			snap37 := d9
			snap38 := d10
			snap39 := d11
			snap40 := d12
			snap41 := d26
			snap42 := d27
			snap43 := d28
			alloc44 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc44)
			d1 = snap29
			d2 = snap30
			d3 = snap31
			d4 = snap32
			d5 = snap33
			d6 = snap34
			d7 = snap35
			d8 = snap36
			d9 = snap37
			d10 = snap38
			d11 = snap39
			d12 = snap40
			d26 = snap41
			d27 = snap42
			d28 = snap43
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d45 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x0))}
		} else {
			r6 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 0)
			d45 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
			ctx.BindReg(r6, &d45)
		}
		d46 = d45
		ctx.SyncDesc(&d46)
		if d46.Loc != LocImm && d46.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d46.Loc == LocImm {
			if d46.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d46.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d45)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap47 := d1
			snap48 := d2
			snap49 := d3
			snap50 := d4
			snap51 := d5
			snap52 := d6
			snap53 := d7
			snap54 := d8
			snap55 := d9
			snap56 := d10
			snap57 := d11
			snap58 := d12
			snap59 := d26
			snap60 := d27
			snap61 := d28
			snap62 := d45
			snap63 := d46
			alloc64 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc64)
			d1 = snap47
			d2 = snap48
			d3 = snap49
			d4 = snap50
			d5 = snap51
			d6 = snap52
			d7 = snap53
			d8 = snap54
			d9 = snap55
			d10 = snap56
			d11 = snap57
			d12 = snap58
			d26 = snap59
			d27 = snap60
			d28 = snap61
			d45 = snap62
			d46 = snap63
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d65 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d65) {
			return d65
		}
		ctx.EnsureDesc(&d65)
		ctx.EmitMovToReg(result.Reg, d65)
		result.Type = d65.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d66 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x0))}
		} else {
			r7 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 0)
			d66 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
			ctx.BindReg(r7, &d66)
		}
		d67 = d66
		ctx.SyncDesc(&d67)
		if d67.Loc != LocImm && d67.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d67.Loc == LocImm {
			if d67.Imm.Bool() {
				return bbs[6].Render()
			}
			return bbs[7].Render()
		}
		ctx.EmitJump(d67.Condition, lbl7)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d66)
		ctx.FlushRegisterMoves()
		if !bbs[7].Rendered {
			snap68 := d1
			snap69 := d2
			snap70 := d3
			snap71 := d4
			snap72 := d5
			snap73 := d6
			snap74 := d7
			snap75 := d8
			snap76 := d9
			snap77 := d10
			snap78 := d11
			snap79 := d12
			snap80 := d26
			snap81 := d27
			snap82 := d28
			snap83 := d45
			snap84 := d46
			snap85 := d65
			snap86 := d66
			snap87 := d67
			alloc88 := ctx.SnapshotAllocState()
			bbs[7].Render()
			ctx.RestoreAllocState(alloc88)
			d1 = snap68
			d2 = snap69
			d3 = snap70
			d4 = snap71
			d5 = snap72
			d6 = snap73
			d7 = snap74
			d8 = snap75
			d9 = snap76
			d10 = snap77
			d11 = snap78
			d12 = snap79
			d26 = snap80
			d27 = snap81
			d28 = snap82
			d45 = snap83
			d46 = snap84
			d65 = snap85
			d66 = snap86
			d67 = snap87
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d89 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d89) {
			return d89
		}
		ctx.EnsureDesc(&d89)
		ctx.EmitMovToReg(result.Reg, d89)
		result.Type = d89.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d90 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x10))}
		} else {
			r8 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 16)
			d90 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondEqual}
			ctx.BindReg(r8, &d90)
		}
		d91 = d90
		ctx.SyncDesc(&d91)
		if d91.Loc != LocImm && d91.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d91.Loc == LocImm {
			if d91.Imm.Bool() {
				return bbs[8].Render()
			}
			return bbs[10].Render()
		}
		ctx.EmitJump(d91.Condition, lbl9)
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
		}
		ctx.FreeDesc(&d90)
		ctx.FlushRegisterMoves()
		if !bbs[10].Rendered {
			snap92 := d1
			snap93 := d2
			snap94 := d3
			snap95 := d4
			snap96 := d5
			snap97 := d6
			snap98 := d7
			snap99 := d8
			snap100 := d9
			snap101 := d10
			snap102 := d11
			snap103 := d12
			snap104 := d26
			snap105 := d27
			snap106 := d28
			snap107 := d45
			snap108 := d46
			snap109 := d65
			snap110 := d66
			snap111 := d67
			snap112 := d89
			snap113 := d90
			snap114 := d91
			alloc115 := ctx.SnapshotAllocState()
			bbs[10].Render()
			ctx.RestoreAllocState(alloc115)
			d1 = snap92
			d2 = snap93
			d3 = snap94
			d4 = snap95
			d5 = snap96
			d6 = snap97
			d7 = snap98
			d8 = snap99
			d9 = snap100
			d10 = snap101
			d11 = snap102
			d12 = snap103
			d26 = snap104
			d27 = snap105
			d28 = snap106
			d45 = snap107
			d46 = snap108
			d65 = snap109
			d66 = snap110
			d67 = snap111
			d89 = snap112
			d90 = snap113
			d91 = snap114
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d116 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x4))}
		} else {
			r9 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 4)
			d116 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondEqual}
			ctx.BindReg(r9, &d116)
		}
		d117 = d116
		ctx.SyncDesc(&d117)
		if d117.Loc != LocImm && d117.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d117.Loc == LocImm {
			if d117.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitJump(d117.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d116)
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap118 := d1
			snap119 := d2
			snap120 := d3
			snap121 := d4
			snap122 := d5
			snap123 := d6
			snap124 := d7
			snap125 := d8
			snap126 := d9
			snap127 := d10
			snap128 := d11
			snap129 := d12
			snap130 := d26
			snap131 := d27
			snap132 := d28
			snap133 := d45
			snap134 := d46
			snap135 := d65
			snap136 := d66
			snap137 := d67
			snap138 := d89
			snap139 := d90
			snap140 := d91
			snap141 := d116
			snap142 := d117
			alloc143 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc143)
			d1 = snap118
			d2 = snap119
			d3 = snap120
			d4 = snap121
			d5 = snap122
			d6 = snap123
			d7 = snap124
			d8 = snap125
			d9 = snap126
			d10 = snap127
			d11 = snap128
			d12 = snap129
			d26 = snap130
			d27 = snap131
			d28 = snap132
			d45 = snap133
			d46 = snap134
			d65 = snap135
			d66 = snap136
			d67 = snap137
			d89 = snap138
			d90 = snap139
			d91 = snap140
			d116 = snap141
			d117 = snap142
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d144 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x10))}
		} else {
			r10 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 16)
			d144 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
			ctx.BindReg(r10, &d144)
		}
		d145 = d144
		ctx.SyncDesc(&d145)
		if d145.Loc != LocImm && d145.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d145.Loc == LocImm {
			if d145.Imm.Bool() {
				return bbs[21].Render()
			}
			return bbs[22].Render()
		}
		ctx.EmitJump(d145.Condition, lbl22)
		if bbs[22].Rendered {
			ctx.EmitJmp(lbl23)
		}
		ctx.FreeDesc(&d144)
		ctx.FlushRegisterMoves()
		if !bbs[22].Rendered {
			snap146 := d1
			snap147 := d2
			snap148 := d3
			snap149 := d4
			snap150 := d5
			snap151 := d6
			snap152 := d7
			snap153 := d8
			snap154 := d9
			snap155 := d10
			snap156 := d11
			snap157 := d12
			snap158 := d26
			snap159 := d27
			snap160 := d28
			snap161 := d45
			snap162 := d46
			snap163 := d65
			snap164 := d66
			snap165 := d67
			snap166 := d89
			snap167 := d90
			snap168 := d91
			snap169 := d116
			snap170 := d117
			snap171 := d144
			snap172 := d145
			alloc173 := ctx.SnapshotAllocState()
			bbs[22].Render()
			ctx.RestoreAllocState(alloc173)
			d1 = snap146
			d2 = snap147
			d3 = snap148
			d4 = snap149
			d5 = snap150
			d6 = snap151
			d7 = snap152
			d8 = snap153
			d9 = snap154
			d10 = snap155
			d11 = snap156
			d12 = snap157
			d26 = snap158
			d27 = snap159
			d28 = snap160
			d45 = snap161
			d46 = snap162
			d65 = snap163
			d66 = snap164
			d67 = snap165
			d89 = snap166
			d90 = snap167
			d91 = snap168
			d116 = snap169
			d117 = snap170
			d144 = snap171
			d145 = snap172
		}
		if !bbs[21].Rendered {
			return bbs[21].Render()
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d174 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x4))}
		} else {
			r11 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 4)
			d174 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondEqual}
			ctx.BindReg(r11, &d174)
		}
		d175 = d174
		ctx.SyncDesc(&d175)
		if d175.Loc != LocImm && d175.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d175.Loc == LocImm {
			if d175.Imm.Bool() {
				return bbs[9].Render()
			}
			return bbs[20].Render()
		}
		ctx.EmitJump(d175.Condition, lbl10)
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
		}
		ctx.FreeDesc(&d174)
		ctx.FlushRegisterMoves()
		if !bbs[20].Rendered {
			snap176 := d1
			snap177 := d2
			snap178 := d3
			snap179 := d4
			snap180 := d5
			snap181 := d6
			snap182 := d7
			snap183 := d8
			snap184 := d9
			snap185 := d10
			snap186 := d11
			snap187 := d12
			snap188 := d26
			snap189 := d27
			snap190 := d28
			snap191 := d45
			snap192 := d46
			snap193 := d65
			snap194 := d66
			snap195 := d67
			snap196 := d89
			snap197 := d90
			snap198 := d91
			snap199 := d116
			snap200 := d117
			snap201 := d144
			snap202 := d145
			snap203 := d174
			snap204 := d175
			alloc205 := ctx.SnapshotAllocState()
			bbs[20].Render()
			ctx.RestoreAllocState(alloc205)
			d1 = snap176
			d2 = snap177
			d3 = snap178
			d4 = snap179
			d5 = snap180
			d6 = snap181
			d7 = snap182
			d8 = snap183
			d9 = snap184
			d10 = snap185
			d11 = snap186
			d12 = snap187
			d26 = snap188
			d27 = snap189
			d28 = snap190
			d45 = snap191
			d46 = snap192
			d65 = snap193
			d66 = snap194
			d67 = snap195
			d89 = snap196
			d90 = snap197
			d91 = snap198
			d116 = snap199
			d117 = snap200
			d144 = snap201
			d145 = snap202
			d174 = snap203
			d175 = snap204
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&args[0])
		if args[0].Loc == LocImm {
			_, auxWord := args[0].Imm.RawWords()
			d206 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
		} else {
			if args[0].Loc != LocRegPair {
				panic("jitgen: desc field base is not LocRegPair")
			}
			r12 := ctx.AllocReg()
			ctx.EmitMovRegReg(r12, args[0].Reg2)
			d206 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
			ctx.BindReg(r12, &d206)
		}
		ctx.EnsureDesc(&d206)
		d207 = d206
		_ = d207
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl33 := ctx.ReserveLabel()
		_ = lbl33
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl33)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d207)
		if d207.Loc == LocImm {
			d208 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d207.Imm.Int()) >> 8))}
		} else {
			r13 := ctx.AllocRegExcept(d207.Reg)
			ctx.EmitMovRegReg(r13, d207.Reg)
			ctx.EmitShrRegImm8(r13, 8)
			d208 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
			ctx.BindReg(r13, &d208)
		}
		if d208.Loc == LocReg && d207.Loc == LocReg && d208.Reg == d207.Reg {
			ctx.TransferReg(d207.Reg)
			d207.Loc = LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d208)
		ctx.FreeDesc(&d206)
		ctx.EnsureDesc(&d208)
		d209 = d208
		_ = d209
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl34 := ctx.ReserveLabel()
		_ = lbl34
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl34)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d209)
		if d209.Loc == LocImm {
			d210 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d209.Imm.Int() & 35184372088831)}
		} else {
			r14 := ctx.AllocRegExcept(d209.Reg)
			ctx.EmitMovRegReg(r14, d209.Reg)
			ctx.EmitMovRegImm64(ctx.ScratchReg, 0x1fffffffffff)
			ctx.EmitAndInt64(r14, ctx.ScratchReg)
			d210 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
			ctx.BindReg(r14, &d210)
		}
		if d210.Loc == LocReg && d209.Loc == LocReg && d210.Reg == d209.Reg {
			ctx.TransferReg(d209.Reg)
			d209.Loc = LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d210)
		if d210.Loc == LocImm {
			d211 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d210.Imm.Int()) << 19))}
		} else {
			ctx.EmitShlRegImm8(d210.Reg, 19)
			d211 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d210.Reg}
			ctx.BindReg(d210.Reg, &d211)
		}
		if d211.Loc == LocReg && d210.Loc == LocReg && d211.Reg == d210.Reg {
			ctx.TransferReg(d210.Reg)
			d210.Loc = LocNone
		}
		ctx.FreeDesc(&d210)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d211)
		ctx.EnsureDesc(&d211)
		if d211.Loc == LocImm {
			d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d211.Imm.Int()))))}
		} else {
			r15 := ctx.AllocReg()
			ctx.EmitMovRegReg(r15, d211.Reg)
			d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
			ctx.BindReg(r15, &d212)
		}
		ctx.FreeDesc(&d211)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == LocImm {
			d213 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d212.Imm.Int()) >> 19))}
		} else {
			ctx.EmitShrRegImm8(d212.Reg, 19)
			d213 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d212.Reg}
			ctx.BindReg(d212.Reg, &d213)
		}
		if d213.Loc == LocReg && d212.Loc == LocReg && d213.Reg == d212.Reg {
			ctx.TransferReg(d212.Reg)
			d212.Loc = LocNone
		}
		ctx.FreeDesc(&d212)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d213)
		ctx.FreeDesc(&d208)
		d214 = args[1]
		_ = d214
		if d214.Loc != LocImm {
			ctx.EnsureDesc(&d214)
		}
		if d214.Loc == LocImm {
			d215 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d214.Imm.Int())}
		} else if d214.Type == tagInt && d214.Loc == LocRegPair {
			ctx.FreeReg(d214.Reg)
			d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d214.Reg2}
			ctx.BindReg(d214.Reg2, &d215)
			ctx.BindReg(d214.Reg2, &d215)
		} else if d214.Type == tagInt && d214.Loc == LocReg {
			d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d214.Reg}
			ctx.BindReg(d214.Reg, &d215)
			ctx.BindReg(d214.Reg, &d215)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d214)
			d215 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d215.Type = tagInt
			ctx.BindReg(d215.Reg, &d215)
		}
		ctx.FreeDesc(&d214)
		ctx.EnsureDesc(&d213)
		resultTarget216 := false
		_ = resultTarget216
		ctx.EnsureDesc(&d215)
		ctx.EnsureDescsTogether(&d213, &d215)
		if d213.Loc == LocImm && d215.Loc == LocImm {
			d217 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d213.Imm.Int() < d215.Imm.Int())}
		} else if d215.Loc == LocImm {
			r16 := ctx.AllocReg()
			if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d213.Reg, int32(d215.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d215.Imm.Int()))
				ctx.EmitCmpInt64(d213.Reg, ctx.ScratchReg)
			}
			d217 = ctx.DeferBooleanFlags(r16, CondSignedLess)
			ctx.BindReg(r16, &d217)
		} else if d213.Loc == LocImm {
			r17 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d213.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d215.Reg)
			d217 = ctx.DeferBooleanFlags(r17, CondSignedLess)
			ctx.BindReg(r17, &d217)
		} else {
			r18 := ctx.AllocReg()
			ctx.EmitCmpInt64(d213.Reg, d215.Reg)
			d217 = ctx.DeferBooleanFlags(r18, CondSignedLess)
			ctx.BindReg(r18, &d217)
		}
		ctx.FreeDesc(&d213)
		ctx.FreeDesc(&d215)
		ctx.SyncDesc(&d217)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d217) {
			return d217
		}
		ctx.EnsureDesc(&d217)
		ctx.EmitMovToReg(result.Reg, d217)
		result.Type = d217.Type
		ctx.EmitJmp(lbl0)
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d218 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x10))}
		} else {
			r19 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 16)
			d218 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondEqual}
			ctx.BindReg(r19, &d218)
		}
		d219 = d218
		ctx.SyncDesc(&d219)
		if d219.Loc != LocImm && d219.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d219.Loc == LocImm {
			if d219.Imm.Bool() {
				return bbs[13].Render()
			}
			return bbs[14].Render()
		}
		ctx.EmitJump(d219.Condition, lbl14)
		if bbs[14].Rendered {
			ctx.EmitJmp(lbl15)
		}
		ctx.FreeDesc(&d218)
		ctx.FlushRegisterMoves()
		if !bbs[14].Rendered {
			snap220 := d1
			snap221 := d2
			snap222 := d3
			snap223 := d4
			snap224 := d5
			snap225 := d6
			snap226 := d7
			snap227 := d8
			snap228 := d9
			snap229 := d10
			snap230 := d11
			snap231 := d12
			snap232 := d26
			snap233 := d27
			snap234 := d28
			snap235 := d45
			snap236 := d46
			snap237 := d65
			snap238 := d66
			snap239 := d67
			snap240 := d89
			snap241 := d90
			snap242 := d91
			snap243 := d116
			snap244 := d117
			snap245 := d144
			snap246 := d145
			snap247 := d174
			snap248 := d175
			snap249 := d206
			snap250 := d207
			snap251 := d208
			snap252 := d209
			snap253 := d210
			snap254 := d211
			snap255 := d212
			snap256 := d213
			snap257 := d214
			snap258 := d215
			snap259 := d217
			snap260 := d218
			snap261 := d219
			alloc262 := ctx.SnapshotAllocState()
			bbs[14].Render()
			ctx.RestoreAllocState(alloc262)
			d1 = snap220
			d2 = snap221
			d3 = snap222
			d4 = snap223
			d5 = snap224
			d6 = snap225
			d7 = snap226
			d8 = snap227
			d9 = snap228
			d10 = snap229
			d11 = snap230
			d12 = snap231
			d26 = snap232
			d27 = snap233
			d28 = snap234
			d45 = snap235
			d46 = snap236
			d65 = snap237
			d66 = snap238
			d67 = snap239
			d89 = snap240
			d90 = snap241
			d91 = snap242
			d116 = snap243
			d117 = snap244
			d144 = snap245
			d145 = snap246
			d174 = snap247
			d175 = snap248
			d206 = snap249
			d207 = snap250
			d208 = snap251
			d209 = snap252
			d210 = snap253
			d211 = snap254
			d212 = snap255
			d213 = snap256
			d214 = snap257
			d215 = snap258
			d217 = snap259
			d218 = snap260
			d219 = snap261
		}
		if !bbs[13].Rendered {
			return bbs[13].Render()
		}
		return result
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d263 = args[0]
		_ = d263
		d264 = args[1]
		_ = d264
		d263 = JITPrepareScmerGoArg(ctx, d263)
		d264 = JITPrepareScmerGoArg(ctx, d264)
		ctx.SyncDesc(&d263)
		ctx.SyncDesc(&d264)
		d265 = ctx.EmitGoCallScalar(jitGoABI_d69f4c85104836f4, []JITValueDesc{d263, d264}, 1)
		d265.NoHeapPointer = true
		ctx.BindReg(d265.Reg, &d265)
		ctx.FreeDesc(&d263)
		ctx.FreeDesc(&d264)
		ctx.EnsureDesc(&d265)
		resultTarget266 := false
		_ = resultTarget266
		if d265.Loc == LocImm {
			d267 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d265.Imm.Int() < 0)}
		} else {
			r20 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d265.Reg, 0)
			d267 = ctx.DeferBooleanFlags(r20, CondSignedLess)
			ctx.BindReg(r20, &d267)
		}
		ctx.FreeDesc(&d265)
		ctx.SyncDesc(&d267)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d267) {
			return d267
		}
		ctx.EnsureDesc(&d267)
		ctx.EmitMovToReg(result.Reg, d267)
		result.Type = d267.Type
		ctx.EmitJmp(lbl0)
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d268 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x1))}
		} else {
			r21 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 1)
			d268 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondEqual}
			ctx.BindReg(r21, &d268)
		}
		d269 = d268
		ctx.SyncDesc(&d269)
		if d269.Loc != LocImm && d269.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d269.Loc == LocImm {
			if d269.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[17].Render()
		}
		ctx.EmitJump(d269.Condition, lbl16)
		if bbs[17].Rendered {
			ctx.EmitJmp(lbl18)
		}
		ctx.FreeDesc(&d268)
		ctx.FlushRegisterMoves()
		if !bbs[17].Rendered {
			snap270 := d1
			snap271 := d2
			snap272 := d3
			snap273 := d4
			snap274 := d5
			snap275 := d6
			snap276 := d7
			snap277 := d8
			snap278 := d9
			snap279 := d10
			snap280 := d11
			snap281 := d12
			snap282 := d26
			snap283 := d27
			snap284 := d28
			snap285 := d45
			snap286 := d46
			snap287 := d65
			snap288 := d66
			snap289 := d67
			snap290 := d89
			snap291 := d90
			snap292 := d91
			snap293 := d116
			snap294 := d117
			snap295 := d144
			snap296 := d145
			snap297 := d174
			snap298 := d175
			snap299 := d206
			snap300 := d207
			snap301 := d208
			snap302 := d209
			snap303 := d210
			snap304 := d211
			snap305 := d212
			snap306 := d213
			snap307 := d214
			snap308 := d215
			snap309 := d217
			snap310 := d218
			snap311 := d219
			snap312 := d263
			snap313 := d264
			snap314 := d265
			snap315 := d267
			snap316 := d268
			snap317 := d269
			alloc318 := ctx.SnapshotAllocState()
			bbs[17].Render()
			ctx.RestoreAllocState(alloc318)
			d1 = snap270
			d2 = snap271
			d3 = snap272
			d4 = snap273
			d5 = snap274
			d6 = snap275
			d7 = snap276
			d8 = snap277
			d9 = snap278
			d10 = snap279
			d11 = snap280
			d12 = snap281
			d26 = snap282
			d27 = snap283
			d28 = snap284
			d45 = snap285
			d46 = snap286
			d65 = snap287
			d66 = snap288
			d67 = snap289
			d89 = snap290
			d90 = snap291
			d91 = snap292
			d116 = snap293
			d117 = snap294
			d144 = snap295
			d145 = snap296
			d174 = snap297
			d175 = snap298
			d206 = snap299
			d207 = snap300
			d208 = snap301
			d209 = snap302
			d210 = snap303
			d211 = snap304
			d212 = snap305
			d213 = snap306
			d214 = snap307
			d215 = snap308
			d217 = snap309
			d218 = snap310
			d219 = snap311
			d263 = snap312
			d264 = snap313
			d265 = snap314
			d267 = snap315
			d268 = snap316
			d269 = snap317
		}
		if !bbs[15].Rendered {
			return bbs[15].Render()
		}
		return result
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d319 = args[1]
		_ = d319
		d321 = d319
		ctx.SyncDesc(&d321)
		if d321.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d321.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d321.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d321 = tmpScalar
		}
		d321 = JITPrepareScmerGoArg(ctx, d321)
		if d321.Loc != LocRegPair && d321.Loc != LocStackPair && d321.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d320 = ctx.EmitGoCallScalar(jitGoABI_0cbff6f55ee96f6d, []JITValueDesc{d321}, 2)
		ctx.FreeDesc(&d319)
		ctx.EnsureDesc(&d320)
		if d320.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d320.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d320.Imm)
			ptrWord, _ := d320.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d320.Imm.String())))
			d320 = tmpPair
		} else if d320.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d320.Type, Reg: ctx.AllocRegExcept(d320.Reg), Reg2: ctx.AllocRegExcept(d320.Reg)}
			switch d320.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d320)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d320)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d320)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d320)
			d320 = tmpPair
		}
		if d320.Loc != LocRegPair && d320.Loc != LocStackPair && d320.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateValue arg0)")
		}
		ctx.SyncDesc(&d320)
		callResults322 := JITEmitGoCallResults(ctx, jitGoABI_58434e8dfe39c263, []JITValueDesc{d320})
		d323 = callResults322[0]
		_ = d323
		d324 = callResults322[1]
		_ = d324
		ctx.StabilizeDescForControlFlow(&d323)
		d325 = d324
		ctx.EnsureDesc(&d325)
		if d325.Loc != LocImm && d325.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d325.Loc == LocImm {
			if d325.Imm.Bool() {
				return bbs[18].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitCmpRegImm32(d325.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl19)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap326 := d1
			snap327 := d2
			snap328 := d3
			snap329 := d4
			snap330 := d5
			snap331 := d6
			snap332 := d7
			snap333 := d8
			snap334 := d9
			snap335 := d10
			snap336 := d11
			snap337 := d12
			snap338 := d26
			snap339 := d27
			snap340 := d28
			snap341 := d45
			snap342 := d46
			snap343 := d65
			snap344 := d66
			snap345 := d67
			snap346 := d89
			snap347 := d90
			snap348 := d91
			snap349 := d116
			snap350 := d117
			snap351 := d144
			snap352 := d145
			snap353 := d174
			snap354 := d175
			snap355 := d206
			snap356 := d207
			snap357 := d208
			snap358 := d209
			snap359 := d210
			snap360 := d211
			snap361 := d212
			snap362 := d213
			snap363 := d214
			snap364 := d215
			snap365 := d217
			snap366 := d218
			snap367 := d219
			snap368 := d263
			snap369 := d264
			snap370 := d265
			snap371 := d267
			snap372 := d268
			snap373 := d269
			snap374 := d319
			snap375 := d320
			snap376 := d321
			snap377 := d323
			snap378 := d324
			snap379 := d325
			alloc380 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc380)
			d1 = snap326
			d2 = snap327
			d3 = snap328
			d4 = snap329
			d5 = snap330
			d6 = snap331
			d7 = snap332
			d8 = snap333
			d9 = snap334
			d10 = snap335
			d11 = snap336
			d12 = snap337
			d26 = snap338
			d27 = snap339
			d28 = snap340
			d45 = snap341
			d46 = snap342
			d65 = snap343
			d66 = snap344
			d67 = snap345
			d89 = snap346
			d90 = snap347
			d91 = snap348
			d116 = snap349
			d117 = snap350
			d144 = snap351
			d145 = snap352
			d174 = snap353
			d175 = snap354
			d206 = snap355
			d207 = snap356
			d208 = snap357
			d209 = snap358
			d210 = snap359
			d211 = snap360
			d212 = snap361
			d213 = snap362
			d214 = snap363
			d215 = snap364
			d217 = snap365
			d218 = snap366
			d219 = snap367
			d263 = snap368
			d264 = snap369
			d265 = snap370
			d267 = snap371
			d268 = snap372
			d269 = snap373
			d319 = snap374
			d320 = snap375
			d321 = snap376
			d323 = snap377
			d324 = snap378
			d325 = snap379
		}
		if !bbs[18].Rendered {
			return bbs[18].Render()
		}
		return result
		ctx.FreeDesc(&d324)
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d381 = args[0]
		_ = d381
		d382 = ctx.EmitFloatDesc(d381)
		ctx.FreeDesc(&d381)
		d383 = args[1]
		_ = d383
		d384 = ctx.EmitFloatDesc(d383)
		ctx.FreeDesc(&d383)
		ctx.EnsureDesc(&d382)
		resultTarget385 := false
		_ = resultTarget385
		ctx.EnsureDesc(&d384)
		ctx.EnsureDescsTogether(&d382, &d384)
		if d382.Loc == LocImm && d384.Loc == LocImm {
			d386 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d382.Imm.Float() < d384.Imm.Float())}
		} else if d384.Loc == LocImm {
			var r22 Reg
			if result.Loc == LocReg && result.Reg != d382.Reg {
				r22 = result.Reg
				resultTarget385 = true
			} else {
				r22 = ctx.AllocRegExcept(d382.Reg)
			}
			_, yBits := d384.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d382.Reg)
			d386 = ctx.DeferBooleanFlags(r22, CondUnsignedAbove)
			ctx.BindReg(r22, &d386)
		} else if d382.Loc == LocImm {
			var r23 Reg
			if result.Loc == LocReg && result.Reg != d384.Reg {
				r23 = result.Reg
				resultTarget385 = true
			} else {
				r23 = ctx.AllocRegExcept(d384.Reg)
			}
			_, xBits := d382.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d384.Reg, ctx.ScratchReg)
			d386 = ctx.DeferBooleanFlags(r23, CondUnsignedAbove)
			ctx.BindReg(r23, &d386)
		} else {
			var r24 Reg
			if result.Loc == LocReg && result.Reg != d382.Reg && result.Reg != d384.Reg {
				r24 = result.Reg
				resultTarget385 = true
			} else {
				r24 = ctx.AllocRegExcept(d382.Reg, d384.Reg)
			}
			ctx.EmitCmpFloat64(d384.Reg, d382.Reg)
			d386 = ctx.DeferBooleanFlags(r24, CondUnsignedAbove)
			ctx.BindReg(r24, &d386)
		}
		ctx.FreeDesc(&d382)
		ctx.FreeDesc(&d384)
		ctx.SyncDesc(&d386)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d386) {
			return d386
		}
		ctx.EnsureDesc(&d386)
		ctx.EmitMovToReg(result.Reg, d386)
		result.Type = d386.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d387 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x2))}
		} else {
			r25 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 2)
			d387 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondEqual}
			ctx.BindReg(r25, &d387)
		}
		d388 = d387
		ctx.SyncDesc(&d388)
		if d388.Loc != LocImm && d388.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d388.Loc == LocImm {
			if d388.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitJump(d388.Condition, lbl16)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d387)
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap389 := d1
			snap390 := d2
			snap391 := d3
			snap392 := d4
			snap393 := d5
			snap394 := d6
			snap395 := d7
			snap396 := d8
			snap397 := d9
			snap398 := d10
			snap399 := d11
			snap400 := d12
			snap401 := d26
			snap402 := d27
			snap403 := d28
			snap404 := d45
			snap405 := d46
			snap406 := d65
			snap407 := d66
			snap408 := d67
			snap409 := d89
			snap410 := d90
			snap411 := d91
			snap412 := d116
			snap413 := d117
			snap414 := d144
			snap415 := d145
			snap416 := d174
			snap417 := d175
			snap418 := d206
			snap419 := d207
			snap420 := d208
			snap421 := d209
			snap422 := d210
			snap423 := d211
			snap424 := d212
			snap425 := d213
			snap426 := d214
			snap427 := d215
			snap428 := d217
			snap429 := d218
			snap430 := d219
			snap431 := d263
			snap432 := d264
			snap433 := d265
			snap434 := d267
			snap435 := d268
			snap436 := d269
			snap437 := d319
			snap438 := d320
			snap439 := d321
			snap440 := d323
			snap441 := d324
			snap442 := d325
			snap443 := d381
			snap444 := d382
			snap445 := d383
			snap446 := d384
			snap447 := d386
			snap448 := d387
			snap449 := d388
			alloc450 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc450)
			d1 = snap389
			d2 = snap390
			d3 = snap391
			d4 = snap392
			d5 = snap393
			d6 = snap394
			d7 = snap395
			d8 = snap396
			d9 = snap397
			d10 = snap398
			d11 = snap399
			d12 = snap400
			d26 = snap401
			d27 = snap402
			d28 = snap403
			d45 = snap404
			d46 = snap405
			d65 = snap406
			d66 = snap407
			d67 = snap408
			d89 = snap409
			d90 = snap410
			d91 = snap411
			d116 = snap412
			d117 = snap413
			d144 = snap414
			d145 = snap415
			d174 = snap416
			d175 = snap417
			d206 = snap418
			d207 = snap419
			d208 = snap420
			d209 = snap421
			d210 = snap422
			d211 = snap423
			d212 = snap424
			d213 = snap425
			d214 = snap426
			d215 = snap427
			d217 = snap428
			d218 = snap429
			d219 = snap430
			d263 = snap431
			d264 = snap432
			d265 = snap433
			d267 = snap434
			d268 = snap435
			d269 = snap436
			d319 = snap437
			d320 = snap438
			d321 = snap439
			d323 = snap440
			d324 = snap441
			d325 = snap442
			d381 = snap443
			d382 = snap444
			d383 = snap445
			d384 = snap446
			d386 = snap447
			d387 = snap448
			d388 = snap449
		}
		if !bbs[15].Rendered {
			return bbs[15].Render()
		}
		return result
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d451 = args[0]
		_ = d451
		d451 = JITPrepareScmerGoArg(ctx, d451)
		d323 = JITPrepareScmerGoArg(ctx, d323)
		ctx.SyncDesc(&d451)
		ctx.SyncDesc(&d323)
		d452 = ctx.EmitGoCallScalar(jitGoABI_d69f4c85104836f4, []JITValueDesc{d451, d323}, 1)
		d452.NoHeapPointer = true
		ctx.BindReg(d452.Reg, &d452)
		ctx.FreeDesc(&d451)
		ctx.EnsureDesc(&d452)
		resultTarget453 := false
		_ = resultTarget453
		if d452.Loc == LocImm {
			d454 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d452.Imm.Int() < 0)}
		} else {
			r26 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d452.Reg, 0)
			d454 = ctx.DeferBooleanFlags(r26, CondSignedLess)
			ctx.BindReg(r26, &d454)
		}
		ctx.FreeDesc(&d452)
		ctx.SyncDesc(&d454)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d454) {
			return d454
		}
		ctx.EnsureDesc(&d454)
		ctx.EmitMovToReg(result.Reg, d454)
		result.Type = d454.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d455 = args[0]
		_ = d455
		d456 = ctx.EmitFloatDesc(d455)
		ctx.FreeDesc(&d455)
		d457 = args[1]
		_ = d457
		d458 = ctx.EmitFloatDesc(d457)
		ctx.FreeDesc(&d457)
		ctx.EnsureDesc(&d456)
		resultTarget459 := false
		_ = resultTarget459
		ctx.EnsureDesc(&d458)
		ctx.EnsureDescsTogether(&d456, &d458)
		if d456.Loc == LocImm && d458.Loc == LocImm {
			d460 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d456.Imm.Float() < d458.Imm.Float())}
		} else if d458.Loc == LocImm {
			var r27 Reg
			if result.Loc == LocReg && result.Reg != d456.Reg {
				r27 = result.Reg
				resultTarget459 = true
			} else {
				r27 = ctx.AllocRegExcept(d456.Reg)
			}
			_, yBits := d458.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d456.Reg)
			d460 = ctx.DeferBooleanFlags(r27, CondUnsignedAbove)
			ctx.BindReg(r27, &d460)
		} else if d456.Loc == LocImm {
			var r28 Reg
			if result.Loc == LocReg && result.Reg != d458.Reg {
				r28 = result.Reg
				resultTarget459 = true
			} else {
				r28 = ctx.AllocRegExcept(d458.Reg)
			}
			_, xBits := d456.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d458.Reg, ctx.ScratchReg)
			d460 = ctx.DeferBooleanFlags(r28, CondUnsignedAbove)
			ctx.BindReg(r28, &d460)
		} else {
			var r29 Reg
			if result.Loc == LocReg && result.Reg != d456.Reg && result.Reg != d458.Reg {
				r29 = result.Reg
				resultTarget459 = true
			} else {
				r29 = ctx.AllocRegExcept(d456.Reg, d458.Reg)
			}
			ctx.EmitCmpFloat64(d458.Reg, d456.Reg)
			d460 = ctx.DeferBooleanFlags(r29, CondUnsignedAbove)
			ctx.BindReg(r29, &d460)
		}
		ctx.FreeDesc(&d456)
		ctx.FreeDesc(&d458)
		ctx.SyncDesc(&d460)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d460) {
			return d460
		}
		ctx.EnsureDesc(&d460)
		ctx.EmitMovToReg(result.Reg, d460)
		result.Type = d460.Type
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d461 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x3))}
		} else {
			r30 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 3)
			d461 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondEqual}
			ctx.BindReg(r30, &d461)
		}
		d462 = d461
		ctx.SyncDesc(&d462)
		if d462.Loc != LocImm && d462.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d462.Loc == LocImm {
			if d462.Imm.Bool() {
				return bbs[19].Render()
			}
			return bbs[30].Render()
		}
		ctx.EmitJump(d462.Condition, lbl20)
		if bbs[30].Rendered {
			ctx.EmitJmp(lbl31)
		}
		ctx.FreeDesc(&d461)
		ctx.FlushRegisterMoves()
		if !bbs[30].Rendered {
			snap463 := d1
			snap464 := d2
			snap465 := d3
			snap466 := d4
			snap467 := d5
			snap468 := d6
			snap469 := d7
			snap470 := d8
			snap471 := d9
			snap472 := d10
			snap473 := d11
			snap474 := d12
			snap475 := d26
			snap476 := d27
			snap477 := d28
			snap478 := d45
			snap479 := d46
			snap480 := d65
			snap481 := d66
			snap482 := d67
			snap483 := d89
			snap484 := d90
			snap485 := d91
			snap486 := d116
			snap487 := d117
			snap488 := d144
			snap489 := d145
			snap490 := d174
			snap491 := d175
			snap492 := d206
			snap493 := d207
			snap494 := d208
			snap495 := d209
			snap496 := d210
			snap497 := d211
			snap498 := d212
			snap499 := d213
			snap500 := d214
			snap501 := d215
			snap502 := d217
			snap503 := d218
			snap504 := d219
			snap505 := d263
			snap506 := d264
			snap507 := d265
			snap508 := d267
			snap509 := d268
			snap510 := d269
			snap511 := d319
			snap512 := d320
			snap513 := d321
			snap514 := d323
			snap515 := d324
			snap516 := d325
			snap517 := d381
			snap518 := d382
			snap519 := d383
			snap520 := d384
			snap521 := d386
			snap522 := d387
			snap523 := d388
			snap524 := d451
			snap525 := d452
			snap526 := d454
			snap527 := d455
			snap528 := d456
			snap529 := d457
			snap530 := d458
			snap531 := d460
			snap532 := d461
			snap533 := d462
			alloc534 := ctx.SnapshotAllocState()
			bbs[30].Render()
			ctx.RestoreAllocState(alloc534)
			d1 = snap463
			d2 = snap464
			d3 = snap465
			d4 = snap466
			d5 = snap467
			d6 = snap468
			d7 = snap469
			d8 = snap470
			d9 = snap471
			d10 = snap472
			d11 = snap473
			d12 = snap474
			d26 = snap475
			d27 = snap476
			d28 = snap477
			d45 = snap478
			d46 = snap479
			d65 = snap480
			d66 = snap481
			d67 = snap482
			d89 = snap483
			d90 = snap484
			d91 = snap485
			d116 = snap486
			d117 = snap487
			d144 = snap488
			d145 = snap489
			d174 = snap490
			d175 = snap491
			d206 = snap492
			d207 = snap493
			d208 = snap494
			d209 = snap495
			d210 = snap496
			d211 = snap497
			d212 = snap498
			d213 = snap499
			d214 = snap500
			d215 = snap501
			d217 = snap502
			d218 = snap503
			d219 = snap504
			d263 = snap505
			d264 = snap506
			d265 = snap507
			d267 = snap508
			d268 = snap509
			d269 = snap510
			d319 = snap511
			d320 = snap512
			d321 = snap513
			d323 = snap514
			d324 = snap515
			d325 = snap516
			d381 = snap517
			d382 = snap518
			d383 = snap519
			d384 = snap520
			d386 = snap521
			d387 = snap522
			d388 = snap523
			d451 = snap524
			d452 = snap525
			d454 = snap526
			d455 = snap527
			d456 = snap528
			d457 = snap529
			d458 = snap530
			d460 = snap531
			d461 = snap532
			d462 = snap533
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
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&args[1])
		if args[1].Loc == LocImm {
			_, auxWord := args[1].Imm.RawWords()
			d535 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
		} else {
			if args[1].Loc != LocRegPair {
				panic("jitgen: desc field base is not LocRegPair")
			}
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, args[1].Reg2)
			d535 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r31}
			ctx.BindReg(r31, &d535)
		}
		ctx.EnsureDesc(&d535)
		d536 = d535
		_ = d536
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl35 := ctx.ReserveLabel()
		_ = lbl35
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl35)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d536)
		if d536.Loc == LocImm {
			d537 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d536.Imm.Int()) >> 8))}
		} else {
			r32 := ctx.AllocRegExcept(d536.Reg)
			ctx.EmitMovRegReg(r32, d536.Reg)
			ctx.EmitShrRegImm8(r32, 8)
			d537 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r32}
			ctx.BindReg(r32, &d537)
		}
		if d537.Loc == LocReg && d536.Loc == LocReg && d537.Reg == d536.Reg {
			ctx.TransferReg(d536.Reg)
			d536.Loc = LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d537)
		ctx.FreeDesc(&d535)
		ctx.EnsureDesc(&d537)
		d538 = d537
		_ = d538
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl36 := ctx.ReserveLabel()
		_ = lbl36
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl36)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d538)
		if d538.Loc == LocImm {
			d539 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d538.Imm.Int() & 35184372088831)}
		} else {
			r33 := ctx.AllocRegExcept(d538.Reg)
			ctx.EmitMovRegReg(r33, d538.Reg)
			ctx.EmitMovRegImm64(ctx.ScratchReg, 0x1fffffffffff)
			ctx.EmitAndInt64(r33, ctx.ScratchReg)
			d539 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
			ctx.BindReg(r33, &d539)
		}
		if d539.Loc == LocReg && d538.Loc == LocReg && d539.Reg == d538.Reg {
			ctx.TransferReg(d538.Reg)
			d538.Loc = LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d539)
		if d539.Loc == LocImm {
			d540 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d539.Imm.Int()) << 19))}
		} else {
			ctx.EmitShlRegImm8(d539.Reg, 19)
			d540 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d539.Reg}
			ctx.BindReg(d539.Reg, &d540)
		}
		if d540.Loc == LocReg && d539.Loc == LocReg && d540.Reg == d539.Reg {
			ctx.TransferReg(d539.Reg)
			d539.Loc = LocNone
		}
		ctx.FreeDesc(&d539)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d540)
		ctx.EnsureDesc(&d540)
		if d540.Loc == LocImm {
			d541 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d540.Imm.Int()))))}
		} else {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegReg(r34, d540.Reg)
			d541 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r34}
			ctx.BindReg(r34, &d541)
		}
		ctx.FreeDesc(&d540)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d541)
		if d541.Loc == LocImm {
			d542 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d541.Imm.Int()) >> 19))}
		} else {
			ctx.EmitShrRegImm8(d541.Reg, 19)
			d542 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d541.Reg}
			ctx.BindReg(d541.Reg, &d542)
		}
		if d542.Loc == LocReg && d541.Loc == LocReg && d542.Reg == d541.Reg {
			ctx.TransferReg(d541.Reg)
			d541.Loc = LocNone
		}
		ctx.FreeDesc(&d541)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d542)
		ctx.StabilizeDescForControlFlow(&d542)
		ctx.FreeDesc(&d537)
		d543 = args[0]
		_ = d543
		if d543.Loc != LocImm {
			ctx.EnsureDesc(&d543)
		}
		if d543.Loc == LocImm {
			d544 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d543.Imm.Int())}
		} else if d543.Type == tagInt && d543.Loc == LocRegPair {
			ctx.FreeReg(d543.Reg)
			d544 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d543.Reg2}
			ctx.BindReg(d543.Reg2, &d544)
			ctx.BindReg(d543.Reg2, &d544)
		} else if d543.Type == tagInt && d543.Loc == LocReg {
			d544 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d543.Reg}
			ctx.BindReg(d543.Reg, &d544)
			ctx.BindReg(d543.Reg, &d544)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d543)
			d544 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d544.Type = tagInt
			ctx.BindReg(d544.Reg, &d544)
		}
		ctx.StabilizeDescForControlFlow(&d544)
		ctx.FreeDesc(&d543)
		ctx.EnsureDesc(&d544)
		ctx.EnsureDesc(&d542)
		ctx.EnsureDescsTogether(&d544, &d542)
		if d544.Loc == LocImm && d542.Loc == LocImm {
			d545 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d544.Imm.Int() < d542.Imm.Int())}
		} else if d542.Loc == LocImm {
			r35 := ctx.AllocRegExcept(d544.Reg)
			if d542.Imm.Int() >= -2147483648 && d542.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d544.Reg, int32(d542.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d542.Imm.Int()))
				ctx.EmitCmpInt64(d544.Reg, ctx.ScratchReg)
			}
			d545 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedLess}
			ctx.BindReg(r35, &d545)
		} else if d544.Loc == LocImm {
			r36 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d544.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d542.Reg)
			d545 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedLess}
			ctx.BindReg(r36, &d545)
		} else {
			r37 := ctx.AllocRegExcept(d544.Reg)
			ctx.EmitCmpInt64(d544.Reg, d542.Reg)
			d545 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedLess}
			ctx.BindReg(r37, &d545)
		}
		d546 = d545
		ctx.SyncDesc(&d546)
		if d546.Loc != LocImm && d546.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d546.Loc == LocImm {
			if d546.Imm.Bool() {
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
				return bbs[24].Render()
			}
			return bbs[23].Render()
		}
		lbl37 := ctx.ReserveLabel()
		ctx.EmitJump(d546.Condition, lbl37)
		ctx.EmitJmp(lbl24)
		ctx.FreeDesc(&d545)
		snap547 := d1
		snap548 := d2
		snap549 := d3
		snap550 := d4
		snap551 := d5
		snap552 := d6
		snap553 := d7
		snap554 := d8
		snap555 := d9
		snap556 := d10
		snap557 := d11
		snap558 := d12
		snap559 := d26
		snap560 := d27
		snap561 := d28
		snap562 := d45
		snap563 := d46
		snap564 := d65
		snap565 := d66
		snap566 := d67
		snap567 := d89
		snap568 := d90
		snap569 := d91
		snap570 := d116
		snap571 := d117
		snap572 := d144
		snap573 := d145
		snap574 := d174
		snap575 := d175
		snap576 := d206
		snap577 := d207
		snap578 := d208
		snap579 := d209
		snap580 := d210
		snap581 := d211
		snap582 := d212
		snap583 := d213
		snap584 := d214
		snap585 := d215
		snap586 := d217
		snap587 := d218
		snap588 := d219
		snap589 := d263
		snap590 := d264
		snap591 := d265
		snap592 := d267
		snap593 := d268
		snap594 := d269
		snap595 := d319
		snap596 := d320
		snap597 := d321
		snap598 := d323
		snap599 := d324
		snap600 := d325
		snap601 := d381
		snap602 := d382
		snap603 := d383
		snap604 := d384
		snap605 := d386
		snap606 := d387
		snap607 := d388
		snap608 := d451
		snap609 := d452
		snap610 := d454
		snap611 := d455
		snap612 := d456
		snap613 := d457
		snap614 := d458
		snap615 := d460
		snap616 := d461
		snap617 := d462
		snap618 := d535
		snap619 := d536
		snap620 := d537
		snap621 := d538
		snap622 := d539
		snap623 := d540
		snap624 := d541
		snap625 := d542
		snap626 := d543
		snap627 := d544
		snap628 := d545
		snap629 := d546
		alloc630 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl37)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
		ctx.EmitJmp(lbl25)
		ctx.RestoreAllocState(alloc630)
		d1 = snap547
		d2 = snap548
		d3 = snap549
		d4 = snap550
		d5 = snap551
		d6 = snap552
		d7 = snap553
		d8 = snap554
		d9 = snap555
		d10 = snap556
		d11 = snap557
		d12 = snap558
		d26 = snap559
		d27 = snap560
		d28 = snap561
		d45 = snap562
		d46 = snap563
		d65 = snap564
		d66 = snap565
		d67 = snap566
		d89 = snap567
		d90 = snap568
		d91 = snap569
		d116 = snap570
		d117 = snap571
		d144 = snap572
		d145 = snap573
		d174 = snap574
		d175 = snap575
		d206 = snap576
		d207 = snap577
		d208 = snap578
		d209 = snap579
		d210 = snap580
		d211 = snap581
		d212 = snap582
		d213 = snap583
		d214 = snap584
		d215 = snap585
		d217 = snap586
		d218 = snap587
		d219 = snap588
		d263 = snap589
		d264 = snap590
		d265 = snap591
		d267 = snap592
		d268 = snap593
		d269 = snap594
		d319 = snap595
		d320 = snap596
		d321 = snap597
		d323 = snap598
		d324 = snap599
		d325 = snap600
		d381 = snap601
		d382 = snap602
		d383 = snap603
		d384 = snap604
		d386 = snap605
		d387 = snap606
		d388 = snap607
		d451 = snap608
		d452 = snap609
		d454 = snap610
		d455 = snap611
		d456 = snap612
		d457 = snap613
		d458 = snap614
		d460 = snap615
		d461 = snap616
		d462 = snap617
		d535 = snap618
		d536 = snap619
		d537 = snap620
		d538 = snap621
		d539 = snap622
		d540 = snap623
		d541 = snap624
		d542 = snap625
		d543 = snap626
		d544 = snap627
		d545 = snap628
		d546 = snap629
		if !bbs[24].Rendered {
			snap631 := d1
			snap632 := d2
			snap633 := d3
			snap634 := d4
			snap635 := d5
			snap636 := d6
			snap637 := d7
			snap638 := d8
			snap639 := d9
			snap640 := d10
			snap641 := d11
			snap642 := d12
			snap643 := d26
			snap644 := d27
			snap645 := d28
			snap646 := d45
			snap647 := d46
			snap648 := d65
			snap649 := d66
			snap650 := d67
			snap651 := d89
			snap652 := d90
			snap653 := d91
			snap654 := d116
			snap655 := d117
			snap656 := d144
			snap657 := d145
			snap658 := d174
			snap659 := d175
			snap660 := d206
			snap661 := d207
			snap662 := d208
			snap663 := d209
			snap664 := d210
			snap665 := d211
			snap666 := d212
			snap667 := d213
			snap668 := d214
			snap669 := d215
			snap670 := d217
			snap671 := d218
			snap672 := d219
			snap673 := d263
			snap674 := d264
			snap675 := d265
			snap676 := d267
			snap677 := d268
			snap678 := d269
			snap679 := d319
			snap680 := d320
			snap681 := d321
			snap682 := d323
			snap683 := d324
			snap684 := d325
			snap685 := d381
			snap686 := d382
			snap687 := d383
			snap688 := d384
			snap689 := d386
			snap690 := d387
			snap691 := d388
			snap692 := d451
			snap693 := d452
			snap694 := d454
			snap695 := d455
			snap696 := d456
			snap697 := d457
			snap698 := d458
			snap699 := d460
			snap700 := d461
			snap701 := d462
			snap702 := d535
			snap703 := d536
			snap704 := d537
			snap705 := d538
			snap706 := d539
			snap707 := d540
			snap708 := d541
			snap709 := d542
			snap710 := d543
			snap711 := d544
			snap712 := d545
			snap713 := d546
			alloc714 := ctx.SnapshotAllocState()
			bbs[24].Render()
			ctx.RestoreAllocState(alloc714)
			d1 = snap631
			d2 = snap632
			d3 = snap633
			d4 = snap634
			d5 = snap635
			d6 = snap636
			d7 = snap637
			d8 = snap638
			d9 = snap639
			d10 = snap640
			d11 = snap641
			d12 = snap642
			d26 = snap643
			d27 = snap644
			d28 = snap645
			d45 = snap646
			d46 = snap647
			d65 = snap648
			d66 = snap649
			d67 = snap650
			d89 = snap651
			d90 = snap652
			d91 = snap653
			d116 = snap654
			d117 = snap655
			d144 = snap656
			d145 = snap657
			d174 = snap658
			d175 = snap659
			d206 = snap660
			d207 = snap661
			d208 = snap662
			d209 = snap663
			d210 = snap664
			d211 = snap665
			d212 = snap666
			d213 = snap667
			d214 = snap668
			d215 = snap669
			d217 = snap670
			d218 = snap671
			d219 = snap672
			d263 = snap673
			d264 = snap674
			d265 = snap675
			d267 = snap676
			d268 = snap677
			d269 = snap678
			d319 = snap679
			d320 = snap680
			d321 = snap681
			d323 = snap682
			d324 = snap683
			d325 = snap684
			d381 = snap685
			d382 = snap686
			d383 = snap687
			d384 = snap688
			d386 = snap689
			d387 = snap690
			d388 = snap691
			d451 = snap692
			d452 = snap693
			d454 = snap694
			d455 = snap695
			d456 = snap696
			d457 = snap697
			d458 = snap698
			d460 = snap699
			d461 = snap700
			d462 = snap701
			d535 = snap702
			d536 = snap703
			d537 = snap704
			d538 = snap705
			d539 = snap706
			d540 = snap707
			d541 = snap708
			d542 = snap709
			d543 = snap710
			d544 = snap711
			d545 = snap712
			d546 = snap713
		}
		if !bbs[23].Rendered {
			return bbs[23].Render()
		}
		return result
		return result
	}
	bbs[22].Render = func() JITValueDesc {
		if bbs[22].Rendered {
			ctx.EmitJmp(lbl23)
			return result
		}
		bbs[22].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_22 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		if d10.Loc == LocImm {
			d715 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d10.Imm.Int()) == uint64(0x4))}
		} else {
			r38 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpRegImm32(d10.Reg, 4)
			d715 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r38, Condition: CondEqual}
			ctx.BindReg(r38, &d715)
		}
		d716 = d715
		ctx.SyncDesc(&d716)
		if d716.Loc != LocImm && d716.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d716.Loc == LocImm {
			if d716.Imm.Bool() {
				return bbs[27].Render()
			}
			return bbs[28].Render()
		}
		ctx.EmitJump(d716.Condition, lbl28)
		if bbs[28].Rendered {
			ctx.EmitJmp(lbl29)
		}
		ctx.FreeDesc(&d715)
		ctx.FlushRegisterMoves()
		if !bbs[28].Rendered {
			snap717 := d1
			snap718 := d2
			snap719 := d3
			snap720 := d4
			snap721 := d5
			snap722 := d6
			snap723 := d7
			snap724 := d8
			snap725 := d9
			snap726 := d10
			snap727 := d11
			snap728 := d12
			snap729 := d26
			snap730 := d27
			snap731 := d28
			snap732 := d45
			snap733 := d46
			snap734 := d65
			snap735 := d66
			snap736 := d67
			snap737 := d89
			snap738 := d90
			snap739 := d91
			snap740 := d116
			snap741 := d117
			snap742 := d144
			snap743 := d145
			snap744 := d174
			snap745 := d175
			snap746 := d206
			snap747 := d207
			snap748 := d208
			snap749 := d209
			snap750 := d210
			snap751 := d211
			snap752 := d212
			snap753 := d213
			snap754 := d214
			snap755 := d215
			snap756 := d217
			snap757 := d218
			snap758 := d219
			snap759 := d263
			snap760 := d264
			snap761 := d265
			snap762 := d267
			snap763 := d268
			snap764 := d269
			snap765 := d319
			snap766 := d320
			snap767 := d321
			snap768 := d323
			snap769 := d324
			snap770 := d325
			snap771 := d381
			snap772 := d382
			snap773 := d383
			snap774 := d384
			snap775 := d386
			snap776 := d387
			snap777 := d388
			snap778 := d451
			snap779 := d452
			snap780 := d454
			snap781 := d455
			snap782 := d456
			snap783 := d457
			snap784 := d458
			snap785 := d460
			snap786 := d461
			snap787 := d462
			snap788 := d535
			snap789 := d536
			snap790 := d537
			snap791 := d538
			snap792 := d539
			snap793 := d540
			snap794 := d541
			snap795 := d542
			snap796 := d543
			snap797 := d544
			snap798 := d545
			snap799 := d546
			snap800 := d715
			snap801 := d716
			alloc802 := ctx.SnapshotAllocState()
			bbs[28].Render()
			ctx.RestoreAllocState(alloc802)
			d1 = snap717
			d2 = snap718
			d3 = snap719
			d4 = snap720
			d5 = snap721
			d6 = snap722
			d7 = snap723
			d8 = snap724
			d9 = snap725
			d10 = snap726
			d11 = snap727
			d12 = snap728
			d26 = snap729
			d27 = snap730
			d28 = snap731
			d45 = snap732
			d46 = snap733
			d65 = snap734
			d66 = snap735
			d67 = snap736
			d89 = snap737
			d90 = snap738
			d91 = snap739
			d116 = snap740
			d117 = snap741
			d144 = snap742
			d145 = snap743
			d174 = snap744
			d175 = snap745
			d206 = snap746
			d207 = snap747
			d208 = snap748
			d209 = snap749
			d210 = snap750
			d211 = snap751
			d212 = snap752
			d213 = snap753
			d214 = snap754
			d215 = snap755
			d217 = snap756
			d218 = snap757
			d219 = snap758
			d263 = snap759
			d264 = snap760
			d265 = snap761
			d267 = snap762
			d268 = snap763
			d269 = snap764
			d319 = snap765
			d320 = snap766
			d321 = snap767
			d323 = snap768
			d324 = snap769
			d325 = snap770
			d381 = snap771
			d382 = snap772
			d383 = snap773
			d384 = snap774
			d386 = snap775
			d387 = snap776
			d388 = snap777
			d451 = snap778
			d452 = snap779
			d454 = snap780
			d455 = snap781
			d456 = snap782
			d457 = snap783
			d458 = snap784
			d460 = snap785
			d461 = snap786
			d462 = snap787
			d535 = snap788
			d536 = snap789
			d537 = snap790
			d538 = snap791
			d539 = snap792
			d540 = snap793
			d541 = snap794
			d542 = snap795
			d543 = snap796
			d544 = snap797
			d545 = snap798
			d546 = snap799
			d715 = snap800
			d716 = snap801
		}
		if !bbs[27].Rendered {
			return bbs[27].Render()
		}
		return result
		return result
	}
	bbs[23].Render = func() JITValueDesc {
		if bbs[23].Rendered {
			ctx.EmitJmp(lbl24)
			return result
		}
		bbs[23].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_23 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl24)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d544)
		ctx.EnsureDesc(&d542)
		ctx.EnsureDescsTogether(&d544, &d542)
		if d544.Loc == LocImm && d542.Loc == LocImm {
			d803 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d544.Imm.Int() == d542.Imm.Int())}
		} else if d542.Loc == LocImm {
			r39 := ctx.AllocRegExcept(d544.Reg)
			if d542.Imm.Int() >= -2147483648 && d542.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d544.Reg, int32(d542.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d542.Imm.Int()))
				ctx.EmitCmpInt64(d544.Reg, ctx.ScratchReg)
			}
			d803 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r39, Condition: CondEqual}
			ctx.BindReg(r39, &d803)
		} else if d544.Loc == LocImm {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d544.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d542.Reg)
			d803 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r40, Condition: CondEqual}
			ctx.BindReg(r40, &d803)
		} else {
			r41 := ctx.AllocRegExcept(d544.Reg)
			ctx.EmitCmpInt64(d544.Reg, d542.Reg)
			d803 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r41, Condition: CondEqual}
			ctx.BindReg(r41, &d803)
		}
		d804 = d803
		ctx.SyncDesc(&d804)
		if d804.Loc != LocImm && d804.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d804.Loc == LocImm {
			if d804.Imm.Bool() {
				return bbs[25].Render()
			}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
			return bbs[26].Render()
		}
		lbl38 := ctx.ReserveLabel()
		ctx.EmitJump(d804.Condition, lbl26)
		ctx.EmitJmp(lbl38)
		ctx.FreeDesc(&d803)
		snap805 := d1
		snap806 := d2
		snap807 := d3
		snap808 := d4
		snap809 := d5
		snap810 := d6
		snap811 := d7
		snap812 := d8
		snap813 := d9
		snap814 := d10
		snap815 := d11
		snap816 := d12
		snap817 := d26
		snap818 := d27
		snap819 := d28
		snap820 := d45
		snap821 := d46
		snap822 := d65
		snap823 := d66
		snap824 := d67
		snap825 := d89
		snap826 := d90
		snap827 := d91
		snap828 := d116
		snap829 := d117
		snap830 := d144
		snap831 := d145
		snap832 := d174
		snap833 := d175
		snap834 := d206
		snap835 := d207
		snap836 := d208
		snap837 := d209
		snap838 := d210
		snap839 := d211
		snap840 := d212
		snap841 := d213
		snap842 := d214
		snap843 := d215
		snap844 := d217
		snap845 := d218
		snap846 := d219
		snap847 := d263
		snap848 := d264
		snap849 := d265
		snap850 := d267
		snap851 := d268
		snap852 := d269
		snap853 := d319
		snap854 := d320
		snap855 := d321
		snap856 := d323
		snap857 := d324
		snap858 := d325
		snap859 := d381
		snap860 := d382
		snap861 := d383
		snap862 := d384
		snap863 := d386
		snap864 := d387
		snap865 := d388
		snap866 := d451
		snap867 := d452
		snap868 := d454
		snap869 := d455
		snap870 := d456
		snap871 := d457
		snap872 := d458
		snap873 := d460
		snap874 := d461
		snap875 := d462
		snap876 := d535
		snap877 := d536
		snap878 := d537
		snap879 := d538
		snap880 := d539
		snap881 := d540
		snap882 := d541
		snap883 := d542
		snap884 := d543
		snap885 := d544
		snap886 := d545
		snap887 := d546
		snap888 := d715
		snap889 := d716
		snap890 := d803
		snap891 := d804
		alloc892 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl38)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
		ctx.EmitJmp(lbl27)
		ctx.RestoreAllocState(alloc892)
		d1 = snap805
		d2 = snap806
		d3 = snap807
		d4 = snap808
		d5 = snap809
		d6 = snap810
		d7 = snap811
		d8 = snap812
		d9 = snap813
		d10 = snap814
		d11 = snap815
		d12 = snap816
		d26 = snap817
		d27 = snap818
		d28 = snap819
		d45 = snap820
		d46 = snap821
		d65 = snap822
		d66 = snap823
		d67 = snap824
		d89 = snap825
		d90 = snap826
		d91 = snap827
		d116 = snap828
		d117 = snap829
		d144 = snap830
		d145 = snap831
		d174 = snap832
		d175 = snap833
		d206 = snap834
		d207 = snap835
		d208 = snap836
		d209 = snap837
		d210 = snap838
		d211 = snap839
		d212 = snap840
		d213 = snap841
		d214 = snap842
		d215 = snap843
		d217 = snap844
		d218 = snap845
		d219 = snap846
		d263 = snap847
		d264 = snap848
		d265 = snap849
		d267 = snap850
		d268 = snap851
		d269 = snap852
		d319 = snap853
		d320 = snap854
		d321 = snap855
		d323 = snap856
		d324 = snap857
		d325 = snap858
		d381 = snap859
		d382 = snap860
		d383 = snap861
		d384 = snap862
		d386 = snap863
		d387 = snap864
		d388 = snap865
		d451 = snap866
		d452 = snap867
		d454 = snap868
		d455 = snap869
		d456 = snap870
		d457 = snap871
		d458 = snap872
		d460 = snap873
		d461 = snap874
		d462 = snap875
		d535 = snap876
		d536 = snap877
		d537 = snap878
		d538 = snap879
		d539 = snap880
		d540 = snap881
		d541 = snap882
		d542 = snap883
		d543 = snap884
		d544 = snap885
		d545 = snap886
		d546 = snap887
		d715 = snap888
		d716 = snap889
		d803 = snap890
		d804 = snap891
		if !bbs[26].Rendered {
			snap893 := d1
			snap894 := d2
			snap895 := d3
			snap896 := d4
			snap897 := d5
			snap898 := d6
			snap899 := d7
			snap900 := d8
			snap901 := d9
			snap902 := d10
			snap903 := d11
			snap904 := d12
			snap905 := d26
			snap906 := d27
			snap907 := d28
			snap908 := d45
			snap909 := d46
			snap910 := d65
			snap911 := d66
			snap912 := d67
			snap913 := d89
			snap914 := d90
			snap915 := d91
			snap916 := d116
			snap917 := d117
			snap918 := d144
			snap919 := d145
			snap920 := d174
			snap921 := d175
			snap922 := d206
			snap923 := d207
			snap924 := d208
			snap925 := d209
			snap926 := d210
			snap927 := d211
			snap928 := d212
			snap929 := d213
			snap930 := d214
			snap931 := d215
			snap932 := d217
			snap933 := d218
			snap934 := d219
			snap935 := d263
			snap936 := d264
			snap937 := d265
			snap938 := d267
			snap939 := d268
			snap940 := d269
			snap941 := d319
			snap942 := d320
			snap943 := d321
			snap944 := d323
			snap945 := d324
			snap946 := d325
			snap947 := d381
			snap948 := d382
			snap949 := d383
			snap950 := d384
			snap951 := d386
			snap952 := d387
			snap953 := d388
			snap954 := d451
			snap955 := d452
			snap956 := d454
			snap957 := d455
			snap958 := d456
			snap959 := d457
			snap960 := d458
			snap961 := d460
			snap962 := d461
			snap963 := d462
			snap964 := d535
			snap965 := d536
			snap966 := d537
			snap967 := d538
			snap968 := d539
			snap969 := d540
			snap970 := d541
			snap971 := d542
			snap972 := d543
			snap973 := d544
			snap974 := d545
			snap975 := d546
			snap976 := d715
			snap977 := d716
			snap978 := d803
			snap979 := d804
			alloc980 := ctx.SnapshotAllocState()
			bbs[26].Render()
			ctx.RestoreAllocState(alloc980)
			d1 = snap893
			d2 = snap894
			d3 = snap895
			d4 = snap896
			d5 = snap897
			d6 = snap898
			d7 = snap899
			d8 = snap900
			d9 = snap901
			d10 = snap902
			d11 = snap903
			d12 = snap904
			d26 = snap905
			d27 = snap906
			d28 = snap907
			d45 = snap908
			d46 = snap909
			d65 = snap910
			d66 = snap911
			d67 = snap912
			d89 = snap913
			d90 = snap914
			d91 = snap915
			d116 = snap916
			d117 = snap917
			d144 = snap918
			d145 = snap919
			d174 = snap920
			d175 = snap921
			d206 = snap922
			d207 = snap923
			d208 = snap924
			d209 = snap925
			d210 = snap926
			d211 = snap927
			d212 = snap928
			d213 = snap929
			d214 = snap930
			d215 = snap931
			d217 = snap932
			d218 = snap933
			d219 = snap934
			d263 = snap935
			d264 = snap936
			d265 = snap937
			d267 = snap938
			d268 = snap939
			d269 = snap940
			d319 = snap941
			d320 = snap942
			d321 = snap943
			d323 = snap944
			d324 = snap945
			d325 = snap946
			d381 = snap947
			d382 = snap948
			d383 = snap949
			d384 = snap950
			d386 = snap951
			d387 = snap952
			d388 = snap953
			d451 = snap954
			d452 = snap955
			d454 = snap956
			d455 = snap957
			d456 = snap958
			d457 = snap959
			d458 = snap960
			d460 = snap961
			d461 = snap962
			d462 = snap963
			d535 = snap964
			d536 = snap965
			d537 = snap966
			d538 = snap967
			d539 = snap968
			d540 = snap969
			d541 = snap970
			d542 = snap971
			d543 = snap972
			d544 = snap973
			d545 = snap974
			d546 = snap975
			d715 = snap976
			d716 = snap977
			d803 = snap978
			d804 = snap979
		}
		if !bbs[25].Rendered {
			return bbs[25].Render()
		}
		return result
		return result
	}
	bbs[24].Render = func() JITValueDesc {
		if bbs[24].Rendered {
			ctx.EmitJmp(lbl25)
			return result
		}
		bbs[24].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_24 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl25)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d1)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1) {
			return d1
		}
		ctx.EnsureDesc(&d1)
		ctx.EmitMovToReg(result.Reg, d1)
		result.Type = d1.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[25].Render = func() JITValueDesc {
		if bbs[25].Rendered {
			ctx.EmitJmp(lbl26)
			return result
		}
		bbs[25].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_25 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl26)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d981 = args[1]
		_ = d981
		d981 = JITPrepareScmerGoArg(ctx, d981)
		ctx.SyncDesc(&d981)
		d982 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d981}, 1)
		d982.NoHeapPointer = true
		ctx.BindReg(d982.Reg, &d982)
		ctx.FreeDesc(&d981)
		ctx.EnsureDesc(&d982)
		if d982.Loc == LocImm {
			d983 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d982.Imm.Int() > 0)}
		} else {
			ctx.EmitCmpRegImm32(d982.Reg, 0)
			r42 := ctx.AllocRegExcept(d982.Reg)
			ctx.EmitSetcc(r42, CondSignedGreater)
			d983 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r42}
			ctx.BindReg(r42, &d983)
		}
		ctx.EnsureDesc(&d983)
		ctx.EmitStoreToStack(d983, int32(bbs[26].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d983)
		ctx.FreeDesc(&d982)
		return bbs[26].Render()
		return result
	}
	bbs[26].Render = func() JITValueDesc {
		if bbs[26].Rendered {
			ctx.EmitJmp(lbl27)
			return result
		}
		bbs[26].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_26 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl27)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d2)
		ctx.SyncDesc(&d2)
		if d2.Loc == LocReg || d2.Loc == LocFPReg {
			ctx.ProtectReg(d2.Reg)
		} else if d2.Loc == LocRegPair {
			ctx.ProtectReg(d2.Reg)
			ctx.ProtectReg(d2.Reg2)
		}
		d984 = d2
		if d984.Loc == LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d984)
		ctx.EmitStoreToStack(d984, int32(bbs[24].PhiBase)+int32(0))
		if d2.Loc == LocReg || d2.Loc == LocFPReg {
			ctx.UnprotectReg(d2.Reg)
		} else if d2.Loc == LocRegPair {
			ctx.UnprotectReg(d2.Reg)
			ctx.UnprotectReg(d2.Reg2)
		}
		return bbs[24].Render()
		return result
	}
	bbs[27].Render = func() JITValueDesc {
		if bbs[27].Rendered {
			ctx.EmitJmp(lbl28)
			return result
		}
		bbs[27].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_27 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl28)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d985 = args[0]
		_ = d985
		if d985.Loc != LocImm {
			ctx.EnsureDesc(&d985)
		}
		if d985.Loc == LocImm {
			d986 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d985.Imm.Int())}
		} else if d985.Type == tagInt && d985.Loc == LocRegPair {
			ctx.FreeReg(d985.Reg)
			d986 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d985.Reg2}
			ctx.BindReg(d985.Reg2, &d986)
			ctx.BindReg(d985.Reg2, &d986)
		} else if d985.Type == tagInt && d985.Loc == LocReg {
			d986 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d985.Reg}
			ctx.BindReg(d985.Reg, &d986)
			ctx.BindReg(d985.Reg, &d986)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d985)
			d986 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d986.Type = tagInt
			ctx.BindReg(d986.Reg, &d986)
		}
		ctx.FreeDesc(&d985)
		d987 = args[1]
		_ = d987
		if d987.Loc != LocImm {
			ctx.EnsureDesc(&d987)
		}
		if d987.Loc == LocImm {
			d988 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d987.Imm.Int())}
		} else if d987.Type == tagInt && d987.Loc == LocRegPair {
			ctx.FreeReg(d987.Reg)
			d988 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d987.Reg2}
			ctx.BindReg(d987.Reg2, &d988)
			ctx.BindReg(d987.Reg2, &d988)
		} else if d987.Type == tagInt && d987.Loc == LocReg {
			d988 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d987.Reg}
			ctx.BindReg(d987.Reg, &d988)
			ctx.BindReg(d987.Reg, &d988)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d987)
			d988 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d988.Type = tagInt
			ctx.BindReg(d988.Reg, &d988)
		}
		ctx.FreeDesc(&d987)
		ctx.EnsureDesc(&d986)
		resultTarget989 := false
		_ = resultTarget989
		ctx.EnsureDesc(&d988)
		ctx.EnsureDescsTogether(&d986, &d988)
		if d986.Loc == LocImm && d988.Loc == LocImm {
			d990 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d986.Imm.Int() < d988.Imm.Int())}
		} else if d988.Loc == LocImm {
			r43 := ctx.AllocReg()
			if d988.Imm.Int() >= -2147483648 && d988.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d986.Reg, int32(d988.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d988.Imm.Int()))
				ctx.EmitCmpInt64(d986.Reg, ctx.ScratchReg)
			}
			d990 = ctx.DeferBooleanFlags(r43, CondSignedLess)
			ctx.BindReg(r43, &d990)
		} else if d986.Loc == LocImm {
			r44 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d986.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d988.Reg)
			d990 = ctx.DeferBooleanFlags(r44, CondSignedLess)
			ctx.BindReg(r44, &d990)
		} else {
			r45 := ctx.AllocReg()
			ctx.EmitCmpInt64(d986.Reg, d988.Reg)
			d990 = ctx.DeferBooleanFlags(r45, CondSignedLess)
			ctx.BindReg(r45, &d990)
		}
		ctx.FreeDesc(&d986)
		ctx.FreeDesc(&d988)
		ctx.SyncDesc(&d990)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d990) {
			return d990
		}
		ctx.EnsureDesc(&d990)
		ctx.EmitMovToReg(result.Reg, d990)
		result.Type = d990.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[28].Render = func() JITValueDesc {
		if bbs[28].Rendered {
			ctx.EmitJmp(lbl29)
			return result
		}
		bbs[28].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_28 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d991 = args[0]
		_ = d991
		if d991.Loc != LocImm {
			ctx.EnsureDesc(&d991)
		}
		if d991.Loc == LocImm {
			d992 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d991.Imm.Int())}
		} else if d991.Type == tagInt && d991.Loc == LocRegPair {
			ctx.FreeReg(d991.Reg)
			d992 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d991.Reg2}
			ctx.BindReg(d991.Reg2, &d992)
			ctx.BindReg(d991.Reg2, &d992)
		} else if d991.Type == tagInt && d991.Loc == LocReg {
			d992 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d991.Reg}
			ctx.BindReg(d991.Reg, &d992)
			ctx.BindReg(d991.Reg, &d992)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d991)
			d992 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d992.Type = tagInt
			ctx.BindReg(d992.Reg, &d992)
		}
		ctx.FreeDesc(&d991)
		ctx.EnsureDesc(&d992)
		ctx.EnsureDesc(&d992)
		if d992.Loc == LocImm {
			d993 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d992.Imm.Int()))}
		} else {
			var r46 Reg
			r46 = d992.Reg
			d992.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r46)
			d993 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r46}
			ctx.BindReg(r46, &d993)
		}
		ctx.FreeDesc(&d992)
		d994 = args[1]
		_ = d994
		d995 = ctx.EmitFloatDesc(d994)
		ctx.FreeDesc(&d994)
		ctx.EnsureDesc(&d993)
		resultTarget996 := false
		_ = resultTarget996
		ctx.EnsureDesc(&d995)
		ctx.EnsureDescsTogether(&d993, &d995)
		if d993.Loc == LocImm && d995.Loc == LocImm {
			d997 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d993.Imm.Float() < d995.Imm.Float())}
		} else if d995.Loc == LocImm {
			var r47 Reg
			if result.Loc == LocReg && result.Reg != d993.Reg {
				r47 = result.Reg
				resultTarget996 = true
			} else {
				r47 = ctx.AllocRegExcept(d993.Reg)
			}
			_, yBits := d995.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d993.Reg)
			d997 = ctx.DeferBooleanFlags(r47, CondUnsignedAbove)
			ctx.BindReg(r47, &d997)
		} else if d993.Loc == LocImm {
			var r48 Reg
			if result.Loc == LocReg && result.Reg != d995.Reg {
				r48 = result.Reg
				resultTarget996 = true
			} else {
				r48 = ctx.AllocRegExcept(d995.Reg)
			}
			_, xBits := d993.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d995.Reg, ctx.ScratchReg)
			d997 = ctx.DeferBooleanFlags(r48, CondUnsignedAbove)
			ctx.BindReg(r48, &d997)
		} else {
			var r49 Reg
			if result.Loc == LocReg && result.Reg != d993.Reg && result.Reg != d995.Reg {
				r49 = result.Reg
				resultTarget996 = true
			} else {
				r49 = ctx.AllocRegExcept(d993.Reg, d995.Reg)
			}
			ctx.EmitCmpFloat64(d995.Reg, d993.Reg)
			d997 = ctx.DeferBooleanFlags(r49, CondUnsignedAbove)
			ctx.BindReg(r49, &d997)
		}
		ctx.FreeDesc(&d993)
		ctx.FreeDesc(&d995)
		ctx.SyncDesc(&d997)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d997) {
			return d997
		}
		ctx.EnsureDesc(&d997)
		ctx.EmitMovToReg(result.Reg, d997)
		result.Type = d997.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[29].Render = func() JITValueDesc {
		if bbs[29].Rendered {
			ctx.EmitJmp(lbl30)
			return result
		}
		bbs[29].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_29 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl30)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d998 = args[0]
		_ = d998
		if d998.Loc != LocImm {
			ctx.EnsureDesc(&d998)
		}
		if d998.Loc == LocImm {
			d999 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d998.Imm.Int())}
		} else if d998.Type == tagInt && d998.Loc == LocRegPair {
			ctx.FreeReg(d998.Reg)
			d999 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d998.Reg2}
			ctx.BindReg(d998.Reg2, &d999)
			ctx.BindReg(d998.Reg2, &d999)
		} else if d998.Type == tagInt && d998.Loc == LocReg {
			d999 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d998.Reg}
			ctx.BindReg(d998.Reg, &d999)
			ctx.BindReg(d998.Reg, &d999)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d998)
			d999 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d999.Type = tagInt
			ctx.BindReg(d999.Reg, &d999)
		}
		ctx.FreeDesc(&d998)
		d1000 = args[1]
		_ = d1000
		if d1000.Loc != LocImm {
			ctx.EnsureDesc(&d1000)
		}
		if d1000.Loc == LocImm {
			d1001 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1000.Imm.Int())}
		} else if d1000.Type == tagInt && d1000.Loc == LocRegPair {
			ctx.FreeReg(d1000.Reg)
			d1001 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1000.Reg2}
			ctx.BindReg(d1000.Reg2, &d1001)
			ctx.BindReg(d1000.Reg2, &d1001)
		} else if d1000.Type == tagInt && d1000.Loc == LocReg {
			d1001 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1000.Reg}
			ctx.BindReg(d1000.Reg, &d1001)
			ctx.BindReg(d1000.Reg, &d1001)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d1000)
			d1001 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1001.Type = tagInt
			ctx.BindReg(d1001.Reg, &d1001)
		}
		ctx.FreeDesc(&d1000)
		ctx.EnsureDesc(&d999)
		resultTarget1002 := false
		_ = resultTarget1002
		ctx.EnsureDesc(&d1001)
		ctx.EnsureDescsTogether(&d999, &d1001)
		if d999.Loc == LocImm && d1001.Loc == LocImm {
			d1003 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d999.Imm.Int() < d1001.Imm.Int())}
		} else if d1001.Loc == LocImm {
			r50 := ctx.AllocReg()
			if d1001.Imm.Int() >= -2147483648 && d1001.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d999.Reg, int32(d1001.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1001.Imm.Int()))
				ctx.EmitCmpInt64(d999.Reg, ctx.ScratchReg)
			}
			d1003 = ctx.DeferBooleanFlags(r50, CondSignedLess)
			ctx.BindReg(r50, &d1003)
		} else if d999.Loc == LocImm {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d999.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d1001.Reg)
			d1003 = ctx.DeferBooleanFlags(r51, CondSignedLess)
			ctx.BindReg(r51, &d1003)
		} else {
			r52 := ctx.AllocReg()
			ctx.EmitCmpInt64(d999.Reg, d1001.Reg)
			d1003 = ctx.DeferBooleanFlags(r52, CondSignedLess)
			ctx.BindReg(r52, &d1003)
		}
		ctx.FreeDesc(&d999)
		ctx.FreeDesc(&d1001)
		ctx.SyncDesc(&d1003)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1003) {
			return d1003
		}
		ctx.EnsureDesc(&d1003)
		ctx.EmitMovToReg(result.Reg, d1003)
		result.Type = d1003.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[30].Render = func() JITValueDesc {
		if bbs[30].Rendered {
			ctx.EmitJmp(lbl31)
			return result
		}
		bbs[30].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_30 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl31)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d7)
		if d7.Loc == LocImm {
			d1004 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) == uint64(0x5))}
		} else {
			r53 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpRegImm32(d7.Reg, 5)
			d1004 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r53, Condition: CondEqual}
			ctx.BindReg(r53, &d1004)
		}
		d1005 = d1004
		ctx.SyncDesc(&d1005)
		if d1005.Loc != LocImm && d1005.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d1005.Loc == LocImm {
			if d1005.Imm.Bool() {
				return bbs[29].Render()
			}
			return bbs[31].Render()
		}
		ctx.EmitJump(d1005.Condition, lbl30)
		if bbs[31].Rendered {
			ctx.EmitJmp(lbl32)
		}
		ctx.FreeDesc(&d1004)
		ctx.FlushRegisterMoves()
		if !bbs[31].Rendered {
			snap1006 := d1
			snap1007 := d2
			snap1008 := d3
			snap1009 := d4
			snap1010 := d5
			snap1011 := d6
			snap1012 := d7
			snap1013 := d8
			snap1014 := d9
			snap1015 := d10
			snap1016 := d11
			snap1017 := d12
			snap1018 := d26
			snap1019 := d27
			snap1020 := d28
			snap1021 := d45
			snap1022 := d46
			snap1023 := d65
			snap1024 := d66
			snap1025 := d67
			snap1026 := d89
			snap1027 := d90
			snap1028 := d91
			snap1029 := d116
			snap1030 := d117
			snap1031 := d144
			snap1032 := d145
			snap1033 := d174
			snap1034 := d175
			snap1035 := d206
			snap1036 := d207
			snap1037 := d208
			snap1038 := d209
			snap1039 := d210
			snap1040 := d211
			snap1041 := d212
			snap1042 := d213
			snap1043 := d214
			snap1044 := d215
			snap1045 := d217
			snap1046 := d218
			snap1047 := d219
			snap1048 := d263
			snap1049 := d264
			snap1050 := d265
			snap1051 := d267
			snap1052 := d268
			snap1053 := d269
			snap1054 := d319
			snap1055 := d320
			snap1056 := d321
			snap1057 := d323
			snap1058 := d324
			snap1059 := d325
			snap1060 := d381
			snap1061 := d382
			snap1062 := d383
			snap1063 := d384
			snap1064 := d386
			snap1065 := d387
			snap1066 := d388
			snap1067 := d451
			snap1068 := d452
			snap1069 := d454
			snap1070 := d455
			snap1071 := d456
			snap1072 := d457
			snap1073 := d458
			snap1074 := d460
			snap1075 := d461
			snap1076 := d462
			snap1077 := d535
			snap1078 := d536
			snap1079 := d537
			snap1080 := d538
			snap1081 := d539
			snap1082 := d540
			snap1083 := d541
			snap1084 := d542
			snap1085 := d543
			snap1086 := d544
			snap1087 := d545
			snap1088 := d546
			snap1089 := d715
			snap1090 := d716
			snap1091 := d803
			snap1092 := d804
			snap1093 := d981
			snap1094 := d982
			snap1095 := d983
			snap1096 := d984
			snap1097 := d985
			snap1098 := d986
			snap1099 := d987
			snap1100 := d988
			snap1101 := d990
			snap1102 := d991
			snap1103 := d992
			snap1104 := d993
			snap1105 := d994
			snap1106 := d995
			snap1107 := d997
			snap1108 := d998
			snap1109 := d999
			snap1110 := d1000
			snap1111 := d1001
			snap1112 := d1003
			snap1113 := d1004
			snap1114 := d1005
			alloc1115 := ctx.SnapshotAllocState()
			bbs[31].Render()
			ctx.RestoreAllocState(alloc1115)
			d1 = snap1006
			d2 = snap1007
			d3 = snap1008
			d4 = snap1009
			d5 = snap1010
			d6 = snap1011
			d7 = snap1012
			d8 = snap1013
			d9 = snap1014
			d10 = snap1015
			d11 = snap1016
			d12 = snap1017
			d26 = snap1018
			d27 = snap1019
			d28 = snap1020
			d45 = snap1021
			d46 = snap1022
			d65 = snap1023
			d66 = snap1024
			d67 = snap1025
			d89 = snap1026
			d90 = snap1027
			d91 = snap1028
			d116 = snap1029
			d117 = snap1030
			d144 = snap1031
			d145 = snap1032
			d174 = snap1033
			d175 = snap1034
			d206 = snap1035
			d207 = snap1036
			d208 = snap1037
			d209 = snap1038
			d210 = snap1039
			d211 = snap1040
			d212 = snap1041
			d213 = snap1042
			d214 = snap1043
			d215 = snap1044
			d217 = snap1045
			d218 = snap1046
			d219 = snap1047
			d263 = snap1048
			d264 = snap1049
			d265 = snap1050
			d267 = snap1051
			d268 = snap1052
			d269 = snap1053
			d319 = snap1054
			d320 = snap1055
			d321 = snap1056
			d323 = snap1057
			d324 = snap1058
			d325 = snap1059
			d381 = snap1060
			d382 = snap1061
			d383 = snap1062
			d384 = snap1063
			d386 = snap1064
			d387 = snap1065
			d388 = snap1066
			d451 = snap1067
			d452 = snap1068
			d454 = snap1069
			d455 = snap1070
			d456 = snap1071
			d457 = snap1072
			d458 = snap1073
			d460 = snap1074
			d461 = snap1075
			d462 = snap1076
			d535 = snap1077
			d536 = snap1078
			d537 = snap1079
			d538 = snap1080
			d539 = snap1081
			d540 = snap1082
			d541 = snap1083
			d542 = snap1084
			d543 = snap1085
			d544 = snap1086
			d545 = snap1087
			d546 = snap1088
			d715 = snap1089
			d716 = snap1090
			d803 = snap1091
			d804 = snap1092
			d981 = snap1093
			d982 = snap1094
			d983 = snap1095
			d984 = snap1096
			d985 = snap1097
			d986 = snap1098
			d987 = snap1099
			d988 = snap1100
			d990 = snap1101
			d991 = snap1102
			d992 = snap1103
			d993 = snap1104
			d994 = snap1105
			d995 = snap1106
			d997 = snap1107
			d998 = snap1108
			d999 = snap1109
			d1000 = snap1110
			d1001 = snap1111
			d1003 = snap1112
			d1004 = snap1113
			d1005 = snap1114
		}
		if !bbs[29].Rendered {
			return bbs[29].Render()
		}
		return result
		return result
	}
	bbs[31].Render = func() JITValueDesc {
		if bbs[31].Rendered {
			ctx.EmitJmp(lbl32)
			return result
		}
		bbs[31].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_31 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl32)
		ctx.ResolveFixups()
		d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
		d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d1116 = args[0]
		_ = d1116
		d1117 = args[1]
		_ = d1117
		d1116 = JITPrepareScmerGoArg(ctx, d1116)
		d1117 = JITPrepareScmerGoArg(ctx, d1117)
		if d7.Loc == LocRegPair || d7.Loc == LocStackPair || d7.Loc == LocRegTriple || d7.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&d1116)
		ctx.SyncDesc(&d1117)
		ctx.SyncDesc(&d7)
		ctx.SyncDesc(&d10)
		d1118 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{d1116, d1117, d7, d10}, 1)
		d1118.NoHeapPointer = true
		ctx.EmitAndRegImm32(d1118.Reg, 1)
		d1118.Type = tagBool
		ctx.BindReg(d1118.Reg, &d1118)
		ctx.FreeDesc(&d1116)
		ctx.FreeDesc(&d1117)
		ctx.SyncDesc(&d1118)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1118) {
			return d1118
		}
		ctx.EnsureDesc(&d1118)
		ctx.EmitMovToReg(result.Reg, d1118)
		result.Type = d1118.Type
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

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_0cbff6f55ee96f6d uint64
var jitGoABI_58434e8dfe39c263 uint64
var jitGoABI_5bd1d0f41a4e4914 uint64
var jitGoABI_973b6d6b19b704fd uint64
var jitGoABI_d69f4c85104836f4 uint64

func init() {
	jitGoABI_0cbff6f55ee96f6d = GoFuncAddr(Scmer.String)
	jitGoABI_58434e8dfe39c263 = GoFuncAddr(ParseDateValue)
	jitGoABI_5bd1d0f41a4e4914 = GoFuncAddr(Scmer.Int)
	jitGoABI_973b6d6b19b704fd = GoFuncAddr((Scmer).DateNanosecond)
	jitGoABI_d69f4c85104836f4 = GoFuncAddr(compareDates)
}
