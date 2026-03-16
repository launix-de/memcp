package packrat

import "testing"

// --- Auto-built charmap tests ---

func TestAutoCharMapAtomChildren(t *testing.T) {
	aParser := NewAtomParser[string]("alpha", "alpha", false, false)
	bParser := NewAtomParser[string]("beta", "beta", false, false)
	gParser := NewAtomParser[string]("gamma", "gamma", false, false)

	or := NewOrParser[string](aParser, bParser, gParser)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"alpha", true, "alpha"},
		{"beta", true, "beta"},
		{"gamma", true, "gamma"},
		{"delta", false, ""},
		{"", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapCaseInsensitive(t *testing.T) {
	selectP := NewAtomParser[string]("SELECT", "SELECT", true, true)
	insertP := NewAtomParser[string]("INSERT", "INSERT", true, true)
	deleteP := NewAtomParser[string]("DELETE", "DELETE", true, true)

	or := NewOrParser[string](selectP, insertP, deleteP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"SELECT", true, "SELECT"},
		{"select", true, "SELECT"},
		{"  SELECT", true, "SELECT"},
		{"INSERT", true, "INSERT"},
		{"DELETE", true, "DELETE"},
		{"  delete", true, "DELETE"},
		{"UPDATE", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, SkipWhitespaceRegex)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapRegexChildren(t *testing.T) {
	identity := func(s string) string { return s }

	intP := NewRegexParser(identity, `-?[0-9]+`, false, false)
	identP := NewRegexParser(identity, `[a-zA-Z_][a-zA-Z0-9_]*`, false, false)

	or := NewOrParser[string](intP, identP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"42", true, "42"},
		{"-7", true, "-7"},
		{"foo", true, "foo"},
		{"_bar", true, "_bar"},
		{"!nope", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapAndChildren(t *testing.T) {
	cb := func(match string, a ...string) string { return match }

	// "abc" followed by "def"
	abcP := NewAtomParser[string]("abc", "abc", false, false)
	defP := NewAtomParser[string]("def", "def", false, false)
	andAbc := NewAndParser(cb, abcP, defP)

	// "xyz" followed by "123"
	xyzP := NewAtomParser[string]("xyz", "xyz", false, false)
	oneP := NewAtomParser[string]("123", "123", false, false)
	andXyz := NewAndParser(cb, xyzP, oneP)

	or := NewOrParser[string](andAbc, andXyz)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"abcdef", true, "abcdef"},
		{"xyz123", true, "xyz123"},
		{"abc123", false, ""},
		{"other", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapNestedOrChildren(t *testing.T) {
	aP := NewAtomParser[string]("a", "a", false, false)
	bP := NewAtomParser[string]("b", "b", false, false)
	cP := NewAtomParser[string]("c", "c", false, false)
	dP := NewAtomParser[string]("d", "d", false, false)

	innerOr := NewOrParser[string](aP, bP)
	outerOr := NewOrParser[string](innerOr, cP, dP)

	tests := []struct {
		input string
		ok    bool
	}{
		{"a", true},
		{"b", true},
		{"c", true},
		{"d", true},
		{"e", false},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		_, err := ParsePartial(outerOr, scanner)
		if tt.ok && err != nil {
			t.Errorf("input %q: expected match, got error", tt.input)
		} else if !tt.ok && err == nil {
			t.Errorf("input %q: expected error, got match", tt.input)
		}
	}
}

func TestAutoCharMapEmptyChild(t *testing.T) {
	// EmptyParser always matches, so it must register all 256 bytes + EOF.
	// An OR with EmptyParser should always succeed, regardless of input.
	aP := NewAtomParser[string]("a", "a", false, false)
	emptyP := NewEmptyParser[string]("empty")

	or := NewOrParser[string](aP, emptyP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"a", true, "a"},      // matches atom
		{"b", true, "empty"},  // atom fails, empty matches
		{"", true, "empty"},   // EOF: empty matches
		{"xyz", true, "empty"},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if !tt.ok {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
			continue
		}
		if err != nil {
			t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
		} else if node.Payload != tt.match {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
		}
	}
}

// EmptyParser always continues with whatever follows — test in an And context
// where Empty is the first child of an OR alternative.
func TestAutoCharMapEmptyInAndContinuesWithSuccessor(t *testing.T) {
	cb := func(match string, a ...string) string { return match }

	// OR( And(empty, "hello"), "world" )
	emptyP := NewEmptyParser[string]("empty")
	helloP := NewAtomParser[string]("hello", "hello", false, false)
	andEH := NewAndParser(cb, emptyP, helloP)

	worldP := NewAtomParser[string]("world", "world", false, false)
	or := NewOrParser[string](andEH, worldP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"hello", true, "hello"},
		{"world", true, "world"},
		{"other", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapKleeneChild(t *testing.T) {
	// KleeneParser can match empty (0 repetitions), so it registers all 256 bytes.
	cb := func(match string, a ...string) string { return match }

	aP := NewAtomParser[string]("a", "a", false, false)
	kleene := NewKleeneParser(cb, aP, nil)
	bP := NewAtomParser[string]("b", "b", false, false)

	or := NewOrParser[string](kleene, bP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"aaa", true, "aaa"},   // kleene matches "aaa"
		{"b", true, ""},        // kleene is first, matches empty (0 repetitions)
		{"x", true, ""},        // kleene matches empty
		{"", true, ""},         // EOF: kleene matches empty
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if err != nil {
			t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
		} else if node.Payload != tt.match {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
		}
	}
}

// Kleene can match 0 items, so in an And sequence, it continues with its successor.
func TestAutoCharMapKleeneInAndContinuesWithSuccessor(t *testing.T) {
	cb := func(match string, a ...string) string { return match }

	// OR( And(Kleene(digit), "end"), "start" )
	identity := func(s string) string { return s }
	digitP := NewRegexParser(identity, `[0-9]+`, false, false)
	kleene := NewKleeneParser(cb, digitP, nil)
	endP := NewAtomParser[string]("end", "end", false, false)
	andKE := NewAndParser(cb, kleene, endP)

	startP := NewAtomParser[string]("start", "start", false, false)
	or := NewOrParser[string](andKE, startP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"123end", true, "123end"},   // kleene matches "123", then "end"
		{"end", true, "end"},         // kleene matches empty, then "end"
		{"start", true, "start"},     // "start" matches
		{"xyz", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapMaybeChild(t *testing.T) {
	// MaybeParser always succeeds (returns valueFalse on failure), registers all 256.
	aP := NewAtomParser[string]("a", "a", false, false)
	maybe := NewMaybeParser[string]("none", aP)
	bP := NewAtomParser[string]("b", "b", false, false)

	or := NewOrParser[string](maybe, bP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"a", true, "a"},
		{"b", true, "none"}, // maybe matches first (empty), returns "none"
		{"x", true, "none"}, // maybe matches empty
		{"", true, "none"},  // EOF: maybe matches
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if err != nil {
			t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
		} else if node.Payload != tt.match {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
		}
	}
}

func TestAutoCharMapManyChild(t *testing.T) {
	// ManyParser requires >= 1 match, so it uses sub-parser's first bytes.
	cb := func(match string, a ...string) string { return match }
	identity := func(s string) string { return s }

	digitP := NewRegexParser(identity, `[0-9]+`, false, false)
	commaP := NewAtomParser[string](",", ",", false, false)
	many := NewManyParser(cb, digitP, commaP)

	alphaP := NewRegexParser(identity, `[a-zA-Z]+`, false, false)
	or := NewOrParser[string](many, alphaP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"123,456", true, "123,456"},
		{"abc", true, "abc"},
		{"!nope", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapEndChild(t *testing.T) {
	// EndParser only matches at EOF. Should be in eofCandidates only.
	aP := NewAtomParser[string]("a", "a", false, false)
	endP := NewEndParser[string]("EOF", false)

	or := NewOrParser[string](aP, endP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"a", true, "a"},
		{"", true, "EOF"},    // EOF: EndParser matches
		{"b", false, ""},     // not 'a' and not EOF
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapNotChild(t *testing.T) {
	// NotParser delegates to mainParser's first bytes.
	identity := func(s string) string { return s }

	identP := NewRegexParser(identity, `[a-zA-Z]+`, false, false)
	reservedP := NewAtomParser[string]("if", "if", false, false)
	notReserved := NewNotParser[string](identP, reservedP)

	numP := NewRegexParser(identity, `[0-9]+`, false, false)
	or := NewOrParser[string](notReserved, numP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"hello", true, "hello"},
		{"42", true, "42"},
		{"if", false, ""},     // "if" matches identP but is excluded by NotParser
		{"!x", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapRestChild(t *testing.T) {
	identity := func(s string) string { return s }

	aP := NewAtomParser[string]("a", "a", false, false)
	restP := NewRestParser(identity)

	or := NewOrParser[string](aP, restP)

	tests := []struct {
		input string
		match string
	}{
		{"a", "a"},
		{"anything", "a"},        // 'a' matches atom first
		{"bbb", "bbb"},           // rest matches
		{"", ""},                  // EOF: rest matches empty
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if err != nil {
			t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
		} else if node.Payload != tt.match {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
		}
	}
}

func TestAutoCharMapLeftRecursion(t *testing.T) {
	// Recursive grammar: expr = expr "+" num | num
	// This tests that charmap building handles cycles.
	cb := func(match string, a ...string) string { return match }
	identity := func(s string) string { return s }

	numP := NewRegexParser(identity, `[0-9]+`, false, false)
	plusP := NewAtomParser[string]("+", "+", false, false)

	expr := NewOrParser[string]()
	sum := NewAndParser(cb, expr, plusP, numP)
	expr.Set(sum, numP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"1", true, "1"},
		{"1+2", true, "1+2"},
		{"1+2+3", true, "1+2+3"},
		{"abc", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := Parse(expr, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapDotRegex(t *testing.T) {
	// "." regex matches any byte → all 256 bytes
	identity := func(s string) string { return s }

	dotP := NewRegexParser(identity, `.`, false, false)
	aP := NewAtomParser[string]("a", "a", false, false)

	or := NewOrParser[string](aP, dotP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"a", true, "a"},
		{"x", true, "x"},
		{"1", true, "1"},
		{"", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapStarRegex(t *testing.T) {
	// [a-z]* can match empty → all 256 bytes + EOF
	identity := func(s string) string { return s }

	starP := NewRegexParser(identity, `[a-z]*`, false, false)
	numP := NewRegexParser(identity, `[0-9]+`, false, false)

	or := NewOrParser[string](starP, numP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"abc", true, "abc"},
		{"123", true, ""},    // starP matches empty first (it's first in OR)
		{"", true, ""},       // EOF: starP matches empty
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapOptionalPrefix(t *testing.T) {
	// -?[0-9]+ has first bytes: '-', '0'-'9'
	identity := func(s string) string { return s }

	intP := NewRegexParser(identity, `-?[0-9]+`, false, false)
	identP := NewRegexParser(identity, `[a-zA-Z]+`, false, false)

	or := NewOrParser[string](intP, identP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{"42", true, "42"},
		{"-7", true, "-7"},
		{"abc", true, "abc"},
		{"!x", false, ""},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

func TestAutoCharMapEscapedStringBody(t *testing.T) {
	// (\\.|[^\\'])* matches string body — can match empty → all 256 + EOF
	identity := func(s string) string { return s }

	strBody := NewRegexParser(identity, `(\\.|[^\\'])*`, false, false)
	numP := NewRegexParser(identity, `[0-9]+`, false, false)

	or := NewOrParser[string](strBody, numP)

	tests := []struct {
		input string
		ok    bool
		match string
	}{
		{`hello`, true, `hello`},
		{`123`, true, `123`},          // strBody matches "123" (not a quote or backslash)
		{``, true, ``},                // EOF: strBody matches empty
		{`hello\'world`, true, `hello\'world`},
	}

	for _, tt := range tests {
		scanner := NewScanner[string](tt.input, nil)
		node, err := ParsePartial(or, scanner)
		if tt.ok {
			if err != nil {
				t.Errorf("input %q: expected match, got error: %s", tt.input, err.Error())
			} else if node.Payload != tt.match {
				t.Errorf("input %q: expected %q, got %q", tt.input, tt.match, node.Payload)
			}
		} else {
			if err == nil {
				t.Errorf("input %q: expected error, got match %q", tt.input, node.Payload)
			}
		}
	}
}

// Test that Set() resets charMap so it auto-rebuilds on next Match.
func TestAutoCharMapRebuildsAfterSet(t *testing.T) {
	aP := NewAtomParser[string]("a", "a", false, false)
	bP := NewAtomParser[string]("b", "b", false, false)
	cP := NewAtomParser[string]("c", "c", false, false)

	or := NewOrParser[string](aP, bP)

	// First parse triggers charmap build
	scanner := NewScanner[string]("a", nil)
	node, err := ParsePartial(or, scanner)
	if err != nil || node.Payload != "a" {
		t.Fatal("expected 'a' to match")
	}

	// 'c' should not match yet
	scanner = NewScanner[string]("c", nil)
	_, err = ParsePartial(or, scanner)
	if err == nil {
		t.Fatal("expected 'c' to fail before Set")
	}

	// Update children, should reset charmap
	or.Set(aP, bP, cP)

	scanner = NewScanner[string]("c", nil)
	node, err = ParsePartial(or, scanner)
	if err != nil {
		t.Fatal("expected 'c' to match after Set")
	}
	if node.Payload != "c" {
		t.Errorf("expected %q, got %q", "c", node.Payload)
	}
}

// --- First-byte extraction unit tests ---

func TestParserFirstBytesAtom(t *testing.T) {
	p := NewAtomParser[string]("SELECT", "SELECT", true, false)
	bytes, eof := parserFirstBytes[string](p, make(map[any]bool))
	if eof {
		t.Error("AtomParser should not match EOF")
	}
	if !bytes['S'] || !bytes['s'] {
		t.Error("case-insensitive 'S'/'s' should be set")
	}
	if bytes['X'] {
		t.Error("'X' should not be set")
	}
}

func TestParserFirstBytesRegexIdentifier(t *testing.T) {
	identity := func(s string) string { return s }
	p := NewRegexParser(identity, `[a-zA-Z_][a-zA-Z0-9_]*`, false, false)
	bytes, eof := parserFirstBytes[string](p, make(map[any]bool))
	if eof {
		t.Error("identifier regex should not match EOF")
	}
	if !bytes['a'] || !bytes['Z'] || !bytes['_'] {
		t.Error("expected a-zA-Z_ in first bytes")
	}
	if bytes['0'] || bytes['9'] {
		t.Error("digits should not be in first bytes (not in first class)")
	}
}

func TestParserFirstBytesRegexOptionalMinus(t *testing.T) {
	identity := func(s string) string { return s }
	p := NewRegexParser(identity, `-?[0-9]+`, false, false)
	bytes, eof := parserFirstBytes[string](p, make(map[any]bool))
	if eof {
		t.Error("should not match EOF")
	}
	if !bytes['-'] {
		t.Error("'-' should be in first bytes")
	}
	if !bytes['0'] || !bytes['5'] || !bytes['9'] {
		t.Error("digits should be in first bytes")
	}
	if bytes['a'] {
		t.Error("'a' should not be in first bytes")
	}
}

func TestParserFirstBytesEmpty(t *testing.T) {
	p := NewEmptyParser[string]("empty")
	bytes, eof := parserFirstBytes[string](p, make(map[any]bool))
	if !eof {
		t.Error("EmptyParser should match EOF")
	}
	// EmptyParser has no active first bytes (it matches empty).
	// buildCharMap adds it to all 256 entries because canMatchEOF=true.
	for i := 0; i < 256; i++ {
		if bytes[i] {
			t.Errorf("EmptyParser should have no active first bytes, byte %d is true", i)
			break
		}
	}
}

func TestParserFirstBytesEnd(t *testing.T) {
	p := NewEndParser[string]("end", false)
	bytes, eof := parserFirstBytes[string](p, make(map[any]bool))
	if !eof {
		t.Error("EndParser should match EOF")
	}
	for i := 0; i < 256; i++ {
		if bytes[i] {
			t.Errorf("EndParser should have no bytes set, byte %d is true", i)
			break
		}
	}
}

func TestParserFirstBytesKleeneInAnd(t *testing.T) {
	// And(Kleene(digit), "end") should have first bytes: 0-9 (from Kleene's sub)
	// AND 'e' (from "end", because Kleene can match empty).
	cb := func(match string, a ...string) string { return match }
	identity := func(s string) string { return s }

	digitP := NewRegexParser(identity, `[0-9]+`, false, false)
	kleene := NewKleeneParser(cb, digitP, nil)
	endP := NewAtomParser[string]("end", "end", false, false)
	and := NewAndParser(cb, kleene, endP)

	bytes, eof := parserFirstBytes[string](and, make(map[any]bool))
	if eof {
		t.Error("And(Kleene, Atom) should not match EOF (Atom is required)")
	}
	// Digits from Kleene's sub-parser
	for _, b := range "0123456789" {
		if !bytes[byte(b)] {
			t.Errorf("expected digit %q in first bytes", string(b))
		}
	}
	// 'e' from "end" successor (because Kleene can match empty)
	if !bytes['e'] {
		t.Error("expected 'e' in first bytes (successor of empty Kleene)")
	}
	// 'a' should NOT be set
	if bytes['a'] {
		t.Error("'a' should not be in first bytes")
	}
}

func TestParserFirstBytesEmptyInAnd(t *testing.T) {
	// And(Empty, "hello") should have first bytes: 'h' (from "hello",
	// because Empty always matches and its successor determines the byte).
	cb := func(match string, a ...string) string { return match }

	emptyP := NewEmptyParser[string]("empty")
	helloP := NewAtomParser[string]("hello", "hello", false, false)
	and := NewAndParser(cb, emptyP, helloP)

	bytes, eof := parserFirstBytes[string](and, make(map[any]bool))
	if eof {
		t.Error("And(Empty, Atom) should not match EOF (Atom is required)")
	}
	if !bytes['h'] {
		t.Error("expected 'h' in first bytes from successor 'hello'")
	}
	// Only 'h' should be set, not all 256
	count := 0
	for i := 0; i < 256; i++ {
		if bytes[i] {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 byte set ('h'), got %d", count)
	}
}

func TestParserFirstBytesKleenePrecise(t *testing.T) {
	// KleeneParser returns sub-parser's first bytes + canMatchEOF=true.
	cb := func(match string, a ...string) string { return match }

	aP := NewAtomParser[string]("a", "a", false, false)
	kleene := NewKleeneParser(cb, aP, nil)

	bytes, eof := parserFirstBytes[string](kleene, make(map[any]bool))
	if !eof {
		t.Error("KleeneParser should report canMatchEOF=true")
	}
	if !bytes['a'] {
		t.Error("KleeneParser should have 'a' from sub-parser")
	}
	// Should NOT have all 256 bytes — only 'a'
	count := 0
	for i := 0; i < 256; i++ {
		if bytes[i] {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 byte set ('a'), got %d", count)
	}
}

func TestParserFirstBytesMaybePrecise(t *testing.T) {
	// MaybeParser returns sub-parser's first bytes + canMatchEOF=true.
	aP := NewAtomParser[string]("a", "a", false, false)
	maybe := NewMaybeParser[string]("none", aP)

	bytes, eof := parserFirstBytes[string](maybe, make(map[any]bool))
	if !eof {
		t.Error("MaybeParser should report canMatchEOF=true")
	}
	if !bytes['a'] {
		t.Error("MaybeParser should have 'a' from sub-parser")
	}
	count := 0
	for i := 0; i < 256; i++ {
		if bytes[i] {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 byte set ('a'), got %d", count)
	}
}
