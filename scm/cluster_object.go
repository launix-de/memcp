/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "bytes"
import "sync"
import "weak"
import "runtime"
import "errors"
import "context"
import "unsafe"

// A local encoding failure before dispatch. It is safe to split an unsent
// output batch; a remote error or ambiguous completion must never be replayed.
var ErrClusterRequestTooLarge = errors.New("oversized MOSI message")

// ClusterRequestHandler is the routing contract for native owner endpoints.
// Implementations own request policy; transport only resolves and dispatches.
// All requests execute outside generated row loops.
type ClusterRequestHandler interface {
	HandleClusterRequest(context.Context, ClusterRequest) (any, error)
}

// ClusterObject adds the serialization contract only for runtime values which
// may be captured by a closure. Nodes/databases are routing targets and require
// only ClusterRequestHandler; they never implement deliberately failing serializers.
type ClusterObject interface {
	ClusterRequestHandler
	// Serialize returns reconstruction code, not an encoded Go representation.
	// Each object owns its choice of a registered reference or portable value.
	Serialize(*SerializationContext) (Scmer, error)
	ClusterObjectType() string
}

type ClusterRequest struct {
	// A borrowed delivery may cancel its local waiter while the native owner
	// retains and joins the remote invocation through its independent Close.
	// This local policy is never serialized and never releases the owner's state.
	CancelWithoutJoin bool `json:"-"`
	Method            string
	Arguments         []Scmer
	Payload           any `json:"-"` // private decoded protocol payload, never runtime serialization
}

// ClusterReference names an incarnation-bound owner endpoint. On the wire Token
// is a pending-transfer reservation; import replaces it with the peer's shared
// lease generation. A late old GC message cannot revoke a newer Go lifetime.
// View is an immutable invocation view, not the shared endpoint's mutable state.
type ClusterReference struct {
	URI         string `json:"uri"`
	Owner       string `json:"owner"`
	Incarnation string `json:"incarnation"`
	Token       string `json:"token"`
	Type        string `json:"type"`
	View        []byte `json:"view"`
}

// ClusterHandle is owned by the imported value, not by a transport registry.
// Keeping a callable/reader alive therefore keeps its remote reference alive.
// The transport attaches GC cleanup to its private implementation of Call.
// ClusterLifetime is opaque peer-local ownership. Native factories opt into
// cache state; the registry never chooses a value's reconstruction/cache policy.
type ClusterLifetime interface{ LocalState(func() any) any }

type ClusterHandle struct {
	Lifetime  ClusterLifetime
	Reference ClusterReference
	View      Scmer // decoded immutable owner view, reconstructed before execution
	Call      func(context.Context, ClusterRequest) (Scmer, error)
}

var ClusterObjectResolver func(ClusterReference) (Scmer, error) // immutable package-startup adapter

var clusterFactories sync.Map // startup-only type -> immutable reconstruction function
func RegisterClusterObjectType(kind string, factory func(ClusterHandle) Scmer) {
	if _, exists := clusterFactories.LoadOrStore(kind, factory); exists {
		panic("duplicate cluster object type: " + kind)
	}
}
func ReconstructClusterObject(handle ClusterHandle) (Scmer, error) {
	factory, ok := clusterFactories.Load(handle.Reference.Type)
	if !ok {
		return NewNil(), fmt.Errorf("unknown cluster object type %q", handle.Reference.Type)
	}
	return factory.(func(ClusterHandle) Scmer)(handle), nil
}

// Native callables retain their existing first-field ABI and type descriptors.
// Only serialization inspects the extra, GC-visible owner pointer. There is no
// additional lookup, lock or indirection in the ordinary callable hot path.
type clusterCallable struct {
	call   func(...Scmer) Scmer
	object ClusterObject
}

var clusterCallableTypes sync.Map
var clusterValueExtractors sync.Map // startup-only custom tag -> object extractor
func RegisterClusterValueExtractor(tag uint8, extract func(Scmer) ClusterObject) {
	clusterValueExtractors.Store(tag, extract)
}

func NewClusterFunc(call func(...Scmer) Scmer, object ClusterObject, kind CallableTypeID) Scmer {
	clusterCallableTypes.Store(kind, true)
	cell := &clusterCallable{call, object}
	return Scmer{(*byte)(unsafe.Pointer(cell)), makeAux(tagFunc, uint64(kind))}
}
func ClusterObjectOf(value Scmer) (ClusterObject, bool) {
	value = value.WithoutSourceInfo()
	if value.GetTag() == tagFunc {
		if _, ok := clusterCallableTypes.Load(CallableTypeID(auxVal(value.aux))); ok {
			return (*clusterCallable)(unsafe.Pointer(value.ptr)).object, true
		}
	}
	if value.GetTag() == tagAny {
		object, ok := value.Any().(ClusterObject)
		return object, ok
	}
	if extract, ok := clusterValueExtractors.Load(value.GetTag()); ok {
		return extract.(func(Scmer) ClusterObject)(value), true
	}
	return nil, false
}

// SerializationContext belongs to one statement. Demands are collected from native
// callback captures, for every object type alike. Objects produce their own
// immutable view once; each target peer gets a distinct registered handle.
type SerializationContext struct {
	ExportHandle func(Handle) (Scmer, error)
	Invocation   Scmer // native owner of invocation-scoped exported callbacks
	Export       func(ClusterObject, Scmer) (ClusterReference, error)
	root         *SerializationContext
	mu           sync.Mutex
	views        map[ClusterObject]*clusterViewCapture
	literals     map[Scmer]*clusterLiteralCapture
	demands      map[ClusterObject]map[Scmer]struct{}
}

type clusterViewCapture struct {
	done chan struct{}
	view Scmer
	err  error
}

func (bindings *SerializationContext) Reference(object ClusterObject) (ClusterReference, error) {
	if bindings == nil || bindings.Export == nil {
		return ClusterReference{}, fmt.Errorf("cluster object %s requires an export context", object.ClusterObjectType())
	}
	export := bindings.Export
	if bindings.root != nil {
		bindings = bindings.root
	}
	bindings.mu.Lock()
	if bindings.views == nil {
		bindings.views = make(map[ClusterObject]*clusterViewCapture)
	}
	capture, exists := bindings.views[object]
	if !exists {
		capture = &clusterViewCapture{done: make(chan struct{})}
		bindings.views[object] = capture
	}
	keys := make([]Scmer, 0, len(bindings.demands[object]))
	for key := range bindings.demands[object] {
		keys = append(keys, key)
	}
	_, project := bindings.demands[object]
	bindings.mu.Unlock()
	if !exists {
		// Object code may itself schedule work. Never call it while holding the
		// publication mutex; concurrent captures join this one immutable result.
		func() {
			defer close(capture.done)
			defer func() {
				if failure := recover(); failure != nil {
					capture.err = fmt.Errorf("cluster view capture: %v", failure)
					panic(failure)
				}
			}()
			capture.view, capture.err = ClusterObjectValue(object, context.Background(), ClusterRequest{Method: "view"})
		}()
	} else {
		<-capture.done
	}
	if capture.err != nil {
		return ClusterReference{}, capture.err
	}
	view := capture.view
	if projection, ok := object.(ClusterReadProjection); ok && project {
		var err error
		view, err = projection.ProjectClusterView(view, keys)
		if err != nil {
			return ClusterReference{}, err
		}
	}
	return export(object, view)
}

// ReferenceExpression is the common reference representation, chosen explicitly
// by reference-backed implementations of Serialize. The object contract permits
// other reconstruction expressions without adding a type switch to the printer.
// Registration, peer ownership and demanded views stay in the supplied context;
// no storage transport is imported or initialized by standalone SCM.
func (bindings *SerializationContext) ReferenceExpression(object ClusterObject) (Scmer, error) {
	ref, err := bindings.Reference(object)
	if err != nil {
		return NewNil(), err
	}
	return ClusterReferenceExpression(ref), nil
}

// ClusterReferenceExpression reconstructs a previously registered reference.
func ClusterReferenceExpression(ref ClusterReference) Scmer {
	return NewSlice([]Scmer{NewSymbol("cluster_object"), NewString(ref.URI), NewString(ref.Owner), NewString(ref.Incarnation), NewString(ref.Token), NewString(ref.Type), NewString(string(ref.View))})
}

// ProcedureDecode is invocation-owned. Reconstruction is completed before
// Eval builds the callback, never lazily during its row execution.
type ProcedureDecode struct {
	ResolveHandle func(uint64) (Handle, error)
	// Prepare is an optional embedding-owned transformation of the validated
	// lambda. SCM knows neither the host's operators nor its execution context.
	Prepare func(Scmer) (Scmer, error)
	Resolve func(ClusterReference) (Scmer, error)
}

// ClusterReadProjection lets the object restrict its prepared read view to
// demanded arguments without teaching serialization about sessions/cachemaps.
type ClusterReadProjection interface {
	ProjectClusterView(Scmer, []Scmer) (Scmer, error)
}

func (bindings *SerializationContext) WithExport(export func(ClusterObject, Scmer) (ClusterReference, error)) *SerializationContext {
	handles := bindings.ExportHandle
	if bindings.root != nil {
		bindings = bindings.root
	}
	return &SerializationContext{Export: export, ExportHandle: handles, root: bindings}
}

// Collect demands before Serialize exports the first object. Runtime lists and
// dictionaries contain values; their symbols are never interpreted as calls.
func (bindings *SerializationContext) prepare(value Scmer, literal bool) {
	if bindings.root != nil {
		bindings = bindings.root
	}
	value = value.WithoutSourceInfo()
	if value.IsProc() {
		bindings.prepare(CloseProcedure(value).Proc().Body, false)
		return
	}
	if value.IsFastDict() {
		for _, item := range value.FastDict().Pairs {
			bindings.prepare(item, true)
		}
		return
	}
	demand := func(object ClusterObject, key Scmer) {
		bindings.mu.Lock()
		defer bindings.mu.Unlock()
		if bindings.demands == nil {
			bindings.demands = make(map[ClusterObject]map[Scmer]struct{})
		}
		if bindings.demands[object] == nil {
			bindings.demands[object] = make(map[Scmer]struct{})
		}
		bindings.demands[object][key] = struct{}{}
	}
	if object, ok := ClusterObjectOf(value); ok {
		demand(object, NewNil())
		return
	}
	if !value.IsSlice() {
		return
	}
	items := value.Slice()
	if !literal && len(items) == 2 && scmerIsSymbol(items[0], "quote") {
		bindings.prepare(items[1], true)
		return
	}
	if !literal && len(items) > 0 {
		if object, ok := ClusterObjectOf(items[0]); ok {
			key := NewNil()
			if len(items) == 2 && items[1].WithoutSourceInfo().IsString() {
				key = items[1].WithoutSourceInfo()
			}
			demand(object, key)
			items = items[1:]
		}
	}
	for _, item := range items {
		bindings.prepare(item, literal)
	}
}

func ResolveSerializedObjects(value Scmer, resolve func(ClusterReference) (Scmer, error), resolveHandle func(uint64) (Handle, error)) (Scmer, error) {
	values := make(map[string]Scmer)
	return resolveProcedureObjectTree(value, make(map[int64]*serializedCapture), resolveHandle, func(ref ClusterReference) (Scmer, error) {
		key := ref.URI + "/" + ref.Token
		if value, exists := values[key]; exists {
			return value, nil
		}
		if resolve == nil {
			return NewNil(), fmt.Errorf("cluster object requires a reconstruction context")
		}
		value, err := resolve(ref)
		if err == nil {
			values[key] = value
		}
		return value, err
	})
}

// A lambda arriving over the network MUST restore its enclosed runtime values
// exactly once BEFORE compilation/execution. Deferring this to the first mapped
// row leaks owner reservations on empty shards and constructs multiple leases
// for repeated rows. The decoded Go value owns the peer's GC cleanup lease.
type serializedCapture struct {
	value Scmer
	ready bool
}

func serializedValueLiteral(value Scmer) Scmer {
	plain := value.WithoutSourceInfo()
	if plain.IsSlice() || plain.IsFastDict() || plain.IsSymbol() {
		return NewSlice([]Scmer{NewSymbol("quote"), value})
	}
	return value
}
func resolveProcedureObjectTree(value Scmer, captures map[int64]*serializedCapture, resolveHandle func(uint64) (Handle, error), resolve func(ClusterReference) (Scmer, error)) (Scmer, error) {
	plain := value.WithoutSourceInfo()
	if !plain.IsSlice() {
		return value, nil
	}
	items := plain.Slice()
	// Numbered locals have a textual wire spelling. Restore their native
	// representation during decoding, without rerunning the optimizer.
	if len(items) == 2 && scmerIsSymbol(items[0], "var") {
		index := items[1].WithoutSourceInfo()
		if !index.IsInt() || index.Int() < 0 || index.Int() >= 65536 {
			return NewNil(), fmt.Errorf("invalid serialized local variable")
		}
		return NewNthLocalVar(NthLocalVar(index.Int())), nil
	}
	if len(items) == 3 && scmerIsSymbol(items[0], "serialized_value") {
		id := items[1].WithoutSourceInfo()
		if !id.IsInt() || id.Int() <= 0 {
			return NewNil(), fmt.Errorf("invalid serialized capture ID")
		}
		if capture := captures[id.Int()]; capture != nil {
			if !capture.ready {
				return NewNil(), fmt.Errorf("cyclic serialized capture")
			}
			return serializedValueLiteral(capture.value), nil
		}
		capture := &serializedCapture{}
		captures[id.Int()] = capture
		expression, err := resolveProcedureObjectTree(items[2], captures, resolveHandle, resolve)
		if err != nil {
			return NewNil(), err
		}
		capture.value = Eval(expression, &Globalenv)
		capture.ready = true
		return serializedValueLiteral(capture.value), nil
	}
	if len(items) == 2 && scmerIsSymbol(items[0], "cluster_handle") {
		if resolveHandle == nil {
			return NewNil(), fmt.Errorf("handle requires a reconstruction context")
		}
		address := items[1].WithoutSourceInfo()
		if !address.IsInt() || address.Int() <= 0 {
			return NewNil(), fmt.Errorf("invalid handle address")
		}
		handle, err := resolveHandle(uint64(address.Int()))
		return NewAny(handle), err
	}
	if len(items) == 7 && scmerIsSymbol(items[0], "cluster_object") {
		if resolve == nil {
			return NewNil(), fmt.Errorf("cluster object requires a reconstruction context")
		}
		reference := ClusterReference{URI: items[1].String(), Owner: items[2].String(), Incarnation: items[3].String(), Token: items[4].String(), Type: items[5].String(), View: []byte(items[6].String())}
		return resolve(reference)
	}
	if len(items) > 0 && scmerIsSymbol(items[0], "quote") {
		return value, nil
	}
	var result []Scmer
	for i, item := range items {
		resolved, err := resolveProcedureObjectTree(item, captures, resolveHandle, resolve)
		if err != nil {
			return NewNil(), err
		}
		if resolved != item && result == nil {
			result = append([]Scmer(nil), items...)
		}
		if result != nil {
			result[i] = resolved
		}
	}
	if result == nil {
		return value, nil
	}
	return NewSlice(result), nil
}

func initClusterObjects() {
	initHandles()
	Declare(&Globalenv, &Declaration{Name: "serialized_date", Fn: func(a ...Scmer) Scmer {
		return NewDateWithZone(a[0].Int(), int(a[1].Int()))
	}, Type: &TypeDescriptor{Kind: "func", Const: true,
		Params: []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}}, Return: &TypeDescriptor{Kind: "date"}}})
	// Outside wire decoding ordinary Scheme evaluation already constructs the
	// argument before this identity call. Wire import consumes the marker first,
	// restoring one capture per ID before building the executable lambda.
	Declare(&Globalenv, &Declaration{Name: "serialized_value", Fn: func(a ...Scmer) Scmer { return a[1] },
		Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "int", Label: "id"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}}})
	Declare(&Globalenv, &Declaration{Name: "cluster_object", Fn: func(a ...Scmer) Scmer {
		if ClusterObjectResolver == nil {
			panic("cluster object resolver is unavailable")
		}
		ref := ClusterReference{URI: a[0].String(), Owner: a[1].String(), Incarnation: a[2].String(), Token: a[3].String(), Type: a[4].String(), View: []byte(a[5].String())}
		value, err := ClusterObjectResolver(ref)
		if err != nil {
			panic(err)
		}
		return value
	}, Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "string", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}})

}

func init() { RegisterClusterObjectType("session", newClusterSession) }

// SerializedClusterReferences inspects the serializer's inert source envelope.
// It never evaluates source and ignores ordinary quoted program data.
func SerializedClusterReferences(data []byte) []ClusterReference {
	source := string(data)
	if len(data) > 0 && data[0] == procedureWireVersion {
		source = string(data[1:])
	}
	var result []ClusterReference
	var walk func(Scmer)
	walk = func(value Scmer) {
		value = value.WithoutSourceInfo()
		if !value.IsSlice() {
			return
		}
		items := value.Slice()
		if len(items) > 0 && scmerIsSymbol(items[0], "quote") {
			return
		}
		if len(items) == 7 && scmerIsSymbol(items[0], "cluster_object") && items[1].WithoutSourceInfo().IsString() {
			result = append(result, ClusterReference{URI: items[1].String(), Owner: items[2].String(), Incarnation: items[3].String(), Token: items[4].String(), Type: items[5].String(), View: []byte(items[6].String())})
			return
		}
		for _, item := range items {
			walk(item)
		}
	}
	walk(Read("cluster reference manifest", source, false))
	return result
}

// ClusterObjectValue is the checked runtime boundary. Native protocol objects
// may return wire DTOs; Scheme callbacks can receive only explicit Scmer values.
func ClusterObjectValue(object ClusterObject, ctx context.Context, request ClusterRequest) (Scmer, error) {
	result, err := object.HandleClusterRequest(ctx, request)
	if err != nil {
		return NewNil(), err
	}
	if result == nil {
		return NewNil(), nil
	}
	value, ok := result.(Scmer)
	if !ok {
		return NewNil(), fmt.Errorf("cluster response is not a runtime value")
	}
	return value, nil
}

// Invocation binding is object-owned too. In particular a transaction reference
// borrows the current native executor/cancellation scope; no transport switch
// knows transaction fields or installs per-row network probes.
type ClusterInvocationBinder interface{ BindClusterInvocation(Scmer) (Scmer, error) }

func BindClusterInvocation(value, scope Scmer) (Scmer, error) {
	if object, ok := ClusterObjectOf(value); ok {
		if binder, ok := object.(ClusterInvocationBinder); ok {
			return binder.BindClusterInvocation(scope)
		}
	}
	return value, nil
}

// Large immutable dictionary captures use the ordinary object protocol. The
// snapshot belongs to this serialization context; it never observes later
// sender mutations. Runtime callables stay in their invocation-bound path.
type clusterLiteral struct{ source string }
type clusterLiteralCapture struct {
	done   chan struct{}
	object *clusterLiteral
	err    error
}

func (*clusterLiteral) ClusterObjectType() string { return "literal" }
func (v *clusterLiteral) Serialize(ctx *SerializationContext) (Scmer, error) {
	return ctx.ReferenceExpression(v)
}
func (v *clusterLiteral) HandleClusterRequest(ctx context.Context, request ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch request.Method {
	case "view":
		return NewNil(), nil
	case "source":
		return NewString(v.source), nil
	}
	return nil, fmt.Errorf("unsupported immutable capture request")
}
func plainClusterLiteral(value Scmer, depth int) bool {
	if depth > 64 {
		return false
	}
	switch value.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagCString, tagBString, tagLZ4, tagSymbol:
		return true
	case tagSlice:
		for _, item := range value.Slice() {
			if !plainClusterLiteral(item, depth+1) {
				return false
			}
		}
		return true
	case tagFastDict:
		if dict := value.FastDict(); dict != nil {
			for _, item := range dict.Pairs {
				if !plainClusterLiteral(item, depth+1) {
					return false
				}
			}
		}
		return true
	}
	return false
}
func (ctx *SerializationContext) literal(value Scmer) (*clusterLiteral, error) {
	if ctx.root != nil {
		ctx = ctx.root
	}
	ctx.mu.Lock()
	if ctx.literals == nil {
		ctx.literals = make(map[Scmer]*clusterLiteralCapture)
	}
	capture := ctx.literals[value]
	first := capture == nil
	if first {
		capture = &clusterLiteralCapture{done: make(chan struct{})}
		ctx.literals[value] = capture
	}
	ctx.mu.Unlock()
	if first {
		func() {
			defer close(capture.done)
			defer func() {
				if failure := recover(); failure != nil {
					capture.err = fmt.Errorf("immutable capture: %v", failure)
				}
			}()
			if !plainClusterLiteral(value, 0) {
				return
			}
			var source bytes.Buffer
			Serialize(&source, value, &Globalenv)
			// The source reply remains one bounded message even after textual
			// string escaping in both the Scmer value and its result envelope.
			if source.Len() <= 48*1024 {
				capture.object = &clusterLiteral{source.String()}
			}
		}()
	} else {
		<-capture.done
	}
	return capture.object, capture.err
}

type clusterLiteralImport struct {
	mu      sync.Mutex
	value   weak.Pointer[FastDict]
	loading *clusterLiteralLoad
}
type clusterLiteralLoad struct {
	ready   chan struct{}
	value   Scmer
	failure any
}

func importClusterLiteral(handle ClusterHandle) (value Scmer) {
	if handle.Lifetime == nil {
		panic("immutable capture requires a peer lifetime")
	}
	state := handle.Lifetime.LocalState(func() any { return new(clusterLiteralImport) }).(*clusterLiteralImport)
	state.mu.Lock()
	if dict := state.value.Value(); dict != nil {
		state.mu.Unlock()
		return NewFastDict(dict)
	}
	if loading := state.loading; loading != nil {
		state.mu.Unlock()
		// This lease owns the accepted materialization. Join it; a cancelled
		// waiter must not abandon shared work needed by another invocation.
		<-loading.ready
		if loading.failure != nil {
			panic(loading.failure)
		}
		return loading.value
	}
	loading := &clusterLiteralLoad{ready: make(chan struct{})}
	state.loading = loading
	state.mu.Unlock()
	defer func() {
		failure := recover()
		state.mu.Lock()
		if failure == nil {
			state.value = weak.Make(value.FastDict())
		}
		loading.value, loading.failure = value, failure
		state.loading = nil // no strong value reachable from its own cleanup argument
		close(loading.ready)
		state.mu.Unlock()
		if failure != nil {
			panic(failure)
		}
	}()
	source, err := handle.Call(context.Background(), ClusterRequest{Method: "source"})
	if err != nil {
		panic(err)
	}
	value = Eval(Read("immutable cluster capture", source.String(), false), &Globalenv)
	if !value.IsFastDict() {
		panic("invalid immutable dictionary capture")
	}
	runtime.AddCleanup(value.FastDict(), func(retained ClusterLifetime) { runtime.KeepAlive(retained) }, handle.Lifetime)
	return value
}
func init() { RegisterClusterObjectType("literal", importClusterLiteral) }
