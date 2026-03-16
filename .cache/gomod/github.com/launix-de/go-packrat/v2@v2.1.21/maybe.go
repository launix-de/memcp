/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type MaybeParser[T any] struct {
	valueFalse T
	subParser Parser[T]
}

func NewMaybeParser[T any](valueFalse T, subparser Parser[T]) *MaybeParser[T] {
	return &MaybeParser[T]{valueFalse: valueFalse, subParser: subparser}
}

func (p *MaybeParser[T]) Set(embedded Parser[T]) {
	p.subParser = embedded
}

// Match matches the embedded parser or the empty string.
func (p *MaybeParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	startPosition := s.position
	node, ok := s.applyRule(p.subParser)

	if !ok {
		s.setPosition(startPosition)
		return Node[T]{Payload: p.valueFalse}, true
	}

	return Node[T]{Payload: node.Payload}, true
}
