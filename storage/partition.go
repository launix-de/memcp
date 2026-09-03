/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
import "github.com/carli2/hybridsort"
import "sort"
import "sync"
import "sync/atomic"
import "time"
import "runtime"
import "strings"
import "github.com/launix-de/NonLockingReadMap"
import "github.com/launix-de/memcp/scm"

const repartitionDrainTimeout = 30 * time.Second

type shardDimension struct {
	Column        string
	NumPartitions int
	Pivots        []scm.Scmer // pivot semantics: a pivot is between two shards. shard[0] contains all values less than or equal pivot[0]; pivots are ordered from lowest to highest
}

// computes the index of a datapoint in PShards -> if item == pivot, sort left
func computeShardDimensionIndex(sd shardDimension, value scm.Scmer) int {
	min := 0                    // greater equal min
	max := sd.NumPartitions - 1 // smaller than max
	for min < max {
		pivot := (min + max - 1) / 2
		if scm.Less(sd.Pivots[pivot], value) {
			min = pivot + 1
		} else {
			max = pivot
		}
	}
	return min
}

func computeShardIndex(schema []shardDimension, values []scm.Scmer) (result int) {
	for i, sd := range schema {
		result = result*sd.NumPartitions + computeShardDimensionIndex(sd, values[i])
	}
	return // schema[0] has the highest stride; the last dimension is least significant
}

// runFanoutTasks executes a fanout without recursively multiplying worker
// pools. Transactions reserve at most half of their remaining budget. When
// only one worker is available, the caller performs the work directly without
// goroutines, channels, or wait groups.
func runFanoutTasks(currentTx *TxContext, taskCount int, task func(int, bool)) <-chan struct{} {
	if taskCount <= 0 {
		return nil
	}
	// This check must stay ahead of claimFanoutWorkers: point/range pruning to
	// one shard must not read, write, or invalidate the budget cache line.
	if taskCount == 1 {
		task(0, true)
		return nil
	}

	workers := runtime.GOMAXPROCS(0) / 2
	claimedWorkers := 0
	if currentTx != nil {
		claimedWorkers = currentTx.claimFanoutWorkers(taskCount)
		workers = claimedWorkers
	}
	if workers > taskCount {
		workers = taskCount
	}
	if workers < 2 {
		for i := 0; i < taskCount; i++ {
			task(i, true)
		}
		return nil
	}

	jobs := make(chan int, taskCount)
	doneCh := make(chan struct{})
	var remainingWorkers atomic.Int32
	remainingWorkers.Store(int32(workers))
	for i := 0; i < workers; i++ {
		go func() {
			defer func() {
				if remainingWorkers.Add(-1) == 0 {
					if currentTx != nil {
						currentTx.releaseFanoutWorkers(claimedWorkers)
					}
					close(doneCh)
				}
			}()
			for taskIndex := range jobs {
				task(taskIndex, false)
			}
		}()
	}
	for i := 0; i < taskCount; i++ {
		jobs <- i
	}
	close(jobs)
	return doneCh
}

// shardResultBufferSize bounds synchronous multi-shard result buffering. It
// snapshots only shard topology and does not touch the transaction fanout
// budget; callers reach it after boundary analysis has selected a table scan.
func (t *table) shardResultBufferSize() int {
	size := len(t.activeTopology().shards)
	if size < 1 {
		return 1
	}
	return size
}

func traceShardScanCallback(callback func(*storageShard, bool), shard *storageShard, solo bool) {
	scm.Trace.Duration(fmt.Sprintf("%p", shard), "shard", func() {
		callback(shard, solo)
	})
}

// runSingleShardScan keeps panic-safe resource release outside the topology
// retry loop. Defers inside that loop cannot be open-coded by Go and used to
// allocate a separate closure for every ordinary one-shard scan.
func runSingleShardScan(currentTx *TxContext, topology *tableShardTopology, shard *storageShard, callback func(*storageShard, bool)) {
	defer topology.releaseOperation()
	defer shard.activeScanners.Add(-1)
	release := shard.acquireReadForScan(currentTx)
	defer release()
	if scm.Trace == nil {
		callback(shard, true)
	} else {
		traceShardScanCallback(callback, shard, true)
	}
}

func runParallelShardScans(currentTx *TxContext, shards []*storageShard, topology *tableShardTopology, callback func(*storageShard, bool)) <-chan struct{} {
	return runFanoutTasks(currentTx, len(shards), func(i int, synchronous bool) {
		shard := shards[i]
		defer topology.releaseOperation()
		defer shard.activeScanners.Add(-1)
		release := shard.acquireReadForScan(currentTx)
		defer release()
		if scm.Trace == nil {
			callback(shard, synchronous)
		} else {
			traceShardScanCallback(callback, shard, synchronous)
		}
	})
}

func (t *table) iterateShardsParallel(currentTx *TxContext, boundaries []columnboundaries, callback func(*storageShard, bool)) <-chan struct{} {
	// Keep shard acquisition outside physical scan callbacks. In clustered mode
	// this is the orchestration point that can choose a local SHARED copy or send
	// the whole shard-local scan pipeline to a remote holder; row readers must not
	// perform another resource acquisition from inside the callback.
	for {
		topology := t.activeTopology()
		var relevant []*storageShard
		if topology.mode == ShardModeFree {
			relevant = topology.shards
			for _, shard := range relevant {
				if shard == nil {
					relevant = collectRelevantShards(nil, boundaries, topology.shards)
					break
				}
			}
		} else {
			relevant = collectRelevantShards(topology.dimensions, boundaries, topology.shards)
		}
		if len(relevant) == 0 {
			return nil
		}

		// Pin every shard task to the loaded immutable generation, then verify
		// that it is still authoritative. A publisher never waits for this
		// registration; late readers simply release it and retry on the new
		// generation before touching shard state.
		registered := 0
		for _, s := range relevant {
			if !topology.acquireOperation() {
				break
			}
			s.activeScanners.Add(1)
			registered++
		}
		if registered != len(relevant) || t.topology.Load() != topology {
			for _, s := range relevant[:registered] {
				s.activeScanners.Add(-1)
				topology.releaseOperation()
			}
			continue
		}

		if len(relevant) == 1 {
			runSingleShardScan(currentTx, topology, relevant[0], callback)
			return nil
		}
		return runParallelShardScans(currentTx, relevant, topology, callback)
	}
}

// acquireReadForScan obtains shard access unless this worker already owns the
// stronger write lock. Re-entering GetRead while holding shard.mu for a
// mutation would make its lazy-load checks deadlock on their own RLock.
func (s *storageShard) acquireReadForScan(currentTx *TxContext) func() {
	if s.hasWriteOwnerForTx(currentTx) {
		return func() {}
	}
	return s.GetRead()
}

func collectRelevantShards(schema []shardDimension, boundaries []columnboundaries, shards []*storageShard) []*storageShard {
	result := make([]*storageShard, 0, len(shards))
	collectRelevantShardsIndex(schema, boundaries, shards, &result)
	return result
}

func partitionForValue(dimension shardDimension, value scm.Scmer) int {
	return sort.Search(len(dimension.Pivots), func(i int) bool {
		return !scm.Less(dimension.Pivots[i], value)
	})
}

func valueEqualsPivot(dimension shardDimension, partition int, value scm.Scmer) bool {
	return partition < len(dimension.Pivots) &&
		!scm.Less(value, dimension.Pivots[partition]) &&
		!scm.Less(dimension.Pivots[partition], value)
}

func collectRelevantShardsIndex(schema []shardDimension, boundaries []columnboundaries, shards []*storageShard, result *[]*storageShard) {
	if len(schema) == 0 {
		for _, s := range shards {
			if s != nil {
				*result = append(*result, s)
			}
		}
		return
	}
	blockdim := 1 // shards[idx * blockdim:idx*blockdim+blockdim]
	for i := 1; i < len(schema); i++ {
		blockdim *= schema[i].NumPartitions
	}

	for _, b := range boundaries {
		if b.col == schema[0].Column {
			// iterate this axis over boundaries
			min := 0
			if !b.lower.IsNil() {
				min = partitionForValue(schema[0], b.lower)
				if !b.lowerInclusive && valueEqualsPivot(schema[0], min, b.lower) {
					min++
				}
			}

			max := schema[0].NumPartitions - 1 // smaller than max
			if !b.upper.IsNil() {
				max = partitionForValue(schema[0], b.upper)
			}

			for i := min; i <= max; i++ {
				collectRelevantShardsIndex(schema[1:], boundaries, shards[i*blockdim:(i+1)*blockdim], result)
			}
			return // finish (don't run into next boundary, don't run into the all-loop)
		}
	}

	// else: no boundaries: iterate all
	for i := 0; i < len(shards); i += blockdim {
		collectRelevantShardsIndex(schema[1:], boundaries, shards[i:i+blockdim], result)
	}
}

func (t *table) NewShardDimension(col string, n int) (result shardDimension) {
	result.Column = col
	if n < 1 {
		return // empty dimension
	}
	result.Pivots = make([]scm.Scmer, 0, n-1)

	// validate column exists in schema; if corrupted, abort loudly rather than proceeding
	hasCol := false
	for _, c := range t.Columns {
		if strings.EqualFold(c.Name, col) {
			hasCol = true
			col = c.Name // normalize to actual case
			break
		}
	}
	if !hasCol {
		panic("partition column does not exist: `" + t.schema.Name + "." + t.Name + "`.`" + col + "`")
	}

	// pivots are extracted from sampling
	shardlist := t.ActiveShards()
	pivotSamples := make([]scm.Scmer, 0, 2*len(shardlist))
	for _, s := range shardlist {
		if s == nil {
			continue
		}
		// Ensure shard and column are loaded. If metadata is corrupted, panic early
		// instead of proceeding with potentially destructive repartitioning.
		s.ensureLoaded()
		stor := s.getColumnStorageOrPanic(col, false, nil)
		// snapshot main_count without holding lock; guard indices and skip if inconsistent
		mc := s.main_count
		if mc > 0 {
			pivotSamples = append(pivotSamples, stor.GetValue(0))
		}
		if mc > 3 {
			pivotSamples = append(pivotSamples, stor.GetValue(mc-1))
		}
		for i := uint32(50); i < mc; i += 101 {
			pivotSamples = append(pivotSamples, stor.GetValue(i))
		}
	}
	if len(pivotSamples) == 0 {
		result.NumPartitions = 1
		return
	}

	// sort samplelist
	hybridsort.Slice(pivotSamples, func(i, j int) bool {
		return scm.Less(pivotSamples[i], pivotSamples[j])
	})
	// extract n-1 pivots
	for i := 1; i < n; i++ {
		sample := pivotSamples[(i*len(pivotSamples))/n]
		// only add new items
		if !sample.IsNil() && (len(result.Pivots) == 0 || scm.Less(result.Pivots[len(result.Pivots)-1], sample)) {
			result.Pivots = append(result.Pivots, sample)
		} else {
			// TODO: what if the sample is equal by chance?
		}
	}
	result.NumPartitions = len(result.Pivots) + 1

	return
}

type uintrange struct {
	min, max uint
}

type partitioningSet struct {
	shardid int
	items   map[int][]uint // TODO: use uintrange instead, so we don't need so much allocations
}

type shardPartitionSnapshot struct {
	deletions NonLockingReadMap.NonBlockingBitMap
	inserts   [][]scm.Scmer
	mainCount uint32
	mainCols  []ColumnStorage
	deltaCols []int
}

func (s *storageShard) snapshotPartitionState(schema []shardDimension) shardPartitionSnapshot {
	snap := shardPartitionSnapshot{
		mainCols:  make([]ColumnStorage, len(schema)),
		deltaCols: make([]int, len(schema)),
	}
	s.mu.RLock()
	snap.deletions = s.deletions.Copy()
	snap.inserts = append([][]scm.Scmer(nil), s.inserts...)
	snap.mainCount = s.main_count
	for i, sd := range schema {
		snap.mainCols[i], _ = s.columns[sd.Column]
		snap.deltaCols[i], _ = s.deltaColumns[sd.Column]
	}
	s.mu.RUnlock()
	return snap
}

func (snap shardPartitionSnapshot) count() uint64 {
	return uint64(snap.mainCount) + uint64(len(snap.inserts)) - uint64(snap.deletions.Count())
}

// partitionMainRows assigns each surviving (non-deleted) row of main storage
// in [0,mainCount) to a target shard. Deletion is a data-independent skip, so
// it is checked once up front to build the surviving recid list; each
// dimension column is then read for the whole batch via one GetValueMulti
// call instead of one GetValue per row per column.
func partitionMainRows(schema []shardDimension, mainCount uint32, deletions *NonLockingReadMap.NonBlockingBitMap, mainCols []ColumnStorage, result map[int][]uint32) {
	surviving := make([]uint32, 0, mainCount)
	for idx := uint32(0); idx < mainCount; idx++ {
		if !deletions.Get(uint(idx)) {
			surviving = append(surviving, idx)
		}
	}
	if len(surviving) == 0 {
		return
	}
	colValues := make([][]scm.Scmer, len(schema))
	for i, cs := range mainCols {
		colValues[i] = make([]scm.Scmer, len(surviving))
		cs.GetValueMulti(surviving, colValues[i], 1)
	}
	values := make([]scm.Scmer, len(schema))
	for row, idx := range surviving {
		for i := range schema {
			values[i] = colValues[i][row]
		}
		shardnum := computeShardIndex(schema, values)
		result[shardnum] = append(result[shardnum], idx)
	}
}

func (snap shardPartitionSnapshot) partition(schema []shardDimension) (result map[int][]uint32) {
	result = make(map[int][]uint32)
	values := make([]scm.Scmer, len(schema))

	partitionMainRows(schema, snap.mainCount, &snap.deletions, snap.mainCols, result)

	for idx, row := range snap.inserts {
		recid := snap.mainCount + uint32(idx)
		if snap.deletions.Get(uint(recid)) {
			continue
		}
		for i, colIdx := range snap.deltaCols {
			values[i] = row[colIdx]
		}
		shardnum := computeShardIndex(schema, values)
		result[shardnum] = append(result[shardnum], recid)
	}

	return
}

func (t *table) proposerepartition(maincount uint) (shardCandidates []shardDimension, shouldChange bool) { // this happens inside t.mu.Lock()
	// reevaluate partitioning schema
	for _, c := range t.Columns {
		if c.PartitioningScore > 0 {
			shardCandidates = append(shardCandidates, shardDimension{c.Name, c.PartitioningScore, nil})
		}
	}
	if len(shardCandidates) == 0 || Settings.PartitionMaxDimensions == 0 {
		return nil, true
	}

	// sort for highest ranking column
	hybridsort.Slice(shardCandidates, func(i, j int) bool { // Less
		return shardCandidates[i].NumPartitions > shardCandidates[j].NumPartitions
	})
	// prune shard candidates to max dimensions
	if len(shardCandidates) > Settings.PartitionMaxDimensions {
		shardCandidates = shardCandidates[:Settings.PartitionMaxDimensions]
	}
	// algorithm from the paper
	sf := 0.01 // scale factor
	best := 100000000
	bestSf := sf
	desiredNumberOfShards := (2*maincount)/Settings.ShardSize + 1 // TODO: find a balancing mechanism
	for iter := 2; iter < 300; iter++ {                           // find perfect scale factor such that we get the best number of shards
		deviation := 1
		for _, sc := range shardCandidates {
			deviation *= int(float64(sc.NumPartitions) * sf)
		}
		deviation -= int(desiredNumberOfShards)
		if deviation < 0 {
			if -deviation < best {
				best, bestSf = deviation, sf
			}
			// too few shards: increase sf
			sf = sf * (1.0 + 1.0/float64(iter))
		} else {
			if deviation < best {
				best, bestSf = deviation, sf
			}
			// too much shards: decrease sf
			sf = sf * (1.0 - 1.0/float64(iter))
		}
	}
	for i, sc := range shardCandidates {
		shardCandidates[i] = t.NewShardDimension(sc.Column, int(float64(sc.NumPartitions)*bestSf))
	}
	// remove empty dimensions
	for len(shardCandidates) > 0 && shardCandidates[len(shardCandidates)-1].NumPartitions <= 1 {
		shardCandidates = shardCandidates[:len(shardCandidates)-1]
	}
	if len(shardCandidates) == 0 {
		return
	}

	// check if we should change partitioning schema already
	if len(shardCandidates) != len(t.PDimensions) {
		shouldChange = true
	} else {
		totalShards1 := 1
		totalShards2 := 1
		for i, sc := range shardCandidates {
			if sc.Column != t.PDimensions[i].Column {
				shouldChange = true
			} else {
				totalShards1 *= sc.NumPartitions
				totalShards2 *= t.PDimensions[i].NumPartitions
			}
		}
		// deviation of >50% of shardsize
		if 2*totalShards1 > 3*totalShards2 || 2*totalShards2 > 3*totalShards1 {
			shouldChange = true
		}
	}
	return // the caller will evaluate shouldChange and shardCandidates
}

// repartition implements lock-free repartitioning with dual-write to prevent
// data loss. During repartition, concurrent inserts/updates/deletes are forwarded
// to both the old and new shard sets via repartitionDualWriteActive dual-write.
//
// Phases:
//
//	A. Prepare PShards (before releasing any locks)
//	B. Snapshot + activate dual-write atomically (under t.mu)
//	C. Build main storage (no locks held — long phase)
//	C½. Build translation map for DELETE dual-write
//	D. Install main storage + Delta shift (brief Lock per new shard)
//	F. Durably publish the new topology
//	G. Retire the old generation after its users drain
//
// This function is called WITHOUT t.mu held (t.mu is released by the caller
// before invoking repartition). It manages its own shard-level locking.
// maintenanceMu is already held and maintenanceKind is set to 2 by the caller.
func (t *table) repartition(shardCandidates []shardDimension) {
	t.schema.persistenceLifecycle.RLock()
	defer t.schema.persistenceLifecycle.RUnlock()
	t.ddlMu.RLock()
	defer t.ddlMu.RUnlock()
	t.repartitionDDLReadLocked(shardCandidates)
}

func (t *table) repartitionDDLReadLocked(shardCandidates []shardDimension) {
	// Safety-net: if somehow called directly without the flag being set.
	if t.maintenanceKind == 2 && t.PShards != nil {
		return
	}

	// If no shard candidates, fall back to parallel sharding based on data size
	if len(shardCandidates) == 0 {
		totalRows := uint(0)
		shards := t.ActiveShards()
		for _, s := range shards {
			if s != nil {
				totalRows += uint(s.Count())
			}
		}
		desiredShards := int(1 + (2*totalRows)/Settings.ShardSize)
		minShards := 2 * runtime.NumCPU()
		if desiredShards < minShards && totalRows > Settings.ShardSize {
			desiredShards = minShards
		}
		if desiredShards > 1 && len(t.Columns) > 0 {
			shardCandidates = []shardDimension{t.NewShardDimension(t.Columns[0].Name, desiredShards)}
		}
	}

	totalShards := 1
	for _, sc := range shardCandidates {
		totalShards *= sc.NumPartitions
	}

	fmt.Println("repartitioning", t.Name, "by", shardCandidates, "into", totalShards, "shards")
	start := time.Now()

	t.mu.Lock()
	oldTopology := t.activeTopology()
	oldshards := append([]*storageShard(nil), oldTopology.shards...)
	oldShardMode := t.ShardMode
	oldFreeShards := append([]*storageShard(nil), t.Shards...)
	oldPartitionShards := append([]*storageShard(nil), t.PShards...)
	oldPartitionDimensions := append([]shardDimension(nil), t.PDimensions...)
	t.mu.Unlock()

	// Eagerly load all shard data before taking any locks for partitioning.
	for _, s := range oldshards {
		if s == nil {
			continue
		}
		s.ensureLoaded()
		s.mu.Lock()
		for _, sd := range shardCandidates {
			if _, ok := s.columns[sd.Column]; ok {
				s.ensureColumnLoaded(sd.Column, true)
			}
		}
		for _, col := range t.Columns {
			if _, ok := s.columns[col.Name]; ok {
				s.ensureColumnLoaded(col.Name, true)
			}
		}
		s.ensureMainCount(true)
		s.mu.Unlock()
	}

	// ── Phase A: Prepare PShards and activate dual-write ──
	// Create empty new shards and set repartitionActive BEFORE releasing locks,
	// so concurrent writes are forwarded to both shard sets.
	newshards := make([]*storageShard, totalShards)
	for i := range newshards {
		newshards[i] = NewShard(t)
		newshards[i].srState = WRITE // live shard, not cold
		if t.PersistencyMode == Safe || t.PersistencyMode == Logged {
			newshards[i].logfile = t.schema.persistence.OpenLog(newshards[i].uuid.String())
		}
	}
	abortRepartition := func() {
		t.mu.Lock()
		t.ShardMode = oldShardMode
		t.Shards = oldFreeShards
		t.PShards = oldPartitionShards
		t.PDimensions = oldPartitionDimensions
		t.maintenanceKind = 0
		t.repartitionDualWriteActive.Store(false)
		t.repartitionSources.Store(nil)
		t.mu.Unlock()
		t.clearRepartitionTranslation()
		t.repartitionPendingMu.Lock()
		t.repartitionPendingDels = nil
		t.repartitionPendingSourceDels = nil
		t.repartitionPendingMu.Unlock()
		t.maintenanceMu.Unlock()
		for _, shard := range newshards {
			discardUnpublishedShard(shard)
		}
	}
	// maintenanceKind was already set to 2 by the caller (db.rebuild or
	// beginManualRepartition) under maintenanceMu.
	// From this point, all concurrent inserts/updates go to BOTH shard sets.

	// ── Phase B: Snapshot under t.mu, partition off-lock ──
	// Hold t.mu only while taking cheap per-shard snapshots and enabling
	// dual-write. The expensive partition scan runs on the immutable snapshots
	// afterwards, so INSERTs are blocked only for the narrow publication window.
	snapshots := make([]shardPartitionSnapshot, len(oldshards))
	datasetids := make([][][]uint32, totalShards) // newshard -> oldshard -> []rowIdx
	total_count := uint64(0)
	t.mu.Lock()
	t.PShards = newshards
	t.PDimensions = shardCandidates
	for si, s := range oldshards {
		snapshots[si] = s.snapshotPartitionState(shardCandidates)
	}
	sourceSet := &repartitionSourceSet{
		shards: append([]*storageShard(nil), oldshards...),
		set:    make(map[*storageShard]struct{}, len(oldshards)),
	}
	for _, shard := range oldshards {
		sourceSet.set[shard] = struct{}{}
	}
	t.repartitionSources.Store(sourceSet)
	t.setRepartitionTranslationMap(make(map[*storageShard]map[uint32]translatedRecid))
	t.repartitionDualWriteActive.Store(true)
	t.mu.Unlock()

	for si, snap := range snapshots {
		total_count += snap.count()
		for idx, items := range snap.partition(shardCandidates) {
			if datasetids[idx] == nil {
				datasetids[idx] = make([][]uint32, len(oldshards))
			}
			datasetids[idx][si] = items
		}
	}
	// Build the snapshot portion of the translation map once the partition scan
	// is done. Concurrent Phase-C dual-writes extend the same map with delta rows.
	translationMap := make(map[*storageShard]map[uint32]translatedRecid)
	for nsi, items := range datasetids {
		if items == nil {
			continue
		}
		for si, oldItems := range items {
			if len(oldItems) == 0 {
				continue
			}
			shard := oldshards[si]
			if translationMap[shard] == nil {
				translationMap[shard] = make(map[uint32]translatedRecid)
			}
			// Compute offset: rows from previous old shards in this PShard
			offset := uint32(0)
			for prevSi := 0; prevSi < si; prevSi++ {
				offset += uint32(len(items[prevSi]))
			}
			for localIdx, oldRecid := range oldItems {
				translationMap[shard][oldRecid] = translatedRecid{
					pshardIdx: nsi,
					newRecid:  offset + uint32(localIdx),
					inDelta:   false,
				}
			}
		}
	}
	t.mergeRepartitionTranslations(translationMap)
	t.resolvePendingRepartitionSourceDeletes()

	// ── Phase C: Build main storage (no locks held — long phase) ──
	// Build column storage into temporary per-shard maps. We must NOT touch
	// the shard's main_count or columns while dual-write inserts are running
	// concurrently (they read main_count under the shard lock to compute recids).
	type builtShardData struct {
		columns   map[string]ColumnStorage
		mainCount uint32
	}
	builtData := make([]builtShardData, len(newshards))

	fmt.Println("moving data from", t.Name, len(oldshards), "into", totalShards, "shards")
	var done sync.WaitGroup
	var buildErrMu sync.Mutex
	var buildErrors []any
	done.Add(totalShards)
	workers := runtime.NumCPU() / 2
	if workers < 1 {
		workers = 1
	}
	progress := make(chan int, workers)
	for i := 0; i < workers; i++ {
		go func() {
			for si := range progress {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Println("error: repartition shard build failed for", t.schema.Name+".", t.Name, "shard", si, ":", r)
							buildErrMu.Lock()
							buildErrors = append(buildErrors, r)
							buildErrMu.Unlock()
						}
						done.Done()
					}()
					s := newshards[si]
					built := &builtData[si]
					built.columns = make(map[string]ColumnStorage)
					// Count main rows for this new shard
					mainCount := uint32(0)
					for _, items := range datasetids[si] {
						mainCount += uint32(len(items))
					}
					built.mainCount = mainCount
					// Allocate column storage and build
					values := make([]scm.Scmer, mainCount)
					for _, col := range t.Columns {
						// Check if ANY old shard has a StorageComputeProxy for this column.
						// If so, port the proxy (computor + validMask) instead of evaluating values.
						var proxyTemplate *StorageComputeProxy
						for _, os := range oldshards {
							os.mu.RLock()
							if cs, ok := os.columns[col.Name]; ok {
								if p, ok := cs.(*StorageComputeProxy); ok {
									proxyTemplate = p
									os.mu.RUnlock()
									break
								}
							}
							os.mu.RUnlock()
						}

						if proxyTemplate != nil {
							// Port StorageComputeProxy: create new proxy for this PShard,
							// copy valid values and validMask, leave invalid rows for lazy recompute.
							newProxy := &StorageComputeProxy{
								delta:     make(map[uint32]scm.Scmer),
								computor:  proxyTemplate.computor,
								inputCols: proxyTemplate.inputCols,
								shard:     s, // back-reference to new PShard
								colName:   proxyTemplate.colName,
								count:     mainCount,
								isOrdered: proxyTemplate.isOrdered,
							}
							// Port values and validMask from old shards
							var newIdx uint32
							for s2id, items := range datasetids[si] {
								oldShard := oldshards[s2id]
								oldShard.mu.RLock()
								oldProxy, isProxy := oldShard.columns[col.Name].(*StorageComputeProxy)
								oldShard.mu.RUnlock()
								if !isProxy {
									// Old shard has plain storage for this column — read values directly
									reader := oldShard.ColumnReaderTx(nil, col.Name)
									for _, item := range items {
										val := reader(uint32(item))
										newProxy.delta[newIdx] = val
										newProxy.validMask.Set(uint(newIdx), true)
										newIdx++
									}
								} else {
									oldRowIDs := make([]uint32, len(items))
									for i, item := range items {
										oldRowIDs[i] = uint32(item)
									}
									newIdx = appendComputeProxyRows(newProxy, oldProxy, oldRowIDs, newIdx)
								}
							}
							built.columns[col.Name] = newProxy
							// Compute proxies are not serialized to disk
						} else {
							// Normal column: read values and compress
							var i uint32
							for s2id, items := range datasetids[si] {
								reader := oldshards[s2id].ColumnReaderTx(nil, col.Name)
								for _, item := range items {
									values[i] = reader(uint32(item))
									i++
								}
							}
							// Compress into optimal storage format
							var newcol ColumnStorage = new(StorageSCMER)
							for {
								newcol.prepare()
								for j, v := range values {
									newcol.scan(uint32(j), v)
								}
								newcol2 := newcol.proposeCompression(uint32(i))
								if newcol2 == nil {
									break
								}
								newcol = newcol2
							}
							if blob, ok := newcol.(*OverlayBlob); ok {
								blob.schema = s.t.schema
							}
							// TODO: when source column is OverlayBlob, shuffle raw
							// compressed blob data without decompressing+recompressing.
							// Copy hash references and blob files directly to avoid
							// the gzip round-trip during repartition.
							newcol.init(uint32(mainCount))
							for j, v := range values {
								newcol.build(uint32(j), v)
							}
							newcol.finish()
							// Store in temporary map (NOT on shard — shard is live for dual-write)
							built.columns[col.Name] = newcol
							// Write to disk
							if s.t.PersistencyMode != Memory {
								f := s.t.schema.persistence.WriteColumn(s.uuid.String(), col.Name)
								newcol.Serialize(f)
								finishColumnWrite(f, s.t.PersistencyMode == Safe)
							}
						}
					}
				}()
			}
		}()
	}
	for si := range newshards {
		progress <- si
		fmt.Println("rebuild", t.Name, si+1, "/", len(newshards))
	}
	close(progress) // signal workers to exit after processing all items
	done.Wait()
	if len(buildErrors) > 0 {
		err := buildErrors[0]
		abortRepartition()
		panic(err)
	}

	// ── Phase D: Install main storage + Delta shift ──
	// Under the shard lock, install the built columns and main_count, then
	// shift all dual-write delta storage. During Phase C, all dual-write
	// inserts used main_count=0, so their recids are in [0, deltaLen).
	// After installing main_count=N, they need to be shifted to [N, N+deltaLen).
	for si, s := range newshards {
		s.mu.Lock()
		built := builtData[si]
		mainN := built.mainCount
		// Install built column storage
		for name, col := range built.columns {
			s.columns[name] = col
		}
		// Install main_count — from this point, new inserts get correct recids
		s.main_count = uint32(mainN)
		s.plannerMainRows.Store(uint32(mainN))
		deltaLen := len(s.inserts)
		if deltaLen > 0 {
			// Shift deletion bitmap: bits in [0, deltaLen) move to [mainN, mainN+deltaLen).
			// Translation-mapped DELETE dual-writes may have set bits in [0, mainN)
			// during Phase C. Collect those first, clear the entire [0, deltaLen+mainN)
			// range carefully, shift delta bits, then re-apply main-storage deletions.
			//
			// Simpler approach: snapshot delta-range bits, clear them, shift.
			// Then re-apply any main-storage deletion bits from the pendingMainDels
			// list (collected by dualWriteDelete via repartitionPendingDels).
			deltaDels := make([]bool, deltaLen)
			for i := 0; i < deltaLen; i++ {
				if s.deletions.Get(uint(i)) {
					deltaDels[i] = true
					s.deletions.Set(uint(i), false)
				}
			}
			for i := 0; i < deltaLen; i++ {
				if deltaDels[i] {
					s.deletions.Set(uint(uint32(mainN)+uint32(i)), true)
				}
			}
			// Shift delta btree indexes
			for _, index := range s.Indexes {
				if index.baseState.deltaBtree != nil {
					// Rebuild with shifted recids
					items := make([]indexPair, 0)
					index.baseState.deltaBtree.Ascend(func(item indexPair) bool {
						items = append(items, indexPair{itemid: item.itemid + int(mainN), data: item.data})
						return true
					})
					index.baseState.deltaBtree.Clear(false)
					for _, item := range items {
						index.baseState.deltaBtree.ReplaceOrInsert(item)
					}
				}
			}
		}
		// Staging inserts were assigned recids while main_count was still zero.
		// Rebuild their unpublished WAL after the main prefix is installed so a
		// restart observes the same shifted recids as the live shard.
		if t.PersistencyMode == Safe || t.PersistencyMode == Logged {
			if s.logfile != nil {
				s.logfile.Close()
			}
			t.schema.persistence.RemoveLog(s.uuid.String())
			s.logfile = t.schema.persistence.OpenLog(s.uuid.String())
			if len(s.inserts) > 0 {
				columns, values := s.materializedInsertedRowsLocked(0)
				s.logfile.Write(LogEntryInsert{columns, values})
				for i := range s.inserts {
					recid := s.main_count + uint32(i)
					if s.deletions.Get(uint(recid)) {
						s.logfile.Write(LogEntryDelete{recid})
					}
				}
			}
			if t.PersistencyMode == Safe {
				s.logfile.Sync()
			}
		}
		s.mu.Unlock()
	}
	mainCounts := make([]uint32, len(newshards))
	for i, built := range builtData {
		mainCounts[i] = built.mainCount
	}
	// Blob ownership belongs to the unpublished shard generation. Persist its
	// manifest before schema.json can make that generation authoritative.
	for _, shard := range newshards {
		writeBlobManifest(shard)
	}
	applyPendingDeletes := func() int {
		t.repartitionPendingMu.Lock()
		pending := t.repartitionPendingDels
		t.repartitionPendingDels = nil
		t.repartitionPendingMu.Unlock()
		for _, tr := range pending {
			ps := newshards[tr.pshardIdx]
			recid := tr.newRecid
			if tr.inDelta {
				recid += mainCounts[tr.pshardIdx]
			}
			ps.mu.Lock()
			wasDeleted := ps.deletions.Get(uint(recid))
			ps.deletions.Set(uint(recid), true)
			if !wasDeleted {
				ps.logVisibilityChangeLocked(recid, true)
			}
			ps.mu.Unlock()
			if !wasDeleted && t.PersistencyMode == Safe && ps.logfile != nil {
				ps.logfile.Sync()
			}
		}
		return len(pending)
	}
	t.shiftDeltaRepartitionTranslations(mainCounts)
	t.resolvePendingRepartitionSourceDeletes()

	// Apply pending main-storage deletions that were buffered during Phase C.
	totalPending := 0
	for {
		t.repartitionPendingMu.Lock()
		pending := t.repartitionPendingDels
		t.repartitionPendingDels = nil
		t.repartitionPendingMu.Unlock()
		if len(pending) == 0 {
			break
		}
		totalPending += len(pending)
		t.repartitionPendingMu.Lock()
		t.repartitionPendingDels = append(t.repartitionPendingDels, pending...)
		t.repartitionPendingMu.Unlock()
		applyPendingDeletes()
	}
	if totalPending > 0 {
		fmt.Println("repartition: applied", totalPending, "pending deletions after Phase D")
	}

	// ── Phase F: Durable publication + topology flip ──
	// Keep the old immutable topology authoritative while schema.json is being
	// committed. Source writes remain synchronous old→new forwards. Locking the
	// old shards only for this short commit boundary prevents a writer from
	// completing between the final catch-up and durable metadata publication.
	startDeadline := time.Now().Add(repartitionDrainTimeout)
	waitWithDeadline := func(step string, cond func() bool) {
		for !cond() {
			if time.Now().After(startDeadline) {
				panic(fmt.Sprintf("repartition %s timed out after %s while %s", t.Name, repartitionDrainTimeout, step))
			}
			runtime.Gosched()
		}
	}

	lockAllOldShards := func() []*storageShard {
		locked := make([]*storageShard, 0, len(oldshards))
		for _, shard := range oldshards {
			if shard == nil {
				continue
			}
			if !shard.mu.TryLock() {
				for i := len(locked) - 1; i >= 0; i-- {
					locked[i].mu.Unlock()
				}
				return nil
			}
			locked = append(locked, shard)
		}
		return locked
	}

	var oldshardsLocked []*storageShard
	for {
		waitWithDeadline("draining shard transactions before phase F lock acquisition", func() bool {
			for _, shard := range oldshards {
				if shard == nil {
					continue
				}
				if shard.activeTransactions.Load() > 0 {
					return false
				}
			}
			return true
		})
		locked := lockAllOldShards()
		if locked == nil {
			if time.Now().After(startDeadline) {
				panic(fmt.Sprintf("repartition %s timed out after %s while acquiring phase F old-shard locks", t.Name, repartitionDrainTimeout))
			}
			runtime.Gosched()
			continue
		}
		active := false
		for _, shard := range oldshards {
			if shard == nil {
				continue
			}
			if shard.activeTransactions.Load() > 0 {
				active = true
				break
			}
		}
		if !active {
			oldshardsLocked = locked
			break
		}
		for i := len(locked); i > 0; i-- {
			locked[i-1].mu.Unlock()
		}
	}
	// With source writers stopped, every completed pre-publication delete must
	// now have a translation and be durable in the new generation.
	t.resolvePendingRepartitionSourceDeletes()
	t.repartitionPendingMu.Lock()
	unresolvedBeforePublication := len(t.repartitionPendingSourceDels)
	t.repartitionPendingMu.Unlock()
	if unresolvedBeforePublication != 0 {
		for i := len(oldshardsLocked) - 1; i >= 0; i-- {
			oldshardsLocked[i].mu.Unlock()
		}
		abortRepartition()
		panic(fmt.Sprintf("repartition cannot publish %s: %d source deletes have no target translation", t.Name, unresolvedBeforePublication))
	}
	applyPendingDeletes()
	t.mu.Lock()
	t.PShards = newshards
	t.PDimensions = shardCandidates
	t.ShardMode = ShardModePartition
	t.Shards = nil
	t.mu.Unlock()
	var savePanic any
	func() {
		defer func() { savePanic = recover() }()
		t.schema.save()
	}()
	if savePanic != nil {
		for i := len(oldshardsLocked) - 1; i >= 0; i-- {
			oldshardsLocked[i].mu.Unlock()
		}
		abortRepartition()
		panic(savePanic)
	}
	t.mu.Lock()
	t.publishTopologyLocked()
	t.mu.Unlock()
	for i := len(oldshardsLocked) - 1; i >= 0; i-- {
		oldshardsLocked[i].mu.Unlock()
	}

	finishRetirement := func() {
		// Every scan, insert, and transaction that selected the old immutable
		// topology has now left it. No writer was stalled by that drain: old
		// mutation pipelines kept forwarding into the published generation.
		t.resolvePendingRepartitionSourceDeletes()
		t.repartitionPendingMu.Lock()
		unresolvedFinal := len(t.repartitionPendingSourceDels)
		t.repartitionPendingMu.Unlock()
		if unresolvedFinal != 0 {
			panic(fmt.Sprintf("repartition published %s with %d unresolved source deletes; old generation retained", t.Name, unresolvedFinal))
		}
		finalPending := applyPendingDeletes()
		if finalPending > 0 {
			fmt.Println("repartition: applied", finalPending, "final pending deletions after generation retirement")
		}

		total_count2 := uint64(0)
		for _, s := range newshards {
			total_count2 += uint64(s.Count())
		}
		if total_count2 < total_count {
			diff := total_count - total_count2
			if diff > total_count/10 {
				fmt.Println("warning: repartition count mismatch for", t.Name, ": before", total_count, "after", total_count2, "(", diff, "rows missing)")
			}
		}
		fmt.Println("activated new partitioning schema for", t.Name, "after", time.Since(start))

		t.mu.Lock()
		t.maintenanceKind = 0
		t.repartitionDualWriteActive.Store(false)
		t.repartitionSources.Store(nil)
		t.mu.Unlock()
		t.clearRepartitionTranslation()
		t.maintenanceMu.Unlock()

		for _, s := range oldshards {
			// Deliberately after successful schema publication and after the last
			// generation user. Persistent cleanup must never run on an active shard.
			GlobalCache.Remove(s)
			s.RemoveFromDisk()
		}

		if t.PersistencyMode == Cache && !t.isEphemeralQueryTable() {
			for _, s := range newshards {
				atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
				GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeCacheEntry, cacheShardCleanup, shardLastUsed, nil)
			}
		} else if t.PersistencyMode != Memory && !t.isEphemeralQueryTable() {
			for _, s := range newshards {
				atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
				GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
			}
		}
	}

	select {
	case <-oldTopology.drained:
		finishRetirement()
	default:
		go func() {
			<-oldTopology.drained
			finishRetirement()
		}()
	}
}

func (s *storageShard) partition(schema []shardDimension) (result map[int][]uint32) {
	// assigns each dataset into a target shard
	result = make(map[int][]uint32)

	/* this is already done from outside and all locks are kept until the rebuild is done
	s.mu.RLock() // TODO: somehow seal that shard such that future inserts/deletes are blocked or forwarded
	defer s.mu.RUnlock()
	*/
	values := make([]scm.Scmer, len(schema))

	/* collect main storage */
	maincols := make([]ColumnStorage, len(schema))
	for i, sd := range schema {
		maincols[i], _ = s.columns[sd.Column]
	}
	partitionMainRows(schema, s.main_count, &s.deletions, maincols, result)

	/* collect delta storage */
	deltacols := make([]int, len(schema))
	for i, sd := range schema {
		deltacols[i], _ = s.deltaColumns[sd.Column]
	}
	for idx, dataset := range s.inserts {
		if s.deletions.Get(uint(s.main_count + uint32(idx))) {
			continue
		}
		for i, cs := range deltacols {
			values[i] = dataset[cs]
		}
		shardnum := computeShardIndex(schema, values)
		oldlist, _ := result[shardnum]
		result[shardnum] = append(oldlist, s.main_count+uint32(idx))
	}

	return
}
