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
type lambdaDemand struct {
	object, keyBinding int
	key                Scmer
}
type lambdaPlan struct {
	identity      lambdaCodeIdentity
	codeIdentity  lambdaCodeIdentity
	codeOwner     *Proc
	sourceBody    weak.Pointer[byte]
	bodyTracked   bool
	paramsTracked bool
	entry         weak.Pointer[JITEntryPoint]
	sourceParams  weak.Pointer[byte]
	factory       Scmer
	bindings      []lambdaBinding
	globals       []Symbol
	demands       []lambdaDemand
	once          sync.Once
	compiled      Scmer
}

// Source bodies can also be evaluated in different lexical scopes. A local
// binding of a primitive is a capture, while its ordinary global declaration
// executes locally. Only compatible binding layouts can share a wire template.
func (plan *lambdaPlan) compatible(proc *Proc) bool {
	env := proc.En
	if env == nil {
		env = &Globalenv
	}
	for _, name := range plan.globals {
		if env.FindRead(name) != &Globalenv {
			return false
		}
	}
	for _, binding := range plan.bindings {
		if binding.inline < 0 && !binding.literal && binding.expression.IsSymbol() {
			owner := env.FindRead(binding.expression.Symbol())
			if owner == nil || owner == &Globalenv {
				return false
			}
		}
	}
	return true
}

var lambdaPlanIndex sync.Map     // lambdaCodeIdentity -> weak.Pointer[lambdaPlan]
var lambdaPlanBuildMu sync.Mutex // cold code publication only
const lambdaScopeSymbol = "\x00lambda:scope"

func lambdaProcedure(value Scmer) *Proc {
	value = value.WithoutSourceInfo()
	if value.IsProc() {
		return value.Proc()
	}
	if value.IsNativeFunc() {
		return JITProcForFunction(value.Func())
	}
	return nil
}

type lambdaMetadata struct {
	plan     atomic.Pointer[lambdaPlan]
	owner    *Proc
	template *lambdaTemplate
	shared   bool
}
type lambdaTemplate struct {
	parameters  []Symbol
	captures    []int
	environment bool
	contextFree bool
	bound       *lambdaMetadata
}

func (proc *Proc) lambdaMetadata() *lambdaMetadata {
	pointer := (*unsafe.Pointer)(unsafe.Pointer(&proc.lambdaMeta))
	if state := (*lambdaMetadata)(atomic.LoadPointer(pointer)); state != nil {
		return state
	}
	candidate := new(lambdaMetadata)
	if atomic.CompareAndSwapPointer(pointer, nil, unsafe.Pointer(candidate)) {
		return candidate
	}
	return (*lambdaMetadata)(atomic.LoadPointer(pointer))
}

// Metadata is heap-owned even when the callable header is borrowed from a JIT
// frame. Publication snapshots must never retain that stack header or its Env.
func (proc *Proc) serializationMetadata() *lambdaMetadata {
	for {
		old := proc.lambdaMetadata()
		if !old.shared {
			return old
		}
		owned := &lambdaMetadata{owner: old.owner, template: old.template}
		owned.plan.Store(old.plan.Load())
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&proc.lambdaMeta)), unsafe.Pointer(old), unsafe.Pointer(owned)) {
			return owned
		}
	}
}
func (proc *Proc) copyLambdaMetadata() *lambdaMetadata {
	old := (*lambdaMetadata)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&proc.lambdaMeta))))
	if old == nil {
		return nil
	}
	owned := &lambdaMetadata{owner: old.owner, template: old.template}
	owned.plan.Store(old.plan.Load())
	return owned
}

func procedureLambdaPlan(proc *Proc) *lambdaPlan {
	state := proc.lambdaMetadata()
	body, params := proc.Body.WithoutSourceInfo(), proc.Params.WithoutSourceInfo()
	key := lambdaCodeIdentity{body: uintptr(unsafe.Pointer(body.ptr)), params: uintptr(unsafe.Pointer(params.ptr)), bodyTag: body.aux, paramsTag: params.aux, vars: proc.NumVars}
	if entry := proc.compiled(); entry != nil {
		key.captureBase, key.captureCount = entry.CaptureBase, entry.CaptureCount
	}
	if body.IsSymbol() || body.IsString() {
		key.body = 0
		key.bodyName = body.String()
	}
	if params.IsSymbol() {
		key.params = 0
		key.paramsName = params.String()
	}
	var codeKey lambdaCodeIdentity
	entry := proc.compiled()
	if entry != nil && entry.CodePtr != nil {
		codeKey = lambdaCodeIdentity{code: uintptr(entry.CodePtr), vars: key.vars, captureBase: key.captureBase, captureCount: key.captureCount}
		for entry.Owner != nil {
			entry = entry.Owner
		}
	}
	if existing := state.plan.Load(); existing != nil && (existing.identity == key || codeKey.code != 0 && existing.codeIdentity == codeKey && existing.entry.Value() == entry) && existing.compatible(proc) {
		return existing
	}
	load := func() *lambdaPlan {
		if value, ok := lambdaPlanIndex.Load(key); ok {
			plan := value.(weak.Pointer[lambdaPlan]).Value()
			if plan != nil && (!plan.bodyTracked || plan.sourceBody.Value() == body.ptr) && (!plan.paramsTracked || plan.sourceParams.Value() == params.ptr) && plan.compatible(proc) {
				return plan
			}
		}
		if codeKey.code != 0 {
			if value, ok := lambdaPlanIndex.Load(codeKey); ok {
				plan := value.(weak.Pointer[lambdaPlan]).Value()
				if plan != nil && plan.entry.Value() == entry && plan.compatible(proc) {
					return plan
				}
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
			plan.codeIdentity = codeKey
			pointer := weak.Make(plan)
			lambdaPlanIndex.Store(key, pointer)
			if codeKey.code != 0 {
				lambdaPlanIndex.Store(codeKey, pointer)
			}
			runtime.AddCleanup(plan, func(value struct {
				key, codeKey lambdaCodeIdentity
				pointer      weak.Pointer[lambdaPlan]
			}) {
				lambdaPlanIndex.CompareAndDelete(value.key, value.pointer)
				if value.codeKey.code != 0 {
					lambdaPlanIndex.CompareAndDelete(value.codeKey, value.pointer)
				}
			}, struct {
				key, codeKey lambdaCodeIdentity
				pointer      weak.Pointer[lambdaPlan]
			}{key, codeKey, pointer})
		}
		lambdaPlanBuildMu.Unlock()
	}
	state.plan.Store(plan)
	return plan
}
func buildLambdaPlan(proc *Proc) *lambdaPlan {
	plan := &lambdaPlan{codeOwner: proc.lambdaMetadata().owner}
	if entry := proc.compiled(); entry != nil && entry.CodePtr != nil {
		for entry.Owner != nil {
			entry = entry.Owner
		}
		plan.entry = weak.Make(entry)
	}
	sourceBody, sourceParams := proc.Body.WithoutSourceInfo(), proc.Params.WithoutSourceInfo()
	plan.bodyTracked = sourceBody.IsSlice() || sourceBody.GetTag() == tagAny || sourceBody.IsProc()
	plan.paramsTracked = sourceParams.IsSlice() && sourceParams.ptr != nil
	if plan.bodyTracked {
		plan.sourceBody = weak.Make(sourceBody.ptr)
	}
	if plan.paramsTracked {
		plan.sourceParams = weak.Make(sourceParams.ptr)
	}

	var params []Scmer
	captureNames := map[Symbol]int{}
	bind := func(binding lambdaBinding) Scmer {
		// Names contain NUL so ordinary source identifiers cannot collide with
		// compiler-generated context parameters.
		name := NewSymbol("\x00lambda:context:" + strconv.Itoa(len(params)))
		captureNames[name.Symbol()] = len(params)
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
			if previous.inline < 0 && !previous.literal && astStructuralEqual(previous.expression, expression) {
				return params[i]
			}
		}
		return bind(lambdaBinding{expression: expression, inline: -1})
	})
	for _, name := range jitLambdaFreeSymbols(proc.Params, body) {
		if frame.Outer.FindRead(name) == &Globalenv {
			plan.globals = append(plan.globals, Symbol(strings.Clone(string(name))))
		}
	}
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
		raw := plain.Slice()
		items := lambdaSyntaxOperands(raw)
		if len(items) == 2 && scmerIsSymbol(items[0], "quote") {
			if !plainClusterLiteral(items[1], 0) {
				return bind(lambdaBinding{pointer: weak.Make(items[1].ptr), aux: items[1].aux, inline: -1, literal: true})
			}
			return value
		}
		var result []Scmer
		if len(items) != len(raw) {
			result = append([]Scmer(nil), items...)
		}
		for i, item := range items {
			next := native(item)
			if next != item && result == nil {
				result = append([]Scmer(nil), items...)
			}
			if result != nil {
				result[i] = next
			}
		}
		if result != nil {
			items = result
			value = NewSlice(result)
		}
		if len(items) > 0 && items[0].IsSymbol() {
			if index, exists := captureNames[items[0].Symbol()]; exists {
				demand := lambdaDemand{object: index, keyBinding: -1}
				if len(items) == 2 {
					key := items[1].WithoutSourceInfo()
					if key.IsString() {
						demand.key = ownLambdaCode(key)
					} else if key.IsSymbol() {
						if binding, exists := captureNames[key.Symbol()]; exists {
							demand.keyBinding = binding
						}
					}
				}
				plan.demands = append(plan.demands, demand)
			}
		}
		return value
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
// compileLambdaTemplate compiles only the code which will actually execute.
// Rebindings copy its GC-typed inline capture array; they neither invoke nor
// compile an enclosing factory. Variadic and fixed-arity lambdas share this path.
func compileLambdaTemplate(expression Scmer) Scmer {
	factory := expression.WithoutSourceInfo().Slice()
	if len(factory) < 3 || !scmerIsSymbol(factory[0], "lambda") || !factory[1].WithoutSourceInfo().IsSlice() {
		panic("invalid lambda template")
	}
	parameters := factory[1].WithoutSourceInfo().Slice()
	vars := make(Vars, len(parameters))
	template := &lambdaTemplate{parameters: make([]Symbol, len(parameters)), environment: true}
	for i, param := range parameters {
		template.parameters[i] = mustSymbol(param)
		vars[template.parameters[i]] = NewNil()
	}
	value := Eval(factory[2], &Env{Vars: vars, Outer: &Globalenv})
	if !value.IsProc() {
		panic("lambda template must contain a lambda")
	}
	original := value.Proc().Snapshot()
	consumesEnvironment := jitExpressionConsumesRuntimeEnv(original.Body)
	free := map[Symbol]bool{}
	for _, name := range jitLambdaFreeSymbols(original.Params, original.Body) {
		free[name] = true
	}
	template.contextFree = !consumesEnvironment
	for _, name := range template.parameters {
		if free[name] {
			template.contextFree = false
		}
	}
	if jitEnabled && !consumesEnvironment {
		base := original.NumVars
		if original.Params.IsSlice() && len(original.Params.Slice()) > base {
			base = len(original.Params.Slice())
		}
		if original.Params.IsSymbol() && base < 1 {
			base = 1
		}
		base, _ = jitRequiredLocalSlots(original.Body, base)
		bindings := map[Symbol]NthLocalVar{}
		var symbols []Symbol
		var keys []Scmer
		for i, name := range template.parameters {
			if !free[name] {
				continue
			}
			bindings[name] = NthLocalVar(base + len(symbols))
			symbols = append(symbols, name)
			keys = append(keys, NewSymbol(string(name)))
			template.captures = append(template.captures, i)
		}
		prepared := original
		prepared.Body = jitBindLambdaCaptures(original.Body, bindings, nil, nil)
		prepared.syntaxSize = nil
		prepared.NumVars = base + len(symbols)
		prepared.En = &Globalenv
		prepared.Compiled = &JITEntryPoint{CaptureBase: base, CaptureCount: len(symbols), CaptureKeys: keys, CaptureSymbols: symbols}
		compiled := jitAllocateProcContext(&prepared, len(symbols))
		value = CompileJIT(Scmer{(*byte)(unsafe.Pointer(compiled)), makeAux(tagProc, 0)}, true)
		if value.Proc().nativeCode() != 0 {
			template.environment = false
		} else {
			value = NewProcStruct(original)
		}
	} else {
		value = CompileJIT(value, true)
	}
	p := value.Proc()
	template.bound = &lambdaMetadata{owner: p, shared: true}
	p.lambdaMeta = &lambdaMetadata{owner: p, template: template}
	return value
}
func LambdaRebind(old Scmer, context ...Scmer) Scmer {
	proc := lambdaProcedure(old)
	if proc == nil {
		panic("lambda_rebind expects a Scheme lambda")
	}
	p := proc
	if proc.lambdaMetadata().template == nil {
		plan := procedureLambdaPlan(proc)
		plan.once.Do(func() { plan.compiled = compileLambdaTemplate(plan.factory) })
		p = plan.compiled.Proc()
	}
	template := p.lambdaMetadata().template
	arity := len(template.parameters)
	if len(context) != arity && len(context) != arity-1 {
		panic(fmt.Sprintf("lambda_rebind expects %d bindings, got %d", arity-1, len(context)))
	}
	// A function which reads no factory bindings already is the complete
	// callable. Its immutable header and code have the same Go/peer lifetime.
	if template.contextFree {
		return Scmer{(*byte)(unsafe.Pointer(p)), makeAux(tagProc, 0)}
	}
	if len(context) == arity-1 {
		scope := NewNil()
		if entry := proc.compiled(); entry != nil && proc.nativeCode() != 0 {
			for i, name := range entry.CaptureSymbols {
				if name == Symbol(lambdaScopeSymbol) {
					scope = jitProcCaptures(proc)[i]
					break
				}
			}
		}
		if scope.IsNil() && proc.En != nil {
			if frame := proc.En.FindRead(Symbol(lambdaScopeSymbol)); frame != nil {
				scope = frame.Vars[Symbol(lambdaScopeSymbol)]
			}
		}
		context = append(append([]Scmer(nil), context...), scope)
	}
	count := 0
	if entry := p.compiled(); entry != nil && entry.CodePtr != nil {
		count = entry.CaptureCount
	}
	var bound *Proc
	if p.nativeCode() != 0 {
		bound = jitAllocateProcContext(p, count)
	} else {
		copy := p.Snapshot()
		bound = &copy
	}
	if !template.environment {
		captures := jitProcCaptures(bound)
		for index, parameter := range template.captures {
			captures[index] = context[parameter]
		}
	} else {
		vars := make(Vars, arity)
		for i, name := range template.parameters {
			vars[name] = context[i]
		}
		bound.En = &Env{Vars: vars, Outer: &Globalenv}
	}
	bound.lambdaMeta = template.bound
	return Scmer{(*byte)(unsafe.Pointer(bound)), makeAux(tagProc, 0)}
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
