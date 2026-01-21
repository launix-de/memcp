/*
Copyright (C) 2024  Carl-Philip Hänsch

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
import "sort"
import "sync"
import "time"
import "runtime"
import "strings"
import "github.com/jtolds/gls"
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

func (t *table) iterateShards(boundaries []columnboundaries, callback_old func(*storageShard)) {
	callback := callback_old
	if scm.Trace != nil {
		// hook on tracing
		callback = func(s *storageShard) {
			scm.Trace.Duration(fmt.Sprintf("%p", s), "shard", func() {
				callback_old(s)
			})
		}
	}
	shards := t.Shards
	var done sync.WaitGroup
	if shards != nil {
		// throttle by CPU cores to avoid massive goroutine fan-out
		workers := runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
		if len(shards) <= workers {
			done.Add(len(shards))
			for _, s := range shards {
				gls.Go(func(s *storageShard) func() {
					return func() {
						if s == nil {
							fmt.Println("Warning: a shard is missing")
							return
						}
						release := s.GetRead()
						callback(s)
						release()
						done.Done()
					}
				}(s))
			}
		} else {
			jobs := make(chan *storageShard, workers)
			done.Add(len(shards))
			for i := 0; i < workers; i++ {
				gls.Go(func() func() {
					return func() {
						for s := range jobs {
							if s == nil {
								fmt.Println("Warning: a shard is missing")
								done.Done()
								continue
							}
							release := s.GetRead()
							callback(s)
							release()
							done.Done()
						}
					}
				}())
			}
			for _, s := range shards {
				jobs <- s
			}
			close(jobs)
		}
	} else {
		iterateShardIndex(t.PDimensions, boundaries, t.PShards, func(s *storageShard) {
			release := s.GetRead()
			callback(s)
			release()
		}, &done, false)
	}
	done.Wait()
}

// iterate over all shards parallely
func iterateShardIndex(schema []shardDimension, boundaries []columnboundaries, shards []*storageShard, callback func(*storageShard), done *sync.WaitGroup, parallel bool) {
	if len(schema) == 0 {
		if len(shards) == 1 && !parallel {
			// execute without go
			release := shards[0].GetRead()
			callback(shards[0])
			release()
		} else {
			// throttle high fan-out to avoid spawning excessive goroutines
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			if len(shards) <= workers {
				done.Add(len(shards))
				for _, s := range shards {
					gls.Go(func(s *storageShard) func() {
						return func() {
							if s == nil {
								fmt.Println("Warning: a shard is missing")
								return
							}
							release := s.GetRead()
							callback(s)
							release()
							done.Done()
						}
					}(s))
				}
			} else {
				// worker pool
				jobs := make(chan *storageShard, workers)
				done.Add(len(shards))
				for i := 0; i < workers; i++ {
					gls.Go(func() func() {
						return func() {
							for s := range jobs {
								if s == nil {
									fmt.Println("Warning: a shard is missing")
									done.Done()
									continue
								}
								release := s.GetRead()
								callback(s)
								release()
								done.Done()
							}
						}
					}())
				}
				for _, s := range shards {
					jobs <- s
				}
				close(jobs)
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
				// recurse over range
				iterateShardIndex(schema[1:], boundaries, shards[i*blockdim:(i+1)*blockdim], callback, done, parallel || (min != max))
			}
			return // finish (don't run into next boundary, don't run into the all-loop)
		}
	}

	// else: no boundaries: iterate all
	for i := 0; i < len(shards); i += blockdim {
		iterateShardIndex(schema[1:], boundaries, shards[i:i+blockdim], callback, done, parallel || len(shards) >= blockdim)
	}
}

func (t *table) NewShardDimension(col string, n int) (result shardDimension) {
	result.Column = col
	if n < 1 {
		return // empty dimension
	}
	result.Pivots = make([]scm.Scmer, 0, n-1)

	// pivots are extracted from sampling
	pivotSamples := make([]scm.Scmer, 0, 2*(len(t.Shards)+len(t.PShards)))

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
		panic("partition column does not exist: `" + t.Schema + "." + t.Name + "`.`" + col + "`")
	}

	shardlist := t.Shards
	if shardlist == nil {
		shardlist = t.PShards
	}
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
		for i := uint(50); i < mc; i += 101 {
			pivotSamples = append(pivotSamples, stor.GetValue(i))
		}
	}
	if len(pivotSamples) == 0 {
		result.NumPartitions = 1
		return
	}

	// sort samplelist
	sort.Slice(pivotSamples, func(i, j int) bool {
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
	sort.Slice(shardCandidates, func(i, j int) bool { // Less
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

// this runs inside a t.mu.Lock()
func (t *table) repartition(shardCandidates []shardDimension) {
	// rebuild sharding schema
	// TODO: check if we can own the table (don't repartition small tables whose shards we don't own)
	// TODO: in case of big tables: search for other nodes who own shards and do the work together, otherwise we run out of ram
	totalShards := 1
	for _, sc := range shardCandidates {
		totalShards *= sc.NumPartitions
	}

	fmt.Println("repartitioning", t.Name, "by", shardCandidates)
	start := time.Now() // time measurement

	oldshards := t.Shards
	if oldshards == nil {
		oldshards = t.PShards
	}

	// Before repartitioning, make sure all shards are loaded and the
	// columns required for partitioning are present in memory. Doing this
	// outside of shard locks avoids lock upgrades inside partition() when
	// reading main storages. Also eagerly load all table columns so that
	// later ColumnReader() calls do not attempt to acquire a write lock
	// while repartition holds long-lived RLocks on the shards.
	for _, s := range oldshards {
		if s == nil {
			continue
		}
		// Load shard state first (may acquire locks internally)
		s.ensureLoaded()
		// Now lock exclusively and perform critical loads without internal locking
		s.mu.Lock()
		for _, sd := range shardCandidates {
			s.ensureColumnLoaded(sd.Column, true)
		}
		for _, col := range t.Columns {
			s.ensureColumnLoaded(col.Name, true)
		}
		s.ensureMainCount(true)
		s.mu.Unlock()
	}

	// collect all dataset IDs (this is done sequentially and takes ~4s for 8G of data)
	datasetids := make([][][]uint, totalShards) // newshard, oldshard, item
	total_count := uint64(0)
	for si, s := range oldshards {
		s.mu.RLock()
		total_count += uint64(s.Count())
		for idx, items := range s.partition(shardCandidates) {
			if datasetids[idx] == nil {
				datasetids[idx] = make([][]uint, len(oldshards))
			}
			datasetids[idx][si] = items
		}
	}
	// TODO: On large tables, perform distributed repartitioning across nodes
	//       instead of local single-node rebuild to reduce downtime and memory pressure.
	// put values into shards
	fmt.Println("moving data from", t.Name, len(oldshards), "into", totalShards, "shards")
	newshards := make([]*storageShard, totalShards)
	var done sync.WaitGroup
	done.Add(totalShards)
	progress := make(chan int, runtime.NumCPU()/2) // don't go all at once, we don't have enough RAM
	for i := 0; i < runtime.NumCPU()/2; i++ {
		go func() { // threadpool with half of the cores
			for si := range progress {
				// create a new shard and put all data in
				s := NewShard(t)
				// directly build main storage from list, no delta
				for _, items := range datasetids[si] {
					s.main_count += uint(len(items))
				}
				// allocate only once
				values := make([]scm.Scmer, s.main_count)
				for _, col := range t.Columns {
					// build the cache-optimized scmer list
					var i uint // index and amount of items
					for s2id, items := range datasetids[si] {
						reader := oldshards[s2id].ColumnReader(col.Name)
						for _, item := range items {
							values[i] = reader(item) // call decompression only once; this uses more RAM at once but is way faster
							i++
						}
					}

					// compress into a new column
					var newcol ColumnStorage = new(StorageSCMER)
					for {
						newcol.prepare()
						for i, v := range values {
							newcol.scan(uint(i), v)
						}
						newcol2 := newcol.proposeCompression(i)
						if newcol2 == nil {
							break // we found the optimal storage format
						} else {
							// redo scan phase with compression
							//fmt.Printf("Compression with %T\n", newcol2)
							newcol = newcol2
						}
					}
					newcol.init(s.main_count) // allocate memory
					for i, v := range values {
						newcol.build(uint(i), v)
					}
					newcol.finish()
					s.columns[col.Name] = newcol

					// write to disc (only if required)
					if s.t.PersistencyMode != Memory {
						f := persistenceForSchema(s.t.Schema).WriteColumn(s.uuid.String(), col.Name)
						newcol.Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
						f.Close()
					}
				}
				newshards[si] = s

				if s.t.PersistencyMode == Safe || s.t.PersistencyMode == Logged {
					// open a logfile
					s.logfile = persistenceForSchema(s.t.Schema).OpenLog(s.uuid.String())
				}
				done.Done()
			}
		}()
	}
	for si, _ := range newshards {
		progress <- si // inserting into this chan blocks when the queue is full, so we can use it as a progress bar
		fmt.Println("rebuild", t.Name, si+1, "/", len(newshards))
	}
	done.Wait()

	for _, s := range oldshards {
		// TODO: set next such that Inserts will be redirected
		s.mu.RUnlock()
	}

	// verify transformation result
	total_count2 := uint64(0)
	for _, s := range newshards {
		total_count2 += uint64(s.Count())
	}
	if total_count != total_count2 {
		fmt.Println("error: aborted partitioning schema for ", t.Name, "after", time.Since(start), " because of inconsistency: before", total_count, "items, after", total_count2)
		return
	}

	// now take over the new sharding schema
	// Publish the new partitioned shard list directly to PShards and
	// keep Shards nil so readers consistently prefer PShards.
	t.PShards = newshards
	t.PDimensions = shardCandidates

	t.Shards = nil // partitioned layout is live
	fmt.Println("activated new partitioning schema for ", t.Name, "after", time.Since(start))

	saveTableMetadata(t)

	for _, s := range oldshards {
		// discard from disk
		s.RemoveFromDisk()
	}
}

func (s *storageShard) partition(schema []shardDimension) (result map[int][]uint) {
	// assigns each dataset into a target shard
	result = make(map[int][]uint)

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
	for idx := uint(0); idx < s.main_count; idx++ {
		if s.deletions.Get(idx) {
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
		if s.deletions.Get(s.main_count + uint(idx)) {
			continue
		}
		for i, cs := range deltacols {
			values[i] = dataset[cs]
		}
		shardnum := computeShardIndex(schema, values)
		oldlist, _ := result[shardnum]
		result[shardnum] = append(oldlist, s.main_count+uint(idx))
	}

	return
}
