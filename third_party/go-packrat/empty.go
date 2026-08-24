/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type EmptyParser[T any] struct {
	// Stub field to prevent compiler from optimizing out &EmptyParser{}
	value T
	_hidden bool
}

func NewEmptyParser[T any](value T) *EmptyParser[T] {
	return &EmptyParser[T]{value: value}
}

// Match matches only the given string. If skipWs is set to true, leading whitespace according to the scanner's skip regexp is skipped, but not matched by the parser.
func (p *EmptyParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	return Node[T]{Payload: p.value}, true
}
