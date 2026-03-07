/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "io"
import "fmt"
import "unsafe"
import "strings"
import "encoding/hex"
import "encoding/base64"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// StringFormat describes how the string bytes in the dictionary are encoded.
// The lowest bit encodes case for formats that have a case variant:
//
//	bit 0 == 0  →  lowercase  (or no-case)
//	bit 0 == 1  →  uppercase  (or no-case variant at odd positions)
type StringFormat uint8

const (
	FormatRaw         StringFormat = 0  // 8 bit/char, no compression
	FormatPhone       StringFormat = 1  // [0-9 +\-/()] 4 bit/char — stored phone numbers with space/slash
	FormatHexLower    StringFormat = 2  // [0-9a-f] packed nibbles, 4 bit/char
	FormatHexUpper    StringFormat = 3  // [0-9A-F] packed nibbles, 4 bit/char
	FormatBase64Lower StringFormat = 4  // URL-safe base64, stored as decoded binary
	FormatBase64Upper StringFormat = 5  // standard base64, stored as decoded binary
	FormatUUIDLower   StringFormat = 6  // xxxxxxxx-…-xxxx lowercase, stored as 16 bytes
	FormatUUIDUpper   StringFormat = 7  // XXXXXXXX-…-XXXX uppercase, stored as 16 bytes
	FormatDecimal     StringFormat = 8  // [0-9+\-.,eE] 4 bit/char — DE+EN numbers, IPv4, scientific
	FormatDateTime    StringFormat = 9  // [0-9\-:. T] 4 bit/char — ISO 8601 dates and times
	FormatPhoneDTMF   StringFormat = 10 // [0-9+\-()#*] 4 bit/char — DTMF dialing sequences
)

// nibbleCharset holds the encode (index→char) and decode (char→index) tables
// for 4-bit nibble-packed string formats.
type nibbleCharset struct {
	enc [16]byte
	dec [256]int8
}

func makeNibbleCharset(chars []byte) nibbleCharset {
	var cs nibbleCharset
	for i := range cs.dec {
		cs.dec[i] = -1
	}
	for i, c := range chars {
		cs.enc[i] = c
		cs.dec[c] = int8(i)
	}
	return cs
}

var (
	// hexLowerCharset: lowercase hex digits [0-9a-f]
	hexLowerCharset = makeNibbleCharset([]byte("0123456789abcdef"))
	// hexUpperCharset: uppercase hex digits [0-9A-F]
	hexUpperCharset = makeNibbleCharset([]byte("0123456789ABCDEF"))
	// phoneCharset: space and slash for formatted stored numbers like "+49 30 123-45/6"
	phoneCharset = makeNibbleCharset([]byte("0123456789 +-/()"))
	// phoneDTMFCharset: hash and star for dialing sequences like "*100#" or "+49123*"
	phoneDTMFCharset = makeNibbleCharset([]byte("0123456789+-()#*"))
	// decimalCharset: both DE (comma) and EN (dot) separators + scientific notation
	decimalCharset = makeNibbleCharset([]byte("0123456789+-.,eE"))
	// dateTimeCharset: ISO 8601 (15 chars, index 15 unused)
	dateTimeCharset = makeNibbleCharset([]byte("0123456789-:. T"))
)

// allFormatsValid is the initial bitmask with all format bits set.
const allFormatsValid uint16 = (1 << 11) - 1 // bits 0..10

// checkFormatBits returns which StringFormat bits remain compatible with s.
// bit i is set iff StringFormat(i) is still a valid encoding for s.
// FormatRaw (bit 0) is always set.
func checkFormatBits(s string) uint16 {
	if len(s) == 0 {
		return allFormatsValid // empty string is compatible with everything
	}
	valid := allFormatsValid

	// Base64 requires length divisible by 4
	if len(s)%4 != 0 {
		valid &^= (1 << FormatBase64Upper) | (1 << FormatBase64Lower)
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		// mask out formats that don't accept c; use nibble dec tables directly
		if hexLowerCharset.dec[c] < 0 {
			valid &^= 1 << FormatHexLower
		}
		if hexUpperCharset.dec[c] < 0 {
			valid &^= 1 << FormatHexUpper
		}
		if phoneCharset.dec[c] < 0 {
			valid &^= 1 << FormatPhone
		}
		if phoneDTMFCharset.dec[c] < 0 {
			valid &^= 1 << FormatPhoneDTMF
		}
		if decimalCharset.dec[c] < 0 {
			valid &^= 1 << FormatDecimal
		}
		if dateTimeCharset.dec[c] < 0 {
			valid &^= 1 << FormatDateTime
		}
		isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isDigit := c >= '0' && c <= '9'
		if !isAlpha && !isDigit && c != '+' && c != '/' && c != '=' {
			valid &^= 1 << FormatBase64Upper
		}
		if !isAlpha && !isDigit && c != '-' && c != '_' && c != '=' {
			valid &^= 1 << FormatBase64Lower
		}
		// early exit once only FormatRaw remains
		if valid == 1 {
			return valid
		}
	}

	isUUIDLow, isUUIDUp := checkUUID(s)
	if !isUUIDLow {
		valid &^= 1 << FormatUUIDLower
	}
	if !isUUIDUp {
		valid &^= 1 << FormatUUIDUpper
	}
	return valid
}

// checkUUID returns whether s is a valid UUID in lowercase / uppercase.
func checkUUID(s string) (lower, upper bool) {
	if len(s) != 36 {
		return false, false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false, false
	}
	lower = true
	upper = true
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		c := s[i]
		if c >= '0' && c <= '9' {
			continue
		}
		if c >= 'a' && c <= 'f' {
			upper = false
			continue
		}
		if c >= 'A' && c <= 'F' {
			lower = false
			continue
		}
		return false, false
	}
	return
}

// chooseBestFormat picks the most space-efficient format from validFormats.
func chooseBestFormat(valid uint16) StringFormat {
	// UUID: 16 bytes vs 36 chars → ~56% savings (best for fixed-length UUID columns)
	if valid&(1<<FormatUUIDLower) != 0 {
		return FormatUUIDLower
	}
	if valid&(1<<FormatUUIDUpper) != 0 {
		return FormatUUIDUpper
	}
	// Hex/nibble sets: 50% savings
	if valid&(1<<FormatHexLower) != 0 {
		return FormatHexLower
	}
	if valid&(1<<FormatHexUpper) != 0 {
		return FormatHexUpper
	}
	if valid&(1<<FormatPhone) != 0 {
		return FormatPhone
	}
	if valid&(1<<FormatPhoneDTMF) != 0 {
		return FormatPhoneDTMF
	}
	if valid&(1<<FormatDecimal) != 0 {
		return FormatDecimal
	}
	if valid&(1<<FormatDateTime) != 0 {
		return FormatDateTime
	}
	// Base64: ~25% savings
	if valid&(1<<FormatBase64Upper) != 0 {
		return FormatBase64Upper
	}
	if valid&(1<<FormatBase64Lower) != 0 {
		return FormatBase64Lower
	}
	return FormatRaw
}

// packNibbles encodes s into 4-bit nibbles using cs and appends to dst.
func packNibbles(dst []byte, s string, cs *nibbleCharset) []byte {
	for i := 0; i < len(s); i += 2 {
		lo := cs.dec[s[i]]
		hi := int8(0)
		if i+1 < len(s) {
			hi = cs.dec[s[i+1]]
		}
		dst = append(dst, byte(lo)|byte(hi)<<4)
	}
	return dst
}

// compressString encodes s into the chosen format and appends bytes to dst.
func compressString(dst []byte, s string, format StringFormat) []byte {
	switch format {
	case FormatHexLower:
		return packNibbles(dst, s, &hexLowerCharset)
	case FormatHexUpper:
		return packNibbles(dst, s, &hexUpperCharset)
	case FormatPhone:
		return packNibbles(dst, s, &phoneCharset)
	case FormatPhoneDTMF:
		return packNibbles(dst, s, &phoneDTMFCharset)
	case FormatDecimal:
		return packNibbles(dst, s, &decimalCharset)
	case FormatDateTime:
		return packNibbles(dst, s, &dateTimeCharset)
	case FormatBase64Upper:
		b, _ := base64.StdEncoding.DecodeString(s)
		return append(dst, b...)
	case FormatBase64Lower:
		b, _ := base64.URLEncoding.DecodeString(s)
		return append(dst, b...)
	case FormatUUIDLower, FormatUUIDUpper:
		hexStr := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
		b, _ := hex.DecodeString(hexStr)
		return append(dst, b...)
	default: // FormatRaw
		return append(dst, s...)
	}
}

// compressedByteLen returns how many bytes compressString produces for a string of charLen chars.
func compressedByteLen(charLen int, format StringFormat) int {
	switch format {
	case FormatHexLower, FormatHexUpper,
		FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
		return (charLen + 1) / 2
	case FormatBase64Upper, FormatBase64Lower:
		// base64.DecodedLen is an upper bound (padding bytes don't produce output).
		// The struct stores the exact byte count in lens, so this helper is only
		// used to pre-size buffers during the build phase.
		return base64.StdEncoding.DecodedLen(charLen)
	case FormatUUIDLower, FormatUUIDUpper:
		return 16
	default:
		return charLen
	}
}

// adjustLensForFormat corrects StorageInt scan statistics (offset/max) to match
// the values that build() will actually write, so that init() allocates the right
// bitsize.  Must be called BEFORE lens.init().
func adjustLensForFormat(lens *StorageInt, format StringFormat) {
	switch format {
	case FormatBase64Upper, FormatBase64Lower:
		// build() stores decoded byte count; use 0 as safe lower bound to avoid
		// negative relative values from padding variability
		lens.offset = 0
		lens.max = int64(base64.StdEncoding.DecodedLen(int(lens.max)))
	// All nibble formats (Hex, Phone, PhoneDTMF, Decimal, DateTime): char count == raw len → no adjustment
	// UUID: lens is never read → no adjustment
	// Raw: byte count == char count → no adjustment
	}
}

// unpackNibbles decodes 4-bit nibble-packed bytes back to a string using cs.
// charLen is the original character count (may be odd; last nibble of last byte ignored if so).
func unpackNibbles(b []byte, charLen int, cs *nibbleCharset) string {
	result := make([]byte, charLen)
	for i := 0; i < charLen; i++ {
		packed := b[i/2]
		if i%2 == 0 {
			result[i] = cs.enc[packed&0x0F]
		} else {
			result[i] = cs.enc[(packed>>4)&0x0F]
		}
	}
	return unsafe.String(&result[0], charLen)
}

// decompressBytes decodes b back to a string using the given format.
// charLen is the original character count (needed for nibble formats with odd length).
func decompressBytes(b []byte, format StringFormat, charLen int) string {
	switch format {
	case FormatHexLower:
		return unpackNibbles(b, charLen, &hexLowerCharset)
	case FormatHexUpper:
		return unpackNibbles(b, charLen, &hexUpperCharset)
	case FormatPhone:
		return unpackNibbles(b, charLen, &phoneCharset)
	case FormatPhoneDTMF:
		return unpackNibbles(b, charLen, &phoneDTMFCharset)
	case FormatDecimal:
		return unpackNibbles(b, charLen, &decimalCharset)
	case FormatDateTime:
		return unpackNibbles(b, charLen, &dateTimeCharset)
	case FormatBase64Upper:
		return base64.StdEncoding.EncodeToString(b)
	case FormatBase64Lower:
		return base64.URLEncoding.EncodeToString(b)
	case FormatUUIDLower:
		h := hex.EncodeToString(b)
		return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
	case FormatUUIDUpper:
		h := strings.ToUpper(hex.EncodeToString(b))
		return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
	default: // FormatRaw
		if len(b) == 0 {
			return ""
		}
		return unsafe.String(&b[0], len(b))
	}
}

type StorageString struct {
	// StorageInt for dictionary entries
	values StorageInt
	// the dictionary: compressed bytes for all values concatenated
	dictionary string
	starts     StorageInt // byte offsets into dictionary
	lens       StorageInt // compressed byte lengths
	nodict     bool       // disable values array
	format     StringFormat

	// helpers (scan/build phase only)
	sb           strings.Builder
	compBuf      []byte // accumulates compressed bytes during build (nodict) or init (dict)
	reverseMap   map[string][3]uint
	count        uint
	allsize      int
	validFormats uint16
	// prefix statistics
	prefixstat map[string]int
	laststr    string
}

func (s *StorageString) ComputeSize() uint {
	return s.values.ComputeSize() + 8 + uint(len(s.dictionary)) + 24 + s.starts.ComputeSize() + s.lens.ComputeSize() + 8*8
}

func (s *StorageString) String() string {
	if s.nodict {
		return fmt.Sprintf("string-buffer[%d bytes, format=%d]", len(s.dictionary), s.format)
	} else {
		return fmt.Sprintf("string-dict[%d entries; %d bytes, format=%d]", s.count, len(s.dictionary), s.format)
	}
}

func (s *StorageString) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(20)) // 20 = StorageString
	var nodict uint8 = 0
	if s.nodict {
		nodict = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(nodict))
	// formerly 6 dummy bytes; first byte now carries the format
	binary.Write(f, binary.LittleEndian, uint8(s.format))
	var pad [5]byte
	f.Write(pad[:])
	if s.nodict {
		binary.Write(f, binary.LittleEndian, uint64(s.starts.count))
	} else {
		binary.Write(f, binary.LittleEndian, uint64(s.values.count))
	}
	s.values.Serialize(f)
	s.starts.Serialize(f)
	s.lens.Serialize(f)
	binary.Write(f, binary.LittleEndian, uint64(len(s.dictionary)))
	io.WriteString(f, s.dictionary)
}

func (s *StorageString) Deserialize(f io.Reader) uint {
	var nodict uint8
	binary.Read(f, binary.LittleEndian, &nodict)
	if nodict == 1 {
		s.nodict = true
	}
	var formatByte uint8
	binary.Read(f, binary.LittleEndian, &formatByte)
	s.format = StringFormat(formatByte)
	var pad [5]byte
	f.Read(pad[:])
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values.DeserializeEx(f, true)
	s.count = s.starts.DeserializeEx(f, true)
	s.lens.DeserializeEx(f, true)
	var dictionarylength uint64
	binary.Read(f, binary.LittleEndian, &dictionarylength)
	if dictionarylength > 0 {
		rawdata := make([]byte, dictionarylength)
		f.Read(rawdata)
		s.dictionary = unsafe.String(&rawdata[0], dictionarylength)
	}
	return uint(l)
}

func (s *StorageString) GetCachedReader() ColumnReader { return s }

func (s *StorageString) GetValue(i uint32) scm.Scmer {
	var byteStart, lensVal uint64
	if s.nodict {
		bs := uint64(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
		if s.starts.hasNull && bs == uint64(s.starts.null) {
			return scm.NewNil()
		}
		byteStart = bs
		lensVal = uint64(int64(s.lens.GetValueUInt(i)) + s.lens.offset)
	} else {
		idx := uint32(int64(s.values.GetValueUInt(i)) + s.values.offset)
		if s.values.hasNull && idx == uint32(s.values.null) {
			return scm.NewNil()
		}
		byteStart = uint64(int64(s.starts.GetValueUInt(idx)) + s.starts.offset)
		lensVal = uint64(int64(s.lens.GetValueUInt(idx)) + s.lens.offset)
	}

	// lensVal semantics depend on format:
	//   FormatRaw, FormatHex*, FormatBase64*: byte count
	//   FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime: char count
	//   FormatUUID*: lens unused; always 16 bytes
	switch s.format {
	case FormatRaw:
		// zero-alloc: slice the dictionary directly
		return scm.NewString(s.dictionary[byteStart : byteStart+lensVal])
	case FormatHexLower, FormatHexUpper,
		FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
		// lensVal = original char count; byte count = ceil(charLen/2)
		charLen := int(lensVal)
		byteLen := uint64((charLen + 1) / 2)
		b := []byte(s.dictionary[byteStart : byteStart+byteLen])
		return scm.NewString(decompressBytes(b, s.format, charLen))
	case FormatUUIDLower, FormatUUIDUpper:
		// lens unused; UUID is always 16 bytes
		b := []byte(s.dictionary[byteStart : byteStart+16])
		return scm.NewString(decompressBytes(b, s.format, 36))
	case FormatBase64Upper:
		if lensVal == 0 {
			return scm.NewString("")
		}
		b := []byte(s.dictionary[byteStart : byteStart+lensVal])
		return scm.NewString(base64.StdEncoding.EncodeToString(b))
	case FormatBase64Lower:
		if lensVal == 0 {
			return scm.NewString("")
		}
		b := []byte(s.dictionary[byteStart : byteStart+lensVal])
		return scm.NewString(base64.URLEncoding.EncodeToString(b))
	default:
		return scm.NewNil()
	}
}

func (s *StorageString) prepare() {
	s.starts.prepare()
	s.lens.prepare()
	s.values.prepare()
	s.reverseMap = make(map[string][3]uint)
	s.prefixstat = make(map[string]int)
	s.validFormats = allFormatsValid
}

func (s *StorageString) scan(i uint32, value scm.Scmer) {
	var v string
	if value.IsNil() {
		if s.nodict {
			s.starts.scan(i, scm.NewNil())
		} else {
			s.values.scan(i, scm.NewNil())
		}
		return
	}
	v = scm.String(value)

	// accumulate format compatibility
	s.validFormats &= checkFormatBits(v)

	// check if we have common prefix (for future prefix compression)
	if s.laststr != v {
		commonlen := 0
		for commonlen < len(s.laststr) && commonlen < len(v) && s.laststr[commonlen] == v[commonlen] {
			s.prefixstat[v[0:commonlen]] = s.prefixstat[v[0:commonlen]] + 1
			commonlen++
		}
		if v != "" {
			s.laststr = v
		}
	}

	// switch to nodict after 100 items with no repetition
	if i == 100 && len(s.reverseMap) > 99 {
		s.nodict = true
		s.reverseMap = nil
		s.sb.Reset()
		if s.values.hasNull {
			s.starts.scan(0, scm.NewNil())
		}
	}

	s.allsize = s.allsize + len(v)
	if s.nodict {
		s.starts.scan(i, scm.NewInt(int64(s.allsize)))
		s.lens.scan(i, scm.NewInt(int64(len(v))))
	} else {
		start, ok := s.reverseMap[v]
		if ok {
			// reuse of string
		} else {
			start[0] = s.count
			start[1] = uint(s.sb.Len())
			start[2] = uint(len(v))
			s.sb.WriteString(v)
			s.starts.scan(uint32(start[0]), scm.NewInt(int64(start[1])))
			s.lens.scan(uint32(start[0]), scm.NewInt(int64(start[2])))
			s.reverseMap[v] = start
			s.count = s.count + 1
		}
		s.values.scan(i, scm.NewInt(int64(start[0])))
	}
}

func (s *StorageString) init(i uint32) {
	s.prefixstat = nil
	s.format = chooseBestFormat(s.validFormats)

	if s.nodict {
		s.starts.init(i)
		adjustLensForFormat(&s.lens, s.format)
		s.lens.init(i)
	} else {
		// compress the dictionary collected during scan
		rawDict := s.sb.String()
		s.sb.Reset()

		s.values.init(i)
		s.starts.init(uint32(s.count))
		adjustLensForFormat(&s.lens, s.format)
		s.lens.init(uint32(s.count))

		if s.format == FormatRaw {
			s.dictionary = rawDict
			for _, start := range s.reverseMap {
				s.starts.build(uint32(start[0]), scm.NewInt(int64(start[1])))
				s.lens.build(uint32(start[0]), scm.NewInt(int64(start[2])))
			}
		} else {
			// re-encode each unique string in compressed form
			s.compBuf = s.compBuf[:0]
			for origStr, start := range s.reverseMap {
				byteOffset := len(s.compBuf)
				s.compBuf = compressString(s.compBuf, origStr, s.format)
				compLen := len(s.compBuf) - byteOffset
				s.starts.build(uint32(start[0]), scm.NewInt(int64(byteOffset)))
				switch s.format {
				case FormatUUIDLower, FormatUUIDUpper:
					// lens unused for UUID (always 16 bytes)
				case FormatBase64Upper, FormatBase64Lower:
					// store decoded byte count
					s.lens.build(uint32(start[0]), scm.NewInt(int64(compLen)))
				default:
					// nibble formats: store original char count
					s.lens.build(uint32(start[0]), scm.NewInt(int64(len(origStr))))
				}
			}
			s.dictionary = string(s.compBuf)
			s.compBuf = nil
		}
	}
}

func (s *StorageString) build(i uint32, value scm.Scmer) {
	if value.IsNil() {
		if s.nodict {
			s.starts.build(i, scm.NewNil())
		} else {
			s.values.build(i, scm.NewNil())
		}
		return
	}
	v := scm.String(value)
	if s.nodict {
		byteOffset := len(s.compBuf)
		s.compBuf = compressString(s.compBuf, v, s.format)
		compLen := len(s.compBuf) - byteOffset
		s.starts.build(i, scm.NewInt(int64(byteOffset)))
		switch s.format {
		case FormatUUIDLower, FormatUUIDUpper:
			// lens unused for UUID (always 16 bytes)
		case FormatBase64Upper, FormatBase64Lower:
			// store decoded byte count
			s.lens.build(i, scm.NewInt(int64(compLen)))
		default:
			// nibble formats: store original char count
			s.lens.build(i, scm.NewInt(int64(len(v))))
		}
	} else {
		start := s.reverseMap[v]
		s.values.build(i, scm.NewInt(int64(start[0])))
	}
}

func (s *StorageString) finish() {
	if s.nodict {
		s.dictionary = string(s.compBuf)
		s.compBuf = nil
	} else {
		s.reverseMap = nil
		s.values.finish()
	}
	s.starts.finish()
	s.lens.finish()
}

func (s *StorageString) proposeCompression(i uint32) ColumnStorage {
	// prefix-tree compression placeholder (see TODO below)
	/* TODO: reactivate as soon as StoragePrefix has a proper implementation for Serialize/Deserialize
	...
	*/
	return nil
}
