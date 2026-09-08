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
	storageJITFunctions
	// every overlay has a base
	Base ColumnStorage
	// values: used during build() for dedup, and for legacy inline data
	values map[[32]byte]string
	size   uint
	schema *database       // reference to owning database
	refs   map[string]bool // hex-hashes referenced in this build()
	legacy bool            // v0/ASCII-49 base encoding; preserved until the next rebuild
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
const overlayBlobVersion = 1

// OverlayBlob binary layout (magic byte 31 consumed by shard loader):
//
//	[version uint8]      ← first byte read by Deserialize
//	[pad 6 bytes]        ← alignment padding
//	[size uint64]        ← number of inline blobs (0 when stored externally)
//	[base storage]       ← magic byte + full serialized base column
//
// Version history:
//
//	1 (current): references are "!b" + 32 hash bytes; literals beginning with
//	             "!" are escaped with another "!". The tags cannot collide.
//	0: references were "!" + 32 hash bytes, ambiguous with escaped literals
//	   when the hash begins with "!". The version byte was previously the first byte
//	             of a 7-byte ASCII dummy "1234567" (byte value '1'=49).
//	             Legacy: version byte '1'=49 → treat as v0 (inline blobs still possible).
func (s *OverlayBlob) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
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
	var d0 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*OverlayBlob)(nil).Base)
		r0 := ctx.AllocReg()
		r1 := ctx.AllocRegExcept(r0)
		ctx.EmitMovRegMem64(r0, fieldAddr)
		ctx.EmitMovRegMem64(r1, fieldAddr+8)
		d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d0)
		ctx.BindReg(r1, &d0)
	} else {
		off := int32(unsafe.Offsetof((*OverlayBlob)(nil).Base))
		r2 := ctx.AllocReg()
		r3 := ctx.AllocRegExcept(r2)
		ctx.EmitMovRegMem(r2, thisptr.Reg, off)
		ctx.EmitMovRegMem(r3, thisptr.Reg, off+8)
		d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
		ctx.BindReg(r2, &d0)
		ctx.BindReg(r3, &d0)
	}
	ctx.EnsureDesc(&d0)
	ctx.EnsureDesc(&idxInt)
	d1 := ctx.EmitGoCallScalar(scm.GoFuncAddr(func(receiver ColumnStorage, arg0 uint32) scm.Scmer { return receiver.GetValue(arg0) }), []scm.JITValueDesc{d0, idxInt}, 2)
	ctx.FreeDesc(&idxInt)
	if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
		panic("jit: generic call arg expects 1-word value")
	}
	d1 = scm.JITPrepareScmerGoArg(ctx, d1)
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
	ctx.EmitMovPairToResult(&d2, &result)
	result.Type = d2.Type
	return result
	return result
}

func (s *OverlayBlob) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(31)) // 31 = OverlayBlob
	version := uint8(overlayBlobVersion)
	if s.legacy {
		version = 0
	} // Never relabel a legacy base as the new encoding.
	binary.Write(f, binary.LittleEndian, version)
	var pad [6]byte
	f.Write(pad[:]) // remaining alignment padding (was "234567")
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	// Detached/in-memory columns and legacy inline payloads must survive a
	// serialize/reload before SetSchema migrates them to external storage.
	for hash, compressed := range s.values {
		f.Write(hash[:])
		binary.Write(f, binary.LittleEndian, uint64(len(compressed)))
		io.WriteString(f, compressed)
	}
	s.Base.Serialize(f) // serialize base
}

func (s *OverlayBlob) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	var pad [6]byte
	f.Read(pad[:])
	switch version {
	case 0, '1': // '1'=49: legacy pre-versioning dummy byte; treat as v0
		s.legacy = true
		return s.deserializeBlobV0(f)
	case 1:
		s.legacy = false
		return s.deserializeBlobV1(f)
	default:
		panic(fmt.Sprintf("OverlayBlob: unknown version %d", version))
	}
}

// v1 changes the base string tags, not the enclosing binary layout.
func (s *OverlayBlob) deserializeBlobV1(f io.Reader) uint {
	return s.deserializeBlobV0(f)
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

func (s *OverlayBlob) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

func (s *OverlayBlob) GetValue(i uint32) scm.Scmer {
	return s.resolveBlob(s.Base.GetValue(i))
}

// GetValueRange and GetValueMulti bulk-fetch the base storage (one call
// instead of n GetValue calls) and then resolve the blob-escape encoding for
// each result in place.
//
//jitgen:control-flow-stable recid count target/1 stride
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

//jitgen:control-flow-stable recids/2 target/1 stride
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

// blobReference recognizes references without fetching payloads. For v0, a
// 33-byte "!!..." is ambiguous: it can also be an escaped 32-byte literal.
// Manifest discovery must conservatively retain that candidate hash.
func (s *OverlayBlob) blobReference(raw string) (hash [32]byte, reference, ambiguous bool) {
	if s.legacy {
		if len(raw) == 33 && raw[0] == '!' {
			copy(hash[:], raw[1:])
			return hash, true, raw[1] == '!'
		}
	} else if len(raw) == 34 && raw[:2] == "!b" {
		copy(hash[:], raw[2:])
		return hash, true, false
	}
	return hash, false, false
}

func (s *OverlayBlob) readBlob(hash [32]byte) (scm.Scmer, bool) {
	if s.schema != nil && s.schema.persistence != nil {
		reader := s.schema.persistence.ReadBlob(fmt.Sprintf("%x", hash))
		defer reader.Close()
		if failure, failed := reader.(ErrorReader); failed {
			if !failure.Missing() {
				panic(failure)
			}
		} else {
			value, ok := gunzipReader(reader)
			if !ok {
				panic(fmt.Sprintf("OverlayBlob: empty blob %x", hash))
			}
			return value, true
		}
	}
	if compressed, found := s.values[hash]; found {
		return gunzipValue(compressed), true
	}
	return scm.NewNil(), false
}

// Legacy ambiguous values prefer a present, content-verified blob. Without
// that evidence they retain the old escaped-literal interpretation. An actual
// literal identical to a present blob reference cannot be distinguished in v0;
// only a rebuild from an authoritative source can resolve that case.
func (s *OverlayBlob) resolveBlob(v scm.Scmer) scm.Scmer {
	if !v.IsString() {
		return v
	}
	raw := v.String()
	if raw == "" || raw[0] != '!' {
		return v
	}
	hash, reference, ambiguous := s.blobReference(raw)
	if reference {
		if value, found := s.readBlob(hash); found {
			if ambiguous && sha256.Sum256([]byte(value.String())) != hash {
				panic(fmt.Sprintf("OverlayBlob: legacy blob checksum mismatch %x", hash))
			}
			return value
		}
		if !ambiguous {
			panic(fmt.Sprintf("OverlayBlob: missing blob %x", hash))
		}
	}
	if len(raw) > 1 && raw[1] == '!' {
		return scm.NewString(raw[1:])
	}
	panic("OverlayBlob: malformed reference encoding")
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
			s.Base.scan(i, scm.NewString("!b"+string(h.Sum(nil))))
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
	s.legacy = false
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
			s.Base.build(i, scm.NewString("!b"+string(hashsum)))

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
	s.storageJITFunctions.finish(s)
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
		if hash, reference, _ := s.blobReference(raw); reference {
			dst[fmt.Sprintf("%x", hash)] = struct{}{}
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

	// A legacy ambiguous value is sufficient evidence to retain a blob, but
	// not to decrement ownership: a colliding literal could otherwise delete
	// another column's last reference. Generation-based clean reclaims such
	// blobs after all possible owning generations have been retired.
	seen := make(map[[32]byte]bool)
	for i := uint32(0); i < uint32(count); i++ {
		value := s.Base.GetValue(i)
		if !value.IsString() {
			continue
		}
		hash, reference, ambiguous := s.blobReference(value.String())
		if reference && !ambiguous && !seen[hash] {
			seen[hash] = true
			s.schema.DecrBlobRefcount(fmt.Sprintf("%x", hash))
		}
	}
}

func (s *OverlayBlob) DistinctCount() uint { return s.Base.DistinctCount() }
