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
import "sort"
import "sync"
import "time"
import "unsafe"
import "strings"
import "encoding/hex"
import "encoding/base64"
import "encoding/binary"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/pierrec/lz4/v4"

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
		// '=' is only valid at the last two positions in a base64 string
		if c == '=' && i < len(s)-2 {
			valid &^= (1 << FormatBase64Upper) | (1 << FormatBase64Lower)
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

// appendNibbles encodes s into 4-bit nibbles using cs and appends to dst.
// compNibble is the absolute nibble position in the buffer (= byte_offset*2 + nibble_within_byte).
// Returns (new dst, new absolute nibble position = compNibble + len(s)).
func appendNibbles(dst []byte, s string, cs *nibbleCharset, compNibble int) ([]byte, int) {
	for i := 0; i < len(s); i++ {
		nib := byte(cs.dec[s[i]])
		if (compNibble+i)%2 == 0 {
			dst = append(dst, nib) // low nibble of a new byte
		} else {
			dst[len(dst)-1] |= nib << 4 // high nibble of last byte
		}
	}
	return dst, compNibble + len(s)
}

// isNibbleFormat reports whether format uses 4-bit nibble packing.
func isNibbleFormat(f StringFormat) bool {
	switch f {
	case FormatHexLower, FormatHexUpper, FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
		return true
	}
	return false
}

// nibbleCharsetFor returns the nibble charset for a nibble format, or nil.
func nibbleCharsetFor(f StringFormat) *nibbleCharset {
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
	case FormatBase64Upper:
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			panic(fmt.Sprintf("compressNonNibble: invalid standard base64 %q: %v", s, err))
		}
		return append(dst, b...)
	case FormatBase64Lower:
		b, err := base64.URLEncoding.DecodeString(s)
		if err != nil {
			panic(fmt.Sprintf("compressNonNibble: invalid URL-safe base64 %q: %v", s, err))
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

// adjustStartsForFormat resets the starts StorageInt offset to 0 for nodict mode.
// During scan, starts accumulates END positions (allsize), but build() writes
// START positions beginning from 0.  Setting offset=0 ensures all build values fit.
// Must be called BEFORE starts.init() in the nodict path.
func adjustStartsForFormat(starts *StorageInt) {
	starts.offset = 0
}

// readNibbles decodes charLen nibble-encoded characters from ptr.
// nibbleOff (0 or 1) is the nibble index within *ptr where decoding starts.
func readNibbles(ptr *byte, nibbleOff int, charLen int, cs *nibbleCharset) string {
	if charLen == 0 {
		return ""
	}
	result := make([]byte, charLen)
	for i := 0; i < charLen; i++ {
		absNibble := nibbleOff + i
		b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(absNibble/2)))
		if absNibble%2 == 0 {
			result[i] = cs.enc[b&0x0F]
		} else {
			result[i] = cs.enc[(b>>4)&0x0F]
		}
	}
	return unsafe.String(&result[0], charLen)
}

// cstringDecompress materialises a tagCString Scmer into a plain Go string.
// ptr is a byte-aligned pointer into the StorageString dictionary.
// val encodes: bits 47-44 = format, bit 43 = nibbleOff, bits 42-0 = charLen.
func cstringDecompress(ptr *byte, val uint64) string {
	format := StringFormat(val >> 44)
	nibbleOff := int((val >> 43) & 1)
	charLen := int(val & ((1 << 43) - 1))
	switch format {
	case FormatHexLower:
		return readNibbles(ptr, nibbleOff, charLen, &hexLowerCharset)
	case FormatHexUpper:
		return readNibbles(ptr, nibbleOff, charLen, &hexUpperCharset)
	case FormatPhone:
		return readNibbles(ptr, nibbleOff, charLen, &phoneCharset)
	case FormatPhoneDTMF:
		return readNibbles(ptr, nibbleOff, charLen, &phoneDTMFCharset)
	case FormatDecimal:
		return readNibbles(ptr, nibbleOff, charLen, &decimalCharset)
	case FormatDateTime:
		return readNibbles(ptr, nibbleOff, charLen, &dateTimeCharset)
	case FormatUUIDLower:
		b := unsafe.Slice(ptr, 16)
		h := hex.EncodeToString(b)
		return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
	case FormatUUIDUpper:
		b := unsafe.Slice(ptr, 16)
		h := strings.ToUpper(hex.EncodeToString(b))
		return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
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
	// StorageInt for dictionary entries
	values StorageInt
	// the dictionary: compressed bytes for all values concatenated
	dictionary string
	starts     StorageInt // byte offsets into dictionary
	lens       StorageInt // compressed byte lengths
	nodict     bool       // disable values array
	format     StringFormat

	// LZ4 compression: when compressed==true, compressedDict holds the
	// lz4-compressed dictionary and dictionary is materialized on demand.
	compressedDict []byte // lz4-compressed dictionary bytes
	compressed     bool   // true → dictionary is stored lz4-compressed
	dictMu         sync.RWMutex
	readCount      uint64 // atomic; incremented on each GetValue for rebuild telemetry

	// helpers (scan/build phase only)
	sb           strings.Builder
	compBuf      []byte // accumulates compressed bytes during build (nodict) or init (dict)
	compNibble   int    // current nibble position in compBuf (nodict nibble build / dict nibble init)
	reverseMap   map[string][3]uint
	count        uint
	allsize      int
	validFormats uint16
	// prefix statistics
	prefixstat map[string]int
	laststr    string
}

func (s *StorageString) ComputeSize() uint {
	base := s.values.ComputeSize() + 8 + uint(len(s.dictionary)) + 24 + s.starts.ComputeSize() + s.lens.ComputeSize() + 8*8
	base += uint(len(s.compressedDict))
	return base
}

func (s *StorageString) String() string {
	lz4Tag := ""
	if s.compressed {
		lz4Tag = ", lz4"
	}
	if s.nodict {
		return fmt.Sprintf("string-buffer[%d bytes, format=%d%s]", len(s.dictionary)+len(s.compressedDict), s.format, lz4Tag)
	} else {
		return fmt.Sprintf("string-dict[%d entries; %d bytes, format=%d%s]", s.count, len(s.dictionary)+len(s.compressedDict), s.format, lz4Tag)
	}
}

// ReadCount returns the number of GetValue calls since the last rebuild.
func (s *StorageString) ReadCount() uint64 {
	return atomic.LoadUint64(&s.readCount)
}

// CompressDictionary lz4-compresses the dictionary and clears the
// materialized copy.  After this call, GetValue will lazily decompress
// on first read.  No-op if the dictionary is already compressed or empty.
func (s *StorageString) CompressDictionary() {
	s.dictMu.Lock()
	defer s.dictMu.Unlock()
	if s.compressed || len(s.dictionary) == 0 {
		return
	}
	src := unsafe.Slice(unsafe.StringData(s.dictionary), len(s.dictionary))
	bound := lz4.CompressBlockBound(len(src))
	dst := make([]byte, bound)
	var c lz4.Compressor
	n, err := c.CompressBlock(src, dst)
	if err != nil || n == 0 {
		// incompressible or error — keep uncompressed
		return
	}
	// store original length as 4-byte LE prefix so we know the decompressed size
	s.compressedDict = make([]byte, 4+n)
	binary.LittleEndian.PutUint32(s.compressedDict, uint32(len(s.dictionary)))
	copy(s.compressedDict[4:], dst[:n])
	s.dictionary = ""
	s.compressed = true
}

// EvictDictionary drops the materialized dictionary, keeping only the
// lz4-compressed form.  Returns the number of bytes freed.
// No-op if the column is not compressed or the dictionary is already evicted.
func (s *StorageString) EvictDictionary() int64 {
	s.dictMu.Lock()
	defer s.dictMu.Unlock()
	if !s.compressed || len(s.dictionary) == 0 {
		return 0
	}
	freed := int64(len(s.dictionary))
	s.dictionary = ""
	return freed
}

// stringDictCleanup is the CacheManager callback for evicting a materialized
// string dictionary.  The pointer is a *StorageString.
func stringDictCleanup(ptr any, freedByType *[numEvictableTypes]int64) bool {
	s := ptr.(*StorageString)
	freed := s.EvictDictionary()
	if freed > 0 && freedByType != nil {
		freedByType[TypeStringDict] += freed
	}
	return freed > 0
}

// stringDictLastUsed returns a zero time so that materialized dicts are
// among the first candidates for eviction (they are cheap to re-decompress).
func stringDictLastUsed(ptr any) time.Time {
	return time.Time{}
}

// ensureDict materializes the dictionary from lz4-compressed form if needed.
// Returns the dictionary string.  Safe for concurrent use.
func (s *StorageString) ensureDict() string {
	if !s.compressed {
		return s.dictionary
	}
	// fast path: already materialized
	s.dictMu.RLock()
	if d := s.dictionary; len(d) > 0 {
		s.dictMu.RUnlock()
		return d
	}
	s.dictMu.RUnlock()

	// slow path: decompress
	s.dictMu.Lock()
	defer s.dictMu.Unlock()
	if d := s.dictionary; len(d) > 0 {
		return d // another goroutine materialized while we waited
	}
	origLen := binary.LittleEndian.Uint32(s.compressedDict[:4])
	buf := make([]byte, origLen)
	n, err := lz4.UncompressBlock(s.compressedDict[4:], buf)
	if err != nil || n != int(origLen) {
		panic(fmt.Sprintf("StorageString: lz4 decompress failed: err=%v n=%d expected=%d", err, n, origLen))
	}
	s.dictionary = unsafe.String(&buf[0], int(origLen))
	// register materialized dictionary with CacheManager so it can be evicted
	GlobalCache.AddItem(s, int64(origLen), TypeStringDict, stringDictCleanup, stringDictLastUsed, nil)
	return s.dictionary
}

// storageStringVersion is the current binary format version for StorageString.
// Increment this constant and add a new deserializeStringV* helper whenever the
// layout after [nodict][format] changes.  Never delete old helpers.
//
// NOTE: The version byte occupies pad[0] (first of the 5 alignment bytes after
// format).  Old "smallerstrings" data had 0 there (pad was zero-filled), so it
// reads correctly as version 0.
//
// CAUTION: StringFormat values only go up to 10.  If a future StringFormat
// reaches 11 or higher, the legacy sentinel (>10 for old "123456" dummy) must
// be revisited.  New binary layout changes must use a version increment, NOT
// a new StringFormat value.
const storageStringVersion = 1

// StorageString binary layout (magic byte 20 consumed by shard loader):
//
//	[nodict uint8]         ← 0=dict mode, 1=buffer mode
//	[format uint8]         ← StringFormat (0..10); if >10: legacy pre-smallerstrings sentinel
//
//	Legacy (format byte > 10, i.e. '1'=49 from old ASCII dummy "123456"):
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
//	Version 1 (current):
//	  [version uint8]      ← 1
//	  [pad 4 bytes]        ← alignment padding
//	  [compressed uint8]   ← 0=uncompressed dict, 1=lz4-compressed dict
//	  [count uint64] [values StorageInt] [starts StorageInt] [lens StorageInt]
//	  if compressed==0: [dictlen uint64] [dict bytes]
//	  if compressed==1: [compressedLen uint64] [compressedDict bytes]
//
// Version history:
//
//	0: smallerstrings format; format byte 0..10; version in pad[0].
//	1 (current): adds lz4-compressed dictionary support.
func (s *StorageString) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(20)) // 20 = StorageString
	var nodict uint8 = 0
	if s.nodict {
		nodict = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(nodict))
	binary.Write(f, binary.LittleEndian, uint8(s.format))
	binary.Write(f, binary.LittleEndian, uint8(storageStringVersion)) // pad[0] repurposed as version byte
	var pad [4]byte
	f.Write(pad[:]) // remaining 4 alignment bytes

	// Version 1: compressed flag
	var compFlag uint8 = 0
	if s.compressed && len(s.compressedDict) > 0 {
		compFlag = 1
	}
	binary.Write(f, binary.LittleEndian, compFlag)

	if s.nodict {
		binary.Write(f, binary.LittleEndian, uint64(s.starts.count))
	} else {
		binary.Write(f, binary.LittleEndian, uint64(s.values.count))
	}
	s.values.Serialize(f)
	s.starts.Serialize(f)
	s.lens.Serialize(f)

	if compFlag == 1 {
		// write only the lz4-compressed dictionary
		binary.Write(f, binary.LittleEndian, uint64(len(s.compressedDict)))
		f.Write(s.compressedDict)
	} else {
		// write uncompressed dictionary (may need to materialize if compressed)
		dict := s.ensureDict()
		binary.Write(f, binary.LittleEndian, uint64(len(dict)))
		io.WriteString(f, dict)
	}
}

func (s *StorageString) Deserialize(f io.Reader) uint {
	var nodict uint8
	binary.Read(f, binary.LittleEndian, &nodict)
	if nodict == 1 {
		s.nodict = true
	}
	var formatByte uint8
	binary.Read(f, binary.LittleEndian, &formatByte)
	// Legacy compatibility: old format wrote the ASCII string "123456" as a
	// 6-byte dummy.  The first byte of that dummy is '1' (0x31 = 49).
	// No valid StringFormat constant uses a value > 10, so we detect legacy
	// by checking formatByte > 10: consume the remaining 5 legacy dummy bytes
	// and treat the column as FormatRaw (the only format the old code supported).
	if formatByte > 10 {
		var legacyPad [5]byte
		f.Read(legacyPad[:])
		s.format = FormatRaw
		return s.deserializeStringBody(f)
	}
	s.format = StringFormat(formatByte)
	var version uint8
	binary.Read(f, binary.LittleEndian, &version) // pad[0] repurposed as version byte
	var pad [4]byte
	f.Read(pad[:])
	switch version {
	case 0:
		return s.deserializeStringBody(f)
	case 1:
		return s.deserializeStringV1(f)
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
		f.Read(rawdata)
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
		f.Read(rawdata)
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

func (s *StorageString) GetCachedReader() ColumnReader { return s }

func (s *StorageString) GetValue(i uint32) scm.Scmer {
	atomic.AddUint64(&s.readCount, 1)

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
	dict := s.ensureDict()
	dictBase := unsafe.Pointer(unsafe.StringData(dict))
	switch s.format {
	case FormatRaw:
		byteStart := startVal
		return scm.NewString(dict[byteStart : byteStart+lensVal])
	case FormatHexLower, FormatHexUpper,
		FormatPhone, FormatPhoneDTMF, FormatDecimal, FormatDateTime:
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
	case FormatBase64Upper, FormatBase64Lower:
		byteOff := startVal
		decodedLen := int(lensVal)
		if decodedLen == 0 {
			return scm.NewString("")
		}
		ptr := (*byte)(unsafe.Pointer(uintptr(dictBase) + uintptr(byteOff)))
		return scm.NewBString(ptr, decodedLen, s.format == FormatBase64Lower)
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
		sort.Slice(entries, func(a, b int) bool { return entries[a].rawOff < entries[b].rawOff })

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
				case FormatBase64Upper, FormatBase64Lower:
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
			case FormatBase64Upper, FormatBase64Lower:
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
}

func (s *StorageString) proposeCompression(i uint32) ColumnStorage {
	// prefix-tree compression placeholder (see TODO below)
	/* TODO: reactivate as soon as StoragePrefix has a proper implementation for Serialize/Deserialize
	...
	*/
	return nil
}
