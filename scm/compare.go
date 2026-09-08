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

import (
	"strings"
	"unsafe"
)

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

// cstringIsNibble reports whether the format field of a tagCString aux value
// uses 4-bit nibble packing (one nibble per character).
// Must stay in sync with storage.StringFormat constants:
//
//	1=Phone, 2=HexLower, 3=HexUpper, 8=Decimal, 9=DateTime, 10=PhoneDTMF
func cstringIsNibble(format uint8) bool {
	return format == 1 || format == 2 || format == 3 || format == 8 || format == 9 || format == 10
}

// cstringEqual compares two tagCString Scmers without materializing strings.
// Algorithm: len≠len → false; fmt≠fmt → materialize; fmt=fmt → nibble or byte compare.
func cstringEqual(a, b Scmer) bool {
	aVal := auxVal(a.aux)
	bVal := auxVal(b.aux)
	aCharLen := int(aVal & ((1 << 43) - 1))
	bCharLen := int(bVal & ((1 << 43) - 1))
	if aCharLen != bCharLen {
		return false
	}
	aFmt := uint8(aVal >> 44)
	bFmt := uint8(bVal >> 44)
	if aFmt != bFmt {
		return a.String() == b.String() // different formats, materialize
	}
	aNibOff := int((aVal >> 43) & 1)
	bNibOff := int((bVal >> 43) & 1)
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
			aLen := int(auxVal(a.aux) & ((1 << 47) - 1))
			bLen := int(auxVal(b.aux) & ((1 << 47) - 1))
			if aLen != bLen {
				return false
			}
			return unsafe.String(a.ptr, aLen) == unsafe.String(b.ptr, bLen)
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
		return a.String() == b.String()
	case tagCString:
		return a.String() == b.String()
	case tagBString:
		if tb == tagBString {
			aLen := int(auxVal(a.aux) & ((1 << 47) - 1))
			bLen := int(auxVal(b.aux) & ((1 << 47) - 1))
			if aLen != bLen {
				return false
			}
			return unsafe.String(a.ptr, aLen) == unsafe.String(b.ptr, bLen)
		}
		return a.String() == b.String()
	case tagBSON:
		if tb == tagBSON {
			return bsonRawEqual(bsonRawValue(a), bsonRawValue(b))
		}
		return a.String() == b.String()
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

	return a.String() == b.String()
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
			return NewBool(strings.EqualFold(a.String(), b.String()))
		case tagCString:
			return NewBool(cstringEqual(a, b))
		case tagBString:
			aLen := int(auxVal(a.aux) & ((1 << 47) - 1))
			bLen := int(auxVal(b.aux) & ((1 << 47) - 1))
			if aLen != bLen {
				return NewBool(false)
			}
			return NewBool(unsafe.String(a.ptr, aLen) == unsafe.String(b.ptr, bLen))
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
		return NewBool(strings.EqualFold(a.String(), b.String()))
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
		if tb == tagBString {
			aLen := int(auxVal(a.aux) & ((1 << 47) - 1))
			bLen := int(auxVal(b.aux) & ((1 << 47) - 1))
			if aLen != bLen {
				return NewBool(false)
			}
			return NewBool(unsafe.String(a.ptr, aLen) == unsafe.String(b.ptr, bLen))
		}
		return NewBool(strings.EqualFold(a.String(), b.String()))
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

	return NewBool(strings.EqualFold(a.String(), b.String()))
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
	var d24 JITValueDesc
	_ = d24
	var d25 JITValueDesc
	_ = d25
	var d26 JITValueDesc
	_ = d26
	var d51 JITValueDesc
	_ = d51
	var d52 JITValueDesc
	_ = d52
	var d81 JITValueDesc
	_ = d81
	var d82 JITValueDesc
	_ = d82
	var d83 JITValueDesc
	_ = d83
	var d118 JITValueDesc
	_ = d118
	var d119 JITValueDesc
	_ = d119
	var d120 JITValueDesc
	_ = d120
	var d161 JITValueDesc
	_ = d161
	var d162 JITValueDesc
	_ = d162
	var d207 JITValueDesc
	_ = d207
	var d208 JITValueDesc
	_ = d208
	var d209 JITValueDesc
	_ = d209
	var d211 JITValueDesc
	_ = d211
	var d212 JITValueDesc
	_ = d212
	var d213 JITValueDesc
	_ = d213
	var d270 JITValueDesc
	_ = d270
	var d271 JITValueDesc
	_ = d271
	var d273 JITValueDesc
	_ = d273
	var d274 JITValueDesc
	_ = d274
	var d275 JITValueDesc
	_ = d275
	var d342 JITValueDesc
	_ = d342
	var d343 JITValueDesc
	_ = d343
	var d344 JITValueDesc
	_ = d344
	var d346 JITValueDesc
	_ = d346
	var d347 JITValueDesc
	_ = d347
	var d348 JITValueDesc
	_ = d348
	var d427 JITValueDesc
	_ = d427
	var d429 JITValueDesc
	_ = d429
	var d430 JITValueDesc
	_ = d430
	var d431 JITValueDesc
	_ = d431
	var d433 JITValueDesc
	_ = d433
	var d434 JITValueDesc
	_ = d434
	var d435 JITValueDesc
	_ = d435
	var d528 JITValueDesc
	_ = d528
	var d529 JITValueDesc
	_ = d529
	var d531 JITValueDesc
	_ = d531
	var d532 JITValueDesc
	_ = d532
	var d533 JITValueDesc
	_ = d533
	var d636 JITValueDesc
	_ = d636
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
	var bbs [20]BBDescriptor
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
			ctx.FlushRegisterMoves()
			bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_0 = bbs[0].Address
			ctx.MarkLabel(lbl1)
			ctx.ResolveFixups()
		}
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
		var d4 JITValueDesc
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
				return bbs[3].RenderPS(ps6)
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
		ctx.EmitJump(d5.Condition, lbl4)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d4)
		snap8 := d0
		snap9 := d1
		snap10 := d2
		snap11 := d3
		snap12 := d4
		snap13 := d5
		alloc14 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc14)
		d0 = snap8
		d1 = snap9
		d2 = snap10
		d3 = snap11
		d4 = snap12
		d5 = snap13
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
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps15)
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
			ctx.FlushRegisterMoves()
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
		d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		ctx.EmitMovToReg(result.Reg, d24)
		result.Type = d24.Type
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d25 JITValueDesc
		if d1.Loc == LocImm {
			d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x0))}
		} else {
			r1 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 0)
			d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
			ctx.BindReg(r1, &d25)
		}
		d26 = d25
		ctx.EnsureDesc(&d26)
		if d26.Loc != LocImm && d26.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
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
				ps27.OverlayValues[4] = d4
				ps27.OverlayValues[5] = d5
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
			ps28.OverlayValues[4] = d4
			ps28.OverlayValues[5] = d5
			ps28.OverlayValues[24] = d24
			ps28.OverlayValues[25] = d25
			ps28.OverlayValues[26] = d26
			return bbs[5].RenderPS(ps28)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d26.Condition, lbl5)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d25)
		snap29 := d0
		snap30 := d1
		snap31 := d2
		snap32 := d3
		snap33 := d4
		snap34 := d5
		snap35 := d24
		snap36 := d25
		snap37 := d26
		alloc38 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc38)
		d0 = snap29
		d1 = snap30
		d2 = snap31
		d3 = snap32
		d4 = snap33
		d5 = snap34
		d24 = snap35
		d25 = snap36
		d26 = snap37
		ctx.RestoreAllocState(alloc38)
		d0 = snap29
		d1 = snap30
		d2 = snap31
		d3 = snap32
		d4 = snap33
		d5 = snap34
		d24 = snap35
		d25 = snap36
		d26 = snap37
		ps39 := PhiState{General: true}
		ps39.OverlayValues = make([]JITValueDesc, 27)
		ps39.OverlayValues[0] = d0
		ps39.OverlayValues[1] = d1
		ps39.OverlayValues[2] = d2
		ps39.OverlayValues[3] = d3
		ps39.OverlayValues[4] = d4
		ps39.OverlayValues[5] = d5
		ps39.OverlayValues[24] = d24
		ps39.OverlayValues[25] = d25
		ps39.OverlayValues[26] = d26
		ps40 := PhiState{General: true}
		ps40.OverlayValues = make([]JITValueDesc, 27)
		ps40.OverlayValues[0] = d0
		ps40.OverlayValues[1] = d1
		ps40.OverlayValues[2] = d2
		ps40.OverlayValues[3] = d3
		ps40.OverlayValues[4] = d4
		ps40.OverlayValues[5] = d5
		ps40.OverlayValues[24] = d24
		ps40.OverlayValues[25] = d25
		ps40.OverlayValues[26] = d26
		snap41 := d0
		snap42 := d1
		snap43 := d2
		snap44 := d3
		snap45 := d4
		snap46 := d5
		snap47 := d24
		snap48 := d25
		snap49 := d26
		alloc50 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps40)
		}
		ctx.RestoreAllocState(alloc50)
		d0 = snap41
		d1 = snap42
		d2 = snap43
		d3 = snap44
		d4 = snap45
		d5 = snap46
		d24 = snap47
		d25 = snap48
		d26 = snap49
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps39)
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
			ctx.FlushRegisterMoves()
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
		ctx.EnsureDesc(&d3)
		var d51 JITValueDesc
		if d3.Loc == LocImm {
			d51 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x0))}
		} else {
			r2 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			d51 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
			ctx.BindReg(r2, &d51)
		}
		d52 = d51
		ctx.EnsureDesc(&d52)
		if d52.Loc != LocImm && d52.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d52.Loc == LocImm {
			if d52.Imm.Bool() {
				if ps.General {
				}
				ps53 := PhiState{General: ps.General}
				ps53.OverlayValues = make([]JITValueDesc, 53)
				ps53.OverlayValues[0] = d0
				ps53.OverlayValues[1] = d1
				ps53.OverlayValues[2] = d2
				ps53.OverlayValues[3] = d3
				ps53.OverlayValues[4] = d4
				ps53.OverlayValues[5] = d5
				ps53.OverlayValues[24] = d24
				ps53.OverlayValues[25] = d25
				ps53.OverlayValues[26] = d26
				ps53.OverlayValues[51] = d51
				ps53.OverlayValues[52] = d52
				return bbs[1].RenderPS(ps53)
			}
			if ps.General {
			}
			ps54 := PhiState{General: ps.General}
			ps54.OverlayValues = make([]JITValueDesc, 53)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[5] = d5
			ps54.OverlayValues[24] = d24
			ps54.OverlayValues[25] = d25
			ps54.OverlayValues[26] = d26
			ps54.OverlayValues[51] = d51
			ps54.OverlayValues[52] = d52
			return bbs[2].RenderPS(ps54)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		ctx.EmitJump(d52.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d51)
		snap55 := d0
		snap56 := d1
		snap57 := d2
		snap58 := d3
		snap59 := d4
		snap60 := d5
		snap61 := d24
		snap62 := d25
		snap63 := d26
		snap64 := d51
		snap65 := d52
		alloc66 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc66)
		d0 = snap55
		d1 = snap56
		d2 = snap57
		d3 = snap58
		d4 = snap59
		d5 = snap60
		d24 = snap61
		d25 = snap62
		d26 = snap63
		d51 = snap64
		d52 = snap65
		ctx.RestoreAllocState(alloc66)
		d0 = snap55
		d1 = snap56
		d2 = snap57
		d3 = snap58
		d4 = snap59
		d5 = snap60
		d24 = snap61
		d25 = snap62
		d26 = snap63
		d51 = snap64
		d52 = snap65
		ps67 := PhiState{General: true}
		ps67.OverlayValues = make([]JITValueDesc, 53)
		ps67.OverlayValues[0] = d0
		ps67.OverlayValues[1] = d1
		ps67.OverlayValues[2] = d2
		ps67.OverlayValues[3] = d3
		ps67.OverlayValues[4] = d4
		ps67.OverlayValues[5] = d5
		ps67.OverlayValues[24] = d24
		ps67.OverlayValues[25] = d25
		ps67.OverlayValues[26] = d26
		ps67.OverlayValues[51] = d51
		ps67.OverlayValues[52] = d52
		ps68 := PhiState{General: true}
		ps68.OverlayValues = make([]JITValueDesc, 53)
		ps68.OverlayValues[0] = d0
		ps68.OverlayValues[1] = d1
		ps68.OverlayValues[2] = d2
		ps68.OverlayValues[3] = d3
		ps68.OverlayValues[4] = d4
		ps68.OverlayValues[5] = d5
		ps68.OverlayValues[24] = d24
		ps68.OverlayValues[25] = d25
		ps68.OverlayValues[26] = d26
		ps68.OverlayValues[51] = d51
		ps68.OverlayValues[52] = d52
		snap69 := d0
		snap70 := d1
		snap71 := d2
		snap72 := d3
		snap73 := d4
		snap74 := d5
		snap75 := d24
		snap76 := d25
		snap77 := d26
		snap78 := d51
		snap79 := d52
		alloc80 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps68)
		}
		ctx.RestoreAllocState(alloc80)
		d0 = snap69
		d1 = snap70
		d2 = snap71
		d3 = snap72
		d4 = snap73
		d5 = snap74
		d24 = snap75
		d25 = snap76
		d26 = snap77
		d51 = snap78
		d52 = snap79
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps67)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
		}
		ctx.ReclaimUntrackedRegs()
		d81 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
		ctx.EmitMovToReg(result.Reg, d81)
		result.Type = d81.Type
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
			d81 = ps.OverlayValues[81]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		var d82 JITValueDesc
		if d3.Loc == LocImm {
			d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x0))}
		} else {
			r3 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			d82 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondEqual}
			ctx.BindReg(r3, &d82)
		}
		d83 = d82
		ctx.EnsureDesc(&d83)
		if d83.Loc != LocImm && d83.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d83.Loc == LocImm {
			if d83.Imm.Bool() {
				if ps.General {
				}
				ps84 := PhiState{General: ps.General}
				ps84.OverlayValues = make([]JITValueDesc, 84)
				ps84.OverlayValues[0] = d0
				ps84.OverlayValues[1] = d1
				ps84.OverlayValues[2] = d2
				ps84.OverlayValues[3] = d3
				ps84.OverlayValues[4] = d4
				ps84.OverlayValues[5] = d5
				ps84.OverlayValues[24] = d24
				ps84.OverlayValues[25] = d25
				ps84.OverlayValues[26] = d26
				ps84.OverlayValues[51] = d51
				ps84.OverlayValues[52] = d52
				ps84.OverlayValues[81] = d81
				ps84.OverlayValues[82] = d82
				ps84.OverlayValues[83] = d83
				return bbs[6].RenderPS(ps84)
			}
			if ps.General {
			}
			ps85 := PhiState{General: ps.General}
			ps85.OverlayValues = make([]JITValueDesc, 84)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[1] = d1
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[24] = d24
			ps85.OverlayValues[25] = d25
			ps85.OverlayValues[26] = d26
			ps85.OverlayValues[51] = d51
			ps85.OverlayValues[52] = d52
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			return bbs[7].RenderPS(ps85)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		ctx.EmitJump(d83.Condition, lbl7)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d82)
		snap86 := d0
		snap87 := d1
		snap88 := d2
		snap89 := d3
		snap90 := d4
		snap91 := d5
		snap92 := d24
		snap93 := d25
		snap94 := d26
		snap95 := d51
		snap96 := d52
		snap97 := d81
		snap98 := d82
		snap99 := d83
		alloc100 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc100)
		d0 = snap86
		d1 = snap87
		d2 = snap88
		d3 = snap89
		d4 = snap90
		d5 = snap91
		d24 = snap92
		d25 = snap93
		d26 = snap94
		d51 = snap95
		d52 = snap96
		d81 = snap97
		d82 = snap98
		d83 = snap99
		ctx.RestoreAllocState(alloc100)
		d0 = snap86
		d1 = snap87
		d2 = snap88
		d3 = snap89
		d4 = snap90
		d5 = snap91
		d24 = snap92
		d25 = snap93
		d26 = snap94
		d51 = snap95
		d52 = snap96
		d81 = snap97
		d82 = snap98
		d83 = snap99
		ps101 := PhiState{General: true}
		ps101.OverlayValues = make([]JITValueDesc, 84)
		ps101.OverlayValues[0] = d0
		ps101.OverlayValues[1] = d1
		ps101.OverlayValues[2] = d2
		ps101.OverlayValues[3] = d3
		ps101.OverlayValues[4] = d4
		ps101.OverlayValues[5] = d5
		ps101.OverlayValues[24] = d24
		ps101.OverlayValues[25] = d25
		ps101.OverlayValues[26] = d26
		ps101.OverlayValues[51] = d51
		ps101.OverlayValues[52] = d52
		ps101.OverlayValues[81] = d81
		ps101.OverlayValues[82] = d82
		ps101.OverlayValues[83] = d83
		ps102 := PhiState{General: true}
		ps102.OverlayValues = make([]JITValueDesc, 84)
		ps102.OverlayValues[0] = d0
		ps102.OverlayValues[1] = d1
		ps102.OverlayValues[2] = d2
		ps102.OverlayValues[3] = d3
		ps102.OverlayValues[4] = d4
		ps102.OverlayValues[5] = d5
		ps102.OverlayValues[24] = d24
		ps102.OverlayValues[25] = d25
		ps102.OverlayValues[26] = d26
		ps102.OverlayValues[51] = d51
		ps102.OverlayValues[52] = d52
		ps102.OverlayValues[81] = d81
		ps102.OverlayValues[82] = d82
		ps102.OverlayValues[83] = d83
		snap103 := d0
		snap104 := d1
		snap105 := d2
		snap106 := d3
		snap107 := d4
		snap108 := d5
		snap109 := d24
		snap110 := d25
		snap111 := d26
		snap112 := d51
		snap113 := d52
		snap114 := d81
		snap115 := d82
		snap116 := d83
		alloc117 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps102)
		}
		ctx.RestoreAllocState(alloc117)
		d0 = snap103
		d1 = snap104
		d2 = snap105
		d3 = snap106
		d4 = snap107
		d5 = snap108
		d24 = snap109
		d25 = snap110
		d26 = snap111
		d51 = snap112
		d52 = snap113
		d81 = snap114
		d82 = snap115
		d83 = snap116
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps101)
		}
		return result
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
			d3 = ps.OverlayValues[3]
		}
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
			d5 = ps.OverlayValues[5]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		ctx.ReclaimUntrackedRegs()
		d118 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		ctx.EmitMovToReg(result.Reg, d118)
		result.Type = d118.Type
		ctx.EmitJmp(lbl0)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
			d3 = ps.OverlayValues[3]
		}
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
			d5 = ps.OverlayValues[5]
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
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d119 JITValueDesc
		if d1.Loc == LocImm {
			d119 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x10))}
		} else {
			r4 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 16)
			d119 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
			ctx.BindReg(r4, &d119)
		}
		d120 = d119
		ctx.EnsureDesc(&d120)
		if d120.Loc != LocImm && d120.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d120.Loc == LocImm {
			if d120.Imm.Bool() {
				if ps.General {
				}
				ps121 := PhiState{General: ps.General}
				ps121.OverlayValues = make([]JITValueDesc, 121)
				ps121.OverlayValues[0] = d0
				ps121.OverlayValues[1] = d1
				ps121.OverlayValues[2] = d2
				ps121.OverlayValues[3] = d3
				ps121.OverlayValues[4] = d4
				ps121.OverlayValues[5] = d5
				ps121.OverlayValues[24] = d24
				ps121.OverlayValues[25] = d25
				ps121.OverlayValues[26] = d26
				ps121.OverlayValues[51] = d51
				ps121.OverlayValues[52] = d52
				ps121.OverlayValues[81] = d81
				ps121.OverlayValues[82] = d82
				ps121.OverlayValues[83] = d83
				ps121.OverlayValues[118] = d118
				ps121.OverlayValues[119] = d119
				ps121.OverlayValues[120] = d120
				return bbs[8].RenderPS(ps121)
			}
			if ps.General {
			}
			ps122 := PhiState{General: ps.General}
			ps122.OverlayValues = make([]JITValueDesc, 121)
			ps122.OverlayValues[0] = d0
			ps122.OverlayValues[1] = d1
			ps122.OverlayValues[2] = d2
			ps122.OverlayValues[3] = d3
			ps122.OverlayValues[4] = d4
			ps122.OverlayValues[5] = d5
			ps122.OverlayValues[24] = d24
			ps122.OverlayValues[25] = d25
			ps122.OverlayValues[26] = d26
			ps122.OverlayValues[51] = d51
			ps122.OverlayValues[52] = d52
			ps122.OverlayValues[81] = d81
			ps122.OverlayValues[82] = d82
			ps122.OverlayValues[83] = d83
			ps122.OverlayValues[118] = d118
			ps122.OverlayValues[119] = d119
			ps122.OverlayValues[120] = d120
			return bbs[10].RenderPS(ps122)
		}
		if !ps.General {
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		ctx.EmitJump(d120.Condition, lbl9)
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
		}
		ctx.FreeDesc(&d119)
		snap123 := d0
		snap124 := d1
		snap125 := d2
		snap126 := d3
		snap127 := d4
		snap128 := d5
		snap129 := d24
		snap130 := d25
		snap131 := d26
		snap132 := d51
		snap133 := d52
		snap134 := d81
		snap135 := d82
		snap136 := d83
		snap137 := d118
		snap138 := d119
		snap139 := d120
		alloc140 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc140)
		d0 = snap123
		d1 = snap124
		d2 = snap125
		d3 = snap126
		d4 = snap127
		d5 = snap128
		d24 = snap129
		d25 = snap130
		d26 = snap131
		d51 = snap132
		d52 = snap133
		d81 = snap134
		d82 = snap135
		d83 = snap136
		d118 = snap137
		d119 = snap138
		d120 = snap139
		ctx.RestoreAllocState(alloc140)
		d0 = snap123
		d1 = snap124
		d2 = snap125
		d3 = snap126
		d4 = snap127
		d5 = snap128
		d24 = snap129
		d25 = snap130
		d26 = snap131
		d51 = snap132
		d52 = snap133
		d81 = snap134
		d82 = snap135
		d83 = snap136
		d118 = snap137
		d119 = snap138
		d120 = snap139
		ps141 := PhiState{General: true}
		ps141.OverlayValues = make([]JITValueDesc, 121)
		ps141.OverlayValues[0] = d0
		ps141.OverlayValues[1] = d1
		ps141.OverlayValues[2] = d2
		ps141.OverlayValues[3] = d3
		ps141.OverlayValues[4] = d4
		ps141.OverlayValues[5] = d5
		ps141.OverlayValues[24] = d24
		ps141.OverlayValues[25] = d25
		ps141.OverlayValues[26] = d26
		ps141.OverlayValues[51] = d51
		ps141.OverlayValues[52] = d52
		ps141.OverlayValues[81] = d81
		ps141.OverlayValues[82] = d82
		ps141.OverlayValues[83] = d83
		ps141.OverlayValues[118] = d118
		ps141.OverlayValues[119] = d119
		ps141.OverlayValues[120] = d120
		ps142 := PhiState{General: true}
		ps142.OverlayValues = make([]JITValueDesc, 121)
		ps142.OverlayValues[0] = d0
		ps142.OverlayValues[1] = d1
		ps142.OverlayValues[2] = d2
		ps142.OverlayValues[3] = d3
		ps142.OverlayValues[4] = d4
		ps142.OverlayValues[5] = d5
		ps142.OverlayValues[24] = d24
		ps142.OverlayValues[25] = d25
		ps142.OverlayValues[26] = d26
		ps142.OverlayValues[51] = d51
		ps142.OverlayValues[52] = d52
		ps142.OverlayValues[81] = d81
		ps142.OverlayValues[82] = d82
		ps142.OverlayValues[83] = d83
		ps142.OverlayValues[118] = d118
		ps142.OverlayValues[119] = d119
		ps142.OverlayValues[120] = d120
		snap143 := d0
		snap144 := d1
		snap145 := d2
		snap146 := d3
		snap147 := d4
		snap148 := d5
		snap149 := d24
		snap150 := d25
		snap151 := d26
		snap152 := d51
		snap153 := d52
		snap154 := d81
		snap155 := d82
		snap156 := d83
		snap157 := d118
		snap158 := d119
		snap159 := d120
		alloc160 := ctx.SnapshotAllocState()
		if !bbs[10].Rendered {
			bbs[10].RenderPS(ps142)
		}
		ctx.RestoreAllocState(alloc160)
		d0 = snap143
		d1 = snap144
		d2 = snap145
		d3 = snap146
		d4 = snap147
		d5 = snap148
		d24 = snap149
		d25 = snap150
		d26 = snap151
		d51 = snap152
		d52 = snap153
		d81 = snap154
		d82 = snap155
		d83 = snap156
		d118 = snap157
		d119 = snap158
		d120 = snap159
		if !bbs[8].Rendered {
			return bbs[8].RenderPS(ps141)
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
			ctx.FlushRegisterMoves()
			bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_8 = bbs[8].Address
			ctx.MarkLabel(lbl9)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		var d161 JITValueDesc
		if d3.Loc == LocImm {
			d161 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x1))}
		} else {
			r5 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 1)
			d161 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
			ctx.BindReg(r5, &d161)
		}
		d162 = d161
		ctx.EnsureDesc(&d162)
		if d162.Loc != LocImm && d162.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d162.Loc == LocImm {
			if d162.Imm.Bool() {
				if ps.General {
				}
				ps163 := PhiState{General: ps.General}
				ps163.OverlayValues = make([]JITValueDesc, 163)
				ps163.OverlayValues[0] = d0
				ps163.OverlayValues[1] = d1
				ps163.OverlayValues[2] = d2
				ps163.OverlayValues[3] = d3
				ps163.OverlayValues[4] = d4
				ps163.OverlayValues[5] = d5
				ps163.OverlayValues[24] = d24
				ps163.OverlayValues[25] = d25
				ps163.OverlayValues[26] = d26
				ps163.OverlayValues[51] = d51
				ps163.OverlayValues[52] = d52
				ps163.OverlayValues[81] = d81
				ps163.OverlayValues[82] = d82
				ps163.OverlayValues[83] = d83
				ps163.OverlayValues[118] = d118
				ps163.OverlayValues[119] = d119
				ps163.OverlayValues[120] = d120
				ps163.OverlayValues[161] = d161
				ps163.OverlayValues[162] = d162
				return bbs[11].RenderPS(ps163)
			}
			if ps.General {
			}
			ps164 := PhiState{General: ps.General}
			ps164.OverlayValues = make([]JITValueDesc, 163)
			ps164.OverlayValues[0] = d0
			ps164.OverlayValues[1] = d1
			ps164.OverlayValues[2] = d2
			ps164.OverlayValues[3] = d3
			ps164.OverlayValues[4] = d4
			ps164.OverlayValues[5] = d5
			ps164.OverlayValues[24] = d24
			ps164.OverlayValues[25] = d25
			ps164.OverlayValues[26] = d26
			ps164.OverlayValues[51] = d51
			ps164.OverlayValues[52] = d52
			ps164.OverlayValues[81] = d81
			ps164.OverlayValues[82] = d82
			ps164.OverlayValues[83] = d83
			ps164.OverlayValues[118] = d118
			ps164.OverlayValues[119] = d119
			ps164.OverlayValues[120] = d120
			ps164.OverlayValues[161] = d161
			ps164.OverlayValues[162] = d162
			return bbs[13].RenderPS(ps164)
		}
		if !ps.General {
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		ctx.EmitJump(d162.Condition, lbl12)
		if bbs[13].Rendered {
			ctx.EmitJmp(lbl14)
		}
		ctx.FreeDesc(&d161)
		snap165 := d0
		snap166 := d1
		snap167 := d2
		snap168 := d3
		snap169 := d4
		snap170 := d5
		snap171 := d24
		snap172 := d25
		snap173 := d26
		snap174 := d51
		snap175 := d52
		snap176 := d81
		snap177 := d82
		snap178 := d83
		snap179 := d118
		snap180 := d119
		snap181 := d120
		snap182 := d161
		snap183 := d162
		alloc184 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc184)
		d0 = snap165
		d1 = snap166
		d2 = snap167
		d3 = snap168
		d4 = snap169
		d5 = snap170
		d24 = snap171
		d25 = snap172
		d26 = snap173
		d51 = snap174
		d52 = snap175
		d81 = snap176
		d82 = snap177
		d83 = snap178
		d118 = snap179
		d119 = snap180
		d120 = snap181
		d161 = snap182
		d162 = snap183
		ctx.RestoreAllocState(alloc184)
		d0 = snap165
		d1 = snap166
		d2 = snap167
		d3 = snap168
		d4 = snap169
		d5 = snap170
		d24 = snap171
		d25 = snap172
		d26 = snap173
		d51 = snap174
		d52 = snap175
		d81 = snap176
		d82 = snap177
		d83 = snap178
		d118 = snap179
		d119 = snap180
		d120 = snap181
		d161 = snap182
		d162 = snap183
		ps185 := PhiState{General: true}
		ps185.OverlayValues = make([]JITValueDesc, 163)
		ps185.OverlayValues[0] = d0
		ps185.OverlayValues[1] = d1
		ps185.OverlayValues[2] = d2
		ps185.OverlayValues[3] = d3
		ps185.OverlayValues[4] = d4
		ps185.OverlayValues[5] = d5
		ps185.OverlayValues[24] = d24
		ps185.OverlayValues[25] = d25
		ps185.OverlayValues[26] = d26
		ps185.OverlayValues[51] = d51
		ps185.OverlayValues[52] = d52
		ps185.OverlayValues[81] = d81
		ps185.OverlayValues[82] = d82
		ps185.OverlayValues[83] = d83
		ps185.OverlayValues[118] = d118
		ps185.OverlayValues[119] = d119
		ps185.OverlayValues[120] = d120
		ps185.OverlayValues[161] = d161
		ps185.OverlayValues[162] = d162
		ps186 := PhiState{General: true}
		ps186.OverlayValues = make([]JITValueDesc, 163)
		ps186.OverlayValues[0] = d0
		ps186.OverlayValues[1] = d1
		ps186.OverlayValues[2] = d2
		ps186.OverlayValues[3] = d3
		ps186.OverlayValues[4] = d4
		ps186.OverlayValues[5] = d5
		ps186.OverlayValues[24] = d24
		ps186.OverlayValues[25] = d25
		ps186.OverlayValues[26] = d26
		ps186.OverlayValues[51] = d51
		ps186.OverlayValues[52] = d52
		ps186.OverlayValues[81] = d81
		ps186.OverlayValues[82] = d82
		ps186.OverlayValues[83] = d83
		ps186.OverlayValues[118] = d118
		ps186.OverlayValues[119] = d119
		ps186.OverlayValues[120] = d120
		ps186.OverlayValues[161] = d161
		ps186.OverlayValues[162] = d162
		snap187 := d0
		snap188 := d1
		snap189 := d2
		snap190 := d3
		snap191 := d4
		snap192 := d5
		snap193 := d24
		snap194 := d25
		snap195 := d26
		snap196 := d51
		snap197 := d52
		snap198 := d81
		snap199 := d82
		snap200 := d83
		snap201 := d118
		snap202 := d119
		snap203 := d120
		snap204 := d161
		snap205 := d162
		alloc206 := ctx.SnapshotAllocState()
		if !bbs[13].Rendered {
			bbs[13].RenderPS(ps186)
		}
		ctx.RestoreAllocState(alloc206)
		d0 = snap187
		d1 = snap188
		d2 = snap189
		d3 = snap190
		d4 = snap191
		d5 = snap192
		d24 = snap193
		d25 = snap194
		d26 = snap195
		d51 = snap196
		d52 = snap197
		d81 = snap198
		d82 = snap199
		d83 = snap200
		d118 = snap201
		d119 = snap202
		d120 = snap203
		d161 = snap204
		d162 = snap205
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps185)
		}
		return result
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
			ctx.FlushRegisterMoves()
			bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_9 = bbs[9].Address
			ctx.MarkLabel(lbl10)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		ctx.ReclaimUntrackedRegs()
		var d207 JITValueDesc
		if args[0].Loc == LocImm {
			d207 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d207)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d207)
		} else {
			d207 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d207.Type = tagInt
			ctx.BindReg(d207.Reg, &d207)
		}
		ctx.EnsureDesc(&d207)
		ctx.EnsureDesc(&d207)
		var d208 JITValueDesc
		if d207.Loc == LocImm {
			d208 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d207.Imm.Int()))}
		} else {
			var r6 Reg
			r6 = d207.Reg
			d207.Loc = LocNone
			ctx.EmitCvtInt64ToFloat64(RegX0, r6)
			d208 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
			ctx.BindReg(r6, &d208)
		}
		ctx.FreeDesc(&d207)
		d209 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d208)
		resultTarget210 := false
		_ = resultTarget210
		ctx.EnsureDesc(&d209)
		ctx.EnsureDescsTogether(&d208, &d209)
		var d211 JITValueDesc
		if d208.Loc == LocImm && d209.Loc == LocImm {
			d211 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d208.Imm.Float() < d209.Imm.Float())}
		} else if d209.Loc == LocImm {
			var r7 Reg
			if result.Loc == LocReg && result.Reg != d208.Reg {
				r7 = result.Reg
				resultTarget210 = true
			} else {
				r7 = ctx.AllocRegExcept(d208.Reg)
			}
			_, yBits := d209.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r7, d208.Reg, RegR11, CondSignedLess)
			d211 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
			ctx.BindReg(r7, &d211)
		} else if d208.Loc == LocImm {
			var r8 Reg
			if result.Loc == LocReg && result.Reg != d209.Reg {
				r8 = result.Reg
				resultTarget210 = true
			} else {
				r8 = ctx.AllocRegExcept(d209.Reg)
			}
			_, xBits := d208.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r8, RegR11, d209.Reg, CondSignedLess)
			d211 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
			ctx.BindReg(r8, &d211)
		} else {
			var r9 Reg
			if result.Loc == LocReg && result.Reg != d208.Reg && result.Reg != d209.Reg {
				r9 = result.Reg
				resultTarget210 = true
			} else {
				r9 = ctx.AllocRegExcept(d208.Reg, d209.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r9, d208.Reg, d209.Reg, CondSignedLess)
			d211 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
			ctx.BindReg(r9, &d211)
		}
		ctx.FreeDesc(&d208)
		ctx.FreeDesc(&d209)
		ctx.EnsureDesc(&d211)
		ctx.EmitMovToReg(result.Reg, d211)
		result.Type = d211.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
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
			ctx.FlushRegisterMoves()
			bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_10 = bbs[10].Address
			ctx.MarkLabel(lbl11)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d212 JITValueDesc
		if d1.Loc == LocImm {
			d212 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x4))}
		} else {
			r10 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 4)
			d212 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondEqual}
			ctx.BindReg(r10, &d212)
		}
		d213 = d212
		ctx.EnsureDesc(&d213)
		if d213.Loc != LocImm && d213.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d213.Loc == LocImm {
			if d213.Imm.Bool() {
				if ps.General {
				}
				ps214 := PhiState{General: ps.General}
				ps214.OverlayValues = make([]JITValueDesc, 214)
				ps214.OverlayValues[0] = d0
				ps214.OverlayValues[1] = d1
				ps214.OverlayValues[2] = d2
				ps214.OverlayValues[3] = d3
				ps214.OverlayValues[4] = d4
				ps214.OverlayValues[5] = d5
				ps214.OverlayValues[24] = d24
				ps214.OverlayValues[25] = d25
				ps214.OverlayValues[26] = d26
				ps214.OverlayValues[51] = d51
				ps214.OverlayValues[52] = d52
				ps214.OverlayValues[81] = d81
				ps214.OverlayValues[82] = d82
				ps214.OverlayValues[83] = d83
				ps214.OverlayValues[118] = d118
				ps214.OverlayValues[119] = d119
				ps214.OverlayValues[120] = d120
				ps214.OverlayValues[161] = d161
				ps214.OverlayValues[162] = d162
				ps214.OverlayValues[207] = d207
				ps214.OverlayValues[208] = d208
				ps214.OverlayValues[209] = d209
				ps214.OverlayValues[211] = d211
				ps214.OverlayValues[212] = d212
				ps214.OverlayValues[213] = d213
				return bbs[9].RenderPS(ps214)
			}
			if ps.General {
			}
			ps215 := PhiState{General: ps.General}
			ps215.OverlayValues = make([]JITValueDesc, 214)
			ps215.OverlayValues[0] = d0
			ps215.OverlayValues[1] = d1
			ps215.OverlayValues[2] = d2
			ps215.OverlayValues[3] = d3
			ps215.OverlayValues[4] = d4
			ps215.OverlayValues[5] = d5
			ps215.OverlayValues[24] = d24
			ps215.OverlayValues[25] = d25
			ps215.OverlayValues[26] = d26
			ps215.OverlayValues[51] = d51
			ps215.OverlayValues[52] = d52
			ps215.OverlayValues[81] = d81
			ps215.OverlayValues[82] = d82
			ps215.OverlayValues[83] = d83
			ps215.OverlayValues[118] = d118
			ps215.OverlayValues[119] = d119
			ps215.OverlayValues[120] = d120
			ps215.OverlayValues[161] = d161
			ps215.OverlayValues[162] = d162
			ps215.OverlayValues[207] = d207
			ps215.OverlayValues[208] = d208
			ps215.OverlayValues[209] = d209
			ps215.OverlayValues[211] = d211
			ps215.OverlayValues[212] = d212
			ps215.OverlayValues[213] = d213
			return bbs[16].RenderPS(ps215)
		}
		if !ps.General {
			ps.General = true
			return bbs[10].RenderPS(ps)
		}
		ctx.EmitJump(d213.Condition, lbl10)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d212)
		snap216 := d0
		snap217 := d1
		snap218 := d2
		snap219 := d3
		snap220 := d4
		snap221 := d5
		snap222 := d24
		snap223 := d25
		snap224 := d26
		snap225 := d51
		snap226 := d52
		snap227 := d81
		snap228 := d82
		snap229 := d83
		snap230 := d118
		snap231 := d119
		snap232 := d120
		snap233 := d161
		snap234 := d162
		snap235 := d207
		snap236 := d208
		snap237 := d209
		snap238 := d211
		snap239 := d212
		snap240 := d213
		alloc241 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc241)
		d0 = snap216
		d1 = snap217
		d2 = snap218
		d3 = snap219
		d4 = snap220
		d5 = snap221
		d24 = snap222
		d25 = snap223
		d26 = snap224
		d51 = snap225
		d52 = snap226
		d81 = snap227
		d82 = snap228
		d83 = snap229
		d118 = snap230
		d119 = snap231
		d120 = snap232
		d161 = snap233
		d162 = snap234
		d207 = snap235
		d208 = snap236
		d209 = snap237
		d211 = snap238
		d212 = snap239
		d213 = snap240
		ctx.RestoreAllocState(alloc241)
		d0 = snap216
		d1 = snap217
		d2 = snap218
		d3 = snap219
		d4 = snap220
		d5 = snap221
		d24 = snap222
		d25 = snap223
		d26 = snap224
		d51 = snap225
		d52 = snap226
		d81 = snap227
		d82 = snap228
		d83 = snap229
		d118 = snap230
		d119 = snap231
		d120 = snap232
		d161 = snap233
		d162 = snap234
		d207 = snap235
		d208 = snap236
		d209 = snap237
		d211 = snap238
		d212 = snap239
		d213 = snap240
		ps242 := PhiState{General: true}
		ps242.OverlayValues = make([]JITValueDesc, 214)
		ps242.OverlayValues[0] = d0
		ps242.OverlayValues[1] = d1
		ps242.OverlayValues[2] = d2
		ps242.OverlayValues[3] = d3
		ps242.OverlayValues[4] = d4
		ps242.OverlayValues[5] = d5
		ps242.OverlayValues[24] = d24
		ps242.OverlayValues[25] = d25
		ps242.OverlayValues[26] = d26
		ps242.OverlayValues[51] = d51
		ps242.OverlayValues[52] = d52
		ps242.OverlayValues[81] = d81
		ps242.OverlayValues[82] = d82
		ps242.OverlayValues[83] = d83
		ps242.OverlayValues[118] = d118
		ps242.OverlayValues[119] = d119
		ps242.OverlayValues[120] = d120
		ps242.OverlayValues[161] = d161
		ps242.OverlayValues[162] = d162
		ps242.OverlayValues[207] = d207
		ps242.OverlayValues[208] = d208
		ps242.OverlayValues[209] = d209
		ps242.OverlayValues[211] = d211
		ps242.OverlayValues[212] = d212
		ps242.OverlayValues[213] = d213
		ps243 := PhiState{General: true}
		ps243.OverlayValues = make([]JITValueDesc, 214)
		ps243.OverlayValues[0] = d0
		ps243.OverlayValues[1] = d1
		ps243.OverlayValues[2] = d2
		ps243.OverlayValues[3] = d3
		ps243.OverlayValues[4] = d4
		ps243.OverlayValues[5] = d5
		ps243.OverlayValues[24] = d24
		ps243.OverlayValues[25] = d25
		ps243.OverlayValues[26] = d26
		ps243.OverlayValues[51] = d51
		ps243.OverlayValues[52] = d52
		ps243.OverlayValues[81] = d81
		ps243.OverlayValues[82] = d82
		ps243.OverlayValues[83] = d83
		ps243.OverlayValues[118] = d118
		ps243.OverlayValues[119] = d119
		ps243.OverlayValues[120] = d120
		ps243.OverlayValues[161] = d161
		ps243.OverlayValues[162] = d162
		ps243.OverlayValues[207] = d207
		ps243.OverlayValues[208] = d208
		ps243.OverlayValues[209] = d209
		ps243.OverlayValues[211] = d211
		ps243.OverlayValues[212] = d212
		ps243.OverlayValues[213] = d213
		snap244 := d0
		snap245 := d1
		snap246 := d2
		snap247 := d3
		snap248 := d4
		snap249 := d5
		snap250 := d24
		snap251 := d25
		snap252 := d26
		snap253 := d51
		snap254 := d52
		snap255 := d81
		snap256 := d82
		snap257 := d83
		snap258 := d118
		snap259 := d119
		snap260 := d120
		snap261 := d161
		snap262 := d162
		snap263 := d207
		snap264 := d208
		snap265 := d209
		snap266 := d211
		snap267 := d212
		snap268 := d213
		alloc269 := ctx.SnapshotAllocState()
		if !bbs[16].Rendered {
			bbs[16].RenderPS(ps243)
		}
		ctx.RestoreAllocState(alloc269)
		d0 = snap244
		d1 = snap245
		d2 = snap246
		d3 = snap247
		d4 = snap248
		d5 = snap249
		d24 = snap250
		d25 = snap251
		d26 = snap252
		d51 = snap253
		d52 = snap254
		d81 = snap255
		d82 = snap256
		d83 = snap257
		d118 = snap258
		d119 = snap259
		d120 = snap260
		d161 = snap261
		d162 = snap262
		d207 = snap263
		d208 = snap264
		d209 = snap265
		d211 = snap266
		d212 = snap267
		d213 = snap268
		if !bbs[9].Rendered {
			return bbs[9].RenderPS(ps242)
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
			ctx.FlushRegisterMoves()
			bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_11 = bbs[11].Address
			ctx.MarkLabel(lbl12)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		ctx.ReclaimUntrackedRegs()
		d271 = args[1]
		ctx.SyncDesc(&d271)
		if d271.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d271.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d271.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d271 = tmpScalar
		}
		d271 = JITPrepareScmerGoArg(ctx, d271)
		if d271.Loc != LocRegPair && d271.Loc != LocStackPair && d271.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d270 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d271}, 2)
		ctx.EnsureDesc(&d270)
		if d270.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d270.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d270.Imm)
			ptrWord, _ := d270.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d270.Imm.String())))
			d270 = tmpPair
		} else if d270.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d270.Type, Reg: ctx.AllocRegExcept(d270.Reg), Reg2: ctx.AllocRegExcept(d270.Reg)}
			switch d270.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d270)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d270)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d270)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d270)
			d270 = tmpPair
		}
		if d270.Loc != LocRegPair && d270.Loc != LocStackPair && d270.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateString arg0)")
		}
		ctx.SyncDesc(&d270)
		callResults272 := JITEmitGoCallResults(ctx, GoFuncAddr(ParseDateString), []JITValueDesc{d270}, []uint8{1, 1}, []uint8{0, 0})
		d273 = callResults272[0]
		_ = d273
		d274 = callResults272[1]
		_ = d274
		ctx.StabilizeDescForControlFlow(&d273)
		d275 = d274
		ctx.EnsureDesc(&d275)
		if d275.Loc != LocImm && d275.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d275.Loc == LocImm {
			if d275.Imm.Bool() {
				if ps.General {
				}
				ps276 := PhiState{General: ps.General}
				ps276.OverlayValues = make([]JITValueDesc, 276)
				ps276.OverlayValues[0] = d0
				ps276.OverlayValues[1] = d1
				ps276.OverlayValues[2] = d2
				ps276.OverlayValues[3] = d3
				ps276.OverlayValues[4] = d4
				ps276.OverlayValues[5] = d5
				ps276.OverlayValues[24] = d24
				ps276.OverlayValues[25] = d25
				ps276.OverlayValues[26] = d26
				ps276.OverlayValues[51] = d51
				ps276.OverlayValues[52] = d52
				ps276.OverlayValues[81] = d81
				ps276.OverlayValues[82] = d82
				ps276.OverlayValues[83] = d83
				ps276.OverlayValues[118] = d118
				ps276.OverlayValues[119] = d119
				ps276.OverlayValues[120] = d120
				ps276.OverlayValues[161] = d161
				ps276.OverlayValues[162] = d162
				ps276.OverlayValues[207] = d207
				ps276.OverlayValues[208] = d208
				ps276.OverlayValues[209] = d209
				ps276.OverlayValues[211] = d211
				ps276.OverlayValues[212] = d212
				ps276.OverlayValues[213] = d213
				ps276.OverlayValues[270] = d270
				ps276.OverlayValues[271] = d271
				ps276.OverlayValues[273] = d273
				ps276.OverlayValues[274] = d274
				ps276.OverlayValues[275] = d275
				return bbs[14].RenderPS(ps276)
			}
			if ps.General {
			}
			ps277 := PhiState{General: ps.General}
			ps277.OverlayValues = make([]JITValueDesc, 276)
			ps277.OverlayValues[0] = d0
			ps277.OverlayValues[1] = d1
			ps277.OverlayValues[2] = d2
			ps277.OverlayValues[3] = d3
			ps277.OverlayValues[4] = d4
			ps277.OverlayValues[5] = d5
			ps277.OverlayValues[24] = d24
			ps277.OverlayValues[25] = d25
			ps277.OverlayValues[26] = d26
			ps277.OverlayValues[51] = d51
			ps277.OverlayValues[52] = d52
			ps277.OverlayValues[81] = d81
			ps277.OverlayValues[82] = d82
			ps277.OverlayValues[83] = d83
			ps277.OverlayValues[118] = d118
			ps277.OverlayValues[119] = d119
			ps277.OverlayValues[120] = d120
			ps277.OverlayValues[161] = d161
			ps277.OverlayValues[162] = d162
			ps277.OverlayValues[207] = d207
			ps277.OverlayValues[208] = d208
			ps277.OverlayValues[209] = d209
			ps277.OverlayValues[211] = d211
			ps277.OverlayValues[212] = d212
			ps277.OverlayValues[213] = d213
			ps277.OverlayValues[270] = d270
			ps277.OverlayValues[271] = d271
			ps277.OverlayValues[273] = d273
			ps277.OverlayValues[274] = d274
			ps277.OverlayValues[275] = d275
			return bbs[12].RenderPS(ps277)
		}
		if !ps.General {
			ps.General = true
			return bbs[11].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d275.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl15)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		snap278 := d0
		snap279 := d1
		snap280 := d2
		snap281 := d3
		snap282 := d4
		snap283 := d5
		snap284 := d24
		snap285 := d25
		snap286 := d26
		snap287 := d51
		snap288 := d52
		snap289 := d81
		snap290 := d82
		snap291 := d83
		snap292 := d118
		snap293 := d119
		snap294 := d120
		snap295 := d161
		snap296 := d162
		snap297 := d207
		snap298 := d208
		snap299 := d209
		snap300 := d211
		snap301 := d212
		snap302 := d213
		snap303 := d270
		snap304 := d271
		snap305 := d273
		snap306 := d274
		snap307 := d275
		alloc308 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc308)
		d0 = snap278
		d1 = snap279
		d2 = snap280
		d3 = snap281
		d4 = snap282
		d5 = snap283
		d24 = snap284
		d25 = snap285
		d26 = snap286
		d51 = snap287
		d52 = snap288
		d81 = snap289
		d82 = snap290
		d83 = snap291
		d118 = snap292
		d119 = snap293
		d120 = snap294
		d161 = snap295
		d162 = snap296
		d207 = snap297
		d208 = snap298
		d209 = snap299
		d211 = snap300
		d212 = snap301
		d213 = snap302
		d270 = snap303
		d271 = snap304
		d273 = snap305
		d274 = snap306
		d275 = snap307
		ctx.RestoreAllocState(alloc308)
		d0 = snap278
		d1 = snap279
		d2 = snap280
		d3 = snap281
		d4 = snap282
		d5 = snap283
		d24 = snap284
		d25 = snap285
		d26 = snap286
		d51 = snap287
		d52 = snap288
		d81 = snap289
		d82 = snap290
		d83 = snap291
		d118 = snap292
		d119 = snap293
		d120 = snap294
		d161 = snap295
		d162 = snap296
		d207 = snap297
		d208 = snap298
		d209 = snap299
		d211 = snap300
		d212 = snap301
		d213 = snap302
		d270 = snap303
		d271 = snap304
		d273 = snap305
		d274 = snap306
		d275 = snap307
		ps309 := PhiState{General: true}
		ps309.OverlayValues = make([]JITValueDesc, 276)
		ps309.OverlayValues[0] = d0
		ps309.OverlayValues[1] = d1
		ps309.OverlayValues[2] = d2
		ps309.OverlayValues[3] = d3
		ps309.OverlayValues[4] = d4
		ps309.OverlayValues[5] = d5
		ps309.OverlayValues[24] = d24
		ps309.OverlayValues[25] = d25
		ps309.OverlayValues[26] = d26
		ps309.OverlayValues[51] = d51
		ps309.OverlayValues[52] = d52
		ps309.OverlayValues[81] = d81
		ps309.OverlayValues[82] = d82
		ps309.OverlayValues[83] = d83
		ps309.OverlayValues[118] = d118
		ps309.OverlayValues[119] = d119
		ps309.OverlayValues[120] = d120
		ps309.OverlayValues[161] = d161
		ps309.OverlayValues[162] = d162
		ps309.OverlayValues[207] = d207
		ps309.OverlayValues[208] = d208
		ps309.OverlayValues[209] = d209
		ps309.OverlayValues[211] = d211
		ps309.OverlayValues[212] = d212
		ps309.OverlayValues[213] = d213
		ps309.OverlayValues[270] = d270
		ps309.OverlayValues[271] = d271
		ps309.OverlayValues[273] = d273
		ps309.OverlayValues[274] = d274
		ps309.OverlayValues[275] = d275
		ps310 := PhiState{General: true}
		ps310.OverlayValues = make([]JITValueDesc, 276)
		ps310.OverlayValues[0] = d0
		ps310.OverlayValues[1] = d1
		ps310.OverlayValues[2] = d2
		ps310.OverlayValues[3] = d3
		ps310.OverlayValues[4] = d4
		ps310.OverlayValues[5] = d5
		ps310.OverlayValues[24] = d24
		ps310.OverlayValues[25] = d25
		ps310.OverlayValues[26] = d26
		ps310.OverlayValues[51] = d51
		ps310.OverlayValues[52] = d52
		ps310.OverlayValues[81] = d81
		ps310.OverlayValues[82] = d82
		ps310.OverlayValues[83] = d83
		ps310.OverlayValues[118] = d118
		ps310.OverlayValues[119] = d119
		ps310.OverlayValues[120] = d120
		ps310.OverlayValues[161] = d161
		ps310.OverlayValues[162] = d162
		ps310.OverlayValues[207] = d207
		ps310.OverlayValues[208] = d208
		ps310.OverlayValues[209] = d209
		ps310.OverlayValues[211] = d211
		ps310.OverlayValues[212] = d212
		ps310.OverlayValues[213] = d213
		ps310.OverlayValues[270] = d270
		ps310.OverlayValues[271] = d271
		ps310.OverlayValues[273] = d273
		ps310.OverlayValues[274] = d274
		ps310.OverlayValues[275] = d275
		snap311 := d0
		snap312 := d1
		snap313 := d2
		snap314 := d3
		snap315 := d4
		snap316 := d5
		snap317 := d24
		snap318 := d25
		snap319 := d26
		snap320 := d51
		snap321 := d52
		snap322 := d81
		snap323 := d82
		snap324 := d83
		snap325 := d118
		snap326 := d119
		snap327 := d120
		snap328 := d161
		snap329 := d162
		snap330 := d207
		snap331 := d208
		snap332 := d209
		snap333 := d211
		snap334 := d212
		snap335 := d213
		snap336 := d270
		snap337 := d271
		snap338 := d273
		snap339 := d274
		snap340 := d275
		alloc341 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps310)
		}
		ctx.RestoreAllocState(alloc341)
		d0 = snap311
		d1 = snap312
		d2 = snap313
		d3 = snap314
		d4 = snap315
		d5 = snap316
		d24 = snap317
		d25 = snap318
		d26 = snap319
		d51 = snap320
		d52 = snap321
		d81 = snap322
		d82 = snap323
		d83 = snap324
		d118 = snap325
		d119 = snap326
		d120 = snap327
		d161 = snap328
		d162 = snap329
		d207 = snap330
		d208 = snap331
		d209 = snap332
		d211 = snap333
		d212 = snap334
		d213 = snap335
		d270 = snap336
		d271 = snap337
		d273 = snap338
		d274 = snap339
		d275 = snap340
		if !bbs[14].Rendered {
			return bbs[14].RenderPS(ps309)
		}
		return result
		ctx.FreeDesc(&d274)
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
			ctx.FlushRegisterMoves()
			bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_12 = bbs[12].Address
			ctx.MarkLabel(lbl13)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		ctx.ReclaimUntrackedRegs()
		var d342 JITValueDesc
		if args[0].Loc == LocImm {
			d342 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d342 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d342)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d342 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d342)
		} else {
			d342 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d342.Type = tagInt
			ctx.BindReg(d342.Reg, &d342)
		}
		ctx.EnsureDesc(&d342)
		ctx.EnsureDesc(&d342)
		var d343 JITValueDesc
		if d342.Loc == LocImm {
			d343 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d342.Imm.Int()))}
		} else {
			var r11 Reg
			r11 = d342.Reg
			d342.Loc = LocNone
			ctx.EmitCvtInt64ToFloat64(RegX0, r11)
			d343 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r11}
			ctx.BindReg(r11, &d343)
		}
		ctx.FreeDesc(&d342)
		d344 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d343)
		resultTarget345 := false
		_ = resultTarget345
		ctx.EnsureDesc(&d344)
		ctx.EnsureDescsTogether(&d343, &d344)
		var d346 JITValueDesc
		if d343.Loc == LocImm && d344.Loc == LocImm {
			d346 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d343.Imm.Float() < d344.Imm.Float())}
		} else if d344.Loc == LocImm {
			var r12 Reg
			if result.Loc == LocReg && result.Reg != d343.Reg {
				r12 = result.Reg
				resultTarget345 = true
			} else {
				r12 = ctx.AllocRegExcept(d343.Reg)
			}
			_, yBits := d344.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r12, d343.Reg, RegR11, CondSignedLess)
			d346 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
			ctx.BindReg(r12, &d346)
		} else if d343.Loc == LocImm {
			var r13 Reg
			if result.Loc == LocReg && result.Reg != d344.Reg {
				r13 = result.Reg
				resultTarget345 = true
			} else {
				r13 = ctx.AllocRegExcept(d344.Reg)
			}
			_, xBits := d343.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r13, RegR11, d344.Reg, CondSignedLess)
			d346 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
			ctx.BindReg(r13, &d346)
		} else {
			var r14 Reg
			if result.Loc == LocReg && result.Reg != d343.Reg && result.Reg != d344.Reg {
				r14 = result.Reg
				resultTarget345 = true
			} else {
				r14 = ctx.AllocRegExcept(d343.Reg, d344.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r14, d343.Reg, d344.Reg, CondSignedLess)
			d346 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
			ctx.BindReg(r14, &d346)
		}
		ctx.FreeDesc(&d343)
		ctx.FreeDesc(&d344)
		ctx.EnsureDesc(&d346)
		ctx.EmitMovToReg(result.Reg, d346)
		result.Type = d346.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[13].VisitCount >= 0 {
				ps.General = true
				return bbs[13].RenderPS(ps)
			}
		}
		bbs[13].VisitCount++
		if ps.General {
			if bbs[13].Rendered {
				ctx.EmitJmp(lbl14)
				return result
			}
			bbs[13].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_13 = bbs[13].Address
			ctx.MarkLabel(lbl14)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		var d347 JITValueDesc
		if d3.Loc == LocImm {
			d347 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x2))}
		} else {
			r15 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 2)
			d347 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondEqual}
			ctx.BindReg(r15, &d347)
		}
		d348 = d347
		ctx.EnsureDesc(&d348)
		if d348.Loc != LocImm && d348.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d348.Loc == LocImm {
			if d348.Imm.Bool() {
				if ps.General {
				}
				ps349 := PhiState{General: ps.General}
				ps349.OverlayValues = make([]JITValueDesc, 349)
				ps349.OverlayValues[0] = d0
				ps349.OverlayValues[1] = d1
				ps349.OverlayValues[2] = d2
				ps349.OverlayValues[3] = d3
				ps349.OverlayValues[4] = d4
				ps349.OverlayValues[5] = d5
				ps349.OverlayValues[24] = d24
				ps349.OverlayValues[25] = d25
				ps349.OverlayValues[26] = d26
				ps349.OverlayValues[51] = d51
				ps349.OverlayValues[52] = d52
				ps349.OverlayValues[81] = d81
				ps349.OverlayValues[82] = d82
				ps349.OverlayValues[83] = d83
				ps349.OverlayValues[118] = d118
				ps349.OverlayValues[119] = d119
				ps349.OverlayValues[120] = d120
				ps349.OverlayValues[161] = d161
				ps349.OverlayValues[162] = d162
				ps349.OverlayValues[207] = d207
				ps349.OverlayValues[208] = d208
				ps349.OverlayValues[209] = d209
				ps349.OverlayValues[211] = d211
				ps349.OverlayValues[212] = d212
				ps349.OverlayValues[213] = d213
				ps349.OverlayValues[270] = d270
				ps349.OverlayValues[271] = d271
				ps349.OverlayValues[273] = d273
				ps349.OverlayValues[274] = d274
				ps349.OverlayValues[275] = d275
				ps349.OverlayValues[342] = d342
				ps349.OverlayValues[343] = d343
				ps349.OverlayValues[344] = d344
				ps349.OverlayValues[346] = d346
				ps349.OverlayValues[347] = d347
				ps349.OverlayValues[348] = d348
				return bbs[11].RenderPS(ps349)
			}
			if ps.General {
			}
			ps350 := PhiState{General: ps.General}
			ps350.OverlayValues = make([]JITValueDesc, 349)
			ps350.OverlayValues[0] = d0
			ps350.OverlayValues[1] = d1
			ps350.OverlayValues[2] = d2
			ps350.OverlayValues[3] = d3
			ps350.OverlayValues[4] = d4
			ps350.OverlayValues[5] = d5
			ps350.OverlayValues[24] = d24
			ps350.OverlayValues[25] = d25
			ps350.OverlayValues[26] = d26
			ps350.OverlayValues[51] = d51
			ps350.OverlayValues[52] = d52
			ps350.OverlayValues[81] = d81
			ps350.OverlayValues[82] = d82
			ps350.OverlayValues[83] = d83
			ps350.OverlayValues[118] = d118
			ps350.OverlayValues[119] = d119
			ps350.OverlayValues[120] = d120
			ps350.OverlayValues[161] = d161
			ps350.OverlayValues[162] = d162
			ps350.OverlayValues[207] = d207
			ps350.OverlayValues[208] = d208
			ps350.OverlayValues[209] = d209
			ps350.OverlayValues[211] = d211
			ps350.OverlayValues[212] = d212
			ps350.OverlayValues[213] = d213
			ps350.OverlayValues[270] = d270
			ps350.OverlayValues[271] = d271
			ps350.OverlayValues[273] = d273
			ps350.OverlayValues[274] = d274
			ps350.OverlayValues[275] = d275
			ps350.OverlayValues[342] = d342
			ps350.OverlayValues[343] = d343
			ps350.OverlayValues[344] = d344
			ps350.OverlayValues[346] = d346
			ps350.OverlayValues[347] = d347
			ps350.OverlayValues[348] = d348
			return bbs[12].RenderPS(ps350)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		ctx.EmitJump(d348.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d347)
		snap351 := d0
		snap352 := d1
		snap353 := d2
		snap354 := d3
		snap355 := d4
		snap356 := d5
		snap357 := d24
		snap358 := d25
		snap359 := d26
		snap360 := d51
		snap361 := d52
		snap362 := d81
		snap363 := d82
		snap364 := d83
		snap365 := d118
		snap366 := d119
		snap367 := d120
		snap368 := d161
		snap369 := d162
		snap370 := d207
		snap371 := d208
		snap372 := d209
		snap373 := d211
		snap374 := d212
		snap375 := d213
		snap376 := d270
		snap377 := d271
		snap378 := d273
		snap379 := d274
		snap380 := d275
		snap381 := d342
		snap382 := d343
		snap383 := d344
		snap384 := d346
		snap385 := d347
		snap386 := d348
		alloc387 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc387)
		d0 = snap351
		d1 = snap352
		d2 = snap353
		d3 = snap354
		d4 = snap355
		d5 = snap356
		d24 = snap357
		d25 = snap358
		d26 = snap359
		d51 = snap360
		d52 = snap361
		d81 = snap362
		d82 = snap363
		d83 = snap364
		d118 = snap365
		d119 = snap366
		d120 = snap367
		d161 = snap368
		d162 = snap369
		d207 = snap370
		d208 = snap371
		d209 = snap372
		d211 = snap373
		d212 = snap374
		d213 = snap375
		d270 = snap376
		d271 = snap377
		d273 = snap378
		d274 = snap379
		d275 = snap380
		d342 = snap381
		d343 = snap382
		d344 = snap383
		d346 = snap384
		d347 = snap385
		d348 = snap386
		ctx.RestoreAllocState(alloc387)
		d0 = snap351
		d1 = snap352
		d2 = snap353
		d3 = snap354
		d4 = snap355
		d5 = snap356
		d24 = snap357
		d25 = snap358
		d26 = snap359
		d51 = snap360
		d52 = snap361
		d81 = snap362
		d82 = snap363
		d83 = snap364
		d118 = snap365
		d119 = snap366
		d120 = snap367
		d161 = snap368
		d162 = snap369
		d207 = snap370
		d208 = snap371
		d209 = snap372
		d211 = snap373
		d212 = snap374
		d213 = snap375
		d270 = snap376
		d271 = snap377
		d273 = snap378
		d274 = snap379
		d275 = snap380
		d342 = snap381
		d343 = snap382
		d344 = snap383
		d346 = snap384
		d347 = snap385
		d348 = snap386
		ps388 := PhiState{General: true}
		ps388.OverlayValues = make([]JITValueDesc, 349)
		ps388.OverlayValues[0] = d0
		ps388.OverlayValues[1] = d1
		ps388.OverlayValues[2] = d2
		ps388.OverlayValues[3] = d3
		ps388.OverlayValues[4] = d4
		ps388.OverlayValues[5] = d5
		ps388.OverlayValues[24] = d24
		ps388.OverlayValues[25] = d25
		ps388.OverlayValues[26] = d26
		ps388.OverlayValues[51] = d51
		ps388.OverlayValues[52] = d52
		ps388.OverlayValues[81] = d81
		ps388.OverlayValues[82] = d82
		ps388.OverlayValues[83] = d83
		ps388.OverlayValues[118] = d118
		ps388.OverlayValues[119] = d119
		ps388.OverlayValues[120] = d120
		ps388.OverlayValues[161] = d161
		ps388.OverlayValues[162] = d162
		ps388.OverlayValues[207] = d207
		ps388.OverlayValues[208] = d208
		ps388.OverlayValues[209] = d209
		ps388.OverlayValues[211] = d211
		ps388.OverlayValues[212] = d212
		ps388.OverlayValues[213] = d213
		ps388.OverlayValues[270] = d270
		ps388.OverlayValues[271] = d271
		ps388.OverlayValues[273] = d273
		ps388.OverlayValues[274] = d274
		ps388.OverlayValues[275] = d275
		ps388.OverlayValues[342] = d342
		ps388.OverlayValues[343] = d343
		ps388.OverlayValues[344] = d344
		ps388.OverlayValues[346] = d346
		ps388.OverlayValues[347] = d347
		ps388.OverlayValues[348] = d348
		ps389 := PhiState{General: true}
		ps389.OverlayValues = make([]JITValueDesc, 349)
		ps389.OverlayValues[0] = d0
		ps389.OverlayValues[1] = d1
		ps389.OverlayValues[2] = d2
		ps389.OverlayValues[3] = d3
		ps389.OverlayValues[4] = d4
		ps389.OverlayValues[5] = d5
		ps389.OverlayValues[24] = d24
		ps389.OverlayValues[25] = d25
		ps389.OverlayValues[26] = d26
		ps389.OverlayValues[51] = d51
		ps389.OverlayValues[52] = d52
		ps389.OverlayValues[81] = d81
		ps389.OverlayValues[82] = d82
		ps389.OverlayValues[83] = d83
		ps389.OverlayValues[118] = d118
		ps389.OverlayValues[119] = d119
		ps389.OverlayValues[120] = d120
		ps389.OverlayValues[161] = d161
		ps389.OverlayValues[162] = d162
		ps389.OverlayValues[207] = d207
		ps389.OverlayValues[208] = d208
		ps389.OverlayValues[209] = d209
		ps389.OverlayValues[211] = d211
		ps389.OverlayValues[212] = d212
		ps389.OverlayValues[213] = d213
		ps389.OverlayValues[270] = d270
		ps389.OverlayValues[271] = d271
		ps389.OverlayValues[273] = d273
		ps389.OverlayValues[274] = d274
		ps389.OverlayValues[275] = d275
		ps389.OverlayValues[342] = d342
		ps389.OverlayValues[343] = d343
		ps389.OverlayValues[344] = d344
		ps389.OverlayValues[346] = d346
		ps389.OverlayValues[347] = d347
		ps389.OverlayValues[348] = d348
		snap390 := d0
		snap391 := d1
		snap392 := d2
		snap393 := d3
		snap394 := d4
		snap395 := d5
		snap396 := d24
		snap397 := d25
		snap398 := d26
		snap399 := d51
		snap400 := d52
		snap401 := d81
		snap402 := d82
		snap403 := d83
		snap404 := d118
		snap405 := d119
		snap406 := d120
		snap407 := d161
		snap408 := d162
		snap409 := d207
		snap410 := d208
		snap411 := d209
		snap412 := d211
		snap413 := d212
		snap414 := d213
		snap415 := d270
		snap416 := d271
		snap417 := d273
		snap418 := d274
		snap419 := d275
		snap420 := d342
		snap421 := d343
		snap422 := d344
		snap423 := d346
		snap424 := d347
		snap425 := d348
		alloc426 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps389)
		}
		ctx.RestoreAllocState(alloc426)
		d0 = snap390
		d1 = snap391
		d2 = snap392
		d3 = snap393
		d4 = snap394
		d5 = snap395
		d24 = snap396
		d25 = snap397
		d26 = snap398
		d51 = snap399
		d52 = snap400
		d81 = snap401
		d82 = snap402
		d83 = snap403
		d118 = snap404
		d119 = snap405
		d120 = snap406
		d161 = snap407
		d162 = snap408
		d207 = snap409
		d208 = snap410
		d209 = snap411
		d211 = snap412
		d212 = snap413
		d213 = snap414
		d270 = snap415
		d271 = snap416
		d273 = snap417
		d274 = snap418
		d275 = snap419
		d342 = snap420
		d343 = snap421
		d344 = snap422
		d346 = snap423
		d347 = snap424
		d348 = snap425
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps388)
		}
		return result
		return result
	}
	bbs[14].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[14].VisitCount >= 0 {
				ps.General = true
				return bbs[14].RenderPS(ps)
			}
		}
		bbs[14].VisitCount++
		if ps.General {
			if bbs[14].Rendered {
				ctx.EmitJmp(lbl15)
				return result
			}
			bbs[14].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[14].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_14 = bbs[14].Address
			ctx.MarkLabel(lbl15)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		ctx.ReclaimUntrackedRegs()
		var d427 JITValueDesc
		if args[0].Loc == LocImm {
			d427 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d427 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d427)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d427 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d427)
		} else {
			d427 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d427.Type = tagInt
			ctx.BindReg(d427.Reg, &d427)
		}
		ctx.EnsureDesc(&d427)
		resultTarget428 := false
		_ = resultTarget428
		ctx.EnsureDesc(&d273)
		ctx.EnsureDescsTogether(&d427, &d273)
		var d429 JITValueDesc
		if d427.Loc == LocImm && d273.Loc == LocImm {
			d429 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d427.Imm.Int() < d273.Imm.Int())}
		} else if d273.Loc == LocImm {
			if d273.Imm.Int() >= -2147483648 && d273.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d427.Reg, int32(d273.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(RegR11, uint64(d273.Imm.Int()))
				ctx.EmitCmpInt64(d427.Reg, RegR11)
			}
			var r16 Reg
			if result.Loc == LocReg && result.Reg != d427.Reg {
				r16 = result.Reg
				resultTarget428 = true
			} else {
				r16 = ctx.AllocRegExcept(d427.Reg)
			}
			ctx.EmitSetcc(r16, CondSignedLess)
			d429 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
			ctx.BindReg(r16, &d429)
		} else if d427.Loc == LocImm {
			ctx.EmitMovRegImm64(RegR11, uint64(d427.Imm.Int()))
			ctx.EmitCmpInt64(RegR11, d273.Reg)
			var r17 Reg
			if result.Loc == LocReg && result.Reg != d273.Reg {
				r17 = result.Reg
				resultTarget428 = true
			} else {
				r17 = ctx.AllocRegExcept(d273.Reg)
			}
			ctx.EmitSetcc(r17, CondSignedLess)
			d429 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
			ctx.BindReg(r17, &d429)
		} else {
			ctx.EmitCmpInt64(d427.Reg, d273.Reg)
			var r18 Reg
			if result.Loc == LocReg && result.Reg != d427.Reg && result.Reg != d273.Reg {
				r18 = result.Reg
				resultTarget428 = true
			} else {
				r18 = ctx.AllocRegExcept(d427.Reg, d273.Reg)
			}
			ctx.EmitSetcc(r18, CondSignedLess)
			d429 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
			ctx.BindReg(r18, &d429)
		}
		ctx.FreeDesc(&d427)
		ctx.EnsureDesc(&d429)
		ctx.EmitMovToReg(result.Reg, d429)
		result.Type = d429.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[15].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[15].VisitCount >= 0 {
				ps.General = true
				return bbs[15].RenderPS(ps)
			}
		}
		bbs[15].VisitCount++
		if ps.General {
			if bbs[15].Rendered {
				ctx.EmitJmp(lbl16)
				return result
			}
			bbs[15].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[15].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_15 = bbs[15].Address
			ctx.MarkLabel(lbl16)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != LocNone {
			d429 = ps.OverlayValues[429]
		}
		ctx.ReclaimUntrackedRegs()
		d430 = ctx.EmitFloatDesc(args[0])
		d431 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d430)
		resultTarget432 := false
		_ = resultTarget432
		ctx.EnsureDesc(&d431)
		ctx.EnsureDescsTogether(&d430, &d431)
		var d433 JITValueDesc
		if d430.Loc == LocImm && d431.Loc == LocImm {
			d433 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d430.Imm.Float() < d431.Imm.Float())}
		} else if d431.Loc == LocImm {
			var r19 Reg
			if result.Loc == LocReg && result.Reg != d430.Reg {
				r19 = result.Reg
				resultTarget432 = true
			} else {
				r19 = ctx.AllocRegExcept(d430.Reg)
			}
			_, yBits := d431.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r19, d430.Reg, RegR11, CondSignedLess)
			d433 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
			ctx.BindReg(r19, &d433)
		} else if d430.Loc == LocImm {
			var r20 Reg
			if result.Loc == LocReg && result.Reg != d431.Reg {
				r20 = result.Reg
				resultTarget432 = true
			} else {
				r20 = ctx.AllocRegExcept(d431.Reg)
			}
			_, xBits := d430.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r20, RegR11, d431.Reg, CondSignedLess)
			d433 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
			ctx.BindReg(r20, &d433)
		} else {
			var r21 Reg
			if result.Loc == LocReg && result.Reg != d430.Reg && result.Reg != d431.Reg {
				r21 = result.Reg
				resultTarget432 = true
			} else {
				r21 = ctx.AllocRegExcept(d430.Reg, d431.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r21, d430.Reg, d431.Reg, CondSignedLess)
			d433 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
			ctx.BindReg(r21, &d433)
		}
		ctx.FreeDesc(&d430)
		ctx.FreeDesc(&d431)
		ctx.EnsureDesc(&d433)
		ctx.EmitMovToReg(result.Reg, d433)
		result.Type = d433.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[16].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[16].VisitCount >= 0 {
				ps.General = true
				return bbs[16].RenderPS(ps)
			}
		}
		bbs[16].VisitCount++
		if ps.General {
			if bbs[16].Rendered {
				ctx.EmitJmp(lbl17)
				return result
			}
			bbs[16].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[16].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_16 = bbs[16].Address
			ctx.MarkLabel(lbl17)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != LocNone {
			d433 = ps.OverlayValues[433]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d434 JITValueDesc
		if d1.Loc == LocImm {
			d434 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x3))}
		} else {
			r22 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 3)
			d434 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondEqual}
			ctx.BindReg(r22, &d434)
		}
		d435 = d434
		ctx.EnsureDesc(&d435)
		if d435.Loc != LocImm && d435.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d435.Loc == LocImm {
			if d435.Imm.Bool() {
				if ps.General {
				}
				ps436 := PhiState{General: ps.General}
				ps436.OverlayValues = make([]JITValueDesc, 436)
				ps436.OverlayValues[0] = d0
				ps436.OverlayValues[1] = d1
				ps436.OverlayValues[2] = d2
				ps436.OverlayValues[3] = d3
				ps436.OverlayValues[4] = d4
				ps436.OverlayValues[5] = d5
				ps436.OverlayValues[24] = d24
				ps436.OverlayValues[25] = d25
				ps436.OverlayValues[26] = d26
				ps436.OverlayValues[51] = d51
				ps436.OverlayValues[52] = d52
				ps436.OverlayValues[81] = d81
				ps436.OverlayValues[82] = d82
				ps436.OverlayValues[83] = d83
				ps436.OverlayValues[118] = d118
				ps436.OverlayValues[119] = d119
				ps436.OverlayValues[120] = d120
				ps436.OverlayValues[161] = d161
				ps436.OverlayValues[162] = d162
				ps436.OverlayValues[207] = d207
				ps436.OverlayValues[208] = d208
				ps436.OverlayValues[209] = d209
				ps436.OverlayValues[211] = d211
				ps436.OverlayValues[212] = d212
				ps436.OverlayValues[213] = d213
				ps436.OverlayValues[270] = d270
				ps436.OverlayValues[271] = d271
				ps436.OverlayValues[273] = d273
				ps436.OverlayValues[274] = d274
				ps436.OverlayValues[275] = d275
				ps436.OverlayValues[342] = d342
				ps436.OverlayValues[343] = d343
				ps436.OverlayValues[344] = d344
				ps436.OverlayValues[346] = d346
				ps436.OverlayValues[347] = d347
				ps436.OverlayValues[348] = d348
				ps436.OverlayValues[427] = d427
				ps436.OverlayValues[429] = d429
				ps436.OverlayValues[430] = d430
				ps436.OverlayValues[431] = d431
				ps436.OverlayValues[433] = d433
				ps436.OverlayValues[434] = d434
				ps436.OverlayValues[435] = d435
				return bbs[15].RenderPS(ps436)
			}
			if ps.General {
			}
			ps437 := PhiState{General: ps.General}
			ps437.OverlayValues = make([]JITValueDesc, 436)
			ps437.OverlayValues[0] = d0
			ps437.OverlayValues[1] = d1
			ps437.OverlayValues[2] = d2
			ps437.OverlayValues[3] = d3
			ps437.OverlayValues[4] = d4
			ps437.OverlayValues[5] = d5
			ps437.OverlayValues[24] = d24
			ps437.OverlayValues[25] = d25
			ps437.OverlayValues[26] = d26
			ps437.OverlayValues[51] = d51
			ps437.OverlayValues[52] = d52
			ps437.OverlayValues[81] = d81
			ps437.OverlayValues[82] = d82
			ps437.OverlayValues[83] = d83
			ps437.OverlayValues[118] = d118
			ps437.OverlayValues[119] = d119
			ps437.OverlayValues[120] = d120
			ps437.OverlayValues[161] = d161
			ps437.OverlayValues[162] = d162
			ps437.OverlayValues[207] = d207
			ps437.OverlayValues[208] = d208
			ps437.OverlayValues[209] = d209
			ps437.OverlayValues[211] = d211
			ps437.OverlayValues[212] = d212
			ps437.OverlayValues[213] = d213
			ps437.OverlayValues[270] = d270
			ps437.OverlayValues[271] = d271
			ps437.OverlayValues[273] = d273
			ps437.OverlayValues[274] = d274
			ps437.OverlayValues[275] = d275
			ps437.OverlayValues[342] = d342
			ps437.OverlayValues[343] = d343
			ps437.OverlayValues[344] = d344
			ps437.OverlayValues[346] = d346
			ps437.OverlayValues[347] = d347
			ps437.OverlayValues[348] = d348
			ps437.OverlayValues[427] = d427
			ps437.OverlayValues[429] = d429
			ps437.OverlayValues[430] = d430
			ps437.OverlayValues[431] = d431
			ps437.OverlayValues[433] = d433
			ps437.OverlayValues[434] = d434
			ps437.OverlayValues[435] = d435
			return bbs[18].RenderPS(ps437)
		}
		if !ps.General {
			ps.General = true
			return bbs[16].RenderPS(ps)
		}
		ctx.EmitJump(d435.Condition, lbl16)
		if bbs[18].Rendered {
			ctx.EmitJmp(lbl19)
		}
		ctx.FreeDesc(&d434)
		snap438 := d0
		snap439 := d1
		snap440 := d2
		snap441 := d3
		snap442 := d4
		snap443 := d5
		snap444 := d24
		snap445 := d25
		snap446 := d26
		snap447 := d51
		snap448 := d52
		snap449 := d81
		snap450 := d82
		snap451 := d83
		snap452 := d118
		snap453 := d119
		snap454 := d120
		snap455 := d161
		snap456 := d162
		snap457 := d207
		snap458 := d208
		snap459 := d209
		snap460 := d211
		snap461 := d212
		snap462 := d213
		snap463 := d270
		snap464 := d271
		snap465 := d273
		snap466 := d274
		snap467 := d275
		snap468 := d342
		snap469 := d343
		snap470 := d344
		snap471 := d346
		snap472 := d347
		snap473 := d348
		snap474 := d427
		snap475 := d429
		snap476 := d430
		snap477 := d431
		snap478 := d433
		snap479 := d434
		snap480 := d435
		alloc481 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc481)
		d0 = snap438
		d1 = snap439
		d2 = snap440
		d3 = snap441
		d4 = snap442
		d5 = snap443
		d24 = snap444
		d25 = snap445
		d26 = snap446
		d51 = snap447
		d52 = snap448
		d81 = snap449
		d82 = snap450
		d83 = snap451
		d118 = snap452
		d119 = snap453
		d120 = snap454
		d161 = snap455
		d162 = snap456
		d207 = snap457
		d208 = snap458
		d209 = snap459
		d211 = snap460
		d212 = snap461
		d213 = snap462
		d270 = snap463
		d271 = snap464
		d273 = snap465
		d274 = snap466
		d275 = snap467
		d342 = snap468
		d343 = snap469
		d344 = snap470
		d346 = snap471
		d347 = snap472
		d348 = snap473
		d427 = snap474
		d429 = snap475
		d430 = snap476
		d431 = snap477
		d433 = snap478
		d434 = snap479
		d435 = snap480
		ctx.RestoreAllocState(alloc481)
		d0 = snap438
		d1 = snap439
		d2 = snap440
		d3 = snap441
		d4 = snap442
		d5 = snap443
		d24 = snap444
		d25 = snap445
		d26 = snap446
		d51 = snap447
		d52 = snap448
		d81 = snap449
		d82 = snap450
		d83 = snap451
		d118 = snap452
		d119 = snap453
		d120 = snap454
		d161 = snap455
		d162 = snap456
		d207 = snap457
		d208 = snap458
		d209 = snap459
		d211 = snap460
		d212 = snap461
		d213 = snap462
		d270 = snap463
		d271 = snap464
		d273 = snap465
		d274 = snap466
		d275 = snap467
		d342 = snap468
		d343 = snap469
		d344 = snap470
		d346 = snap471
		d347 = snap472
		d348 = snap473
		d427 = snap474
		d429 = snap475
		d430 = snap476
		d431 = snap477
		d433 = snap478
		d434 = snap479
		d435 = snap480
		ps482 := PhiState{General: true}
		ps482.OverlayValues = make([]JITValueDesc, 436)
		ps482.OverlayValues[0] = d0
		ps482.OverlayValues[1] = d1
		ps482.OverlayValues[2] = d2
		ps482.OverlayValues[3] = d3
		ps482.OverlayValues[4] = d4
		ps482.OverlayValues[5] = d5
		ps482.OverlayValues[24] = d24
		ps482.OverlayValues[25] = d25
		ps482.OverlayValues[26] = d26
		ps482.OverlayValues[51] = d51
		ps482.OverlayValues[52] = d52
		ps482.OverlayValues[81] = d81
		ps482.OverlayValues[82] = d82
		ps482.OverlayValues[83] = d83
		ps482.OverlayValues[118] = d118
		ps482.OverlayValues[119] = d119
		ps482.OverlayValues[120] = d120
		ps482.OverlayValues[161] = d161
		ps482.OverlayValues[162] = d162
		ps482.OverlayValues[207] = d207
		ps482.OverlayValues[208] = d208
		ps482.OverlayValues[209] = d209
		ps482.OverlayValues[211] = d211
		ps482.OverlayValues[212] = d212
		ps482.OverlayValues[213] = d213
		ps482.OverlayValues[270] = d270
		ps482.OverlayValues[271] = d271
		ps482.OverlayValues[273] = d273
		ps482.OverlayValues[274] = d274
		ps482.OverlayValues[275] = d275
		ps482.OverlayValues[342] = d342
		ps482.OverlayValues[343] = d343
		ps482.OverlayValues[344] = d344
		ps482.OverlayValues[346] = d346
		ps482.OverlayValues[347] = d347
		ps482.OverlayValues[348] = d348
		ps482.OverlayValues[427] = d427
		ps482.OverlayValues[429] = d429
		ps482.OverlayValues[430] = d430
		ps482.OverlayValues[431] = d431
		ps482.OverlayValues[433] = d433
		ps482.OverlayValues[434] = d434
		ps482.OverlayValues[435] = d435
		ps483 := PhiState{General: true}
		ps483.OverlayValues = make([]JITValueDesc, 436)
		ps483.OverlayValues[0] = d0
		ps483.OverlayValues[1] = d1
		ps483.OverlayValues[2] = d2
		ps483.OverlayValues[3] = d3
		ps483.OverlayValues[4] = d4
		ps483.OverlayValues[5] = d5
		ps483.OverlayValues[24] = d24
		ps483.OverlayValues[25] = d25
		ps483.OverlayValues[26] = d26
		ps483.OverlayValues[51] = d51
		ps483.OverlayValues[52] = d52
		ps483.OverlayValues[81] = d81
		ps483.OverlayValues[82] = d82
		ps483.OverlayValues[83] = d83
		ps483.OverlayValues[118] = d118
		ps483.OverlayValues[119] = d119
		ps483.OverlayValues[120] = d120
		ps483.OverlayValues[161] = d161
		ps483.OverlayValues[162] = d162
		ps483.OverlayValues[207] = d207
		ps483.OverlayValues[208] = d208
		ps483.OverlayValues[209] = d209
		ps483.OverlayValues[211] = d211
		ps483.OverlayValues[212] = d212
		ps483.OverlayValues[213] = d213
		ps483.OverlayValues[270] = d270
		ps483.OverlayValues[271] = d271
		ps483.OverlayValues[273] = d273
		ps483.OverlayValues[274] = d274
		ps483.OverlayValues[275] = d275
		ps483.OverlayValues[342] = d342
		ps483.OverlayValues[343] = d343
		ps483.OverlayValues[344] = d344
		ps483.OverlayValues[346] = d346
		ps483.OverlayValues[347] = d347
		ps483.OverlayValues[348] = d348
		ps483.OverlayValues[427] = d427
		ps483.OverlayValues[429] = d429
		ps483.OverlayValues[430] = d430
		ps483.OverlayValues[431] = d431
		ps483.OverlayValues[433] = d433
		ps483.OverlayValues[434] = d434
		ps483.OverlayValues[435] = d435
		snap484 := d0
		snap485 := d1
		snap486 := d2
		snap487 := d3
		snap488 := d4
		snap489 := d5
		snap490 := d24
		snap491 := d25
		snap492 := d26
		snap493 := d51
		snap494 := d52
		snap495 := d81
		snap496 := d82
		snap497 := d83
		snap498 := d118
		snap499 := d119
		snap500 := d120
		snap501 := d161
		snap502 := d162
		snap503 := d207
		snap504 := d208
		snap505 := d209
		snap506 := d211
		snap507 := d212
		snap508 := d213
		snap509 := d270
		snap510 := d271
		snap511 := d273
		snap512 := d274
		snap513 := d275
		snap514 := d342
		snap515 := d343
		snap516 := d344
		snap517 := d346
		snap518 := d347
		snap519 := d348
		snap520 := d427
		snap521 := d429
		snap522 := d430
		snap523 := d431
		snap524 := d433
		snap525 := d434
		snap526 := d435
		alloc527 := ctx.SnapshotAllocState()
		if !bbs[18].Rendered {
			bbs[18].RenderPS(ps483)
		}
		ctx.RestoreAllocState(alloc527)
		d0 = snap484
		d1 = snap485
		d2 = snap486
		d3 = snap487
		d4 = snap488
		d5 = snap489
		d24 = snap490
		d25 = snap491
		d26 = snap492
		d51 = snap493
		d52 = snap494
		d81 = snap495
		d82 = snap496
		d83 = snap497
		d118 = snap498
		d119 = snap499
		d120 = snap500
		d161 = snap501
		d162 = snap502
		d207 = snap503
		d208 = snap504
		d209 = snap505
		d211 = snap506
		d212 = snap507
		d213 = snap508
		d270 = snap509
		d271 = snap510
		d273 = snap511
		d274 = snap512
		d275 = snap513
		d342 = snap514
		d343 = snap515
		d344 = snap516
		d346 = snap517
		d347 = snap518
		d348 = snap519
		d427 = snap520
		d429 = snap521
		d430 = snap522
		d431 = snap523
		d433 = snap524
		d434 = snap525
		d435 = snap526
		if !bbs[15].Rendered {
			return bbs[15].RenderPS(ps482)
		}
		return result
		return result
	}
	bbs[17].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[17].VisitCount >= 0 {
				ps.General = true
				return bbs[17].RenderPS(ps)
			}
		}
		bbs[17].VisitCount++
		if ps.General {
			if bbs[17].Rendered {
				ctx.EmitJmp(lbl18)
				return result
			}
			bbs[17].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[17].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_17 = bbs[17].Address
			ctx.MarkLabel(lbl18)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
			d435 = ps.OverlayValues[435]
		}
		ctx.ReclaimUntrackedRegs()
		var d528 JITValueDesc
		if args[0].Loc == LocImm {
			d528 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d528 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d528)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d528 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d528)
		} else {
			d528 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d528.Type = tagInt
			ctx.BindReg(d528.Reg, &d528)
		}
		var d529 JITValueDesc
		if args[1].Loc == LocImm {
			d529 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d529 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d529)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d529 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d529)
		} else {
			d529 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[1]}, 1)
			d529.Type = tagInt
			ctx.BindReg(d529.Reg, &d529)
		}
		ctx.EnsureDesc(&d528)
		resultTarget530 := false
		_ = resultTarget530
		ctx.EnsureDesc(&d529)
		ctx.EnsureDescsTogether(&d528, &d529)
		var d531 JITValueDesc
		if d528.Loc == LocImm && d529.Loc == LocImm {
			d531 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d528.Imm.Int() < d529.Imm.Int())}
		} else if d529.Loc == LocImm {
			if d529.Imm.Int() >= -2147483648 && d529.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d528.Reg, int32(d529.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(RegR11, uint64(d529.Imm.Int()))
				ctx.EmitCmpInt64(d528.Reg, RegR11)
			}
			var r23 Reg
			if result.Loc == LocReg && result.Reg != d528.Reg {
				r23 = result.Reg
				resultTarget530 = true
			} else {
				r23 = ctx.AllocRegExcept(d528.Reg)
			}
			ctx.EmitSetcc(r23, CondSignedLess)
			d531 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
			ctx.BindReg(r23, &d531)
		} else if d528.Loc == LocImm {
			ctx.EmitMovRegImm64(RegR11, uint64(d528.Imm.Int()))
			ctx.EmitCmpInt64(RegR11, d529.Reg)
			var r24 Reg
			if result.Loc == LocReg && result.Reg != d529.Reg {
				r24 = result.Reg
				resultTarget530 = true
			} else {
				r24 = ctx.AllocRegExcept(d529.Reg)
			}
			ctx.EmitSetcc(r24, CondSignedLess)
			d531 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
			ctx.BindReg(r24, &d531)
		} else {
			ctx.EmitCmpInt64(d528.Reg, d529.Reg)
			var r25 Reg
			if result.Loc == LocReg && result.Reg != d528.Reg && result.Reg != d529.Reg {
				r25 = result.Reg
				resultTarget530 = true
			} else {
				r25 = ctx.AllocRegExcept(d528.Reg, d529.Reg)
			}
			ctx.EmitSetcc(r25, CondSignedLess)
			d531 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
			ctx.BindReg(r25, &d531)
		}
		ctx.FreeDesc(&d528)
		ctx.FreeDesc(&d529)
		ctx.EnsureDesc(&d531)
		ctx.EmitMovToReg(result.Reg, d531)
		result.Type = d531.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[18].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[18].VisitCount >= 0 {
				ps.General = true
				return bbs[18].RenderPS(ps)
			}
		}
		bbs[18].VisitCount++
		if ps.General {
			if bbs[18].Rendered {
				ctx.EmitJmp(lbl19)
				return result
			}
			bbs[18].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[18].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_18 = bbs[18].Address
			ctx.MarkLabel(lbl19)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != LocNone {
			d531 = ps.OverlayValues[531]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d532 JITValueDesc
		if d1.Loc == LocImm {
			d532 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x5))}
		} else {
			r26 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 5)
			d532 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondEqual}
			ctx.BindReg(r26, &d532)
		}
		d533 = d532
		ctx.EnsureDesc(&d533)
		if d533.Loc != LocImm && d533.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d533.Loc == LocImm {
			if d533.Imm.Bool() {
				if ps.General {
				}
				ps534 := PhiState{General: ps.General}
				ps534.OverlayValues = make([]JITValueDesc, 534)
				ps534.OverlayValues[0] = d0
				ps534.OverlayValues[1] = d1
				ps534.OverlayValues[2] = d2
				ps534.OverlayValues[3] = d3
				ps534.OverlayValues[4] = d4
				ps534.OverlayValues[5] = d5
				ps534.OverlayValues[24] = d24
				ps534.OverlayValues[25] = d25
				ps534.OverlayValues[26] = d26
				ps534.OverlayValues[51] = d51
				ps534.OverlayValues[52] = d52
				ps534.OverlayValues[81] = d81
				ps534.OverlayValues[82] = d82
				ps534.OverlayValues[83] = d83
				ps534.OverlayValues[118] = d118
				ps534.OverlayValues[119] = d119
				ps534.OverlayValues[120] = d120
				ps534.OverlayValues[161] = d161
				ps534.OverlayValues[162] = d162
				ps534.OverlayValues[207] = d207
				ps534.OverlayValues[208] = d208
				ps534.OverlayValues[209] = d209
				ps534.OverlayValues[211] = d211
				ps534.OverlayValues[212] = d212
				ps534.OverlayValues[213] = d213
				ps534.OverlayValues[270] = d270
				ps534.OverlayValues[271] = d271
				ps534.OverlayValues[273] = d273
				ps534.OverlayValues[274] = d274
				ps534.OverlayValues[275] = d275
				ps534.OverlayValues[342] = d342
				ps534.OverlayValues[343] = d343
				ps534.OverlayValues[344] = d344
				ps534.OverlayValues[346] = d346
				ps534.OverlayValues[347] = d347
				ps534.OverlayValues[348] = d348
				ps534.OverlayValues[427] = d427
				ps534.OverlayValues[429] = d429
				ps534.OverlayValues[430] = d430
				ps534.OverlayValues[431] = d431
				ps534.OverlayValues[433] = d433
				ps534.OverlayValues[434] = d434
				ps534.OverlayValues[435] = d435
				ps534.OverlayValues[528] = d528
				ps534.OverlayValues[529] = d529
				ps534.OverlayValues[531] = d531
				ps534.OverlayValues[532] = d532
				ps534.OverlayValues[533] = d533
				return bbs[17].RenderPS(ps534)
			}
			if ps.General {
			}
			ps535 := PhiState{General: ps.General}
			ps535.OverlayValues = make([]JITValueDesc, 534)
			ps535.OverlayValues[0] = d0
			ps535.OverlayValues[1] = d1
			ps535.OverlayValues[2] = d2
			ps535.OverlayValues[3] = d3
			ps535.OverlayValues[4] = d4
			ps535.OverlayValues[5] = d5
			ps535.OverlayValues[24] = d24
			ps535.OverlayValues[25] = d25
			ps535.OverlayValues[26] = d26
			ps535.OverlayValues[51] = d51
			ps535.OverlayValues[52] = d52
			ps535.OverlayValues[81] = d81
			ps535.OverlayValues[82] = d82
			ps535.OverlayValues[83] = d83
			ps535.OverlayValues[118] = d118
			ps535.OverlayValues[119] = d119
			ps535.OverlayValues[120] = d120
			ps535.OverlayValues[161] = d161
			ps535.OverlayValues[162] = d162
			ps535.OverlayValues[207] = d207
			ps535.OverlayValues[208] = d208
			ps535.OverlayValues[209] = d209
			ps535.OverlayValues[211] = d211
			ps535.OverlayValues[212] = d212
			ps535.OverlayValues[213] = d213
			ps535.OverlayValues[270] = d270
			ps535.OverlayValues[271] = d271
			ps535.OverlayValues[273] = d273
			ps535.OverlayValues[274] = d274
			ps535.OverlayValues[275] = d275
			ps535.OverlayValues[342] = d342
			ps535.OverlayValues[343] = d343
			ps535.OverlayValues[344] = d344
			ps535.OverlayValues[346] = d346
			ps535.OverlayValues[347] = d347
			ps535.OverlayValues[348] = d348
			ps535.OverlayValues[427] = d427
			ps535.OverlayValues[429] = d429
			ps535.OverlayValues[430] = d430
			ps535.OverlayValues[431] = d431
			ps535.OverlayValues[433] = d433
			ps535.OverlayValues[434] = d434
			ps535.OverlayValues[435] = d435
			ps535.OverlayValues[528] = d528
			ps535.OverlayValues[529] = d529
			ps535.OverlayValues[531] = d531
			ps535.OverlayValues[532] = d532
			ps535.OverlayValues[533] = d533
			return bbs[19].RenderPS(ps535)
		}
		if !ps.General {
			ps.General = true
			return bbs[18].RenderPS(ps)
		}
		ctx.EmitJump(d533.Condition, lbl18)
		if bbs[19].Rendered {
			ctx.EmitJmp(lbl20)
		}
		ctx.FreeDesc(&d532)
		snap536 := d0
		snap537 := d1
		snap538 := d2
		snap539 := d3
		snap540 := d4
		snap541 := d5
		snap542 := d24
		snap543 := d25
		snap544 := d26
		snap545 := d51
		snap546 := d52
		snap547 := d81
		snap548 := d82
		snap549 := d83
		snap550 := d118
		snap551 := d119
		snap552 := d120
		snap553 := d161
		snap554 := d162
		snap555 := d207
		snap556 := d208
		snap557 := d209
		snap558 := d211
		snap559 := d212
		snap560 := d213
		snap561 := d270
		snap562 := d271
		snap563 := d273
		snap564 := d274
		snap565 := d275
		snap566 := d342
		snap567 := d343
		snap568 := d344
		snap569 := d346
		snap570 := d347
		snap571 := d348
		snap572 := d427
		snap573 := d429
		snap574 := d430
		snap575 := d431
		snap576 := d433
		snap577 := d434
		snap578 := d435
		snap579 := d528
		snap580 := d529
		snap581 := d531
		snap582 := d532
		snap583 := d533
		alloc584 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc584)
		d0 = snap536
		d1 = snap537
		d2 = snap538
		d3 = snap539
		d4 = snap540
		d5 = snap541
		d24 = snap542
		d25 = snap543
		d26 = snap544
		d51 = snap545
		d52 = snap546
		d81 = snap547
		d82 = snap548
		d83 = snap549
		d118 = snap550
		d119 = snap551
		d120 = snap552
		d161 = snap553
		d162 = snap554
		d207 = snap555
		d208 = snap556
		d209 = snap557
		d211 = snap558
		d212 = snap559
		d213 = snap560
		d270 = snap561
		d271 = snap562
		d273 = snap563
		d274 = snap564
		d275 = snap565
		d342 = snap566
		d343 = snap567
		d344 = snap568
		d346 = snap569
		d347 = snap570
		d348 = snap571
		d427 = snap572
		d429 = snap573
		d430 = snap574
		d431 = snap575
		d433 = snap576
		d434 = snap577
		d435 = snap578
		d528 = snap579
		d529 = snap580
		d531 = snap581
		d532 = snap582
		d533 = snap583
		ctx.RestoreAllocState(alloc584)
		d0 = snap536
		d1 = snap537
		d2 = snap538
		d3 = snap539
		d4 = snap540
		d5 = snap541
		d24 = snap542
		d25 = snap543
		d26 = snap544
		d51 = snap545
		d52 = snap546
		d81 = snap547
		d82 = snap548
		d83 = snap549
		d118 = snap550
		d119 = snap551
		d120 = snap552
		d161 = snap553
		d162 = snap554
		d207 = snap555
		d208 = snap556
		d209 = snap557
		d211 = snap558
		d212 = snap559
		d213 = snap560
		d270 = snap561
		d271 = snap562
		d273 = snap563
		d274 = snap564
		d275 = snap565
		d342 = snap566
		d343 = snap567
		d344 = snap568
		d346 = snap569
		d347 = snap570
		d348 = snap571
		d427 = snap572
		d429 = snap573
		d430 = snap574
		d431 = snap575
		d433 = snap576
		d434 = snap577
		d435 = snap578
		d528 = snap579
		d529 = snap580
		d531 = snap581
		d532 = snap582
		d533 = snap583
		ps585 := PhiState{General: true}
		ps585.OverlayValues = make([]JITValueDesc, 534)
		ps585.OverlayValues[0] = d0
		ps585.OverlayValues[1] = d1
		ps585.OverlayValues[2] = d2
		ps585.OverlayValues[3] = d3
		ps585.OverlayValues[4] = d4
		ps585.OverlayValues[5] = d5
		ps585.OverlayValues[24] = d24
		ps585.OverlayValues[25] = d25
		ps585.OverlayValues[26] = d26
		ps585.OverlayValues[51] = d51
		ps585.OverlayValues[52] = d52
		ps585.OverlayValues[81] = d81
		ps585.OverlayValues[82] = d82
		ps585.OverlayValues[83] = d83
		ps585.OverlayValues[118] = d118
		ps585.OverlayValues[119] = d119
		ps585.OverlayValues[120] = d120
		ps585.OverlayValues[161] = d161
		ps585.OverlayValues[162] = d162
		ps585.OverlayValues[207] = d207
		ps585.OverlayValues[208] = d208
		ps585.OverlayValues[209] = d209
		ps585.OverlayValues[211] = d211
		ps585.OverlayValues[212] = d212
		ps585.OverlayValues[213] = d213
		ps585.OverlayValues[270] = d270
		ps585.OverlayValues[271] = d271
		ps585.OverlayValues[273] = d273
		ps585.OverlayValues[274] = d274
		ps585.OverlayValues[275] = d275
		ps585.OverlayValues[342] = d342
		ps585.OverlayValues[343] = d343
		ps585.OverlayValues[344] = d344
		ps585.OverlayValues[346] = d346
		ps585.OverlayValues[347] = d347
		ps585.OverlayValues[348] = d348
		ps585.OverlayValues[427] = d427
		ps585.OverlayValues[429] = d429
		ps585.OverlayValues[430] = d430
		ps585.OverlayValues[431] = d431
		ps585.OverlayValues[433] = d433
		ps585.OverlayValues[434] = d434
		ps585.OverlayValues[435] = d435
		ps585.OverlayValues[528] = d528
		ps585.OverlayValues[529] = d529
		ps585.OverlayValues[531] = d531
		ps585.OverlayValues[532] = d532
		ps585.OverlayValues[533] = d533
		ps586 := PhiState{General: true}
		ps586.OverlayValues = make([]JITValueDesc, 534)
		ps586.OverlayValues[0] = d0
		ps586.OverlayValues[1] = d1
		ps586.OverlayValues[2] = d2
		ps586.OverlayValues[3] = d3
		ps586.OverlayValues[4] = d4
		ps586.OverlayValues[5] = d5
		ps586.OverlayValues[24] = d24
		ps586.OverlayValues[25] = d25
		ps586.OverlayValues[26] = d26
		ps586.OverlayValues[51] = d51
		ps586.OverlayValues[52] = d52
		ps586.OverlayValues[81] = d81
		ps586.OverlayValues[82] = d82
		ps586.OverlayValues[83] = d83
		ps586.OverlayValues[118] = d118
		ps586.OverlayValues[119] = d119
		ps586.OverlayValues[120] = d120
		ps586.OverlayValues[161] = d161
		ps586.OverlayValues[162] = d162
		ps586.OverlayValues[207] = d207
		ps586.OverlayValues[208] = d208
		ps586.OverlayValues[209] = d209
		ps586.OverlayValues[211] = d211
		ps586.OverlayValues[212] = d212
		ps586.OverlayValues[213] = d213
		ps586.OverlayValues[270] = d270
		ps586.OverlayValues[271] = d271
		ps586.OverlayValues[273] = d273
		ps586.OverlayValues[274] = d274
		ps586.OverlayValues[275] = d275
		ps586.OverlayValues[342] = d342
		ps586.OverlayValues[343] = d343
		ps586.OverlayValues[344] = d344
		ps586.OverlayValues[346] = d346
		ps586.OverlayValues[347] = d347
		ps586.OverlayValues[348] = d348
		ps586.OverlayValues[427] = d427
		ps586.OverlayValues[429] = d429
		ps586.OverlayValues[430] = d430
		ps586.OverlayValues[431] = d431
		ps586.OverlayValues[433] = d433
		ps586.OverlayValues[434] = d434
		ps586.OverlayValues[435] = d435
		ps586.OverlayValues[528] = d528
		ps586.OverlayValues[529] = d529
		ps586.OverlayValues[531] = d531
		ps586.OverlayValues[532] = d532
		ps586.OverlayValues[533] = d533
		snap587 := d0
		snap588 := d1
		snap589 := d2
		snap590 := d3
		snap591 := d4
		snap592 := d5
		snap593 := d24
		snap594 := d25
		snap595 := d26
		snap596 := d51
		snap597 := d52
		snap598 := d81
		snap599 := d82
		snap600 := d83
		snap601 := d118
		snap602 := d119
		snap603 := d120
		snap604 := d161
		snap605 := d162
		snap606 := d207
		snap607 := d208
		snap608 := d209
		snap609 := d211
		snap610 := d212
		snap611 := d213
		snap612 := d270
		snap613 := d271
		snap614 := d273
		snap615 := d274
		snap616 := d275
		snap617 := d342
		snap618 := d343
		snap619 := d344
		snap620 := d346
		snap621 := d347
		snap622 := d348
		snap623 := d427
		snap624 := d429
		snap625 := d430
		snap626 := d431
		snap627 := d433
		snap628 := d434
		snap629 := d435
		snap630 := d528
		snap631 := d529
		snap632 := d531
		snap633 := d532
		snap634 := d533
		alloc635 := ctx.SnapshotAllocState()
		if !bbs[19].Rendered {
			bbs[19].RenderPS(ps586)
		}
		ctx.RestoreAllocState(alloc635)
		d0 = snap587
		d1 = snap588
		d2 = snap589
		d3 = snap590
		d4 = snap591
		d5 = snap592
		d24 = snap593
		d25 = snap594
		d26 = snap595
		d51 = snap596
		d52 = snap597
		d81 = snap598
		d82 = snap599
		d83 = snap600
		d118 = snap601
		d119 = snap602
		d120 = snap603
		d161 = snap604
		d162 = snap605
		d207 = snap606
		d208 = snap607
		d209 = snap608
		d211 = snap609
		d212 = snap610
		d213 = snap611
		d270 = snap612
		d271 = snap613
		d273 = snap614
		d274 = snap615
		d275 = snap616
		d342 = snap617
		d343 = snap618
		d344 = snap619
		d346 = snap620
		d347 = snap621
		d348 = snap622
		d427 = snap623
		d429 = snap624
		d430 = snap625
		d431 = snap626
		d433 = snap627
		d434 = snap628
		d435 = snap629
		d528 = snap630
		d529 = snap631
		d531 = snap632
		d532 = snap633
		d533 = snap634
		if !bbs[17].Rendered {
			return bbs[17].RenderPS(ps585)
		}
		return result
		return result
	}
	bbs[19].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[19].VisitCount >= 0 {
				ps.General = true
				return bbs[19].RenderPS(ps)
			}
		}
		bbs[19].VisitCount++
		if ps.General {
			if bbs[19].Rendered {
				ctx.EmitJmp(lbl20)
				return result
			}
			bbs[19].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[19].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_19 = bbs[19].Address
			ctx.MarkLabel(lbl20)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
			d207 = ps.OverlayValues[207]
		}
		if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
			d208 = ps.OverlayValues[208]
		}
		if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
			d209 = ps.OverlayValues[209]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
			d211 = ps.OverlayValues[211]
		}
		if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
			d212 = ps.OverlayValues[212]
		}
		if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
			d213 = ps.OverlayValues[213]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
			d273 = ps.OverlayValues[273]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
			d342 = ps.OverlayValues[342]
		}
		if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
			d343 = ps.OverlayValues[343]
		}
		if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
			d344 = ps.OverlayValues[344]
		}
		if len(ps.OverlayValues) > 346 && ps.OverlayValues[346].Loc != LocNone {
			d346 = ps.OverlayValues[346]
		}
		if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
			d347 = ps.OverlayValues[347]
		}
		if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
			d348 = ps.OverlayValues[348]
		}
		if len(ps.OverlayValues) > 427 && ps.OverlayValues[427].Loc != LocNone {
			d427 = ps.OverlayValues[427]
		}
		if len(ps.OverlayValues) > 429 && ps.OverlayValues[429].Loc != LocNone {
			d429 = ps.OverlayValues[429]
		}
		if len(ps.OverlayValues) > 430 && ps.OverlayValues[430].Loc != LocNone {
			d430 = ps.OverlayValues[430]
		}
		if len(ps.OverlayValues) > 431 && ps.OverlayValues[431].Loc != LocNone {
			d431 = ps.OverlayValues[431]
		}
		if len(ps.OverlayValues) > 433 && ps.OverlayValues[433].Loc != LocNone {
			d433 = ps.OverlayValues[433]
		}
		if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
			d434 = ps.OverlayValues[434]
		}
		if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
			d435 = ps.OverlayValues[435]
		}
		if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != LocNone {
			d528 = ps.OverlayValues[528]
		}
		if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != LocNone {
			d529 = ps.OverlayValues[529]
		}
		if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != LocNone {
			d531 = ps.OverlayValues[531]
		}
		if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != LocNone {
			d532 = ps.OverlayValues[532]
		}
		if len(ps.OverlayValues) > 533 && ps.OverlayValues[533].Loc != LocNone {
			d533 = ps.OverlayValues[533]
		}
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
		d636 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d1, d3}, 1)
		d636.NoHeapPointer = true
		ctx.EmitAndRegImm32(d636.Reg, 1)
		d636.Type = tagBool
		ctx.BindReg(d636.Reg, &d636)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.EnsureDesc(&d636)
		ctx.EmitMovToReg(result.Reg, d636)
		result.Type = d636.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	ps637 := PhiState{General: false}
	_ = bbs[0].RenderPS(ps637)
	ctx.MarkLabel(lbl0)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg)
	}
	return result

}
