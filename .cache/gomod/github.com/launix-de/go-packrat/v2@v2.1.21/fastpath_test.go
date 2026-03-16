package packrat

import "testing"

// Category A: [class]+
func TestFastPathCharClassPlus(t *testing.T) {
	fp := detectFastPath(`[a-zA-Z0-9_]+`, false)
	if fp == nil {
		t.Fatal("Should detect [a-zA-Z0-9_]+ as fast path")
	}
	if n := fp("hello_123 rest"); n != 9 {
		t.Errorf("Expected 9, got %d", n)
	}
	if n := fp("!nope"); n != -1 {
		t.Errorf("Expected -1 for no match, got %d", n)
	}
	if n := fp(""); n != -1 {
		t.Errorf("Expected -1 for empty input, got %d", n)
	}
}

// Category B: [class]*
func TestFastPathCharClassStar(t *testing.T) {
	fp := detectFastPath(`[a-zA-Z0-9_]*`, false)
	if fp == nil {
		t.Fatal("Should detect [a-zA-Z0-9_]* as fast path")
	}
	if n := fp("hello_123 rest"); n != 9 {
		t.Errorf("Expected 9, got %d", n)
	}
	if n := fp("!nope"); n != 0 {
		t.Errorf("Expected 0 for * with no match, got %d", n)
	}
	if n := fp(""); n != 0 {
		t.Errorf("Expected 0 for empty input, got %d", n)
	}
}

// Category C: [^X]+ / [^X]*
func TestFastPathNegatedClassPlus(t *testing.T) {
	fp := detectFastPath(`[^"]+`, false)
	if fp == nil {
		t.Fatal("Should detect [^\"]+ as fast path")
	}
	if n := fp(`hello"`); n != 5 {
		t.Errorf("Expected 5, got %d", n)
	}
	if n := fp(`"`); n != -1 {
		t.Errorf("Expected -1 for immediate excluded char, got %d", n)
	}
	if n := fp(""); n != -1 {
		t.Errorf("Expected -1 for empty input with +, got %d", n)
	}
}

func TestFastPathNegatedClassStar(t *testing.T) {
	fp := detectFastPath(`[^>]*`, false)
	if fp == nil {
		t.Fatal("Should detect [^>]* as fast path")
	}
	if n := fp("hello>"); n != 5 {
		t.Errorf("Expected 5, got %d", n)
	}
	if n := fp(">"); n != 0 {
		t.Errorf("Expected 0 for * with immediate excluded, got %d", n)
	}
	if n := fp(""); n != 0 {
		t.Errorf("Expected 0 for empty input with *, got %d", n)
	}
}

// Category D: [class1][class2]*
func TestFastPathIdentifier(t *testing.T) {
	fp := detectFastPath(`[a-zA-Z_][a-zA-Z0-9_]*`, false)
	if fp == nil {
		t.Fatal("Should detect identifier pattern as fast path")
	}
	if n := fp("foo_123 rest"); n != 7 {
		t.Errorf("Expected 7, got %d", n)
	}
	if n := fp("123"); n != -1 {
		t.Errorf("Expected -1 for digit-leading, got %d", n)
	}
	if n := fp(""); n != -1 {
		t.Errorf("Expected -1 for empty, got %d", n)
	}
	if n := fp("x"); n != 1 {
		t.Errorf("Expected 1 for single char, got %d", n)
	}
}

// Category E: numeric patterns
func TestFastPathSignedInt(t *testing.T) {
	fp := detectFastPath(`-?[0-9]+`, false)
	if fp == nil {
		t.Fatal("Should detect signed int pattern")
	}
	if n := fp("-42abc"); n != 3 {
		t.Errorf("Expected 3, got %d", n)
	}
	if n := fp("42"); n != 2 {
		t.Errorf("Expected 2, got %d", n)
	}
	if n := fp("-"); n != -1 {
		t.Errorf("Expected -1 for bare minus, got %d", n)
	}
	if n := fp("abc"); n != -1 {
		t.Errorf("Expected -1 for no digits, got %d", n)
	}
	if n := fp(""); n != -1 {
		t.Errorf("Expected -1 for empty, got %d", n)
	}
}

func TestFastPathSignedFloat(t *testing.T) {
	fp := detectFastPath(`-?[0-9]+\.?[0-9]*(?:e-?[0-9]+)?`, false)
	if fp == nil {
		t.Fatal("Should detect signed float pattern")
	}
	if n := fp("3.14"); n != 4 {
		t.Errorf("Expected 4, got %d", n)
	}
	if n := fp("-1.5e10"); n != 7 {
		t.Errorf("Expected 7, got %d", n)
	}
	if n := fp("42e-3x"); n != 5 {
		t.Errorf("Expected 5, got %d", n)
	}
	if n := fp("1e"); n != 1 {
		t.Errorf("Expected 1 (e without digits not consumed), got %d", n)
	}
	if n := fp("-"); n != -1 {
		t.Errorf("Expected -1 for bare minus, got %d", n)
	}
}

// Category F: escaped string body
func TestFastPathEscapedStringBody(t *testing.T) {
	fp := detectFastPath(`(\\.|[^\\'])*`, false)
	if fp == nil {
		t.Fatal("Should detect escaped string body pattern")
	}
	// hello\' = 7 bytes (escape pair) + world = 5 bytes = 12, then stops at unescaped '
	if n := fp(`hello\'world'`); n != 12 {
		t.Errorf("Expected 12, got %d", n)
	}
	if n := fp(""); n != 0 {
		t.Errorf("Expected 0 for empty (star quantifier), got %d", n)
	}
	if n := fp("'"); n != 0 {
		t.Errorf("Expected 0 for immediate quote, got %d", n)
	}
	// Trailing backslash: stop before it
	if n := fp(`hello\`); n != 5 {
		t.Errorf("Expected 5 for trailing backslash, got %d", n)
	}
}

func TestFastPathEscapedStringBodyDouble(t *testing.T) {
	fp := detectFastPath(`(\\.|[^\\"])*`, false)
	if fp == nil {
		t.Fatal("Should detect double-quote escaped string body")
	}
	if n := fp(`hello\"world"`); n != 12 {
		t.Errorf("Expected 12, got %d", n)
	}
}

// Category G: literal-prefix + class
func TestFastPathLiteralPrefixClass(t *testing.T) {
	fp := detectFastPath(`\?[a-zA-Z0-9_]+`, false)
	if fp == nil {
		t.Fatal("Should detect literal prefix class pattern")
	}
	if n := fp("?var1 x"); n != 5 {
		t.Errorf("Expected 5, got %d", n)
	}
	if n := fp("nope"); n != -1 {
		t.Errorf("Expected -1 for no prefix, got %d", n)
	}
	if n := fp("?"); n != -1 {
		t.Errorf("Expected -1 for ? with no class chars, got %d", n)
	}
	if n := fp(""); n != -1 {
		t.Errorf("Expected -1 for empty, got %d", n)
	}
}

// Integration: verify RegexParser produces correct payload with fast path
func TestFastPathRegexParserIntegration(t *testing.T) {
	identity := func(s string) string { return s }

	tests := []struct {
		name    string
		pattern string
		input   string
		want    string
	}{
		{"identifier", `[a-zA-Z_][a-zA-Z0-9_]*`, "hello_world rest", "hello_world"},
		{"digits+", `[0-9]+`, "42abc", "42"},
		{"signed int", `-?[0-9]+`, "-7xyz", "-7"},
		{"negated class", `[^"]+`, `hello"`, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewRegexParser(identity, tt.pattern, false, false)
			scanner := NewScanner[string](tt.input, nil)
			node, err := ParsePartial(p, scanner)
			if err != nil {
				t.Fatalf("Parse failed for pattern %s on input %q: %s", tt.pattern, tt.input, err)
			}
			if node.Payload != tt.want {
				t.Errorf("Expected payload %q, got %q", tt.want, node.Payload)
			}
		})
	}
}

// Verify that unrecognized patterns still fall back to regex
func TestFastPathFallbackToRegex(t *testing.T) {
	identity := func(s string) string { return s }
	// A complex pattern that won't be detected as a fast path
	p := NewRegexParser(identity, `[a-z]+(?:_[a-z]+)*`, false, false)
	scanner := NewScanner[string]("hello_world rest", nil)
	node, err := ParsePartial(p, scanner)
	if err != nil {
		t.Fatal("Regex fallback should still work")
	}
	if node.Payload != "hello_world" {
		t.Errorf("Expected 'hello_world', got %q", node.Payload)
	}
}
