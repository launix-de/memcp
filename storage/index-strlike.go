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

import "sort"
import "unsafe"
import "strings"
import "unicode"

import "github.com/launix-de/memcp/scm"

type likeMatcher struct{}

const minimumBigramIndexRows uint32 = 256

type noopIndexHook struct{}

func (*noopIndexHook) Bind(scm.Scmer) IndexRowMatcher { return nil }
func (*noopIndexHook) ComputeSize() uint              { return 0 }

var sharedNoopIndexHook IndexHook = &noopIndexHook{}

func (m *likeMatcher) Kind() string      { return "like" }
func (m *likeMatcher) IsSorted() bool    { return false }
func (m *likeMatcher) IsPointLike() bool { return true }

func (m *likeMatcher) Analyze(ctx IndexAnalyzeContext, node scm.Scmer) (IndexBoundary, bool) {
	v, ok := scmerSlice(node)
	if !ok || len(v) < 3 || !ctx.FunctionIs(v[0], "strlike") {
		return IndexBoundary{}, false
	}
	col, ok := ctx.ResolveColumn(v[1])
	if !ok {
		return IndexBoundary{}, false
	}
	patternValue, ok := ctx.ExtractConstant(v[2])
	if !ok || !patternValue.IsString() {
		return IndexBoundary{}, false
	}
	collation := "utf8mb4_general_ci"
	if len(v) >= 4 {
		value, constant := ctx.ExtractConstant(v[3])
		if !constant || !value.IsString() {
			return IndexBoundary{}, false
		}
		collation = strings.ToLower(value.String())
	}
	pattern := patternValue.String()
	if wildcard := strings.IndexAny(pattern, "%_"); wildcard > 0 && !strings.Contains(collation, "_ci") {
		prefix := pattern[:wildcard]
		upper := []byte(prefix)
		upper[len(upper)-1]++
		return columnboundaries{
			col: col, matcher: RangeMatcher, lower: scm.NewString(prefix), upper: scm.NewString(string(upper)),
			lowerInclusive: true, upperInclusive: false,
		}, true
	}
	return NewIndexBoundary(col, m, patternValue, collation), true
}

func (m *likeMatcher) Deploy(ctx IndexDeployContext, persistent bool) IndexHook {
	if !persistent || ctx.MainCount == 0 || ctx.Column == nil {
		return nil
	}
	if ctx.MainCount < minimumBigramIndexRows {
		return sharedNoopIndexHook
	}
	return buildBigramIndex(ctx.MainCount, ctx.Column)
}

type bigramIndex struct {
	universe uint32
	grams    bigramTable
	bytes    uint64 // owned compressed posting storage
}

// bigramTable is immutable and sorted. A LIKE pattern performs only a handful
// of lookups, so compact binary search touches fewer cache lines than a sparse
// hash table while eliminating empty buckets and their occupancy bitmap.
type bigramTable struct {
	keys   []uint64
	sets   []CompressedRecSet
	counts []uint32
}

func (t *bigramTable) Len() int           { return len(t.keys) }
func (t *bigramTable) Less(i, j int) bool { return t.keys[i] < t.keys[j] }
func (t *bigramTable) Swap(i, j int) {
	t.keys[i], t.keys[j] = t.keys[j], t.keys[i]
	t.sets[i], t.sets[j] = t.sets[j], t.sets[i]
	t.counts[i], t.counts[j] = t.counts[j], t.counts[i]
}

func (t *bigramTable) append(key uint64, value compressedRecSet) {
	t.keys = append(t.keys, key)
	t.sets = append(t.sets, value.set)
	t.counts = append(t.counts, value.count)
}

func (t *bigramTable) find(key uint64) int {
	position := sort.Search(len(t.keys), func(position int) bool {
		return t.keys[position] >= key
	})
	if position < len(t.keys) && t.keys[position] == key {
		return position
	}
	return -1
}

func (t *bigramTable) ComputeSize() uint64 {
	return uint64(cap(t.keys))*uint64(unsafe.Sizeof(uint64(0))) +
		uint64(cap(t.sets))*uint64(unsafe.Sizeof(CompressedRecSet(nil))) +
		uint64(cap(t.counts))*uint64(unsafe.Sizeof(uint32(0)))
}

func (s *bigramIndex) ComputeSize() uint {
	return uint(unsafe.Sizeof(*s)) + uint(s.bytes+s.grams.ComputeSize())
}

func (s *bigramIndex) Bind(lower scm.Scmer) IndexRowMatcher {
	candidates, constrained := s.candidatesPattern(lower.String())
	if !constrained {
		return nil
	}
	candidateWords := candidates.data
	universe := s.universe
	return func(ids []uint32) []uint32 {
		out := ids[:0]
		for _, id := range ids {
			// The immutable bigram index covers main rows. Delta rows retain the
			// original STRLIKE residual and must pass this candidate filter.
			if id >= universe || (int(id>>5) < len(candidateWords) && candidateWords[id>>5]&(uint32(1)<<(id&31)) != 0) {
				out = append(out, id)
			}
		}
		return out
	}
}

func normalizedBigramKey(left, right rune) uint64 {
	return uint64(uint32(left))<<32 | uint64(uint32(right))
}

func normalizeBigramRune(value rune) rune {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	if value <= 127 {
		return value
	}
	return unicode.ToLower(value)
}

type bigramPostingBuilder struct {
	key      uint64
	data     []uint32
	last     uint32
	haveLast bool
	bitmap   bool
}

func (b *bigramPostingBuilder) add(recid, wordCount uint32) {
	if b.haveLast && b.last == recid {
		return
	}
	b.last, b.haveLast = recid, true
	if b.bitmap {
		b.data[recid>>5] |= uint32(1) << (recid & 31)
		return
	}
	b.data = append(b.data, recid)
	// A uint32 RecID list and a uint32 bitmap break even at one hit per
	// bitmap word. Promote once; dense postings then stop growing entirely.
	if uint32(len(b.data)) < wordCount {
		return
	}
	bitmap := make([]uint32, wordCount)
	for _, value := range b.data {
		bitmap[value>>5] |= uint32(1) << (value & 31)
	}
	b.data, b.bitmap = bitmap, true
}

func (b *bigramPostingBuilder) finish(universe uint32) compressedRecSet {
	if b.bitmap {
		return compressRecSetBitmap(b.data, universe)
	}
	wordCount := (universe + 31) / 32
	// At moderate density entropy coding can beat an absolute StorageInt list.
	// Materialize one temporary bitmap only during finalization; unlike #571's
	// builder, sparse postings never retain one concurrently with all others.
	if len(b.data) >= max(16, int(wordCount/8)) {
		bitmap := make([]uint32, wordCount)
		for _, value := range b.data {
			bitmap[value>>5] |= uint32(1) << (value & 31)
		}
		return compressRecSetBitmap(bitmap, universe)
	}
	return compressRecSetIDs(b.data, universe)
}

func buildBigramIndex(count uint32, reader ColumnReader) *bigramIndex {
	index := &bigramIndex{universe: count}
	const readBatch = 1024
	values := make([]scm.Scmer, readBatch)
	wordCount := (count + 31) / 32
	// PDF/text payloads are overwhelmingly byte-range text. Resolve those
	// bigrams through one temporary direct table; only genuinely wide Unicode
	// pairs pay for a hash lookup while the index is being built.
	asciiPosting := make([]uint32, 1<<16)
	unicodePostingByKey := make(map[uint64]uint32)
	postings := make([]bigramPostingBuilder, 0)
	for base := uint32(0); base < count; base += readBatch {
		batchCount := uint32(readBatch)
		if remaining := count - base; remaining < batchCount {
			batchCount = remaining
		}
		reader.GetValueRange(base, batchCount, values[:batchCount], 1)
		for offset := uint32(0); offset < batchCount; offset++ {
			value := values[offset]
			if value.IsString() {
				var previous rune
				havePrevious := false
				for _, current := range value.String() {
					current = normalizeBigramRune(current)
					if havePrevious {
						key := normalizedBigramKey(previous, current)
						var postingIndex uint32
						if previous <= 255 && current <= 255 {
							slot := &asciiPosting[uint16(previous)<<8|uint16(current)]
							if *slot == 0 {
								postingIndex = uint32(len(postings))
								postings = append(postings, bigramPostingBuilder{key: key})
								*slot = postingIndex + 1
							} else {
								postingIndex = *slot - 1
							}
						} else {
							var exists bool
							postingIndex, exists = unicodePostingByKey[key]
							if !exists {
								postingIndex = uint32(len(postings))
								unicodePostingByKey[key] = postingIndex
								postings = append(postings, bigramPostingBuilder{key: key})
							}
						}
						postings[postingIndex].add(base+offset, wordCount)
					}
					previous = current
					havePrevious = true
				}
			}
			values[offset] = scm.NewNil()
		}
	}
	// The lookup structures are build-only. Make them unreachable before
	// final posting compression, whose peak memory should contain only the
	// posting builders currently being replaced and their final encodings.
	asciiPosting = nil
	unicodePostingByKey = nil
	values = nil
	index.grams.keys = make([]uint64, 0, len(postings))
	index.grams.sets = make([]CompressedRecSet, 0, len(postings))
	index.grams.counts = make([]uint32, 0, len(postings))
	for postingIndex := range postings {
		compressed := postings[postingIndex].finish(count)
		index.bytes += uint64(compressed.bytes)
		index.grams.append(postings[postingIndex].key, compressed)
		postings[postingIndex].data = nil
	}
	sort.Sort(&index.grams)
	return index
}

// candidatesPattern fuses pattern decoding, case normalization and immutable
// index lookup. No intermediate []uint64 is produced in the query path.
func (s *bigramIndex) candidatesPattern(pattern string) (recSetShard, bool) {
	var setScratch [32]queryBigramPosting
	sets := setScratch[:0]
	var previous rune
	havePrevious := false
	for _, current := range pattern {
		if current == '%' || current == '_' {
			havePrevious = false
			continue
		}
		current = normalizeBigramRune(current)
		if havePrevious {
			position := s.grams.find(normalizedBigramKey(previous, current))
			if position < 0 {
				return recSetShard{kind: recSetRanges, universe: s.universe}, true
			}
			duplicate := false
			for _, existing := range sets {
				if existing.position == position {
					duplicate = true
					break
				}
			}
			if !duplicate {
				sets = append(sets, queryBigramPosting{
					set: s.grams.sets[position], count: s.grams.counts[position], position: position,
				})
			}
		}
		previous = current
		havePrevious = true
	}
	if len(sets) == 0 {
		return recSetShard{}, false
	}
	return s.intersectCandidates(sets), true
}

type queryBigramPosting struct {
	set      CompressedRecSet
	count    uint32
	position int
}

func (s *bigramIndex) intersectCandidates(sets []queryBigramPosting) recSetShard {
	// Patterns normally contain only a few distinct bigrams. Insertion sort
	// avoids sort.Interface/closure escape overhead and is optimal here.
	for position := 1; position < len(sets); position++ {
		value := sets[position]
		previous := position - 1
		for previous >= 0 && sets[previous].count > value.count {
			sets[previous+1] = sets[previous]
			previous--
		}
		sets[previous+1] = value
	}
	// Persistent postings optimize resident size; this query-local result
	// optimizes repeated intersections and membership probes. Materialize one
	// disposable full bitmap and let every compressed source clear it in place.
	words := make([]uint32, (s.universe+31)/32)
	for index := range words {
		words[index] = ^uint32(0)
	}
	if tail := s.universe & 31; tail != 0 && len(words) > 0 {
		words[len(words)-1] = uint32(1)<<tail - 1
	}
	result := recSetShard{kind: recSetBitmap, universe: s.universe, data: words, count: int64(s.universe)}
	for _, set := range sets {
		set.set.AndMut(&result)
		if result.count == 0 {
			break
		}
	}
	return result
}
