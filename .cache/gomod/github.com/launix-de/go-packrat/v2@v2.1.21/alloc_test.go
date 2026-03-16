package packrat

import "testing"

// BenchmarkInsertLike simulates the hot path of parsing repetitive value tuples
// similar to INSERT INTO t(a,b) VALUES(1,'x'),(2,'y'),(3,'z'),...
func BenchmarkInsertLike(b *testing.B) {
	identity := func(s string) string { return s }
	cb := func(match string, a ...string) string { return match }

	// parsers
	lparen := NewAtomParser[string]("(", "(", false, false)
	rparen := NewAtomParser[string](")", ")", false, false)
	comma := NewAtomParser[string](",", ",", false, false)
	intP := NewRegexParser(identity, `-?[0-9]+`, false, false)
	strBody := NewRegexParser(identity, `(\\.|[^\\'])*`, false, false)
	quote := NewAtomParser[string]("'", "'", false, false)

	strP := NewAndParser(cb, quote, strBody, quote)
	value := NewOrParser[string](intP, strP)
	valueList := NewManyParser(cb, value, comma)
	tuple := NewAndParser(cb, lparen, valueList, rparen)
	tuples := NewManyParser(cb, tuple, comma)

	// build input: (1,'hello'),(2,'world'),...repeated
	input := ""
	for i := 0; i < 200; i++ {
		if i > 0 {
			input += ","
		}
		input += "(1,'hello')"
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := NewScanner[string](input, nil)
		_, err := ParsePartial(tuples, scanner)
		if err != nil {
			b.Fatal(err)
		}
	}
}
