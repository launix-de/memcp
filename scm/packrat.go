/*
Copyright (C) 2023  Carl-Philip Hänsch

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
import packrat "github.com/launix-de/go-packrat"
import regexp "github.com/wasilibs/go-re2"

type ScmParser struct {
	Root packrat.Parser // wrapper for parser
	Syntax Scmer // keep syntax for deserializer
	Generator Scmer
	Skipper *regexp.Regexp
}

type ScmParserVariable struct {
	Parser packrat.Parser // wrapper for parser
	Variable Symbol
}

type UndefinedParser struct { // a parser with forward declaration
	Parser packrat.Parser // if we finally found
	En *Env
	Sym Symbol
}

// allows self recursion on parsers
func (b *UndefinedParser) Match(s *packrat.Scanner) *packrat.Node {
	if b.Parser == nil {
		en2 := b.En.FindRead(b.Sym)
		val, ok := en2.Vars[b.Sym]
		if !ok {
			panic("error parsing parser: variable does not contain a valid parser: " + string(b.Sym))
		}
		b.Parser = val.(packrat.Parser)
	}
	return b.Parser.Match(s)
}

func (b *ScmParser) String() string {
	return "(parser ...)" // fallback generator
}

func (b *ScmParser) Match(s *packrat.Scanner) *packrat.Node {
	m := b.Root.Match(s)
	if m == nil {
		return nil
	}
	return &packrat.Node{m.Matched, m.Start, b, []*packrat.Node{m}}
}

func findVarNodes(node *packrat.Node, en *Env) {
	if extractor, ok := node.Parser.(*ScmParserVariable); ok {
		en.Vars[extractor.Variable] = ExtractScmer(node.Children[0], en)
	}
	if _, ok := node.Parser.(*ScmParser); ok {
		return // early exit, don't deep-dive into their variables
	}
	for _, child := range node.Children {
		findVarNodes(child, en)
	}
}

func ExtractScmer(n *packrat.Node, en *Env) Scmer {
	switch parser := n.Parser.(type) {
		case *ScmParser:
			if parser.Generator == nil {
				return ExtractScmer(n.Children[0], en)
			} else {
				// call generator
				var en2 Env
				en2.Vars = make(map[Symbol]Scmer)
				en2.Outer = en
				en2.Nodefine = true
				findVarNodes(n.Children[0], &en2)
				return Eval(parser.Generator, &en2)
			}
		case *packrat.OrParser:
			return ExtractScmer(n.Children[0], en)
		case *packrat.KleeneParser:
			// build list from n.Children
			result := make([]Scmer, 0, len(n.Children)/2+1)
			for i := 0; i < len(n.Children); i += 2 {
				result = append(result, ExtractScmer(n.Children[i], en))
			}
			return result
		case *packrat.ManyParser:
			// build list from n.Children
			result := make([]Scmer, 0, len(n.Children)/2+1)
			for i := 0; i < len(n.Children); i += 2 {
				result = append(result, ExtractScmer(n.Children[i], en))
			}
			return result
		case *packrat.MaybeParser: // nil or value
			if len(n.Children) > 0 {
				return ExtractScmer(n.Children[0], en)
			} else {
				return nil
			}
	}
	return n.Matched
}

func (b *ScmParser) Execute(str string, en *Env) Scmer {
	var skipper *regexp.Regexp = b.Skipper
	if skipper == nil {
		skipper = packrat.SkipWhitespaceAndCommentsRegex // also skip C-style comments as whitespaces
	}
	scanner := packrat.NewScanner(str, skipper)
	node, err := packrat.Parse(b, scanner)
	if err != nil {
		panic(err)
	}
	return ExtractScmer(node, en)
}

func (b *ScmParserVariable) Match(s *packrat.Scanner) *packrat.Node {
	m := b.Parser.Match(s)
	if m == nil {
		return nil
	}
	return &packrat.Node{m.Matched, m.Start, b, []*packrat.Node{m}}
}


func parseSyntax(syntax Scmer, en *Env, ome *optimizerMetainfo) packrat.Parser {
	switch n := syntax.(type) {
		case SourceInfo:
			return parseSyntax(n.value, en, ome)
		case string:
			return packrat.NewAtomParser(n, false, true)
		case packrat.Parser: // parser passthrough for precompiled parsers
			return n
		case Symbol:
			if n == Symbol("$") {
				return packrat.NewEndParser(true)
			}
			if n == Symbol("empty") {
				return packrat.NewEmptyParser()
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
						return NewParser(n[1], resulter, skipper, en)
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
					return packrat.NewAtomParser(String(n[1]), caseinsensitive, skipws)
				case Symbol("regex"):
					caseinsensitive := false
					if len(n) > 2 {
						caseinsensitive = ToBool(n[2])
					}
					skipws := true
					if len(n) > 3 {
						skipws = ToBool(n[3])
					}
					return packrat.NewRegexParser(String(n[1]), caseinsensitive, skipws)
				case Symbol("list"):
					subparser := make([]packrat.Parser, len(n)-1)
					for i := 1; i < len(n); i++ {
						subparser[i-1] = parseSyntax(n[i], en, ome)
						if subparser[i-1] == nil {
							return nil
						}
					}
					return packrat.NewAndParser(subparser...)
				case Symbol("or"):
					subparser := make([]packrat.Parser, len(n)-1)
					for i := 1; i < len(n); i++ {
						subparser[i-1] = parseSyntax(n[i], en, ome)
						if subparser[i-1] == nil {
							return nil
						}
					}
					return packrat.NewOrParser(subparser...)
				case Symbol("*"):
					subparser := parseSyntax(n[1], en, ome)
					if subparser == nil {
						return nil
					}
					var sepparser packrat.Parser
					if len(n) > 2 {
						sepparser = parseSyntax(n[2], en, ome)
						if sepparser == nil {
							return nil
						}
					} else {
						sepparser = packrat.NewEmptyParser()
					}
					return packrat.NewKleeneParser(subparser, sepparser)
				case Symbol("+"):
					subparser := parseSyntax(n[1], en, ome)
					if subparser == nil {
						return nil
					}
					var sepparser packrat.Parser
					if len(n) > 2 {
						sepparser = parseSyntax(n[2], en, ome)
						if sepparser == nil {
							return nil
						}
					} else {
						sepparser = packrat.NewEmptyParser()
					}
					return packrat.NewKleeneParser(subparser, sepparser)
				case Symbol("?"):
					if len(n) == 2 {
						// single element
						subparser := parseSyntax(n[1], en, ome)
						if subparser == nil {
							return nil
						}
						return packrat.NewMaybeParser(subparser)
					} else {
						// maybe with a list
						subparser := make([]packrat.Parser, len(n)-1)
						for i := 1; i < len(n); i++ {
							subparser[i-1] = parseSyntax(n[i], en, ome)
							if subparser[i-1] == nil {
								return nil
							}
						}
						return packrat.NewMaybeParser(packrat.NewAndParser(subparser...))
					}
				case Symbol("define"):
					result := new(ScmParserVariable)
					result.Variable = n[1].(Symbol)
					result.Parser = parseSyntax(n[2], en, ome)
					if result.Parser == nil {
						// uncompilable in the moment
						return nil
					}
					return result
			}
			// the optimizer does this, so we have to handle it
			if fmt.Sprint(n[0]) == fmt.Sprint(List) {
				subparser := make([]packrat.Parser, len(n)-1)
				for i := 1; i < len(n); i++ {
					subparser[i-1] = parseSyntax(n[i], en, ome)
					if subparser[i-1] == nil {
						return nil
					}
				}
				return packrat.NewAndParser(subparser...)
			}
	}
	panic("Unknown parser syntax: " + fmt.Sprint(syntax))
}

func NewParser(syntax, generator, whitespace Scmer, en *Env) *ScmParser {
	result := new(ScmParser)
	result.Root = parseSyntax(syntax, en, nil)
	result.Syntax = syntax // for serialization purposes
	result.Generator = generator
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

