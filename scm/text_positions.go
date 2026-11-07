// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import (
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func originalFoldOffset(original, folded string, offset int) int {
	return textPrefixBytes(original, int64(utf8.RuneCountInString(folded[:offset])), "rune")
}
func stringFindPosition(text, needle string, start int64, fold bool, unit string) int64 {
	if start < 0 {
		panic("string_find start must be nonnegative")
	}
	if start > textUnits(text, unit) {
		return -1
	}
	begin := textPrefixBytes(text, start, unit)
	if textUnits(text[:begin], unit) != start {
		// A scalar needle cannot begin at an isolated UTF16 low surrogate.
		// Continue at the next scalar while retaining original coordinates.
		if needle == "" {
			return start
		}
		_, width := utf8.DecodeRuneInString(text[begin:])
		begin += width
	}
	suffix := text[begin:]
	sought := needle
	if fold {
		suffix = CaseFoldKey(suffix)
		sought = CaseFoldKey(sought)
	}
	found := strings.Index(suffix, sought)
	if found < 0 {
		return -1
	}
	if fold {
		found = originalFoldOffset(text[begin:], suffix, found)
	}
	return textUnits(text[:begin+found], unit)
}

// A private supplementary alphabet makes each UTF16 code unit one regexp
// scalar. Original supplementary scalars are always encoded as two symbols,
// including original private-use scalars, so the mapping is injective.
func regexpUTF16Units(text string) string {
	var out strings.Builder
	for _, r := range text {
		if r <= 0xffff {
			out.WriteRune(r)
		} else {
			hi, lo := utf16.EncodeRune(r)
			out.WriteRune(0xf0000 + hi - 0xd800)
			out.WriteRune(0xf0000 + lo - 0xd800)
		}
	}
	return out.String()
}
func regexpFindPosition(text, pattern string, fold bool, unit string) int64 {
	searched := text
	if fold {
		pattern = "(?i)" + pattern
	}
	if unit == "utf16" {
		searched = regexpUTF16Units(searched)
		pattern = regexpUTF16Units(pattern)
	} else if unit != "rune" {
		panic("regexp_find requires rune or utf16 coordinates")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic("regexp_find: " + err.Error())
	}
	found := re.FindStringIndex(searched)
	if found == nil {
		return -1
	}
	return int64(utf8.RuneCountInString(searched[:found[0]]))
}
func initTextPositions() {
	declareTextOperator("string_find", func(a ...Scmer) Scmer {
		return NewInt(stringFindPosition(a[0].String(), a[1].String(), int64(ToInt(a[2])), a[3].Bool(), a[4].String()))
	}, []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "needle"}, {Kind: "number", Label: "start"}, {Kind: "bool", Label: "fold_case"}, {Kind: "string", Label: "unit"}}, "int")
	declareTextOperator("regexp_find", func(a ...Scmer) Scmer {
		return NewInt(regexpFindPosition(a[0].String(), a[1].String(), a[2].Bool(), a[3].String()))
	}, []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "pattern"}, {Kind: "bool", Label: "fold_case"}, {Kind: "string", Label: "unit"}}, "int")
}
