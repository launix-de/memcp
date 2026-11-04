/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "unsafe"
import "github.com/launix-de/memcp/scm"

// Analyze only truth contexts which reject zero scores. Projection and filters
// admitting zero retain the complete relation, including nonmatching rows.
func analyzeTextSearch(ctx IndexAnalyzeContext, node scm.Scmer, matcher IndexAnalyzer) (IndexBoundary, bool) {
	v, ok := scmerSlice(node)
	if !ok || len(v) < 2 {
		return IndexBoundary{}, false
	}
	if len(v) == 3 && (ctx.FunctionIs(v[0], ">") || ctx.FunctionIs(v[0], ">=")) {
		threshold, constant := ctx.ExtractConstant(v[2])
		if !constant || !threshold.IsInt() && !threshold.IsFloat() {
			return IndexBoundary{}, false
		}
		if scm.ToFloat(threshold) < 0 || scm.ToFloat(threshold) == 0 && ctx.FunctionIs(v[0], ">=") {
			return IndexBoundary{}, false
		}
		v, ok = scmerSlice(v[1])
		if !ok {
			return IndexBoundary{}, false
		}
	}
	if len(v) != 3 || !ctx.FunctionIs(v[0], "text_search_score") {
		return IndexBoundary{}, false
	}
	terms, constant := ctx.ExtractConstant(v[2])
	if !constant || !terms.IsSlice() {
		return IndexBoundary{}, false
	}
	columns, ok := scmerSlice(v[1])
	if !ok || len(columns) < 2 || !ctx.FunctionIs(columns[0], "list") {
		return IndexBoundary{}, false
	}
	for _, column := range columns[1:] {
		if _, ok := ctx.ResolveColumn(column); !ok {
			return IndexBoundary{}, false
		}
	}
	binding := scm.NewSlice([]scm.Scmer{scm.NewString(":search"), terms})
	if len(columns) == 2 {
		column, _ := ctx.ResolveColumn(columns[1])
		return NewIndexBoundary(column, matcher, binding, "utf8mb4_general_ci"), true
	}
	// A canonical tuple identifies the set of source readers, not a score or
	// query-specific computed column. Deploy binds their existing posting caches.
	c := ctx.(*indexAnalyzeContext)
	cols, fn := buildComputedFn(v[1], c.proc.Params, c.proc.En, c.conditionCols)
	if fn.IsNil() || len(cols) != len(columns)-1 {
		return IndexBoundary{}, false
	}
	return analyzedBoundary{col: canonicalColName(v[1], c.params, c.conditionCols), matcher: matcher,
		lower: binding, collation: "utf8mb4_general_ci", mapCols: cols, mapFn: fn}, true
}

func textSearchBinding(value scm.Scmer) (scm.Scmer, bool) {
	if value.IsSlice() {
		v := value.Slice()
		if (len(v) == 2 || len(v) == 3) && v[0].String() == ":search" {
			if len(v) == 3 && !scm.ToBool(v[2]) {
				return scm.NewNil(), true
			}
			return v[1], true
		}
	}
	return scm.NewNil(), false
}

type textSearchIndex struct{ indexes []*bigramIndex }

func (s *textSearchIndex) ComputeSize() uint {
	size := uint(unsafe.Sizeof(*s)) + uint(cap(s.indexes))*uint(unsafe.Sizeof((*bigramIndex)(nil)))
	// Each retaining hook accounts for its sources, conservatively also when a
	// second live index shares them. The weak lookup itself never retains data.
	for _, index := range s.indexes {
		size += index.ComputeSize()
	}
	return size
}

func (s *textSearchIndex) Bind(binding scm.Scmer) IndexRowMatcher {
	terms, ok := textSearchBinding(binding)
	if !ok || terms.IsNil() {
		return nil
	}
	set, constrained := textSearchCandidates(s.indexes, terms)
	if !constrained {
		return nil
	}
	return textSearchRowMatcher(set, s.indexes[0].universe)
}

func textSearchRowMatcher(set recSetShard, universe uint32) IndexRowMatcher {
	return func(ids []uint32) []uint32 {
		out := ids[:0]
		for _, id := range ids {
			if id >= universe || int(id>>5) < len(set.data) && set.data[id>>5]&(uint32(1)<<(id&31)) != 0 {
				out = append(out, id)
			}
		}
		return out
	}
}

// Unknown postings (one-character terms, for example) mean the entire universe.
// Exclusions never subtract approximate candidates: a gram hit is not proof of
// a word or phrase hit. Delta rows always bypass this immutable main-row set.
func textSearchCandidates(indexes []*bigramIndex, terms scm.Scmer) (recSetShard, bool) {
	var required, optional recSetShard
	haveRequired, requiredConstrained := false, false
	haveOptional, optionalUnknown := false, false
	for _, term := range terms.Slice() {
		t := term.Slice()
		if scm.ToFloat(t[0]) <= 0 || t[2].Bool() {
			continue
		}
		var candidate recSetShard
		constrained := false
		if t[3].String() == "group" {
			candidate, constrained = textSearchCandidates(indexes, t[4])
		} else {
			runs := []string{t[4].String()}
			if t[3].String() == "phrase" {
				runs = nil
				for _, word := range t[4].Slice() {
					runs = append(runs, word.String())
				}
			}
			constrained = true
			for _, index := range indexes {
				part, known := index.candidatesLiterals(runs)
				if !known {
					constrained = false
					break
				}
				candidate = combineTextSearchSets(candidate, part, false)
			}
		}
		if t[1].Bool() {
			haveRequired = true
			if constrained {
				if !requiredConstrained {
					required = candidate
				} else {
					required = combineTextSearchSets(required, candidate, true)
				}
				requiredConstrained = true
			}
		} else {
			haveOptional = true
			if !constrained {
				optionalUnknown = true
			} else {
				optional = combineTextSearchSets(optional, candidate, false)
			}
		}
	}
	if haveRequired {
		return required, requiredConstrained
	}
	if !haveOptional {
		return recSetShard{universe: indexes[0].universe}, true
	}
	return optional, !optionalUnknown
}

func combineTextSearchSets(left, right recSetShard, intersect bool) recSetShard {
	if left.universe == 0 {
		return right
	}
	words := make([]uint32, (left.universe+31)/32)
	for i := range words {
		var a, b uint32
		if i < len(left.data) {
			a = left.data[i]
		}
		if i < len(right.data) {
			b = right.data[i]
		}
		if intersect {
			words[i] = a & b
		} else {
			words[i] = a | b
		}
	}
	return recSetShard{kind: recSetBitmap, universe: left.universe, data: words}
}
