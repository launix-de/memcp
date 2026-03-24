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

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)
import "github.com/jtolds/gls"

// cachedMemStats provides a cached version of runtime.ReadMemStats.
// The cache is refreshed when older than 1 minute. Between refreshes,
// Alloc and HeapAlloc are adjusted by deltas reported via AdjustMemStats.
var (
	cachedStats     runtime.MemStats
	cachedStatsTime time.Time
	cachedStatsMu   sync.Mutex
	memStatsDelta   int64 // accumulated delta since last refresh
)

// CachedMemStats returns a cached runtime.MemStats, refreshing only when
// the cache is older than 1 minute. Between refreshes, Alloc and HeapAlloc
// are adjusted by tracked deltas from the CacheManager.
func CachedMemStats() runtime.MemStats {
	cachedStatsMu.Lock()
	defer cachedStatsMu.Unlock()
	if time.Since(cachedStatsTime) > time.Minute {
		runtime.ReadMemStats(&cachedStats)
		cachedStatsTime = time.Now()
		memStatsDelta = 0
	}
	result := cachedStats
	if memStatsDelta > 0 {
		result.Alloc += uint64(memStatsDelta)
		result.HeapAlloc += uint64(memStatsDelta)
	} else if memStatsDelta < 0 && uint64(-memStatsDelta) < result.Alloc {
		result.Alloc -= uint64(-memStatsDelta)
		result.HeapAlloc -= uint64(-memStatsDelta)
	}
	return result
}

// AdjustMemStats adjusts the cached memory stats by a delta (in bytes).
// Call this when tracked memory changes (e.g. CacheManager add/remove).
func AdjustMemStats(delta int64) {
	cachedStatsMu.Lock()
	memStatsDelta += delta
	cachedStatsMu.Unlock()
}

/* promise: single-value cell */

// NOTE: current implementation is intentionally not thread-safe.
// It is sufficient for sequential query-plan execution. Fresh promises use a
// dedicated [2]Scmer backing; (newpromise list) reuses an existing >=2-element
// slice with zero extra allocation.
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

var promiseLockSentinel = makeAux(tagBool, 0)
var promiseFailedAux = makeAux(tagBool, 2)

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

func promiseUnlock(cells *[2]Scmer, newStateAux uint64) {
	atomic.StoreUint64(&cells[1].aux, newStateAux)
}

// ApplyPromise dispatches a tagPromise call. Thread-safe via CAS spin-lock.
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
			prevAux := promiseLock(cells)
			if prevAux == NewNil().aux {
				promiseUnlock(cells, prevAux)
				return NewNil()
			}
			val := cells[0]
			promiseUnlock(cells, prevAux)
			return val
		case "state":
			prevAux := promiseLock(cells)
			promiseUnlock(cells, prevAux)
			if prevAux == NewNil().aux {
				return NewNil()
			}
			if prevAux == promiseFailedAux {
				return NewBool(false)
			}
			return NewBool(true)
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
			promiseUnlock(cells, NewBool(true).aux)
			return args[1]
		}
		if key == "once" {
			prevAux := promiseLock(cells)
			if prevAux != NewNil().aux {
				promiseUnlock(cells, prevAux)
				panic("promise already fulfilled/failed")
			}
			cells[0] = args[1]
			promiseUnlock(cells, NewBool(true).aux)
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
			prevAux := promiseLock(cells)
			if prevAux != NewNil().aux {
				promiseUnlock(cells, prevAux)
				panic(args[2].String())
			}
			cells[0] = args[1]
			promiseUnlock(cells, NewBool(true).aux)
			return args[1]
		}
		panic("promise: unknown operation: " + key)
	default:
		panic("promise: too many arguments")
	}
}

/* threadsafe session storage */

type session struct {
	Mu  sync.RWMutex
	Map map[string]Scmer
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
			sess.Map[a[0].String()] = a[1]
			return a[1]
		case 1:
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			if v, ok := sess.Map[a[0].String()]; ok {
				return v
			}
			return NewNil()
		case 0:
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			keys := make([]Scmer, 0, len(sess.Map))
			for k := range sess.Map {
				keys = append(keys, NewString(k))
			}
			return NewSlice(keys)
		default:
			panic("wrong number of parameters provided to session: 0, 1 or 2 required")
		}
	})
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
		case "check":
			ctxVal, ok := mgr.GetValue("context")
			if !ok {
				panic("no context set")
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

// NewContextWithSession is like NewContext but uses a pre-existing Scheme session
// instead of creating a fresh one. Used by persistent HTTP sessions so that
// @variables set in one request are visible in subsequent requests.
func NewContextWithSession(ctx context.Context, session Scmer, fn func()) {
	if mgr == nil {
		mgr = gls.NewContextManager()
	}
	mgr.SetValues(gls.Values{
		"session": session,
		"context": ctx,
	}, fn)
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
		Desc: "Creates a single-value promise cell (not thread-safe). Returns a tagPromise Scmer. (newpromise) allocates a [2]Scmer backing; (newpromise list) reuses an existing ≥2-element slice as backing with zero extra allocation. API: (p \"value\") reads current value (nil if pending), (p \"value\" v) resolves, (p \"once\" v) resolves once (panics if already fulfilled/failed), (p \"once\" v msg) resolves once with custom panic message, (p \"state\") returns state (nil/true/false), (p \"fail\") sets failed and clears the stored value, (p \"fail\" err) sets failed and stores err as payload.",
		Fn: NewPromise,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "list", ParamDesc: "optional: ≥2-element slice to use as backing", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "func", HasSideEffects: true},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "newsession",
		Desc: "Creates a new session which is a threadsafe key-value store represented as a function that can be either called as a getter (session key) or setter (session key value) or list all keys with (session)",
		Fn: NewSession,
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "func", HasSideEffects: true},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "with_session",
		Desc: "Executes a function with the given session installed in the execution context, so storage operations can access the session's transaction state.",
		Fn: func(a ...Scmer) Scmer {
			return WithSession(a[0], a[1])
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "func", ParamName: "session", ParamDesc: "the session to install"},
				{Kind: "func", ParamName: "fn", ParamDesc: "the function to execute"},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "context",
		Desc: "Context helper function. Each context also contains a session. (context func args) creates a new context and runs func in that context, (context \"session\") reads the session variable, (context \"check\") will check the liveliness of the context and otherwise throw an error",
		Fn: Context,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "args...", ParamDesc: "depends on the usage", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sleep",
		Desc: "sleeps the amount of seconds",
		Fn: func(a ...Scmer) Scmer {
			ctx := GetContext()
			select {
			case <-ctx.Done():
				panic(ctx.Err())
			case <-time.After(time.Duration(ToFloat(a[0]) * float64(time.Second))):
				return NewBool(true)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "duration", ParamDesc: "number of seconds to sleep"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "once",
		Desc: "Creates a function wrapper that you can call multiple times but only gets executed once. The result value is cached and returned on a second call. You can add parameters to that resulting function that will be passed to the first run of the wrapped function.",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "func", ParamName: "f", ParamDesc: "function that produces the result value"},
			},
			Return: &TypeDescriptor{Kind: "func", HasSideEffects: true},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mutex",
		Desc: "Creates a mutex. The return value is a function that takes one parameter which is a parameterless function. The mutex is guaranteed that all calls to that mutex get serialized.",
		Fn: func(a ...Scmer) Scmer {
			var mutex sync.Mutex
			return NewFunc(func(a ...Scmer) Scmer {
				mutex.Lock()
				defer func() {
					mutex.Unlock() // free after return or panic, so we don't get into deadlocks
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
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "func", HasSideEffects: true},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "numcpu",
		Desc: "Returns the number of logical CPUs available for parallel execution",
		Fn: func(a ...Scmer) Scmer {
			return NewInt(int64(runtime.NumCPU()))
		},
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "memstats",
		Desc: "Returns memory statistics as a dict with keys: alloc, total_alloc, sys, heap_alloc, heap_sys (all in bytes)",
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
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "dict"},
			Const: true,
		},
	})
}
