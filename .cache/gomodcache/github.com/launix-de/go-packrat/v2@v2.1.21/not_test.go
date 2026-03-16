/*
	(c) 2024 Launix, Inh. Carl-Philip Hänsch
	Author: Carl-Philip Hänsch

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"


func TestNot(t *testing.T) {
	input := "moms"
	scanner := NewScanner[string](input, SkipWhitespaceRegex)

	anyParser := NewRegexParser(func (s string) string {return s}, ".*", false, false)
	helloParser := NewAtomParser[string]("a", "Hello", true, true)
	worldParser := NewAtomParser[string]("b", "World", true, true)
	helloAndWorldParser := NewNotParser[string](anyParser, helloParser, worldParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != "moms" {
			t.Error("Not parser creates wrong result")
		}
	}

	input = "hello"
	scanner = NewScanner[string](input, SkipWhitespaceRegex)

	n, err = Parse(helloAndWorldParser, scanner)
	if err == nil {
		t.Error("Not parser parses a forbidden word")
	} else {
		// ok
	}
}


