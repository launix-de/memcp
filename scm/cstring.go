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
package scm

import "unsafe"
import "strings"
import "unicode"
import "unicode/utf8"

// CString's aux layout is private, transient RAM state, never a disk layout.
const CStringFormatShift = 43
const CStringOffsetShift = 42
const CStringLengthMask = (1 << CStringOffsetShift) - 1

// cstringAlphabets is the shared immutable format registry for storage and
// execution. IDs 1..10 are permanent legacy assignments; 11..16 are sorted,
// high-nibble-first encodings, as are timestamp IDs 19/20. UUIDs have fixed
// separators, not a nibble table. IDs 17/18 belong to BString storage.
var cstringAlphabets = [...]string{
	1:  "0123456789 +-/()",
	2:  "0123456789abcdef",
	3:  "0123456789ABCDEF",
	8:  "0123456789+-.,eE",
	9:  "0123456789-:. T",
	10: "0123456789+-()#*",
	11: "0123456789abcdef",
	12: "0123456789ABCDEF",
	13: " ()+-/0123456789",
	14: "#()*+-0123456789",
	15: "+,-.0123456789Ee",
	16: " -.0123456789:T",
	19: " -.0123456789:TZ",
	20: "+-.0123456789:TZ",
}

// CStringAlphabet returns an immutable nibble alphabet, or empty for other formats.
func CStringAlphabet(format uint8) string {
	if int(format) < len(cstringAlphabets) {
		return cstringAlphabets[format]
	}
	return ""
}

// Two decoded ASCII characters per packed byte, in text order. The folded
// table keeps case conversion out of compressed/plain comparison loops.
var cstringPairs, cstringFoldedPairs = func() ([21][256]uint16, [21][256]uint16) {
	var plain, folded [21][256]uint16
	for f := 0; f < len(plain); f++ {
		alphabet := CStringAlphabet(uint8(f))
		if alphabet == "" {
			continue
		}
		for packed := 0; packed < 256; packed++ {
			x, y := packed&15, packed>>4
			if f >= 11 {
				x, y = y, x
			}
			if x >= len(alphabet) || y >= len(alphabet) {
				continue
			}
			a, b := alphabet[x], alphabet[y]
			plain[f][packed] = uint16(a) | uint16(b)<<8
			folded[f][packed] = uint16(asciiFoldByte(a)) | uint16(asciiFoldByte(b))<<8
		}
	}
	return plain, folded
}()

// stringView supports random access without materializing compressed text.
// Its source Scmer keeps dictionary memory alive throughout the operation.
type stringView struct {
	data      string
	alphabet  string
	n, offset int
	format    uint8
}

func makeStringView(s Scmer) (stringView, bool) {
	switch s.GetTag() {
	case tagString, tagSymbol:
		v := unsafe.String(s.ptr, int(auxVal(s.aux)))
		return stringView{data: v, n: len(v)}, true
	case tagCString:
		val := auxVal(s.aux)
		f := uint8(val >> CStringFormatShift)
		n := int(val & CStringLengthMask)
		off := int((val >> CStringOffsetShift) & 1)
		alphabet := CStringAlphabet(f)
		bytes := n
		if alphabet != "" {
			bytes = (n + off + 1) / 2
		} else if f == 6 || f == 7 {
			bytes = 16
		} else if f != 0 {
			return stringView{}, false
		}
		return stringView{data: unsafe.String(s.ptr, bytes), alphabet: alphabet, n: n, offset: off, format: f}, true
	}
	return stringView{}, false
}

func (s stringView) at(i int) byte {
	if s.alphabet != "" {
		pos := s.offset + i
		b := s.data[pos>>1]
		shift := uint((pos & 1) * 4)
		if s.format >= 11 {
			shift ^= 4
		}
		return s.alphabet[(b>>shift)&15]
	}
	if s.format == 6 || s.format == 7 {
		switch i {
		case 8, 13, 18, 23:
			return '-'
		}
		pos := i
		if i > 23 {
			pos -= 4
		} else if i > 18 {
			pos -= 3
		} else if i > 13 {
			pos -= 2
		} else if i > 8 {
			pos--
		}
		nib := (s.data[pos>>1] >> uint(4-(pos&1)*4)) & 15
		if s.format == 7 {
			return "0123456789ABCDEF"[nib]
		}
		return "0123456789abcdef"[nib]
	}
	return s.data[i]
}

func (v stringView) decodeInto(dst []byte, start int, fold bool) {
	if v.format == 0 {
		copy(dst, v.data[start:start+len(dst)])
		if fold {
			for i, c := range dst {
				dst[i] = asciiFoldByte(c)
			}
		}
		return
	}
	if v.alphabet != "" {
		pairs := &cstringPairs[v.format]
		if fold {
			pairs = &cstringFoldedPairs[v.format]
		}
		i, pos := 0, v.offset+start
		if pos&1 != 0 && len(dst) > 0 {
			c := v.at(start)
			if fold {
				c = asciiFoldByte(c)
			}
			dst[0] = c
			i++
			pos++
		}
		for ; i+1 < len(dst); i, pos = i+2, pos+2 {
			pair := pairs[v.data[pos>>1]]
			dst[i] = byte(pair)
			dst[i+1] = byte(pair >> 8)
		}
		if i < len(dst) {
			c := v.at(start + i)
			if fold {
				c = asciiFoldByte(c)
			}
			dst[i] = c
		}
		return
	}
	for i := range dst {
		c := v.at(start + i)
		if fold {
			c = asciiFoldByte(c)
		}
		dst[i] = c
	}
}

func (v stringView) matchRange(start int, pattern string, fold bool) (bool, bool) {
	if start < 0 || len(pattern) > v.n-start {
		return false, true
	}
	if v.alphabet != "" {
		pos := v.offset + start
		v.data = v.data[pos>>1:]
		v.offset = pos & 1
		v.n = len(pattern)
		c, ok := compareNibblePlain(v, pattern, fold)
		return c == 0, ok
	}
	for i := range pattern {
		c, p := v.at(start+i), pattern[i]
		if fold {
			if p >= utf8.RuneSelf {
				return false, false
			}
			c, p = asciiFoldByte(c), asciiFoldByte(p)
		}
		if c != p {
			return false, true
		}
	}
	return true, true
}

func cstringSubstring(s Scmer, start, end int) Scmer {
	v, ok := makeOperatorView(s)
	if !ok {
		return NewString(s.String()[start:end])
	}
	if start < 0 || end < start || end > v.n {
		panic("substr: slice bounds out of range")
	}
	if start == end {
		return NewString("")
	}
	if start == 0 && end == v.n {
		return s
	}
	if v.alphabet != "" {
		pos := v.offset + start
		return NewCString((*byte)(unsafe.Add(unsafe.Pointer(s.ptr), pos/2)), v.format, uint8(pos&1), end-start)
	}
	result := make([]byte, end-start)
	v.decodeInto(result, start, false)
	return NewString(unsafe.String(&result[0], len(result)))
}

func compareLength(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Resolve format dispatch once, outside the comparison loop. In particular,
// mixed query comparisons must not call the general UUID/raw accessor per byte.
func compareNibblePlain(a stringView, b string, fold bool) (int, bool) {
	n := min(a.n, len(b))
	pairs := &cstringPairs[a.format]
	if fold {
		pairs = &cstringFoldedPairs[a.format]
	}
	i, start := 0, 0
	if a.offset == 1 && n > 0 {
		x, y := a.at(0), b[0]
		if fold {
			if y >= utf8.RuneSelf {
				return 0, false
			}
			x, y = asciiFoldByte(x), asciiFoldByte(y)
		}
		if x != y {
			return int(x) - int(y), true
		}
		i, start = 1, 1
	}
	for ; i+1 < n; i, start = i+2, start+1 {
		pair := pairs[a.data[start]]
		x, y := b[i], b[i+1]
		if fold {
			if x >= utf8.RuneSelf || y >= utf8.RuneSelf {
				return 0, false
			}
			x, y = asciiFoldByte(x), asciiFoldByte(y)
		}
		if pair == uint16(x)|uint16(y)<<8 {
			continue
		}
		if byte(pair) != x {
			return int(byte(pair)) - int(x), true
		}
		return int(byte(pair>>8)) - int(y), true
	}
	if i < n {
		// The unused neighbouring nibble must never participate in the result.
		x, y := a.at(i), b[i]
		if fold {
			if y >= utf8.RuneSelf {
				return 0, false
			}
			x, y = asciiFoldByte(x), asciiFoldByte(y)
		}
		if x != y {
			return int(x) - int(y), true
		}
	}
	return compareLength(a.n, len(b)), true
}

func compareNibbles(a, b stringView, fold bool) (int, bool) {
	ax, bx := 0, 0
	if a.format >= 11 {
		ax = 4
	}
	if b.format >= 11 {
		bx = 4
	}
	for i := 0; i < min(a.n, b.n); i++ {
		ap, bp := a.offset+i, b.offset+i
		x := a.alphabet[(a.data[ap>>1]>>uint(((ap&1)*4)^ax))&15]
		y := b.alphabet[(b.data[bp>>1]>>uint(((bp&1)*4)^bx))&15]
		if fold {
			x, y = asciiFoldByte(x), asciiFoldByte(y)
		}
		if x != y {
			return int(x) - int(y), true
		}
	}
	return compareLength(a.n, b.n), true
}

// Legacy low-nibble-first bytes are not ordered, but equal machine-sized
// prefixes can still be skipped before decoding the first differing byte.
func compareLegacyNibbles(a, b stringView) int {
	n := min(a.n, b.n)
	pos := 0
	if a.offset == 1 && n > 0 {
		x, y := a.alphabet[a.data[0]>>4], b.alphabet[b.data[0]>>4]
		if x != y {
			return int(x) - int(y)
		}
		pos++
		n--
	}
	end := pos + n/2
	for pos+8 <= end && a.data[pos:pos+8] == b.data[pos:pos+8] {
		pos += 8
	}
	for ; pos < end; pos++ {
		x, y := a.data[pos], b.data[pos]
		if x == y {
			continue
		}
		if x&15 != y&15 {
			return int(a.alphabet[x&15]) - int(b.alphabet[y&15])
		}
		return int(a.alphabet[x>>4]) - int(b.alphabet[y>>4])
	}
	if n&1 != 0 {
		x, y := a.alphabet[a.data[pos]&15], b.alphabet[b.data[pos]&15]
		if x != y {
			return int(x) - int(y)
		}
	}
	return compareLength(a.n, b.n)
}

func compareStringViews(a, b stringView, fold bool) (int, bool) {
	// Every alphabet except Decimal has at most one spelling of each folded
	// character and preserves order under folding. Raw text needs Unicode rules.
	if fold && a.format == b.format && a.format != 0 && a.format != 8 && a.format != 15 {
		fold = false
	}
	if fold && (a.format == 6 || a.format == 7) && (b.format == 6 || b.format == 7) {
		return strings.Compare(a.data, b.data), true
	}
	if !fold && a.format == b.format {
		if a.format == 0 {
			return strings.Compare(a.data, b.data), true
		}
		if a.format == 6 || a.format == 7 {
			return strings.Compare(a.data, b.data), true
		}
		if a.alphabet != "" && a.format < 11 && a.offset == b.offset {
			return compareLegacyNibbles(a, b), true
		}
		// In ordered encodings, aligned full bytes have exactly text ordering.
		// Exclude neighbour nibbles at both edges of packed dictionary entries.
		if a.format >= 11 && a.offset == b.offset {
			n := min(a.n, b.n)
			start := 0
			if a.offset == 1 && n > 0 {
				x, y := a.data[0]&15, b.data[0]&15
				if x != y {
					return int(x) - int(y), true
				}
				start = 1
				n--
			}
			full := n / 2
			if c := strings.Compare(a.data[start:start+full], b.data[start:start+full]); c != 0 {
				return c, true
			}
			if n&1 != 0 {
				x, y := a.data[start+full]>>4, b.data[start+full]>>4
				if x != y {
					return int(x) - int(y), true
				}
			}
			return compareLength(a.n, b.n), true
		}
	}
	if a.alphabet != "" {
		if b.format == 0 {
			return compareNibblePlain(a, b.data, fold)
		}
		if b.alphabet != "" {
			return compareNibbles(a, b, fold)
		}
	} else if a.format == 0 && b.alphabet != "" {
		c, ok := compareNibblePlain(b, a.data, fold)
		return -c, ok
	}

	n := min(a.n, b.n)
	for i := 0; i < n; i++ {
		x, y := a.at(i), b.at(i)
		if fold {
			// Unicode case folding may change byte width or equate ASCII with a
			// non-ASCII rune. The caller retains its canonical Unicode operation.
			if x >= utf8.RuneSelf || y >= utf8.RuneSelf {
				return 0, false
			}
			x, y = asciiFoldByte(x), asciiFoldByte(y)
		}
		if x != y {
			return int(x) - int(y), true
		}
	}
	// An unmatched non-ASCII suffix cannot equal an empty ASCII suffix, and
	// both ToLower and EqualFold preserve non-emptiness.
	return compareLength(a.n, b.n), true
}

func compareCString(a, b Scmer, fold bool) (int, bool) {
	av, ok := makeStringView(a)
	if !ok {
		return 0, false
	}
	bv, ok := makeStringView(b)
	if !ok {
		return 0, false
	}
	return compareStringViews(av, bv, fold)
}

func equalStringValues(a, b Scmer, fold bool) bool {
	if a.GetTag() == tagCString || b.GetTag() == tagCString {
		av, aok := makeStringView(a)
		bv, bok := makeStringView(b)
		if aok && bok {
			if !fold && av.n != bv.n {
				return false
			}
			if c, ok := compareStringViews(av, bv, fold); ok {
				return c == 0
			}
		}
	}
	if a.IsBString() || b.IsBString() {
		if !fold && a.IsBString() && b.IsBString() && auxVal(a.aux)>>46 == auxVal(b.aux)>>46 {
			an, bn := int(auxVal(a.aux)&bstringLengthMask), int(auxVal(b.aux)&bstringLengthMask)
			return unsafe.String(a.ptr, an) == unsafe.String(b.ptr, bn)
		}
		if !fold {
			if equal, ok := equalBase64Bytes(a, b); ok {
				return equal
			}
		}
		if c, ok := compareBase64Text(a, b, fold); ok {
			return c == 0
		}
	}
	if fold {
		return strings.EqualFold(a.String(), b.String())
	}
	return a.String() == b.String()
}

// General collation deliberately keeps its historical leading-aa class and
// binary ordering inside the non-letter class, including descending order.
func generalCStringLess(a, b Scmer, reverse bool) (bool, bool) {
	av, ok := makeStringView(a)
	if !ok {
		return false, false
	}
	bv, ok := makeStringView(b)
	if !ok {
		return false, false
	}
	classify := func(v stringView) (bool, byte) {
		if v.n == 0 {
			return true, 0
		}
		c := asciiFoldByte(v.at(0))
		if v.n > 1 && c == 'a' && asciiFoldByte(v.at(1)) == 'a' {
			return false, 0
		}
		return c >= 'a' && c <= 'z', c
	}
	ac, ak := classify(av)
	bc, bk := classify(bv)
	if ac != bc {
		return ac, true
	}
	if ac && ak != bk {
		if reverse {
			return ak > bk, true
		}
		return ak < bk, true
	}
	c, ok := compareStringViews(av, bv, ac)
	if !ok {
		return false, false
	}
	if reverse {
		return c > 0, true
	}
	return c < 0, true
}

// strLikeCString consumes ASCII compressed text directly. Random access makes
// suffixes and wildcard backtracking possible without a decoded copy. For
// literal contains, decode overlapping stack chunks to use strings.Contains.
func strLikeCString(value Scmer, pattern, collation string) (bool, bool) {
	v, ok := makeOperatorView(value)
	if !ok || (v.format == 0 && v.base64 == "") {
		return false, false
	}
	fold := strings.Contains(strings.ToLower(collation), "_ci")
	if fold {
		// ASCII search constants need no allocated lowercase copy. Unicode
		// lowercasing retains the canonical LIKE behavior (including width).
		for i := range pattern {
			if pattern[i] >= utf8.RuneSelf {
				pattern = strings.ToLower(pattern)
				break
			}
		}
	}
	if !likeAlphabetPossible(v, pattern, fold) {
		return false, true
	}
	if !strings.ContainsAny(pattern, "_\\") {
		count := strings.Count(pattern, "%")
		switch {
		case count == 0:
			if len(pattern) != v.n {
				return false, true
			}
			return v.matchRange(0, pattern, fold)
		case count == 1 && strings.HasPrefix(pattern, "%"):
			return v.matchRange(v.n-len(pattern)+1, pattern[1:], fold)
		case count == 1 && strings.HasSuffix(pattern, "%"):
			return v.matchRange(0, pattern[:len(pattern)-1], fold)
		}
	}
	if !strings.ContainsAny(pattern, "_\\") && strings.Count(pattern, "%") == 2 && strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		needle := pattern[1 : len(pattern)-1]
		if len(needle) == 0 {
			return true, true
		}
		if len(needle) > v.n {
			return false, true
		}
		if len(needle) <= 128 {
			var needleBuffer [128]byte
			if fold {
				for i := range needle {
					needleBuffer[i] = asciiFoldByte(needle[i])
				}
				needle = unsafe.String(&needleBuffer[0], len(needle))
			}
			var buf [256]byte
			kept := 0
			for pos := 0; pos < v.n; {
				count := min(len(buf)-kept, v.n-pos)
				v.decodeInto(buf[kept:kept+count], pos, fold)
				size := kept + count
				if strings.Contains(unsafe.String(&buf[0], size), needle) {
					return true, true
				}
				pos += count
				kept = len(needle) - 1
				copy(buf[:kept], buf[size-kept:size])
			}
			return false, true
		}
	}
	// Greedy SQL wildcard matcher. A remembered '%' spans arbitrary lengths;
	// carrying that position is essential for patterns such as a%b%c.
	i, p, star, retry := 0, 0, -1, 0
	for i < v.n {
		if p < len(pattern) && pattern[p] == '%' {
			star = p
			p++
			retry = i
			continue
		}
		if p < len(pattern) {
			r, size := utf8.DecodeRuneInString(pattern[p:])
			wildcard := r == '_'
			if r == '\\' && p+size < len(pattern) {
				r, size = utf8.DecodeRuneInString(pattern[p+1:])
				size++
			}
			c := v.at(i)
			if fold {
				c = asciiFoldByte(c)
			}
			if fold {
				r = unicode.ToLower(r)
			}
			if wildcard || rune(c) == r {
				i++
				p += size
				continue
			}
		}
		if star < 0 {
			return false, true
		}
		retry++
		i = retry
		p = star + 1
	}
	for p < len(pattern) && pattern[p] == '%' {
		p++
	}
	return p == len(pattern), true
}
