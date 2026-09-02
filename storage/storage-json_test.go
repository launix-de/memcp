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
	"bytes"
	"reflect"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func buildTestColumn(storage ColumnStorage, values []scm.Scmer) ColumnStorage {
	storage.prepare()
	for i, value := range values {
		storage.scan(uint32(i), value)
	}
	storage.init(uint32(len(values)))
	for i, value := range values {
		storage.build(uint32(i), value)
	}
	storage.finish()
	return storage
}

func roundTripTestColumn(t *testing.T, source ColumnStorage) ColumnStorage {
	t.Helper()
	var encoded bytes.Buffer
	source.Serialize(&encoded)
	magic, err := encoded.ReadByte()
	if err != nil {
		t.Fatal(err)
	}
	typ, ok := storages[magic]
	if !ok {
		t.Fatalf("unknown storage magic byte %d", magic)
	}
	target := reflect.New(typ).Interface().(ColumnStorage)
	target.Deserialize(&encoded)
	return target
}

func TestJSONBackedStorageRoundTripPreservesIntegerTags(t *testing.T) {
	const timestamp int64 = 1788172496
	tests := []struct {
		name   string
		column ColumnStorage
		values []scm.Scmer
		index  uint32
	}{
		{"scmer", new(StorageSCMER), []scm.Scmer{scm.NewInt(timestamp)}, 0},
		{"sparse", new(StorageSparse), []scm.Scmer{scm.NewNil(), scm.NewInt(timestamp)}, 1},
		{"enum", new(StorageEnum), []scm.Scmer{scm.NewInt(timestamp), scm.NewInt(7), scm.NewInt(timestamp)}, 0},
		{"const", new(StorageConst), []scm.Scmer{scm.NewInt(timestamp), scm.NewInt(timestamp)}, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			column := roundTripTestColumn(t, buildTestColumn(test.column, test.values))
			got := column.GetValue(test.index)
			if !got.IsInt() || got.Int() != timestamp {
				t.Fatalf("round trip returned %v (tag %d), want native integer %d", got, got.GetTag(), timestamp)
			}
		})
	}
}
