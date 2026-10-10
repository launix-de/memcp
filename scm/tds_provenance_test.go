/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTDSDeclaredProvenanceAndBrowseTokens(t *testing.T) {
	var wire bytes.Buffer
	r := &tdsResult{w: &wire}
	descriptor := func(name, kind, column string, writable, key, rv bool) Scmer {
		source := NewNil()
		if column != "" {
			source = NewSlice([]Scmer{NewString("fixture"), NewString("dbo"), NewString("MixedTable"), NewString(column)})
		}
		return tdsTestDescription([]Scmer{NewString("name"), NewString(name), NewString("sql_type"), NewString(kind),
			NewString("nullable"), NewBool(false), NewString("source"), source,
			NewString("updatable"), NewBool(writable), NewString("key"), NewBool(key),
			NewString("rowversion"), NewBool(rv), NewString("browse"), NewBool(true),
			NewString("computed"), NewBool(name == "generated")})
	}
	r.fields(NewSlice([]Scmer{NewString("alias_id"), NewString("rv"), NewString("expression_value"), NewString("generated")}),
		NewSlice([]Scmer{descriptor("alias_id", "INT", "id", true, true, false),
			descriptor("rv", "ROWVERSION", "rv", true, false, true),
			descriptor("expression_value", "INT", "", true, false, false),
			descriptor("generated", "INT", "generated", true, false, false)}))
	r.publish()
	data := wire.Bytes()
	if data[0] != 0x81 || binary.LittleEndian.Uint16(data[1:]) != 4 {
		t.Fatal("missing column metadata")
	}
	offset := 3
	for i, expectedFlags := range []uint16{4, 0, 0, 0x20} {
		userType := binary.LittleEndian.Uint32(data[offset:])
		flags := binary.LittleEndian.Uint16(data[offset+4:])
		if flags != expectedFlags || (i == 1 && userType != 0x50) {
			t.Fatalf("column %d flags=%x userType=%x", i, flags, userType)
		}
		kind := data[offset+6]
		offset += 7
		if kind == 0x26 {
			offset++
		} else if kind == 0xad {
			if binary.LittleEndian.Uint16(data[offset:]) != 8 {
				t.Fatal("rowversion lost binary-eight declaration")
			}
			offset += 2
		} else {
			t.Fatalf("unexpected wire kind %x", kind)
		}
		offset += 1 + 2*int(data[offset])
	}
	if data[offset] != 0xa4 {
		t.Fatal("missing browse source-table token")
	}
	length := int(binary.LittleEndian.Uint16(data[offset+1:]))
	names := data[offset+3 : offset+3+length]
	if names[0] != 3 {
		t.Fatal("source table must have catalog/schema/table parts")
	}
	position := 1
	for _, expected := range []string{"fixture", "dbo", "MixedTable"} {
		n := 2 * int(binary.LittleEndian.Uint16(names[position:]))
		actual, err := tdsText(names[position+2 : position+2+n])
		if err != nil || actual != expected {
			t.Fatal("wrong base-table part", actual, err)
		}
		position += 2 + n
	}
	offset += 3 + length
	if data[offset] != 0xa5 {
		t.Fatal("missing browse column-information token")
	}
	info := data[offset+3:]
	if !bytes.Equal(info[:3], []byte{1, 1, 0x28}) {
		t.Fatal("aliased primary key lost source ordinal, key or different-name bit")
	}
	if !bytes.Equal(info[3:8], append([]byte{2}, tdsUTF16("id")...)) ||
		!bytes.Equal(info[8:], []byte{2, 1, 0, 3, 0, 4, 4, 1, 0}) {
		t.Fatal("base-column aliases or expression provenance are incorrect", info)
	}
}

func TestTDSStaticCursorOverridesWritableDeclaration(t *testing.T) {
	cursor := &tdsCursor{columns: []tdsColumn{{name: "id", updatable: true, source: [4]string{"fixture", "dbo", "tbl", "id"}}}}
	columns := cursor.wireColumns()
	if columns[0].updatable || !cursor.columns[0].updatable || columns[0].source != [4]string{} || cursor.columns[0].source[2] != "tbl" {
		t.Fatal("static wire snapshots must remain read-only without changing the logical cursor declaration")
	}
}
