/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "testing"
import "github.com/launix-de/memcp/scm"

func TestUUIDDefaultsArePerRowAndExplicitBeforeInsert(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tuuiddefaults")
	c := tbl.Columns[0]
	c.Typ = "UUID"
	c.AutoIncrement = false
	c.DefaultExpression = "UUIDV7"
	c.UpdateSanitizer()
	tbl.publishShowColumnsSnapshot(true)
	tbl.Unique = []uniqueKey{{Id: "PRIMARY", Cols: []string{"id"}}}
	cols, rows := tbl.prepareInsertDefaults(nil, [][]scm.Scmer{{}, {}, {}})
	if len(cols) != 1 || len(rows) != 3 {
		t.Fatal("default expansion")
	}
	for i, row := range rows {
		if !row[0].IsUUID() {
			t.Fatal("UUID default was materialized as text")
		}
		for j := 0; j < i; j++ {
			if scm.Equal(row[0], rows[j][0]) {
				t.Fatal("default sampled once for batch")
			}
		}
	}
	tbl.Insert(cols, rows, nil, scm.NewNil(), false, nil)
	if got := len(tbl.ActiveShards()[0].inserts); got != 3 {
		t.Fatalf("inserted %d", got)
	}
	_, explicit := tbl.prepareInsertDefaults([]string{"id"}, [][]scm.Scmer{{scm.NewNil()}})
	if !explicit[0][0].IsNil() {
		t.Fatal("explicit NULL replaced by UUID default")
	}
}

func TestUUIDBulkRetainsPackedValues(t *testing.T) {
	values := []string{"550e8400-e29b-41d4-a716-446655440000", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f"}
	col := buildStringColumn(values)
	out := make([]scm.Scmer, len(values))
	col.GetValueRange(0, uint32(len(values)), out, 1)
	for i, v := range out {
		if !v.IsUUID() || v.String() != values[i] {
			t.Fatalf("bulk UUID %d: %s", i, v.String())
		}
	}
	col.CompressDictionary()
	reader := col.GetCachedReader(ReadCompressed)
	defer ReleaseColumnReader(reader)
	reader.GetValueMulti([]uint32{1, 0}, out, 1)
	for i, v := range out {
		if !v.IsUUID() || v.String() != values[1-i] {
			t.Fatal("LZ4 UUID reader materialized text")
		}
	}
}
