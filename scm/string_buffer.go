// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import "io"
import "unsafe"
import "unicode/utf8"
import "github.com/launix-de/memcp/storage/stringconcat"

const StringCodecConcat = 3

// StringConcatCompression is immutable and contains no decoded cache.
type StringConcatCompression = stringconcat.Buffer

func BuildStringConcatBuffer(entries []string) (*StringConcatCompression, []uint64) {
	if len(entries) == 1 {
		return stringconcat.Raw(entries[0]), []uint64{0}
	}
	builder := stringconcat.NewBuilder()
	offsets := make([]uint64, len(entries))
	for i, text := range entries {
		offsets[i] = builder.Add(text)
	}
	return builder.Finish(), offsets
}
func ReadStringConcatBuffer(r io.Reader) *StringConcatCompression { return stringconcat.Read(r) }

// All 56 non-tag bits identify the concat entry. Its first word is its length.
func NewBufferString(b *StringConcatCompression, offset uint64) Scmer {
	if offset > stringconcat.MaxOffset {
		panic("concat offset exceeds 56 bits")
	}
	return Scmer{(*byte)(unsafe.Pointer(b)), makeAux(tagConcat, offset)}
}
func (s Scmer) IsBufferString() bool { return s.GetTag() == tagConcat }
func bufferOf(s Scmer) *StringConcatCompression {
	return (*StringConcatCompression)(unsafe.Pointer(s.ptr))
}
func BufferStringRequiresScratch(s Scmer) bool { return false }
func BufferStringLength(s Scmer) int           { return bufferStringLen(s) }
func BufferStringOffset(s Scmer) uint64        { return bufferStringOffset(s) }
func bufferStringLen(s Scmer) int              { return bufferOf(s).Strlen(auxVal(s.aux)) }
func bufferStringOffset(s Scmer) uint64        { return auxVal(s.aux) }

// TextScratch is empty: concat cursors borrow immutable spans directly.
type TextScratch struct{}
type TextCursor struct {
	source              Scmer
	buffer              *StringConcatCompression
	concat              stringconcat.Cursor
	view                operatorStringView
	offset, length, pos int
	push                [3]byte
	pushCount           int
	single              [1]byte
}

func (c *TextCursor) Reset(s Scmer) {
	*c = TextCursor{source: s}
	if s.IsBufferString() {
		c.buffer = bufferOf(s)
		c.length = c.concat.Reset(c.buffer, auxVal(s.aux))
		return
	}
	var ok bool
	c.view, ok = makeOperatorView(s)
	if !ok {
		panic("TextCursor requires string")
	}
	c.length = c.view.n
}
func (c *TextCursor) NextByte(scratch *TextScratch) (byte, bool) {
	if c.pos >= c.length {
		return 0, false
	}
	if c.pushCount > 0 {
		c.pushCount--
		v := c.push[c.pushCount]
		c.pos++
		return v, true
	}
	chunk := c.nextChunk(scratch)
	v := chunk[0]
	c.advanceChunk(1)
	return v, true
}
func (c *TextCursor) NextRune(scratch *TextScratch) (rune, int, bool) {
	if c.pushCount == 0 {
		chunk := c.nextChunk(scratch)
		if len(chunk) == 0 {
			return 0, 0, false
		}
		if chunk[0] < utf8.RuneSelf {
			c.advanceChunk(1)
			return rune(chunk[0]), 1, true
		}
		if utf8.FullRune(chunk) {
			r, n := utf8.DecodeRune(chunk)
			c.advanceChunk(n)
			return r, n, true
		}
	}
	v, ok := c.NextByte(scratch)
	if !ok {
		return 0, 0, false
	}
	if v < utf8.RuneSelf {
		return rune(v), 1, true
	}
	var bytes [4]byte
	bytes[0] = v
	n := 1
	for !utf8.FullRune(bytes[:n]) && n < len(bytes) {
		v, ok = c.NextByte(scratch)
		if !ok {
			break
		}
		bytes[n] = v
		n++
	}
	r, size := utf8.DecodeRune(bytes[:n])
	for i := n - 1; i >= size; i-- {
		c.push[c.pushCount] = bytes[i]
		c.pushCount++
		c.pos--
	}
	return r, size, true
}

func (c *TextCursor) Position() int { return c.pos }
func (c *TextCursor) Read(dst []byte, scratch *TextScratch) int {
	n := min(len(dst), c.length-c.pos)
	if c.pushCount != 0 || c.buffer == nil && (c.view.format != 0 || c.view.base64 != "") {
		for i := 0; i < n; i++ {
			dst[i], _ = c.NextByte(scratch)
		}
		return n
	}
	written := 0
	for written < n {
		chunk := c.nextChunk(scratch)
		count := min(n-written, len(chunk))
		copy(dst[written:], chunk[:count])
		c.advanceChunk(count)
		written += count
	}
	return n
}
func (c *TextCursor) nextChunk(scratch *TextScratch) []byte {
	if c.pos >= c.length {
		return nil
	}
	if c.buffer != nil {
		return c.concat.Chunk()
	}
	if c.view.format == 0 && c.view.base64 == "" {
		return unsafe.Slice(unsafe.StringData(c.view.data[c.pos:]), c.length-c.pos)
	}
	c.single[0] = c.view.at(c.pos)
	return c.single[:]
}
func (c *TextCursor) advanceChunk(n int) {
	c.pos += n
	if c.buffer != nil {
		c.concat.Advance(n)
	}
}
func bufferSubstring(s Scmer, start, end int) Scmer {
	if start < 0 || end < start || end > bufferStringLen(s) {
		panic("substr: slice bounds out of range")
	}
	if start == 0 && end == bufferStringLen(s) {
		return s
	}
	if start == end {
		return NewString("")
	}
	if text, ok := bufferOf(s).Direct(auxVal(s.aux)); ok {
		return NewString(text[start:end])
	}
	var cursor TextCursor
	cursor.Reset(s)
	cursor.concat.Skip(start)
	cursor.pos = start
	if chunk := cursor.concat.Chunk(); len(chunk) >= end-start {
		return NewString(unsafe.String(unsafe.SliceData(chunk), end-start))
	}
	out := make([]byte, end-start)
	cursor.Read(out, nil)
	return NewString(unsafe.String(unsafe.SliceData(out), len(out)))
}
func appendBufferString(dst []byte, s Scmer) (string, []byte) {
	start := len(dst)
	dst = bufferOf(s).Append(auxVal(s.aux), dst)
	return unsafe.String(unsafe.SliceData(dst[start:]), len(dst)-start), dst
}

// collationText supplies x/text's complete collation engine with contiguous
// bytes. Single-span handles borrow their pool; fragmented values append to
// invocation-owned reusable output, without a Go string or decoded cache.
func collationText(value Scmer, dst *[]byte) []byte {
	if value.IsBufferString() {
		pool := bufferOf(value)
		offset := auxVal(value.aux)
		if text, ok := pool.Direct(offset); ok {
			return unsafe.Slice(unsafe.StringData(text), len(text))
		}
		*dst = pool.Append(offset, (*dst)[:0])
		return *dst
	}
	text, buffer := value.AppendString((*dst)[:0])
	*dst = buffer
	return unsafe.Slice(unsafe.StringData(text), len(text))
}
