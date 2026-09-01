/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func captureTracePrint(t *testing.T, maxLength int) *string {
	t.Helper()
	oldOutput := TracePrintFunc
	oldMaxLength := TracePrintMaxLength()
	var captured string
	TracePrintFunc = func(message string) { captured = message }
	SetTracePrintMaxLength(maxLength)
	t.Cleanup(func() {
		TracePrintFunc = oldOutput
		SetTracePrintMaxLength(oldMaxLength)
	})
	return &captured
}

func TestEmitTracePrintIsUnlimitedByDefault(t *testing.T) {
	captured := captureTracePrint(t, 0)
	message := strings.Repeat("SELECT complete_query_text ", 1000)

	EmitTracePrint(message)

	if *captured != message {
		t.Fatalf("unlimited TracePrint changed the record: got %d bytes, want %d", len(*captured), len(message))
	}
}

func TestEmitTracePrintMarksExplicitTruncation(t *testing.T) {
	captured := captureTracePrint(t, 6)

	EmitTracePrint("SELECT complete query")

	want := "SELECT... [truncated; original_bytes=21]"
	if *captured != want {
		t.Fatalf("unexpected truncated record: got %q, want %q", *captured, want)
	}
}

func TestEmitTracePrintKeepsValidUTF8(t *testing.T) {
	captured := captureTracePrint(t, 2)

	EmitTracePrint("äbc")

	if !utf8.ValidString(*captured) {
		t.Fatalf("truncated record is not valid UTF-8: %q", *captured)
	}
	if !strings.HasPrefix(*captured, "ä... [truncated;") {
		t.Fatalf("truncated record split a UTF-8 rune: %q", *captured)
	}
}

func TestNegativeTracePrintLimitMeansUnlimited(t *testing.T) {
	captured := captureTracePrint(t, -1)

	EmitTracePrint("complete")

	if *captured != "complete" || TracePrintMaxLength() != 0 {
		t.Fatalf("negative limit was not normalized to unlimited: limit=%d output=%q", TracePrintMaxLength(), *captured)
	}
}
