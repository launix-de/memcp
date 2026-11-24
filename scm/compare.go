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
			if date, ok := ParseDateValue(b.String()); ok {
				return compareDates(a, date) == 0
			}
		}
		if tb == tagInt || tb == tagBool {
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
			if date, ok := ParseDateValue(a.String()); ok {
				return compareDates(date, b) == 0
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
			if date, ok := ParseDateValue(b.String()); ok {
				return NewBool(compareDates(a, date) == 0)
			}
		}
		if tb == tagInt || tb == tagBool {
			return NewBool(a.DateNanosecond() == 0 && a.Int() == b.Int())
		}
		return NewBool(a.Float() == b.Float())
	case tagInt:
		if tb == tagFloat {
			return NewBool(float64(a.Int()) == b.Float())
		}
		if tb == tagDate {
			return NewBool(b.DateNanosecond() == 0 && a.Int() == b.Int())
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
			if date, ok := ParseDateValue(a.String()); ok {
				return NewBool(compareDates(date, b) == 0)
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
		if tb == tagDate {
			return compareDates(a, b) < 0
		}
		if tb == tagString || tb == tagSymbol {
			if date, ok := ParseDateValue(b.String()); ok {
				return compareDates(a, date) < 0
			}
		}
		return a.Float() < b.Float()
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
			if date, ok := ParseDateValue(a.String()); ok {
				return compareDates(date, b) < 0
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
	var d153 JITValueDesc
	_ = d153
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
	var d167 JITValueDesc
	_ = d167
	var d168 JITValueDesc
	_ = d168
	var d170 JITValueDesc
	_ = d170
	var d171 JITValueDesc
	_ = d171
	var d172 JITValueDesc
	_ = d172
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
	var d179 JITValueDesc
	_ = d179
	var d180 JITValueDesc
	_ = d180
	var d181 JITValueDesc
	_ = d181
	var d233 JITValueDesc
	_ = d233
	var d234 JITValueDesc
	_ = d234
	var d236 JITValueDesc
	_ = d236
	var d237 JITValueDesc
	_ = d237
	var d238 JITValueDesc
	_ = d238
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
	var d362 JITValueDesc
	_ = d362
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
	var d371 JITValueDesc
	_ = d371
	var d372 JITValueDesc
	_ = d372
	var d373 JITValueDesc
	_ = d373
	var d374 JITValueDesc
	_ = d374
	var d375 JITValueDesc
	_ = d375
	var d376 JITValueDesc
	_ = d376
	var d377 JITValueDesc
	_ = d377
	var d378 JITValueDesc
	_ = d378
	var d379 JITValueDesc
	_ = d379
	var d381 JITValueDesc
	_ = d381
	var d382 JITValueDesc
	_ = d382
	var d383 JITValueDesc
	_ = d383
	var d384 JITValueDesc
	_ = d384
	var d385 JITValueDesc
	_ = d385
	var d386 JITValueDesc
	_ = d386
	var d387 JITValueDesc
	_ = d387
	var d388 JITValueDesc
	_ = d388
	var d390 JITValueDesc
	_ = d390
	var d391 JITValueDesc
	_ = d391
	var d392 JITValueDesc
	_ = d392
	var d394 JITValueDesc
	_ = d394
	var d395 JITValueDesc
	_ = d395
	var d396 JITValueDesc
	_ = d396
	var d488 JITValueDesc
	_ = d488
	var d489 JITValueDesc
	_ = d489
	var d491 JITValueDesc
	_ = d491
	var d492 JITValueDesc
	_ = d492
	var d493 JITValueDesc
	_ = d493
	var d494 JITValueDesc
	_ = d494
	var d496 JITValueDesc
	_ = d496
	var d497 JITValueDesc
	_ = d497
	var d498 JITValueDesc
	_ = d498
	var d500 JITValueDesc
	_ = d500
	var d501 JITValueDesc
	_ = d501
	var d502 JITValueDesc
	_ = d502
	var d606 JITValueDesc
	_ = d606
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
	var bbs [24]BBDescriptor
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
		ctx.SyncDesc(&d5)
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
		ctx.SyncDesc(&d15)
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
		ctx.SyncDesc(&d27)
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
		ctx.SyncDesc(&d42)
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
		ctx.SyncDesc(&d60)
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
			d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x10))}
		} else {
			r5 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 16)
			d79 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
			ctx.BindReg(r5, &d79)
		}
		d80 = d79
		ctx.SyncDesc(&d80)
		if d80.Loc != LocImm && d80.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d80.Loc == LocImm {
			if d80.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitJump(d80.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d79)
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
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
			bbs[12].Render()
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
		ctx.SyncDesc(&d102)
		if d102.Loc != LocImm && d102.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d102.Loc == LocImm {
			if d102.Imm.Bool() {
				return bbs[19].Render()
			}
			return bbs[20].Render()
		}
		ctx.EmitJump(d102.Condition, lbl20)
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
		}
		ctx.FreeDesc(&d101)
		ctx.FlushRegisterMoves()
		if !bbs[20].Rendered {
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
			bbs[20].Render()
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
		if !bbs[19].Rendered {
			return bbs[19].Render()
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
		ctx.SyncDesc(&d126)
		if d126.Loc != LocImm && d126.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d126.Loc == LocImm {
			if d126.Imm.Bool() {
				return bbs[9].Render()
			}
			return bbs[18].Render()
		}
		ctx.EmitJump(d126.Condition, lbl10)
		if bbs[18].Rendered {
			ctx.EmitJmp(lbl19)
		}
		ctx.FreeDesc(&d125)
		ctx.FlushRegisterMoves()
		if !bbs[18].Rendered {
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
			bbs[18].Render()
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
		ctx.EnsureDesc(&args[0])
		ctx.EnsureDesc(&args[1])
		d151 = args[0]
		_ = d151
		ctx.StabilizeDescForControlFlow(&d151)
		inlineHome152 := d151
		inlineHome152.ID = 0
		d153 = args[1]
		_ = d153
		ctx.StabilizeDescForControlFlow(&d153)
		inlineHome154 := d153
		inlineHome154.ID = 0
		lbl25 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl26 := ctx.ReserveLabel()
		_ = lbl26
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		lbl27 := ctx.ReserveLabel()
		_ = lbl27
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		lbl28 := ctx.ReserveLabel()
		_ = lbl28
		bbpos_1_3 := int32(-1)
		_ = bbpos_1_3
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_1_4 := int32(-1)
		_ = bbpos_1_4
		lbl30 := ctx.ReserveLabel()
		_ = lbl30
		bbpos_1_5 := int32(-1)
		_ = bbpos_1_5
		lbl31 := ctx.ReserveLabel()
		_ = lbl31
		bbpos_1_6 := int32(-1)
		_ = bbpos_1_6
		lbl32 := ctx.ReserveLabel()
		_ = lbl32
		bbpos_1_7 := int32(-1)
		_ = bbpos_1_7
		lbl33 := ctx.ReserveLabel()
		_ = lbl33
		bbpos_1_8 := int32(-1)
		_ = bbpos_1_8
		lbl34 := ctx.ReserveLabel()
		_ = lbl34
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl26)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d151.Loc != LocImm {
			ctx.EnsureDesc(&d151)
		}
		if d151.Loc == LocImm {
			d155 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d151.Imm.Int())}
		} else if d151.Type == tagInt && d151.Loc == LocRegPair {
			ctx.FreeReg(d151.Reg)
			d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d151.Reg2}
			ctx.BindReg(d151.Reg2, &d155)
			ctx.BindReg(d151.Reg2, &d155)
		} else if d151.Type == tagInt && d151.Loc == LocReg {
			d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d151.Reg}
			ctx.BindReg(d151.Reg, &d155)
			ctx.BindReg(d151.Reg, &d155)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d151)
			d155 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d155.Type = tagInt
			ctx.BindReg(d155.Reg, &d155)
		}
		ctx.ReclaimUntrackedRegs()
		if d153.Loc != LocImm {
			ctx.EnsureDesc(&d153)
		}
		if d153.Loc == LocImm {
			d156 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d153.Imm.Int())}
		} else if d153.Type == tagInt && d153.Loc == LocRegPair {
			ctx.FreeReg(d153.Reg)
			d156 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d153.Reg2}
			ctx.BindReg(d153.Reg2, &d156)
			ctx.BindReg(d153.Reg2, &d156)
		} else if d153.Type == tagInt && d153.Loc == LocReg {
			d156 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d153.Reg}
			ctx.BindReg(d153.Reg, &d156)
			ctx.BindReg(d153.Reg, &d156)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d153)
			d156 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d156.Type = tagInt
			ctx.BindReg(d156.Reg, &d156)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d155)
		ctx.EnsureDesc(&d156)
		ctx.EnsureDescsTogether(&d155, &d156)
		if d155.Loc == LocImm && d156.Loc == LocImm {
			d157 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d155.Imm.Int() < d156.Imm.Int())}
		} else if d156.Loc == LocImm {
			r8 := ctx.AllocReg()
			if d156.Imm.Int() >= -2147483648 && d156.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d155.Reg, int32(d156.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d156.Imm.Int()))
				ctx.EmitCmpInt64(d155.Reg, ctx.ScratchReg)
			}
			d157 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedLess}
			ctx.BindReg(r8, &d157)
		} else if d155.Loc == LocImm {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d155.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d156.Reg)
			d157 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
			ctx.BindReg(r9, &d157)
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitCmpInt64(d155.Reg, d156.Reg)
			d157 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
			ctx.BindReg(r10, &d157)
		}
		ctx.FreeDesc(&d155)
		ctx.FreeDesc(&d156)
		ctx.ReclaimUntrackedRegs()
		d158 = d157
		ctx.SyncDesc(&d158)
		if d158.Loc != LocImm && d158.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl35 := ctx.ReserveLabel()
		lbl36 := ctx.ReserveLabel()
		if d158.Loc == LocImm {
			if d158.Imm.Bool() {
				ctx.MarkLabel(lbl35)
				ctx.EmitJmp(lbl27)
			} else {
				ctx.MarkLabel(lbl36)
				ctx.EmitJmp(lbl28)
			}
		} else {
			ctx.EmitJump(d158.Condition, lbl35)
			ctx.EmitJmp(lbl36)
			ctx.FreeDesc(&d157)
			ctx.MarkLabel(lbl35)
			ctx.EmitJmp(lbl27)
			ctx.MarkLabel(lbl36)
			ctx.EmitJmp(lbl28)
		}
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl28)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d151.Loc != LocImm {
			ctx.EnsureDesc(&d151)
		}
		if d151.Loc == LocImm {
			d159 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d151.Imm.Int())}
		} else if d151.Type == tagInt && d151.Loc == LocRegPair {
			ctx.FreeReg(d151.Reg)
			d159 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d151.Reg2}
			ctx.BindReg(d151.Reg2, &d159)
			ctx.BindReg(d151.Reg2, &d159)
		} else if d151.Type == tagInt && d151.Loc == LocReg {
			d159 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d151.Reg}
			ctx.BindReg(d151.Reg, &d159)
			ctx.BindReg(d151.Reg, &d159)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d151)
			d159 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d159.Type = tagInt
			ctx.BindReg(d159.Reg, &d159)
		}
		ctx.ReclaimUntrackedRegs()
		if d153.Loc != LocImm {
			ctx.EnsureDesc(&d153)
		}
		if d153.Loc == LocImm {
			d160 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d153.Imm.Int())}
		} else if d153.Type == tagInt && d153.Loc == LocRegPair {
			ctx.FreeReg(d153.Reg)
			d160 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d153.Reg2}
			ctx.BindReg(d153.Reg2, &d160)
			ctx.BindReg(d153.Reg2, &d160)
		} else if d153.Type == tagInt && d153.Loc == LocReg {
			d160 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d153.Reg}
			ctx.BindReg(d153.Reg, &d160)
			ctx.BindReg(d153.Reg, &d160)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d153)
			d160 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d160.Type = tagInt
			ctx.BindReg(d160.Reg, &d160)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d159)
		ctx.EnsureDesc(&d160)
		ctx.EnsureDescsTogether(&d159, &d160)
		if d159.Loc == LocImm && d160.Loc == LocImm {
			d161 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d159.Imm.Int() > d160.Imm.Int())}
		} else if d160.Loc == LocImm {
			r11 := ctx.AllocReg()
			if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d159.Reg, int32(d160.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d160.Imm.Int()))
				ctx.EmitCmpInt64(d159.Reg, ctx.ScratchReg)
			}
			d161 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedGreater}
			ctx.BindReg(r11, &d161)
		} else if d159.Loc == LocImm {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d159.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d160.Reg)
			d161 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedGreater}
			ctx.BindReg(r12, &d161)
		} else {
			r13 := ctx.AllocReg()
			ctx.EmitCmpInt64(d159.Reg, d160.Reg)
			d161 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedGreater}
			ctx.BindReg(r13, &d161)
		}
		ctx.FreeDesc(&d159)
		ctx.FreeDesc(&d160)
		ctx.ReclaimUntrackedRegs()
		d162 = d161
		ctx.SyncDesc(&d162)
		if d162.Loc != LocImm && d162.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl37 := ctx.ReserveLabel()
		lbl38 := ctx.ReserveLabel()
		if d162.Loc == LocImm {
			if d162.Imm.Bool() {
				ctx.MarkLabel(lbl37)
				ctx.EmitJmp(lbl29)
			} else {
				ctx.MarkLabel(lbl38)
				ctx.EmitJmp(lbl30)
			}
		} else {
			ctx.EmitJump(d162.Condition, lbl37)
			ctx.EmitJmp(lbl38)
			ctx.FreeDesc(&d161)
			ctx.MarkLabel(lbl37)
			ctx.EmitJmp(lbl29)
			ctx.MarkLabel(lbl38)
			ctx.EmitJmp(lbl30)
		}
		bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl30)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d151 = JITPrepareScmerGoArg(ctx, d151)
		ctx.SyncDesc(&d151)
		d163 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d151}, 1)
		d163.NoHeapPointer = true
		ctx.BindReg(d163.Reg, &d163)
		ctx.ReclaimUntrackedRegs()
		d153 = JITPrepareScmerGoArg(ctx, d153)
		ctx.SyncDesc(&d153)
		d164 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d153}, 1)
		d164.NoHeapPointer = true
		ctx.BindReg(d164.Reg, &d164)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d163)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDescsTogether(&d163, &d164)
		if d163.Loc == LocImm && d164.Loc == LocImm {
			d165 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d163.Imm.Int() < d164.Imm.Int())}
		} else if d164.Loc == LocImm {
			r14 := ctx.AllocReg()
			if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d163.Reg, int32(d164.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d164.Imm.Int()))
				ctx.EmitCmpInt64(d163.Reg, ctx.ScratchReg)
			}
			d165 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
			ctx.BindReg(r14, &d165)
		} else if d163.Loc == LocImm {
			r15 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d163.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d164.Reg)
			d165 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
			ctx.BindReg(r15, &d165)
		} else {
			r16 := ctx.AllocReg()
			ctx.EmitCmpInt64(d163.Reg, d164.Reg)
			d165 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
			ctx.BindReg(r16, &d165)
		}
		ctx.FreeDesc(&d163)
		ctx.FreeDesc(&d164)
		ctx.ReclaimUntrackedRegs()
		d166 = d165
		ctx.SyncDesc(&d166)
		if d166.Loc != LocImm && d166.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl39 := ctx.ReserveLabel()
		lbl40 := ctx.ReserveLabel()
		if d166.Loc == LocImm {
			if d166.Imm.Bool() {
				ctx.MarkLabel(lbl39)
				ctx.EmitJmp(lbl31)
			} else {
				ctx.MarkLabel(lbl40)
				ctx.EmitJmp(lbl32)
			}
		} else {
			ctx.EmitJump(d166.Condition, lbl39)
			ctx.EmitJmp(lbl40)
			ctx.FreeDesc(&d165)
			ctx.MarkLabel(lbl39)
			ctx.EmitJmp(lbl31)
			ctx.MarkLabel(lbl40)
			ctx.EmitJmp(lbl32)
		}
		bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl32)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d151 = JITPrepareScmerGoArg(ctx, d151)
		ctx.SyncDesc(&d151)
		d167 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d151}, 1)
		d167.NoHeapPointer = true
		ctx.BindReg(d167.Reg, &d167)
		ctx.ReclaimUntrackedRegs()
		d153 = JITPrepareScmerGoArg(ctx, d153)
		ctx.SyncDesc(&d153)
		d168 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d153}, 1)
		d168.NoHeapPointer = true
		ctx.BindReg(d168.Reg, &d168)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d167)
		resultTarget169 := false
		_ = resultTarget169
		ctx.EnsureDesc(&d168)
		ctx.EnsureDescsTogether(&d167, &d168)
		if d167.Loc == LocImm && d168.Loc == LocImm {
			d170 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d167.Imm.Int() > d168.Imm.Int())}
		} else if d168.Loc == LocImm {
			r17 := ctx.AllocReg()
			if d168.Imm.Int() >= -2147483648 && d168.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d167.Reg, int32(d168.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d168.Imm.Int()))
				ctx.EmitCmpInt64(d167.Reg, ctx.ScratchReg)
			}
			d170 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedGreater}
			ctx.BindReg(r17, &d170)
		} else if d167.Loc == LocImm {
			r18 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d167.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d168.Reg)
			d170 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedGreater}
			ctx.BindReg(r18, &d170)
		} else {
			r19 := ctx.AllocReg()
			ctx.EmitCmpInt64(d167.Reg, d168.Reg)
			d170 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedGreater}
			ctx.BindReg(r19, &d170)
		}
		ctx.FreeDesc(&d167)
		ctx.FreeDesc(&d168)
		ctx.ReclaimUntrackedRegs()
		d171 = d170
		ctx.SyncDesc(&d171)
		if d171.Loc != LocImm && d171.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl41 := ctx.ReserveLabel()
		lbl42 := ctx.ReserveLabel()
		if d171.Loc == LocImm {
			if d171.Imm.Bool() {
				ctx.MarkLabel(lbl41)
				ctx.EmitJmp(lbl33)
			} else {
				ctx.MarkLabel(lbl42)
				ctx.EmitJmp(lbl34)
			}
		} else {
			ctx.EmitJump(d171.Condition, lbl41)
			ctx.EmitJmp(lbl42)
			ctx.FreeDesc(&d170)
			ctx.MarkLabel(lbl41)
			ctx.EmitJmp(lbl33)
			ctx.MarkLabel(lbl42)
			ctx.EmitJmp(lbl34)
		}
		bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl34)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r20 := ctx.AllocReg()
		d172 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
		ctx.EnsureDesc(&d172)
		if d172.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r20, d172)
		}
		ctx.EmitJmp(lbl25)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl27)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d173 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d173)
		if d173.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r20, d173)
		}
		ctx.EmitJmp(lbl25)
		bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d174 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d174)
		if d174.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r20, d174)
		}
		ctx.EmitJmp(lbl25)
		bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl31)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d175 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d175)
		if d175.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r20, d175)
		}
		ctx.EmitJmp(lbl25)
		bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl33)
		ctx.ResolveFixups()
		d151 = inlineHome152
		d153 = inlineHome154
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d176 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d176)
		if d176.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r20, d176)
		}
		ctx.EmitJmp(lbl25)
		ctx.MarkLabel(lbl25)
		d177 = JITValueDesc{Loc: LocReg, Reg: r20}
		ctx.BindReg(r20, &d177)
		ctx.BindReg(r20, &d177)
		ctx.EnsureDesc(&d177)
		resultTarget178 := false
		_ = resultTarget178
		if d177.Loc == LocImm {
			d179 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d177.Imm.Int() < 0)}
		} else {
			r21 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d177.Reg, 0)
			d179 = ctx.DeferBooleanFlags(r21, CondSignedLess)
			ctx.BindReg(r21, &d179)
		}
		ctx.FreeDesc(&d177)
		ctx.SyncDesc(&d179)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d179) {
			return d179
		}
		ctx.EnsureDesc(&d179)
		ctx.EmitMovToReg(result.Reg, d179)
		result.Type = d179.Type
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d180 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x1))}
		} else {
			r22 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 1)
			d180 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondEqual}
			ctx.BindReg(r22, &d180)
		}
		d181 = d180
		ctx.SyncDesc(&d181)
		if d181.Loc != LocImm && d181.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d181.Loc == LocImm {
			if d181.Imm.Bool() {
				return bbs[13].Render()
			}
			return bbs[15].Render()
		}
		ctx.EmitJump(d181.Condition, lbl14)
		if bbs[15].Rendered {
			ctx.EmitJmp(lbl16)
		}
		ctx.FreeDesc(&d180)
		ctx.FlushRegisterMoves()
		if !bbs[15].Rendered {
			snap182 := d0
			snap183 := d1
			snap184 := d2
			snap185 := d3
			snap186 := d4
			snap187 := d5
			snap188 := d13
			snap189 := d14
			snap190 := d15
			snap191 := d26
			snap192 := d27
			snap193 := d40
			snap194 := d41
			snap195 := d42
			snap196 := d58
			snap197 := d59
			snap198 := d60
			snap199 := d79
			snap200 := d80
			snap201 := d101
			snap202 := d102
			snap203 := d125
			snap204 := d126
			snap205 := d151
			snap206 := d153
			snap207 := d155
			snap208 := d156
			snap209 := d157
			snap210 := d158
			snap211 := d159
			snap212 := d160
			snap213 := d161
			snap214 := d162
			snap215 := d163
			snap216 := d164
			snap217 := d165
			snap218 := d166
			snap219 := d167
			snap220 := d168
			snap221 := d170
			snap222 := d171
			snap223 := d172
			snap224 := d173
			snap225 := d174
			snap226 := d175
			snap227 := d176
			snap228 := d177
			snap229 := d179
			snap230 := d180
			snap231 := d181
			alloc232 := ctx.SnapshotAllocState()
			bbs[15].Render()
			ctx.RestoreAllocState(alloc232)
			d0 = snap182
			d1 = snap183
			d2 = snap184
			d3 = snap185
			d4 = snap186
			d5 = snap187
			d13 = snap188
			d14 = snap189
			d15 = snap190
			d26 = snap191
			d27 = snap192
			d40 = snap193
			d41 = snap194
			d42 = snap195
			d58 = snap196
			d59 = snap197
			d60 = snap198
			d79 = snap199
			d80 = snap200
			d101 = snap201
			d102 = snap202
			d125 = snap203
			d126 = snap204
			d151 = snap205
			d153 = snap206
			d155 = snap207
			d156 = snap208
			d157 = snap209
			d158 = snap210
			d159 = snap211
			d160 = snap212
			d161 = snap213
			d162 = snap214
			d163 = snap215
			d164 = snap216
			d165 = snap217
			d166 = snap218
			d167 = snap219
			d168 = snap220
			d170 = snap221
			d171 = snap222
			d172 = snap223
			d173 = snap224
			d174 = snap225
			d175 = snap226
			d176 = snap227
			d177 = snap228
			d179 = snap229
			d180 = snap230
			d181 = snap231
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
		ctx.ReclaimUntrackedRegs()
		d234 = args[1]
		ctx.SyncDesc(&d234)
		if d234.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d234.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d234.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d234 = tmpScalar
		}
		d234 = JITPrepareScmerGoArg(ctx, d234)
		if d234.Loc != LocRegPair && d234.Loc != LocStackPair && d234.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d233 = ctx.EmitGoCallScalar(jitGoABI_0cbff6f55ee96f6d, []JITValueDesc{d234}, 2)
		ctx.EnsureDesc(&d233)
		if d233.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d233.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d233.Imm)
			ptrWord, _ := d233.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d233.Imm.String())))
			d233 = tmpPair
		} else if d233.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d233.Type, Reg: ctx.AllocRegExcept(d233.Reg), Reg2: ctx.AllocRegExcept(d233.Reg)}
			switch d233.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d233)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d233)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d233)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d233)
			d233 = tmpPair
		}
		if d233.Loc != LocRegPair && d233.Loc != LocStackPair && d233.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateValue arg0)")
		}
		ctx.SyncDesc(&d233)
		callResults235 := JITEmitGoCallResults(ctx, jitGoABI_58434e8dfe39c263, []JITValueDesc{d233})
		d236 = callResults235[0]
		_ = d236
		d237 = callResults235[1]
		_ = d237
		ctx.StabilizeDescForControlFlow(&d236)
		d238 = d237
		ctx.EnsureDesc(&d238)
		if d238.Loc != LocImm && d238.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d238.Loc == LocImm {
			if d238.Imm.Bool() {
				return bbs[16].Render()
			}
			return bbs[14].Render()
		}
		ctx.EmitCmpRegImm32(d238.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl17)
		if bbs[14].Rendered {
			ctx.EmitJmp(lbl15)
		}
		ctx.FlushRegisterMoves()
		if !bbs[14].Rendered {
			snap239 := d0
			snap240 := d1
			snap241 := d2
			snap242 := d3
			snap243 := d4
			snap244 := d5
			snap245 := d13
			snap246 := d14
			snap247 := d15
			snap248 := d26
			snap249 := d27
			snap250 := d40
			snap251 := d41
			snap252 := d42
			snap253 := d58
			snap254 := d59
			snap255 := d60
			snap256 := d79
			snap257 := d80
			snap258 := d101
			snap259 := d102
			snap260 := d125
			snap261 := d126
			snap262 := d151
			snap263 := d153
			snap264 := d155
			snap265 := d156
			snap266 := d157
			snap267 := d158
			snap268 := d159
			snap269 := d160
			snap270 := d161
			snap271 := d162
			snap272 := d163
			snap273 := d164
			snap274 := d165
			snap275 := d166
			snap276 := d167
			snap277 := d168
			snap278 := d170
			snap279 := d171
			snap280 := d172
			snap281 := d173
			snap282 := d174
			snap283 := d175
			snap284 := d176
			snap285 := d177
			snap286 := d179
			snap287 := d180
			snap288 := d181
			snap289 := d233
			snap290 := d234
			snap291 := d236
			snap292 := d237
			snap293 := d238
			alloc294 := ctx.SnapshotAllocState()
			bbs[14].Render()
			ctx.RestoreAllocState(alloc294)
			d0 = snap239
			d1 = snap240
			d2 = snap241
			d3 = snap242
			d4 = snap243
			d5 = snap244
			d13 = snap245
			d14 = snap246
			d15 = snap247
			d26 = snap248
			d27 = snap249
			d40 = snap250
			d41 = snap251
			d42 = snap252
			d58 = snap253
			d59 = snap254
			d60 = snap255
			d79 = snap256
			d80 = snap257
			d101 = snap258
			d102 = snap259
			d125 = snap260
			d126 = snap261
			d151 = snap262
			d153 = snap263
			d155 = snap264
			d156 = snap265
			d157 = snap266
			d158 = snap267
			d159 = snap268
			d160 = snap269
			d161 = snap270
			d162 = snap271
			d163 = snap272
			d164 = snap273
			d165 = snap274
			d166 = snap275
			d167 = snap276
			d168 = snap277
			d170 = snap278
			d171 = snap279
			d172 = snap280
			d173 = snap281
			d174 = snap282
			d175 = snap283
			d176 = snap284
			d177 = snap285
			d179 = snap286
			d180 = snap287
			d181 = snap288
			d233 = snap289
			d234 = snap290
			d236 = snap291
			d237 = snap292
			d238 = snap293
		}
		if !bbs[16].Rendered {
			return bbs[16].Render()
		}
		return result
		ctx.FreeDesc(&d237)
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
		d295 = ctx.EmitFloatDesc(args[0])
		d296 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d295)
		resultTarget297 := false
		_ = resultTarget297
		ctx.EnsureDesc(&d296)
		ctx.EnsureDescsTogether(&d295, &d296)
		if d295.Loc == LocImm && d296.Loc == LocImm {
			d298 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d295.Imm.Float() < d296.Imm.Float())}
		} else if d296.Loc == LocImm {
			var r23 Reg
			if result.Loc == LocReg && result.Reg != d295.Reg {
				r23 = result.Reg
				resultTarget297 = true
			} else {
				r23 = ctx.AllocRegExcept(d295.Reg)
			}
			_, yBits := d296.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d295.Reg)
			d298 = ctx.DeferBooleanFlags(r23, CondUnsignedAbove)
			ctx.BindReg(r23, &d298)
		} else if d295.Loc == LocImm {
			var r24 Reg
			if result.Loc == LocReg && result.Reg != d296.Reg {
				r24 = result.Reg
				resultTarget297 = true
			} else {
				r24 = ctx.AllocRegExcept(d296.Reg)
			}
			_, xBits := d295.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d296.Reg, ctx.ScratchReg)
			d298 = ctx.DeferBooleanFlags(r24, CondUnsignedAbove)
			ctx.BindReg(r24, &d298)
		} else {
			var r25 Reg
			if result.Loc == LocReg && result.Reg != d295.Reg && result.Reg != d296.Reg {
				r25 = result.Reg
				resultTarget297 = true
			} else {
				r25 = ctx.AllocRegExcept(d295.Reg, d296.Reg)
			}
			ctx.EmitCmpFloat64(d296.Reg, d295.Reg)
			d298 = ctx.DeferBooleanFlags(r25, CondUnsignedAbove)
			ctx.BindReg(r25, &d298)
		}
		ctx.FreeDesc(&d295)
		ctx.FreeDesc(&d296)
		ctx.SyncDesc(&d298)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d298) {
			return d298
		}
		ctx.EnsureDesc(&d298)
		ctx.EmitMovToReg(result.Reg, d298)
		result.Type = d298.Type
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
		ctx.EnsureDesc(&d3)
		if d3.Loc == LocImm {
			d299 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x2))}
		} else {
			r26 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 2)
			d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondEqual}
			ctx.BindReg(r26, &d299)
		}
		d300 = d299
		ctx.SyncDesc(&d300)
		if d300.Loc != LocImm && d300.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d300.Loc == LocImm {
			if d300.Imm.Bool() {
				return bbs[13].Render()
			}
			return bbs[14].Render()
		}
		ctx.EmitJump(d300.Condition, lbl14)
		if bbs[14].Rendered {
			ctx.EmitJmp(lbl15)
		}
		ctx.FreeDesc(&d299)
		ctx.FlushRegisterMoves()
		if !bbs[14].Rendered {
			snap301 := d0
			snap302 := d1
			snap303 := d2
			snap304 := d3
			snap305 := d4
			snap306 := d5
			snap307 := d13
			snap308 := d14
			snap309 := d15
			snap310 := d26
			snap311 := d27
			snap312 := d40
			snap313 := d41
			snap314 := d42
			snap315 := d58
			snap316 := d59
			snap317 := d60
			snap318 := d79
			snap319 := d80
			snap320 := d101
			snap321 := d102
			snap322 := d125
			snap323 := d126
			snap324 := d151
			snap325 := d153
			snap326 := d155
			snap327 := d156
			snap328 := d157
			snap329 := d158
			snap330 := d159
			snap331 := d160
			snap332 := d161
			snap333 := d162
			snap334 := d163
			snap335 := d164
			snap336 := d165
			snap337 := d166
			snap338 := d167
			snap339 := d168
			snap340 := d170
			snap341 := d171
			snap342 := d172
			snap343 := d173
			snap344 := d174
			snap345 := d175
			snap346 := d176
			snap347 := d177
			snap348 := d179
			snap349 := d180
			snap350 := d181
			snap351 := d233
			snap352 := d234
			snap353 := d236
			snap354 := d237
			snap355 := d238
			snap356 := d295
			snap357 := d296
			snap358 := d298
			snap359 := d299
			snap360 := d300
			alloc361 := ctx.SnapshotAllocState()
			bbs[14].Render()
			ctx.RestoreAllocState(alloc361)
			d0 = snap301
			d1 = snap302
			d2 = snap303
			d3 = snap304
			d4 = snap305
			d5 = snap306
			d13 = snap307
			d14 = snap308
			d15 = snap309
			d26 = snap310
			d27 = snap311
			d40 = snap312
			d41 = snap313
			d42 = snap314
			d58 = snap315
			d59 = snap316
			d60 = snap317
			d79 = snap318
			d80 = snap319
			d101 = snap320
			d102 = snap321
			d125 = snap322
			d126 = snap323
			d151 = snap324
			d153 = snap325
			d155 = snap326
			d156 = snap327
			d157 = snap328
			d158 = snap329
			d159 = snap330
			d160 = snap331
			d161 = snap332
			d162 = snap333
			d163 = snap334
			d164 = snap335
			d165 = snap336
			d166 = snap337
			d167 = snap338
			d168 = snap339
			d170 = snap340
			d171 = snap341
			d172 = snap342
			d173 = snap343
			d174 = snap344
			d175 = snap345
			d176 = snap346
			d177 = snap347
			d179 = snap348
			d180 = snap349
			d181 = snap350
			d233 = snap351
			d234 = snap352
			d236 = snap353
			d237 = snap354
			d238 = snap355
			d295 = snap356
			d296 = snap357
			d298 = snap358
			d299 = snap359
			d300 = snap360
		}
		if !bbs[13].Rendered {
			return bbs[13].Render()
		}
		return result
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
		ctx.EnsureDesc(&args[0])
		ctx.EnsureDesc(&d236)
		d362 = args[0]
		_ = d362
		ctx.StabilizeDescForControlFlow(&d362)
		inlineHome363 := d362
		inlineHome363.ID = 0
		d364 = d236
		_ = d364
		ctx.StabilizeDescForControlFlow(&d364)
		inlineHome365 := d364
		inlineHome365.ID = 0
		lbl43 := ctx.ReserveLabel()
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl44 := ctx.ReserveLabel()
		_ = lbl44
		bbpos_2_1 := int32(-1)
		_ = bbpos_2_1
		lbl45 := ctx.ReserveLabel()
		_ = lbl45
		bbpos_2_2 := int32(-1)
		_ = bbpos_2_2
		lbl46 := ctx.ReserveLabel()
		_ = lbl46
		bbpos_2_3 := int32(-1)
		_ = bbpos_2_3
		lbl47 := ctx.ReserveLabel()
		_ = lbl47
		bbpos_2_4 := int32(-1)
		_ = bbpos_2_4
		lbl48 := ctx.ReserveLabel()
		_ = lbl48
		bbpos_2_5 := int32(-1)
		_ = bbpos_2_5
		lbl49 := ctx.ReserveLabel()
		_ = lbl49
		bbpos_2_6 := int32(-1)
		_ = bbpos_2_6
		lbl50 := ctx.ReserveLabel()
		_ = lbl50
		bbpos_2_7 := int32(-1)
		_ = bbpos_2_7
		lbl51 := ctx.ReserveLabel()
		_ = lbl51
		bbpos_2_8 := int32(-1)
		_ = bbpos_2_8
		lbl52 := ctx.ReserveLabel()
		_ = lbl52
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl44)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d362.Loc != LocImm {
			ctx.EnsureDesc(&d362)
		}
		if d362.Loc == LocImm {
			d366 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d362.Imm.Int())}
		} else if d362.Type == tagInt && d362.Loc == LocRegPair {
			ctx.FreeReg(d362.Reg)
			d366 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d362.Reg2}
			ctx.BindReg(d362.Reg2, &d366)
			ctx.BindReg(d362.Reg2, &d366)
		} else if d362.Type == tagInt && d362.Loc == LocReg {
			d366 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d362.Reg}
			ctx.BindReg(d362.Reg, &d366)
			ctx.BindReg(d362.Reg, &d366)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d362)
			d366 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d366.Type = tagInt
			ctx.BindReg(d366.Reg, &d366)
		}
		ctx.ReclaimUntrackedRegs()
		if d364.Loc != LocImm {
			ctx.EnsureDesc(&d364)
		}
		if d364.Loc == LocImm {
			d367 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d364.Imm.Int())}
		} else if d364.Type == tagInt && d364.Loc == LocRegPair {
			ctx.FreeReg(d364.Reg)
			d367 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d364.Reg2}
			ctx.BindReg(d364.Reg2, &d367)
			ctx.BindReg(d364.Reg2, &d367)
		} else if d364.Type == tagInt && d364.Loc == LocReg {
			d367 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d364.Reg}
			ctx.BindReg(d364.Reg, &d367)
			ctx.BindReg(d364.Reg, &d367)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d364)
			d367 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d367.Type = tagInt
			ctx.BindReg(d367.Reg, &d367)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d366)
		ctx.EnsureDesc(&d367)
		ctx.EnsureDescsTogether(&d366, &d367)
		if d366.Loc == LocImm && d367.Loc == LocImm {
			d368 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d366.Imm.Int() < d367.Imm.Int())}
		} else if d367.Loc == LocImm {
			r27 := ctx.AllocReg()
			if d367.Imm.Int() >= -2147483648 && d367.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d366.Reg, int32(d367.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d367.Imm.Int()))
				ctx.EmitCmpInt64(d366.Reg, ctx.ScratchReg)
			}
			d368 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
			ctx.BindReg(r27, &d368)
		} else if d366.Loc == LocImm {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d366.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d367.Reg)
			d368 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
			ctx.BindReg(r28, &d368)
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitCmpInt64(d366.Reg, d367.Reg)
			d368 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondSignedLess}
			ctx.BindReg(r29, &d368)
		}
		ctx.FreeDesc(&d366)
		ctx.FreeDesc(&d367)
		ctx.ReclaimUntrackedRegs()
		d369 = d368
		ctx.SyncDesc(&d369)
		if d369.Loc != LocImm && d369.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl53 := ctx.ReserveLabel()
		lbl54 := ctx.ReserveLabel()
		if d369.Loc == LocImm {
			if d369.Imm.Bool() {
				ctx.MarkLabel(lbl53)
				ctx.EmitJmp(lbl45)
			} else {
				ctx.MarkLabel(lbl54)
				ctx.EmitJmp(lbl46)
			}
		} else {
			ctx.EmitJump(d369.Condition, lbl53)
			ctx.EmitJmp(lbl54)
			ctx.FreeDesc(&d368)
			ctx.MarkLabel(lbl53)
			ctx.EmitJmp(lbl45)
			ctx.MarkLabel(lbl54)
			ctx.EmitJmp(lbl46)
		}
		bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl46)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if d362.Loc != LocImm {
			ctx.EnsureDesc(&d362)
		}
		if d362.Loc == LocImm {
			d370 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d362.Imm.Int())}
		} else if d362.Type == tagInt && d362.Loc == LocRegPair {
			ctx.FreeReg(d362.Reg)
			d370 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d362.Reg2}
			ctx.BindReg(d362.Reg2, &d370)
			ctx.BindReg(d362.Reg2, &d370)
		} else if d362.Type == tagInt && d362.Loc == LocReg {
			d370 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d362.Reg}
			ctx.BindReg(d362.Reg, &d370)
			ctx.BindReg(d362.Reg, &d370)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d362)
			d370 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d370.Type = tagInt
			ctx.BindReg(d370.Reg, &d370)
		}
		ctx.ReclaimUntrackedRegs()
		if d364.Loc != LocImm {
			ctx.EnsureDesc(&d364)
		}
		if d364.Loc == LocImm {
			d371 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d364.Imm.Int())}
		} else if d364.Type == tagInt && d364.Loc == LocRegPair {
			ctx.FreeReg(d364.Reg)
			d371 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d364.Reg2}
			ctx.BindReg(d364.Reg2, &d371)
			ctx.BindReg(d364.Reg2, &d371)
		} else if d364.Type == tagInt && d364.Loc == LocReg {
			d371 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d364.Reg}
			ctx.BindReg(d364.Reg, &d371)
			ctx.BindReg(d364.Reg, &d371)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, d364)
			d371 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d371.Type = tagInt
			ctx.BindReg(d371.Reg, &d371)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d370)
		ctx.EnsureDesc(&d371)
		ctx.EnsureDescsTogether(&d370, &d371)
		if d370.Loc == LocImm && d371.Loc == LocImm {
			d372 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d370.Imm.Int() > d371.Imm.Int())}
		} else if d371.Loc == LocImm {
			r30 := ctx.AllocReg()
			if d371.Imm.Int() >= -2147483648 && d371.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d370.Reg, int32(d371.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d371.Imm.Int()))
				ctx.EmitCmpInt64(d370.Reg, ctx.ScratchReg)
			}
			d372 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondSignedGreater}
			ctx.BindReg(r30, &d372)
		} else if d370.Loc == LocImm {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d370.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d371.Reg)
			d372 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondSignedGreater}
			ctx.BindReg(r31, &d372)
		} else {
			r32 := ctx.AllocReg()
			ctx.EmitCmpInt64(d370.Reg, d371.Reg)
			d372 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedGreater}
			ctx.BindReg(r32, &d372)
		}
		ctx.FreeDesc(&d370)
		ctx.FreeDesc(&d371)
		ctx.ReclaimUntrackedRegs()
		d373 = d372
		ctx.SyncDesc(&d373)
		if d373.Loc != LocImm && d373.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl55 := ctx.ReserveLabel()
		lbl56 := ctx.ReserveLabel()
		if d373.Loc == LocImm {
			if d373.Imm.Bool() {
				ctx.MarkLabel(lbl55)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl56)
				ctx.EmitJmp(lbl48)
			}
		} else {
			ctx.EmitJump(d373.Condition, lbl55)
			ctx.EmitJmp(lbl56)
			ctx.FreeDesc(&d372)
			ctx.MarkLabel(lbl55)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl56)
			ctx.EmitJmp(lbl48)
		}
		bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl48)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d362 = JITPrepareScmerGoArg(ctx, d362)
		ctx.SyncDesc(&d362)
		d374 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d362}, 1)
		d374.NoHeapPointer = true
		ctx.BindReg(d374.Reg, &d374)
		ctx.ReclaimUntrackedRegs()
		d364 = JITPrepareScmerGoArg(ctx, d364)
		ctx.SyncDesc(&d364)
		d375 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d364}, 1)
		d375.NoHeapPointer = true
		ctx.BindReg(d375.Reg, &d375)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d374)
		ctx.EnsureDesc(&d375)
		ctx.EnsureDescsTogether(&d374, &d375)
		if d374.Loc == LocImm && d375.Loc == LocImm {
			d376 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d374.Imm.Int() < d375.Imm.Int())}
		} else if d375.Loc == LocImm {
			r33 := ctx.AllocReg()
			if d375.Imm.Int() >= -2147483648 && d375.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d374.Reg, int32(d375.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d375.Imm.Int()))
				ctx.EmitCmpInt64(d374.Reg, ctx.ScratchReg)
			}
			d376 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
			ctx.BindReg(r33, &d376)
		} else if d374.Loc == LocImm {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d374.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d375.Reg)
			d376 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r34, Condition: CondSignedLess}
			ctx.BindReg(r34, &d376)
		} else {
			r35 := ctx.AllocReg()
			ctx.EmitCmpInt64(d374.Reg, d375.Reg)
			d376 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedLess}
			ctx.BindReg(r35, &d376)
		}
		ctx.FreeDesc(&d374)
		ctx.FreeDesc(&d375)
		ctx.ReclaimUntrackedRegs()
		d377 = d376
		ctx.SyncDesc(&d377)
		if d377.Loc != LocImm && d377.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl57 := ctx.ReserveLabel()
		lbl58 := ctx.ReserveLabel()
		if d377.Loc == LocImm {
			if d377.Imm.Bool() {
				ctx.MarkLabel(lbl57)
				ctx.EmitJmp(lbl49)
			} else {
				ctx.MarkLabel(lbl58)
				ctx.EmitJmp(lbl50)
			}
		} else {
			ctx.EmitJump(d377.Condition, lbl57)
			ctx.EmitJmp(lbl58)
			ctx.FreeDesc(&d376)
			ctx.MarkLabel(lbl57)
			ctx.EmitJmp(lbl49)
			ctx.MarkLabel(lbl58)
			ctx.EmitJmp(lbl50)
		}
		bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl50)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d362 = JITPrepareScmerGoArg(ctx, d362)
		ctx.SyncDesc(&d362)
		d378 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d362}, 1)
		d378.NoHeapPointer = true
		ctx.BindReg(d378.Reg, &d378)
		ctx.ReclaimUntrackedRegs()
		d364 = JITPrepareScmerGoArg(ctx, d364)
		ctx.SyncDesc(&d364)
		d379 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{d364}, 1)
		d379.NoHeapPointer = true
		ctx.BindReg(d379.Reg, &d379)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d378)
		resultTarget380 := false
		_ = resultTarget380
		ctx.EnsureDesc(&d379)
		ctx.EnsureDescsTogether(&d378, &d379)
		if d378.Loc == LocImm && d379.Loc == LocImm {
			d381 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d378.Imm.Int() > d379.Imm.Int())}
		} else if d379.Loc == LocImm {
			r36 := ctx.AllocReg()
			if d379.Imm.Int() >= -2147483648 && d379.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d378.Reg, int32(d379.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d379.Imm.Int()))
				ctx.EmitCmpInt64(d378.Reg, ctx.ScratchReg)
			}
			d381 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedGreater}
			ctx.BindReg(r36, &d381)
		} else if d378.Loc == LocImm {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d378.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d379.Reg)
			d381 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedGreater}
			ctx.BindReg(r37, &d381)
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitCmpInt64(d378.Reg, d379.Reg)
			d381 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r38, Condition: CondSignedGreater}
			ctx.BindReg(r38, &d381)
		}
		ctx.FreeDesc(&d378)
		ctx.FreeDesc(&d379)
		ctx.ReclaimUntrackedRegs()
		d382 = d381
		ctx.SyncDesc(&d382)
		if d382.Loc != LocImm && d382.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		lbl59 := ctx.ReserveLabel()
		lbl60 := ctx.ReserveLabel()
		if d382.Loc == LocImm {
			if d382.Imm.Bool() {
				ctx.MarkLabel(lbl59)
				ctx.EmitJmp(lbl51)
			} else {
				ctx.MarkLabel(lbl60)
				ctx.EmitJmp(lbl52)
			}
		} else {
			ctx.EmitJump(d382.Condition, lbl59)
			ctx.EmitJmp(lbl60)
			ctx.FreeDesc(&d381)
			ctx.MarkLabel(lbl59)
			ctx.EmitJmp(lbl51)
			ctx.MarkLabel(lbl60)
			ctx.EmitJmp(lbl52)
		}
		bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl52)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r39 := ctx.AllocReg()
		d383 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
		ctx.EnsureDesc(&d383)
		if d383.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r39, d383)
		}
		ctx.EmitJmp(lbl43)
		bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl45)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d384 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d384)
		if d384.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r39, d384)
		}
		ctx.EmitJmp(lbl43)
		bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl47)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d385 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d385)
		if d385.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r39, d385)
		}
		ctx.EmitJmp(lbl43)
		bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl49)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d386 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
		ctx.EnsureDesc(&d386)
		if d386.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r39, d386)
		}
		ctx.EmitJmp(lbl43)
		bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl51)
		ctx.ResolveFixups()
		d362 = inlineHome363
		d364 = inlineHome365
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d387 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
		ctx.EnsureDesc(&d387)
		if d387.Loc == LocRegPair {
			panic("jit: scalar inline return has LocRegPair")
		} else {
			ctx.EmitMovToReg(r39, d387)
		}
		ctx.EmitJmp(lbl43)
		ctx.MarkLabel(lbl43)
		d388 = JITValueDesc{Loc: LocReg, Reg: r39}
		ctx.BindReg(r39, &d388)
		ctx.BindReg(r39, &d388)
		ctx.EnsureDesc(&d388)
		resultTarget389 := false
		_ = resultTarget389
		if d388.Loc == LocImm {
			d390 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d388.Imm.Int() < 0)}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d388.Reg, 0)
			d390 = ctx.DeferBooleanFlags(r40, CondSignedLess)
			ctx.BindReg(r40, &d390)
		}
		ctx.FreeDesc(&d388)
		ctx.SyncDesc(&d390)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d390) {
			return d390
		}
		ctx.EnsureDesc(&d390)
		ctx.EmitMovToReg(result.Reg, d390)
		result.Type = d390.Type
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
		ctx.ReclaimUntrackedRegs()
		d391 = ctx.EmitFloatDesc(args[0])
		d392 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d391)
		resultTarget393 := false
		_ = resultTarget393
		ctx.EnsureDesc(&d392)
		ctx.EnsureDescsTogether(&d391, &d392)
		if d391.Loc == LocImm && d392.Loc == LocImm {
			d394 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d391.Imm.Float() < d392.Imm.Float())}
		} else if d392.Loc == LocImm {
			var r41 Reg
			if result.Loc == LocReg && result.Reg != d391.Reg {
				r41 = result.Reg
				resultTarget393 = true
			} else {
				r41 = ctx.AllocRegExcept(d391.Reg)
			}
			_, yBits := d392.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d391.Reg)
			d394 = ctx.DeferBooleanFlags(r41, CondUnsignedAbove)
			ctx.BindReg(r41, &d394)
		} else if d391.Loc == LocImm {
			var r42 Reg
			if result.Loc == LocReg && result.Reg != d392.Reg {
				r42 = result.Reg
				resultTarget393 = true
			} else {
				r42 = ctx.AllocRegExcept(d392.Reg)
			}
			_, xBits := d391.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d392.Reg, ctx.ScratchReg)
			d394 = ctx.DeferBooleanFlags(r42, CondUnsignedAbove)
			ctx.BindReg(r42, &d394)
		} else {
			var r43 Reg
			if result.Loc == LocReg && result.Reg != d391.Reg && result.Reg != d392.Reg {
				r43 = result.Reg
				resultTarget393 = true
			} else {
				r43 = ctx.AllocRegExcept(d391.Reg, d392.Reg)
			}
			ctx.EmitCmpFloat64(d392.Reg, d391.Reg)
			d394 = ctx.DeferBooleanFlags(r43, CondUnsignedAbove)
			ctx.BindReg(r43, &d394)
		}
		ctx.FreeDesc(&d391)
		ctx.FreeDesc(&d392)
		ctx.SyncDesc(&d394)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d394) {
			return d394
		}
		ctx.EnsureDesc(&d394)
		ctx.EmitMovToReg(result.Reg, d394)
		result.Type = d394.Type
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
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d395 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x3))}
		} else {
			r44 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 3)
			d395 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r44, Condition: CondEqual}
			ctx.BindReg(r44, &d395)
		}
		d396 = d395
		ctx.SyncDesc(&d396)
		if d396.Loc != LocImm && d396.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d396.Loc == LocImm {
			if d396.Imm.Bool() {
				return bbs[17].Render()
			}
			return bbs[22].Render()
		}
		ctx.EmitJump(d396.Condition, lbl18)
		if bbs[22].Rendered {
			ctx.EmitJmp(lbl23)
		}
		ctx.FreeDesc(&d395)
		ctx.FlushRegisterMoves()
		if !bbs[22].Rendered {
			snap397 := d0
			snap398 := d1
			snap399 := d2
			snap400 := d3
			snap401 := d4
			snap402 := d5
			snap403 := d13
			snap404 := d14
			snap405 := d15
			snap406 := d26
			snap407 := d27
			snap408 := d40
			snap409 := d41
			snap410 := d42
			snap411 := d58
			snap412 := d59
			snap413 := d60
			snap414 := d79
			snap415 := d80
			snap416 := d101
			snap417 := d102
			snap418 := d125
			snap419 := d126
			snap420 := d151
			snap421 := d153
			snap422 := d155
			snap423 := d156
			snap424 := d157
			snap425 := d158
			snap426 := d159
			snap427 := d160
			snap428 := d161
			snap429 := d162
			snap430 := d163
			snap431 := d164
			snap432 := d165
			snap433 := d166
			snap434 := d167
			snap435 := d168
			snap436 := d170
			snap437 := d171
			snap438 := d172
			snap439 := d173
			snap440 := d174
			snap441 := d175
			snap442 := d176
			snap443 := d177
			snap444 := d179
			snap445 := d180
			snap446 := d181
			snap447 := d233
			snap448 := d234
			snap449 := d236
			snap450 := d237
			snap451 := d238
			snap452 := d295
			snap453 := d296
			snap454 := d298
			snap455 := d299
			snap456 := d300
			snap457 := d362
			snap458 := d364
			snap459 := d366
			snap460 := d367
			snap461 := d368
			snap462 := d369
			snap463 := d370
			snap464 := d371
			snap465 := d372
			snap466 := d373
			snap467 := d374
			snap468 := d375
			snap469 := d376
			snap470 := d377
			snap471 := d378
			snap472 := d379
			snap473 := d381
			snap474 := d382
			snap475 := d383
			snap476 := d384
			snap477 := d385
			snap478 := d386
			snap479 := d387
			snap480 := d388
			snap481 := d390
			snap482 := d391
			snap483 := d392
			snap484 := d394
			snap485 := d395
			snap486 := d396
			alloc487 := ctx.SnapshotAllocState()
			bbs[22].Render()
			ctx.RestoreAllocState(alloc487)
			d0 = snap397
			d1 = snap398
			d2 = snap399
			d3 = snap400
			d4 = snap401
			d5 = snap402
			d13 = snap403
			d14 = snap404
			d15 = snap405
			d26 = snap406
			d27 = snap407
			d40 = snap408
			d41 = snap409
			d42 = snap410
			d58 = snap411
			d59 = snap412
			d60 = snap413
			d79 = snap414
			d80 = snap415
			d101 = snap416
			d102 = snap417
			d125 = snap418
			d126 = snap419
			d151 = snap420
			d153 = snap421
			d155 = snap422
			d156 = snap423
			d157 = snap424
			d158 = snap425
			d159 = snap426
			d160 = snap427
			d161 = snap428
			d162 = snap429
			d163 = snap430
			d164 = snap431
			d165 = snap432
			d166 = snap433
			d167 = snap434
			d168 = snap435
			d170 = snap436
			d171 = snap437
			d172 = snap438
			d173 = snap439
			d174 = snap440
			d175 = snap441
			d176 = snap442
			d177 = snap443
			d179 = snap444
			d180 = snap445
			d181 = snap446
			d233 = snap447
			d234 = snap448
			d236 = snap449
			d237 = snap450
			d238 = snap451
			d295 = snap452
			d296 = snap453
			d298 = snap454
			d299 = snap455
			d300 = snap456
			d362 = snap457
			d364 = snap458
			d366 = snap459
			d367 = snap460
			d368 = snap461
			d369 = snap462
			d370 = snap463
			d371 = snap464
			d372 = snap465
			d373 = snap466
			d374 = snap467
			d375 = snap468
			d376 = snap469
			d377 = snap470
			d378 = snap471
			d379 = snap472
			d381 = snap473
			d382 = snap474
			d383 = snap475
			d384 = snap476
			d385 = snap477
			d386 = snap478
			d387 = snap479
			d388 = snap480
			d390 = snap481
			d391 = snap482
			d392 = snap483
			d394 = snap484
			d395 = snap485
			d396 = snap486
		}
		if !bbs[17].Rendered {
			return bbs[17].Render()
		}
		return result
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d488 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d488)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d488 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d488)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d488 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d488.Type = tagInt
			ctx.BindReg(d488.Reg, &d488)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d489 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d489 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d489)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d489 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d489)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d489 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d489.Type = tagInt
			ctx.BindReg(d489.Reg, &d489)
		}
		ctx.EnsureDesc(&d488)
		resultTarget490 := false
		_ = resultTarget490
		ctx.EnsureDesc(&d489)
		ctx.EnsureDescsTogether(&d488, &d489)
		if d488.Loc == LocImm && d489.Loc == LocImm {
			d491 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d488.Imm.Int() < d489.Imm.Int())}
		} else if d489.Loc == LocImm {
			r45 := ctx.AllocReg()
			if d489.Imm.Int() >= -2147483648 && d489.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d488.Reg, int32(d489.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d489.Imm.Int()))
				ctx.EmitCmpInt64(d488.Reg, ctx.ScratchReg)
			}
			d491 = ctx.DeferBooleanFlags(r45, CondSignedLess)
			ctx.BindReg(r45, &d491)
		} else if d488.Loc == LocImm {
			r46 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d488.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d489.Reg)
			d491 = ctx.DeferBooleanFlags(r46, CondSignedLess)
			ctx.BindReg(r46, &d491)
		} else {
			r47 := ctx.AllocReg()
			ctx.EmitCmpInt64(d488.Reg, d489.Reg)
			d491 = ctx.DeferBooleanFlags(r47, CondSignedLess)
			ctx.BindReg(r47, &d491)
		}
		ctx.FreeDesc(&d488)
		ctx.FreeDesc(&d489)
		ctx.SyncDesc(&d491)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d491) {
			return d491
		}
		ctx.EnsureDesc(&d491)
		ctx.EmitMovToReg(result.Reg, d491)
		result.Type = d491.Type
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d492 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d492 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d492)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d492 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d492)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d492 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d492.Type = tagInt
			ctx.BindReg(d492.Reg, &d492)
		}
		ctx.EnsureDesc(&d492)
		ctx.EnsureDesc(&d492)
		if d492.Loc == LocImm {
			d493 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d492.Imm.Int()))}
		} else {
			var r48 Reg
			r48 = d492.Reg
			d492.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r48)
			d493 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r48}
			ctx.BindReg(r48, &d493)
		}
		ctx.FreeDesc(&d492)
		d494 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d493)
		resultTarget495 := false
		_ = resultTarget495
		ctx.EnsureDesc(&d494)
		ctx.EnsureDescsTogether(&d493, &d494)
		if d493.Loc == LocImm && d494.Loc == LocImm {
			d496 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d493.Imm.Float() < d494.Imm.Float())}
		} else if d494.Loc == LocImm {
			var r49 Reg
			if result.Loc == LocReg && result.Reg != d493.Reg {
				r49 = result.Reg
				resultTarget495 = true
			} else {
				r49 = ctx.AllocRegExcept(d493.Reg)
			}
			_, yBits := d494.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d493.Reg)
			d496 = ctx.DeferBooleanFlags(r49, CondUnsignedAbove)
			ctx.BindReg(r49, &d496)
		} else if d493.Loc == LocImm {
			var r50 Reg
			if result.Loc == LocReg && result.Reg != d494.Reg {
				r50 = result.Reg
				resultTarget495 = true
			} else {
				r50 = ctx.AllocRegExcept(d494.Reg)
			}
			_, xBits := d493.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d494.Reg, ctx.ScratchReg)
			d496 = ctx.DeferBooleanFlags(r50, CondUnsignedAbove)
			ctx.BindReg(r50, &d496)
		} else {
			var r51 Reg
			if result.Loc == LocReg && result.Reg != d493.Reg && result.Reg != d494.Reg {
				r51 = result.Reg
				resultTarget495 = true
			} else {
				r51 = ctx.AllocRegExcept(d493.Reg, d494.Reg)
			}
			ctx.EmitCmpFloat64(d494.Reg, d493.Reg)
			d496 = ctx.DeferBooleanFlags(r51, CondUnsignedAbove)
			ctx.BindReg(r51, &d496)
		}
		ctx.FreeDesc(&d493)
		ctx.FreeDesc(&d494)
		ctx.SyncDesc(&d496)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d496) {
			return d496
		}
		ctx.EnsureDesc(&d496)
		ctx.EmitMovToReg(result.Reg, d496)
		result.Type = d496.Type
		ctx.EmitJmp(lbl0)
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
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d497 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d497 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d497)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d497 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d497)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d497 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d497.Type = tagInt
			ctx.BindReg(d497.Reg, &d497)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d498 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d498 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d498)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d498 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d498)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d498 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d498.Type = tagInt
			ctx.BindReg(d498.Reg, &d498)
		}
		ctx.EnsureDesc(&d497)
		resultTarget499 := false
		_ = resultTarget499
		ctx.EnsureDesc(&d498)
		ctx.EnsureDescsTogether(&d497, &d498)
		if d497.Loc == LocImm && d498.Loc == LocImm {
			d500 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d497.Imm.Int() < d498.Imm.Int())}
		} else if d498.Loc == LocImm {
			r52 := ctx.AllocReg()
			if d498.Imm.Int() >= -2147483648 && d498.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d497.Reg, int32(d498.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d498.Imm.Int()))
				ctx.EmitCmpInt64(d497.Reg, ctx.ScratchReg)
			}
			d500 = ctx.DeferBooleanFlags(r52, CondSignedLess)
			ctx.BindReg(r52, &d500)
		} else if d497.Loc == LocImm {
			r53 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d497.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d498.Reg)
			d500 = ctx.DeferBooleanFlags(r53, CondSignedLess)
			ctx.BindReg(r53, &d500)
		} else {
			r54 := ctx.AllocReg()
			ctx.EmitCmpInt64(d497.Reg, d498.Reg)
			d500 = ctx.DeferBooleanFlags(r54, CondSignedLess)
			ctx.BindReg(r54, &d500)
		}
		ctx.FreeDesc(&d497)
		ctx.FreeDesc(&d498)
		ctx.SyncDesc(&d500)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d500) {
			return d500
		}
		ctx.EnsureDesc(&d500)
		ctx.EmitMovToReg(result.Reg, d500)
		result.Type = d500.Type
		ctx.EmitJmp(lbl0)
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		if d1.Loc == LocImm {
			d501 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x5))}
		} else {
			r55 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 5)
			d501 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r55, Condition: CondEqual}
			ctx.BindReg(r55, &d501)
		}
		d502 = d501
		ctx.SyncDesc(&d502)
		if d502.Loc != LocImm && d502.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d502.Loc == LocImm {
			if d502.Imm.Bool() {
				return bbs[21].Render()
			}
			return bbs[23].Render()
		}
		ctx.EmitJump(d502.Condition, lbl22)
		if bbs[23].Rendered {
			ctx.EmitJmp(lbl24)
		}
		ctx.FreeDesc(&d501)
		ctx.FlushRegisterMoves()
		if !bbs[23].Rendered {
			snap503 := d0
			snap504 := d1
			snap505 := d2
			snap506 := d3
			snap507 := d4
			snap508 := d5
			snap509 := d13
			snap510 := d14
			snap511 := d15
			snap512 := d26
			snap513 := d27
			snap514 := d40
			snap515 := d41
			snap516 := d42
			snap517 := d58
			snap518 := d59
			snap519 := d60
			snap520 := d79
			snap521 := d80
			snap522 := d101
			snap523 := d102
			snap524 := d125
			snap525 := d126
			snap526 := d151
			snap527 := d153
			snap528 := d155
			snap529 := d156
			snap530 := d157
			snap531 := d158
			snap532 := d159
			snap533 := d160
			snap534 := d161
			snap535 := d162
			snap536 := d163
			snap537 := d164
			snap538 := d165
			snap539 := d166
			snap540 := d167
			snap541 := d168
			snap542 := d170
			snap543 := d171
			snap544 := d172
			snap545 := d173
			snap546 := d174
			snap547 := d175
			snap548 := d176
			snap549 := d177
			snap550 := d179
			snap551 := d180
			snap552 := d181
			snap553 := d233
			snap554 := d234
			snap555 := d236
			snap556 := d237
			snap557 := d238
			snap558 := d295
			snap559 := d296
			snap560 := d298
			snap561 := d299
			snap562 := d300
			snap563 := d362
			snap564 := d364
			snap565 := d366
			snap566 := d367
			snap567 := d368
			snap568 := d369
			snap569 := d370
			snap570 := d371
			snap571 := d372
			snap572 := d373
			snap573 := d374
			snap574 := d375
			snap575 := d376
			snap576 := d377
			snap577 := d378
			snap578 := d379
			snap579 := d381
			snap580 := d382
			snap581 := d383
			snap582 := d384
			snap583 := d385
			snap584 := d386
			snap585 := d387
			snap586 := d388
			snap587 := d390
			snap588 := d391
			snap589 := d392
			snap590 := d394
			snap591 := d395
			snap592 := d396
			snap593 := d488
			snap594 := d489
			snap595 := d491
			snap596 := d492
			snap597 := d493
			snap598 := d494
			snap599 := d496
			snap600 := d497
			snap601 := d498
			snap602 := d500
			snap603 := d501
			snap604 := d502
			alloc605 := ctx.SnapshotAllocState()
			bbs[23].Render()
			ctx.RestoreAllocState(alloc605)
			d0 = snap503
			d1 = snap504
			d2 = snap505
			d3 = snap506
			d4 = snap507
			d5 = snap508
			d13 = snap509
			d14 = snap510
			d15 = snap511
			d26 = snap512
			d27 = snap513
			d40 = snap514
			d41 = snap515
			d42 = snap516
			d58 = snap517
			d59 = snap518
			d60 = snap519
			d79 = snap520
			d80 = snap521
			d101 = snap522
			d102 = snap523
			d125 = snap524
			d126 = snap525
			d151 = snap526
			d153 = snap527
			d155 = snap528
			d156 = snap529
			d157 = snap530
			d158 = snap531
			d159 = snap532
			d160 = snap533
			d161 = snap534
			d162 = snap535
			d163 = snap536
			d164 = snap537
			d165 = snap538
			d166 = snap539
			d167 = snap540
			d168 = snap541
			d170 = snap542
			d171 = snap543
			d172 = snap544
			d173 = snap545
			d174 = snap546
			d175 = snap547
			d176 = snap548
			d177 = snap549
			d179 = snap550
			d180 = snap551
			d181 = snap552
			d233 = snap553
			d234 = snap554
			d236 = snap555
			d237 = snap556
			d238 = snap557
			d295 = snap558
			d296 = snap559
			d298 = snap560
			d299 = snap561
			d300 = snap562
			d362 = snap563
			d364 = snap564
			d366 = snap565
			d367 = snap566
			d368 = snap567
			d369 = snap568
			d370 = snap569
			d371 = snap570
			d372 = snap571
			d373 = snap572
			d374 = snap573
			d375 = snap574
			d376 = snap575
			d377 = snap576
			d378 = snap577
			d379 = snap578
			d381 = snap579
			d382 = snap580
			d383 = snap581
			d384 = snap582
			d385 = snap583
			d386 = snap584
			d387 = snap585
			d388 = snap586
			d390 = snap587
			d391 = snap588
			d392 = snap589
			d394 = snap590
			d395 = snap591
			d396 = snap592
			d488 = snap593
			d489 = snap594
			d491 = snap595
			d492 = snap596
			d493 = snap597
			d494 = snap598
			d496 = snap599
			d497 = snap600
			d498 = snap601
			d500 = snap602
			d501 = snap603
			d502 = snap604
		}
		if !bbs[21].Rendered {
			return bbs[21].Render()
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
		d606 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d1, d3}, 1)
		d606.NoHeapPointer = true
		ctx.EmitAndRegImm32(d606.Reg, 1)
		d606.Type = tagBool
		ctx.BindReg(d606.Reg, &d606)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.SyncDesc(&d606)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d606) {
			return d606
		}
		ctx.EnsureDesc(&d606)
		ctx.EmitMovToReg(result.Reg, d606)
		result.Type = d606.Type
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
