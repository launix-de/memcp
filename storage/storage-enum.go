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
	var d30 scm.JITValueDesc
	_ = d30
	var d49 scm.JITValueDesc
	_ = d49
	var d50 scm.JITValueDesc
	_ = d50
	var d51 scm.JITValueDesc
	_ = d51
	var d52 scm.JITValueDesc
	_ = d52
	var d97 scm.JITValueDesc
	_ = d97
	var d98 scm.JITValueDesc
	_ = d98
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
	var d107 scm.JITValueDesc
	_ = d107
	var d108 scm.JITValueDesc
	_ = d108
	var d109 scm.JITValueDesc
	_ = d109
	var d145 scm.JITValueDesc
	_ = d145
	var d182 scm.JITValueDesc
	_ = d182
	var d183 scm.JITValueDesc
	_ = d183
	var d184 scm.JITValueDesc
	_ = d184
	var d185 scm.JITValueDesc
	_ = d185
	var d186 scm.JITValueDesc
	_ = d186
	var d228 scm.JITValueDesc
	_ = d228
	var d229 scm.JITValueDesc
	_ = d229
	var d230 scm.JITValueDesc
	_ = d230
	var d231 scm.JITValueDesc
	_ = d231
	var d232 scm.JITValueDesc
	_ = d232
	var d233 scm.JITValueDesc
	_ = d233
	var d234 scm.JITValueDesc
	_ = d234
	var d235 scm.JITValueDesc
	_ = d235
	var d236 scm.JITValueDesc
	_ = d236
	var d237 scm.JITValueDesc
	_ = d237
	var d238 scm.JITValueDesc
	_ = d238
	var d239 scm.JITValueDesc
	_ = d239
	var d240 scm.JITValueDesc
	_ = d240
	var d241 scm.JITValueDesc
	_ = d241
	var d242 scm.JITValueDesc
	_ = d242
	var d243 scm.JITValueDesc
	_ = d243
	var d244 scm.JITValueDesc
	_ = d244
	var d245 scm.JITValueDesc
	_ = d245
	var d246 scm.JITValueDesc
	_ = d246
	var d247 scm.JITValueDesc
	_ = d247
	var d248 scm.JITValueDesc
	_ = d248
	var d249 scm.JITValueDesc
	_ = d249
	var d250 scm.JITValueDesc
	_ = d250
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
	bbs[7].PhiBase = int32(phiBase0) + int32(16)
	bbs[9].PhiBase = int32(phiBase0) + int32(64)
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
	bbs[0].Render = func() scm.JITValueDesc {
		if bbs[0].Rendered {
			ctx.EmitJmp(lbl1)
			return result
		}
		bbs[0].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl1)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
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
		if idxInt.Loc == scm.LocImm && d10.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) >= uint64(d10.Imm.Int()))}
		} else if d10.Loc == scm.LocImm {
			r3 := ctx.AllocRegExcept(idxInt.Reg)
			if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d10.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d10.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, ctx.ScratchReg)
			}
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r3, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r3, &d11)
		} else if idxInt.Loc == scm.LocImm {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d10.Reg)
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
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d12.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d11)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap13 := d4
			snap14 := d5
			snap15 := d6
			snap16 := d7
			snap17 := d8
			snap18 := d9
			snap19 := d10
			snap20 := d11
			snap21 := d12
			alloc22 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc22)
			d4 = snap13
			d5 = snap14
			d6 = snap15
			d7 = snap16
			d8 = snap17
			d9 = snap18
			d10 = snap19
			d11 = snap20
			d12 = snap21
		}
		if !bbs[1].Rendered {
			return bbs[1].Render()
		}
		return result
		return result
	}
	bbs[1].Render = func() scm.JITValueDesc {
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		bbs[1].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl2)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d24 = result
		ctx.EnsureDesc(&d23)
		if d23.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d23, &d24)
		} else {
			switch d23.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d24, d23)
			case scm.TagInt:
				ctx.EmitMakeInt(d24, d23)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d24, d23)
			case scm.TagNil:
				ctx.EmitMakeNil(d24)
			default:
				ctx.EmitMovPairToResult(&d23, &d24)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[2].Render = func() scm.JITValueDesc {
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
			return result
		}
		bbs[2].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl3)
		ctx.ResolveFixups()
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
		d26 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).findChunk), []scm.JITValueDesc{thisptr, idxInt}, 1)
		d26.NoHeapPointer = true
		ctx.BindReg(d26.Reg, &d26)
		ctx.StabilizeDescForControlFlow(&d26)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d27 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r6 := ctx.AllocReg()
			r7 := ctx.AllocRegExcept(r6)
			r8 := ctx.AllocRegExcept(r6, r7)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r6, thisptr.Reg, off)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+16)
			d27 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r6, Reg2: r7, Reg3: r8}
			ctx.BindReg(r6, &d27)
			ctx.BindReg(r7, &d27)
			ctx.BindReg(r8, &d27)
			ctx.BindReg(r6, &d27)
			ctx.BindReg(r7, &d27)
			ctx.BindReg(r8, &d27)
		}
		if d27.SliceSizeKnown {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d27.KnownSliceLen))}
		} else if d27.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d27.StackOff))}
		} else if d27.Loc == scm.LocStackTriple {
			d28 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d27.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d27)
			if d27.Loc == scm.LocRegPair || d27.Loc == scm.LocRegTriple {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg2, ID: 0}
			} else if d27.Loc == scm.LocReg {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d28)
		ctx.EnsureDescsTogether(&d26, &d28)
		if d26.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d26.Imm.Int() >= d28.Imm.Int())}
		} else if d28.Loc == scm.LocImm {
			r9 := ctx.AllocRegExcept(d26.Reg)
			if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d26.Reg, int32(d28.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d28.Imm.Int()))
				ctx.EmitCmpInt64(d26.Reg, ctx.ScratchReg)
			}
			d29 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r9, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r9, &d29)
		} else if d26.Loc == scm.LocImm {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d26.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d28.Reg)
			d29 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r10, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r10, &d29)
		} else {
			r11 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitCmpInt64(d26.Reg, d28.Reg)
			d29 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r11, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r11, &d29)
		}
		ctx.FreeDesc(&d28)
		d30 = d29
		ctx.EnsureDesc(&d30)
		if d30.Loc != scm.LocImm && d30.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d30.Loc == scm.LocImm {
			if d30.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[4].Render()
		}
		ctx.EmitJump(d30.Condition, lbl4)
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
		}
		ctx.FreeDesc(&d29)
		ctx.FlushRegisterMoves()
		if !bbs[4].Rendered {
			snap31 := d4
			snap32 := d5
			snap33 := d6
			snap34 := d7
			snap35 := d8
			snap36 := d9
			snap37 := d10
			snap38 := d11
			snap39 := d12
			snap40 := d23
			snap41 := d24
			snap42 := d25
			snap43 := d26
			snap44 := d27
			snap45 := d28
			snap46 := d29
			snap47 := d30
			alloc48 := ctx.SnapshotAllocState()
			bbs[4].Render()
			ctx.RestoreAllocState(alloc48)
			d4 = snap31
			d5 = snap32
			d6 = snap33
			d7 = snap34
			d8 = snap35
			d9 = snap36
			d10 = snap37
			d11 = snap38
			d12 = snap39
			d23 = snap40
			d24 = snap41
			d25 = snap42
			d26 = snap43
			d27 = snap44
			d28 = snap45
			d29 = snap46
			d30 = snap47
		}
		if !bbs[3].Rendered {
			return bbs[3].Render()
		}
		return result
		return result
	}
	bbs[3].Render = func() scm.JITValueDesc {
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
			return result
		}
		bbs[3].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl4)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d50 = result
		ctx.EnsureDesc(&d49)
		if d49.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d49, &d50)
		} else {
			switch d49.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d50, d49)
			case scm.TagInt:
				ctx.EmitMakeInt(d50, d49)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d50, d49)
			case scm.TagNil:
				ctx.EmitMakeNil(d50)
			default:
				ctx.EmitMovPairToResult(&d49, &d50)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[4].Render = func() scm.JITValueDesc {
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		bbs[4].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		if d26.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d26.Imm.Int() > 0)}
		} else {
			r12 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitCmpRegImm32(d26.Reg, 0)
			d51 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r12, Condition: scm.CondSignedGreater}
			ctx.BindReg(r12, &d51)
		}
		d52 = d51
		ctx.EnsureDesc(&d52)
		if d52.Loc != scm.LocImm && d52.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d52.Loc == scm.LocImm {
			if d52.Imm.Bool() {
				return bbs[5].Render()
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
			return bbs[6].Render()
		}
		lbl13 := ctx.ReserveLabel()
		ctx.EmitJump(d52.Condition, lbl6)
		ctx.EmitJmp(lbl13)
		ctx.FreeDesc(&d51)
		snap53 := d4
		snap54 := d5
		snap55 := d6
		snap56 := d7
		snap57 := d8
		snap58 := d9
		snap59 := d10
		snap60 := d11
		snap61 := d12
		snap62 := d23
		snap63 := d24
		snap64 := d25
		snap65 := d26
		snap66 := d27
		snap67 := d28
		snap68 := d29
		snap69 := d30
		snap70 := d49
		snap71 := d50
		snap72 := d51
		snap73 := d52
		alloc74 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl13)
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc74)
		d4 = snap53
		d5 = snap54
		d6 = snap55
		d7 = snap56
		d8 = snap57
		d9 = snap58
		d10 = snap59
		d11 = snap60
		d12 = snap61
		d23 = snap62
		d24 = snap63
		d25 = snap64
		d26 = snap65
		d27 = snap66
		d28 = snap67
		d29 = snap68
		d30 = snap69
		d49 = snap70
		d50 = snap71
		d51 = snap72
		d52 = snap73
		if !bbs[6].Rendered {
			snap75 := d4
			snap76 := d5
			snap77 := d6
			snap78 := d7
			snap79 := d8
			snap80 := d9
			snap81 := d10
			snap82 := d11
			snap83 := d12
			snap84 := d23
			snap85 := d24
			snap86 := d25
			snap87 := d26
			snap88 := d27
			snap89 := d28
			snap90 := d29
			snap91 := d30
			snap92 := d49
			snap93 := d50
			snap94 := d51
			snap95 := d52
			alloc96 := ctx.SnapshotAllocState()
			bbs[6].Render()
			ctx.RestoreAllocState(alloc96)
			d4 = snap75
			d5 = snap76
			d6 = snap77
			d7 = snap78
			d8 = snap79
			d9 = snap80
			d10 = snap81
			d11 = snap82
			d12 = snap83
			d23 = snap84
			d24 = snap85
			d25 = snap86
			d26 = snap87
			d27 = snap88
			d28 = snap89
			d29 = snap90
			d30 = snap91
			d49 = snap92
			d50 = snap93
			d51 = snap94
			d52 = snap95
		}
		if !bbs[5].Rendered {
			return bbs[5].Render()
		}
		return result
		return result
	}
	bbs[5].Render = func() scm.JITValueDesc {
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
			return result
		}
		bbs[5].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl6)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d26)
		if d26.Loc == scm.LocImm {
			d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(scratch, d26.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, 1)
			d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d97)
		}
		if d97.Loc == scm.LocReg && d26.Loc == scm.LocReg && d97.Reg == d26.Reg {
			ctx.TransferReg(d26.Reg)
			d26.Loc = scm.LocNone
		}
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d97.Loc == scm.LocRegPair || d97.Loc == scm.LocStackPair || d97.Loc == scm.LocRegTriple || d97.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d97)
		d98 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).jumpCum), []scm.JITValueDesc{thisptr, d97}, 1)
		d98.NoHeapPointer = true
		ctx.BindReg(d98.Reg, &d98)
		ctx.StabilizeDescForControlFlow(&d98)
		ctx.FreeDesc(&d97)
		ctx.SyncDesc(&d98)
		if d98.Loc == scm.LocReg || d98.Loc == scm.LocFPReg {
			ctx.ProtectReg(d98.Reg)
		} else if d98.Loc == scm.LocRegPair {
			ctx.ProtectReg(d98.Reg)
			ctx.ProtectReg(d98.Reg2)
		}
		d99 = d98
		if d99.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d99)
		ctx.EmitStoreToStack(d99, int32(bbs[6].PhiBase)+int32(0))
		if d98.Loc == scm.LocReg || d98.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d98.Reg)
		} else if d98.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d98.Reg)
			ctx.UnprotectReg(d98.Reg2)
		}
		return bbs[6].Render()
		return result
	}
	bbs[6].Render = func() scm.JITValueDesc {
		if bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		bbs[6].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d100 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r13 := ctx.AllocReg()
			r14 := ctx.AllocRegExcept(r13)
			r15 := ctx.AllocRegExcept(r13, r14)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r13, thisptr.Reg, off)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+16)
			d100 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r13, Reg2: r14, Reg3: r15}
			ctx.BindReg(r13, &d100)
			ctx.BindReg(r14, &d100)
			ctx.BindReg(r15, &d100)
			ctx.BindReg(r13, &d100)
			ctx.BindReg(r14, &d100)
			ctx.BindReg(r15, &d100)
		}
		if d100.SliceSizeKnown {
			d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d100.KnownSliceLen))}
		} else if d100.Loc == scm.LocImm {
			d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d100.StackOff))}
		} else if d100.Loc == scm.LocStackTriple {
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d100.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d100)
			if d100.Loc == scm.LocRegPair || d100.Loc == scm.LocRegTriple {
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d100.Reg2, ID: 0}
			} else if d100.Loc == scm.LocReg {
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d100.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d101)
		ctx.EnsureDesc(&d101)
		if d101.Loc == scm.LocImm {
			d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d101.Reg)
			ctx.EmitMovRegReg(scratch, d101.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, 1)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d102)
		}
		if d102.Loc == scm.LocReg && d101.Loc == scm.LocReg && d102.Reg == d101.Reg {
			ctx.TransferReg(d101.Reg)
			d101.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d101)
		ctx.EnsureDesc(&d102)
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d102, &d26)
		if d102.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() - d26.Imm.Int())}
		} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
			ctx.EnsureDesc(&d102)
			r16 := ctx.AllocRegExcept(d102.Reg)
			ctx.EmitMovRegReg(r16, d102.Reg)
			d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d103)
		} else if d102.Loc == scm.LocImm {
			ctx.EnsureDesc(&d26)
			scratch := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d26)
			d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d103)
		} else if d26.Loc == scm.LocImm {
			ctx.EnsureDesc(&d102)
			scratch := ctx.AllocRegExcept(d102.Reg)
			ctx.EmitMovRegReg(scratch, d102.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d26.Imm.Int())
			d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d103)
		} else {
			ctx.EnsureDesc(&d102)
			ctx.SyncDesc(&d26)
			r17 := ctx.AllocRegExcept(d102.Reg, d26.Reg)
			ctx.EmitMovRegReg(r17, d102.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r17, &d26)
			d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d103)
		}
		if d103.Loc == scm.LocReg && d102.Loc == scm.LocReg && d103.Reg == d102.Reg {
			ctx.TransferReg(d102.Reg)
			d102.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d102)
		ctx.EnsureDesc(&d103)
		d104 = ctx.EmitLoadScalarSliceElement(&d100, &d103, 8, scm.TagInt)
		ctx.FreeDesc(&d103)
		ctx.StabilizeDescForControlFlow(&d104)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDescsTogether(&idxInt, &d4)
		if idxInt.Loc == scm.LocImm && d4.Loc == scm.LocImm {
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d4.Imm.Int())}
		} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
			ctx.EnsureDesc(&idxInt)
			r18 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r18, idxInt.Reg)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d105)
		} else if idxInt.Loc == scm.LocImm {
			ctx.EnsureDesc(&d4)
			scratch := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d4)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d105)
		} else if d4.Loc == scm.LocImm {
			ctx.EnsureDesc(&idxInt)
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d4.Imm.Int())
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d105)
		} else {
			ctx.EnsureDesc(&idxInt)
			ctx.SyncDesc(&d4)
			r19 := ctx.AllocRegExcept(idxInt.Reg, d4.Reg)
			ctx.EmitMovRegReg(r19, idxInt.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r19, &d4)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d105)
		}
		if d105.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d105.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d105)
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d4)
		ctx.SyncDesc(&d104)
		if d104.Loc == scm.LocReg || d104.Loc == scm.LocFPReg {
			ctx.ProtectReg(d104.Reg)
		} else if d104.Loc == scm.LocRegPair {
			ctx.ProtectReg(d104.Reg)
			ctx.ProtectReg(d104.Reg2)
		}
		d106 = d104
		if d106.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d106)
		if phiHomeOK2 {
			ctx.EmitMovToReg(r0, d106)
		} else {
			ctx.EmitStoreToStack(d106, int32(bbs[7].PhiBase)+int32(0))
		}
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
		if phiHomeOK3 {
			ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
		} else {
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(32))
		}
		if d104.Loc == scm.LocReg || d104.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d104.Reg)
		} else if d104.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d104.Reg)
			ctx.UnprotectReg(d104.Reg2)
		}
		return bbs[7].Render()
		return result
	}
	bbs[7].Render = func() scm.JITValueDesc {
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		bbs[7].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl8)
		ctx.ResolveFixups()
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
		if phiHomeOK2 && d5.Loc == scm.LocReg {
			ctx.BindReg(r0, &d5)
		}
		if phiHomeOK3 && d7.Loc == scm.LocReg {
			ctx.BindReg(r1, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d105)
		ctx.EnsureDescsTogether(&d7, &d105)
		if d7.Loc == scm.LocImm && d105.Loc == scm.LocImm {
			d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d7.Imm.Int() <= d105.Imm.Int())}
		} else if d105.Loc == scm.LocImm {
			r20 := ctx.AllocRegExcept(d7.Reg)
			if d105.Imm.Int() >= -2147483648 && d105.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d7.Reg, int32(d105.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d105.Imm.Int()))
				ctx.EmitCmpInt64(d7.Reg, ctx.ScratchReg)
			}
			d107 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r20, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r20, &d107)
		} else if d7.Loc == scm.LocImm {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d7.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d105.Reg)
			d107 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r21, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r21, &d107)
		} else {
			r22 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpInt64(d7.Reg, d105.Reg)
			d107 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r22, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r22, &d107)
		}
		d108 = d107
		ctx.EnsureDesc(&d108)
		if d108.Loc != scm.LocImm && d108.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d108.Loc == scm.LocImm {
			if d108.Imm.Bool() {
				return bbs[8].Render()
			}
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d109 = d6
			if d109.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d109)
			if d109.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d109, int32(bbs[9].PhiBase)+int32(0), 2)
			} else if d109.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d109)
				ctx.EmitStoreScmerToStack(d109, int32(bbs[9].PhiBase)+int32(0))
			} else if d109.Loc == scm.LocRegPair || d109.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d109, int32(bbs[9].PhiBase)+int32(0))
			} else {
				ctx.EnsureDesc(&d109)
				ctx.EmitStoreToStack(d109, int32(bbs[9].PhiBase)+int32(0))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
			}
			if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
			return bbs[9].Render()
		}
		lbl14 := ctx.ReserveLabel()
		ctx.EmitJump(d108.Condition, lbl9)
		ctx.EmitJmp(lbl14)
		ctx.FreeDesc(&d107)
		snap110 := d4
		snap111 := d5
		snap112 := d6
		snap113 := d7
		snap114 := d8
		snap115 := d9
		snap116 := d10
		snap117 := d11
		snap118 := d12
		snap119 := d23
		snap120 := d24
		snap121 := d25
		snap122 := d26
		snap123 := d27
		snap124 := d28
		snap125 := d29
		snap126 := d30
		snap127 := d49
		snap128 := d50
		snap129 := d51
		snap130 := d52
		snap131 := d97
		snap132 := d98
		snap133 := d99
		snap134 := d100
		snap135 := d101
		snap136 := d102
		snap137 := d103
		snap138 := d104
		snap139 := d105
		snap140 := d106
		snap141 := d107
		snap142 := d108
		snap143 := d109
		alloc144 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl14)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d145 = d6
		if d145.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.SyncDesc(&d145)
		if d145.Loc == scm.LocStackPair {
			ctx.EmitCopyStackWords(d145, int32(bbs[9].PhiBase)+int32(0), 2)
		} else if d145.Loc == scm.LocInputPair {
			ctx.EnsureDesc(&d145)
			ctx.EmitStoreScmerToStack(d145, int32(bbs[9].PhiBase)+int32(0))
		} else if d145.Loc == scm.LocRegPair || d145.Loc == scm.LocImm {
			ctx.EmitStoreScmerToStack(d145, int32(bbs[9].PhiBase)+int32(0))
		} else {
			ctx.EnsureDesc(&d145)
			ctx.EmitStoreToStack(d145, int32(bbs[9].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
		}
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl10)
		ctx.RestoreAllocState(alloc144)
		d4 = snap110
		d5 = snap111
		d6 = snap112
		d7 = snap113
		d8 = snap114
		d9 = snap115
		d10 = snap116
		d11 = snap117
		d12 = snap118
		d23 = snap119
		d24 = snap120
		d25 = snap121
		d26 = snap122
		d27 = snap123
		d28 = snap124
		d29 = snap125
		d30 = snap126
		d49 = snap127
		d50 = snap128
		d51 = snap129
		d52 = snap130
		d97 = snap131
		d98 = snap132
		d99 = snap133
		d100 = snap134
		d101 = snap135
		d102 = snap136
		d103 = snap137
		d104 = snap138
		d105 = snap139
		d106 = snap140
		d107 = snap141
		d108 = snap142
		d109 = snap143
		if !bbs[9].Rendered {
			snap146 := d4
			snap147 := d5
			snap148 := d6
			snap149 := d7
			snap150 := d8
			snap151 := d9
			snap152 := d10
			snap153 := d11
			snap154 := d12
			snap155 := d23
			snap156 := d24
			snap157 := d25
			snap158 := d26
			snap159 := d27
			snap160 := d28
			snap161 := d29
			snap162 := d30
			snap163 := d49
			snap164 := d50
			snap165 := d51
			snap166 := d52
			snap167 := d97
			snap168 := d98
			snap169 := d99
			snap170 := d100
			snap171 := d101
			snap172 := d102
			snap173 := d103
			snap174 := d104
			snap175 := d105
			snap176 := d106
			snap177 := d107
			snap178 := d108
			snap179 := d109
			snap180 := d145
			alloc181 := ctx.SnapshotAllocState()
			bbs[9].Render()
			ctx.RestoreAllocState(alloc181)
			d4 = snap146
			d5 = snap147
			d6 = snap148
			d7 = snap149
			d8 = snap150
			d9 = snap151
			d10 = snap152
			d11 = snap153
			d12 = snap154
			d23 = snap155
			d24 = snap156
			d25 = snap157
			d26 = snap158
			d27 = snap159
			d28 = snap160
			d29 = snap161
			d30 = snap162
			d49 = snap163
			d50 = snap164
			d51 = snap165
			d52 = snap166
			d97 = snap167
			d98 = snap168
			d99 = snap169
			d100 = snap170
			d101 = snap171
			d102 = snap172
			d103 = snap173
			d104 = snap174
			d105 = snap175
			d106 = snap176
			d107 = snap177
			d108 = snap178
			d109 = snap179
			d145 = snap180
		}
		if !bbs[8].Rendered {
			return bbs[8].Render()
		}
		return result
		return result
	}
	bbs[8].Render = func() scm.JITValueDesc {
		if bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		bbs[8].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		r23 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r23, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r23, thisptr.Reg)
			ctx.EmitAddRegImm32(r23, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r23, &d183)
		d184 = ctx.EmitLoadScalarSliceElement(&d183, &d182, 8, scm.TagInt)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d184)
		ctx.EnsureDescsTogether(&d5, &d184)
		if d5.Loc == scm.LocImm && d184.Loc == scm.LocImm {
			d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d5.Imm.Int()) < uint64(d184.Imm.Int()))}
		} else if d184.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d5.Reg)
			if d184.Imm.Int() >= -2147483648 && d184.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d5.Reg, int32(d184.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d184.Imm.Int()))
				ctx.EmitCmpInt64(d5.Reg, ctx.ScratchReg)
			}
			d185 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r24, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r24, &d185)
		} else if d5.Loc == scm.LocImm {
			r25 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d5.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d184.Reg)
			d185 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r25, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r25, &d185)
		} else {
			r26 := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitCmpInt64(d5.Reg, d184.Reg)
			d185 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r26, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r26, &d185)
		}
		ctx.FreeDesc(&d184)
		d186 = d185
		ctx.EnsureDesc(&d186)
		if d186.Loc != scm.LocImm && d186.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d186.Loc == scm.LocImm {
			if d186.Imm.Bool() {
				return bbs[10].Render()
			}
			return bbs[11].Render()
		}
		ctx.EmitJump(d186.Condition, lbl11)
		if bbs[11].Rendered {
			ctx.EmitJmp(lbl12)
		}
		ctx.FreeDesc(&d185)
		ctx.FlushRegisterMoves()
		if !bbs[11].Rendered {
			snap187 := d4
			snap188 := d5
			snap189 := d6
			snap190 := d7
			snap191 := d8
			snap192 := d9
			snap193 := d10
			snap194 := d11
			snap195 := d12
			snap196 := d23
			snap197 := d24
			snap198 := d25
			snap199 := d26
			snap200 := d27
			snap201 := d28
			snap202 := d29
			snap203 := d30
			snap204 := d49
			snap205 := d50
			snap206 := d51
			snap207 := d52
			snap208 := d97
			snap209 := d98
			snap210 := d99
			snap211 := d100
			snap212 := d101
			snap213 := d102
			snap214 := d103
			snap215 := d104
			snap216 := d105
			snap217 := d106
			snap218 := d107
			snap219 := d108
			snap220 := d109
			snap221 := d145
			snap222 := d182
			snap223 := d183
			snap224 := d184
			snap225 := d185
			snap226 := d186
			alloc227 := ctx.SnapshotAllocState()
			bbs[11].Render()
			ctx.RestoreAllocState(alloc227)
			d4 = snap187
			d5 = snap188
			d6 = snap189
			d7 = snap190
			d8 = snap191
			d9 = snap192
			d10 = snap193
			d11 = snap194
			d12 = snap195
			d23 = snap196
			d24 = snap197
			d25 = snap198
			d26 = snap199
			d27 = snap200
			d28 = snap201
			d29 = snap202
			d30 = snap203
			d49 = snap204
			d50 = snap205
			d51 = snap206
			d52 = snap207
			d97 = snap208
			d98 = snap209
			d99 = snap210
			d100 = snap211
			d101 = snap212
			d102 = snap213
			d103 = snap214
			d104 = snap215
			d105 = snap216
			d106 = snap217
			d107 = snap218
			d108 = snap219
			d109 = snap220
			d145 = snap221
			d182 = snap222
			d183 = snap223
			d184 = snap224
			d185 = snap225
			d186 = snap226
		}
		if !bbs[10].Rendered {
			return bbs[10].Render()
		}
		return result
		return result
	}
	bbs[9].Render = func() scm.JITValueDesc {
		if bbs[9].Rendered {
			ctx.EmitJmp(lbl10)
			return result
		}
		bbs[9].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl10)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		d228 = result
		ctx.EnsureDesc(&d8)
		if d8.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d8, &d228)
		} else {
			switch d8.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d228, d8)
			case scm.TagInt:
				ctx.EmitMakeInt(d228, d8)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d228, d8)
			case scm.TagNil:
				ctx.EmitMakeNil(d228)
			default:
				ctx.EmitMovPairToResult(&d8, &d228)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[10].Render = func() scm.JITValueDesc {
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
			return result
		}
		bbs[10].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl11)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		r27 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r27, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r27, thisptr.Reg)
			ctx.EmitAddRegImm32(r27, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r27, &d230)
		d232 = ctx.EmitSliceElementAddress(&d230, &d229, 16)
		ctx.EmitLoadScmerToStack(&d232, int32(bbs[9].PhiBase)+int32(0))
		ctx.FreeDesc(&d232)
		d231 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: int32(bbs[9].PhiBase) + int32(0)}
		ctx.StabilizeDescForControlFlow(&d231)
		ctx.SyncDesc(&d231)
		if d231.Loc == scm.LocReg || d231.Loc == scm.LocFPReg {
			ctx.ProtectReg(d231.Reg)
		} else if d231.Loc == scm.LocRegPair {
			ctx.ProtectReg(d231.Reg)
			ctx.ProtectReg(d231.Reg2)
		}
		d233 = d231
		if d233.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.SyncDesc(&d233)
		if d233.Loc == scm.LocStackPair {
			ctx.EmitCopyStackWords(d233, int32(bbs[9].PhiBase)+int32(0), 2)
		} else if d233.Loc == scm.LocInputPair {
			ctx.EnsureDesc(&d233)
			ctx.EmitStoreScmerToStack(d233, int32(bbs[9].PhiBase)+int32(0))
		} else if d233.Loc == scm.LocRegPair || d233.Loc == scm.LocImm {
			ctx.EmitStoreScmerToStack(d233, int32(bbs[9].PhiBase)+int32(0))
		} else {
			ctx.EnsureDesc(&d233)
			ctx.EmitStoreToStack(d233, int32(bbs[9].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[9].PhiBase)+int32(0))+8)
		}
		if d231.Loc == scm.LocReg || d231.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d231.Reg)
		} else if d231.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d231.Reg)
			ctx.UnprotectReg(d231.Reg2)
		}
		return bbs[9].Render()
		return result
	}
	bbs[11].Render = func() scm.JITValueDesc {
		if bbs[11].Rendered {
			ctx.EmitJmp(lbl12)
			return result
		}
		bbs[11].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&d5)
		d234 = d5
		_ = d234
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
		ctx.EnsureDesc(&d234)
		if d234.Loc == scm.LocImm {
			d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d234.Imm.Int() & 255)}
		} else {
			r28 := ctx.AllocRegExcept(d234.Reg)
			ctx.EmitMovRegReg(r28, d234.Reg)
			ctx.EmitAndRegImm32(r28, int32(255))
			d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d235)
		}
		if d235.Loc == scm.LocReg && d234.Loc == scm.LocReg && d235.Reg == d234.Reg {
			ctx.TransferReg(d234.Reg)
			d234.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d235.Loc == scm.LocRegPair || d235.Loc == scm.LocStackPair || d235.Loc == scm.LocRegTriple || d235.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d235)
		d236 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).decodeSymbol), []scm.JITValueDesc{thisptr, d235}, 1)
		d236.NoHeapPointer = true
		ctx.BindReg(d236.Reg, &d236)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d236)
		r29 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r29, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).widths)))
		} else {
			ctx.EmitMovRegReg(r29, thisptr.Reg)
			ctx.EmitAddRegImm32(r29, int32(unsafe.Offsetof((*StorageEnum)(nil).widths)))
		}
		d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r29, &d237)
		ctx.ReclaimUntrackedRegs()
		d238 = ctx.EmitLoadScalarSliceElement(&d237, &d236, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d236)
		r30 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r30, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r30, thisptr.Reg)
			ctx.EmitAddRegImm32(r30, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d239 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r30, &d239)
		ctx.ReclaimUntrackedRegs()
		d241 = ctx.EmitSliceElementAddress(&d239, &d236, 16)
		ctx.EnsureDesc(&d241)
		r31 := ctx.AllocRegExcept(d241.Reg)
		ctx.EmitMovRegMem(r31, d241.Reg, 8)
		ctx.EmitMovRegMem(d241.Reg, d241.Reg, 0)
		d240 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d241.Reg, Reg2: r31}
		ctx.BindReg(d241.Reg, &d240)
		ctx.BindReg(r31, &d240)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d234)
		if d234.Loc == scm.LocImm {
			d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d234.Imm.Int()) >> 8))}
		} else {
			r32 := ctx.AllocRegExcept(d234.Reg)
			ctx.EmitMovRegReg(r32, d234.Reg)
			ctx.EmitShrRegImm8(r32, 8)
			d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d242)
		}
		if d242.Loc == scm.LocReg && d234.Loc == scm.LocReg && d242.Reg == d234.Reg {
			ctx.TransferReg(d234.Reg)
			d234.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d242)
		ctx.EnsureDesc(&d238)
		ctx.EnsureDescsTogether(&d242, &d238)
		if d242.Loc == scm.LocImm && d238.Loc == scm.LocImm {
			d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d242.Imm.Int() * d238.Imm.Int())}
		} else if d242.Loc == scm.LocImm {
			ctx.EnsureDesc(&d238)
			scratch := ctx.AllocRegExcept(d238.Reg)
			ctx.EmitMovRegReg(scratch, d238.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d242.Imm.Int())
			d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d243)
		} else if d238.Loc == scm.LocImm {
			ctx.EnsureDesc(&d242)
			scratch := ctx.AllocRegExcept(d242.Reg)
			ctx.EmitMovRegReg(scratch, d242.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d238.Imm.Int())
			d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d243)
		} else {
			ctx.EnsureDesc(&d242)
			ctx.SyncDesc(&d238)
			r33 := ctx.AllocRegExcept(d242.Reg, d238.Reg)
			ctx.EmitMovRegReg(r33, d242.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r33, &d238)
			d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d243)
		}
		if d243.Loc == scm.LocReg && d242.Loc == scm.LocReg && d243.Reg == d242.Reg {
			ctx.TransferReg(d242.Reg)
			d242.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d242)
		ctx.FreeDesc(&d238)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d243)
		ctx.EnsureDesc(&d235)
		ctx.EnsureDescsTogether(&d243, &d235)
		if d243.Loc == scm.LocImm && d235.Loc == scm.LocImm {
			d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() + d235.Imm.Int())}
		} else if d235.Loc == scm.LocImm && d235.Imm.Int() == 0 {
			ctx.EnsureDesc(&d243)
			r34 := ctx.AllocRegExcept(d243.Reg)
			ctx.EmitMovRegReg(r34, d243.Reg)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d244)
		} else if d243.Loc == scm.LocImm && d243.Imm.Int() == 0 {
			ctx.EnsureDesc(&d235)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d235.Reg}
			ctx.BindReg(d235.Reg, &d244)
		} else if d243.Loc == scm.LocImm {
			ctx.EnsureDesc(&d235)
			scratch := ctx.AllocRegExcept(d235.Reg)
			ctx.EmitMovRegReg(scratch, d235.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d243.Imm.Int())
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d244)
		} else if d235.Loc == scm.LocImm {
			ctx.EnsureDesc(&d243)
			scratch := ctx.AllocRegExcept(d243.Reg)
			ctx.EmitMovRegReg(scratch, d243.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d235.Imm.Int())
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d244)
		} else {
			ctx.EnsureDesc(&d243)
			ctx.SyncDesc(&d235)
			r35 := ctx.AllocRegExcept(d243.Reg, d235.Reg)
			ctx.EmitMovRegReg(r35, d243.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r35, &d235)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d244)
		}
		if d244.Loc == scm.LocReg && d243.Loc == scm.LocReg && d244.Reg == d243.Reg {
			ctx.TransferReg(d243.Reg)
			d243.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d243)
		ctx.FreeDesc(&d235)
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d236.Loc == scm.LocRegPair || d236.Loc == scm.LocStackPair || d236.Loc == scm.LocRegTriple || d236.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d236)
		d245 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).symbolLo), []scm.JITValueDesc{thisptr, d236}, 1)
		d245.NoHeapPointer = true
		ctx.BindReg(d245.Reg, &d245)
		ctx.FreeDesc(&d236)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d244)
		ctx.EnsureDesc(&d245)
		ctx.EnsureDescsTogether(&d244, &d245)
		if d244.Loc == scm.LocImm && d245.Loc == scm.LocImm {
			d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d244.Imm.Int() - d245.Imm.Int())}
		} else if d245.Loc == scm.LocImm && d245.Imm.Int() == 0 {
			ctx.EnsureDesc(&d244)
			r36 := ctx.AllocRegExcept(d244.Reg)
			ctx.EmitMovRegReg(r36, d244.Reg)
			d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d246)
		} else if d244.Loc == scm.LocImm {
			ctx.EnsureDesc(&d245)
			scratch := ctx.AllocRegExcept(d245.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d244.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d245)
			d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d246)
		} else if d245.Loc == scm.LocImm {
			ctx.EnsureDesc(&d244)
			scratch := ctx.AllocRegExcept(d244.Reg)
			ctx.EmitMovRegReg(scratch, d244.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d245.Imm.Int())
			d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d246)
		} else {
			ctx.EnsureDesc(&d244)
			ctx.SyncDesc(&d245)
			r37 := ctx.AllocRegExcept(d244.Reg, d245.Reg)
			ctx.EmitMovRegReg(r37, d244.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r37, &d245)
			d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d246)
		}
		if d246.Loc == scm.LocReg && d244.Loc == scm.LocReg && d246.Reg == d244.Reg {
			ctx.TransferReg(d244.Reg)
			d244.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d244)
		ctx.FreeDesc(&d245)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d240)
		ctx.EnsureDesc(&d246)
		ctx.StabilizeDescForControlFlow(&d240)
		ctx.StabilizeDescForControlFlow(&d246)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d7)
		if d7.Loc == scm.LocImm {
			d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + 1)}
		} else {
			var scratch scm.Reg
			if phiHomeOK3 {
				scratch = r1
			} else {
				scratch = ctx.AllocRegExcept(d7.Reg)
			}
			ctx.EmitMovRegReg(scratch, d7.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d247)
		}
		if d247.Loc == scm.LocReg && d7.Loc == scm.LocReg && d247.Reg == d7.Reg {
			ctx.TransferReg(d7.Reg)
			d7.Loc = scm.LocNone
		}
		ctx.SyncDesc(&d240)
		if d240.Loc == scm.LocReg || d240.Loc == scm.LocFPReg {
			ctx.ProtectReg(d240.Reg)
		} else if d240.Loc == scm.LocRegPair {
			ctx.ProtectReg(d240.Reg)
			ctx.ProtectReg(d240.Reg2)
		}
		ctx.SyncDesc(&d246)
		if d246.Loc == scm.LocReg || d246.Loc == scm.LocFPReg {
			ctx.ProtectReg(d246.Reg)
		} else if d246.Loc == scm.LocRegPair {
			ctx.ProtectReg(d246.Reg)
			ctx.ProtectReg(d246.Reg2)
		}
		ctx.SyncDesc(&d247)
		if d247.Loc == scm.LocReg || d247.Loc == scm.LocFPReg {
			ctx.ProtectReg(d247.Reg)
		} else if d247.Loc == scm.LocRegPair {
			ctx.ProtectReg(d247.Reg)
			ctx.ProtectReg(d247.Reg2)
		}
		d248 = d246
		if d248.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d248)
		if phiHomeOK2 {
			ctx.EmitMovToReg(r0, d248)
		} else {
			ctx.EmitStoreToStack(d248, int32(bbs[7].PhiBase)+int32(0))
		}
		d249 = d240
		if d249.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.SyncDesc(&d249)
		if d249.Loc == scm.LocStackPair {
			ctx.EmitCopyStackWords(d249, int32(bbs[7].PhiBase)+int32(16), 2)
		} else if d249.Loc == scm.LocInputPair {
			ctx.EnsureDesc(&d249)
			ctx.EmitStoreScmerToStack(d249, int32(bbs[7].PhiBase)+int32(16))
		} else if d249.Loc == scm.LocRegPair || d249.Loc == scm.LocImm {
			ctx.EmitStoreScmerToStack(d249, int32(bbs[7].PhiBase)+int32(16))
		} else {
			ctx.EnsureDesc(&d249)
			ctx.EmitStoreToStack(d249, int32(bbs[7].PhiBase)+int32(16))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
		}
		d250 = d247
		if d250.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d250)
		if phiHomeOK3 {
			ctx.EmitMovToReg(r1, d250)
		} else {
			ctx.EmitStoreToStack(d250, int32(bbs[7].PhiBase)+int32(32))
		}
		if d240.Loc == scm.LocReg || d240.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d240.Reg)
		} else if d240.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d240.Reg)
			ctx.UnprotectReg(d240.Reg2)
		}
		if d246.Loc == scm.LocReg || d246.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d246.Reg)
		} else if d246.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d246.Reg)
			ctx.UnprotectReg(d246.Reg2)
		}
		if d247.Loc == scm.LocReg || d247.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d247.Reg)
		} else if d247.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d247.Reg)
			ctx.UnprotectReg(d247.Reg2)
		}
		return bbs[7].Render()
		return result
	}
	_ = bbs[0].Render()
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
