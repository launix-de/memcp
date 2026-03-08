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
		w := db.persistence.WriteBlob(hexHash)
		io.WriteString(w, data)
		w.Close()
		db.IncrBlobRefcount(hexHash)
		s.refs[hexHash] = true
	}
	s.values = nil
	s.size = 0
}

func gunzipValue(gzipped string) scm.Scmer {
	var b strings.Builder
	reader, err := gzip.NewReader(strings.NewReader(gzipped))
	if err != nil {
		panic(err)
	}
	_, _ = io.Copy(&b, reader)
	reader.Close()
	return scm.NewString(b.String())
}

func (s *OverlayBlob) GetCachedReader() ColumnReader { return s }

func (s *OverlayBlob) GetValue(i uint32) scm.Scmer {
	v := s.Base.GetValue(i)
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
				data, err := io.ReadAll(r)
				r.Close()
				if err == nil && len(data) > 0 {
					return gunzipValue(string(data))
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
		if len(vs) > 255 {
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
		if len(vs) > 255 {
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

				// write-through to persistence
				if s.schema != nil && s.schema.persistence != nil {
					hexHash := fmt.Sprintf("%x", hashKey[:])
					w := s.schema.persistence.WriteBlob(hexHash)
					io.WriteString(w, gzipped)
					w.Close()
					if !s.refs[hexHash] {
						s.schema.IncrBlobRefcount(hexHash)
						s.refs[hexHash] = true
					}
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
