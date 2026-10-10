/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package main

import (
	"encoding/json"
	"testing"
)

func TestProbeNumericExpectations(t *testing.T) {
	for _, test := range []struct {
		got   any
		want  json.Number
		equal bool
	}{
		{int64(41943040), "41943040", true},
		{float64(41943040), "41943040", true},
		{"4.194304e+07", "41943040", true},
		{"41943040", "4.194304e+07", true},
		{"41943041", "41943040", false},
		{int64(9007199254740993), "9007199254740993", true},
		{float64(9007199254740992), "9007199254740993", false},
		{int64(9223372036854775807), "9223372036854775807", true},
		{"9223372036854775808", "9223372036854775807", false},
		{nil, "0", false},
		{true, "1", false},
		{"NaN", "0", false},
		{"1x", "1", false},
	} {
		if got := probeValuesEqual(test.got, test.want); got != test.equal {
			t.Errorf("probeValuesEqual(%v, %s) = %v, want %v", test.got, test.want, got, test.equal)
		}
	}
}
