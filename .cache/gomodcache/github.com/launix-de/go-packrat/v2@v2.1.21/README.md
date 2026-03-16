go-packrat
============================

[![Build Status](https://travis-ci.com/launix-de/go-packrat.svg?branch=master)](https://travis-ci.com/launix-de/go-packrat)
[![GoDoc](https://godoc.org/github.com/launix-de/go-packrat?status.png)](https://godoc.org/github.com/launix-de/go-packrat)


This library allows to construct backtracking top down packrat parsers in Go using parser combination. Packrat parsing enables the parsing of PEG Grammars in linear time. Parsers are combinated using the following basic parsers:

- `AtomParser`: Matches only a specified UTF8 string
- `RegexParser`: Matches a regular expression
- `AndParser`: Matches a given list of parsers sequentially
- `OrParser`: Matches if any of a given list of parsers matches
- `KleeneParser`: Matches a parser 0 to `n` times, optionally separated by another parser
- `ManyParser`: Matches a parser 1 to `n` times, optionally separated by another parser
- `MaybeParser`: Matches a parser 0 or 1 times
- `EmptyParser`: Does not read any input and matches in every case
- `EndParser`: Matches only if the scanner has reached the end of the input string 

By default, `Atom` and `Regex` parsers skip (but do not match on) leading whitespace. This can be configured per parser.

If a parser matches, it returns an syntax tree `*Node`. Every node points to the parser that produced it, the matched text, and a list of child nodes. AST callbacks are not provided atm, so a full syntax tree traversal is needed to process the parse results.

To construct recursive parsers, create parser combinators with `nil` as the sub parser. After creating the sub parser that itself uses the parent parser, use the `Set` function on the parent parser to update its children. The [JSON parser](./json_test.go) provides an example for this.

This library is currently used in production, but some rarely used features may be broken. Additional documentation is ToDo.

Example
-----------

A full example in form of a working json parser is provided in [json_test.go](./json_test.go).

```go
import (
    packrat "github.com/launix-de/go-packrat"
)

func main(){
    input := "Hello World"
    scanner := NewScanner[string](input, true)

    helloParser := NewAtomParser("noun", "Hello", true)
    worldParser := NewAtomParser("noun", "World", true)
    helloAndWorldParser := NewAndParser(func (s string, parts ...string) string {
	    return "found " + parts[0] + " + " + parts[1]
    }, helloParser, worldParser)

    result, err := Parse(helloAndWorldParser, scanner)
    // result will contain "found noun + noun"
}
```

Use case
-----------
Using this library, you can dynamically define and parse PEG grammars at runtime. Parsing time is proportional to the input length and grammar complexity. Note that if you do not need to build grammars at runtime, a parser generator like [gocc](https://github.com/goccmack/gocc) will produce a static LR parser that is both faster and uses less memory than `go-packrat`. If you want to use a parser combinator, worst-case exponential runtime is not a problem or low memory consumption is required, consider using [goparsec](https://github.com/prataprc/goparsec). `go-packrat` was written as a replacement for `goparsec` that solves the time complexity issue using a simple memoization cache. 

License
------------
Dual licensed with custom aggreements or GPLv3. See [LICENSE](./LICENSE).
