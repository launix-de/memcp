// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

// Package stringconcat owns the immutable concat-pool format. Keeping the codec
// below both storage and scm lets runtime values retain it without an import cycle.
package stringconcat

import "io"
import "bytes"
import "math/bits"
import "unsafe"
import "encoding/binary"

const MaxOffset = uint64(1<<56 - 1)

// Buffer contains raw byte spans and concat entries in the same immutable pool.
// An entry is [byte length, (span offset, span length)*, 0]. Word 1 instead of
// a span offset denotes one inline raw string of the declared length, then 0.
// Words use unsigned base-128 continuation bytes. Raw spans never refer to other entries.
type Buffer struct {
	data []byte
}

// Unsigned base-128 words reserve the high bit as a continuation marker.
func word(dst []byte, value uint64) []byte { return binary.AppendUvarint(dst, value) }
func readWord(data []byte, pos *int) uint64 {
	if *pos >= len(data) {
		panic("truncated concat word")
	}
	value, n := binary.Uvarint(data[*pos:])
	if n <= 0 {
		panic("invalid concat word")
	}
	*pos += n
	return value
}
func wordSize(v uint64) int {
	n := (bits.Len64(v) + 6) / 7
	if n == 0 {
		return 1
	}
	return n
}

func (b *Buffer) Strlen(offset uint64) int {
	if offset > uint64(len(b.data)) {
		panic("concat offset out of range")
	}
	p := int(offset)
	n := readWord(b.data, &p)
	if n > uint64(int(^uint(0)>>1)) {
		panic("concat string too large")
	}
	return int(n)
}

func (b *Buffer) Size() uint { return uint(unsafe.Sizeof(*b)) + uint(cap(b.data)) }

// Cursor borrows immutable literal spans. Copies are independent and require no
// stack of references, decoder workspace, locks, or allocations.
type Cursor struct {
	buffer    *Buffer
	pos       int
	remaining int
	span      []byte
}

func (c *Cursor) Reset(b *Buffer, offset uint64) int {
	*c = Cursor{buffer: b, pos: int(offset)}
	c.remaining = int(readWord(b.data, &c.pos))
	if c.remaining != 0 && c.pos < len(b.data) && b.data[c.pos] == 1 {
		c.pos++
		c.span = b.data[c.pos : c.pos+c.remaining]
		c.pos += c.remaining
	}
	return c.remaining
}

func (c *Cursor) Chunk() []byte {
	if c.remaining == 0 {
		return nil
	}
	if len(c.span) == 0 {
		offset := readWord(c.buffer.data, &c.pos)
		length := readWord(c.buffer.data, &c.pos)
		c.span = c.buffer.data[offset : offset+length]
	}
	return c.span
}

func (c *Cursor) Advance(n int) { c.span = c.span[n:]; c.remaining -= n }

func (c *Cursor) Skip(n int) {
	if n < 0 || n > c.remaining {
		panic("concat slice bounds")
	}
	for n > 0 {
		m := min(n, len(c.Chunk()))
		c.Advance(m)
		n -= m
	}
}

// Direct returns a borrowed Go string for a single contiguous literal span.
func (b *Buffer) Direct(offset uint64) (string, bool) {
	var c Cursor
	c.Reset(b, offset)
	chunk := c.Chunk()
	if len(chunk) != c.remaining {
		return "", false
	}
	return unsafe.String(unsafe.SliceData(chunk), len(chunk)), true
}

func (b *Buffer) Append(offset uint64, dst []byte) []byte {
	var c Cursor
	c.Reset(b, offset)
	if cap(dst)-len(dst) < c.remaining {
		next := make([]byte, len(dst), len(dst)+c.remaining)
		copy(next, dst)
		dst = next
	}
	for len(c.Chunk()) > 0 {
		chunk := c.Chunk()
		dst = append(dst, chunk...)
		c.Advance(len(chunk))
	}
	return dst
}

func (b *Buffer) Extract(offset uint64) string {
	if s, ok := b.Direct(offset); ok {
		return s
	}
	dst := make([]byte, 0, b.Strlen(offset))
	dst = b.Append(offset, dst)
	return unsafe.String(unsafe.SliceData(dst), len(dst))
}

type span struct{ offset, end int }

// Builder's trigram index addresses only literal spans, never concat metadata.
// It is discarded on Finish. An integer trigram key avoids allocating map keys.
type Builder struct {
	buffer   Buffer
	grams    map[uint32][]span
	inserted []uint32
}

func Raw(text string) *Buffer { builder := new(Builder); builder.Add(text); return builder.Finish() }

func NewBuilder() *Builder { return &Builder{grams: make(map[uint32][]span)} }
func gram(s string) uint32 { return uint32(s[0])<<16 | uint32(s[1])<<8 | uint32(s[2]) }

func (b *Builder) literal(s string) span {
	off := len(b.buffer.data)
	b.buffer.data = append(b.buffer.data, s...)
	end := len(b.buffer.data)
	if b.grams == nil {
		return span{off, end}
	}
	for i := 0; i+3 <= len(s); i++ {
		key := gram(s[i:])
		b.grams[key] = append(b.grams[key], span{off + i, end})
		b.inserted = append(b.inserted, key)
	}
	return span{off, end}
}

func (b *Builder) match(s string) span {
	best := span{}
	if len(s) < 3 || len(b.grams[gram(s)]) == 0 {
		return best
	}
	for _, candidate := range b.grams[gram(s)] {
		n := min(len(s), candidate.end-candidate.offset)
		if n <= best.end-best.offset {
			continue
		}
		i := 3
		for i < n && b.buffer.data[candidate.offset+i] == s[i] {
			i++
		}
		if i > best.end-best.offset {
			best = span{candidate.offset, candidate.offset + i}
		}
		if i == len(s) {
			break
		}
	}
	if n := best.end - best.offset; n <= wordSize(uint64(best.offset))+wordSize(uint64(n)) {
		return span{}
	}
	return best
}

func (b *Builder) Add(s string) uint64 {
	b.inserted = b.inserted[:0]
	base := len(b.buffer.data)
	// The first entry starts at zero. Its raw payload starts after its length
	// word and the reserved inline marker, so zero/one can never be span offsets.
	if len(b.buffer.data) == 0 {
		b.buffer.data = word(b.buffer.data, uint64(len(s)))
		if len(s) > 0 {
			b.buffer.data = append(b.buffer.data, 1)
			b.literal(s)
		}
		b.buffer.data = append(b.buffer.data, 0)
		return 0
	}
	var local [128]span
	parts := local[:0]
	literalStart := 0
	for i := 0; i < len(s); {
		match := b.match(s[i:])
		if match.end == 0 {
			i++
			continue
		}
		if literalStart < i {
			parts = append(parts, b.literal(s[literalStart:i]))
		}
		parts = append(parts, match)
		i += match.end - match.offset
		literalStart = i
	}
	if len(parts) == 0 {
		offset := uint64(len(b.buffer.data))
		b.buffer.data = word(b.buffer.data, uint64(len(s)))
		if len(s) > 0 {
			b.buffer.data = append(b.buffer.data, 1)
			b.literal(s)
		}
		b.buffer.data = append(b.buffer.data, 0)
		return offset
	}
	if literalStart < len(s) {
		parts = append(parts, b.literal(s[literalStart:]))
	}
	offset := uint64(len(b.buffer.data))
	if offset > MaxOffset {
		panic("concat pool exceeds 56-bit offset")
	}
	var scratch [512]byte
	sequence := word(scratch[:0], uint64(len(s)))
	for _, p := range parts {
		sequence = word(sequence, uint64(p.offset))
		sequence = word(sequence, uint64(p.end-p.offset))
	}
	sequence = append(sequence, 0)
	if len(b.buffer.data)-base+len(sequence) >= wordSize(uint64(len(s)))+len(s)+2 && len(s) > 0 {
		// Retire this private attempt before choosing raw, including all index
		// entries for discarded literal spans. No unreachable payload is retained.
		for i := len(b.inserted) - 1; i >= 0; i-- {
			key := b.inserted[i]
			items := b.grams[key]
			items = items[:len(items)-1]
			if len(items) == 0 {
				delete(b.grams, key)
			} else {
				b.grams[key] = items
			}
		}
		b.inserted = b.inserted[:0]
		b.buffer.data = b.buffer.data[:base]
		offset = uint64(base)
		b.buffer.data = word(b.buffer.data, uint64(len(s)))
		b.buffer.data = append(b.buffer.data, 1)
		b.literal(s)
		b.buffer.data = append(b.buffer.data, 0)
	} else {
		b.buffer.data = append(b.buffer.data, sequence...)
	}

	return offset
}

func (b *Builder) Finish() *Buffer {
	b.buffer.data = append([]byte(nil), b.buffer.data...)
	result := b.buffer
	b.grams = nil
	b.inserted = nil
	b.buffer = Buffer{}
	return &result
}

// WriteTo writes only the immutable pool. Entry offsets live with the owning
// column or wire values; no build index or decoded cache is persisted.
func (b *Buffer) WriteTo(w io.Writer) {
	if err := binary.Write(w, binary.LittleEndian, uint64(len(b.data))); err != nil {
		panic(err)
	}
	if _, err := w.Write(b.data); err != nil {
		panic(err)
	}
}

func Read(r io.Reader) *Buffer {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		panic(err)
	}
	if length > MaxOffset || length > uint64(int(^uint(0)>>1)) {
		panic("invalid concat pool size")
	}
	var data bytes.Buffer
	if _, err := io.CopyN(&data, r, int64(length)); err != nil {
		panic(err)
	}
	b := &Buffer{data: data.Bytes()}
	if len(b.data) != int(length) {
		panic("truncated concat pool")
	}
	return b
}

func (b *Buffer) Validate(offset uint64) {
	if offset >= uint64(len(b.data)) {
		panic("invalid concat entry offset")
	}
	p := int(offset)
	remaining := readWord(b.data, &p)
	for remaining > 0 {
		ref := readWord(b.data, &p)
		if ref == 1 {
			if remaining > uint64(len(b.data)-p) {
				panic("truncated raw concat entry")
			}
			p += int(remaining)
			remaining = 0
			break
		}
		length := readWord(b.data, &p)
		if ref < 2 || length == 0 || length > remaining || ref > uint64(len(b.data)) || length > uint64(len(b.data))-ref {
			panic("invalid concat span")
		}
		remaining -= length
	}
	if readWord(b.data, &p) != 0 {
		panic("missing concat terminator")
	}
}
