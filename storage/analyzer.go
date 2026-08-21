/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "time"
import "unsafe"
import "strings"
import "math/bits"
import "sync/atomic"
import "github.com/carli2/hybridsort"
import "github.com/launix-de/memcp/scm"

func mustSymbolValue(v scm.Scmer) scm.Symbol {
	if v.IsSymbol() {
		return scm.Symbol(v.String())
	}
	panic("expected symbol")
}

// BoundaryMatcher is the plugin interface for index column types.
// Three singletons (Equal, Range, Like) are created at startup.
// Zero per-query allocation — the singletons are stored on columnboundaries
// and on StorageIndex.ColMatchers as type markers.
//
// To add a new index-aware operation:
//  1. Implement BoundaryMatcher.
//  2. Add a singleton to boundaryMatchers below.
//  3. Add detection logic in extractBoundaries (hardcoded for now).
//     Future: generalize via TryMatch method for user-defined matchers.
type BoundaryMatcher interface {
	// Kind returns a short identifier (e.g. "equal", "range", "like").
	// Used for index deduplication: same column + same kind = same index.
	Kind() string

	// IsSorted reports whether this column participates in index sort order.
	// Equal and Range: true. LIKE, Regex, IN: false.
	IsSorted() bool

	// IsPointLike reports whether this column is a point lookup for index ordering.
	// Equal and Like: true (sorted before range). Range: false.
	IsPointLike() bool

	// BuildSkipList is called once during buildIndex to create the skip list
	// for this column. Only meaningful for non-sorted matchers (LIKE etc.).
	// For sorted matchers this is a no-op. The pattern is the search value
	// (e.g. the LIKE pattern). The result is stored on the StorageIndex.
	// colStorage is the column's ColumnStorage for reading values.
	BuildSkipList(pattern, collation string, count uint32, getRecid func(uint32) uint32, colStorage ColumnStorage) *SkipList
}

// SkipList holds an exact adaptive set of matching index positions.
type SkipList struct {
	matches      recSetShard
	lastUsedNano atomic.Int64
	hitCount     atomic.Uint64
}

func (s *SkipList) recordUse() {
	if s == nil {
		return
	}
	s.lastUsedNano.Store(time.Now().UnixNano())
	s.hitCount.Add(1)
}

func (s *SkipList) lastUsed() time.Time {
	if s == nil {
		return time.Time{}
	}
	return time.Unix(0, s.lastUsedNano.Load())
}

func (s *SkipList) cacheScore() float64 {
	if s == nil {
		return 0
	}
	hits := s.hitCount.Load()
	if hits > 1024 {
		hits = 1024
	}
	return float64(hits)
}

type skipListCursor struct {
	skip    *SkipList
	listPos int
}

func (s *SkipList) cursor() skipListCursor {
	return skipListCursor{skip: s}
}

func (c *skipListCursor) NextBlock(pos uint32) (uint32, uint32, bool) {
	if c.skip == nil {
		return 0, 0, false
	}
	set := &c.skip.matches
	if pos >= set.universe || set.count == 0 {
		return 0, 0, false
	}
	switch set.kind {
	case recSetRanges:
		// A "full" set (everything matches) is just one pair covering
		// [0,universe) here — no separate case needed; if pos lands inside
		// that pair, it's trimmed to start at pos below, same as it would
		// for any other range.
		ranges := set.listedRanges()
		for c.listPos < len(ranges) && ranges[c.listPos]+ranges[c.listPos+1] <= pos {
			c.listPos += 2
		}
		if c.listPos >= len(ranges) {
			return 0, 0, false
		}
		base, count := ranges[c.listPos], ranges[c.listPos+1]
		if base < pos {
			count -= pos - base
			base = pos
		}
		return base, count, true
	case recSetPositive:
		values := set.listedValues()
		for c.listPos < len(values) && values[c.listPos] < pos {
			c.listPos++
		}
		if c.listPos == len(values) {
			return 0, 0, false
		}
		start := values[c.listPos]
		end := start + 1
		c.listPos++
		for c.listPos < len(values) && values[c.listPos] == end {
			end++
			c.listPos++
		}
		return start, end - start, true
	case recSetBitmap:
		wordIndex := pos >> 5
		word := set.data[wordIndex] & (^uint32(0) << (pos & 31))
		for word == 0 {
			wordIndex++
			if int(wordIndex) >= len(set.data) {
				return 0, 0, false
			}
			word = set.data[wordIndex]
		}
		start := (wordIndex << 5) + uint32(bits.TrailingZeros32(word))
		end := start + 1
		for end < set.universe && set.contains(end) {
			end++
		}
		return start, end - start, true
	default:
		return 0, 0, false
	}
}

func (s *SkipList) ComputeSize() uint {
	if s == nil {
		return 0
	}
	return uint(len(s.matches.data))*4 + uint(unsafe.Sizeof(*s))
}

// Built-in matcher singletons. Every columnboundaries.matcher points to one of these.
// Created once at startup, never reallocated.
//
// TODO: future matcher types to add:
//   - RegexMatcher: IsSorted=false, same SkipList architecture as LIKE, different match fn
//   - InMatcher: IsSorted=false, SkipList from sorted ID list
//   - VectorDistanceMatcher: IsSorted=false, ORDER BY vector_distance(col, query)
//     (query varies per query → cluster-based SkipList, not sort-order based)
var (
	EqualMatcher  BoundaryMatcher = &equalMatcher{}
	RangeMatcher  BoundaryMatcher = &rangeMatcher{}
	LikeMatcher   BoundaryMatcher = &likeMatcher{}
	RecSetMatcher BoundaryMatcher = &recSetMatcher{}
)

// boundaryMatchers lists all known matcher types.
var boundaryMatchers = []BoundaryMatcher{EqualMatcher, RangeMatcher, LikeMatcher, RecSetMatcher}

// --- Equal ---

type equalMatcher struct{}

func (m *equalMatcher) Kind() string      { return "equal" }
func (m *equalMatcher) IsSorted() bool    { return true }
func (m *equalMatcher) IsPointLike() bool { return true }
func (m *equalMatcher) BuildSkipList(_, _ string, _ uint32, _ func(uint32) uint32, _ ColumnStorage) *SkipList {
	return nil // sorted: no skip list needed
}

// --- Range ---

type rangeMatcher struct{}

func (m *rangeMatcher) Kind() string      { return "range" }
func (m *rangeMatcher) IsSorted() bool    { return true }
func (m *rangeMatcher) IsPointLike() bool { return false }
func (m *rangeMatcher) BuildSkipList(_, _ string, _ uint32, _ func(uint32) uint32, _ ColumnStorage) *SkipList {
	return nil // sorted: no skip list needed
}

// --- LIKE ---

type likeMatcher struct{}

func (m *likeMatcher) Kind() string      { return "like" }
func (m *likeMatcher) IsSorted() bool    { return false }
func (m *likeMatcher) IsPointLike() bool { return true }
func (m *likeMatcher) BuildSkipList(pattern, collation string, count uint32, getRecid func(uint32) uint32, colStorage ColumnStorage) *SkipList {
	if count == 0 || colStorage == nil {
		return nil
	}

	builder := newRecSetShardBuilder(nil, count, true, 0)
	matches := func(pos uint32) bool {
		recid := getRecid(pos)
		v := colStorage.GetValue(recid)
		return v.IsString() && scm.StrLikeCollation(v.String(), pattern, collation)
	}
	add := func(pos uint32) bool {
		builder.add(pos, matches(pos))
		return true
	}
	for pos := uint32(0); pos < count; pos++ {
		add(pos)
	}
	sl := &SkipList{matches: builder.finish()}
	return sl
}

// --- RecSet ---

type recSetMatcher struct{}

func (m *recSetMatcher) Kind() string      { return "recset" }
func (m *recSetMatcher) IsSorted() bool    { return false }
func (m *recSetMatcher) IsPointLike() bool { return true }
func (m *recSetMatcher) BuildSkipList(_, _ string, _ uint32, _ func(uint32) uint32, _ ColumnStorage) *SkipList {
	return nil
}

type columnboundaries struct {
	col              string
	matcher          BoundaryMatcher // always set: EqualMatcher, RangeMatcher, LikeMatcher, ...
	lower            scm.Scmer
	lowerInclusive   bool
	upper            scm.Scmer
	upperInclusive   bool
	lowerBatch       bool
	lowerBatchSubidx int
	upperBatch       bool
	upperBatchSubidx int
	collation        string // non-empty only for collation-sensitive matchers
	// order is the complete strict relation for an ORDER BY suffix, including
	// direction, collation, and NULL placement. Nil means the column is only a
	// filter boundary and uses its schema's canonical ascending relation.
	order     func(...scm.Scmer) scm.Scmer
	orderMeta string
	// for computed index columns (col starts with ".")
	mapCols []string  // source columns needed to compute the value
	mapFn   scm.Scmer // function: mapFn(mapCols values...) → index value
}

type boundaries []columnboundaries

// boundaryValueEqual compares boundary values in index-order semantics.
// Do not use scm.Equal here: it intentionally applies SQL-ish truthy/nil
// coercions (e.g. 0 == nil), which breaks range/equality boundary decisions.
func boundaryValueEqual(a, b scm.Scmer) bool {
	return !scm.Less(a, b) && !scm.Less(b, a)
}

// boundaryIsPoint delegates to the matcher's IsPointLike.
func boundaryIsPoint(b columnboundaries) bool {
	return b.matcher.IsPointLike()
}

func boundaryIsUnboundedOrder(b columnboundaries) bool {
	return b.order != nil && b.lower.IsNil() && b.upper.IsNil()
}

// addConstraint merges a column boundary into an existing set, narrowing the
// range for an already-present column (AND semantics) or appending a new entry.
func addConstraint(in boundaries, b2 columnboundaries) boundaries {
	for i, b := range in {
		if b.col == b2.col {
			if matcherKindEqual(b.matcher, RecSetMatcher) || matcherKindEqual(b2.matcher, RecSetMatcher) {
				return in
			}
			// matcher promotion: more selective matcher wins (equal > like > range)
			if b2.matcher.IsPointLike() && !b.matcher.IsPointLike() {
				in[i].matcher = b2.matcher
			}
			// lower: pick the tighter (higher) bound
			if b.lower.IsNil() || (!b2.lower.IsNil() && scm.Less(b.lower, b2.lower)) {
				in[i].lower = b2.lower
				in[i].lowerInclusive = b2.lowerInclusive
				in[i].lowerBatch = b2.lowerBatch
				in[i].lowerBatchSubidx = b2.lowerBatchSubidx
			} else if !b.lower.IsNil() && !b2.lower.IsNil() && boundaryValueEqual(b.lower, b2.lower) {
				in[i].lowerInclusive = b.lowerInclusive && b2.lowerInclusive
			}
			// upper: pick the tighter (lower) bound
			if b.upper.IsNil() || (!b2.upper.IsNil() && scm.Less(b2.upper, b.upper)) {
				in[i].upper = b2.upper
				in[i].upperInclusive = b2.upperInclusive
				in[i].upperBatch = b2.upperBatch
				in[i].upperBatchSubidx = b2.upperBatchSubidx
			} else if !b.upper.IsNil() && !b2.upper.IsNil() && boundaryValueEqual(b.upper, b2.upper) {
				in[i].upperInclusive = b.upperInclusive && b2.upperInclusive
			}
			return in
		}
	}
	return append(in, b2)
}

// widenBounds widens a into the union with b (OR semantics).
// Keeps only columns present in both; for shared columns, takes the wider range.
// Modifies a in-place, zero allocations.
func widenBounds(a, b boundaries) boundaries {
	n := 0
	for i := range a {
		found := false
		for _, cb := range b {
			if a[i].col != cb.col {
				continue
			}
			found = true
			// matcher demotion: OR takes the weaker matcher (range < like < equal)
			if !cb.matcher.IsPointLike() && a[i].matcher.IsPointLike() {
				a[i].matcher = cb.matcher
			}
			// widen lower: take the smaller
			if a[i].lower.IsNil() {
				// already unbounded
			} else if cb.lower.IsNil() {
				a[i].lower = scm.NewNil()
				a[i].lowerInclusive = false
				a[i].lowerBatch = false
				a[i].lowerBatchSubidx = 0
			} else if scm.Less(cb.lower, a[i].lower) {
				a[i].lower = cb.lower
				a[i].lowerInclusive = cb.lowerInclusive
				a[i].lowerBatch = cb.lowerBatch
				a[i].lowerBatchSubidx = cb.lowerBatchSubidx
			} else if boundaryValueEqual(cb.lower, a[i].lower) {
				a[i].lowerInclusive = a[i].lowerInclusive || cb.lowerInclusive
			}
			// widen upper: take the larger
			if a[i].upper.IsNil() {
				// already unbounded
			} else if cb.upper.IsNil() {
				a[i].upper = scm.NewNil()
				a[i].upperInclusive = false
				a[i].upperBatch = false
				a[i].upperBatchSubidx = 0
			} else if scm.Less(a[i].upper, cb.upper) {
				a[i].upper = cb.upper
				a[i].upperInclusive = cb.upperInclusive
				a[i].upperBatch = cb.upperBatch
				a[i].upperBatchSubidx = cb.upperBatchSubidx
			} else if boundaryValueEqual(a[i].upper, cb.upper) {
				a[i].upperInclusive = a[i].upperInclusive || cb.upperInclusive
			}
			// The union of distinct equality points is a range. Keeping the
			// EqualMatcher here would let adaptive index ordering place this
			// widened column before real equality columns and make the B-tree
			// scan treat only the lower point as an exact prefix.
			if matcherKindEqual(a[i].matcher, EqualMatcher) {
				samePoint := a[i].lowerBatch && a[i].upperBatch &&
					a[i].lowerBatchSubidx == a[i].upperBatchSubidx
				if !a[i].lowerBatch && !a[i].upperBatch {
					samePoint = boundaryValueEqual(a[i].lower, a[i].upper)
				}
				if !samePoint {
					a[i].matcher = RangeMatcher
				}
			}
			break
		}
		if found {
			a[n] = a[i]
			n++
		}
	}
	return a[:n]
}

// analyzes a lambda expression for value boundaries, so the best index can be found
func extractBoundaries(conditionCols []string, condition scm.Scmer) boundaries {
	var p scm.Proc
	if condition.IsProc() {
		p = *condition.Proc()
	} else if si, ok := condition.Any().(scm.Proc); ok {
		// fallback for legacy tagAny procs
		p = si
	} else {
		// native Go function - no boundary extraction possible (full scan)
		return nil
	}
	var params []scm.Scmer
	if p.Params.IsSlice() {
		params = p.Params.Slice()
	}
	resolveParamName := func(node scm.Scmer) (string, bool) {
		if node.IsSymbol() {
			name := node.String()
			for i, sym := range params {
				if sym.IsSymbol() && sym.String() == name {
					return conditionCols[i], true
				}
			}
		}
		if node.IsNthLocalVar() {
			idx := int(node.NthLocalVar())
			if idx < len(conditionCols) {
				return conditionCols[idx], true
			}
		}
		return "", false
	}
	// resolveColVar maps a node to a column name.
	// Handles both symbol params (linear scan, no alloc) and NthLocalVar(i).
	resolveColVar := func(node scm.Scmer) (string, bool) {
		if name, ok := resolveParamName(node); ok {
			if !isScanPseudoColName(name) {
				return name, true
			}
		}
		return "", false
	}
	resolveBatchSubidx := func(node scm.Scmer) (int, bool) {
		if name, ok := resolveParamName(node); ok {
			return parseBatchPseudoColName(name)
		}
		return 0, false
	}
	resolveOuterReference := func(node scm.Scmer) (scm.Scmer, bool) {
		depth := 0
		for {
			parts, ok := scmerSlice(node)
			if !ok || len(parts) != 2 || !scanSymbolIs(parts[0], "outer") {
				break
			}
			depth++
			node = parts[1]
		}
		if depth == 0 {
			return scm.NewNil(), false
		}

		env := p.En
		for level := 1; level < depth && env != nil; level++ {
			env = env.Outer
		}
		if env == nil {
			return scm.NewNil(), false
		}
		if node.IsSymbol() {
			sym := scm.Symbol(node.String())
			if binding := env.FindRead(sym); binding != nil {
				value, ok := binding.Vars[sym]
				return value, ok
			}
		}
		if node.IsNthLocalVar() {
			idx := int(node.NthLocalVar())
			if idx < len(env.VarsNumbered) {
				return env.VarsNumbered[idx], true
			}
		}
		return scm.NewNil(), false
	}
	// analyze condition for AND clauses, equal? < > <= >= BETWEEN
	extractConstant := func(v scm.Scmer) (scm.Scmer, bool) {
		if v.IsInt() || v.IsFloat() || v.IsString() || v.IsBool() || v.IsCustom(TagRecSet) {
			return v, true
		}
		if v.IsSymbol() {
			if val2, ok := p.En.Vars[scm.Symbol(v.String())]; ok {
				if val2.IsInt() || val2.IsFloat() || val2.IsString() || val2.IsCustom(TagRecSet) {
					return val2, true
				}
			}
		}
		if val2, ok := resolveOuterReference(v); ok {
			if val2.IsInt() || val2.IsFloat() || val2.IsString() || val2.IsCustom(TagRecSet) {
				return val2, true
			}
		}
		if isIndependent(params, v) {
			if val2, ok := evalIndependentScmer(v, p.En); ok {
				if val2.IsInt() || val2.IsFloat() || val2.IsString() || val2.IsBool() || val2.IsNil() || val2.IsCustom(TagRecSet) {
					return val2, true
				}
			}
		}
		return scm.NewNil(), false
	}
	// traverseCondition returns boundaries for a single AST node.
	// nil means "unknown node, no bounds extractable".
	// AND: merge children (intersect). OR: widen children (union).
	var traverseCondition func(scm.Scmer) boundaries
	traverseCondition = func(node scm.Scmer) boundaries {
		if !node.IsSlice() {
			return nil
		}
		v := node.Slice()
		if len(v) == 0 {
			return nil
		}
		// funcIs checks if head represents the named function.
		// Works for both unoptimized (symbol) and optimizer-resolved (tagFunc) forms.
		funcIs := func(head scm.Scmer, name string) bool {
			if head.SymbolEquals(name) {
				return true
			}
			d := scm.DeclarationForValue(head)
			return d != nil && d.Name == name
		}
		if funcIs(v[0], "optimize") && len(v) == 2 {
			return traverseCondition(v[1])
		}
		if col, ok := resolveParamName(v[0]); ok && col == "$recset_contains" && len(v) == 2 {
			if rs, ok := extractConstant(v[1]); ok && rs.IsCustom(TagRecSet) {
				return boundaries{columnboundaries{col: col, matcher: RecSetMatcher, lower: rs, lowerInclusive: true, upper: rs, upperInclusive: true}}
			}
		}
		if funcIs(v[0], "equal?") || funcIs(v[0], "equal??") {
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
				if subidx, ok := resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// reversed: (equal? const col)
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
				if subidx, ok := resolveBatchSubidx(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// computed col: (equal? rawDataset independent) or reversed
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[1]) {
					if subidx, ok := resolveBatchSubidx(v[2]); ok {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			if len(params) > 0 && v[2].IsSlice() {
				if isRawDataset(params, v[2]) && isIndependent(params, v[1]) {
					if v2, ok2 := evalIndependentScmer(v[1], p.En); ok2 {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[2]) {
					if subidx, ok := resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], "<") || funcIs(v[0], "<=") {
			incl := v[0].SymbolEquals("<=")
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
				if subidx, ok := resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// reversed: (< const col) means col > const, (<= const col) means col >= const
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
				if subidx, ok := resolveBatchSubidx(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx}}
				}
			}
			// computed col: rawDataset < independent → rawDataset has upper bound
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[1]) {
					if subidx, ok := resolveBatchSubidx(v[2]); ok {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			// reversed computed: independent < rawDataset → rawDataset has lower bound
			if len(params) > 0 && v[2].IsSlice() {
				if isRawDataset(params, v[2]) && isIndependent(params, v[1]) {
					if v2, ok2 := evalIndependentScmer(v[1], p.En); ok2 {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[2]) {
					if subidx, ok := resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], ">") || funcIs(v[0], ">=") {
			incl := v[0].SymbolEquals(">=")
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
				if subidx, ok := resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx}}
				}
			}
			// reversed: (> const col) means col < const, (>= const col) means col <= const
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
				if subidx, ok := resolveBatchSubidx(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// computed col: rawDataset > independent → rawDataset has lower bound
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[1]) {
					if subidx, ok := resolveBatchSubidx(v[2]); ok {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			// reversed computed: independent > rawDataset → rawDataset has upper bound
			if len(params) > 0 && v[2].IsSlice() {
				if isRawDataset(params, v[2]) && isIndependent(params, v[1]) {
					if v2, ok2 := evalIndependentScmer(v[1], p.En); ok2 {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl, mapCols: mc, mapFn: mf}}
						}
					}
				} else if isRawDataset(params, v[2]) {
					if subidx, ok := resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], "nil?") && len(v) >= 2 {
			// IS NULL: (nil? col)
			if col, ok := resolveColVar(v[1]); ok {
				return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: scm.NewNil(), lowerInclusive: true, upper: scm.NewNil(), upperInclusive: true}}
			}
			return nil
		} else if funcIs(v[0], "strlike") && len(v) >= 3 {
			// LIKE: (strlike col "pattern" collation)
			if col, ok := resolveColVar(v[1]); ok {
				if pat, ok := extractConstant(v[2]); ok && pat.IsString() {
					collation := "utf8mb4_general_ci"
					if len(v) >= 4 {
						coll, constant := extractConstant(v[3])
						if !constant || !coll.IsString() {
							return nil
						}
						collation = strings.ToLower(coll.String())
					}
					pattern := pat.String()
					idx := strings.IndexAny(pattern, "%_")
					if idx > 0 && !strings.Contains(collation, "_ci") {
						// prefix-anchored LIKE "foo%" → range boundary
						prefix := pattern[:idx]
						upperBytes := []byte(prefix)
						upperBytes[len(upperBytes)-1]++
						return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewString(prefix), lowerInclusive: true, upper: scm.NewString(string(upperBytes)), upperInclusive: false}}
					}
					// non-prefix LIKE "%foo%" → matcher boundary
					return boundaries{columnboundaries{col: col, matcher: LikeMatcher, lower: pat, upper: pat, collation: collation}}
				}
			}
			return nil
		} else if v[0].SymbolEquals("and") {
			var result boundaries
			for i := 1; i < len(v); i++ {
				child := traverseCondition(v[i])
				if child == nil {
					continue
				}
				if result == nil {
					result = child
				} else {
					for _, cb := range child {
						result = addConstraint(result, cb)
					}
				}
			}
			return result
		} else if v[0].SymbolEquals("or") {
			// If the whole OR is a pure row-column expression, index as computed bool col.
			// This avoids range-merging that would span too wide.
			if len(params) > 0 && isRawDataset(params, node) {
				canon := canonicalColName(node, params, conditionCols)
				mc, mf := buildComputedFn(node, p.Params, p.En, conditionCols)
				if !mf.IsNil() && mc != nil {
					return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lower: scm.NewBool(true), lowerInclusive: true, upper: scm.NewBool(true), upperInclusive: true, mapCols: mc, mapFn: mf}}
				}
			}
			var result boundaries
			for i := 1; i < len(v); i++ {
				child := traverseCondition(v[i])
				if child == nil {
					return nil // can't narrow this branch → full scan
				}
				if result == nil {
					result = child
				} else {
					result = widenBounds(result, child)
					if len(result) == 0 {
						return nil
					}
				}
				for _, cb := range result {
					if matcherKindEqual(cb.matcher, RecSetMatcher) {
						return nil
					}
				}
			}
			return result
		}
		// Fallback: if the whole expression is a pure function of row columns
		// (no comparison operator matched above), treat it as a computed bool column.
		// Boundary {true, true} means: only scan rows where the expression is true.
		if len(params) > 0 && isRawDataset(params, node) {
			canon := canonicalColName(node, params, conditionCols)
			mc, mf := buildComputedFn(node, p.Params, p.En, conditionCols)
			if !mf.IsNil() && mc != nil {
				return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lower: scm.NewBool(true), lowerInclusive: true, upper: scm.NewBool(true), upperInclusive: true, mapCols: mc, mapFn: mf}}
			}
		}
		return nil
	}
	cols := traverseCondition(p.Body)

	// Keep transient RecSet boundaries in a prefix so scans can remove them by
	// slicing. Sort real columns point-like (equal + like) first, then range,
	// alphabetically within each group. LIKE columns are treated as point-like because
	// the index sort treats them like any other column; the LIKE pattern is a query-level
	// overlay that filters via rowWithinBounds, not via sort order.
	if len(cols) > 1 {
		hybridsort.Slice(cols, func(i, j int) bool {
			iRecSet := matcherKindEqual(cols[i].matcher, RecSetMatcher)
			jRecSet := matcherKindEqual(cols[j].matcher, RecSetMatcher)
			if iRecSet != jRecSet {
				return iRecSet
			}
			iPoint := boundaryIsPoint(cols[i])
			jPoint := boundaryIsPoint(cols[j])
			if iPoint != jPoint {
				return iPoint // point-like (equal/like) before range
			}
			return cols[i].col < cols[j].col // tiebreak alphabetically
		})
	}

	return cols
}

// singleLikeBoundaryCoversCondition proves that the condition consists solely
// of the LIKE call represented by bound. An exact main-row match set may bypass
// residual evaluation only under this structural proof; deltas are never
// covered because they are not part of the immutable set.
func singleLikeBoundaryCoversCondition(conditionCols []string, condition scm.Scmer, bound columnboundaries) bool {
	if bound.matcher != LikeMatcher {
		return false
	}
	var p scm.Proc
	if condition.IsProc() {
		p = *condition.Proc()
	} else if legacy, ok := condition.Any().(scm.Proc); ok {
		p = legacy
	} else {
		return false
	}
	if !p.Params.IsSlice() {
		return false
	}
	params := p.Params.Slice()
	body := p.Body
	for {
		items, ok := scmerSlice(body)
		if !ok || len(items) != 2 {
			break
		}
		declaration := scm.DeclarationForValue(items[0])
		if !items[0].SymbolEquals("optimize") && (declaration == nil || declaration.Name != "optimize") {
			break
		}
		body = items[1]
	}
	items, ok := scmerSlice(body)
	if !ok || len(items) < 3 {
		return false
	}
	declaration := scm.DeclarationForValue(items[0])
	if !items[0].SymbolEquals("strlike") && (declaration == nil || declaration.Name != "strlike") {
		return false
	}
	if items[1].IsNthLocalVar() {
		idx := int(items[1].NthLocalVar())
		return idx < len(conditionCols) && conditionCols[idx] == bound.col
	}
	if items[1].IsSymbol() {
		for i, param := range params {
			if i < len(conditionCols) && param.IsSymbol() && param.String() == items[1].String() {
				return conditionCols[i] == bound.col
			}
		}
	}
	return false
}

func splitRecSetBoundary(b boundaries, backingTable *table) (boundaries, *recSet) {
	var rs *recSet
	prefixLen := 0
	for prefixLen < len(b) && matcherKindEqual(b[prefixLen].matcher, RecSetMatcher) {
		if rs == nil && b[prefixLen].lower.IsCustom(TagRecSet) {
			candidate := RecSetFromScmer(b[prefixLen].lower)
			if candidate != nil && candidate.table == backingTable {
				rs = candidate
			}
		}
		prefixLen++
	}
	return b[prefixLen:], rs
}

func hasBatchBoundaries(bounds boundaries) bool {
	for _, b := range bounds {
		if b.lowerBatch || b.upperBatch {
			return true
		}
	}
	return false
}

func materializeBatchBoundaries(template boundaries, stride int, batchdata []scm.Scmer, batchid uint32) boundaries {
	result := append(boundaries(nil), template...)
	base := int(batchid) * stride
	for i := range result {
		if result[i].lowerBatch {
			result[i].lower = batchdata[base+result[i].lowerBatchSubidx]
		}
		if result[i].upperBatch {
			result[i].upper = batchdata[base+result[i].upperBatchSubidx]
		}
	}
	return result
}

// reorderByFrequency keeps exact sorted points ahead of matcher-backed points.
// A PK equality must be allowed to narrow a nested probe before an expensive LIKE
// matcher is considered. Frequency only reorders boundaries with the same access
// characteristics, preserving prefix reuse without overriding that cost property.
func reorderByFrequency(bounds boundaries, t *table) {
	for _, b := range bounds {
		t.bumpColFreq(b.col)
	}
	hybridsort.SliceStable(bounds, func(i, j int) bool {
		iEq := boundaryIsPoint(bounds[i])
		jEq := boundaryIsPoint(bounds[j])
		if iEq != jEq {
			return iEq // equality first
		}
		iSortedPoint := iEq && bounds[i].matcher.IsSorted()
		jSortedPoint := jEq && bounds[j].matcher.IsSorted()
		if iSortedPoint != jSortedPoint {
			return iSortedPoint
		}
		if iEq && jEq {
			fi, fj := t.getColFreq(bounds[i].col), t.getColFreq(bounds[j].col)
			if fi != fj {
				return fi > fj // higher frequency first
			}
		}
		return bounds[i].col < bounds[j].col // tiebreak alphabetically
	})
}

// analyzeOrcPartition inspects reduceFn + reduceInit + sortCols to detect
// whether the ORC uses a partition wrapper. Returns the number of leading
// sort columns that serve as partition keys (0 = no partitioning).
//
// Detection: reduceInit = (list inner_init nil) with exactly 2 elements
// AND at least 2 sort columns (need partition + order). The first sort
// column(s) become the partition key, the last is the order column.
//
// This correctly distinguishes:
//   - DENSE_RANK (list 0 nil) + 1 sortCol → 0 (no partition)
//   - Partitioned ROW_NUMBER (list 0 nil) + 2 sortCols → 1
//   - Partitioned RANK (list (list 0 0 nil) nil) + 2 sortCols → 1
func analyzeOrcPartition(col *column) int {
	if col.OrcPartitionCount > 0 {
		if col.OrcPartitionCount > len(col.OrcSortCols) {
			return len(col.OrcSortCols)
		}
		return col.OrcPartitionCount
	}
	if len(col.OrcSortCols) < 2 {
		return 0
	}
	init := col.OrcReduceInit
	if init.IsNil() || !init.IsSlice() {
		return 0
	}
	items := init.Slice()
	if len(items) != 2 || !items[1].IsNil() {
		return 0
	}
	// Detected: (list inner_init nil) with 2+ sort columns.
	// First sort column is the partition key.
	return 1
}

// ORC suffix recompute mode classification.
const (
	OrcSuffixOpaque          = 0 // can't analyze → full recompute only
	OrcSuffixIdentity        = 1 // acc == emitted value (SUM, ROW_NUMBER) → stored value is accumulator
	OrcSuffixReconstructible = 2 // acc = (emitted, ...state) → need extra state from row data
)

// OrcAdditiveInfo describes a reducer that computes acc + f(mapped).
// When detected, INSERT/DELETE can be handled by adding/subtracting the delta
// to all subsequent stored values instead of running a full suffix recompute.
type OrcAdditiveInfo struct {
	IsAdditive bool      // true if reducer is (+ acc f(mapped))
	DeltaExpr  scm.Scmer // the f(mapped) expression (e.g. (cadr mapped) for running SUM)
}

// analyzeOrcAdditive inspects the ORC reduceFn to detect the additive pattern:
//
//	return value = (+ acc X) where X depends only on mapped, not acc.
//
// This enables O(N) delta propagation instead of O(N) suffix recompute.
func analyzeOrcAdditive(reduceFn scm.Scmer) OrcAdditiveInfo {
	if reduceFn.IsNil() {
		return OrcAdditiveInfo{}
	}
	var body scm.Scmer
	var accParam string
	if reduceFn.IsProc() {
		body = reduceFn.Proc().Body
		if reduceFn.Proc().Params.IsSlice() {
			params := reduceFn.Proc().Params.Slice()
			if len(params) >= 1 && params[0].IsSymbol() {
				accParam = params[0].String()
			}
		}
	} else if reduceFn.IsSlice() {
		items := reduceFn.Slice()
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			body = items[2]
			if items[1].IsSlice() {
				params := items[1].Slice()
				if len(params) >= 1 && params[0].IsSymbol() {
					accParam = params[0].String()
				}
			}
		}
	}
	if body.IsNil() || accParam == "" {
		return OrcAdditiveInfo{}
	}

	// Find return value (last expr in begin block)
	var returnVal scm.Scmer
	if body.IsSlice() {
		items := body.Slice()
		if len(items) >= 2 && items[0].IsSymbol() && items[0].String() == "begin" {
			returnVal = items[len(items)-1]
		} else {
			returnVal = body
		}
	}
	if returnVal.IsNil() || !returnVal.IsSlice() {
		return OrcAdditiveInfo{}
	}

	// Check: is returnVal = (+ acc X) ?
	rv := returnVal.Slice()
	if len(rv) != 3 {
		return OrcAdditiveInfo{}
	}
	isPlus := rv[0].IsSymbol() && rv[0].String() == "+"
	if !isPlus {
		// Check for tagFunc-resolved +
		d := scm.DeclarationForValue(rv[0])
		if d == nil || d.Name != "+" {
			return OrcAdditiveInfo{}
		}
	}

	// One operand must be acc, the other must not reference acc.
	var deltaExpr scm.Scmer
	if rv[1].IsSymbol() && rv[1].String() == accParam {
		deltaExpr = rv[2]
	} else if rv[1].IsNthLocalVar() && rv[1].NthLocalVar() == 0 {
		// NthLocalVar(0) = first param = acc
		deltaExpr = rv[2]
	} else if rv[2].IsSymbol() && rv[2].String() == accParam {
		deltaExpr = rv[1]
	} else if rv[2].IsNthLocalVar() && rv[2].NthLocalVar() == 0 {
		deltaExpr = rv[1]
	}

	if deltaExpr.IsNil() {
		return OrcAdditiveInfo{}
	}

	// Verify deltaExpr does not reference acc
	if containsSymbol(deltaExpr, accParam) {
		return OrcAdditiveInfo{}
	}

	return OrcAdditiveInfo{IsAdditive: true, DeltaExpr: deltaExpr}
}

// containsSymbol checks if an AST node references a given symbol name.
func containsSymbol(expr scm.Scmer, name string) bool {
	if expr.IsSymbol() && expr.String() == name {
		return true
	}
	if expr.IsSlice() {
		for _, item := range expr.Slice() {
			if containsSymbol(item, name) {
				return true
			}
		}
	}
	return false
}

// analyzeOrcSuffix inspects an ORC reduceFn to determine if the accumulator
// equals the emitted value ($set argument). This enables suffix recompute
// by reading the stored ORC value as the start accumulator.
//
// The reducer has the form: (lambda (acc mapped) body)
// where body calls (setter value) and returns new_acc.
// If value == new_acc, it's an identity accumulator.
func analyzeOrcSuffix(reduceFn scm.Scmer) int {
	if reduceFn.IsNil() {
		return OrcSuffixOpaque
	}
	var body scm.Scmer
	if reduceFn.IsProc() {
		body = reduceFn.Proc().Body
	} else if reduceFn.IsSlice() {
		items := reduceFn.Slice()
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			body = items[2]
		}
	}
	if body.IsNil() {
		return OrcSuffixOpaque
	}

	// Unwrap (begin ...) to find the last expression (= return value)
	// and any setter call (= $set invocation).
	var setArg scm.Scmer    // the value passed to $set
	var returnVal scm.Scmer // the return value of the reducer

	if body.IsSlice() {
		items := body.Slice()
		if len(items) >= 2 && items[0].IsSymbol() && items[0].String() == "begin" {
			returnVal = items[len(items)-1]
			// Search for setter call: ((car mapped) val) or ((nth mapped 0) val)
			for _, item := range items[1 : len(items)-1] {
				if sa := findSetterArg(item); !sa.IsNil() {
					setArg = sa
				}
			}
		} else {
			// No begin — body IS the return value
			returnVal = body
		}
	}

	if setArg.IsNil() || returnVal.IsNil() {
		return OrcSuffixOpaque
	}

	// Compare: are they structurally equal?
	if scmerStructEqual(setArg, returnVal) {
		return OrcSuffixIdentity
	}

	return OrcSuffixOpaque
}

// findSetterArg looks for a call pattern ((car mapped) val) or ((nth mapped 0) val)
// and returns val. These are the patterns produced by ORC reducers calling the $set closure.
func findSetterArg(expr scm.Scmer) scm.Scmer {
	if !expr.IsSlice() {
		return scm.NewNil()
	}
	items := expr.Slice()
	if len(items) < 2 {
		return scm.NewNil()
	}
	// Check if items[0] is (car mapped) or (nth mapped 0)
	if items[0].IsSlice() {
		head := items[0].Slice()
		if len(head) == 2 && head[0].IsSymbol() && head[0].String() == "car" {
			return items[1] // the value passed to $set
		}
		if len(head) == 3 && head[0].IsSymbol() && head[0].String() == "nth" {
			return items[1]
		}
	}
	// Recurse into begin blocks
	if items[0].IsSymbol() && items[0].String() == "begin" {
		for _, item := range items[1:] {
			if sa := findSetterArg(item); !sa.IsNil() {
				return sa
			}
		}
	}
	return scm.NewNil()
}

// scmerStructEqual compares two Scmer AST nodes for structural equality.
// Handles symbols, ints, floats, strings, and nested slices.
func scmerStructEqual(a, b scm.Scmer) bool {
	if a.IsSymbol() && b.IsSymbol() {
		return a.String() == b.String()
	}
	if a.IsInt() && b.IsInt() {
		return a.Int() == b.Int()
	}
	if a.IsFloat() && b.IsFloat() {
		return a.Float() == b.Float()
	}
	if a.IsString() && b.IsString() {
		return a.String() == b.String()
	}
	if a.IsNthLocalVar() && b.IsNthLocalVar() {
		return a.NthLocalVar() == b.NthLocalVar()
	}
	if a.IsSlice() && b.IsSlice() {
		as, bs := a.Slice(), b.Slice()
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !scmerStructEqual(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func indexFromBoundaries(cols boundaries) (lower []scm.Scmer, upperLast scm.Scmer) {
	if len(cols) > 0 {
		// A non-sorted matcher after an exact sorted prefix is cheaper as a
		// residual predicate: the prefix has already reduced the candidate set,
		// while building a matcher skip list would scan the complete shard. Keep
		// matcher-backed access for scans which have no exact sorted prefix.
		hasExactPrefix := false
		for i, col := range cols {
			if col.matcher.IsSorted() && col.matcher.IsPointLike() {
				hasExactPrefix = true
				continue
			}
			if hasExactPrefix && !col.matcher.IsSorted() {
				cols = cols[:i]
				break
			}
		}
		//fmt.Println("conditions:", cols)
		// build up lower and upper bounds of index
		for {
			if len(cols) >= 2 && !boundaryIsPoint(cols[len(cols)-2]) &&
				!(boundaryIsUnboundedOrder(cols[len(cols)-2]) && boundaryIsUnboundedOrder(cols[len(cols)-1])) {
				// remove last col -> we cant have two ranged cols
				cols = cols[:len(cols)-1]
			} else {
				break // finished -> pure index
			}
		}
		// find out boundaries
		lower = make([]scm.Scmer, len(cols))
		for i, v := range cols {
			lower[i] = v.lower
		}
		upperLast = cols[len(cols)-1].upper
		//fmt.Println(cols, lower, upperLast) // debug output if we found the right boundaries
	}
	return
}
