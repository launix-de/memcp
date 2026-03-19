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

import "sort"
import "github.com/carli2/hybridsort"
import "strings"
import "github.com/launix-de/memcp/scm"

func mustSymbolValue(v scm.Scmer) scm.Symbol {
	if v.IsSymbol() {
		return scm.Symbol(v.String())
	}
	panic("expected symbol")
}

type columnboundaries struct {
	col            string
	lower          scm.Scmer
	lowerInclusive bool
	upper          scm.Scmer
	upperInclusive bool
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

// boundaryIsPoint reports whether a boundary represents an exact point lookup.
// Special case: nil..nil is only a point for explicit IS NULL bounds where both
// inclusiveness flags are true; unbounded placeholders use nil..nil with false flags.
func boundaryIsPoint(b columnboundaries) bool {
	if !boundaryValueEqual(b.lower, b.upper) {
		return false
	}
	if b.lower.IsNil() && b.upper.IsNil() {
		return b.lowerInclusive && b.upperInclusive
	}
	return true
}

// addConstraint merges a column boundary into an existing set, narrowing the
// range for an already-present column (AND semantics) or appending a new entry.
func addConstraint(in boundaries, b2 columnboundaries) boundaries {
	for i, b := range in {
		if b.col == b2.col {
			// lower: pick the tighter (higher) bound
			if b.lower.IsNil() || (!b2.lower.IsNil() && scm.Less(b.lower, b2.lower)) {
				in[i].lower = b2.lower
				in[i].lowerInclusive = b2.lowerInclusive
			} else if !b.lower.IsNil() && !b2.lower.IsNil() && boundaryValueEqual(b.lower, b2.lower) {
				in[i].lowerInclusive = b.lowerInclusive && b2.lowerInclusive
			}
			// upper: pick the tighter (lower) bound
			if b.upper.IsNil() || (!b2.upper.IsNil() && scm.Less(b2.upper, b.upper)) {
				in[i].upper = b2.upper
				in[i].upperInclusive = b2.upperInclusive
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
			// widen lower: take the smaller
			if a[i].lower.IsNil() {
				// already unbounded
			} else if cb.lower.IsNil() {
				a[i].lower = scm.NewNil()
				a[i].lowerInclusive = false
			} else if scm.Less(cb.lower, a[i].lower) {
				a[i].lower = cb.lower
				a[i].lowerInclusive = cb.lowerInclusive
			} else if boundaryValueEqual(cb.lower, a[i].lower) {
				a[i].lowerInclusive = a[i].lowerInclusive || cb.lowerInclusive
			}
			// widen upper: take the larger
			if a[i].upper.IsNil() {
				// already unbounded
			} else if cb.upper.IsNil() {
				a[i].upper = scm.NewNil()
				a[i].upperInclusive = false
			} else if scm.Less(a[i].upper, cb.upper) {
				a[i].upper = cb.upper
				a[i].upperInclusive = cb.upperInclusive
			} else if boundaryValueEqual(a[i].upper, cb.upper) {
				a[i].upperInclusive = a[i].upperInclusive || cb.upperInclusive
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
	// resolveColVar maps a node to a column name.
	// Handles both symbol params (linear scan, no alloc) and NthLocalVar(i).
	resolveColVar := func(node scm.Scmer) (string, bool) {
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
	// analyze condition for AND clauses, equal? < > <= >= BETWEEN
	extractConstant := func(v scm.Scmer) (scm.Scmer, bool) {
		if v.IsInt() || v.IsFloat() || v.IsString() || v.IsBool() {
			return v, true
		}
		if v.IsSymbol() {
			if val2, ok := p.En.Vars[scm.Symbol(v.String())]; ok {
				if val2.IsInt() || val2.IsFloat() || val2.IsString() {
					return val2, true
				}
			}
		}
		if v.IsSlice() {
			val := v.Slice()
			if len(val) > 0 && val[0].SymbolEquals("outer") {
				if val[1].IsSymbol() {
					sym := scm.Symbol(val[1].String())
					if val2, ok := p.En.Vars[sym]; ok {
						if val2.IsInt() || val2.IsFloat() || val2.IsString() {
							return val2, true
						}
					}
				} else if val[1].IsNthLocalVar() {
					// (outer NthLocalVar(i)) — free variable from outer captured environment
					idx := int(val[1].NthLocalVar())
					if p.En.VarsNumbered != nil && idx < len(p.En.VarsNumbered) {
						val2 := p.En.VarsNumbered[idx]
						if val2.IsInt() || val2.IsFloat() || val2.IsString() {
							return val2, true
						}
					}
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
		if funcIs(v[0], "equal?") || funcIs(v[0], "equal??") {
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
			}
			// reversed: (equal? const col)
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true}}
				}
			}
			// computed col: (equal? rawDataset independent) or reversed
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true, mapCols: mc, mapFn: mf}}
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
							return boundaries{columnboundaries{col: canon, lower: v2, lowerInclusive: true, upper: v2, upperInclusive: true, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], "<") || funcIs(v[0], "<=") {
			incl := v[0].SymbolEquals("<=")
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
			}
			// reversed: (< const col) means col > const, (<= const col) means col >= const
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
			}
			// computed col: rawDataset < independent → rawDataset has upper bound
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl, mapCols: mc, mapFn: mf}}
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
							return boundaries{columnboundaries{col: canon, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], ">") || funcIs(v[0], ">=") {
			incl := v[0].SymbolEquals(">=")
			if col, ok := resolveColVar(v[1]); ok {
				if v2, ok := extractConstant(v[2]); ok {
					return boundaries{columnboundaries{col: col, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false}}
				}
			}
			// reversed: (> const col) means col < const, (>= const col) means col <= const
			if col, ok := resolveColVar(v[2]); ok {
				if v2, ok := extractConstant(v[1]); ok {
					return boundaries{columnboundaries{col: col, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl}}
				}
			}
			// computed col: rawDataset > independent → rawDataset has lower bound
			if len(params) > 0 && v[1].IsSlice() {
				if isRawDataset(params, v[1]) && isIndependent(params, v[2]) {
					if v2, ok2 := evalIndependentScmer(v[2], p.En); ok2 {
						canon := canonicalColName(v[1], params, conditionCols)
						mc, mf := buildComputedFn(v[1], p.Params, p.En, conditionCols)
						if !mf.IsNil() && mc != nil {
							return boundaries{columnboundaries{col: canon, lower: v2, lowerInclusive: incl, upper: scm.NewNil(), upperInclusive: false, mapCols: mc, mapFn: mf}}
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
							return boundaries{columnboundaries{col: canon, lower: scm.NewNil(), lowerInclusive: false, upper: v2, upperInclusive: incl, mapCols: mc, mapFn: mf}}
						}
					}
				}
			}
			return nil
		} else if funcIs(v[0], "nil?") && len(v) >= 2 {
			// IS NULL: (nil? col)
			if col, ok := resolveColVar(v[1]); ok {
				return boundaries{columnboundaries{col: col, lower: scm.NewNil(), lowerInclusive: true, upper: scm.NewNil(), upperInclusive: true}}
			}
			return nil
		} else if funcIs(v[0], "strlike") && len(v) >= 3 {
			// LIKE prefix: (strlike col "foo%" collation) → range [prefix, prefix+1)
			if col, ok := resolveColVar(v[1]); ok {
				if pat, ok := extractConstant(v[2]); ok && pat.IsString() {
					pattern := pat.String()
					idx := strings.IndexAny(pattern, "%_")
					if idx > 0 {
						prefix := pattern[:idx]
						upperBytes := []byte(prefix)
						upperBytes[len(upperBytes)-1]++
						return boundaries{columnboundaries{col: col, lower: scm.NewString(prefix), lowerInclusive: true, upper: scm.NewString(string(upperBytes)), upperInclusive: false}}
					}
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
					return boundaries{columnboundaries{col: canon, lower: scm.NewBool(true), lowerInclusive: true, upper: scm.NewBool(true), upperInclusive: true, mapCols: mc, mapFn: mf}}
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
				return boundaries{columnboundaries{col: canon, lower: scm.NewBool(true), lowerInclusive: true, upper: scm.NewBool(true), upperInclusive: true, mapCols: mc, mapFn: mf}}
			}
		}
		return nil
	}
	cols := traverseCondition(p.Body)

	// Sort columns so that equality conditions come first, remainder alphabetically.
	// This canonical ordering serves two purposes:
	//   1. Deduplication: queries with the same equality columns in different AST order
	//      (e.g. "WHERE a=1 AND b=2" vs "WHERE b=2 AND a=1") map to the same column
	//      sequence and thus reuse the same adaptive index instead of creating duplicates.
	//   2. Selectivity: placing equality columns as the index key prefix lets the shard
	//      skip directly to the matching bucket before applying any range bound, which
	//      reduces both the scan window and memory pressure during index lookup.
	// Precompute isEq to avoid repeated scm.Equal calls inside the sort comparator.
	if len(cols) > 1 {
		isEq := make([]bool, len(cols))
		for i := range cols {
			isEq[i] = boundaryIsPoint(cols[i])
		}
		hybridsort.Slice(cols, func(i, j int) bool {
			if isEq[i] != isEq[j] {
				return isEq[i] // equality conditions leftmost
			}
			return cols[i].col < cols[j].col // tiebreak alphabetically
		})
	}

	return cols
}

// reorderByFrequency re-sorts equality columns by query frequency (most-used first)
// so that the most-queried columns appear first in the index key, maximizing prefix
// overlap across queries. Also bumps the frequency counters for each boundary column.
func reorderByFrequency(bounds boundaries, t *table) {
	for _, b := range bounds {
		t.bumpColFreq(b.col)
	}
	sort.SliceStable(bounds, func(i, j int) bool {
		iEq := boundaryIsPoint(bounds[i])
		jEq := boundaryIsPoint(bounds[j])
		if iEq != jEq {
			return iEq // equality first
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
	OrcSuffixOpaque         = 0 // can't analyze → full recompute only
	OrcSuffixIdentity       = 1 // acc == emitted value (SUM, ROW_NUMBER) → stored value is accumulator
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
//   return value = (+ acc X) where X depends only on mapped, not acc.
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
	var setArg scm.Scmer   // the value passed to $set
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
		//fmt.Println("conditions:", cols)
		// build up lower and upper bounds of index
		for {
			if len(cols) >= 2 && !boundaryIsPoint(cols[len(cols)-2]) {
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
