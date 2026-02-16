/*
Copyright (C) 2023  Carl-Philip Hänsch

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
	cols := make([]columnboundaries, 0, 4)
	addConstraint := func(in []columnboundaries, b2 columnboundaries) []columnboundaries {
		for i, b := range in {
			if b.col == b2.col {
				// column match -> merge value range
				if b.lower.IsNil() || (!b2.lower.IsNil() && scm.Less(b.lower, b2.lower)) {
					// both values are ANDed, so take the higher value as lower bound
					in[i].lower = b2.lower
				}
				in[i].lowerInclusive = b.lowerInclusive || b2.lowerInclusive // TODO: check correctness
				if b.upper.IsNil() || (!b2.upper.IsNil() && scm.Less(b2.upper, b.upper)) {
					// the lower of both upper values will be the new upper bound
					in[i].upper = b2.upper
				}
				in[i].upperInclusive = b.upperInclusive || b2.upperInclusive // TODO: check correctness
				return in
			}
		}
		// else: append
		return append(in, b2)
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
	var traverseCondition func(scm.Scmer)
	traverseCondition = func(node scm.Scmer) {
		if !node.IsSlice() {
			return
		}
		v := node.Slice()
		if len(v) == 0 || !v[0].IsSymbol() {
			return
		}
		if v[0].SymbolEquals("equal?") || v[0].SymbolEquals("equal??") {
			// equi
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok { // left is a column
					if v2, ok := extractConstant(v[2]); ok { // right is a constant
						// ?equal var const
						cols = addConstraint(cols, columnboundaries{col, v2, true, v2, true})
					}
				}
			}
		} else if v[0].SymbolEquals("<") || v[0].SymbolEquals("<=") {
			// compare
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok { // left is a column
					if v2, ok := extractConstant(v[2]); ok { // right is a constant
						cols = addConstraint(cols, columnboundaries{col, scm.NewNil(), false, v2, v[0].SymbolEquals("<=")})
					}
				}
			}
		} else if v[0].SymbolEquals(">") || v[0].SymbolEquals(">=") {
			// compare
			if v[1].IsSymbol() {
				sym := scm.Symbol(v[1].String())
				if col, ok := symbolmapping[sym]; ok { // left is a column
					if v2, ok := extractConstant(v[2]); ok { // right is a constant
						cols = addConstraint(cols, columnboundaries{col, v2, v[0].SymbolEquals(">="), scm.NewNil(), false})
					}
				}
			}
		} else if v[0].SymbolEquals("and") {
			// AND -> recursive traverse
			for i := 1; i < len(v); i++ {
				traverseCondition(v[i])
			}
		}
		// TODO: OR and more complex analysis
	}
	traverseCondition(p.Body) // recursive analysis over condition

	// sort columns -> at first, the lower==upper alphabetically; then one lower!=upper according to best selectivity; discard the rest
	sort.Slice(cols, func(i, j int) bool {
		if scm.Equal(cols[i].lower, cols[i].upper) && !scm.Equal(cols[j].lower, cols[j].upper) {
			return true // put equal?-conditions leftmost
		}
		return cols[i].col < cols[j].col // otherwise: alphabetically
	})

	return cols
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
