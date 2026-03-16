/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "testing"

func TestKleene(t *testing.T) {
	input := "Hello Hello Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(1, "Hello", false, true)
	helloAndWorldParser := NewKleeneParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, nil)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 3 {
			t.Error("Kleene combinator doesn't produce 3 children")
		}
	}

	irregularInput := "Sonne"
	irregularScanner := NewScanner[int](irregularInput, SkipWhitespaceRegex)
	irregularParser := NewKleeneParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, nil)

	in, ierr := ParsePartial(irregularParser, irregularScanner)
	if ierr != nil {
		t.Error("Kleene combinator doesn't match irregular input")
	}
	if in.Payload != 0 {
		t.Error("Kleene combinator doesn't produce zero children for irregular input")
	}
}
func TestKleeneSeparator(t *testing.T) {
	input := "  Hello, Hello, Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(2, "Hello", false, true)
	sepParser := NewAtomParser(0, ",", false, true)
	helloAndWorldParser := NewKleeneParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, sepParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 6 {
			t.Error("Kleene combinator doesn't produce 3 children")
		}
	}

	irregularInput := "Sonne"
	irregularScanner := NewScanner[int](irregularInput, SkipWhitespaceRegex)
	irregularParser := NewKleeneParser(func (s string, a ...int) int {
		r := 9
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, nil)

	in, ierr := ParsePartial(irregularParser, irregularScanner)
	if ierr != nil {
		t.Error("Kleene combinator doesn't match irregular input")
	}
	if in.Payload != 9 {
		t.Error("Kleene combinator doesn't produce zero children for irregular input")
	}
}
