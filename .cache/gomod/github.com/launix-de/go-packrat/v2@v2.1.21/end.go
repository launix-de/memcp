/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type EndParser[T any] struct {
	value T
	skipWs bool
}

func NewEndParser[T any](value T, skipWs bool) *EndParser[T] {
	return &EndParser[T]{value: value, skipWs: skipWs}
}

// Match accepts only the end of the scanner's input and will not match any input.
func (p *EndParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	startPosition := s.position
	if p.skipWs {
		s.Skip()
	}

	if len(s.remainingInput) == 0 {
		return Node[T]{Payload: p.value}, true
	}

	s.setPosition(startPosition)
	return Node[T]{}, false
}
