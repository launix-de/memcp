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
	"sync"
	"sync/atomic"
	"time"
)

// SessionState tracks one active connection for SHOW [FULL] PROCESSLIST.
// The owning goroutine is the only writer of mutable fields — no global lock
// needed on the hot path. Readers (SHOW PROCESSLIST, KILL) use atomics.
type SessionState struct {
	ID   uint64
	User string // immutable after registration
	Host string // immutable after registration

	DB      atomic.Pointer[string] // current schema (changes on USE)
	Command atomic.Pointer[string] // "Query", "Sleep", "Connect"
	Info    atomic.Pointer[string] // current SQL (empty when idle)
	State   atomic.Pointer[string] // "Waiting for table lock", "" etc.

	startedAt atomic.Int64 // unix nanos of last command start

	killed atomic.Bool // set by Kill(); checked at scan entry points

	cancel   context.CancelFunc // called by KILL QUERY; nil if not cancellable
	cancelMu sync.Mutex         // protects cancel assignment

	heldLocks   []func()   // unlock callbacks for LOCK TABLES
	heldLocksMu sync.Mutex // protects heldLocks slice

	scmSession     Scmer     // persistent Scheme session for HTTP connections
	scmSessionOnce sync.Once // ensures scmSession is initialized exactly once
}

// GetOrCreateScmSession returns the persistent Scheme session for this SessionState,
// creating it on first call. Used by HTTP sessions to persist @variables across requests.
func (s *SessionState) GetOrCreateScmSession() Scmer {
	s.scmSessionOnce.Do(func() {
		s.scmSession = NewSession()
	})
	return s.scmSession
}

// ElapsedSeconds returns seconds since the last command started.
func (s *SessionState) ElapsedSeconds() int64 {
	ns := s.startedAt.Load()
	if ns == 0 {
		return 0
	}
	return int64(time.Since(time.Unix(0, ns)).Seconds())
}

// SetCommand updates Command, Info, and resets the elapsed timer.
func (s *SessionState) SetCommand(cmd, info string) {
	s.Command.Store(&cmd)
	s.Info.Store(&info)
	s.startedAt.Store(time.Now().UnixNano())
}

// SetState updates the State field (e.g. "Waiting for table lock").
func (s *SessionState) SetState(state string) {
	s.State.Store(&state)
}

// SetDB updates the current database name.
func (s *SessionState) SetDB(db string) {
	s.DB.Store(&db)
}

// SetCancel stores the cancel function for KILL QUERY support.
func (s *SessionState) SetCancel(fn context.CancelFunc) {
	s.cancelMu.Lock()
	s.cancel = fn
	s.cancelMu.Unlock()
}

// ClearCancel removes the cancel function after query completion.
func (s *SessionState) ClearCancel() {
	s.cancelMu.Lock()
	s.cancel = nil
	s.cancelMu.Unlock()
}

// IsKilled returns true if this session has been killed.
func (s *SessionState) IsKilled() bool {
	return s.killed.Load()
}

// ResetKilled clears the killed flag, e.g. at the start of a new request on a reused session.
func (s *SessionState) ResetKilled() {
	s.killed.Store(false)
}

// Kill marks the session as killed and fires the cancel function if set.
// Returns true if a running query was cancelled.
func (s *SessionState) Kill() bool {
	s.killed.Store(true)
	s.cancelMu.Lock()
	fn := s.cancel
	s.cancelMu.Unlock()
	if fn != nil {
		fn()
		return true
	}
	return false
}

// AddLock registers an unlock callback for a LOCK TABLES lock.
func (s *SessionState) AddLock(unlock func()) {
	s.heldLocksMu.Lock()
	s.heldLocks = append(s.heldLocks, unlock)
	s.heldLocksMu.Unlock()
}

// ReleaseAllLocks releases all table locks held by this session.
func (s *SessionState) ReleaseAllLocks() {
	s.heldLocksMu.Lock()
	fns := s.heldLocks
	s.heldLocks = nil
	s.heldLocksMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// strPtr is a helper to load an atomic string pointer safely.
func strPtr(p *atomic.Pointer[string]) string {
	if v := p.Load(); v != nil {
		return *v
	}
	return ""
}

// --- Global registry ---

var (
	processList   sync.Map      // map[uint64]*SessionState
	nextSessionID atomic.Uint64 // monotonic counter for session IDs
	httpStates    sync.Map      // map[string]*SessionState for persistent HTTP sessions (X-Session-Id)
)

// HTTPSessionAddHook is called when a new persistent HTTP session is created.
// The storage package wires in GlobalCache registration via SetHTTPSessionAddHook.
var httpSessionAddHook func(key string, ss *SessionState)

// SetHTTPSessionAddHook wires in a callback for when a new persistent HTTP session is created.
// Intended to be called once from storage after GlobalCache.Init().
func SetHTTPSessionAddHook(fn func(key string, ss *SessionState)) {
	httpSessionAddHook = fn
}

// EvictHTTPSession removes a persistent HTTP session from the processlist.
// Called by the cache manager's cleanup callback.
func EvictHTTPSession(key string) bool {
	v, ok := httpStates.LoadAndDelete(key)
	if !ok {
		return false
	}
	UnregisterSession(v.(*SessionState).ID)
	return true
}

// LastUsedNano returns the unix nanosecond timestamp of the last command start.
func (s *SessionState) LastUsedNano() int64 {
	return s.startedAt.Load()
}

// GetCurrentSessionState returns the *SessionState for the current goroutine's
// GLS context, or nil if none is set.
func GetCurrentSessionState() *SessionState {
	if mgr == nil {
		return nil
	}
	v, ok := mgr.GetValue("sessionStatePtr")
	if !ok {
		return nil
	}
	ss, _ := v.(*SessionState)
	return ss
}

func init_processlist() {
	nextSessionID.Store(1)
		Declare(&Globalenv, &Declaration{
		Name: "show_processlist",
		Desc: "returns a list of active sessions for SHOW [FULL] PROCESSLIST; pass true for full info",
		Fn: func(a ...Scmer) Scmer {
				full := len(a) > 0 && a[0].Bool()
				sessions := Snapshot()
				result := make([]Scmer, len(sessions))
				for i, s := range sessions {
					info := strPtr(&s.Info)
					if !full && len(info) > 100 {
						info = info[:100]
					}
					result[i] = NewSlice([]Scmer{
						NewString("Id"), NewInt(int64(s.ID)),
						NewString("User"), NewString(s.User),
						NewString("Host"), NewString(s.Host),
						NewString("db"), NewString(strPtr(&s.DB)),
						NewString("Command"), NewString(strPtr(&s.Command)),
						NewString("Time"), NewInt(s.ElapsedSeconds()),
						NewString("State"), NewString(strPtr(&s.State)),
						NewString("Info"), NewString(info),
					})
				}
				return NewSlice(result)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "bool", ParamName: "full", ParamDesc: "if true, include full Info text", Optional: true}},
			Return: &TypeDescriptor{Kind: "list"},
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "connection_id",
		Desc: "returns the process-list ID of the current session (MySQL CONNECTION_ID() equivalent)",
		Fn: func(a ...Scmer) Scmer {
				if ss := GetCurrentSessionState(); ss != nil {
					return NewInt(int64(ss.ID))
				}
				return NewInt(0)
			},
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "int"},
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "kill_query",
		Desc: "cancel the running query in session id; returns true if a query was killed",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(KillSession(uint64(a[0].Int())))
			},
		Type: &TypeDescriptor{
			HasSideEffects: true,
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "int", ParamName: "id", ParamDesc: "session ID from SHOW PROCESSLIST"}},
			Return: &TypeDescriptor{Kind: "bool"},
		},
	})
}

// RegisterSession adds a new session to the process list and returns its state.
func RegisterSession(user, host, db string) *SessionState {
	s := &SessionState{
		ID:   nextSessionID.Add(1),
		User: user,
		Host: host,
	}
	s.SetDB(db)
	cmd := "Connect"
	s.Command.Store(&cmd)
	empty := ""
	s.Info.Store(&empty)
	s.State.Store(&empty)
	s.startedAt.Store(time.Now().UnixNano())
	processList.Store(s.ID, s)
	return s
}

// UnregisterSession removes a session from the process list.
func UnregisterSession(id uint64) {
	processList.Delete(id)
}

// Snapshot returns a point-in-time copy of all active sessions.
// Reading individual atomic fields outside the lock is safe: the session
// struct is never freed while the snapshot holds a pointer to it.
func Snapshot() []*SessionState {
	result := make([]*SessionState, 0, 16)
	processList.Range(func(_, v any) bool {
		result = append(result, v.(*SessionState))
		return true
	})
	return result
}

// KillSession cancels the query running in session id.
// Returns true if the session was found and had an active query.
func KillSession(id uint64) bool {
	v, ok := processList.Load(id)
	if !ok {
		return false
	}
	return v.(*SessionState).Kill()
}
