/*
	(c) 2026 Launix, Inh. Carl-Philip Hänsch
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

// parserFirstBytes returns the set of first bytes that parser p could match
// (after whitespace has been skipped by the parent OrParser). The second return
// value indicates whether the parser can match at end-of-input (empty match or
// EndParser).
func parserFirstBytes[T any](p Parser[T], visited map[any]bool) (bytes [256]bool, canMatchEOF bool) {
	// Cycle detection for recursive grammars
	key := any(p)
	if visited[key] {
		// Already visiting this parser — return empty to break the cycle.
		// The caller will union results, so omitting bytes here is safe:
		// the recursive reference can only match what the non-recursive
		// alternatives already cover.
		return bytes, false
	}
	visited[key] = true

	switch pp := p.(type) {
	case *AtomParser[T]:
		if len(pp.atom) == 0 {
			fillAllBytes(&bytes)
			return bytes, true
		}
		b := pp.atom[0]
		bytes[b] = true
		if pp.caseInsensitive {
			if b >= 'a' && b <= 'z' {
				bytes[b-32] = true
			} else if b >= 'A' && b <= 'Z' {
				bytes[b+32] = true
			}
		}
		return bytes, false

	case *RegexParser[T]:
		eof := regexFirstBytes(pp.rs, pp.caseInsensitive, &bytes)
		return bytes, eof

	case *AndParser[T]:
		// Walk children: if a child can match empty, also include the
		// next child's first bytes (because the empty-matching child
		// may be skipped and the successor determines the first byte).
		for _, child := range pp.subParser {
			cb, ceof := parserFirstBytes[T](child, visited)
			for i := range bytes {
				bytes[i] = bytes[i] || cb[i]
			}
			if !ceof {
				// This child is required, stop here
				return bytes, false
			}
		}
		// All children can match empty
		return bytes, true

	case *OrParser[T]:
		for _, child := range pp.subParser {
			cb, ceof := parserFirstBytes[T](child, visited)
			canMatchEOF = canMatchEOF || ceof
			for i := range bytes {
				bytes[i] = bytes[i] || cb[i]
			}
		}
		return bytes, canMatchEOF

	case *ManyParser[T]:
		// ManyParser requires at least 1 match of subParser
		return parserFirstBytes[T](pp.subParser, visited)

	case *NotParser[T]:
		return parserFirstBytes[T](pp.mainParser, visited)

	case *EmptyParser[T]:
		// Matches empty — no active first bytes, but canMatchEOF=true
		return bytes, true

	case *KleeneParser[T]:
		// Can match empty (zero repetitions). First bytes are from subParser
		// (what it matches when non-empty). canMatchEOF=true lets parent
		// And-nodes union successor bytes.
		cb, _ := parserFirstBytes[T](pp.subParser, visited)
		return cb, true

	case *MaybeParser[T]:
		// Can match empty. First bytes are from subParser.
		cb, _ := parserFirstBytes[T](pp.subParser, visited)
		return cb, true

	case *RestParser[T]:
		fillAllBytes(&bytes)
		return bytes, true

	case *EndParser[T]:
		// Only matches end of input, no bytes
		return bytes, true

	default:
		// Unknown parser type: conservative fallback
		fillAllBytes(&bytes)
		return bytes, true
	}
}

func fillAllBytes(bytes *[256]bool) {
	for i := range bytes {
		bytes[i] = true
	}
}

// regexFirstBytes extracts possible first bytes from a regex pattern string.
// Returns true if the pattern can match empty input.
func regexFirstBytes(rs string, caseInsensitive bool, bytes *[256]bool) bool {
	if len(rs) == 0 {
		return true // empty pattern matches empty
	}
	return regexFirstBytesAt(rs, 0, caseInsensitive, bytes)
}

// regexFirstBytesAt analyzes the regex element starting at pos, fills bytes
// with possible first bytes, and returns whether the pattern can match empty.
func regexFirstBytesAt(rs string, pos int, caseInsensitive bool, bytes *[256]bool) bool {
	if pos >= len(rs) {
		return true // end of pattern → matches empty
	}

	var elemBytes [256]bool
	nextPos := pos

	switch {
	case rs[pos] == '(':
		// Group: find matching close paren
		closePos := findGroupClose(rs, pos)
		if closePos < 0 {
			fillAllBytes(bytes)
			return true
		}
		nextPos = closePos + 1

		// Extract group content, skip ?: prefix for non-capturing groups
		groupContent := rs[pos+1 : closePos]
		if len(groupContent) >= 2 && groupContent[0] == '?' && groupContent[1] == ':' {
			groupContent = groupContent[2:]
		}

		// Split by | at depth 0 and analyze each alternative
		alts := splitAlternatives(groupContent)
		for _, alt := range alts {
			regexFirstBytesAt(alt, 0, caseInsensitive, &elemBytes)
		}

	case rs[pos] == '[':
		content, end := extractBracketExpr(rs, pos)
		if end < 0 {
			fillAllBytes(bytes)
			return true
		}
		nextPos = end

		if len(content) > 0 && content[0] == '^' {
			// Negated class: all bytes except excluded
			fillAllBytes(&elemBytes)
			table, ok := buildBitmap(content[1:], caseInsensitive)
			if ok {
				for i := 0; i < 256; i++ {
					if bitmapMatch(&table, byte(i)) {
						elemBytes[i] = false
					}
				}
			}
		} else {
			table, ok := buildBitmap(content, caseInsensitive)
			if ok {
				for i := 0; i < 256; i++ {
					if bitmapMatch(&table, byte(i)) {
						elemBytes[i] = true
					}
				}
			} else {
				fillAllBytes(&elemBytes)
			}
		}

	case rs[pos] == '.':
		fillAllBytes(&elemBytes)
		nextPos = pos + 1

	case rs[pos] == '\\':
		if pos+1 < len(rs) {
			ch := rs[pos+1]
			elemBytes[ch] = true
			if caseInsensitive {
				if ch >= 'a' && ch <= 'z' {
					elemBytes[ch-32] = true
				} else if ch >= 'A' && ch <= 'Z' {
					elemBytes[ch+32] = true
				}
			}
			nextPos = pos + 2
		} else {
			fillAllBytes(bytes)
			return true
		}

	default:
		ch := rs[pos]
		elemBytes[ch] = true
		if caseInsensitive {
			if ch >= 'a' && ch <= 'z' {
				elemBytes[ch-32] = true
			} else if ch >= 'A' && ch <= 'Z' {
				elemBytes[ch+32] = true
			}
		}
		nextPos = pos + 1
	}

	// Check quantifier
	optional := false
	if nextPos < len(rs) {
		switch rs[nextPos] {
		case '?', '*':
			optional = true
			nextPos++
		case '+':
			nextPos++
		}
	}

	// Add this element's bytes to result
	for i := 0; i < 256; i++ {
		if elemBytes[i] {
			bytes[i] = true
		}
	}

	if optional {
		// Element is optional, also consider what comes next
		eof := regexFirstBytesAt(rs, nextPos, caseInsensitive, bytes)
		return eof
	}

	return false
}

// findGroupClose finds the position of the closing ) for a group starting at pos.
// Returns -1 if not found.
func findGroupClose(rs string, pos int) int {
	depth := 1
	i := pos + 1
	for i < len(rs) && depth > 0 {
		if rs[i] == '\\' && i+1 < len(rs) {
			i += 2
			continue
		}
		if rs[i] == '(' {
			depth++
		} else if rs[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// splitAlternatives splits a regex string by | at depth 0.
func splitAlternatives(rs string) []string {
	var alts []string
	depth := 0
	start := 0
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\\' && i+1 < len(rs) {
			i++
			continue
		}
		switch rs[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '|':
			if depth == 0 {
				alts = append(alts, rs[start:i])
				start = i + 1
			}
		}
	}
	alts = append(alts, rs[start:])
	return alts
}

// buildCharMap constructs the character dispatch map for an OrParser.
func buildCharMap[T any](subParsers []Parser[T]) (cm *[256][]int, eofCandidates []int) {
	cm = &[256][]int{}

	for idx, child := range subParsers {
		visited := make(map[any]bool)
		childBytes, canEOF := parserFirstBytes[T](child, visited)

		// Check if any byte is set
		hasBytes := false
		for b := 0; b < 256; b++ {
			if childBytes[b] {
				hasBytes = true
				break
			}
		}

		if canEOF || !hasBytes {
			// Parser can match empty, or analysis was incomplete (e.g. left
			// recursion broke the cycle with no bytes). Add to all entries.
			if canEOF {
				eofCandidates = append(eofCandidates, idx)
			}
			for b := 0; b < 256; b++ {
				cm[b] = append(cm[b], idx)
			}
		} else {
			for b := 0; b < 256; b++ {
				if childBytes[b] {
					cm[b] = append(cm[b], idx)
				}
			}
		}
	}

	return cm, eofCandidates
}
