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
	"fmt"
	"sort"
	"testing"
)

const recSetBoundaryBenchmarkRows = 800_000

type recSetBoundaryBenchmarkFixture struct {
	forward StorageInt
	inverse StorageInt
}

func newRecSetBoundaryBenchmarkFixture() *recSetBoundaryBenchmarkFixture {
	const rows = uint32(recSetBoundaryBenchmarkRows)
	fixture := new(recSetBoundaryBenchmarkFixture)
	fixture.forward.initValuesUInt32(rows, 0, rows-1)
	fixture.inverse.initValuesUInt32(rows, 0, rows-1)
	// Multiplication by an odd number is a permutation modulo this power-of-two
	// multiple and deliberately removes correlation between RecID and index
	// position while retaining a reproducible benchmark.
	const multiplier = uint32(400_001)
	for position := uint32(0); position < rows; position++ {
		recid := position * multiplier % rows
		fixture.forward.buildValueUInt32(position, recid)
		fixture.inverse.buildValueUInt32(recid, position)
	}
	return fixture
}

func benchmarkRecSetIDs(fixture *recSetBoundaryBenchmarkFixture, rows int, late bool) []uint32 {
	ids := make([]uint32, rows)
	span := recSetBoundaryBenchmarkRows
	for i := range ids {
		position := (i + 1) * span / (rows + 1)
		if late {
			position = span - rows + i
		}
		ids[i] = uint32(int64(fixture.forward.GetValueUInt(uint32(position))) + fixture.forward.offset)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func benchmarkIndexMembership(fixture *recSetBoundaryBenchmarkFixture, part *recSetShard, span int, limit int) uint32 {
	var decoded [1024]uint32
	found := 0
	var result uint32
	for base := 0; base < span && found < limit; base += len(decoded) {
		count := span - base
		if count > len(decoded) {
			count = len(decoded)
		}
		fixture.forward.GetValuesUInt32Range(uint32(base), uint32(count), decoded[:count], 1)
		for _, recid := range decoded[:count] {
			if part.contains(recid) {
				result = recid
				found++
				if found == limit {
					break
				}
			}
		}
	}
	return result
}

func benchmarkRecSetPositions(fixture *recSetBoundaryBenchmarkFixture, ids []uint32, span int, limit int) uint32 {
	positions := make([]uint32, len(ids))
	fixture.inverse.GetValuesUInt32Multi(ids, positions, 1)
	sort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
	found := 0
	var result uint32
	var recid [1]uint32
	for _, position := range positions {
		if int(position) >= span {
			break
		}
		fixture.forward.GetValuesUInt32Range(position, 1, recid[:], 1)
		result = recid[0]
		found++
		if found == limit {
			break
		}
	}
	return result
}

func benchmarkRecSetDirect(part *recSetShard) uint32 {
	var buffer [1024]uint32
	count := 0
	var result uint32
	flush := func() {
		for _, recid := range buffer[:count] {
			result = recid
		}
		count = 0
	}
	part.forEachID(func(recid uint32) bool {
		buffer[count] = recid
		count++
		if count == len(buffer) {
			flush()
		}
		return true
	})
	flush()
	return result
}

func benchmarkBuildInverse(forward *StorageInt) StorageInt {
	count := uint32(forward.count)
	var inverse StorageInt
	inverse.initValuesUInt32(count, 0, count-1)
	var recids [1024]uint32
	for base := uint32(0); base < count; {
		chunkCount := count - base
		if chunkCount > uint32(len(recids)) {
			chunkCount = uint32(len(recids))
		}
		chunk := recids[:chunkCount]
		forward.GetValuesUInt32Range(base, chunkCount, chunk, 1)
		for offset, recid := range chunk {
			inverse.buildValueUInt32(recid, base+uint32(offset))
		}
		base += chunkCount
	}
	return inverse
}

var benchmarkRecSetBoundaryResult uint32

// BenchmarkRecSetBoundaryCrossover is the calibration source for
// unorderedRecSetDominates and orderedRecSetDominates. It varies the exact
// basis-index min/max span independently from RecSet cardinality and includes
// both uniform and adversarial late-hit distributions.
func BenchmarkRecSetBoundaryCrossover(b *testing.B) {
	fixture := newRecSetBoundaryBenchmarkFixture()
	for _, span := range []int{10_000, 100_000, recSetBoundaryBenchmarkRows} {
		for _, recsetRows := range []int{16, 64, 256, 1024, 4096, 16384, 100000, 400000, 720000} {
			for _, late := range []bool{false, true} {
				ids := benchmarkRecSetIDs(fixture, recsetRows, late)
				part := newRecSetShardFromSortedIDs(nil, recSetBoundaryBenchmarkRows, ids)
				name := fmt.Sprintf("span_%06d/rows_%05d/late_%t", span, recsetRows, late)
				b.Run(name+"/index_membership", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						benchmarkRecSetBoundaryResult = benchmarkIndexMembership(fixture, &part, span, 72)
					}
				})
				b.Run(name+"/recset_positions", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						benchmarkRecSetBoundaryResult = benchmarkRecSetPositions(fixture, ids, span, 72)
					}
				})
				b.Run(name+"/index_membership_unbounded", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						benchmarkRecSetBoundaryResult = benchmarkIndexMembership(fixture, &part, span, recsetRows+1)
					}
				})
				b.Run(name+"/recset_direct", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						benchmarkRecSetBoundaryResult = benchmarkRecSetDirect(&part)
					}
				})
			}
		}
	}
}

func BenchmarkRecSetBoundaryBuildInverse(b *testing.B) {
	fixture := newRecSetBoundaryBenchmarkFixture()
	b.ReportAllocs()
	for range b.N {
		inverse := benchmarkBuildInverse(&fixture.forward)
		benchmarkRecSetBoundaryResult = uint32(inverse.count)
	}
}
