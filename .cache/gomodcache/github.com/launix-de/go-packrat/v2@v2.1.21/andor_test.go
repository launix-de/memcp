/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"


func TestAndInsensitive(t *testing.T) {
	input := "HELLO world"
	scanner := NewScanner[string](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser[string]("a", "Hello", true, true)
	worldParser := NewAtomParser[string]("b", "World", true, true)
	helloAndWorldParser := NewAndParser[string](func (match string, a ...string) string {return a[0] + a[1]}, helloParser, worldParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != "ab" {
			t.Error("And combinator creates wrong result")
		}
	}
}


func TestAnd(t *testing.T) {
	input := "Hello World"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(1, "Hello", false, true)
	worldParser := NewAtomParser(2, "World", false, true)
	helloAndWorldParser := NewAndParser(func (x string, a ...int) int {return a[0] + a[1]}, helloParser, worldParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 3 {
			t.Error("And combinator doesn't match payload")
		}
	}

	irregularInput := "Hello"
	irregularScanner := NewScanner[int](irregularInput, SkipWhitespaceRegex)
	irregularParser := NewAndParser[int](func(match string, a ...int) int {return 13}, helloParser, worldParser)

	_, ierr := Parse(irregularParser, irregularScanner)
	if ierr == nil {
		t.Error("And combinator matches irregular input")
	}
}

func TestOr(t *testing.T) {
	input := "World"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(1, "Hello", false, true)
	worldParser := NewAtomParser(2, "World", false, true)
	helloAndWorldParser := NewOrParser(helloParser, worldParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 2 {
			t.Error("Or combinator doesn't match payload")
		}
	}

	irregularInput := "Sonne"
	irregularScanner := NewScanner[int](irregularInput, SkipWhitespaceRegex)
	irregularParser := NewAndParser(func (x string, a ...int) int {return a[0] + a[1]}, helloParser, worldParser)

	_, ierr := Parse(irregularParser, irregularScanner)
	if ierr == nil {
		t.Error("Or combinator matches irregular input")
	}
}
