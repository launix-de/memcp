/*
Copyright (C) 2023  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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

import (
	"strings"
	"strconv"
)

func Simplify(s string) Scmer {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

func Read(s string) (expression Scmer) {
	tokens := tokenize(s)
	return readFrom(&tokens)
}

func EvalAll(source, s string, en *Env) (expression Scmer) {
	tokens := tokenize(s)
	for len(tokens) > 0 {
		code := readFrom(&tokens)
		Validate(source, code) // TODO: add some extra line number info to source??
		code = Optimize(code, en)
		expression = Eval(code, en)
	}
	return
}

//Syntactic Analysis
func readFrom(tokens *[]Scmer) (expression Scmer) {
	if len(*tokens) == 0 {
		return nil
	}
	//pop first element from tokens
	token := (*tokens)[0]
	*tokens = (*tokens)[1:]
	switch token.(type) {
		case Symbol:
			if token == Symbol("(") {
				L := make([]Scmer, 0)
				for { // read params until )
					if len(*tokens) == 0 {
						panic("expecting matching )")
					}
					if (*tokens)[0] == Symbol(")") {
						// eat )
						*tokens = (*tokens)[1:]
						return L // finish read process
					}
					// add param
					L = append(L, readFrom(tokens))
				}
			} else if token == Symbol("'") && len(*tokens) > 0 {
				if (*tokens)[0] == Symbol("(") {
					*tokens = (*tokens)[1:]
					// list literal
					L := make([]Scmer, 1)
					L[0] = Symbol("list")
					for (*tokens)[0] != Symbol(")") {
						L = append(L, readFrom(tokens))
					}
					*tokens = (*tokens)[1:]
					return L
				} else {
					return token
				}
			} else {
				return token
			}
		default:
			// string, Number
			return token
	}
}

//Lexical Analysis
func tokenize(s string) []Scmer {
	/* tokenizer state machine:
		0 = expecting next item
		1 = inside Number
		2 = inside Symbol
		3 = inside string
		4 = inside escaping sequence of string
		5 = inside comment
		6 = comment ending * from * /
	
	tokens are either Number, Symbol, string or Symbol('(') or Symbol(')')
	*/
	stringreplacer := strings.NewReplacer("\\\"", "\"", "\\\\", "\\", "\\n", "\n", "\\r", "\r", "\\t", "\t")
	state := 0
	startToken := 0
	result := make([]Scmer, 0)
	for i, ch := range s {
		if state == 1 && (ch == '.' || ch >= '0' && ch <= '9') {
			// another character added to Number
		} else if state == 2 && ch == '*' && s[startToken:i] == "/" {
			// begin of comment
			state = 5
		} else if state == 5 && ch == '*' {
			// comment seems to end
			state = 6
		} else if state == 5 {
			// consume another character in comment (TODO: nested comment counting??)
		} else if state == 6 && ch == '/' {
			// end comment
			state = 0
		} else if state == 6 {
			// continue comment
			state = 5
		} else if state == 2 && ch != ' ' && ch != '\r' && ch != '\n' && ch != '\t' && ch != ')' && ch != '(' {
			// another character added to Symbol
		} else if state == 3 && ch != '"' && ch != '\\' {
			// another character added to string
		} else if state == 3 && ch == '\\' {
			// escape sequence
			state = 4
		} else if state == 4 {
			state = 3 // continue with string
		} else if state == 3 && ch == '"' {
			// finish string
			result = append(result, stringreplacer.Replace(string(s[startToken+1:i])))
			state = 0
		} else {
			// otherwise: state change!
			if state == 1 {
				// finish Number
				if f, err := strconv.ParseFloat(s[startToken:i], 64); err == nil {
					result = append(result, float64(f))
				} else if s[startToken:i] == "-" {
					result = append(result, Symbol("-"))
				} else {
					result = append(result, Symbol("NaN"))
				}
			}
			if state == 2 {
				// finish Symbol
				result = append(result, Symbol(s[startToken:i]))
			}
			// now detect what to parse next
			startToken = i
			if ch == '(' {
				result = append(result, Symbol("("))
				state = 0
			} else if ch == ')' {
				result = append(result, Symbol(")"))
				state = 0
			} else if ch == '"' {
				// start string
				state = 3
			} else if ch >= '0' && ch <= '9' || ch == '-' {
				// start Number
				state = 1
			} else if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
				// white space
				state = 0
			} else {
				// everything else is a Symbol! (Symbols only are stopped by ' ()')
				state = 2
			}

		}
	}
	// in the end: finish unfinished Symbols and Numbers
	if state == 1 {
		// finish Number
		if f, err := strconv.ParseFloat(s[startToken:], 64); err == nil {
			result = append(result, float64(f))
		} else if s[startToken:] == "-" {
			result = append(result, Symbol("-"))
		} else {
			result = append(result, Symbol("NaN"))
		}
	}
	if state == 2 {
		// finish Symbol
		result = append(result, Symbol(s[startToken:]))
	}
	return result
}
