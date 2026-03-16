/*
	(c) 2019, 2023 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"

func TestComments(t *testing.T) {
	input := "HELLO /* this is a comment */ world"
	scanner := NewScanner[int](input, SkipWhitespaceAndCommentsRegex)

	helloParser := NewAtomParser[int](1, "Hello", true, true)
	worldParser := NewAtomParser[int](2, "World", true, true)
	helloAndWorldParser := NewAndParser(func(match string, a ...int) int {return 13}, helloParser, worldParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 13 {
			t.Error("And combinator creates wrong payload")
		}
	}
}

