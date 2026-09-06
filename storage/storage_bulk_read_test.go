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
	"math/rand"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// bulkReadFixtures drives buildViaCompression with generators chosen to hit
// each concrete ColumnStorage format's proposeCompression path, so the
// generic checks below exercise every format's GetValueRange/GetValueMulti
// implementation, not just one.
func bulkReadFixtures() []struct {
	name string
	n    int
	gen  func(int) scm.Scmer
} {
	return []struct {
		name string
		n    int
		gen  func(int) scm.Scmer
	}{
		{"Const", 500, func(i int) scm.Scmer { return scm.NewInt(42) }},
		{"IntRandom", 500, func(i int) scm.Scmer { return scm.NewInt(int64((i*2654435761)%9973) - 5000) }},
		{"IntWithNulls", 500, func(i int) scm.Scmer {
			if i%7 == 0 {
				return scm.NewNil()
			}
			return scm.NewInt(int64(i % 100))
		}},
		{"Seq", 500, func(i int) scm.Scmer { return scm.NewInt(int64(i) * 3) }},
		{"SeqWithNullRuns", 500, func(i int) scm.Scmer {
			if (i/17)%5 == 0 {
				return scm.NewNil()
			}
			return scm.NewInt(int64(i))
		}},
		{"Decimal", 500, func(i int) scm.Scmer { return scm.NewFloat(float64(i%1000) / 100.0) }},
		{"Sparse", 500, func(i int) scm.Scmer {
			if i%23 != 0 {
				return scm.NewNil()
			}
			return scm.NewString("v")
		}},
		{"Enum", 500, func(i int) scm.Scmer {
			switch i % 5 {
			case 0:
				return scm.NewString("alpha")
			case 1:
				return scm.NewString("beta")
			case 2:
				return scm.NewString("gamma")
			case 3:
				return scm.NewNil()
			default:
				return scm.NewString("delta")
			}
		}},
		{"StringDict", 500, func(i int) scm.Scmer { return scm.NewString("str_" + scm.String(scm.NewInt(int64(i%37)))) }},
		{"StringPrefix", 500, func(i int) scm.Scmer {
			return scm.NewString("https://example.com/path/" + scm.String(scm.NewInt(int64(i))))
		}},
		{"Float", 500, func(i int) scm.Scmer { return scm.NewFloat(float64(i) * 1.0000001) }},
	}
}

// bulkReadable is the minimal read surface shared by ColumnStorage and
// ColumnReader, so the checks below run identically against either.
type bulkReadable interface {
	GetValue(uint32) scm.Scmer
	GetValueMulti(recids []uint32, target []scm.Scmer, stride int)
	GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int)
}

func verifyBulkAgainstGetValue(t *testing.T, name string, col bulkReadable, n int) {
	t.Helper()

	want := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		want[i] = col.GetValue(uint32(i))
	}

	// GetValueRange over the full column, stride 1.
	got := make([]scm.Scmer, n)
	col.GetValueRange(0, uint32(n), got, 1)
	for i := 0; i < n; i++ {
		if !scm.Equal(got[i], want[i]) {
			t.Errorf("%s: GetValueRange(0,%d)[%d] = %v, want %v", name, n, i, got[i], want[i])
		}
	}

	// GetValueRange over a mid-column window.
	base := uint32(n / 4)
	count := uint32(n / 2)
	got2 := make([]scm.Scmer, count)
	col.GetValueRange(base, count, got2, 1)
	for k := uint32(0); k < count; k++ {
		if !scm.Equal(got2[k], want[base+k]) {
			t.Errorf("%s: GetValueRange(%d,%d)[%d] = %v, want %v", name, base, count, k, got2[k], want[base+k])
		}
	}

	// GetValueMulti with an ascending, gappy subset (the common index-scan shape).
	var ascending []uint32
	for i := 0; i < n; i += 3 {
		ascending = append(ascending, uint32(i))
	}
	gotAsc := make([]scm.Scmer, len(ascending))
	col.GetValueMulti(ascending, gotAsc, 1)
	for k, recid := range ascending {
		if !scm.Equal(gotAsc[k], want[recid]) {
			t.Errorf("%s: GetValueMulti(ascending)[%d] (recid %d) = %v, want %v", name, k, recid, gotAsc[k], want[recid])
		}
	}

	// GetValueMulti with a shuffled (non-monotonic) subset, including repeats.
	rng := rand.New(rand.NewSource(1))
	shuffled := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		shuffled = append(shuffled, uint32(i))
	}
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	shuffled = append(shuffled, shuffled[:n/5]...) // duplicate a prefix to exercise repeats
	gotShuf := make([]scm.Scmer, len(shuffled))
	col.GetValueMulti(shuffled, gotShuf, 1)
	for k, recid := range shuffled {
		if !scm.Equal(gotShuf[k], want[recid]) {
			t.Errorf("%s: GetValueMulti(shuffled)[%d] (recid %d) = %v, want %v", name, k, recid, gotShuf[k], want[recid])
		}
	}

	// stride>1: writes must land at target[k*stride], leaving the gaps alone.
	const stride = 3
	target := make([]scm.Scmer, len(ascending)*stride)
	for i := range target {
		target[i] = scm.NewString("sentinel")
	}
	col.GetValueMulti(ascending, target, stride)
	for k, recid := range ascending {
		if !scm.Equal(target[k*stride], want[recid]) {
			t.Errorf("%s: GetValueMulti stride=%d [%d] (recid %d) = %v, want %v", name, stride, k, recid, target[k*stride], want[recid])
		}
	}
	for i := range ascending {
		for g := 1; g < stride; g++ {
			if !scm.Equal(target[i*stride+g], scm.NewString("sentinel")) {
				t.Errorf("%s: GetValueMulti stride=%d clobbered gap slot %d", name, stride, i*stride+g)
			}
		}
	}
}

// TestBulkReadMatchesGetValue checks GetValueRange/GetValueMulti against a
// plain per-element GetValue loop across every ColumnStorage format
// proposeCompression can produce, plus the reader returned by
// GetCachedReader() for each.
func TestBulkReadMatchesGetValue(t *testing.T) {
	for _, fx := range bulkReadFixtures() {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			col := buildViaCompression(fx.n, fx.gen)
			t.Logf("%s compressed to %s", fx.name, col.String())
			verifyBulkAgainstGetValue(t, fx.name+"/storage", col, fx.n)

			reader := col.GetCachedReader()
			verifyBulkAgainstGetValue(t, fx.name+"/reader", reader, fx.n)
		})
	}
}
