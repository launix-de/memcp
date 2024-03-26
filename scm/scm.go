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
	"time"
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

type Applicable interface {
	Apply(...Scmer) Scmer
}

/*
 Eval / Apply
*/

func Eval(expression Scmer, en *Env) (value Scmer) {
	restart: // goto label because golang is lacking tail recursion, so just overwrite params and goto restart
	switch e := expression.(type) {
	case SourceInfo:
		// omit source info
		expression = e.value
		defer func() {
			err := recover()
			if err != nil {
				// recursively panic a stack trace
				panic(fmt.Sprintf("%s\nin %s:%d:%d", fmt.Sprint(err), e.source, e.line, e.col))
			}
		}()
		goto restart
	case bool:
		value = e
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
		case "time":
			// similar to eval
			start := time.Now() // time measurement
			value = Eval(e[1], en)
			fmt.Println(time.Since(start))
			return
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
		case "collate":
			for i, x := range e {
				x2 := Eval(x, en)
				if i > 0 && ToBool(x2) {
					return x2
				}
			}
			return nil
		case "collateNil":
			for i, x := range e {
				x2 := Eval(x, en)
				if i > 0 && x2 != nil {
					return x2
				}
			}
			return nil
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
		case "define", "set": // set only works in innermost env
			// define will return itself back
			value = Eval(e[2], en)
			for en.Nodefine {
				// skip nodefine envs so that imports write to the global env
				en = en.Outer
			}
			en.Vars[e[1].(Symbol)] = value
		case "parser": // special form of lambda function
			if len(e) > 3 {
				value = NewParser(e[1], e[2], e[3], en)
			} else if len(e) > 2 {
				value = NewParser(e[1], e[2], nil, en)
			} else {
				value = NewParser(e[1], nil, nil, en)
			}
		case "lambda":
			switch si := e[1].(type) {
				case SourceInfo:
					// strip SourceInfo from lambda declarations
					e[1] = si.value
			}
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
			case *ScmParser:
				return p.Execute(String(Eval(e[1], en)), en)
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
			case Applicable:
				return p.Apply(args)
			case nil:
				panic("Unknown function: " + fmt.Sprint(e[0]))
			default:
				panic("Unknown procedure type - APPLY " + fmt.Sprint(p))
			}
		}
	case nil:
		return nil
	default:
		panic("Unknown expression type - EVAL " + fmt.Sprint(e))
	}
	return
}

func ApplyAssoc(procedure Scmer, args []Scmer) (value Scmer) {
	switch p := procedure.(type) {
	case Proc:
		switch params := p.Params.(type) {
		case []Scmer:
			new_params := make([]Scmer, len(params))
			for i, sym := range params {
				for j := 0; j < len(args); j += 2 {
					if args[j] == String(sym) {
						new_params[i] = args[j+1]
					}
				}
			}
			return Apply(procedure, new_params)
		default:
			panic("apply_assoc cannot run on non-list parameters")
		}
	default:
		panic("apply_assoc cannot run on non-lambdas")
	}
}

// helper function; Eval uses a code duplicate to get the tail recursion done right
func Apply(procedure Scmer, args []Scmer) (value Scmer) {
	switch p := procedure.(type) {
	case func(...Scmer) Scmer:
		return p(args...)
	case *ScmParser:
		return p.Execute(String(args[0]), &Globalenv)
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
	case Applicable:
		return p.Apply(args)
	case nil:
		panic("Unknown function")
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
			"true": true,
			"false": false,

			// basic
			"list": Eval(Optimize(Read("internal", "(lambda z z)"), &Globalenv), &Globalenv),
		},
		nil,
		false,
	}

	// system
	DeclareTitle("SCM Builtins")
	Declare(&Globalenv, &Declaration{
		"quote", "returns a symbol or list without evaluating it",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"symbol", "symbol", "symbol to quote"},
		}, "symbol", nil,
	})
	Declare(&Globalenv, &Declaration{
		"eval", "executes the given scheme program in the current environment",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters"},
		}, "any", nil,
	})
	Declare(&Globalenv, &Declaration{
		"if", "checks a condition and then conditionally evaluates code branches",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "bool", "condition to evaluate"},
			DeclarationParameter{"true-branch", "returntype", "code to evaluate if condition is true"},
			DeclarationParameter{"false-branch", "returntype", "code to evaluate if condition is false"},
		}, "returntype", nil,
	})
	Declare(&Globalenv, &Declaration{
		"and", "returns true if all conditions evaluate to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "bool", "condition to evaluate"},
		}, "bool", nil,
	})
	Declare(&Globalenv, &Declaration{
		"or", "returns true if at least one condition evaluates to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "any", "condition to evaluate"},
		}, "bool", nil,
	})
	Declare(&Globalenv, &Declaration{
		"collate", "returns the first value that has a non-zero value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine"},
		}, "returntype", nil,
	})
	Declare(&Globalenv, &Declaration{
		"collateNil", "returns the first value that has a non-nil value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine"},
		}, "returntype", nil,
	})
	Declare(&Globalenv, &Declaration{
		"define", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set"},
			DeclarationParameter{"value", "returntype", "value to set the variable to"},
		}, "bool", nil,
	})
	Declare(&Globalenv, &Declaration{
		"set", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set"},
			DeclarationParameter{"value", "returntype", "value to set the variable to"},
		}, "bool", nil,
	})

	// basic
	Declare(&Globalenv, &Declaration{
		"error", "halts the whole execution thread and throws an error message",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value or message to throw"},
		}, "string",
		func (a ...Scmer) Scmer {
			panic(a[0])
		},
	})
	Declare(&Globalenv, &Declaration{
		"apply", "runs the function with its arguments",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute"},
			DeclarationParameter{"arguments", "list", "list of arguments to apply"},
		}, "symbol",
		func (a ...Scmer) Scmer {
			return Apply(a[0], a[1].([]Scmer))
		},
	})
	Declare(&Globalenv, &Declaration{
		"apply_assoc", "runs the function with its arguments but arguments is a assoc list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute (must be a lambda)"},
			DeclarationParameter{"arguments", "list", "assoc list of arguments to apply"},
		}, "symbol",
		func (a ...Scmer) Scmer {
			return ApplyAssoc(a[0], a[1].([]Scmer))
		},
	})
	Declare(&Globalenv, &Declaration{
		"symbol", "returns a symbol built from that string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string value that will be converted into a symbol"},
		}, "symbol",
		func (a ...Scmer) Scmer {
			return Symbol(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"list", "returns a list containing the parameters as alements",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value for the list"},
		}, "list",
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"match", `takes a value evaluates the branch that first matches the given pattern
Patterns can be any of:
 - symbol matches any value and stores is into a variable
 - "string" (matches only this string)
 - number (matches only this value)
 - (symbol "something") will only match the symbol 'something'
 - '(subpattern subpattern...) matches a list with exactly these subpatterns
 - (concat str1 str2 str3) will decompose a string into one of the following patterns: "prefix" variable, variable "postfix", variable "infix" variable
 - (cons a b) will reverse the cons function, so it will match the head of the list with a and the rest with b
 - (regex "pattern" text var1 var2...) will match the given regex pattern, store the whole string into text and all capture groups into var1, var2...
`,
		3, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to evaluate"},
			DeclarationParameter{"pattern...", "any", "pattern"},
			DeclarationParameter{"result...", "returntype", "result value when the pattern matches; this code can use the variables matched in the pattern"},
			DeclarationParameter{"default", "any", "(optional) value that is returned when no pattern matches"}, /* TODO: turn to returntype as soon as pattern+result are properly repeaded in Validate */
		}, "any", // TODO: returntype as soon as repead validate is implemented */
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"lambda", "returns a function (func) constructed from the given code",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"parameters", "symbol|list", "if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol"},
			DeclarationParameter{"code", "any", "value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above"},
		}, "func", // TODO: func(...)->returntype as soon as function types are implemented
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"begin", "creates a own variable scope, evaluates all sub expressions and returns the result of the last one",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate"},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "any", // TODO: returntype as soon as repeat is implemented
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"source", "annotates the node with filename and line information for better backtraces",
		4, 4,
		[]DeclarationParameter{
			DeclarationParameter{"filename", "string", "Filename of the code"},
			DeclarationParameter{"line", "number", "Line of the code"},
			DeclarationParameter{"column", "number", "Column of the code"},
			DeclarationParameter{"code", "returntype", "code"},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "returntype",
		func (a ...Scmer) Scmer {
			return SourceInfo{
				String(a[0]),
				ToInt(a[1]),
				ToInt(a[2]),
				a[3],
			}
		},
	})

	init_alu()
	init_strings()
	init_list()
	init_parser()
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

