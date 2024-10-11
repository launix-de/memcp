/*
Copyright (C) 2023-2024  Carl-Philip Hänsch
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
	"strings"
	"github.com/jtolds/gls"
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
	case nil:
		return nil
	case bool:
		value = e
	case string:
		value = e
	case int64:
		value = e
	case float64:
		value = e
	case Proc:
		value = e
	case *ScmParser:
		value = e
	case func(a ...Scmer) Scmer:
		value = e
	case Symbol:
		value = en.FindRead(e).Vars[e]
	case NthLocalVar:
		value = en.VarsNumbered[e]
	case []Scmer:
		if car, ok := e[0].(Symbol); ok {
			// switch-case through all special symbols
			switch car {
			case "outer": // execute value in outer scope
				//fmt.Println("eval outer",e[1],en.Outer,"in",en)
				value = Eval(e[1], en.Outer)
			case "quote":
				value = e[1]
			case "eval":
				// ...
				expression = Eval(e[1], en)
				goto restart
			case "time":
				// measure the time a step has taken
				if Trace != nil {
					if len(e) > 2 { // with label
						Trace.Duration(String(Eval(e[2], en)), "scm", func() {
							value = Eval(e[1], en)
						})
					} else {
						Trace.Duration("(time)", "scm", func() {
							value = Eval(e[1], en)
						})
					}
				} else {
					value = Eval(e[1], en)
				}
				return
			case "if":
				i := 1
				for i+1 < len(e) {
					if ToBool(Eval(e[i], en)) {
						expression = e[i+1]
						goto restart
					}
					i += 2
				}
				if i < len(e) { // else block
					expression = e[i]
					goto restart
				}
				return nil
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
			case "coalesce":
				for i := 1; i < len(e); i++ {
					x2 := Eval(e[i], en)
					if i == len(e)-1 || ToBool(x2) { // last value is taken even if ToBool is false, especially used for '()
						return x2
					}
				}
				return nil
			case "coalesceNil":
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
				en2 := Env{make(Vars), en.VarsNumbered, en, true}
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
			case "setN": // set numbered
				value = Eval(e[2], en)
				en.VarsNumbered[int(e[1].(NthLocalVar))] = value
			case "parser": // special form of lambda function
				if len(e) > 3 {
					value = NewParser(e[1], e[2], e[3], en, true)
				} else if len(e) > 2 {
					value = NewParser(e[1], e[2], nil, en, true)
				} else {
					value = NewParser(e[1], nil, nil, en, false)
				}
			case "lambda":
				switch si := e[1].(type) {
					case SourceInfo:
						// strip SourceInfo from lambda declarations
						e[1] = si.value
				}
				numVars := 0
				if len(e) > 3 {
					numVars = ToInt(e[3])
				}
				value = Proc{e[1], e[2], en, numVars}
			case "begin":
				// execute begin.. in own environment
				en2 := Env{make(Vars), en.VarsNumbered, en, false}
				for _, i := range e[1:len(e)-1] {
					Eval(i, &en2)
				}
				// tail call optimized version: last begin part will be tailed
				expression = e[len(e)-1]
				en = &en2
				goto restart
			case "!begin":
				// execute begin.. in parent environment
				for _, i := range e[1:len(e)-1] {
					Eval(i, en)
				}
				// tail call optimized version: last begin part will be tailed
				expression = e[len(e)-1]
				goto restart
			case "parallel":
				// execute all childs parallely, return null after finish
				errs := make(chan Scmer, 1)
				for _, i := range e[1:] {
					gls.Go(func(i Scmer) func() {
						return func() {
							defer func() {
								// catch errors and pass them on
								errs <- recover()
							}()
							Eval(i, en)
						}
					}(i))
				}
				for range e[1:] {
					if err := <- errs; err != nil {
						panic(err)
					}
				}
				return nil
			default:
				goto to_apply
			}
			return
		}
		to_apply:
		// apply
		operands := e[1:]
		procedure := Eval(e[0], en)
		switch p := procedure.(type) {
			case func(...Scmer) Scmer:
				args := make([]Scmer, len(operands))
				for i, x := range operands {
					args[i] = Eval(x, en)
				}
				return p(args...)
			case func(*Env, ...Scmer) Scmer:
				args := make([]Scmer, len(operands))
				for i, x := range operands {
					args[i] = Eval(x, en)
				}
				return p(en, args...)
			case *ScmParser:
				return p.Execute(String(Eval(e[1], en)), en)
			case Proc:
				en2 := Env{make(Vars), make([]Scmer, p.NumVars), p.En, false}
				switch params := p.Params.(type) {
				case []Scmer:
					if len(params) < len(operands) {
						panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(operands)))
					}
					if p.NumVars > 0 {
						for i, _ := range params {
							if i < len(operands) {
								en2.VarsNumbered[i] = Eval(operands[i], en)
							}
						}
					} else {
						for i, param := range params {
							if param != Symbol("_") {
								if i < len(operands) {
									en2.Vars[param.(Symbol)] = Eval(operands[i], en)
								} else {
									en2.Vars[param.(Symbol)] = nil
								}
							}
						}
					}
				case Symbol:
					args := make([]Scmer, len(operands))
					for i, x := range operands {
						args[i] = Eval(x, en)
					}
					if p.NumVars > 0 {
						en2.VarsNumbered[0] = args
					} else {
						en2.Vars[params] = args
					}
				case nil:
					// no arguments
				default:
				}
				en = &en2
				expression = p.Body
				goto restart // tail call optimized
			case []Scmer: // associative list
				// format: (key value key value ... default)
				if i, ok := operands[0].(NthLocalVar); ok {
					// indexed access generated through optimizer
					return p[i]
				} else {
					arg := Eval(operands[0], en)
					i := 0
					for i < len(p)-1 {
						if reflect.DeepEqual(arg, p[i]) {
							return p[i+1]
						}
						i += 2
					}
					if i < len(p) {
						return p[i] // default value on n+1
					}
					return nil // no default value
				}
			case nil:
				panic("Unknown function: " + fmt.Sprint(e[0]))
			default:
				panic("Unknown procedure type - APPLY " + fmt.Sprint(p))
		}
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
			return Apply(procedure, new_params...)
		default:
			panic("apply_assoc cannot run on non-list parameters")
		}
	default:
		panic("apply_assoc cannot run on non-lambdas")
	}
}

// helper function; Eval uses a code duplicate to get the tail recursion done right
func Apply(procedure Scmer, args ...Scmer) (value Scmer) {
	return ApplyEx(procedure, args, &Globalenv)
}
func ApplyEx(procedure Scmer, args []Scmer, en *Env) (value Scmer) {
	switch p := procedure.(type) {
	case func(...Scmer) Scmer:
		return p(args...)
	case func(*Env, ...Scmer) Scmer:
		return p(en, args...)
	case *ScmParser:
		return p.Execute(String(args[0]), en)
	case Proc:
		en := &Env{make(Vars), make([]Scmer, p.NumVars), p.En, false}
		switch params := p.Params.(type) {
		case []Scmer:
			if p.NumVars > 0 {
				for i, _ := range params {
					if i < len(args) {
						en.VarsNumbered[i] = args[i]
					}
				}
			} else {
				for i, param := range params {
					if param != Symbol("_") && i < len(args) {
						en.Vars[param.(Symbol)] = args[i]
					}
				}
			}
		case Symbol:
			if p.NumVars > 0 {
				en.VarsNumbered[0] = args
			} else {
				en.Vars[params] = args
			}
		case nil:
		}
		return Eval(p.Body, en)
	case []Scmer: // associative list
		// format: (key value key value ... default)
		if i, ok := args[0].(NthLocalVar); ok {
			// indexed access generated through optimizer
			return p[i]
		} else {
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
	NumVars      int
}

// helper pseudo type to optimize parameter reading from indices
type NthLocalVar uint8 // equals to (var i)

/*
 Environments
*/

type Vars map[Symbol]Scmer
type Env struct {
	Vars Vars
	VarsNumbered []Scmer // <- for the optimizer
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

func List(a ...Scmer) Scmer {
	return a
}
func isList(v Scmer)  bool {
	f, ok := v.(func(...Scmer) Scmer)
	if !ok {
		return false
	}
	return reflect.ValueOf(f).Pointer() == reflect.ValueOf(List).Pointer()
}
func init() {
	Globalenv = Env{
		Vars{ //aka an incomplete set of compiled-in functions
			"true": true,
			"false": false,

			// basic
			"list": List,
		},
		nil,
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
		"optimize", "optimize the given scheme program",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters"},
		}, "any", func (a ...Scmer) Scmer {
			return Optimize(a[0], &Globalenv)
		},
	})
	Declare(&Globalenv, &Declaration{
		"time", "measures the time it takes to compute the first argument",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "any", "code to execute"},
			DeclarationParameter{"label", "string", "label to print in the log or trace"},
		}, "any", nil,
	})
	Declare(&Globalenv, &Declaration{
		"if", "checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition...", "bool", "condition to evaluate"},
			DeclarationParameter{"true-branch...", "returntype", "code to evaluate if condition is true"},
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
		"coalesce", "returns the first value that has a non-zero value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine"},
		}, "returntype", nil,
	})
	Declare(&Globalenv, &Declaration{
		"coalesceNil", "returns the first value that has a non-nil value",
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
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value or message to throw"},
		}, "string",
		func (a ...Scmer) Scmer {
			if len(a) == 1 {
				panic(a[0])
			} else {
				var b strings.Builder
				for _, v := range a {
					b.WriteString(String(v))
				}
				panic(b.String())
			}
		},
	})
	Declare(&Globalenv, &Declaration{
		"try", "tries to execute a function and returns its result. In case of a failure, the error is fed to the second function and its result value will be used",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"func", "func", "function with no parameters that will be called"},
			DeclarationParameter{"errorhandler", "func", "function that takes the error as parameter"},
		}, "any",
		func (a ...Scmer) (result Scmer) {
			defer func() {
				err := recover()
				if err != nil {
					result = Apply(a[1], err)
				}
			}()
			result = Apply(a[0])
			return
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
			return Apply(a[0], a[1].([]Scmer)...)
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
		"string", "converts the given value into string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "any value"},
		}, "string",
		func (a ...Scmer) Scmer {
			return String(a[0])
		},
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
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"parameters", "symbol|list|nil", "if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol"},
			DeclarationParameter{"code", "any", "value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above"},
			DeclarationParameter{"numvars", "number", "number of unnamed variables that can be accessed via (var 0) (var 1) etc."},
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
		"parallel", "executes all parameters in parallel and returns nil if they are finished",
		1, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate in parallel"},
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
	init_date()
	init_parser()
	init_sync()
}

/* TODO: abs, quotient, remainder, modulo, gcd, lcm, expt, sqrt
zero?, negative?, positive?, off?, even?
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

