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

import "sort"
import "github.com/launix-de/memcp/scm"

type shardDimension struct {
	Column string
	NumPartitions int
	Pivots []scm.Scmer
}

func (t *table) NewShardDimension(col string, n int) (result shardDimension) {
	result.Column = col
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
	result.NumPartitions = len(result.Pivots) - 1

	return
}
