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

import (
	"math"
	"sort"
	"unsafe"

	"math/bits"
)

import "github.com/launix-de/memcp/scm"

// inMatcher provides exact integer membership candidates. Strings and mixed
// SQL coercions remain residual-only until their canonical ordering is proven.
type inMatcher struct{}

func (*inMatcher) Kind() string      { return "in" }
func (*inMatcher) IsSorted() bool    { return false }
func (*inMatcher) IsPointLike() bool { return false }

func (m *inMatcher) Analyze(ctx IndexAnalyzeContext, node scm.Scmer) (IndexBoundary, bool) {
	v, ok := scmerSlice(node)
	if !ok || len(v) != 3 || !ctx.FunctionIs(v[0], "sql_in") {
		return IndexBoundary{}, false
	}
	col, ok := ctx.ResolveColumn(v[2])
	if !ok {
		return IndexBoundary{}, false
	}
	values, ok := ctx.ExtractConstant(v[1])
	if !ok || !values.IsSlice() {
		return IndexBoundary{}, false
	}
	return NewIndexBoundary(col, m, values, ""), true
}

// Preserve full BIGINT precision; generic Scheme Less compares via float64.
func integerInLess(a, b scm.Scmer) bool {
	if a.IsNil() {
		return !b.IsNil()
	}
	if b.IsNil() {
		return false
	}
	return a.Int() < b.Int()
}

func integerInValue(value scm.Scmer) bool {
	if value.IsInt() {
		return true
	}
	// StorageSeq and Scheme numeric literals use float Scmers even for SQL
	// integer columns. Admit only exactly represented integral values; beyond
	// 2^53, float/int equality can overlap distinct integer probe ranges.
	if !value.IsFloat() {
		return false
	}
	n := value.Float()
	return n > -9007199254740992 && n < 9007199254740992 && n == math.Trunc(n)
}

func (*inMatcher) Deploy(ctx IndexDeployContext, persistent bool) IndexHook {
	if !persistent || ctx.Column == nil || ctx.MainCount == 0 {
		return nil
	}
	values := make([]scm.Scmer, ctx.MainCount)
	ctx.Column.GetValueRange(0, ctx.MainCount, values, 1)
	for _, value := range values {
		if !value.IsNil() && !integerInValue(value) {
			return sharedNoopIndexHook
		}
	}
	ids := make([]uint32, ctx.MainCount)
	for i := range ids {
		ids[i] = uint32(i)
	}
	sort.Slice(ids, func(i, j int) bool { return integerInLess(values[ids[i]], values[ids[j]]) })
	h := &inIndexHook{universe: ctx.MainCount}
	h.positions.prepare()
	for i, id := range ids {
		h.positions.scan(uint32(i), scm.NewInt(int64(id)))
	}
	h.positions.init(ctx.MainCount)
	for i, id := range ids {
		h.positions.build(uint32(i), scm.NewInt(int64(id)))
	}
	h.positions.finish()
	return h
}

// positions is an immutable compressed main-generation permutation. It owns
// no shard/catalog pointer, reader, transaction, or request-local values.
type inIndexHook struct {
	universe  uint32
	positions StorageInt
}

func (*inIndexHook) Bind(scm.Scmer) IndexRowMatcher { return nil }
func (h *inIndexHook) ComputeSize() uint {
	return uint(unsafe.Sizeof(*h)) + h.positions.ComputeSize() - uint(unsafe.Sizeof(h.positions))
}

func (h *inIndexHook) BindCandidates(lower scm.Scmer, column ColumnReader, spanRows int) IndexCandidateIterator {
	if !lower.IsSlice() || column == nil {
		return nil
	}
	values := lower.Slice()
	// Include duplicate checking in the setup work estimate. Large lists use
	// the existing sequential kernel instead of quadratic duplicate checks.
	work := int64(len(values)) * (int64(len(values)) + int64(bits.Len32(h.universe))*2)
	if work >= int64(spanRows) {
		return nil
	}
	for _, value := range values {
		if !value.IsNil() && !integerInValue(value) {
			return nil
		}
	}
	recid := func(position int) uint32 {
		return uint32(int64(h.positions.GetValueUInt(uint32(position))) + h.positions.offset)
	}
	return func(buf []uint32, callback func([]uint32) bool) bool {
		count := 0
		for i, value := range values {
			if value.IsNil() {
				continue
			}
			duplicate := false
			for _, previous := range values[:i] {
				if !previous.IsNil() && value.Int() == previous.Int() {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			start := sort.Search(int(h.universe), func(position int) bool {
				return !integerInLess(column.GetValue(recid(position)), value)
			})
			end := start + sort.Search(int(h.universe)-start, func(offset int) bool {
				return integerInLess(value, column.GetValue(recid(start+offset)))
			})
			for position := start; position < end; position++ {
				buf[count] = recid(position)
				count++
				if count == len(buf) {
					if !callback(buf[:count]) {
						return false
					}
					count = 0
				}
			}
		}
		return count == 0 || callback(buf[:count])
	}
}
