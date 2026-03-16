/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type ManyParser[T any] struct {
	callback func(string, ...T) T
	subParser, sepParser Parser[T]
	buf []T
	depth int
	NoMemo bool
}

func NewManyParser[T any](callback func(string, ...T) T, subparser Parser[T], sepparser Parser[T]) *ManyParser[T] {
	return &ManyParser[T]{callback: callback, subParser: subparser, sepParser: sepparser, buf: make([]T, 0, 8)}
}

func (p *ManyParser[T]) Set(embedded Parser[T], separator Parser[T]) {
	p.subParser = embedded
	p.sepParser = separator
}

func (p *ManyParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	var nodes []T
	if p.depth == 0 {
		nodes = p.buf[:0]
	} else {
		nodes = make([]T, 0, 8)
	}
	p.depth++
	start := s.position

	i := 0
	lastValidPos := s.position
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
		lastValidPos = s.position
	}
	s.setPosition(lastValidPos)

	// grow buf for next time if outermost call
	if p.depth == 1 && cap(nodes) > cap(p.buf) {
		p.buf = nodes[:0]
	}
	p.depth--

	if len(nodes) >= 1 {
		return Node[T]{Payload: p.callback(s.input[start:s.position], nodes...)}, true
	}

	return Node[T]{}, false
}
