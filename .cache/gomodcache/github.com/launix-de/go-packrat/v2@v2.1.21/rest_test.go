package packrat

import "testing"

func TestRestParser(t *testing.T) {
	input := "Hello"
	scanner := NewScanner[string](input, SkipWhitespaceRegex)

	helloParser := NewAtomParser(":", "He", false, false)
	restParser := NewRestParser[string](func(s string) string { return s; })
	helloAndWorldParser := NewAndParser(func (x string, a ...string) string {return a[0] + a[1]}, helloParser, restParser)

	n, err := Parse(helloAndWorldParser, scanner)
	if err != nil {
		t.Error(err)
	}

	if n.Payload != ":llo" {
		t.Error("Rest combinator doesn't produce correct payload")
	}
}

