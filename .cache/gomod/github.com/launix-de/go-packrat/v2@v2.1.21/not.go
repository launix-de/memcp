/*
	(c) 2024 Launix, Inh. Carl-Philip Hänsch
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

// AndParser accepts an input if all sub parsers accept the input sequentially
type NotParser[T any] struct {
	mainParser Parser[T]
	notParser []Parser[T]
}

// NewAndParser constructs a new AndParser with the given sub parsers. An AndParser accepts an input if all sub parsers accept the input sequentially.
func NewNotParser[T any](main Parser[T], subparser ...Parser[T]) *NotParser[T] {
	return &NotParser[T]{mainParser: main, notParser: subparser}
}

// Match matches all given parsers sequentially.
func (p *NotParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	start := s.position
	node, ok := s.applyRule(p.mainParser)
	if !ok {
		return node, ok
	}
	cont := s.position

	for _, c := range p.notParser {
		s.setPosition(start)
		_, ok := s.applyRule(c)
		if ok { // a not-parser matched, so reset and tell it dosen't work
			s.setPosition(start)
			return Node[T]{}, false
		}
	}
	s.setPosition(cont) // go back to old scanner position

	return node, true
}

