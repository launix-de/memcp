/*
Copyright (C) 2024  Carl-Philip Hänsch

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
	// values
	values map[[32]byte]string // gzipped contents content addressable
	size uint
}

func (s *OverlayBlob) Size() uint {
	return 48 + 48 * uint(len(s.values)) + s.size + s.Base.Size()
}

func (s *OverlayBlob) String() string {
	return fmt.Sprintf("overlay[%dx zip-blob %d]+%s", len(s.values), s.size, s.Base.String())
}

func (s *OverlayBlob) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(31)) // 31 = OverlayBlob
	io.WriteString(f, "1234567") // dummy
	var size uint64 = uint64(len(s.values))
	binary.Write(f, binary.LittleEndian, size) // write number of overlay items
	for k, v := range s.values {
		f.Write(k[:])
		binary.Write(f, binary.LittleEndian, uint64(len(v))) // write length
		io.WriteString(f, v) // write content
	}
	s.Base.Serialize(f) // serialize base
}

func (s *OverlayBlob) Deserialize(f io.Reader) uint {
	var dummy [7]byte
	f.Read(dummy[:]) // read padding

	var size uint64
	binary.Read(f, binary.LittleEndian, &size) // read size
	s.values = make(map[[32]byte]string)

	for i := uint64(0); i < size; i++ {
		var key [32]byte
		f.Read(key[:])
		var l uint64
		binary.Read(f, binary.LittleEndian, &l)
		value := make([]byte, l)
		f.Read(value)
		s.size += uint(l) // statistics
		s.values[key] = string(value)
	}
	var basetype uint8
	f.Read(unsafe.Slice(&basetype, 1))
	s.Base = reflect.New(storages[basetype]).Interface().(ColumnStorage)
	l := s.Base.Deserialize(f) // read base
	return l
}

func (s *OverlayBlob) GetValue(i uint) scm.Scmer {
	v := s.Base.GetValue(i)
	switch v_ := v.(type) {
		case string:
			if v_ != "" && v_[0] == '!' {
				if v_[1] == '!' {
					return v_[1:] // escaped string
				} else {
					// unpack from storage
					if v, ok := s.values[*(*[32]byte)(unsafe.Pointer(unsafe.StringData(v_[1:])))]; ok {
						var b strings.Builder
						reader, err := gzip.NewReader(strings.NewReader(v))
						if err != nil {
							panic(err)
						}
						io.Copy(&b, reader)
						reader.Close()
						return b.String()
					}
					return nil // value was lost (this should not happen)
				}
			} else {
				return v
			}
		default:
			return v
	}
}

func (s *OverlayBlob) prepare() {
	// set up scan
	s.Base.prepare()
}
func (s *OverlayBlob) scan(i uint, value scm.Scmer) {
	switch v_ := value.(type) {
		case scm.LazyString:
			if v_.Hash != "" {
				s.Base.scan(i, "!" + v_.Hash)
			} else {
				s.Base.scan(i, v_.GetValue())
			}
		case string:
			if len(v_) > 255 {
				h := sha256.New()
				io.WriteString(h, v_)
				s.Base.scan(i, fmt.Sprintf("!%s", h.Sum(nil)))
			} else {
				if v_ != "" && v_[0] == '!' {
					s.Base.scan(i, "!" + v_) // escape strings that start with !
				} else {
					s.Base.scan(i, value)
				}
			}
		default:
			s.Base.scan(i, value)
	}
}
func (s *OverlayBlob) init(i uint) {
	s.values = make(map[[32]byte]string)
	s.size = 0
	s.Base.init(i)
}
func (s *OverlayBlob) build(i uint, value scm.Scmer) {
	switch v_ := value.(type) {
		case string:
			if len(v_) > 255 {
				h := sha256.New()
				io.WriteString(h, v_)
				hashsum := h.Sum(nil)
				s.Base.build(i, fmt.Sprintf("!%s", hashsum))
				var b strings.Builder
				z := gzip.NewWriter(&b)
				io.Copy(z, strings.NewReader(v_))
				z.Close()
				s.size += uint(b.Len())
				s.values[*(*[32]byte)(unsafe.Pointer(&hashsum[0]))] = b.String()
			} else {
				if v_ != "" && v_[0] == '!' {
					s.Base.build(i, "!" + v_) // escape strings that start with !
				} else {
					s.Base.build(i, value)
				}
			}
		default:
			s.Base.build(i, value)
	}
}
func (s *OverlayBlob) finish() {
	s.Base.finish()
}
func (s *OverlayBlob) proposeCompression(i uint) ColumnStorage {
	// dont't propose another pass
	return nil
}


