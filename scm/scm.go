/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
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
	if v.GetTag() == tagSymbol {
		return v.String(), true
	}
	if v.GetTag() == tagAny {
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
	if v.GetTag() == tagNthLocalVar {
		return v.NthLocalVar()
	}
	if v.GetTag() == tagAny {
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
	switch expression.GetTag() {
	case tagSourceInfo:
		return evalWithSourceInfo(*expression.SourceInfo(), en)
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagVector, tagFastDict, tagParser, tagAny, tagFunc, tagProc, tagJIT:
		// literals
		return expression
	case tagSymbol:
		// get named variable
		return en.FindRead(mustSymbol(expression)).Vars[mustSymbol(expression)]
	case tagNthLocalVar:
		// get numbered variable
		return en.VarsNumbered[expression.NthLocalVar()]
	case tagSlice:
		// slice -> function call
		list := expression.Slice()
		if len(list) == 0 {
			return expression
		}
		if list[0].GetTag() == tagSymbol {
			headSym := list[0].Symbol()
			switch string(headSym) {
			case "outer":
				if en.Outer == nil {
					return NewNil()
				}
				if list[1].IsSymbol() {
					sym := list[1].Symbol()
					if env := en.Outer.FindRead(sym); env != nil {
						if val, ok := env.Vars[sym]; ok {
							return val
						}
					}
					symStr := string(sym)
					if strings.Contains(symStr, ".") && !strings.Contains(symStr, ":") {
						suffix := ":" + symStr
						for env := en.Outer; env != nil; env = env.Outer {
							for key, val := range env.Vars {
								if strings.HasSuffix(string(key), suffix) {
									return val
								}
							}
						}
					}
				}
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
				if target.Vars == nil {
					target.Vars = make(Vars)
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
					return NewScmParser(NewParser(list[1], list[2], list[3], en, true))
				} else if len(list) > 2 {
					return NewScmParser(NewParser(list[1], list[2], NewNil(), en, true))
				}
				return NewScmParser(NewParser(list[1], NewNil(), NewNil(), en, false))
			case "lambda":
				params := list[1]
				if params.IsSourceInfo() {
					params = params.SourceInfo().value
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
			case "!list":
				// Stack-allocated list: (!list NthLocalVar(start) count expr...)
				// Evaluates exprs into VarsNumbered[start..start+count] and returns
				// a slice view. The slice MUST NOT escape the current lambda frame.
				start := int(list[1].NthLocalVar())
				count := int(ToInt(list[2]))
				for i := 0; i < count && i+3 < len(list); i++ {
					en.VarsNumbered[start+i] = Eval(list[i+3], en)
				}
				return NewSlice(en.VarsNumbered[start : start+count])
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
			// control flow will never be here
		}
	to_apply:
		// apply
		operands := list[1:]
		procedure := Eval(list[0], en) // resolve the head (compute lambdas or lookup function from symbol)
		switch procedure.GetTag() {
		case tagFunc:
			// Native funcs
			fn := procedure.Func()
			if n := len(operands); n <= 4 {
				var buf [4]Scmer
				for i := 0; i < n; i++ {
					buf[i] = Eval(operands[i], en)
				}
				return fn(buf[:n]...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return fn(args...)
		case tagFuncEnv:
			// Native funcs with env
			fn := procedure.FuncEnv()
			if n := len(operands); n <= 4 {
				var buf [4]Scmer
				for i := 0; i < n; i++ {
					buf[i] = Eval(operands[i], en)
				}
				return fn(en, buf[:n]...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return fn(en, args...)
		case tagProc:
			// Lambdas (procs)
			en, expression = prepareProcCall(procedure.Proc(), operands, en)
			goto restart
		case tagSlice:
			// Associative list
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
		case tagParser:
			// Parser or FastDict
			if len(operands) == 0 {
				return NewNil()
			}
			return procedure.Parser().Execute(String(Eval(operands[0], en)), en)
		case tagFastDict:
			fd := procedure.FastDict()
			arg := Eval(operands[0], en)
			if fd != nil {
				if v, ok := fd.Get(arg); ok {
					return v
				}
				if ln := len(fd.Pairs); ln%2 == 1 && ln > 0 {
					return fd.Pairs[ln-1]
				}
			}
			return NewNil()
		case tagJIT:
			jep := procedure.JIT()
			if jep.Native == nil {
				env, body := prepareProcCall(&jep.Proc, operands, en)
				return Eval(body, env)
			}
			switch len(operands) {
			case 0:
				return jep.Native()
			case 1:
				a0 := Eval(operands[0], en)
				return jep.Native(a0)
			case 2:
				a0 := Eval(operands[0], en)
				a1 := Eval(operands[1], en)
				return jep.Native(a0, a1)
			case 3:
				a0 := Eval(operands[0], en)
				a1 := Eval(operands[1], en)
				a2 := Eval(operands[2], en)
				return jep.Native(a0, a1, a2)
			case 4:
				a0 := Eval(operands[0], en)
				a1 := Eval(operands[1], en)
				a2 := Eval(operands[2], en)
				a3 := Eval(operands[3], en)
				return jep.Native(a0, a1, a2, a3)
			default:
				args := make([]Scmer, len(operands))
				for i, x := range operands {
					args[i] = Eval(x, en)
				}
				return jep.Native(args...)
			}
		default:
			panic("Unknown function: " + list[0].String())
		}
	default:
		panic("Unknown expression type - EVAL " + expression.String())
	}
	return
}

func prepareProcCall(p *Proc, operands []Scmer, caller *Env) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := *p
	var vars Vars
	if proc.NumVars == 0 {
		vars = make(Vars)
	}
	env := &Env{Vars: vars, VarsNumbered: make([]Scmer, proc.NumVars), Outer: proc.En, Nodefine: false}
	switch proc.Params.GetTag() {
	case tagSlice:
		params := proc.Params.Slice()
		if len(params) < len(operands) {
			panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(operands)))
		}
		if proc.NumVars > 0 {
			for i := range params {
				if i < len(operands) && i < proc.NumVars {
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
	var vars Vars
	if proc.NumVars == 0 {
		vars = make(Vars)
	}
	env := &Env{Vars: vars, VarsNumbered: make([]Scmer, proc.NumVars), Outer: proc.En, Nodefine: false}
	switch proc.Params.GetTag() {
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
	} else {
		panic("apply_assoc cannot run on non-lambdas")
	}
	if proc == nil {
		panic("apply_assoc cannot run on nil lambdas")
	}
	if proc.Params.GetTag() == tagSlice {
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
	switch procedure.GetTag() {
	case tagFuncEnv:
		return procedure.FuncEnv()(en, args...)
	case tagFunc:
		return procedure.Func()(args...)
	// Lambdas
	case tagProc:
		env, body := prepareProcCallWithArgs(procedure.Proc(), args)
		return Eval(body, env)
	// Assoc list
	case tagSlice:
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
	// Parser and FastDict via tagAny
	case tagParser:
		return procedure.Parser().Execute(String(args[0]), en)
	case tagFastDict:
		fd := procedure.FastDict()
		if fd != nil {
			if v, ok := fd.Get(args[0]); ok {
				return v
			}
			if ln := len(fd.Pairs); ln%2 == 1 && ln > 0 {
				return fd.Pairs[ln-1]
			}
		}
		return NewNil()
	case tagJIT:
		jep := procedure.JIT()
		if jep.Native != nil {
			switch len(args) {
			case 0:
				return jep.Native()
			case 1:
				return jep.Native(args[0])
			case 2:
				return jep.Native(args[0], args[1])
			case 3:
				return jep.Native(args[0], args[1], args[2])
			case 4:
				return jep.Native(args[0], args[1], args[2], args[3])
			default:
				return jep.Native(args...)
			}
		}
		env, body := prepareProcCallWithArgs(&jep.Proc, args)
		return Eval(body, env)
	default:
		panic("Unknown function: " + procedure.String())
	}
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
	if v.GetTag() == tagFunc {
		return reflect.ValueOf(v.Func()).Pointer() == reflect.ValueOf(List).Pointer()
	}
	if v.GetTag() == tagAny {
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
			DeclarationParameter{"symbol", "symbol", "symbol to quote", nil},
		}, "symbol", nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"eval", "executes the given scheme program in the current environment",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters", nil},
		}, "any", nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"size", "compute the memory size of a value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to examine", nil},
		}, "int", func(a ...Scmer) Scmer {
			return NewInt(int64(ComputeSize(a[0])))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ComputeSize arg0)")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(ComputeSize), []JITValueDesc{d0}, 1)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d1.Imm.Int()))))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r0, d1.Reg)
				d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeInt(result, d2)
			} else {
				ctx.W.EmitMakeInt(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagInt
			return result
			}
			return bbs[0].Render()
		}, /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.Proc */
	})
	Declare(&Globalenv, &Declaration{
		"optimize", "optimize the given scheme program",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters", nil},
		}, "any", func(a ...Scmer) Scmer {
			return Optimize(a[0], &Globalenv)
		}, true, false, nil,
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */
	})
	Declare(&Globalenv, &Declaration{
		"time", "measures the time it takes to compute the first argument",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "any", "code to execute", nil},
			DeclarationParameter{"label", "string", "label to print in the log or trace", nil},
		}, "any", nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"if", "checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition...", "any", "condition to evaluate", nil},
			DeclarationParameter{"true-branch...", "returntype", "code to evaluate if condition is true", nil},
			DeclarationParameter{"false-branch", "any", "code to evaluate if condition is false", nil},
		}, "returntype", nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"and", "returns true if all conditions evaluate to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "bool", "condition to evaluate", nil},
		}, "bool", nil, true, false, &TypeDescriptor{Optimize: optimizeAnd},
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"or", "returns true if at least one condition evaluates to true",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"condition", "any", "condition to evaluate", nil},
		}, "bool", nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"coalesce", "returns the first value that has a non-zero value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine", nil},
		}, "returntype", nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"coalesceNil", "returns the first value that has a non-nil value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value", "returntype", "value to examine", nil},
		}, "returntype", nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"define", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set", nil},
			DeclarationParameter{"value", "returntype", "value to set the variable to", nil},
		}, "bool", nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"set", "defines or sets a variable in the current environment",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"variable", "symbol", "variable to set", nil},
			DeclarationParameter{"value", "returntype", "value to set the variable to", nil},
		}, "bool", nil, false, false, nil,
		nil,
	})

	// basic
	Declare(&Globalenv, &Declaration{
		"error", "halts the whole execution thread and throws an error message",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value or message to throw", nil},
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
		}, false, false, nil,
		nil /* TODO: FieldAddr on non-receiver: &b.addr [#0] */, /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */ /* TODO: FieldAddr on non-receiver: &b.addr [#0] */
	})
	Declare(&Globalenv, &Declaration{
		"try", "tries to execute a function and returns its result. In case of a failure, the error is fed to the second function and its result value will be used",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"func", "func", "function with no parameters that will be called", nil},
			DeclarationParameter{"errorhandler", "func", "function that takes the error as parameter", nil},
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
		}, true, false, nil,
		nil /* TODO: MakeClosure with 2 bindings */, /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */ /* TODO: MakeClosure with 2 bindings */
	})
	Declare(&Globalenv, &Declaration{
		"apply", "runs the function with its arguments",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute", nil},
			DeclarationParameter{"arguments", "list", "list of arguments to apply", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			return Apply(a[0], asSlice(a[1], "apply")...)
		}, true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"apply_assoc", "runs the function with its arguments but arguments is a assoc list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"function", "func", "function to execute (must be a lambda)", nil},
			DeclarationParameter{"arguments", "list", "assoc list of arguments to apply", nil},
		}, "symbol",
		func(a ...Scmer) Scmer {
			return ApplyAssoc(a[0], asSlice(a[1], "apply_assoc"))
		}, true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"symbol", "returns a symbol built from that string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string value that will be converted into a symbol", nil},
		}, "symbol",
		func(a ...Scmer) Scmer {
			return NewSymbol(String(a[0]))
		}, false, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(NewSymbol), []JITValueDesc{d1}, 2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d2, &result)
				result.Type = d2.Type
			} else {
				switch d2.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d2)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d2)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d2)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
			}
			return bbs[0].Render()
		}, /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */ /* TODO: FieldAddr on non-receiver: &t7.ptr [#0] */
	})
	Declare(&Globalenv, &Declaration{
		"list", "returns a list containing the parameters as alements",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value for the list", nil},
		}, "list",
		nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"for", "Sequential loop over a list state; applies a condition and step function and returns the final state list.\nUse only when iterations have strong data dependencies and must run sequentially.\n\nExamples:\n- Count to 10: (for '(0) (lambda (x) (< x 10)) (lambda (x) (list (+ x 1))))  => '(10)\n- Sum 0..9:   (for '(0 0) (lambda (x sum) (< x 10)) (lambda (x sum) (list (+ x 1) (+ sum x)))) => '(10 45)",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"init", "list", "initial state as a list", nil},
			DeclarationParameter{"condition", "func", "func that receives the current state as parameters and must return true if the loop shall be continued", nil},
			DeclarationParameter{"step", "func", "step func that returns the next state as a list", nil},
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
		}, true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("for_mut")},
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
	Declare(&Globalenv, &Declaration{
		"for_mut", "in-place for loop (optimizer-only, skips defensive state copy)",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"init", "list", "owned initial state", nil},
			DeclarationParameter{"condition", "func", "func(state...) -> bool", nil},
			DeclarationParameter{"step", "func", "step func returning next state as list", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			state := asSlice(a[0], "for_mut init")
			cond := OptimizeProcToSerialFunction(a[1])
			next := OptimizeProcToSerialFunction(a[2])
			for ToBool(cond(state...)) {
				v := next(state...)
				if v.IsNil() {
					state = []Scmer{}
					continue
				}
				state = asSlice(v, "for_mut step")
			}
			return NewSlice(state)
		}, true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"string", "converts the given value into string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "any value", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(String(a[0]))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			if d0.Loc != LocImm && d0.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d0}, 2)
			ctx.FreeDesc(&d0)
			ctx.W.ResolveFixups()
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d1}, 2)
			if result.Loc == LocAny { return d2 }
			ctx.EmitMovPairToResult(&d2, &result)
			result.Type = tagString
			return result
			}
			return bbs[0].Render()
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
			DeclarationParameter{"value", "any", "value to evaluate", nil},
			DeclarationParameter{"pattern...", "any", "pattern", nil},
			DeclarationParameter{"result...", "returntype", "result value when the pattern matches; this code can use the variables matched in the pattern", nil},
			DeclarationParameter{"default", "any", "(optional) value that is returned when no pattern matches", nil}, /* TODO: turn to returntype as soon as pattern+result are properly repeaded in Validate */
		}, "any", // TODO: returntype as soon as repead validate is implemented */
		nil, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"lambda", "returns a function (func) constructed from the given code",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"parameters", "symbol|list|nil", "if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol", nil},
			DeclarationParameter{"code", "any", "value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above", nil},
			DeclarationParameter{"numvars", "number", "number of unnamed variables that can be accessed via (var 0) (var 1) etc.", nil},
		}, "func", // TODO: func(...)->returntype as soon as function types are implemented
		nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"begin", "creates a own variable scope, evaluates all sub expressions and returns the result of the last one",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate", nil},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "any", // TODO: returntype as soon as repeat is implemented
		nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"parallel", "executes all parameters in parallel and returns nil if they are finished",
		1, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"expression...", "any", "expressions to evaluate in parallel", nil},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "any", // TODO: returntype as soon as repeat is implemented
		nil, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"source", "annotates the node with filename and line information for better backtraces",
		4, 4,
		[]DeclarationParameter{
			DeclarationParameter{"filename", "string", "Filename of the code", nil},
			DeclarationParameter{"line", "number", "Line of the code", nil},
			DeclarationParameter{"column", "number", "Column of the code", nil},
			DeclarationParameter{"code", "returntype", "code", nil},
			/* TODO: lastexpression = returntype as soon as expression... is properly repeated */
		}, "returntype",
		func(a ...Scmer) Scmer {
			return NewSourceInfo(SourceInfo{
				String(a[0]),
				ToInt(a[1]),
				ToInt(a[2]),
				a[3],
			})
		}, true, false, nil,
		nil /* TODO: FieldAddr on non-receiver: &t0.source [#0] */, /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */ /* TODO: FieldAddr on non-receiver: &t0.source [#0] */
	})
	Declare(&Globalenv, &Declaration{
		"scheme", "parses a scheme expression into a list",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "string", "Scheme code", nil},
			DeclarationParameter{"filename", "string", "optional filename", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			filename := "eval"
			if len(a) > 1 {
				filename = String(a[1])
			}
			return Read(filename, String(a[0]))
		}, true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d0)
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() > 1)}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d0.Reg, 1)
				ctx.W.EmitSetcc(r1, CcG)
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d1)
			}
			ctx.FreeDesc(&d0)
			d2 := d1
			ctx.EnsureDesc(&d2)
			if d2.Loc != LocImm && d2.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d2.Loc == LocImm {
				if d2.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl4)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl5)
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl3)
			d3 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 := args[0]
			if d4.Loc != LocImm && d4.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d5 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d4}, 2)
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d3.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d3)
				} else if d3.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Read arg0)")
			}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			if d5.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d5.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d5)
				} else if d5.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d5)
				} else if d5.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d5)
				} else if d5.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d5.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d5 = tmpPair
			} else if d5.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
				switch d5.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d5)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d5)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d5)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d5)
				d5 = tmpPair
			}
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Read arg1)")
			}
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(Read), []JITValueDesc{d3, d5}, 2)
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d6)
			if d6.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d6, &result)
				result.Type = d6.Type
			} else {
				switch d6.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d6)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d6)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d6)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d6, &result)
					result.Type = d6.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl2)
			d3 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d7 := args[1]
			if d7.Loc != LocImm && d7.Type == JITTypeUnknown {
				panic("jit: Scmer.String on unknown dynamic type")
			}
			d8 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d7}, 2)
			ctx.FreeDesc(&d7)
			d9 := d8
			if d9.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d9)
			if d9.Loc == LocRegPair || d9.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d9, 0)
			} else {
				ctx.EmitStoreToStack(d9, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(16))
			ctx.W.EmitAddRSP32(int32(16))
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */ /* TODO: IndexAddr on non-parameter: &t0[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"serialize", "serializes a piece of code into a (hopefully) reparsable string; you shall be able to send that code over network and reparse with (scheme)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "Scheme code", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			return NewString(SerializeToString(a[0], &Globalenv))
		}, false, false, nil,
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */
	})

	init_alu()
	init_strings()
	init_streams()
	init_list()
	init_date()
	init_vector()
	init_parser()
	init_sync()
	init_window()
	init_scheduler()
	init_jit()
	InitMetricsDeclarations()
}

/* TODO: quotient, remainder, modulo, gcd, lcm, expt
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
	switch v.GetTag() {
	case tagNil:
		return base
	case tagBool, tagInt, tagFloat, tagDate:
		return base
	case tagFunc, tagFuncEnv:
		return base + goAllocOverhead
	case tagNthLocalVar:
		return base
	case tagProc:
		p := v.Proc()
		if p == nil {
			return base
		}
		// Proc struct: Params(16) + Body(16) + En(8) + NumVars(8) = 48 bytes
		// Params and Body are inline Scmer fields — their slots are covered by
		// the recursive ComputeSize base (same pattern as slice backing array).
		// Only count the non-Scmer fields: *Env(8) + NumVars(8) = 16 bytes.
		return base + goAllocOverhead + 16 + ComputeSize(p.Params) + ComputeSize(p.Body)
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
	case tagFastDict:
		return base + goAllocOverhead + fastDictPayloadSize(v.FastDict())
	case tagSourceInfo:
		si := v.SourceInfo()
		// SourceInfo struct: source(16) + line(8) + col(8) + value(16) = 48 bytes
		// value is an inline Scmer — covered by recursive ComputeSize base.
		// Non-Scmer fields: source header(16) + line(8) + col(8) = 32 bytes.
		sz := base + goAllocOverhead + 32
		if si.source != "" {
			sz += align8(uint(len(si.source)))
		}
		sz += ComputeSize(si.value)
		return sz
	case tagRegex:
		return base + goAllocOverhead
	case tagParser:
		return base + goAllocOverhead
	case tagAny:
		payload := v.Any()
		return base + goAllocOverhead + computeGoPayload(payload)
	case tagJIT:
		jep := v.JIT()
		sz := base + goAllocOverhead
		for range jep.Pages {
			sz += goAllocOverhead + 4096 // JITPage overhead
		}
		sz += ComputeSize(NewProcStruct(jep.Proc))
		return sz
	default:
		if v.GetTag() >= 100 {
			return base
		}
		fmt.Println(fmt.Sprintf("warning: unknown tag %d", v.GetTag()))
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
	case *FastDict:
		return fastDictPayloadSize(v)
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

func fastDictPayloadSize(fd *FastDict) uint {
	if fd == nil {
		return 0
	}
	sz := goAllocOverhead
	if len(fd.Pairs) > 0 {
		sz += goAllocOverhead
		for _, elem := range fd.Pairs {
			sz += ComputeSize(elem)
		}
	}
	if len(fd.index) > 0 {
		sz += goAllocOverhead + uint(len(fd.index))*16
		for _, bucket := range fd.index {
			if len(bucket) == 0 {
				continue
			}
			sz += goAllocOverhead + uint(len(bucket))*8
		}
	}
	return sz
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
