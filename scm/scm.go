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
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

func symbolName(v Scmer) (string, bool) {
	if v.IsSourceInfo() {
		return symbolName(v.SourceInfo().value)
	}
	if v.GetTag() == tagSymbol {
		return v.String(), true
	}
	if v.GetTag() == tagSpecialForm {
		return v.SpecialFormName(), true
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

func evalWithSourceInfo(si *SourceInfo, en *Env) (value Scmer) {
	if si == nil {
		return NewNil()
	}
	si.markInterpreted()
	if !SettingsHaveGoodBacktraces {
		return Eval(si.value, en)
	}
	defer func(src SourceInfo) {
		if err := recover(); err != nil {
			panic(fmt.Sprintf("%s\nin %s:%d:%d", fmt.Sprint(err), src.source, src.line, src.col))
		}
	}(*si)
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
	case tagSlice:
		// Hot path: optimized queryplan/runtime code is dominated by call forms.
		list := expression.Slice()
		if len(list) == 0 {
			return expression
		}
		// apply
		operands := list[1:]
		procedure := Eval(list[0], en) // resolve syntax, lambdas, and ordinary functions
		switch procedure.GetTag() {
		case tagSpecialForm:
			switch procedure.SpecialFormName() {
			case "outer":
				if en.Outer == nil {
					return NewNil()
				}
				if operands[0].IsSymbol() {
					symbol := operands[0].Symbol()
					if outer := en.Outer.FindRead(symbol); outer != nil {
						if result, exists := outer.Vars[symbol]; exists {
							return result
						}
					}
					symbolName := string(symbol)
					if strings.Contains(symbolName, ".") && !strings.Contains(symbolName, "\x00") {
						suffix := "\x00" + symbolName
						for outer := en.Outer; outer != nil; outer = outer.Outer {
							for key, result := range outer.Vars {
								if strings.HasSuffix(string(key), suffix) {
									return result
								}
							}
						}
					}
				}
				en = en.Outer
				expression = operands[0]
				goto restart
			case "eval":
				expression = Eval(operands[0], en)
				goto restart
			case "if":
				i := 0
				for i+1 < len(operands) {
					if Eval(operands[i], en).Bool() {
						expression = operands[i+1]
						goto restart
					}
					i += 2
				}
				if i < len(operands) {
					expression = operands[i]
					goto restart
				}
				return NewNil()
			case "match", "match_mut":
				matchedValue := Eval(operands[0], en)
				matchEnv := Env{VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: true}
				i := 1
				mutable := procedure.SpecialFormName() == "match_mut"
				for i < len(operands)-1 {
					if match(matchedValue, operands[i], &matchEnv, mutable) {
						en = &matchEnv
						expression = operands[i+1]
						goto restart
					}
					i += 2
				}
				if i < len(operands) {
					expression = operands[i]
					goto restart
				}
				return NewNil()
			case "begin":
				beginEnv := &Env{Vars: make(Vars), VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: false}
				for _, form := range operands[:len(operands)-1] {
					Eval(form, beginEnv)
				}
				en = beginEnv
				expression = operands[len(operands)-1]
				goto restart
			case "begin_mut":
				reserve := 0
				if len(operands) > 0 {
					reserve = int(ToInt(Eval(operands[0], en)))
				}
				if reserve < 0 {
					reserve = 0
				}
				varsNumbered := en.VarsNumbered
				if reserve > 0 {
					varsNumbered = make([]Scmer, len(en.VarsNumbered)+reserve)
					copy(varsNumbered, en.VarsNumbered)
				}
				beginEnv := &Env{Vars: make(Vars), VarsNumbered: varsNumbered, Outer: en, Nodefine: false}
				for _, form := range operands[1 : len(operands)-1] {
					Eval(form, beginEnv)
				}
				en = beginEnv
				expression = operands[len(operands)-1]
				goto restart
			case "!begin":
				for _, form := range operands[:len(operands)-1] {
					Eval(form, en)
				}
				expression = operands[len(operands)-1]
				goto restart
			default:
				return procedure.SpecialForm()(operands, en)
			}
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
		case tagProc:
			// Lambdas (procs)
			if proc := procedure.Proc(); proc != nil && proc.Compiled != nil {
				args := make([]Scmer, len(operands))
				for i, operand := range operands {
					args[i] = Eval(operand, en)
				}
				return proc.Compiled.Call(args...)
			}
			en, expression = prepareProcCall(procedure.Proc(), operands, en)
			goto restart
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
		case tagPromise:
			if n := len(operands); n <= 4 {
				var buf [4]Scmer
				for i := 0; i < n; i++ {
					buf[i] = Eval(operands[i], en)
				}
				return ApplyPromise(procedure, buf[:n])
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return ApplyPromise(procedure, args)
		case tagSlice:
			// Associative list
			p := procedure.Slice()
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
				return jep.Call(buf[:n]...)
			}
			args := make([]Scmer, len(operands))
			for i, x := range operands {
				args[i] = Eval(x, en)
			}
			return jep.Call(args...)
		default:
			panic("Unknown function: " + list[0].String())
		}
	case tagFunc, tagFuncEnv, tagProc, tagJIT, tagClosure, tagPromise, tagSpecialForm:
		// Optimizer-resolved native callables.
		return expression
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagVector, tagFastDict, tagParser, tagAny, tagBSON:
		// Self-evaluating literals.
		return expression
	case tagNthLocalVar:
		// Optimized lambda bodies resolve locals directly through numbered slots.
		idx := int(expression.NthLocalVar())
		if idx >= len(en.VarsNumbered) {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			panic(fmt.Sprintf("NthLocalVar(%d) out of range (len=%d)\n%s", idx, len(en.VarsNumbered), buf[:n]))
		}
		return en.VarsNumbered[idx]
	case tagSymbol:
		// Fallback for names not folded to numbered vars/native funcs by the optimizer.
		sym := mustSymbol(expression)
		if scope := en.FindRead(sym); scope != nil {
			if value, ok := scope.Vars[sym]; ok {
				return value
			}
		}
		return NewNil()
	case tagSourceInfo:
		return evalWithSourceInfo(expression.SourceInfo(), en)
	default:
		if expression.GetTag() >= 100 {
			// custom tags (e.g. TagTable) are opaque literals
			return expression
		}
		panic("Unknown expression type - EVAL " + expression.String())
	}
	return
}

type smallProcCallEnv struct {
	env      Env
	numbered [4]Scmer
}

func newProcCallEnv(proc Proc) *Env {
	if proc.NumVars <= 4 {
		frame := &smallProcCallEnv{}
		frame.env.Outer = proc.En
		frame.env.VarsNumbered = frame.numbered[:proc.NumVars]
		return &frame.env
	}
	return &Env{VarsNumbered: make([]Scmer, proc.NumVars), Outer: proc.En}
}

func prepareProcCall(p *Proc, operands []Scmer, caller *Env) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := *p
	var vars Vars
	env := newProcCallEnv(proc)
	switch proc.Params.GetTag() {
	case tagSlice:
		params := proc.Params.Slice()
		if len(params) < len(operands) {
			panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(operands)))
		}
		if proc.NumVars > 0 {
			if !proc.NumberedOnly {
				vars = make(Vars, len(params))
				env.Vars = vars
			}
			for i := range params {
				if i < len(operands) && i < proc.NumVars {
					val := Eval(operands[i], caller)
					env.VarsNumbered[i] = val
					if !proc.NumberedOnly && !params[i].SymbolEquals("_") {
						env.Vars[mustSymbol(params[i])] = val
					}
				} else if !proc.NumberedOnly && !params[i].SymbolEquals("_") {
					env.Vars[mustSymbol(params[i])] = NewNil()
				}
			}
		} else {
			vars = make(Vars, len(params))
			env.Vars = vars
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
			if !proc.NumberedOnly {
				vars = make(Vars, 1)
				env.Vars = vars
				env.Vars[mustSymbol(proc.Params)] = argsList
			}
		} else {
			vars = make(Vars, 1)
			env.Vars = vars
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
	env := newProcCallEnv(proc)
	switch proc.Params.GetTag() {
	case tagSlice:
		params := proc.Params.Slice()
		if proc.NumVars > 0 {
			if !proc.NumberedOnly {
				vars = make(Vars, len(params))
				env.Vars = vars
			}
			for i := range params {
				if i < len(args) {
					env.VarsNumbered[i] = args[i]
					if !proc.NumberedOnly && !params[i].SymbolEquals("_") {
						env.Vars[mustSymbol(params[i])] = args[i]
					}
				} else if !proc.NumberedOnly && !params[i].SymbolEquals("_") {
					env.Vars[mustSymbol(params[i])] = NewNil()
				}
			}
		} else {
			vars = make(Vars, len(params))
			env.Vars = vars
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
			if !proc.NumberedOnly {
				vars = make(Vars, 1)
				env.Vars = vars
				env.Vars[mustSymbol(proc.Params)] = argsList
			}
		} else {
			vars = make(Vars, 1)
			env.Vars = vars
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
		if proc := procedure.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call(args...)
		}
		env, body := prepareProcCallWithArgs(procedure.Proc(), args)
		return Eval(body, env)
	// Assoc list
	case tagSlice:
		p := procedure.Slice()
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
		return procedure.JIT().Call(args...)
	case tagPromise:
		return ApplyPromise(procedure, args)
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
	NumberedOnly bool
	// Compiled is an optional native implementation of this procedure. The
	// original body remains attached so storage scan callbacks can later be
	// specialized and recompiled against concrete column/storage types.
	Compiled *JITEntryPoint
	// OptimizerMeta belongs to this concrete procedure identity. Exact Proc
	// copies, including the JIT's source/compiled pair, share it. Copies which
	// change Body or En must allocate a new identity so cached specializations
	// can never retain code or captures from a different Proc.
	// Keep optimizer-only fields after the runtime/JIT-facing Proc layout.
	OptimizerMeta *ProcOptimizerMeta
}

// ProcOptimizerMeta belongs to one concrete Proc identity. Specialization
// variants publish immutable snapshots for lock-free reads; only a miss for the
// same procedure and specialization key coordinates compilation.
type ProcOptimizerMeta struct {
	Return    TypeInfo
	HasReturn bool

	// specializations is an immutable, atomically published snapshot. Its
	// values are complete Proc values (wrapped as Scmer), not aliases in an
	// Env. Consequently every variant retains its optimized body, captures,
	// return metadata and independent JIT entry point. Rejected keys prevent a
	// read-only Transfer fact from repeatedly recompiling an unchanged Proc.
	specializations  atomic.Pointer[procSpecializationSnapshot]
	specializationMu sync.Mutex
	building         map[procSpecializationKey]*procSpecializationBuild
}

type procSpecializationKey uint64

type procSpecializationSnapshot struct {
	variants map[procSpecializationKey]Scmer
	rejected map[procSpecializationKey]struct{}
}

type procSpecializationBuild struct {
	done chan struct{}
}

func (m *ProcOptimizerMeta) specialization(key procSpecializationKey) (Scmer, bool) {
	if m == nil {
		return NewNil(), false
	}
	snapshot := m.specializations.Load()
	if snapshot == nil {
		return NewNil(), false
	}
	variant, exists := snapshot.variants[key]
	return variant, exists
}

func (m *ProcOptimizerMeta) specializationRejected(key procSpecializationKey) bool {
	if m == nil {
		return false
	}
	snapshot := m.specializations.Load()
	if snapshot == nil {
		return false
	}
	_, rejected := snapshot.rejected[key]
	return rejected
}

// beginSpecialization elects one compiler for a Proc/key pair. Callers for a
// different key do not wait; callers for the same key wait on build.done.
func (m *ProcOptimizerMeta) beginSpecialization(key procSpecializationKey) (*procSpecializationBuild, bool) {
	m.specializationMu.Lock()
	defer m.specializationMu.Unlock()
	if _, exists := m.specialization(key); exists || m.specializationRejected(key) {
		return nil, false
	}
	if build := m.building[key]; build != nil {
		return build, false
	}
	if m.building == nil {
		m.building = make(map[procSpecializationKey]*procSpecializationBuild)
	}
	build := &procSpecializationBuild{done: make(chan struct{})}
	m.building[key] = build
	return build, true
}

func (m *ProcOptimizerMeta) finishSpecialization(key procSpecializationKey, variant Scmer) {
	m.specializationMu.Lock()
	build := m.building[key]
	previous := m.specializations.Load()
	variantCount := 0
	rejectedCount := 0
	if previous != nil {
		variantCount = len(previous.variants)
		rejectedCount = len(previous.rejected)
	}
	variants := make(map[procSpecializationKey]Scmer, variantCount+1)
	rejected := make(map[procSpecializationKey]struct{}, rejectedCount+1)
	if previous != nil {
		for previousKey, previousVariant := range previous.variants {
			variants[previousKey] = previousVariant
		}
		for previousKey := range previous.rejected {
			rejected[previousKey] = struct{}{}
		}
	}
	if variant.IsProc() {
		variants[key] = variant
	} else {
		rejected[key] = struct{}{}
	}
	m.specializations.Store(&procSpecializationSnapshot{variants: variants, rejected: rejected})
	delete(m.building, key)
	if build != nil {
		close(build.done)
	}
	m.specializationMu.Unlock()
}

// optimizerProcReturnTemplate is embedded in optimizer-generated ASTs. Eval
// turns the immutable template into a new ProcOptimizerMeta identity.
type optimizerProcReturnTemplate struct {
	Return    TypeInfo
	HasReturn bool
}

// CloseProcedure snapshots explicit captures of a procedure without retaining
// its request-local environment. The optimizer represents a capture from the
// procedure's creation frame as (outer expr). Resolve only those forms in the
// current procedure body; nested lambdas keep their own outer references,
// which bind to frames created when the closed procedure runs.
func CloseProcedure(value Scmer) Scmer {
	if value.GetTag() != tagProc {
		return value
	}
	proc := *value.Proc()
	if proc.En == nil || proc.En == &Globalenv {
		return value
	}
	callFrame := &Env{Outer: proc.En}
	bound := make(map[Symbol]struct{})
	if proc.Params.IsSlice() {
		for _, param := range proc.Params.Slice() {
			if param.IsSymbol() {
				bound[param.Symbol()] = struct{}{}
			}
		}
	} else if proc.Params.IsSymbol() {
		bound[proc.Params.Symbol()] = struct{}{}
	}
	collectProcedureBindings(proc.Body, bound)
	proc.Body = closeProcedureCaptures(proc.Body, callFrame, bound)
	proc.En = &Globalenv
	proc.Compiled = nil
	if proc.OptimizerMeta != nil {
		proc.OptimizerMeta = &ProcOptimizerMeta{
			Return:    proc.OptimizerMeta.Return,
			HasReturn: proc.OptimizerMeta.HasReturn,
		}
	}
	return NewProcStruct(proc)
}

func collectProcedureBindings(expression Scmer, bound map[Symbol]struct{}) {
	if expression.IsSourceInfo() {
		collectProcedureBindings(expression.SourceInfo().value, bound)
		return
	}
	if !expression.IsSlice() {
		return
	}
	items := expression.Slice()
	if len(items) > 0 {
		if head, ok := scmerSymbol(items[0]); ok {
			switch string(head) {
			case "lambda", "quote":
				return
			case "define", "set":
				if len(items) > 1 && items[1].IsSymbol() {
					bound[items[1].Symbol()] = struct{}{}
				}
			}
		}
	}
	for _, item := range items {
		collectProcedureBindings(item, bound)
	}
}

func closeProcedureCaptures(expression Scmer, callFrame *Env, bound map[Symbol]struct{}) Scmer {
	if expression.IsSourceInfo() {
		source := *expression.SourceInfo()
		source.value = closeProcedureCaptures(source.value, callFrame, bound)
		return NewSourceInfo(source)
	}
	if expression.IsSymbol() {
		symbol := expression.Symbol()
		if _, local := bound[symbol]; !local {
			if binding := callFrame.Outer.FindRead(symbol); binding != nil && binding != &Globalenv {
				if captured, ok := binding.Vars[symbol]; ok {
					return captured
				}
			}
		}
		return expression
	}
	if !expression.IsSlice() {
		return expression
	}
	items := expression.Slice()
	if len(items) > 0 {
		if head, ok := scmerSymbol(items[0]); ok {
			switch string(head) {
			case "outer":
				return Eval(expression, callFrame)
			case "lambda", "quote":
				return expression
			}
		}
	}
	closed := make([]Scmer, len(items))
	for i, item := range items {
		closed[i] = closeProcedureCaptures(item, callFrame, bound)
	}
	return NewSlice(closed)
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

func (e *Env) definitionTarget() *Env {
	for e != nil && e.Nodefine {
		e = e.Outer
	}
	if e == nil {
		return &Globalenv
	}
	return e
}

func (e *Env) optimizerProcHint(s Symbol) (TypeInfo, bool) {
	binding := e.FindRead(s)
	if binding == nil {
		return tiZero, false
	}
	bound, exists := binding.Vars[s]
	if !exists || bound.GetTag() != tagProc {
		return tiZero, false
	}
	proc := bound.Proc()
	if proc.OptimizerMeta == nil || !proc.OptimizerMeta.HasReturn {
		return tiZero, false
	}
	return proc.OptimizerMeta.Return, true
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
	registerSpecialForms()

	// system
	DeclareTitle("SCM Builtins")
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "quote",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "returns a symbol or list without evaluating it",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to quote"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,
		},
	}, specialQuote)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "eval",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "executes the given scheme program in the current environment",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "code", Description: "list with head and optional parameters"},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	}, nil)
	Declare(&Globalenv, &Declaration{
		Name: "size",

		Fn: func(a ...Scmer) Scmer {
			return NewInt(int64(ComputeSize(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compute the memory size of a value",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to examine"},
			},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
		},
	})
	optimizerTelemetryType := &TypeDescriptor{Kind: "assoc", Keys: map[string]*TypeDescriptor{
		"compile_ns":        {Kind: "int"},
		"input_nodes":       {Kind: "int"},
		"output_nodes":      {Kind: "int"},
		"rewrites":          {Kind: "int"},
		"rejected_rewrites": {Kind: "int"},
		"budget_remaining":  {Kind: "int"},
		"callback_analyses": {Kind: "int"},
		"callback_clones":   {Kind: "int"},
	}}
	Declare(&Globalenv, &Declaration{
		Name: "optimize",

		Fn: func(a ...Scmer) Scmer {
			var report func(Scmer)
			if len(a) == 2 {
				callback := a[1]
				report = func(telemetry Scmer) {
					Apply(callback, telemetry)
				}
			}
			return Optimize(a[0], &Globalenv, report)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimize the given scheme program and optionally report telemetry after completion",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "code", Description: "list with head and optional parameters"},
				{
					Kind: "func", Label: "telemetry_callback", Description: "optional callback invoked once with optimizer telemetry", Optional: true, NoEscape: true,
					Params: []*TypeDescriptor{optimizerTelemetryType}, Return: &TypeDescriptor{Kind: "any"},
				},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["optimize"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
		Optimize: func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
			if len(v) == 2 {
				return oc.ApplyDefaultOptimization(v, useResult)
			}
			optimizedInput, inputType := oc.OptimizeSub(v[1], true)
			v[1] = optimizedInput
			oc.SetCallbackParamTypes([]*TypeDescriptor{optimizerTelemetryType})
			v[2], _ = oc.OptimizeSub(v[2], true)
			resultType := copyTypeDescriptor(inputType)
			if resultType == nil {
				resultType = &TypeDescriptor{Kind: "any"}
			}
			resultType.Const = false
			return NewSlice(v), resultType
		},
	})
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "time",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "measures the time it takes to compute the first argument",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "code", Description: "code to execute"},
				{Kind: "string", Label: "label", Description: "label to print in the log or trace", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	}, specialTime)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "if",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "condition...", Description: "condition to evaluate"},
				{Kind: "returntype", Label: "true-branch...", Description: "code to evaluate if condition is true"},
				{Kind: "any", Label: "false-branch", Description: "code to evaluate if condition is false", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "returntype"},
			Const:  true,
		},
		Optimize: optimizeIf,
	}, nil)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "and",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "lazily combines conditions using SQL three-valued logic; returns false on the first false value, nil for UNKNOWN, otherwise true",
			Params: []*TypeDescriptor{
				{Kind: "bool", Label: "condition", Description: "condition to evaluate", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
		},
		Optimize: optimizeAnd,
	}, specialAnd)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "or",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "lazily combines conditions using SQL three-valued logic; returns true on the first true value, nil for UNKNOWN, otherwise false",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "condition", Description: "condition to evaluate", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
		},
		Optimize: optimizeOr,
	}, specialOr)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "coalesce",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "returns the first value that has a non-zero value",
			Params: []*TypeDescriptor{
				{Kind: "returntype", Label: "value", Description: "value to examine", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "returntype"},
			Const:  true,
		},
		Optimize: optimizeCoalesce,
	}, specialCoalesce)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "coalesceNil",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "returns the first value that has a non-nil value",
			Params: []*TypeDescriptor{
				{Kind: "returntype", Label: "value", Description: "value to examine", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "returntype"},
			Const:  true,
		},
		Optimize: optimizeCoalesce,
	}, specialCoalesceNil)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "define",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "defines or sets a variable in the current environment",
			Params: []*TypeDescriptor{
				{Kind: "symbol", Label: "variable", Description: "variable to set"},
				{Kind: "returntype", Label: "value", Description: "value to set the variable to"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
		},
	}, specialDefine)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "set",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "defines or sets a variable in the current environment",
			Params: []*TypeDescriptor{
				{Kind: "symbol", Label: "variable", Description: "variable to set"},
				{Kind: "returntype", Label: "value", Description: "value to set the variable to"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
		},
	}, specialDefine)

	// basic
	Declare(&Globalenv, &Declaration{
		Name: "error",

		Fn: func(a ...Scmer) Scmer {
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
		Type: &TypeDescriptor{Kind: "func", Description: "halts the whole execution thread and throws an error message",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value...", Description: "value or message to throw", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "string"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "try",

		Fn: func(a ...Scmer) (result Scmer) {
			defer func() {
				err := recover()
				if err != nil {
					result = Apply(a[1], FromAny(err))
				}
			}()
			result = Apply(a[0])
			return
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tries to execute a function and returns its result. In case of a failure, the error is fed to the second function and its result value will be used",
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "func", Description: "function with no parameters that will be called", Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "errorhandler", Description: "function that takes the error as parameter", Params: []*TypeDescriptor{{Kind: "any", Label: "error"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["try"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "apply",

		Fn: func(a ...Scmer) Scmer {
			return Apply(a[0], asSlice(a[1], "apply")...)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "runs the function with its arguments",
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "function", Description: "function to execute", Params: []*TypeDescriptor{{Kind: "any", Label: "argument", Variadic: true}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}},
				{Kind: "list", Label: "arguments", Description: "list of arguments to apply"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*3)
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
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned0 = append(argPinned0, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned0 {
						ctx.UnprotectReg(r)
					}
				}()
				d1 := args[0]
				d1.ID = 0
				d2 := args[1]
				d2.ID = 0
				var d3 JITValueDesc
				if d2.Type == tagSlice {
					d3 = jitKnownSliceHeader(ctx, &d2)
				} else {
					d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
				}
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.BindReg(d3.Reg3, &d3)
				ctx.FreeDesc(&d2)
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
					panic("jit: generic call arg expects 2-word value (Apply arg0)")
				}
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				if d3.Loc != LocRegTriple && d3.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (Apply arg1)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(Apply), []JITValueDesc{d1, d3}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.FreeDesc(&d1)
				if d4.Loc == LocImm {
					if result.Loc == LocAny {
						return d4
					}
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
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "apply_assoc",

		Fn: func(a ...Scmer) Scmer {
			return ApplyAssoc(a[0], asSlice(a[1], "apply_assoc"))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "runs the function with its arguments but arguments is a assoc list",
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "function", Description: "function to execute (must be a lambda)", Params: []*TypeDescriptor{{Kind: "any", Label: "named argument", Variadic: true}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}},
				{Kind: "list", Label: "arguments", Description: "assoc list of arguments to apply"},
			},
			Return: &TypeDescriptor{Kind: "symbol"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*3)
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
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned0 = append(argPinned0, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned0 {
						ctx.UnprotectReg(r)
					}
				}()
				d1 := args[0]
				d1.ID = 0
				d2 := args[1]
				d2.ID = 0
				var d3 JITValueDesc
				if d2.Type == tagSlice {
					d3 = jitKnownSliceHeader(ctx, &d2)
				} else {
					d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
				}
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.BindReg(d3.Reg3, &d3)
				ctx.FreeDesc(&d2)
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
					panic("jit: generic call arg expects 2-word value (ApplyAssoc arg0)")
				}
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				if d3.Loc != LocRegTriple && d3.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (ApplyAssoc arg1)")
				}
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(ApplyAssoc), []JITValueDesc{d1, d3}, 2)
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.FreeDesc(&d1)
				if d4.Loc == LocImm {
					if result.Loc == LocAny {
						return d4
					}
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
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "symbol",

		Fn: func(a ...Scmer) Scmer {
			return NewSymbol(String(a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a symbol built from that string",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "value", Description: "string value that will be converted into a symbol"},
			},
			Return: &TypeDescriptor{Kind: "symbol"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
					if result.Loc == LocAny {
						return d4
					}
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
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "list",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list containing the parameters as alements",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value...", Description: "value for the list", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "for",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Sequential loop over a list state; applies a condition and step function and returns the final state list.\nUse only when iterations have strong data dependencies and must run sequentially.\n\nExamples:\n- Count to 10: (for '(0) (lambda (x) (< x 10)) (lambda (x) (list (+ x 1))))  => '(10)\n- Sum 0..9:   (for '(0 0) (lambda (x sum) (< x 10)) (lambda (x sum) (list (+ x 1) (+ sum x)))) => '(10 45)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "init", Description: "initial state as a list"},
				{Kind: "func", Label: "condition", Description: "func that receives the current state as parameters and must return true if the loop shall be continued", Params: []*TypeDescriptor{{Kind: "any", Label: "state", Variadic: true}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Label: "step", Description: "step func that returns the next state as a list", Params: []*TypeDescriptor{{Kind: "any", Label: "state", Variadic: true}}, Return: &TypeDescriptor{Kind: "list"}},
			},
			Return: FreshAlloc,
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["for"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
		Optimize:                 FirstParameterMutable("for_mut"),
		OptimizeFirstArgTransfer: true,
	})
	Declare(&Globalenv, &Declaration{
		Name: "for_mut",

		Fn: func(a ...Scmer) Scmer {
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
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place for loop (optimizer-only, skips defensive state copy)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "init", Description: "owned initial state", Transfer: true},
				{Kind: "func", Label: "condition", Description: "determines whether another loop iteration should run", Params: []*TypeDescriptor{{Kind: "any", Label: "state", Description: "current loop state values", Variadic: true}}, Return: &TypeDescriptor{Kind: "bool", Label: "continue", Description: "whether to continue iterating"}},
				{Kind: "func", Label: "step", Description: "step func returning next state as list", Params: []*TypeDescriptor{{Kind: "any", Label: "state", Variadic: true}}, Return: &TypeDescriptor{Kind: "list"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["for_mut"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "string",

		Fn: func(a ...Scmer) Scmer {
			return NewString(String(a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "converts the given value into string",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "any value"},
			},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
				if result.Loc == LocAny {
					return d4
				}
				ctx.EmitMovPairToResult(&d4, &result)
				result.Type = tagString
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "match",

		Fn:// TODO: returntype as soon as repead validate is implemented */
		nil,
		Type: &TypeDescriptor{Kind: "func", Description: `takes a value evaluates the branch that first matches the given pattern
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
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to evaluate"},
				{Kind: "any", Label: "pattern...", Description: "pattern"},
				{Kind: "returntype", Label: "result...", Description: "result value when the pattern matches; this code can use the variables matched in the pattern"},
				{Kind: "any", Label: "default", Description: "(optional) value that is returned when no pattern matches", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,
		},
	}, nil)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "lambda",

		Fn:// TODO: func(...)->returntype as soon as function types are implemented
		nil,
		Type: &TypeDescriptor{Kind: "func", Description: "returns a function (func) constructed from the given code",
			Params: []*TypeDescriptor{
				{Kind: "symbol|list|nil", Label: "parameters", Description: "if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol"},
				{Kind: "any", Label: "code", Description: "value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above"},
				{Kind: "number", Label: "numvars", Description: "number of unnamed variables that can be accessed via (var 0) (var 1) etc.", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "func", Label: "lambda", Description: "function constructed from parameters and code",
				Params: []*TypeDescriptor{{Kind: "any", Label: "argument", Description: "value bound to the corresponding declared parameter", Variadic: true}},
				Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "value produced by code"},
			},
		},
	}, specialLambda)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "begin",

		Fn:// TODO: returntype as soon as repeat is implemented
		nil,
		Type: &TypeDescriptor{Kind: "func", Description: "creates a own variable scope, evaluates all sub expressions and returns the result of the last one",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "expression...", Description: "expressions to evaluate", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	}, nil)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "parallel",

		Fn:// TODO: returntype as soon as repeat is implemented
		nil,
		Type: &TypeDescriptor{Kind: "func", Description: "executes all parameters in parallel and returns nil if they are finished",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "expression...", Description: "expressions to evaluate in parallel", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	}, specialParallel)
	Declare(&Globalenv, &Declaration{
		Name: "source",

		Fn: func(a ...Scmer) Scmer {
			return NewSourceInfo(SourceInfo{
				source: String(a[0]),
				line:   ToInt(a[1]),
				col:    ToInt(a[2]),
				value:  a[3],
			})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "annotates the node with filename and line information for better backtraces",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "filename", Description: "Filename of the code"},
				{Kind: "number", Label: "line", Description: "Line of the code"},
				{Kind: "number", Label: "column", Description: "Column of the code"},
				{Kind: "returntype", Label: "code", Description: "code"},
			},
			Return: &TypeDescriptor{Kind: "returntype"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["source"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "source_coverage_report",

		Fn: sourceCoverageReport,
		Type: &TypeDescriptor{Kind: "func", Description: "returns Scheme source coverage statistics, optionally filtered by source path prefix",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "prefix", Description: "source path prefix", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "assoc"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["source_coverage_report"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "scheme",

		Fn: func(a ...Scmer) Scmer {
			filename := "eval"
			if len(a) > 1 {
				filename = String(a[1])
			}
			return Read(filename, String(a[0]))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a scheme expression into a list",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "code", Description: "Scheme code"},
				{Kind: "string", Label: "filename", Description: "optional filename", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
					if d21.Loc == LocNone {
						panic("jit: phi source has no location")
					}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
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
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "serialize",

		Fn: func(a ...Scmer) Scmer {
			return NewString(SerializeToString(a[0], &Globalenv))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "serializes a piece of code into a (hopefully) reparsable string; you shall be able to send that code over network and reparse with (scheme)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "code", Description: "Scheme code"},
			},
			Return: &TypeDescriptor{Kind: "string"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["serialize"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "pretty_print",

		Fn: func(a ...Scmer) Scmer {
			width := 20
			if len(a) >= 2 {
				width = ToInt(a[1])
			}
			return NewString(PrettyPrint(a[0], &Globalenv, width))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "formats Scheme code as an indented, human-readable string; expressions up to width characters are kept on one line, longer ones are expanded with one argument per line",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "code", Description: "Scheme code to format"},
				{Kind: "int", Label: "width", Description: "max characters before expanding (default 20)", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "string"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["pretty_print"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	init_alu()
	init_strings()
	init_json_functions()
	init_streams()
	init_list()
	init_list_assoc_extra()
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
		// Params and Body are inline Scmer fields. Their slots are covered by
		// recursive ComputeSize calls, so count the remaining Proc layout here.
		// unsafe.Sizeof keeps this accounting in sync when Proc grows.
		procFields := uint(unsafe.Sizeof(*p)) - 2*uint(unsafe.Sizeof(Scmer{}))
		sz := base + goAllocOverhead + procFields + ComputeSize(p.Params) + ComputeSize(p.Body)
		if p.OptimizerMeta != nil {
			sz += goAllocOverhead + uint(unsafe.Sizeof(*p.OptimizerMeta))
			sz += typeDescriptorRetainedSize(p.OptimizerMeta.Return.Extra, make(map[*TypeDescriptor]struct{}))
			if snapshot := p.OptimizerMeta.specializations.Load(); snapshot != nil {
				sz += 2 * goAllocOverhead
				sz += uint(len(snapshot.variants)+len(snapshot.rejected)) * uint(unsafe.Sizeof(procSpecializationKey(0)))
				for _, variant := range snapshot.variants {
					sz += ComputeSize(variant)
				}
			}
		}
		if p.Compiled != nil {
			sz += goAllocOverhead + align8(uint(p.Compiled.CodeLen))
		}
		return sz
	case tagString, tagSymbol:
		ln := uint(auxVal(v.aux))
		if ln == 0 {
			return base
		}
		return base + goAllocOverhead + align8(ln)
	case tagBSON:
		_, payload := bsonTypeAndBytes(v)
		if len(payload) == 0 {
			return base
		}
		return base + goAllocOverhead + align8(uint(len(payload)))
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
		// SourceInfo struct: source(16) + line(8) + col(8) + value(16) + coverage(1 padded to 8) = 56 bytes
		// value is an inline Scmer — covered by recursive ComputeSize base.
		// Non-Scmer fields: source header(16) + line(8) + col(8) + coverage padding(8) = 40 bytes.
		sz := base + goAllocOverhead + 40
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
		sz := base + goAllocOverhead + align8(uint(jep.CodeLen))
		sz += ComputeSize(NewProcStruct(jep.Proc))
		return sz
	case tagPromise:
		if auxVal(v.aux) == 0 {
			return base + goAllocOverhead + 32
		}
		return base
	case tagSpecialForm:
		return base + goAllocOverhead + uint(unsafe.Sizeof(SpecialForm(nil)))
	default:
		if v.GetTag() >= 100 {
			return base
		}
		fmt.Println(fmt.Sprintf("warning: unknown tag %d", v.GetTag()))
		return base
	}
}

func typeDescriptorRetainedSize(td *TypeDescriptor, visited map[*TypeDescriptor]struct{}) uint {
	if td == nil {
		return 0
	}
	if _, exists := visited[td]; exists {
		return 0
	}
	visited[td] = struct{}{}

	sz := goAllocOverhead + uint(unsafe.Sizeof(*td))
	sz += align8(uint(len(td.Kind) + len(td.Label) + len(td.Description)))
	if len(td.Params) > 0 {
		sz += goAllocOverhead + align8(uint(len(td.Params))*uint(unsafe.Sizeof(td)))
		for _, param := range td.Params {
			sz += typeDescriptorRetainedSize(param, visited)
		}
	}
	if len(td.Keys) > 0 {
		sz += goAllocOverhead
		for key, child := range td.Keys {
			sz += align8(uint(len(key)))
			sz += typeDescriptorRetainedSize(child, visited)
		}
	}
	sz += typeDescriptorRetainedSize(td.Return, visited)
	sz += typeDescriptorRetainedSize(td.Element, visited)
	return sz
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
		sz += 40
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
		sz += 40
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
		// Opaque runtime helpers (notably parser combinators) are shared and are
		// intentionally not charged to each cached AST reference. Do not emit a
		// warning per reference: a wide query can contain thousands of them and
		// turn cache accounting into synchronous log I/O.
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
	}
	if len(fd.collisions) > 0 {
		sz += goAllocOverhead + uint(len(fd.collisions))*32
		for _, positions := range fd.collisions {
			if len(positions) > 0 {
				sz += goAllocOverhead + uint(len(positions))*8
			}
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
