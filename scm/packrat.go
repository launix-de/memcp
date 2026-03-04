/*
Copyright (C) 2023-2024  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import "fmt"
import "regexp"
import packrat "github.com/launix-de/go-packrat/v2"

type parserResult struct {
	value Scmer
	env   map[Symbol]Scmer
}

type ScmParser struct {
	Root      packrat.Parser[*parserResult]
	Syntax    Scmer
	Generator Scmer
	Outer     *Env
	Skipper   *regexp.Regexp
}

type ScmParserVariable struct {
	Parser   packrat.Parser[*parserResult]
	Variable Symbol
}

type UndefinedParser struct {
	Parser packrat.Parser[*parserResult]
	En     *Env
	Sym    Symbol
}

// ScmCaptureParser wraps a parser to capture the matched text
type ScmCaptureParser struct {
	Parser packrat.Parser[*parserResult]
}

func (b *ScmCaptureParser) Match(s *packrat.Scanner[*parserResult]) (packrat.Node[*parserResult], bool) {
	s.Skip() // skip whitespace first to get clean capture start
	startPos := s.GetPosition()
	m, ok := b.Parser.Match(s)
	if !ok {
		return m, false
	}
	endPos := s.GetPosition()
	matchedText := s.Substring(startPos, endPos)
	// Return a list: (matched_text parsed_result)
	result := []Scmer{NewString(matchedText), m.Payload.value}
	env := m.Payload.env
	return packrat.Node[*parserResult]{Payload: &parserResult{value: NewSlice(result), env: env}}, true
}

func parserSymbolEquals(v Scmer, name string) bool {
	return v.IsSymbol() && v.String() == name
}

func scmerToParser(v Scmer) packrat.Parser[*parserResult] {
	switch v.GetTag() {
	case tagAny:
		if parser, ok := v.Any().(packrat.Parser[*parserResult]); ok {
			return parser
		}
	case tagParser:
		return v.Parser()
	}
	panic("value is not a parser: " + v.String())
}

// allows self recursion on parsers
func (b *UndefinedParser) Match(s *packrat.Scanner[*parserResult]) (packrat.Node[*parserResult], bool) {
	if b.Parser == nil {
		en2 := b.En.FindRead(b.Sym)
		val, ok := en2.Vars[b.Sym]
		if !ok {
			panic("error parsing parser: variable does not contain a valid parser: " + string(b.Sym))
		}
		b.Parser = scmerToParser(val)
	}
	return b.Parser.Match(s)
}

func (b *ScmParser) String() string {
	return "(parser ...)" // fallback generator
}

func (b *ScmParser) Match(s *packrat.Scanner[*parserResult]) (packrat.Node[*parserResult], bool) {
	m, ok := b.Root.Match(s)
	if !ok {
		return m, false
	}
	if b.Generator.IsNil() {
		return packrat.Node[*parserResult]{Payload: &parserResult{value: m.Payload.value, env: nil}}, true
	} else {
		var en2 Env
		// evaluate parser
		en2.Vars = m.Payload.env // take variable assignments
		en2.Outer = b.Outer
		en2.Nodefine = true
		return packrat.Node[*parserResult]{Payload: &parserResult{value: Eval(b.Generator, &en2), env: nil}}, true
	}
}

// TODO: create two variants of mergeParserResults, one where the result is not needed and thus nil can be returned as payload
func mergeParserResults(s string, r ...*parserResult) *parserResult {
	var env map[Symbol]Scmer
	arr := make([]Scmer, 0, len(r))
	for _, e := range r {
		if e == nil {
			continue
		}
		v := e.value
		if v.GetTag() == tagAny {
			if inner, ok := v.Any().(*parserResult); ok {
				v = inner.value
			}
		}
		arr = append(arr, v)
		if e.env != nil {
			if env == nil {
				env = make(map[Symbol]Scmer)
			}
			for k, v := range e.env {
				env[k] = v
			}
		}
	}
	return &parserResult{value: NewSlice(arr), env: env}
}
func mergeParserResultsNil(s string, r ...*parserResult) *parserResult {
	var env map[Symbol]Scmer
	for _, e := range r {
		if e != nil && e.env != nil {
			if env == nil {
				env = make(map[Symbol]Scmer)
			}
			for k, v := range e.env {
				env[k] = v
			}
		}
	}
	return &parserResult{value: NewNil(), env: env}
}

func (b *ScmParser) Execute(str string, en *Env) Scmer {
	var skipper *regexp.Regexp = b.Skipper
	if skipper == nil {
		skipper = packrat.SkipWhitespaceAndCommentsRegex // also skip C-style comments as whitespaces
	}
	scanner := packrat.NewScanner[*parserResult](str, skipper)
	node, err := packrat.Parse(b, scanner)
	if err != nil {
		panic(err)
	}
	return node.Payload.value
}

func (b *ScmParserVariable) Match(s *packrat.Scanner[*parserResult]) (packrat.Node[*parserResult], bool) {
	m, ok := b.Parser.Match(s)
	if !ok {
		return m, ok
	}
	env := m.Payload.env
	if env == nil {
		env = make(map[Symbol]Scmer)
	}
	env[b.Variable] = m.Payload.value
	return packrat.Node[*parserResult]{Payload: &parserResult{value: m.Payload.value, env: env}}, true
}

func parseSyntax(syntax Scmer, en *Env, ome *optimizerMetainfo, ignoreResult bool) packrat.Parser[*parserResult] {
	merger := mergeParserResults
	if ignoreResult {
		merger = mergeParserResultsNil
	}
	switch syntax.GetTag() {
	case tagSourceInfo:
		return parseSyntax(syntax.SourceInfo().value, en, ome, ignoreResult)
	case tagFastDict:
		fd := syntax.FastDict()
		if fd == nil {
			syntax = NewSlice([]Scmer{})
		} else {
			syntax = NewSlice(fd.Pairs)
		}
	case tagParser:
		return syntax.Parser()
	case tagAny:
		if p, ok := syntax.Any().(packrat.Parser[*parserResult]); ok {
			return p
		}
	case tagString:
		return packrat.NewAtomParser(&parserResult{value: syntax, env: nil}, syntax.String(), false, true)
	case tagSymbol:
		sym := syntax.Symbol()
		switch sym {
		case "$":
			return packrat.NewEndParser(&parserResult{value: NewNil(), env: nil}, true)
		case "empty":
			return packrat.NewEmptyParser(&parserResult{value: NewNil(), env: nil})
		case "rest":
			return packrat.NewRestParser(func(s string) *parserResult { return &parserResult{value: NewString(s), env: nil} })
		}
		if ome != nil {
			return nil
		}
		en2 := en.FindRead(sym)
		val, ok := en2.Vars[sym]
		if !ok {
			return &UndefinedParser{En: en, Sym: sym}
		}
		return scmerToParser(val)
	case tagNthLocalVar:
		if ome != nil {
			return nil
		}
		if parserScmer := en.VarsNumbered[syntax.NthLocalVar()]; !parserScmer.IsNil() {
			return scmerToParser(parserScmer)
		}
		panic("error invalid parser: " + syntax.String())
	case tagSlice:
		list := syntax.Slice()
		if len(list) == 0 {
			panic("invalid parser ()")
		}
		if list[0].IsSymbol() {
			switch list[0].Symbol() {
			case "parser":
				var resulter Scmer = NewNil()
				if len(list) > 2 {
					Validate(list[2], "any")
					resulter = list[2]
				}
				var skipper Scmer = NewNil()
				if len(list) > 3 {
					Validate(list[3], "string")
					skipper = list[3]
				}
				if ome != nil {
					return nil
				}
				return NewParser(list[1], resulter, skipper, en, ignoreResult)
			case "atom":
				caseInsensitive := false
				if len(list) > 2 {
					caseInsensitive = list[2].Bool()
				}
				skipws := true
				if len(list) > 3 {
					skipws = list[3].Bool()
				}
				value := list[1]
				if len(list) > 4 {
					value = list[4]
				}
				return packrat.NewAtomParser(&parserResult{value: value, env: nil}, list[1].String(), caseInsensitive, skipws)
			case "empty":
				return packrat.NewEmptyParser(&parserResult{value: NewNil(), env: nil})
			case "regex":
				caseInsensitive := false
				if len(list) > 2 {
					caseInsensitive = list[2].Bool()
				}
				skipws := true
				if len(list) > 3 {
					skipws = list[3].Bool()
				}
				pattern := list[1].String()
				return packrat.NewRegexParser(func(s string) *parserResult { return &parserResult{value: NewString(s), env: nil} }, pattern, caseInsensitive, skipws)
			case "list":
				sub := make([]packrat.Parser[*parserResult], len(list)-1)
				for i := 1; i < len(list); i++ {
					sub[i-1] = parseSyntax(list[i], en, ome, ignoreResult)
					if sub[i-1] == nil {
						return nil
					}
				}
				return packrat.NewAndParser(merger, sub...)
			case "or":
				sub := make([]packrat.Parser[*parserResult], len(list)-1)
				for i := 1; i < len(list); i++ {
					sub[i-1] = parseSyntax(list[i], en, ome, ignoreResult)
					if sub[i-1] == nil {
						return nil
					}
				}
				return packrat.NewOrParser(sub...)
			case "not":
				sub := make([]packrat.Parser[*parserResult], len(list)-1)
				for i := 1; i < len(list); i++ {
					sub[i-1] = parseSyntax(list[i], en, ome, ignoreResult)
					if sub[i-1] == nil {
						return nil
					}
				}
				return packrat.NewNotParser(sub[0], sub[1:]...)
			case "*":
				sub := parseSyntax(list[1], en, ome, ignoreResult)
				if sub == nil {
					return nil
				}
				var sep packrat.Parser[*parserResult]
				if len(list) > 2 {
					sep = parseSyntax(list[2], en, ome, ignoreResult)
					if sep == nil {
						return nil
					}
				} else {
					sep = packrat.NewEmptyParser(&parserResult{value: NewNil(), env: nil})
				}
				return packrat.NewKleeneParser(merger, sub, sep)
			case "+":
				sub := parseSyntax(list[1], en, ome, ignoreResult)
				if sub == nil {
					return nil
				}
				var sep packrat.Parser[*parserResult]
				if len(list) > 2 {
					sep = parseSyntax(list[2], en, ome, ignoreResult)
					if sep == nil {
						return nil
					}
				} else {
					sep = packrat.NewEmptyParser(&parserResult{value: NewNil(), env: nil})
				}
				return packrat.NewManyParser(merger, sub, sep)
			case "?":
				if len(list) == 2 {
					sub := parseSyntax(list[1], en, ome, ignoreResult)
					if sub == nil {
						return nil
					}
					return packrat.NewMaybeParser(&parserResult{value: NewNil(), env: nil}, sub)
				}
				sub := make([]packrat.Parser[*parserResult], len(list)-1)
				for i := 1; i < len(list); i++ {
					sub[i-1] = parseSyntax(list[i], en, ome, ignoreResult)
					if sub[i-1] == nil {
						return nil
					}
				}
				return packrat.NewMaybeParser(&parserResult{value: NewNil(), env: nil}, packrat.NewAndParser(merger, sub...))
			case "define":
				result := &ScmParserVariable{}
				result.Variable = Symbol(list[1].String())
				result.Parser = parseSyntax(list[2], en, ome, false)
				if result.Parser == nil {
					return nil
				}
				return result
			case "capture":
				// (capture subparser) - returns (matched_text parsed_result)
				sub := parseSyntax(list[1], en, ome, false)
				if sub == nil {
					return nil
				}
				return &ScmCaptureParser{Parser: sub}
			}
		}
		if isList(list[0]) {
			sub := make([]packrat.Parser[*parserResult], len(list)-1)
			for i := 1; i < len(list); i++ {
				sub[i-1] = parseSyntax(list[i], en, ome, ignoreResult)
				if sub[i-1] == nil {
					return nil
				}
			}
			return packrat.NewAndParser(merger, sub...)
		}
	}
	panic("Unknown parser syntax: " + fmt.Sprint(syntax))
}

func NewParser(syntax, generator, whitespace Scmer, en *Env, ignoreResult bool) *ScmParser {
	if !generator.IsNil() {
		ignoreResult = true
	}
	result := new(ScmParser)
	result.Root = parseSyntax(syntax, en, nil, ignoreResult)
	result.Syntax = syntax // for serialization purposes
	result.Generator = generator
	result.Outer = en
	if !whitespace.IsNil() {
		result.Skipper = regexp.MustCompile(whitespace.String())
		// "^(?:/\\*.*?\\*/|[\r\n\t ]+)+"
	}
	return result
}

func init_parser() {
	DeclareTitle("Parsers")
	Declare(&Globalenv, &Declaration{
		"parser", `creates a parser

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
(capture subparser) wraps a parser and returns (matched_text parsed_result)
$ EndParser
empty EmptyParser
symbol -> use other parser defined in env

for further details on packrat parsers, take a look at https://github.com/launix-de/go-packrat
`,
		1, 3,
		[]DeclarationParameter{
			DeclarationParameter{"syntax", "any", "syntax of the grammar (see docs)", nil},
			DeclarationParameter{"generator", "any", "(optional) expressions to evaluate. All captured variables are available in the scope.", nil},
			DeclarationParameter{"skipper", "string", "(optional) string that defines the skip mechanism for whitespaces as regexp", nil},
		}, "func",
		nil,
		false, false, nil,
		nil,
	})
}
