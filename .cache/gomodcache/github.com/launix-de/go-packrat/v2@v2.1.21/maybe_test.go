/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/
package packrat

import "testing"

func TestMaybe(t *testing.T) {
	input := "Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(17, "Hello", false, true)
	helloAndWorldParser := NewMaybeParser(13, helloParser)

	n, err := ParsePartial(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 17 {
			t.Error("Maybe combinator doesn't produce correct payload")
		}
	}

	irregularInput := "Sonne"
	irregularScanner := NewScanner[int](irregularInput, SkipWhitespaceRegex)
	irregularParser := NewMaybeParser(13, helloParser)

	in, ierr := ParsePartial(irregularParser, irregularScanner)
	if ierr != nil {
		t.Error("Maybe combinator doesn't match irregular input")
	}
	if in.Payload != 13 {
		t.Error("Maybe combinator doesn't produce correct payload")
	}
}
