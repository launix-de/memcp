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

type columnboundaries struct{
	col string
	lower scm.Scmer
	lowerInclusive bool
	upper scm.Scmer
	upperInclusive bool
}

type boundaries []columnboundaries

// analyzes a lambda expression for value boundaries, so the best index can be found
func extractBoundaries(conditionCols []string, condition scm.Scmer) boundaries {
	p := condition.(scm.Proc)
	symbolmapping := make(map[scm.Symbol]string)
	for i, sym := range p.Params.([]scm.Scmer) {
		symbolmapping[sym.(scm.Symbol)] = conditionCols[i]
	}
	cols := make([]columnboundaries, 0, 4)
	addConstraint := func(in []columnboundaries, b2 columnboundaries) []columnboundaries {
		for i, b := range in {
			if b.col == b2.col {
				// column match -> merge value range
				if b.lower == nil || b2.lower != nil && scm.Less(b.lower, b2.lower) {
					// both values are ANDed, so take the higher value as lower bound
					in[i].lower = b2.lower
				}
				in[i].lowerInclusive = b.lowerInclusive || b2.lowerInclusive // TODO: check correctness
				if b.upper == nil || b2.upper != nil && scm.Less(b2.upper, b.upper) {
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
		switch val := v.(type) {
			case float64, string:
				// equals column vs. constant
				return val, true
			case scm.Symbol:
				if val2, ok := condition.(scm.Proc).En.Vars[val]; ok {
					switch val3 := val2.(type) {
						// bound constant
						case float64, string:
							// equals column vs. constant
							return val3, true
					}
				}
			case []scm.Scmer:
				if val[0] == scm.Symbol("outer") {
					if sym, ok := val[1].(scm.Symbol); ok {
						if val2, ok := condition.(scm.Proc).En.Vars[sym]; ok {
							switch val3 := val2.(type) {
								// bound constant
								case float64, string:
									// equals column vs. constant
									return val3, true
							}
						}
					}
				}
		}
		return nil, false
	}
	var traverseCondition func(scm.Scmer)
	traverseCondition = func (node scm.Scmer) {
		switch v := node.(type) {
			case []scm.Scmer:
				if v[0] == scm.Symbol("equal?") || v[0] == scm.Symbol("equal??") {
					// equi
					switch v1 := v[1].(type) {
						case scm.Symbol:
							if col, ok := symbolmapping[v1]; ok { // left is a column
								if v2, ok := extractConstant(v[2]); ok { // right is a constant
									// ?equal var const
									cols = addConstraint(cols, columnboundaries{col, v2, true, v2, true})
								}
							}
						// TODO: equals constant vs. column
					}
				} else if v[0] == scm.Symbol("<") || v[0] == scm.Symbol("<=") {
					// compare
					switch v1 := v[1].(type) {
						case scm.Symbol:
							if col, ok := symbolmapping[v1]; ok { // left is a column
								if v2, ok := extractConstant(v[2]); ok { // right is a constant
									// ?equal var const
									cols = addConstraint(cols, columnboundaries{col, nil, false, v2, v[0] == scm.Symbol("<=")})
								}
							}
						// TODO: constant vs. column
					}
				} else if v[0] == scm.Symbol(">") || v[0] == scm.Symbol(">=") {
					// compare
					switch v1 := v[1].(type) {
						case scm.Symbol:
							if col, ok := symbolmapping[v1]; ok { // left is a column
								if v2, ok := extractConstant(v[2]); ok { // right is a constant
									// ?equal var const
									cols = addConstraint(cols, columnboundaries{col, v2, v[0] == scm.Symbol(">="), nil, false})
								}
							}
						// TODO: constant vs. column
					}
				} else if v[0] == scm.Symbol("and") {
					// AND -> recursive traverse
					for i := 1; i < len(v); i++ {
						traverseCondition(v[i])
					}
				}
				// TODO: <, >, <=, >=
				// TODO: OR -> merge multiple
				// TODO: variable expressions that can be expanded
		}
	}
	traverseCondition(p.Body) // recursive analysis over condition

	// sort columns -> at first, the lower==upper alphabetically; then one lower!=upper according to best selectivity; discard the rest
	sort.Slice(cols, func (i, j int) bool {
		if cols[i].lower == cols[i].upper && cols[j].lower != cols[j].upper {
			return true // put equal?-conditions leftmost
		}
		return cols[i].col < cols[j].col // otherwise: alphabetically
	})

	return cols
}

