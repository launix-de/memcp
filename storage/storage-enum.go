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
	sz := uint(unsafe.Sizeof(*s))
	sz += 8 * uint(cap(s.data))
	sz += 4 * uint(cap(s.jumpL1))
	sz += 2 * uint(cap(s.jumpL2))
	for i := uint8(0); i < s.k; i++ {
		sz += scm.ComputeSize(s.values[i]) - uint(unsafe.Sizeof(s.values[i]))
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
		if storageValueEqual(s.values[i], val) {
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
	// Dictionary identity must not coerce distinct stored values.
	for j := uint8(0); j < s.k; j++ {
		if storageValueEqual(s.values[j], value) {
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
		// States below the first width are fixed points: remaining symbols
		// are values[0], so random access need not replay that suffix.
		if buffer < s.widths[0] {
			result = s.values[0]
			break
		}
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
				// States below the first width are fixed points: remaining symbols
				// are values[0], so random access need not replay that suffix.
				if buffer < s.widths[0] {
					result = s.values[0]
					break
				}
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
					// States below the first width are fixed points: remaining symbols
					// are values[0], so random access need not replay that suffix.
					if buffer < s.widths[0] {
						result = s.values[0]
						break
					}
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
		// States below the first width are fixed points: remaining symbols
		// are values[0], so random access need not replay that suffix.
		if buffer < s.widths[0] {
			result = s.values[0]
			break
		}
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
//
//jitgen:control-flow-stable recid count target/1 stride
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

//jitgen:control-flow-stable recids/2 target/1 stride
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
	var d9 scm.JITValueDesc
	_ = d9
	var d10 scm.JITValueDesc
	_ = d10
	var d11 scm.JITValueDesc
	_ = d11
	var d12 scm.JITValueDesc
	_ = d12
	var d37 scm.JITValueDesc
	_ = d37
	var d38 scm.JITValueDesc
	_ = d38
	var d39 scm.JITValueDesc
	_ = d39
	var d40 scm.JITValueDesc
	_ = d40
	var d41 scm.JITValueDesc
	_ = d41
	var d42 scm.JITValueDesc
	_ = d42
	var d43 scm.JITValueDesc
	_ = d43
	var d44 scm.JITValueDesc
	_ = d44
	var d85 scm.JITValueDesc
	_ = d85
	var d86 scm.JITValueDesc
	_ = d86
	var d87 scm.JITValueDesc
	_ = d87
	var d88 scm.JITValueDesc
	_ = d88
	var d91 scm.JITValueDesc
	_ = d91
	var d117 scm.JITValueDesc
	_ = d117
	var d142 scm.JITValueDesc
	_ = d142
	var d143 scm.JITValueDesc
	_ = d143
	var d144 scm.JITValueDesc
	_ = d144
	var d146 scm.JITValueDesc
	_ = d146
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
	var d156 scm.JITValueDesc
	_ = d156
	var d157 scm.JITValueDesc
	_ = d157
	var d158 scm.JITValueDesc
	_ = d158
	var d159 scm.JITValueDesc
	_ = d159
	var d160 scm.JITValueDesc
	_ = d160
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d165 scm.JITValueDesc
	_ = d165
	var d167 scm.JITValueDesc
	_ = d167
	var d168 scm.JITValueDesc
	_ = d168
	var d169 scm.JITValueDesc
	_ = d169
	var d170 scm.JITValueDesc
	_ = d170
	var d220 scm.JITValueDesc
	_ = d220
	var d223 scm.JITValueDesc
	_ = d223
	var d275 scm.JITValueDesc
	_ = d275
	var d276 scm.JITValueDesc
	_ = d276
	var d277 scm.JITValueDesc
	_ = d277
	var d278 scm.JITValueDesc
	_ = d278
	var d279 scm.JITValueDesc
	_ = d279
	var d396 scm.JITValueDesc
	_ = d396
	var d397 scm.JITValueDesc
	_ = d397
	var d398 scm.JITValueDesc
	_ = d398
	var d399 scm.JITValueDesc
	_ = d399
	var d400 scm.JITValueDesc
	_ = d400
	var d401 scm.JITValueDesc
	_ = d401
	var d402 scm.JITValueDesc
	_ = d402
	var d404 scm.JITValueDesc
	_ = d404
	var d405 scm.JITValueDesc
	_ = d405
	var d406 scm.JITValueDesc
	_ = d406
	var d407 scm.JITValueDesc
	_ = d407
	var d408 scm.JITValueDesc
	_ = d408
	var d409 scm.JITValueDesc
	_ = d409
	var d410 scm.JITValueDesc
	_ = d410
	var d411 scm.JITValueDesc
	_ = d411
	var d412 scm.JITValueDesc
	_ = d412
	var d413 scm.JITValueDesc
	_ = d413
	var d414 scm.JITValueDesc
	_ = d414
	var d415 scm.JITValueDesc
	_ = d415
	var d416 scm.JITValueDesc
	_ = d416
	var d417 scm.JITValueDesc
	_ = d417
	var d418 scm.JITValueDesc
	_ = d418
	var d419 scm.JITValueDesc
	_ = d419
	var d420 scm.JITValueDesc
	_ = d420
	var d421 scm.JITValueDesc
	_ = d421
	var d423 scm.JITValueDesc
	_ = d423
	var d424 scm.JITValueDesc
	_ = d424
	var d425 scm.JITValueDesc
	_ = d425
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
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
		defer ctx.UnprotectReg(idxPinnedReg)
	}
	phiBase0 := ctx.AllocStack(int32(80))
	var bbs [12]scm.BBDescriptor
	bbs[6].PhiBase = int32(phiBase0) + int32(0)
	bbs[6].PhiCount = uint16(1)
	bbs[7].PhiBase = int32(phiBase0) + int32(16)
	bbs[7].PhiCount = uint16(3)
	bbs[9].PhiBase = int32(phiBase0) + int32(64)
	bbs[9].PhiCount = uint16(1)
	registerHomes1 := ctx.AllocRegisterHomes(scm.JITRegisterPlan{Slots: [16]scm.JITRegisterSlot{{Color: 0, Width: 1, Cost: 17}, {Color: 1, Width: 1, Cost: 17}}, Count: 2})
	defer ctx.ReleaseRegisterHomes(registerHomes1)
	var r0 scm.Reg
	phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
	if phiHomeOK2 {
		r0 = registerHomes1.Registers[0]
	}
	var r1 scm.Reg
	phiHomeOK3 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
	if phiHomeOK3 {
		r1 = registerHomes1.Registers[1]
	}
	d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	_ = d4
	var d5 scm.JITValueDesc
	if phiHomeOK2 {
		d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
	} else {
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	}
	_ = d5
	d6 := scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
	ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(32))
	_ = d6
	var d7 scm.JITValueDesc
	if phiHomeOK3 {
		d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
	} else {
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
	}
	_ = d7
	d8 := scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
	ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(64))
	_ = d8
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
	bbpos_0_10 := int32(-1)
	_ = bbpos_0_10
	lbl11 := ctx.ReserveLabel()
	_ = lbl11
	bbpos_0_11 := int32(-1)
	_ = bbpos_0_11
	lbl12 := ctx.ReserveLabel()
	_ = lbl12
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
			ctx.FlushRegisterMoves()
			bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_0 = bbs[0].Address
			ctx.MarkLabel(lbl1)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d10 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).count)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).count))
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMem(r2, thisptr.Reg, off)
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d10)
		}
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d10)
		ctx.EnsureDescsTogether(&idxInt, &d10)
		var d11 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d10.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) >= uint64(d10.Imm.Int()))}
		} else if d10.Loc == scm.LocImm {
			r3 := ctx.AllocRegExcept(idxInt.Reg)
			if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d10.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r3, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r3, &d11)
		} else if idxInt.Loc == scm.LocImm {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d10.Reg)
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r4, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r4, &d11)
		} else {
			r5 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d10.Reg)
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r5, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r5, &d11)
		}
		ctx.FreeDesc(&d10)
		d12 = d11
		ctx.EnsureDesc(&d12)
		if d12.Loc != scm.LocImm && d12.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d12.Loc == scm.LocImm {
			if d12.Imm.Bool() {
				if ps.General {
				}
				ps13 := scm.PhiState{General: ps.General}
				ps13.OverlayValues = make([]scm.JITValueDesc, 13)
				ps13.OverlayValues[4] = d4
				ps13.OverlayValues[5] = d5
				ps13.OverlayValues[6] = d6
				ps13.OverlayValues[7] = d7
				ps13.OverlayValues[8] = d8
				ps13.OverlayValues[9] = d9
				ps13.OverlayValues[10] = d10
				ps13.OverlayValues[11] = d11
				ps13.OverlayValues[12] = d12
				return bbs[1].RenderPS(ps13)
			}
			if ps.General {
			}
			ps14 := scm.PhiState{General: ps.General}
			ps14.OverlayValues = make([]scm.JITValueDesc, 13)
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
			return bbs[2].RenderPS(ps14)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitJump(d12.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d11)
		snap15 := d4
		snap16 := d5
		snap17 := d6
		snap18 := d7
		snap19 := d8
		snap20 := d9
		snap21 := d10
		snap22 := d11
		snap23 := d12
		alloc24 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc24)
		d4 = snap15
		d5 = snap16
		d6 = snap17
		d7 = snap18
		d8 = snap19
		d9 = snap20
		d10 = snap21
		d11 = snap22
		d12 = snap23
		ctx.RestoreAllocState(alloc24)
		d4 = snap15
		d5 = snap16
		d6 = snap17
		d7 = snap18
		d8 = snap19
		d9 = snap20
		d10 = snap21
		d11 = snap22
		d12 = snap23
		ps25 := scm.PhiState{General: true}
		ps25.OverlayValues = make([]scm.JITValueDesc, 13)
		ps25.OverlayValues[4] = d4
		ps25.OverlayValues[5] = d5
		ps25.OverlayValues[6] = d6
		ps25.OverlayValues[7] = d7
		ps25.OverlayValues[8] = d8
		ps25.OverlayValues[9] = d9
		ps25.OverlayValues[10] = d10
		ps25.OverlayValues[11] = d11
		ps25.OverlayValues[12] = d12
		ps26 := scm.PhiState{General: true}
		ps26.OverlayValues = make([]scm.JITValueDesc, 13)
		ps26.OverlayValues[4] = d4
		ps26.OverlayValues[5] = d5
		ps26.OverlayValues[6] = d6
		ps26.OverlayValues[7] = d7
		ps26.OverlayValues[8] = d8
		ps26.OverlayValues[9] = d9
		ps26.OverlayValues[10] = d10
		ps26.OverlayValues[11] = d11
		ps26.OverlayValues[12] = d12
		snap27 := d4
		snap28 := d5
		snap29 := d6
		snap30 := d7
		snap31 := d8
		snap32 := d9
		snap33 := d10
		snap34 := d11
		snap35 := d12
		alloc36 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps26)
		}
		ctx.RestoreAllocState(alloc36)
		d4 = snap27
		d5 = snap28
		d6 = snap29
		d7 = snap30
		d8 = snap31
		d9 = snap32
		d10 = snap33
		d11 = snap34
		d12 = snap35
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps25)
		}
		return result
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
			ctx.FlushRegisterMoves()
			bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_1 = bbs[1].Address
			ctx.MarkLabel(lbl2)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		ctx.ReclaimUntrackedRegs()
		d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d38 = result
		ctx.EnsureDesc(&d37)
		if d37.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d37, &d38)
		} else {
			switch d37.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d38, d37)
			case scm.TagInt:
				ctx.EmitMakeInt(d38, d37)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d38, d37)
			case scm.TagNil:
				ctx.EmitMakeNil(d38)
			default:
				ctx.EmitMovPairToResult(&d37, &d38)
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
			ctx.FlushRegisterMoves()
			bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_2 = bbs[2].Address
			ctx.MarkLabel(lbl3)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.StabilizeDescForControlFlow(&idxInt)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&idxInt)
		d40 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).findChunk), []scm.JITValueDesc{thisptr, idxInt}, 1)
		d40.NoHeapPointer = true
		ctx.BindReg(d40.Reg, &d40)
		ctx.StabilizeDescForControlFlow(&d40)
		var d41 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d41 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r6 := ctx.AllocReg()
			r7 := ctx.AllocRegExcept(r6)
			r8 := ctx.AllocRegExcept(r6, r7)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r6, thisptr.Reg, off)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+16)
			d41 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r6, Reg2: r7, Reg3: r8}
			ctx.BindReg(r6, &d41)
			ctx.BindReg(r7, &d41)
			ctx.BindReg(r8, &d41)
			ctx.BindReg(r6, &d41)
			ctx.BindReg(r7, &d41)
			ctx.BindReg(r8, &d41)
		}
		var d42 scm.JITValueDesc
		if d41.SliceSizeKnown {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d41.KnownSliceLen))}
		} else if d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d41.StackOff))}
		} else if d41.Loc == scm.LocStackTriple {
			d42 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d41)
			if d41.Loc == scm.LocRegPair || d41.Loc == scm.LocRegTriple {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg2, ID: 0}
			} else if d41.Loc == scm.LocReg {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d40, &d42)
		var d43 scm.JITValueDesc
		if d40.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d40.Imm.Int() >= d42.Imm.Int())}
		} else if d42.Loc == scm.LocImm {
			r9 := ctx.AllocRegExcept(d40.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d40.Reg, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
				ctx.EmitCmpInt64(d40.Reg, scm.RegR11)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r9, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r9, &d43)
		} else if d40.Loc == scm.LocImm {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r10, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r10, &d43)
		} else {
			r11 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitCmpInt64(d40.Reg, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r11, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r11, &d43)
		}
		ctx.FreeDesc(&d42)
		d44 = d43
		ctx.EnsureDesc(&d44)
		if d44.Loc != scm.LocImm && d44.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d44.Loc == scm.LocImm {
			if d44.Imm.Bool() {
				if ps.General {
				}
				ps45 := scm.PhiState{General: ps.General}
				ps45.OverlayValues = make([]scm.JITValueDesc, 45)
				ps45.OverlayValues[4] = d4
				ps45.OverlayValues[5] = d5
				ps45.OverlayValues[6] = d6
				ps45.OverlayValues[7] = d7
				ps45.OverlayValues[8] = d8
				ps45.OverlayValues[9] = d9
				ps45.OverlayValues[10] = d10
				ps45.OverlayValues[11] = d11
				ps45.OverlayValues[12] = d12
				ps45.OverlayValues[37] = d37
				ps45.OverlayValues[38] = d38
				ps45.OverlayValues[39] = d39
				ps45.OverlayValues[40] = d40
				ps45.OverlayValues[41] = d41
				ps45.OverlayValues[42] = d42
				ps45.OverlayValues[43] = d43
				ps45.OverlayValues[44] = d44
				return bbs[3].RenderPS(ps45)
			}
			if ps.General {
			}
			ps46 := scm.PhiState{General: ps.General}
			ps46.OverlayValues = make([]scm.JITValueDesc, 45)
			ps46.OverlayValues[4] = d4
			ps46.OverlayValues[5] = d5
			ps46.OverlayValues[6] = d6
			ps46.OverlayValues[7] = d7
			ps46.OverlayValues[8] = d8
			ps46.OverlayValues[9] = d9
			ps46.OverlayValues[10] = d10
			ps46.OverlayValues[11] = d11
			ps46.OverlayValues[12] = d12
			ps46.OverlayValues[37] = d37
			ps46.OverlayValues[38] = d38
			ps46.OverlayValues[39] = d39
			ps46.OverlayValues[40] = d40
			ps46.OverlayValues[41] = d41
			ps46.OverlayValues[42] = d42
			ps46.OverlayValues[43] = d43
			ps46.OverlayValues[44] = d44
			return bbs[4].RenderPS(ps46)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d44.Condition, lbl4)
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
		}
		ctx.FreeDesc(&d43)
		snap47 := d4
		snap48 := d5
		snap49 := d6
		snap50 := d7
		snap51 := d8
		snap52 := d9
		snap53 := d10
		snap54 := d11
		snap55 := d12
		snap56 := d37
		snap57 := d38
		snap58 := d39
		snap59 := d40
		snap60 := d41
		snap61 := d42
		snap62 := d43
		snap63 := d44
		alloc64 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc64)
		d4 = snap47
		d5 = snap48
		d6 = snap49
		d7 = snap50
		d8 = snap51
		d9 = snap52
		d10 = snap53
		d11 = snap54
		d12 = snap55
		d37 = snap56
		d38 = snap57
		d39 = snap58
		d40 = snap59
		d41 = snap60
		d42 = snap61
		d43 = snap62
		d44 = snap63
		ctx.RestoreAllocState(alloc64)
		d4 = snap47
		d5 = snap48
		d6 = snap49
		d7 = snap50
		d8 = snap51
		d9 = snap52
		d10 = snap53
		d11 = snap54
		d12 = snap55
		d37 = snap56
		d38 = snap57
		d39 = snap58
		d40 = snap59
		d41 = snap60
		d42 = snap61
		d43 = snap62
		d44 = snap63
		ps65 := scm.PhiState{General: true}
		ps65.OverlayValues = make([]scm.JITValueDesc, 45)
		ps65.OverlayValues[4] = d4
		ps65.OverlayValues[5] = d5
		ps65.OverlayValues[6] = d6
		ps65.OverlayValues[7] = d7
		ps65.OverlayValues[8] = d8
		ps65.OverlayValues[9] = d9
		ps65.OverlayValues[10] = d10
		ps65.OverlayValues[11] = d11
		ps65.OverlayValues[12] = d12
		ps65.OverlayValues[37] = d37
		ps65.OverlayValues[38] = d38
		ps65.OverlayValues[39] = d39
		ps65.OverlayValues[40] = d40
		ps65.OverlayValues[41] = d41
		ps65.OverlayValues[42] = d42
		ps65.OverlayValues[43] = d43
		ps65.OverlayValues[44] = d44
		ps66 := scm.PhiState{General: true}
		ps66.OverlayValues = make([]scm.JITValueDesc, 45)
		ps66.OverlayValues[4] = d4
		ps66.OverlayValues[5] = d5
		ps66.OverlayValues[6] = d6
		ps66.OverlayValues[7] = d7
		ps66.OverlayValues[8] = d8
		ps66.OverlayValues[9] = d9
		ps66.OverlayValues[10] = d10
		ps66.OverlayValues[11] = d11
		ps66.OverlayValues[12] = d12
		ps66.OverlayValues[37] = d37
		ps66.OverlayValues[38] = d38
		ps66.OverlayValues[39] = d39
		ps66.OverlayValues[40] = d40
		ps66.OverlayValues[41] = d41
		ps66.OverlayValues[42] = d42
		ps66.OverlayValues[43] = d43
		ps66.OverlayValues[44] = d44
		snap67 := d4
		snap68 := d5
		snap69 := d6
		snap70 := d7
		snap71 := d8
		snap72 := d9
		snap73 := d10
		snap74 := d11
		snap75 := d12
		snap76 := d37
		snap77 := d38
		snap78 := d39
		snap79 := d40
		snap80 := d41
		snap81 := d42
		snap82 := d43
		snap83 := d44
		alloc84 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps66)
		}
		ctx.RestoreAllocState(alloc84)
		d4 = snap67
		d5 = snap68
		d6 = snap69
		d7 = snap70
		d8 = snap71
		d9 = snap72
		d10 = snap73
		d11 = snap74
		d12 = snap75
		d37 = snap76
		d38 = snap77
		d39 = snap78
		d40 = snap79
		d41 = snap80
		d42 = snap81
		d43 = snap82
		d44 = snap83
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps65)
		}
		return result
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
			ctx.FlushRegisterMoves()
			bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_3 = bbs[3].Address
			ctx.MarkLabel(lbl4)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		ctx.ReclaimUntrackedRegs()
		d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d86 = result
		ctx.EnsureDesc(&d85)
		if d85.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d85, &d86)
		} else {
			switch d85.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d86, d85)
			case scm.TagInt:
				ctx.EmitMakeInt(d86, d85)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d86, d85)
			case scm.TagNil:
				ctx.EmitMakeNil(d86)
			default:
				ctx.EmitMovPairToResult(&d85, &d86)
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
			ctx.FlushRegisterMoves()
			bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_4 = bbs[4].Address
			ctx.MarkLabel(lbl5)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		var d87 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d40.Imm.Int() > 0)}
		} else {
			r12 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitCmpRegImm32(d40.Reg, 0)
			d87 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r12, Condition: scm.CondSignedGreater}
			ctx.BindReg(r12, &d87)
		}
		d88 = d87
		ctx.EnsureDesc(&d88)
		if d88.Loc != scm.LocImm && d88.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d88.Loc == scm.LocImm {
			if d88.Imm.Bool() {
				if ps.General {
				}
				ps89 := scm.PhiState{General: ps.General}
				ps89.OverlayValues = make([]scm.JITValueDesc, 89)
				ps89.OverlayValues[4] = d4
				ps89.OverlayValues[5] = d5
				ps89.OverlayValues[6] = d6
				ps89.OverlayValues[7] = d7
				ps89.OverlayValues[8] = d8
				ps89.OverlayValues[9] = d9
				ps89.OverlayValues[10] = d10
				ps89.OverlayValues[11] = d11
				ps89.OverlayValues[12] = d12
				ps89.OverlayValues[37] = d37
				ps89.OverlayValues[38] = d38
				ps89.OverlayValues[39] = d39
				ps89.OverlayValues[40] = d40
				ps89.OverlayValues[41] = d41
				ps89.OverlayValues[42] = d42
				ps89.OverlayValues[43] = d43
				ps89.OverlayValues[44] = d44
				ps89.OverlayValues[85] = d85
				ps89.OverlayValues[86] = d86
				ps89.OverlayValues[87] = d87
				ps89.OverlayValues[88] = d88
				return bbs[5].RenderPS(ps89)
			}
			if ps.General {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
			}
			ps90 := scm.PhiState{General: ps.General}
			ps90.OverlayValues = make([]scm.JITValueDesc, 89)
			ps90.OverlayValues[4] = d4
			ps90.OverlayValues[5] = d5
			ps90.OverlayValues[6] = d6
			ps90.OverlayValues[7] = d7
			ps90.OverlayValues[8] = d8
			ps90.OverlayValues[9] = d9
			ps90.OverlayValues[10] = d10
			ps90.OverlayValues[11] = d11
			ps90.OverlayValues[12] = d12
			ps90.OverlayValues[37] = d37
			ps90.OverlayValues[38] = d38
			ps90.OverlayValues[39] = d39
			ps90.OverlayValues[40] = d40
			ps90.OverlayValues[41] = d41
			ps90.OverlayValues[42] = d42
			ps90.OverlayValues[43] = d43
			ps90.OverlayValues[44] = d44
			ps90.OverlayValues[85] = d85
			ps90.OverlayValues[86] = d86
			ps90.OverlayValues[87] = d87
			ps90.OverlayValues[88] = d88
			ps90.PhiValues = make([]scm.JITValueDesc, 1)
			d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps90.PhiValues[0] = d91
			return bbs[6].RenderPS(ps90)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl13 := ctx.ReserveLabel()
		ctx.EmitJump(d88.Condition, lbl6)
		ctx.EmitJmp(lbl13)
		ctx.FreeDesc(&d87)
		snap92 := d4
		snap93 := d5
		snap94 := d6
		snap95 := d7
		snap96 := d8
		snap97 := d9
		snap98 := d10
		snap99 := d11
		snap100 := d12
		snap101 := d37
		snap102 := d38
		snap103 := d39
		snap104 := d40
		snap105 := d41
		snap106 := d42
		snap107 := d43
		snap108 := d44
		snap109 := d85
		snap110 := d86
		snap111 := d87
		snap112 := d88
		snap113 := d91
		alloc114 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc114)
		d4 = snap92
		d5 = snap93
		d6 = snap94
		d7 = snap95
		d8 = snap96
		d9 = snap97
		d10 = snap98
		d11 = snap99
		d12 = snap100
		d37 = snap101
		d38 = snap102
		d39 = snap103
		d40 = snap104
		d41 = snap105
		d42 = snap106
		d43 = snap107
		d44 = snap108
		d85 = snap109
		d86 = snap110
		d87 = snap111
		d88 = snap112
		d91 = snap113
		ctx.MarkLabel(lbl13)
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc114)
		d4 = snap92
		d5 = snap93
		d6 = snap94
		d7 = snap95
		d8 = snap96
		d9 = snap97
		d10 = snap98
		d11 = snap99
		d12 = snap100
		d37 = snap101
		d38 = snap102
		d39 = snap103
		d40 = snap104
		d41 = snap105
		d42 = snap106
		d43 = snap107
		d44 = snap108
		d85 = snap109
		d86 = snap110
		d87 = snap111
		d88 = snap112
		d91 = snap113
		ps115 := scm.PhiState{General: true}
		ps115.OverlayValues = make([]scm.JITValueDesc, 92)
		ps115.OverlayValues[4] = d4
		ps115.OverlayValues[5] = d5
		ps115.OverlayValues[6] = d6
		ps115.OverlayValues[7] = d7
		ps115.OverlayValues[8] = d8
		ps115.OverlayValues[9] = d9
		ps115.OverlayValues[10] = d10
		ps115.OverlayValues[11] = d11
		ps115.OverlayValues[12] = d12
		ps115.OverlayValues[37] = d37
		ps115.OverlayValues[38] = d38
		ps115.OverlayValues[39] = d39
		ps115.OverlayValues[40] = d40
		ps115.OverlayValues[41] = d41
		ps115.OverlayValues[42] = d42
		ps115.OverlayValues[43] = d43
		ps115.OverlayValues[44] = d44
		ps115.OverlayValues[85] = d85
		ps115.OverlayValues[86] = d86
		ps115.OverlayValues[87] = d87
		ps115.OverlayValues[88] = d88
		ps115.OverlayValues[91] = d91
		ps116 := scm.PhiState{General: true}
		ps116.OverlayValues = make([]scm.JITValueDesc, 92)
		ps116.OverlayValues[4] = d4
		ps116.OverlayValues[5] = d5
		ps116.OverlayValues[6] = d6
		ps116.OverlayValues[7] = d7
		ps116.OverlayValues[8] = d8
		ps116.OverlayValues[9] = d9
		ps116.OverlayValues[10] = d10
		ps116.OverlayValues[11] = d11
		ps116.OverlayValues[12] = d12
		ps116.OverlayValues[37] = d37
		ps116.OverlayValues[38] = d38
		ps116.OverlayValues[39] = d39
		ps116.OverlayValues[40] = d40
		ps116.OverlayValues[41] = d41
		ps116.OverlayValues[42] = d42
		ps116.OverlayValues[43] = d43
		ps116.OverlayValues[44] = d44
		ps116.OverlayValues[85] = d85
		ps116.OverlayValues[86] = d86
		ps116.OverlayValues[87] = d87
		ps116.OverlayValues[88] = d88
		ps116.OverlayValues[91] = d91
		ps116.PhiValues = make([]scm.JITValueDesc, 1)
		d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps116.PhiValues[0] = d117
		snap118 := d4
		snap119 := d5
		snap120 := d6
		snap121 := d7
		snap122 := d8
		snap123 := d9
		snap124 := d10
		snap125 := d11
		snap126 := d12
		snap127 := d37
		snap128 := d38
		snap129 := d39
		snap130 := d40
		snap131 := d41
		snap132 := d42
		snap133 := d43
		snap134 := d44
		snap135 := d85
		snap136 := d86
		snap137 := d87
		snap138 := d88
		snap139 := d91
		snap140 := d117
		alloc141 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps116)
		}
		ctx.RestoreAllocState(alloc141)
		d4 = snap118
		d5 = snap119
		d6 = snap120
		d7 = snap121
		d8 = snap122
		d9 = snap123
		d10 = snap124
		d11 = snap125
		d12 = snap126
		d37 = snap127
		d38 = snap128
		d39 = snap129
		d40 = snap130
		d41 = snap131
		d42 = snap132
		d43 = snap133
		d44 = snap134
		d85 = snap135
		d86 = snap136
		d87 = snap137
		d88 = snap138
		d91 = snap139
		d117 = snap140
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps115)
		}
		return result
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
			ctx.FlushRegisterMoves()
			bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_5 = bbs[5].Address
			ctx.MarkLabel(lbl6)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d40)
		var d142 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, 1)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d142)
		}
		if d142.Loc == scm.LocReg && d40.Loc == scm.LocReg && d142.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d142.Loc == scm.LocRegPair || d142.Loc == scm.LocStackPair || d142.Loc == scm.LocRegTriple || d142.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d142)
		d143 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).jumpCum), []scm.JITValueDesc{thisptr, d142}, 1)
		d143.NoHeapPointer = true
		ctx.BindReg(d143.Reg, &d143)
		ctx.StabilizeDescForControlFlow(&d143)
		ctx.FreeDesc(&d142)
		if ps.General {
			ctx.SyncDesc(&d143)
			if d143.Loc == scm.LocReg || d143.Loc == scm.LocFPReg {
				ctx.ProtectReg(d143.Reg)
			} else if d143.Loc == scm.LocRegPair {
				ctx.ProtectReg(d143.Reg)
				ctx.ProtectReg(d143.Reg2)
			}
			d144 = d143
			if d144.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d144)
			ctx.EmitStoreToStack(d144, int32(bbs[6].PhiBase)+int32(0))
			if d143.Loc == scm.LocReg || d143.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d143.Reg)
			} else if d143.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d143.Reg)
				ctx.UnprotectReg(d143.Reg2)
			}
		}
		ps145 := scm.PhiState{General: ps.General}
		ps145.OverlayValues = make([]scm.JITValueDesc, 145)
		ps145.OverlayValues[4] = d4
		ps145.OverlayValues[5] = d5
		ps145.OverlayValues[6] = d6
		ps145.OverlayValues[7] = d7
		ps145.OverlayValues[8] = d8
		ps145.OverlayValues[9] = d9
		ps145.OverlayValues[10] = d10
		ps145.OverlayValues[11] = d11
		ps145.OverlayValues[12] = d12
		ps145.OverlayValues[37] = d37
		ps145.OverlayValues[38] = d38
		ps145.OverlayValues[39] = d39
		ps145.OverlayValues[40] = d40
		ps145.OverlayValues[41] = d41
		ps145.OverlayValues[42] = d42
		ps145.OverlayValues[43] = d43
		ps145.OverlayValues[44] = d44
		ps145.OverlayValues[85] = d85
		ps145.OverlayValues[86] = d86
		ps145.OverlayValues[87] = d87
		ps145.OverlayValues[88] = d88
		ps145.OverlayValues[91] = d91
		ps145.OverlayValues[117] = d117
		ps145.OverlayValues[142] = d142
		ps145.OverlayValues[143] = d143
		ps145.OverlayValues[144] = d144
		ps145.PhiValues = make([]scm.JITValueDesc, 1)
		d146 = d143
		ps145.PhiValues[0] = d146
		if ps145.General && bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		return bbs[6].RenderPS(ps145)
		return result
	}
	bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d147 := ps.PhiValues[0]
				ctx.EnsureDesc(&d147)
				ctx.EmitStoreToStack(d147, int32(bbs[6].PhiBase)+int32(0))
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
			ctx.FlushRegisterMoves()
			bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_6 = bbs[6].Address
			ctx.MarkLabel(lbl7)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		var d148 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d148 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r13 := ctx.AllocReg()
			r14 := ctx.AllocRegExcept(r13)
			r15 := ctx.AllocRegExcept(r13, r14)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r13, thisptr.Reg, off)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+16)
			d148 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r13, Reg2: r14, Reg3: r15}
			ctx.BindReg(r13, &d148)
			ctx.BindReg(r14, &d148)
			ctx.BindReg(r15, &d148)
			ctx.BindReg(r13, &d148)
			ctx.BindReg(r14, &d148)
			ctx.BindReg(r15, &d148)
		}
		var d149 scm.JITValueDesc
		if d148.SliceSizeKnown {
			d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d148.KnownSliceLen))}
		} else if d148.Loc == scm.LocImm {
			d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d148.StackOff))}
		} else if d148.Loc == scm.LocStackTriple {
			d149 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d148.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d148)
			if d148.Loc == scm.LocRegPair || d148.Loc == scm.LocRegTriple {
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d148.Reg2, ID: 0}
			} else if d148.Loc == scm.LocReg {
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d148.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d149)
		ctx.EnsureDesc(&d149)
		var d150 scm.JITValueDesc
		if d149.Loc == scm.LocImm {
			d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d149.Reg)
			ctx.EmitMovRegReg(scratch, d149.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, 1)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d150)
		}
		if d150.Loc == scm.LocReg && d149.Loc == scm.LocReg && d150.Reg == d149.Reg {
			ctx.TransferReg(d149.Reg)
			d149.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d149)
		ctx.EnsureDesc(&d150)
		ctx.EnsureDesc(&d40)
		ctx.EnsureDescsTogether(&d150, &d40)
		var d151 scm.JITValueDesc
		if d150.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() - d40.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			ctx.EnsureDesc(&d150)
			r16 := ctx.AllocRegExcept(d150.Reg)
			ctx.EmitMovRegReg(r16, d150.Reg)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d151)
		} else if d150.Loc == scm.LocImm {
			ctx.EnsureDesc(&d40)
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d150.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d40)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d151)
		} else if d40.Loc == scm.LocImm {
			ctx.EnsureDesc(&d150)
			scratch := ctx.AllocRegExcept(d150.Reg)
			ctx.EmitMovRegReg(scratch, d150.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d40.Imm.Int())
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d151)
		} else {
			ctx.EnsureDesc(&d150)
			ctx.SyncDesc(&d40)
			r17 := ctx.AllocRegExcept(d150.Reg, d40.Reg)
			ctx.EmitMovRegReg(r17, d150.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r17, &d40)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d151)
		}
		if d151.Loc == scm.LocReg && d150.Loc == scm.LocReg && d151.Reg == d150.Reg {
			ctx.TransferReg(d150.Reg)
			d150.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d150)
		ctx.EnsureDesc(&d151)
		d152 = ctx.EmitLoadScalarSliceElement(&d148, &d151, 8, scm.TagInt)
		ctx.FreeDesc(&d151)
		ctx.StabilizeDescForControlFlow(&d152)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDescsTogether(&idxInt, &d4)
		var d153 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d4.Loc == scm.LocImm {
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d4.Imm.Int())}
		} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
			ctx.EnsureDesc(&idxInt)
			r18 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r18, idxInt.Reg)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d153)
		} else if idxInt.Loc == scm.LocImm {
			ctx.EnsureDesc(&d4)
			scratch := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d4)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d153)
		} else if d4.Loc == scm.LocImm {
			ctx.EnsureDesc(&idxInt)
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d4.Imm.Int())
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d153)
		} else {
			ctx.EnsureDesc(&idxInt)
			ctx.SyncDesc(&d4)
			r19 := ctx.AllocRegExcept(idxInt.Reg, d4.Reg)
			ctx.EmitMovRegReg(r19, idxInt.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r19, &d4)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d153)
		}
		if d153.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d153.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d153)
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d4)
		if ps.General {
			ctx.SyncDesc(&d152)
			if d152.Loc == scm.LocReg || d152.Loc == scm.LocFPReg {
				ctx.ProtectReg(d152.Reg)
			} else if d152.Loc == scm.LocRegPair {
				ctx.ProtectReg(d152.Reg)
				ctx.ProtectReg(d152.Reg2)
			}
			d154 = d152
			if d154.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d154)
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d154)
			} else {
				ctx.EmitStoreToStack(d154, int32(bbs[7].PhiBase)+int32(0))
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
			} else {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(32))
			}
			if d152.Loc == scm.LocReg || d152.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d152.Reg)
			} else if d152.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d152.Reg)
				ctx.UnprotectReg(d152.Reg2)
			}
		}
		ps155 := scm.PhiState{General: ps.General}
		ps155.OverlayValues = make([]scm.JITValueDesc, 155)
		ps155.OverlayValues[4] = d4
		ps155.OverlayValues[5] = d5
		ps155.OverlayValues[6] = d6
		ps155.OverlayValues[7] = d7
		ps155.OverlayValues[8] = d8
		ps155.OverlayValues[9] = d9
		ps155.OverlayValues[10] = d10
		ps155.OverlayValues[11] = d11
		ps155.OverlayValues[12] = d12
		ps155.OverlayValues[37] = d37
		ps155.OverlayValues[38] = d38
		ps155.OverlayValues[39] = d39
		ps155.OverlayValues[40] = d40
		ps155.OverlayValues[41] = d41
		ps155.OverlayValues[42] = d42
		ps155.OverlayValues[43] = d43
		ps155.OverlayValues[44] = d44
		ps155.OverlayValues[85] = d85
		ps155.OverlayValues[86] = d86
		ps155.OverlayValues[87] = d87
		ps155.OverlayValues[88] = d88
		ps155.OverlayValues[91] = d91
		ps155.OverlayValues[117] = d117
		ps155.OverlayValues[142] = d142
		ps155.OverlayValues[143] = d143
		ps155.OverlayValues[144] = d144
		ps155.OverlayValues[146] = d146
		ps155.OverlayValues[147] = d147
		ps155.OverlayValues[148] = d148
		ps155.OverlayValues[149] = d149
		ps155.OverlayValues[150] = d150
		ps155.OverlayValues[151] = d151
		ps155.OverlayValues[152] = d152
		ps155.OverlayValues[153] = d153
		ps155.OverlayValues[154] = d154
		ps155.PhiValues = make([]scm.JITValueDesc, 3)
		d156 = d152
		ps155.PhiValues[0] = d156
		d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ps155.PhiValues[1] = d157
		d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps155.PhiValues[2] = d158
		if ps155.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps155)
		return result
	}
	bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d159 := ps.PhiValues[0]
				ctx.EnsureDesc(&d159)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d159)
				} else {
					ctx.EmitStoreToStack(d159, int32(bbs[7].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d160 := ps.PhiValues[1]
				ctx.EnsureDesc(&d160)
				ctx.EmitStoreScmerToStack(d160, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d161 := ps.PhiValues[2]
				ctx.EnsureDesc(&d161)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d161)
				} else {
					ctx.EmitStoreToStack(d161, int32(bbs[7].PhiBase)+int32(32))
				}
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
			ctx.FlushRegisterMoves()
			bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_7 = bbs[7].Address
			ctx.MarkLabel(lbl8)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
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
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d5 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d6 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d7 = ps.PhiValues[2]
		}
		if phiHomeOK2 && d5.Loc == scm.LocReg {
			ctx.BindReg(r0, &d5)
		}
		if phiHomeOK3 && d7.Loc == scm.LocReg {
			ctx.BindReg(r1, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d153)
		ctx.EnsureDescsTogether(&d7, &d153)
		var d162 scm.JITValueDesc
		if d7.Loc == scm.LocImm && d153.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d7.Imm.Int() <= d153.Imm.Int())}
		} else if d153.Loc == scm.LocImm {
			r20 := ctx.AllocRegExcept(d7.Reg)
			if d153.Imm.Int() >= -2147483648 && d153.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d7.Reg, int32(d153.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d153.Imm.Int()))
				ctx.EmitCmpInt64(d7.Reg, scm.RegR11)
			}
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r20, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r20, &d162)
		} else if d7.Loc == scm.LocImm {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d153.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r21, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r21, &d162)
		} else {
			r22 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpInt64(d7.Reg, d153.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r22, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r22, &d162)
		}
		d163 = d162
		ctx.EnsureDesc(&d163)
		if d163.Loc != scm.LocImm && d163.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d163.Loc == scm.LocImm {
			if d163.Imm.Bool() {
				if ps.General {
				}
				ps164 := scm.PhiState{General: ps.General}
				ps164.OverlayValues = make([]scm.JITValueDesc, 164)
				ps164.OverlayValues[4] = d4
				ps164.OverlayValues[5] = d5
				ps164.OverlayValues[6] = d6
				ps164.OverlayValues[7] = d7
				ps164.OverlayValues[8] = d8
				ps164.OverlayValues[9] = d9
				ps164.OverlayValues[10] = d10
				ps164.OverlayValues[11] = d11
				ps164.OverlayValues[12] = d12
				ps164.OverlayValues[37] = d37
				ps164.OverlayValues[38] = d38
				ps164.OverlayValues[39] = d39
				ps164.OverlayValues[40] = d40
				ps164.OverlayValues[41] = d41
				ps164.OverlayValues[42] = d42
				ps164.OverlayValues[43] = d43
				ps164.OverlayValues[44] = d44
				ps164.OverlayValues[85] = d85
				ps164.OverlayValues[86] = d86
				ps164.OverlayValues[87] = d87
				ps164.OverlayValues[88] = d88
				ps164.OverlayValues[91] = d91
				ps164.OverlayValues[117] = d117
				ps164.OverlayValues[142] = d142
				ps164.OverlayValues[143] = d143
				ps164.OverlayValues[144] = d144
				ps164.OverlayValues[146] = d146
				ps164.OverlayValues[147] = d147
				ps164.OverlayValues[148] = d148
				ps164.OverlayValues[149] = d149
				ps164.OverlayValues[150] = d150
				ps164.OverlayValues[151] = d151
				ps164.OverlayValues[152] = d152
				ps164.OverlayValues[153] = d153
				ps164.OverlayValues[154] = d154
				ps164.OverlayValues[156] = d156
				ps164.OverlayValues[157] = d157
				ps164.OverlayValues[158] = d158
				ps164.OverlayValues[159] = d159
				ps164.OverlayValues[160] = d160
				ps164.OverlayValues[161] = d161
				ps164.OverlayValues[162] = d162
				ps164.OverlayValues[163] = d163
				return bbs[8].RenderPS(ps164)
			}
			if ps.General {
				ctx.SyncDesc(&d6)
				if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
					ctx.ProtectReg(d6.Reg)
				} else if d6.Loc == scm.LocRegPair {
					ctx.ProtectReg(d6.Reg)
					ctx.ProtectReg(d6.Reg2)
				}
				d165 = d6
				if d165.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d165)
				if d165.Loc == scm.LocStackPair {
					ctx.EmitCopyStackWords(d165, int32(bbs[9].PhiBase)+int32(0), 2)
				} else if d165.Loc == scm.LocInputPair {
					ctx.EnsureDesc(&d165)
					ctx.EmitStoreScmerToStack(d165, int32(bbs[9].PhiBase)+int32(0))
				} else if d165.Loc == scm.LocRegPair || d165.Loc == scm.LocImm {
					ctx.EmitStoreScmerToStack(d165, int32(bbs[9].PhiBase)+int32(0))
				} else {
					ctx.EnsureDesc(&d165)
					ctx.EmitStoreToStack(d165, int32(bbs[9].PhiBase)+int32(0))
					ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
				}
				if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d6.Reg)
				} else if d6.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d6.Reg)
					ctx.UnprotectReg(d6.Reg2)
				}
			}
			ps166 := scm.PhiState{General: ps.General}
			ps166.OverlayValues = make([]scm.JITValueDesc, 166)
			ps166.OverlayValues[4] = d4
			ps166.OverlayValues[5] = d5
			ps166.OverlayValues[6] = d6
			ps166.OverlayValues[7] = d7
			ps166.OverlayValues[8] = d8
			ps166.OverlayValues[9] = d9
			ps166.OverlayValues[10] = d10
			ps166.OverlayValues[11] = d11
			ps166.OverlayValues[12] = d12
			ps166.OverlayValues[37] = d37
			ps166.OverlayValues[38] = d38
			ps166.OverlayValues[39] = d39
			ps166.OverlayValues[40] = d40
			ps166.OverlayValues[41] = d41
			ps166.OverlayValues[42] = d42
			ps166.OverlayValues[43] = d43
			ps166.OverlayValues[44] = d44
			ps166.OverlayValues[85] = d85
			ps166.OverlayValues[86] = d86
			ps166.OverlayValues[87] = d87
			ps166.OverlayValues[88] = d88
			ps166.OverlayValues[91] = d91
			ps166.OverlayValues[117] = d117
			ps166.OverlayValues[142] = d142
			ps166.OverlayValues[143] = d143
			ps166.OverlayValues[144] = d144
			ps166.OverlayValues[146] = d146
			ps166.OverlayValues[147] = d147
			ps166.OverlayValues[148] = d148
			ps166.OverlayValues[149] = d149
			ps166.OverlayValues[150] = d150
			ps166.OverlayValues[151] = d151
			ps166.OverlayValues[152] = d152
			ps166.OverlayValues[153] = d153
			ps166.OverlayValues[154] = d154
			ps166.OverlayValues[156] = d156
			ps166.OverlayValues[157] = d157
			ps166.OverlayValues[158] = d158
			ps166.OverlayValues[159] = d159
			ps166.OverlayValues[160] = d160
			ps166.OverlayValues[161] = d161
			ps166.OverlayValues[162] = d162
			ps166.OverlayValues[163] = d163
			ps166.OverlayValues[165] = d165
			ps166.PhiValues = make([]scm.JITValueDesc, 1)
			d167 = d6
			ps166.PhiValues[0] = d167
			return bbs[9].RenderPS(ps166)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d168 := ps.PhiValues[0]
				ctx.EnsureDesc(&d168)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d168)
				} else {
					ctx.EmitStoreToStack(d168, int32(bbs[7].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d169 := ps.PhiValues[1]
				ctx.EnsureDesc(&d169)
				ctx.EmitStoreScmerToStack(d169, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d170 := ps.PhiValues[2]
				ctx.EnsureDesc(&d170)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d170)
				} else {
					ctx.EmitStoreToStack(d170, int32(bbs[7].PhiBase)+int32(32))
				}
			}
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		ctx.EmitJump(d163.Condition, lbl9)
		ctx.EmitJmp(lbl14)
		ctx.FreeDesc(&d162)
		snap171 := d4
		snap172 := d5
		snap173 := d6
		snap174 := d7
		snap175 := d8
		snap176 := d9
		snap177 := d10
		snap178 := d11
		snap179 := d12
		snap180 := d37
		snap181 := d38
		snap182 := d39
		snap183 := d40
		snap184 := d41
		snap185 := d42
		snap186 := d43
		snap187 := d44
		snap188 := d85
		snap189 := d86
		snap190 := d87
		snap191 := d88
		snap192 := d91
		snap193 := d117
		snap194 := d142
		snap195 := d143
		snap196 := d144
		snap197 := d146
		snap198 := d147
		snap199 := d148
		snap200 := d149
		snap201 := d150
		snap202 := d151
		snap203 := d152
		snap204 := d153
		snap205 := d154
		snap206 := d156
		snap207 := d157
		snap208 := d158
		snap209 := d159
		snap210 := d160
		snap211 := d161
		snap212 := d162
		snap213 := d163
		snap214 := d165
		snap215 := d167
		snap216 := d168
		snap217 := d169
		snap218 := d170
		alloc219 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc219)
		d4 = snap171
		d5 = snap172
		d6 = snap173
		d7 = snap174
		d8 = snap175
		d9 = snap176
		d10 = snap177
		d11 = snap178
		d12 = snap179
		d37 = snap180
		d38 = snap181
		d39 = snap182
		d40 = snap183
		d41 = snap184
		d42 = snap185
		d43 = snap186
		d44 = snap187
		d85 = snap188
		d86 = snap189
		d87 = snap190
		d88 = snap191
		d91 = snap192
		d117 = snap193
		d142 = snap194
		d143 = snap195
		d144 = snap196
		d146 = snap197
		d147 = snap198
		d148 = snap199
		d149 = snap200
		d150 = snap201
		d151 = snap202
		d152 = snap203
		d153 = snap204
		d154 = snap205
		d156 = snap206
		d157 = snap207
		d158 = snap208
		d159 = snap209
		d160 = snap210
		d161 = snap211
		d162 = snap212
		d163 = snap213
		d165 = snap214
		d167 = snap215
		d168 = snap216
		d169 = snap217
		d170 = snap218
		ctx.MarkLabel(lbl14)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d220 = d6
		if d220.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.SyncDesc(&d220)
		if d220.Loc == scm.LocStackPair {
			ctx.EmitCopyStackWords(d220, int32(bbs[9].PhiBase)+int32(0), 2)
		} else if d220.Loc == scm.LocInputPair {
			ctx.EnsureDesc(&d220)
			ctx.EmitStoreScmerToStack(d220, int32(bbs[9].PhiBase)+int32(0))
		} else if d220.Loc == scm.LocRegPair || d220.Loc == scm.LocImm {
			ctx.EmitStoreScmerToStack(d220, int32(bbs[9].PhiBase)+int32(0))
		} else {
			ctx.EnsureDesc(&d220)
			ctx.EmitStoreToStack(d220, int32(bbs[9].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
		}
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl10)
		ctx.RestoreAllocState(alloc219)
		d4 = snap171
		d5 = snap172
		d6 = snap173
		d7 = snap174
		d8 = snap175
		d9 = snap176
		d10 = snap177
		d11 = snap178
		d12 = snap179
		d37 = snap180
		d38 = snap181
		d39 = snap182
		d40 = snap183
		d41 = snap184
		d42 = snap185
		d43 = snap186
		d44 = snap187
		d85 = snap188
		d86 = snap189
		d87 = snap190
		d88 = snap191
		d91 = snap192
		d117 = snap193
		d142 = snap194
		d143 = snap195
		d144 = snap196
		d146 = snap197
		d147 = snap198
		d148 = snap199
		d149 = snap200
		d150 = snap201
		d151 = snap202
		d152 = snap203
		d153 = snap204
		d154 = snap205
		d156 = snap206
		d157 = snap207
		d158 = snap208
		d159 = snap209
		d160 = snap210
		d161 = snap211
		d162 = snap212
		d163 = snap213
		d165 = snap214
		d167 = snap215
		d168 = snap216
		d169 = snap217
		d170 = snap218
		ps221 := scm.PhiState{General: true}
		ps221.OverlayValues = make([]scm.JITValueDesc, 221)
		ps221.OverlayValues[4] = d4
		ps221.OverlayValues[5] = d5
		ps221.OverlayValues[6] = d6
		ps221.OverlayValues[7] = d7
		ps221.OverlayValues[8] = d8
		ps221.OverlayValues[9] = d9
		ps221.OverlayValues[10] = d10
		ps221.OverlayValues[11] = d11
		ps221.OverlayValues[12] = d12
		ps221.OverlayValues[37] = d37
		ps221.OverlayValues[38] = d38
		ps221.OverlayValues[39] = d39
		ps221.OverlayValues[40] = d40
		ps221.OverlayValues[41] = d41
		ps221.OverlayValues[42] = d42
		ps221.OverlayValues[43] = d43
		ps221.OverlayValues[44] = d44
		ps221.OverlayValues[85] = d85
		ps221.OverlayValues[86] = d86
		ps221.OverlayValues[87] = d87
		ps221.OverlayValues[88] = d88
		ps221.OverlayValues[91] = d91
		ps221.OverlayValues[117] = d117
		ps221.OverlayValues[142] = d142
		ps221.OverlayValues[143] = d143
		ps221.OverlayValues[144] = d144
		ps221.OverlayValues[146] = d146
		ps221.OverlayValues[147] = d147
		ps221.OverlayValues[148] = d148
		ps221.OverlayValues[149] = d149
		ps221.OverlayValues[150] = d150
		ps221.OverlayValues[151] = d151
		ps221.OverlayValues[152] = d152
		ps221.OverlayValues[153] = d153
		ps221.OverlayValues[154] = d154
		ps221.OverlayValues[156] = d156
		ps221.OverlayValues[157] = d157
		ps221.OverlayValues[158] = d158
		ps221.OverlayValues[159] = d159
		ps221.OverlayValues[160] = d160
		ps221.OverlayValues[161] = d161
		ps221.OverlayValues[162] = d162
		ps221.OverlayValues[163] = d163
		ps221.OverlayValues[165] = d165
		ps221.OverlayValues[167] = d167
		ps221.OverlayValues[168] = d168
		ps221.OverlayValues[169] = d169
		ps221.OverlayValues[170] = d170
		ps221.OverlayValues[220] = d220
		ps222 := scm.PhiState{General: true}
		ps222.OverlayValues = make([]scm.JITValueDesc, 221)
		ps222.OverlayValues[4] = d4
		ps222.OverlayValues[5] = d5
		ps222.OverlayValues[6] = d6
		ps222.OverlayValues[7] = d7
		ps222.OverlayValues[8] = d8
		ps222.OverlayValues[9] = d9
		ps222.OverlayValues[10] = d10
		ps222.OverlayValues[11] = d11
		ps222.OverlayValues[12] = d12
		ps222.OverlayValues[37] = d37
		ps222.OverlayValues[38] = d38
		ps222.OverlayValues[39] = d39
		ps222.OverlayValues[40] = d40
		ps222.OverlayValues[41] = d41
		ps222.OverlayValues[42] = d42
		ps222.OverlayValues[43] = d43
		ps222.OverlayValues[44] = d44
		ps222.OverlayValues[85] = d85
		ps222.OverlayValues[86] = d86
		ps222.OverlayValues[87] = d87
		ps222.OverlayValues[88] = d88
		ps222.OverlayValues[91] = d91
		ps222.OverlayValues[117] = d117
		ps222.OverlayValues[142] = d142
		ps222.OverlayValues[143] = d143
		ps222.OverlayValues[144] = d144
		ps222.OverlayValues[146] = d146
		ps222.OverlayValues[147] = d147
		ps222.OverlayValues[148] = d148
		ps222.OverlayValues[149] = d149
		ps222.OverlayValues[150] = d150
		ps222.OverlayValues[151] = d151
		ps222.OverlayValues[152] = d152
		ps222.OverlayValues[153] = d153
		ps222.OverlayValues[154] = d154
		ps222.OverlayValues[156] = d156
		ps222.OverlayValues[157] = d157
		ps222.OverlayValues[158] = d158
		ps222.OverlayValues[159] = d159
		ps222.OverlayValues[160] = d160
		ps222.OverlayValues[161] = d161
		ps222.OverlayValues[162] = d162
		ps222.OverlayValues[163] = d163
		ps222.OverlayValues[165] = d165
		ps222.OverlayValues[167] = d167
		ps222.OverlayValues[168] = d168
		ps222.OverlayValues[169] = d169
		ps222.OverlayValues[170] = d170
		ps222.OverlayValues[220] = d220
		ps222.PhiValues = make([]scm.JITValueDesc, 1)
		d223 = d6
		ps222.PhiValues[0] = d223
		snap224 := d4
		snap225 := d5
		snap226 := d6
		snap227 := d7
		snap228 := d8
		snap229 := d9
		snap230 := d10
		snap231 := d11
		snap232 := d12
		snap233 := d37
		snap234 := d38
		snap235 := d39
		snap236 := d40
		snap237 := d41
		snap238 := d42
		snap239 := d43
		snap240 := d44
		snap241 := d85
		snap242 := d86
		snap243 := d87
		snap244 := d88
		snap245 := d91
		snap246 := d117
		snap247 := d142
		snap248 := d143
		snap249 := d144
		snap250 := d146
		snap251 := d147
		snap252 := d148
		snap253 := d149
		snap254 := d150
		snap255 := d151
		snap256 := d152
		snap257 := d153
		snap258 := d154
		snap259 := d156
		snap260 := d157
		snap261 := d158
		snap262 := d159
		snap263 := d160
		snap264 := d161
		snap265 := d162
		snap266 := d163
		snap267 := d165
		snap268 := d167
		snap269 := d168
		snap270 := d169
		snap271 := d170
		snap272 := d220
		snap273 := d223
		alloc274 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps222)
		}
		ctx.RestoreAllocState(alloc274)
		d4 = snap224
		d5 = snap225
		d6 = snap226
		d7 = snap227
		d8 = snap228
		d9 = snap229
		d10 = snap230
		d11 = snap231
		d12 = snap232
		d37 = snap233
		d38 = snap234
		d39 = snap235
		d40 = snap236
		d41 = snap237
		d42 = snap238
		d43 = snap239
		d44 = snap240
		d85 = snap241
		d86 = snap242
		d87 = snap243
		d88 = snap244
		d91 = snap245
		d117 = snap246
		d142 = snap247
		d143 = snap248
		d144 = snap249
		d146 = snap250
		d147 = snap251
		d148 = snap252
		d149 = snap253
		d150 = snap254
		d151 = snap255
		d152 = snap256
		d153 = snap257
		d154 = snap258
		d156 = snap259
		d157 = snap260
		d158 = snap261
		d159 = snap262
		d160 = snap263
		d161 = snap264
		d162 = snap265
		d163 = snap266
		d165 = snap267
		d167 = snap268
		d168 = snap269
		d169 = snap270
		d170 = snap271
		d220 = snap272
		d223 = snap273
		if !bbs[8].Rendered {
			return bbs[8].RenderPS(ps221)
		}
		return result
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
			ctx.FlushRegisterMoves()
			bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_8 = bbs[8].Address
			ctx.MarkLabel(lbl9)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
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
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
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
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
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
		if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
			d220 = ps.OverlayValues[220]
		}
		if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
			d223 = ps.OverlayValues[223]
		}
		ctx.ReclaimUntrackedRegs()
		d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		var d276 scm.JITValueDesc
		r23 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r23, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r23, thisptr.Reg)
			ctx.EmitAddRegImm32(r23, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d276 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r23, &d276)
		d277 = ctx.EmitLoadScalarSliceElement(&d276, &d275, 8, scm.TagInt)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d277)
		ctx.EnsureDescsTogether(&d5, &d277)
		var d278 scm.JITValueDesc
		if d5.Loc == scm.LocImm && d277.Loc == scm.LocImm {
			d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d5.Imm.Int()) < uint64(d277.Imm.Int()))}
		} else if d277.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d5.Reg)
			if d277.Imm.Int() >= -2147483648 && d277.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d5.Reg, int32(d277.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d277.Imm.Int()))
				ctx.EmitCmpInt64(d5.Reg, scm.RegR11)
			}
			d278 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r24, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r24, &d278)
		} else if d5.Loc == scm.LocImm {
			r25 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d277.Reg)
			d278 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r25, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r25, &d278)
		} else {
			r26 := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitCmpInt64(d5.Reg, d277.Reg)
			d278 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r26, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r26, &d278)
		}
		ctx.FreeDesc(&d277)
		d279 = d278
		ctx.EnsureDesc(&d279)
		if d279.Loc != scm.LocImm && d279.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d279.Loc == scm.LocImm {
			if d279.Imm.Bool() {
				if ps.General {
				}
				ps280 := scm.PhiState{General: ps.General}
				ps280.OverlayValues = make([]scm.JITValueDesc, 280)
				ps280.OverlayValues[4] = d4
				ps280.OverlayValues[5] = d5
				ps280.OverlayValues[6] = d6
				ps280.OverlayValues[7] = d7
				ps280.OverlayValues[8] = d8
				ps280.OverlayValues[9] = d9
				ps280.OverlayValues[10] = d10
				ps280.OverlayValues[11] = d11
				ps280.OverlayValues[12] = d12
				ps280.OverlayValues[37] = d37
				ps280.OverlayValues[38] = d38
				ps280.OverlayValues[39] = d39
				ps280.OverlayValues[40] = d40
				ps280.OverlayValues[41] = d41
				ps280.OverlayValues[42] = d42
				ps280.OverlayValues[43] = d43
				ps280.OverlayValues[44] = d44
				ps280.OverlayValues[85] = d85
				ps280.OverlayValues[86] = d86
				ps280.OverlayValues[87] = d87
				ps280.OverlayValues[88] = d88
				ps280.OverlayValues[91] = d91
				ps280.OverlayValues[117] = d117
				ps280.OverlayValues[142] = d142
				ps280.OverlayValues[143] = d143
				ps280.OverlayValues[144] = d144
				ps280.OverlayValues[146] = d146
				ps280.OverlayValues[147] = d147
				ps280.OverlayValues[148] = d148
				ps280.OverlayValues[149] = d149
				ps280.OverlayValues[150] = d150
				ps280.OverlayValues[151] = d151
				ps280.OverlayValues[152] = d152
				ps280.OverlayValues[153] = d153
				ps280.OverlayValues[154] = d154
				ps280.OverlayValues[156] = d156
				ps280.OverlayValues[157] = d157
				ps280.OverlayValues[158] = d158
				ps280.OverlayValues[159] = d159
				ps280.OverlayValues[160] = d160
				ps280.OverlayValues[161] = d161
				ps280.OverlayValues[162] = d162
				ps280.OverlayValues[163] = d163
				ps280.OverlayValues[165] = d165
				ps280.OverlayValues[167] = d167
				ps280.OverlayValues[168] = d168
				ps280.OverlayValues[169] = d169
				ps280.OverlayValues[170] = d170
				ps280.OverlayValues[220] = d220
				ps280.OverlayValues[223] = d223
				ps280.OverlayValues[275] = d275
				ps280.OverlayValues[276] = d276
				ps280.OverlayValues[277] = d277
				ps280.OverlayValues[278] = d278
				ps280.OverlayValues[279] = d279
				return bbs[10].RenderPS(ps280)
			}
			if ps.General {
			}
			ps281 := scm.PhiState{General: ps.General}
			ps281.OverlayValues = make([]scm.JITValueDesc, 280)
			ps281.OverlayValues[4] = d4
			ps281.OverlayValues[5] = d5
			ps281.OverlayValues[6] = d6
			ps281.OverlayValues[7] = d7
			ps281.OverlayValues[8] = d8
			ps281.OverlayValues[9] = d9
			ps281.OverlayValues[10] = d10
			ps281.OverlayValues[11] = d11
			ps281.OverlayValues[12] = d12
			ps281.OverlayValues[37] = d37
			ps281.OverlayValues[38] = d38
			ps281.OverlayValues[39] = d39
			ps281.OverlayValues[40] = d40
			ps281.OverlayValues[41] = d41
			ps281.OverlayValues[42] = d42
			ps281.OverlayValues[43] = d43
			ps281.OverlayValues[44] = d44
			ps281.OverlayValues[85] = d85
			ps281.OverlayValues[86] = d86
			ps281.OverlayValues[87] = d87
			ps281.OverlayValues[88] = d88
			ps281.OverlayValues[91] = d91
			ps281.OverlayValues[117] = d117
			ps281.OverlayValues[142] = d142
			ps281.OverlayValues[143] = d143
			ps281.OverlayValues[144] = d144
			ps281.OverlayValues[146] = d146
			ps281.OverlayValues[147] = d147
			ps281.OverlayValues[148] = d148
			ps281.OverlayValues[149] = d149
			ps281.OverlayValues[150] = d150
			ps281.OverlayValues[151] = d151
			ps281.OverlayValues[152] = d152
			ps281.OverlayValues[153] = d153
			ps281.OverlayValues[154] = d154
			ps281.OverlayValues[156] = d156
			ps281.OverlayValues[157] = d157
			ps281.OverlayValues[158] = d158
			ps281.OverlayValues[159] = d159
			ps281.OverlayValues[160] = d160
			ps281.OverlayValues[161] = d161
			ps281.OverlayValues[162] = d162
			ps281.OverlayValues[163] = d163
			ps281.OverlayValues[165] = d165
			ps281.OverlayValues[167] = d167
			ps281.OverlayValues[168] = d168
			ps281.OverlayValues[169] = d169
			ps281.OverlayValues[170] = d170
			ps281.OverlayValues[220] = d220
			ps281.OverlayValues[223] = d223
			ps281.OverlayValues[275] = d275
			ps281.OverlayValues[276] = d276
			ps281.OverlayValues[277] = d277
			ps281.OverlayValues[278] = d278
			ps281.OverlayValues[279] = d279
			return bbs[11].RenderPS(ps281)
		}
		if !ps.General {
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		ctx.EmitJump(d279.Condition, lbl11)
		if bbs[11].Rendered {
			ctx.EmitJmp(lbl12)
		}
		ctx.FreeDesc(&d278)
		snap282 := d4
		snap283 := d5
		snap284 := d6
		snap285 := d7
		snap286 := d8
		snap287 := d9
		snap288 := d10
		snap289 := d11
		snap290 := d12
		snap291 := d37
		snap292 := d38
		snap293 := d39
		snap294 := d40
		snap295 := d41
		snap296 := d42
		snap297 := d43
		snap298 := d44
		snap299 := d85
		snap300 := d86
		snap301 := d87
		snap302 := d88
		snap303 := d91
		snap304 := d117
		snap305 := d142
		snap306 := d143
		snap307 := d144
		snap308 := d146
		snap309 := d147
		snap310 := d148
		snap311 := d149
		snap312 := d150
		snap313 := d151
		snap314 := d152
		snap315 := d153
		snap316 := d154
		snap317 := d156
		snap318 := d157
		snap319 := d158
		snap320 := d159
		snap321 := d160
		snap322 := d161
		snap323 := d162
		snap324 := d163
		snap325 := d165
		snap326 := d167
		snap327 := d168
		snap328 := d169
		snap329 := d170
		snap330 := d220
		snap331 := d223
		snap332 := d275
		snap333 := d276
		snap334 := d277
		snap335 := d278
		snap336 := d279
		alloc337 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc337)
		d4 = snap282
		d5 = snap283
		d6 = snap284
		d7 = snap285
		d8 = snap286
		d9 = snap287
		d10 = snap288
		d11 = snap289
		d12 = snap290
		d37 = snap291
		d38 = snap292
		d39 = snap293
		d40 = snap294
		d41 = snap295
		d42 = snap296
		d43 = snap297
		d44 = snap298
		d85 = snap299
		d86 = snap300
		d87 = snap301
		d88 = snap302
		d91 = snap303
		d117 = snap304
		d142 = snap305
		d143 = snap306
		d144 = snap307
		d146 = snap308
		d147 = snap309
		d148 = snap310
		d149 = snap311
		d150 = snap312
		d151 = snap313
		d152 = snap314
		d153 = snap315
		d154 = snap316
		d156 = snap317
		d157 = snap318
		d158 = snap319
		d159 = snap320
		d160 = snap321
		d161 = snap322
		d162 = snap323
		d163 = snap324
		d165 = snap325
		d167 = snap326
		d168 = snap327
		d169 = snap328
		d170 = snap329
		d220 = snap330
		d223 = snap331
		d275 = snap332
		d276 = snap333
		d277 = snap334
		d278 = snap335
		d279 = snap336
		ctx.RestoreAllocState(alloc337)
		d4 = snap282
		d5 = snap283
		d6 = snap284
		d7 = snap285
		d8 = snap286
		d9 = snap287
		d10 = snap288
		d11 = snap289
		d12 = snap290
		d37 = snap291
		d38 = snap292
		d39 = snap293
		d40 = snap294
		d41 = snap295
		d42 = snap296
		d43 = snap297
		d44 = snap298
		d85 = snap299
		d86 = snap300
		d87 = snap301
		d88 = snap302
		d91 = snap303
		d117 = snap304
		d142 = snap305
		d143 = snap306
		d144 = snap307
		d146 = snap308
		d147 = snap309
		d148 = snap310
		d149 = snap311
		d150 = snap312
		d151 = snap313
		d152 = snap314
		d153 = snap315
		d154 = snap316
		d156 = snap317
		d157 = snap318
		d158 = snap319
		d159 = snap320
		d160 = snap321
		d161 = snap322
		d162 = snap323
		d163 = snap324
		d165 = snap325
		d167 = snap326
		d168 = snap327
		d169 = snap328
		d170 = snap329
		d220 = snap330
		d223 = snap331
		d275 = snap332
		d276 = snap333
		d277 = snap334
		d278 = snap335
		d279 = snap336
		ps338 := scm.PhiState{General: true}
		ps338.OverlayValues = make([]scm.JITValueDesc, 280)
		ps338.OverlayValues[4] = d4
		ps338.OverlayValues[5] = d5
		ps338.OverlayValues[6] = d6
		ps338.OverlayValues[7] = d7
		ps338.OverlayValues[8] = d8
		ps338.OverlayValues[9] = d9
		ps338.OverlayValues[10] = d10
		ps338.OverlayValues[11] = d11
		ps338.OverlayValues[12] = d12
		ps338.OverlayValues[37] = d37
		ps338.OverlayValues[38] = d38
		ps338.OverlayValues[39] = d39
		ps338.OverlayValues[40] = d40
		ps338.OverlayValues[41] = d41
		ps338.OverlayValues[42] = d42
		ps338.OverlayValues[43] = d43
		ps338.OverlayValues[44] = d44
		ps338.OverlayValues[85] = d85
		ps338.OverlayValues[86] = d86
		ps338.OverlayValues[87] = d87
		ps338.OverlayValues[88] = d88
		ps338.OverlayValues[91] = d91
		ps338.OverlayValues[117] = d117
		ps338.OverlayValues[142] = d142
		ps338.OverlayValues[143] = d143
		ps338.OverlayValues[144] = d144
		ps338.OverlayValues[146] = d146
		ps338.OverlayValues[147] = d147
		ps338.OverlayValues[148] = d148
		ps338.OverlayValues[149] = d149
		ps338.OverlayValues[150] = d150
		ps338.OverlayValues[151] = d151
		ps338.OverlayValues[152] = d152
		ps338.OverlayValues[153] = d153
		ps338.OverlayValues[154] = d154
		ps338.OverlayValues[156] = d156
		ps338.OverlayValues[157] = d157
		ps338.OverlayValues[158] = d158
		ps338.OverlayValues[159] = d159
		ps338.OverlayValues[160] = d160
		ps338.OverlayValues[161] = d161
		ps338.OverlayValues[162] = d162
		ps338.OverlayValues[163] = d163
		ps338.OverlayValues[165] = d165
		ps338.OverlayValues[167] = d167
		ps338.OverlayValues[168] = d168
		ps338.OverlayValues[169] = d169
		ps338.OverlayValues[170] = d170
		ps338.OverlayValues[220] = d220
		ps338.OverlayValues[223] = d223
		ps338.OverlayValues[275] = d275
		ps338.OverlayValues[276] = d276
		ps338.OverlayValues[277] = d277
		ps338.OverlayValues[278] = d278
		ps338.OverlayValues[279] = d279
		ps339 := scm.PhiState{General: true}
		ps339.OverlayValues = make([]scm.JITValueDesc, 280)
		ps339.OverlayValues[4] = d4
		ps339.OverlayValues[5] = d5
		ps339.OverlayValues[6] = d6
		ps339.OverlayValues[7] = d7
		ps339.OverlayValues[8] = d8
		ps339.OverlayValues[9] = d9
		ps339.OverlayValues[10] = d10
		ps339.OverlayValues[11] = d11
		ps339.OverlayValues[12] = d12
		ps339.OverlayValues[37] = d37
		ps339.OverlayValues[38] = d38
		ps339.OverlayValues[39] = d39
		ps339.OverlayValues[40] = d40
		ps339.OverlayValues[41] = d41
		ps339.OverlayValues[42] = d42
		ps339.OverlayValues[43] = d43
		ps339.OverlayValues[44] = d44
		ps339.OverlayValues[85] = d85
		ps339.OverlayValues[86] = d86
		ps339.OverlayValues[87] = d87
		ps339.OverlayValues[88] = d88
		ps339.OverlayValues[91] = d91
		ps339.OverlayValues[117] = d117
		ps339.OverlayValues[142] = d142
		ps339.OverlayValues[143] = d143
		ps339.OverlayValues[144] = d144
		ps339.OverlayValues[146] = d146
		ps339.OverlayValues[147] = d147
		ps339.OverlayValues[148] = d148
		ps339.OverlayValues[149] = d149
		ps339.OverlayValues[150] = d150
		ps339.OverlayValues[151] = d151
		ps339.OverlayValues[152] = d152
		ps339.OverlayValues[153] = d153
		ps339.OverlayValues[154] = d154
		ps339.OverlayValues[156] = d156
		ps339.OverlayValues[157] = d157
		ps339.OverlayValues[158] = d158
		ps339.OverlayValues[159] = d159
		ps339.OverlayValues[160] = d160
		ps339.OverlayValues[161] = d161
		ps339.OverlayValues[162] = d162
		ps339.OverlayValues[163] = d163
		ps339.OverlayValues[165] = d165
		ps339.OverlayValues[167] = d167
		ps339.OverlayValues[168] = d168
		ps339.OverlayValues[169] = d169
		ps339.OverlayValues[170] = d170
		ps339.OverlayValues[220] = d220
		ps339.OverlayValues[223] = d223
		ps339.OverlayValues[275] = d275
		ps339.OverlayValues[276] = d276
		ps339.OverlayValues[277] = d277
		ps339.OverlayValues[278] = d278
		ps339.OverlayValues[279] = d279
		snap340 := d4
		snap341 := d5
		snap342 := d6
		snap343 := d7
		snap344 := d8
		snap345 := d9
		snap346 := d10
		snap347 := d11
		snap348 := d12
		snap349 := d37
		snap350 := d38
		snap351 := d39
		snap352 := d40
		snap353 := d41
		snap354 := d42
		snap355 := d43
		snap356 := d44
		snap357 := d85
		snap358 := d86
		snap359 := d87
		snap360 := d88
		snap361 := d91
		snap362 := d117
		snap363 := d142
		snap364 := d143
		snap365 := d144
		snap366 := d146
		snap367 := d147
		snap368 := d148
		snap369 := d149
		snap370 := d150
		snap371 := d151
		snap372 := d152
		snap373 := d153
		snap374 := d154
		snap375 := d156
		snap376 := d157
		snap377 := d158
		snap378 := d159
		snap379 := d160
		snap380 := d161
		snap381 := d162
		snap382 := d163
		snap383 := d165
		snap384 := d167
		snap385 := d168
		snap386 := d169
		snap387 := d170
		snap388 := d220
		snap389 := d223
		snap390 := d275
		snap391 := d276
		snap392 := d277
		snap393 := d278
		snap394 := d279
		alloc395 := ctx.SnapshotAllocState()
		if !bbs[11].Rendered {
			bbs[11].RenderPS(ps339)
		}
		ctx.RestoreAllocState(alloc395)
		d4 = snap340
		d5 = snap341
		d6 = snap342
		d7 = snap343
		d8 = snap344
		d9 = snap345
		d10 = snap346
		d11 = snap347
		d12 = snap348
		d37 = snap349
		d38 = snap350
		d39 = snap351
		d40 = snap352
		d41 = snap353
		d42 = snap354
		d43 = snap355
		d44 = snap356
		d85 = snap357
		d86 = snap358
		d87 = snap359
		d88 = snap360
		d91 = snap361
		d117 = snap362
		d142 = snap363
		d143 = snap364
		d144 = snap365
		d146 = snap366
		d147 = snap367
		d148 = snap368
		d149 = snap369
		d150 = snap370
		d151 = snap371
		d152 = snap372
		d153 = snap373
		d154 = snap374
		d156 = snap375
		d157 = snap376
		d158 = snap377
		d159 = snap378
		d160 = snap379
		d161 = snap380
		d162 = snap381
		d163 = snap382
		d165 = snap383
		d167 = snap384
		d168 = snap385
		d169 = snap386
		d170 = snap387
		d220 = snap388
		d223 = snap389
		d275 = snap390
		d276 = snap391
		d277 = snap392
		d278 = snap393
		d279 = snap394
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps338)
		}
		return result
		return result
	}
	bbs[9].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d396 := ps.PhiValues[0]
				ctx.EnsureDesc(&d396)
				ctx.EmitStoreScmerToStack(d396, int32(bbs[9].PhiBase)+int32(0))
			}
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
			ctx.FlushRegisterMoves()
			bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_9 = bbs[9].Address
			ctx.MarkLabel(lbl10)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
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
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
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
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
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
		if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
			d220 = ps.OverlayValues[220]
		}
		if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
			d223 = ps.OverlayValues[223]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
			d276 = ps.OverlayValues[276]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d8 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		d397 = result
		ctx.EnsureDesc(&d8)
		if d8.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d8, &d397)
		} else {
			switch d8.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d397, d8)
			case scm.TagInt:
				ctx.EmitMakeInt(d397, d8)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d397, d8)
			case scm.TagNil:
				ctx.EmitMakeNil(d397)
			default:
				ctx.EmitMovPairToResult(&d8, &d397)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[10].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[10].VisitCount >= 0 {
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
		}
		bbs[10].VisitCount++
		if ps.General {
			if bbs[10].Rendered {
				ctx.EmitJmp(lbl11)
				return result
			}
			bbs[10].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_10 = bbs[10].Address
			ctx.MarkLabel(lbl11)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
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
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
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
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
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
		if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
			d220 = ps.OverlayValues[220]
		}
		if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
			d223 = ps.OverlayValues[223]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
			d276 = ps.OverlayValues[276]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		ctx.ReclaimUntrackedRegs()
		d398 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		var d399 scm.JITValueDesc
		r27 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r27, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r27, thisptr.Reg)
			ctx.EmitAddRegImm32(r27, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d399 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r27, &d399)
		d401 = ctx.EmitSliceElementAddress(&d399, &d398, 16)
		ctx.EmitLoadScmerToStack(&d401, int32(bbs[9].PhiBase)+int32(0))
		ctx.FreeDesc(&d401)
		d400 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(bbs[9].PhiBase) + int32(0)}
		ctx.StabilizeDescForControlFlow(&d400)
		if ps.General {
			ctx.SyncDesc(&d400)
			if d400.Loc == scm.LocReg || d400.Loc == scm.LocFPReg {
				ctx.ProtectReg(d400.Reg)
			} else if d400.Loc == scm.LocRegPair {
				ctx.ProtectReg(d400.Reg)
				ctx.ProtectReg(d400.Reg2)
			}
			d402 = d400
			if d402.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d402)
			if d402.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d402, int32(bbs[9].PhiBase)+int32(0), 2)
			} else if d402.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d402)
				ctx.EmitStoreScmerToStack(d402, int32(bbs[9].PhiBase)+int32(0))
			} else if d402.Loc == scm.LocRegPair || d402.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d402, int32(bbs[9].PhiBase)+int32(0))
			} else {
				ctx.EnsureDesc(&d402)
				ctx.EmitStoreToStack(d402, int32(bbs[9].PhiBase)+int32(0))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
			}
			if d400.Loc == scm.LocReg || d400.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d400.Reg)
			} else if d400.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d400.Reg)
				ctx.UnprotectReg(d400.Reg2)
			}
		}
		ps403 := scm.PhiState{General: ps.General}
		ps403.OverlayValues = make([]scm.JITValueDesc, 403)
		ps403.OverlayValues[4] = d4
		ps403.OverlayValues[5] = d5
		ps403.OverlayValues[6] = d6
		ps403.OverlayValues[7] = d7
		ps403.OverlayValues[8] = d8
		ps403.OverlayValues[9] = d9
		ps403.OverlayValues[10] = d10
		ps403.OverlayValues[11] = d11
		ps403.OverlayValues[12] = d12
		ps403.OverlayValues[37] = d37
		ps403.OverlayValues[38] = d38
		ps403.OverlayValues[39] = d39
		ps403.OverlayValues[40] = d40
		ps403.OverlayValues[41] = d41
		ps403.OverlayValues[42] = d42
		ps403.OverlayValues[43] = d43
		ps403.OverlayValues[44] = d44
		ps403.OverlayValues[85] = d85
		ps403.OverlayValues[86] = d86
		ps403.OverlayValues[87] = d87
		ps403.OverlayValues[88] = d88
		ps403.OverlayValues[91] = d91
		ps403.OverlayValues[117] = d117
		ps403.OverlayValues[142] = d142
		ps403.OverlayValues[143] = d143
		ps403.OverlayValues[144] = d144
		ps403.OverlayValues[146] = d146
		ps403.OverlayValues[147] = d147
		ps403.OverlayValues[148] = d148
		ps403.OverlayValues[149] = d149
		ps403.OverlayValues[150] = d150
		ps403.OverlayValues[151] = d151
		ps403.OverlayValues[152] = d152
		ps403.OverlayValues[153] = d153
		ps403.OverlayValues[154] = d154
		ps403.OverlayValues[156] = d156
		ps403.OverlayValues[157] = d157
		ps403.OverlayValues[158] = d158
		ps403.OverlayValues[159] = d159
		ps403.OverlayValues[160] = d160
		ps403.OverlayValues[161] = d161
		ps403.OverlayValues[162] = d162
		ps403.OverlayValues[163] = d163
		ps403.OverlayValues[165] = d165
		ps403.OverlayValues[167] = d167
		ps403.OverlayValues[168] = d168
		ps403.OverlayValues[169] = d169
		ps403.OverlayValues[170] = d170
		ps403.OverlayValues[220] = d220
		ps403.OverlayValues[223] = d223
		ps403.OverlayValues[275] = d275
		ps403.OverlayValues[276] = d276
		ps403.OverlayValues[277] = d277
		ps403.OverlayValues[278] = d278
		ps403.OverlayValues[279] = d279
		ps403.OverlayValues[396] = d396
		ps403.OverlayValues[397] = d397
		ps403.OverlayValues[398] = d398
		ps403.OverlayValues[399] = d399
		ps403.OverlayValues[400] = d400
		ps403.OverlayValues[401] = d401
		ps403.OverlayValues[402] = d402
		ps403.PhiValues = make([]scm.JITValueDesc, 1)
		d404 = d400
		ps403.PhiValues[0] = d404
		if ps403.General && bbs[9].Rendered {
			ctx.EmitJmp(lbl10)
			return result
		}
		return bbs[9].RenderPS(ps403)
		return result
	}
	bbs[11].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[11].VisitCount >= 0 {
				ps.General = true
				return bbs[11].RenderPS(ps)
			}
		}
		bbs[11].VisitCount++
		if ps.General {
			if bbs[11].Rendered {
				ctx.EmitJmp(lbl12)
				return result
			}
			bbs[11].Rendered = true
			ctx.FlushRegisterMoves()
			bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_11 = bbs[11].Address
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
		}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		d6 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(32)}
		if phiHomeOK3 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(phiBase0) + int32(64)}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
			d85 = ps.OverlayValues[85]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
			d87 = ps.OverlayValues[87]
		}
		if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
			d88 = ps.OverlayValues[88]
		}
		if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != scm.LocNone {
			d91 = ps.OverlayValues[91]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
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
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
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
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
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
		if len(ps.OverlayValues) > 220 && ps.OverlayValues[220].Loc != scm.LocNone {
			d220 = ps.OverlayValues[220]
		}
		if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != scm.LocNone {
			d223 = ps.OverlayValues[223]
		}
		if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != scm.LocNone {
			d275 = ps.OverlayValues[275]
		}
		if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
			d276 = ps.OverlayValues[276]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != scm.LocNone {
			d278 = ps.OverlayValues[278]
		}
		if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != scm.LocNone {
			d279 = ps.OverlayValues[279]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 401 && ps.OverlayValues[401].Loc != scm.LocNone {
			d401 = ps.OverlayValues[401]
		}
		if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != scm.LocNone {
			d402 = ps.OverlayValues[402]
		}
		if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != scm.LocNone {
			d404 = ps.OverlayValues[404]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&d5)
		d405 = d5
		_ = d405
		ctx.StabilizeDescForControlFlow(&d5)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl15 := ctx.ReserveLabel()
		_ = lbl15
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl15)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d405)
		var d406 scm.JITValueDesc
		if d405.Loc == scm.LocImm {
			d406 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d405.Imm.Int() & 255)}
		} else {
			r28 := ctx.AllocRegExcept(d405.Reg)
			ctx.EmitMovRegReg(r28, d405.Reg)
			ctx.EmitAndRegImm32(r28, int32(255))
			d406 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d406)
		}
		if d406.Loc == scm.LocReg && d405.Loc == scm.LocReg && d406.Reg == d405.Reg {
			ctx.TransferReg(d405.Reg)
			d405.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d406.Loc == scm.LocRegPair || d406.Loc == scm.LocStackPair || d406.Loc == scm.LocRegTriple || d406.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d406)
		d407 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).decodeSymbol), []scm.JITValueDesc{thisptr, d406}, 1)
		d407.NoHeapPointer = true
		ctx.BindReg(d407.Reg, &d407)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d407)
		var d408 scm.JITValueDesc
		r29 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r29, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r29, thisptr.Reg)
			ctx.EmitAddRegImm32(r29, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d408 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r29, &d408)
		ctx.ReclaimUntrackedRegs()
		d409 = ctx.EmitLoadScalarSliceElement(&d408, &d407, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d407)
		var d410 scm.JITValueDesc
		r30 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r30, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r30, thisptr.Reg)
			ctx.EmitAddRegImm32(r30, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d410 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r30, &d410)
		ctx.ReclaimUntrackedRegs()
		d412 = ctx.EmitSliceElementAddress(&d410, &d407, 16)
		ctx.EnsureDesc(&d412)
		r31 := ctx.AllocRegExcept(d412.Reg)
		ctx.EmitMovRegMem(r31, d412.Reg, 8)
		ctx.EmitMovRegMem(d412.Reg, d412.Reg, 0)
		d411 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d412.Reg, Reg2: r31}
		ctx.BindReg(d412.Reg, &d411)
		ctx.BindReg(r31, &d411)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d405)
		var d413 scm.JITValueDesc
		if d405.Loc == scm.LocImm {
			d413 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d405.Imm.Int()) >> 8))}
		} else {
			r32 := ctx.AllocRegExcept(d405.Reg)
			ctx.EmitMovRegReg(r32, d405.Reg)
			ctx.EmitShrRegImm8(r32, 8)
			d413 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d413)
		}
		if d413.Loc == scm.LocReg && d405.Loc == scm.LocReg && d413.Reg == d405.Reg {
			ctx.TransferReg(d405.Reg)
			d405.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d413)
		ctx.EnsureDesc(&d409)
		ctx.EnsureDescsTogether(&d413, &d409)
		var d414 scm.JITValueDesc
		if d413.Loc == scm.LocImm && d409.Loc == scm.LocImm {
			d414 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d413.Imm.Int() * d409.Imm.Int())}
		} else if d413.Loc == scm.LocImm {
			ctx.EnsureDesc(&d409)
			scratch := ctx.AllocRegExcept(d409.Reg)
			ctx.EmitMovRegReg(scratch, d409.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d413.Imm.Int())
			d414 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d414)
		} else if d409.Loc == scm.LocImm {
			ctx.EnsureDesc(&d413)
			scratch := ctx.AllocRegExcept(d413.Reg)
			ctx.EmitMovRegReg(scratch, d413.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d409.Imm.Int())
			d414 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d414)
		} else {
			ctx.EnsureDesc(&d413)
			ctx.SyncDesc(&d409)
			r33 := ctx.AllocRegExcept(d413.Reg, d409.Reg)
			ctx.EmitMovRegReg(r33, d413.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r33, &d409)
			d414 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d414)
		}
		if d414.Loc == scm.LocReg && d413.Loc == scm.LocReg && d414.Reg == d413.Reg {
			ctx.TransferReg(d413.Reg)
			d413.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d413)
		ctx.FreeDesc(&d409)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d414)
		ctx.EnsureDesc(&d406)
		ctx.EnsureDescsTogether(&d414, &d406)
		var d415 scm.JITValueDesc
		if d414.Loc == scm.LocImm && d406.Loc == scm.LocImm {
			d415 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d414.Imm.Int() + d406.Imm.Int())}
		} else if d406.Loc == scm.LocImm && d406.Imm.Int() == 0 {
			ctx.EnsureDesc(&d414)
			r34 := ctx.AllocRegExcept(d414.Reg)
			ctx.EmitMovRegReg(r34, d414.Reg)
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d415)
		} else if d414.Loc == scm.LocImm && d414.Imm.Int() == 0 {
			ctx.EnsureDesc(&d406)
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d406.Reg}
			ctx.BindReg(d406.Reg, &d415)
		} else if d414.Loc == scm.LocImm {
			ctx.EnsureDesc(&d406)
			scratch := ctx.AllocRegExcept(d406.Reg)
			ctx.EmitMovRegReg(scratch, d406.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d414.Imm.Int())
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d415)
		} else if d406.Loc == scm.LocImm {
			ctx.EnsureDesc(&d414)
			scratch := ctx.AllocRegExcept(d414.Reg)
			ctx.EmitMovRegReg(scratch, d414.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d406.Imm.Int())
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d415)
		} else {
			ctx.EnsureDesc(&d414)
			ctx.SyncDesc(&d406)
			r35 := ctx.AllocRegExcept(d414.Reg, d406.Reg)
			ctx.EmitMovRegReg(r35, d414.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r35, &d406)
			d415 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d415)
		}
		if d415.Loc == scm.LocReg && d414.Loc == scm.LocReg && d415.Reg == d414.Reg {
			ctx.TransferReg(d414.Reg)
			d414.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d414)
		ctx.FreeDesc(&d406)
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d407.Loc == scm.LocRegPair || d407.Loc == scm.LocStackPair || d407.Loc == scm.LocRegTriple || d407.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d407)
		d416 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).symbolLo), []scm.JITValueDesc{thisptr, d407}, 1)
		d416.NoHeapPointer = true
		ctx.BindReg(d416.Reg, &d416)
		ctx.FreeDesc(&d407)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d415)
		ctx.EnsureDesc(&d416)
		ctx.EnsureDescsTogether(&d415, &d416)
		var d417 scm.JITValueDesc
		if d415.Loc == scm.LocImm && d416.Loc == scm.LocImm {
			d417 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d415.Imm.Int() - d416.Imm.Int())}
		} else if d416.Loc == scm.LocImm && d416.Imm.Int() == 0 {
			ctx.EnsureDesc(&d415)
			r36 := ctx.AllocRegExcept(d415.Reg)
			ctx.EmitMovRegReg(r36, d415.Reg)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d417)
		} else if d415.Loc == scm.LocImm {
			ctx.EnsureDesc(&d416)
			scratch := ctx.AllocRegExcept(d416.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d415.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d416)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d417)
		} else if d416.Loc == scm.LocImm {
			ctx.EnsureDesc(&d415)
			scratch := ctx.AllocRegExcept(d415.Reg)
			ctx.EmitMovRegReg(scratch, d415.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d416.Imm.Int())
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d417)
		} else {
			ctx.EnsureDesc(&d415)
			ctx.SyncDesc(&d416)
			r37 := ctx.AllocRegExcept(d415.Reg, d416.Reg)
			ctx.EmitMovRegReg(r37, d415.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r37, &d416)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d417)
		}
		if d417.Loc == scm.LocReg && d415.Loc == scm.LocReg && d417.Reg == d415.Reg {
			ctx.TransferReg(d415.Reg)
			d415.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d415)
		ctx.FreeDesc(&d416)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d411)
		ctx.EnsureDesc(&d417)
		ctx.StabilizeDescForControlFlow(&d411)
		ctx.StabilizeDescForControlFlow(&d417)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d7)
		var d418 scm.JITValueDesc
		if d7.Loc == scm.LocImm {
			d418 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + 1)}
		} else {
			var scratch scm.Reg
			if phiHomeOK3 {
				scratch = r1
			} else {
				scratch = ctx.AllocRegExcept(d7.Reg)
			}
			ctx.EmitMovRegReg(scratch, d7.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d418 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d418)
		}
		if d418.Loc == scm.LocReg && d7.Loc == scm.LocReg && d418.Reg == d7.Reg {
			ctx.TransferReg(d7.Reg)
			d7.Loc = scm.LocNone
		}
		if ps.General {
			ctx.SyncDesc(&d411)
			if d411.Loc == scm.LocReg || d411.Loc == scm.LocFPReg {
				ctx.ProtectReg(d411.Reg)
			} else if d411.Loc == scm.LocRegPair {
				ctx.ProtectReg(d411.Reg)
				ctx.ProtectReg(d411.Reg2)
			}
			ctx.SyncDesc(&d417)
			if d417.Loc == scm.LocReg || d417.Loc == scm.LocFPReg {
				ctx.ProtectReg(d417.Reg)
			} else if d417.Loc == scm.LocRegPair {
				ctx.ProtectReg(d417.Reg)
				ctx.ProtectReg(d417.Reg2)
			}
			ctx.SyncDesc(&d418)
			if d418.Loc == scm.LocReg || d418.Loc == scm.LocFPReg {
				ctx.ProtectReg(d418.Reg)
			} else if d418.Loc == scm.LocRegPair {
				ctx.ProtectReg(d418.Reg)
				ctx.ProtectReg(d418.Reg2)
			}
			d419 = d417
			if d419.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d419)
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d419)
			} else {
				ctx.EmitStoreToStack(d419, int32(bbs[7].PhiBase)+int32(0))
			}
			d420 = d411
			if d420.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d420)
			if d420.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d420, int32(bbs[7].PhiBase)+int32(16), 2)
			} else if d420.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d420)
				ctx.EmitStoreScmerToStack(d420, int32(bbs[7].PhiBase)+int32(16))
			} else if d420.Loc == scm.LocRegPair || d420.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d420, int32(bbs[7].PhiBase)+int32(16))
			} else {
				ctx.EnsureDesc(&d420)
				ctx.EmitStoreToStack(d420, int32(bbs[7].PhiBase)+int32(16))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			}
			d421 = d418
			if d421.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d421)
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d421)
			} else {
				ctx.EmitStoreToStack(d421, int32(bbs[7].PhiBase)+int32(32))
			}
			if d411.Loc == scm.LocReg || d411.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d411.Reg)
			} else if d411.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d411.Reg)
				ctx.UnprotectReg(d411.Reg2)
			}
			if d417.Loc == scm.LocReg || d417.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d417.Reg)
			} else if d417.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d417.Reg)
				ctx.UnprotectReg(d417.Reg2)
			}
			if d418.Loc == scm.LocReg || d418.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d418.Reg)
			} else if d418.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d418.Reg)
				ctx.UnprotectReg(d418.Reg2)
			}
		}
		ps422 := scm.PhiState{General: ps.General}
		ps422.OverlayValues = make([]scm.JITValueDesc, 422)
		ps422.OverlayValues[4] = d4
		ps422.OverlayValues[5] = d5
		ps422.OverlayValues[6] = d6
		ps422.OverlayValues[7] = d7
		ps422.OverlayValues[8] = d8
		ps422.OverlayValues[9] = d9
		ps422.OverlayValues[10] = d10
		ps422.OverlayValues[11] = d11
		ps422.OverlayValues[12] = d12
		ps422.OverlayValues[37] = d37
		ps422.OverlayValues[38] = d38
		ps422.OverlayValues[39] = d39
		ps422.OverlayValues[40] = d40
		ps422.OverlayValues[41] = d41
		ps422.OverlayValues[42] = d42
		ps422.OverlayValues[43] = d43
		ps422.OverlayValues[44] = d44
		ps422.OverlayValues[85] = d85
		ps422.OverlayValues[86] = d86
		ps422.OverlayValues[87] = d87
		ps422.OverlayValues[88] = d88
		ps422.OverlayValues[91] = d91
		ps422.OverlayValues[117] = d117
		ps422.OverlayValues[142] = d142
		ps422.OverlayValues[143] = d143
		ps422.OverlayValues[144] = d144
		ps422.OverlayValues[146] = d146
		ps422.OverlayValues[147] = d147
		ps422.OverlayValues[148] = d148
		ps422.OverlayValues[149] = d149
		ps422.OverlayValues[150] = d150
		ps422.OverlayValues[151] = d151
		ps422.OverlayValues[152] = d152
		ps422.OverlayValues[153] = d153
		ps422.OverlayValues[154] = d154
		ps422.OverlayValues[156] = d156
		ps422.OverlayValues[157] = d157
		ps422.OverlayValues[158] = d158
		ps422.OverlayValues[159] = d159
		ps422.OverlayValues[160] = d160
		ps422.OverlayValues[161] = d161
		ps422.OverlayValues[162] = d162
		ps422.OverlayValues[163] = d163
		ps422.OverlayValues[165] = d165
		ps422.OverlayValues[167] = d167
		ps422.OverlayValues[168] = d168
		ps422.OverlayValues[169] = d169
		ps422.OverlayValues[170] = d170
		ps422.OverlayValues[220] = d220
		ps422.OverlayValues[223] = d223
		ps422.OverlayValues[275] = d275
		ps422.OverlayValues[276] = d276
		ps422.OverlayValues[277] = d277
		ps422.OverlayValues[278] = d278
		ps422.OverlayValues[279] = d279
		ps422.OverlayValues[396] = d396
		ps422.OverlayValues[397] = d397
		ps422.OverlayValues[398] = d398
		ps422.OverlayValues[399] = d399
		ps422.OverlayValues[400] = d400
		ps422.OverlayValues[401] = d401
		ps422.OverlayValues[402] = d402
		ps422.OverlayValues[404] = d404
		ps422.OverlayValues[405] = d405
		ps422.OverlayValues[406] = d406
		ps422.OverlayValues[407] = d407
		ps422.OverlayValues[408] = d408
		ps422.OverlayValues[409] = d409
		ps422.OverlayValues[410] = d410
		ps422.OverlayValues[411] = d411
		ps422.OverlayValues[412] = d412
		ps422.OverlayValues[413] = d413
		ps422.OverlayValues[414] = d414
		ps422.OverlayValues[415] = d415
		ps422.OverlayValues[416] = d416
		ps422.OverlayValues[417] = d417
		ps422.OverlayValues[418] = d418
		ps422.OverlayValues[419] = d419
		ps422.OverlayValues[420] = d420
		ps422.OverlayValues[421] = d421
		ps422.PhiValues = make([]scm.JITValueDesc, 3)
		d423 = d417
		ps422.PhiValues[0] = d423
		d424 = d411
		ps422.PhiValues[1] = d424
		d425 = d418
		ps422.PhiValues[2] = d425
		if ps422.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps422)
		return result
	}
	ps426 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps426)
	ctx.MarkLabel(lbl0)
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
