/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

type OrParser[T any] struct {
	subParser     []Parser[T]
	charMap       *[256][]int
	eofCandidates []int
	charMapBuilt  bool
}

func NewOrParser[T any](subparser ...Parser[T]) *OrParser[T] {
	return &OrParser[T]{subParser: subparser}
}

func (p *OrParser[T]) Set(embedded ...Parser[T]) {
	p.subParser = embedded
	p.charMap = nil
	p.eofCandidates = nil
	p.charMapBuilt = false
}

// SetCharMap installs a manual first-byte dispatch table, overriding auto-build.
func (p *OrParser[T]) SetCharMap(cm [256][]int) {
	p.charMap = &cm
	p.charMapBuilt = true
}

// Match tries sub-parsers until one succeeds. On first call, a charMap is
// automatically built from the sub-parsers' first-byte sets. Only the
// sub-parsers whose first byte matches the current input byte are tried.
func (p *OrParser[T]) Match(s *Scanner[T]) (Node[T], bool) {
	if !p.charMapBuilt {
		p.charMap, p.eofCandidates = buildCharMap[T](p.subParser)
		p.charMapBuilt = true
	}

	origPosition := s.position
	s.Skip()

	if s.position >= len(s.input) {
		for _, idx := range p.eofCandidates {
			node, ok := s.applyRule(p.subParser[idx])
			if ok {
				return Node[T]{Payload: node.Payload}, true
			}
			s.setPosition(s.position)
		}
		s.setPosition(origPosition)
		return Node[T]{}, false
	}

	candidates := p.charMap[s.input[s.position]]
	skipPosition := s.position
	for _, idx := range candidates {
		node, ok := s.applyRule(p.subParser[idx])
		if ok {
			return Node[T]{Payload: node.Payload}, true
		}
		s.setPosition(skipPosition)
	}
	s.setPosition(origPosition)
	return Node[T]{}, false
}
