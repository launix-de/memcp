/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"encoding/json"
	"math"
	"testing"
)

func TestExactPrimitivesAcceptSchemeNumericLiterals(t *testing.T) {
	RegisterExactPrimitives()
	RegisterUnixPrimitives()
	RegisterTextPrimitives()
	env := &Env{Vars: make(map[Symbol]Scmer), Outer: &Globalenv}
	for _, fixture := range []struct {
		source string
		want   int64
	}{
		{`((lambda (value) (coefficient_to_integer value 0 false)) (coefficient_encode "9007199254740993"))`, 9007199254740993},
		{`(coefficient_to_integer (fixed_point_rescale (coefficient_encode "125") 2 1 false) 1 true)`, 1},
		{`(fixed_point_compare (coefficient_encode "1") (coefficient_encode "10") 0 1)`, 0},
		{`(unix_from_parts 1900 1 1 0 0 0 0 10000000)`, -22089888000000000},
		{`(car (unix_parts -22089888000000000 10000000))`, 1900},
		{`(utf16_len (utf16_prefix "a🐘b" 3))`, 3},
		{`(codepoint_length (codepoint_prefix "a🐘b" 2))`, 2},
		{`(integer_from_order_key (integer_order_key -1))`, -1},
	} {
		got := EvalAll(t.Name(), fixture.source, env)
		if !got.IsInt() || got.Int() != fixture.want {
			t.Fatal("ordinary Scheme literal failed explicit integer primitive", fixture.source, got)
		}
	}
	for _, value := range []Scmer{NewFloat(0.5), NewFloat(math.NaN()), NewFloat(math.Inf(1)), NewFloat(0x1p63), NewFloat(-0x1p64), NewString("1"), NewBool(true), NewNil()} {
		expectTDSPanic(t, func() { IntegerToCoefficient(value) })
		expectTDSPanic(t, func() { FixedPointRescale(CoefficientEncode(NewString("1")), value, NewInt(0), NewBool(true)) })
		expectTDSPanic(t, func() { UnixParts(value, NewInt(10000000)) })
		expectTDSPanic(t, func() { UTF16Prefix(NewString("text"), value) })
	}
}

func TestCoefficientCodecBoundariesAndPersistence(t *testing.T) {
	previous := ""
	for _, text := range []string{"-170141183460469231731687303715884105728", "-99999999999999999999999999999999999999", "-9223372036854775808", "-10", "-2", "-1", "0", "1", "9007199254740993", "99999999999999999999999999999999999999", "170141183460469231731687303715884105727"} {
		encoded := CoefficientEncode(NewString(text))
		if len(encoded.String()) != 32 || CoefficientDecode(encoded).String() != text {
			t.Fatal("coefficient codec changed", text)
		}
		if previous != "" && previous >= encoded.String() {
			t.Fatal("binary order disagrees", text)
		}
		previous = encoded.String()
		raw, err := json.Marshal(encoded)
		if err != nil {
			t.Fatal(err)
		}
		var restored Scmer
		if err = json.Unmarshal(raw, &restored); err != nil || !Equal(encoded, restored) {
			t.Fatal("string persistence lost coefficient", err)
		}
	}
	for _, text := range []string{"170141183460469231731687303715884105728", "-170141183460469231731687303715884105729", "+1", "01", "-0", "1.0", " 1"} {
		expectTDSPanic(t, func() { CoefficientEncode(NewString(text)) })
	}
	for _, value := range []Scmer{NewNil(), NewInt(1), NewString("1"), NewString("8000000000000000000000000000000G00")} {
		expectTDSPanic(t, func() { CoefficientDecode(value) })
	}
	// A string carrying the same bytes remains an ordinary string; generic
	// arithmetic has no implicit coefficient detection or promotion.
	if !CoefficientEncode(NewString("1")).IsString() {
		t.Fatal("coefficient became a custom runtime type")
	}
}

func TestFixedPointWideIntermediatesAndExplicitRounding(t *testing.T) {
	encode := func(text string) Scmer { return CoefficientEncode(NewString(text)) }
	calculate := func(op, l, r string, ls, rs, resultScale int64, truncate bool) string {
		value := FixedPointBinary(NewString(op), encode(l), encode(r), NewInt(ls), NewInt(rs), NewInt(resultScale), NewBool(truncate))
		return CoefficientFormat(value, NewInt(resultScale)).String()
	}
	for _, fixture := range []struct {
		op, left, right, want string
		ls, rs, scale         int64
		truncate              bool
	}{
		{"+", "125", "3125", "4.375", 2, 3, 3, false},
		{"-", "125", "3125", "-1.875", 2, 3, 3, false},
		{"*", "99999999999999999999999999999999999999", "99999999999999999999999999999999999999", "0.99999999999999999999999999999999999998", 38, 38, 38, false},
		{"/", "1", "8", "0.13", 0, 0, 2, false}, {"/", "-1", "8", "-0.13", 0, 0, 2, false}, {"/", "-1", "8", "-0.12", 0, 0, 2, true},
		{"%", "-575", "2", "-1.75", 2, 0, 2, false},
	} {
		if got := calculate(fixture.op, fixture.left, fixture.right, fixture.ls, fixture.rs, fixture.scale, fixture.truncate); got != fixture.want {
			t.Fatal(fixture, got)
		}
	}
	for _, text := range []string{"12345", "-12345"} {
		value := FixedPointRescale(encode(text), NewInt(3), NewInt(2), NewBool(false))
		want := "12.35"
		if text[0] == '-' {
			want = "-12.35"
		}
		if CoefficientFormat(value, NewInt(2)).String() != want {
			t.Fatal("half rounding", text)
		}
	}
	minimum := IntegerToCoefficient(NewInt(-9223372036854775808))
	if CoefficientToInteger(minimum, NewInt(0), NewBool(true)).Int() != -9223372036854775808 {
		t.Fatal("minimum integer magnitude lost")
	}
	expectTDSPanic(t, func() { CoefficientToInteger(encode("9223372036854775808"), NewInt(0), NewBool(true)) })
	expectTDSPanic(t, func() { calculate("/", "1", "0", 0, 0, 1, false) })
	expectTDSPanic(t, func() { calculate("*", "99999999999999999999999999999999999999", "10", 0, 0, 0, false) })
}

func TestFixedPointComparisonWideIntermediates(t *testing.T) {
	encode := func(text string) Scmer { return CoefficientEncode(NewString(text)) }
	wide := encode("99999999999999999999999999999999999999")
	for _, fixture := range []struct {
		left, right  Scmer
		ls, rs, want int64
	}{
		{wide, wide, 0, 38, 1},
		{wide, wide, 38, 0, -1},
		{encode("1"), encode("100000000000000000000000000000000000000"), 0, 38, 0},
		{encode("-1"), encode("-10"), 0, 1, 0},
	} {
		if got := FixedPointCompare(fixture.left, fixture.right, NewInt(fixture.ls), NewInt(fixture.rs)).Int(); got != fixture.want {
			t.Fatal("cross-scale comparison", fixture, got)
		}
	}
}

func TestIntegerOrderKeyBoundaries(t *testing.T) {
	previous := ""
	for _, value := range []int64{-9223372036854775808, -9007199254740993, -1, 0, 1, 9007199254740993, 9223372036854775807} {
		key := IntegerOrderKey(NewInt(value))
		if len(key.String()) != 16 || IntegerFromOrderKey(key).Int() != value || previous != "" && previous >= key.String() {
			t.Fatal("integer order key changed", value, key)
		}
		previous = key.String()
	}
	for _, text := range []string{"0", "FFFFFFFFFFFFFFFF", "00000000000000000", "000000000000000g"} {
		expectTDSPanic(t, func() { IntegerFromOrderKey(NewString(text)) })
	}
}
