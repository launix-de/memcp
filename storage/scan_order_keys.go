/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "math"
import "sort"
import "time"
import "container/heap"
import "github.com/google/btree"
import "github.com/launix-de/memcp/scm"

// orderedPrefixAccess is immutable query-local data. Cursor positions and
// borrowed readers live only inside iteratePrefixMerge, under the caller's
// shard read rights and RLock. Nothing here is persisted or retained by indexes.
type orderedPrefixAccess struct {
	column string
	keys   recSetProjectKeys
	order  []string
	dirs   []func(...scm.Scmer) scm.Scmer
}

func declareScanOrderKeys(en *scm.Env, ordinary *scm.Declaration) {
	declaration := *ordinary
	declaration.Name = "scan_order_keys"
	descriptor := *ordinary.Type
	descriptor.Description = "ordered scan over a dynamic projected keyset; chooses prefix merge, ordered membership, or RecSet projection from invocation cardinalities"
	descriptor.Params = append([]*scm.TypeDescriptor(nil), ordinary.Type.Params...)
	descriptor.Params[1] = &scm.TypeDescriptor{Kind: "list", Label: "source", Description: "(target table, source RecSet, source key columns, target key columns, optional cost prices)"}
	declaration.Type = &descriptor
	declaration.Fn = func(a ...scm.Scmer) scm.Scmer {
		source := mustScmerSlice(a[1], "scan_order_keys source")
		if len(source) != 4 && len(source) != 5 {
			panic("scan_order_keys: expected target, source RecSet, source keys, target keys")
		}
		target := TableFromScmer(source[0])
		sourceKeys := scmerSliceToStrings(mustScmerSlice(source[2], "source keys"))
		targetKeys := scmerSliceToStrings(mustScmerSlice(source[3], "target keys"))
		if len(sourceKeys) == 0 || len(sourceKeys) != len(targetKeys) {
			panic("scan_order_keys: key widths must match and be nonzero")
		}
		tx := scmerToTxContext(a[0])
		keyStart := time.Now()
		keys := RecSetFromScmer(source[1]).collectProjectJoinKeys(tx, sourceKeys, SessionStateFromTx(tx))
		if Settings.ScanDebugging {
			fmt.Printf("[SCAN_ORDER_KEYS] key_build_ns=%d\n", time.Since(keyStart).Nanoseconds())
		}
		fallback := func() scm.Scmer {
			if Settings.ScanDebugging {
				fmt.Printf("[SCAN_ORDER_KEYS] projected keys=%d width=%d\n", keys.count(), keys.width)
			}
			args := append([]scm.Scmer(nil), a...)
			args[1] = NewRecSetScmer(target.projectJoinKeysToRecSet(tx, targetKeys, keys, SessionStateFromTx(tx)))
			return ordinary.Fn(args...)
		}
		if keys.count() == 0 {
			return fallback()
		}
		if len(targetKeys) != 1 || scm.ToInt(a[8]) != 0 || !keys.buildNumericLookup() {
			return fallback()
		}
		// Numeric prefix ordering must agree with projection equality. Large or
		// coerced keys retain the established projection path rather than weakening it.
		for _, key := range keys.values {
			if !key.IsInt() || key.Int() > 1<<53 || key.Int() < -(1<<53) {
				return fallback()
			}
		}
		sortValues := mustScmerSlice(a[6], "sort columns")
		order := make([]string, len(sortValues))
		for i, v := range sortValues {
			if !v.IsString() {
				return fallback()
			}
			order[i] = v.String()
		}
		if len(order) == 0 {
			return fallback()
		}
		bounds, ok := scanAccessFromScheme(a[2], mustScmerSlice(a[3], "access values"), nil)
		if !ok {
			panic("scan_order_keys: invalid compiled access schema")
		}
		for i := 0; i < bounds.len(); i++ {
			if bounds.boundaryAnalyzer(i) != RangeMatcher || !bounds.boundValue(i, false).IsNil() || !bounds.boundValue(i, true).IsNil() {
				return fallback()
			}
		}
		rows := float64(target.CountEstimate())
		distinct := rows
		integer := false
		// Metadata only: no shard load, index build, or statistics traversal here.
		for _, row := range target.ShowColumns().Slice() {
			fields := row.Slice()
			name, typ := "", ""
			ndv := float64(0)
			for i := 0; i+1 < len(fields); i += 2 {
				switch scm.String(fields[i]) {
				case "Field":
					name = scm.String(fields[i+1])
				case "RawType":
					typ = scm.String(fields[i+1])
				case "DistinctEstimate":
					ndv = scm.ToFloat(fields[i+1])
				}
			}
			if name == targetKeys[0] {
				integer = typ == "int" || typ == "integer" || typ == "bigint"
				if ndv > 0 {
					distinct = ndv
				}
				break
			}
		}
		if !integer {
			return fallback()
		}
		prefix := &orderedPrefixAccess{column: targetKeys[0], keys: keys, order: order, dirs: scanSortDirections(mustScmerSlice(a[7], "sort directions"))}
		filterCols := scmerSliceToStrings(mustScmerSlice(a[4], "filter columns"))
		filter := a[5]
		membership := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			if !keys.contains(values[:1]) {
				return scm.NewBool(false)
			}
			return scm.Apply(filter, values[1:]...)
		})
		window := float64(scm.ToInt(a[9]) + scm.ToInt(a[10]))
		if scm.ToInt(a[10]) < 0 {
			window = rows
		}
		fraction := math.Min(1, float64(keys.count())/math.Max(1, distinct))
		// Compare seek/heap work with ordered rejection work. These are candidate
		// operations, not an unconditional key-count threshold: LIMIT, selectivity,
		// relation cardinality, and the number of streams all affect the decision.
		prices := []float64{1, 1, 1, 1, 1}
		if len(source) == 5 {
			values := mustScmerSlice(source[4], "ordered keyset prices")
			if len(values) != len(prices) {
				panic("scan_order_keys: expected five cost prices")
			}
			for i, value := range values {
				prices[i] = math.Max(0, scm.ToFloat(value))
			}
		}
		seekWork := (float64(keys.count())*math.Log2(math.Max(2, rows)) + window*math.Log2(float64(keys.count())+1)) * prices[0]
		orderedWork := math.Min(rows, window/math.Max(fraction, 1/math.Max(1, rows))) * prices[1]
		projectedRows := rows * fraction
		projectionWork := rows*prices[2] + projectedRows*(prices[3]+math.Log2(math.Max(2, window))*prices[4])
		if projectionWork < math.Min(seekWork, orderedWork) {
			return fallback()
		}
		if orderedWork <= seekWork {
			prefix = nil
		}
		if Settings.ScanDebugging {
			fmt.Printf("[SCAN_ORDER_KEYS] keys=%d rows=%.0f distinct=%.0f merge=%v seek_ns=%.0f ordered_ns=%.0f projection_ns=%.0f\n", keys.count(), rows, distinct, prefix != nil, seekWork, orderedWork, projectionWork)
		}
		neutral := scm.NewNil()
		if len(a) > 13 {
			neutral = a[13]
		}
		outer := len(a) > 14 && scm.ToBool(a[14])
		notFound := neutral
		if len(a) > 15 {
			notFound = a[15]
		}
		postCols := []string(nil)
		postFilter := scm.NewNil()
		if len(a) > 17 {
			postCols = scmerSliceToStrings(mustScmerSlice(a[16], "post-order columns"))
			postFilter = a[17]
		}
		spec := scanOrderTableSpec{table: target, prefixMerge: prefix, conditionCols: append([]string{targetKeys[0]}, filterCols...), condition: membership, sortcols: sortValues, callbackCols: scmerSliceToStrings(mustScmerSlice(a[11], "map columns")), callback: a[12], postOrderCols: postCols, postOrderFilter: postFilter, accessSchema: a[2], accessValues: mustScmerSlice(a[3], "access values"), perTableLimit: -1}
		scanStart := time.Now()
		result := scanOrderMulti(tx, []scanOrderTableSpec{spec}, scanSortDirections(mustScmerSlice(a[7], "sort directions")), 0, scm.ToInt(a[9]), scm.ToInt(a[10]), neutral, outer, notFound)
		if Settings.ScanDebugging {
			fmt.Printf("[SCAN_ORDER_KEYS] scan_ns=%d\n", time.Since(scanStart).Nanoseconds())
		}
		return result
	}
	scm.Declare(en, &declaration)
}

func orderedPrefixBounds(original scanAccess, prefix *orderedPrefixAccess) scanAccess {
	relationValue := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString("bin"), scm.NewBool(false))
	less, meta := scm.OrderRelationLess(relationValue.Func()), orderRelationMeta(relationValue.Func())
	relation := func(v ...scm.Scmer) scm.Scmer { return scm.NewBool(less(v[0], v[1])) }
	boundaries := analyzedBoundaries{{col: prefix.column, matcher: RangeMatcher, order: relation, orderMeta: meta, lowerInclusive: true, upperInclusive: true}}
	for i, col := range prefix.order {
		if col != prefix.column {
			boundaries = append(boundaries, analyzedBoundary{col: col, matcher: RangeMatcher, order: prefix.dirs[i], orderMeta: orderRelationMeta(prefix.dirs[i]), lowerInclusive: true, upperInclusive: true})
		}
	}
	bounds := scanAccessFromAnalyzed(boundaries)
	bounds.ensureRuntime().prefixMerge = prefix
	bounds.feedback = original.feedback
	return bounds
}

// A cursor borrows an immutable main permutation or a bounded delta page.
// Refilling a delta cursor seeks from its last B-tree item, never rescans the
// prefix or materializes its complete matching run.
type orderedPrefixCursor struct {
	key           scm.Scmer
	position, end int
	delta         bool
	page          []indexPair
	next          int
	resume        indexPair
	exhausted     bool
	id            uint32
}
type orderedPrefixHeap struct {
	cursors []*orderedPrefixCursor
	less    func(*orderedPrefixCursor, *orderedPrefixCursor) bool
}

func (h orderedPrefixHeap) Len() int           { return len(h.cursors) }
func (h orderedPrefixHeap) Less(i, j int) bool { return h.less(h.cursors[i], h.cursors[j]) }
func (h orderedPrefixHeap) Swap(i, j int)      { h.cursors[i], h.cursors[j] = h.cursors[j], h.cursors[i] }
func (h *orderedPrefixHeap) Push(v any)        { h.cursors = append(h.cursors, v.(*orderedPrefixCursor)) }
func (h *orderedPrefixHeap) Pop() any {
	n := len(h.cursors) - 1
	v := h.cursors[n]
	h.cursors = h.cursors[:n]
	return v
}

func (s *StorageIndex) iteratePrefixMerge(tx *TxContext, prefix *orderedPrefixAccess, cols []colGetter, main StorageInt, delta *btree.BTreeG[indexPair], native bool, maxInsert int, buf []uint32, callback func([]uint32) bool) {
	recid := func(position int) uint32 {
		if native {
			return uint32(position)
		}
		return uint32(int64(main.GetValueUInt(uint32(position))) + main.offset)
	}
	advance := func(c *orderedPrefixCursor) bool {
		if !c.delta {
			if c.position >= c.end {
				return false
			}
			c.id = recid(c.position)
			c.position++
			return true
		}
		if c.next >= len(c.page) {
			if c.exhausted {
				return false
			}
			probe := c.resume
			c.page = c.page[:0]
			c.next = 0
			delta.AscendGreaterOrEqual(probe, func(pair indexPair) bool {
				if pair.itemid == probe.itemid {
					return true
				}
				id := uint32(pair.itemid)
				if s.compareAt(0, s.getDeltaColValueTx(tx, id, pair.data, 0), c.key) != 0 {
					c.exhausted = true
					return false
				}
				if id >= s.t.main_count+uint32(maxInsert) {
					return true
				}
				c.page = append(c.page, pair)
				if len(c.page) == 32 {
					return false
				}
				return true
			})
			if len(c.page) == 0 {
				c.exhausted = true
				return false
			}
			c.resume = c.page[len(c.page)-1]
		}
		c.id = uint32(c.page[c.next].itemid)
		c.next++
		return true
	}
	orderSlots := make([]int, len(prefix.order))
	orderLess := make([]func(scm.Scmer, scm.Scmer) bool, len(prefix.order))
	for i, col := range prefix.order {
		orderLess[i] = scm.OrderRelationLess(prefix.dirs[i])
		for j, indexCol := range s.Cols {
			if col == indexCol {
				orderSlots[i] = j
				break
			}
		}
	}
	value := func(c *orderedPrefixCursor, col int) scm.Scmer {
		if c.delta {
			return s.getDeltaColValueTx(tx, c.id, c.page[c.next-1].data, col)
		}
		return cols[col].get(c.id)
	}
	h := orderedPrefixHeap{less: func(a, b *orderedPrefixCursor) bool {
		for i, slot := range orderSlots {
			av, bv := value(a, slot), value(b, slot)
			if orderLess[i](av, bv) {
				return true
			}
			if orderLess[i](bv, av) {
				return false
			}
		}
		return a.id < b.id
	}}
	for _, key := range prefix.keys.values {
		start := sort.Search(int(s.t.main_count), func(i int) bool { return s.compareAt(0, cols[0].get(recid(i)), key) >= 0 })
		end := start + sort.Search(int(s.t.main_count)-start, func(i int) bool { return s.compareAt(0, cols[0].get(recid(start+i)), key) > 0 })
		c := &orderedPrefixCursor{key: key, position: start, end: end}
		if advance(c) {
			h.cursors = append(h.cursors, c)
		}
		if delta != nil {
			d := &orderedPrefixCursor{key: key, delta: true, page: make([]indexPair, 0, 32), resume: indexPair{itemid: -1, data: []scm.Scmer{key}}}
			if advance(d) {
				h.cursors = append(h.cursors, d)
			}
		}
	}
	heap.Init(&h)
	// Let LIMIT brake after its first window, rather than decoding a full batch.
	width := len(buf)
	if width > 32 {
		width = 32
	}
	count := 0
	for h.Len() > 0 {
		c := h.cursors[0]
		buf[count] = c.id
		count++
		if !advance(c) {
			heap.Pop(&h)
		} else {
			heap.Fix(&h, 0)
		}
		if count == width {
			if !callback(buf[:count]) {
				return
			}
			count = 0
		}
	}
	if count > 0 {
		callback(buf[:count])
	}
}

// A cold composite index must not force an allocating Scheme predicate across
// the complete relation. Reuse the native projection alternative while the
// normal autoindex savings policy trains the ordered index. The caller already
// holds shard rights and RLock and has loaded the physical prefix column.
func (s *StorageIndex) projectPrefixCandidates(tx *TxContext, bounds scanAccess, buf []uint32, callback func([]uint32) bool) bool {
	if bounds.runtime == nil || bounds.runtime.prefixMerge == nil {
		return false
	}
	prefix := bounds.runtime.prefixMerge
	part := s.t.projectJoinKeysPart(tx, []string{prefix.column}, prefix.keys, SessionStateFromTx(tx), true, true)
	count := 0
	stopped := false
	part.forEachID(func(id uint32) bool {
		buf[count] = id
		count++
		if count == len(buf) {
			if !callback(buf[:count]) {
				stopped = true
				return false
			}
			count = 0
		}
		return true
	})
	if !stopped && count > 0 {
		callback(buf[:count])
	}
	return true
}
