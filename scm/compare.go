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
			return nativeDateSeconds(a) < b.Int()
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
			seconds := nativeDateSeconds(b)
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
	var d210 JITValueDesc
	_ = d210
	var d211 JITValueDesc
	_ = d211
	var d247 JITValueDesc
	_ = d247
	var d248 JITValueDesc
	_ = d248
	var d250 JITValueDesc
	_ = d250
	var d251 JITValueDesc
	_ = d251
	var d252 JITValueDesc
	_ = d252
	var d293 JITValueDesc
	_ = d293
	var d294 JITValueDesc
	_ = d294
	var d296 JITValueDesc
	_ = d296
	var d297 JITValueDesc
	_ = d297
	var d298 JITValueDesc
	_ = d298
	var d344 JITValueDesc
	_ = d344
	var d346 JITValueDesc
	_ = d346
	var d347 JITValueDesc
	_ = d347
	var d348 JITValueDesc
	_ = d348
	var d350 JITValueDesc
	_ = d350
	var d351 JITValueDesc
	_ = d351
	var d352 JITValueDesc
	_ = d352
	var d405 JITValueDesc
	_ = d405
	var d406 JITValueDesc
	_ = d406
	var d407 JITValueDesc
	_ = d407
	var d408 JITValueDesc
	_ = d408
	var d521 JITValueDesc
	_ = d521
	var d522 JITValueDesc
	_ = d522
	var d581 JITValueDesc
	_ = d581
	var d582 JITValueDesc
	_ = d582
	var d703 JITValueDesc
	_ = d703
	var d704 JITValueDesc
	_ = d704
	var d705 JITValueDesc
	_ = d705
	var d706 JITValueDesc
	_ = d706
	var d707 JITValueDesc
	_ = d707
	var d709 JITValueDesc
	_ = d709
	var d710 JITValueDesc
	_ = d710
	var d711 JITValueDesc
	_ = d711
	var d712 JITValueDesc
	_ = d712
	var d714 JITValueDesc
	_ = d714
	var d715 JITValueDesc
	_ = d715
	var d716 JITValueDesc
	_ = d716
	var d718 JITValueDesc
	_ = d718
	var d719 JITValueDesc
	_ = d719
	var d720 JITValueDesc
	_ = d720
	var d796 JITValueDesc
	_ = d796
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
		args[0] = JITPrepareScmerGoArg(ctx, args[0])
		ctx.SyncDesc(&args[0])
		d170 = ctx.EmitGoCallScalar(jitGoABI_7b4abd9531444504, []JITValueDesc{args[0]}, 1)
		d170.NoHeapPointer = true
		ctx.BindReg(d170.Reg, &d170)
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
		args[0] = JITPrepareScmerGoArg(ctx, args[0])
		args[1] = JITPrepareScmerGoArg(ctx, args[1])
		ctx.SyncDesc(&args[0])
		ctx.SyncDesc(&args[1])
		d207 = ctx.EmitGoCallScalar(jitGoABI_d69f4c85104836f4, []JITValueDesc{args[0], args[1]}, 1)
		d207.NoHeapPointer = true
		ctx.BindReg(d207.Reg, &d207)
		ctx.EnsureDesc(&d207)
		resultTarget208 := false
		_ = resultTarget208
		if d207.Loc == LocImm {
			d209 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d207.Imm.Int() < 0)}
		} else {
			r12 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d207.Reg, 0)
			d209 = ctx.DeferBooleanFlags(r12, CondSignedLess)
			ctx.BindReg(r12, &d209)
		}
		ctx.FreeDesc(&d207)
		ctx.SyncDesc(&d209)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d209) {
			return d209
		}
		ctx.EnsureDesc(&d209)
		ctx.EmitMovToReg(result.Reg, d209)
		result.Type = d209.Type
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
			d210 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x1))}
		} else {
			r13 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 1)
			d210 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondEqual}
			ctx.BindReg(r13, &d210)
		}
		d211 = d210
		ctx.SyncDesc(&d211)
		if d211.Loc != LocImm && d211.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d211.Loc == LocImm {
			if d211.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[17].Render()
		}
		ctx.EmitJump(d211.Condition, lbl16)
		if bbs[17].Rendered {
			ctx.EmitJmp(lbl18)
		}
		ctx.FreeDesc(&d210)
		ctx.FlushRegisterMoves()
		if !bbs[17].Rendered {
			snap212 := d1
			snap213 := d2
			snap214 := d3
			snap215 := d4
			snap216 := d5
			snap217 := d6
			snap218 := d7
			snap219 := d8
			snap220 := d18
			snap221 := d19
			snap222 := d20
			snap223 := d33
			snap224 := d34
			snap225 := d49
			snap226 := d50
			snap227 := d51
			snap228 := d69
			snap229 := d70
			snap230 := d71
			snap231 := d92
			snap232 := d93
			snap233 := d116
			snap234 := d117
			snap235 := d142
			snap236 := d143
			snap237 := d170
			snap238 := d171
			snap239 := d173
			snap240 := d174
			snap241 := d175
			snap242 := d207
			snap243 := d209
			snap244 := d210
			snap245 := d211
			alloc246 := ctx.SnapshotAllocState()
			bbs[17].Render()
			ctx.RestoreAllocState(alloc246)
			d1 = snap212
			d2 = snap213
			d3 = snap214
			d4 = snap215
			d5 = snap216
			d6 = snap217
			d7 = snap218
			d8 = snap219
			d18 = snap220
			d19 = snap221
			d20 = snap222
			d33 = snap223
			d34 = snap224
			d49 = snap225
			d50 = snap226
			d51 = snap227
			d69 = snap228
			d70 = snap229
			d71 = snap230
			d92 = snap231
			d93 = snap232
			d116 = snap233
			d117 = snap234
			d142 = snap235
			d143 = snap236
			d170 = snap237
			d171 = snap238
			d173 = snap239
			d174 = snap240
			d175 = snap241
			d207 = snap242
			d209 = snap243
			d210 = snap244
			d211 = snap245
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
		d248 = args[1]
		ctx.SyncDesc(&d248)
		if d248.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d248.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d248.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d248 = tmpScalar
		}
		d248 = JITPrepareScmerGoArg(ctx, d248)
		if d248.Loc != LocRegPair && d248.Loc != LocStackPair && d248.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d247 = ctx.EmitGoCallScalar(jitGoABI_0cbff6f55ee96f6d, []JITValueDesc{d248}, 2)
		ctx.EnsureDesc(&d247)
		if d247.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d247.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d247.Imm)
			ptrWord, _ := d247.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d247.Imm.String())))
			d247 = tmpPair
		} else if d247.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d247.Type, Reg: ctx.AllocRegExcept(d247.Reg), Reg2: ctx.AllocRegExcept(d247.Reg)}
			switch d247.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d247)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d247)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d247)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d247)
			d247 = tmpPair
		}
		if d247.Loc != LocRegPair && d247.Loc != LocStackPair && d247.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateValue arg0)")
		}
		ctx.SyncDesc(&d247)
		callResults249 := JITEmitGoCallResults(ctx, jitGoABI_58434e8dfe39c263, []JITValueDesc{d247})
		d250 = callResults249[0]
		_ = d250
		d251 = callResults249[1]
		_ = d251
		ctx.StabilizeDescForControlFlow(&d250)
		d252 = d251
		ctx.EnsureDesc(&d252)
		if d252.Loc != LocImm && d252.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d252.Loc == LocImm {
			if d252.Imm.Bool() {
				return bbs[18].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitCmpRegImm32(d252.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl19)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
			snap253 := d1
			snap254 := d2
			snap255 := d3
			snap256 := d4
			snap257 := d5
			snap258 := d6
			snap259 := d7
			snap260 := d8
			snap261 := d18
			snap262 := d19
			snap263 := d20
			snap264 := d33
			snap265 := d34
			snap266 := d49
			snap267 := d50
			snap268 := d51
			snap269 := d69
			snap270 := d70
			snap271 := d71
			snap272 := d92
			snap273 := d93
			snap274 := d116
			snap275 := d117
			snap276 := d142
			snap277 := d143
			snap278 := d170
			snap279 := d171
			snap280 := d173
			snap281 := d174
			snap282 := d175
			snap283 := d207
			snap284 := d209
			snap285 := d210
			snap286 := d211
			snap287 := d247
			snap288 := d248
			snap289 := d250
			snap290 := d251
			snap291 := d252
			alloc292 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc292)
			d1 = snap253
			d2 = snap254
			d3 = snap255
			d4 = snap256
			d5 = snap257
			d6 = snap258
			d7 = snap259
			d8 = snap260
			d18 = snap261
			d19 = snap262
			d20 = snap263
			d33 = snap264
			d34 = snap265
			d49 = snap266
			d50 = snap267
			d51 = snap268
			d69 = snap269
			d70 = snap270
			d71 = snap271
			d92 = snap272
			d93 = snap273
			d116 = snap274
			d117 = snap275
			d142 = snap276
			d143 = snap277
			d170 = snap278
			d171 = snap279
			d173 = snap280
			d174 = snap281
			d175 = snap282
			d207 = snap283
			d209 = snap284
			d210 = snap285
			d211 = snap286
			d247 = snap287
			d248 = snap288
			d250 = snap289
			d251 = snap290
			d252 = snap291
		}
		if !bbs[18].Rendered {
			return bbs[18].Render()
		}
		return result
		ctx.FreeDesc(&d251)
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
		d293 = ctx.EmitFloatDesc(args[0])
		d294 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d293)
		resultTarget295 := false
		_ = resultTarget295
		ctx.EnsureDesc(&d294)
		ctx.EnsureDescsTogether(&d293, &d294)
		if d293.Loc == LocImm && d294.Loc == LocImm {
			d296 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d293.Imm.Float() < d294.Imm.Float())}
		} else if d294.Loc == LocImm {
			var r14 Reg
			if result.Loc == LocReg && result.Reg != d293.Reg {
				r14 = result.Reg
				resultTarget295 = true
			} else {
				r14 = ctx.AllocRegExcept(d293.Reg)
			}
			_, yBits := d294.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d293.Reg)
			d296 = ctx.DeferBooleanFlags(r14, CondUnsignedAbove)
			ctx.BindReg(r14, &d296)
		} else if d293.Loc == LocImm {
			var r15 Reg
			if result.Loc == LocReg && result.Reg != d294.Reg {
				r15 = result.Reg
				resultTarget295 = true
			} else {
				r15 = ctx.AllocRegExcept(d294.Reg)
			}
			_, xBits := d293.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d294.Reg, ctx.ScratchReg)
			d296 = ctx.DeferBooleanFlags(r15, CondUnsignedAbove)
			ctx.BindReg(r15, &d296)
		} else {
			var r16 Reg
			if result.Loc == LocReg && result.Reg != d293.Reg && result.Reg != d294.Reg {
				r16 = result.Reg
				resultTarget295 = true
			} else {
				r16 = ctx.AllocRegExcept(d293.Reg, d294.Reg)
			}
			ctx.EmitCmpFloat64(d294.Reg, d293.Reg)
			d296 = ctx.DeferBooleanFlags(r16, CondUnsignedAbove)
			ctx.BindReg(r16, &d296)
		}
		ctx.FreeDesc(&d293)
		ctx.FreeDesc(&d294)
		ctx.SyncDesc(&d296)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d296) {
			return d296
		}
		ctx.EnsureDesc(&d296)
		ctx.EmitMovToReg(result.Reg, d296)
		result.Type = d296.Type
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
			d297 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x2))}
		} else {
			r17 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 2)
			d297 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondEqual}
			ctx.BindReg(r17, &d297)
		}
		d298 = d297
		ctx.SyncDesc(&d298)
		if d298.Loc != LocImm && d298.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d298.Loc == LocImm {
			if d298.Imm.Bool() {
				return bbs[15].Render()
			}
			return bbs[16].Render()
		}
		ctx.EmitJump(d298.Condition, lbl16)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d297)
		ctx.FlushRegisterMoves()
		if !bbs[16].Rendered {
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
			snap310 := d33
			snap311 := d34
			snap312 := d49
			snap313 := d50
			snap314 := d51
			snap315 := d69
			snap316 := d70
			snap317 := d71
			snap318 := d92
			snap319 := d93
			snap320 := d116
			snap321 := d117
			snap322 := d142
			snap323 := d143
			snap324 := d170
			snap325 := d171
			snap326 := d173
			snap327 := d174
			snap328 := d175
			snap329 := d207
			snap330 := d209
			snap331 := d210
			snap332 := d211
			snap333 := d247
			snap334 := d248
			snap335 := d250
			snap336 := d251
			snap337 := d252
			snap338 := d293
			snap339 := d294
			snap340 := d296
			snap341 := d297
			snap342 := d298
			alloc343 := ctx.SnapshotAllocState()
			bbs[16].Render()
			ctx.RestoreAllocState(alloc343)
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
			d33 = snap310
			d34 = snap311
			d49 = snap312
			d50 = snap313
			d51 = snap314
			d69 = snap315
			d70 = snap316
			d71 = snap317
			d92 = snap318
			d93 = snap319
			d116 = snap320
			d117 = snap321
			d142 = snap322
			d143 = snap323
			d170 = snap324
			d171 = snap325
			d173 = snap326
			d174 = snap327
			d175 = snap328
			d207 = snap329
			d209 = snap330
			d210 = snap331
			d211 = snap332
			d247 = snap333
			d248 = snap334
			d250 = snap335
			d251 = snap336
			d252 = snap337
			d293 = snap338
			d294 = snap339
			d296 = snap340
			d297 = snap341
			d298 = snap342
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
		args[0] = JITPrepareScmerGoArg(ctx, args[0])
		d250 = JITPrepareScmerGoArg(ctx, d250)
		ctx.SyncDesc(&args[0])
		ctx.SyncDesc(&d250)
		d344 = ctx.EmitGoCallScalar(jitGoABI_d69f4c85104836f4, []JITValueDesc{args[0], d250}, 1)
		d344.NoHeapPointer = true
		ctx.BindReg(d344.Reg, &d344)
		ctx.EnsureDesc(&d344)
		resultTarget345 := false
		_ = resultTarget345
		if d344.Loc == LocImm {
			d346 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d344.Imm.Int() < 0)}
		} else {
			r18 := ctx.AllocReg()
			ctx.EmitCmpRegImm32(d344.Reg, 0)
			d346 = ctx.DeferBooleanFlags(r18, CondSignedLess)
			ctx.BindReg(r18, &d346)
		}
		ctx.FreeDesc(&d344)
		ctx.SyncDesc(&d346)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d346) {
			return d346
		}
		ctx.EnsureDesc(&d346)
		ctx.EmitMovToReg(result.Reg, d346)
		result.Type = d346.Type
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
		d347 = ctx.EmitFloatDesc(args[0])
		d348 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d347)
		resultTarget349 := false
		_ = resultTarget349
		ctx.EnsureDesc(&d348)
		ctx.EnsureDescsTogether(&d347, &d348)
		if d347.Loc == LocImm && d348.Loc == LocImm {
			d350 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d347.Imm.Float() < d348.Imm.Float())}
		} else if d348.Loc == LocImm {
			var r19 Reg
			if result.Loc == LocReg && result.Reg != d347.Reg {
				r19 = result.Reg
				resultTarget349 = true
			} else {
				r19 = ctx.AllocRegExcept(d347.Reg)
			}
			_, yBits := d348.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d347.Reg)
			d350 = ctx.DeferBooleanFlags(r19, CondUnsignedAbove)
			ctx.BindReg(r19, &d350)
		} else if d347.Loc == LocImm {
			var r20 Reg
			if result.Loc == LocReg && result.Reg != d348.Reg {
				r20 = result.Reg
				resultTarget349 = true
			} else {
				r20 = ctx.AllocRegExcept(d348.Reg)
			}
			_, xBits := d347.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d348.Reg, ctx.ScratchReg)
			d350 = ctx.DeferBooleanFlags(r20, CondUnsignedAbove)
			ctx.BindReg(r20, &d350)
		} else {
			var r21 Reg
			if result.Loc == LocReg && result.Reg != d347.Reg && result.Reg != d348.Reg {
				r21 = result.Reg
				resultTarget349 = true
			} else {
				r21 = ctx.AllocRegExcept(d347.Reg, d348.Reg)
			}
			ctx.EmitCmpFloat64(d348.Reg, d347.Reg)
			d350 = ctx.DeferBooleanFlags(r21, CondUnsignedAbove)
			ctx.BindReg(r21, &d350)
		}
		ctx.FreeDesc(&d347)
		ctx.FreeDesc(&d348)
		ctx.SyncDesc(&d350)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d350) {
			return d350
		}
		ctx.EnsureDesc(&d350)
		ctx.EmitMovToReg(result.Reg, d350)
		result.Type = d350.Type
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
			d351 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x3))}
		} else {
			r22 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 3)
			d351 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondEqual}
			ctx.BindReg(r22, &d351)
		}
		d352 = d351
		ctx.SyncDesc(&d352)
		if d352.Loc != LocImm && d352.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d352.Loc == LocImm {
			if d352.Imm.Bool() {
				return bbs[19].Render()
			}
			return bbs[30].Render()
		}
		ctx.EmitJump(d352.Condition, lbl20)
		if bbs[30].Rendered {
			ctx.EmitJmp(lbl31)
		}
		ctx.FreeDesc(&d351)
		ctx.FlushRegisterMoves()
		if !bbs[30].Rendered {
			snap353 := d1
			snap354 := d2
			snap355 := d3
			snap356 := d4
			snap357 := d5
			snap358 := d6
			snap359 := d7
			snap360 := d8
			snap361 := d18
			snap362 := d19
			snap363 := d20
			snap364 := d33
			snap365 := d34
			snap366 := d49
			snap367 := d50
			snap368 := d51
			snap369 := d69
			snap370 := d70
			snap371 := d71
			snap372 := d92
			snap373 := d93
			snap374 := d116
			snap375 := d117
			snap376 := d142
			snap377 := d143
			snap378 := d170
			snap379 := d171
			snap380 := d173
			snap381 := d174
			snap382 := d175
			snap383 := d207
			snap384 := d209
			snap385 := d210
			snap386 := d211
			snap387 := d247
			snap388 := d248
			snap389 := d250
			snap390 := d251
			snap391 := d252
			snap392 := d293
			snap393 := d294
			snap394 := d296
			snap395 := d297
			snap396 := d298
			snap397 := d344
			snap398 := d346
			snap399 := d347
			snap400 := d348
			snap401 := d350
			snap402 := d351
			snap403 := d352
			alloc404 := ctx.SnapshotAllocState()
			bbs[30].Render()
			ctx.RestoreAllocState(alloc404)
			d1 = snap353
			d2 = snap354
			d3 = snap355
			d4 = snap356
			d5 = snap357
			d6 = snap358
			d7 = snap359
			d8 = snap360
			d18 = snap361
			d19 = snap362
			d20 = snap363
			d33 = snap364
			d34 = snap365
			d49 = snap366
			d50 = snap367
			d51 = snap368
			d69 = snap369
			d70 = snap370
			d71 = snap371
			d92 = snap372
			d93 = snap373
			d116 = snap374
			d117 = snap375
			d142 = snap376
			d143 = snap377
			d170 = snap378
			d171 = snap379
			d173 = snap380
			d174 = snap381
			d175 = snap382
			d207 = snap383
			d209 = snap384
			d210 = snap385
			d211 = snap386
			d247 = snap387
			d248 = snap388
			d250 = snap389
			d251 = snap390
			d252 = snap391
			d293 = snap392
			d294 = snap393
			d296 = snap394
			d297 = snap395
			d298 = snap396
			d344 = snap397
			d346 = snap398
			d347 = snap399
			d348 = snap400
			d350 = snap401
			d351 = snap402
			d352 = snap403
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
		args[1] = JITPrepareScmerGoArg(ctx, args[1])
		ctx.SyncDesc(&args[1])
		d405 = ctx.EmitGoCallScalar(jitGoABI_7b4abd9531444504, []JITValueDesc{args[1]}, 1)
		d405.NoHeapPointer = true
		ctx.BindReg(d405.Reg, &d405)
		ctx.StabilizeDescForControlFlow(&d405)
		if args[0].Loc != LocImm {
			ctx.EnsureDesc(&args[0])
		}
		if args[0].Loc == LocImm {
			d406 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d406 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d406)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d406 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d406)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d406 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d406.Type = tagInt
			ctx.BindReg(d406.Reg, &d406)
		}
		ctx.StabilizeDescForControlFlow(&d406)
		ctx.EnsureDesc(&d406)
		ctx.EnsureDesc(&d405)
		ctx.EnsureDescsTogether(&d406, &d405)
		if d406.Loc == LocImm && d405.Loc == LocImm {
			d407 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d406.Imm.Int() < d405.Imm.Int())}
		} else if d405.Loc == LocImm {
			r23 := ctx.AllocRegExcept(d406.Reg)
			if d405.Imm.Int() >= -2147483648 && d405.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d406.Reg, int32(d405.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d405.Imm.Int()))
				ctx.EmitCmpInt64(d406.Reg, ctx.ScratchReg)
			}
			d407 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedLess}
			ctx.BindReg(r23, &d407)
		} else if d406.Loc == LocImm {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d406.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d405.Reg)
			d407 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
			ctx.BindReg(r24, &d407)
		} else {
			r25 := ctx.AllocRegExcept(d406.Reg)
			ctx.EmitCmpInt64(d406.Reg, d405.Reg)
			d407 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedLess}
			ctx.BindReg(r25, &d407)
		}
		d408 = d407
		ctx.SyncDesc(&d408)
		if d408.Loc != LocImm && d408.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d408.Loc == LocImm {
			if d408.Imm.Bool() {
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
				return bbs[24].Render()
			}
			return bbs[23].Render()
		}
		lbl33 := ctx.ReserveLabel()
		ctx.EmitJump(d408.Condition, lbl33)
		ctx.EmitJmp(lbl24)
		ctx.FreeDesc(&d407)
		snap409 := d1
		snap410 := d2
		snap411 := d3
		snap412 := d4
		snap413 := d5
		snap414 := d6
		snap415 := d7
		snap416 := d8
		snap417 := d18
		snap418 := d19
		snap419 := d20
		snap420 := d33
		snap421 := d34
		snap422 := d49
		snap423 := d50
		snap424 := d51
		snap425 := d69
		snap426 := d70
		snap427 := d71
		snap428 := d92
		snap429 := d93
		snap430 := d116
		snap431 := d117
		snap432 := d142
		snap433 := d143
		snap434 := d170
		snap435 := d171
		snap436 := d173
		snap437 := d174
		snap438 := d175
		snap439 := d207
		snap440 := d209
		snap441 := d210
		snap442 := d211
		snap443 := d247
		snap444 := d248
		snap445 := d250
		snap446 := d251
		snap447 := d252
		snap448 := d293
		snap449 := d294
		snap450 := d296
		snap451 := d297
		snap452 := d298
		snap453 := d344
		snap454 := d346
		snap455 := d347
		snap456 := d348
		snap457 := d350
		snap458 := d351
		snap459 := d352
		snap460 := d405
		snap461 := d406
		snap462 := d407
		snap463 := d408
		alloc464 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl33)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[24].PhiBase)+int32(0))
		ctx.EmitJmp(lbl25)
		ctx.RestoreAllocState(alloc464)
		d1 = snap409
		d2 = snap410
		d3 = snap411
		d4 = snap412
		d5 = snap413
		d6 = snap414
		d7 = snap415
		d8 = snap416
		d18 = snap417
		d19 = snap418
		d20 = snap419
		d33 = snap420
		d34 = snap421
		d49 = snap422
		d50 = snap423
		d51 = snap424
		d69 = snap425
		d70 = snap426
		d71 = snap427
		d92 = snap428
		d93 = snap429
		d116 = snap430
		d117 = snap431
		d142 = snap432
		d143 = snap433
		d170 = snap434
		d171 = snap435
		d173 = snap436
		d174 = snap437
		d175 = snap438
		d207 = snap439
		d209 = snap440
		d210 = snap441
		d211 = snap442
		d247 = snap443
		d248 = snap444
		d250 = snap445
		d251 = snap446
		d252 = snap447
		d293 = snap448
		d294 = snap449
		d296 = snap450
		d297 = snap451
		d298 = snap452
		d344 = snap453
		d346 = snap454
		d347 = snap455
		d348 = snap456
		d350 = snap457
		d351 = snap458
		d352 = snap459
		d405 = snap460
		d406 = snap461
		d407 = snap462
		d408 = snap463
		if !bbs[24].Rendered {
			snap465 := d1
			snap466 := d2
			snap467 := d3
			snap468 := d4
			snap469 := d5
			snap470 := d6
			snap471 := d7
			snap472 := d8
			snap473 := d18
			snap474 := d19
			snap475 := d20
			snap476 := d33
			snap477 := d34
			snap478 := d49
			snap479 := d50
			snap480 := d51
			snap481 := d69
			snap482 := d70
			snap483 := d71
			snap484 := d92
			snap485 := d93
			snap486 := d116
			snap487 := d117
			snap488 := d142
			snap489 := d143
			snap490 := d170
			snap491 := d171
			snap492 := d173
			snap493 := d174
			snap494 := d175
			snap495 := d207
			snap496 := d209
			snap497 := d210
			snap498 := d211
			snap499 := d247
			snap500 := d248
			snap501 := d250
			snap502 := d251
			snap503 := d252
			snap504 := d293
			snap505 := d294
			snap506 := d296
			snap507 := d297
			snap508 := d298
			snap509 := d344
			snap510 := d346
			snap511 := d347
			snap512 := d348
			snap513 := d350
			snap514 := d351
			snap515 := d352
			snap516 := d405
			snap517 := d406
			snap518 := d407
			snap519 := d408
			alloc520 := ctx.SnapshotAllocState()
			bbs[24].Render()
			ctx.RestoreAllocState(alloc520)
			d1 = snap465
			d2 = snap466
			d3 = snap467
			d4 = snap468
			d5 = snap469
			d6 = snap470
			d7 = snap471
			d8 = snap472
			d18 = snap473
			d19 = snap474
			d20 = snap475
			d33 = snap476
			d34 = snap477
			d49 = snap478
			d50 = snap479
			d51 = snap480
			d69 = snap481
			d70 = snap482
			d71 = snap483
			d92 = snap484
			d93 = snap485
			d116 = snap486
			d117 = snap487
			d142 = snap488
			d143 = snap489
			d170 = snap490
			d171 = snap491
			d173 = snap492
			d174 = snap493
			d175 = snap494
			d207 = snap495
			d209 = snap496
			d210 = snap497
			d211 = snap498
			d247 = snap499
			d248 = snap500
			d250 = snap501
			d251 = snap502
			d252 = snap503
			d293 = snap504
			d294 = snap505
			d296 = snap506
			d297 = snap507
			d298 = snap508
			d344 = snap509
			d346 = snap510
			d347 = snap511
			d348 = snap512
			d350 = snap513
			d351 = snap514
			d352 = snap515
			d405 = snap516
			d406 = snap517
			d407 = snap518
			d408 = snap519
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
			d521 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0x4))}
		} else {
			r26 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpRegImm32(d6.Reg, 4)
			d521 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondEqual}
			ctx.BindReg(r26, &d521)
		}
		d522 = d521
		ctx.SyncDesc(&d522)
		if d522.Loc != LocImm && d522.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d522.Loc == LocImm {
			if d522.Imm.Bool() {
				return bbs[27].Render()
			}
			return bbs[28].Render()
		}
		ctx.EmitJump(d522.Condition, lbl28)
		if bbs[28].Rendered {
			ctx.EmitJmp(lbl29)
		}
		ctx.FreeDesc(&d521)
		ctx.FlushRegisterMoves()
		if !bbs[28].Rendered {
			snap523 := d1
			snap524 := d2
			snap525 := d3
			snap526 := d4
			snap527 := d5
			snap528 := d6
			snap529 := d7
			snap530 := d8
			snap531 := d18
			snap532 := d19
			snap533 := d20
			snap534 := d33
			snap535 := d34
			snap536 := d49
			snap537 := d50
			snap538 := d51
			snap539 := d69
			snap540 := d70
			snap541 := d71
			snap542 := d92
			snap543 := d93
			snap544 := d116
			snap545 := d117
			snap546 := d142
			snap547 := d143
			snap548 := d170
			snap549 := d171
			snap550 := d173
			snap551 := d174
			snap552 := d175
			snap553 := d207
			snap554 := d209
			snap555 := d210
			snap556 := d211
			snap557 := d247
			snap558 := d248
			snap559 := d250
			snap560 := d251
			snap561 := d252
			snap562 := d293
			snap563 := d294
			snap564 := d296
			snap565 := d297
			snap566 := d298
			snap567 := d344
			snap568 := d346
			snap569 := d347
			snap570 := d348
			snap571 := d350
			snap572 := d351
			snap573 := d352
			snap574 := d405
			snap575 := d406
			snap576 := d407
			snap577 := d408
			snap578 := d521
			snap579 := d522
			alloc580 := ctx.SnapshotAllocState()
			bbs[28].Render()
			ctx.RestoreAllocState(alloc580)
			d1 = snap523
			d2 = snap524
			d3 = snap525
			d4 = snap526
			d5 = snap527
			d6 = snap528
			d7 = snap529
			d8 = snap530
			d18 = snap531
			d19 = snap532
			d20 = snap533
			d33 = snap534
			d34 = snap535
			d49 = snap536
			d50 = snap537
			d51 = snap538
			d69 = snap539
			d70 = snap540
			d71 = snap541
			d92 = snap542
			d93 = snap543
			d116 = snap544
			d117 = snap545
			d142 = snap546
			d143 = snap547
			d170 = snap548
			d171 = snap549
			d173 = snap550
			d174 = snap551
			d175 = snap552
			d207 = snap553
			d209 = snap554
			d210 = snap555
			d211 = snap556
			d247 = snap557
			d248 = snap558
			d250 = snap559
			d251 = snap560
			d252 = snap561
			d293 = snap562
			d294 = snap563
			d296 = snap564
			d297 = snap565
			d298 = snap566
			d344 = snap567
			d346 = snap568
			d347 = snap569
			d348 = snap570
			d350 = snap571
			d351 = snap572
			d352 = snap573
			d405 = snap574
			d406 = snap575
			d407 = snap576
			d408 = snap577
			d521 = snap578
			d522 = snap579
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
		ctx.EnsureDesc(&d406)
		ctx.EnsureDesc(&d405)
		ctx.EnsureDescsTogether(&d406, &d405)
		if d406.Loc == LocImm && d405.Loc == LocImm {
			d581 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d406.Imm.Int() == d405.Imm.Int())}
		} else if d405.Loc == LocImm {
			r27 := ctx.AllocRegExcept(d406.Reg)
			if d405.Imm.Int() >= -2147483648 && d405.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d406.Reg, int32(d405.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d405.Imm.Int()))
				ctx.EmitCmpInt64(d406.Reg, ctx.ScratchReg)
			}
			d581 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondEqual}
			ctx.BindReg(r27, &d581)
		} else if d406.Loc == LocImm {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d406.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d405.Reg)
			d581 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondEqual}
			ctx.BindReg(r28, &d581)
		} else {
			r29 := ctx.AllocRegExcept(d406.Reg)
			ctx.EmitCmpInt64(d406.Reg, d405.Reg)
			d581 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondEqual}
			ctx.BindReg(r29, &d581)
		}
		d582 = d581
		ctx.SyncDesc(&d582)
		if d582.Loc != LocImm && d582.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d582.Loc == LocImm {
			if d582.Imm.Bool() {
				return bbs[25].Render()
			}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
			return bbs[26].Render()
		}
		lbl34 := ctx.ReserveLabel()
		ctx.EmitJump(d582.Condition, lbl26)
		ctx.EmitJmp(lbl34)
		ctx.FreeDesc(&d581)
		snap583 := d1
		snap584 := d2
		snap585 := d3
		snap586 := d4
		snap587 := d5
		snap588 := d6
		snap589 := d7
		snap590 := d8
		snap591 := d18
		snap592 := d19
		snap593 := d20
		snap594 := d33
		snap595 := d34
		snap596 := d49
		snap597 := d50
		snap598 := d51
		snap599 := d69
		snap600 := d70
		snap601 := d71
		snap602 := d92
		snap603 := d93
		snap604 := d116
		snap605 := d117
		snap606 := d142
		snap607 := d143
		snap608 := d170
		snap609 := d171
		snap610 := d173
		snap611 := d174
		snap612 := d175
		snap613 := d207
		snap614 := d209
		snap615 := d210
		snap616 := d211
		snap617 := d247
		snap618 := d248
		snap619 := d250
		snap620 := d251
		snap621 := d252
		snap622 := d293
		snap623 := d294
		snap624 := d296
		snap625 := d297
		snap626 := d298
		snap627 := d344
		snap628 := d346
		snap629 := d347
		snap630 := d348
		snap631 := d350
		snap632 := d351
		snap633 := d352
		snap634 := d405
		snap635 := d406
		snap636 := d407
		snap637 := d408
		snap638 := d521
		snap639 := d522
		snap640 := d581
		snap641 := d582
		alloc642 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl34)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[26].PhiBase)+int32(0))
		ctx.EmitJmp(lbl27)
		ctx.RestoreAllocState(alloc642)
		d1 = snap583
		d2 = snap584
		d3 = snap585
		d4 = snap586
		d5 = snap587
		d6 = snap588
		d7 = snap589
		d8 = snap590
		d18 = snap591
		d19 = snap592
		d20 = snap593
		d33 = snap594
		d34 = snap595
		d49 = snap596
		d50 = snap597
		d51 = snap598
		d69 = snap599
		d70 = snap600
		d71 = snap601
		d92 = snap602
		d93 = snap603
		d116 = snap604
		d117 = snap605
		d142 = snap606
		d143 = snap607
		d170 = snap608
		d171 = snap609
		d173 = snap610
		d174 = snap611
		d175 = snap612
		d207 = snap613
		d209 = snap614
		d210 = snap615
		d211 = snap616
		d247 = snap617
		d248 = snap618
		d250 = snap619
		d251 = snap620
		d252 = snap621
		d293 = snap622
		d294 = snap623
		d296 = snap624
		d297 = snap625
		d298 = snap626
		d344 = snap627
		d346 = snap628
		d347 = snap629
		d348 = snap630
		d350 = snap631
		d351 = snap632
		d352 = snap633
		d405 = snap634
		d406 = snap635
		d407 = snap636
		d408 = snap637
		d521 = snap638
		d522 = snap639
		d581 = snap640
		d582 = snap641
		if !bbs[26].Rendered {
			snap643 := d1
			snap644 := d2
			snap645 := d3
			snap646 := d4
			snap647 := d5
			snap648 := d6
			snap649 := d7
			snap650 := d8
			snap651 := d18
			snap652 := d19
			snap653 := d20
			snap654 := d33
			snap655 := d34
			snap656 := d49
			snap657 := d50
			snap658 := d51
			snap659 := d69
			snap660 := d70
			snap661 := d71
			snap662 := d92
			snap663 := d93
			snap664 := d116
			snap665 := d117
			snap666 := d142
			snap667 := d143
			snap668 := d170
			snap669 := d171
			snap670 := d173
			snap671 := d174
			snap672 := d175
			snap673 := d207
			snap674 := d209
			snap675 := d210
			snap676 := d211
			snap677 := d247
			snap678 := d248
			snap679 := d250
			snap680 := d251
			snap681 := d252
			snap682 := d293
			snap683 := d294
			snap684 := d296
			snap685 := d297
			snap686 := d298
			snap687 := d344
			snap688 := d346
			snap689 := d347
			snap690 := d348
			snap691 := d350
			snap692 := d351
			snap693 := d352
			snap694 := d405
			snap695 := d406
			snap696 := d407
			snap697 := d408
			snap698 := d521
			snap699 := d522
			snap700 := d581
			snap701 := d582
			alloc702 := ctx.SnapshotAllocState()
			bbs[26].Render()
			ctx.RestoreAllocState(alloc702)
			d1 = snap643
			d2 = snap644
			d3 = snap645
			d4 = snap646
			d5 = snap647
			d6 = snap648
			d7 = snap649
			d8 = snap650
			d18 = snap651
			d19 = snap652
			d20 = snap653
			d33 = snap654
			d34 = snap655
			d49 = snap656
			d50 = snap657
			d51 = snap658
			d69 = snap659
			d70 = snap660
			d71 = snap661
			d92 = snap662
			d93 = snap663
			d116 = snap664
			d117 = snap665
			d142 = snap666
			d143 = snap667
			d170 = snap668
			d171 = snap669
			d173 = snap670
			d174 = snap671
			d175 = snap672
			d207 = snap673
			d209 = snap674
			d210 = snap675
			d211 = snap676
			d247 = snap677
			d248 = snap678
			d250 = snap679
			d251 = snap680
			d252 = snap681
			d293 = snap682
			d294 = snap683
			d296 = snap684
			d297 = snap685
			d298 = snap686
			d344 = snap687
			d346 = snap688
			d347 = snap689
			d348 = snap690
			d350 = snap691
			d351 = snap692
			d352 = snap693
			d405 = snap694
			d406 = snap695
			d407 = snap696
			d408 = snap697
			d521 = snap698
			d522 = snap699
			d581 = snap700
			d582 = snap701
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
		d703 = ctx.EmitGoCallScalar(jitGoABI_973b6d6b19b704fd, []JITValueDesc{args[1]}, 1)
		d703.NoHeapPointer = true
		ctx.BindReg(d703.Reg, &d703)
		ctx.EnsureDesc(&d703)
		if d703.Loc == LocImm {
			d704 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d703.Imm.Int() > 0)}
		} else {
			ctx.EmitCmpRegImm32(d703.Reg, 0)
			r30 := ctx.AllocRegExcept(d703.Reg)
			ctx.EmitSetcc(r30, CondSignedGreater)
			d704 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r30}
			ctx.BindReg(r30, &d704)
		}
		ctx.EnsureDesc(&d704)
		ctx.EmitStoreToStack(d704, int32(bbs[26].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d704)
		ctx.FreeDesc(&d703)
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
		d705 = d2
		if d705.Loc == LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d705)
		ctx.EmitStoreToStack(d705, int32(bbs[24].PhiBase)+int32(0))
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
			d706 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d706 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d706)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d706 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d706)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d706 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d706.Type = tagInt
			ctx.BindReg(d706.Reg, &d706)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d707 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d707 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d707)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d707 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d707)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d707 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d707.Type = tagInt
			ctx.BindReg(d707.Reg, &d707)
		}
		ctx.EnsureDesc(&d706)
		resultTarget708 := false
		_ = resultTarget708
		ctx.EnsureDesc(&d707)
		ctx.EnsureDescsTogether(&d706, &d707)
		if d706.Loc == LocImm && d707.Loc == LocImm {
			d709 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d706.Imm.Int() < d707.Imm.Int())}
		} else if d707.Loc == LocImm {
			r31 := ctx.AllocReg()
			if d707.Imm.Int() >= -2147483648 && d707.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d706.Reg, int32(d707.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d707.Imm.Int()))
				ctx.EmitCmpInt64(d706.Reg, ctx.ScratchReg)
			}
			d709 = ctx.DeferBooleanFlags(r31, CondSignedLess)
			ctx.BindReg(r31, &d709)
		} else if d706.Loc == LocImm {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d706.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d707.Reg)
			d709 = ctx.DeferBooleanFlags(r32, CondSignedLess)
			ctx.BindReg(r32, &d709)
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitCmpInt64(d706.Reg, d707.Reg)
			d709 = ctx.DeferBooleanFlags(r33, CondSignedLess)
			ctx.BindReg(r33, &d709)
		}
		ctx.FreeDesc(&d706)
		ctx.FreeDesc(&d707)
		ctx.SyncDesc(&d709)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d709) {
			return d709
		}
		ctx.EnsureDesc(&d709)
		ctx.EmitMovToReg(result.Reg, d709)
		result.Type = d709.Type
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
			d710 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d710 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d710)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d710 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d710)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d710 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d710.Type = tagInt
			ctx.BindReg(d710.Reg, &d710)
		}
		ctx.EnsureDesc(&d710)
		ctx.EnsureDesc(&d710)
		if d710.Loc == LocImm {
			d711 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d710.Imm.Int()))}
		} else {
			var r34 Reg
			r34 = d710.Reg
			d710.Loc = LocNone
			ctx.EmitInt64ToFloatBits(r34)
			d711 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r34}
			ctx.BindReg(r34, &d711)
		}
		ctx.FreeDesc(&d710)
		d712 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d711)
		resultTarget713 := false
		_ = resultTarget713
		ctx.EnsureDesc(&d712)
		ctx.EnsureDescsTogether(&d711, &d712)
		if d711.Loc == LocImm && d712.Loc == LocImm {
			d714 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d711.Imm.Float() < d712.Imm.Float())}
		} else if d712.Loc == LocImm {
			var r35 Reg
			if result.Loc == LocReg && result.Reg != d711.Reg {
				r35 = result.Reg
				resultTarget713 = true
			} else {
				r35 = ctx.AllocRegExcept(d711.Reg)
			}
			_, yBits := d712.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitCmpFloat64(ctx.ScratchReg, d711.Reg)
			d714 = ctx.DeferBooleanFlags(r35, CondUnsignedAbove)
			ctx.BindReg(r35, &d714)
		} else if d711.Loc == LocImm {
			var r36 Reg
			if result.Loc == LocReg && result.Reg != d712.Reg {
				r36 = result.Reg
				resultTarget713 = true
			} else {
				r36 = ctx.AllocRegExcept(d712.Reg)
			}
			_, xBits := d711.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, xBits)
			ctx.EmitCmpFloat64(d712.Reg, ctx.ScratchReg)
			d714 = ctx.DeferBooleanFlags(r36, CondUnsignedAbove)
			ctx.BindReg(r36, &d714)
		} else {
			var r37 Reg
			if result.Loc == LocReg && result.Reg != d711.Reg && result.Reg != d712.Reg {
				r37 = result.Reg
				resultTarget713 = true
			} else {
				r37 = ctx.AllocRegExcept(d711.Reg, d712.Reg)
			}
			ctx.EmitCmpFloat64(d712.Reg, d711.Reg)
			d714 = ctx.DeferBooleanFlags(r37, CondUnsignedAbove)
			ctx.BindReg(r37, &d714)
		}
		ctx.FreeDesc(&d711)
		ctx.FreeDesc(&d712)
		ctx.SyncDesc(&d714)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d714) {
			return d714
		}
		ctx.EnsureDesc(&d714)
		ctx.EmitMovToReg(result.Reg, d714)
		result.Type = d714.Type
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
			d715 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d715 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d715)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d715 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d715)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[0])
			d715 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d715.Type = tagInt
			ctx.BindReg(d715.Reg, &d715)
		}
		if args[1].Loc != LocImm {
			ctx.EnsureDesc(&args[1])
		}
		if args[1].Loc == LocImm {
			d716 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d716 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d716)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d716 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d716)
		} else {
			boxed := JITPrepareScmerGoArg(ctx, args[1])
			d716 = ctx.EmitGoCallScalar(jitGoABI_5bd1d0f41a4e4914, []JITValueDesc{boxed}, 1)
			d716.Type = tagInt
			ctx.BindReg(d716.Reg, &d716)
		}
		ctx.EnsureDesc(&d715)
		resultTarget717 := false
		_ = resultTarget717
		ctx.EnsureDesc(&d716)
		ctx.EnsureDescsTogether(&d715, &d716)
		if d715.Loc == LocImm && d716.Loc == LocImm {
			d718 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d715.Imm.Int() < d716.Imm.Int())}
		} else if d716.Loc == LocImm {
			r38 := ctx.AllocReg()
			if d716.Imm.Int() >= -2147483648 && d716.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d715.Reg, int32(d716.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d716.Imm.Int()))
				ctx.EmitCmpInt64(d715.Reg, ctx.ScratchReg)
			}
			d718 = ctx.DeferBooleanFlags(r38, CondSignedLess)
			ctx.BindReg(r38, &d718)
		} else if d715.Loc == LocImm {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d715.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d716.Reg)
			d718 = ctx.DeferBooleanFlags(r39, CondSignedLess)
			ctx.BindReg(r39, &d718)
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitCmpInt64(d715.Reg, d716.Reg)
			d718 = ctx.DeferBooleanFlags(r40, CondSignedLess)
			ctx.BindReg(r40, &d718)
		}
		ctx.FreeDesc(&d715)
		ctx.FreeDesc(&d716)
		ctx.SyncDesc(&d718)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d718) {
			return d718
		}
		ctx.EnsureDesc(&d718)
		ctx.EmitMovToReg(result.Reg, d718)
		result.Type = d718.Type
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
			d719 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0x5))}
		} else {
			r41 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpRegImm32(d4.Reg, 5)
			d719 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r41, Condition: CondEqual}
			ctx.BindReg(r41, &d719)
		}
		d720 = d719
		ctx.SyncDesc(&d720)
		if d720.Loc != LocImm && d720.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d720.Loc == LocImm {
			if d720.Imm.Bool() {
				return bbs[29].Render()
			}
			return bbs[31].Render()
		}
		ctx.EmitJump(d720.Condition, lbl30)
		if bbs[31].Rendered {
			ctx.EmitJmp(lbl32)
		}
		ctx.FreeDesc(&d719)
		ctx.FlushRegisterMoves()
		if !bbs[31].Rendered {
			snap721 := d1
			snap722 := d2
			snap723 := d3
			snap724 := d4
			snap725 := d5
			snap726 := d6
			snap727 := d7
			snap728 := d8
			snap729 := d18
			snap730 := d19
			snap731 := d20
			snap732 := d33
			snap733 := d34
			snap734 := d49
			snap735 := d50
			snap736 := d51
			snap737 := d69
			snap738 := d70
			snap739 := d71
			snap740 := d92
			snap741 := d93
			snap742 := d116
			snap743 := d117
			snap744 := d142
			snap745 := d143
			snap746 := d170
			snap747 := d171
			snap748 := d173
			snap749 := d174
			snap750 := d175
			snap751 := d207
			snap752 := d209
			snap753 := d210
			snap754 := d211
			snap755 := d247
			snap756 := d248
			snap757 := d250
			snap758 := d251
			snap759 := d252
			snap760 := d293
			snap761 := d294
			snap762 := d296
			snap763 := d297
			snap764 := d298
			snap765 := d344
			snap766 := d346
			snap767 := d347
			snap768 := d348
			snap769 := d350
			snap770 := d351
			snap771 := d352
			snap772 := d405
			snap773 := d406
			snap774 := d407
			snap775 := d408
			snap776 := d521
			snap777 := d522
			snap778 := d581
			snap779 := d582
			snap780 := d703
			snap781 := d704
			snap782 := d705
			snap783 := d706
			snap784 := d707
			snap785 := d709
			snap786 := d710
			snap787 := d711
			snap788 := d712
			snap789 := d714
			snap790 := d715
			snap791 := d716
			snap792 := d718
			snap793 := d719
			snap794 := d720
			alloc795 := ctx.SnapshotAllocState()
			bbs[31].Render()
			ctx.RestoreAllocState(alloc795)
			d1 = snap721
			d2 = snap722
			d3 = snap723
			d4 = snap724
			d5 = snap725
			d6 = snap726
			d7 = snap727
			d8 = snap728
			d18 = snap729
			d19 = snap730
			d20 = snap731
			d33 = snap732
			d34 = snap733
			d49 = snap734
			d50 = snap735
			d51 = snap736
			d69 = snap737
			d70 = snap738
			d71 = snap739
			d92 = snap740
			d93 = snap741
			d116 = snap742
			d117 = snap743
			d142 = snap744
			d143 = snap745
			d170 = snap746
			d171 = snap747
			d173 = snap748
			d174 = snap749
			d175 = snap750
			d207 = snap751
			d209 = snap752
			d210 = snap753
			d211 = snap754
			d247 = snap755
			d248 = snap756
			d250 = snap757
			d251 = snap758
			d252 = snap759
			d293 = snap760
			d294 = snap761
			d296 = snap762
			d297 = snap763
			d298 = snap764
			d344 = snap765
			d346 = snap766
			d347 = snap767
			d348 = snap768
			d350 = snap769
			d351 = snap770
			d352 = snap771
			d405 = snap772
			d406 = snap773
			d407 = snap774
			d408 = snap775
			d521 = snap776
			d522 = snap777
			d581 = snap778
			d582 = snap779
			d703 = snap780
			d704 = snap781
			d705 = snap782
			d706 = snap783
			d707 = snap784
			d709 = snap785
			d710 = snap786
			d711 = snap787
			d712 = snap788
			d714 = snap789
			d715 = snap790
			d716 = snap791
			d718 = snap792
			d719 = snap793
			d720 = snap794
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
		d796 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d4, d6}, 1)
		d796.NoHeapPointer = true
		ctx.EmitAndRegImm32(d796.Reg, 1)
		d796.Type = tagBool
		ctx.BindReg(d796.Reg, &d796)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.SyncDesc(&d796)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d796) {
			return d796
		}
		ctx.EnsureDesc(&d796)
		ctx.EmitMovToReg(result.Reg, d796)
		result.Type = d796.Type
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
var jitGoABI_7b4abd9531444504 uint64
var jitGoABI_973b6d6b19b704fd uint64
var jitGoABI_d69f4c85104836f4 uint64

func init() {
	jitGoABI_0cbff6f55ee96f6d = GoFuncAddr(Scmer.String)
	jitGoABI_58434e8dfe39c263 = GoFuncAddr(ParseDateValue)
	jitGoABI_5bd1d0f41a4e4914 = GoFuncAddr(Scmer.Int)
	jitGoABI_7b4abd9531444504 = GoFuncAddr(nativeDateSeconds)
	jitGoABI_973b6d6b19b704fd = GoFuncAddr((Scmer).DateNanosecond)
	jitGoABI_d69f4c85104836f4 = GoFuncAddr(compareDates)
}
