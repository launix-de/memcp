/*
	(c) 2024 Launix, Inh. Carl-Philip Hänsch
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type RestParser[T any] struct {
	converter func (string) T
}

func NewRestParser[T any](converter func (string) T) *RestParser[T] {
	return &RestParser[T]{converter: converter}
}

func (p *RestParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	// just slice away the rest
	v := s.remainingInput
	s.setPosition(len(s.input))
	return Node[T]{Payload: p.converter(v)}, true
}

