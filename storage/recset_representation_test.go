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

import (
	"math/rand"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestProjectJoinNumericLookupMatchesIntegralStorageRepresentations(t *testing.T) {
	keys := recSetProjectKeys{
		width:  1,
		values: []scm.Scmer{scm.NewInt(7), scm.NewInt(42)},
	}
	if !keys.buildNumericLookup() {
		t.Fatal("integral keys should use the numeric lookup")
	}
	if !keys.contains([]scm.Scmer{scm.NewFloat(42)}) {
		t.Fatal("StorageSeq float representation should match an integral source key")
	}
	if keys.contains([]scm.Scmer{scm.NewFloat(42.5)}) {
		t.Fatal("non-integral float must not match an integral source key")
	}
}

func TestProjectJoinPhysicalAccessUsesCostModel(t *testing.T) {
	if !projectJoinDensePreferred(1, 1_000_000, true, false) {
		t.Fatal("projection without a usable index must scan")
	}
	if projectJoinDensePreferred(10, 80_000, true, true) {
		t.Fatal("a few indexed keys should use point probes")
	}
	if !projectJoinDensePreferred(100, 80_000, true, true) {
		t.Fatal("a broad integral key set should use the numeric dense scan")
	}
	if projectJoinDensePreferred(100, 80_000, false, true) {
		t.Fatal("generic tuple comparisons should retain indexed probes at this cardinality")
	}
}

// buildRecSetShard drives a recSetShardBuilder over want (a boolean mask
// indexed by recid, len(want)==universe) exactly the way every real caller
// (recset.go's collectRecSet/projectJoinKeysPart, analyzer.go's
// BuildSkipList) does: one straight ascending add() call per recid. Every
// escalation is self-contained inside the builder (see its doc comment), so
// unlike an earlier version of this harness, there is nothing to watch for
// or replay here — mirroring that a real caller's filter predicate is never
// re-run either.
func buildRecSetShard(want []bool) recSetShard {
	return buildRecSetShardWithBreak(want, 0)
}

func buildRecSetShardWithBreak(want []bool, breakAt uint32) recSetShard {
	universe := uint32(len(want))
	b := newRecSetShardBuilder(nil, universe, true, breakAt)
	for recid, hit := range want {
		b.add(uint32(recid), hit)
	}
	return b.finish()
}

func wantFromRange(universe uint32, ranges [][2]uint32) []bool {
	want := make([]bool, universe)
	for _, r := range ranges {
		for i := r[0]; i < r[0]+r[1]; i++ {
			want[i] = true
		}
	}
	return want
}

func verifyRecSetShard(t *testing.T, name string, part recSetShard, want []bool) {
	t.Helper()
	universe := uint32(len(want))

	var wantCount int64
	for _, v := range want {
		if v {
			wantCount++
		}
	}
	if part.count != wantCount {
		t.Errorf("%s: count = %d, want %d", name, part.count, wantCount)
	}

	// contains() must agree with want for every recid, plus recids beyond
	// (and exactly at) the universe boundary must always be false — this is
	// the correctness property the whole "no Full/Negative tag" redesign is
	// about: nothing after the universe boundary can ever read as present.
	for recid := uint32(0); recid < universe; recid++ {
		if got := part.contains(recid); got != want[recid] {
			t.Errorf("%s: contains(%d) = %v, want %v", name, recid, got, want[recid])
		}
	}
	for _, recid := range []uint32{universe, universe + 1, universe + 1000} {
		if part.contains(recid) {
			t.Errorf("%s: contains(%d) beyond universe %d = true, want false", name, recid, universe)
		}
	}

	// forEachID must visit exactly the matching recids, in ascending order.
	var gotIDs []uint32
	part.forEachID(func(id uint32) bool {
		gotIDs = append(gotIDs, id)
		return true
	})
	checkIDsMatchWant(t, name, gotIDs, want)

	// forEachRange must be consistent with forEachID: same total ids (via
	// expansion), ascending, non-overlapping, non-adjacent-when-coalescible
	// ranges (adjacency isn't required to be coalesced across representations
	// other than recSetRanges — bitmap coalesces greedily too — so just
	// require expansion equality plus strictly-increasing, non-overlapping
	// bases).
	var expanded []uint32
	var lastEnd uint32
	first := true
	part.forEachRange(func(base, count uint32) bool {
		if count == 0 {
			t.Errorf("%s: forEachRange yielded a zero-length range at base %d", name, base)
		}
		if !first && base < lastEnd {
			t.Errorf("%s: forEachRange ranges overlap or go backwards: base=%d < lastEnd=%d", name, base, lastEnd)
		}
		first = false
		lastEnd = base + count
		for id := base; id < base+count; id++ {
			expanded = append(expanded, id)
		}
		return true
	})
	checkIDsMatchWant(t, name, expanded, want)
}

func checkIDsMatchWant(t *testing.T, name string, ids []uint32, want []bool) {
	t.Helper()
	i := 0
	for recid, hit := range want {
		if !hit {
			continue
		}
		if i >= len(ids) {
			t.Fatalf("%s: id list too short, missing recid %d", name, recid)
		}
		if ids[i] != uint32(recid) {
			t.Fatalf("%s: id list[%d] = %d, want %d", name, i, ids[i], recid)
		}
		i++
	}
	if i != len(ids) {
		t.Fatalf("%s: id list has %d extra trailing entries (first extra: %d)", name, len(ids)-i, ids[i])
	}
}

func TestRecSetShardBuilderPatterns(t *testing.T) {
	cases := []struct {
		name     string
		universe uint32
		ranges   [][2]uint32 // (base,count) hit runs
	}{
		{"Empty", 1000, nil},
		{"Full", 1000, [][2]uint32{{0, 1000}}},
		{"SingleBigRange", 100000, [][2]uint32{{12345, 54321}}},
		{"SingleRangeAtStart", 1000, [][2]uint32{{0, 500}}},
		{"SingleRangeAtEnd", 1000, [][2]uint32{{500, 500}}},
		{"TwoBigRanges", 10000, [][2]uint32{{100, 3000}, {5000, 3000}}},
		{"ThreeSingletons", 1000, [][2]uint32{{10, 1}, {500, 1}, {900, 1}}},
		{"FourSingletons_TriggersPositive", 1000, [][2]uint32{{10, 1}, {300, 1}, {600, 1}, {900, 1}}},
		{"MostlyFullFewHoles", 10000, [][2]uint32{{0, 100}, {101, 2000}, {2005, 7995}}}, // holes at 100, 2000-2005
		{"ManySingletonsPastBudget", 2000, nil},                                         // filled below
		{"AdjacentRuns", 1000, [][2]uint32{{0, 10}, {10, 10}, {20, 10}}},                // builder must coalesce contiguous add() calls
		{"UniverseZero", 0, nil},
		{"UniverseOne_Hit", 1, [][2]uint32{{0, 1}}},
		{"UniverseOne_Miss", 1, nil},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			want := wantFromRange(tc.universe, tc.ranges)
			if tc.name == "ManySingletonsPastBudget" {
				rng := rand.New(rand.NewSource(1))
				for i := range want {
					want[i] = rng.Intn(3) == 0 // ~33% scattered hits, no clustering
				}
			}
			part := buildRecSetShard(want)
			verifyRecSetShard(t, tc.name, part, want)
		})
	}
}

// TestRecSetShardBuilderNeverReEvaluatesFilter is the regression test for
// the bug this file's escalation redesign fixes: an earlier version cleared
// the backing array on escalation and relied on the *caller* re-running its
// scan and filter predicate to replay the abandoned prefix. Since add()'s
// hit argument stands in for a real SQL predicate evaluation (interpreted
// Scheme, column reads — arbitrarily expensive), that meant escalating late
// silently doubled scan+filter cost. Every escalation must now be
// self-contained: driving add() in one straight forward pass, reading each
// recid's hit value exactly once, must produce a correct shard regardless of
// which phase(s) it escalates through.
func TestRecSetShardBuilderNeverReEvaluatesFilter(t *testing.T) {
	drive := func(name string, universe uint32, want []bool, wantEscalatesTo recSetRepresentation) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			calls := make([]int, universe)
			b := newRecSetShardBuilder(nil, universe, true, 0)
			for recid := uint32(0); recid < universe; recid++ {
				calls[recid]++
				b.add(recid, want[recid])
			}
			part := b.finish()
			verifyRecSetShard(t, name, part, want)
			if part.kind != wantEscalatesTo {
				t.Fatalf("%s: kind = %v, want %v (fixture didn't escalate the way this test expects)", name, part.kind, wantEscalatesTo)
			}
			for recid, n := range calls {
				if n != 1 {
					t.Fatalf("%s: recid %d had its filter value read %d times, want exactly 1 (escalation re-drove the caller's predicate)", name, recid, n)
				}
			}
		})
	}

	// Direct ranges→bitmap: many non-singleton (count=2) runs so the
	// range-pair budget overflows in addRange's own escalation check, before
	// the singleton-run threshold (which stays 0) is ever reached.
	universeA := uint32(2000) // budget = (2000+31)/32 = 63 words = 31 pairs
	wantA := make([]bool, universeA)
	for base := uint32(0); base+1 < universeA; base += 4 {
		wantA[base] = true
		wantA[base+1] = true
	}
	drive("DirectRangesToBitmap", universeA, wantA, recSetBitmap)

	// Direct positive→bitmap: many isolated singletons, past both the
	// singleton-run threshold (ranges→positive) and the flat-list budget
	// (positive→bitmap), all reached via addPositive's own escalation check.
	universeB := uint32(2000) // budget = 63 hits
	wantB := make([]bool, universeB)
	for pos := uint32(5); pos < universeB; pos += 20 { // ~100 scattered singletons > 63
		wantB[pos] = true
	}
	drive("DirectPositiveToBitmap", universeB, wantB, recSetBitmap)

	// ranges→positive's *own* overflow-to-bitmap fallback (inside
	// tryConvertToPositive): a few large (non-singleton) ranges whose total
	// hit count alone exceeds budget, plus just enough singletons (>3) to
	// trigger the ranges→positive conversion that discovers the overflow.
	universeC := uint32(100000) // budget = (100000+31)/32 = 3126 words
	wantC := wantFromRange(universeC, [][2]uint32{
		{100, 2000}, {30000, 2000}, {60000, 2000}, // 3 big ranges, 6000 hits total > 3126 budget
		{10, 1}, {50000, 1}, {70000, 1}, {90000, 1}, // 4 singletons > maxSingletonRuns(3)
	})
	drive("RangesToPositiveOverflowToBitmap", universeC, wantC, recSetBitmap)
}

// TestRecSetShardBuilderPhaseTransitions specifically checks the two
// escalation boundaries (singleton-count threshold, and budget overflow) at
// and around their exact trigger points, since off-by-one errors there are
// the likeliest bug class in an online run-length builder.
func TestRecSetShardBuilderPhaseTransitions(t *testing.T) {
	universe := uint32(100000)
	// Exactly maxSingletonRuns (3) isolated hits: must stay in ranges.
	ranges3 := [][2]uint32{{10, 1}, {50000, 1}, {90000, 1}}
	want3 := wantFromRange(universe, ranges3)
	part3 := buildRecSetShard(want3)
	if part3.kind != recSetRanges {
		t.Errorf("3 singletons: kind = %v, want recSetRanges", part3.kind)
	}
	verifyRecSetShard(t, "3singletons", part3, want3)

	// One more (4) must flip to positive.
	ranges4 := [][2]uint32{{10, 1}, {30000, 1}, {50000, 1}, {90000, 1}}
	want4 := wantFromRange(universe, ranges4)
	part4 := buildRecSetShard(want4)
	if part4.kind != recSetPositive {
		t.Errorf("4 singletons: kind = %v, want recSetPositive", part4.kind)
	}
	verifyRecSetShard(t, "4singletons", part4, want4)

	// A large number of scattered singletons that overflow even the flat
	// positive-list budget must land in bitmap.
	rng := rand.New(rand.NewSource(2))
	wantDense := make([]bool, universe)
	for i := range wantDense {
		wantDense[i] = rng.Intn(2) == 0 // ~50% scattered — no clustering, way past any list budget
	}
	partDense := buildRecSetShard(wantDense)
	if partDense.kind != recSetBitmap {
		t.Errorf("dense random: kind = %v, want recSetBitmap", partDense.kind)
	}
	verifyRecSetShard(t, "dense-random", partDense, wantDense)
}

// TestRecSetShardBuilderMainDeltaBreak checks that breakAt (set by real
// callers to a shard's main_count) forces a contiguous hit-run to split into
// two range pairs exactly at the boundary, instead of coalescing across it —
// so forEachVisibleRun and the bulk-wired scan paths in recset.go can slice
// a range into its main/delta sub-runs with a plain bound instead of
// clamping. Correctness (contains/forEachID/forEachRange) must be unaffected
// by the split — it's purely a change to how the same logical hits are
// grouped into pairs.
func TestRecSetShardBuilderMainDeltaBreak(t *testing.T) {
	universe := uint32(1000)

	t.Run("RunStraddlesBreak", func(t *testing.T) {
		want := wantFromRange(universe, [][2]uint32{{400, 300}}) // hits 400..699
		part := buildRecSetShardWithBreak(want, 550)
		verifyRecSetShard(t, "RunStraddlesBreak", part, want)
		if part.kind != recSetRanges {
			t.Fatalf("kind = %v, want recSetRanges", part.kind)
		}
		got := part.listedRanges()
		wantPairs := []uint32{400, 150, 550, 150}
		if len(got) != len(wantPairs) {
			t.Fatalf("listedRanges = %v, want %v (run must split into two pairs at breakAt=550)", got, wantPairs)
		}
		for i := range wantPairs {
			if got[i] != wantPairs[i] {
				t.Fatalf("listedRanges = %v, want %v", got, wantPairs)
			}
		}
	})

	t.Run("BreakAtExistingRunStart_NoSpuriousSplit", func(t *testing.T) {
		want := wantFromRange(universe, [][2]uint32{{100, 50}, {600, 50}})
		part := buildRecSetShardWithBreak(want, 600)
		verifyRecSetShard(t, "BreakAtExistingRunStart", part, want)
		got := part.listedRanges()
		wantPairs := []uint32{100, 50, 600, 50}
		if len(got) != len(wantPairs) {
			t.Fatalf("listedRanges = %v, want %v (breakAt landing exactly on a run start must not add an empty pair)", got, wantPairs)
		}
		for i := range wantPairs {
			if got[i] != wantPairs[i] {
				t.Fatalf("listedRanges = %v, want %v", got, wantPairs)
			}
		}
	})

	t.Run("BreakDisabled_ZeroMeansNoSplit", func(t *testing.T) {
		want := wantFromRange(universe, [][2]uint32{{0, 1000}})
		part := buildRecSetShardWithBreak(want, 0)
		verifyRecSetShard(t, "BreakDisabled", part, want)
		got := part.listedRanges()
		if len(got) != 2 || got[0] != 0 || got[1] != 1000 {
			t.Fatalf("listedRanges = %v, want [0 1000] (breakAt=0 must be a no-op)", got)
		}
	})

	t.Run("OnlyOneOfTwoRunsStraddles", func(t *testing.T) {
		want := wantFromRange(universe, [][2]uint32{{100, 50}, {400, 300}}) // second run 400..699 straddles 550
		part := buildRecSetShardWithBreak(want, 550)
		verifyRecSetShard(t, "OnlyOneOfTwoRunsStraddles", part, want)
		got := part.listedRanges()
		wantPairs := []uint32{100, 50, 400, 150, 550, 150}
		if len(got) != len(wantPairs) {
			t.Fatalf("listedRanges = %v, want %v", got, wantPairs)
		}
		for i := range wantPairs {
			if got[i] != wantPairs[i] {
				t.Fatalf("listedRanges = %v, want %v", got, wantPairs)
			}
		}
	})
}

// TestRecSetShardUnionIntersect checks unionRecSetShards/intersectRecSetShards
// across representative kind combinations (ranges/ranges, ranges/positive,
// ranges/bitmap, positive/bitmap, mixed universes) against a reference
// boolean-array computation.
func TestRecSetShardUnionIntersect(t *testing.T) {
	universe := uint32(10000)
	rng := rand.New(rand.NewSource(3))

	makeClustered := func(nRanges int, maxLen uint32) []bool {
		want := make([]bool, universe)
		for i := 0; i < nRanges; i++ {
			base := uint32(rng.Intn(int(universe)))
			length := uint32(rng.Intn(int(maxLen))) + 1
			for j := base; j < base+length && j < universe; j++ {
				want[j] = true
			}
		}
		return want
	}
	makeScattered := func(density int) []bool {
		want := make([]bool, universe)
		for i := range want {
			want[i] = rng.Intn(density) == 0
		}
		return want
	}

	scenarios := []struct {
		name string
		want []bool
	}{
		{"clustered_A", makeClustered(3, 2000)},
		{"clustered_B", makeClustered(2, 500)},
		{"scattered_A", makeScattered(50)}, // 2% density, no clustering -> positive
		{"scattered_B", makeScattered(3)},  // 33% density, no clustering -> bitmap
		{"empty", make([]bool, universe)},
		{"full", func() []bool {
			w := make([]bool, universe)
			for i := range w {
				w[i] = true
			}
			return w
		}()},
	}

	for i := 0; i < len(scenarios); i++ {
		for j := i + 1; j < len(scenarios); j++ {
			a, b := scenarios[i], scenarios[j]
			partA := buildRecSetShard(a.want)
			partB := buildRecSetShard(b.want)

			t.Run(a.name+"_union_"+b.name, func(t *testing.T) {
				wantUnion := make([]bool, universe)
				for k := range wantUnion {
					wantUnion[k] = a.want[k] || b.want[k]
				}
				got := unionRecSetShards(nil, []*recSetShard{&partA, &partB})
				verifyRecSetShard(t, "union", got, wantUnion)
			})
			t.Run(a.name+"_intersect_"+b.name, func(t *testing.T) {
				wantIntersect := make([]bool, universe)
				for k := range wantIntersect {
					wantIntersect[k] = a.want[k] && b.want[k]
				}
				got := intersectRecSetShards(nil, []*recSetShard{&partA, &partB})
				verifyRecSetShard(t, "intersect", got, wantIntersect)
			})
		}
	}
}

// TestRecSetShardUnionIntersectThreeWay covers >2-way combination, since the
// N-way interval sweep (intersectSortedRanges) and the min-of-N-lists loop
// (unionSortedRanges) have different edge cases than a plain pairwise merge.
func TestRecSetShardUnionIntersectThreeWay(t *testing.T) {
	universe := uint32(5000)
	wantA := wantFromRange(universe, [][2]uint32{{0, 3000}})
	wantB := wantFromRange(universe, [][2]uint32{{1000, 3000}})
	wantC := wantFromRange(universe, [][2]uint32{{2000, 2000}})
	partA := buildRecSetShard(wantA)
	partB := buildRecSetShard(wantB)
	partC := buildRecSetShard(wantC)

	wantUnion := make([]bool, universe)
	wantIntersect := make([]bool, universe)
	for i := range wantUnion {
		wantUnion[i] = wantA[i] || wantB[i] || wantC[i]
		wantIntersect[i] = wantA[i] && wantB[i] && wantC[i]
	}
	gotUnion := unionRecSetShards(nil, []*recSetShard{&partA, &partB, &partC})
	verifyRecSetShard(t, "3way-union", gotUnion, wantUnion)
	gotIntersect := intersectRecSetShards(nil, []*recSetShard{&partA, &partB, &partC})
	verifyRecSetShard(t, "3way-intersect", gotIntersect, wantIntersect)
}

// TestCombineTwoRecSetShardsDispatchesOptimizedPath proves the dedicated
// pairwise algorithm actually ran for each (kind,kind) combination that has
// one — not just that the result is correct, which the generic bitmap
// fallback (combineRecSetBitmapsWithData) would also produce. The
// discriminator: for every pairing below, the dedicated algorithm's own
// result kind is ranges or positive (never bitmap) whenever the true
// intersect/union is small and non-empty, while the fallback always returns
// bitmap-kind for any non-empty result (recSetShardFromBitmap only
// downgrades to ranges on an empty result) — so kind==recSetBitmap here
// would mean the optimized path was skipped.
func TestCombineTwoRecSetShardsDispatchesOptimizedPath(t *testing.T) {
	universe := uint32(2000)

	rangesWant := wantFromRange(universe, [][2]uint32{{100, 300}})                                          // -> recSetRanges
	positiveWant := wantFromRange(universe, [][2]uint32{{50, 1}, {500, 1}, {900, 1}, {1500, 1}, {1900, 1}}) // 5 singletons -> recSetPositive
	bitmapWant := func() []bool {
		w := make([]bool, universe)
		rng := rand.New(rand.NewSource(7))
		for i := range w {
			if i >= 100 && i < 400 {
				continue // keep ranges' own [100,400) span clear (below) so the
				// ranges/bitmap intersect stays small enough to fit the
				// dedicated path's budget instead of genuinely overflowing —
				// a ~50% random fill *inside* that span would legitimately
				// fragment past budget and isn't what this test is checking
			}
			w[i] = rng.Intn(2) == 0 // ~50% dense scattered elsewhere -> recSetBitmap
		}
		for i := 150; i < 160; i++ {
			w[i] = true // one small clustered block inside ranges' span
		}
		return w
	}()

	ranges := buildRecSetShard(rangesWant)
	positive := buildRecSetShard(positiveWant)
	bitmap := buildRecSetShard(bitmapWant)
	if ranges.kind != recSetRanges || positive.kind != recSetPositive || bitmap.kind != recSetBitmap {
		t.Fatalf("fixture kinds = %v/%v/%v, want ranges/positive/bitmap", ranges.kind, positive.kind, bitmap.kind)
	}

	cases := []struct {
		name              string
		a, b              *recSetShard
		wantA             []bool
		wantB             []bool
		wantNonBitmapKind bool // false only where bitmap output is architecturally correct, not a fallback signal
	}{
		{"ranges_ranges", &ranges, &ranges, rangesWant, rangesWant, true},
		{"positive_positive", &positive, &positive, positiveWant, positiveWant, true},
		{"ranges_positive", &ranges, &positive, rangesWant, positiveWant, true},
		{"ranges_bitmap", &ranges, &bitmap, rangesWant, bitmapWant, false}, // union must stay non-bitmap only for intersect; checked separately below
		{"positive_bitmap", &positive, &bitmap, positiveWant, bitmapWant, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name+"_intersect", func(t *testing.T) {
			want := make([]bool, universe)
			for i := range want {
				want[i] = tc.wantA[i] && tc.wantB[i]
			}
			got := intersectRecSetShards(nil, []*recSetShard{tc.a, tc.b})
			verifyRecSetShard(t, tc.name+"_intersect", got, want)
			// Intersect with a compact (ranges/positive) operand is always
			// bounded by that operand's own size, so it must never overflow
			// to bitmap for these small fixtures regardless of which side
			// is bitmap-kind.
			if got.count > 0 && got.kind == recSetBitmap {
				t.Errorf("%s: intersect kind = recSetBitmap, want ranges/positive (fallback path ran instead of the dedicated one)", tc.name)
			}
		})
		t.Run(tc.name+"_union", func(t *testing.T) {
			want := make([]bool, universe)
			for i := range want {
				want[i] = tc.wantA[i] || tc.wantB[i]
			}
			got := unionRecSetShards(nil, []*recSetShard{tc.a, tc.b})
			verifyRecSetShard(t, tc.name+"_union", got, want)
			if tc.wantNonBitmapKind && got.count > 0 && got.kind == recSetBitmap {
				t.Errorf("%s: union kind = recSetBitmap, want ranges/positive (fallback path ran instead of the dedicated one)", tc.name)
			}
		})
	}
}

func TestDifferenceRecSetShardsAcrossRepresentations(t *testing.T) {
	universe := uint32(2048)
	rangesWant := wantFromRange(universe, [][2]uint32{{100, 700}, {1200, 400}})
	positiveWant := wantFromRange(universe, [][2]uint32{{50, 1}, {100, 1}, {333, 1}, {1250, 1}, {1900, 1}})
	bitmapWant := make([]bool, universe)
	rng := rand.New(rand.NewSource(23))
	for i := range bitmapWant {
		bitmapWant[i] = rng.Intn(2) == 0
	}

	fixtures := []struct {
		name string
		part recSetShard
		want []bool
	}{
		{"ranges", buildRecSetShard(rangesWant), rangesWant},
		{"positive", buildRecSetShard(positiveWant), positiveWant},
		{"bitmap", buildRecSetShard(bitmapWant), bitmapWant},
	}
	for _, left := range fixtures {
		for _, right := range fixtures {
			left, right := left, right
			t.Run(left.name+"_minus_"+right.name, func(t *testing.T) {
				want := make([]bool, universe)
				for i := range want {
					want[i] = left.want[i] && !right.want[i]
				}
				got := differenceRecSetShards(nil, &left.part, &right.part)
				verifyRecSetShard(t, "difference", got, want)
			})
		}
	}
}

// TestNewRecSetShardFromSortedIDs checks the project-join builder path
// (pre-sorted external id list, not an ascending add() scan).
func TestNewRecSetShardFromSortedIDs(t *testing.T) {
	universe := uint32(100000)
	cases := []struct {
		name string
		ids  []uint32
	}{
		{"empty", nil},
		{"contiguous", func() []uint32 {
			ids := make([]uint32, 5000)
			for i := range ids {
				ids[i] = uint32(1000 + i)
			}
			return ids
		}()},
		{"scattered", func() []uint32 {
			rng := rand.New(rand.NewSource(4))
			seen := map[uint32]bool{}
			var ids []uint32
			for len(ids) < 200 {
				id := uint32(rng.Intn(int(universe)))
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
			sortUint32(ids)
			return ids
		}()},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			part := newRecSetShardFromSortedIDs(nil, universe, tc.ids)
			want := make([]bool, universe)
			for _, id := range tc.ids {
				want[id] = true
			}
			verifyRecSetShard(t, tc.name, part, want)
		})
	}
}

// TestRecSetShardBuilderFuzz drives the builder over many random universe
// sizes, densities, and clustering factors, including sizes chosen to land
// exactly on/near the ranges/positive/bitmap budget boundaries, since
// off-by-one errors in an online run-length builder concentrate there.
func TestRecSetShardBuilderFuzz(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for iter := 0; iter < 500; iter++ {
		universe := uint32(1 + rng.Intn(2000))
		want := make([]bool, universe)
		switch rng.Intn(4) {
		case 0: // sparse scattered
			density := 1 + rng.Intn(50)
			for i := range want {
				want[i] = rng.Intn(density) == 0
			}
		case 1: // clustered runs
			nRuns := rng.Intn(20)
			for r := 0; r < nRuns; r++ {
				base := uint32(rng.Intn(int(universe)))
				length := uint32(rng.Intn(int(universe)/4 + 1))
				for i := base; i < base+length && i < universe; i++ {
					want[i] = true
				}
			}
		case 2: // dense with scattered holes
			for i := range want {
				want[i] = true
			}
			nHoles := rng.Intn(int(universe))
			for h := 0; h < nHoles; h++ {
				want[rng.Intn(int(universe))] = false
			}
		case 3: // all-or-nothing / boundary-ish
			if rng.Intn(2) == 0 {
				for i := range want {
					want[i] = true
				}
			}
		}
		part := buildRecSetShard(want)
		verifyRecSetShard(t, "fuzz", part, want)
	}
}

// buildForEachVisibleRunShard constructs a minimal storageShard with the
// given main row count, delta (insert) row count, and deletion set, for
// testing forEachVisibleRun's run-splitting in isolation from the rest of
// the scan machinery.
func buildForEachVisibleRunShard(mainCount uint32, deltaCount int, deleted map[uint32]bool) *storageShard {
	shard := &storageShard{
		t:            &table{Columns: []*column{{Name: "x", Typ: "int"}}},
		columns:      map[string]ColumnStorage{},
		deltaColumns: make(map[string]int),
		main_count:   mainCount,
		inserts:      make([][]scm.Scmer, deltaCount),
	}
	shard.deletions.Reset()
	for idx := range deleted {
		shard.deletions.Set(uint(idx), true)
	}
	return shard
}

// TestForEachVisibleRun is the regression test for a real bug found while
// wiring recset.go's scan/project sites to bulk column reads: when a
// recSetRange run lands entirely in the delta region (base >= mainCount),
// clamping mainEnd to mainCount (without also clamping it up to base) made
// the delta loop start at mainCount instead of base, reprocessing indices
// that were never part of the run — one specific instance of which was
// caught by tests/execution/operators/recset.yaml returning double the
// expected row count. This test exercises the run-splitting directly rather
// than only through that one SQL scenario.
func TestForEachVisibleRun(t *testing.T) {
	cases := []struct {
		name        string
		mainCount   uint32
		deltaCount  int
		deleted     map[uint32]bool
		rangeInput  [][2]uint32 // recSetRanges pairs to build part from
		wantVisible []bool      // indexed 0..mainCount+deltaCount-1
	}{
		{
			name:        "pure main run",
			mainCount:   100,
			deltaCount:  0,
			rangeInput:  [][2]uint32{{10, 20}},
			wantVisible: wantFromRange(100, [][2]uint32{{10, 20}}),
		},
		{
			name:        "pure delta run",
			mainCount:   50,
			deltaCount:  50,
			rangeInput:  [][2]uint32{{60, 10}},
			wantVisible: wantFromRange(100, [][2]uint32{{60, 10}}),
		},
		{
			name:        "run spanning the mainCount boundary",
			mainCount:   50,
			deltaCount:  50,
			rangeInput:  [][2]uint32{{45, 10}}, // 45..54, straddles 50
			wantVisible: wantFromRange(100, [][2]uint32{{45, 10}}),
		},
		{
			name:       "delta run with a deletion hole",
			mainCount:  50,
			deltaCount: 50,
			deleted:    map[uint32]bool{65: true},
			rangeInput: [][2]uint32{{60, 20}}, // 60..79, minus 65
			wantVisible: func() []bool {
				w := wantFromRange(100, [][2]uint32{{60, 20}})
				w[65] = false
				return w
			}(),
		},
		{
			name:       "main run with a deletion hole right at the boundary",
			mainCount:  50,
			deltaCount: 50,
			deleted:    map[uint32]bool{49: true},
			rangeInput: [][2]uint32{{45, 10}}, // 45..54, minus 49
			wantVisible: func() []bool {
				w := wantFromRange(100, [][2]uint32{{45, 10}})
				w[49] = false
				return w
			}(),
		},
		{
			name:        "multiple ranges, one purely main, one purely delta",
			mainCount:   50,
			deltaCount:  50,
			rangeInput:  [][2]uint32{{5, 5}, {70, 5}},
			wantVisible: wantFromRange(100, [][2]uint32{{5, 5}, {70, 5}}),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			shard := buildForEachVisibleRunShard(tc.mainCount, tc.deltaCount, tc.deleted)
			universe := tc.mainCount + uint32(tc.deltaCount)
			part := recSetShardFromRangePairs(shard, universe, flattenPairs(tc.rangeInput), uint32(len(tc.rangeInput)))

			var got []uint32
			shard.forEachVisibleRun(&part, universe, false, nil, func(base, count uint32) bool {
				for id := base; id < base+count; id++ {
					got = append(got, id)
				}
				return true
			})
			checkIDsMatchWant(t, tc.name, got, tc.wantVisible)
		})
	}
}

func flattenPairs(ranges [][2]uint32) []uint32 {
	flat := make([]uint32, 0, len(ranges)*2)
	for _, r := range ranges {
		flat = append(flat, r[0], r[1])
	}
	return flat
}

// TestRecSetShardResultIsRightSized guards against a real bug found while
// benchmarking: finish() (and the union/intersect result builders) used to
// return `part.data` as the full (universe+31)/32-word backing array they
// build into, even when the logical content (ranges/positive) only used a
// tiny prefix of it — silently keeping the entire bitmap-worst-case buffer
// alive for the shard's whole lifetime regardless of how compact its
// content actually was. For the case ranges exists to compress (one big
// range spanning a huge universe), that defeated the entire memory point of
// having a compact representation. This checks that a shard's data slice
// length matches its logical content, not the universe-sized budget it was
// built from.
func TestRecSetShardResultIsRightSized(t *testing.T) {
	universe := uint32(500000)
	bitmapWords := int((universe + 31) / 32)

	clustered := buildRecSetShard(wantFromRange(universe, [][2]uint32{{100000, 200000}}))
	if clustered.kind != recSetRanges {
		t.Fatalf("expected recSetRanges, got %v", clustered.kind)
	}
	if got, want := len(clustered.data), 2; got != want {
		t.Errorf("single big range: len(data) = %d words, want %d (got a %dx bigger buffer than needed)", got, want, got/want)
	}

	scattered := buildRecSetShard(func() []bool {
		w := make([]bool, universe)
		for _, id := range []uint32{10, universe / 4, universe / 2, 3 * universe / 4, universe - 10} {
			w[id] = true
		}
		return w
	}())
	if scattered.kind != recSetPositive {
		t.Fatalf("expected recSetPositive, got %v", scattered.kind)
	}
	if got, want := len(scattered.data), 5; got != want {
		t.Errorf("5 scattered hits: len(data) = %d words, want %d", got, want)
	}

	// Union/intersect result builders had the identical bug — check those
	// too, not just the builder's own finish().
	a := buildRecSetShard(wantFromRange(universe, [][2]uint32{{0, 100000}}))
	b := buildRecSetShard(wantFromRange(universe, [][2]uint32{{50000, 100000}}))
	union := unionRecSetShards(nil, []*recSetShard{&a, &b})
	if got, want := len(union.data), 2; got != want {
		t.Errorf("union of two overlapping ranges: len(data) = %d words, want %d", got, want)
	}
	intersect := intersectRecSetShards(nil, []*recSetShard{&a, &b})
	if got, want := len(intersect.data), 2; got != want {
		t.Errorf("intersect of two overlapping ranges: len(data) = %d words, want %d", got, want)
	}

	if bitmapWords < 1000 {
		t.Fatalf("test setup: universe too small to make the point (bitmap budget = %d words)", bitmapWords)
	}
}

func sortUint32(ids []uint32) {
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j-1] > ids[j]; j-- {
			ids[j-1], ids[j] = ids[j], ids[j-1]
		}
	}
}

// --- Benchmarks: dedicated pairwise combine vs. full-universe fallback ----
//
// These isolate the combine step itself (combineTwoRecSetShards vs.
// combineRecSetBitmapsWithData) from everything else a real query pays for
// (index probe, scan, filter predicate) — an end-to-end SQL benchmark is
// dominated by scan cost and can't show this difference cleanly. Both sides
// call the exact functions unionRecSetShards/intersectRecSetShards actually
// dispatch to; FullMaterializeFallback is what every mixed-kind combine did
// unconditionally before this file's pairwise dispatch matrix existed.

func benchmarkFixtures(universe uint32) (rangesPart, positivePart, bitmapPart recSetShard) {
	rangesPart = buildRecSetShard(wantFromRange(universe, [][2]uint32{{universe / 2, 2000}}))
	positiveWant := make([]bool, universe)
	rng := rand.New(rand.NewSource(11))
	for i := 0; i < 100; i++ {
		positiveWant[rng.Intn(int(universe))] = true
	}
	positivePart = buildRecSetShard(positiveWant)
	bitmapWant := make([]bool, universe)
	for i := range bitmapWant {
		bitmapWant[i] = rng.Intn(3) == 0
	}
	bitmapPart = buildRecSetShard(bitmapWant)
	if rangesPart.kind != recSetRanges || positivePart.kind != recSetPositive || bitmapPart.kind != recSetBitmap {
		panic("benchmark fixture kinds drifted (ranges/positive/bitmap) — adjust the generators above")
	}
	return
}

func BenchmarkRecSetShardCombine(b *testing.B) {
	universe := uint32(5_000_000)
	ranges, positive, bitmap := benchmarkFixtures(universe)

	cases := []struct {
		name string
		a, b *recSetShard
	}{
		{"RangesVsBitmap", &ranges, &bitmap},
		{"RangesVsPositive", &ranges, &positive},
	}
	for _, tc := range cases {
		b.Run(tc.name+"/Intersect/DedicatedPath", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				intersectRecSetShards(nil, []*recSetShard{tc.a, tc.b})
			}
		})
		b.Run(tc.name+"/Intersect/FullMaterializeFallback", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				data := make([]uint32, (universe+31)/32)
				combineRecSetBitmapsWithData(nil, universe, []*recSetShard{tc.a, tc.b}, false, data)
			}
		})
		b.Run(tc.name+"/Union/DedicatedPath", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				unionRecSetShards(nil, []*recSetShard{tc.a, tc.b})
			}
		})
		b.Run(tc.name+"/Union/FullMaterializeFallback", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				data := make([]uint32, (universe+31)/32)
				combineRecSetBitmapsWithData(nil, universe, []*recSetShard{tc.a, tc.b}, true, data)
			}
		})
	}
}
