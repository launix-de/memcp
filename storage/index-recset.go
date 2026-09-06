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

import "github.com/launix-de/memcp/scm"

type recSetMatcher struct{}

// appendRecSetBoundary turns a table-shaped RecSet source into an exact
// non-ordering boundary on its backing table. Scan operators retain one common
// table/index pipeline; choosing whether the base index or this RecSet drives
// iteration is an execution-kernel decision, not a second relational shape.
func appendRecSetBoundary(bounds analyzedBoundaries, source *recSet) analyzedBoundaries {
	if source == nil {
		return bounds
	}
	if source.table == nil {
		panic("recset scan source has no backing table")
	}
	value := NewRecSetScmer(source)
	return append(bounds, analyzedBoundary{
		col:            "$recset_contains",
		matcher:        RecSetMatcher,
		lower:          value,
		lowerInclusive: true,
		upper:          value,
		upperInclusive: true,
		mandatory:      true,
	})
}

// smallestRecSetBoundary returns the cheapest exact RecSet from a boundary
// suffix. Other RecSet boundaries remain ordinary matchers and are intersected
// by bindRowMatchers. This permits several independent RecSet hooks without
// creating another scan-source shape.
func smallestRecSetBoundary(bounds scanAccess, shard *storageShard) (*recSetShard, bool) {
	var smallest *recSetShard
	found := false
	for i := 0; i < bounds.len(); i++ {
		if !matcherKindEqual(bounds.boundaryAnalyzer(i), RecSetMatcher) {
			continue
		}
		lower := bounds.boundValue(i, false)
		if !lower.IsCustom(TagRecSet) {
			continue
		}
		source := RecSetFromScmer(lower)
		if source == nil || source.table != shard.t {
			continue
		}
		found = true
		part := source.shardEntry(shard)
		if part == nil {
			return nil, true
		}
		if smallest == nil || part.count < smallest.count {
			smallest = part
		}
	}
	return smallest, found
}

func (m *recSetMatcher) Kind() string      { return "recset" }
func (m *recSetMatcher) IsSorted() bool    { return false }
func (m *recSetMatcher) IsPointLike() bool { return true }

func (m *recSetMatcher) Analyze(ctx IndexAnalyzeContext, node scm.Scmer) (IndexBoundary, bool) {
	v, ok := scmerSlice(node)
	if !ok || len(v) != 2 {
		return IndexBoundary{}, false
	}
	col, ok := ctx.ResolveParameter(v[0])
	if !ok || col != "$recset_contains" {
		return IndexBoundary{}, false
	}
	value, ok := ctx.ExtractConstant(v[1])
	if !ok || !value.IsCustom(TagRecSet) {
		return IndexBoundary{}, false
	}
	return NewIndexBoundary(col, m, value, ""), true
}

func (m *recSetMatcher) Deploy(ctx IndexDeployContext, _ bool) IndexHook {
	return &recSetIndexHook{shard: ctx.shard}
}

type recSetIndexHook struct {
	shard *storageShard
}

func (h *recSetIndexHook) ComputeSize() uint { return 0 }

func (h *recSetIndexHook) EstimateCandidates(lower scm.Scmer) (uint32, uint32, bool) {
	if h == nil || h.shard == nil || !lower.IsCustom(TagRecSet) {
		return 0, 0, false
	}
	set := RecSetFromScmer(lower)
	if set == nil || set.table != h.shard.t {
		return 0, 0, false
	}
	part := set.shardEntry(h.shard)
	if part == nil {
		return 0, h.shard.main_count, true
	}
	return uint32(part.count), h.shard.main_count, true
}

func (h *recSetIndexHook) Bind(lower scm.Scmer) IndexRowMatcher {
	if h == nil || h.shard == nil || !lower.IsCustom(TagRecSet) {
		return nil
	}
	set := RecSetFromScmer(lower)
	if set == nil || set.table != h.shard.t {
		return nil
	}
	part := set.shardEntry(h.shard)
	return func(ids []uint32) []uint32 {
		out := ids[:0]
		for _, id := range ids {
			if part != nil && part.contains(id) {
				out = append(out, id)
			}
		}
		return out
	}
}
