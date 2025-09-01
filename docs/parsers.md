# Parsers

## parser

creates a parser

Scm parsers work this way:
(parser syntax scmerresult) -> func

syntax can be one of:
(parser syntax scmerresult) will execute scmerresult after parsing syntax
(parser syntax scmerresult "skipper") will add a different whitespace skipper regex to the root parser
(define var syntax) valid inside (parser...), stores the result of syntax into var for use in scmerresult
"str" AtomParser
(atom "str" caseinsensitive skipws) AtomParser
(regex "asdf" caseinsensitive skipws) RegexParser
'(a b c) AndParser
(or a b c) OrParser
(* sub separator) KleeneParser
(+ sub separator) ManyParser
(? xyz) MaybeParser (if >1 AndParser)
(not mainparser parser1 parser2 parser3 ...) a parser that matches mainparser but not parser1...
$ EndParser
empty EmptyParser
symbol -> use other parser defined in env

for further details on packrat parsers, take a look at https://github.com/launix-de/go-packrat


**Allowed number of parameters:** 1–3

### Parameters

- **syntax** (`any`): syntax of the grammar (see docs)
- **generator** (`any`): (optional) expressions to evaluate. All captured variables are available in the scope.
- **skipper** (`string`): (optional) string that defines the skip mechanism for whitespaces as regexp

### Returns

`func`

