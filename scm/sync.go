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
import "github.com/jtolds/gls"

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

func sessionEnsureScopedCleanup(sess *session, scope Scmer) {
	if sess.ScopedCleanup[scope] {
		return
	}
	value, ok := GetGLSValue("context")
	if !ok {
		return
	}
	ctx, ok := value.(context.Context)
	if !ok || ctx.Done() == nil {
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

func sessionGetOrComputeScoped(sess *session, scope Scmer, key string, producer Scmer) Scmer {
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
	sessionEnsureScopedCleanup(sess, scope)
	if flight := sess.ScopedFlights[computeKey]; flight != nil {
		sess.Mu.Unlock()
		ctx := context.Background()
		if value, ok := GetGLSValue("context"); ok {
			if current, ok := value.(context.Context); ok {
				ctx = current
			}
		}
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
		flight.value = Apply(producer)
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
			return sessionGetOrComputeScoped(sess, a[1], a[2].String(), a[3])
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
			panic("wrong number of parameters provided to session: 0, 1, 2, or 4 required")
		}
	})
}

var sessionCallableType = &TypeDescriptor{Kind: "func", Label: "session", Description: "session accessor accepting exactly zero, one, two, or four arguments", HasSideEffects: true,
	Params: []*TypeDescriptor{
		{Kind: "any", Label: "key_or_operation", Description: "key, or get_or_compute_scoped", Optional: true},
		{Kind: "any", Label: "value_or_scope", Description: "value to store, or scope for get_or_compute_scoped", Optional: true},
		{Kind: "any", Label: "scoped_key", Description: "cache key used by get_or_compute_scoped", Optional: true},
		{Kind: "func", CallsOnce: true, Label: "scoped_producer", Description: "producer used only by the four-argument get_or_compute_scoped form", Optional: true, Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any", Label: "value", Description: "computed value cached for the scope and key"}},
	},
	Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "value list, stored value, retrieved value, or shared computed value"},
}

var mgr *gls.ContextManager

func Context(a ...Scmer) (result Scmer) {
	if mgr == nil {
		// prone to race conditions, to the first call should be called in the initialization
		mgr = gls.NewContextManager()
	}
	if a[0].IsString() {
		switch a[0].String() {
		case "session":
			val, ok := mgr.GetValue("session")
			if !ok {
				panic("no session set")
			}
			return val.(Scmer)
		case "query":
			return NewInt(int64(CurrentQuerySeq()))
		case "check":
			ctxVal, ok := mgr.GetValue("context")
			if !ok {
				// Startup compilation and Scheme self-tests have no request to
				// cancel. Treat that environment as live while preserving prompt
				// cancellation whenever a request context exists.
				return NewBool(true)
			}
			e := ctxVal.(context.Context).Err()
			if e != nil {
				panic(e)
			}
			return NewBool(true)
		}
	}
	if !a[0].IsNil() {
		NewContext(context.TODO(), func() {
			result = Apply(a[0], a[1:]...)
		})
		return result
	}
	panic("unimplemented")
}

func NewContext(ctx context.Context, fn func()) {
	if mgr == nil {
		// prone to race conditions, to the first call should be called in the initialization
		mgr = gls.NewContextManager()
	}
	mgr.SetValues(gls.Values{
		"session": NewSession(),
		"context": ctx,
		// TODO: logger for print and time, process ID etc. etc.
	}, fn)
}

// NewContextWithSession installs a pre-existing Scheme session and any
// request-local values in one GLS frame. Keeping one frame avoids repeating
// goroutine stack tagging for every HTTP request.
func NewContextWithSession(ctx context.Context, session Scmer, values map[string]any, fn func()) {
	if mgr == nil {
		mgr = gls.NewContextManager()
	}
	glsValues := gls.Values{
		"session": session,
		"context": ctx,
	}
	for key, value := range values {
		glsValues[key] = value
	}
	mgr.SetValues(glsValues, fn)
}

func GetContext() context.Context {
	if mgr == nil {
		// prone to race conditions, to the first call should be called in the initialization
		mgr = gls.NewContextManager()
	}
	r, ok := mgr.GetValue("context")
	if !ok {
		panic("no context set")
	}
	return r.(context.Context)
}

// GetCurrentTx returns the current transaction context by looking up the
// session from GLS and reading the "__memcp_tx" key. Returns nil if no
// transaction is active or no session is available.
func GetCurrentTx() any {
	if mgr == nil {
		return nil
	}
	val, ok := mgr.GetValue("session")
	if !ok {
		return nil
	}
	sessionScmer := val.(Scmer)
	txScmer := Apply(sessionScmer, NewString("__memcp_tx"))
	if txScmer.IsNil() {
		return nil
	}
	return txScmer.Any()
}

// SetValues wraps mgr.SetValues for use by other packages (e.g. MySQL
// frontend) that need to install session/context into GLS.
func SetValues(vals map[string]any, fn func()) {
	if mgr == nil {
		mgr = gls.NewContextManager()
	}
	glsVals := make(gls.Values, len(vals))
	for k, v := range vals {
		glsVals[k] = v
	}
	mgr.SetValues(glsVals, fn)
}

// GetGLSValue returns the GLS value for a given key, or nil if no GLS context
// is installed. Used by packages outside scm to read goroutine-local markers
// that propagate across gls.Go-spawned worker goroutines.
func GetGLSValue(key string) (any, bool) {
	if mgr == nil {
		return nil, false
	}
	return mgr.GetValue(key)
}

// WithSession executes fn with the given session installed in GLS,
// so that GetCurrentTx() and other GLS-based lookups use this session.
func WithSession(session Scmer, fn Scmer) Scmer {
	var result Scmer
	if mgr == nil {
		mgr = gls.NewContextManager()
	}
	mgr.SetValues(gls.Values{"session": session}, func() {
		result = Apply(fn)
	})
	return result
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
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["newpromise"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "newsession",

		Fn: NewSession,
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a thread-safe key-value session. Call it without arguments to list values, with a key to read, with a key and value to store, or with get_or_compute_scoped, a scope, a key, and a producer to share one concurrent computation.",
			Return: sessionCallableType,
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["newsession"].Fn, args, result)
			},
			JITVirtualArgs: true,
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d2 := args[1]
				d2.ID = 0
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d1.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
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
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (WithSession arg0)")
				}
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d2.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d2 = tmpPair
				} else if d2.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d2)
					d2 = tmpPair
				}
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (WithSession arg1)")
				}
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(WithSession), []JITValueDesc{d1, d2}, 2)
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d2)
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
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocRegPair {
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
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
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
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["context"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sleep",

		Fn: func(a ...Scmer) Scmer {
			ctx := GetContext()
			select {
			case <-ctx.Done():
				panic(ctx.Err())
			case <-time.After(time.Duration(ToFloat(a[0]) * float64(time.Second))):
				return NewBool(true)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "sleeps the amount of seconds",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "duration", Description: "number of seconds to sleep"},
			},
			Return: &TypeDescriptor{Kind: "bool"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["sleep"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "once",

		Fn: func(a ...Scmer) Scmer {
			var params []Scmer
			once := sync.OnceValue[Scmer](func() Scmer {
				return Apply(a[0], params...)
			})
			return NewFunc(func(a ...Scmer) Scmer {
				params = a
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["once"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mutex",

		Fn: func(a ...Scmer) Scmer {
			token := make(chan struct{}, 1)
			token <- struct{}{}
			return NewFunc(func(a ...Scmer) Scmer {
				ctx := context.Background()
				if value, ok := GetGLSValue("context"); ok {
					if current, ok := value.(context.Context); ok {
						ctx = current
					}
				}
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
				return Apply(a[0])
			})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "Creates a context-aware mutex. The return value serializes calls to parameterless functions and stops waiting when the current request is cancelled.",
			Params: []*TypeDescriptor{},
			Return: &TypeDescriptor{Kind: "func", Label: "locked", Description: "executes one parameterless function while holding the mutex", HasSideEffects: true,
				Params: []*TypeDescriptor{
					{Kind: "func", CallsOnce: true, Label: "fn", Description: "parameterless function to execute under the lock", Params: []*TypeDescriptor{}, Return: &TypeDescriptor{Kind: "any", Label: "result"}},
				},
				Return: &TypeDescriptor{Kind: "any", Label: "result", Description: "result returned by the protected function"},
			},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["mutex"].Fn, args, result)
			},
			JITVirtualArgs: true,
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := ctx.EmitGoCallScalar(GoFuncAddr(runtime.NumCPU), []JITValueDesc{}, 1)
				ctx.BindReg(d1.Reg, &d1)
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
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["memstats"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
}
