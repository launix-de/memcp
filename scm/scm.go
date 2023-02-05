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
			// argument is a list with already evaluated values that we put into apply
			arr := Eval(e[1], en).([]Scmer)
			value = Apply(arr[0], arr[1:])
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
			"append": func(a ...Scmer) Scmer {
				// cons a b: prepend item a to list b (construct list from item + tail)
				return append([]Scmer{a[0]}, a[1:]...)
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
			"error": func (a ...Scmer) Scmer {
				panic(a[0])
			},
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

