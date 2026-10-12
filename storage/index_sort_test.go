// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import (
	"math/bits"
	"testing"
)

func TestIndexRowSortPreservesPermutationAndBoundsRunWork(t *testing.T) {
	const count = 22000
	for _, shape := range []string{"ascending", "descending", "equal", "two_blocks", "interleaved", "shuffled"} {
		t.Run(shape, func(t *testing.T) {
			rows := make([]uint32, count)
			for i := range rows {
				rows[i] = uint32(i)
				if shape == "shuffled" {
					rows[i] = uint32((i * 7919) % count)
				}
			}
			comparisons := 0
			less := func(a, b uint32) bool {
				comparisons++
				switch shape {
				case "ascending":
					return a < b
				case "descending":
					return a > b
				case "equal":
					return false
				}
				firstA, firstB := a%11 == 0, b%11 == 0
				if shape == "two_blocks" {
					firstA, firstB = a < 2000, b < 2000
				}
				if firstA != firstB {
					return firstA
				}
				return a > b
			}
			sortIndexRows(rows, less)
			// Bound comparison work rather than a machine-dependent wall time.
			// Repeatedly merging a growing prefix exceeds this N log N budget.
			if comparisons > 8*count*bits.Len(count) {
				t.Fatalf("sort used %d comparisons for %d rows", comparisons, count)
			}
			seen := make([]bool, count)
			for i, row := range rows {
				if row >= count || seen[row] {
					t.Fatalf("row permutation contains invalid/duplicate row %d", row)
				}
				seen[row] = true
				if i > 0 && less(row, rows[i-1]) {
					t.Fatalf("rows %d and %d are out of order", rows[i-1], row)
				}
			}
		})
	}
}
