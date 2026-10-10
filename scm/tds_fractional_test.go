/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"testing"
	"time"
)

func TestTDSFractionalTimeAndLegacyDatetimeGoldens(t *testing.T) {
	for _, fixture := range []struct {
		payload   []byte
		column    tdsColumn
		kind      string
		day       int32
		ticks     uint64
		precision int
	}{
		{[]byte{0x29, 7, 5, 1, 0, 0, 0, 0}, tdsColumn{kind: 0x29, scale: 7}, "TIME", 0, 1, 7},
		{[]byte{0x6f, 8, 8, 0, 0, 0, 0, 1, 0, 0, 0}, tdsColumn{kind: 0x6f, size: 8}, "DATETIME", 0, 1, 3},
		{[]byte{0x6f, 8, 8, 255, 255, 255, 255, 43, 1, 0, 0}, tdsColumn{kind: 0x6f, size: 8}, "DATETIME", -1, 299, 3},
	} {
		decoder := tdsDecoder{data: fixture.payload}
		value := decoder.parameter()
		day, ticks, offset := tdsTemporalParts(value)
		if decoder.err != nil || day != int64(fixture.day) || ticks != int64(fixture.ticks) || offset != 0 {
			t.Fatal(fixture, value, decoder.err)
		}
		if encoded := tdsEncodeTemporal(fixture.column, value); !bytes.Equal(encoded, fixture.payload[2:]) {
			t.Fatalf("fractional ticks changed %x != %x", encoded, fixture.payload[2:])
		}
	}
	value := tdsTemporalPayload(0, 1, 0)
	expectTDSPanic(t, func() { tdsEncodeTemporal(tdsColumn{kind: 0x29, scale: 8}, value) })
	expectTDSPanic(t, func() { tdsEncodeTemporal(tdsColumn{kind: 0x29, scale: 7}, tdsTemporalPayload(1, 1, 0)) })
}

func TestTDSFractionalDatetime2RangesAndPrecision(t *testing.T) {
	for _, precision := range []int{1, 2, 3, 4, 5, 6, 7} {
		clock := uint64(13*3600+45*60+30)*tdsTimeFactor(byte(precision)) + 1
		value := tdsTemporalPayload(738944, int64(clock), 0)
		column := tdsColumn{kind: 0x2a, scale: byte(precision)}
		encoded := tdsEncodeTemporal(column, value)
		decoder := tdsDecoder{data: append([]byte{0x2a, byte(precision)}, encoded...)}
		actual := decoder.parameter()
		day, ticks, offset := tdsTemporalParts(actual)
		if decoder.err != nil || day != 738944 || ticks != int64(clock) || offset != 0 {
			t.Fatal(precision, actual, decoder.err)
		}
	}
	value := tdsTemporalPayload(3652058, 86400*10000000-1, 0)
	encoded := tdsEncodeTemporal(tdsColumn{kind: 0x2a, scale: 7}, value)
	decoder := tdsDecoder{data: append([]byte{0x2a, 7}, encoded...)}
	if actual := decoder.parameter(); decoder.err != nil || !Equal(actual, value) {
		t.Fatal("upper endpoint", actual, decoder.err)
	}
	// Calendar/type-domain bounds are frontend policy. The wire adapter only
	// enforces packet field widths and preserves otherwise representable data.
	invalid := []byte{0x2a, 7, 8, 1}
	decoder = tdsDecoder{data: invalid}
	decoder.parameter()
	if decoder.err == nil {
		t.Fatal("truncated clock payload accepted")
	}

}

func TestTDSFractionalDatetime2ReleasedClient(t *testing.T) {
	date := time.Date(2024, 2, 29, 13, 45, 30, 123456700, time.UTC)
	day := int32((time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC).Unix() - tdsDateEpoch.Unix()) / 86400)
	clock := uint64(13*3600+45*60+30)*10000000 + 1234567
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		query := server.query
		server.query = NewFunc(func(a ...Scmer) Scmer {
			if a[1].String() != "SELECT fractional" {
				return Apply(query, a...)
			}
			a[3].Func()(NewSlice([]Scmer{NewString("v")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("DATETIME2"), NewString("scale"), NewInt(7)})}))
			a[2].Func()(NewSlice([]Scmer{NewString("v"), tdsTemporalPayload(int64(day), int64(clock), 0)}))
			return NewInt(1)
		})
	})
	var actual time.Time
	if err := db.QueryRow("SELECT fractional").Scan(&actual); err != nil || !actual.Equal(date) {
		t.Fatal(actual, err)
	}
}

func TestTDSOffsetWirePreservesOriginalOffset(t *testing.T) {
	for _, offset := range []int64{-840, -60, 0, 90, 840} {
		payload := tdsTemporalPayload(738944, 123456789012, offset)
		column := tdsColumn{kind: 0x2b, scale: 7}
		encoded := tdsEncodeTemporal(column, payload)
		decoder := tdsDecoder{data: append([]byte{0x2b, 7}, encoded...)}
		actual := decoder.parameter()
		if decoder.err != nil || !Equal(actual, payload) {
			t.Fatal("offset payload changed", offset, decoder.err)
		}
		if len(encoded) != 11 {
			t.Fatal("offset wire width", len(encoded))
		}
	}
	for _, offset := range []int64{-32769, 32768} {
		expectTDSPanic(t, func() { tdsEncodeTemporal(tdsColumn{kind: 0x2b, scale: 7}, tdsTemporalPayload(738944, 1, offset)) })
	}
	decoder := tdsDecoder{data: []byte{0x2b, 7, 0}}
	if value := decoder.parameter(); decoder.err != nil || !value.IsNil() || tdsDescriptorValue(decoder.descriptor, "wire_kind").Int() != 0x2b || tdsDescriptorValue(decoder.descriptor, "scale").Int() != 7 {
		t.Fatal("typed offset NULL lost descriptor", decoder.err)
	}
}
