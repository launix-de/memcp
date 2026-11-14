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
		case tagCString, tagLZ4:
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
			return a.DateNanosecond() == 0 && a.Int() == b.Int()
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
			return b.DateNanosecond() == 0 && a.Int() == b.Int()
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
	case tagCString, tagLZ4:
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
		case tagCString, tagLZ4:
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
			return NewBool(a.DateNanosecond() == 0 && a.Int() == b.Int())
		}
		return NewBool(a.Float() == b.Float())
	case tagInt:
		if tb == tagDate {
			return NewBool(b.DateNanosecond() == 0 && a.Int() == b.Int())
		}
		if tb == tagFloat {
			return NewBool(float64(a.Int()) == b.Float())
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString || tb == tagLZ4 {
			return NewBool(numericStringEqualsInt(b.String(), a.Int()))
		}
		return NewBool(a.Int() == b.Int())
	case tagFloat:
		if tb == tagInt {
			return NewBool(a.Float() == float64(b.Int()))
		}
		if tb == tagString || tb == tagSymbol || tb == tagCString || tb == tagBString || tb == tagLZ4 {
			return NewBool(a.Float() == b.Float())
		}
		return NewBool(a.Float() == b.Float())
	case tagString, tagSymbol, tagCString, tagLZ4:
		if tb == tagDate {
			if value, ok := ParseDateValue(a.String()); ok {
				return NewBool(compareDates(value, b) == 0)
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
			return a.Int() < b.Int()
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
			return a.Int() < b.Int() || a.Int() == b.Int() && b.DateNanosecond() > 0
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
	case tagString, tagSymbol, tagCString, tagBString, tagLZ4:
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
		case tagCString, tagBString, tagLZ4, tagBSON:
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
	var d18 JITValueDesc
	_ = d18
	var d19 JITValueDesc
	_ = d19
	var d20 JITValueDesc
	_ = d20
	var d33 JITValueDesc
	_ = d33
	var d34 JITValueDesc
	_ = d34
	var d49 JITValueDesc
	_ = d49
	var d50 JITValueDesc
	_ = d50
	var d51 JITValueDesc
	_ = d51
	var d69 JITValueDesc
	_ = d69
	var d70 JITValueDesc
	_ = d70
	var d71 JITValueDesc
	_ = d71
	var d92 JITValueDesc
	_ = d92
	var d93 JITValueDesc
	_ = d93
	var d116 JITValueDesc
	_ = d116
	var d117 JITValueDesc
	_ = d117
	var d142 JITValueDesc
	_ = d142
	var d143 JITValueDesc
	_ = d143
	var d170 JITValueDesc
	_ = d170
	var d171 JITValueDesc
	_ = d171
	var d173 JITValueDesc
	_ = d173
	var d174 JITValueDesc
	_ = d174
	var d175 JITValueDesc
	_ = d175
	var d207 JITValueDesc
	_ = d207
	var d209 JITValueDesc
	_ = d209
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
	var d216 JITValueDesc
	_ = d216
	var d217 JITValueDesc
	_ = d217
	var d218 JITValueDesc
	_ = d218
	var d219 JITValueDesc
	_ = d219
	var d220 JITValueDesc
	_ = d220
	var d221 JITValueDesc
	_ = d221
	var d222 JITValueDesc
	_ = d222
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
	var d295 JITValueDesc
	_ = d295
	var d296 JITValueDesc
	_ = d296
	var d298 JITValueDesc
	_ = d298
	var d299 JITValueDesc
	_ = d299
	var d300 JITValueDesc
	_ = d300
	var d364 JITValueDesc
	_ = d364
	var d365 JITValueDesc
	_ = d365
	var d367 JITValueDesc
	_ = d367
	var d368 JITValueDesc
	_ = d368
	var d369 JITValueDesc
	_ = d369
	var d438 JITValueDesc
	_ = d438
	var d440 JITValueDesc
	_ = d440
	var d442 JITValueDesc
	_ = d442
	var d443 JITValueDesc
	_ = d443
	var d444 JITValueDesc
	_ = d444
	var d445 JITValueDesc
	_ = d445
	var d446 JITValueDesc
	_ = d446
	var d447 JITValueDesc
	_ = d447
	var d448 JITValueDesc
	_ = d448
	var d449 JITValueDesc
	_ = d449
	var d450 JITValueDesc
	_ = d450
	var d451 JITValueDesc
	_ = d451
	var d452 JITValueDesc
	_ = d452
	var d453 JITValueDesc
	_ = d453
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
	var d465 JITValueDesc
	_ = d465
	var d466 JITValueDesc
	_ = d466
	var d467 JITValueDesc
	_ = d467
	var d469 JITValueDesc
	_ = d469
	var d470 JITValueDesc
	_ = d470
	var d471 JITValueDesc
	_ = d471
	var d570 JITValueDesc
	_ = d570
	var d571 JITValueDesc
	_ = d571
	var d572 JITValueDesc
	_ = d572
	var d573 JITValueDesc
	_ = d573
	var d778 JITValueDesc
	_ = d778
	var d779 JITValueDesc
	_ = d779
	var d884 JITValueDesc
	_ = d884
	var d885 JITValueDesc
	_ = d885
	var d886 JITValueDesc
	_ = d886
	var d887 JITValueDesc
	_ = d887
	var d1104 JITValueDesc
	_ = d1104
	var d1105 JITValueDesc
	_ = d1105
	var d1106 JITValueDesc
	_ = d1106
	var d1107 JITValueDesc
	_ = d1107
	var d1108 JITValueDesc
	_ = d1108
	var d1110 JITValueDesc
	_ = d1110
	var d1111 JITValueDesc
	_ = d1111
	var d1112 JITValueDesc
	_ = d1112
	var d1113 JITValueDesc
	_ = d1113
	var d1115 JITValueDesc
	_ = d1115
	var d1116 JITValueDesc
	_ = d1116
	var d1117 JITValueDesc
	_ = d1117
	var d1119 JITValueDesc
	_ = d1119
	var d1120 JITValueDesc
	_ = d1120
	var d1121 JITValueDesc
	_ = d1121
	var d1245 JITValueDesc
	_ = d1245
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
		d3 = args[0]
		d3.ID = 0
		d4 = ctx.EmitGetTagDesc(&d3, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d4)
		d5 = args[1]
		d5.ID = 0
		d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x0))}
		} else {
			r0 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 0)
			d7 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
			ctx.BindReg(r0, &d7)
		}
		d8 = d7
		ctx.SyncDesc(&d8)
		if d8.Loc != LocImm && d8.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d8.Loc == LocImm {
			if d8.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d8.Condition, lbl4)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d7)
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
		d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d18) {
			return d18
		}
		ctx.EnsureDesc(&d18)
		ctx.EmitMovToReg(result.Reg, d18)
		result.Type = d18.Type
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
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x0))}
		} else {
			r1 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 0)
			d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
			ctx.BindReg(r1, &d19)
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
			snap21 := d1
			snap22 := d2
			snap23 := d3
			snap24 := d4
			snap25 := d5
			snap26 := d6
			snap27 := d7
			snap28 := d8
			snap29 := d18
			snap30 := d19
			snap31 := d20
			alloc32 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc32)
			d1 = snap21
			d2 = snap22
			d3 = snap23
			d4 = snap24
			d5 = snap25
			d6 = snap26
			d7 = snap27
			d8 = snap28
			d18 = snap29
			d19 = snap30
			d20 = snap31
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x0))}
		} else {
			r2 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 0)
			d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
			ctx.BindReg(r2, &d33)
		}
		d34 = d33
		ctx.SyncDesc(&d34)
		if d34.Loc != LocImm && d34.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d34.Loc == LocImm {
			if d34.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d34.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d33)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
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
			snap46 := d33
			snap47 := d34
			alloc48 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc48)
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
			d33 = snap46
			d34 = snap47
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
		d49 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d49) {
			return d49
		}
		ctx.EnsureDesc(&d49)
		ctx.EmitMovToReg(result.Reg, d49)
		result.Type = d49.Type
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d50 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x0))}
		} else {
			r3 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 0)
			d50 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
			ctx.BindReg(r3, &d50)
		}
		d51 = d50
		ctx.SyncDesc(&d51)
		if d51.Loc != LocImm && d51.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d51.Loc == LocImm {
			if d51.Imm.Bool() {
				return bbs[6].Render()
			}
			return bbs[7].Render()
		}
		ctx.EmitJump(d51.Condition, lbl7)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d50)
		ctx.FlushRegisterMoves()
		if !bbs[7].Rendered {
			snap52 := d1
			snap53 := d2
			snap54 := d3
			snap55 := d4
			snap56 := d5
			snap57 := d6
			snap58 := d7
			snap59 := d8
			snap60 := d18
			snap61 := d19
			snap62 := d20
			snap63 := d33
			snap64 := d34
			snap65 := d49
			snap66 := d50
			snap67 := d51
			alloc68 := ctx.SnapshotAllocState()
			bbs[7].Render()
			ctx.RestoreAllocState(alloc68)
			d1 = snap52
			d2 = snap53
			d3 = snap54
			d4 = snap55
			d5 = snap56
			d6 = snap57
			d7 = snap58
			d8 = snap59
			d18 = snap60
			d19 = snap61
			d20 = snap62
			d33 = snap63
			d34 = snap64
			d49 = snap65
			d50 = snap66
			d51 = snap67
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
		d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d69) {
			return d69
		}
		ctx.EnsureDesc(&d69)
		ctx.EmitMovToReg(result.Reg, d69)
		result.Type = d69.Type
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
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d70 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x10))}
		} else {
			r4 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 16)
			d70 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
			ctx.BindReg(r4, &d70)
		}
		d71 = d70
		ctx.SyncDesc(&d71)
		if d71.Loc != LocImm && d71.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d71.Loc == LocImm {
			if d71.Imm.Bool() {
				return bbs[8].Render()
			}
			return bbs[10].Render()
		}
		ctx.EmitJump(d71.Condition, lbl9)
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
		}
		ctx.FreeDesc(&d70)
		ctx.FlushRegisterMoves()
		if !bbs[10].Rendered {
			snap72 := d1
			snap73 := d2
			snap74 := d3
			snap75 := d4
			snap76 := d5
			snap77 := d6
			snap78 := d7
			snap79 := d8
			snap80 := d18
			snap81 := d19
			snap82 := d20
			snap83 := d33
			snap84 := d34
			snap85 := d49
			snap86 := d50
			snap87 := d51
			snap88 := d69
			snap89 := d70
			snap90 := d71
			alloc91 := ctx.SnapshotAllocState()
			bbs[10].Render()
			ctx.RestoreAllocState(alloc91)
			d1 = snap72
			d2 = snap73
			d3 = snap74
			d4 = snap75
			d5 = snap76
			d6 = snap77
			d7 = snap78
			d8 = snap79
			d18 = snap80
			d19 = snap81
			d20 = snap82
			d33 = snap83
			d34 = snap84
			d49 = snap85
			d50 = snap86
			d51 = snap87
			d69 = snap88
			d70 = snap89
			d71 = snap90
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d92 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x4))}
		} else {
			r5 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 4)
			d92 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
			ctx.BindReg(r5, &d92)
		}
		d93 = d92
		ctx.SyncDesc(&d93)
		if d93.Loc != LocImm && d93.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d93.Loc == LocImm {
			if d93.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitJump(d93.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d92)
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap94 := d1
			snap95 := d2
			snap96 := d3
			snap97 := d4
			snap98 := d5
			snap99 := d6
			snap100 := d7
			snap101 := d8
			snap102 := d18
			snap103 := d19
			snap104 := d20
			snap105 := d33
			snap106 := d34
			snap107 := d49
			snap108 := d50
			snap109 := d51
			snap110 := d69
			snap111 := d70
			snap112 := d71
			snap113 := d92
			snap114 := d93
			alloc115 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc115)
			d1 = snap94
			d2 = snap95
			d3 = snap96
			d4 = snap97
			d5 = snap98
			d6 = snap99
			d7 = snap100
			d8 = snap101
			d18 = snap102
			d19 = snap103
			d20 = snap104
			d33 = snap105
			d34 = snap106
			d49 = snap107
			d50 = snap108
			d51 = snap109
			d69 = snap110
			d70 = snap111
			d71 = snap112
			d92 = snap113
			d93 = snap114
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d116 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x10))}
		} else {
			r6 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 16)
			d116 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
			ctx.BindReg(r6, &d116)
		}
		d117 = d116
		ctx.SyncDesc(&d117)
		if d117.Loc != LocImm && d117.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d117.Loc == LocImm {
			if d117.Imm.Bool() {
				return bbs[21].Render()
			}
			return bbs[22].Render()
		}
		ctx.EmitJump(d117.Condition, lbl22)
		if bbs[22].Rendered {
			ctx.EmitJmp(lbl23)
		}
		ctx.FreeDesc(&d116)
		ctx.FlushRegisterMoves()
		if !bbs[22].Rendered {
			snap118 := d1
			snap119 := d2
			snap120 := d3
			snap121 := d4
			snap122 := d5
			snap123 := d6
			snap124 := d7
			snap125 := d8
			snap126 := d18
			snap127 := d19
			snap128 := d20
			snap129 := d33
			snap130 := d34
			snap131 := d49
			snap132 := d50
			snap133 := d51
			snap134 := d69
			snap135 := d70
			snap136 := d71
			snap137 := d92
			snap138 := d93
			snap139 := d116
			snap140 := d117
			alloc141 := ctx.SnapshotAllocState()
			bbs[22].Render()
			ctx.RestoreAllocState(alloc141)
			d1 = snap118
			d2 = snap119
			d3 = snap120
			d4 = snap121
			d5 = snap122
			d6 = snap123
			d7 = snap124
			d8 = snap125
			d18 = snap126
			d19 = snap127
			d20 = snap128
			d33 = snap129
			d34 = snap130
			d49 = snap131
			d50 = snap132
			d51 = snap133
			d69 = snap134
			d70 = snap135
			d71 = snap136
			d92 = snap137
			d93 = snap138
			d116 = snap139
			d117 = snap140
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
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d142 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x4))}
		} else {
			r7 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 4)
			d142 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
			ctx.BindReg(r7, &d142)
		}
		d143 = d142
		ctx.SyncDesc(&d143)
		if d143.Loc != LocImm && d143.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d143.Loc == LocImm {
			if d143.Imm.Bool() {
				return bbs[9].Render()
			}
			return bbs[20].Render()
		}
		ctx.EmitJump(d143.Condition, lbl10)
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
		}
		ctx.FreeDesc(&d142)
		ctx.FlushRegisterMoves()
		if !bbs[20].Rendered {
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
			snap155 := d33
			snap156 := d34
			snap157 := d49
			snap158 := d50
			snap159 := d51
			snap160 := d69
			snap161 := d70
			snap162 := d71
			snap163 := d92
			snap164 := d93
			snap165 := d116
			snap166 := d117
			snap167 := d142
			snap168 := d143
			alloc169 := ctx.SnapshotAllocState()
			bbs[20].Render()
			ctx.RestoreAllocState(alloc169)
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
			d33 = snap155
			d34 = snap156
			d49 = snap157
			d50 = snap158
			d51 = snap159
			d69 = snap160
			d70 = snap161
			d71 = snap162
			d92 = snap163
			d93 = snap164
			d116 = snap165
			d117 = snap166
			d142 = snap167
			d143 = snap168
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d170 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d170 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d170)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d170 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d170)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d170 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d170.Type = tagInt
			ctx.BindReg(d170.Reg, &d170)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d171 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d171 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d171)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d171 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d171)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d171 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d171.Type = tagInt
			ctx.BindReg(d171.Reg, &d171)
		}
		ctx.EnsureDesc(&d170)
		resultTarget172 := false
		_ = resultTarget172
		ctx.EnsureDesc(&d171)
		ctx.EnsureDescsTogether(&d170, &d171)
		if d170.Loc == LocImm && d171.Loc == LocImm {
			d173 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d170.Imm.Int() < d171.Imm.Int())}
		} else if d171.Loc == LocImm {
			r8 := ctx.AllocReg()
			if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d170.Reg, int32(d171.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d171.Imm.Int()))
				ctx.EmitCmpInt64(d170.Reg, ctx.ScratchReg)
			}
			d173 = ctx.DeferBooleanFlags(r8, CondSignedLess)
			ctx.BindReg(r8, &d173)
		} else if d170.Loc == LocImm {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d170.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d171.Reg)
			d173 = ctx.DeferBooleanFlags(r9, CondSignedLess)
			ctx.BindReg(r9, &d173)
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitCmpInt64(d170.Reg, d171.Reg)
			d173 = ctx.DeferBooleanFlags(r10, CondSignedLess)
			ctx.BindReg(r10, &d173)
		}
		ctx.FreeDesc(&d170)
		ctx.FreeDesc(&d171)
		ctx.SyncDesc(&d173)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d173) {
			return d173
		}
		ctx.EnsureDesc(&d173)
		ctx.EmitMovToReg(result.Reg, d173)
		result.Type = d173.Type
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d174 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x10))}
		} else {
			r11 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 16)
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
				return bbs[13].Render()
			}
			return bbs[14].Render()
		}
		ctx.EmitJump(d175.Condition, lbl14)
		if bbs[14].Rendered {
			ctx.EmitJmp(lbl15)
		}
		ctx.FreeDesc(&d174)
		ctx.FlushRegisterMoves()
		if !bbs[14].Rendered {
			snap176 := d1
			snap177 := d2
			snap178 := d3
			snap179 := d4
			snap180 := d5
			snap181 := d6
			snap182 := d7
			snap183 := d8
			snap184 := d18
			snap185 := d19
			snap186 := d20
			snap187 := d33
			snap188 := d34
			snap189 := d49
			snap190 := d50
			snap191 := d51
			snap192 := d69
			snap193 := d70
			snap194 := d71
			snap195 := d92
			snap196 := d93
			snap197 := d116
			snap198 := d117
			snap199 := d142
			snap200 := d143
			snap201 := d170
			snap202 := d171
			snap203 := d173
			snap204 := d174
			snap205 := d175
			alloc206 := ctx.SnapshotAllocState()
			bbs[14].Render()
			ctx.RestoreAllocState(alloc206)
			d1 = snap176
			d2 = snap177
			d3 = snap178
			d4 = snap179
			d5 = snap180
			d6 = snap181
			d7 = snap182
			d8 = snap183
			d18 = snap184
			d19 = snap185
			d20 = snap186
			d33 = snap187
			d34 = snap188
			d49 = snap189
			d50 = snap190
			d51 = snap191
			d69 = snap192
			d70 = snap193
			d71 = snap194
			d92 = snap195
			d93 = snap196
			d116 = snap197
			d117 = snap198
			d142 = snap199
			d143 = snap200
			d170 = snap201
			d171 = snap202
			d173 = snap203
			d174 = snap204
			d175 = snap205
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
		ctx.EnsureDesc(&args[0])
		ctx.EnsureDesc(&args[1])
		d207 = args[0]
		_ = d207
		ctx.StabilizeDescForControlFlow(&d207)
		inlineHome208 := d207
		inlineHome208.ID = 0
		d209 = args[1]
		_ = d209
		ctx.StabilizeDescForControlFlow(&d209)
		inlineHome210 := d209
		inlineHome210.ID = 0
		lbl33 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl34 := ctx.ReserveLabel()
		_ = lbl34
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		lbl35 := ctx.ReserveLabel()
		_ = lbl35
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		lbl36 := ctx.ReserveLabel()
		_ = lbl36
		bbpos_1_3 := int32(-1)
		_ = bbpos_1_3
		lbl37 := ctx.ReserveLabel()
		_ = lbl37
		bbpos_1_4 := int32(-1)
		_ = bbpos_1_4
		lbl38 := ctx.ReserveLabel()
		_ = lbl38
		bbpos_1_5 := int32(-1)
		_ = bbpos_1_5
		lbl39 := ctx.ReserveLabel()
		_ = lbl39
		bbpos_1_6 := int32(-1)
		_ = bbpos_1_6
		lbl40 := ctx.ReserveLabel()
		_ = lbl40
		bbpos_1_7 := int32(-1)
		_ = bbpos_1_7
		lbl41 := ctx.ReserveLabel()
		_ = lbl41
		bbpos_1_8 := int32(-1)
		_ = bbpos_1_8
		lbl42 := ctx.ReserveLabel()
		_ = lbl42
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl34)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d207.Loc != LocImm {
			ctx.EnsureDesc(&d207)
		}
		if d207.Loc == LocImm {
			d211 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d207.Imm.Int())}
		} else if d207.Type == tagInt && d207.Loc == LocRegPair {
			ctx.FreeReg(d207.Reg)
			d211 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d207.Reg2}
			ctx.BindReg(d207.Reg2, &d211)
			ctx.BindReg(d207.Reg2, &d211)
		} else if d207.Type == tagInt && d207.Loc == LocReg {
			d211 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d207.Reg}
			ctx.BindReg(d207.Reg, &d211)
			ctx.BindReg(d207.Reg, &d211)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d207)
			d211 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d211.Type = tagInt
			ctx.BindReg(d211.Reg, &d211)
		}
		ctx.ReclaimUntrackedRegs()
		if d209.Loc != LocImm {
			ctx.EnsureDesc(&d209)
		}
		if d209.Loc == LocImm {
			d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d209.Imm.Int())}
		} else if d209.Type == tagInt && d209.Loc == LocRegPair {
			ctx.FreeReg(d209.Reg)
			d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d209.Reg2}
			ctx.BindReg(d209.Reg2, &d212)
			ctx.BindReg(d209.Reg2, &d212)
		} else if d209.Type == tagInt && d209.Loc == LocReg {
			d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d209.Reg}
			ctx.BindReg(d209.Reg, &d212)
			ctx.BindReg(d209.Reg, &d212)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d209)
			d212 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d212.Type = tagInt
			ctx.BindReg(d212.Reg, &d212)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d211)
		ctx.EnsureDesc(&d212)
		ctx.EnsureDescsTogether(&d211, &d212)
		if d211.Loc == LocImm && d212.Loc == LocImm {
			d213 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d211.Imm.Int() < d212.Imm.Int())}
		} else if d212.Loc == LocImm {
			r12 := ctx.AllocReg()
			if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d211.Reg, int32(d212.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d212.Imm.Int()))
				ctx.EmitCmpInt64(d211.Reg, ctx.ScratchReg)
			}
			d213 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
			ctx.BindReg(r12, &d213)
		} else if d211.Loc == LocImm {
			r13 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d211.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d212.Reg)
			d213 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
			ctx.BindReg(r13, &d213)
		} else {
			r14 := ctx.AllocReg()
			ctx.EmitCmpInt64(d211.Reg, d212.Reg)
			d213 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
			ctx.BindReg(r14, &d213)
		}
		ctx.FreeDesc(&d211)
		ctx.FreeDesc(&d212)
		ctx.ReclaimUntrackedRegs()
		d214 = d213
		ctx.SyncDesc(&d214)
		if d214.Loc != LocImm && d214.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl43 := ctx.ReserveLabel()
		lbl44 := ctx.ReserveLabel()
		if d214.Loc == LocImm {
			if d214.Imm.Bool() {
				ctx.MarkLabel(lbl43)
				ctx.EmitJmp(lbl35)
			} else {
				ctx.MarkLabel(lbl44)
				ctx.EmitJmp(lbl36)
			}
		} else {
			ctx.EmitJump(d214.Condition, lbl43)
			ctx.EmitJmp(lbl44)
			ctx.FreeDesc(&d213)
			ctx.MarkLabel(lbl43)
			ctx.EmitJmp(lbl35)
			ctx.MarkLabel(lbl44)
			ctx.EmitJmp(lbl36)
		}
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl36)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d207.Loc != LocImm {
			ctx.EnsureDesc(&d207)
		}
		if d207.Loc == LocImm {
			d215 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d207.Imm.Int())}
		} else if d207.Type == tagInt && d207.Loc == LocRegPair {
			ctx.FreeReg(d207.Reg)
			d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d207.Reg2}
			ctx.BindReg(d207.Reg2, &d215)
			ctx.BindReg(d207.Reg2, &d215)
		} else if d207.Type == tagInt && d207.Loc == LocReg {
			d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d207.Reg}
			ctx.BindReg(d207.Reg, &d215)
			ctx.BindReg(d207.Reg, &d215)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d207)
			d215 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d215.Type = tagInt
			ctx.BindReg(d215.Reg, &d215)
		}
		ctx.ReclaimUntrackedRegs()
		if d209.Loc != LocImm {
			ctx.EnsureDesc(&d209)
		}
		if d209.Loc == LocImm {
			d216 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d209.Imm.Int())}
		} else if d209.Type == tagInt && d209.Loc == LocRegPair {
			ctx.FreeReg(d209.Reg)
			d216 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d209.Reg2}
			ctx.BindReg(d209.Reg2, &d216)
			ctx.BindReg(d209.Reg2, &d216)
		} else if d209.Type == tagInt && d209.Loc == LocReg {
			d216 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d209.Reg}
			ctx.BindReg(d209.Reg, &d216)
			ctx.BindReg(d209.Reg, &d216)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d209)
			d216 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d216.Type = tagInt
			ctx.BindReg(d216.Reg, &d216)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d215)
		ctx.EnsureDesc(&d216)
		ctx.EnsureDescsTogether(&d215, &d216)
		if d215.Loc == LocImm && d216.Loc == LocImm {
			d217 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d215.Imm.Int() > d216.Imm.Int())}
		} else if d216.Loc == LocImm {
			r15 := ctx.AllocReg()
			if d216.Imm.Int() >= -2147483648 && d216.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d215.Reg, int32(d216.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d216.Imm.Int()))
				ctx.EmitCmpInt64(d215.Reg, ctx.ScratchReg)
			}
			d217 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedGreater}
			ctx.BindReg(r15, &d217)
		} else if d215.Loc == LocImm {
			r16 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d215.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d216.Reg)
			d217 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedGreater}
			ctx.BindReg(r16, &d217)
		} else {
			r17 := ctx.AllocReg()
			ctx.EmitCmpInt64(d215.Reg, d216.Reg)
			d217 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedGreater}
			ctx.BindReg(r17, &d217)
		}
		ctx.FreeDesc(&d215)
		ctx.FreeDesc(&d216)
		ctx.ReclaimUntrackedRegs()
		d218 = d217
		ctx.SyncDesc(&d218)
		if d218.Loc != LocImm && d218.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl45 := ctx.ReserveLabel()
		lbl46 := ctx.ReserveLabel()
		if d218.Loc == LocImm {
			if d218.Imm.Bool() {
				ctx.MarkLabel(lbl45)
				ctx.EmitJmp(lbl37)
			} else {
				ctx.MarkLabel(lbl46)
				ctx.EmitJmp(lbl38)
			}
		} else {
			ctx.EmitJump(d218.Condition, lbl45)
			ctx.EmitJmp(lbl46)
			ctx.FreeDesc(&d217)
			ctx.MarkLabel(lbl45)
			ctx.EmitJmp(lbl37)
			ctx.MarkLabel(lbl46)
			ctx.EmitJmp(lbl38)
		}
		bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl38)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d207 = JITPrepareScmerGoArg(ctx, d207)
		ctx.SyncDesc(&d207)
		d219 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d207}, 1)
		d219.NoHeapPointer = true
		ctx.BindReg(d219.Reg, &d219)
		ctx.ReclaimUntrackedRegs()
		d209 = JITPrepareScmerGoArg(ctx, d209)
		ctx.SyncDesc(&d209)
		d220 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d209}, 1)
		d220.NoHeapPointer = true
		ctx.BindReg(d220.Reg, &d220)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d219)
		ctx.EnsureDesc(&d220)
		ctx.EnsureDescsTogether(&d219, &d220)
		if d219.Loc == LocImm && d220.Loc == LocImm {
			d221 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d219.Imm.Int() < d220.Imm.Int())}
		} else if d220.Loc == LocImm {
			r18 := ctx.AllocReg()
			if d220.Imm.Int() >= -2147483648 && d220.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d219.Reg, int32(d220.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d220.Imm.Int()))
				ctx.EmitCmpInt64(d219.Reg, ctx.ScratchReg)
			}
			d221 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
			ctx.BindReg(r18, &d221)
		} else if d219.Loc == LocImm {
			r19 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d219.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d220.Reg)
			d221 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedLess}
			ctx.BindReg(r19, &d221)
		} else {
			r20 := ctx.AllocReg()
			ctx.EmitCmpInt64(d219.Reg, d220.Reg)
			d221 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
			ctx.BindReg(r20, &d221)
		}
		ctx.FreeDesc(&d219)
		ctx.FreeDesc(&d220)
		ctx.ReclaimUntrackedRegs()
		d222 = d221
		ctx.SyncDesc(&d222)
		if d222.Loc != LocImm && d222.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl47 := ctx.ReserveLabel()
		lbl48 := ctx.ReserveLabel()
		if d222.Loc == LocImm {
			if d222.Imm.Bool() {
				ctx.MarkLabel(lbl47)
				ctx.EmitJmp(lbl39)
			} else {
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl40)
			}
		} else {
			ctx.EmitJump(d222.Condition, lbl47)
			ctx.EmitJmp(lbl48)
			ctx.FreeDesc(&d221)
			ctx.MarkLabel(lbl47)
			ctx.EmitJmp(lbl39)
			ctx.MarkLabel(lbl48)
			ctx.EmitJmp(lbl40)
		}
		bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl40)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d207 = JITPrepareScmerGoArg(ctx, d207)
		ctx.SyncDesc(&d207)
		d223 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d207}, 1)
		d223.NoHeapPointer = true
		ctx.BindReg(d223.Reg, &d223)
		ctx.ReclaimUntrackedRegs()
		d209 = JITPrepareScmerGoArg(ctx, d209)
		ctx.SyncDesc(&d209)
		d224 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d209}, 1)
		d224.NoHeapPointer = true
		ctx.BindReg(d224.Reg, &d224)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d223)
		ctx.EnsureDesc(&d224)
		ctx.EnsureDescsTogether(&d223, &d224)
		if d223.Loc == LocImm && d224.Loc == LocImm {
			d225 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d223.Imm.Int() > d224.Imm.Int())}
		} else if d224.Loc == LocImm {
			r21 := ctx.AllocReg()
			if d224.Imm.Int() >= -2147483648 && d224.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d223.Reg, int32(d224.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d224.Imm.Int()))
				ctx.EmitCmpInt64(d223.Reg, ctx.ScratchReg)
			}
			d225 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedGreater}
			ctx.BindReg(r21, &d225)
		} else if d223.Loc == LocImm {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d223.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d224.Reg)
			d225 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedGreater}
			ctx.BindReg(r22, &d225)
		} else {
			r23 := ctx.AllocReg()
			ctx.EmitCmpInt64(d223.Reg, d224.Reg)
			d225 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedGreater}
			ctx.BindReg(r23, &d225)
		}
		ctx.FreeDesc(&d223)
		ctx.FreeDesc(&d224)
		ctx.ReclaimUntrackedRegs()
		d226 = d225
		ctx.SyncDesc(&d226)
		if d226.Loc != LocImm && d226.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl49 := ctx.ReserveLabel()
		lbl50 := ctx.ReserveLabel()
		if d226.Loc == LocImm {
			if d226.Imm.Bool() {
				ctx.MarkLabel(lbl49)
				ctx.EmitJmp(lbl41)
			} else {
				ctx.MarkLabel(lbl50)
				ctx.EmitJmp(lbl42)
			}
		} else {
			ctx.EmitJump(d226.Condition, lbl49)
			ctx.EmitJmp(lbl50)
			ctx.FreeDesc(&d225)
			ctx.MarkLabel(lbl49)
			ctx.EmitJmp(lbl41)
			ctx.MarkLabel(lbl50)
			ctx.EmitJmp(lbl42)
		}
		bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl42)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r24 := ctx.AllocReg()
		d227 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
		ctx.EnsureDesc(&d227)
		if d227.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r24, d227)
		}
		ctx.EmitJmp(lbl33)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl35)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d228 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d228)
		if d228.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r24, d228)
		}
		ctx.EmitJmp(lbl33)
		bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl37)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d229 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d229)
		if d229.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r24, d229)
		}
		ctx.EmitJmp(lbl33)
		bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl39)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d230 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d230)
		if d230.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r24, d230)
		}
		ctx.EmitJmp(lbl33)
		bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl41)
		ctx.ResolveFixups()
		d207 = inlineHome208
		d209 = inlineHome210
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d231 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d231)
		if d231.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r24, d231)
		}
		ctx.EmitJmp(lbl33)
		ctx.MarkLabel(lbl33)
		d232 = JITValueDesc{Loc: LocReg, Reg: r24}
		ctx.BindReg(r24, &d232)
		ctx.BindReg(r24, &d232)
		ctx.EnsureDesc(&d232)
		resultTarget233 := false
		_ = resultTarget233
		if d232.Loc == LocImm {
			d234 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d232.Imm.Int() < 0)}
		} else {
			r25 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d232.Reg, 0)
			d234 = ctx.DeferBooleanFlags(r25, CondSignedLess)
			ctx.BindReg(r25, &d234)
		}
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d235 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1))}
		} else {
			r26 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 1)
			d235 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondEqual}
			ctx.BindReg(r26, &d235)
		}
		d236 = d235
		ctx.SyncDesc(&d236)
		if d236.Loc != LocImm && d236.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d236.Loc == LocImm {
			if d236.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[17].Render()
		}
		ctx.EmitJump(d236.Condition, lbl16)
		if bbs[17].Rendered {
			ctx.EmitJmp(lbl18)
		}
		ctx.FreeDesc(&d235)
		ctx.FlushRegisterMoves()
		if !bbs[17].Rendered {
			snap237 := d1
			snap238 := d2
			snap239 := d3
			snap240 := d4
			snap241 := d5
			snap242 := d6
			snap243 := d7
			snap244 := d8
			snap245 := d18
			snap246 := d19
			snap247 := d20
			snap248 := d33
			snap249 := d34
			snap250 := d49
			snap251 := d50
			snap252 := d51
			snap253 := d69
			snap254 := d70
			snap255 := d71
			snap256 := d92
			snap257 := d93
			snap258 := d116
			snap259 := d117
			snap260 := d142
			snap261 := d143
			snap262 := d170
			snap263 := d171
			snap264 := d173
			snap265 := d174
			snap266 := d175
			snap267 := d207
			snap268 := d209
			snap269 := d211
			snap270 := d212
			snap271 := d213
			snap272 := d214
			snap273 := d215
			snap274 := d216
			snap275 := d217
			snap276 := d218
			snap277 := d219
			snap278 := d220
			snap279 := d221
			snap280 := d222
			snap281 := d223
			snap282 := d224
			snap283 := d225
			snap284 := d226
			snap285 := d227
			snap286 := d228
			snap287 := d229
			snap288 := d230
			snap289 := d231
			snap290 := d232
			snap291 := d234
			snap292 := d235
			snap293 := d236
			alloc294 := ctx.SnapshotAllocState()
			bbs[17].Render()
			ctx.RestoreAllocState(alloc294)
			d1 = snap237
			d2 = snap238
			d3 = snap239
			d4 = snap240
			d5 = snap241
			d6 = snap242
			d7 = snap243
			d8 = snap244
			d18 = snap245
			d19 = snap246
			d20 = snap247
			d33 = snap248
			d34 = snap249
			d49 = snap250
			d50 = snap251
			d51 = snap252
			d69 = snap253
			d70 = snap254
			d71 = snap255
			d92 = snap256
			d93 = snap257
			d116 = snap258
			d117 = snap259
			d142 = snap260
			d143 = snap261
			d170 = snap262
			d171 = snap263
			d173 = snap264
			d174 = snap265
			d175 = snap266
			d207 = snap267
			d209 = snap268
			d211 = snap269
			d212 = snap270
			d213 = snap271
			d214 = snap272
			d215 = snap273
			d216 = snap274
			d217 = snap275
			d218 = snap276
			d219 = snap277
			d220 = snap278
			d221 = snap279
			d222 = snap280
			d223 = snap281
			d224 = snap282
			d225 = snap283
			d226 = snap284
			d227 = snap285
			d228 = snap286
			d229 = snap287
			d230 = snap288
			d231 = snap289
			d232 = snap290
			d234 = snap291
			d235 = snap292
			d236 = snap293
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
		d296 = args[1]
		ctx.SyncDesc(&d296)
		if d296.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d296.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d296.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d296 = tmpScalar
		}
		d296 = JITPrepareScmerGoArg(ctx, d296)
		if d296.Loc != LocRegPair && d296.Loc != LocStackPair && d296.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d295 = ctx.EmitGoCallScalar(jitGoABI_0cbff6f55ee96f6d, []JITValueDesc{d296}, 2)
		ctx.EnsureDesc(&d295)
		if d295.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d295.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d295.Imm)
			ptrWord, _ := d295.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d295.Imm.String())))
			d295 = tmpPair
		} else if d295.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d295.Type, Reg: ctx.AllocRegExcept(d295.Reg), Reg2: ctx.AllocRegExcept(d295.Reg)}
			switch d295.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d295)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d295)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d295)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d295)
			d295 = tmpPair
		}
		if d295.Loc != LocRegPair && d295.Loc != LocStackPair && d295.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateValue arg0)")
		}
		ctx.SyncDesc(&d295)
		callResults297 := JITEmitGoCallResults(ctx, jitGoABI_58434e8dfe39c263, []JITValueDesc{d295})
		d298 = callResults297[0]
		_ = d298
		d299 = callResults297[1]
		_ = d299
		ctx.StabilizeDescForControlFlow(&d298)
		d300 = d299
		ctx.EnsureDesc(&d300)
		if d300.Loc != LocImm && d300.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d300.Loc == LocImm {
			if d300.Imm.Bool() {
				return bbs[18].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitCmpRegImm32(d300.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl19)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap301 := d1
			snap302 := d2
			snap303 := d3
			snap304 := d4
			snap305 := d5
			snap306 := d6
			snap307 := d7
			snap308 := d8
			snap309 := d18
			snap310 := d19
			snap311 := d20
			snap312 := d33
			snap313 := d34
			snap314 := d49
			snap315 := d50
			snap316 := d51
			snap317 := d69
			snap318 := d70
			snap319 := d71
			snap320 := d92
			snap321 := d93
			snap322 := d116
			snap323 := d117
			snap324 := d142
			snap325 := d143
			snap326 := d170
			snap327 := d171
			snap328 := d173
			snap329 := d174
			snap330 := d175
			snap331 := d207
			snap332 := d209
			snap333 := d211
			snap334 := d212
			snap335 := d213
			snap336 := d214
			snap337 := d215
			snap338 := d216
			snap339 := d217
			snap340 := d218
			snap341 := d219
			snap342 := d220
			snap343 := d221
			snap344 := d222
			snap345 := d223
			snap346 := d224
			snap347 := d225
			snap348 := d226
			snap349 := d227
			snap350 := d228
			snap351 := d229
			snap352 := d230
			snap353 := d231
			snap354 := d232
			snap355 := d234
			snap356 := d235
			snap357 := d236
			snap358 := d295
			snap359 := d296
			snap360 := d298
			snap361 := d299
			snap362 := d300
			alloc363 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc363)
			d1 = snap301
			d2 = snap302
			d3 = snap303
			d4 = snap304
			d5 = snap305
			d6 = snap306
			d7 = snap307
			d8 = snap308
			d18 = snap309
			d19 = snap310
			d20 = snap311
			d33 = snap312
			d34 = snap313
			d49 = snap314
			d50 = snap315
			d51 = snap316
			d69 = snap317
			d70 = snap318
			d71 = snap319
			d92 = snap320
			d93 = snap321
			d116 = snap322
			d117 = snap323
			d142 = snap324
			d143 = snap325
			d170 = snap326
			d171 = snap327
			d173 = snap328
			d174 = snap329
			d175 = snap330
			d207 = snap331
			d209 = snap332
			d211 = snap333
			d212 = snap334
			d213 = snap335
			d214 = snap336
			d215 = snap337
			d216 = snap338
			d217 = snap339
			d218 = snap340
			d219 = snap341
			d220 = snap342
			d221 = snap343
			d222 = snap344
			d223 = snap345
			d224 = snap346
			d225 = snap347
			d226 = snap348
			d227 = snap349
			d228 = snap350
			d229 = snap351
			d230 = snap352
			d231 = snap353
			d232 = snap354
			d234 = snap355
			d235 = snap356
			d236 = snap357
			d295 = snap358
			d296 = snap359
			d298 = snap360
			d299 = snap361
			d300 = snap362
		}
		if !bbs[18].Rendered {
			return bbs[18].Render()
		}
		return result
		ctx.FreeDesc(&d299)
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
		d364 = ctx.EmitFloatDesc(args[0])
		d365 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d364)
		resultTarget366 := false
		_ = resultTarget366
		ctx.EnsureDesc(&d365)
		ctx.EnsureDescsTogether(&d364, &d365)
		if d364.Loc == LocImm && d365.Loc == LocImm {
			d367 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d364.Imm.Float() < d365.Imm.Float())}
		} else if d365.Loc == LocImm {
			var r27 Reg
			if result.Loc == LocReg && result.Reg != d364.Reg {
				r27 = result.Reg
				resultTarget366 = true
			} else {
				r27 = ctx.AllocRegExcept(d364.Reg)
			}
			_, yBits := d365.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d364.Reg)
			d367 = ctx.DeferBooleanFlags(r27, CondUnsignedAbove)
			ctx.BindReg(r27, &d367)
		} else if d364.Loc == LocImm {
			var r28 Reg
			if result.Loc == LocReg && result.Reg != d365.Reg {
				r28 = result.Reg
				resultTarget366 = true
			} else {
				r28 = ctx.AllocRegExcept(d365.Reg)
			}
			_, xBits := d364.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d365.Reg, ctx.ScratchReg)
			d367 = ctx.DeferBooleanFlags(r28, CondUnsignedAbove)
			ctx.BindReg(r28, &d367)
		} else {
			var r29 Reg
			if result.Loc == LocReg && result.Reg != d364.Reg && result.Reg != d365.Reg {
				r29 = result.Reg
				resultTarget366 = true
			} else {
				r29 = ctx.AllocRegExcept(d364.Reg, d365.Reg)
			}
			ctx.EmitCmpFloat64(d365.Reg, d364.Reg)
			d367 = ctx.DeferBooleanFlags(r29, CondUnsignedAbove)
			ctx.BindReg(r29, &d367)
		}
		ctx.FreeDesc(&d364)
		ctx.FreeDesc(&d365)
		ctx.SyncDesc(&d367)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d367) {
			return d367
		}
		ctx.EnsureDesc(&d367)
		ctx.EmitMovToReg(result.Reg, d367)
		result.Type = d367.Type
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d368 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x2))}
		} else {
			r30 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 2)
			d368 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondEqual}
			ctx.BindReg(r30, &d368)
		}
		d369 = d368
		ctx.SyncDesc(&d369)
		if d369.Loc != LocImm && d369.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d369.Loc == LocImm {
			if d369.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitJump(d369.Condition, lbl16)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d368)
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap370 := d1
			snap371 := d2
			snap372 := d3
			snap373 := d4
			snap374 := d5
			snap375 := d6
			snap376 := d7
			snap377 := d8
			snap378 := d18
			snap379 := d19
			snap380 := d20
			snap381 := d33
			snap382 := d34
			snap383 := d49
			snap384 := d50
			snap385 := d51
			snap386 := d69
			snap387 := d70
			snap388 := d71
			snap389 := d92
			snap390 := d93
			snap391 := d116
			snap392 := d117
			snap393 := d142
			snap394 := d143
			snap395 := d170
			snap396 := d171
			snap397 := d173
			snap398 := d174
			snap399 := d175
			snap400 := d207
			snap401 := d209
			snap402 := d211
			snap403 := d212
			snap404 := d213
			snap405 := d214
			snap406 := d215
			snap407 := d216
			snap408 := d217
			snap409 := d218
			snap410 := d219
			snap411 := d220
			snap412 := d221
			snap413 := d222
			snap414 := d223
			snap415 := d224
			snap416 := d225
			snap417 := d226
			snap418 := d227
			snap419 := d228
			snap420 := d229
			snap421 := d230
			snap422 := d231
			snap423 := d232
			snap424 := d234
			snap425 := d235
			snap426 := d236
			snap427 := d295
			snap428 := d296
			snap429 := d298
			snap430 := d299
			snap431 := d300
			snap432 := d364
			snap433 := d365
			snap434 := d367
			snap435 := d368
			snap436 := d369
			alloc437 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc437)
			d1 = snap370
			d2 = snap371
			d3 = snap372
			d4 = snap373
			d5 = snap374
			d6 = snap375
			d7 = snap376
			d8 = snap377
			d18 = snap378
			d19 = snap379
			d20 = snap380
			d33 = snap381
			d34 = snap382
			d49 = snap383
			d50 = snap384
			d51 = snap385
			d69 = snap386
			d70 = snap387
			d71 = snap388
			d92 = snap389
			d93 = snap390
			d116 = snap391
			d117 = snap392
			d142 = snap393
			d143 = snap394
			d170 = snap395
			d171 = snap396
			d173 = snap397
			d174 = snap398
			d175 = snap399
			d207 = snap400
			d209 = snap401
			d211 = snap402
			d212 = snap403
			d213 = snap404
			d214 = snap405
			d215 = snap406
			d216 = snap407
			d217 = snap408
			d218 = snap409
			d219 = snap410
			d220 = snap411
			d221 = snap412
			d222 = snap413
			d223 = snap414
			d224 = snap415
			d225 = snap416
			d226 = snap417
			d227 = snap418
			d228 = snap419
			d229 = snap420
			d230 = snap421
			d231 = snap422
			d232 = snap423
			d234 = snap424
			d235 = snap425
			d236 = snap426
			d295 = snap427
			d296 = snap428
			d298 = snap429
			d299 = snap430
			d300 = snap431
			d364 = snap432
			d365 = snap433
			d367 = snap434
			d368 = snap435
			d369 = snap436
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
		ctx.EnsureDesc(&args[0])
		ctx.EnsureDesc(&d298)
		d438 = args[0]
		_ = d438
		ctx.StabilizeDescForControlFlow(&d438)
		inlineHome439 := d438
		inlineHome439.ID = 0
		d440 = d298
		_ = d440
		ctx.StabilizeDescForControlFlow(&d440)
		inlineHome441 := d440
		inlineHome441.ID = 0
		lbl51 := ctx.ReserveLabel()
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl52 := ctx.ReserveLabel()
		_ = lbl52
		bbpos_2_1 := int32(-1)
		_ = bbpos_2_1
		lbl53 := ctx.ReserveLabel()
		_ = lbl53
		bbpos_2_2 := int32(-1)
		_ = bbpos_2_2
		lbl54 := ctx.ReserveLabel()
		_ = lbl54
		bbpos_2_3 := int32(-1)
		_ = bbpos_2_3
		lbl55 := ctx.ReserveLabel()
		_ = lbl55
		bbpos_2_4 := int32(-1)
		_ = bbpos_2_4
		lbl56 := ctx.ReserveLabel()
		_ = lbl56
		bbpos_2_5 := int32(-1)
		_ = bbpos_2_5
		lbl57 := ctx.ReserveLabel()
		_ = lbl57
		bbpos_2_6 := int32(-1)
		_ = bbpos_2_6
		lbl58 := ctx.ReserveLabel()
		_ = lbl58
		bbpos_2_7 := int32(-1)
		_ = bbpos_2_7
		lbl59 := ctx.ReserveLabel()
		_ = lbl59
		bbpos_2_8 := int32(-1)
		_ = bbpos_2_8
		lbl60 := ctx.ReserveLabel()
		_ = lbl60
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl52)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d438.Loc != LocImm {
			ctx.EnsureDesc(&d438)
		}
		if d438.Loc == LocImm {
			d442 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d438.Imm.Int())}
		} else if d438.Type == tagInt && d438.Loc == LocRegPair {
			ctx.FreeReg(d438.Reg)
			d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d438.Reg2}
			ctx.BindReg(d438.Reg2, &d442)
			ctx.BindReg(d438.Reg2, &d442)
		} else if d438.Type == tagInt && d438.Loc == LocReg {
			d442 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d438.Reg}
			ctx.BindReg(d438.Reg, &d442)
			ctx.BindReg(d438.Reg, &d442)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d438)
			d442 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d442.Type = tagInt
			ctx.BindReg(d442.Reg, &d442)
		}
		ctx.ReclaimUntrackedRegs()
		if d440.Loc != LocImm {
			ctx.EnsureDesc(&d440)
		}
		if d440.Loc == LocImm {
			d443 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d440.Imm.Int())}
		} else if d440.Type == tagInt && d440.Loc == LocRegPair {
			ctx.FreeReg(d440.Reg)
			d443 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d440.Reg2}
			ctx.BindReg(d440.Reg2, &d443)
			ctx.BindReg(d440.Reg2, &d443)
		} else if d440.Type == tagInt && d440.Loc == LocReg {
			d443 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d440.Reg}
			ctx.BindReg(d440.Reg, &d443)
			ctx.BindReg(d440.Reg, &d443)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d440)
			d443 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d443.Type = tagInt
			ctx.BindReg(d443.Reg, &d443)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d442)
		ctx.EnsureDesc(&d443)
		ctx.EnsureDescsTogether(&d442, &d443)
		if d442.Loc == LocImm && d443.Loc == LocImm {
			d444 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d442.Imm.Int() < d443.Imm.Int())}
		} else if d443.Loc == LocImm {
			r31 := ctx.AllocReg()
			if d443.Imm.Int() >= -2147483648 && d443.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d442.Reg, int32(d443.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d443.Imm.Int()))
				ctx.EmitCmpInt64(d442.Reg, ctx.ScratchReg)
			}
			d444 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondSignedLess}
			ctx.BindReg(r31, &d444)
		} else if d442.Loc == LocImm {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d442.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d443.Reg)
			d444 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
			ctx.BindReg(r32, &d444)
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitCmpInt64(d442.Reg, d443.Reg)
			d444 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
			ctx.BindReg(r33, &d444)
		}
		ctx.FreeDesc(&d442)
		ctx.FreeDesc(&d443)
		ctx.ReclaimUntrackedRegs()
		d445 = d444
		ctx.SyncDesc(&d445)
		if d445.Loc != LocImm && d445.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl61 := ctx.ReserveLabel()
		lbl62 := ctx.ReserveLabel()
		if d445.Loc == LocImm {
			if d445.Imm.Bool() {
				ctx.MarkLabel(lbl61)
				ctx.EmitJmp(lbl53)
			} else {
				ctx.MarkLabel(lbl62)
				ctx.EmitJmp(lbl54)
			}
		} else {
			ctx.EmitJump(d445.Condition, lbl61)
			ctx.EmitJmp(lbl62)
			ctx.FreeDesc(&d444)
			ctx.MarkLabel(lbl61)
			ctx.EmitJmp(lbl53)
			ctx.MarkLabel(lbl62)
			ctx.EmitJmp(lbl54)
		}
		bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl54)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d438.Loc != LocImm {
			ctx.EnsureDesc(&d438)
		}
		if d438.Loc == LocImm {
			d446 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d438.Imm.Int())}
		} else if d438.Type == tagInt && d438.Loc == LocRegPair {
			ctx.FreeReg(d438.Reg)
			d446 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d438.Reg2}
			ctx.BindReg(d438.Reg2, &d446)
			ctx.BindReg(d438.Reg2, &d446)
		} else if d438.Type == tagInt && d438.Loc == LocReg {
			d446 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d438.Reg}
			ctx.BindReg(d438.Reg, &d446)
			ctx.BindReg(d438.Reg, &d446)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d438)
			d446 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d446.Type = tagInt
			ctx.BindReg(d446.Reg, &d446)
		}
		ctx.ReclaimUntrackedRegs()
		if d440.Loc != LocImm {
			ctx.EnsureDesc(&d440)
		}
		if d440.Loc == LocImm {
			d447 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d440.Imm.Int())}
		} else if d440.Type == tagInt && d440.Loc == LocRegPair {
			ctx.FreeReg(d440.Reg)
			d447 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d440.Reg2}
			ctx.BindReg(d440.Reg2, &d447)
			ctx.BindReg(d440.Reg2, &d447)
		} else if d440.Type == tagInt && d440.Loc == LocReg {
			d447 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d440.Reg}
			ctx.BindReg(d440.Reg, &d447)
			ctx.BindReg(d440.Reg, &d447)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d440)
			d447 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d447.Type = tagInt
			ctx.BindReg(d447.Reg, &d447)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d446)
		ctx.EnsureDesc(&d447)
		ctx.EnsureDescsTogether(&d446, &d447)
		if d446.Loc == LocImm && d447.Loc == LocImm {
			d448 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d446.Imm.Int() > d447.Imm.Int())}
		} else if d447.Loc == LocImm {
			r34 := ctx.AllocReg()
			if d447.Imm.Int() >= -2147483648 && d447.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d446.Reg, int32(d447.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d447.Imm.Int()))
				ctx.EmitCmpInt64(d446.Reg, ctx.ScratchReg)
			}
			d448 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r34, Condition: CondSignedGreater}
			ctx.BindReg(r34, &d448)
		} else if d446.Loc == LocImm {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d446.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d447.Reg)
			d448 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedGreater}
			ctx.BindReg(r35, &d448)
		} else {
			r36 := ctx.AllocReg()
			ctx.EmitCmpInt64(d446.Reg, d447.Reg)
			d448 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedGreater}
			ctx.BindReg(r36, &d448)
		}
		ctx.FreeDesc(&d446)
		ctx.FreeDesc(&d447)
		ctx.ReclaimUntrackedRegs()
		d449 = d448
		ctx.SyncDesc(&d449)
		if d449.Loc != LocImm && d449.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl63 := ctx.ReserveLabel()
		lbl64 := ctx.ReserveLabel()
		if d449.Loc == LocImm {
			if d449.Imm.Bool() {
				ctx.MarkLabel(lbl63)
				ctx.EmitJmp(lbl55)
			} else {
				ctx.MarkLabel(lbl64)
				ctx.EmitJmp(lbl56)
			}
		} else {
			ctx.EmitJump(d449.Condition, lbl63)
			ctx.EmitJmp(lbl64)
			ctx.FreeDesc(&d448)
			ctx.MarkLabel(lbl63)
			ctx.EmitJmp(lbl55)
			ctx.MarkLabel(lbl64)
			ctx.EmitJmp(lbl56)
		}
		bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl56)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d438 = JITPrepareScmerGoArg(ctx, d438)
		ctx.SyncDesc(&d438)
		d450 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d438}, 1)
		d450.NoHeapPointer = true
		ctx.BindReg(d450.Reg, &d450)
		ctx.ReclaimUntrackedRegs()
		d440 = JITPrepareScmerGoArg(ctx, d440)
		ctx.SyncDesc(&d440)
		d451 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d440}, 1)
		d451.NoHeapPointer = true
		ctx.BindReg(d451.Reg, &d451)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d450)
		ctx.EnsureDesc(&d451)
		ctx.EnsureDescsTogether(&d450, &d451)
		if d450.Loc == LocImm && d451.Loc == LocImm {
			d452 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d450.Imm.Int() < d451.Imm.Int())}
		} else if d451.Loc == LocImm {
			r37 := ctx.AllocReg()
			if d451.Imm.Int() >= -2147483648 && d451.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d450.Reg, int32(d451.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d451.Imm.Int()))
				ctx.EmitCmpInt64(d450.Reg, ctx.ScratchReg)
			}
			d452 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedLess}
			ctx.BindReg(r37, &d452)
		} else if d450.Loc == LocImm {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d450.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d451.Reg)
			d452 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r38, Condition: CondSignedLess}
			ctx.BindReg(r38, &d452)
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitCmpInt64(d450.Reg, d451.Reg)
			d452 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r39, Condition: CondSignedLess}
			ctx.BindReg(r39, &d452)
		}
		ctx.FreeDesc(&d450)
		ctx.FreeDesc(&d451)
		ctx.ReclaimUntrackedRegs()
		d453 = d452
		ctx.SyncDesc(&d453)
		if d453.Loc != LocImm && d453.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl65 := ctx.ReserveLabel()
		lbl66 := ctx.ReserveLabel()
		if d453.Loc == LocImm {
			if d453.Imm.Bool() {
				ctx.MarkLabel(lbl65)
				ctx.EmitJmp(lbl57)
			} else {
				ctx.MarkLabel(lbl66)
				ctx.EmitJmp(lbl58)
			}
		} else {
			ctx.EmitJump(d453.Condition, lbl65)
			ctx.EmitJmp(lbl66)
			ctx.FreeDesc(&d452)
			ctx.MarkLabel(lbl65)
			ctx.EmitJmp(lbl57)
			ctx.MarkLabel(lbl66)
			ctx.EmitJmp(lbl58)
		}
		bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl58)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d438 = JITPrepareScmerGoArg(ctx, d438)
		ctx.SyncDesc(&d438)
		d454 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d438}, 1)
		d454.NoHeapPointer = true
		ctx.BindReg(d454.Reg, &d454)
		ctx.ReclaimUntrackedRegs()
		d440 = JITPrepareScmerGoArg(ctx, d440)
		ctx.SyncDesc(&d440)
		d455 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d440}, 1)
		d455.NoHeapPointer = true
		ctx.BindReg(d455.Reg, &d455)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d454)
		ctx.EnsureDesc(&d455)
		ctx.EnsureDescsTogether(&d454, &d455)
		if d454.Loc == LocImm && d455.Loc == LocImm {
			d456 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d454.Imm.Int() > d455.Imm.Int())}
		} else if d455.Loc == LocImm {
			r40 := ctx.AllocReg()
			if d455.Imm.Int() >= -2147483648 && d455.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d454.Reg, int32(d455.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d455.Imm.Int()))
				ctx.EmitCmpInt64(d454.Reg, ctx.ScratchReg)
			}
			d456 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r40, Condition: CondSignedGreater}
			ctx.BindReg(r40, &d456)
		} else if d454.Loc == LocImm {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d454.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d455.Reg)
			d456 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r41, Condition: CondSignedGreater}
			ctx.BindReg(r41, &d456)
		} else {
			r42 := ctx.AllocReg()
			ctx.EmitCmpInt64(d454.Reg, d455.Reg)
			d456 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r42, Condition: CondSignedGreater}
			ctx.BindReg(r42, &d456)
		}
		ctx.FreeDesc(&d454)
		ctx.FreeDesc(&d455)
		ctx.ReclaimUntrackedRegs()
		d457 = d456
		ctx.SyncDesc(&d457)
		if d457.Loc != LocImm && d457.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl67 := ctx.ReserveLabel()
		lbl68 := ctx.ReserveLabel()
		if d457.Loc == LocImm {
			if d457.Imm.Bool() {
				ctx.MarkLabel(lbl67)
				ctx.EmitJmp(lbl59)
			} else {
				ctx.MarkLabel(lbl68)
				ctx.EmitJmp(lbl60)
			}
		} else {
			ctx.EmitJump(d457.Condition, lbl67)
			ctx.EmitJmp(lbl68)
			ctx.FreeDesc(&d456)
			ctx.MarkLabel(lbl67)
			ctx.EmitJmp(lbl59)
			ctx.MarkLabel(lbl68)
			ctx.EmitJmp(lbl60)
		}
		bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl60)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r43 := ctx.AllocReg()
		d458 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
		ctx.EnsureDesc(&d458)
		if d458.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r43, d458)
		}
		ctx.EmitJmp(lbl51)
		bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl53)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d459 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d459)
		if d459.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r43, d459)
		}
		ctx.EmitJmp(lbl51)
		bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl55)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d460 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d460)
		if d460.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r43, d460)
		}
		ctx.EmitJmp(lbl51)
		bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl57)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d461 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d461)
		if d461.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r43, d461)
		}
		ctx.EmitJmp(lbl51)
		bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl59)
		ctx.ResolveFixups()
		d438 = inlineHome439
		d440 = inlineHome441
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d462 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d462)
		if d462.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r43, d462)
		}
		ctx.EmitJmp(lbl51)
		ctx.MarkLabel(lbl51)
		d463 = JITValueDesc{Loc: LocReg, Reg: r43}
		ctx.BindReg(r43, &d463)
		ctx.BindReg(r43, &d463)
		ctx.EnsureDesc(&d463)
		resultTarget464 := false
		_ = resultTarget464
		if d463.Loc == LocImm {
			d465 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d463.Imm.Int() < 0)}
		} else {
			r44 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d463.Reg, 0)
			d465 = ctx.DeferBooleanFlags(r44, CondSignedLess)
			ctx.BindReg(r44, &d465)
		}
		ctx.FreeDesc(&d463)
		ctx.SyncDesc(&d465)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d465) {
			return d465
		}
		ctx.EnsureDesc(&d465)
		ctx.EmitMovToReg(result.Reg, d465)
		result.Type = d465.Type
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
		d466 = ctx.EmitFloatDesc(args[0])
		d467 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d466)
		resultTarget468 := false
		_ = resultTarget468
		ctx.EnsureDesc(&d467)
		ctx.EnsureDescsTogether(&d466, &d467)
		if d466.Loc == LocImm && d467.Loc == LocImm {
			d469 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d466.Imm.Float() < d467.Imm.Float())}
		} else if d467.Loc == LocImm {
			var r45 Reg
			if result.Loc == LocReg && result.Reg != d466.Reg {
				r45 = result.Reg
				resultTarget468 = true
			} else {
				r45 = ctx.AllocRegExcept(d466.Reg)
			}
			_, yBits := d467.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d466.Reg)
			d469 = ctx.DeferBooleanFlags(r45, CondUnsignedAbove)
			ctx.BindReg(r45, &d469)
		} else if d466.Loc == LocImm {
			var r46 Reg
			if result.Loc == LocReg && result.Reg != d467.Reg {
				r46 = result.Reg
				resultTarget468 = true
			} else {
				r46 = ctx.AllocRegExcept(d467.Reg)
			}
			_, xBits := d466.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d467.Reg, ctx.ScratchReg)
			d469 = ctx.DeferBooleanFlags(r46, CondUnsignedAbove)
			ctx.BindReg(r46, &d469)
		} else {
			var r47 Reg
			if result.Loc == LocReg && result.Reg != d466.Reg && result.Reg != d467.Reg {
				r47 = result.Reg
				resultTarget468 = true
			} else {
				r47 = ctx.AllocRegExcept(d466.Reg, d467.Reg)
			}
			ctx.EmitCmpFloat64(d467.Reg, d466.Reg)
			d469 = ctx.DeferBooleanFlags(r47, CondUnsignedAbove)
			ctx.BindReg(r47, &d469)
		}
		ctx.FreeDesc(&d466)
		ctx.FreeDesc(&d467)
		ctx.SyncDesc(&d469)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d469) {
			return d469
		}
		ctx.EnsureDesc(&d469)
		ctx.EmitMovToReg(result.Reg, d469)
		result.Type = d469.Type
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
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d470 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x3))}
		} else {
			r48 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 3)
			d470 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r48, Condition: CondEqual}
			ctx.BindReg(r48, &d470)
		}
		d471 = d470
		ctx.SyncDesc(&d471)
		if d471.Loc != LocImm && d471.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d471.Loc == LocImm {
			if d471.Imm.Bool() {
				return bbs[19].Render()
			}
			return bbs[30].Render()
		}
		ctx.EmitJump(d471.Condition, lbl20)
		if bbs[30].Rendered {
			ctx.EmitJmp(lbl31)
		}
		ctx.FreeDesc(&d470)
		ctx.FlushRegisterMoves()
		if !bbs[30].Rendered {
			snap472 := d1
			snap473 := d2
			snap474 := d3
			snap475 := d4
			snap476 := d5
			snap477 := d6
			snap478 := d7
			snap479 := d8
			snap480 := d18
			snap481 := d19
			snap482 := d20
			snap483 := d33
			snap484 := d34
			snap485 := d49
			snap486 := d50
			snap487 := d51
			snap488 := d69
			snap489 := d70
			snap490 := d71
			snap491 := d92
			snap492 := d93
			snap493 := d116
			snap494 := d117
			snap495 := d142
			snap496 := d143
			snap497 := d170
			snap498 := d171
			snap499 := d173
			snap500 := d174
			snap501 := d175
			snap502 := d207
			snap503 := d209
			snap504 := d211
			snap505 := d212
			snap506 := d213
			snap507 := d214
			snap508 := d215
			snap509 := d216
			snap510 := d217
			snap511 := d218
			snap512 := d219
			snap513 := d220
			snap514 := d221
			snap515 := d222
			snap516 := d223
			snap517 := d224
			snap518 := d225
			snap519 := d226
			snap520 := d227
			snap521 := d228
			snap522 := d229
			snap523 := d230
			snap524 := d231
			snap525 := d232
			snap526 := d234
			snap527 := d235
			snap528 := d236
			snap529 := d295
			snap530 := d296
			snap531 := d298
			snap532 := d299
			snap533 := d300
			snap534 := d364
			snap535 := d365
			snap536 := d367
			snap537 := d368
			snap538 := d369
			snap539 := d438
			snap540 := d440
			snap541 := d442
			snap542 := d443
			snap543 := d444
			snap544 := d445
			snap545 := d446
			snap546 := d447
			snap547 := d448
			snap548 := d449
			snap549 := d450
			snap550 := d451
			snap551 := d452
			snap552 := d453
			snap553 := d454
			snap554 := d455
			snap555 := d456
			snap556 := d457
			snap557 := d458
			snap558 := d459
			snap559 := d460
			snap560 := d461
			snap561 := d462
			snap562 := d463
			snap563 := d465
			snap564 := d466
			snap565 := d467
			snap566 := d469
			snap567 := d470
			snap568 := d471
			alloc569 := ctx.SnapshotAllocState()
			bbs[30].Render()
			ctx.RestoreAllocState(alloc569)
			d1 = snap472
			d2 = snap473
			d3 = snap474
			d4 = snap475
			d5 = snap476
			d6 = snap477
			d7 = snap478
			d8 = snap479
			d18 = snap480
			d19 = snap481
			d20 = snap482
			d33 = snap483
			d34 = snap484
			d49 = snap485
			d50 = snap486
			d51 = snap487
			d69 = snap488
			d70 = snap489
			d71 = snap490
			d92 = snap491
			d93 = snap492
			d116 = snap493
			d117 = snap494
			d142 = snap495
			d143 = snap496
			d170 = snap497
			d171 = snap498
			d173 = snap499
			d174 = snap500
			d175 = snap501
			d207 = snap502
			d209 = snap503
			d211 = snap504
			d212 = snap505
			d213 = snap506
			d214 = snap507
			d215 = snap508
			d216 = snap509
			d217 = snap510
			d218 = snap511
			d219 = snap512
			d220 = snap513
			d221 = snap514
			d222 = snap515
			d223 = snap516
			d224 = snap517
			d225 = snap518
			d226 = snap519
			d227 = snap520
			d228 = snap521
			d229 = snap522
			d230 = snap523
			d231 = snap524
			d232 = snap525
			d234 = snap526
			d235 = snap527
			d236 = snap528
			d295 = snap529
			d296 = snap530
			d298 = snap531
			d299 = snap532
			d300 = snap533
			d364 = snap534
			d365 = snap535
			d367 = snap536
			d368 = snap537
			d369 = snap538
			d438 = snap539
			d440 = snap540
			d442 = snap541
			d443 = snap542
			d444 = snap543
			d445 = snap544
			d446 = snap545
			d447 = snap546
			d448 = snap547
			d449 = snap548
			d450 = snap549
			d451 = snap550
			d452 = snap551
			d453 = snap552
			d454 = snap553
			d455 = snap554
			d456 = snap555
			d457 = snap556
			d458 = snap557
			d459 = snap558
			d460 = snap559
			d461 = snap560
			d462 = snap561
			d463 = snap562
			d465 = snap563
			d466 = snap564
			d467 = snap565
			d469 = snap566
			d470 = snap567
			d471 = snap568
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d570 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d570 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d570)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d570 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d570)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d570 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d570.Type = tagInt
			ctx.BindReg(d570.Reg, &d570)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d571 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d571 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d571)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d571 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d571)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d571 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d571.Type = tagInt
			ctx.BindReg(d571.Reg, &d571)
		}
		ctx.EnsureDesc(&d570)
		ctx.EnsureDesc(&d571)
		ctx.EnsureDescsTogether(&d570, &d571)
		if d570.Loc == LocImm && d571.Loc == LocImm {
			d572 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d570.Imm.Int() < d571.Imm.Int())}
		} else if d571.Loc == LocImm {
			r49 := ctx.AllocReg()
			if d571.Imm.Int() >= -2147483648 && d571.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d570.Reg, int32(d571.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d571.Imm.Int()))
				ctx.EmitCmpInt64(d570.Reg, ctx.ScratchReg)
			}
			d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r49, Condition: CondSignedLess}
			ctx.BindReg(r49, &d572)
		} else if d570.Loc == LocImm {
			r50 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d570.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d571.Reg)
			d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r50, Condition: CondSignedLess}
			ctx.BindReg(r50, &d572)
		} else {
			r51 := ctx.AllocReg()
			ctx.EmitCmpInt64(d570.Reg, d571.Reg)
			d572 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r51, Condition: CondSignedLess}
			ctx.BindReg(r51, &d572)
		}
		ctx.FreeDesc(&d570)
		ctx.FreeDesc(&d571)
		d573 = d572
		ctx.SyncDesc(&d573)
		if d573.Loc != LocImm && d573.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d573.Loc == LocImm {
			if d573.Imm.Bool() {
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
				return bbs[24].Render()
			}
			return bbs[23].Render()
		}
		lbl69 := ctx.ReserveLabel()
		ctx.EmitJump(d573.Condition, lbl69)
		ctx.EmitJmp(lbl24)
		ctx.FreeDesc(&d572)
		snap574 := d1
		snap575 := d2
		snap576 := d3
		snap577 := d4
		snap578 := d5
		snap579 := d6
		snap580 := d7
		snap581 := d8
		snap582 := d18
		snap583 := d19
		snap584 := d20
		snap585 := d33
		snap586 := d34
		snap587 := d49
		snap588 := d50
		snap589 := d51
		snap590 := d69
		snap591 := d70
		snap592 := d71
		snap593 := d92
		snap594 := d93
		snap595 := d116
		snap596 := d117
		snap597 := d142
		snap598 := d143
		snap599 := d170
		snap600 := d171
		snap601 := d173
		snap602 := d174
		snap603 := d175
		snap604 := d207
		snap605 := d209
		snap606 := d211
		snap607 := d212
		snap608 := d213
		snap609 := d214
		snap610 := d215
		snap611 := d216
		snap612 := d217
		snap613 := d218
		snap614 := d219
		snap615 := d220
		snap616 := d221
		snap617 := d222
		snap618 := d223
		snap619 := d224
		snap620 := d225
		snap621 := d226
		snap622 := d227
		snap623 := d228
		snap624 := d229
		snap625 := d230
		snap626 := d231
		snap627 := d232
		snap628 := d234
		snap629 := d235
		snap630 := d236
		snap631 := d295
		snap632 := d296
		snap633 := d298
		snap634 := d299
		snap635 := d300
		snap636 := d364
		snap637 := d365
		snap638 := d367
		snap639 := d368
		snap640 := d369
		snap641 := d438
		snap642 := d440
		snap643 := d442
		snap644 := d443
		snap645 := d444
		snap646 := d445
		snap647 := d446
		snap648 := d447
		snap649 := d448
		snap650 := d449
		snap651 := d450
		snap652 := d451
		snap653 := d452
		snap654 := d453
		snap655 := d454
		snap656 := d455
		snap657 := d456
		snap658 := d457
		snap659 := d458
		snap660 := d459
		snap661 := d460
		snap662 := d461
		snap663 := d462
		snap664 := d463
		snap665 := d465
		snap666 := d466
		snap667 := d467
		snap668 := d469
		snap669 := d470
		snap670 := d471
		snap671 := d570
		snap672 := d571
		snap673 := d572
		snap674 := d573
		alloc675 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl69)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
		ctx.EmitJmp(lbl25)
		ctx.RestoreAllocState(alloc675)
		d1 = snap574
		d2 = snap575
		d3 = snap576
		d4 = snap577
		d5 = snap578
		d6 = snap579
		d7 = snap580
		d8 = snap581
		d18 = snap582
		d19 = snap583
		d20 = snap584
		d33 = snap585
		d34 = snap586
		d49 = snap587
		d50 = snap588
		d51 = snap589
		d69 = snap590
		d70 = snap591
		d71 = snap592
		d92 = snap593
		d93 = snap594
		d116 = snap595
		d117 = snap596
		d142 = snap597
		d143 = snap598
		d170 = snap599
		d171 = snap600
		d173 = snap601
		d174 = snap602
		d175 = snap603
		d207 = snap604
		d209 = snap605
		d211 = snap606
		d212 = snap607
		d213 = snap608
		d214 = snap609
		d215 = snap610
		d216 = snap611
		d217 = snap612
		d218 = snap613
		d219 = snap614
		d220 = snap615
		d221 = snap616
		d222 = snap617
		d223 = snap618
		d224 = snap619
		d225 = snap620
		d226 = snap621
		d227 = snap622
		d228 = snap623
		d229 = snap624
		d230 = snap625
		d231 = snap626
		d232 = snap627
		d234 = snap628
		d235 = snap629
		d236 = snap630
		d295 = snap631
		d296 = snap632
		d298 = snap633
		d299 = snap634
		d300 = snap635
		d364 = snap636
		d365 = snap637
		d367 = snap638
		d368 = snap639
		d369 = snap640
		d438 = snap641
		d440 = snap642
		d442 = snap643
		d443 = snap644
		d444 = snap645
		d445 = snap646
		d446 = snap647
		d447 = snap648
		d448 = snap649
		d449 = snap650
		d450 = snap651
		d451 = snap652
		d452 = snap653
		d453 = snap654
		d454 = snap655
		d455 = snap656
		d456 = snap657
		d457 = snap658
		d458 = snap659
		d459 = snap660
		d460 = snap661
		d461 = snap662
		d462 = snap663
		d463 = snap664
		d465 = snap665
		d466 = snap666
		d467 = snap667
		d469 = snap668
		d470 = snap669
		d471 = snap670
		d570 = snap671
		d571 = snap672
		d572 = snap673
		d573 = snap674
		if !bbs[24].Rendered {
			snap676 := d1
			snap677 := d2
			snap678 := d3
			snap679 := d4
			snap680 := d5
			snap681 := d6
			snap682 := d7
			snap683 := d8
			snap684 := d18
			snap685 := d19
			snap686 := d20
			snap687 := d33
			snap688 := d34
			snap689 := d49
			snap690 := d50
			snap691 := d51
			snap692 := d69
			snap693 := d70
			snap694 := d71
			snap695 := d92
			snap696 := d93
			snap697 := d116
			snap698 := d117
			snap699 := d142
			snap700 := d143
			snap701 := d170
			snap702 := d171
			snap703 := d173
			snap704 := d174
			snap705 := d175
			snap706 := d207
			snap707 := d209
			snap708 := d211
			snap709 := d212
			snap710 := d213
			snap711 := d214
			snap712 := d215
			snap713 := d216
			snap714 := d217
			snap715 := d218
			snap716 := d219
			snap717 := d220
			snap718 := d221
			snap719 := d222
			snap720 := d223
			snap721 := d224
			snap722 := d225
			snap723 := d226
			snap724 := d227
			snap725 := d228
			snap726 := d229
			snap727 := d230
			snap728 := d231
			snap729 := d232
			snap730 := d234
			snap731 := d235
			snap732 := d236
			snap733 := d295
			snap734 := d296
			snap735 := d298
			snap736 := d299
			snap737 := d300
			snap738 := d364
			snap739 := d365
			snap740 := d367
			snap741 := d368
			snap742 := d369
			snap743 := d438
			snap744 := d440
			snap745 := d442
			snap746 := d443
			snap747 := d444
			snap748 := d445
			snap749 := d446
			snap750 := d447
			snap751 := d448
			snap752 := d449
			snap753 := d450
			snap754 := d451
			snap755 := d452
			snap756 := d453
			snap757 := d454
			snap758 := d455
			snap759 := d456
			snap760 := d457
			snap761 := d458
			snap762 := d459
			snap763 := d460
			snap764 := d461
			snap765 := d462
			snap766 := d463
			snap767 := d465
			snap768 := d466
			snap769 := d467
			snap770 := d469
			snap771 := d470
			snap772 := d471
			snap773 := d570
			snap774 := d571
			snap775 := d572
			snap776 := d573
			alloc777 := ctx.SnapshotAllocState()
			bbs[24].Render()
			ctx.RestoreAllocState(alloc777)
			d1 = snap676
			d2 = snap677
			d3 = snap678
			d4 = snap679
			d5 = snap680
			d6 = snap681
			d7 = snap682
			d8 = snap683
			d18 = snap684
			d19 = snap685
			d20 = snap686
			d33 = snap687
			d34 = snap688
			d49 = snap689
			d50 = snap690
			d51 = snap691
			d69 = snap692
			d70 = snap693
			d71 = snap694
			d92 = snap695
			d93 = snap696
			d116 = snap697
			d117 = snap698
			d142 = snap699
			d143 = snap700
			d170 = snap701
			d171 = snap702
			d173 = snap703
			d174 = snap704
			d175 = snap705
			d207 = snap706
			d209 = snap707
			d211 = snap708
			d212 = snap709
			d213 = snap710
			d214 = snap711
			d215 = snap712
			d216 = snap713
			d217 = snap714
			d218 = snap715
			d219 = snap716
			d220 = snap717
			d221 = snap718
			d222 = snap719
			d223 = snap720
			d224 = snap721
			d225 = snap722
			d226 = snap723
			d227 = snap724
			d228 = snap725
			d229 = snap726
			d230 = snap727
			d231 = snap728
			d232 = snap729
			d234 = snap730
			d235 = snap731
			d236 = snap732
			d295 = snap733
			d296 = snap734
			d298 = snap735
			d299 = snap736
			d300 = snap737
			d364 = snap738
			d365 = snap739
			d367 = snap740
			d368 = snap741
			d369 = snap742
			d438 = snap743
			d440 = snap744
			d442 = snap745
			d443 = snap746
			d444 = snap747
			d445 = snap748
			d446 = snap749
			d447 = snap750
			d448 = snap751
			d449 = snap752
			d450 = snap753
			d451 = snap754
			d452 = snap755
			d453 = snap756
			d454 = snap757
			d455 = snap758
			d456 = snap759
			d457 = snap760
			d458 = snap761
			d459 = snap762
			d460 = snap763
			d461 = snap764
			d462 = snap765
			d463 = snap766
			d465 = snap767
			d466 = snap768
			d467 = snap769
			d469 = snap770
			d470 = snap771
			d471 = snap772
			d570 = snap773
			d571 = snap774
			d572 = snap775
			d573 = snap776
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
		ctx.EnsureDesc(&d6)
		if d6.Loc == LocImm {
			d778 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x4))}
		} else {
			r52 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 4)
			d778 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r52, Condition: CondEqual}
			ctx.BindReg(r52, &d778)
		}
		d779 = d778
		ctx.SyncDesc(&d779)
		if d779.Loc != LocImm && d779.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d779.Loc == LocImm {
			if d779.Imm.Bool() {
				return bbs[27].Render()
			}
			return bbs[28].Render()
		}
		ctx.EmitJump(d779.Condition, lbl28)
		if bbs[28].Rendered {
			ctx.EmitJmp(lbl29)
		}
		ctx.FreeDesc(&d778)
		ctx.FlushRegisterMoves()
		if !bbs[28].Rendered {
			snap780 := d1
			snap781 := d2
			snap782 := d3
			snap783 := d4
			snap784 := d5
			snap785 := d6
			snap786 := d7
			snap787 := d8
			snap788 := d18
			snap789 := d19
			snap790 := d20
			snap791 := d33
			snap792 := d34
			snap793 := d49
			snap794 := d50
			snap795 := d51
			snap796 := d69
			snap797 := d70
			snap798 := d71
			snap799 := d92
			snap800 := d93
			snap801 := d116
			snap802 := d117
			snap803 := d142
			snap804 := d143
			snap805 := d170
			snap806 := d171
			snap807 := d173
			snap808 := d174
			snap809 := d175
			snap810 := d207
			snap811 := d209
			snap812 := d211
			snap813 := d212
			snap814 := d213
			snap815 := d214
			snap816 := d215
			snap817 := d216
			snap818 := d217
			snap819 := d218
			snap820 := d219
			snap821 := d220
			snap822 := d221
			snap823 := d222
			snap824 := d223
			snap825 := d224
			snap826 := d225
			snap827 := d226
			snap828 := d227
			snap829 := d228
			snap830 := d229
			snap831 := d230
			snap832 := d231
			snap833 := d232
			snap834 := d234
			snap835 := d235
			snap836 := d236
			snap837 := d295
			snap838 := d296
			snap839 := d298
			snap840 := d299
			snap841 := d300
			snap842 := d364
			snap843 := d365
			snap844 := d367
			snap845 := d368
			snap846 := d369
			snap847 := d438
			snap848 := d440
			snap849 := d442
			snap850 := d443
			snap851 := d444
			snap852 := d445
			snap853 := d446
			snap854 := d447
			snap855 := d448
			snap856 := d449
			snap857 := d450
			snap858 := d451
			snap859 := d452
			snap860 := d453
			snap861 := d454
			snap862 := d455
			snap863 := d456
			snap864 := d457
			snap865 := d458
			snap866 := d459
			snap867 := d460
			snap868 := d461
			snap869 := d462
			snap870 := d463
			snap871 := d465
			snap872 := d466
			snap873 := d467
			snap874 := d469
			snap875 := d470
			snap876 := d471
			snap877 := d570
			snap878 := d571
			snap879 := d572
			snap880 := d573
			snap881 := d778
			snap882 := d779
			alloc883 := ctx.SnapshotAllocState()
			bbs[28].Render()
			ctx.RestoreAllocState(alloc883)
			d1 = snap780
			d2 = snap781
			d3 = snap782
			d4 = snap783
			d5 = snap784
			d6 = snap785
			d7 = snap786
			d8 = snap787
			d18 = snap788
			d19 = snap789
			d20 = snap790
			d33 = snap791
			d34 = snap792
			d49 = snap793
			d50 = snap794
			d51 = snap795
			d69 = snap796
			d70 = snap797
			d71 = snap798
			d92 = snap799
			d93 = snap800
			d116 = snap801
			d117 = snap802
			d142 = snap803
			d143 = snap804
			d170 = snap805
			d171 = snap806
			d173 = snap807
			d174 = snap808
			d175 = snap809
			d207 = snap810
			d209 = snap811
			d211 = snap812
			d212 = snap813
			d213 = snap814
			d214 = snap815
			d215 = snap816
			d216 = snap817
			d217 = snap818
			d218 = snap819
			d219 = snap820
			d220 = snap821
			d221 = snap822
			d222 = snap823
			d223 = snap824
			d224 = snap825
			d225 = snap826
			d226 = snap827
			d227 = snap828
			d228 = snap829
			d229 = snap830
			d230 = snap831
			d231 = snap832
			d232 = snap833
			d234 = snap834
			d235 = snap835
			d236 = snap836
			d295 = snap837
			d296 = snap838
			d298 = snap839
			d299 = snap840
			d300 = snap841
			d364 = snap842
			d365 = snap843
			d367 = snap844
			d368 = snap845
			d369 = snap846
			d438 = snap847
			d440 = snap848
			d442 = snap849
			d443 = snap850
			d444 = snap851
			d445 = snap852
			d446 = snap853
			d447 = snap854
			d448 = snap855
			d449 = snap856
			d450 = snap857
			d451 = snap858
			d452 = snap859
			d453 = snap860
			d454 = snap861
			d455 = snap862
			d456 = snap863
			d457 = snap864
			d458 = snap865
			d459 = snap866
			d460 = snap867
			d461 = snap868
			d462 = snap869
			d463 = snap870
			d465 = snap871
			d466 = snap872
			d467 = snap873
			d469 = snap874
			d470 = snap875
			d471 = snap876
			d570 = snap877
			d571 = snap878
			d572 = snap879
			d573 = snap880
			d778 = snap881
			d779 = snap882
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d884 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d884 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d884)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d884 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d884)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d884 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d884.Type = tagInt
			ctx.BindReg(d884.Reg, &d884)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d885 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d885 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d885)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d885 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d885)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d885 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d885.Type = tagInt
			ctx.BindReg(d885.Reg, &d885)
		}
		ctx.EnsureDesc(&d884)
		ctx.EnsureDesc(&d885)
		ctx.EnsureDescsTogether(&d884, &d885)
		if d884.Loc == LocImm && d885.Loc == LocImm {
			d886 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d884.Imm.Int() == d885.Imm.Int())}
		} else if d885.Loc == LocImm {
			r53 := ctx.AllocReg()
			if d885.Imm.Int() >= -2147483648 && d885.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d884.Reg, int32(d885.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d885.Imm.Int()))
				ctx.EmitCmpInt64(d884.Reg, ctx.ScratchReg)
			}
			d886 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r53, Condition: CondEqual}
			ctx.BindReg(r53, &d886)
		} else if d884.Loc == LocImm {
			r54 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d884.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d885.Reg)
			d886 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r54, Condition: CondEqual}
			ctx.BindReg(r54, &d886)
		} else {
			r55 := ctx.AllocReg()
			ctx.EmitCmpInt64(d884.Reg, d885.Reg)
			d886 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r55, Condition: CondEqual}
			ctx.BindReg(r55, &d886)
		}
		ctx.FreeDesc(&d884)
		ctx.FreeDesc(&d885)
		d887 = d886
		ctx.SyncDesc(&d887)
		if d887.Loc != LocImm && d887.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d887.Loc == LocImm {
			if d887.Imm.Bool() {
				return bbs[25].Render()
			}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
			return bbs[26].Render()
		}
		lbl70 := ctx.ReserveLabel()
		ctx.EmitJump(d887.Condition, lbl26)
		ctx.EmitJmp(lbl70)
		ctx.FreeDesc(&d886)
		snap888 := d1
		snap889 := d2
		snap890 := d3
		snap891 := d4
		snap892 := d5
		snap893 := d6
		snap894 := d7
		snap895 := d8
		snap896 := d18
		snap897 := d19
		snap898 := d20
		snap899 := d33
		snap900 := d34
		snap901 := d49
		snap902 := d50
		snap903 := d51
		snap904 := d69
		snap905 := d70
		snap906 := d71
		snap907 := d92
		snap908 := d93
		snap909 := d116
		snap910 := d117
		snap911 := d142
		snap912 := d143
		snap913 := d170
		snap914 := d171
		snap915 := d173
		snap916 := d174
		snap917 := d175
		snap918 := d207
		snap919 := d209
		snap920 := d211
		snap921 := d212
		snap922 := d213
		snap923 := d214
		snap924 := d215
		snap925 := d216
		snap926 := d217
		snap927 := d218
		snap928 := d219
		snap929 := d220
		snap930 := d221
		snap931 := d222
		snap932 := d223
		snap933 := d224
		snap934 := d225
		snap935 := d226
		snap936 := d227
		snap937 := d228
		snap938 := d229
		snap939 := d230
		snap940 := d231
		snap941 := d232
		snap942 := d234
		snap943 := d235
		snap944 := d236
		snap945 := d295
		snap946 := d296
		snap947 := d298
		snap948 := d299
		snap949 := d300
		snap950 := d364
		snap951 := d365
		snap952 := d367
		snap953 := d368
		snap954 := d369
		snap955 := d438
		snap956 := d440
		snap957 := d442
		snap958 := d443
		snap959 := d444
		snap960 := d445
		snap961 := d446
		snap962 := d447
		snap963 := d448
		snap964 := d449
		snap965 := d450
		snap966 := d451
		snap967 := d452
		snap968 := d453
		snap969 := d454
		snap970 := d455
		snap971 := d456
		snap972 := d457
		snap973 := d458
		snap974 := d459
		snap975 := d460
		snap976 := d461
		snap977 := d462
		snap978 := d463
		snap979 := d465
		snap980 := d466
		snap981 := d467
		snap982 := d469
		snap983 := d470
		snap984 := d471
		snap985 := d570
		snap986 := d571
		snap987 := d572
		snap988 := d573
		snap989 := d778
		snap990 := d779
		snap991 := d884
		snap992 := d885
		snap993 := d886
		snap994 := d887
		alloc995 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl70)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
		ctx.EmitJmp(lbl27)
		ctx.RestoreAllocState(alloc995)
		d1 = snap888
		d2 = snap889
		d3 = snap890
		d4 = snap891
		d5 = snap892
		d6 = snap893
		d7 = snap894
		d8 = snap895
		d18 = snap896
		d19 = snap897
		d20 = snap898
		d33 = snap899
		d34 = snap900
		d49 = snap901
		d50 = snap902
		d51 = snap903
		d69 = snap904
		d70 = snap905
		d71 = snap906
		d92 = snap907
		d93 = snap908
		d116 = snap909
		d117 = snap910
		d142 = snap911
		d143 = snap912
		d170 = snap913
		d171 = snap914
		d173 = snap915
		d174 = snap916
		d175 = snap917
		d207 = snap918
		d209 = snap919
		d211 = snap920
		d212 = snap921
		d213 = snap922
		d214 = snap923
		d215 = snap924
		d216 = snap925
		d217 = snap926
		d218 = snap927
		d219 = snap928
		d220 = snap929
		d221 = snap930
		d222 = snap931
		d223 = snap932
		d224 = snap933
		d225 = snap934
		d226 = snap935
		d227 = snap936
		d228 = snap937
		d229 = snap938
		d230 = snap939
		d231 = snap940
		d232 = snap941
		d234 = snap942
		d235 = snap943
		d236 = snap944
		d295 = snap945
		d296 = snap946
		d298 = snap947
		d299 = snap948
		d300 = snap949
		d364 = snap950
		d365 = snap951
		d367 = snap952
		d368 = snap953
		d369 = snap954
		d438 = snap955
		d440 = snap956
		d442 = snap957
		d443 = snap958
		d444 = snap959
		d445 = snap960
		d446 = snap961
		d447 = snap962
		d448 = snap963
		d449 = snap964
		d450 = snap965
		d451 = snap966
		d452 = snap967
		d453 = snap968
		d454 = snap969
		d455 = snap970
		d456 = snap971
		d457 = snap972
		d458 = snap973
		d459 = snap974
		d460 = snap975
		d461 = snap976
		d462 = snap977
		d463 = snap978
		d465 = snap979
		d466 = snap980
		d467 = snap981
		d469 = snap982
		d470 = snap983
		d471 = snap984
		d570 = snap985
		d571 = snap986
		d572 = snap987
		d573 = snap988
		d778 = snap989
		d779 = snap990
		d884 = snap991
		d885 = snap992
		d886 = snap993
		d887 = snap994
		if !bbs[26].Rendered {
			snap996 := d1
			snap997 := d2
			snap998 := d3
			snap999 := d4
			snap1000 := d5
			snap1001 := d6
			snap1002 := d7
			snap1003 := d8
			snap1004 := d18
			snap1005 := d19
			snap1006 := d20
			snap1007 := d33
			snap1008 := d34
			snap1009 := d49
			snap1010 := d50
			snap1011 := d51
			snap1012 := d69
			snap1013 := d70
			snap1014 := d71
			snap1015 := d92
			snap1016 := d93
			snap1017 := d116
			snap1018 := d117
			snap1019 := d142
			snap1020 := d143
			snap1021 := d170
			snap1022 := d171
			snap1023 := d173
			snap1024 := d174
			snap1025 := d175
			snap1026 := d207
			snap1027 := d209
			snap1028 := d211
			snap1029 := d212
			snap1030 := d213
			snap1031 := d214
			snap1032 := d215
			snap1033 := d216
			snap1034 := d217
			snap1035 := d218
			snap1036 := d219
			snap1037 := d220
			snap1038 := d221
			snap1039 := d222
			snap1040 := d223
			snap1041 := d224
			snap1042 := d225
			snap1043 := d226
			snap1044 := d227
			snap1045 := d228
			snap1046 := d229
			snap1047 := d230
			snap1048 := d231
			snap1049 := d232
			snap1050 := d234
			snap1051 := d235
			snap1052 := d236
			snap1053 := d295
			snap1054 := d296
			snap1055 := d298
			snap1056 := d299
			snap1057 := d300
			snap1058 := d364
			snap1059 := d365
			snap1060 := d367
			snap1061 := d368
			snap1062 := d369
			snap1063 := d438
			snap1064 := d440
			snap1065 := d442
			snap1066 := d443
			snap1067 := d444
			snap1068 := d445
			snap1069 := d446
			snap1070 := d447
			snap1071 := d448
			snap1072 := d449
			snap1073 := d450
			snap1074 := d451
			snap1075 := d452
			snap1076 := d453
			snap1077 := d454
			snap1078 := d455
			snap1079 := d456
			snap1080 := d457
			snap1081 := d458
			snap1082 := d459
			snap1083 := d460
			snap1084 := d461
			snap1085 := d462
			snap1086 := d463
			snap1087 := d465
			snap1088 := d466
			snap1089 := d467
			snap1090 := d469
			snap1091 := d470
			snap1092 := d471
			snap1093 := d570
			snap1094 := d571
			snap1095 := d572
			snap1096 := d573
			snap1097 := d778
			snap1098 := d779
			snap1099 := d884
			snap1100 := d885
			snap1101 := d886
			snap1102 := d887
			alloc1103 := ctx.SnapshotAllocState()
			bbs[26].Render()
			ctx.RestoreAllocState(alloc1103)
			d1 = snap996
			d2 = snap997
			d3 = snap998
			d4 = snap999
			d5 = snap1000
			d6 = snap1001
			d7 = snap1002
			d8 = snap1003
			d18 = snap1004
			d19 = snap1005
			d20 = snap1006
			d33 = snap1007
			d34 = snap1008
			d49 = snap1009
			d50 = snap1010
			d51 = snap1011
			d69 = snap1012
			d70 = snap1013
			d71 = snap1014
			d92 = snap1015
			d93 = snap1016
			d116 = snap1017
			d117 = snap1018
			d142 = snap1019
			d143 = snap1020
			d170 = snap1021
			d171 = snap1022
			d173 = snap1023
			d174 = snap1024
			d175 = snap1025
			d207 = snap1026
			d209 = snap1027
			d211 = snap1028
			d212 = snap1029
			d213 = snap1030
			d214 = snap1031
			d215 = snap1032
			d216 = snap1033
			d217 = snap1034
			d218 = snap1035
			d219 = snap1036
			d220 = snap1037
			d221 = snap1038
			d222 = snap1039
			d223 = snap1040
			d224 = snap1041
			d225 = snap1042
			d226 = snap1043
			d227 = snap1044
			d228 = snap1045
			d229 = snap1046
			d230 = snap1047
			d231 = snap1048
			d232 = snap1049
			d234 = snap1050
			d235 = snap1051
			d236 = snap1052
			d295 = snap1053
			d296 = snap1054
			d298 = snap1055
			d299 = snap1056
			d300 = snap1057
			d364 = snap1058
			d365 = snap1059
			d367 = snap1060
			d368 = snap1061
			d369 = snap1062
			d438 = snap1063
			d440 = snap1064
			d442 = snap1065
			d443 = snap1066
			d444 = snap1067
			d445 = snap1068
			d446 = snap1069
			d447 = snap1070
			d448 = snap1071
			d449 = snap1072
			d450 = snap1073
			d451 = snap1074
			d452 = snap1075
			d453 = snap1076
			d454 = snap1077
			d455 = snap1078
			d456 = snap1079
			d457 = snap1080
			d458 = snap1081
			d459 = snap1082
			d460 = snap1083
			d461 = snap1084
			d462 = snap1085
			d463 = snap1086
			d465 = snap1087
			d466 = snap1088
			d467 = snap1089
			d469 = snap1090
			d470 = snap1091
			d471 = snap1092
			d570 = snap1093
			d571 = snap1094
			d572 = snap1095
			d573 = snap1096
			d778 = snap1097
			d779 = snap1098
			d884 = snap1099
			d885 = snap1100
			d886 = snap1101
			d887 = snap1102
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
		args[1] = JITPrepareScmerGoArg(ctx, args[1])
		ctx.SyncDesc(&args[1])
		d1104 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{args[1]}, 1)
		d1104.NoHeapPointer = true
		ctx.BindReg(d1104.Reg, &d1104)
		ctx.EnsureDesc(&d1104)
		if d1104.Loc == LocImm {
			d1105 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1104.Imm.Int() > 0)}
		} else {
			ctx.EmitCmpRegImm32(d1104.Reg, 0)
			r56 := ctx.AllocRegExcept(d1104.Reg)
			ctx.EmitSetcc(r56, CondSignedGreater)
			d1105 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r56}
			ctx.BindReg(r56, &d1105)
		}
		ctx.EnsureDesc(&d1105)
		ctx.EmitStoreToStack(d1105, int32(bbs[26].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d1105)
		ctx.FreeDesc(&d1104)
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
		d1106 = d2
		if d1106.Loc == LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d1106)
		ctx.EmitStoreToStack(d1106, int32(bbs[24].PhiBase)+int32(0))
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d1107 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d1107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d1107)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d1107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d1107)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d1107 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1107.Type = tagInt
			ctx.BindReg(d1107.Reg, &d1107)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d1108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d1108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d1108)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d1108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d1108)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d1108 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1108.Type = tagInt
			ctx.BindReg(d1108.Reg, &d1108)
		}
		ctx.EnsureDesc(&d1107)
		resultTarget1109 := false
		_ = resultTarget1109
		ctx.EnsureDesc(&d1108)
		ctx.EnsureDescsTogether(&d1107, &d1108)
		if d1107.Loc == LocImm && d1108.Loc == LocImm {
			d1110 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1107.Imm.Int() < d1108.Imm.Int())}
		} else if d1108.Loc == LocImm {
			r57 := ctx.AllocReg()
			if d1108.Imm.Int() >= -2147483648 && d1108.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d1107.Reg, int32(d1108.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1108.Imm.Int()))
				ctx.EmitCmpInt64(d1107.Reg, ctx.ScratchReg)
			}
			d1110 = ctx.DeferBooleanFlags(r57, CondSignedLess)
			ctx.BindReg(r57, &d1110)
		} else if d1107.Loc == LocImm {
			r58 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1107.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d1108.Reg)
			d1110 = ctx.DeferBooleanFlags(r58, CondSignedLess)
			ctx.BindReg(r58, &d1110)
		} else {
			r59 := ctx.AllocReg()
			ctx.EmitCmpInt64(d1107.Reg, d1108.Reg)
			d1110 = ctx.DeferBooleanFlags(r59, CondSignedLess)
			ctx.BindReg(r59, &d1110)
		}
		ctx.FreeDesc(&d1107)
		ctx.FreeDesc(&d1108)
		ctx.SyncDesc(&d1110)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1110) {
			return d1110
		}
		ctx.EnsureDesc(&d1110)
		ctx.EmitMovToReg(result.Reg, d1110)
		result.Type = d1110.Type
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d1111 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d1111 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d1111)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d1111 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d1111)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d1111 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1111.Type = tagInt
			ctx.BindReg(d1111.Reg, &d1111)
		}
		ctx.EnsureDesc(&d1111)
		ctx.EnsureDesc(&d1111)
		if d1111.Loc == LocImm {
			d1112 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1111.Imm.Int()))}
		} else {
			var r60 Reg
			r60 = d1111.Reg
			d1111.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r60)
			d1112 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r60}
			ctx.BindReg(r60, &d1112)
		}
		ctx.FreeDesc(&d1111)
		d1113 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d1112)
		resultTarget1114 := false
		_ = resultTarget1114
		ctx.EnsureDesc(&d1113)
		ctx.EnsureDescsTogether(&d1112, &d1113)
		if d1112.Loc == LocImm && d1113.Loc == LocImm {
			d1115 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1112.Imm.Float() < d1113.Imm.Float())}
		} else if d1113.Loc == LocImm {
			var r61 Reg
			if result.Loc == LocReg && result.Reg != d1112.Reg {
				r61 = result.Reg
				resultTarget1114 = true
			} else {
				r61 = ctx.AllocRegExcept(d1112.Reg)
			}
			_, yBits := d1113.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d1112.Reg)
			d1115 = ctx.DeferBooleanFlags(r61, CondUnsignedAbove)
			ctx.BindReg(r61, &d1115)
		} else if d1112.Loc == LocImm {
			var r62 Reg
			if result.Loc == LocReg && result.Reg != d1113.Reg {
				r62 = result.Reg
				resultTarget1114 = true
			} else {
				r62 = ctx.AllocRegExcept(d1113.Reg)
			}
			_, xBits := d1112.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d1113.Reg, ctx.ScratchReg)
			d1115 = ctx.DeferBooleanFlags(r62, CondUnsignedAbove)
			ctx.BindReg(r62, &d1115)
		} else {
			var r63 Reg
			if result.Loc == LocReg && result.Reg != d1112.Reg && result.Reg != d1113.Reg {
				r63 = result.Reg
				resultTarget1114 = true
			} else {
				r63 = ctx.AllocRegExcept(d1112.Reg, d1113.Reg)
			}
			ctx.EmitCmpFloat64(d1113.Reg, d1112.Reg)
			d1115 = ctx.DeferBooleanFlags(r63, CondUnsignedAbove)
			ctx.BindReg(r63, &d1115)
		}
		ctx.FreeDesc(&d1112)
		ctx.FreeDesc(&d1113)
		ctx.SyncDesc(&d1115)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1115) {
			return d1115
		}
		ctx.EnsureDesc(&d1115)
		ctx.EmitMovToReg(result.Reg, d1115)
		result.Type = d1115.Type
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d1116 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d1116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d1116)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d1116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d1116)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d1116 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1116.Type = tagInt
			ctx.BindReg(d1116.Reg, &d1116)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d1117 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d1117 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d1117)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d1117 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d1117)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d1117 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d1117.Type = tagInt
			ctx.BindReg(d1117.Reg, &d1117)
		}
		ctx.EnsureDesc(&d1116)
		resultTarget1118 := false
		_ = resultTarget1118
		ctx.EnsureDesc(&d1117)
		ctx.EnsureDescsTogether(&d1116, &d1117)
		if d1116.Loc == LocImm && d1117.Loc == LocImm {
			d1119 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1116.Imm.Int() < d1117.Imm.Int())}
		} else if d1117.Loc == LocImm {
			r64 := ctx.AllocReg()
			if d1117.Imm.Int() >= -2147483648 && d1117.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d1116.Reg, int32(d1117.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1117.Imm.Int()))
				ctx.EmitCmpInt64(d1116.Reg, ctx.ScratchReg)
			}
			d1119 = ctx.DeferBooleanFlags(r64, CondSignedLess)
			ctx.BindReg(r64, &d1119)
		} else if d1116.Loc == LocImm {
			r65 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1116.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d1117.Reg)
			d1119 = ctx.DeferBooleanFlags(r65, CondSignedLess)
			ctx.BindReg(r65, &d1119)
		} else {
			r66 := ctx.AllocReg()
			ctx.EmitCmpInt64(d1116.Reg, d1117.Reg)
			d1119 = ctx.DeferBooleanFlags(r66, CondSignedLess)
			ctx.BindReg(r66, &d1119)
		}
		ctx.FreeDesc(&d1116)
		ctx.FreeDesc(&d1117)
		ctx.SyncDesc(&d1119)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1119) {
			return d1119
		}
		ctx.EnsureDesc(&d1119)
		ctx.EmitMovToReg(result.Reg, d1119)
		result.Type = d1119.Type
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
		ctx.EnsureDesc(&d4)
		if d4.Loc == LocImm {
			d1120 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x5))}
		} else {
			r67 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 5)
			d1120 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r67, Condition: CondEqual}
			ctx.BindReg(r67, &d1120)
		}
		d1121 = d1120
		ctx.SyncDesc(&d1121)
		if d1121.Loc != LocImm && d1121.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d1121.Loc == LocImm {
			if d1121.Imm.Bool() {
				return bbs[29].Render()
			}
			return bbs[31].Render()
		}
		ctx.EmitJump(d1121.Condition, lbl30)
		if bbs[31].Rendered {
			ctx.EmitJmp(lbl32)
		}
		ctx.FreeDesc(&d1120)
		ctx.FlushRegisterMoves()
		if !bbs[31].Rendered {
			snap1122 := d1
			snap1123 := d2
			snap1124 := d3
			snap1125 := d4
			snap1126 := d5
			snap1127 := d6
			snap1128 := d7
			snap1129 := d8
			snap1130 := d18
			snap1131 := d19
			snap1132 := d20
			snap1133 := d33
			snap1134 := d34
			snap1135 := d49
			snap1136 := d50
			snap1137 := d51
			snap1138 := d69
			snap1139 := d70
			snap1140 := d71
			snap1141 := d92
			snap1142 := d93
			snap1143 := d116
			snap1144 := d117
			snap1145 := d142
			snap1146 := d143
			snap1147 := d170
			snap1148 := d171
			snap1149 := d173
			snap1150 := d174
			snap1151 := d175
			snap1152 := d207
			snap1153 := d209
			snap1154 := d211
			snap1155 := d212
			snap1156 := d213
			snap1157 := d214
			snap1158 := d215
			snap1159 := d216
			snap1160 := d217
			snap1161 := d218
			snap1162 := d219
			snap1163 := d220
			snap1164 := d221
			snap1165 := d222
			snap1166 := d223
			snap1167 := d224
			snap1168 := d225
			snap1169 := d226
			snap1170 := d227
			snap1171 := d228
			snap1172 := d229
			snap1173 := d230
			snap1174 := d231
			snap1175 := d232
			snap1176 := d234
			snap1177 := d235
			snap1178 := d236
			snap1179 := d295
			snap1180 := d296
			snap1181 := d298
			snap1182 := d299
			snap1183 := d300
			snap1184 := d364
			snap1185 := d365
			snap1186 := d367
			snap1187 := d368
			snap1188 := d369
			snap1189 := d438
			snap1190 := d440
			snap1191 := d442
			snap1192 := d443
			snap1193 := d444
			snap1194 := d445
			snap1195 := d446
			snap1196 := d447
			snap1197 := d448
			snap1198 := d449
			snap1199 := d450
			snap1200 := d451
			snap1201 := d452
			snap1202 := d453
			snap1203 := d454
			snap1204 := d455
			snap1205 := d456
			snap1206 := d457
			snap1207 := d458
			snap1208 := d459
			snap1209 := d460
			snap1210 := d461
			snap1211 := d462
			snap1212 := d463
			snap1213 := d465
			snap1214 := d466
			snap1215 := d467
			snap1216 := d469
			snap1217 := d470
			snap1218 := d471
			snap1219 := d570
			snap1220 := d571
			snap1221 := d572
			snap1222 := d573
			snap1223 := d778
			snap1224 := d779
			snap1225 := d884
			snap1226 := d885
			snap1227 := d886
			snap1228 := d887
			snap1229 := d1104
			snap1230 := d1105
			snap1231 := d1106
			snap1232 := d1107
			snap1233 := d1108
			snap1234 := d1110
			snap1235 := d1111
			snap1236 := d1112
			snap1237 := d1113
			snap1238 := d1115
			snap1239 := d1116
			snap1240 := d1117
			snap1241 := d1119
			snap1242 := d1120
			snap1243 := d1121
			alloc1244 := ctx.SnapshotAllocState()
			bbs[31].Render()
			ctx.RestoreAllocState(alloc1244)
			d1 = snap1122
			d2 = snap1123
			d3 = snap1124
			d4 = snap1125
			d5 = snap1126
			d6 = snap1127
			d7 = snap1128
			d8 = snap1129
			d18 = snap1130
			d19 = snap1131
			d20 = snap1132
			d33 = snap1133
			d34 = snap1134
			d49 = snap1135
			d50 = snap1136
			d51 = snap1137
			d69 = snap1138
			d70 = snap1139
			d71 = snap1140
			d92 = snap1141
			d93 = snap1142
			d116 = snap1143
			d117 = snap1144
			d142 = snap1145
			d143 = snap1146
			d170 = snap1147
			d171 = snap1148
			d173 = snap1149
			d174 = snap1150
			d175 = snap1151
			d207 = snap1152
			d209 = snap1153
			d211 = snap1154
			d212 = snap1155
			d213 = snap1156
			d214 = snap1157
			d215 = snap1158
			d216 = snap1159
			d217 = snap1160
			d218 = snap1161
			d219 = snap1162
			d220 = snap1163
			d221 = snap1164
			d222 = snap1165
			d223 = snap1166
			d224 = snap1167
			d225 = snap1168
			d226 = snap1169
			d227 = snap1170
			d228 = snap1171
			d229 = snap1172
			d230 = snap1173
			d231 = snap1174
			d232 = snap1175
			d234 = snap1176
			d235 = snap1177
			d236 = snap1178
			d295 = snap1179
			d296 = snap1180
			d298 = snap1181
			d299 = snap1182
			d300 = snap1183
			d364 = snap1184
			d365 = snap1185
			d367 = snap1186
			d368 = snap1187
			d369 = snap1188
			d438 = snap1189
			d440 = snap1190
			d442 = snap1191
			d443 = snap1192
			d444 = snap1193
			d445 = snap1194
			d446 = snap1195
			d447 = snap1196
			d448 = snap1197
			d449 = snap1198
			d450 = snap1199
			d451 = snap1200
			d452 = snap1201
			d453 = snap1202
			d454 = snap1203
			d455 = snap1204
			d456 = snap1205
			d457 = snap1206
			d458 = snap1207
			d459 = snap1208
			d460 = snap1209
			d461 = snap1210
			d462 = snap1211
			d463 = snap1212
			d465 = snap1213
			d466 = snap1214
			d467 = snap1215
			d469 = snap1216
			d470 = snap1217
			d471 = snap1218
			d570 = snap1219
			d571 = snap1220
			d572 = snap1221
			d573 = snap1222
			d778 = snap1223
			d779 = snap1224
			d884 = snap1225
			d885 = snap1226
			d886 = snap1227
			d887 = snap1228
			d1104 = snap1229
			d1105 = snap1230
			d1106 = snap1231
			d1107 = snap1232
			d1108 = snap1233
			d1110 = snap1234
			d1111 = snap1235
			d1112 = snap1236
			d1113 = snap1237
			d1115 = snap1238
			d1116 = snap1239
			d1117 = snap1240
			d1119 = snap1241
			d1120 = snap1242
			d1121 = snap1243
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
		args[0] = JITPrepareScmerGoArg(ctx, args[0])
		args[1] = JITPrepareScmerGoArg(ctx, args[1])
		if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocRegTriple || d4.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocRegTriple || d6.Loc == LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&args[0])
		ctx.SyncDesc(&args[1])
		ctx.SyncDesc(&d4)
		ctx.SyncDesc(&d6)
		d1245 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d4, d6}, 1)
		d1245.NoHeapPointer = true
		ctx.EmitAndRegImm32(d1245.Reg, 1)
		d1245.Type = tagBool
		ctx.BindReg(d1245.Reg, &d1245)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.SyncDesc(&d1245)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1245) {
			return d1245
		}
		ctx.EnsureDesc(&d1245)
		ctx.EmitMovToReg(result.Reg, d1245)
		result.Type = d1245.Type
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

func init() {
	jitGoABI_0cbff6f55ee96f6d = GoFuncAddr(Scmer.String)
	jitGoABI_58434e8dfe39c263 = GoFuncAddr(ParseDateValue)
	jitGoABI_5bd1d0f41a4e4914 = GoFuncAddr(Scmer.Int)
	jitGoABI_973b6d6b19b704fd = GoFuncAddr((Scmer).DateNanosecond)
}
