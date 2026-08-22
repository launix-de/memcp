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

import "math/rand"
import "reflect"
import "testing"

import "github.com/launix-de/memcp/scm"

func bitmapForPredicate(universe uint32, predicate func(uint32) bool) ([]uint32, []bool) {
	words := make([]uint32, (universe+31)/32)
	want := make([]bool, universe)
	for position := uint32(0); position < universe; position++ {
		want[position] = predicate(position)
		if want[position] {
			words[position>>5] |= uint32(1) << (position & 31)
		}
	}
	return words, want
}

func TestCompressedRecSetRepresentationsAndMut(t *testing.T) {
	const universe = uint32(1009)
	patterns := []struct {
		name      string
		predicate func(uint32) bool
	}{
		{"dense", func(position uint32) bool { return position%7 != 0 }},
		{"sparse", func(position uint32) bool { return position%97 == 0 }},
		{"ranges", func(position uint32) bool { return position >= 80 && position < 900 }},
		{"entropy", func(position uint32) bool { return rand.New(rand.NewSource(int64(position))).Intn(5) == 0 }},
	}
	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			words, want := bitmapForPredicate(universe, pattern.predicate)
			compressed := compressRecSetBitmap(words, universe)
			dst := recSetShardFromSingleFullRange(nil, universe)
			compressed.set.AndMut(&dst)
			verifyRecSetShard(t, pattern.name, dst, want)
		})
	}
}

func TestCompressedRecSetAdaptiveKinds(t *testing.T) {
	const universe = uint32(65536)
	fixtures := []struct {
		name      string
		predicate func(uint32) bool
		want      any
	}{
		{"raw", func(position uint32) bool { return position%2 == 0 }, (*compressedBitmap)(nil)},
		{"rans", func(position uint32) bool { return position%5 == 0 }, (*compressedRANSBitmap)(nil)},
		{"positive", func(position uint32) bool { return position%10007 == 0 }, (*compressedPositive)(nil)},
		{"ranges", func(position uint32) bool { return position >= 1000 && position < 60000 }, (*compressedRanges)(nil)},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			words, _ := bitmapForPredicate(universe, fixture.predicate)
			got := compressRecSetBitmap(words, universe).set
			if reflect.TypeOf(got) != reflect.TypeOf(fixture.want) {
				t.Fatalf("selected %#v (%T), want %T", got, got, fixture.want)
			}
		})
	}
}

func TestRANSBitmapRoundTrip(t *testing.T) {
	for _, divisor := range []uint32{2, 3, 17, 101} {
		const universe = uint32(4099)
		words, want := bitmapForPredicate(universe, func(position uint32) bool {
			return (position*2654435761+17)%divisor == 0
		})
		encoded := encodeRANSBitmap(words, universe, countBitmap(words))
		dst := recSetShardFromSingleFullRange(nil, universe)
		encoded.AndMut(&dst)
		verifyRecSetShard(t, "rans", dst, want)
	}
}

func TestLikePatternBigrams(t *testing.T) {
	want := []uint64{
		bigramKey('C', 'a'),
		bigramKey('a', 's'),
		bigramKey('n', 'o'),
	}
	if got := patternBigrams("%CaS_no%"); !reflect.DeepEqual(got, want) {
		t.Fatalf("bigrams = %v, want %v", got, want)
	}
	if got := patternBigrams("%x%"); len(got) != 0 {
		t.Fatalf("single-rune pattern produced bigrams: %v", got)
	}
}

func TestLikeBigramIndexCandidatesAreSafeSuperset(t *testing.T) {
	values := []string{"Casino", "CASINO", "Cas-no", "basic", "needle", ""}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	index := buildBigramIndex(uint32(len(values)), reader)

	check := func(pattern string, want []uint32) {
		t.Helper()
		candidate := index.candidates(patternBigrams(pattern))
		got := make([]uint32, 0)
		candidate.forEachRange(func(base, count uint32) bool {
			for position := base; position < base+count; position++ {
				got = append(got, position)
			}
			return true
		})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%q candidates = %v, want %v", pattern, got, want)
		}
	}
	check("%casino%", []uint32{0, 1})
	check("%needle%", []uint32{4})
	check("%missing%", []uint32{})
}

func TestLikeIndexThreeStageBindingReusesCache(t *testing.T) {
	values := []string{"Casino", "plain", "CASINO", "needle"}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	hook := LikeMatcher.Deploy(IndexDeployContext{MainCount: uint32(len(values)), Column: reader}, true)
	if hook == nil {
		t.Fatal("LIKE analyzer did not deploy its shard-local hook")
	}
	before := hook.ComputeSize()
	casino := hook.Bind(scm.NewString("%casino%"))
	needle := hook.Bind(scm.NewString("%needle%"))
	if casino == nil || needle == nil {
		t.Fatal("LIKE hook did not bind both query-local row matchers")
	}
	if got := casino([]uint32{0, 1, 2, 3}); !reflect.DeepEqual(got, []uint32{0, 2}) {
		t.Fatalf("casino matcher = %v, want [0 2]", got)
	}
	if got := needle([]uint32{3, 2, 1, 0}); !reflect.DeepEqual(got, []uint32{3}) {
		t.Fatalf("needle matcher = %v, want [3]", got)
	}
	if after := hook.ComputeSize(); after != before {
		t.Fatalf("binding changed reusable hook size from %d to %d", before, after)
	}
}

func TestIndexRowMatcherAllocatesNothingPerBatch(t *testing.T) {
	values := []string{"Casino", "plain", "CASINO", "needle"}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	hook := LikeMatcher.Deploy(IndexDeployContext{MainCount: uint32(len(values)), Column: reader}, true)
	matcher := hook.Bind(scm.NewString("%casino%"))
	var batch [4]uint32
	allocs := testing.AllocsPerRun(100, func() {
		batch = [4]uint32{0, 1, 2, 3}
		if got := matcher(batch[:]); len(got) != 2 {
			t.Fatalf("matcher returned %d rows, want 2", len(got))
		}
	})
	if allocs != 0 {
		t.Fatalf("IndexRowMatcher allocations per batch = %v, want 0", allocs)
	}
}
