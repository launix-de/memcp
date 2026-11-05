/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "weak"
import "unsafe"
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
	contexts map[Scmer]*lambdaContextSnapshot
	mu       sync.Mutex
	closed   bool
	values   map[*exportedClusterValue]struct{}
}

func newSerializationPublication() *serializationPublication {
	p := &serializationPublication{lifetime: &publicationLifetime{values: make(map[*exportedClusterValue]struct{}), contexts: make(map[Scmer]*lambdaContextSnapshot)}}
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

type receivedClusterValue struct {
	peer       *SerializationPeer
	id         string
	source     []byte // retirement window only; never roots the parsed/compiled value
	lambda     bool
	mu         sync.Mutex
	pointer    weak.Pointer[byte]
	aux        uint64
	retiring   bool
	unregister func()
}

var clusterValues sync.Map // UUID -> receivedClusterValue, weak decoded value
func (value *receivedClusterValue) decode() (Scmer, error) {
	tokens := tokenize("cluster declaration", strings.Clone(string(value.source)), false)
	expression := readFrom(&tokens)
	if len(tokens) != 0 {
		return NewNil(), fmt.Errorf("cluster declaration contains trailing data")
	}
	if value.lambda && value.peer.Prepare != nil {
		var err error
		expression, err = value.peer.Prepare(expression)
		if err != nil {
			return NewNil(), err
		}
	}
	if err := validateProcedureWire(expression, make(map[Symbol]struct{}), &Globalenv, false, 0); err != nil {
		return NewNil(), err
	}
	expression, err := ResolveSerializedObjects(expression, nil, nil)
	if err != nil {
		return NewNil(), err
	}
	result := Eval(expression, &Globalenv)
	if value.lambda {
		if !result.IsProc() {
			return NewNil(), fmt.Errorf("lambda declaration did not produce a lambda")
		}
		result = CompileJIT(result, true)
		result.Proc().lambdaFactory = true
	}
	if result.ptr == nil || (!value.lambda && (!plainClusterLiteral(result, 0) || (!result.IsSlice() && !result.IsFastDict() && !result.IsString()))) {
		return NewNil(), fmt.Errorf("cluster declaration needs an immutable GC-tracked value")
	}
	value.pointer = weak.Make(result.ptr)
	value.aux = result.aux
	peer, id := value.peer, value.id
	runtime.AddCleanup(result.ptr, func(retired struct {
		peer *SerializationPeer
		id   string
	}) {
		retired.peer.retireValue(retired.id)
	}, struct {
		peer *SerializationPeer
		id   string
	}{peer, id})
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
	value = &receivedClusterValue{peer: peer, id: id, source: bytes.Clone(source), lambda: lambda}
	value.unregister = peer.Register(id, func(data []byte) error {
		if len(data) != 1 || data[0] != valueRetireAck {
			return fmt.Errorf("invalid value retirement acknowledgement")
		}
		release := func() {
			clusterValues.CompareAndDelete(id, value)
			peer.received.CompareAndDelete(id, value)
			value.unregister()
		}
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
func (peer *SerializationPeer) retireValue(id string) {
	entry, ok := peer.received.Load(id)
	if !ok {
		return
	}
	value := entry.(*receivedClusterValue)
	value.mu.Lock()
	if value.retiring {
		value.mu.Unlock()
		return
	}
	value.retiring = true
	value.mu.Unlock()
	_ = peer.Send(id, []byte{valueRetire})
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
	peer.received.Range(func(key, entry any) bool {
		value := entry.(*receivedClusterValue)
		clusterValues.CompareAndDelete(key, value)
		value.unregister()
		peer.received.Delete(key)
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
	key := Scmer{(*byte)(unsafe.Pointer(proc)), makeAux(tagProc, 0)}
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
