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
			sess.Mu.Lock()
			defer sess.Mu.Unlock()
			if a[0].GetTag() >= 100 {
				if sess.Handles == nil {
					sess.Handles = make(map[Scmer]Scmer)
				}
				sess.Handles[a[0]] = a[1]
			} else {
				sess.Map[a[0].String()] = a[1]
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
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			if a[0].GetTag() >= 100 {
				if v, ok := sess.Handles[a[0]]; ok {
					return v
				}
				return NewNil()
			}
			if v, ok := sess.Map[a[0].String()]; ok {
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

func init_sync() {
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
				var stackArray11 int32
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
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [5]BBDescriptor
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
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
					var d1 JITValueDesc
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							if ps.General {
							}
							ps3 := PhiState{General: ps.General}
							ps3.OverlayValues = make([]JITValueDesc, 3)
							ps3.OverlayValues[0] = d0
							ps3.OverlayValues[1] = d1
							ps3.OverlayValues[2] = d2
							return bbs[1].RenderPS(ps3)
						}
						if ps.General {
						}
						ps4 := PhiState{General: ps.General}
						ps4.OverlayValues = make([]JITValueDesc, 3)
						ps4.OverlayValues[0] = d0
						ps4.OverlayValues[1] = d1
						ps4.OverlayValues[2] = d2
						return bbs[2].RenderPS(ps4)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl3)
					ps5 := PhiState{General: true}
					ps5.OverlayValues = make([]JITValueDesc, 3)
					ps5.OverlayValues[0] = d0
					ps5.OverlayValues[1] = d1
					ps5.OverlayValues[2] = d2
					ps6 := PhiState{General: true}
					ps6.OverlayValues = make([]JITValueDesc, 3)
					ps6.OverlayValues[0] = d0
					ps6.OverlayValues[1] = d1
					ps6.OverlayValues[2] = d2
					snap7 := d0
					snap8 := d1
					snap9 := d2
					alloc10 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps6)
					}
					ctx.RestoreAllocState(alloc10)
					d0 = snap7
					d1 = snap8
					d2 = snap9
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps5)
					}
					return result
					ctx.FreeDesc(&d1)
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					stackArray11 = ctx.AllocStack(int32(32))
					_ = stackArray11
					d12 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d12
					d13 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d13)
					ctx.EmitStoreScmerToStack(d13, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d13)
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d14 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r2, &d14)
					d15 = args[0]
					d15.ID = 0
					r3 := ctx.AllocReg()
					ctx.EmitLeaRegMem(r3, ctx.StackReg, int32(stackArray11)+int32(0))
					d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, NoHeapPointer: true}
					ctx.BindReg(r3, &d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					d19 = args[0]
					d19.ID = 0
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d22 = d20
					_ = d22
					ctx.StabilizeDescForControlFlow(&d22)
					d23 = d21
					_ = d23
					ctx.StabilizeDescForControlFlow(&d23)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					var d24 JITValueDesc
					if d23.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d23.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d23.Reg, 8)
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg}
						ctx.BindReg(d23.Reg, &d24)
					}
					if d24.Loc == LocReg && d23.Loc == LocReg && d24.Reg == d23.Reg {
						ctx.TransferReg(d23.Reg)
						d23.Loc = LocNone
					}
					ctx.FreeDesc(&d23)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d22)
					var d25 JITValueDesc
					if d22.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d22.Reg, int32(255))
						d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg}
						ctx.BindReg(d22.Reg, &d25)
					}
					if d25.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: d25.Type, Imm: NewInt(int64(uint64(d25.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d25.Reg, 56)
						ctx.EmitShrRegImm8(d25.Reg, 56)
					}
					if d25.Loc == LocReg && d22.Loc == LocReg && d25.Reg == d22.Reg {
						ctx.TransferReg(d22.Reg)
						d22.Loc = LocNone
					}
					ctx.FreeDesc(&d22)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d25)
					ctx.EnsureDesc(&d25)
					var d26 JITValueDesc
					if d25.Loc == LocImm {
						d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d25.Imm.Int()))))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegReg(r4, d25.Reg)
						ctx.EmitShlRegImm8(r4, 56)
						ctx.EmitShrRegImm8(r4, 56)
						d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d26)
					}
					ctx.FreeDesc(&d25)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d26)
					var d27 JITValueDesc
					if d24.Loc == LocImm && d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d24.Imm.Int() | d26.Imm.Int())}
					} else if d24.Loc == LocImm && d24.Imm.Int() == 0 {
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d26.Reg}
						ctx.BindReg(d26.Reg, &d27)
					} else if d26.Loc == LocImm && d26.Imm.Int() == 0 {
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
						ctx.BindReg(d24.Reg, &d27)
					} else if d24.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d26.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
						ctx.EmitOrInt64(scratch, d26.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d27)
					} else if d26.Loc == LocImm {
						if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d24.Reg, int32(d26.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d26.Imm.Int()))
							ctx.EmitOrInt64(d24.Reg, RegR11)
						}
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
						ctx.BindReg(d24.Reg, &d27)
					} else {
						ctx.EmitOrInt64(d24.Reg, d26.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
						ctx.BindReg(d24.Reg, &d27)
					}
					if d27.Loc == LocReg && d24.Loc == LocReg && d27.Reg == d24.Reg {
						ctx.TransferReg(d24.Reg)
						d24.Loc = LocNone
					}
					ctx.FreeDesc(&d24)
					ctx.FreeDesc(&d26)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					ctx.EmitMovToReg(d15.Reg, d16)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.EmitMovToReg(d19.Reg2, d27)
					ctx.FreeDesc(&d27)
					d28 = d14
					_ = d28
					ctx.SyncDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocStackPair || d28.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d28, &result)
						result.Type = d28.Type
					} else {
						switch d28.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d28)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d28)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d28)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d28, &result)
							result.Type = d28.Type
						}
					}
					ctx.EmitJmp(lbl0)
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					ctx.ReclaimUntrackedRegs()
					d29 = args[0]
					d29.ID = 0
					d30 = jitKnownSliceHeader(ctx, &d29)
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d29)
					var d31 JITValueDesc
					if d30.SliceSizeKnown {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.KnownSliceLen))}
					} else if d30.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.StackOff))}
					} else if d30.Loc == LocStackTriple {
						d31 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d30.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d30)
						if d30.Loc == LocRegPair || d30.Loc == LocRegTriple {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2, ID: 0}
						} else if d30.Loc == LocReg {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d31)
					var d32 JITValueDesc
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() < 2)}
					} else {
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d31.Reg, 2)
						ctx.EmitSetcc(r5, CondSignedLess)
						d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d32)
					}
					ctx.FreeDesc(&d31)
					d33 = d32
					ctx.EnsureDesc(&d33)
					if d33.Loc != LocImm && d33.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d33.Loc == LocImm {
						if d33.Imm.Bool() {
							if ps.General {
							}
							ps34 := PhiState{General: ps.General}
							ps34.OverlayValues = make([]JITValueDesc, 34)
							ps34.OverlayValues[0] = d0
							ps34.OverlayValues[1] = d1
							ps34.OverlayValues[2] = d2
							ps34.OverlayValues[12] = d12
							ps34.OverlayValues[13] = d13
							ps34.OverlayValues[14] = d14
							ps34.OverlayValues[15] = d15
							ps34.OverlayValues[16] = d16
							ps34.OverlayValues[17] = d17
							ps34.OverlayValues[18] = d18
							ps34.OverlayValues[19] = d19
							ps34.OverlayValues[20] = d20
							ps34.OverlayValues[21] = d21
							ps34.OverlayValues[22] = d22
							ps34.OverlayValues[23] = d23
							ps34.OverlayValues[24] = d24
							ps34.OverlayValues[25] = d25
							ps34.OverlayValues[26] = d26
							ps34.OverlayValues[27] = d27
							ps34.OverlayValues[28] = d28
							ps34.OverlayValues[29] = d29
							ps34.OverlayValues[30] = d30
							ps34.OverlayValues[31] = d31
							ps34.OverlayValues[32] = d32
							ps34.OverlayValues[33] = d33
							return bbs[3].RenderPS(ps34)
						}
						if ps.General {
						}
						ps35 := PhiState{General: ps.General}
						ps35.OverlayValues = make([]JITValueDesc, 34)
						ps35.OverlayValues[0] = d0
						ps35.OverlayValues[1] = d1
						ps35.OverlayValues[2] = d2
						ps35.OverlayValues[12] = d12
						ps35.OverlayValues[13] = d13
						ps35.OverlayValues[14] = d14
						ps35.OverlayValues[15] = d15
						ps35.OverlayValues[16] = d16
						ps35.OverlayValues[17] = d17
						ps35.OverlayValues[18] = d18
						ps35.OverlayValues[19] = d19
						ps35.OverlayValues[20] = d20
						ps35.OverlayValues[21] = d21
						ps35.OverlayValues[22] = d22
						ps35.OverlayValues[23] = d23
						ps35.OverlayValues[24] = d24
						ps35.OverlayValues[25] = d25
						ps35.OverlayValues[26] = d26
						ps35.OverlayValues[27] = d27
						ps35.OverlayValues[28] = d28
						ps35.OverlayValues[29] = d29
						ps35.OverlayValues[30] = d30
						ps35.OverlayValues[31] = d31
						ps35.OverlayValues[32] = d32
						ps35.OverlayValues[33] = d33
						return bbs[4].RenderPS(ps35)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d33.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl5)
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 34)
					ps36.OverlayValues[0] = d0
					ps36.OverlayValues[1] = d1
					ps36.OverlayValues[2] = d2
					ps36.OverlayValues[12] = d12
					ps36.OverlayValues[13] = d13
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[16] = d16
					ps36.OverlayValues[17] = d17
					ps36.OverlayValues[18] = d18
					ps36.OverlayValues[19] = d19
					ps36.OverlayValues[20] = d20
					ps36.OverlayValues[21] = d21
					ps36.OverlayValues[22] = d22
					ps36.OverlayValues[23] = d23
					ps36.OverlayValues[24] = d24
					ps36.OverlayValues[25] = d25
					ps36.OverlayValues[26] = d26
					ps36.OverlayValues[27] = d27
					ps36.OverlayValues[28] = d28
					ps36.OverlayValues[29] = d29
					ps36.OverlayValues[30] = d30
					ps36.OverlayValues[31] = d31
					ps36.OverlayValues[32] = d32
					ps36.OverlayValues[33] = d33
					ps37 := PhiState{General: true}
					ps37.OverlayValues = make([]JITValueDesc, 34)
					ps37.OverlayValues[0] = d0
					ps37.OverlayValues[1] = d1
					ps37.OverlayValues[2] = d2
					ps37.OverlayValues[12] = d12
					ps37.OverlayValues[13] = d13
					ps37.OverlayValues[14] = d14
					ps37.OverlayValues[15] = d15
					ps37.OverlayValues[16] = d16
					ps37.OverlayValues[17] = d17
					ps37.OverlayValues[18] = d18
					ps37.OverlayValues[19] = d19
					ps37.OverlayValues[20] = d20
					ps37.OverlayValues[21] = d21
					ps37.OverlayValues[22] = d22
					ps37.OverlayValues[23] = d23
					ps37.OverlayValues[24] = d24
					ps37.OverlayValues[25] = d25
					ps37.OverlayValues[26] = d26
					ps37.OverlayValues[27] = d27
					ps37.OverlayValues[28] = d28
					ps37.OverlayValues[29] = d29
					ps37.OverlayValues[30] = d30
					ps37.OverlayValues[31] = d31
					ps37.OverlayValues[32] = d32
					ps37.OverlayValues[33] = d33
					snap38 := d0
					snap39 := d1
					snap40 := d2
					snap41 := d12
					snap42 := d13
					snap43 := d14
					snap44 := d15
					snap45 := d16
					snap46 := d17
					snap47 := d18
					snap48 := d19
					snap49 := d20
					snap50 := d21
					snap51 := d22
					snap52 := d23
					snap53 := d24
					snap54 := d25
					snap55 := d26
					snap56 := d27
					snap57 := d28
					snap58 := d29
					snap59 := d30
					snap60 := d31
					snap61 := d32
					snap62 := d33
					alloc63 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps37)
					}
					ctx.RestoreAllocState(alloc63)
					d0 = snap38
					d1 = snap39
					d2 = snap40
					d12 = snap41
					d13 = snap42
					d14 = snap43
					d15 = snap44
					d16 = snap45
					d17 = snap46
					d18 = snap47
					d19 = snap48
					d20 = snap49
					d21 = snap50
					d22 = snap51
					d23 = snap52
					d24 = snap53
					d25 = snap54
					d26 = snap55
					d27 = snap56
					d28 = snap57
					d29 = snap58
					d30 = snap59
					d31 = snap60
					d32 = snap61
					d33 = snap62
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps36)
					}
					return result
					ctx.FreeDesc(&d32)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["newpromise"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					ctx.ReclaimUntrackedRegs()
					d64 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.SyncDesc(&d64)
					d66 = d30
					d66.ID = 0
					d67 = d65
					d67.ID = 0
					d68 = ctx.EmitSliceElementAddress(&d66, &d67, int32(16))
					ctx.FreeDesc(&d67)
					ctx.EmitStoreScmerAt(&d68, &d64)
					ctx.FreeDesc(&d68)
					ctx.FreeDesc(&d64)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegImm64(r6, 0)
					ctx.EmitMovRegImm64(r7, 0)
					d69 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d69)
					ctx.BindReg(r7, &d69)
					d70 = args[0]
					d70.ID = 0
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d72 = ctx.EmitSliceElementAddress(&d30, &d71, int32(16))
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					d75 = args[0]
					d75.ID = 0
					d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(22)}
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d78 = d76
					_ = d78
					ctx.StabilizeDescForControlFlow(&d78)
					d79 = d77
					_ = d79
					ctx.StabilizeDescForControlFlow(&d79)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d79)
					var d80 JITValueDesc
					if d79.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d79.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d79.Reg, 8)
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d79.Reg}
						ctx.BindReg(d79.Reg, &d80)
					}
					if d80.Loc == LocReg && d79.Loc == LocReg && d80.Reg == d79.Reg {
						ctx.TransferReg(d79.Reg)
						d79.Loc = LocNone
					}
					ctx.FreeDesc(&d79)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					var d81 JITValueDesc
					if d78.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d78.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d78.Reg, int32(255))
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d78.Reg}
						ctx.BindReg(d78.Reg, &d81)
					}
					if d81.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: d81.Type, Imm: NewInt(int64(uint64(d81.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d81.Reg, 56)
						ctx.EmitShrRegImm8(d81.Reg, 56)
					}
					if d81.Loc == LocReg && d78.Loc == LocReg && d81.Reg == d78.Reg {
						ctx.TransferReg(d78.Reg)
						d78.Loc = LocNone
					}
					ctx.FreeDesc(&d78)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					var d82 JITValueDesc
					if d81.Loc == LocImm {
						d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d81.Imm.Int()))))}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegReg(r8, d81.Reg)
						ctx.EmitShlRegImm8(r8, 56)
						ctx.EmitShrRegImm8(r8, 56)
						d82 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
						ctx.BindReg(r8, &d82)
					}
					ctx.FreeDesc(&d81)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d82)
					var d83 JITValueDesc
					if d80.Loc == LocImm && d82.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d80.Imm.Int() | d82.Imm.Int())}
					} else if d80.Loc == LocImm && d80.Imm.Int() == 0 {
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d82.Reg}
						ctx.BindReg(d82.Reg, &d83)
					} else if d82.Loc == LocImm && d82.Imm.Int() == 0 {
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d80.Reg}
						ctx.BindReg(d80.Reg, &d83)
					} else if d80.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d82.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d80.Imm.Int()))
						ctx.EmitOrInt64(scratch, d82.Reg)
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d83)
					} else if d82.Loc == LocImm {
						if d82.Imm.Int() >= -2147483648 && d82.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d80.Reg, int32(d82.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d82.Imm.Int()))
							ctx.EmitOrInt64(d80.Reg, RegR11)
						}
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d80.Reg}
						ctx.BindReg(d80.Reg, &d83)
					} else {
						ctx.EmitOrInt64(d80.Reg, d82.Reg)
						d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d80.Reg}
						ctx.BindReg(d80.Reg, &d83)
					}
					if d83.Loc == LocReg && d80.Loc == LocReg && d83.Reg == d80.Reg {
						ctx.TransferReg(d80.Reg)
						d80.Loc = LocNone
					}
					ctx.FreeDesc(&d80)
					ctx.FreeDesc(&d82)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					ctx.EmitMovToReg(d70.Reg, d72)
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d83)
					ctx.EmitMovToReg(d75.Reg2, d83)
					ctx.FreeDesc(&d83)
					d84 = d69
					_ = d84
					ctx.SyncDesc(&d84)
					if d84.Loc == LocRegPair || d84.Loc == LocStackPair || d84.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d84, &result)
						result.Type = d84.Type
					} else {
						switch d84.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d84)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d84)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d84)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d84, &result)
							result.Type = d84.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps85 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps85)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
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
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
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
				ctx.Coverage.NativeCalls++
				declaration := declarations["context"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
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
