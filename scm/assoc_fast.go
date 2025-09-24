/*
Copyright (C) 2025  Carl-Philip Hänsch

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
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"reflect"
)

// Stable seed for hashing to ensure consistent indices across Set/Get calls.
var fastDictSeed maphash.Seed

func init() {
	fastDictSeed = maphash.MakeSeed()
}

// FastDict: shard-local assoc optimized for frequent set/merge operations.
// Implementation uses a flat pairs array plus a lightweight hash index
// to avoid O(N^2) behavior as it grows.
type FastDict struct {
	Pairs []Scmer          // [k0, v0, k1, v1, ...]
	index map[uint64][]int // hash -> positions (indices into Pairs, even only)
}

func NewFastDict(capacityPairs int) *FastDict {
	if capacityPairs < 0 {
		capacityPairs = 0
	}
	return &FastDict{Pairs: make([]Scmer, 0, capacityPairs*2), index: make(map[uint64][]int)}
}

func (d *FastDict) Iterate(fn func(k, v Scmer) bool) {
	for i := 0; i < len(d.Pairs); i += 2 {
		if !fn(d.Pairs[i], d.Pairs[i+1]) {
			return
		}
	}
}

// HashKey computes a stable hash for a Scheme value.
// It avoids allocating intermediate strings by inspecting types and
// feeding bytes directly to a streaming hasher. Lists are hashed by
// recursively hashing their elements with structural markers.
func HashKey(k Scmer) uint64 {
	var h maphash.Hash
	h.SetSeed(fastDictSeed)
	var writeScmer func(v Scmer)
	writeScmer = func(v Scmer) {
		switch auxTag(v.aux) {
		case tagNil:
			h.WriteByte(0)
		case tagBool:
			h.WriteByte(1)
			if v.Bool() {
				h.WriteByte(1)
			} else {
				h.WriteByte(0)
			}
		case tagInt:
			h.WriteByte(2)
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(v.Int()))
			h.Write(b[:])
		case tagFloat:
			h.WriteByte(3)
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(v.Float()))
			h.Write(b[:])
		case tagString:
			h.WriteByte(4)
			h.WriteString(v.String())
		case tagSymbol:
			h.WriteByte(5)
			h.WriteString(v.String())
		case tagSlice:
			h.WriteByte(6)
			// write length to reduce collisions for different list sizes
			var b [8]byte
			slice := v.Slice()
			binary.LittleEndian.PutUint64(b[:], uint64(len(slice)))
			h.Write(b[:])
			for _, el := range slice {
				writeScmer(el)
			}
		case tagVector:
			h.WriteByte(7)
			vec := v.Vector()
			var b [8]byte
			binary.LittleEndian.PutUint64(b[:], uint64(len(vec)))
			h.Write(b[:])
			for _, el := range vec {
				var bb [8]byte
				binary.LittleEndian.PutUint64(bb[:], math.Float64bits(el))
				h.Write(bb[:])
			}
		case tagFunc:
			h.WriteByte(8)
			fn := v.Func()
			if reflect.ValueOf(fn).Kind() == reflect.Func {
				var b [8]byte
				binary.LittleEndian.PutUint64(b[:], reflect.ValueOf(fn).Pointer())
				h.Write(b[:])
			} else {
				h.WriteString(fmt.Sprintf("%v", fn))
			}
		case tagAny:
			if fd, ok := v.Any().(*FastDict); ok {
				// Hash as list of pairs to match []Scmer assoc representation
				h.WriteByte(6)
				var b [8]byte
				binary.LittleEndian.PutUint64(b[:], uint64(len(fd.Pairs)))
				h.Write(b[:])
				for i := 0; i < len(fd.Pairs); i += 2 {
					writeScmer(fd.Pairs[i])
					writeScmer(fd.Pairs[i+1])
				}
				return
			}
			fallback := fmt.Sprintf("%T", v.Any())
			h.WriteByte(255)
			h.WriteString(fallback)
		default:
			// Hash as list of pairs to match []Scmer assoc representation
			h.WriteByte(255)
			h.WriteString(v.String())
		}
	}
	writeScmer(k)
	return h.Sum64()
}

func (d *FastDict) findPos(key Scmer, h uint64) (int, bool) {
	if d.index == nil {
		return -1, false
	}
	if bucket, ok := d.index[h]; ok {
		for _, pos := range bucket {
			if Equal(d.Pairs[pos], key) {
				return pos, true
			}
		}
	}
	return -1, false
}

func (d *FastDict) Get(key Scmer) (Scmer, bool) {
	h := HashKey(key)
	if pos, ok := d.findPos(key, h); ok {
		return d.Pairs[pos+1], true
	}
	return NewNil(), false
}

// Set sets or merges a value for key. If merge is nil, it overwrites.
func (d *FastDict) Set(key, value Scmer, merge func(oldV, newV Scmer) Scmer) {
	if d.index == nil {
		d.index = make(map[uint64][]int)
	}
	h := HashKey(key)
	if pos, ok := d.findPos(key, h); ok {
		if merge != nil {
			d.Pairs[pos+1] = merge(d.Pairs[pos+1], value)
		} else {
			d.Pairs[pos+1] = value
		}
		return
	}
	// append new
	pos := len(d.Pairs)
	d.Pairs = append(d.Pairs, key, value)
	d.index[h] = append(d.index[h], pos)
}

func (d *FastDict) ToList() []Scmer { return d.Pairs }
