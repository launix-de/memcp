//go:build goexperiment.jit && amd64

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
	"testing"

	"github.com/launix-de/memcp/scm"
)

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

func TestStorageIntJITMultiReaderAcrossLargeBatch(t *testing.T) {
	const count = 60000
	values := make([]scm.Scmer, count)
	recids := make([]uint32, count)
	for index := range values {
		values[index] = scm.NewInt(int64(index*7919%1000 + 1))
		recids[index] = uint32(index)
	}
	column := buildExactStorage(t, &StorageInt{}, values)
	rangeValues := make([]scm.Scmer, count)
	column.GetJITGetValueRange()(0, count, rangeValues, 1)
	for index := range values {
		if !scm.Equal(rangeValues[index], values[index]) {
			t.Fatalf("GetValueRange[%d] = %v, want %v", index, rangeValues[index], values[index])
		}
	}
	got := make([]scm.Scmer, count)
	column.GetJITGetValueMulti()(recids, got, 1)
	for index := range values {
		if !scm.Equal(got[index], values[index]) {
			t.Fatalf("GetValueMulti[%d] = %v, want %v", index, got[index], values[index])
		}
	}
}
