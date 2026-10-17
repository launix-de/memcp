/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/golang-sql/civil"
)

func TestNumericExpectationsPreserveLargeIntegers(t *testing.T) {
	if !equalValue(int64(9007199254740993), json.Number("9007199254740993")) {
		t.Fatal("exact integer result was rejected")
	}
	if equalValue(float64(9007199254740993), json.Number("9007199254740993")) {
		t.Fatal("rounded floating result passed an exact integer expectation")
	}
	if equalValue("2", json.Number("2")) {
		t.Fatal("text result passed a numeric expectation")
	}
	if !equalValue(2.5, json.Number("2.50")) {
		t.Fatal("equivalent decimal result was rejected")
	}
}

func TestBinaryExpectationsRequireBytes(t *testing.T) {
	want := map[string]any{"binary_hex": "0080ff"}
	if !equalValue([]byte{0, 128, 255}, want) || equalValue("\x00\x80\xff", want) || equalValue([]byte{0, 128}, want) {
		t.Fatal("binary expectation did not preserve byte identity")
	}
	if !equalValue([]byte{}, map[string]any{"binary_hex": ""}) || equalValue(nil, map[string]any{"binary_hex": ""}) {
		t.Fatal("empty bytes were confused with NULL")
	}
}

func TestTemporalParameterUsesExplicitDriverContracts(t *testing.T) {
	for _, test := range []struct{ kind, text, want string }{
		{"date", "0001-01-01", "civil.Date"},
		{"time", "23:59:59.1234567", "civil.Time"},
		{"datetime2", "9999-12-31T23:59:59Z", "civil.DateTime"},
	} {
		value, err := temporalParameter(map[string]any{"temporal_type": test.kind, "value": test.text})
		if err != nil || reflect.TypeOf(value).String() != test.want {
			t.Fatalf("%s: %T %v", test.kind, value, err)
		}
	}
	value, err := temporalParameter(map[string]any{"temporal_type": "time", "value": "23:59:59.1234567"})
	if err != nil || value.(civil.Time).Nanosecond != 123456700 {
		t.Fatal("probe lost fractional time before transport", value, err)
	}
	for _, encoded := range []map[string]any{
		{"temporal_type": "date", "value": "2026-02-30"},
		{"temporal_type": "datetime2", "value": "not a clock"},
		{"temporal_type": "unsupported", "value": "2026-03-15"},
		{"temporal_type": "date", "value": "2026-03-15", "extra": true},
	} {
		if _, err := temporalParameter(encoded); err == nil {
			t.Fatal("invalid probe contract accepted", encoded)
		}
	}
}
