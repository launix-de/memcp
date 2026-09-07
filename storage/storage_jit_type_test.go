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

func TestFinishedMainStorageJITValueTypes(t *testing.T) {
	tests := []struct {
		name   string
		values []scm.Scmer
		want   uint8
	}{
		{"int", []scm.Scmer{scm.NewInt(1), scm.NewInt(7)}, scm.TagInt},
		{"nullable-int", []scm.Scmer{scm.NewInt(1), scm.NewNil()}, scm.JITTypeUnknown},
		{"float", []scm.Scmer{scm.NewFloat(1.5), scm.NewFloat(7.25)}, scm.TagFloat},
		{"nullable-float", []scm.Scmer{scm.NewFloat(1.5), scm.NewNil()}, scm.JITTypeUnknown},
		{"string", []scm.Scmer{scm.NewString("alpha/path"), scm.NewString("beta path")}, scm.TagString},
		{"nullable-string", []scm.Scmer{scm.NewString("alpha/path"), scm.NewNil()}, scm.JITTypeUnknown},
		{"constant", []scm.Scmer{scm.NewBool(true), scm.NewBool(true)}, scm.TagBool},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage := buildViaCompression(len(test.values), func(i int) scm.Scmer { return test.values[i] })
			got := storage.JITValueType()
			if got != test.want {
				t.Fatalf("%T.JITValueType() = %d, want %d", storage, got, test.want)
			}
			if got != scm.JITTypeUnknown {
				for i := range test.values {
					if valueTag := storage.GetValue(uint32(i)).GetTag(); valueTag != got {
						t.Fatalf("%T promised tag %d but row %d returned %d", storage, got, i, valueTag)
					}
				}
			}
		})
	}

	homogeneousEnum := buildEnum(3, func(i int) scm.Scmer {
		return []scm.Scmer{scm.NewString("a"), scm.NewString("b"), scm.NewString("a")}[i]
	})
	if got := homogeneousEnum.JITValueType(); got != scm.TagString {
		t.Fatalf("homogeneous enum type = %d, want string", got)
	}
	mixedEnum := buildEnum(3, func(i int) scm.Scmer {
		return []scm.Scmer{scm.NewInt(1), scm.NewString("one"), scm.NewInt(1)}[i]
	})
	if got := mixedEnum.JITValueType(); got != scm.JITTypeUnknown {
		t.Fatalf("mixed enum type = %d, want unknown", got)
	}

	mutable := &StorageSCMER{}
	mutable.prepare()
	mutable.scan(0, scm.NewInt(1))
	mutable.init(1)
	mutable.build(0, scm.NewInt(1))
	mutable.finish()
	if got := mutable.JITValueType(); got != scm.JITTypeUnknown {
		t.Fatalf("mutable SCMER storage type = %d, want unknown", got)
	}
}

func TestDirectReadDeltaKeepsRuntimeScmerType(t *testing.T) {
	main := buildStorageInt([]scm.Scmer{scm.NewInt(7)})
	shard := &storageShard{
		main_count:   1,
		columns:      map[string]ColumnStorage{"value": main},
		deltaColumns: map[string]int{"value": 0},
		inserts:      [][]scm.Scmer{{scm.NewString("delta")}},
	}
	mr := ShardMapReducer{
		shard:       shard,
		mainCount:   1,
		mainCols:    []ColumnStorage{main},
		colNames:    []string{"value"},
		args:        make([]scm.Scmer, 2),
		directRead:  true,
		mainGetters: nil,
	}

	mr.loadDirectReadArgs(1, 0, false)
	if got := mr.args[1]; got.GetTag() != scm.TagString || got.String() != "delta" {
		t.Fatalf("delta value was specialized as main storage: got %v (tag %d)", got, got.GetTag())
	}
}
