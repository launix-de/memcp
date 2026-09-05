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
package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/launix-de/memcp/scm"
	"io"
	"math/bits"
	"unsafe"
)

/*
	StorageEnum: k-ary rANS entropy-coded columnar storage for low-cardinality
	columns (up to 8 distinct values including NULL).

	Compared to PFOR (StorageInt), rANS encodes each symbol with a cost
	proportional to -log2(probability). For a boolean column that is 99% false,
	PFOR uses 1 bit/elem, while rANS uses ~0.08 bits/elem — a 12x improvement.

	Storage layout:
	  data[]    — uint64 rANS-coded chunks (variable elements per chunk)
	  jumpL1[]  — uint32 absolute cumulative counts every stride chunks
	  jumpL2[]  — uint16 relative cumulative counts per chunk

	Access patterns:
	  With cache hint:  O(chunk_size) — skips binary search, decodes from chunk start
	  Random access:    O(log(chunks) + chunk_size) via binary search on jump index
	  With per-thread EnumDecodeCache: O(1) sequential via GetValueCached
*/

const enumBitShift = 8
const enumBitMask = ^uint64(0) >> (64 - enumBitShift) // 0xFF
const enumBitModulo = uint64(1) << enumBitShift       // 256
const enumMaxSymbols = 8

// enumMaxChunkElems bounds how many elements a single rANS chunk may hold.
// jumpL2 cumulative counts are stored as uint16, so a chunk (or a jumpL1
// group of chunks) must never account for more than 65535 elements. Highly
// skewed columns (near-zero entropy) can otherwise pack far more than 65535
// elements into one 64-bit buffer before it ever overflows on bit-width
// alone, silently wrapping jumpL2 and corrupting random access.
const enumMaxChunkElems = 65535

type StorageEnum struct {
	storageJITFunctions
	// rANS coded payload
	data []uint64 `jit:"immutable-after-finish"`

	// 2-level jump index
	jumpL1       []uint32 `jit:"immutable-after-finish"`
	jumpL2       []uint16 `jit:"immutable-after-finish"`
	jumpL1Stride int      `jit:"immutable-after-finish"`

	// symbol table
	values     [enumMaxSymbols]scm.Scmer  `jit:"immutable-after-finish"`
	k          uint8                      `jit:"immutable-after-finish"` // number of symbols (including NULL if present)
	thresholds [enumMaxSymbols - 1]uint64 `jit:"immutable-after-finish"`
	widths     [enumMaxSymbols]uint64     `jit:"immutable-after-finish"`
	invWidths  [enumMaxSymbols]uint64     `jit:"immutable-after-finish"`

	count uint64 `jit:"immutable-after-finish"`

	// scan-phase temporaries
	scanFreqs [enumMaxSymbols]uint64
	scanTotal uint64

	// build-phase temporaries: we need to reverse-buffer elements
	// because rANS encodes in reverse order
	buildBuf []scm.Scmer
}

func enumFastDivMod(n, d, inv uint64) (q, r uint64) {
	q, _ = bits.Mul64(n, inv)
	r = n - q*d
	if r >= d {
		q++
		r -= d
	}
	return
}

func (s *StorageEnum) String() string {
	return fmt.Sprintf("enum[%d]", s.k)
}

func (s *StorageEnum) ComputeSize() uint {
	var sz uint = 200 // struct overhead estimate
	sz += 8 * uint(len(s.data))
	sz += 4 * uint(len(s.jumpL1))
	sz += 2 * uint(len(s.jumpL2))
	for i := uint8(0); i < s.k; i++ {
		sz += scm.ComputeSize(s.values[i])
	}
	return sz
}

// --- rANS codec helpers ---

func (s *StorageEnum) symbolLo(idx int) uint64 {
	if idx == 0 {
		return 0
	}
	return s.thresholds[idx-1]
}

func (s *StorageEnum) findValue(val scm.Scmer) int {
	for i := uint8(0); i < s.k; i++ {
		// strict: NULL only matches NULL
		if val.IsNil() == s.values[i].IsNil() && (val.IsNil() || scm.Equal(s.values[i], val)) {
			return int(i)
		}
	}
	panic(fmt.Sprintf("StorageEnum: value %v not in symbol set", val))
}

func (s *StorageEnum) decodeSymbol(slice uint64) int {
	for i := uint8(0); i < s.k-1; i++ {
		if slice < s.thresholds[i] {
			return int(i)
		}
	}
	return int(s.k) - 1
}

func (s *StorageEnum) jumpCum(j int) int {
	g := j / s.jumpL1Stride
	base := uint32(0)
	if g > 0 {
		base = s.jumpL1[g-1]
	}
	return int(base) + int(s.jumpL2[j])
}

// chunkEnd returns the element count at which chunk j ends. This is normally
// just jumpCum(j), but for the last chunk in a file written by the
// pre-enumMaxChunkElems encoder, jumpCum can under-report due to uint16
// wraparound (see enumMaxChunkElems and findChunk). The last chunk always
// truly extends to s.count, so callers that cache a chunk boundary (e.g.
// GetValueCached's sequential fast path) must use this instead of jumpCum
// directly, or they silently fall back to the slow path for every remaining
// element instead of just once.
func (s *StorageEnum) chunkEnd(j int) int {
	end := s.jumpCum(j)
	if j == len(s.jumpL2)-1 && end < int(s.count) {
		return int(s.count)
	}
	return end
}

func (s *StorageEnum) decodeOne(buffer uint64) (scm.Scmer, uint64) {
	slice := buffer & enumBitMask
	symIdx := s.decodeSymbol(slice)
	width := s.widths[symIdx]
	return s.values[symIdx], (buffer>>enumBitShift)*width + slice - s.symbolLo(symIdx)
}

// --- ColumnStorage interface ---

func (s *StorageEnum) prepare() {
	s.scanFreqs = [enumMaxSymbols]uint64{}
	s.scanTotal = 0
	s.k = 0
}

func (s *StorageEnum) scan(i uint32, value scm.Scmer) {
	s.scanTotal++
	// find existing symbol (strict: NULL only matches NULL)
	for j := uint8(0); j < s.k; j++ {
		if value.IsNil() == s.values[j].IsNil() && (value.IsNil() || scm.Equal(s.values[j], value)) {
			s.scanFreqs[j]++
			return
		}
	}
	// new symbol
	if s.k < enumMaxSymbols {
		s.values[s.k] = value
		s.scanFreqs[s.k] = 1
		s.k++
	}
}

func (s *StorageEnum) proposeCompression(i uint32) ColumnStorage {
	return nil // terminal
}

func (s *StorageEnum) init(i uint32) {
	s.count = uint64(i)

	if s.k < 2 {
		// degenerate: 0 or 1 distinct values; pad to 2 symbols
		if s.k == 0 {
			s.values[0] = scm.NewNil()
			s.values[1] = scm.NewBool(false)
			s.scanFreqs[0] = s.scanTotal
			s.scanFreqs[1] = 0
		} else {
			// 1 symbol: add a dummy
			s.values[s.k] = scm.NewNil()
			s.scanFreqs[s.k] = 0
		}
		s.k = 2
	}

	// Build slot widths from frequencies
	k := int(s.k)
	total := uint64(0)
	for j := 0; j < k; j++ {
		total += s.scanFreqs[j]
	}
	if total == 0 {
		total = 1
	}

	slots := [enumMaxSymbols]uint64{}
	remaining := int(enumBitModulo) - k
	for j := 0; j < k; j++ {
		slots[j] = 1 // minimum 1
	}
	distributed := 0
	for j := 0; j < k; j++ {
		extra := int(s.scanFreqs[j]) * remaining / int(total)
		slots[j] += uint64(extra)
		distributed += extra
	}
	leftover := remaining - distributed
	if leftover > 0 {
		maxIdx := 0
		for j := 1; j < k; j++ {
			if s.scanFreqs[j] > s.scanFreqs[maxIdx] {
				maxIdx = j
			}
		}
		slots[maxIdx] += uint64(leftover)
	}

	// Build thresholds, widths, inverse table
	cum := uint64(0)
	for j := 0; j < k; j++ {
		s.widths[j] = slots[j]
		s.invWidths[j] = ^uint64(0) / slots[j]
		if j < k-1 {
			cum += slots[j]
			s.thresholds[j] = cum
		}
	}

	// Allocate build buffer (freed in finish)
	s.buildBuf = make([]scm.Scmer, i)
}

func (s *StorageEnum) build(i uint32, value scm.Scmer) {
	s.buildBuf[i] = value
}

func (s *StorageEnum) finish() {
	n := int(s.count)
	s.data = s.data[:0]
	var chunkSizes []int

	var buffer uint64
	bufferlen := 0

	// encode in reverse order (rANS requirement)
	for i := n - 1; i >= 0; i-- {
		symIdx := s.findValue(s.buildBuf[i])
		lo := s.symbolLo(symIdx)
		width := s.widths[symIdx]
		inv := s.invWidths[symIdx]

		bufferx, rest := enumFastDivMod(buffer, width, inv)
		if bufferx > ^uint64(0)>>enumBitShift || bufferlen >= enumMaxChunkElems {
			s.data = append(s.data, buffer)
			chunkSizes = append(chunkSizes, bufferlen)
			buffer = 0
			bufferlen = 0
			bufferx = 0
		}
		buffer = (bufferx << enumBitShift) + lo + rest
		bufferlen++
	}
	s.data = append(s.data, buffer)
	chunkSizes = append(chunkSizes, bufferlen)

	// Free build buffer
	s.buildBuf = nil

	// Build 2-level jump index
	numChunks := len(s.data)

	// Auto-tune stride
	maxCS := 0
	for _, cs := range chunkSizes {
		if cs > maxCS {
			maxCS = cs
		}
	}
	s.jumpL1Stride = 1
	for s.jumpL1Stride*2*maxCS <= 65535 {
		s.jumpL1Stride *= 2
	}
	if s.jumpL1Stride < 1 {
		s.jumpL1Stride = 1
	}

	numGroups := (numChunks + s.jumpL1Stride - 1) / s.jumpL1Stride
	s.jumpL1 = make([]uint32, numGroups)
	s.jumpL2 = make([]uint16, numChunks)

	cumAbs := uint32(0)
	groupBase := uint32(0)
	for j := 0; j < numChunks; j++ {
		cumAbs += uint32(chunkSizes[numChunks-1-j])
		s.jumpL2[j] = uint16(cumAbs - groupBase)
		if (j+1)%s.jumpL1Stride == 0 {
			g := (j + 1) / s.jumpL1Stride
			s.jumpL1[g-1] = cumAbs
			groupBase = cumAbs
		}
	}
	if numChunks%s.jumpL1Stride != 0 {
		s.jumpL1[numGroups-1] = cumAbs
	}
	s.storageJITFunctions.finish(s)
}

// EnumDecodeCache holds per-goroutine rANS decode state for O(1) sequential access.
// Allocate one per worker goroutine and pass to GetValueCached.
type EnumDecodeCache struct {
	fwdChunk int
	start    int
	pos      int
	buf      uint64
	valid    bool
}

// cachedEnumReader wraps a StorageEnum with a private EnumDecodeCache.
// Returned by StorageEnum.GetCachedReader(). Must not be shared between goroutines.
type cachedEnumReader struct {
	s     *StorageEnum
	cache EnumDecodeCache
}

func (r *cachedEnumReader) GetValue(i uint32) scm.Scmer {
	return r.s.GetValueCached(i, &r.cache)
}

// GetValueRange and GetValueMulti reuse this reader's persistent decode
// cache across the whole batch via GetValueCached, so a scan that gathers
// many rows through one reader gets the O(1)-amortized sequential/jump
// decode path GetValueCached already implements, instead of paying a fresh
// binary search (or a shared-cache reset) per row.
func (r *cachedEnumReader) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = r.s.GetValueCached(recid+k, &r.cache)
		idx += stride
	}
}

func (r *cachedEnumReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for _, recid := range recids {
		target[idx] = r.s.GetValueCached(recid, &r.cache)
		idx += stride
	}
}

func (s *StorageEnum) GetCachedReader() ColumnReader {
	if reader := s.storageJITFunctions.reader(nil); reader != nil {
		return reader
	}
	return &cachedEnumReader{s: s}
}

// GetValue is safe for concurrent use — it is fully read-only on the struct.
// Uses binary search + sequential decode from chunk start. For O(1) sequential
// access, use GetCachedReader() which returns a per-goroutine cached wrapper.
func (s *StorageEnum) GetValue(i uint32) scm.Scmer {
	if uint64(i) >= s.count {
		return scm.NewNil()
	}
	idx := int(i)
	fwdIdx := s.findChunk(idx)
	if fwdIdx >= len(s.data) {
		return scm.NewNil()
	}
	chunkStart := 0
	if fwdIdx > 0 {
		chunkStart = s.jumpCum(fwdIdx - 1)
	}
	dataIdx := len(s.data) - 1 - fwdIdx
	buffer := s.data[dataIdx]
	posInChunk := idx - chunkStart
	var result scm.Scmer
	for j := 0; j <= posInChunk; j++ {
		result, buffer = s.decodeOne(buffer)
	}
	return result
}

// GetValueCached provides O(1) sequential access using a per-goroutine cache.
// The cache must not be shared between goroutines.
func (s *StorageEnum) GetValueCached(i uint32, c *EnumDecodeCache) scm.Scmer {
	if uint64(i) >= s.count {
		return scm.NewNil()
	}
	idx := int(i)

	if c.valid && c.fwdChunk < len(s.jumpL2) && c.fwdChunk < len(s.data) {
		chunkEnd := s.chunkEnd(c.fwdChunk)
		// fast path: index is ahead of cache position in same chunk
		if idx >= c.start+c.pos && idx < chunkEnd {
			buffer := c.buf
			var result scm.Scmer
			target := idx - c.start
			for j := c.pos; j <= target; j++ {
				result, buffer = s.decodeOne(buffer)
			}
			c.pos = target + 1
			c.buf = buffer
			return result
		}
		// next chunk fast path
		if idx >= chunkEnd {
			nextFwd := c.fwdChunk + 1
			if nextFwd < len(s.jumpL2) && nextFwd < len(s.data) && idx < s.chunkEnd(nextFwd) {
				dataIdx := len(s.data) - 1 - nextFwd
				buffer := s.data[dataIdx]
				posInChunk := idx - chunkEnd
				var result scm.Scmer
				for j := 0; j <= posInChunk; j++ {
					result, buffer = s.decodeOne(buffer)
				}
				c.fwdChunk = nextFwd
				c.start = chunkEnd
				c.pos = posInChunk + 1
				c.buf = buffer
				return result
			}
		}
	} else if c.valid {
		c.valid = false
	}

	// Binary search fallback
	fwdIdx := s.findChunk(idx)
	if fwdIdx >= len(s.data) {
		return scm.NewNil()
	}
	chunkStart := 0
	if fwdIdx > 0 {
		chunkStart = s.jumpCum(fwdIdx - 1)
	}

	dataIdx := len(s.data) - 1 - fwdIdx
	buffer := s.data[dataIdx]
	posInChunk := idx - chunkStart
	var result scm.Scmer
	for j := 0; j <= posInChunk; j++ {
		result, buffer = s.decodeOne(buffer)
	}

	c.valid = true
	c.fwdChunk = fwdIdx
	c.start = chunkStart
	c.pos = posInChunk + 1
	c.buf = buffer
	return result
}

// GetValueRange and GetValueMulti decode a whole batch through a single
// local (stack-allocated, not shared-atomic) EnumDecodeCache, so the rANS
// chunk decode state — the expensive part of an enum read — is amortized
// across the batch via GetValueCached's O(1) sequential/jump fast paths
// instead of every element paying its own binary search from GetValue.
func (s *StorageEnum) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	var cache EnumDecodeCache
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = s.GetValueCached(recid+k, &cache)
		idx += stride
	}
}

func (s *StorageEnum) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	var cache EnumDecodeCache
	idx := 0
	for _, recid := range recids {
		target[idx] = s.GetValueCached(recid, &cache)
		idx += stride
	}
}

// findChunk returns the chunk index containing element idx via binary search.
func (s *StorageEnum) findChunk(idx int) int {
	lo, hi := 0, len(s.jumpL2)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s.jumpCum(mid) <= idx {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo >= len(s.jumpL2) && len(s.jumpL2) > 0 && idx < int(s.count) {
		// jumpL2 cumulative counts are uint16 and can wrap for files written
		// by the pre-enumMaxChunkElems encoder, which let one rANS chunk
		// (typically for a near-constant, low-entropy column) hold more than
		// 65535 elements. idx is still a genuinely valid row per the
		// untruncated s.count, so the search "falling off the end" means the
		// row lives in the last real chunk, not that it's out of range.
		return len(s.jumpL2) - 1
	}
	return lo
}

// --- Serialization ---
//
// StorageEnum binary layout (magic byte 40 consumed by shard loader):
//
//	[k uint8]              ← number of symbols (2..8)
//	[count uint64]
//	[jumpL1Stride uint32]
//	[dataLen uint64]
//	[l1Len uint64]
//	[l2Len uint64]
//	[scanFreqs: k × uint64]
//	[symbol values: k × (uint32 length + JSON bytes)]
//	[data: dataLen × uint64]
//	[jumpL1: l1Len × uint32]
//	[jumpL2: l2Len × uint16]
//
// Version history:
//
//	v0 (original, no version byte): layout as above.  The first byte after the
//	magic is k (uint8, always 2..8), so there is no safe location for an inline
//	version byte.  Format changes require a NEW magic byte in storages[]
//	(storage.go); keep magic 40 as a legacy reader forever.

func (s *StorageEnum) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d5 scm.JITValueDesc
	_ = d5
	var d6 scm.JITValueDesc
	_ = d6
	var d7 scm.JITValueDesc
	_ = d7
	var d8 scm.JITValueDesc
	_ = d8
	var d22 scm.JITValueDesc
	_ = d22
	var d23 scm.JITValueDesc
	_ = d23
	var d24 scm.JITValueDesc
	_ = d24
	var d25 scm.JITValueDesc
	_ = d25
	var d26 scm.JITValueDesc
	_ = d26
	var d27 scm.JITValueDesc
	_ = d27
	var d28 scm.JITValueDesc
	_ = d28
	var d29 scm.JITValueDesc
	_ = d29
	var d51 scm.JITValueDesc
	_ = d51
	var d52 scm.JITValueDesc
	_ = d52
	var d53 scm.JITValueDesc
	_ = d53
	var d54 scm.JITValueDesc
	_ = d54
	var d57 scm.JITValueDesc
	_ = d57
	var d60 scm.JITValueDesc
	_ = d60
	var d84 scm.JITValueDesc
	_ = d84
	var d85 scm.JITValueDesc
	_ = d85
	var d86 scm.JITValueDesc
	_ = d86
	var d88 scm.JITValueDesc
	_ = d88
	var d89 scm.JITValueDesc
	_ = d89
	var d90 scm.JITValueDesc
	_ = d90
	var d91 scm.JITValueDesc
	_ = d91
	var d92 scm.JITValueDesc
	_ = d92
	var d93 scm.JITValueDesc
	_ = d93
	var d94 scm.JITValueDesc
	_ = d94
	var d95 scm.JITValueDesc
	_ = d95
	var d96 scm.JITValueDesc
	_ = d96
	var d97 scm.JITValueDesc
	_ = d97
	var d99 scm.JITValueDesc
	_ = d99
	var d100 scm.JITValueDesc
	_ = d100
	var d101 scm.JITValueDesc
	_ = d101
	var d102 scm.JITValueDesc
	_ = d102
	var d103 scm.JITValueDesc
	_ = d103
	var d104 scm.JITValueDesc
	_ = d104
	var d105 scm.JITValueDesc
	_ = d105
	var d106 scm.JITValueDesc
	_ = d106
	var d109 scm.JITValueDesc
	_ = d109
	var d110 scm.JITValueDesc
	_ = d110
	var d111 scm.JITValueDesc
	_ = d111
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d164 scm.JITValueDesc
	_ = d164
	var d165 scm.JITValueDesc
	_ = d165
	var d166 scm.JITValueDesc
	_ = d166
	var d167 scm.JITValueDesc
	_ = d167
	var d168 scm.JITValueDesc
	_ = d168
	var d169 scm.JITValueDesc
	_ = d169
	var d170 scm.JITValueDesc
	_ = d170
	var d171 scm.JITValueDesc
	_ = d171
	var d172 scm.JITValueDesc
	_ = d172
	var d173 scm.JITValueDesc
	_ = d173
	var d174 scm.JITValueDesc
	_ = d174
	var d175 scm.JITValueDesc
	_ = d175
	var d176 scm.JITValueDesc
	_ = d176
	var d177 scm.JITValueDesc
	_ = d177
	var d179 scm.JITValueDesc
	_ = d179
	var d180 scm.JITValueDesc
	_ = d180
	var d181 scm.JITValueDesc
	_ = d181
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	standaloneFrame := ctx.BeginStandaloneFrame()
	var idxInt scm.JITValueDesc
	if idx.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
	} else if idx.Loc == scm.LocRegPair {
		ctx.FreeReg(idx.Reg)
		idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
		ctx.BindReg(idx.Reg2, &idxInt)
	} else {
		idxInt = idx
	}
	if idxInt.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
		defer ctx.UnprotectReg(idxPinnedReg)
	}
	phiBase0 := ctx.AllocStack(int32(64))
	d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	_ = d1
	d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	_ = d2
	d3 := scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
	ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(32))
	_ = d3
	d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
	_ = d4
	var bbs [10]scm.BBDescriptor
	bbs[6].PhiBase = int32(phiBase0) + int32(0)
	bbs[6].PhiCount = uint16(1)
	bbs[7].PhiBase = int32(phiBase0) + int32(16)
	bbs[7].PhiCount = uint16(3)
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	resultRegsProtected := result.Loc == scm.LocRegPair
	if resultRegsProtected {
		ctx.ProtectReg(result.Reg)
		ctx.ProtectReg(result.Reg2)
	}
	r0 := ctx.AllocReg()
	r1 := ctx.AllocRegExcept(r0)
	lbl0 := ctx.ReserveLabel()
	bbpos_0_0 := int32(-1)
	_ = bbpos_0_0
	lbl1 := ctx.ReserveLabel()
	_ = lbl1
	bbpos_0_1 := int32(-1)
	_ = bbpos_0_1
	lbl2 := ctx.ReserveLabel()
	_ = lbl2
	bbpos_0_2 := int32(-1)
	_ = bbpos_0_2
	lbl3 := ctx.ReserveLabel()
	_ = lbl3
	bbpos_0_3 := int32(-1)
	_ = bbpos_0_3
	lbl4 := ctx.ReserveLabel()
	_ = lbl4
	bbpos_0_4 := int32(-1)
	_ = bbpos_0_4
	lbl5 := ctx.ReserveLabel()
	_ = lbl5
	bbpos_0_5 := int32(-1)
	_ = bbpos_0_5
	lbl6 := ctx.ReserveLabel()
	_ = lbl6
	bbpos_0_6 := int32(-1)
	_ = bbpos_0_6
	lbl7 := ctx.ReserveLabel()
	_ = lbl7
	bbpos_0_7 := int32(-1)
	_ = bbpos_0_7
	lbl8 := ctx.ReserveLabel()
	_ = lbl8
	bbpos_0_8 := int32(-1)
	_ = bbpos_0_8
	lbl9 := ctx.ReserveLabel()
	_ = lbl9
	bbpos_0_9 := int32(-1)
	_ = bbpos_0_9
	lbl10 := ctx.ReserveLabel()
	_ = lbl10
	bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[0].VisitCount >= 0 {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
		}
		bbs[0].VisitCount++
		if ps.General {
			if bbs[0].Rendered {
				ctx.EmitJmp(lbl1)
				return result
			}
			bbs[0].Rendered = true
			bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_0 = bbs[0].Address
			ctx.MarkLabel(lbl1)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d5 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, idxInt.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d5)
		}
		var d6 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).count)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).count))
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMem(r3, thisptr.Reg, off)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d6)
		}
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d5, &d6)
		var d7 scm.JITValueDesc
		if d5.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d5.Imm.Int()) >= uint64(d6.Imm.Int()))}
		} else if d6.Loc == scm.LocImm {
			r4 := ctx.AllocRegExcept(d5.Reg)
			if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d5.Reg, int32(d6.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitCmpInt64(d5.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r4, scm.CondUnsignedAboveOrEqual)
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r4}
			ctx.BindReg(r4, &d7)
		} else if d5.Loc == scm.LocImm {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d6.Reg)
			ctx.EmitSetcc(r5, scm.CondUnsignedAboveOrEqual)
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
			ctx.BindReg(r5, &d7)
		} else {
			r6 := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitCmpInt64(d5.Reg, d6.Reg)
			ctx.EmitSetcc(r6, scm.CondUnsignedAboveOrEqual)
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
			ctx.BindReg(r6, &d7)
		}
		ctx.FreeDesc(&d5)
		ctx.FreeDesc(&d6)
		d8 = d7
		ctx.EnsureDesc(&d8)
		if d8.Loc != scm.LocImm && d8.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d8.Loc == scm.LocImm {
			if d8.Imm.Bool() {
				if ps.General {
				}
				ps9 := scm.PhiState{General: ps.General}
				ps9.OverlayValues = make([]scm.JITValueDesc, 9)
				ps9.OverlayValues[1] = d1
				ps9.OverlayValues[2] = d2
				ps9.OverlayValues[3] = d3
				ps9.OverlayValues[4] = d4
				ps9.OverlayValues[5] = d5
				ps9.OverlayValues[6] = d6
				ps9.OverlayValues[7] = d7
				ps9.OverlayValues[8] = d8
				return bbs[1].RenderPS(ps9)
			}
			if ps.General {
			}
			ps10 := scm.PhiState{General: ps.General}
			ps10.OverlayValues = make([]scm.JITValueDesc, 9)
			ps10.OverlayValues[1] = d1
			ps10.OverlayValues[2] = d2
			ps10.OverlayValues[3] = d3
			ps10.OverlayValues[4] = d4
			ps10.OverlayValues[5] = d5
			ps10.OverlayValues[6] = d6
			ps10.OverlayValues[7] = d7
			ps10.OverlayValues[8] = d8
			return bbs[2].RenderPS(ps10)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl11 := ctx.ReserveLabel()
		lbl12 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d8.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl11)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl3)
		ps11 := scm.PhiState{General: true}
		ps11.OverlayValues = make([]scm.JITValueDesc, 9)
		ps11.OverlayValues[1] = d1
		ps11.OverlayValues[2] = d2
		ps11.OverlayValues[3] = d3
		ps11.OverlayValues[4] = d4
		ps11.OverlayValues[5] = d5
		ps11.OverlayValues[6] = d6
		ps11.OverlayValues[7] = d7
		ps11.OverlayValues[8] = d8
		ps12 := scm.PhiState{General: true}
		ps12.OverlayValues = make([]scm.JITValueDesc, 9)
		ps12.OverlayValues[1] = d1
		ps12.OverlayValues[2] = d2
		ps12.OverlayValues[3] = d3
		ps12.OverlayValues[4] = d4
		ps12.OverlayValues[5] = d5
		ps12.OverlayValues[6] = d6
		ps12.OverlayValues[7] = d7
		ps12.OverlayValues[8] = d8
		snap13 := d1
		snap14 := d2
		snap15 := d3
		snap16 := d4
		snap17 := d5
		snap18 := d6
		snap19 := d7
		snap20 := d8
		alloc21 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps12)
		}
		ctx.RestoreAllocState(alloc21)
		d1 = snap13
		d2 = snap14
		d3 = snap15
		d4 = snap16
		d5 = snap17
		d6 = snap18
		d7 = snap19
		d8 = snap20
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps11)
		}
		return result
		ctx.FreeDesc(&d7)
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[1].VisitCount >= 0 {
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
		}
		bbs[1].VisitCount++
		if ps.General {
			if bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			bbs[1].Rendered = true
			bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_1 = bbs[1].Address
			ctx.MarkLabel(lbl2)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		ctx.ReclaimUntrackedRegs()
		d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d23 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d23)
		ctx.BindReg(r1, &d23)
		ctx.EnsureDesc(&d22)
		if d22.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d22, &d23)
		} else {
			switch d22.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d23, d22)
			case scm.TagInt:
				ctx.EmitMakeInt(d23, d22)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d23, d22)
			case scm.TagNil:
				ctx.EmitMakeNil(d23)
			default:
				ctx.EmitMovPairToResult(&d22, &d23)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[2].VisitCount >= 0 {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
		}
		bbs[2].VisitCount++
		if ps.General {
			if bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			bbs[2].Rendered = true
			bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_2 = bbs[2].Address
			ctx.MarkLabel(lbl3)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d24 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, idxInt.Reg)
			ctx.EmitShlRegImm8(r7, 32)
			ctx.EmitShrRegImm8(r7, 32)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d24)
		}
		ctx.StabilizeDescForControlFlow(&d24)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		if d24.Loc == scm.LocRegPair || d24.Loc == scm.LocStackPair || d24.Loc == scm.LocRegTriple || d24.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d24)
		d25 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).findChunk), []scm.JITValueDesc{thisptr, d24}, 1)
		d25.NoHeapPointer = true
		ctx.BindReg(d25.Reg, &d25)
		ctx.StabilizeDescForControlFlow(&d25)
		var d26 scm.JITValueDesc
		r8 := ctx.AllocReg()
		r9 := ctx.AllocRegExcept(r8)
		r10 := ctx.AllocRegExcept(r8, r9)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r8, uint64(dataPtr))
			ctx.EmitMovRegImm64(r9, uint64(sliceLen))
			ctx.EmitMovRegImm64(r10, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r8, thisptr.Reg, off)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off+16)
		}
		d26 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
		ctx.BindReg(r8, &d26)
		ctx.BindReg(r9, &d26)
		ctx.BindReg(r10, &d26)
		ctx.BindReg(r8, &d26)
		ctx.BindReg(r9, &d26)
		ctx.BindReg(r10, &d26)
		var d27 scm.JITValueDesc
		if d26.SliceSizeKnown {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d26.KnownSliceLen))}
		} else if d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d26.StackOff))}
		} else if d26.Loc == scm.LocStackTriple {
			d27 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d26.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d26)
			if d26.Loc == scm.LocRegPair || d26.Loc == scm.LocRegTriple {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg2, ID: 0}
			} else if d26.Loc == scm.LocReg {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d25)
		ctx.EnsureDesc(&d27)
		ctx.EnsureDescsTogether(&d25, &d27)
		var d28 scm.JITValueDesc
		if d25.Loc == scm.LocImm && d27.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d25.Imm.Int() >= d27.Imm.Int())}
		} else if d27.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d25.Reg)
			if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d25.Reg, int32(d27.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.EmitCmpInt64(d25.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r11, scm.CondSignedGreaterOrEqual)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r11}
			ctx.BindReg(r11, &d28)
		} else if d25.Loc == scm.LocImm {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d27.Reg)
			ctx.EmitSetcc(r12, scm.CondSignedGreaterOrEqual)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r12}
			ctx.BindReg(r12, &d28)
		} else {
			r13 := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitCmpInt64(d25.Reg, d27.Reg)
			ctx.EmitSetcc(r13, scm.CondSignedGreaterOrEqual)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r13}
			ctx.BindReg(r13, &d28)
		}
		ctx.FreeDesc(&d27)
		d29 = d28
		ctx.EnsureDesc(&d29)
		if d29.Loc != scm.LocImm && d29.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d29.Loc == scm.LocImm {
			if d29.Imm.Bool() {
				if ps.General {
				}
				ps30 := scm.PhiState{General: ps.General}
				ps30.OverlayValues = make([]scm.JITValueDesc, 30)
				ps30.OverlayValues[1] = d1
				ps30.OverlayValues[2] = d2
				ps30.OverlayValues[3] = d3
				ps30.OverlayValues[4] = d4
				ps30.OverlayValues[5] = d5
				ps30.OverlayValues[6] = d6
				ps30.OverlayValues[7] = d7
				ps30.OverlayValues[8] = d8
				ps30.OverlayValues[22] = d22
				ps30.OverlayValues[23] = d23
				ps30.OverlayValues[24] = d24
				ps30.OverlayValues[25] = d25
				ps30.OverlayValues[26] = d26
				ps30.OverlayValues[27] = d27
				ps30.OverlayValues[28] = d28
				ps30.OverlayValues[29] = d29
				return bbs[3].RenderPS(ps30)
			}
			if ps.General {
			}
			ps31 := scm.PhiState{General: ps.General}
			ps31.OverlayValues = make([]scm.JITValueDesc, 30)
			ps31.OverlayValues[1] = d1
			ps31.OverlayValues[2] = d2
			ps31.OverlayValues[3] = d3
			ps31.OverlayValues[4] = d4
			ps31.OverlayValues[5] = d5
			ps31.OverlayValues[6] = d6
			ps31.OverlayValues[7] = d7
			ps31.OverlayValues[8] = d8
			ps31.OverlayValues[22] = d22
			ps31.OverlayValues[23] = d23
			ps31.OverlayValues[24] = d24
			ps31.OverlayValues[25] = d25
			ps31.OverlayValues[26] = d26
			ps31.OverlayValues[27] = d27
			ps31.OverlayValues[28] = d28
			ps31.OverlayValues[29] = d29
			return bbs[4].RenderPS(ps31)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl13 := ctx.ReserveLabel()
		lbl14 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d29.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl13)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl5)
		ps32 := scm.PhiState{General: true}
		ps32.OverlayValues = make([]scm.JITValueDesc, 30)
		ps32.OverlayValues[1] = d1
		ps32.OverlayValues[2] = d2
		ps32.OverlayValues[3] = d3
		ps32.OverlayValues[4] = d4
		ps32.OverlayValues[5] = d5
		ps32.OverlayValues[6] = d6
		ps32.OverlayValues[7] = d7
		ps32.OverlayValues[8] = d8
		ps32.OverlayValues[22] = d22
		ps32.OverlayValues[23] = d23
		ps32.OverlayValues[24] = d24
		ps32.OverlayValues[25] = d25
		ps32.OverlayValues[26] = d26
		ps32.OverlayValues[27] = d27
		ps32.OverlayValues[28] = d28
		ps32.OverlayValues[29] = d29
		ps33 := scm.PhiState{General: true}
		ps33.OverlayValues = make([]scm.JITValueDesc, 30)
		ps33.OverlayValues[1] = d1
		ps33.OverlayValues[2] = d2
		ps33.OverlayValues[3] = d3
		ps33.OverlayValues[4] = d4
		ps33.OverlayValues[5] = d5
		ps33.OverlayValues[6] = d6
		ps33.OverlayValues[7] = d7
		ps33.OverlayValues[8] = d8
		ps33.OverlayValues[22] = d22
		ps33.OverlayValues[23] = d23
		ps33.OverlayValues[24] = d24
		ps33.OverlayValues[25] = d25
		ps33.OverlayValues[26] = d26
		ps33.OverlayValues[27] = d27
		ps33.OverlayValues[28] = d28
		ps33.OverlayValues[29] = d29
		snap34 := d1
		snap35 := d2
		snap36 := d3
		snap37 := d4
		snap38 := d5
		snap39 := d6
		snap40 := d7
		snap41 := d8
		snap42 := d22
		snap43 := d23
		snap44 := d24
		snap45 := d25
		snap46 := d26
		snap47 := d27
		snap48 := d28
		snap49 := d29
		alloc50 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps33)
		}
		ctx.RestoreAllocState(alloc50)
		d1 = snap34
		d2 = snap35
		d3 = snap36
		d4 = snap37
		d5 = snap38
		d6 = snap39
		d7 = snap40
		d8 = snap41
		d22 = snap42
		d23 = snap43
		d24 = snap44
		d25 = snap45
		d26 = snap46
		d27 = snap47
		d28 = snap48
		d29 = snap49
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps32)
		}
		return result
		ctx.FreeDesc(&d28)
		return result
	}
	bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[3].VisitCount >= 0 {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
		}
		bbs[3].VisitCount++
		if ps.General {
			if bbs[3].Rendered {
				ctx.EmitJmp(lbl4)
				return result
			}
			bbs[3].Rendered = true
			bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_3 = bbs[3].Address
			ctx.MarkLabel(lbl4)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		ctx.ReclaimUntrackedRegs()
		d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d52 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d52)
		ctx.BindReg(r1, &d52)
		ctx.EnsureDesc(&d51)
		if d51.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d51, &d52)
		} else {
			switch d51.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d52, d51)
			case scm.TagInt:
				ctx.EmitMakeInt(d52, d51)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d52, d51)
			case scm.TagNil:
				ctx.EmitMakeNil(d52)
			default:
				ctx.EmitMovPairToResult(&d51, &d52)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[4].VisitCount >= 0 {
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
		}
		bbs[4].VisitCount++
		if ps.General {
			if bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			bbs[4].Rendered = true
			bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_4 = bbs[4].Address
			ctx.MarkLabel(lbl5)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		var d53 scm.JITValueDesc
		if d25.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d25.Imm.Int() > 0)}
		} else {
			r14 := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitCmpRegImm32(d25.Reg, 0)
			ctx.EmitSetcc(r14, scm.CondSignedGreater)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r14}
			ctx.BindReg(r14, &d53)
		}
		d54 = d53
		ctx.EnsureDesc(&d54)
		if d54.Loc != scm.LocImm && d54.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d54.Loc == scm.LocImm {
			if d54.Imm.Bool() {
				if ps.General {
				}
				ps55 := scm.PhiState{General: ps.General}
				ps55.OverlayValues = make([]scm.JITValueDesc, 55)
				ps55.OverlayValues[1] = d1
				ps55.OverlayValues[2] = d2
				ps55.OverlayValues[3] = d3
				ps55.OverlayValues[4] = d4
				ps55.OverlayValues[5] = d5
				ps55.OverlayValues[6] = d6
				ps55.OverlayValues[7] = d7
				ps55.OverlayValues[8] = d8
				ps55.OverlayValues[22] = d22
				ps55.OverlayValues[23] = d23
				ps55.OverlayValues[24] = d24
				ps55.OverlayValues[25] = d25
				ps55.OverlayValues[26] = d26
				ps55.OverlayValues[27] = d27
				ps55.OverlayValues[28] = d28
				ps55.OverlayValues[29] = d29
				ps55.OverlayValues[51] = d51
				ps55.OverlayValues[52] = d52
				ps55.OverlayValues[53] = d53
				ps55.OverlayValues[54] = d54
				return bbs[5].RenderPS(ps55)
			}
			if ps.General {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
			}
			ps56 := scm.PhiState{General: ps.General}
			ps56.OverlayValues = make([]scm.JITValueDesc, 55)
			ps56.OverlayValues[1] = d1
			ps56.OverlayValues[2] = d2
			ps56.OverlayValues[3] = d3
			ps56.OverlayValues[4] = d4
			ps56.OverlayValues[5] = d5
			ps56.OverlayValues[6] = d6
			ps56.OverlayValues[7] = d7
			ps56.OverlayValues[8] = d8
			ps56.OverlayValues[22] = d22
			ps56.OverlayValues[23] = d23
			ps56.OverlayValues[24] = d24
			ps56.OverlayValues[25] = d25
			ps56.OverlayValues[26] = d26
			ps56.OverlayValues[27] = d27
			ps56.OverlayValues[28] = d28
			ps56.OverlayValues[29] = d29
			ps56.OverlayValues[51] = d51
			ps56.OverlayValues[52] = d52
			ps56.OverlayValues[53] = d53
			ps56.OverlayValues[54] = d54
			ps56.PhiValues = make([]scm.JITValueDesc, 1)
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps56.PhiValues[0] = d57
			return bbs[6].RenderPS(ps56)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl15 := ctx.ReserveLabel()
		lbl16 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d54.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl15)
		ctx.EmitJmp(lbl16)
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl6)
		ctx.MarkLabel(lbl16)
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
		ctx.EmitJmp(lbl7)
		ps58 := scm.PhiState{General: true}
		ps58.OverlayValues = make([]scm.JITValueDesc, 58)
		ps58.OverlayValues[1] = d1
		ps58.OverlayValues[2] = d2
		ps58.OverlayValues[3] = d3
		ps58.OverlayValues[4] = d4
		ps58.OverlayValues[5] = d5
		ps58.OverlayValues[6] = d6
		ps58.OverlayValues[7] = d7
		ps58.OverlayValues[8] = d8
		ps58.OverlayValues[22] = d22
		ps58.OverlayValues[23] = d23
		ps58.OverlayValues[24] = d24
		ps58.OverlayValues[25] = d25
		ps58.OverlayValues[26] = d26
		ps58.OverlayValues[27] = d27
		ps58.OverlayValues[28] = d28
		ps58.OverlayValues[29] = d29
		ps58.OverlayValues[51] = d51
		ps58.OverlayValues[52] = d52
		ps58.OverlayValues[53] = d53
		ps58.OverlayValues[54] = d54
		ps58.OverlayValues[57] = d57
		ps59 := scm.PhiState{General: true}
		ps59.OverlayValues = make([]scm.JITValueDesc, 58)
		ps59.OverlayValues[1] = d1
		ps59.OverlayValues[2] = d2
		ps59.OverlayValues[3] = d3
		ps59.OverlayValues[4] = d4
		ps59.OverlayValues[5] = d5
		ps59.OverlayValues[6] = d6
		ps59.OverlayValues[7] = d7
		ps59.OverlayValues[8] = d8
		ps59.OverlayValues[22] = d22
		ps59.OverlayValues[23] = d23
		ps59.OverlayValues[24] = d24
		ps59.OverlayValues[25] = d25
		ps59.OverlayValues[26] = d26
		ps59.OverlayValues[27] = d27
		ps59.OverlayValues[28] = d28
		ps59.OverlayValues[29] = d29
		ps59.OverlayValues[51] = d51
		ps59.OverlayValues[52] = d52
		ps59.OverlayValues[53] = d53
		ps59.OverlayValues[54] = d54
		ps59.OverlayValues[57] = d57
		ps59.PhiValues = make([]scm.JITValueDesc, 1)
		d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps59.PhiValues[0] = d60
		snap61 := d1
		snap62 := d2
		snap63 := d3
		snap64 := d4
		snap65 := d5
		snap66 := d6
		snap67 := d7
		snap68 := d8
		snap69 := d22
		snap70 := d23
		snap71 := d24
		snap72 := d25
		snap73 := d26
		snap74 := d27
		snap75 := d28
		snap76 := d29
		snap77 := d51
		snap78 := d52
		snap79 := d53
		snap80 := d54
		snap81 := d57
		snap82 := d60
		alloc83 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps59)
		}
		ctx.RestoreAllocState(alloc83)
		d1 = snap61
		d2 = snap62
		d3 = snap63
		d4 = snap64
		d5 = snap65
		d6 = snap66
		d7 = snap67
		d8 = snap68
		d22 = snap69
		d23 = snap70
		d24 = snap71
		d25 = snap72
		d26 = snap73
		d27 = snap74
		d28 = snap75
		d29 = snap76
		d51 = snap77
		d52 = snap78
		d53 = snap79
		d54 = snap80
		d57 = snap81
		d60 = snap82
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps58)
		}
		return result
		ctx.FreeDesc(&d53)
		return result
	}
	bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[5].VisitCount >= 0 {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
		}
		bbs[5].VisitCount++
		if ps.General {
			if bbs[5].Rendered {
				ctx.EmitJmp(lbl6)
				return result
			}
			bbs[5].Rendered = true
			bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_5 = bbs[5].Address
			ctx.MarkLabel(lbl6)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.EnsureDesc(&d25)
		var d84 scm.JITValueDesc
		if d25.Loc == scm.LocImm {
			d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(scratch, d25.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d84)
		}
		if d84.Loc == scm.LocReg && d25.Loc == scm.LocReg && d84.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d84)
		ctx.EnsureDesc(&d84)
		if d84.Loc == scm.LocRegPair || d84.Loc == scm.LocStackPair || d84.Loc == scm.LocRegTriple || d84.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d84)
		d85 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).jumpCum), []scm.JITValueDesc{thisptr, d84}, 1)
		d85.NoHeapPointer = true
		ctx.BindReg(d85.Reg, &d85)
		ctx.StabilizeDescForControlFlow(&d85)
		ctx.FreeDesc(&d84)
		if ps.General {
			ctx.SyncDesc(&d85)
			if d85.Loc == scm.LocReg {
				ctx.ProtectReg(d85.Reg)
			} else if d85.Loc == scm.LocRegPair {
				ctx.ProtectReg(d85.Reg)
				ctx.ProtectReg(d85.Reg2)
			}
			d86 = d85
			if d86.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d86)
			ctx.EmitStoreToStack(d86, int32(bbs[6].PhiBase)+int32(0))
			if d85.Loc == scm.LocReg {
				ctx.UnprotectReg(d85.Reg)
			} else if d85.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d85.Reg)
				ctx.UnprotectReg(d85.Reg2)
			}
		}
		ps87 := scm.PhiState{General: ps.General}
		ps87.OverlayValues = make([]scm.JITValueDesc, 87)
		ps87.OverlayValues[1] = d1
		ps87.OverlayValues[2] = d2
		ps87.OverlayValues[3] = d3
		ps87.OverlayValues[4] = d4
		ps87.OverlayValues[5] = d5
		ps87.OverlayValues[6] = d6
		ps87.OverlayValues[7] = d7
		ps87.OverlayValues[8] = d8
		ps87.OverlayValues[22] = d22
		ps87.OverlayValues[23] = d23
		ps87.OverlayValues[24] = d24
		ps87.OverlayValues[25] = d25
		ps87.OverlayValues[26] = d26
		ps87.OverlayValues[27] = d27
		ps87.OverlayValues[28] = d28
		ps87.OverlayValues[29] = d29
		ps87.OverlayValues[51] = d51
		ps87.OverlayValues[52] = d52
		ps87.OverlayValues[53] = d53
		ps87.OverlayValues[54] = d54
		ps87.OverlayValues[57] = d57
		ps87.OverlayValues[60] = d60
		ps87.OverlayValues[84] = d84
		ps87.OverlayValues[85] = d85
		ps87.OverlayValues[86] = d86
		ps87.PhiValues = make([]scm.JITValueDesc, 1)
		d88 = d85
		ps87.PhiValues[0] = d88
		if ps87.General && bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		return bbs[6].RenderPS(ps87)
		return result
	}
	bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d89 := ps.PhiValues[0]
				ctx.EnsureDesc(&d89)
				ctx.EmitStoreToStack(d89, int32(bbs[6].PhiBase)+int32(0))
			}
			if bbs[6].VisitCount >= 0 {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
		}
		bbs[6].VisitCount++
		if ps.General {
			if bbs[6].Rendered {
				ctx.EmitJmp(lbl7)
				return result
			}
			bbs[6].Rendered = true
			bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_6 = bbs[6].Address
			ctx.MarkLabel(lbl7)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
			d89 = ps.OverlayValues[89]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d1 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		var d90 scm.JITValueDesc
		r15 := ctx.AllocReg()
		r16 := ctx.AllocRegExcept(r15)
		r17 := ctx.AllocRegExcept(r15, r16)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r15, uint64(dataPtr))
			ctx.EmitMovRegImm64(r16, uint64(sliceLen))
			ctx.EmitMovRegImm64(r17, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r15, thisptr.Reg, off)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off+16)
		}
		d90 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r15, Reg2: r16, Reg3: r17}
		ctx.BindReg(r15, &d90)
		ctx.BindReg(r16, &d90)
		ctx.BindReg(r17, &d90)
		ctx.BindReg(r15, &d90)
		ctx.BindReg(r16, &d90)
		ctx.BindReg(r17, &d90)
		var d91 scm.JITValueDesc
		if d90.SliceSizeKnown {
			d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d90.KnownSliceLen))}
		} else if d90.Loc == scm.LocImm {
			d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d90.StackOff))}
		} else if d90.Loc == scm.LocStackTriple {
			d91 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d90.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d90)
			if d90.Loc == scm.LocRegPair || d90.Loc == scm.LocRegTriple {
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d90.Reg2, ID: 0}
			} else if d90.Loc == scm.LocReg {
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d90.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d91)
		ctx.EnsureDesc(&d91)
		var d92 scm.JITValueDesc
		if d91.Loc == scm.LocImm {
			d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d91.Reg)
			ctx.EmitMovRegReg(scratch, d91.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d92)
		}
		if d92.Loc == scm.LocReg && d91.Loc == scm.LocReg && d92.Reg == d91.Reg {
			ctx.TransferReg(d91.Reg)
			d91.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d91)
		ctx.EnsureDesc(&d92)
		ctx.EnsureDesc(&d25)
		ctx.EnsureDescsTogether(&d92, &d25)
		var d93 scm.JITValueDesc
		if d92.Loc == scm.LocImm && d25.Loc == scm.LocImm {
			d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() - d25.Imm.Int())}
		} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
			r18 := ctx.AllocRegExcept(d92.Reg)
			ctx.EmitMovRegReg(r18, d92.Reg)
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d93)
		} else if d92.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d92.Imm.Int()))
			ctx.EmitSubInt64(scratch, d25.Reg)
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d93)
		} else if d25.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d92.Reg)
			ctx.EmitMovRegReg(scratch, d92.Reg)
			if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d93)
		} else {
			r19 := ctx.AllocRegExcept(d92.Reg, d25.Reg)
			ctx.EmitMovRegReg(r19, d92.Reg)
			ctx.EmitSubInt64(r19, d25.Reg)
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d93)
		}
		if d93.Loc == scm.LocReg && d92.Loc == scm.LocReg && d93.Reg == d92.Reg {
			ctx.TransferReg(d92.Reg)
			d92.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d92)
		ctx.EnsureDesc(&d93)
		d95 = ctx.EmitSliceElementAddress(&d90, &d93, 8)
		ctx.EnsureDesc(&d95)
		ctx.EmitMovRegMem(d95.Reg, d95.Reg, 0)
		d94 = d95
		d94.Type = scm.TagInt
		ctx.FreeDesc(&d93)
		ctx.StabilizeDescForControlFlow(&d94)
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDescsTogether(&d24, &d1)
		var d96 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d1.Loc == scm.LocImm {
			d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() - d1.Imm.Int())}
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			r20 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r20, d24.Reg)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d96)
		} else if d24.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1.Reg)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d96)
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(scratch, d24.Reg)
			if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d96)
		} else {
			r21 := ctx.AllocRegExcept(d24.Reg, d1.Reg)
			ctx.EmitMovRegReg(r21, d24.Reg)
			ctx.EmitSubInt64(r21, d1.Reg)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d96)
		}
		if d96.Loc == scm.LocReg && d24.Loc == scm.LocReg && d96.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d96)
		ctx.FreeDesc(&d1)
		if ps.General {
			ctx.SyncDesc(&d94)
			if d94.Loc == scm.LocReg {
				ctx.ProtectReg(d94.Reg)
			} else if d94.Loc == scm.LocRegPair {
				ctx.ProtectReg(d94.Reg)
				ctx.ProtectReg(d94.Reg2)
			}
			d97 = d94
			if d97.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d97)
			ctx.EmitStoreToStack(d97, int32(bbs[7].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(32))
			if d94.Loc == scm.LocReg {
				ctx.UnprotectReg(d94.Reg)
			} else if d94.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d94.Reg)
				ctx.UnprotectReg(d94.Reg2)
			}
		}
		ps98 := scm.PhiState{General: ps.General}
		ps98.OverlayValues = make([]scm.JITValueDesc, 98)
		ps98.OverlayValues[1] = d1
		ps98.OverlayValues[2] = d2
		ps98.OverlayValues[3] = d3
		ps98.OverlayValues[4] = d4
		ps98.OverlayValues[5] = d5
		ps98.OverlayValues[6] = d6
		ps98.OverlayValues[7] = d7
		ps98.OverlayValues[8] = d8
		ps98.OverlayValues[22] = d22
		ps98.OverlayValues[23] = d23
		ps98.OverlayValues[24] = d24
		ps98.OverlayValues[25] = d25
		ps98.OverlayValues[26] = d26
		ps98.OverlayValues[27] = d27
		ps98.OverlayValues[28] = d28
		ps98.OverlayValues[29] = d29
		ps98.OverlayValues[51] = d51
		ps98.OverlayValues[52] = d52
		ps98.OverlayValues[53] = d53
		ps98.OverlayValues[54] = d54
		ps98.OverlayValues[57] = d57
		ps98.OverlayValues[60] = d60
		ps98.OverlayValues[84] = d84
		ps98.OverlayValues[85] = d85
		ps98.OverlayValues[86] = d86
		ps98.OverlayValues[88] = d88
		ps98.OverlayValues[89] = d89
		ps98.OverlayValues[90] = d90
		ps98.OverlayValues[91] = d91
		ps98.OverlayValues[92] = d92
		ps98.OverlayValues[93] = d93
		ps98.OverlayValues[94] = d94
		ps98.OverlayValues[95] = d95
		ps98.OverlayValues[96] = d96
		ps98.OverlayValues[97] = d97
		ps98.PhiValues = make([]scm.JITValueDesc, 3)
		d99 = d94
		ps98.PhiValues[0] = d99
		d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ps98.PhiValues[1] = d100
		d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps98.PhiValues[2] = d101
		if ps98.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps98)
		return result
	}
	bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d102 := ps.PhiValues[0]
				ctx.EnsureDesc(&d102)
				ctx.EmitStoreToStack(d102, int32(bbs[7].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d103 := ps.PhiValues[1]
				ctx.EnsureDesc(&d103)
				ctx.EmitStoreScmerToStack(d103, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d104 := ps.PhiValues[2]
				ctx.EnsureDesc(&d104)
				ctx.EmitStoreToStack(d104, int32(bbs[7].PhiBase)+int32(32))
			}
			if bbs[7].VisitCount >= 0 {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
		}
		bbs[7].VisitCount++
		if ps.General {
			if bbs[7].Rendered {
				ctx.EmitJmp(lbl8)
				return result
			}
			bbs[7].Rendered = true
			bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_7 = bbs[7].Address
			ctx.MarkLabel(lbl8)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
			d89 = ps.OverlayValues[89]
		}
		if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
			d90 = ps.OverlayValues[90]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
			d92 = ps.OverlayValues[92]
		}
		if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
			d93 = ps.OverlayValues[93]
		}
		if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
			d94 = ps.OverlayValues[94]
		}
		if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
			d95 = ps.OverlayValues[95]
		}
		if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
			d96 = ps.OverlayValues[96]
		}
		if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
			d97 = ps.OverlayValues[97]
		}
		if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
			d99 = ps.OverlayValues[99]
		}
		if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
			d100 = ps.OverlayValues[100]
		}
		if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
			d101 = ps.OverlayValues[101]
		}
		if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
			d102 = ps.OverlayValues[102]
		}
		if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
			d103 = ps.OverlayValues[103]
		}
		if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
			d104 = ps.OverlayValues[104]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d2 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d3 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d4 = ps.PhiValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d2)
		ctx.StabilizeDescForControlFlow(&d3)
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d96)
		ctx.EnsureDescsTogether(&d4, &d96)
		var d105 scm.JITValueDesc
		if d4.Loc == scm.LocImm && d96.Loc == scm.LocImm {
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d4.Imm.Int() <= d96.Imm.Int())}
		} else if d96.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d4.Reg)
			if d96.Imm.Int() >= -2147483648 && d96.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d4.Reg, int32(d96.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d96.Imm.Int()))
				ctx.EmitCmpInt64(d4.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r22, scm.CondSignedLessOrEqual)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r22}
			ctx.BindReg(r22, &d105)
		} else if d4.Loc == scm.LocImm {
			r23 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d96.Reg)
			ctx.EmitSetcc(r23, scm.CondSignedLessOrEqual)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r23}
			ctx.BindReg(r23, &d105)
		} else {
			r24 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpInt64(d4.Reg, d96.Reg)
			ctx.EmitSetcc(r24, scm.CondSignedLessOrEqual)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r24}
			ctx.BindReg(r24, &d105)
		}
		d106 = d105
		ctx.EnsureDesc(&d106)
		if d106.Loc != scm.LocImm && d106.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d106.Loc == scm.LocImm {
			if d106.Imm.Bool() {
				if ps.General {
				}
				ps107 := scm.PhiState{General: ps.General}
				ps107.OverlayValues = make([]scm.JITValueDesc, 107)
				ps107.OverlayValues[1] = d1
				ps107.OverlayValues[2] = d2
				ps107.OverlayValues[3] = d3
				ps107.OverlayValues[4] = d4
				ps107.OverlayValues[5] = d5
				ps107.OverlayValues[6] = d6
				ps107.OverlayValues[7] = d7
				ps107.OverlayValues[8] = d8
				ps107.OverlayValues[22] = d22
				ps107.OverlayValues[23] = d23
				ps107.OverlayValues[24] = d24
				ps107.OverlayValues[25] = d25
				ps107.OverlayValues[26] = d26
				ps107.OverlayValues[27] = d27
				ps107.OverlayValues[28] = d28
				ps107.OverlayValues[29] = d29
				ps107.OverlayValues[51] = d51
				ps107.OverlayValues[52] = d52
				ps107.OverlayValues[53] = d53
				ps107.OverlayValues[54] = d54
				ps107.OverlayValues[57] = d57
				ps107.OverlayValues[60] = d60
				ps107.OverlayValues[84] = d84
				ps107.OverlayValues[85] = d85
				ps107.OverlayValues[86] = d86
				ps107.OverlayValues[88] = d88
				ps107.OverlayValues[89] = d89
				ps107.OverlayValues[90] = d90
				ps107.OverlayValues[91] = d91
				ps107.OverlayValues[92] = d92
				ps107.OverlayValues[93] = d93
				ps107.OverlayValues[94] = d94
				ps107.OverlayValues[95] = d95
				ps107.OverlayValues[96] = d96
				ps107.OverlayValues[97] = d97
				ps107.OverlayValues[99] = d99
				ps107.OverlayValues[100] = d100
				ps107.OverlayValues[101] = d101
				ps107.OverlayValues[102] = d102
				ps107.OverlayValues[103] = d103
				ps107.OverlayValues[104] = d104
				ps107.OverlayValues[105] = d105
				ps107.OverlayValues[106] = d106
				return bbs[8].RenderPS(ps107)
			}
			if ps.General {
			}
			ps108 := scm.PhiState{General: ps.General}
			ps108.OverlayValues = make([]scm.JITValueDesc, 107)
			ps108.OverlayValues[1] = d1
			ps108.OverlayValues[2] = d2
			ps108.OverlayValues[3] = d3
			ps108.OverlayValues[4] = d4
			ps108.OverlayValues[5] = d5
			ps108.OverlayValues[6] = d6
			ps108.OverlayValues[7] = d7
			ps108.OverlayValues[8] = d8
			ps108.OverlayValues[22] = d22
			ps108.OverlayValues[23] = d23
			ps108.OverlayValues[24] = d24
			ps108.OverlayValues[25] = d25
			ps108.OverlayValues[26] = d26
			ps108.OverlayValues[27] = d27
			ps108.OverlayValues[28] = d28
			ps108.OverlayValues[29] = d29
			ps108.OverlayValues[51] = d51
			ps108.OverlayValues[52] = d52
			ps108.OverlayValues[53] = d53
			ps108.OverlayValues[54] = d54
			ps108.OverlayValues[57] = d57
			ps108.OverlayValues[60] = d60
			ps108.OverlayValues[84] = d84
			ps108.OverlayValues[85] = d85
			ps108.OverlayValues[86] = d86
			ps108.OverlayValues[88] = d88
			ps108.OverlayValues[89] = d89
			ps108.OverlayValues[90] = d90
			ps108.OverlayValues[91] = d91
			ps108.OverlayValues[92] = d92
			ps108.OverlayValues[93] = d93
			ps108.OverlayValues[94] = d94
			ps108.OverlayValues[95] = d95
			ps108.OverlayValues[96] = d96
			ps108.OverlayValues[97] = d97
			ps108.OverlayValues[99] = d99
			ps108.OverlayValues[100] = d100
			ps108.OverlayValues[101] = d101
			ps108.OverlayValues[102] = d102
			ps108.OverlayValues[103] = d103
			ps108.OverlayValues[104] = d104
			ps108.OverlayValues[105] = d105
			ps108.OverlayValues[106] = d106
			return bbs[9].RenderPS(ps108)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d109 := ps.PhiValues[0]
				ctx.EnsureDesc(&d109)
				ctx.EmitStoreToStack(d109, int32(bbs[7].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d110 := ps.PhiValues[1]
				ctx.EnsureDesc(&d110)
				ctx.EmitStoreScmerToStack(d110, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d111 := ps.PhiValues[2]
				ctx.EnsureDesc(&d111)
				ctx.EmitStoreToStack(d111, int32(bbs[7].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		lbl17 := ctx.ReserveLabel()
		lbl18 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d106.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl17)
		ctx.EmitJmp(lbl18)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl18)
		ctx.EmitJmp(lbl10)
		ps112 := scm.PhiState{General: true}
		ps112.OverlayValues = make([]scm.JITValueDesc, 112)
		ps112.OverlayValues[1] = d1
		ps112.OverlayValues[2] = d2
		ps112.OverlayValues[3] = d3
		ps112.OverlayValues[4] = d4
		ps112.OverlayValues[5] = d5
		ps112.OverlayValues[6] = d6
		ps112.OverlayValues[7] = d7
		ps112.OverlayValues[8] = d8
		ps112.OverlayValues[22] = d22
		ps112.OverlayValues[23] = d23
		ps112.OverlayValues[24] = d24
		ps112.OverlayValues[25] = d25
		ps112.OverlayValues[26] = d26
		ps112.OverlayValues[27] = d27
		ps112.OverlayValues[28] = d28
		ps112.OverlayValues[29] = d29
		ps112.OverlayValues[51] = d51
		ps112.OverlayValues[52] = d52
		ps112.OverlayValues[53] = d53
		ps112.OverlayValues[54] = d54
		ps112.OverlayValues[57] = d57
		ps112.OverlayValues[60] = d60
		ps112.OverlayValues[84] = d84
		ps112.OverlayValues[85] = d85
		ps112.OverlayValues[86] = d86
		ps112.OverlayValues[88] = d88
		ps112.OverlayValues[89] = d89
		ps112.OverlayValues[90] = d90
		ps112.OverlayValues[91] = d91
		ps112.OverlayValues[92] = d92
		ps112.OverlayValues[93] = d93
		ps112.OverlayValues[94] = d94
		ps112.OverlayValues[95] = d95
		ps112.OverlayValues[96] = d96
		ps112.OverlayValues[97] = d97
		ps112.OverlayValues[99] = d99
		ps112.OverlayValues[100] = d100
		ps112.OverlayValues[101] = d101
		ps112.OverlayValues[102] = d102
		ps112.OverlayValues[103] = d103
		ps112.OverlayValues[104] = d104
		ps112.OverlayValues[105] = d105
		ps112.OverlayValues[106] = d106
		ps112.OverlayValues[109] = d109
		ps112.OverlayValues[110] = d110
		ps112.OverlayValues[111] = d111
		ps113 := scm.PhiState{General: true}
		ps113.OverlayValues = make([]scm.JITValueDesc, 112)
		ps113.OverlayValues[1] = d1
		ps113.OverlayValues[2] = d2
		ps113.OverlayValues[3] = d3
		ps113.OverlayValues[4] = d4
		ps113.OverlayValues[5] = d5
		ps113.OverlayValues[6] = d6
		ps113.OverlayValues[7] = d7
		ps113.OverlayValues[8] = d8
		ps113.OverlayValues[22] = d22
		ps113.OverlayValues[23] = d23
		ps113.OverlayValues[24] = d24
		ps113.OverlayValues[25] = d25
		ps113.OverlayValues[26] = d26
		ps113.OverlayValues[27] = d27
		ps113.OverlayValues[28] = d28
		ps113.OverlayValues[29] = d29
		ps113.OverlayValues[51] = d51
		ps113.OverlayValues[52] = d52
		ps113.OverlayValues[53] = d53
		ps113.OverlayValues[54] = d54
		ps113.OverlayValues[57] = d57
		ps113.OverlayValues[60] = d60
		ps113.OverlayValues[84] = d84
		ps113.OverlayValues[85] = d85
		ps113.OverlayValues[86] = d86
		ps113.OverlayValues[88] = d88
		ps113.OverlayValues[89] = d89
		ps113.OverlayValues[90] = d90
		ps113.OverlayValues[91] = d91
		ps113.OverlayValues[92] = d92
		ps113.OverlayValues[93] = d93
		ps113.OverlayValues[94] = d94
		ps113.OverlayValues[95] = d95
		ps113.OverlayValues[96] = d96
		ps113.OverlayValues[97] = d97
		ps113.OverlayValues[99] = d99
		ps113.OverlayValues[100] = d100
		ps113.OverlayValues[101] = d101
		ps113.OverlayValues[102] = d102
		ps113.OverlayValues[103] = d103
		ps113.OverlayValues[104] = d104
		ps113.OverlayValues[105] = d105
		ps113.OverlayValues[106] = d106
		ps113.OverlayValues[109] = d109
		ps113.OverlayValues[110] = d110
		ps113.OverlayValues[111] = d111
		snap114 := d1
		snap115 := d2
		snap116 := d3
		snap117 := d4
		snap118 := d5
		snap119 := d6
		snap120 := d7
		snap121 := d8
		snap122 := d22
		snap123 := d23
		snap124 := d24
		snap125 := d25
		snap126 := d26
		snap127 := d27
		snap128 := d28
		snap129 := d29
		snap130 := d51
		snap131 := d52
		snap132 := d53
		snap133 := d54
		snap134 := d57
		snap135 := d60
		snap136 := d84
		snap137 := d85
		snap138 := d86
		snap139 := d88
		snap140 := d89
		snap141 := d90
		snap142 := d91
		snap143 := d92
		snap144 := d93
		snap145 := d94
		snap146 := d95
		snap147 := d96
		snap148 := d97
		snap149 := d99
		snap150 := d100
		snap151 := d101
		snap152 := d102
		snap153 := d103
		snap154 := d104
		snap155 := d105
		snap156 := d106
		snap157 := d109
		snap158 := d110
		snap159 := d111
		alloc160 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps113)
		}
		ctx.RestoreAllocState(alloc160)
		d1 = snap114
		d2 = snap115
		d3 = snap116
		d4 = snap117
		d5 = snap118
		d6 = snap119
		d7 = snap120
		d8 = snap121
		d22 = snap122
		d23 = snap123
		d24 = snap124
		d25 = snap125
		d26 = snap126
		d27 = snap127
		d28 = snap128
		d29 = snap129
		d51 = snap130
		d52 = snap131
		d53 = snap132
		d54 = snap133
		d57 = snap134
		d60 = snap135
		d84 = snap136
		d85 = snap137
		d86 = snap138
		d88 = snap139
		d89 = snap140
		d90 = snap141
		d91 = snap142
		d92 = snap143
		d93 = snap144
		d94 = snap145
		d95 = snap146
		d96 = snap147
		d97 = snap148
		d99 = snap149
		d100 = snap150
		d101 = snap151
		d102 = snap152
		d103 = snap153
		d104 = snap154
		d105 = snap155
		d106 = snap156
		d109 = snap157
		d110 = snap158
		d111 = snap159
		if !bbs[8].Rendered {
			return bbs[8].RenderPS(ps112)
		}
		return result
		ctx.FreeDesc(&d105)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[8].VisitCount >= 0 {
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
		}
		bbs[8].VisitCount++
		if ps.General {
			if bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			bbs[8].Rendered = true
			bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_8 = bbs[8].Address
			ctx.MarkLabel(lbl9)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
			d89 = ps.OverlayValues[89]
		}
		if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
			d90 = ps.OverlayValues[90]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
			d92 = ps.OverlayValues[92]
		}
		if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
			d93 = ps.OverlayValues[93]
		}
		if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
			d94 = ps.OverlayValues[94]
		}
		if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
			d95 = ps.OverlayValues[95]
		}
		if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
			d96 = ps.OverlayValues[96]
		}
		if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
			d97 = ps.OverlayValues[97]
		}
		if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
			d99 = ps.OverlayValues[99]
		}
		if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
			d100 = ps.OverlayValues[100]
		}
		if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
			d101 = ps.OverlayValues[101]
		}
		if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
			d102 = ps.OverlayValues[102]
		}
		if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
			d103 = ps.OverlayValues[103]
		}
		if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
			d104 = ps.OverlayValues[104]
		}
		if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
			d105 = ps.OverlayValues[105]
		}
		if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
			d106 = ps.OverlayValues[106]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&d2)
		d161 = d2
		_ = d161
		ctx.StabilizeDescForControlFlow(&d161)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl19 := ctx.ReserveLabel()
		_ = lbl19
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl19)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d161)
		var d162 scm.JITValueDesc
		if d161.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d161.Imm.Int() & 255)}
		} else {
			r25 := ctx.AllocRegExcept(d161.Reg)
			ctx.EmitMovRegReg(r25, d161.Reg)
			ctx.EmitAndRegImm32(r25, int32(255))
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d162)
		}
		if d162.Loc == scm.LocReg && d161.Loc == scm.LocReg && d162.Reg == d161.Reg {
			ctx.TransferReg(d161.Reg)
			d161.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d162)
		if d162.Loc == scm.LocRegPair || d162.Loc == scm.LocStackPair || d162.Loc == scm.LocRegTriple || d162.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d162)
		d163 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).decodeSymbol), []scm.JITValueDesc{thisptr, d162}, 1)
		d163.NoHeapPointer = true
		ctx.BindReg(d163.Reg, &d163)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d163)
		var d164 scm.JITValueDesc
		r26 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r26, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r26, thisptr.Reg)
			ctx.EmitAddRegImm32(r26, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d164 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r26, &d164)
		ctx.ReclaimUntrackedRegs()
		d166 = ctx.EmitSliceElementAddress(&d164, &d163, 8)
		ctx.EnsureDesc(&d166)
		ctx.EmitMovRegMem(d166.Reg, d166.Reg, 0)
		d165 = d166
		d165.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d163)
		var d167 scm.JITValueDesc
		r27 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r27, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r27, thisptr.Reg)
			ctx.EmitAddRegImm32(r27, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d167 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r27, &d167)
		ctx.ReclaimUntrackedRegs()
		d169 = ctx.EmitSliceElementAddress(&d167, &d163, 16)
		ctx.EnsureDesc(&d169)
		r28 := ctx.AllocRegExcept(d169.Reg)
		ctx.EmitMovRegMem(r28, d169.Reg, 8)
		ctx.EmitMovRegMem(d169.Reg, d169.Reg, 0)
		d168 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d169.Reg, Reg2: r28}
		ctx.BindReg(d169.Reg, &d168)
		ctx.BindReg(r28, &d168)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d161)
		var d170 scm.JITValueDesc
		if d161.Loc == scm.LocImm {
			d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d161.Imm.Int()) >> 8))}
		} else {
			r29 := ctx.AllocRegExcept(d161.Reg)
			ctx.EmitMovRegReg(r29, d161.Reg)
			ctx.EmitShrRegImm8(r29, 8)
			d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d170)
		}
		if d170.Loc == scm.LocReg && d161.Loc == scm.LocReg && d170.Reg == d161.Reg {
			ctx.TransferReg(d161.Reg)
			d161.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d170)
		ctx.EnsureDesc(&d165)
		ctx.EnsureDescsTogether(&d170, &d165)
		var d171 scm.JITValueDesc
		if d170.Loc == scm.LocImm && d165.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() * d165.Imm.Int())}
		} else if d170.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d165.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
			ctx.EmitImulInt64(scratch, d165.Reg)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d171)
		} else if d165.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(scratch, d170.Reg)
			if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d165.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d171)
		} else {
			r30 := ctx.AllocRegExcept(d170.Reg, d165.Reg)
			ctx.EmitMovRegReg(r30, d170.Reg)
			ctx.EmitImulInt64(r30, d165.Reg)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d171)
		}
		if d171.Loc == scm.LocReg && d170.Loc == scm.LocReg && d171.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d170)
		ctx.FreeDesc(&d165)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d171)
		ctx.EnsureDesc(&d162)
		ctx.EnsureDescsTogether(&d171, &d162)
		var d172 scm.JITValueDesc
		if d171.Loc == scm.LocImm && d162.Loc == scm.LocImm {
			d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() + d162.Imm.Int())}
		} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
			r31 := ctx.AllocRegExcept(d171.Reg)
			ctx.EmitMovRegReg(r31, d171.Reg)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d172)
		} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			ctx.BindReg(d162.Reg, &d172)
		} else if d171.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d162.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
			ctx.EmitAddInt64(scratch, d162.Reg)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d172)
		} else if d162.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d171.Reg)
			ctx.EmitMovRegReg(scratch, d171.Reg)
			if d162.Imm.Int() >= -2147483648 && d162.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d162.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d162.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d172)
		} else {
			r32 := ctx.AllocRegExcept(d171.Reg, d162.Reg)
			ctx.EmitMovRegReg(r32, d171.Reg)
			ctx.EmitAddInt64(r32, d162.Reg)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d172)
		}
		if d172.Loc == scm.LocReg && d171.Loc == scm.LocReg && d172.Reg == d171.Reg {
			ctx.TransferReg(d171.Reg)
			d171.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d171)
		ctx.FreeDesc(&d162)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d163)
		ctx.EnsureDesc(&d163)
		if d163.Loc == scm.LocRegPair || d163.Loc == scm.LocStackPair || d163.Loc == scm.LocRegTriple || d163.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d163)
		d173 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).symbolLo), []scm.JITValueDesc{thisptr, d163}, 1)
		d173.NoHeapPointer = true
		ctx.BindReg(d173.Reg, &d173)
		ctx.FreeDesc(&d163)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d172)
		ctx.EnsureDesc(&d173)
		ctx.EnsureDescsTogether(&d172, &d173)
		var d174 scm.JITValueDesc
		if d172.Loc == scm.LocImm && d173.Loc == scm.LocImm {
			d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d172.Imm.Int() - d173.Imm.Int())}
		} else if d173.Loc == scm.LocImm && d173.Imm.Int() == 0 {
			r33 := ctx.AllocRegExcept(d172.Reg)
			ctx.EmitMovRegReg(r33, d172.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d174)
		} else if d172.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d173.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d172.Imm.Int()))
			ctx.EmitSubInt64(scratch, d173.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d174)
		} else if d173.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d172.Reg)
			ctx.EmitMovRegReg(scratch, d172.Reg)
			if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d173.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d174)
		} else {
			r34 := ctx.AllocRegExcept(d172.Reg, d173.Reg)
			ctx.EmitMovRegReg(r34, d172.Reg)
			ctx.EmitSubInt64(r34, d173.Reg)
			d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d174)
		}
		if d174.Loc == scm.LocReg && d172.Loc == scm.LocReg && d174.Reg == d172.Reg {
			ctx.TransferReg(d172.Reg)
			d172.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d172)
		ctx.FreeDesc(&d173)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d168)
		ctx.EnsureDesc(&d174)
		ctx.StabilizeDescForControlFlow(&d168)
		ctx.StabilizeDescForControlFlow(&d174)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d175 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(scratch, d4.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d175)
		}
		if d175.Loc == scm.LocReg && d4.Loc == scm.LocReg && d175.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d175)
		ctx.EmitStoreToStack(d175, int32(bbs[7].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d175)
		if ps.General {
			ctx.SyncDesc(&d168)
			if d168.Loc == scm.LocReg {
				ctx.ProtectReg(d168.Reg)
			} else if d168.Loc == scm.LocRegPair {
				ctx.ProtectReg(d168.Reg)
				ctx.ProtectReg(d168.Reg2)
			}
			ctx.SyncDesc(&d174)
			if d174.Loc == scm.LocReg {
				ctx.ProtectReg(d174.Reg)
			} else if d174.Loc == scm.LocRegPair {
				ctx.ProtectReg(d174.Reg)
				ctx.ProtectReg(d174.Reg2)
			}
			d176 = d174
			if d176.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d176)
			ctx.EmitStoreToStack(d176, int32(bbs[7].PhiBase)+int32(0))
			d177 = d168
			if d177.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d177)
			if d177.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d177, int32(bbs[7].PhiBase)+int32(16), 2)
			} else if d177.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d177)
				ctx.EmitStoreScmerToStack(d177, int32(bbs[7].PhiBase)+int32(16))
			} else if d177.Loc == scm.LocRegPair || d177.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d177, int32(bbs[7].PhiBase)+int32(16))
			} else {
				ctx.EnsureDesc(&d177)
				ctx.EmitStoreToStack(d177, int32(bbs[7].PhiBase)+int32(16))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			}
			if d168.Loc == scm.LocReg {
				ctx.UnprotectReg(d168.Reg)
			} else if d168.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d168.Reg)
				ctx.UnprotectReg(d168.Reg2)
			}
			if d174.Loc == scm.LocReg {
				ctx.UnprotectReg(d174.Reg)
			} else if d174.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d174.Reg)
				ctx.UnprotectReg(d174.Reg2)
			}
		}
		ps178 := scm.PhiState{General: ps.General}
		ps178.OverlayValues = make([]scm.JITValueDesc, 178)
		ps178.OverlayValues[1] = d1
		ps178.OverlayValues[2] = d2
		ps178.OverlayValues[3] = d3
		ps178.OverlayValues[4] = d4
		ps178.OverlayValues[5] = d5
		ps178.OverlayValues[6] = d6
		ps178.OverlayValues[7] = d7
		ps178.OverlayValues[8] = d8
		ps178.OverlayValues[22] = d22
		ps178.OverlayValues[23] = d23
		ps178.OverlayValues[24] = d24
		ps178.OverlayValues[25] = d25
		ps178.OverlayValues[26] = d26
		ps178.OverlayValues[27] = d27
		ps178.OverlayValues[28] = d28
		ps178.OverlayValues[29] = d29
		ps178.OverlayValues[51] = d51
		ps178.OverlayValues[52] = d52
		ps178.OverlayValues[53] = d53
		ps178.OverlayValues[54] = d54
		ps178.OverlayValues[57] = d57
		ps178.OverlayValues[60] = d60
		ps178.OverlayValues[84] = d84
		ps178.OverlayValues[85] = d85
		ps178.OverlayValues[86] = d86
		ps178.OverlayValues[88] = d88
		ps178.OverlayValues[89] = d89
		ps178.OverlayValues[90] = d90
		ps178.OverlayValues[91] = d91
		ps178.OverlayValues[92] = d92
		ps178.OverlayValues[93] = d93
		ps178.OverlayValues[94] = d94
		ps178.OverlayValues[95] = d95
		ps178.OverlayValues[96] = d96
		ps178.OverlayValues[97] = d97
		ps178.OverlayValues[99] = d99
		ps178.OverlayValues[100] = d100
		ps178.OverlayValues[101] = d101
		ps178.OverlayValues[102] = d102
		ps178.OverlayValues[103] = d103
		ps178.OverlayValues[104] = d104
		ps178.OverlayValues[105] = d105
		ps178.OverlayValues[106] = d106
		ps178.OverlayValues[109] = d109
		ps178.OverlayValues[110] = d110
		ps178.OverlayValues[111] = d111
		ps178.OverlayValues[161] = d161
		ps178.OverlayValues[162] = d162
		ps178.OverlayValues[163] = d163
		ps178.OverlayValues[164] = d164
		ps178.OverlayValues[165] = d165
		ps178.OverlayValues[166] = d166
		ps178.OverlayValues[167] = d167
		ps178.OverlayValues[168] = d168
		ps178.OverlayValues[169] = d169
		ps178.OverlayValues[170] = d170
		ps178.OverlayValues[171] = d171
		ps178.OverlayValues[172] = d172
		ps178.OverlayValues[173] = d173
		ps178.OverlayValues[174] = d174
		ps178.OverlayValues[175] = d175
		ps178.OverlayValues[176] = d176
		ps178.OverlayValues[177] = d177
		ps178.PhiValues = make([]scm.JITValueDesc, 3)
		d179 = d174
		ps178.PhiValues[0] = d179
		d180 = d168
		ps178.PhiValues[1] = d180
		if ps178.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps178)
		return result
	}
	bbs[9].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[9].VisitCount >= 0 {
				ps.General = true
				return bbs[9].RenderPS(ps)
			}
		}
		bbs[9].VisitCount++
		if ps.General {
			if bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			bbs[9].Rendered = true
			bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_9 = bbs[9].Address
			ctx.MarkLabel(lbl10)
			ctx.ResolveFixups()
		}
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
			d89 = ps.OverlayValues[89]
		}
		if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
			d90 = ps.OverlayValues[90]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != scm.LocNone {
			d92 = ps.OverlayValues[92]
		}
		if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != scm.LocNone {
			d93 = ps.OverlayValues[93]
		}
		if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != scm.LocNone {
			d94 = ps.OverlayValues[94]
		}
		if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != scm.LocNone {
			d95 = ps.OverlayValues[95]
		}
		if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != scm.LocNone {
			d96 = ps.OverlayValues[96]
		}
		if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != scm.LocNone {
			d97 = ps.OverlayValues[97]
		}
		if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
			d99 = ps.OverlayValues[99]
		}
		if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
			d100 = ps.OverlayValues[100]
		}
		if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
			d101 = ps.OverlayValues[101]
		}
		if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
			d102 = ps.OverlayValues[102]
		}
		if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
			d103 = ps.OverlayValues[103]
		}
		if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != scm.LocNone {
			d104 = ps.OverlayValues[104]
		}
		if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != scm.LocNone {
			d105 = ps.OverlayValues[105]
		}
		if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != scm.LocNone {
			d106 = ps.OverlayValues[106]
		}
		if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != scm.LocNone {
			d109 = ps.OverlayValues[109]
		}
		if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != scm.LocNone {
			d110 = ps.OverlayValues[110]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
			d167 = ps.OverlayValues[167]
		}
		if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
			d168 = ps.OverlayValues[168]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
			d170 = ps.OverlayValues[170]
		}
		if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
			d171 = ps.OverlayValues[171]
		}
		if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
			d172 = ps.OverlayValues[172]
		}
		if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != scm.LocNone {
			d173 = ps.OverlayValues[173]
		}
		if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
			d174 = ps.OverlayValues[174]
		}
		if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
			d175 = ps.OverlayValues[175]
		}
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
		}
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		ctx.ReclaimUntrackedRegs()
		d181 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d181)
		ctx.BindReg(r1, &d181)
		ctx.EnsureDesc(&d3)
		if d3.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d3, &d181)
		} else {
			switch d3.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d181, d3)
			case scm.TagInt:
				ctx.EmitMakeInt(d181, d3)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d181, d3)
			case scm.TagNil:
				ctx.EmitMakeNil(d181)
			default:
				ctx.EmitMovPairToResult(&d3, &d181)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps182 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps182)
	ctx.MarkLabel(lbl0)
	d183 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d183)
	ctx.BindReg(r1, &d183)
	ctx.EmitMovPairToResult(&d183, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
	ctx.EndStandaloneFrame(standaloneFrame)
	return result
}

func (s *StorageEnum) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(40)) // magic byte 40 = StorageEnum
	binary.Write(f, binary.LittleEndian, uint8(s.k))
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint32(s.jumpL1Stride))
	binary.Write(f, binary.LittleEndian, uint64(len(s.data)))
	binary.Write(f, binary.LittleEndian, uint64(len(s.jumpL1)))
	binary.Write(f, binary.LittleEndian, uint64(len(s.jumpL2)))

	// symbol frequencies for rebuilding widths/thresholds
	for j := uint8(0); j < s.k; j++ {
		binary.Write(f, binary.LittleEndian, s.scanFreqs[j])
	}

	// symbol values as JSON lines
	for j := uint8(0); j < s.k; j++ {
		b, _ := json.Marshal(s.values[j])
		binary.Write(f, binary.LittleEndian, uint32(len(b)))
		f.Write(b)
	}

	// data chunks
	if len(s.data) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.data[0])), 8*len(s.data)))
	}
	// jumpL1
	if len(s.jumpL1) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.jumpL1[0])), 4*len(s.jumpL1)))
	}
	// jumpL2
	if len(s.jumpL2) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.jumpL2[0])), 2*len(s.jumpL2)))
	}
}

func (s *StorageEnum) Deserialize(f io.Reader) uint {
	// No version byte: the first byte is k (number of symbols).
	// Format changes require a new magic byte.
	binary.Read(f, binary.LittleEndian, &s.k)
	binary.Read(f, binary.LittleEndian, &s.count)
	var stride uint32
	binary.Read(f, binary.LittleEndian, &stride)
	s.jumpL1Stride = int(stride)
	var dataLen, l1Len, l2Len uint64
	binary.Read(f, binary.LittleEndian, &dataLen)
	binary.Read(f, binary.LittleEndian, &l1Len)
	binary.Read(f, binary.LittleEndian, &l2Len)

	// Read frequencies
	for j := uint8(0); j < s.k; j++ {
		binary.Read(f, binary.LittleEndian, &s.scanFreqs[j])
	}

	// Read symbol values
	for j := uint8(0); j < s.k; j++ {
		var vlen uint32
		binary.Read(f, binary.LittleEndian, &vlen)
		buf := make([]byte, vlen)
		io.ReadFull(f, buf)
		if err := json.Unmarshal(buf, &s.values[j]); err != nil {
			panic(err)
		}
	}

	// Rebuild widths/thresholds from frequencies
	s.rebuildCodec()

	// Read data
	if dataLen > 0 {
		raw := make([]byte, dataLen*8)
		io.ReadFull(f, raw)
		s.data = unsafe.Slice((*uint64)(unsafe.Pointer(&raw[0])), dataLen)
	}
	// Read jumpL1
	if l1Len > 0 {
		raw := make([]byte, l1Len*4)
		io.ReadFull(f, raw)
		s.jumpL1 = unsafe.Slice((*uint32)(unsafe.Pointer(&raw[0])), l1Len)
	}
	// Read jumpL2
	if l2Len > 0 {
		raw := make([]byte, l2Len*2)
		io.ReadFull(f, raw)
		s.jumpL2 = unsafe.Slice((*uint16)(unsafe.Pointer(&raw[0])), l2Len)
	}

	return uint(s.count)
}

// rebuildCodec reconstructs thresholds/widths/invWidths from scanFreqs.
func (s *StorageEnum) rebuildCodec() {
	k := int(s.k)
	if k < 2 {
		k = 2
		s.k = 2
	}
	total := uint64(0)
	for j := 0; j < k; j++ {
		total += s.scanFreqs[j]
	}
	if total == 0 {
		total = 1
	}

	slots := [enumMaxSymbols]uint64{}
	remaining := int(enumBitModulo) - k
	for j := 0; j < k; j++ {
		slots[j] = 1
	}
	distributed := 0
	for j := 0; j < k; j++ {
		extra := int(s.scanFreqs[j]) * remaining / int(total)
		slots[j] += uint64(extra)
		distributed += extra
	}
	leftover := remaining - distributed
	if leftover > 0 {
		maxIdx := 0
		for j := 1; j < k; j++ {
			if s.scanFreqs[j] > s.scanFreqs[maxIdx] {
				maxIdx = j
			}
		}
		slots[maxIdx] += uint64(leftover)
	}

	cum := uint64(0)
	for j := 0; j < k; j++ {
		s.widths[j] = slots[j]
		s.invWidths[j] = ^uint64(0) / slots[j]
		if j < k-1 {
			cum += slots[j]
			s.thresholds[j] = cum
		}
	}
}

func (s *StorageEnum) DistinctCount() uint { return uint(s.k) }
