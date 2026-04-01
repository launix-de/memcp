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
import "sync"
import "sync/atomic"
import "time"
import "runtime"
import "strings"
import "github.com/jtolds/gls"
import "github.com/launix-de/NonLockingReadMap"
import "github.com/launix-de/memcp/scm"

type shardDimension struct {
	Column        string
	NumPartitions int
	Pivots        []scm.Scmer // pivot semantics: a pivot is between two shards. shard[0] contains all values less than or equal pivot[0]; pivots are ordered from lowest to highest
}

// computes the index of a datapoint in PShards -> if item == pivot, sort left
func computeShardIndex(schema []shardDimension, values []scm.Scmer) (result int) {
	for i, sd := range schema {
		// get slice idx of this dimension
		min := 0                    // greater equal min
		max := sd.NumPartitions - 1 // smaller than max
		for min < max {
			pivot := (min + max - 1) / 2
			if scm.Less(sd.Pivots[pivot], values[i]) {
				min = pivot + 1
			} else {
				max = pivot
			}
		}
		result = result*sd.NumPartitions + min // accumulate
	}
	return // schema[0] has the higest stride; schema[len(schema)-1] is the least significant bit
}

func (t *table) iterateShardsParallel(boundaries []columnboundaries, callback_old func(*storageShard, bool)) <-chan struct{} {
	callback := callback_old
	if scm.Trace != nil {
		// hook on tracing
		callback = func(s *storageShard, solo bool) {
			scm.Trace.Duration(fmt.Sprintf("%p", s), "shard", func() {
				callback_old(s, solo)
			})
		}
	}

	runWorkers := func(shards []*storageShard, onDone func(*storageShard)) <-chan struct{} {
		workers := runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
		if workers > len(shards) {
			workers = len(shards)
		}

		jobs := make(chan *storageShard, len(shards))
		doneCh := make(chan struct{})
		var done sync.WaitGroup
		done.Add(len(shards))
		for i := 0; i < workers; i++ {
			gls.Go(func() {
				for s := range jobs {
					release := s.GetRead()
					callback(s, false)
					release()
					if onDone != nil {
						onDone(s)
					}
					done.Done()
				}
			})
		}
		for _, s := range shards {
			jobs <- s
		}
		close(jobs)
		go func() {
			done.Wait()
			close(doneCh)
		}()
		return doneCh
	}

	// Hold shardModeMu.RLock while reading ShardMode and capturing shard list.
	// Phase F's drain uses shardModeMu.Lock() to synchronize, ensuring all
	// iterateShards calls that read FreeShard have incremented activeScanners
	// before the drain check begins.
	t.shardModeMu.RLock()
	mode := t.ShardMode
	if mode == ShardModeFree {
		shards := t.Shards
		// Increment activeScanners while holding shardModeMu.RLock so Phase F
		// sees all in-flight scans after its shardModeMu.Lock()/Unlock().
		relevant := make([]*storageShard, 0, len(shards))
		for _, s := range shards {
			if s != nil {
				s.activeScanners.Add(1)
				relevant = append(relevant, s)
			}
		}
		t.shardModeMu.RUnlock()

		if len(relevant) == 0 {
			return nil
		}
		if len(relevant) == 1 {
			s := relevant[0]
			release := s.GetRead()
			callback(s, true)
			release()
			s.activeScanners.Add(-1)
			return nil
		}
		return runWorkers(relevant, func(s *storageShard) {
			s.activeScanners.Add(-1)
		})
	} else {
		t.shardModeMu.RUnlock()
		relevant := collectRelevantShards(t.PDimensions, boundaries, t.PShards)
		if len(relevant) == 0 {
			return nil
		}
		if len(relevant) == 1 {
			s := relevant[0]
			release := s.GetRead()
			callback(s, true)
			release()
			return nil
		}
		return runWorkers(relevant, nil)
	}
}

func collectRelevantShards(schema []shardDimension, boundaries []columnboundaries, shards []*storageShard) []*storageShard {
	result := make([]*storageShard, 0, len(shards))
	collectRelevantShardsIndex(schema, boundaries, shards, &result)
	return result
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
				// lower bound is given -> find lowest part
				max := schema[0].NumPartitions - 1
				for min < max {
					pivot := (min + max - 1) / 2
					if !b.lowerInclusive {
						if scm.Less(b.lower, schema[0].Pivots[pivot]) {
							max = pivot
						} else {
							min = pivot + 1
						}
					} else {
						if !scm.Less(schema[0].Pivots[pivot], b.lower) {
							max = pivot
						} else {
							min = pivot + 1
						}
					}
				}
			}

			max := schema[0].NumPartitions - 1 // smaller than max
			if !b.upper.IsNil() {
				// upper bound is given -> find highest part
				umin := min
				for umin < max {
					pivot := (umin + max - 1) / 2
					if !b.upperInclusive {
						if scm.Less(b.upper, schema[0].Pivots[pivot]) {
							umin = pivot + 1
						} else {
							max = pivot
						}
					} else {
						if !scm.Less(schema[0].Pivots[pivot], b.upper) {
							umin = pivot + 1
						} else {
							max = pivot
						}
					}
				}
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
		stor := s.getColumnStorageOrPanic(col)
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

func (snap shardPartitionSnapshot) partition(schema []shardDimension) (result map[int][]uint32) {
	result = make(map[int][]uint32)
	values := make([]scm.Scmer, len(schema))

	for idx := uint32(0); idx < snap.mainCount; idx++ {
		if snap.deletions.Get(uint(idx)) {
			continue
		}
		for i, cs := range snap.mainCols {
			values[i] = cs.GetValue(idx)
		}
		shardnum := computeShardIndex(schema, values)
		result[shardnum] = append(result[shardnum], idx)
	}

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
//	F. Flip ShardMode + Drain
//	G. Cleanup
//
// This function is called WITHOUT t.mu held (t.mu is released by the caller
// before invoking repartition). It manages its own shard-level locking.
// maintenanceMu is already held and maintenanceKind is set to 2 by the caller.
func (t *table) repartition(shardCandidates []shardDimension) {
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

	oldshards := t.ActiveShards()

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
	t.PShards = newshards
	t.PDimensions = shardCandidates
	// maintenanceKind was already set to 2 by the caller (db.rebuild or
	// beginManualRepartition) under maintenanceMu.
	// From this point, all concurrent inserts/updates go to BOTH shard sets.

	// ── Phase B: Snapshot + activate dual-write atomically ──
	// Hold t.mu while taking all snapshots and activating dual-write.
	// This closes the INSERT loss window: any INSERT that commits after the
	// last snapshot will see the flag and dual-write.
	datasetids := make([][][]uint32, totalShards) // newshard -> oldshard -> []rowIdx
	total_count := uint64(0)
	t.mu.Lock()
	for si, s := range oldshards {
		snap := s.snapshotPartitionState(shardCandidates)
		total_count += snap.count()
		for idx, items := range snap.partition(shardCandidates) {
			if datasetids[idx] == nil {
				datasetids[idx] = make([][]uint32, len(oldshards))
			}
			datasetids[idx][si] = items
		}
	}
	// Build translation map immediately so DELETE dual-write works during Phase C.
	// The mapping (oldShard, oldRecid) → (pshardIdx, newRecid) is fully determined
	// by datasetids from the snapshot; it does not depend on Phase C's column build.
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
				}
			}
		}
	}
	t.repartitionTranslation = translationMap
	t.repartitionDualWriteActive.Store(true)
	t.mu.Unlock()

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
									reader := oldShard.ColumnReader(col.Name)
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
								reader := oldshards[s2id].ColumnReader(col.Name)
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
								f.Close()
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
			// Rebuild hashmaps with shifted keys
			for k, v := range s.hashmaps1 {
				newmap := make(map[[1]scm.Scmer]uint32, len(v))
				for key, recid := range v {
					newmap[key] = recid + uint32(mainN)
				}
				s.hashmaps1[k] = newmap
			}
			for k, v := range s.hashmaps2 {
				newmap := make(map[[2]scm.Scmer]uint32, len(v))
				for key, recid := range v {
					newmap[key] = recid + uint32(mainN)
				}
				s.hashmaps2[k] = newmap
			}
			for k, v := range s.hashmaps3 {
				newmap := make(map[[3]scm.Scmer]uint32, len(v))
				for key, recid := range v {
					newmap[key] = recid + uint32(mainN)
				}
				s.hashmaps3[k] = newmap
			}
			// Shift delta btree indexes
			for _, index := range s.Indexes {
				if index.deltaBtree != nil {
					// Rebuild with shifted recids
					items := make([]indexPair, 0)
					index.deltaBtree.Ascend(func(item indexPair) bool {
						items = append(items, indexPair{item.itemid + int(mainN), item.data})
						return true
					})
					index.deltaBtree.Clear(false)
					for _, item := range items {
						index.deltaBtree.ReplaceOrInsert(item)
					}
				}
			}
		}
		s.mu.Unlock()
	}

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
		for _, tr := range pending {
			ps := newshards[tr.pshardIdx]
			ps.mu.Lock()
			ps.deletions.Set(uint(tr.newRecid), true)
			ps.mu.Unlock()
		}
	}
	if totalPending > 0 {
		fmt.Println("repartition: applied", totalPending, "pending deletions after Phase D")
	}

	// ── Phase F: Flip ShardMode + Drain ──
	// Step 1: Flip ShardMode so new scans/inserts use PShards.
	t.PShards = newshards
	t.PDimensions = shardCandidates
	t.ShardMode = ShardModePartition
	// Step 2: Acquire/release shardModeMu to synchronize with iterateShards.
	// After this, all iterateShards calls that captured FreeShard mode have
	// incremented activeScanners on old shards. New iterateShards calls see
	// ShardModePartition and use PShards.
	t.shardModeMu.Lock()
	t.shardModeMu.Unlock()
	// Step 3: Wait for all in-flight scans on old shards to complete.
	// During this drain, maintenanceKind == 2 is still true, so dual-write
	// continues forwarding writes from old shards to PShards.
	for _, s := range oldshards {
		for s.activeScanners.Load() > 0 {
			runtime.Gosched()
		}
	}
	// Step 4: Drain any in-flight mutating scan pipelines (UPDATE / DELETE) that
	// still operate on the old shard set. They hold mutationMu for the whole
	// scan+callback lifetime.
	t.mutationMu.Lock()
	t.mutationMu.Unlock()
	// Step 5: Drain any in-flight partition-path dual-writes that write to old
	// Shards (Partition→Shards direction). These hold t.mu briefly.
	t.mu.Lock()
	t.mu.Unlock()

	// Final drain of pending deletions. After Phase F drain, all in-flight
	// DELETE scans have completed and their dual-write callbacks have finished.
	// Any remaining pending entries are now final.
	t.repartitionPendingMu.Lock()
	finalPending := t.repartitionPendingDels
	t.repartitionPendingDels = nil
	t.repartitionPendingMu.Unlock()
	for _, tr := range finalPending {
		ps := newshards[tr.pshardIdx]
		ps.mu.Lock()
		ps.deletions.Set(uint(tr.newRecid), true)
		ps.mu.Unlock()
	}
	if len(finalPending) > 0 {
		fmt.Println("repartition: applied", len(finalPending), "final pending deletions after Phase F drain")
	}

	// Verify transformation result
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

	// ── Phase G: Cleanup ──
	t.schema.schemalock.Lock()
	t.schema.saveLockedWithDurabilityAndUnlock(t.schemaWriteDurable())

	// Nil out old shards after the schema is saved, so no FreeShard
	// code path can reference them. At this point, ShardMode is Partition,
	// so new inserts use PShards exclusively.
	t.mu.Lock()
	t.Shards = nil
	t.maintenanceKind = 0
	t.repartitionDualWriteActive.Store(false)
	t.repartitionTranslation = nil
	t.mu.Unlock()
	t.maintenanceMu.Unlock()

	for _, s := range oldshards {
		GlobalCache.Remove(s)
		s.RemoveFromDisk()
	}

	// Register new PShards with CacheManager
	if t.PersistencyMode != Memory && !strings.HasPrefix(t.Name, ".") {
		for _, s := range newshards {
			atomic.StoreUint64(&s.lastAccessed, uint64(time.Now().UnixNano()))
			GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeShard, shardCleanup, shardLastUsed, nil)
		}
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
	for idx := uint32(0); idx < s.main_count; idx++ {
		if s.deletions.Get(uint(idx)) {
			continue
		}
		for i, cs := range maincols {
			values[i] = cs.GetValue(idx)
		}
		shardnum := computeShardIndex(schema, values)
		oldlist, _ := result[shardnum]
		result[shardnum] = append(oldlist, idx)
	}

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
