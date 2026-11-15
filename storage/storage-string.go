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
import "sync/atomic"
import "encoding/hex"
import "encoding/base64"
import "encoding/binary"
import "github.com/carli2/hybridsort"
import "github.com/launix-de/memcp/scm"

// StringFormat describes how the string bytes in the dictionary are encoded.
// In legacy IDs 0..10, the lowest bit encodes case for paired formats:
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

	// Permanent new IDs: sorted alphabets, high nibble first. Old IDs stay readable.
	FormatOrderedHexLower  StringFormat = 11
	FormatOrderedHexUpper  StringFormat = 12
	FormatOrderedPhone     StringFormat = 13
	FormatOrderedPhoneDTMF StringFormat = 14
	FormatOrderedDecimal   StringFormat = 15
	FormatOrderedDateTime  StringFormat = 16
	// Permanent IDs: unpadded Base64 and sorted timestamp alphabets.
	FormatRawBase64Std   StringFormat = 17
	FormatRawBase64URL   StringFormat = 18
	FormatDateTimeZulu   StringFormat = 19
	FormatDateTimeOffset StringFormat = 20
)

// nibbleCharset holds the encode (index→char) and decode (char→index) tables
// for 4-bit nibble-packed string formats.
type nibbleCharset struct {
	highFirst bool
	enc       [16]byte
	dec       [256]int8
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

// orderedCharsets uses the same immutable alphabet definitions as CString comparisons.
var orderedCharsets = func() [10]nibbleCharset {
	var result [10]nibbleCharset
	for i := range result {
		result[i] = makeNibbleCharset([]byte(scm.CStringAlphabet(uint8(i + 11))))
		result[i].highFirst = true
	}
	return result
}()

// allFormatsValid tracks legacy alphabet IDs and the new IDs 17..20.
// chooseBestFormat maps eligible nibble alphabets to their ordered writer IDs.
const allFormatsValid uint32 = (1 << 11) - 1 | 1<<17 | 1<<18 | 1<<19 | 1<<20

// checkFormatBits returns which StringFormat bits remain compatible with s.
// bit i is set iff StringFormat(i) is still a valid encoding for s.
// FormatRaw (bit 0) is always set.
func checkFormatBits(s string) uint32 {
	if len(s) == 0 {
		// Fixed-width UUID readers always emit 36 characters, so cannot represent empty text.
		return allFormatsValid &^ (1<<FormatUUIDLower | 1<<FormatUUIDUpper)
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
			valid &^= 1<<FormatBase64Upper | 1<<FormatRawBase64Std
		}
		if !isAlpha && !isDigit && c != '-' && c != '_' && c != '=' {
			valid &^= 1<<FormatBase64Lower | 1<<FormatRawBase64URL
		}
		// '=' is only valid at the last two positions in a base64 string
		if c == '=' && i < len(s)-2 {
			valid &^= (1 << FormatBase64Upper) | (1 << FormatBase64Lower)
		}
		if orderedCharsets[FormatDateTimeZulu-11].dec[c] < 0 {
			valid &^= 1 << FormatDateTimeZulu
		}
		if orderedCharsets[FormatDateTimeOffset-11].dec[c] < 0 {
			valid &^= 1 << FormatDateTimeOffset
		}
		if c == '=' {
			valid &^= 1<<FormatRawBase64Std | 1<<FormatRawBase64URL
		}
		// early exit once only FormatRaw remains
		if valid == 1 {
			return valid
		}
	}

	// Character checks above prove all complete interior quartets valid.
	// Strictly validate the tail: padding shape and unused bits must round-trip.
	tail := len(s) % 4
	if tail == 0 {
		tail = 4
	}
	var decoded [3]byte
	for _, candidate := range base64Formats {
		if valid&(1<<candidate.format) != 0 {
			if _, err := candidate.encoding.Decode(decoded[:], []byte(s[len(s)-tail:])); err != nil {
				valid &^= 1 << candidate.format
			}
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
func chooseBestFormat(valid uint32) StringFormat {
	// UUID: 16 bytes vs 36 chars → ~56% savings (best for fixed-length UUID columns)
	if valid&(1<<FormatUUIDLower) != 0 {
		return FormatUUIDLower
	}
	if valid&(1<<FormatUUIDUpper) != 0 {
		return FormatUUIDUpper
	}
	// Hex/nibble sets: 50% savings
	if valid&(1<<FormatHexLower) != 0 {
		return FormatOrderedHexLower
	}
	if valid&(1<<FormatHexUpper) != 0 {
		return FormatOrderedHexUpper
	}
	if valid&(1<<FormatPhone) != 0 {
		return FormatOrderedPhone
	}
	if valid&(1<<FormatPhoneDTMF) != 0 {
		return FormatOrderedPhoneDTMF
	}
	if valid&(1<<FormatDecimal) != 0 {
		return FormatOrderedDecimal
	}
	if valid&(1<<FormatDateTime) != 0 {
		return FormatOrderedDateTime
	}
	for _, f := range []StringFormat{FormatDateTimeZulu, FormatDateTimeOffset} {
		if valid&(1<<f) != 0 {
			return f
		}
	}
	// Base64: ~25% savings
	if valid&(1<<FormatBase64Upper) != 0 {
		return FormatBase64Upper
	}
	if valid&(1<<FormatBase64Lower) != 0 {
		return FormatBase64Lower
	}
	if valid&(1<<FormatRawBase64Std) != 0 {
		return FormatRawBase64Std
	}
	if valid&(1<<FormatRawBase64URL) != 0 {
		return FormatRawBase64URL
	}
	return FormatRaw
}

// appendNibbles encodes s into 4-bit nibbles using cs and appends to dst.
// compNibble is the absolute nibble position in the buffer (= byte_offset*2 + nibble_within_byte).
// Returns (new dst, new absolute nibble position = compNibble + len(s)).
func appendNibbles(dst []byte, s string, cs *nibbleCharset, compNibble int) ([]byte, int) {
	firstShift := uint(0)
	if cs.highFirst {
		firstShift = 4
	}
	for i := 0; i < len(s); i++ {
		nib := byte(cs.dec[s[i]])
		if (compNibble+i)%2 == 0 {
			dst = append(dst, nib<<firstShift)
		} else {
			dst[len(dst)-1] |= nib << (firstShift ^ 4)
		}
	}
	return dst, compNibble + len(s)
}

// isNibbleFormat reports whether format uses 4-bit nibble packing.
func isNibbleFormat(f StringFormat) bool {
	if (f >= FormatOrderedHexLower && f <= FormatOrderedDateTime) || f == FormatDateTimeZulu || f == FormatDateTimeOffset {
		return true
	}
	switch f {
	case FormatHexLower, FormatHexUpper, FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
		return true
	}
	return false
}

// nibbleCharsetFor returns the nibble charset for a nibble format, or nil.
func nibbleCharsetFor(f StringFormat) *nibbleCharset {
	if (f >= FormatOrderedHexLower && f <= FormatOrderedDateTime) || f == FormatDateTimeZulu || f == FormatDateTimeOffset {
		return &orderedCharsets[f-FormatOrderedHexLower]
	}
	switch f {
	case FormatHexLower:
		return &hexLowerCharset
	case FormatHexUpper:
		return &hexUpperCharset
	case FormatPhone:
		return &phoneCharset
	case FormatPhoneDTMF:
		return &phoneDTMFCharset
	case FormatDecimal:
		return &decimalCharset
	case FormatDateTime:
		return &dateTimeCharset
	}
	return nil
}

// compressNonNibble encodes s for UUID, Base64, or Raw formats and appends to dst.
func compressNonNibble(dst []byte, s string, format StringFormat) []byte {
	switch format {
	case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
		b, err := base64Encoding(format).DecodeString(s)
		if err != nil {
			panic(fmt.Sprintf("compressNonNibble: invalid base64: %v", err))
		}
		return append(dst, b...)
	case FormatUUIDLower, FormatUUIDUpper:
		hexStr := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
		b, _ := hex.DecodeString(hexStr)
		return append(dst, b...)
	default: // FormatRaw
		return append(dst, s...)
	}
}

// adjustLensForFormat corrects StorageInt scan statistics (offset/max) to match
// the values that build() will actually write, so that init() allocates the right
// bitsize.  Must be called BEFORE lens.init().
func adjustLensForFormat(lens *StorageInt, format StringFormat) {
	switch format {
	case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
		// build() stores decoded byte count; use 0 as safe lower bound to avoid
		// negative relative values from padding variability
		lens.offset = 0
		lens.max = int64(base64Encoding(format).DecodedLen(int(lens.max)))
		// All nibble formats (Hex, Phone, PhoneDTMF, Decimal, DateTime): char count == raw len → no adjustment
		// UUID: lens is never read → no adjustment
		// Raw: byte count == char count → no adjustment
	}
}

// adjustStartsForFormat resets the starts StorageInt offset to 0 for nodict mode.
// During scan, starts accumulates END positions (allsize), but build() writes
// START positions beginning from 0.  Setting offset=0 ensures all build values fit.
// Must be called BEFORE starts.init() in the nodict path.
func adjustStartsForFormat(starts *StorageInt) {
	starts.offset = 0
}

// readNibbles decodes charLen nibble-encoded characters from ptr.
// nibbleOff (0 or 1) is the nibble index within *ptr where decoding starts.
// writeNibblesInto decodes len(dst) nibble-encoded characters from ptr
// directly into dst. Shared by the single-value lazy decode path
// (readNibbles) and the bulk arena decode path (bulkDecoder), so the nibble
// unpacking logic exists exactly once regardless of whether the caller
// wants a freshly allocated string or a slice of a shared batch buffer.
// Decode full bytes in pairs. Peel the first/last half-byte and select the
// nibble order once, keeping parity calculations out of the main loops.
func writeNibblesInto(dst []byte, ptr *byte, nibbleOff int, cs *nibbleCharset) {
	if len(dst) == 0 {
		return
	}
	data := unsafe.Slice(ptr, (len(dst)+nibbleOff+1)/2)
	i, j := 0, 0
	if nibbleOff != 0 {
		if cs.highFirst {
			dst[0] = cs.enc[data[0]&15]
		} else {
			dst[0] = cs.enc[data[0]>>4]
		}
		i++
		j++
	}
	if cs.highFirst {
		for ; i+1 < len(dst); i, j = i+2, j+1 {
			v := data[j]
			dst[i] = cs.enc[v>>4]
			dst[i+1] = cs.enc[v&15]
		}
		if i < len(dst) {
			dst[i] = cs.enc[data[j]>>4]
		}
	} else {
		for ; i+1 < len(dst); i, j = i+2, j+1 {
			v := data[j]
			dst[i] = cs.enc[v&15]
			dst[i+1] = cs.enc[v>>4]
		}
		if i < len(dst) {
			dst[i] = cs.enc[data[j]&15]
		}
	}
}

func readNibbles(ptr *byte, nibbleOff int, charLen int, cs *nibbleCharset) string {
	if charLen == 0 {
		return ""
	}
	result := make([]byte, charLen)
	writeNibblesInto(result, ptr, nibbleOff, cs)
	return unsafe.String(&result[0], charLen)
}

// writeUUIDInto decodes the 16 raw bytes at ptr into dst[:36] as a dashed
// UUID string (upper selects the letter case). Shared by readUUID (single
// value) and bulkDecoder (batch arena).
func writeUUIDInto(dst []byte, ptr *byte, upper bool) {
	b := unsafe.Slice(ptr, 16)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	if upper {
		for i, c := range dst {
			if c >= 'a' && c <= 'f' {
				dst[i] = c - ('a' - 'A')
			}
		}
	}
}

func readUUID(ptr *byte, upper bool) string {
	var buf [36]byte
	writeUUIDInto(buf[:], ptr, upper)
	return string(buf[:])
}

// cstringDecompress materialises a tagCString Scmer into a plain Go string.
// ptr is a byte-aligned pointer into the StorageString dictionary.
// val encodes: bits 47-43 = format, bit 42 = nibbleOff, bits 41-0 = charLen (RAM only).
func cstringDecompress(ptr *byte, val uint64) string {
	format := StringFormat(val >> scm.CStringFormatShift)
	nibbleOff := int((val >> scm.CStringOffsetShift) & 1)
	charLen := int(val & scm.CStringLengthMask)
	if cs := nibbleCharsetFor(format); cs != nil {
		return readNibbles(ptr, nibbleOff, charLen, cs)
	}
	switch format {
	case FormatUUIDLower:
		return readUUID(ptr, false)
	case FormatUUIDUpper:
		return readUUID(ptr, true)
	default: // FormatRaw
		if charLen == 0 {
			return ""
		}
		return unsafe.String(ptr, charLen)
	}
}

func init() {
	scm.CStringDecompress = cstringDecompress
}

type StorageString struct {
	storageJITFunctions
	// StorageInt for dictionary entries
	values StorageInt `jit:"immutable-after-finish"`
	// the dictionary: compressed bytes for all values concatenated
	dictionary string
	starts     StorageInt   `jit:"immutable-after-finish"` // byte offsets into dictionary
	lens       StorageInt   `jit:"immutable-after-finish"` // compressed byte lengths
	nodict     bool         `jit:"immutable-after-finish"` // disable values array
	format     StringFormat `jit:"immutable-after-finish"`

	// LZ4 compression: when compressed==true, compressedDict holds the
	// Legacy-only LZ4 payload; normalized before publication.
	compressedDict []byte // lz4-compressed dictionary bytes
	compressed     bool   // true → dictionary is stored lz4-compressed
	// Immutable post-compression payload; starts/lens retain the entry directory. Construction and
	// cold loading exclusively own these fields before generation publication.
	// Scmer handles retain the buffer header, never this mutable storage object.
	textBuffer *StringConcatCompression
	// Immutable preparation result: only eligible raw text can expose tagConcat.
	bufferHandles bool   `jit:"immutable-after-finish"`
	readCount     uint64 // atomic; batch-boundary telemetry, never scalar row-loop writes

	// helpers (scan/build phase only)
	sb           strings.Builder
	compBuf      []byte // accumulates compressed bytes during build (nodict) or init (dict)
	compNibble   int    // current nibble position in compBuf (nodict nibble build / dict nibble init)
	reverseMap   map[string][3]uint
	count        uint
	allsize      int
	validFormats uint32
	// prefix statistics

}

func (s *StorageString) ComputeSize() uint { return s.payloadSize() }
func (s *StorageString) payloadSize() uint {
	base := uint(unsafe.Sizeof(*s) - unsafe.Sizeof(s.values) - unsafe.Sizeof(s.starts) - unsafe.Sizeof(s.lens))
	base += s.values.ComputeSize() + s.starts.ComputeSize() + s.lens.ComputeSize() + uint(len(s.dictionary)) + uint(cap(s.compressedDict))
	if s.textBuffer != nil {
		base += s.textBuffer.Size()
	}
	return base
}
func (s *StorageString) String() string {
	if s.textBuffer != nil {
		return fmt.Sprintf("string-concat[%d entries; %d bytes]", s.starts.count, s.textBuffer.Size())
	}
	return fmt.Sprintf("string-raw[%d entries; %d bytes, format=%d]", s.starts.count, len(s.dictionary), s.format)
}
func (s *StorageString) ReadCount() uint64  { return atomic.LoadUint64(&s.readCount) }
func (s *StorageString) ensureDict() string { return s.dictionary }

// storageStringVersion is the current binary format version for StorageString.
// Increment this constant and add a new deserializeStringV* helper whenever the
// layout after [nodict][format] changes.  Never delete old helpers.
//
// NOTE: The version byte occupies pad[0] (first of the 5 alignment bytes after
// format).  Old "smallerstrings" data had 0 there (pad was zero-filled), so it
// reads correctly as version 0.
//
// Version 2 introduces ordered nibble IDs 11..16; version 3 adds IDs 17..20.
// Version 4 adds immutable LZ4 buffer payloads (flag 2). Flags 0/1 retain
// the V1 layout. Version 5 always writes a block buffer, without a flag or
// redundant offsets. Version 6 replaces LZ4 with direct concat-pool references.
// Historical readers normalize private data before publication.
// The historical raw sentinel is exactly ASCII '1' (49), never a format range.
const storageStringVersion = 6

// StorageString binary layout (magic byte 20 consumed by shard loader):
//
//	[nodict uint8]         ← 0=dict mode, 1=buffer mode
//	[format uint8]         ← StringFormat (0..20); ASCII 49: legacy raw sentinel
//
//	Legacy (format byte == 49, i.e. '1'=49 from old ASCII dummy "123456"):
//	  [legacyPad 5 bytes]  ← consume remaining dummy bytes; format = FormatRaw
//	  [count uint64] [values StorageInt] [starts StorageInt] [lens StorageInt]
//	  [dictlen uint64] [dict bytes]
//
//	Version 0:
//	  [version uint8]      ← pad[0], was 0 in all pre-versioning smallerstrings data
//	  [pad 4 bytes]        ← alignment padding
//	  [count uint64] [values StorageInt] [starts StorageInt] [lens StorageInt]
//	  [dictlen uint64] [dict bytes]
//
//	Version 1/2/3:
//	  [version uint8]      ← 1, 2 or 3
//	  [pad 4 bytes]        ← alignment padding
//	  [compressed uint8]   ← 0=uncompressed dict, 1=lz4-compressed dict
//	  [count uint64] [values StorageInt] [starts StorageInt] [lens StorageInt]
//	  if compressed==0: [dictlen uint64] [dict bytes]
//	  if compressed==1: [compressedLen uint64] [compressedDict bytes]
//
// Version history:
//
//	0: smallerstrings format; format byte 0..10; version in pad[0].
//	1: adds lz4-compressed dictionary support.
//	2: ordered nibble IDs 11..16; V1 body layout.
//	3: unpadded Base64 and timestamp IDs 17..20; V1 body layout.
//	4: immutable buffer payloads with flags 0/1/2.
//	5: block payload only.
//	6 (current): raw bytes or concat pool; see storage-string-buffer.go.
func (s *StorageString) Serialize(f io.Writer) { s.serializeStringV6(f) }
func (s *StorageString) Deserialize(f io.Reader) uint {
	count := s.deserializeString(f)
	s.CompressDictionary()
	return count
}

func (s *StorageString) deserializeString(f io.Reader) uint {
	var nodict uint8
	binary.Read(f, binary.LittleEndian, &nodict)
	if nodict == 1 {
		s.nodict = true
	}
	var formatByte uint8
	binary.Read(f, binary.LittleEndian, &formatByte)
	// Legacy compatibility: old format wrote the ASCII string "123456" as a
	// 6-byte dummy.  The first byte of that dummy is '1' (0x31 = 49).
	// Recognize the actual historical sentinel, preserving IDs above 10.
	if formatByte == '1' {
		var legacyPad [5]byte
		readStorageBytes(f, legacyPad[:])
		s.format = FormatRaw
		return s.deserializeStringBody(f)
	}
	if formatByte > uint8(FormatDateTimeOffset) {
		panic(fmt.Sprintf("StorageString: unknown format %d", formatByte))
	}
	s.format = StringFormat(formatByte)
	var version uint8
	binary.Read(f, binary.LittleEndian, &version) // pad[0] repurposed as version byte
	var pad [4]byte
	readStorageBytes(f, pad[:])
	if formatByte >= 11 && version < 2 {
		panic("StorageString: ordered format requires version 2")
	}
	if formatByte >= 17 && version < 3 {
		panic("StorageString: new format requires version 3")
	}
	switch version {
	case 0:
		return s.deserializeStringBody(f)
	case 1:
		return s.deserializeStringV1(f)
	case 2:
		return s.deserializeStringV2(f)
	case 3:
		return s.deserializeStringV3(f)
	case 4:
		return s.deserializeStringV4(f)
	case 5:
		return s.deserializeStringV5(f)
	case 6:
		return s.deserializeStringV6(f)
	default:
		panic(fmt.Sprintf("StorageString: unknown version %d", version))
	}
}

func (s *StorageString) deserializeStringBody(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values.DeserializeEx(f, true)
	s.count = s.starts.DeserializeEx(f, true)
	s.lens.DeserializeEx(f, true)
	var dictionarylength uint64
	binary.Read(f, binary.LittleEndian, &dictionarylength)
	if dictionarylength > 0 {
		rawdata := make([]byte, dictionarylength)
		readStorageBytes(f, rawdata)
		s.dictionary = unsafe.String(&rawdata[0], dictionarylength)
	}
	return uint(l)
}

func (s *StorageString) deserializeStringV1(f io.Reader) uint {
	var compFlag uint8
	binary.Read(f, binary.LittleEndian, &compFlag)

	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values.DeserializeEx(f, true)
	s.count = s.starts.DeserializeEx(f, true)
	s.lens.DeserializeEx(f, true)

	var dictLen uint64
	binary.Read(f, binary.LittleEndian, &dictLen)
	if dictLen > 0 {
		rawdata := make([]byte, dictLen)
		readStorageBytes(f, rawdata)
		if compFlag == 1 {
			s.compressedDict = rawdata
			s.compressed = true
			// dictionary stays empty; ensureDict() will decompress on first read
		} else {
			s.dictionary = unsafe.String(&rawdata[0], dictLen)
		}
	}
	return uint(l)
}

func (s *StorageString) deserializeStringV2(f io.Reader) uint {
	return s.deserializeStringV1(f)
}

func (s *StorageString) deserializeStringV3(f io.Reader) uint { return s.deserializeStringV1(f) }

func (s *StorageString) GetCachedReader(hints ColumnReadHints) ColumnReader {
	return s.storageJITFunctions.reader(s)
}

func (s *StorageString) GetValue(i uint32) scm.Scmer {
	if s.bufferHandles {
		return s.compressedAt(i)
	}
	dict := s.ensureDict()
	dictBase := unsafe.Pointer(unsafe.StringData(dict))
	return s.decodeAt(i, dict, dictBase)
}

// decodeAt is the single-row decode path: it returns a lazy CString/BString
// for nibble/UUID/Base64 formats, deferring the actual decode to whenever
// (and however many times) the value is eventually stringified. That is the
// right tradeoff for a single GetValue call, where the caller may or may not
// end up using the value (e.g. a rejected WHERE-filter row). The bulk
// GetValueMulti/GetValueRange path below intentionally does not share this:
// a bulk call is a promise that every value will be used, so it decodes
// eagerly into one shared arena instead (see decodeBulk).
func (s *StorageString) decodeAt(i uint32, dict string, dictBase unsafe.Pointer) scm.Scmer {
	var startVal, lensVal uint64
	if s.nodict {
		sv := uint64(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
		if s.starts.hasNull && sv == uint64(s.starts.null) {
			return scm.NewNil()
		}
		startVal = sv
		lensVal = uint64(int64(s.lens.GetValueUInt(i)) + s.lens.offset)
	} else {
		idx := uint32(int64(s.values.GetValueUInt(i)) + s.values.offset)
		if s.values.hasNull && idx == uint32(s.values.null) {
			return scm.NewNil()
		}
		startVal = uint64(int64(s.starts.GetValueUInt(idx)) + s.starts.offset)
		lensVal = uint64(int64(s.lens.GetValueUInt(idx)) + s.lens.offset)
	}

	// startVal semantics by format:
	//   nibble formats: nibble position (byteOff*2 + nibbleOff)
	//   UUID, Base64, Raw: byte offset into dictionary
	// lensVal semantics:
	//   nibble formats: original char count
	//   Base64: decoded byte count
	//   UUID: unused (always 16 bytes / 36 chars)
	//   Raw: byte count
	switch s.format {
	case FormatRaw:
		byteStart := startVal
		return scm.NewString(dict[byteStart : byteStart+lensVal])
	case FormatHexLower, FormatHexUpper,
		FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime,
		FormatOrderedHexLower, FormatOrderedHexUpper, FormatOrderedPhone, FormatOrderedPhoneDTMF, FormatOrderedDecimal, FormatOrderedDateTime, FormatDateTimeZulu, FormatDateTimeOffset:
		nibblePos := startVal
		nibbleOff := uint8(nibblePos & 1)
		byteOff := nibblePos >> 1
		charLen := int(lensVal)
		if charLen == 0 {
			return scm.NewString("")
		}
		ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(byteOff)))
		return scm.NewCString(ptr, uint8(s.format), nibbleOff, charLen)
	case FormatUUIDLower, FormatUUIDUpper:
		byteOff := startVal
		ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(byteOff)))
		return scm.NewCString(ptr, uint8(s.format), 0, 36)
	case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
		byteOff := startVal
		decodedLen := int(lensVal)
		if decodedLen == 0 {
			return scm.NewString("")
		}
		ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(byteOff)))
		return scm.NewBString(ptr, decodedLen, s.format == FormatBase64Lower || s.format == FormatRawBase64URL, s.format == FormatRawBase64Std || s.format == FormatRawBase64URL)
	default:
		return scm.NewNil()
	}
}

// GetValueRange and GetValueMulti resolve every requested row's dictionary
// position through the underlying starts/lens/values StorageInts' own bulk
// methods (one call each, reusing their rolling-cursor optimization) instead
// of decodeAt's per-row GetValueUInt calls, then decode the whole batch's
// string content into one shared []byte arena instead of returning one lazy
// CString/BString Scmer per row. A single GetValue only materializes the one
// value it was asked for, so staying lazy there is correct; a bulk call is a
// promise that every requested value will actually be used, so eagerly
// decoding once into a shared, cache-local buffer beats paying for N small
// allocations (one per lazy Scmer's eventual .String() call, or more if
// called repeatedly) later.
//
//jitgen:control-flow-stable recid count target/1 stride
func (s *StorageString) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	if s.bufferHandles {
		atomic.AddUint64(&s.readCount, uint64(count))
		for i := uint32(0); i < count; i++ {
			target[int(i)*stride] = s.compressedAt(recid + i)
		}
		return
	}
	recids := make([]uint32, count)
	for k := range recids {
		recids[k] = recid + uint32(k)
	}
	s.getValueBulk(recids, target, stride)
}

//jitgen:control-flow-stable recids/2 target/1 stride
func (s *StorageString) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if len(recids) == 0 {
		return
	}
	if s.bufferHandles {
		atomic.AddUint64(&s.readCount, uint64(len(recids)))
		for i, recid := range recids {
			target[i*stride] = s.compressedAt(recid)
		}
		return
	}
	s.getValueBulk(recids, target, stride)
}

func (s *StorageString) getValueBulk(recids []uint32, target []scm.Scmer, stride int) {
	atomic.AddUint64(&s.readCount, uint64(len(recids)))
	rawStarts, rawLens, isNil := s.resolvePositions(recids)
	s.decodeBulk(rawStarts, rawLens, isNil, target, stride)
}

// resolvePositions bulk-resolves the raw (offset not yet applied) start+lens
// position for every requested row. In dict mode, a row's dictionary entry
// index is looked up first (bulk), and starts/lens are then looked up at
// those entry indices (also bulk — getUIntMultiRaw handles the resulting
// non-sequential index list correctly, just without the sequential fast
// path). dictIdxs stays row-indexed (length n, garbage-but-safe 0 for NULL
// rows) rather than compacted to the non-NULL subset, trading a few wasted
// starts/lens lookups for NULL rows against dropping a whole extra
// position-mapping array and pass.
func (s *StorageString) resolvePositions(recids []uint32) (rawStarts, rawLens []uint64, isNil []bool) {
	n := len(recids)
	isNil = make([]bool, n)

	if s.nodict {
		rawStarts = make([]uint64, n)
		s.starts.getUIntMultiRaw(recids, rawStarts)
		for i := 0; i < n; i++ {
			if s.starts.hasNull && rawStarts[i] == s.starts.null {
				isNil[i] = true
			}
		}
		rawLens = make([]uint64, n)
		s.lens.getUIntMultiRaw(recids, rawLens)
		return
	}

	rawValues := make([]uint64, n)
	s.values.getUIntMultiRaw(recids, rawValues)
	dictIdxs := make([]uint32, n)
	for i := 0; i < n; i++ {
		if s.values.hasNull && rawValues[i] == s.values.null {
			isNil[i] = true
			continue
		}
		dictIdxs[i] = uint32(int64(rawValues[i]) + s.values.offset)
	}
	rawStarts = make([]uint64, n)
	rawLens = make([]uint64, n)
	if s.starts.count > 0 {
		s.starts.getUIntMultiRaw(dictIdxs, rawStarts)
		s.lens.getUIntMultiRaw(dictIdxs, rawLens)
	}
	return
}

// bulkDecodeFn decodes one row's content into dst (already sized to that
// row's exact output length) given its dictionary start/lens position.
type bulkDecodeFn func(dst []byte, dictBase unsafe.Pointer, dict string, startVal, lensVal uint64)

// bulkDecoder resolves s.format once per batch (not once per row) into an
// output-length function and a decode function, mirroring cstringDecompress's
// format switch but writing into a caller-provided slice instead of
// allocating a fresh string.
func (s *StorageString) bulkDecoder() (outLen func(lensVal uint64) int, decode bulkDecodeFn, unknownFormat bool) {
	if cs := nibbleCharsetFor(s.format); cs != nil {
		return func(lensVal uint64) int { return int(lensVal) },
			func(dst []byte, dictBase unsafe.Pointer, dict string, startVal, lensVal uint64) {
				ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(startVal>>1)))
				writeNibblesInto(dst, ptr, int(startVal&1), cs)
			}, false
	}
	switch s.format {
	case FormatRaw:
		return func(lensVal uint64) int { return int(lensVal) },
			func(dst []byte, dictBase unsafe.Pointer, dict string, startVal, lensVal uint64) {
				copy(dst, dict[startVal:startVal+lensVal])
			}, false
	case FormatUUIDLower, FormatUUIDUpper:
		upper := s.format == FormatUUIDUpper
		return func(uint64) int { return 36 },
			func(dst []byte, dictBase unsafe.Pointer, dict string, startVal, lensVal uint64) {
				ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(startVal)))
				writeUUIDInto(dst, ptr, upper)
			}, false
	case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
		enc := base64Encoding(s.format)
		return func(lensVal uint64) int { return enc.EncodedLen(int(lensVal)) },
			func(dst []byte, dictBase unsafe.Pointer, dict string, startVal, lensVal uint64) {
				ptr := (*byte)(unsafe.Add(dictBase, startVal))
				enc.Encode(dst, unsafe.Slice(ptr, int(lensVal)))
			}, false
	default:
		return nil, nil, true
	}
}

// decodeBulk decodes every row's content into one shared []byte arena
// (sized in a first pass, filled in a second) and wraps each row's slice of
// it as a zero-copy scm.NewString view — one allocation for the whole batch
// instead of one per row.
// decodeBulk decodes every row's content into one shared []byte arena and
// wraps each row's slice of it as a zero-copy scm.NewString view — one
// allocation for the whole batch instead of one per row. It applies the
// starts/lens StorageInt offsets inline (per row, in the size pass and again
// in the decode pass) instead of pre-computing and storing offset-applied
// values in their own arrays, trading a cheap redundant add+outLen() call
// for one fewer n-sized array and one fewer pass over it.
func (s *StorageString) decodeBulk(rawStarts, rawLens []uint64, isNil []bool, target []scm.Scmer, stride int) {
	n := len(rawStarts)
	idx := 0
	if s.format == FormatUUIDLower || s.format == FormatUUIDUpper {
		// Keep UUID batches packed too. Each handle owns its immutable dictionary
		// bytes through a GC-visible pointer, independently of decoded-cache eviction.
		dict := s.ensureDict()
		base := unsafe.Pointer(unsafe.StringData(dict))
		for i := range rawStarts {
			if isNil[i] {
				target[i*stride] = scm.NewNil()
				continue
			}
			start := int64(rawStarts[i]) + s.starts.offset
			target[i*stride] = scm.NewCString((*byte)(unsafe.Add(base, start)), uint8(s.format), 0, 36)
		}
		return
	}
	outLen, decode, unknownFormat := s.bulkDecoder()
	if unknownFormat {
		for i := 0; i < n; i++ {
			target[idx] = scm.NewNil()
			idx += stride
		}
		return
	}

	dict := s.ensureDict()
	dictBase := unsafe.Pointer(unsafe.StringData(dict))
	startsOffset := s.starts.offset
	lensOffset := s.lens.offset

	total := 0
	for i := 0; i < n; i++ {
		if !isNil[i] {
			total += outLen(uint64(int64(rawLens[i]) + lensOffset))
		}
	}

	buf := make([]byte, total)
	offset := 0
	for i := 0; i < n; i++ {
		if isNil[i] {
			target[idx] = scm.NewNil()
			idx += stride
			continue
		}
		lensVal := uint64(int64(rawLens[i]) + lensOffset)
		l := outLen(lensVal)
		if l == 0 {
			target[idx] = scm.NewString("")
		} else {
			startVal := uint64(int64(rawStarts[i]) + startsOffset)
			row := buf[offset : offset+l]
			decode(row, dictBase, dict, startVal, lensVal)
			target[idx] = scm.NewString(unsafe.String(&row[0], l))
			offset += l
		}
		idx += stride
	}
}

func (s *StorageString) prepare() {
	s.starts.prepare()
	s.lens.prepare()
	s.values.prepare()
	s.reverseMap = make(map[string][3]uint)
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
	s.format = chooseBestFormat(s.validFormats)

	if s.nodict {
		adjustStartsForFormat(&s.starts)
		s.starts.init(i)
		adjustLensForFormat(&s.lens, s.format)
		s.lens.init(i)
	} else {
		// Collect unique strings sorted by raw byte offset (= nibble position for nibble formats).
		// Sorting ensures dense nibble packing produces nibble positions that match scan stats.
		type dictEntry struct {
			orig   string
			idx    uint32
			rawOff uint
		}
		entries := make([]dictEntry, 0, len(s.reverseMap))
		for orig, start := range s.reverseMap {
			entries = append(entries, dictEntry{orig, uint32(start[0]), start[1]})
		}
		hybridsort.Slice(entries, func(a, b int) bool { return entries[a].rawOff < entries[b].rawOff })

		s.values.init(i)
		s.starts.init(uint32(s.count))
		adjustLensForFormat(&s.lens, s.format)
		s.lens.init(uint32(s.count))

		rawDict := s.sb.String()
		s.sb.Reset()

		if s.format == FormatRaw {
			s.dictionary = rawDict
			for _, e := range entries {
				s.starts.build(e.idx, scm.NewInt(int64(e.rawOff)))
				start := s.reverseMap[e.orig]
				s.lens.build(e.idx, scm.NewInt(int64(start[2])))
			}
		} else if isNibbleFormat(s.format) {
			// Dense nibble packing. For ASCII nibble charsets: nibble position = raw byte offset,
			// so processing in sorted order keeps starts.build values equal to scan values.
			cs := nibbleCharsetFor(s.format)
			s.compBuf = s.compBuf[:0]
			s.compNibble = 0
			for _, e := range entries {
				nibblePos := s.compNibble
				s.starts.build(e.idx, scm.NewInt(int64(nibblePos)))
				s.lens.build(e.idx, scm.NewInt(int64(len(e.orig))))
				s.compBuf, s.compNibble = appendNibbles(s.compBuf, e.orig, cs, nibblePos)
			}
			s.dictionary = string(s.compBuf)
			s.compBuf = nil
		} else {
			// UUID and Base64: byte-aligned, no dense packing.
			s.compBuf = s.compBuf[:0]
			for _, e := range entries {
				byteOffset := len(s.compBuf)
				s.compBuf = compressNonNibble(s.compBuf, e.orig, s.format)
				compLen := len(s.compBuf) - byteOffset
				s.starts.build(e.idx, scm.NewInt(int64(byteOffset)))
				switch s.format {
				case FormatUUIDLower, FormatUUIDUpper:
					// lens unused for UUID (always 16 bytes)
				case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
					s.lens.build(e.idx, scm.NewInt(int64(compLen)))
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
		if isNibbleFormat(s.format) {
			cs := nibbleCharsetFor(s.format)
			nibblePos := s.compNibble
			s.starts.build(i, scm.NewInt(int64(nibblePos)))
			s.lens.build(i, scm.NewInt(int64(len(v))))
			s.compBuf, s.compNibble = appendNibbles(s.compBuf, v, cs, nibblePos)
		} else {
			byteOffset := len(s.compBuf)
			s.compBuf = compressNonNibble(s.compBuf, v, s.format)
			compLen := len(s.compBuf) - byteOffset
			s.starts.build(i, scm.NewInt(int64(byteOffset)))
			switch s.format {
			case FormatUUIDLower, FormatUUIDUpper:
				// lens unused for UUID (always 16 bytes)
			case FormatBase64Upper, FormatBase64Lower, FormatRawBase64Std, FormatRawBase64URL:
				s.lens.build(i, scm.NewInt(int64(compLen)))
			default: // FormatRaw
				s.lens.build(i, scm.NewInt(int64(len(v))))
			}
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
	s.CompressDictionary()
	s.storageJITFunctions.finish(s)
}

func (s *StorageString) proposeCompression(i uint32) ColumnStorage { return nil }

func (s *StorageString) DistinctCount() uint {
	if s.nodict {
		return uint(s.starts.count) // no dictionary: row count is the conservative upper bound
	}
	return s.count // dict entry count = distinct values
}

// JITEmit currently preserves the complete string decoding path through GetValue.
func (s *StorageString) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d0 scm.JITValueDesc
	_ = d0
	var d1 scm.JITValueDesc
	_ = d1
	var d5 scm.JITValueDesc
	_ = d5
	var phiBase7 int32
	_ = phiBase7
	var d8 scm.JITValueDesc
	_ = d8
	var d10 scm.JITValueDesc
	_ = d10
	var inlineResultOff9 int32
	var d11 scm.JITValueDesc
	_ = d11
	var d12 scm.JITValueDesc
	_ = d12
	var d13 scm.JITValueDesc
	_ = d13
	var d14 scm.JITValueDesc
	_ = d14
	var d15 scm.JITValueDesc
	_ = d15
	var d16 scm.JITValueDesc
	_ = d16
	var d17 scm.JITValueDesc
	_ = d17
	var d18 scm.JITValueDesc
	_ = d18
	var d19 scm.JITValueDesc
	_ = d19
	var d20 scm.JITValueDesc
	_ = d20
	var d21 scm.JITValueDesc
	_ = d21
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
	var d30 scm.JITValueDesc
	_ = d30
	var d31 scm.JITValueDesc
	_ = d31
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
	var d45 scm.JITValueDesc
	_ = d45
	var d46 scm.JITValueDesc
	_ = d46
	var d47 scm.JITValueDesc
	_ = d47
	var d48 scm.JITValueDesc
	_ = d48
	var d49 scm.JITValueDesc
	_ = d49
	var d50 scm.JITValueDesc
	_ = d50
	var d51 scm.JITValueDesc
	_ = d51
	var d52 scm.JITValueDesc
	_ = d52
	var d53 scm.JITValueDesc
	_ = d53
	var d54 scm.JITValueDesc
	_ = d54
	var d55 scm.JITValueDesc
	_ = d55
	var d56 scm.JITValueDesc
	_ = d56
	var d57 scm.JITValueDesc
	_ = d57
	var d58 scm.JITValueDesc
	_ = d58
	var d59 scm.JITValueDesc
	_ = d59
	var d60 scm.JITValueDesc
	_ = d60
	var d61 scm.JITValueDesc
	_ = d61
	var d62 scm.JITValueDesc
	_ = d62
	var d63 scm.JITValueDesc
	_ = d63
	var d64 scm.JITValueDesc
	_ = d64
	var d65 scm.JITValueDesc
	_ = d65
	var d67 scm.JITValueDesc
	_ = d67
	var d70 scm.JITValueDesc
	_ = d70
	var inlineResultOff69 int32
	var d71 scm.JITValueDesc
	_ = d71
	var d72 scm.JITValueDesc
	_ = d72
	var d73 scm.JITValueDesc
	_ = d73
	var d74 scm.JITValueDesc
	_ = d74
	var d75 scm.JITValueDesc
	_ = d75
	var d76 scm.JITValueDesc
	_ = d76
	var d77 scm.JITValueDesc
	_ = d77
	var d78 scm.JITValueDesc
	_ = d78
	var d79 scm.JITValueDesc
	_ = d79
	var d80 scm.JITValueDesc
	_ = d80
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d84 scm.JITValueDesc
	_ = d84
	var d85 scm.JITValueDesc
	_ = d85
	var d86 scm.JITValueDesc
	_ = d86
	var d87 scm.JITValueDesc
	_ = d87
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
	var d110 scm.JITValueDesc
	_ = d110
	var d111 scm.JITValueDesc
	_ = d111
	var d112 scm.JITValueDesc
	_ = d112
	var d113 scm.JITValueDesc
	_ = d113
	var d114 scm.JITValueDesc
	_ = d114
	var d115 scm.JITValueDesc
	_ = d115
	var d116 scm.JITValueDesc
	_ = d116
	var d117 scm.JITValueDesc
	_ = d117
	var d118 scm.JITValueDesc
	_ = d118
	var d119 scm.JITValueDesc
	_ = d119
	var d120 scm.JITValueDesc
	_ = d120
	var d121 scm.JITValueDesc
	_ = d121
	var d123 scm.JITValueDesc
	_ = d123
	var d125 scm.JITValueDesc
	_ = d125
	var phiBase127 int32
	_ = phiBase127
	var d128 scm.JITValueDesc
	_ = d128
	var d129 scm.JITValueDesc
	_ = d129
	var d130 scm.JITValueDesc
	_ = d130
	var d131 scm.JITValueDesc
	_ = d131
	var d133 scm.JITValueDesc
	_ = d133
	var inlineResultOff132 int32
	var d134 scm.JITValueDesc
	_ = d134
	var d135 scm.JITValueDesc
	_ = d135
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
	var d155 scm.JITValueDesc
	_ = d155
	var d156 scm.JITValueDesc
	_ = d156
	var d157 scm.JITValueDesc
	_ = d157
	var d158 scm.JITValueDesc
	_ = d158
	var d160 scm.JITValueDesc
	_ = d160
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
	var d178 scm.JITValueDesc
	_ = d178
	var d179 scm.JITValueDesc
	_ = d179
	var d180 scm.JITValueDesc
	_ = d180
	var d181 scm.JITValueDesc
	_ = d181
	var d182 scm.JITValueDesc
	_ = d182
	var d183 scm.JITValueDesc
	_ = d183
	var d184 scm.JITValueDesc
	_ = d184
	var d186 scm.JITValueDesc
	_ = d186
	var d187 scm.JITValueDesc
	_ = d187
	var d188 scm.JITValueDesc
	_ = d188
	var d189 scm.JITValueDesc
	_ = d189
	var d190 scm.JITValueDesc
	_ = d190
	var d191 scm.JITValueDesc
	_ = d191
	var d192 scm.JITValueDesc
	_ = d192
	var d193 scm.JITValueDesc
	_ = d193
	var d194 scm.JITValueDesc
	_ = d194
	var d195 scm.JITValueDesc
	_ = d195
	var d196 scm.JITValueDesc
	_ = d196
	var d197 scm.JITValueDesc
	_ = d197
	var d198 scm.JITValueDesc
	_ = d198
	var d199 scm.JITValueDesc
	_ = d199
	var d200 scm.JITValueDesc
	_ = d200
	var d201 scm.JITValueDesc
	_ = d201
	var d202 scm.JITValueDesc
	_ = d202
	var d203 scm.JITValueDesc
	_ = d203
	var d204 scm.JITValueDesc
	_ = d204
	var d205 scm.JITValueDesc
	_ = d205
	var d206 scm.JITValueDesc
	_ = d206
	var d207 scm.JITValueDesc
	_ = d207
	var d208 scm.JITValueDesc
	_ = d208
	var d210 scm.JITValueDesc
	_ = d210
	var d211 scm.JITValueDesc
	_ = d211
	var d212 scm.JITValueDesc
	_ = d212
	var d214 scm.JITValueDesc
	_ = d214
	var d215 scm.JITValueDesc
	_ = d215
	var d216 scm.JITValueDesc
	_ = d216
	var d217 scm.JITValueDesc
	_ = d217
	var d218 scm.JITValueDesc
	_ = d218
	var d219 scm.JITValueDesc
	_ = d219
	var d220 scm.JITValueDesc
	_ = d220
	var d221 scm.JITValueDesc
	_ = d221
	var d222 scm.JITValueDesc
	_ = d222
	var d223 scm.JITValueDesc
	_ = d223
	var d224 scm.JITValueDesc
	_ = d224
	var d225 scm.JITValueDesc
	_ = d225
	var d226 scm.JITValueDesc
	_ = d226
	var d227 scm.JITValueDesc
	_ = d227
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
	var d251 scm.JITValueDesc
	_ = d251
	var d252 scm.JITValueDesc
	_ = d252
	var d253 scm.JITValueDesc
	_ = d253
	var d254 scm.JITValueDesc
	_ = d254
	var d255 scm.JITValueDesc
	_ = d255
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
	var d273 scm.JITValueDesc
	_ = d273
	var d274 scm.JITValueDesc
	_ = d274
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
	var d281 scm.JITValueDesc
	_ = d281
	var d282 scm.JITValueDesc
	_ = d282
	var d283 scm.JITValueDesc
	_ = d283
	var d284 scm.JITValueDesc
	_ = d284
	var d285 scm.JITValueDesc
	_ = d285
	var d286 scm.JITValueDesc
	_ = d286
	var d287 scm.JITValueDesc
	_ = d287
	var d288 scm.JITValueDesc
	_ = d288
	var d289 scm.JITValueDesc
	_ = d289
	var d290 scm.JITValueDesc
	_ = d290
	var d291 scm.JITValueDesc
	_ = d291
	var d292 scm.JITValueDesc
	_ = d292
	var d293 scm.JITValueDesc
	_ = d293
	var d294 scm.JITValueDesc
	_ = d294
	var d295 scm.JITValueDesc
	_ = d295
	var d296 scm.JITValueDesc
	_ = d296
	var d297 scm.JITValueDesc
	_ = d297
	var d298 scm.JITValueDesc
	_ = d298
	var d299 scm.JITValueDesc
	_ = d299
	var d300 scm.JITValueDesc
	_ = d300
	var d301 scm.JITValueDesc
	_ = d301
	var d302 scm.JITValueDesc
	_ = d302
	var d303 scm.JITValueDesc
	_ = d303
	var d304 scm.JITValueDesc
	_ = d304
	var d305 scm.JITValueDesc
	_ = d305
	var d307 scm.JITValueDesc
	_ = d307
	var d308 scm.JITValueDesc
	_ = d308
	var d309 scm.JITValueDesc
	_ = d309
	var d310 scm.JITValueDesc
	_ = d310
	var d311 scm.JITValueDesc
	_ = d311
	var d312 scm.JITValueDesc
	_ = d312
	var d313 scm.JITValueDesc
	_ = d313
	var d314 scm.JITValueDesc
	_ = d314
	var d315 scm.JITValueDesc
	_ = d315
	var d316 scm.JITValueDesc
	_ = d316
	var d317 scm.JITValueDesc
	_ = d317
	var d318 scm.JITValueDesc
	_ = d318
	var d319 scm.JITValueDesc
	_ = d319
	var d320 scm.JITValueDesc
	_ = d320
	var d322 scm.JITValueDesc
	_ = d322
	var d324 scm.JITValueDesc
	_ = d324
	var d326 scm.JITValueDesc
	_ = d326
	var d327 scm.JITValueDesc
	_ = d327
	var d328 scm.JITValueDesc
	_ = d328
	var d329 scm.JITValueDesc
	_ = d329
	var d330 scm.JITValueDesc
	_ = d330
	var d331 scm.JITValueDesc
	_ = d331
	var d332 scm.JITValueDesc
	_ = d332
	var d333 scm.JITValueDesc
	_ = d333
	var d334 scm.JITValueDesc
	_ = d334
	var d335 scm.JITValueDesc
	_ = d335
	var d336 scm.JITValueDesc
	_ = d336
	var d337 scm.JITValueDesc
	_ = d337
	var d338 scm.JITValueDesc
	_ = d338
	var d339 scm.JITValueDesc
	_ = d339
	var d340 scm.JITValueDesc
	_ = d340
	var d341 scm.JITValueDesc
	_ = d341
	var d342 scm.JITValueDesc
	_ = d342
	var d343 scm.JITValueDesc
	_ = d343
	var d344 scm.JITValueDesc
	_ = d344
	var d345 scm.JITValueDesc
	_ = d345
	var d346 scm.JITValueDesc
	_ = d346
	var d347 scm.JITValueDesc
	_ = d347
	var d348 scm.JITValueDesc
	_ = d348
	var d349 scm.JITValueDesc
	_ = d349
	var d350 scm.JITValueDesc
	_ = d350
	var d351 scm.JITValueDesc
	_ = d351
	var d352 scm.JITValueDesc
	_ = d352
	var d353 scm.JITValueDesc
	_ = d353
	var d354 scm.JITValueDesc
	_ = d354
	var d355 scm.JITValueDesc
	_ = d355
	var d356 scm.JITValueDesc
	_ = d356
	var d357 scm.JITValueDesc
	_ = d357
	var d358 scm.JITValueDesc
	_ = d358
	var d359 scm.JITValueDesc
	_ = d359
	var d360 scm.JITValueDesc
	_ = d360
	var d361 scm.JITValueDesc
	_ = d361
	var d362 scm.JITValueDesc
	_ = d362
	var d363 scm.JITValueDesc
	_ = d363
	var d364 scm.JITValueDesc
	_ = d364
	var d365 scm.JITValueDesc
	_ = d365
	var d366 scm.JITValueDesc
	_ = d366
	var d367 scm.JITValueDesc
	_ = d367
	var d368 scm.JITValueDesc
	_ = d368
	var d369 scm.JITValueDesc
	_ = d369
	var d370 scm.JITValueDesc
	_ = d370
	var d371 scm.JITValueDesc
	_ = d371
	var d372 scm.JITValueDesc
	_ = d372
	var d373 scm.JITValueDesc
	_ = d373
	var d374 scm.JITValueDesc
	_ = d374
	var d375 scm.JITValueDesc
	_ = d375
	var d376 scm.JITValueDesc
	_ = d376
	var d377 scm.JITValueDesc
	_ = d377
	var d378 scm.JITValueDesc
	_ = d378
	var d379 scm.JITValueDesc
	_ = d379
	var d380 scm.JITValueDesc
	_ = d380
	var d381 scm.JITValueDesc
	_ = d381
	var d382 scm.JITValueDesc
	_ = d382
	var d384 scm.JITValueDesc
	_ = d384
	var d385 scm.JITValueDesc
	_ = d385
	var d386 scm.JITValueDesc
	_ = d386
	var d387 scm.JITValueDesc
	_ = d387
	var d388 scm.JITValueDesc
	_ = d388
	var d389 scm.JITValueDesc
	_ = d389
	var d390 scm.JITValueDesc
	_ = d390
	var d392 scm.JITValueDesc
	_ = d392
	var d393 scm.JITValueDesc
	_ = d393
	var d394 scm.JITValueDesc
	_ = d394
	var d395 scm.JITValueDesc
	_ = d395
	var d396 scm.JITValueDesc
	_ = d396
	var d397 scm.JITValueDesc
	_ = d397
	var d398 scm.JITValueDesc
	_ = d398
	var d400 scm.JITValueDesc
	_ = d400
	var d402 scm.JITValueDesc
	_ = d402
	var d404 scm.JITValueDesc
	_ = d404
	var phiBase406 int32
	_ = phiBase406
	var d407 scm.JITValueDesc
	_ = d407
	var d408 scm.JITValueDesc
	_ = d408
	var d410 scm.JITValueDesc
	_ = d410
	var inlineResultOff409 int32
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
	var d422 scm.JITValueDesc
	_ = d422
	var d423 scm.JITValueDesc
	_ = d423
	var d424 scm.JITValueDesc
	_ = d424
	var d425 scm.JITValueDesc
	_ = d425
	var d426 scm.JITValueDesc
	_ = d426
	var d428 scm.JITValueDesc
	_ = d428
	var d429 scm.JITValueDesc
	_ = d429
	var d430 scm.JITValueDesc
	_ = d430
	var d431 scm.JITValueDesc
	_ = d431
	var d432 scm.JITValueDesc
	_ = d432
	var d433 scm.JITValueDesc
	_ = d433
	var d434 scm.JITValueDesc
	_ = d434
	var d435 scm.JITValueDesc
	_ = d435
	var d437 scm.JITValueDesc
	_ = d437
	var d438 scm.JITValueDesc
	_ = d438
	var d440 scm.JITValueDesc
	_ = d440
	var d441 scm.JITValueDesc
	_ = d441
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
	var bbs [3]scm.BBDescriptor
	var storageIdxIntHome scm.JITValueDesc
	storageInputsProtected := ctx.StorageInputsInRegisters
	storageIdxIntHome = idxInt
	if storageInputsProtected {
		ctx.ProtectDescRegisters(&storageIdxIntHome)
	} else {
		ctx.StabilizeDescForControlFlow(&storageIdxIntHome)
	}
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
		idxInt = storageIdxIntHome
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).bufferHandles)
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).bufferHandles))
			r0 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r0, thisptr.Reg, off)
			d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			ctx.BindReg(r0, &d0)
		}
		d1 = d0
		ctx.EnsureDesc(&d1)
		if d1.Loc != scm.LocImm && d1.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d1.Loc == scm.LocImm {
			if d1.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitCmpRegImm32(d1.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap2 := d0
			snap3 := d1
			alloc4 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc4)
			d0 = snap2
			d1 = snap3
		}
		if !bbs[1].Rendered {
			return bbs[1].Render()
		}
		return result
		ctx.FreeDesc(&d0)
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
		idxInt = storageIdxIntHome
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d5 = idxInt
		_ = d5
		ctx.StabilizeDescForControlFlow(&d5)
		inlineHome6 := d5
		inlineHome6.ID = 0
		phiBase7 = ctx.AllocStack(int32(16))
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		_ = d8
		inlineResultOff9 = ctx.AllocStack(int32(16))
		d10 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: inlineResultOff9}
		lbl4 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl5 := ctx.ReserveLabel()
		_ = lbl5
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		lbl6 := ctx.ReserveLabel()
		_ = lbl6
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		lbl7 := ctx.ReserveLabel()
		_ = lbl7
		bbpos_1_3 := int32(-1)
		_ = bbpos_1_3
		lbl8 := ctx.ReserveLabel()
		_ = lbl8
		bbpos_1_4 := int32(-1)
		_ = bbpos_1_4
		lbl9 := ctx.ReserveLabel()
		_ = lbl9
		bbpos_1_5 := int32(-1)
		_ = bbpos_1_5
		lbl10 := ctx.ReserveLabel()
		_ = lbl10
		bbpos_1_6 := int32(-1)
		_ = bbpos_1_6
		lbl11 := ctx.ReserveLabel()
		_ = lbl11
		bbpos_1_7 := int32(-1)
		_ = bbpos_1_7
		lbl12 := ctx.ReserveLabel()
		_ = lbl12
		bbpos_1_8 := int32(-1)
		_ = bbpos_1_8
		lbl13 := ctx.ReserveLabel()
		_ = lbl13
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
			r1 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r1, thisptr.Reg, off)
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.BindReg(r1, &d11)
		}
		ctx.ReclaimUntrackedRegs()
		d12 = d11
		ctx.EnsureDesc(&d12)
		if d12.Loc != scm.LocImm && d12.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		if d12.Loc == scm.LocImm {
			if d12.Imm.Bool() {
				ctx.MarkLabel(lbl14)
				ctx.EmitJmp(lbl6)
			} else {
				ctx.MarkLabel(lbl15)
				ctx.EmitJmp(lbl8)
			}
		} else {
			ctx.EmitCmpRegImm32(d12.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl14)
			ctx.EmitJmp(lbl15)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d11)
		bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl8)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d13 = d5
		_ = d13
		ctx.StabilizeDescForControlFlow(&d5)
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl16 := ctx.ReserveLabel()
		_ = lbl16
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl16)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 48)
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r2, thisptr.Reg, off)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d14)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d14)
		if d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d14.Imm.Int()))))}
		} else {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegReg(r3, d14.Reg)
			ctx.EmitShlRegImm8(r3, 56)
			ctx.EmitShrRegImm8(r3, 56)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d15)
		}
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d13)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d15)
		ctx.EnsureDescsTogether(&d13, &d15)
		if d13.Loc == scm.LocImm && d15.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() * d15.Imm.Int())}
		} else if d13.Loc == scm.LocImm {
			ctx.EnsureDesc(&d15)
			scratch := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegReg(scratch, d15.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d13.Imm.Int())
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d15.Loc == scm.LocImm {
			ctx.EnsureDesc(&d13)
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(scratch, d13.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d15.Imm.Int())
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else {
			ctx.EnsureDesc(&d13)
			ctx.SyncDesc(&d15)
			r4 := ctx.AllocRegExcept(d13.Reg, d15.Reg)
			ctx.EmitMovRegReg(r4, d13.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r4, &d15)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d17)
		}
		if d17.Loc == scm.LocReg && d13.Loc == scm.LocReg && d17.Reg == d13.Reg {
			ctx.TransferReg(d13.Reg)
			d13.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		if d17.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() / 64)}
		} else {
			r5 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r5, d17.Reg)
			ctx.EmitShrRegImm8(r5, 6)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d18)
		}
		if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		if d17.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() % 64)}
		} else {
			r6 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r6, d17.Reg)
			ctx.EmitAndRegImm32(r6, 63)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d19)
		}
		if d19.Loc == scm.LocReg && d17.Loc == scm.LocReg && d19.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d20 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r7 := ctx.AllocReg()
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+16)
			d20 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r7, Reg2: r8, Reg3: r9}
			ctx.BindReg(r7, &d20)
			ctx.BindReg(r8, &d20)
			ctx.BindReg(r9, &d20)
			ctx.BindReg(r7, &d20)
			ctx.BindReg(r8, &d20)
			ctx.BindReg(r9, &d20)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		d21 = ctx.EmitLoadScalarSliceElement(&d20, &d18, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d21, &d19)
		if d21.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d21.Imm.Int()) << uint64(d19.Imm.Int())))}
		} else if d19.Loc == scm.LocImm {
			r10 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r10, d21.Reg)
			ctx.EmitShlRegImm8(r10, uint8(d19.Imm.Int()))
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d22)
		} else {
			shiftSrc := d21.Reg
			r11 := ctx.AllocRegExcept(d21.Reg, d19.Reg)
			ctx.EmitMovRegReg(r11, d21.Reg)
			shiftSrc = r11
			ctx.EmitShiftLeft(shiftSrc, d19.Reg, true)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d22)
		}
		if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
			ctx.TransferReg(d21.Reg)
			d21.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d21)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d18)
		if d18.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegReg(scratch, d18.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d23)
		}
		if d23.Loc == scm.LocReg && d18.Loc == scm.LocReg && d23.Reg == d18.Reg {
			ctx.TransferReg(d18.Reg)
			d18.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		d24 = ctx.EmitLoadScalarSliceElement(&d20, &d23, 8, scm.TagInt)
		ctx.FreeDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d25, &d19)
		if d25.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d19.Imm.Int())}
		} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
			ctx.EnsureDesc(&d25)
			r12 := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(r12, d25.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d26)
		} else if d25.Loc == scm.LocImm {
			ctx.EnsureDesc(&d19)
			scratch := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d19)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else if d19.Loc == scm.LocImm {
			ctx.EnsureDesc(&d25)
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(scratch, d25.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d19.Imm.Int())
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else {
			ctx.EnsureDesc(&d25)
			ctx.SyncDesc(&d19)
			r13 := ctx.AllocRegExcept(d25.Reg, d19.Reg)
			ctx.EmitMovRegReg(r13, d25.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r13, &d19)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d26)
		}
		if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d19)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d24, &d26)
		if d24.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d26.Imm.Int())))}
		} else if d26.Loc == scm.LocImm {
			r14 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r14, d24.Reg)
			ctx.EmitShrRegImm8(r14, uint8(d26.Imm.Int()))
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d27)
		} else {
			shiftSrc := d24.Reg
			r15 := ctx.AllocRegExcept(d24.Reg, d26.Reg)
			ctx.EmitMovRegReg(r15, d24.Reg)
			shiftSrc = r15
			ctx.EmitShiftRight(shiftSrc, d26.Reg, false)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d27)
		}
		if d27.Loc == scm.LocReg && d24.Loc == scm.LocReg && d27.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d27)
		if d22.Loc == scm.LocImm && d27.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() | d27.Imm.Int())}
		} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
			ctx.BindReg(d27.Reg, &d28)
		} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
			r16 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(r16, d22.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d28)
		} else if d22.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
			ctx.EmitOrInt64(scratch, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else if d27.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(r17, d22.Reg)
			if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r17, int32(d27.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d27.Imm.Int()))
				ctx.EmitOrInt64(r17, ctx.ScratchReg)
			}
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d28)
		} else {
			r18 := ctx.AllocRegExcept(d22.Reg, d27.Reg)
			ctx.EmitMovRegReg(r18, d22.Reg)
			ctx.EmitOrInt64(r18, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d28)
		}
		if d28.Loc == scm.LocReg && d22.Loc == scm.LocReg && d28.Reg == d22.Reg {
			ctx.TransferReg(d22.Reg)
			d22.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d22)
		ctx.FreeDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d15)
		ctx.EnsureDescsTogether(&d29, &d15)
		if d29.Loc == scm.LocImm && d15.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() - d15.Imm.Int())}
		} else if d15.Loc == scm.LocImm && d15.Imm.Int() == 0 {
			ctx.EnsureDesc(&d29)
			r19 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r19, d29.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d30)
		} else if d29.Loc == scm.LocImm {
			ctx.EnsureDesc(&d15)
			scratch := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d15)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else if d15.Loc == scm.LocImm {
			ctx.EnsureDesc(&d29)
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(scratch, d29.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d15.Imm.Int())
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else {
			ctx.EnsureDesc(&d29)
			ctx.SyncDesc(&d15)
			r20 := ctx.AllocRegExcept(d29.Reg, d15.Reg)
			ctx.EmitMovRegReg(r20, d29.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r20, &d15)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d30)
		}
		if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
			ctx.TransferReg(d29.Reg)
			d29.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d15)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d30)
		ctx.EnsureDescsTogether(&d28, &d30)
		if d28.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) >> uint64(d30.Imm.Int())))}
		} else if d30.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r21, d28.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d30.Imm.Int()))
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d31)
		} else {
			shiftSrc := d28.Reg
			r22 := ctx.AllocRegExcept(d28.Reg, d30.Reg)
			ctx.EmitMovRegReg(r22, d28.Reg)
			shiftSrc = r22
			ctx.EmitShiftRight(shiftSrc, d30.Reg, false)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d31)
		}
		if d31.Loc == scm.LocReg && d28.Loc == scm.LocReg && d31.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.StabilizeDescForControlFlow(&d31)
		inlineHome32 := d31
		inlineHome32.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 80)
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r23, thisptr.Reg, off)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d33)
		}
		ctx.ReclaimUntrackedRegs()
		d34 = d33
		ctx.EnsureDesc(&d34)
		if d34.Loc != scm.LocImm && d34.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl17 := ctx.ReserveLabel()
		lbl18 := ctx.ReserveLabel()
		if d34.Loc == scm.LocImm {
			if d34.Imm.Bool() {
				ctx.MarkLabel(lbl17)
				ctx.EmitJmp(lbl13)
			} else {
				ctx.MarkLabel(lbl18)
				ctx.EmitJmp(lbl12)
			}
		} else {
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl12)
		}
		ctx.FreeDesc(&d33)
		bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		if d31.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d31.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d31.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d35)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
			r25 := ctx.AllocReg()
			ctx.EmitMovRegMem(r25, thisptr.Reg, off)
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d36)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d36)
		ctx.EnsureDescsTogether(&d35, &d36)
		if d35.Loc == scm.LocImm && d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + d36.Imm.Int())}
		} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
			ctx.EnsureDesc(&d35)
			r26 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r26, d35.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d37)
		} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
			ctx.EnsureDesc(&d36)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			ctx.BindReg(d36.Reg, &d37)
		} else if d35.Loc == scm.LocImm {
			ctx.EnsureDesc(&d36)
			scratch := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(scratch, d36.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d35.Imm.Int())
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else if d36.Loc == scm.LocImm {
			ctx.EnsureDesc(&d35)
			scratch := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(scratch, d35.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d36.Imm.Int())
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else {
			ctx.EnsureDesc(&d35)
			ctx.SyncDesc(&d36)
			r27 := ctx.AllocRegExcept(d35.Reg, d36.Reg)
			ctx.EmitMovRegReg(r27, d35.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r27, &d36)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d37)
		}
		if d37.Loc == scm.LocReg && d35.Loc == scm.LocReg && d37.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.FreeDesc(&d36)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d37)
		if d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d37.Imm.Int()))))}
		} else {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegReg(r28, d37.Reg)
			ctx.EmitShlRegImm8(r28, 32)
			ctx.EmitShrRegImm8(r28, 32)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d38)
		}
		ctx.StabilizeDescForControlFlow(&d38)
		inlineHome39 := d38
		inlineHome39.ID = 0
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d38)
		if d38.Loc == scm.LocReg || d38.Loc == scm.LocFPReg {
			ctx.ProtectReg(d38.Reg)
		} else if d38.Loc == scm.LocRegPair {
			ctx.ProtectReg(d38.Reg)
			ctx.ProtectReg(d38.Reg2)
		}
		d40 = d38
		if d40.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d40)
		ctx.EmitStoreToStack(d40, int32(phiBase7)+int32(0))
		if d38.Loc == scm.LocReg || d38.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d38.Reg)
		} else if d38.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d38.Reg)
			ctx.UnprotectReg(d38.Reg2)
		}
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).textBuffer)
			r29 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r29, fieldAddr)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d41)
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).textBuffer))
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMem(r30, thisptr.Reg, off)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d41)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		d42 = d8
		_ = d42
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl19 := ctx.ReserveLabel()
		_ = lbl19
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl19)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 48)
			r31 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r31, thisptr.Reg, off)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d43)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		if d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegReg(r32, d43.Reg)
			ctx.EmitShlRegImm8(r32, 56)
			ctx.EmitShrRegImm8(r32, 56)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d44)
		}
		ctx.FreeDesc(&d43)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d42)
		if d42.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d42.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d42.Reg)
			ctx.EmitShlRegImm8(r33, 32)
			ctx.EmitShrRegImm8(r33, 32)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d45)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d45, &d44)
		if d45.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() * d44.Imm.Int())}
		} else if d45.Loc == scm.LocImm {
			ctx.EnsureDesc(&d44)
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(scratch, d44.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d45.Imm.Int())
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else if d44.Loc == scm.LocImm {
			ctx.EnsureDesc(&d45)
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(scratch, d45.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d44.Imm.Int())
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else {
			ctx.EnsureDesc(&d45)
			ctx.SyncDesc(&d44)
			r34 := ctx.AllocRegExcept(d45.Reg, d44.Reg)
			ctx.EmitMovRegReg(r34, d45.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r34, &d44)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d46)
		}
		if d46.Loc == scm.LocReg && d45.Loc == scm.LocReg && d46.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		if d46.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() / 64)}
		} else {
			r35 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r35, d46.Reg)
			ctx.EmitShrRegImm8(r35, 6)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d47)
		}
		if d47.Loc == scm.LocReg && d46.Loc == scm.LocReg && d47.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		if d46.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() % 64)}
		} else {
			r36 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r36, d46.Reg)
			ctx.EmitAndRegImm32(r36, 63)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d48)
		}
		if d48.Loc == scm.LocReg && d46.Loc == scm.LocReg && d48.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d49 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r37 := ctx.AllocReg()
			r38 := ctx.AllocRegExcept(r37)
			r39 := ctx.AllocRegExcept(r37, r38)
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
			ctx.EmitMovRegMem(r37, thisptr.Reg, off)
			ctx.EmitMovRegMem(r38, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r39, thisptr.Reg, off+16)
			d49 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r37, Reg2: r38, Reg3: r39}
			ctx.BindReg(r37, &d49)
			ctx.BindReg(r38, &d49)
			ctx.BindReg(r39, &d49)
			ctx.BindReg(r37, &d49)
			ctx.BindReg(r38, &d49)
			ctx.BindReg(r39, &d49)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d50 = ctx.EmitLoadScalarSliceElement(&d49, &d47, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d50)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d50, &d48)
		if d50.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) << uint64(d48.Imm.Int())))}
		} else if d48.Loc == scm.LocImm {
			r40 := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(r40, d50.Reg)
			ctx.EmitShlRegImm8(r40, uint8(d48.Imm.Int()))
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d51)
		} else {
			shiftSrc := d50.Reg
			r41 := ctx.AllocRegExcept(d50.Reg, d48.Reg)
			ctx.EmitMovRegReg(r41, d50.Reg)
			shiftSrc = r41
			ctx.EmitShiftLeft(shiftSrc, d48.Reg, true)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d51)
		}
		if d51.Loc == scm.LocReg && d50.Loc == scm.LocReg && d51.Reg == d50.Reg {
			ctx.TransferReg(d50.Reg)
			d50.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d47)
		if d47.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(scratch, d47.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		}
		if d52.Loc == scm.LocReg && d47.Loc == scm.LocReg && d52.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		d53 = ctx.EmitLoadScalarSliceElement(&d49, &d52, 8, scm.TagInt)
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d54, &d48)
		if d54.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() - d48.Imm.Int())}
		} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
			ctx.EnsureDesc(&d54)
			r42 := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(r42, d54.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d55)
		} else if d54.Loc == scm.LocImm {
			ctx.EnsureDesc(&d48)
			scratch := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d48)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else if d48.Loc == scm.LocImm {
			ctx.EnsureDesc(&d54)
			scratch := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(scratch, d54.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d48.Imm.Int())
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else {
			ctx.EnsureDesc(&d54)
			ctx.SyncDesc(&d48)
			r43 := ctx.AllocRegExcept(d54.Reg, d48.Reg)
			ctx.EmitMovRegReg(r43, d54.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r43, &d48)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d55)
		}
		if d55.Loc == scm.LocReg && d54.Loc == scm.LocReg && d55.Reg == d54.Reg {
			ctx.TransferReg(d54.Reg)
			d54.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&d53, &d55)
		if d53.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d55.Imm.Int())))}
		} else if d55.Loc == scm.LocImm {
			r44 := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(r44, d53.Reg)
			ctx.EmitShrRegImm8(r44, uint8(d55.Imm.Int()))
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d56)
		} else {
			shiftSrc := d53.Reg
			r45 := ctx.AllocRegExcept(d53.Reg, d55.Reg)
			ctx.EmitMovRegReg(r45, d53.Reg)
			shiftSrc = r45
			ctx.EmitShiftRight(shiftSrc, d55.Reg, false)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d56)
		}
		if d56.Loc == scm.LocReg && d53.Loc == scm.LocReg && d56.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d53)
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d56)
		if d51.Loc == scm.LocImm && d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() | d56.Imm.Int())}
		} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
			ctx.BindReg(d56.Reg, &d57)
		} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
			r46 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r46, d51.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d57)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitOrInt64(scratch, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d56.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r47, d51.Reg)
			if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r47, int32(d56.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d56.Imm.Int()))
				ctx.EmitOrInt64(r47, ctx.ScratchReg)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d57)
		} else {
			r48 := ctx.AllocRegExcept(d51.Reg, d56.Reg)
			ctx.EmitMovRegReg(r48, d51.Reg)
			ctx.EmitOrInt64(r48, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d57)
		}
		if d57.Loc == scm.LocReg && d51.Loc == scm.LocReg && d57.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d51)
		ctx.FreeDesc(&d56)
		ctx.ReclaimUntrackedRegs()
		d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d58, &d44)
		if d58.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() - d44.Imm.Int())}
		} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
			ctx.EnsureDesc(&d58)
			r49 := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(r49, d58.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d59)
		} else if d58.Loc == scm.LocImm {
			ctx.EnsureDesc(&d44)
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d44)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d59)
		} else if d44.Loc == scm.LocImm {
			ctx.EnsureDesc(&d58)
			scratch := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(scratch, d58.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d44.Imm.Int())
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d59)
		} else {
			ctx.EnsureDesc(&d58)
			ctx.SyncDesc(&d44)
			r50 := ctx.AllocRegExcept(d58.Reg, d44.Reg)
			ctx.EmitMovRegReg(r50, d58.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r50, &d44)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d59)
		}
		if d59.Loc == scm.LocReg && d58.Loc == scm.LocReg && d59.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d57, &d59)
		if d57.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) >> uint64(d59.Imm.Int())))}
		} else if d59.Loc == scm.LocImm {
			r51 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r51, d57.Reg)
			ctx.EmitShrRegImm8(r51, uint8(d59.Imm.Int()))
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d60)
		} else {
			shiftSrc := d57.Reg
			r52 := ctx.AllocRegExcept(d57.Reg, d59.Reg)
			ctx.EmitMovRegReg(r52, d57.Reg)
			shiftSrc = r52
			ctx.EmitShiftRight(shiftSrc, d59.Reg, false)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d60)
		}
		if d60.Loc == scm.LocReg && d57.Loc == scm.LocReg && d60.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d57)
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d60)
		ctx.FreeDesc(&d8)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d60)
		ctx.EnsureDesc(&d60)
		if d60.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d60.Imm.Int()))))}
		} else {
			r53 := ctx.AllocReg()
			ctx.EmitMovRegReg(r53, d60.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d61)
		}
		ctx.FreeDesc(&d60)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
			r54 := ctx.AllocReg()
			ctx.EmitMovRegMem(r54, thisptr.Reg, off)
			d62 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
			ctx.BindReg(r54, &d62)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d61)
		ctx.EnsureDesc(&d62)
		ctx.EnsureDescsTogether(&d61, &d62)
		if d61.Loc == scm.LocImm && d62.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() + d62.Imm.Int())}
		} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
			ctx.EnsureDesc(&d61)
			r55 := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitMovRegReg(r55, d61.Reg)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d63)
		} else if d61.Loc == scm.LocImm && d61.Imm.Int() == 0 {
			ctx.EnsureDesc(&d62)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d62.Reg}
			ctx.BindReg(d62.Reg, &d63)
		} else if d61.Loc == scm.LocImm {
			ctx.EnsureDesc(&d62)
			scratch := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegReg(scratch, d62.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d61.Imm.Int())
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d63)
		} else if d62.Loc == scm.LocImm {
			ctx.EnsureDesc(&d61)
			scratch := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitMovRegReg(scratch, d61.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d62.Imm.Int())
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d63)
		} else {
			ctx.EnsureDesc(&d61)
			ctx.SyncDesc(&d62)
			r56 := ctx.AllocRegExcept(d61.Reg, d62.Reg)
			ctx.EmitMovRegReg(r56, d61.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r56, &d62)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d63)
		}
		if d63.Loc == scm.LocReg && d61.Loc == scm.LocReg && d63.Reg == d61.Reg {
			ctx.TransferReg(d61.Reg)
			d61.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d61)
		ctx.FreeDesc(&d62)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d63)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d63)
		d65 = d41
		_ = d65
		ctx.StabilizeDescForControlFlow(&d65)
		inlineHome66 := d65
		inlineHome66.ID = 0
		d67 = d63
		_ = d67
		ctx.StabilizeDescForControlFlow(&d67)
		inlineHome68 := d67
		inlineHome68.ID = 0
		inlineResultOff69 = ctx.AllocStack(int32(16))
		d70 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: inlineResultOff69}
		lbl20 := ctx.ReserveLabel()
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl21 := ctx.ReserveLabel()
		_ = lbl21
		bbpos_4_1 := int32(-1)
		_ = bbpos_4_1
		lbl22 := ctx.ReserveLabel()
		_ = lbl22
		bbpos_4_2 := int32(-1)
		_ = bbpos_4_2
		lbl23 := ctx.ReserveLabel()
		_ = lbl23
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl21)
		ctx.ResolveFixups()
		d65 = inlineHome66
		d67 = inlineHome68
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d67)
		if d67.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d67.Imm.Int()) > uint64(0xffffffffffffff))}
		} else {
			r57 := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegImm64(ctx.ScratchReg, 0xffffffffffffff)
			ctx.EmitCmpInt64(d67.Reg, ctx.ScratchReg)
			d71 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r57, Condition: scm.CondUnsignedAbove}
			ctx.BindReg(r57, &d71)
		}
		ctx.ReclaimUntrackedRegs()
		d72 = d71
		ctx.SyncDesc(&d72)
		if d72.Loc != scm.LocImm && d72.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		if d72.Loc == scm.LocImm {
			if d72.Imm.Bool() {
				ctx.MarkLabel(lbl24)
				ctx.EmitJmp(lbl22)
			} else {
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
			}
		} else {
			ctx.EmitJump(d72.Condition, lbl24)
			ctx.EmitJmp(lbl25)
			ctx.FreeDesc(&d71)
			ctx.MarkLabel(lbl24)
			ctx.EmitJmp(lbl22)
			ctx.MarkLabel(lbl25)
			ctx.EmitJmp(lbl23)
		}
		bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		d65 = inlineHome66
		d67 = inlineHome68
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r58 := ctx.AllocReg()
		r59 := ctx.AllocRegExcept(r58)
		ctx.EmitMovRegImm64(r58, 0)
		ctx.EmitMovRegImm64(r59, 0)
		d73 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r58, Reg2: r59}
		ctx.BindReg(r58, &d73)
		ctx.BindReg(r59, &d73)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.EnsureDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.EnsureDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(26)}
		ctx.EnsureDesc(&d67)
		d77 = d67
		_ = d77
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl26 := ctx.ReserveLabel()
		_ = lbl26
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl26)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d77)
		if d77.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) << 8))}
		} else {
			r60 := ctx.AllocRegExcept(d77.Reg)
			ctx.EmitMovRegReg(r60, d77.Reg)
			ctx.EmitShlRegImm8(r60, 8)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d78)
		}
		if d78.Loc == scm.LocReg && d77.Loc == scm.LocReg && d78.Reg == d77.Reg {
			ctx.TransferReg(d77.Reg)
			d77.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(26)}
		ctx.ReclaimUntrackedRegs()
		d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(26)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d78)
		d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(26)}
		if d78.Loc == scm.LocImm && d82.Loc == scm.LocImm {
			d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() | d82.Imm.Int())}
		} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d82.Reg}
			ctx.BindReg(d82.Reg, &d81)
		} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
			r61 := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitMovRegReg(r61, d78.Reg)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			ctx.BindReg(r61, &d81)
		} else if d78.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d82.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
			ctx.EmitOrInt64(scratch, d82.Reg)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d81)
		} else if d82.Loc == scm.LocImm {
			r62 := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitMovRegReg(r62, d78.Reg)
			if d82.Imm.Int() >= -2147483648 && d82.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r62, int32(d82.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d82.Imm.Int()))
				ctx.EmitOrInt64(r62, ctx.ScratchReg)
			}
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d81)
		} else {
			r63 := ctx.AllocRegExcept(d78.Reg, d82.Reg)
			ctx.EmitMovRegReg(r63, d78.Reg)
			ctx.EmitOrInt64(r63, d82.Reg)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d81)
		}
		if d81.Loc == scm.LocReg && d78.Loc == scm.LocReg && d81.Reg == d78.Reg {
			ctx.TransferReg(d78.Reg)
			d78.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d78)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d81)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.EnsureDescsTogether(&d73, &d65)
		ctx.EmitMovToReg(d73.Reg, d65)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d81)
		ctx.EnsureDescsTogether(&d73, &d81)
		ctx.EmitMovToReg(d73.Reg2, d81)
		ctx.FreeDesc(&d81)
		ctx.ReclaimUntrackedRegs()
		d83 = d73
		_ = d83
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d83)
		ctx.EmitCopyScmerToDesc(&d70, &d83)
		ctx.EmitJmp(lbl20)
		bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl22)
		ctx.ResolveFixups()
		d65 = inlineHome66
		d67 = inlineHome68
		ctx.ReclaimUntrackedRegs()
		ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
		ctx.MarkLabel(lbl20)
		ctx.FreeDesc(&d41)
		ctx.FreeDesc(&d63)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d70)
		ctx.EmitCopyScmerToDesc(&d10, &d70)
		ctx.EmitJmp(lbl4)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl6)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 80)
			r64 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r64, thisptr.Reg, off)
			d84 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
			ctx.BindReg(r64, &d84)
		}
		ctx.ReclaimUntrackedRegs()
		d85 = d84
		ctx.EnsureDesc(&d85)
		if d85.Loc != scm.LocImm && d85.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl27 := ctx.ReserveLabel()
		lbl28 := ctx.ReserveLabel()
		if d85.Loc == scm.LocImm {
			if d85.Imm.Bool() {
				ctx.MarkLabel(lbl27)
				ctx.EmitJmp(lbl10)
			} else {
				ctx.MarkLabel(lbl28)
				ctx.SyncDesc(&d5)
				if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
					ctx.ProtectReg(d5.Reg)
				} else if d5.Loc == scm.LocRegPair {
					ctx.ProtectReg(d5.Reg)
					ctx.ProtectReg(d5.Reg2)
				}
				d86 = d5
				if d86.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d86)
				ctx.EmitStoreToStack(d86, int32(phiBase7)+int32(0))
				if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d5.Reg)
				} else if d5.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d5.Reg)
					ctx.UnprotectReg(d5.Reg2)
				}
				ctx.EmitJmp(lbl7)
			}
		} else {
			ctx.EmitCmpRegImm32(d85.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl27)
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl28)
			ctx.SyncDesc(&d5)
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			d87 = d5
			if d87.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d87)
			ctx.EmitStoreToStack(d87, int32(phiBase7)+int32(0))
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			ctx.EmitJmp(lbl7)
		}
		ctx.FreeDesc(&d84)
		bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 88)
			r65 := ctx.AllocReg()
			ctx.EmitMovRegMem(r65, thisptr.Reg, off)
			d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r65}
			ctx.BindReg(r65, &d88)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d88)
		ctx.EnsureDescsTogether(&d31, &d88)
		if d31.Loc == scm.LocImm && d88.Loc == scm.LocImm {
			d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d31.Imm.Int()) == uint64(d88.Imm.Int()))}
		} else if d88.Loc == scm.LocImm {
			r66 := ctx.AllocRegExcept(d31.Reg)
			if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d31.Reg, int32(d88.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d88.Imm.Int()))
				ctx.EmitCmpInt64(d31.Reg, ctx.ScratchReg)
			}
			d89 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r66, Condition: scm.CondEqual}
			ctx.BindReg(r66, &d89)
		} else if d31.Loc == scm.LocImm {
			r67 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d31.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d88.Reg)
			d89 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r67, Condition: scm.CondEqual}
			ctx.BindReg(r67, &d89)
		} else {
			r68 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitCmpInt64(d31.Reg, d88.Reg)
			d89 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r68, Condition: scm.CondEqual}
			ctx.BindReg(r68, &d89)
		}
		ctx.FreeDesc(&d88)
		ctx.ReclaimUntrackedRegs()
		d90 = d89
		ctx.SyncDesc(&d90)
		if d90.Loc != scm.LocImm && d90.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl29 := ctx.ReserveLabel()
		lbl30 := ctx.ReserveLabel()
		if d90.Loc == scm.LocImm {
			if d90.Imm.Bool() {
				ctx.MarkLabel(lbl29)
				ctx.EmitJmp(lbl11)
			} else {
				ctx.MarkLabel(lbl30)
				ctx.EmitJmp(lbl12)
			}
		} else {
			ctx.EmitJump(d90.Condition, lbl29)
			ctx.EmitJmp(lbl30)
			ctx.FreeDesc(&d89)
			ctx.MarkLabel(lbl29)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl12)
		}
		bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl10)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d91 = d5
		_ = d91
		ctx.StabilizeDescForControlFlow(&d5)
		bbpos_6_0 := int32(-1)
		_ = bbpos_6_0
		lbl31 := ctx.ReserveLabel()
		_ = lbl31
		bbpos_6_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl31)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 48)
			r69 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r69, thisptr.Reg, off)
			d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
			ctx.BindReg(r69, &d92)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d92)
		ctx.EnsureDesc(&d92)
		if d92.Loc == scm.LocImm {
			d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d92.Imm.Int()))))}
		} else {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegReg(r70, d92.Reg)
			ctx.EmitShlRegImm8(r70, 56)
			ctx.EmitShrRegImm8(r70, 56)
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d93)
		}
		ctx.FreeDesc(&d92)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d91)
		ctx.EnsureDesc(&d91)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d91)
		ctx.EnsureDesc(&d93)
		ctx.EnsureDescsTogether(&d91, &d93)
		if d91.Loc == scm.LocImm && d93.Loc == scm.LocImm {
			d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() * d93.Imm.Int())}
		} else if d91.Loc == scm.LocImm {
			ctx.EnsureDesc(&d93)
			scratch := ctx.AllocRegExcept(d93.Reg)
			ctx.EmitMovRegReg(scratch, d93.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d91.Imm.Int())
			d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d95)
		} else if d93.Loc == scm.LocImm {
			ctx.EnsureDesc(&d91)
			scratch := ctx.AllocRegExcept(d91.Reg)
			ctx.EmitMovRegReg(scratch, d91.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d93.Imm.Int())
			d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d95)
		} else {
			ctx.EnsureDesc(&d91)
			ctx.SyncDesc(&d93)
			r71 := ctx.AllocRegExcept(d91.Reg, d93.Reg)
			ctx.EmitMovRegReg(r71, d91.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r71, &d93)
			d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			ctx.BindReg(r71, &d95)
		}
		if d95.Loc == scm.LocReg && d91.Loc == scm.LocReg && d95.Reg == d91.Reg {
			ctx.TransferReg(d91.Reg)
			d91.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d95)
		if d95.Loc == scm.LocImm {
			d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() / 64)}
		} else {
			r72 := ctx.AllocRegExcept(d95.Reg)
			ctx.EmitMovRegReg(r72, d95.Reg)
			ctx.EmitShrRegImm8(r72, 6)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d96)
		}
		if d96.Loc == scm.LocReg && d95.Loc == scm.LocReg && d96.Reg == d95.Reg {
			ctx.TransferReg(d95.Reg)
			d95.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d95)
		if d95.Loc == scm.LocImm {
			d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() % 64)}
		} else {
			r73 := ctx.AllocRegExcept(d95.Reg)
			ctx.EmitMovRegReg(r73, d95.Reg)
			ctx.EmitAndRegImm32(r73, 63)
			d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d97)
		}
		if d97.Loc == scm.LocReg && d95.Loc == scm.LocReg && d97.Reg == d95.Reg {
			ctx.TransferReg(d95.Reg)
			d95.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d95)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d98 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r74 := ctx.AllocReg()
			r75 := ctx.AllocRegExcept(r74)
			r76 := ctx.AllocRegExcept(r74, r75)
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
			ctx.EmitMovRegMem(r74, thisptr.Reg, off)
			ctx.EmitMovRegMem(r75, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r76, thisptr.Reg, off+16)
			d98 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r74, Reg2: r75, Reg3: r76}
			ctx.BindReg(r74, &d98)
			ctx.BindReg(r75, &d98)
			ctx.BindReg(r76, &d98)
			ctx.BindReg(r74, &d98)
			ctx.BindReg(r75, &d98)
			ctx.BindReg(r76, &d98)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d96)
		ctx.ReclaimUntrackedRegs()
		d99 = ctx.EmitLoadScalarSliceElement(&d98, &d96, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d99)
		ctx.EnsureDesc(&d97)
		ctx.EnsureDescsTogether(&d99, &d97)
		if d99.Loc == scm.LocImm && d97.Loc == scm.LocImm {
			d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d99.Imm.Int()) << uint64(d97.Imm.Int())))}
		} else if d97.Loc == scm.LocImm {
			r77 := ctx.AllocRegExcept(d99.Reg)
			ctx.EmitMovRegReg(r77, d99.Reg)
			ctx.EmitShlRegImm8(r77, uint8(d97.Imm.Int()))
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			ctx.BindReg(r77, &d100)
		} else {
			shiftSrc := d99.Reg
			r78 := ctx.AllocRegExcept(d99.Reg, d97.Reg)
			ctx.EmitMovRegReg(r78, d99.Reg)
			shiftSrc = r78
			ctx.EmitShiftLeft(shiftSrc, d97.Reg, true)
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d100)
		}
		if d100.Loc == scm.LocReg && d99.Loc == scm.LocReg && d100.Reg == d99.Reg {
			ctx.TransferReg(d99.Reg)
			d99.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d99)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d96)
		ctx.EnsureDesc(&d96)
		if d96.Loc == scm.LocImm {
			d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d96.Reg)
			ctx.EmitMovRegReg(scratch, d96.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d101)
		}
		if d101.Loc == scm.LocReg && d96.Loc == scm.LocReg && d101.Reg == d96.Reg {
			ctx.TransferReg(d96.Reg)
			d96.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d96)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d101)
		ctx.ReclaimUntrackedRegs()
		d102 = ctx.EmitLoadScalarSliceElement(&d98, &d101, 8, scm.TagInt)
		ctx.FreeDesc(&d101)
		ctx.ReclaimUntrackedRegs()
		d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d97)
		ctx.EnsureDescsTogether(&d103, &d97)
		if d103.Loc == scm.LocImm && d97.Loc == scm.LocImm {
			d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() - d97.Imm.Int())}
		} else if d97.Loc == scm.LocImm && d97.Imm.Int() == 0 {
			ctx.EnsureDesc(&d103)
			r79 := ctx.AllocRegExcept(d103.Reg)
			ctx.EmitMovRegReg(r79, d103.Reg)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d104)
		} else if d103.Loc == scm.LocImm {
			ctx.EnsureDesc(&d97)
			scratch := ctx.AllocRegExcept(d97.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d97)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d104)
		} else if d97.Loc == scm.LocImm {
			ctx.EnsureDesc(&d103)
			scratch := ctx.AllocRegExcept(d103.Reg)
			ctx.EmitMovRegReg(scratch, d103.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d97.Imm.Int())
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d104)
		} else {
			ctx.EnsureDesc(&d103)
			ctx.SyncDesc(&d97)
			r80 := ctx.AllocRegExcept(d103.Reg, d97.Reg)
			ctx.EmitMovRegReg(r80, d103.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r80, &d97)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d104)
		}
		if d104.Loc == scm.LocReg && d103.Loc == scm.LocReg && d104.Reg == d103.Reg {
			ctx.TransferReg(d103.Reg)
			d103.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d97)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d102)
		ctx.EnsureDesc(&d104)
		ctx.EnsureDescsTogether(&d102, &d104)
		if d102.Loc == scm.LocImm && d104.Loc == scm.LocImm {
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d102.Imm.Int()) >> uint64(d104.Imm.Int())))}
		} else if d104.Loc == scm.LocImm {
			r81 := ctx.AllocRegExcept(d102.Reg)
			ctx.EmitMovRegReg(r81, d102.Reg)
			ctx.EmitShrRegImm8(r81, uint8(d104.Imm.Int()))
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d105)
		} else {
			shiftSrc := d102.Reg
			r82 := ctx.AllocRegExcept(d102.Reg, d104.Reg)
			ctx.EmitMovRegReg(r82, d102.Reg)
			shiftSrc = r82
			ctx.EmitShiftRight(shiftSrc, d104.Reg, false)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d105)
		}
		if d105.Loc == scm.LocReg && d102.Loc == scm.LocReg && d105.Reg == d102.Reg {
			ctx.TransferReg(d102.Reg)
			d102.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d102)
		ctx.FreeDesc(&d104)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d100)
		ctx.EnsureDesc(&d105)
		if d100.Loc == scm.LocImm && d105.Loc == scm.LocImm {
			d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d100.Imm.Int() | d105.Imm.Int())}
		} else if d100.Loc == scm.LocImm && d100.Imm.Int() == 0 {
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d105.Reg}
			ctx.BindReg(d105.Reg, &d106)
		} else if d105.Loc == scm.LocImm && d105.Imm.Int() == 0 {
			r83 := ctx.AllocRegExcept(d100.Reg)
			ctx.EmitMovRegReg(r83, d100.Reg)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d106)
		} else if d100.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d105.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d100.Imm.Int()))
			ctx.EmitOrInt64(scratch, d105.Reg)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d106)
		} else if d105.Loc == scm.LocImm {
			r84 := ctx.AllocRegExcept(d100.Reg)
			ctx.EmitMovRegReg(r84, d100.Reg)
			if d105.Imm.Int() >= -2147483648 && d105.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r84, int32(d105.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d105.Imm.Int()))
				ctx.EmitOrInt64(r84, ctx.ScratchReg)
			}
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			ctx.BindReg(r84, &d106)
		} else {
			r85 := ctx.AllocRegExcept(d100.Reg, d105.Reg)
			ctx.EmitMovRegReg(r85, d100.Reg)
			ctx.EmitOrInt64(r85, d105.Reg)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d106)
		}
		if d106.Loc == scm.LocReg && d100.Loc == scm.LocReg && d106.Reg == d100.Reg {
			ctx.TransferReg(d100.Reg)
			d100.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d100)
		ctx.FreeDesc(&d105)
		ctx.ReclaimUntrackedRegs()
		d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d93)
		ctx.EnsureDescsTogether(&d107, &d93)
		if d107.Loc == scm.LocImm && d93.Loc == scm.LocImm {
			d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() - d93.Imm.Int())}
		} else if d93.Loc == scm.LocImm && d93.Imm.Int() == 0 {
			ctx.EnsureDesc(&d107)
			r86 := ctx.AllocRegExcept(d107.Reg)
			ctx.EmitMovRegReg(r86, d107.Reg)
			d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d108)
		} else if d107.Loc == scm.LocImm {
			ctx.EnsureDesc(&d93)
			scratch := ctx.AllocRegExcept(d93.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d107.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d93)
			d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d108)
		} else if d93.Loc == scm.LocImm {
			ctx.EnsureDesc(&d107)
			scratch := ctx.AllocRegExcept(d107.Reg)
			ctx.EmitMovRegReg(scratch, d107.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d93.Imm.Int())
			d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d108)
		} else {
			ctx.EnsureDesc(&d107)
			ctx.SyncDesc(&d93)
			r87 := ctx.AllocRegExcept(d107.Reg, d93.Reg)
			ctx.EmitMovRegReg(r87, d107.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r87, &d93)
			d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d108)
		}
		if d108.Loc == scm.LocReg && d107.Loc == scm.LocReg && d108.Reg == d107.Reg {
			ctx.TransferReg(d107.Reg)
			d107.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d93)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d106)
		ctx.EnsureDesc(&d108)
		ctx.EnsureDescsTogether(&d106, &d108)
		if d106.Loc == scm.LocImm && d108.Loc == scm.LocImm {
			d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d106.Imm.Int()) >> uint64(d108.Imm.Int())))}
		} else if d108.Loc == scm.LocImm {
			r88 := ctx.AllocRegExcept(d106.Reg)
			ctx.EmitMovRegReg(r88, d106.Reg)
			ctx.EmitShrRegImm8(r88, uint8(d108.Imm.Int()))
			d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d109)
		} else {
			shiftSrc := d106.Reg
			r89 := ctx.AllocRegExcept(d106.Reg, d108.Reg)
			ctx.EmitMovRegReg(r89, d106.Reg)
			shiftSrc = r89
			ctx.EmitShiftRight(shiftSrc, d108.Reg, false)
			d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d109)
		}
		if d109.Loc == scm.LocReg && d106.Loc == scm.LocReg && d109.Reg == d106.Reg {
			ctx.TransferReg(d106.Reg)
			d106.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d106)
		ctx.FreeDesc(&d108)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d109)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 88)
			r90 := ctx.AllocReg()
			ctx.EmitMovRegMem(r90, thisptr.Reg, off)
			d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r90}
			ctx.BindReg(r90, &d110)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d109)
		ctx.EnsureDesc(&d110)
		ctx.EnsureDescsTogether(&d109, &d110)
		if d109.Loc == scm.LocImm && d110.Loc == scm.LocImm {
			d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d109.Imm.Int()) == uint64(d110.Imm.Int()))}
		} else if d110.Loc == scm.LocImm {
			r91 := ctx.AllocRegExcept(d109.Reg)
			if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d109.Reg, int32(d110.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d110.Imm.Int()))
				ctx.EmitCmpInt64(d109.Reg, ctx.ScratchReg)
			}
			d111 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r91, Condition: scm.CondEqual}
			ctx.BindReg(r91, &d111)
		} else if d109.Loc == scm.LocImm {
			r92 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d109.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d110.Reg)
			d111 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r92, Condition: scm.CondEqual}
			ctx.BindReg(r92, &d111)
		} else {
			r93 := ctx.AllocRegExcept(d109.Reg)
			ctx.EmitCmpInt64(d109.Reg, d110.Reg)
			d111 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r93, Condition: scm.CondEqual}
			ctx.BindReg(r93, &d111)
		}
		ctx.FreeDesc(&d109)
		ctx.FreeDesc(&d110)
		ctx.ReclaimUntrackedRegs()
		d112 = d111
		ctx.SyncDesc(&d112)
		if d112.Loc != scm.LocImm && d112.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl32 := ctx.ReserveLabel()
		lbl33 := ctx.ReserveLabel()
		if d112.Loc == scm.LocImm {
			if d112.Imm.Bool() {
				ctx.MarkLabel(lbl32)
				ctx.EmitJmp(lbl9)
			} else {
				ctx.MarkLabel(lbl33)
				ctx.SyncDesc(&d5)
				if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
					ctx.ProtectReg(d5.Reg)
				} else if d5.Loc == scm.LocRegPair {
					ctx.ProtectReg(d5.Reg)
					ctx.ProtectReg(d5.Reg2)
				}
				d113 = d5
				if d113.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d113)
				ctx.EmitStoreToStack(d113, int32(phiBase7)+int32(0))
				if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d5.Reg)
				} else if d5.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d5.Reg)
					ctx.UnprotectReg(d5.Reg2)
				}
				ctx.EmitJmp(lbl7)
			}
		} else {
			ctx.EmitJump(d112.Condition, lbl32)
			ctx.EmitJmp(lbl33)
			ctx.FreeDesc(&d111)
			ctx.MarkLabel(lbl32)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl33)
			ctx.SyncDesc(&d5)
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			d114 = d5
			if d114.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, int32(phiBase7)+int32(0))
			if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			ctx.EmitJmp(lbl7)
		}
		bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl11)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		ctx.EmitCopyScmerToDesc(&d10, &d115)
		ctx.EmitJmp(lbl4)
		bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
		d5 = inlineHome6
		d31 = inlineHome32
		d38 = inlineHome39
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase7) + int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d116)
		ctx.EmitCopyScmerToDesc(&d10, &d116)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl4)
		d117 = result
		ctx.EnsureDesc(&d10)
		if d10.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d10, &d117)
		} else {
			switch d10.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d117, d10)
			case scm.TagInt:
				ctx.EmitMakeInt(d117, d10)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d117, d10)
			case scm.TagNil:
				ctx.EmitMakeNil(d117)
			default:
				ctx.EmitMovPairToResult(&d10, &d117)
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
		idxInt = storageIdxIntHome
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		bbpos_7_0 := int32(-1)
		_ = bbpos_7_0
		lbl34 := ctx.ReserveLabel()
		_ = lbl34
		bbpos_7_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl34)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
			r94 := ctx.AllocReg()
			r95 := ctx.AllocRegExcept(r94)
			ctx.EmitMovRegMem64(r94, fieldAddr)
			ctx.EmitMovRegMem64(r95, fieldAddr+8)
			d118 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r94, Reg2: r95}
			ctx.BindReg(r94, &d118)
			ctx.BindReg(r95, &d118)
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
			r96 := ctx.AllocReg()
			r97 := ctx.AllocRegExcept(r96)
			ctx.EmitMovRegMem(r96, thisptr.Reg, off)
			ctx.EmitMovRegMem(r97, thisptr.Reg, off+8)
			d118 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r96, Reg2: r97}
			ctx.BindReg(r96, &d118)
			ctx.BindReg(r97, &d118)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d118)
		ctx.EnsureDesc(&d118)
		if d118.Loc == scm.LocImm {
			ptr, _ := d118.Imm.RawWords()
			d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(ptr))))}
		} else if d118.Loc == scm.LocStackPair {
			d119 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d118.StackOff}
		} else {
			ctx.EnsureDesc(&d118)
			if d118.Loc != scm.LocRegPair {
				panic("StringData requires a Go string pair")
			}
			d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d118.Reg, ID: 0}
		}
		ctx.EnsureDesc(&d119)
		ctx.EnsureDesc(&d119)
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d118)
		ctx.EnsureDesc(&d119)
		d121 = idxInt
		_ = d121
		ctx.StabilizeDescForControlFlow(&d121)
		inlineHome122 := d121
		inlineHome122.ID = 0
		d123 = d118
		_ = d123
		ctx.StabilizeDescForControlFlow(&d123)
		inlineHome124 := d123
		inlineHome124.ID = 0
		d125 = d119
		_ = d125
		ctx.StabilizeDescForControlFlow(&d125)
		inlineHome126 := d125
		inlineHome126.ID = 0
		phiBase127 = ctx.AllocStack(int32(64))
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		_ = d128
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		_ = d129
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		_ = d130
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		_ = d131
		inlineResultOff132 = ctx.AllocStack(int32(16))
		d133 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: inlineResultOff132}
		lbl35 := ctx.ReserveLabel()
		bbpos_8_0 := int32(-1)
		_ = bbpos_8_0
		lbl36 := ctx.ReserveLabel()
		_ = lbl36
		bbpos_8_1 := int32(-1)
		_ = bbpos_8_1
		lbl37 := ctx.ReserveLabel()
		_ = lbl37
		bbpos_8_2 := int32(-1)
		_ = bbpos_8_2
		lbl38 := ctx.ReserveLabel()
		_ = lbl38
		bbpos_8_3 := int32(-1)
		_ = bbpos_8_3
		lbl39 := ctx.ReserveLabel()
		_ = lbl39
		bbpos_8_4 := int32(-1)
		_ = bbpos_8_4
		lbl40 := ctx.ReserveLabel()
		_ = lbl40
		bbpos_8_5 := int32(-1)
		_ = bbpos_8_5
		lbl41 := ctx.ReserveLabel()
		_ = lbl41
		bbpos_8_6 := int32(-1)
		_ = bbpos_8_6
		lbl42 := ctx.ReserveLabel()
		_ = lbl42
		bbpos_8_7 := int32(-1)
		_ = bbpos_8_7
		lbl43 := ctx.ReserveLabel()
		_ = lbl43
		bbpos_8_8 := int32(-1)
		_ = bbpos_8_8
		lbl44 := ctx.ReserveLabel()
		_ = lbl44
		bbpos_8_9 := int32(-1)
		_ = bbpos_8_9
		lbl45 := ctx.ReserveLabel()
		_ = lbl45
		bbpos_8_10 := int32(-1)
		_ = bbpos_8_10
		lbl46 := ctx.ReserveLabel()
		_ = lbl46
		bbpos_8_11 := int32(-1)
		_ = bbpos_8_11
		lbl47 := ctx.ReserveLabel()
		_ = lbl47
		bbpos_8_12 := int32(-1)
		_ = bbpos_8_12
		lbl48 := ctx.ReserveLabel()
		_ = lbl48
		bbpos_8_13 := int32(-1)
		_ = bbpos_8_13
		lbl49 := ctx.ReserveLabel()
		_ = lbl49
		bbpos_8_14 := int32(-1)
		_ = bbpos_8_14
		lbl50 := ctx.ReserveLabel()
		_ = lbl50
		bbpos_8_15 := int32(-1)
		_ = bbpos_8_15
		lbl51 := ctx.ReserveLabel()
		_ = lbl51
		bbpos_8_16 := int32(-1)
		_ = bbpos_8_16
		lbl52 := ctx.ReserveLabel()
		_ = lbl52
		bbpos_8_17 := int32(-1)
		_ = bbpos_8_17
		lbl53 := ctx.ReserveLabel()
		_ = lbl53
		bbpos_8_18 := int32(-1)
		_ = bbpos_8_18
		lbl54 := ctx.ReserveLabel()
		_ = lbl54
		bbpos_8_19 := int32(-1)
		_ = bbpos_8_19
		lbl55 := ctx.ReserveLabel()
		_ = lbl55
		bbpos_8_20 := int32(-1)
		_ = bbpos_8_20
		lbl56 := ctx.ReserveLabel()
		_ = lbl56
		bbpos_8_21 := int32(-1)
		_ = bbpos_8_21
		lbl57 := ctx.ReserveLabel()
		_ = lbl57
		bbpos_8_22 := int32(-1)
		_ = bbpos_8_22
		lbl58 := ctx.ReserveLabel()
		_ = lbl58
		bbpos_8_23 := int32(-1)
		_ = bbpos_8_23
		lbl59 := ctx.ReserveLabel()
		_ = lbl59
		bbpos_8_24 := int32(-1)
		_ = bbpos_8_24
		lbl60 := ctx.ReserveLabel()
		_ = lbl60
		bbpos_8_25 := int32(-1)
		_ = bbpos_8_25
		lbl61 := ctx.ReserveLabel()
		_ = lbl61
		bbpos_8_26 := int32(-1)
		_ = bbpos_8_26
		lbl62 := ctx.ReserveLabel()
		_ = lbl62
		bbpos_8_27 := int32(-1)
		_ = bbpos_8_27
		lbl63 := ctx.ReserveLabel()
		_ = lbl63
		bbpos_8_28 := int32(-1)
		_ = bbpos_8_28
		lbl64 := ctx.ReserveLabel()
		_ = lbl64
		bbpos_8_29 := int32(-1)
		_ = bbpos_8_29
		lbl65 := ctx.ReserveLabel()
		_ = lbl65
		bbpos_8_30 := int32(-1)
		_ = bbpos_8_30
		lbl66 := ctx.ReserveLabel()
		_ = lbl66
		bbpos_8_31 := int32(-1)
		_ = bbpos_8_31
		lbl67 := ctx.ReserveLabel()
		_ = lbl67
		bbpos_8_32 := int32(-1)
		_ = bbpos_8_32
		lbl68 := ctx.ReserveLabel()
		_ = lbl68
		bbpos_8_33 := int32(-1)
		_ = bbpos_8_33
		lbl69 := ctx.ReserveLabel()
		_ = lbl69
		bbpos_8_34 := int32(-1)
		_ = bbpos_8_34
		lbl70 := ctx.ReserveLabel()
		_ = lbl70
		bbpos_8_35 := int32(-1)
		_ = bbpos_8_35
		lbl71 := ctx.ReserveLabel()
		_ = lbl71
		bbpos_8_36 := int32(-1)
		_ = bbpos_8_36
		lbl72 := ctx.ReserveLabel()
		_ = lbl72
		bbpos_8_37 := int32(-1)
		_ = bbpos_8_37
		lbl73 := ctx.ReserveLabel()
		_ = lbl73
		bbpos_8_38 := int32(-1)
		_ = bbpos_8_38
		lbl74 := ctx.ReserveLabel()
		_ = lbl74
		bbpos_8_39 := int32(-1)
		_ = bbpos_8_39
		lbl75 := ctx.ReserveLabel()
		_ = lbl75
		bbpos_8_40 := int32(-1)
		_ = bbpos_8_40
		lbl76 := ctx.ReserveLabel()
		_ = lbl76
		bbpos_8_41 := int32(-1)
		_ = bbpos_8_41
		lbl77 := ctx.ReserveLabel()
		_ = lbl77
		bbpos_8_42 := int32(-1)
		_ = bbpos_8_42
		lbl78 := ctx.ReserveLabel()
		_ = lbl78
		bbpos_8_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl36)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
			r98 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r98, thisptr.Reg, off)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			ctx.BindReg(r98, &d134)
		}
		ctx.ReclaimUntrackedRegs()
		d135 = d134
		ctx.EnsureDesc(&d135)
		if d135.Loc != scm.LocImm && d135.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl79 := ctx.ReserveLabel()
		lbl80 := ctx.ReserveLabel()
		if d135.Loc == scm.LocImm {
			if d135.Imm.Bool() {
				ctx.MarkLabel(lbl79)
				ctx.EmitJmp(lbl37)
			} else {
				ctx.MarkLabel(lbl80)
				ctx.EmitJmp(lbl39)
			}
		} else {
			ctx.EmitCmpRegImm32(d135.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl79)
			ctx.EmitJmp(lbl80)
			ctx.MarkLabel(lbl79)
			ctx.EmitJmp(lbl37)
			ctx.MarkLabel(lbl80)
			ctx.EmitJmp(lbl39)
		}
		ctx.FreeDesc(&d134)
		bbpos_8_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl39)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		d136 = d121
		_ = d136
		bbpos_9_0 := int32(-1)
		_ = bbpos_9_0
		lbl81 := ctx.ReserveLabel()
		_ = lbl81
		bbpos_9_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl81)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 48)
			r99 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r99, thisptr.Reg, off)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			ctx.BindReg(r99, &d137)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d137)
		ctx.EnsureDesc(&d137)
		if d137.Loc == scm.LocImm {
			d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d137.Imm.Int()))))}
		} else {
			r100 := ctx.AllocReg()
			ctx.EmitMovRegReg(r100, d137.Reg)
			ctx.EmitShlRegImm8(r100, 56)
			ctx.EmitShrRegImm8(r100, 56)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d138)
		}
		ctx.FreeDesc(&d137)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		ctx.EnsureDesc(&d136)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d136)
		ctx.EnsureDesc(&d138)
		ctx.EnsureDescsTogether(&d136, &d138)
		if d136.Loc == scm.LocImm && d138.Loc == scm.LocImm {
			d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() * d138.Imm.Int())}
		} else if d136.Loc == scm.LocImm {
			ctx.EnsureDesc(&d138)
			scratch := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitMovRegReg(scratch, d138.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d136.Imm.Int())
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d140)
		} else if d138.Loc == scm.LocImm {
			ctx.EnsureDesc(&d136)
			scratch := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(scratch, d136.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d138.Imm.Int())
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d140)
		} else {
			ctx.EnsureDesc(&d136)
			ctx.SyncDesc(&d138)
			r101 := ctx.AllocRegExcept(d136.Reg, d138.Reg)
			ctx.EmitMovRegReg(r101, d136.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r101, &d138)
			d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d140)
		}
		if d140.Loc == scm.LocReg && d136.Loc == scm.LocReg && d140.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		if d140.Loc == scm.LocImm {
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() / 64)}
		} else {
			r102 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r102, d140.Reg)
			ctx.EmitShrRegImm8(r102, 6)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d141)
		}
		if d141.Loc == scm.LocReg && d140.Loc == scm.LocReg && d141.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d140)
		if d140.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
		} else {
			r103 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r103, d140.Reg)
			ctx.EmitAndRegImm32(r103, 63)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			ctx.BindReg(r103, &d142)
		}
		if d142.Loc == scm.LocReg && d140.Loc == scm.LocReg && d142.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d140)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d143 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r104 := ctx.AllocReg()
			r105 := ctx.AllocRegExcept(r104)
			r106 := ctx.AllocRegExcept(r104, r105)
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
			ctx.EmitMovRegMem(r104, thisptr.Reg, off)
			ctx.EmitMovRegMem(r105, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r106, thisptr.Reg, off+16)
			d143 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r104, Reg2: r105, Reg3: r106}
			ctx.BindReg(r104, &d143)
			ctx.BindReg(r105, &d143)
			ctx.BindReg(r106, &d143)
			ctx.BindReg(r104, &d143)
			ctx.BindReg(r105, &d143)
			ctx.BindReg(r106, &d143)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d141)
		ctx.ReclaimUntrackedRegs()
		d144 = ctx.EmitLoadScalarSliceElement(&d143, &d141, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d144)
		ctx.EnsureDesc(&d142)
		ctx.EnsureDescsTogether(&d144, &d142)
		if d144.Loc == scm.LocImm && d142.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d144.Imm.Int()) << uint64(d142.Imm.Int())))}
		} else if d142.Loc == scm.LocImm {
			r107 := ctx.AllocRegExcept(d144.Reg)
			ctx.EmitMovRegReg(r107, d144.Reg)
			ctx.EmitShlRegImm8(r107, uint8(d142.Imm.Int()))
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d145)
		} else {
			shiftSrc := d144.Reg
			r108 := ctx.AllocRegExcept(d144.Reg, d142.Reg)
			ctx.EmitMovRegReg(r108, d144.Reg)
			shiftSrc = r108
			ctx.EmitShiftLeft(shiftSrc, d142.Reg, true)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d145)
		}
		if d145.Loc == scm.LocReg && d144.Loc == scm.LocReg && d145.Reg == d144.Reg {
			ctx.TransferReg(d144.Reg)
			d144.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d144)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d141)
		ctx.EnsureDesc(&d141)
		if d141.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d141.Reg)
			ctx.EmitMovRegReg(scratch, d141.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d146)
		}
		if d146.Loc == scm.LocReg && d141.Loc == scm.LocReg && d146.Reg == d141.Reg {
			ctx.TransferReg(d141.Reg)
			d141.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d141)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d146)
		ctx.ReclaimUntrackedRegs()
		d147 = ctx.EmitLoadScalarSliceElement(&d143, &d146, 8, scm.TagInt)
		ctx.FreeDesc(&d146)
		ctx.ReclaimUntrackedRegs()
		d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d142)
		ctx.EnsureDescsTogether(&d148, &d142)
		if d148.Loc == scm.LocImm && d142.Loc == scm.LocImm {
			d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() - d142.Imm.Int())}
		} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
			ctx.EnsureDesc(&d148)
			r109 := ctx.AllocRegExcept(d148.Reg)
			ctx.EmitMovRegReg(r109, d148.Reg)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			ctx.BindReg(r109, &d149)
		} else if d148.Loc == scm.LocImm {
			ctx.EnsureDesc(&d142)
			scratch := ctx.AllocRegExcept(d142.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d148.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d142)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d149)
		} else if d142.Loc == scm.LocImm {
			ctx.EnsureDesc(&d148)
			scratch := ctx.AllocRegExcept(d148.Reg)
			ctx.EmitMovRegReg(scratch, d148.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d142.Imm.Int())
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d149)
		} else {
			ctx.EnsureDesc(&d148)
			ctx.SyncDesc(&d142)
			r110 := ctx.AllocRegExcept(d148.Reg, d142.Reg)
			ctx.EmitMovRegReg(r110, d148.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r110, &d142)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d149)
		}
		if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
			ctx.TransferReg(d148.Reg)
			d148.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d142)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d147)
		ctx.EnsureDesc(&d149)
		ctx.EnsureDescsTogether(&d147, &d149)
		if d147.Loc == scm.LocImm && d149.Loc == scm.LocImm {
			d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d147.Imm.Int()) >> uint64(d149.Imm.Int())))}
		} else if d149.Loc == scm.LocImm {
			r111 := ctx.AllocRegExcept(d147.Reg)
			ctx.EmitMovRegReg(r111, d147.Reg)
			ctx.EmitShrRegImm8(r111, uint8(d149.Imm.Int()))
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d150)
		} else {
			shiftSrc := d147.Reg
			r112 := ctx.AllocRegExcept(d147.Reg, d149.Reg)
			ctx.EmitMovRegReg(r112, d147.Reg)
			shiftSrc = r112
			ctx.EmitShiftRight(shiftSrc, d149.Reg, false)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d150)
		}
		if d150.Loc == scm.LocReg && d147.Loc == scm.LocReg && d150.Reg == d147.Reg {
			ctx.TransferReg(d147.Reg)
			d147.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d147)
		ctx.FreeDesc(&d149)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d145)
		ctx.EnsureDesc(&d150)
		if d145.Loc == scm.LocImm && d150.Loc == scm.LocImm {
			d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() | d150.Imm.Int())}
		} else if d145.Loc == scm.LocImm && d145.Imm.Int() == 0 {
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
			ctx.BindReg(d150.Reg, &d151)
		} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
			r113 := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitMovRegReg(r113, d145.Reg)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d151)
		} else if d145.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d150.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
			ctx.EmitOrInt64(scratch, d150.Reg)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d151)
		} else if d150.Loc == scm.LocImm {
			r114 := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitMovRegReg(r114, d145.Reg)
			if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r114, int32(d150.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d150.Imm.Int()))
				ctx.EmitOrInt64(r114, ctx.ScratchReg)
			}
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d151)
		} else {
			r115 := ctx.AllocRegExcept(d145.Reg, d150.Reg)
			ctx.EmitMovRegReg(r115, d145.Reg)
			ctx.EmitOrInt64(r115, d150.Reg)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			ctx.BindReg(r115, &d151)
		}
		if d151.Loc == scm.LocReg && d145.Loc == scm.LocReg && d151.Reg == d145.Reg {
			ctx.TransferReg(d145.Reg)
			d145.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d145)
		ctx.FreeDesc(&d150)
		ctx.ReclaimUntrackedRegs()
		d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d138)
		ctx.EnsureDescsTogether(&d152, &d138)
		if d152.Loc == scm.LocImm && d138.Loc == scm.LocImm {
			d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() - d138.Imm.Int())}
		} else if d138.Loc == scm.LocImm && d138.Imm.Int() == 0 {
			ctx.EnsureDesc(&d152)
			r116 := ctx.AllocRegExcept(d152.Reg)
			ctx.EmitMovRegReg(r116, d152.Reg)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d153)
		} else if d152.Loc == scm.LocImm {
			ctx.EnsureDesc(&d138)
			scratch := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d152.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d138)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d153)
		} else if d138.Loc == scm.LocImm {
			ctx.EnsureDesc(&d152)
			scratch := ctx.AllocRegExcept(d152.Reg)
			ctx.EmitMovRegReg(scratch, d152.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d138.Imm.Int())
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d153)
		} else {
			ctx.EnsureDesc(&d152)
			ctx.SyncDesc(&d138)
			r117 := ctx.AllocRegExcept(d152.Reg, d138.Reg)
			ctx.EmitMovRegReg(r117, d152.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r117, &d138)
			d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d153)
		}
		if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
			ctx.TransferReg(d152.Reg)
			d152.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d138)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d151)
		ctx.EnsureDesc(&d153)
		ctx.EnsureDescsTogether(&d151, &d153)
		if d151.Loc == scm.LocImm && d153.Loc == scm.LocImm {
			d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d151.Imm.Int()) >> uint64(d153.Imm.Int())))}
		} else if d153.Loc == scm.LocImm {
			r118 := ctx.AllocRegExcept(d151.Reg)
			ctx.EmitMovRegReg(r118, d151.Reg)
			ctx.EmitShrRegImm8(r118, uint8(d153.Imm.Int()))
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d154)
		} else {
			shiftSrc := d151.Reg
			r119 := ctx.AllocRegExcept(d151.Reg, d153.Reg)
			ctx.EmitMovRegReg(r119, d151.Reg)
			shiftSrc = r119
			ctx.EmitShiftRight(shiftSrc, d153.Reg, false)
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d154)
		}
		if d154.Loc == scm.LocReg && d151.Loc == scm.LocReg && d154.Reg == d151.Reg {
			ctx.TransferReg(d151.Reg)
			d151.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d151)
		ctx.FreeDesc(&d153)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d154)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d154)
		ctx.EnsureDesc(&d154)
		if d154.Loc == scm.LocImm {
			d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d154.Imm.Int()))))}
		} else {
			r120 := ctx.AllocReg()
			ctx.EmitMovRegReg(r120, d154.Reg)
			d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d155)
		}
		ctx.FreeDesc(&d154)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
			r121 := ctx.AllocReg()
			ctx.EmitMovRegMem(r121, thisptr.Reg, off)
			d156 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d156)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d155)
		ctx.EnsureDesc(&d156)
		ctx.EnsureDescsTogether(&d155, &d156)
		if d155.Loc == scm.LocImm && d156.Loc == scm.LocImm {
			d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() + d156.Imm.Int())}
		} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
			ctx.EnsureDesc(&d155)
			r122 := ctx.AllocRegExcept(d155.Reg)
			ctx.EmitMovRegReg(r122, d155.Reg)
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d157)
		} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
			ctx.EnsureDesc(&d156)
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d156.Reg}
			ctx.BindReg(d156.Reg, &d157)
		} else if d155.Loc == scm.LocImm {
			ctx.EnsureDesc(&d156)
			scratch := ctx.AllocRegExcept(d156.Reg)
			ctx.EmitMovRegReg(scratch, d156.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d155.Imm.Int())
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d157)
		} else if d156.Loc == scm.LocImm {
			ctx.EnsureDesc(&d155)
			scratch := ctx.AllocRegExcept(d155.Reg)
			ctx.EmitMovRegReg(scratch, d155.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d156.Imm.Int())
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d157)
		} else {
			ctx.EnsureDesc(&d155)
			ctx.SyncDesc(&d156)
			r123 := ctx.AllocRegExcept(d155.Reg, d156.Reg)
			ctx.EmitMovRegReg(r123, d155.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r123, &d156)
			d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d157)
		}
		if d157.Loc == scm.LocReg && d155.Loc == scm.LocReg && d157.Reg == d155.Reg {
			ctx.TransferReg(d155.Reg)
			d155.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d155)
		ctx.FreeDesc(&d156)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d157)
		ctx.EnsureDesc(&d157)
		if d157.Loc == scm.LocImm {
			d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d157.Imm.Int()))))}
		} else {
			r124 := ctx.AllocReg()
			ctx.EmitMovRegReg(r124, d157.Reg)
			ctx.EmitShlRegImm8(r124, 32)
			ctx.EmitShrRegImm8(r124, 32)
			d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d158)
		}
		ctx.StabilizeDescForControlFlow(&d158)
		inlineHome159 := d158
		inlineHome159.ID = 0
		ctx.FreeDesc(&d157)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 80)
			r125 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r125, thisptr.Reg, off)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r125}
			ctx.BindReg(r125, &d160)
		}
		ctx.ReclaimUntrackedRegs()
		d161 = d160
		ctx.EnsureDesc(&d161)
		if d161.Loc != scm.LocImm && d161.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl82 := ctx.ReserveLabel()
		lbl83 := ctx.ReserveLabel()
		if d161.Loc == scm.LocImm {
			if d161.Imm.Bool() {
				ctx.MarkLabel(lbl82)
				ctx.EmitJmp(lbl45)
			} else {
				ctx.MarkLabel(lbl83)
				ctx.EmitJmp(lbl44)
			}
		} else {
			ctx.EmitCmpRegImm32(d161.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl82)
			ctx.EmitJmp(lbl83)
			ctx.MarkLabel(lbl82)
			ctx.EmitJmp(lbl45)
			ctx.MarkLabel(lbl83)
			ctx.EmitJmp(lbl44)
		}
		ctx.FreeDesc(&d160)
		bbpos_8_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl44)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d158)
		d162 = d158
		_ = d162
		ctx.StabilizeDescForControlFlow(&d158)
		bbpos_10_0 := int32(-1)
		_ = bbpos_10_0
		lbl84 := ctx.ReserveLabel()
		_ = lbl84
		bbpos_10_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl84)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 48)
			r126 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r126, thisptr.Reg, off)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d163)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d163)
		ctx.EnsureDesc(&d163)
		if d163.Loc == scm.LocImm {
			d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d163.Imm.Int()))))}
		} else {
			r127 := ctx.AllocReg()
			ctx.EmitMovRegReg(r127, d163.Reg)
			ctx.EmitShlRegImm8(r127, 56)
			ctx.EmitShrRegImm8(r127, 56)
			d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d164)
		}
		ctx.FreeDesc(&d163)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d162)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDescsTogether(&d162, &d164)
		if d162.Loc == scm.LocImm && d164.Loc == scm.LocImm {
			d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() * d164.Imm.Int())}
		} else if d162.Loc == scm.LocImm {
			ctx.EnsureDesc(&d164)
			scratch := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitMovRegReg(scratch, d164.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d162.Imm.Int())
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d166)
		} else if d164.Loc == scm.LocImm {
			ctx.EnsureDesc(&d162)
			scratch := ctx.AllocRegExcept(d162.Reg)
			ctx.EmitMovRegReg(scratch, d162.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d164.Imm.Int())
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d166)
		} else {
			ctx.EnsureDesc(&d162)
			ctx.SyncDesc(&d164)
			r128 := ctx.AllocRegExcept(d162.Reg, d164.Reg)
			ctx.EmitMovRegReg(r128, d162.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r128, &d164)
			d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d166)
		}
		if d166.Loc == scm.LocReg && d162.Loc == scm.LocReg && d166.Reg == d162.Reg {
			ctx.TransferReg(d162.Reg)
			d162.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d166)
		if d166.Loc == scm.LocImm {
			d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d166.Imm.Int() / 64)}
		} else {
			r129 := ctx.AllocRegExcept(d166.Reg)
			ctx.EmitMovRegReg(r129, d166.Reg)
			ctx.EmitShrRegImm8(r129, 6)
			d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d167)
		}
		if d167.Loc == scm.LocReg && d166.Loc == scm.LocReg && d167.Reg == d166.Reg {
			ctx.TransferReg(d166.Reg)
			d166.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d166)
		if d166.Loc == scm.LocImm {
			d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d166.Imm.Int() % 64)}
		} else {
			r130 := ctx.AllocRegExcept(d166.Reg)
			ctx.EmitMovRegReg(r130, d166.Reg)
			ctx.EmitAndRegImm32(r130, 63)
			d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d168)
		}
		if d168.Loc == scm.LocReg && d166.Loc == scm.LocReg && d168.Reg == d166.Reg {
			ctx.TransferReg(d166.Reg)
			d166.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d166)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d169 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r131 := ctx.AllocReg()
			r132 := ctx.AllocRegExcept(r131)
			r133 := ctx.AllocRegExcept(r131, r132)
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
			ctx.EmitMovRegMem(r131, thisptr.Reg, off)
			ctx.EmitMovRegMem(r132, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r133, thisptr.Reg, off+16)
			d169 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r131, Reg2: r132, Reg3: r133}
			ctx.BindReg(r131, &d169)
			ctx.BindReg(r132, &d169)
			ctx.BindReg(r133, &d169)
			ctx.BindReg(r131, &d169)
			ctx.BindReg(r132, &d169)
			ctx.BindReg(r133, &d169)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d167)
		ctx.ReclaimUntrackedRegs()
		d170 = ctx.EmitLoadScalarSliceElement(&d169, &d167, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d170)
		ctx.EnsureDesc(&d168)
		ctx.EnsureDescsTogether(&d170, &d168)
		if d170.Loc == scm.LocImm && d168.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d170.Imm.Int()) << uint64(d168.Imm.Int())))}
		} else if d168.Loc == scm.LocImm {
			r134 := ctx.AllocRegExcept(d170.Reg)
			ctx.EmitMovRegReg(r134, d170.Reg)
			ctx.EmitShlRegImm8(r134, uint8(d168.Imm.Int()))
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d171)
		} else {
			shiftSrc := d170.Reg
			r135 := ctx.AllocRegExcept(d170.Reg, d168.Reg)
			ctx.EmitMovRegReg(r135, d170.Reg)
			shiftSrc = r135
			ctx.EmitShiftLeft(shiftSrc, d168.Reg, true)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d171)
		}
		if d171.Loc == scm.LocReg && d170.Loc == scm.LocReg && d171.Reg == d170.Reg {
			ctx.TransferReg(d170.Reg)
			d170.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d170)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d167)
		ctx.EnsureDesc(&d167)
		if d167.Loc == scm.LocImm {
			d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d167.Reg)
			ctx.EmitMovRegReg(scratch, d167.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d172)
		}
		if d172.Loc == scm.LocReg && d167.Loc == scm.LocReg && d172.Reg == d167.Reg {
			ctx.TransferReg(d167.Reg)
			d167.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d167)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d172)
		ctx.ReclaimUntrackedRegs()
		d173 = ctx.EmitLoadScalarSliceElement(&d169, &d172, 8, scm.TagInt)
		ctx.FreeDesc(&d172)
		ctx.ReclaimUntrackedRegs()
		d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d168)
		ctx.EnsureDescsTogether(&d174, &d168)
		if d174.Loc == scm.LocImm && d168.Loc == scm.LocImm {
			d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() - d168.Imm.Int())}
		} else if d168.Loc == scm.LocImm && d168.Imm.Int() == 0 {
			ctx.EnsureDesc(&d174)
			r136 := ctx.AllocRegExcept(d174.Reg)
			ctx.EmitMovRegReg(r136, d174.Reg)
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d175)
		} else if d174.Loc == scm.LocImm {
			ctx.EnsureDesc(&d168)
			scratch := ctx.AllocRegExcept(d168.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d168)
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d175)
		} else if d168.Loc == scm.LocImm {
			ctx.EnsureDesc(&d174)
			scratch := ctx.AllocRegExcept(d174.Reg)
			ctx.EmitMovRegReg(scratch, d174.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d168.Imm.Int())
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d175)
		} else {
			ctx.EnsureDesc(&d174)
			ctx.SyncDesc(&d168)
			r137 := ctx.AllocRegExcept(d174.Reg, d168.Reg)
			ctx.EmitMovRegReg(r137, d174.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r137, &d168)
			d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			ctx.BindReg(r137, &d175)
		}
		if d175.Loc == scm.LocReg && d174.Loc == scm.LocReg && d175.Reg == d174.Reg {
			ctx.TransferReg(d174.Reg)
			d174.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d168)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d173)
		ctx.EnsureDesc(&d175)
		ctx.EnsureDescsTogether(&d173, &d175)
		if d173.Loc == scm.LocImm && d175.Loc == scm.LocImm {
			d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d173.Imm.Int()) >> uint64(d175.Imm.Int())))}
		} else if d175.Loc == scm.LocImm {
			r138 := ctx.AllocRegExcept(d173.Reg)
			ctx.EmitMovRegReg(r138, d173.Reg)
			ctx.EmitShrRegImm8(r138, uint8(d175.Imm.Int()))
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d176)
		} else {
			shiftSrc := d173.Reg
			r139 := ctx.AllocRegExcept(d173.Reg, d175.Reg)
			ctx.EmitMovRegReg(r139, d173.Reg)
			shiftSrc = r139
			ctx.EmitShiftRight(shiftSrc, d175.Reg, false)
			d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d176)
		}
		if d176.Loc == scm.LocReg && d173.Loc == scm.LocReg && d176.Reg == d173.Reg {
			ctx.TransferReg(d173.Reg)
			d173.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d173)
		ctx.FreeDesc(&d175)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d171)
		ctx.EnsureDesc(&d176)
		if d171.Loc == scm.LocImm && d176.Loc == scm.LocImm {
			d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() | d176.Imm.Int())}
		} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			ctx.BindReg(d176.Reg, &d177)
		} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
			r140 := ctx.AllocRegExcept(d171.Reg)
			ctx.EmitMovRegReg(r140, d171.Reg)
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d177)
		} else if d171.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d176.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
			ctx.EmitOrInt64(scratch, d176.Reg)
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d177)
		} else if d176.Loc == scm.LocImm {
			r141 := ctx.AllocRegExcept(d171.Reg)
			ctx.EmitMovRegReg(r141, d171.Reg)
			if d176.Imm.Int() >= -2147483648 && d176.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r141, int32(d176.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d176.Imm.Int()))
				ctx.EmitOrInt64(r141, ctx.ScratchReg)
			}
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d177)
		} else {
			r142 := ctx.AllocRegExcept(d171.Reg, d176.Reg)
			ctx.EmitMovRegReg(r142, d171.Reg)
			ctx.EmitOrInt64(r142, d176.Reg)
			d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d177)
		}
		if d177.Loc == scm.LocReg && d171.Loc == scm.LocReg && d177.Reg == d171.Reg {
			ctx.TransferReg(d171.Reg)
			d171.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d171)
		ctx.FreeDesc(&d176)
		ctx.ReclaimUntrackedRegs()
		d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d164)
		ctx.EnsureDescsTogether(&d178, &d164)
		if d178.Loc == scm.LocImm && d164.Loc == scm.LocImm {
			d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() - d164.Imm.Int())}
		} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
			ctx.EnsureDesc(&d178)
			r143 := ctx.AllocRegExcept(d178.Reg)
			ctx.EmitMovRegReg(r143, d178.Reg)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d179)
		} else if d178.Loc == scm.LocImm {
			ctx.EnsureDesc(&d164)
			scratch := ctx.AllocRegExcept(d164.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d178.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d164)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d179)
		} else if d164.Loc == scm.LocImm {
			ctx.EnsureDesc(&d178)
			scratch := ctx.AllocRegExcept(d178.Reg)
			ctx.EmitMovRegReg(scratch, d178.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d164.Imm.Int())
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d179)
		} else {
			ctx.EnsureDesc(&d178)
			ctx.SyncDesc(&d164)
			r144 := ctx.AllocRegExcept(d178.Reg, d164.Reg)
			ctx.EmitMovRegReg(r144, d178.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r144, &d164)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d179)
		}
		if d179.Loc == scm.LocReg && d178.Loc == scm.LocReg && d179.Reg == d178.Reg {
			ctx.TransferReg(d178.Reg)
			d178.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d164)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d177)
		ctx.EnsureDesc(&d179)
		ctx.EnsureDescsTogether(&d177, &d179)
		if d177.Loc == scm.LocImm && d179.Loc == scm.LocImm {
			d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d177.Imm.Int()) >> uint64(d179.Imm.Int())))}
		} else if d179.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d177.Reg)
			ctx.EmitMovRegReg(r145, d177.Reg)
			ctx.EmitShrRegImm8(r145, uint8(d179.Imm.Int()))
			d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d180)
		} else {
			shiftSrc := d177.Reg
			r146 := ctx.AllocRegExcept(d177.Reg, d179.Reg)
			ctx.EmitMovRegReg(r146, d177.Reg)
			shiftSrc = r146
			ctx.EmitShiftRight(shiftSrc, d179.Reg, false)
			d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d180)
		}
		if d180.Loc == scm.LocReg && d177.Loc == scm.LocReg && d180.Reg == d177.Reg {
			ctx.TransferReg(d177.Reg)
			d177.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d177)
		ctx.FreeDesc(&d179)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d180)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d180)
		ctx.EnsureDesc(&d180)
		if d180.Loc == scm.LocImm {
			d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d180.Imm.Int()))))}
		} else {
			r147 := ctx.AllocReg()
			ctx.EmitMovRegReg(r147, d180.Reg)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d181)
		}
		ctx.FreeDesc(&d180)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
			r148 := ctx.AllocReg()
			ctx.EmitMovRegMem(r148, thisptr.Reg, off)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
			ctx.BindReg(r148, &d182)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d181)
		ctx.EnsureDesc(&d182)
		ctx.EnsureDescsTogether(&d181, &d182)
		if d181.Loc == scm.LocImm && d182.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() + d182.Imm.Int())}
		} else if d182.Loc == scm.LocImm && d182.Imm.Int() == 0 {
			ctx.EnsureDesc(&d181)
			r149 := ctx.AllocRegExcept(d181.Reg)
			ctx.EmitMovRegReg(r149, d181.Reg)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d183)
		} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
			ctx.EnsureDesc(&d182)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d182.Reg}
			ctx.BindReg(d182.Reg, &d183)
		} else if d181.Loc == scm.LocImm {
			ctx.EnsureDesc(&d182)
			scratch := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegReg(scratch, d182.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d181.Imm.Int())
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		} else if d182.Loc == scm.LocImm {
			ctx.EnsureDesc(&d181)
			scratch := ctx.AllocRegExcept(d181.Reg)
			ctx.EmitMovRegReg(scratch, d181.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d182.Imm.Int())
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		} else {
			ctx.EnsureDesc(&d181)
			ctx.SyncDesc(&d182)
			r150 := ctx.AllocRegExcept(d181.Reg, d182.Reg)
			ctx.EmitMovRegReg(r150, d181.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r150, &d182)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d183)
		}
		if d183.Loc == scm.LocReg && d181.Loc == scm.LocReg && d183.Reg == d181.Reg {
			ctx.TransferReg(d181.Reg)
			d181.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d181)
		ctx.FreeDesc(&d182)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d183)
		ctx.EnsureDesc(&d183)
		ctx.StabilizeDescForControlFlow(&d183)
		inlineHome185 := d183
		inlineHome185.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d158)
		d186 = d158
		_ = d186
		bbpos_11_0 := int32(-1)
		_ = bbpos_11_0
		lbl85 := ctx.ReserveLabel()
		_ = lbl85
		bbpos_11_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl85)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 48)
			r151 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r151, thisptr.Reg, off)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r151}
			ctx.BindReg(r151, &d187)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d187)
		ctx.EnsureDesc(&d187)
		if d187.Loc == scm.LocImm {
			d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d187.Imm.Int()))))}
		} else {
			r152 := ctx.AllocReg()
			ctx.EmitMovRegReg(r152, d187.Reg)
			ctx.EmitShlRegImm8(r152, 56)
			ctx.EmitShrRegImm8(r152, 56)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d188)
		}
		ctx.FreeDesc(&d187)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d186)
		ctx.EnsureDesc(&d186)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d186)
		ctx.EnsureDesc(&d188)
		ctx.EnsureDescsTogether(&d186, &d188)
		if d186.Loc == scm.LocImm && d188.Loc == scm.LocImm {
			d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() * d188.Imm.Int())}
		} else if d186.Loc == scm.LocImm {
			ctx.EnsureDesc(&d188)
			scratch := ctx.AllocRegExcept(d188.Reg)
			ctx.EmitMovRegReg(scratch, d188.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d186.Imm.Int())
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d190)
		} else if d188.Loc == scm.LocImm {
			ctx.EnsureDesc(&d186)
			scratch := ctx.AllocRegExcept(d186.Reg)
			ctx.EmitMovRegReg(scratch, d186.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d188.Imm.Int())
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d190)
		} else {
			ctx.EnsureDesc(&d186)
			ctx.SyncDesc(&d188)
			r153 := ctx.AllocRegExcept(d186.Reg, d188.Reg)
			ctx.EmitMovRegReg(r153, d186.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r153, &d188)
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d190)
		}
		if d190.Loc == scm.LocReg && d186.Loc == scm.LocReg && d190.Reg == d186.Reg {
			ctx.TransferReg(d186.Reg)
			d186.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d190)
		if d190.Loc == scm.LocImm {
			d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() / 64)}
		} else {
			r154 := ctx.AllocRegExcept(d190.Reg)
			ctx.EmitMovRegReg(r154, d190.Reg)
			ctx.EmitShrRegImm8(r154, 6)
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			ctx.BindReg(r154, &d191)
		}
		if d191.Loc == scm.LocReg && d190.Loc == scm.LocReg && d191.Reg == d190.Reg {
			ctx.TransferReg(d190.Reg)
			d190.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d190)
		if d190.Loc == scm.LocImm {
			d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() % 64)}
		} else {
			r155 := ctx.AllocRegExcept(d190.Reg)
			ctx.EmitMovRegReg(r155, d190.Reg)
			ctx.EmitAndRegImm32(r155, 63)
			d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d192)
		}
		if d192.Loc == scm.LocReg && d190.Loc == scm.LocReg && d192.Reg == d190.Reg {
			ctx.TransferReg(d190.Reg)
			d190.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d190)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d193 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r156 := ctx.AllocReg()
			r157 := ctx.AllocRegExcept(r156)
			r158 := ctx.AllocRegExcept(r156, r157)
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
			ctx.EmitMovRegMem(r156, thisptr.Reg, off)
			ctx.EmitMovRegMem(r157, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r158, thisptr.Reg, off+16)
			d193 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r156, Reg2: r157, Reg3: r158}
			ctx.BindReg(r156, &d193)
			ctx.BindReg(r157, &d193)
			ctx.BindReg(r158, &d193)
			ctx.BindReg(r156, &d193)
			ctx.BindReg(r157, &d193)
			ctx.BindReg(r158, &d193)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d191)
		ctx.ReclaimUntrackedRegs()
		d194 = ctx.EmitLoadScalarSliceElement(&d193, &d191, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d194)
		ctx.EnsureDesc(&d192)
		ctx.EnsureDescsTogether(&d194, &d192)
		if d194.Loc == scm.LocImm && d192.Loc == scm.LocImm {
			d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d194.Imm.Int()) << uint64(d192.Imm.Int())))}
		} else if d192.Loc == scm.LocImm {
			r159 := ctx.AllocRegExcept(d194.Reg)
			ctx.EmitMovRegReg(r159, d194.Reg)
			ctx.EmitShlRegImm8(r159, uint8(d192.Imm.Int()))
			d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d195)
		} else {
			shiftSrc := d194.Reg
			r160 := ctx.AllocRegExcept(d194.Reg, d192.Reg)
			ctx.EmitMovRegReg(r160, d194.Reg)
			shiftSrc = r160
			ctx.EmitShiftLeft(shiftSrc, d192.Reg, true)
			d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d195)
		}
		if d195.Loc == scm.LocReg && d194.Loc == scm.LocReg && d195.Reg == d194.Reg {
			ctx.TransferReg(d194.Reg)
			d194.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d194)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d191)
		ctx.EnsureDesc(&d191)
		if d191.Loc == scm.LocImm {
			d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d191.Reg)
			ctx.EmitMovRegReg(scratch, d191.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d196)
		}
		if d196.Loc == scm.LocReg && d191.Loc == scm.LocReg && d196.Reg == d191.Reg {
			ctx.TransferReg(d191.Reg)
			d191.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d191)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d196)
		ctx.ReclaimUntrackedRegs()
		d197 = ctx.EmitLoadScalarSliceElement(&d193, &d196, 8, scm.TagInt)
		ctx.FreeDesc(&d196)
		ctx.ReclaimUntrackedRegs()
		d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d192)
		ctx.EnsureDescsTogether(&d198, &d192)
		if d198.Loc == scm.LocImm && d192.Loc == scm.LocImm {
			d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() - d192.Imm.Int())}
		} else if d192.Loc == scm.LocImm && d192.Imm.Int() == 0 {
			ctx.EnsureDesc(&d198)
			r161 := ctx.AllocRegExcept(d198.Reg)
			ctx.EmitMovRegReg(r161, d198.Reg)
			d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d199)
		} else if d198.Loc == scm.LocImm {
			ctx.EnsureDesc(&d192)
			scratch := ctx.AllocRegExcept(d192.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d198.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d192)
			d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d199)
		} else if d192.Loc == scm.LocImm {
			ctx.EnsureDesc(&d198)
			scratch := ctx.AllocRegExcept(d198.Reg)
			ctx.EmitMovRegReg(scratch, d198.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d192.Imm.Int())
			d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d199)
		} else {
			ctx.EnsureDesc(&d198)
			ctx.SyncDesc(&d192)
			r162 := ctx.AllocRegExcept(d198.Reg, d192.Reg)
			ctx.EmitMovRegReg(r162, d198.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r162, &d192)
			d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
			ctx.BindReg(r162, &d199)
		}
		if d199.Loc == scm.LocReg && d198.Loc == scm.LocReg && d199.Reg == d198.Reg {
			ctx.TransferReg(d198.Reg)
			d198.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d192)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d197)
		ctx.EnsureDesc(&d199)
		ctx.EnsureDescsTogether(&d197, &d199)
		if d197.Loc == scm.LocImm && d199.Loc == scm.LocImm {
			d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d197.Imm.Int()) >> uint64(d199.Imm.Int())))}
		} else if d199.Loc == scm.LocImm {
			r163 := ctx.AllocRegExcept(d197.Reg)
			ctx.EmitMovRegReg(r163, d197.Reg)
			ctx.EmitShrRegImm8(r163, uint8(d199.Imm.Int()))
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
			ctx.BindReg(r163, &d200)
		} else {
			shiftSrc := d197.Reg
			r164 := ctx.AllocRegExcept(d197.Reg, d199.Reg)
			ctx.EmitMovRegReg(r164, d197.Reg)
			shiftSrc = r164
			ctx.EmitShiftRight(shiftSrc, d199.Reg, false)
			d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d200)
		}
		if d200.Loc == scm.LocReg && d197.Loc == scm.LocReg && d200.Reg == d197.Reg {
			ctx.TransferReg(d197.Reg)
			d197.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d197)
		ctx.FreeDesc(&d199)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d195)
		ctx.EnsureDesc(&d200)
		if d195.Loc == scm.LocImm && d200.Loc == scm.LocImm {
			d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d195.Imm.Int() | d200.Imm.Int())}
		} else if d195.Loc == scm.LocImm && d195.Imm.Int() == 0 {
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
			ctx.BindReg(d200.Reg, &d201)
		} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
			r165 := ctx.AllocRegExcept(d195.Reg)
			ctx.EmitMovRegReg(r165, d195.Reg)
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d201)
		} else if d195.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d200.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d195.Imm.Int()))
			ctx.EmitOrInt64(scratch, d200.Reg)
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d201)
		} else if d200.Loc == scm.LocImm {
			r166 := ctx.AllocRegExcept(d195.Reg)
			ctx.EmitMovRegReg(r166, d195.Reg)
			if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r166, int32(d200.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d200.Imm.Int()))
				ctx.EmitOrInt64(r166, ctx.ScratchReg)
			}
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d201)
		} else {
			r167 := ctx.AllocRegExcept(d195.Reg, d200.Reg)
			ctx.EmitMovRegReg(r167, d195.Reg)
			ctx.EmitOrInt64(r167, d200.Reg)
			d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d201)
		}
		if d201.Loc == scm.LocReg && d195.Loc == scm.LocReg && d201.Reg == d195.Reg {
			ctx.TransferReg(d195.Reg)
			d195.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d195)
		ctx.FreeDesc(&d200)
		ctx.ReclaimUntrackedRegs()
		d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d188)
		ctx.EnsureDescsTogether(&d202, &d188)
		if d202.Loc == scm.LocImm && d188.Loc == scm.LocImm {
			d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() - d188.Imm.Int())}
		} else if d188.Loc == scm.LocImm && d188.Imm.Int() == 0 {
			ctx.EnsureDesc(&d202)
			r168 := ctx.AllocRegExcept(d202.Reg)
			ctx.EmitMovRegReg(r168, d202.Reg)
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d203)
		} else if d202.Loc == scm.LocImm {
			ctx.EnsureDesc(&d188)
			scratch := ctx.AllocRegExcept(d188.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d188)
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d203)
		} else if d188.Loc == scm.LocImm {
			ctx.EnsureDesc(&d202)
			scratch := ctx.AllocRegExcept(d202.Reg)
			ctx.EmitMovRegReg(scratch, d202.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d188.Imm.Int())
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d203)
		} else {
			ctx.EnsureDesc(&d202)
			ctx.SyncDesc(&d188)
			r169 := ctx.AllocRegExcept(d202.Reg, d188.Reg)
			ctx.EmitMovRegReg(r169, d202.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r169, &d188)
			d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d203)
		}
		if d203.Loc == scm.LocReg && d202.Loc == scm.LocReg && d203.Reg == d202.Reg {
			ctx.TransferReg(d202.Reg)
			d202.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d188)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d201)
		ctx.EnsureDesc(&d203)
		ctx.EnsureDescsTogether(&d201, &d203)
		if d201.Loc == scm.LocImm && d203.Loc == scm.LocImm {
			d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d201.Imm.Int()) >> uint64(d203.Imm.Int())))}
		} else if d203.Loc == scm.LocImm {
			r170 := ctx.AllocRegExcept(d201.Reg)
			ctx.EmitMovRegReg(r170, d201.Reg)
			ctx.EmitShrRegImm8(r170, uint8(d203.Imm.Int()))
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			ctx.BindReg(r170, &d204)
		} else {
			shiftSrc := d201.Reg
			r171 := ctx.AllocRegExcept(d201.Reg, d203.Reg)
			ctx.EmitMovRegReg(r171, d201.Reg)
			shiftSrc = r171
			ctx.EmitShiftRight(shiftSrc, d203.Reg, false)
			d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d204)
		}
		if d204.Loc == scm.LocReg && d201.Loc == scm.LocReg && d204.Reg == d201.Reg {
			ctx.TransferReg(d201.Reg)
			d201.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d201)
		ctx.FreeDesc(&d203)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d204)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d204)
		ctx.EnsureDesc(&d204)
		if d204.Loc == scm.LocImm {
			d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d204.Imm.Int()))))}
		} else {
			r172 := ctx.AllocReg()
			ctx.EmitMovRegReg(r172, d204.Reg)
			d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d205)
		}
		ctx.FreeDesc(&d204)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 56)
			r173 := ctx.AllocReg()
			ctx.EmitMovRegMem(r173, thisptr.Reg, off)
			d206 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
			ctx.BindReg(r173, &d206)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d205)
		ctx.EnsureDesc(&d206)
		ctx.EnsureDescsTogether(&d205, &d206)
		if d205.Loc == scm.LocImm && d206.Loc == scm.LocImm {
			d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d205.Imm.Int() + d206.Imm.Int())}
		} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
			ctx.EnsureDesc(&d205)
			r174 := ctx.AllocRegExcept(d205.Reg)
			ctx.EmitMovRegReg(r174, d205.Reg)
			d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
			ctx.BindReg(r174, &d207)
		} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
			ctx.EnsureDesc(&d206)
			d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d206.Reg}
			ctx.BindReg(d206.Reg, &d207)
		} else if d205.Loc == scm.LocImm {
			ctx.EnsureDesc(&d206)
			scratch := ctx.AllocRegExcept(d206.Reg)
			ctx.EmitMovRegReg(scratch, d206.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d205.Imm.Int())
			d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d207)
		} else if d206.Loc == scm.LocImm {
			ctx.EnsureDesc(&d205)
			scratch := ctx.AllocRegExcept(d205.Reg)
			ctx.EmitMovRegReg(scratch, d205.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d206.Imm.Int())
			d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d207)
		} else {
			ctx.EnsureDesc(&d205)
			ctx.SyncDesc(&d206)
			r175 := ctx.AllocRegExcept(d205.Reg, d206.Reg)
			ctx.EmitMovRegReg(r175, d205.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r175, &d206)
			d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
			ctx.BindReg(r175, &d207)
		}
		if d207.Loc == scm.LocReg && d205.Loc == scm.LocReg && d207.Reg == d205.Reg {
			ctx.TransferReg(d205.Reg)
			d205.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d205)
		ctx.FreeDesc(&d206)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d207)
		ctx.EnsureDesc(&d207)
		ctx.StabilizeDescForControlFlow(&d207)
		inlineHome209 := d207
		inlineHome209.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d183)
		if d183.Loc == scm.LocReg || d183.Loc == scm.LocFPReg {
			ctx.ProtectReg(d183.Reg)
		} else if d183.Loc == scm.LocRegPair {
			ctx.ProtectReg(d183.Reg)
			ctx.ProtectReg(d183.Reg2)
		}
		ctx.SyncDesc(&d207)
		if d207.Loc == scm.LocReg || d207.Loc == scm.LocFPReg {
			ctx.ProtectReg(d207.Reg)
		} else if d207.Loc == scm.LocRegPair {
			ctx.ProtectReg(d207.Reg)
			ctx.ProtectReg(d207.Reg2)
		}
		d210 = d183
		if d210.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d210)
		ctx.EmitStoreToStack(d210, int32(phiBase127)+int32(0))
		d211 = d207
		if d211.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d211)
		ctx.EmitStoreToStack(d211, int32(phiBase127)+int32(16))
		if d183.Loc == scm.LocReg || d183.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d183.Reg)
		} else if d183.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d183.Reg)
			ctx.UnprotectReg(d183.Reg2)
		}
		if d207.Loc == scm.LocReg || d207.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d207.Reg)
		} else if d207.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d207.Reg)
			ctx.UnprotectReg(d207.Reg2)
		}
		bbpos_8_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl38)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d128)
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d129)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r176 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r176, thisptr.Reg, off)
			d212 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r176}
			ctx.BindReg(r176, &d212)
		}
		ctx.StabilizeDescForControlFlow(&d212)
		inlineHome213 := d212
		inlineHome213.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x0))}
		} else {
			r177 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 0)
			d214 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r177, Condition: scm.CondEqual}
			ctx.BindReg(r177, &d214)
		}
		ctx.ReclaimUntrackedRegs()
		d215 = d214
		ctx.SyncDesc(&d215)
		if d215.Loc != scm.LocImm && d215.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl86 := ctx.ReserveLabel()
		lbl87 := ctx.ReserveLabel()
		if d215.Loc == scm.LocImm {
			if d215.Imm.Bool() {
				ctx.MarkLabel(lbl86)
				ctx.EmitJmp(lbl46)
			} else {
				ctx.MarkLabel(lbl87)
				ctx.EmitJmp(lbl48)
			}
		} else {
			ctx.EmitJump(d215.Condition, lbl86)
			ctx.EmitJmp(lbl87)
			ctx.FreeDesc(&d214)
			ctx.MarkLabel(lbl86)
			ctx.EmitJmp(lbl46)
			ctx.MarkLabel(lbl87)
			ctx.EmitJmp(lbl48)
		}
		bbpos_8_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl48)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x2))}
		} else {
			r178 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 2)
			d216 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r178, Condition: scm.CondEqual}
			ctx.BindReg(r178, &d216)
		}
		ctx.ReclaimUntrackedRegs()
		d217 = d216
		ctx.SyncDesc(&d217)
		if d217.Loc != scm.LocImm && d217.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl88 := ctx.ReserveLabel()
		lbl89 := ctx.ReserveLabel()
		if d217.Loc == scm.LocImm {
			if d217.Imm.Bool() {
				ctx.MarkLabel(lbl88)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl89)
				ctx.EmitJmp(lbl50)
			}
		} else {
			ctx.EmitJump(d217.Condition, lbl88)
			ctx.EmitJmp(lbl89)
			ctx.FreeDesc(&d216)
			ctx.MarkLabel(lbl88)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl89)
			ctx.EmitJmp(lbl50)
		}
		bbpos_8_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl50)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x3))}
		} else {
			r179 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 3)
			d218 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r179, Condition: scm.CondEqual}
			ctx.BindReg(r179, &d218)
		}
		ctx.ReclaimUntrackedRegs()
		d219 = d218
		ctx.SyncDesc(&d219)
		if d219.Loc != scm.LocImm && d219.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl90 := ctx.ReserveLabel()
		lbl91 := ctx.ReserveLabel()
		if d219.Loc == scm.LocImm {
			if d219.Imm.Bool() {
				ctx.MarkLabel(lbl90)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl91)
				ctx.EmitJmp(lbl51)
			}
		} else {
			ctx.EmitJump(d219.Condition, lbl90)
			ctx.EmitJmp(lbl91)
			ctx.FreeDesc(&d218)
			ctx.MarkLabel(lbl90)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl91)
			ctx.EmitJmp(lbl51)
		}
		bbpos_8_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl51)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x1))}
		} else {
			r180 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 1)
			d220 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r180, Condition: scm.CondEqual}
			ctx.BindReg(r180, &d220)
		}
		ctx.ReclaimUntrackedRegs()
		d221 = d220
		ctx.SyncDesc(&d221)
		if d221.Loc != scm.LocImm && d221.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl92 := ctx.ReserveLabel()
		lbl93 := ctx.ReserveLabel()
		if d221.Loc == scm.LocImm {
			if d221.Imm.Bool() {
				ctx.MarkLabel(lbl92)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl93)
				ctx.EmitJmp(lbl52)
			}
		} else {
			ctx.EmitJump(d221.Condition, lbl92)
			ctx.EmitJmp(lbl93)
			ctx.FreeDesc(&d220)
			ctx.MarkLabel(lbl92)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl93)
			ctx.EmitJmp(lbl52)
		}
		bbpos_8_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl52)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xa))}
		} else {
			r181 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 10)
			d222 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r181, Condition: scm.CondEqual}
			ctx.BindReg(r181, &d222)
		}
		ctx.ReclaimUntrackedRegs()
		d223 = d222
		ctx.SyncDesc(&d223)
		if d223.Loc != scm.LocImm && d223.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl94 := ctx.ReserveLabel()
		lbl95 := ctx.ReserveLabel()
		if d223.Loc == scm.LocImm {
			if d223.Imm.Bool() {
				ctx.MarkLabel(lbl94)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl95)
				ctx.EmitJmp(lbl53)
			}
		} else {
			ctx.EmitJump(d223.Condition, lbl94)
			ctx.EmitJmp(lbl95)
			ctx.FreeDesc(&d222)
			ctx.MarkLabel(lbl94)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl95)
			ctx.EmitJmp(lbl53)
		}
		bbpos_8_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl53)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x8))}
		} else {
			r182 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 8)
			d224 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r182, Condition: scm.CondEqual}
			ctx.BindReg(r182, &d224)
		}
		ctx.ReclaimUntrackedRegs()
		d225 = d224
		ctx.SyncDesc(&d225)
		if d225.Loc != scm.LocImm && d225.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl96 := ctx.ReserveLabel()
		lbl97 := ctx.ReserveLabel()
		if d225.Loc == scm.LocImm {
			if d225.Imm.Bool() {
				ctx.MarkLabel(lbl96)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl97)
				ctx.EmitJmp(lbl54)
			}
		} else {
			ctx.EmitJump(d225.Condition, lbl96)
			ctx.EmitJmp(lbl97)
			ctx.FreeDesc(&d224)
			ctx.MarkLabel(lbl96)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl97)
			ctx.EmitJmp(lbl54)
		}
		bbpos_8_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl54)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x9))}
		} else {
			r183 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 9)
			d226 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r183, Condition: scm.CondEqual}
			ctx.BindReg(r183, &d226)
		}
		ctx.ReclaimUntrackedRegs()
		d227 = d226
		ctx.SyncDesc(&d227)
		if d227.Loc != scm.LocImm && d227.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl98 := ctx.ReserveLabel()
		lbl99 := ctx.ReserveLabel()
		if d227.Loc == scm.LocImm {
			if d227.Imm.Bool() {
				ctx.MarkLabel(lbl98)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl99)
				ctx.EmitJmp(lbl55)
			}
		} else {
			ctx.EmitJump(d227.Condition, lbl98)
			ctx.EmitJmp(lbl99)
			ctx.FreeDesc(&d226)
			ctx.MarkLabel(lbl98)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl99)
			ctx.EmitJmp(lbl55)
		}
		bbpos_8_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl55)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xb))}
		} else {
			r184 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 11)
			d228 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r184, Condition: scm.CondEqual}
			ctx.BindReg(r184, &d228)
		}
		ctx.ReclaimUntrackedRegs()
		d229 = d228
		ctx.SyncDesc(&d229)
		if d229.Loc != scm.LocImm && d229.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl100 := ctx.ReserveLabel()
		lbl101 := ctx.ReserveLabel()
		if d229.Loc == scm.LocImm {
			if d229.Imm.Bool() {
				ctx.MarkLabel(lbl100)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl101)
				ctx.EmitJmp(lbl56)
			}
		} else {
			ctx.EmitJump(d229.Condition, lbl100)
			ctx.EmitJmp(lbl101)
			ctx.FreeDesc(&d228)
			ctx.MarkLabel(lbl100)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl101)
			ctx.EmitJmp(lbl56)
		}
		bbpos_8_20 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl56)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xc))}
		} else {
			r185 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 12)
			d230 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r185, Condition: scm.CondEqual}
			ctx.BindReg(r185, &d230)
		}
		ctx.ReclaimUntrackedRegs()
		d231 = d230
		ctx.SyncDesc(&d231)
		if d231.Loc != scm.LocImm && d231.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl102 := ctx.ReserveLabel()
		lbl103 := ctx.ReserveLabel()
		if d231.Loc == scm.LocImm {
			if d231.Imm.Bool() {
				ctx.MarkLabel(lbl102)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl103)
				ctx.EmitJmp(lbl57)
			}
		} else {
			ctx.EmitJump(d231.Condition, lbl102)
			ctx.EmitJmp(lbl103)
			ctx.FreeDesc(&d230)
			ctx.MarkLabel(lbl102)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl103)
			ctx.EmitJmp(lbl57)
		}
		bbpos_8_21 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl57)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xd))}
		} else {
			r186 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 13)
			d232 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r186, Condition: scm.CondEqual}
			ctx.BindReg(r186, &d232)
		}
		ctx.ReclaimUntrackedRegs()
		d233 = d232
		ctx.SyncDesc(&d233)
		if d233.Loc != scm.LocImm && d233.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl104 := ctx.ReserveLabel()
		lbl105 := ctx.ReserveLabel()
		if d233.Loc == scm.LocImm {
			if d233.Imm.Bool() {
				ctx.MarkLabel(lbl104)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl105)
				ctx.EmitJmp(lbl58)
			}
		} else {
			ctx.EmitJump(d233.Condition, lbl104)
			ctx.EmitJmp(lbl105)
			ctx.FreeDesc(&d232)
			ctx.MarkLabel(lbl104)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl105)
			ctx.EmitJmp(lbl58)
		}
		bbpos_8_22 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl58)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xe))}
		} else {
			r187 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 14)
			d234 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r187, Condition: scm.CondEqual}
			ctx.BindReg(r187, &d234)
		}
		ctx.ReclaimUntrackedRegs()
		d235 = d234
		ctx.SyncDesc(&d235)
		if d235.Loc != scm.LocImm && d235.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl106 := ctx.ReserveLabel()
		lbl107 := ctx.ReserveLabel()
		if d235.Loc == scm.LocImm {
			if d235.Imm.Bool() {
				ctx.MarkLabel(lbl106)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl107)
				ctx.EmitJmp(lbl59)
			}
		} else {
			ctx.EmitJump(d235.Condition, lbl106)
			ctx.EmitJmp(lbl107)
			ctx.FreeDesc(&d234)
			ctx.MarkLabel(lbl106)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl107)
			ctx.EmitJmp(lbl59)
		}
		bbpos_8_23 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl59)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0xf))}
		} else {
			r188 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 15)
			d236 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r188, Condition: scm.CondEqual}
			ctx.BindReg(r188, &d236)
		}
		ctx.ReclaimUntrackedRegs()
		d237 = d236
		ctx.SyncDesc(&d237)
		if d237.Loc != scm.LocImm && d237.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl108 := ctx.ReserveLabel()
		lbl109 := ctx.ReserveLabel()
		if d237.Loc == scm.LocImm {
			if d237.Imm.Bool() {
				ctx.MarkLabel(lbl108)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl109)
				ctx.EmitJmp(lbl60)
			}
		} else {
			ctx.EmitJump(d237.Condition, lbl108)
			ctx.EmitJmp(lbl109)
			ctx.FreeDesc(&d236)
			ctx.MarkLabel(lbl108)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl109)
			ctx.EmitJmp(lbl60)
		}
		bbpos_8_24 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl60)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x10))}
		} else {
			r189 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 16)
			d238 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r189, Condition: scm.CondEqual}
			ctx.BindReg(r189, &d238)
		}
		ctx.ReclaimUntrackedRegs()
		d239 = d238
		ctx.SyncDesc(&d239)
		if d239.Loc != scm.LocImm && d239.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl110 := ctx.ReserveLabel()
		lbl111 := ctx.ReserveLabel()
		if d239.Loc == scm.LocImm {
			if d239.Imm.Bool() {
				ctx.MarkLabel(lbl110)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl111)
				ctx.EmitJmp(lbl61)
			}
		} else {
			ctx.EmitJump(d239.Condition, lbl110)
			ctx.EmitJmp(lbl111)
			ctx.FreeDesc(&d238)
			ctx.MarkLabel(lbl110)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl111)
			ctx.EmitJmp(lbl61)
		}
		bbpos_8_25 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl61)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x13))}
		} else {
			r190 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 19)
			d240 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r190, Condition: scm.CondEqual}
			ctx.BindReg(r190, &d240)
		}
		ctx.ReclaimUntrackedRegs()
		d241 = d240
		ctx.SyncDesc(&d241)
		if d241.Loc != scm.LocImm && d241.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl112 := ctx.ReserveLabel()
		lbl113 := ctx.ReserveLabel()
		if d241.Loc == scm.LocImm {
			if d241.Imm.Bool() {
				ctx.MarkLabel(lbl112)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl113)
				ctx.EmitJmp(lbl62)
			}
		} else {
			ctx.EmitJump(d241.Condition, lbl112)
			ctx.EmitJmp(lbl113)
			ctx.FreeDesc(&d240)
			ctx.MarkLabel(lbl112)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl113)
			ctx.EmitJmp(lbl62)
		}
		bbpos_8_26 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl62)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x14))}
		} else {
			r191 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 20)
			d242 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r191, Condition: scm.CondEqual}
			ctx.BindReg(r191, &d242)
		}
		ctx.ReclaimUntrackedRegs()
		d243 = d242
		ctx.SyncDesc(&d243)
		if d243.Loc != scm.LocImm && d243.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl114 := ctx.ReserveLabel()
		lbl115 := ctx.ReserveLabel()
		if d243.Loc == scm.LocImm {
			if d243.Imm.Bool() {
				ctx.MarkLabel(lbl114)
				ctx.EmitJmp(lbl47)
			} else {
				ctx.MarkLabel(lbl115)
				ctx.EmitJmp(lbl63)
			}
		} else {
			ctx.EmitJump(d243.Condition, lbl114)
			ctx.EmitJmp(lbl115)
			ctx.FreeDesc(&d242)
			ctx.MarkLabel(lbl114)
			ctx.EmitJmp(lbl47)
			ctx.MarkLabel(lbl115)
			ctx.EmitJmp(lbl63)
		}
		bbpos_8_27 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl63)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x6))}
		} else {
			r192 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 6)
			d244 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r192, Condition: scm.CondEqual}
			ctx.BindReg(r192, &d244)
		}
		ctx.ReclaimUntrackedRegs()
		d245 = d244
		ctx.SyncDesc(&d245)
		if d245.Loc != scm.LocImm && d245.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl116 := ctx.ReserveLabel()
		lbl117 := ctx.ReserveLabel()
		if d245.Loc == scm.LocImm {
			if d245.Imm.Bool() {
				ctx.MarkLabel(lbl116)
				ctx.EmitJmp(lbl49)
			} else {
				ctx.MarkLabel(lbl117)
				ctx.EmitJmp(lbl67)
			}
		} else {
			ctx.EmitJump(d245.Condition, lbl116)
			ctx.EmitJmp(lbl117)
			ctx.FreeDesc(&d244)
			ctx.MarkLabel(lbl116)
			ctx.EmitJmp(lbl49)
			ctx.MarkLabel(lbl117)
			ctx.EmitJmp(lbl67)
		}
		bbpos_8_31 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl67)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x7))}
		} else {
			r193 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 7)
			d246 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r193, Condition: scm.CondEqual}
			ctx.BindReg(r193, &d246)
		}
		ctx.ReclaimUntrackedRegs()
		d247 = d246
		ctx.SyncDesc(&d247)
		if d247.Loc != scm.LocImm && d247.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl118 := ctx.ReserveLabel()
		lbl119 := ctx.ReserveLabel()
		if d247.Loc == scm.LocImm {
			if d247.Imm.Bool() {
				ctx.MarkLabel(lbl118)
				ctx.EmitJmp(lbl49)
			} else {
				ctx.MarkLabel(lbl119)
				ctx.EmitJmp(lbl68)
			}
		} else {
			ctx.EmitJump(d247.Condition, lbl118)
			ctx.EmitJmp(lbl119)
			ctx.FreeDesc(&d246)
			ctx.MarkLabel(lbl118)
			ctx.EmitJmp(lbl49)
			ctx.MarkLabel(lbl119)
			ctx.EmitJmp(lbl68)
		}
		bbpos_8_32 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl68)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x5))}
		} else {
			r194 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 5)
			d248 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r194, Condition: scm.CondEqual}
			ctx.BindReg(r194, &d248)
		}
		ctx.ReclaimUntrackedRegs()
		d249 = d248
		ctx.SyncDesc(&d249)
		if d249.Loc != scm.LocImm && d249.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl120 := ctx.ReserveLabel()
		lbl121 := ctx.ReserveLabel()
		if d249.Loc == scm.LocImm {
			if d249.Imm.Bool() {
				ctx.MarkLabel(lbl120)
				ctx.EmitJmp(lbl66)
			} else {
				ctx.MarkLabel(lbl121)
				ctx.EmitJmp(lbl69)
			}
		} else {
			ctx.EmitJump(d249.Condition, lbl120)
			ctx.EmitJmp(lbl121)
			ctx.FreeDesc(&d248)
			ctx.MarkLabel(lbl120)
			ctx.EmitJmp(lbl66)
			ctx.MarkLabel(lbl121)
			ctx.EmitJmp(lbl69)
		}
		bbpos_8_33 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl69)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x4))}
		} else {
			r195 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 4)
			d250 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r195, Condition: scm.CondEqual}
			ctx.BindReg(r195, &d250)
		}
		ctx.ReclaimUntrackedRegs()
		d251 = d250
		ctx.SyncDesc(&d251)
		if d251.Loc != scm.LocImm && d251.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl122 := ctx.ReserveLabel()
		lbl123 := ctx.ReserveLabel()
		if d251.Loc == scm.LocImm {
			if d251.Imm.Bool() {
				ctx.MarkLabel(lbl122)
				ctx.EmitJmp(lbl66)
			} else {
				ctx.MarkLabel(lbl123)
				ctx.EmitJmp(lbl70)
			}
		} else {
			ctx.EmitJump(d251.Condition, lbl122)
			ctx.EmitJmp(lbl123)
			ctx.FreeDesc(&d250)
			ctx.MarkLabel(lbl122)
			ctx.EmitJmp(lbl66)
			ctx.MarkLabel(lbl123)
			ctx.EmitJmp(lbl70)
		}
		bbpos_8_34 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl70)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x11))}
		} else {
			r196 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 17)
			d252 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r196, Condition: scm.CondEqual}
			ctx.BindReg(r196, &d252)
		}
		ctx.ReclaimUntrackedRegs()
		d253 = d252
		ctx.SyncDesc(&d253)
		if d253.Loc != scm.LocImm && d253.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl124 := ctx.ReserveLabel()
		lbl125 := ctx.ReserveLabel()
		if d253.Loc == scm.LocImm {
			if d253.Imm.Bool() {
				ctx.MarkLabel(lbl124)
				ctx.EmitJmp(lbl66)
			} else {
				ctx.MarkLabel(lbl125)
				ctx.EmitJmp(lbl71)
			}
		} else {
			ctx.EmitJump(d253.Condition, lbl124)
			ctx.EmitJmp(lbl125)
			ctx.FreeDesc(&d252)
			ctx.MarkLabel(lbl124)
			ctx.EmitJmp(lbl66)
			ctx.MarkLabel(lbl125)
			ctx.EmitJmp(lbl71)
		}
		bbpos_8_35 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl71)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d212)
		if d212.Loc == scm.LocImm {
			d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d212.Imm.Int()) == uint64(0x12))}
		} else {
			r197 := ctx.AllocRegExcept(d212.Reg)
			ctx.EmitCmpRegImm32(d212.Reg, 18)
			d254 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r197, Condition: scm.CondEqual}
			ctx.BindReg(r197, &d254)
		}
		ctx.ReclaimUntrackedRegs()
		d255 = d254
		ctx.SyncDesc(&d255)
		if d255.Loc != scm.LocImm && d255.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl126 := ctx.ReserveLabel()
		lbl127 := ctx.ReserveLabel()
		if d255.Loc == scm.LocImm {
			if d255.Imm.Bool() {
				ctx.MarkLabel(lbl126)
				ctx.EmitJmp(lbl66)
			} else {
				ctx.MarkLabel(lbl127)
				ctx.EmitJmp(lbl72)
			}
		} else {
			ctx.EmitJump(d255.Condition, lbl126)
			ctx.EmitJmp(lbl127)
			ctx.FreeDesc(&d254)
			ctx.MarkLabel(lbl126)
			ctx.EmitJmp(lbl66)
			ctx.MarkLabel(lbl127)
			ctx.EmitJmp(lbl72)
		}
		bbpos_8_36 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl72)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d256)
		ctx.EmitCopyScmerToDesc(&d133, &d256)
		ctx.EmitJmp(lbl35)
		bbpos_8_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl37)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		d257 = d121
		_ = d257
		ctx.StabilizeDescForControlFlow(&d121)
		bbpos_12_0 := int32(-1)
		_ = bbpos_12_0
		lbl128 := ctx.ReserveLabel()
		_ = lbl128
		bbpos_12_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl128)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 48)
			r198 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r198, thisptr.Reg, off)
			d258 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
			ctx.BindReg(r198, &d258)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d258)
		ctx.EnsureDesc(&d258)
		if d258.Loc == scm.LocImm {
			d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d258.Imm.Int()))))}
		} else {
			r199 := ctx.AllocReg()
			ctx.EmitMovRegReg(r199, d258.Reg)
			ctx.EmitShlRegImm8(r199, 56)
			ctx.EmitShrRegImm8(r199, 56)
			d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
			ctx.BindReg(r199, &d259)
		}
		ctx.FreeDesc(&d258)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d257)
		ctx.EnsureDesc(&d257)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d257)
		ctx.EnsureDesc(&d259)
		ctx.EnsureDescsTogether(&d257, &d259)
		if d257.Loc == scm.LocImm && d259.Loc == scm.LocImm {
			d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d257.Imm.Int() * d259.Imm.Int())}
		} else if d257.Loc == scm.LocImm {
			ctx.EnsureDesc(&d259)
			scratch := ctx.AllocRegExcept(d259.Reg)
			ctx.EmitMovRegReg(scratch, d259.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d257.Imm.Int())
			d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d261)
		} else if d259.Loc == scm.LocImm {
			ctx.EnsureDesc(&d257)
			scratch := ctx.AllocRegExcept(d257.Reg)
			ctx.EmitMovRegReg(scratch, d257.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d259.Imm.Int())
			d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d261)
		} else {
			ctx.EnsureDesc(&d257)
			ctx.SyncDesc(&d259)
			r200 := ctx.AllocRegExcept(d257.Reg, d259.Reg)
			ctx.EmitMovRegReg(r200, d257.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r200, &d259)
			d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
			ctx.BindReg(r200, &d261)
		}
		if d261.Loc == scm.LocReg && d257.Loc == scm.LocReg && d261.Reg == d257.Reg {
			ctx.TransferReg(d257.Reg)
			d257.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d261)
		if d261.Loc == scm.LocImm {
			d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d261.Imm.Int() / 64)}
		} else {
			r201 := ctx.AllocRegExcept(d261.Reg)
			ctx.EmitMovRegReg(r201, d261.Reg)
			ctx.EmitShrRegImm8(r201, 6)
			d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
			ctx.BindReg(r201, &d262)
		}
		if d262.Loc == scm.LocReg && d261.Loc == scm.LocReg && d262.Reg == d261.Reg {
			ctx.TransferReg(d261.Reg)
			d261.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d261)
		if d261.Loc == scm.LocImm {
			d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d261.Imm.Int() % 64)}
		} else {
			r202 := ctx.AllocRegExcept(d261.Reg)
			ctx.EmitMovRegReg(r202, d261.Reg)
			ctx.EmitAndRegImm32(r202, 63)
			d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
			ctx.BindReg(r202, &d263)
		}
		if d263.Loc == scm.LocReg && d261.Loc == scm.LocReg && d263.Reg == d261.Reg {
			ctx.TransferReg(d261.Reg)
			d261.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d261)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d264 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r203 := ctx.AllocReg()
			r204 := ctx.AllocRegExcept(r203)
			r205 := ctx.AllocRegExcept(r203, r204)
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
			ctx.EmitMovRegMem(r203, thisptr.Reg, off)
			ctx.EmitMovRegMem(r204, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r205, thisptr.Reg, off+16)
			d264 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r203, Reg2: r204, Reg3: r205}
			ctx.BindReg(r203, &d264)
			ctx.BindReg(r204, &d264)
			ctx.BindReg(r205, &d264)
			ctx.BindReg(r203, &d264)
			ctx.BindReg(r204, &d264)
			ctx.BindReg(r205, &d264)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d262)
		ctx.ReclaimUntrackedRegs()
		d265 = ctx.EmitLoadScalarSliceElement(&d264, &d262, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d265)
		ctx.EnsureDesc(&d263)
		ctx.EnsureDescsTogether(&d265, &d263)
		if d265.Loc == scm.LocImm && d263.Loc == scm.LocImm {
			d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d265.Imm.Int()) << uint64(d263.Imm.Int())))}
		} else if d263.Loc == scm.LocImm {
			r206 := ctx.AllocRegExcept(d265.Reg)
			ctx.EmitMovRegReg(r206, d265.Reg)
			ctx.EmitShlRegImm8(r206, uint8(d263.Imm.Int()))
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
			ctx.BindReg(r206, &d266)
		} else {
			shiftSrc := d265.Reg
			r207 := ctx.AllocRegExcept(d265.Reg, d263.Reg)
			ctx.EmitMovRegReg(r207, d265.Reg)
			shiftSrc = r207
			ctx.EmitShiftLeft(shiftSrc, d263.Reg, true)
			d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d266)
		}
		if d266.Loc == scm.LocReg && d265.Loc == scm.LocReg && d266.Reg == d265.Reg {
			ctx.TransferReg(d265.Reg)
			d265.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d265)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d262)
		ctx.EnsureDesc(&d262)
		if d262.Loc == scm.LocImm {
			d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d262.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d262.Reg)
			ctx.EmitMovRegReg(scratch, d262.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d267)
		}
		if d267.Loc == scm.LocReg && d262.Loc == scm.LocReg && d267.Reg == d262.Reg {
			ctx.TransferReg(d262.Reg)
			d262.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d262)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d267)
		ctx.ReclaimUntrackedRegs()
		d268 = ctx.EmitLoadScalarSliceElement(&d264, &d267, 8, scm.TagInt)
		ctx.FreeDesc(&d267)
		ctx.ReclaimUntrackedRegs()
		d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d263)
		ctx.EnsureDescsTogether(&d269, &d263)
		if d269.Loc == scm.LocImm && d263.Loc == scm.LocImm {
			d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d269.Imm.Int() - d263.Imm.Int())}
		} else if d263.Loc == scm.LocImm && d263.Imm.Int() == 0 {
			ctx.EnsureDesc(&d269)
			r208 := ctx.AllocRegExcept(d269.Reg)
			ctx.EmitMovRegReg(r208, d269.Reg)
			d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
			ctx.BindReg(r208, &d270)
		} else if d269.Loc == scm.LocImm {
			ctx.EnsureDesc(&d263)
			scratch := ctx.AllocRegExcept(d263.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d269.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d263)
			d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d270)
		} else if d263.Loc == scm.LocImm {
			ctx.EnsureDesc(&d269)
			scratch := ctx.AllocRegExcept(d269.Reg)
			ctx.EmitMovRegReg(scratch, d269.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d263.Imm.Int())
			d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d270)
		} else {
			ctx.EnsureDesc(&d269)
			ctx.SyncDesc(&d263)
			r209 := ctx.AllocRegExcept(d269.Reg, d263.Reg)
			ctx.EmitMovRegReg(r209, d269.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r209, &d263)
			d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
			ctx.BindReg(r209, &d270)
		}
		if d270.Loc == scm.LocReg && d269.Loc == scm.LocReg && d270.Reg == d269.Reg {
			ctx.TransferReg(d269.Reg)
			d269.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d263)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d268)
		ctx.EnsureDesc(&d270)
		ctx.EnsureDescsTogether(&d268, &d270)
		if d268.Loc == scm.LocImm && d270.Loc == scm.LocImm {
			d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d268.Imm.Int()) >> uint64(d270.Imm.Int())))}
		} else if d270.Loc == scm.LocImm {
			r210 := ctx.AllocRegExcept(d268.Reg)
			ctx.EmitMovRegReg(r210, d268.Reg)
			ctx.EmitShrRegImm8(r210, uint8(d270.Imm.Int()))
			d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
			ctx.BindReg(r210, &d271)
		} else {
			shiftSrc := d268.Reg
			r211 := ctx.AllocRegExcept(d268.Reg, d270.Reg)
			ctx.EmitMovRegReg(r211, d268.Reg)
			shiftSrc = r211
			ctx.EmitShiftRight(shiftSrc, d270.Reg, false)
			d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d271)
		}
		if d271.Loc == scm.LocReg && d268.Loc == scm.LocReg && d271.Reg == d268.Reg {
			ctx.TransferReg(d268.Reg)
			d268.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d268)
		ctx.FreeDesc(&d270)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d266)
		ctx.EnsureDesc(&d271)
		if d266.Loc == scm.LocImm && d271.Loc == scm.LocImm {
			d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d266.Imm.Int() | d271.Imm.Int())}
		} else if d266.Loc == scm.LocImm && d266.Imm.Int() == 0 {
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d271.Reg}
			ctx.BindReg(d271.Reg, &d272)
		} else if d271.Loc == scm.LocImm && d271.Imm.Int() == 0 {
			r212 := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(r212, d266.Reg)
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
			ctx.BindReg(r212, &d272)
		} else if d266.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d271.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d266.Imm.Int()))
			ctx.EmitOrInt64(scratch, d271.Reg)
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d272)
		} else if d271.Loc == scm.LocImm {
			r213 := ctx.AllocRegExcept(d266.Reg)
			ctx.EmitMovRegReg(r213, d266.Reg)
			if d271.Imm.Int() >= -2147483648 && d271.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r213, int32(d271.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d271.Imm.Int()))
				ctx.EmitOrInt64(r213, ctx.ScratchReg)
			}
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
			ctx.BindReg(r213, &d272)
		} else {
			r214 := ctx.AllocRegExcept(d266.Reg, d271.Reg)
			ctx.EmitMovRegReg(r214, d266.Reg)
			ctx.EmitOrInt64(r214, d271.Reg)
			d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
			ctx.BindReg(r214, &d272)
		}
		if d272.Loc == scm.LocReg && d266.Loc == scm.LocReg && d272.Reg == d266.Reg {
			ctx.TransferReg(d266.Reg)
			d266.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d266)
		ctx.FreeDesc(&d271)
		ctx.ReclaimUntrackedRegs()
		d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d259)
		ctx.EnsureDescsTogether(&d273, &d259)
		if d273.Loc == scm.LocImm && d259.Loc == scm.LocImm {
			d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d273.Imm.Int() - d259.Imm.Int())}
		} else if d259.Loc == scm.LocImm && d259.Imm.Int() == 0 {
			ctx.EnsureDesc(&d273)
			r215 := ctx.AllocRegExcept(d273.Reg)
			ctx.EmitMovRegReg(r215, d273.Reg)
			d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
			ctx.BindReg(r215, &d274)
		} else if d273.Loc == scm.LocImm {
			ctx.EnsureDesc(&d259)
			scratch := ctx.AllocRegExcept(d259.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d273.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d259)
			d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d274)
		} else if d259.Loc == scm.LocImm {
			ctx.EnsureDesc(&d273)
			scratch := ctx.AllocRegExcept(d273.Reg)
			ctx.EmitMovRegReg(scratch, d273.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d259.Imm.Int())
			d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d274)
		} else {
			ctx.EnsureDesc(&d273)
			ctx.SyncDesc(&d259)
			r216 := ctx.AllocRegExcept(d273.Reg, d259.Reg)
			ctx.EmitMovRegReg(r216, d273.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r216, &d259)
			d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
			ctx.BindReg(r216, &d274)
		}
		if d274.Loc == scm.LocReg && d273.Loc == scm.LocReg && d274.Reg == d273.Reg {
			ctx.TransferReg(d273.Reg)
			d273.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d259)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d272)
		ctx.EnsureDesc(&d274)
		ctx.EnsureDescsTogether(&d272, &d274)
		if d272.Loc == scm.LocImm && d274.Loc == scm.LocImm {
			d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d272.Imm.Int()) >> uint64(d274.Imm.Int())))}
		} else if d274.Loc == scm.LocImm {
			r217 := ctx.AllocRegExcept(d272.Reg)
			ctx.EmitMovRegReg(r217, d272.Reg)
			ctx.EmitShrRegImm8(r217, uint8(d274.Imm.Int()))
			d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
			ctx.BindReg(r217, &d275)
		} else {
			shiftSrc := d272.Reg
			r218 := ctx.AllocRegExcept(d272.Reg, d274.Reg)
			ctx.EmitMovRegReg(r218, d272.Reg)
			shiftSrc = r218
			ctx.EmitShiftRight(shiftSrc, d274.Reg, false)
			d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d275)
		}
		if d275.Loc == scm.LocReg && d272.Loc == scm.LocReg && d275.Reg == d272.Reg {
			ctx.TransferReg(d272.Reg)
			d272.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d272)
		ctx.FreeDesc(&d274)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d275)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d275)
		ctx.EnsureDesc(&d275)
		if d275.Loc == scm.LocImm {
			d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d275.Imm.Int()))))}
		} else {
			r219 := ctx.AllocReg()
			ctx.EmitMovRegReg(r219, d275.Reg)
			d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
			ctx.BindReg(r219, &d276)
		}
		ctx.FreeDesc(&d275)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
			r220 := ctx.AllocReg()
			ctx.EmitMovRegMem(r220, thisptr.Reg, off)
			d277 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r220}
			ctx.BindReg(r220, &d277)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d276)
		ctx.EnsureDesc(&d277)
		ctx.EnsureDescsTogether(&d276, &d277)
		if d276.Loc == scm.LocImm && d277.Loc == scm.LocImm {
			d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d276.Imm.Int() + d277.Imm.Int())}
		} else if d277.Loc == scm.LocImm && d277.Imm.Int() == 0 {
			ctx.EnsureDesc(&d276)
			r221 := ctx.AllocRegExcept(d276.Reg)
			ctx.EmitMovRegReg(r221, d276.Reg)
			d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
			ctx.BindReg(r221, &d278)
		} else if d276.Loc == scm.LocImm && d276.Imm.Int() == 0 {
			ctx.EnsureDesc(&d277)
			d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d277.Reg}
			ctx.BindReg(d277.Reg, &d278)
		} else if d276.Loc == scm.LocImm {
			ctx.EnsureDesc(&d277)
			scratch := ctx.AllocRegExcept(d277.Reg)
			ctx.EmitMovRegReg(scratch, d277.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d276.Imm.Int())
			d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d278)
		} else if d277.Loc == scm.LocImm {
			ctx.EnsureDesc(&d276)
			scratch := ctx.AllocRegExcept(d276.Reg)
			ctx.EmitMovRegReg(scratch, d276.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d277.Imm.Int())
			d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d278)
		} else {
			ctx.EnsureDesc(&d276)
			ctx.SyncDesc(&d277)
			r222 := ctx.AllocRegExcept(d276.Reg, d277.Reg)
			ctx.EmitMovRegReg(r222, d276.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r222, &d277)
			d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
			ctx.BindReg(r222, &d278)
		}
		if d278.Loc == scm.LocReg && d276.Loc == scm.LocReg && d278.Reg == d276.Reg {
			ctx.TransferReg(d276.Reg)
			d276.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d276)
		ctx.FreeDesc(&d277)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d278)
		ctx.EnsureDesc(&d278)
		ctx.StabilizeDescForControlFlow(&d278)
		inlineHome280 := d278
		inlineHome280.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 80)
			r223 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r223, thisptr.Reg, off)
			d281 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r223}
			ctx.BindReg(r223, &d281)
		}
		ctx.ReclaimUntrackedRegs()
		d282 = d281
		ctx.EnsureDesc(&d282)
		if d282.Loc != scm.LocImm && d282.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl129 := ctx.ReserveLabel()
		lbl130 := ctx.ReserveLabel()
		if d282.Loc == scm.LocImm {
			if d282.Imm.Bool() {
				ctx.MarkLabel(lbl129)
				ctx.EmitJmp(lbl42)
			} else {
				ctx.MarkLabel(lbl130)
				ctx.EmitJmp(lbl41)
			}
		} else {
			ctx.EmitCmpRegImm32(d282.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl129)
			ctx.EmitJmp(lbl130)
			ctx.MarkLabel(lbl129)
			ctx.EmitJmp(lbl42)
			ctx.MarkLabel(lbl130)
			ctx.EmitJmp(lbl41)
		}
		ctx.FreeDesc(&d281)
		bbpos_8_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl41)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		d283 = d121
		_ = d283
		bbpos_13_0 := int32(-1)
		_ = bbpos_13_0
		lbl131 := ctx.ReserveLabel()
		_ = lbl131
		bbpos_13_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl131)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 48)
			r224 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r224, thisptr.Reg, off)
			d284 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r224}
			ctx.BindReg(r224, &d284)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d284)
		ctx.EnsureDesc(&d284)
		if d284.Loc == scm.LocImm {
			d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d284.Imm.Int()))))}
		} else {
			r225 := ctx.AllocReg()
			ctx.EmitMovRegReg(r225, d284.Reg)
			ctx.EmitShlRegImm8(r225, 56)
			ctx.EmitShrRegImm8(r225, 56)
			d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
			ctx.BindReg(r225, &d285)
		}
		ctx.FreeDesc(&d284)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d283)
		ctx.EnsureDesc(&d283)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d283)
		ctx.EnsureDesc(&d285)
		ctx.EnsureDescsTogether(&d283, &d285)
		if d283.Loc == scm.LocImm && d285.Loc == scm.LocImm {
			d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d283.Imm.Int() * d285.Imm.Int())}
		} else if d283.Loc == scm.LocImm {
			ctx.EnsureDesc(&d285)
			scratch := ctx.AllocRegExcept(d285.Reg)
			ctx.EmitMovRegReg(scratch, d285.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d283.Imm.Int())
			d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d287)
		} else if d285.Loc == scm.LocImm {
			ctx.EnsureDesc(&d283)
			scratch := ctx.AllocRegExcept(d283.Reg)
			ctx.EmitMovRegReg(scratch, d283.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d285.Imm.Int())
			d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d287)
		} else {
			ctx.EnsureDesc(&d283)
			ctx.SyncDesc(&d285)
			r226 := ctx.AllocRegExcept(d283.Reg, d285.Reg)
			ctx.EmitMovRegReg(r226, d283.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r226, &d285)
			d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
			ctx.BindReg(r226, &d287)
		}
		if d287.Loc == scm.LocReg && d283.Loc == scm.LocReg && d287.Reg == d283.Reg {
			ctx.TransferReg(d283.Reg)
			d283.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d287)
		if d287.Loc == scm.LocImm {
			d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d287.Imm.Int() / 64)}
		} else {
			r227 := ctx.AllocRegExcept(d287.Reg)
			ctx.EmitMovRegReg(r227, d287.Reg)
			ctx.EmitShrRegImm8(r227, 6)
			d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
			ctx.BindReg(r227, &d288)
		}
		if d288.Loc == scm.LocReg && d287.Loc == scm.LocReg && d288.Reg == d287.Reg {
			ctx.TransferReg(d287.Reg)
			d287.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d287)
		if d287.Loc == scm.LocImm {
			d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d287.Imm.Int() % 64)}
		} else {
			r228 := ctx.AllocRegExcept(d287.Reg)
			ctx.EmitMovRegReg(r228, d287.Reg)
			ctx.EmitAndRegImm32(r228, 63)
			d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
			ctx.BindReg(r228, &d289)
		}
		if d289.Loc == scm.LocReg && d287.Loc == scm.LocReg && d289.Reg == d287.Reg {
			ctx.TransferReg(d287.Reg)
			d287.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d287)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d290 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r229 := ctx.AllocReg()
			r230 := ctx.AllocRegExcept(r229)
			r231 := ctx.AllocRegExcept(r229, r230)
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
			ctx.EmitMovRegMem(r229, thisptr.Reg, off)
			ctx.EmitMovRegMem(r230, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r231, thisptr.Reg, off+16)
			d290 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r229, Reg2: r230, Reg3: r231}
			ctx.BindReg(r229, &d290)
			ctx.BindReg(r230, &d290)
			ctx.BindReg(r231, &d290)
			ctx.BindReg(r229, &d290)
			ctx.BindReg(r230, &d290)
			ctx.BindReg(r231, &d290)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d288)
		ctx.ReclaimUntrackedRegs()
		d291 = ctx.EmitLoadScalarSliceElement(&d290, &d288, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d291)
		ctx.EnsureDesc(&d289)
		ctx.EnsureDescsTogether(&d291, &d289)
		if d291.Loc == scm.LocImm && d289.Loc == scm.LocImm {
			d292 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d291.Imm.Int()) << uint64(d289.Imm.Int())))}
		} else if d289.Loc == scm.LocImm {
			r232 := ctx.AllocRegExcept(d291.Reg)
			ctx.EmitMovRegReg(r232, d291.Reg)
			ctx.EmitShlRegImm8(r232, uint8(d289.Imm.Int()))
			d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
			ctx.BindReg(r232, &d292)
		} else {
			shiftSrc := d291.Reg
			r233 := ctx.AllocRegExcept(d291.Reg, d289.Reg)
			ctx.EmitMovRegReg(r233, d291.Reg)
			shiftSrc = r233
			ctx.EmitShiftLeft(shiftSrc, d289.Reg, true)
			d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d292)
		}
		if d292.Loc == scm.LocReg && d291.Loc == scm.LocReg && d292.Reg == d291.Reg {
			ctx.TransferReg(d291.Reg)
			d291.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d291)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d288)
		ctx.EnsureDesc(&d288)
		if d288.Loc == scm.LocImm {
			d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d288.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d288.Reg)
			ctx.EmitMovRegReg(scratch, d288.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d293 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d293)
		}
		if d293.Loc == scm.LocReg && d288.Loc == scm.LocReg && d293.Reg == d288.Reg {
			ctx.TransferReg(d288.Reg)
			d288.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d288)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d293)
		ctx.ReclaimUntrackedRegs()
		d294 = ctx.EmitLoadScalarSliceElement(&d290, &d293, 8, scm.TagInt)
		ctx.FreeDesc(&d293)
		ctx.ReclaimUntrackedRegs()
		d295 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d289)
		ctx.EnsureDescsTogether(&d295, &d289)
		if d295.Loc == scm.LocImm && d289.Loc == scm.LocImm {
			d296 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d295.Imm.Int() - d289.Imm.Int())}
		} else if d289.Loc == scm.LocImm && d289.Imm.Int() == 0 {
			ctx.EnsureDesc(&d295)
			r234 := ctx.AllocRegExcept(d295.Reg)
			ctx.EmitMovRegReg(r234, d295.Reg)
			d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
			ctx.BindReg(r234, &d296)
		} else if d295.Loc == scm.LocImm {
			ctx.EnsureDesc(&d289)
			scratch := ctx.AllocRegExcept(d289.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d295.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d289)
			d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d296)
		} else if d289.Loc == scm.LocImm {
			ctx.EnsureDesc(&d295)
			scratch := ctx.AllocRegExcept(d295.Reg)
			ctx.EmitMovRegReg(scratch, d295.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d289.Imm.Int())
			d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d296)
		} else {
			ctx.EnsureDesc(&d295)
			ctx.SyncDesc(&d289)
			r235 := ctx.AllocRegExcept(d295.Reg, d289.Reg)
			ctx.EmitMovRegReg(r235, d295.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r235, &d289)
			d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
			ctx.BindReg(r235, &d296)
		}
		if d296.Loc == scm.LocReg && d295.Loc == scm.LocReg && d296.Reg == d295.Reg {
			ctx.TransferReg(d295.Reg)
			d295.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d289)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d294)
		ctx.EnsureDesc(&d296)
		ctx.EnsureDescsTogether(&d294, &d296)
		if d294.Loc == scm.LocImm && d296.Loc == scm.LocImm {
			d297 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d294.Imm.Int()) >> uint64(d296.Imm.Int())))}
		} else if d296.Loc == scm.LocImm {
			r236 := ctx.AllocRegExcept(d294.Reg)
			ctx.EmitMovRegReg(r236, d294.Reg)
			ctx.EmitShrRegImm8(r236, uint8(d296.Imm.Int()))
			d297 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
			ctx.BindReg(r236, &d297)
		} else {
			shiftSrc := d294.Reg
			r237 := ctx.AllocRegExcept(d294.Reg, d296.Reg)
			ctx.EmitMovRegReg(r237, d294.Reg)
			shiftSrc = r237
			ctx.EmitShiftRight(shiftSrc, d296.Reg, false)
			d297 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d297)
		}
		if d297.Loc == scm.LocReg && d294.Loc == scm.LocReg && d297.Reg == d294.Reg {
			ctx.TransferReg(d294.Reg)
			d294.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d294)
		ctx.FreeDesc(&d296)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d292)
		ctx.EnsureDesc(&d297)
		if d292.Loc == scm.LocImm && d297.Loc == scm.LocImm {
			d298 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d292.Imm.Int() | d297.Imm.Int())}
		} else if d292.Loc == scm.LocImm && d292.Imm.Int() == 0 {
			d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d297.Reg}
			ctx.BindReg(d297.Reg, &d298)
		} else if d297.Loc == scm.LocImm && d297.Imm.Int() == 0 {
			r238 := ctx.AllocRegExcept(d292.Reg)
			ctx.EmitMovRegReg(r238, d292.Reg)
			d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
			ctx.BindReg(r238, &d298)
		} else if d292.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d297.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d292.Imm.Int()))
			ctx.EmitOrInt64(scratch, d297.Reg)
			d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d298)
		} else if d297.Loc == scm.LocImm {
			r239 := ctx.AllocRegExcept(d292.Reg)
			ctx.EmitMovRegReg(r239, d292.Reg)
			if d297.Imm.Int() >= -2147483648 && d297.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r239, int32(d297.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d297.Imm.Int()))
				ctx.EmitOrInt64(r239, ctx.ScratchReg)
			}
			d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
			ctx.BindReg(r239, &d298)
		} else {
			r240 := ctx.AllocRegExcept(d292.Reg, d297.Reg)
			ctx.EmitMovRegReg(r240, d292.Reg)
			ctx.EmitOrInt64(r240, d297.Reg)
			d298 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
			ctx.BindReg(r240, &d298)
		}
		if d298.Loc == scm.LocReg && d292.Loc == scm.LocReg && d298.Reg == d292.Reg {
			ctx.TransferReg(d292.Reg)
			d292.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d292)
		ctx.FreeDesc(&d297)
		ctx.ReclaimUntrackedRegs()
		d299 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d285)
		ctx.EnsureDescsTogether(&d299, &d285)
		if d299.Loc == scm.LocImm && d285.Loc == scm.LocImm {
			d300 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d299.Imm.Int() - d285.Imm.Int())}
		} else if d285.Loc == scm.LocImm && d285.Imm.Int() == 0 {
			ctx.EnsureDesc(&d299)
			r241 := ctx.AllocRegExcept(d299.Reg)
			ctx.EmitMovRegReg(r241, d299.Reg)
			d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
			ctx.BindReg(r241, &d300)
		} else if d299.Loc == scm.LocImm {
			ctx.EnsureDesc(&d285)
			scratch := ctx.AllocRegExcept(d285.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d299.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d285)
			d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d300)
		} else if d285.Loc == scm.LocImm {
			ctx.EnsureDesc(&d299)
			scratch := ctx.AllocRegExcept(d299.Reg)
			ctx.EmitMovRegReg(scratch, d299.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d285.Imm.Int())
			d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d300)
		} else {
			ctx.EnsureDesc(&d299)
			ctx.SyncDesc(&d285)
			r242 := ctx.AllocRegExcept(d299.Reg, d285.Reg)
			ctx.EmitMovRegReg(r242, d299.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r242, &d285)
			d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
			ctx.BindReg(r242, &d300)
		}
		if d300.Loc == scm.LocReg && d299.Loc == scm.LocReg && d300.Reg == d299.Reg {
			ctx.TransferReg(d299.Reg)
			d299.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d285)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d298)
		ctx.EnsureDesc(&d300)
		ctx.EnsureDescsTogether(&d298, &d300)
		if d298.Loc == scm.LocImm && d300.Loc == scm.LocImm {
			d301 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d298.Imm.Int()) >> uint64(d300.Imm.Int())))}
		} else if d300.Loc == scm.LocImm {
			r243 := ctx.AllocRegExcept(d298.Reg)
			ctx.EmitMovRegReg(r243, d298.Reg)
			ctx.EmitShrRegImm8(r243, uint8(d300.Imm.Int()))
			d301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
			ctx.BindReg(r243, &d301)
		} else {
			shiftSrc := d298.Reg
			r244 := ctx.AllocRegExcept(d298.Reg, d300.Reg)
			ctx.EmitMovRegReg(r244, d298.Reg)
			shiftSrc = r244
			ctx.EmitShiftRight(shiftSrc, d300.Reg, false)
			d301 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d301)
		}
		if d301.Loc == scm.LocReg && d298.Loc == scm.LocReg && d301.Reg == d298.Reg {
			ctx.TransferReg(d298.Reg)
			d298.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d298)
		ctx.FreeDesc(&d300)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d301)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d301)
		ctx.EnsureDesc(&d301)
		if d301.Loc == scm.LocImm {
			d302 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d301.Imm.Int()))))}
		} else {
			r245 := ctx.AllocReg()
			ctx.EmitMovRegReg(r245, d301.Reg)
			d302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
			ctx.BindReg(r245, &d302)
		}
		ctx.FreeDesc(&d301)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d303 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 56)
			r246 := ctx.AllocReg()
			ctx.EmitMovRegMem(r246, thisptr.Reg, off)
			d303 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r246}
			ctx.BindReg(r246, &d303)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d302)
		ctx.EnsureDesc(&d303)
		ctx.EnsureDescsTogether(&d302, &d303)
		if d302.Loc == scm.LocImm && d303.Loc == scm.LocImm {
			d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d302.Imm.Int() + d303.Imm.Int())}
		} else if d303.Loc == scm.LocImm && d303.Imm.Int() == 0 {
			ctx.EnsureDesc(&d302)
			r247 := ctx.AllocRegExcept(d302.Reg)
			ctx.EmitMovRegReg(r247, d302.Reg)
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
			ctx.BindReg(r247, &d304)
		} else if d302.Loc == scm.LocImm && d302.Imm.Int() == 0 {
			ctx.EnsureDesc(&d303)
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d303.Reg}
			ctx.BindReg(d303.Reg, &d304)
		} else if d302.Loc == scm.LocImm {
			ctx.EnsureDesc(&d303)
			scratch := ctx.AllocRegExcept(d303.Reg)
			ctx.EmitMovRegReg(scratch, d303.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d302.Imm.Int())
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d304)
		} else if d303.Loc == scm.LocImm {
			ctx.EnsureDesc(&d302)
			scratch := ctx.AllocRegExcept(d302.Reg)
			ctx.EmitMovRegReg(scratch, d302.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d303.Imm.Int())
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d304)
		} else {
			ctx.EnsureDesc(&d302)
			ctx.SyncDesc(&d303)
			r248 := ctx.AllocRegExcept(d302.Reg, d303.Reg)
			ctx.EmitMovRegReg(r248, d302.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r248, &d303)
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
			ctx.BindReg(r248, &d304)
		}
		if d304.Loc == scm.LocReg && d302.Loc == scm.LocReg && d304.Reg == d302.Reg {
			ctx.TransferReg(d302.Reg)
			d302.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d302)
		ctx.FreeDesc(&d303)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d304)
		ctx.EnsureDesc(&d304)
		ctx.StabilizeDescForControlFlow(&d304)
		inlineHome306 := d304
		inlineHome306.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d278)
		if d278.Loc == scm.LocReg || d278.Loc == scm.LocFPReg {
			ctx.ProtectReg(d278.Reg)
		} else if d278.Loc == scm.LocRegPair {
			ctx.ProtectReg(d278.Reg)
			ctx.ProtectReg(d278.Reg2)
		}
		ctx.SyncDesc(&d304)
		if d304.Loc == scm.LocReg || d304.Loc == scm.LocFPReg {
			ctx.ProtectReg(d304.Reg)
		} else if d304.Loc == scm.LocRegPair {
			ctx.ProtectReg(d304.Reg)
			ctx.ProtectReg(d304.Reg2)
		}
		d307 = d278
		if d307.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d307)
		ctx.EmitStoreToStack(d307, int32(phiBase127)+int32(0))
		d308 = d304
		if d308.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d308)
		ctx.EmitStoreToStack(d308, int32(phiBase127)+int32(16))
		if d278.Loc == scm.LocReg || d278.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d278.Reg)
		} else if d278.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d278.Reg)
			ctx.UnprotectReg(d278.Reg2)
		}
		if d304.Loc == scm.LocReg || d304.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d304.Reg)
		} else if d304.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d304.Reg)
			ctx.UnprotectReg(d304.Reg2)
		}
		ctx.EmitJmp(lbl38)
		bbpos_8_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl45)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d309 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 88)
			r249 := ctx.AllocReg()
			ctx.EmitMovRegMem(r249, thisptr.Reg, off)
			d309 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
			ctx.BindReg(r249, &d309)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d309)
		ctx.EnsureDesc(&d309)
		if d309.Loc == scm.LocImm {
			d310 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d309.Imm.Int()))))}
		} else {
			r250 := ctx.AllocReg()
			ctx.EmitMovRegReg(r250, d309.Reg)
			ctx.EmitShlRegImm8(r250, 32)
			ctx.EmitShrRegImm8(r250, 32)
			d310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
			ctx.BindReg(r250, &d310)
		}
		ctx.FreeDesc(&d309)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d158)
		ctx.EnsureDesc(&d310)
		ctx.EnsureDescsTogether(&d158, &d310)
		if d158.Loc == scm.LocImm && d310.Loc == scm.LocImm {
			d311 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d158.Imm.Int()) == uint64(d310.Imm.Int()))}
		} else if d310.Loc == scm.LocImm {
			r251 := ctx.AllocRegExcept(d158.Reg)
			if d310.Imm.Int() >= -2147483648 && d310.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d158.Reg, int32(d310.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d310.Imm.Int()))
				ctx.EmitCmpInt64(d158.Reg, ctx.ScratchReg)
			}
			d311 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r251, Condition: scm.CondEqual}
			ctx.BindReg(r251, &d311)
		} else if d158.Loc == scm.LocImm {
			r252 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d158.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d310.Reg)
			d311 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r252, Condition: scm.CondEqual}
			ctx.BindReg(r252, &d311)
		} else {
			r253 := ctx.AllocRegExcept(d158.Reg)
			ctx.EmitCmpInt64(d158.Reg, d310.Reg)
			d311 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r253, Condition: scm.CondEqual}
			ctx.BindReg(r253, &d311)
		}
		ctx.FreeDesc(&d310)
		ctx.ReclaimUntrackedRegs()
		d312 = d311
		ctx.SyncDesc(&d312)
		if d312.Loc != scm.LocImm && d312.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl132 := ctx.ReserveLabel()
		lbl133 := ctx.ReserveLabel()
		if d312.Loc == scm.LocImm {
			if d312.Imm.Bool() {
				ctx.MarkLabel(lbl132)
				ctx.EmitJmp(lbl43)
			} else {
				ctx.MarkLabel(lbl133)
				ctx.EmitJmp(lbl44)
			}
		} else {
			ctx.EmitJump(d312.Condition, lbl132)
			ctx.EmitJmp(lbl133)
			ctx.FreeDesc(&d311)
			ctx.MarkLabel(lbl132)
			ctx.EmitJmp(lbl43)
			ctx.MarkLabel(lbl133)
			ctx.EmitJmp(lbl44)
		}
		bbpos_8_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl46)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d129)
		ctx.EnsureDescsTogether(&d128, &d129)
		if d128.Loc == scm.LocImm && d129.Loc == scm.LocImm {
			d313 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() + d129.Imm.Int())}
		} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
			ctx.EnsureDesc(&d128)
			r254 := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(r254, d128.Reg)
			d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
			ctx.BindReg(r254, &d313)
		} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
			ctx.EnsureDesc(&d129)
			d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
			ctx.BindReg(d129.Reg, &d313)
		} else if d128.Loc == scm.LocImm {
			ctx.EnsureDesc(&d129)
			scratch := ctx.AllocRegExcept(d129.Reg)
			ctx.EmitMovRegReg(scratch, d129.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d128.Imm.Int())
			d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d313)
		} else if d129.Loc == scm.LocImm {
			ctx.EnsureDesc(&d128)
			scratch := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(scratch, d128.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d129.Imm.Int())
			d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d313)
		} else {
			ctx.EnsureDesc(&d128)
			ctx.SyncDesc(&d129)
			r255 := ctx.AllocRegExcept(d128.Reg, d129.Reg)
			ctx.EmitMovRegReg(r255, d128.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r255, &d129)
			d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
			ctx.BindReg(r255, &d313)
		}
		if d313.Loc == scm.LocReg && d128.Loc == scm.LocReg && d313.Reg == d128.Reg {
			ctx.TransferReg(d128.Reg)
			d128.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d313)
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d313)
		ctx.EnsureDescsTogether(&d313, &d128)
		if d313.Loc == scm.LocImm && d128.Loc == scm.LocImm {
			d315 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d313.Imm.Int() - d128.Imm.Int())}
		} else {
			ctx.ProtectDescRegisters(&d313)
			ctx.ProtectDescRegisters(&d128)
			r256 := ctx.AllocReg()
			if d313.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r256, uint64(d313.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r256, d313.Reg)
			}
			if d128.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d128.Imm.Int()))
				ctx.EmitSubInt64(r256, ctx.ScratchReg)
			} else {
				ctx.EmitSubInt64(r256, d128.Reg)
			}
			ctx.UnprotectDescRegisters(&d128)
			ctx.UnprotectDescRegisters(&d313)
			d315 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
			ctx.BindReg(r256, &d315)
		}
		r257 := ctx.EmitSliceDataAfterLow(&d123, &d128, 1)
		d316 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
		ctx.BindReg(r257, &d316)
		ctx.BindReg(r257, &d316)
		var r258 scm.Reg
		var r259 scm.Reg
		ctx.SyncDesc(&d316)
		ctx.EnsureDesc(&d316)
		if d316.Loc == scm.LocImm {
			r258 = ctx.AllocReg()
			ctx.EmitMovRegImm64(r258, uint64(d316.Imm.Int()))
		} else {
			r258 = d316.Reg
		}
		ctx.ProtectReg(r258)
		ctx.SyncDesc(&d315)
		ctx.EnsureDesc(&d315)
		if d315.Loc == scm.LocImm {
			r259 = ctx.AllocReg()
			ctx.EmitMovRegImm64(r259, uint64(d315.Imm.Int()))
		} else {
			r259 = d315.Reg
		}
		ctx.ProtectReg(r259)
		ctx.UnprotectReg(r259)
		ctx.UnprotectReg(r258)
		d317 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r258, Reg2: r259}
		ctx.BindReg(r258, &d317)
		ctx.BindReg(r259, &d317)
		ctx.BindReg(r258, &d317)
		ctx.BindReg(r259, &d317)
		ctx.FreeDesc(&d313)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d317)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d317)
		d318 = ctx.EmitGoCallScalar(jitGoABI_abee595aa8f5322c, []scm.JITValueDesc{d317}, 2)
		ctx.EmitCopyScmerToDesc(&d133, &d318)
		ctx.EmitJmp(lbl35)
		bbpos_8_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl47)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		if d128.Loc == scm.LocImm {
			d319 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() & 1)}
		} else {
			r260 := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(r260, d128.Reg)
			ctx.EmitAndRegImm32(r260, int32(1))
			d319 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r260}
			ctx.BindReg(r260, &d319)
		}
		if d319.Loc == scm.LocReg && d128.Loc == scm.LocReg && d319.Reg == d128.Reg {
			ctx.TransferReg(d128.Reg)
			d128.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d319)
		ctx.EnsureDesc(&d319)
		if d319.Loc == scm.LocImm {
			d320 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint8(uint64(d319.Imm.Int()))))}
		} else {
			r261 := ctx.AllocReg()
			ctx.EmitMovRegReg(r261, d319.Reg)
			ctx.EmitShlRegImm8(r261, 56)
			ctx.EmitShrRegImm8(r261, 56)
			d320 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
			ctx.BindReg(r261, &d320)
		}
		ctx.StabilizeDescForControlFlow(&d320)
		inlineHome321 := d320
		inlineHome321.ID = 0
		ctx.FreeDesc(&d319)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		if d128.Loc == scm.LocImm {
			d322 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d128.Imm.Int()) >> 1))}
		} else {
			r262 := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(r262, d128.Reg)
			ctx.EmitShrRegImm8(r262, 1)
			d322 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
			ctx.BindReg(r262, &d322)
		}
		if d322.Loc == scm.LocReg && d128.Loc == scm.LocReg && d322.Reg == d128.Reg {
			ctx.TransferReg(d128.Reg)
			d128.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d322)
		inlineHome323 := d322
		inlineHome323.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d129)
		ctx.EnsureDesc(&d129)
		if d129.Loc == scm.LocImm {
			d324 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d129.Imm.Int()))))}
		} else {
			r263 := ctx.AllocReg()
			ctx.EmitMovRegReg(r263, d129.Reg)
			d324 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
			ctx.BindReg(r263, &d324)
		}
		ctx.StabilizeDescForControlFlow(&d324)
		inlineHome325 := d324
		inlineHome325.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d324)
		if d324.Loc == scm.LocImm {
			d326 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d324.Imm.Int() == 0)}
		} else {
			r264 := ctx.AllocRegExcept(d324.Reg)
			ctx.EmitCmpRegImm32(d324.Reg, 0)
			d326 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r264, Condition: scm.CondEqual}
			ctx.BindReg(r264, &d326)
		}
		ctx.ReclaimUntrackedRegs()
		d327 = d326
		ctx.SyncDesc(&d327)
		if d327.Loc != scm.LocImm && d327.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl134 := ctx.ReserveLabel()
		lbl135 := ctx.ReserveLabel()
		if d327.Loc == scm.LocImm {
			if d327.Imm.Bool() {
				ctx.MarkLabel(lbl134)
				ctx.EmitJmp(lbl64)
			} else {
				ctx.MarkLabel(lbl135)
				ctx.EmitJmp(lbl65)
			}
		} else {
			ctx.EmitJump(d327.Condition, lbl134)
			ctx.EmitJmp(lbl135)
			ctx.FreeDesc(&d326)
			ctx.MarkLabel(lbl134)
			ctx.EmitJmp(lbl64)
			ctx.MarkLabel(lbl135)
			ctx.EmitJmp(lbl65)
		}
		bbpos_8_29 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl65)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d125)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d322)
		ctx.EnsureDesc(&d322)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d322)
		ctx.EnsureDescsTogether(&d125, &d322)
		if d125.Loc == scm.LocImm && d322.Loc == scm.LocImm {
			d330 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d322.Imm.Int())}
		} else if d322.Loc == scm.LocImm && d322.Imm.Int() == 0 {
			ctx.EnsureDesc(&d125)
			r265 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r265, d125.Reg)
			d330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r265}
			ctx.BindReg(r265, &d330)
		} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
			ctx.EnsureDesc(&d322)
			d330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d322.Reg}
			ctx.BindReg(d322.Reg, &d330)
		} else if d125.Loc == scm.LocImm {
			ctx.EnsureDesc(&d322)
			scratch := ctx.AllocRegExcept(d322.Reg)
			ctx.EmitMovRegReg(scratch, d322.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d125.Imm.Int())
			d330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d330)
		} else if d322.Loc == scm.LocImm {
			ctx.EnsureDesc(&d125)
			scratch := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(scratch, d125.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d322.Imm.Int())
			d330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d330)
		} else {
			ctx.EnsureDesc(&d125)
			ctx.SyncDesc(&d322)
			r266 := ctx.AllocRegExcept(d125.Reg, d322.Reg)
			ctx.EmitMovRegReg(r266, d125.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r266, &d322)
			d330 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r266}
			ctx.BindReg(r266, &d330)
		}
		if d330.Loc == scm.LocReg && d125.Loc == scm.LocReg && d330.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d330)
		ctx.EnsureDesc(&d330)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d330)
		ctx.EnsureDesc(&d330)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d333 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r267 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r267, thisptr.Reg, off)
			d333 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r267}
			ctx.BindReg(r267, &d333)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d333)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d330)
		ctx.EnsureDesc(&d333)
		ctx.EnsureDesc(&d320)
		ctx.EnsureDesc(&d324)
		d334 = d330
		_ = d334
		d335 = d333
		_ = d335
		d336 = d320
		_ = d336
		d337 = d324
		_ = d337
		bbpos_14_0 := int32(-1)
		_ = bbpos_14_0
		lbl136 := ctx.ReserveLabel()
		_ = lbl136
		bbpos_14_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl136)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d335)
		ctx.EnsureDesc(&d335)
		if d335.Loc == scm.LocImm {
			d338 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d335.Imm.Int()))))}
		} else {
			r268 := ctx.AllocReg()
			ctx.EmitMovRegReg(r268, d335.Reg)
			ctx.EmitShlRegImm8(r268, 56)
			ctx.EmitShrRegImm8(r268, 56)
			d338 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r268}
			ctx.BindReg(r268, &d338)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d338)
		if d338.Loc == scm.LocImm {
			d339 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d338.Imm.Int()) << 43))}
		} else {
			r269 := ctx.AllocRegExcept(d338.Reg)
			ctx.EmitMovRegReg(r269, d338.Reg)
			ctx.EmitShlRegImm8(r269, 43)
			d339 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
			ctx.BindReg(r269, &d339)
		}
		if d339.Loc == scm.LocReg && d338.Loc == scm.LocReg && d339.Reg == d338.Reg {
			ctx.TransferReg(d338.Reg)
			d338.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d338)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d336)
		ctx.EnsureDesc(&d336)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d336)
		if d336.Loc == scm.LocImm {
			d341 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d336.Imm.Int()) << 42))}
		} else {
			r270 := ctx.AllocRegExcept(d336.Reg)
			ctx.EmitMovRegReg(r270, d336.Reg)
			ctx.EmitShlRegImm8(r270, 42)
			d341 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
			ctx.BindReg(r270, &d341)
		}
		if d341.Loc == scm.LocReg && d336.Loc == scm.LocReg && d341.Reg == d336.Reg {
			ctx.TransferReg(d336.Reg)
			d336.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d339)
		ctx.EnsureDesc(&d341)
		if d339.Loc == scm.LocImm && d341.Loc == scm.LocImm {
			d342 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d339.Imm.Int() | d341.Imm.Int())}
		} else if d339.Loc == scm.LocImm && d339.Imm.Int() == 0 {
			d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d341.Reg}
			ctx.BindReg(d341.Reg, &d342)
		} else if d341.Loc == scm.LocImm && d341.Imm.Int() == 0 {
			r271 := ctx.AllocRegExcept(d339.Reg)
			ctx.EmitMovRegReg(r271, d339.Reg)
			d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
			ctx.BindReg(r271, &d342)
		} else if d339.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d341.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d339.Imm.Int()))
			ctx.EmitOrInt64(scratch, d341.Reg)
			d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d342)
		} else if d341.Loc == scm.LocImm {
			r272 := ctx.AllocRegExcept(d339.Reg)
			ctx.EmitMovRegReg(r272, d339.Reg)
			if d341.Imm.Int() >= -2147483648 && d341.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r272, int32(d341.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d341.Imm.Int()))
				ctx.EmitOrInt64(r272, ctx.ScratchReg)
			}
			d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r272}
			ctx.BindReg(r272, &d342)
		} else {
			r273 := ctx.AllocRegExcept(d339.Reg, d341.Reg)
			ctx.EmitMovRegReg(r273, d339.Reg)
			ctx.EmitOrInt64(r273, d341.Reg)
			d342 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
			ctx.BindReg(r273, &d342)
		}
		if d342.Loc == scm.LocReg && d339.Loc == scm.LocReg && d342.Reg == d339.Reg {
			ctx.TransferReg(d339.Reg)
			d339.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d339)
		ctx.FreeDesc(&d341)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d337)
		ctx.EnsureDesc(&d337)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d342)
		ctx.EnsureDesc(&d337)
		if d342.Loc == scm.LocImm && d337.Loc == scm.LocImm {
			d344 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d342.Imm.Int() | d337.Imm.Int())}
		} else if d342.Loc == scm.LocImm && d342.Imm.Int() == 0 {
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d337.Reg}
			ctx.BindReg(d337.Reg, &d344)
		} else if d337.Loc == scm.LocImm && d337.Imm.Int() == 0 {
			r274 := ctx.AllocRegExcept(d342.Reg)
			ctx.EmitMovRegReg(r274, d342.Reg)
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r274}
			ctx.BindReg(r274, &d344)
		} else if d342.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d337.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d342.Imm.Int()))
			ctx.EmitOrInt64(scratch, d337.Reg)
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d344)
		} else if d337.Loc == scm.LocImm {
			r275 := ctx.AllocRegExcept(d342.Reg)
			ctx.EmitMovRegReg(r275, d342.Reg)
			if d337.Imm.Int() >= -2147483648 && d337.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r275, int32(d337.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d337.Imm.Int()))
				ctx.EmitOrInt64(r275, ctx.ScratchReg)
			}
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r275}
			ctx.BindReg(r275, &d344)
		} else {
			r276 := ctx.AllocRegExcept(d342.Reg, d337.Reg)
			ctx.EmitMovRegReg(r276, d342.Reg)
			ctx.EmitOrInt64(r276, d337.Reg)
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
			ctx.BindReg(r276, &d344)
		}
		if d344.Loc == scm.LocReg && d342.Loc == scm.LocReg && d344.Reg == d342.Reg {
			ctx.TransferReg(d342.Reg)
			d342.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d342)
		ctx.ReclaimUntrackedRegs()
		r277 := ctx.AllocReg()
		r278 := ctx.AllocRegExcept(r277)
		ctx.EmitMovRegImm64(r277, 0)
		ctx.EmitMovRegImm64(r278, 0)
		d345 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r277, Reg2: r278}
		ctx.BindReg(r277, &d345)
		ctx.BindReg(r278, &d345)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d346 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.EnsureDesc(&d344)
		d347 = d344
		_ = d347
		bbpos_15_0 := int32(-1)
		_ = bbpos_15_0
		lbl137 := ctx.ReserveLabel()
		_ = lbl137
		bbpos_15_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl137)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d347)
		if d347.Loc == scm.LocImm {
			d348 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d347.Imm.Int()) << 8))}
		} else {
			r279 := ctx.AllocRegExcept(d347.Reg)
			ctx.EmitMovRegReg(r279, d347.Reg)
			ctx.EmitShlRegImm8(r279, 8)
			d348 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
			ctx.BindReg(r279, &d348)
		}
		if d348.Loc == scm.LocReg && d347.Loc == scm.LocReg && d348.Reg == d347.Reg {
			ctx.TransferReg(d347.Reg)
			d347.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		d349 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.ReclaimUntrackedRegs()
		d350 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d348)
		d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		if d348.Loc == scm.LocImm && d352.Loc == scm.LocImm {
			d351 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d348.Imm.Int() | d352.Imm.Int())}
		} else if d348.Loc == scm.LocImm && d348.Imm.Int() == 0 {
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d352.Reg}
			ctx.BindReg(d352.Reg, &d351)
		} else if d352.Loc == scm.LocImm && d352.Imm.Int() == 0 {
			r280 := ctx.AllocRegExcept(d348.Reg)
			ctx.EmitMovRegReg(r280, d348.Reg)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r280}
			ctx.BindReg(r280, &d351)
		} else if d348.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d352.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d348.Imm.Int()))
			ctx.EmitOrInt64(scratch, d352.Reg)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d351)
		} else if d352.Loc == scm.LocImm {
			r281 := ctx.AllocRegExcept(d348.Reg)
			ctx.EmitMovRegReg(r281, d348.Reg)
			if d352.Imm.Int() >= -2147483648 && d352.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r281, int32(d352.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d352.Imm.Int()))
				ctx.EmitOrInt64(r281, ctx.ScratchReg)
			}
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
			ctx.BindReg(r281, &d351)
		} else {
			r282 := ctx.AllocRegExcept(d348.Reg, d352.Reg)
			ctx.EmitMovRegReg(r282, d348.Reg)
			ctx.EmitOrInt64(r282, d352.Reg)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
			ctx.BindReg(r282, &d351)
		}
		if d351.Loc == scm.LocReg && d348.Loc == scm.LocReg && d351.Reg == d348.Reg {
			ctx.TransferReg(d348.Reg)
			d348.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d348)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d351)
		ctx.FreeDesc(&d344)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d334)
		ctx.EnsureDescsTogether(&d345, &d334)
		ctx.EmitMovToReg(d345.Reg, d334)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d351)
		ctx.EnsureDescsTogether(&d345, &d351)
		ctx.EmitMovToReg(d345.Reg2, d351)
		ctx.FreeDesc(&d351)
		ctx.ReclaimUntrackedRegs()
		d353 = d345
		_ = d353
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d353)
		ctx.FreeDesc(&d330)
		ctx.FreeDesc(&d333)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d353)
		ctx.EmitCopyScmerToDesc(&d133, &d353)
		ctx.EmitJmp(lbl35)
		bbpos_8_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl49)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d125)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d128)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d128)
		ctx.EnsureDescsTogether(&d125, &d128)
		if d125.Loc == scm.LocImm && d128.Loc == scm.LocImm {
			d356 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d128.Imm.Int())}
		} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
			ctx.EnsureDesc(&d125)
			r283 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r283, d125.Reg)
			d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
			ctx.BindReg(r283, &d356)
		} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
			ctx.EnsureDesc(&d128)
			d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			ctx.BindReg(d128.Reg, &d356)
		} else if d125.Loc == scm.LocImm {
			ctx.EnsureDesc(&d128)
			scratch := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(scratch, d128.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d125.Imm.Int())
			d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d356)
		} else if d128.Loc == scm.LocImm {
			ctx.EnsureDesc(&d125)
			scratch := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(scratch, d125.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d128.Imm.Int())
			d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d356)
		} else {
			ctx.EnsureDesc(&d125)
			ctx.SyncDesc(&d128)
			r284 := ctx.AllocRegExcept(d125.Reg, d128.Reg)
			ctx.EmitMovRegReg(r284, d125.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r284, &d128)
			d356 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
			ctx.BindReg(r284, &d356)
		}
		if d356.Loc == scm.LocReg && d125.Loc == scm.LocReg && d356.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d356)
		ctx.EnsureDesc(&d356)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d356)
		ctx.EnsureDesc(&d356)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d359 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r285 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r285, thisptr.Reg, off)
			d359 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
			ctx.BindReg(r285, &d359)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d359)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d356)
		ctx.EnsureDesc(&d359)
		d360 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		d361 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(36)}
		d362 = d356
		_ = d362
		d363 = d359
		_ = d363
		bbpos_16_0 := int32(-1)
		_ = bbpos_16_0
		lbl138 := ctx.ReserveLabel()
		_ = lbl138
		bbpos_16_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl138)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d363)
		ctx.EnsureDesc(&d363)
		if d363.Loc == scm.LocImm {
			d364 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d363.Imm.Int()))))}
		} else {
			r286 := ctx.AllocReg()
			ctx.EmitMovRegReg(r286, d363.Reg)
			ctx.EmitShlRegImm8(r286, 56)
			ctx.EmitShrRegImm8(r286, 56)
			d364 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
			ctx.BindReg(r286, &d364)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d364)
		if d364.Loc == scm.LocImm {
			d365 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d364.Imm.Int()) << 43))}
		} else {
			r287 := ctx.AllocRegExcept(d364.Reg)
			ctx.EmitMovRegReg(r287, d364.Reg)
			ctx.EmitShlRegImm8(r287, 43)
			d365 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
			ctx.BindReg(r287, &d365)
		}
		if d365.Loc == scm.LocReg && d364.Loc == scm.LocReg && d365.Reg == d364.Reg {
			ctx.TransferReg(d364.Reg)
			d364.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d364)
		ctx.ReclaimUntrackedRegs()
		d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ctx.ReclaimUntrackedRegs()
		d367 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d365)
		d369 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		if d365.Loc == scm.LocImm && d369.Loc == scm.LocImm {
			d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d365.Imm.Int() | d369.Imm.Int())}
		} else if d365.Loc == scm.LocImm && d365.Imm.Int() == 0 {
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d369.Reg}
			ctx.BindReg(d369.Reg, &d368)
		} else if d369.Loc == scm.LocImm && d369.Imm.Int() == 0 {
			r288 := ctx.AllocRegExcept(d365.Reg)
			ctx.EmitMovRegReg(r288, d365.Reg)
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r288}
			ctx.BindReg(r288, &d368)
		} else if d365.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d369.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d365.Imm.Int()))
			ctx.EmitOrInt64(scratch, d369.Reg)
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d368)
		} else if d369.Loc == scm.LocImm {
			r289 := ctx.AllocRegExcept(d365.Reg)
			ctx.EmitMovRegReg(r289, d365.Reg)
			if d369.Imm.Int() >= -2147483648 && d369.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r289, int32(d369.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d369.Imm.Int()))
				ctx.EmitOrInt64(r289, ctx.ScratchReg)
			}
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r289}
			ctx.BindReg(r289, &d368)
		} else {
			r290 := ctx.AllocRegExcept(d365.Reg, d369.Reg)
			ctx.EmitMovRegReg(r290, d365.Reg)
			ctx.EmitOrInt64(r290, d369.Reg)
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r290}
			ctx.BindReg(r290, &d368)
		}
		if d368.Loc == scm.LocReg && d365.Loc == scm.LocReg && d368.Reg == d365.Reg {
			ctx.TransferReg(d365.Reg)
			d365.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d365)
		ctx.ReclaimUntrackedRegs()
		d370 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(36)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d368)
		d372 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(36)}
		if d368.Loc == scm.LocImm && d372.Loc == scm.LocImm {
			d371 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d368.Imm.Int() | d372.Imm.Int())}
		} else if d368.Loc == scm.LocImm && d368.Imm.Int() == 0 {
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d372.Reg}
			ctx.BindReg(d372.Reg, &d371)
		} else if d372.Loc == scm.LocImm && d372.Imm.Int() == 0 {
			r291 := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegReg(r291, d368.Reg)
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r291}
			ctx.BindReg(r291, &d371)
		} else if d368.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d372.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d368.Imm.Int()))
			ctx.EmitOrInt64(scratch, d372.Reg)
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d371)
		} else if d372.Loc == scm.LocImm {
			r292 := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegReg(r292, d368.Reg)
			if d372.Imm.Int() >= -2147483648 && d372.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r292, int32(d372.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d372.Imm.Int()))
				ctx.EmitOrInt64(r292, ctx.ScratchReg)
			}
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r292}
			ctx.BindReg(r292, &d371)
		} else {
			r293 := ctx.AllocRegExcept(d368.Reg, d372.Reg)
			ctx.EmitMovRegReg(r293, d368.Reg)
			ctx.EmitOrInt64(r293, d372.Reg)
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r293}
			ctx.BindReg(r293, &d371)
		}
		if d371.Loc == scm.LocReg && d368.Loc == scm.LocReg && d371.Reg == d368.Reg {
			ctx.TransferReg(d368.Reg)
			d368.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d368)
		ctx.ReclaimUntrackedRegs()
		r294 := ctx.AllocReg()
		r295 := ctx.AllocRegExcept(r294)
		ctx.EmitMovRegImm64(r294, 0)
		ctx.EmitMovRegImm64(r295, 0)
		d373 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r294, Reg2: r295}
		ctx.BindReg(r294, &d373)
		ctx.BindReg(r295, &d373)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d374 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.EnsureDesc(&d371)
		d375 = d371
		_ = d375
		bbpos_17_0 := int32(-1)
		_ = bbpos_17_0
		lbl139 := ctx.ReserveLabel()
		_ = lbl139
		bbpos_17_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl139)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d375)
		if d375.Loc == scm.LocImm {
			d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d375.Imm.Int()) << 8))}
		} else {
			r296 := ctx.AllocRegExcept(d375.Reg)
			ctx.EmitMovRegReg(r296, d375.Reg)
			ctx.EmitShlRegImm8(r296, 8)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r296}
			ctx.BindReg(r296, &d376)
		}
		if d376.Loc == scm.LocReg && d375.Loc == scm.LocReg && d376.Reg == d375.Reg {
			ctx.TransferReg(d375.Reg)
			d375.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		d377 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.ReclaimUntrackedRegs()
		d378 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d376)
		d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(19)}
		if d376.Loc == scm.LocImm && d380.Loc == scm.LocImm {
			d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d376.Imm.Int() | d380.Imm.Int())}
		} else if d376.Loc == scm.LocImm && d376.Imm.Int() == 0 {
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d380.Reg}
			ctx.BindReg(d380.Reg, &d379)
		} else if d380.Loc == scm.LocImm && d380.Imm.Int() == 0 {
			r297 := ctx.AllocRegExcept(d376.Reg)
			ctx.EmitMovRegReg(r297, d376.Reg)
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r297}
			ctx.BindReg(r297, &d379)
		} else if d376.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d380.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d376.Imm.Int()))
			ctx.EmitOrInt64(scratch, d380.Reg)
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d379)
		} else if d380.Loc == scm.LocImm {
			r298 := ctx.AllocRegExcept(d376.Reg)
			ctx.EmitMovRegReg(r298, d376.Reg)
			if d380.Imm.Int() >= -2147483648 && d380.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r298, int32(d380.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d380.Imm.Int()))
				ctx.EmitOrInt64(r298, ctx.ScratchReg)
			}
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r298}
			ctx.BindReg(r298, &d379)
		} else {
			r299 := ctx.AllocRegExcept(d376.Reg, d380.Reg)
			ctx.EmitMovRegReg(r299, d376.Reg)
			ctx.EmitOrInt64(r299, d380.Reg)
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r299}
			ctx.BindReg(r299, &d379)
		}
		if d379.Loc == scm.LocReg && d376.Loc == scm.LocReg && d379.Reg == d376.Reg {
			ctx.TransferReg(d376.Reg)
			d376.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d376)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d379)
		ctx.FreeDesc(&d371)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d362)
		ctx.EnsureDescsTogether(&d373, &d362)
		ctx.EmitMovToReg(d373.Reg, d362)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d379)
		ctx.EnsureDescsTogether(&d373, &d379)
		ctx.EmitMovToReg(d373.Reg2, d379)
		ctx.FreeDesc(&d379)
		ctx.ReclaimUntrackedRegs()
		d381 = d373
		_ = d381
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d381)
		ctx.FreeDesc(&d356)
		ctx.FreeDesc(&d359)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d381)
		ctx.EmitCopyScmerToDesc(&d133, &d381)
		ctx.EmitJmp(lbl35)
		bbpos_8_30 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl66)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d129)
		ctx.EnsureDesc(&d129)
		if d129.Loc == scm.LocImm {
			d382 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d129.Imm.Int()))))}
		} else {
			r300 := ctx.AllocReg()
			ctx.EmitMovRegReg(r300, d129.Reg)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r300}
			ctx.BindReg(r300, &d382)
		}
		ctx.StabilizeDescForControlFlow(&d382)
		inlineHome383 := d382
		inlineHome383.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d382)
		if d382.Loc == scm.LocImm {
			d384 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d382.Imm.Int() == 0)}
		} else {
			r301 := ctx.AllocRegExcept(d382.Reg)
			ctx.EmitCmpRegImm32(d382.Reg, 0)
			d384 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r301, Condition: scm.CondEqual}
			ctx.BindReg(r301, &d384)
		}
		ctx.ReclaimUntrackedRegs()
		d385 = d384
		ctx.SyncDesc(&d385)
		if d385.Loc != scm.LocImm && d385.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl140 := ctx.ReserveLabel()
		lbl141 := ctx.ReserveLabel()
		if d385.Loc == scm.LocImm {
			if d385.Imm.Bool() {
				ctx.MarkLabel(lbl140)
				ctx.EmitJmp(lbl73)
			} else {
				ctx.MarkLabel(lbl141)
				ctx.EmitJmp(lbl74)
			}
		} else {
			ctx.EmitJump(d385.Condition, lbl140)
			ctx.EmitJmp(lbl141)
			ctx.FreeDesc(&d384)
			ctx.MarkLabel(lbl140)
			ctx.EmitJmp(lbl73)
			ctx.MarkLabel(lbl141)
			ctx.EmitJmp(lbl74)
		}
		bbpos_8_38 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl74)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d125)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d128)
		ctx.EnsureDesc(&d128)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d128)
		ctx.EnsureDescsTogether(&d125, &d128)
		if d125.Loc == scm.LocImm && d128.Loc == scm.LocImm {
			d388 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d128.Imm.Int())}
		} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
			ctx.EnsureDesc(&d125)
			r302 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r302, d125.Reg)
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r302}
			ctx.BindReg(r302, &d388)
		} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
			ctx.EnsureDesc(&d128)
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			ctx.BindReg(d128.Reg, &d388)
		} else if d125.Loc == scm.LocImm {
			ctx.EnsureDesc(&d128)
			scratch := ctx.AllocRegExcept(d128.Reg)
			ctx.EmitMovRegReg(scratch, d128.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d125.Imm.Int())
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d388)
		} else if d128.Loc == scm.LocImm {
			ctx.EnsureDesc(&d125)
			scratch := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(scratch, d125.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d128.Imm.Int())
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d388)
		} else {
			ctx.EnsureDesc(&d125)
			ctx.SyncDesc(&d128)
			r303 := ctx.AllocRegExcept(d125.Reg, d128.Reg)
			ctx.EmitMovRegReg(r303, d125.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r303, &d128)
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r303}
			ctx.BindReg(r303, &d388)
		}
		if d388.Loc == scm.LocReg && d125.Loc == scm.LocReg && d388.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d388)
		ctx.EnsureDesc(&d388)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d388)
		ctx.EnsureDesc(&d388)
		ctx.StabilizeDescForControlFlow(&d388)
		inlineHome391 := d388
		inlineHome391.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d392 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r304 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r304, thisptr.Reg, off)
			d392 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r304}
			ctx.BindReg(r304, &d392)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d392)
		if d392.Loc == scm.LocImm {
			d393 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d392.Imm.Int()) == uint64(0x4))}
		} else {
			r305 := ctx.AllocRegExcept(d392.Reg)
			ctx.EmitCmpRegImm32(d392.Reg, 4)
			d393 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r305, Condition: scm.CondEqual}
			ctx.BindReg(r305, &d393)
		}
		ctx.FreeDesc(&d392)
		ctx.ReclaimUntrackedRegs()
		d394 = d393
		ctx.SyncDesc(&d394)
		if d394.Loc != scm.LocImm && d394.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl142 := ctx.ReserveLabel()
		lbl143 := ctx.ReserveLabel()
		if d394.Loc == scm.LocImm {
			if d394.Imm.Bool() {
				ctx.MarkLabel(lbl142)
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewInt(1)}, int32(phiBase127)+int32(32))
				ctx.EmitJmp(lbl76)
			} else {
				ctx.MarkLabel(lbl143)
				ctx.EmitJmp(lbl75)
			}
		} else {
			ctx.EmitJump(d394.Condition, lbl142)
			ctx.EmitJmp(lbl143)
			ctx.FreeDesc(&d393)
			ctx.MarkLabel(lbl142)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewInt(1)}, int32(phiBase127)+int32(32))
			ctx.EmitJmp(lbl76)
			ctx.MarkLabel(lbl143)
			ctx.EmitJmp(lbl75)
		}
		bbpos_8_40 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl76)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d130)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d395 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r306 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r306, thisptr.Reg, off)
			d395 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r306}
			ctx.BindReg(r306, &d395)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d395)
		if d395.Loc == scm.LocImm {
			d396 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d395.Imm.Int()) == uint64(0x11))}
		} else {
			r307 := ctx.AllocRegExcept(d395.Reg)
			ctx.EmitCmpRegImm32(d395.Reg, 17)
			d396 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r307, Condition: scm.CondEqual}
			ctx.BindReg(r307, &d396)
		}
		ctx.FreeDesc(&d395)
		ctx.ReclaimUntrackedRegs()
		d397 = d396
		ctx.SyncDesc(&d397)
		if d397.Loc != scm.LocImm && d397.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl144 := ctx.ReserveLabel()
		lbl145 := ctx.ReserveLabel()
		if d397.Loc == scm.LocImm {
			if d397.Imm.Bool() {
				ctx.MarkLabel(lbl144)
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewInt(1)}, int32(phiBase127)+int32(48))
				ctx.EmitJmp(lbl78)
			} else {
				ctx.MarkLabel(lbl145)
				ctx.EmitJmp(lbl77)
			}
		} else {
			ctx.EmitJump(d397.Condition, lbl144)
			ctx.EmitJmp(lbl145)
			ctx.FreeDesc(&d396)
			ctx.MarkLabel(lbl144)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewInt(1)}, int32(phiBase127)+int32(48))
			ctx.EmitJmp(lbl78)
			ctx.MarkLabel(lbl145)
			ctx.EmitJmp(lbl77)
		}
		bbpos_8_42 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl78)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d388)
		ctx.EnsureDesc(&d382)
		ctx.EnsureDesc(&d130)
		ctx.EnsureDesc(&d131)
		d398 = d388
		_ = d398
		ctx.StabilizeDescForControlFlow(&d398)
		inlineHome399 := d398
		inlineHome399.ID = 0
		d400 = d382
		_ = d400
		ctx.StabilizeDescForControlFlow(&d400)
		inlineHome401 := d400
		inlineHome401.ID = 0
		d402 = d130
		_ = d402
		ctx.StabilizeDescForControlFlow(&d402)
		inlineHome403 := d402
		inlineHome403.ID = 0
		d404 = d131
		_ = d404
		ctx.StabilizeDescForControlFlow(&d404)
		inlineHome405 := d404
		inlineHome405.ID = 0
		phiBase406 = ctx.AllocStack(int32(32))
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		_ = d407
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		_ = d408
		inlineResultOff409 = ctx.AllocStack(int32(16))
		d410 = scm.JITValueDesc{Loc: scm.LocStackPair, Type: scm.JITTypeUnknown, StackOff: inlineResultOff409}
		lbl146 := ctx.ReserveLabel()
		bbpos_18_0 := int32(-1)
		_ = bbpos_18_0
		lbl147 := ctx.ReserveLabel()
		_ = lbl147
		bbpos_18_1 := int32(-1)
		_ = bbpos_18_1
		lbl148 := ctx.ReserveLabel()
		_ = lbl148
		bbpos_18_2 := int32(-1)
		_ = bbpos_18_2
		lbl149 := ctx.ReserveLabel()
		_ = lbl149
		bbpos_18_3 := int32(-1)
		_ = bbpos_18_3
		lbl150 := ctx.ReserveLabel()
		_ = lbl150
		bbpos_18_4 := int32(-1)
		_ = bbpos_18_4
		lbl151 := ctx.ReserveLabel()
		_ = lbl151
		bbpos_18_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl147)
		ctx.ResolveFixups()
		d398 = inlineHome399
		d400 = inlineHome401
		d402 = inlineHome403
		d404 = inlineHome405
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d411 = d402
		ctx.EnsureDesc(&d411)
		if d411.Loc != scm.LocImm && d411.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl152 := ctx.ReserveLabel()
		lbl153 := ctx.ReserveLabel()
		if d411.Loc == scm.LocImm {
			if d411.Imm.Bool() {
				ctx.MarkLabel(lbl152)
				ctx.EmitJmp(lbl148)
			} else {
				ctx.MarkLabel(lbl153)
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(phiBase406)+int32(0))
				ctx.EmitJmp(lbl149)
			}
		} else {
			ctx.EmitCmpRegImm32(d411.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl152)
			ctx.EmitJmp(lbl153)
			ctx.MarkLabel(lbl152)
			ctx.EmitJmp(lbl148)
			ctx.MarkLabel(lbl153)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(phiBase406)+int32(0))
			ctx.EmitJmp(lbl149)
		}
		bbpos_18_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl149)
		ctx.ResolveFixups()
		d398 = inlineHome399
		d400 = inlineHome401
		d402 = inlineHome403
		d404 = inlineHome405
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d407)
		ctx.ReclaimUntrackedRegs()
		d412 = d404
		ctx.EnsureDesc(&d412)
		if d412.Loc != scm.LocImm && d412.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl154 := ctx.ReserveLabel()
		lbl155 := ctx.ReserveLabel()
		if d412.Loc == scm.LocImm {
			if d412.Imm.Bool() {
				ctx.MarkLabel(lbl154)
				ctx.EmitJmp(lbl150)
			} else {
				ctx.MarkLabel(lbl155)
				ctx.SyncDesc(&d407)
				if d407.Loc == scm.LocReg || d407.Loc == scm.LocFPReg {
					ctx.ProtectReg(d407.Reg)
				} else if d407.Loc == scm.LocRegPair {
					ctx.ProtectReg(d407.Reg)
					ctx.ProtectReg(d407.Reg2)
				}
				d413 = d407
				if d413.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d413)
				ctx.EmitStoreToStack(d413, int32(phiBase406)+int32(16))
				if d407.Loc == scm.LocReg || d407.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d407.Reg)
				} else if d407.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d407.Reg)
					ctx.UnprotectReg(d407.Reg2)
				}
				ctx.EmitJmp(lbl151)
			}
		} else {
			ctx.EmitCmpRegImm32(d412.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl154)
			ctx.EmitJmp(lbl155)
			ctx.MarkLabel(lbl154)
			ctx.EmitJmp(lbl150)
			ctx.MarkLabel(lbl155)
			ctx.SyncDesc(&d407)
			if d407.Loc == scm.LocReg || d407.Loc == scm.LocFPReg {
				ctx.ProtectReg(d407.Reg)
			} else if d407.Loc == scm.LocRegPair {
				ctx.ProtectReg(d407.Reg)
				ctx.ProtectReg(d407.Reg2)
			}
			d414 = d407
			if d414.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d414)
			ctx.EmitStoreToStack(d414, int32(phiBase406)+int32(16))
			if d407.Loc == scm.LocReg || d407.Loc == scm.LocFPReg {
				ctx.UnprotectReg(d407.Reg)
			} else if d407.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d407.Reg)
				ctx.UnprotectReg(d407.Reg2)
			}
			ctx.EmitJmp(lbl151)
		}
		bbpos_18_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl151)
		ctx.ResolveFixups()
		d398 = inlineHome399
		d400 = inlineHome401
		d402 = inlineHome403
		d404 = inlineHome405
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		r308 := ctx.AllocReg()
		r309 := ctx.AllocRegExcept(r308)
		ctx.EmitMovRegImm64(r308, 0)
		ctx.EmitMovRegImm64(r309, 0)
		d415 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r308, Reg2: r309}
		ctx.BindReg(r308, &d415)
		ctx.BindReg(r309, &d415)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d400)
		ctx.EnsureDesc(&d400)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d408)
		ctx.EnsureDesc(&d400)
		if d408.Loc == scm.LocImm && d400.Loc == scm.LocImm {
			d417 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d408.Imm.Int() | d400.Imm.Int())}
		} else if d408.Loc == scm.LocImm && d408.Imm.Int() == 0 {
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d400.Reg}
			ctx.BindReg(d400.Reg, &d417)
		} else if d400.Loc == scm.LocImm && d400.Imm.Int() == 0 {
			r310 := ctx.AllocRegExcept(d408.Reg)
			ctx.EmitMovRegReg(r310, d408.Reg)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r310}
			ctx.BindReg(r310, &d417)
		} else if d408.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d400.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d408.Imm.Int()))
			ctx.EmitOrInt64(scratch, d400.Reg)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d417)
		} else if d400.Loc == scm.LocImm {
			r311 := ctx.AllocRegExcept(d408.Reg)
			ctx.EmitMovRegReg(r311, d408.Reg)
			if d400.Imm.Int() >= -2147483648 && d400.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r311, int32(d400.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d400.Imm.Int()))
				ctx.EmitOrInt64(r311, ctx.ScratchReg)
			}
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r311}
			ctx.BindReg(r311, &d417)
		} else {
			r312 := ctx.AllocRegExcept(d408.Reg, d400.Reg)
			ctx.EmitMovRegReg(r312, d408.Reg)
			ctx.EmitOrInt64(r312, d400.Reg)
			d417 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r312}
			ctx.BindReg(r312, &d417)
		}
		if d417.Loc == scm.LocReg && d408.Loc == scm.LocReg && d417.Reg == d408.Reg {
			ctx.TransferReg(d408.Reg)
			d408.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d408)
		ctx.ReclaimUntrackedRegs()
		d418 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(20)}
		ctx.EnsureDesc(&d417)
		d419 = d417
		_ = d419
		bbpos_19_0 := int32(-1)
		_ = bbpos_19_0
		lbl156 := ctx.ReserveLabel()
		_ = lbl156
		bbpos_19_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl156)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d419)
		if d419.Loc == scm.LocImm {
			d420 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d419.Imm.Int()) << 8))}
		} else {
			r313 := ctx.AllocRegExcept(d419.Reg)
			ctx.EmitMovRegReg(r313, d419.Reg)
			ctx.EmitShlRegImm8(r313, 8)
			d420 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r313}
			ctx.BindReg(r313, &d420)
		}
		if d420.Loc == scm.LocReg && d419.Loc == scm.LocReg && d420.Reg == d419.Reg {
			ctx.TransferReg(d419.Reg)
			d419.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		d421 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(20)}
		ctx.ReclaimUntrackedRegs()
		d422 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(20)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d420)
		d424 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(20)}
		if d420.Loc == scm.LocImm && d424.Loc == scm.LocImm {
			d423 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d420.Imm.Int() | d424.Imm.Int())}
		} else if d420.Loc == scm.LocImm && d420.Imm.Int() == 0 {
			d423 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d424.Reg}
			ctx.BindReg(d424.Reg, &d423)
		} else if d424.Loc == scm.LocImm && d424.Imm.Int() == 0 {
			r314 := ctx.AllocRegExcept(d420.Reg)
			ctx.EmitMovRegReg(r314, d420.Reg)
			d423 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r314}
			ctx.BindReg(r314, &d423)
		} else if d420.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d424.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d420.Imm.Int()))
			ctx.EmitOrInt64(scratch, d424.Reg)
			d423 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d423)
		} else if d424.Loc == scm.LocImm {
			r315 := ctx.AllocRegExcept(d420.Reg)
			ctx.EmitMovRegReg(r315, d420.Reg)
			if d424.Imm.Int() >= -2147483648 && d424.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r315, int32(d424.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d424.Imm.Int()))
				ctx.EmitOrInt64(r315, ctx.ScratchReg)
			}
			d423 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r315}
			ctx.BindReg(r315, &d423)
		} else {
			r316 := ctx.AllocRegExcept(d420.Reg, d424.Reg)
			ctx.EmitMovRegReg(r316, d420.Reg)
			ctx.EmitOrInt64(r316, d424.Reg)
			d423 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r316}
			ctx.BindReg(r316, &d423)
		}
		if d423.Loc == scm.LocReg && d420.Loc == scm.LocReg && d423.Reg == d420.Reg {
			ctx.TransferReg(d420.Reg)
			d420.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d420)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d423)
		ctx.FreeDesc(&d417)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d398)
		ctx.EnsureDescsTogether(&d415, &d398)
		ctx.EmitMovToReg(d415.Reg, d398)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d423)
		ctx.EnsureDescsTogether(&d415, &d423)
		ctx.EmitMovToReg(d415.Reg2, d423)
		ctx.FreeDesc(&d423)
		ctx.ReclaimUntrackedRegs()
		d425 = d415
		_ = d425
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d425)
		ctx.EmitCopyScmerToDesc(&d410, &d425)
		ctx.EmitJmp(lbl146)
		bbpos_18_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl148)
		ctx.ResolveFixups()
		d398 = inlineHome399
		d400 = inlineHome401
		d402 = inlineHome403
		d404 = inlineHome405
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(140737488355328)}, int32(phiBase406)+int32(0))
		ctx.EmitJmp(lbl149)
		bbpos_18_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl150)
		ctx.ResolveFixups()
		d398 = inlineHome399
		d400 = inlineHome401
		d402 = inlineHome403
		d404 = inlineHome405
		d407 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(0)}
		d408 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase406) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d407)
		if d407.Loc == scm.LocImm {
			d426 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d407.Imm.Int() | 70368744177664)}
		} else if 70368744177664 == 0 {
			r317 := ctx.AllocRegExcept(d407.Reg)
			ctx.EmitMovRegReg(r317, d407.Reg)
			d426 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r317}
			ctx.BindReg(r317, &d426)
		} else {
			r318 := ctx.AllocRegExcept(d407.Reg)
			ctx.EmitMovRegReg(r318, d407.Reg)
			ctx.EmitMovRegImm64(ctx.ScratchReg, 0x400000000000)
			ctx.EmitOrInt64(r318, ctx.ScratchReg)
			d426 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r318}
			ctx.BindReg(r318, &d426)
		}
		if d426.Loc == scm.LocReg && d407.Loc == scm.LocReg && d426.Reg == d407.Reg {
			ctx.TransferReg(d407.Reg)
			d407.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d426)
		ctx.EmitStoreToStack(d426, int32(phiBase406)+int32(16))
		ctx.StabilizeDescForControlFlow(&d426)
		inlineHome427 := d426
		inlineHome427.ID = 0
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl151)
		ctx.MarkLabel(lbl146)
		ctx.FreeDesc(&d388)
		ctx.FreeDesc(&d131)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d410)
		ctx.EmitCopyScmerToDesc(&d133, &d410)
		ctx.EmitJmp(lbl35)
		bbpos_8_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl42)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d428 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 88)
			r319 := ctx.AllocReg()
			ctx.EmitMovRegMem(r319, thisptr.Reg, off)
			d428 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r319}
			ctx.BindReg(r319, &d428)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d278)
		ctx.EnsureDesc(&d428)
		ctx.EnsureDescsTogether(&d278, &d428)
		if d278.Loc == scm.LocImm && d428.Loc == scm.LocImm {
			d429 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d278.Imm.Int()) == uint64(d428.Imm.Int()))}
		} else if d428.Loc == scm.LocImm {
			r320 := ctx.AllocRegExcept(d278.Reg)
			if d428.Imm.Int() >= -2147483648 && d428.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d278.Reg, int32(d428.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d428.Imm.Int()))
				ctx.EmitCmpInt64(d278.Reg, ctx.ScratchReg)
			}
			d429 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r320, Condition: scm.CondEqual}
			ctx.BindReg(r320, &d429)
		} else if d278.Loc == scm.LocImm {
			r321 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d278.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d428.Reg)
			d429 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r321, Condition: scm.CondEqual}
			ctx.BindReg(r321, &d429)
		} else {
			r322 := ctx.AllocRegExcept(d278.Reg)
			ctx.EmitCmpInt64(d278.Reg, d428.Reg)
			d429 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r322, Condition: scm.CondEqual}
			ctx.BindReg(r322, &d429)
		}
		ctx.FreeDesc(&d428)
		ctx.ReclaimUntrackedRegs()
		d430 = d429
		ctx.SyncDesc(&d430)
		if d430.Loc != scm.LocImm && d430.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		lbl157 := ctx.ReserveLabel()
		lbl158 := ctx.ReserveLabel()
		if d430.Loc == scm.LocImm {
			if d430.Imm.Bool() {
				ctx.MarkLabel(lbl157)
				ctx.EmitJmp(lbl40)
			} else {
				ctx.MarkLabel(lbl158)
				ctx.EmitJmp(lbl41)
			}
		} else {
			ctx.EmitJump(d430.Condition, lbl157)
			ctx.EmitJmp(lbl158)
			ctx.FreeDesc(&d429)
			ctx.MarkLabel(lbl157)
			ctx.EmitJmp(lbl40)
			ctx.MarkLabel(lbl158)
			ctx.EmitJmp(lbl41)
		}
		bbpos_8_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl43)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d431 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d431)
		ctx.EmitCopyScmerToDesc(&d133, &d431)
		ctx.EmitJmp(lbl35)
		bbpos_8_28 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl64)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d432 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("")}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d432)
		ctx.EmitCopyScmerToDesc(&d133, &d432)
		ctx.EmitJmp(lbl35)
		bbpos_8_37 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl73)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d433 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("")}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d433)
		ctx.EmitCopyScmerToDesc(&d133, &d433)
		ctx.EmitJmp(lbl35)
		bbpos_8_39 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl75)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d434 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r323 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r323, thisptr.Reg, off)
			d434 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r323}
			ctx.BindReg(r323, &d434)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d434)
		if d434.Loc == scm.LocImm {
			d435 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d434.Imm.Int()) == uint64(0x12))}
		} else {
			ctx.EmitCmpRegImm32(d434.Reg, 18)
			r324 := ctx.AllocRegExcept(d434.Reg)
			ctx.EmitSetcc(r324, scm.CondEqual)
			d435 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r324}
			ctx.BindReg(r324, &d435)
		}
		ctx.EnsureDesc(&d435)
		ctx.EmitStoreToStack(d435, int32(phiBase127)+int32(32))
		ctx.StabilizeDescForControlFlow(&d435)
		inlineHome436 := d435
		inlineHome436.ID = 0
		ctx.FreeDesc(&d434)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl76)
		bbpos_8_41 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl77)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d435 = inlineHome436
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).format)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d437 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageString)(nil).format))
			r325 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r325, thisptr.Reg, off)
			d437 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r325}
			ctx.BindReg(r325, &d437)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d437)
		if d437.Loc == scm.LocImm {
			d438 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d437.Imm.Int()) == uint64(0x12))}
		} else {
			ctx.EmitCmpRegImm32(d437.Reg, 18)
			r326 := ctx.AllocRegExcept(d437.Reg)
			ctx.EmitSetcc(r326, scm.CondEqual)
			d438 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r326}
			ctx.BindReg(r326, &d438)
		}
		ctx.EnsureDesc(&d438)
		ctx.EmitStoreToStack(d438, int32(phiBase127)+int32(48))
		ctx.StabilizeDescForControlFlow(&d438)
		inlineHome439 := d438
		inlineHome439.ID = 0
		ctx.FreeDesc(&d437)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl78)
		bbpos_8_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl40)
		ctx.ResolveFixups()
		d121 = inlineHome122
		d123 = inlineHome124
		d125 = inlineHome126
		d158 = inlineHome159
		d183 = inlineHome185
		d207 = inlineHome209
		d212 = inlineHome213
		d278 = inlineHome280
		d304 = inlineHome306
		d320 = inlineHome321
		d322 = inlineHome323
		d324 = inlineHome325
		d382 = inlineHome383
		d388 = inlineHome391
		d435 = inlineHome436
		d438 = inlineHome439
		d128 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(0)}
		d129 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase127) + int32(16)}
		d130 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(32)}
		d131 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagBool, StackOff: int32(phiBase127) + int32(48)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		d440 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d440)
		ctx.EmitCopyScmerToDesc(&d133, &d440)
		ctx.EmitJmp(lbl35)
		ctx.MarkLabel(lbl35)
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d119)
		d441 = result
		ctx.EnsureDesc(&d133)
		if d133.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d133, &d441)
		} else {
			switch d133.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d441, d133)
			case scm.TagInt:
				ctx.EmitMakeInt(d441, d133)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d441, d133)
			case scm.TagNil:
				ctx.EmitMakeNil(d441)
			default:
				ctx.EmitMovPairToResult(&d133, &d441)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	_ = bbs[0].Render()
	ctx.MarkLabel(lbl0)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
	if storageInputsProtected {
		ctx.UnprotectDescRegisters(&storageIdxIntHome)
	}
	ctx.EndStandaloneFrame(standaloneFrame)
	return result
}

var base64Formats = [...]struct {
	format   StringFormat
	encoding *base64.Encoding
}{
	{FormatBase64Upper, base64.StdEncoding.Strict()},
	{FormatBase64Lower, base64.URLEncoding.Strict()},
	{FormatRawBase64Std, base64.RawStdEncoding.Strict()},
	{FormatRawBase64URL, base64.RawURLEncoding.Strict()},
}

func base64Encoding(format StringFormat) *base64.Encoding {
	switch format {
	case FormatBase64Upper:
		return base64.StdEncoding
	case FormatBase64Lower:
		return base64.URLEncoding
	case FormatRawBase64Std:
		return base64.RawStdEncoding
	case FormatRawBase64URL:
		return base64.RawURLEncoding
	}
	panic("not a Base64 format")
}

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_abee595aa8f5322c uint64

func init() {
	jitGoABI_abee595aa8f5322c = scm.GoFuncAddr(scm.NewString)
}
