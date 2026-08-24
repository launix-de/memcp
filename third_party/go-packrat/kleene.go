/*
	(c) 2019-2026 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "sync"

type KleeneParser[T any] struct {
	callback             func(string, ...T) T
	subParser, sepParser Parser[T]
	buf                  sync.Pool
	NoMemo               bool
}

func NewKleeneParser[T any](callback func(string, ...T) T, subparser Parser[T], sepparser Parser[T]) *KleeneParser[T] {
	p := &KleeneParser[T]{callback: callback, subParser: subparser, sepParser: sepparser}
	p.buf.New = func() any {
		buffer := make([]T, 0, 8)
		return &buffer
	}
	return p
}

func (p *KleeneParser[T]) Set(embedded Parser[T], separator Parser[T]) {
	p.subParser = embedded
	p.sepParser = separator
}

// Match matches the embedded parser or the empty string.
func (p *KleeneParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	buffer := p.buf.Get().(*[]T)
	nodes := (*buffer)[:0]
	defer func() {
		clear(nodes)
		*buffer = nodes[:0]
		p.buf.Put(buffer)
	}()
	start := s.position

	i := 0
	lastValidPosition := s.position
	applyFn := s.applyRule
	if p.NoMemo {
		applyFn = func(rule Parser[T]) (Node[T], bool) { return rule.Match(s) }
	}
	for {
		if i > 0 && p.sepParser != nil {
			_, ok := applyFn(p.sepParser)
			if !ok {
				break
			}
		}
		i++

		node, ok := applyFn(p.subParser)
		if !ok {
			break
		}

		nodes = append(nodes, node.Payload)
		lastValidPosition = s.position
	}
	s.setPosition(lastValidPosition)

	if len(nodes) == 0 {
		return Node[T]{Payload: p.callback("")}, true
	}
	return Node[T]{p.callback(s.input[start:s.position], nodes...)}, true
}
