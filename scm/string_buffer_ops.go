/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "bytes"
import "io"
import "fmt"
import "unsafe"
import "strings"
import "unicode"
import "unicode/utf8"
import "hash/maphash"
import "encoding/hex"
import "encoding/base64"

func compareBufferText(a, b Scmer, fold bool) int {
	if a.ptr == b.ptr && a.aux == b.aux {
		return 0
	}
	if a.GetTag() == tagLZ4 || b.GetTag() == tagLZ4 {
		return compareLZ4Text(a, b, fold)
	}
	return compareTextCursors(a, b, fold, nil, nil)
}

func compareLZ4Text(a, b Scmer, fold bool) int {
	var left, right TextScratch
	return compareTextCursors(a, b, fold, &left, &right)
}

func compareTextCursors(a, b Scmer, fold bool, left, right *TextScratch) int {
	var av, bv TextCursor
	av.Reset(a)
	bv.Reset(b)
	if fold {
		for {
			x, _, xok := av.NextRune(left)
			y, _, yok := bv.NextRune(right)
			if !xok || !yok {
				if xok {
					return 1
				}
				if yok {
					return -1
				}
				return 0
			}
			x, y = unicode.ToLower(x), unicode.ToLower(y)
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
		}
	}
	for {
		x, y := av.nextChunk(left), bv.nextChunk(right)
		n := min(len(x), len(y))
		if c := bytes.Compare(x[:n], y[:n]); c != 0 {
			return c
		}
		if n == 0 {
			return compareLength(len(x), len(y))
		}
		av.advanceChunk(n)
		bv.advanceChunk(n)
	}
}

func equalBufferText(a, b Scmer, fold bool) bool {
	if a.ptr == b.ptr && a.aux == b.aux {
		return true
	}
	if !fold {
		return compareBufferText(a, b, false) == 0
	}
	if a.GetTag() == tagLZ4 || b.GetTag() == tagLZ4 {
		return equalLZ4Fold(a, b)
	}
	return equalCursorFold(a, b, nil, nil)
}

func equalLZ4Fold(a, b Scmer) bool {
	var left, right TextScratch
	return equalCursorFold(a, b, &left, &right)
}

func equalCursorFold(a, b Scmer, left, right *TextScratch) bool {
	var av, bv TextCursor
	av.Reset(a)
	bv.Reset(b)
	for {
		x, _, xok := av.NextRune(left)
		y, _, yok := bv.NextRune(right)
		if !xok || !yok {
			return xok == yok
		}
		if x == y {
			continue
		}
		if x > y {
			x, y = y, x
		}
		for r := unicode.SimpleFold(x); r != x && r <= y; r = unicode.SimpleFold(r) {
			if r == y {
				x = y
				break
			}
		}
		if x != y {
			return false
		}
	}
}

func generalBufferLess(a, b Scmer, reverse bool) bool {
	if a.ptr == b.ptr && a.aux == b.aux {
		return false
	}
	if a.GetTag() == tagLZ4 || b.GetTag() == tagLZ4 {
		var x, y TextScratch
		return generalBufferLessScratch(a, b, reverse, &x, &y)
	}
	return generalBufferLessScratch(a, b, reverse, nil, nil)
}

func generalBufferLessScratch(a, b Scmer, reverse bool, x, y *TextScratch) bool {
	classify := func(s Scmer, scratch *TextScratch) (bool, byte) {
		var c TextCursor
		c.Reset(s)
		first, ok := c.NextByte(scratch)
		if !ok {
			return true, 0
		}
		first = asciiFoldByte(first)
		second, _ := c.NextByte(scratch)
		if first == 'a' && asciiFoldByte(second) == 'a' {
			return false, 0
		}
		return first >= 'a' && first <= 'z', first
	}
	ac, ak := classify(a, x)
	bc, bk := classify(b, y)
	if ac != bc {
		return ac
	}
	if ac && ak != bk {
		if reverse {
			return ak > bk
		}
		return ak < bk
	}
	cmp := compareTextCursors(a, b, ac, x, y)
	if reverse {
		return cmp > 0
	}
	return cmp < 0
}

func bufferLike(s Scmer, pattern string, fold bool) bool {
	fold = fold && likePatternNeedsCaseFold(pattern)
	if s.GetTag() == tagLZ4 {
		var scratch TextScratch
		return bufferLikeScratch(s, pattern, fold, &scratch)
	}
	return bufferLikeScratch(s, pattern, fold, nil)
}

func bufferLikeScratch(s Scmer, pattern string, fold bool, scratch *TextScratch) bool {
	// The common literal/contains forms use a bounded KMP table. Long patterns
	// retain the allocation-free wildcard cursor rather than allocating a table.
	if !strings.ContainsAny(pattern, "_\\") && len(pattern) <= 1024 {
		mode := 0
		literal := pattern
		if strings.HasPrefix(literal, "%") {
			mode |= 1
			literal = literal[1:]
		}
		if strings.HasSuffix(literal, "%") {
			mode |= 2
			literal = literal[:len(literal)-1]
		}
		if !strings.Contains(literal, "%") {
			return bufferLiteralLike(s, literal, fold, mode, scratch)
		}
	}
	var cursor, star TextCursor
	cursor.Reset(s)
	pos, starPattern := 0, -1
	for {
		if pos < len(pattern) && pattern[pos] == '%' {
			for pos < len(pattern) && pattern[pos] == '%' {
				pos++
			}
			star, starPattern = cursor, pos
			if pos == len(pattern) {
				return true
			}
			continue
		}
		before := cursor
		r, _, ok := cursor.NextRune(scratch)
		if !ok {
			for pos < len(pattern) && pattern[pos] == '%' {
				pos++
			}
			return pos == len(pattern)
		}
		if pos < len(pattern) {
			p, size := utf8.DecodeRuneInString(pattern[pos:])
			wild := p == '_'
			if p == '\\' && pos+size < len(pattern) {
				q, n := utf8.DecodeRuneInString(pattern[pos+size:])
				p = q
				size += n
				wild = false
			}
			if fold {
				r, p = unicode.ToLower(r), unicode.ToLower(p)
			}
			if wild || r == p {
				pos += size
				continue
			}
		}
		if starPattern < 0 {
			return false
		}
		cursor = before
		if _, _, ok := star.NextRune(scratch); !ok {
			return false
		}
		cursor, pos = star, starPattern
	}
}

// mode bits: 1 permits a leading arbitrary prefix; 2 permits a trailing suffix.
func bufferLiteralLike(s Scmer, pattern string, fold bool, mode int, scratch *TextScratch) bool {
	var symbols [1024]rune
	var prefix [1024]int
	n := 0
	if fold {
		for _, r := range pattern {
			symbols[n] = unicode.ToLower(r)
			n++
		}
	} else {
		for i := 0; i < len(pattern); i++ {
			symbols[n] = rune(pattern[i])
			n++
		}
	}
	if n == 0 {
		return mode != 0 || compressedTextLen(s) == 0
	}
	for i, j := 1, 0; i < n; i++ {
		for j > 0 && symbols[i] != symbols[j] {
			j = prefix[j-1]
		}
		if symbols[i] == symbols[j] {
			j++
		}
		prefix[i] = j
	}
	var cursor TextCursor
	cursor.Reset(s)
	matched, position := 0, 0
	for {
		var r rune
		var ok bool
		if fold {
			r, _, ok = cursor.NextRune(scratch)
			r = unicode.ToLower(r)
		} else {
			var b byte
			b, ok = cursor.NextByte(scratch)
			r = rune(b)
		}
		if !ok {
			return false
		}
		for matched > 0 && r != symbols[matched] {
			matched = prefix[matched-1]
		}
		if r == symbols[matched] {
			matched++
		}
		position++
		if matched == n {
			if mode&1 == 0 && position != n {
				return false
			}
			if mode&2 != 0 {
				return true
			}
			probe := cursor
			if _, ok := probe.NextByte(scratch); !ok {
				return true
			}
			if mode&1 == 0 {
				return false
			}
			matched = prefix[matched-1]
		}
		if mode&1 == 0 && position >= n {
			return false
		}
	}
}

// BufferBytePrefix is used by OverlayBlob to recognize escape/reference
// markers without materializing an ordinary inline value.
func BufferBytePrefix(s Scmer, dst []byte) int {
	if s.GetTag() == tagLZ4 {
		var scratch TextScratch
		var c TextCursor
		c.Reset(s)
		return c.Read(dst, &scratch)
	}
	var c TextCursor
	c.Reset(s)
	return c.Read(dst, nil)
}

func bufferCase(s Scmer, upper bool) Scmer {
	if s.GetTag() == tagLZ4 {
		var scratch TextScratch
		return bufferCaseScratch(s, upper, &scratch)
	}
	return bufferCaseScratch(s, upper, nil)
}

func bufferCaseScratch(s Scmer, upper bool, scratch *TextScratch) Scmer {
	var c TextCursor
	c.Reset(s)
	result := make([]byte, 0, compressedTextLen(s))
	for {
		r, _, ok := c.NextRune(scratch)
		if !ok {
			break
		}
		if upper {
			r = unicode.ToUpper(r)
		} else {
			r = unicode.ToLower(r)
		}
		result = utf8.AppendRune(result, r)
	}
	return NewString(unsafe.String(unsafe.SliceData(result), len(result)))
}

func writeBufferText(w io.Writer, s Scmer) {
	var scratch TextScratch
	var cursor TextCursor
	var chunk [768]byte
	cursor.Reset(s)
	for {
		n := cursor.Read(chunk[:], &scratch)
		if n == 0 {
			return
		}
		if _, err := w.Write(chunk[:n]); err != nil {
			panic(err)
		}
	}
}

func appendBufferText(w *strings.Builder, s Scmer) {
	var scratch TextScratch
	var cursor TextCursor
	var chunk [768]byte
	cursor.Reset(s)
	for {
		n := cursor.Read(chunk[:], &scratch)
		if n == 0 {
			return
		}
		w.Write(chunk[:n])
	}
}

func hashBufferText(w *schemeTextWriter, s Scmer) {
	var scratch TextScratch
	var cursor TextCursor
	var chunk [768]byte
	cursor.Reset(s)
	for {
		n := cursor.Read(chunk[:], &scratch)
		if n == 0 {
			return
		}
		w.Write(chunk[:n])
	}
}

func hashBufferMap(w *maphash.Hash, s Scmer) {
	var scratch TextScratch
	var cursor TextCursor
	var chunk [768]byte
	cursor.Reset(s)
	for {
		n := cursor.Read(chunk[:], &scratch)
		if n == 0 {
			return
		}
		w.Write(chunk[:n])
	}
}

func encodeBufferBase64(s Scmer) Scmer {
	out := make([]byte, bufferStringLen(s))
	var cursor TextCursor
	var scratch TextScratch
	cursor.Reset(s)
	cursor.Read(out, &scratch)
	return NewBString(unsafe.SliceData(out), len(out), false, false)
}

func encodeBufferHex(s Scmer) Scmer {
	out := make([]byte, hex.EncodedLen(bufferStringLen(s)))
	var cursor TextCursor
	var scratch TextScratch
	var chunk [256]byte
	cursor.Reset(s)
	for pos := 0; ; {
		n := cursor.Read(chunk[:], &scratch)
		if n == 0 {
			break
		}
		hex.Encode(out[pos:], chunk[:n])
		pos += 2 * n
	}
	return NewString(unsafe.String(unsafe.SliceData(out), len(out)))
}

func decodeBufferHex(s Scmer) ([]byte, error) {
	out := make([]byte, bufferStringLen(s)/2)
	var cursor TextCursor
	var scratch TextScratch
	cursor.Reset(s)
	var pair [2]byte
	for i := range out {
		cursor.Read(pair[:], &scratch)
		if _, err := hex.Decode(out[i:i+1], pair[:]); err != nil {
			return out[:i], err
		}
	}
	if last, ok := cursor.NextByte(&scratch); ok {
		if !strings.ContainsRune("0123456789abcdefABCDEF", rune(last)) {
			return out, hex.InvalidByteError(last)
		}
		return out, hex.ErrLength
	}
	return out, nil
}

func decodeBufferBase64(s Scmer) Scmer {
	out := make([]byte, base64.StdEncoding.DecodedLen(bufferStringLen(s)))
	var cursor TextCursor
	var scratch TextScratch
	cursor.Reset(s)
	var input [4]byte
	var offsets [4]int
	count, written := 0, 0
	padded := false
	fail := func(pos int) { panic("error while decoding base64: " + fmt.Sprint(base64.CorruptInputError(pos))) }
	for {
		v, ok := cursor.NextByte(&scratch)
		if !ok {
			break
		}
		if v == '\r' || v == '\n' {
			continue
		}
		if padded {
			fail(cursor.Position() - 1)
		}
		offsets[count] = cursor.Position() - 1
		input[count] = v
		count++
		if count == 4 {
			n, err := base64.StdEncoding.Decode(out[written:], input[:])
			if err != nil {
				pos := int(err.(base64.CorruptInputError))
				fail(offsets[min(pos, 3)])
			}
			written += n
			padded = input[2] == '=' || input[3] == '='
			count = 0
		}
	}
	if count != 0 {
		n, err := base64.StdEncoding.Decode(out[written:], input[:count])
		written += n
		if err != nil {
			pos := int(err.(base64.CorruptInputError))
			if pos < count {
				fail(offsets[pos])
			}
			fail(cursor.Position())
		}
	}
	return NewString(unsafe.String(unsafe.SliceData(out), written))
}

func serializeBufferString(w *schemeTextWriter, s Scmer) {
	var cursor TextCursor
	var scratch TextScratch
	cursor.Reset(s)
	for {
		v, ok := cursor.NextByte(&scratch)
		if !ok {
			return
		}
		switch v {
		case '\\', '"':
			w.WriteByte('\\')
			w.WriteByte(v)
		case '\r':
			w.WriteString("\\r")
		case '\n':
			w.WriteString("\\n")
		default:
			w.WriteByte(v)
		}
	}
}

func bufferReplace(s Scmer, find, replacement string) Scmer {
	var cursor, output TextCursor
	var scratch TextScratch
	cursor.Reset(s)
	output.Reset(s)
	var result strings.Builder
	if find == "" {
		result.Grow(bufferStringLen(s) + len(replacement))
		result.WriteString(replacement)
		var bytes [4]byte
		for {
			before := cursor
			_, size, ok := cursor.NextRune(&scratch)
			if !ok {
				break
			}
			before.Read(bytes[:size], &scratch)
			result.Write(bytes[:size])
			result.WriteString(replacement)
		}
		return NewString(result.String())
	}
	var prefix [1024]int
	if len(find) <= len(prefix) {
		for i, j := 1, 0; i < len(find); i++ {
			for j > 0 && find[i] != find[j] {
				j = prefix[j-1]
			}
			if find[i] == find[j] {
				j++
			}
			prefix[i] = j
		}
	}
	changed := false
	for {
		start, end, ok := nextBufferMatch(&cursor, find, prefix[:], &scratch)
		if !ok {
			if !changed {
				return s
			}
			appendCursorBytes(&result, &output, bufferStringLen(s)-output.Position(), &scratch, true)
			return NewString(result.String())
		}
		if !changed {
			result.Grow(bufferStringLen(s))
			changed = true
		}
		appendCursorBytes(&result, &output, start-output.Position(), &scratch, true)
		appendCursorBytes(&result, &output, end-start, &scratch, false)
		result.WriteString(replacement)
	}
}

func appendCursorBytes(w *strings.Builder, cursor *TextCursor, count int, scratch *TextScratch, write bool) {
	var chunk [768]byte
	for count > 0 {
		n := cursor.Read(chunk[:min(count, len(chunk))], scratch)
		if n == 0 {
			panic("short compressed replacement input")
		}
		if write {
			w.Write(chunk[:n])
		}
		count -= n
	}
}

func nextBufferMatch(cursor *TextCursor, find string, prefix []int, scratch *TextScratch) (int, int, bool) {
	if len(find) > len(prefix) {
		for {
			start := cursor.Position()
			probe := *cursor
			matched := true
			for i := 0; i < len(find); i++ {
				v, ok := probe.NextByte(scratch)
				if !ok || v != find[i] {
					matched = false
					break
				}
			}
			if matched {
				*cursor = probe
				return start, cursor.Position(), true
			}
			if _, ok := cursor.NextByte(scratch); !ok {
				return 0, 0, false
			}
		}
	}
	matched := 0
	for {
		v, ok := cursor.NextByte(scratch)
		if !ok {
			return 0, 0, false
		}
		for matched > 0 && v != find[matched] {
			matched = prefix[matched-1]
		}
		if v == find[matched] {
			matched++
		}
		if matched == len(find) {
			return cursor.Position() - len(find), cursor.Position(), true
		}
	}
}

// BufferSlice preserves the same backing buffer and can cross LZ4 blocks without allocating a descriptor or borrowing decoder scratch.
func BufferSlice(s Scmer, start, end int) Scmer { return bufferSubstring(s, start, end) }

func bufferTrim(s Scmer, direction int) Scmer {
	if s.GetTag() == tagLZ4 {
		var scratch TextScratch
		return bufferTrimScratch(s, direction, &scratch)
	}
	return bufferTrimScratch(s, direction, nil)
}

func bufferTrimScratch(s Scmer, direction int, scratch *TextScratch) Scmer {
	var c TextCursor
	c.Reset(s)
	start, end := 0, 0
	leading := true
	for {
		r, size, ok := c.NextRune(scratch)
		if !ok {
			break
		}
		space := unicode.IsSpace(r)
		if direction != 0 {
			space = r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}
		if leading && space {
			start = c.Position()
		} else {
			leading = false
		}
		if !space {
			end = c.Position()
		}
		_ = size
	}
	if direction > 0 {
		start = 0
	}
	if direction < 0 {
		end = compressedTextLen(s)
	}
	if end < start {
		end = start
	}
	return bufferSubstring(s, start, end)
}
