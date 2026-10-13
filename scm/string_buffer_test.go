/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "bytes"
import "testing"
import "runtime"
import "strings"

func TestStringBufferOperators(t *testing.T) {
	texts := []string{"", "abc", "ABC", "aa-first", "A-FIRST", "Prüfbericht Gerät K Σ ς ſ", "k σ s", "\xffx\xe2\x82!", " \tÄpfel\u00a0", strings.Repeat("Prüfbericht Inventar 10420001\n", 2600), "ab%b_c\\", "123"}
	codec := StringCodecLZ4
	b, offsets := BuildCompressedStringBuffer(texts)
	values := make([]Scmer, len(texts))
	for i, text := range texts {
		values[i] = NewBufferString(b, offsets[i], len(text))
	}
	for i, v := range values {
		if !v.IsString() || v.String() != texts[i] {
			t.Fatalf("codec %d row %d roundtrip", codec, i)
		}
		var scratch TextScratch
		var cursor TextCursor
		cursor.Reset(v)
		var runes []rune
		for {
			r, _, ok := cursor.NextRune(&scratch)
			if !ok {
				break
			}
			runes = append(runes, r)
		}
		if string(runes) != string([]rune(texts[i])) {
			t.Fatalf("rune stream %d %d", codec, i)
		}
		for j, w := range values {
			plain := NewString(texts[j])
			for _, other := range []Scmer{w, plain} {
				if Equal(v, other) != (texts[i] == texts[j]) {
					t.Fatalf("Equal %d %d %d", codec, i, j)
				}
				if Less(v, other) != (texts[i] < texts[j]) {
					t.Fatalf("Less %d %d %d", codec, i, j)
				}
				if equalStringValues(v, other, true) != strings.EqualFold(texts[i], texts[j]) {
					t.Fatalf("EqualFold %d %d %d", codec, i, j)
				}
			}
		}
		for _, pattern := range []string{"", "%", "_", "%bericht%", "Prüf%Gerät%", "%K%", "%Σ%", "%\\%%", "%\\_%", "a%b%c", "%_x%", "\\", strings.Repeat("a", 1100) + "%"} {
			for _, fold := range []bool{false, true} {
				collation := "bin"
				if fold {
					collation = "utf8mb4_general_ci"
				}
				if got, want := bufferLike(v, pattern, fold), StrLikeCollation(texts[i], pattern, collation); got != want {
					t.Fatalf("LIKE codec=%d row=%d pattern=%q fold=%t got=%t want=%t", codec, i, pattern, fold, got, want)
				}
			}
		}
		for _, start := range []int{0, 1, 7, 65530} {
			if start > len(texts[i]) {
				continue
			}
			end := min(start+17, len(texts[i]))
			sub := bufferSubstring(v, start, end)
			if sub.String() != texts[i][start:end] {
				t.Fatalf("substring %d %d", codec, i)
			}
		}
		if bufferCase(v, false).String() != strings.ToLower(texts[i]) || bufferCase(v, true).String() != strings.ToUpper(texts[i]) {
			t.Fatalf("case %d %d", codec, i)
		}
		if bufferTrim(v, 0).String() != strings.TrimSpace(texts[i]) {
			t.Fatalf("trim %d %d", codec, i)
		}
		if HashKey(v) != HashKey(NewString(texts[i])) {
			t.Fatalf("codec %d changed text hash", codec)
		}
	}
	var encoded bytes.Buffer
	b.WriteTo(&encoded)
	restored := ReadCompressedStringBuffer(&encoded)
	for i, text := range texts {
		if got := NewBufferString(restored, offsets[i], len(text)).String(); got != text {
			t.Fatalf("restored buffer %d %d", codec, i)
		}
	}
	for _, i := range []int{1, 5, 9} {
		v := values[i]
		plain := NewString(texts[i])
		sub := bufferSubstring(v, 1, len(texts[i]))
		if got := testing.AllocsPerRun(30, func() {
			if !Equal(v, plain) || Less(v, plain) || !bufferLike(v, "%", true) || sub.IsNil() || HashKey(v) != HashKey(plain) {
				panic("bad operation")
			}
		}); got != 0 {
			t.Fatalf("codec %d read operations allocate: %v", codec, got)
		}
	}
	v := values[5]
	b = nil
	values = nil
	runtime.GC()
	if v.String() != texts[5] {
		t.Fatal("handle lost buffer")
	}
}

func TestStringBufferRejectsMalformedPayload(t *testing.T) {
	codec := StringCodecLZ4
	b, _ := BuildCompressedStringBuffer([]string{"Prüfbericht repeated repeated"})
	var serialized bytes.Buffer
	b.WriteTo(&serialized)
	for _, n := range []int{0, 1, 5, 12, len(serialized.Bytes()) - 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("codec %d truncated %d accepted", codec, n)
				}
			}()
			ReadCompressedStringBuffer(bytes.NewReader(serialized.Bytes()[:n]))
		}()
	}
}

func TestStringBufferLargeLength(t *testing.T) {
	text := strings.Repeat("large LZ4 report\n", 150000)
	buffer, offsets := BuildCompressedStringBuffer([]string{text})
	value := NewBufferString(buffer, offsets[0], len(text))
	if BufferStringLength(value) != len(text) || value.String() != text {
		t.Fatal("large handle length truncated")
	}
	if got := BufferSlice(value, len(text)-4, len(text)).String(); got != text[len(text)-4:] {
		t.Fatal("large substring offset truncated")
	}
}

// Identical immutable handles prove equality without touching encoded blocks.
func TestStringBufferIdentityComparison(t *testing.T) {
	text := strings.Repeat("identical dictionary group key ", 1000)
	buffer, offsets := BuildCompressedStringBuffer([]string{text})
	value := NewBufferString(buffer, offsets[0], len(text))
	// Make decoding observable; storage validates buffers before publication.
	buffer.data = nil
	if !Equal(value, value) || Less(value, value) || compareBufferText(value, value, true) != 0 || generalBufferLess(value, value, false) || generalBufferLess(value, value, true) {
		t.Fatal("identical handle comparison decoded its payload")
	}
}
