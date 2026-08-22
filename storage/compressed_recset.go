/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import "math/bits"
import "unsafe"

// CompressedRecSet is the deliberately small contract needed by immutable
// search indexes. Implementations may allocate query-local scratch while
// narrowing dst, but never expose a partial set as normal RecSet membership.
type CompressedRecSet interface {
	AndMut(dst *recSetShard)
}

func (s *recSetShard) AndMut(dst *recSetShard) {
	intersectRecSetShardMut(dst, s)
}

type compressedRecSet struct {
	set   CompressedRecSet
	count uint32
	bytes uint32
}

type compressedBitmap struct {
	universe uint32
	words    []uint32
}

func (s *compressedBitmap) AndMut(dst *recSetShard) {
	part := recSetShard{kind: recSetBitmap, universe: s.universe, data: s.words, count: int64(countBitmap(s.words))}
	intersectRecSetShardMut(dst, &part)
}

type packedUint32s struct {
	data  []uint64
	base  uint32
	count uint32
	width uint8
}

func packUint32s(values []uint32) packedUint32s {
	if len(values) == 0 {
		return packedUint32s{}
	}
	base, maximum := values[0], values[0]
	for _, value := range values[1:] {
		if value < base {
			base = value
		}
		if value > maximum {
			maximum = value
		}
	}
	width := uint8(bits.Len32(maximum - base))
	if width == 0 {
		width = 1
	}
	packed := packedUint32s{
		data:  make([]uint64, (uint64(len(values))*uint64(width)+63)/64+1),
		base:  base,
		count: uint32(len(values)),
		width: width,
	}
	for i, value := range values {
		packed.set(uint32(i), value-base)
	}
	return packed
}

func (s *packedUint32s) set(index, value uint32) {
	bit := uint64(index) * uint64(s.width)
	word, shift := bit>>6, bit&63
	s.data[word] |= uint64(value) << shift
	if shift+uint64(s.width) > 64 {
		s.data[word+1] |= uint64(value) >> (64 - shift)
	}
}

func (s *packedUint32s) get(index uint32) uint32 {
	bit := uint64(index) * uint64(s.width)
	word, shift := bit>>6, bit&63
	value := s.data[word] >> shift
	if shift+uint64(s.width) > 64 {
		value |= s.data[word+1] << (64 - shift)
	}
	mask := uint64(1)<<s.width - 1
	return s.base + uint32(value&mask)
}

type compressedPositive struct {
	universe uint32
	values   packedUint32s
}

func (s *compressedPositive) AndMut(dst *recSetShard) {
	values := make([]uint32, s.values.count)
	for i := range values {
		values[i] = s.values.get(uint32(i))
	}
	part := recSetShard{kind: recSetPositive, universe: s.universe, data: values, used: uint32(len(values)), count: int64(len(values))}
	intersectRecSetShardMut(dst, &part)
}

type compressedRanges struct {
	universe uint32
	count    uint32
	bases    packedUint32s
	lengths  packedUint32s
}

func (s *compressedRanges) AndMut(dst *recSetShard) {
	ranges := make([]uint32, s.bases.count*2)
	for i := uint32(0); i < s.bases.count; i++ {
		ranges[i*2] = s.bases.get(i)
		ranges[i*2+1] = s.lengths.get(i)
	}
	part := recSetShard{kind: recSetRanges, universe: s.universe, data: ranges, used: s.bases.count, count: int64(s.count)}
	intersectRecSetShardMut(dst, &part)
}

const ransByteLowerBound uint32 = 1 << 23

type compressedRANSBitmap struct {
	universe uint32
	ones     uint32
	freqOne  uint16
	state    uint32
	data     []byte
}

func encodeRANSBitmap(words []uint32, universe, ones uint32) *compressedRANSBitmap {
	freqOne := uint32((uint64(ones)*256 + uint64(universe)/2) / uint64(universe))
	if freqOne == 0 {
		freqOne = 1
	}
	if freqOne == 256 {
		freqOne = 255
	}
	state := ransByteLowerBound
	encoded := make([]byte, 0, universe/8)
	for position := universe; position > 0; position-- {
		one := words[(position-1)>>5]&(uint32(1)<<((position-1)&31)) != 0
		frequency, start := uint32(256)-freqOne, freqOne
		if one {
			frequency, start = freqOne, 0
		}
		maximum := ((ransByteLowerBound >> 8) << 8) * frequency
		for state >= maximum {
			encoded = append(encoded, byte(state))
			state >>= 8
		}
		state = (state/frequency)*256 + state%frequency + start
	}
	return &compressedRANSBitmap{universe: universe, ones: ones, freqOne: uint16(freqOne), state: state, data: encoded}
}

func (s *compressedRANSBitmap) AndMut(dst *recSetShard) {
	words := make([]uint32, (s.universe+31)/32)
	state := s.state
	input := len(s.data) - 1
	freqOne := uint32(s.freqOne)
	for position := uint32(0); position < s.universe; position++ {
		slot := state & 255
		frequency, start := uint32(256)-freqOne, freqOne
		if slot < freqOne {
			words[position>>5] |= uint32(1) << (position & 31)
			frequency, start = freqOne, 0
		}
		state = frequency*(state>>8) + slot - start
		for state < ransByteLowerBound && input >= 0 {
			state = state<<8 | uint32(s.data[input])
			input--
		}
	}
	part := recSetShard{kind: recSetBitmap, universe: s.universe, data: words, count: int64(s.ones)}
	intersectRecSetShardMut(dst, &part)
}

func countBitmap(words []uint32) uint32 {
	var count uint32
	for _, word := range words {
		count += uint32(bits.OnesCount32(word))
	}
	return count
}

func compressRecSetBitmap(words []uint32, universe uint32) compressedRecSet {
	count := countBitmap(words)
	ids := make([]uint32, 0, count)
	bases := make([]uint32, 0)
	lengths := make([]uint32, 0)
	var runBase, runEnd uint32
	haveRun := false
	for wordIndex, source := range words {
		for source != 0 {
			position := uint32(wordIndex*32 + bits.TrailingZeros32(source))
			if position >= universe {
				break
			}
			ids = append(ids, position)
			if haveRun && position == runEnd {
				runEnd++
			} else {
				if haveRun {
					bases = append(bases, runBase)
					lengths = append(lengths, runEnd-runBase)
				}
				runBase, runEnd, haveRun = position, position+1, true
			}
			source &= source - 1
		}
	}
	if haveRun {
		bases = append(bases, runBase)
		lengths = append(lengths, runEnd-runBase)
	}

	raw := &compressedBitmap{universe: universe, words: append([]uint32(nil), words...)}
	best := compressedRecSet{set: raw, count: count, bytes: uint32(unsafe.Sizeof(*raw)) + uint32(len(raw.words))*4}
	positive := &compressedPositive{universe: universe, values: packUint32s(ids)}
	if size := uint32(unsafe.Sizeof(*positive)) + uint32(len(positive.values.data))*8; size < best.bytes {
		best = compressedRecSet{set: positive, count: count, bytes: size}
	}
	ranges := &compressedRanges{universe: universe, count: count, bases: packUint32s(bases), lengths: packUint32s(lengths)}
	if size := uint32(unsafe.Sizeof(*ranges)) + uint32(len(ranges.bases.data)+len(ranges.lengths.data))*8; size < best.bytes {
		best = compressedRecSet{set: ranges, count: count, bytes: size}
	}
	if universe > 0 && count > 0 && count < universe {
		rans := encodeRANSBitmap(words, universe, count)
		if size := uint32(unsafe.Sizeof(*rans)) + uint32(len(rans.data)); size < best.bytes {
			best = compressedRecSet{set: rans, count: count, bytes: size}
		}
	}
	return best
}
