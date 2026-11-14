// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import "unsafe"
import "github.com/launix-de/memcp/scm"
import "encoding/binary"
import "github.com/pierrec/lz4/v4"
import "github.com/launix-de/memcp/storage/stringconcat"

// StringConcatCompression is shared with the Scheme runtime through the
// dependency-free codec package. Strlen, Extract and Append borrow the pool.
type StringConcatCompression = stringconcat.Buffer

func (s *StorageString) normalizeLegacyDictionary() {
	if !s.compressed {
		return
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
	s.dictionary = unsafe.String(unsafe.SliceData(dst), len(dst))
	s.compressedDict = nil
	s.compressed = false
}

// CompressDictionary runs only on a private generation. New raw text receives
// concat entries. Compact nibble/UUID/Base64 inner encodings stay raw and need
// neither materialized dictionaries nor a decoded-cache registration.
func (s *StorageString) CompressDictionary() bool {
	if s.textBuffer != nil {
		return false
	}
	s.normalizeLegacyDictionary()
	if s.format != FormatRaw {
		return false
	}
	builder := stringconcat.NewBuilder()
	offsets := make([]uint64, s.starts.count)
	nulls := make([]bool, s.starts.count)
	for i := uint32(0); uint64(i) < s.starts.count; i++ {
		if s.nodict && s.starts.hasNull && s.starts.GetValueUInt(i) == s.starts.null {
			nulls[i] = true
			continue
		}
		start := int64(s.starts.GetValueUInt(i)) + s.starts.offset
		length := int64(s.lens.GetValueUInt(i)) + s.lens.offset
		if start < 0 || length < 0 || start > int64(len(s.dictionary)) || length > int64(len(s.dictionary))-start {
			panic("invalid legacy string extent")
		}
		offsets[i] = builder.Add(s.dictionary[start : start+length])
	}
	s.starts = concatStarts(offsets, nulls)
	s.textBuffer = builder.Finish()
	s.dictionary = ""
	s.lens = StorageInt{}
	s.bufferHandles = true
	return true
}
func concatStarts(offsets []uint64, nulls []bool) StorageInt {
	var result StorageInt
	result.prepare()
	for i, v := range offsets {
		if nulls != nil && nulls[i] {
			result.scan(uint32(i), scm.NewNil())
		} else {
			result.scan(uint32(i), scm.NewInt(int64(v)))
		}
	}
	result.init(uint32(len(offsets)))
	for i, v := range offsets {
		if nulls != nil && nulls[i] {
			result.build(uint32(i), scm.NewNil())
		} else {
			result.build(uint32(i), scm.NewInt(int64(v)))
		}
	}
	result.finish()
	return result
}
