package packrat

import "testing"

func TestEndParser(t *testing.T) {
	input := "Hello"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	endParser := NewEndParser[int](4, false)
	helloParser := NewAtomParser[int](5, "Hello", false, true)
	helloAndWorldParser := NewAndParser(func (x string, a ...int) int {return a[0] + a[1]}, helloParser, endParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	}

	if n.Payload != 9 {
		t.Error("End Test combinator doesn't produce correct payload")
	}
}
