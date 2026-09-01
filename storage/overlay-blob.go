/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
import "sync"
import "unsafe"
import "reflect"
import "strings"
import "compress/gzip"
import "crypto/sha256"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type OverlayBlob struct {
	// every overlay has a base
	Base ColumnStorage
	// values: used during build() for dedup, and for legacy inline data
	values map[[32]byte]string
	size   uint
	schema *database       // reference to owning database
	refs   map[string]bool // hex-hashes referenced in this build()
}

// Keep ordinary text values in the columnar string storage. Small text values
// are cheap to scan in place, while externalizing them turns every
// substring scan into hundreds of thousands of small file reads and gzip
// initializations. Truly large values remain deduplicated external blobs.
const maxInlineBlobBytes = 2 * 1024

func (s *OverlayBlob) ComputeSize() uint {
	return 48 + s.Base.ComputeSize()
}

func (s *OverlayBlob) String() string {
	return fmt.Sprintf("overlay[blob]+%s", s.Base.String())
}

// overlayBlobVersion is the current binary format version for OverlayBlob.
// Increment this constant and add a new deserializeBlobV* helper whenever the
// layout after the magic byte changes.  Never delete old helpers.
const overlayBlobVersion = 0

// OverlayBlob binary layout (magic byte 31 consumed by shard loader):
//
//	[version uint8]      ← first byte read by Deserialize
//	[pad 6 bytes]        ← alignment padding
//	[size uint64]        ← number of inline blobs (always 0 in v0; legacy may have >0)
//	[base storage]       ← magic byte + full serialized base column
//
// Version history:
//
//	0 (current): layout as above; the version byte was previously the first byte
//	             of a 7-byte ASCII dummy "1234567" (byte value '1'=49).
//	             Legacy: version byte '1'=49 → treat as v0 (inline blobs still possible).
func (s *OverlayBlob) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	var d0 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*OverlayBlob)(nil).Base)
		r0 := ctx.AllocReg()
		ctx.EmitMovRegMem64(r0, fieldAddr)
		d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
		ctx.BindReg(r0, &d0)
	} else {
		off := int32(unsafe.Offsetof((*OverlayBlob)(nil).Base))
		r1 := ctx.AllocReg()
		ctx.EmitMovRegMem(r1, thisptr.Reg, off)
		d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
		ctx.BindReg(r1, &d0)
	}
	ctx.EnsureDesc(&d0)
	ctx.EnsureDesc(&idxInt)
	d1 := ctx.EmitGoCallScalar(scm.GoFuncAddr(func(receiver ColumnStorage, arg0 uint32) scm.Scmer { return receiver.GetValue(arg0) }), []scm.JITValueDesc{d0, idxInt}, 2)
	ctx.FreeDesc(&idxInt)
	ctx.EnsureDesc(&thisptr)
	ctx.EnsureDesc(&thisptr)
	if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
		panic("jit: generic call arg expects 1-word value")
	}
	ctx.EnsureDesc(&d1)
	ctx.EnsureDesc(&d1)
	ctx.EnsureDesc(&d1)
	if d1.Loc == scm.LocImm {
		tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		if d1.Imm.GetTag() == scm.TagBool {
			ctx.EmitMakeBool(tmpPair, d1)
		} else if d1.Imm.GetTag() == scm.TagInt {
			ctx.EmitMakeInt(tmpPair, d1)
		} else if d1.Imm.GetTag() == scm.TagFloat {
			ctx.EmitMakeFloat(tmpPair, d1)
		} else if d1.Imm.GetTag() == scm.TagNil {
			ctx.EmitMakeNil(tmpPair)
		} else {
			ptrWord, auxWord := d1.Imm.RawWords()
			ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
			ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
		}
		d1 = tmpPair
	} else if d1.Loc == scm.LocReg {
		tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
		switch d1.Type {
		case scm.TagBool:
			ctx.EmitMakeBool(tmpPair, d1)
		case scm.TagInt:
			ctx.EmitMakeInt(tmpPair, d1)
		case scm.TagFloat:
			ctx.EmitMakeFloat(tmpPair, d1)
		default:
			panic("jit: generic call arg scalar type unknown for 2-word value")
		}
		ctx.FreeDesc(&d1)
		d1 = tmpPair
	}
	if d1.Loc != scm.LocRegPair && d1.Loc != scm.LocStackPair {
		panic("jit: generic call arg expects 2-word value ((*OverlayBlob).resolveBlob arg1)")
	}
	ctx.SyncDesc(&thisptr)
	ctx.SyncDesc(&d1)
	d2 := ctx.EmitGoCallScalar(scm.GoFuncAddr((*OverlayBlob).resolveBlob), []scm.JITValueDesc{thisptr, d1}, 2)
	d2.NoHeapPointer = false
	ctx.BindReg(d2.Reg, &d2)
	ctx.BindReg(d2.Reg2, &d2)
	ctx.FreeDesc(&d1)
	if d2.Loc == scm.LocImm {
		if result.Loc == scm.LocAny {
			return d2
		}
	}
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	ctx.SyncDesc(&d2)
	if d2.Loc == scm.LocRegPair || d2.Loc == scm.LocStackPair || d2.Loc == scm.LocInputPair {
		ctx.EmitMovPairToResult(&d2, &result)
		result.Type = d2.Type
	} else {
		switch d2.Type {
		case scm.TagBool:
			ctx.EmitMakeBool(result, d2)
			result.Type = scm.TagBool
		case scm.TagInt:
			ctx.EmitMakeInt(result, d2)
			result.Type = scm.TagInt
		case scm.TagFloat:
			ctx.EmitMakeFloat(result, d2)
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

func (s *OverlayBlob) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(31))                 // 31 = OverlayBlob
	binary.Write(f, binary.LittleEndian, uint8(overlayBlobVersion)) // version byte (was '1' in legacy)
	var pad [6]byte
	f.Write(pad[:])                                 // remaining alignment padding (was "234567")
	binary.Write(f, binary.LittleEndian, uint64(0)) // size=0: no inline blobs
	s.Base.Serialize(f)                             // serialize base
}

func (s *OverlayBlob) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	var pad [6]byte
	f.Read(pad[:])
	switch version {
	case 0, '1': // '1'=49: legacy pre-versioning dummy byte; treat as v0
		return s.deserializeBlobV0(f)
	default:
		panic(fmt.Sprintf("OverlayBlob: unknown version %d", version))
	}
}

func (s *OverlayBlob) deserializeBlobV0(f io.Reader) uint {
	var size uint64
	binary.Read(f, binary.LittleEndian, &size) // read size
	s.values = make(map[[32]byte]string)

	if size > 0 {
		// LEGACY: read inline blobs (migration in SetPersistence)
		for i := uint64(0); i < size; i++ {
			var key [32]byte
			f.Read(key[:])
			var l uint64
			binary.Read(f, binary.LittleEndian, &l)
			value := make([]byte, l)
			f.Read(value)
			s.size += uint(l)
			s.values[key] = string(value)
		}
	}
	var basetype uint8
	f.Read(unsafe.Slice(&basetype, 1))
	s.Base = reflect.New(storages[basetype]).Interface().(ColumnStorage)
	l := s.Base.Deserialize(f) // read base
	return l
}

// SetSchema sets the owning database and migrates legacy inline blobs.
func (s *OverlayBlob) SetSchema(db *database) {
	s.schema = db
	s.refs = make(map[string]bool)
	for hash, data := range s.values {
		hexHash := fmt.Sprintf("%x", hash[:])
		db.IncrBlobRefcount(hexHash)
		w := db.persistence.WriteBlob(hexHash)
		io.WriteString(w, data)
		w.Close()
		s.refs[hexHash] = true
	}
	s.values = nil
	s.size = 0
}

var gzipReaderPool sync.Pool

func gunzipReader(compressed io.Reader) (scm.Scmer, bool) {
	var b strings.Builder
	reader, _ := gzipReaderPool.Get().(*gzip.Reader)
	var err error
	if reader == nil {
		reader, err = gzip.NewReader(compressed)
	} else {
		err = reader.Reset(compressed)
	}
	if err == io.EOF {
		return scm.NewNil(), false
	}
	if err != nil {
		panic(err)
	}
	_, copyErr := io.Copy(&b, reader)
	closeErr := reader.Close()
	gzipReaderPool.Put(reader)
	if copyErr != nil {
		panic(copyErr)
	}
	if closeErr != nil {
		panic(closeErr)
	}
	return scm.NewString(b.String()), true
}

func gunzipValue(gzipped string) scm.Scmer {
	value, ok := gunzipReader(strings.NewReader(gzipped))
	if !ok {
		panic("empty gzip value")
	}
	return value
}

func (s *OverlayBlob) GetCachedReader() ColumnReader { return s }

func (s *OverlayBlob) GetValue(i uint32) scm.Scmer {
	return s.resolveBlob(s.Base.GetValue(i))
}

// GetValueRange and GetValueMulti bulk-fetch the base storage (one call
// instead of n GetValue calls) and then resolve the blob-escape encoding for
// each result in place.
func (s *OverlayBlob) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.Base.GetValueRange(recid, count, target, stride)
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = s.resolveBlob(target[idx])
		idx += stride
	}
}

func (s *OverlayBlob) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.Base.GetValueMulti(recids, target, stride)
	idx := 0
	for range recids {
		target[idx] = s.resolveBlob(target[idx])
		idx += stride
	}
}

// resolveBlob turns a base-storage value into its logical value: a plain
// value passes through unchanged; a "!"-prefixed string is either an
// escaped literal ("!!...") or a blob reference that must be loaded from
// persistence (or the in-memory build-time cache) and gunzipped.
func (s *OverlayBlob) resolveBlob(v scm.Scmer) scm.Scmer {
	if v.IsString() {
		vs := v.String()
		if vs != "" && vs[0] == '!' {
			if len(vs) > 1 && vs[1] == '!' {
				return scm.NewString(vs[1:]) // escaped string
			}
			hashKey := *(*[32]byte)(unsafe.Pointer(unsafe.StringData(vs[1:])))

			// load from persistence (no RAM caching)
			if s.schema != nil && s.schema.persistence != nil {
				hexHash := fmt.Sprintf("%x", hashKey[:])
				r := s.schema.persistence.ReadBlob(hexHash)
				if _, readFailed := r.(ErrorReader); !readFailed {
					value, ok := gunzipReader(r)
					r.Close()
					if ok {
						return value
					}
				} else {
					r.Close()
				}
			}

			// fallback: check in-memory values (memory-mode or during build)
			if s.values != nil {
				if val, ok := s.values[hashKey]; ok {
					return gunzipValue(val)
				}
			}

			return scm.NewNil() // value lost
		}
	}
	return v
}

func (s *OverlayBlob) prepare() {
	// set up scan
	s.Base.prepare()
}
func (s *OverlayBlob) scan(i uint32, value scm.Scmer) {
	if value.IsString() {
		vs := value.String()
		if len(vs) > maxInlineBlobBytes {
			h := sha256.New()
			io.WriteString(h, vs)
			s.Base.scan(i, scm.NewString("!"+string(h.Sum(nil))))
		} else {
			if vs != "" && vs[0] == '!' {
				s.Base.scan(i, scm.NewString("!"+vs))
			} else {
				s.Base.scan(i, value)
			}
		}
		return
	}
	s.Base.scan(i, value)
}
func (s *OverlayBlob) init(i uint32) {
	s.values = make(map[[32]byte]string)
	s.size = 0
	s.refs = make(map[string]bool)
	s.Base.init(i)
}
func (s *OverlayBlob) build(i uint32, value scm.Scmer) {
	// TODO: for rebuild/repartition, allow passing raw gzipped blob data
	// through without decompressing+recompressing. When the source column
	// is also an OverlayBlob we could copy the hash reference and the
	// compressed blob file directly, avoiding the gzip round-trip entirely.
	if value.IsString() {
		vs := value.String()
		if len(vs) > maxInlineBlobBytes {
			h := sha256.New()
			io.WriteString(h, vs)
			hashsum := h.Sum(nil)
			hashKey := *(*[32]byte)(unsafe.Pointer(&hashsum[0]))
			s.Base.build(i, scm.NewString("!"+string(hashsum)))

			// deduplicate: only compress+write if not already seen
			if _, exists := s.values[hashKey]; !exists {
				var b strings.Builder
				z := gzip.NewWriter(&b)
				_, _ = io.Copy(z, strings.NewReader(vs))
				z.Close()
				gzipped := b.String()
				s.size += uint(len(gzipped))
				s.values[hashKey] = gzipped

				// write-through to persistence (refcount first, then file)
				if s.schema != nil && s.schema.persistence != nil {
					hexHash := fmt.Sprintf("%x", hashKey[:])
					if !s.refs[hexHash] {
						s.schema.IncrBlobRefcount(hexHash)
						s.refs[hexHash] = true
					}
					w := s.schema.persistence.WriteBlob(hexHash)
					io.WriteString(w, gzipped)
					w.Close()
				}
			}
		} else {
			if vs != "" && vs[0] == '!' {
				s.Base.build(i, scm.NewString("!"+vs))
			} else {
				s.Base.build(i, value)
			}
		}
		return
	}
	s.Base.build(i, value)
}
func (s *OverlayBlob) finish() {
	if s.schema != nil {
		s.values = nil
		s.size = 0
	}
	s.Base.finish()
}

// appendBlobReferences adds the content-addressed objects owned by this column
// generation. References are generation metadata: cleanup may delete a blob
// only after every active generation supplied this proof.
func (s *OverlayBlob) appendBlobReferences(dst map[string]struct{}, count uint32) {
	if len(s.refs) > 0 {
		for hash := range s.refs {
			dst[hash] = struct{}{}
		}
		return
	}
	for i := uint32(0); i < count; i++ {
		value := s.Base.GetValue(i)
		if !value.IsString() {
			continue
		}
		raw := value.String()
		if len(raw) == 33 && raw[0] == '!' && raw[1] != '!' {
			dst[fmt.Sprintf("%x", []byte(raw[1:]))] = struct{}{}
		}
	}
}
func (s *OverlayBlob) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

// ReleaseBlobs decrements RC for all blob hashes referenced by this OverlayBlob.
func (s *OverlayBlob) ReleaseBlobs(count uint) {
	if s.schema == nil {
		return
	}

	// Case 1: refs from build() available
	if s.refs != nil && len(s.refs) > 0 {
		for hexHash := range s.refs {
			s.schema.DecrBlobRefcount(hexHash)
		}
		s.refs = nil
		return
	}

	// Case 2: loaded from disk, refs unknown -- scan Base column
	seen := make(map[string]bool)
	for i := uint32(0); i < uint32(count); i++ {
		v := s.Base.GetValue(i)
		if v.IsString() {
			vs := v.String()
			// Blob reference: "!" + 32 bytes hash, NOT "!!" (escaped)
			if len(vs) == 33 && vs[0] == '!' && vs[1] != '!' {
				hashKey := *(*[32]byte)(unsafe.Pointer(unsafe.StringData(vs[1:])))
				hexHash := fmt.Sprintf("%x", hashKey[:])
				if !seen[hexHash] {
					seen[hexHash] = true
					s.schema.DecrBlobRefcount(hexHash)
				}
			}
		}
	}
}

func (s *OverlayBlob) DistinctCount() uint { return s.Base.DistinctCount() }
