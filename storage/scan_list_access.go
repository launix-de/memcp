/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "github.com/launix-de/memcp/scm"

// List sources share the physical access contract with native tables. Exact
// equal/range predicates may have been removed from the residual callback;
// enforce those constraints before filtering, ordering or taking a window.
// Candidate-only hooks remain advisory and their original callback still runs.
type listScanConstraint struct {
	column                                                    string
	lower, upper                                              scm.Scmer
	lowerSet, upperSet, lowerInclusive, upperInclusive, point bool
	less                                                      func(scm.Scmer, scm.Scmer) bool
	mapColumns                                                []string
	mapper                                                    scm.SerialProc
	mapped                                                    bool
	mapValues                                                 []scm.Scmer
}

type listScanAccess struct {
	impossible  bool
	constraints []listScanConstraint
	batch       []scm.Scmer
}

func bindListScanAccess(schema scm.Scmer, values []scm.Scmer) listScanAccess {
	access, valid := scanAccessFromScheme(schema, values, nil)
	if !valid {
		panic("invalid list scan access schema")
	}
	return prepareListScanAccess(access, nil)
}

func prepareListScanAccess(access scanAccess, batch []scm.Scmer) listScanAccess {
	result := listScanAccess{impossible: access.impossible(), batch: batch}
	for i := 0; i < access.len(); i++ {
		analyzer := access.boundaryAnalyzer(i)
		if analyzer != EqualMatcher && analyzer != RangeMatcher {
			if access.boundaryMandatory(i) {
				panic("list scan cannot enforce mandatory candidate hook: " + analyzer.Kind())
			}
			continue
		}
		lower, upper := access.boundValue(i, false), access.boundValue(i, true)
		point := analyzer == EqualMatcher
		lowerSet, upperSet := scanAccessBoundaryHasLowerBound(access, i), scanAccessBoundaryHasUpperBound(access, i)
		if !point && !lowerSet && !upperSet {
			continue
		}
		c := listScanConstraint{column: access.boundaryColumn(i), lower: lower, upper: upper,
			lowerSet: lowerSet, upperSet: upperSet, lowerInclusive: access.boundaryLowerInclusive(i), upperInclusive: access.boundaryUpperInclusive(i), point: point, less: scm.Less}
		if collation := access.boundaryCollation(i); collation != "" {
			order := scm.Apply(scm.Globalenv.Vars["collate"], scm.NewString(collation), scm.NewBool(false))
			c.less = scm.OrderRelationLess(order.Func())
		}
		columns, mapper := access.boundaryMap(i)
		if !mapper.IsNil() {
			c.mapColumns = columns
			c.mapper = scm.PrepareSerialProc(mapper)
			c.mapped = true
			c.mapValues = make([]scm.Scmer, len(columns))
		}
		result.constraints = append(result.constraints, c)
	}
	return result
}

func (access *listScanAccess) matches(row dataset) bool {
	if access.impossible {
		return false
	}
	for i := range access.constraints {
		c := &access.constraints[i]
		value, _ := row.GetI(c.column)
		if c.mapped {
			for j, column := range c.mapColumns {
				if index, ok := parseBatchPseudoColName(column); ok && access.batch != nil {
					c.mapValues[j] = access.batch[index]
				} else {
					c.mapValues[j], _ = row.GetI(column)
				}
			}
			value = c.mapper.Call(c.mapValues)
		}
		if c.point {
			if c.less(value, c.lower) || c.less(c.lower, value) {
				return false
			}
			continue
		}
		if c.lowerSet && (c.less(value, c.lower) || (!c.lowerInclusive && !c.less(c.lower, value))) {
			return false
		}
		if c.upperSet && (c.less(c.upper, value) || (!c.upperInclusive && !c.less(value, c.upper))) {
			return false
		}
	}
	return true
}
