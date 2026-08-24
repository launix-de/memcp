/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import (
	"regexp"
)

type RegexParser[T any] struct {
	callback        func(string) T
	regex           *regexp.Regexp
	fastPath        func(string) int
	skipWs          bool
	caseInsensitive bool
	rs              string
}

func NewRegexParser[T any](callback func(string) T, rs string, caseInsensitive bool, skipWs bool) *RegexParser[T] {
	fp := detectFastPath(rs, caseInsensitive)
	if fp != nil {
		return &RegexParser[T]{callback: callback, fastPath: fp, skipWs: skipWs, caseInsensitive: caseInsensitive, rs: rs}
	}
	prefix := ""
	if caseInsensitive {
		prefix += "(?i)"
	}
	prefix += "^"
	r := regexp.MustCompile(prefix + rs)
	return &RegexParser[T]{callback: callback, regex: r, skipWs: skipWs, caseInsensitive: caseInsensitive, rs: rs}
}

// Regex matches only the given regexp. If skipWs is set to true, leading whitespace according to the scanner's skip regexp is skipped, but not matched by the parser.
// Regex panics if rs is not a valid regex string.
func (p *RegexParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	startPosition := s.position
	if p.skipWs {
		s.Skip()
		if !s.isAtBreak() {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	}

	if p.fastPath != nil {
		matchLen := p.fastPath(s.remainingInput)
		if matchLen < 0 {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
		matchedStr := s.remainingInput[:matchLen]
		s.move(matchLen)

		if p.skipWs {
			if !s.isAtBreak() {
				s.setPosition(startPosition)
				return Node[T]{}, false
			}
		}

		return Node[T]{Payload: p.callback(matchedStr)}, true
	}

	matched := s.MatchRegexp(p.regex)
	if matched == nil {
		s.setPosition(startPosition)
		return Node[T]{}, false
	}

	if p.skipWs {
		if !s.isAtBreak() {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	}

	return Node[T]{Payload: p.callback(*matched)}, true
}
