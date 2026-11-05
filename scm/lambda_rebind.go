/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "weak"
import "unsafe"
import "strconv"
import "strings"
import "runtime"
import "sync/atomic"

// A plan owns code and binding paths, never the creation environment. The weak
// index does not turn transient query code into a process-lifetime code cache.
type lambdaCodeIdentity struct {
	code                            uintptr
	bodyName, paramsName            string
	body, params                    uintptr
	bodyTag, paramsTag              uint64
	vars, captureBase, captureCount int
}
type lambdaBinding struct {
	expression Scmer
	inline     int
	literal    bool
	pointer    weak.Pointer[byte]
	aux        uint64
}
type lambdaPlan struct {
	identity      lambdaCodeIdentity
	codeOwner     *Proc
	sourceBody    weak.Pointer[byte]
	bodyTracked   bool
	paramsTracked bool
	entry         weak.Pointer[JITEntryPoint]
	sourceParams  weak.Pointer[byte]
	factory       Scmer
	bindings      []lambdaBinding
	once          sync.Once
	compiled      Scmer
}

var lambdaPlanIndex sync.Map     // lambdaCodeIdentity -> weak.Pointer[lambdaPlan]
var lambdaPlanBuildMu sync.Mutex // cold code publication only
const lambdaScopeSymbol = "\x00lambda:scope"

func lambdaProcedure(value Scmer) *Proc {
	if value.IsProc() {
		return value.Proc()
	}
	if value.IsNativeFunc() {
		return JITProcForFunction(value.Func())
	}
	return nil
}
func procedureLambdaPlan(proc *Proc) *lambdaPlan {
	key := lambdaCodeIdentity{body: uintptr(unsafe.Pointer(proc.Body.ptr)), params: uintptr(unsafe.Pointer(proc.Params.ptr)), bodyTag: proc.Body.aux, paramsTag: proc.Params.aux, vars: proc.NumVars}
	if entry := proc.compiled(); entry != nil {
		key.captureBase, key.captureCount = entry.CaptureBase, entry.CaptureCount
	}
	if proc.Body.IsSymbol() || proc.Body.IsString() {
		key.body = 0
		key.bodyName = proc.Body.String()
	}
	if proc.Params.IsSymbol() {
		key.params = 0
		key.paramsName = proc.Params.String()
	}
	if entry := proc.compiled(); entry != nil && entry.CodePtr != nil {
		key.code = uintptr(entry.CodePtr)
		key.body, key.params = 0, 0
		key.bodyName, key.paramsName = "", ""
		key.bodyTag, key.paramsTag = 0, 0
	}

	if existing := (*lambdaPlan)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&proc.rebindPlan)))); existing != nil && existing.identity == key {
		return existing
	}
	load := func() *lambdaPlan {
		if value, ok := lambdaPlanIndex.Load(key); ok {
			plan := value.(weak.Pointer[lambdaPlan]).Value()
			if plan != nil && (!plan.bodyTracked || plan.sourceBody.Value() == proc.Body.ptr) && (!plan.paramsTracked || plan.sourceParams.Value() == proc.Params.ptr) && (key.code == 0 || plan.entry.Value() == proc.compiled()) {
				return plan
			}
		}
		return nil
	}
	plan := load()
	if plan == nil {
		lambdaPlanBuildMu.Lock()
		plan = load()
		if plan == nil {
			plan = buildLambdaPlan(proc)
			key.bodyName = strings.Clone(key.bodyName)
			key.paramsName = strings.Clone(key.paramsName)
			plan.identity = key
			pointer := weak.Make(plan)
			lambdaPlanIndex.Store(key, pointer)
			runtime.AddCleanup(plan, func(value struct {
				key     lambdaCodeIdentity
				pointer weak.Pointer[lambdaPlan]
			}) {
				lambdaPlanIndex.CompareAndDelete(value.key, value.pointer)
			}, struct {
				key     lambdaCodeIdentity
				pointer weak.Pointer[lambdaPlan]
			}{key, pointer})
		}
		lambdaPlanBuildMu.Unlock()
	}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&proc.rebindPlan)), unsafe.Pointer(plan))
	return plan
}
func buildLambdaPlan(proc *Proc) *lambdaPlan {
	plan := &lambdaPlan{codeOwner: proc.codeOwner}
	if entry := proc.compiled(); entry != nil && entry.CodePtr != nil {
		plan.entry = weak.Make(entry)
	} else {
		plan.bodyTracked = proc.Body.IsSlice() || proc.Body.IsSourceInfo() || proc.Body.GetTag() == tagAny || proc.Body.IsProc()
		plan.paramsTracked = proc.Params.IsSlice() && proc.Params.ptr != nil
		if plan.bodyTracked {
			plan.sourceBody = weak.Make(proc.Body.ptr)
		}
		if plan.paramsTracked {
			plan.sourceParams = weak.Make(proc.Params.ptr)
		}
	}
	var params []Scmer
	bind := func(binding lambdaBinding) Scmer {
		// Names contain NUL so ordinary source identifiers cannot collide with
		// compiler-generated context parameters.
		name := NewSymbol("\x00lambda:context:" + strconv.Itoa(len(params)))
		params = append(params, name)
		plan.bindings = append(plan.bindings, binding)
		return name
	}
	body := proc.Body
	if entry := proc.compiled(); entry != nil && entry.CaptureCount != 0 {
		indices := map[int]Scmer{}
		body = transformJITCaptures(body, entry.CaptureBase, entry.CaptureCount, 0, func(index int) Scmer {
			if index < len(entry.CaptureSymbols) && entry.CaptureSymbols[index] == Symbol(lambdaScopeSymbol) {
				return NewSymbol(lambdaScopeSymbol)
			}
			if name, ok := indices[index]; ok {
				return name
			}
			name := bind(lambdaBinding{inline: index})
			indices[index] = name
			return name
		})
	}
	frame := &Env{Outer: proc.En}
	if frame.Outer == nil {
		frame.Outer = &Globalenv
	}
	bound := procedureWireBindings(proc.Params, body, nil)
	body = transformProcedureCaptures(body, frame, []map[Symbol]struct{}{bound}, func(expression Scmer) Scmer {
		if scmerIsSymbol(expression, lambdaScopeSymbol) {
			return expression
		}
		for i, previous := range plan.bindings {
			if previous.inline < 0 && !previous.literal && Equal(previous.expression, expression) {
				return params[i]
			}
		}
		return bind(lambdaBinding{expression: expression, inline: -1})
	})
	var native func(Scmer) Scmer
	native = func(value Scmer) Scmer {
		plain := value.WithoutSourceInfo()
		if plain.ptr == nil {
			return value
		}
		if _, ok := ClusterObjectOf(plain); ok {
			return bind(lambdaBinding{pointer: weak.Make(plain.ptr), aux: plain.aux, inline: -1, literal: true})
		}
		switch plain.GetTag() {
		case tagAny, tagProc, tagFuncEnv, tagClosure, tagJIT, tagParser, tagPromise, tagVector, tagBSON, tagFastDict:
			return bind(lambdaBinding{pointer: weak.Make(plain.ptr), aux: plain.aux, inline: -1, literal: true})
		case tagFunc:
			if DeclarationForValue(plain) == nil {
				return bind(lambdaBinding{pointer: weak.Make(plain.ptr), aux: plain.aux, inline: -1, literal: true})
			}
		}
		if !plain.IsSlice() {
			return value
		}
		items := plain.Slice()
		if len(items) == 2 && scmerIsSymbol(items[0], "quote") {
			if !plainClusterLiteral(items[1], 0) {
				return bind(lambdaBinding{pointer: weak.Make(items[1].ptr), aux: items[1].aux, inline: -1, literal: true})
			}
			return value
		}
		var result []Scmer
		for i, item := range items {
			next := native(item)
			if next != item && result == nil {
				result = append([]Scmer(nil), items...)
			}
			if result != nil {
				result[i] = next
			}
		}
		if result == nil {
			return value
		}
		return NewSlice(result)
	}
	body = native(body)
	params = append(params, NewSymbol(lambdaScopeSymbol))
	inner := NewSlice([]Scmer{NewSymbol("lambda"), proc.Params, body, NewInt(int64(proc.NumVars))})
	plan.factory = ownLambdaCode(NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(params), inner}))
	return plan
}
func (plan *lambdaPlan) context(proc *Proc) []Scmer {
	defer runtime.KeepAlive(proc)
	values := make([]Scmer, len(plan.bindings))
	frame := &Env{Outer: proc.En}
	if frame.Outer == nil {
		frame.Outer = &Globalenv
	}
	inline := jitProcCaptures(proc)
	for i, binding := range plan.bindings {
		if binding.inline >= 0 {
			values[i] = inline[binding.inline]
		} else if binding.literal {
			pointer := binding.pointer.Value()
			if pointer == nil {
				panic("lambda context expired while its procedure is live")
			}
			values[i] = Scmer{pointer, binding.aux}
		} else {
			values[i] = Eval(binding.expression, frame)
		}
	}
	return values
}

// LambdaRebind shares a code factory while constructing a private capture
// context. The original closure and all other rebindings remain immutable.
func LambdaRebind(old Scmer, context ...Scmer) Scmer {
	proc := lambdaProcedure(old)
	if proc == nil {
		panic("lambda_rebind expects a Scheme lambda")
	}
	factory := old
	if !proc.lambdaFactory {
		plan := procedureLambdaPlan(proc)
		plan.once.Do(func() {
			plan.compiled = CompileJIT(Eval(plan.factory, &Globalenv), true)
			plan.compiled.Proc().lambdaFactory = true
		})
		factory = plan.compiled
	}
	p := factory.Proc()
	arity := len(p.Params.Slice())
	if len(context) == arity-1 {
		context = append(append([]Scmer(nil), context...), NewNil())
	}
	if len(context) != arity {
		panic(fmt.Sprintf("lambda_rebind expects %d bindings, got %d", arity-1, len(context)))
	}
	value := Apply(factory, context...)
	if !value.IsProc() {
		panic("lambda_rebind factory did not return a lambda")
	}
	if entry := value.Proc().compiled(); entry != nil && entry.CaptureCount == 0 {
		value = NewProcStruct(value.Proc().snapshot())
	}
	value.Proc().codeOwner = p
	return value
}
func initLambdaRebind() {
	Declare(&Globalenv, &Declaration{Name: "lambda_rebind", Fn: func(args ...Scmer) Scmer {
		if len(args) == 0 {
			panic("lambda_rebind needs a lambda")
		}
		return LambdaRebind(args[0], args[1:]...)
	}, Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "func"}, {Kind: "any", Variadic: true}}, Return: &TypeDescriptor{Kind: "func"}}})
}

// LambdaScopeSymbol names the receiver-local execution binding in code factories.
func LambdaScopeSymbol() string { return lambdaScopeSymbol }

// Native callables use one wrapper code body; the callable itself is a binding.
// Creating a fresh AST for every host callback would defeat pointer identity.
var nativeProcedureBody = NewSlice([]Scmer{NewSymbol("apply"), NewSymbol("\x00lambda:native"), NewSymbol("args")})

var nativeDeclaredBodies sync.Map // startup declarations own their immutable code
func nativeProcedureEnvelope(value Scmer) Scmer {
	if declaration := DeclarationForValue(value); declaration != nil {
		body, ok := nativeDeclaredBodies.Load(declaration)
		if !ok {
			candidate := NewSlice([]Scmer{NewSymbol("apply"), NewSymbol(declaration.Name), NewSymbol("args")})
			body, _ = nativeDeclaredBodies.LoadOrStore(declaration, candidate)
		}
		return NewProcStruct(Proc{Params: NewSymbol("args"), Body: body.(Scmer), En: &Globalenv})
	}
	return NewProcStruct(Proc{Params: NewSymbol("args"), Body: nativeProcedureBody, En: &Env{Vars: Vars{Symbol("\x00lambda:native"): value}, Outer: &Globalenv}})
}

// Detach normalized code once. Source-tree backing arrays can also contain
// context constants outside a retained subslice; those must not become roots.
func ownLambdaCode(value Scmer) Scmer {
	value = value.WithoutSourceInfo()
	switch value.GetTag() {
	case tagString:
		return NewString(strings.Clone(value.String()))
	case tagSymbol:
		return NewSymbol(strings.Clone(value.String()))
	case tagSlice:
		items := value.Slice()
		copy := make([]Scmer, len(items))
		for i, item := range items {
			copy[i] = ownLambdaCode(item)
		}
		return NewSlice(copy)
	default:
		return value
	}
}
