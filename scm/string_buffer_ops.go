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
import "strconv"
import "unicode"
import "unicode/utf8"
import "hash/maphash"
import "encoding/hex"
import "encoding/base64"

func compareBufferText(a, b Scmer, fold bool) int {
	if a.ptr == b.ptr && a.aux == b.aux {
		return 0
	}
	if a.GetTag() == tagConcat || b.GetTag() == tagConcat {
		return compareConcatText(a, b, fold)
	}
	return CompareTextCursors(a, b, fold, nil, nil)
}

func compareConcatText(a, b Scmer, fold bool) int {
	var left, right TextScratch
	return CompareTextCursors(a, b, fold, &left, &right)
}

// CompareTextCursors compares borrowed literal spans without decoding a pool.
// Shared spans compare by identity; otherwise comparison stops at the first mismatch.
func CompareTextCursors(a, b Scmer, fold bool, left, right *TextScratch) int {
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
		if c := compareTextSpans(x[:n], y[:n]); c != 0 {
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
	if !fold && compressedTextLen(a) != compressedTextLen(b) {
		return false
	}
	if a.ptr == b.ptr && a.aux == b.aux {
		return true
	}
	if !fold {
		return compareBufferText(a, b, false) == 0
	}
	if a.GetTag() == tagConcat || b.GetTag() == tagConcat {
		return equalConcatFold(a, b)
	}
	return equalCursorFold(a, b, nil, nil)
}

func equalConcatFold(a, b Scmer) bool {
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
	if a.GetTag() == tagConcat || b.GetTag() == tagConcat {
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
	cmp := CompareTextCursors(a, b, ac, x, y)
	if reverse {
		return cmp > 0
	}
	return cmp < 0
}

func bufferLike(s Scmer, pattern string, fold bool) bool {
	fold = fold && likePatternNeedsCaseFold(pattern)
	if s.GetTag() == tagConcat {
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
	return bufferLikeTokens(s, DecodeLikePattern(pattern, "\\"), fold)
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
	if s.GetTag() == tagConcat {
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
	if s.GetTag() == tagConcat {
		var scratch TextScratch
		return bufferCaseScratch(s, upper, &scratch)
	}
	return bufferCaseScratch(s, upper, nil)
}

func bufferCaseScratch(s Scmer, upper bool, scratch *TextScratch) Scmer {
	var cursor TextCursor
	cursor.Reset(s)
	var out []byte
	for {
		before := cursor.Position()
		r, width, ok := cursor.NextRune(scratch)
		if !ok {
			break
		}
		mapped := unicode.ToLower(r)
		if upper {
			mapped = unicode.ToUpper(r)
		}
		changed := mapped != r || r == utf8.RuneError && width == 1
		if out == nil && !changed {
			continue
		}
		if out == nil {
			out = make([]byte, before, compressedTextLen(s))
			var prefix TextCursor
			prefix.Reset(s)
			prefix.Read(out, scratch)
		}
		out = utf8.AppendRune(out, mapped)
	}
	if out == nil {
		return s
	}
	return NewString(unsafe.String(unsafe.SliceData(out), len(out)))
}

func writeBufferText(w io.Writer, s Scmer) {
	var cursor TextCursor
	cursor.Reset(s)
	for {
		chunk := cursor.nextChunk(nil)
		if len(chunk) == 0 {
			return
		}
		_, _ = w.Write(chunk)
		cursor.advanceChunk(len(chunk))
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
	if find == replacement {
		return s
	}
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
	if s.GetTag() == tagConcat {
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

// bufferLikeTokens keeps a rewindable compressed cursor for custom escapes.
func bufferLikeTokens(value Scmer, tokens []LikeToken, fold bool) bool {
	var cursor, star TextCursor
	var scratch TextScratch
	cursor.Reset(value)
	p, starPattern := 0, -1
	for {
		if p < len(tokens) && tokens[p].Kind == LikeMany {
			p++
			star, starPattern = cursor, p
			if p == len(tokens) {
				return true
			}
			continue
		}
		before := cursor
		r, _, ok := cursor.NextRune(&scratch)
		if !ok {
			for p < len(tokens) && tokens[p].Kind == LikeMany {
				p++
			}
			return p == len(tokens)
		}
		if p < len(tokens) {
			token := tokens[p]
			literal := token.Rune
			if fold {
				r = unicode.ToLower(r)
				literal = unicode.ToLower(literal)
			}
			if token.Kind == LikeAny || token.Kind == LikeLiteral && r == literal {
				p++
				continue
			}
		}
		if starPattern < 0 {
			return false
		}
		cursor = before
		if _, _, ok := star.NextRune(&scratch); !ok {
			return false
		}
		cursor = star
		p = starPattern
	}
}

func compareTextSpans(a, b []byte) int {
	if unsafe.SliceData(a) == unsafe.SliceData(b) {
		return 0
	}
	return bytes.Compare(a, b)
}

// appendBufferJSON renders encoding/json-compatible strings straight from
// pool spans. UTF-8 runes may cross span boundaries; ASCII runs copy at once.
func appendBufferJSON(dst []byte, s Scmer) []byte {
	dst = append(dst, '"')
	var cursor TextCursor
	cursor.Reset(s)
	for {
		chunk := cursor.nextChunk(nil)
		if len(chunk) == 0 {
			break
		}
		n := 0
		for n < len(chunk) {
			c := chunk[n]
			if c < 32 || c >= 128 || c == '"' || c == '\\' || c == '<' || c == '>' || c == '&' {
				break
			}
			n++
		}
		if n > 0 {
			dst = append(dst, chunk[:n]...)
			cursor.advanceChunk(n)
			continue
		}
		r, width, _ := cursor.NextRune(nil)
		switch r {
		case '\\', '"':
			dst = append(dst, '\\', byte(r))
		case '\b':
			dst = append(dst, `\b`...)
		case '\f':
			dst = append(dst, `\f`...)
		case '\n':
			dst = append(dst, `\n`...)
		case '\r':
			dst = append(dst, `\r`...)
		case '\t':
			dst = append(dst, `\t`...)
		case '\u2028':
			dst = append(dst, `\u2028`...)
		case '\u2029':
			dst = append(dst, `\u2029`...)
		default:
			if r < 32 || r == '<' || r == '>' || r == '&' {
				const hex = "0123456789abcdef"
				dst = append(dst, `\u00`...)
				dst = append(dst, hex[byte(r)>>4], hex[byte(r)&15])
			} else if r == utf8.RuneError && width == 1 {
				dst = append(dst, `\ufffd`...)
			} else {
				dst = utf8.AppendRune(dst, r)
			}
		}
	}
	return append(dst, '"')
}

// Numeric coercions inspect only the leading numeric token, never a complete
// report/query value. The ordinary stack buffer grows only for long numbers.
func bufferNumericToken(s Scmer, dst []byte) []byte {
	start := len(dst)
	var cursor TextCursor
	cursor.Reset(s)
	ch, ok := cursor.NextByte(nil)
	for ok && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v') {
		ch, ok = cursor.NextByte(nil)
	}
	if ok && (ch == '+' || ch == '-') {
		dst = append(dst, ch)
		ch, ok = cursor.NextByte(nil)
	}
	digits := 0
	for ok && ch >= '0' && ch <= '9' {
		dst = append(dst, ch)
		digits++
		ch, ok = cursor.NextByte(nil)
	}
	if ok && ch == '.' {
		dst = append(dst, ch)
		ch, ok = cursor.NextByte(nil)
		for ok && ch >= '0' && ch <= '9' {
			dst = append(dst, ch)
			digits++
			ch, ok = cursor.NextByte(nil)
		}
	}
	if digits == 0 {
		return dst[:start]
	}
	if ok && (ch == 'e' || ch == 'E') {
		exponent := len(dst)
		dst = append(dst, ch)
		ch, ok = cursor.NextByte(nil)
		if ok && (ch == '+' || ch == '-') {
			dst = append(dst, ch)
			ch, ok = cursor.NextByte(nil)
		}
		firstDigit := len(dst)
		for ok && ch >= '0' && ch <= '9' {
			dst = append(dst, ch)
			ch, ok = cursor.NextByte(nil)
		}
		if firstDigit == len(dst) {
			dst = dst[:exponent]
		}
	}
	return dst
}
func bufferBool(s Scmer) bool {
	if bufferStringLen(s) == 0 || equalBufferText(s, NewString("false"), true) {
		return false
	}
	var cursor TextCursor
	cursor.Reset(s)
	for {
		r, _, ok := cursor.NextRune(nil)
		if !ok {
			return true
		}
		if unicode.IsSpace(r) {
			continue
		}
		if !(r >= '0' && r <= '9') && r != '+' && r != '-' && r != '.' && r != 'N' && r != 'I' {
			return true
		}
		break
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(s.String()), 64)
	return err != nil || number != 0
}
func numericTextEqualsInt(value Scmer, integer int64) bool {
	if !value.IsBufferString() {
		return numericStringEqualsInt(value.String(), integer)
	}
	var local [128]byte
	token := bufferNumericToken(value, local[:0])
	return numericStringEqualsInt(unsafe.String(unsafe.SliceData(token), len(token)), integer)
}
