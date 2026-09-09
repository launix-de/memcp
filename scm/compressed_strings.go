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

import "io"
import "hash"
import "unsafe"
import "strings"
import "crypto/md5"
import "crypto/sha1"
import "encoding/hex"
import "crypto/sha256"
import "encoding/base64"

func isCompressedText(s Scmer) bool { tag := s.GetTag(); return tag == tagCString || tag == tagBString }

func compressedTextLen(s Scmer) int {
	if s.IsBString() {
		return base64.StdEncoding.EncodedLen(int(auxVal(s.aux) & ((1 << 47) - 1)))
	}
	return int(auxVal(s.aux) & CStringLengthMask)
}

// Unlike makeStringView, this view also exposes Base64 text without allocating
// its encoded representation. It must not enter raw packed-byte comparisons.
func makeOperatorView(s Scmer) (operatorStringView, bool) {
	if !s.IsBString() {
		v, ok := makeStringView(s)
		return operatorStringView{stringView: v}, ok
	}
	val := auxVal(s.aux)
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if val>>47 != 0 {
		alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	}
	data := unsafe.String(s.ptr, int(val&((1<<47)-1)))
	return operatorStringView{stringView: stringView{data: data, n: base64.StdEncoding.EncodedLen(len(data))}, base64: alphabet}, true
}

// Writers consume bounded decoded chunks; no complete intermediate text is
// allocated. BString emits complete Base64 groups using the standard encoder.
func writeOperatorText(w io.Writer, s Scmer) {
	if !isCompressedText(s) {
		_, _ = io.WriteString(w, String(s))
		return
	}
	v, ok := makeOperatorView(s)
	if !ok {
		_, _ = io.WriteString(w, String(s))
		return
	}
	var buf [768]byte
	if s.IsBString() {
		enc := base64.StdEncoding
		if auxVal(s.aux)>>47 != 0 {
			enc = base64.URLEncoding
		}
		for pos := 0; pos < len(v.data); {
			n := min(576, len(v.data)-pos)
			enc.Encode(buf[:], unsafe.Slice(unsafe.StringData(v.data[pos:]), n))
			_, _ = w.Write(buf[:enc.EncodedLen(n)])
			pos += n
		}
		return
	}
	for pos := 0; pos < v.n; {
		n := min(len(buf), v.n-pos)
		v.decodeInto(buf[:n], pos, false)
		_, _ = w.Write(buf[:n])
		pos += n
	}
}

func compressedTextCase(s Scmer, upper bool) Scmer {
	if s.IsCString() {
		v, ok := makeStringView(s)
		if ok && v.format != 0 {
			f := v.format
			if (upper && f == 7) || (!upper && f == 6) {
				return s
			}
			if upper {
				switch f {
				case 2, 6, 11:
					return NewCString(s.ptr, f+1, uint8(v.offset), v.n)
				}
			} else {
				switch f {
				case 3, 7, 12:
					return NewCString(s.ptr, f-1, uint8(v.offset), v.n)
				}
			}
			alphabet := v.alphabet
			if alphabet == "" {
				alphabet = "0123456789abcdefABCDEF-"
			}
			changes := false
			for i := range alphabet {
				c := alphabet[i]
				if (upper && c >= 'a' && c <= 'z') || (!upper && c >= 'A' && c <= 'Z') {
					changes = true
					break
				}
			}
			if !changes {
				return s
			}
		}
	}
	if isCompressedText(s) {
		v, ok := makeOperatorView(s)
		if ok && (v.format != 0 || v.base64 != "") {
			out := make([]byte, v.n)
			v.decodeInto(out, 0, false)
			for i, c := range out {
				if upper && c >= 'a' && c <= 'z' {
					out[i] = c - 32
				}
				if !upper && c >= 'A' && c <= 'Z' {
					out[i] = c + 32
				}
			}
			return NewString(unsafe.String(unsafe.SliceData(out), len(out)))
		}
	}
	if upper {
		return NewString(strings.ToUpper(String(s)))
	}
	return NewString(strings.ToLower(String(s)))
}

// All registered compressed alphabets are ASCII. Raw/unknown CString formats
// retain the Unicode TrimSpace fallback. Direction 0 trims both ends.
func compressedTextTrim(s Scmer, direction int) Scmer {
	if isCompressedText(s) {
		v, ok := makeOperatorView(s)
		if ok && (v.format != 0 || v.base64 != "") {
			start, end := 0, v.n
			space := func(c byte) bool {
				return c == ' ' || c == '\t' || c == '\n' || c == '\r' || (direction == 0 && (c == '\v' || c == '\f'))
			}
			if direction <= 0 {
				for start < end && space(v.at(start)) {
					start++
				}
			}
			if direction >= 0 {
				for end > start && space(v.at(end-1)) {
					end--
				}
			}
			if start == 0 && end == v.n {
				return s
			}
			return cstringSubstring(s, start, end)
		}
	}
	if direction < 0 {
		return NewString(strings.TrimLeft(String(s), " \t\n\r"))
	}
	if direction > 0 {
		return NewString(strings.TrimRight(String(s), " \t\n\r"))
	}
	return NewString(strings.TrimSpace(String(s)))
}

func decodeTextHex(s Scmer) ([]byte, error) {
	if !isCompressedText(s) {
		return hex.DecodeString(String(s))
	}
	v, ok := makeOperatorView(s)
	if !ok {
		return hex.DecodeString(String(s))
	}
	out := make([]byte, v.n/2)
	nibble := func(c byte) (byte, bool) {
		if c >= '0' && c <= '9' {
			return c - '0', true
		}
		c |= 32
		if c >= 'a' && c <= 'f' {
			return c - 'a' + 10, true
		}
		return 0, false
	}
	for i := range out {
		a, b := v.at(i*2), v.at(i*2+1)
		x, valid := nibble(a)
		if !valid {
			return out[:i], hex.InvalidByteError(a)
		}
		y, valid := nibble(b)
		if !valid {
			return out[:i], hex.InvalidByteError(b)
		}
		out[i] = x<<4 | y
	}
	if v.n&1 != 0 {
		c := v.at(v.n - 1)
		if _, valid := nibble(c); !valid {
			return out, hex.InvalidByteError(c)
		}
		return out, hex.ErrLength
	}
	return out, nil
}

func (v operatorStringView) decodeBase64Into(dst []byte, start int, fold bool) {
	i := 0
	for i < len(dst) && (start+i)&3 != 0 {
		dst[i] = v.at(start + i)
		i++
	}
	pos := (start + i) / 4 * 3
	groups := min((len(dst)-i)/4, (len(v.data)-pos)/3)
	if groups > 0 {
		enc := base64.StdEncoding
		if v.base64[62] == '-' {
			enc = base64.URLEncoding
		}
		enc.Encode(dst[i:i+groups*4], unsafe.Slice(unsafe.StringData(v.data[pos:]), groups*3))
		i += groups * 4
	}
	for ; i < len(dst); i++ {
		dst[i] = v.at(start + i)
	}
	if fold {
		for i, c := range dst {
			dst[i] = asciiFoldByte(c)
		}
	}
}

// Base64 text is represented by its source bytes until an output consumer
// actually needs encoded text. The Scmer pointer keeps that immutable string alive.
func encodeTextBase64(s Scmer) Scmer {
	text := String(s)
	return NewBString(unsafe.StringData(text), len(text), false)
}

func encodeTextHex(s Scmer) Scmer {
	v, ok := makeOperatorView(s)
	if !ok {
		return NewString(hex.EncodeToString([]byte(String(s))))
	}
	out := make([]byte, hex.EncodedLen(v.n))
	var chunk [256]byte
	for pos := 0; pos < v.n; {
		n := min(len(chunk), v.n-pos)
		v.decodeInto(chunk[:n], pos, false)
		hex.Encode(out[pos*2:], chunk[:n])
		pos += n
	}
	return NewString(unsafe.String(unsafe.SliceData(out), len(out)))
}

// Only mandatory literals can reject a pattern. Escaped wildcards are literals;
// Unicode patterns retain the established case-folding matcher.
func likeAlphabetPossible(v operatorStringView, pattern string, fold bool) bool {
	index := int(v.format)
	if v.base64 != "" {
		index = 17
		if v.base64[62] == '-' {
			index = 18
		}
	}
	if index == 0 || index >= len(textAlphabetMasks) {
		return true
	}
	allowed := textAlphabetMasks[index]
	if fold {
		allowed = textFoldedAlphabetMasks[index]
	}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c >= 128 {
			return true
		}
		if c == '%' || c == '_' {
			continue
		}
		if c == '\\' && i+1 < len(pattern) {
			i++
			c = pattern[i]
			if c >= 128 {
				return true
			}
		}
		if fold {
			c = asciiFoldByte(c)
		}
		if allowed[c>>6]&(uint64(1)<<(c&63)) == 0 {
			return false
		}
	}
	return true
}

func appendOperatorText(w *strings.Builder, s Scmer) {
	if !isCompressedText(s) {
		w.WriteString(String(s))
		return
	}
	v, ok := makeOperatorView(s)
	if !ok {
		w.WriteString(String(s))
		return
	}
	var chunk [768]byte
	for pos := 0; pos < v.n; {
		n := min(len(chunk), v.n-pos)
		v.decodeInto(chunk[:n], pos, false)
		w.Write(chunk[:n])
		pos += n
	}
}

func (w *schemeTextWriter) writeCompressedText(s Scmer) {
	// Small strings are faster through the existing scalar decoder; streaming
	// setup pays off once it avoids a substantial intermediate allocation.
	if compressedTextLen(s) < 128 {
		w.WriteString(String(s))
		return
	}
	v, ok := makeOperatorView(s)
	if !ok {
		w.WriteString(String(s))
		return
	}
	var chunk [768]byte
	for pos := 0; pos < v.n; {
		n := min(len(chunk), v.n-pos)
		v.decodeInto(chunk[:n], pos, false)
		w.Write(chunk[:n])
		pos += n
	}
}

// The final two table entries are operator-only Base64 alphabets, not format IDs.
var textAlphabetMasks, textFoldedAlphabetMasks = func() ([19][2]uint64, [19][2]uint64) {
	var plain, folded [19][2]uint64
	for i := 1; i < len(plain); i++ {
		a := CStringAlphabet(uint8(i))
		switch i {
		case 6:
			a = "0123456789abcdef-"
		case 7:
			a = "0123456789ABCDEF-"
		case 17:
			a = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
		case 18:
			a = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_="
		}
		for j := range a {
			c := a[j]
			plain[i][c>>6] |= uint64(1) << (c & 63)
			c = asciiFoldByte(c)
			folded[i][c>>6] |= uint64(1) << (c & 63)
		}
	}
	return plain, folded
}()

// Keep interface-based hash setup out of the declarations so their ordinary
// short/plain string paths remain eligible for generated JIT inlining.
func compressedTextDigest(s Scmer, kind int) Scmer {
	var h hash.Hash
	switch kind {
	case 0:
		h = md5.New()
	case 1:
		h = sha1.New()
	case 2:
		h = sha256.New()
	default:
		panic("invalid compressed text digest")
	}
	writeOperatorText(h, s)
	return NewString(hex.EncodeToString(h.Sum(nil)))
}

// Keep Base64 state out of stringView: that smaller type is passed by value in
// comparison and index-sort loops, which must not pay for operator-only fields.
type operatorStringView struct {
	stringView
	base64 string
}

func (s operatorStringView) at(i int) byte {
	if s.base64 != "" {
		pos, part := (i/4)*3, i&3
		if part == 0 {
			return s.base64[s.data[pos]>>2]
		}
		if part == 1 {
			bits := (s.data[pos] & 3) << 4
			if pos+1 < len(s.data) {
				bits |= s.data[pos+1] >> 4
			}
			return s.base64[bits]
		}
		if part == 2 {
			if pos+1 >= len(s.data) {
				return '='
			}
			bits := (s.data[pos+1] & 15) << 2
			if pos+2 < len(s.data) {
				bits |= s.data[pos+2] >> 6
			}
			return s.base64[bits]
		}
		if pos+2 >= len(s.data) {
			return '='
		}
		return s.base64[s.data[pos+2]&63]
	}
	return s.stringView.at(i)
}

func (v operatorStringView) decodeInto(dst []byte, start int, fold bool) {
	if v.base64 != "" {
		v.decodeBase64Into(dst, start, fold)
		return
	}
	v.stringView.decodeInto(dst, start, fold)
}

func (v operatorStringView) matchRange(start int, pattern string, fold bool) (bool, bool) {
	if v.base64 == "" {
		return v.stringView.matchRange(start, pattern, fold)
	}
	if start < 0 || len(pattern) > v.n-start {
		return false, true
	}
	for i := range pattern {
		c, p := v.at(start+i), pattern[i]
		if fold {
			if p >= 128 {
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

func bstringRawText(s Scmer) Scmer {
	return NewString(unsafe.String(s.ptr, int(auxVal(s.aux)&((1<<47)-1))))
}
