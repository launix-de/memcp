/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"

func TestRegex(t *testing.T) {
	input := "-3.4"
	scanner := NewScanner[string](input, SkipWhitespaceRegex)

	numParser := NewRegexParser(func (s string) string {return s}, "-?\\d+\\.\\d+", false, false)

	n, err := Parse(numParser, scanner)
	if err != nil {
		t.Error(err)
	}
	if n.Payload != "-3.4" {
		t.Error("Regex combinator dosen't produce correct payload")
	}

	irregularInput := "3,4"
	irregularScanner := NewScanner[string](irregularInput, SkipWhitespaceRegex)

	_, ierr := Parse(numParser, irregularScanner)
	if ierr == nil {
		t.Error("Regex combinator matches irregular input")
	}
}
