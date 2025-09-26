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
	"github.com/jtolds/gls"
	"reflect"
	"strings"
	"time"
)

func symbolName(v Scmer) (string, bool) {
	if v.IsSourceInfo() {
		return symbolName(v.SourceInfo().value)
	}
	if auxTag(v.aux) == tagSymbol {
		return v.String(), true
	}
	if auxTag(v.aux) == tagAny {
		if sym, ok := v.Any().(Symbol); ok {
			return string(sym), true
		}
	}
	return "", false
}

func mustSymbol(v Scmer) Symbol {
	if name, ok := symbolName(v); ok {
		return Symbol(name)
	}
	panic("expected symbol")
}

func mustNthLocalVar(v Scmer) NthLocalVar {
	if v.IsSourceInfo() {
		return mustNthLocalVar(v.SourceInfo().value)
	}
	if auxTag(v.aux) == tagAny {
		if idx, ok := v.Any().(NthLocalVar); ok {
			return idx
		}
	}
	panic("expected numbered local variable")
}

func evalWithSourceInfo(si SourceInfo, en *Env) (value Scmer) {
	defer func(src SourceInfo) {
		if err := recover(); err != nil {
			panic(fmt.Sprintf("%s\nin %s:%d:%d", fmt.Sprint(err), src.source, src.line, src.col))
		}
	}(si)
	return Eval(si.value, en)
}

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
restart:
	switch auxTag(expression.aux) {
	case tagSourceInfo:
		return evalWithSourceInfo(*expression.SourceInfo(), en)
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return expression
	case tagFunc:
		return expression
	case tagProc:
		return expression
	case tagSymbol:
		return en.FindRead(mustSymbol(expression)).Vars[mustSymbol(expression)]
	case tagNthLocalVar:
		return en.VarsNumbered[expression.NthLocalVar()]
	case tagSlice:
		list := expression.Slice()
		if len(list) == 0 {
			return expression
		}
		if headSym, ok := list[0].Any().(Symbol); ok {
			switch string(headSym) {
			case "outer":
				return Eval(list[1], en.Outer)
			case "quote":
				return list[1]
			case "eval":
				expression = Eval(list[1], en)
				goto restart
			case "time":
				var start time.Time
				if TracePrint {
					start = time.Now()
				}
				var timedResult Scmer
				if Trace != nil {
					if len(list) > 2 {
						Trace.Duration(String(Eval(list[2], en)), "scm", func() {
							timedResult = Eval(list[1], en)
						})
					} else {
						Trace.Duration("(time)", "scm", func() {
							timedResult = Eval(list[1], en)
						})
					}
				} else {
					timedResult = Eval(list[1], en)
				}
				if TracePrint {
					d := time.Since(start).String()
					if len(list) > 2 {
						fmt.Println("trace", d, String(Eval(list[2], en)))
					} else {
						fmt.Println("trace", d)
					}
				}
				return timedResult
			case "if":
				i := 1
				for i+1 < len(list) {
					if Eval(list[i], en).Bool() {
						expression = list[i+1]
						goto restart
					}
					i += 2
				}
				if i < len(list) {
					expression = list[i]
					goto restart
				}
				return NewNil()
			case "and":
				for idx, x := range list {
					if idx > 0 && !Eval(x, en).Bool() {
						return NewBool(false)
					}
				}
				return NewBool(true)
			case "or":
				for idx, x := range list {
					if idx > 0 && Eval(x, en).Bool() {
						return NewBool(true)
					}
				}
				return NewBool(false)
			case "coalesce":
				for i := 1; i < len(list); i++ {
					v := Eval(list[i], en)
					if i == len(list)-1 || v.Bool() {
						return v
					}
				}
				return NewNil()
			case "coalesceNil":
				for i, x := range list {
					v := Eval(x, en)
					if i > 0 && !v.IsNil() {
						return v
					}
				}
				return NewNil()
			case "match":
				val := Eval(list[1], en)
				i := 2
				en2 := Env{Vars: make(Vars), VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: true}
				for i < len(list)-1 {
					if match(val, list[i], &en2) {
						en = &en2
						expression = list[i+1]
						goto restart
					}
					i += 2
				}
				if i < len(list) {
					expression = list[i]
					goto restart
				}
				return NewNil()
			case "define", "set":
				val := Eval(list[2], en)
				target := en
				for target != nil && target.Nodefine {
					target = target.Outer
				}
				if target == nil {
					target = &Globalenv
				}
				target.Vars[mustSymbol(list[1])] = val
				return val
			case "setN":
				val := Eval(list[2], en)
				idx := mustNthLocalVar(list[1])
				en.VarsNumbered[int(idx)] = val
				return val
			case "parser":
				if len(list) > 3 {
					return NewAny(NewParser(list[1], list[2], list[3], en, true))
				} else if len(list) > 2 {
					return NewAny(NewParser(list[1], list[2], NewNil(), en, true))
				}
				return NewAny(NewParser(list[1], NewNil(), NewNil(), en, false))
			case "lambda":
				params := list[1]
				if params.IsSourceInfo() {
					params = params.SourceInfo().value
				} else if auxTag(params.aux) == tagAny {
					if si, ok := params.Any().(SourceInfo); ok {
						params = si.value
					}
				}
				numVars := 0
				if len(list) > 3 {
					numVars = int(list[3].Int())
				}
				return NewProcStruct(Proc{Params: params, Body: list[2], En: en, NumVars: numVars})
			case "begin":
				en2 := &Env{Vars: make(Vars), VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: false}
				for _, form := range list[1 : len(list)-1] {
					Eval(form, en2)
				}
				expression = list[len(list)-1]
				en = en2
				goto restart
			case "!begin":
				for _, form := range list[1 : len(list)-1] {
					Eval(form, en)
				}
				expression = list[len(list)-1]
				goto restart
			case "parallel":
				// execute all childs parallely, return nil after finish
				childExprs := list[1:]
				if len(childExprs) == 0 {
					return NewNil()
				}
				errs := make(chan any, len(childExprs))
				for _, expr := range childExprs {
					expr := expr
					gls.Go(func(e Scmer) func() {
						return func() {
							defer func() {
								if r := recover(); r != nil {
									errs <- r
								} else {
									errs <- nil
								}
							}()
							Eval(e, en)
						}
					}(expr))
				}
				for range childExprs {
					if err := <-errs; err != nil {
						panic(err)
					}
				}
				return NewNil()
			default:
				goto to_apply
			}
			return
		}
	to_apply:
		// apply
		operands := list[1:]
		procedure := Eval(list[0], en)
		// Native funcs
		if auxTag(procedure.aux) == tagFunc {
			if auxVal(procedure.aux) == funcKindWithEnv {
				args := make([]Scmer, len(operands))
				for i, x := range operands {
					args[i] = Eval(x, en)
				}
				return procedure.EnvFunc()(en, args...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return procedure.Func()(args...)
		}
		// Lambdas (procs)
		if auxTag(procedure.aux) == tagProc {
			en, expression = prepareProcCall(procedure.Proc(), operands, en)
			goto restart
		}
		// Associative list
		if procedure.IsSlice() {
			p := procedure.Slice()
			if operands[0].IsNthLocalVar() { // optimized indexed access
				return p[int(operands[0].NthLocalVar())]
			}
			arg := Eval(operands[0], en)
			i := 0
			for i < len(p)-1 {
				if Equal(arg, p[i]) {
					return p[i+1]
				}
				i += 2
			}
			if i < len(p) {
				return p[i]
			}
			return NewNil()
		}
		// Parser or FastDict
		if auxTag(procedure.aux) == tagAny {
			if parser, ok := procedure.Any().(*ScmParser); ok {
				if len(operands) == 0 {
					return NewNil()
				}
				return parser.Execute(String(Eval(operands[0], en)), en)
			}
			if fd, ok := procedure.Any().(*FastDict); ok {
				arg := Eval(operands[0], en)
				if v, ok := fd.Get(arg); ok {
					return v
				}
				if ln := len(fd.Pairs); ln%2 == 1 && ln > 0 {
					return fd.Pairs[ln-1]
				}
				return NewNil()
			}
		}
		panic("Unknown function: " + fmt.Sprint(list[0]))
	default:
		panic("Unknown expression type - EVAL " + fmt.Sprint(expression))
	}
	return
}

func prepareProcCall(p *Proc, operands []Scmer, caller *Env) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := *p
	env := &Env{Vars: make(Vars), VarsNumbered: make([]Scmer, proc.NumVars), Outer: proc.En, Nodefine: false}
	switch auxTag(proc.Params.aux) {
	case tagSlice:
		params := proc.Params.Slice()
		if len(params) < len(operands) {
			panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(operands)))
		}
		if proc.NumVars > 0 {
			for i := range params {
				if i < len(operands) {
					env.VarsNumbered[i] = Eval(operands[i], caller)
				}
			}
		} else {
			for i, param := range params {
				if !param.SymbolEquals("_") {
					if i < len(operands) {
						env.Vars[mustSymbol(param)] = Eval(operands[i], caller)
					} else {
						env.Vars[mustSymbol(param)] = NewNil()
					}
				}
			}
		}
	case tagSymbol:
		args := make([]Scmer, len(operands))
		for i, operand := range operands {
			args[i] = Eval(operand, caller)
		}
		argsList := NewSlice(args)
		if proc.NumVars > 0 {
			env.VarsNumbered[0] = argsList
		} else {
			env.Vars[mustSymbol(proc.Params)] = argsList
		}
	case tagNil:
		// no arguments to bind
	default:
		panic("proc parameters must be list, symbol, or nil")
	}
	return env, proc.Body
}

func prepareProcCallWithArgs(p *Proc, args []Scmer) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := *p
	env := &Env{Vars: make(Vars), VarsNumbered: make([]Scmer, proc.NumVars), Outer: proc.En, Nodefine: false}
	switch auxTag(proc.Params.aux) {
	case tagSlice:
		params := proc.Params.Slice()
		if proc.NumVars > 0 {
			for i := range params {
				if i < len(args) {
					env.VarsNumbered[i] = args[i]
				}
			}
		} else {
			for i, param := range params {
				if !param.SymbolEquals("_") {
					if i < len(args) {
						env.Vars[mustSymbol(param)] = args[i]
					} else {
						env.Vars[mustSymbol(param)] = NewNil()
					}
				}
			}
		}
	case tagSymbol:
		argsList := NewSlice(args)
		if proc.NumVars > 0 {
			env.VarsNumbered[0] = argsList
		} else {
			env.Vars[mustSymbol(proc.Params)] = argsList
		}
	case tagNil:
		// no arguments to bind
	default:
		panic("proc parameters must be list, symbol, or nil")
	}
	return env, proc.Body
}

func ApplyAssoc(procedure Scmer, args []Scmer) (value Scmer) {
	var proc *Proc
	if procedure.IsProc() {
		proc = procedure.Proc()
	} else if p, ok := procedure.Any().(*Proc); ok {
		proc = p
	} else if pv, ok := procedure.Any().(Proc); ok {
		cp := pv
		proc = &cp
	} else {
		panic("apply_assoc cannot run on non-lambdas")
	}
	if proc == nil {
		panic("apply_assoc cannot run on nil lambdas")
	}
	if auxTag(proc.Params.aux) == tagSlice {
		params := proc.Params.Slice()
		newParams := make([]Scmer, len(params))
		for i, sym := range params {
			symName := mustSymbol(sym)
			for j := 0; j < len(args); j += 2 {
				if args[j].String() == string(symName) {
					newParams[i] = args[j+1]
				}
			}
		}
		return Apply(procedure, newParams...)
	}
	panic("apply_assoc cannot run on non-list parameters")
}

// helper function; Eval uses a code duplicate to get the tail recursion done right
func Apply(procedure Scmer, args ...Scmer) (value Scmer) {
	return ApplyEx(procedure, args, &Globalenv)
}
func ApplyEx(procedure Scmer, args []Scmer, en *Env) (value Scmer) {
	// Native funcs
	if auxTag(procedure.aux) == tagFunc {
		if auxVal(procedure.aux) == funcKindWithEnv {
			return procedure.EnvFunc()(en, args...)
		}
		return procedure.Func()(args...)
	}
	// Lambdas
	if auxTag(procedure.aux) == tagProc {
		env, body := prepareProcCallWithArgs(procedure.Proc(), args)
		return Eval(body, env)
	}
	// Assoc list
	if procedure.IsSlice() {
		p := procedure.Slice()
		if idx, ok := args[0].Any().(NthLocalVar); ok {
			return p[int(idx)]
		}
		i := 0
		for i < len(p)-1 {
			if Equal(args[0], p[i]) {
				return p[i+1]
			}
			i += 2
		}
		if i < len(p) {
			return p[i]
		}
		return NewNil()
	}
	// Parser and FastDict via tagAny
	if auxTag(procedure.aux) == tagAny {
		if parser, ok := procedure.Any().(*ScmParser); ok {
			if len(args) == 0 {
				return NewNil()
			}
			return parser.Execute(String(args[0]), en)
		}
		if fd, ok := procedure.Any().(*FastDict); ok {
			if v, ok := fd.Get(args[0]); ok {
				return v
			}
			if ln := len(fd.Pairs); ln%2 == 1 && ln > 0 {
				return fd.Pairs[ln-1]
			}
			return NewNil()
		}
	}
	panic("Unknown function")
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
	Vars         Vars
	VarsNumbered []Scmer // <- for the optimizer
	Outer        *Env
	Nodefine     bool // define will write to Outer
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
	return NewSlice(a)
}
func isList(v Scmer) bool {
	if auxTag(v.aux) == tagFunc {
		return reflect.ValueOf(v.Func()).Pointer() == reflect.ValueOf(List).Pointer()
	}
	if auxTag(v.aux) == tagAny {
		if fn, ok := v.Any().(func(...Scmer) Scmer); ok {
			return reflect.ValueOf(fn).Pointer() == reflect.ValueOf(List).Pointer()
		}
	}
	return false
}
func init() {
	Globalenv = Env{
		Vars{ //aka an incomplete set of compiled-in functions
			Symbol("true"):  NewBool(true),
			Symbol("false"): NewBool(false),

			// basic
			Symbol("list"): NewFunc(List),
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
		}, "symbol", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"eval", "executes the given scheme program in the current environment",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters"},
		}, "any", nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"size", "compute the memory size of a value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to examine"},
		}, "int", func(a ...Scmer) Scmer {
			return NewInt(int64(ComputeSize(a[0])))
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"optimize", "optimize the given scheme program",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters"},
		}, "any", func(a ...Scmer) Scmer {
			//fmt.Println("optimize", SerializeToString(a[0], &Globalenv), " -> ", SerializeToString(Optimize(a[0], &Globalenv), &Globalenv))
			return Optimize(a[0], &Globalenv)
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"time", "measures the time it takes to compute the first argument",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "any", "code to execute"},
			DeclarationParameter{"label", "string", "label to print in the log or trace"},
		}, "any", nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"if", "checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition...", "any", "condition to evaluate"},
			DeclarationParameter{"true-branch...", "returntype", "code to evaluate if condition is true"},
			DeclarationParameter{"false-branch", "any", "code to evaluate if condition is false"},
		}, "returntype", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"and", "returns true if all conditions evaluate to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "bool", "condition to evaluate"},
		}, "bool", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"or", "returns true if at least one condition evaluates to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "any", "condition to evaluate"},
		}, "bool", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"coalesce", "returns the first value that has a non-zero value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine"},
		}, "returntype", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"coalesceNil", "returns the first value that has a non-nil value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine"},
		}, "returntype", nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"define", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set"},
			DeclarationParameter{"value", "returntype", "value to set the variable to"},
		}, "bool", nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"set", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set"},
			DeclarationParameter{"value", "returntype", "value to set the variable to"},
		}, "bool", nil, false,
	})

	// basic
	Declare(&Globalenv, &Declaration{
		"error", "halts the whole execution thread and throws an error message",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value or message to throw"},
		}, "string",
		func(a ...Scmer) Scmer {
			if len(a) == 1 {
				panic(a[0])
			} else {
				var b strings.Builder
				for _, v := range a {
					b.WriteString(String(v))
				}
				panic(b.String())
			}
		}, false,
	})
	Declare(&Globalenv, &Declaration{
		"try", "tries to execute a function and returns its result. In case of a failure, the error is fed to the second function and its result value will be used",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"func", "func", "function with no parameters that will be called"},
			DeclarationParameter{"errorhandler", "func", "function that takes the error as parameter"},
		}, "any",
		func(a ...Scmer) (result Scmer) {
			defer func() {
				err := recover()
				if err != nil {
					result = Apply(a[1], FromAny(err))
				}
			}()
			result = Apply(a[0])
			return
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"apply", "runs the function with its arguments",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute"},
			DeclarationParameter{"arguments", "list", "list of arguments to apply"},
		}, "any",
		func(a ...Scmer) Scmer {
			return Apply(a[0], asSlice(a[1], "apply")...)
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"apply_assoc", "runs the function with its arguments but arguments is a assoc list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute (must be a lambda)"},
			DeclarationParameter{"arguments", "list", "assoc list of arguments to apply"},
		}, "symbol",
		func(a ...Scmer) Scmer {
			return ApplyAssoc(a[0], asSlice(a[1], "apply_assoc"))
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"symbol", "returns a symbol built from that string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string value that will be converted into a symbol"},
		}, "symbol",
		func(a ...Scmer) Scmer {
			return NewSymbol(String(a[0]))
		}, false,
	})
	Declare(&Globalenv, &Declaration{
		"list", "returns a list containing the parameters as alements",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value for the list"},
		}, "list",
		nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"for", "Sequential loop over a list state; applies a condition and step function and returns the final state list.\nUse only when iterations have strong data dependencies and must run sequentially.\n\nExamples:\n- Count to 10: (for '(0) (lambda (x) (< x 10)) (lambda (x) (list (+ x 1))))  => '(10)\n- Sum 0..9:   (for '(0 0) (lambda (x sum) (< x 10)) (lambda (x sum) (list (+ x 1) (+ sum x)))) => '(10 45)",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"init", "list", "initial state as a list"},
			DeclarationParameter{"condition", "func", "func that receives the current state as parameters and must return true if the loop shall be continued"},
			DeclarationParameter{"step", "func", "step func that returns the next state as a list"},
		}, "list",
		func(a ...Scmer) Scmer {
			state := append([]Scmer{}, asSlice(a[0], "for init")...)
			cond := OptimizeProcToSerialFunction(a[1])
			next := OptimizeProcToSerialFunction(a[2])
			for ToBool(cond(state...)) {
				v := next(state...)
				if v.IsNil() {
					state = []Scmer{}
					continue
				}
				state = append([]Scmer{}, asSlice(v, "for step")...)
			}
			return NewSlice(state)
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"string", "converts the given value into string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "any value"},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(String(a[0]))
		}, true,
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
		nil, true,
	})
	Declare(&Globalenv, &Declaration{
		"lambda", "returns a function (func) constructed from the given code",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"parameters", "symbol|list|nil", "if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol"},
			DeclarationParameter{"code", "any", "value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above"},
			DeclarationParameter{"numvars", "number", "number of unnamed variables that can be accessed via (var 0) (var 1) etc."},
		}, "func", // TODO: func(...)->returntype as soon as function types are implemented
		nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"begin", "creates a own variable scope, evaluates all sub expressions and returns the result of the last one",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate"},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "any", // TODO: returntype as soon as repeat is implemented
		nil, false,
	})
	Declare(&Globalenv, &Declaration{
		"parallel", "executes all parameters in parallel and returns nil if they are finished",
		1, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate in parallel"},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "any", // TODO: returntype as soon as repeat is implemented
		nil, false,
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
		func(a ...Scmer) Scmer {
			return NewSourceInfo(SourceInfo{
				String(a[0]),
				ToInt(a[1]),
				ToInt(a[2]),
				a[3],
			})
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"scheme", "parses a scheme expression into a list",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "string", "Scheme code"},
			DeclarationParameter{"filename", "string", "optional filename"},
		}, "any",
		func(a ...Scmer) Scmer {
			filename := "eval"
			if len(a) > 1 {
				filename = String(a[1])
			}
			return Read(filename, String(a[0]))
		}, true,
	})
	Declare(&Globalenv, &Declaration{
		"serialize", "serializes a piece of code into a (hopefully) reparsable string; you shall be able to send that code over network and reparse with (scheme)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "Scheme code"},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(SerializeToString(a[0], &Globalenv))
		}, false,
	})

	init_alu()
	init_strings()
	init_streams()
	init_list()
	init_date()
	init_vector()
	init_parser()
	init_sync()

	RunJitTest()
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

type Symbol string //Symbols are represented by strings
//Numbers by float64 (but no extra type)

type Sizable interface {
	ComputeSize() uint
}

func ComputeSize(v Scmer) uint {
	base := scmerStructOverhead
	switch auxTag(v.aux) {
	case tagNil:
		return base
	case tagBool, tagInt, tagFloat:
		return base
	case tagFunc:
		return base + goAllocOverhead
	case tagString, tagSymbol:
		ln := uint(auxVal(v.aux))
		if ln == 0 {
			return base
		}
		return base + goAllocOverhead + align8(ln)
	case tagSlice:
		slice := v.Slice()
		sz := base
		if len(slice) == 0 {
			return sz
		}
		sz += goAllocOverhead
		for _, vi := range slice {
			sz += ComputeSize(vi)
		}
		return sz
	case tagVector:
		vec := v.Vector()
		sz := base
		if len(vec) == 0 {
			return sz
		}
		data := uint(len(vec)) * 8
		sz += goAllocOverhead + align8(data)
		return sz
	case tagSourceInfo:
		si := v.SourceInfo()
		sz := base + goAllocOverhead
		if si.source != "" {
			sz += align8(uint(len(si.source)))
		}
		sz += ComputeSize(si.value)
		return sz
	case tagAny:
		payload := v.Any()
		return base + goAllocOverhead + computeGoPayload(payload)
	default:
		if auxTag(v.aux) >= 100 {
			return base
		}
		fmt.Println(fmt.Sprintf("warning: unknown tag %d", auxTag(v.aux)))
		return base
	}
}

func computeGoPayload(val any) uint {
	switch v := val.(type) {
	case nil:
		return 0
	case Scmer:
		return ComputeSize(v)
	case *Scmer:
		if v == nil {
			return 0
		}
		return ComputeSize(*v)
	case []Scmer:
		if len(v) == 0 {
			return 0
		}
		sz := goAllocOverhead
		for _, elem := range v {
			sz += ComputeSize(elem)
		}
		return sz
	case SourceInfo:
		sz := goAllocOverhead
		if v.source != "" {
			sz += align8(uint(len(v.source)))
		}
		sz += ComputeSize(v.value)
		return sz
	case *SourceInfo:
		if v == nil {
			return 0
		}
		sz := goAllocOverhead
		if v.source != "" {
			sz += align8(uint(len(v.source)))
		}
		sz += ComputeSize(v.value)
		return sz
	case [][]Scmer:
		if len(v) == 0 {
			return 0
		}
		sz := goAllocOverhead
		for _, row := range v {
			if len(row) == 0 {
				continue
			}
			sz += goAllocOverhead
			for _, elem := range row {
				sz += ComputeSize(elem)
			}
		}
		return sz
	case []float64:
		if len(v) == 0 {
			return 0
		}
		return goAllocOverhead + align8(uint(len(v))*8)
	case []byte:
		if len(v) == 0 {
			return 0
		}
		return goAllocOverhead + align8(uint(len(v)))
	case string:
		if len(v) == 0 {
			return 0
		}
		return goAllocOverhead + align8(uint(len(v)))
	case Symbol:
		if len(v) == 0 {
			return 0
		}
		return goAllocOverhead + align8(uint(len(v)))
	case Sizable:
		return v.ComputeSize()
	case *Sizable:
		if v == nil {
			return 0
		}
		return (*v).ComputeSize()
	case map[string]Scmer:
		sz := goAllocOverhead
		for k, val := range v {
			if len(k) > 0 {
				sz += goAllocOverhead + align8(uint(len(k)))
			}
			sz += ComputeSize(val)
		}
		return sz
	case map[Scmer]Scmer:
		sz := goAllocOverhead
		for k, val := range v {
			sz += ComputeSize(k)
			sz += ComputeSize(val)
		}
		return sz
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return 0
	default:
		fmt.Println(fmt.Sprintf("warning: unknown any payload %T", v))
		return 0
	}
}

func align8(n uint) uint {
	if n == 0 {
		return 0
	}
	if r := n & 7; r != 0 {
		return n + (8 - r)
	}
	return n
}
