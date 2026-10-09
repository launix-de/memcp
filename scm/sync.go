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
	return NewFunc(func(a ...Scmer) (result Scmer) {
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
	})
}

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
	proc := *original
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
				var d29 JITValueDesc
				_ = d29
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
					ctx.EnsureDesc(&d2)
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
					d18 = d16
					_ = d18
					d19 = d17
					_ = d19
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d19.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d19.Reg, 8)
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
						ctx.BindReg(d19.Reg, &d20)
					}
					if d20.Loc == LocReg && d19.Loc == LocReg && d20.Reg == d19.Reg {
						ctx.TransferReg(d19.Reg)
						d19.Loc = LocNone
					}
					ctx.FreeDesc(&d19)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d18.Reg, int32(255))
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d18.Reg}
						ctx.BindReg(d18.Reg, &d21)
					}
					if d21.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: d21.Type, Imm: NewInt(int64(uint64(d21.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d21.Reg, 56)
						ctx.EmitShrRegImm8(d21.Reg, 56)
					}
					if d21.Loc == LocReg && d18.Loc == LocReg && d21.Reg == d18.Reg {
						ctx.TransferReg(d18.Reg)
						d18.Loc = LocNone
					}
					ctx.FreeDesc(&d18)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegReg(r4, d21.Reg)
						ctx.EmitShlRegImm8(r4, 56)
						ctx.EmitShrRegImm8(r4, 56)
						d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d22)
					}
					ctx.FreeDesc(&d21)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d22)
					if d20.Loc == LocImm && d22.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() | d22.Imm.Int())}
					} else if d20.Loc == LocImm && d20.Imm.Int() == 0 {
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg}
						ctx.BindReg(d22.Reg, &d23)
					} else if d22.Loc == LocImm && d22.Imm.Int() == 0 {
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg}
						ctx.BindReg(d20.Reg, &d23)
					} else if d20.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d22.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
						ctx.EmitOrInt64(scratch, d22.Reg)
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					} else if d22.Loc == LocImm {
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d20.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d22.Imm.Int()))
							ctx.EmitOrInt64(d20.Reg, ctx.ScratchReg)
						}
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg}
						ctx.BindReg(d20.Reg, &d23)
					} else {
						ctx.EmitOrInt64(d20.Reg, d22.Reg)
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg}
						ctx.BindReg(d20.Reg, &d23)
					}
					if d23.Loc == LocReg && d20.Loc == LocReg && d23.Reg == d20.Reg {
						ctx.TransferReg(d20.Reg)
						d20.Loc = LocNone
					}
					ctx.FreeDesc(&d20)
					ctx.FreeDesc(&d22)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d12)
					ctx.EmitMovToReg(d11.Reg, d12)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d23)
					ctx.EmitMovToReg(d15.Reg2, d23)
					ctx.FreeDesc(&d23)
					d24 = d10
					_ = d24
					ctx.SyncDesc(&d24)
					if d24.Loc == LocRegPair || d24.Loc == LocStackPair || d24.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d24, &result)
						result.Type = d24.Type
					} else {
						switch d24.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d24)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d24)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d24)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d24, &result)
							result.Type = d24.Type
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
					d25 = args[0]
					d25.ID = 0
					d26 = jitKnownSliceHeader(ctx, &d25)
					ctx.StabilizeDescForControlFlow(&d26)
					ctx.FreeDesc(&d25)
					if d26.SliceSizeKnown {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d26.KnownSliceLen))}
					} else if d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d26.StackOff))}
					} else if d26.Loc == LocStackTriple {
						d27 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d26.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d26)
						if d26.Loc == LocRegPair || d26.Loc == LocRegTriple {
							d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg2, ID: 0}
						} else if d26.Loc == LocReg {
							d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						d28 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() < 2)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d27.Reg, 2)
						d28 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d28)
					}
					ctx.FreeDesc(&d27)
					d29 = d28
					ctx.EnsureDesc(&d29)
					if d29.Loc != LocImm && d29.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d29.Loc == LocImm {
						if d29.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitJump(d29.Condition, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FreeDesc(&d28)
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap30 := d0
						snap31 := d1
						snap32 := d2
						snap33 := d8
						snap34 := d9
						snap35 := d10
						snap36 := d11
						snap37 := d12
						snap38 := d13
						snap39 := d14
						snap40 := d15
						snap41 := d16
						snap42 := d17
						snap43 := d18
						snap44 := d19
						snap45 := d20
						snap46 := d21
						snap47 := d22
						snap48 := d23
						snap49 := d24
						snap50 := d25
						snap51 := d26
						snap52 := d27
						snap53 := d28
						snap54 := d29
						alloc55 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc55)
						d0 = snap30
						d1 = snap31
						d2 = snap32
						d8 = snap33
						d9 = snap34
						d10 = snap35
						d11 = snap36
						d12 = snap37
						d13 = snap38
						d14 = snap39
						d15 = snap40
						d16 = snap41
						d17 = snap42
						d18 = snap43
						d19 = snap44
						d20 = snap45
						d21 = snap46
						d22 = snap47
						d23 = snap48
						d24 = snap49
						d25 = snap50
						d26 = snap51
						d27 = snap52
						d28 = snap53
						d29 = snap54
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
					d56 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d56)
					d58 = d26
					d58.ID = 0
					d59 = d57
					d59.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d58, &d59, &d56, int32(16)) {
						ctx.EmitStoreScmerSliceElement(&d58, &d59, &d56, int32(16))
					}
					ctx.FreeDesc(&d59)
					ctx.FreeDesc(&d56)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegImm64(r6, 0)
					ctx.EmitMovRegImm64(r7, 0)
					d60 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d60)
					ctx.BindReg(r7, &d60)
					d61 = args[0]
					d61.ID = 0
					d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d63 = ctx.EmitSliceElementAddress(&d26, &d62, int32(16))
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					d66 = args[0]
					d66.ID = 0
					d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d69 = d67
					_ = d69
					d70 = d68
					_ = d70
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocImm {
						d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d70.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d70.Reg, 8)
						d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d70.Reg}
						ctx.BindReg(d70.Reg, &d71)
					}
					if d71.Loc == LocReg && d70.Loc == LocReg && d71.Reg == d70.Reg {
						ctx.TransferReg(d70.Reg)
						d70.Loc = LocNone
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					if d69.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d69.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d69.Reg, int32(255))
						d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg}
						ctx.BindReg(d69.Reg, &d72)
					}
					if d72.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: d72.Type, Imm: NewInt(int64(uint64(d72.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d72.Reg, 56)
						ctx.EmitShrRegImm8(d72.Reg, 56)
					}
					if d72.Loc == LocReg && d69.Loc == LocReg && d72.Reg == d69.Reg {
						ctx.TransferReg(d69.Reg)
						d69.Loc = LocNone
					}
					ctx.FreeDesc(&d69)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegReg(r8, d72.Reg)
						ctx.EmitShlRegImm8(r8, 56)
						ctx.EmitShrRegImm8(r8, 56)
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d73)
					}
					ctx.FreeDesc(&d72)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d73)
					if d71.Loc == LocImm && d73.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() | d73.Imm.Int())}
					} else if d71.Loc == LocImm && d71.Imm.Int() == 0 {
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d73.Reg}
						ctx.BindReg(d73.Reg, &d74)
					} else if d73.Loc == LocImm && d73.Imm.Int() == 0 {
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d74)
					} else if d71.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d71.Imm.Int()))
						ctx.EmitOrInt64(scratch, d73.Reg)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d74)
					} else if d73.Loc == LocImm {
						if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d71.Reg, int32(d73.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d73.Imm.Int()))
							ctx.EmitOrInt64(d71.Reg, ctx.ScratchReg)
						}
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d74)
					} else {
						ctx.EmitOrInt64(d71.Reg, d73.Reg)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
						ctx.BindReg(d71.Reg, &d74)
					}
					if d74.Loc == LocReg && d71.Loc == LocReg && d74.Reg == d71.Reg {
						ctx.TransferReg(d71.Reg)
						d71.Loc = LocNone
					}
					ctx.FreeDesc(&d71)
					ctx.FreeDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					ctx.EmitMovToReg(d61.Reg, d63)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.EmitMovToReg(d66.Reg2, d74)
					ctx.FreeDesc(&d74)
					d75 = d60
					_ = d75
					ctx.SyncDesc(&d75)
					if d75.Loc == LocRegPair || d75.Loc == LocStackPair || d75.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d75, &result)
						result.Type = d75.Type
					} else {
						switch d75.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d75)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d75)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d75)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d75, &result)
							result.Type = d75.Type
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
					ctx.EnsureDesc(&d2)
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
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d2}, 1)
				d4 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("alloc")}
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
				d7 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				d4 = JITPrepareScmerGoArg(ctx, d4)
				d6 = JITPrepareScmerGoArg(ctx, d6)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d3, d4, d6, d7})
				d8 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("total_alloc")}
				var d9 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 8
					r3 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r3, fieldAddr)
					d9 = JITValueDesc{Loc: LocReg, Reg: r3}
					ctx.BindReg(r3, &d9)
				} else {
					off := int32(8)
					baseReg := d0.Reg
					r4 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r4, baseReg, off)
					d9 = JITValueDesc{Loc: LocReg, Reg: r4}
					ctx.BindReg(r4, &d9)
				}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d9)
				var d10 JITValueDesc
				if d9.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d9.Imm.Int()))))}
				} else {
					r5 := ctx.AllocReg()
					ctx.EmitMovRegReg(r5, d9.Reg)
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d10)
				}
				ctx.FreeDesc(&d9)
				ctx.EnsureDesc(&d10)
				d11 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				d8 = JITPrepareScmerGoArg(ctx, d8)
				d10 = JITPrepareScmerGoArg(ctx, d10)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d3, d8, d10, d11})
				d12 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("sys")}
				var d13 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 16
					r6 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r6, fieldAddr)
					d13 = JITValueDesc{Loc: LocReg, Reg: r6}
					ctx.BindReg(r6, &d13)
				} else {
					off := int32(16)
					baseReg := d0.Reg
					r7 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r7, baseReg, off)
					d13 = JITValueDesc{Loc: LocReg, Reg: r7}
					ctx.BindReg(r7, &d13)
				}
				ctx.EnsureDesc(&d13)
				ctx.EnsureDesc(&d13)
				var d14 JITValueDesc
				if d13.Loc == LocImm {
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d13.Imm.Int()))))}
				} else {
					r8 := ctx.AllocReg()
					ctx.EmitMovRegReg(r8, d13.Reg)
					d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
					ctx.BindReg(r8, &d14)
				}
				ctx.FreeDesc(&d13)
				ctx.EnsureDesc(&d14)
				d15 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				d12 = JITPrepareScmerGoArg(ctx, d12)
				d14 = JITPrepareScmerGoArg(ctx, d14)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d3, d12, d14, d15})
				d16 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("heap_alloc")}
				var d17 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 48
					r9 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r9, fieldAddr)
					d17 = JITValueDesc{Loc: LocReg, Reg: r9}
					ctx.BindReg(r9, &d17)
				} else {
					off := int32(48)
					baseReg := d0.Reg
					r10 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r10, baseReg, off)
					d17 = JITValueDesc{Loc: LocReg, Reg: r10}
					ctx.BindReg(r10, &d17)
				}
				ctx.EnsureDesc(&d17)
				ctx.EnsureDesc(&d17)
				var d18 JITValueDesc
				if d17.Loc == LocImm {
					d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d17.Imm.Int()))))}
				} else {
					r11 := ctx.AllocReg()
					ctx.EmitMovRegReg(r11, d17.Reg)
					d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d18)
				}
				ctx.FreeDesc(&d17)
				ctx.EnsureDesc(&d18)
				d19 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				d16 = JITPrepareScmerGoArg(ctx, d16)
				d18 = JITPrepareScmerGoArg(ctx, d18)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d3, d16, d18, d19})
				d20 := JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("heap_sys")}
				var d21 JITValueDesc
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					fieldAddr := uintptr(d0.Imm.Int()) + 56
					r12 := ctx.AllocReg()
					ctx.EmitMovRegMem64(r12, fieldAddr)
					d21 = JITValueDesc{Loc: LocReg, Reg: r12}
					ctx.BindReg(r12, &d21)
				} else {
					off := int32(56)
					baseReg := d0.Reg
					r13 := ctx.AllocRegExcept(baseReg)
					ctx.EmitMovRegMem(r13, baseReg, off)
					d21 = JITValueDesc{Loc: LocReg, Reg: r13}
					ctx.BindReg(r13, &d21)
				}
				ctx.EnsureDesc(&d21)
				ctx.EnsureDesc(&d21)
				var d22 JITValueDesc
				if d21.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d21.Imm.Int()))))}
				} else {
					r14 := ctx.AllocReg()
					ctx.EmitMovRegReg(r14, d21.Reg)
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
					ctx.BindReg(r14, &d22)
				}
				ctx.FreeDesc(&d21)
				ctx.EnsureDesc(&d22)
				d23 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				d20 = JITPrepareScmerGoArg(ctx, d20)
				d22 = JITPrepareScmerGoArg(ctx, d22)
				ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d3, d20, d22, d23})
				var d24 JITValueDesc
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					panic("NewFastDict: LocImm not expected at JIT compile time")
				} else {
					r15 := ctx.AllocReg()
					ctx.EmitMovRegImm64(r15, makeAux(tagFastDict, 0))
					d24 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d3.Reg, Reg2: r15}
					ctx.BindReg(d3.Reg, &d24)
					ctx.BindReg(r15, &d24)
					ctx.TransferReg(d3.Reg)
					ctx.BindReg(d3.Reg, &d24)
					ctx.BindReg(r15, &d24)
					d3.Loc = LocNone
				}
				ctx.FreeDesc(&d3)
				if d24.Loc == LocImm {
					if result.Loc == LocAny {
						return d24
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d24)
				if d24.Loc == LocRegPair || d24.Loc == LocStackPair || d24.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d24, &result)
					result.Type = d24.Type
				} else {
					switch d24.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d24)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d24)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d24)
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
			JITInlineCost:  36,
		},
	})
}
