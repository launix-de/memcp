package packrat

import "testing"

func TestAtomCaseSensitiveMatch(t *testing.T) {
	scanner := NewScanner[int]("SELECT", nil)
	p := NewAtomParser[int](1, "SELECT", false, false)
	node, err := Parse(p, scanner)
	if err != nil {
		t.Error("Case-sensitive atom should match exact string")
	} else if node.Payload != 1 {
		t.Error("Wrong payload")
	}
}

func TestAtomCaseSensitiveFail(t *testing.T) {
	scanner := NewScanner[int]("select", nil)
	p := NewAtomParser[int](1, "SELECT", false, false)
	_, err := Parse(p, scanner)
	if err == nil {
		t.Error("Case-sensitive atom should not match different case")
	}
}

func TestAtomCaseInsensitiveMatch(t *testing.T) {
	scanner := NewScanner[int]("sElEcT", nil)
	p := NewAtomParser[int](1, "SELECT", true, false)
	node, err := Parse(p, scanner)
	if err != nil {
		t.Error("Case-insensitive atom should match mixed case input")
	} else if node.Payload != 1 {
		t.Error("Wrong payload")
	}
}

func TestAtomRegexSpecialChars(t *testing.T) {
	for _, str := range []string{"(", ")", "*", "+", ".", "[", "]", "?", "{", "}", "|", "^", "$"} {
		scanner := NewScanner[int](str, nil)
		p := NewAtomParser[int](1, str, false, false)
		node, err := Parse(p, scanner)
		if err != nil {
			t.Errorf("Atom should match regex special char %q literally", str)
		} else if node.Payload != 1 {
			t.Errorf("Wrong payload for %q", str)
		}
	}
}

func TestAtomWordBoundary(t *testing.T) {
	scanner := NewScanner[int]("SELECT", SkipWhitespaceRegex)
	p := NewAtomParser[int](1, "SEL", false, true)
	_, err := Parse(p, scanner)
	if err == nil {
		t.Error("Atom 'SEL' with skipWs should not match inside 'SELECT' due to word boundary")
	}
}

func TestAtomEmptyInput(t *testing.T) {
	scanner := NewScanner[int]("", nil)
	p := NewAtomParser[int](1, "SELECT", false, false)
	_, err := Parse(p, scanner)
	if err == nil {
		t.Error("Atom should fail on empty input")
	}
}

func TestAtomExactEndOfInput(t *testing.T) {
	scanner := NewScanner[int]("OK", nil)
	p := NewAtomParser[int](1, "OK", false, false)
	node, err := Parse(p, scanner)
	if err != nil {
		t.Error("Atom should match at exact end of input")
	} else if node.Payload != 1 {
		t.Error("Wrong payload")
	}
}
