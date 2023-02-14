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
	hasNegative bool
	hasNull bool
	null uint64 // which value is null
}

func toInt(x scm.Scmer) int64 {
	switch v := x.(type) {
		case float64:
			return int64(v)
		case uint:
			return int64(v)
		case int64:
			return v
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
	if (s.hasNegative) { // with sign expansion
		v := s.getValueInt(i)
		if s.hasNull && uint64(v) == s.null {
			return nil
		}
		return float64(v)
	} else { // without sign expansion
		v := s.getValueUInt(i)
		if s.hasNull && v == s.null {
			return nil
		}
		return float64(v)
	}
}

func (s *StorageInt) getValueInt(i uint) int64 {
	bitpos := i * uint(s.bitsize)

	v := s.chunk[bitpos / 64] << (bitpos % 64) // align to leftmost position
	if bitpos % 64 + uint(s.bitsize) >= 64 {
		v = v | s.chunk[bitpos / 64 + 1] >> (64 - bitpos % 64)
	}

	return int64(v) >> (64 - uint(s.bitsize)) // shift right preserving sign
}

func (s *StorageInt) getValueUInt(i uint) uint64 {
	bitpos := i * uint(s.bitsize)

	v := s.chunk[bitpos / 64] << (bitpos % 64) // align to leftmost position
	if bitpos % 64 + uint(s.bitsize) >= 64 {
		v = v | s.chunk[bitpos / 64 + 1] >> (64 - bitpos % 64)
	}

	return uint64(v) >> (64 - uint(s.bitsize)) // shift right without sign
}

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.hasNegative = false
}
func (s *StorageInt) scan(i uint, value scm.Scmer) {
	// storage is so simple, dont need scan
	if value == nil {
		s.hasNull = true
		return
	}
	v := toInt(value)
	if v >= int64(s.null) {
		// mark 1+highest value as null
		s.null = uint64(v) + 1
	}
	if v < 0 {
		s.hasNegative = true
		v = -v
	}
	l := uint8(bits.Len64(uint64(v)))
	if l > s.bitsize {
		s.bitsize = l
	}
}
func (s *StorageInt) init(i uint) {
	if s.hasNull {
		// need an extra bit because of null??
		l := uint8(bits.Len64(uint64(s.null)))
		if l > s.bitsize {
			s.bitsize = l
		}
	}
	if s.hasNegative {
		s.bitsize = s.bitsize + 1
	}
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
	bitpos := i * uint(s.bitsize)
	v := uint64(vi) << (64 - uint(s.bitsize)) // shift value to the leftmost position of 64bit int
	s.chunk[bitpos / 64] = s.chunk[bitpos / 64] | (v >> (bitpos % 64)) // first chunk
	if (bitpos % 64 + uint(s.bitsize) > 64) {
		s.chunk[bitpos / 64 + 1] = s.chunk[bitpos / 64 + 1] | v << (64 - bitpos % 64) // second chunk
	}
}
func (s *StorageInt) finish() {
}
func (s *StorageInt) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}
