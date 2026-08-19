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
// DESIGN INVARIANT (build phase): exactly one allocation while add() is
// being called. newRecSetShardBuilder allocates the eventual bitmap
// footprint ((universe+31)/32 words) once; every phase (ranges, positive,
// bitmap) reuses that same backing array — ranges as (base,count) pairs,
// positive as a flat hit-id list, bitmap as literal bits — and escalating
// from one phase to a less compact one is a plain clear() of that same
// array, never a new allocation. This is why escalation historically
// discarded whatever the abandoned phase had recorded and relied on the
// caller replaying: see the `wasBitmap`-watching code in recset.go's
// collectRecSet/projectJoinKeysPart and analyzer.go's BuildSkipList,
// unchanged by this rewrite. For that contract to be safe, every escalation
// must happen synchronously inside an add() call the caller can see — never
// deferred to finish(), which runs after the caller's add() loop has
// already ended with no replay window left. The ranges phase's escalation
// is therefore triggered the moment a new run *starts* (reserving its pair
// slot immediately, in addRange), not when it closes — closeRun() only ever
// fills in an already-reserved slot, so it can run safely from finish()
// without ever needing to escalate itself.
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
// universe, and paid exactly once — it is not part of the build-phase
// invariant above, which is specifically about not re-allocating or
// growing anything *while* add() is still being called.
//
// The one deliberate exception is the tryConvertToPositive path
// (ranges→positive, triggered by too many isolated singleton runs, plus its
// own overflow-to-bitmap fallback if positive wouldn't fit either):
// flattening (base,count) pairs into individual ids is not a pure
// clear+rebuild, and a naive in-place left-to-right expansion is unsafe —
// any range with count>1 writes ahead of where a not-yet-read later pair
// lives, corrupting it before it's read (a range with count=1000 at the
// front, for instance, overwrites the read position of a pair many slots
// further in before that pair has been read). Making this transition
// correct needs a temporary buffer sized to the flattened hit count —
// bounded by the same budget as everything else here, allocated at most
// once per shard build, and only on the (by design, rare) path where ranges
// already gave up on clustering.
type recSetShardBuilder struct {
	shard       *storageShard
	universe    uint32
	data        []uint32
	matched     uint32
	initialized bool

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
// backing array's budget escalates to bitmap the zero-allocation way: clear
// and rely on the caller's replay (see the type doc). This is always safe
// here because addPositive's budget check fires on every single hit, not on
// some deferred flush — there is no closeRun()-style batching that could
// let the last hit of a shard slip past the caller's add() loop unseen.
func (b *recSetShardBuilder) addPositive(recid uint32, hit bool) {
	if !hit {
		return
	}
	if b.hits >= uint32(len(b.data)) {
		clear(b.data)
		b.bitmap = true
		b.addBitmap(recid, true)
		return
	}
	b.data[b.hits] = recid
	b.hits++
}

func (b *recSetShardBuilder) addRange(recid uint32, hit bool) {
	if hit {
		if b.haveRun && recid == b.runStart+b.runLen {
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
		// real add() call — never deferred to closeRun()/finish() — so
		// this is the one and only place ranges→bitmap escalation can
		// trigger, guaranteeing the caller always has a replay window.
		if (b.rangePairs+1)*2 > uint32(len(b.data)) {
			clear(b.data)
			b.bitmap = true
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
// with no caller replay window left) purely because tryConvertToPositive
// itself is lossless — it doesn't clear anything here.
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
// count — see the type doc for why this one transition can't be a plain
// clear()+replay or a naive in-place expansion.
//
// If the total hit count doesn't fit the shared backing array's budget
// either, this falls back to bitmap — but NOT via the zero-allocation
// clear()+replay pattern addRange/addPositive use above. closeRun() (this
// function's only caller) can run from finish(), flushing the very last run
// of a shard with no caller add() loop left to replay from; a bare clear()
// here would silently drop every range pair recorded before this one,
// exactly the bug this design is built around avoiding (see the type doc).
// escalateRangesToBitmap keeps this transition self-contained the same way
// the flatten above is: convert what's already in `data` before touching
// it. This — like the flatten itself — is a second, disclosed exception to
// the one-allocation invariant, reached only via the same rare ">3
// singleton runs" path, bounded by the same budget, at most once per shard
// build.
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
// bitmap bits before clearing the backing array, so nothing is lost even
// when there's no caller replay window left to fall back on (see
// tryConvertToPositive, its only caller).
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

// listedValues returns the flat hit-id list (recSetPositive only).
func (s *recSetShard) listedValues() []uint32 {
	return s.data[:s.used]
}

// listedRanges returns the flat [base0,count0,base1,count1,...] pair array
// (recSetRanges only).
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
	}
	// Mixed kinds (or a native-representation overflow): every part still
	// answers .contains()/word() correctly regardless of its own kind, so
	// this is always correct, just without the compact-representation
	// shortcut above.
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
	}
	return combineRecSetBitmapsWithData(shard, universe, parts, false, data)
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
