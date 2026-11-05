/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "weak"
import "bytes"
import "runtime"
import "strings"
import "sync/atomic"
import "github.com/google/uuid"

// SerializationPeer is the peer's knowledge of immutable values and code. The
// embedding supplies queue publication and channel registration, not RPCs.
type SerializationPeer struct {
	Declare       func(string, []byte, bool) error
	Send          func(string, []byte) error
	Register      func(string, func([]byte) error) func()
	Prepare       func(Scmer) (Scmer, error)
	AfterReceived func(func())
	known         sync.Map // code/value identity -> exportedClusterValue
	exported      sync.Map // UUID -> exportedClusterValue
	received      sync.Map // UUID -> receivedClusterValue
	closed        atomic.Bool
}

const valueRetire byte = 0
const valueRetireAck byte = 1
const valueStopped uint64 = 1 << 63

type exportedClusterValue struct {
	peer       *SerializationPeer
	key        any
	id         string
	ready      chan struct{}
	err        error
	activity   atomic.Uint64
	unregister func()
}

func (value *exportedClusterValue) hold() bool {
	for {
		old := value.activity.Load()
		if old&valueStopped != 0 {
			return false
		}
		if value.activity.CompareAndSwap(old, old+1) {
			return true
		}
	}
}
func (value *exportedClusterValue) release() {
	if value.activity.Add(^uint64(0)) == valueStopped {
		value.retired()
	}
}
func (value *exportedClusterValue) retired() {
	value.unregister()
	_ = value.peer.Send(value.id, []byte{valueRetireAck})
}
func (value *exportedClusterValue) receive(data []byte) error {
	if len(data) != 1 || data[0] != valueRetire {
		return fmt.Errorf("invalid value retirement")
	}
	value.peer.known.CompareAndDelete(value.key, value)
	value.peer.exported.CompareAndDelete(value.id, value)
	for {
		old := value.activity.Load()
		if old&valueStopped != 0 {
			return nil
		}
		if value.activity.CompareAndSwap(old, old|valueStopped) {
			if old == 0 {
				value.retired()
			}
			return nil
		}
	}
}

type serializationPublication struct {
	lifetime *publicationLifetime
	cleanup  runtime.Cleanup
}
type publicationLifetime struct {
	contexts map[*lambdaMetadata]*lambdaContextSnapshot
	mu       sync.Mutex
	closed   bool
	values   map[*exportedClusterValue]struct{}
}

func newSerializationPublication() *serializationPublication {
	p := &serializationPublication{lifetime: &publicationLifetime{values: make(map[*exportedClusterValue]struct{}), contexts: make(map[*lambdaMetadata]*lambdaContextSnapshot)}}
	p.cleanup = runtime.AddCleanup(p, (*publicationLifetime).close, p.lifetime)
	return p
}
func (l *publicationLifetime) close() {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	l.closed = true
	values := l.values
	l.values = nil
	l.contexts = nil
	l.mu.Unlock()
	for value := range values {
		value.release()
	}
}
func (p *serializationPublication) hold(value *exportedClusterValue) bool {
	p.lifetime.mu.Lock()
	defer p.lifetime.mu.Unlock()
	if p.lifetime.closed {
		panic("serialization publication already completed")
	}
	if _, ok := p.lifetime.values[value]; ok {
		return true
	}
	if !value.hold() {
		return false
	}
	p.lifetime.values[value] = struct{}{}
	return true
}

// Queued releases preparation reservations only after all referring bytes have
// entered the FIFO, or after encoding was abandoned. It never waits for a peer.
func (ctx *SerializationContext) Queued() {
	if ctx != nil && ctx.publication != nil {
		ctx.publication.lifetime.close()
		ctx.publication.cleanup.Stop()
	}
}
func (ctx *SerializationContext) WithPeer(peer *SerializationPeer) *SerializationContext {
	copy := ctx.WithExport(ctx.Export)
	copy.Peer = peer
	copy.publication = newSerializationPublication()
	return copy
}
func (ctx *SerializationContext) cachedValue(value Scmer) (Scmer, bool, error) {
	if ctx == nil || ctx.Peer == nil {
		return NewNil(), false, nil
	}
	peer := ctx.Peer
	if peer.closed.Load() {
		return NewNil(), false, fmt.Errorf("serialization peer closed")
	}
	var key any
	var source Scmer
	var contexts []Scmer
	lambda := false
	if proc := lambdaProcedure(value); proc != nil {
		snapshot := ctx.lambdaContext(proc)
		key = snapshot.plan
		source = snapshot.plan.factory
		contexts = snapshot.values
		lambda = true
	} else {
		plain := value.WithoutSourceInfo()
		if plain.ptr == nil {
			return NewNil(), false, nil
		}
		if plain.IsString() && len(plain.String()) < 128 {
			return NewNil(), false, nil
		}
		if !plain.IsSlice() && !plain.IsFastDict() && !plain.IsString() {
			return NewNil(), false, nil
		}
		if plain.IsSlice() && len(plain.Slice()) < 32 {
			return NewNil(), false, nil
		}
		if plain.IsFastDict() && len(plain.FastDict().Pairs) < 128 {
			return NewNil(), false, nil
		}
		key = plain
		if plain.IsSlice() || plain.IsFastDict() {
			// Optimizer-owned builders may reuse a container after this invocation.
			// Freeze once across its shards, without retaining the invocation/Tx.
			root := ctx
			if root.root != nil {
				root = root.root
			}
			root.mu.Lock()
			if root.valueSnapshot == nil {
				root.valueSnapshot = new(serializationValueSnapshot)
			}
			key = cachedSnapshotValue{plain, root.valueSnapshot}
			root.mu.Unlock()
		}
		if _, known := peer.known.Load(key); !known && !plainClusterLiteral(plain, 0) {
			return NewNil(), false, nil
		}
		source = serializedValueLiteral(plain)
	}
	if ctx.publication == nil {
		ctx.mu.Lock()
		if ctx.publication == nil {
			ctx.publication = newSerializationPublication()
		}
		ctx.mu.Unlock()
	}
	for {
		entry, loaded := peer.known.Load(key)
		if !loaded {
			candidate := &exportedClusterValue{peer: peer, key: key, id: uuid.NewString(), ready: make(chan struct{})}
			entry, loaded = peer.known.LoadOrStore(key, candidate)
		}
		record := entry.(*exportedClusterValue)
		if !loaded {
			if !ctx.publication.hold(record) {
				continue
			}
			func() {
				defer close(record.ready)
				defer func() {
					if failure := recover(); failure != nil {
						record.err = fmt.Errorf("cluster value declaration: %v", failure)
					}
				}()
				record.unregister = peer.Register(record.id, record.receive)
				peer.exported.Store(record.id, record)
				var text bytes.Buffer
				if err := validateProcedureWire(source, make(map[Symbol]struct{}), &Globalenv, false, 0); err != nil {
					record.err = err
					return
				}
				Serialize(&text, source, &Globalenv, new(SerializationContext))
				record.err = peer.Declare(record.id, text.Bytes(), lambda)
			}()
		}
		<-record.ready
		if record.err != nil {
			peer.known.CompareAndDelete(key, record)
			peer.exported.CompareAndDelete(record.id, record)
			if record.unregister != nil {
				record.unregister()
			}
			return NewNil(), false, record.err
		}
		if !ctx.publication.hold(record) {
			continue
		}
		expression := NewSlice([]Scmer{NewSymbol("cluster_value"), NewString(record.id)})
		if lambda {
			args := make([]Scmer, 0, len(contexts)+2)
			args = append(args, NewSymbol("lambda_rebind"), expression)
			for _, value := range contexts {
				args = append(args, serializedValueLiteral(value))
			}
			expression = NewSlice(args)
		}
		return expression, true, nil
	}
}

type serializationValueSnapshot struct{ identity byte }
type cachedSnapshotValue struct {
	value    Scmer
	snapshot *serializationValueSnapshot
}

type receivedClusterValue struct {
	peer         weak.Pointer[SerializationPeer]
	id           string
	source       []byte // retirement window only; never roots the parsed/compiled value
	lambda       bool
	prepared     bool
	mu           sync.Mutex
	pointer      weak.Pointer[byte]
	aux          uint64
	retiring     bool
	acknowledged bool
	released     atomic.Bool
	unregister   func()
}

var clusterValues sync.Map // UUID -> receivedClusterValue, weak decoded value
func (value *receivedClusterValue) decode() (Scmer, error) {
	tokens := tokenize("cluster declaration", strings.Clone(string(value.source)), false)
	expression := readFrom(&tokens)
	if len(tokens) != 0 {
		return NewNil(), fmt.Errorf("cluster declaration contains trailing data")
	}
	peer := value.peer.Value()
	if value.lambda && !value.prepared && peer != nil && peer.Prepare != nil {
		var err error
		expression, err = peer.Prepare(expression)
		if err != nil {
			return NewNil(), err
		}
	}
	if err := validateProcedureWire(expression, make(map[Symbol]struct{}), &Globalenv, false, 0); err != nil {
		return NewNil(), err
	}
	if !value.prepared {
		// Keep prepared source self-contained; a disconnected peer/its callbacks
		// must not be rooted by a weak cache record or needed for local revival.
		var prepared bytes.Buffer
		Serialize(&prepared, expression, &Globalenv, new(SerializationContext))
		value.source = bytes.Clone(prepared.Bytes())
		value.prepared = true
	}
	expression, err := ResolveSerializedObjects(expression, nil, nil)
	if err != nil {
		return NewNil(), err
	}
	var result Scmer
	if value.lambda {
		result = compileLambdaTemplate(expression)
	} else {
		result = Eval(expression, &Globalenv)
	}
	if result.ptr == nil || (!value.lambda && (!plainClusterLiteral(result, 0) || (!result.IsSlice() && !result.IsFastDict() && !result.IsString()))) {
		return NewNil(), fmt.Errorf("cluster declaration needs an immutable GC-tracked value")
	}
	value.pointer = weak.Make(result.ptr)
	value.aux = result.aux
	identity := weak.Make(value)
	generation := value.pointer
	runtime.AddCleanup(result.ptr, func(retired struct {
		record     weak.Pointer[receivedClusterValue]
		generation weak.Pointer[byte]
	}) {
		if value := retired.record.Value(); value != nil {
			value.retire(retired.generation)
		}
	}, struct {
		record     weak.Pointer[receivedClusterValue]
		generation weak.Pointer[byte]
	}{identity, generation})
	return result, nil
}
func (peer *SerializationPeer) Receive(id string, source []byte, lambda bool) (err error) {
	var value *receivedClusterValue
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("cluster declaration: %v", failure)
		}
		if err != nil && value != nil {
			clusterValues.CompareAndDelete(id, value)
			peer.received.CompareAndDelete(id, value)
			if value.unregister != nil {
				value.unregister()
			}
		}
	}()
	if _, exists := clusterValues.Load(id); exists {
		return fmt.Errorf("duplicate cluster value UUID")
	}
	if peer.closed.Load() {
		return fmt.Errorf("serialization peer closed")
	}
	if _, err := uuid.Parse(id); err != nil {
		return err
	}
	value = &receivedClusterValue{peer: weak.Make(peer), id: id, source: bytes.Clone(source), lambda: lambda}
	value.unregister = peer.Register(id, func(data []byte) error {
		if len(data) != 1 || data[0] != valueRetireAck {
			return fmt.Errorf("invalid value retirement acknowledgement")
		}
		release := value.acknowledge
		if peer.AfterReceived != nil {
			peer.AfterReceived(release)
		} else {
			release()
		}
		return nil
	})
	result, err := func() (Scmer, error) {
		value.mu.Lock()
		defer value.mu.Unlock()
		return value.decode()
	}()
	if err != nil {
		value.unregister()
		return err
	}
	peer.received.Store(id, value)
	if _, exists := clusterValues.LoadOrStore(id, value); exists {
		peer.received.Delete(id)
		value.unregister()
		return fmt.Errorf("duplicate cluster value UUID")
	}
	runtime.KeepAlive(result)
	return nil
}
func (value *receivedClusterValue) release() {
	if !value.released.CompareAndSwap(false, true) {
		return
	}
	clusterValues.CompareAndDelete(value.id, value)
	if peer := value.peer.Value(); peer != nil {
		peer.received.CompareAndDelete(value.id, value)
	}
	value.unregister()
}
func (value *receivedClusterValue) acknowledge() {
	value.mu.Lock()
	value.acknowledged = true
	gone := value.pointer.Value() == nil
	value.mu.Unlock()
	if gone {
		value.release()
	}
}
func (value *receivedClusterValue) retire(generation weak.Pointer[byte]) {
	value.mu.Lock()
	if value.pointer != generation {
		value.mu.Unlock()
		return
	}
	peer := value.peer.Value()
	if value.acknowledged || peer == nil || peer.closed.Load() {
		value.mu.Unlock()
		value.release()
		return
	}
	if value.retiring {
		value.mu.Unlock()
		return
	}
	value.retiring = true
	value.mu.Unlock()
	_ = peer.Send(value.id, []byte{valueRetire})
}
func ClusterValue(id string) Scmer {
	entry, ok := clusterValues.Load(id)
	if !ok {
		panic("unknown cluster value: " + id)
	}
	value := entry.(*receivedClusterValue)
	value.mu.Lock()
	defer value.mu.Unlock()
	if pointer := value.pointer.Value(); pointer != nil {
		return Scmer{pointer, value.aux}
	}
	// A GC retirement can cross an already prepared task. Its local source is
	// retained until the sender's ordered barrier; no pull or network wait.
	result, err := value.decode()
	if err != nil {
		panic(err)
	}
	return result
}
func (peer *SerializationPeer) Close() {
	if !peer.closed.CompareAndSwap(false, true) {
		return
	}
	peer.exported.Range(func(key, value any) bool {
		record := value.(*exportedClusterValue)
		peer.known.CompareAndDelete(record.key, record)
		record.unregister()
		peer.exported.Delete(key)
		return true
	})
	peer.received.Range(func(_, entry any) bool {
		value := entry.(*receivedClusterValue)
		value.unregister()
		// Decoded values and retained view/result buffers survive reconnect. Their
		// last local Go owner removes the weak UUID entry without contacting a dead edge.
		value.acknowledge()
		return true
	})
}
func initClusterValues() {
	Declare(&Globalenv, &Declaration{Name: "cluster_value", Fn: func(args ...Scmer) Scmer { return ClusterValue(args[0].String()) }, Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "string"}}, Return: &TypeDescriptor{Kind: "any"}}})
}

// A publication snapshots each closure once. Preparation and printing must not
// evaluate imperative outer capture expressions twice.
type lambdaContextSnapshot struct {
	plan     *lambdaPlan
	values   []Scmer
	ready    chan struct{}
	prepared atomic.Bool
}

func (ctx *SerializationContext) lambdaContext(proc *Proc) *lambdaContextSnapshot {
	if ctx.publication == nil {
		ctx.mu.Lock()
		if ctx.publication == nil {
			ctx.publication = newSerializationPublication()
		}
		ctx.mu.Unlock()
	}
	key := proc.serializationMetadata()
	state := ctx.publication.lifetime
	state.mu.Lock()
	if state.closed {
		state.mu.Unlock()
		panic("serialization publication already completed")
	}
	snapshot := state.contexts[key]
	first := snapshot == nil
	if first {
		snapshot = &lambdaContextSnapshot{ready: make(chan struct{})}
		state.contexts[key] = snapshot
	}
	state.mu.Unlock()
	if first {
		func() {
			defer close(snapshot.ready)
			snapshot.plan = procedureLambdaPlan(proc)
			snapshot.values = snapshot.plan.context(proc)
		}()
	} else {
		<-snapshot.ready
	}
	return snapshot
}

// RetainClusterValues ties an owned receive buffer to the cached values named
// by its inert serialization source. Most data has no references and only pays
// a byte search. No constructors or query code execute during the walk.
func RetainClusterValues(source []byte) (err error) {
	if !bytes.Contains(source, []byte("(cluster_value ")) {
		return nil
	}
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("cached receive buffer: %v", failure)
		}
	}()
	var roots []Scmer
	seen := map[string]bool{}
	var walk func(Scmer, int)
	walk = func(value Scmer, depth int) {
		if depth > 128 {
			panic("cached receive buffer nesting limit")
		}
		value = value.WithoutSourceInfo()
		if !value.IsSlice() {
			return
		}
		items := value.Slice()
		if len(items) == 0 {
			return
		}
		if scmerIsSymbol(items[0], "quote") {
			return
		}
		if len(items) == 2 && scmerIsSymbol(items[0], "cluster_value") && items[1].IsString() {
			id := items[1].String()
			if !seen[id] {
				roots = append(roots, ClusterValue(id))
				seen[id] = true
			}
			return
		}
		if len(items) == 7 && scmerIsSymbol(items[0], "cluster_object") && items[6].IsString() {
			nested := items[6].String()
			if strings.Contains(nested, "(cluster_value ") {
				walk(Read("cached object view", nested, false), depth+1)
			}
		}
		for _, item := range items {
			walk(item, depth+1)
		}
	}
	walk(Read("cached receive buffer", string(source), false), 0)
	if len(roots) > 0 {
		runtime.AddCleanup(&source[0], func(values []Scmer) { runtime.KeepAlive(values) }, roots)
	}
	runtime.KeepAlive(source)
	return nil
}
