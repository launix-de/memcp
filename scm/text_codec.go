/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"encoding/json"
	"math"
	"unicode/utf8"
)

func UTF16Length(a ...Scmer) Scmer {
	if !a[0].IsString() || !utf8.ValidString(a[0].String()) {
		panic("UTF-16 length requires valid Unicode text")
	}
	count := int64(0)
	for _, r := range a[0].String() {
		count++
		if r > 0xffff {
			count++
		}
	}
	return NewInt(count)
}
func UTF16Prefix(a ...Scmer) Scmer {
	remaining := integralArgument(a[1])
	if !a[0].IsString() || !utf8.ValidString(a[0].String()) || remaining < 0 {
		panic("UTF-16 prefix requires valid text and a nonnegative unit limit")
	}
	text := a[0].String()
	for index, r := range text {
		units := int64(1)
		if r > 0xffff {
			units = 2
		}
		if units > remaining {
			return NewString(text[:index])
		}
		remaining -= units
	}
	return a[0]
}

func CodepointLength(a ...Scmer) Scmer {
	if !a[0].IsString() || !utf8.ValidString(a[0].String()) {
		panic("codepoint length requires valid Unicode text")
	}
	return NewInt(int64(utf8.RuneCountInString(a[0].String())))
}

func CodepointPrefix(a ...Scmer) Scmer {
	remaining := integralArgument(a[1])
	if !a[0].IsString() || !utf8.ValidString(a[0].String()) || remaining < 0 {
		panic("codepoint prefix requires valid text and a nonnegative limit")
	}
	for index := range a[0].String() {
		if remaining == 0 {
			return NewString(a[0].String()[:index])
		}
		remaining--
	}
	return a[0]
}
func CodepointString(a ...Scmer) Scmer {
	r := integralArgument(a[0])
	if r < 0 || r > utf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
		panic("invalid Unicode scalar")
	}
	return NewString(string(rune(r)))
}
func Float32Round(a ...Scmer) Scmer {
	if !a[0].IsFloat() && !a[0].IsInt() {
		panic("float32_round requires a numeric scalar")
	}
	x := a[0].Float()
	y := float64(float32(x))
	if math.IsNaN(x) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		panic("nonfinite binary32 conversion")
	}
	return NewFloat(y)
}

// JSONNumber constructs an explicit transport scalar without converting its
// decimal spelling to a binary float. It is not a storage or arithmetic type.
func JSONNumber(a ...Scmer) Scmer {
	if !a[0].IsString() {
		panic("json_number requires text")
	}
	text := a[0].String()
	if len(text) == 0 || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) || !json.Valid([]byte(text)) {
		panic("invalid JSON number")
	}
	var number json.Number
	if err := json.Unmarshal([]byte(text), &number); err != nil || string(number) != text {
		panic("invalid JSON number")
	}
	return NewAny(number)
}
func RegisterTextPrimitives() {
	for _, d := range []struct {
		name   string
		fn     func(...Scmer) Scmer
		kinds  []string
		result string
	}{
		{"utf16_len", UTF16Length, []string{"string"}, "int"},
		{"utf16_prefix", UTF16Prefix, []string{"string", "number"}, "string"},
		{"codepoint_length", CodepointLength, []string{"string"}, "int"},
		{"codepoint_prefix", CodepointPrefix, []string{"string", "number"}, "string"},
		{"codepoint_string", CodepointString, []string{"number"}, "string"},
		{"float32_round", Float32Round, []string{"number"}, "number"},
		{"json_number", JSONNumber, []string{"string"}, "any"},
	} {
		params := make([]*TypeDescriptor, len(d.kinds))
		for i, k := range d.kinds {
			params[i] = &TypeDescriptor{Kind: k}
		}
		Declare(&Globalenv, &Declaration{Name: d.name, Fn: d.fn, Type: &TypeDescriptor{Kind: "func", Description: "Explicit scalar encoding conversion", Params: params, Return: &TypeDescriptor{Kind: d.result}, Const: true}})
	}
}
