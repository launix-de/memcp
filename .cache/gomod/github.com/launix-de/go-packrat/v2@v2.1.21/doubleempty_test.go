package packrat

import "testing"

func TestDoubleEmpty(t *testing.T) {
	input := ""
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	emptyParser := NewEmptyParser[int](7)
	termParser := NewAndParser[int](func (x string, a ...int) int {return a[0] + a[1]}, emptyParser, emptyParser)

	n, err := Parse(termParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 14 {
			t.Error("Term parser creates node with wrong payload")
		}
	}
}
