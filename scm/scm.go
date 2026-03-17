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
	"unsafe"
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
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagVector, tagFastDict, tagParser, tagAny, tagFunc, tagFuncEnv, tagProc, tagJIT, tagClosure:
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
			if n := len(operands); n <= 4 {
				var buf [4]Scmer
				for i := 0; i < n; i++ {
					buf[i] = Eval(operands[i], en)
				}
				return callJIT(jep.Native, buf[:n]...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return callJIT(jep.Native, args...)
		case tagClosure:
			fn := *(*func(uint32, ...Scmer) Scmer)(unsafe.Pointer(procedure.ptr))
			id := uint32(auxVal(procedure.aux))
			if n := len(operands); n <= 4 {
				var buf [4]Scmer
				for i := 0; i < n; i++ {
					buf[i] = Eval(operands[i], en)
				}
				return fn(id, buf[:n]...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return fn(id, args...)
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
	case tagClosure:
		fn := *(*func(uint32, ...Scmer) Scmer)(unsafe.Pointer(procedure.ptr))
		id := uint32(auxVal(procedure.aux))
		return fn(id, args...)
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
		return callJIT(procedure.JIT().Native, args...)
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
		func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			argPinned0 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned0 = append(argPinned0, ai.Reg2)
					}
				}
			}
			d1 := args[0]
			d1.ID = 0
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ComputeSize arg0)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(ComputeSize), []JITValueDesc{d1}, 1)
			ctx.BindReg(d2.Reg, &d2)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d2.Imm.Int()))))}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitMovRegReg(r0, d2.Reg)
				d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
				ctx.BindReg(r0, &d3)
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeInt(result, d3)
			} else {
				ctx.EmitMakeInt(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagInt
			return result
			for _, r := range argPinned0 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"optimize", "optimize the given scheme program",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "list with head and optional parameters", nil},
		}, "any", func(a ...Scmer) Scmer {
			return Optimize(a[0], &Globalenv)
		}, true, false, nil,
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */,
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
		nil /* TODO: FieldAddr on non-receiver: &b.addr [#0] */,
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
		nil /* TODO: MakeClosure with 2 bindings */,
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */,
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */,
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
		func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			argPinned0 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned0 = append(argPinned0, ai.Reg2)
					}
				}
			}
			d1 := args[0]
			d1.ID = 0
			d3 := d1
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d3.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			} else if d3.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d2.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d2)
				} else if d2.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d2.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d2 = tmpPair
			} else if d2.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
				switch d2.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d2)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d2)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d2)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d2)
				d2 = tmpPair
			}
			if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
			}
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewSymbol), []JITValueDesc{d2}, 2)
			ctx.BindReg(d4.Reg, &d4)
			ctx.BindReg(d4.Reg2, &d4)
			if d4.Loc == LocImm {
				if result.Loc == LocAny { return d4 }
			}
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			ctx.EnsureDesc(&d4)
			if d4.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = d4.Type
			} else {
				switch d4.Type {
				case tagBool:
					ctx.EmitMakeBool(result, d4)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d4)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d4)
					result.Type = tagFloat
				case tagNil:
					ctx.EmitMakeNil(result)
					result.Type = tagNil
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
			for _, r := range argPinned0 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"list", "returns a list containing the parameters as alements",
		0, 10000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "value for the list", nil},
		}, "list",
		nil, true, false, nil,
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
		nil /* TODO: Slice on non-desc: slice t0[:] */,
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */,
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
		func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			argPinned0 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned0 = append(argPinned0, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned0 = append(argPinned0, ai.Reg2)
					}
				}
			}
			d1 := args[0]
			d1.ID = 0
			d3 := d1
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d3.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d3 = tmpPair
			} else if d3.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
				switch d3.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d3)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d3)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d3)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d3)
				d3 = tmpPair
			} else if d3.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d3 = tmpPair
			}
			if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d3}, 2)
			ctx.FreeDesc(&d1)
			d4 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
			if result.Loc == LocAny { return d4 }
			ctx.EmitMovPairToResult(&d4, &result)
			result.Type = tagString
			return result
			for _, r := range argPinned0 {
				ctx.UnprotectReg(r)
			}
			return result
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
		nil /* TODO: FieldAddr on non-receiver: &t0.source [#0] */,
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
		func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			var d7 JITValueDesc
			_ = d7
			var d10 JITValueDesc
			_ = d10
			var d18 JITValueDesc
			_ = d18
			var d19 JITValueDesc
			_ = d19
			var d20 JITValueDesc
			_ = d20
			var d21 JITValueDesc
			_ = d21
			var d23 JITValueDesc
			_ = d23
			var d24 JITValueDesc
			_ = d24
			var d25 JITValueDesc
			_ = d25
			var d26 JITValueDesc
			_ = d26
			var d27 JITValueDesc
			_ = d27
			var d28 JITValueDesc
			_ = d28
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			phiBase0 := ctx.AllocStack(int32(16))
			d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0)+int32(0)}
			var bbs [3]BBDescriptor
			bbs[2].PhiBase = int32(phiBase0) + int32(0)
			bbs[2].PhiCount = uint16(1)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 0 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() > 1)}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d2.Reg, 1)
				ctx.EmitSetcc(r0, CcG)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d3)
			}
			ctx.FreeDesc(&d2)
			d4 = d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 5)
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
			ps5.OverlayValues[4] = d4
					return bbs[1].RenderPS(ps5)
				}
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, int32(bbs[2].PhiBase)+int32(0))
			ps6 := PhiState{General: ps.General}
			ps6.OverlayValues = make([]JITValueDesc, 5)
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			ps6.PhiValues = make([]JITValueDesc, 1)
			d7 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}
			ps6.PhiValues[0] = d7
				return bbs[2].RenderPS(ps6)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d4.Reg, 0)
			ctx.EmitJcc(CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, int32(bbs[2].PhiBase)+int32(0))
			ctx.EmitJmp(lbl3)
			ps8 := PhiState{General: true}
			ps8.OverlayValues = make([]JITValueDesc, 8)
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			ps8.OverlayValues[7] = d7
			ps9 := PhiState{General: true}
			ps9.OverlayValues = make([]JITValueDesc, 8)
			ps9.OverlayValues[1] = d1
			ps9.OverlayValues[2] = d2
			ps9.OverlayValues[3] = d3
			ps9.OverlayValues[4] = d4
			ps9.OverlayValues[7] = d7
			ps9.PhiValues = make([]JITValueDesc, 1)
			d10 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}
			ps9.PhiValues[0] = d10
			snap11 := d1
			snap12 := d2
			snap13 := d3
			snap14 := d4
			snap15 := d7
			snap16 := d10
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps9)
			}
			ctx.RestoreAllocState(alloc17)
			d1 = snap11
			d2 = snap12
			d3 = snap13
			d4 = snap14
			d7 = snap15
			d10 = snap16
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps8)
			}
			return result
			ctx.FreeDesc(&d3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 0 {
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			ctx.ReclaimUntrackedRegs()
			d18 = args[1]
			d18.ID = 0
			d20 = d18
			ctx.EnsureDesc(&d20)
			if d20.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d20.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d20)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d20)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d20)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d20.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d20 = tmpPair
			} else if d20.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d20.Reg), Reg2: ctx.AllocRegExcept(d20.Reg)}
				switch d20.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d20)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d20)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d20)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d20)
				d20 = tmpPair
			} else if d20.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d20.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d20 = tmpPair
			}
			if d20.Loc != LocRegPair && d20.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d19 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d20}, 2)
			ctx.FreeDesc(&d18)
			ctx.EnsureDesc(&d19)
			if d19.Loc == LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d21 = d19
			if d21.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocRegPair || d21.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
			}
			if d19.Loc == LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
			ps22 := PhiState{General: ps.General}
			ps22.OverlayValues = make([]JITValueDesc, 22)
			ps22.OverlayValues[1] = d1
			ps22.OverlayValues[2] = d2
			ps22.OverlayValues[3] = d3
			ps22.OverlayValues[4] = d4
			ps22.OverlayValues[7] = d7
			ps22.OverlayValues[10] = d10
			ps22.OverlayValues[18] = d18
			ps22.OverlayValues[19] = d19
			ps22.OverlayValues[20] = d20
			ps22.OverlayValues[21] = d21
			ps22.PhiValues = make([]JITValueDesc, 1)
			d23 = d19
			ps22.PhiValues[0] = d23
			if ps22.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps22)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d24 := ps.PhiValues[0]
					ctx.EnsureDesc(&d24)
					ctx.EmitStoreScmerToStack(d24, int32(bbs[2].PhiBase)+int32(0))
				}
				if bbs[2].VisitCount >= 0 {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
				d24 = ps.OverlayValues[24]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d25 = args[0]
			d25.ID = 0
			d27 = d25
			ctx.EnsureDesc(&d27)
			if d27.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				tag := d27.Imm.GetTag()
				switch tag {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d27)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d27)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d27)
				case tagNil:
					ctx.EmitMakeNil(tmpPair)
				default:
					ptrWord, auxWord := d27.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d27 = tmpPair
			} else if d27.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d27.Reg), Reg2: ctx.AllocRegExcept(d27.Reg)}
				switch d27.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d27)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d27)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d27)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&d27)
				d27 = tmpPair
			} else if d27.Loc == LocMem {
				tmpScalar := JITValueDesc{Loc: LocReg, Type: d27.Type, Reg: ctx.AllocReg()}
				scratch := ctx.AllocRegExcept(tmpScalar.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d27.MemPtr))
				ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
				ctx.FreeReg(scratch)
				ctx.BindReg(tmpScalar.Reg, &tmpScalar)
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
				switch tmpScalar.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, tmpScalar)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, tmpScalar)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, tmpScalar)
				default:
					panic("jit: Scmer.String requires Scmer pair receiver")
				}
				ctx.FreeDesc(&tmpScalar)
				d27 = tmpPair
			}
			if d27.Loc != LocRegPair && d27.Loc != LocStackPair {
				panic("jit: Scmer.String receiver not materialized as pair")
			}
			d26 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d27}, 2)
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Read arg0)")
			}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			if d26.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d26.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d26.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d26)
				} else if d26.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d26)
				} else if d26.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d26)
				} else if d26.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d26.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d26 = tmpPair
			} else if d26.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d26.Type, Reg: ctx.AllocRegExcept(d26.Reg), Reg2: ctx.AllocRegExcept(d26.Reg)}
				switch d26.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d26)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d26)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d26)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d26)
				d26 = tmpPair
			}
			if d26.Loc != LocRegPair && d26.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Read arg1)")
			}
			d28 = ctx.EmitGoCallScalar(GoFuncAddr(Read), []JITValueDesc{d1, d26}, 2)
			ctx.BindReg(d28.Reg, &d28)
			ctx.BindReg(d28.Reg2, &d28)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d28)
			if d28.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d28, &result)
				result.Type = d28.Type
			} else {
				switch d28.Type {
				case tagBool:
					ctx.EmitMakeBool(result, d28)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d28)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d28)
					result.Type = tagFloat
				case tagNil:
					ctx.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d28, &result)
					result.Type = d28.Type
				}
			}
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned29 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned29 = append(argPinned29, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned29 = append(argPinned29, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned29 = append(argPinned29, ai.Reg2)
					}
				}
			}
			ps30 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps30)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.FreeStack(int32(16))
			for _, r := range argPinned29 {
				ctx.UnprotectReg(r)
			}
			return result
		},
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */,
	})
	Declare(&Globalenv, &Declaration{
		"pretty_print", "formats Scheme code as an indented, human-readable string; expressions up to width characters are kept on one line, longer ones are expanded with one argument per line",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"code", "list", "Scheme code to format", nil},
			DeclarationParameter{"width", "int", "max characters before expanding (default 20)", nil},
		}, "string",
		func(a ...Scmer) Scmer {
			width := 20
			if len(a) >= 2 {
				width = ToInt(a[1])
			}
			return NewString(PrettyPrint(a[0], &Globalenv, width))
		}, false, false, nil,
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.Globalenv */,
	})

	init_alu()
	init_strings()
	init_streams()
	init_list()
	init_date()
	init_timezone()
	init_vector()
	init_parser()
	init_sync()
	init_scheduler()
	init_window()
	init_processlist()
	init_jit()
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
		sz += uint(jep.CodeLen) // actual code bytes in arena
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
