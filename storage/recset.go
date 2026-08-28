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

import "fmt"
import "math"
import "runtime/debug"
import "sort"
import "sync/atomic"
import "unsafe"
import "github.com/launix-de/memcp/scm"

// TagRecSet is the custom Scmer tag for query-local RecSet handles.
const TagRecSet = 102

// recSetShard is one shard's contribution to a query-local RecSet, in one of
// three kinds (see recSetRepresentation in recset_representation.go).
//
// INVARIANT: for recSetRanges and recSetPositive, data is always sorted
// ascending — recSetRanges as non-overlapping, non-adjacent (base,count)
// pairs ordered by base; recSetPositive as a deduplicated ascending id list.
// This is not incidental: every constructor (recSetShardBuilder.finish(),
// newRecSetShardFromSortedIDs, and every union/intersect result builder in
// recset_representation.go) either scans/merges its inputs in ascending
// order by construction or is fed a precondition-documented pre-sorted
// input, and recSetShardBuilder.finish()'s positive path additionally
// enforces it with an explicit sort as a guard. The entire point of
// maintaining this is that every merge/intersect algorithm operating on
// recSetShards (unionSortedRanges, intersectSortedRanges, unionSortedLists,
// intersectSortedLists, intersectRangesWithList, intersectRangesWithBitmap,
// intersectListWithBitmap, ...) is a linear two-pointer/N-way sweep, not a
// per-element binary search or O(n*m) scan — that only works because both
// operands are already ordered. A shard reaching those algorithms out of
// order would silently corrupt them rather than fail loudly, so this
// invariant must never be broken by a future constructor.
type recSetShard struct {
	shard    *storageShard
	kind     recSetRepresentation
	universe uint32
	data     []uint32
	used     uint32
	count    int64
}

func (s *recSetShard) contains(recid uint32) bool {
	if s == nil || recid >= s.universe {
		return false
	}
	switch s.kind {
	case recSetRanges:
		return s.rangesContains(recid)
	case recSetPositive:
		return sortedUint32Contains(s.listedValues(), recid)
	case recSetBitmap:
		return s.data[recid>>5]&(uint32(1)<<(recid&31)) != 0
	default:
		return false
	}
}

// forEachVisibleRun walks part via forEachRange, splitting each range into
// maximal runs of currently-visible rows (below visibleUpper, and neither
// deleted nor tx-invisible), and calls onRun(base, count) once per such
// run — the natural granularity for a bulk column read (GetValueRange)
// instead of one GetValue call per row. Visibility is still checked one row
// at a time (that was never avoidable — it's a property check, not a column
// read, and doesn't get any cheaper by batching); what this saves is the
// column reads inside onRun, which can cover however many consecutive rows
// happen to be visible in a row, often the whole run for a recSetRanges
// shard with no deletions in flight.
func (t *storageShard) forEachVisibleRun(part *recSetShard, visibleUpper uint32, acidMode bool, currentTx *TxContext, onRun func(base, count uint32) bool) {
	cont := true
	part.forEachRange(func(base, count uint32) bool {
		end := base + count
		if base >= visibleUpper {
			return true
		}
		if end > visibleUpper {
			end = visibleUpper
		}
		var runStart uint32
		haveRun := false
		for idx := base; idx < end; idx++ {
			visible := true
			if acidMode {
				visible = currentTx.IsVisible(t, idx)
			} else if t.deletions.Get(uint(idx)) {
				visible = false
			}
			if visible {
				if !haveRun {
					runStart = idx
					haveRun = true
				}
				continue
			}
			if haveRun {
				if !onRun(runStart, idx-runStart) {
					cont = false
					return false
				}
				haveRun = false
			}
		}
		if haveRun {
			if !onRun(runStart, end-runStart) {
				cont = false
				return false
			}
		}
		return cont
	})
}

// recSet is a query-local, non-persistent subset of one table represented by
// physical record IDs. It is intentionally table-shaped so the planner can swap
// repeated scans over the same filtered base relation for a scan over this value.
type recSet struct {
	tx     *TxContext
	table  *table
	shards []recSetShard
	count  int64
}

func NewRecSetScmer(rs *recSet) scm.Scmer {
	return scm.NewCustom(TagRecSet, unsafe.Pointer(rs))
}

func RecSetFromScmer(s scm.Scmer) *recSet {
	return (*recSet)(s.Custom(TagRecSet))
}

func (r *recSet) String() string {
	if r == nil || r.table == nil {
		return "(recset nil)"
	}
	return fmt.Sprintf("(recset %q %q %d)", r.table.schema.Name, r.table.Name, r.count)
}

func (r *recSet) shardEntry(shard *storageShard) *recSetShard {
	if r == nil || shard == nil || r.table != shard.t {
		return nil
	}
	for i := range r.shards {
		rs := &r.shards[i]
		if rs.shard == shard {
			return rs
		}
	}
	return nil
}

func (r *recSet) contains(shard *storageShard, recid uint32) bool {
	return r.shardEntry(shard).contains(recid)
}

func recSetUnion(items []*recSet) *recSet {
	var base *table
	var tx *TxContext
	for _, rs := range items {
		if rs == nil || rs.table == nil {
			continue
		}
		if base == nil {
			base = rs.table
			tx = rs.tx
		} else if base != rs.table {
			panic("recset_union: all recsets must belong to the same table")
		}
	}
	result := &recSet{tx: tx, table: base}
	if base == nil {
		return result
	}

	byShard := make(map[*storageShard][]*recSetShard)
	for _, rs := range items {
		if rs == nil || rs.table == nil {
			continue
		}
		for i := range rs.shards {
			part := rs.shards[i]
			if part.count == 0 {
				continue
			}
			byShard[part.shard] = append(byShard[part.shard], &rs.shards[i])
		}
	}
	type shardUnionTask struct {
		shard *storageShard
		parts []*recSetShard
	}
	tasks := make([]shardUnionTask, 0, len(byShard))
	for shard, parts := range byShard {
		tasks = append(tasks, shardUnionTask{shard, parts})
	}

	// Each shard's combine is an independent, allocation-only computation
	// (no shard lock needed — the recSetShards involved are already-built,
	// query-local snapshots) costing on the order of 100µs-1ms; fanning out
	// via runFanoutTasks pays off even for a single mixed-kind shard, since
	// its own channel-dispatch overhead is roughly 1µs — well under 1% of a
	// single combine's cost, let alone many shards' worth done serially.
	values := make(chan recSetBuildResult, len(tasks))
	done := runFanoutTasks(tx, len(tasks), func(taskIndex int, _ bool) {
		withTxSession(tx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			task := tasks[taskIndex]
			values <- recSetBuildResult{part: unionRecSetShards(task.shard, task.parts)}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if buildErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if buildErr.r != nil {
		panic(buildErr)
	}

	sort.Slice(result.shards, func(i, j int) bool {
		return result.shards[i].shard.uuid.String() < result.shards[j].shard.uuid.String()
	})
	return result
}

func recSetIntersect(items []*recSet) *recSet {
	var base *table
	var tx *TxContext
	for _, rs := range items {
		if rs == nil || rs.table == nil {
			continue
		}
		if base == nil {
			base = rs.table
			tx = rs.tx
		} else if base != rs.table {
			panic("recset_intersect: all recsets must belong to the same table")
		}
	}
	result := &recSet{tx: tx, table: base}
	if base == nil || len(items) == 0 {
		return result
	}
	shards := base.ActiveShards()

	// See recSetUnion for why this is worth fanning out even for a single
	// shard: channel-dispatch overhead (~1µs) is under 1% of a single
	// combine's cost (~100µs-1ms).
	values := make(chan recSetBuildResult, len(shards))
	done := runFanoutTasks(tx, len(shards), func(taskIndex int, _ bool) {
		withTxSession(tx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			shard := shards[taskIndex]
			parts := make([]*recSetShard, len(items))
			for i, rs := range items {
				if rs != nil {
					parts[i] = rs.shardEntry(shard)
				}
			}
			values <- recSetBuildResult{part: intersectRecSetShards(shard, parts)}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if buildErr.r != nil {
			continue
		}
		if msg.part.count > 0 {
			result.count += msg.part.count
			result.shards = append(result.shards, msg.part)
		}
	}
	if buildErr.r != nil {
		panic(buildErr)
	}
	// The serial loop this replaced preserved ActiveShards()'s deterministic
	// order implicitly; fanout results arrive in completion order instead,
	// so restore a stable order the same way recSetUnion does.
	sort.Slice(result.shards, func(i, j int) bool {
		return result.shards[i].shard.uuid.String() < result.shards[j].shard.uuid.String()
	})
	return result
}

// recSetDifference returns the records in the first RecSet which occur in none
// of the following RecSets. All operands must belong to the same base table.
func recSetDifference(items []*recSet) *recSet {
	if len(items) == 0 || items[0] == nil || items[0].table == nil {
		panic("recset_difference requires a non-empty list of recsets")
	}
	left := items[0]
	for _, rs := range items[1:] {
		if rs == nil || rs.table != left.table {
			panic("recset_difference: all recsets must belong to the same table")
		}
	}
	result := &recSet{tx: left.tx, table: left.table}
	values := make(chan recSetBuildResult, len(left.shards))
	done := runFanoutTasks(left.tx, len(left.shards), func(taskIndex int, _ bool) {
		withTxSession(left.tx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			leftPart := &left.shards[taskIndex]
			exclusions := make([]*recSetShard, 0, len(items)-1)
			for _, rs := range items[1:] {
				if part := rs.shardEntry(leftPart.shard); part != nil && part.count > 0 {
					exclusions = append(exclusions, part)
				}
			}
			var excluded recSetShard
			if len(exclusions) > 0 {
				excluded = unionRecSetShards(leftPart.shard, exclusions)
			}
			values <- recSetBuildResult{part: differenceRecSetShards(leftPart.shard, leftPart, &excluded)}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if msg.part.count > 0 {
			result.count += msg.part.count
			result.shards = append(result.shards, msg.part)
		}
	}
	if buildErr.r != nil {
		panic(buildErr)
	}
	sort.Slice(result.shards, func(i, j int) bool {
		return result.shards[i].shard.uuid.String() < result.shards[j].shard.uuid.String()
	})
	return result
}

// recSetNot complements a RecSet against the currently visible rows of its
// base table. Building that visible universe through the ordinary scan path is
// essential: raw recid inversion would resurrect deleted or tx-invisible rows.
func recSetNot(item *recSet) *recSet {
	if item == nil || item.table == nil {
		panic("recset_not requires a recset with a base table")
	}
	visible := item.table.scanRecSet(item.tx, nil, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.NewBool(true)
	}))
	return recSetDifference([]*recSet{visible, item})
}

func recSetContainsClosure(shard *storageShard) *func(uint32, ...scm.Scmer) scm.Scmer {
	var cachedRecSet *recSet
	var cachedShard *recSetShard
	fn := func(recid uint32, args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsCustom(TagRecSet) {
			return scm.NewBool(false)
		}
		rs := RecSetFromScmer(args[0])
		if rs != cachedRecSet {
			cachedRecSet = rs
			cachedShard = rs.shardEntry(shard)
		}
		return scm.NewBool(cachedShard.contains(recid))
	}
	return &fn
}

func recSetAlreadyMatchedClosure() *func(uint32, ...scm.Scmer) scm.Scmer {
	fn := func(_ uint32, _ ...scm.Scmer) scm.Scmer {
		return scm.NewBool(true)
	}
	return &fn
}

type recSetBuildResult struct {
	part recSetShard
	err  scanError
}

type recSetKeyResult struct {
	part int
	used int
	err  scanError
}

func (t *table) scanRecSet(currentTx *TxContext, conditionCols []string, condition scm.Scmer) *recSet {
	ss := SessionStateFromTx(currentTx)
	querySeq := scm.CurrentQuerySeq()
	boundaries := extractBoundaries(conditionCols, condition)
	reorderByFrequency(boundaries, t)
	lower, upperLast := indexFromBoundaries(boundaries)
	result := &recSet{tx: currentTx, table: t}

	values := make(chan recSetBuildResult, t.shardResultBufferSize())
	done := t.iterateShardsParallel(currentTx, boundaries, func(shard *storageShard, solo bool) {
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			// Cancellation contract: check only at the scheduling boundary, before entering
			// the shard. Once entered, a shard runs atomically without cancellation checks.
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			values <- recSetBuildResult{
				part: shard.collectRecSet(boundaries, lower, upperLast, conditionCols, condition, currentTx, ss),
			}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if buildErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if buildErr.r != nil {
		panic(buildErr)
	}
	return result
}

func (t *storageShard) collectRecSet(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState) recSetShard {
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(boundaries, lower, t.t, conditionCols, condition)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsCachedReader := make([]bool, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			if recsetBoundaryCoversCondition {
				fnptr = recSetAlreadyMatchedClosure()
			}
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			conditionGetters[i] = getter
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		if _, ok := ccols[i].(*StorageComputeProxy); ok {
			cNeedsCachedReader[i] = true
		}
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	completeTraversal := len(boundaries) == 0
	singleExactLike := len(boundaries) == 1 && singleLikeBoundaryCoversCondition(conditionCols, condition, boundaries[0])
	exactLikeMain := false
	builder := newRecSetShardBuilder(t, visibleUpper, completeTraversal, t.main_count)
	evaluate := func(idx uint32) bool {
		if idx >= visibleUpper {
			return false
		}
		if acidMode {
			if !currentTx.IsVisible(t, idx) {
				return false
			}
		} else if t.deletions.Get(uint(idx)) {
			return false
		}
		if idx < t.main_count && exactLikeMain && singleExactLike {
			return true
		}
		if idx < t.main_count {
			for i, c := range cReaders {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if cNeedsCachedReader[i] {
					cdataset[i] = c.GetValue(idx)
				} else {
					cdataset[i] = ccols[i].GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if cNeedsCachedReader[i] {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = ccols[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-t.main_count), col)
				}
			}
		}
		return scm.ToBool(conditionFn(cdataset...))
	}
	var buf [1024]uint32
	t.iterateIndexMatchAware(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf[:], true, &exactLikeMain, func(batch []uint32) bool {
		for _, idx := range batch {
			if idx >= visibleUpper {
				continue
			}
			builder.add(idx, evaluate(idx))
		}
		return true
	})
	return builder.finish()
}

func (r *recSet) projectJoin(currentTx *TxContext, sourceKeyCols []string, target *table, targetKeyCols []string) *recSet {
	if r == nil || r.table == nil {
		return &recSet{tx: currentTx, table: target}
	}
	if len(sourceKeyCols) == 0 || len(sourceKeyCols) != len(targetKeyCols) {
		panic("recset_project_join: source and target key columns must have the same non-zero length")
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	ss := SessionStateFromTx(currentTx)
	keys := r.collectProjectJoinKeys(currentTx, sourceKeyCols, ss)
	return target.projectJoinKeysToRecSet(currentTx, targetKeyCols, keys, ss)
}

type recSetProjectKeys struct {
	width         int
	values        []scm.Scmer
	numericValues map[int64]struct{}
}

func (k *recSetProjectKeys) count() int {
	if k.width == 0 {
		return 0
	}
	return len(k.values) / k.width
}

func (k *recSetProjectKeys) tuple(index int) []scm.Scmer {
	start := index * k.width
	return k.values[start : start+k.width]
}

func (k *recSetProjectKeys) contains(key []scm.Scmer) bool {
	if k.numericValues != nil && len(key) == 1 {
		if integer, ok := projectNumericInteger(key[0]); ok {
			_, found := k.numericValues[integer]
			return found
		}
	}
	position := sort.Search(k.count(), func(i int) bool {
		return compareProjectKey(k.tuple(i), key) >= 0
	})
	return position < k.count() && compareProjectKey(k.tuple(position), key) == 0
}

func projectNumericInteger(value scm.Scmer) (int64, bool) {
	switch value.GetTag() {
	case scm.TagInt:
		return value.Int(), true
	case scm.TagFloat:
		number := value.Float()
		integer := int64(number)
		return integer, number == float64(integer)
	default:
		return 0, false
	}
}

func (k *recSetProjectKeys) buildNumericLookup() bool {
	if k.width != 1 {
		return false
	}
	values := make(map[int64]struct{}, k.count())
	for position := 0; position < k.count(); position++ {
		value := k.tuple(position)[0]
		integer, ok := projectNumericInteger(value)
		if !ok {
			return false
		}
		values[integer] = struct{}{}
	}
	k.numericValues = values
	return true
}

// Rounded conservative constants from the production-shaped A/B fixture:
// 105 keys projected through 22 indexed target shards took 24ms, while the
// numeric dense path projected about 80k keys through 1.84m rows in 8ms. The
// old generic binary-search scan took roughly 250ns per comparison. Keep the
// units explicit so costgen can replace these constants without changing the
// decision function when its projection calibration lands.
const projectJoinIndexProbeNs = 10000.0
const projectJoinNumericScanRowNs = 5.0
const projectJoinGenericCompareNs = 250.0

// projectJoinDensePreferred compares the two physical implementations using
// the cardinalities known at execution time. An indexed projection performs
// one point lookup per source key and shard. A dense projection visits each
// target row once; homogeneous integral keys use the O(1) numeric lookup,
// while generic tuples retain the sorted-key binary search.
func projectJoinDensePreferred(keyCount, targetRows int, numeric bool, indexed bool) bool {
	if !indexed {
		return true
	}
	if keyCount == 0 || targetRows == 0 {
		return false
	}
	indexCost := float64(keyCount) * projectJoinIndexProbeNs
	if numeric {
		return float64(targetRows)*projectJoinNumericScanRowNs <= indexCost
	}
	comparisons := math.Log2(float64(keyCount) + 1)
	denseCost := float64(targetRows) * comparisons * projectJoinGenericCompareNs
	return denseCost <= indexCost
}

func projectJoinNumericColumn(column ColumnStorage) bool {
	switch column.(type) {
	case *StorageInt, *StorageSeq:
		return true
	default:
		return false
	}
}

func compareProjectKey(left, right []scm.Scmer) int {
	for i := range left {
		if scm.Equal(left[i], right[i]) {
			continue
		}
		if scm.Less(left[i], right[i]) {
			return -1
		}
		return 1
	}
	return 0
}

func (k *recSetProjectKeys) sortAndDeduplicate() {
	count := k.count()
	if count < 2 {
		return
	}
	sort.Sort(makeProjectKeyIndices(k))
	write := 1
	for read := 1; read < count; read++ {
		if compareProjectKey(k.tuple(write-1), k.tuple(read)) == 0 {
			continue
		}
		if write != read {
			copy(k.tuple(write), k.tuple(read))
		}
		write++
	}
	k.values = k.values[:write*k.width]
}

type projectKeyIndices struct {
	keys *recSetProjectKeys
}

func makeProjectKeyIndices(keys *recSetProjectKeys) *projectKeyIndices {
	return &projectKeyIndices{keys: keys}
}

func (p *projectKeyIndices) Len() int { return p.keys.count() }
func (p *projectKeyIndices) Less(i, j int) bool {
	return compareProjectKey(p.keys.tuple(i), p.keys.tuple(j)) < 0
}
func (p *projectKeyIndices) Swap(i, j int) {
	left := p.keys.tuple(i)
	right := p.keys.tuple(j)
	for col := range left {
		left[col], right[col] = right[col], left[col]
	}
}

func (r *recSet) collectProjectJoinKeys(currentTx *TxContext, sourceKeyCols []string, ss *scm.SessionState) recSetProjectKeys {
	querySeq := scm.CurrentQuerySeq()
	values := make(chan recSetKeyResult, len(r.shards))
	width := len(sourceKeyCols)
	keys := recSetProjectKeys{width: width, values: make([]scm.Scmer, int(r.count)*width)}
	offsets := make([]int, len(r.shards)+1)
	for i := range r.shards {
		offsets[i+1] = offsets[i] + int(r.shards[i].count)*width
	}
	activeParts := make([]int, 0, len(r.shards))
	for i := range r.shards {
		if r.shards[i].count > 0 {
			activeParts = append(activeParts, i)
		}
	}
	done := runFanoutTasks(currentTx, len(activeParts), func(taskIndex int, _ bool) {
		partIndex := activeParts[taskIndex]
		part := r.shards[partIndex]
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetKeyResult{part: partIndex, err: scanError{rec, string(debug.Stack())}}
				}
			}()
			// Cancellation contract: check only at the scheduling boundary, before entering
			// the shard. Once entered, a shard runs atomically without cancellation checks.
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			dst := keys.values[offsets[partIndex]:offsets[partIndex+1]]
			used := part.shard.collectProjectJoinKeys(&part, sourceKeyCols, currentTx, ss, dst)
			values <- recSetKeyResult{part: partIndex, used: used}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	used := make([]int, len(r.shards))
	var keyErr scanError
	for i := 0; i < len(activeParts); i++ {
		msg := <-values
		if msg.err.r != nil {
			if keyErr.r == nil {
				keyErr = msg.err
			}
			continue
		}
		used[msg.part] = msg.used
	}
	if keyErr.r != nil {
		panic(keyErr)
	}
	write := 0
	for i := range r.shards {
		if used[i] == 0 {
			continue
		}
		start := offsets[i]
		copy(keys.values[write:], keys.values[start:start+used[i]])
		write += used[i]
	}
	keys.values = keys.values[:write]
	keys.sortAndDeduplicate()
	return keys
}

func (t *storageShard) collectProjectJoinKeys(part *recSetShard, sourceKeyCols []string, currentTx *TxContext, ss *scm.SessionState, dst []scm.Scmer) int {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	cols := make([]ColumnStorage, len(sourceKeyCols))
	readers := make([]ColumnReader, len(sourceKeyCols))
	needsTxReader := make([]bool, len(sourceKeyCols))
	for i, col := range sourceKeyCols {
		cols[i] = t.getColumnStorageOrPanicEx(col, skipShardReadLock, currentTx)
		readers[i] = newCachedColumnReaderTx(cols[i], currentTx)
		if proxy, ok := cols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			needsTxReader[i] = true
		}
	}

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	visibleUpper := mainCount + uint32(len(t.inserts))
	write := 0
	colBufs := make([][]scm.Scmer, len(sourceKeyCols))
	t.forEachVisibleRun(part, visibleUpper, acidMode, currentTx, func(base, count uint32) bool {
		end := base + count
		mainEnd := end
		if mainEnd > mainCount {
			mainEnd = mainCount
		}
		if mainEnd < base {
			// Run lies entirely in the delta region (base >= mainCount):
			// without this, the delta loop below would incorrectly start
			// at mainCount instead of base, reprocessing indices
			// [mainCount,base) that aren't part of this run at all.
			mainEnd = base
		}
		if base < mainEnd {
			// Main-storage sub-run: bulk-fetch every column for the whole
			// run in one GetValueRange call each, instead of one GetValue
			// call per row per column.
			runLen := mainEnd - base
			for i := range sourceKeyCols {
				if cap(colBufs[i]) < int(runLen) {
					colBufs[i] = make([]scm.Scmer, runLen)
				}
				colBufs[i] = colBufs[i][:runLen]
				if needsTxReader[i] {
					readers[i].GetValueRange(base, runLen, colBufs[i], 1)
				} else {
					cols[i].GetValueRange(base, runLen, colBufs[i], 1)
				}
			}
			for row := uint32(0); row < runLen; row++ {
				skip := false
				for i := range sourceKeyCols {
					v := colBufs[i][row]
					dst[write+i] = v
					if v.IsNil() {
						skip = true
					}
				}
				if !skip {
					write += len(sourceKeyCols)
				}
			}
		}
		// Delta sub-run (idx >= mainCount): getDelta is a plain map lookup,
		// not a ColumnStorage read, so there's nothing to batch here.
		for idx := mainEnd; idx < end; idx++ {
			skip := false
			for i, col := range sourceKeyCols {
				if needsTxReader[i] {
					dst[write+i] = readers[i].GetValue(idx)
				} else if _, isProxy := cols[i].(*StorageComputeProxy); isProxy {
					dst[write+i] = cols[i].GetValue(idx)
				} else {
					dst[write+i] = t.getDelta(int(idx-mainCount), col)
				}
				if dst[write+i].IsNil() {
					skip = true
				}
			}
			if !skip {
				write += len(sourceKeyCols)
			}
		}
		return true
	})
	return write
}

func (t *table) projectJoinKeysToRecSet(currentTx *TxContext, targetKeyCols []string, keys recSetProjectKeys, ss *scm.SessionState) *recSet {
	result := &recSet{tx: currentTx, table: t}
	if keys.count() == 0 {
		return result
	}
	querySeq := scm.CurrentQuerySeq()
	numeric := keys.buildNumericLookup()
	type targetPartResult struct {
		part recSetShard
		err  scanError
	}
	values := make(chan targetPartResult, t.shardResultBufferSize())
	done := t.iterateShardsParallel(currentTx, nil, func(shard *storageShard, solo bool) {
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- targetPartResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			// Cancellation contract: check only at the scheduling boundary, before entering
			// the shard. Once entered, a shard runs atomically without cancellation checks.
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			values <- targetPartResult{part: shard.projectJoinKeysPart(currentTx, targetKeyCols, keys, ss, numeric)}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)
	var joinErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if joinErr.r == nil {
				joinErr = msg.err
			}
			continue
		}
		if joinErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if joinErr.r != nil {
		panic(joinErr)
	}
	return result
}

// hasEqualityIndexPrefix reports whether an active index already covers cols
// as an equality prefix, without triggering a build. The sparse
// per-key lookup path below is only cheap when such an index already
// exists; forcing one into existence for a single ad-hoc query can cost far
// more than the linear scan it was meant to avoid.
func (t *storageShard) hasEqualityIndexPrefix(currentTx *TxContext, cols []string) bool {
	for _, index := range t.Indexes {
		if len(index.Cols) < len(cols) {
			continue
		}
		state := index.stateForTx(currentTx, false)
		index.mu.Lock()
		active := state != nil && state.active
		index.mu.Unlock()
		if !active {
			continue
		}
		match := true
		for i := range cols {
			if index.Cols[i] != cols[i] {
				match = false
				break
			}
			var indexedMatcher IndexAnalyzer
			if len(index.ColMatchers) > i {
				indexedMatcher = index.ColMatchers[i]
			}
			if !indexMatcherCompatible(EqualMatcher, indexedMatcher) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (t *storageShard) projectJoinKeysPart(currentTx *TxContext, targetKeyCols []string, keys recSetProjectKeys, ss *scm.SessionState, numeric bool) recSetShard {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	targetCols := make([]ColumnStorage, len(targetKeyCols))
	targetReaders := make([]ColumnReader, len(targetKeyCols))
	targetNeedsTxReader := make([]bool, len(targetKeyCols))
	for i, col := range targetKeyCols {
		targetCols[i] = t.getColumnStorageOrPanicEx(col, skipShardReadLock, currentTx)
		targetReaders[i] = newCachedColumnReaderTx(targetCols[i], currentTx)
		if proxy, ok := targetCols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			targetNeedsTxReader[i] = true
		}
	}

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	numericDense := numeric && len(targetCols) == 1 && projectJoinNumericColumn(targetCols[0])
	dense := projectJoinDensePreferred(keys.count(), int(visibleUpper), numericDense,
		t.hasEqualityIndexPrefix(currentTx, targetKeyCols))
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	if dense {
		builder := newRecSetShardBuilder(t, visibleUpper, true, t.main_count)
		actual := make([]scm.Scmer, len(targetKeyCols))
		evaluate := func(idx uint32) bool {
			visible := true
			if acidMode {
				visible = currentTx.IsVisible(t, idx)
			} else if t.deletions.Get(uint(idx)) {
				visible = false
			}
			if !visible {
				return false
			}
			for col := range targetKeyCols {
				if idx < t.main_count {
					if targetNeedsTxReader[col] {
						actual[col] = targetReaders[col].GetValue(idx)
					} else {
						actual[col] = targetCols[col].GetValue(idx)
					}
				} else if targetNeedsTxReader[col] {
					actual[col] = targetReaders[col].GetValue(idx)
				} else if _, proxy := targetCols[col].(*StorageComputeProxy); proxy {
					actual[col] = targetCols[col].GetValue(idx)
				} else {
					actual[col] = t.getDelta(int(idx-t.main_count), targetKeyCols[col])
				}
			}
			return keys.contains(actual)
		}
		for idx := uint32(0); idx < visibleUpper; idx++ {
			builder.add(idx, evaluate(idx))
		}
		return builder.finish()
	}
	seen := make(map[uint32]struct{})
	recids := make([]uint32, 0, 64)
	var buf [1024]uint32
	for keyIndex := 0; keyIndex < keys.count(); keyIndex++ {
		key := keys.tuple(keyIndex)
		bounds := make(boundaries, len(targetKeyCols))
		for i, col := range targetKeyCols {
			bounds[i] = columnboundaries{
				col:            col,
				matcher:        EqualMatcher,
				lower:          key[i],
				lowerInclusive: true,
				upper:          key[i],
				upperInclusive: true,
			}
		}
		reorderByFrequency(bounds, t.t)
		lower, upperLast := indexFromBoundaries(bounds)
		t.iterateIndexForce(currentTx, bounds, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
			for _, idx := range batch {
				if idx >= visibleUpper {
					continue
				}
				if acidMode {
					if !currentTx.IsVisible(t, idx) {
						continue
					}
				} else if t.deletions.Get(uint(idx)) {
					continue
				}
				if _, ok := seen[idx]; ok {
					continue
				}
				if !t.projectJoinTargetMatches(idx, key, targetKeyCols, targetCols, targetReaders, targetNeedsTxReader, currentTx) {
					continue
				}
				seen[idx] = struct{}{}
				recids = append(recids, idx)
			}
			return true
		})
	}
	sort.Slice(recids, func(i, j int) bool { return recids[i] < recids[j] })
	return newRecSetShardFromSortedIDs(t, visibleUpper, recids)
}

func (t *storageShard) projectJoinTargetMatches(idx uint32, key []scm.Scmer, targetKeyCols []string, targetCols []ColumnStorage, targetReaders []ColumnReader, targetNeedsTxReader []bool, currentTx *TxContext) bool {
	for i, expected := range key {
		var actual scm.Scmer
		if idx < t.main_count {
			if targetNeedsTxReader[i] {
				actual = targetReaders[i].GetValue(idx)
			} else {
				actual = targetCols[i].GetValue(idx)
			}
		} else if targetNeedsTxReader[i] {
			actual = targetReaders[i].GetValue(idx)
		} else if _, isProxy := targetCols[i].(*StorageComputeProxy); isProxy {
			actual = targetCols[i].GetValue(idx)
		} else {
			actual = t.getDelta(int(idx-t.main_count), targetKeyCols[i])
		}
		if !scm.Equal(actual, expected) {
			return false
		}
	}
	return true
}

func (r *recSet) scan(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	if r == nil || r.table == nil {
		return neutral
	}
	for _, col := range callbackCols {
		if col == "$update" || (len(col) > 11 && col[:11] == "$increment:") ||
			(len(col) > 5 && col[:5] == "$set:") ||
			(len(col) > 12 && col[:12] == "$invalidate:") {
			panic("recset scan mutation callbacks are not implemented")
		}
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	return r.table.scanWithBatchFrom(currentTx, r, conditionCols, condition,
		callbackCols, callback, aggregate, neutral, aggregate2, isOuter, 0, nil)
}

func (r *recSet) scanExists(currentTx *TxContext, conditionCols []string, condition scm.Scmer) bool {
	if r == nil || r.table == nil {
		return false
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	return r.table.scanExistsFrom(currentTx, r, conditionCols, condition)
}

func (t *storageShard) recSetPartExists(part *recSetShard, conditionCols []string, conditionFn func(...scm.Scmer) scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) bool {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			conditionGetters[i] = getter
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	visibleUpper := mainCount + uint32(len(t.inserts))
	found := false
	part.forEachID(func(idx uint32) bool {
		if stop != nil && stop.Load() {
			return false
		}
		if idx >= visibleUpper {
			return true
		}
		if acidMode {
			if !currentTx.IsVisible(t, idx) {
				return true
			}
		} else if t.deletions.Get(uint(idx)) {
			return true
		}
		if idx < mainCount {
			for i, c := range cReaders {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else {
					cdataset[i] = c.GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
		}
		if scm.ToBool(conditionFn(cdataset...)) {
			found = true
			return false
		}
		return true
	})
	return found
}

func (t *storageShard) scanRecSetPart(part *recSetShard, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64) {
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			conditionGetters[i] = getter
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))
	var mapperStorage ShardMapReducer
	var mapperWorkspace shardMapReducerWorkspace
	mapper := &mapperStorage
	if mapReducerCanUseReadWorkspace(callbackCols) {
		prepareReadMapReducerStorage(&mapperStorage, &mapperWorkspace, len(callbackCols))
		t.initReadMapReducer(&mapperStorage, callbackCols, callback, aggregate, skipShardReadLock, currentTx)
	} else {
		mapper = t.OpenMapReducer(callbackCols, callback, aggregate, skipShardReadLock, 0, nil, currentTx)
	}
	defer mapper.Close()

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	visibleUpper := mainCount + uint32(len(t.inserts))
	akkumulator := neutral
	var outCount int64
	var buf [1024]uint32
	pending := buf[:0]
	flush := func() {
		if len(pending) == 0 {
			return
		}
		if locked {
			t.mu.RUnlock()
			locked = false
		}
		akkumulator = mapper.Stream(akkumulator, pending, nil)
		pending = buf[:0]
		if !skipShardReadLock {
			t.mu.RLock()
			locked = true
		}
	}
	colBufs := make([][]scm.Scmer, len(conditionCols))
	t.forEachVisibleRun(part, visibleUpper, acidMode, currentTx, func(base, count uint32) bool {
		end := base + count
		mainEnd := end
		if mainEnd > mainCount {
			mainEnd = mainCount
		}
		if mainEnd < base {
			// Run lies entirely in the delta region (base >= mainCount):
			// without this, the delta loop below would incorrectly start
			// at mainCount instead of base, reprocessing indices
			// [mainCount,base) that aren't part of this run at all.
			mainEnd = base
		}
		if base < mainEnd {
			// Main-storage sub-run: bulk-fetch every non-getter condition
			// column for the whole run in one GetValueRange call each.
			runLen := mainEnd - base
			for i := range conditionCols {
				if conditionGetters[i] != nil {
					continue
				}
				if cap(colBufs[i]) < int(runLen) {
					colBufs[i] = make([]scm.Scmer, runLen)
				}
				colBufs[i] = colBufs[i][:runLen]
				cReaders[i].GetValueRange(base, runLen, colBufs[i], 1)
			}
			for row := uint32(0); row < runLen; row++ {
				idx := base + row
				for i := range conditionCols {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(idx, 0)
					} else {
						cdataset[i] = colBufs[i][row]
					}
				}
				if !scm.ToBool(conditionFn(cdataset...)) {
					continue
				}
				pending = append(pending, idx)
				outCount++
				if len(pending) == cap(buf) {
					flush()
				}
			}
		}
		// Delta sub-run: unchanged, per-row (getDelta is a map lookup, not a
		// ColumnStorage read).
		for idx := mainEnd; idx < end; idx++ {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
			if !scm.ToBool(conditionFn(cdataset...)) {
				continue
			}
			pending = append(pending, idx)
			outCount++
			if len(pending) == cap(buf) {
				flush()
			}
		}
		return true
	})
	flush()
	if locked {
		t.mu.RUnlock()
		locked = false
	}
	mapper.FlushSideEffects()
	if outCount == 0 {
		return scm.NewNil(), 0
	}
	return akkumulator, outCount
}

// filterRecSetPart re-evaluates condition only over the recids already present
// in part, instead of the shard's full boundary-indexed row space. Used to
// narrow an existing RecSet by a further condition (including one with its own
// subscans) without ever touching rows outside the incoming membership --
// e.g. evaluating an expensive correlated ACL check only over the ~30k rows a
// mandant filter already narrowed a table down to, not the full table.
func (t *storageShard) filterRecSetPart(owner *recSet, part *recSetShard, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState) recSetShard {
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	// Keep RecSet narrowing on the ordinary index-boundary path. The exact
	// RecSet matcher and approximate hooks such as Bigram LIKE then prune the
	// same candidate batches before the residual predicate reads any columns.
	// This is an operator capability, not a planner shape rule: every
	// scan_recset whose input is a RecSet gets the same combined boundaries.
	boundaries := recSetFilterBoundaries(owner, conditionCols, condition)
	lower, upperLast := indexFromBoundaries(boundaries)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			conditionGetters[i] = getter
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	visibleUpper := mainCount + uint32(len(t.inserts))
	builder := newRecSetShardBuilder(t, visibleUpper, false, mainCount)
	evaluateOne := func(idx uint32) {
		if idx >= visibleUpper {
			return
		}
		if acidMode {
			if !currentTx.IsVisible(t, idx) {
				return
			}
		} else if t.deletions.Get(uint(idx)) {
			return
		}
		if idx < mainCount {
			for i, c := range cReaders {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else {
					cdataset[i] = c.GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
		}
		builder.add(idx, scm.ToBool(conditionFn(cdataset...)))
	}
	evaluateBatch := func(batch []uint32) bool {
		for _, idx := range batch {
			evaluateOne(idx)
		}
		return true
	}
	/* A tiny batch is already the cheapest exact candidate iterator; walking a
	whole inactive boundary index just to rediscover those IDs would defeat
	adaptive LIMIT. Broad RecSets take the normal boundary path, where LIKE and
	other hooks can reduce work below the incoming membership cardinality. */
	if part.count*4 < int64(visibleUpper) {
		part.forEachID(func(idx uint32) bool {
			evaluateOne(idx)
			return true
		})
	} else {
		var buf [1024]uint32
		t.iterateIndexMatchAware(currentTx, boundaries, lower, upperLast, len(t.inserts), buf[:], true, nil, evaluateBatch)
	}
	return builder.finish()
}

func recSetFilterBoundaries(owner *recSet, conditionCols []string, condition scm.Scmer) boundaries {
	result := extractBoundaries(conditionCols, condition)
	return append(result, NewIndexBoundary("$recset_contains", RecSetMatcher, NewRecSetScmer(owner), ""))
}

// filterToRecSet narrows r to the members that also satisfy condition,
// re-evaluating condition only over r's existing membership. See
// filterRecSetPart for why this is a distinct, cheaper operation than
// building a fresh RecSet over the whole table and intersecting.
func (r *recSet) filterToRecSet(currentTx *TxContext, conditionCols []string, condition scm.Scmer) *recSet {
	if r == nil {
		return nil
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	ss := SessionStateFromTx(currentTx)
	querySeq := scm.CurrentQuerySeq()
	result := &recSet{tx: currentTx, table: r.table}
	values := make(chan recSetBuildResult, len(r.shards))
	activeParts := make([]int, 0, len(r.shards))
	for i := range r.shards {
		if r.shards[i].count > 0 {
			activeParts = append(activeParts, i)
		}
	}
	done := runFanoutTasks(currentTx, len(activeParts), func(taskIndex int, _ bool) {
		part := r.shards[activeParts[taskIndex]]
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			values <- recSetBuildResult{
				part: part.shard.filterRecSetPart(r, &part, conditionCols, condition, currentTx, ss),
			}
			return scm.NewNil()
		})
	})
	if done != nil {
		<-done
	}
	close(values)

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if buildErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if buildErr.r != nil {
		panic(buildErr)
	}
	return result
}
