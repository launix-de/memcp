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
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func TestExactScalarStorageKeepsHistoricalNativeValues(t *testing.T) {
	date := scm.NewDateNanoWithZone(1788172496, 123456789, 7)
	values := []scm.Scmer{date, scm.NewNil(), scm.NewString("literal"),
		scm.NewBSONRaw(0x10, []byte{42, 0, 0, 0})}
	column := roundTripTestColumn(t, buildTestColumn(new(StorageSCMER), values))
	for i, want := range values {
		got := column.GetValue(uint32(i))
		if got.GetTag() != want.GetTag() || !scm.Equal(got, want) {
			t.Fatalf("scalar %d = %s (tag %d), want %s (tag %d)", i, got, got.GetTag(), want, want.GetTag())
		}
	}
	got := column.GetValue(0)
	if got.DateNanosecond() != 123456789 || got.DateZone() != 7 {
		t.Fatal("generic scalar storage lost native date parts")
	}
	calendar := roundTripTestColumn(t, buildTestColumn(new(StorageDate), []scm.Scmer{date, scm.NewNil(), scm.NewDate(0)}))
	if got := calendar.GetValue(0); !got.IsDate() || got.DateNanosecond() != 123456789 || got.DateZone() != 7 || calendar.GetValue(1).IsNil() == false {
		t.Fatal("calendar stream lost fractional/zone/null identity")
	}
}

func TestLegacyScalarMagicRemainsReadable(t *testing.T) {
	var frame bytes.Buffer
	binary.Write(&frame, binary.LittleEndian, uint64(3))
	frame.WriteString("7\n\"legacy\"\n{\"$memcp.scmer\":\"bson-v1\",\"type\":16,\"payload\":\"KgAAAA\"}\n")
	legacy := new(storageSCMERLegacy)
	if got := legacy.Deserialize(&frame); got != 3 || legacy.GetValue(0).Int() != 7 || legacy.GetValue(1).String() != "legacy" {
		t.Fatal("legacy magic-1 scalar column did not restore its full values")
	}
	kind, payload := legacy.GetValue(2).BSONRaw()
	if !legacy.GetValue(2).IsBSON() || kind != 0x10 || !bytes.Equal(payload, []byte{42, 0, 0, 0}) {
		t.Fatal("legacy magic-1 BSON envelope changed its native value")
	}
	var next bytes.Buffer
	legacy.Serialize(&next)
	if magic, _ := next.ReadByte(); magic != 3 {
		t.Fatalf("re-saving legacy values emitted magic %d, want current magic 3", magic)
	}
}

func TestComputedProxyV3NestedScalarFraming(t *testing.T) {
	date := scm.NewDateNanoWithZone(1788172496, 987654321, 2)
	for _, compressed := range []bool{false, true} {
		proxy := &StorageComputeProxy{count: 1, delta: make(map[uint32]scm.Scmer), compressed: compressed,
			computor: scm.Eval(scm.Read("fixture", "(lambda () 17)", true), &scm.Globalenv)}
		if compressed {
			proxy.main = buildTestColumn(new(StorageSCMER), []scm.Scmer{date})
		} else {
			proxy.setDeltaValueLocked(0, date)
		}
		proxy.validMask.AtomicSet(0, true)
		var frame bytes.Buffer
		proxy.Serialize(&frame)
		frame.WriteByte(0xa5)
		if magic, _ := frame.ReadByte(); magic != 50 {
			t.Fatal("unexpected computed-column magic")
		}
		var restored StorageComputeProxy
		if restored.Deserialize(&frame) != 1 || !restored.validMask.AtomicGet(0) || restored.compressed != compressed {
			t.Fatal("computed-column header/mask changed")
		}
		value := restored.delta[0]
		if compressed {
			value = restored.main.GetValue(0)
		}
		if !value.IsDate() || !scm.Equal(value, date) || value.DateZone() != 2 || value.DateNanosecond() != 987654321 {
			t.Fatal("computed-column cached value lost exact calendar parts")
		}
		if scm.Apply(restored.computor).Int() != 17 {
			t.Fatal("computed-column recipe did not survive reload")
		}
		if tail, _ := frame.ReadByte(); tail != 0xa5 || frame.Len() != 0 {
			t.Fatal("nested scalar reader consumed its enclosing frame")
		}
	}
}

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
