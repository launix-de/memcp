package packrat

import "testing"

func TestEmptyParser(t *testing.T) {
	input := "Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	emptyParser := NewEmptyParser[int](7)
	helloParser := NewAtomParser(1, "Hello", false, true)
	helloAndWorldParser := NewAndParser(func (x string, a ...int) int {return a[0] + a[1]}, emptyParser, helloParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	}

	if n.Payload != 8 {
		t.Error("Empty Test combinator doesn't produce correct payload")
	}
}
