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

import "os"
import "fmt"
import "sort"
import "sync"
import "time"
import "github.com/launix-de/memcp/scm"

type shardDimension struct {
	Column string
	NumPartitions int
	Pivots []scm.Scmer
}

// computes the index of a datapoint in PShards
func computeShardIndex(schema []shardDimension, values []scm.Scmer) (result int) {
	for i, sd := range schema {
		// get slice idx of this dimension
		min := 0 // greater equal min
		max := sd.NumPartitions-1 // smaller than max
		for min < max {
			pivot := (min + max) / 2
			if scm.Less(values[i], sd.Pivots[pivot]) {
				max = pivot - 1
			} else {
				min = pivot + 1
			}
		}
		result = result * sd.NumPartitions + min // accumulate
	}
	return // schema[0] has the higest stride; schema[len(schema)-1] is the least significant bit
}

func (t *table) iterateShards(boundaries []columnboundaries, callback func(*storageShard)) {
	shards := t.Shards
	var done sync.WaitGroup
	if shards != nil {
		done.Add(len(shards))
		for _, s := range shards {
			// iterateShardIndex will go
			go func(s *storageShard) {
				if s == nil {
					fmt.Println("Warning: a shard is missing")
					return
				}
				s.RunOn()
				callback(s)
				done.Done()
			}(s)
		}
	} else {
		iterateShardIndex(t.PDimensions, boundaries, t.PShards, callback, &done)
	}
	done.Wait()
}

// iterate over all shards parallely
func iterateShardIndex(schema []shardDimension, boundaries []columnboundaries, shards []*storageShard, callback func(*storageShard), done *sync.WaitGroup) {
	if len(schema) == 0 {
		done.Add(len(shards))
		for _, s := range shards {
			// iterateShardIndex will go
			go func(s *storageShard) {
				if s == nil {
					fmt.Println("Warning: a shard is missing")
					return
				}
				s.RunOn()
				callback(s)
				done.Done()
			}(s)
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
			if b.lower != nil {
				// lower bound is given -> find lowest part
				max := len(shards) / blockdim
				for min < max {
					pivot := (min + max) / 2
					if b.lowerInclusive {
						if scm.Less(b.lower, schema[0].Pivots[pivot]) {
							max = pivot - 1
						} else {
							min = pivot
						}
					} else {
						if !scm.Less(schema[0].Pivots[pivot], b.lower) {
							max = pivot - 1
						} else {
							min = pivot
						}
					}
				}
			}

			max := len(shards) / blockdim
			if b.upper != nil {
				// upper bound is given -> find highest part
				umin := min
				for umin < max {
					pivot := (umin + max) / 2
					if b.upperInclusive {
						if scm.Less(b.upper, schema[0].Pivots[pivot]) {
							max = pivot - 1
						} else {
							umin = pivot
						}
					} else {
						if !scm.Less(schema[0].Pivots[pivot], b.upper) {
							max = pivot - 1
						} else {
							umin = pivot
						}
					}
				}
			}

			for i := min; i < max; i++ {
				// recurse over range
				iterateShardIndex(schema[1:], boundaries, shards[i*blockdim:(i+1)*blockdim], callback, done)
			}
			return // finish (don't run into next boundary, don't run into the all-loop)
		}
	}

	// else: no boundaries: iterate all
	for i := 0; i < len(shards); i += blockdim {
		iterateShardIndex(schema[1:], boundaries, shards[i:i+blockdim], callback, done)
	}
}

func (t *table) NewShardDimension(col string, n int) (result shardDimension) {
	result.Column = col
	if n < 1 {
		return // empty dimension
	}
	result.Pivots = make([]scm.Scmer, 0, n-1)

	// pivots are extracted from sampling
	pivotSamples := make([]scm.Scmer, 0, 2 * (len(t.Shards) + len(t.PShards)))

	shardlist := t.Shards
	if shardlist == nil {
		shardlist = t.PShards
	}
	for _, s := range shardlist {
		// collect samples from all the shards
		if stor, ok := s.columns[col]; ok {
			// sample first element
			if s.main_count > 0 {
				pivotSamples = append(pivotSamples, stor.GetValue(0))
			}
			// sample last element
			if s.main_count > 3 {
				pivotSamples = append(pivotSamples, stor.GetValue(s.main_count - 1))
			}
			// sample some elements inbetween
			for i := uint(50); i < s.main_count; i += 101 {
				pivotSamples = append(pivotSamples, stor.GetValue(i))
			}
		}
	}
	if len(pivotSamples) == 0 {
		result.NumPartitions = 1
		return
	}

	// sort samplelist
	sort.Slice(pivotSamples, func (i, j int) bool {
		return scm.Less(pivotSamples[i], pivotSamples[j])
	})
	// extract n-1 pivots
	for i := 1; i < n; i++ {
		sample := pivotSamples[(i * len(pivotSamples)) / n]
		// only add new items
		if sample != nil && (len(result.Pivots) == 0 || scm.Less(result.Pivots[len(result.Pivots)-1], sample)) {
			result.Pivots = append(result.Pivots, sample)
		}
	}
	result.NumPartitions = len(result.Pivots) + 1

	return
}

type partitioningSet struct {
	shard *storageShard
	items map[int][]uint
}

func (t *table) proposerepartition(maincount uint) (shardCandidates []shardDimension, shouldChange bool) { // this happens inside t.mu.Lock()
	// reevaluate partitioning schema
	for _, c := range t.Columns {
		if c.PartitioningScore > 0 {
			shardCandidates = append(shardCandidates, shardDimension{c.Name, c.PartitioningScore, nil})
		}
	}
	if len(shardCandidates) == 0 {
		return
	}

	// sort for highest ranking column
	sort.Slice(shardCandidates, func (i, j int) bool { // Less
		return shardCandidates[i].NumPartitions > shardCandidates[j].NumPartitions
	})
	sf := 0.01 // scale factor
	desiredNumberOfShards := maincount / 30000 + 1 // keep some extra room
	for iter := 2; iter < 300; iter++ { // find perfect scale factor such that we get the best number of shards
		deviation := 1
		for _, sc := range shardCandidates {
			deviation *= int(float64(sc.NumPartitions) * sf)
		}
		deviation -= int(desiredNumberOfShards)
		if deviation < 0 {
			// too few shards: increase sf
			sf = sf * (1.0+1.0/float64(iter))
		} else {
			// too much shards: decrease sf
			sf = sf * (1.0-1.0/float64(iter))
		}
	}
	for i, sc := range shardCandidates {
		shardCandidates[i] = t.NewShardDimension(sc.Column, int(float64(sc.NumPartitions) * sf))
	}
	// remove empty partitions
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
		if 2 * totalShards1 > 3 * totalShards2 || 2 * totalShards2 > 3 * totalShards1 {
			shouldChange = true
		}
	}
	return // the caller will evaluate shouldChange and shardCandidates
}

func (t *table) repartition(shardCandidates []shardDimension) {
	// rebuild sharding schema
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

	// collect all dataset IDs
	datasetids := make([]map[*storageShard][]uint, totalShards)
	collector := make(chan partitioningSet, 16)
	for _, s := range oldshards {
		s.mu.RLock()
		go func (s *storageShard) {
			collector <- partitioningSet{s, s.partition(shardCandidates)}
		}(s)
	}
	// collect and resort
	for range oldshards {
		itemset := <- collector
		for idx, items := range itemset.items {
			if datasetids[idx] == nil {
				datasetids[idx] = make(map[*storageShard][]uint)
			}
			datasetids[idx][itemset.shard] = items
		}
	}
	// put values into shards
	fmt.Println("moving data from", t.Name, "into", totalShards,"shards")
	newshards := make([]*storageShard, totalShards)
	var done sync.WaitGroup
	done.Add(totalShards)
	for si, _ := range newshards {
		go func(si int) {
			// create a new shard and put all data in
			s := NewShard(t)
			// directly build main storage from list, no delta
			for _, items := range datasetids[si] {
				s.main_count += uint(len(items))
			}
			for _, col := range t.Columns {
				var newcol ColumnStorage = new(StorageSCMER)
				var i uint // index and amount of items
				for {
					newcol.prepare()
					i = 0
					for s2, items := range datasetids[si] {
						reader := s2.ColumnReader(col.Name)
						for _, item := range items {
							newcol.scan(i, reader(item))
							i++
						}
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
				newcol.init(i) // allocate memory
				i = 0
				for s2, items := range datasetids[si] {
					reader := s2.ColumnReader(col.Name)
					for _, item := range items {
						newcol.build(i, reader(item))
						i++
					}
				}
				newcol.finish()
				s.columns[col.Name] = newcol

				// write to disc (only if required)
				if s.t.PersistencyMode != Memory {
					f, err := os.Create(s.t.schema.path + s.uuid.String() + "-" + ProcessColumnName(col.Name))
					if err != nil {
						panic(err)
					}
					newcol.Serialize(f) // col takes ownership of f, so they will defer f.Close() at the right time
					f.Close()
				}
			}
			newshards[si] = s

			if s.t.PersistencyMode == Safe {
				// open a logfile
				f, err := os.OpenFile(s.t.schema.path + s.uuid.String() + ".log", os.O_RDWR|os.O_CREATE, 0750)
				if err != nil {
					panic(err)
				}
				s.logfile = f
			}
			done.Done()
		}(si)
	}
	done.Wait()

	for _, s := range oldshards {
		// TODO: set next such that Inserts will be redirected
		s.mu.RUnlock()
	}

	// now take over the new sharding schema
	if t.Shards == nil {
		t.Shards = t.PShards // move shard list over to unordered shardlist
		// warning! = on slices may not be atomic and thus dangerous
	}
	t.PShards = newshards
	t.PDimensions = shardCandidates

	t.Shards = nil // now it's live!
	fmt.Println("activated new partitioning schema for ", t.Name, "after", time.Since(start))

	t.schema.schemalock.Lock()
	t.schema.save()
	t.schema.schemalock.Unlock()

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
	for i, dataset := range s.inserts {
		if s.deletions.Get(s.main_count + uint(i)) {
			continue
		}
		for i, cs := range deltacols {
			values[i] = dataset[cs]
		}
		shardnum := computeShardIndex(schema, values)
		oldlist, _ := result[shardnum]
		result[shardnum] = append(oldlist, s.main_count + uint(i))
	}

	return
}
