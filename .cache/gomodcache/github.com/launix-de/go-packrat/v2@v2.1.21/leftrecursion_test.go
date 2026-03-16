package packrat

import "fmt"
import "testing"
import "strconv"

func TestLeftRecursion(t *testing.T) {
	input := "5-1-4-3"
	scanner := NewScanner[int](input, SkipWhitespaceRegex)

	emptyParser := NewEmptyParser(0)
	emptyParser1 := NewAndParser(func (s string, a ...int) int {
		return a[0]
	}, emptyParser)

	numParser := NewRegexParser(func (s string) int {
		i, _ := strconv.ParseInt(s, 10, 32)
		return int(i)
	}, `\d+`, false, true)
	numCombo1 := NewAndParser(func (s string, a ...int) int {
		return a[0] + a[1] + a[2]
	}, emptyParser1, emptyParser1, numParser)
	minusParser := NewAtomParser(0, `-`, false, true)

	termParser := NewAndParser(func (s string, a ...int) int {
		return a[0] + a[2]
	})
	exprParser := NewOrParser(termParser, numCombo1)
	termParser.Set(exprParser, minusParser, numCombo1)

	n, err := Parse(exprParser, scanner)
	if err != nil {
		t.Error(err)
	} else {
		if n.Payload != 13 {
			t.Error("Term parser creates node with wrong payload: " + fmt.Sprint(n.Payload))
		}
	}
}
