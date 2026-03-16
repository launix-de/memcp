/*
	(c) 2019 Launix, Inh. Carl-Philip Hänsch
	Author: Tim Kluge

	Dual licensed with custom aggreements or GPLv3
*/

package packrat

import "strconv"
import "testing"

func TestManySeparator(t *testing.T) {
	input := "Hello, Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(4, "Hello", false, true)
	helloAndWorldParser := NewManyParser(func (s string, a ...int) int {
		r := 0
		for _, i := range a {
			r += i
		}
		return r
	}, helloParser, NewAtomParser(0, ",", false, true))

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 8 {
			t.Error("Many combinator doesn't produce correct payload")
		}
	}

	irregularScanner := NewScanner[int]("Hello, Hello, Hello, ", SkipWhitespaceRegex)
	irregularParser := NewManyParser(func (s string, a ...int) int {
		var r int
		for _, i := range a {
			r += i
		}
		return r
	}, helloParser, NewAtomParser(0, ",", false, true))

	_, ierr := Parse(irregularParser, irregularScanner)
	if ierr == nil {
		t.Error("Many combinator matches irregular input")
	}
}

func TestManySeparatorRegex(t *testing.T) {
	input := "   23, 45"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewRegexParser(func (s string) int {
		i, _ := strconv.ParseInt(s, 10, 32)
		return int(i)
	}, `\d+`, false, true)
	helloAndWorldParser := NewManyParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, NewAtomParser(0, ",", false, true))

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 68 {
			t.Error("Many combinator doesn't produce correct payload")
		}
	}

	irregularScanner := NewScanner[int]("Hello, Hello, Hello, ", SkipWhitespaceRegex)
	irregularParser := NewManyParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, NewAtomParser(0, ",", false, true))

	_, ierr := Parse(irregularParser, irregularScanner)
	if ierr == nil {
		t.Error("Many combinator matches irregular input")
	}
}

func TestMany(t *testing.T) {
	input := "Hello Hello Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(5, "Hello", false, true)
	helloAndWorldParser := NewManyParser(func (s string, a ...int) int {
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
		if n.Payload != 15 {
			t.Error("Many combinator doesn't produce correct payload")
		}
	}

	irregularScanner := NewScanner[int]("Sonne", SkipWhitespaceRegex)
	irregularParser := NewManyParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, nil)

	_, ierr := ParsePartial(irregularParser, irregularScanner)
	if ierr == nil {
		t.Error("Many combinator matches irregular input")
	}

	irregularScanner2 := NewScanner[int]("", SkipWhitespaceRegex)
	irregularParser2 := NewManyParser(func (s string, a ...int) int {
		r := 0
		for _, v := range a {
			r += v
		}
		return r
	}, helloParser, nil)

	_, ierr = Parse(irregularParser2, irregularScanner2)
	if ierr == nil {
		t.Error("Many combinator matches irregular input")
	}
}
