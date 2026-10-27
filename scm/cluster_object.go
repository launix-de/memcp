/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "context"
import "unsafe"
import "encoding/json"

// ClusterObject is the common owner endpoint for runtime objects. Implementations
// choose their read view and mutation policy; transport never inspects their Go
// representation. All requests execute outside generated row loops.
type ClusterObject interface {
	ClusterObjectType() string
	HandleClusterRequest(context.Context, ClusterRequest) (any, error)
}

type ClusterRequest struct {
	Method    string
	Arguments []Scmer
	Payload   any `json:"-"` // private decoded protocol payload, never runtime serialization
}

// ClusterReference names a process-incarnation-bound owner registration. Each
// export has its own token: releasing an old handle cannot revoke a newer one.
// View is an immutable, demanded read view prepared by the object owner.
type ClusterReference struct {
	URI         string          `json:"uri"`
	Owner       string          `json:"owner"`
	Incarnation string          `json:"incarnation"`
	Token       string          `json:"token"`
	Type        string          `json:"type"`
	View        json.RawMessage `json:"view"`
}

// ClusterHandle is owned by the imported value, not by a transport registry.
// Keeping a callable/reader alive therefore keeps its remote reference alive.
// The transport attaches GC cleanup to its private implementation of Call.
type ClusterHandle struct {
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

// ClusterBindings belongs to one statement. Demands are collected from native
// callback captures, for every object type alike. Objects produce their own
// immutable view once; each target peer gets a distinct registered handle.
type ClusterBindings struct {
	Export  func(ClusterObject, Scmer) (ClusterReference, error)
	root    *ClusterBindings
	mu      sync.Mutex
	views   map[ClusterObject]*clusterViewCapture
	demands map[ClusterObject]map[Scmer]struct{}
}

type clusterViewCapture struct {
	done chan struct{}
	view Scmer
	err  error
}

func (bindings *ClusterBindings) Reference(object ClusterObject) (ClusterReference, error) {
	if bindings == nil || bindings.Export == nil {
		return ClusterReference{}, fmt.Errorf("cluster object requires an export context")
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

// ProcedureDecode is invocation-owned. Reconstruction is completed before
// Optimize/Eval builds the callback, never lazily during its row execution.
type ProcedureDecode struct {
	QueryContext Scmer
	Resolve      func(ClusterReference) (Scmer, error)
}

// ClusterReadProjection lets the object restrict its prepared read view to
// demanded arguments without teaching serialization about sessions/cachemaps.
type ClusterReadProjection interface {
	ProjectClusterView(Scmer, []Scmer) (Scmer, error)
}

func (bindings *ClusterBindings) WithExport(export func(ClusterObject, Scmer) (ClusterReference, error)) *ClusterBindings {
	if bindings.root != nil {
		bindings = bindings.root
	}
	return &ClusterBindings{Export: export, root: bindings}
}

func (bindings *ClusterBindings) prepare(value Scmer) {
	if bindings.root != nil {
		bindings = bindings.root
	}
	value = value.WithoutSourceInfo()
	if value.IsProc() {
		bindings.prepare(value.Proc().Body)
		return
	}
	if !value.IsSlice() {
		return
	}
	items := value.Slice()
	if len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return
	}
	if object, ok := ClusterObjectOf(items[0]); ok {
		bindings.mu.Lock()
		if bindings.demands == nil {
			bindings.demands = make(map[ClusterObject]map[Scmer]struct{})
		}
		if bindings.demands[object] == nil {
			bindings.demands[object] = make(map[Scmer]struct{})
		}
		key := NewNil()
		if len(items) == 2 && items[1].WithoutSourceInfo().IsString() {
			key = items[1].WithoutSourceInfo()
		}
		bindings.demands[object][key] = struct{}{}
		bindings.mu.Unlock()
	}
	for _, item := range items {
		bindings.prepare(item)
	}
}

func resolveProcedureObjects(value Scmer, resolve func(ClusterReference) (Scmer, error)) (Scmer, error) {
	values := make(map[string]Scmer)
	return resolveProcedureObjectTree(value, func(ref ClusterReference) (Scmer, error) {
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
func resolveProcedureObjectTree(value Scmer, resolve func(ClusterReference) (Scmer, error)) (Scmer, error) {
	plain := value.WithoutSourceInfo()
	if !plain.IsSlice() {
		return value, nil
	}
	items := plain.Slice()
	if len(items) == 2 && scmerIsSymbol(items[0], "cluster_object") {
		if resolve == nil {
			return NewNil(), fmt.Errorf("cluster object requires a reconstruction context")
		}
		var reference ClusterReference
		if !items[1].WithoutSourceInfo().IsString() {
			return NewNil(), fmt.Errorf("invalid cluster object reference")
		}
		if err := json.Unmarshal([]byte(items[1].String()), &reference); err != nil {
			return NewNil(), err
		}
		return resolve(reference)
	}
	if len(items) > 0 && scmerIsSymbol(items[0], "quote") {
		return value, nil
	}
	result := make([]Scmer, len(items))
	for i, item := range items {
		var err error
		result[i], err = resolveProcedureObjectTree(item, resolve)
		if err != nil {
			return NewNil(), err
		}
	}
	return NewSlice(result), nil
}

func initClusterObjects() {
	Declare(&Globalenv, &Declaration{Name: "cluster_object", Fn: func(a ...Scmer) Scmer {
		if ClusterObjectResolver == nil {
			panic("cluster object resolver is unavailable")
		}
		var ref ClusterReference
		if err := json.Unmarshal([]byte(a[0].String()), &ref); err != nil {
			panic(err)
		}
		value, err := ClusterObjectResolver(ref)
		if err != nil {
			panic(err)
		}
		return value
	}, Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "string", Label: "reference"}}, Return: &TypeDescriptor{Kind: "any"}}})

}

func init() { RegisterClusterObjectType("session", newClusterSession) }

// ProcedureClusterReferences inspects the serializer's inert source envelope.
// It never evaluates source and ignores ordinary quoted program data.
func ProcedureClusterReferences(data []byte) []ClusterReference {
	var envelope struct {
		Source string `json:"source"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return nil
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
		if len(items) == 2 && scmerIsSymbol(items[0], "cluster_object") && items[1].WithoutSourceInfo().IsString() {
			var ref ClusterReference
			if json.Unmarshal([]byte(items[1].String()), &ref) == nil {
				result = append(result, ref)
			}
			return
		}
		for _, item := range items {
			walk(item)
		}
	}
	walk(Read("cluster reference manifest", envelope.Source))
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
