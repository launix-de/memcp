/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"encoding/json"
	"math"
	"testing"
)

func TestExplicitTextAndJSONCodecs(t *testing.T) {
	text := NewString("a🐘b")
	if UTF16Length(text).Int() != 4 || UTF16Prefix(text, NewInt(2)).String() != "a" || UTF16Prefix(text, NewInt(3)).String() != "a🐘" {
		t.Fatal("UTF-16 units split a supplementary character")
	}
	if CodepointString(NewInt(0x1f418)).String() != "🐘" {
		t.Fatal("Unicode scalar changed")
	}
	if CodepointLength(text).Int() != 3 || CodepointPrefix(text, NewInt(2)).String() != "a🐘" || CodepointPrefix(text, NewInt(0)).String() != "" || CodepointPrefix(text, NewInt(9)).String() != text.String() {
		t.Fatal("codepoint prefix split a Unicode scalar")
	}
	// Combining marks are separate scalars, rather than a grapheme policy.
	if CodepointLength(NewString("a\u0308")).Int() != 2 || CodepointPrefix(NewString("a\u0308"), NewInt(1)).String() != "a" {
		t.Fatal("codepoint operations acquired a grapheme policy")
	}
	expectTDSPanic(t, func() { CodepointLength(NewString("\xff")) })
	expectTDSPanic(t, func() { CodepointPrefix(NewString("\xff"), NewInt(0)) })
	expectTDSPanic(t, func() { CodepointPrefix(text, NewInt(-1)) })
	expectTDSPanic(t, func() { CodepointString(NewInt(0xd800)) })
	if Float32Round(NewFloat(1.23456789)).Float() != float64(float32(1.23456789)) {
		t.Fatal("binary32 rounding changed")
	}
	expectTDSPanic(t, func() { Float32Round(NewFloat(math.Inf(1))) })
	text = NewString("9999999999999999999999999999.9999999999")
	raw, err := json.Marshal(scmerToGo(JSONNumber(text)))
	if err != nil || string(raw) != text.String() {
		t.Fatal("exact transport number changed", string(raw), err)
	}
	for _, invalid := range []string{"01", "+1", "-0x1", "NaN", "\"1\"", "[1]", "1 "} {
		expectTDSPanic(t, func() { JSONNumber(NewString(invalid)) })
	}
}
