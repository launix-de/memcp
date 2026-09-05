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

type directJITStorageReader struct {
	getValue      scm.JITStorageGetValueFunc
	getValueRange scm.JITStorageGetValueRangeFunc
	getValueMulti scm.JITStorageGetValueMultiFunc
}

func (reader directJITStorageReader) GetValue(recid uint32) scm.Scmer {
	return reader.getValue(recid)
}

func (reader directJITStorageReader) GetValueRange(recid, count uint32, target []scm.Scmer, stride int) {
	reader.getValueRange(recid, count, target, stride)
}

func (reader directJITStorageReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	reader.getValueMulti(recids, target, stride)
}

type exactJITStorage interface {
	prepare()
	scan(uint32, scm.Scmer)
	init(uint32)
	build(uint32, scm.Scmer)
	finish()
	GetValue(uint32) scm.Scmer
	GetJITGetValue() scm.JITStorageGetValueFunc
	GetJITGetValueRange() scm.JITStorageGetValueRangeFunc
	GetJITGetValueMulti() scm.JITStorageGetValueMultiFunc
}

func buildExactStorage(t *testing.T, storage exactJITStorage, values []scm.Scmer) exactJITStorage {
	t.Helper()
	storage.prepare()
	for index, value := range values {
		storage.scan(uint32(index), value)
	}
	storage.init(uint32(len(values)))
	for index, value := range values {
		storage.build(uint32(index), value)
	}
	storage.finish()
	return storage
}

func exactJITStorageFixtures() []struct {
	name   string
	create func() exactJITStorage
	value  func(int) scm.Scmer
} {
	return []struct {
		name   string
		create func() exactJITStorage
		value  func(int) scm.Scmer
	}{
		{"StorageSCMER", func() exactJITStorage { return &StorageSCMER{} }, func(index int) scm.Scmer {
			switch index % 4 {
			case 0:
				return scm.NewNil()
			case 1:
				return scm.NewInt(int64(index) - 40)
			case 2:
				return scm.NewFloat(float64(index) / 7)
			default:
				return scm.NewString("mixed_" + scm.String(scm.NewInt(int64(index))))
			}
		}},
		{"StorageInt", func() exactJITStorage { return &StorageInt{} }, func(index int) scm.Scmer {
			if index%11 == 0 {
				return scm.NewNil()
			}
			return scm.NewInt(int64(index*7919%65521) - 32760)
		}},
		{"StorageFloat", func() exactJITStorage { return &StorageFloat{} }, func(index int) scm.Scmer {
			if index%13 == 0 {
				return scm.NewNil()
			}
			return scm.NewFloat(float64(index*index-3000) / 19)
		}},
		{"StorageString", func() exactJITStorage { return &StorageString{} }, func(index int) scm.Scmer {
			if index%17 == 0 {
				return scm.NewNil()
			}
			return scm.NewString("value_" + scm.String(scm.NewInt(int64(index%23))))
		}},
		{"StorageSeq", func() exactJITStorage { return &StorageSeq{} }, func(index int) scm.Scmer {
			if index >= 40 && index < 47 {
				return scm.NewNil()
			}
			return scm.NewInt(int64((index/9)*100 + index%9*3))
		}},
		{"StorageSparse", func() exactJITStorage { return &StorageSparse{} }, func(index int) scm.Scmer {
			if index%19 != 0 {
				return scm.NewNil()
			}
			return scm.NewString("sparse_" + scm.String(scm.NewInt(int64(index))))
		}},
		{"StorageDecimal", func() exactJITStorage { return &StorageDecimal{scaleExp: -2} }, func(index int) scm.Scmer {
			if index%29 == 0 {
				return scm.NewNil()
			}
			return scm.NewFloat(float64(index-50) / 100)
		}},
		{"StoragePrefix", func() exactJITStorage {
			return &StoragePrefix{prefixdictionary: []string{"", "https://example.test/common/path/"}}
		}, func(index int) scm.Scmer {
			if index%31 == 0 {
				return scm.NewNil()
			}
			return scm.NewString("https://example.test/common/path/" + scm.String(scm.NewInt(int64(index))))
		}},
		{"StorageEnum", func() exactJITStorage { return &StorageEnum{} }, func(index int) scm.Scmer {
			values := [...]scm.Scmer{scm.NewNil(), scm.NewString("a"), scm.NewString("beta"), scm.NewString("gamma")}
			return values[index%len(values)]
		}},
		{"StorageConst", func() exactJITStorage { return &StorageConst{} }, func(int) scm.Scmer { return scm.NewString("constant") }},
		{"OverlayBlob", func() exactJITStorage { return &OverlayBlob{Base: &StorageSCMER{}} }, func(index int) scm.Scmer {
			if index%7 == 0 {
				return scm.NewNil()
			}
			return scm.NewString("blob_" + scm.String(scm.NewInt(int64(index))))
		}},
	}
}

func TestFinishedStorageJITReadersMatchAllThreeReadModes(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires the JIT experiment")
	}
	const count = 127
	for _, fixture := range exactJITStorageFixtures() {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			values := make([]scm.Scmer, count)
			for index := range values {
				values[index] = fixture.value(index)
			}
			column := buildExactStorage(t, fixture.create(), values)
			reader := directJITStorageReader{
				getValue:      column.GetJITGetValue(),
				getValueRange: column.GetJITGetValueRange(),
				getValueMulti: column.GetJITGetValueMulti(),
			}
			if reader.getValue == nil || reader.getValueRange == nil || reader.getValueMulti == nil {
				t.Fatalf("finished storage JIT readers: scalar=%t range=%t multi=%t", reader.getValue != nil, reader.getValueRange != nil, reader.getValueMulti != nil)
			}

			t.Run("scalar", func(t *testing.T) {
				for recid := range values {
					want := column.GetValue(uint32(recid))
					if got := reader.GetValue(uint32(recid)); !scm.Equal(got, want) {
						t.Fatalf("GetValue(%d) = %v, want %v", recid, got, want)
					}
				}
			})
			t.Run("range", func(t *testing.T) {
				for _, test := range []struct{ start, length uint32 }{{0, 0}, {0, 1}, {0, count}, {31, 65}, {count - 1, 1}} {
					const stride = 3
					got := make([]scm.Scmer, int(test.length)*stride)
					for index := range got {
						got[index] = scm.NewString("untouched")
					}
					reader.GetValueRange(test.start, test.length, got, stride)
					for index := uint32(0); index < test.length; index++ {
						want := column.GetValue(test.start + index)
						if !scm.Equal(got[int(index)*stride], want) {
							t.Fatalf("GetValueRange(%d,%d)[%d] = %v, want %v", test.start, test.length, index, got[int(index)*stride], want)
						}
					}
				}
			})
			t.Run("multi", func(t *testing.T) {
				for _, recids := range [][]uint32{nil, {0}, {0, 3, 3, 64, 126}, {126, 2, 90, 1, 77, 0}} {
					const stride = 2
					got := make([]scm.Scmer, len(recids)*stride)
					reader.GetValueMulti(recids, got, stride)
					for index, recid := range recids {
						want := column.GetValue(recid)
						if !scm.Equal(got[index*stride], want) {
							t.Fatalf("GetValueMulti[%d](%d) = %v, want %v", index, recid, got[index*stride], want)
						}
					}
				}
			})
		})
	}
}

func TestStorageComputeProxyJITReadersMatchAllThreeReadModes(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires the JIT experiment")
	}
	const count = 127
	values := make([]scm.Scmer, count)
	for index := range values {
		if index%11 == 0 {
			values[index] = scm.NewNil()
		} else {
			values[index] = scm.NewString("computed_" + scm.String(scm.NewInt(int64(index))))
		}
	}
	main := buildExactStorage(t, &StorageSCMER{}, values)
	proxy := &StorageComputeProxy{
		main:       main.(ColumnStorage),
		delta:      make(map[uint32]scm.Scmer),
		compressed: true,
		count:      count,
	}
	// A compute proxy is never a rebuild target, so its public finish method
	// deliberately panics. Its immutable, already-compressed read state can
	// still install the same three specialized entry points used by consumers.
	proxy.storageJITFunctions.finish(proxy)
	reader := directJITStorageReader{
		getValue:      proxy.GetJITGetValue(),
		getValueRange: proxy.GetJITGetValueRange(),
		getValueMulti: proxy.GetJITGetValueMulti(),
	}
	if reader.getValue == nil || reader.getValueRange == nil || reader.getValueMulti == nil {
		t.Fatalf("compute proxy JIT readers: scalar=%t range=%t multi=%t", reader.getValue != nil, reader.getValueRange != nil, reader.getValueMulti != nil)
	}
	for recid := range values {
		if got, want := reader.GetValue(uint32(recid)), proxy.GetValue(uint32(recid)); !scm.Equal(got, want) {
			t.Fatalf("GetValue(%d) = %v, want %v", recid, got, want)
		}
	}
	gotRange := make([]scm.Scmer, 65*3)
	reader.GetValueRange(31, 65, gotRange, 3)
	for index := 0; index < 65; index++ {
		if got, want := gotRange[index*3], proxy.GetValue(uint32(31+index)); !scm.Equal(got, want) {
			t.Fatalf("GetValueRange[%d] = %v, want %v", index, got, want)
		}
	}
	recids := []uint32{126, 2, 90, 2, 77, 0}
	gotMulti := make([]scm.Scmer, len(recids)*2)
	reader.GetValueMulti(recids, gotMulti, 2)
	for index, recid := range recids {
		if got, want := gotMulti[index*2], proxy.GetValue(recid); !scm.Equal(got, want) {
			t.Fatalf("GetValueMulti[%d](%d) = %v, want %v", index, recid, got, want)
		}
	}
}
