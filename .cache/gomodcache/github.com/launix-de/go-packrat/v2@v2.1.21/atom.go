/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import (
	"strings"
)

type AtomParser[T any] struct {
	value           T
	skipWs          bool
	atom            string
	caseInsensitive bool
}

func NewAtomParser[T any](value T, str string, caseInsensitive bool, skipWs bool) *AtomParser[T] {
	p := &AtomParser[T]{value: value, skipWs: skipWs, atom: str, caseInsensitive: caseInsensitive}
	return p
}

// Match matches only the given string. If skipWs is set to true, leading whitespace according to the scanner's skip regexp is skipped, but not matched by the parser.
func (p *AtomParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	startPosition := s.position

	if p.skipWs {
		s.Skip()

		if !s.isAtBreak() {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	}

	atomLen := len(p.atom)
	if len(s.remainingInput) < atomLen {
		s.setPosition(startPosition)
		return Node[T]{}, false
	}
	candidate := s.remainingInput[:atomLen]
	if p.caseInsensitive {
		if !strings.EqualFold(candidate, p.atom) {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	} else {
		if candidate != p.atom {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	}
	s.move(atomLen)

	if p.skipWs {
		if !s.isAtBreak() {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
	}

	return Node[T]{Payload: p.value}, true
}
