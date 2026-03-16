/*
	(c) 2026 Launix, Inh. Carl-Philip Hänsch
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import (
	"strings"
)

// detectFastPath analyzes a regex pattern string and returns a specialized
// matcher function if the pattern matches a known fast-path category.
// Returns nil if no specialization is available.
// The returned function takes the remaining input and returns the match length
// or -1 if no match.
func detectFastPath(rs string, caseInsensitive bool) func(string) int {
	// Category E: exact numeric patterns (most specific, check first)
	if fp := detectNumeric(rs); fp != nil {
		return fp
	}

	// Category F: escaped-string-body patterns
	if fp := detectEscapedStringBody(rs); fp != nil {
		return fp
	}

	// Category D: identifier [class1][class2]*
	if fp := detectIdentifier(rs, caseInsensitive); fp != nil {
		return fp
	}

	// Category G: literal-prefix + class \?[class]+
	if fp := detectLiteralPrefixClass(rs, caseInsensitive); fp != nil {
		return fp
	}

	// Category C: negated single-char [^X]+ / [^X]*
	if fp := detectNegatedClass(rs); fp != nil {
		return fp
	}

	// Category A: [class]+
	if fp := detectCharClassPlus(rs, caseInsensitive); fp != nil {
		return fp
	}

	// Category B: [class]*
	if fp := detectCharClassStar(rs, caseInsensitive); fp != nil {
		return fp
	}

	return nil
}

// buildBitmap parses a bracket expression content (without the surrounding [ ])
// and returns a 256-bit lookup table.
func buildBitmap(charClass string, caseInsensitive bool) ([4]uint64, bool) {
	var table [4]uint64
	i := 0
	for i < len(charClass) {
		var ch byte
		if charClass[i] == '\\' {
			if i+1 >= len(charClass) {
				return table, false
			}
			switch charClass[i+1] {
			case 'n':
				ch = '\n'
			case 'r':
				ch = '\r'
			case 't':
				ch = '\t'
			case '\\':
				ch = '\\'
			case '[', ']', '-', '^':
				ch = charClass[i+1]
			default:
				return table, false // unknown escape
			}
			i += 2
		} else {
			ch = charClass[i]
			i++
		}

		// Check for range: ch-end
		if i+1 < len(charClass) && charClass[i] == '-' {
			i++ // skip '-'
			var end byte
			if charClass[i] == '\\' {
				if i+1 >= len(charClass) {
					return table, false
				}
				switch charClass[i+1] {
				case 'n':
					end = '\n'
				case 'r':
					end = '\r'
				case 't':
					end = '\t'
				case '\\':
					end = '\\'
				case '[', ']', '-', '^':
					end = charClass[i+1]
				default:
					return table, false
				}
				i += 2
			} else {
				end = charClass[i]
				i++
			}
			if end < ch {
				return table, false
			}
			for c := ch; c <= end; c++ {
				setBit(&table, c, caseInsensitive)
			}
		} else {
			setBit(&table, ch, caseInsensitive)
		}
	}
	return table, true
}

func setBit(table *[4]uint64, b byte, caseInsensitive bool) {
	table[b>>6] |= 1 << (b & 63)
	if caseInsensitive {
		if b >= 'a' && b <= 'z' {
			upper := b - 32
			table[upper>>6] |= 1 << (upper & 63)
		} else if b >= 'A' && b <= 'Z' {
			lower := b + 32
			table[lower>>6] |= 1 << (lower & 63)
		}
	}
}

func bitmapMatch(table *[4]uint64, b byte) bool {
	return table[b>>6]&(1<<(b&63)) != 0
}

// extractBracketExpr tries to extract a bracket expression starting at rs[pos].
// Returns the content (without [ ]) and the position after the closing ].
// Returns "", -1 on failure.
func extractBracketExpr(rs string, pos int) (string, int) {
	if pos >= len(rs) || rs[pos] != '[' {
		return "", -1
	}
	depth := 0
	start := pos + 1
	for i := pos; i < len(rs); i++ {
		if rs[i] == '\\' && i+1 < len(rs) {
			i++ // skip escaped char
			continue
		}
		if rs[i] == '[' {
			depth++
		} else if rs[i] == ']' {
			depth--
			if depth == 0 {
				return rs[start:i], i + 1
			}
		}
	}
	return "", -1
}

// Category A: [class]+
func detectCharClassPlus(rs string, caseInsensitive bool) func(string) int {
	content, end := extractBracketExpr(rs, 0)
	if end < 0 || end >= len(rs) || rs[end] != '+' || end+1 != len(rs) {
		return nil
	}
	if len(content) > 0 && content[0] == '^' {
		return nil // negated class handled by Category C
	}
	table, ok := buildBitmap(content, caseInsensitive)
	if !ok {
		return nil
	}
	return func(input string) int {
		i := 0
		for i < len(input) && bitmapMatch(&table, input[i]) {
			i++
		}
		if i == 0 {
			return -1
		}
		return i
	}
}

// Category B: [class]*
func detectCharClassStar(rs string, caseInsensitive bool) func(string) int {
	content, end := extractBracketExpr(rs, 0)
	if end < 0 || end >= len(rs) || rs[end] != '*' || end+1 != len(rs) {
		return nil
	}
	if len(content) > 0 && content[0] == '^' {
		return nil
	}
	table, ok := buildBitmap(content, caseInsensitive)
	if !ok {
		return nil
	}
	return func(input string) int {
		i := 0
		for i < len(input) && bitmapMatch(&table, input[i]) {
			i++
		}
		return i // * always succeeds, even with 0 length
	}
}

// Category C: [^X]+ or [^X]* or (?:[^X])+
func detectNegatedClass(rs string) func(string) int {
	actual := rs
	plus := true

	// Handle (?:[^X])+ wrapper
	if strings.HasPrefix(rs, "(?:") {
		if !strings.HasSuffix(rs, ")+") && !strings.HasSuffix(rs, ")*") {
			return nil
		}
		if strings.HasSuffix(rs, ")*") {
			plus = false
		}
		actual = rs[3 : len(rs)-2]
	} else {
		if strings.HasSuffix(rs, "+") {
			actual = rs[:len(rs)-1]
			plus = true
		} else if strings.HasSuffix(rs, "*") {
			actual = rs[:len(rs)-1]
			plus = false
		} else {
			return nil
		}
	}

	// Must be [^...] with a single excluded byte
	if len(actual) < 4 || actual[0] != '[' || actual[1] != '^' || actual[len(actual)-1] != ']' {
		return nil
	}
	excluded := actual[2 : len(actual)-1]
	if len(excluded) != 1 {
		return nil // only single-byte exclusion for now
	}
	excludedByte := excluded[0]

	return func(input string) int {
		i := 0
		for i < len(input) && input[i] != excludedByte {
			i++
		}
		if plus && i == 0 {
			return -1
		}
		return i
	}
}

// Category D: [class1][class2]*
func detectIdentifier(rs string, caseInsensitive bool) func(string) int {
	content1, end1 := extractBracketExpr(rs, 0)
	if end1 < 0 {
		return nil
	}
	content2, end2 := extractBracketExpr(rs, end1)
	if end2 < 0 || end2 >= len(rs) || rs[end2] != '*' || end2+1 != len(rs) {
		return nil
	}
	if len(content1) > 0 && content1[0] == '^' {
		return nil
	}
	if len(content2) > 0 && content2[0] == '^' {
		return nil
	}
	table1, ok := buildBitmap(content1, caseInsensitive)
	if !ok {
		return nil
	}
	table2, ok := buildBitmap(content2, caseInsensitive)
	if !ok {
		return nil
	}
	return func(input string) int {
		if len(input) == 0 || !bitmapMatch(&table1, input[0]) {
			return -1
		}
		i := 1
		for i < len(input) && bitmapMatch(&table2, input[i]) {
			i++
		}
		return i
	}
}

// Category E: signed numeric patterns
func detectNumeric(rs string) func(string) int {
	if rs == `-?[0-9]+` {
		return matchSignedInt
	}
	if rs == `-?[0-9]+\.?[0-9]*(?:e-?[0-9]+)?` {
		return matchSignedFloat
	}
	return nil
}

func matchSignedInt(input string) int {
	i := 0
	if i < len(input) && input[i] == '-' {
		i++
	}
	start := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i == start {
		return -1
	}
	return i
}

func matchSignedFloat(input string) int {
	i := 0
	if i < len(input) && input[i] == '-' {
		i++
	}
	start := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i == start {
		return -1 // no digits
	}
	// optional .digits
	if i < len(input) && input[i] == '.' {
		i++
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			i++
		}
	}
	// optional exponent e-?digits
	if i < len(input) && input[i] == 'e' {
		expStart := i + 1
		j := expStart
		if j < len(input) && input[j] == '-' {
			j++
		}
		digitStart := j
		for j < len(input) && input[j] >= '0' && input[j] <= '9' {
			j++
		}
		if j > digitStart {
			i = j // only consume exponent if digits followed
		}
	}
	return i
}

// Category F: escaped-string-body (\\.|[^\\Q])*
func detectEscapedStringBody(rs string) func(string) int {
	// Match pattern: (\\.|[^\\Q])*
	// where Q is the quote character
	prefix := `(\\.|[^\\`
	suffix := `])*`
	if !strings.HasPrefix(rs, prefix) || !strings.HasSuffix(rs, suffix) {
		return nil
	}
	middle := rs[len(prefix) : len(rs)-len(suffix)]
	if len(middle) != 1 {
		return nil
	}
	quoteByte := middle[0]

	return func(input string) int {
		i := 0
		for i < len(input) {
			if input[i] == '\\' {
				if i+1 >= len(input) {
					break // trailing backslash, stop
				}
				i += 2 // skip escaped char
			} else if input[i] == quoteByte {
				break
			} else {
				i++
			}
		}
		return i // * quantifier, always succeeds (0 is valid)
	}
}

// Category G: literal-prefix + class: \?[class]+
func detectLiteralPrefixClass(rs string, caseInsensitive bool) func(string) int {
	// Look for \? prefix (escaped question mark)
	if !strings.HasPrefix(rs, `\?`) {
		return nil
	}
	rest := rs[2:]
	// Rest must be [class]+
	content, end := extractBracketExpr(rest, 0)
	if end < 0 || end >= len(rest) || rest[end] != '+' || end+1 != len(rest) {
		return nil
	}
	if len(content) > 0 && content[0] == '^' {
		return nil
	}
	table, ok := buildBitmap(content, caseInsensitive)
	if !ok {
		return nil
	}
	return func(input string) int {
		if len(input) == 0 || input[0] != '?' {
			return -1
		}
		i := 1
		for i < len(input) && bitmapMatch(&table, input[i]) {
			i++
		}
		if i == 1 {
			return -1 // need at least one char after ?
		}
		return i
	}
}
