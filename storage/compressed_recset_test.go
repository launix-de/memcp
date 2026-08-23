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

import "fmt"
import "unsafe"
import "reflect"
import "testing"
import "math/rand"

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

func TestCompressedPositiveAndMutStreamsWithoutAllocation(t *testing.T) {
	source := &compressedPositive{
		universe: 64,
		count:    6,
		values:   compressedStorageIntFromValues([]uint32{1, 3, 8, 21, 34, 55}),
	}
	initial := [...]uint32{0, 1, 2, 3, 5, 8, 13, 21, 34, 55, 63}
	backing := make([]uint32, len(initial))
	var got []uint32
	allocs := testing.AllocsPerRun(100, func() {
		copy(backing, initial[:])
		dst := recSetShard{
			kind:     recSetPositive,
			universe: 64,
			data:     backing,
			used:     uint32(len(backing)),
			count:    int64(len(backing)),
		}
		source.AndMut(&dst)
		got = dst.listedValues()
	})
	if allocs != 0 {
		t.Fatalf("compressed positive intersection allocations = %v, want 0", allocs)
	}
	if !reflect.DeepEqual(got, []uint32{1, 3, 8, 21, 34, 55}) {
		t.Fatalf("streamed intersection = %v", got)
	}
}

func TestCompressedRecSetIntersectionRepresentationMatrix(t *testing.T) {
	const universe = uint32(1 << 20)
	sources := []struct {
		name      string
		predicate func(uint32) bool
		want      any
	}{
		{"raw", func(position uint32) bool { return position%2 == 0 }, (*compressedBitmap)(nil)},
		{"positive", func(position uint32) bool { return position%10007 == 0 }, (*compressedPositive)(nil)},
		{"ranges", func(position uint32) bool { return position >= 1000 && position < 60000 }, (*compressedRanges)(nil)},
	}
	destinations := []struct {
		name      string
		predicate func(uint32) bool
	}{
		{"full", func(uint32) bool { return true }},
		{"bitmap", func(position uint32) bool { return position%3 != 0 }},
		{"positive", func(position uint32) bool { return position%997 == 0 }},
		{"ranges", func(position uint32) bool {
			return (position >= 500 && position < 3000) || (position >= 40000 && position < 62000)
		}},
		{"empty", func(uint32) bool { return false }},
	}
	for _, source := range sources {
		words, sourceWant := bitmapForPredicate(universe, source.predicate)
		compressed := compressRecSetBitmap(words, universe)
		if reflect.TypeOf(compressed.set) != reflect.TypeOf(source.want) {
			t.Fatalf("matrix source %s selected %T, want %T", source.name, compressed.set, source.want)
		}
		for _, destination := range destinations {
			t.Run(source.name+"/"+destination.name, func(t *testing.T) {
				destinationWords, destinationWant := bitmapForPredicate(universe, destination.predicate)
				dst := buildRecSetShard(destinationWant)
				// Force the bitmap destination named above; the adaptive builder is
				// intentionally free to select another representation.
				if destination.name == "bitmap" {
					dst = recSetShardFromBitmap(nil, universe, destinationWords)
				}
				compressed.set.AndMut(&dst)
				want := make([]bool, universe)
				for position := range want {
					want[position] = sourceWant[position] && destinationWant[position]
				}
				verifyRecSetShard(t, source.name+"/"+destination.name, dst, want)
			})
		}
	}
}

func TestRecSetCompressedInterfaceDirectRepresentationMatrix(t *testing.T) {
	const universe = uint32(257)
	bitmapWords, bitmapWant := bitmapForPredicate(universe, func(position uint32) bool { return position%3 != 0 })
	positiveValues := []uint32{1, 7, 31, 32, 128, 255}
	positiveWant := boolValues(universe, positiveValues)
	rangeData := []uint32{3, 29, 80, 71, 200, 40}
	rangeWant := make([]bool, universe)
	for pair := 0; pair < len(rangeData); pair += 2 {
		for value := rangeData[pair]; value < rangeData[pair]+rangeData[pair+1]; value++ {
			rangeWant[value] = true
		}
	}
	sources := []struct {
		name string
		set  recSetShard
		want []bool
	}{
		{"bitmap", recSetShardFromBitmap(nil, universe, bitmapWords), bitmapWant},
		{"positive", recSetShardFromList(nil, universe, positiveValues, uint32(len(positiveValues))), positiveWant},
		{"ranges", recSetShardFromRangePairs(nil, universe, rangeData, uint32(len(rangeData)/2)), rangeWant},
	}
	destinationPredicates := []struct {
		name      string
		predicate func(uint32) bool
	}{
		{"bitmap", func(position uint32) bool { return position%5 != 0 }},
		{"positive", func(position uint32) bool { return position%17 == 0 }},
		{"ranges", func(position uint32) bool { return position >= 20 && position < 230 }},
		{"full", func(uint32) bool { return true }},
	}
	for _, source := range sources {
		for _, destination := range destinationPredicates {
			t.Run(source.name+"/"+destination.name, func(t *testing.T) {
				words, destinationWant := bitmapForPredicate(universe, destination.predicate)
				dst := buildRecSetShard(destinationWant)
				if destination.name == "bitmap" {
					dst = recSetShardFromBitmap(nil, universe, words)
				}
				if destination.name == "full" {
					dst = recSetShardFromSingleFullRange(nil, universe)
				}
				source.set.AndMut(&dst)
				want := make([]bool, universe)
				for position := range want {
					want[position] = source.want[position] && destinationWant[position]
				}
				verifyRecSetShard(t, source.name+"/"+destination.name, dst, want)
			})
		}
	}
}

func TestCompressedRecSetAndMutRejectsUniverseMismatch(t *testing.T) {
	words, _ := bitmapForPredicate(64, func(position uint32) bool { return position%3 == 0 })
	compressed := compressRecSetBitmap(words, 64)
	dst := recSetShardFromSingleFullRange(nil, 65)
	defer func() {
		if recover() == nil {
			t.Fatal("AndMut accepted different RecSet universes")
		}
	}()
	compressed.set.AndMut(&dst)
}

func TestCompressedRecSetAndMutRandomizedTailAndRepresentations(t *testing.T) {
	random := rand.New(rand.NewSource(4711))
	for _, universe := range []uint32{1, 2, 31, 32, 33, 63, 64, 65, 257, 1009} {
		for iteration := 0; iteration < 40; iteration++ {
			sourceWords := make([]uint32, (universe+31)/32)
			destinationWords := make([]uint32, len(sourceWords))
			destinationValues := make([]uint32, 0, universe/2)
			want := make([]bool, universe)
			for position := uint32(0); position < universe; position++ {
				source := random.Intn(9) < 3
				destination := random.Intn(7) < 4
				if source {
					sourceWords[position>>5] |= uint32(1) << (position & 31)
				}
				if destination {
					destinationWords[position>>5] |= uint32(1) << (position & 31)
					destinationValues = append(destinationValues, position)
				}
				want[position] = source && destination
			}
			compressed := compressRecSetBitmap(sourceWords, universe)
			destinations := []recSetShard{
				recSetShardFromBitmap(nil, universe, append([]uint32(nil), destinationWords...)),
				recSetShardFromList(nil, universe, append([]uint32(nil), destinationValues...), uint32(len(destinationValues))),
				buildRecSetShard(boolValues(universe, destinationValues)),
			}
			for destinationIndex, destination := range destinations {
				compressed.set.AndMut(&destination)
				verifyRecSetShard(t, fmt.Sprintf("universe=%d iteration=%d destination=%d", universe, iteration, destinationIndex), destination, want)
			}
		}
	}
}

func boolValues(universe uint32, values []uint32) []bool {
	result := make([]bool, universe)
	for _, value := range values {
		result[value] = true
	}
	return result
}

func TestCompressedRecSetAdaptiveKinds(t *testing.T) {
	const universe = uint32(1 << 20)
	fixtures := []struct {
		name      string
		predicate func(uint32) bool
		want      any
	}{
		{"raw", func(position uint32) bool { return position%2 == 0 }, (*compressedBitmap)(nil)},
		{"positive", func(position uint32) bool { return position%100003 == 0 }, (*compressedPositive)(nil)},
		{"ranges", func(position uint32) bool { return position >= 1000 && position < 1_000_000 }, (*compressedRanges)(nil)},
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

func TestLikeBigramIndexCandidatesAreSafeSuperset(t *testing.T) {
	values := []string{"Casino", "CASINO", "Cas-no", "basic", "needle", ""}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	index := buildBigramIndex(uint32(len(values)), reader)

	check := func(pattern string, want []uint32) {
		t.Helper()
		candidate, constrained := index.candidatesPattern(pattern)
		if !constrained {
			t.Fatalf("%q did not produce a candidate constraint", pattern)
		}
		if len(want) > 0 && candidate.kind != recSetBitmap {
			t.Fatalf("%q query-local candidates use %v, want bitmap", pattern, candidate.kind)
		}
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
	if candidates, universe, ok := index.EstimateCandidates(scm.NewString("%casino%")); !ok || candidates != 2 || universe != uint32(len(values)) {
		t.Fatalf("casino estimate = (%d, %d, %v), want (2, %d, true)", candidates, universe, ok, len(values))
	}
	if candidates, universe, ok := index.EstimateCandidates(scm.NewString("%missing%")); !ok || candidates != 0 || universe != uint32(len(values)) {
		t.Fatalf("missing estimate = (%d, %d, %v), want (0, %d, true)", candidates, universe, ok, len(values))
	}
	if _, constrained := index.candidatesPattern("%x%"); constrained {
		t.Fatal("single-rune pattern unexpectedly produced a bigram constraint")
	}
	if _, _, ok := index.EstimateCandidates(scm.NewString("%x%")); ok {
		t.Fatal("single-rune pattern unexpectedly produced a cardinality estimate")
	}
}

func TestLikeIndexThreeStageBindingReusesCache(t *testing.T) {
	values := make([]string, minimumBigramIndexRows)
	copy(values, []string{"Casino", "plain", "CASINO", "needle"})
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

func TestLikeIndexLeavesDeltaRowsForResidualFilter(t *testing.T) {
	values := []string{"Casino", "plain", "CASINO", "needle"}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	hook := buildBigramIndex(uint32(len(values)), reader)
	matcher := hook.Bind(scm.NewString("%missing%"))
	if got := matcher([]uint32{0, 1, 2, 3, 4, 5}); !reflect.DeepEqual(got, []uint32{4, 5}) {
		t.Fatalf("matcher dropped or admitted wrong rows: %v, want delta rows [4 5]", got)
	}
}

func TestLikeIndexComputeSizeIncludesCompleteSortedTable(t *testing.T) {
	values := []string{"Casino", "plain", "CASINO", "needle"}
	reader := ColumnReaderFunc(func(recid uint32) scm.Scmer { return scm.NewString(values[recid]) })
	index := buildBigramIndex(uint32(len(values)), reader)
	want := uint(unsafe.Sizeof(*index)) + uint(index.bytes+index.grams.ComputeSize())
	if got := index.ComputeSize(); got != want {
		t.Fatalf("ComputeSize = %d, want exact owned size %d", got, want)
	}
	if cap(index.grams.keys) != len(index.grams.keys) || cap(index.grams.sets) != len(index.grams.sets) || cap(index.grams.counts) != len(index.grams.counts) {
		t.Fatalf("persistent gram table retains spare capacity: keys=%d/%d sets=%d/%d counts=%d/%d",
			len(index.grams.keys), cap(index.grams.keys), len(index.grams.sets), cap(index.grams.sets), len(index.grams.counts), cap(index.grams.counts))
	}
	minimumPayload := uint(unsafe.Sizeof(*index)) + uint(index.bytes) + uint(len(index.grams.keys))*(uint(unsafe.Sizeof(uint64(0)))+uint(unsafe.Sizeof(CompressedRecSet(nil)))+uint(unsafe.Sizeof(uint32(0))))
	if index.ComputeSize() != minimumPayload {
		t.Fatalf("ComputeSize %d differs from exact sorted-table payload %d", index.ComputeSize(), minimumPayload)
	}
}

func TestCompressedRecSetStoredSizeMatchesOwnedCapacity(t *testing.T) {
	bitmapWords := make([]uint32, 1024)
	for index := range bitmapWords {
		if index&1 == 0 {
			bitmapWords[index] = 0xaaaaaaaa
		} else {
			bitmapWords[index] = 0x55555555
		}
	}
	positiveWords := make([]uint32, 1024)
	positiveWords[0] = 1
	rangeWords := make([]uint32, 1024)
	for index := 0; index < 16; index++ {
		rangeWords[index] = ^uint32(0)
		rangeWords[len(rangeWords)-1-index] = ^uint32(0)
	}
	tests := []struct {
		name     string
		words    []uint32
		wantKind any
	}{
		{name: "bitmap", words: bitmapWords, wantKind: (*compressedBitmap)(nil)},
		{name: "positive", words: positiveWords, wantKind: (*compressedPositive)(nil)},
		{name: "ranges", words: rangeWords, wantKind: (*compressedRanges)(nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compressed := compressRecSetBitmap(test.words, uint32(len(test.words)*32))
			if reflect.TypeOf(compressed.set) != reflect.TypeOf(test.wantKind) {
				t.Fatalf("representation = %T, want %T", compressed.set, test.wantKind)
			}
			var owned uint32
			switch set := compressed.set.(type) {
			case *compressedBitmap:
				owned = uint32(unsafe.Sizeof(*set)) + uint32(cap(set.words))*uint32(unsafe.Sizeof(uint32(0)))
			case *compressedPositive:
				owned = uint32(unsafe.Sizeof(*set)) + uint32(cap(set.values.chunk))*uint32(unsafe.Sizeof(uint64(0)))
			case *compressedRanges:
				owned = uint32(unsafe.Sizeof(*set)) +
					uint32(cap(set.bases.chunk)+cap(set.lengths.chunk))*uint32(unsafe.Sizeof(uint64(0)))
			default:
				t.Fatalf("unexpected compressed representation %T", compressed.set)
			}
			if compressed.bytes != owned {
				t.Fatalf("stored size = %d, actual owned capacity = %d for %T", compressed.bytes, owned, compressed.set)
			}
		})
	}
}

func TestIndexRowMatcherAllocatesNothingPerBatch(t *testing.T) {
	values := make([]string, minimumBigramIndexRows)
	copy(values, []string{"Casino", "plain", "CASINO", "needle"})
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

func TestLikeIndexSmallShardUsesSharedNoopHook(t *testing.T) {
	reader := ColumnReaderFunc(func(uint32) scm.Scmer { return scm.NewString("Casino") })
	first := LikeMatcher.Deploy(IndexDeployContext{MainCount: minimumBigramIndexRows - 1, Column: reader}, true)
	second := LikeMatcher.Deploy(IndexDeployContext{MainCount: 1, Column: reader}, true)
	if first == nil || first != second || first.ComputeSize() != 0 {
		t.Fatalf("small shards did not reuse zero-size hook: first=%T second=%T", first, second)
	}
	if matcher := first.Bind(scm.NewString("%casino%")); matcher != nil {
		t.Fatal("small-shard hook unexpectedly installed a row matcher")
	}
}
