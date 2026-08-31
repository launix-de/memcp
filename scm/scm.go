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

func outerDepthLiteral(value Scmer) (int64, bool) {
	for value.IsSourceInfo() {
		value = value.SourceInfo().value
	}
	if value.GetTag() != tagInt && value.GetTag() != tagFloat {
		return 0, false
	}
	depth := value.Int()
	if depth < 0 || (value.GetTag() == tagFloat && value.Float() != float64(depth)) {
		return 0, false
	}
	return depth, true
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
			dispatch := procedure.specialFormDispatch()
			switch dispatch {
			case specialFormOuter:
				depth, validDepth := int64(0), false
				if len(operands) == 2 {
					depth, validDepth = outerDepthLiteral(operands[0])
				}
				if !validDepth {
					panic(fmt.Sprintf("outer expects a non-negative scope depth and an expression: %s", SerializeToString(expression, en)))
				}
				for ; depth > 0; depth-- {
					if en == nil {
						return NewNil()
					}
					en = en.Outer
				}
				if en == nil {
					return NewNil()
				}
				if operands[1].IsSymbol() {
					symbol := operands[1].Symbol()
					if outer := en.FindRead(symbol); outer != nil {
						if result, exists := outer.Vars[symbol]; exists {
							return result
						}
					}
					symbolName := string(symbol)
					if strings.Contains(symbolName, ".") && !strings.Contains(symbolName, "\x00") {
						suffix := "\x00" + symbolName
						for outer := en; outer != nil; outer = outer.Outer {
							for key, result := range outer.Vars {
								if strings.HasSuffix(string(key), suffix) {
									return result
								}
							}
						}
					}
				}
				expression = operands[1]
				goto restart
			case specialFormEval:
				expression = Eval(operands[0], en)
				goto restart
			case specialFormIf:
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
			case specialFormMatch, specialFormMatchMut:
				matchedValue := Eval(operands[0], en)
				matchEnv := Env{VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: true}
				i := 1
				mutable := dispatch == specialFormMatchMut
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
			case specialFormBegin:
				beginEnv := &Env{Vars: make(Vars), VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: false}
				for _, form := range operands[:len(operands)-1] {
					Eval(form, beginEnv)
				}
				en = beginEnv
				expression = operands[len(operands)-1]
				goto restart
			case specialFormBeginMut:
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
			case specialFormBangBegin:
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
	Sequence  procSequenceKind

	// specializations is an immutable, atomically published snapshot. Its
	// values are complete Proc values (wrapped as Scmer), not aliases in an
	// Env. Consequently every variant retains its optimized body, captures,
	// return metadata and independent JIT entry point. Rejected keys prevent a
	// read-only Transfer fact from repeatedly recompiling an unchanged Proc.
	specializations  atomic.Pointer[procSpecializationSnapshot]
	specializationMu sync.Mutex
	building         map[procSpecializationKey]*procSpecializationBuild
}

type procSequenceKind uint8

const (
	procSequenceNone procSequenceKind = iota
	procSequenceAndTerms
)

// procSpecializationKey distinguishes specialized parameters and the optimizer
// shape below them. A Proc can therefore retain independent full variants for
// ownership trees as well as higher-order callable contracts.
type procSpecializationKey struct {
	paramMask uint64
	shapeLo   uint64
	shapeHi   uint64
}

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
	Sequence  procSequenceKind
}

// CloseProcedure snapshots explicit captures of a procedure without retaining
// its request-local environment. The optimizer represents a capture from the
// procedure's creation frame as (outer depth expr). Resolve only those forms in the
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
			Sequence:  proc.OptimizerMeta.Sequence,
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
	}, specialQuote, jitEmitSpecialQuote)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "eval",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "executes the given scheme program in the current environment",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "code", Description: "list with head and optional parameters"},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	}, nil, jitEmitSpecialEval)
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["size"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d0.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d0.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d0 = tmpPair
				} else if d0.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
					switch d0.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d0)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d0)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d0)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d0)
					d0 = tmpPair
				}
				if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (ComputeSize arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(ComputeSize), []JITValueDesc{d0}, 1)
				ctx.BindReg(d1.Reg, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				var d2 JITValueDesc
				if d1.Loc == LocImm {
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d1.Imm.Int()))))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitMovRegReg(r0, d1.Reg)
					d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d2.Loc == LocImm {
					ctx.EmitMakeInt(result, d2)
				} else {
					ctx.EmitMakeInt(result, d2)
					ctx.FreeReg(d2.Reg)
				}
				result.Type = tagInt
				return result
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["optimize"].Fn, args, result)
				}
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
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 2)
						ctx.EmitSetcc(r0, CondEqual)
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
							if ps.General {
							}
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						ps6.PhiValues = make([]JITValueDesc, 1)
						d7 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
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
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
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
					d10 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
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
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d18 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d18)
					ctx.BindReg(r2, &d18)
					d19 = args[1]
					d19.ID = 0
					ctx.SyncDesc(&d19)
					ctx.FreeDesc(&d19)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(JITBuildScmerCallback), []JITValueDesc{d19}, 1)
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d19)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d21 = d20
						if d21.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d21)
						ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
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
					d23 = d20
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
							ctx.EmitStoreToStack(d24, int32(bbs[2].PhiBase)+int32(0))
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
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					blockPinnedRegs25 := make([]Reg, 0, 3)
					seenBlockPinnedRegs26 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs26
					for _, r := range []Reg{d20.Reg, d20.Reg2, d20.Reg3} {
						live := d20.Loc == LocRegTriple && (r == d20.Reg || r == d20.Reg2 || r == d20.Reg3)
						if live && !seenBlockPinnedRegs26[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs26[r] = true
							blockPinnedRegs25 = append(blockPinnedRegs25, r)
						}
					}
					unpinBlockRegs27 := func() {
						for _, r := range blockPinnedRegs25 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs27()
					d28 = args[0]
					d28.ID = 0
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d28.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d28)
						} else if d28.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d28)
						} else if d28.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d28)
						} else if d28.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d28.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d28 = tmpPair
					} else if d28.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocRegExcept(d28.Reg), Reg2: ctx.AllocRegExcept(d28.Reg)}
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d28)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d28)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d28)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d28)
						d28 = tmpPair
					}
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (Optimize arg0)")
					}
					d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d29.Loc == LocRegPair || d29.Loc == LocStackPair || d29.Loc == LocRegTriple || d29.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d29)
					ctx.SyncDesc(&d1)
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(Optimize), []JITValueDesc{d28, d29, d1}, 2)
					ctx.BindReg(d30.Reg, &d30)
					ctx.BindReg(d30.Reg2, &d30)
					ctx.FreeDesc(&d28)
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d30, &result)
						result.Type = d30.Type
					} else {
						switch d30.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d30)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d30)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d30)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d30, &result)
							result.Type = d30.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps31 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps31)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
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
	}, specialTime, jitEmitSpecialTime)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "if",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "condition...", Description: "condition to evaluate"},
				{Kind: "returntype", Label: "true-branch...", Description: "code to evaluate if condition is true"},
				{Kind: "any", Label: "false-branch", Description: "code to evaluate if condition is false", Variadic: true},
			},
			Return:      &TypeDescriptor{Kind: "returntype"},
			Const:       true,
			JITEmitCond: jitEmitSpecialIfCond,
		},
		Optimize: optimizeIf,
	}, nil, jitEmitSpecialIf)
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "and",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "lazily combines conditions using SQL three-valued logic; returns false on the first false value, nil for UNKNOWN, otherwise true",
			Params: []*TypeDescriptor{
				{Kind: "bool", Label: "condition", Description: "condition to evaluate", Variadic: true},
			},
			Return:      &TypeDescriptor{Kind: "bool"},
			Const:       true,
			JITEmitCond: jitEmitSpecialBoolFoldCond(false),
		},
		Optimize: optimizeAnd,
	}, specialAnd, jitEmitSpecialBoolFold(false))
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name: "or",

		Fn: nil,
		Type: &TypeDescriptor{Kind: "func", Description: "lazily combines conditions using SQL three-valued logic; returns true on the first true value, nil for UNKNOWN, otherwise false",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "condition", Description: "condition to evaluate", Variadic: true},
			},
			Return:      &TypeDescriptor{Kind: "bool"},
			Const:       true,
			JITEmitCond: jitEmitSpecialBoolFoldCond(true),
		},
		Optimize: optimizeOr,
	}, specialOr, jitEmitSpecialBoolFold(true))
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
	}, specialCoalesce, jitEmitSpecialCoalesce(false))
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
	}, specialCoalesceNil, jitEmitSpecialCoalesce(true))
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
	}, specialDefine, jitEmitSpecialDefine)
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
	}, specialDefine, jitEmitSpecialDefine)

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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d24 JITValueDesc
				_ = d24
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiCount = uint16(1)
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
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 1)
						ctx.EmitSetcc(r0, CondEqual)
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
							if ps.General {
							}
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						if ps.General {
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						return bbs[2].RenderPS(ps6)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 5)
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 5)
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					alloc13 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps8)
					}
					ctx.RestoreAllocState(alloc13)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps7)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					ctx.ReclaimUntrackedRegs()
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(func() *strings.Builder { return new(strings.Builder) }), nil, 1)
					ctx.BindReg(d14.Reg, &d14)
					ctx.StabilizeDescForControlFlow(&d14)
					d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.StabilizeDescForControlFlow(&d15)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					}
					ps16 := PhiState{General: ps.General}
					ps16.OverlayValues = make([]JITValueDesc, 16)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[14] = d14
					ps16.OverlayValues[15] = d15
					ps16.PhiValues = make([]JITValueDesc, 1)
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps16.PhiValues[0] = d17
					if ps16.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps16)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[3].PhiBase)+int32(0))
						}
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d19 JITValueDesc
					if d1.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d19)
					}
					if d19.Loc == LocReg && d1.Loc == LocReg && d19.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d19)
					ctx.EmitStoreToStack(d19, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d15)
					var d20 JITValueDesc
					if d19.Loc == LocImm && d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d19.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d19.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d19.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedLess)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d20)
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d15.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d20)
					} else {
						r3 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpInt64(d19.Reg, d15.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d20)
					}
					ctx.FreeDesc(&d15)
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							if ps.General {
							}
							ps22 := PhiState{General: ps.General}
							ps22.OverlayValues = make([]JITValueDesc, 22)
							ps22.OverlayValues[1] = d1
							ps22.OverlayValues[2] = d2
							ps22.OverlayValues[3] = d3
							ps22.OverlayValues[4] = d4
							ps22.OverlayValues[14] = d14
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[17] = d17
							ps22.OverlayValues[18] = d18
							ps22.OverlayValues[19] = d19
							ps22.OverlayValues[20] = d20
							ps22.OverlayValues[21] = d21
							return bbs[4].RenderPS(ps22)
						}
						if ps.General {
						}
						ps23 := PhiState{General: ps.General}
						ps23.OverlayValues = make([]JITValueDesc, 22)
						ps23.OverlayValues[1] = d1
						ps23.OverlayValues[2] = d2
						ps23.OverlayValues[3] = d3
						ps23.OverlayValues[4] = d4
						ps23.OverlayValues[14] = d14
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[17] = d17
						ps23.OverlayValues[18] = d18
						ps23.OverlayValues[19] = d19
						ps23.OverlayValues[20] = d20
						ps23.OverlayValues[21] = d21
						return bbs[5].RenderPS(ps23)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d24 := ps.PhiValues[0]
							ctx.EnsureDesc(&d24)
							ctx.EmitStoreToStack(d24, int32(bbs[3].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl6)
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 25)
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[4] = d4
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[15] = d15
					ps25.OverlayValues[17] = d17
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[19] = d19
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[21] = d21
					ps25.OverlayValues[24] = d24
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 25)
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[15] = d15
					ps26.OverlayValues[17] = d17
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[24] = d24
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d4
					snap31 := d14
					snap32 := d15
					snap33 := d17
					snap34 := d18
					snap35 := d19
					snap36 := d20
					snap37 := d21
					snap38 := d24
					alloc39 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps26)
					}
					ctx.RestoreAllocState(alloc39)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d14 = snap31
					d15 = snap32
					d17 = snap33
					d18 = snap34
					d19 = snap35
					d20 = snap36
					d21 = snap37
					d24 = snap38
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps25)
					}
					return result
					ctx.FreeDesc(&d20)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs40 := make([]Reg, 0, 3)
					seenBlockPinnedRegs41 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs41
					for _, r := range []Reg{d14.Reg, d14.Reg2, d14.Reg3} {
						live := d14.Loc == LocRegTriple && (r == d14.Reg || r == d14.Reg2 || r == d14.Reg3)
						if live && !seenBlockPinnedRegs41[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs41[r] = true
							blockPinnedRegs40 = append(blockPinnedRegs40, r)
						}
					}
					unpinBlockRegs42 := func() {
						for _, r := range blockPinnedRegs40 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs42()
					ctx.EnsureDesc(&d19)
					var d43 JITValueDesc
					if d19.Loc == LocImm {
						idx := int(d19.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d43 = args[idx]
						d43.ID = 0
					} else {
						ctx.EnsureDesc(&d19)
						protected := make([]Reg, 0, len(args)*2+1)
						seen := make(map[Reg]bool)
						if !seen[d19.Reg] {
							ctx.ProtectReg(d19.Reg)
							seen[d19.Reg] = true
							protected = append(protected, d19.Reg)
						}
						for _, ai := range args {
							if ai.Loc == LocReg {
								if !seen[ai.Reg] {
									ctx.ProtectReg(ai.Reg)
									seen[ai.Reg] = true
									protected = append(protected, ai.Reg)
								}
							} else if ai.Loc == LocRegPair {
								if !seen[ai.Reg] {
									ctx.ProtectReg(ai.Reg)
									seen[ai.Reg] = true
									protected = append(protected, ai.Reg)
								}
								if !seen[ai.Reg2] {
									ctx.ProtectReg(ai.Reg2)
									seen[ai.Reg2] = true
									protected = append(protected, ai.Reg2)
								}
							} else if ai.Loc == LocStackPair {
								// no direct registers to protect
							}
						}
						r4 := ctx.AllocReg()
						r5 := ctx.AllocRegExcept(r4)
						lbl11 := ctx.ReserveLabel()
						lbl12 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d19.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl12)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d19.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							switch ai.Loc {
							case LocRegPair:
								ctx.EmitMovRegReg(r4, ai.Reg)
								ctx.EmitMovRegReg(r5, ai.Reg2)
							case LocStackPair:
								tmp := ai
								ctx.EnsureDesc(&tmp)
								if tmp.Loc != LocRegPair {
									panic("jitgen: emitter args index expected Scmer pair")
								}
								ctx.EmitMovRegReg(r4, tmp.Reg)
								ctx.EmitMovRegReg(r5, tmp.Reg2)
								ctx.FreeDesc(&tmp)
							case LocImm:
								pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
								ctx.BindReg(r4, &pair)
								ctx.BindReg(r5, &pair)
								if ai.Imm.GetTag() == tagInt {
									src := ai
									src.Type = tagInt
									src.Imm = NewInt(ai.Imm.Int())
									ctx.EmitMakeInt(pair, src)
								} else if ai.Imm.GetTag() == tagFloat {
									src := ai
									src.Type = tagFloat
									src.Imm = NewFloat(ai.Imm.Float())
									ctx.EmitMakeFloat(pair, src)
								} else if ai.Imm.GetTag() == tagBool {
									src := ai
									src.Type = tagBool
									src.Imm = NewBool(ai.Imm.Bool())
									ctx.EmitMakeBool(pair, src)
								} else if ai.Imm.GetTag() == tagNil {
									ctx.EmitMakeNil(pair)
								} else {
									ptrWord, auxWord := ai.Imm.RawWords()
									ctx.EmitMovRegImm64(r4, uint64(ptrWord))
									ctx.EmitMovRegImm64(r5, auxWord)
								}
							default:
								panic("jitgen: emitter args index expected Scmer pair")
							}
							ctx.EmitJmp(lbl11)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl12)
						d44 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &d44)
						ctx.BindReg(r5, &d44)
						ctx.BindReg(r4, &d44)
						ctx.BindReg(r5, &d44)
						ctx.EmitMakeNil(d44)
						ctx.MarkLabel(lbl11)
						for _, r := range protected {
							ctx.UnprotectReg(r)
						}
						d43 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &d43)
						ctx.BindReg(r5, &d43)
					}
					d46 = d43
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d46.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d46)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d46)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d46)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d46.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d46 = tmpPair
					} else if d46.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d46.Reg), Reg2: ctx.AllocRegExcept(d46.Reg)}
						switch d46.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d46)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d46)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d46)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d46)
						d46 = tmpPair
					} else if d46.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d46.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d46.MemPtr))
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
						d46 = tmpPair
					}
					if d46.Loc != LocRegPair && d46.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d46}, 2)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d45)
					ctx.EnsureDesc(&d45)
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d45.Imm)
						ptrWord, _ := d45.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d45.Imm.String())))
						d45 = tmpPair
					} else if d45.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d45.Type, Reg: ctx.AllocRegExcept(d45.Reg), Reg2: ctx.AllocRegExcept(d45.Reg)}
						switch d45.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d45)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d45)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d45)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d45)
						d45 = tmpPair
					}
					if d45.Loc != LocRegPair && d45.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
					}
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d45)
					callResults47 := JITEmitGoCallResults(ctx, GoFuncAddr((*strings.Builder).WriteString), []JITValueDesc{d14, d45}, []uint8{1, 2}, []uint8{0, 3})
					d48 = callResults47[0]
					_ = d48
					d49 = callResults47[1]
					_ = d49
					if ps.General {
					}
					ps50 := PhiState{General: ps.General}
					ps50.OverlayValues = make([]JITValueDesc, 50)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[14] = d14
					ps50.OverlayValues[15] = d15
					ps50.OverlayValues[17] = d17
					ps50.OverlayValues[18] = d18
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[20] = d20
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[24] = d24
					ps50.OverlayValues[43] = d43
					ps50.OverlayValues[44] = d44
					ps50.OverlayValues[45] = d45
					ps50.OverlayValues[46] = d46
					ps50.OverlayValues[48] = d48
					ps50.OverlayValues[49] = d49
					ps50.PhiValues = make([]JITValueDesc, 1)
					if ps50.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps50)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs51 := make([]Reg, 0, 3)
					seenBlockPinnedRegs52 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs52
					for _, r := range []Reg{d14.Reg, d14.Reg2, d14.Reg3} {
						live := d14.Loc == LocRegTriple && (r == d14.Reg || r == d14.Reg2 || r == d14.Reg3)
						if live && !seenBlockPinnedRegs52[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs52[r] = true
							blockPinnedRegs51 = append(blockPinnedRegs51, r)
						}
					}
					unpinBlockRegs53 := func() {
						for _, r := range blockPinnedRegs51 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs53()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps54 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps54)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
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
				{Kind: "func", CallsOnce: true, Label: "func", Description: "function with no parameters that will be called", Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", CallsOnce: true, Label: "errorhandler", Description: "function that takes the error as parameter", Params: []*TypeDescriptor{{Kind: "any", Label: "error"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
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
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["apply"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				var d2 JITValueDesc
				if d1.Type == tagSlice {
					d2 = jitKnownSliceHeader(ctx, &d1)
				} else {
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d1}, 3)
				}
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.BindReg(d2.Reg3, &d2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := d0
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				d4 := d2
				_ = d4
				ctx.StabilizeDescForControlFlow(&d4)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d3.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d3)
					} else if d3.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d3)
					} else if d3.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d3)
					} else if d3.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (ApplyEx arg0)")
				}
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if d4.Loc != LocRegTriple && d4.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
				}
				d5 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d5.Loc == LocRegPair || d5.Loc == LocStackPair || d5.Loc == LocRegTriple || d5.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d3)
				ctx.SyncDesc(&d4)
				ctx.SyncDesc(&d5)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d3, d4, d5}, 2)
				ctx.BindReg(d6.Reg, &d6)
				ctx.BindReg(d6.Reg2, &d6)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d6)
				ctx.FreeDesc(&d0)
				if d6.Loc == LocImm {
					if result.Loc == LocAny {
						return d6
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d6)
				if d6.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d6, &result)
					result.Type = d6.Type
				} else {
					switch d6.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d6)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d6)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d6)
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
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["apply_assoc"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				var d2 JITValueDesc
				if d1.Type == tagSlice {
					d2 = jitKnownSliceHeader(ctx, &d1)
				} else {
					d2 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d1}, 3)
				}
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.BindReg(d2.Reg3, &d2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := d0
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				d4 := d2
				_ = d4
				ctx.StabilizeDescForControlFlow(&d4)
				phiBase5 := ctx.AllocStack(int32(32))
				d6 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				_ = d6
				d7 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				_ = d7
				lbl0 := ctx.ReserveLabel()
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_1 := int32(-1)
				_ = bbpos_1_1
				bbpos_1_2 := int32(-1)
				_ = bbpos_1_2
				bbpos_1_3 := int32(-1)
				_ = bbpos_1_3
				bbpos_1_4 := int32(-1)
				_ = bbpos_1_4
				bbpos_1_5 := int32(-1)
				_ = bbpos_1_5
				bbpos_1_6 := int32(-1)
				_ = bbpos_1_6
				bbpos_1_7 := int32(-1)
				_ = bbpos_1_7
				bbpos_1_8 := int32(-1)
				_ = bbpos_1_8
				bbpos_1_9 := int32(-1)
				_ = bbpos_1_9
				bbpos_1_10 := int32(-1)
				_ = bbpos_1_10
				bbpos_1_11 := int32(-1)
				_ = bbpos_1_11
				bbpos_1_12 := int32(-1)
				_ = bbpos_1_12
				bbpos_1_13 := int32(-1)
				_ = bbpos_1_13
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				r0 := d3.Loc == LocReg || d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r1 := d3.Reg
				if r0 {
					ctx.ProtectReg(r1)
				}
				r2 := d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r3 := d3.Reg2
				if r2 {
					ctx.ProtectReg(r3)
				}
				r4 := d3.Loc == LocRegTriple
				r5 := d3.Reg3
				if r4 {
					ctx.ProtectReg(r5)
				}
				lbl1 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				ctx.EmitMovRegImm64(r6, 0)
				ctx.EmitMovRegImm64(r7, 0)
				d8 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d8)
				ctx.BindReg(r7, &d8)
				ctx.StabilizeDescForControlFlow(&d8)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d3)
				ctx.ReclaimUntrackedRegs()
				d9 := args[0]
				d9.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d10 JITValueDesc
				ctx.EnsureDesc(&d9)
				if d9.Loc == LocImm {
					ptrWord, _ := d9.Imm.RawWords()
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
				} else {
					if d9.Loc != LocRegPair {
						panic("jitgen: desc field base is not LocRegPair")
					}
					r8 := ctx.AllocReg()
					ctx.EmitMovRegReg(r8, d9.Reg)
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d10)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d12 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d12)
				var d11 JITValueDesc
				if d10.Loc == LocImm && d12.Loc == LocImm {
					d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() == d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r9 := ctx.AllocReg()
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d10.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d10.Reg, RegR11)
					}
					ctx.EmitSetcc(r9, CondEqual)
					d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
					ctx.BindReg(r9, &d11)
				} else if d10.Loc == LocImm {
					r10 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d12.Reg)
					ctx.EmitSetcc(r10, CondEqual)
					d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
					ctx.BindReg(r10, &d11)
				} else {
					r11 := ctx.AllocReg()
					ctx.EmitCmpInt64(d10.Reg, d12.Reg)
					ctx.EmitSetcc(r11, CondEqual)
					d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
					ctx.BindReg(r11, &d11)
				}
				ctx.FreeDesc(&d10)
				ctx.ReclaimUntrackedRegs()
				d13 := d11
				ctx.EnsureDesc(&d13)
				if d13.Loc != LocImm && d13.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl2 := ctx.ReserveLabel()
				lbl3 := ctx.ReserveLabel()
				lbl4 := ctx.ReserveLabel()
				lbl5 := ctx.ReserveLabel()
				if d13.Loc == LocImm {
					if d13.Imm.Bool() {
						ctx.MarkLabel(lbl4)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl5)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
				}
				ctx.FreeDesc(&d11)
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d14 := args[0]
				d14.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d15 JITValueDesc
				ctx.EnsureDesc(&d14)
				if d14.Loc == LocImm {
					ptrWord, _ := d14.Imm.RawWords()
					d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
				} else {
					if d14.Loc != LocRegPair {
						panic("jitgen: desc field base is not LocRegPair")
					}
					r12 := ctx.AllocReg()
					ctx.EmitMovRegReg(r12, d14.Reg)
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
					ctx.BindReg(r12, &d15)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d15)
				d17 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
				ctx.EnsureDesc(&d15)
				ctx.EnsureDesc(&d17)
				ctx.EnsureDesc(&d15)
				ctx.EnsureDesc(&d17)
				var d16 JITValueDesc
				if d15.Loc == LocImm && d17.Loc == LocImm {
					d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() == d17.Imm.Int())}
				} else if d17.Loc == LocImm {
					r13 := ctx.AllocReg()
					if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d15.Reg, int32(d17.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
						ctx.EmitCmpInt64(d15.Reg, RegR11)
					}
					ctx.EmitSetcc(r13, CondEqual)
					d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
					ctx.BindReg(r13, &d16)
				} else if d15.Loc == LocImm {
					r14 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d17.Reg)
					ctx.EmitSetcc(r14, CondEqual)
					d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
					ctx.BindReg(r14, &d16)
				} else {
					r15 := ctx.AllocReg()
					ctx.EmitCmpInt64(d15.Reg, d17.Reg)
					ctx.EmitSetcc(r15, CondEqual)
					d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
					ctx.BindReg(r15, &d16)
				}
				ctx.FreeDesc(&d15)
				ctx.ReclaimUntrackedRegs()
				d18 := d16
				ctx.EnsureDesc(&d18)
				if d18.Loc != LocImm && d18.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				lbl8 := ctx.ReserveLabel()
				if d18.Loc == LocImm {
					if d18.Imm.Bool() {
						ctx.MarkLabel(lbl7)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
					}
				} else {
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
				}
				ctx.FreeDesc(&d16)
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d19 := args[0]
				d19.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d20 JITValueDesc
				ctx.EnsureDesc(&d19)
				if d19.Loc == LocImm {
					_, auxWord := d19.Imm.RawWords()
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
				} else {
					if d19.Loc != LocRegPair {
						panic("jitgen: desc field base is not LocRegPair")
					}
					r16 := ctx.AllocReg()
					ctx.EmitMovRegReg(r16, d19.Reg2)
					d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
					ctx.BindReg(r16, &d20)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d20)
				d21 := d20
				_ = d21
				ctx.StabilizeDescForControlFlow(&d21)
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d21)
				var d22 JITValueDesc
				if d21.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d21.Imm.Int() & 255)}
				} else {
					r17 := ctx.AllocRegExcept(d21.Reg)
					ctx.EmitMovRegReg(r17, d21.Reg)
					ctx.EmitAndRegImm32(r17, int32(255))
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
					ctx.BindReg(r17, &d22)
				}
				if d22.Loc == LocReg && d21.Loc == LocReg && d22.Reg == d21.Reg {
					ctx.TransferReg(d21.Reg)
					d21.Loc = LocNone
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d22)
				var d23 JITValueDesc
				if d22.Loc == LocImm {
					d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d22.Imm.Int()))))}
				} else {
					r18 := ctx.AllocReg()
					ctx.EmitMovRegReg(r18, d22.Reg)
					ctx.EmitShlRegImm8(r18, 56)
					ctx.EmitShrRegImm8(r18, 56)
					d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
					ctx.BindReg(r18, &d23)
				}
				ctx.FreeDesc(&d22)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d23)
				ctx.FreeDesc(&d20)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d23)
				var d24 JITValueDesc
				if d23.Loc == LocImm {
					d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d23.Imm.Int()) == uint64(0xa))}
				} else {
					r19 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d23.Reg, 10)
					ctx.EmitSetcc(r19, CondEqual)
					d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
					ctx.BindReg(r19, &d24)
				}
				ctx.FreeDesc(&d23)
				ctx.ReclaimUntrackedRegs()
				r20 := ctx.AllocReg()
				ctx.EnsureDesc(&d24)
				ctx.EnsureDesc(&d24)
				if d24.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r20, d24)
				}
				ctx.EmitJmp(lbl1)
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d25 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				ctx.EnsureDesc(&d25)
				if d25.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r20, d25)
				}
				ctx.EmitJmp(lbl1)
				ctx.MarkLabel(lbl1)
				d26 := JITValueDesc{Loc: LocReg, Reg: r20}
				ctx.BindReg(r20, &d26)
				ctx.BindReg(r20, &d26)
				if r0 {
					ctx.UnprotectReg(r1)
				}
				if r2 {
					ctx.UnprotectReg(r3)
				}
				if r4 {
					ctx.UnprotectReg(r5)
				}
				ctx.ReclaimUntrackedRegs()
				d27 := d26
				ctx.EnsureDesc(&d27)
				if d27.Loc != LocImm && d27.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl9 := ctx.ReserveLabel()
				lbl10 := ctx.ReserveLabel()
				lbl11 := ctx.ReserveLabel()
				lbl12 := ctx.ReserveLabel()
				if d27.Loc == LocImm {
					if d27.Imm.Bool() {
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitCmpRegImm32(d27.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl10)
				}
				ctx.FreeDesc(&d26)
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				r21 := d3.Loc == LocReg || d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r22 := d3.Reg
				if r21 {
					ctx.ProtectReg(r22)
				}
				r23 := d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r24 := d3.Reg2
				if r23 {
					ctx.ProtectReg(r24)
				}
				r25 := d3.Loc == LocRegTriple
				r26 := d3.Reg3
				if r25 {
					ctx.ProtectReg(r26)
				}
				lbl13 := ctx.ReserveLabel()
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				bbpos_4_1 := int32(-1)
				_ = bbpos_4_1
				bbpos_4_2 := int32(-1)
				_ = bbpos_4_2
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r27 := ctx.AllocReg()
				r28 := ctx.AllocRegExcept(r27)
				ctx.EmitMovRegImm64(r27, 0)
				ctx.EmitMovRegImm64(r28, 0)
				d28 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d28)
				ctx.BindReg(r28, &d28)
				ctx.StabilizeDescForControlFlow(&d28)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d3)
				ctx.ReclaimUntrackedRegs()
				d29 := d3
				_ = d29
				ctx.ReclaimUntrackedRegs()
				d30 := ctx.EmitGetTagDesc(&d29, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d29)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				var d31 JITValueDesc
				if d30.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d30.Imm.Int()) != uint64(0xa))}
				} else {
					r29 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d30.Reg, 10)
					ctx.EmitSetcc(r29, CondNotEqual)
					d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r29}
					ctx.BindReg(r29, &d31)
				}
				ctx.FreeDesc(&d30)
				ctx.ReclaimUntrackedRegs()
				d32 := d31
				ctx.EnsureDesc(&d32)
				if d32.Loc != LocImm && d32.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl14 := ctx.ReserveLabel()
				lbl15 := ctx.ReserveLabel()
				lbl16 := ctx.ReserveLabel()
				lbl17 := ctx.ReserveLabel()
				if d32.Loc == LocImm {
					if d32.Imm.Bool() {
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					} else {
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitCmpRegImm32(d32.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl15)
				}
				ctx.FreeDesc(&d31)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d33 := args[0]
				d33.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d34 JITValueDesc
				ctx.EnsureDesc(&d33)
				if d33.Loc == LocImm {
					ptrWord, _ := d33.Imm.RawWords()
					d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
				} else {
					if d33.Loc != LocRegPair {
						panic("jitgen: desc field base is not LocRegPair")
					}
					r30 := ctx.AllocReg()
					ctx.EmitMovRegReg(r30, d33.Reg)
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
					ctx.BindReg(r30, &d34)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d34)
				ctx.EnsureDesc(&d34)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d34)
				ctx.EnsureDesc(&d34)
				ctx.FreeDesc(&d34)
				ctx.ReclaimUntrackedRegs()
				r31 := ctx.AllocReg()
				ctx.EnsureDesc(&d34)
				ctx.EnsureDesc(&d34)
				if d34.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r31, d34)
				}
				ctx.EmitJmp(lbl13)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				ctx.MarkLabel(lbl13)
				d37 := JITValueDesc{Loc: LocReg, Reg: r31}
				ctx.BindReg(r31, &d37)
				ctx.BindReg(r31, &d37)
				if r21 {
					ctx.UnprotectReg(r22)
				}
				if r23 {
					ctx.UnprotectReg(r24)
				}
				if r25 {
					ctx.UnprotectReg(r26)
				}
				ctx.StabilizeDescForControlFlow(&d37)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d37)
				var d38 JITValueDesc
				if d37.Loc == LocImm {
					d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d37)
					if d37.Loc != LocReg && d37.Loc != LocRegPair && d37.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r32 := ctx.AllocRegExcept(d37.Reg)
					ctx.EmitCmpRegImm32(d37.Reg, 0)
					ctx.EmitSetcc(r32, CondEqual)
					d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r32}
					ctx.BindReg(r32, &d38)
				}
				ctx.ReclaimUntrackedRegs()
				d39 := d38
				ctx.EnsureDesc(&d39)
				if d39.Loc != LocImm && d39.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl18 := ctx.ReserveLabel()
				lbl19 := ctx.ReserveLabel()
				lbl20 := ctx.ReserveLabel()
				lbl21 := ctx.ReserveLabel()
				if d39.Loc == LocImm {
					if d39.Imm.Bool() {
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					} else {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
					}
				} else {
					ctx.EmitCmpRegImm32(d39.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl19)
				}
				ctx.FreeDesc(&d38)
				bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d40 JITValueDesc
				ctx.EnsureDesc(&d37)
				if d37.Loc == LocImm {
					fieldAddr := uintptr(d37.Imm.Int()) + 0
					r33 := ctx.AllocReg()
					r34 := ctx.AllocRegExcept(r33)
					ctx.EmitMovRegMem64(r33, fieldAddr)
					ctx.EmitMovRegMem64(r34, fieldAddr+8)
					d40 = JITValueDesc{Loc: LocRegPair, Reg: r33, Reg2: r34}
					ctx.BindReg(r33, &d40)
					ctx.BindReg(r34, &d40)
				} else {
					off := int32(0)
					baseReg := d37.Reg
					r35 := ctx.AllocRegExcept(baseReg)
					r36 := ctx.AllocRegExcept(baseReg, r35)
					ctx.EmitMovRegMem(r35, baseReg, off)
					ctx.EmitMovRegMem(r36, baseReg, off+8)
					d40 = JITValueDesc{Loc: LocRegPair, Reg: r35, Reg2: r36}
					ctx.BindReg(r35, &d40)
					ctx.BindReg(r36, &d40)
				}
				ctx.ReclaimUntrackedRegs()
				d41 := ctx.EmitGetTagDesc(&d40, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d41)
				var d42 JITValueDesc
				if d41.Loc == LocImm {
					d42 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d41.Imm.Int()) == uint64(0x6))}
				} else {
					r37 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d41.Reg, 6)
					ctx.EmitSetcc(r37, CondEqual)
					d42 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r37}
					ctx.BindReg(r37, &d42)
				}
				ctx.FreeDesc(&d41)
				ctx.ReclaimUntrackedRegs()
				d43 := d42
				ctx.EnsureDesc(&d43)
				if d43.Loc != LocImm && d43.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				lbl24 := ctx.ReserveLabel()
				lbl25 := ctx.ReserveLabel()
				if d43.Loc == LocImm {
					if d43.Imm.Bool() {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
					} else {
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
				} else {
					ctx.EmitCmpRegImm32(d43.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
				}
				ctx.FreeDesc(&d42)
				bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl23)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl22)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d44 JITValueDesc
				ctx.EnsureDesc(&d37)
				if d37.Loc == LocImm {
					fieldAddr := uintptr(d37.Imm.Int()) + 0
					r38 := ctx.AllocReg()
					r39 := ctx.AllocRegExcept(r38)
					ctx.EmitMovRegMem64(r38, fieldAddr)
					ctx.EmitMovRegMem64(r39, fieldAddr+8)
					d44 = JITValueDesc{Loc: LocRegPair, Reg: r38, Reg2: r39}
					ctx.BindReg(r38, &d44)
					ctx.BindReg(r39, &d44)
				} else {
					off := int32(0)
					baseReg := d37.Reg
					r40 := ctx.AllocRegExcept(baseReg)
					r41 := ctx.AllocRegExcept(baseReg, r40)
					ctx.EmitMovRegMem(r40, baseReg, off)
					ctx.EmitMovRegMem(r41, baseReg, off+8)
					d44 = JITValueDesc{Loc: LocRegPair, Reg: r40, Reg2: r41}
					ctx.BindReg(r40, &d44)
					ctx.BindReg(r41, &d44)
				}
				ctx.FreeDesc(&d37)
				ctx.ReclaimUntrackedRegs()
				d45 := jitKnownSliceHeader(ctx, &d44)
				ctx.StabilizeDescForControlFlow(&d45)
				ctx.FreeDesc(&d44)
				ctx.ReclaimUntrackedRegs()
				var d46 JITValueDesc
				if d45.SliceSizeKnown {
					d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d45.KnownSliceLen))}
				} else if d45.Loc == LocImm {
					d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d45.StackOff))}
				} else if d45.Loc == LocStackTriple {
					d46 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d45.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocRegPair || d45.Loc == LocRegTriple {
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg2, ID: 0}
					} else if d45.Loc == LocReg {
						d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d46)
				callResults47 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d46, d46}, []uint8{3}, []uint8{1})
				d48 := callResults47[0]
				d48.Type = tagSlice
				ctx.StabilizeDescForControlFlow(&d48)
				ctx.FreeDesc(&d46)
				ctx.ReclaimUntrackedRegs()
				var d49 JITValueDesc
				if d45.SliceSizeKnown {
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d45.KnownSliceLen))}
				} else if d45.Loc == LocImm {
					d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d45.StackOff))}
				} else if d45.Loc == LocStackTriple {
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d45.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d45)
					if d45.Loc == LocRegPair || d45.Loc == LocRegTriple {
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg2, ID: 0}
					} else if d45.Loc == LocReg {
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.StabilizeDescForControlFlow(&d49)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase5)+int32(0))
				bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d50 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase5) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d50)
				ctx.EnsureDesc(&d50)
				var d51 JITValueDesc
				if d50.Loc == LocImm {
					d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d50.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d50.Reg)
					ctx.EmitMovRegReg(scratch, d50.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d51)
				}
				if d51.Loc == LocReg && d50.Loc == LocReg && d51.Reg == d50.Reg {
					ctx.TransferReg(d50.Reg)
					d50.Loc = LocNone
				}
				ctx.EnsureDesc(&d51)
				ctx.EmitStoreToStack(d51, int32(phiBase5)+int32(0))
				ctx.StabilizeDescForControlFlow(&d51)
				ctx.FreeDesc(&d50)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d51)
				ctx.EnsureDesc(&d49)
				ctx.EnsureDesc(&d51)
				ctx.EnsureDesc(&d49)
				ctx.EnsureDesc(&d51)
				ctx.EnsureDesc(&d49)
				var d52 JITValueDesc
				if d51.Loc == LocImm && d49.Loc == LocImm {
					d52 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d51.Imm.Int() < d49.Imm.Int())}
				} else if d49.Loc == LocImm {
					r42 := ctx.AllocRegExcept(d51.Reg)
					if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d51.Reg, int32(d49.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
						ctx.EmitCmpInt64(d51.Reg, RegR11)
					}
					ctx.EmitSetcc(r42, CondSignedLess)
					d52 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r42}
					ctx.BindReg(r42, &d52)
				} else if d51.Loc == LocImm {
					r43 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d51.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d49.Reg)
					ctx.EmitSetcc(r43, CondSignedLess)
					d52 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r43}
					ctx.BindReg(r43, &d52)
				} else {
					r44 := ctx.AllocRegExcept(d51.Reg)
					ctx.EmitCmpInt64(d51.Reg, d49.Reg)
					ctx.EmitSetcc(r44, CondSignedLess)
					d52 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r44}
					ctx.BindReg(r44, &d52)
				}
				ctx.FreeDesc(&d49)
				ctx.ReclaimUntrackedRegs()
				d53 := d52
				ctx.EnsureDesc(&d53)
				if d53.Loc != LocImm && d53.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl26 := ctx.ReserveLabel()
				lbl27 := ctx.ReserveLabel()
				lbl28 := ctx.ReserveLabel()
				lbl29 := ctx.ReserveLabel()
				if d53.Loc == LocImm {
					if d53.Imm.Bool() {
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
					} else {
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
				} else {
					ctx.EmitCmpRegImm32(d53.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl28)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl27)
				}
				ctx.FreeDesc(&d52)
				bbpos_1_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d48)
				d54 := d3
				_ = d54
				ctx.StabilizeDescForControlFlow(&d54)
				d55 := d48
				_ = d55
				ctx.StabilizeDescForControlFlow(&d55)
				r45 := d3.Loc == LocReg || d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r46 := d3.Reg
				if r45 {
					ctx.ProtectReg(r46)
				}
				r47 := d3.Loc == LocRegPair || d3.Loc == LocRegTriple
				r48 := d3.Reg2
				if r47 {
					ctx.ProtectReg(r48)
				}
				r49 := d3.Loc == LocRegTriple
				r50 := d3.Reg3
				if r49 {
					ctx.ProtectReg(r50)
				}
				r51 := d48.Loc == LocReg || d48.Loc == LocRegPair || d48.Loc == LocRegTriple
				r52 := d48.Reg
				if r51 {
					ctx.ProtectReg(r52)
				}
				r53 := d48.Loc == LocRegPair || d48.Loc == LocRegTriple
				r54 := d48.Reg2
				if r53 {
					ctx.ProtectReg(r54)
				}
				r55 := d48.Loc == LocRegTriple
				r56 := d48.Reg3
				if r55 {
					ctx.ProtectReg(r56)
				}
				bbpos_5_0 := int32(-1)
				_ = bbpos_5_0
				bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				if d54.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d54.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d54.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d54)
					} else if d54.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d54)
					} else if d54.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d54)
					} else if d54.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d54.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d54 = tmpPair
				} else if d54.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d54.Type, Reg: ctx.AllocRegExcept(d54.Reg), Reg2: ctx.AllocRegExcept(d54.Reg)}
					switch d54.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d54)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d54)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d54)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d54)
					d54 = tmpPair
				}
				if d54.Loc != LocRegPair && d54.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (ApplyEx arg0)")
				}
				ctx.EnsureDesc(&d55)
				ctx.EnsureDesc(&d55)
				ctx.EnsureDesc(&d55)
				if d55.Loc != LocRegTriple && d55.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
				}
				d56 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d56.Loc == LocRegPair || d56.Loc == LocStackPair || d56.Loc == LocRegTriple || d56.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d54)
				ctx.SyncDesc(&d55)
				ctx.SyncDesc(&d56)
				d57 := ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d54, d55, d56}, 2)
				ctx.BindReg(d57.Reg, &d57)
				ctx.BindReg(d57.Reg2, &d57)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d57)
				if r45 {
					ctx.UnprotectReg(r46)
				}
				if r47 {
					ctx.UnprotectReg(r48)
				}
				if r49 {
					ctx.UnprotectReg(r50)
				}
				if r51 {
					ctx.UnprotectReg(r52)
				}
				if r53 {
					ctx.UnprotectReg(r54)
				}
				if r55 {
					ctx.UnprotectReg(r56)
				}
				ctx.ReclaimUntrackedRegs()
				r57 := ctx.AllocReg()
				r58 := ctx.AllocRegExcept(r57)
				d58 := JITValueDesc{Loc: LocRegPair, Reg: r57, Reg2: r58}
				ctx.BindReg(r57, &d58)
				ctx.BindReg(r58, &d58)
				ctx.EmitMovPairToResult(&d57, &d58)
				ctx.EmitJmp(lbl0)
				bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl26)
				ctx.ResolveFixups()
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d51)
				ctx.ReclaimUntrackedRegs()
				d60 := ctx.EmitSliceElementAddress(&d45, &d51, 16)
				ctx.EnsureDesc(&d60)
				r59 := ctx.AllocRegExcept(d60.Reg)
				ctx.EmitMovRegMem(r59, d60.Reg, 8)
				ctx.EmitMovRegMem(d60.Reg, d60.Reg, 0)
				d59 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d60.Reg, Reg2: r59}
				ctx.BindReg(d60.Reg, &d59)
				ctx.BindReg(r59, &d59)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d59)
				d61 := d59
				_ = d61
				ctx.StabilizeDescForControlFlow(&d61)
				lbl30 := ctx.ReserveLabel()
				bbpos_6_0 := int32(-1)
				_ = bbpos_6_0
				bbpos_6_1 := int32(-1)
				_ = bbpos_6_1
				bbpos_6_2 := int32(-1)
				_ = bbpos_6_2
				bbpos_6_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d61)
				ctx.EnsureDesc(&d61)
				ctx.EnsureDesc(&d61)
				if d61.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d61.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d61.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d61)
					} else if d61.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d61)
					} else if d61.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d61)
					} else if d61.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d61.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d61 = tmpPair
				} else if d61.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d61.Type, Reg: ctx.AllocRegExcept(d61.Reg), Reg2: ctx.AllocRegExcept(d61.Reg)}
					switch d61.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d61)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d61)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d61)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d61)
					d61 = tmpPair
				}
				if d61.Loc != LocRegPair && d61.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (symbolName arg0)")
				}
				ctx.SyncDesc(&d61)
				callResults62 := JITEmitGoCallResults(ctx, GoFuncAddr(symbolName), []JITValueDesc{d61}, []uint8{2, 1}, []uint8{1, 0})
				d63 := callResults62[0]
				_ = d63
				d64 := callResults62[1]
				_ = d64
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d63)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d65 := d64
				ctx.EnsureDesc(&d65)
				if d65.Loc != LocImm && d65.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl31 := ctx.ReserveLabel()
				lbl32 := ctx.ReserveLabel()
				lbl33 := ctx.ReserveLabel()
				lbl34 := ctx.ReserveLabel()
				if d65.Loc == LocImm {
					if d65.Imm.Bool() {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl31)
					} else {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl32)
					}
				} else {
					ctx.EmitCmpRegImm32(d65.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl33)
					ctx.EmitJmp(lbl34)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl31)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl32)
				}
				ctx.FreeDesc(&d64)
				bbpos_6_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl32)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_6_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl31)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d63)
				ctx.ReclaimUntrackedRegs()
				r60 := ctx.AllocReg()
				ctx.EnsureDesc(&d63)
				ctx.EnsureDesc(&d63)
				if d63.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r60, d63)
				}
				ctx.EmitJmp(lbl30)
				ctx.MarkLabel(lbl30)
				d66 := JITValueDesc{Loc: LocReg, Reg: r60}
				ctx.BindReg(r60, &d66)
				ctx.BindReg(r60, &d66)
				ctx.StabilizeDescForControlFlow(&d66)
				ctx.FreeDesc(&d59)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase5)+int32(16))
				bbpos_1_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d67 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase5) + int32(16)}
				ctx.StabilizeDescForControlFlow(&d67)
				ctx.ReclaimUntrackedRegs()
				var d68 JITValueDesc
				if d4.SliceSizeKnown {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else if d4.Loc == LocStackTriple {
					d68 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d4.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d67)
				ctx.EnsureDesc(&d68)
				ctx.EnsureDesc(&d67)
				ctx.EnsureDesc(&d68)
				ctx.EnsureDesc(&d67)
				ctx.EnsureDesc(&d68)
				var d69 JITValueDesc
				if d67.Loc == LocImm && d68.Loc == LocImm {
					d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d67.Imm.Int() < d68.Imm.Int())}
				} else if d68.Loc == LocImm {
					r61 := ctx.AllocRegExcept(d67.Reg)
					if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d67.Reg, int32(d68.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d68.Imm.Int()))
						ctx.EmitCmpInt64(d67.Reg, RegR11)
					}
					ctx.EmitSetcc(r61, CondSignedLess)
					d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r61}
					ctx.BindReg(r61, &d69)
				} else if d67.Loc == LocImm {
					r62 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d67.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d68.Reg)
					ctx.EmitSetcc(r62, CondSignedLess)
					d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r62}
					ctx.BindReg(r62, &d69)
				} else {
					r63 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitCmpInt64(d67.Reg, d68.Reg)
					ctx.EmitSetcc(r63, CondSignedLess)
					d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r63}
					ctx.BindReg(r63, &d69)
				}
				ctx.FreeDesc(&d68)
				ctx.ReclaimUntrackedRegs()
				d70 := d69
				ctx.EnsureDesc(&d70)
				if d70.Loc != LocImm && d70.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl35 := ctx.ReserveLabel()
				lbl36 := ctx.ReserveLabel()
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				if d70.Loc == LocImm {
					if d70.Imm.Bool() {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl35)
					} else {
						ctx.MarkLabel(lbl38)
						ctx.EmitJmp(lbl36)
					}
				} else {
					ctx.EmitCmpRegImm32(d70.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl35)
					ctx.MarkLabel(lbl38)
					ctx.EmitJmp(lbl36)
				}
				ctx.FreeDesc(&d69)
				bbpos_1_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl35)
				ctx.ResolveFixups()
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d67 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d67)
				ctx.ReclaimUntrackedRegs()
				d72 := ctx.EmitSliceElementAddress(&d4, &d67, 16)
				ctx.EnsureDesc(&d72)
				r64 := ctx.AllocRegExcept(d72.Reg)
				ctx.EmitMovRegMem(r64, d72.Reg, 8)
				ctx.EmitMovRegMem(d72.Reg, d72.Reg, 0)
				d71 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d72.Reg, Reg2: r64}
				ctx.BindReg(d72.Reg, &d71)
				ctx.BindReg(r64, &d71)
				ctx.ReclaimUntrackedRegs()
				d74 := d71
				ctx.EnsureDesc(&d74)
				if d74.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d74.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d74)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d74)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d74)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d74.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d74 = tmpPair
				} else if d74.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d74.Reg), Reg2: ctx.AllocRegExcept(d74.Reg)}
					switch d74.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d74)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d74)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d74)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d74)
					d74 = tmpPair
				} else if d74.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d74.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d74.MemPtr))
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
					d74 = tmpPair
				}
				if d74.Loc != LocRegPair && d74.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d73 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d74}, 2)
				ctx.FreeDesc(&d71)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d66)
				ctx.FreeDesc(&d66)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d73)
				ctx.EnsureDesc(&d66)
				var d75 JITValueDesc
				if d66.Loc == LocImm {
					ctx.TrackImm(d66.Imm)
					ptrWord, _ := d66.Imm.RawWords()
					d75 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.EmitMovRegImm64(d75.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(d75.Reg2, uint64(len(d66.Imm.String())))
					ctx.BindReg(d75.Reg, &d75)
					ctx.BindReg(d75.Reg2, &d75)
				} else {
					d75 = d66
				}
				d76 := ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d73, d75}, 1)
				ctx.EmitAndRegImm32(d76.Reg, 1)
				d76.Type = tagBool
				ctx.BindReg(d76.Reg, &d76)
				ctx.FreeDesc(&d66)
				ctx.ReclaimUntrackedRegs()
				d77 := d76
				ctx.EnsureDesc(&d77)
				if d77.Loc != LocImm && d77.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				lbl41 := ctx.ReserveLabel()
				lbl42 := ctx.ReserveLabel()
				if d77.Loc == LocImm {
					if d77.Imm.Bool() {
						ctx.MarkLabel(lbl41)
						ctx.EmitJmp(lbl39)
					} else {
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl40)
					}
				} else {
					ctx.EmitCmpRegImm32(d77.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl41)
					ctx.EmitJmp(lbl42)
					ctx.MarkLabel(lbl41)
					ctx.EmitJmp(lbl39)
					ctx.MarkLabel(lbl42)
					ctx.EmitJmp(lbl40)
				}
				ctx.FreeDesc(&d76)
				bbpos_1_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl40)
				ctx.ResolveFixups()
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d67 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d67)
				ctx.EnsureDesc(&d67)
				var d78 JITValueDesc
				if d67.Loc == LocImm {
					d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d67.Imm.Int() + 2)}
				} else {
					scratch := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegReg(scratch, d67.Reg)
					ctx.EmitAddRegImm32(scratch, int32(2))
					d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d78)
				}
				if d78.Loc == LocReg && d67.Loc == LocReg && d78.Reg == d67.Reg {
					ctx.TransferReg(d67.Reg)
					d67.Loc = LocNone
				}
				ctx.EnsureDesc(&d78)
				ctx.EmitStoreToStack(d78, int32(phiBase5)+int32(16))
				ctx.StabilizeDescForControlFlow(&d78)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmpToPos(bbpos_1_10)
				bbpos_1_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl39)
				ctx.ResolveFixups()
				d50 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d67 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d67)
				ctx.EnsureDesc(&d67)
				var d79 JITValueDesc
				if d67.Loc == LocImm {
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d67.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegReg(scratch, d67.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d79 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d79)
				}
				if d79.Loc == LocReg && d67.Loc == LocReg && d79.Reg == d67.Reg {
					ctx.TransferReg(d67.Reg)
					d67.Loc = LocNone
				}
				ctx.FreeDesc(&d67)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				d81 := ctx.EmitSliceElementAddress(&d4, &d79, 16)
				ctx.EnsureDesc(&d81)
				r65 := ctx.AllocRegExcept(d81.Reg)
				ctx.EmitMovRegMem(r65, d81.Reg, 8)
				ctx.EmitMovRegMem(d81.Reg, d81.Reg, 0)
				d80 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d81.Reg, Reg2: r65}
				ctx.BindReg(d81.Reg, &d80)
				ctx.BindReg(r65, &d80)
				ctx.FreeDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d51)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d80)
				d82 := ctx.EmitSliceElementAddress(&d48, &d51, int32(16))
				ctx.EmitStoreScmerAt(&d82, &d80)
				ctx.FreeDesc(&d82)
				ctx.FreeDesc(&d80)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl40)
				ctx.MarkLabel(lbl0)
				d83 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r57, Reg2: r58}
				ctx.BindReg(r57, &d83)
				ctx.BindReg(r58, &d83)
				ctx.BindReg(r57, &d83)
				ctx.BindReg(r58, &d83)
				ctx.FreeDesc(&d0)
				if d83.Loc == LocImm {
					if result.Loc == LocAny {
						return d83
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d83)
				if d83.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d83, &result)
					result.Type = d83.Type
				} else {
					switch d83.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d83)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d83)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d83)
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["symbol"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d2.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d2 = tmpPair
				} else if d2.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d2)
					d2 = tmpPair
				} else if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
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
					d2 = tmpPair
				}
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d1.Imm)
					ptrWord, _ := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d1.Imm.String())))
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
					panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewSymbol), []JITValueDesc{d1}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				if d3.Loc == LocImm {
					if result.Loc == LocAny {
						return d3
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d3, &result)
					result.Type = d3.Type
				} else {
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d3)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d3)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d3)
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["for"].Fn, args, result)
				}
				var stackArray2 int32
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d60 JITValueDesc
				_ = d60
				var stackArray88 int32
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d92 JITValueDesc
				_ = d92
				var stackArray93 int32
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(24))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiCount = uint16(1)
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
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					stackArray2 = ctx.AllocStack(int32(0))
					_ = stackArray2
					d3 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d3
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.FreeDesc(&d4)
					callResults6 := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{d5}, []uint8{3}, []uint8{1})
					d7 = callResults6[0]
					d8 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(bbs[3].PhiBase) + int32(0)}
					ctx.EmitCopyDescWords(&d8, &d7, 3)
					ctx.FreeDesc(&d7)
					d7 = d8
					ctx.StabilizeDescForControlFlow(&d7)
					d9 = args[1]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Loc == LocLambdaTemplate {
						d10 = d9
					} else if d9.Loc == LocImm {
						optimizedCallback11 := NewFunc(OptimizeProcToSerialFunction(d9.Imm))
						ctx.TrackImm(optimizedCallback11)
						d10 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback11, Rooted: true}
					} else {
						d10 = ctx.RequestOptimizedCallback(1)
					}
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d9)
					d12 = args[2]
					d12.ID = 0
					var d13 JITValueDesc
					if d12.Loc == LocLambdaTemplate {
						d13 = d12
					} else if d12.Loc == LocImm {
						optimizedCallback14 := NewFunc(OptimizeProcToSerialFunction(d12.Imm))
						ctx.TrackImm(optimizedCallback14)
						d13 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback14, Rooted: true}
					} else {
						d13 = ctx.RequestOptimizedCallback(2)
					}
					ctx.StabilizeDescForControlFlow(&d13)
					ctx.FreeDesc(&d12)
					if ps.General {
					}
					ps15 := PhiState{General: ps.General}
					ps15.OverlayValues = make([]JITValueDesc, 14)
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps15.OverlayValues[7] = d7
					ps15.OverlayValues[8] = d8
					ps15.OverlayValues[9] = d9
					ps15.OverlayValues[10] = d10
					ps15.OverlayValues[12] = d12
					ps15.OverlayValues[13] = d13
					ps15.PhiValues = make([]JITValueDesc, 1)
					if ps15.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps15)
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs16 := make([]Reg, 0, 3)
					seenBlockPinnedRegs17 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs17
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
						if live && !seenBlockPinnedRegs17[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs17[r] = true
							blockPinnedRegs16 = append(blockPinnedRegs16, r)
						}
					}
					unpinBlockRegs18 := func() {
						for _, r := range blockPinnedRegs16 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs18()
					d19 = jitCopyScmerToPair(ctx, d13)
					d20 = ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeCallbackSlice), []JITValueDesc{d19, d1}, 2)
					ctx.StabilizeDescForControlFlow(&d20)
					d22 = d20
					d22.ID = 0
					d21 = ctx.EmitTagEqualsBorrowed(&d22, tagNil, JITValueDesc{Loc: LocAny})
					d23 = d21
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							if ps.General {
							}
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[7] = d7
							ps24.OverlayValues[8] = d8
							ps24.OverlayValues[9] = d9
							ps24.OverlayValues[10] = d10
							ps24.OverlayValues[12] = d12
							ps24.OverlayValues[13] = d13
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[4].RenderPS(ps24)
						}
						if ps.General {
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[7] = d7
						ps25.OverlayValues[8] = d8
						ps25.OverlayValues[9] = d9
						ps25.OverlayValues[10] = d10
						ps25.OverlayValues[12] = d12
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[5].RenderPS(ps25)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 24)
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[7] = d7
					ps26.OverlayValues[8] = d8
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[10] = d10
					ps26.OverlayValues[12] = d12
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[23] = d23
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 24)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[9] = d9
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					snap28 := d1
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d7
					snap33 := d8
					snap34 := d9
					snap35 := d10
					snap36 := d12
					snap37 := d13
					snap38 := d19
					snap39 := d20
					snap40 := d21
					snap41 := d22
					snap42 := d23
					alloc43 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc43)
					d1 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d19 = snap38
					d20 = snap39
					d21 = snap40
					d22 = snap41
					d23 = snap42
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps26)
					}
					return result
					ctx.FreeDesc(&d21)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs44 := make([]Reg, 0, 3)
					seenBlockPinnedRegs45 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs45
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
						if live && !seenBlockPinnedRegs45[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs45[r] = true
							blockPinnedRegs44 = append(blockPinnedRegs44, r)
						}
					}
					unpinBlockRegs46 := func() {
						for _, r := range blockPinnedRegs44 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs46()
					d47 = ctx.EmitNewSliceFromGoSlice(&d1)
					ctx.EnsureDesc(&d47)
					if d47.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d47, &result)
						result.Type = d47.Type
					} else {
						switch d47.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d47)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d47)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d47)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d47, &result)
							result.Type = d47.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d48 := ps.PhiValues[0]
							ctx.EnsureDesc(&d48)
							ctx.EmitStoreRegMem(d48.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d48.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d48.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs49 := make([]Reg, 0, 3)
					seenBlockPinnedRegs50 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs50
					for _, r := range []Reg{d7.Reg, d7.Reg2, d7.Reg3} {
						live := d7.Loc == LocRegTriple && (r == d7.Reg || r == d7.Reg2 || r == d7.Reg3)
						if live && !seenBlockPinnedRegs50[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs50[r] = true
							blockPinnedRegs49 = append(blockPinnedRegs49, r)
						}
					}
					unpinBlockRegs51 := func() {
						for _, r := range blockPinnedRegs49 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs51()
					ctx.StabilizeDescForControlFlow(&d1)
					d52 = jitCopyScmerToPair(ctx, d10)
					d53 = ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeCallbackSlice), []JITValueDesc{d52, d1}, 2)
					ctx.EnsureDesc(&d53)
					d54 = d53
					_ = d54
					ctx.StabilizeDescForControlFlow(&d54)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d56 = d54
					d56.ID = 0
					d55 = ctx.EmitBoolDesc(&d56, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					ctx.FreeDesc(&d53)
					d57 = d55
					ctx.EnsureDesc(&d57)
					if d57.Loc != LocImm && d57.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d57.Loc == LocImm {
						if d57.Imm.Bool() {
							if ps.General {
							}
							ps58 := PhiState{General: ps.General}
							ps58.OverlayValues = make([]JITValueDesc, 58)
							ps58.OverlayValues[1] = d1
							ps58.OverlayValues[3] = d3
							ps58.OverlayValues[4] = d4
							ps58.OverlayValues[5] = d5
							ps58.OverlayValues[7] = d7
							ps58.OverlayValues[8] = d8
							ps58.OverlayValues[9] = d9
							ps58.OverlayValues[10] = d10
							ps58.OverlayValues[12] = d12
							ps58.OverlayValues[13] = d13
							ps58.OverlayValues[19] = d19
							ps58.OverlayValues[20] = d20
							ps58.OverlayValues[21] = d21
							ps58.OverlayValues[22] = d22
							ps58.OverlayValues[23] = d23
							ps58.OverlayValues[47] = d47
							ps58.OverlayValues[48] = d48
							ps58.OverlayValues[52] = d52
							ps58.OverlayValues[53] = d53
							ps58.OverlayValues[54] = d54
							ps58.OverlayValues[55] = d55
							ps58.OverlayValues[56] = d56
							ps58.OverlayValues[57] = d57
							return bbs[1].RenderPS(ps58)
						}
						if ps.General {
						}
						ps59 := PhiState{General: ps.General}
						ps59.OverlayValues = make([]JITValueDesc, 58)
						ps59.OverlayValues[1] = d1
						ps59.OverlayValues[3] = d3
						ps59.OverlayValues[4] = d4
						ps59.OverlayValues[5] = d5
						ps59.OverlayValues[7] = d7
						ps59.OverlayValues[8] = d8
						ps59.OverlayValues[9] = d9
						ps59.OverlayValues[10] = d10
						ps59.OverlayValues[12] = d12
						ps59.OverlayValues[13] = d13
						ps59.OverlayValues[19] = d19
						ps59.OverlayValues[20] = d20
						ps59.OverlayValues[21] = d21
						ps59.OverlayValues[22] = d22
						ps59.OverlayValues[23] = d23
						ps59.OverlayValues[47] = d47
						ps59.OverlayValues[48] = d48
						ps59.OverlayValues[52] = d52
						ps59.OverlayValues[53] = d53
						ps59.OverlayValues[54] = d54
						ps59.OverlayValues[55] = d55
						ps59.OverlayValues[56] = d56
						ps59.OverlayValues[57] = d57
						return bbs[2].RenderPS(ps59)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d60 := ps.PhiValues[0]
							ctx.EnsureDesc(&d60)
							ctx.EmitStoreRegMem(d60.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d60.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d60.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d57.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ps61 := PhiState{General: true}
					ps61.OverlayValues = make([]JITValueDesc, 61)
					ps61.OverlayValues[1] = d1
					ps61.OverlayValues[3] = d3
					ps61.OverlayValues[4] = d4
					ps61.OverlayValues[5] = d5
					ps61.OverlayValues[7] = d7
					ps61.OverlayValues[8] = d8
					ps61.OverlayValues[9] = d9
					ps61.OverlayValues[10] = d10
					ps61.OverlayValues[12] = d12
					ps61.OverlayValues[13] = d13
					ps61.OverlayValues[19] = d19
					ps61.OverlayValues[20] = d20
					ps61.OverlayValues[21] = d21
					ps61.OverlayValues[22] = d22
					ps61.OverlayValues[23] = d23
					ps61.OverlayValues[47] = d47
					ps61.OverlayValues[48] = d48
					ps61.OverlayValues[52] = d52
					ps61.OverlayValues[53] = d53
					ps61.OverlayValues[54] = d54
					ps61.OverlayValues[55] = d55
					ps61.OverlayValues[56] = d56
					ps61.OverlayValues[57] = d57
					ps61.OverlayValues[60] = d60
					ps62 := PhiState{General: true}
					ps62.OverlayValues = make([]JITValueDesc, 61)
					ps62.OverlayValues[1] = d1
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[7] = d7
					ps62.OverlayValues[8] = d8
					ps62.OverlayValues[9] = d9
					ps62.OverlayValues[10] = d10
					ps62.OverlayValues[12] = d12
					ps62.OverlayValues[13] = d13
					ps62.OverlayValues[19] = d19
					ps62.OverlayValues[20] = d20
					ps62.OverlayValues[21] = d21
					ps62.OverlayValues[22] = d22
					ps62.OverlayValues[23] = d23
					ps62.OverlayValues[47] = d47
					ps62.OverlayValues[48] = d48
					ps62.OverlayValues[52] = d52
					ps62.OverlayValues[53] = d53
					ps62.OverlayValues[54] = d54
					ps62.OverlayValues[55] = d55
					ps62.OverlayValues[56] = d56
					ps62.OverlayValues[57] = d57
					ps62.OverlayValues[60] = d60
					snap63 := d1
					snap64 := d3
					snap65 := d4
					snap66 := d5
					snap67 := d7
					snap68 := d8
					snap69 := d9
					snap70 := d10
					snap71 := d12
					snap72 := d13
					snap73 := d19
					snap74 := d20
					snap75 := d21
					snap76 := d22
					snap77 := d23
					snap78 := d47
					snap79 := d48
					snap80 := d52
					snap81 := d53
					snap82 := d54
					snap83 := d55
					snap84 := d56
					snap85 := d57
					snap86 := d60
					alloc87 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps62)
					}
					ctx.RestoreAllocState(alloc87)
					d1 = snap63
					d3 = snap64
					d4 = snap65
					d5 = snap66
					d7 = snap67
					d8 = snap68
					d9 = snap69
					d10 = snap70
					d12 = snap71
					d13 = snap72
					d19 = snap73
					d20 = snap74
					d21 = snap75
					d22 = snap76
					d23 = snap77
					d47 = snap78
					d48 = snap79
					d52 = snap80
					d53 = snap81
					d54 = snap82
					d55 = snap83
					d56 = snap84
					d57 = snap85
					d60 = snap86
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps61)
					}
					return result
					ctx.FreeDesc(&d55)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					ctx.ReclaimUntrackedRegs()
					stackArray88 = ctx.AllocStack(int32(0))
					_ = stackArray88
					d89 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d89
					ctx.StabilizeDescForControlFlow(&d89)
					if ps.General {
						ctx.SyncDesc(&d89)
						if d89.Loc == LocReg {
							ctx.ProtectReg(d89.Reg)
						} else if d89.Loc == LocRegPair {
							ctx.ProtectReg(d89.Reg)
							ctx.ProtectReg(d89.Reg2)
						}
						d90 = d89
						if d90.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d90)
						if d90.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d90, int32(bbs[3].PhiBase)+int32(0), 3)
						} else {
							if d90.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d90.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d90.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d90.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if d89.Loc == LocReg {
							ctx.UnprotectReg(d89.Reg)
						} else if d89.Loc == LocRegPair {
							ctx.UnprotectReg(d89.Reg)
							ctx.UnprotectReg(d89.Reg2)
						}
					}
					ps91 := PhiState{General: ps.General}
					ps91.OverlayValues = make([]JITValueDesc, 91)
					ps91.OverlayValues[1] = d1
					ps91.OverlayValues[3] = d3
					ps91.OverlayValues[4] = d4
					ps91.OverlayValues[5] = d5
					ps91.OverlayValues[7] = d7
					ps91.OverlayValues[8] = d8
					ps91.OverlayValues[9] = d9
					ps91.OverlayValues[10] = d10
					ps91.OverlayValues[12] = d12
					ps91.OverlayValues[13] = d13
					ps91.OverlayValues[19] = d19
					ps91.OverlayValues[20] = d20
					ps91.OverlayValues[21] = d21
					ps91.OverlayValues[22] = d22
					ps91.OverlayValues[23] = d23
					ps91.OverlayValues[47] = d47
					ps91.OverlayValues[48] = d48
					ps91.OverlayValues[52] = d52
					ps91.OverlayValues[53] = d53
					ps91.OverlayValues[54] = d54
					ps91.OverlayValues[55] = d55
					ps91.OverlayValues[56] = d56
					ps91.OverlayValues[57] = d57
					ps91.OverlayValues[60] = d60
					ps91.OverlayValues[89] = d89
					ps91.OverlayValues[90] = d90
					ps91.PhiValues = make([]JITValueDesc, 1)
					d92 = d89
					ps91.PhiValues[0] = d92
					if ps91.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps91)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					ctx.ReclaimUntrackedRegs()
					stackArray93 = ctx.AllocStack(int32(0))
					_ = stackArray93
					d94 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d94
					var d95 JITValueDesc
					if d20.Type == tagSlice {
						d95 = jitKnownSliceHeader(ctx, &d20)
					} else {
						d95 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d20}, 3)
					}
					ctx.BindReg(d95.Reg, &d95)
					ctx.BindReg(d95.Reg2, &d95)
					ctx.BindReg(d95.Reg3, &d95)
					ctx.FreeDesc(&d20)
					callResults96 := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{d95}, []uint8{3}, []uint8{1})
					d97 = callResults96[0]
					d98 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(bbs[3].PhiBase) + int32(0)}
					ctx.EmitCopyDescWords(&d98, &d97, 3)
					ctx.FreeDesc(&d97)
					d97 = d98
					ctx.StabilizeDescForControlFlow(&d97)
					if ps.General {
					}
					ps99 := PhiState{General: ps.General}
					ps99.OverlayValues = make([]JITValueDesc, 99)
					ps99.OverlayValues[1] = d1
					ps99.OverlayValues[3] = d3
					ps99.OverlayValues[4] = d4
					ps99.OverlayValues[5] = d5
					ps99.OverlayValues[7] = d7
					ps99.OverlayValues[8] = d8
					ps99.OverlayValues[9] = d9
					ps99.OverlayValues[10] = d10
					ps99.OverlayValues[12] = d12
					ps99.OverlayValues[13] = d13
					ps99.OverlayValues[19] = d19
					ps99.OverlayValues[20] = d20
					ps99.OverlayValues[21] = d21
					ps99.OverlayValues[22] = d22
					ps99.OverlayValues[23] = d23
					ps99.OverlayValues[47] = d47
					ps99.OverlayValues[48] = d48
					ps99.OverlayValues[52] = d52
					ps99.OverlayValues[53] = d53
					ps99.OverlayValues[54] = d54
					ps99.OverlayValues[55] = d55
					ps99.OverlayValues[56] = d56
					ps99.OverlayValues[57] = d57
					ps99.OverlayValues[60] = d60
					ps99.OverlayValues[89] = d89
					ps99.OverlayValues[90] = d90
					ps99.OverlayValues[92] = d92
					ps99.OverlayValues[94] = d94
					ps99.OverlayValues[95] = d95
					ps99.OverlayValues[97] = d97
					ps99.OverlayValues[98] = d98
					ps99.PhiValues = make([]JITValueDesc, 1)
					if ps99.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps99)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps100 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps100)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(24))
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["for_mut"].Fn, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d10 JITValueDesc
				_ = d10
				var d12 JITValueDesc
				_ = d12
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d53 JITValueDesc
				_ = d53
				var stackArray80 int32
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d88 JITValueDesc
				_ = d88
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(24))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiCount = uint16(1)
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
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
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
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						d5 = ctx.RequestOptimizedCallback(1)
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := NewFunc(OptimizeProcToSerialFunction(d7.Imm))
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						d8 = ctx.RequestOptimizedCallback(2)
					}
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.FreeDesc(&d7)
					if ps.General {
						ctx.SyncDesc(&d3)
						if d3.Loc == LocReg {
							ctx.ProtectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.ProtectReg(d3.Reg)
							ctx.ProtectReg(d3.Reg2)
						}
						d10 = d3
						if d10.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d10)
						if d10.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d10, int32(bbs[3].PhiBase)+int32(0), 3)
						} else {
							if d10.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d10.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d10.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d10.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if d3.Loc == LocReg {
							ctx.UnprotectReg(d3.Reg)
						} else if d3.Loc == LocRegPair {
							ctx.UnprotectReg(d3.Reg)
							ctx.UnprotectReg(d3.Reg2)
						}
					}
					ps11 := PhiState{General: ps.General}
					ps11.OverlayValues = make([]JITValueDesc, 11)
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[7] = d7
					ps11.OverlayValues[8] = d8
					ps11.OverlayValues[10] = d10
					ps11.PhiValues = make([]JITValueDesc, 1)
					d12 = d3
					ps11.PhiValues[0] = d12
					if ps11.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps11)
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs13 := make([]Reg, 0, 3)
					seenBlockPinnedRegs14 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs14
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
						if live && !seenBlockPinnedRegs14[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs14[r] = true
							blockPinnedRegs13 = append(blockPinnedRegs13, r)
						}
					}
					unpinBlockRegs15 := func() {
						for _, r := range blockPinnedRegs13 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs15()
					d16 = jitCopyScmerToPair(ctx, d8)
					d17 = ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeCallbackSlice), []JITValueDesc{d16, d1}, 2)
					ctx.StabilizeDescForControlFlow(&d17)
					d19 = d17
					d19.ID = 0
					d18 = ctx.EmitTagEqualsBorrowed(&d19, tagNil, JITValueDesc{Loc: LocAny})
					d20 = d18
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[2] = d2
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[7] = d7
							ps21.OverlayValues[8] = d8
							ps21.OverlayValues[10] = d10
							ps21.OverlayValues[12] = d12
							ps21.OverlayValues[16] = d16
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[4].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[2] = d2
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[7] = d7
						ps22.OverlayValues[8] = d8
						ps22.OverlayValues[10] = d10
						ps22.OverlayValues[12] = d12
						ps22.OverlayValues[16] = d16
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[5].RenderPS(ps22)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 21)
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[10] = d10
					ps23.OverlayValues[12] = d12
					ps23.OverlayValues[16] = d16
					ps23.OverlayValues[17] = d17
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					ps23.OverlayValues[20] = d20
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 21)
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[7] = d7
					ps24.OverlayValues[8] = d8
					ps24.OverlayValues[10] = d10
					ps24.OverlayValues[12] = d12
					ps24.OverlayValues[16] = d16
					ps24.OverlayValues[17] = d17
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d7
					snap31 := d8
					snap32 := d10
					snap33 := d12
					snap34 := d16
					snap35 := d17
					snap36 := d18
					snap37 := d19
					snap38 := d20
					alloc39 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps24)
					}
					ctx.RestoreAllocState(alloc39)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d7 = snap30
					d8 = snap31
					d10 = snap32
					d12 = snap33
					d16 = snap34
					d17 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps23)
					}
					return result
					ctx.FreeDesc(&d18)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs40 := make([]Reg, 0, 3)
					seenBlockPinnedRegs41 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs41
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
						if live && !seenBlockPinnedRegs41[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs41[r] = true
							blockPinnedRegs40 = append(blockPinnedRegs40, r)
						}
					}
					unpinBlockRegs42 := func() {
						for _, r := range blockPinnedRegs40 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs42()
					d43 = ctx.EmitNewSliceFromGoSlice(&d1)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d43, &result)
						result.Type = d43.Type
					} else {
						switch d43.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d43)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d43)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d43)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d43, &result)
							result.Type = d43.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d44 := ps.PhiValues[0]
							ctx.EnsureDesc(&d44)
							ctx.EmitStoreRegMem(d44.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d44.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d44.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d45 = jitCopyScmerToPair(ctx, d5)
					d46 = ctx.EmitGoCallScalar(GoFuncAddr(jitInvokeCallbackSlice), []JITValueDesc{d45, d1}, 2)
					ctx.EnsureDesc(&d46)
					d47 = d46
					_ = d47
					ctx.StabilizeDescForControlFlow(&d47)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d49 = d47
					d49.ID = 0
					d48 = ctx.EmitBoolDesc(&d49, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.FreeDesc(&d46)
					d50 = d48
					ctx.EnsureDesc(&d50)
					if d50.Loc != LocImm && d50.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d50.Loc == LocImm {
						if d50.Imm.Bool() {
							if ps.General {
							}
							ps51 := PhiState{General: ps.General}
							ps51.OverlayValues = make([]JITValueDesc, 51)
							ps51.OverlayValues[1] = d1
							ps51.OverlayValues[2] = d2
							ps51.OverlayValues[3] = d3
							ps51.OverlayValues[4] = d4
							ps51.OverlayValues[5] = d5
							ps51.OverlayValues[7] = d7
							ps51.OverlayValues[8] = d8
							ps51.OverlayValues[10] = d10
							ps51.OverlayValues[12] = d12
							ps51.OverlayValues[16] = d16
							ps51.OverlayValues[17] = d17
							ps51.OverlayValues[18] = d18
							ps51.OverlayValues[19] = d19
							ps51.OverlayValues[20] = d20
							ps51.OverlayValues[43] = d43
							ps51.OverlayValues[44] = d44
							ps51.OverlayValues[45] = d45
							ps51.OverlayValues[46] = d46
							ps51.OverlayValues[47] = d47
							ps51.OverlayValues[48] = d48
							ps51.OverlayValues[49] = d49
							ps51.OverlayValues[50] = d50
							return bbs[1].RenderPS(ps51)
						}
						if ps.General {
						}
						ps52 := PhiState{General: ps.General}
						ps52.OverlayValues = make([]JITValueDesc, 51)
						ps52.OverlayValues[1] = d1
						ps52.OverlayValues[2] = d2
						ps52.OverlayValues[3] = d3
						ps52.OverlayValues[4] = d4
						ps52.OverlayValues[5] = d5
						ps52.OverlayValues[7] = d7
						ps52.OverlayValues[8] = d8
						ps52.OverlayValues[10] = d10
						ps52.OverlayValues[12] = d12
						ps52.OverlayValues[16] = d16
						ps52.OverlayValues[17] = d17
						ps52.OverlayValues[18] = d18
						ps52.OverlayValues[19] = d19
						ps52.OverlayValues[20] = d20
						ps52.OverlayValues[43] = d43
						ps52.OverlayValues[44] = d44
						ps52.OverlayValues[45] = d45
						ps52.OverlayValues[46] = d46
						ps52.OverlayValues[47] = d47
						ps52.OverlayValues[48] = d48
						ps52.OverlayValues[49] = d49
						ps52.OverlayValues[50] = d50
						return bbs[2].RenderPS(ps52)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d53 := ps.PhiValues[0]
							ctx.EnsureDesc(&d53)
							ctx.EmitStoreRegMem(d53.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d53.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d53.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d50.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ps54 := PhiState{General: true}
					ps54.OverlayValues = make([]JITValueDesc, 54)
					ps54.OverlayValues[1] = d1
					ps54.OverlayValues[2] = d2
					ps54.OverlayValues[3] = d3
					ps54.OverlayValues[4] = d4
					ps54.OverlayValues[5] = d5
					ps54.OverlayValues[7] = d7
					ps54.OverlayValues[8] = d8
					ps54.OverlayValues[10] = d10
					ps54.OverlayValues[12] = d12
					ps54.OverlayValues[16] = d16
					ps54.OverlayValues[17] = d17
					ps54.OverlayValues[18] = d18
					ps54.OverlayValues[19] = d19
					ps54.OverlayValues[20] = d20
					ps54.OverlayValues[43] = d43
					ps54.OverlayValues[44] = d44
					ps54.OverlayValues[45] = d45
					ps54.OverlayValues[46] = d46
					ps54.OverlayValues[47] = d47
					ps54.OverlayValues[48] = d48
					ps54.OverlayValues[49] = d49
					ps54.OverlayValues[50] = d50
					ps54.OverlayValues[53] = d53
					ps55 := PhiState{General: true}
					ps55.OverlayValues = make([]JITValueDesc, 54)
					ps55.OverlayValues[1] = d1
					ps55.OverlayValues[2] = d2
					ps55.OverlayValues[3] = d3
					ps55.OverlayValues[4] = d4
					ps55.OverlayValues[5] = d5
					ps55.OverlayValues[7] = d7
					ps55.OverlayValues[8] = d8
					ps55.OverlayValues[10] = d10
					ps55.OverlayValues[12] = d12
					ps55.OverlayValues[16] = d16
					ps55.OverlayValues[17] = d17
					ps55.OverlayValues[18] = d18
					ps55.OverlayValues[19] = d19
					ps55.OverlayValues[20] = d20
					ps55.OverlayValues[43] = d43
					ps55.OverlayValues[44] = d44
					ps55.OverlayValues[45] = d45
					ps55.OverlayValues[46] = d46
					ps55.OverlayValues[47] = d47
					ps55.OverlayValues[48] = d48
					ps55.OverlayValues[49] = d49
					ps55.OverlayValues[50] = d50
					ps55.OverlayValues[53] = d53
					snap56 := d1
					snap57 := d2
					snap58 := d3
					snap59 := d4
					snap60 := d5
					snap61 := d7
					snap62 := d8
					snap63 := d10
					snap64 := d12
					snap65 := d16
					snap66 := d17
					snap67 := d18
					snap68 := d19
					snap69 := d20
					snap70 := d43
					snap71 := d44
					snap72 := d45
					snap73 := d46
					snap74 := d47
					snap75 := d48
					snap76 := d49
					snap77 := d50
					snap78 := d53
					alloc79 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps55)
					}
					ctx.RestoreAllocState(alloc79)
					d1 = snap56
					d2 = snap57
					d3 = snap58
					d4 = snap59
					d5 = snap60
					d7 = snap61
					d8 = snap62
					d10 = snap63
					d12 = snap64
					d16 = snap65
					d17 = snap66
					d18 = snap67
					d19 = snap68
					d20 = snap69
					d43 = snap70
					d44 = snap71
					d45 = snap72
					d46 = snap73
					d47 = snap74
					d48 = snap75
					d49 = snap76
					d50 = snap77
					d53 = snap78
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps54)
					}
					return result
					ctx.FreeDesc(&d48)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					ctx.ReclaimUntrackedRegs()
					stackArray80 = ctx.AllocStack(int32(0))
					_ = stackArray80
					d81 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d81
					ctx.StabilizeDescForControlFlow(&d81)
					if ps.General {
						ctx.SyncDesc(&d81)
						if d81.Loc == LocReg {
							ctx.ProtectReg(d81.Reg)
						} else if d81.Loc == LocRegPair {
							ctx.ProtectReg(d81.Reg)
							ctx.ProtectReg(d81.Reg2)
						}
						d82 = d81
						if d82.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d82)
						if d82.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d82, int32(bbs[3].PhiBase)+int32(0), 3)
						} else {
							if d82.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d82.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d82.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d82.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if d81.Loc == LocReg {
							ctx.UnprotectReg(d81.Reg)
						} else if d81.Loc == LocRegPair {
							ctx.UnprotectReg(d81.Reg)
							ctx.UnprotectReg(d81.Reg2)
						}
					}
					ps83 := PhiState{General: ps.General}
					ps83.OverlayValues = make([]JITValueDesc, 83)
					ps83.OverlayValues[1] = d1
					ps83.OverlayValues[2] = d2
					ps83.OverlayValues[3] = d3
					ps83.OverlayValues[4] = d4
					ps83.OverlayValues[5] = d5
					ps83.OverlayValues[7] = d7
					ps83.OverlayValues[8] = d8
					ps83.OverlayValues[10] = d10
					ps83.OverlayValues[12] = d12
					ps83.OverlayValues[16] = d16
					ps83.OverlayValues[17] = d17
					ps83.OverlayValues[18] = d18
					ps83.OverlayValues[19] = d19
					ps83.OverlayValues[20] = d20
					ps83.OverlayValues[43] = d43
					ps83.OverlayValues[44] = d44
					ps83.OverlayValues[45] = d45
					ps83.OverlayValues[46] = d46
					ps83.OverlayValues[47] = d47
					ps83.OverlayValues[48] = d48
					ps83.OverlayValues[49] = d49
					ps83.OverlayValues[50] = d50
					ps83.OverlayValues[53] = d53
					ps83.OverlayValues[81] = d81
					ps83.OverlayValues[82] = d82
					ps83.PhiValues = make([]JITValueDesc, 1)
					d84 = d81
					ps83.PhiValues[0] = d84
					if ps83.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps83)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					ctx.ReclaimUntrackedRegs()
					var d85 JITValueDesc
					if d17.Type == tagSlice {
						d85 = jitKnownSliceHeader(ctx, &d17)
					} else {
						d85 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d17}, 3)
					}
					ctx.BindReg(d85.Reg, &d85)
					ctx.BindReg(d85.Reg2, &d85)
					ctx.BindReg(d85.Reg3, &d85)
					ctx.StabilizeDescForControlFlow(&d85)
					ctx.FreeDesc(&d17)
					if ps.General {
						ctx.SyncDesc(&d85)
						if d85.Loc == LocReg {
							ctx.ProtectReg(d85.Reg)
						} else if d85.Loc == LocRegPair {
							ctx.ProtectReg(d85.Reg)
							ctx.ProtectReg(d85.Reg2)
						}
						d86 = d85
						if d86.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d86)
						if d86.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d86, int32(bbs[3].PhiBase)+int32(0), 3)
						} else {
							if d86.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d86.Reg, RegRSP, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreRegMem(d86.Reg2, RegRSP, int32(bbs[3].PhiBase)+int32(0)+8)
							ctx.EmitStoreRegMem(d86.Reg3, RegRSP, int32(bbs[3].PhiBase)+int32(0)+16)
						}
						if d85.Loc == LocReg {
							ctx.UnprotectReg(d85.Reg)
						} else if d85.Loc == LocRegPair {
							ctx.UnprotectReg(d85.Reg)
							ctx.UnprotectReg(d85.Reg2)
						}
					}
					ps87 := PhiState{General: ps.General}
					ps87.OverlayValues = make([]JITValueDesc, 87)
					ps87.OverlayValues[1] = d1
					ps87.OverlayValues[2] = d2
					ps87.OverlayValues[3] = d3
					ps87.OverlayValues[4] = d4
					ps87.OverlayValues[5] = d5
					ps87.OverlayValues[7] = d7
					ps87.OverlayValues[8] = d8
					ps87.OverlayValues[10] = d10
					ps87.OverlayValues[12] = d12
					ps87.OverlayValues[16] = d16
					ps87.OverlayValues[17] = d17
					ps87.OverlayValues[18] = d18
					ps87.OverlayValues[19] = d19
					ps87.OverlayValues[20] = d20
					ps87.OverlayValues[43] = d43
					ps87.OverlayValues[44] = d44
					ps87.OverlayValues[45] = d45
					ps87.OverlayValues[46] = d46
					ps87.OverlayValues[47] = d47
					ps87.OverlayValues[48] = d48
					ps87.OverlayValues[49] = d49
					ps87.OverlayValues[50] = d50
					ps87.OverlayValues[53] = d53
					ps87.OverlayValues[81] = d81
					ps87.OverlayValues[82] = d82
					ps87.OverlayValues[84] = d84
					ps87.OverlayValues[85] = d85
					ps87.OverlayValues[86] = d86
					ps87.PhiValues = make([]JITValueDesc, 1)
					d88 = d85
					ps87.PhiValues[0] = d88
					if ps87.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps87)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps89 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps89)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(24))
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["string"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d2.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d2 = tmpPair
				} else if d2.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d2)
					d2 = tmpPair
				} else if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
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
					d2 = tmpPair
				}
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d1}, 2)
				if result.Loc == LocAny {
					return d3
				}
				ctx.EmitMovPairToResult(&d3, &result)
				result.Type = tagString
				return result
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
	}, nil, jitEmitSpecialMatch("match"))
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
	}, specialLambda, jitEmitSpecialLambda)
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
	}, nil, jitEmitSpecialBegin(true, false))
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
	}, specialParallel, jitEmitSpecialParallel)
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["source"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
				ctx.BindReg(d0.Reg, &d0)
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
				d4 := args[1]
				d4.ID = 0
				ctx.EnsureDesc(&d4)
				d5 := d4
				_ = d5
				ctx.StabilizeDescForControlFlow(&d5)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d6 JITValueDesc
				if d5.Loc == LocImm {
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int())}
				} else if d5.Type == tagInt && d5.Loc == LocRegPair {
					ctx.FreeReg(d5.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2}
					ctx.BindReg(d5.Reg2, &d6)
					ctx.BindReg(d5.Reg2, &d6)
				} else if d5.Type == tagInt && d5.Loc == LocReg {
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg}
					ctx.BindReg(d5.Reg, &d6)
					ctx.BindReg(d5.Reg, &d6)
				} else {
					d6 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d5}, 1)
					d6.Type = tagInt
					ctx.BindReg(d6.Reg, &d6)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d6)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d6)
				ctx.FreeDesc(&d4)
				d8 := args[2]
				d8.ID = 0
				ctx.EnsureDesc(&d8)
				d9 := d8
				_ = d9
				ctx.StabilizeDescForControlFlow(&d9)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d10 JITValueDesc
				if d9.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d9.Imm.Int())}
				} else if d9.Type == tagInt && d9.Loc == LocRegPair {
					ctx.FreeReg(d9.Reg)
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2}
					ctx.BindReg(d9.Reg2, &d10)
					ctx.BindReg(d9.Reg2, &d10)
				} else if d9.Type == tagInt && d9.Loc == LocReg {
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg}
					ctx.BindReg(d9.Reg, &d10)
					ctx.BindReg(d9.Reg, &d10)
				} else {
					d10 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d9}, 1)
					d10.Type = tagInt
					ctx.BindReg(d10.Reg, &d10)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d10)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				ctx.FreeDesc(&d8)
				d12 := args[3]
				d12.ID = 0
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d2)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value string) { base.source = value }), []JITValueDesc{d0, d2})
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d6)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value int) { base.line = value }), []JITValueDesc{d0, d6})
				ctx.FreeDesc(&d6)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d10)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value int) { base.col = value }), []JITValueDesc{d0, d10})
				ctx.FreeDesc(&d10)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d12)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value Scmer) { base.value = value }), []JITValueDesc{d0, d12})
				ctx.FreeDesc(&d12)
				ctx.EnsureDesc(&d0)
				d13 := d0
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d14 := ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
				ctx.BindReg(d14.Reg, &d14)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *SourceInfo) { *dst = *src }), []JITValueDesc{d14, d13})
				ctx.ReclaimUntrackedRegs()
				d15 := ctx.EmitGoCallScalar(GoFuncAddr(func() []*SourceInfo { return sourceCoverageInfos }), nil, 3)
				ctx.ReclaimUntrackedRegs()
				d16 := ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]*SourceInfo { return new([1]*SourceInfo) }), nil, 1)
				ctx.ReclaimUntrackedRegs()
				d17 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]*SourceInfo, index int, value *SourceInfo) { dst[index] = value }), []JITValueDesc{d16, d17, d14})
				ctx.ReclaimUntrackedRegs()
				sliceResults18 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]*SourceInfo) []*SourceInfo { return value[0:1:1] }), []JITValueDesc{d16}, []uint8{3}, []uint8{1})
				d19 := sliceResults18[0]
				ctx.ReclaimUntrackedRegs()
				callResults20 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []*SourceInfo) []*SourceInfo { return append(dst, src...) }), []JITValueDesc{d15, d19}, []uint8{3}, []uint8{1})
				d21 := callResults20[0]
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d21)
				ctx.EmitGoCallVoid(GoFuncAddr(func(value []*SourceInfo) { sourceCoverageInfos = value }), []JITValueDesc{d21})
				ctx.ReclaimUntrackedRegs()
				r0 := ctx.AllocReg()
				r1 := ctx.AllocRegExcept(r0)
				ctx.EmitMovRegImm64(r0, 0)
				ctx.EmitMovRegImm64(r1, 0)
				d22 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
				ctx.BindReg(r0, &d22)
				ctx.BindReg(r1, &d22)
				ctx.ReclaimUntrackedRegs()
				d23 := args[0]
				d23.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.ReclaimUntrackedRegs()
				d26 := args[0]
				d26.ID = 0
				ctx.ReclaimUntrackedRegs()
				d27 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(14)}
				d28 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				d29 := d27
				_ = d29
				ctx.StabilizeDescForControlFlow(&d29)
				d30 := d28
				_ = d30
				ctx.StabilizeDescForControlFlow(&d30)
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				var d31 JITValueDesc
				if d30.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d30.Imm.Int()) << 8))}
				} else {
					ctx.EmitShlRegImm8(d30.Reg, 8)
					d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg}
					ctx.BindReg(d30.Reg, &d31)
				}
				if d31.Loc == LocReg && d30.Loc == LocReg && d31.Reg == d30.Reg {
					ctx.TransferReg(d30.Reg)
					d30.Loc = LocNone
				}
				ctx.FreeDesc(&d30)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d29)
				var d32 JITValueDesc
				if d29.Loc == LocImm {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() & 255)}
				} else {
					ctx.EmitAndRegImm32(d29.Reg, int32(255))
					d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg}
					ctx.BindReg(d29.Reg, &d32)
				}
				if d32.Loc == LocImm {
					d32 = JITValueDesc{Loc: LocImm, Type: d32.Type, Imm: NewInt(int64(uint64(d32.Imm.Int()) & 0xff))}
				} else {
					ctx.EmitShlRegImm8(d32.Reg, 56)
					ctx.EmitShrRegImm8(d32.Reg, 56)
				}
				if d32.Loc == LocReg && d29.Loc == LocReg && d32.Reg == d29.Reg {
					ctx.TransferReg(d29.Reg)
					d29.Loc = LocNone
				}
				ctx.FreeDesc(&d29)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d32)
				ctx.EnsureDesc(&d32)
				var d33 JITValueDesc
				if d32.Loc == LocImm {
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d32.Imm.Int()))))}
				} else {
					r2 := ctx.AllocReg()
					ctx.EmitMovRegReg(r2, d32.Reg)
					ctx.EmitShlRegImm8(r2, 56)
					ctx.EmitShrRegImm8(r2, 56)
					d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d33)
				}
				ctx.FreeDesc(&d32)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d31)
				ctx.EnsureDesc(&d33)
				var d34 JITValueDesc
				if d31.Loc == LocImm && d33.Loc == LocImm {
					d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() | d33.Imm.Int())}
				} else if d31.Loc == LocImm && d31.Imm.Int() == 0 {
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg}
					ctx.BindReg(d33.Reg, &d34)
				} else if d33.Loc == LocImm && d33.Imm.Int() == 0 {
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
					ctx.BindReg(d31.Reg, &d34)
				} else if d31.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d33.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
					ctx.EmitOrInt64(scratch, d33.Reg)
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d34)
				} else if d33.Loc == LocImm {
					if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
						ctx.EmitOrRegImm32(d31.Reg, int32(d33.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()))
						ctx.EmitOrInt64(d31.Reg, RegR11)
					}
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
					ctx.BindReg(d31.Reg, &d34)
				} else {
					ctx.EmitOrInt64(d31.Reg, d33.Reg)
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg}
					ctx.BindReg(d31.Reg, &d34)
				}
				if d34.Loc == LocReg && d31.Loc == LocReg && d34.Reg == d31.Reg {
					ctx.TransferReg(d31.Reg)
					d31.Loc = LocNone
				}
				ctx.FreeDesc(&d31)
				ctx.FreeDesc(&d33)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d34)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d14)
				ctx.EmitMovToReg(d23.Reg, d14)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d34)
				ctx.EnsureDesc(&d34)
				ctx.EmitMovToReg(d26.Reg2, d34)
				ctx.FreeDesc(&d34)
				ctx.ReclaimUntrackedRegs()
				d35 := d22
				_ = d35
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				if d35.Loc == LocImm {
					if result.Loc == LocAny {
						return d35
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d35)
				if d35.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d35, &result)
					result.Type = d35.Type
				} else {
					switch d35.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d35)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d35)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d35)
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
				// JITGen native call boundary: native range iterator.
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["scheme"].Fn, args, result)
				}
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
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var phiBase33 int32
				_ = phiBase33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var stackArray123 int32
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d154 JITValueDesc
				_ = d154
				var stackArray155 int32
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d159 JITValueDesc
				_ = d159
				var d160 JITValueDesc
				_ = d160
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d171 JITValueDesc
				_ = d171
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				var d175 JITValueDesc
				_ = d175
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var stackArray218 int32
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d222 JITValueDesc
				_ = d222
				var d224 JITValueDesc
				_ = d224
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d227 JITValueDesc
				_ = d227
				var d228 JITValueDesc
				_ = d228
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d231 JITValueDesc
				_ = d231
				var d232 JITValueDesc
				_ = d232
				var d233 JITValueDesc
				_ = d233
				var d234 JITValueDesc
				_ = d234
				var d235 JITValueDesc
				_ = d235
				var d236 JITValueDesc
				_ = d236
				var d237 JITValueDesc
				_ = d237
				var d238 JITValueDesc
				_ = d238
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				var d242 JITValueDesc
				_ = d242
				var d243 JITValueDesc
				_ = d243
				var d244 JITValueDesc
				_ = d244
				var d246 JITValueDesc
				_ = d246
				var d248 JITValueDesc
				_ = d248
				var d249 JITValueDesc
				_ = d249
				var d250 JITValueDesc
				_ = d250
				var d251 JITValueDesc
				_ = d251
				var d252 JITValueDesc
				_ = d252
				var d253 JITValueDesc
				_ = d253
				var d254 JITValueDesc
				_ = d254
				var d255 JITValueDesc
				_ = d255
				var d256 JITValueDesc
				_ = d256
				var d257 JITValueDesc
				_ = d257
				var d258 JITValueDesc
				_ = d258
				var d259 JITValueDesc
				_ = d259
				var d260 JITValueDesc
				_ = d260
				var d261 JITValueDesc
				_ = d261
				var d262 JITValueDesc
				_ = d262
				var d263 JITValueDesc
				_ = d263
				var d264 JITValueDesc
				_ = d264
				var d265 JITValueDesc
				_ = d265
				var d266 JITValueDesc
				_ = d266
				var d267 JITValueDesc
				_ = d267
				var d268 JITValueDesc
				_ = d268
				var d269 JITValueDesc
				_ = d269
				var d270 JITValueDesc
				_ = d270
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				var d273 JITValueDesc
				_ = d273
				var d274 JITValueDesc
				_ = d274
				var d275 JITValueDesc
				_ = d275
				var d276 JITValueDesc
				_ = d276
				var d277 JITValueDesc
				_ = d277
				var d278 JITValueDesc
				_ = d278
				var d279 JITValueDesc
				_ = d279
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				var d282 JITValueDesc
				_ = d282
				var d283 JITValueDesc
				_ = d283
				var d284 JITValueDesc
				_ = d284
				var d285 JITValueDesc
				_ = d285
				var d286 JITValueDesc
				_ = d286
				var d287 JITValueDesc
				_ = d287
				var d289 JITValueDesc
				_ = d289
				var d291 JITValueDesc
				_ = d291
				var d292 JITValueDesc
				_ = d292
				var d293 JITValueDesc
				_ = d293
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d296 JITValueDesc
				_ = d296
				var d297 JITValueDesc
				_ = d297
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d300 JITValueDesc
				_ = d300
				var d301 JITValueDesc
				_ = d301
				var d302 JITValueDesc
				_ = d302
				var d303 JITValueDesc
				_ = d303
				var d304 JITValueDesc
				_ = d304
				var d305 JITValueDesc
				_ = d305
				var d306 JITValueDesc
				_ = d306
				var d307 JITValueDesc
				_ = d307
				var d308 JITValueDesc
				_ = d308
				var d309 JITValueDesc
				_ = d309
				var d310 JITValueDesc
				_ = d310
				var d311 JITValueDesc
				_ = d311
				var d312 JITValueDesc
				_ = d312
				var d313 JITValueDesc
				_ = d313
				var stackArray314 int32
				var d315 JITValueDesc
				_ = d315
				var d316 JITValueDesc
				_ = d316
				var d317 JITValueDesc
				_ = d317
				var d318 JITValueDesc
				_ = d318
				var d319 JITValueDesc
				_ = d319
				var d320 JITValueDesc
				_ = d320
				var d321 JITValueDesc
				_ = d321
				var d322 JITValueDesc
				_ = d322
				var d323 JITValueDesc
				_ = d323
				var d324 JITValueDesc
				_ = d324
				var d325 JITValueDesc
				_ = d325
				var d326 JITValueDesc
				_ = d326
				var d327 JITValueDesc
				_ = d327
				var d328 JITValueDesc
				_ = d328
				var d329 JITValueDesc
				_ = d329
				var d330 JITValueDesc
				_ = d330
				var d331 JITValueDesc
				_ = d331
				var d332 JITValueDesc
				_ = d332
				var d333 JITValueDesc
				_ = d333
				var d334 JITValueDesc
				_ = d334
				var d335 JITValueDesc
				_ = d335
				var d336 JITValueDesc
				_ = d336
				var d337 JITValueDesc
				_ = d337
				var d338 JITValueDesc
				_ = d338
				var d339 JITValueDesc
				_ = d339
				var d340 JITValueDesc
				_ = d340
				var d341 JITValueDesc
				_ = d341
				var d342 JITValueDesc
				_ = d342
				var d343 JITValueDesc
				_ = d343
				var d344 JITValueDesc
				_ = d344
				var d345 JITValueDesc
				_ = d345
				var d346 JITValueDesc
				_ = d346
				var d347 JITValueDesc
				_ = d347
				var stackArray348 int32
				var d349 JITValueDesc
				_ = d349
				var d350 JITValueDesc
				_ = d350
				var d352 JITValueDesc
				_ = d352
				var d353 JITValueDesc
				_ = d353
				var d354 JITValueDesc
				_ = d354
				var d355 JITValueDesc
				_ = d355
				var d356 JITValueDesc
				_ = d356
				var d357 JITValueDesc
				_ = d357
				var d358 JITValueDesc
				_ = d358
				var d359 JITValueDesc
				_ = d359
				var d360 JITValueDesc
				_ = d360
				var d361 JITValueDesc
				_ = d361
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d368 JITValueDesc
				_ = d368
				var d369 JITValueDesc
				_ = d369
				var d370 JITValueDesc
				_ = d370
				var d371 JITValueDesc
				_ = d371
				var d372 JITValueDesc
				_ = d372
				var d373 JITValueDesc
				_ = d373
				var d374 JITValueDesc
				_ = d374
				var d375 JITValueDesc
				_ = d375
				var d376 JITValueDesc
				_ = d376
				var d377 JITValueDesc
				_ = d377
				var d378 JITValueDesc
				_ = d378
				var d379 JITValueDesc
				_ = d379
				var d381 JITValueDesc
				_ = d381
				var d383 JITValueDesc
				_ = d383
				var d384 JITValueDesc
				_ = d384
				var d385 JITValueDesc
				_ = d385
				var d386 JITValueDesc
				_ = d386
				var d387 JITValueDesc
				_ = d387
				var d388 JITValueDesc
				_ = d388
				var d389 JITValueDesc
				_ = d389
				var d390 JITValueDesc
				_ = d390
				var d391 JITValueDesc
				_ = d391
				var d392 JITValueDesc
				_ = d392
				var d393 JITValueDesc
				_ = d393
				var d394 JITValueDesc
				_ = d394
				var d395 JITValueDesc
				_ = d395
				var d396 JITValueDesc
				_ = d396
				var d397 JITValueDesc
				_ = d397
				var d398 JITValueDesc
				_ = d398
				var d399 JITValueDesc
				_ = d399
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
						ctx.EmitSetcc(r0, CondSignedGreater)
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
							if ps.General {
							}
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, int32(bbs[2].PhiBase)+int32(0))
						}
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
					ctx.EmitJump(CondNotEqual, lbl4)
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
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					if ps.General {
						ctx.SyncDesc(&d19)
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
						ctx.SyncDesc(&d21)
						if d21.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d21, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d21.Loc == LocInputPair {
							ctx.EnsureDesc(&d21)
							ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
						} else if d21.Loc == LocRegPair || d21.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d21, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d21)
							ctx.EmitStoreToStack(d21, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d19.Loc == LocReg {
							ctx.UnprotectReg(d19.Reg)
						} else if d19.Loc == LocRegPair {
							ctx.UnprotectReg(d19.Reg)
							ctx.UnprotectReg(d19.Reg2)
						}
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
					ctx.EnsureDesc(&d26)
					d28 = d1
					_ = d28
					ctx.StabilizeDescForControlFlow(&d28)
					d29 = d26
					_ = d29
					ctx.StabilizeDescForControlFlow(&d29)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d30 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[]Scmer { return new([]Scmer) }), nil, 1)
					ctx.BindReg(d30.Reg, &d30)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d28.Imm)
						ptrWord, _ := d28.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d28.Imm.String())))
						d28 = tmpPair
					} else if d28.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d28.Type, Reg: ctx.AllocRegExcept(d28.Reg), Reg2: ctx.AllocRegExcept(d28.Reg)}
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d28)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d28)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d28)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d28)
						d28 = tmpPair
					}
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (tokenize arg0)")
					}
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d29.Imm)
						ptrWord, _ := d29.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d29.Imm.String())))
						d29 = tmpPair
					} else if d29.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocRegExcept(d29.Reg), Reg2: ctx.AllocRegExcept(d29.Reg)}
						switch d29.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d29)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d29)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d29)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d29)
						d29 = tmpPair
					}
					if d29.Loc != LocRegPair && d29.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (tokenize arg1)")
					}
					ctx.SyncDesc(&d28)
					ctx.SyncDesc(&d29)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(tokenize), []JITValueDesc{d28, d29}, 3)
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.BindReg(d31.Reg3, &d31)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d31)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[]Scmer, value []Scmer) { *dst = value }), []JITValueDesc{d30, d31})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					d32 = d30
					_ = d32
					ctx.StabilizeDescForControlFlow(&d32)
					phiBase33 = ctx.AllocStack(int32(80))
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					_ = d34
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					_ = d35
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					_ = d36
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					_ = d37
					lbl6 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_5 := int32(-1)
					_ = bbpos_2_5
					bbpos_2_6 := int32(-1)
					_ = bbpos_2_6
					bbpos_2_7 := int32(-1)
					_ = bbpos_2_7
					bbpos_2_8 := int32(-1)
					_ = bbpos_2_8
					bbpos_2_9 := int32(-1)
					_ = bbpos_2_9
					bbpos_2_10 := int32(-1)
					_ = bbpos_2_10
					bbpos_2_11 := int32(-1)
					_ = bbpos_2_11
					bbpos_2_12 := int32(-1)
					_ = bbpos_2_12
					bbpos_2_13 := int32(-1)
					_ = bbpos_2_13
					bbpos_2_14 := int32(-1)
					_ = bbpos_2_14
					bbpos_2_15 := int32(-1)
					_ = bbpos_2_15
					bbpos_2_16 := int32(-1)
					_ = bbpos_2_16
					bbpos_2_17 := int32(-1)
					_ = bbpos_2_17
					bbpos_2_18 := int32(-1)
					_ = bbpos_2_18
					bbpos_2_19 := int32(-1)
					_ = bbpos_2_19
					bbpos_2_20 := int32(-1)
					_ = bbpos_2_20
					bbpos_2_21 := int32(-1)
					_ = bbpos_2_21
					bbpos_2_22 := int32(-1)
					_ = bbpos_2_22
					bbpos_2_23 := int32(-1)
					_ = bbpos_2_23
					bbpos_2_24 := int32(-1)
					_ = bbpos_2_24
					bbpos_2_25 := int32(-1)
					_ = bbpos_2_25
					bbpos_2_26 := int32(-1)
					_ = bbpos_2_26
					bbpos_2_27 := int32(-1)
					_ = bbpos_2_27
					bbpos_2_28 := int32(-1)
					_ = bbpos_2_28
					bbpos_2_29 := int32(-1)
					_ = bbpos_2_29
					bbpos_2_30 := int32(-1)
					_ = bbpos_2_30
					bbpos_2_31 := int32(-1)
					_ = bbpos_2_31
					bbpos_2_32 := int32(-1)
					_ = bbpos_2_32
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d38 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					var d39 JITValueDesc
					if d38.SliceSizeKnown {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.KnownSliceLen))}
					} else if d38.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.StackOff))}
					} else if d38.Loc == LocStackTriple {
						d39 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2, ID: 0}
						} else if d38.Loc == LocReg {
							d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					var d40 JITValueDesc
					if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d39.Imm.Int() == 0)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d39.Reg, 0)
						ctx.EmitSetcc(r1, CondEqual)
						d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d40)
					}
					ctx.FreeDesc(&d39)
					ctx.ReclaimUntrackedRegs()
					d41 = d40
					ctx.EnsureDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitCmpRegImm32(d41.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d40)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d42 = ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
					ctx.BindReg(d42.Reg, &d42)
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.ReclaimUntrackedRegs()
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d46 = ctx.EmitSliceElementAddress(&d43, &d44, 16)
					ctx.EmitLoadScmerToStack(&d46, int32(phiBase33)+int32(0))
					ctx.FreeDesc(&d46)
					d45 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					ctx.StabilizeDescForControlFlow(&d45)
					ctx.ReclaimUntrackedRegs()
					d47 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					var d49 JITValueDesc
					ctx.EnsureDesc(&d47)
					if d47.Loc == LocRegPair || d47.Loc == LocRegTriple {
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg2}
						ctx.BindReg(d47.Reg2, &d49)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d49)
					var d51 JITValueDesc
					if d49.Loc == LocImm && d48.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() - d48.Imm.Int())}
					} else {
						r2 := ctx.AllocReg()
						if d49.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d49.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d49.Reg)
						}
						if d48.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
							ctx.EmitSubInt64(r2, RegR11)
						} else {
							ctx.EmitSubInt64(r2, d48.Reg)
						}
						d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d51)
					}
					var d52 JITValueDesc
					if d47.Loc == LocImm && d48.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() + d48.Imm.Int()*16)}
					} else {
						r3 := ctx.AllocReg()
						if d47.Loc == LocImm {
							ctx.EmitMovRegImm64(r3, uint64(d47.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r3, d47.Reg)
						}
						if d48.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()*16))
							ctx.EmitAddInt64(r3, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r3, d48.Reg)
							ctx.EmitMovRegReg(offsetReg, d48.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r3, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d52)
					}
					var d53 JITValueDesc
					var r4 Reg
					var r5 Reg
					ctx.SyncDesc(&d52)
					ctx.EnsureDesc(&d52)
					if d52.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d52.Imm.Int()))
					} else {
						r4 = d52.Reg
					}
					ctx.ProtectReg(r4)
					ctx.SyncDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d51.Imm.Int()))
					} else {
						r5 = d51.Reg
					}
					ctx.ProtectReg(r5)
					r6 := ctx.EmitSliceCapAfterLow(&d47, &d48, r4, r5)
					ctx.UnprotectReg(r5)
					ctx.UnprotectReg(r4)
					d53 = JITValueDesc{Loc: LocRegTriple, Reg: r4, Reg2: r5, Reg3: r6}
					ctx.BindReg(r4, &d53)
					ctx.BindReg(r5, &d53)
					ctx.BindReg(r6, &d53)
					ctx.BindReg(r4, &d53)
					ctx.BindReg(r5, &d53)
					ctx.BindReg(r6, &d53)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d53)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[]Scmer, value []Scmer) { *dst = value }), []JITValueDesc{d32, d53})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d45)
					r7 := d45.Loc == LocReg || d45.Loc == LocRegPair || d45.Loc == LocRegTriple
					r8 := d45.Reg
					if r7 {
						ctx.ProtectReg(r8)
					}
					r9 := d45.Loc == LocRegPair || d45.Loc == LocRegTriple
					r10 := d45.Reg2
					if r9 {
						ctx.ProtectReg(r10)
					}
					r11 := d45.Loc == LocRegTriple
					r12 := d45.Reg3
					if r11 {
						ctx.ProtectReg(r12)
					}
					lbl11 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r13 := ctx.AllocReg()
					r14 := ctx.AllocRegExcept(r13)
					ctx.EmitMovRegImm64(r13, 0)
					ctx.EmitMovRegImm64(r14, 0)
					d54 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r13, Reg2: r14}
					ctx.BindReg(r13, &d54)
					ctx.BindReg(r14, &d54)
					ctx.StabilizeDescForControlFlow(&d54)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d45)
					ctx.ReclaimUntrackedRegs()
					d55 = args[0]
					d55.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d56 JITValueDesc
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						ptrWord, _ := d55.Imm.RawWords()
						d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d55.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r15 := ctx.AllocReg()
						ctx.EmitMovRegReg(r15, d55.Reg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d56)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d56)
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d58)
					var d57 JITValueDesc
					if d56.Loc == LocImm && d58.Loc == LocImm {
						d57 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d56.Imm.Int() == d58.Imm.Int())}
					} else if d58.Loc == LocImm {
						r16 := ctx.AllocReg()
						if d58.Imm.Int() >= -2147483648 && d58.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d56.Reg, int32(d58.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d58.Imm.Int()))
							ctx.EmitCmpInt64(d56.Reg, RegR11)
						}
						ctx.EmitSetcc(r16, CondEqual)
						d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
						ctx.BindReg(r16, &d57)
					} else if d56.Loc == LocImm {
						r17 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d56.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d58.Reg)
						ctx.EmitSetcc(r17, CondEqual)
						d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
						ctx.BindReg(r17, &d57)
					} else {
						r18 := ctx.AllocReg()
						ctx.EmitCmpInt64(d56.Reg, d58.Reg)
						ctx.EmitSetcc(r18, CondEqual)
						d57 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
						ctx.BindReg(r18, &d57)
					}
					ctx.FreeDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					d59 = d57
					ctx.EnsureDesc(&d59)
					if d59.Loc != LocImm && d59.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d59.Loc == LocImm {
						if d59.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitCmpRegImm32(d59.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d57)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d60 = args[0]
					d60.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d61 JITValueDesc
					ctx.EnsureDesc(&d60)
					if d60.Loc == LocImm {
						ptrWord, _ := d60.Imm.RawWords()
						d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d60.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r19 := ctx.AllocReg()
						ctx.EmitMovRegReg(r19, d60.Reg)
						d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d61)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d61)
					d63 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d61)
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d61)
					ctx.EnsureDesc(&d63)
					var d62 JITValueDesc
					if d61.Loc == LocImm && d63.Loc == LocImm {
						d62 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d61.Imm.Int() == d63.Imm.Int())}
					} else if d63.Loc == LocImm {
						r20 := ctx.AllocReg()
						if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d61.Reg, int32(d63.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d63.Imm.Int()))
							ctx.EmitCmpInt64(d61.Reg, RegR11)
						}
						ctx.EmitSetcc(r20, CondEqual)
						d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d62)
					} else if d61.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d61.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d63.Reg)
						ctx.EmitSetcc(r21, CondEqual)
						d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d62)
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpInt64(d61.Reg, d63.Reg)
						ctx.EmitSetcc(r22, CondEqual)
						d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d62)
					}
					ctx.FreeDesc(&d61)
					ctx.ReclaimUntrackedRegs()
					d64 = d62
					ctx.EnsureDesc(&d64)
					if d64.Loc != LocImm && d64.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					if d64.Loc == LocImm {
						if d64.Imm.Bool() {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl16)
						}
					} else {
						ctx.EmitCmpRegImm32(d64.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl17)
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl16)
					}
					ctx.FreeDesc(&d62)
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d65 = args[0]
					d65.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d66 JITValueDesc
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						_, auxWord := d65.Imm.RawWords()
						d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d65.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r23 := ctx.AllocReg()
						ctx.EmitMovRegReg(r23, d65.Reg2)
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
						ctx.BindReg(r23, &d66)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d66)
					d67 = d66
					_ = d67
					ctx.StabilizeDescForControlFlow(&d67)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d67)
					var d68 JITValueDesc
					if d67.Loc == LocImm {
						d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d67.Imm.Int() & 255)}
					} else {
						r24 := ctx.AllocRegExcept(d67.Reg)
						ctx.EmitMovRegReg(r24, d67.Reg)
						ctx.EmitAndRegImm32(r24, int32(255))
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
						ctx.BindReg(r24, &d68)
					}
					if d68.Loc == LocReg && d67.Loc == LocReg && d68.Reg == d67.Reg {
						ctx.TransferReg(d67.Reg)
						d67.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d68)
					ctx.EnsureDesc(&d68)
					var d69 JITValueDesc
					if d68.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d68.Imm.Int()))))}
					} else {
						r25 := ctx.AllocReg()
						ctx.EmitMovRegReg(r25, d68.Reg)
						ctx.EmitShlRegImm8(r25, 56)
						ctx.EmitShrRegImm8(r25, 56)
						d69 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d69)
					}
					ctx.FreeDesc(&d68)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					ctx.FreeDesc(&d66)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					var d70 JITValueDesc
					if d69.Loc == LocImm {
						d70 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d69.Imm.Int()) == uint64(0xe))}
					} else {
						r26 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d69.Reg, 14)
						ctx.EmitSetcc(r26, CondEqual)
						d70 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d70)
					}
					ctx.FreeDesc(&d69)
					ctx.ReclaimUntrackedRegs()
					r27 := ctx.AllocReg()
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r27, d70)
					}
					ctx.EmitJmp(lbl11)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d71 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r27, d71)
					}
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl11)
					d72 = JITValueDesc{Loc: LocReg, Reg: r27}
					ctx.BindReg(r27, &d72)
					ctx.BindReg(r27, &d72)
					if r7 {
						ctx.UnprotectReg(r8)
					}
					if r9 {
						ctx.UnprotectReg(r10)
					}
					if r11 {
						ctx.UnprotectReg(r12)
					}
					ctx.ReclaimUntrackedRegs()
					d73 = d72
					ctx.EnsureDesc(&d73)
					if d73.Loc != LocImm && d73.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					if d73.Loc == LocImm {
						if d73.Imm.Bool() {
							ctx.MarkLabel(lbl21)
							ctx.EmitJmp(lbl19)
						} else {
							ctx.MarkLabel(lbl22)
							ctx.SyncDesc(&d45)
							if d45.Loc == LocReg {
								ctx.ProtectReg(d45.Reg)
							} else if d45.Loc == LocRegPair {
								ctx.ProtectReg(d45.Reg)
								ctx.ProtectReg(d45.Reg2)
							}
							d74 = d45
							if d74.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d74)
							if d74.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d74, int32(phiBase33)+int32(0), 2)
							} else if d74.Loc == LocInputPair {
								ctx.EnsureDesc(&d74)
								ctx.EmitStoreScmerToStack(d74, int32(phiBase33)+int32(0))
							} else if d74.Loc == LocRegPair || d74.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d74, int32(phiBase33)+int32(0))
							} else {
								ctx.EnsureDesc(&d74)
								ctx.EmitStoreToStack(d74, int32(phiBase33)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(0))+8)
							}
							if d45.Loc == LocReg {
								ctx.UnprotectReg(d45.Reg)
							} else if d45.Loc == LocRegPair {
								ctx.UnprotectReg(d45.Reg)
								ctx.UnprotectReg(d45.Reg2)
							}
							ctx.EmitJmp(lbl20)
						}
					} else {
						ctx.EmitCmpRegImm32(d73.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl21)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl19)
						ctx.MarkLabel(lbl22)
						ctx.SyncDesc(&d45)
						if d45.Loc == LocReg {
							ctx.ProtectReg(d45.Reg)
						} else if d45.Loc == LocRegPair {
							ctx.ProtectReg(d45.Reg)
							ctx.ProtectReg(d45.Reg2)
						}
						d75 = d45
						if d75.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d75)
						if d75.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d75, int32(phiBase33)+int32(0), 2)
						} else if d75.Loc == LocInputPair {
							ctx.EnsureDesc(&d75)
							ctx.EmitStoreScmerToStack(d75, int32(phiBase33)+int32(0))
						} else if d75.Loc == LocRegPair || d75.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d75, int32(phiBase33)+int32(0))
						} else {
							ctx.EnsureDesc(&d75)
							ctx.EmitStoreToStack(d75, int32(phiBase33)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(0))+8)
						}
						if d45.Loc == LocReg {
							ctx.UnprotectReg(d45.Reg)
						} else if d45.Loc == LocRegPair {
							ctx.UnprotectReg(d45.Reg)
							ctx.UnprotectReg(d45.Reg2)
						}
						ctx.EmitJmp(lbl20)
					}
					ctx.FreeDesc(&d72)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					r28 := d34.Loc == LocReg || d34.Loc == LocRegPair || d34.Loc == LocRegTriple
					r29 := d34.Reg
					if r28 {
						ctx.ProtectReg(r29)
					}
					r30 := d34.Loc == LocRegPair || d34.Loc == LocRegTriple
					r31 := d34.Reg2
					if r30 {
						ctx.ProtectReg(r31)
					}
					r32 := d34.Loc == LocRegTriple
					r33 := d34.Reg3
					if r32 {
						ctx.ProtectReg(r33)
					}
					lbl23 := ctx.ReserveLabel()
					bbpos_5_0 := int32(-1)
					_ = bbpos_5_0
					bbpos_5_1 := int32(-1)
					_ = bbpos_5_1
					bbpos_5_2 := int32(-1)
					_ = bbpos_5_2
					bbpos_5_3 := int32(-1)
					_ = bbpos_5_3
					bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r34 := ctx.AllocReg()
					r35 := ctx.AllocRegExcept(r34)
					ctx.EmitMovRegImm64(r34, 0)
					ctx.EmitMovRegImm64(r35, 0)
					d76 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r34, Reg2: r35}
					ctx.BindReg(r34, &d76)
					ctx.BindReg(r35, &d76)
					ctx.StabilizeDescForControlFlow(&d76)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d34)
					ctx.ReclaimUntrackedRegs()
					d77 = args[0]
					d77.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d78 JITValueDesc
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocImm {
						ptrWord, _ := d77.Imm.RawWords()
						d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d77.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r36 := ctx.AllocReg()
						ctx.EmitMovRegReg(r36, d77.Reg)
						d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r36}
						ctx.BindReg(r36, &d78)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d80)
					var d79 JITValueDesc
					if d78.Loc == LocImm && d80.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d78.Imm.Int() == d80.Imm.Int())}
					} else if d80.Loc == LocImm {
						r37 := ctx.AllocReg()
						if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d78.Reg, int32(d80.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d80.Imm.Int()))
							ctx.EmitCmpInt64(d78.Reg, RegR11)
						}
						ctx.EmitSetcc(r37, CondEqual)
						d79 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r37}
						ctx.BindReg(r37, &d79)
					} else if d78.Loc == LocImm {
						r38 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d78.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d80.Reg)
						ctx.EmitSetcc(r38, CondEqual)
						d79 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r38}
						ctx.BindReg(r38, &d79)
					} else {
						r39 := ctx.AllocReg()
						ctx.EmitCmpInt64(d78.Reg, d80.Reg)
						ctx.EmitSetcc(r39, CondEqual)
						d79 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r39}
						ctx.BindReg(r39, &d79)
					}
					ctx.FreeDesc(&d78)
					ctx.ReclaimUntrackedRegs()
					d81 = d79
					ctx.EnsureDesc(&d81)
					if d81.Loc != LocImm && d81.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					if d81.Loc == LocImm {
						if d81.Imm.Bool() {
							ctx.MarkLabel(lbl26)
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl27)
							ctx.EmitJmp(lbl25)
						}
					} else {
						ctx.EmitCmpRegImm32(d81.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl26)
						ctx.EmitJmp(lbl27)
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl27)
						ctx.EmitJmp(lbl25)
					}
					ctx.FreeDesc(&d79)
					bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl25)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d82 = args[0]
					d82.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d83 JITValueDesc
					ctx.EnsureDesc(&d82)
					if d82.Loc == LocImm {
						ptrWord, _ := d82.Imm.RawWords()
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d82.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r40 := ctx.AllocReg()
						ctx.EmitMovRegReg(r40, d82.Reg)
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r40}
						ctx.BindReg(r40, &d83)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d83)
					d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d85)
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d85)
					var d84 JITValueDesc
					if d83.Loc == LocImm && d85.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d83.Imm.Int() == d85.Imm.Int())}
					} else if d85.Loc == LocImm {
						r41 := ctx.AllocReg()
						if d85.Imm.Int() >= -2147483648 && d85.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d83.Reg, int32(d85.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d85.Imm.Int()))
							ctx.EmitCmpInt64(d83.Reg, RegR11)
						}
						ctx.EmitSetcc(r41, CondEqual)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r41}
						ctx.BindReg(r41, &d84)
					} else if d83.Loc == LocImm {
						r42 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d83.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d85.Reg)
						ctx.EmitSetcc(r42, CondEqual)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r42}
						ctx.BindReg(r42, &d84)
					} else {
						r43 := ctx.AllocReg()
						ctx.EmitCmpInt64(d83.Reg, d85.Reg)
						ctx.EmitSetcc(r43, CondEqual)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r43}
						ctx.BindReg(r43, &d84)
					}
					ctx.FreeDesc(&d83)
					ctx.ReclaimUntrackedRegs()
					d86 = d84
					ctx.EnsureDesc(&d86)
					if d86.Loc != LocImm && d86.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					lbl30 := ctx.ReserveLabel()
					if d86.Loc == LocImm {
						if d86.Imm.Bool() {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl24)
						} else {
							ctx.MarkLabel(lbl30)
							ctx.EmitJmp(lbl28)
						}
					} else {
						ctx.EmitCmpRegImm32(d86.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl29)
						ctx.EmitJmp(lbl30)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl24)
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl28)
					}
					ctx.FreeDesc(&d84)
					bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d87 = args[0]
					d87.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d88 JITValueDesc
					ctx.EnsureDesc(&d87)
					if d87.Loc == LocImm {
						_, auxWord := d87.Imm.RawWords()
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d87.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r44 := ctx.AllocReg()
						ctx.EmitMovRegReg(r44, d87.Reg2)
						d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r44}
						ctx.BindReg(r44, &d88)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					d89 = d88
					_ = d89
					ctx.StabilizeDescForControlFlow(&d89)
					bbpos_6_0 := int32(-1)
					_ = bbpos_6_0
					bbpos_6_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					var d90 JITValueDesc
					if d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d89.Imm.Int() & 255)}
					} else {
						r45 := ctx.AllocRegExcept(d89.Reg)
						ctx.EmitMovRegReg(r45, d89.Reg)
						ctx.EmitAndRegImm32(r45, int32(255))
						d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r45}
						ctx.BindReg(r45, &d90)
					}
					if d90.Loc == LocReg && d89.Loc == LocReg && d90.Reg == d89.Reg {
						ctx.TransferReg(d89.Reg)
						d89.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					var d91 JITValueDesc
					if d90.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d90.Imm.Int()))))}
					} else {
						r46 := ctx.AllocReg()
						ctx.EmitMovRegReg(r46, d90.Reg)
						ctx.EmitShlRegImm8(r46, 56)
						ctx.EmitShrRegImm8(r46, 56)
						d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r46}
						ctx.BindReg(r46, &d91)
					}
					ctx.FreeDesc(&d90)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					ctx.FreeDesc(&d88)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					var d92 JITValueDesc
					if d91.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d91.Imm.Int()) == uint64(0x2))}
					} else {
						r47 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d91.Reg, 2)
						ctx.EmitSetcc(r47, CondEqual)
						d92 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r47}
						ctx.BindReg(r47, &d92)
					}
					ctx.FreeDesc(&d91)
					ctx.ReclaimUntrackedRegs()
					r48 := ctx.AllocReg()
					ctx.EnsureDesc(&d92)
					ctx.EnsureDesc(&d92)
					if d92.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r48, d92)
					}
					ctx.EmitJmp(lbl23)
					bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d93 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d93)
					if d93.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r48, d93)
					}
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl23)
					d94 = JITValueDesc{Loc: LocReg, Reg: r48}
					ctx.BindReg(r48, &d94)
					ctx.BindReg(r48, &d94)
					if r28 {
						ctx.UnprotectReg(r29)
					}
					if r30 {
						ctx.UnprotectReg(r31)
					}
					if r32 {
						ctx.UnprotectReg(r33)
					}
					ctx.ReclaimUntrackedRegs()
					d95 = d94
					ctx.EnsureDesc(&d95)
					if d95.Loc != LocImm && d95.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl31 := ctx.ReserveLabel()
					lbl32 := ctx.ReserveLabel()
					lbl33 := ctx.ReserveLabel()
					lbl34 := ctx.ReserveLabel()
					if d95.Loc == LocImm {
						if d95.Imm.Bool() {
							ctx.MarkLabel(lbl33)
							ctx.EmitJmp(lbl31)
						} else {
							ctx.MarkLabel(lbl34)
							ctx.EmitJmp(lbl32)
						}
					} else {
						ctx.EmitCmpRegImm32(d95.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl33)
						ctx.EmitJmp(lbl34)
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl31)
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl32)
					}
					ctx.FreeDesc(&d94)
					bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl32)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r49 := ctx.AllocReg()
					r50 := ctx.AllocRegExcept(r49)
					d96 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d96)
					ctx.BindReg(r50, &d96)
					ctx.EmitMovPairToResult(&d34, &d96)
					ctx.EmitJmp(lbl6)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d97 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.ReclaimUntrackedRegs()
					d98 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d98)
					ctx.BindReg(r50, &d98)
					ctx.EmitMovPairToResult(&d97, &d98)
					ctx.EmitJmp(lbl6)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d45)
					r51 := d45.Loc == LocReg || d45.Loc == LocRegPair || d45.Loc == LocRegTriple
					r52 := d45.Reg
					if r51 {
						ctx.ProtectReg(r52)
					}
					r53 := d45.Loc == LocRegPair || d45.Loc == LocRegTriple
					r54 := d45.Reg2
					if r53 {
						ctx.ProtectReg(r54)
					}
					r55 := d45.Loc == LocRegTriple
					r56 := d45.Reg3
					if r55 {
						ctx.ProtectReg(r56)
					}
					lbl35 := ctx.ReserveLabel()
					bbpos_7_0 := int32(-1)
					_ = bbpos_7_0
					bbpos_7_1 := int32(-1)
					_ = bbpos_7_1
					bbpos_7_2 := int32(-1)
					_ = bbpos_7_2
					bbpos_7_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r57 := ctx.AllocReg()
					r58 := ctx.AllocRegExcept(r57)
					ctx.EmitMovRegImm64(r57, 0)
					ctx.EmitMovRegImm64(r58, 0)
					d99 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r57, Reg2: r58}
					ctx.BindReg(r57, &d99)
					ctx.BindReg(r58, &d99)
					ctx.StabilizeDescForControlFlow(&d99)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d45)
					ctx.ReclaimUntrackedRegs()
					d100 = d45
					_ = d100
					ctx.ReclaimUntrackedRegs()
					d101 = ctx.EmitGetTagDesc(&d100, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d101)
					var d102 JITValueDesc
					if d101.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d101.Imm.Int()) != uint64(0xe))}
					} else {
						r59 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d101.Reg, 14)
						ctx.EmitSetcc(r59, CondNotEqual)
						d102 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r59}
						ctx.BindReg(r59, &d102)
					}
					ctx.FreeDesc(&d101)
					ctx.ReclaimUntrackedRegs()
					d103 = d102
					ctx.EnsureDesc(&d103)
					if d103.Loc != LocImm && d103.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl36 := ctx.ReserveLabel()
					lbl37 := ctx.ReserveLabel()
					lbl38 := ctx.ReserveLabel()
					lbl39 := ctx.ReserveLabel()
					if d103.Loc == LocImm {
						if d103.Imm.Bool() {
							ctx.MarkLabel(lbl38)
							ctx.EmitJmp(lbl36)
						} else {
							ctx.MarkLabel(lbl39)
							ctx.EmitJmp(lbl37)
						}
					} else {
						ctx.EmitCmpRegImm32(d103.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl38)
						ctx.EmitJmp(lbl39)
						ctx.MarkLabel(lbl38)
						ctx.EmitJmp(lbl36)
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl37)
					}
					ctx.FreeDesc(&d102)
					bbpos_7_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl37)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d104 = args[0]
					d104.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d105 JITValueDesc
					ctx.EnsureDesc(&d104)
					if d104.Loc == LocImm {
						ptrWord, _ := d104.Imm.RawWords()
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d104.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r60 := ctx.AllocReg()
						ctx.EmitMovRegReg(r60, d104.Reg)
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r60}
						ctx.BindReg(r60, &d105)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					r61 := ctx.AllocReg()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r61, d105)
					}
					ctx.EmitJmp(lbl35)
					bbpos_7_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl36)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl35)
					d108 = JITValueDesc{Loc: LocReg, Reg: r61}
					ctx.BindReg(r61, &d108)
					ctx.BindReg(r61, &d108)
					if r51 {
						ctx.UnprotectReg(r52)
					}
					if r53 {
						ctx.UnprotectReg(r54)
					}
					if r55 {
						ctx.UnprotectReg(r56)
					}
					ctx.ReclaimUntrackedRegs()
					d109 = args[0]
					d109.ID = 0
					ctx.FreeDesc(&d108)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d109)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *SourceInfo, value SourceInfo) { *dst = value }), []JITValueDesc{d42, d109})
					ctx.FreeDesc(&d109)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d110 JITValueDesc
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						fieldAddr := uintptr(d42.Imm.Int()) + 32
						r62 := ctx.AllocReg()
						r63 := ctx.AllocRegExcept(r62)
						ctx.EmitMovRegMem64(r62, fieldAddr)
						ctx.EmitMovRegMem64(r63, fieldAddr+8)
						d110 = JITValueDesc{Loc: LocRegPair, Reg: r62, Reg2: r63}
						ctx.BindReg(r62, &d110)
						ctx.BindReg(r63, &d110)
					} else {
						off := int32(32)
						baseReg := d42.Reg
						r64 := ctx.AllocRegExcept(baseReg)
						r65 := ctx.AllocRegExcept(baseReg, r64)
						ctx.EmitMovRegMem(r64, baseReg, off)
						ctx.EmitMovRegMem(r65, baseReg, off+8)
						d110 = JITValueDesc{Loc: LocRegPair, Reg: r64, Reg2: r65}
						ctx.BindReg(r64, &d110)
						ctx.BindReg(r65, &d110)
					}
					ctx.StabilizeDescForControlFlow(&d110)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d110)
					if d110.Loc == LocReg {
						ctx.ProtectReg(d110.Reg)
					} else if d110.Loc == LocRegPair {
						ctx.ProtectReg(d110.Reg)
						ctx.ProtectReg(d110.Reg2)
					}
					d111 = d110
					if d111.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d111)
					if d111.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d111, int32(phiBase33)+int32(0), 2)
					} else if d111.Loc == LocInputPair {
						ctx.EnsureDesc(&d111)
						ctx.EmitStoreScmerToStack(d111, int32(phiBase33)+int32(0))
					} else if d111.Loc == LocRegPair || d111.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d111, int32(phiBase33)+int32(0))
					} else {
						ctx.EnsureDesc(&d111)
						ctx.EmitStoreToStack(d111, int32(phiBase33)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(0))+8)
					}
					if d110.Loc == LocReg {
						ctx.UnprotectReg(d110.Reg)
					} else if d110.Loc == LocRegPair {
						ctx.UnprotectReg(d110.Reg)
						ctx.UnprotectReg(d110.Reg2)
					}
					ctx.EmitJmp(lbl20)
					bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl31)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d113 = d34
					ctx.EnsureDesc(&d113)
					if d113.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d113.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d113)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d113)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d113)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d113.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d113 = tmpPair
					} else if d113.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d113.Reg), Reg2: ctx.AllocRegExcept(d113.Reg)}
						switch d113.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d113)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d113)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d113)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d113)
						d113 = tmpPair
					} else if d113.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d113.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d113.MemPtr))
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
						d113 = tmpPair
					}
					if d113.Loc != LocRegPair && d113.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d112 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d113}, 2)
					ctx.StabilizeDescForControlFlow(&d112)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d112)
					d114 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("(")}
					var d115 JITValueDesc
					if d114.Loc == LocImm {
						ctx.TrackImm(d114.Imm)
						ptrWord, _ := d114.Imm.RawWords()
						d115 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d115.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d115.Reg2, uint64(len(d114.Imm.String())))
						ctx.BindReg(d115.Reg, &d115)
						ctx.BindReg(d115.Reg2, &d115)
					} else {
						d115 = d114
					}
					d116 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d112, d115}, 1)
					ctx.EmitAndRegImm32(d116.Reg, 1)
					d116.Type = tagBool
					ctx.BindReg(d116.Reg, &d116)
					ctx.ReclaimUntrackedRegs()
					d117 = d116
					ctx.EnsureDesc(&d117)
					if d117.Loc != LocImm && d117.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl40 := ctx.ReserveLabel()
					lbl41 := ctx.ReserveLabel()
					lbl42 := ctx.ReserveLabel()
					lbl43 := ctx.ReserveLabel()
					if d117.Loc == LocImm {
						if d117.Imm.Bool() {
							ctx.MarkLabel(lbl42)
							ctx.EmitJmp(lbl40)
						} else {
							ctx.MarkLabel(lbl43)
							ctx.EmitJmp(lbl41)
						}
					} else {
						ctx.EmitCmpRegImm32(d117.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl42)
						ctx.EmitJmp(lbl43)
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl40)
						ctx.MarkLabel(lbl43)
						ctx.EmitJmp(lbl41)
					}
					ctx.FreeDesc(&d116)
					bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl41)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d112)
					d118 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("'")}
					var d119 JITValueDesc
					if d118.Loc == LocImm {
						ctx.TrackImm(d118.Imm)
						ptrWord, _ := d118.Imm.RawWords()
						d119 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d119.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d119.Reg2, uint64(len(d118.Imm.String())))
						ctx.BindReg(d119.Reg, &d119)
						ctx.BindReg(d119.Reg2, &d119)
					} else {
						d119 = d118
					}
					d120 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d112, d119}, 1)
					ctx.EmitAndRegImm32(d120.Reg, 1)
					d120.Type = tagBool
					ctx.BindReg(d120.Reg, &d120)
					ctx.ReclaimUntrackedRegs()
					d121 = d120
					ctx.EnsureDesc(&d121)
					if d121.Loc != LocImm && d121.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl44 := ctx.ReserveLabel()
					lbl45 := ctx.ReserveLabel()
					lbl46 := ctx.ReserveLabel()
					lbl47 := ctx.ReserveLabel()
					if d121.Loc == LocImm {
						if d121.Imm.Bool() {
							ctx.MarkLabel(lbl46)
							ctx.EmitJmp(lbl44)
						} else {
							ctx.MarkLabel(lbl47)
							ctx.EmitJmp(lbl45)
						}
					} else {
						ctx.EmitCmpRegImm32(d121.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl46)
						ctx.EmitJmp(lbl47)
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl44)
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl45)
					}
					ctx.FreeDesc(&d120)
					bbpos_2_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl45)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d122 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d122)
					ctx.BindReg(r50, &d122)
					ctx.EmitMovPairToResult(&d34, &d122)
					ctx.EmitJmp(lbl6)
					bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl40)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray123 = ctx.AllocStack(int32(0))
					_ = stackArray123
					ctx.ReclaimUntrackedRegs()
					d124 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d124
					ctx.StabilizeDescForControlFlow(&d124)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d124)
					if d124.Loc == LocReg {
						ctx.ProtectReg(d124.Reg)
					} else if d124.Loc == LocRegPair {
						ctx.ProtectReg(d124.Reg)
						ctx.ProtectReg(d124.Reg2)
					}
					d125 = d124
					if d125.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d125)
					if d125.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d125, int32(phiBase33)+int32(16), 3)
					} else {
						if d125.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d125.Reg, RegRSP, int32(phiBase33)+int32(16))
						ctx.EmitStoreRegMem(d125.Reg2, RegRSP, int32(phiBase33)+int32(16)+8)
						ctx.EmitStoreRegMem(d125.Reg3, RegRSP, int32(phiBase33)+int32(16)+16)
					}
					if d124.Loc == LocReg {
						ctx.UnprotectReg(d124.Reg)
					} else if d124.Loc == LocRegPair {
						ctx.UnprotectReg(d124.Reg)
						ctx.UnprotectReg(d124.Reg2)
					}
					bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.ReclaimUntrackedRegs()
					d126 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					var d127 JITValueDesc
					if d126.SliceSizeKnown {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d126.KnownSliceLen))}
					} else if d126.Loc == LocImm {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d126.StackOff))}
					} else if d126.Loc == LocStackTriple {
						d127 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d126.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d126)
						if d126.Loc == LocRegPair || d126.Loc == LocRegTriple {
							d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg2, ID: 0}
						} else if d126.Loc == LocReg {
							d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d126.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d127)
					var d128 JITValueDesc
					if d127.Loc == LocImm {
						d128 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d127.Imm.Int() == 0)}
					} else {
						r66 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d127.Reg, 0)
						ctx.EmitSetcc(r66, CondEqual)
						d128 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r66}
						ctx.BindReg(r66, &d128)
					}
					ctx.FreeDesc(&d127)
					ctx.ReclaimUntrackedRegs()
					d129 = d128
					ctx.EnsureDesc(&d129)
					if d129.Loc != LocImm && d129.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl48 := ctx.ReserveLabel()
					lbl49 := ctx.ReserveLabel()
					lbl50 := ctx.ReserveLabel()
					lbl51 := ctx.ReserveLabel()
					if d129.Loc == LocImm {
						if d129.Imm.Bool() {
							ctx.MarkLabel(lbl50)
							ctx.EmitJmp(lbl48)
						} else {
							ctx.MarkLabel(lbl51)
							ctx.EmitJmp(lbl49)
						}
					} else {
						ctx.EmitCmpRegImm32(d129.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl50)
						ctx.EmitJmp(lbl51)
						ctx.MarkLabel(lbl50)
						ctx.EmitJmp(lbl48)
						ctx.MarkLabel(lbl51)
						ctx.EmitJmp(lbl49)
					}
					ctx.FreeDesc(&d128)
					bbpos_2_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl49)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d130 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d133 = ctx.EmitSliceElementAddress(&d130, &d131, 16)
					ctx.EnsureDesc(&d133)
					r67 := ctx.AllocRegExcept(d133.Reg)
					ctx.EmitMovRegMem(r67, d133.Reg, 8)
					ctx.EmitMovRegMem(d133.Reg, d133.Reg, 0)
					d132 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d133.Reg, Reg2: r67}
					ctx.BindReg(d133.Reg, &d132)
					ctx.BindReg(r67, &d132)
					ctx.StabilizeDescForControlFlow(&d132)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d132)
					r68 := d132.Loc == LocReg || d132.Loc == LocRegPair || d132.Loc == LocRegTriple
					r69 := d132.Reg
					if r68 {
						ctx.ProtectReg(r69)
					}
					r70 := d132.Loc == LocRegPair || d132.Loc == LocRegTriple
					r71 := d132.Reg2
					if r70 {
						ctx.ProtectReg(r71)
					}
					r72 := d132.Loc == LocRegTriple
					r73 := d132.Reg3
					if r72 {
						ctx.ProtectReg(r73)
					}
					lbl52 := ctx.ReserveLabel()
					bbpos_8_0 := int32(-1)
					_ = bbpos_8_0
					bbpos_8_1 := int32(-1)
					_ = bbpos_8_1
					bbpos_8_2 := int32(-1)
					_ = bbpos_8_2
					bbpos_8_3 := int32(-1)
					_ = bbpos_8_3
					bbpos_8_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r74 := ctx.AllocReg()
					r75 := ctx.AllocRegExcept(r74)
					ctx.EmitMovRegImm64(r74, 0)
					ctx.EmitMovRegImm64(r75, 0)
					d134 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r74, Reg2: r75}
					ctx.BindReg(r74, &d134)
					ctx.BindReg(r75, &d134)
					ctx.StabilizeDescForControlFlow(&d134)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d132)
					ctx.ReclaimUntrackedRegs()
					d135 = args[0]
					d135.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d136 JITValueDesc
					ctx.EnsureDesc(&d135)
					if d135.Loc == LocImm {
						ptrWord, _ := d135.Imm.RawWords()
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d135.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r76 := ctx.AllocReg()
						ctx.EmitMovRegReg(r76, d135.Reg)
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r76}
						ctx.BindReg(r76, &d136)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d136)
					d138 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d138)
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d138)
					var d137 JITValueDesc
					if d136.Loc == LocImm && d138.Loc == LocImm {
						d137 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d136.Imm.Int() == d138.Imm.Int())}
					} else if d138.Loc == LocImm {
						r77 := ctx.AllocReg()
						if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d136.Reg, int32(d138.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d138.Imm.Int()))
							ctx.EmitCmpInt64(d136.Reg, RegR11)
						}
						ctx.EmitSetcc(r77, CondEqual)
						d137 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r77}
						ctx.BindReg(r77, &d137)
					} else if d136.Loc == LocImm {
						r78 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d136.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d138.Reg)
						ctx.EmitSetcc(r78, CondEqual)
						d137 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r78}
						ctx.BindReg(r78, &d137)
					} else {
						r79 := ctx.AllocReg()
						ctx.EmitCmpInt64(d136.Reg, d138.Reg)
						ctx.EmitSetcc(r79, CondEqual)
						d137 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r79}
						ctx.BindReg(r79, &d137)
					}
					ctx.FreeDesc(&d136)
					ctx.ReclaimUntrackedRegs()
					d139 = d137
					ctx.EnsureDesc(&d139)
					if d139.Loc != LocImm && d139.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl53 := ctx.ReserveLabel()
					lbl54 := ctx.ReserveLabel()
					lbl55 := ctx.ReserveLabel()
					lbl56 := ctx.ReserveLabel()
					if d139.Loc == LocImm {
						if d139.Imm.Bool() {
							ctx.MarkLabel(lbl55)
							ctx.EmitJmp(lbl53)
						} else {
							ctx.MarkLabel(lbl56)
							ctx.EmitJmp(lbl54)
						}
					} else {
						ctx.EmitCmpRegImm32(d139.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl55)
						ctx.EmitJmp(lbl56)
						ctx.MarkLabel(lbl55)
						ctx.EmitJmp(lbl53)
						ctx.MarkLabel(lbl56)
						ctx.EmitJmp(lbl54)
					}
					ctx.FreeDesc(&d137)
					bbpos_8_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl54)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d140 = args[0]
					d140.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d141 JITValueDesc
					ctx.EnsureDesc(&d140)
					if d140.Loc == LocImm {
						ptrWord, _ := d140.Imm.RawWords()
						d141 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d140.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r80 := ctx.AllocReg()
						ctx.EmitMovRegReg(r80, d140.Reg)
						d141 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r80}
						ctx.BindReg(r80, &d141)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d141)
					d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d143)
					var d142 JITValueDesc
					if d141.Loc == LocImm && d143.Loc == LocImm {
						d142 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d141.Imm.Int() == d143.Imm.Int())}
					} else if d143.Loc == LocImm {
						r81 := ctx.AllocReg()
						if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d141.Reg, int32(d143.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d143.Imm.Int()))
							ctx.EmitCmpInt64(d141.Reg, RegR11)
						}
						ctx.EmitSetcc(r81, CondEqual)
						d142 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r81}
						ctx.BindReg(r81, &d142)
					} else if d141.Loc == LocImm {
						r82 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d141.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d143.Reg)
						ctx.EmitSetcc(r82, CondEqual)
						d142 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r82}
						ctx.BindReg(r82, &d142)
					} else {
						r83 := ctx.AllocReg()
						ctx.EmitCmpInt64(d141.Reg, d143.Reg)
						ctx.EmitSetcc(r83, CondEqual)
						d142 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r83}
						ctx.BindReg(r83, &d142)
					}
					ctx.FreeDesc(&d141)
					ctx.ReclaimUntrackedRegs()
					d144 = d142
					ctx.EnsureDesc(&d144)
					if d144.Loc != LocImm && d144.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl57 := ctx.ReserveLabel()
					lbl58 := ctx.ReserveLabel()
					lbl59 := ctx.ReserveLabel()
					if d144.Loc == LocImm {
						if d144.Imm.Bool() {
							ctx.MarkLabel(lbl58)
							ctx.EmitJmp(lbl53)
						} else {
							ctx.MarkLabel(lbl59)
							ctx.EmitJmp(lbl57)
						}
					} else {
						ctx.EmitCmpRegImm32(d144.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl58)
						ctx.EmitJmp(lbl59)
						ctx.MarkLabel(lbl58)
						ctx.EmitJmp(lbl53)
						ctx.MarkLabel(lbl59)
						ctx.EmitJmp(lbl57)
					}
					ctx.FreeDesc(&d142)
					bbpos_8_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl57)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d145 = args[0]
					d145.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d146 JITValueDesc
					ctx.EnsureDesc(&d145)
					if d145.Loc == LocImm {
						_, auxWord := d145.Imm.RawWords()
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d145.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r84 := ctx.AllocReg()
						ctx.EmitMovRegReg(r84, d145.Reg2)
						d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r84}
						ctx.BindReg(r84, &d146)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d146)
					d147 = d146
					_ = d147
					ctx.StabilizeDescForControlFlow(&d147)
					bbpos_9_0 := int32(-1)
					_ = bbpos_9_0
					bbpos_9_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d147)
					var d148 JITValueDesc
					if d147.Loc == LocImm {
						d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d147.Imm.Int() & 255)}
					} else {
						r85 := ctx.AllocRegExcept(d147.Reg)
						ctx.EmitMovRegReg(r85, d147.Reg)
						ctx.EmitAndRegImm32(r85, int32(255))
						d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r85}
						ctx.BindReg(r85, &d148)
					}
					if d148.Loc == LocReg && d147.Loc == LocReg && d148.Reg == d147.Reg {
						ctx.TransferReg(d147.Reg)
						d147.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d148)
					var d149 JITValueDesc
					if d148.Loc == LocImm {
						d149 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d148.Imm.Int()))))}
					} else {
						r86 := ctx.AllocReg()
						ctx.EmitMovRegReg(r86, d148.Reg)
						ctx.EmitShlRegImm8(r86, 56)
						ctx.EmitShrRegImm8(r86, 56)
						d149 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r86}
						ctx.BindReg(r86, &d149)
					}
					ctx.FreeDesc(&d148)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d149)
					ctx.FreeDesc(&d146)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d149)
					var d150 JITValueDesc
					if d149.Loc == LocImm {
						d150 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d149.Imm.Int()) == uint64(0x2))}
					} else {
						r87 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d149.Reg, 2)
						ctx.EmitSetcc(r87, CondEqual)
						d150 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r87}
						ctx.BindReg(r87, &d150)
					}
					ctx.FreeDesc(&d149)
					ctx.ReclaimUntrackedRegs()
					r88 := ctx.AllocReg()
					ctx.EnsureDesc(&d150)
					ctx.EnsureDesc(&d150)
					if d150.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r88, d150)
					}
					ctx.EmitJmp(lbl52)
					bbpos_8_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl53)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d151 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d151)
					if d151.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r88, d151)
					}
					ctx.EmitJmp(lbl52)
					ctx.MarkLabel(lbl52)
					d152 = JITValueDesc{Loc: LocReg, Reg: r88}
					ctx.BindReg(r88, &d152)
					ctx.BindReg(r88, &d152)
					if r68 {
						ctx.UnprotectReg(r69)
					}
					if r70 {
						ctx.UnprotectReg(r71)
					}
					if r72 {
						ctx.UnprotectReg(r73)
					}
					ctx.ReclaimUntrackedRegs()
					d153 = d152
					ctx.EnsureDesc(&d153)
					if d153.Loc != LocImm && d153.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl60 := ctx.ReserveLabel()
					lbl61 := ctx.ReserveLabel()
					lbl62 := ctx.ReserveLabel()
					lbl63 := ctx.ReserveLabel()
					if d153.Loc == LocImm {
						if d153.Imm.Bool() {
							ctx.MarkLabel(lbl62)
							ctx.EmitJmp(lbl60)
						} else {
							ctx.MarkLabel(lbl63)
							ctx.EmitJmp(lbl61)
						}
					} else {
						ctx.EmitCmpRegImm32(d153.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl62)
						ctx.EmitJmp(lbl63)
						ctx.MarkLabel(lbl62)
						ctx.EmitJmp(lbl60)
						ctx.MarkLabel(lbl63)
						ctx.EmitJmp(lbl61)
					}
					ctx.FreeDesc(&d152)
					bbpos_2_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl61)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocRegTriple || d32.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d32)
					d154 = ctx.EmitGoCallScalar(GoFuncAddr(readFrom), []JITValueDesc{d32}, 2)
					ctx.BindReg(d154.Reg, &d154)
					ctx.BindReg(d154.Reg2, &d154)
					ctx.ReclaimUntrackedRegs()
					stackArray155 = ctx.AllocStack(int32(16))
					_ = stackArray155
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d154)
					ctx.EnsureDesc(&d154)
					ctx.EmitStoreScmerToStack(d154, int32(stackArray155)+int32(0))
					ctx.FreeDesc(&d154)
					ctx.ReclaimUntrackedRegs()
					d156 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d156
					ctx.ReclaimUntrackedRegs()
					r89 := ctx.AllocReg()
					r90 := ctx.AllocRegExcept(r89)
					r91 := ctx.AllocRegExcept(r89, r90)
					d157 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r89, Reg2: r90, Reg3: r91}
					ctx.BindReg(r89, &d157)
					ctx.BindReg(r90, &d157)
					ctx.BindReg(r91, &d157)
					ctx.BindReg(r89, &d157)
					ctx.BindReg(r90, &d157)
					ctx.BindReg(r91, &d157)
					ctx.EmitLeaRegMem(d157.Reg, ctx.StackReg, int32(stackArray155))
					ctx.EmitMovRegImm64(d157.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d157.Reg3, uint64(1))
					callResults158 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d35, d157}, []uint8{3}, []uint8{1})
					d159 = callResults158[0]
					d160 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase33) + int32(16)}
					ctx.EmitCopyDescWords(&d160, &d159, 3)
					ctx.FreeDesc(&d159)
					d159 = d160
					ctx.StabilizeDescForControlFlow(&d159)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_2_9)
					bbpos_2_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl44)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d161 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					var d162 JITValueDesc
					if d161.SliceSizeKnown {
						d162 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d161.KnownSliceLen))}
					} else if d161.Loc == LocImm {
						d162 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d161.StackOff))}
					} else if d161.Loc == LocStackTriple {
						d162 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d161.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d161)
						if d161.Loc == LocRegPair || d161.Loc == LocRegTriple {
							d162 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d161.Reg2, ID: 0}
						} else if d161.Loc == LocReg {
							d162 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d161.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d162)
					var d163 JITValueDesc
					if d162.Loc == LocImm {
						d163 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d162.Imm.Int() > 0)}
					} else {
						r92 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d162.Reg, 0)
						ctx.EmitSetcc(r92, CondSignedGreater)
						d163 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r92}
						ctx.BindReg(r92, &d163)
					}
					ctx.FreeDesc(&d162)
					ctx.ReclaimUntrackedRegs()
					d164 = d163
					ctx.EnsureDesc(&d164)
					if d164.Loc != LocImm && d164.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl64 := ctx.ReserveLabel()
					lbl65 := ctx.ReserveLabel()
					lbl66 := ctx.ReserveLabel()
					if d164.Loc == LocImm {
						if d164.Imm.Bool() {
							ctx.MarkLabel(lbl65)
							ctx.EmitJmp(lbl64)
						} else {
							ctx.MarkLabel(lbl66)
							ctx.EmitJmp(lbl45)
						}
					} else {
						ctx.EmitCmpRegImm32(d164.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl65)
						ctx.EmitJmp(lbl66)
						ctx.MarkLabel(lbl65)
						ctx.EmitJmp(lbl64)
						ctx.MarkLabel(lbl66)
						ctx.EmitJmp(lbl45)
					}
					ctx.FreeDesc(&d163)
					bbpos_2_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl48)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					bbpos_2_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl60)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d166 = d132
					ctx.EnsureDesc(&d166)
					if d166.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d166.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d166)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d166)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d166)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d166.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d166 = tmpPair
					} else if d166.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d166.Reg), Reg2: ctx.AllocRegExcept(d166.Reg)}
						switch d166.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d166)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d166)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d166)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d166)
						d166 = tmpPair
					} else if d166.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d166.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d166.MemPtr))
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
						d166 = tmpPair
					}
					if d166.Loc != LocRegPair && d166.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d165 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d166}, 2)
					ctx.FreeDesc(&d132)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d165)
					d167 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(")")}
					var d168 JITValueDesc
					if d167.Loc == LocImm {
						ctx.TrackImm(d167.Imm)
						ptrWord, _ := d167.Imm.RawWords()
						d168 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d168.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d168.Reg2, uint64(len(d167.Imm.String())))
						ctx.BindReg(d168.Reg, &d168)
						ctx.BindReg(d168.Reg2, &d168)
					} else {
						d168 = d167
					}
					d169 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d165, d168}, 1)
					ctx.EmitAndRegImm32(d169.Reg, 1)
					d169.Type = tagBool
					ctx.BindReg(d169.Reg, &d169)
					ctx.ReclaimUntrackedRegs()
					d170 = d169
					ctx.EnsureDesc(&d170)
					if d170.Loc != LocImm && d170.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl67 := ctx.ReserveLabel()
					lbl68 := ctx.ReserveLabel()
					lbl69 := ctx.ReserveLabel()
					if d170.Loc == LocImm {
						if d170.Imm.Bool() {
							ctx.MarkLabel(lbl68)
							ctx.EmitJmp(lbl67)
						} else {
							ctx.MarkLabel(lbl69)
							ctx.EmitJmp(lbl61)
						}
					} else {
						ctx.EmitCmpRegImm32(d170.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl68)
						ctx.EmitJmp(lbl69)
						ctx.MarkLabel(lbl68)
						ctx.EmitJmp(lbl67)
						ctx.MarkLabel(lbl69)
						ctx.EmitJmp(lbl61)
					}
					ctx.FreeDesc(&d169)
					bbpos_2_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl64)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d171 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d172 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d174 = ctx.EmitSliceElementAddress(&d171, &d172, 16)
					ctx.EmitLoadScmerToStack(&d174, int32(phiBase33)+int32(40))
					ctx.FreeDesc(&d174)
					d173 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					ctx.StabilizeDescForControlFlow(&d173)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d173)
					r93 := d173.Loc == LocReg || d173.Loc == LocRegPair || d173.Loc == LocRegTriple
					r94 := d173.Reg
					if r93 {
						ctx.ProtectReg(r94)
					}
					r95 := d173.Loc == LocRegPair || d173.Loc == LocRegTriple
					r96 := d173.Reg2
					if r95 {
						ctx.ProtectReg(r96)
					}
					r97 := d173.Loc == LocRegTriple
					r98 := d173.Reg3
					if r97 {
						ctx.ProtectReg(r98)
					}
					lbl70 := ctx.ReserveLabel()
					bbpos_10_0 := int32(-1)
					_ = bbpos_10_0
					bbpos_10_1 := int32(-1)
					_ = bbpos_10_1
					bbpos_10_2 := int32(-1)
					_ = bbpos_10_2
					bbpos_10_3 := int32(-1)
					_ = bbpos_10_3
					bbpos_10_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r99 := ctx.AllocReg()
					r100 := ctx.AllocRegExcept(r99)
					ctx.EmitMovRegImm64(r99, 0)
					ctx.EmitMovRegImm64(r100, 0)
					d175 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r99, Reg2: r100}
					ctx.BindReg(r99, &d175)
					ctx.BindReg(r100, &d175)
					ctx.StabilizeDescForControlFlow(&d175)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d173)
					ctx.ReclaimUntrackedRegs()
					d176 = args[0]
					d176.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d177 JITValueDesc
					ctx.EnsureDesc(&d176)
					if d176.Loc == LocImm {
						ptrWord, _ := d176.Imm.RawWords()
						d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d176.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r101 := ctx.AllocReg()
						ctx.EmitMovRegReg(r101, d176.Reg)
						d177 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r101}
						ctx.BindReg(r101, &d177)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d177)
					d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d177)
					ctx.EnsureDesc(&d179)
					ctx.EnsureDesc(&d177)
					ctx.EnsureDesc(&d179)
					var d178 JITValueDesc
					if d177.Loc == LocImm && d179.Loc == LocImm {
						d178 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d177.Imm.Int() == d179.Imm.Int())}
					} else if d179.Loc == LocImm {
						r102 := ctx.AllocReg()
						if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d177.Reg, int32(d179.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d179.Imm.Int()))
							ctx.EmitCmpInt64(d177.Reg, RegR11)
						}
						ctx.EmitSetcc(r102, CondEqual)
						d178 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r102}
						ctx.BindReg(r102, &d178)
					} else if d177.Loc == LocImm {
						r103 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d177.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d179.Reg)
						ctx.EmitSetcc(r103, CondEqual)
						d178 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r103}
						ctx.BindReg(r103, &d178)
					} else {
						r104 := ctx.AllocReg()
						ctx.EmitCmpInt64(d177.Reg, d179.Reg)
						ctx.EmitSetcc(r104, CondEqual)
						d178 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r104}
						ctx.BindReg(r104, &d178)
					}
					ctx.FreeDesc(&d177)
					ctx.ReclaimUntrackedRegs()
					d180 = d178
					ctx.EnsureDesc(&d180)
					if d180.Loc != LocImm && d180.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl71 := ctx.ReserveLabel()
					lbl72 := ctx.ReserveLabel()
					lbl73 := ctx.ReserveLabel()
					lbl74 := ctx.ReserveLabel()
					if d180.Loc == LocImm {
						if d180.Imm.Bool() {
							ctx.MarkLabel(lbl73)
							ctx.EmitJmp(lbl71)
						} else {
							ctx.MarkLabel(lbl74)
							ctx.EmitJmp(lbl72)
						}
					} else {
						ctx.EmitCmpRegImm32(d180.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl73)
						ctx.EmitJmp(lbl74)
						ctx.MarkLabel(lbl73)
						ctx.EmitJmp(lbl71)
						ctx.MarkLabel(lbl74)
						ctx.EmitJmp(lbl72)
					}
					ctx.FreeDesc(&d178)
					bbpos_10_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl72)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d181 = args[0]
					d181.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d182 JITValueDesc
					ctx.EnsureDesc(&d181)
					if d181.Loc == LocImm {
						ptrWord, _ := d181.Imm.RawWords()
						d182 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d181.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r105 := ctx.AllocReg()
						ctx.EmitMovRegReg(r105, d181.Reg)
						d182 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r105}
						ctx.BindReg(r105, &d182)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d182)
					d184 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d182)
					ctx.EnsureDesc(&d184)
					ctx.EnsureDesc(&d182)
					ctx.EnsureDesc(&d184)
					var d183 JITValueDesc
					if d182.Loc == LocImm && d184.Loc == LocImm {
						d183 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d182.Imm.Int() == d184.Imm.Int())}
					} else if d184.Loc == LocImm {
						r106 := ctx.AllocReg()
						if d184.Imm.Int() >= -2147483648 && d184.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d182.Reg, int32(d184.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d184.Imm.Int()))
							ctx.EmitCmpInt64(d182.Reg, RegR11)
						}
						ctx.EmitSetcc(r106, CondEqual)
						d183 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r106}
						ctx.BindReg(r106, &d183)
					} else if d182.Loc == LocImm {
						r107 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d182.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d184.Reg)
						ctx.EmitSetcc(r107, CondEqual)
						d183 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r107}
						ctx.BindReg(r107, &d183)
					} else {
						r108 := ctx.AllocReg()
						ctx.EmitCmpInt64(d182.Reg, d184.Reg)
						ctx.EmitSetcc(r108, CondEqual)
						d183 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r108}
						ctx.BindReg(r108, &d183)
					}
					ctx.FreeDesc(&d182)
					ctx.ReclaimUntrackedRegs()
					d185 = d183
					ctx.EnsureDesc(&d185)
					if d185.Loc != LocImm && d185.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl75 := ctx.ReserveLabel()
					lbl76 := ctx.ReserveLabel()
					lbl77 := ctx.ReserveLabel()
					if d185.Loc == LocImm {
						if d185.Imm.Bool() {
							ctx.MarkLabel(lbl76)
							ctx.EmitJmp(lbl71)
						} else {
							ctx.MarkLabel(lbl77)
							ctx.EmitJmp(lbl75)
						}
					} else {
						ctx.EmitCmpRegImm32(d185.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl76)
						ctx.EmitJmp(lbl77)
						ctx.MarkLabel(lbl76)
						ctx.EmitJmp(lbl71)
						ctx.MarkLabel(lbl77)
						ctx.EmitJmp(lbl75)
					}
					ctx.FreeDesc(&d183)
					bbpos_10_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl75)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d186 = args[0]
					d186.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d187 JITValueDesc
					ctx.EnsureDesc(&d186)
					if d186.Loc == LocImm {
						_, auxWord := d186.Imm.RawWords()
						d187 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d186.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r109 := ctx.AllocReg()
						ctx.EmitMovRegReg(r109, d186.Reg2)
						d187 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r109}
						ctx.BindReg(r109, &d187)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d187)
					d188 = d187
					_ = d188
					ctx.StabilizeDescForControlFlow(&d188)
					bbpos_11_0 := int32(-1)
					_ = bbpos_11_0
					bbpos_11_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d188)
					var d189 JITValueDesc
					if d188.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d188.Imm.Int() & 255)}
					} else {
						r110 := ctx.AllocRegExcept(d188.Reg)
						ctx.EmitMovRegReg(r110, d188.Reg)
						ctx.EmitAndRegImm32(r110, int32(255))
						d189 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r110}
						ctx.BindReg(r110, &d189)
					}
					if d189.Loc == LocReg && d188.Loc == LocReg && d189.Reg == d188.Reg {
						ctx.TransferReg(d188.Reg)
						d188.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d189)
					ctx.EnsureDesc(&d189)
					var d190 JITValueDesc
					if d189.Loc == LocImm {
						d190 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d189.Imm.Int()))))}
					} else {
						r111 := ctx.AllocReg()
						ctx.EmitMovRegReg(r111, d189.Reg)
						ctx.EmitShlRegImm8(r111, 56)
						ctx.EmitShrRegImm8(r111, 56)
						d190 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r111}
						ctx.BindReg(r111, &d190)
					}
					ctx.FreeDesc(&d189)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d190)
					ctx.FreeDesc(&d187)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d190)
					var d191 JITValueDesc
					if d190.Loc == LocImm {
						d191 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d190.Imm.Int()) == uint64(0xe))}
					} else {
						r112 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d190.Reg, 14)
						ctx.EmitSetcc(r112, CondEqual)
						d191 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r112}
						ctx.BindReg(r112, &d191)
					}
					ctx.FreeDesc(&d190)
					ctx.ReclaimUntrackedRegs()
					r113 := ctx.AllocReg()
					ctx.EnsureDesc(&d191)
					ctx.EnsureDesc(&d191)
					if d191.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r113, d191)
					}
					ctx.EmitJmp(lbl70)
					bbpos_10_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl71)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d192 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d192)
					if d192.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r113, d192)
					}
					ctx.EmitJmp(lbl70)
					ctx.MarkLabel(lbl70)
					d193 = JITValueDesc{Loc: LocReg, Reg: r113}
					ctx.BindReg(r113, &d193)
					ctx.BindReg(r113, &d193)
					if r93 {
						ctx.UnprotectReg(r94)
					}
					if r95 {
						ctx.UnprotectReg(r96)
					}
					if r97 {
						ctx.UnprotectReg(r98)
					}
					ctx.ReclaimUntrackedRegs()
					d194 = d193
					ctx.EnsureDesc(&d194)
					if d194.Loc != LocImm && d194.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl78 := ctx.ReserveLabel()
					lbl79 := ctx.ReserveLabel()
					lbl80 := ctx.ReserveLabel()
					lbl81 := ctx.ReserveLabel()
					if d194.Loc == LocImm {
						if d194.Imm.Bool() {
							ctx.MarkLabel(lbl80)
							ctx.EmitJmp(lbl78)
						} else {
							ctx.MarkLabel(lbl81)
							ctx.SyncDesc(&d173)
							if d173.Loc == LocReg {
								ctx.ProtectReg(d173.Reg)
							} else if d173.Loc == LocRegPair {
								ctx.ProtectReg(d173.Reg)
								ctx.ProtectReg(d173.Reg2)
							}
							d195 = d173
							if d195.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d195)
							if d195.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d195, int32(phiBase33)+int32(40), 2)
							} else if d195.Loc == LocInputPair {
								ctx.EnsureDesc(&d195)
								ctx.EmitStoreScmerToStack(d195, int32(phiBase33)+int32(40))
							} else if d195.Loc == LocRegPair || d195.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d195, int32(phiBase33)+int32(40))
							} else {
								ctx.EnsureDesc(&d195)
								ctx.EmitStoreToStack(d195, int32(phiBase33)+int32(40))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(40))+8)
							}
							if d173.Loc == LocReg {
								ctx.UnprotectReg(d173.Reg)
							} else if d173.Loc == LocRegPair {
								ctx.UnprotectReg(d173.Reg)
								ctx.UnprotectReg(d173.Reg2)
							}
							ctx.EmitJmp(lbl79)
						}
					} else {
						ctx.EmitCmpRegImm32(d194.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl80)
						ctx.EmitJmp(lbl81)
						ctx.MarkLabel(lbl80)
						ctx.EmitJmp(lbl78)
						ctx.MarkLabel(lbl81)
						ctx.SyncDesc(&d173)
						if d173.Loc == LocReg {
							ctx.ProtectReg(d173.Reg)
						} else if d173.Loc == LocRegPair {
							ctx.ProtectReg(d173.Reg)
							ctx.ProtectReg(d173.Reg2)
						}
						d196 = d173
						if d196.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d196)
						if d196.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d196, int32(phiBase33)+int32(40), 2)
						} else if d196.Loc == LocInputPair {
							ctx.EnsureDesc(&d196)
							ctx.EmitStoreScmerToStack(d196, int32(phiBase33)+int32(40))
						} else if d196.Loc == LocRegPair || d196.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d196, int32(phiBase33)+int32(40))
						} else {
							ctx.EnsureDesc(&d196)
							ctx.EmitStoreToStack(d196, int32(phiBase33)+int32(40))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(40))+8)
						}
						if d173.Loc == LocReg {
							ctx.UnprotectReg(d173.Reg)
						} else if d173.Loc == LocRegPair {
							ctx.UnprotectReg(d173.Reg)
							ctx.UnprotectReg(d173.Reg2)
						}
						ctx.EmitJmp(lbl79)
					}
					ctx.FreeDesc(&d193)
					bbpos_2_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl79)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					r114 := d36.Loc == LocReg || d36.Loc == LocRegPair || d36.Loc == LocRegTriple
					r115 := d36.Reg
					if r114 {
						ctx.ProtectReg(r115)
					}
					r116 := d36.Loc == LocRegPair || d36.Loc == LocRegTriple
					r117 := d36.Reg2
					if r116 {
						ctx.ProtectReg(r117)
					}
					r118 := d36.Loc == LocRegTriple
					r119 := d36.Reg3
					if r118 {
						ctx.ProtectReg(r119)
					}
					lbl82 := ctx.ReserveLabel()
					bbpos_12_0 := int32(-1)
					_ = bbpos_12_0
					bbpos_12_1 := int32(-1)
					_ = bbpos_12_1
					bbpos_12_2 := int32(-1)
					_ = bbpos_12_2
					bbpos_12_3 := int32(-1)
					_ = bbpos_12_3
					bbpos_12_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r120 := ctx.AllocReg()
					r121 := ctx.AllocRegExcept(r120)
					ctx.EmitMovRegImm64(r120, 0)
					ctx.EmitMovRegImm64(r121, 0)
					d197 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r120, Reg2: r121}
					ctx.BindReg(r120, &d197)
					ctx.BindReg(r121, &d197)
					ctx.StabilizeDescForControlFlow(&d197)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					d198 = args[0]
					d198.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d199 JITValueDesc
					ctx.EnsureDesc(&d198)
					if d198.Loc == LocImm {
						ptrWord, _ := d198.Imm.RawWords()
						d199 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d198.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r122 := ctx.AllocReg()
						ctx.EmitMovRegReg(r122, d198.Reg)
						d199 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r122}
						ctx.BindReg(r122, &d199)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d199)
					d201 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d199)
					ctx.EnsureDesc(&d201)
					ctx.EnsureDesc(&d199)
					ctx.EnsureDesc(&d201)
					var d200 JITValueDesc
					if d199.Loc == LocImm && d201.Loc == LocImm {
						d200 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d199.Imm.Int() == d201.Imm.Int())}
					} else if d201.Loc == LocImm {
						r123 := ctx.AllocReg()
						if d201.Imm.Int() >= -2147483648 && d201.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d199.Reg, int32(d201.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d201.Imm.Int()))
							ctx.EmitCmpInt64(d199.Reg, RegR11)
						}
						ctx.EmitSetcc(r123, CondEqual)
						d200 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r123}
						ctx.BindReg(r123, &d200)
					} else if d199.Loc == LocImm {
						r124 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d199.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d201.Reg)
						ctx.EmitSetcc(r124, CondEqual)
						d200 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r124}
						ctx.BindReg(r124, &d200)
					} else {
						r125 := ctx.AllocReg()
						ctx.EmitCmpInt64(d199.Reg, d201.Reg)
						ctx.EmitSetcc(r125, CondEqual)
						d200 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r125}
						ctx.BindReg(r125, &d200)
					}
					ctx.FreeDesc(&d199)
					ctx.ReclaimUntrackedRegs()
					d202 = d200
					ctx.EnsureDesc(&d202)
					if d202.Loc != LocImm && d202.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl83 := ctx.ReserveLabel()
					lbl84 := ctx.ReserveLabel()
					lbl85 := ctx.ReserveLabel()
					lbl86 := ctx.ReserveLabel()
					if d202.Loc == LocImm {
						if d202.Imm.Bool() {
							ctx.MarkLabel(lbl85)
							ctx.EmitJmp(lbl83)
						} else {
							ctx.MarkLabel(lbl86)
							ctx.EmitJmp(lbl84)
						}
					} else {
						ctx.EmitCmpRegImm32(d202.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl85)
						ctx.EmitJmp(lbl86)
						ctx.MarkLabel(lbl85)
						ctx.EmitJmp(lbl83)
						ctx.MarkLabel(lbl86)
						ctx.EmitJmp(lbl84)
					}
					ctx.FreeDesc(&d200)
					bbpos_12_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl84)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d203 = args[0]
					d203.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d204 JITValueDesc
					ctx.EnsureDesc(&d203)
					if d203.Loc == LocImm {
						ptrWord, _ := d203.Imm.RawWords()
						d204 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d203.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r126 := ctx.AllocReg()
						ctx.EmitMovRegReg(r126, d203.Reg)
						d204 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r126}
						ctx.BindReg(r126, &d204)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d204)
					d206 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d204)
					ctx.EnsureDesc(&d206)
					ctx.EnsureDesc(&d204)
					ctx.EnsureDesc(&d206)
					var d205 JITValueDesc
					if d204.Loc == LocImm && d206.Loc == LocImm {
						d205 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d204.Imm.Int() == d206.Imm.Int())}
					} else if d206.Loc == LocImm {
						r127 := ctx.AllocReg()
						if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d204.Reg, int32(d206.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d206.Imm.Int()))
							ctx.EmitCmpInt64(d204.Reg, RegR11)
						}
						ctx.EmitSetcc(r127, CondEqual)
						d205 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r127}
						ctx.BindReg(r127, &d205)
					} else if d204.Loc == LocImm {
						r128 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d204.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d206.Reg)
						ctx.EmitSetcc(r128, CondEqual)
						d205 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r128}
						ctx.BindReg(r128, &d205)
					} else {
						r129 := ctx.AllocReg()
						ctx.EmitCmpInt64(d204.Reg, d206.Reg)
						ctx.EmitSetcc(r129, CondEqual)
						d205 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r129}
						ctx.BindReg(r129, &d205)
					}
					ctx.FreeDesc(&d204)
					ctx.ReclaimUntrackedRegs()
					d207 = d205
					ctx.EnsureDesc(&d207)
					if d207.Loc != LocImm && d207.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl87 := ctx.ReserveLabel()
					lbl88 := ctx.ReserveLabel()
					lbl89 := ctx.ReserveLabel()
					if d207.Loc == LocImm {
						if d207.Imm.Bool() {
							ctx.MarkLabel(lbl88)
							ctx.EmitJmp(lbl83)
						} else {
							ctx.MarkLabel(lbl89)
							ctx.EmitJmp(lbl87)
						}
					} else {
						ctx.EmitCmpRegImm32(d207.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl88)
						ctx.EmitJmp(lbl89)
						ctx.MarkLabel(lbl88)
						ctx.EmitJmp(lbl83)
						ctx.MarkLabel(lbl89)
						ctx.EmitJmp(lbl87)
					}
					ctx.FreeDesc(&d205)
					bbpos_12_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl87)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d208 = args[0]
					d208.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d209 JITValueDesc
					ctx.EnsureDesc(&d208)
					if d208.Loc == LocImm {
						_, auxWord := d208.Imm.RawWords()
						d209 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d208.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r130 := ctx.AllocReg()
						ctx.EmitMovRegReg(r130, d208.Reg2)
						d209 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r130}
						ctx.BindReg(r130, &d209)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d209)
					d210 = d209
					_ = d210
					ctx.StabilizeDescForControlFlow(&d210)
					bbpos_13_0 := int32(-1)
					_ = bbpos_13_0
					bbpos_13_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d210)
					var d211 JITValueDesc
					if d210.Loc == LocImm {
						d211 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d210.Imm.Int() & 255)}
					} else {
						r131 := ctx.AllocRegExcept(d210.Reg)
						ctx.EmitMovRegReg(r131, d210.Reg)
						ctx.EmitAndRegImm32(r131, int32(255))
						d211 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r131}
						ctx.BindReg(r131, &d211)
					}
					if d211.Loc == LocReg && d210.Loc == LocReg && d211.Reg == d210.Reg {
						ctx.TransferReg(d210.Reg)
						d210.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d211)
					ctx.EnsureDesc(&d211)
					var d212 JITValueDesc
					if d211.Loc == LocImm {
						d212 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d211.Imm.Int()))))}
					} else {
						r132 := ctx.AllocReg()
						ctx.EmitMovRegReg(r132, d211.Reg)
						ctx.EmitShlRegImm8(r132, 56)
						ctx.EmitShrRegImm8(r132, 56)
						d212 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r132}
						ctx.BindReg(r132, &d212)
					}
					ctx.FreeDesc(&d211)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d212)
					ctx.FreeDesc(&d209)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d212)
					var d213 JITValueDesc
					if d212.Loc == LocImm {
						d213 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d212.Imm.Int()) == uint64(0x2))}
					} else {
						r133 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d212.Reg, 2)
						ctx.EmitSetcc(r133, CondEqual)
						d213 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r133}
						ctx.BindReg(r133, &d213)
					}
					ctx.FreeDesc(&d212)
					ctx.ReclaimUntrackedRegs()
					r134 := ctx.AllocReg()
					ctx.EnsureDesc(&d213)
					ctx.EnsureDesc(&d213)
					if d213.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r134, d213)
					}
					ctx.EmitJmp(lbl82)
					bbpos_12_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl83)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d214 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d214)
					if d214.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r134, d214)
					}
					ctx.EmitJmp(lbl82)
					ctx.MarkLabel(lbl82)
					d215 = JITValueDesc{Loc: LocReg, Reg: r134}
					ctx.BindReg(r134, &d215)
					ctx.BindReg(r134, &d215)
					if r114 {
						ctx.UnprotectReg(r115)
					}
					if r116 {
						ctx.UnprotectReg(r117)
					}
					if r118 {
						ctx.UnprotectReg(r119)
					}
					ctx.ReclaimUntrackedRegs()
					d216 = d215
					ctx.EnsureDesc(&d216)
					if d216.Loc != LocImm && d216.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl90 := ctx.ReserveLabel()
					lbl91 := ctx.ReserveLabel()
					lbl92 := ctx.ReserveLabel()
					lbl93 := ctx.ReserveLabel()
					if d216.Loc == LocImm {
						if d216.Imm.Bool() {
							ctx.MarkLabel(lbl92)
							ctx.EmitJmp(lbl90)
						} else {
							ctx.MarkLabel(lbl93)
							ctx.EmitJmp(lbl91)
						}
					} else {
						ctx.EmitCmpRegImm32(d216.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl92)
						ctx.EmitJmp(lbl93)
						ctx.MarkLabel(lbl92)
						ctx.EmitJmp(lbl90)
						ctx.MarkLabel(lbl93)
						ctx.EmitJmp(lbl91)
					}
					ctx.FreeDesc(&d215)
					bbpos_2_21 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl91)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocRegTriple || d32.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d32)
					d217 = ctx.EmitGoCallScalar(GoFuncAddr(readFrom), []JITValueDesc{d32}, 2)
					ctx.BindReg(d217.Reg, &d217)
					ctx.BindReg(d217.Reg2, &d217)
					ctx.ReclaimUntrackedRegs()
					stackArray218 = ctx.AllocStack(int32(32))
					_ = stackArray218
					ctx.ReclaimUntrackedRegs()
					d219 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d219
					ctx.ReclaimUntrackedRegs()
					d220 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("quote")}
					ctx.EnsureDesc(&d220)
					if d220.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d220.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d220.Imm)
						ptrWord, _ := d220.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d220.Imm.String())))
						d220 = tmpPair
					} else if d220.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d220.Type, Reg: ctx.AllocRegExcept(d220.Reg), Reg2: ctx.AllocRegExcept(d220.Reg)}
						switch d220.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d220)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d220)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d220)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d220)
						d220 = tmpPair
					}
					if d220.Loc != LocRegPair && d220.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
					}
					ctx.SyncDesc(&d220)
					d221 = ctx.EmitGoCallScalar(GoFuncAddr(NewSymbol), []JITValueDesc{d220}, 2)
					ctx.BindReg(d221.Reg, &d221)
					ctx.BindReg(d221.Reg2, &d221)
					ctx.FreeDesc(&d220)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d221)
					ctx.EnsureDesc(&d221)
					ctx.EmitStoreScmerToStack(d221, int32(stackArray218)+int32(0))
					ctx.FreeDesc(&d221)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d217)
					ctx.EnsureDesc(&d217)
					ctx.EmitStoreScmerToStack(d217, int32(stackArray218)+int32(16))
					ctx.FreeDesc(&d217)
					ctx.ReclaimUntrackedRegs()
					r135 := ctx.AllocReg()
					r136 := ctx.AllocRegExcept(r135)
					r137 := ctx.AllocRegExcept(r135, r136)
					d222 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r135, Reg2: r136, Reg3: r137}
					ctx.BindReg(r135, &d222)
					ctx.BindReg(r136, &d222)
					ctx.BindReg(r137, &d222)
					ctx.BindReg(r135, &d222)
					ctx.BindReg(r136, &d222)
					ctx.BindReg(r137, &d222)
					ctx.EmitLeaRegMem(d222.Reg, ctx.StackReg, int32(stackArray218))
					ctx.EmitMovRegImm64(d222.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d222.Reg3, uint64(2))
					callResults223 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d222}, []uint8{2}, []uint8{1})
					d224 = callResults223[0]
					ctx.StabilizeDescForControlFlow(&d224)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d225 JITValueDesc
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						fieldAddr := uintptr(d42.Imm.Int()) + 0
						r138 := ctx.AllocReg()
						r139 := ctx.AllocRegExcept(r138)
						r140 := ctx.AllocRegExcept(r138, r139)
						ctx.EmitMovRegMem64(r138, fieldAddr)
						ctx.EmitMovRegMem64(r139, fieldAddr+8)
						ctx.EmitMovRegMem64(r140, fieldAddr+16)
						d225 = JITValueDesc{Loc: LocRegTriple, Reg: r138, Reg2: r139, Reg3: r140}
						ctx.BindReg(r138, &d225)
						ctx.BindReg(r139, &d225)
						ctx.BindReg(r140, &d225)
					} else {
						off := int32(0)
						baseReg := d42.Reg
						r141 := ctx.AllocRegExcept(baseReg)
						r142 := ctx.AllocRegExcept(baseReg, r141)
						r143 := ctx.AllocRegExcept(baseReg, r141, r142)
						ctx.EmitMovRegMem(r141, baseReg, off)
						ctx.EmitMovRegMem(r142, baseReg, off+8)
						ctx.EmitMovRegMem(r143, baseReg, off+16)
						d225 = JITValueDesc{Loc: LocRegTriple, Reg: r141, Reg2: r142, Reg3: r143}
						ctx.BindReg(r141, &d225)
						ctx.BindReg(r142, &d225)
						ctx.BindReg(r143, &d225)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d225)
					var d226 JITValueDesc
					if d225.Loc == LocImm {
						ctx.TrackImm(d225.Imm)
						ptrWord, _ := d225.Imm.RawWords()
						d226 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d226.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d226.Reg2, uint64(len(d225.Imm.String())))
						ctx.BindReg(d226.Reg, &d226)
						ctx.BindReg(d226.Reg2, &d226)
					} else {
						d226 = d225
					}
					d227 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					var d228 JITValueDesc
					if d227.Loc == LocImm {
						ctx.TrackImm(d227.Imm)
						ptrWord, _ := d227.Imm.RawWords()
						d228 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d228.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d228.Reg2, uint64(len(d227.Imm.String())))
						ctx.BindReg(d228.Reg, &d228)
						ctx.BindReg(d228.Reg2, &d228)
					} else {
						d228 = d227
					}
					d229 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d226, d228}, 1)
					ctx.EmitAndRegImm32(d229.Reg, 1)
					ctx.EmitCmpRegImm32(d229.Reg, 0)
					ctx.EmitSetcc(d229.Reg, CondEqual)
					d229.Type = tagBool
					ctx.BindReg(d229.Reg, &d229)
					ctx.ReclaimUntrackedRegs()
					d230 = d229
					ctx.EnsureDesc(&d230)
					if d230.Loc != LocImm && d230.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl94 := ctx.ReserveLabel()
					lbl95 := ctx.ReserveLabel()
					lbl96 := ctx.ReserveLabel()
					lbl97 := ctx.ReserveLabel()
					if d230.Loc == LocImm {
						if d230.Imm.Bool() {
							ctx.MarkLabel(lbl96)
							ctx.EmitJmp(lbl94)
						} else {
							ctx.MarkLabel(lbl97)
							ctx.EmitJmp(lbl95)
						}
					} else {
						ctx.EmitCmpRegImm32(d230.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl96)
						ctx.EmitJmp(lbl97)
						ctx.MarkLabel(lbl96)
						ctx.EmitJmp(lbl94)
						ctx.MarkLabel(lbl97)
						ctx.EmitJmp(lbl95)
					}
					ctx.FreeDesc(&d229)
					bbpos_2_32 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl95)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d231 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d231)
					ctx.BindReg(r50, &d231)
					ctx.EmitMovPairToResult(&d224, &d231)
					ctx.EmitJmp(lbl6)
					bbpos_2_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl67)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d232 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d233 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					var d234 JITValueDesc
					ctx.EnsureDesc(&d232)
					if d232.Loc == LocRegPair || d232.Loc == LocRegTriple {
						d234 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d232.Reg2}
						ctx.BindReg(d232.Reg2, &d234)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d232)
					ctx.EnsureDesc(&d233)
					ctx.EnsureDesc(&d234)
					var d236 JITValueDesc
					if d234.Loc == LocImm && d233.Loc == LocImm {
						d236 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d234.Imm.Int() - d233.Imm.Int())}
					} else {
						r144 := ctx.AllocReg()
						if d234.Loc == LocImm {
							ctx.EmitMovRegImm64(r144, uint64(d234.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r144, d234.Reg)
						}
						if d233.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d233.Imm.Int()))
							ctx.EmitSubInt64(r144, RegR11)
						} else {
							ctx.EmitSubInt64(r144, d233.Reg)
						}
						d236 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r144}
						ctx.BindReg(r144, &d236)
					}
					var d237 JITValueDesc
					if d232.Loc == LocImm && d233.Loc == LocImm {
						d237 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d232.Imm.Int() + d233.Imm.Int()*16)}
					} else {
						r145 := ctx.AllocReg()
						if d232.Loc == LocImm {
							ctx.EmitMovRegImm64(r145, uint64(d232.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r145, d232.Reg)
						}
						if d233.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d233.Imm.Int()*16))
							ctx.EmitAddInt64(r145, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r145, d233.Reg)
							ctx.EmitMovRegReg(offsetReg, d233.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r145, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d237 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r145}
						ctx.BindReg(r145, &d237)
					}
					var d238 JITValueDesc
					var r146 Reg
					var r147 Reg
					ctx.SyncDesc(&d237)
					ctx.EnsureDesc(&d237)
					if d237.Loc == LocImm {
						r146 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r146, uint64(d237.Imm.Int()))
					} else {
						r146 = d237.Reg
					}
					ctx.ProtectReg(r146)
					ctx.SyncDesc(&d236)
					ctx.EnsureDesc(&d236)
					if d236.Loc == LocImm {
						r147 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r147, uint64(d236.Imm.Int()))
					} else {
						r147 = d236.Reg
					}
					ctx.ProtectReg(r147)
					r148 := ctx.EmitSliceCapAfterLow(&d232, &d233, r146, r147)
					ctx.UnprotectReg(r147)
					ctx.UnprotectReg(r146)
					d238 = JITValueDesc{Loc: LocRegTriple, Reg: r146, Reg2: r147, Reg3: r148}
					ctx.BindReg(r146, &d238)
					ctx.BindReg(r147, &d238)
					ctx.BindReg(r148, &d238)
					ctx.BindReg(r146, &d238)
					ctx.BindReg(r147, &d238)
					ctx.BindReg(r148, &d238)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d238)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[]Scmer, value []Scmer) { *dst = value }), []JITValueDesc{d32, d238})
					ctx.ReclaimUntrackedRegs()
					d239 = ctx.EmitNewSliceFromGoSlice(&d35)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d239)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d239)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value Scmer) { base.value = value }), []JITValueDesc{d42, d239})
					ctx.FreeDesc(&d239)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					d240 = d42
					_ = d240
					ctx.StabilizeDescForControlFlow(&d240)
					bbpos_14_0 := int32(-1)
					_ = bbpos_14_0
					bbpos_14_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d241 = ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
					ctx.BindReg(d241.Reg, &d241)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d240)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *SourceInfo) { *dst = *src }), []JITValueDesc{d241, d240})
					ctx.ReclaimUntrackedRegs()
					d242 = ctx.EmitGoCallScalar(GoFuncAddr(func() []*SourceInfo { return sourceCoverageInfos }), nil, 3)
					ctx.ReclaimUntrackedRegs()
					d243 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]*SourceInfo { return new([1]*SourceInfo) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d244 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d241)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]*SourceInfo, index int, value *SourceInfo) { dst[index] = value }), []JITValueDesc{d243, d244, d241})
					ctx.ReclaimUntrackedRegs()
					sliceResults245 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]*SourceInfo) []*SourceInfo { return value[0:1:1] }), []JITValueDesc{d243}, []uint8{3}, []uint8{1})
					d246 = sliceResults245[0]
					ctx.ReclaimUntrackedRegs()
					callResults247 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []*SourceInfo) []*SourceInfo { return append(dst, src...) }), []JITValueDesc{d242, d246}, []uint8{3}, []uint8{1})
					d248 = callResults247[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d248)
					ctx.EmitGoCallVoid(GoFuncAddr(func(value []*SourceInfo) { sourceCoverageInfos = value }), []JITValueDesc{d248})
					ctx.ReclaimUntrackedRegs()
					r149 := ctx.AllocReg()
					r150 := ctx.AllocRegExcept(r149)
					ctx.EmitMovRegImm64(r149, 0)
					ctx.EmitMovRegImm64(r150, 0)
					d249 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r149, Reg2: r150}
					ctx.BindReg(r149, &d249)
					ctx.BindReg(r150, &d249)
					ctx.ReclaimUntrackedRegs()
					d250 = args[0]
					d250.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d241)
					ctx.EnsureDesc(&d241)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d241)
					ctx.EnsureDesc(&d241)
					ctx.ReclaimUntrackedRegs()
					d253 = args[0]
					d253.ID = 0
					ctx.ReclaimUntrackedRegs()
					d254 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(14)}
					d255 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d256 = d254
					_ = d256
					ctx.StabilizeDescForControlFlow(&d256)
					d257 = d255
					_ = d257
					ctx.StabilizeDescForControlFlow(&d257)
					bbpos_15_0 := int32(-1)
					_ = bbpos_15_0
					bbpos_15_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d257)
					var d258 JITValueDesc
					if d257.Loc == LocImm {
						d258 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d257.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d257.Reg, 8)
						d258 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d257.Reg}
						ctx.BindReg(d257.Reg, &d258)
					}
					if d258.Loc == LocReg && d257.Loc == LocReg && d258.Reg == d257.Reg {
						ctx.TransferReg(d257.Reg)
						d257.Loc = LocNone
					}
					ctx.FreeDesc(&d257)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d256)
					var d259 JITValueDesc
					if d256.Loc == LocImm {
						d259 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d256.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d256.Reg, int32(255))
						d259 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d256.Reg}
						ctx.BindReg(d256.Reg, &d259)
					}
					if d259.Loc == LocImm {
						d259 = JITValueDesc{Loc: LocImm, Type: d259.Type, Imm: NewInt(int64(uint64(d259.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d259.Reg, 56)
						ctx.EmitShrRegImm8(d259.Reg, 56)
					}
					if d259.Loc == LocReg && d256.Loc == LocReg && d259.Reg == d256.Reg {
						ctx.TransferReg(d256.Reg)
						d256.Loc = LocNone
					}
					ctx.FreeDesc(&d256)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d259)
					ctx.EnsureDesc(&d259)
					var d260 JITValueDesc
					if d259.Loc == LocImm {
						d260 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d259.Imm.Int()))))}
					} else {
						r151 := ctx.AllocReg()
						ctx.EmitMovRegReg(r151, d259.Reg)
						ctx.EmitShlRegImm8(r151, 56)
						ctx.EmitShrRegImm8(r151, 56)
						d260 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r151}
						ctx.BindReg(r151, &d260)
					}
					ctx.FreeDesc(&d259)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d258)
					ctx.EnsureDesc(&d260)
					var d261 JITValueDesc
					if d258.Loc == LocImm && d260.Loc == LocImm {
						d261 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d258.Imm.Int() | d260.Imm.Int())}
					} else if d258.Loc == LocImm && d258.Imm.Int() == 0 {
						d261 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d260.Reg}
						ctx.BindReg(d260.Reg, &d261)
					} else if d260.Loc == LocImm && d260.Imm.Int() == 0 {
						d261 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d258.Reg}
						ctx.BindReg(d258.Reg, &d261)
					} else if d258.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d260.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d258.Imm.Int()))
						ctx.EmitOrInt64(scratch, d260.Reg)
						d261 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d261)
					} else if d260.Loc == LocImm {
						if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d258.Reg, int32(d260.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d260.Imm.Int()))
							ctx.EmitOrInt64(d258.Reg, RegR11)
						}
						d261 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d258.Reg}
						ctx.BindReg(d258.Reg, &d261)
					} else {
						ctx.EmitOrInt64(d258.Reg, d260.Reg)
						d261 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d258.Reg}
						ctx.BindReg(d258.Reg, &d261)
					}
					if d261.Loc == LocReg && d258.Loc == LocReg && d261.Reg == d258.Reg {
						ctx.TransferReg(d258.Reg)
						d258.Loc = LocNone
					}
					ctx.FreeDesc(&d258)
					ctx.FreeDesc(&d260)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d261)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d241)
					ctx.EnsureDesc(&d241)
					ctx.EmitMovToReg(d250.Reg, d241)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d261)
					ctx.EnsureDesc(&d261)
					ctx.EmitMovToReg(d253.Reg2, d261)
					ctx.FreeDesc(&d261)
					ctx.ReclaimUntrackedRegs()
					d262 = d249
					_ = d262
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d262)
					ctx.ReclaimUntrackedRegs()
					d263 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d263)
					ctx.BindReg(r50, &d263)
					ctx.EmitMovPairToResult(&d262, &d263)
					ctx.EmitJmp(lbl6)
					bbpos_2_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl78)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d173)
					r152 := d173.Loc == LocReg || d173.Loc == LocRegPair || d173.Loc == LocRegTriple
					r153 := d173.Reg
					if r152 {
						ctx.ProtectReg(r153)
					}
					r154 := d173.Loc == LocRegPair || d173.Loc == LocRegTriple
					r155 := d173.Reg2
					if r154 {
						ctx.ProtectReg(r155)
					}
					r156 := d173.Loc == LocRegTriple
					r157 := d173.Reg3
					if r156 {
						ctx.ProtectReg(r157)
					}
					lbl98 := ctx.ReserveLabel()
					bbpos_16_0 := int32(-1)
					_ = bbpos_16_0
					bbpos_16_1 := int32(-1)
					_ = bbpos_16_1
					bbpos_16_2 := int32(-1)
					_ = bbpos_16_2
					bbpos_16_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r158 := ctx.AllocReg()
					r159 := ctx.AllocRegExcept(r158)
					ctx.EmitMovRegImm64(r158, 0)
					ctx.EmitMovRegImm64(r159, 0)
					d264 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r158, Reg2: r159}
					ctx.BindReg(r158, &d264)
					ctx.BindReg(r159, &d264)
					ctx.StabilizeDescForControlFlow(&d264)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d173)
					ctx.ReclaimUntrackedRegs()
					d265 = d173
					_ = d265
					ctx.ReclaimUntrackedRegs()
					d266 = ctx.EmitGetTagDesc(&d265, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d265)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d266)
					var d267 JITValueDesc
					if d266.Loc == LocImm {
						d267 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d266.Imm.Int()) != uint64(0xe))}
					} else {
						r160 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d266.Reg, 14)
						ctx.EmitSetcc(r160, CondNotEqual)
						d267 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r160}
						ctx.BindReg(r160, &d267)
					}
					ctx.FreeDesc(&d266)
					ctx.ReclaimUntrackedRegs()
					d268 = d267
					ctx.EnsureDesc(&d268)
					if d268.Loc != LocImm && d268.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl99 := ctx.ReserveLabel()
					lbl100 := ctx.ReserveLabel()
					lbl101 := ctx.ReserveLabel()
					lbl102 := ctx.ReserveLabel()
					if d268.Loc == LocImm {
						if d268.Imm.Bool() {
							ctx.MarkLabel(lbl101)
							ctx.EmitJmp(lbl99)
						} else {
							ctx.MarkLabel(lbl102)
							ctx.EmitJmp(lbl100)
						}
					} else {
						ctx.EmitCmpRegImm32(d268.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl101)
						ctx.EmitJmp(lbl102)
						ctx.MarkLabel(lbl101)
						ctx.EmitJmp(lbl99)
						ctx.MarkLabel(lbl102)
						ctx.EmitJmp(lbl100)
					}
					ctx.FreeDesc(&d267)
					bbpos_16_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl100)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d269 = args[0]
					d269.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d270 JITValueDesc
					ctx.EnsureDesc(&d269)
					if d269.Loc == LocImm {
						ptrWord, _ := d269.Imm.RawWords()
						d270 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d269.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r161 := ctx.AllocReg()
						ctx.EmitMovRegReg(r161, d269.Reg)
						d270 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r161}
						ctx.BindReg(r161, &d270)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d270)
					ctx.EnsureDesc(&d270)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d270)
					ctx.EnsureDesc(&d270)
					ctx.FreeDesc(&d270)
					ctx.ReclaimUntrackedRegs()
					r162 := ctx.AllocReg()
					ctx.EnsureDesc(&d270)
					ctx.EnsureDesc(&d270)
					if d270.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r162, d270)
					}
					ctx.EmitJmp(lbl98)
					bbpos_16_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl99)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl98)
					d273 = JITValueDesc{Loc: LocReg, Reg: r162}
					ctx.BindReg(r162, &d273)
					ctx.BindReg(r162, &d273)
					if r152 {
						ctx.UnprotectReg(r153)
					}
					if r154 {
						ctx.UnprotectReg(r155)
					}
					if r156 {
						ctx.UnprotectReg(r157)
					}
					ctx.ReclaimUntrackedRegs()
					d274 = args[0]
					d274.ID = 0
					ctx.FreeDesc(&d273)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d274)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *SourceInfo, value SourceInfo) { *dst = value }), []JITValueDesc{d42, d274})
					ctx.FreeDesc(&d274)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d275 JITValueDesc
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						fieldAddr := uintptr(d42.Imm.Int()) + 32
						r163 := ctx.AllocReg()
						r164 := ctx.AllocRegExcept(r163)
						ctx.EmitMovRegMem64(r163, fieldAddr)
						ctx.EmitMovRegMem64(r164, fieldAddr+8)
						d275 = JITValueDesc{Loc: LocRegPair, Reg: r163, Reg2: r164}
						ctx.BindReg(r163, &d275)
						ctx.BindReg(r164, &d275)
					} else {
						off := int32(32)
						baseReg := d42.Reg
						r165 := ctx.AllocRegExcept(baseReg)
						r166 := ctx.AllocRegExcept(baseReg, r165)
						ctx.EmitMovRegMem(r165, baseReg, off)
						ctx.EmitMovRegMem(r166, baseReg, off+8)
						d275 = JITValueDesc{Loc: LocRegPair, Reg: r165, Reg2: r166}
						ctx.BindReg(r165, &d275)
						ctx.BindReg(r166, &d275)
					}
					ctx.StabilizeDescForControlFlow(&d275)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d275)
					if d275.Loc == LocReg {
						ctx.ProtectReg(d275.Reg)
					} else if d275.Loc == LocRegPair {
						ctx.ProtectReg(d275.Reg)
						ctx.ProtectReg(d275.Reg2)
					}
					d276 = d275
					if d276.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d276)
					if d276.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d276, int32(phiBase33)+int32(40), 2)
					} else if d276.Loc == LocInputPair {
						ctx.EnsureDesc(&d276)
						ctx.EmitStoreScmerToStack(d276, int32(phiBase33)+int32(40))
					} else if d276.Loc == LocRegPair || d276.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d276, int32(phiBase33)+int32(40))
					} else {
						ctx.EnsureDesc(&d276)
						ctx.EmitStoreToStack(d276, int32(phiBase33)+int32(40))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase33)+int32(40))+8)
					}
					if d275.Loc == LocReg {
						ctx.UnprotectReg(d275.Reg)
					} else if d275.Loc == LocRegPair {
						ctx.UnprotectReg(d275.Reg)
						ctx.UnprotectReg(d275.Reg2)
					}
					ctx.EmitJmp(lbl79)
					bbpos_2_22 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl90)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d278 = d36
					ctx.EnsureDesc(&d278)
					if d278.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d278.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d278)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d278)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d278)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d278.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d278 = tmpPair
					} else if d278.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d278.Reg), Reg2: ctx.AllocRegExcept(d278.Reg)}
						switch d278.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d278)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d278)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d278)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d278)
						d278 = tmpPair
					} else if d278.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d278.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d278.MemPtr))
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
						d278 = tmpPair
					}
					if d278.Loc != LocRegPair && d278.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d277 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d278}, 2)
					ctx.FreeDesc(&d36)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d277)
					d279 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("(")}
					var d280 JITValueDesc
					if d279.Loc == LocImm {
						ctx.TrackImm(d279.Imm)
						ptrWord, _ := d279.Imm.RawWords()
						d280 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d280.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d280.Reg2, uint64(len(d279.Imm.String())))
						ctx.BindReg(d280.Reg, &d280)
						ctx.BindReg(d280.Reg2, &d280)
					} else {
						d280 = d279
					}
					d281 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d277, d280}, 1)
					ctx.EmitAndRegImm32(d281.Reg, 1)
					d281.Type = tagBool
					ctx.BindReg(d281.Reg, &d281)
					ctx.ReclaimUntrackedRegs()
					d282 = d281
					ctx.EnsureDesc(&d282)
					if d282.Loc != LocImm && d282.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl103 := ctx.ReserveLabel()
					lbl104 := ctx.ReserveLabel()
					lbl105 := ctx.ReserveLabel()
					if d282.Loc == LocImm {
						if d282.Imm.Bool() {
							ctx.MarkLabel(lbl104)
							ctx.EmitJmp(lbl103)
						} else {
							ctx.MarkLabel(lbl105)
							ctx.EmitJmp(lbl91)
						}
					} else {
						ctx.EmitCmpRegImm32(d282.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl104)
						ctx.EmitJmp(lbl105)
						ctx.MarkLabel(lbl104)
						ctx.EmitJmp(lbl103)
						ctx.MarkLabel(lbl105)
						ctx.EmitJmp(lbl91)
					}
					ctx.FreeDesc(&d281)
					bbpos_2_31 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl94)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d224)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value Scmer) { base.value = value }), []JITValueDesc{d42, d224})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					d283 = d42
					_ = d283
					ctx.StabilizeDescForControlFlow(&d283)
					bbpos_17_0 := int32(-1)
					_ = bbpos_17_0
					bbpos_17_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d284 = ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
					ctx.BindReg(d284.Reg, &d284)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d283)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *SourceInfo) { *dst = *src }), []JITValueDesc{d284, d283})
					ctx.ReclaimUntrackedRegs()
					d285 = ctx.EmitGoCallScalar(GoFuncAddr(func() []*SourceInfo { return sourceCoverageInfos }), nil, 3)
					ctx.ReclaimUntrackedRegs()
					d286 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]*SourceInfo { return new([1]*SourceInfo) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d287 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d284)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]*SourceInfo, index int, value *SourceInfo) { dst[index] = value }), []JITValueDesc{d286, d287, d284})
					ctx.ReclaimUntrackedRegs()
					sliceResults288 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]*SourceInfo) []*SourceInfo { return value[0:1:1] }), []JITValueDesc{d286}, []uint8{3}, []uint8{1})
					d289 = sliceResults288[0]
					ctx.ReclaimUntrackedRegs()
					callResults290 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []*SourceInfo) []*SourceInfo { return append(dst, src...) }), []JITValueDesc{d285, d289}, []uint8{3}, []uint8{1})
					d291 = callResults290[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d291)
					ctx.EmitGoCallVoid(GoFuncAddr(func(value []*SourceInfo) { sourceCoverageInfos = value }), []JITValueDesc{d291})
					ctx.ReclaimUntrackedRegs()
					r167 := ctx.AllocReg()
					r168 := ctx.AllocRegExcept(r167)
					ctx.EmitMovRegImm64(r167, 0)
					ctx.EmitMovRegImm64(r168, 0)
					d292 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r167, Reg2: r168}
					ctx.BindReg(r167, &d292)
					ctx.BindReg(r168, &d292)
					ctx.ReclaimUntrackedRegs()
					d293 = args[0]
					d293.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d284)
					ctx.EnsureDesc(&d284)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d284)
					ctx.EnsureDesc(&d284)
					ctx.ReclaimUntrackedRegs()
					d296 = args[0]
					d296.ID = 0
					ctx.ReclaimUntrackedRegs()
					d297 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(14)}
					d298 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d299 = d297
					_ = d299
					ctx.StabilizeDescForControlFlow(&d299)
					d300 = d298
					_ = d300
					ctx.StabilizeDescForControlFlow(&d300)
					bbpos_18_0 := int32(-1)
					_ = bbpos_18_0
					bbpos_18_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d300)
					var d301 JITValueDesc
					if d300.Loc == LocImm {
						d301 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d300.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d300.Reg, 8)
						d301 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d300.Reg}
						ctx.BindReg(d300.Reg, &d301)
					}
					if d301.Loc == LocReg && d300.Loc == LocReg && d301.Reg == d300.Reg {
						ctx.TransferReg(d300.Reg)
						d300.Loc = LocNone
					}
					ctx.FreeDesc(&d300)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d299)
					var d302 JITValueDesc
					if d299.Loc == LocImm {
						d302 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d299.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d299.Reg, int32(255))
						d302 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d299.Reg}
						ctx.BindReg(d299.Reg, &d302)
					}
					if d302.Loc == LocImm {
						d302 = JITValueDesc{Loc: LocImm, Type: d302.Type, Imm: NewInt(int64(uint64(d302.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d302.Reg, 56)
						ctx.EmitShrRegImm8(d302.Reg, 56)
					}
					if d302.Loc == LocReg && d299.Loc == LocReg && d302.Reg == d299.Reg {
						ctx.TransferReg(d299.Reg)
						d299.Loc = LocNone
					}
					ctx.FreeDesc(&d299)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d302)
					ctx.EnsureDesc(&d302)
					var d303 JITValueDesc
					if d302.Loc == LocImm {
						d303 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d302.Imm.Int()))))}
					} else {
						r169 := ctx.AllocReg()
						ctx.EmitMovRegReg(r169, d302.Reg)
						ctx.EmitShlRegImm8(r169, 56)
						ctx.EmitShrRegImm8(r169, 56)
						d303 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r169}
						ctx.BindReg(r169, &d303)
					}
					ctx.FreeDesc(&d302)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d301)
					ctx.EnsureDesc(&d303)
					var d304 JITValueDesc
					if d301.Loc == LocImm && d303.Loc == LocImm {
						d304 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d301.Imm.Int() | d303.Imm.Int())}
					} else if d301.Loc == LocImm && d301.Imm.Int() == 0 {
						d304 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d303.Reg}
						ctx.BindReg(d303.Reg, &d304)
					} else if d303.Loc == LocImm && d303.Imm.Int() == 0 {
						d304 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d301.Reg}
						ctx.BindReg(d301.Reg, &d304)
					} else if d301.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d303.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d301.Imm.Int()))
						ctx.EmitOrInt64(scratch, d303.Reg)
						d304 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d304)
					} else if d303.Loc == LocImm {
						if d303.Imm.Int() >= -2147483648 && d303.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d301.Reg, int32(d303.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d303.Imm.Int()))
							ctx.EmitOrInt64(d301.Reg, RegR11)
						}
						d304 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d301.Reg}
						ctx.BindReg(d301.Reg, &d304)
					} else {
						ctx.EmitOrInt64(d301.Reg, d303.Reg)
						d304 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d301.Reg}
						ctx.BindReg(d301.Reg, &d304)
					}
					if d304.Loc == LocReg && d301.Loc == LocReg && d304.Reg == d301.Reg {
						ctx.TransferReg(d301.Reg)
						d301.Loc = LocNone
					}
					ctx.FreeDesc(&d301)
					ctx.FreeDesc(&d303)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d304)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d284)
					ctx.EnsureDesc(&d284)
					ctx.EmitMovToReg(d293.Reg, d284)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d304)
					ctx.EnsureDesc(&d304)
					ctx.EmitMovToReg(d296.Reg2, d304)
					ctx.FreeDesc(&d304)
					ctx.ReclaimUntrackedRegs()
					d305 = d292
					_ = d305
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d305)
					ctx.ReclaimUntrackedRegs()
					d306 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d306)
					ctx.BindReg(r50, &d306)
					ctx.EmitMovPairToResult(&d305, &d306)
					ctx.EmitJmp(lbl6)
					bbpos_2_20 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl103)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d307 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d308 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					var d309 JITValueDesc
					ctx.EnsureDesc(&d307)
					if d307.Loc == LocRegPair || d307.Loc == LocRegTriple {
						d309 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d307.Reg2}
						ctx.BindReg(d307.Reg2, &d309)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d307)
					ctx.EnsureDesc(&d308)
					ctx.EnsureDesc(&d309)
					var d311 JITValueDesc
					if d309.Loc == LocImm && d308.Loc == LocImm {
						d311 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d309.Imm.Int() - d308.Imm.Int())}
					} else {
						r170 := ctx.AllocReg()
						if d309.Loc == LocImm {
							ctx.EmitMovRegImm64(r170, uint64(d309.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r170, d309.Reg)
						}
						if d308.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d308.Imm.Int()))
							ctx.EmitSubInt64(r170, RegR11)
						} else {
							ctx.EmitSubInt64(r170, d308.Reg)
						}
						d311 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r170}
						ctx.BindReg(r170, &d311)
					}
					var d312 JITValueDesc
					if d307.Loc == LocImm && d308.Loc == LocImm {
						d312 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d307.Imm.Int() + d308.Imm.Int()*16)}
					} else {
						r171 := ctx.AllocReg()
						if d307.Loc == LocImm {
							ctx.EmitMovRegImm64(r171, uint64(d307.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r171, d307.Reg)
						}
						if d308.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d308.Imm.Int()*16))
							ctx.EmitAddInt64(r171, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r171, d308.Reg)
							ctx.EmitMovRegReg(offsetReg, d308.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r171, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d312 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r171}
						ctx.BindReg(r171, &d312)
					}
					var d313 JITValueDesc
					var r172 Reg
					var r173 Reg
					ctx.SyncDesc(&d312)
					ctx.EnsureDesc(&d312)
					if d312.Loc == LocImm {
						r172 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r172, uint64(d312.Imm.Int()))
					} else {
						r172 = d312.Reg
					}
					ctx.ProtectReg(r172)
					ctx.SyncDesc(&d311)
					ctx.EnsureDesc(&d311)
					if d311.Loc == LocImm {
						r173 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r173, uint64(d311.Imm.Int()))
					} else {
						r173 = d311.Reg
					}
					ctx.ProtectReg(r173)
					r174 := ctx.EmitSliceCapAfterLow(&d307, &d308, r172, r173)
					ctx.UnprotectReg(r173)
					ctx.UnprotectReg(r172)
					d313 = JITValueDesc{Loc: LocRegTriple, Reg: r172, Reg2: r173, Reg3: r174}
					ctx.BindReg(r172, &d313)
					ctx.BindReg(r173, &d313)
					ctx.BindReg(r174, &d313)
					ctx.BindReg(r172, &d313)
					ctx.BindReg(r173, &d313)
					ctx.BindReg(r174, &d313)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d313)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[]Scmer, value []Scmer) { *dst = value }), []JITValueDesc{d32, d313})
					ctx.ReclaimUntrackedRegs()
					stackArray314 = ctx.AllocStack(int32(16))
					_ = stackArray314
					ctx.ReclaimUntrackedRegs()
					d315 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d315
					ctx.StabilizeDescForControlFlow(&d315)
					ctx.ReclaimUntrackedRegs()
					d316 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("list")}
					ctx.EnsureDesc(&d316)
					if d316.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d316.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d316.Imm)
						ptrWord, _ := d316.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d316.Imm.String())))
						d316 = tmpPair
					} else if d316.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d316.Type, Reg: ctx.AllocRegExcept(d316.Reg), Reg2: ctx.AllocRegExcept(d316.Reg)}
						switch d316.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d316)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d316)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d316)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d316)
						d316 = tmpPair
					}
					if d316.Loc != LocRegPair && d316.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
					}
					ctx.SyncDesc(&d316)
					d317 = ctx.EmitGoCallScalar(GoFuncAddr(NewSymbol), []JITValueDesc{d316}, 2)
					ctx.BindReg(d317.Reg, &d317)
					ctx.BindReg(d317.Reg2, &d317)
					ctx.FreeDesc(&d316)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d317)
					ctx.EnsureDesc(&d317)
					ctx.EmitStoreScmerToStack(d317, int32(stackArray314)+int32(0))
					ctx.FreeDesc(&d317)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d315)
					if d315.Loc == LocReg {
						ctx.ProtectReg(d315.Reg)
					} else if d315.Loc == LocRegPair {
						ctx.ProtectReg(d315.Reg)
						ctx.ProtectReg(d315.Reg2)
					}
					d318 = d315
					if d318.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d318)
					if d318.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d318, int32(phiBase33)+int32(56), 3)
					} else {
						if d318.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d318.Reg, RegRSP, int32(phiBase33)+int32(56))
						ctx.EmitStoreRegMem(d318.Reg2, RegRSP, int32(phiBase33)+int32(56)+8)
						ctx.EmitStoreRegMem(d318.Reg3, RegRSP, int32(phiBase33)+int32(56)+16)
					}
					if d315.Loc == LocReg {
						ctx.UnprotectReg(d315.Reg)
					} else if d315.Loc == LocRegPair {
						ctx.UnprotectReg(d315.Reg)
						ctx.UnprotectReg(d315.Reg2)
					}
					bbpos_2_23 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d37)
					ctx.ReclaimUntrackedRegs()
					d319 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					var d320 JITValueDesc
					if d319.SliceSizeKnown {
						d320 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d319.KnownSliceLen))}
					} else if d319.Loc == LocImm {
						d320 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d319.StackOff))}
					} else if d319.Loc == LocStackTriple {
						d320 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d319.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d319)
						if d319.Loc == LocRegPair || d319.Loc == LocRegTriple {
							d320 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d319.Reg2, ID: 0}
						} else if d319.Loc == LocReg {
							d320 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d319.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d320)
					var d321 JITValueDesc
					if d320.Loc == LocImm {
						d321 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d320.Imm.Int() == 0)}
					} else {
						r175 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d320.Reg, 0)
						ctx.EmitSetcc(r175, CondEqual)
						d321 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r175}
						ctx.BindReg(r175, &d321)
					}
					ctx.FreeDesc(&d320)
					ctx.ReclaimUntrackedRegs()
					d322 = d321
					ctx.EnsureDesc(&d322)
					if d322.Loc != LocImm && d322.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl106 := ctx.ReserveLabel()
					lbl107 := ctx.ReserveLabel()
					lbl108 := ctx.ReserveLabel()
					lbl109 := ctx.ReserveLabel()
					if d322.Loc == LocImm {
						if d322.Imm.Bool() {
							ctx.MarkLabel(lbl108)
							ctx.EmitJmp(lbl106)
						} else {
							ctx.MarkLabel(lbl109)
							ctx.EmitJmp(lbl107)
						}
					} else {
						ctx.EmitCmpRegImm32(d322.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl108)
						ctx.EmitJmp(lbl109)
						ctx.MarkLabel(lbl108)
						ctx.EmitJmp(lbl106)
						ctx.MarkLabel(lbl109)
						ctx.EmitJmp(lbl107)
					}
					ctx.FreeDesc(&d321)
					bbpos_2_25 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl107)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d323 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d324 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d326 = ctx.EmitSliceElementAddress(&d323, &d324, 16)
					ctx.EnsureDesc(&d326)
					r176 := ctx.AllocRegExcept(d326.Reg)
					ctx.EmitMovRegMem(r176, d326.Reg, 8)
					ctx.EmitMovRegMem(d326.Reg, d326.Reg, 0)
					d325 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d326.Reg, Reg2: r176}
					ctx.BindReg(d326.Reg, &d325)
					ctx.BindReg(r176, &d325)
					ctx.StabilizeDescForControlFlow(&d325)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d325)
					r177 := d325.Loc == LocReg || d325.Loc == LocRegPair || d325.Loc == LocRegTriple
					r178 := d325.Reg
					if r177 {
						ctx.ProtectReg(r178)
					}
					r179 := d325.Loc == LocRegPair || d325.Loc == LocRegTriple
					r180 := d325.Reg2
					if r179 {
						ctx.ProtectReg(r180)
					}
					r181 := d325.Loc == LocRegTriple
					r182 := d325.Reg3
					if r181 {
						ctx.ProtectReg(r182)
					}
					lbl110 := ctx.ReserveLabel()
					bbpos_19_0 := int32(-1)
					_ = bbpos_19_0
					bbpos_19_1 := int32(-1)
					_ = bbpos_19_1
					bbpos_19_2 := int32(-1)
					_ = bbpos_19_2
					bbpos_19_3 := int32(-1)
					_ = bbpos_19_3
					bbpos_19_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r183 := ctx.AllocReg()
					r184 := ctx.AllocRegExcept(r183)
					ctx.EmitMovRegImm64(r183, 0)
					ctx.EmitMovRegImm64(r184, 0)
					d327 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r183, Reg2: r184}
					ctx.BindReg(r183, &d327)
					ctx.BindReg(r184, &d327)
					ctx.StabilizeDescForControlFlow(&d327)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d325)
					ctx.ReclaimUntrackedRegs()
					d328 = args[0]
					d328.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d329 JITValueDesc
					ctx.EnsureDesc(&d328)
					if d328.Loc == LocImm {
						ptrWord, _ := d328.Imm.RawWords()
						d329 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d328.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r185 := ctx.AllocReg()
						ctx.EmitMovRegReg(r185, d328.Reg)
						d329 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r185}
						ctx.BindReg(r185, &d329)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d329)
					d331 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerIntSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d329)
					ctx.EnsureDesc(&d331)
					ctx.EnsureDesc(&d329)
					ctx.EnsureDesc(&d331)
					var d330 JITValueDesc
					if d329.Loc == LocImm && d331.Loc == LocImm {
						d330 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d329.Imm.Int() == d331.Imm.Int())}
					} else if d331.Loc == LocImm {
						r186 := ctx.AllocReg()
						if d331.Imm.Int() >= -2147483648 && d331.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d329.Reg, int32(d331.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d331.Imm.Int()))
							ctx.EmitCmpInt64(d329.Reg, RegR11)
						}
						ctx.EmitSetcc(r186, CondEqual)
						d330 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r186}
						ctx.BindReg(r186, &d330)
					} else if d329.Loc == LocImm {
						r187 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d329.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d331.Reg)
						ctx.EmitSetcc(r187, CondEqual)
						d330 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r187}
						ctx.BindReg(r187, &d330)
					} else {
						r188 := ctx.AllocReg()
						ctx.EmitCmpInt64(d329.Reg, d331.Reg)
						ctx.EmitSetcc(r188, CondEqual)
						d330 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r188}
						ctx.BindReg(r188, &d330)
					}
					ctx.FreeDesc(&d329)
					ctx.ReclaimUntrackedRegs()
					d332 = d330
					ctx.EnsureDesc(&d332)
					if d332.Loc != LocImm && d332.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl111 := ctx.ReserveLabel()
					lbl112 := ctx.ReserveLabel()
					lbl113 := ctx.ReserveLabel()
					lbl114 := ctx.ReserveLabel()
					if d332.Loc == LocImm {
						if d332.Imm.Bool() {
							ctx.MarkLabel(lbl113)
							ctx.EmitJmp(lbl111)
						} else {
							ctx.MarkLabel(lbl114)
							ctx.EmitJmp(lbl112)
						}
					} else {
						ctx.EmitCmpRegImm32(d332.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl113)
						ctx.EmitJmp(lbl114)
						ctx.MarkLabel(lbl113)
						ctx.EmitJmp(lbl111)
						ctx.MarkLabel(lbl114)
						ctx.EmitJmp(lbl112)
					}
					ctx.FreeDesc(&d330)
					bbpos_19_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl112)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d333 = args[0]
					d333.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d334 JITValueDesc
					ctx.EnsureDesc(&d333)
					if d333.Loc == LocImm {
						ptrWord, _ := d333.Imm.RawWords()
						d334 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d333.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r189 := ctx.AllocReg()
						ctx.EmitMovRegReg(r189, d333.Reg)
						d334 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r189}
						ctx.BindReg(r189, &d334)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d334)
					d336 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&scmerFloatSentinel)))), NoHeapPointer: true, Rooted: true}
					ctx.EnsureDesc(&d334)
					ctx.EnsureDesc(&d336)
					ctx.EnsureDesc(&d334)
					ctx.EnsureDesc(&d336)
					var d335 JITValueDesc
					if d334.Loc == LocImm && d336.Loc == LocImm {
						d335 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d334.Imm.Int() == d336.Imm.Int())}
					} else if d336.Loc == LocImm {
						r190 := ctx.AllocReg()
						if d336.Imm.Int() >= -2147483648 && d336.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d334.Reg, int32(d336.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d336.Imm.Int()))
							ctx.EmitCmpInt64(d334.Reg, RegR11)
						}
						ctx.EmitSetcc(r190, CondEqual)
						d335 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r190}
						ctx.BindReg(r190, &d335)
					} else if d334.Loc == LocImm {
						r191 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d334.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d336.Reg)
						ctx.EmitSetcc(r191, CondEqual)
						d335 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r191}
						ctx.BindReg(r191, &d335)
					} else {
						r192 := ctx.AllocReg()
						ctx.EmitCmpInt64(d334.Reg, d336.Reg)
						ctx.EmitSetcc(r192, CondEqual)
						d335 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r192}
						ctx.BindReg(r192, &d335)
					}
					ctx.FreeDesc(&d334)
					ctx.ReclaimUntrackedRegs()
					d337 = d335
					ctx.EnsureDesc(&d337)
					if d337.Loc != LocImm && d337.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl115 := ctx.ReserveLabel()
					lbl116 := ctx.ReserveLabel()
					lbl117 := ctx.ReserveLabel()
					if d337.Loc == LocImm {
						if d337.Imm.Bool() {
							ctx.MarkLabel(lbl116)
							ctx.EmitJmp(lbl111)
						} else {
							ctx.MarkLabel(lbl117)
							ctx.EmitJmp(lbl115)
						}
					} else {
						ctx.EmitCmpRegImm32(d337.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl116)
						ctx.EmitJmp(lbl117)
						ctx.MarkLabel(lbl116)
						ctx.EmitJmp(lbl111)
						ctx.MarkLabel(lbl117)
						ctx.EmitJmp(lbl115)
					}
					ctx.FreeDesc(&d335)
					bbpos_19_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl115)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d338 = args[0]
					d338.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d339 JITValueDesc
					ctx.EnsureDesc(&d338)
					if d338.Loc == LocImm {
						_, auxWord := d338.Imm.RawWords()
						d339 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(auxWord))}
					} else {
						if d338.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r193 := ctx.AllocReg()
						ctx.EmitMovRegReg(r193, d338.Reg2)
						d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r193}
						ctx.BindReg(r193, &d339)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d339)
					d340 = d339
					_ = d340
					ctx.StabilizeDescForControlFlow(&d340)
					bbpos_20_0 := int32(-1)
					_ = bbpos_20_0
					bbpos_20_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d340)
					var d341 JITValueDesc
					if d340.Loc == LocImm {
						d341 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d340.Imm.Int() & 255)}
					} else {
						r194 := ctx.AllocRegExcept(d340.Reg)
						ctx.EmitMovRegReg(r194, d340.Reg)
						ctx.EmitAndRegImm32(r194, int32(255))
						d341 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r194}
						ctx.BindReg(r194, &d341)
					}
					if d341.Loc == LocReg && d340.Loc == LocReg && d341.Reg == d340.Reg {
						ctx.TransferReg(d340.Reg)
						d340.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d341)
					ctx.EnsureDesc(&d341)
					var d342 JITValueDesc
					if d341.Loc == LocImm {
						d342 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint8(uint64(d341.Imm.Int()))))}
					} else {
						r195 := ctx.AllocReg()
						ctx.EmitMovRegReg(r195, d341.Reg)
						ctx.EmitShlRegImm8(r195, 56)
						ctx.EmitShrRegImm8(r195, 56)
						d342 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r195}
						ctx.BindReg(r195, &d342)
					}
					ctx.FreeDesc(&d341)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d342)
					ctx.FreeDesc(&d339)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d342)
					var d343 JITValueDesc
					if d342.Loc == LocImm {
						d343 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d342.Imm.Int()) == uint64(0x2))}
					} else {
						r196 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d342.Reg, 2)
						ctx.EmitSetcc(r196, CondEqual)
						d343 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r196}
						ctx.BindReg(r196, &d343)
					}
					ctx.FreeDesc(&d342)
					ctx.ReclaimUntrackedRegs()
					r197 := ctx.AllocReg()
					ctx.EnsureDesc(&d343)
					ctx.EnsureDesc(&d343)
					if d343.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r197, d343)
					}
					ctx.EmitJmp(lbl110)
					bbpos_19_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl111)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d344 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d344)
					if d344.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r197, d344)
					}
					ctx.EmitJmp(lbl110)
					ctx.MarkLabel(lbl110)
					d345 = JITValueDesc{Loc: LocReg, Reg: r197}
					ctx.BindReg(r197, &d345)
					ctx.BindReg(r197, &d345)
					if r177 {
						ctx.UnprotectReg(r178)
					}
					if r179 {
						ctx.UnprotectReg(r180)
					}
					if r181 {
						ctx.UnprotectReg(r182)
					}
					ctx.ReclaimUntrackedRegs()
					d346 = d345
					ctx.EnsureDesc(&d346)
					if d346.Loc != LocImm && d346.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl118 := ctx.ReserveLabel()
					lbl119 := ctx.ReserveLabel()
					lbl120 := ctx.ReserveLabel()
					lbl121 := ctx.ReserveLabel()
					if d346.Loc == LocImm {
						if d346.Imm.Bool() {
							ctx.MarkLabel(lbl120)
							ctx.EmitJmp(lbl118)
						} else {
							ctx.MarkLabel(lbl121)
							ctx.EmitJmp(lbl119)
						}
					} else {
						ctx.EmitCmpRegImm32(d346.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl120)
						ctx.EmitJmp(lbl121)
						ctx.MarkLabel(lbl120)
						ctx.EmitJmp(lbl118)
						ctx.MarkLabel(lbl121)
						ctx.EmitJmp(lbl119)
					}
					ctx.FreeDesc(&d345)
					bbpos_2_27 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl119)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					if d32.Loc == LocRegPair || d32.Loc == LocStackPair || d32.Loc == LocRegTriple || d32.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d32)
					d347 = ctx.EmitGoCallScalar(GoFuncAddr(readFrom), []JITValueDesc{d32}, 2)
					ctx.BindReg(d347.Reg, &d347)
					ctx.BindReg(d347.Reg2, &d347)
					ctx.ReclaimUntrackedRegs()
					stackArray348 = ctx.AllocStack(int32(16))
					_ = stackArray348
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d347)
					ctx.EnsureDesc(&d347)
					ctx.EmitStoreScmerToStack(d347, int32(stackArray348)+int32(0))
					ctx.FreeDesc(&d347)
					ctx.ReclaimUntrackedRegs()
					d349 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d349
					ctx.ReclaimUntrackedRegs()
					r198 := ctx.AllocReg()
					r199 := ctx.AllocRegExcept(r198)
					r200 := ctx.AllocRegExcept(r198, r199)
					d350 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r198, Reg2: r199, Reg3: r200}
					ctx.BindReg(r198, &d350)
					ctx.BindReg(r199, &d350)
					ctx.BindReg(r200, &d350)
					ctx.BindReg(r198, &d350)
					ctx.BindReg(r199, &d350)
					ctx.BindReg(r200, &d350)
					ctx.EmitLeaRegMem(d350.Reg, ctx.StackReg, int32(stackArray348))
					ctx.EmitMovRegImm64(d350.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d350.Reg3, uint64(1))
					callResults351 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d37, d350}, []uint8{3}, []uint8{1})
					d352 = callResults351[0]
					d353 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase33) + int32(56)}
					ctx.EmitCopyDescWords(&d353, &d352, 3)
					ctx.FreeDesc(&d352)
					d352 = d353
					ctx.StabilizeDescForControlFlow(&d352)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_2_23)
					bbpos_2_24 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl106)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					bbpos_2_28 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl118)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d355 = d325
					ctx.EnsureDesc(&d355)
					if d355.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d355.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d355)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d355)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d355)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d355.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d355 = tmpPair
					} else if d355.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d355.Reg), Reg2: ctx.AllocRegExcept(d355.Reg)}
						switch d355.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d355)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d355)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d355)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d355)
						d355 = tmpPair
					} else if d355.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d355.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d355.MemPtr))
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
						d355 = tmpPair
					}
					if d355.Loc != LocRegPair && d355.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d354 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d355}, 2)
					ctx.FreeDesc(&d325)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d354)
					d356 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString(")")}
					var d357 JITValueDesc
					if d356.Loc == LocImm {
						ctx.TrackImm(d356.Imm)
						ptrWord, _ := d356.Imm.RawWords()
						d357 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d357.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d357.Reg2, uint64(len(d356.Imm.String())))
						ctx.BindReg(d357.Reg, &d357)
						ctx.BindReg(d357.Reg2, &d357)
					} else {
						d357 = d356
					}
					d358 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d354, d357}, 1)
					ctx.EmitAndRegImm32(d358.Reg, 1)
					d358.Type = tagBool
					ctx.BindReg(d358.Reg, &d358)
					ctx.ReclaimUntrackedRegs()
					d359 = d358
					ctx.EnsureDesc(&d359)
					if d359.Loc != LocImm && d359.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl122 := ctx.ReserveLabel()
					lbl123 := ctx.ReserveLabel()
					lbl124 := ctx.ReserveLabel()
					if d359.Loc == LocImm {
						if d359.Imm.Bool() {
							ctx.MarkLabel(lbl123)
							ctx.EmitJmp(lbl122)
						} else {
							ctx.MarkLabel(lbl124)
							ctx.EmitJmp(lbl119)
						}
					} else {
						ctx.EmitCmpRegImm32(d359.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl123)
						ctx.EmitJmp(lbl124)
						ctx.MarkLabel(lbl123)
						ctx.EmitJmp(lbl122)
						ctx.MarkLabel(lbl124)
						ctx.EmitJmp(lbl119)
					}
					ctx.FreeDesc(&d358)
					bbpos_2_26 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl122)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d360 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *[]Scmer) []Scmer { return *value }), []JITValueDesc{d32}, 3)
					ctx.ReclaimUntrackedRegs()
					d361 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					var d362 JITValueDesc
					ctx.EnsureDesc(&d360)
					if d360.Loc == LocRegPair || d360.Loc == LocRegTriple {
						d362 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d360.Reg2}
						ctx.BindReg(d360.Reg2, &d362)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d360)
					ctx.EnsureDesc(&d361)
					ctx.EnsureDesc(&d362)
					var d364 JITValueDesc
					if d362.Loc == LocImm && d361.Loc == LocImm {
						d364 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d362.Imm.Int() - d361.Imm.Int())}
					} else {
						r201 := ctx.AllocReg()
						if d362.Loc == LocImm {
							ctx.EmitMovRegImm64(r201, uint64(d362.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r201, d362.Reg)
						}
						if d361.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d361.Imm.Int()))
							ctx.EmitSubInt64(r201, RegR11)
						} else {
							ctx.EmitSubInt64(r201, d361.Reg)
						}
						d364 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r201}
						ctx.BindReg(r201, &d364)
					}
					var d365 JITValueDesc
					if d360.Loc == LocImm && d361.Loc == LocImm {
						d365 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d360.Imm.Int() + d361.Imm.Int()*16)}
					} else {
						r202 := ctx.AllocReg()
						if d360.Loc == LocImm {
							ctx.EmitMovRegImm64(r202, uint64(d360.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r202, d360.Reg)
						}
						if d361.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d361.Imm.Int()*16))
							ctx.EmitAddInt64(r202, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r202, d361.Reg)
							ctx.EmitMovRegReg(offsetReg, d361.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r202, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d365 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r202}
						ctx.BindReg(r202, &d365)
					}
					var d366 JITValueDesc
					var r203 Reg
					var r204 Reg
					ctx.SyncDesc(&d365)
					ctx.EnsureDesc(&d365)
					if d365.Loc == LocImm {
						r203 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r203, uint64(d365.Imm.Int()))
					} else {
						r203 = d365.Reg
					}
					ctx.ProtectReg(r203)
					ctx.SyncDesc(&d364)
					ctx.EnsureDesc(&d364)
					if d364.Loc == LocImm {
						r204 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r204, uint64(d364.Imm.Int()))
					} else {
						r204 = d364.Reg
					}
					ctx.ProtectReg(r204)
					r205 := ctx.EmitSliceCapAfterLow(&d360, &d361, r203, r204)
					ctx.UnprotectReg(r204)
					ctx.UnprotectReg(r203)
					d366 = JITValueDesc{Loc: LocRegTriple, Reg: r203, Reg2: r204, Reg3: r205}
					ctx.BindReg(r203, &d366)
					ctx.BindReg(r204, &d366)
					ctx.BindReg(r205, &d366)
					ctx.BindReg(r203, &d366)
					ctx.BindReg(r204, &d366)
					ctx.BindReg(r205, &d366)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d366)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[]Scmer, value []Scmer) { *dst = value }), []JITValueDesc{d32, d366})
					ctx.ReclaimUntrackedRegs()
					d367 = ctx.EmitNewSliceFromGoSlice(&d37)
					ctx.StabilizeDescForControlFlow(&d367)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d368 JITValueDesc
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						fieldAddr := uintptr(d42.Imm.Int()) + 0
						r206 := ctx.AllocReg()
						r207 := ctx.AllocRegExcept(r206)
						r208 := ctx.AllocRegExcept(r206, r207)
						ctx.EmitMovRegMem64(r206, fieldAddr)
						ctx.EmitMovRegMem64(r207, fieldAddr+8)
						ctx.EmitMovRegMem64(r208, fieldAddr+16)
						d368 = JITValueDesc{Loc: LocRegTriple, Reg: r206, Reg2: r207, Reg3: r208}
						ctx.BindReg(r206, &d368)
						ctx.BindReg(r207, &d368)
						ctx.BindReg(r208, &d368)
					} else {
						off := int32(0)
						baseReg := d42.Reg
						r209 := ctx.AllocRegExcept(baseReg)
						r210 := ctx.AllocRegExcept(baseReg, r209)
						r211 := ctx.AllocRegExcept(baseReg, r209, r210)
						ctx.EmitMovRegMem(r209, baseReg, off)
						ctx.EmitMovRegMem(r210, baseReg, off+8)
						ctx.EmitMovRegMem(r211, baseReg, off+16)
						d368 = JITValueDesc{Loc: LocRegTriple, Reg: r209, Reg2: r210, Reg3: r211}
						ctx.BindReg(r209, &d368)
						ctx.BindReg(r210, &d368)
						ctx.BindReg(r211, &d368)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d368)
					var d369 JITValueDesc
					if d368.Loc == LocImm {
						ctx.TrackImm(d368.Imm)
						ptrWord, _ := d368.Imm.RawWords()
						d369 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d369.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d369.Reg2, uint64(len(d368.Imm.String())))
						ctx.BindReg(d369.Reg, &d369)
						ctx.BindReg(d369.Reg2, &d369)
					} else {
						d369 = d368
					}
					d370 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("")}
					var d371 JITValueDesc
					if d370.Loc == LocImm {
						ctx.TrackImm(d370.Imm)
						ptrWord, _ := d370.Imm.RawWords()
						d371 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d371.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d371.Reg2, uint64(len(d370.Imm.String())))
						ctx.BindReg(d371.Reg, &d371)
						ctx.BindReg(d371.Reg2, &d371)
					} else {
						d371 = d370
					}
					d372 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d369, d371}, 1)
					ctx.EmitAndRegImm32(d372.Reg, 1)
					ctx.EmitCmpRegImm32(d372.Reg, 0)
					ctx.EmitSetcc(d372.Reg, CondEqual)
					d372.Type = tagBool
					ctx.BindReg(d372.Reg, &d372)
					ctx.ReclaimUntrackedRegs()
					d373 = d372
					ctx.EnsureDesc(&d373)
					if d373.Loc != LocImm && d373.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl125 := ctx.ReserveLabel()
					lbl126 := ctx.ReserveLabel()
					lbl127 := ctx.ReserveLabel()
					lbl128 := ctx.ReserveLabel()
					if d373.Loc == LocImm {
						if d373.Imm.Bool() {
							ctx.MarkLabel(lbl127)
							ctx.EmitJmp(lbl125)
						} else {
							ctx.MarkLabel(lbl128)
							ctx.EmitJmp(lbl126)
						}
					} else {
						ctx.EmitCmpRegImm32(d373.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl127)
						ctx.EmitJmp(lbl128)
						ctx.MarkLabel(lbl127)
						ctx.EmitJmp(lbl125)
						ctx.MarkLabel(lbl128)
						ctx.EmitJmp(lbl126)
					}
					ctx.FreeDesc(&d372)
					bbpos_2_30 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl126)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d374 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d374)
					ctx.BindReg(r50, &d374)
					ctx.EmitMovPairToResult(&d367, &d374)
					ctx.EmitJmp(lbl6)
					bbpos_2_29 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl125)
					ctx.ResolveFixups()
					d34 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(0)}
					d35 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(16)}
					d36 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(40)}
					d37 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase33) + int32(56)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d367)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d367)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *SourceInfo, value Scmer) { base.value = value }), []JITValueDesc{d42, d367})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d42)
					d375 = d42
					_ = d375
					ctx.StabilizeDescForControlFlow(&d375)
					bbpos_21_0 := int32(-1)
					_ = bbpos_21_0
					bbpos_21_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d376 = ctx.EmitGoCallScalar(GoFuncAddr(func() *SourceInfo { return new(SourceInfo) }), nil, 1)
					ctx.BindReg(d376.Reg, &d376)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d375)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *SourceInfo) { *dst = *src }), []JITValueDesc{d376, d375})
					ctx.ReclaimUntrackedRegs()
					d377 = ctx.EmitGoCallScalar(GoFuncAddr(func() []*SourceInfo { return sourceCoverageInfos }), nil, 3)
					ctx.ReclaimUntrackedRegs()
					d378 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]*SourceInfo { return new([1]*SourceInfo) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d379 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d376)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]*SourceInfo, index int, value *SourceInfo) { dst[index] = value }), []JITValueDesc{d378, d379, d376})
					ctx.ReclaimUntrackedRegs()
					sliceResults380 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]*SourceInfo) []*SourceInfo { return value[0:1:1] }), []JITValueDesc{d378}, []uint8{3}, []uint8{1})
					d381 = sliceResults380[0]
					ctx.ReclaimUntrackedRegs()
					callResults382 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []*SourceInfo) []*SourceInfo { return append(dst, src...) }), []JITValueDesc{d377, d381}, []uint8{3}, []uint8{1})
					d383 = callResults382[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d383)
					ctx.EmitGoCallVoid(GoFuncAddr(func(value []*SourceInfo) { sourceCoverageInfos = value }), []JITValueDesc{d383})
					ctx.ReclaimUntrackedRegs()
					r212 := ctx.AllocReg()
					r213 := ctx.AllocRegExcept(r212)
					ctx.EmitMovRegImm64(r212, 0)
					ctx.EmitMovRegImm64(r213, 0)
					d384 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r212, Reg2: r213}
					ctx.BindReg(r212, &d384)
					ctx.BindReg(r213, &d384)
					ctx.ReclaimUntrackedRegs()
					d385 = args[0]
					d385.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d376)
					ctx.EnsureDesc(&d376)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d376)
					ctx.EnsureDesc(&d376)
					ctx.ReclaimUntrackedRegs()
					d388 = args[0]
					d388.ID = 0
					ctx.ReclaimUntrackedRegs()
					d389 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(14)}
					d390 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d391 = d389
					_ = d391
					ctx.StabilizeDescForControlFlow(&d391)
					d392 = d390
					_ = d392
					ctx.StabilizeDescForControlFlow(&d392)
					bbpos_22_0 := int32(-1)
					_ = bbpos_22_0
					bbpos_22_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d392)
					var d393 JITValueDesc
					if d392.Loc == LocImm {
						d393 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d392.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d392.Reg, 8)
						d393 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d392.Reg}
						ctx.BindReg(d392.Reg, &d393)
					}
					if d393.Loc == LocReg && d392.Loc == LocReg && d393.Reg == d392.Reg {
						ctx.TransferReg(d392.Reg)
						d392.Loc = LocNone
					}
					ctx.FreeDesc(&d392)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d391)
					var d394 JITValueDesc
					if d391.Loc == LocImm {
						d394 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d391.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d391.Reg, int32(255))
						d394 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d391.Reg}
						ctx.BindReg(d391.Reg, &d394)
					}
					if d394.Loc == LocImm {
						d394 = JITValueDesc{Loc: LocImm, Type: d394.Type, Imm: NewInt(int64(uint64(d394.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d394.Reg, 56)
						ctx.EmitShrRegImm8(d394.Reg, 56)
					}
					if d394.Loc == LocReg && d391.Loc == LocReg && d394.Reg == d391.Reg {
						ctx.TransferReg(d391.Reg)
						d391.Loc = LocNone
					}
					ctx.FreeDesc(&d391)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d394)
					ctx.EnsureDesc(&d394)
					var d395 JITValueDesc
					if d394.Loc == LocImm {
						d395 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d394.Imm.Int()))))}
					} else {
						r214 := ctx.AllocReg()
						ctx.EmitMovRegReg(r214, d394.Reg)
						ctx.EmitShlRegImm8(r214, 56)
						ctx.EmitShrRegImm8(r214, 56)
						d395 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r214}
						ctx.BindReg(r214, &d395)
					}
					ctx.FreeDesc(&d394)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d393)
					ctx.EnsureDesc(&d395)
					var d396 JITValueDesc
					if d393.Loc == LocImm && d395.Loc == LocImm {
						d396 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d393.Imm.Int() | d395.Imm.Int())}
					} else if d393.Loc == LocImm && d393.Imm.Int() == 0 {
						d396 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d395.Reg}
						ctx.BindReg(d395.Reg, &d396)
					} else if d395.Loc == LocImm && d395.Imm.Int() == 0 {
						d396 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d393.Reg}
						ctx.BindReg(d393.Reg, &d396)
					} else if d393.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d395.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d393.Imm.Int()))
						ctx.EmitOrInt64(scratch, d395.Reg)
						d396 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d396)
					} else if d395.Loc == LocImm {
						if d395.Imm.Int() >= -2147483648 && d395.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d393.Reg, int32(d395.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d395.Imm.Int()))
							ctx.EmitOrInt64(d393.Reg, RegR11)
						}
						d396 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d393.Reg}
						ctx.BindReg(d393.Reg, &d396)
					} else {
						ctx.EmitOrInt64(d393.Reg, d395.Reg)
						d396 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d393.Reg}
						ctx.BindReg(d393.Reg, &d396)
					}
					if d396.Loc == LocReg && d393.Loc == LocReg && d396.Reg == d393.Reg {
						ctx.TransferReg(d393.Reg)
						d393.Loc = LocNone
					}
					ctx.FreeDesc(&d393)
					ctx.FreeDesc(&d395)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d396)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d376)
					ctx.EnsureDesc(&d376)
					ctx.EmitMovToReg(d385.Reg, d376)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d396)
					ctx.EnsureDesc(&d396)
					ctx.EmitMovToReg(d388.Reg2, d396)
					ctx.FreeDesc(&d396)
					ctx.ReclaimUntrackedRegs()
					d397 = d384
					_ = d397
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d397)
					ctx.ReclaimUntrackedRegs()
					d398 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d398)
					ctx.BindReg(r50, &d398)
					ctx.EmitMovPairToResult(&d397, &d398)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl6)
					d399 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d399)
					ctx.BindReg(r50, &d399)
					ctx.BindReg(r49, &d399)
					ctx.BindReg(r50, &d399)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d399)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d399)
					if d399.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d399, &result)
						result.Type = d399.Type
					} else {
						switch d399.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d399)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d399)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d399)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d399, &result)
							result.Type = d399.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps400 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps400)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["serialize"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d0.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d0.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d0 = tmpPair
				} else if d0.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
					switch d0.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d0)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d0)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d0)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d0)
					d0 = tmpPair
				}
				if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (SerializeToString arg0)")
				}
				d1 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(SerializeToString), []JITValueDesc{d0, d1}, 2)
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d2}, 2)
				if result.Loc == LocAny {
					return d3
				}
				ctx.EmitMovPairToResult(&d3, &result)
				result.Type = tagString
				return result
				return result
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["pretty_print"].Fn, args, result)
				}
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
				var d22 JITValueDesc
				_ = d22
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
				var d29 JITValueDesc
				_ = d29
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() >= 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 2)
						ctx.EmitSetcc(r0, CondSignedGreaterOrEqual)
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
							if ps.General {
							}
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						ps6.PhiValues = make([]JITValueDesc, 1)
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}
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
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}, int32(bbs[2].PhiBase)+int32(0))
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
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					ctx.EnsureDesc(&d18)
					d19 = d18
					_ = d19
					ctx.StabilizeDescForControlFlow(&d19)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d20 JITValueDesc
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int())}
					} else if d19.Type == tagInt && d19.Loc == LocRegPair {
						ctx.FreeReg(d19.Reg)
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
						ctx.BindReg(d19.Reg2, &d20)
						ctx.BindReg(d19.Reg2, &d20)
					} else if d19.Type == tagInt && d19.Loc == LocReg {
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
						ctx.BindReg(d19.Reg, &d20)
						ctx.BindReg(d19.Reg, &d20)
					} else {
						d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d19}, 1)
						d20.Type = tagInt
						ctx.BindReg(d20.Reg, &d20)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d20)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d18)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d22 = d20
						if d22.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d22)
						ctx.EmitStoreToStack(d22, int32(bbs[2].PhiBase)+int32(0))
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
					}
					ps23 := PhiState{General: ps.General}
					ps23.OverlayValues = make([]JITValueDesc, 23)
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[10] = d10
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					ps23.OverlayValues[20] = d20
					ps23.OverlayValues[21] = d21
					ps23.OverlayValues[22] = d22
					ps23.PhiValues = make([]JITValueDesc, 1)
					d24 = d20
					ps23.PhiValues[0] = d24
					if ps23.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps23)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d25 := ps.PhiValues[0]
							ctx.EnsureDesc(&d25)
							ctx.EmitStoreToStack(d25, int32(bbs[2].PhiBase)+int32(0))
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d26 = args[0]
					d26.ID = 0
					ctx.EnsureDesc(&d26)
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
						panic("jit: generic call arg expects 2-word value (PrettyPrint arg0)")
					}
					d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocRegTriple || d27.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d26)
					ctx.SyncDesc(&d27)
					ctx.SyncDesc(&d1)
					d28 = ctx.EmitGoCallScalar(GoFuncAddr(PrettyPrint), []JITValueDesc{d26, d27, d1}, 2)
					ctx.BindReg(d28.Reg, &d28)
					ctx.BindReg(d28.Reg2, &d28)
					ctx.FreeDesc(&d26)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d28)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(NewString), []JITValueDesc{d28}, 2)
					ctx.EmitMovPairToResult(&d29, &result)
					result.Type = tagString
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps30 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps30)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
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
				keyCount := len(snapshot.variants) + len(snapshot.rejected)
				sz += uint(keyCount) * uint(unsafe.Sizeof(procSpecializationKey{}))
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
