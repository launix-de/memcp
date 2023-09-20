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
	"bytes"
	"strings"
	"strconv"
	"reflect"
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

//go:inline
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
//go:inline
func ToInt(v Scmer) int {
	switch vv := v.(type) {
		case nil:
			return 0
		case string:
			x, _ := strconv.Atoi(vv)
			return x
		case float64:
			return int(vv)
		case bool:
			if vv {
				return 1
			} else {
				return 0
			}
		default:
			// []Scmer, native function, lambdas
			return 1
	}
}
//go:inline
func ToFloat(v Scmer) float64 {
	switch vv := v.(type) {
		case string:
			x, _ := strconv.ParseFloat(vv, 64)
			return x
		case float64:
			return vv
		case bool:
			if vv {
				return 1.0
			} else {
				return 0.0
			}
		default:
			// nil, []Scmer, native function, lambdas
			return 0.0
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
			// ...
			expression = Eval(e[1], en)
			goto restart
		case "if":
			if ToBool(Eval(e[1], en)) {
				expression = e[2]
				goto restart
			} else {
				if len(e) > 3 {
					expression = e[3]
					goto restart
				} else {
					return nil
				}
			}
		case "and":
			for i, x := range e {
				if i > 0 && !ToBool(Eval(x, en)) {
					return false
				}
			}
			return true
		case "or":
			for i, x := range e {
				if i > 0 && ToBool(Eval(x, en)) {
					return true
				}
			}
			return false
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
					if len(params) > len(args) {
						panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(args)))
					}
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
				if len(p) == 0 {
					return nil
				} else {
					switch p[0].(type) {
						case []Scmer:
							// format: ((key values ...) (key values ...) ...)
							i := 0
							for i < len(p) {
								if reflect.DeepEqual(args[0], p[i].([]Scmer)[0]) {
									return p[i]
								}
								i++
							}
							return nil // no default value
						default:
							// format: (key value key value ... default)
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
					}
				}
			default:
				panic("Unknown procedure type - APPLY " + fmt.Sprint(p))
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
			if len(params) > len(args) {
				panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(args)))
			}
			for i, param := range params {
				en.Vars[param.(Symbol)] = args[i]
			}
		default:
			en.Vars[params.(Symbol)] = args
		}
		return Eval(p.Body, en)
	case []Scmer: // associative list
		if len(p) == 0 {
			return nil
		} else {
			switch p[0].(type) {
				case []Scmer:
					// format: ((key values ...) (key values ...) ...)
					i := 0
					for i < len(p) {
						if reflect.DeepEqual(args[0], p[i].([]Scmer)[0]) {
							return p[i]
						}
						i++
					}
					return nil // no default value
				default:
					// format: (key value key value ... default)
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
			}
		}
	default:
		panic("Unknown procedure type - APPLY " + fmt.Sprint(p))
	}
	return
}

// TODO: func optimize für parzielle lambda-Ausdrücke und JIT
// TODO: Proc2 for an optimized Env based on arrays rather than maps

type Proc struct {
	Params, Body Scmer
	En           *Env
}

/*
 Environments
*/

type Vars map[Symbol]Scmer
type Env struct {
	Vars Vars
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
			// arithmetic / logic
			"+": func(a ...Scmer) Scmer {
				v := ToFloat(a[0])
				for _, i := range a[1:] {
					v += ToFloat(i)
				}
				return v
			},
			"-": func(a ...Scmer) Scmer {
				v := ToFloat(a[0])
				for _, i := range a[1:] {
					v -= ToFloat(i)
				}
				return v
			},
			"*": func(a ...Scmer) Scmer {
				v := ToFloat(a[0])
				for _, i := range a[1:] {
					v *= ToFloat(i)
				}
				return v
			},
			"/": func(a ...Scmer) Scmer {
				v := ToFloat(a[0])
				for _, i := range a[1:] {
					v /= ToFloat(i)
				}
				return v
			},
			"<=": func(a ...Scmer) Scmer {
				// TODO: string vs. float
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
			"!": func(a ...Scmer) Scmer {
				return !ToBool(a[0]);
			},
			"not": func(a ...Scmer) Scmer {
				return !ToBool(a[0]);
			},
			"true": true,
			"false": false,

			// string functions
			"concat": func(a ...Scmer) Scmer {
				// concat strings
				var b bytes.Buffer
				for _, s := range a {
					b.WriteString(String(s))
				}
				return b.String()
			},
			"simplify": func(a ...Scmer) Scmer {
				// turn string to number or so
				return Simplify(String(a[0]))
			},
			"strlen": func(a ...Scmer) Scmer {
				// string
				return float64(len(String(a[0])))
			},
			"toLower": func(a ...Scmer) Scmer {
				// string
				return strings.ToLower(String(a[0]))
			},
			"toUpper": func(a ...Scmer) Scmer {
				// string
				return strings.ToUpper(String(a[0]))
			},
			"split": func(a ...Scmer) Scmer {
				// string, sep
				split := " "
				if len(a) > 1 {
					split = String(a[1])
				}
				ar := strings.Split(String(a[0]), split)
				result := make([]Scmer, len(ar))
				for i, v := range ar {
					result[i] = v
				}
				return result
			},

			// list functions
			"append": func(a ...Scmer) Scmer {
				// append a b ...: append item b to list a (construct list from item + tail)
				return append(a[0].([]Scmer), a[1:]...)
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
			"merge": func (a ...Scmer) Scmer {
				// merge arrays into one
				size := 0
				for _, v := range a[0].([]Scmer) {
					size = size + len(v.([]Scmer))
				}
				result := make([]Scmer, size)
				pos := 0
				for _, v := range a[0].([]Scmer) {
					inner := v.([]Scmer)
					copy(result[pos:pos+len(inner)], inner)
					pos = pos + len(inner)
				}
				return result
			},
			"has?": func(a ...Scmer) Scmer {
				// arr, element
				list := a[0].([]Scmer)
				for _, v := range list {
					if reflect.DeepEqual(a[1], v) {
						return true
					}
				}
				return false
			},
			"filter": func(a ...Scmer) Scmer {
				result := make([]Scmer, 0)
				for _, v := range a[0].([]Scmer) {
					if ToBool(Apply(a[1], []Scmer{v,})) {
						result = append(result, v)
					}
				}
				return result
			},
			"map": func(a ...Scmer) Scmer {
				list := a[0].([]Scmer)
				result := make([]Scmer, len(list))
				for i, v := range list {
					result[i] = Apply(a[1], []Scmer{v,})
				}
				return result
			},
			"reduce": func(a ...Scmer) Scmer {
				// arr, reducefn(a, b), [neutral]
				list := a[0].([]Scmer)
				var result Scmer = nil
				i := 0
				if len(a) > 2 {
					result = a[2]
				} else {
					if len(list) > 0 {
						result = list[0]
						i = i + 1
					}
				}
				for i < len(list) {
					result = Apply(a[1], []Scmer{result, list[i],})
					i = i + 1
				}
				return result
			},

			// dictionary functions
			"filter_assoc": func(a ...Scmer) Scmer {
				// list, fn(key, value)
				list := a[0].([]Scmer)
				result := make([]Scmer, 0)
				for i := 0; i < len(list); i += 2 {
					if ToBool(Apply(a[1], []Scmer{list[i], list[i+1]})) {
						result = append(result, list[i], list[i+1])
					}
				}
				return result
			},
			"map_assoc": func(a ...Scmer) Scmer {
				// apply fn(key value) to each assoc item and return mapped dict
				list := a[0].([]Scmer)
				result := make([]Scmer, len(list))
				var k Scmer
				for i, v := range list {
					if i % 2 == 0 {
						// key -> remain
						result[i] = v
						k = v
					} else {
						// value -> map fn(key, value)
						result[i] = Apply(a[1], []Scmer{k, v,})
					}
				}
				return result
			},
			"reduce_assoc": func(a ...Scmer) Scmer {
				// dict, reducefn(a, key, value), neutral
				list := a[0].([]Scmer)
				result := a[2]
				for i := 0; i < len(list); i += 2 {
					result = Apply(a[1], []Scmer{result, list[i], list[i+1],})
				}
				return result
			},
			"has_assoc?": func(a ...Scmer) Scmer {
				// dict, element
				list := a[0].([]Scmer)
				for i := 0; i < len(list); i += 2 {
					if reflect.DeepEqual(list[i], a[1]) {
						return true
					}
				}
				return false
			},
			"extract_assoc": func(a ...Scmer) Scmer {
				// apply fn(key value) to each assoc item and return results as array
				list := a[0].([]Scmer)
				result := make([]Scmer, len(list) / 2)
				var k Scmer
				for i, v := range list {
					if i % 2 == 0 {
						// key -> remain
						k = v
					} else {
						// value -> map fn(key, value)
						result[i / 2] = Apply(a[1], []Scmer{k, v,})
					}
				}
				return result
			},
			"set_assoc": func(a ...Scmer) Scmer {
				// may eventually destroy the original list; use in aggregations
				// params: dict, key, new_value, [merge_func]
				// return: dict
				list := a[0].([]Scmer)
				for i := 0; i < len(list); i += 2 {
					if reflect.DeepEqual(list[i], a[1]) {
						// overwrite
						if len(a) > 3 {
							// overwrite with merge function
							list[i + 1] = Apply(a[3], []Scmer{list[i + 1], a[2],})
						} else {
							// overwrite naive
							list[i + 1] = a[2]
						}
						return list // return changed list (this violates immutability for performance)
					}
				}
				// else: append
				return append(list, a[1], a[2])
			},
			"merge_assoc": func(a ...Scmer) Scmer {
				// params: dict1, dict2, [merge_func]
				// naive implementation, bad performance
				set_assoc := Globalenv.Vars["set_assoc"].(func(...Scmer) Scmer)
				list := a[0]
				dict := a[1].([]Scmer)
				if len(a) > 2 {
					for i := 0; i < len(dict); i += 2 {
						list = set_assoc(list, dict[i], dict[i+1], a[2])
					}
				} else {
					for i := 0; i < len(dict); i += 2 {
						list = set_assoc(list, dict[i], dict[i+1])
					}
				}
				return list
			},

			// basic
			"error": func (a ...Scmer) Scmer {
				panic(a[0])
			},
			"symbol": func (a ...Scmer) Scmer {
				return Symbol(String(a[0]))
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

