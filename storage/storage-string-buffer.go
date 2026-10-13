/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "io"
import "time"
import "unsafe"
import "sync/atomic"
import "github.com/pierrec/lz4/v4"
import "bytes"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// canUseBufferHandles is evaluated once before generation publication.
// Special inner encodings and oversized text use the shared decoded cache.
func (s *StorageString) canUseBufferHandles() bool {
	if s.format != FormatRaw {
		return false
	}
	for i := uint32(0); uint64(i) < s.starts.count; i++ {
		if s.nodict && s.starts.hasNull && s.starts.GetValueUInt(i) == s.starts.null {
			continue
		}
		length := int64(s.lens.GetValueUInt(i)) + s.lens.offset
		if length < 0 || length > scm.StringBufferMaxStringBytes {
			return false
		}
	}
	return true
}

//jitgen:noinline
func (s *StorageString) compressedAt(recid uint32) scm.Scmer {
	entry := recid
	if s.nodict {
		if s.starts.hasNull && s.starts.GetValueUInt(recid) == s.starts.null {
			return scm.NewNil()
		}
	} else {
		value := s.values.GetValueUInt(recid)
		if s.values.hasNull && value == s.values.null {
			return scm.NewNil()
		}
		entry = uint32(int64(value) + s.values.offset)
	}
	length := int(int64(s.lens.GetValueUInt(entry)) + s.lens.offset)
	return scm.NewBufferString(s.textBuffer, uint32(int64(s.starts.GetValueUInt(entry))+s.starts.offset), length)
}

// Version 4 preserves the V1 body for flags 0/1 and adds flag 2:
// count, values/starts/lens, offsetCount u32, offsets u32[], buffer payload.
// Buffer payload: codec u8 (2=LZ4), logicalBytes u32, payloadBytes u64,
// payload;
// LZ4: blockCount u32, boundaries[blockCount+1]u32, rawFlags[blockCount]u8.
// LZ4 logical blocks are permanently 65536 bytes for this payload version.

func (s *StorageString) deserializeStringV4(r io.Reader) uint {
	var flag uint8
	if err := binary.Read(r, binary.LittleEndian, &flag); err != nil {
		panic(err)
	}
	if flag < 2 {
		return s.deserializeStringV1(io.MultiReader(bytes.NewReader([]byte{flag}), r))
	}
	if flag != 2 || s.format != FormatRaw {
		panic("invalid compressed string buffer format")
	}
	read := func(v any) {
		if err := binary.Read(r, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	var count uint64
	read(&count)
	s.values.DeserializeEx(r, true)
	s.count = s.starts.DeserializeEx(r, true)
	s.lens.DeserializeEx(r, true)
	var offsets uint32
	read(&offsets)
	if uint64(offsets) != s.starts.count {
		panic("invalid compressed string entry count")
	}
	legacyOffsets := make([]uint32, offsets)
	read(legacyOffsets)
	s.textBuffer = scm.ReadCompressedStringBuffer(r)
	// Validate every extent while unpublished. Malformed data must fail before
	// it can become a SQL-visible handle. V4 offsets are validated then discarded.
	var scratch scm.TextScratch
	var cursor scm.TextCursor
	var chunk [1024]byte
	for i := uint32(0); uint64(i) < s.starts.count; i++ {
		if s.nodict && s.starts.hasNull && s.starts.GetValueUInt(i) == s.starts.null {
			continue
		}
		length := int(int64(s.lens.GetValueUInt(i)) + s.lens.offset)
		start := uint32(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
		if legacyOffsets[i] != start {
			panic("invalid compressed string dictionary offset")
		}
		value := scm.NewBufferString(s.textBuffer, start, length)
		cursor.Reset(value)
		for cursor.Read(chunk[:], &scratch) > 0 {
		}
	}
	s.bufferHandles = true
	return uint(count)
}

// V5: count, values/starts/lens, block payload. No compression flag and no
// duplicate offset directory. StringFormat still describes the inner encoding.
func (s *StorageString) serializeStringBufferV5(w io.Writer, buffer *scm.CompressedStringBuffer) {
	write := func(v any) {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(uint8(20))
	var nodict uint8
	if s.nodict {
		nodict = 1
	}
	write(nodict)
	write(uint8(s.format))
	write(uint8(storageStringVersion))
	write([4]byte{})
	count := s.values.count
	if s.nodict {
		count = s.starts.count
	}
	write(count)
	s.values.Serialize(w)
	s.starts.Serialize(w)
	s.lens.Serialize(w)
	buffer.WriteTo(w)
}

func (s *StorageString) deserializeStringV5(r io.Reader) uint {
	if s.format > FormatDateTimeOffset {
		panic("invalid compressed string inner format")
	}
	var count uint64
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		panic(err)
	}
	s.values.DeserializeEx(r, true)
	s.count = s.starts.DeserializeEx(r, true)
	s.lens.DeserializeEx(r, true)
	if s.lens.count != s.starts.count || s.nodict && count != s.starts.count || !s.nodict && count != s.values.count {
		panic("invalid compressed string entry count")
	}
	s.textBuffer = scm.ReadCompressedStringBuffer(r)
	// Check all block bytes with bounded scratch, never a full decoded cache.
	var scratch scm.TextScratch
	var cursor scm.TextCursor
	var chunk [1024]byte
	size := s.textBuffer.DecodedSize()
	for offset := 0; offset < size; offset += scm.StringBufferBlockBytes {
		cursor.Reset(scm.NewBufferString(s.textBuffer, uint32(offset), min(scm.StringBufferBlockBytes, size-offset)))
		for cursor.Read(chunk[:], &scratch) > 0 {
		}
	}
	for i := uint32(0); uint64(i) < s.starts.count; i++ {
		start := s.starts.GetValueUInt(i)
		if s.nodict && s.starts.hasNull && start == s.starts.null {
			continue
		}
		position := int64(start) + s.starts.offset
		length := int64(s.lens.GetValueUInt(i)) + s.lens.offset
		if position < 0 || length < 0 {
			panic("invalid compressed string extent")
		}
		end := position + length
		if nibbleCharsetFor(s.format) != nil {
			end = (end + 1) / 2
		} else if s.format == FormatUUIDLower || s.format == FormatUUIDUpper {
			end = position + 16
		}
		if end > int64(size) {
			panic("invalid compressed string extent")
		}
	}
	if !s.nodict {
		for i := uint32(0); uint64(i) < s.values.count; i++ {
			entry := s.values.GetValueUInt(i)
			if s.values.hasNull && entry == s.values.null {
				continue
			}
			index := int64(entry) + s.values.offset
			if index < 0 || uint64(index) >= s.starts.count {
				panic("invalid compressed string dictionary entry")
			}
		}
	}
	s.bufferHandles = s.canUseBufferHandles()
	return uint(count)
}

// stringDictionaryCache is one immutable decoded payload. StorageString.dictMu
// owns its lease count and admission/retirement; lastUsed changes once per
// preparation. The atomic directory never exposes a partially decoded buffer.
// CacheManager owns the exclusive registration, separate from the column's
// immutable compressed payload. Neither row reads nor comparisons change it.
type stringDictionaryCache struct {
	storage  *StorageString
	text     string
	users    int
	lastUsed atomic.Int64
}

func (c *stringDictionaryCache) computeSize() uint {
	return uint(unsafe.Sizeof(*c)) + uint(len(c.text))
}

func (s *StorageString) acquireDictionary(create bool) *stringDictionaryCache {
	if !create && s.dictCache.Load() == nil {
		return nil
	}
	if create {
		s.dictMu.Lock()
	} else if !s.dictMu.TryLock() {
		// Cache reuse is optional. A point probe must not wait for another
		// operator's cache preparation or eviction; use compressed handles.
		return nil
	}
	defer s.dictMu.Unlock()
	cache := s.dictCache.Load()
	if cache == nil {
		if !create {
			return nil
		}
		cache = &stringDictionaryCache{storage: s, text: s.decodeDictionary()}
		cache.users = 1
		cache.lastUsed.Store(time.Now().UnixNano())
		s.dictCache.Store(cache)
		GlobalCache.AddItem(cache, int64(cache.computeSize()), TypeStringDict, stringDictCleanup, stringDictLastUsed, nil)
		return cache
	}
	cache.users++
	cache.lastUsed.Store(time.Now().UnixNano())
	return cache
}

func (s *StorageString) releaseDictionary(cache *stringDictionaryCache) {
	s.dictMu.Lock()
	defer s.dictMu.Unlock()
	cache.users--
	if cache.users < 0 {
		panic("string dictionary lease released twice")
	}
}

// decodeDictionary is the only full materialization path. Legacy monolithic
// LZ4 files and new compressed handles share admission, accounting and eviction.
func (s *StorageString) decodeDictionary() string {
	if s.textBuffer != nil {
		dst := make([]byte, s.textBuffer.DecodedSize())
		s.textBuffer.DecodeInto(dst)
		return unsafe.String(unsafe.SliceData(dst), len(dst))
	}
	if !s.compressed {
		return s.dictionary
	}
	if len(s.compressedDict) < 4 {
		panic("invalid legacy LZ4 dictionary")
	}
	length := int(binary.LittleEndian.Uint32(s.compressedDict[:4]))
	dst := make([]byte, length)
	n, err := lz4.UncompressBlock(s.compressedDict[4:], dst)
	if err != nil || n != length {
		panic("invalid legacy LZ4 dictionary")
	}
	return unsafe.String(unsafe.SliceData(dst), len(dst))
}

type decodedStringReader struct {
	storage *StorageString
	cache   *stringDictionaryCache
}

// Reader preparation can change the physical tag without changing the column
// type. Typed JIT filters must specialize against this prepared representation.
func (r *decodedStringReader) JITValueType() uint8 {
	if r.storage.stringHasNull() {
		return scm.JITTypeUnknown
	}
	if r.storage.format == FormatRaw {
		return scm.TagString
	}
	return r.storage.JITValueType()
}

func (r *decodedStringReader) release() {
	if r.cache != nil {
		r.storage.releaseDictionary(r.cache)
		r.cache = nil
	}
}

func (r *decodedStringReader) GetValue(row uint32) scm.Scmer {
	text := r.cache.text
	return r.storage.decodeAt(row, text, unsafe.Pointer(unsafe.StringData(text)))
}

func (r *decodedStringReader) GetValueMulti(rows []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	atomic.AddUint64(&r.storage.readCount, uint64(len(rows)))
	text := r.cache.text
	base := unsafe.Pointer(unsafe.StringData(text))
	for i, row := range rows {
		target[i*stride] = r.storage.decodeAt(row, text, base)
	}
}

func (r *decodedStringReader) GetValueRange(row, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	atomic.AddUint64(&r.storage.readCount, uint64(count))
	text := r.cache.text
	base := unsafe.Pointer(unsafe.StringData(text))
	for i := uint32(0); i < count; i++ {
		target[int(i)*stride] = r.storage.decodeAt(row+i, text, base)
	}
}
