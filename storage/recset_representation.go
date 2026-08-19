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

// Three kinds. There is deliberately no "full" or "empty" tag: a shard that
// matches everything is recSetRanges with one pair covering [0,universe), and
// a shard that matches nothing is recSetRanges with zero pairs — both fall
// out of the general ranges logic with no special-casing, and (unlike the
// former recSetFull) can never accidentally read past universe or absorb
// rows a caller forgot to bound-check.
//
// There is also deliberately no "negative" (exclusion-list) tag anymore.
// recSetRanges already represents "everything except a few scattered holes"
// compactly — each hole just ends one run and starts the next, so a handful
// of exceptions costs a handful of range pairs, not a bitmap. A negative tag
// would only additionally help "everything except MANY scattered individual
// exceptions", which is rare in practice, and its bulk/iteration form is the
// *complement* of what's stored — derived, stateful, and a real source of
// off-by-one bugs at shard/universe boundaries. That pattern now falls
// through to recSetBitmap once ranges fragments past budget, which is
// simpler and always correct.
const (
	recSetRanges recSetRepresentation = iota
	recSetPositive
	recSetBitmap
)

// recSetShardBuilder builds a recSetShard incrementally in ascending recid
// order (see the add() callers in recset.go / analyzer.go — all drive it via
// a single forward scan).
//
// DESIGN INVARIANT (build phase): exactly one allocation for the shared
// backing array while add() is driving normal (non-escalating) progress.
// newRecSetShardBuilder allocates the eventual bitmap footprint
// ((universe+31)/32 words) once; every phase (ranges, positive, bitmap)
// reuses that same backing array — ranges as (base,count) pairs, positive as
// a flat hit-id list, bitmap as literal bits.
//
// HARDER INVARIANT, and the one that actually matters: a caller's filter
// predicate (the recid→hit decision fed into add()) must NEVER be evaluated
// more than once per recid. add() is driven from a live SQL scan — the hit
// value the caller passes in already paid for an index lookup and a
// predicate evaluation, both of which can be arbitrarily expensive
// (interpreted Scheme, column reads). An earlier version of this file
// escalated phases with a bare clear() of the backing array and left the
// caller to "replay" the abandoned prefix by re-running its scan and
// predicate a second time — this was wrong: it silently doubled scan+filter
// cost on any shard that escalates late, and is exactly what must never
// happen. Every escalation in this file is therefore self-contained instead:
// escalateRangesToBitmap and escalatePositiveToBitmap each copy the
// already-collected content (which already reflects every filter result
// seen so far) into a small temporary buffer, clear the shared backing
// array, and replay *from that copy* — never by asking the caller to
// re-decide anything. This temporary buffer is the one place a second
// allocation is deliberately permitted (bounded by the same budget as
// everything else here, at most once per escalation); the shared backing
// array itself is still allocated exactly once. Because escalation no longer
// needs a caller-visible replay window, callers (recset.go, analyzer.go)
// drive add() in one straight forward pass and never re-run their scan.
//
// finish() itself adds exactly one more allocation for ranges/positive
// results (not bitmap): it copies the logically-used prefix of the backing
// array down to a right-sized slice before returning. Without that, a
// recSetShard would keep the entire (universe+31)/32-word worst-case buffer
// alive for its whole lifetime regardless of how compact its actual
// content is — for the case ranges exists to compress (one big range
// spanning a huge universe, down to 2 words), that would silently defeat
// the entire memory point of the representation. This copy is bounded by
// the logical size actually recorded (2×rangePairs or hits words), not by
// universe, and paid exactly once.
//
// tryConvertToPositive (ranges→positive, triggered by too many isolated
// singleton runs) needs the same kind of temporary buffer for a different
// reason: flattening (base,count) pairs into individual ids is not a pure
// clear+rebuild, and a naive in-place left-to-right expansion is unsafe —
// any range with count>1 writes ahead of where a not-yet-read later pair
// lives, corrupting it before it's read (a range with count=1000 at the
// front, for instance, overwrites the read position of a pair many slots
// further in before that pair has been read).
type recSetShardBuilder struct {
	shard       *storageShard
	universe    uint32
	data        []uint32
	matched     uint32
	initialized bool

	// breakAt, when nonzero, forces the ranges phase to end whatever run is
	// open the moment recid reaches it, even if the hit is otherwise
	// contiguous with the run — see addRange. Callers set this to a shard's
	// main_count so a stored range pair never straddles the main/delta
	// boundary, letting consumers (forEachVisibleRun and friends) split a
	// range into its main-storage and delta sub-runs with a plain Go slice
	// bound instead of clamping/re-deriving the split point themselves.
	breakAt uint32

	bitmap   bool // once true, add() operates in bitmap mode
	positive bool // once true (and !bitmap), add() operates in flat hit-list mode

	// ranges-phase state
	rangePairs    uint32
	singletonRuns uint32
	runStart      uint32
	runLen        uint32
	haveRun       bool

	// positive-phase state
	hits uint32
}

// maxSingletonRuns is how many isolated (count==1) hit-ranges the ranges
// phase tolerates before concluding the data isn't clustering and switching
// to a flat hit list instead.
const maxSingletonRuns = 3

// breakAt forces the ranges phase to close a run the moment recid reaches
// it (see recSetShardBuilder.breakAt) — pass a shard's main_count so ranges
// never straddle the main/delta boundary, or 0 to disable (no meaningful
// main/delta split, e.g. the SkipList builder in analyzer.go).
func newRecSetShardBuilder(shard *storageShard, universe uint32, allowFull bool, breakAt uint32) *recSetShardBuilder {
	builder := &recSetShardBuilder{
		shard:    shard,
		universe: universe,
		data:     make([]uint32, (universe+31)/32),
		breakAt:  breakAt,
	}
	if !allowFull {
		builder.initialized = true
	}
	return builder
}

func (b *recSetShardBuilder) add(recid uint32, hit bool) {
	b.initialized = true
	if hit {
		b.matched++
	}
	if b.bitmap {
		b.addBitmap(recid, hit)
		return
	}
	if b.positive {
		b.addPositive(recid, hit)
		return
	}
	b.addRange(recid, hit)
}

func (b *recSetShardBuilder) addBitmap(recid uint32, hit bool) {
	if hit {
		b.data[recid>>5] |= uint32(1) << (recid & 31)
	}
}

// addPositive appends a hit recid to the flat list. Overflowing the shared
// backing array's budget escalates to bitmap self-contained (see
// escalatePositiveToBitmap): the already-collected hits are converted from a
// temp copy, never re-derived by asking the caller to re-run its filter
// predicate — see recSetShardBuilder's doc comment for why that's a hard
// requirement, not just an optimization.
func (b *recSetShardBuilder) addPositive(recid uint32, hit bool) {
	if !hit {
		return
	}
	if b.hits >= uint32(len(b.data)) {
		b.escalatePositiveToBitmap()
		b.addBitmap(recid, true)
		return
	}
	b.data[b.hits] = recid
	b.hits++
}

// escalatePositiveToBitmap converts the hit ids already collected into
// bitmap bits before clearing the backing array, exactly like
// escalateRangesToBitmap does for the ranges phase — see there for why this
// must never be a bare clear()-and-rely-on-replay.
func (b *recSetShardBuilder) escalatePositiveToBitmap() {
	oldHits := append([]uint32(nil), b.data[:b.hits]...)
	clear(b.data)
	b.bitmap = true
	for _, id := range oldHits {
		b.addBitmap(id, true)
	}
}

func (b *recSetShardBuilder) addRange(recid uint32, hit bool) {
	if hit {
		if b.haveRun && recid == b.runStart+b.runLen && recid != b.breakAt {
			b.runLen++
			return
		}
		if b.haveRun {
			b.closeRun()
		}
		if b.bitmap {
			b.addBitmap(recid, true)
			return
		}
		if b.positive {
			b.addPositive(recid, true)
			return
		}
		// Reserve this new run's pair slot right now, while still inside a
		// real add() call — never deferred to closeRun()/finish(). Escalation
		// here is self-contained (escalateRangesToBitmap converts the pairs
		// already recorded from a temp copy), so it needs no replay window
		// from the caller — see recSetShardBuilder's doc comment for why a
		// filter predicate must never be re-run to reconstruct lost state.
		if (b.rangePairs+1)*2 > uint32(len(b.data)) {
			b.escalateRangesToBitmap()
			b.addBitmap(recid, true)
			return
		}
		b.rangePairs++
		b.runStart = recid
		b.runLen = 1
		b.haveRun = true
		return
	}
	if b.haveRun {
		b.closeRun()
	}
}

// closeRun flushes the currently open hit-run into its already-reserved
// pair slot (reserved by addRange when the run started — see there) and
// checks the one escalation closeRun can still trigger: too many isolated
// singleton runs, meaning the data isn't clustering, so switch to a flat
// hit list instead (tryConvertToPositive). Safe to run from finish() (i.e.
// after the caller's add() loop has already ended) because every escalation
// this file performs is self-contained — see recSetShardBuilder's doc
// comment.
func (b *recSetShardBuilder) closeRun() {
	if !b.haveRun {
		return
	}
	base, count := b.runStart, b.runLen
	b.haveRun = false

	b.data[(b.rangePairs-1)*2] = base
	b.data[(b.rangePairs-1)*2+1] = count
	if count == 1 {
		b.singletonRuns++
		if b.singletonRuns > maxSingletonRuns {
			b.tryConvertToPositive()
		}
	}
}

// tryConvertToPositive flattens the ranges accumulated so far into a flat
// hit-id list, using a small temporary buffer sized exactly to the total hit
// count — see recSetShardBuilder's doc comment for why every transition in
// this file (this one included) must convert what's already recorded rather
// than clear()+ask the caller to replay: a bare clear() here would silently
// drop every range pair recorded before this one, and closeRun() (this
// function's only caller) can run from finish(), after the caller's add()
// loop has already ended with nothing left to replay from even if that were
// acceptable. If the total hit count doesn't fit the shared backing array's
// budget either, escalateRangesToBitmap keeps that fallback self-contained
// the same way.
func (b *recSetShardBuilder) tryConvertToPositive() {
	var total uint32
	for i := uint32(0); i < b.rangePairs; i++ {
		total += b.data[i*2+1]
	}
	if total > uint32(len(b.data)) {
		b.escalateRangesToBitmap()
		return
	}
	flat := make([]uint32, total)
	var write uint32
	for i := uint32(0); i < b.rangePairs; i++ {
		base, count := b.data[i*2], b.data[i*2+1]
		for k := uint32(0); k < count; k++ {
			flat[write] = base + k
			write++
		}
	}
	copy(b.data, flat)
	b.hits = write
	b.positive = true
	b.rangePairs = 0
	b.singletonRuns = 0
}

// escalateRangesToBitmap converts the range pairs already recorded into
// bitmap bits before clearing the backing array, so nothing is lost and the
// caller's filter predicate never needs to be re-run — called from both
// addRange (direct ranges→bitmap escalation) and tryConvertToPositive (its
// own overflow-to-bitmap fallback).
func (b *recSetShardBuilder) escalateRangesToBitmap() {
	oldRanges := append([]uint32(nil), b.data[:b.rangePairs*2]...)
	clear(b.data)
	b.bitmap = true
	for i := 0; i < len(oldRanges); i += 2 {
		base, count := oldRanges[i], oldRanges[i+1]
		for id := base; id < base+count; id++ {
			b.addBitmap(id, true)
		}
	}
}

// finish returns the built recSetShard. For bitmap, the full backing array
// is genuinely all in use — returned as-is, no copy. For ranges/positive,
// the whole point of staying in those phases is that the logical content is
// (for the intended clustered/sparse cases) far smaller than the
// (universe+31)/32-word backing array reserved for the bitmap worst case —
// a single big range is 2 words no matter how large universe is. Returning
// `b.data` directly there would silently keep that whole worst-case buffer
// alive for the shard's entire (query-local, but potentially
// large-universe) lifetime, defeating the memory point of having a compact
// representation at all. So finish() copies down to a right-sized array
// here — one more allocation, but bounded by the logical size actually
// recorded (2*rangePairs or hits words), not by universe, and paid exactly
// once, after the fact, never repeated or proportional to escalation
// attempts the way the mid-build allocations discussed above are.
func (b *recSetShardBuilder) finish() recSetShard {
	if !b.bitmap && !b.positive {
		b.closeRun()
	}
	part := recSetShard{shard: b.shard, universe: b.universe, count: int64(b.matched)}
	if b.matched == 0 {
		part.kind = recSetRanges
		return part
	}
	if b.bitmap {
		part.data = b.data
		part.kind = recSetBitmap
		return part
	}
	if b.positive {
		part.used = b.hits
		part.kind = recSetPositive
		part.data = append([]uint32(nil), b.data[:part.used]...)
		sort.Slice(part.data, func(i, j int) bool { return part.data[i] < part.data[j] })
		return part
	}
	part.used = b.rangePairs
	part.kind = recSetRanges
	part.data = append([]uint32(nil), b.data[:2*part.used]...)
	return part
}

// newRecSetShardFromSortedIDs builds a recSetShard from an already-sorted,
// deduplicated recid list (used by project-join, where ids are collected via
// index probes rather than a single ascending scan). It picks whichever of
// ranges/positive/bitmap ends up smallest, coalescing adjacent/consecutive
// ids into ranges first since that's virtually free given the input is
// already sorted.
func newRecSetShardFromSortedIDs(shard *storageShard, universe uint32, ids []uint32) recSetShard {
	part := recSetShard{shard: shard, universe: universe, count: int64(len(ids))}
	if len(ids) == 0 {
		part.kind = recSetRanges
		return part
	}
	budgetWords := (universe + 31) / 32

	// Try ranges: coalesce consecutive ids into pairs.
	rangePairs := make([]uint32, 0, 2*len(ids))
	i := 0
	for i < len(ids) {
		base := ids[i]
		j := i + 1
		for j < len(ids) && ids[j] == ids[j-1]+1 {
			j++
		}
		rangePairs = append(rangePairs, base, ids[j-1]-base+1)
		i = j
	}
	if uint32(len(rangePairs)) <= budgetWords {
		part.data = rangePairs
		part.used = uint32(len(rangePairs)) / 2
		part.kind = recSetRanges
		return part
	}
	if uint32(len(ids)) <= budgetWords {
		part.data = append([]uint32(nil), ids...)
		part.used = uint32(len(ids))
		part.kind = recSetPositive
		return part
	}
	data := make([]uint32, budgetWords)
	for _, id := range ids {
		data[id>>5] |= uint32(1) << (id & 31)
	}
	part.data = data
	part.kind = recSetBitmap
	return part
}

func sortedUint32Contains(values []uint32, value uint32) bool {
	pos := sort.Search(len(values), func(i int) bool { return values[i] >= value })
	return pos < len(values) && values[pos] == value
}

// listedValues returns the flat hit-id list (recSetPositive only). Always
// sorted ascending, deduplicated — see recSetShard's INVARIANT doc comment
// in recset.go. Every pairwise combine below that consumes this (union/
// intersect with another recSetPositive, recSetRanges, or recSetBitmap
// operand) relies on that ordering to run as a linear sweep.
func (s *recSetShard) listedValues() []uint32 {
	return s.data[:s.used]
}

// listedRanges returns the flat [base0,count0,base1,count1,...] pair array
// (recSetRanges only). Always sorted ascending by base, non-overlapping and
// non-adjacent (the builder always coalesces touching runs) — see
// recSetShard's INVARIANT doc comment in recset.go. Every pairwise combine
// below that consumes this relies on that ordering to run as a linear
// sweep instead of a per-pair search.
func (s *recSetShard) listedRanges() []uint32 {
	return s.data[:2*s.used]
}

// rangesContains does a binary search over the sorted, non-overlapping range
// pairs for the one that might contain recid.
func (s *recSetShard) rangesContains(recid uint32) bool {
	ranges := s.listedRanges()
	n := len(ranges) / 2
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		if ranges[mid*2] <= recid {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return false
	}
	base, count := ranges[(lo-1)*2], ranges[(lo-1)*2+1]
	return recid < base+count
}

func unionRecSetShards(shard *storageShard, parts []*recSetShard) recSetShard {
	var universe uint32
	for _, part := range parts {
		if part != nil && part.universe > universe {
			universe = part.universe
		}
	}
	compact := parts[:0]
	mixedUniverse := false
	allRanges := true
	allPositive := true
	for _, part := range parts {
		if part == nil || part.count == 0 {
			continue
		}
		if part.count == int64(part.universe) && part.universe == universe {
			return recSetShardFromSingleFullRange(shard, universe)
		}
		mixedUniverse = mixedUniverse || part.universe != universe
		allRanges = allRanges && part.kind == recSetRanges
		allPositive = allPositive && part.kind == recSetPositive
		compact = append(compact, part)
	}
	parts = compact
	if len(parts) == 0 {
		return recSetShard{shard: shard, kind: recSetRanges, universe: universe}
	}
	if len(parts) == 1 {
		return cloneRecSetShardTo(shard, parts[0])
	}
	data := make([]uint32, (universe+31)/32)
	if mixedUniverse {
		return combineRecSetBitmapsWithData(shard, universe, parts, true, data)
	}
	if allRanges {
		lists := recSetRangeLists(parts)
		used, overflow := unionSortedRanges(data, lists)
		if !overflow {
			return recSetShardFromRangePairs(shard, universe, data, used)
		}
	} else if allPositive {
		positiveLists := recSetLists(parts, recSetPositive)
		used, overflow := unionSortedLists(data, positiveLists)
		if !overflow {
			return recSetShardFromList(shard, universe, data, used)
		}
	} else {
		// Mixed kinds: fold pairwise through combineTwoRecSetShards instead of
		// jumping straight to full-universe bitmap materialization — see its
		// doc comment for why every (kind,kind) pairing gets an algorithm
		// proportional to the more compact operand.
		return foldRecSetShards(shard, universe, true, parts)
	}
	// A native-representation N-way merge overflowed its budget (every part
	// was the same kind, but the combined result still didn't fit) — every
	// part still answers .contains()/word() correctly regardless of its own
	// kind, so full materialization is always correct here, just without the
	// compact-representation shortcut above.
	return combineRecSetBitmapsWithData(shard, universe, parts, true, data)
}

func intersectRecSetShards(shard *storageShard, parts []*recSetShard) recSetShard {
	if len(parts) == 0 {
		return recSetShard{shard: shard, kind: recSetRanges}
	}
	var universe uint32
	for _, part := range parts {
		if part == nil || part.count == 0 {
			return recSetShard{shard: shard, kind: recSetRanges, universe: universe}
		}
		if part.universe > universe {
			universe = part.universe
		}
	}
	compact := parts[:0]
	mixedUniverse := false
	allRanges := true
	for _, part := range parts {
		if part.count == int64(part.universe) && part.universe == universe {
			continue
		}
		mixedUniverse = mixedUniverse || part.universe != universe
		allRanges = allRanges && part.kind == recSetRanges
		compact = append(compact, part)
	}
	parts = compact
	if len(parts) == 0 {
		return recSetShardFromSingleFullRange(shard, universe)
	}
	if len(parts) == 1 {
		return cloneRecSetShardTo(shard, parts[0])
	}
	data := make([]uint32, (universe+31)/32)
	if mixedUniverse {
		return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
	}
	if allRanges {
		lists := recSetRangeLists(parts)
		used, overflow := intersectSortedRanges(data, lists)
		if !overflow {
			return recSetShardFromRangePairs(shard, universe, data, used)
		}
		// Same-kind N-way merge overflowed its budget: every part still
		// answers .contains()/word() correctly regardless of kind, so full
		// materialization is always correct here, just without the
		// compact-representation shortcut above.
		return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
	}
	// Mixed kinds: fold pairwise through combineTwoRecSetShards instead of
	// jumping straight to full-universe bitmap materialization — see its doc
	// comment for why every (kind,kind) pairing gets an algorithm
	// proportional to the more compact operand.
	return foldRecSetShards(shard, universe, false, parts)
}

func cloneRecSetShardTo(shard *storageShard, part *recSetShard) recSetShard {
	clone := *part
	clone.shard = shard
	return clone
}

func recSetShardFromSingleFullRange(shard *storageShard, universe uint32) recSetShard {
	if universe == 0 {
		return recSetShard{shard: shard, kind: recSetRanges, universe: universe}
	}
	return recSetShard{shard: shard, kind: recSetRanges, universe: universe, data: []uint32{0, universe}, used: 1, count: int64(universe)}
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

func recSetRangeLists(parts []*recSetShard) [][]uint32 {
	lists := make([][]uint32, len(parts))
	for i, part := range parts {
		lists[i] = part.listedRanges()
	}
	return lists
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

// unionSortedRanges merges multiple sorted, non-overlapping range-pair lists
// into one sorted, maximally-coalesced list of non-overlapping ranges,
// writing pairs into dst. Returns (pairs used, overflow).
func unionSortedRanges(dst []uint32, lists [][]uint32) (uint32, bool) {
	positions := make([]int, len(lists))
	var used uint32
	var curBase, curEnd uint32
	haveCur := false
	flush := func() bool {
		if !haveCur {
			return true
		}
		if (used+1)*2 > uint32(len(dst)) {
			return false
		}
		dst[used*2] = curBase
		dst[used*2+1] = curEnd - curBase
		used++
		return true
	}
	for {
		var nextBase, nextEnd uint32
		found := false
		srcIdx := -1
		for i, ranges := range lists {
			if positions[i] < len(ranges) {
				base := ranges[positions[i]]
				if !found || base < nextBase {
					nextBase = base
					nextEnd = base + ranges[positions[i]+1]
					found = true
					srcIdx = i
				}
			}
		}
		if !found {
			break
		}
		positions[srcIdx] += 2
		if haveCur && nextBase <= curEnd {
			if nextEnd > curEnd {
				curEnd = nextEnd
			}
			continue
		}
		if !flush() {
			return used, true
		}
		curBase, curEnd = nextBase, nextEnd
		haveCur = true
	}
	if !flush() {
		return used, true
	}
	return used, false
}

// intersectSortedRanges computes the N-way intersection of sorted,
// non-overlapping range-pair lists via a sweep: at each step the candidate
// interval is [max(current starts), min(current ends)); every list whose
// current interval ends there is advanced.
func intersectSortedRanges(dst []uint32, lists [][]uint32) (uint32, bool) {
	positions := make([]int, len(lists))
	var used uint32
	for {
		var maxStart, minEnd uint32
		first := true
		for i, ranges := range lists {
			if positions[i] >= len(ranges) {
				return used, false
			}
			base := ranges[positions[i]]
			end := base + ranges[positions[i]+1]
			if first {
				maxStart, minEnd = base, end
				first = false
				continue
			}
			if base > maxStart {
				maxStart = base
			}
			if end < minEnd {
				minEnd = end
			}
		}
		if maxStart < minEnd {
			if (used+1)*2 > uint32(len(dst)) {
				return used, true
			}
			dst[used*2] = maxStart
			dst[used*2+1] = minEnd - maxStart
			used++
		}
		advanced := false
		for i, ranges := range lists {
			end := ranges[positions[i]] + ranges[positions[i]+1]
			if end == minEnd {
				positions[i] += 2
				advanced = true
			}
		}
		if !advanced {
			return used, false
		}
	}
}

// --- Pairwise mixed-kind combine -------------------------------------------
//
// unionRecSetShards/intersectRecSetShards above have fast N-way paths for
// the uniform case (every part the same kind). When parts mix kinds, this
// section supplies a dedicated algorithm for every (kind,kind) pairing —
// ranges/positive/bitmap gives 3 kinds, so 6 unordered pairs per op — rather
// than falling back to combineRecSetBitmapsWithData's full-universe word
// materialization for anything not uniform. Only bitmap-bitmap is a genuine
// full-universe operation (there's no way to skip words when both sides are
// already fully expanded); every pairing involving at least one ranges or
// positive operand instead costs proportional to that operand's own
// compactness (its range-pair count or hit count), which is the entire
// point of not having flattened it to a bitmap in the first place.
//
// foldRecSetShards drives this pairwise, reducing a >2-part heterogeneous
// list left to right; each step still gets a kind-optimized algorithm
// because the dispatch in combineTwoRecSetShards is re-evaluated on the
// (possibly kind-changed) accumulator every time.
//
// Every pairing below is a linear sweep, never a per-element binary search
// or an O(n*m) scan, because both operands are guaranteed sorted ascending
// (see recSetShard's INVARIANT doc comment): ranges-ranges/positive-positive
// reuse the same sorted-merge primitives the uniform-kind N-way paths use;
// ranges-positive walks both sorted sequences in lockstep
// (intersectRangesWithList) or folds the positive side into the same sorted
// sweep as ranges (listAsRangePairs + unionSortedRanges); ranges-bitmap and
// positive-bitmap touch the bitmap only within the compact side's own spans
// (intersectRangesWithBitmap, setBitRange) or only at the compact side's own
// ids (intersectListWithBitmap, unionListWithBitmap) rather than the whole
// universe — sortedness isn't what bounds their cost (bitmap access is O(1)
// either way), but it's still what guarantees their output stays a valid
// ascending recSetPositive/recSetRanges instead of needing a re-sort after.
func foldRecSetShards(shard *storageShard, universe uint32, union bool, parts []*recSetShard) recSetShard {
	acc := parts[0]
	for _, part := range parts[1:] {
		combined := combineTwoRecSetShards(shard, universe, union, acc, part)
		acc = &combined
	}
	return cloneRecSetShardTo(shard, acc)
}

// combineTwoRecSetShards combines exactly two shards over the same universe
// (mixed-universe combines are handled by the caller before reaching here —
// see unionRecSetShards/intersectRecSetShards). left/right are read-only;
// the result always lives in a freshly allocated buffer.
func combineTwoRecSetShards(shard *storageShard, universe uint32, union bool, left, right *recSetShard) recSetShard {
	// Canonicalize so the dispatch below only needs one entry per unordered
	// pair — recSetRanges < recSetPositive < recSetBitmap by iota order.
	if left.kind > right.kind {
		left, right = right, left
	}
	data := make([]uint32, (universe+31)/32)

	switch {
	case left.kind == recSetRanges && right.kind == recSetRanges:
		lists := [][]uint32{left.listedRanges(), right.listedRanges()}
		var used uint32
		var overflow bool
		if union {
			used, overflow = unionSortedRanges(data, lists)
		} else {
			used, overflow = intersectSortedRanges(data, lists)
		}
		if !overflow {
			return recSetShardFromRangePairs(shard, universe, data, used)
		}

	case left.kind == recSetPositive && right.kind == recSetPositive:
		if union {
			used, overflow := unionSortedLists(data, [][]uint32{left.listedValues(), right.listedValues()})
			if !overflow {
				return recSetShardFromList(shard, universe, data, used)
			}
		} else {
			used := intersectSortedLists(data, left.listedValues(), right.listedValues())
			return recSetShardFromList(shard, universe, data, used)
		}

	case left.kind == recSetRanges && right.kind == recSetPositive:
		if union {
			synthetic := make([]uint32, 2*len(right.listedValues()))
			listAsRangePairs(synthetic, right.listedValues())
			used, overflow := unionSortedRanges(data, [][]uint32{left.listedRanges(), synthetic})
			if !overflow {
				return recSetShardFromRangePairs(shard, universe, data, used)
			}
		} else {
			used := intersectRangesWithList(data, left.listedRanges(), right.listedValues())
			return recSetShardFromList(shard, universe, data, used)
		}

	case left.kind == recSetRanges && right.kind == recSetBitmap:
		if union {
			result := unionRangesWithBitmap(data, left.listedRanges(), right.data)
			return recSetShardFromBitmap(shard, universe, result)
		}
		used, overflow := intersectRangesWithBitmap(data, left.listedRanges(), right.data)
		if !overflow {
			return recSetShardFromRangePairs(shard, universe, data, used)
		}

	case left.kind == recSetPositive && right.kind == recSetBitmap:
		if union {
			result := unionListWithBitmap(data, left.listedValues(), right.data)
			return recSetShardFromBitmap(shard, universe, result)
		}
		used := intersectListWithBitmap(data, left.listedValues(), right.data)
		return recSetShardFromList(shard, universe, data, used)

	default: // bitmap, bitmap
		if union {
			unionBitmaps(data, left.data, right.data)
		} else {
			intersectBitmaps(data, left.data, right.data)
		}
		return recSetShardFromBitmap(shard, universe, data)
	}
	// The dedicated path's own budget overflowed (only reachable for the
	// ranges-involving cases above — every other case returns unconditionally):
	// fall back to full materialization, still always correct.
	return combineRecSetBitmapsWithData(shard, universe, []*recSetShard{left, right}, union, data)
}

// intersectSortedLists computes the sorted intersection of two sorted,
// deduplicated recid lists via a two-pointer merge. The result can never
// exceed min(len(a),len(b)), which is itself bounded by the shared
// recSetPositive budget both a and b were built under, so unlike the
// range-producing combines above this can never overflow dst.
func intersectSortedLists(dst []uint32, a, b []uint32) uint32 {
	var used, i, j uint32
	for i < uint32(len(a)) && j < uint32(len(b)) {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			dst[used] = a[i]
			used++
			i++
			j++
		}
	}
	return used
}

// listAsRangePairs writes values (a sorted recid list) into dst as
// synthetic singleton (id,1) range pairs, so a recSetPositive operand can be
// fed through the same unionSortedRanges sweep a recSetRanges operand uses.
// dst must be 2*len(values) long.
func listAsRangePairs(dst []uint32, values []uint32) {
	for i, id := range values {
		dst[i*2] = id
		dst[i*2+1] = 1
	}
}

// intersectRangesWithList keeps every value that falls inside one of ranges'
// pairs, via a two-pointer sweep over both (already sorted, non-overlapping)
// inputs. The result is a subsequence of values, so it's already sorted and
// bounded by len(values) — never overflows dst.
func intersectRangesWithList(dst []uint32, ranges, values []uint32) uint32 {
	var used uint32
	ri := 0
	for _, id := range values {
		for ri < len(ranges) && ranges[ri]+ranges[ri+1] <= id {
			ri += 2
		}
		if ri >= len(ranges) {
			break
		}
		if ranges[ri] <= id {
			dst[used] = id
			used++
		}
	}
	return used
}

// setBitRange sets bits [base,base+count) in a bitmap word array, touching
// only the words the range actually spans (whole interior words get a
// single OR with ^0, boundary words get a masked OR) instead of looping bit
// by bit — the primitive both unionRangesWithBitmap and unionListWithBitmap
// (for count==1) are built on.
func setBitRange(data []uint32, base, count uint32) {
	if count == 0 {
		return
	}
	end := base + count
	for pos, wordIdx := base, base>>5; pos < end; wordIdx++ {
		wordStart := wordIdx << 5
		lo := pos - wordStart
		hi := uint32(32)
		if wordStart+32 > end {
			hi = end - wordStart
		}
		var mask uint32
		if hi-lo >= 32 {
			mask = ^uint32(0)
		} else {
			mask = ((uint32(1) << (hi - lo)) - 1) << lo
		}
		if int(wordIdx) < len(data) {
			data[wordIdx] |= mask
		}
		pos = wordStart + hi
	}
}

// unionRangesWithBitmap ORs ranges' pairs into a clone of bitmapData. The
// result is necessarily bitmap-sized (it must preserve every one-bit of
// bitmapData, which may be scattered across the whole universe), so unlike
// the ranges-producing combines above this never overflows — dst is exactly
// len(bitmapData) long.
func unionRangesWithBitmap(dst []uint32, ranges []uint32, bitmapData []uint32) []uint32 {
	copy(dst, bitmapData)
	for i := 0; i < len(ranges); i += 2 {
		setBitRange(dst, ranges[i], ranges[i+1])
	}
	return dst
}

// unionListWithBitmap ORs individual hit ids into a clone of bitmapData. See
// unionRangesWithBitmap — same reasoning, never overflows.
func unionListWithBitmap(dst []uint32, values []uint32, bitmapData []uint32) []uint32 {
	copy(dst, bitmapData)
	for _, id := range values {
		dst[id>>5] |= uint32(1) << (id & 31)
	}
	return dst
}

// intersectRangesWithBitmap visits only the bitmap words within ranges'
// spans (never the full universe) and coalesces the set bits found there
// into output range pairs. Unlike the union direction, this genuinely can
// overflow dst: a highly fragmented bitmap (e.g. alternating bits) within a
// large range span can produce far more runs than the range/positive budget
// allows, so the caller must still be ready to fall back to full
// materialization.
func intersectRangesWithBitmap(dst []uint32, ranges []uint32, bitmapData []uint32) (uint32, bool) {
	var used uint32
	for i := 0; i < len(ranges); i += 2 {
		base, count := ranges[i], ranges[i+1]
		end := base + count
		var runStart uint32
		haveRun := false
		flush := func(runEnd uint32) bool {
			if !haveRun {
				return true
			}
			if (used+1)*2 > uint32(len(dst)) {
				return false
			}
			dst[used*2] = runStart
			dst[used*2+1] = runEnd - runStart
			used++
			haveRun = false
			return true
		}
		for pos := base; pos < end; {
			wordIdx := pos >> 5
			wordStart := wordIdx << 5
			var word uint32
			if int(wordIdx) < len(bitmapData) {
				word = bitmapData[wordIdx]
			}
			hi := uint32(32)
			if wordStart+32 > end {
				hi = end - wordStart
			}
			for b := pos - wordStart; b < hi; b++ {
				id := wordStart + b
				if word&(uint32(1)<<b) != 0 {
					if !haveRun {
						runStart = id
						haveRun = true
					}
				} else if !flush(id) {
					return used, true
				}
			}
			pos = wordStart + hi
		}
		if !flush(end) {
			return used, true
		}
	}
	return used, false
}

// intersectListWithBitmap keeps every value whose bit is set in bitmapData.
// The result is a subsequence of values — already sorted, bounded by
// len(values), never overflows dst.
func intersectListWithBitmap(dst []uint32, values []uint32, bitmapData []uint32) uint32 {
	var used uint32
	for _, id := range values {
		word := id >> 5
		if int(word) < len(bitmapData) && bitmapData[word]&(uint32(1)<<(id&31)) != 0 {
			dst[used] = id
			used++
		}
	}
	return used
}

// unionBitmaps/intersectBitmaps are the direct word-parallel AND/OR for two
// genuinely bitmap-kind operands — the one pairing in this section that
// really does cost a full universe pass, since neither side has anything
// more compact to skip to.
func unionBitmaps(dst, a, b []uint32) {
	for i := range dst {
		var av, bv uint32
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		dst[i] = av | bv
	}
}

func intersectBitmaps(dst, a, b []uint32) {
	for i := range dst {
		var av, bv uint32
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		dst[i] = av & bv
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
	case recSetBitmap:
		if int(wordIndex) < len(c.part.data) {
			return c.part.data[wordIndex] & tailMask
		}
		return 0
	case recSetPositive:
		values := c.part.listedValues()
		for c.pos < len(values) && values[c.pos]>>5 < wordIndex {
			c.pos++
		}
		mask := uint32(0)
		for c.pos < len(values) && values[c.pos]>>5 == wordIndex {
			mask |= uint32(1) << (values[c.pos] & 31)
			c.pos++
		}
		return mask
	case recSetRanges:
		wordEnd := wordStart + 32
		ranges := c.part.listedRanges()
		for c.pos < len(ranges) && ranges[c.pos]+ranges[c.pos+1] <= wordStart {
			c.pos += 2
		}
		mask := uint32(0)
		for p := c.pos; p < len(ranges) && ranges[p] < wordEnd; p += 2 {
			base, count := ranges[p], ranges[p+1]
			lo, hi := base, base+count
			if lo < wordStart {
				lo = wordStart
			}
			if hi > wordEnd {
				hi = wordEnd
			}
			for id := lo; id < hi; id++ {
				mask |= uint32(1) << (id - wordStart)
			}
		}
		return mask & tailMask
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

// recSetShardFromRangePairs and recSetShardFromList both receive `data`
// as the full (universe+31)/32-word buffer the union/intersect caller
// allocated for its bitmap-fallback path, of which only the first
// 2*used/used words ended up holding the actual native-fast-path result —
// same situation as recSetShardBuilder.finish() (see its doc comment), same
// fix: copy down to a right-sized array rather than silently keeping the
// whole worst-case buffer alive for a result that's supposed to be compact.

func recSetShardFromRangePairs(shard *storageShard, universe uint32, data []uint32, used uint32) recSetShard {
	var count int64
	for i := uint32(0); i < used; i++ {
		count += int64(data[i*2+1])
	}
	part := recSetShard{shard: shard, kind: recSetRanges, universe: universe, used: used, count: count}
	if count > 0 {
		part.data = append([]uint32(nil), data[:2*used]...)
	}
	return part
}

func recSetShardFromList(shard *storageShard, universe uint32, data []uint32, used uint32) recSetShard {
	part := recSetShard{shard: shard, kind: recSetPositive, universe: universe, used: used, count: int64(used)}
	if used == 0 {
		part.kind = recSetRanges
	} else {
		part.data = append([]uint32(nil), data[:used]...)
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
		part.kind = recSetRanges
		part.data = nil
	} else {
		part.kind = recSetBitmap
	}
	return part
}

// forEachID visits every matching recid in ascending order. Built on top of
// forEachRange so the run-based iteration logic exists exactly once.
func (s *recSetShard) forEachID(callback func(uint32) bool) {
	s.forEachRange(func(base, count uint32) bool {
		for id := base; id < base+count; id++ {
			if !callback(id) {
				return false
			}
		}
		return true
	})
}

// forEachRange visits every maximal run of matching recids in ascending
// order as (base, count) — the natural granularity for a bulk column read
// (GetValueRange(base, count, ...)) instead of one GetValue call per row.
// recSetRanges yields its stored pairs directly (zero conversion — this is
// the case the whole representation exists for); recSetPositive yields each
// hit as its own (id,1) range (they're scattered by construction, so there's
// nothing to coalesce); recSetBitmap scans bit-by-bit coalescing consecutive
// set bits into runs as it goes (not the fastest possible bitmap-run scan —
// a bit-trick version could skip whole words via TrailingZeros/leading-run
// masks — but bitmap is already the fallback tier where compression failed,
// and correctness matters more than shaving that further here).
func (s *recSetShard) forEachRange(callback func(base, count uint32) bool) {
	switch s.kind {
	case recSetRanges:
		ranges := s.listedRanges()
		for i := 0; i < len(ranges); i += 2 {
			if !callback(ranges[i], ranges[i+1]) {
				return
			}
		}
	case recSetPositive:
		for _, id := range s.listedValues() {
			if !callback(id, 1) {
				return
			}
		}
	case recSetBitmap:
		var runStart uint32
		haveRun := false
		for wordIndex, word := range s.data {
			base := uint32(wordIndex) * 32
			for bitPos := uint32(0); bitPos < 32; bitPos++ {
				id := base + bitPos
				if id >= s.universe {
					word = 0
					break
				}
				if word&(uint32(1)<<bitPos) != 0 {
					if !haveRun {
						runStart = id
						haveRun = true
					}
					continue
				}
				if haveRun {
					if !callback(runStart, id-runStart) {
						return
					}
					haveRun = false
				}
			}
		}
		if haveRun {
			if !callback(runStart, s.universe-runStart) {
				return
			}
		}
	}
}
