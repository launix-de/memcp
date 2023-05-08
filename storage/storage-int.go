/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "fmt"
import "math/bits"
import "github.com/launix-de/memcp/scm"

type StorageInt struct {
	chunk []uint64
	bitsize uint8
	offset int64
	max int64
	hasNull bool
	null uint64 // which value is null
}

func toInt(x scm.Scmer) int64 {
	switch v := x.(type) {
		case float64:
			return int64(v)
		case int:
			return int64(v)
		case uint:
			return int64(v)
		case uint64:
			return int64(v)
		case int64:
			return v
		// TODO: 8 bit, 16 bit, 32 bit
		default:
			return 0
	}
}

func (s *StorageInt) String() string {
	if s.hasNull {
		return fmt.Sprintf("int[%d]NULL", s.bitsize)
	} else {
		return fmt.Sprintf("int[%d]", s.bitsize)
	}
}

func (s *StorageInt) getValue(i uint) scm.Scmer {
	v := s.getValueUInt(i)
	if s.hasNull && v == s.null {
		return nil
	}
	return float64(int64(v) + s.offset)
}

func (s *StorageInt) getValueInt(i uint) int64 {
	bitpos := i * uint(s.bitsize)

	v := s.chunk[bitpos / 64] << (bitpos % 64) // align to leftmost position
	if bitpos % 64 + uint(s.bitsize) > 64 {
		v = v | s.chunk[bitpos / 64 + 1] >> (64 - bitpos % 64)
	}

	return int64(v) >> (64 - uint(s.bitsize)) // shift right preserving sign
}

func (s *StorageInt) getValueUInt(i uint) uint64 {
	bitpos := i * uint(s.bitsize)

	v := s.chunk[bitpos / 64] << (bitpos % 64) // align to leftmost position
	if bitpos % 64 + uint(s.bitsize) > 64 {
		v = v | s.chunk[bitpos / 64 + 1] >> (64 - bitpos % 64)
	}

	return uint64(v) >> (64 - uint(s.bitsize)) // shift right without sign
}

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.offset = int64(1 << 63 - 1)
	s.max = -s.offset - 1
}
func (s *StorageInt) scan(i uint, value scm.Scmer) {
	// storage is so simple, dont need scan
	if value == nil {
		s.hasNull = true
		return
	}
	v := toInt(value)
	if v < s.offset {
		s.offset = v
	}
	if v > s.max {
		s.max = v
	}
}
func (s *StorageInt) init(i uint) {
	v := s.max - s.offset
	if s.hasNull {
		// store the value
		s.null = uint64(v)
		v = v + 1
	}
	if v == -1 {
		// no values at all
		v = 0
		s.offset = 0
		s.null = 0
	}
	s.bitsize = uint8(bits.Len64(uint64(v)))
	if s.bitsize == 0 {
		s.bitsize = 1
	}
	// allocate
	s.chunk = make([]uint64, (i * uint(s.bitsize) + 63) / 64)
	//fmt.Println("Allocate bitsize", s.bitsize)
}
func (s *StorageInt) build(i uint, value scm.Scmer) {
	// store
	vi := toInt(value)
	if value == nil {
		// null value
		vi = int64(s.null)
	}
	vi = vi - s.offset
	bitpos := i * uint(s.bitsize)
	v := uint64(vi) << (64 - uint(s.bitsize)) // shift value to the leftmost position of 64bit int
	s.chunk[bitpos / 64] = s.chunk[bitpos / 64] | (v >> (bitpos % 64)) // first chunk
	if (bitpos % 64 + uint(s.bitsize) > 64) {
		s.chunk[bitpos / 64 + 1] = s.chunk[bitpos / 64 + 1] | v << (64 - bitpos % 64) // second chunk
	}
}
func (s *StorageInt) finish() {
}
func (s *StorageInt) proposeCompression(i uint) ColumnStorage {
	// dont't propose another pass
	return nil
}
