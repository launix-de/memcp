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
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"bytes"
)

// TODO: (unquote string) -> symbol
// lexer defs: (set rules (list)); (set rules (cons new_rule rules))
// pattern matching (match pattern ifmatch pattern ifmatch else) -> function!
// -> pattern = string; pattern = regex
// -> (eval (cons (quote match) (cons value rules)))
// lexer = func (string, ruleset) -> nextfunc
// nextfunc = () -> (token, line, nextfunc)
// parser: func (token, state) -> state
// some kind of dictionary is needed
// (dict key value key value key value)
// (dict key value rest_dict)
// dict acts like a function; apply to a dict will yield the value

func ToBool(v Scmer) bool {
	switch v.(type) {
		case nil:
			return false
		case string:
			return v != ""
		case float64:
			return v != 0.0
		case bool:
			return v != false
		default:
			// []Scmer, native function, lambdas
			return true
	}
}

/*
 Eval / Apply
*/

func Eval(expression Scmer, en *Env) (value Scmer) {
	restart: // goto label because golang is lacking tail recursion, so just overwrite params and goto restart
	switch e := expression.(type) {
	case string:
		value = e
	case float64:
		value = e
	case Symbol:
		value = en.FindRead(e).Vars[e]
	case []Scmer:
		switch car, _ := e[0].(Symbol); car {
		case "quote":
			value = e[1]
		case "eval":
			// tail call optimized version
			expression = e[1]
			goto restart
		case "if":
			if ToBool(Eval(e[1], en)) {
				expression = e[2]
				goto restart
			} else {
				expression = e[3]
				goto restart
			}
		case "match": // (match <value> <pattern> <result> <pattern> <result> <pattern> <result> [<default>])
			val := Eval(e[1], en)
			i := 2
			en2 := Env{make(Vars), en, true}
			for i < len(e)-1 {
				if match(val, e[i], &en2) {
					// pattern has matched
					en = &en2
					expression = e[i+1]
					goto restart
				}
				i += 2
			}
			if i < len(e) {
				// default: nothing matched
				expression = e[i]
				goto restart // tail call
			} else {
				// otherwise: nil
				value = nil
			}
		/* set! is forbidden due to side effects
		case "set!":
			v := e[1].(Symbol)
			en2 := en.FindWrite(v)
			if en2 == nil {
				// not yet defined: set in innermost env
				en2 = en
			}
			en.Vars[v] = Eval(e[2], en)
			value = "ok"*/
		case "define", "set", "def": // set only works in innermost env
			// define will return itself back
			value = Eval(e[2], en)
			for en.Nodefine {
				// skip nodefine envs so that imports write to the global env
				en = en.Outer
			}
			en.Vars[e[1].(Symbol)] = value
		case "lambda":
			value = Proc{e[1], e[2], en}
		case "begin":
			// execute begin.. in own environment
			en2 := Env{make(Vars), en, false}
			for _, i := range e[1:len(e)-1] {
				Eval(i, &en2)
			}
			// tail call optimized version: last begin part will be tailed
			expression = e[len(e)-1]
			en = &en2
			goto restart
		default:
			// apply
			operands := e[1:]
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			procedure := Eval(e[0], en)
			switch p := procedure.(type) {
			case func(...Scmer) Scmer:
				return p(args...)
			case Proc:
				en2 := Env{make(Vars), p.En, false}
				switch params := p.Params.(type) {
				case []Scmer:
					for i, param := range params {
						en2.Vars[param.(Symbol)] = args[i]
					}
				default:
					en2.Vars[params.(Symbol)] = args
				}
				en = &en2
				expression = p.Body
				goto restart // tail call optimized
			case []Scmer: // associative list
				i := 0
				for i < len(p)-1 {
					if reflect.DeepEqual(args[0], p[i]) {
						return p[i+1]
					}
					i += 2
				}
				if i < len(p) {
					return p[i] // default value on n+1
				}
				return nil // no default value
			default:
				panic("Unknown procedure type - APPLY" + fmt.Sprint(p))
			}
		}
	case nil:
		return nil
	default:
		panic("Unknown expression type - EVAL" + fmt.Sprint(e))
	}
	return
}

// helper function; Eval uses a code duplicate to get the tail recursion done right
func Apply(procedure Scmer, args []Scmer) (value Scmer) {
	switch p := procedure.(type) {
	case func(...Scmer) Scmer:
		return p(args...)
	case Proc:
		en := &Env{make(Vars), p.En, false}
		switch params := p.Params.(type) {
		case []Scmer:
			for i, param := range params {
				en.Vars[param.(Symbol)] = args[i]
			}
		default:
			en.Vars[params.(Symbol)] = args
		}
		return Eval(p.Body, en)
	case []Scmer: // associative list
		i := 0
		for i < len(p)-1 {
			if reflect.DeepEqual(args[0], p[i]) {
				return p[i+1]
			}
			i += 2
		}
		if i < len(p) {
			return p[i] // default value on n+1
		}
		return nil // no default value
	default:
		panic("Unknown procedure type - APPLY" + fmt.Sprint(p))
	}
	return
}

// pattern matching
func match(val Scmer, pattern Scmer, en *Env) bool {
	/* our custom implementation of match consisting of:
	(match value pattern result pattern result pattern result [default])
	where pattern may be string, number, Symbol or list or applications
	 - string and float64 will match on equality
	 - Symbol will read the value into a variable
	 - list will unify the list contents ([]scmer{"list", ...})
	 - _ is dontcare
	 - (concat string Symbol) will split prefix
	 - (concat Symbol string Symbol) will split infix
	 - (concat Symbol string) will split postfix
	 - (cons x y) will split a list (x and y will be unified)
	 - (regex "(.*)=(.*)" _ Symbol Symbol) will parse regex
	*/
	switch p := pattern.(type) {
		case float64, string:
			return reflect.DeepEqual(val, p)
		case Symbol:
			en.Vars[p] = val
			return true
		case []Scmer:
			switch p[0] {
				case Symbol("list"):
					// list matching
					switch v := val.(type) {
						case []Scmer:
							// only list and list will match
							if len(v) != len(p) {
								return false
							}
							for i, p_item := range p {
								if !match(v[i], p_item, en) {
									return false
								}
							}
							return true
						default:
							return false
					}
				case Symbol("concat"):
					switch v := val.(type) {
						case string: // only allowed for strings
							// examine the pattern
							if len(p) == 3 {
								switch p1 := p[1].(type) {
									case string:
										switch p2 := p[2].(type) {
											case Symbol:
												// string Symbol
												if strings.HasPrefix(v, p1) {
													// extract postfix and match
													en.Vars[p2] = v[len(p1):]
													return true
												}
												// else
												return false
											default:
												// panic
										}
									default:
										// panic
								}
								// TODO: Symbol string
							}
							panic("unknown concat pattern: " + fmt.Sprint(p))
						default:
							return false // non-strings are not matching
					}
				case Symbol("cons"):
					panic("TODO: cons matching")
				case Symbol("regex"):
					panic("TODO: regex matching")
				default:
					panic("unknown match pattern: " + fmt.Sprint(p))
			}
		default:
			panic("unknown match pattern: " + fmt.Sprint(p))
	}
}

// TODO: func optimize für parzielle lambda-Ausdrücke und JIT

type Proc struct {
	Params, Body Scmer
	En           *Env
}

/*
 Environments
*/

type Vars map[Symbol]Scmer
type Env struct {
	Vars
	Outer *Env
	Nodefine bool // define will write to Outer
}

func (e *Env) FindRead(s Symbol) *Env {
	if _, ok := e.Vars[s]; ok {
		return e
	} else {
		if e.Outer == nil {
			return e
		}
		return e.Outer.FindRead(s)
	}
}

func (e *Env) FindWrite(s Symbol) *Env {
	if _, ok := e.Vars[s]; ok {
		return e
	} else {
		if e.Outer == nil {
			return nil
		}
		return e.Outer.FindWrite(s)
	}
}

/*
 Primitives
*/

var Globalenv Env

func init() {
	Globalenv = Env{
		Vars{ //aka an incomplete set of compiled-in functions
			"+": func(a ...Scmer) Scmer {
				v := a[0].(float64)
				for _, i := range a[1:] {
					v += i.(float64)
				}
				return v
			},
			"-": func(a ...Scmer) Scmer {
				v := a[0].(float64)
				for _, i := range a[1:] {
					v -= i.(float64)
				}
				return v
			},
			"*": func(a ...Scmer) Scmer {
				v := a[0].(float64)
				for _, i := range a[1:] {
					v *= i.(float64)
				}
				return v
			},
			"/": func(a ...Scmer) Scmer {
				v := a[0].(float64)
				for _, i := range a[1:] {
					v /= i.(float64)
				}
				return v
			},
			"<=": func(a ...Scmer) Scmer {
				return a[0].(float64) <= a[1].(float64)
			},
			"<": func(a ...Scmer) Scmer {
				return a[0].(float64) < a[1].(float64)
			},
			">": func(a ...Scmer) Scmer {
				return a[0].(float64) > a[1].(float64)
			},
			">=": func(a ...Scmer) Scmer {
				return a[0].(float64) >= a[1].(float64)
			},
			"equal?": func(a ...Scmer) Scmer {
				return reflect.DeepEqual(a[0], a[1])
			},
			"cons": func(a ...Scmer) Scmer {
				// cons a b: prepend item a to list b (construct list from item + tail)
				switch car := a[0]; cdr := a[1].(type) {
				case []Scmer:
					return append([]Scmer{car}, cdr...)
				default:
					return []Scmer{car, cdr}
				}
			},
			"car": func(a ...Scmer) Scmer {
				// head of tuple
				return a[0].([]Scmer)[0]
			},
			"cdr": func(a ...Scmer) Scmer {
				// rest of tuple
				return a[0].([]Scmer)[1:]
			},
			"concat": func(a ...Scmer) Scmer {
				// concat strings
				var b bytes.Buffer
				for _, s := range a {
					b.WriteString(String(s))
				}
				return b.String()
			},
			"true": true,
			"false": false,
			"symbol": func (a ...Scmer) Scmer {
				return Symbol(a[0].(string))
			},
			"list": Eval(Read(
				"(lambda z z)"),
				&Globalenv),
		},
		nil,
		false}
}

/* TODO: abs, quotient, remainder, modulo, gcd, lcm, expt, sqrt
zero?, negative?, positive?, off?, even?
max, min
sin, cos, tan, asin, acos, atan
exp, log
number->string, string->number
integer?, rational?, real?, complex?, number?
*/

/*
 Parsing
*/

//Symbols, numbers, expressions, procedures, lists, ... all implement this interface, which enables passing them along in the interpreter
type Scmer interface{}

type Symbol string  //Symbols are represented by strings
//Numbers by float64 (but no extra type)

func Read(s string) (expression Scmer) {
	tokens := tokenize(s)
	return readFrom(&tokens)
}

func EvalAll(s string, en *Env) (expression Scmer) {
	tokens := tokenize(s)
	for len(tokens) > 0 {
		expression = Eval(readFrom(&tokens), en)
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

/*
 Interactivity
*/

func String(v Scmer) string {
	switch v := v.(type) {
	case []Scmer:
		l := make([]string, len(v))
		for i, x := range v {
			l[i] = String(x)
		}
		return "(" + strings.Join(l, " ") + ")"
	case Proc:
		return "[func]"
	case func(...Scmer) Scmer:
		return "[native func]"
	default:
		return fmt.Sprint(v)
	}
}
func Serialize(b *bytes.Buffer, v Scmer, en *Env, glob *Env) {
	if en != glob {
		b.WriteString("(begin ")
		for k, v := range en.Vars {
			// if Symbol is defined in a lambda, print the real value
			// filter out redefinition of global functions
			if fmt.Sprint(glob.Vars[Symbol(k)]) != fmt.Sprint(v) {
				b.WriteString("(define ")
				b.WriteString(string(k)) // what if k contains spaces?? can it?
				b.WriteString(" ")
				Serialize(b, v, en.Outer, glob)
				b.WriteString(") ")
			}
		}
		Serialize(b, v, en.Outer, glob)
		b.WriteString(")")
		return
	}
	switch v := v.(type) {
	case []Scmer:
		b.WriteByte('(')
		for i, x := range v {
			if i != 0 {
				b.WriteByte(' ')
			}
			Serialize(b, x, en, glob)
		}
		b.WriteByte(')')
	case func(...Scmer) Scmer:
		// native func serialization is the hardest; reverse the env!
		// when later functional JIT is done, this must also handle deoptimization
		en2 := en
		for en2 != nil {
			for k, v2 := range en2.Vars {
				// compare function pointers (hacky but golang dosent give another opt)
				if fmt.Sprint(v) == fmt.Sprint(v2) {
					// found the right global function
					b.WriteString(fmt.Sprint(k)) // print out variable name
					return
				}
			}
			en2 = en2.Outer
		}
		b.WriteString("[unserializable native func]")
	case Proc:
		b.WriteString("(lambda ")
		Serialize(b, v.Params, glob, glob)
		b.WriteByte(' ')
		Serialize(b, v.Body, v.En, glob)
		b.WriteByte(')')
	case Symbol:
		// print as Symbol (because we already used a begin-block for defining our env)
		b.WriteString(fmt.Sprint(v))
	case string:
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\r", "\\r", "\n", "\\n").Replace(v))
		b.WriteByte('"')
	default:
		b.WriteString(fmt.Sprint(v))
	}
}
