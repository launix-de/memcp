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
	var d8 scm.JITValueDesc
	_ = d8
	var d9 scm.JITValueDesc
	_ = d9
	var d10 scm.JITValueDesc
	_ = d10
	var d11 scm.JITValueDesc
	_ = d11
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
	var d39 scm.JITValueDesc
	_ = d39
	var d40 scm.JITValueDesc
	_ = d40
	var d41 scm.JITValueDesc
	_ = d41
	var d80 scm.JITValueDesc
	_ = d80
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d86 scm.JITValueDesc
	_ = d86
	var d111 scm.JITValueDesc
	_ = d111
	var d135 scm.JITValueDesc
	_ = d135
	var d136 scm.JITValueDesc
	_ = d136
	var d137 scm.JITValueDesc
	_ = d137
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
	var d146 scm.JITValueDesc
	_ = d146
	var d147 scm.JITValueDesc
	_ = d147
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
	var d155 scm.JITValueDesc
	_ = d155
	var d156 scm.JITValueDesc
	_ = d156
	var d159 scm.JITValueDesc
	_ = d159
	var d160 scm.JITValueDesc
	_ = d160
	var d161 scm.JITValueDesc
	_ = d161
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
	var d277 scm.JITValueDesc
	_ = d277
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
	phiBase0 := ctx.AllocStack(int32(64))
	var bbs [10]scm.BBDescriptor
	bbs[6].PhiBase = int32(phiBase0) + int32(0)
	bbs[6].PhiCount = uint16(1)
	bbs[7].PhiBase = int32(phiBase0) + int32(16)
	bbs[7].PhiCount = uint16(3)
	registerHomes1 := ctx.AllocRegisterHomes(scm.JITRegisterPlan{Slots: [16]scm.JITRegisterSlot{{Color: 0, Width: 1, Cost: 17}, {Color: 1, Width: 1, Cost: 12}}, Count: 2})
	defer ctx.ReleaseRegisterHomes(registerHomes1)
	var r0 scm.Reg
	phiHomeOK2 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
	if phiHomeOK2 {
		r0 = registerHomes1.Registers[1]
	}
	var r1 scm.Reg
	phiHomeOK3 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
	if phiHomeOK3 {
		r1 = registerHomes1.Registers[0]
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
	r2 := ctx.AllocReg()
	r3 := ctx.AllocRegExcept(r2)
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d9 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).count)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).count))
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMem(r4, thisptr.Reg, off)
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d9)
		}
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&idxInt, &d9)
		var d10 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) >= uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r5 := ctx.AllocRegExcept(idxInt.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			d10 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r5, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r5, &d10)
		} else if idxInt.Loc == scm.LocImm {
			r6 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			d10 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r6, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r6, &d10)
		} else {
			r7 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d9.Reg)
			d10 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r7, Condition: scm.CondUnsignedAboveOrEqual}
			ctx.BindReg(r7, &d10)
		}
		ctx.FreeDesc(&d9)
		d11 = d10
		ctx.EnsureDesc(&d11)
		if d11.Loc != scm.LocImm && d11.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d11.Loc == scm.LocImm {
			if d11.Imm.Bool() {
				if ps.General {
				}
				ps12 := scm.PhiState{General: ps.General}
				ps12.OverlayValues = make([]scm.JITValueDesc, 12)
				ps12.OverlayValues[4] = d4
				ps12.OverlayValues[5] = d5
				ps12.OverlayValues[6] = d6
				ps12.OverlayValues[7] = d7
				ps12.OverlayValues[8] = d8
				ps12.OverlayValues[9] = d9
				ps12.OverlayValues[10] = d10
				ps12.OverlayValues[11] = d11
				return bbs[1].RenderPS(ps12)
			}
			if ps.General {
			}
			ps13 := scm.PhiState{General: ps.General}
			ps13.OverlayValues = make([]scm.JITValueDesc, 12)
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			return bbs[2].RenderPS(ps13)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitJump(d11.Condition, lbl2)
		ctx.FreeDesc(&d10)
		snap14 := d4
		snap15 := d5
		snap16 := d6
		snap17 := d7
		snap18 := d8
		snap19 := d9
		snap20 := d10
		snap21 := d11
		alloc22 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc22)
		d4 = snap14
		d5 = snap15
		d6 = snap16
		d7 = snap17
		d8 = snap18
		d9 = snap19
		d10 = snap20
		d11 = snap21
		ctx.RestoreAllocState(alloc22)
		d4 = snap14
		d5 = snap15
		d6 = snap16
		d7 = snap17
		d8 = snap18
		d9 = snap19
		d10 = snap20
		d11 = snap21
		ps23 := scm.PhiState{General: true}
		ps23.OverlayValues = make([]scm.JITValueDesc, 12)
		ps23.OverlayValues[4] = d4
		ps23.OverlayValues[5] = d5
		ps23.OverlayValues[6] = d6
		ps23.OverlayValues[7] = d7
		ps23.OverlayValues[8] = d8
		ps23.OverlayValues[9] = d9
		ps23.OverlayValues[10] = d10
		ps23.OverlayValues[11] = d11
		ps24 := scm.PhiState{General: true}
		ps24.OverlayValues = make([]scm.JITValueDesc, 12)
		ps24.OverlayValues[4] = d4
		ps24.OverlayValues[5] = d5
		ps24.OverlayValues[6] = d6
		ps24.OverlayValues[7] = d7
		ps24.OverlayValues[8] = d8
		ps24.OverlayValues[9] = d9
		ps24.OverlayValues[10] = d10
		ps24.OverlayValues[11] = d11
		snap25 := d4
		snap26 := d5
		snap27 := d6
		snap28 := d7
		snap29 := d8
		snap30 := d9
		snap31 := d10
		snap32 := d11
		alloc33 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps24)
		}
		ctx.RestoreAllocState(alloc33)
		d4 = snap25
		d5 = snap26
		d6 = snap27
		d7 = snap28
		d8 = snap29
		d9 = snap30
		d10 = snap31
		d11 = snap32
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps23)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		ctx.ReclaimUntrackedRegs()
		d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d35 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
		ctx.BindReg(r2, &d35)
		ctx.BindReg(r3, &d35)
		ctx.EnsureDesc(&d34)
		if d34.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d34, &d35)
		} else {
			switch d34.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d35, d34)
			case scm.TagInt:
				ctx.EmitMakeInt(d35, d34)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d35, d34)
			case scm.TagNil:
				ctx.EmitMakeNil(d35)
			default:
				ctx.EmitMovPairToResult(&d34, &d35)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.StabilizeDescForControlFlow(&idxInt)
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&idxInt)
		d37 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).findChunk), []scm.JITValueDesc{thisptr, idxInt}, 1)
		d37.NoHeapPointer = true
		ctx.BindReg(d37.Reg, &d37)
		ctx.StabilizeDescForControlFlow(&d37)
		var d38 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d38 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r8 := ctx.AllocReg()
			r9 := ctx.AllocRegExcept(r8)
			r10 := ctx.AllocRegExcept(r8, r9)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r8, thisptr.Reg, off)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off+16)
			d38 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
			ctx.BindReg(r8, &d38)
			ctx.BindReg(r9, &d38)
			ctx.BindReg(r10, &d38)
			ctx.BindReg(r8, &d38)
			ctx.BindReg(r9, &d38)
			ctx.BindReg(r10, &d38)
		}
		var d39 scm.JITValueDesc
		if d38.SliceSizeKnown {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d38.KnownSliceLen))}
		} else if d38.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d38.StackOff))}
		} else if d38.Loc == scm.LocStackTriple {
			d39 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d38)
			if d38.Loc == scm.LocRegPair || d38.Loc == scm.LocRegTriple {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg2, ID: 0}
			} else if d38.Loc == scm.LocReg {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d39)
		ctx.EnsureDescsTogether(&d37, &d39)
		var d40 scm.JITValueDesc
		if d37.Loc == scm.LocImm && d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d37.Imm.Int() >= d39.Imm.Int())}
		} else if d39.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d37.Reg)
			if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d37.Reg, int32(d39.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.EmitCmpInt64(d37.Reg, scm.RegR11)
			}
			d40 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r11, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r11, &d40)
		} else if d37.Loc == scm.LocImm {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r12, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r12, &d40)
		} else {
			r13 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitCmpInt64(d37.Reg, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r13, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r13, &d40)
		}
		ctx.FreeDesc(&d39)
		d41 = d40
		ctx.EnsureDesc(&d41)
		if d41.Loc != scm.LocImm && d41.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d41.Loc == scm.LocImm {
			if d41.Imm.Bool() {
				if ps.General {
				}
				ps42 := scm.PhiState{General: ps.General}
				ps42.OverlayValues = make([]scm.JITValueDesc, 42)
				ps42.OverlayValues[4] = d4
				ps42.OverlayValues[5] = d5
				ps42.OverlayValues[6] = d6
				ps42.OverlayValues[7] = d7
				ps42.OverlayValues[8] = d8
				ps42.OverlayValues[9] = d9
				ps42.OverlayValues[10] = d10
				ps42.OverlayValues[11] = d11
				ps42.OverlayValues[34] = d34
				ps42.OverlayValues[35] = d35
				ps42.OverlayValues[36] = d36
				ps42.OverlayValues[37] = d37
				ps42.OverlayValues[38] = d38
				ps42.OverlayValues[39] = d39
				ps42.OverlayValues[40] = d40
				ps42.OverlayValues[41] = d41
				return bbs[3].RenderPS(ps42)
			}
			if ps.General {
			}
			ps43 := scm.PhiState{General: ps.General}
			ps43.OverlayValues = make([]scm.JITValueDesc, 42)
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[6] = d6
			ps43.OverlayValues[7] = d7
			ps43.OverlayValues[8] = d8
			ps43.OverlayValues[9] = d9
			ps43.OverlayValues[10] = d10
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[37] = d37
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[39] = d39
			ps43.OverlayValues[40] = d40
			ps43.OverlayValues[41] = d41
			return bbs[4].RenderPS(ps43)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d41.Condition, lbl4)
		ctx.FreeDesc(&d40)
		snap44 := d4
		snap45 := d5
		snap46 := d6
		snap47 := d7
		snap48 := d8
		snap49 := d9
		snap50 := d10
		snap51 := d11
		snap52 := d34
		snap53 := d35
		snap54 := d36
		snap55 := d37
		snap56 := d38
		snap57 := d39
		snap58 := d40
		snap59 := d41
		alloc60 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc60)
		d4 = snap44
		d5 = snap45
		d6 = snap46
		d7 = snap47
		d8 = snap48
		d9 = snap49
		d10 = snap50
		d11 = snap51
		d34 = snap52
		d35 = snap53
		d36 = snap54
		d37 = snap55
		d38 = snap56
		d39 = snap57
		d40 = snap58
		d41 = snap59
		ctx.RestoreAllocState(alloc60)
		d4 = snap44
		d5 = snap45
		d6 = snap46
		d7 = snap47
		d8 = snap48
		d9 = snap49
		d10 = snap50
		d11 = snap51
		d34 = snap52
		d35 = snap53
		d36 = snap54
		d37 = snap55
		d38 = snap56
		d39 = snap57
		d40 = snap58
		d41 = snap59
		ps61 := scm.PhiState{General: true}
		ps61.OverlayValues = make([]scm.JITValueDesc, 42)
		ps61.OverlayValues[4] = d4
		ps61.OverlayValues[5] = d5
		ps61.OverlayValues[6] = d6
		ps61.OverlayValues[7] = d7
		ps61.OverlayValues[8] = d8
		ps61.OverlayValues[9] = d9
		ps61.OverlayValues[10] = d10
		ps61.OverlayValues[11] = d11
		ps61.OverlayValues[34] = d34
		ps61.OverlayValues[35] = d35
		ps61.OverlayValues[36] = d36
		ps61.OverlayValues[37] = d37
		ps61.OverlayValues[38] = d38
		ps61.OverlayValues[39] = d39
		ps61.OverlayValues[40] = d40
		ps61.OverlayValues[41] = d41
		ps62 := scm.PhiState{General: true}
		ps62.OverlayValues = make([]scm.JITValueDesc, 42)
		ps62.OverlayValues[4] = d4
		ps62.OverlayValues[5] = d5
		ps62.OverlayValues[6] = d6
		ps62.OverlayValues[7] = d7
		ps62.OverlayValues[8] = d8
		ps62.OverlayValues[9] = d9
		ps62.OverlayValues[10] = d10
		ps62.OverlayValues[11] = d11
		ps62.OverlayValues[34] = d34
		ps62.OverlayValues[35] = d35
		ps62.OverlayValues[36] = d36
		ps62.OverlayValues[37] = d37
		ps62.OverlayValues[38] = d38
		ps62.OverlayValues[39] = d39
		ps62.OverlayValues[40] = d40
		ps62.OverlayValues[41] = d41
		snap63 := d4
		snap64 := d5
		snap65 := d6
		snap66 := d7
		snap67 := d8
		snap68 := d9
		snap69 := d10
		snap70 := d11
		snap71 := d34
		snap72 := d35
		snap73 := d36
		snap74 := d37
		snap75 := d38
		snap76 := d39
		snap77 := d40
		snap78 := d41
		alloc79 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps62)
		}
		ctx.RestoreAllocState(alloc79)
		d4 = snap63
		d5 = snap64
		d6 = snap65
		d7 = snap66
		d8 = snap67
		d9 = snap68
		d10 = snap69
		d11 = snap70
		d34 = snap71
		d35 = snap72
		d36 = snap73
		d37 = snap74
		d38 = snap75
		d39 = snap76
		d40 = snap77
		d41 = snap78
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps61)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		ctx.ReclaimUntrackedRegs()
		d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d81 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
		ctx.BindReg(r2, &d81)
		ctx.BindReg(r3, &d81)
		ctx.EnsureDesc(&d80)
		if d80.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d80, &d81)
		} else {
			switch d80.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d81, d80)
			case scm.TagInt:
				ctx.EmitMakeInt(d81, d80)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d81, d80)
			case scm.TagNil:
				ctx.EmitMakeNil(d81)
			default:
				ctx.EmitMovPairToResult(&d80, &d81)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		var d82 scm.JITValueDesc
		if d37.Loc == scm.LocImm {
			d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d37.Imm.Int() > 0)}
		} else {
			r14 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitCmpRegImm32(d37.Reg, 0)
			d82 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r14, Condition: scm.CondSignedGreater}
			ctx.BindReg(r14, &d82)
		}
		d83 = d82
		ctx.EnsureDesc(&d83)
		if d83.Loc != scm.LocImm && d83.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d83.Loc == scm.LocImm {
			if d83.Imm.Bool() {
				if ps.General {
				}
				ps84 := scm.PhiState{General: ps.General}
				ps84.OverlayValues = make([]scm.JITValueDesc, 84)
				ps84.OverlayValues[4] = d4
				ps84.OverlayValues[5] = d5
				ps84.OverlayValues[6] = d6
				ps84.OverlayValues[7] = d7
				ps84.OverlayValues[8] = d8
				ps84.OverlayValues[9] = d9
				ps84.OverlayValues[10] = d10
				ps84.OverlayValues[11] = d11
				ps84.OverlayValues[34] = d34
				ps84.OverlayValues[35] = d35
				ps84.OverlayValues[36] = d36
				ps84.OverlayValues[37] = d37
				ps84.OverlayValues[38] = d38
				ps84.OverlayValues[39] = d39
				ps84.OverlayValues[40] = d40
				ps84.OverlayValues[41] = d41
				ps84.OverlayValues[80] = d80
				ps84.OverlayValues[81] = d81
				ps84.OverlayValues[82] = d82
				ps84.OverlayValues[83] = d83
				return bbs[5].RenderPS(ps84)
			}
			if ps.General {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
			}
			ps85 := scm.PhiState{General: ps.General}
			ps85.OverlayValues = make([]scm.JITValueDesc, 84)
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[6] = d6
			ps85.OverlayValues[7] = d7
			ps85.OverlayValues[8] = d8
			ps85.OverlayValues[9] = d9
			ps85.OverlayValues[10] = d10
			ps85.OverlayValues[11] = d11
			ps85.OverlayValues[34] = d34
			ps85.OverlayValues[35] = d35
			ps85.OverlayValues[36] = d36
			ps85.OverlayValues[37] = d37
			ps85.OverlayValues[38] = d38
			ps85.OverlayValues[39] = d39
			ps85.OverlayValues[40] = d40
			ps85.OverlayValues[41] = d41
			ps85.OverlayValues[80] = d80
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			ps85.PhiValues = make([]scm.JITValueDesc, 1)
			d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps85.PhiValues[0] = d86
			return bbs[6].RenderPS(ps85)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl11 := ctx.ReserveLabel()
		ctx.EmitJump(d83.Condition, lbl6)
		ctx.EmitJmp(lbl11)
		ctx.FreeDesc(&d82)
		snap87 := d4
		snap88 := d5
		snap89 := d6
		snap90 := d7
		snap91 := d8
		snap92 := d9
		snap93 := d10
		snap94 := d11
		snap95 := d34
		snap96 := d35
		snap97 := d36
		snap98 := d37
		snap99 := d38
		snap100 := d39
		snap101 := d40
		snap102 := d41
		snap103 := d80
		snap104 := d81
		snap105 := d82
		snap106 := d83
		snap107 := d86
		alloc108 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc108)
		d4 = snap87
		d5 = snap88
		d6 = snap89
		d7 = snap90
		d8 = snap91
		d9 = snap92
		d10 = snap93
		d11 = snap94
		d34 = snap95
		d35 = snap96
		d36 = snap97
		d37 = snap98
		d38 = snap99
		d39 = snap100
		d40 = snap101
		d41 = snap102
		d80 = snap103
		d81 = snap104
		d82 = snap105
		d83 = snap106
		d86 = snap107
		ctx.MarkLabel(lbl11)
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc108)
		d4 = snap87
		d5 = snap88
		d6 = snap89
		d7 = snap90
		d8 = snap91
		d9 = snap92
		d10 = snap93
		d11 = snap94
		d34 = snap95
		d35 = snap96
		d36 = snap97
		d37 = snap98
		d38 = snap99
		d39 = snap100
		d40 = snap101
		d41 = snap102
		d80 = snap103
		d81 = snap104
		d82 = snap105
		d83 = snap106
		d86 = snap107
		ps109 := scm.PhiState{General: true}
		ps109.OverlayValues = make([]scm.JITValueDesc, 87)
		ps109.OverlayValues[4] = d4
		ps109.OverlayValues[5] = d5
		ps109.OverlayValues[6] = d6
		ps109.OverlayValues[7] = d7
		ps109.OverlayValues[8] = d8
		ps109.OverlayValues[9] = d9
		ps109.OverlayValues[10] = d10
		ps109.OverlayValues[11] = d11
		ps109.OverlayValues[34] = d34
		ps109.OverlayValues[35] = d35
		ps109.OverlayValues[36] = d36
		ps109.OverlayValues[37] = d37
		ps109.OverlayValues[38] = d38
		ps109.OverlayValues[39] = d39
		ps109.OverlayValues[40] = d40
		ps109.OverlayValues[41] = d41
		ps109.OverlayValues[80] = d80
		ps109.OverlayValues[81] = d81
		ps109.OverlayValues[82] = d82
		ps109.OverlayValues[83] = d83
		ps109.OverlayValues[86] = d86
		ps110 := scm.PhiState{General: true}
		ps110.OverlayValues = make([]scm.JITValueDesc, 87)
		ps110.OverlayValues[4] = d4
		ps110.OverlayValues[5] = d5
		ps110.OverlayValues[6] = d6
		ps110.OverlayValues[7] = d7
		ps110.OverlayValues[8] = d8
		ps110.OverlayValues[9] = d9
		ps110.OverlayValues[10] = d10
		ps110.OverlayValues[11] = d11
		ps110.OverlayValues[34] = d34
		ps110.OverlayValues[35] = d35
		ps110.OverlayValues[36] = d36
		ps110.OverlayValues[37] = d37
		ps110.OverlayValues[38] = d38
		ps110.OverlayValues[39] = d39
		ps110.OverlayValues[40] = d40
		ps110.OverlayValues[41] = d41
		ps110.OverlayValues[80] = d80
		ps110.OverlayValues[81] = d81
		ps110.OverlayValues[82] = d82
		ps110.OverlayValues[83] = d83
		ps110.OverlayValues[86] = d86
		ps110.PhiValues = make([]scm.JITValueDesc, 1)
		d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps110.PhiValues[0] = d111
		snap112 := d4
		snap113 := d5
		snap114 := d6
		snap115 := d7
		snap116 := d8
		snap117 := d9
		snap118 := d10
		snap119 := d11
		snap120 := d34
		snap121 := d35
		snap122 := d36
		snap123 := d37
		snap124 := d38
		snap125 := d39
		snap126 := d40
		snap127 := d41
		snap128 := d80
		snap129 := d81
		snap130 := d82
		snap131 := d83
		snap132 := d86
		snap133 := d111
		alloc134 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps110)
		}
		ctx.RestoreAllocState(alloc134)
		d4 = snap112
		d5 = snap113
		d6 = snap114
		d7 = snap115
		d8 = snap116
		d9 = snap117
		d10 = snap118
		d11 = snap119
		d34 = snap120
		d35 = snap121
		d36 = snap122
		d37 = snap123
		d38 = snap124
		d39 = snap125
		d40 = snap126
		d41 = snap127
		d80 = snap128
		d81 = snap129
		d82 = snap130
		d83 = snap131
		d86 = snap132
		d111 = snap133
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps109)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d37)
		var d135 scm.JITValueDesc
		if d37.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(scratch, d37.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d135)
		}
		if d135.Loc == scm.LocReg && d37.Loc == scm.LocReg && d135.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d135)
		if d135.Loc == scm.LocRegPair || d135.Loc == scm.LocStackPair || d135.Loc == scm.LocRegTriple || d135.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d135)
		d136 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).jumpCum), []scm.JITValueDesc{thisptr, d135}, 1)
		d136.NoHeapPointer = true
		ctx.BindReg(d136.Reg, &d136)
		ctx.StabilizeDescForControlFlow(&d136)
		ctx.FreeDesc(&d135)
		if ps.General {
			ctx.SyncDesc(&d136)
			if d136.Loc == scm.LocReg {
				ctx.ProtectReg(d136.Reg)
			} else if d136.Loc == scm.LocRegPair {
				ctx.ProtectReg(d136.Reg)
				ctx.ProtectReg(d136.Reg2)
			}
			d137 = d136
			if d137.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d137)
			ctx.EmitStoreToStack(d137, int32(bbs[6].PhiBase)+int32(0))
			if d136.Loc == scm.LocReg {
				ctx.UnprotectReg(d136.Reg)
			} else if d136.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d136.Reg)
				ctx.UnprotectReg(d136.Reg2)
			}
		}
		ps138 := scm.PhiState{General: ps.General}
		ps138.OverlayValues = make([]scm.JITValueDesc, 138)
		ps138.OverlayValues[4] = d4
		ps138.OverlayValues[5] = d5
		ps138.OverlayValues[6] = d6
		ps138.OverlayValues[7] = d7
		ps138.OverlayValues[8] = d8
		ps138.OverlayValues[9] = d9
		ps138.OverlayValues[10] = d10
		ps138.OverlayValues[11] = d11
		ps138.OverlayValues[34] = d34
		ps138.OverlayValues[35] = d35
		ps138.OverlayValues[36] = d36
		ps138.OverlayValues[37] = d37
		ps138.OverlayValues[38] = d38
		ps138.OverlayValues[39] = d39
		ps138.OverlayValues[40] = d40
		ps138.OverlayValues[41] = d41
		ps138.OverlayValues[80] = d80
		ps138.OverlayValues[81] = d81
		ps138.OverlayValues[82] = d82
		ps138.OverlayValues[83] = d83
		ps138.OverlayValues[86] = d86
		ps138.OverlayValues[111] = d111
		ps138.OverlayValues[135] = d135
		ps138.OverlayValues[136] = d136
		ps138.OverlayValues[137] = d137
		ps138.PhiValues = make([]scm.JITValueDesc, 1)
		d139 = d136
		ps138.PhiValues[0] = d139
		if ps138.General && bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		return bbs[6].RenderPS(ps138)
		return result
	}
	bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d140 := ps.PhiValues[0]
				ctx.EnsureDesc(&d140)
				ctx.EmitStoreToStack(d140, int32(bbs[6].PhiBase)+int32(0))
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
			d135 = ps.OverlayValues[135]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
		}
		if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
			d139 = ps.OverlayValues[139]
		}
		if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
			d140 = ps.OverlayValues[140]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		var d141 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageEnum)(nil).data)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d141 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r15 := ctx.AllocReg()
			r16 := ctx.AllocRegExcept(r15)
			r17 := ctx.AllocRegExcept(r15, r16)
			off := int32(unsafe.Offsetof((*StorageEnum)(nil).data))
			ctx.EmitMovRegMem(r15, thisptr.Reg, off)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off+16)
			d141 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r15, Reg2: r16, Reg3: r17}
			ctx.BindReg(r15, &d141)
			ctx.BindReg(r16, &d141)
			ctx.BindReg(r17, &d141)
			ctx.BindReg(r15, &d141)
			ctx.BindReg(r16, &d141)
			ctx.BindReg(r17, &d141)
		}
		var d142 scm.JITValueDesc
		if d141.SliceSizeKnown {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d141.KnownSliceLen))}
		} else if d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d141.StackOff))}
		} else if d141.Loc == scm.LocStackTriple {
			d142 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d141.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d141)
			if d141.Loc == scm.LocRegPair || d141.Loc == scm.LocRegTriple {
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d141.Reg2, ID: 0}
			} else if d141.Loc == scm.LocReg {
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d141.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d142)
		ctx.EnsureDesc(&d142)
		var d143 scm.JITValueDesc
		if d142.Loc == scm.LocImm {
			d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d142.Reg)
			ctx.EmitMovRegReg(scratch, d142.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d143)
		}
		if d143.Loc == scm.LocReg && d142.Loc == scm.LocReg && d143.Reg == d142.Reg {
			ctx.TransferReg(d142.Reg)
			d142.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d142)
		ctx.EnsureDesc(&d143)
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d143, &d37)
		var d144 scm.JITValueDesc
		if d143.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() - d37.Imm.Int())}
		} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
			r18 := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(r18, d143.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d144)
		} else if d143.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d143.Imm.Int()))
			ctx.EmitSubInt64(scratch, d37.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d144)
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(scratch, d143.Reg)
			if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d37.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d144)
		} else {
			r19 := ctx.AllocRegExcept(d143.Reg, d37.Reg)
			ctx.EmitMovRegReg(r19, d143.Reg)
			ctx.EmitSubInt64(r19, d37.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d144)
		}
		if d144.Loc == scm.LocReg && d143.Loc == scm.LocReg && d144.Reg == d143.Reg {
			ctx.TransferReg(d143.Reg)
			d143.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d143)
		ctx.EnsureDesc(&d144)
		d145 = ctx.EmitLoadScalarSliceElement(&d141, &d144, 8, scm.TagInt)
		ctx.FreeDesc(&d144)
		ctx.StabilizeDescForControlFlow(&d145)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDescsTogether(&idxInt, &d4)
		var d146 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d4.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d4.Imm.Int())}
		} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
			r20 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r20, idxInt.Reg)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d146)
		} else if idxInt.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitSubInt64(scratch, d4.Reg)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d146)
		} else if d4.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d4.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d146)
		} else {
			r21 := ctx.AllocRegExcept(idxInt.Reg, d4.Reg)
			ctx.EmitMovRegReg(r21, idxInt.Reg)
			ctx.EmitSubInt64(r21, d4.Reg)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d146)
		}
		if d146.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d146.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d146)
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d4)
		if ps.General {
			ctx.SyncDesc(&d145)
			if d145.Loc == scm.LocReg {
				ctx.ProtectReg(d145.Reg)
			} else if d145.Loc == scm.LocRegPair {
				ctx.ProtectReg(d145.Reg)
				ctx.ProtectReg(d145.Reg2)
			}
			d147 = d145
			if d147.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d147)
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d147)
			} else {
				ctx.EmitStoreToStack(d147, int32(bbs[7].PhiBase)+int32(0))
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
			} else {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[7].PhiBase)+int32(32))
			}
			if d145.Loc == scm.LocReg {
				ctx.UnprotectReg(d145.Reg)
			} else if d145.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d145.Reg)
				ctx.UnprotectReg(d145.Reg2)
			}
		}
		ps148 := scm.PhiState{General: ps.General}
		ps148.OverlayValues = make([]scm.JITValueDesc, 148)
		ps148.OverlayValues[4] = d4
		ps148.OverlayValues[5] = d5
		ps148.OverlayValues[6] = d6
		ps148.OverlayValues[7] = d7
		ps148.OverlayValues[8] = d8
		ps148.OverlayValues[9] = d9
		ps148.OverlayValues[10] = d10
		ps148.OverlayValues[11] = d11
		ps148.OverlayValues[34] = d34
		ps148.OverlayValues[35] = d35
		ps148.OverlayValues[36] = d36
		ps148.OverlayValues[37] = d37
		ps148.OverlayValues[38] = d38
		ps148.OverlayValues[39] = d39
		ps148.OverlayValues[40] = d40
		ps148.OverlayValues[41] = d41
		ps148.OverlayValues[80] = d80
		ps148.OverlayValues[81] = d81
		ps148.OverlayValues[82] = d82
		ps148.OverlayValues[83] = d83
		ps148.OverlayValues[86] = d86
		ps148.OverlayValues[111] = d111
		ps148.OverlayValues[135] = d135
		ps148.OverlayValues[136] = d136
		ps148.OverlayValues[137] = d137
		ps148.OverlayValues[139] = d139
		ps148.OverlayValues[140] = d140
		ps148.OverlayValues[141] = d141
		ps148.OverlayValues[142] = d142
		ps148.OverlayValues[143] = d143
		ps148.OverlayValues[144] = d144
		ps148.OverlayValues[145] = d145
		ps148.OverlayValues[146] = d146
		ps148.OverlayValues[147] = d147
		ps148.PhiValues = make([]scm.JITValueDesc, 3)
		d149 = d145
		ps148.PhiValues[0] = d149
		d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ps148.PhiValues[1] = d150
		d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps148.PhiValues[2] = d151
		if ps148.General && bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		return bbs[7].RenderPS(ps148)
		return result
	}
	bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d152 := ps.PhiValues[0]
				ctx.EnsureDesc(&d152)
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d152)
				} else {
					ctx.EmitStoreToStack(d152, int32(bbs[7].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d153 := ps.PhiValues[1]
				ctx.EnsureDesc(&d153)
				ctx.EmitStoreScmerToStack(d153, int32(bbs[7].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d154 := ps.PhiValues[2]
				ctx.EnsureDesc(&d154)
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d154)
				} else {
					ctx.EmitStoreToStack(d154, int32(bbs[7].PhiBase)+int32(32))
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
			d135 = ps.OverlayValues[135]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
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
		ctx.EnsureDesc(&d146)
		ctx.EnsureDescsTogether(&d7, &d146)
		var d155 scm.JITValueDesc
		if d7.Loc == scm.LocImm && d146.Loc == scm.LocImm {
			d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d7.Imm.Int() <= d146.Imm.Int())}
		} else if d146.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d7.Reg)
			if d146.Imm.Int() >= -2147483648 && d146.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d7.Reg, int32(d146.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d146.Imm.Int()))
				ctx.EmitCmpInt64(d7.Reg, scm.RegR11)
			}
			d155 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r22, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r22, &d155)
		} else if d7.Loc == scm.LocImm {
			r23 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d146.Reg)
			d155 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r23, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r23, &d155)
		} else {
			r24 := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitCmpInt64(d7.Reg, d146.Reg)
			d155 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r24, Condition: scm.CondSignedLessOrEqual}
			ctx.BindReg(r24, &d155)
		}
		d156 = d155
		ctx.EnsureDesc(&d156)
		if d156.Loc != scm.LocImm && d156.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d156.Loc == scm.LocImm {
			if d156.Imm.Bool() {
				if ps.General {
				}
				ps157 := scm.PhiState{General: ps.General}
				ps157.OverlayValues = make([]scm.JITValueDesc, 157)
				ps157.OverlayValues[4] = d4
				ps157.OverlayValues[5] = d5
				ps157.OverlayValues[6] = d6
				ps157.OverlayValues[7] = d7
				ps157.OverlayValues[8] = d8
				ps157.OverlayValues[9] = d9
				ps157.OverlayValues[10] = d10
				ps157.OverlayValues[11] = d11
				ps157.OverlayValues[34] = d34
				ps157.OverlayValues[35] = d35
				ps157.OverlayValues[36] = d36
				ps157.OverlayValues[37] = d37
				ps157.OverlayValues[38] = d38
				ps157.OverlayValues[39] = d39
				ps157.OverlayValues[40] = d40
				ps157.OverlayValues[41] = d41
				ps157.OverlayValues[80] = d80
				ps157.OverlayValues[81] = d81
				ps157.OverlayValues[82] = d82
				ps157.OverlayValues[83] = d83
				ps157.OverlayValues[86] = d86
				ps157.OverlayValues[111] = d111
				ps157.OverlayValues[135] = d135
				ps157.OverlayValues[136] = d136
				ps157.OverlayValues[137] = d137
				ps157.OverlayValues[139] = d139
				ps157.OverlayValues[140] = d140
				ps157.OverlayValues[141] = d141
				ps157.OverlayValues[142] = d142
				ps157.OverlayValues[143] = d143
				ps157.OverlayValues[144] = d144
				ps157.OverlayValues[145] = d145
				ps157.OverlayValues[146] = d146
				ps157.OverlayValues[147] = d147
				ps157.OverlayValues[149] = d149
				ps157.OverlayValues[150] = d150
				ps157.OverlayValues[151] = d151
				ps157.OverlayValues[152] = d152
				ps157.OverlayValues[153] = d153
				ps157.OverlayValues[154] = d154
				ps157.OverlayValues[155] = d155
				ps157.OverlayValues[156] = d156
				return bbs[8].RenderPS(ps157)
			}
			if ps.General {
			}
			ps158 := scm.PhiState{General: ps.General}
			ps158.OverlayValues = make([]scm.JITValueDesc, 157)
			ps158.OverlayValues[4] = d4
			ps158.OverlayValues[5] = d5
			ps158.OverlayValues[6] = d6
			ps158.OverlayValues[7] = d7
			ps158.OverlayValues[8] = d8
			ps158.OverlayValues[9] = d9
			ps158.OverlayValues[10] = d10
			ps158.OverlayValues[11] = d11
			ps158.OverlayValues[34] = d34
			ps158.OverlayValues[35] = d35
			ps158.OverlayValues[36] = d36
			ps158.OverlayValues[37] = d37
			ps158.OverlayValues[38] = d38
			ps158.OverlayValues[39] = d39
			ps158.OverlayValues[40] = d40
			ps158.OverlayValues[41] = d41
			ps158.OverlayValues[80] = d80
			ps158.OverlayValues[81] = d81
			ps158.OverlayValues[82] = d82
			ps158.OverlayValues[83] = d83
			ps158.OverlayValues[86] = d86
			ps158.OverlayValues[111] = d111
			ps158.OverlayValues[135] = d135
			ps158.OverlayValues[136] = d136
			ps158.OverlayValues[137] = d137
			ps158.OverlayValues[139] = d139
			ps158.OverlayValues[140] = d140
			ps158.OverlayValues[141] = d141
			ps158.OverlayValues[142] = d142
			ps158.OverlayValues[143] = d143
			ps158.OverlayValues[144] = d144
			ps158.OverlayValues[145] = d145
			ps158.OverlayValues[146] = d146
			ps158.OverlayValues[147] = d147
			ps158.OverlayValues[149] = d149
			ps158.OverlayValues[150] = d150
			ps158.OverlayValues[151] = d151
			ps158.OverlayValues[152] = d152
			ps158.OverlayValues[153] = d153
			ps158.OverlayValues[154] = d154
			ps158.OverlayValues[155] = d155
			ps158.OverlayValues[156] = d156
			return bbs[9].RenderPS(ps158)
		}
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
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		ctx.EmitJump(d156.Condition, lbl9)
		ctx.FreeDesc(&d155)
		snap162 := d4
		snap163 := d5
		snap164 := d6
		snap165 := d7
		snap166 := d8
		snap167 := d9
		snap168 := d10
		snap169 := d11
		snap170 := d34
		snap171 := d35
		snap172 := d36
		snap173 := d37
		snap174 := d38
		snap175 := d39
		snap176 := d40
		snap177 := d41
		snap178 := d80
		snap179 := d81
		snap180 := d82
		snap181 := d83
		snap182 := d86
		snap183 := d111
		snap184 := d135
		snap185 := d136
		snap186 := d137
		snap187 := d139
		snap188 := d140
		snap189 := d141
		snap190 := d142
		snap191 := d143
		snap192 := d144
		snap193 := d145
		snap194 := d146
		snap195 := d147
		snap196 := d149
		snap197 := d150
		snap198 := d151
		snap199 := d152
		snap200 := d153
		snap201 := d154
		snap202 := d155
		snap203 := d156
		snap204 := d159
		snap205 := d160
		snap206 := d161
		alloc207 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc207)
		d4 = snap162
		d5 = snap163
		d6 = snap164
		d7 = snap165
		d8 = snap166
		d9 = snap167
		d10 = snap168
		d11 = snap169
		d34 = snap170
		d35 = snap171
		d36 = snap172
		d37 = snap173
		d38 = snap174
		d39 = snap175
		d40 = snap176
		d41 = snap177
		d80 = snap178
		d81 = snap179
		d82 = snap180
		d83 = snap181
		d86 = snap182
		d111 = snap183
		d135 = snap184
		d136 = snap185
		d137 = snap186
		d139 = snap187
		d140 = snap188
		d141 = snap189
		d142 = snap190
		d143 = snap191
		d144 = snap192
		d145 = snap193
		d146 = snap194
		d147 = snap195
		d149 = snap196
		d150 = snap197
		d151 = snap198
		d152 = snap199
		d153 = snap200
		d154 = snap201
		d155 = snap202
		d156 = snap203
		d159 = snap204
		d160 = snap205
		d161 = snap206
		ctx.RestoreAllocState(alloc207)
		d4 = snap162
		d5 = snap163
		d6 = snap164
		d7 = snap165
		d8 = snap166
		d9 = snap167
		d10 = snap168
		d11 = snap169
		d34 = snap170
		d35 = snap171
		d36 = snap172
		d37 = snap173
		d38 = snap174
		d39 = snap175
		d40 = snap176
		d41 = snap177
		d80 = snap178
		d81 = snap179
		d82 = snap180
		d83 = snap181
		d86 = snap182
		d111 = snap183
		d135 = snap184
		d136 = snap185
		d137 = snap186
		d139 = snap187
		d140 = snap188
		d141 = snap189
		d142 = snap190
		d143 = snap191
		d144 = snap192
		d145 = snap193
		d146 = snap194
		d147 = snap195
		d149 = snap196
		d150 = snap197
		d151 = snap198
		d152 = snap199
		d153 = snap200
		d154 = snap201
		d155 = snap202
		d156 = snap203
		d159 = snap204
		d160 = snap205
		d161 = snap206
		ps208 := scm.PhiState{General: true}
		ps208.OverlayValues = make([]scm.JITValueDesc, 162)
		ps208.OverlayValues[4] = d4
		ps208.OverlayValues[5] = d5
		ps208.OverlayValues[6] = d6
		ps208.OverlayValues[7] = d7
		ps208.OverlayValues[8] = d8
		ps208.OverlayValues[9] = d9
		ps208.OverlayValues[10] = d10
		ps208.OverlayValues[11] = d11
		ps208.OverlayValues[34] = d34
		ps208.OverlayValues[35] = d35
		ps208.OverlayValues[36] = d36
		ps208.OverlayValues[37] = d37
		ps208.OverlayValues[38] = d38
		ps208.OverlayValues[39] = d39
		ps208.OverlayValues[40] = d40
		ps208.OverlayValues[41] = d41
		ps208.OverlayValues[80] = d80
		ps208.OverlayValues[81] = d81
		ps208.OverlayValues[82] = d82
		ps208.OverlayValues[83] = d83
		ps208.OverlayValues[86] = d86
		ps208.OverlayValues[111] = d111
		ps208.OverlayValues[135] = d135
		ps208.OverlayValues[136] = d136
		ps208.OverlayValues[137] = d137
		ps208.OverlayValues[139] = d139
		ps208.OverlayValues[140] = d140
		ps208.OverlayValues[141] = d141
		ps208.OverlayValues[142] = d142
		ps208.OverlayValues[143] = d143
		ps208.OverlayValues[144] = d144
		ps208.OverlayValues[145] = d145
		ps208.OverlayValues[146] = d146
		ps208.OverlayValues[147] = d147
		ps208.OverlayValues[149] = d149
		ps208.OverlayValues[150] = d150
		ps208.OverlayValues[151] = d151
		ps208.OverlayValues[152] = d152
		ps208.OverlayValues[153] = d153
		ps208.OverlayValues[154] = d154
		ps208.OverlayValues[155] = d155
		ps208.OverlayValues[156] = d156
		ps208.OverlayValues[159] = d159
		ps208.OverlayValues[160] = d160
		ps208.OverlayValues[161] = d161
		ps209 := scm.PhiState{General: true}
		ps209.OverlayValues = make([]scm.JITValueDesc, 162)
		ps209.OverlayValues[4] = d4
		ps209.OverlayValues[5] = d5
		ps209.OverlayValues[6] = d6
		ps209.OverlayValues[7] = d7
		ps209.OverlayValues[8] = d8
		ps209.OverlayValues[9] = d9
		ps209.OverlayValues[10] = d10
		ps209.OverlayValues[11] = d11
		ps209.OverlayValues[34] = d34
		ps209.OverlayValues[35] = d35
		ps209.OverlayValues[36] = d36
		ps209.OverlayValues[37] = d37
		ps209.OverlayValues[38] = d38
		ps209.OverlayValues[39] = d39
		ps209.OverlayValues[40] = d40
		ps209.OverlayValues[41] = d41
		ps209.OverlayValues[80] = d80
		ps209.OverlayValues[81] = d81
		ps209.OverlayValues[82] = d82
		ps209.OverlayValues[83] = d83
		ps209.OverlayValues[86] = d86
		ps209.OverlayValues[111] = d111
		ps209.OverlayValues[135] = d135
		ps209.OverlayValues[136] = d136
		ps209.OverlayValues[137] = d137
		ps209.OverlayValues[139] = d139
		ps209.OverlayValues[140] = d140
		ps209.OverlayValues[141] = d141
		ps209.OverlayValues[142] = d142
		ps209.OverlayValues[143] = d143
		ps209.OverlayValues[144] = d144
		ps209.OverlayValues[145] = d145
		ps209.OverlayValues[146] = d146
		ps209.OverlayValues[147] = d147
		ps209.OverlayValues[149] = d149
		ps209.OverlayValues[150] = d150
		ps209.OverlayValues[151] = d151
		ps209.OverlayValues[152] = d152
		ps209.OverlayValues[153] = d153
		ps209.OverlayValues[154] = d154
		ps209.OverlayValues[155] = d155
		ps209.OverlayValues[156] = d156
		ps209.OverlayValues[159] = d159
		ps209.OverlayValues[160] = d160
		ps209.OverlayValues[161] = d161
		snap210 := d4
		snap211 := d5
		snap212 := d6
		snap213 := d7
		snap214 := d8
		snap215 := d9
		snap216 := d10
		snap217 := d11
		snap218 := d34
		snap219 := d35
		snap220 := d36
		snap221 := d37
		snap222 := d38
		snap223 := d39
		snap224 := d40
		snap225 := d41
		snap226 := d80
		snap227 := d81
		snap228 := d82
		snap229 := d83
		snap230 := d86
		snap231 := d111
		snap232 := d135
		snap233 := d136
		snap234 := d137
		snap235 := d139
		snap236 := d140
		snap237 := d141
		snap238 := d142
		snap239 := d143
		snap240 := d144
		snap241 := d145
		snap242 := d146
		snap243 := d147
		snap244 := d149
		snap245 := d150
		snap246 := d151
		snap247 := d152
		snap248 := d153
		snap249 := d154
		snap250 := d155
		snap251 := d156
		snap252 := d159
		snap253 := d160
		snap254 := d161
		alloc255 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps209)
		}
		ctx.RestoreAllocState(alloc255)
		d4 = snap210
		d5 = snap211
		d6 = snap212
		d7 = snap213
		d8 = snap214
		d9 = snap215
		d10 = snap216
		d11 = snap217
		d34 = snap218
		d35 = snap219
		d36 = snap220
		d37 = snap221
		d38 = snap222
		d39 = snap223
		d40 = snap224
		d41 = snap225
		d80 = snap226
		d81 = snap227
		d82 = snap228
		d83 = snap229
		d86 = snap230
		d111 = snap231
		d135 = snap232
		d136 = snap233
		d137 = snap234
		d139 = snap235
		d140 = snap236
		d141 = snap237
		d142 = snap238
		d143 = snap239
		d144 = snap240
		d145 = snap241
		d146 = snap242
		d147 = snap243
		d149 = snap244
		d150 = snap245
		d151 = snap246
		d152 = snap247
		d153 = snap248
		d154 = snap249
		d155 = snap250
		d156 = snap251
		d159 = snap252
		d160 = snap253
		d161 = snap254
		if !bbs[8].Rendered {
			return bbs[8].RenderPS(ps208)
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
			d135 = ps.OverlayValues[135]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
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
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&d5)
		d256 = d5
		_ = d256
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl12 := ctx.ReserveLabel()
		_ = lbl12
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
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
		d260 = ctx.EmitLoadScalarSliceElement(&d259, &d258, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d258)
		var d261 scm.JITValueDesc
		r27 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r27, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageEnum)(nil).values)))
		} else {
			ctx.EmitMovRegReg(r27, thisptr.Reg)
			ctx.EmitAddRegImm32(r27, int32(unsafe.Offsetof((*StorageEnum)(nil).values)))
		}
		d261 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27, GoArray: true, RelocatablePointer: true}
		ctx.BindReg(r27, &d261)
		ctx.ReclaimUntrackedRegs()
		d263 = ctx.EmitSliceElementAddress(&d261, &d258, 16)
		ctx.EnsureDesc(&d263)
		r28 := ctx.AllocRegExcept(d263.Reg)
		ctx.EmitMovRegMem(r28, d263.Reg, 8)
		ctx.EmitMovRegMem(d263.Reg, d263.Reg, 0)
		d262 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d263.Reg, Reg2: r28}
		ctx.BindReg(d263.Reg, &d262)
		ctx.BindReg(r28, &d262)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d256)
		var d264 scm.JITValueDesc
		if d256.Loc == scm.LocImm {
			d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) >> 8))}
		} else {
			r29 := ctx.AllocRegExcept(d256.Reg)
			ctx.EmitMovRegReg(r29, d256.Reg)
			ctx.EmitShrRegImm8(r29, 8)
			d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d264)
		}
		if d264.Loc == scm.LocReg && d256.Loc == scm.LocReg && d264.Reg == d256.Reg {
			ctx.TransferReg(d256.Reg)
			d256.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d264)
		ctx.EnsureDesc(&d260)
		ctx.EnsureDescsTogether(&d264, &d260)
		var d265 scm.JITValueDesc
		if d264.Loc == scm.LocImm && d260.Loc == scm.LocImm {
			d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d264.Imm.Int() * d260.Imm.Int())}
		} else if d264.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d260.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d264.Imm.Int()))
			ctx.EmitImulInt64(scratch, d260.Reg)
			d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d265)
		} else if d260.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d264.Reg)
			ctx.EmitMovRegReg(scratch, d264.Reg)
			if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d260.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d260.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d265)
		} else {
			r30 := ctx.AllocRegExcept(d264.Reg, d260.Reg)
			ctx.EmitMovRegReg(r30, d264.Reg)
			ctx.EmitImulInt64(r30, d260.Reg)
			d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d265)
		}
		if d265.Loc == scm.LocReg && d264.Loc == scm.LocReg && d265.Reg == d264.Reg {
			ctx.TransferReg(d264.Reg)
			d264.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d264)
		ctx.FreeDesc(&d260)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d265)
		ctx.EnsureDesc(&d257)
		ctx.EnsureDescsTogether(&d265, &d257)
		var d266 scm.JITValueDesc
		if d265.Loc == scm.LocImm && d257.Loc == scm.LocImm {
			d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d265.Imm.Int() + d257.Imm.Int())}
		} else if d257.Loc == scm.LocImm && d257.Imm.Int() == 0 {
			r31 := ctx.AllocRegExcept(d265.Reg)
			ctx.EmitMovRegReg(r31, d265.Reg)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d266)
		} else if d265.Loc == scm.LocImm && d265.Imm.Int() == 0 {
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d257.Reg}
			ctx.BindReg(d257.Reg, &d266)
		} else if d265.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d257.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d265.Imm.Int()))
			ctx.EmitAddInt64(scratch, d257.Reg)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d266)
		} else if d257.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d265.Reg)
			ctx.EmitMovRegReg(scratch, d265.Reg)
			if d257.Imm.Int() >= -2147483648 && d257.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d257.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d257.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d266)
		} else {
			r32 := ctx.AllocRegExcept(d265.Reg, d257.Reg)
			ctx.EmitMovRegReg(r32, d265.Reg)
			ctx.EmitAddInt64(r32, d257.Reg)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d266)
		}
		if d266.Loc == scm.LocReg && d265.Loc == scm.LocReg && d266.Reg == d265.Reg {
			ctx.TransferReg(d265.Reg)
			d265.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d265)
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
		d267 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageEnum).symbolLo), []scm.JITValueDesc{thisptr, d258}, 1)
		d267.NoHeapPointer = true
		ctx.BindReg(d267.Reg, &d267)
		ctx.FreeDesc(&d258)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d266)
		ctx.EnsureDesc(&d267)
		ctx.EnsureDescsTogether(&d266, &d267)
		var d268 scm.JITValueDesc
		if d266.Loc == scm.LocImm && d267.Loc == scm.LocImm {
			d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d266.Imm.Int() - d267.Imm.Int())}
		} else if d267.Loc == scm.LocImm && d267.Imm.Int() == 0 {
			r33 := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(r33, d266.Reg)
			d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d268)
		} else if d266.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d267.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d266.Imm.Int()))
			ctx.EmitSubInt64(scratch, d267.Reg)
			d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d268)
		} else if d267.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(scratch, d266.Reg)
			if d267.Imm.Int() >= -2147483648 && d267.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d267.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d268)
		} else {
			r34 := ctx.AllocRegExcept(d266.Reg, d267.Reg)
			ctx.EmitMovRegReg(r34, d266.Reg)
			ctx.EmitSubInt64(r34, d267.Reg)
			d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d268)
		}
		if d268.Loc == scm.LocReg && d266.Loc == scm.LocReg && d268.Reg == d266.Reg {
			ctx.TransferReg(d266.Reg)
			d266.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d266)
		ctx.FreeDesc(&d267)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d262)
		ctx.EnsureDesc(&d268)
		ctx.StabilizeDescForControlFlow(&d262)
		ctx.StabilizeDescForControlFlow(&d268)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDesc(&d7)
		var d269 scm.JITValueDesc
		if d7.Loc == scm.LocImm {
			d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + 1)}
		} else {
			var scratch scm.Reg
			if phiHomeOK3 {
				scratch = r1
			} else {
				scratch = ctx.AllocRegExcept(d7.Reg)
			}
			ctx.EmitMovRegReg(scratch, d7.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d269)
		}
		if d269.Loc == scm.LocReg && d7.Loc == scm.LocReg && d269.Reg == d7.Reg {
			ctx.TransferReg(d7.Reg)
			d7.Loc = scm.LocNone
		}
		if ps.General {
			ctx.SyncDesc(&d262)
			if d262.Loc == scm.LocReg {
				ctx.ProtectReg(d262.Reg)
			} else if d262.Loc == scm.LocRegPair {
				ctx.ProtectReg(d262.Reg)
				ctx.ProtectReg(d262.Reg2)
			}
			ctx.SyncDesc(&d268)
			if d268.Loc == scm.LocReg {
				ctx.ProtectReg(d268.Reg)
			} else if d268.Loc == scm.LocRegPair {
				ctx.ProtectReg(d268.Reg)
				ctx.ProtectReg(d268.Reg2)
			}
			ctx.SyncDesc(&d269)
			if d269.Loc == scm.LocReg {
				ctx.ProtectReg(d269.Reg)
			} else if d269.Loc == scm.LocRegPair {
				ctx.ProtectReg(d269.Reg)
				ctx.ProtectReg(d269.Reg2)
			}
			d270 = d268
			if d270.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d270)
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d270)
			} else {
				ctx.EmitStoreToStack(d270, int32(bbs[7].PhiBase)+int32(0))
			}
			d271 = d262
			if d271.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.SyncDesc(&d271)
			if d271.Loc == scm.LocStackPair {
				ctx.EmitCopyStackWords(d271, int32(bbs[7].PhiBase)+int32(16), 2)
			} else if d271.Loc == scm.LocInputPair {
				ctx.EnsureDesc(&d271)
				ctx.EmitStoreScmerToStack(d271, int32(bbs[7].PhiBase)+int32(16))
			} else if d271.Loc == scm.LocRegPair || d271.Loc == scm.LocImm {
				ctx.EmitStoreScmerToStack(d271, int32(bbs[7].PhiBase)+int32(16))
			} else {
				ctx.EnsureDesc(&d271)
				ctx.EmitStoreToStack(d271, int32(bbs[7].PhiBase)+int32(16))
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, (int32(bbs[7].PhiBase)+int32(16))+8)
			}
			d272 = d269
			if d272.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d272)
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d272)
			} else {
				ctx.EmitStoreToStack(d272, int32(bbs[7].PhiBase)+int32(32))
			}
			if d262.Loc == scm.LocReg {
				ctx.UnprotectReg(d262.Reg)
			} else if d262.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d262.Reg)
				ctx.UnprotectReg(d262.Reg2)
			}
			if d268.Loc == scm.LocReg {
				ctx.UnprotectReg(d268.Reg)
			} else if d268.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d268.Reg)
				ctx.UnprotectReg(d268.Reg2)
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
		ps273.OverlayValues[4] = d4
		ps273.OverlayValues[5] = d5
		ps273.OverlayValues[6] = d6
		ps273.OverlayValues[7] = d7
		ps273.OverlayValues[8] = d8
		ps273.OverlayValues[9] = d9
		ps273.OverlayValues[10] = d10
		ps273.OverlayValues[11] = d11
		ps273.OverlayValues[34] = d34
		ps273.OverlayValues[35] = d35
		ps273.OverlayValues[36] = d36
		ps273.OverlayValues[37] = d37
		ps273.OverlayValues[38] = d38
		ps273.OverlayValues[39] = d39
		ps273.OverlayValues[40] = d40
		ps273.OverlayValues[41] = d41
		ps273.OverlayValues[80] = d80
		ps273.OverlayValues[81] = d81
		ps273.OverlayValues[82] = d82
		ps273.OverlayValues[83] = d83
		ps273.OverlayValues[86] = d86
		ps273.OverlayValues[111] = d111
		ps273.OverlayValues[135] = d135
		ps273.OverlayValues[136] = d136
		ps273.OverlayValues[137] = d137
		ps273.OverlayValues[139] = d139
		ps273.OverlayValues[140] = d140
		ps273.OverlayValues[141] = d141
		ps273.OverlayValues[142] = d142
		ps273.OverlayValues[143] = d143
		ps273.OverlayValues[144] = d144
		ps273.OverlayValues[145] = d145
		ps273.OverlayValues[146] = d146
		ps273.OverlayValues[147] = d147
		ps273.OverlayValues[149] = d149
		ps273.OverlayValues[150] = d150
		ps273.OverlayValues[151] = d151
		ps273.OverlayValues[152] = d152
		ps273.OverlayValues[153] = d153
		ps273.OverlayValues[154] = d154
		ps273.OverlayValues[155] = d155
		ps273.OverlayValues[156] = d156
		ps273.OverlayValues[159] = d159
		ps273.OverlayValues[160] = d160
		ps273.OverlayValues[161] = d161
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
		d274 = d268
		ps273.PhiValues[0] = d274
		d275 = d262
		ps273.PhiValues[1] = d275
		d276 = d269
		ps273.PhiValues[2] = d276
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
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
			d86 = ps.OverlayValues[86]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
			d135 = ps.OverlayValues[135]
		}
		if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
			d136 = ps.OverlayValues[136]
		}
		if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
			d137 = ps.OverlayValues[137]
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
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
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
		if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != scm.LocNone {
			d155 = ps.OverlayValues[155]
		}
		if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != scm.LocNone {
			d156 = ps.OverlayValues[156]
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
		if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != scm.LocNone {
			d276 = ps.OverlayValues[276]
		}
		ctx.ReclaimUntrackedRegs()
		d277 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
		ctx.BindReg(r2, &d277)
		ctx.BindReg(r3, &d277)
		ctx.EnsureDesc(&d6)
		if d6.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d6, &d277)
		} else {
			switch d6.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d277, d6)
			case scm.TagInt:
				ctx.EmitMakeInt(d277, d6)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d277, d6)
			case scm.TagNil:
				ctx.EmitMakeNil(d277)
			default:
				ctx.EmitMovPairToResult(&d6, &d277)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps278 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps278)
	ctx.MarkLabel(lbl0)
	d279 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
	ctx.BindReg(r2, &d279)
	ctx.BindReg(r3, &d279)
	ctx.EmitMovPairToResult(&d279, &result)
	ctx.FreeReg(r2)
	ctx.FreeReg(r3)
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
