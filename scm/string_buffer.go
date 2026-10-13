/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "io"
import "fmt"
import "unsafe"
import "unicode/utf8"
import "encoding/binary"
import "github.com/pierrec/lz4/v4"

const (
	StringCodecLZ4             = 2
	StringBufferBlockBytes     = 65536
	bufferLengthBits           = 24
	bufferLengthMask           = (1 << bufferLengthBits) - 1
	StringBufferMaxStringBytes = bufferLengthMask
)

// CompressedStringBuffer is owned by one immutable column generation. Handles
// retain this header (and therefore its payload) through a real GC pointer.
// No transaction, shard, decoded cache, or mutable encoder is retained here.
// Offsets address the logical byte stream.
// aux packs offset[32], byte length[24], then the tag[8].
type CompressedStringBuffer struct {
	codec     uint8
	data      []byte
	rawSize   uint32
	blocks    []uint32 // compressed block boundaries, including final sentinel
	rawBlocks []bool
}

func (b *CompressedStringBuffer) Codec() int       { return int(b.codec) }
func (b *CompressedStringBuffer) DecodedSize() int { return int(b.rawSize) }

// BlockDecodeBytes describes immutable codec work for an invocation-owned
// reader. Raw blocks need no decompression; the final block may be shorter.
func (b *CompressedStringBuffer) BlockDecodeBytes(block int) int {
	if b.rawBlocks[block] {
		return 0
	}
	return min(StringBufferBlockBytes, int(b.rawSize)-block*StringBufferBlockBytes)
}

// DecodeInto expands a complete LZ4 dictionary into caller-owned preparation
// memory. The same block decoder serves cursor walks and decoded cache fills.
func (b *CompressedStringBuffer) DecodeInto(dst []byte) {
	if b.codec != StringCodecLZ4 || len(dst) != int(b.rawSize) {
		panic("invalid full string buffer decode")
	}
	for block := range b.rawBlocks {
		start := block * StringBufferBlockBytes
		b.decodeBlock(block, dst[start:min(start+StringBufferBlockBytes, len(dst))])
	}
}

func (b *CompressedStringBuffer) decodeBlock(block int, dst []byte) {
	encoded := b.data[b.blocks[block]:b.blocks[block+1]]
	if b.rawBlocks[block] {
		if len(encoded) != len(dst) {
			panic("invalid raw string block")
		}
		copy(dst, encoded)
		return
	}
	n, err := lz4.UncompressBlock(encoded, dst)
	if err != nil || n != len(dst) {
		panic("invalid LZ4 string block")
	}
}
func (b *CompressedStringBuffer) ComputeSize() uint {
	return uint(unsafe.Sizeof(*b)) + uint(cap(b.data)) + uint(cap(b.blocks))*4 + uint(cap(b.rawBlocks))
}

// BuildCompressedStringBuffer consumes already-deduplicated dictionary entries.
// All allocations happen before generation publication.
func BuildCompressedStringBuffer(entries []string) (*CompressedStringBuffer, []uint32) {
	var total uint64
	for _, s := range entries {
		if len(s) > bufferLengthMask {
			panic("compressed string exceeds handle length limit")
		}
		total += uint64(len(s))
	}
	if total > uint64(^uint32(0)) {
		panic("compressed string buffer exceeds offset limit")
	}
	offsets := make([]uint32, len(entries))
	raw := make([]byte, 0, int(total))
	for i, text := range entries {
		offsets[i] = uint32(len(raw))
		raw = append(raw, text...)
	}
	return CompressStringBuffer(unsafe.String(unsafe.SliceData(raw), len(raw))), offsets
}

// CompressStringBuffer borrows an existing dictionary during private generation
// construction. It never retains the raw bytes or allocates another raw copy.
func CompressStringBuffer(dictionary string) *CompressedStringBuffer {
	if uint64(len(dictionary)) > uint64(^uint32(0)) {
		panic("compressed string buffer exceeds offset limit")
	}
	raw := unsafe.Slice(unsafe.StringData(dictionary), len(dictionary))
	b := &CompressedStringBuffer{codec: StringCodecLZ4, rawSize: uint32(len(raw))}
	var compressor lz4.Compressor
	encoded := make([]byte, lz4.CompressBlockBound(StringBufferBlockBytes))
	for start := 0; start < len(raw); start += StringBufferBlockBytes {
		block := raw[start:min(start+StringBufferBlockBytes, len(raw))]
		n, err := compressor.CompressBlock(block, encoded)
		if err != nil {
			panic(err)
		}
		b.blocks = append(b.blocks, uint32(len(b.data)))
		b.rawBlocks = append(b.rawBlocks, n == 0)
		if n == 0 {
			b.data = append(b.data, block...)
		} else {
			b.data = append(b.data, encoded[:n]...)
		}
	}
	b.blocks = append(b.blocks, uint32(len(b.data)))
	if uint64(len(b.data)) > uint64(^uint32(0)) {
		panic("encoded string buffer exceeds offset limit")
	}
	// Drop construction slack rather than charging retained encoder capacity.
	b.data = append([]byte(nil), b.data...)
	return b
}

func NewBufferString(b *CompressedStringBuffer, offset uint32, length int) Scmer {
	if length < 0 || length > bufferLengthMask {
		panic("invalid compressed string handle")
	}
	if length == 0 {
		return NewString("")
	}
	val := uint64(offset)<<24 | uint64(length)
	return Scmer{(*byte)(unsafe.Pointer(b)), makeAux(tagLZ4, val)}
}

func (s Scmer) IsBufferString() bool { return s.GetTag() == tagLZ4 }
func bufferOf(s Scmer) *CompressedStringBuffer {
	return (*CompressedStringBuffer)(unsafe.Pointer(s.ptr))
}

func BufferStringLength(s Scmer) int    { return bufferStringLen(s) }
func BufferStringOffset(s Scmer) uint32 { return bufferStringOffset(s) }
func bufferStringLen(s Scmer) int       { return int(auxVal(s.aux) & bufferLengthMask) }
func bufferStringOffset(s Scmer) uint32 { return uint32(auxVal(s.aux) >> 24) }

// TextScratch belongs exclusively to an operator/reader invocation. It is
// reused across rows and is never referenced by a Scmer or shared generation.
// The cache key prevents copied/backtracking cursors from borrowing stale bytes.
type TextScratch struct {
	data  [StringBufferBlockBytes]byte
	owner *CompressedStringBuffer
	block int
}

// TextCursor reads represented bytes/runes without creating a Go string.
// Its state is invocation-local; plain values do not need TextScratch.
type TextCursor struct {
	source              Scmer
	buffer              *CompressedStringBuffer
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
		c.offset = int(bufferStringOffset(s))
		c.length = bufferStringLen(s)
		return
	}
	var ok bool
	c.view, ok = makeOperatorView(s)
	if !ok {
		panic("TextCursor requires a string")
	}
	c.length = c.view.n
}

func (c *TextCursor) lz4Block(s *TextScratch) []byte {
	b := c.buffer
	block := (c.offset + c.pos) / StringBufferBlockBytes
	if block >= len(b.rawBlocks) {
		panic("LZ4 string offset out of bounds")
	}
	encoded := b.data[b.blocks[block]:b.blocks[block+1]]
	if b.rawBlocks[block] {
		return encoded
	}
	if s == nil {
		panic("LZ4 text cursor requires invocation scratch")
	}
	n := min(StringBufferBlockBytes, int(b.rawSize)-block*StringBufferBlockBytes)
	if s.owner != b || s.block != block {
		b.decodeBlock(block, s.data[:n])
		s.owner, s.block = b, block
	}
	return s.data[:n]
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
	var v byte
	if c.buffer == nil {
		v = c.view.at(c.pos)
	} else {
		v = c.lz4Block(scratch)[(c.offset+c.pos)%StringBufferBlockBytes]
	}
	c.pos++
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
	if c.buffer == nil && c.view.format == 0 && c.view.base64 == "" && c.pushCount == 0 {
		copy(dst[:n], c.view.data[c.pos:c.pos+n])
		c.pos += n
		return n
	}
	if c.pushCount != 0 || c.buffer == nil {
		for i := 0; i < n; i++ {
			dst[i], _ = c.NextByte(scratch)
		}
		return n
	}
	written := 0
	for written < n {
		src := c.lz4Block(scratch)[(c.offset+c.pos)%StringBufferBlockBytes:]
		count := min(n-written, len(src))
		copy(dst[written:], src[:count])
		c.pos += count
		written += count
	}
	return n
}

func (c *TextCursor) nextChunk(scratch *TextScratch) []byte {
	if c.pos >= c.length {
		return nil
	}
	if c.buffer == nil {
		if c.view.format == 0 && c.view.base64 == "" {
			return unsafe.Slice(unsafe.StringData(c.view.data[c.pos:]), c.length-c.pos)
		}
		c.single[0] = c.view.at(c.pos)
		return c.single[:]
	}
	chunk := c.lz4Block(scratch)[(c.offset+c.pos)%StringBufferBlockBytes:]
	return chunk[:min(len(chunk), c.length-c.pos)]
}

func (c *TextCursor) advanceChunk(n int) {
	c.pos += n
}

func bufferSubstring(s Scmer, start, end int) Scmer {
	if start < 0 || end < start || end > bufferStringLen(s) {
		panic("substr: slice bounds out of range")
	}
	if start == end {
		return NewString("")
	}
	b := bufferOf(s)
	offset := bufferStringOffset(s)
	return NewBufferString(b, offset+uint32(start), end-start)
}

func appendBufferString(dst []byte, s Scmer) (string, []byte) {
	start := len(dst)
	dst = append(dst, make([]byte, bufferStringLen(s))...)
	readLZ4String(dst[start:], s)
	return unsafe.String(unsafe.SliceData(dst[start:]), len(dst)-start), dst
}

func readLZ4String(dst []byte, s Scmer) {
	var scratch TextScratch
	var cursor TextCursor
	cursor.Reset(s)
	cursor.Read(dst, &scratch)
}

// This is MemCP's version-4 buffer payload, not the upstream encoder's ABI.
// LZ4 blocks are immutable wire data; no Go layout is saved.
func (b *CompressedStringBuffer) WriteTo(w io.Writer) {
	write := func(v any) {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(b.codec)
	write(b.rawSize)
	write(uint64(len(b.data)))
	if _, err := w.Write(b.data); err != nil {
		panic(err)
	}
	write(uint32(len(b.rawBlocks)))
	write(b.blocks)
	for _, raw := range b.rawBlocks {
		var flag uint8
		if raw {
			flag = 1
		}
		write(flag)
	}
}

func ReadCompressedStringBuffer(r io.Reader) *CompressedStringBuffer {
	b := &CompressedStringBuffer{}
	read := func(v any) {
		if err := binary.Read(r, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	read(&b.codec)
	read(&b.rawSize)
	var length uint64
	read(&length)
	if length > uint64(^uint32(0)) {
		panic("encoded string buffer exceeds offset limit")
	}
	b.data = make([]byte, int(length))
	if _, err := io.ReadFull(r, b.data); err != nil {
		panic(err)
	}
	switch b.codec {
	case StringCodecLZ4:
		var count uint32
		read(&count)
		if uint64(count) != (uint64(b.rawSize)+StringBufferBlockBytes-1)/StringBufferBlockBytes {
			panic("invalid LZ4 block count")
		}
		b.blocks = make([]uint32, int(count)+1)
		read(b.blocks)
		if b.blocks[0] != 0 || uint64(b.blocks[len(b.blocks)-1]) != length {
			panic("invalid LZ4 block extent")
		}
		b.rawBlocks = make([]bool, count)
		for i := range b.rawBlocks {
			var flag uint8
			read(&flag)
			if flag > 1 || b.blocks[i] > b.blocks[i+1] {
				panic("invalid LZ4 block metadata")
			}
			b.rawBlocks[i] = flag == 1
			if flag == 1 && int(b.blocks[i+1]-b.blocks[i]) != min(StringBufferBlockBytes, int(b.rawSize)-i*StringBufferBlockBytes) {
				panic("invalid raw LZ4 block extent")
			}
		}
	default:
		panic(fmt.Sprintf("unknown string buffer codec %d", b.codec))
	}
	return b
}
