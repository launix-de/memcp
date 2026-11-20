// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "unicode/utf8"

// Text coordinates are explicit operator inputs, independent of SQL types.
func textUnits(text, unit string) int64 {
	switch unit {
	case "byte":
		return int64(len(text))
	case "rune":
		return int64(utf8.RuneCountInString(text))
	case "utf16":
		return utf16Length(text)
	default:
		panic("unknown text coordinate unit")
	}
}
func textPrefixBytes(text string, limit int64, unit string) int {
	if limit < 0 {
		panic("text prefix limit must be nonnegative")
	}
	if unit == "byte" {
		if limit > int64(len(text)) {
			return len(text)
		}
		return int(limit)
	}
	if unit != "rune" && unit != "utf16" {
		panic("unknown text coordinate unit")
	}
	var used int64
	for offset, r := range text {
		width := int64(1)
		if unit == "utf16" && r > 0xffff {
			width = 2
		}
		if width > limit-used {
			return offset
		}
		used += width
	}
	return len(text)
}
func declareTextOperator(name string, fn func(...Scmer) Scmer, params []*TypeDescriptor, result string) {
	Declare(&Globalenv, &Declaration{Name: name, Fn: fn, Type: &TypeDescriptor{Kind: "func", Params: params, Return: &TypeDescriptor{Kind: result}, Const: true,
		JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, out JITValueDesc) JITValueDesc {
			ctx.Coverage.NativeCalls++
			return jitEmitGeneratedCallBoundary(ctx, declarations[name], sourceArgs, args, out)
		}, JITVirtualArgs: true, JITInlineCost: 65535}})
}
func initTextUnits() {
	declareTextOperator("text_length", func(a ...Scmer) Scmer { return NewInt(textUnits(a[0].String(), a[1].String())) }, []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "unit"}}, "int")
	declareTextOperator("text_prefix", func(a ...Scmer) Scmer {
		s := a[0].String()
		return NewString(s[:textPrefixBytes(s, int64(ToInt(a[1])), a[2].String())])
	}, []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "number", Label: "length"}, {Kind: "string", Label: "unit"}}, "string")
}
