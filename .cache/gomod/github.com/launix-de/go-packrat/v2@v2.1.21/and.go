/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

// AndParser accepts an input if all sub parsers accept the input sequentially
type AndParser[T any] struct {
	callback func(string, ...T) T
	subParser []Parser[T]
	buf []T
	depth int
}

// NewAndParser constructs a new AndParser with the given sub parsers. An AndParser accepts an input if all sub parsers accept the input sequentially.
func NewAndParser[T any](callback func(string, ...T) T, subparser ...Parser[T]) *AndParser[T] {
	return &AndParser[T]{callback: callback, subParser: subparser, buf: make([]T, len(subparser))}
}

// Set updates the sub parsers. This can be used to construct recursive parsers.
func (p *AndParser[T]) Set(embedded ...Parser[T]) {
	p.subParser = embedded
	p.buf = make([]T, len(embedded))
}

// Match matches all given parsers sequentially.
func (p *AndParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	var nodes []T
	if p.depth == 0 {
		nodes = p.buf[:0]
	} else {
		nodes = make([]T, 0, len(p.subParser))
	}
	p.depth++
	start := s.position
	startPosition := s.position
	for _, c := range p.subParser {
		node, ok := s.applyRule(c)
		if !ok {
			s.setPosition(startPosition)
			p.depth--
			return Node[T]{}, false
		}
		nodes = append(nodes, node.Payload)
	}

	result := Node[T]{Payload: p.callback(s.input[start:s.position], nodes...)}
	p.depth--
	return result, true
}
