/*
Copyright (C) 2024  Carl-Philip Hänsch

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
import "github.com/jtolds/gls"

/* threadsafe session storage */

type session struct {
	Mu  sync.RWMutex
	Map map[string]Scmer
}

// build this function into your SCM environment to offer http server capabilities
func NewSession(a ...Scmer) Scmer {
	// params: port, authcallback, schemacallback, querycallback
	sess := new(session)
	sess.Map = make(map[string]Scmer)
	return func(a ...Scmer) (result Scmer) {
		if len(a) == 2 {
			// set
			sess.Mu.Lock()
			defer sess.Mu.Unlock()
			sess.Map[String(a[0])] = a[1]
			return a[1] // reflect the value as of mysql semantics
		} else if len(a) == 1 {
			// get
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			result, _ = sess.Map[String(a[0])]
			return
		} else if len(a) == 0 {
			// list keys
			sess.Mu.RLock()
			defer sess.Mu.RUnlock()
			l := make([]Scmer, 0, len(sess.Map))
			for k, _ := range sess.Map {
				l = append(l, k)
			}
			result = l
			return
		} else {
			panic("wrong number of parameters provided to session: 0, 1 or 2 required")
		}
	}
}

var mgr *gls.ContextManager

func Context(a ...Scmer) (result Scmer) {
	if mgr == nil {
		// prone to race conditions, to the first call should be called in the initialization
		mgr = gls.NewContextManager()
	}
	if a[0] == "session" {
		result, ok := mgr.GetValue("session")
		if !ok {
			panic("no session set")
		}
		return result
	} else if a[0] == "check" {
		result, ok := mgr.GetValue("context")
		if !ok {
			panic("no context set")
		}
		e := result.(context.Context).Err()
		if e != nil {
			panic(e)
		}
		return true
	} else if a[0] != nil {
		// constructor
		NewContext(context.TODO(), func() {
			result = Apply(a[0], a[1:]...) // call the inner function in the context
		})
		return
	} else {
		panic("unimplemented")
	}
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

func init_sync() {
	DeclareTitle("Sync")
	Declare(&Globalenv, &Declaration{
		"newsession", "Creates a new session which is a threadsafe key-value store represented as a function that can be either called as a getter (session key) or setter (session key value) or list all keys with (session)",
		0, 0,
		[]DeclarationParameter{}, "func",
		NewSession, false,
	})
	Declare(&Globalenv, &Declaration{
		"context", "Context helper function. Each context also contains a session. (context func args) creates a new context and runs func in that context, (context \"session\") reads the session variable, (context \"check\") will check the liveliness of the context and otherwise throw an error",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"args...", "any", "depends on the usage"},
		}, "any",
		Context, false,
	})
	Declare(&Globalenv, &Declaration{
		"sleep", "sleeps the amount of seconds",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"duration", "number", "number of seconds to sleep"},
		}, "bool",
		func(a ...Scmer) Scmer {
			ctx := GetContext()
			select {
			case <-ctx.Done():
				panic(ctx.Err)
			case <-time.After(time.Duration(ToFloat(a[0]) * float64(time.Second))):
				return true
			}
		}, false,
	})
	Declare(&Globalenv, &Declaration{
		"once", "Creates a function wrapper that you can call multiple times but only gets executed once. The result value is cached and returned on a second call. You can add parameters to that resulting function that will be passed to the first run of the wrapped function.",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"f", "func", "function that produces the result value"},
		}, "func",
		func(a ...Scmer) Scmer {
			var params []Scmer
			once := sync.OnceValue[Scmer](func() Scmer {
				return Apply(a[0], params...)
			})
			return func(a ...Scmer) Scmer {
				params = a
				return once()
			}
		}, false,
	})
	Declare(&Globalenv, &Declaration{
		"mutex", "Creates a mutex. The return value is a function that takes one parameter which is a parameterless function. The mutex is guaranteed that all calls to that mutex get serialized.",
		1, 1,
		[]DeclarationParameter{}, "func",
		func(a ...Scmer) Scmer {
			var mutex sync.Mutex
			return func(a ...Scmer) Scmer {
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
			}
		}, false,
	})
}
