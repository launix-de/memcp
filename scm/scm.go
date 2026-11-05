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
				// define allocates named bindings on demand; numbered scopes need none.
				beginEnv := &Env{VarsNumbered: en.VarsNumbered, Outer: en, Nodefine: false}
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
				beginEnv := &Env{VarsNumbered: varsNumbered, Outer: en, Nodefine: false}
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
			if proc := procedure.Proc(); proc != nil && proc.nativeCode() != 0 {
				args := make([]Scmer, len(operands))
				for i, operand := range operands {
					args[i] = Eval(operand, en)
				}
				return proc.callJIT(args)
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
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagCString, tagBString, tagLZ4, tagRegex, tagVector, tagFastDict, tagParser, tagAny, tagBSON:
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
		value, _ := en.Lookup(sym)
		return value
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

func installProcCallCaptures(env *Env, proc *Proc) {
	captures := jitProcCaptures(proc)
	if len(captures) == 0 {
		return
	}
	end := proc.compiled().CaptureBase + len(captures)
	if proc.compiled().CaptureBase < 0 || end > len(env.VarsNumbered) {
		panic("apply: JIT captures exceed procedure frame")
	}
	copy(env.VarsNumbered[proc.compiled().CaptureBase:end], captures)
}

func prepareProcCall(p *Proc, operands []Scmer, caller *Env) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := p.snapshot()
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
	installProcCallCaptures(env, p)
	return env, proc.Body
}

func prepareProcCallWithArgs(p *Proc, args []Scmer) (*Env, Scmer) {
	if p == nil {
		panic("apply: nil procedure")
	}
	proc := p.snapshot()
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
		// The variadic binding is a first-class list which can escape through
		// the result or a closure. Callers may lend reusable or JIT-stack
		// argument frames, so the interpreter must own this list's backing.
		argsList := NewSlice(append([]Scmer(nil), args...))
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
	installProcCallCaptures(env, p)
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

// scmerCallable reports whether procedure can be invoked with Apply - a lambda,
// a native func, a closure, or a compiled JIT entry (assoc-list slices are
// callable too but are not what a "replacement function" argument means).
func scmerCallable(procedure Scmer) bool {
	switch procedure.GetTag() {
	case tagFunc, tagFuncEnv, tagClosure, tagProc, tagJIT:
		return true
	}
	return false
}

func ApplyEx(procedure Scmer, args []Scmer, en *Env) (value Scmer) {
	// Native funcs
	switch procedure.GetTag() {
	case tagFuncEnv:
		return procedure.FuncEnv()(en, args...)
	case tagFunc:
		if declaration := DeclarationForValue(procedure); declaration != nil && declaration.RetainsCallArgs {
			args = append([]Scmer(nil), args...)
		}
		return procedure.Func()(args...)
	case tagClosure:
		fn := *(*func(uint32, ...Scmer) Scmer)(unsafe.Pointer(procedure.ptr))
		id := uint32(auxVal(procedure.aux))
		return fn(id, args...)
	// Lambdas
	case tagProc:
		if proc := procedure.Proc(); proc != nil && proc.nativeCode() != 0 {
			return proc.callJIT(args)
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
	// JITCode is first so a *Proc is also a Go-compatible funcval context.
	// Native calls receive this same pointer in the closure-context register.
	// Shared Go readers use nativeCode; publication stores it atomically last.
	JITCode      uintptr
	Params, Body Scmer
	En           *Env
	// Compiled is an optional native implementation of this procedure. The
	// original body remains attached so storage scan callbacks can later be
	// specialized and recompiled against concrete column/storage types.
	// Shared readers use compiled. A private capture plan may precede code;
	// the executable entry is immutable and atomically published before JITCode.
	Compiled *JITEntryPoint
	// OptimizerMeta belongs to this concrete procedure identity. Exact Proc
	// copies, including the JIT's source/compiled pair, share it. Copies which
	// change Body or En must allocate a new identity so cached specializations
	// can never retain code or captures from a different Proc.
	// Keep optimizer-only fields after the runtime/JIT-facing Proc layout.
	OptimizerMeta *ProcOptimizerMeta
	NumVars       int
	NumberedOnly  bool
	// jitCompiling breaks mutually recursive on-demand compilation without
	// serializing unrelated procedures.
	jitCompiling uint32
	// Shared code identity and GC ownership are independent of invocation values.
	rebindPlan    *lambdaPlan
	codeOwner     *Proc
	lambdaFactory bool
}

// ProcJIT describes the native closure ABI. Context is a flexible array member:
// concrete allocations replace its zero length with the required capture count.
type ProcJIT struct {
	Proc    Proc
	Context [0]Scmer
}

// nativeCode is the execution publication: a nonzero value is stored only
// after immutable entry metadata and all transitive stack maps are ready.
func (p *Proc) nativeCode() uintptr {
	return atomic.LoadUintptr(&p.JITCode)
}

func (p *Proc) compiled() *JITEntryPoint {
	entry := (*JITEntryPoint)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&p.Compiled))))
	if entry != nil && entry.CodePtr != nil && p.nativeCode() == 0 {
		return nil
	}
	return entry
}

// snapshot copies the immutable source declaration and atomically observes its
// optional native implementation. The per-identity compilation CAS is never
// copied: callers either own the copy or establish a new compilation owner.
func (p *Proc) snapshot() Proc {
	entry := p.compiled()
	var code uintptr
	if entry != nil && entry.CodePtr != nil {
		code = p.nativeCode()
	}
	return Proc{
		JITCode: code, Params: p.Params, Body: p.Body, En: p.En,
		Compiled: entry, OptimizerMeta: p.OptimizerMeta,
		rebindPlan: (*lambdaPlan)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&p.rebindPlan)))), codeOwner: p.codeOwner, lambdaFactory: p.lambdaFactory,
		NumVars: p.NumVars, NumberedOnly: p.NumberedOnly,
	}
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

// closedCaptureLiteral embeds a captured value as data in executable code.
// Lists/dictionaries are also callable and symbols normally resolve bindings;
// quoting preserves the value an ordinary lexical read would have returned.
func closedCaptureLiteral(value Scmer) Scmer {
	if value.IsSlice() || value.IsFastDict() || value.IsSymbol() || value.IsSourceInfo() {
		return NewSlice([]Scmer{NewSymbol("quote"), value})
	}
	return value
}

// CloseProcedure snapshots explicit captures of a procedure without retaining
// its request-local environment. The optimizer represents a capture from the
// procedure's creation frame as (outer depth expr). Resolve only those forms in the
// current and nested bodies. References to invocation-local scopes stay intact;
// references beyond them snapshot the source environment before it is released.
func CloseProcedure(value Scmer) Scmer {
	if value.GetTag() != tagProc {
		return value
	}
	proc := value.Proc().snapshot()
	hadJITCaptures := proc.nativeCode() != 0 && proc.compiled() != nil && proc.compiled().CaptureCount != 0
	if hadJITCaptures {
		captures := append([]Scmer(nil), jitProcCaptures(value.Proc())...)
		for i, symbol := range proc.compiled().CaptureSymbols {
			if symbol == Symbol(lambdaScopeSymbol) && i < len(captures) {
				captures[i] = NewNil()
			}
		}
		proc.Body = closeJITProcedureCaptures(proc.Body, proc.compiled().CaptureBase, captures, 0)
	}
	if proc.En == nil || proc.En == &Globalenv {
		if !hadJITCaptures {
			return value
		}
		proc.JITCode = 0
		proc.Compiled = nil
		return NewProcStruct(proc)
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
	proc.JITCode = 0
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
	return closeProcedureScopeCaptures(expression, callFrame, []map[Symbol]struct{}{bound})
}

// depth counts frames created when this body runs. Only references beyond
// those future frames capture the source environment; inner references must
// keep binding to each invocation's own parameters, begin and match scopes.
func closeProcedureScopeCaptures(expression Scmer, callFrame *Env, scopes []map[Symbol]struct{}) Scmer {
	return transformProcedureCaptures(expression, callFrame, scopes, func(key Scmer) Scmer {
		if scmerIsSymbol(key, lambdaScopeSymbol) {
			return NewNil()
		}
		return closedCaptureLiteral(Eval(key, callFrame))
	})
}

// transformProcedureCaptures owns lexical capture boundaries for both closing
// a value and separating reusable code from invocation bindings.
func transformProcedureCaptures(expression Scmer, callFrame *Env, scopes []map[Symbol]struct{}, capture func(Scmer) Scmer) Scmer {
	depth := len(scopes) - 1
	bound := scopes[depth]
	if expression.IsSourceInfo() {
		source := *expression.SourceInfo()
		closed := transformProcedureCaptures(source.value, callFrame, scopes, capture)
		if closed == source.value {
			return expression
		}
		source.value = closed
		return NewSourceInfo(source)
	}
	if expression.IsSymbol() {
		symbol := expression.Symbol()
		if _, local := bound[symbol]; !local {
			if binding := callFrame.Outer.FindRead(symbol); binding != nil && binding != &Globalenv {
				if _, ok := binding.Vars[symbol]; ok {
					return capture(expression)
				}
			}
		}
		return expression
	}
	if !expression.IsSlice() {
		return expression
	}
	items := lambdaSyntaxOperands(expression.Slice())
	if len(items) == 0 {
		return expression
	}
	head, _ := scmerSymbol(items[0])
	if head == "quote" {
		return expression
	}
	if head == "outer" && len(items) == 3 {
		outerDepth, valid := outerDepthLiteral(items[1])
		if !valid {
			return expression
		}
		if outerDepth > int64(depth) {
			outer := NewSlice([]Scmer{items[0], NewInt(outerDepth - int64(depth)), items[2]})
			return capture(outer)
		}
		operand := transformProcedureCaptures(items[2], callFrame, scopes[:len(scopes)-int(outerDepth)], capture)
		if operand == items[2] {
			return expression
		}
		return NewSlice([]Scmer{items[0], items[1], operand})
	}
	// Most operands already refer to invocation-local or global bindings.
	// Copy only when closing a capture actually changes an operand.
	var closed []Scmer
	replace := func(index int, value Scmer) {
		if value == items[index] {
			return
		}
		if closed == nil {
			closed = append([]Scmer(nil), items...)
		}
		closed[index] = value
	}
	switch head {
	case "lambda":
		if len(items) >= 3 {
			locals := procedureWireBindings(items[1], items[2], bound)
			replace(2, transformProcedureCaptures(items[2], callFrame, append(scopes, locals), capture))
		}
	case "begin", "begin_mut":
		first := 1
		if head == "begin_mut" && len(items) > 1 {
			replace(1, transformProcedureCaptures(items[1], callFrame, scopes, capture))
			first = 2
		}
		for i := first; i < len(items); i++ {
			replace(i, transformProcedureCaptures(items[i], callFrame, append(scopes, bound), capture))
		}
	case "match", "match_mut":
		if len(items) > 1 {
			replace(1, transformProcedureCaptures(items[1], callFrame, scopes, capture))
		}
		for i := 3; i < len(items); i += 2 {
			replace(i-1, transformProcedurePattern(items[i-1], callFrame, scopes, capture))
			locals := procedureWireBindings(NewSlice(nil), items[i], bound)
			procedurePatternBindings(items[i-1], locals)
			replace(i, transformProcedureCaptures(items[i], callFrame, append(scopes, locals), capture))
		}
		if len(items)%2 == 1 {
			replace(len(items)-1, transformProcedureCaptures(items[len(items)-1], callFrame, scopes, capture))
		}
	default:
		for i, item := range items {
			replace(i, transformProcedureCaptures(item, callFrame, scopes, capture))
		}
	}
	if closed == nil {
		return expression
	}
	if head == "lambda" {
		// Closing a capture must not discard the optimizer's binding proof.
		// Rebuild it for the new body once, rather than walking the callback
		// every time a row instantiates this lambda.
		closed = prepareLambdaBindings(closed)
	}
	return NewSlice(closed)
}

// Numbered locals include the captures appended when the JIT closes a lambda.
// Wide projections can exceed 255 slots even when each SQL row is smaller.
type NthLocalVar uint32 // equals to (var i)

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
	bound, exists := e.Lookup(s)
	if !exists || bound.GetTag() != tagProc {
		return tiZero, false
	}
	proc := bound.Proc()
	if proc.OptimizerMeta == nil || !proc.OptimizerMeta.HasReturn {
		return tiZero, false
	}
	return proc.OptimizerMeta.Return, true
}

// Lookup reads a named binding in one walk. A present nil still shadows outer
// bindings; callers needing the owning frame use FindRead instead. Keep this
// traversal shared rather than duplicating its loop in the large evaluator.
//
//go:noinline
func (e *Env) Lookup(s Symbol) (Scmer, bool) {
	for e != nil {
		if value, ok := e.Vars[s]; ok {
			return value, true
		}
		e = e.Outer
	}
	return NewNil(), false
}

// Keep frame traversal out of line, as with its former recursive implementation.
//
//go:noinline
func (e *Env) FindRead(s Symbol) *Env {
	for {
		if _, ok := e.Vars[s]; ok || e.Outer == nil {
			return e
		}
		e = e.Outer
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

			// Host metadata for language-level frontends and diagnostics.
			Symbol("runtime_os"): NewString(runtime.GOOS),

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
		Name:       "quote",
		SyntaxKind: SyntaxQuote,

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
		Name:         "eval",
		SyntaxKind:   SyntaxEval,
		evalDispatch: specialFormEval,

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
				declaration := declarations["size"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(ComputeSize), []JITValueDesc{d0}, 1)
				d1.NoHeapPointer = true
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
			JITInlineCost: 6,
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
		Name: "close_procedure",

		Fn: func(a ...Scmer) Scmer {
			return CloseProcedure(a[0])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "closes a procedure over explicit captures without retaining its creation environment",
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "procedure", Description: "procedure to close"},
			},
			Return: &TypeDescriptor{Kind: "func"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["close_procedure"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				d1 := d0
				_ = d1
				ctx.StabilizeDescForControlFlow(&d1)
				inlineHome2 := d1
				inlineHome2.ID = 0
				phiBase3 := ctx.AllocStack(int32(32))
				d4 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				_ = d4
				d5 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				_ = d5
				lbl0 := ctx.ReserveLabel()
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_1_1 := int32(-1)
				_ = bbpos_1_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_1_2 := int32(-1)
				_ = bbpos_1_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_1_3 := int32(-1)
				_ = bbpos_1_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_1_4 := int32(-1)
				_ = bbpos_1_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_1_5 := int32(-1)
				_ = bbpos_1_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_1_6 := int32(-1)
				_ = bbpos_1_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_1_7 := int32(-1)
				_ = bbpos_1_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_1_8 := int32(-1)
				_ = bbpos_1_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_1_9 := int32(-1)
				_ = bbpos_1_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_1_10 := int32(-1)
				_ = bbpos_1_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_1_11 := int32(-1)
				_ = bbpos_1_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_1_12 := int32(-1)
				_ = bbpos_1_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_1_13 := int32(-1)
				_ = bbpos_1_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_1_14 := int32(-1)
				_ = bbpos_1_14
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_1_15 := int32(-1)
				_ = bbpos_1_15
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_1_16 := int32(-1)
				_ = bbpos_1_16
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_1_17 := int32(-1)
				_ = bbpos_1_17
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_1_18 := int32(-1)
				_ = bbpos_1_18
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_1_19 := int32(-1)
				_ = bbpos_1_19
				lbl20 := ctx.ReserveLabel()
				_ = lbl20
				bbpos_1_20 := int32(-1)
				_ = bbpos_1_20
				lbl21 := ctx.ReserveLabel()
				_ = lbl21
				bbpos_1_21 := int32(-1)
				_ = bbpos_1_21
				lbl22 := ctx.ReserveLabel()
				_ = lbl22
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d6 := d1
				d6.ID = 0
				d7 := ctx.EmitGetTagDesc(&d6, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d7)
				var d8 JITValueDesc
				if d7.Loc == LocImm {
					d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d7.Imm.Int()) != uint64(0xa))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d7.Reg, 10)
					d8 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
					ctx.BindReg(r0, &d8)
				}
				ctx.FreeDesc(&d7)
				ctx.ReclaimUntrackedRegs()
				d9 := d8
				ctx.SyncDesc(&d9)
				if d9.Loc != LocImm && d9.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl23 := ctx.ReserveLabel()
				lbl24 := ctx.ReserveLabel()
				if d9.Loc == LocImm {
					if d9.Imm.Bool() {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitJump(d9.Condition, lbl23)
					ctx.EmitJmp(lbl24)
					ctx.FreeDesc(&d8)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl3)
				}
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d10 := ctx.EmitGoCallScalar(jitGoABI_79b61201b9e57f9d, nil, 1)
				ctx.BindReg(d10.Reg, &d10)
				ctx.StabilizeDescForControlFlow(&d10)
				inlineHome11 := d10
				inlineHome11.ID = 0
				ctx.ReclaimUntrackedRegs()
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d1)
				d12 := ctx.EmitGoCallScalar(jitGoABI_c3658a69a02c135c, []JITValueDesc{d1}, 1)
				d12.NoHeapPointer = false
				ctx.BindReg(d12.Reg, &d12)
				ctx.ReclaimUntrackedRegs()
				if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d12)
				d13 := ctx.EmitGoCallScalar(jitGoABI_bd15746ee4fe9617, []JITValueDesc{d12}, 1)
				d13.NoHeapPointer = false
				ctx.BindReg(d13.Reg, &d13)
				ctx.FreeDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.EmitGoCallVoid(jitGoABI_14cfa92b2b87ef7c, []JITValueDesc{d10, d13})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d14 := d10
				_ = d14
				ctx.StabilizeDescForControlFlow(&d10)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl25 := ctx.ReserveLabel()
				_ = lbl25
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl25)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d15 := ctx.EmitGoCallScalar(jitGoABI_751bf088537a5f21, []JITValueDesc{d14}, 1)
				d15.NoHeapPointer = true
				ctx.BindReg(d15.Reg, &d15)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d15)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d15)
				var d16 JITValueDesc
				if d15.Loc == LocImm {
					d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d15.Imm.Int()) != uint64(0x0))}
				} else {
					r1 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					d16 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
					ctx.BindReg(r1, &d16)
				}
				ctx.FreeDesc(&d15)
				ctx.ReclaimUntrackedRegs()
				d17 := d16
				ctx.SyncDesc(&d17)
				if d17.Loc != LocImm && d17.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl26 := ctx.ReserveLabel()
				lbl27 := ctx.ReserveLabel()
				if d17.Loc == LocImm {
					if d17.Imm.Bool() {
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl6)
					} else {
						ctx.MarkLabel(lbl27)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d17.Condition, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.FreeDesc(&d16)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl27)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase3)+int32(0))
					ctx.EmitJmp(lbl5)
				}
				bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d4 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d18 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase3) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d18)
				ctx.ReclaimUntrackedRegs()
				d19 := d18
				ctx.EnsureDesc(&d19)
				if d19.Loc != LocImm && d19.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl28 := ctx.ReserveLabel()
				lbl29 := ctx.ReserveLabel()
				if d19.Loc == LocImm {
					if d19.Imm.Bool() {
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl7)
					} else {
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl28)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl8)
				}
				bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d20 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 40
					r2 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r2, fieldAddr)
					d20 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d20)
				} else {
					off := int32(40)
					baseReg := d10.Reg
					r3 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r3, baseReg, off)
					d20 = JITValueDesc{Loc: LocReg, Reg: r3}
					ctx.BindReg(r3, &d20)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d20)
				var d21 JITValueDesc
				if d20.Loc == LocImm {
					d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocReg && d20.Loc != LocRegPair && d20.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r4 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondEqual}
					ctx.BindReg(r4, &d21)
				}
				ctx.FreeDesc(&d20)
				ctx.ReclaimUntrackedRegs()
				d22 := d21
				ctx.SyncDesc(&d22)
				if d22.Loc != LocImm && d22.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl30 := ctx.ReserveLabel()
				lbl31 := ctx.ReserveLabel()
				if d22.Loc == LocImm {
					if d22.Imm.Bool() {
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl31)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitJump(d22.Condition, lbl30)
					ctx.EmitJmp(lbl31)
					ctx.FreeDesc(&d21)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl11)
				}
				bbpos_1_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d23 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 40
					r5 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r5, fieldAddr)
					d23 = JITValueDesc{Loc: LocReg, Reg: r5}
					ctx.BindReg(r5, &d23)
				} else {
					off := int32(40)
					baseReg := d10.Reg
					r6 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r6, baseReg, off)
					d23 = JITValueDesc{Loc: LocReg, Reg: r6}
					ctx.BindReg(r6, &d23)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d23)
				d25 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				ctx.EnsureDescsTogether(&d23, &d25)
				var d24 JITValueDesc
				if d23.Loc == LocImm && d25.Loc == LocImm {
					d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() == d25.Imm.Int())}
				} else if d25.Loc == LocImm {
					r7 := ctx.AllocReg()
					if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d23.Reg, int32(d25.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d25.Imm.Int()))
						ctx.EmitCmpInt64(d23.Reg, ctx.ScratchReg)
					}
					d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondEqual}
					ctx.BindReg(r7, &d24)
				} else if d23.Loc == LocImm {
					r8 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d23.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d25.Reg)
					d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondEqual}
					ctx.BindReg(r8, &d24)
				} else {
					r9 := ctx.AllocReg()
					ctx.EmitCmpInt64(d23.Reg, d25.Reg)
					d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondEqual}
					ctx.BindReg(r9, &d24)
				}
				ctx.FreeDesc(&d23)
				ctx.ReclaimUntrackedRegs()
				d26 := d24
				ctx.SyncDesc(&d26)
				if d26.Loc != LocImm && d26.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl32 := ctx.ReserveLabel()
				lbl33 := ctx.ReserveLabel()
				if d26.Loc == LocImm {
					if d26.Imm.Bool() {
						ctx.MarkLabel(lbl32)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitJump(d26.Condition, lbl32)
					ctx.EmitJmp(lbl33)
					ctx.FreeDesc(&d24)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl10)
				}
				bbpos_1_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d27 := ctx.EmitGoCallScalar(jitGoABI_3f45ec6607233365, nil, 1)
				ctx.BindReg(d27.Reg, &d27)
				ctx.StabilizeDescForControlFlow(&d27)
				inlineHome28 := d27
				inlineHome28.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d29 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 40
					r10 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r10, fieldAddr)
					d29 = JITValueDesc{Loc: LocReg, Reg: r10}
					ctx.BindReg(r10, &d29)
				} else {
					off := int32(40)
					baseReg := d10.Reg
					r11 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r11, baseReg, off)
					d29 = JITValueDesc{Loc: LocReg, Reg: r11}
					ctx.BindReg(r11, &d29)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d29)
				ctx.EnsureDesc(&d27)
				ctx.EnsureDesc(&d29)
				ctx.EmitGoCallVoid(jitGoABI_18fd4c44c7a31da6, []JITValueDesc{d27, d29})
				ctx.FreeDesc(&d29)
				ctx.ReclaimUntrackedRegs()
				d30 := ctx.EmitGoCallScalar(jitGoABI_9bd0890fbc57aa9f, []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.StabilizeDescForControlFlow(&d30)
				inlineHome31 := d30
				inlineHome31.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d32 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 8
					r12 := ctx.AllocReg()
					r13 := ctx.AllocRegExcept(r12)
					ctx.EmitMovRegMem64(r12, fieldAddr)
					ctx.EmitMovRegMem64(r13, fieldAddr+8)
					d32 = JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
					ctx.BindReg(r12, &d32)
					ctx.BindReg(r13, &d32)
				} else {
					off := int32(8)
					baseReg := d10.Reg
					r14 := ctx.AllocRegExcept(baseReg)
					r15 := ctx.AllocRegExcept(baseReg, r14)
					ctx.EmitMovRegMem(r14, baseReg, off)
					ctx.EmitMovRegMem(r15, baseReg, off+8)
					d32 = JITValueDesc{Loc: LocRegPair, Reg: r14, Reg2: r15}
					ctx.BindReg(r14, &d32)
					ctx.BindReg(r15, &d32)
				}
				ctx.ReclaimUntrackedRegs()
				d34 := d32
				d34.ID = 0
				d33 := ctx.EmitTagEqualsBorrowed(&d34, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d32)
				ctx.ReclaimUntrackedRegs()
				d35 := d33
				ctx.EnsureDesc(&d35)
				if d35.Loc != LocImm && d35.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl34 := ctx.ReserveLabel()
				lbl35 := ctx.ReserveLabel()
				if d35.Loc == LocImm {
					if d35.Imm.Bool() {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl14)
					} else {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl16)
					}
				} else {
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl34)
					ctx.EmitJmp(lbl35)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl16)
				}
				ctx.FreeDesc(&d33)
				bbpos_1_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d36 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 8
					r16 := ctx.AllocReg()
					r17 := ctx.AllocRegExcept(r16)
					ctx.EmitMovRegMem64(r16, fieldAddr)
					ctx.EmitMovRegMem64(r17, fieldAddr+8)
					d36 = JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
					ctx.BindReg(r16, &d36)
					ctx.BindReg(r17, &d36)
				} else {
					off := int32(8)
					baseReg := d10.Reg
					r18 := ctx.AllocRegExcept(baseReg)
					r19 := ctx.AllocRegExcept(baseReg, r18)
					ctx.EmitMovRegMem(r18, baseReg, off)
					ctx.EmitMovRegMem(r19, baseReg, off+8)
					d36 = JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
					ctx.BindReg(r18, &d36)
					ctx.BindReg(r19, &d36)
				}
				ctx.ReclaimUntrackedRegs()
				d36 = JITPrepareScmerGoArg(ctx, d36)
				ctx.SyncDesc(&d36)
				d37 := ctx.EmitGoCallScalar(jitGoABI_c94eb298f761efc9, []JITValueDesc{d36}, 1)
				d37.NoHeapPointer = true
				ctx.EmitAndRegImm32(d37.Reg, 1)
				d37.Type = tagBool
				ctx.BindReg(d37.Reg, &d37)
				ctx.FreeDesc(&d36)
				ctx.ReclaimUntrackedRegs()
				d38 := d37
				ctx.EnsureDesc(&d38)
				if d38.Loc != LocImm && d38.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl36 := ctx.ReserveLabel()
				lbl37 := ctx.ReserveLabel()
				if d38.Loc == LocImm {
					if d38.Imm.Bool() {
						ctx.MarkLabel(lbl36)
						ctx.EmitJmp(lbl20)
					} else {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitCmpRegImm32(d38.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl36)
					ctx.EmitJmp(lbl37)
					ctx.MarkLabel(lbl36)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl15)
				}
				ctx.FreeDesc(&d37)
				bbpos_1_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d39 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 24
					r20 := ctx.AllocReg()
					r21 := ctx.AllocRegExcept(r20)
					ctx.EmitMovRegMem64(r20, fieldAddr)
					ctx.EmitMovRegMem64(r21, fieldAddr+8)
					d39 = JITValueDesc{Loc: LocRegPair, Reg: r20, Reg2: r21}
					ctx.BindReg(r20, &d39)
					ctx.BindReg(r21, &d39)
				} else {
					off := int32(24)
					baseReg := d10.Reg
					r22 := ctx.AllocRegExcept(baseReg)
					r23 := ctx.AllocRegExcept(baseReg, r22)
					ctx.EmitMovRegMem(r22, baseReg, off)
					ctx.EmitMovRegMem(r23, baseReg, off+8)
					d39 = JITValueDesc{Loc: LocRegPair, Reg: r22, Reg2: r23}
					ctx.BindReg(r22, &d39)
					ctx.BindReg(r23, &d39)
				}
				ctx.ReclaimUntrackedRegs()
				d39 = JITPrepareScmerGoArg(ctx, d39)
				if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocRegTriple || d30.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d39)
				ctx.SyncDesc(&d30)
				ctx.EmitGoCallVoid(GoFuncAddr(collectProcedureBindings), []JITValueDesc{d39, d30})
				ctx.FreeDesc(&d39)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d40 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 24
					r24 := ctx.AllocReg()
					r25 := ctx.AllocRegExcept(r24)
					ctx.EmitMovRegMem64(r24, fieldAddr)
					ctx.EmitMovRegMem64(r25, fieldAddr+8)
					d40 = JITValueDesc{Loc: LocRegPair, Reg: r24, Reg2: r25}
					ctx.BindReg(r24, &d40)
					ctx.BindReg(r25, &d40)
				} else {
					off := int32(24)
					baseReg := d10.Reg
					r26 := ctx.AllocRegExcept(baseReg)
					r27 := ctx.AllocRegExcept(baseReg, r26)
					ctx.EmitMovRegMem(r26, baseReg, off)
					ctx.EmitMovRegMem(r27, baseReg, off+8)
					d40 = JITValueDesc{Loc: LocRegPair, Reg: r26, Reg2: r27}
					ctx.BindReg(r26, &d40)
					ctx.BindReg(r27, &d40)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d40)
				ctx.EnsureDesc(&d27)
				ctx.EnsureDesc(&d30)
				d41 := d40
				_ = d41
				d42 := d27
				_ = d42
				d43 := d30
				_ = d43
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				lbl38 := ctx.ReserveLabel()
				_ = lbl38
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl38)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d44 := ctx.EmitGoCallScalar(jitGoABI_ed9f7fe688ad8b78, nil, 1)
				ctx.ReclaimUntrackedRegs()
				d45 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d45
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				ctx.EmitGoCallVoid(jitGoABI_0651917b305ad187, []JITValueDesc{d44, d45, d43})
				ctx.ReclaimUntrackedRegs()
				sliceResults46 := JITEmitGoCallResults(ctx, jitGoABI_575a0c3de7e0d5ed, []JITValueDesc{d44})
				d47 := sliceResults46[0]
				_ = d47
				ctx.ReclaimUntrackedRegs()
				d41 = JITPrepareScmerGoArg(ctx, d41)
				if d42.Loc == LocRegPair || d42.Loc == LocStackPair || d42.Loc == LocRegTriple || d42.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d47 = JITPrepareGoSliceArg(ctx, d47)
				if d47.Loc != LocRegTriple && d47.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (closeProcedureScopeCaptures arg2)")
				}
				ctx.SyncDesc(&d41)
				ctx.SyncDesc(&d42)
				ctx.SyncDesc(&d47)
				d48 := ctx.EmitGoCallScalar(GoFuncAddr(closeProcedureScopeCaptures), []JITValueDesc{d41, d42, d47}, 2)
				d48.NoHeapPointer = false
				ctx.BindReg(d48.Reg, &d48)
				ctx.BindReg(d48.Reg2, &d48)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d48)
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d48)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d48)
				ctx.EmitGoCallVoid(jitGoABI_c8bd3fbb876619d4, []JITValueDesc{d10, d48})
				ctx.FreeDesc(&d48)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d49 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d49)
				ctx.EmitGoCallVoid(jitGoABI_b7afa8aa7afec23f, []JITValueDesc{d10, d49})
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d50 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d50
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d50)
				ctx.EmitGoCallVoid(jitGoABI_b20afda8856e61ef, []JITValueDesc{d10, d50})
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d51 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d51
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d51)
				ctx.EmitGoCallVoid(jitGoABI_830596eef656ff9c, []JITValueDesc{d10, d51})
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d52 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 56
					r28 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r28, fieldAddr)
					d52 = JITValueDesc{Loc: LocReg, Reg: r28}
					ctx.BindReg(r28, &d52)
				} else {
					off := int32(56)
					baseReg := d10.Reg
					r29 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r29, baseReg, off)
					d52 = JITValueDesc{Loc: LocReg, Reg: r29}
					ctx.BindReg(r29, &d52)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d52)
				var d53 JITValueDesc
				if d52.Loc == LocImm {
					d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d52.Imm.IsNil() != true)}
				} else {
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocReg && d52.Loc != LocRegPair && d52.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r30 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d52.Reg, 0)
					d53 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondNotEqual}
					ctx.BindReg(r30, &d53)
				}
				ctx.FreeDesc(&d52)
				ctx.ReclaimUntrackedRegs()
				d54 := d53
				ctx.SyncDesc(&d54)
				if d54.Loc != LocImm && d54.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d54.Loc == LocImm {
					if d54.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl21)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl22)
					}
				} else {
					ctx.EmitJump(d54.Condition, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.FreeDesc(&d53)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl22)
				}
				bbpos_1_21 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl22)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d55 := d10
				_ = d55
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				lbl41 := ctx.ReserveLabel()
				_ = lbl41
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl41)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d56 := ctx.EmitGoCallScalar(jitGoABI_79b61201b9e57f9d, nil, 1)
				ctx.BindReg(d56.Reg, &d56)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d55)
				ctx.EmitGoCallVoid(jitGoABI_14cfa92b2b87ef7c, []JITValueDesc{d56, d55})
				ctx.ReclaimUntrackedRegs()
				if d56.Loc == LocRegPair || d56.Loc == LocStackPair || d56.Loc == LocRegTriple || d56.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d56)
				d57 := ctx.EmitGoCallScalar(jitGoABI_1b8412732af78364, []JITValueDesc{d56}, 2)
				d57.NoHeapPointer = false
				ctx.BindReg(d57.Reg, &d57)
				ctx.BindReg(d57.Reg2, &d57)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d57)
				ctx.ReclaimUntrackedRegs()
				r31 := ctx.AllocReg()
				r32 := ctx.AllocRegExcept(r31)
				d58 := JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d58)
				ctx.BindReg(r32, &d58)
				ctx.EmitMovPairToResult(&d57, &d58)
				ctx.EmitJmp(lbl0)
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d59 := JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d59)
				ctx.BindReg(r32, &d59)
				ctx.EmitMovPairToResult(&d1, &d59)
				ctx.EmitJmp(lbl0)
				bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d10)
				d60 := ctx.EmitGoCallScalar(jitGoABI_1e8379006790c0c3, []JITValueDesc{d10}, 1)
				d60.NoHeapPointer = false
				ctx.BindReg(d60.Reg, &d60)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d60)
				var d61 JITValueDesc
				if d60.Loc == LocImm {
					d61 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d60.Imm.IsNil() != true)}
				} else {
					ctx.EnsureDesc(&d60)
					if d60.Loc != LocReg && d60.Loc != LocRegPair && d60.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r33 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d60.Reg, 0)
					d61 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondNotEqual}
					ctx.BindReg(r33, &d61)
				}
				ctx.FreeDesc(&d60)
				ctx.ReclaimUntrackedRegs()
				d62 := d61
				ctx.SyncDesc(&d62)
				if d62.Loc != LocImm && d62.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl42 := ctx.ReserveLabel()
				lbl43 := ctx.ReserveLabel()
				if d62.Loc == LocImm {
					if d62.Imm.Bool() {
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl43)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase3)+int32(0))
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d62.Condition, lbl42)
					ctx.EmitJmp(lbl43)
					ctx.FreeDesc(&d61)
					ctx.MarkLabel(lbl42)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl43)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase3)+int32(0))
					ctx.EmitJmp(lbl5)
				}
				bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d1)
				d63 := ctx.EmitGoCallScalar(jitGoABI_c3658a69a02c135c, []JITValueDesc{d1}, 1)
				d63.NoHeapPointer = false
				ctx.BindReg(d63.Reg, &d63)
				ctx.ReclaimUntrackedRegs()
				if d63.Loc == LocRegPair || d63.Loc == LocStackPair || d63.Loc == LocRegTriple || d63.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d63)
				d64 := ctx.EmitGoCallScalar(jitGoABI_93f1f83c9e505d62, []JITValueDesc{d63}, 3)
				d64.NoHeapPointer = false
				ctx.BindReg(d64.Reg, &d64)
				ctx.BindReg(d64.Reg2, &d64)
				ctx.BindReg(d64.Reg3, &d64)
				ctx.FreeDesc(&d63)
				ctx.ReclaimUntrackedRegs()
				callResults65 := JITEmitGoCallResults(ctx, jitGoABI_caeccb0b1551362d, []JITValueDesc{d64})
				d66 := callResults65[0]
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d67 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 24
					r34 := ctx.AllocReg()
					r35 := ctx.AllocRegExcept(r34)
					ctx.EmitMovRegMem64(r34, fieldAddr)
					ctx.EmitMovRegMem64(r35, fieldAddr+8)
					d67 = JITValueDesc{Loc: LocRegPair, Reg: r34, Reg2: r35}
					ctx.BindReg(r34, &d67)
					ctx.BindReg(r35, &d67)
				} else {
					off := int32(24)
					baseReg := d10.Reg
					r36 := ctx.AllocRegExcept(baseReg)
					r37 := ctx.AllocRegExcept(baseReg, r36)
					ctx.EmitMovRegMem(r36, baseReg, off)
					ctx.EmitMovRegMem(r37, baseReg, off+8)
					d67 = JITValueDesc{Loc: LocRegPair, Reg: r36, Reg2: r37}
					ctx.BindReg(r36, &d67)
					ctx.BindReg(r37, &d67)
				}
				ctx.ReclaimUntrackedRegs()
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d10)
				d68 := ctx.EmitGoCallScalar(jitGoABI_1e8379006790c0c3, []JITValueDesc{d10}, 1)
				d68.NoHeapPointer = false
				ctx.BindReg(d68.Reg, &d68)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d69 JITValueDesc
				ctx.EnsureDesc(&d68)
				if d68.Loc == LocImm {
					fieldAddr := uintptr(d68.Imm.Int()) + 288
					r38 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r38, fieldAddr)
					d69 = JITValueDesc{Loc: LocReg, Reg: r38}
					ctx.BindReg(r38, &d69)
				} else {
					off := int32(288)
					baseReg := d68.Reg
					r39 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r39, baseReg, off)
					d69 = JITValueDesc{Loc: LocReg, Reg: r39}
					ctx.BindReg(r39, &d69)
				}
				ctx.FreeDesc(&d68)
				ctx.ReclaimUntrackedRegs()
				d67 = JITPrepareScmerGoArg(ctx, d67)
				if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocRegTriple || d69.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d66 = JITPrepareGoSliceArg(ctx, d66)
				if d66.Loc != LocRegTriple && d66.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (closeJITProcedureCaptures arg2)")
				}
				d70 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d70
				if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d67)
				ctx.SyncDesc(&d69)
				ctx.SyncDesc(&d66)
				ctx.SyncDesc(&d70)
				d71 := ctx.EmitGoCallScalar(jitGoABI_b3734367ba267d31, []JITValueDesc{d67, d69, d66, d70}, 2)
				d71.NoHeapPointer = false
				ctx.BindReg(d71.Reg, &d71)
				ctx.BindReg(d71.Reg2, &d71)
				ctx.FreeDesc(&d70)
				ctx.FreeDesc(&d67)
				ctx.FreeDesc(&d69)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d71)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d71)
				ctx.EmitGoCallVoid(jitGoABI_c8bd3fbb876619d4, []JITValueDesc{d10, d71})
				ctx.FreeDesc(&d71)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl8)
				bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d72 := d18
				ctx.EnsureDesc(&d72)
				if d72.Loc != LocImm && d72.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl44 := ctx.ReserveLabel()
				lbl45 := ctx.ReserveLabel()
				if d72.Loc == LocImm {
					if d72.Imm.Bool() {
						ctx.MarkLabel(lbl44)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl45)
						ctx.EmitJmp(lbl12)
					}
				} else {
					ctx.EmitCmpRegImm32(d72.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl44)
					ctx.EmitJmp(lbl45)
					ctx.MarkLabel(lbl44)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl12)
				}
				bbpos_1_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d73 := JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d73)
				ctx.BindReg(r32, &d73)
				ctx.EmitMovPairToResult(&d1, &d73)
				ctx.EmitJmp(lbl0)
				bbpos_1_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d74 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 8
					r40 := ctx.AllocReg()
					r41 := ctx.AllocRegExcept(r40)
					ctx.EmitMovRegMem64(r40, fieldAddr)
					ctx.EmitMovRegMem64(r41, fieldAddr+8)
					d74 = JITValueDesc{Loc: LocRegPair, Reg: r40, Reg2: r41}
					ctx.BindReg(r40, &d74)
					ctx.BindReg(r41, &d74)
				} else {
					off := int32(8)
					baseReg := d10.Reg
					r42 := ctx.AllocRegExcept(baseReg)
					r43 := ctx.AllocRegExcept(baseReg, r42)
					ctx.EmitMovRegMem(r42, baseReg, off)
					ctx.EmitMovRegMem(r43, baseReg, off+8)
					d74 = JITValueDesc{Loc: LocRegPair, Reg: r42, Reg2: r43}
					ctx.BindReg(r42, &d74)
					ctx.BindReg(r43, &d74)
				}
				ctx.ReclaimUntrackedRegs()
				d75 := jitKnownSliceHeader(ctx, &d74)
				ctx.StabilizeDescForControlFlow(&d75)
				inlineHome76 := d75
				inlineHome76.ID = 0
				ctx.FreeDesc(&d74)
				ctx.ReclaimUntrackedRegs()
				var d77 JITValueDesc
				if d75.SliceSizeKnown {
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.KnownSliceLen))}
				} else if d75.Loc == LocImm {
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.StackOff))}
				} else if d75.Loc == LocStackTriple {
					d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d75.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocRegPair || d75.Loc == LocRegTriple {
						d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg2, ID: 0}
					} else if d75.Loc == LocReg {
						d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d77
				ctx.StabilizeDescForControlFlow(&d77)
				inlineHome78 := d77
				inlineHome78.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase3)+int32(16))
				bbpos_1_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d5 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d79 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d79)
				ctx.EnsureDesc(&d79)
				var d80 JITValueDesc
				if d79.Loc == LocImm {
					d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d79.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d79.Reg)
					ctx.EmitMovRegReg(scratch, d79.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d80)
				}
				if d80.Loc == LocReg && d79.Loc == LocReg && d80.Reg == d79.Reg {
					ctx.TransferReg(d79.Reg)
					d79.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d80)
				inlineHome81 := d80
				inlineHome81.ID = 0
				ctx.FreeDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d80)
				ctx.EnsureDesc(&d77)
				ctx.EnsureDescsTogether(&d80, &d77)
				var d82 JITValueDesc
				if d80.Loc == LocImm && d77.Loc == LocImm {
					d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d80.Imm.Int() < d77.Imm.Int())}
				} else if d77.Loc == LocImm {
					r44 := ctx.AllocRegExcept(d80.Reg)
					if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d80.Reg, int32(d77.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d77.Imm.Int()))
						ctx.EmitCmpInt64(d80.Reg, ctx.ScratchReg)
					}
					d82 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r44, Condition: CondSignedLess}
					ctx.BindReg(r44, &d82)
				} else if d80.Loc == LocImm {
					r45 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d80.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d77.Reg)
					d82 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r45, Condition: CondSignedLess}
					ctx.BindReg(r45, &d82)
				} else {
					r46 := ctx.AllocRegExcept(d80.Reg)
					ctx.EmitCmpInt64(d80.Reg, d77.Reg)
					d82 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r46, Condition: CondSignedLess}
					ctx.BindReg(r46, &d82)
				}
				ctx.ReclaimUntrackedRegs()
				d83 := d82
				ctx.SyncDesc(&d83)
				if d83.Loc != LocImm && d83.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl46 := ctx.ReserveLabel()
				lbl47 := ctx.ReserveLabel()
				if d83.Loc == LocImm {
					if d83.Imm.Bool() {
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl18)
					} else {
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitJump(d83.Condition, lbl46)
					ctx.EmitJmp(lbl47)
					ctx.FreeDesc(&d82)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl15)
				}
				bbpos_1_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl20)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d84 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 8
					r47 := ctx.AllocReg()
					r48 := ctx.AllocRegExcept(r47)
					ctx.EmitMovRegMem64(r47, fieldAddr)
					ctx.EmitMovRegMem64(r48, fieldAddr+8)
					d84 = JITValueDesc{Loc: LocRegPair, Reg: r47, Reg2: r48}
					ctx.BindReg(r47, &d84)
					ctx.BindReg(r48, &d84)
				} else {
					off := int32(8)
					baseReg := d10.Reg
					r49 := ctx.AllocRegExcept(baseReg)
					r50 := ctx.AllocRegExcept(baseReg, r49)
					ctx.EmitMovRegMem(r49, baseReg, off)
					ctx.EmitMovRegMem(r50, baseReg, off+8)
					d84 = JITValueDesc{Loc: LocRegPair, Reg: r49, Reg2: r50}
					ctx.BindReg(r49, &d84)
					ctx.BindReg(r50, &d84)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d84)
				d85 := d84
				_ = d85
				ctx.StabilizeDescForControlFlow(&d85)
				inlineHome86 := d85
				inlineHome86.ID = 0
				inlineResultOff87 := ctx.AllocStack(int32(16))
				d88 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff87}
				lbl48 := ctx.ReserveLabel()
				bbpos_5_0 := int32(-1)
				_ = bbpos_5_0
				lbl49 := ctx.ReserveLabel()
				_ = lbl49
				bbpos_5_1 := int32(-1)
				_ = bbpos_5_1
				lbl50 := ctx.ReserveLabel()
				_ = lbl50
				bbpos_5_2 := int32(-1)
				_ = bbpos_5_2
				lbl51 := ctx.ReserveLabel()
				_ = lbl51
				bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl49)
				ctx.ResolveFixups()
				d85 = inlineHome86
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d89 := d85
				d89.ID = 0
				d90 := ctx.EmitGetTagDesc(&d89, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d90)
				var d91 JITValueDesc
				if d90.Loc == LocImm {
					d91 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d90.Imm.Int()) != uint64(0x2))}
				} else {
					r51 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d90.Reg, 2)
					d91 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r51, Condition: CondNotEqual}
					ctx.BindReg(r51, &d91)
				}
				ctx.FreeDesc(&d90)
				ctx.ReclaimUntrackedRegs()
				d92 := d91
				ctx.SyncDesc(&d92)
				if d92.Loc != LocImm && d92.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl52 := ctx.ReserveLabel()
				lbl53 := ctx.ReserveLabel()
				if d92.Loc == LocImm {
					if d92.Imm.Bool() {
						ctx.MarkLabel(lbl52)
						ctx.EmitJmp(lbl50)
					} else {
						ctx.MarkLabel(lbl53)
						ctx.EmitJmp(lbl51)
					}
				} else {
					ctx.EmitJump(d92.Condition, lbl52)
					ctx.EmitJmp(lbl53)
					ctx.FreeDesc(&d91)
					ctx.MarkLabel(lbl52)
					ctx.EmitJmp(lbl50)
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl51)
				}
				bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl51)
				ctx.ResolveFixups()
				d85 = inlineHome86
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d94 := d85
				ctx.SyncDesc(&d94)
				if d94.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d94.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d94.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d94 = tmpScalar
				}
				d94 = JITPrepareScmerGoArg(ctx, d94)
				if d94.Loc != LocRegPair && d94.Loc != LocStackPair && d94.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d93 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d94}, 2)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d93)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d93)
				ctx.EmitCopyDescWords(&d88, &d93, 2)
				ctx.EmitJmp(lbl48)
				bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl50)
				ctx.ResolveFixups()
				d85 = inlineHome86
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				ctx.MarkLabel(lbl48)
				ctx.FreeDesc(&d84)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d88)
				d95 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d95
				ctx.EmitGoCallVoid(jitGoABI_446dee5074537592, []JITValueDesc{d30, d88, d95})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl15)
				bbpos_1_20 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl21)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d96 := ctx.EmitGoCallScalar(jitGoABI_cd25371cee7f2008, nil, 1)
				ctx.BindReg(d96.Reg, &d96)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d97 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 56
					r52 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r52, fieldAddr)
					d97 = JITValueDesc{Loc: LocReg, Reg: r52}
					ctx.BindReg(r52, &d97)
				} else {
					off := int32(56)
					baseReg := d10.Reg
					r53 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r53, baseReg, off)
					d97 = JITValueDesc{Loc: LocReg, Reg: r53}
					ctx.BindReg(r53, &d97)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d98 JITValueDesc
				ctx.EnsureDesc(&d97)
				if d97.Loc == LocImm {
					fieldAddr := uintptr(d97.Imm.Int()) + 0
					r54 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r54, fieldAddr)
					d98 = JITValueDesc{Loc: LocReg, Reg: r54}
					ctx.BindReg(r54, &d98)
				} else {
					off := int32(0)
					baseReg := d97.Reg
					r55 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r55, baseReg, off)
					d98 = JITValueDesc{Loc: LocReg, Reg: r55}
					ctx.BindReg(r55, &d98)
				}
				ctx.FreeDesc(&d97)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d99 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 56
					r56 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r56, fieldAddr)
					d99 = JITValueDesc{Loc: LocReg, Reg: r56}
					ctx.BindReg(r56, &d99)
				} else {
					off := int32(56)
					baseReg := d10.Reg
					r57 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r57, baseReg, off)
					d99 = JITValueDesc{Loc: LocReg, Reg: r57}
					ctx.BindReg(r57, &d99)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d100 JITValueDesc
				ctx.EnsureDesc(&d99)
				if d99.Loc == LocImm {
					fieldAddr := uintptr(d99.Imm.Int()) + 24
					r58 := ctx.AllocReg()
					ctx.EmitMovRegMem8(r58, fieldAddr)
					d100 = JITValueDesc{Loc: LocReg, Reg: r58}
					ctx.BindReg(r58, &d100)
				} else {
					off := int32(24)
					baseReg := d99.Reg
					r59 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMemB(r59, baseReg, off)
					d100 = JITValueDesc{Loc: LocReg, Reg: r59}
					ctx.BindReg(r59, &d100)
				}
				ctx.FreeDesc(&d99)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d101 JITValueDesc
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocImm {
					fieldAddr := uintptr(d10.Imm.Int()) + 56
					r60 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r60, fieldAddr)
					d101 = JITValueDesc{Loc: LocReg, Reg: r60}
					ctx.BindReg(r60, &d101)
				} else {
					off := int32(56)
					baseReg := d10.Reg
					r61 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r61, baseReg, off)
					d101 = JITValueDesc{Loc: LocReg, Reg: r61}
					ctx.BindReg(r61, &d101)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d102 JITValueDesc
				ctx.EnsureDesc(&d101)
				if d101.Loc == LocImm {
					fieldAddr := uintptr(d101.Imm.Int()) + 25
					r62 := ctx.AllocReg()
					ctx.EmitMovRegMem8(r62, fieldAddr)
					d102 = JITValueDesc{Loc: LocReg, Reg: r62}
					ctx.BindReg(r62, &d102)
				} else {
					off := int32(25)
					baseReg := d101.Reg
					r63 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMemB(r63, baseReg, off)
					d102 = JITValueDesc{Loc: LocReg, Reg: r63}
					ctx.BindReg(r63, &d102)
				}
				ctx.FreeDesc(&d101)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d98)
				ctx.EnsureDesc(&d96)
				ctx.EnsureDesc(&d98)
				ctx.EmitGoCallVoid(jitGoABI_158fc888362fa531, []JITValueDesc{d96, d98})
				ctx.FreeDesc(&d98)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d100)
				ctx.EnsureDesc(&d96)
				ctx.EnsureDesc(&d100)
				ctx.EmitGoCallVoid(jitGoABI_f471bfc54f82491c, []JITValueDesc{d96, d100})
				ctx.FreeDesc(&d100)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d102)
				ctx.EnsureDesc(&d96)
				ctx.EnsureDesc(&d102)
				ctx.EmitGoCallVoid(jitGoABI_2aaf5b10787be869, []JITValueDesc{d96, d102})
				ctx.FreeDesc(&d102)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d96)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d96)
				ctx.EmitGoCallVoid(jitGoABI_c7963fc7464639ca, []JITValueDesc{d10, d96})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl22)
				bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d10)
				d103 := ctx.EmitGoCallScalar(jitGoABI_1e8379006790c0c3, []JITValueDesc{d10}, 1)
				d103.NoHeapPointer = false
				ctx.BindReg(d103.Reg, &d103)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d104 JITValueDesc
				ctx.EnsureDesc(&d103)
				if d103.Loc == LocImm {
					fieldAddr := uintptr(d103.Imm.Int()) + 296
					r64 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r64, fieldAddr)
					d104 = JITValueDesc{Loc: LocReg, Reg: r64}
					ctx.BindReg(r64, &d104)
				} else {
					off := int32(296)
					baseReg := d103.Reg
					r65 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r65, baseReg, off)
					d104 = JITValueDesc{Loc: LocReg, Reg: r65}
					ctx.BindReg(r65, &d104)
				}
				ctx.FreeDesc(&d103)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d104)
				var d105 JITValueDesc
				if d104.Loc == LocImm {
					d105 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d104.Imm.Int() != 0)}
				} else {
					ctx.EmitCmpRegImm32(d104.Reg, 0)
					r66 := ctx.AllocRegExcept(d104.Reg)
					ctx.EmitSetcc(r66, CondNotEqual)
					d105 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r66}
					ctx.BindReg(r66, &d105)
				}
				ctx.EnsureDesc(&d105)
				ctx.EmitStoreToStack(d105, int32(phiBase3)+int32(0))
				ctx.StabilizeDescForControlFlow(&d105)
				inlineHome106 := d105
				inlineHome106.ID = 0
				ctx.FreeDesc(&d104)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl5)
				bbpos_1_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d105 = inlineHome106
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d107 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d107
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d107)
				ctx.EmitGoCallVoid(jitGoABI_b20afda8856e61ef, []JITValueDesc{d10, d107})
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d108 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d108
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d108)
				ctx.EmitGoCallVoid(jitGoABI_830596eef656ff9c, []JITValueDesc{d10, d108})
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d10)
				d109 := d10
				_ = d109
				bbpos_6_0 := int32(-1)
				_ = bbpos_6_0
				lbl54 := ctx.ReserveLabel()
				_ = lbl54
				bbpos_6_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl54)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d110 := ctx.EmitGoCallScalar(jitGoABI_79b61201b9e57f9d, nil, 1)
				ctx.BindReg(d110.Reg, &d110)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d109)
				ctx.EmitGoCallVoid(jitGoABI_14cfa92b2b87ef7c, []JITValueDesc{d110, d109})
				ctx.ReclaimUntrackedRegs()
				if d110.Loc == LocRegPair || d110.Loc == LocStackPair || d110.Loc == LocRegTriple || d110.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d110)
				d111 := ctx.EmitGoCallScalar(jitGoABI_1b8412732af78364, []JITValueDesc{d110}, 2)
				d111.NoHeapPointer = false
				ctx.BindReg(d111.Reg, &d111)
				ctx.BindReg(d111.Reg2, &d111)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d111)
				ctx.ReclaimUntrackedRegs()
				d112 := JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d112)
				ctx.BindReg(r32, &d112)
				ctx.EmitMovPairToResult(&d111, &d112)
				ctx.EmitJmp(lbl0)
				bbpos_1_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d105 = inlineHome106
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d80)
				ctx.ReclaimUntrackedRegs()
				d114 := ctx.EmitSliceElementAddress(&d75, &d80, 16)
				ctx.EnsureDesc(&d114)
				r67 := ctx.AllocRegExcept(d114.Reg)
				ctx.EmitMovRegMem(r67, d114.Reg, 8)
				ctx.EmitMovRegMem(d114.Reg, d114.Reg, 0)
				d113 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d114.Reg, Reg2: r67}
				ctx.BindReg(d114.Reg, &d113)
				ctx.BindReg(r67, &d113)
				ctx.StabilizeDescForControlFlow(&d113)
				inlineHome115 := d113
				inlineHome115.ID = 0
				ctx.ReclaimUntrackedRegs()
				d113 = JITPrepareScmerGoArg(ctx, d113)
				ctx.SyncDesc(&d113)
				d116 := ctx.EmitGoCallScalar(jitGoABI_c94eb298f761efc9, []JITValueDesc{d113}, 1)
				d116.NoHeapPointer = true
				ctx.EmitAndRegImm32(d116.Reg, 1)
				d116.Type = tagBool
				ctx.BindReg(d116.Reg, &d116)
				ctx.ReclaimUntrackedRegs()
				d117 := d116
				ctx.EnsureDesc(&d117)
				if d117.Loc != LocImm && d117.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl55 := ctx.ReserveLabel()
				lbl56 := ctx.ReserveLabel()
				if d117.Loc == LocImm {
					if d117.Imm.Bool() {
						ctx.MarkLabel(lbl55)
						ctx.EmitJmp(lbl19)
					} else {
						ctx.MarkLabel(lbl56)
						ctx.SyncDesc(&d80)
						if d80.Loc == LocReg || d80.Loc == LocFPReg {
							ctx.ProtectReg(d80.Reg)
						} else if d80.Loc == LocRegPair {
							ctx.ProtectReg(d80.Reg)
							ctx.ProtectReg(d80.Reg2)
						}
						d118 := d80
						if d118.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d118)
						ctx.EmitStoreToStack(d118, int32(phiBase3)+int32(16))
						if d80.Loc == LocReg || d80.Loc == LocFPReg {
							ctx.UnprotectReg(d80.Reg)
						} else if d80.Loc == LocRegPair {
							ctx.UnprotectReg(d80.Reg)
							ctx.UnprotectReg(d80.Reg2)
						}
						ctx.EmitJmp(lbl17)
					}
				} else {
					ctx.EmitCmpRegImm32(d117.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl55)
					ctx.EmitJmp(lbl56)
					ctx.MarkLabel(lbl55)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl56)
					ctx.SyncDesc(&d80)
					if d80.Loc == LocReg || d80.Loc == LocFPReg {
						ctx.ProtectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.ProtectReg(d80.Reg)
						ctx.ProtectReg(d80.Reg2)
					}
					d119 := d80
					if d119.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d119)
					ctx.EmitStoreToStack(d119, int32(phiBase3)+int32(16))
					if d80.Loc == LocReg || d80.Loc == LocFPReg {
						ctx.UnprotectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.UnprotectReg(d80.Reg)
						ctx.UnprotectReg(d80.Reg2)
					}
					ctx.EmitJmp(lbl17)
				}
				ctx.FreeDesc(&d116)
				bbpos_1_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d1 = inlineHome2
				d10 = inlineHome11
				d27 = inlineHome28
				d30 = inlineHome31
				d75 = inlineHome76
				d77 = inlineHome78
				d80 = inlineHome81
				d105 = inlineHome106
				d113 = inlineHome115
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase3) + int32(0)}
				d79 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase3) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d113)
				d120 := d113
				_ = d120
				ctx.StabilizeDescForControlFlow(&d120)
				inlineHome121 := d120
				inlineHome121.ID = 0
				ctx.StabilizeDescForControlFlow(&d113)
				inlineResultOff122 := ctx.AllocStack(int32(16))
				d123 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inlineResultOff122}
				lbl57 := ctx.ReserveLabel()
				bbpos_7_0 := int32(-1)
				_ = bbpos_7_0
				lbl58 := ctx.ReserveLabel()
				_ = lbl58
				bbpos_7_1 := int32(-1)
				_ = bbpos_7_1
				lbl59 := ctx.ReserveLabel()
				_ = lbl59
				bbpos_7_2 := int32(-1)
				_ = bbpos_7_2
				lbl60 := ctx.ReserveLabel()
				_ = lbl60
				bbpos_7_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl58)
				ctx.ResolveFixups()
				d120 = inlineHome121
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d124 := d120
				d124.ID = 0
				d125 := ctx.EmitGetTagDesc(&d124, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d125)
				var d126 JITValueDesc
				if d125.Loc == LocImm {
					d126 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d125.Imm.Int()) != uint64(0x2))}
				} else {
					r68 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d125.Reg, 2)
					d126 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r68, Condition: CondNotEqual}
					ctx.BindReg(r68, &d126)
				}
				ctx.FreeDesc(&d125)
				ctx.ReclaimUntrackedRegs()
				d127 := d126
				ctx.SyncDesc(&d127)
				if d127.Loc != LocImm && d127.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl61 := ctx.ReserveLabel()
				lbl62 := ctx.ReserveLabel()
				if d127.Loc == LocImm {
					if d127.Imm.Bool() {
						ctx.MarkLabel(lbl61)
						ctx.EmitJmp(lbl59)
					} else {
						ctx.MarkLabel(lbl62)
						ctx.EmitJmp(lbl60)
					}
				} else {
					ctx.EmitJump(d127.Condition, lbl61)
					ctx.EmitJmp(lbl62)
					ctx.FreeDesc(&d126)
					ctx.MarkLabel(lbl61)
					ctx.EmitJmp(lbl59)
					ctx.MarkLabel(lbl62)
					ctx.EmitJmp(lbl60)
				}
				bbpos_7_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl60)
				ctx.ResolveFixups()
				d120 = inlineHome121
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d129 := d120
				ctx.SyncDesc(&d129)
				if d129.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d129.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d129.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d129 = tmpScalar
				}
				d129 = JITPrepareScmerGoArg(ctx, d129)
				if d129.Loc != LocRegPair && d129.Loc != LocStackPair && d129.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d128 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d129}, 2)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d128)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d128)
				ctx.EmitCopyDescWords(&d123, &d128, 2)
				ctx.EmitJmp(lbl57)
				bbpos_7_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl59)
				ctx.ResolveFixups()
				d120 = inlineHome121
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				ctx.MarkLabel(lbl57)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d123)
				d130 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d130
				ctx.EmitGoCallVoid(jitGoABI_446dee5074537592, []JITValueDesc{d30, d123, d130})
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d80)
				if d80.Loc == LocReg || d80.Loc == LocFPReg {
					ctx.ProtectReg(d80.Reg)
				} else if d80.Loc == LocRegPair {
					ctx.ProtectReg(d80.Reg)
					ctx.ProtectReg(d80.Reg2)
				}
				d131 := d80
				if d131.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d131)
				ctx.EmitStoreToStack(d131, int32(phiBase3)+int32(16))
				if d80.Loc == LocReg || d80.Loc == LocFPReg {
					ctx.UnprotectReg(d80.Reg)
				} else if d80.Loc == LocRegPair {
					ctx.UnprotectReg(d80.Reg)
					ctx.UnprotectReg(d80.Reg2)
				}
				ctx.EmitJmp(lbl17)
				ctx.MarkLabel(lbl0)
				d132 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d132)
				ctx.BindReg(r32, &d132)
				ctx.BindReg(r31, &d132)
				ctx.BindReg(r32, &d132)
				ctx.FreeDesc(&d0)
				if d132.Loc == LocImm {
					if result.Loc == LocAny {
						return d132
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d132)
				if d132.Loc == LocRegPair || d132.Loc == LocStackPair || d132.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d132, &result)
					result.Type = d132.Type
				} else {
					switch d132.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d132)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d132)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d132)
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
			JITInlineCost: 165,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "clone_optimizer_expression",

		Fn: func(a ...Scmer) Scmer {
			return CloneOptimizerExpression(a[0])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "copy an AST before transferring a separate copy to the optimizer",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "expression"},
			},
			Return:         &TypeDescriptor{Kind: "any"},
			HasSideEffects: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["clone_optimizer_expression"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(jitGoABI_e32fdcf449cd74a7, []JITValueDesc{d0}, 2)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.FreeDesc(&d0)
				if d1.Loc == LocImm {
					if result.Loc == LocAny {
						return d1
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d1)
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d1, &result)
					result.Type = d1.Type
				} else {
					switch d1.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d1)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d1)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d1)
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
			JITInlineCost: 4,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "tryfold",
		Fn: func(a ...Scmer) Scmer {
			return TryFold(a[0], &Globalenv)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "return the constant value of raw Scheme code, or raise if it cannot be folded",
			Params: []*TypeDescriptor{{Kind: "any", Label: "code"}},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["tryfold"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(jitGoABI_efe6a72bfb144025, []JITValueDesc{d0, d1}, 2)
				d2.NoHeapPointer = false
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.FreeDesc(&d0)
				if d2.Loc == LocImm {
					if result.Loc == LocAny {
						return d2
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d2)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d2, &result)
					result.Type = d2.Type
				} else {
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d2)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d2)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d2)
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
			JITInlineCost: 4,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "tree_node_count",
		Fn: func(a ...Scmer) Scmer {
			return NewInt(int64(optimizerNodeCount(a[0])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "count list containers and scalar leaves by occurrence in an immutable value tree",
			Params: []*TypeDescriptor{{Kind: "any", Label: "tree", NoEscape: true}},
			Return: &TypeDescriptor{Kind: "int"}, Const: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["tree_node_count"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(jitGoABI_45b2e9aa8d87590a, []JITValueDesc{d0}, 1)
				d1.NoHeapPointer = true
				ctx.BindReg(d1.Reg, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d1.Loc == LocImm {
					ctx.EmitMakeInt(result, d1)
				} else {
					ctx.EmitMakeInt(result, d1)
					ctx.FreeReg(d1.Reg)
				}
				result.Type = tagInt
				return result
				return result
			},
			JITInlineCost: 6,
		},
	})
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
				declaration := declarations["optimize"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
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
				var d21 JITValueDesc
				_ = d21
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d2
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 2)
						d3 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl4 := ctx.ReserveLabel()
					ctx.EmitJump(d4.Condition, lbl2)
					ctx.EmitJmp(lbl4)
					ctx.FreeDesc(&d3)
					snap5 := d1
					snap6 := d2
					snap7 := d3
					snap8 := d4
					alloc9 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc9)
					d1 = snap5
					d2 = snap6
					d3 = snap7
					d4 = snap8
					if !bbs[2].Rendered {
						snap10 := d1
						snap11 := d2
						snap12 := d3
						snap13 := d4
						alloc14 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc14)
						d1 = snap10
						d2 = snap11
						d3 = snap12
						d4 = snap13
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d15 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r2, &d15)
					d16 = args[1]
					d16.ID = 0
					ctx.SyncDesc(&d16)
					ctx.FreeDesc(&d16)
					d17 = ctx.EmitGoCallScalar(jitGoABI_428f6123a6539c5d, []JITValueDesc{d16}, 1)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d16)
					ctx.SyncDesc(&d17)
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.ProtectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.ProtectReg(d17.Reg)
						ctx.ProtectReg(d17.Reg2)
					}
					d18 = d17
					if d18.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(bbs[2].PhiBase)+int32(0))
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.UnprotectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.UnprotectReg(d17.Reg)
						ctx.UnprotectReg(d17.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d17)
					d19 = args[0]
					d19.ID = 0
					d19 = JITPrepareScmerGoArg(ctx, d19)
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d20.Loc == LocRegPair || d20.Loc == LocStackPair || d20.Loc == LocRegTriple || d20.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d19)
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d1)
					d21 = ctx.EmitGoCallScalar(jitGoABI_4c692bc4c57a241a, []JITValueDesc{d19, d20, d1}, 2)
					d21.NoHeapPointer = false
					ctx.BindReg(d21.Reg, &d21)
					ctx.BindReg(d21.Reg2, &d21)
					ctx.FreeDesc(&d19)
					ctx.SyncDesc(&d21)
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d21, &result)
						result.Type = d21.Type
					} else {
						switch d21.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d21)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d21)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d21)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d21, &result)
							result.Type = d21.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  14,
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
		Name:         "if",
		evalDispatch: specialFormIf,

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
				declaration := declarations["error"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d25 JITValueDesc
				_ = d25
				var dynamicArgOff26 int32
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d2
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 1)
						d3 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d4.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d3)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap5 := d1
						snap6 := d2
						snap7 := d3
						snap8 := d4
						alloc9 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc9)
						d1 = snap5
						d2 = snap6
						d3 = snap7
						d4 = snap8
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d10 = ctx.EmitGoCallScalar(jitGoABI_b90d43f641e38b20, nil, 1)
					ctx.BindReg(d10.Reg, &d10)
					ctx.StabilizeDescForControlFlow(&d10)
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d12)
					}
					if d12.Loc == LocReg && d1.Loc == LocReg && d12.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDescsTogether(&d12, &d11)
					if d12.Loc == LocImm && d11.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < d11.Imm.Int())}
					} else if d11.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d12.Reg)
						if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d12.Reg, int32(d11.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d11.Imm.Int()))
							ctx.EmitCmpInt64(d12.Reg, ctx.ScratchReg)
						}
						d13 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d13)
					} else if d12.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d11.Reg)
						d13 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d13)
					} else {
						r3 := ctx.AllocRegExcept(d12.Reg)
						ctx.EmitCmpInt64(d12.Reg, d11.Reg)
						d13 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d13)
					}
					d14 = d13
					ctx.SyncDesc(&d14)
					if d14.Loc != LocImm && d14.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d14.Loc == LocImm {
						if d14.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d14.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d13)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap15 := d1
						snap16 := d2
						snap17 := d3
						snap18 := d4
						snap19 := d10
						snap20 := d11
						snap21 := d12
						snap22 := d13
						snap23 := d14
						alloc24 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc24)
						d1 = snap15
						d2 = snap16
						d3 = snap17
						d4 = snap18
						d10 = snap19
						d11 = snap20
						d12 = snap21
						d13 = snap22
						d14 = snap23
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						idx := int(d12.Imm.Int()) + 0
						if idx < 0 || idx >= len(args) {
							panic("jitgen: dynamic args index out of range")
						}
						d25 = args[idx]
						d25.ID = 0
					} else {
						ctx.EnsureDesc(&d12)
						dynamicArgOff26 = ctx.AllocStack(16)
						ctx.ProtectReg(d12.Reg)
						lbl7 := ctx.ReserveLabel()
						lbl8 := ctx.ReserveLabel()
						ctx.EmitCmpRegImm32(d12.Reg, int32(len(args)-0))
						ctx.EmitJump(CondUnsignedAboveOrEqual, lbl8)
						for i := 0; i < len(args); i++ {
							nextLbl := ctx.ReserveLabel()
							ctx.EmitCmpRegImm32(d12.Reg, int32(i-0))
							ctx.EmitJump(CondNotEqual, nextLbl)
							ai := args[i]
							ai.ID = 0
							ctx.EmitStoreScmerToStack(ai, int32(dynamicArgOff26))
							ctx.EmitJmp(lbl7)
							ctx.MarkLabel(nextLbl)
						}
						ctx.MarkLabel(lbl8)
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}, int32(dynamicArgOff26))
						ctx.MarkLabel(lbl7)
						ctx.UnprotectReg(d12.Reg)
						d25 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(dynamicArgOff26), Rooted: true}
					}
					d28 = d25
					ctx.SyncDesc(&d28)
					if d28.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d28.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d28.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d28 = tmpScalar
					}
					d28 = JITPrepareScmerGoArg(ctx, d28)
					if d28.Loc != LocRegPair && d28.Loc != LocStackPair && d28.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d27 = ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d28}, 2)
					if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d27.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d27.Imm)
						ptrWord, _ := d27.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d27.Imm.String())))
						d27 = tmpPair
					} else if d27.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d27.Type, Reg: ctx.AllocRegExcept(d27.Reg), Reg2: ctx.AllocRegExcept(d27.Reg)}
						switch d27.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d27)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d27)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d27)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d27)
						d27 = tmpPair
					}
					if d27.Loc != LocRegPair && d27.Loc != LocStackPair && d27.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value ((*strings.Builder).WriteString arg1)")
					}
					ctx.SyncDesc(&d10)
					ctx.SyncDesc(&d27)
					callResults29 := JITEmitGoCallResults(ctx, jitGoABI_e80a79328f388f6b, []JITValueDesc{d10, d27})
					d30 = callResults29[0]
					_ = d30
					d31 = callResults29[1]
					_ = d31
					ctx.SyncDesc(&d12)
					if d12.Loc == LocReg || d12.Loc == LocFPReg {
						ctx.ProtectReg(d12.Reg)
					} else if d12.Loc == LocRegPair {
						ctx.ProtectReg(d12.Reg)
						ctx.ProtectReg(d12.Reg2)
					}
					d32 = d12
					if d32.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d32)
					ctx.EmitStoreToStack(d32, int32(bbs[3].PhiBase)+int32(0))
					if d12.Loc == LocReg || d12.Loc == LocFPReg {
						ctx.UnprotectReg(d12.Reg)
					} else if d12.Loc == LocRegPair {
						ctx.UnprotectReg(d12.Reg)
						ctx.UnprotectReg(d12.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d10)
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["error"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  22,
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["try"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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
				declaration := declarations["apply"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
					d2 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d1}, 3)
				}
				if d2.Loc == LocRegTriple {
					ctx.BindReg(d2.Reg, &d2)
					ctx.BindReg(d2.Reg2, &d2)
					ctx.BindReg(d2.Reg3, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := d0
				_ = d3
				d4 := d2
				_ = d4
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d3 = JITPrepareScmerGoArg(ctx, d3)
				d4 = JITPrepareGoSliceArg(ctx, d4)
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
				d6.NoHeapPointer = false
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
				ctx.SyncDesc(&d6)
				if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocInputPair {
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
			JITInlineCost: 9,
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
				declaration := declarations["apply_assoc"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
					d2 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d1}, 3)
				}
				if d2.Loc == LocRegTriple {
					ctx.BindReg(d2.Reg, &d2)
					ctx.BindReg(d2.Reg2, &d2)
					ctx.BindReg(d2.Reg3, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := d0
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				inlineHome4 := d3
				inlineHome4.ID = 0
				d5 := d2
				_ = d5
				ctx.StabilizeDescForControlFlow(&d5)
				inlineHome6 := d5
				inlineHome6.ID = 0
				phiBase7 := ctx.AllocStack(int32(32))
				d8 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				_ = d8
				d9 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				_ = d9
				lbl0 := ctx.ReserveLabel()
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_1_1 := int32(-1)
				_ = bbpos_1_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_1_2 := int32(-1)
				_ = bbpos_1_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_1_3 := int32(-1)
				_ = bbpos_1_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_1_4 := int32(-1)
				_ = bbpos_1_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_1_5 := int32(-1)
				_ = bbpos_1_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_1_6 := int32(-1)
				_ = bbpos_1_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_1_7 := int32(-1)
				_ = bbpos_1_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_1_8 := int32(-1)
				_ = bbpos_1_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_1_9 := int32(-1)
				_ = bbpos_1_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_1_10 := int32(-1)
				_ = bbpos_1_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_1_11 := int32(-1)
				_ = bbpos_1_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_1_12 := int32(-1)
				_ = bbpos_1_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_1_13 := int32(-1)
				_ = bbpos_1_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d3 = JITPrepareScmerGoArg(ctx, d3)
				ctx.SyncDesc(&d3)
				d10 := ctx.EmitGoCallScalar(jitGoABI_5077bb76faa6a775, []JITValueDesc{d3}, 1)
				d10.NoHeapPointer = true
				ctx.EmitAndRegImm32(d10.Reg, 1)
				d10.Type = tagBool
				ctx.BindReg(d10.Reg, &d10)
				ctx.ReclaimUntrackedRegs()
				d11 := d10
				ctx.EnsureDesc(&d11)
				if d11.Loc != LocImm && d11.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl15 := ctx.ReserveLabel()
				lbl16 := ctx.ReserveLabel()
				if d11.Loc == LocImm {
					if d11.Imm.Bool() {
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitCmpRegImm32(d11.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl3)
				}
				ctx.FreeDesc(&d10)
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d3 = JITPrepareScmerGoArg(ctx, d3)
				ctx.SyncDesc(&d3)
				d12 := ctx.EmitGoCallScalar(jitGoABI_c3658a69a02c135c, []JITValueDesc{d3}, 1)
				d12.NoHeapPointer = false
				ctx.BindReg(d12.Reg, &d12)
				ctx.StabilizeDescForControlFlow(&d12)
				inlineHome13 := d12
				inlineHome13.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d14 JITValueDesc
				if d12.Loc == LocImm {
					d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocReg && d12.Loc != LocRegPair && d12.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r0 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
					ctx.BindReg(r0, &d14)
				}
				ctx.ReclaimUntrackedRegs()
				d15 := d14
				ctx.SyncDesc(&d15)
				if d15.Loc != LocImm && d15.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl17 := ctx.ReserveLabel()
				lbl18 := ctx.ReserveLabel()
				if d15.Loc == LocImm {
					if d15.Imm.Bool() {
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d15.Condition, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.FreeDesc(&d14)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl5)
				}
				bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d16 JITValueDesc
				ctx.EnsureDesc(&d12)
				if d12.Loc == LocImm {
					fieldAddr := uintptr(d12.Imm.Int()) + 8
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegMem64(r1, fieldAddr)
					ctx.EmitMovRegMem64(r2, fieldAddr+8)
					d16 = JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d16)
					ctx.BindReg(r2, &d16)
				} else {
					off := int32(8)
					baseReg := d12.Reg
					r3 := ctx.AllocRegExcept(baseReg)
					r4 := ctx.AllocRegExcept(baseReg, r3)
					ctx.EmitMovRegMem(r3, baseReg, off)
					ctx.EmitMovRegMem(r4, baseReg, off+8)
					d16 = JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
					ctx.BindReg(r3, &d16)
					ctx.BindReg(r4, &d16)
				}
				ctx.ReclaimUntrackedRegs()
				d17 := d16
				d17.ID = 0
				d18 := ctx.EmitGetTagDesc(&d17, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d16)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d18)
				var d19 JITValueDesc
				if d18.Loc == LocImm {
					d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(0x6))}
				} else {
					r5 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d18.Reg, 6)
					d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
					ctx.BindReg(r5, &d19)
				}
				ctx.FreeDesc(&d18)
				ctx.ReclaimUntrackedRegs()
				d20 := d19
				ctx.SyncDesc(&d20)
				if d20.Loc != LocImm && d20.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl19 := ctx.ReserveLabel()
				lbl20 := ctx.ReserveLabel()
				if d20.Loc == LocImm {
					if d20.Imm.Bool() {
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl6)
					} else {
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d20.Condition, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.FreeDesc(&d19)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl7)
				}
				bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d21 JITValueDesc
				ctx.EnsureDesc(&d12)
				if d12.Loc == LocImm {
					fieldAddr := uintptr(d12.Imm.Int()) + 8
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegMem64(r6, fieldAddr)
					ctx.EmitMovRegMem64(r7, fieldAddr+8)
					d21 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d21)
					ctx.BindReg(r7, &d21)
				} else {
					off := int32(8)
					baseReg := d12.Reg
					r8 := ctx.AllocRegExcept(baseReg)
					r9 := ctx.AllocRegExcept(baseReg, r8)
					ctx.EmitMovRegMem(r8, baseReg, off)
					ctx.EmitMovRegMem(r9, baseReg, off+8)
					d21 = JITValueDesc{Loc: LocRegPair, Reg: r8, Reg2: r9}
					ctx.BindReg(r8, &d21)
					ctx.BindReg(r9, &d21)
				}
				ctx.ReclaimUntrackedRegs()
				d22 := jitKnownSliceHeader(ctx, &d21)
				ctx.StabilizeDescForControlFlow(&d22)
				inlineHome23 := d22
				inlineHome23.ID = 0
				ctx.FreeDesc(&d21)
				ctx.ReclaimUntrackedRegs()
				var d24 JITValueDesc
				if d22.SliceSizeKnown {
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d22.KnownSliceLen))}
				} else if d22.Loc == LocImm {
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d22.StackOff))}
				} else if d22.Loc == LocStackTriple {
					d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d22.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocRegPair || d22.Loc == LocRegTriple {
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg2, ID: 0}
					} else if d22.Loc == LocReg {
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d24
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d24)
				ctx.EnsureDesc(&d24)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d24)
				ctx.EnsureDesc(&d24)
				callResults25 := JITEmitGoCallResults(ctx, jitGoABI_7ebaa2856648e378, []JITValueDesc{d24, d24})
				d26 := callResults25[0]
				d26.Type = tagSlice
				ctx.StabilizeDescForControlFlow(&d26)
				inlineHome27 := d26
				inlineHome27.ID = 0
				ctx.FreeDesc(&d24)
				ctx.ReclaimUntrackedRegs()
				var d28 JITValueDesc
				if d22.SliceSizeKnown {
					d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d22.KnownSliceLen))}
				} else if d22.Loc == LocImm {
					d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d22.StackOff))}
				} else if d22.Loc == LocStackTriple {
					d28 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d22.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocRegPair || d22.Loc == LocRegTriple {
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg2, ID: 0}
					} else if d22.Loc == LocReg {
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d28
				ctx.StabilizeDescForControlFlow(&d28)
				inlineHome29 := d28
				inlineHome29.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase7)+int32(0))
				bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d30 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d30)
				var d31 JITValueDesc
				if d30.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d30.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegReg(scratch, d30.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d31)
				}
				if d31.Loc == LocReg && d30.Loc == LocReg && d31.Reg == d30.Reg {
					ctx.TransferReg(d30.Reg)
					d30.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d31)
				inlineHome32 := d31
				inlineHome32.ID = 0
				ctx.FreeDesc(&d30)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d31)
				ctx.EnsureDesc(&d28)
				ctx.EnsureDescsTogether(&d31, &d28)
				var d33 JITValueDesc
				if d31.Loc == LocImm && d28.Loc == LocImm {
					d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() < d28.Imm.Int())}
				} else if d28.Loc == LocImm {
					r10 := ctx.AllocRegExcept(d31.Reg)
					if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d31.Reg, int32(d28.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d28.Imm.Int()))
						ctx.EmitCmpInt64(d31.Reg, ctx.ScratchReg)
					}
					d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
					ctx.BindReg(r10, &d33)
				} else if d31.Loc == LocImm {
					r11 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d31.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d28.Reg)
					d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
					ctx.BindReg(r11, &d33)
				} else {
					r12 := ctx.AllocRegExcept(d31.Reg)
					ctx.EmitCmpInt64(d31.Reg, d28.Reg)
					d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
					ctx.BindReg(r12, &d33)
				}
				ctx.ReclaimUntrackedRegs()
				d34 := d33
				ctx.SyncDesc(&d34)
				if d34.Loc != LocImm && d34.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl21 := ctx.ReserveLabel()
				lbl22 := ctx.ReserveLabel()
				if d34.Loc == LocImm {
					if d34.Imm.Bool() {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitJump(d34.Condition, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.FreeDesc(&d33)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl10)
				}
				bbpos_1_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d26)
				d35 := d3
				_ = d35
				d36 := d26
				_ = d36
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl23 := ctx.ReserveLabel()
				_ = lbl23
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl23)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d35 = JITPrepareScmerGoArg(ctx, d35)
				d36 = JITPrepareGoSliceArg(ctx, d36)
				if d36.Loc != LocRegTriple && d36.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
				}
				d37 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d37.Loc == LocRegPair || d37.Loc == LocStackPair || d37.Loc == LocRegTriple || d37.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d35)
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d37)
				d38 := ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d35, d36, d37}, 2)
				d38.NoHeapPointer = false
				ctx.BindReg(d38.Reg, &d38)
				ctx.BindReg(d38.Reg2, &d38)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d38)
				ctx.ReclaimUntrackedRegs()
				r13 := ctx.AllocReg()
				r14 := ctx.AllocRegExcept(r13)
				d39 := JITValueDesc{Loc: LocRegPair, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d39)
				ctx.BindReg(r14, &d39)
				ctx.EmitMovPairToResult(&d38, &d39)
				ctx.EmitJmp(lbl0)
				bbpos_1_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d31)
				ctx.ReclaimUntrackedRegs()
				d41 := ctx.EmitSliceElementAddress(&d22, &d31, 16)
				ctx.EnsureDesc(&d41)
				r15 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegMem(r15, d41.Reg, 8)
				ctx.EmitMovRegMem(d41.Reg, d41.Reg, 0)
				d40 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d41.Reg, Reg2: r15}
				ctx.BindReg(d41.Reg, &d40)
				ctx.BindReg(r15, &d40)
				ctx.ReclaimUntrackedRegs()
				d40 = JITPrepareScmerGoArg(ctx, d40)
				ctx.SyncDesc(&d40)
				d42 := ctx.EmitGoCallScalar(GoFuncAddr(mustSymbol), []JITValueDesc{d40}, 2)
				d42.NoHeapPointer = false
				ctx.BindReg(d42.Reg, &d42)
				ctx.BindReg(d42.Reg2, &d42)
				ctx.StabilizeDescForControlFlow(&d42)
				inlineHome43 := d42
				inlineHome43.ID = 0
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase7)+int32(16))
				bbpos_1_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d42 = inlineHome43
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d44 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase7) + int32(16)}
				ctx.StabilizeDescForControlFlow(&d44)
				ctx.ReclaimUntrackedRegs()
				var d45 JITValueDesc
				if d5.SliceSizeKnown {
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
				} else if d5.Loc == LocImm {
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
				} else if d5.Loc == LocStackTriple {
					d45 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
					} else if d5.Loc == LocReg {
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d45
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d45)
				ctx.EnsureDescsTogether(&d44, &d45)
				var d46 JITValueDesc
				if d44.Loc == LocImm && d45.Loc == LocImm {
					d46 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d44.Imm.Int() < d45.Imm.Int())}
				} else if d45.Loc == LocImm {
					r16 := ctx.AllocRegExcept(d44.Reg)
					if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d44.Reg, int32(d45.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d45.Imm.Int()))
						ctx.EmitCmpInt64(d44.Reg, ctx.ScratchReg)
					}
					d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
					ctx.BindReg(r16, &d46)
				} else if d44.Loc == LocImm {
					r17 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d44.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d45.Reg)
					d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
					ctx.BindReg(r17, &d46)
				} else {
					r18 := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitCmpInt64(d44.Reg, d45.Reg)
					d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
					ctx.BindReg(r18, &d46)
				}
				ctx.FreeDesc(&d45)
				ctx.ReclaimUntrackedRegs()
				d47 := d46
				ctx.SyncDesc(&d47)
				if d47.Loc != LocImm && d47.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl24 := ctx.ReserveLabel()
				lbl25 := ctx.ReserveLabel()
				if d47.Loc == LocImm {
					if d47.Imm.Bool() {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl25)
						ctx.SyncDesc(&d31)
						if d31.Loc == LocReg || d31.Loc == LocFPReg {
							ctx.ProtectReg(d31.Reg)
						} else if d31.Loc == LocRegPair {
							ctx.ProtectReg(d31.Reg)
							ctx.ProtectReg(d31.Reg2)
						}
						d48 := d31
						if d48.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d48)
						ctx.EmitStoreToStack(d48, int32(phiBase7)+int32(0))
						if d31.Loc == LocReg || d31.Loc == LocFPReg {
							ctx.UnprotectReg(d31.Reg)
						} else if d31.Loc == LocRegPair {
							ctx.UnprotectReg(d31.Reg)
							ctx.UnprotectReg(d31.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d47.Condition, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.FreeDesc(&d46)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl25)
					ctx.SyncDesc(&d31)
					if d31.Loc == LocReg || d31.Loc == LocFPReg {
						ctx.ProtectReg(d31.Reg)
					} else if d31.Loc == LocRegPair {
						ctx.ProtectReg(d31.Reg)
						ctx.ProtectReg(d31.Reg2)
					}
					d49 := d31
					if d49.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d49)
					ctx.EmitStoreToStack(d49, int32(phiBase7)+int32(0))
					if d31.Loc == LocReg || d31.Loc == LocFPReg {
						ctx.UnprotectReg(d31.Reg)
					} else if d31.Loc == LocRegPair {
						ctx.UnprotectReg(d31.Reg)
						ctx.UnprotectReg(d31.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_1_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d42 = inlineHome43
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.ReclaimUntrackedRegs()
				d51 := ctx.EmitSliceElementAddress(&d5, &d44, 16)
				ctx.EnsureDesc(&d51)
				r19 := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegMem(r19, d51.Reg, 8)
				ctx.EmitMovRegMem(d51.Reg, d51.Reg, 0)
				d50 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d51.Reg, Reg2: r19}
				ctx.BindReg(d51.Reg, &d50)
				ctx.BindReg(r19, &d50)
				ctx.ReclaimUntrackedRegs()
				d53 := d50
				ctx.SyncDesc(&d53)
				if d53.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d53.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d53.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d53 = tmpScalar
				}
				d53 = JITPrepareScmerGoArg(ctx, d53)
				if d53.Loc != LocRegPair && d53.Loc != LocStackPair && d53.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d52 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d53}, 2)
				ctx.FreeDesc(&d50)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d42)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d52)
				ctx.EnsureDesc(&d42)
				d54 := ctx.EmitGoCallScalar(jitGoABI_5130bc3ed31521d6, []JITValueDesc{d52, d42}, 1)
				ctx.EmitAndRegImm32(d54.Reg, 1)
				d54.Type = tagBool
				ctx.BindReg(d54.Reg, &d54)
				ctx.ReclaimUntrackedRegs()
				d55 := d54
				ctx.EnsureDesc(&d55)
				if d55.Loc != LocImm && d55.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl26 := ctx.ReserveLabel()
				lbl27 := ctx.ReserveLabel()
				if d55.Loc == LocImm {
					if d55.Imm.Bool() {
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl27)
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d55.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d54)
				bbpos_1_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d42 = inlineHome43
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d44)
				var d56 JITValueDesc
				if d44.Loc == LocImm {
					d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + 2)}
				} else {
					scratch := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitMovRegReg(scratch, d44.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 2)
					d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d56)
				}
				if d56.Loc == LocReg && d44.Loc == LocReg && d56.Reg == d44.Reg {
					ctx.TransferReg(d44.Reg)
					d44.Loc = LocNone
				}
				ctx.EnsureDesc(&d56)
				ctx.EmitStoreToStack(d56, int32(phiBase7)+int32(16))
				ctx.StabilizeDescForControlFlow(&d56)
				inlineHome57 := d56
				inlineHome57.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl11)
				bbpos_1_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d3 = inlineHome4
				d5 = inlineHome6
				d12 = inlineHome13
				d22 = inlineHome23
				d26 = inlineHome27
				d28 = inlineHome29
				d31 = inlineHome32
				d42 = inlineHome43
				d56 = inlineHome57
				d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				d44 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				ctx.EnsureDesc(&d44)
				var d58 JITValueDesc
				if d44.Loc == LocImm {
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitMovRegReg(scratch, d44.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d58)
				}
				if d58.Loc == LocReg && d44.Loc == LocReg && d58.Reg == d44.Reg {
					ctx.TransferReg(d44.Reg)
					d44.Loc = LocNone
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d58)
				ctx.ReclaimUntrackedRegs()
				d60 := ctx.EmitSliceElementAddress(&d5, &d58, 16)
				ctx.EnsureDesc(&d60)
				r20 := ctx.AllocRegExcept(d60.Reg)
				ctx.EmitMovRegMem(r20, d60.Reg, 8)
				ctx.EmitMovRegMem(d60.Reg, d60.Reg, 0)
				d59 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d60.Reg, Reg2: r20}
				ctx.BindReg(d60.Reg, &d59)
				ctx.BindReg(r20, &d59)
				ctx.FreeDesc(&d58)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d31)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d59)
				ctx.SyncDesc(&d26)
				ctx.SyncDesc(&d31)
				d61 := d26
				d61.ID = 0
				d62 := d31
				d62.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d61, &d62, &d59, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d31)
					d62 = d31
					d62.ID = 0
					ctx.EmitStoreScmerSliceElement(&d61, &d62, &d59, int32(16))
				}
				ctx.FreeDesc(&d62)
				ctx.FreeDesc(&d59)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl14)
				ctx.MarkLabel(lbl0)
				d63 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d63)
				ctx.BindReg(r14, &d63)
				ctx.BindReg(r13, &d63)
				ctx.BindReg(r14, &d63)
				ctx.FreeDesc(&d0)
				if d63.Loc == LocImm {
					if result.Loc == LocAny {
						return d63
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d63)
				if d63.Loc == LocRegPair || d63.Loc == LocStackPair || d63.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d63, &result)
					result.Type = d63.Type
				} else {
					switch d63.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d63)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d63)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d63)
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
			JITInlineCost: 60,
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
				declaration := declarations["symbol"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d1 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (NewSymbol arg0)")
				}
				ctx.SyncDesc(&d1)
				d3 := ctx.EmitGoCallScalar(jitGoABI_678e55d324700337, []JITValueDesc{d1}, 2)
				d3.NoHeapPointer = false
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
				ctx.SyncDesc(&d3)
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocInputPair {
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
			JITInlineCost: 5,
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
			cond := PrepareSerialProc(a[1])
			next := PrepareSerialProc(a[2])
			for ToBool(cond.Call(state)) {
				v := next.Call(state)
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
				declaration := declarations["for"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
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
				var stackArray66 int32
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var stackArray69 int32
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(24))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase0)+int32(0), 3)
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					stackArray2 = ctx.AllocStack(int32(0))
					_ = stackArray2
					d3 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d3
					d4 = args[0]
					d4.ID = 0
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d4}, 3)
					}
					if d5.Loc == LocRegTriple {
						ctx.BindReg(d5.Reg, &d5)
						ctx.BindReg(d5.Reg2, &d5)
						ctx.BindReg(d5.Reg3, &d5)
					}
					ctx.FreeDesc(&d4)
					callResults6 := JITEmitGoCallResults(ctx, jitGoABI_caeccb0b1551362d, []JITValueDesc{d5})
					d7 = callResults6[0]
					d8 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(bbs[3].PhiBase) + int32(0)}
					ctx.EmitCopyDescWords(&d8, &d7, 3)
					ctx.FreeDesc(&d7)
					d7 = d8
					ctx.StabilizeDescForControlFlow(&d7)
					d9 = args[1]
					d9.ID = 0
					if d9.Loc == LocLambdaTemplate {
						d10 = d9
					} else if d9.Loc == LocImm {
						optimizedCallback11 := jitPrepareCallback(d9.Imm)
						ctx.TrackImm(optimizedCallback11)
						d10 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback11, Rooted: true}
					} else {
						if d9.Loc == LocInputPair && int(d9.StackOff) < ctx.InputArgCount {
							d10 = ctx.RequestOptimizedCallback(int(d9.StackOff))
						} else {
							d10 = jitCopyScmerToPair(ctx, d9)
						}
					}
					ctx.FreeDesc(&d9)
					ctx.EnsureDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d10)
					d12 = args[2]
					d12.ID = 0
					if d12.Loc == LocLambdaTemplate {
						d13 = d12
					} else if d12.Loc == LocImm {
						optimizedCallback14 := jitPrepareCallback(d12.Imm)
						ctx.TrackImm(optimizedCallback14)
						d13 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback14, Rooted: true}
					} else {
						if d12.Loc == LocInputPair && int(d12.StackOff) < ctx.InputArgCount {
							d13 = ctx.RequestOptimizedCallback(int(d12.StackOff))
						} else {
							d13 = jitCopyScmerToPair(ctx, d12)
						}
					}
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d13)
					ctx.StabilizeDescForControlFlow(&d13)
					return bbs[3].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d13)
					d15 = jitCopyScmerToPair(ctx, d13)
					d16 = ctx.EmitGoCallScalar(jitGoABI_fbd8c70175770728, []JITValueDesc{d15, d1}, 2)
					ctx.StabilizeDescForControlFlow(&d16)
					d18 = d16
					d18.ID = 0
					d17 = ctx.EmitTagEqualsBorrowed(&d18, tagNil, JITValueDesc{Loc: LocAny})
					d19 = d17
					ctx.EnsureDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap20 := d1
						snap21 := d3
						snap22 := d4
						snap23 := d5
						snap24 := d7
						snap25 := d8
						snap26 := d9
						snap27 := d10
						snap28 := d12
						snap29 := d13
						snap30 := d15
						snap31 := d16
						snap32 := d17
						snap33 := d18
						snap34 := d19
						alloc35 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc35)
						d1 = snap20
						d3 = snap21
						d4 = snap22
						d5 = snap23
						d7 = snap24
						d8 = snap25
						d9 = snap26
						d10 = snap27
						d12 = snap28
						d13 = snap29
						d15 = snap30
						d16 = snap31
						d17 = snap32
						d18 = snap33
						d19 = snap34
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d36 = ctx.EmitNewSliceFromGoSlice(&d1)
					ctx.SyncDesc(&d36)
					if d36.Loc == LocRegPair || d36.Loc == LocStackPair || d36.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d36, &result)
						result.Type = d36.Type
					} else {
						switch d36.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d36)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d36)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d36)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d36, &result)
							result.Type = d36.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.StabilizeDescForControlFlow(&d1)
					d37 = jitCopyScmerToPair(ctx, d10)
					d38 = ctx.EmitGoCallScalar(jitGoABI_fbd8c70175770728, []JITValueDesc{d37, d1}, 2)
					ctx.EnsureDesc(&d38)
					d39 = d38
					_ = d39
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d41 = d39
					d41.ID = 0
					d40 = ctx.EmitBoolDesc(&d41, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.FreeDesc(&d38)
					d42 = d40
					ctx.EnsureDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d42.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap43 := d1
						snap44 := d3
						snap45 := d4
						snap46 := d5
						snap47 := d7
						snap48 := d8
						snap49 := d9
						snap50 := d10
						snap51 := d12
						snap52 := d13
						snap53 := d15
						snap54 := d16
						snap55 := d17
						snap56 := d18
						snap57 := d19
						snap58 := d36
						snap59 := d37
						snap60 := d38
						snap61 := d39
						snap62 := d40
						snap63 := d41
						snap64 := d42
						alloc65 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc65)
						d1 = snap43
						d3 = snap44
						d4 = snap45
						d5 = snap46
						d7 = snap47
						d8 = snap48
						d9 = snap49
						d10 = snap50
						d12 = snap51
						d13 = snap52
						d15 = snap53
						d16 = snap54
						d17 = snap55
						d18 = snap56
						d19 = snap57
						d36 = snap58
						d37 = snap59
						d38 = snap60
						d39 = snap61
						d40 = snap62
						d41 = snap63
						d42 = snap64
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d40)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					stackArray66 = ctx.AllocStack(int32(0))
					_ = stackArray66
					d67 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d67
					ctx.StabilizeDescForControlFlow(&d67)
					ctx.SyncDesc(&d67)
					if d67.Loc == LocReg || d67.Loc == LocFPReg {
						ctx.ProtectReg(d67.Reg)
					} else if d67.Loc == LocRegPair {
						ctx.ProtectReg(d67.Reg)
						ctx.ProtectReg(d67.Reg2)
					}
					d68 = d67
					if d68.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d68)
					if d68.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d68, int32(bbs[3].PhiBase)+int32(0), 3)
					} else {
						if d68.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d68.Reg, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d68.Reg2, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d68.Reg3, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+16)
					}
					if d67.Loc == LocReg || d67.Loc == LocFPReg {
						ctx.UnprotectReg(d67.Reg)
					} else if d67.Loc == LocRegPair {
						ctx.UnprotectReg(d67.Reg)
						ctx.UnprotectReg(d67.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					stackArray69 = ctx.AllocStack(int32(0))
					_ = stackArray69
					d70 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d70
					if d16.Type == tagSlice {
						d71 = jitKnownSliceHeader(ctx, &d16)
					} else {
						d71 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d16}, 3)
					}
					if d71.Loc == LocRegTriple {
						ctx.BindReg(d71.Reg, &d71)
						ctx.BindReg(d71.Reg2, &d71)
						ctx.BindReg(d71.Reg3, &d71)
					}
					callResults72 := JITEmitGoCallResults(ctx, jitGoABI_caeccb0b1551362d, []JITValueDesc{d71})
					d73 = callResults72[0]
					d74 = JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(bbs[3].PhiBase) + int32(0)}
					ctx.EmitCopyDescWords(&d74, &d73, 3)
					ctx.FreeDesc(&d73)
					d73 = d74
					ctx.StabilizeDescForControlFlow(&d73)
					return bbs[3].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      36,
		},
		Optimize:                 FirstParameterMutable("for_mut"),
		OptimizeFirstArgTransfer: true,
	})
	Declare(&Globalenv, &Declaration{
		Name: "for_mut",

		Fn: func(a ...Scmer) Scmer {
			state := asSlice(a[0], "for_mut init")
			cond := PrepareSerialProc(a[1])
			next := PrepareSerialProc(a[2])
			for ToBool(cond.Call(state)) {
				v := next.Call(state)
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
				declaration := declarations["for_mut"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var stackArray58 int32
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(24))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase0)+int32(0), 3)
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					d7 = args[2]
					d7.ID = 0
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := jitPrepareCallback(d7.Imm)
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.SyncDesc(&d3)
					if d3.Loc == LocReg || d3.Loc == LocFPReg {
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
						ctx.EmitStoreRegMem(d10.Reg, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d10.Reg2, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d10.Reg3, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+16)
					}
					if d3.Loc == LocReg || d3.Loc == LocFPReg {
						ctx.UnprotectReg(d3.Reg)
					} else if d3.Loc == LocRegPair {
						ctx.UnprotectReg(d3.Reg)
						ctx.UnprotectReg(d3.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d8)
					d11 = jitCopyScmerToPair(ctx, d8)
					d12 = ctx.EmitGoCallScalar(jitGoABI_fbd8c70175770728, []JITValueDesc{d11, d1}, 2)
					ctx.StabilizeDescForControlFlow(&d12)
					d14 = d12
					d14.ID = 0
					d13 = ctx.EmitTagEqualsBorrowed(&d14, tagNil, JITValueDesc{Loc: LocAny})
					d15 = d13
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap16 := d1
						snap17 := d2
						snap18 := d3
						snap19 := d4
						snap20 := d5
						snap21 := d7
						snap22 := d8
						snap23 := d10
						snap24 := d11
						snap25 := d12
						snap26 := d13
						snap27 := d14
						snap28 := d15
						alloc29 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc29)
						d1 = snap16
						d2 = snap17
						d3 = snap18
						d4 = snap19
						d5 = snap20
						d7 = snap21
						d8 = snap22
						d10 = snap23
						d11 = snap24
						d12 = snap25
						d13 = snap26
						d14 = snap27
						d15 = snap28
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
					ctx.FreeDesc(&d13)
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d30 = ctx.EmitNewSliceFromGoSlice(&d1)
					ctx.SyncDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocStackPair || d30.Loc == LocInputPair {
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d1)
					d31 = jitCopyScmerToPair(ctx, d5)
					d32 = ctx.EmitGoCallScalar(jitGoABI_fbd8c70175770728, []JITValueDesc{d31, d1}, 2)
					ctx.EnsureDesc(&d32)
					d33 = d32
					_ = d33
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d35 = d33
					d35.ID = 0
					d34 = ctx.EmitBoolDesc(&d35, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.FreeDesc(&d32)
					d36 = d34
					ctx.EnsureDesc(&d36)
					if d36.Loc != LocImm && d36.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d36.Loc == LocImm {
						if d36.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d36.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap37 := d1
						snap38 := d2
						snap39 := d3
						snap40 := d4
						snap41 := d5
						snap42 := d7
						snap43 := d8
						snap44 := d10
						snap45 := d11
						snap46 := d12
						snap47 := d13
						snap48 := d14
						snap49 := d15
						snap50 := d30
						snap51 := d31
						snap52 := d32
						snap53 := d33
						snap54 := d34
						snap55 := d35
						snap56 := d36
						alloc57 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc57)
						d1 = snap37
						d2 = snap38
						d3 = snap39
						d4 = snap40
						d5 = snap41
						d7 = snap42
						d8 = snap43
						d10 = snap44
						d11 = snap45
						d12 = snap46
						d13 = snap47
						d14 = snap48
						d15 = snap49
						d30 = snap50
						d31 = snap51
						d32 = snap52
						d33 = snap53
						d34 = snap54
						d35 = snap55
						d36 = snap56
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d34)
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					stackArray58 = ctx.AllocStack(int32(0))
					_ = stackArray58
					d59 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d59
					ctx.StabilizeDescForControlFlow(&d59)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d60 = d59
					if d60.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d60)
					if d60.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d60, int32(bbs[3].PhiBase)+int32(0), 3)
					} else {
						if d60.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d60.Reg, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d60.Reg2, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d60.Reg3, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					if d12.Type == tagSlice {
						d61 = jitKnownSliceHeader(ctx, &d12)
					} else {
						d61 = ctx.EmitGoCallScalar(jitGoABI_8f95458a6432b52e, []JITValueDesc{d12}, 3)
					}
					if d61.Loc == LocRegTriple {
						ctx.BindReg(d61.Reg, &d61)
						ctx.BindReg(d61.Reg2, &d61)
						ctx.BindReg(d61.Reg3, &d61)
					}
					ctx.StabilizeDescForControlFlow(&d61)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.ProtectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.ProtectReg(d61.Reg)
						ctx.ProtectReg(d61.Reg2)
					}
					d62 = d61
					if d62.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d62)
					if d62.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d62, int32(bbs[3].PhiBase)+int32(0), 3)
					} else {
						if d62.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d62.Reg, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d62.Reg2, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d62.Reg3, ctx.StackReg, int32(bbs[3].PhiBase)+int32(0)+16)
					}
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.UnprotectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.UnprotectReg(d61.Reg)
						ctx.UnprotectReg(d61.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      30,
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
				declaration := declarations["string"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d2 := d0
				ctx.SyncDesc(&d2)
				if d2.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d2.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d2.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d2 = tmpScalar
				}
				d2 = JITPrepareScmerGoArg(ctx, d2)
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d1 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d2}, 2)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				d3 := ctx.EmitGoCallScalar(jitGoABI_44610dda2d2977b6, []JITValueDesc{d1}, 2)
				if result.Loc == LocAny {
					return d3
				}
				ctx.EmitMovPairToResult(&d3, &result)
				result.Type = tagString
				return result
				return result
			},
			JITInlineCost: 5,
		},
	})
	DeclareSpecialForm(&Globalenv, &Declaration{
		Name:         "match",
		SyntaxKind:   SyntaxMatch,
		evalDispatch: specialFormMatch,

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
		Name:       "lambda",
		SyntaxKind: SyntaxLambda,

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
		Name:         "begin",
		SyntaxKind:   SyntaxBegin,
		evalDispatch: specialFormBegin,

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
				declaration := declarations["source"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(jitGoABI_08ec8031d8f52ec0, nil, 1)
				ctx.BindReg(d0.Reg, &d0)
				d1 := args[0]
				d1.ID = 0
				d3 := d1
				ctx.SyncDesc(&d3)
				if d3.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d3.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d3.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					d3 = tmpScalar
				}
				d3 = JITPrepareScmerGoArg(ctx, d3)
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair && d3.Loc != LocInputPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d2 := ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d3}, 2)
				ctx.FreeDesc(&d1)
				d4 := args[1]
				d4.ID = 0
				ctx.EnsureDesc(&d4)
				d5 := d4
				_ = d5
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d5.Loc != LocImm {
					ctx.EnsureDesc(&d5)
				}
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
					boxed := JITPrepareScmerGoArg(ctx, d5)
					d6 = ctx.EmitGoCallScalar(jitGoABI_724899b80b2780aa, []JITValueDesc{boxed}, 1)
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
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d9.Loc != LocImm {
					ctx.EnsureDesc(&d9)
				}
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
					boxed := JITPrepareScmerGoArg(ctx, d9)
					d10 = ctx.EmitGoCallScalar(jitGoABI_724899b80b2780aa, []JITValueDesc{boxed}, 1)
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
				ctx.EmitGoCallVoid(jitGoABI_49dd75b49ffbdf4b, []JITValueDesc{d0, d2})
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d6)
				ctx.EmitGoCallVoid(jitGoABI_b740735df4418c48, []JITValueDesc{d0, d6})
				ctx.FreeDesc(&d6)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d10)
				ctx.EmitGoCallVoid(jitGoABI_b3ef7032d672b137, []JITValueDesc{d0, d10})
				ctx.FreeDesc(&d10)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d12)
				ctx.EmitGoCallVoid(jitGoABI_24128c7ae9d19e5b, []JITValueDesc{d0, d12})
				ctx.FreeDesc(&d12)
				ctx.SyncDesc(&d0)
				d13 := ctx.EmitGoCallScalar(jitGoABI_11a997ee3b751875, []JITValueDesc{d0}, 2)
				d13.NoHeapPointer = false
				ctx.BindReg(d13.Reg, &d13)
				ctx.BindReg(d13.Reg2, &d13)
				if d13.Loc == LocImm {
					if result.Loc == LocAny {
						return d13
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d13)
				if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d13, &result)
					result.Type = d13.Type
				} else {
					switch d13.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d13)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d13)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d13)
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
			JITInlineCost:  29,
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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: native range iterator.
				ctx.Coverage.NativeCalls++
				declaration := declarations["source_coverage_report"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "scheme",

		Fn: func(a ...Scmer) Scmer {
			filename := "eval"
			if len(a) > 1 {
				filename = String(a[1])
			}
			return Read(filename, String(a[0]), true)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a scheme expression into a list",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "code", Description: "Scheme code"},
				{Kind: "string", Label: "filename", Description: "optional filename", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["scheme"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
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
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d2
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() > 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 1)
						d3 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl4 := ctx.ReserveLabel()
					ctx.EmitJump(d4.Condition, lbl2)
					ctx.EmitJmp(lbl4)
					ctx.FreeDesc(&d3)
					snap5 := d1
					snap6 := d2
					snap7 := d3
					snap8 := d4
					alloc9 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("eval")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc9)
					d1 = snap5
					d2 = snap6
					d3 = snap7
					d4 = snap8
					if !bbs[2].Rendered {
						snap10 := d1
						snap11 := d2
						snap12 := d3
						snap13 := d4
						alloc14 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc14)
						d1 = snap10
						d2 = snap11
						d3 = snap12
						d4 = snap13
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d15 = args[1]
					d15.ID = 0
					d17 = d15
					ctx.SyncDesc(&d17)
					if d17.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d17.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d17.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d17 = tmpScalar
					}
					d17 = JITPrepareScmerGoArg(ctx, d17)
					if d17.Loc != LocRegPair && d17.Loc != LocStackPair && d17.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d16 = ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d17}, 2)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d15)
					ctx.SyncDesc(&d16)
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.ProtectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.ProtectReg(d16.Reg)
						ctx.ProtectReg(d16.Reg2)
					}
					d18 = d16
					if d18.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d18)
					if d18.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d18, int32(bbs[2].PhiBase)+int32(0), 2)
					} else if d18.Loc == LocInputPair {
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreScmerToStack(d18, int32(bbs[2].PhiBase)+int32(0))
					} else if d18.Loc == LocRegPair || d18.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d18, int32(bbs[2].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreToStack(d18, int32(bbs[2].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
					}
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.UnprotectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.UnprotectReg(d16.Reg)
						ctx.UnprotectReg(d16.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d19 = args[0]
					d19.ID = 0
					d21 = d19
					ctx.SyncDesc(&d21)
					if d21.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d21.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d21.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d21 = tmpScalar
					}
					d21 = JITPrepareScmerGoArg(ctx, d21)
					if d21.Loc != LocRegPair && d21.Loc != LocStackPair && d21.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d20 = ctx.EmitGoCallScalar(jitGoABI_e1a33cf0a7aeca49, []JITValueDesc{d21}, 2)
					ctx.FreeDesc(&d19)
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
					if d1.Loc != LocRegPair && d1.Loc != LocStackPair && d1.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (Read arg0)")
					}
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d20.Imm)
						ptrWord, _ := d20.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d20.Imm.String())))
						d20 = tmpPair
					} else if d20.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d20.Type, Reg: ctx.AllocRegExcept(d20.Reg), Reg2: ctx.AllocRegExcept(d20.Reg)}
						switch d20.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d20)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d20)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d20)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d20)
						d20 = tmpPair
					}
					if d20.Loc != LocRegPair && d20.Loc != LocStackPair && d20.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (Read arg1)")
					}
					d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d1)
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d22)
					d23 = ctx.EmitGoCallScalar(jitGoABI_4db487eef1179dae, []JITValueDesc{d1, d20, d22}, 2)
					d23.NoHeapPointer = false
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.FreeDesc(&d22)
					ctx.FreeDesc(&d1)
					ctx.SyncDesc(&d23)
					if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d23, &result)
						result.Type = d23.Type
					} else {
						switch d23.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d23)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d23)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d23)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d23, &result)
							result.Type = d23.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 13,
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
				declaration := declarations["serialize"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(jitGoABI_e4e95af1c4f4c854, []JITValueDesc{d0, d1}, 2)
				d2.NoHeapPointer = false
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d2)
				d3 := ctx.EmitGoCallScalar(jitGoABI_44610dda2d2977b6, []JITValueDesc{d2}, 2)
				if result.Loc == LocAny {
					return d3
				}
				ctx.EmitMovPairToResult(&d3, &result)
				result.Type = tagString
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  5,
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
				declaration := declarations["pretty_print"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
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
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d2
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() >= 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 2)
						d3 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreaterOrEqual}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.SyncDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl4 := ctx.ReserveLabel()
					ctx.EmitJump(d4.Condition, lbl2)
					ctx.EmitJmp(lbl4)
					ctx.FreeDesc(&d3)
					snap5 := d1
					snap6 := d2
					snap7 := d3
					snap8 := d4
					alloc9 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl4)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(20)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc9)
					d1 = snap5
					d2 = snap6
					d3 = snap7
					d4 = snap8
					if !bbs[2].Rendered {
						snap10 := d1
						snap11 := d2
						snap12 := d3
						snap13 := d4
						alloc14 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc14)
						d1 = snap10
						d2 = snap11
						d3 = snap12
						d4 = snap13
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d15 = args[1]
					d15.ID = 0
					ctx.EnsureDesc(&d15)
					d16 = d15
					_ = d16
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d16.Loc != LocImm {
						ctx.EnsureDesc(&d16)
					}
					if d16.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int())}
					} else if d16.Type == tagInt && d16.Loc == LocRegPair {
						ctx.FreeReg(d16.Reg)
						d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg2}
						ctx.BindReg(d16.Reg2, &d17)
						ctx.BindReg(d16.Reg2, &d17)
					} else if d16.Type == tagInt && d16.Loc == LocReg {
						d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
						ctx.BindReg(d16.Reg, &d17)
						ctx.BindReg(d16.Reg, &d17)
					} else {
						boxed := JITPrepareScmerGoArg(ctx, d16)
						d17 = ctx.EmitGoCallScalar(jitGoABI_724899b80b2780aa, []JITValueDesc{boxed}, 1)
						d17.Type = tagInt
						ctx.BindReg(d17.Reg, &d17)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d17)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d17)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d15)
					ctx.SyncDesc(&d17)
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.ProtectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.ProtectReg(d17.Reg)
						ctx.ProtectReg(d17.Reg2)
					}
					d19 = d17
					if d19.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d19)
					ctx.EmitStoreToStack(d19, int32(bbs[2].PhiBase)+int32(0))
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.UnprotectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.UnprotectReg(d17.Reg)
						ctx.UnprotectReg(d17.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d20 = args[0]
					d20.ID = 0
					d20 = JITPrepareScmerGoArg(ctx, d20)
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocRegTriple || d1.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d20)
					ctx.SyncDesc(&d21)
					ctx.SyncDesc(&d1)
					d22 = ctx.EmitGoCallScalar(jitGoABI_90debfce13c931e5, []JITValueDesc{d20, d21, d1}, 2)
					d22.NoHeapPointer = false
					ctx.BindReg(d22.Reg, &d22)
					ctx.BindReg(d22.Reg2, &d22)
					ctx.FreeDesc(&d20)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d22)
					d23 = ctx.EmitGoCallScalar(jitGoABI_44610dda2d2977b6, []JITValueDesc{d22}, 2)
					ctx.EmitMovPairToResult(&d23, &result)
					result.Type = tagString
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  16,
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
		if p.compiled() != nil {
			sz += goAllocOverhead + align8(uint(p.compiled().CodeLen))
		}
		return sz
	case tagString, tagSymbol:
		ln := uint(auxVal(v.aux))
		if ln == 0 {
			return base
		}
		return base + goAllocOverhead + align8(ln)
	case tagCString, tagBString, tagLZ4:
		// These values borrow dictionary/blob storage. Its owning column accounts
		// for the backing allocation; this reference owns only its Scmer slot.
		// Do not decode strings or charge the shared dictionary per reference.
		return base
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

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_0651917b305ad187 uint64
var jitGoABI_08ec8031d8f52ec0 uint64
var jitGoABI_11a997ee3b751875 uint64
var jitGoABI_14cfa92b2b87ef7c uint64
var jitGoABI_158fc888362fa531 uint64
var jitGoABI_18fd4c44c7a31da6 uint64
var jitGoABI_1b8412732af78364 uint64
var jitGoABI_1e8379006790c0c3 uint64
var jitGoABI_24128c7ae9d19e5b uint64
var jitGoABI_2aaf5b10787be869 uint64
var jitGoABI_3f45ec6607233365 uint64
var jitGoABI_428f6123a6539c5d uint64
var jitGoABI_44610dda2d2977b6 uint64
var jitGoABI_446dee5074537592 uint64
var jitGoABI_45b2e9aa8d87590a uint64
var jitGoABI_49dd75b49ffbdf4b uint64
var jitGoABI_4c692bc4c57a241a uint64
var jitGoABI_4db487eef1179dae uint64
var jitGoABI_5077bb76faa6a775 uint64
var jitGoABI_5130bc3ed31521d6 uint64
var jitGoABI_575a0c3de7e0d5ed uint64
var jitGoABI_678e55d324700337 uint64
var jitGoABI_724899b80b2780aa uint64
var jitGoABI_751bf088537a5f21 uint64
var jitGoABI_79b61201b9e57f9d uint64
var jitGoABI_7ebaa2856648e378 uint64
var jitGoABI_830596eef656ff9c uint64
var jitGoABI_8f95458a6432b52e uint64
var jitGoABI_90debfce13c931e5 uint64
var jitGoABI_93f1f83c9e505d62 uint64
var jitGoABI_9bd0890fbc57aa9f uint64
var jitGoABI_b20afda8856e61ef uint64
var jitGoABI_b3734367ba267d31 uint64
var jitGoABI_b3ef7032d672b137 uint64
var jitGoABI_b740735df4418c48 uint64
var jitGoABI_b7afa8aa7afec23f uint64
var jitGoABI_b90d43f641e38b20 uint64
var jitGoABI_bd15746ee4fe9617 uint64
var jitGoABI_c3658a69a02c135c uint64
var jitGoABI_c7963fc7464639ca uint64
var jitGoABI_c8bd3fbb876619d4 uint64
var jitGoABI_c94eb298f761efc9 uint64
var jitGoABI_caeccb0b1551362d uint64
var jitGoABI_cd25371cee7f2008 uint64
var jitGoABI_e1a33cf0a7aeca49 uint64
var jitGoABI_e32fdcf449cd74a7 uint64
var jitGoABI_e4e95af1c4f4c854 uint64
var jitGoABI_e80a79328f388f6b uint64
var jitGoABI_ed9f7fe688ad8b78 uint64
var jitGoABI_efe6a72bfb144025 uint64
var jitGoABI_f471bfc54f82491c uint64
var jitGoABI_fbd8c70175770728 uint64

func init() {
	jitGoABI_0651917b305ad187 = GoFuncAddr(func(dst *[1]map[Symbol]struct{}, index int, value map[Symbol]struct{}) { dst[index] = value })
	jitGoABI_08ec8031d8f52ec0 = GoFuncAddr(func() *SourceInfo { return new(SourceInfo) })
	jitGoABI_11a997ee3b751875 = GoFuncAddr((func(arg0 *SourceInfo) Scmer { return NewSourceInfo(*arg0) }))
	jitGoABI_14cfa92b2b87ef7c = GoFuncAddr(func(dst, src *Proc) { *dst = *src })
	jitGoABI_158fc888362fa531 = GoFuncAddr(func(base *ProcOptimizerMeta, value TypeInfo) { base.Return = value })
	jitGoABI_18fd4c44c7a31da6 = GoFuncAddr(func(base *Env, value *Env) { base.Outer = value })
	jitGoABI_1b8412732af78364 = GoFuncAddr(NewProc)
	jitGoABI_1e8379006790c0c3 = GoFuncAddr((*Proc).compiled)
	jitGoABI_24128c7ae9d19e5b = GoFuncAddr(func(base *SourceInfo, value Scmer) { base.value = value })
	jitGoABI_2aaf5b10787be869 = GoFuncAddr(func(base *ProcOptimizerMeta, value procSequenceKind) { base.Sequence = value })
	jitGoABI_3f45ec6607233365 = GoFuncAddr(func() *Env { return new(Env) })
	jitGoABI_428f6123a6539c5d = GoFuncAddr(JITBuildScmerCallback)
	jitGoABI_44610dda2d2977b6 = GoFuncAddr(NewString)
	jitGoABI_446dee5074537592 = GoFuncAddr(func(m map[Symbol]struct{}, key Symbol, value struct{}) { m[key] = value })
	jitGoABI_45b2e9aa8d87590a = GoFuncAddr(optimizerNodeCount)
	jitGoABI_49dd75b49ffbdf4b = GoFuncAddr(func(base *SourceInfo, value string) { base.source = value })
	jitGoABI_4c692bc4c57a241a = GoFuncAddr(Optimize)
	jitGoABI_4db487eef1179dae = GoFuncAddr(Read)
	jitGoABI_5077bb76faa6a775 = GoFuncAddr((Scmer).IsProc)
	jitGoABI_5130bc3ed31521d6 = GoFuncAddr(JITStringEqual)
	jitGoABI_575a0c3de7e0d5ed = GoFuncAddr(func(value *[1]map[Symbol]struct{}) []map[Symbol]struct{} { return value[0:1:1] })
	jitGoABI_678e55d324700337 = GoFuncAddr(NewSymbol)
	jitGoABI_724899b80b2780aa = GoFuncAddr(Scmer.Int)
	jitGoABI_751bf088537a5f21 = GoFuncAddr(atomic.LoadUintptr)
	jitGoABI_79b61201b9e57f9d = GoFuncAddr(func() *Proc { return new(Proc) })
	jitGoABI_7ebaa2856648e378 = GoFuncAddr(jitMakeScmerSlice)
	jitGoABI_830596eef656ff9c = GoFuncAddr(func(base *Proc, value *JITEntryPoint) { base.Compiled = value })
	jitGoABI_8f95458a6432b52e = GoFuncAddr(jitAsSlice)
	jitGoABI_90debfce13c931e5 = GoFuncAddr(PrettyPrint)
	jitGoABI_93f1f83c9e505d62 = GoFuncAddr(jitProcCaptures)
	jitGoABI_9bd0890fbc57aa9f = GoFuncAddr(func(size int) map[Symbol]struct{} { return make(map[Symbol]struct{}, size) })
	jitGoABI_b20afda8856e61ef = GoFuncAddr(func(base *Proc, value uintptr) { base.JITCode = value })
	jitGoABI_b3734367ba267d31 = GoFuncAddr(closeJITProcedureCaptures)
	jitGoABI_b3ef7032d672b137 = GoFuncAddr(func(base *SourceInfo, value int) { base.col = value })
	jitGoABI_b740735df4418c48 = GoFuncAddr(func(base *SourceInfo, value int) { base.line = value })
	jitGoABI_b7afa8aa7afec23f = GoFuncAddr(func(base *Proc, value *Env) { base.En = value })
	jitGoABI_b90d43f641e38b20 = GoFuncAddr(func() *strings.Builder { return new(strings.Builder) })
	jitGoABI_bd15746ee4fe9617 = GoFuncAddr((func(arg0 *Proc) *Proc { value := (*Proc).snapshot(arg0); return &value }))
	jitGoABI_c3658a69a02c135c = GoFuncAddr((Scmer).Proc)
	jitGoABI_c7963fc7464639ca = GoFuncAddr(func(base *Proc, value *ProcOptimizerMeta) { base.OptimizerMeta = value })
	jitGoABI_c8bd3fbb876619d4 = GoFuncAddr(func(base *Proc, value Scmer) { base.Body = value })
	jitGoABI_c94eb298f761efc9 = GoFuncAddr((Scmer).IsSymbol)
	jitGoABI_caeccb0b1551362d = GoFuncAddr(JITCloneScmerSlice)
	jitGoABI_cd25371cee7f2008 = GoFuncAddr(func() *ProcOptimizerMeta { return new(ProcOptimizerMeta) })
	jitGoABI_e1a33cf0a7aeca49 = GoFuncAddr(Scmer.String)
	jitGoABI_e32fdcf449cd74a7 = GoFuncAddr(CloneOptimizerExpression)
	jitGoABI_e4e95af1c4f4c854 = GoFuncAddr(SerializeToString)
	jitGoABI_e80a79328f388f6b = GoFuncAddr((*strings.Builder).WriteString)
	jitGoABI_ed9f7fe688ad8b78 = GoFuncAddr(func() *[1]map[Symbol]struct{} { return new([1]map[Symbol]struct{}) })
	jitGoABI_efe6a72bfb144025 = GoFuncAddr(TryFold)
	jitGoABI_f471bfc54f82491c = GoFuncAddr(func(base *ProcOptimizerMeta, value bool) { base.HasReturn = value })
	jitGoABI_fbd8c70175770728 = GoFuncAddr(jitInvokeCallbackSlice)
}
