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
}

type boundaries []columnboundaries

// addConstraint merges a column boundary into an existing set, narrowing the
// range for an already-present column (AND semantics) or appending a new entry.
func addConstraint(in boundaries, b2 columnboundaries) boundaries {
	for i, b := range in {
		if b.col == b2.col {
			// lower: pick the tighter (higher) bound
			if b.lower.IsNil() || (!b2.lower.IsNil() && scm.Less(b.lower, b2.lower)) {
				in[i].lower = b2.lower
				in[i].lowerInclusive = b2.lowerInclusive
			} else if !b.lower.IsNil() && !b2.lower.IsNil() && scm.Equal(b.lower, b2.lower) {
				in[i].lowerInclusive = b.lowerInclusive && b2.lowerInclusive
			}
			// upper: pick the tighter (lower) bound
			if b.upper.IsNil() || (!b2.upper.IsNil() && scm.Less(b2.upper, b.upper)) {
				in[i].upper = b2.upper
				in[i].upperInclusive = b2.upperInclusive
			} else if !b.upper.IsNil() && !b2.upper.IsNil() && scm.Equal(b.upper, b2.upper) {
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
			} else if scm.Equal(cb.lower, a[i].lower) {
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
			} else if scm.Equal(a[i].upper, cb.upper) {
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
	symbolmapping := make(map[scm.Symbol]string)
	if p.Params.IsSlice() {
		for i, sym := range p.Params.Slice() {
			symbolmapping[mustSymbolValue(sym)] = conditionCols[i]
		}
	}
	// analyze condition for AND clauses, equal? < > <= >= BETWEEN
	extractConstant := func(v scm.Scmer) (scm.Scmer, bool) {
		if v.IsInt() || v.IsFloat() || v.IsString() {
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
			if len(val) > 0 && val[0].IsSymbol() && val[0].String() == "outer" {
				sym := mustSymbolValue(val[1])
				if val2, ok := p.En.Vars[sym]; ok {
					if val2.IsInt() || val2.IsFloat() || val2.IsString() {
						return val2, true
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
		if len(v) == 0 || !v[0].IsSymbol() {
			return nil
		}
		if v[0].SymbolEquals("equal?") || v[0].SymbolEquals("equal??") {
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[2]); ok {
						return boundaries{columnboundaries{col, v2, true, v2, true}}
					}
				}
			}
			// reversed: (equal? const col)
			if v[2].IsSymbol() {
				sym := scm.Symbol(v[2].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[1]); ok {
						return boundaries{columnboundaries{col, v2, true, v2, true}}
					}
				}
			}
			return nil
		} else if v[0].SymbolEquals("<") || v[0].SymbolEquals("<=") {
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[2]); ok {
						return boundaries{columnboundaries{col, scm.NewNil(), false, v2, v[0].SymbolEquals("<=")}}
					}
				}
			}
			// reversed: (< const col) means col > const, (<=  const col) means col >= const
			if v[2].IsSymbol() {
				sym := scm.Symbol(v[2].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[1]); ok {
						return boundaries{columnboundaries{col, v2, v[0].SymbolEquals("<="), scm.NewNil(), false}}
					}
				}
			}
			return nil
		} else if v[0].SymbolEquals(">") || v[0].SymbolEquals(">=") {
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[2]); ok {
						return boundaries{columnboundaries{col, v2, v[0].SymbolEquals(">="), scm.NewNil(), false}}
					}
				}
			}
			// reversed: (> const col) means col < const, (>= const col) means col <= const
			if v[2].IsSymbol() {
				sym := scm.Symbol(v[2].String())
				if col, ok := symbolmapping[sym]; ok {
					if v2, ok := extractConstant(v[1]); ok {
						return boundaries{columnboundaries{col, scm.NewNil(), false, v2, v[0].SymbolEquals(">=")}}
					}
				}
			}
			return nil
		} else if v[0].SymbolEquals("nil?") && len(v) >= 2 {
			// IS NULL: (nil? col)
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok {
					return boundaries{columnboundaries{col, scm.NewNil(), true, scm.NewNil(), true}}
				}
			}
			return nil
		} else if v[0].SymbolEquals("contains?") && len(v) >= 3 {
			// IN-list: (contains? (list 1 2 3) col) → range [min, max]
			if v[2].IsSymbol() {
				sym := scm.Symbol(v[2].String())
				if col, ok := symbolmapping[sym]; ok {
					if v[1].IsSlice() {
						items := v[1].Slice()
						if len(items) > 1 && items[0].IsSymbol() && items[0].String() == "list" {
							var lo, hi scm.Scmer
							found := false
							for _, item := range items[1:] {
								if c, ok := extractConstant(item); ok {
									if !found {
										lo = c
										hi = c
										found = true
									} else {
										if scm.Less(c, lo) {
											lo = c
										}
										if scm.Less(hi, c) {
											hi = c
										}
									}
								}
							}
							if found {
								return boundaries{columnboundaries{col, lo, true, hi, true}}
							}
						}
					}
				}
			}
			return nil
		} else if v[0].SymbolEquals("strlike") && len(v) >= 3 {
			// LIKE prefix: (strlike col "foo%" collation) → range [prefix, prefix+1)
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok {
					if pat, ok := extractConstant(v[2]); ok && pat.IsString() {
						pattern := pat.String()
						idx := strings.IndexAny(pattern, "%_")
						if idx > 0 {
							prefix := pattern[:idx]
							upperBytes := []byte(prefix)
							upperBytes[len(upperBytes)-1]++
							return boundaries{columnboundaries{col, scm.NewString(prefix), true, scm.NewString(string(upperBytes)), false}}
						}
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
		return nil
	}
	cols := traverseCondition(p.Body)

	// sort columns -> at first, the lower==upper alphabetically; then one lower!=upper according to best selectivity; discard the rest
	sort.Slice(cols, func(i, j int) bool {
		if scm.Equal(cols[i].lower, cols[i].upper) && !scm.Equal(cols[j].lower, cols[j].upper) {
			return true // put equal?-conditions leftmost
		}
		return cols[i].col < cols[j].col // otherwise: alphabetically
	})

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
		iEq := scm.Equal(bounds[i].lower, bounds[i].upper)
		jEq := scm.Equal(bounds[j].lower, bounds[j].upper)
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

func indexFromBoundaries(cols boundaries) (lower []scm.Scmer, upperLast scm.Scmer) {
	if len(cols) > 0 {
		//fmt.Println("conditions:", cols)
		// build up lower and upper bounds of index
		for {
			if len(cols) >= 2 && !scm.Equal(cols[len(cols)-2].lower, cols[len(cols)-2].upper) {
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
