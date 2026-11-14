// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import "io"
import "bytes"
import "unsafe"
import "encoding/binary"
import "github.com/pierrec/lz4/v4"
import "github.com/launix-de/memcp/scm"

//jitgen:noinline
func (s *StorageString) compressedAt(recid uint32) scm.Scmer {
	entry := recid
	if s.nodict {
		if s.starts.hasNull && s.starts.GetValueUInt(recid) == s.starts.null {
			return scm.NewNil()
		}
	} else {
		v := s.values.GetValueUInt(recid)
		if s.values.hasNull && v == s.values.null {
			return scm.NewNil()
		}
		entry = uint32(int64(v) + s.values.offset)
	}
	return scm.NewBufferString(s.textBuffer, uint64(int64(s.starts.GetValueUInt(entry))+s.starts.offset))
}

// readLegacyStringBuffer is the only block-LZ4 path. It runs during private
// deserialization, and its decoded bytes are converted before publication.
func readLegacyStringBuffer(r io.Reader) string {
	read := func(v any) {
		if err := binary.Read(r, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	var codec uint8
	var size uint32
	var length uint64
	read(&codec)
	read(&size)
	read(&length)
	if codec != 2 || length > uint64(^uint32(0)) {
		panic("invalid legacy LZ4 buffer")
	}
	encoded := make([]byte, int(length))
	if _, err := io.ReadFull(r, encoded); err != nil {
		panic(err)
	}
	var count uint32
	read(&count)
	if uint64(count) != (uint64(size)+65535)/65536 {
		panic("invalid legacy LZ4 block count")
	}
	boundaries := make([]uint32, int(count)+1)
	read(boundaries)
	if boundaries[0] != 0 || uint64(boundaries[count]) != length {
		panic("invalid legacy LZ4 extent")
	}
	flags := make([]byte, int(count))
	if _, err := io.ReadFull(r, flags); err != nil {
		panic(err)
	}
	raw := make([]byte, int(size))
	for i := 0; i < int(count); i++ {
		if boundaries[i] > boundaries[i+1] || uint64(boundaries[i+1]) > length || flags[i] > 1 {
			panic("invalid legacy LZ4 block")
		}
		src := encoded[boundaries[i]:boundaries[i+1]]
		dst := raw[i*65536 : min((i+1)*65536, len(raw))]
		if flags[i] == 1 {
			if len(src) != len(dst) {
				panic("invalid legacy literal block")
			}
			copy(dst, src)
		} else {
			n, err := lz4.UncompressBlock(src, dst)
			if err != nil || n != len(dst) {
				panic("invalid legacy LZ4 block")
			}
		}
	}
	return unsafe.String(unsafe.SliceData(raw), len(raw))
}

func (s *StorageString) deserializeStringV4(r io.Reader) uint {
	var flag uint8
	if err := binary.Read(r, binary.LittleEndian, &flag); err != nil {
		panic(err)
	}
	if flag < 2 {
		return s.deserializeStringV1(io.MultiReader(bytes.NewReader([]byte{flag}), r))
	}
	if flag != 2 || s.format != FormatRaw {
		panic("invalid legacy string flag")
	}
	var count uint64
	mustReadString(r, &count)
	s.values.DeserializeEx(r, true)
	s.count = s.starts.DeserializeEx(r, true)
	s.lens.DeserializeEx(r, true)
	var n uint32
	mustReadString(r, &n)
	if uint64(n) != s.starts.count {
		panic("invalid legacy offset count")
	}
	offsets := make([]uint32, n)
	mustReadString(r, offsets)
	for i, offset := range offsets {
		if int64(offset) != int64(s.starts.GetValueUInt(uint32(i)))+s.starts.offset {
			panic("invalid legacy offset")
		}
	}
	s.dictionary = readLegacyStringBuffer(r)
	return uint(count)
}
func mustReadString(r io.Reader, v any) {
	if err := binary.Read(r, binary.LittleEndian, v); err != nil {
		panic(err)
	}
}
func (s *StorageString) deserializeStringV5(r io.Reader) uint {
	var count uint64
	mustReadString(r, &count)
	s.values.DeserializeEx(r, true)
	s.count = s.starts.DeserializeEx(r, true)
	s.lens.DeserializeEx(r, true)
	if s.lens.count != s.starts.count || s.nodict && count != s.starts.count || !s.nodict && count != s.values.count {
		panic("invalid legacy string counts")
	}
	s.dictionary = readLegacyStringBuffer(r)
	return uint(count)
}

// V6: count, values/starts, mode u8. Mode 0 stores lengths and raw inner-encoded
// bytes [length u64, bytes]. Mode 1 stores a concat pool [length u64, bytes];
// starts then contains 56-bit pool entry offsets, not logical dictionary offsets.
func (s *StorageString) serializeStringV6(w io.Writer) {
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
	if s.textBuffer != nil {
		write(uint8(1))
		s.textBuffer.WriteTo(w)
	} else {
		write(uint8(0))
		s.lens.Serialize(w)
		write(uint64(len(s.dictionary)))
		if _, err := io.WriteString(w, s.dictionary); err != nil {
			panic(err)
		}
	}
}
func (s *StorageString) deserializeStringV6(r io.Reader) uint {
	var count uint64
	mustReadString(r, &count)
	s.values.DeserializeEx(r, true)
	s.count = s.starts.DeserializeEx(r, true)
	if s.nodict && count != s.starts.count || !s.nodict && count != s.values.count {
		panic("invalid concat string counts")
	}
	var mode uint8
	mustReadString(r, &mode)
	switch mode {
	case 0:
		s.lens.DeserializeEx(r, true)
		if s.lens.count != s.starts.count {
			panic("invalid raw string lengths")
		}
		var n uint64
		mustReadString(r, &n)
		if n > uint64(int(^uint(0)>>1)) {
			panic("invalid raw string size")
		}
		data := make([]byte, int(n))
		if _, err := io.ReadFull(r, data); err != nil {
			panic(err)
		}
		s.dictionary = unsafe.String(unsafe.SliceData(data), len(data))
	case 1:
		if s.format != FormatRaw {
			panic("concat pool requires raw inner encoding")
		}
		s.textBuffer = scm.ReadStringConcatBuffer(r)
		for i := uint32(0); uint64(i) < s.starts.count; i++ {
			if s.nodict && s.starts.hasNull && s.starts.GetValueUInt(i) == s.starts.null {
				continue
			}
			offset := uint64(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
			s.textBuffer.Validate(offset)
		}
		s.bufferHandles = true
	default:
		panic("invalid concat string mode")
	}
	if !s.nodict {
		for i := uint32(0); uint64(i) < s.values.count; i++ {
			v := s.values.GetValueUInt(i)
			if s.values.hasNull && v == s.values.null {
				continue
			}
			entry := int64(v) + s.values.offset
			if entry < 0 || uint64(entry) >= s.starts.count {
				panic("invalid string dictionary index")
			}
		}
	}
	return uint(count)
}
