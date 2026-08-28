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

import "github.com/carli2/hybridsort"
import "github.com/launix-de/memcp/scm"

func mustSymbolValue(v scm.Scmer) scm.Symbol {
	if v.IsSymbol() {
		return scm.Symbol(v.String())
	}
	panic("expected symbol")
}

// IndexRowMatcher is bound once per index run. It mutates and may shorten the
// caller-owned RecID batch without allocating. The scan owns LIMIT and stop.
type IndexRowMatcher func([]uint32) []uint32

// IndexHook is shard-bound and reusable. Custom indexes keep their
// complete query-independent cache behind this public interface.
type IndexHook interface {
	Bind(lower scm.Scmer) IndexRowMatcher
	ComputeSize() uint
}

// IndexCandidateEstimator is an optional, constant-time cardinality view over
// a bound hook. Candidate counts may be upper bounds because the residual SQL
// predicate remains authoritative for correctness.
type IndexCandidateEstimator interface {
	EstimateCandidates(lower scm.Scmer) (candidates uint32, universe uint32, ok bool)
}

// IndexDeployContext is passed while binding an analyzer to one shard. External
// custom indexes can use the public batch reader without accessing shard state.
type IndexDeployContext struct {
	MainCount uint32
	Column    ColumnReader
	shard     *storageShard
}

// IndexAnalyzeContext exposes the existing lambda resolver to registered
// analyzers without exposing optimizer internals.
type IndexAnalyzeContext interface {
	ResolveParameter(scm.Scmer) (string, bool)
	ResolveColumn(scm.Scmer) (string, bool)
	ExtractConstant(scm.Scmer) (scm.Scmer, bool)
	FunctionIs(scm.Scmer, string) bool
}

// IndexAnalyzer is the allocation-free global analyzer singleton stored
// directly on every boundary and StorageIndex column.
type IndexAnalyzer interface {
	Analyze(IndexAnalyzeContext, scm.Scmer) (IndexBoundary, bool)

	// Kind returns a short identifier (e.g. "equal", "range", "like").
	// Used for index deduplication: same column + same kind = same index.
	Kind() string

	// IsSorted reports whether this column participates in index sort order.
	// Equal and Range: true. LIKE, Regex, IN: false.
	IsSorted() bool

	// IsPointLike reports whether this column is a point lookup for index ordering.
	// Equal and Like: true (sorted before range). Range: false.
	IsPointLike() bool

	// Deploy binds this analyzer to a shard. persistent is false for a
	// cold fallback run; expensive indexes may return nil until autoindex build.
	Deploy(IndexDeployContext, bool) IndexHook
}

// Built-in matcher singletons. Every columnboundaries.matcher points to one of these.
// Created once at startup, never reallocated.
var (
	EqualMatcher  IndexAnalyzer = &equalMatcher{}
	RangeMatcher  IndexAnalyzer = &rangeMatcher{}
	LikeMatcher   IndexAnalyzer = &likeMatcher{}
	RecSetMatcher IndexAnalyzer = &recSetMatcher{}
)

// boundaryMatchers lists all known matcher types.
var boundaryMatchers = []IndexAnalyzer{EqualMatcher, RangeMatcher, LikeMatcher, RecSetMatcher}

// RegisterIndexAnalyzer installs a custom analyzer. Registration is intended
// for package initialization, before queries start.
func RegisterIndexAnalyzer(analyzer IndexAnalyzer) {
	boundaryMatchers = append(boundaryMatchers, analyzer)
}

// --- Equal ---

type equalMatcher struct{}

func (m *equalMatcher) Kind() string      { return "equal" }
func (m *equalMatcher) IsSorted() bool    { return true }
func (m *equalMatcher) IsPointLike() bool { return true }
func (m *equalMatcher) Analyze(IndexAnalyzeContext, scm.Scmer) (IndexBoundary, bool) {
	return IndexBoundary{}, false
}
func (m *equalMatcher) Deploy(IndexDeployContext, bool) IndexHook { return nil }

// --- Range ---

type rangeMatcher struct{}

func (m *rangeMatcher) Kind() string      { return "range" }
func (m *rangeMatcher) IsSorted() bool    { return true }
func (m *rangeMatcher) IsPointLike() bool { return false }
func (m *rangeMatcher) Analyze(IndexAnalyzeContext, scm.Scmer) (IndexBoundary, bool) {
	return IndexBoundary{}, false
}
func (m *rangeMatcher) Deploy(IndexDeployContext, bool) IndexHook { return nil }

type columnboundaries struct {
	col              string
	matcher          IndexAnalyzer // always set: EqualMatcher, RangeMatcher, LikeMatcher, ...
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
	// mandatory marks a physical source constraint which has no duplicate in
	// the residual SQL predicate. Optimizations may discard advisory candidate
	// hooks after a unique point lookup, but must retain mandatory boundaries.
	mandatory bool
}

type boundaries []columnboundaries

// IndexBoundary is the public boundary value returned by custom analyzers. Its
// storage stays equal to the internal boundary representation.
type IndexBoundary = columnboundaries

func NewIndexBoundary(column string, analyzer IndexAnalyzer, binding scm.Scmer, collation string) IndexBoundary {
	return columnboundaries{col: column, matcher: analyzer, lower: binding, upper: binding, lowerInclusive: true, upperInclusive: true, collation: collation}
}

func (b columnboundaries) ColumnName() string { return b.col }
func (b columnboundaries) Binding() scm.Scmer { return b.lower }
func (b columnboundaries) Collation() string  { return b.collation }

type indexAnalyzeContext struct {
	proc          *scm.Proc
	params        []scm.Scmer
	conditionCols []string
}

func (c *indexAnalyzeContext) ResolveParameter(v scm.Scmer) (string, bool) {
	v = v.WithoutSourceInfo()
	if v.IsSymbol() {
		name := v.String()
		for i, sym := range c.params {
			if i < len(c.conditionCols) && sym.IsSymbol() && sym.String() == name {
				return c.conditionCols[i], true
			}
		}
	}
	if v.IsNthLocalVar() {
		idx := int(v.NthLocalVar())
		if idx < len(c.conditionCols) {
			return c.conditionCols[idx], true
		}
	}
	return "", false
}

func (c *indexAnalyzeContext) ResolveColumn(v scm.Scmer) (string, bool) {
	name, ok := c.ResolveParameter(v)
	if !ok || isScanPseudoColName(name) {
		return "", false
	}
	return name, true
}

func (c *indexAnalyzeContext) resolveBatchSubidx(v scm.Scmer) (int, bool) {
	name, ok := c.ResolveParameter(v)
	if !ok {
		return 0, false
	}
	return parseBatchPseudoColName(name)
}

func (c *indexAnalyzeContext) resolveOuterReference(v scm.Scmer) (scm.Scmer, bool) {
	depth := 0
	for {
		parts, ok := scmerSlice(v)
		if !ok || len(parts) != 2 || !scanSymbolIs(parts[0], "outer") {
			break
		}
		depth++
		v = parts[1]
	}
	if depth == 0 {
		return scm.NewNil(), false
	}

	env := c.proc.En
	for level := 1; level < depth && env != nil; level++ {
		env = env.Outer
	}
	if env == nil {
		return scm.NewNil(), false
	}
	if v.IsSymbol() {
		sym := scm.Symbol(v.String())
		if binding := env.FindRead(sym); binding != nil {
			value, ok := binding.Vars[sym]
			return value, ok
		}
	}
	if v.IsNthLocalVar() {
		idx := int(v.NthLocalVar())
		if idx < len(env.VarsNumbered) {
			return env.VarsNumbered[idx], true
		}
	}
	return scm.NewNil(), false
}

func (c *indexAnalyzeContext) ExtractConstant(v scm.Scmer) (scm.Scmer, bool) {
	v = v.WithoutSourceInfo()
	if v.IsInt() || v.IsFloat() || v.IsString() || v.IsBool() || v.IsCustom(TagRecSet) {
		return v, true
	}
	if v.IsSymbol() {
		if value, ok := c.proc.En.Vars[scm.Symbol(v.String())]; ok {
			if value.IsInt() || value.IsFloat() || value.IsString() || value.IsCustom(TagRecSet) {
				return value, true
			}
		}
	}
	if value, ok := c.resolveOuterReference(v); ok {
		if value.IsInt() || value.IsFloat() || value.IsString() || value.IsCustom(TagRecSet) {
			return value, true
		}
	}
	if isIndependent(c.params, v) {
		if value, ok := evalIndependentScmer(v, c.proc.En); ok {
			if value.IsInt() || value.IsFloat() || value.IsString() || value.IsBool() || value.IsNil() || value.IsCustom(TagRecSet) {
				return value, true
			}
		}
	}
	return scm.NewNil(), false
}

func (c *indexAnalyzeContext) FunctionIs(v scm.Scmer, name string) bool {
	v = v.WithoutSourceInfo()
	if v.SymbolEquals(name) {
		return true
	}
	declaration := scm.DeclarationForValue(v)
	return declaration != nil && declaration.Name == name
}

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

func boundaryIsUnboundedRange(b columnboundaries) bool {
	return b.matcher.IsSorted() && !boundaryIsPoint(b) && b.lower.IsNil() && b.upper.IsNil()
}

// addConstraint merges a column boundary into an existing set, narrowing the
// range for an already-present column (AND semantics) or appending a new entry.
func addConstraint(in boundaries, b2 columnboundaries) boundaries {
	for i, b := range in {
		if b.col == b2.col {
			// Non-ordering indexes are independent candidate filters. Preserve
			// every one so different custom indexes can be stacked in one scan.
			if !b.matcher.IsSorted() || !b2.matcher.IsSorted() {
				return append(in, b2)
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
			distinctPointUnion := false
			if boundaryIsPoint(a[i]) && boundaryIsPoint(cb) {
				samePoint := a[i].lowerBatch && cb.lowerBatch &&
					a[i].lowerBatchSubidx == cb.lowerBatchSubidx
				if !a[i].lowerBatch && !cb.lowerBatch {
					samePoint = boundaryValueEqual(a[i].lower, cb.lower)
				}
				distinctPointUnion = !samePoint
			}
			literalPointUnion := distinctPointUnion && !a[i].lowerBatch && !cb.lowerBatch
			// matcher demotion: OR takes the weaker matcher (range < like < equal)
			if !cb.matcher.IsPointLike() && a[i].matcher.IsPointLike() {
				a[i].matcher = cb.matcher
			}
			if literalPointUnion {
				// Nil is both SQL NULL and the legacy unbounded sentinel. While
				// widening two known points it is a real sortable value, so retain
				// the finite [min,max] interval instead of turning NULL OR value into
				// a full scan. The residual predicate removes values between points.
				lower, upper := a[i].lower, cb.lower
				if scm.Less(upper, lower) {
					lower, upper = upper, lower
				}
				a[i].lower, a[i].upper = lower, upper
				a[i].lowerInclusive, a[i].upperInclusive = true, true
				a[i].lowerBatch, a[i].upperBatch = false, false
				a[i].lowerBatchSubidx, a[i].upperBatchSubidx = 0, 0
			} else {
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
			}
			// The union of distinct equality points is a range. Keeping the
			// EqualMatcher here would let adaptive index ordering place this
			// widened column before real equality columns and make the B-tree
			// scan treat only the lower point as an exact prefix.
			if distinctPointUnion {
				a[i].matcher = RangeMatcher
			} else if matcherKindEqual(a[i].matcher, EqualMatcher) {
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

func conditionAnalyzeContext(conditionCols []string, condition scm.Scmer) (indexAnalyzeContext, bool) {
	var p *scm.Proc
	if condition.IsProc() {
		p = condition.Proc()
	} else if si, ok := condition.Any().(scm.Proc); ok {
		// fallback for legacy tagAny procs
		p = &si
	} else {
		return indexAnalyzeContext{}, false
	}
	var params []scm.Scmer
	if p.Params.IsSlice() {
		params = p.Params.Slice()
	}
	return indexAnalyzeContext{
		proc:          p,
		params:        params,
		conditionCols: conditionCols,
	}, true
}

func conditionMayHaveBoundaries(condition scm.Scmer) bool {
	if condition.IsProc() {
		return true
	}
	_, ok := condition.Any().(scm.Proc)
	return ok
}

func extractCustomBoundary(ctx indexAnalyzeContext, node scm.Scmer) (columnboundaries, bool) {
	for _, analyzer := range boundaryMatchers {
		if analyzer == EqualMatcher || analyzer == RangeMatcher {
			continue
		}
		if boundary, found := analyzer.Analyze(&ctx, node); found {
			return boundary, true
		}
	}
	return columnboundaries{}, false
}

func unwrapAnalyzeNode(ctx *indexAnalyzeContext, node scm.Scmer) scm.Scmer {
	for {
		items, sliced := scmerSlice(node)
		if !sliced || len(items) != 2 || !ctx.FunctionIs(items[0], "optimize") {
			return node
		}
		node = items[1]
	}
}

// extractSingleBoundary recognizes one indexable predicate without
// constructing the recursive general-analyzer closure or an intermediate
// singleton slice.
func extractSingleBoundary(ctx *indexAnalyzeContext, node scm.Scmer) (columnboundaries, bool) {
	node = unwrapAnalyzeNode(ctx, node)
	v, ok := scmerSlice(node)
	if !ok || len(v) < 2 {
		return columnboundaries{}, false
	}
	makeComparison := func(columnNode, valueNode scm.Scmer, matcher IndexAnalyzer, lower bool, inclusive bool) (columnboundaries, bool) {
		col, columnOK := ctx.ResolveColumn(columnNode)
		if !columnOK {
			return columnboundaries{}, false
		}
		bound := columnboundaries{col: col, matcher: matcher}
		if value, constant := ctx.ExtractConstant(valueNode); constant {
			if matcher == EqualMatcher {
				bound.lower, bound.upper = value, value
				bound.lowerInclusive, bound.upperInclusive = true, true
			} else if lower {
				bound.lower, bound.lowerInclusive = value, inclusive
			} else {
				bound.upper, bound.upperInclusive = value, inclusive
			}
			return bound, true
		}
		if subidx, batch := ctx.resolveBatchSubidx(valueNode); batch {
			if matcher == EqualMatcher {
				bound.lowerBatch, bound.upperBatch = true, true
				bound.lowerBatchSubidx, bound.upperBatchSubidx = subidx, subidx
				bound.lowerInclusive, bound.upperInclusive = true, true
			} else if lower {
				bound.lowerBatch, bound.lowerBatchSubidx, bound.lowerInclusive = true, subidx, inclusive
			} else {
				bound.upperBatch, bound.upperBatchSubidx, bound.upperInclusive = true, subidx, inclusive
			}
			return bound, true
		}
		return columnboundaries{}, false
	}
	if len(v) >= 3 && (ctx.FunctionIs(v[0], "equal?") || ctx.FunctionIs(v[0], "equal??")) {
		if bound, found := makeComparison(v[1], v[2], EqualMatcher, true, true); found {
			return bound, true
		}
		return makeComparison(v[2], v[1], EqualMatcher, true, true)
	}
	if len(v) >= 3 && (ctx.FunctionIs(v[0], "<") || ctx.FunctionIs(v[0], "<=")) {
		inclusive := ctx.FunctionIs(v[0], "<=")
		if bound, found := makeComparison(v[1], v[2], RangeMatcher, false, inclusive); found {
			return bound, true
		}
		return makeComparison(v[2], v[1], RangeMatcher, true, inclusive)
	}
	if len(v) >= 3 && (ctx.FunctionIs(v[0], ">") || ctx.FunctionIs(v[0], ">=")) {
		inclusive := ctx.FunctionIs(v[0], ">=")
		if bound, found := makeComparison(v[1], v[2], RangeMatcher, true, inclusive); found {
			return bound, true
		}
		return makeComparison(v[2], v[1], RangeMatcher, false, inclusive)
	}
	if ctx.FunctionIs(v[0], "nil?") {
		if col, found := ctx.ResolveColumn(v[1]); found {
			return columnboundaries{col: col, matcher: EqualMatcher, lower: scm.NewNil(), lowerInclusive: true, upper: scm.NewNil(), upperInclusive: true}, true
		}
	}
	return extractCustomBoundary(*ctx, node)
}

// extractSimpleBoundaries appends the dominant simple predicate class directly
// into caller-owned storage. Flat or nested AND trees remain allocation-free;
// OR widening and computed-column synthesis retain the complete general path.
func extractSimpleBoundaries(ctx *indexAnalyzeContext, node scm.Scmer, storage boundaries) (boundaries, bool) {
	node = unwrapAnalyzeNode(ctx, node)
	items, sliced := scmerSlice(node)
	if sliced && len(items) > 1 && items[0].SymbolEquals("and") {
		result := storage
		for _, child := range items[1:] {
			var ok bool
			result, ok = extractSimpleBoundaries(ctx, child, result)
			if !ok {
				return storage, false
			}
		}
		return result, true
	}
	boundary, ok := extractSingleBoundary(ctx, node)
	if !ok {
		return storage, false
	}
	return addConstraint(storage, boundary), true
}

func extractBoundariesInto(storage boundaries, conditionCols []string, condition scm.Scmer) boundaries {
	ctx, ok := conditionAnalyzeContext(conditionCols, condition)
	if ok {
		if result, simple := extractSimpleBoundaries(&ctx, ctx.proc.Body, storage); simple {
			return result
		}
	}
	return append(storage, extractBoundariesGeneral(conditionCols, condition)...)
}

// analyzes a lambda expression for value boundaries, so the best index can be found
func extractBoundaries(conditionCols []string, condition scm.Scmer) boundaries {
	return extractBoundariesInto(nil, conditionCols, condition)
}

// extractBoundariesGeneral retains the complete recursive analyzer for OR
// widening, computed columns and any future shape not handled by the simple
// allocation-free path.
func extractBoundariesGeneral(conditionCols []string, condition scm.Scmer) boundaries {
	analyzeContextValue, ok := conditionAnalyzeContext(conditionCols, condition)
	if !ok {
		// native Go function - no boundary extraction possible (full scan)
		return nil
	}
	analyzeContext := &analyzeContextValue
	p := analyzeContext.proc
	params := analyzeContext.params
	// traverseCondition returns boundaries for a single AST node.
	// nil means "unknown node, no bounds extractable".
	// AND: merge children (intersect). OR: widen children (union).
	var traverseCondition func(scm.Scmer) boundaries
	traverseCondition = func(node scm.Scmer) boundaries {
		v, ok := scmerSlice(node)
		if !ok {
			return nil
		}
		if len(v) == 0 {
			return nil
		}
		if analyzeContext.FunctionIs(v[0], "optimize") && len(v) == 2 {
			return traverseCondition(v[1])
		}
		for _, analyzer := range boundaryMatchers {
			if boundary, ok := analyzer.Analyze(analyzeContext, node); ok {
				return boundaries{boundary}
			}
		}
		if analyzeContext.FunctionIs(v[0], "equal?") || analyzeContext.FunctionIs(v[0], "equal??") {
			if col, ok := analyzeContext.ResolveColumn(v[1]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// reversed: (equal? const col)
			if col, ok := analyzeContext.ResolveColumn(v[2]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lowerInclusive: true, upperInclusive: true, lowerBatch: true, lowerBatchSubidx: subidx, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if analyzeContext.FunctionIs(v[0], "<") || analyzeContext.FunctionIs(v[0], "<=") {
			incl := v[0].SymbolEquals("<=")
			if col, ok := analyzeContext.ResolveColumn(v[1]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx}}
				}
			}
			// reversed: (< const col) means col > const, (<= const col) means col >= const
			if col, ok := analyzeContext.ResolveColumn(v[2]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if analyzeContext.FunctionIs(v[0], ">") || analyzeContext.FunctionIs(v[0], ">=") {
			incl := v[0].SymbolEquals(">=")
			if col, ok := analyzeContext.ResolveColumn(v[1]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, lowerBatch: true, lowerBatchSubidx: subidx}}
				}
			}
			// reversed: (> const col) means col < const, (>= const col) means col <= const
			if col, ok := analyzeContext.ResolveColumn(v[2]); ok {
				if v2, ok := analyzeContext.ExtractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
				if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[2]); ok {
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
					if subidx, ok := analyzeContext.resolveBatchSubidx(v[1]); ok {
						canon := canonicalColName(v[2], params, conditionCols)
						mc, mf := buildComputedFn(v[2], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), lowerInclusive: false, upperInclusive: incl, upperBatch: true, upperBatchSubidx: subidx, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if analyzeContext.FunctionIs(v[0], "nil?") && len(v) >= 2 {
			// IS NULL: (nil? col)
			if col, ok := analyzeContext.ResolveColumn(v[1]); ok {
				return boundaries{columnboundaries{col: col, matcher: EqualMatcher, lower: scm.NewNil(), lowerInclusive: true, upper: scm.NewNil(), upperInclusive: true}}
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
			if len(params) > 0 && !hasSessionRead(node) && isRawDataset(params, node) {
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
					if !cb.matcher.IsSorted() {
						return nil
					}
				}
			}
			return result
		}
		// Fallback: if the whole expression is a pure function of row columns
		// (no comparison operator matched above), treat it as a computed bool column.
		// Boundary {true, true} means: only scan rows where the expression is true.
		if len(params) > 0 && !hasSessionRead(node) && isRawDataset(params, node) {
			canon := canonicalColName(node, params, conditionCols)
			mc, mf := buildComputedFn(node, p.Params, p.En, conditionCols)
			if !mf.IsNil() && mc != nil {
				return boundaries{columnboundaries{col: canon, matcher: EqualMatcher, lower: scm.NewBool(true), lowerInclusive: true, upper: scm.NewBool(true), upperInclusive: true, mapCols: mc, mapFn: mf}}
			}
		}
		return nil
	}
	cols := traverseCondition(p.Body)

	// Sort physical index boundaries first and candidate matchers last. The
	// latter consume the ordered RecID batches without participating in the
	// index key or binary-search prefix.
	if len(cols) > 1 {
		hybridsort.Slice(cols, func(i, j int) bool {
			iSorted := cols[i].matcher.IsSorted()
			jSorted := cols[j].matcher.IsSorted()
			if iSorted != jSorted {
				return iSorted
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
		iSorted := bounds[i].matcher.IsSorted()
		jSorted := bounds[j].matcher.IsSorted()
		if iSorted != jSorted {
			return iSorted
		}
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

func indexFromBoundariesInto(storage []scm.Scmer, cols boundaries) (lower []scm.Scmer, upperLast scm.Scmer) {
	if len(cols) > 0 {
		sortedEnd := 0
		for sortedEnd < len(cols) && cols[sortedEnd].matcher.IsSorted() {
			sortedEnd++
		}
		usableSorted := sortedEnd
		for usableSorted >= 2 {
			if !boundaryIsPoint(cols[usableSorted-2]) &&
				!(boundaryIsUnboundedOrder(cols[usableSorted-2]) && boundaryIsUnboundedOrder(cols[usableSorted-1])) {
				usableSorted--
			} else {
				break
			}
		}
		// Hooks form a usable suffix only when no physical boundary between the
		// retained prefix and that suffix had to be dropped.
		effectiveLen := usableSorted
		if usableSorted == sortedEnd {
			effectiveLen = len(cols)
		}
		if effectiveLen == 0 {
			return nil, scm.NewNil()
		}
		if cap(storage) >= effectiveLen {
			lower = storage[:effectiveLen]
		} else {
			lower = make([]scm.Scmer, effectiveLen)
		}
		for i, v := range cols[:effectiveLen] {
			lower[i] = v.lower
		}
		if usableSorted > 0 {
			upperLast = cols[usableSorted-1].upper
		}
	}
	return
}

func indexFromBoundaries(cols boundaries) ([]scm.Scmer, scm.Scmer) {
	return indexFromBoundariesInto(nil, cols)
}
