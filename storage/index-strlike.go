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
import "weak"
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
	if bound, found := analyzeTextSearch(ctx, node, m); found {
		return bound, true
	}
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
	escape := "\\"
	if len(v) >= 5 {
		value, constant := ctx.ExtractConstant(v[4])
		if !constant || !value.IsString() {
			return IndexBoundary{}, false
		}
		escape = value.String()
	}
	pattern := patternValue.String()
	tokens := scm.DecodeLikePattern(pattern, escape)
	var prefix strings.Builder
	wildcard := false
	for _, token := range tokens {
		if token.Kind != scm.LikeLiteral {
			wildcard = true
			break
		}
		prefix.WriteRune(token.Rune)
	}
	if wildcard && prefix.Len() > 0 && (strings.HasSuffix(collation, "_bin") || collation == "bin" || collation == "binary") {
		upper := []byte(prefix.String())
		for i := len(upper) - 1; i >= 0; i-- {
			if upper[i] != 255 {
				upper[i]++
				upper = upper[:i+1]
				return analyzedBoundary{col: col, matcher: RangeMatcher, lower: scm.NewString(prefix.String()), upper: scm.NewString(string(upper)), lowerInclusive: true, collation: collation}, true
			}
		}
	}
	binding := patternValue
	if escape != "\\" {
		binding = scm.NewSlice([]scm.Scmer{scm.NewString(":like"), patternValue, scm.NewString(escape)})
	}
	return NewIndexBoundary(col, m, binding, collation), true
}

func (m *likeMatcher) Deploy(ctx IndexDeployContext, persistent bool) IndexHook {
	if !persistent || ctx.MainCount == 0 {
		return nil
	}
	if ctx.MainCount < minimumBigramIndexRows {
		return sharedNoopIndexHook
	}
	if len(ctx.readers) == 0 {
		if ctx.Column == nil {
			return nil
		}
		return buildBigramIndex(ctx.MainCount, ctx.Column)
	}
	indexes := make([]*bigramIndex, len(ctx.readers))
	for i, reader := range ctx.readers {
		source := ctx.sources[i]
		if source == nil {
			return nil
		}
		// Computed definitions can change within the same main generation.
		// Their index revision owns invalidation; do not reuse its old postings.
		if _, computed := source.(*StorageComputeProxy); computed {
			indexes[i] = buildBigramIndex(ctx.MainCount, reader)
			continue
		}
		identity := source.(interface {
			bigramCacheIdentity() weak.Pointer[storageJITFunctions]
		}).bigramCacheIdentity()
		if cached, ok := ctx.shard.bigramIndexes.Load(identity); ok {
			indexes[i] = cached.(weak.Pointer[bigramIndex]).Value()
		}
		if indexes[i] != nil {
			continue
		}
		// Retired/reloaded source columns must not accumulate weak registry keys.
		ctx.shard.bigramIndexes.Range(func(key, _ any) bool {
			if key.(weak.Pointer[storageJITFunctions]).Value() == nil {
				ctx.shard.bigramIndexes.Delete(key)
			}
			return true
		})
		built := buildBigramIndex(ctx.MainCount, reader)
		for {
			old, loaded := ctx.shard.bigramIndexes.LoadOrStore(identity, weak.Make(built))
			if !loaded {
				indexes[i] = built
				break
			}
			if existing := old.(weak.Pointer[bigramIndex]).Value(); existing != nil {
				indexes[i] = existing
				break
			}
			if ctx.shard.bigramIndexes.CompareAndSwap(identity, old, weak.Make(built)) {
				indexes[i] = built
				break
			}
		}
	}
	if len(indexes) == 1 {
		return indexes[0]
	}
	return &textSearchIndex{indexes: indexes}
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
	var candidates recSetShard
	var constrained bool
	if terms, search := textSearchBinding(lower); search {
		if terms.IsNil() {
			return nil
		}
		candidates, constrained = textSearchCandidates([]*bigramIndex{s}, terms)
	} else {
		if !validLikeBinding(lower) {
			return nil
		}
		candidates, constrained = s.candidatesPattern(likeBinding(lower))
	}
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

func (s *bigramIndex) EstimateCandidates(lower scm.Scmer) (uint32, uint32, bool) {
	if _, search := textSearchBinding(lower); search {
		return s.universe, s.universe, false
	}
	if !validLikeBinding(lower) {
		return s.universe, s.universe, false
	}
	pattern, escape := likeBinding(lower)
	minimum := s.universe
	constrained := false
	for _, run := range scm.LikeLiteralRuns(scm.DecodeLikePattern(pattern, escape)) {
		var previous rune
		havePrevious := false
		for _, current := range run {
			current = normalizeBigramRune(current)
			if havePrevious {
				position := s.grams.find(normalizedBigramKey(previous, current))
				if position < 0 {
					return 0, s.universe, true
				}
				constrained = true
				if count := s.grams.counts[position]; count < minimum {
					minimum = count
				}
			}
			previous = current
			havePrevious = true
		}
	}
	return minimum, s.universe, constrained
}

func likeBinding(value scm.Scmer) (string, string) {
	if value.IsSlice() {
		v := value.Slice()
		if len(v) == 3 && v[0].String() == ":like" {
			return v[1].String(), v[2].String()
		}
	}
	return value.String(), "\\"
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
	if storage, ok := reader.(ColumnStorage); ok {
		reader = storage.GetCachedReader(ReadCompressed)
		defer ReleaseColumnReader(reader)
	}
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
	var cursor scm.TextCursor
	var scratch scm.TextScratch
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
				addKey := func(previous, current rune) {
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
				addRune := func(current rune) {
					current = normalizeBigramRune(current)
					if havePrevious {
						addKey(previous, current)
					}
					previous, havePrevious = current, true
				}
				if value.IsBufferString() {
					cursor.Reset(value)
					for {
						current, _, ok := cursor.NextRune(&scratch)
						if !ok {
							break
						}
						addRune(current)
					}
				} else {
					for _, current := range value.String() {
						addRune(current)
					}
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
func (s *bigramIndex) candidatesPattern(pattern, escape string) (recSetShard, bool) {
	return s.candidatesLiterals(scm.LikeLiteralRuns(scm.DecodeLikePattern(pattern, escape)))
}

func (s *bigramIndex) candidatesLiterals(runs []string) (recSetShard, bool) {
	var setScratch [32]queryBigramPosting
	sets := setScratch[:0]
	var previous rune
	havePrevious := false
	for _, run := range runs {
		havePrevious = false
		for _, current := range run {
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

// Weak identities preserve object identity across GC/address reuse without
// retaining evicted column payloads. This method is promoted by every storage.
func (s *storageJITFunctions) bigramCacheIdentity() weak.Pointer[storageJITFunctions] {
	return weak.Make(s)
}

func validLikeBinding(value scm.Scmer) bool {
	if value.IsSlice() {
		v := value.Slice()
		if len(v) == 3 && v[0].String() == ":like" {
			return !v[1].IsNil() && !v[2].IsNil()
		}
	}
	return !value.IsNil()
}
