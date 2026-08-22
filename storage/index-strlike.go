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
	return buildBigramIndex(ctx.MainCount, ctx.Column)
}

type bigramIndex struct {
	universe uint32
	grams    map[uint64]compressedRecSet
	bytes    uint64
}

func (s *bigramIndex) ComputeSize() uint { return uint(s.bytes) }

func (s *bigramIndex) Bind(lower scm.Scmer) IndexRowMatcher {
	keys := patternBigrams(lower.String())
	if len(keys) == 0 {
		return nil
	}
	candidates := s.candidates(keys)
	return func(ids []uint32) []uint32 {
		out := ids[:0]
		for _, id := range ids {
			// The immutable bigram index covers main rows. Delta rows retain the
			// original STRLIKE residual and must pass this candidate filter.
			if id >= s.universe || candidates.contains(id) {
				out = append(out, id)
			}
		}
		return out
	}
}

func bigramKey(left, right rune) uint64 {
	return uint64(uint32(unicode.ToLower(left)))<<32 | uint64(uint32(unicode.ToLower(right)))
}

func patternBigrams(pattern string) []uint64 {
	seen := make(map[uint64]struct{})
	result := make([]uint64, 0, len(pattern)/2)
	var previous rune
	havePrevious := false
	for _, current := range pattern {
		if current == '%' || current == '_' {
			havePrevious = false
			continue
		}
		if havePrevious {
			key := bigramKey(previous, current)
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, key)
			}
		}
		previous = current
		havePrevious = true
	}
	return result
}

func buildBigramIndex(count uint32, reader ColumnReader) *bigramIndex {
	index := &bigramIndex{universe: count, grams: make(map[uint64]compressedRecSet)}
	const readBatch = 1024
	values := make([]scm.Scmer, readBatch)
	wordCount := (count + 31) / 32
	temporary := make(map[uint64][]uint32)
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
					if havePrevious {
						key := bigramKey(previous, current)
						words := temporary[key]
						if words == nil {
							words = make([]uint32, wordCount)
							temporary[key] = words
						}
						position := base + offset
						words[position>>5] |= uint32(1) << (position & 31)
					}
					previous = current
					havePrevious = true
				}
			}
			values[offset] = scm.NewNil()
		}
	}
	for key, words := range temporary {
		compressed := compressRecSetBitmap(words, count)
		index.bytes += uint64(compressed.bytes)
		index.grams[key] = compressed
	}
	index.bytes += uint64(len(index.grams)) * uint64(unsafe.Sizeof(uint64(0))+unsafe.Sizeof(compressedRecSet{}))
	return index
}

func (s *bigramIndex) candidates(keys []uint64) recSetShard {
	sets := make([]compressedRecSet, 0, len(keys))
	for _, key := range keys {
		set, exists := s.grams[key]
		if !exists {
			return recSetShard{kind: recSetRanges, universe: s.universe}
		}
		sets = append(sets, set)
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].count < sets[j].count })
	result := recSetShardFromSingleFullRange(nil, s.universe)
	for _, set := range sets {
		set.set.AndMut(&result)
		if result.count == 0 {
			break
		}
	}
	return result
}
