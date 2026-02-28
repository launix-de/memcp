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
import "context"
import "runtime"
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

// TxSyncer is implemented by TxContext to allow deferred sync from
// the MySQL/HTTP frontends without importing the storage package.
type TxSyncer interface {
	SyncTouchedShards()
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
		"newsession", "Creates a new session which is a threadsafe key-value store represented as a function that can be either called as a getter (session key) or setter (session key value) or list all keys with (session)",
		0, 0,
		[]DeclarationParameter{}, "func",
		NewSession, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"with_session", "Executes a function with the given session installed in the execution context, so storage operations can access the session's transaction state.",
		2, 2,
		[]DeclarationParameter{
			{"session", "func", "the session to install", nil},
			{"fn", "func", "the function to execute", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			return WithSession(a[0], a[1])
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"context", "Context helper function. Each context also contains a session. (context func args) creates a new context and runs func in that context, (context \"session\") reads the session variable, (context \"check\") will check the liveliness of the context and otherwise throw an error",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"args...", "any", "depends on the usage", nil},
		}, "any",
		Context, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"sleep", "sleeps the amount of seconds",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"duration", "number", "number of seconds to sleep", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			ctx := GetContext()
			select {
			case <-ctx.Done():
				panic(ctx.Err())
			case <-time.After(time.Duration(ToFloat(a[0]) * float64(time.Second))):
				return NewBool(true)
			}
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"once", "Creates a function wrapper that you can call multiple times but only gets executed once. The result value is cached and returned on a second call. You can add parameters to that resulting function that will be passed to the first run of the wrapped function.",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"f", "func", "function that produces the result value", nil},
		}, "func",
		func(a ...Scmer) Scmer {
			var params []Scmer
			once := sync.OnceValue[Scmer](func() Scmer {
				return Apply(a[0], params...)
			})
			return NewFunc(func(a ...Scmer) Scmer {
				params = a
				return once()
			})
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"mutex", "Creates a mutex. The return value is a function that takes one parameter which is a parameterless function. The mutex is guaranteed that all calls to that mutex get serialized.",
		0, 0,
		[]DeclarationParameter{}, "func",
		func(a ...Scmer) Scmer {
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
		}, false, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"numcpu", "Returns the number of logical CPUs available for parallel execution",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			return NewInt(int64(runtime.NumCPU()))
		}, true, false, nil,
		nil,
	})
	Declare(&Globalenv, &Declaration{
		"memstats", "Returns memory statistics as a dict with keys: alloc, total_alloc, sys, heap_alloc, heap_sys (all in bytes)",
		0, 0,
		[]DeclarationParameter{}, "dict",
		func(a ...Scmer) Scmer {
			m := CachedMemStats()
			fd := NewFastDictValue(5)
			fd.Set(NewString("alloc"), NewInt(int64(m.Alloc)), nil)
			fd.Set(NewString("total_alloc"), NewInt(int64(m.TotalAlloc)), nil)
			fd.Set(NewString("sys"), NewInt(int64(m.Sys)), nil)
			fd.Set(NewString("heap_alloc"), NewInt(int64(m.HeapAlloc)), nil)
			fd.Set(NewString("heap_sys"), NewInt(int64(m.HeapSys)), nil)
			return NewFastDict(fd)
		}, true, false, nil,
		nil,
	})
}
