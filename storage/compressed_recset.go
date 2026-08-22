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
	if dst == nil || dst.count == 0 {
		return
	}
	if s == nil || s.count == 0 {
		dst.kind = recSetRanges
		dst.data = nil
		dst.used = 0
		dst.count = 0
		return
	}
	if dst.universe != s.universe {
		panic("recset mutable intersection: universe mismatch")
	}
	switch s.kind {
	case recSetBitmap:
		(&compressedBitmap{universe: s.universe, words: s.data}).AndMut(dst)
	case recSetPositive:
		positiveValuesAndMut(dst, s.listedValues())
	case recSetRanges:
		rangeValuesAndMut(dst, s.listedRanges(), s.count)
	}
}

func positiveValuesAndMut(dst *recSetShard, source []uint32) {
	if recSetShardIsFull(dst) {
		dst.kind = recSetPositive
		dst.data = append([]uint32(nil), source...)
		dst.used = uint32(len(source))
		dst.count = int64(len(source))
		return
	}
	if dst.kind == recSetRanges {
		maximumMatches := min(uint64(len(source)), uint64(dst.count))
		if maximumMatches*4 < uint64((dst.universe+31)/32*4) {
			ranges := dst.listedRanges()
			result := make([]uint32, 0, maximumMatches)
			rangePosition := 0
			for _, candidate := range source {
				for rangePosition < len(ranges) && ranges[rangePosition]+ranges[rangePosition+1] <= candidate {
					rangePosition += 2
				}
				if rangePosition == len(ranges) {
					break
				}
				if candidate >= ranges[rangePosition] {
					result = append(result, candidate)
				}
			}
			dst.kind = recSetPositive
			dst.data = result
			dst.used = uint32(len(result))
			dst.count = int64(len(result))
			return
		}
		ranges := dst.listedRanges()
		result := make([]uint32, (dst.universe+31)/32)
		rangePosition := 0
		for _, candidate := range source {
			for rangePosition < len(ranges) && ranges[rangePosition]+ranges[rangePosition+1] <= candidate {
				rangePosition += 2
			}
			if rangePosition == len(ranges) {
				break
			}
			if candidate >= ranges[rangePosition] {
				result[candidate>>5] |= uint32(1) << (candidate & 31)
			}
		}
		dst.kind = recSetBitmap
		dst.data = result
		dst.used = 0
		dst.count = int64(countBitmap(result))
		return
	}
	if dst.kind == recSetBitmap {
		cursor := uint32(0)
		for _, candidate := range source {
			clearRecSetBitmapRange(dst.data, cursor, candidate)
			cursor = candidate + 1
		}
		clearRecSetBitmapRange(dst.data, cursor, dst.universe)
		dst.count = int64(countBitmap(dst.data))
		return
	}
	values := dst.listedValues()
	left, right, kept := 0, 0, 0
	for left < len(values) && right < len(source) {
		switch {
		case values[left] < source[right]:
			left++
		case values[left] > source[right]:
			right++
		default:
			values[kept] = values[left]
			kept++
			left++
			right++
		}
	}
	dst.data = dst.data[:kept]
	dst.used = uint32(kept)
	dst.count = int64(kept)
}

func rangeValuesAndMut(dst *recSetShard, source []uint32, sourceCount int64) {
	if len(source) == 2 && source[0] == 0 && source[1] == dst.universe {
		return
	}
	if recSetShardIsFull(dst) {
		dst.kind = recSetRanges
		dst.data = append([]uint32(nil), source...)
		dst.used = uint32(len(source) / 2)
		dst.count = sourceCount
		return
	}
	if dst.kind == recSetPositive {
		values := dst.listedValues()
		left, kept := 0, 0
		for pair := 0; pair < len(source) && left < len(values); pair += 2 {
			base, end := source[pair], source[pair]+source[pair+1]
			for left < len(values) && values[left] < base {
				left++
			}
			for left < len(values) && values[left] < end {
				values[kept] = values[left]
				kept++
				left++
			}
		}
		dst.data = dst.data[:kept]
		dst.used = uint32(kept)
		dst.count = int64(kept)
		return
	}
	if dst.kind == recSetRanges {
		maximumRangeBytes := uint64(dst.used+uint32(len(source)/2)) * 8
		if maximumRangeBytes < uint64((dst.universe+31)/32*4) {
			rangeCount, _ := intersectRangeValues(dst.listedRanges(), source, nil, nil)
			result := make([]uint32, rangeCount*2)
			_, rowCount := intersectRangeValues(dst.listedRanges(), source, result, nil)
			dst.data = result
			dst.used = rangeCount
			dst.count = rowCount
			return
		}
		result := make([]uint32, (dst.universe+31)/32)
		_, rowCount := intersectRangeValues(dst.listedRanges(), source, nil, result)
		dst.kind = recSetBitmap
		dst.data = result
		dst.used = 0
		dst.count = rowCount
		return
	}
	cursor := uint32(0)
	for pair := 0; pair < len(source); pair += 2 {
		base, end := source[pair], source[pair]+source[pair+1]
		clearRecSetBitmapRange(dst.data, cursor, base)
		cursor = min(end, dst.universe)
	}
	clearRecSetBitmapRange(dst.data, cursor, dst.universe)
	dst.count = int64(countBitmap(dst.data))
}

func intersectRangeValues(left, right, output, bitmap []uint32) (uint32, int64) {
	leftPosition, rightPosition := 0, 0
	var outputRanges uint32
	var outputRows int64
	for leftPosition < len(left) && rightPosition < len(right) {
		leftBase, leftEnd := left[leftPosition], left[leftPosition]+left[leftPosition+1]
		rightBase, rightEnd := right[rightPosition], right[rightPosition]+right[rightPosition+1]
		base, end := max(leftBase, rightBase), min(leftEnd, rightEnd)
		if base < end {
			if output != nil {
				output[outputRanges*2] = base
				output[outputRanges*2+1] = end - base
			}
			if bitmap != nil {
				setBitmapRange(bitmap, base, end)
			}
			outputRanges++
			outputRows += int64(end - base)
		}
		if leftEnd < rightEnd {
			leftPosition += 2
		} else {
			rightPosition += 2
		}
	}
	return outputRanges, outputRows
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
	if !compressedRecSetCanNarrow(dst, s.universe) {
		return
	}
	if recSetShardIsFull(dst) {
		dst.kind = recSetBitmap
		dst.data = append([]uint32(nil), s.words...)
		if tail := s.universe & 31; tail != 0 && len(dst.data) > 0 {
			dst.data[len(dst.data)-1] &= uint32(1)<<tail - 1
		}
		dst.used = 0
		dst.count = int64(countBitmap(dst.data))
		return
	}
	switch dst.kind {
	case recSetBitmap:
		for index := range dst.data {
			dst.data[index] &= s.words[index]
		}
		dst.count = int64(countBitmap(dst.data))
	case recSetPositive:
		values := dst.listedValues()
		kept := 0
		for _, value := range values {
			if s.words[value>>5]&(uint32(1)<<(value&31)) != 0 {
				values[kept] = value
				kept++
			}
		}
		dst.data = dst.data[:kept]
		dst.used = uint32(kept)
		dst.count = int64(kept)
	case recSetRanges:
		ranges := dst.listedRanges()
		bitmapBytes := uint64((dst.universe + 31) / 32 * 4)
		if uint64(dst.count)*4 < bitmapBytes {
			values := make([]uint32, 0, dst.count)
			for pair := 0; pair < len(ranges); pair += 2 {
				values = appendBitmapValues(values, s.words, ranges[pair], ranges[pair]+ranges[pair+1])
			}
			dst.kind = recSetPositive
			dst.data = values
			dst.used = uint32(len(values))
			dst.count = int64(len(values))
			return
		}
		result := make([]uint32, len(s.words))
		for pair := 0; pair < len(ranges); pair += 2 {
			copyBitmapRange(result, s.words, ranges[pair], ranges[pair]+ranges[pair+1])
		}
		dst.kind = recSetBitmap
		dst.data = result
		dst.used = 0
		dst.count = int64(countBitmap(result))
	}
}

func bitmapWordInRange(words []uint32, wordIndex, begin, end uint32) uint32 {
	word := words[wordIndex]
	if wordIndex == begin>>5 {
		word &= ^uint32(0) << (begin & 31)
	}
	if tail := end & 31; tail != 0 && wordIndex == end>>5 {
		word &= uint32(1)<<tail - 1
	}
	return word
}

func appendBitmapValues(dst []uint32, words []uint32, begin, end uint32) []uint32 {
	for wordIndex, limit := begin>>5, (end+31)>>5; wordIndex < limit; wordIndex++ {
		word := bitmapWordInRange(words, wordIndex, begin, end)
		for word != 0 {
			dst = append(dst, wordIndex*32+uint32(bits.TrailingZeros32(word)))
			word &= word - 1
		}
	}
	return dst
}

func copyBitmapRange(dst, source []uint32, begin, end uint32) {
	for wordIndex, limit := begin>>5, (end+31)>>5; wordIndex < limit; wordIndex++ {
		dst[wordIndex] |= bitmapWordInRange(source, wordIndex, begin, end)
	}
}

func newCompressedStorageInt(count, minimum, maximum uint32) StorageInt {
	var result StorageInt
	result.initValuesUInt32(count, minimum, maximum)
	return result
}

func compressedStorageIntBytes(count, minimum, maximum uint32) uint32 {
	if count == 0 {
		return 0
	}
	width := uint8(bits.Len32(maximum - minimum))
	if width == 0 {
		width = 1
	}
	return uint32((((uint64(count)-1)*uint64(width)+65)/64 + 1) * 8)
}

func setCompressedStorageInt(storage *StorageInt, index, value uint32) {
	storage.buildValueUInt32(index, value)
}

func compressedStorageIntFromValues(values []uint32) StorageInt {
	if len(values) == 0 {
		return StorageInt{}
	}
	result := newCompressedStorageInt(uint32(len(values)), values[0], values[len(values)-1])
	for index, value := range values {
		setCompressedStorageInt(&result, uint32(index), value)
	}
	return result
}

type compressedPositive struct {
	universe uint32
	count    uint32
	values   StorageInt
}

func (s *compressedPositive) AndMut(dst *recSetShard) {
	if !compressedRecSetCanNarrow(dst, s.universe) {
		return
	}
	if recSetShardIsFull(dst) {
		values := make([]uint32, s.count)
		s.values.GetValuesUInt32Range(0, s.count, values, 1)
		dst.kind = recSetPositive
		dst.data = values
		dst.used = uint32(len(values))
		dst.count = int64(len(values))
		return
	}
	if dst.kind == recSetRanges {
		maximumMatches := min(uint64(s.count), uint64(dst.count))
		bitmapBytes := uint64((dst.universe + 31) / 32 * 4)
		if maximumMatches*4 < bitmapBytes {
			ranges := dst.listedRanges()
			result := make([]uint32, 0, maximumMatches)
			rangePosition := 0
			var scratch [256]uint32
			for sourceBase := uint32(0); sourceBase < s.count && rangePosition < len(ranges); {
				batchCount := min(uint32(len(scratch)), s.count-sourceBase)
				s.values.GetValuesUInt32Range(sourceBase, batchCount, scratch[:], 1)
				for _, candidate := range scratch[:batchCount] {
					for rangePosition < len(ranges) && ranges[rangePosition]+ranges[rangePosition+1] <= candidate {
						rangePosition += 2
					}
					if rangePosition == len(ranges) {
						break
					}
					if candidate >= ranges[rangePosition] {
						result = append(result, candidate)
					}
				}
				sourceBase += batchCount
			}
			dst.kind = recSetPositive
			dst.data = result
			dst.used = uint32(len(result))
			dst.count = int64(len(result))
			return
		}
		ranges := dst.listedRanges()
		result := make([]uint32, (dst.universe+31)/32)
		rangePosition := 0
		var scratch [256]uint32
		for sourceBase := uint32(0); sourceBase < s.count && rangePosition < len(ranges); {
			batchCount := min(uint32(len(scratch)), s.count-sourceBase)
			s.values.GetValuesUInt32Range(sourceBase, batchCount, scratch[:], 1)
			for _, candidate := range scratch[:batchCount] {
				for rangePosition < len(ranges) && ranges[rangePosition]+ranges[rangePosition+1] <= candidate {
					rangePosition += 2
				}
				if rangePosition == len(ranges) {
					break
				}
				if candidate >= ranges[rangePosition] {
					result[candidate>>5] |= uint32(1) << (candidate & 31)
				}
			}
			sourceBase += batchCount
		}
		dst.kind = recSetBitmap
		dst.data = result
		dst.used = 0
		dst.count = int64(countBitmap(result))
		return
	}
	if dst.kind == recSetBitmap {
		var scratch [256]uint32
		cursor := uint32(0)
		for base := uint32(0); base < s.count; {
			batchCount := min(uint32(len(scratch)), s.count-base)
			s.values.GetValuesUInt32Range(base, batchCount, scratch[:], 1)
			for _, candidate := range scratch[:batchCount] {
				clearRecSetBitmapRange(dst.data, cursor, candidate)
				cursor = candidate + 1
			}
			base += batchCount
		}
		clearRecSetBitmapRange(dst.data, cursor, dst.universe)
		dst.count = int64(countBitmap(dst.data))
		return
	}

	values := dst.listedValues()
	left, kept := 0, 0
	var scratch [256]uint32
	for base := uint32(0); base < s.count && left < len(values); {
		batchCount := min(uint32(len(scratch)), s.count-base)
		s.values.GetValuesUInt32Range(base, batchCount, scratch[:], 1)
		for _, candidate := range scratch[:batchCount] {
			for left < len(values) && values[left] < candidate {
				left++
			}
			if left < len(values) && values[left] == candidate {
				values[kept] = candidate
				kept++
				left++
			}
		}
		base += batchCount
	}
	dst.data = dst.data[:kept]
	dst.used = uint32(kept)
	dst.count = int64(kept)
}

type compressedRanges struct {
	universe uint32
	count    uint32
	ranges   uint32
	bases    StorageInt
	lengths  StorageInt
}

func (s *compressedRanges) AndMut(dst *recSetShard) {
	if !compressedRecSetCanNarrow(dst, s.universe) {
		return
	}
	if recSetShardIsFull(dst) {
		ranges := make([]uint32, s.ranges*2)
		s.bases.GetValuesUInt32Range(0, s.ranges, ranges, 2)
		s.lengths.GetValuesUInt32Range(0, s.ranges, ranges[1:], 2)
		dst.kind = recSetRanges
		dst.data = ranges
		dst.used = s.ranges
		dst.count = int64(s.count)
		return
	}
	if dst.kind == recSetPositive {
		values := dst.listedValues()
		left, kept := 0, 0
		var scratch [256]uint32
		for rangeBase := uint32(0); rangeBase < s.ranges && left < len(values); {
			batchCount := min(uint32(len(scratch)/2), s.ranges-rangeBase)
			s.bases.GetValuesUInt32Range(rangeBase, batchCount, scratch[:], 2)
			s.lengths.GetValuesUInt32Range(rangeBase, batchCount, scratch[1:], 2)
			for pair := uint32(0); pair < batchCount; pair++ {
				base := scratch[pair*2]
				end := base + scratch[pair*2+1]
				for left < len(values) && values[left] < base {
					left++
				}
				for left < len(values) && values[left] < end {
					values[kept] = values[left]
					kept++
					left++
				}
			}
			rangeBase += batchCount
		}
		dst.data = dst.data[:kept]
		dst.used = uint32(kept)
		dst.count = int64(kept)
		return
	}
	if dst.kind == recSetRanges {
		maximumRangeBytes := uint64(dst.used+s.ranges) * 2 * 4
		bitmapBytes := uint64((dst.universe + 31) / 32 * 4)
		if maximumRangeBytes < bitmapBytes {
			rangeCount, _ := intersectCompressedRanges(dst.listedRanges(), s, nil, nil)
			result := make([]uint32, rangeCount*2)
			_, rowCount := intersectCompressedRanges(dst.listedRanges(), s, result, nil)
			dst.data = result
			dst.used = rangeCount
			dst.count = rowCount
			return
		}
		result := make([]uint32, (dst.universe+31)/32)
		_, rowCount := intersectCompressedRanges(dst.listedRanges(), s, nil, result)
		dst.kind = recSetBitmap
		dst.data = result
		dst.used = 0
		dst.count = rowCount
		return
	}
	var scratch [256]uint32
	cursor := uint32(0)
	for rangeBase := uint32(0); rangeBase < s.ranges; {
		batchCount := min(uint32(len(scratch)/2), s.ranges-rangeBase)
		s.bases.GetValuesUInt32Range(rangeBase, batchCount, scratch[:], 2)
		s.lengths.GetValuesUInt32Range(rangeBase, batchCount, scratch[1:], 2)
		for pair := uint32(0); pair < batchCount; pair++ {
			base := scratch[pair*2]
			clearRecSetBitmapRange(dst.data, cursor, base)
			cursor = base + scratch[pair*2+1]
			if cursor >= dst.universe {
				cursor = dst.universe
				break
			}
		}
		rangeBase += batchCount
	}
	clearRecSetBitmapRange(dst.data, cursor, dst.universe)
	dst.count = int64(countBitmap(dst.data))
}

func intersectCompressedRanges(left []uint32, right *compressedRanges, output, bitmap []uint32) (uint32, int64) {
	leftPosition := 0
	var outputRanges uint32
	var outputRows int64
	var lastBase, lastEnd uint32
	var scratch [256]uint32
	for rightBase := uint32(0); rightBase < right.ranges && leftPosition < len(left); {
		batchCount := min(uint32(len(scratch)/2), right.ranges-rightBase)
		right.bases.GetValuesUInt32Range(rightBase, batchCount, scratch[:], 2)
		right.lengths.GetValuesUInt32Range(rightBase, batchCount, scratch[1:], 2)
		for pair := uint32(0); pair < batchCount; pair++ {
			base := scratch[pair*2]
			end := base + scratch[pair*2+1]
			for leftPosition < len(left) && left[leftPosition]+left[leftPosition+1] <= base {
				leftPosition += 2
			}
			position := leftPosition
			for position < len(left) && left[position] < end {
				intersectionBase := max(left[position], base)
				intersectionEnd := min(left[position]+left[position+1], end)
				if intersectionBase < intersectionEnd {
					if bitmap != nil {
						setBitmapRange(bitmap, intersectionBase, intersectionEnd)
					}
					if outputRanges > 0 && intersectionBase <= lastEnd {
						if intersectionEnd > lastEnd {
							outputRows += int64(intersectionEnd - lastEnd)
							lastEnd = intersectionEnd
							if output != nil {
								output[(outputRanges-1)*2+1] = lastEnd - lastBase
							}
						}
					} else {
						lastBase, lastEnd = intersectionBase, intersectionEnd
						outputRows += int64(intersectionEnd - intersectionBase)
						if output != nil {
							output[outputRanges*2] = intersectionBase
							output[outputRanges*2+1] = intersectionEnd - intersectionBase
						}
						outputRanges++
					}
				}
				if position < len(left) && left[position]+left[position+1] <= end {
					position += 2
				} else {
					break
				}
			}
			leftPosition = position
		}
		rightBase += batchCount
	}
	return outputRanges, outputRows
}

func setBitmapRange(words []uint32, begin, end uint32) {
	for wordIndex, limit := begin>>5, (end+31)>>5; wordIndex < limit; wordIndex++ {
		mask := ^uint32(0)
		if wordIndex == begin>>5 {
			mask &= ^uint32(0) << (begin & 31)
		}
		if tail := end & 31; tail != 0 && wordIndex == end>>5 {
			mask &= uint32(1)<<tail - 1
		}
		words[wordIndex] |= mask
	}
}

func clearRecSetBitmapRange(words []uint32, begin, end uint32) {
	for begin < end {
		wordIndex := begin >> 5
		wordEnd := min(end, (wordIndex+1)<<5)
		from, count := begin&31, wordEnd-begin
		mask := ^uint32(0) << from
		if from+count < 32 {
			mask &= uint32(1)<<(from+count) - 1
		}
		words[wordIndex] &^= mask
		begin = wordEnd
	}
}

func recSetShardIsFull(dst *recSetShard) bool {
	if dst == nil || dst.kind != recSetRanges || dst.count != int64(dst.universe) {
		return false
	}
	return (dst.used == 0 && len(dst.data) == 0) ||
		(dst.used == 1 && len(dst.data) >= 2 && dst.data[0] == 0 && dst.data[1] == dst.universe)
}

func compressedRecSetCanNarrow(dst *recSetShard, universe uint32) bool {
	if dst == nil || dst.count == 0 {
		return false
	}
	if dst.universe != universe {
		panic("compressed recset mutable intersection: universe mismatch")
	}
	return true
}

func countBitmap(words []uint32) uint32 {
	var count uint32
	for _, word := range words {
		count += uint32(bits.OnesCount32(word))
	}
	return count
}

type bitmapRecSetStats struct {
	count                        uint32
	runs                         uint32
	minimum, maximum             uint32
	baseMinimum, baseMaximum     uint32
	lengthMinimum, lengthMaximum uint32
}

func analyzeRecSetBitmap(words []uint32, universe uint32) bitmapRecSetStats {
	var result bitmapRecSetStats
	var runBase, runEnd uint32
	haveRun := false
	finishRun := func() {
		length := runEnd - runBase
		if result.runs == 0 {
			result.baseMinimum, result.baseMaximum = runBase, runBase
			result.lengthMinimum, result.lengthMaximum = length, length
		} else {
			result.baseMinimum = min(result.baseMinimum, runBase)
			result.baseMaximum = max(result.baseMaximum, runBase)
			result.lengthMinimum = min(result.lengthMinimum, length)
			result.lengthMaximum = max(result.lengthMaximum, length)
		}
		result.runs++
	}
	for wordIndex, source := range words {
		for source != 0 {
			position := uint32(wordIndex*32 + bits.TrailingZeros32(source))
			if position >= universe {
				break
			}
			if result.count == 0 {
				result.minimum = position
			}
			result.maximum = position
			result.count++
			if haveRun && position == runEnd {
				runEnd++
			} else {
				if haveRun {
					finishRun()
				}
				runBase, runEnd, haveRun = position, position+1, true
			}
			source &= source - 1
		}
	}
	if haveRun {
		finishRun()
	}
	return result
}

func packBitmapIDs(words []uint32, universe uint32, stats bitmapRecSetStats) StorageInt {
	result := newCompressedStorageInt(stats.count, stats.minimum, stats.maximum)
	index := uint32(0)
	for wordIndex, source := range words {
		for source != 0 {
			position := uint32(wordIndex*32 + bits.TrailingZeros32(source))
			if position >= universe {
				break
			}
			setCompressedStorageInt(&result, index, position)
			index++
			source &= source - 1
		}
	}
	return result
}

func packBitmapRanges(words []uint32, universe uint32, stats bitmapRecSetStats) (StorageInt, StorageInt) {
	bases := newCompressedStorageInt(stats.runs, stats.baseMinimum, stats.baseMaximum)
	lengths := newCompressedStorageInt(stats.runs, stats.lengthMinimum, stats.lengthMaximum)
	var runBase, runEnd uint32
	haveRun, index := false, uint32(0)
	finishRun := func() {
		setCompressedStorageInt(&bases, index, runBase)
		setCompressedStorageInt(&lengths, index, runEnd-runBase)
		index++
	}
	for wordIndex, source := range words {
		for source != 0 {
			position := uint32(wordIndex*32 + bits.TrailingZeros32(source))
			if position >= universe {
				break
			}
			if haveRun && position == runEnd {
				runEnd++
			} else {
				if haveRun {
					finishRun()
				}
				runBase, runEnd, haveRun = position, position+1, true
			}
			source &= source - 1
		}
	}
	if haveRun {
		finishRun()
	}
	return bases, lengths
}

func analyzeRecSetIDs(ids []uint32) bitmapRecSetStats {
	if len(ids) == 0 {
		return bitmapRecSetStats{}
	}
	result := bitmapRecSetStats{
		count:   uint32(len(ids)),
		minimum: ids[0],
		maximum: ids[len(ids)-1],
	}
	runBase, runEnd := ids[0], ids[0]+1
	finishRun := func() {
		length := runEnd - runBase
		if result.runs == 0 {
			result.baseMinimum, result.baseMaximum = runBase, runBase
			result.lengthMinimum, result.lengthMaximum = length, length
		} else {
			result.baseMaximum = runBase
			result.lengthMinimum = min(result.lengthMinimum, length)
			result.lengthMaximum = max(result.lengthMaximum, length)
		}
		result.runs++
	}
	for _, position := range ids[1:] {
		if position == runEnd {
			runEnd++
		} else {
			finishRun()
			runBase, runEnd = position, position+1
		}
	}
	finishRun()
	return result
}

func packIDRanges(ids []uint32, stats bitmapRecSetStats) (StorageInt, StorageInt) {
	bases := newCompressedStorageInt(stats.runs, stats.baseMinimum, stats.baseMaximum)
	lengths := newCompressedStorageInt(stats.runs, stats.lengthMinimum, stats.lengthMaximum)
	if len(ids) == 0 {
		return bases, lengths
	}
	runBase, runEnd, index := ids[0], ids[0]+1, uint32(0)
	finishRun := func() {
		setCompressedStorageInt(&bases, index, runBase)
		setCompressedStorageInt(&lengths, index, runEnd-runBase)
		index++
	}
	for _, position := range ids[1:] {
		if position == runEnd {
			runEnd++
		} else {
			finishRun()
			runBase, runEnd = position, position+1
		}
	}
	finishRun()
	return bases, lengths
}

func compressRecSetIDs(ids []uint32, universe uint32) compressedRecSet {
	stats := analyzeRecSetIDs(ids)
	positiveSize := uint32(unsafe.Sizeof(compressedPositive{})) + compressedStorageIntBytes(stats.count, stats.minimum, stats.maximum)
	rangesSize := uint32(unsafe.Sizeof(compressedRanges{})) +
		compressedStorageIntBytes(stats.runs, stats.baseMinimum, stats.baseMaximum) +
		compressedStorageIntBytes(stats.runs, stats.lengthMinimum, stats.lengthMaximum)
	if rangesSize < positiveSize {
		bases, lengths := packIDRanges(ids, stats)
		return compressedRecSet{
			set:   &compressedRanges{universe: universe, count: stats.count, ranges: stats.runs, bases: bases, lengths: lengths},
			count: stats.count,
			bytes: rangesSize,
		}
	}
	return compressedRecSet{
		set:   &compressedPositive{universe: universe, count: stats.count, values: compressedStorageIntFromValues(ids)},
		count: stats.count,
		bytes: positiveSize,
	}
}

func compressRecSetBitmap(words []uint32, universe uint32) compressedRecSet {
	stats := analyzeRecSetBitmap(words, universe)
	kind := uint8(0)
	bestSize := uint32(unsafe.Sizeof(compressedBitmap{})) + uint32(len(words))*4
	positiveSize := uint32(unsafe.Sizeof(compressedPositive{})) + compressedStorageIntBytes(stats.count, stats.minimum, stats.maximum)
	if positiveSize < bestSize {
		kind, bestSize = 1, positiveSize
	}
	rangesSize := uint32(unsafe.Sizeof(compressedRanges{})) +
		compressedStorageIntBytes(stats.runs, stats.baseMinimum, stats.baseMaximum) +
		compressedStorageIntBytes(stats.runs, stats.lengthMinimum, stats.lengthMaximum)
	if rangesSize < bestSize {
		kind, bestSize = 2, rangesSize
	}
	var set CompressedRecSet
	switch kind {
	case 0:
		ownedWords := make([]uint32, len(words))
		copy(ownedWords, words)
		set = &compressedBitmap{universe: universe, words: ownedWords}
	case 1:
		set = &compressedPositive{universe: universe, count: stats.count, values: packBitmapIDs(words, universe, stats)}
	case 2:
		bases, lengths := packBitmapRanges(words, universe, stats)
		set = &compressedRanges{universe: universe, count: stats.count, ranges: stats.runs, bases: bases, lengths: lengths}
	}
	return compressedRecSet{set: set, count: stats.count, bytes: bestSize}
}
