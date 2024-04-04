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
import packrat "github.com/cphaensch/go-packrat"

type parserResult struct {
	value Scmer
	env map[Symbol]Scmer // asdf
}

type ScmParser struct {
	Root packrat.Parser[parserResult] // wrapper for parser
	Syntax Scmer // keep syntax for deserializer
	Generator Scmer
	Outer *Env
	Skipper *regexp.Regexp
}

type ScmParserVariable struct {
	Parser packrat.Parser[parserResult] // wrapper for parser
	Variable Symbol
}

type UndefinedParser struct { // a parser with forward declaration
	Parser packrat.Parser[parserResult] // if we finally found
	En *Env
	Sym Symbol
}

// allows self recursion on parsers
func (b *UndefinedParser) Match(s *packrat.Scanner[parserResult]) (packrat.Node[parserResult], bool) {
	if b.Parser == nil {
		en2 := b.En.FindRead(b.Sym)
		val, ok := en2.Vars[b.Sym]
		if !ok {
			panic("error parsing parser: variable does not contain a valid parser: " + string(b.Sym))
		}
		b.Parser = val.(packrat.Parser[parserResult])
	}
	return b.Parser.Match(s)
}

func (b *ScmParser) String() string {
	return "(parser ...)" // fallback generator
}

func (b *ScmParser) Match(s *packrat.Scanner[parserResult]) (packrat.Node[parserResult], bool) {
	m, ok := b.Root.Match(s)
	if !ok {
		return m, false
	}
	if b.Generator == nil {
		return packrat.Node[parserResult]{parserResult{m.Payload.value, nil}}, true // TODO: m.env, too?
	} else {
		var en2 Env
		// evaluate parser
		en2.Vars = m.Payload.env // take variable assignments
		en2.Outer = b.Outer
		en2.Nodefine = true
		return packrat.Node[parserResult]{parserResult{Eval(b.Generator, &en2), nil}}, true
	}
}

// TODO: create two variants of mergeParserResults, one where the result is not needed and thus nil can be returned as payload
func mergeParserResults(s string, r ...parserResult) parserResult {
	var m map[Symbol]Scmer
	var arr []Scmer
	for _, e := range r {
		arr = append(arr, e.value) // put results into array
		if e.env != nil { // merge env variables
			if m == nil {
				m = e.env // reuse the object (untested)
			} else {
				for k, v := range e.env {
					m[k] = v
				}
			}
		}
	}
	return parserResult{arr, m}
}
func mergeParserResultsNil(s string, r ...parserResult) parserResult {
	var m map[Symbol]Scmer
	for _, e := range r {
		if e.env != nil { // merge env variables
			if m == nil {
				m = e.env // reuse the object (untested)
			} else {
				for k, v := range e.env {
					m[k] = v
				}
			}
		}
	}
	return parserResult{nil, m}
}

func (b *ScmParser) Execute(str string, en *Env) Scmer {
	var skipper *regexp.Regexp = b.Skipper
	if skipper == nil {
		skipper = packrat.SkipWhitespaceAndCommentsRegex // also skip C-style comments as whitespaces
	}
	scanner := packrat.NewScanner[parserResult](str, skipper)
	node, err := packrat.Parse(b, scanner)
	if err != nil {
		panic(err)
	}
	return node.Payload.value
}

func (b *ScmParserVariable) Match(s *packrat.Scanner[parserResult]) (packrat.Node[parserResult], bool) {
	m, ok := b.Parser.Match(s)
	if !ok {
		return m, ok
	}
	env := m.Payload.env // reuse map from inner scope (risky but faster; corner cases not examined yet)
	if env == nil {
		env = make(map[Symbol]Scmer) // if problems occur, make this the default
	}
	/* otherwise:
	if m.Payload.env != nil {
		for k, v := range m.Payload.env {
			env[k] = v
		}
	}*/
	env[b.Variable] = m.Payload.value // add variable to scope (TODO: replace with fixed-size arrays?)
	return packrat.Node[parserResult]{Payload: parserResult{m.Payload, env}}, true
}


func parseSyntax(syntax Scmer, en *Env, ome *optimizerMetainfo, ignoreResult bool) packrat.Parser[parserResult] {
	merger := mergeParserResults
	if ignoreResult {
		merger = mergeParserResultsNil
	}
	switch n := syntax.(type) {
		case SourceInfo:
			return parseSyntax(n.value, en, ome, ignoreResult)
		case string:
			return packrat.NewAtomParser(parserResult{n, nil}, n, false, true)
		case packrat.Parser[parserResult]: // parser passthrough for precompiled parsers
			return n
		case Symbol:
			if n == Symbol("$") {
				return packrat.NewEndParser(parserResult{nil, nil}, true)
			}
			if n == Symbol("empty") {
				return packrat.NewEmptyParser(parserResult{nil, nil})
			}
			if ome != nil {
				// variables cannot be predefined
				// TODO: precompiled parsers from the OME environment?
				return nil
			}
			en2 := en.FindRead(n)
			if result, ok := en2.Vars[n].(*ScmParser); !ok {
				return &UndefinedParser{nil, en, n}
			} else {
				return result
			}
		case NthLocalVar:
			if ome != nil {
				// variables cannot be predefined
				return nil
			}
			if result, ok := en.VarsNumbered[n].(*ScmParser); !ok {
				panic("error invalid parser: " + String(en.VarsNumbered[n]))
			} else {
				return result
			}
		case []Scmer:
			if len(n) == 0 {
				panic("invalid parser ()")
			}
			switch n[0] {
				case Symbol("parser"): // inner anonymous parser
					var resulter Scmer
					if len(n) > 2 {
						Validate(n[2], "any")
						resulter = n[2]
					}
					var skipper Scmer = nil
					if len(n) > 3 {
						Validate(n[3], "string")
						skipper = n[3]
					}
					if ome != nil {
						// parsers cannot be created now, but we can sub-optimize them
						//n[1] = OptimizeParser(n[1], en, ome)
						return nil
					} else {
						// instanciate subparser
						return NewParser(n[1], resulter, skipper, en, ignoreResult)
					}
				case Symbol("atom"):
					caseinsensitive := false
					if len(n) > 2 {
						caseinsensitive = ToBool(n[2])
					}
					skipws := true
					if len(n) > 3 {
						skipws = ToBool(n[3])
					}
					value := n[1] // 4th param: atom value (default: the string itself)
					if len(n) > 4 {
						value = n[4]
					}
					return packrat.NewAtomParser(parserResult{value, nil}, String(n[1]), caseinsensitive, skipws)
				case Symbol("regex"):
					caseinsensitive := false
					if len(n) > 2 {
						caseinsensitive = ToBool(n[2])
					}
					skipws := true
					if len(n) > 3 {
						skipws = ToBool(n[3])
					}
					return packrat.NewRegexParser(func (s string) parserResult {return parserResult{s, nil}}, String(n[1]), caseinsensitive, skipws)
				case Symbol("list"):
					subparser := make([]packrat.Parser[parserResult], len(n)-1)
					for i := 1; i < len(n); i++ {
						subparser[i-1] = parseSyntax(n[i], en, ome, ignoreResult)
						if subparser[i-1] == nil {
							return nil
						}
					}
					return packrat.NewAndParser(merger, subparser...)
				case Symbol("or"):
					subparser := make([]packrat.Parser[parserResult], len(n)-1)
					for i := 1; i < len(n); i++ {
						subparser[i-1] = parseSyntax(n[i], en, ome, ignoreResult)
						if subparser[i-1] == nil {
							return nil
						}
					}
					return packrat.NewOrParser(subparser...)
				case Symbol("*"):
					subparser := parseSyntax(n[1], en, ome, ignoreResult)
					if subparser == nil {
						return nil
					}
					var sepparser packrat.Parser[parserResult]
					if len(n) > 2 {
						sepparser = parseSyntax(n[2], en, ome, ignoreResult)
						if sepparser == nil {
							return nil
						}
					} else {
						sepparser = packrat.NewEmptyParser(parserResult{nil, nil})
					}
					return packrat.NewKleeneParser(merger, subparser, sepparser)
				case Symbol("+"):
					subparser := parseSyntax(n[1], en, ome, ignoreResult)
					if subparser == nil {
						return nil
					}
					var sepparser packrat.Parser[parserResult]
					if len(n) > 2 {
						sepparser = parseSyntax(n[2], en, ome, ignoreResult)
						if sepparser == nil {
							return nil
						}
					} else {
						sepparser = packrat.NewEmptyParser(parserResult{nil, nil})
					}
					return packrat.NewKleeneParser(merger, subparser, sepparser)
				case Symbol("?"):
					if len(n) == 2 {
						// single element
						subparser := parseSyntax(n[1], en, ome, ignoreResult)
						if subparser == nil {
							return nil
						}
						return packrat.NewMaybeParser(parserResult{nil, nil}, subparser)
					} else {
						// maybe with a list
						subparser := make([]packrat.Parser[parserResult], len(n)-1)
						for i := 1; i < len(n); i++ {
							subparser[i-1] = parseSyntax(n[i], en, ome, ignoreResult)
							if subparser[i-1] == nil {
								return nil
							}
						}
						return packrat.NewMaybeParser(parserResult{nil, nil}, packrat.NewAndParser(merger, subparser...))
					}
				case Symbol("define"):
					result := new(ScmParserVariable)
					result.Variable = n[1].(Symbol)
					result.Parser = parseSyntax(n[2], en, ome, false)
					if result.Parser == nil {
						// uncompilable in the moment
						return nil
					}
					return result
			}
			// the optimizer does this, so we have to handle it
			if fmt.Sprint(n[0]) == fmt.Sprint(List) {
				subparser := make([]packrat.Parser[parserResult], len(n)-1)
				for i := 1; i < len(n); i++ {
					subparser[i-1] = parseSyntax(n[i], en, ome, ignoreResult)
					if subparser[i-1] == nil {
						return nil
					}
				}
				return packrat.NewAndParser(merger, subparser...)
			}
	}
	panic("Unknown parser syntax: " + fmt.Sprint(syntax))
}

func NewParser(syntax, generator, whitespace Scmer, en *Env, ignoreResult bool) *ScmParser {
	if generator != nil {
		ignoreResult = true
	}
	result := new(ScmParser)
	result.Root = parseSyntax(syntax, en, nil, ignoreResult)
	result.Syntax = syntax // for serialization purposes
	result.Generator = generator
	result.Outer = en
	if whitespace != nil {
		result.Skipper = regexp.MustCompile(String(whitespace))
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
$ EndParser
empty EmptyParser
symbol -> use other parser defined in env

for further details on packrat parsers, take a look at https://github.com/launix-de/go-packrat
`,
		1, 3,
		[]DeclarationParameter{
			DeclarationParameter{"syntax", "any", "syntax of the grammar (see docs)"},
			DeclarationParameter{"generator", "any", "(optional) expressions to evaluate. All captured variables are available in the scope."},
			DeclarationParameter{"skipper", "string", "(optional) string that defines the skip mechanism for whitespaces as regexp"},
		}, "func",
		nil,
	})
}

