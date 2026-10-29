/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

package scm

import "sync"
import "time"
import "unsafe"
import "context"
import "container/heap"
import "runtime"
import "sync/atomic"

// cachedMemStats provides a cached version of runtime.ReadMemStats. Cache
// ownership accounting must not be added to this snapshot: those allocations
// are already part of the Go heap, and doing so would double-count them.
var (
	cachedStats     runtime.MemStats
	cachedStatsTime time.Time
	cachedStatsMu   sync.Mutex
)

// CachedMemStats returns a runtime snapshot cached for one minute. Operational
// endpoints use process RSS and CacheManager ownership counters for live data;
// this function deliberately avoids expensive runtime scans on every request.
func CachedMemStats() runtime.MemStats {
	cachedStatsMu.Lock()
	defer cachedStatsMu.Unlock()
	if time.Since(cachedStatsTime) > time.Minute {
		runtime.ReadMemStats(&cachedStats)
		cachedStatsTime = time.Now()
	}
	return cachedStats
}

/* promise: single-value cell (thread-safe via CAS spin-lock on cells[1].aux) */

var (
	promiseLockSentinel = makeAux(tagBool, 0) // tagBool|false = Lock
	promiseFailedAux    = makeAux(tagBool, 2) // tagBool|2 = Failed (NOT tagBool|0!)
	promisePendingAux   = makeAux(tagNil, 0)  // pending (nil)
	promiseResolvedAux  = makeAux(tagBool, 1) // fulfilled (true)
)

// promiseLock spins until it acquires the lock. Returns the previous state aux.
func promiseLock(cells *[2]Scmer) uint64 {
	statePtr := (*uint64)(unsafe.Pointer(&cells[1].aux))
	for {
		old := atomic.LoadUint64(statePtr)
		if old == promiseLockSentinel {
			runtime.Gosched()
			continue
		}
		if atomic.CompareAndSwapUint64(statePtr, old, promiseLockSentinel) {
			return old
		}
		runtime.Gosched()
	}
}

// promiseUnlock releases the lock by storing the new state.
func promiseUnlock(cells *[2]Scmer, newStateAux uint64) {
	atomic.StoreUint64(&cells[1].aux, newStateAux)
}

// Fresh promises use a dedicated [2]Scmer backing; (newpromise list) reuses
// an existing >=2-element slice with zero extra allocation.
func NewPromise(a ...Scmer) Scmer {
	var cells []Scmer
	if len(a) == 0 {
		cells = make([]Scmer, 2)
		cells[1] = NewNil()
		return Scmer{(*byte)(unsafe.Pointer(&cells[0])), makeAux(tagPromise, 0)}
	}
	cells = a[0].Slice()
	if len(cells) < 2 {
		panic("newpromise: list backing requires at least 2 elements")
	}
	cells[1] = NewNil()
	return Scmer{(*byte)(unsafe.Pointer(&cells[0])), makeAux(tagPromise, 1)}
}

// ApplyPromise dispatches a tagPromise call. Called from ApplyEx.
func ApplyPromise(p Scmer, args []Scmer) Scmer {
	cells := (*[2]Scmer)(unsafe.Pointer(p.ptr))
	if len(args) == 0 {
		panic("promise: at least 1 argument required")
	}
	key := args[0].String()
	switch len(args) {
	case 1:
		switch key {
		case "value":
			old := promiseLock(cells)
			if old == promisePendingAux {
				promiseUnlock(cells, old)
				return NewNil()
			}
			val := cells[0]
			promiseUnlock(cells, old)
			return val
		case "state":
			old := promiseLock(cells)
			promiseUnlock(cells, old)
			switch old {
			case promisePendingAux:
				return NewNil()
			case promiseResolvedAux:
				return NewBool(true)
			default:
				return NewBool(false)
			}
		case "fail":
			promiseLock(cells)
			cells[0] = NewNil()
			promiseUnlock(cells, promiseFailedAux)
			return NewBool(false)
		default:
			panic("promise: unknown operation: " + key)
		}
	case 2:
		if key == "value" {
			promiseLock(cells)
			cells[0] = args[1]
			promiseUnlock(cells, promiseResolvedAux)
			return args[1]
		}
		if key == "once" {
			old := promiseLock(cells)
			if old != promisePendingAux {
				promiseUnlock(cells, old)
				panic("promise already fulfilled/failed")
			}
			cells[0] = args[1]
			promiseUnlock(cells, promiseResolvedAux)
			return args[1]
		}
		if key == "fail" {
			promiseLock(cells)
			cells[0] = args[1]
			promiseUnlock(cells, promiseFailedAux)
			return args[1]
		}
		panic("promise: unknown operation: " + key)
	case 3:
		if key == "once" {
			old := promiseLock(cells)
			if old != promisePendingAux {
				promiseUnlock(cells, old)
				panic(args[2].String())
			}
			cells[0] = args[1]
			promiseUnlock(cells, promiseResolvedAux)
			return args[1]
		}
		panic("promise: unknown operation: " + key)
	default:
		panic("promise: too many arguments")
	}
}

/* threadsafe session storage */

type session struct {
	call           func(...Scmer) Scmer // immutable native accessor, initialized before publication
	Mu             sync.RWMutex
	Map            map[string]Scmer
	Handles        map[Scmer]Scmer
	ScopedValues   map[sessionScopedKey]Scmer
	ScopedFlights  map[sessionScopedKey]*sessionFlight
	ScopedCleanup  map[Scmer]bool
	ScopedCanceled map[Scmer]bool
}

type sessionScopedKey struct {
	scope Scmer
	key   string
}

type sessionFlight struct {
	done       chan struct{}
	value      Scmer
	panicValue any
	failed     bool
}

func sessionHasScopedFlight(sess *session, scope Scmer) bool {
	for key := range sess.ScopedFlights {
		if key.scope == scope {
			return true
		}
	}
	return false
}

func executionContextFrom(value Scmer) context.Context {
	ss, seq := querySessionState(value)
	if ss == nil || seq == 0 {
		return context.Background()
	}
	return ss.QueryContext(seq)
}

func sessionEnsureScopedCleanup(sess *session, scope Scmer, ctx context.Context) {
	if sess.ScopedCleanup[scope] {
		return
	}
	if ctx == nil || ctx.Done() == nil {
		return
	}
	sess.ScopedCleanup[scope] = true
	context.AfterFunc(ctx, func() {
		sess.Mu.Lock()
		defer sess.Mu.Unlock()
		sess.ScopedCanceled[scope] = true
		for key := range sess.ScopedValues {
			if key.scope == scope {
				delete(sess.ScopedValues, key)
			}
		}
		if !sessionHasScopedFlight(sess, scope) {
			delete(sess.ScopedCleanup, scope)
			delete(sess.ScopedCanceled, scope)
		}
	})
}

func sessionGetOrComputeScoped(sess *session, scope Scmer, key string, tx Scmer, producer Scmer, passTx bool) Scmer {
	computeKey := sessionScopedKey{scope: scope, key: key}
	sess.Mu.Lock()
	if value, ok := sess.ScopedValues[computeKey]; ok {
		sess.Mu.Unlock()
		return value
	}
	if sess.ScopedValues == nil {
		sess.ScopedValues = make(map[sessionScopedKey]Scmer)
		sess.ScopedFlights = make(map[sessionScopedKey]*sessionFlight)
		sess.ScopedCleanup = make(map[Scmer]bool)
		sess.ScopedCanceled = make(map[Scmer]bool)
	}
	ctx := executionContextFrom(tx)
	sessionEnsureScopedCleanup(sess, scope, ctx)
	if flight := sess.ScopedFlights[computeKey]; flight != nil {
		sess.Mu.Unlock()
		select {
		case <-flight.done:
			if flight.failed {
				panic(flight.panicValue)
			}
			return flight.value
		case <-ctx.Done():
			panic(ctx.Err())
		}
	}
	flight := &sessionFlight{done: make(chan struct{})}
	sess.ScopedFlights[computeKey] = flight
	sess.Mu.Unlock()

	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				flight.panicValue = recovered
				flight.failed = true
			}
		}()
		if passTx {
			flight.value = Apply(producer, tx)
		} else {
			flight.value = Apply(producer)
		}
	}()

	sess.Mu.Lock()
	delete(sess.ScopedFlights, computeKey)
	if !flight.failed && !sess.ScopedCanceled[scope] {
		sess.ScopedValues[computeKey] = flight.value
	}
	close(flight.done)
	if sess.ScopedCanceled[scope] && !sessionHasScopedFlight(sess, scope) {
		delete(sess.ScopedCleanup, scope)
		delete(sess.ScopedCanceled, scope)
	}
	sess.Mu.Unlock()
	if flight.failed {
		panic(flight.panicValue)
	}
	return flight.value
}

// build this function into your SCM environment to offer http server capabilities
func NewSession(a ...Scmer) Scmer {
	sess := new(session)
	sess.Map = make(map[string]Scmer)
	call := func(a ...Scmer) (result Scmer) {
		switch len(a) {
		case 2:
			// No panic path between lock and unlock, so an explicit unlock avoids
			// the per-call deferred-record cost on this very hot accessor.
			if a[0].GetTag() >= 100 {
				sess.Mu.Lock()
				if sess.Handles == nil {
					sess.Handles = make(map[Scmer]Scmer)
				}
				sess.Handles[a[0]] = a[1]
				sess.Mu.Unlock()
			} else {
				key := a[0].String()
				sess.Mu.Lock()
				sess.Map[key] = a[1]
				sess.Mu.Unlock()
			}
			return a[1]
		case 4:
			if a[0].String() != "get_or_compute_scoped" {
				panic("session: unknown 4-argument operation")
			}
			return sessionGetOrComputeScoped(sess, a[1], a[2].String(), a[1], a[3], false)
		case 5:
			if a[0].String() != "get_or_compute_scoped" {
				panic("session: unknown 5-argument operation")
			}
			return sessionGetOrComputeScoped(sess, a[1], a[2].String(), a[3], a[4], true)
		case 1:
			if a[0].GetTag() >= 100 {
				sess.Mu.RLock()
				v, ok := sess.Handles[a[0]]
				sess.Mu.RUnlock()
				if ok {
					return v
				}
				return NewNil()
			}
			key := a[0].String()
			sess.Mu.RLock()
			v, ok := sess.Map[key]
			sess.Mu.RUnlock()
			if ok {
				return v
			}
			return NewNil()
		case 0:
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			keys := make([]Scmer, 0, len(sess.Map)+len(sess.Handles))
			for k := range sess.Map {
				keys = append(keys, NewString(k))
			}
			for k := range sess.Handles {
				keys = append(keys, k)
			}
			return NewSlice(keys)
		default:
			panic("wrong number of parameters provided to session: 0, 1, 2, 4, or 5 required")
		}
	}
	sess.call = call
	return NewClusterFunc(call, sess, sessionCallableID)
}

var sessionCallableID = RegisterCallableType(sessionCallableType)

var sessionCallableType = &TypeDescriptor{Kind: "func", Label: "session", Description: "session accessor accepting exactly zero, one, two, four, or five arguments", HasSideEffects: true,
	Params: []*TypeDescriptor{
		{Kind: "any", Label: "key_or_operation", Description: "key, or get_or_compute_scoped", Optional: true},
		{Kind: "any", Label: "value_or_scope", Description: "value to store, or scope for get_or_compute_scoped", Optional: true},
		{Kind: "any", Label: "scoped_key", Description: "cache key used by get_or_compute_scoped", Optional: true},
		{Kind: "func", CallsOnce: true, Label: "scoped_producer", Description: "producer used by the four-argument get_or_compute_scoped form", Optional: true, Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any", Label: "value", Description: "computed value cached for the scope and key"}},
		{Kind: "func", CallsOnce: true, Label: "transactional_scoped_producer", Description: "producer used by the five-argument get_or_compute_scoped form and called with its explicit transaction argument", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &TypeDescriptor{Kind: "any", Label: "value", Description: "computed value cached for the scope and key"}},
	},
	Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "value list, stored value, retrieved value, or shared computed value"},
}

// Context creates a Scheme session and passes it explicitly to fn.
func Context(a ...Scmer) Scmer {
	if len(a) == 0 || a[0].IsNil() {
		panic("context requires a callback")
	}
	args := make([]Scmer, len(a))
	args[0] = NewSession()
	copy(args[1:], a[1:])
	return Apply(a[0], args...)
}

// WithSession evaluates fn in a copy of its lexical closure with session bound.
// Copying the existing frame, instead of inserting another one, preserves the
// exact depth of optimizer-resolved outer references and keeps concurrent calls
// isolated from each other.
func WithSession(session Scmer, fn Scmer) Scmer {
	if !fn.IsProc() {
		return Apply(fn)
	}
	original := fn.Proc()
	proc := original.snapshot()
	outer := original.En
	if outer == nil {
		outer = &Globalenv
	}
	previousSession, hadPreviousSession := outer.Vars[Symbol("session")]
	vars := make(Vars, len(outer.Vars)+1)
	for name, value := range outer.Vars {
		vars[name] = value
	}
	vars[Symbol("session")] = session
	reboundEnv := &Env{
		Vars:         vars,
		VarsNumbered: outer.VarsNumbered,
		Outer:        outer.Outer,
		Nodefine:     outer.Nodefine,
	}
	if rebound := jitRebindProcCapture(original, reboundEnv, NewSymbol("session"), previousSession, hadPreviousSession, session); rebound != nil {
		return Apply(Scmer{ptr: (*byte)(unsafe.Pointer(rebound)), aux: makeAux(tagProc, 0)})
	}
	proc.En = reboundEnv
	proc.JITCode = 0
	proc.Compiled = nil
	callable := NewProcStruct(proc)
	if jitEnabled {
		callable = jitCompileMode(true, callable)
	}
	return Apply(callable)
}

// orderedProducerBatch owns its row arrays. Producers may borrow native call
// frames, so emit copies each complete row before crossing the stream boundary.
// Queue and current batch sizes are bounded independently of result cardinality.
type orderedProducerBatch struct {
	rows    [][]Scmer
	failure any
}

type orderedProducerCursor struct {
	input   <-chan orderedProducerBatch
	rows    [][]Scmer
	index   int
	ordinal int
}

type orderedProducerHeap struct {
	cursors   []*orderedProducerCursor
	positions []int
	relations []SerialProc
}

func (h orderedProducerHeap) Len() int      { return len(h.cursors) }
func (h orderedProducerHeap) Swap(i, j int) { h.cursors[i], h.cursors[j] = h.cursors[j], h.cursors[i] }
func (h orderedProducerHeap) Less(i, j int) bool {
	a, b := h.cursors[i], h.cursors[j]
	left, right := a.rows[a.index], b.rows[b.index]
	var args [2]Scmer
	for k, position := range h.positions {
		args[0], args[1] = left[position], right[position]
		if ToBool(h.relations[k].Call(args[:])) {
			return true
		}
		args[0], args[1] = right[position], left[position]
		if ToBool(h.relations[k].Call(args[:])) {
			return false
		}
	}
	return a.ordinal < b.ordinal
}
func (h *orderedProducerHeap) Push(v any) { h.cursors = append(h.cursors, v.(*orderedProducerCursor)) }
func (h *orderedProducerHeap) Pop() any {
	n := len(h.cursors) - 1
	v := h.cursors[n]
	h.cursors[n] = nil
	h.cursors = h.cursors[:n]
	return v
}

// scanOrderMerge merges already ordered, complete-row producers. It does not
// acquire storage locks or retain an invocation beyond the call. Cleanup joins
// every producer, including on consumer failure/cancellation; producers whose
// storage operator does not support braking finish with emission disabled.
func scanOrderMerge(a ...Scmer) Scmer {
	ctx := executionContextFrom(a[0])
	if ctx == nil {
		ctx = context.Background()
	}
	producers := asSlice(a[1], "scan_order_merge producers")
	positions := asSlice(a[2], "scan_order_merge positions")
	relations := asSlice(a[3], "scan_order_merge relations")
	offset, limit := int(ToInt(a[4])), int(ToInt(a[5]))
	if offset < 0 || limit < -1 || len(positions) != len(relations) {
		panic("scan_order_merge: invalid order/window")
	}
	h := orderedProducerHeap{positions: make([]int, len(positions)), relations: make([]SerialProc, len(relations))}
	for i := range positions {
		h.positions[i] = int(ToInt(positions[i]))
		if h.positions[i] < 0 {
			panic("scan_order_merge: negative order position")
		}
		h.relations[i] = PrepareSerialProc(relations[i])
	}
	if limit == 0 {
		return NewNil()
	}
	stop := make(chan struct{})
	var workers sync.WaitGroup
	var failureMu sync.Mutex
	var producerFailure any
	defer func() {
		failure := recover()
		close(stop)
		workers.Wait()
		if failure != nil {
			panic(failure)
		}
		// A producer which finishes after LIMIT must not silently lose a failure.
		if producerFailure != nil {
			panic(producerFailure)
		}
	}()
	for ordinal, producer := range producers {
		input := make(chan orderedProducerBatch, 1)
		program := PrepareSerialProc(producer)
		workers.Add(1)
		go func() {
			defer workers.Done()
			defer close(input)
			defer func() {
				if failure := recover(); failure != nil {
					failureMu.Lock()
					if producerFailure == nil {
						producerFailure = failure
					}
					failureMu.Unlock()
					select {
					case input <- orderedProducerBatch{failure: failure}:
					case <-stop:
					}
				}
			}()
			const batchSize = 64
			batch := make([][]Scmer, 0, batchSize)
			flush := func() bool {
				if len(batch) == 0 {
					return true
				}
				select {
				case input <- orderedProducerBatch{rows: batch}:
					batch = make([][]Scmer, 0, batchSize)
					return true
				case <-stop:
					return false
				case <-ctx.Done():
					panic(ctx.Err())
				}
			}
			var mu sync.Mutex
			emit := NewFunc(func(values ...Scmer) Scmer {
				mu.Lock()
				defer mu.Unlock()
				select {
				case <-stop:
					return NewBool(false)
				default:
				}
				if len(values) != 1 {
					panic("scan_order_merge: emit expects one row")
				}
				row := asSlice(values[0], "scan_order_merge row")
				for _, position := range h.positions {
					if position >= len(row) {
						panic("scan_order_merge: order position outside row")
					}
				}
				batch = append(batch, append([]Scmer(nil), row...))
				if len(batch) == batchSize {
					return NewBool(flush())
				}
				return NewBool(true)
			})
			program.Call([]Scmer{emit})
			flush()
		}()
		h.cursors = append(h.cursors, &orderedProducerCursor{input: input, ordinal: ordinal})
	}
	next := func(cursor *orderedProducerCursor) bool {
		if cursor.index+1 < len(cursor.rows) {
			cursor.index++
			return true
		}
		select {
		case batch, ok := <-cursor.input:
			if !ok {
				cursor.rows = nil
				return false
			}
			if batch.failure != nil {
				panic(batch.failure)
			}
			cursor.rows, cursor.index = batch.rows, 0
			return len(cursor.rows) > 0
		case <-ctx.Done():
			panic(ctx.Err())
		}
	}
	live := h.cursors[:0]
	for _, cursor := range h.cursors {
		if next(cursor) {
			live = append(live, cursor)
		}
	}
	h.cursors = live
	heap.Init(&h)
	consumer := PrepareSerialProc(a[6])
	var args [1]Scmer
	seen, emitted := 0, 0
	for h.Len() > 0 {
		cursor := h.cursors[0]
		if seen >= offset {
			args[0] = NewSlice(cursor.rows[cursor.index])
			consumer.Call(args[:])
			emitted++
			if limit >= 0 && emitted >= limit {
				break
			}
		}
		seen++
		if next(cursor) {
			heap.Fix(&h, 0)
		} else {
			heap.Pop(&h)
		}
	}
	return NewNil()
}

func init_sync() {
	initClusterObjects()
	Declare(&Globalenv, &Declaration{
		Name: "scan_order_merge", Fn: scanOrderMerge,
		Type: &TypeDescriptor{Kind: "func", Description: "Bounded merge of ordered complete-row producers", HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "tx"},
				{Kind: "list", Label: "producers", Element: &TypeDescriptor{Kind: "func", CrossGoroutine: true, Params: []*TypeDescriptor{{Kind: "func", Label: "emit", Params: []*TypeDescriptor{{Kind: "list", Label: "row"}}, Return: &TypeDescriptor{Kind: "any"}}}, Return: &TypeDescriptor{Kind: "any"}}},
				{Kind: "list", Label: "positions"}, {Kind: "list", Label: "relations"},
				{Kind: "number", Label: "offset"}, {Kind: "number", Label: "limit"},
				{Kind: "func", Label: "consumer", Params: []*TypeDescriptor{{Kind: "list", Label: "row"}}, Return: &TypeDescriptor{Kind: "any"}},
			}, Return: &TypeDescriptor{Kind: "any"},
			JITInlineCost: 65535,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: channel construction.
				ctx.Coverage.NativeCalls++
				declaration := declarations["scan_order_merge"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	DeclareTitle("Sync")
	Declare(&Globalenv, &Declaration{
		Name: "newpromise",

		Fn: NewPromise,
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a thread-safe promise that lets parallel work publish one result for other code to inspect. Use it for shared initialization, asynchronous results, or ensuring that only the first successful producer resolves a value. Read with (promise \"value\"), inspect completion with (promise \"state\"), resolve with (promise \"value\" value), resolve exactly once with (promise \"once\" value), and mark failure with (promise \"fail\" error).",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "storage", Description: "optional existing two-item list used to hold the promise state; most callers omit this", Optional: true, Element: &TypeDescriptor{Kind: "any", Label: "slot", Description: "promise state or value slot"}},
			},
			Return: &TypeDescriptor{Kind: "func", Label: "promise", Description: "operation-based accessor for reading, resolving, or failing the promise", HasSideEffects: true,
				Params: []*TypeDescriptor{
					{Kind: "string", Label: "operation", Description: "one of: value, state, fail, once"},
					{Kind: "any", Label: "value", Description: "value to store (for value/once/fail)", Optional: true},
					{Kind: "string", Label: "message", Description: "optional error message used when once finds an already completed promise", Optional: true},
				},
				Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "stored value, state flag, or operation result"},
			},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["newpromise"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var stackArray7 int32
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
				var bbs [5]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
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
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d0
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.SyncDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d2.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d1)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
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
					ctx.ReclaimUntrackedRegs()
					stackArray7 = ctx.AllocStack(int32(32))
					_ = stackArray7
					d8 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d8
					d9 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d9)
					ctx.EmitStoreScmerToStack(d9, int32(stackArray7)+int32(16))
					ctx.FreeDesc(&d9)
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d10 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d10)
					ctx.BindReg(r2, &d10)
					d11 = args[0]
					d11.ID = 0
					r3 := ctx.AllocReg()
					ctx.EmitLeaRegMem(r3, ctx.StackReg, int32(stackArray7)+int32(0))
					d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d12)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					d15 = args[0]
					d15.ID = 0
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.ReclaimUntrackedRegs()
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.ReclaimUntrackedRegs()
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.ReclaimUntrackedRegs()
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.EmitMovToReg(d11.Reg, d12)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d22)
					ctx.EmitMovToReg(d15.Reg2, d22)
					ctx.FreeDesc(&d22)
					d23 = d10
					_ = d23
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
					ctx.ReclaimUntrackedRegs()
					d24 = args[0]
					d24.ID = 0
					d25 = jitKnownSliceHeader(ctx, &d24)
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.FreeDesc(&d24)
					if d25.SliceSizeKnown {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
					} else if d25.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
					} else if d25.Loc == LocStackTriple {
						d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d25)
						if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
						} else if d25.Loc == LocReg {
							d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d26
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() < 2)}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d26.Reg, 2)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d27)
					}
					ctx.FreeDesc(&d26)
					d28 = d27
					ctx.SyncDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d28.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d27)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap29 := d0
						snap30 := d1
						snap31 := d2
						snap32 := d8
						snap33 := d9
						snap34 := d10
						snap35 := d11
						snap36 := d12
						snap37 := d13
						snap38 := d14
						snap39 := d15
						snap40 := d16
						snap41 := d17
						snap42 := d18
						snap43 := d19
						snap44 := d20
						snap45 := d21
						snap46 := d22
						snap47 := d23
						snap48 := d24
						snap49 := d25
						snap50 := d26
						snap51 := d27
						snap52 := d28
						alloc53 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc53)
						d0 = snap29
						d1 = snap30
						d2 = snap31
						d8 = snap32
						d9 = snap33
						d10 = snap34
						d11 = snap35
						d12 = snap36
						d13 = snap37
						d14 = snap38
						d15 = snap39
						d16 = snap40
						d17 = snap41
						d18 = snap42
						d19 = snap43
						d20 = snap44
						d21 = snap45
						d22 = snap46
						d23 = snap47
						d24 = snap48
						d25 = snap49
						d26 = snap50
						d27 = snap51
						d28 = snap52
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["newpromise"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					ctx.ReclaimUntrackedRegs()
					d54 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d54)
					ctx.SyncDesc(&d25)
					ctx.SyncDesc(&d55)
					d56 = d25
					d56.ID = 0
					d57 = d55
					d57.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d56, &d57, &d54, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d56, &d57, &d54, int32(16))
					}
					ctx.FreeDesc(&d57)
					ctx.FreeDesc(&d54)
					r5 := ctx.AllocReg()
					r6 := ctx.AllocRegExcept(r5)
					ctx.EmitMovRegImm64(r5, 0)
					ctx.EmitMovRegImm64(r6, 0)
					d58 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
					ctx.BindReg(r5, &d58)
					ctx.BindReg(r6, &d58)
					d59 = args[0]
					d59.ID = 0
					d60 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d61 = ctx.EmitSliceElementAddress(&d25, &d60, int32(16))
					ctx.EnsureDesc(&d61)
					ctx.EnsureDesc(&d61)
					ctx.EnsureDesc(&d61)
					d64 = args[0]
					d64.ID = 0
					d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(256)}
					ctx.ReclaimUntrackedRegs()
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.ReclaimUntrackedRegs()
					d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					ctx.ReclaimUntrackedRegs()
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(278)}
					ctx.ReclaimUntrackedRegs()
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(278)}
					ctx.EnsureDesc(&d61)
					ctx.EnsureDesc(&d61)
					ctx.EmitMovToReg(d59.Reg, d61)
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					ctx.EmitMovToReg(d64.Reg2, d71)
					ctx.FreeDesc(&d71)
					d72 = d58
					_ = d72
					ctx.SyncDesc(&d72)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d72, &result)
						result.Type = d72.Type
					} else {
						switch d72.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d72)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d72)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d72)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d72, &result)
							result.Type = d72.Type
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
			JITInlineCost:  51,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "newsession",

		Fn: NewSession,
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a thread-safe key-value session. Call it without arguments to list values, with a key to read, with a key and value to store, or with get_or_compute_scoped, a scope, a key, and a producer to share one concurrent computation.",
			Return: sessionCallableType,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["newsession"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "with_session",

		Fn: func(a ...Scmer) Scmer {
			return WithSession(a[0], a[1])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Executes a function with the given session installed in the execution context, so storage operations can access the session's transaction state.",
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "session", Description: "the session to install", Params: []*TypeDescriptor{{Kind: "any", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", CallsOnce: true, Label: "fn", Description: "the function to execute", Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: &TypeDescriptor{Kind: "any"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["with_session"]
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
				d0 = JITPrepareScmerGoArg(ctx, d0)
				d1 = JITPrepareScmerGoArg(ctx, d1)
				ctx.SyncDesc(&d0)
				ctx.SyncDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(WithSession), []JITValueDesc{d0, d1}, 2)
				d2.NoHeapPointer = false
				ctx.BindReg(d2.Reg, &d2)
				ctx.BindReg(d2.Reg2, &d2)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
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
			JITInlineCost: 6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "context",

		Fn: Context,
		Type: &TypeDescriptor{Kind: "func", Description: "Context helper function. Each context also contains a session. (context func args) creates a new context and runs func in that context, (context \"session\") reads the session variable, (context \"check\") will check the liveliness of the context and otherwise throw an error",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "args...", Description: "depends on the usage", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["context"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d7 JITValueDesc
				_ = d7
				var d9 JITValueDesc
				_ = d9
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
				var d22 JITValueDesc
				_ = d22
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
				var bbs [4]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
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
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d0
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.SyncDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d2.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d1)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
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
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["context"].Fn, args, result)
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
					ctx.ReclaimUntrackedRegs()
					d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					_ = d7
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					callResults8 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d7, d7}, []uint8{3}, []uint8{1})
					d9 = callResults8[0]
					d9.Type = tagSlice
					ctx.FreeDesc(&d7)
					d10 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d10 = JITPrepareGoSliceArg(ctx, d10)
					if d10.Loc != LocRegTriple && d10.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (NewSession arg0)")
					}
					ctx.SyncDesc(&d10)
					d11 = ctx.EmitGoCallScalar(GoFuncAddr(NewSession), []JITValueDesc{d10}, 2)
					d11.NoHeapPointer = false
					ctx.BindReg(d11.Reg, &d11)
					ctx.BindReg(d11.Reg2, &d11)
					ctx.FreeDesc(&d10)
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d9)
					ctx.SyncDesc(&d12)
					d13 = d9
					d13.ID = 0
					d14 = d12
					d14.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d13, &d14, &d11, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d13, &d14, &d11, int32(16))
					}
					ctx.FreeDesc(&d14)
					ctx.FreeDesc(&d11)
					d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair || d9.Loc == LocRegTriple {
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2}
						ctx.BindReg(d9.Reg2, &d16)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm && d15.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() - d15.Imm.Int())}
					} else {
						r1 := ctx.AllocReg()
						if d16.Loc == LocImm {
							ctx.EmitMovRegImm64(r1, uint64(d16.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r1, d16.Reg)
						}
						if d15.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d15.Imm.Int()))
							ctx.EmitSubInt64(r1, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r1, d15.Reg)
						}
						d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d18)
					}
					r2 := ctx.EmitSliceDataAfterLow(&d9, &d15, 16)
					d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d19)
					ctx.BindReg(r2, &d19)
					var r3 Reg
					var r4 Reg
					ctx.SyncDesc(&d19)
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d19.Imm.Int()))
					} else {
						r3 = d19.Reg
					}
					ctx.ProtectReg(r3)
					ctx.SyncDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d18.Imm.Int()))
					} else {
						r4 = d18.Reg
					}
					ctx.ProtectReg(r4)
					r5 := ctx.EmitSliceCapAfterLow(&d9, &d15, r3, r4)
					ctx.UnprotectReg(r4)
					ctx.UnprotectReg(r3)
					d20 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
					ctx.BindReg(r3, &d20)
					ctx.BindReg(r4, &d20)
					ctx.BindReg(r5, &d20)
					ctx.BindReg(r3, &d20)
					ctx.BindReg(r4, &d20)
					ctx.BindReg(r5, &d20)
					ctx.EnsureDesc(&d20)
					callResults21 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d20}, []uint8{1}, []uint8{0})
					d22 = callResults21[0]
					d22.Type = tagInt
					d23 = args[0]
					d23.ID = 0
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d9)
					d24 = d23
					_ = d24
					d25 = d9
					_ = d25
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl5 := ctx.ReserveLabel()
					_ = lbl5
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d24 = JITPrepareScmerGoArg(ctx, d24)
					d25 = JITPrepareGoSliceArg(ctx, d25)
					if d25.Loc != LocRegTriple && d25.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d26.Loc == LocRegPair || d26.Loc == LocStackPair || d26.Loc == LocRegTriple || d26.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d24)
					ctx.SyncDesc(&d25)
					ctx.SyncDesc(&d26)
					d27 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d24, d25, d26}, 2)
					d27.NoHeapPointer = false
					ctx.BindReg(d27.Reg, &d27)
					ctx.BindReg(d27.Reg2, &d27)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.FreeDesc(&d23)
					ctx.SyncDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocStackPair || d27.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d27, &result)
						result.Type = d27.Type
					} else {
						switch d27.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d27)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d27)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d27)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d27, &result)
							result.Type = d27.Type
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
					ctx.ReclaimUntrackedRegs()
					d28 = args[0]
					d28.ID = 0
					d30 = d28
					d30.ID = 0
					d29 = ctx.EmitTagEqualsBorrowed(&d30, tagNil, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d28)
					d31 = d29
					ctx.EnsureDesc(&d31)
					if d31.Loc != LocImm && d31.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d31.Loc == LocImm {
						if d31.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitCmpRegImm32(d31.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap32 := d0
						snap33 := d1
						snap34 := d2
						snap35 := d7
						snap36 := d9
						snap37 := d10
						snap38 := d11
						snap39 := d12
						snap40 := d13
						snap41 := d14
						snap42 := d15
						snap43 := d16
						snap44 := d17
						snap45 := d18
						snap46 := d19
						snap47 := d20
						snap48 := d22
						snap49 := d23
						snap50 := d24
						snap51 := d25
						snap52 := d26
						snap53 := d27
						snap54 := d28
						snap55 := d29
						snap56 := d30
						snap57 := d31
						alloc58 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc58)
						d0 = snap32
						d1 = snap33
						d2 = snap34
						d7 = snap35
						d9 = snap36
						d10 = snap37
						d11 = snap38
						d12 = snap39
						d13 = snap40
						d14 = snap41
						d15 = snap42
						d16 = snap43
						d17 = snap44
						d18 = snap45
						d19 = snap46
						d20 = snap47
						d22 = snap48
						d23 = snap49
						d24 = snap50
						d25 = snap51
						d26 = snap52
						d27 = snap53
						d28 = snap54
						d29 = snap55
						d30 = snap56
						d31 = snap57
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					ctx.FreeDesc(&d29)
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
			JITInlineCallbacks: false,
			JITInlineCost:      23,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sleep",

		Fn: func(a ...Scmer) Scmer {
			queryState := NewNil()
			duration := a[0]
			if len(a) > 1 {
				queryState = a[0]
				duration = a[1]
			}
			ctx := executionContextFrom(queryState)
			select {
			case <-ctx.Done():
				panic(ctx.Err())
			case <-time.After(time.Duration(ToFloat(duration) * float64(time.Second))):
				return NewBool(true)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "sleeps the amount of seconds and observes cancellation through tx",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "duration_or_tx", Description: "duration, or explicit transaction context followed by duration", Optional: true},
				{Kind: "number", Label: "duration", Description: "number of seconds to sleep when a transaction context is supplied", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: channel select.
				ctx.Coverage.NativeCalls++
				declaration := declarations["sleep"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "once",

		Fn: func(a ...Scmer) Scmer {
			callable := a[0]
			var params []Scmer
			var paramsOnce sync.Once
			once := sync.OnceValue[Scmer](func() Scmer {
				return Apply(callable, params...)
			})
			return NewFunc(func(a ...Scmer) Scmer {
				paramsOnce.Do(func() {
					params = append([]Scmer(nil), a...)
				})
				return once()
			})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a function wrapper that you can call multiple times but only gets executed once. The result value is cached and returned on a second call. You can add parameters to that resulting function that will be passed to the first run of the wrapped function.",
			Params: []*TypeDescriptor{
				{Kind: "func", CallsOnce: true, Label: "f", Description: "function that produces the result value", Params: []*TypeDescriptor{{Kind: "any", Label: "argument", Variadic: true}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}},
			},
			Return: &TypeDescriptor{Kind: "func", Label: "once_wrapper", Description: "calls the wrapped function once and returns its cached result thereafter",
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "args", Description: "arguments forwarded to the wrapped function on first call", Variadic: true},
				},
				Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "result cached from the first call"},
			},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["once"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mutex",

		Fn: func(a ...Scmer) Scmer {
			token := make(chan struct{}, 1)
			token <- struct{}{}
			return NewFunc(func(a ...Scmer) Scmer {
				queryState := NewNil()
				fn := a[0]
				if len(a) > 1 {
					queryState = a[0]
					fn = a[1]
				}
				ctx := executionContextFrom(queryState)
				select {
				case <-token:
					if err := ctx.Err(); err != nil {
						token <- struct{}{}
						panic(err)
					}
				case <-ctx.Done():
					panic(ctx.Err())
				}
				defer func() {
					token <- struct{}{} // free after return or panic, so we don't get into deadlocks
					/* this code happens automatically
					if r := recover(); r != nil {
						// rethrow panics
						panic(r)
					}*/
				}()

				// execute serially
				return Apply(fn)
			})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a context-aware mutex. The return value serializes calls to parameterless functions and stops waiting when the current request is cancelled.",
			Params: []*TypeDescriptor{},
			Return: &TypeDescriptor{Kind: "func", Label: "locked", Description: "executes one parameterless function while holding the mutex", HasSideEffects: true,
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "fn_or_tx", Description: "function, or explicit transaction context followed by function", Optional: true},
					{Kind: "func", CallsOnce: true, Label: "fn", Description: "parameterless function to execute under the lock", Optional: true, Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any", Label: "result"}},
				},
				Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "result returned by the protected function"},
			},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: channel construction.
				ctx.Coverage.NativeCalls++
				declaration := declarations["mutex"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "numcpu",

		Fn: func(a ...Scmer) Scmer {
			return NewInt(int64(runtime.NumCPU()))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Returns the number of logical CPUs available for parallel execution",
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["numcpu"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(runtime.NumCPU), []JITValueDesc{}, 1)
				d0.NoHeapPointer = true
				ctx.BindReg(d0.Reg, &d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d0.Loc == LocImm {
					ctx.EmitMakeInt(result, d0)
				} else {
					ctx.EmitMakeInt(result, d0)
					ctx.FreeReg(d0.Reg)
				}
				result.Type = tagInt
				return result
				return result
			},
			JITInlineCost: 4,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "memstats",

		Fn: func(a ...Scmer) Scmer {
			m := CachedMemStats()
			fd := NewFastDictValue(5)
			fd.Set(NewString("alloc"), NewInt(int64(m.Alloc)), nil)
			fd.Set(NewString("total_alloc"), NewInt(int64(m.TotalAlloc)), nil)
			fd.Set(NewString("sys"), NewInt(int64(m.Sys)), nil)
			fd.Set(NewString("heap_alloc"), NewInt(int64(m.HeapAlloc)), nil)
			fd.Set(NewString("heap_sys"), NewInt(int64(m.HeapSys)), nil)
			return NewFastDict(fd)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Returns memory statistics as a dict with keys: alloc, total_alloc, sys, heap_alloc, heap_sys (all in bytes)",
			Return: &TypeDescriptor{Kind: "dict"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["memstats"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := ctx.EmitGoCallScalar(GoFuncAddr(func() *runtime.MemStats { return new(runtime.MemStats) }), nil, 1)
				ctx.BindReg(d0.Reg, &d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((func() *runtime.MemStats { value := CachedMemStats(); return &value })), []JITValueDesc{}, 1)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.EnsureDesc(&d1)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst, src *runtime.MemStats) { *dst = *src }), []JITValueDesc{d0, d1})
				d2 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(5)}
				_ = d2
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d2}, 1)
				d4 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("alloc")}
				_ = d4
				var d5 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 0
					r0 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r0, fieldAddr)
					d5 = JITValueDesc{Loc: LocReg, Reg: r0}
					ctx.BindReg(r0, &d5)
				} else {
					off := int32(0)
					baseReg := d0.Reg
					r1 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r1, baseReg, off)
					d5 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d5)
				}
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
				var d6 JITValueDesc
				if d5.Loc == LocImm {
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d5.Imm.Int()))))}
				} else {
					r2 := ctx.AllocReg()
					ctx.EmitMovRegReg(r2, d5.Reg)
					d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d6)
				}
				ctx.FreeDesc(&d5)
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d6)
				d7 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d7
				d8 := d3
				_ = d8
				ctx.StabilizeDescForControlFlow(&d8)
				inlineHome9 := d8
				inlineHome9.ID = 0
				d10 := d4
				_ = d10
				ctx.StabilizeDescForControlFlow(&d10)
				inlineHome11 := d10
				inlineHome11.ID = 0
				d12 := d6
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				inlineHome13 := d12
				inlineHome13.ID = 0
				ctx.StabilizeDescForControlFlow(&d3)
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
				bbpos_1_6 := int32(-1)
				_ = bbpos_1_6
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_1_7 := int32(-1)
				_ = bbpos_1_7
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d14 JITValueDesc
				ctx.EnsureDesc(&d8)
				if d8.Loc == LocImm {
					fieldAddr := uintptr(d8.Imm.Int()) + 24
					r3 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r3, fieldAddr)
					d14 = JITValueDesc{Loc: LocReg, Reg: r3}
					ctx.BindReg(r3, &d14)
				} else {
					off := int32(24)
					baseReg := d8.Reg
					r4 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r4, baseReg, off)
					d14 = JITValueDesc{Loc: LocReg, Reg: r4}
					ctx.BindReg(r4, &d14)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				var d15 JITValueDesc
				if d14.Loc == LocImm {
					d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d14)
					if d14.Loc != LocReg && d14.Loc != LocRegPair && d14.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r5 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					d15 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondEqual}
					ctx.BindReg(r5, &d15)
				}
				ctx.FreeDesc(&d14)
				ctx.ReclaimUntrackedRegs()
				d16 := d15
				ctx.SyncDesc(&d16)
				if d16.Loc != LocImm && d16.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl8 := ctx.ReserveLabel()
				lbl9 := ctx.ReserveLabel()
				if d16.Loc == LocImm {
					if d16.Imm.Bool() {
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl2)
					} else {
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl3)
					}
				} else {
					ctx.EmitJump(d16.Condition, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.FreeDesc(&d15)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl3)
				}
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d10 = JITPrepareScmerGoArg(ctx, d10)
				ctx.SyncDesc(&d10)
				d17 := ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d10}, 1)
				d17.NoHeapPointer = true
				ctx.BindReg(d17.Reg, &d17)
				ctx.StabilizeDescForControlFlow(&d17)
				inlineHome18 := d17
				inlineHome18.ID = 0
				ctx.ReclaimUntrackedRegs()
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d10 = JITPrepareScmerGoArg(ctx, d10)
				if d17.Loc == LocRegPair || d17.Loc == LocStackPair || d17.Loc == LocRegTriple || d17.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d8)
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d17)
				callResults19 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d8, d10, d17}, []uint8{1, 1}, []uint8{0, 0})
				d20 := callResults19[0]
				_ = d20
				d21 := callResults19[1]
				_ = d21
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d20)
				inlineHome22 := d20
				inlineHome22.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d23 := d21
				ctx.EnsureDesc(&d23)
				if d23.Loc != LocImm && d23.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl10 := ctx.ReserveLabel()
				lbl11 := ctx.ReserveLabel()
				if d23.Loc == LocImm {
					if d23.Imm.Bool() {
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl5)
				}
				ctx.FreeDesc(&d21)
				bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				d17 = inlineHome18
				d20 = inlineHome22
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d10 = JITPrepareScmerGoArg(ctx, d10)
				d12 = JITPrepareScmerGoArg(ctx, d12)
				if d17.Loc == LocRegPair || d17.Loc == LocStackPair || d17.Loc == LocRegTriple || d17.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d8)
				ctx.SyncDesc(&d10)
				ctx.SyncDesc(&d12)
				ctx.SyncDesc(&d17)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).insertHashed), []JITValueDesc{d8, d10, d12, d17})
				ctx.FreeDesc(&d8)
				ctx.FreeDesc(&d17)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl0)
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				d17 = inlineHome18
				d20 = inlineHome22
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d24 := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d24)
				ctx.EnsureDesc(&d8)
				ctx.EnsureDesc(&d24)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d8, d24})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl3)
				bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				d17 = inlineHome18
				d20 = inlineHome22
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d25 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d25
				ctx.ReclaimUntrackedRegs()
				bbpos_1_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				d17 = inlineHome18
				d20 = inlineHome22
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d26 JITValueDesc
				ctx.EnsureDesc(&d8)
				if d8.Loc == LocImm {
					fieldAddr := uintptr(d8.Imm.Int()) + 0
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					r8 := ctx.AllocRegExcept(r6, r7)
					ctx.EmitMovRegMem64(r6, fieldAddr)
					ctx.EmitMovRegMem64(r7, fieldAddr+8)
					ctx.EmitMovRegMem64(r8, fieldAddr+16)
					d26 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d26)
					ctx.BindReg(r7, &d26)
					ctx.BindReg(r8, &d26)
				} else {
					off := int32(0)
					baseReg := d8.Reg
					r9 := ctx.AllocRegExcept(baseReg)
					r10 := ctx.AllocRegExcept(baseReg, r9)
					r11 := ctx.AllocRegExcept(baseReg, r9, r10)
					ctx.EmitMovRegMem(r9, baseReg, off)
					ctx.EmitMovRegMem(r10, baseReg, off+8)
					ctx.EmitMovRegMem(r11, baseReg, off+16)
					d26 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11}
					ctx.BindReg(r9, &d26)
					ctx.BindReg(r10, &d26)
					ctx.BindReg(r11, &d26)
				}
				ctx.FreeDesc(&d8)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d20)
				ctx.EnsureDesc(&d20)
				var d27 JITValueDesc
				if d20.Loc == LocImm {
					d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d20.Reg)
					ctx.EmitMovRegReg(scratch, d20.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d27)
				}
				if d27.Loc == LocReg && d20.Loc == LocReg && d27.Reg == d20.Reg {
					ctx.TransferReg(d20.Reg)
					d20.Loc = LocNone
				}
				ctx.FreeDesc(&d20)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d27)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d12)
				ctx.SyncDesc(&d26)
				ctx.SyncDesc(&d27)
				d28 := d26
				d28.ID = 0
				d29 := d27
				d29.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d28, &d29, &d12, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d27)
					d29 = d27
					d29.ID = 0
					ctx.EmitStoreScmerSliceElement(&d28, &d29, &d12, int32(16))
				}
				ctx.FreeDesc(&d29)
				ctx.FreeDesc(&d27)
				ctx.ReclaimUntrackedRegs()
				bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d8 = inlineHome9
				d10 = inlineHome11
				d12 = inlineHome13
				d17 = inlineHome18
				d20 = inlineHome22
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl0)
				ctx.MarkLabel(lbl0)
				d30 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("total_alloc")}
				_ = d30
				var d31 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 8
					r12 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r12, fieldAddr)
					d31 = JITValueDesc{Loc: LocReg, Reg: r12}
					ctx.BindReg(r12, &d31)
				} else {
					off := int32(8)
					baseReg := d0.Reg
					r13 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r13, baseReg, off)
					d31 = JITValueDesc{Loc: LocReg, Reg: r13}
					ctx.BindReg(r13, &d31)
				}
				ctx.EnsureDesc(&d31)
				ctx.EnsureDesc(&d31)
				var d32 JITValueDesc
				if d31.Loc == LocImm {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d31.Imm.Int()))))}
				} else {
					r14 := ctx.AllocReg()
					ctx.EmitMovRegReg(r14, d31.Reg)
					d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d32)
				}
				ctx.FreeDesc(&d31)
				ctx.EnsureDesc(&d32)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d32)
				d33 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d33
				d34 := d3
				_ = d34
				ctx.StabilizeDescForControlFlow(&d34)
				inlineHome35 := d34
				inlineHome35.ID = 0
				d36 := d30
				_ = d36
				ctx.StabilizeDescForControlFlow(&d36)
				inlineHome37 := d36
				inlineHome37.ID = 0
				d38 := d32
				_ = d38
				ctx.StabilizeDescForControlFlow(&d38)
				inlineHome39 := d38
				inlineHome39.ID = 0
				ctx.StabilizeDescForControlFlow(&d3)
				lbl12 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_2_4 := int32(-1)
				_ = bbpos_2_4
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_2_6 := int32(-1)
				_ = bbpos_2_6
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_2_7 := int32(-1)
				_ = bbpos_2_7
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d40 JITValueDesc
				ctx.EnsureDesc(&d34)
				if d34.Loc == LocImm {
					fieldAddr := uintptr(d34.Imm.Int()) + 24
					r15 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r15, fieldAddr)
					d40 = JITValueDesc{Loc: LocReg, Reg: r15}
					ctx.BindReg(r15, &d40)
				} else {
					off := int32(24)
					baseReg := d34.Reg
					r16 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r16, baseReg, off)
					d40 = JITValueDesc{Loc: LocReg, Reg: r16}
					ctx.BindReg(r16, &d40)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d40)
				var d41 JITValueDesc
				if d40.Loc == LocImm {
					d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d40)
					if d40.Loc != LocReg && d40.Loc != LocRegPair && d40.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r17 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d40.Reg, 0)
					d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondEqual}
					ctx.BindReg(r17, &d41)
				}
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				d42 := d41
				ctx.SyncDesc(&d42)
				if d42.Loc != LocImm && d42.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl20 := ctx.ReserveLabel()
				lbl21 := ctx.ReserveLabel()
				if d42.Loc == LocImm {
					if d42.Imm.Bool() {
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl14)
					} else {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitJump(d42.Condition, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.FreeDesc(&d41)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl15)
				}
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d36 = JITPrepareScmerGoArg(ctx, d36)
				ctx.SyncDesc(&d36)
				d43 := ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d36}, 1)
				d43.NoHeapPointer = true
				ctx.BindReg(d43.Reg, &d43)
				ctx.StabilizeDescForControlFlow(&d43)
				inlineHome44 := d43
				inlineHome44.ID = 0
				ctx.ReclaimUntrackedRegs()
				if d34.Loc == LocRegPair || d34.Loc == LocStackPair || d34.Loc == LocRegTriple || d34.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d36 = JITPrepareScmerGoArg(ctx, d36)
				if d43.Loc == LocRegPair || d43.Loc == LocStackPair || d43.Loc == LocRegTriple || d43.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d34)
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d43)
				callResults45 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d34, d36, d43}, []uint8{1, 1}, []uint8{0, 0})
				d46 := callResults45[0]
				_ = d46
				d47 := callResults45[1]
				_ = d47
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d46)
				inlineHome48 := d46
				inlineHome48.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d49 := d47
				ctx.EnsureDesc(&d49)
				if d49.Loc != LocImm && d49.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				if d49.Loc == LocImm {
					if d49.Imm.Bool() {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl16)
					} else {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl17)
					}
				} else {
					ctx.EmitCmpRegImm32(d49.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl17)
				}
				ctx.FreeDesc(&d47)
				bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				d43 = inlineHome44
				d46 = inlineHome48
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d34.Loc == LocRegPair || d34.Loc == LocStackPair || d34.Loc == LocRegTriple || d34.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d36 = JITPrepareScmerGoArg(ctx, d36)
				d38 = JITPrepareScmerGoArg(ctx, d38)
				if d43.Loc == LocRegPair || d43.Loc == LocStackPair || d43.Loc == LocRegTriple || d43.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d34)
				ctx.SyncDesc(&d36)
				ctx.SyncDesc(&d38)
				ctx.SyncDesc(&d43)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).insertHashed), []JITValueDesc{d34, d36, d38, d43})
				ctx.FreeDesc(&d34)
				ctx.FreeDesc(&d43)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl12)
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				d43 = inlineHome44
				d46 = inlineHome48
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d50 := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d50)
				ctx.EnsureDesc(&d34)
				ctx.EnsureDesc(&d50)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d34, d50})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl15)
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				d43 = inlineHome44
				d46 = inlineHome48
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d51 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d51
				ctx.ReclaimUntrackedRegs()
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				d43 = inlineHome44
				d46 = inlineHome48
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d52 JITValueDesc
				ctx.EnsureDesc(&d34)
				if d34.Loc == LocImm {
					fieldAddr := uintptr(d34.Imm.Int()) + 0
					r18 := ctx.AllocReg()
					r19 := ctx.AllocRegExcept(r18)
					r20 := ctx.AllocRegExcept(r18, r19)
					ctx.EmitMovRegMem64(r18, fieldAddr)
					ctx.EmitMovRegMem64(r19, fieldAddr+8)
					ctx.EmitMovRegMem64(r20, fieldAddr+16)
					d52 = JITValueDesc{Loc: LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
					ctx.BindReg(r18, &d52)
					ctx.BindReg(r19, &d52)
					ctx.BindReg(r20, &d52)
				} else {
					off := int32(0)
					baseReg := d34.Reg
					r21 := ctx.AllocRegExcept(baseReg)
					r22 := ctx.AllocRegExcept(baseReg, r21)
					r23 := ctx.AllocRegExcept(baseReg, r21, r22)
					ctx.EmitMovRegMem(r21, baseReg, off)
					ctx.EmitMovRegMem(r22, baseReg, off+8)
					ctx.EmitMovRegMem(r23, baseReg, off+16)
					d52 = JITValueDesc{Loc: LocRegTriple, Reg: r21, Reg2: r22, Reg3: r23}
					ctx.BindReg(r21, &d52)
					ctx.BindReg(r22, &d52)
					ctx.BindReg(r23, &d52)
				}
				ctx.FreeDesc(&d34)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d46)
				var d53 JITValueDesc
				if d46.Loc == LocImm {
					d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d46.Reg)
					ctx.EmitMovRegReg(scratch, d46.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d53)
				}
				if d53.Loc == LocReg && d46.Loc == LocReg && d53.Reg == d46.Reg {
					ctx.TransferReg(d46.Reg)
					d46.Loc = LocNone
				}
				ctx.FreeDesc(&d46)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d53)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d38)
				ctx.SyncDesc(&d52)
				ctx.SyncDesc(&d53)
				d54 := d52
				d54.ID = 0
				d55 := d53
				d55.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d54, &d55, &d38, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d53)
					d55 = d53
					d55.ID = 0
					ctx.EmitStoreScmerSliceElement(&d54, &d55, &d38, int32(16))
				}
				ctx.FreeDesc(&d55)
				ctx.FreeDesc(&d53)
				ctx.ReclaimUntrackedRegs()
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d34 = inlineHome35
				d36 = inlineHome37
				d38 = inlineHome39
				d43 = inlineHome44
				d46 = inlineHome48
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl12)
				d56 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("sys")}
				_ = d56
				var d57 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 16
					r24 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r24, fieldAddr)
					d57 = JITValueDesc{Loc: LocReg, Reg: r24}
					ctx.BindReg(r24, &d57)
				} else {
					off := int32(16)
					baseReg := d0.Reg
					r25 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r25, baseReg, off)
					d57 = JITValueDesc{Loc: LocReg, Reg: r25}
					ctx.BindReg(r25, &d57)
				}
				ctx.EnsureDesc(&d57)
				ctx.EnsureDesc(&d57)
				var d58 JITValueDesc
				if d57.Loc == LocImm {
					d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d57.Imm.Int()))))}
				} else {
					r26 := ctx.AllocReg()
					ctx.EmitMovRegReg(r26, d57.Reg)
					d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
					ctx.BindReg(r26, &d58)
				}
				ctx.FreeDesc(&d57)
				ctx.EnsureDesc(&d58)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d56)
				ctx.EnsureDesc(&d58)
				d59 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d59
				d60 := d3
				_ = d60
				ctx.StabilizeDescForControlFlow(&d60)
				inlineHome61 := d60
				inlineHome61.ID = 0
				d62 := d56
				_ = d62
				ctx.StabilizeDescForControlFlow(&d62)
				inlineHome63 := d62
				inlineHome63.ID = 0
				d64 := d58
				_ = d64
				ctx.StabilizeDescForControlFlow(&d64)
				inlineHome65 := d64
				inlineHome65.ID = 0
				ctx.StabilizeDescForControlFlow(&d3)
				lbl24 := ctx.ReserveLabel()
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				lbl25 := ctx.ReserveLabel()
				_ = lbl25
				bbpos_3_1 := int32(-1)
				_ = bbpos_3_1
				lbl26 := ctx.ReserveLabel()
				_ = lbl26
				bbpos_3_2 := int32(-1)
				_ = bbpos_3_2
				lbl27 := ctx.ReserveLabel()
				_ = lbl27
				bbpos_3_3 := int32(-1)
				_ = bbpos_3_3
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_3_4 := int32(-1)
				_ = bbpos_3_4
				lbl29 := ctx.ReserveLabel()
				_ = lbl29
				bbpos_3_6 := int32(-1)
				_ = bbpos_3_6
				lbl30 := ctx.ReserveLabel()
				_ = lbl30
				bbpos_3_7 := int32(-1)
				_ = bbpos_3_7
				lbl31 := ctx.ReserveLabel()
				_ = lbl31
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl25)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d66 JITValueDesc
				ctx.EnsureDesc(&d60)
				if d60.Loc == LocImm {
					fieldAddr := uintptr(d60.Imm.Int()) + 24
					r27 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r27, fieldAddr)
					d66 = JITValueDesc{Loc: LocReg, Reg: r27}
					ctx.BindReg(r27, &d66)
				} else {
					off := int32(24)
					baseReg := d60.Reg
					r28 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r28, baseReg, off)
					d66 = JITValueDesc{Loc: LocReg, Reg: r28}
					ctx.BindReg(r28, &d66)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d66)
				var d67 JITValueDesc
				if d66.Loc == LocImm {
					d67 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d66.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d66)
					if d66.Loc != LocReg && d66.Loc != LocRegPair && d66.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r29 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d66.Reg, 0)
					d67 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondEqual}
					ctx.BindReg(r29, &d67)
				}
				ctx.FreeDesc(&d66)
				ctx.ReclaimUntrackedRegs()
				d68 := d67
				ctx.SyncDesc(&d68)
				if d68.Loc != LocImm && d68.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl32 := ctx.ReserveLabel()
				lbl33 := ctx.ReserveLabel()
				if d68.Loc == LocImm {
					if d68.Imm.Bool() {
						ctx.MarkLabel(lbl32)
						ctx.EmitJmp(lbl26)
					} else {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl27)
					}
				} else {
					ctx.EmitJump(d68.Condition, lbl32)
					ctx.EmitJmp(lbl33)
					ctx.FreeDesc(&d67)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl27)
				}
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d62 = JITPrepareScmerGoArg(ctx, d62)
				ctx.SyncDesc(&d62)
				d69 := ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d62}, 1)
				d69.NoHeapPointer = true
				ctx.BindReg(d69.Reg, &d69)
				ctx.StabilizeDescForControlFlow(&d69)
				inlineHome70 := d69
				inlineHome70.ID = 0
				ctx.ReclaimUntrackedRegs()
				if d60.Loc == LocRegPair || d60.Loc == LocStackPair || d60.Loc == LocRegTriple || d60.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d62 = JITPrepareScmerGoArg(ctx, d62)
				if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocRegTriple || d69.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d60)
				ctx.SyncDesc(&d62)
				ctx.SyncDesc(&d69)
				callResults71 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d60, d62, d69}, []uint8{1, 1}, []uint8{0, 0})
				d72 := callResults71[0]
				_ = d72
				d73 := callResults71[1]
				_ = d73
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d72)
				inlineHome74 := d72
				inlineHome74.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d75 := d73
				ctx.EnsureDesc(&d75)
				if d75.Loc != LocImm && d75.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl34 := ctx.ReserveLabel()
				lbl35 := ctx.ReserveLabel()
				if d75.Loc == LocImm {
					if d75.Imm.Bool() {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl28)
					} else {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl29)
					}
				} else {
					ctx.EmitCmpRegImm32(d75.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl34)
					ctx.EmitJmp(lbl35)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl29)
				}
				ctx.FreeDesc(&d73)
				bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl29)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				d69 = inlineHome70
				d72 = inlineHome74
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d60.Loc == LocRegPair || d60.Loc == LocStackPair || d60.Loc == LocRegTriple || d60.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d62 = JITPrepareScmerGoArg(ctx, d62)
				d64 = JITPrepareScmerGoArg(ctx, d64)
				if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocRegTriple || d69.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d60)
				ctx.SyncDesc(&d62)
				ctx.SyncDesc(&d64)
				ctx.SyncDesc(&d69)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).insertHashed), []JITValueDesc{d60, d62, d64, d69})
				ctx.FreeDesc(&d60)
				ctx.FreeDesc(&d69)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl24)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl26)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				d69 = inlineHome70
				d72 = inlineHome74
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d76 := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d76)
				ctx.EnsureDesc(&d60)
				ctx.EnsureDesc(&d76)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d60, d76})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl27)
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				d69 = inlineHome70
				d72 = inlineHome74
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d77 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d77
				ctx.ReclaimUntrackedRegs()
				bbpos_3_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl31)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				d69 = inlineHome70
				d72 = inlineHome74
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d78 JITValueDesc
				ctx.EnsureDesc(&d60)
				if d60.Loc == LocImm {
					fieldAddr := uintptr(d60.Imm.Int()) + 0
					r30 := ctx.AllocReg()
					r31 := ctx.AllocRegExcept(r30)
					r32 := ctx.AllocRegExcept(r30, r31)
					ctx.EmitMovRegMem64(r30, fieldAddr)
					ctx.EmitMovRegMem64(r31, fieldAddr+8)
					ctx.EmitMovRegMem64(r32, fieldAddr+16)
					d78 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
					ctx.BindReg(r30, &d78)
					ctx.BindReg(r31, &d78)
					ctx.BindReg(r32, &d78)
				} else {
					off := int32(0)
					baseReg := d60.Reg
					r33 := ctx.AllocRegExcept(baseReg)
					r34 := ctx.AllocRegExcept(baseReg, r33)
					r35 := ctx.AllocRegExcept(baseReg, r33, r34)
					ctx.EmitMovRegMem(r33, baseReg, off)
					ctx.EmitMovRegMem(r34, baseReg, off+8)
					ctx.EmitMovRegMem(r35, baseReg, off+16)
					d78 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
					ctx.BindReg(r33, &d78)
					ctx.BindReg(r34, &d78)
					ctx.BindReg(r35, &d78)
				}
				ctx.FreeDesc(&d60)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d72)
				ctx.EnsureDesc(&d72)
				var d79 JITValueDesc
				if d72.Loc == LocImm {
					d79 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d72.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d72.Reg)
					ctx.EmitMovRegReg(scratch, d72.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d79 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d79)
				}
				if d79.Loc == LocReg && d72.Loc == LocReg && d79.Reg == d72.Reg {
					ctx.TransferReg(d72.Reg)
					d72.Loc = LocNone
				}
				ctx.FreeDesc(&d72)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d64)
				ctx.SyncDesc(&d78)
				ctx.SyncDesc(&d79)
				d80 := d78
				d80.ID = 0
				d81 := d79
				d81.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d80, &d81, &d64, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d79)
					d81 = d79
					d81.ID = 0
					ctx.EmitStoreScmerSliceElement(&d80, &d81, &d64, int32(16))
				}
				ctx.FreeDesc(&d81)
				ctx.FreeDesc(&d79)
				ctx.ReclaimUntrackedRegs()
				bbpos_3_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl30)
				ctx.ResolveFixups()
				d60 = inlineHome61
				d62 = inlineHome63
				d64 = inlineHome65
				d69 = inlineHome70
				d72 = inlineHome74
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl24)
				ctx.MarkLabel(lbl24)
				d82 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("heap_alloc")}
				_ = d82
				var d83 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 48
					r36 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r36, fieldAddr)
					d83 = JITValueDesc{Loc: LocReg, Reg: r36}
					ctx.BindReg(r36, &d83)
				} else {
					off := int32(48)
					baseReg := d0.Reg
					r37 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r37, baseReg, off)
					d83 = JITValueDesc{Loc: LocReg, Reg: r37}
					ctx.BindReg(r37, &d83)
				}
				ctx.EnsureDesc(&d83)
				ctx.EnsureDesc(&d83)
				var d84 JITValueDesc
				if d83.Loc == LocImm {
					d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d83.Imm.Int()))))}
				} else {
					r38 := ctx.AllocReg()
					ctx.EmitMovRegReg(r38, d83.Reg)
					d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r38}
					ctx.BindReg(r38, &d84)
				}
				ctx.FreeDesc(&d83)
				ctx.EnsureDesc(&d84)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d82)
				ctx.EnsureDesc(&d84)
				d85 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d85
				d86 := d3
				_ = d86
				ctx.StabilizeDescForControlFlow(&d86)
				inlineHome87 := d86
				inlineHome87.ID = 0
				d88 := d82
				_ = d88
				ctx.StabilizeDescForControlFlow(&d88)
				inlineHome89 := d88
				inlineHome89.ID = 0
				d90 := d84
				_ = d90
				ctx.StabilizeDescForControlFlow(&d90)
				inlineHome91 := d90
				inlineHome91.ID = 0
				ctx.StabilizeDescForControlFlow(&d3)
				lbl36 := ctx.ReserveLabel()
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				lbl37 := ctx.ReserveLabel()
				_ = lbl37
				bbpos_4_1 := int32(-1)
				_ = bbpos_4_1
				lbl38 := ctx.ReserveLabel()
				_ = lbl38
				bbpos_4_2 := int32(-1)
				_ = bbpos_4_2
				lbl39 := ctx.ReserveLabel()
				_ = lbl39
				bbpos_4_3 := int32(-1)
				_ = bbpos_4_3
				lbl40 := ctx.ReserveLabel()
				_ = lbl40
				bbpos_4_4 := int32(-1)
				_ = bbpos_4_4
				lbl41 := ctx.ReserveLabel()
				_ = lbl41
				bbpos_4_6 := int32(-1)
				_ = bbpos_4_6
				lbl42 := ctx.ReserveLabel()
				_ = lbl42
				bbpos_4_7 := int32(-1)
				_ = bbpos_4_7
				lbl43 := ctx.ReserveLabel()
				_ = lbl43
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl37)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d92 JITValueDesc
				ctx.EnsureDesc(&d86)
				if d86.Loc == LocImm {
					fieldAddr := uintptr(d86.Imm.Int()) + 24
					r39 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r39, fieldAddr)
					d92 = JITValueDesc{Loc: LocReg, Reg: r39}
					ctx.BindReg(r39, &d92)
				} else {
					off := int32(24)
					baseReg := d86.Reg
					r40 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r40, baseReg, off)
					d92 = JITValueDesc{Loc: LocReg, Reg: r40}
					ctx.BindReg(r40, &d92)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d92)
				var d93 JITValueDesc
				if d92.Loc == LocImm {
					d93 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d92.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d92)
					if d92.Loc != LocReg && d92.Loc != LocRegPair && d92.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r41 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d92.Reg, 0)
					d93 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r41, Condition: CondEqual}
					ctx.BindReg(r41, &d93)
				}
				ctx.FreeDesc(&d92)
				ctx.ReclaimUntrackedRegs()
				d94 := d93
				ctx.SyncDesc(&d94)
				if d94.Loc != LocImm && d94.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl44 := ctx.ReserveLabel()
				lbl45 := ctx.ReserveLabel()
				if d94.Loc == LocImm {
					if d94.Imm.Bool() {
						ctx.MarkLabel(lbl44)
						ctx.EmitJmp(lbl38)
					} else {
						ctx.MarkLabel(lbl45)
						ctx.EmitJmp(lbl39)
					}
				} else {
					ctx.EmitJump(d94.Condition, lbl44)
					ctx.EmitJmp(lbl45)
					ctx.FreeDesc(&d93)
					ctx.MarkLabel(lbl44)
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl39)
				}
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl39)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d88 = JITPrepareScmerGoArg(ctx, d88)
				ctx.SyncDesc(&d88)
				d95 := ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d88}, 1)
				d95.NoHeapPointer = true
				ctx.BindReg(d95.Reg, &d95)
				ctx.StabilizeDescForControlFlow(&d95)
				inlineHome96 := d95
				inlineHome96.ID = 0
				ctx.ReclaimUntrackedRegs()
				if d86.Loc == LocRegPair || d86.Loc == LocStackPair || d86.Loc == LocRegTriple || d86.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d88 = JITPrepareScmerGoArg(ctx, d88)
				if d95.Loc == LocRegPair || d95.Loc == LocStackPair || d95.Loc == LocRegTriple || d95.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d86)
				ctx.SyncDesc(&d88)
				ctx.SyncDesc(&d95)
				callResults97 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d86, d88, d95}, []uint8{1, 1}, []uint8{0, 0})
				d98 := callResults97[0]
				_ = d98
				d99 := callResults97[1]
				_ = d99
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d98)
				inlineHome100 := d98
				inlineHome100.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d101 := d99
				ctx.EnsureDesc(&d101)
				if d101.Loc != LocImm && d101.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl46 := ctx.ReserveLabel()
				lbl47 := ctx.ReserveLabel()
				if d101.Loc == LocImm {
					if d101.Imm.Bool() {
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl40)
					} else {
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl41)
					}
				} else {
					ctx.EmitCmpRegImm32(d101.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl46)
					ctx.EmitJmp(lbl47)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl40)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl41)
				}
				ctx.FreeDesc(&d99)
				bbpos_4_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl41)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				d95 = inlineHome96
				d98 = inlineHome100
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d86.Loc == LocRegPair || d86.Loc == LocStackPair || d86.Loc == LocRegTriple || d86.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d88 = JITPrepareScmerGoArg(ctx, d88)
				d90 = JITPrepareScmerGoArg(ctx, d90)
				if d95.Loc == LocRegPair || d95.Loc == LocStackPair || d95.Loc == LocRegTriple || d95.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d86)
				ctx.SyncDesc(&d88)
				ctx.SyncDesc(&d90)
				ctx.SyncDesc(&d95)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).insertHashed), []JITValueDesc{d86, d88, d90, d95})
				ctx.FreeDesc(&d86)
				ctx.FreeDesc(&d95)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl36)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl38)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				d95 = inlineHome96
				d98 = inlineHome100
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d102 := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d102)
				ctx.EnsureDesc(&d86)
				ctx.EnsureDesc(&d102)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d86, d102})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl39)
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl40)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				d95 = inlineHome96
				d98 = inlineHome100
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d103 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d103
				ctx.ReclaimUntrackedRegs()
				bbpos_4_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl43)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				d95 = inlineHome96
				d98 = inlineHome100
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d104 JITValueDesc
				ctx.EnsureDesc(&d86)
				if d86.Loc == LocImm {
					fieldAddr := uintptr(d86.Imm.Int()) + 0
					r42 := ctx.AllocReg()
					r43 := ctx.AllocRegExcept(r42)
					r44 := ctx.AllocRegExcept(r42, r43)
					ctx.EmitMovRegMem64(r42, fieldAddr)
					ctx.EmitMovRegMem64(r43, fieldAddr+8)
					ctx.EmitMovRegMem64(r44, fieldAddr+16)
					d104 = JITValueDesc{Loc: LocRegTriple, Reg: r42, Reg2: r43, Reg3: r44}
					ctx.BindReg(r42, &d104)
					ctx.BindReg(r43, &d104)
					ctx.BindReg(r44, &d104)
				} else {
					off := int32(0)
					baseReg := d86.Reg
					r45 := ctx.AllocRegExcept(baseReg)
					r46 := ctx.AllocRegExcept(baseReg, r45)
					r47 := ctx.AllocRegExcept(baseReg, r45, r46)
					ctx.EmitMovRegMem(r45, baseReg, off)
					ctx.EmitMovRegMem(r46, baseReg, off+8)
					ctx.EmitMovRegMem(r47, baseReg, off+16)
					d104 = JITValueDesc{Loc: LocRegTriple, Reg: r45, Reg2: r46, Reg3: r47}
					ctx.BindReg(r45, &d104)
					ctx.BindReg(r46, &d104)
					ctx.BindReg(r47, &d104)
				}
				ctx.FreeDesc(&d86)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d98)
				ctx.EnsureDesc(&d98)
				var d105 JITValueDesc
				if d98.Loc == LocImm {
					d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d98.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d98.Reg)
					ctx.EmitMovRegReg(scratch, d98.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d105)
				}
				if d105.Loc == LocReg && d98.Loc == LocReg && d105.Reg == d98.Reg {
					ctx.TransferReg(d98.Reg)
					d98.Loc = LocNone
				}
				ctx.FreeDesc(&d98)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d105)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d90)
				ctx.SyncDesc(&d104)
				ctx.SyncDesc(&d105)
				d106 := d104
				d106.ID = 0
				d107 := d105
				d107.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d106, &d107, &d90, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d105)
					d107 = d105
					d107.ID = 0
					ctx.EmitStoreScmerSliceElement(&d106, &d107, &d90, int32(16))
				}
				ctx.FreeDesc(&d107)
				ctx.FreeDesc(&d105)
				ctx.ReclaimUntrackedRegs()
				bbpos_4_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl42)
				ctx.ResolveFixups()
				d86 = inlineHome87
				d88 = inlineHome89
				d90 = inlineHome91
				d95 = inlineHome96
				d98 = inlineHome100
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl36)
				ctx.MarkLabel(lbl36)
				d108 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("heap_sys")}
				_ = d108
				var d109 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 56
					r48 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r48, fieldAddr)
					d109 = JITValueDesc{Loc: LocReg, Reg: r48}
					ctx.BindReg(r48, &d109)
				} else {
					off := int32(56)
					baseReg := d0.Reg
					r49 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r49, baseReg, off)
					d109 = JITValueDesc{Loc: LocReg, Reg: r49}
					ctx.BindReg(r49, &d109)
				}
				ctx.EnsureDesc(&d109)
				ctx.EnsureDesc(&d109)
				var d110 JITValueDesc
				if d109.Loc == LocImm {
					d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d109.Imm.Int()))))}
				} else {
					r50 := ctx.AllocReg()
					ctx.EmitMovRegReg(r50, d109.Reg)
					d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r50}
					ctx.BindReg(r50, &d110)
				}
				ctx.FreeDesc(&d109)
				ctx.EnsureDesc(&d110)
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d108)
				ctx.EnsureDesc(&d110)
				d111 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = d111
				d112 := d3
				_ = d112
				ctx.StabilizeDescForControlFlow(&d112)
				inlineHome113 := d112
				inlineHome113.ID = 0
				d114 := d108
				_ = d114
				ctx.StabilizeDescForControlFlow(&d114)
				inlineHome115 := d114
				inlineHome115.ID = 0
				d116 := d110
				_ = d116
				ctx.StabilizeDescForControlFlow(&d116)
				inlineHome117 := d116
				inlineHome117.ID = 0
				ctx.StabilizeDescForControlFlow(&d3)
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
				bbpos_5_3 := int32(-1)
				_ = bbpos_5_3
				lbl52 := ctx.ReserveLabel()
				_ = lbl52
				bbpos_5_4 := int32(-1)
				_ = bbpos_5_4
				lbl53 := ctx.ReserveLabel()
				_ = lbl53
				bbpos_5_6 := int32(-1)
				_ = bbpos_5_6
				lbl54 := ctx.ReserveLabel()
				_ = lbl54
				bbpos_5_7 := int32(-1)
				_ = bbpos_5_7
				lbl55 := ctx.ReserveLabel()
				_ = lbl55
				bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl49)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d118 JITValueDesc
				ctx.EnsureDesc(&d112)
				if d112.Loc == LocImm {
					fieldAddr := uintptr(d112.Imm.Int()) + 24
					r51 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r51, fieldAddr)
					d118 = JITValueDesc{Loc: LocReg, Reg: r51}
					ctx.BindReg(r51, &d118)
				} else {
					off := int32(24)
					baseReg := d112.Reg
					r52 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r52, baseReg, off)
					d118 = JITValueDesc{Loc: LocReg, Reg: r52}
					ctx.BindReg(r52, &d118)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d118)
				var d119 JITValueDesc
				if d118.Loc == LocImm {
					d119 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d118.Imm.IsNil() == true)}
				} else {
					ctx.EnsureDesc(&d118)
					if d118.Loc != LocReg && d118.Loc != LocRegPair && d118.Loc != LocRegTriple {
						panic("jit: nil comparison requires a register value")
					}
					r53 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d118.Reg, 0)
					d119 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r53, Condition: CondEqual}
					ctx.BindReg(r53, &d119)
				}
				ctx.FreeDesc(&d118)
				ctx.ReclaimUntrackedRegs()
				d120 := d119
				ctx.SyncDesc(&d120)
				if d120.Loc != LocImm && d120.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl56 := ctx.ReserveLabel()
				lbl57 := ctx.ReserveLabel()
				if d120.Loc == LocImm {
					if d120.Imm.Bool() {
						ctx.MarkLabel(lbl56)
						ctx.EmitJmp(lbl50)
					} else {
						ctx.MarkLabel(lbl57)
						ctx.EmitJmp(lbl51)
					}
				} else {
					ctx.EmitJump(d120.Condition, lbl56)
					ctx.EmitJmp(lbl57)
					ctx.FreeDesc(&d119)
					ctx.MarkLabel(lbl56)
					ctx.EmitJmp(lbl50)
					ctx.MarkLabel(lbl57)
					ctx.EmitJmp(lbl51)
				}
				bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl51)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d114 = JITPrepareScmerGoArg(ctx, d114)
				ctx.SyncDesc(&d114)
				d121 := ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d114}, 1)
				d121.NoHeapPointer = true
				ctx.BindReg(d121.Reg, &d121)
				ctx.StabilizeDescForControlFlow(&d121)
				inlineHome122 := d121
				inlineHome122.ID = 0
				ctx.ReclaimUntrackedRegs()
				if d112.Loc == LocRegPair || d112.Loc == LocStackPair || d112.Loc == LocRegTriple || d112.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d114 = JITPrepareScmerGoArg(ctx, d114)
				if d121.Loc == LocRegPair || d121.Loc == LocStackPair || d121.Loc == LocRegTriple || d121.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d112)
				ctx.SyncDesc(&d114)
				ctx.SyncDesc(&d121)
				callResults123 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d112, d114, d121}, []uint8{1, 1}, []uint8{0, 0})
				d124 := callResults123[0]
				_ = d124
				d125 := callResults123[1]
				_ = d125
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d124)
				inlineHome126 := d124
				inlineHome126.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d127 := d125
				ctx.EnsureDesc(&d127)
				if d127.Loc != LocImm && d127.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl58 := ctx.ReserveLabel()
				lbl59 := ctx.ReserveLabel()
				if d127.Loc == LocImm {
					if d127.Imm.Bool() {
						ctx.MarkLabel(lbl58)
						ctx.EmitJmp(lbl52)
					} else {
						ctx.MarkLabel(lbl59)
						ctx.EmitJmp(lbl53)
					}
				} else {
					ctx.EmitCmpRegImm32(d127.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl58)
					ctx.EmitJmp(lbl59)
					ctx.MarkLabel(lbl58)
					ctx.EmitJmp(lbl52)
					ctx.MarkLabel(lbl59)
					ctx.EmitJmp(lbl53)
				}
				ctx.FreeDesc(&d125)
				bbpos_5_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl53)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				d121 = inlineHome122
				d124 = inlineHome126
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				if d112.Loc == LocRegPair || d112.Loc == LocStackPair || d112.Loc == LocRegTriple || d112.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				d114 = JITPrepareScmerGoArg(ctx, d114)
				d116 = JITPrepareScmerGoArg(ctx, d116)
				if d121.Loc == LocRegPair || d121.Loc == LocStackPair || d121.Loc == LocRegTriple || d121.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d112)
				ctx.SyncDesc(&d114)
				ctx.SyncDesc(&d116)
				ctx.SyncDesc(&d121)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).insertHashed), []JITValueDesc{d112, d114, d116, d121})
				ctx.FreeDesc(&d112)
				ctx.FreeDesc(&d121)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl48)
				bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl50)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				d121 = inlineHome122
				d124 = inlineHome126
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d128 := ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d128)
				ctx.EnsureDesc(&d112)
				ctx.EnsureDesc(&d128)
				ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d112, d128})
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl51)
				bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl52)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				d121 = inlineHome122
				d124 = inlineHome126
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d129 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d129
				ctx.ReclaimUntrackedRegs()
				bbpos_5_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl55)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				d121 = inlineHome122
				d124 = inlineHome126
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d130 JITValueDesc
				ctx.EnsureDesc(&d112)
				if d112.Loc == LocImm {
					fieldAddr := uintptr(d112.Imm.Int()) + 0
					r54 := ctx.AllocReg()
					r55 := ctx.AllocRegExcept(r54)
					r56 := ctx.AllocRegExcept(r54, r55)
					ctx.EmitMovRegMem64(r54, fieldAddr)
					ctx.EmitMovRegMem64(r55, fieldAddr+8)
					ctx.EmitMovRegMem64(r56, fieldAddr+16)
					d130 = JITValueDesc{Loc: LocRegTriple, Reg: r54, Reg2: r55, Reg3: r56}
					ctx.BindReg(r54, &d130)
					ctx.BindReg(r55, &d130)
					ctx.BindReg(r56, &d130)
				} else {
					off := int32(0)
					baseReg := d112.Reg
					r57 := ctx.AllocRegExcept(baseReg)
					r58 := ctx.AllocRegExcept(baseReg, r57)
					r59 := ctx.AllocRegExcept(baseReg, r57, r58)
					ctx.EmitMovRegMem(r57, baseReg, off)
					ctx.EmitMovRegMem(r58, baseReg, off+8)
					ctx.EmitMovRegMem(r59, baseReg, off+16)
					d130 = JITValueDesc{Loc: LocRegTriple, Reg: r57, Reg2: r58, Reg3: r59}
					ctx.BindReg(r57, &d130)
					ctx.BindReg(r58, &d130)
					ctx.BindReg(r59, &d130)
				}
				ctx.FreeDesc(&d112)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d124)
				ctx.EnsureDesc(&d124)
				var d131 JITValueDesc
				if d124.Loc == LocImm {
					d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d124.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d124.Reg)
					ctx.EmitMovRegReg(scratch, d124.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d131 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d131)
				}
				if d131.Loc == LocReg && d124.Loc == LocReg && d131.Reg == d124.Reg {
					ctx.TransferReg(d124.Reg)
					d124.Loc = LocNone
				}
				ctx.FreeDesc(&d124)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d131)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d116)
				ctx.SyncDesc(&d130)
				ctx.SyncDesc(&d131)
				d132 := d130
				d132.ID = 0
				d133 := d131
				d133.ID = 0
				if !ctx.TryEmitStoreScmerSliceElement(&d132, &d133, &d116, int32(16)) {
					ctx.StabilizeDescAcrossNestedCall(&d131)
					d133 = d131
					d133.ID = 0
					ctx.EmitStoreScmerSliceElement(&d132, &d133, &d116, int32(16))
				}
				ctx.FreeDesc(&d133)
				ctx.FreeDesc(&d131)
				ctx.ReclaimUntrackedRegs()
				bbpos_5_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl54)
				ctx.ResolveFixups()
				d112 = inlineHome113
				d114 = inlineHome115
				d116 = inlineHome117
				d121 = inlineHome122
				d124 = inlineHome126
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl48)
				ctx.MarkLabel(lbl48)
				var d134 JITValueDesc
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					panic("NewFastDict: LocImm not expected at JIT compile time")
				} else {
					r60 := ctx.AllocReg()
					ctx.EmitMovRegImm64(r60, makeAux(tagFastDict, 0))
					d134 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d3.Reg, Reg2: r60}
					ctx.BindReg(d3.Reg, &d134)
					ctx.BindReg(r60, &d134)
					ctx.TransferReg(d3.Reg)
					ctx.BindReg(d3.Reg, &d134)
					ctx.BindReg(r60, &d134)
					d3.Loc = LocNone
				}
				ctx.FreeDesc(&d3)
				if d134.Loc == LocImm {
					if result.Loc == LocAny {
						return d134
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d134)
				if d134.Loc == LocRegPair || d134.Loc == LocStackPair || d134.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d134, &result)
					result.Type = d134.Type
				} else {
					switch d134.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d134)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d134)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d134)
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
			JITInlineCost:  156,
		},
	})
}
