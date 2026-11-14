/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "bytes"
import "encoding/json"
import "unicode/utf8"
import "testing"
import "runtime"
import "strings"

func TestStringBufferOperators(t *testing.T) {
	texts := []string{"", "abc", "ABC", "aa-first", "A-FIRST", "Prüfbericht Gerät K Σ ς ſ", "k σ s", "\xffx\xe2\x82!", " \tÄpfel\u00a0", strings.Repeat("Prüfbericht Inventar 10420001\n", 2600), "ab%b_c\\", "123"}
	codec := StringCodecConcat
	b, offsets := BuildStringConcatBuffer(texts)
	values := make([]Scmer, len(texts))
	for i := range texts {
		values[i] = NewBufferString(b, offsets[i])
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
				if got, want := bufferLike(v, pattern, fold), StrLikeCollation(texts[i], pattern, collation, "\\"); got != want {
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
	restored := ReadStringConcatBuffer(&encoded)
	for i, text := range texts {
		if got := NewBufferString(restored, offsets[i]).String(); got != text {
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
	codec := StringCodecConcat
	b, _ := BuildStringConcatBuffer([]string{"Prüfbericht repeated repeated"})
	var serialized bytes.Buffer
	b.WriteTo(&serialized)
	for _, n := range []int{0, 1, 5, 12, len(serialized.Bytes()) - 1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("codec %d truncated %d accepted", codec, n)
				}
			}()
			ReadStringConcatBuffer(bytes.NewReader(serialized.Bytes()[:n]))
		}()
	}
}

func TestStringBufferLargeLength(t *testing.T) {
	text := strings.Repeat("large LZ4 report\n", 150000)
	buffer, offsets := BuildStringConcatBuffer([]string{text})
	value := NewBufferString(buffer, offsets[0])
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
	buffer, offsets := BuildStringConcatBuffer([]string{text})
	value := NewBufferString(buffer, offsets[0])
	// Make decoding observable; storage validates buffers before publication.
	if !Equal(value, value) || Less(value, value) || compareBufferText(value, value, true) != 0 || generalBufferLess(value, value, false) || generalBufferLess(value, value, true) {
		t.Fatal("identical handle comparison decoded its payload")
	}
}

// Literal blocks already expose immutable bytes to TextCursor. Codec scratch
// must be required only for selected compressed blocks, including a value that
// straddles a literal/compressed boundary.

func TestStringBufferLikeCustomEscape(t *testing.T) {
	texts := []string{"alpha_beta", "a%b", "a\\xb", "日本語", "Ä_b"}
	buffer, offsets := BuildStringConcatBuffer(texts)
	for i, text := range texts {
		compressed := NewBufferString(buffer, offsets[i])
		for _, pattern := range []string{"%alpha!_%", "a!%b", "a\\_b", "_本%", "Ä!_b"} {
			for _, escape := range []string{"!", "", "Ä", "%"} {
				for _, collation := range []string{"binary", "utf8mb4_general_ci"} {
					want := StrLikeCollation(text, pattern, collation, escape)
					got := Apply(Globalenv.Vars[Symbol("strlike")], compressed, NewString(pattern), NewString(collation), NewString(escape))
					if got.Bool() != want {
						t.Fatalf("compressed %q LIKE %q ESCAPE %q (%s) = %v, want %v", text, pattern, escape, collation, got, want)
					}
				}
			}
		}
	}
}

func TestConcatJSONAndUnchangedOperators(t *testing.T) {
	texts := []string{"prefix unchanged suffix", "prefix other suffix", "<>&\"\\\x00\b\f\n\r\t\u2028\u2029\xff Prüfen", "prefix unchanged suffix prefix unchanged suffix"}
	pool, offsets := BuildStringConcatBuffer(texts)
	for i, text := range texts {
		value := NewBufferString(pool, offsets[i])
		want, _ := json.Marshal(text)
		got := appendBufferJSON(nil, value)
		var gotText, wantText string
		if err := json.Unmarshal(got, &gotText); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(want, &wantText); err != nil {
			t.Fatal(err)
		}
		if gotText != wantText || utf8.ValidString(text) && string(got) != string(want) {
			t.Fatalf("JSON %q got %s want %s", text, got, want)
		}
	}
	value := NewBufferString(pool, offsets[3])
	if bufferTrim(value, 0) != value || bufferReplace(value, "missing", "x") != value || bufferSubstring(value, 0, bufferStringLen(value)) != value {
		t.Fatal("unchanged operator lost handle identity")
	}
	lower := NewBufferString(pool, offsets[0])
	if bufferCase(lower, false) != lower {
		t.Fatal("unchanged lowercase materialized")
	}
	dst := make([]byte, 0, 1024)
	if n := testing.AllocsPerRun(100, func() {
		dst = appendBufferJSON(dst[:0], value)
		if bufferCase(lower, false) != lower {
			panic("case")
		}
	}); n != 0 {
		t.Fatalf("JSON/case allocated %v", n)
	}
	for _, name := range []string{"utf8mb4_bin", "utf8mb4_general_ci", "utf8mb4_unicode_ci", "utf8mb4_german2_ci"} {
		for _, reverse := range []bool{false, true} {
			fn := Globalenv.Vars["collate"].Func()(NewString(name), NewBool(reverse)).Func()
			for i, a := range texts {
				for j, b := range texts {
					if fn(NewBufferString(pool, offsets[i]), NewBufferString(pool, offsets[j])).Bool() != fn(NewString(a), NewString(b)).Bool() {
						t.Fatalf("collation %s changed", name)
					}
				}
			}
		}
	}
}

func TestConcatNumericCoercions(t *testing.T) {
	texts := []string{"", "false", "FALSE", "falſe", " false ", "\t", "  -42x", "+.5e2suffix", "1e+suffix", "1-999", "1.2.3", "0", "\u00a00\u00a0", "NaN", "-Inf", strings.Repeat("trace SELECT payload common suffix ", 10000)}
	pool, offsets := BuildStringConcatBuffer(texts)
	for i, text := range texts {
		v, p := NewBufferString(pool, offsets[i]), NewString(text)
		if v.Int() != p.Int() || v.Float() != p.Float() || v.Bool() != p.Bool() {
			t.Fatalf("numeric coercion %q changed", text)
		}
	}
	v := NewBufferString(pool, offsets[len(texts)-1])
	if n := testing.AllocsPerRun(100, func() {
		if v.Int() != 0 || v.Float() != 0 || !v.Bool() {
			panic("coercion")
		}
	}); n != 0 {
		t.Fatalf("nonnumeric text allocated %v", n)
	}
}

func TestConcatLengthExceedsFormerAuxLength(t *testing.T) {
	text := strings.Repeat("x", 1<<24+17)
	pool, offsets := BuildStringConcatBuffer([]string{text})
	value := NewBufferString(pool, offsets[0])
	if BufferStringLength(value) != len(text) || Globalenv.Vars["strlen"].Func()(value).Int() != int64(len(text)) {
		t.Fatal("header length truncated to former 24-bit length")
	}
	if n := testing.AllocsPerRun(10, func() {
		if value.String() != text {
			panic("direct cast")
		}
	}); n != 0 {
		t.Fatalf("single span cast allocated %v", n)
	}
}
