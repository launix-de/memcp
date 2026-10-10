/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"database/sql"
	"strconv"
	"strings"
	"testing"
)

func TestTDSExactNumericWireValues(t *testing.T) {
	for _, fixture := range []struct {
		kind                   byte
		precision, scale, size int
		text                   string
	}{{0x6a, 9, 2, 5, "-1234567.89"}, {0x6c, 19, 4, 9, "900719925474099.1234"}, {0x6a, 28, 8, 13, "12345678901234567890.12345678"}, {0x6a, 38, 10, 17, "9999999999999999999999999999.9999999999"}, {0x6e, 19, 4, 8, "-922337203685477.5808"}, {0x6e, 19, 4, 8, "922337203685477.5807"}, {0x6e, 10, 4, 4, "-214748.3648"}} {
		column := tdsColumn{kind: fixture.kind, precision: byte(fixture.precision), scale: byte(fixture.scale), size: fixture.size}
		coefficient := strings.ReplaceAll(fixture.text, ".", "")
		value := CoefficientEncode(NewString(coefficient))
		if fixture.kind == 0x6e {
			n, err := strconv.ParseInt(coefficient, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			value = NewInt(n)
		}
		encoded := tdsEncodeNumeric(column, value)
		metadata := []byte{fixture.kind, byte(fixture.size)}
		if fixture.kind != 0x6e {
			metadata = append(metadata, byte(fixture.precision), byte(fixture.scale))
		}
		decoder := tdsDecoder{data: append(metadata, encoded...)}
		actual := decoder.parameter()
		if decoder.err != nil || !Equal(actual, value) || len(decoder.data) != 0 {
			t.Fatal(fixture, actual, decoder.err)
		}
		decoder = tdsDecoder{data: append(metadata, 0)}
		if value := decoder.parameter(); decoder.err != nil || !value.IsNil() {
			t.Fatal("numeric NULL", fixture, value, decoder.err)
		}
	}
	// Golden signed-high/unsigned-low MONEY representation of its lower bound.
	value := NewInt(-9223372036854775808)
	if actual := tdsEncodeNumeric(tdsColumn{kind: 0x6e, size: 8}, value); !bytes.Equal(actual, []byte{8, 0, 0, 0, 128, 0, 0, 0, 0}) {
		t.Fatalf("money high-low ordering %x", actual)
	}
	// A precision-38 value may use a shorter legal magnitude on input.
	decoder := tdsDecoder{data: []byte{0x6a, 17, 38, 0, 5, 1, 42, 0, 0, 0}}
	if value := decoder.parameter(); decoder.err != nil || CoefficientDecode(value).String() != "42" {
		t.Fatal(value, decoder.err)
	}
}

func TestTDSExactNumericMalformedValues(t *testing.T) {
	for _, fixture := range [][]byte{
		{0x6a, 17, 39, 0, 0}, {0x6a, 5, 9, 10, 0}, {0x6a, 9, 9, 0, 0},
		{0x6a, 5, 9, 0, 5, 2, 0, 0, 0, 0},
		{0x6a, 5, 9, 0, 9, 1, 1, 0, 0, 0, 0, 0, 0, 0}, {0x6a, 5, 9, 0, 5, 1, 1},
		{0x6e, 8, 4, 0, 0, 0, 0}, {0x6e, 3, 0},
	} {
		decoder := tdsDecoder{data: fixture}
		decoder.parameter()
		if decoder.err == nil {
			t.Fatalf("invalid numeric accepted %x", fixture)
		}
	}
	expectTDSPanic(t, func() { tdsEncodeNumeric(tdsColumn{kind: 0x6a, precision: 9, scale: 2, size: 5}, NewFloat(0.1)) })
	expectTDSPanic(t, func() { tdsEncodeNumeric(tdsColumn{kind: 0x6a, precision: 9, scale: 2, size: 5}, NewString("0.1")) })
	expectTDSPanic(t, func() {
		tdsEncodeNumeric(tdsColumn{kind: 0x6a, precision: 9, scale: 2, size: 5}, CoefficientEncode(NewString("4294967296")))
	})
}

func TestTDSExactNumericReleasedClient(t *testing.T) {
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		server.convertParameter = NewFunc(func(a ...Scmer) Scmer {
			if !a[0].IsString() {
				panic("fixture input must be text")
			}
			value := CoefficientEncode(NewString(strings.ReplaceAll(a[0].String(), ".", "")))
			return NewSlice([]Scmer{a[4], value})
		})
		query := server.query
		server.query = NewFunc(func(a ...Scmer) Scmer {
			if a[1].String() != "SELECT exact" {
				return Apply(query, a...)
			}
			a[3].Func()(NewSlice([]Scmer{NewString("v")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("DECIMAL"), NewString("precision"), NewInt(38), NewString("scale"), NewInt(10)})}))
			value := a[4].Func()(NewString("tsql_param:v"))
			a[2].Func()(NewSlice([]Scmer{NewString("v"), value}))
			return NewInt(1)
		})
	})
	text := "1234567890123456789012345678.1234567890"
	var actual string
	err := db.QueryRow("sp_executesql", sql.Named("stmt", "SELECT exact"), sql.Named("params", "@v decimal(38,10)"), sql.Named("v", text)).Scan(&actual)
	if err != nil || actual != text {
		t.Fatal(actual, err)
	}
	if err := db.QueryRow("sp_executesql", sql.Named("stmt", "SELECT exact"), sql.Named("params", "@v decimal(38,10)"), sql.Named("v", "overflow.5")).Scan(&actual); err == nil {
		t.Fatal("invalid declared decimal input accepted")
	}
	var integer int
	if err := db.QueryRow("SELECT 1").Scan(&integer); err != nil || integer != 1 {
		t.Fatal("numeric error desynchronized connection", integer, err)
	}
}
