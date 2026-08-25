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
package storage

import "io"
import "fmt"
import "math"
import "bufio"
import "unsafe"
import "encoding/json"
import "encoding/base64"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// main type for storage: can store any value, is inefficient but does type analysis how to optimize
type StorageSCMER struct {
	values      []scm.Scmer
	minIntScale int8 // power-of-ten exponent of finest granularity
	//   0 = pure ints, -2 = 2 decimal places, 2 = multiples of 100
	//   math.MinInt8 = not representable as scaled int
	hasString    bool
	hasBSON      bool
	longStrings  int
	null         uint  // amount of NULL values (sparse map!)
	numSeq       uint  // sequence statistics
	last1, last2 int64 // sequence statistics

	// enum detection: collect up to enumMaxSymbols distinct values
	enumVals  [enumMaxSymbols]scm.Scmer
	enumFreqs [enumMaxSymbols]uint64
	enumK     uint8 // number of distinct values seen so far (0xFF = abandoned)
}

func (s *StorageSCMER) ComputeSize() uint {
	// ! size of Scmer values is not considered
	var sz uint = 80 + 24
	for _, v := range s.values {
		sz += scm.ComputeSize(v)
	}
	return sz
}

func (s *StorageSCMER) String() string {
	return "SCMER"
}

// StorageSCMER binary layout (magic byte 42 consumed by shard loader):
//
//	[version uint8]
//	[count uint64]
//	[values: count × tagged value]
//	  tag 0: [JSON length uint32][legacy Scmer JSON]
//	  tag 1: [BSON type uint8][payload length uint32][raw BSON payload]
//
// Version history:
//
//	v1: raw BSON payloads; other Scmer tags retain their established JSON form.
//
// Magic byte 1 is handled by StorageSCMERLegacy below and remains readable
// forever. The raw BSON payload is the sole persisted JSON representation;
// there is no Base64 or ordinary-JSON copy beside it.
const storageSCMERVersion = 1

const (
	storageSCMERJSON = iota
	storageSCMERBSON
)

func (s *StorageSCMER) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
				ctx.BindReg(idx.Reg2, &idxInt)
			} else {
				idxInt = idx
			}
			if idxInt.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EnsureDesc(&idxInt)
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSCMER)(nil).values)
				r0 := ctx.AllocReg()
				r1 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r0, fieldAddr)
				ctx.EmitMovRegMem64(r1, fieldAddr+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
				ctx.BindReg(r0, &d0)
				ctx.BindReg(r1, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageSCMER)(nil).values))
				r2 := ctx.AllocReg()
				r3 := ctx.AllocReg()
				ctx.EmitMovRegMem(r2, thisptr.Reg, off)
				ctx.EmitMovRegMem(r3, thisptr.Reg, off+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d0)
				ctx.BindReg(r3, &d0)
			}
			ctx.EnsureDesc(&idxInt)
			r4 := ctx.AllocReg()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d0)
			if idxInt.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r4, uint64(idxInt.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r4, idxInt.Reg)
				ctx.EmitShlRegImm8(r4, 4)
			}
			if d0.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(r4, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r4, d0.Reg)
			}
			r5 := ctx.AllocRegExcept(r4)
			r6 := ctx.AllocRegExcept(r4, r5)
			ctx.EmitMovRegMem(r5, r4, 0)
			ctx.EmitMovRegMem(r6, r4, 8)
			ctx.FreeReg(r4)
			d1 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r5, Reg2: r6}
			ctx.BindReg(r5, &d1)
			ctx.BindReg(r6, &d1)
			ctx.FreeDesc(&idxInt)
			if d1.Loc == scm.LocImm {
				if result.Loc == scm.LocAny { return d1 }
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d1, &result)
				result.Type = d1.Type
			} else {
				switch d1.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(result, d1)
					result.Type = scm.TagBool
				case scm.TagInt:
					ctx.EmitMakeInt(result, d1)
					result.Type = scm.TagInt
				case scm.TagFloat:
					ctx.EmitMakeFloat(result, d1)
					result.Type = scm.TagFloat
				case scm.TagNil:
					ctx.EmitMakeNil(result)
					result.Type = scm.TagNil
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
			return result
}

func (s *StorageSCMER) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(42))
	binary.Write(f, binary.LittleEndian, uint8(storageSCMERVersion))
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for _, value := range s.values {
		if value.IsBSON() {
			typ, payload := value.BSONRaw()
			binary.Write(f, binary.LittleEndian, uint8(storageSCMERBSON))
			binary.Write(f, binary.LittleEndian, typ)
			binary.Write(f, binary.LittleEndian, uint32(len(payload)))
			if _, err := f.Write(payload); err != nil {
				panic(err)
			}
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		if uint64(len(encoded)) > math.MaxUint32 {
			panic("StorageSCMER value exceeds 32-bit size limit")
		}
		binary.Write(f, binary.LittleEndian, uint8(storageSCMERJSON))
		binary.Write(f, binary.LittleEndian, uint32(len(encoded)))
		if _, err := f.Write(encoded); err != nil {
			panic(err)
		}
	}
}

func (s *StorageSCMER) Deserialize(f io.Reader) uint {
	var version uint8
	if err := binary.Read(f, binary.LittleEndian, &version); err != nil {
		panic(err)
	}
	if version != storageSCMERVersion {
		panic(fmt.Sprintf("StorageSCMER: unknown version %d", version))
	}
	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		panic(err)
	}
	s.values = make([]scm.Scmer, count)
	for i := range s.values {
		var kind uint8
		if err := binary.Read(f, binary.LittleEndian, &kind); err != nil {
			panic(err)
		}
		switch kind {
		case storageSCMERJSON:
			var length uint32
			if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
				panic(err)
			}
			encoded := make([]byte, length)
			if _, err := io.ReadFull(f, encoded); err != nil {
				panic(err)
			}
			var value any
			if err := json.Unmarshal(encoded, &value); err != nil {
				panic(err)
			}
			s.values[i] = scm.TransformFromJSON(value)
		case storageSCMERBSON:
			var typ uint8
			var length uint32
			if err := binary.Read(f, binary.LittleEndian, &typ); err != nil {
				panic(err)
			}
			if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
				panic(err)
			}
			payload := make([]byte, length)
			if _, err := io.ReadFull(f, payload); err != nil {
				panic(err)
			}
			s.values[i] = scm.NewBSONRaw(typ, payload)
		default:
			panic(fmt.Sprintf("StorageSCMER: unknown value tag %d", kind))
		}
	}
	return uint(count)
}

// StorageSCMERLegacy is the read-only decoder for permanent magic byte 1. Its
// embedded StorageSCMER supplies the regular column operations. If a legacy
// shard is rebuilt, Serialize writes the current magic byte 42 format.
type StorageSCMERLegacy struct {
	StorageSCMER
}

func (s *StorageSCMERLegacy) Deserialize(f io.Reader) uint {
	// No version byte: this type had no padding byte in v0.1.0.
	// Count is read directly.  Format changes require a new magic byte.
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.StorageSCMER.values = make([]scm.Scmer, l)
	scanner := bufio.NewScanner(f)
	// BSON documents may be substantially larger than Scanner's 64 KiB default.
	scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)
	for i := uint64(0); i < l; i++ {
		if scanner.Scan() {
			var v any
			json.Unmarshal(scanner.Bytes(), &v)
			if envelope, ok := v.(map[string]any); ok && len(envelope) == 3 && envelope["$memcp.scmer"] == "bson-v1" {
				typ, typeOK := envelope["type"].(float64)
				payload, payloadOK := envelope["payload"].(string)
				decoded, err := base64.RawStdEncoding.DecodeString(payload)
				if typeOK && typ >= 0 && typ <= 255 && typ == math.Trunc(typ) && payloadOK && err == nil {
					s.StorageSCMER.values[i] = scm.NewBSONRaw(byte(typ), decoded)
					continue
				}
				panic("invalid BSON envelope in StorageSCMER")
			}
			s.StorageSCMER.values[i] = scm.TransformFromJSON(v)
		}
	}
	return uint(l)
}

func (s *StorageSCMER) GetCachedReader() ColumnReader { return s }

func (s *StorageSCMER) GetValue(i uint32) scm.Scmer {
	return s.values[i]
}

func (s *StorageSCMER) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if stride == 1 {
		copy(target[:count], s.values[recid:uint32(recid)+count])
		return
	}
	values := s.values[recid : uint32(recid)+count]
	idx := 0
	for _, v := range values {
		target[idx] = v
		idx += stride
	}
}

func (s *StorageSCMER) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	values := s.values
	idx := 0
	for _, recid := range recids {
		target[idx] = values[recid]
		idx += stride
	}
}

func (s *StorageSCMER) SetValue(i uint32, v scm.Scmer) {
	s.values[i] = v
}

func (s *StorageSCMER) scan(i uint32, value scm.Scmer) {
	// enum detection: track up to enumMaxSymbols distinct values with frequencies
	if s.enumK != 0xFF {
		found := false
		for j := uint8(0); j < s.enumK; j++ {
			// Use strict comparison: NULL is only equal to NULL
			// (scm.Equal treats NULL == 0 == false per Scheme semantics,
			// but storage needs them distinguished)
			if value.IsNil() == s.enumVals[j].IsNil() && (value.IsNil() || scm.Equal(s.enumVals[j], value)) {
				s.enumFreqs[j]++
				found = true
				break
			}
		}
		if !found {
			if s.enumK < enumMaxSymbols {
				s.enumVals[s.enumK] = value
				s.enumFreqs[s.enumK] = 1
				s.enumK++
			} else {
				s.enumK = 0xFF // abandon enum tracking
			}
		}
	}

	if value.IsNil() {
		s.null++
		return
	}
	if value.IsBSON() {
		s.hasBSON = true
		s.minIntScale = math.MinInt8
		return
	}
	if value.GetTag() == scm.TagDate {
		v2 := value.Int()
		if v2-s.last1 == s.last1-s.last2 {
			s.numSeq++
		}
		s.last2 = s.last1
		s.last1 = v2
		return
	}
	if value.IsInt() {
		v2 := value.Int()
		// scale detection: only if not yet abandoned
		if s.minIntScale > math.MinInt8 {
			exp := trailingZeroPow10(v2)
			if exp < s.minIntScale {
				s.minIntScale = exp
			}
		}
		if v2-s.last1 == s.last1-s.last2 {
			s.numSeq++
		}
		s.last2 = s.last1
		s.last1 = v2
		return
	}
	if value.IsFloat() {
		f := value.Float()
		// scale detection: only if not yet abandoned
		if s.minIntScale > math.MinInt8 {
			exp := detectFloatScale(f)
			if exp < s.minIntScale {
				s.minIntScale = exp
			}
		}
		// sequence statistics for integer-valued floats
		if _, frac := math.Modf(f); frac == 0.0 {
			v := int64(f)
			if v-s.last1 == s.last1-s.last2 {
				s.numSeq++
			}
			s.last2 = s.last1
			s.last1 = v
		}
		return
	}
	// non-numeric → no integer scaling possible
	s.minIntScale = math.MinInt8
	if value.IsString() {
		s.hasString = true
		if len(value.String()) > maxInlineBlobBytes {
			s.longStrings++
		}
	}
}
func (s *StorageSCMER) prepare() {
	s.minIntScale = math.MaxInt8 // neutral, gets driven down by scan
	s.hasString = false
	s.hasBSON = false
	s.enumK = 0
}
func (s *StorageSCMER) init(i uint32) {
	// allocate
	s.values = make([]scm.Scmer, i)
}
func (s *StorageSCMER) build(i uint32, value scm.Scmer) {
	// store
	s.values[i] = value
}
func (s *StorageSCMER) finish() {
}

// soley to StorageSCMER
func (s *StorageSCMER) proposeCompression(i uint32) ColumnStorage {
	if s.hasBSON {
		return nil
	}
	// const: all values identical — store only the single value (beats everything incl. sparse)
	if s.enumK == 1 && s.longStrings <= 2 {
		c := new(StorageConst)
		c.value = s.enumVals[0]
		return c
	}
	if s.null*100 > uint(i)*13 {
		// sparse payoff against bitcompressed is at ~13%
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageSparse)
			return b
		}
		return new(StorageSparse)
	}
	// enum detection: if <=8 distinct values and not abandoned, propose StorageEnum
	// but only when the distribution is skewed enough to beat PFOR's ceil(log2(k)) bits/element
	// Skip when longStrings > 2: OverlayBlob is more appropriate for blob-sized values
	if s.enumK != 0xFF && s.enumK >= 2 && i > 0 && s.longStrings <= 2 {
		// compute Shannon entropy in bits
		n := float64(i)
		entropy := 0.0
		for j := uint8(0); j < s.enumK; j++ {
			if s.enumFreqs[j] > 0 {
				p := float64(s.enumFreqs[j]) / n
				entropy -= p * math.Log2(p)
			}
		}
		// PFOR cost: ceil(log2(k)) bits/element
		pforBits := math.Ceil(math.Log2(float64(s.enumK)))
		// only propose rANS when entropy is significantly below PFOR
		if entropy < pforBits*0.8 {
			return new(StorageEnum)
		}
	}
	if s.hasString {
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageString)
			return b
		}
		return new(StorageString)
	}
	// scalable numerics (replaces onlyInt and onlyFloat)
	if s.minIntScale > math.MinInt8 {
		if s.minIntScale == 0 || s.minIntScale == math.MaxInt8 {
			// pure integers (MaxInt8 = all values were 0 or no non-NULL numerics)
			if i > 5 && 2*(uint(i)-s.numSeq) < uint(i) {
				return new(StorageSeq)
			}
			return new(StorageInt)
		}
		// scaled integers: multiples of 10^n or decimal places
		return &StorageDecimal{scaleExp: s.minIntScale}
	}
	// arbitrary floats (minIntScale == MinInt8 && !hasString)
	if !s.hasString {
		return new(StorageFloat)
	}
	if s.null*2 > uint(i) {
		// sparse payoff against StorageSCMER is at 2.1
		return new(StorageSparse)
	}
	// don't propose another pass
	return nil
}

func (s *StorageSCMER) DistinctCount() uint { return uint(len(s.values)) }
