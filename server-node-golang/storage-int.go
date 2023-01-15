package main

import "math/bits"
//import "fmt"

type StorageInt struct {
	chunk []uint64
	bitsize uint8
	hasNegative bool
}

func (s *StorageInt) getValue(i uint) scmer {
	bitpos := i * uint(s.bitsize)

	v := s.chunk[bitpos / 64] << (bitpos % 64) // align to leftmost position
	if bitpos % 64 + uint(s.bitsize) > 64 {
		v = v | s.chunk[bitpos / 64 + 1] >> (64 - bitpos % 64)
	}

	if (s.hasNegative) {
		return number(int64(v) >> (64 - uint(s.bitsize))) // shift right preserving sign
	} else {
		return number(uint64(v) >> (64 - uint(s.bitsize))) // shift right without sign
	}
}

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.hasNegative = false
}
func (s *StorageInt) scan(i uint, value scmer) {
	// storage is so simple, dont need scan
	v := int64(value.(float64))
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
	// allocate
	if s.hasNegative {
		s.bitsize = s.bitsize + 1
	}
	if s.bitsize == 0 {
		s.bitsize = 1
	}
	s.chunk = make([]uint64, (i * uint(s.bitsize) + 63) / 64)
	//fmt.Println("Allocate bitsize", s.bitsize)
}
func (s *StorageInt) build(i uint, value scmer) {
	// store
	bitpos := i * uint(s.bitsize)
	v := uint64(int64(value.(float64)) << (64 - uint(s.bitsize))) // shift value to the leftmost position of 64bit int
	s.chunk[bitpos / 64] = s.chunk[bitpos / 64] | (v >> (bitpos % 64)) // first chunk
	if (bitpos % 64 + uint(s.bitsize) > 64) {
		s.chunk[bitpos / 64 + 1] = s.chunk[bitpos / 64 + 1] | v << (64 - bitpos % 64) // second chunk
	}
}
func (s *StorageInt) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}
