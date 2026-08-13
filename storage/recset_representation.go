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
import "sort"

type recSetRepresentation uint8

const (
	recSetEmpty recSetRepresentation = iota
	recSetFull
	recSetPositive
	recSetNegative
	recSetBitmap
)

// recSetShardBuilder allocates exactly the eventual bitmap footprint. Until
// deviations from the first result fill that storage, the same words hold a
// positive or negative recid list. Overflow converts the words in place.
type recSetShardBuilder struct {
	shard       *storageShard
	universe    uint32
	data        []uint32
	deviations  uint32
	matched     uint32
	defaultHit  bool
	initialized bool
	bitmap      bool
}

func newRecSetShardBuilder(shard *storageShard, universe uint32, allowFull bool) *recSetShardBuilder {
	builder := &recSetShardBuilder{
		shard:    shard,
		universe: universe,
		data:     make([]uint32, (universe+31)/32),
	}
	if !allowFull {
		builder.initialized = true
	}
	return builder
}

func (b *recSetShardBuilder) add(recid uint32, hit bool) {
	if !b.initialized {
		b.initialized = true
		b.defaultHit = hit
	}
	if hit {
		b.matched++
	}
	if b.bitmap {
		b.addBitmap(recid, hit)
		return
	}
	if hit == b.defaultHit {
		return
	}
	if b.deviations < uint32(len(b.data)) {
		b.data[b.deviations] = recid
		b.deviations++
		return
	}
	clear(b.data)
	b.bitmap = true
	b.addBitmap(recid, hit)
}

func (b *recSetShardBuilder) addBitmap(recid uint32, hit bool) {
	if hit {
		b.data[recid>>5] |= uint32(1) << (recid & 31)
	}
}

func (b *recSetShardBuilder) finish() recSetShard {
	part := recSetShard{shard: b.shard, universe: b.universe, count: int64(b.matched)}
	if b.matched == 0 {
		part.kind = recSetEmpty
		return part
	}
	if b.matched == b.universe {
		part.kind = recSetFull
		return part
	}
	part.data = b.data
	if !b.bitmap {
		sort.Slice(part.data[:b.deviations], func(i, j int) bool {
			return part.data[i] < part.data[j]
		})
		part.used = b.deviations
		if b.defaultHit {
			part.kind = recSetNegative
		} else {
			part.kind = recSetPositive
		}
		return part
	}
	part.kind = recSetBitmap
	return part
}

func newRecSetShardFromSortedIDs(shard *storageShard, universe uint32, ids []uint32) recSetShard {
	part := recSetShard{shard: shard, universe: universe, count: int64(len(ids))}
	if len(ids) == 0 {
		part.kind = recSetEmpty
		return part
	}
	if len(ids) == int(universe) {
		part.kind = recSetFull
		return part
	}
	part.data = make([]uint32, (universe+31)/32)
	if len(ids) <= len(part.data) {
		part.kind = recSetPositive
		part.used = uint32(len(ids))
		copy(part.data, ids)
		return part
	}
	missing := int(universe) - len(ids)
	if missing <= len(part.data) {
		part.kind = recSetNegative
		part.used = uint32(missing)
		write, present := 0, 0
		for id := uint32(0); id < universe; id++ {
			if present < len(ids) && ids[present] == id {
				present++
				continue
			}
			part.data[write] = id
			write++
		}
		return part
	}
	part.kind = recSetBitmap
	for _, id := range ids {
		part.data[id>>5] |= uint32(1) << (id & 31)
	}
	return part
}

func sortedUint32Contains(values []uint32, value uint32) bool {
	pos := sort.Search(len(values), func(i int) bool { return values[i] >= value })
	return pos < len(values) && values[pos] == value
}

func (s *recSetShard) listedValues() []uint32 {
	return s.data[:s.used]
}

func unionRecSetShards(shard *storageShard, parts []*recSetShard) recSetShard {
	var universe uint32
	for _, part := range parts {
		if part != nil && part.universe > universe {
			universe = part.universe
		}
	}
	compact := parts[:0]
	hasBitmap := false
	mixedUniverse := false
	for _, part := range parts {
		if part == nil || part.kind == recSetEmpty {
			continue
		}
		if part.kind == recSetFull && part.universe == universe {
			return recSetShard{shard: shard, kind: recSetFull, universe: universe, count: int64(universe)}
		}
		hasBitmap = hasBitmap || part.kind == recSetBitmap
		mixedUniverse = mixedUniverse || part.universe != universe
		compact = append(compact, part)
	}
	parts = compact
	if len(parts) == 0 {
		return recSetShard{shard: shard, kind: recSetEmpty, universe: universe}
	}
	data := make([]uint32, (universe+31)/32)
	if mixedUniverse {
		return combineRecSetBitmapsWithData(shard, universe, parts, true, data)
	}
	if candidate := shortestRecSetList(parts, recSetNegative); candidate != nil {
		used := filterRecSetCandidates(data, candidate.listedValues(), parts, false)
		return recSetShardFromList(shard, universe, data, used, recSetNegative)
	}
	if hasBitmap {
		return combineRecSetBitmapsWithData(shard, universe, parts, true, data)
	}
	positiveLists := recSetLists(parts, recSetPositive)
	used, overflow := unionSortedLists(data, positiveLists)
	if overflow {
		return combineRecSetBitmapsWithData(shard, universe, parts, true, data)
	}
	return recSetShardFromList(shard, universe, data, used, recSetPositive)
}

func intersectRecSetShards(shard *storageShard, parts []*recSetShard) recSetShard {
	if len(parts) == 0 {
		return recSetShard{shard: shard, kind: recSetEmpty}
	}
	var universe uint32
	for _, part := range parts {
		if part == nil || part.kind == recSetEmpty {
			return recSetShard{shard: shard, kind: recSetEmpty, universe: universe}
		}
		if part.universe > universe {
			universe = part.universe
		}
	}
	compact := parts[:0]
	hasBitmap := false
	mixedUniverse := false
	for _, part := range parts {
		if part.kind == recSetFull && part.universe == universe {
			continue
		}
		hasBitmap = hasBitmap || part.kind == recSetBitmap
		mixedUniverse = mixedUniverse || part.universe != universe
		compact = append(compact, part)
	}
	parts = compact
	if len(parts) == 0 {
		return recSetShard{shard: shard, kind: recSetFull, universe: universe, count: int64(universe)}
	}
	data := make([]uint32, (universe+31)/32)
	if mixedUniverse {
		return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
	}
	if candidate := shortestRecSetList(parts, recSetPositive); candidate != nil {
		used := filterRecSetCandidates(data, candidate.listedValues(), parts, true)
		return recSetShardFromList(shard, universe, data, used, recSetPositive)
	}
	if hasBitmap {
		return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
	}
	negativeLists := recSetLists(parts, recSetNegative)
	used, overflow := unionSortedLists(data, negativeLists)
	if overflow {
		return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
	}
	return recSetShardFromList(shard, universe, data, used, recSetNegative)
}

func recSetLists(parts []*recSetShard, kind recSetRepresentation) [][]uint32 {
	lists := make([][]uint32, 0, len(parts))
	for _, part := range parts {
		if part.kind == kind {
			lists = append(lists, part.listedValues())
		}
	}
	return lists
}

func shortestRecSetList(parts []*recSetShard, kind recSetRepresentation) *recSetShard {
	var shortest *recSetShard
	for _, part := range parts {
		if part.kind == kind && (shortest == nil || part.used < shortest.used) {
			shortest = part
		}
	}
	return shortest
}

// filterRecSetCandidates handles the sparse-result quadrants of the algebra
// matrix. Intersections with a positive list can only contain its IDs; unions
// with a negative list can only exclude its IDs.
func filterRecSetCandidates(dst, candidates []uint32, parts []*recSetShard, wantPresent bool) uint32 {
	used := 0
	for _, id := range candidates {
		matches := true
		for _, part := range parts {
			if part.contains(id) != wantPresent {
				matches = false
				break
			}
		}
		if matches {
			dst[used] = id
			used++
		}
	}
	return uint32(used)
}

// unionSortedLists merges sorted recid lists directly into the final backing.
// False means the list representation overflowed and the caller must reuse the
// same storage as a bitmap.
func unionSortedLists(dst []uint32, lists [][]uint32) (uint32, bool) {
	positions := make([]int, len(lists))
	used := 0
	for {
		var next uint32
		found := false
		for i, values := range lists {
			if positions[i] < len(values) && (!found || values[positions[i]] < next) {
				next = values[positions[i]]
				found = true
			}
		}
		if !found {
			return uint32(used), false
		}
		if used == len(dst) {
			return uint32(used), true
		}
		dst[used] = next
		used++
		for i, values := range lists {
			for positions[i] < len(values) && values[positions[i]] == next {
				positions[i]++
			}
		}
	}
}

type recSetWordCursor struct {
	part *recSetShard
	pos  int
}

func (c *recSetWordCursor) word(wordIndex uint32) uint32 {
	wordStart := wordIndex << 5
	if wordStart >= c.part.universe {
		return 0
	}
	tailMask := ^uint32(0)
	if remaining := c.part.universe - wordStart; remaining < 32 {
		tailMask = (uint32(1) << remaining) - 1
	}
	switch c.part.kind {
	case recSetFull:
		return tailMask
	case recSetBitmap:
		if int(wordIndex) < len(c.part.data) {
			return c.part.data[wordIndex] & tailMask
		}
		return 0
	case recSetPositive, recSetNegative:
		values := c.part.listedValues()
		for c.pos < len(values) && values[c.pos]>>5 < wordIndex {
			c.pos++
		}
		mask := uint32(0)
		for c.pos < len(values) && values[c.pos]>>5 == wordIndex {
			mask |= uint32(1) << (values[c.pos] & 31)
			c.pos++
		}
		if c.part.kind == recSetNegative {
			return ^mask & tailMask
		}
		return mask
	default:
		return 0
	}
}

func combineRecSetBitmapsWithData(shard *storageShard, universe uint32, parts []*recSetShard, union bool, data []uint32) recSetShard {
	clear(data)
	cursors := make([]recSetWordCursor, len(parts))
	for i, part := range parts {
		cursors[i].part = part
	}
	for wordIndex := range data {
		value := uint32(0)
		if !union {
			value = ^uint32(0)
		}
		for i := range cursors {
			if union {
				value |= cursors[i].word(uint32(wordIndex))
			} else {
				value &= cursors[i].word(uint32(wordIndex))
			}
		}
		data[wordIndex] = value
	}
	return recSetShardFromBitmap(shard, universe, data)
}

func recSetShardFromList(shard *storageShard, universe uint32, data []uint32, used uint32, kind recSetRepresentation) recSetShard {
	count := used
	if kind == recSetNegative {
		count = universe - used
	}
	part := recSetShard{shard: shard, kind: kind, universe: universe, data: data, used: used, count: int64(count)}
	if count == 0 {
		part.kind = recSetEmpty
		part.data = nil
	} else if count == universe {
		part.kind = recSetFull
		part.data = nil
	}
	return part
}

func recSetShardFromBitmap(shard *storageShard, universe uint32, data []uint32) recSetShard {
	if tail := universe & 31; tail != 0 && len(data) > 0 {
		data[len(data)-1] &= (uint32(1) << tail) - 1
	}
	count := uint32(0)
	for _, word := range data {
		count += uint32(bits.OnesCount32(word))
	}
	part := recSetShard{shard: shard, universe: universe, data: data, count: int64(count)}
	if count == 0 {
		part.kind = recSetEmpty
	} else if count == universe {
		part.kind = recSetFull
	} else {
		part.kind = recSetBitmap
	}
	return part
}

func (s *recSetShard) forEachID(callback func(uint32) bool) {
	switch s.kind {
	case recSetEmpty:
		return
	case recSetPositive:
		for _, id := range s.listedValues() {
			if !callback(id) {
				return
			}
		}
	case recSetFull:
		for id := uint32(0); id < s.universe; id++ {
			if !callback(id) {
				return
			}
		}
	case recSetNegative:
		excluded := s.listedValues()
		pos := 0
		for id := uint32(0); id < s.universe; id++ {
			if pos < len(excluded) && excluded[pos] == id {
				pos++
				continue
			}
			if !callback(id) {
				return
			}
		}
	case recSetBitmap:
		for wordIndex, word := range s.data {
			for word != 0 {
				bit := bits.TrailingZeros32(word)
				id := uint32(wordIndex*32 + bit)
				if id < s.universe && !callback(id) {
					return
				}
				word &= word - 1
			}
		}
	}
}
