// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package scm

import "io"
import "bytes"
import "unsafe"
import "math/bits"
import "encoding/binary"

const maxStringConcatOffset = uint64(1<<56 - 1)

// StringConcatCompression contains raw byte spans and concat entries in the same immutable pool.
// An entry is [byte length, (span offset, span length)*, 0]. Word 1 instead of
// a span offset denotes one inline raw string of the declared length, then 0.
// Words use unsigned base-128 continuation bytes. Raw spans never refer to other entries.
type StringConcatCompression struct {
	data []byte
}

// Unsigned base-128 words reserve the high bit as a continuation marker.
func concatWord(dst []byte, value uint64) []byte { return binary.AppendUvarint(dst, value) }
func readConcatWord(data []byte, pos *int) uint64 {
	if *pos >= len(data) {
		panic("truncated concat word")
	}
	if value := data[*pos]; value < 128 {
		*pos++
		return uint64(value)
	}
	value, n := binary.Uvarint(data[*pos:])
	if n <= 0 {
		panic("invalid concat word")
	}
	*pos += n
	return value
}

// Published pools have already been validated. The hot cursor decoder only
// reads words; malformed input is rejected by the loading caller's Validate.
func concatPoolWord(data []byte, pos *int) uint64 {
	if value := data[*pos]; value < 128 {
		*pos++
		return uint64(value)
	}
	value, n := binary.Uvarint(data[*pos:])
	*pos += n
	return value
}
func concatWordSize(v uint64) int {
	n := (bits.Len64(v) + 6) / 7
	if n == 0 {
		return 1
	}
	return n
}

func (b *StringConcatCompression) Strlen(offset uint64) int {
	if offset > uint64(len(b.data)) {
		panic("concat offset out of range")
	}
	p := int(offset)
	n := concatPoolWord(b.data, &p)
	if n > uint64(int(^uint(0)>>1)) {
		panic("concat string too large")
	}
	return int(n)
}

func (b *StringConcatCompression) Size() uint { return uint(unsafe.Sizeof(*b)) + uint(cap(b.data)) }

// stringConcatCursor borrows immutable literal spans. Copies are independent and require no
// stack of references, decoder workspace, locks, or allocations.
type stringConcatCursor struct {
	buffer    *StringConcatCompression
	pos       int
	remaining int
	span      []byte
}

func (c *stringConcatCursor) Reset(b *StringConcatCompression, offset uint64) int {
	*c = stringConcatCursor{buffer: b, pos: int(offset)}
	c.remaining = int(concatPoolWord(b.data, &c.pos))
	if c.remaining != 0 && c.pos < len(b.data) && b.data[c.pos] == 1 {
		c.pos++
		c.span = b.data[c.pos : c.pos+c.remaining]
		c.pos += c.remaining
	}
	return c.remaining
}

func (c *stringConcatCursor) Chunk() []byte {
	if c.remaining == 0 {
		return nil
	}
	if len(c.span) == 0 {
		c.loadSpan()
	}
	return c.span
}

func (c *stringConcatCursor) loadSpan() {
	offset := concatPoolWord(c.buffer.data, &c.pos)
	length := concatPoolWord(c.buffer.data, &c.pos)
	c.span = c.buffer.data[offset : offset+length]
}

func (c *stringConcatCursor) Advance(n int) { c.span = c.span[n:]; c.remaining -= n }

func (c *stringConcatCursor) Skip(n int) {
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
func (b *StringConcatCompression) Direct(offset uint64) (string, bool) {
	var c stringConcatCursor
	c.Reset(b, offset)
	chunk := c.Chunk()
	if len(chunk) != c.remaining {
		return "", false
	}
	return unsafe.String(unsafe.SliceData(chunk), len(chunk)), true
}

func (b *StringConcatCompression) Append(offset uint64, dst []byte) []byte {
	var c stringConcatCursor
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

func (b *StringConcatCompression) Extract(offset uint64) string {
	if s, ok := b.Direct(offset); ok {
		return s
	}
	dst := make([]byte, 0, b.Strlen(offset))
	dst = b.Append(offset, dst)
	return unsafe.String(unsafe.SliceData(dst), len(dst))
}

type concatSpan struct{ offset, end int }

// StringConcatBuilder's trigram index addresses only literal spans, never concat metadata.
// It is discarded on Finish. An integer trigram key avoids allocating map keys.
type StringConcatBuilder struct {
	buffer   StringConcatCompression
	grams    map[uint32][]concatSpan
	inserted []uint32
	sampling bool
}

func rawStringConcatPool(text string) *StringConcatCompression {
	builder := new(StringConcatBuilder)
	builder.Add(text)
	return builder.Finish()
}

// NewStringConcatBuilder reserves a small entry directory without reserving the full text.
func NewStringConcatBuilder(entries int) *StringConcatBuilder {
	return &StringConcatBuilder{buffer: StringConcatCompression{data: make([]byte, 0, entries*16)}, grams: make(map[uint32][]concatSpan)}
}

// NewSamplingStringConcatBuilder uses the ordinary wire format with a bounded search index.
// At most 8192 trigrams and four offsets per trigram participate in matching.
// Its conservative matches are suitable for a small compression-choice probe.
func NewSamplingStringConcatBuilder(entries int) *StringConcatBuilder {
	b := NewStringConcatBuilder(entries)
	b.sampling = true
	return b
}

// Len returns encoded bytes built so far, excluding the temporary search index.
func (b *StringConcatBuilder) Len() int { return len(b.buffer.data) }

func concatGram(s string) uint32 { return uint32(s[0])<<16 | uint32(s[1])<<8 | uint32(s[2]) }

func (b *StringConcatBuilder) literal(s string) concatSpan {
	off := len(b.buffer.data)
	b.buffer.data = append(b.buffer.data, s...)
	end := len(b.buffer.data)
	if b.grams == nil {
		return concatSpan{off, end}
	}
	if b.sampling {
		for i := 0; i+3 <= len(s); i++ {
			key := concatGram(s[i:])
			items := b.grams[key]
			if len(items) >= 4 || len(items) == 0 && len(b.grams) >= 8192 {
				continue
			}
			b.grams[key] = append(items, concatSpan{off + i, end})
			b.inserted = append(b.inserted, key)
		}
		return concatSpan{off, end}
	}
	for i := 0; i+3 <= len(s); i++ {
		key := concatGram(s[i:])
		b.grams[key] = append(b.grams[key], concatSpan{off + i, end})
		b.inserted = append(b.inserted, key)
	}
	return concatSpan{off, end}
}

func (b *StringConcatBuilder) match(s string) concatSpan {
	best := concatSpan{}
	if len(s) < 3 {
		return best
	}
	for _, candidate := range b.grams[concatGram(s)] {
		n := min(len(s), candidate.end-candidate.offset)
		if n <= best.end-best.offset {
			continue
		}
		i := 3
		for i < n && b.buffer.data[candidate.offset+i] == s[i] {
			i++
		}
		if i > best.end-best.offset {
			best = concatSpan{candidate.offset, candidate.offset + i}
		}
		if i == len(s) {
			break
		}
	}
	if n := best.end - best.offset; n <= concatWordSize(uint64(best.offset))+concatWordSize(uint64(n)) {
		return concatSpan{}
	}
	return best
}

func (b *StringConcatBuilder) Add(s string) uint64 {
	b.inserted = b.inserted[:0]
	base := len(b.buffer.data)
	// The first entry starts at zero. Its raw payload starts after its length
	// word and the reserved inline marker, so zero/one can never be span offsets.
	if len(b.buffer.data) == 0 {
		b.buffer.data = concatWord(b.buffer.data, uint64(len(s)))
		if len(s) > 0 {
			b.buffer.data = append(b.buffer.data, 1)
			b.literal(s)
		}
		b.buffer.data = append(b.buffer.data, 0)
		return 0
	}
	var local [16]concatSpan
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
		b.buffer.data = concatWord(b.buffer.data, uint64(len(s)))
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
	if offset > maxStringConcatOffset {
		panic("concat pool exceeds 56-bit offset")
	}
	var scratch [128]byte
	sequence := concatWord(scratch[:0], uint64(len(s)))
	for _, p := range parts {
		sequence = concatWord(sequence, uint64(p.offset))
		sequence = concatWord(sequence, uint64(p.end-p.offset))
	}
	sequence = append(sequence, 0)
	if len(b.buffer.data)-base+len(sequence) >= concatWordSize(uint64(len(s)))+len(s)+2 && len(s) > 0 {
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
		b.buffer.data = concatWord(b.buffer.data, uint64(len(s)))
		b.buffer.data = append(b.buffer.data, 1)
		b.literal(s)
		b.buffer.data = append(b.buffer.data, 0)
	} else {
		b.buffer.data = append(b.buffer.data, sequence...)
	}

	return offset
}

func (b *StringConcatBuilder) Finish() *StringConcatCompression {
	b.buffer.data = append([]byte(nil), b.buffer.data...)
	result := b.buffer
	b.grams = nil
	b.inserted = nil
	b.buffer = StringConcatCompression{}
	return &result
}

// WriteTo writes only the immutable pool. Entry offsets live with the owning
// column or wire values; no build index or decoded cache is persisted.
func (b *StringConcatCompression) WriteTo(w io.Writer) {
	if err := binary.Write(w, binary.LittleEndian, uint64(len(b.data))); err != nil {
		panic(err)
	}
	if _, err := w.Write(b.data); err != nil {
		panic(err)
	}
}

func ReadStringConcatBuffer(r io.Reader) *StringConcatCompression {
	var length uint64
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		panic(err)
	}
	if length > maxStringConcatOffset || length > uint64(int(^uint(0)>>1)) {
		panic("invalid concat pool size")
	}
	var data bytes.Buffer
	if _, err := io.CopyN(&data, r, int64(length)); err != nil {
		panic(err)
	}
	b := &StringConcatCompression{data: data.Bytes()}
	if len(b.data) != int(length) {
		panic("truncated concat pool")
	}
	return b
}

func (b *StringConcatCompression) Validate(offset uint64) {
	if offset >= uint64(len(b.data)) {
		panic("invalid concat entry offset")
	}
	p := int(offset)
	remaining := readConcatWord(b.data, &p)
	for remaining > 0 {
		ref := readConcatWord(b.data, &p)
		if ref == 1 {
			if remaining > uint64(len(b.data)-p) {
				panic("truncated raw concat entry")
			}
			p += int(remaining)
			remaining = 0
			break
		}
		length := readConcatWord(b.data, &p)
		if ref < 2 || length == 0 || length > remaining || ref > uint64(len(b.data)) || length > uint64(len(b.data))-ref {
			panic("invalid concat span")
		}
		remaining -= length
	}
	if readConcatWord(b.data, &p) != 0 {
		panic("missing concat terminator")
	}
}
