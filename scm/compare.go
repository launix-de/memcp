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
	var d257 JITValueDesc
	_ = d257
	var d258 JITValueDesc
	_ = d258
	var d311 JITValueDesc
	_ = d311
	var d312 JITValueDesc
	_ = d312
	var d314 JITValueDesc
	_ = d314
	var d315 JITValueDesc
	_ = d315
	var d316 JITValueDesc
	_ = d316
	var d379 JITValueDesc
	_ = d379
	var d380 JITValueDesc
	_ = d380
	var d381 JITValueDesc
	_ = d381
	var d383 JITValueDesc
	_ = d383
	var d384 JITValueDesc
	_ = d384
	var d385 JITValueDesc
	_ = d385
	var d460 JITValueDesc
	_ = d460
	var d462 JITValueDesc
	_ = d462
	var d463 JITValueDesc
	_ = d463
	var d464 JITValueDesc
	_ = d464
	var d466 JITValueDesc
	_ = d466
	var d467 JITValueDesc
	_ = d467
	var d468 JITValueDesc
	_ = d468
	var d557 JITValueDesc
	_ = d557
	var d558 JITValueDesc
	_ = d558
	var d560 JITValueDesc
	_ = d560
	var d561 JITValueDesc
	_ = d561
	var d562 JITValueDesc
	_ = d562
	var d563 JITValueDesc
	_ = d563
	var d565 JITValueDesc
	_ = d565
	var d566 JITValueDesc
	_ = d566
	var d567 JITValueDesc
	_ = d567
	var d569 JITValueDesc
	_ = d569
	var d570 JITValueDesc
	_ = d570
	var d571 JITValueDesc
	_ = d571
	var d684 JITValueDesc
	_ = d684
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
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d24) {
			return d24
		}
		ctx.EnsureDesc(&d24)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
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
				ps27.OverlayValues = make([]JITValueDesc, 9)
				ps27.OverlayValues[0] = d0
				ps27.OverlayValues[1] = d1
				ps27.OverlayValues[2] = d2
				ps27.OverlayValues[3] = d3
				ps27.OverlayValues[4] = d4
				ps27.OverlayValues[5] = d5
				ps27.OverlayValues[6] = d24
				ps27.OverlayValues[7] = d25
				ps27.OverlayValues[8] = d26
				return bbs[4].RenderPS(ps27)
			}
			if ps.General {
			}
			ps28 := PhiState{General: ps.General}
			ps28.OverlayValues = make([]JITValueDesc, 9)
			ps28.OverlayValues[0] = d0
			ps28.OverlayValues[1] = d1
			ps28.OverlayValues[2] = d2
			ps28.OverlayValues[3] = d3
			ps28.OverlayValues[4] = d4
			ps28.OverlayValues[5] = d5
			ps28.OverlayValues[6] = d24
			ps28.OverlayValues[7] = d25
			ps28.OverlayValues[8] = d26
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
		ps39.OverlayValues = make([]JITValueDesc, 9)
		ps39.OverlayValues[0] = d0
		ps39.OverlayValues[1] = d1
		ps39.OverlayValues[2] = d2
		ps39.OverlayValues[3] = d3
		ps39.OverlayValues[4] = d4
		ps39.OverlayValues[5] = d5
		ps39.OverlayValues[6] = d24
		ps39.OverlayValues[7] = d25
		ps39.OverlayValues[8] = d26
		ps40 := PhiState{General: true}
		ps40.OverlayValues = make([]JITValueDesc, 9)
		ps40.OverlayValues[0] = d0
		ps40.OverlayValues[1] = d1
		ps40.OverlayValues[2] = d2
		ps40.OverlayValues[3] = d3
		ps40.OverlayValues[4] = d4
		ps40.OverlayValues[5] = d5
		ps40.OverlayValues[6] = d24
		ps40.OverlayValues[7] = d25
		ps40.OverlayValues[8] = d26
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
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
				ps53.OverlayValues = make([]JITValueDesc, 11)
				ps53.OverlayValues[0] = d0
				ps53.OverlayValues[1] = d1
				ps53.OverlayValues[2] = d2
				ps53.OverlayValues[3] = d3
				ps53.OverlayValues[4] = d4
				ps53.OverlayValues[5] = d5
				ps53.OverlayValues[6] = d24
				ps53.OverlayValues[7] = d25
				ps53.OverlayValues[8] = d26
				ps53.OverlayValues[9] = d51
				ps53.OverlayValues[10] = d52
				return bbs[1].RenderPS(ps53)
			}
			if ps.General {
			}
			ps54 := PhiState{General: ps.General}
			ps54.OverlayValues = make([]JITValueDesc, 11)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[5] = d5
			ps54.OverlayValues[6] = d24
			ps54.OverlayValues[7] = d25
			ps54.OverlayValues[8] = d26
			ps54.OverlayValues[9] = d51
			ps54.OverlayValues[10] = d52
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
		ps67.OverlayValues = make([]JITValueDesc, 11)
		ps67.OverlayValues[0] = d0
		ps67.OverlayValues[1] = d1
		ps67.OverlayValues[2] = d2
		ps67.OverlayValues[3] = d3
		ps67.OverlayValues[4] = d4
		ps67.OverlayValues[5] = d5
		ps67.OverlayValues[6] = d24
		ps67.OverlayValues[7] = d25
		ps67.OverlayValues[8] = d26
		ps67.OverlayValues[9] = d51
		ps67.OverlayValues[10] = d52
		ps68 := PhiState{General: true}
		ps68.OverlayValues = make([]JITValueDesc, 11)
		ps68.OverlayValues[0] = d0
		ps68.OverlayValues[1] = d1
		ps68.OverlayValues[2] = d2
		ps68.OverlayValues[3] = d3
		ps68.OverlayValues[4] = d4
		ps68.OverlayValues[5] = d5
		ps68.OverlayValues[6] = d24
		ps68.OverlayValues[7] = d25
		ps68.OverlayValues[8] = d26
		ps68.OverlayValues[9] = d51
		ps68.OverlayValues[10] = d52
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		ctx.ReclaimUntrackedRegs()
		d81 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d81) {
			return d81
		}
		ctx.EnsureDesc(&d81)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
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
				ps84.OverlayValues = make([]JITValueDesc, 14)
				ps84.OverlayValues[0] = d0
				ps84.OverlayValues[1] = d1
				ps84.OverlayValues[2] = d2
				ps84.OverlayValues[3] = d3
				ps84.OverlayValues[4] = d4
				ps84.OverlayValues[5] = d5
				ps84.OverlayValues[6] = d24
				ps84.OverlayValues[7] = d25
				ps84.OverlayValues[8] = d26
				ps84.OverlayValues[9] = d51
				ps84.OverlayValues[10] = d52
				ps84.OverlayValues[11] = d81
				ps84.OverlayValues[12] = d82
				ps84.OverlayValues[13] = d83
				return bbs[6].RenderPS(ps84)
			}
			if ps.General {
			}
			ps85 := PhiState{General: ps.General}
			ps85.OverlayValues = make([]JITValueDesc, 14)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[1] = d1
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[6] = d24
			ps85.OverlayValues[7] = d25
			ps85.OverlayValues[8] = d26
			ps85.OverlayValues[9] = d51
			ps85.OverlayValues[10] = d52
			ps85.OverlayValues[11] = d81
			ps85.OverlayValues[12] = d82
			ps85.OverlayValues[13] = d83
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
		ps101.OverlayValues = make([]JITValueDesc, 14)
		ps101.OverlayValues[0] = d0
		ps101.OverlayValues[1] = d1
		ps101.OverlayValues[2] = d2
		ps101.OverlayValues[3] = d3
		ps101.OverlayValues[4] = d4
		ps101.OverlayValues[5] = d5
		ps101.OverlayValues[6] = d24
		ps101.OverlayValues[7] = d25
		ps101.OverlayValues[8] = d26
		ps101.OverlayValues[9] = d51
		ps101.OverlayValues[10] = d52
		ps101.OverlayValues[11] = d81
		ps101.OverlayValues[12] = d82
		ps101.OverlayValues[13] = d83
		ps102 := PhiState{General: true}
		ps102.OverlayValues = make([]JITValueDesc, 14)
		ps102.OverlayValues[0] = d0
		ps102.OverlayValues[1] = d1
		ps102.OverlayValues[2] = d2
		ps102.OverlayValues[3] = d3
		ps102.OverlayValues[4] = d4
		ps102.OverlayValues[5] = d5
		ps102.OverlayValues[6] = d24
		ps102.OverlayValues[7] = d25
		ps102.OverlayValues[8] = d26
		ps102.OverlayValues[9] = d51
		ps102.OverlayValues[10] = d52
		ps102.OverlayValues[11] = d81
		ps102.OverlayValues[12] = d82
		ps102.OverlayValues[13] = d83
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		ctx.ReclaimUntrackedRegs()
		d118 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d118) {
			return d118
		}
		ctx.EnsureDesc(&d118)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
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
				ps121.OverlayValues = make([]JITValueDesc, 17)
				ps121.OverlayValues[0] = d0
				ps121.OverlayValues[1] = d1
				ps121.OverlayValues[2] = d2
				ps121.OverlayValues[3] = d3
				ps121.OverlayValues[4] = d4
				ps121.OverlayValues[5] = d5
				ps121.OverlayValues[6] = d24
				ps121.OverlayValues[7] = d25
				ps121.OverlayValues[8] = d26
				ps121.OverlayValues[9] = d51
				ps121.OverlayValues[10] = d52
				ps121.OverlayValues[11] = d81
				ps121.OverlayValues[12] = d82
				ps121.OverlayValues[13] = d83
				ps121.OverlayValues[14] = d118
				ps121.OverlayValues[15] = d119
				ps121.OverlayValues[16] = d120
				return bbs[8].RenderPS(ps121)
			}
			if ps.General {
			}
			ps122 := PhiState{General: ps.General}
			ps122.OverlayValues = make([]JITValueDesc, 17)
			ps122.OverlayValues[0] = d0
			ps122.OverlayValues[1] = d1
			ps122.OverlayValues[2] = d2
			ps122.OverlayValues[3] = d3
			ps122.OverlayValues[4] = d4
			ps122.OverlayValues[5] = d5
			ps122.OverlayValues[6] = d24
			ps122.OverlayValues[7] = d25
			ps122.OverlayValues[8] = d26
			ps122.OverlayValues[9] = d51
			ps122.OverlayValues[10] = d52
			ps122.OverlayValues[11] = d81
			ps122.OverlayValues[12] = d82
			ps122.OverlayValues[13] = d83
			ps122.OverlayValues[14] = d118
			ps122.OverlayValues[15] = d119
			ps122.OverlayValues[16] = d120
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
		ps141.OverlayValues = make([]JITValueDesc, 17)
		ps141.OverlayValues[0] = d0
		ps141.OverlayValues[1] = d1
		ps141.OverlayValues[2] = d2
		ps141.OverlayValues[3] = d3
		ps141.OverlayValues[4] = d4
		ps141.OverlayValues[5] = d5
		ps141.OverlayValues[6] = d24
		ps141.OverlayValues[7] = d25
		ps141.OverlayValues[8] = d26
		ps141.OverlayValues[9] = d51
		ps141.OverlayValues[10] = d52
		ps141.OverlayValues[11] = d81
		ps141.OverlayValues[12] = d82
		ps141.OverlayValues[13] = d83
		ps141.OverlayValues[14] = d118
		ps141.OverlayValues[15] = d119
		ps141.OverlayValues[16] = d120
		ps142 := PhiState{General: true}
		ps142.OverlayValues = make([]JITValueDesc, 17)
		ps142.OverlayValues[0] = d0
		ps142.OverlayValues[1] = d1
		ps142.OverlayValues[2] = d2
		ps142.OverlayValues[3] = d3
		ps142.OverlayValues[4] = d4
		ps142.OverlayValues[5] = d5
		ps142.OverlayValues[6] = d24
		ps142.OverlayValues[7] = d25
		ps142.OverlayValues[8] = d26
		ps142.OverlayValues[9] = d51
		ps142.OverlayValues[10] = d52
		ps142.OverlayValues[11] = d81
		ps142.OverlayValues[12] = d82
		ps142.OverlayValues[13] = d83
		ps142.OverlayValues[14] = d118
		ps142.OverlayValues[15] = d119
		ps142.OverlayValues[16] = d120
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
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
				ps163.OverlayValues = make([]JITValueDesc, 19)
				ps163.OverlayValues[0] = d0
				ps163.OverlayValues[1] = d1
				ps163.OverlayValues[2] = d2
				ps163.OverlayValues[3] = d3
				ps163.OverlayValues[4] = d4
				ps163.OverlayValues[5] = d5
				ps163.OverlayValues[6] = d24
				ps163.OverlayValues[7] = d25
				ps163.OverlayValues[8] = d26
				ps163.OverlayValues[9] = d51
				ps163.OverlayValues[10] = d52
				ps163.OverlayValues[11] = d81
				ps163.OverlayValues[12] = d82
				ps163.OverlayValues[13] = d83
				ps163.OverlayValues[14] = d118
				ps163.OverlayValues[15] = d119
				ps163.OverlayValues[16] = d120
				ps163.OverlayValues[17] = d161
				ps163.OverlayValues[18] = d162
				return bbs[11].RenderPS(ps163)
			}
			if ps.General {
			}
			ps164 := PhiState{General: ps.General}
			ps164.OverlayValues = make([]JITValueDesc, 19)
			ps164.OverlayValues[0] = d0
			ps164.OverlayValues[1] = d1
			ps164.OverlayValues[2] = d2
			ps164.OverlayValues[3] = d3
			ps164.OverlayValues[4] = d4
			ps164.OverlayValues[5] = d5
			ps164.OverlayValues[6] = d24
			ps164.OverlayValues[7] = d25
			ps164.OverlayValues[8] = d26
			ps164.OverlayValues[9] = d51
			ps164.OverlayValues[10] = d52
			ps164.OverlayValues[11] = d81
			ps164.OverlayValues[12] = d82
			ps164.OverlayValues[13] = d83
			ps164.OverlayValues[14] = d118
			ps164.OverlayValues[15] = d119
			ps164.OverlayValues[16] = d120
			ps164.OverlayValues[17] = d161
			ps164.OverlayValues[18] = d162
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
		ps185.OverlayValues = make([]JITValueDesc, 19)
		ps185.OverlayValues[0] = d0
		ps185.OverlayValues[1] = d1
		ps185.OverlayValues[2] = d2
		ps185.OverlayValues[3] = d3
		ps185.OverlayValues[4] = d4
		ps185.OverlayValues[5] = d5
		ps185.OverlayValues[6] = d24
		ps185.OverlayValues[7] = d25
		ps185.OverlayValues[8] = d26
		ps185.OverlayValues[9] = d51
		ps185.OverlayValues[10] = d52
		ps185.OverlayValues[11] = d81
		ps185.OverlayValues[12] = d82
		ps185.OverlayValues[13] = d83
		ps185.OverlayValues[14] = d118
		ps185.OverlayValues[15] = d119
		ps185.OverlayValues[16] = d120
		ps185.OverlayValues[17] = d161
		ps185.OverlayValues[18] = d162
		ps186 := PhiState{General: true}
		ps186.OverlayValues = make([]JITValueDesc, 19)
		ps186.OverlayValues[0] = d0
		ps186.OverlayValues[1] = d1
		ps186.OverlayValues[2] = d2
		ps186.OverlayValues[3] = d3
		ps186.OverlayValues[4] = d4
		ps186.OverlayValues[5] = d5
		ps186.OverlayValues[6] = d24
		ps186.OverlayValues[7] = d25
		ps186.OverlayValues[8] = d26
		ps186.OverlayValues[9] = d51
		ps186.OverlayValues[10] = d52
		ps186.OverlayValues[11] = d81
		ps186.OverlayValues[12] = d82
		ps186.OverlayValues[13] = d83
		ps186.OverlayValues[14] = d118
		ps186.OverlayValues[15] = d119
		ps186.OverlayValues[16] = d120
		ps186.OverlayValues[17] = d161
		ps186.OverlayValues[18] = d162
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		var d207 JITValueDesc
		if d3.Loc == LocImm {
			d207 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x4))}
		} else {
			r6 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 4)
			d207 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondEqual}
			ctx.BindReg(r6, &d207)
		}
		d208 = d207
		ctx.EnsureDesc(&d208)
		if d208.Loc != LocImm && d208.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d208.Loc == LocImm {
			if d208.Imm.Bool() {
				if ps.General {
				}
				ps209 := PhiState{General: ps.General}
				ps209.OverlayValues = make([]JITValueDesc, 21)
				ps209.OverlayValues[0] = d0
				ps209.OverlayValues[1] = d1
				ps209.OverlayValues[2] = d2
				ps209.OverlayValues[3] = d3
				ps209.OverlayValues[4] = d4
				ps209.OverlayValues[5] = d5
				ps209.OverlayValues[6] = d24
				ps209.OverlayValues[7] = d25
				ps209.OverlayValues[8] = d26
				ps209.OverlayValues[9] = d51
				ps209.OverlayValues[10] = d52
				ps209.OverlayValues[11] = d81
				ps209.OverlayValues[12] = d82
				ps209.OverlayValues[13] = d83
				ps209.OverlayValues[14] = d118
				ps209.OverlayValues[15] = d119
				ps209.OverlayValues[16] = d120
				ps209.OverlayValues[17] = d161
				ps209.OverlayValues[18] = d162
				ps209.OverlayValues[19] = d207
				ps209.OverlayValues[20] = d208
				return bbs[17].RenderPS(ps209)
			}
			if ps.General {
			}
			ps210 := PhiState{General: ps.General}
			ps210.OverlayValues = make([]JITValueDesc, 21)
			ps210.OverlayValues[0] = d0
			ps210.OverlayValues[1] = d1
			ps210.OverlayValues[2] = d2
			ps210.OverlayValues[3] = d3
			ps210.OverlayValues[4] = d4
			ps210.OverlayValues[5] = d5
			ps210.OverlayValues[6] = d24
			ps210.OverlayValues[7] = d25
			ps210.OverlayValues[8] = d26
			ps210.OverlayValues[9] = d51
			ps210.OverlayValues[10] = d52
			ps210.OverlayValues[11] = d81
			ps210.OverlayValues[12] = d82
			ps210.OverlayValues[13] = d83
			ps210.OverlayValues[14] = d118
			ps210.OverlayValues[15] = d119
			ps210.OverlayValues[16] = d120
			ps210.OverlayValues[17] = d161
			ps210.OverlayValues[18] = d162
			ps210.OverlayValues[19] = d207
			ps210.OverlayValues[20] = d208
			return bbs[18].RenderPS(ps210)
		}
		if !ps.General {
			ps.General = true
			return bbs[9].RenderPS(ps)
		}
		ctx.EmitJump(d208.Condition, lbl18)
		if bbs[18].Rendered {
			ctx.EmitJmp(lbl19)
		}
		ctx.FreeDesc(&d207)
		snap211 := d0
		snap212 := d1
		snap213 := d2
		snap214 := d3
		snap215 := d4
		snap216 := d5
		snap217 := d24
		snap218 := d25
		snap219 := d26
		snap220 := d51
		snap221 := d52
		snap222 := d81
		snap223 := d82
		snap224 := d83
		snap225 := d118
		snap226 := d119
		snap227 := d120
		snap228 := d161
		snap229 := d162
		snap230 := d207
		snap231 := d208
		alloc232 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc232)
		d0 = snap211
		d1 = snap212
		d2 = snap213
		d3 = snap214
		d4 = snap215
		d5 = snap216
		d24 = snap217
		d25 = snap218
		d26 = snap219
		d51 = snap220
		d52 = snap221
		d81 = snap222
		d82 = snap223
		d83 = snap224
		d118 = snap225
		d119 = snap226
		d120 = snap227
		d161 = snap228
		d162 = snap229
		d207 = snap230
		d208 = snap231
		ctx.RestoreAllocState(alloc232)
		d0 = snap211
		d1 = snap212
		d2 = snap213
		d3 = snap214
		d4 = snap215
		d5 = snap216
		d24 = snap217
		d25 = snap218
		d26 = snap219
		d51 = snap220
		d52 = snap221
		d81 = snap222
		d82 = snap223
		d83 = snap224
		d118 = snap225
		d119 = snap226
		d120 = snap227
		d161 = snap228
		d162 = snap229
		d207 = snap230
		d208 = snap231
		ps233 := PhiState{General: true}
		ps233.OverlayValues = make([]JITValueDesc, 21)
		ps233.OverlayValues[0] = d0
		ps233.OverlayValues[1] = d1
		ps233.OverlayValues[2] = d2
		ps233.OverlayValues[3] = d3
		ps233.OverlayValues[4] = d4
		ps233.OverlayValues[5] = d5
		ps233.OverlayValues[6] = d24
		ps233.OverlayValues[7] = d25
		ps233.OverlayValues[8] = d26
		ps233.OverlayValues[9] = d51
		ps233.OverlayValues[10] = d52
		ps233.OverlayValues[11] = d81
		ps233.OverlayValues[12] = d82
		ps233.OverlayValues[13] = d83
		ps233.OverlayValues[14] = d118
		ps233.OverlayValues[15] = d119
		ps233.OverlayValues[16] = d120
		ps233.OverlayValues[17] = d161
		ps233.OverlayValues[18] = d162
		ps233.OverlayValues[19] = d207
		ps233.OverlayValues[20] = d208
		ps234 := PhiState{General: true}
		ps234.OverlayValues = make([]JITValueDesc, 21)
		ps234.OverlayValues[0] = d0
		ps234.OverlayValues[1] = d1
		ps234.OverlayValues[2] = d2
		ps234.OverlayValues[3] = d3
		ps234.OverlayValues[4] = d4
		ps234.OverlayValues[5] = d5
		ps234.OverlayValues[6] = d24
		ps234.OverlayValues[7] = d25
		ps234.OverlayValues[8] = d26
		ps234.OverlayValues[9] = d51
		ps234.OverlayValues[10] = d52
		ps234.OverlayValues[11] = d81
		ps234.OverlayValues[12] = d82
		ps234.OverlayValues[13] = d83
		ps234.OverlayValues[14] = d118
		ps234.OverlayValues[15] = d119
		ps234.OverlayValues[16] = d120
		ps234.OverlayValues[17] = d161
		ps234.OverlayValues[18] = d162
		ps234.OverlayValues[19] = d207
		ps234.OverlayValues[20] = d208
		snap235 := d0
		snap236 := d1
		snap237 := d2
		snap238 := d3
		snap239 := d4
		snap240 := d5
		snap241 := d24
		snap242 := d25
		snap243 := d26
		snap244 := d51
		snap245 := d52
		snap246 := d81
		snap247 := d82
		snap248 := d83
		snap249 := d118
		snap250 := d119
		snap251 := d120
		snap252 := d161
		snap253 := d162
		snap254 := d207
		snap255 := d208
		alloc256 := ctx.SnapshotAllocState()
		if !bbs[18].Rendered {
			bbs[18].RenderPS(ps234)
		}
		ctx.RestoreAllocState(alloc256)
		d0 = snap235
		d1 = snap236
		d2 = snap237
		d3 = snap238
		d4 = snap239
		d5 = snap240
		d24 = snap241
		d25 = snap242
		d26 = snap243
		d51 = snap244
		d52 = snap245
		d81 = snap246
		d82 = snap247
		d83 = snap248
		d118 = snap249
		d119 = snap250
		d120 = snap251
		d161 = snap252
		d162 = snap253
		d207 = snap254
		d208 = snap255
		if !bbs[17].Rendered {
			return bbs[17].RenderPS(ps233)
		}
		return result
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d257 JITValueDesc
		if d1.Loc == LocImm {
			d257 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x4))}
		} else {
			r7 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 4)
			d257 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
			ctx.BindReg(r7, &d257)
		}
		d258 = d257
		ctx.EnsureDesc(&d258)
		if d258.Loc != LocImm && d258.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d258.Loc == LocImm {
			if d258.Imm.Bool() {
				if ps.General {
				}
				ps259 := PhiState{General: ps.General}
				ps259.OverlayValues = make([]JITValueDesc, 23)
				ps259.OverlayValues[0] = d0
				ps259.OverlayValues[1] = d1
				ps259.OverlayValues[2] = d2
				ps259.OverlayValues[3] = d3
				ps259.OverlayValues[4] = d4
				ps259.OverlayValues[5] = d5
				ps259.OverlayValues[6] = d24
				ps259.OverlayValues[7] = d25
				ps259.OverlayValues[8] = d26
				ps259.OverlayValues[9] = d51
				ps259.OverlayValues[10] = d52
				ps259.OverlayValues[11] = d81
				ps259.OverlayValues[12] = d82
				ps259.OverlayValues[13] = d83
				ps259.OverlayValues[14] = d118
				ps259.OverlayValues[15] = d119
				ps259.OverlayValues[16] = d120
				ps259.OverlayValues[17] = d161
				ps259.OverlayValues[18] = d162
				ps259.OverlayValues[19] = d207
				ps259.OverlayValues[20] = d208
				ps259.OverlayValues[21] = d257
				ps259.OverlayValues[22] = d258
				return bbs[9].RenderPS(ps259)
			}
			if ps.General {
			}
			ps260 := PhiState{General: ps.General}
			ps260.OverlayValues = make([]JITValueDesc, 23)
			ps260.OverlayValues[0] = d0
			ps260.OverlayValues[1] = d1
			ps260.OverlayValues[2] = d2
			ps260.OverlayValues[3] = d3
			ps260.OverlayValues[4] = d4
			ps260.OverlayValues[5] = d5
			ps260.OverlayValues[6] = d24
			ps260.OverlayValues[7] = d25
			ps260.OverlayValues[8] = d26
			ps260.OverlayValues[9] = d51
			ps260.OverlayValues[10] = d52
			ps260.OverlayValues[11] = d81
			ps260.OverlayValues[12] = d82
			ps260.OverlayValues[13] = d83
			ps260.OverlayValues[14] = d118
			ps260.OverlayValues[15] = d119
			ps260.OverlayValues[16] = d120
			ps260.OverlayValues[17] = d161
			ps260.OverlayValues[18] = d162
			ps260.OverlayValues[19] = d207
			ps260.OverlayValues[20] = d208
			ps260.OverlayValues[21] = d257
			ps260.OverlayValues[22] = d258
			return bbs[16].RenderPS(ps260)
		}
		if !ps.General {
			ps.General = true
			return bbs[10].RenderPS(ps)
		}
		ctx.EmitJump(d258.Condition, lbl10)
		if bbs[16].Rendered {
			ctx.EmitJmp(lbl17)
		}
		ctx.FreeDesc(&d257)
		snap261 := d0
		snap262 := d1
		snap263 := d2
		snap264 := d3
		snap265 := d4
		snap266 := d5
		snap267 := d24
		snap268 := d25
		snap269 := d26
		snap270 := d51
		snap271 := d52
		snap272 := d81
		snap273 := d82
		snap274 := d83
		snap275 := d118
		snap276 := d119
		snap277 := d120
		snap278 := d161
		snap279 := d162
		snap280 := d207
		snap281 := d208
		snap282 := d257
		snap283 := d258
		alloc284 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc284)
		d0 = snap261
		d1 = snap262
		d2 = snap263
		d3 = snap264
		d4 = snap265
		d5 = snap266
		d24 = snap267
		d25 = snap268
		d26 = snap269
		d51 = snap270
		d52 = snap271
		d81 = snap272
		d82 = snap273
		d83 = snap274
		d118 = snap275
		d119 = snap276
		d120 = snap277
		d161 = snap278
		d162 = snap279
		d207 = snap280
		d208 = snap281
		d257 = snap282
		d258 = snap283
		ctx.RestoreAllocState(alloc284)
		d0 = snap261
		d1 = snap262
		d2 = snap263
		d3 = snap264
		d4 = snap265
		d5 = snap266
		d24 = snap267
		d25 = snap268
		d26 = snap269
		d51 = snap270
		d52 = snap271
		d81 = snap272
		d82 = snap273
		d83 = snap274
		d118 = snap275
		d119 = snap276
		d120 = snap277
		d161 = snap278
		d162 = snap279
		d207 = snap280
		d208 = snap281
		d257 = snap282
		d258 = snap283
		ps285 := PhiState{General: true}
		ps285.OverlayValues = make([]JITValueDesc, 23)
		ps285.OverlayValues[0] = d0
		ps285.OverlayValues[1] = d1
		ps285.OverlayValues[2] = d2
		ps285.OverlayValues[3] = d3
		ps285.OverlayValues[4] = d4
		ps285.OverlayValues[5] = d5
		ps285.OverlayValues[6] = d24
		ps285.OverlayValues[7] = d25
		ps285.OverlayValues[8] = d26
		ps285.OverlayValues[9] = d51
		ps285.OverlayValues[10] = d52
		ps285.OverlayValues[11] = d81
		ps285.OverlayValues[12] = d82
		ps285.OverlayValues[13] = d83
		ps285.OverlayValues[14] = d118
		ps285.OverlayValues[15] = d119
		ps285.OverlayValues[16] = d120
		ps285.OverlayValues[17] = d161
		ps285.OverlayValues[18] = d162
		ps285.OverlayValues[19] = d207
		ps285.OverlayValues[20] = d208
		ps285.OverlayValues[21] = d257
		ps285.OverlayValues[22] = d258
		ps286 := PhiState{General: true}
		ps286.OverlayValues = make([]JITValueDesc, 23)
		ps286.OverlayValues[0] = d0
		ps286.OverlayValues[1] = d1
		ps286.OverlayValues[2] = d2
		ps286.OverlayValues[3] = d3
		ps286.OverlayValues[4] = d4
		ps286.OverlayValues[5] = d5
		ps286.OverlayValues[6] = d24
		ps286.OverlayValues[7] = d25
		ps286.OverlayValues[8] = d26
		ps286.OverlayValues[9] = d51
		ps286.OverlayValues[10] = d52
		ps286.OverlayValues[11] = d81
		ps286.OverlayValues[12] = d82
		ps286.OverlayValues[13] = d83
		ps286.OverlayValues[14] = d118
		ps286.OverlayValues[15] = d119
		ps286.OverlayValues[16] = d120
		ps286.OverlayValues[17] = d161
		ps286.OverlayValues[18] = d162
		ps286.OverlayValues[19] = d207
		ps286.OverlayValues[20] = d208
		ps286.OverlayValues[21] = d257
		ps286.OverlayValues[22] = d258
		snap287 := d0
		snap288 := d1
		snap289 := d2
		snap290 := d3
		snap291 := d4
		snap292 := d5
		snap293 := d24
		snap294 := d25
		snap295 := d26
		snap296 := d51
		snap297 := d52
		snap298 := d81
		snap299 := d82
		snap300 := d83
		snap301 := d118
		snap302 := d119
		snap303 := d120
		snap304 := d161
		snap305 := d162
		snap306 := d207
		snap307 := d208
		snap308 := d257
		snap309 := d258
		alloc310 := ctx.SnapshotAllocState()
		if !bbs[16].Rendered {
			bbs[16].RenderPS(ps286)
		}
		ctx.RestoreAllocState(alloc310)
		d0 = snap287
		d1 = snap288
		d2 = snap289
		d3 = snap290
		d4 = snap291
		d5 = snap292
		d24 = snap293
		d25 = snap294
		d26 = snap295
		d51 = snap296
		d52 = snap297
		d81 = snap298
		d82 = snap299
		d83 = snap300
		d118 = snap301
		d119 = snap302
		d120 = snap303
		d161 = snap304
		d162 = snap305
		d207 = snap306
		d208 = snap307
		d257 = snap308
		d258 = snap309
		if !bbs[9].Rendered {
			return bbs[9].RenderPS(ps285)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		ctx.ReclaimUntrackedRegs()
		d312 = args[1]
		ctx.SyncDesc(&d312)
		if d312.Loc == LocMem {
			tmpScalar := JITValueDesc{Loc: LocReg, Type: d312.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d312.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d312 = tmpScalar
		}
		d312 = JITPrepareScmerGoArg(ctx, d312)
		if d312.Loc != LocRegPair && d312.Loc != LocStackPair && d312.Loc != LocInputPair {
			panic("jit: Scmer.String receiver not materialized as pair")
		}
		d311 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d312}, 2)
		ctx.EnsureDesc(&d311)
		if d311.Loc == LocImm {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d311.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			ctx.TrackImm(d311.Imm)
			ptrWord, _ := d311.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d311.Imm.String())))
			d311 = tmpPair
		} else if d311.Loc == LocReg {
			tmpPair := JITValueDesc{Loc: LocRegPair, Type: d311.Type, Reg: ctx.AllocRegExcept(d311.Reg), Reg2: ctx.AllocRegExcept(d311.Reg)}
			switch d311.Type {
			case tagBool:
				ctx.EmitMakeBool(tmpPair, d311)
			case tagInt:
				ctx.EmitMakeInt(tmpPair, d311)
			case tagFloat:
				ctx.EmitMakeFloat(tmpPair, d311)
			default:
				panic("jit: generic call arg scalar type unknown for 2-word value")
			}
			ctx.FreeDesc(&d311)
			d311 = tmpPair
		}
		if d311.Loc != LocRegPair && d311.Loc != LocStackPair && d311.Loc != LocInputPair {
			panic("jit: generic call arg expects 2-word value (ParseDateString arg0)")
		}
		ctx.SyncDesc(&d311)
		callResults313 := JITEmitGoCallResults(ctx, GoFuncAddr(ParseDateString), []JITValueDesc{d311}, []uint8{1, 1}, []uint8{0, 0})
		d314 = callResults313[0]
		_ = d314
		d315 = callResults313[1]
		_ = d315
		ctx.StabilizeDescForControlFlow(&d314)
		d316 = d315
		ctx.EnsureDesc(&d316)
		if d316.Loc != LocImm && d316.Loc != LocReg {
			panic("jit: If condition is neither LocImm nor LocReg")
		}
		if d316.Loc == LocImm {
			if d316.Imm.Bool() {
				if ps.General {
				}
				ps317 := PhiState{General: ps.General}
				ps317.OverlayValues = make([]JITValueDesc, 28)
				ps317.OverlayValues[0] = d0
				ps317.OverlayValues[1] = d1
				ps317.OverlayValues[2] = d2
				ps317.OverlayValues[3] = d3
				ps317.OverlayValues[4] = d4
				ps317.OverlayValues[5] = d5
				ps317.OverlayValues[6] = d24
				ps317.OverlayValues[7] = d25
				ps317.OverlayValues[8] = d26
				ps317.OverlayValues[9] = d51
				ps317.OverlayValues[10] = d52
				ps317.OverlayValues[11] = d81
				ps317.OverlayValues[12] = d82
				ps317.OverlayValues[13] = d83
				ps317.OverlayValues[14] = d118
				ps317.OverlayValues[15] = d119
				ps317.OverlayValues[16] = d120
				ps317.OverlayValues[17] = d161
				ps317.OverlayValues[18] = d162
				ps317.OverlayValues[19] = d207
				ps317.OverlayValues[20] = d208
				ps317.OverlayValues[21] = d257
				ps317.OverlayValues[22] = d258
				ps317.OverlayValues[23] = d311
				ps317.OverlayValues[24] = d312
				ps317.OverlayValues[25] = d314
				ps317.OverlayValues[26] = d315
				ps317.OverlayValues[27] = d316
				return bbs[14].RenderPS(ps317)
			}
			if ps.General {
			}
			ps318 := PhiState{General: ps.General}
			ps318.OverlayValues = make([]JITValueDesc, 28)
			ps318.OverlayValues[0] = d0
			ps318.OverlayValues[1] = d1
			ps318.OverlayValues[2] = d2
			ps318.OverlayValues[3] = d3
			ps318.OverlayValues[4] = d4
			ps318.OverlayValues[5] = d5
			ps318.OverlayValues[6] = d24
			ps318.OverlayValues[7] = d25
			ps318.OverlayValues[8] = d26
			ps318.OverlayValues[9] = d51
			ps318.OverlayValues[10] = d52
			ps318.OverlayValues[11] = d81
			ps318.OverlayValues[12] = d82
			ps318.OverlayValues[13] = d83
			ps318.OverlayValues[14] = d118
			ps318.OverlayValues[15] = d119
			ps318.OverlayValues[16] = d120
			ps318.OverlayValues[17] = d161
			ps318.OverlayValues[18] = d162
			ps318.OverlayValues[19] = d207
			ps318.OverlayValues[20] = d208
			ps318.OverlayValues[21] = d257
			ps318.OverlayValues[22] = d258
			ps318.OverlayValues[23] = d311
			ps318.OverlayValues[24] = d312
			ps318.OverlayValues[25] = d314
			ps318.OverlayValues[26] = d315
			ps318.OverlayValues[27] = d316
			return bbs[12].RenderPS(ps318)
		}
		if !ps.General {
			ps.General = true
			return bbs[11].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d316.Reg, 0)
		ctx.EmitJump(CondNotEqual, lbl15)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		snap319 := d0
		snap320 := d1
		snap321 := d2
		snap322 := d3
		snap323 := d4
		snap324 := d5
		snap325 := d24
		snap326 := d25
		snap327 := d26
		snap328 := d51
		snap329 := d52
		snap330 := d81
		snap331 := d82
		snap332 := d83
		snap333 := d118
		snap334 := d119
		snap335 := d120
		snap336 := d161
		snap337 := d162
		snap338 := d207
		snap339 := d208
		snap340 := d257
		snap341 := d258
		snap342 := d311
		snap343 := d312
		snap344 := d314
		snap345 := d315
		snap346 := d316
		alloc347 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc347)
		d0 = snap319
		d1 = snap320
		d2 = snap321
		d3 = snap322
		d4 = snap323
		d5 = snap324
		d24 = snap325
		d25 = snap326
		d26 = snap327
		d51 = snap328
		d52 = snap329
		d81 = snap330
		d82 = snap331
		d83 = snap332
		d118 = snap333
		d119 = snap334
		d120 = snap335
		d161 = snap336
		d162 = snap337
		d207 = snap338
		d208 = snap339
		d257 = snap340
		d258 = snap341
		d311 = snap342
		d312 = snap343
		d314 = snap344
		d315 = snap345
		d316 = snap346
		ctx.RestoreAllocState(alloc347)
		d0 = snap319
		d1 = snap320
		d2 = snap321
		d3 = snap322
		d4 = snap323
		d5 = snap324
		d24 = snap325
		d25 = snap326
		d26 = snap327
		d51 = snap328
		d52 = snap329
		d81 = snap330
		d82 = snap331
		d83 = snap332
		d118 = snap333
		d119 = snap334
		d120 = snap335
		d161 = snap336
		d162 = snap337
		d207 = snap338
		d208 = snap339
		d257 = snap340
		d258 = snap341
		d311 = snap342
		d312 = snap343
		d314 = snap344
		d315 = snap345
		d316 = snap346
		ps348 := PhiState{General: true}
		ps348.OverlayValues = make([]JITValueDesc, 28)
		ps348.OverlayValues[0] = d0
		ps348.OverlayValues[1] = d1
		ps348.OverlayValues[2] = d2
		ps348.OverlayValues[3] = d3
		ps348.OverlayValues[4] = d4
		ps348.OverlayValues[5] = d5
		ps348.OverlayValues[6] = d24
		ps348.OverlayValues[7] = d25
		ps348.OverlayValues[8] = d26
		ps348.OverlayValues[9] = d51
		ps348.OverlayValues[10] = d52
		ps348.OverlayValues[11] = d81
		ps348.OverlayValues[12] = d82
		ps348.OverlayValues[13] = d83
		ps348.OverlayValues[14] = d118
		ps348.OverlayValues[15] = d119
		ps348.OverlayValues[16] = d120
		ps348.OverlayValues[17] = d161
		ps348.OverlayValues[18] = d162
		ps348.OverlayValues[19] = d207
		ps348.OverlayValues[20] = d208
		ps348.OverlayValues[21] = d257
		ps348.OverlayValues[22] = d258
		ps348.OverlayValues[23] = d311
		ps348.OverlayValues[24] = d312
		ps348.OverlayValues[25] = d314
		ps348.OverlayValues[26] = d315
		ps348.OverlayValues[27] = d316
		ps349 := PhiState{General: true}
		ps349.OverlayValues = make([]JITValueDesc, 28)
		ps349.OverlayValues[0] = d0
		ps349.OverlayValues[1] = d1
		ps349.OverlayValues[2] = d2
		ps349.OverlayValues[3] = d3
		ps349.OverlayValues[4] = d4
		ps349.OverlayValues[5] = d5
		ps349.OverlayValues[6] = d24
		ps349.OverlayValues[7] = d25
		ps349.OverlayValues[8] = d26
		ps349.OverlayValues[9] = d51
		ps349.OverlayValues[10] = d52
		ps349.OverlayValues[11] = d81
		ps349.OverlayValues[12] = d82
		ps349.OverlayValues[13] = d83
		ps349.OverlayValues[14] = d118
		ps349.OverlayValues[15] = d119
		ps349.OverlayValues[16] = d120
		ps349.OverlayValues[17] = d161
		ps349.OverlayValues[18] = d162
		ps349.OverlayValues[19] = d207
		ps349.OverlayValues[20] = d208
		ps349.OverlayValues[21] = d257
		ps349.OverlayValues[22] = d258
		ps349.OverlayValues[23] = d311
		ps349.OverlayValues[24] = d312
		ps349.OverlayValues[25] = d314
		ps349.OverlayValues[26] = d315
		ps349.OverlayValues[27] = d316
		snap350 := d0
		snap351 := d1
		snap352 := d2
		snap353 := d3
		snap354 := d4
		snap355 := d5
		snap356 := d24
		snap357 := d25
		snap358 := d26
		snap359 := d51
		snap360 := d52
		snap361 := d81
		snap362 := d82
		snap363 := d83
		snap364 := d118
		snap365 := d119
		snap366 := d120
		snap367 := d161
		snap368 := d162
		snap369 := d207
		snap370 := d208
		snap371 := d257
		snap372 := d258
		snap373 := d311
		snap374 := d312
		snap375 := d314
		snap376 := d315
		snap377 := d316
		alloc378 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps349)
		}
		ctx.RestoreAllocState(alloc378)
		d0 = snap350
		d1 = snap351
		d2 = snap352
		d3 = snap353
		d4 = snap354
		d5 = snap355
		d24 = snap356
		d25 = snap357
		d26 = snap358
		d51 = snap359
		d52 = snap360
		d81 = snap361
		d82 = snap362
		d83 = snap363
		d118 = snap364
		d119 = snap365
		d120 = snap366
		d161 = snap367
		d162 = snap368
		d207 = snap369
		d208 = snap370
		d257 = snap371
		d258 = snap372
		d311 = snap373
		d312 = snap374
		d314 = snap375
		d315 = snap376
		d316 = snap377
		if !bbs[14].Rendered {
			return bbs[14].RenderPS(ps348)
		}
		return result
		ctx.FreeDesc(&d315)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		ctx.ReclaimUntrackedRegs()
		var d379 JITValueDesc
		if args[0].Loc == LocImm {
			d379 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d379 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d379)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d379 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d379)
		} else {
			d379 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d379.Type = tagInt
			ctx.BindReg(d379.Reg, &d379)
		}
		ctx.EnsureDesc(&d379)
		ctx.EnsureDesc(&d379)
		var d380 JITValueDesc
		if d379.Loc == LocImm {
			d380 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d379.Imm.Int()))}
		} else {
			var r8 Reg
			r8 = d379.Reg
			d379.Loc = LocNone
			ctx.EmitCvtInt64ToFloat64(RegX0, r8)
			d380 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
			ctx.BindReg(r8, &d380)
		}
		ctx.FreeDesc(&d379)
		d381 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d380)
		resultTarget382 := false
		_ = resultTarget382
		ctx.EnsureDesc(&d381)
		ctx.EnsureDescsTogether(&d380, &d381)
		var d383 JITValueDesc
		if d380.Loc == LocImm && d381.Loc == LocImm {
			d383 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d380.Imm.Float() < d381.Imm.Float())}
		} else if d381.Loc == LocImm {
			var r9 Reg
			if result.Loc == LocReg && result.Reg != d380.Reg {
				r9 = result.Reg
				resultTarget382 = true
			} else {
				r9 = ctx.AllocRegExcept(d380.Reg)
			}
			_, yBits := d381.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r9, d380.Reg, RegR11, CondSignedLess)
			d383 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
			ctx.BindReg(r9, &d383)
		} else if d380.Loc == LocImm {
			var r10 Reg
			if result.Loc == LocReg && result.Reg != d381.Reg {
				r10 = result.Reg
				resultTarget382 = true
			} else {
				r10 = ctx.AllocRegExcept(d381.Reg)
			}
			_, xBits := d380.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r10, RegR11, d381.Reg, CondSignedLess)
			d383 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
			ctx.BindReg(r10, &d383)
		} else {
			var r11 Reg
			if result.Loc == LocReg && result.Reg != d380.Reg && result.Reg != d381.Reg {
				r11 = result.Reg
				resultTarget382 = true
			} else {
				r11 = ctx.AllocRegExcept(d380.Reg, d381.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r11, d380.Reg, d381.Reg, CondSignedLess)
			d383 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
			ctx.BindReg(r11, &d383)
		}
		ctx.FreeDesc(&d380)
		ctx.FreeDesc(&d381)
		ctx.SyncDesc(&d383)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d383) {
			return d383
		}
		ctx.EnsureDesc(&d383)
		ctx.EmitMovToReg(result.Reg, d383)
		result.Type = d383.Type
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		var d384 JITValueDesc
		if d3.Loc == LocImm {
			d384 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d3.Imm.Int()) == uint64(0x2))}
		} else {
			r12 := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitCmpRegImm32(d3.Reg, 2)
			d384 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondEqual}
			ctx.BindReg(r12, &d384)
		}
		d385 = d384
		ctx.EnsureDesc(&d385)
		if d385.Loc != LocImm && d385.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d385.Loc == LocImm {
			if d385.Imm.Bool() {
				if ps.General {
				}
				ps386 := PhiState{General: ps.General}
				ps386.OverlayValues = make([]JITValueDesc, 34)
				ps386.OverlayValues[0] = d0
				ps386.OverlayValues[1] = d1
				ps386.OverlayValues[2] = d2
				ps386.OverlayValues[3] = d3
				ps386.OverlayValues[4] = d4
				ps386.OverlayValues[5] = d5
				ps386.OverlayValues[6] = d24
				ps386.OverlayValues[7] = d25
				ps386.OverlayValues[8] = d26
				ps386.OverlayValues[9] = d51
				ps386.OverlayValues[10] = d52
				ps386.OverlayValues[11] = d81
				ps386.OverlayValues[12] = d82
				ps386.OverlayValues[13] = d83
				ps386.OverlayValues[14] = d118
				ps386.OverlayValues[15] = d119
				ps386.OverlayValues[16] = d120
				ps386.OverlayValues[17] = d161
				ps386.OverlayValues[18] = d162
				ps386.OverlayValues[19] = d207
				ps386.OverlayValues[20] = d208
				ps386.OverlayValues[21] = d257
				ps386.OverlayValues[22] = d258
				ps386.OverlayValues[23] = d311
				ps386.OverlayValues[24] = d312
				ps386.OverlayValues[25] = d314
				ps386.OverlayValues[26] = d315
				ps386.OverlayValues[27] = d316
				ps386.OverlayValues[28] = d379
				ps386.OverlayValues[29] = d380
				ps386.OverlayValues[30] = d381
				ps386.OverlayValues[31] = d383
				ps386.OverlayValues[32] = d384
				ps386.OverlayValues[33] = d385
				return bbs[11].RenderPS(ps386)
			}
			if ps.General {
			}
			ps387 := PhiState{General: ps.General}
			ps387.OverlayValues = make([]JITValueDesc, 34)
			ps387.OverlayValues[0] = d0
			ps387.OverlayValues[1] = d1
			ps387.OverlayValues[2] = d2
			ps387.OverlayValues[3] = d3
			ps387.OverlayValues[4] = d4
			ps387.OverlayValues[5] = d5
			ps387.OverlayValues[6] = d24
			ps387.OverlayValues[7] = d25
			ps387.OverlayValues[8] = d26
			ps387.OverlayValues[9] = d51
			ps387.OverlayValues[10] = d52
			ps387.OverlayValues[11] = d81
			ps387.OverlayValues[12] = d82
			ps387.OverlayValues[13] = d83
			ps387.OverlayValues[14] = d118
			ps387.OverlayValues[15] = d119
			ps387.OverlayValues[16] = d120
			ps387.OverlayValues[17] = d161
			ps387.OverlayValues[18] = d162
			ps387.OverlayValues[19] = d207
			ps387.OverlayValues[20] = d208
			ps387.OverlayValues[21] = d257
			ps387.OverlayValues[22] = d258
			ps387.OverlayValues[23] = d311
			ps387.OverlayValues[24] = d312
			ps387.OverlayValues[25] = d314
			ps387.OverlayValues[26] = d315
			ps387.OverlayValues[27] = d316
			ps387.OverlayValues[28] = d379
			ps387.OverlayValues[29] = d380
			ps387.OverlayValues[30] = d381
			ps387.OverlayValues[31] = d383
			ps387.OverlayValues[32] = d384
			ps387.OverlayValues[33] = d385
			return bbs[12].RenderPS(ps387)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		ctx.EmitJump(d385.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d384)
		snap388 := d0
		snap389 := d1
		snap390 := d2
		snap391 := d3
		snap392 := d4
		snap393 := d5
		snap394 := d24
		snap395 := d25
		snap396 := d26
		snap397 := d51
		snap398 := d52
		snap399 := d81
		snap400 := d82
		snap401 := d83
		snap402 := d118
		snap403 := d119
		snap404 := d120
		snap405 := d161
		snap406 := d162
		snap407 := d207
		snap408 := d208
		snap409 := d257
		snap410 := d258
		snap411 := d311
		snap412 := d312
		snap413 := d314
		snap414 := d315
		snap415 := d316
		snap416 := d379
		snap417 := d380
		snap418 := d381
		snap419 := d383
		snap420 := d384
		snap421 := d385
		alloc422 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc422)
		d0 = snap388
		d1 = snap389
		d2 = snap390
		d3 = snap391
		d4 = snap392
		d5 = snap393
		d24 = snap394
		d25 = snap395
		d26 = snap396
		d51 = snap397
		d52 = snap398
		d81 = snap399
		d82 = snap400
		d83 = snap401
		d118 = snap402
		d119 = snap403
		d120 = snap404
		d161 = snap405
		d162 = snap406
		d207 = snap407
		d208 = snap408
		d257 = snap409
		d258 = snap410
		d311 = snap411
		d312 = snap412
		d314 = snap413
		d315 = snap414
		d316 = snap415
		d379 = snap416
		d380 = snap417
		d381 = snap418
		d383 = snap419
		d384 = snap420
		d385 = snap421
		ctx.RestoreAllocState(alloc422)
		d0 = snap388
		d1 = snap389
		d2 = snap390
		d3 = snap391
		d4 = snap392
		d5 = snap393
		d24 = snap394
		d25 = snap395
		d26 = snap396
		d51 = snap397
		d52 = snap398
		d81 = snap399
		d82 = snap400
		d83 = snap401
		d118 = snap402
		d119 = snap403
		d120 = snap404
		d161 = snap405
		d162 = snap406
		d207 = snap407
		d208 = snap408
		d257 = snap409
		d258 = snap410
		d311 = snap411
		d312 = snap412
		d314 = snap413
		d315 = snap414
		d316 = snap415
		d379 = snap416
		d380 = snap417
		d381 = snap418
		d383 = snap419
		d384 = snap420
		d385 = snap421
		ps423 := PhiState{General: true}
		ps423.OverlayValues = make([]JITValueDesc, 34)
		ps423.OverlayValues[0] = d0
		ps423.OverlayValues[1] = d1
		ps423.OverlayValues[2] = d2
		ps423.OverlayValues[3] = d3
		ps423.OverlayValues[4] = d4
		ps423.OverlayValues[5] = d5
		ps423.OverlayValues[6] = d24
		ps423.OverlayValues[7] = d25
		ps423.OverlayValues[8] = d26
		ps423.OverlayValues[9] = d51
		ps423.OverlayValues[10] = d52
		ps423.OverlayValues[11] = d81
		ps423.OverlayValues[12] = d82
		ps423.OverlayValues[13] = d83
		ps423.OverlayValues[14] = d118
		ps423.OverlayValues[15] = d119
		ps423.OverlayValues[16] = d120
		ps423.OverlayValues[17] = d161
		ps423.OverlayValues[18] = d162
		ps423.OverlayValues[19] = d207
		ps423.OverlayValues[20] = d208
		ps423.OverlayValues[21] = d257
		ps423.OverlayValues[22] = d258
		ps423.OverlayValues[23] = d311
		ps423.OverlayValues[24] = d312
		ps423.OverlayValues[25] = d314
		ps423.OverlayValues[26] = d315
		ps423.OverlayValues[27] = d316
		ps423.OverlayValues[28] = d379
		ps423.OverlayValues[29] = d380
		ps423.OverlayValues[30] = d381
		ps423.OverlayValues[31] = d383
		ps423.OverlayValues[32] = d384
		ps423.OverlayValues[33] = d385
		ps424 := PhiState{General: true}
		ps424.OverlayValues = make([]JITValueDesc, 34)
		ps424.OverlayValues[0] = d0
		ps424.OverlayValues[1] = d1
		ps424.OverlayValues[2] = d2
		ps424.OverlayValues[3] = d3
		ps424.OverlayValues[4] = d4
		ps424.OverlayValues[5] = d5
		ps424.OverlayValues[6] = d24
		ps424.OverlayValues[7] = d25
		ps424.OverlayValues[8] = d26
		ps424.OverlayValues[9] = d51
		ps424.OverlayValues[10] = d52
		ps424.OverlayValues[11] = d81
		ps424.OverlayValues[12] = d82
		ps424.OverlayValues[13] = d83
		ps424.OverlayValues[14] = d118
		ps424.OverlayValues[15] = d119
		ps424.OverlayValues[16] = d120
		ps424.OverlayValues[17] = d161
		ps424.OverlayValues[18] = d162
		ps424.OverlayValues[19] = d207
		ps424.OverlayValues[20] = d208
		ps424.OverlayValues[21] = d257
		ps424.OverlayValues[22] = d258
		ps424.OverlayValues[23] = d311
		ps424.OverlayValues[24] = d312
		ps424.OverlayValues[25] = d314
		ps424.OverlayValues[26] = d315
		ps424.OverlayValues[27] = d316
		ps424.OverlayValues[28] = d379
		ps424.OverlayValues[29] = d380
		ps424.OverlayValues[30] = d381
		ps424.OverlayValues[31] = d383
		ps424.OverlayValues[32] = d384
		ps424.OverlayValues[33] = d385
		snap425 := d0
		snap426 := d1
		snap427 := d2
		snap428 := d3
		snap429 := d4
		snap430 := d5
		snap431 := d24
		snap432 := d25
		snap433 := d26
		snap434 := d51
		snap435 := d52
		snap436 := d81
		snap437 := d82
		snap438 := d83
		snap439 := d118
		snap440 := d119
		snap441 := d120
		snap442 := d161
		snap443 := d162
		snap444 := d207
		snap445 := d208
		snap446 := d257
		snap447 := d258
		snap448 := d311
		snap449 := d312
		snap450 := d314
		snap451 := d315
		snap452 := d316
		snap453 := d379
		snap454 := d380
		snap455 := d381
		snap456 := d383
		snap457 := d384
		snap458 := d385
		alloc459 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps424)
		}
		ctx.RestoreAllocState(alloc459)
		d0 = snap425
		d1 = snap426
		d2 = snap427
		d3 = snap428
		d4 = snap429
		d5 = snap430
		d24 = snap431
		d25 = snap432
		d26 = snap433
		d51 = snap434
		d52 = snap435
		d81 = snap436
		d82 = snap437
		d83 = snap438
		d118 = snap439
		d119 = snap440
		d120 = snap441
		d161 = snap442
		d162 = snap443
		d207 = snap444
		d208 = snap445
		d257 = snap446
		d258 = snap447
		d311 = snap448
		d312 = snap449
		d314 = snap450
		d315 = snap451
		d316 = snap452
		d379 = snap453
		d380 = snap454
		d381 = snap455
		d383 = snap456
		d384 = snap457
		d385 = snap458
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps423)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		ctx.ReclaimUntrackedRegs()
		var d460 JITValueDesc
		if args[0].Loc == LocImm {
			d460 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d460 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d460)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d460 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d460)
		} else {
			d460 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d460.Type = tagInt
			ctx.BindReg(d460.Reg, &d460)
		}
		ctx.EnsureDesc(&d460)
		resultTarget461 := false
		_ = resultTarget461
		ctx.EnsureDesc(&d314)
		ctx.EnsureDescsTogether(&d460, &d314)
		var d462 JITValueDesc
		if d460.Loc == LocImm && d314.Loc == LocImm {
			d462 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d460.Imm.Int() < d314.Imm.Int())}
		} else if d314.Loc == LocImm {
			r13 := ctx.AllocReg()
			if d314.Imm.Int() >= -2147483648 && d314.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d460.Reg, int32(d314.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(RegR11, uint64(d314.Imm.Int()))
				ctx.EmitCmpInt64(d460.Reg, RegR11)
			}
			d462 = ctx.DeferBooleanFlags(r13, CondSignedLess)
			ctx.BindReg(r13, &d462)
		} else if d460.Loc == LocImm {
			r14 := ctx.AllocReg()
			ctx.EmitMovRegImm64(RegR11, uint64(d460.Imm.Int()))
			ctx.EmitCmpInt64(RegR11, d314.Reg)
			d462 = ctx.DeferBooleanFlags(r14, CondSignedLess)
			ctx.BindReg(r14, &d462)
		} else {
			r15 := ctx.AllocReg()
			ctx.EmitCmpInt64(d460.Reg, d314.Reg)
			d462 = ctx.DeferBooleanFlags(r15, CondSignedLess)
			ctx.BindReg(r15, &d462)
		}
		ctx.FreeDesc(&d460)
		ctx.SyncDesc(&d462)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d462) {
			return d462
		}
		ctx.EnsureDesc(&d462)
		ctx.EmitMovToReg(result.Reg, d462)
		result.Type = d462.Type
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		ctx.ReclaimUntrackedRegs()
		d463 = ctx.EmitFloatDesc(args[0])
		d464 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d463)
		resultTarget465 := false
		_ = resultTarget465
		ctx.EnsureDesc(&d464)
		ctx.EnsureDescsTogether(&d463, &d464)
		var d466 JITValueDesc
		if d463.Loc == LocImm && d464.Loc == LocImm {
			d466 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d463.Imm.Float() < d464.Imm.Float())}
		} else if d464.Loc == LocImm {
			var r16 Reg
			if result.Loc == LocReg && result.Reg != d463.Reg {
				r16 = result.Reg
				resultTarget465 = true
			} else {
				r16 = ctx.AllocRegExcept(d463.Reg)
			}
			_, yBits := d464.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r16, d463.Reg, RegR11, CondSignedLess)
			d466 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
			ctx.BindReg(r16, &d466)
		} else if d463.Loc == LocImm {
			var r17 Reg
			if result.Loc == LocReg && result.Reg != d464.Reg {
				r17 = result.Reg
				resultTarget465 = true
			} else {
				r17 = ctx.AllocRegExcept(d464.Reg)
			}
			_, xBits := d463.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r17, RegR11, d464.Reg, CondSignedLess)
			d466 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
			ctx.BindReg(r17, &d466)
		} else {
			var r18 Reg
			if result.Loc == LocReg && result.Reg != d463.Reg && result.Reg != d464.Reg {
				r18 = result.Reg
				resultTarget465 = true
			} else {
				r18 = ctx.AllocRegExcept(d463.Reg, d464.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r18, d463.Reg, d464.Reg, CondSignedLess)
			d466 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
			ctx.BindReg(r18, &d466)
		}
		ctx.FreeDesc(&d463)
		ctx.FreeDesc(&d464)
		ctx.SyncDesc(&d466)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d466) {
			return d466
		}
		ctx.EnsureDesc(&d466)
		ctx.EmitMovToReg(result.Reg, d466)
		result.Type = d466.Type
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d467 JITValueDesc
		if d1.Loc == LocImm {
			d467 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x3))}
		} else {
			r19 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 3)
			d467 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondEqual}
			ctx.BindReg(r19, &d467)
		}
		d468 = d467
		ctx.EnsureDesc(&d468)
		if d468.Loc != LocImm && d468.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d468.Loc == LocImm {
			if d468.Imm.Bool() {
				if ps.General {
				}
				ps469 := PhiState{General: ps.General}
				ps469.OverlayValues = make([]JITValueDesc, 41)
				ps469.OverlayValues[0] = d0
				ps469.OverlayValues[1] = d1
				ps469.OverlayValues[2] = d2
				ps469.OverlayValues[3] = d3
				ps469.OverlayValues[4] = d4
				ps469.OverlayValues[5] = d5
				ps469.OverlayValues[6] = d24
				ps469.OverlayValues[7] = d25
				ps469.OverlayValues[8] = d26
				ps469.OverlayValues[9] = d51
				ps469.OverlayValues[10] = d52
				ps469.OverlayValues[11] = d81
				ps469.OverlayValues[12] = d82
				ps469.OverlayValues[13] = d83
				ps469.OverlayValues[14] = d118
				ps469.OverlayValues[15] = d119
				ps469.OverlayValues[16] = d120
				ps469.OverlayValues[17] = d161
				ps469.OverlayValues[18] = d162
				ps469.OverlayValues[19] = d207
				ps469.OverlayValues[20] = d208
				ps469.OverlayValues[21] = d257
				ps469.OverlayValues[22] = d258
				ps469.OverlayValues[23] = d311
				ps469.OverlayValues[24] = d312
				ps469.OverlayValues[25] = d314
				ps469.OverlayValues[26] = d315
				ps469.OverlayValues[27] = d316
				ps469.OverlayValues[28] = d379
				ps469.OverlayValues[29] = d380
				ps469.OverlayValues[30] = d381
				ps469.OverlayValues[31] = d383
				ps469.OverlayValues[32] = d384
				ps469.OverlayValues[33] = d385
				ps469.OverlayValues[34] = d460
				ps469.OverlayValues[35] = d462
				ps469.OverlayValues[36] = d463
				ps469.OverlayValues[37] = d464
				ps469.OverlayValues[38] = d466
				ps469.OverlayValues[39] = d467
				ps469.OverlayValues[40] = d468
				return bbs[15].RenderPS(ps469)
			}
			if ps.General {
			}
			ps470 := PhiState{General: ps.General}
			ps470.OverlayValues = make([]JITValueDesc, 41)
			ps470.OverlayValues[0] = d0
			ps470.OverlayValues[1] = d1
			ps470.OverlayValues[2] = d2
			ps470.OverlayValues[3] = d3
			ps470.OverlayValues[4] = d4
			ps470.OverlayValues[5] = d5
			ps470.OverlayValues[6] = d24
			ps470.OverlayValues[7] = d25
			ps470.OverlayValues[8] = d26
			ps470.OverlayValues[9] = d51
			ps470.OverlayValues[10] = d52
			ps470.OverlayValues[11] = d81
			ps470.OverlayValues[12] = d82
			ps470.OverlayValues[13] = d83
			ps470.OverlayValues[14] = d118
			ps470.OverlayValues[15] = d119
			ps470.OverlayValues[16] = d120
			ps470.OverlayValues[17] = d161
			ps470.OverlayValues[18] = d162
			ps470.OverlayValues[19] = d207
			ps470.OverlayValues[20] = d208
			ps470.OverlayValues[21] = d257
			ps470.OverlayValues[22] = d258
			ps470.OverlayValues[23] = d311
			ps470.OverlayValues[24] = d312
			ps470.OverlayValues[25] = d314
			ps470.OverlayValues[26] = d315
			ps470.OverlayValues[27] = d316
			ps470.OverlayValues[28] = d379
			ps470.OverlayValues[29] = d380
			ps470.OverlayValues[30] = d381
			ps470.OverlayValues[31] = d383
			ps470.OverlayValues[32] = d384
			ps470.OverlayValues[33] = d385
			ps470.OverlayValues[34] = d460
			ps470.OverlayValues[35] = d462
			ps470.OverlayValues[36] = d463
			ps470.OverlayValues[37] = d464
			ps470.OverlayValues[38] = d466
			ps470.OverlayValues[39] = d467
			ps470.OverlayValues[40] = d468
			return bbs[20].RenderPS(ps470)
		}
		if !ps.General {
			ps.General = true
			return bbs[16].RenderPS(ps)
		}
		ctx.EmitJump(d468.Condition, lbl16)
		if bbs[20].Rendered {
			ctx.EmitJmp(lbl21)
		}
		ctx.FreeDesc(&d467)
		snap471 := d0
		snap472 := d1
		snap473 := d2
		snap474 := d3
		snap475 := d4
		snap476 := d5
		snap477 := d24
		snap478 := d25
		snap479 := d26
		snap480 := d51
		snap481 := d52
		snap482 := d81
		snap483 := d82
		snap484 := d83
		snap485 := d118
		snap486 := d119
		snap487 := d120
		snap488 := d161
		snap489 := d162
		snap490 := d207
		snap491 := d208
		snap492 := d257
		snap493 := d258
		snap494 := d311
		snap495 := d312
		snap496 := d314
		snap497 := d315
		snap498 := d316
		snap499 := d379
		snap500 := d380
		snap501 := d381
		snap502 := d383
		snap503 := d384
		snap504 := d385
		snap505 := d460
		snap506 := d462
		snap507 := d463
		snap508 := d464
		snap509 := d466
		snap510 := d467
		snap511 := d468
		alloc512 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc512)
		d0 = snap471
		d1 = snap472
		d2 = snap473
		d3 = snap474
		d4 = snap475
		d5 = snap476
		d24 = snap477
		d25 = snap478
		d26 = snap479
		d51 = snap480
		d52 = snap481
		d81 = snap482
		d82 = snap483
		d83 = snap484
		d118 = snap485
		d119 = snap486
		d120 = snap487
		d161 = snap488
		d162 = snap489
		d207 = snap490
		d208 = snap491
		d257 = snap492
		d258 = snap493
		d311 = snap494
		d312 = snap495
		d314 = snap496
		d315 = snap497
		d316 = snap498
		d379 = snap499
		d380 = snap500
		d381 = snap501
		d383 = snap502
		d384 = snap503
		d385 = snap504
		d460 = snap505
		d462 = snap506
		d463 = snap507
		d464 = snap508
		d466 = snap509
		d467 = snap510
		d468 = snap511
		ctx.RestoreAllocState(alloc512)
		d0 = snap471
		d1 = snap472
		d2 = snap473
		d3 = snap474
		d4 = snap475
		d5 = snap476
		d24 = snap477
		d25 = snap478
		d26 = snap479
		d51 = snap480
		d52 = snap481
		d81 = snap482
		d82 = snap483
		d83 = snap484
		d118 = snap485
		d119 = snap486
		d120 = snap487
		d161 = snap488
		d162 = snap489
		d207 = snap490
		d208 = snap491
		d257 = snap492
		d258 = snap493
		d311 = snap494
		d312 = snap495
		d314 = snap496
		d315 = snap497
		d316 = snap498
		d379 = snap499
		d380 = snap500
		d381 = snap501
		d383 = snap502
		d384 = snap503
		d385 = snap504
		d460 = snap505
		d462 = snap506
		d463 = snap507
		d464 = snap508
		d466 = snap509
		d467 = snap510
		d468 = snap511
		ps513 := PhiState{General: true}
		ps513.OverlayValues = make([]JITValueDesc, 41)
		ps513.OverlayValues[0] = d0
		ps513.OverlayValues[1] = d1
		ps513.OverlayValues[2] = d2
		ps513.OverlayValues[3] = d3
		ps513.OverlayValues[4] = d4
		ps513.OverlayValues[5] = d5
		ps513.OverlayValues[6] = d24
		ps513.OverlayValues[7] = d25
		ps513.OverlayValues[8] = d26
		ps513.OverlayValues[9] = d51
		ps513.OverlayValues[10] = d52
		ps513.OverlayValues[11] = d81
		ps513.OverlayValues[12] = d82
		ps513.OverlayValues[13] = d83
		ps513.OverlayValues[14] = d118
		ps513.OverlayValues[15] = d119
		ps513.OverlayValues[16] = d120
		ps513.OverlayValues[17] = d161
		ps513.OverlayValues[18] = d162
		ps513.OverlayValues[19] = d207
		ps513.OverlayValues[20] = d208
		ps513.OverlayValues[21] = d257
		ps513.OverlayValues[22] = d258
		ps513.OverlayValues[23] = d311
		ps513.OverlayValues[24] = d312
		ps513.OverlayValues[25] = d314
		ps513.OverlayValues[26] = d315
		ps513.OverlayValues[27] = d316
		ps513.OverlayValues[28] = d379
		ps513.OverlayValues[29] = d380
		ps513.OverlayValues[30] = d381
		ps513.OverlayValues[31] = d383
		ps513.OverlayValues[32] = d384
		ps513.OverlayValues[33] = d385
		ps513.OverlayValues[34] = d460
		ps513.OverlayValues[35] = d462
		ps513.OverlayValues[36] = d463
		ps513.OverlayValues[37] = d464
		ps513.OverlayValues[38] = d466
		ps513.OverlayValues[39] = d467
		ps513.OverlayValues[40] = d468
		ps514 := PhiState{General: true}
		ps514.OverlayValues = make([]JITValueDesc, 41)
		ps514.OverlayValues[0] = d0
		ps514.OverlayValues[1] = d1
		ps514.OverlayValues[2] = d2
		ps514.OverlayValues[3] = d3
		ps514.OverlayValues[4] = d4
		ps514.OverlayValues[5] = d5
		ps514.OverlayValues[6] = d24
		ps514.OverlayValues[7] = d25
		ps514.OverlayValues[8] = d26
		ps514.OverlayValues[9] = d51
		ps514.OverlayValues[10] = d52
		ps514.OverlayValues[11] = d81
		ps514.OverlayValues[12] = d82
		ps514.OverlayValues[13] = d83
		ps514.OverlayValues[14] = d118
		ps514.OverlayValues[15] = d119
		ps514.OverlayValues[16] = d120
		ps514.OverlayValues[17] = d161
		ps514.OverlayValues[18] = d162
		ps514.OverlayValues[19] = d207
		ps514.OverlayValues[20] = d208
		ps514.OverlayValues[21] = d257
		ps514.OverlayValues[22] = d258
		ps514.OverlayValues[23] = d311
		ps514.OverlayValues[24] = d312
		ps514.OverlayValues[25] = d314
		ps514.OverlayValues[26] = d315
		ps514.OverlayValues[27] = d316
		ps514.OverlayValues[28] = d379
		ps514.OverlayValues[29] = d380
		ps514.OverlayValues[30] = d381
		ps514.OverlayValues[31] = d383
		ps514.OverlayValues[32] = d384
		ps514.OverlayValues[33] = d385
		ps514.OverlayValues[34] = d460
		ps514.OverlayValues[35] = d462
		ps514.OverlayValues[36] = d463
		ps514.OverlayValues[37] = d464
		ps514.OverlayValues[38] = d466
		ps514.OverlayValues[39] = d467
		ps514.OverlayValues[40] = d468
		snap515 := d0
		snap516 := d1
		snap517 := d2
		snap518 := d3
		snap519 := d4
		snap520 := d5
		snap521 := d24
		snap522 := d25
		snap523 := d26
		snap524 := d51
		snap525 := d52
		snap526 := d81
		snap527 := d82
		snap528 := d83
		snap529 := d118
		snap530 := d119
		snap531 := d120
		snap532 := d161
		snap533 := d162
		snap534 := d207
		snap535 := d208
		snap536 := d257
		snap537 := d258
		snap538 := d311
		snap539 := d312
		snap540 := d314
		snap541 := d315
		snap542 := d316
		snap543 := d379
		snap544 := d380
		snap545 := d381
		snap546 := d383
		snap547 := d384
		snap548 := d385
		snap549 := d460
		snap550 := d462
		snap551 := d463
		snap552 := d464
		snap553 := d466
		snap554 := d467
		snap555 := d468
		alloc556 := ctx.SnapshotAllocState()
		if !bbs[20].Rendered {
			bbs[20].RenderPS(ps514)
		}
		ctx.RestoreAllocState(alloc556)
		d0 = snap515
		d1 = snap516
		d2 = snap517
		d3 = snap518
		d4 = snap519
		d5 = snap520
		d24 = snap521
		d25 = snap522
		d26 = snap523
		d51 = snap524
		d52 = snap525
		d81 = snap526
		d82 = snap527
		d83 = snap528
		d118 = snap529
		d119 = snap530
		d120 = snap531
		d161 = snap532
		d162 = snap533
		d207 = snap534
		d208 = snap535
		d257 = snap536
		d258 = snap537
		d311 = snap538
		d312 = snap539
		d314 = snap540
		d315 = snap541
		d316 = snap542
		d379 = snap543
		d380 = snap544
		d381 = snap545
		d383 = snap546
		d384 = snap547
		d385 = snap548
		d460 = snap549
		d462 = snap550
		d463 = snap551
		d464 = snap552
		d466 = snap553
		d467 = snap554
		d468 = snap555
		if !bbs[15].Rendered {
			return bbs[15].RenderPS(ps513)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
			d467 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
			d468 = ps.OverlayValues[40]
		}
		ctx.ReclaimUntrackedRegs()
		var d557 JITValueDesc
		if args[0].Loc == LocImm {
			d557 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d557 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d557)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d557 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d557)
		} else {
			d557 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d557.Type = tagInt
			ctx.BindReg(d557.Reg, &d557)
		}
		var d558 JITValueDesc
		if args[1].Loc == LocImm {
			d558 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d558 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d558)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d558 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d558)
		} else {
			d558 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[1]}, 1)
			d558.Type = tagInt
			ctx.BindReg(d558.Reg, &d558)
		}
		ctx.EnsureDesc(&d557)
		resultTarget559 := false
		_ = resultTarget559
		ctx.EnsureDesc(&d558)
		ctx.EnsureDescsTogether(&d557, &d558)
		var d560 JITValueDesc
		if d557.Loc == LocImm && d558.Loc == LocImm {
			d560 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d557.Imm.Int() < d558.Imm.Int())}
		} else if d558.Loc == LocImm {
			r20 := ctx.AllocReg()
			if d558.Imm.Int() >= -2147483648 && d558.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d557.Reg, int32(d558.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(RegR11, uint64(d558.Imm.Int()))
				ctx.EmitCmpInt64(d557.Reg, RegR11)
			}
			d560 = ctx.DeferBooleanFlags(r20, CondSignedLess)
			ctx.BindReg(r20, &d560)
		} else if d557.Loc == LocImm {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegImm64(RegR11, uint64(d557.Imm.Int()))
			ctx.EmitCmpInt64(RegR11, d558.Reg)
			d560 = ctx.DeferBooleanFlags(r21, CondSignedLess)
			ctx.BindReg(r21, &d560)
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitCmpInt64(d557.Reg, d558.Reg)
			d560 = ctx.DeferBooleanFlags(r22, CondSignedLess)
			ctx.BindReg(r22, &d560)
		}
		ctx.FreeDesc(&d557)
		ctx.FreeDesc(&d558)
		ctx.SyncDesc(&d560)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d560) {
			return d560
		}
		ctx.EnsureDesc(&d560)
		ctx.EmitMovToReg(result.Reg, d560)
		result.Type = d560.Type
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
			d467 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
			d468 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
			d557 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
			d558 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
			d560 = ps.OverlayValues[43]
		}
		ctx.ReclaimUntrackedRegs()
		var d561 JITValueDesc
		if args[0].Loc == LocImm {
			d561 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d561 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d561)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d561 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d561)
		} else {
			d561 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d561.Type = tagInt
			ctx.BindReg(d561.Reg, &d561)
		}
		ctx.EnsureDesc(&d561)
		ctx.EnsureDesc(&d561)
		var d562 JITValueDesc
		if d561.Loc == LocImm {
			d562 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d561.Imm.Int()))}
		} else {
			var r23 Reg
			r23 = d561.Reg
			d561.Loc = LocNone
			ctx.EmitCvtInt64ToFloat64(RegX0, r23)
			d562 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r23}
			ctx.BindReg(r23, &d562)
		}
		ctx.FreeDesc(&d561)
		d563 = ctx.EmitFloatDesc(args[1])
		ctx.EnsureDesc(&d562)
		resultTarget564 := false
		_ = resultTarget564
		ctx.EnsureDesc(&d563)
		ctx.EnsureDescsTogether(&d562, &d563)
		var d565 JITValueDesc
		if d562.Loc == LocImm && d563.Loc == LocImm {
			d565 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d562.Imm.Float() < d563.Imm.Float())}
		} else if d563.Loc == LocImm {
			var r24 Reg
			if result.Loc == LocReg && result.Reg != d562.Reg {
				r24 = result.Reg
				resultTarget564 = true
			} else {
				r24 = ctx.AllocRegExcept(d562.Reg)
			}
			_, yBits := d563.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r24, d562.Reg, RegR11, CondSignedLess)
			d565 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
			ctx.BindReg(r24, &d565)
		} else if d562.Loc == LocImm {
			var r25 Reg
			if result.Loc == LocReg && result.Reg != d563.Reg {
				r25 = result.Reg
				resultTarget564 = true
			} else {
				r25 = ctx.AllocRegExcept(d563.Reg)
			}
			_, xBits := d562.Imm.RawWords()
			ctx.EmitMovRegImm64(RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r25, RegR11, d563.Reg, CondSignedLess)
			d565 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
			ctx.BindReg(r25, &d565)
		} else {
			var r26 Reg
			if result.Loc == LocReg && result.Reg != d562.Reg && result.Reg != d563.Reg {
				r26 = result.Reg
				resultTarget564 = true
			} else {
				r26 = ctx.AllocRegExcept(d562.Reg, d563.Reg)
			}
			ctx.EmitCmpFloat64Setcc(r26, d562.Reg, d563.Reg, CondSignedLess)
			d565 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
			ctx.BindReg(r26, &d565)
		}
		ctx.FreeDesc(&d562)
		ctx.FreeDesc(&d563)
		ctx.SyncDesc(&d565)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d565) {
			return d565
		}
		ctx.EnsureDesc(&d565)
		ctx.EmitMovToReg(result.Reg, d565)
		result.Type = d565.Type
		ctx.EmitJmp(lbl0)
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
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
			d467 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
			d468 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
			d557 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
			d558 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
			d560 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
			d561 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
			d562 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
			d563 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
			d565 = ps.OverlayValues[47]
		}
		ctx.ReclaimUntrackedRegs()
		var d566 JITValueDesc
		if args[0].Loc == LocImm {
			d566 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[0].Imm.Int())}
		} else if args[0].Type == tagInt && args[0].Loc == LocRegPair {
			ctx.FreeReg(args[0].Reg)
			d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg2}
			ctx.BindReg(args[0].Reg2, &d566)
		} else if args[0].Type == tagInt && args[0].Loc == LocReg {
			d566 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[0].Reg}
			ctx.BindReg(args[0].Reg, &d566)
		} else {
			d566 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[0]}, 1)
			d566.Type = tagInt
			ctx.BindReg(d566.Reg, &d566)
		}
		var d567 JITValueDesc
		if args[1].Loc == LocImm {
			d567 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(args[1].Imm.Int())}
		} else if args[1].Type == tagInt && args[1].Loc == LocRegPair {
			ctx.FreeReg(args[1].Reg)
			d567 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg2}
			ctx.BindReg(args[1].Reg2, &d567)
		} else if args[1].Type == tagInt && args[1].Loc == LocReg {
			d567 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: args[1].Reg}
			ctx.BindReg(args[1].Reg, &d567)
		} else {
			d567 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{args[1]}, 1)
			d567.Type = tagInt
			ctx.BindReg(d567.Reg, &d567)
		}
		ctx.EnsureDesc(&d566)
		resultTarget568 := false
		_ = resultTarget568
		ctx.EnsureDesc(&d567)
		ctx.EnsureDescsTogether(&d566, &d567)
		var d569 JITValueDesc
		if d566.Loc == LocImm && d567.Loc == LocImm {
			d569 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d566.Imm.Int() < d567.Imm.Int())}
		} else if d567.Loc == LocImm {
			r27 := ctx.AllocReg()
			if d567.Imm.Int() >= -2147483648 && d567.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d566.Reg, int32(d567.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(RegR11, uint64(d567.Imm.Int()))
				ctx.EmitCmpInt64(d566.Reg, RegR11)
			}
			d569 = ctx.DeferBooleanFlags(r27, CondSignedLess)
			ctx.BindReg(r27, &d569)
		} else if d566.Loc == LocImm {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegImm64(RegR11, uint64(d566.Imm.Int()))
			ctx.EmitCmpInt64(RegR11, d567.Reg)
			d569 = ctx.DeferBooleanFlags(r28, CondSignedLess)
			ctx.BindReg(r28, &d569)
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitCmpInt64(d566.Reg, d567.Reg)
			d569 = ctx.DeferBooleanFlags(r29, CondSignedLess)
			ctx.BindReg(r29, &d569)
		}
		ctx.FreeDesc(&d566)
		ctx.FreeDesc(&d567)
		ctx.SyncDesc(&d569)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d569) {
			return d569
		}
		ctx.EnsureDesc(&d569)
		ctx.EmitMovToReg(result.Reg, d569)
		result.Type = d569.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[20].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[20].VisitCount >= 0 {
				ps.General = true
				return bbs[20].RenderPS(ps)
			}
		}
		bbs[20].VisitCount++
		if ps.General {
			if bbs[20].Rendered {
				ctx.EmitJmp(lbl21)
				return result
			}
			bbs[20].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[20].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_20 = bbs[20].Address
			ctx.MarkLabel(lbl21)
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
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
			d467 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
			d468 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
			d557 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
			d558 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
			d560 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
			d561 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
			d562 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
			d563 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
			d565 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
			d566 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
			d567 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
			d569 = ps.OverlayValues[50]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		var d570 JITValueDesc
		if d1.Loc == LocImm {
			d570 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(0x5))}
		} else {
			r30 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpRegImm32(d1.Reg, 5)
			d570 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondEqual}
			ctx.BindReg(r30, &d570)
		}
		d571 = d570
		ctx.EnsureDesc(&d571)
		if d571.Loc != LocImm && d571.Loc != LocFlags {
			panic("jit: fused If condition is neither LocImm nor LocFlags")
		}
		if d571.Loc == LocImm {
			if d571.Imm.Bool() {
				if ps.General {
				}
				ps572 := PhiState{General: ps.General}
				ps572.OverlayValues = make([]JITValueDesc, 53)
				ps572.OverlayValues[0] = d0
				ps572.OverlayValues[1] = d1
				ps572.OverlayValues[2] = d2
				ps572.OverlayValues[3] = d3
				ps572.OverlayValues[4] = d4
				ps572.OverlayValues[5] = d5
				ps572.OverlayValues[6] = d24
				ps572.OverlayValues[7] = d25
				ps572.OverlayValues[8] = d26
				ps572.OverlayValues[9] = d51
				ps572.OverlayValues[10] = d52
				ps572.OverlayValues[11] = d81
				ps572.OverlayValues[12] = d82
				ps572.OverlayValues[13] = d83
				ps572.OverlayValues[14] = d118
				ps572.OverlayValues[15] = d119
				ps572.OverlayValues[16] = d120
				ps572.OverlayValues[17] = d161
				ps572.OverlayValues[18] = d162
				ps572.OverlayValues[19] = d207
				ps572.OverlayValues[20] = d208
				ps572.OverlayValues[21] = d257
				ps572.OverlayValues[22] = d258
				ps572.OverlayValues[23] = d311
				ps572.OverlayValues[24] = d312
				ps572.OverlayValues[25] = d314
				ps572.OverlayValues[26] = d315
				ps572.OverlayValues[27] = d316
				ps572.OverlayValues[28] = d379
				ps572.OverlayValues[29] = d380
				ps572.OverlayValues[30] = d381
				ps572.OverlayValues[31] = d383
				ps572.OverlayValues[32] = d384
				ps572.OverlayValues[33] = d385
				ps572.OverlayValues[34] = d460
				ps572.OverlayValues[35] = d462
				ps572.OverlayValues[36] = d463
				ps572.OverlayValues[37] = d464
				ps572.OverlayValues[38] = d466
				ps572.OverlayValues[39] = d467
				ps572.OverlayValues[40] = d468
				ps572.OverlayValues[41] = d557
				ps572.OverlayValues[42] = d558
				ps572.OverlayValues[43] = d560
				ps572.OverlayValues[44] = d561
				ps572.OverlayValues[45] = d562
				ps572.OverlayValues[46] = d563
				ps572.OverlayValues[47] = d565
				ps572.OverlayValues[48] = d566
				ps572.OverlayValues[49] = d567
				ps572.OverlayValues[50] = d569
				ps572.OverlayValues[51] = d570
				ps572.OverlayValues[52] = d571
				return bbs[19].RenderPS(ps572)
			}
			if ps.General {
			}
			ps573 := PhiState{General: ps.General}
			ps573.OverlayValues = make([]JITValueDesc, 53)
			ps573.OverlayValues[0] = d0
			ps573.OverlayValues[1] = d1
			ps573.OverlayValues[2] = d2
			ps573.OverlayValues[3] = d3
			ps573.OverlayValues[4] = d4
			ps573.OverlayValues[5] = d5
			ps573.OverlayValues[6] = d24
			ps573.OverlayValues[7] = d25
			ps573.OverlayValues[8] = d26
			ps573.OverlayValues[9] = d51
			ps573.OverlayValues[10] = d52
			ps573.OverlayValues[11] = d81
			ps573.OverlayValues[12] = d82
			ps573.OverlayValues[13] = d83
			ps573.OverlayValues[14] = d118
			ps573.OverlayValues[15] = d119
			ps573.OverlayValues[16] = d120
			ps573.OverlayValues[17] = d161
			ps573.OverlayValues[18] = d162
			ps573.OverlayValues[19] = d207
			ps573.OverlayValues[20] = d208
			ps573.OverlayValues[21] = d257
			ps573.OverlayValues[22] = d258
			ps573.OverlayValues[23] = d311
			ps573.OverlayValues[24] = d312
			ps573.OverlayValues[25] = d314
			ps573.OverlayValues[26] = d315
			ps573.OverlayValues[27] = d316
			ps573.OverlayValues[28] = d379
			ps573.OverlayValues[29] = d380
			ps573.OverlayValues[30] = d381
			ps573.OverlayValues[31] = d383
			ps573.OverlayValues[32] = d384
			ps573.OverlayValues[33] = d385
			ps573.OverlayValues[34] = d460
			ps573.OverlayValues[35] = d462
			ps573.OverlayValues[36] = d463
			ps573.OverlayValues[37] = d464
			ps573.OverlayValues[38] = d466
			ps573.OverlayValues[39] = d467
			ps573.OverlayValues[40] = d468
			ps573.OverlayValues[41] = d557
			ps573.OverlayValues[42] = d558
			ps573.OverlayValues[43] = d560
			ps573.OverlayValues[44] = d561
			ps573.OverlayValues[45] = d562
			ps573.OverlayValues[46] = d563
			ps573.OverlayValues[47] = d565
			ps573.OverlayValues[48] = d566
			ps573.OverlayValues[49] = d567
			ps573.OverlayValues[50] = d569
			ps573.OverlayValues[51] = d570
			ps573.OverlayValues[52] = d571
			return bbs[21].RenderPS(ps573)
		}
		if !ps.General {
			ps.General = true
			return bbs[20].RenderPS(ps)
		}
		ctx.EmitJump(d571.Condition, lbl20)
		if bbs[21].Rendered {
			ctx.EmitJmp(lbl22)
		}
		ctx.FreeDesc(&d570)
		snap574 := d0
		snap575 := d1
		snap576 := d2
		snap577 := d3
		snap578 := d4
		snap579 := d5
		snap580 := d24
		snap581 := d25
		snap582 := d26
		snap583 := d51
		snap584 := d52
		snap585 := d81
		snap586 := d82
		snap587 := d83
		snap588 := d118
		snap589 := d119
		snap590 := d120
		snap591 := d161
		snap592 := d162
		snap593 := d207
		snap594 := d208
		snap595 := d257
		snap596 := d258
		snap597 := d311
		snap598 := d312
		snap599 := d314
		snap600 := d315
		snap601 := d316
		snap602 := d379
		snap603 := d380
		snap604 := d381
		snap605 := d383
		snap606 := d384
		snap607 := d385
		snap608 := d460
		snap609 := d462
		snap610 := d463
		snap611 := d464
		snap612 := d466
		snap613 := d467
		snap614 := d468
		snap615 := d557
		snap616 := d558
		snap617 := d560
		snap618 := d561
		snap619 := d562
		snap620 := d563
		snap621 := d565
		snap622 := d566
		snap623 := d567
		snap624 := d569
		snap625 := d570
		snap626 := d571
		alloc627 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc627)
		d0 = snap574
		d1 = snap575
		d2 = snap576
		d3 = snap577
		d4 = snap578
		d5 = snap579
		d24 = snap580
		d25 = snap581
		d26 = snap582
		d51 = snap583
		d52 = snap584
		d81 = snap585
		d82 = snap586
		d83 = snap587
		d118 = snap588
		d119 = snap589
		d120 = snap590
		d161 = snap591
		d162 = snap592
		d207 = snap593
		d208 = snap594
		d257 = snap595
		d258 = snap596
		d311 = snap597
		d312 = snap598
		d314 = snap599
		d315 = snap600
		d316 = snap601
		d379 = snap602
		d380 = snap603
		d381 = snap604
		d383 = snap605
		d384 = snap606
		d385 = snap607
		d460 = snap608
		d462 = snap609
		d463 = snap610
		d464 = snap611
		d466 = snap612
		d467 = snap613
		d468 = snap614
		d557 = snap615
		d558 = snap616
		d560 = snap617
		d561 = snap618
		d562 = snap619
		d563 = snap620
		d565 = snap621
		d566 = snap622
		d567 = snap623
		d569 = snap624
		d570 = snap625
		d571 = snap626
		ctx.RestoreAllocState(alloc627)
		d0 = snap574
		d1 = snap575
		d2 = snap576
		d3 = snap577
		d4 = snap578
		d5 = snap579
		d24 = snap580
		d25 = snap581
		d26 = snap582
		d51 = snap583
		d52 = snap584
		d81 = snap585
		d82 = snap586
		d83 = snap587
		d118 = snap588
		d119 = snap589
		d120 = snap590
		d161 = snap591
		d162 = snap592
		d207 = snap593
		d208 = snap594
		d257 = snap595
		d258 = snap596
		d311 = snap597
		d312 = snap598
		d314 = snap599
		d315 = snap600
		d316 = snap601
		d379 = snap602
		d380 = snap603
		d381 = snap604
		d383 = snap605
		d384 = snap606
		d385 = snap607
		d460 = snap608
		d462 = snap609
		d463 = snap610
		d464 = snap611
		d466 = snap612
		d467 = snap613
		d468 = snap614
		d557 = snap615
		d558 = snap616
		d560 = snap617
		d561 = snap618
		d562 = snap619
		d563 = snap620
		d565 = snap621
		d566 = snap622
		d567 = snap623
		d569 = snap624
		d570 = snap625
		d571 = snap626
		ps628 := PhiState{General: true}
		ps628.OverlayValues = make([]JITValueDesc, 53)
		ps628.OverlayValues[0] = d0
		ps628.OverlayValues[1] = d1
		ps628.OverlayValues[2] = d2
		ps628.OverlayValues[3] = d3
		ps628.OverlayValues[4] = d4
		ps628.OverlayValues[5] = d5
		ps628.OverlayValues[6] = d24
		ps628.OverlayValues[7] = d25
		ps628.OverlayValues[8] = d26
		ps628.OverlayValues[9] = d51
		ps628.OverlayValues[10] = d52
		ps628.OverlayValues[11] = d81
		ps628.OverlayValues[12] = d82
		ps628.OverlayValues[13] = d83
		ps628.OverlayValues[14] = d118
		ps628.OverlayValues[15] = d119
		ps628.OverlayValues[16] = d120
		ps628.OverlayValues[17] = d161
		ps628.OverlayValues[18] = d162
		ps628.OverlayValues[19] = d207
		ps628.OverlayValues[20] = d208
		ps628.OverlayValues[21] = d257
		ps628.OverlayValues[22] = d258
		ps628.OverlayValues[23] = d311
		ps628.OverlayValues[24] = d312
		ps628.OverlayValues[25] = d314
		ps628.OverlayValues[26] = d315
		ps628.OverlayValues[27] = d316
		ps628.OverlayValues[28] = d379
		ps628.OverlayValues[29] = d380
		ps628.OverlayValues[30] = d381
		ps628.OverlayValues[31] = d383
		ps628.OverlayValues[32] = d384
		ps628.OverlayValues[33] = d385
		ps628.OverlayValues[34] = d460
		ps628.OverlayValues[35] = d462
		ps628.OverlayValues[36] = d463
		ps628.OverlayValues[37] = d464
		ps628.OverlayValues[38] = d466
		ps628.OverlayValues[39] = d467
		ps628.OverlayValues[40] = d468
		ps628.OverlayValues[41] = d557
		ps628.OverlayValues[42] = d558
		ps628.OverlayValues[43] = d560
		ps628.OverlayValues[44] = d561
		ps628.OverlayValues[45] = d562
		ps628.OverlayValues[46] = d563
		ps628.OverlayValues[47] = d565
		ps628.OverlayValues[48] = d566
		ps628.OverlayValues[49] = d567
		ps628.OverlayValues[50] = d569
		ps628.OverlayValues[51] = d570
		ps628.OverlayValues[52] = d571
		ps629 := PhiState{General: true}
		ps629.OverlayValues = make([]JITValueDesc, 53)
		ps629.OverlayValues[0] = d0
		ps629.OverlayValues[1] = d1
		ps629.OverlayValues[2] = d2
		ps629.OverlayValues[3] = d3
		ps629.OverlayValues[4] = d4
		ps629.OverlayValues[5] = d5
		ps629.OverlayValues[6] = d24
		ps629.OverlayValues[7] = d25
		ps629.OverlayValues[8] = d26
		ps629.OverlayValues[9] = d51
		ps629.OverlayValues[10] = d52
		ps629.OverlayValues[11] = d81
		ps629.OverlayValues[12] = d82
		ps629.OverlayValues[13] = d83
		ps629.OverlayValues[14] = d118
		ps629.OverlayValues[15] = d119
		ps629.OverlayValues[16] = d120
		ps629.OverlayValues[17] = d161
		ps629.OverlayValues[18] = d162
		ps629.OverlayValues[19] = d207
		ps629.OverlayValues[20] = d208
		ps629.OverlayValues[21] = d257
		ps629.OverlayValues[22] = d258
		ps629.OverlayValues[23] = d311
		ps629.OverlayValues[24] = d312
		ps629.OverlayValues[25] = d314
		ps629.OverlayValues[26] = d315
		ps629.OverlayValues[27] = d316
		ps629.OverlayValues[28] = d379
		ps629.OverlayValues[29] = d380
		ps629.OverlayValues[30] = d381
		ps629.OverlayValues[31] = d383
		ps629.OverlayValues[32] = d384
		ps629.OverlayValues[33] = d385
		ps629.OverlayValues[34] = d460
		ps629.OverlayValues[35] = d462
		ps629.OverlayValues[36] = d463
		ps629.OverlayValues[37] = d464
		ps629.OverlayValues[38] = d466
		ps629.OverlayValues[39] = d467
		ps629.OverlayValues[40] = d468
		ps629.OverlayValues[41] = d557
		ps629.OverlayValues[42] = d558
		ps629.OverlayValues[43] = d560
		ps629.OverlayValues[44] = d561
		ps629.OverlayValues[45] = d562
		ps629.OverlayValues[46] = d563
		ps629.OverlayValues[47] = d565
		ps629.OverlayValues[48] = d566
		ps629.OverlayValues[49] = d567
		ps629.OverlayValues[50] = d569
		ps629.OverlayValues[51] = d570
		ps629.OverlayValues[52] = d571
		snap630 := d0
		snap631 := d1
		snap632 := d2
		snap633 := d3
		snap634 := d4
		snap635 := d5
		snap636 := d24
		snap637 := d25
		snap638 := d26
		snap639 := d51
		snap640 := d52
		snap641 := d81
		snap642 := d82
		snap643 := d83
		snap644 := d118
		snap645 := d119
		snap646 := d120
		snap647 := d161
		snap648 := d162
		snap649 := d207
		snap650 := d208
		snap651 := d257
		snap652 := d258
		snap653 := d311
		snap654 := d312
		snap655 := d314
		snap656 := d315
		snap657 := d316
		snap658 := d379
		snap659 := d380
		snap660 := d381
		snap661 := d383
		snap662 := d384
		snap663 := d385
		snap664 := d460
		snap665 := d462
		snap666 := d463
		snap667 := d464
		snap668 := d466
		snap669 := d467
		snap670 := d468
		snap671 := d557
		snap672 := d558
		snap673 := d560
		snap674 := d561
		snap675 := d562
		snap676 := d563
		snap677 := d565
		snap678 := d566
		snap679 := d567
		snap680 := d569
		snap681 := d570
		snap682 := d571
		alloc683 := ctx.SnapshotAllocState()
		if !bbs[21].Rendered {
			bbs[21].RenderPS(ps629)
		}
		ctx.RestoreAllocState(alloc683)
		d0 = snap630
		d1 = snap631
		d2 = snap632
		d3 = snap633
		d4 = snap634
		d5 = snap635
		d24 = snap636
		d25 = snap637
		d26 = snap638
		d51 = snap639
		d52 = snap640
		d81 = snap641
		d82 = snap642
		d83 = snap643
		d118 = snap644
		d119 = snap645
		d120 = snap646
		d161 = snap647
		d162 = snap648
		d207 = snap649
		d208 = snap650
		d257 = snap651
		d258 = snap652
		d311 = snap653
		d312 = snap654
		d314 = snap655
		d315 = snap656
		d316 = snap657
		d379 = snap658
		d380 = snap659
		d381 = snap660
		d383 = snap661
		d384 = snap662
		d385 = snap663
		d460 = snap664
		d462 = snap665
		d463 = snap666
		d464 = snap667
		d466 = snap668
		d467 = snap669
		d468 = snap670
		d557 = snap671
		d558 = snap672
		d560 = snap673
		d561 = snap674
		d562 = snap675
		d563 = snap676
		d565 = snap677
		d566 = snap678
		d567 = snap679
		d569 = snap680
		d570 = snap681
		d571 = snap682
		if !bbs[19].Rendered {
			return bbs[19].RenderPS(ps628)
		}
		return result
		return result
	}
	bbs[21].RenderPS = func(ps PhiState) JITValueDesc {
		if !ps.General {
			if bbs[21].VisitCount >= 0 {
				ps.General = true
				return bbs[21].RenderPS(ps)
			}
		}
		bbs[21].VisitCount++
		if ps.General {
			if bbs[21].Rendered {
				ctx.EmitJmp(lbl22)
				return result
			}
			bbs[21].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[21].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_21 = bbs[21].Address
			ctx.MarkLabel(lbl22)
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
			d24 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
			d25 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
			d26 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
			d51 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
			d52 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
			d81 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
			d82 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
			d83 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
			d118 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
			d119 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
			d120 = ps.OverlayValues[16]
		}
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
			d161 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
			d162 = ps.OverlayValues[18]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
			d207 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
			d208 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
			d257 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
			d258 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
			d311 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
			d312 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
			d314 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
			d315 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
			d316 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
			d379 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
			d380 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
			d381 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
			d383 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
			d384 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
			d385 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
			d460 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
			d462 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
			d463 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
			d464 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
			d466 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
			d467 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
			d468 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
			d557 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
			d558 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
			d560 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
			d561 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
			d562 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
			d563 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
			d565 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
			d566 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
			d567 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
			d569 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
			d570 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
			d571 = ps.OverlayValues[52]
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
		d684 = ctx.EmitGoCallScalar(GoFuncAddr(lessNonNumeric), []JITValueDesc{args[0], args[1], d1, d3}, 1)
		d684.NoHeapPointer = true
		ctx.EmitAndRegImm32(d684.Reg, 1)
		d684.Type = tagBool
		ctx.BindReg(d684.Reg, &d684)
		ctx.FreeDesc(&args[0])
		ctx.FreeDesc(&args[1])
		ctx.SyncDesc(&d684)
		if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d684) {
			return d684
		}
		ctx.EnsureDesc(&d684)
		ctx.EmitMovToReg(result.Reg, d684)
		result.Type = d684.Type
		ctx.EmitJmp(lbl0)
		return result
	}
	ps685 := PhiState{General: false}
	returned := bbs[0].RenderPS(ps685)
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
