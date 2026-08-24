/*
	(c) 2019-2026 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "sync"

// AndParser accepts an input if all sub parsers accept the input sequentially
type AndParser[T any] struct {
	callback  func(string, ...T) T
	subParser []Parser[T]
	buf       sync.Pool
}

// NewAndParser constructs a new AndParser with the given sub parsers. An AndParser accepts an input if all sub parsers accept the input sequentially.
func NewAndParser[T any](callback func(string, ...T) T, subparser ...Parser[T]) *AndParser[T] {
	p := &AndParser[T]{callback: callback, subParser: subparser}
	p.initPool()
	return p
}

func (p *AndParser[T]) initPool() {
	p.buf = sync.Pool{New: func() any {
		buffer := make([]T, 0, len(p.subParser))
		return &buffer
	}}
}

// Set updates the sub parsers. This can be used to construct recursive parsers.
func (p *AndParser[T]) Set(embedded ...Parser[T]) {
	p.subParser = embedded
	p.initPool()
}

// Match matches all given parsers sequentially.
func (p *AndParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	buffer := p.buf.Get().(*[]T)
	nodes := (*buffer)[:0]
	defer func() {
		clear(nodes)
		*buffer = nodes[:0]
		p.buf.Put(buffer)
	}()
	start := s.position
	startPosition := s.position
	for _, c := range p.subParser {
		node, ok := s.applyRule(c)
		if !ok {
			s.setPosition(startPosition)
			return Node[T]{}, false
		}
		nodes = append(nodes, node.Payload)
	}

	return Node[T]{Payload: p.callback(s.input[start:s.position], nodes...)}, true
}
