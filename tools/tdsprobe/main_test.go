/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package main

import (
	"encoding/json"
	"testing"
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
