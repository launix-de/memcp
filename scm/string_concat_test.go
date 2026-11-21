// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "sync"
import "bytes"
import "testing"
import "strings"
import "math/rand"

func TestSamplingPoolBoundsAndRoundtrip(t *testing.T) {
	random := rand.New(rand.NewSource(1729))
	builder := NewSamplingStringConcatBuilder(8)
	texts := []string{strings.Repeat("common report ÄÖß\n", 300), strings.Repeat("common report ÄÖß\n", 299) + "changed"}
	for i := 0; i < 8; i++ {
		data := make([]byte, 32768)
		random.Read(data)
		texts = append(texts, string(data))
	}
	offsets := make([]uint64, len(texts))
	for i, text := range texts {
		offsets[i] = builder.Add(text)
		if len(builder.grams) > 8192 {
			t.Fatal("sampling index exceeded key budget")
		}
		for _, candidates := range builder.grams {
			if len(candidates) > 4 {
				t.Fatal("sampling index exceeded candidate budget")
			}
		}
	}
	pool := builder.Finish()
	for i, text := range texts {
		pool.Validate(offsets[i])
		if pool.Extract(offsets[i]) != text {
			t.Fatal("bounded index changed text", i)
		}
	}
}

func TestPoolRoundtripAndSeek(t *testing.T) {
	random := rand.New(rand.NewSource(94013))
	texts := []string{"", "a", "ab", "\x00\xff", "Prüfbericht ÄÖß ΣςK", strings.Repeat("long literal\x00", 10000)}
	for i := 0; i < 500; i++ {
		data := make([]byte, 10+random.Intn(256))
		random.Read(data)
		texts = append(texts, string(data), "common report prefix "+string(data[:10])+" common report suffix")
	}
	builder := NewStringConcatBuilder(0)
	offsets := make([]uint64, len(texts))
	for i, text := range texts {
		offsets[i] = builder.Add(text)
	}
	pool := builder.Finish()
	var wire bytes.Buffer
	pool.WriteTo(&wire)
	restored := ReadStringConcatBuffer(&wire)
	for _, pool := range []*StringConcatCompression{pool, restored} {
		for i, text := range texts {
			pool.Validate(offsets[i])
			if pool.Strlen(offsets[i]) != len(text) || pool.Extract(offsets[i]) != text {
				t.Fatalf("entry %d changed", i)
			}
			for _, start := range []int{0, len(text) / 2, len(text)} {
				var cursor stringConcatCursor
				cursor.Reset(pool, offsets[i])
				cursor.Skip(start)
				var out []byte
				for len(cursor.Chunk()) > 0 {
					chunk := cursor.Chunk()
					out = append(out, chunk...)
					cursor.Advance(len(chunk))
				}
				if string(out) != text[start:] {
					t.Fatalf("seek entry %d at %d", i, start)
				}
			}
		}
	}
}

func TestPoolNoReadAllocations(t *testing.T) {
	b := NewStringConcatBuilder(0)
	a := b.Add("common prefix long literal common suffix")
	c := b.Add("common prefix another literal common suffix")
	pool := b.Finish()
	if _, ok := pool.Direct(a); !ok {
		t.Fatal("first string must be directly castable")
	}
	dst := make([]byte, 0, 128)
	if n := testing.AllocsPerRun(100, func() {
		if pool.Strlen(c) != len("common prefix another literal common suffix") {
			panic("length")
		}
		dst = pool.Append(c, dst[:0])
		if string(dst) != "common prefix another literal common suffix" {
			panic("append")
		}
		if pool.Extract(a) != "common prefix long literal common suffix" {
			panic("direct cast")
		}
	}); n != 0 {
		t.Fatalf("read allocated %v", n)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				if pool.Extract(c) != "common prefix another literal common suffix" {
					t.Error("concurrent read changed")
				}
			}
		}()
	}
	workers.Wait()
}

func TestWordsAndMalformedEntries(t *testing.T) {
	for _, value := range []uint64{0, 1, 127, 128, 16383, 16384, 1 << 32, maxStringConcatOffset, ^uint64(0)} {
		data := concatWord(nil, value)
		p := 0
		if got := readConcatWord(data, &p); got != value || p != len(data) {
			t.Fatalf("word %d", value)
		}
	}
	for _, data := range [][]byte{{128}, {1, 0}, {3, 2, 4, 0}, {3, 1, 'a'}, {3, 0, 0}, {0, 1}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("accepted corrupt entry %x", data)
				}
			}()
			(&StringConcatCompression{data: data}).Validate(0)
		}()
	}
}

func TestPoolLongestLiteralMatch(t *testing.T) {
	b := NewStringConcatBuilder(4)
	b.Add("initial reserved payload")
	text := "abc-common-1234567890-tail!"
	b.literal("abc-common-1234567890-tail?")
	b.literal("abc-common-XYZ4567890-tail!")
	want := b.literal(text)
	got := b.match(text)
	if got != want {
		t.Fatalf("longest literal = %v, want %v", got, want)
	}
}

func TestPoolConcatScratchOverflow(t *testing.T) {
	random := rand.New(rand.NewSource(38012))
	b := NewStringConcatBuilder(81)
	parts := make([]string, 80)
	for i := range parts {
		data := make([]byte, 24)
		random.Read(data)
		parts[i] = string(data)
		b.Add(parts[i])
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	text := strings.Join(parts, "")
	offset := b.Add(text)
	pool := b.Finish()
	pool.Validate(offset)
	var cursor stringConcatCursor
	cursor.Reset(pool, offset)
	spans := 0
	for len(cursor.Chunk()) > 0 {
		chunk := cursor.Chunk()
		cursor.Advance(len(chunk))
		spans++
	}
	if spans <= 16 || pool.Extract(offset) != text {
		t.Fatalf("concat scratch overflow did not roundtrip: spans=%d", spans)
	}
}
