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
	// Enum decoding is stateful even though the finished storage is immutable.
	// Keep one private rANS cursor per consumer: replacing it with the stateless
	// scalar JIT entry would restart symbol lookup and decoding for every row.
	// The separately exposed JIT range and multi readers remain available to
	// consumers which do not request this stateful reader contract.
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
	var d31 scm.JITValueDesc
	_ = d31
	var d32 scm.JITValueDesc
	_ = d32
	var d33 scm.JITValueDesc
	_ = d33
	var d34 scm.JITValueDesc
	_ = d34
	var d35 scm.JITValueDesc
	_ = d35
	var d36 scm.JITValueDesc
	_ = d36
	var d37 scm.JITValueDesc
	_ = d37
	var d38 scm.JITValueDesc
	_ = d38
	var d77 scm.JITValueDesc
	_ = d77
	var d78 scm.JITValueDesc
	_ = d78
	var d79 scm.JITValueDesc
	_ = d79
	var d80 scm.JITValueDesc
	_ = d80
	var d83 scm.JITValueDesc
	_ = d83
	var d108 scm.JITValueDesc
	_ = d108
	var d132 scm.JITValueDesc
	_ = d132
	var d133 scm.JITValueDesc
	_ = d133
	var d134 scm.JITValueDesc
	_ = d134
	var d136 scm.JITValueDesc
	_ = d136
	var d137 scm.JITValueDesc
	_ = d137
	var d138 scm.JITValueDesc
	_ = d138
	var d139 scm.JITValueDesc
	_ = d139
	var d140 scm.JITValueDesc
	_ = d140
	var d141 scm.JITValueDesc
	_ = d141
	var d142 scm.JITValueDesc
	_ = d142
	var d143 scm.JITValueDesc
	_ = d143
	var d144 scm.JITValueDesc
	_ = d144
	var d145 scm.JITValueDesc
	_ = d145
	var d147 scm.JITValueDesc
	_ = d147
	var d148 scm.JITValueDesc
	_ = d148
	var d149 scm.JITValueDesc
	_ = d149
	var d150 scm.JITValueDesc
	_ = d150
	var d151 scm.JITValueDesc
	_ = d151
	var d152 scm.JITValueDesc
	_ = d152
	var d153 scm.JITValueDesc
	_ = d153
	var d154 scm.JITValueDesc
	_ = d154
	var d157 scm.JITValueDesc
	_ = d157
	var d158 scm.JITValueDesc
	_ = d158
	var d159 scm.JITValueDesc
	_ = d159
	var d256 scm.JITValueDesc
	_ = d256
	var d257 scm.JITValueDesc
	_ = d257
	var d258 scm.JITValueDesc
	_ = d258
	var d259 scm.JITValueDesc
	_ = d259
	var d260 scm.JITValueDesc
	_ = d260
	var d261 scm.JITValueDesc
	_ = d261
	var d262 scm.JITValueDesc
	_ = d262
	var d263 scm.JITValueDesc
	_ = d263
	var d264 scm.JITValueDesc
	_ = d264
	var d265 scm.JITValueDesc
	_ = d265
	var d266 scm.JITValueDesc
	_ = d266
	var d267 scm.JITValueDesc
	_ = d267
	var d268 scm.JITValueDesc
	_ = d268
	var d269 scm.JITValueDesc
	_ = d269
	var d270 scm.JITValueDesc
	_ = d270
	var d271 scm.JITValueDesc
	_ = d271
	var d272 scm.JITValueDesc
	_ = d272
	var d274 scm.JITValueDesc
	_ = d274
	var d275 scm.JITValueDesc
	_ = d275
	var d276 scm.JITValueDesc
	_ = d276
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
		snap11 := d1
		snap12 := d2
		snap13 := d3
		snap14 := d4
		snap15 := d5
		snap16 := d6
		snap17 := d7
		snap18 := d8
		alloc19 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl2)
		ctx.RestoreAllocState(alloc19)
		d1 = snap11
		d2 = snap12
		d3 = snap13
		d4 = snap14
		d5 = snap15
		d6 = snap16
		d7 = snap17
		d8 = snap18
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc19)
		d1 = snap11
		d2 = snap12
		d3 = snap13
		d4 = snap14
		d5 = snap15
		d6 = snap16
		d7 = snap17
		d8 = snap18
		ps20 := scm.PhiState{General: true}
		ps20.OverlayValues = make([]scm.JITValueDesc, 9)
		ps20.OverlayValues[1] = d1
		ps20.OverlayValues[2] = d2
		ps20.OverlayValues[3] = d3
		ps20.OverlayValues[4] = d4
		ps20.OverlayValues[5] = d5
		ps20.OverlayValues[6] = d6
		ps20.OverlayValues[7] = d7
		ps20.OverlayValues[8] = d8
		ps21 := scm.PhiState{General: true}
		ps21.OverlayValues = make([]scm.JITValueDesc, 9)
		ps21.OverlayValues[1] = d1
		ps21.OverlayValues[2] = d2
		ps21.OverlayValues[3] = d3
		ps21.OverlayValues[4] = d4
		ps21.OverlayValues[5] = d5
		ps21.OverlayValues[6] = d6
		ps21.OverlayValues[7] = d7
		ps21.OverlayValues[8] = d8
		snap22 := d1
		snap23 := d2
		snap24 := d3
		snap25 := d4
		snap26 := d5
		snap27 := d6
		snap28 := d7
		snap29 := d8
		alloc30 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps21)
		}
		ctx.RestoreAllocState(alloc30)
		d1 = snap22
		d2 = snap23
		d3 = snap24
		d4 = snap25
		d5 = snap26
		d6 = snap27
		d7 = snap28
		d8 = snap29
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps20)
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
		d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d32 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d32)
		ctx.BindReg(r1, &d32)
		ctx.EnsureDesc(&d31)
		if d31.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d31, &d32)
		} else {
			switch d31.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d32, d31)
			case scm.TagInt:
				ctx.EmitMakeInt(d32, d31)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d32, d31)
			case scm.TagNil:
				ctx.EmitMakeNil(d32)
			default:
				ctx.EmitMovPairToResult(&d31, &d32)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d33 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, idxInt.Reg)
			ctx.EmitShlRegImm8(r7, 32)
			ctx.EmitShrRegImm8(r7, 32)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d33)
		}
		ctx.StabilizeDescForControlFlow(&d33)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d33)
		if d33.Loc == scm.LocRegPair || d33.Loc == scm.LocStackPair || d33.Loc == scm.LocRegTriple || d33.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d33)
		d34 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).findChunk), []scm.JITValueDesc{thisptr, d33}, 1)
		d34.NoHeapPointer = true
		ctx.BindReg(d34.Reg, &d34)
		ctx.StabilizeDescForControlFlow(&d34)
		var d35 scm.JITValueDesc
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
		d35 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
		ctx.BindReg(r8, &d35)
		ctx.BindReg(r9, &d35)
		ctx.BindReg(r10, &d35)
		ctx.BindReg(r8, &d35)
		ctx.BindReg(r9, &d35)
		ctx.BindReg(r10, &d35)
		var d36 scm.JITValueDesc
		if d35.SliceSizeKnown {
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d35.KnownSliceLen))}
		} else if d35.Loc == scm.LocImm {
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d35.StackOff))}
		} else if d35.Loc == scm.LocStackTriple {
			d36 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d35)
			if d35.Loc == scm.LocRegPair || d35.Loc == scm.LocRegTriple {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d35.Reg2, ID: 0}
			} else if d35.Loc == scm.LocReg {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d35.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d36)
		ctx.EnsureDescsTogether(&d34, &d36)
		var d37 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d34.Imm.Int() >= d36.Imm.Int())}
		} else if d36.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d34.Reg)
			if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d34.Reg, int32(d36.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d36.Imm.Int()))
				ctx.EmitCmpInt64(d34.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r11, scm.CondSignedGreaterOrEqual)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r11}
			ctx.BindReg(r11, &d37)
		} else if d34.Loc == scm.LocImm {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d36.Reg)
			ctx.EmitSetcc(r12, scm.CondSignedGreaterOrEqual)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r12}
			ctx.BindReg(r12, &d37)
		} else {
			r13 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitCmpInt64(d34.Reg, d36.Reg)
			ctx.EmitSetcc(r13, scm.CondSignedGreaterOrEqual)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r13}
			ctx.BindReg(r13, &d37)
		}
		ctx.FreeDesc(&d36)
		d38 = d37
		ctx.EnsureDesc(&d38)
		if d38.Loc != scm.LocImm && d38.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d38.Loc == scm.LocImm {
			if d38.Imm.Bool() {
				if ps.General {
				}
				ps39 := scm.PhiState{General: ps.General}
				ps39.OverlayValues = make([]scm.JITValueDesc, 39)
				ps39.OverlayValues[1] = d1
				ps39.OverlayValues[2] = d2
				ps39.OverlayValues[3] = d3
				ps39.OverlayValues[4] = d4
				ps39.OverlayValues[5] = d5
				ps39.OverlayValues[6] = d6
				ps39.OverlayValues[7] = d7
				ps39.OverlayValues[8] = d8
				ps39.OverlayValues[31] = d31
				ps39.OverlayValues[32] = d32
				ps39.OverlayValues[33] = d33
				ps39.OverlayValues[34] = d34
				ps39.OverlayValues[35] = d35
				ps39.OverlayValues[36] = d36
				ps39.OverlayValues[37] = d37
				ps39.OverlayValues[38] = d38
				return bbs[3].RenderPS(ps39)
			}
			if ps.General {
			}
			ps40 := scm.PhiState{General: ps.General}
			ps40.OverlayValues = make([]scm.JITValueDesc, 39)
			ps40.OverlayValues[1] = d1
			ps40.OverlayValues[2] = d2
			ps40.OverlayValues[3] = d3
			ps40.OverlayValues[4] = d4
			ps40.OverlayValues[5] = d5
			ps40.OverlayValues[6] = d6
			ps40.OverlayValues[7] = d7
			ps40.OverlayValues[8] = d8
			ps40.OverlayValues[31] = d31
			ps40.OverlayValues[32] = d32
			ps40.OverlayValues[33] = d33
			ps40.OverlayValues[34] = d34
			ps40.OverlayValues[35] = d35
			ps40.OverlayValues[36] = d36
			ps40.OverlayValues[37] = d37
			ps40.OverlayValues[38] = d38
			return bbs[4].RenderPS(ps40)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl13 := ctx.ReserveLabel()
		lbl14 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d38.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl13)
		ctx.EmitJmp(lbl14)
		snap41 := d1
		snap42 := d2
		snap43 := d3
		snap44 := d4
		snap45 := d5
		snap46 := d6
		snap47 := d7
		snap48 := d8
		snap49 := d31
		snap50 := d32
		snap51 := d33
		snap52 := d34
		snap53 := d35
		snap54 := d36
		snap55 := d37
		snap56 := d38
		alloc57 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl4)
		ctx.RestoreAllocState(alloc57)
		d1 = snap41
		d2 = snap42
		d3 = snap43
		d4 = snap44
		d5 = snap45
		d6 = snap46
		d7 = snap47
		d8 = snap48
		d31 = snap49
		d32 = snap50
		d33 = snap51
		d34 = snap52
		d35 = snap53
		d36 = snap54
		d37 = snap55
		d38 = snap56
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl5)
		ctx.RestoreAllocState(alloc57)
		d1 = snap41
		d2 = snap42
		d3 = snap43
		d4 = snap44
		d5 = snap45
		d6 = snap46
		d7 = snap47
		d8 = snap48
		d31 = snap49
		d32 = snap50
		d33 = snap51
		d34 = snap52
		d35 = snap53
		d36 = snap54
		d37 = snap55
		d38 = snap56
		ps58 := scm.PhiState{General: true}
		ps58.OverlayValues = make([]scm.JITValueDesc, 39)
		ps58.OverlayValues[1] = d1
		ps58.OverlayValues[2] = d2
		ps58.OverlayValues[3] = d3
		ps58.OverlayValues[4] = d4
		ps58.OverlayValues[5] = d5
		ps58.OverlayValues[6] = d6
		ps58.OverlayValues[7] = d7
		ps58.OverlayValues[8] = d8
		ps58.OverlayValues[31] = d31
		ps58.OverlayValues[32] = d32
		ps58.OverlayValues[33] = d33
		ps58.OverlayValues[34] = d34
		ps58.OverlayValues[35] = d35
		ps58.OverlayValues[36] = d36
		ps58.OverlayValues[37] = d37
		ps58.OverlayValues[38] = d38
		ps59 := scm.PhiState{General: true}
		ps59.OverlayValues = make([]scm.JITValueDesc, 39)
		ps59.OverlayValues[1] = d1
		ps59.OverlayValues[2] = d2
		ps59.OverlayValues[3] = d3
		ps59.OverlayValues[4] = d4
		ps59.OverlayValues[5] = d5
		ps59.OverlayValues[6] = d6
		ps59.OverlayValues[7] = d7
		ps59.OverlayValues[8] = d8
		ps59.OverlayValues[31] = d31
		ps59.OverlayValues[32] = d32
		ps59.OverlayValues[33] = d33
		ps59.OverlayValues[34] = d34
		ps59.OverlayValues[35] = d35
		ps59.OverlayValues[36] = d36
		ps59.OverlayValues[37] = d37
		ps59.OverlayValues[38] = d38
		snap60 := d1
		snap61 := d2
		snap62 := d3
		snap63 := d4
		snap64 := d5
		snap65 := d6
		snap66 := d7
		snap67 := d8
		snap68 := d31
		snap69 := d32
		snap70 := d33
		snap71 := d34
		snap72 := d35
		snap73 := d36
		snap74 := d37
		snap75 := d38
		alloc76 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps59)
		}
		ctx.RestoreAllocState(alloc76)
		d1 = snap60
		d2 = snap61
		d3 = snap62
		d4 = snap63
		d5 = snap64
		d6 = snap65
		d7 = snap66
		d8 = snap67
		d31 = snap68
		d32 = snap69
		d33 = snap70
		d34 = snap71
		d35 = snap72
		d36 = snap73
		d37 = snap74
		d38 = snap75
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps58)
		}
		return result
		ctx.FreeDesc(&d37)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		ctx.ReclaimUntrackedRegs()
		d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d78 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d78)
		ctx.BindReg(r1, &d78)
		ctx.EnsureDesc(&d77)
		if d77.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d77, &d78)
		} else {
			switch d77.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d78, d77)
			case scm.TagInt:
				ctx.EmitMakeInt(d78, d77)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d78, d77)
			case scm.TagNil:
				ctx.EmitMakeNil(d78)
			default:
				ctx.EmitMovPairToResult(&d77, &d78)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		var d79 scm.JITValueDesc
		if d34.Loc == scm.LocImm {
			d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d34.Imm.Int() > 0)}
		} else {
			r14 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitSetcc(r14, scm.CondSignedGreater)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r14}
			ctx.BindReg(r14, &d79)
		}
		d80 = d79
		ctx.EnsureDesc(&d80)
		if d80.Loc != scm.LocImm && d80.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d80.Loc == scm.LocImm {
			if d80.Imm.Bool() {
				if ps.General {
				}
				ps81 := scm.PhiState{General: ps.General}
				ps81.OverlayValues = make([]scm.JITValueDesc, 81)
				ps81.OverlayValues[1] = d1
				ps81.OverlayValues[2] = d2
				ps81.OverlayValues[3] = d3
				ps81.OverlayValues[4] = d4
				ps81.OverlayValues[5] = d5
				ps81.OverlayValues[6] = d6
				ps81.OverlayValues[7] = d7
				ps81.OverlayValues[8] = d8
				ps81.OverlayValues[31] = d31
				ps81.OverlayValues[32] = d32
				ps81.OverlayValues[33] = d33
				ps81.OverlayValues[34] = d34
				ps81.OverlayValues[35] = d35
				ps81.OverlayValues[36] = d36
				ps81.OverlayValues[37] = d37
				ps81.OverlayValues[38] = d38
				ps81.OverlayValues[77] = d77
				ps81.OverlayValues[78] = d78
				ps81.OverlayValues[79] = d79
				ps81.OverlayValues[80] = d80
				return bbs[5].RenderPS(ps81)
			}
			if ps.General {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
			}
			ps82 := scm.PhiState{General: ps.General}
			ps82.OverlayValues = make([]scm.JITValueDesc, 81)
			ps82.OverlayValues[1] = d1
			ps82.OverlayValues[2] = d2
			ps82.OverlayValues[3] = d3
			ps82.OverlayValues[4] = d4
			ps82.OverlayValues[5] = d5
			ps82.OverlayValues[6] = d6
			ps82.OverlayValues[7] = d7
			ps82.OverlayValues[8] = d8
			ps82.OverlayValues[31] = d31
			ps82.OverlayValues[32] = d32
			ps82.OverlayValues[33] = d33
			ps82.OverlayValues[34] = d34
			ps82.OverlayValues[35] = d35
			ps82.OverlayValues[36] = d36
			ps82.OverlayValues[37] = d37
			ps82.OverlayValues[38] = d38
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			ps82.OverlayValues[79] = d79
			ps82.OverlayValues[80] = d80
			ps82.PhiValues = make([]scm.JITValueDesc, 1)
			d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps82.PhiValues[0] = d83
			return bbs[6].RenderPS(ps82)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl15 := ctx.ReserveLabel()
		lbl16 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d80.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl15)
		ctx.EmitJmp(lbl16)
		snap84 := d1
		snap85 := d2
		snap86 := d3
		snap87 := d4
		snap88 := d5
		snap89 := d6
		snap90 := d7
		snap91 := d8
		snap92 := d31
		snap93 := d32
		snap94 := d33
		snap95 := d34
		snap96 := d35
		snap97 := d36
		snap98 := d37
		snap99 := d38
		snap100 := d77
		snap101 := d78
		snap102 := d79
		snap103 := d80
		snap104 := d83
		alloc105 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc105)
		d1 = snap84
		d2 = snap85
		d3 = snap86
		d4 = snap87
		d5 = snap88
		d6 = snap89
		d7 = snap90
		d8 = snap91
		d31 = snap92
		d32 = snap93
		d33 = snap94
		d34 = snap95
		d35 = snap96
		d36 = snap97
		d37 = snap98
		d38 = snap99
		d77 = snap100
		d78 = snap101
		d79 = snap102
		d80 = snap103
		d83 = snap104
		ctx.MarkLabel(lbl16)
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc105)
		d1 = snap84
		d2 = snap85
		d3 = snap86
		d4 = snap87
		d5 = snap88
		d6 = snap89
		d7 = snap90
		d8 = snap91
		d31 = snap92
		d32 = snap93
		d33 = snap94
		d34 = snap95
		d35 = snap96
		d36 = snap97
		d37 = snap98
		d38 = snap99
		d77 = snap100
		d78 = snap101
		d79 = snap102
		d80 = snap103
		d83 = snap104
		ps106 := scm.PhiState{General: true}
		ps106.OverlayValues = make([]scm.JITValueDesc, 84)
		ps106.OverlayValues[1] = d1
		ps106.OverlayValues[2] = d2
		ps106.OverlayValues[3] = d3
		ps106.OverlayValues[4] = d4
		ps106.OverlayValues[5] = d5
		ps106.OverlayValues[6] = d6
		ps106.OverlayValues[7] = d7
		ps106.OverlayValues[8] = d8
		ps106.OverlayValues[31] = d31
		ps106.OverlayValues[32] = d32
		ps106.OverlayValues[33] = d33
		ps106.OverlayValues[34] = d34
		ps106.OverlayValues[35] = d35
		ps106.OverlayValues[36] = d36
		ps106.OverlayValues[37] = d37
		ps106.OverlayValues[38] = d38
		ps106.OverlayValues[77] = d77
		ps106.OverlayValues[78] = d78
		ps106.OverlayValues[79] = d79
		ps106.OverlayValues[80] = d80
		ps106.OverlayValues[83] = d83
		ps107 := scm.PhiState{General: true}
		ps107.OverlayValues = make([]scm.JITValueDesc, 84)
		ps107.OverlayValues[1] = d1
		ps107.OverlayValues[2] = d2
		ps107.OverlayValues[3] = d3
		ps107.OverlayValues[4] = d4
		ps107.OverlayValues[5] = d5
		ps107.OverlayValues[6] = d6
		ps107.OverlayValues[7] = d7
		ps107.OverlayValues[8] = d8
		ps107.OverlayValues[31] = d31
		ps107.OverlayValues[32] = d32
		ps107.OverlayValues[33] = d33
		ps107.OverlayValues[34] = d34
		ps107.OverlayValues[35] = d35
		ps107.OverlayValues[36] = d36
		ps107.OverlayValues[37] = d37
		ps107.OverlayValues[38] = d38
		ps107.OverlayValues[77] = d77
		ps107.OverlayValues[78] = d78
		ps107.OverlayValues[79] = d79
		ps107.OverlayValues[80] = d80
		ps107.OverlayValues[83] = d83
		ps107.PhiValues = make([]scm.JITValueDesc, 1)
		d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps107.PhiValues[0] = d108
		snap109 := d1
		snap110 := d2
		snap111 := d3
		snap112 := d4
		snap113 := d5
		snap114 := d6
		snap115 := d7
		snap116 := d8
		snap117 := d31
		snap118 := d32
		snap119 := d33
		snap120 := d34
		snap121 := d35
		snap122 := d36
		snap123 := d37
		snap124 := d38
		snap125 := d77
		snap126 := d78
		snap127 := d79
		snap128 := d80
		snap129 := d83
		snap130 := d108
		alloc131 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps107)
		}
		ctx.RestoreAllocState(alloc131)
		d1 = snap109
		d2 = snap110
		d3 = snap111
		d4 = snap112
		d5 = snap113
		d6 = snap114
		d7 = snap115
		d8 = snap116
		d31 = snap117
		d32 = snap118
		d33 = snap119
		d34 = snap120
		d35 = snap121
		d36 = snap122
		d37 = snap123
		d38 = snap124
		d77 = snap125
		d78 = snap126
		d79 = snap127
		d80 = snap128
		d83 = snap129
		d108 = snap130
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps106)
		}
		return result
		ctx.FreeDesc(&d79)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
			d108 = ps.OverlayValues[108]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d34)
		var d132 scm.JITValueDesc
		if d34.Loc == scm.LocImm {
			d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(scratch, d34.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d132)
		}
		if d132.Loc == scm.LocReg && d34.Loc == scm.LocReg && d132.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d132)
		if d132.Loc == scm.LocRegPair || d132.Loc == scm.LocStackPair || d132.Loc == scm.LocRegTriple || d132.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d132)
		d133 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).jumpCum), []scm.JITValueDesc{thisptr, d132}, 1)
		d133.NoHeapPointer = true
		ctx.BindReg(d133.Reg, &d133)
		ctx.StabilizeDescForControlFlow(&d133)
		ctx.FreeDesc(&d132)
		if ps.General {
			ctx.SyncDesc(&d133)
			if d133.Loc == scm.LocReg {
				ctx.ProtectReg(d133.Reg)
			} else if d133.Loc == scm.LocRegPair {
				ctx.ProtectReg(d133.Reg)
				ctx.ProtectReg(d133.Reg2)
			}
			d134 = d133
			if d134.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d134)
			ctx.EmitStoreToStack(d134, int32(bbs[6].PhiBase)+int32(0))
			if d133.Loc == scm.LocReg {
				ctx.UnprotectReg(d133.Reg)
			} else if d133.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d133.Reg)
				ctx.UnprotectReg(d133.Reg2)
			}
		}
		ps135 := scm.PhiState{General: ps.General}
		ps135.OverlayValues = make([]scm.JITValueDesc, 135)
		ps135.OverlayValues[1] = d1
		ps135.OverlayValues[2] = d2
		ps135.OverlayValues[3] = d3
		ps135.OverlayValues[4] = d4
		ps135.OverlayValues[5] = d5
		ps135.OverlayValues[6] = d6
		ps135.OverlayValues[7] = d7
		ps135.OverlayValues[8] = d8
		ps135.OverlayValues[31] = d31
		ps135.OverlayValues[32] = d32
		ps135.OverlayValues[33] = d33
		ps135.OverlayValues[34] = d34
		ps135.OverlayValues[35] = d35
		ps135.OverlayValues[36] = d36
		ps135.OverlayValues[37] = d37
		ps135.OverlayValues[38] = d38
		ps135.OverlayValues[77] = d77
		ps135.OverlayValues[78] = d78
		ps135.OverlayValues[79] = d79
		ps135.OverlayValues[80] = d80
		ps135.OverlayValues[83] = d83
		ps135.OverlayValues[108] = d108
		ps135.OverlayValues[132] = d132
		ps135.OverlayValues[133] = d133
		ps135.OverlayValues[134] = d134
		ps135.PhiValues = make([]scm.JITValueDesc, 1)
		d136 = d133
		ps135.PhiValues[0] = d136
		if ps135.General && bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		return bbs[6].RenderPS(ps135)
		return result
	}
	bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d137 := ps.PhiValues[0]
				ctx.EnsureDesc(&d137)
				ctx.EmitStoreToStack(d137, int32(bbs[6].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
			d108 = ps.OverlayValues[108]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d1 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		var d138 scm.JITValueDesc
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
		d138 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r15, Reg2: r16, Reg3: r17}
		ctx.BindReg(r15, &d138)
		ctx.BindReg(r16, &d138)
		ctx.BindReg(r17, &d138)
		ctx.BindReg(r15, &d138)
		ctx.BindReg(r16, &d138)
		ctx.BindReg(r17, &d138)
		var d139 scm.JITValueDesc
		if d138.SliceSizeKnown {
			d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d138.KnownSliceLen))}
		} else if d138.Loc == scm.LocImm {
			d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d138.StackOff))}
		} else if d138.Loc == scm.LocStackTriple {
			d139 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d138.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d138)
			if d138.Loc == scm.LocRegPair || d138.Loc == scm.LocRegTriple {
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d138.Reg2, ID: 0}
			} else if d138.Loc == scm.LocReg {
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d138.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d139)
		ctx.EnsureDesc(&d139)
		var d140 scm.JITValueDesc
		if d139.Loc == scm.LocImm {
			d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d139.Reg)
			ctx.EmitMovRegReg(scratch, d139.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d140)
		}
		if d140.Loc == scm.LocReg && d139.Loc == scm.LocReg && d140.Reg == d139.Reg {
			ctx.TransferReg(d139.Reg)
			d139.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d139)
		ctx.EnsureDesc(&d140)
		ctx.EnsureDesc(&d34)
		ctx.EnsureDescsTogether(&d140, &d34)
		var d141 scm.JITValueDesc
		if d140.Loc == scm.LocImm && d34.Loc == scm.LocImm {
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() - d34.Imm.Int())}
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			r18 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r18, d140.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d141)
		} else if d140.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d140.Imm.Int()))
			ctx.EmitSubInt64(scratch, d34.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d141)
		} else if d34.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(scratch, d140.Reg)
			if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d141)
		} else {
			r19 := ctx.AllocRegExcept(d140.Reg, d34.Reg)
			ctx.EmitMovRegReg(r19, d140.Reg)
			ctx.EmitSubInt64(r19, d34.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d141)
		}
		if d141.Loc == scm.LocReg && d140.Loc == scm.LocReg && d141.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d140)
		ctx.EnsureDesc(&d141)
		d143 = ctx.EmitSliceElementAddress(&d138, &d141, 8)
		ctx.EnsureDesc(&d143)
		ctx.EmitMovRegMem(d143.Reg, d143.Reg, 0)
		d142 = d143
		d142.Type = scm.TagInt
		ctx.FreeDesc(&d141)
		ctx.StabilizeDescForControlFlow(&d142)
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDescsTogether(&d33, &d1)
		var d144 scm.JITValueDesc
		if d33.Loc == scm.LocImm && d1.Loc == scm.LocImm {
			d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() - d1.Imm.Int())}
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			r20 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(r20, d33.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d144)
		} else if d33.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
			ctx.EmitSubInt64(scratch, d1.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d144)
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(scratch, d33.Reg)
			if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d1.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d144)
		} else {
			r21 := ctx.AllocRegExcept(d33.Reg, d1.Reg)
			ctx.EmitMovRegReg(r21, d33.Reg)
			ctx.EmitSubInt64(r21, d1.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d144)
		}
		if d144.Loc == scm.LocReg && d33.Loc == scm.LocReg && d144.Reg == d33.Reg {
			ctx.TransferReg(d33.Reg)
			d33.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d144)
		ctx.FreeDesc(&d1)
		if ps.General {
			ctx.SyncDesc(&d142)
			if d142.Loc == scm.LocReg {
				ctx.ProtectReg(d142.Reg)
			} else if d142.Loc == scm.LocRegPair {
				ctx.ProtectReg(d142.Reg)
				ctx.ProtectReg(d142.Reg2)
			}
			d145 = d142
			if d145.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d145)
			ctx.EmitStoreToStack(d145, int32(bbs[7].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(32))
			if d142.Loc == scm.LocReg {
				ctx.UnprotectReg(d142.Reg)
			} else if d142.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d142.Reg)
				ctx.UnprotectReg(d142.Reg2)
			}
		}
		ps146 := scm.PhiState{General: ps.General}
		ps146.OverlayValues = make([]scm.JITValueDesc, 146)
		ps146.OverlayValues[1] = d1
		ps146.OverlayValues[2] = d2
		ps146.OverlayValues[3] = d3
		ps146.OverlayValues[4] = d4
		ps146.OverlayValues[5] = d5
		ps146.OverlayValues[6] = d6
		ps146.OverlayValues[7] = d7
		ps146.OverlayValues[8] = d8
		ps146.OverlayValues[31] = d31
		ps146.OverlayValues[32] = d32
		ps146.OverlayValues[33] = d33
		ps146.OverlayValues[34] = d34
		ps146.OverlayValues[35] = d35
		ps146.OverlayValues[36] = d36
		ps146.OverlayValues[37] = d37
		ps146.OverlayValues[38] = d38
		ps146.OverlayValues[77] = d77
		ps146.OverlayValues[78] = d78
		ps146.OverlayValues[79] = d79
		ps146.OverlayValues[80] = d80
		ps146.OverlayValues[83] = d83
		ps146.OverlayValues[108] = d108
		ps146.OverlayValues[132] = d132
		ps146.OverlayValues[133] = d133
		ps146.OverlayValues[134] = d134
		ps146.OverlayValues[136] = d136
		ps146.OverlayValues[137] = d137
		ps146.OverlayValues[138] = d138
		ps146.OverlayValues[139] = d139
		ps146.OverlayValues[140] = d140
		ps146.OverlayValues[141] = d141
		ps146.OverlayValues[142] = d142
		ps146.OverlayValues[143] = d143
		ps146.OverlayValues[144] = d144
		ps146.OverlayValues[145] = d145
		ps146.PhiValues = make([]scm.JITValueDesc, 3)
		d147 = d142
		ps146.PhiValues[0] = d147
		d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ps146.PhiValues[1] = d148
		d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps146.PhiValues[2] = d149
		if ps146.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps146)
		return result
	}
	bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d150 := ps.PhiValues[0]
				ctx.EnsureDesc(&d150)
				ctx.EmitStoreToStack(d150, int32(bbs[7].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d151 := ps.PhiValues[1]
				ctx.EnsureDesc(&d151)
				ctx.EmitStoreScmerToStack(d151, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d152 := ps.PhiValues[2]
				ctx.EnsureDesc(&d152)
				ctx.EmitStoreToStack(d152, int32(bbs[7].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
			d108 = ps.OverlayValues[108]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
		}
		if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
			d138 = ps.OverlayValues[138]
		}
		if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
			d139 = ps.OverlayValues[139]
		}
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
			d141 = ps.OverlayValues[141]
		}
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
			d145 = ps.OverlayValues[145]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
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
		ctx.EnsureDesc(&d144)
		ctx.EnsureDescsTogether(&d4, &d144)
		var d153 scm.JITValueDesc
		if d4.Loc == scm.LocImm && d144.Loc == scm.LocImm {
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d4.Imm.Int() <= d144.Imm.Int())}
		} else if d144.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d4.Reg)
			if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d4.Reg, int32(d144.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.EmitCmpInt64(d4.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r22, scm.CondSignedLessOrEqual)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r22}
			ctx.BindReg(r22, &d153)
		} else if d4.Loc == scm.LocImm {
			r23 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d144.Reg)
			ctx.EmitSetcc(r23, scm.CondSignedLessOrEqual)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r23}
			ctx.BindReg(r23, &d153)
		} else {
			r24 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitCmpInt64(d4.Reg, d144.Reg)
			ctx.EmitSetcc(r24, scm.CondSignedLessOrEqual)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r24}
			ctx.BindReg(r24, &d153)
		}
		d154 = d153
		ctx.EnsureDesc(&d154)
		if d154.Loc != scm.LocImm && d154.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d154.Loc == scm.LocImm {
			if d154.Imm.Bool() {
				if ps.General {
				}
				ps155 := scm.PhiState{General: ps.General}
				ps155.OverlayValues = make([]scm.JITValueDesc, 155)
				ps155.OverlayValues[1] = d1
				ps155.OverlayValues[2] = d2
				ps155.OverlayValues[3] = d3
				ps155.OverlayValues[4] = d4
				ps155.OverlayValues[5] = d5
				ps155.OverlayValues[6] = d6
				ps155.OverlayValues[7] = d7
				ps155.OverlayValues[8] = d8
				ps155.OverlayValues[31] = d31
				ps155.OverlayValues[32] = d32
				ps155.OverlayValues[33] = d33
				ps155.OverlayValues[34] = d34
				ps155.OverlayValues[35] = d35
				ps155.OverlayValues[36] = d36
				ps155.OverlayValues[37] = d37
				ps155.OverlayValues[38] = d38
				ps155.OverlayValues[77] = d77
				ps155.OverlayValues[78] = d78
				ps155.OverlayValues[79] = d79
				ps155.OverlayValues[80] = d80
				ps155.OverlayValues[83] = d83
				ps155.OverlayValues[108] = d108
				ps155.OverlayValues[132] = d132
				ps155.OverlayValues[133] = d133
				ps155.OverlayValues[134] = d134
				ps155.OverlayValues[136] = d136
				ps155.OverlayValues[137] = d137
				ps155.OverlayValues[138] = d138
				ps155.OverlayValues[139] = d139
				ps155.OverlayValues[140] = d140
				ps155.OverlayValues[141] = d141
				ps155.OverlayValues[142] = d142
				ps155.OverlayValues[143] = d143
				ps155.OverlayValues[144] = d144
				ps155.OverlayValues[145] = d145
				ps155.OverlayValues[147] = d147
				ps155.OverlayValues[148] = d148
				ps155.OverlayValues[149] = d149
				ps155.OverlayValues[150] = d150
				ps155.OverlayValues[151] = d151
				ps155.OverlayValues[152] = d152
				ps155.OverlayValues[153] = d153
				ps155.OverlayValues[154] = d154
				return bbs[8].RenderPS(ps155)
			}
			if ps.General {
			}
			ps156 := scm.PhiState{General: ps.General}
			ps156.OverlayValues = make([]scm.JITValueDesc, 155)
			ps156.OverlayValues[1] = d1
			ps156.OverlayValues[2] = d2
			ps156.OverlayValues[3] = d3
			ps156.OverlayValues[4] = d4
			ps156.OverlayValues[5] = d5
			ps156.OverlayValues[6] = d6
			ps156.OverlayValues[7] = d7
			ps156.OverlayValues[8] = d8
			ps156.OverlayValues[31] = d31
			ps156.OverlayValues[32] = d32
			ps156.OverlayValues[33] = d33
			ps156.OverlayValues[34] = d34
			ps156.OverlayValues[35] = d35
			ps156.OverlayValues[36] = d36
			ps156.OverlayValues[37] = d37
			ps156.OverlayValues[38] = d38
			ps156.OverlayValues[77] = d77
			ps156.OverlayValues[78] = d78
			ps156.OverlayValues[79] = d79
			ps156.OverlayValues[80] = d80
			ps156.OverlayValues[83] = d83
			ps156.OverlayValues[108] = d108
			ps156.OverlayValues[132] = d132
			ps156.OverlayValues[133] = d133
			ps156.OverlayValues[134] = d134
			ps156.OverlayValues[136] = d136
			ps156.OverlayValues[137] = d137
			ps156.OverlayValues[138] = d138
			ps156.OverlayValues[139] = d139
			ps156.OverlayValues[140] = d140
			ps156.OverlayValues[141] = d141
			ps156.OverlayValues[142] = d142
			ps156.OverlayValues[143] = d143
			ps156.OverlayValues[144] = d144
			ps156.OverlayValues[145] = d145
			ps156.OverlayValues[147] = d147
			ps156.OverlayValues[148] = d148
			ps156.OverlayValues[149] = d149
			ps156.OverlayValues[150] = d150
			ps156.OverlayValues[151] = d151
			ps156.OverlayValues[152] = d152
			ps156.OverlayValues[153] = d153
			ps156.OverlayValues[154] = d154
			return bbs[9].RenderPS(ps156)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d157 := ps.PhiValues[0]
				ctx.EnsureDesc(&d157)
				ctx.EmitStoreToStack(d157, int32(bbs[7].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d158 := ps.PhiValues[1]
				ctx.EnsureDesc(&d158)
				ctx.EmitStoreScmerToStack(d158, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d159 := ps.PhiValues[2]
				ctx.EnsureDesc(&d159)
				ctx.EmitStoreToStack(d159, int32(bbs[7].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		lbl17 := ctx.ReserveLabel()
		lbl18 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d154.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl17)
		ctx.EmitJmp(lbl18)
		snap160 := d1
		snap161 := d2
		snap162 := d3
		snap163 := d4
		snap164 := d5
		snap165 := d6
		snap166 := d7
		snap167 := d8
		snap168 := d31
		snap169 := d32
		snap170 := d33
		snap171 := d34
		snap172 := d35
		snap173 := d36
		snap174 := d37
		snap175 := d38
		snap176 := d77
		snap177 := d78
		snap178 := d79
		snap179 := d80
		snap180 := d83
		snap181 := d108
		snap182 := d132
		snap183 := d133
		snap184 := d134
		snap185 := d136
		snap186 := d137
		snap187 := d138
		snap188 := d139
		snap189 := d140
		snap190 := d141
		snap191 := d142
		snap192 := d143
		snap193 := d144
		snap194 := d145
		snap195 := d147
		snap196 := d148
		snap197 := d149
		snap198 := d150
		snap199 := d151
		snap200 := d152
		snap201 := d153
		snap202 := d154
		snap203 := d157
		snap204 := d158
		snap205 := d159
		alloc206 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl9)
		ctx.RestoreAllocState(alloc206)
		d1 = snap160
		d2 = snap161
		d3 = snap162
		d4 = snap163
		d5 = snap164
		d6 = snap165
		d7 = snap166
		d8 = snap167
		d31 = snap168
		d32 = snap169
		d33 = snap170
		d34 = snap171
		d35 = snap172
		d36 = snap173
		d37 = snap174
		d38 = snap175
		d77 = snap176
		d78 = snap177
		d79 = snap178
		d80 = snap179
		d83 = snap180
		d108 = snap181
		d132 = snap182
		d133 = snap183
		d134 = snap184
		d136 = snap185
		d137 = snap186
		d138 = snap187
		d139 = snap188
		d140 = snap189
		d141 = snap190
		d142 = snap191
		d143 = snap192
		d144 = snap193
		d145 = snap194
		d147 = snap195
		d148 = snap196
		d149 = snap197
		d150 = snap198
		d151 = snap199
		d152 = snap200
		d153 = snap201
		d154 = snap202
		d157 = snap203
		d158 = snap204
		d159 = snap205
		ctx.MarkLabel(lbl18)
		ctx.EmitJmp(lbl10)
		ctx.RestoreAllocState(alloc206)
		d1 = snap160
		d2 = snap161
		d3 = snap162
		d4 = snap163
		d5 = snap164
		d6 = snap165
		d7 = snap166
		d8 = snap167
		d31 = snap168
		d32 = snap169
		d33 = snap170
		d34 = snap171
		d35 = snap172
		d36 = snap173
		d37 = snap174
		d38 = snap175
		d77 = snap176
		d78 = snap177
		d79 = snap178
		d80 = snap179
		d83 = snap180
		d108 = snap181
		d132 = snap182
		d133 = snap183
		d134 = snap184
		d136 = snap185
		d137 = snap186
		d138 = snap187
		d139 = snap188
		d140 = snap189
		d141 = snap190
		d142 = snap191
		d143 = snap192
		d144 = snap193
		d145 = snap194
		d147 = snap195
		d148 = snap196
		d149 = snap197
		d150 = snap198
		d151 = snap199
		d152 = snap200
		d153 = snap201
		d154 = snap202
		d157 = snap203
		d158 = snap204
		d159 = snap205
		ps207 := scm.PhiState{General: true}
		ps207.OverlayValues = make([]scm.JITValueDesc, 160)
		ps207.OverlayValues[1] = d1
		ps207.OverlayValues[2] = d2
		ps207.OverlayValues[3] = d3
		ps207.OverlayValues[4] = d4
		ps207.OverlayValues[5] = d5
		ps207.OverlayValues[6] = d6
		ps207.OverlayValues[7] = d7
		ps207.OverlayValues[8] = d8
		ps207.OverlayValues[31] = d31
		ps207.OverlayValues[32] = d32
		ps207.OverlayValues[33] = d33
		ps207.OverlayValues[34] = d34
		ps207.OverlayValues[35] = d35
		ps207.OverlayValues[36] = d36
		ps207.OverlayValues[37] = d37
		ps207.OverlayValues[38] = d38
		ps207.OverlayValues[77] = d77
		ps207.OverlayValues[78] = d78
		ps207.OverlayValues[79] = d79
		ps207.OverlayValues[80] = d80
		ps207.OverlayValues[83] = d83
		ps207.OverlayValues[108] = d108
		ps207.OverlayValues[132] = d132
		ps207.OverlayValues[133] = d133
		ps207.OverlayValues[134] = d134
		ps207.OverlayValues[136] = d136
		ps207.OverlayValues[137] = d137
		ps207.OverlayValues[138] = d138
		ps207.OverlayValues[139] = d139
		ps207.OverlayValues[140] = d140
		ps207.OverlayValues[141] = d141
		ps207.OverlayValues[142] = d142
		ps207.OverlayValues[143] = d143
		ps207.OverlayValues[144] = d144
		ps207.OverlayValues[145] = d145
		ps207.OverlayValues[147] = d147
		ps207.OverlayValues[148] = d148
		ps207.OverlayValues[149] = d149
		ps207.OverlayValues[150] = d150
		ps207.OverlayValues[151] = d151
		ps207.OverlayValues[152] = d152
		ps207.OverlayValues[153] = d153
		ps207.OverlayValues[154] = d154
		ps207.OverlayValues[157] = d157
		ps207.OverlayValues[158] = d158
		ps207.OverlayValues[159] = d159
		ps208 := scm.PhiState{General: true}
		ps208.OverlayValues = make([]scm.JITValueDesc, 160)
		ps208.OverlayValues[1] = d1
		ps208.OverlayValues[2] = d2
		ps208.OverlayValues[3] = d3
		ps208.OverlayValues[4] = d4
		ps208.OverlayValues[5] = d5
		ps208.OverlayValues[6] = d6
		ps208.OverlayValues[7] = d7
		ps208.OverlayValues[8] = d8
		ps208.OverlayValues[31] = d31
		ps208.OverlayValues[32] = d32
		ps208.OverlayValues[33] = d33
		ps208.OverlayValues[34] = d34
		ps208.OverlayValues[35] = d35
		ps208.OverlayValues[36] = d36
		ps208.OverlayValues[37] = d37
		ps208.OverlayValues[38] = d38
		ps208.OverlayValues[77] = d77
		ps208.OverlayValues[78] = d78
		ps208.OverlayValues[79] = d79
		ps208.OverlayValues[80] = d80
		ps208.OverlayValues[83] = d83
		ps208.OverlayValues[108] = d108
		ps208.OverlayValues[132] = d132
		ps208.OverlayValues[133] = d133
		ps208.OverlayValues[134] = d134
		ps208.OverlayValues[136] = d136
		ps208.OverlayValues[137] = d137
		ps208.OverlayValues[138] = d138
		ps208.OverlayValues[139] = d139
		ps208.OverlayValues[140] = d140
		ps208.OverlayValues[141] = d141
		ps208.OverlayValues[142] = d142
		ps208.OverlayValues[143] = d143
		ps208.OverlayValues[144] = d144
		ps208.OverlayValues[145] = d145
		ps208.OverlayValues[147] = d147
		ps208.OverlayValues[148] = d148
		ps208.OverlayValues[149] = d149
		ps208.OverlayValues[150] = d150
		ps208.OverlayValues[151] = d151
		ps208.OverlayValues[152] = d152
		ps208.OverlayValues[153] = d153
		ps208.OverlayValues[154] = d154
		ps208.OverlayValues[157] = d157
		ps208.OverlayValues[158] = d158
		ps208.OverlayValues[159] = d159
		snap209 := d1
		snap210 := d2
		snap211 := d3
		snap212 := d4
		snap213 := d5
		snap214 := d6
		snap215 := d7
		snap216 := d8
		snap217 := d31
		snap218 := d32
		snap219 := d33
		snap220 := d34
		snap221 := d35
		snap222 := d36
		snap223 := d37
		snap224 := d38
		snap225 := d77
		snap226 := d78
		snap227 := d79
		snap228 := d80
		snap229 := d83
		snap230 := d108
		snap231 := d132
		snap232 := d133
		snap233 := d134
		snap234 := d136
		snap235 := d137
		snap236 := d138
		snap237 := d139
		snap238 := d140
		snap239 := d141
		snap240 := d142
		snap241 := d143
		snap242 := d144
		snap243 := d145
		snap244 := d147
		snap245 := d148
		snap246 := d149
		snap247 := d150
		snap248 := d151
		snap249 := d152
		snap250 := d153
		snap251 := d154
		snap252 := d157
		snap253 := d158
		snap254 := d159
		alloc255 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps208)
		}
		ctx.RestoreAllocState(alloc255)
		d1 = snap209
		d2 = snap210
		d3 = snap211
		d4 = snap212
		d5 = snap213
		d6 = snap214
		d7 = snap215
		d8 = snap216
		d31 = snap217
		d32 = snap218
		d33 = snap219
		d34 = snap220
		d35 = snap221
		d36 = snap222
		d37 = snap223
		d38 = snap224
		d77 = snap225
		d78 = snap226
		d79 = snap227
		d80 = snap228
		d83 = snap229
		d108 = snap230
		d132 = snap231
		d133 = snap232
		d134 = snap233
		d136 = snap234
		d137 = snap235
		d138 = snap236
		d139 = snap237
		d140 = snap238
		d141 = snap239
		d142 = snap240
		d143 = snap241
		d144 = snap242
		d145 = snap243
		d147 = snap244
		d148 = snap245
		d149 = snap246
		d150 = snap247
		d151 = snap248
		d152 = snap249
		d153 = snap250
		d154 = snap251
		d157 = snap252
		d158 = snap253
		d159 = snap254
		if !bbs[8].Rendered {
			return bbs[8].RenderPS(ps207)
		}
		return result
		ctx.FreeDesc(&d153)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
			d108 = ps.OverlayValues[108]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
		}
		if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
			d138 = ps.OverlayValues[138]
		}
		if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
			d139 = ps.OverlayValues[139]
		}
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
			d141 = ps.OverlayValues[141]
		}
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
			d145 = ps.OverlayValues[145]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&d2)
		d256 = d2
		_ = d256
		ctx.StabilizeDescForControlFlow(&d256)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl19 := ctx.ReserveLabel()
		_ = lbl19
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl19)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d256)
		var d257 scm.JITValueDesc
		if d256.Loc == scm.LocImm {
			d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d256.Imm.Int() & 255)}
		} else {
			r25 := ctx.AllocRegExcept(d256.Reg)
			ctx.EmitMovRegReg(r25, d256.Reg)
			ctx.EmitAndRegImm32(r25, int32(255))
			d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d257)
		}
		if d257.Loc == scm.LocReg && d256.Loc == scm.LocReg && d257.Reg == d256.Reg {
			ctx.TransferReg(d256.Reg)
			d256.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d257)
		ctx.EnsureDesc(&d257)
		if d257.Loc == scm.LocRegPair || d257.Loc == scm.LocStackPair || d257.Loc == scm.LocRegTriple || d257.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d257)
		d258 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).decodeSymbol), []scm.JITValueDesc{thisptr, d257}, 1)
		d258.NoHeapPointer = true
		ctx.BindReg(d258.Reg, &d258)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d258)
		var d259 scm.JITValueDesc
		r26 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r26, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r26, thisptr.Reg)
			ctx.EmitAddRegImm32(r26, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d259 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r26, &d259)
		ctx.ReclaimUntrackedRegs()
		d261 = ctx.EmitSliceElementAddress(&d259, &d258, 8)
		ctx.EnsureDesc(&d261)
		ctx.EmitMovRegMem(d261.Reg, d261.Reg, 0)
		d260 = d261
		d260.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d258)
		var d262 scm.JITValueDesc
		r27 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r27, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r27, thisptr.Reg)
			ctx.EmitAddRegImm32(r27, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d262 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r27, &d262)
		ctx.ReclaimUntrackedRegs()
		d264 = ctx.EmitSliceElementAddress(&d262, &d258, 16)
		ctx.EnsureDesc(&d264)
		r28 := ctx.AllocRegExcept(d264.Reg)
		ctx.EmitMovRegMem(r28, d264.Reg, 8)
		ctx.EmitMovRegMem(d264.Reg, d264.Reg, 0)
		d263 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d264.Reg, Reg2: r28}
		ctx.BindReg(d264.Reg, &d263)
		ctx.BindReg(r28, &d263)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d256)
		var d265 scm.JITValueDesc
		if d256.Loc == scm.LocImm {
			d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) >> 8))}
		} else {
			r29 := ctx.AllocRegExcept(d256.Reg)
			ctx.EmitMovRegReg(r29, d256.Reg)
			ctx.EmitShrRegImm8(r29, 8)
			d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d265)
		}
		if d265.Loc == scm.LocReg && d256.Loc == scm.LocReg && d265.Reg == d256.Reg {
			ctx.TransferReg(d256.Reg)
			d256.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d265)
		ctx.EnsureDesc(&d260)
		ctx.EnsureDescsTogether(&d265, &d260)
		var d266 scm.JITValueDesc
		if d265.Loc == scm.LocImm && d260.Loc == scm.LocImm {
			d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d265.Imm.Int() * d260.Imm.Int())}
		} else if d265.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d260.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d265.Imm.Int()))
			ctx.EmitImulInt64(scratch, d260.Reg)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d266)
		} else if d260.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d265.Reg)
			ctx.EmitMovRegReg(scratch, d265.Reg)
			if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d260.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d260.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d266)
		} else {
			r30 := ctx.AllocRegExcept(d265.Reg, d260.Reg)
			ctx.EmitMovRegReg(r30, d265.Reg)
			ctx.EmitImulInt64(r30, d260.Reg)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d266)
		}
		if d266.Loc == scm.LocReg && d265.Loc == scm.LocReg && d266.Reg == d265.Reg {
			ctx.TransferReg(d265.Reg)
			d265.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d265)
		ctx.FreeDesc(&d260)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d266)
		ctx.EnsureDesc(&d257)
		ctx.EnsureDescsTogether(&d266, &d257)
		var d267 scm.JITValueDesc
		if d266.Loc == scm.LocImm && d257.Loc == scm.LocImm {
			d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d266.Imm.Int() + d257.Imm.Int())}
		} else if d257.Loc == scm.LocImm && d257.Imm.Int() == 0 {
			r31 := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(r31, d266.Reg)
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d267)
		} else if d266.Loc == scm.LocImm && d266.Imm.Int() == 0 {
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d257.Reg}
			ctx.BindReg(d257.Reg, &d267)
		} else if d266.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d257.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d266.Imm.Int()))
			ctx.EmitAddInt64(scratch, d257.Reg)
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d267)
		} else if d257.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(scratch, d266.Reg)
			if d257.Imm.Int() >= -2147483648 && d257.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d257.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d257.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d267)
		} else {
			r32 := ctx.AllocRegExcept(d266.Reg, d257.Reg)
			ctx.EmitMovRegReg(r32, d266.Reg)
			ctx.EmitAddInt64(r32, d257.Reg)
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d267)
		}
		if d267.Loc == scm.LocReg && d266.Loc == scm.LocReg && d267.Reg == d266.Reg {
			ctx.TransferReg(d266.Reg)
			d266.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d266)
		ctx.FreeDesc(&d257)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d258)
		ctx.EnsureDesc(&d258)
		if d258.Loc == scm.LocRegPair || d258.Loc == scm.LocStackPair || d258.Loc == scm.LocRegTriple || d258.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d258)
		d268 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).symbolLo), []scm.JITValueDesc{thisptr, d258}, 1)
		d268.NoHeapPointer = true
		ctx.BindReg(d268.Reg, &d268)
		ctx.FreeDesc(&d258)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d267)
		ctx.EnsureDesc(&d268)
		ctx.EnsureDescsTogether(&d267, &d268)
		var d269 scm.JITValueDesc
		if d267.Loc == scm.LocImm && d268.Loc == scm.LocImm {
			d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d267.Imm.Int() - d268.Imm.Int())}
		} else if d268.Loc == scm.LocImm && d268.Imm.Int() == 0 {
			r33 := ctx.AllocRegExcept(d267.Reg)
			ctx.EmitMovRegReg(r33, d267.Reg)
			d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d269)
		} else if d267.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d268.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d267.Imm.Int()))
			ctx.EmitSubInt64(scratch, d268.Reg)
			d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d269)
		} else if d268.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d267.Reg)
			ctx.EmitMovRegReg(scratch, d267.Reg)
			if d268.Imm.Int() >= -2147483648 && d268.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d268.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d268.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d269)
		} else {
			r34 := ctx.AllocRegExcept(d267.Reg, d268.Reg)
			ctx.EmitMovRegReg(r34, d267.Reg)
			ctx.EmitSubInt64(r34, d268.Reg)
			d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d269)
		}
		if d269.Loc == scm.LocReg && d267.Loc == scm.LocReg && d269.Reg == d267.Reg {
			ctx.TransferReg(d267.Reg)
			d267.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d267)
		ctx.FreeDesc(&d268)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d263)
		ctx.EnsureDesc(&d269)
		ctx.StabilizeDescForControlFlow(&d263)
		ctx.StabilizeDescForControlFlow(&d269)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d270 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(scratch, d4.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d270)
		}
		if d270.Loc == scm.LocReg && d4.Loc == scm.LocReg && d270.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d270)
		ctx.EmitStoreToStack(d270, int32(bbs[7].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d270)
		if ps.General {
			ctx.SyncDesc(&d263)
			if d263.Loc == scm.LocReg {
				ctx.ProtectReg(d263.Reg)
			} else if d263.Loc == scm.LocRegPair {
				ctx.ProtectReg(d263.Reg)
				ctx.ProtectReg(d263.Reg2)
			}
			ctx.SyncDesc(&d269)
			if d269.Loc == scm.LocReg {
				ctx.ProtectReg(d269.Reg)
			} else if d269.Loc == scm.LocRegPair {
				ctx.ProtectReg(d269.Reg)
				ctx.ProtectReg(d269.Reg2)
			}
			d271 = d269
			if d271.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d271)
			ctx.EmitStoreToStack(d271, int32(bbs[7].PhiBase)+int32(0))
			d272 = d263
			if d272.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d272)
			if d272.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d272, int32(bbs[7].PhiBase)+int32(16), 2)
			} else if d272.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d272)
				ctx.EmitStoreScmerToStack(d272, int32(bbs[7].PhiBase)+int32(16))
			} else if d272.Loc == scm.LocRegPair || d272.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d272, int32(bbs[7].PhiBase)+int32(16))
			} else {
				ctx.EnsureDesc(&d272)
				ctx.EmitStoreToStack(d272, int32(bbs[7].PhiBase)+int32(16))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			}
			if d263.Loc == scm.LocReg {
				ctx.UnprotectReg(d263.Reg)
			} else if d263.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d263.Reg)
				ctx.UnprotectReg(d263.Reg2)
			}
			if d269.Loc == scm.LocReg {
				ctx.UnprotectReg(d269.Reg)
			} else if d269.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d269.Reg)
				ctx.UnprotectReg(d269.Reg2)
			}
		}
		ps273 := scm.PhiState{General: ps.General}
		ps273.OverlayValues = make([]scm.JITValueDesc, 273)
		ps273.OverlayValues[1] = d1
		ps273.OverlayValues[2] = d2
		ps273.OverlayValues[3] = d3
		ps273.OverlayValues[4] = d4
		ps273.OverlayValues[5] = d5
		ps273.OverlayValues[6] = d6
		ps273.OverlayValues[7] = d7
		ps273.OverlayValues[8] = d8
		ps273.OverlayValues[31] = d31
		ps273.OverlayValues[32] = d32
		ps273.OverlayValues[33] = d33
		ps273.OverlayValues[34] = d34
		ps273.OverlayValues[35] = d35
		ps273.OverlayValues[36] = d36
		ps273.OverlayValues[37] = d37
		ps273.OverlayValues[38] = d38
		ps273.OverlayValues[77] = d77
		ps273.OverlayValues[78] = d78
		ps273.OverlayValues[79] = d79
		ps273.OverlayValues[80] = d80
		ps273.OverlayValues[83] = d83
		ps273.OverlayValues[108] = d108
		ps273.OverlayValues[132] = d132
		ps273.OverlayValues[133] = d133
		ps273.OverlayValues[134] = d134
		ps273.OverlayValues[136] = d136
		ps273.OverlayValues[137] = d137
		ps273.OverlayValues[138] = d138
		ps273.OverlayValues[139] = d139
		ps273.OverlayValues[140] = d140
		ps273.OverlayValues[141] = d141
		ps273.OverlayValues[142] = d142
		ps273.OverlayValues[143] = d143
		ps273.OverlayValues[144] = d144
		ps273.OverlayValues[145] = d145
		ps273.OverlayValues[147] = d147
		ps273.OverlayValues[148] = d148
		ps273.OverlayValues[149] = d149
		ps273.OverlayValues[150] = d150
		ps273.OverlayValues[151] = d151
		ps273.OverlayValues[152] = d152
		ps273.OverlayValues[153] = d153
		ps273.OverlayValues[154] = d154
		ps273.OverlayValues[157] = d157
		ps273.OverlayValues[158] = d158
		ps273.OverlayValues[159] = d159
		ps273.OverlayValues[256] = d256
		ps273.OverlayValues[257] = d257
		ps273.OverlayValues[258] = d258
		ps273.OverlayValues[259] = d259
		ps273.OverlayValues[260] = d260
		ps273.OverlayValues[261] = d261
		ps273.OverlayValues[262] = d262
		ps273.OverlayValues[263] = d263
		ps273.OverlayValues[264] = d264
		ps273.OverlayValues[265] = d265
		ps273.OverlayValues[266] = d266
		ps273.OverlayValues[267] = d267
		ps273.OverlayValues[268] = d268
		ps273.OverlayValues[269] = d269
		ps273.OverlayValues[270] = d270
		ps273.OverlayValues[271] = d271
		ps273.OverlayValues[272] = d272
		ps273.PhiValues = make([]scm.JITValueDesc, 3)
		d274 = d269
		ps273.PhiValues[0] = d274
		d275 = d263
		ps273.PhiValues[1] = d275
		if ps273.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps273)
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
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != scm.LocNone {
			d108 = ps.OverlayValues[108]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
		}
		if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
			d138 = ps.OverlayValues[138]
		}
		if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
			d139 = ps.OverlayValues[139]
		}
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
			d141 = ps.OverlayValues[141]
		}
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
			d145 = ps.OverlayValues[145]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
			d149 = ps.OverlayValues[149]
		}
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
			d151 = ps.OverlayValues[151]
		}
		if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
			d152 = ps.OverlayValues[152]
		}
		if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != scm.LocNone {
			d153 = ps.OverlayValues[153]
		}
		if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != scm.LocNone {
			d154 = ps.OverlayValues[154]
		}
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != scm.LocNone {
			d159 = ps.OverlayValues[159]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != scm.LocNone {
			d259 = ps.OverlayValues[259]
		}
		if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != scm.LocNone {
			d260 = ps.OverlayValues[260]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != scm.LocNone {
			d262 = ps.OverlayValues[262]
		}
		if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != scm.LocNone {
			d263 = ps.OverlayValues[263]
		}
		if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != scm.LocNone {
			d264 = ps.OverlayValues[264]
		}
		if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != scm.LocNone {
			d265 = ps.OverlayValues[265]
		}
		if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != scm.LocNone {
			d266 = ps.OverlayValues[266]
		}
		if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != scm.LocNone {
			d267 = ps.OverlayValues[267]
		}
		if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != scm.LocNone {
			d268 = ps.OverlayValues[268]
		}
		if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != scm.LocNone {
			d269 = ps.OverlayValues[269]
		}
		if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != scm.LocNone {
			d270 = ps.OverlayValues[270]
		}
		if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != scm.LocNone {
			d271 = ps.OverlayValues[271]
		}
		if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != scm.LocNone {
			d272 = ps.OverlayValues[272]
		}
		if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != scm.LocNone {
			d274 = ps.OverlayValues[274]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		ctx.ReclaimUntrackedRegs()
		d276 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d276)
		ctx.BindReg(r1, &d276)
		ctx.EnsureDesc(&d3)
		if d3.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d3, &d276)
		} else {
			switch d3.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d276, d3)
			case scm.TagInt:
				ctx.EmitMakeInt(d276, d3)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d276, d3)
			case scm.TagNil:
				ctx.EmitMakeNil(d276)
			default:
				ctx.EmitMovPairToResult(&d3, &d276)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps277 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps277)
	ctx.MarkLabel(lbl0)
	d278 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d278)
	ctx.BindReg(r1, &d278)
	ctx.EmitMovPairToResult(&d278, &result)
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
