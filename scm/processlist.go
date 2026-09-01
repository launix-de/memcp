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
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)
import "unsafe"

// SessionState tracks one active connection for SHOW [FULL] PROCESSLIST.
// The owning goroutine is the only writer of mutable fields — no global lock
// needed on the hot path. Readers (SHOW PROCESSLIST, KILL) use atomics.
type SessionState struct {
	ID   uint64
	User string // immutable after registration
	Host string // immutable after registration

	DB        atomic.Pointer[string] // current schema (changes on USE)
	Command   atomic.Pointer[string] // "Query", "Sleep", "Connect"
	Info      atomic.Pointer[string] // current SQL (empty when idle)
	State     atomic.Pointer[string] // "Waiting for table lock", "" etc.
	lockWaits atomic.Int64           // number of active table-lock waits for processlist display

	startedAt atomic.Int64 // unix nanos of last command start
	lastUsed  atomic.Int64 // unix nanos of last observed access; used for cache eviction

	nextQuerySeq atomic.Uint64 // monotonically increasing request/query generation
	activeQuery  atomic.Uint64 // latest running generation for processlist display
	activeCount  atomic.Int64  // number of concurrently running requests/queries

	cancelFns map[uint64]context.CancelFunc
	queryCtxs map[uint64]context.Context
	active    map[uint64]bool
	killed    map[uint64]bool
	cancelMu  sync.Mutex // protects active query bookkeeping

	heldLocks   []func()   // unlock callbacks for LOCK TABLES
	heldLocksMu sync.Mutex // protects heldLocks slice

	scmSession     Scmer     // persistent Scheme session for HTTP connections
	scmSessionOnce sync.Once // ensures scmSession is initialized exactly once

}

// QueryExecutionContext carries request-local state across the Scheme SQL
// boundary without rediscovering it through goroutine-local stack inspection.
// The transaction itself remains owned by storage and is attached by
// with_autocommit after the executing Scheme session has been selected.
type QueryExecutionContext struct {
	SessionState *SessionState
	QuerySeq     uint64
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

// Touch marks this session as recently used for cache eviction purposes.
func (s *SessionState) Touch() {
	s.lastUsed.Store(time.Now().UnixNano())
}

// SetCommand updates Command, Info, and resets the elapsed timer.
func (s *SessionState) SetCommand(cmd, info string) {
	s.Command.Store(&cmd)
	s.Info.Store(&info)
	now := time.Now().UnixNano()
	s.startedAt.Store(now)
	s.lastUsed.Store(now)
}

// BeginQuery marks a new request/query generation as active on this session.
// This prevents late disconnects from earlier HTTP requests from killing a
// subsequent request reusing the same SessionState.
func (s *SessionState) BeginQuery(cmd, info string) uint64 {
	seq := s.nextQuerySeq.Add(1)
	s.activeQuery.Store(seq)
	s.activeCount.Add(1)
	s.cancelMu.Lock()
	if s.active == nil {
		s.active = make(map[uint64]bool)
	}
	s.active[seq] = true
	if s.killed != nil {
		delete(s.killed, seq)
	}
	s.cancelMu.Unlock()
	s.SetState("")
	s.SetCommand(cmd, info)
	return seq
}

// SetCancel stores the cancel function for one specific active query generation.
func (s *SessionState) SetCancel(seq uint64, fn context.CancelFunc) {
	s.cancelMu.Lock()
	if s.cancelFns == nil {
		s.cancelFns = make(map[uint64]context.CancelFunc)
	}
	s.cancelFns[seq] = fn
	s.cancelMu.Unlock()
}

// SetQueryContext records the query-generation-specific cancellation signal.
// Persistent HTTP sessions may execute overlapping generations, so table-lock
// waiters must never infer this context from the session's latest query.
func (s *SessionState) SetQueryContext(seq uint64, ctx context.Context) {
	s.cancelMu.Lock()
	if s.queryCtxs == nil {
		s.queryCtxs = make(map[uint64]context.Context)
	}
	s.queryCtxs[seq] = ctx
	s.cancelMu.Unlock()
}

// SetQueryInfo updates the processlist text for an active query generation.
// HTTP requests use this after lazily reading a SQL request body. A stale
// request must not overwrite the text of a newer request sharing the session.
func (s *SessionState) SetQueryInfo(seq uint64, info string) bool {
	if seq == 0 || s.activeQuery.Load() != seq {
		return false
	}
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	if !s.active[seq] || s.activeQuery.Load() != seq {
		return false
	}
	s.Info.Store(&info)
	return true
}

// EndQuery clears the active generation if it still matches seq and restores
// the idle processlist state. Older requests finishing late must not overwrite
// a newer active request on the same persistent HTTP session.
func (s *SessionState) EndQuery(seq uint64, idleCmd, idleInfo string) {
	s.ClearCancel(seq)
	if s.activeCount.Add(-1) == 0 {
		s.activeQuery.Store(0)
		s.SetState("")
		s.SetCommand(idleCmd, idleInfo)
	}
}

// SetState updates the State field (e.g. "Waiting for table lock").
func (s *SessionState) SetState(state string) {
	s.State.Store(&state)
}

func (s *SessionState) BeginLockWait() {
	s.lockWaits.Add(1)
}

func (s *SessionState) EndLockWait() {
	for {
		cur := s.lockWaits.Load()
		if cur <= 0 {
			return
		}
		if s.lockWaits.CompareAndSwap(cur, cur-1) {
			return
		}
	}
}

func (s *SessionState) processListState() string {
	if s.lockWaits.Load() > 0 {
		return "Waiting for table lock"
	}
	return strPtr(&s.State)
}

// SetDB updates the current database name.
func (s *SessionState) SetDB(db string) {
	s.DB.Store(&db)
}

// QueryContext returns the cancellation context owned by one query generation.
func (s *SessionState) QueryContext(seq uint64) context.Context {
	if seq == 0 {
		return nil
	}
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	return s.queryCtxs[seq]
}

// ClearCancel removes the cancel function if it still belongs to seq.
func (s *SessionState) ClearCancel(seq uint64) {
	s.cancelMu.Lock()
	if s.cancelFns != nil {
		delete(s.cancelFns, seq)
	}
	if s.queryCtxs != nil {
		delete(s.queryCtxs, seq)
	}
	if s.active != nil {
		delete(s.active, seq)
	}
	if s.killed != nil {
		delete(s.killed, seq)
	}
	s.cancelMu.Unlock()
}

// IsKilled returns true if this session has been killed.
//
// Storage execution contract: callers may check cancellation while scheduling
// shard jobs, but never after entering a shard. Shard execution is atomic and
// must not contain cancellation checks in index, batch, or row loops.
func (s *SessionState) IsKilled() bool {
	return s.IsKilledSeq(CurrentQuerySeq())
}

// CurrentQuerySeq returns the query generation installed in the current
// execution context. Storage workers should capture this once in the parent
// goroutine and use IsKilledSeq instead of reading GLS concurrently.
func CurrentQuerySeq() uint64 {
	if mgr == nil {
		return 0
	}
	v, ok := mgr.GetValue("querySeq")
	if !ok {
		return 0
	}
	seq, ok := v.(uint64)
	if !ok || seq == 0 {
		return 0
	}
	return seq
}

// IsKilledSeq returns true if the given query generation has been killed.
//
// Storage execution contract: callers may check cancellation while scheduling
// shard jobs, but never after entering a shard. Shard execution is atomic and
// must not contain cancellation checks in index, batch, or row loops.
func (s *SessionState) IsKilledSeq(seq uint64) bool {
	if seq == 0 {
		return false
	}
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	return s.killed != nil && s.killed[seq]
}

func formatKillLog(s *SessionState, action string) string {
	info := strPtr(&s.Info)
	if len(info) > 160 {
		info = info[:160] + "..."
	}
	if info == "" {
		return fmt.Sprintf("kill %s id=%d user=%s host=%s db=%s", action, s.ID, s.User, s.Host, strPtr(&s.DB))
	}
	return fmt.Sprintf("kill %s id=%d user=%s host=%s db=%s sql=%s", action, s.ID, s.User, s.Host, strPtr(&s.DB), info)
}

func logKill(s *SessionState, action string) {
	TracePrintFunc(formatKillLog(s, action))
}

// Kill marks the session as killed and fires the cancel function if set.
// Returns true if at least one running query was cancelled.
func (s *SessionState) Kill() bool {
	s.cancelMu.Lock()
	if len(s.active) == 0 {
		s.cancelMu.Unlock()
		return false
	}
	if s.killed == nil {
		s.killed = make(map[uint64]bool, len(s.active))
	}
	fns := make([]context.CancelFunc, 0, len(s.cancelFns))
	for seq := range s.active {
		s.killed[seq] = true
		if fn := s.cancelFns[seq]; fn != nil {
			fns = append(fns, fn)
		}
	}
	s.cancelMu.Unlock()
	for _, fn := range fns {
		fn()
	}
	logKill(s, "session")
	return true
}

// KillQuery marks the given active query generation as killed.
// Returns false if the session has already advanced to a different request.
func (s *SessionState) KillQuery(seq uint64) bool {
	if seq == 0 {
		return false
	}
	s.cancelMu.Lock()
	if s.active == nil || !s.active[seq] {
		s.cancelMu.Unlock()
		return false
	}
	fn := s.cancelFns[seq]
	if s.killed == nil {
		s.killed = make(map[uint64]bool)
	}
	s.killed[seq] = true
	s.cancelMu.Unlock()
	if fn != nil {
		fn()
	}
	logKill(s, fmt.Sprintf("query seq=%d", seq))
	return true
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
	ss := v.(*SessionState)
	ss.Kill()
	ss.ReleaseAllLocks()
	UnregisterSession(ss.ID)
	return true
}

// LastUsedNano returns the unix nanosecond timestamp of the last command start.
func (s *SessionState) LastUsedNano() int64 {
	if ts := s.lastUsed.Load(); ts != 0 {
		return ts
	}
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

		Fn: func(a ...Scmer) Scmer {
			full := len(a) > 0 && a[0].Bool()
			sessions := Snapshot()
			result := make([]Scmer, len(sessions))
			for i, s := range sessions {
				info := strPtr(&s.Info)
				if !full && len(info) > 100 {
					info = info[:100]
				}
				state := s.processListState()
				result[i] = NewSlice([]Scmer{
					NewString("Id"), NewInt(int64(s.ID)),
					NewString("User"), NewString(s.User),
					NewString("Host"), NewString(s.Host),
					NewString("db"), NewString(strPtr(&s.DB)),
					NewString("Command"), NewString(strPtr(&s.Command)),
					NewString("Time"), NewInt(s.ElapsedSeconds()),
					NewString("State"), NewString(state),
					NewString("Info"), NewString(info),
				})
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list of active sessions for SHOW [FULL] PROCESSLIST; pass true for full info",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "bool", Label: "full", Description: "if true, include full Info text", Optional: true}},
			Return: &TypeDescriptor{Kind: "list"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["show_processlist"].Fn, args, result)
				}
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d9 JITValueDesc
				_ = d9
				var d12 JITValueDesc
				_ = d12
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d42 JITValueDesc
				_ = d42
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
				var d76 JITValueDesc
				_ = d76
				var d78 JITValueDesc
				_ = d78
				var d81 JITValueDesc
				_ = d81
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d132 JITValueDesc
				_ = d132
				var stackArray133 int32
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d155 JITValueDesc
				_ = d155
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d159 JITValueDesc
				_ = d159
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d165 JITValueDesc
				_ = d165
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				var bbs [9]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				bbs[3].PhiCount = uint16(1)
				bbs[7].PhiBase = int32(phiBase0) + int32(32)
				bbs[7].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d4)
					var d5 JITValueDesc
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() > 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d5)
					}
					ctx.FreeDesc(&d4)
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							if ps.General {
							}
							ps7 := PhiState{General: ps.General}
							ps7.OverlayValues = make([]JITValueDesc, 7)
							ps7.OverlayValues[1] = d1
							ps7.OverlayValues[2] = d2
							ps7.OverlayValues[3] = d3
							ps7.OverlayValues[4] = d4
							ps7.OverlayValues[5] = d5
							ps7.OverlayValues[6] = d6
							return bbs[1].RenderPS(ps7)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps8 := PhiState{General: ps.General}
						ps8.OverlayValues = make([]JITValueDesc, 7)
						ps8.OverlayValues[1] = d1
						ps8.OverlayValues[2] = d2
						ps8.OverlayValues[3] = d3
						ps8.OverlayValues[4] = d4
						ps8.OverlayValues[5] = d5
						ps8.OverlayValues[6] = d6
						ps8.PhiValues = make([]JITValueDesc, 1)
						d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps8.PhiValues[0] = d9
						return bbs[2].RenderPS(ps8)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl11)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 10)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[9] = d9
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 10)
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[9] = d9
					ps11.PhiValues = make([]JITValueDesc, 1)
					d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps11.PhiValues[0] = d12
					snap13 := d1
					snap14 := d2
					snap15 := d3
					snap16 := d4
					snap17 := d5
					snap18 := d6
					snap19 := d9
					snap20 := d12
					alloc21 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps11)
					}
					ctx.RestoreAllocState(alloc21)
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d5 = snap17
					d6 = snap18
					d9 = snap19
					d12 = snap20
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps10)
					}
					return result
					ctx.FreeDesc(&d5)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					ctx.ReclaimUntrackedRegs()
					d22 = args[0]
					d22.ID = 0
					d24 = d22
					d24.ID = 0
					d23 = ctx.EmitBoolDesc(&d24, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d22)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d25 = d23
						if d25.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d25)
						ctx.EmitStoreToStack(d25, int32(bbs[2].PhiBase)+int32(0))
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps26 := PhiState{General: ps.General}
					ps26.OverlayValues = make([]JITValueDesc, 26)
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[6] = d6
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[12] = d12
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[23] = d23
					ps26.OverlayValues[24] = d24
					ps26.OverlayValues[25] = d25
					ps26.PhiValues = make([]JITValueDesc, 1)
					d27 = d23
					ps26.PhiValues[0] = d27
					if ps26.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps26)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d28 := ps.PhiValues[0]
							ctx.EnsureDesc(&d28)
							ctx.EmitStoreToStack(d28, int32(bbs[2].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d29 = ctx.EmitGoCallScalar(GoFuncAddr(Snapshot), []JITValueDesc{}, 3)
					ctx.BindReg(d29.Reg, &d29)
					ctx.BindReg(d29.Reg2, &d29)
					ctx.BindReg(d29.Reg3, &d29)
					ctx.StabilizeDescForControlFlow(&d29)
					var d30 JITValueDesc
					if d29.SliceSizeKnown {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.KnownSliceLen))}
					} else if d29.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.StackOff))}
					} else if d29.Loc == LocStackTriple {
						d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d29.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d29)
						if d29.Loc == LocRegPair || d29.Loc == LocRegTriple {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2, ID: 0}
						} else if d29.Loc == LocReg {
							d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d30)
					callResults31 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d30, d30}, []uint8{3}, []uint8{1})
					d32 = callResults31[0]
					d32.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d32)
					ctx.FreeDesc(&d30)
					var d33 JITValueDesc
					if d29.SliceSizeKnown {
						d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.KnownSliceLen))}
					} else if d29.Loc == LocImm {
						d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.StackOff))}
					} else if d29.Loc == LocStackTriple {
						d33 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d29.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d29)
						if d29.Loc == LocRegPair || d29.Loc == LocRegTriple {
							d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2, ID: 0}
						} else if d29.Loc == LocReg {
							d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d33)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					}
					ps34 := PhiState{General: ps.General}
					ps34.OverlayValues = make([]JITValueDesc, 34)
					ps34.OverlayValues[1] = d1
					ps34.OverlayValues[2] = d2
					ps34.OverlayValues[3] = d3
					ps34.OverlayValues[4] = d4
					ps34.OverlayValues[5] = d5
					ps34.OverlayValues[6] = d6
					ps34.OverlayValues[9] = d9
					ps34.OverlayValues[12] = d12
					ps34.OverlayValues[22] = d22
					ps34.OverlayValues[23] = d23
					ps34.OverlayValues[24] = d24
					ps34.OverlayValues[25] = d25
					ps34.OverlayValues[27] = d27
					ps34.OverlayValues[28] = d28
					ps34.OverlayValues[29] = d29
					ps34.OverlayValues[30] = d30
					ps34.OverlayValues[32] = d32
					ps34.OverlayValues[33] = d33
					ps34.PhiValues = make([]JITValueDesc, 1)
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps34.PhiValues[0] = d35
					if ps34.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps34)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d36 := ps.PhiValues[0]
							ctx.EnsureDesc(&d36)
							ctx.EmitStoreToStack(d36, int32(bbs[3].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d37 JITValueDesc
					if d2.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d37)
					}
					if d37.Loc == LocReg && d2.Loc == LocReg && d37.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d37)
					ctx.EmitStoreToStack(d37, int32(bbs[3].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d37)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d33)
					var d38 JITValueDesc
					if d37.Loc == LocImm && d33.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() < d33.Imm.Int())}
					} else if d33.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d37.Reg)
						if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d37.Reg, int32(d33.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()))
							ctx.EmitCmpInt64(d37.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedLess)
						d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d38)
					} else if d37.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d33.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d38)
					} else {
						r3 := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitCmpInt64(d37.Reg, d33.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d38)
					}
					ctx.FreeDesc(&d33)
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							if ps.General {
							}
							ps40 := PhiState{General: ps.General}
							ps40.OverlayValues = make([]JITValueDesc, 40)
							ps40.OverlayValues[1] = d1
							ps40.OverlayValues[2] = d2
							ps40.OverlayValues[3] = d3
							ps40.OverlayValues[4] = d4
							ps40.OverlayValues[5] = d5
							ps40.OverlayValues[6] = d6
							ps40.OverlayValues[9] = d9
							ps40.OverlayValues[12] = d12
							ps40.OverlayValues[22] = d22
							ps40.OverlayValues[23] = d23
							ps40.OverlayValues[24] = d24
							ps40.OverlayValues[25] = d25
							ps40.OverlayValues[27] = d27
							ps40.OverlayValues[28] = d28
							ps40.OverlayValues[29] = d29
							ps40.OverlayValues[30] = d30
							ps40.OverlayValues[32] = d32
							ps40.OverlayValues[33] = d33
							ps40.OverlayValues[35] = d35
							ps40.OverlayValues[36] = d36
							ps40.OverlayValues[37] = d37
							ps40.OverlayValues[38] = d38
							ps40.OverlayValues[39] = d39
							return bbs[4].RenderPS(ps40)
						}
						if ps.General {
						}
						ps41 := PhiState{General: ps.General}
						ps41.OverlayValues = make([]JITValueDesc, 40)
						ps41.OverlayValues[1] = d1
						ps41.OverlayValues[2] = d2
						ps41.OverlayValues[3] = d3
						ps41.OverlayValues[4] = d4
						ps41.OverlayValues[5] = d5
						ps41.OverlayValues[6] = d6
						ps41.OverlayValues[9] = d9
						ps41.OverlayValues[12] = d12
						ps41.OverlayValues[22] = d22
						ps41.OverlayValues[23] = d23
						ps41.OverlayValues[24] = d24
						ps41.OverlayValues[25] = d25
						ps41.OverlayValues[27] = d27
						ps41.OverlayValues[28] = d28
						ps41.OverlayValues[29] = d29
						ps41.OverlayValues[30] = d30
						ps41.OverlayValues[32] = d32
						ps41.OverlayValues[33] = d33
						ps41.OverlayValues[35] = d35
						ps41.OverlayValues[36] = d36
						ps41.OverlayValues[37] = d37
						ps41.OverlayValues[38] = d38
						ps41.OverlayValues[39] = d39
						return bbs[5].RenderPS(ps41)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d42 := ps.PhiValues[0]
							ctx.EnsureDesc(&d42)
							ctx.EmitStoreToStack(d42, int32(bbs[3].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d39.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl6)
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 43)
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[4] = d4
					ps43.OverlayValues[5] = d5
					ps43.OverlayValues[6] = d6
					ps43.OverlayValues[9] = d9
					ps43.OverlayValues[12] = d12
					ps43.OverlayValues[22] = d22
					ps43.OverlayValues[23] = d23
					ps43.OverlayValues[24] = d24
					ps43.OverlayValues[25] = d25
					ps43.OverlayValues[27] = d27
					ps43.OverlayValues[28] = d28
					ps43.OverlayValues[29] = d29
					ps43.OverlayValues[30] = d30
					ps43.OverlayValues[32] = d32
					ps43.OverlayValues[33] = d33
					ps43.OverlayValues[35] = d35
					ps43.OverlayValues[36] = d36
					ps43.OverlayValues[37] = d37
					ps43.OverlayValues[38] = d38
					ps43.OverlayValues[39] = d39
					ps43.OverlayValues[42] = d42
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 43)
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[6] = d6
					ps44.OverlayValues[9] = d9
					ps44.OverlayValues[12] = d12
					ps44.OverlayValues[22] = d22
					ps44.OverlayValues[23] = d23
					ps44.OverlayValues[24] = d24
					ps44.OverlayValues[25] = d25
					ps44.OverlayValues[27] = d27
					ps44.OverlayValues[28] = d28
					ps44.OverlayValues[29] = d29
					ps44.OverlayValues[30] = d30
					ps44.OverlayValues[32] = d32
					ps44.OverlayValues[33] = d33
					ps44.OverlayValues[35] = d35
					ps44.OverlayValues[36] = d36
					ps44.OverlayValues[37] = d37
					ps44.OverlayValues[38] = d38
					ps44.OverlayValues[39] = d39
					ps44.OverlayValues[42] = d42
					snap45 := d1
					snap46 := d2
					snap47 := d3
					snap48 := d4
					snap49 := d5
					snap50 := d6
					snap51 := d9
					snap52 := d12
					snap53 := d22
					snap54 := d23
					snap55 := d24
					snap56 := d25
					snap57 := d27
					snap58 := d28
					snap59 := d29
					snap60 := d30
					snap61 := d32
					snap62 := d33
					snap63 := d35
					snap64 := d36
					snap65 := d37
					snap66 := d38
					snap67 := d39
					snap68 := d42
					alloc69 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps44)
					}
					ctx.RestoreAllocState(alloc69)
					d1 = snap45
					d2 = snap46
					d3 = snap47
					d4 = snap48
					d5 = snap49
					d6 = snap50
					d9 = snap51
					d12 = snap52
					d22 = snap53
					d23 = snap54
					d24 = snap55
					d25 = snap56
					d27 = snap57
					d28 = snap58
					d29 = snap59
					d30 = snap60
					d32 = snap61
					d33 = snap62
					d35 = snap63
					d36 = snap64
					d37 = snap65
					d38 = snap66
					d39 = snap67
					d42 = snap68
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps43)
					}
					return result
					ctx.FreeDesc(&d38)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					d71 = ctx.EmitSliceElementAddress(&d29, &d37, 8)
					ctx.EnsureDesc(&d71)
					ctx.EmitMovRegMem(d71.Reg, d71.Reg, 0)
					d70 = d71
					ctx.StabilizeDescForControlFlow(&d70)
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d72 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d70}, 2)
					ctx.BindReg(d72.Reg, &d72)
					ctx.BindReg(d72.Reg2, &d72)
					ctx.StabilizeDescForControlFlow(&d72)
					d73 = d1
					ctx.EnsureDesc(&d73)
					if d73.Loc != LocImm && d73.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d73.Loc == LocImm {
						if d73.Imm.Bool() {
							if ps.General {
								ctx.SyncDesc(&d72)
								if d72.Loc == LocReg {
									ctx.ProtectReg(d72.Reg)
								} else if d72.Loc == LocRegPair {
									ctx.ProtectReg(d72.Reg)
									ctx.ProtectReg(d72.Reg2)
								}
								d74 = d72
								if d74.Loc == LocNone {
									panic("jit: phi source has no location")
								}
								ctx.SyncDesc(&d74)
								if d74.Loc == LocStackPair {
									ctx.EmitCopyStackWords(d74, int32(bbs[7].PhiBase)+int32(0), 2)
								} else if d74.Loc == LocInputPair {
									ctx.EnsureDesc(&d74)
									ctx.EmitStoreScmerToStack(d74, int32(bbs[7].PhiBase)+int32(0))
								} else if d74.Loc == LocRegPair || d74.Loc == LocImm {
									ctx.EmitStoreScmerToStack(d74, int32(bbs[7].PhiBase)+int32(0))
								} else {
									ctx.EnsureDesc(&d74)
									ctx.EmitStoreToStack(d74, int32(bbs[7].PhiBase)+int32(0))
									ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
								}
								if d72.Loc == LocReg {
									ctx.UnprotectReg(d72.Reg)
								} else if d72.Loc == LocRegPair {
									ctx.UnprotectReg(d72.Reg)
									ctx.UnprotectReg(d72.Reg2)
								}
							}
							ps75 := PhiState{General: ps.General}
							ps75.OverlayValues = make([]JITValueDesc, 75)
							ps75.OverlayValues[1] = d1
							ps75.OverlayValues[2] = d2
							ps75.OverlayValues[3] = d3
							ps75.OverlayValues[4] = d4
							ps75.OverlayValues[5] = d5
							ps75.OverlayValues[6] = d6
							ps75.OverlayValues[9] = d9
							ps75.OverlayValues[12] = d12
							ps75.OverlayValues[22] = d22
							ps75.OverlayValues[23] = d23
							ps75.OverlayValues[24] = d24
							ps75.OverlayValues[25] = d25
							ps75.OverlayValues[27] = d27
							ps75.OverlayValues[28] = d28
							ps75.OverlayValues[29] = d29
							ps75.OverlayValues[30] = d30
							ps75.OverlayValues[32] = d32
							ps75.OverlayValues[33] = d33
							ps75.OverlayValues[35] = d35
							ps75.OverlayValues[36] = d36
							ps75.OverlayValues[37] = d37
							ps75.OverlayValues[38] = d38
							ps75.OverlayValues[39] = d39
							ps75.OverlayValues[42] = d42
							ps75.OverlayValues[70] = d70
							ps75.OverlayValues[71] = d71
							ps75.OverlayValues[72] = d72
							ps75.OverlayValues[73] = d73
							ps75.OverlayValues[74] = d74
							ps75.PhiValues = make([]JITValueDesc, 1)
							d76 = d72
							ps75.PhiValues[0] = d76
							return bbs[7].RenderPS(ps75)
						}
						if ps.General {
						}
						ps77 := PhiState{General: ps.General}
						ps77.OverlayValues = make([]JITValueDesc, 77)
						ps77.OverlayValues[1] = d1
						ps77.OverlayValues[2] = d2
						ps77.OverlayValues[3] = d3
						ps77.OverlayValues[4] = d4
						ps77.OverlayValues[5] = d5
						ps77.OverlayValues[6] = d6
						ps77.OverlayValues[9] = d9
						ps77.OverlayValues[12] = d12
						ps77.OverlayValues[22] = d22
						ps77.OverlayValues[23] = d23
						ps77.OverlayValues[24] = d24
						ps77.OverlayValues[25] = d25
						ps77.OverlayValues[27] = d27
						ps77.OverlayValues[28] = d28
						ps77.OverlayValues[29] = d29
						ps77.OverlayValues[30] = d30
						ps77.OverlayValues[32] = d32
						ps77.OverlayValues[33] = d33
						ps77.OverlayValues[35] = d35
						ps77.OverlayValues[36] = d36
						ps77.OverlayValues[37] = d37
						ps77.OverlayValues[38] = d38
						ps77.OverlayValues[39] = d39
						ps77.OverlayValues[42] = d42
						ps77.OverlayValues[70] = d70
						ps77.OverlayValues[71] = d71
						ps77.OverlayValues[72] = d72
						ps77.OverlayValues[73] = d73
						ps77.OverlayValues[74] = d74
						ps77.OverlayValues[76] = d76
						return bbs[8].RenderPS(ps77)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d73.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.SyncDesc(&d72)
					if d72.Loc == LocReg {
						ctx.ProtectReg(d72.Reg)
					} else if d72.Loc == LocRegPair {
						ctx.ProtectReg(d72.Reg)
						ctx.ProtectReg(d72.Reg2)
					}
					d78 = d72
					if d78.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d78)
					if d78.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d78, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d78.Loc == LocInputPair {
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreScmerToStack(d78, int32(bbs[7].PhiBase)+int32(0))
					} else if d78.Loc == LocRegPair || d78.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d78, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreToStack(d78, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d72.Loc == LocReg {
						ctx.UnprotectReg(d72.Reg)
					} else if d72.Loc == LocRegPair {
						ctx.UnprotectReg(d72.Reg)
						ctx.UnprotectReg(d72.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl9)
					ps79 := PhiState{General: true}
					ps79.OverlayValues = make([]JITValueDesc, 79)
					ps79.OverlayValues[1] = d1
					ps79.OverlayValues[2] = d2
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[6] = d6
					ps79.OverlayValues[9] = d9
					ps79.OverlayValues[12] = d12
					ps79.OverlayValues[22] = d22
					ps79.OverlayValues[23] = d23
					ps79.OverlayValues[24] = d24
					ps79.OverlayValues[25] = d25
					ps79.OverlayValues[27] = d27
					ps79.OverlayValues[28] = d28
					ps79.OverlayValues[29] = d29
					ps79.OverlayValues[30] = d30
					ps79.OverlayValues[32] = d32
					ps79.OverlayValues[33] = d33
					ps79.OverlayValues[35] = d35
					ps79.OverlayValues[36] = d36
					ps79.OverlayValues[37] = d37
					ps79.OverlayValues[38] = d38
					ps79.OverlayValues[39] = d39
					ps79.OverlayValues[42] = d42
					ps79.OverlayValues[70] = d70
					ps79.OverlayValues[71] = d71
					ps79.OverlayValues[72] = d72
					ps79.OverlayValues[73] = d73
					ps79.OverlayValues[74] = d74
					ps79.OverlayValues[76] = d76
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					d81 = d72
					ps79.PhiValues[0] = d81
					ps80 := PhiState{General: true}
					ps80.OverlayValues = make([]JITValueDesc, 82)
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[6] = d6
					ps80.OverlayValues[9] = d9
					ps80.OverlayValues[12] = d12
					ps80.OverlayValues[22] = d22
					ps80.OverlayValues[23] = d23
					ps80.OverlayValues[24] = d24
					ps80.OverlayValues[25] = d25
					ps80.OverlayValues[27] = d27
					ps80.OverlayValues[28] = d28
					ps80.OverlayValues[29] = d29
					ps80.OverlayValues[30] = d30
					ps80.OverlayValues[32] = d32
					ps80.OverlayValues[33] = d33
					ps80.OverlayValues[35] = d35
					ps80.OverlayValues[36] = d36
					ps80.OverlayValues[37] = d37
					ps80.OverlayValues[38] = d38
					ps80.OverlayValues[39] = d39
					ps80.OverlayValues[42] = d42
					ps80.OverlayValues[70] = d70
					ps80.OverlayValues[71] = d71
					ps80.OverlayValues[72] = d72
					ps80.OverlayValues[73] = d73
					ps80.OverlayValues[74] = d74
					ps80.OverlayValues[76] = d76
					ps80.OverlayValues[78] = d78
					ps80.OverlayValues[81] = d81
					snap82 := d1
					snap83 := d2
					snap84 := d3
					snap85 := d4
					snap86 := d5
					snap87 := d6
					snap88 := d9
					snap89 := d12
					snap90 := d22
					snap91 := d23
					snap92 := d24
					snap93 := d25
					snap94 := d27
					snap95 := d28
					snap96 := d29
					snap97 := d30
					snap98 := d32
					snap99 := d33
					snap100 := d35
					snap101 := d36
					snap102 := d37
					snap103 := d38
					snap104 := d39
					snap105 := d42
					snap106 := d70
					snap107 := d71
					snap108 := d72
					snap109 := d73
					snap110 := d74
					snap111 := d76
					snap112 := d78
					snap113 := d81
					alloc114 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps79)
					}
					ctx.RestoreAllocState(alloc114)
					d1 = snap82
					d2 = snap83
					d3 = snap84
					d4 = snap85
					d5 = snap86
					d6 = snap87
					d9 = snap88
					d12 = snap89
					d22 = snap90
					d23 = snap91
					d24 = snap92
					d25 = snap93
					d27 = snap94
					d28 = snap95
					d29 = snap96
					d30 = snap97
					d32 = snap98
					d33 = snap99
					d35 = snap100
					d36 = snap101
					d37 = snap102
					d38 = snap103
					d39 = snap104
					d42 = snap105
					d70 = snap106
					d71 = snap107
					d72 = snap108
					d73 = snap109
					d74 = snap110
					d76 = snap111
					d78 = snap112
					d81 = snap113
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps80)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs115 := make([]Reg, 0, 3)
					seenBlockPinnedRegs116 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs116
					for _, r := range []Reg{d32.Reg, d32.Reg2, d32.Reg3} {
						live := d32.Loc == LocRegTriple && (r == d32.Reg || r == d32.Reg2 || r == d32.Reg3)
						if live && !seenBlockPinnedRegs116[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs116[r] = true
							blockPinnedRegs115 = append(blockPinnedRegs115, r)
						}
					}
					unpinBlockRegs117 := func() {
						for _, r := range blockPinnedRegs115 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs117()
					d118 = ctx.EmitNewSliceFromGoSlice(&d32)
					ctx.EnsureDesc(&d118)
					if d118.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d118, &result)
						result.Type = d118.Type
					} else {
						switch d118.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d118)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d118)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d118)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d118, &result)
							result.Type = d118.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					ctx.ReclaimUntrackedRegs()
					d119 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d120 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d119)
					ctx.EnsureDesc(&d120)
					var d122 JITValueDesc
					if d120.Loc == LocImm && d119.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d120.Imm.Int() - d119.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d120.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d120.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d120.Reg)
						}
						if d119.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d119.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d119.Reg)
						}
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d122)
					}
					var d123 JITValueDesc
					if d72.Loc == LocImm && d119.Loc == LocImm {
						d123 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d72.Imm.Int() + d119.Imm.Int()*1)}
					} else {
						r5 := ctx.AllocReg()
						if d72.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d72.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d72.Reg)
						}
						if d119.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d119.Imm.Int()*1))
							ctx.EmitAddInt64(r5, RegR11)
						} else {
							ctx.EmitAddInt64(r5, d119.Reg)
						}
						d123 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d123)
					}
					var d124 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d123)
					ctx.EnsureDesc(&d123)
					if d123.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d123.Imm.Int()))
					} else {
						r6 = d123.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d122)
					ctx.EnsureDesc(&d122)
					if d122.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d122.Imm.Int()))
					} else {
						r7 = d122.Reg
					}
					ctx.ProtectReg(r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d124 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d124)
					ctx.BindReg(r7, &d124)
					ctx.BindReg(r6, &d124)
					ctx.BindReg(r7, &d124)
					ctx.StabilizeDescForControlFlow(&d124)
					if ps.General {
						ctx.SyncDesc(&d124)
						if d124.Loc == LocReg {
							ctx.ProtectReg(d124.Reg)
						} else if d124.Loc == LocRegPair {
							ctx.ProtectReg(d124.Reg)
							ctx.ProtectReg(d124.Reg2)
						}
						d125 = d124
						if d125.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d125)
						if d125.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d125, int32(bbs[7].PhiBase)+int32(0), 2)
						} else if d125.Loc == LocInputPair {
							ctx.EnsureDesc(&d125)
							ctx.EmitStoreScmerToStack(d125, int32(bbs[7].PhiBase)+int32(0))
						} else if d125.Loc == LocRegPair || d125.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d125, int32(bbs[7].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d125)
							ctx.EmitStoreToStack(d125, int32(bbs[7].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
						}
						if d124.Loc == LocReg {
							ctx.UnprotectReg(d124.Reg)
						} else if d124.Loc == LocRegPair {
							ctx.UnprotectReg(d124.Reg)
							ctx.UnprotectReg(d124.Reg2)
						}
					}
					ps126 := PhiState{General: ps.General}
					ps126.OverlayValues = make([]JITValueDesc, 126)
					ps126.OverlayValues[1] = d1
					ps126.OverlayValues[2] = d2
					ps126.OverlayValues[3] = d3
					ps126.OverlayValues[4] = d4
					ps126.OverlayValues[5] = d5
					ps126.OverlayValues[6] = d6
					ps126.OverlayValues[9] = d9
					ps126.OverlayValues[12] = d12
					ps126.OverlayValues[22] = d22
					ps126.OverlayValues[23] = d23
					ps126.OverlayValues[24] = d24
					ps126.OverlayValues[25] = d25
					ps126.OverlayValues[27] = d27
					ps126.OverlayValues[28] = d28
					ps126.OverlayValues[29] = d29
					ps126.OverlayValues[30] = d30
					ps126.OverlayValues[32] = d32
					ps126.OverlayValues[33] = d33
					ps126.OverlayValues[35] = d35
					ps126.OverlayValues[36] = d36
					ps126.OverlayValues[37] = d37
					ps126.OverlayValues[38] = d38
					ps126.OverlayValues[39] = d39
					ps126.OverlayValues[42] = d42
					ps126.OverlayValues[70] = d70
					ps126.OverlayValues[71] = d71
					ps126.OverlayValues[72] = d72
					ps126.OverlayValues[73] = d73
					ps126.OverlayValues[74] = d74
					ps126.OverlayValues[76] = d76
					ps126.OverlayValues[78] = d78
					ps126.OverlayValues[81] = d81
					ps126.OverlayValues[118] = d118
					ps126.OverlayValues[119] = d119
					ps126.OverlayValues[120] = d120
					ps126.OverlayValues[121] = d121
					ps126.OverlayValues[122] = d122
					ps126.OverlayValues[123] = d123
					ps126.OverlayValues[124] = d124
					ps126.OverlayValues[125] = d125
					ps126.PhiValues = make([]JITValueDesc, 1)
					d127 = d124
					ps126.PhiValues[0] = d127
					if ps126.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps126)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d128 := ps.PhiValues[0]
							ctx.EnsureDesc(&d128)
							ctx.EmitStoreScmerToStack(d128, int32(bbs[7].PhiBase)+int32(0))
						}
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs129 := make([]Reg, 0, 3)
					seenBlockPinnedRegs130 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs130
					for _, r := range []Reg{d32.Reg, d32.Reg2, d32.Reg3} {
						live := d32.Loc == LocRegTriple && (r == d32.Reg || r == d32.Reg2 || r == d32.Reg3)
						if live && !seenBlockPinnedRegs130[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs130[r] = true
							blockPinnedRegs129 = append(blockPinnedRegs129, r)
						}
					}
					unpinBlockRegs131 := func() {
						for _, r := range blockPinnedRegs129 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs131()
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d70)
					d132 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d70}, 2)
					ctx.BindReg(d132.Reg, &d132)
					ctx.BindReg(d132.Reg2, &d132)
					stackArray133 = ctx.AllocStack(int32(256))
					_ = stackArray133
					d134 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.EnsureDesc(&d134)
					ctx.EnsureDesc(&d134)
					ctx.EmitStoreScmerToStack(d134, int32(stackArray133)+int32(0))
					var d135 JITValueDesc
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocImm {
						fieldAddr := uintptr(d70.Imm.Int()) + 0
						r8 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r8, fieldAddr)
						d135 = JITValueDesc{Loc: LocReg, Reg: r8}
						ctx.BindReg(r8, &d135)
					} else {
						off := int32(0)
						baseReg := d70.Reg
						r9 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r9, baseReg, off)
						d135 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d135)
					}
					ctx.EnsureDesc(&d135)
					ctx.EnsureDesc(&d135)
					var d136 JITValueDesc
					if d135.Loc == LocImm {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d135.Imm.Int()))))}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegReg(r10, d135.Reg)
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d136)
					}
					ctx.FreeDesc(&d135)
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d136)
					ctx.EnsureDesc(&d136)
					ctx.EmitStoreTypedScmerToStack(d136, tagInt, int32(stackArray133)+int32(16))
					d137 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.EnsureDesc(&d137)
					ctx.EnsureDesc(&d137)
					ctx.EmitStoreScmerToStack(d137, int32(stackArray133)+int32(32))
					var d138 JITValueDesc
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocImm {
						fieldAddr := uintptr(d70.Imm.Int()) + 8
						r11 := ctx.AllocReg()
						r12 := ctx.AllocRegExcept(r11)
						r13 := ctx.AllocRegExcept(r11, r12)
						ctx.EmitMovRegMem64(r11, fieldAddr)
						ctx.EmitMovRegMem64(r12, fieldAddr+8)
						ctx.EmitMovRegMem64(r13, fieldAddr+16)
						d138 = JITValueDesc{Loc: LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
						ctx.BindReg(r11, &d138)
						ctx.BindReg(r12, &d138)
						ctx.BindReg(r13, &d138)
					} else {
						off := int32(8)
						baseReg := d70.Reg
						r14 := ctx.AllocRegExcept(baseReg)
						r15 := ctx.AllocRegExcept(baseReg, r14)
						r16 := ctx.AllocRegExcept(baseReg, r14, r15)
						ctx.EmitMovRegMem(r14, baseReg, off)
						ctx.EmitMovRegMem(r15, baseReg, off+8)
						ctx.EmitMovRegMem(r16, baseReg, off+16)
						d138 = JITValueDesc{Loc: LocRegTriple, Reg: r14, Reg2: r15, Reg3: r16}
						ctx.BindReg(r14, &d138)
						ctx.BindReg(r15, &d138)
						ctx.BindReg(r16, &d138)
					}
					ctx.EnsureDesc(&d138)
					ctx.EnsureDesc(&d138)
					ctx.EnsureDesc(&d138)
					ctx.EmitStoreScmerToStack(d138, int32(stackArray133)+int32(48))
					d139 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.EnsureDesc(&d139)
					ctx.EnsureDesc(&d139)
					ctx.EmitStoreScmerToStack(d139, int32(stackArray133)+int32(64))
					var d140 JITValueDesc
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocImm {
						fieldAddr := uintptr(d70.Imm.Int()) + 24
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d140 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d140)
						ctx.BindReg(r18, &d140)
						ctx.BindReg(r19, &d140)
					} else {
						off := int32(24)
						baseReg := d70.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d140 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d140)
						ctx.BindReg(r21, &d140)
						ctx.BindReg(r22, &d140)
					}
					ctx.EnsureDesc(&d140)
					ctx.EnsureDesc(&d140)
					ctx.EnsureDesc(&d140)
					ctx.EmitStoreScmerToStack(d140, int32(stackArray133)+int32(80))
					d141 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.EnsureDesc(&d141)
					ctx.EnsureDesc(&d141)
					ctx.EmitStoreScmerToStack(d141, int32(stackArray133)+int32(96))
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d142 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d70}, 2)
					ctx.BindReg(d142.Reg, &d142)
					ctx.BindReg(d142.Reg2, &d142)
					ctx.EnsureDesc(&d142)
					ctx.EnsureDesc(&d142)
					ctx.EnsureDesc(&d142)
					ctx.EmitStoreScmerToStack(d142, int32(stackArray133)+int32(112))
					d143 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.EnsureDesc(&d143)
					ctx.EnsureDesc(&d143)
					ctx.EmitStoreScmerToStack(d143, int32(stackArray133)+int32(128))
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d144 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d70}, 2)
					ctx.BindReg(d144.Reg, &d144)
					ctx.BindReg(d144.Reg2, &d144)
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d144)
					ctx.EmitStoreScmerToStack(d144, int32(stackArray133)+int32(144))
					d145 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.EnsureDesc(&d145)
					ctx.EnsureDesc(&d145)
					ctx.EmitStoreScmerToStack(d145, int32(stackArray133)+int32(160))
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocRegPair || d70.Loc == LocStackPair || d70.Loc == LocRegTriple || d70.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d70)
					d146 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d70}, 1)
					ctx.BindReg(d146.Reg, &d146)
					ctx.EnsureDesc(&d146)
					ctx.EnsureDesc(&d146)
					ctx.EnsureDesc(&d146)
					ctx.EmitStoreTypedScmerToStack(d146, tagInt, int32(stackArray133)+int32(176))
					d147 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.EnsureDesc(&d147)
					ctx.EnsureDesc(&d147)
					ctx.EmitStoreScmerToStack(d147, int32(stackArray133)+int32(192))
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d132)
					ctx.EmitStoreScmerToStack(d132, int32(stackArray133)+int32(208))
					d148 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.EnsureDesc(&d148)
					ctx.EnsureDesc(&d148)
					ctx.EmitStoreScmerToStack(d148, int32(stackArray133)+int32(224))
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EmitStoreScmerToStack(d3, int32(stackArray133)+int32(240))
					d149 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d149
					r23 := ctx.AllocReg()
					r24 := ctx.AllocRegExcept(r23)
					r25 := ctx.AllocRegExcept(r23, r24)
					d150 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r23, Reg2: r24, Reg3: r25}
					ctx.BindReg(r23, &d150)
					ctx.BindReg(r24, &d150)
					ctx.BindReg(r25, &d150)
					ctx.BindReg(r23, &d150)
					ctx.BindReg(r24, &d150)
					ctx.BindReg(r25, &d150)
					ctx.EmitLeaRegMem(d150.Reg, ctx.StackReg, int32(stackArray133))
					ctx.EmitMovRegImm64(d150.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d150.Reg3, uint64(16))
					callResults151 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d150}, []uint8{2}, []uint8{1})
					d152 = callResults151[0]
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d152)
					d153 = ctx.EmitSliceElementAddress(&d32, &d37, int32(16))
					ctx.EmitStoreScmerAt(&d153, &d152)
					ctx.FreeDesc(&d153)
					ctx.FreeDesc(&d152)
					if ps.General {
					}
					ps154 := PhiState{General: ps.General}
					ps154.OverlayValues = make([]JITValueDesc, 154)
					ps154.OverlayValues[1] = d1
					ps154.OverlayValues[2] = d2
					ps154.OverlayValues[3] = d3
					ps154.OverlayValues[4] = d4
					ps154.OverlayValues[5] = d5
					ps154.OverlayValues[6] = d6
					ps154.OverlayValues[9] = d9
					ps154.OverlayValues[12] = d12
					ps154.OverlayValues[22] = d22
					ps154.OverlayValues[23] = d23
					ps154.OverlayValues[24] = d24
					ps154.OverlayValues[25] = d25
					ps154.OverlayValues[27] = d27
					ps154.OverlayValues[28] = d28
					ps154.OverlayValues[29] = d29
					ps154.OverlayValues[30] = d30
					ps154.OverlayValues[32] = d32
					ps154.OverlayValues[33] = d33
					ps154.OverlayValues[35] = d35
					ps154.OverlayValues[36] = d36
					ps154.OverlayValues[37] = d37
					ps154.OverlayValues[38] = d38
					ps154.OverlayValues[39] = d39
					ps154.OverlayValues[42] = d42
					ps154.OverlayValues[70] = d70
					ps154.OverlayValues[71] = d71
					ps154.OverlayValues[72] = d72
					ps154.OverlayValues[73] = d73
					ps154.OverlayValues[74] = d74
					ps154.OverlayValues[76] = d76
					ps154.OverlayValues[78] = d78
					ps154.OverlayValues[81] = d81
					ps154.OverlayValues[118] = d118
					ps154.OverlayValues[119] = d119
					ps154.OverlayValues[120] = d120
					ps154.OverlayValues[121] = d121
					ps154.OverlayValues[122] = d122
					ps154.OverlayValues[123] = d123
					ps154.OverlayValues[124] = d124
					ps154.OverlayValues[125] = d125
					ps154.OverlayValues[127] = d127
					ps154.OverlayValues[128] = d128
					ps154.OverlayValues[132] = d132
					ps154.OverlayValues[134] = d134
					ps154.OverlayValues[135] = d135
					ps154.OverlayValues[136] = d136
					ps154.OverlayValues[137] = d137
					ps154.OverlayValues[138] = d138
					ps154.OverlayValues[139] = d139
					ps154.OverlayValues[140] = d140
					ps154.OverlayValues[141] = d141
					ps154.OverlayValues[142] = d142
					ps154.OverlayValues[143] = d143
					ps154.OverlayValues[144] = d144
					ps154.OverlayValues[145] = d145
					ps154.OverlayValues[146] = d146
					ps154.OverlayValues[147] = d147
					ps154.OverlayValues[148] = d148
					ps154.OverlayValues[149] = d149
					ps154.OverlayValues[150] = d150
					ps154.OverlayValues[152] = d152
					ps154.OverlayValues[153] = d153
					ps154.PhiValues = make([]JITValueDesc, 1)
					if ps154.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps154)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
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
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					ctx.ReclaimUntrackedRegs()
					var d155 JITValueDesc
					if d72.SliceSizeKnown {
						d155 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d72.KnownSliceLen))}
					} else if d72.Loc == LocImm {
						d155 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d72.Imm.String())))}
					} else if d72.Loc == LocStackTriple {
						d155 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d72.StackOff + 8, NoHeapPointer: true}
					} else if d72.Loc == LocStackPair {
						d155 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d72.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d72)
						if d72.Loc == LocRegPair || d72.Loc == LocRegTriple {
							d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg2, ID: 0}
						} else if d72.Loc == LocReg {
							d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d155)
					var d156 JITValueDesc
					if d155.Loc == LocImm {
						d156 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d155.Imm.Int() > 100)}
					} else {
						r26 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d155.Reg, 100)
						ctx.EmitSetcc(r26, CondSignedGreater)
						d156 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d156)
					}
					ctx.FreeDesc(&d155)
					d157 = d156
					ctx.EnsureDesc(&d157)
					if d157.Loc != LocImm && d157.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d157.Loc == LocImm {
						if d157.Imm.Bool() {
							if ps.General {
							}
							ps158 := PhiState{General: ps.General}
							ps158.OverlayValues = make([]JITValueDesc, 158)
							ps158.OverlayValues[1] = d1
							ps158.OverlayValues[2] = d2
							ps158.OverlayValues[3] = d3
							ps158.OverlayValues[4] = d4
							ps158.OverlayValues[5] = d5
							ps158.OverlayValues[6] = d6
							ps158.OverlayValues[9] = d9
							ps158.OverlayValues[12] = d12
							ps158.OverlayValues[22] = d22
							ps158.OverlayValues[23] = d23
							ps158.OverlayValues[24] = d24
							ps158.OverlayValues[25] = d25
							ps158.OverlayValues[27] = d27
							ps158.OverlayValues[28] = d28
							ps158.OverlayValues[29] = d29
							ps158.OverlayValues[30] = d30
							ps158.OverlayValues[32] = d32
							ps158.OverlayValues[33] = d33
							ps158.OverlayValues[35] = d35
							ps158.OverlayValues[36] = d36
							ps158.OverlayValues[37] = d37
							ps158.OverlayValues[38] = d38
							ps158.OverlayValues[39] = d39
							ps158.OverlayValues[42] = d42
							ps158.OverlayValues[70] = d70
							ps158.OverlayValues[71] = d71
							ps158.OverlayValues[72] = d72
							ps158.OverlayValues[73] = d73
							ps158.OverlayValues[74] = d74
							ps158.OverlayValues[76] = d76
							ps158.OverlayValues[78] = d78
							ps158.OverlayValues[81] = d81
							ps158.OverlayValues[118] = d118
							ps158.OverlayValues[119] = d119
							ps158.OverlayValues[120] = d120
							ps158.OverlayValues[121] = d121
							ps158.OverlayValues[122] = d122
							ps158.OverlayValues[123] = d123
							ps158.OverlayValues[124] = d124
							ps158.OverlayValues[125] = d125
							ps158.OverlayValues[127] = d127
							ps158.OverlayValues[128] = d128
							ps158.OverlayValues[132] = d132
							ps158.OverlayValues[134] = d134
							ps158.OverlayValues[135] = d135
							ps158.OverlayValues[136] = d136
							ps158.OverlayValues[137] = d137
							ps158.OverlayValues[138] = d138
							ps158.OverlayValues[139] = d139
							ps158.OverlayValues[140] = d140
							ps158.OverlayValues[141] = d141
							ps158.OverlayValues[142] = d142
							ps158.OverlayValues[143] = d143
							ps158.OverlayValues[144] = d144
							ps158.OverlayValues[145] = d145
							ps158.OverlayValues[146] = d146
							ps158.OverlayValues[147] = d147
							ps158.OverlayValues[148] = d148
							ps158.OverlayValues[149] = d149
							ps158.OverlayValues[150] = d150
							ps158.OverlayValues[152] = d152
							ps158.OverlayValues[153] = d153
							ps158.OverlayValues[155] = d155
							ps158.OverlayValues[156] = d156
							ps158.OverlayValues[157] = d157
							return bbs[6].RenderPS(ps158)
						}
						if ps.General {
							ctx.SyncDesc(&d72)
							if d72.Loc == LocReg {
								ctx.ProtectReg(d72.Reg)
							} else if d72.Loc == LocRegPair {
								ctx.ProtectReg(d72.Reg)
								ctx.ProtectReg(d72.Reg2)
							}
							d159 = d72
							if d159.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d159)
							if d159.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d159, int32(bbs[7].PhiBase)+int32(0), 2)
							} else if d159.Loc == LocInputPair {
								ctx.EnsureDesc(&d159)
								ctx.EmitStoreScmerToStack(d159, int32(bbs[7].PhiBase)+int32(0))
							} else if d159.Loc == LocRegPair || d159.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d159, int32(bbs[7].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d159)
								ctx.EmitStoreToStack(d159, int32(bbs[7].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
							}
							if d72.Loc == LocReg {
								ctx.UnprotectReg(d72.Reg)
							} else if d72.Loc == LocRegPair {
								ctx.UnprotectReg(d72.Reg)
								ctx.UnprotectReg(d72.Reg2)
							}
						}
						ps160 := PhiState{General: ps.General}
						ps160.OverlayValues = make([]JITValueDesc, 160)
						ps160.OverlayValues[1] = d1
						ps160.OverlayValues[2] = d2
						ps160.OverlayValues[3] = d3
						ps160.OverlayValues[4] = d4
						ps160.OverlayValues[5] = d5
						ps160.OverlayValues[6] = d6
						ps160.OverlayValues[9] = d9
						ps160.OverlayValues[12] = d12
						ps160.OverlayValues[22] = d22
						ps160.OverlayValues[23] = d23
						ps160.OverlayValues[24] = d24
						ps160.OverlayValues[25] = d25
						ps160.OverlayValues[27] = d27
						ps160.OverlayValues[28] = d28
						ps160.OverlayValues[29] = d29
						ps160.OverlayValues[30] = d30
						ps160.OverlayValues[32] = d32
						ps160.OverlayValues[33] = d33
						ps160.OverlayValues[35] = d35
						ps160.OverlayValues[36] = d36
						ps160.OverlayValues[37] = d37
						ps160.OverlayValues[38] = d38
						ps160.OverlayValues[39] = d39
						ps160.OverlayValues[42] = d42
						ps160.OverlayValues[70] = d70
						ps160.OverlayValues[71] = d71
						ps160.OverlayValues[72] = d72
						ps160.OverlayValues[73] = d73
						ps160.OverlayValues[74] = d74
						ps160.OverlayValues[76] = d76
						ps160.OverlayValues[78] = d78
						ps160.OverlayValues[81] = d81
						ps160.OverlayValues[118] = d118
						ps160.OverlayValues[119] = d119
						ps160.OverlayValues[120] = d120
						ps160.OverlayValues[121] = d121
						ps160.OverlayValues[122] = d122
						ps160.OverlayValues[123] = d123
						ps160.OverlayValues[124] = d124
						ps160.OverlayValues[125] = d125
						ps160.OverlayValues[127] = d127
						ps160.OverlayValues[128] = d128
						ps160.OverlayValues[132] = d132
						ps160.OverlayValues[134] = d134
						ps160.OverlayValues[135] = d135
						ps160.OverlayValues[136] = d136
						ps160.OverlayValues[137] = d137
						ps160.OverlayValues[138] = d138
						ps160.OverlayValues[139] = d139
						ps160.OverlayValues[140] = d140
						ps160.OverlayValues[141] = d141
						ps160.OverlayValues[142] = d142
						ps160.OverlayValues[143] = d143
						ps160.OverlayValues[144] = d144
						ps160.OverlayValues[145] = d145
						ps160.OverlayValues[146] = d146
						ps160.OverlayValues[147] = d147
						ps160.OverlayValues[148] = d148
						ps160.OverlayValues[149] = d149
						ps160.OverlayValues[150] = d150
						ps160.OverlayValues[152] = d152
						ps160.OverlayValues[153] = d153
						ps160.OverlayValues[155] = d155
						ps160.OverlayValues[156] = d156
						ps160.OverlayValues[157] = d157
						ps160.OverlayValues[159] = d159
						ps160.PhiValues = make([]JITValueDesc, 1)
						d161 = d72
						ps160.PhiValues[0] = d161
						return bbs[7].RenderPS(ps160)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d157.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl17)
					ctx.SyncDesc(&d72)
					if d72.Loc == LocReg {
						ctx.ProtectReg(d72.Reg)
					} else if d72.Loc == LocRegPair {
						ctx.ProtectReg(d72.Reg)
						ctx.ProtectReg(d72.Reg2)
					}
					d162 = d72
					if d162.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d162)
					if d162.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d162, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d162.Loc == LocInputPair {
						ctx.EnsureDesc(&d162)
						ctx.EmitStoreScmerToStack(d162, int32(bbs[7].PhiBase)+int32(0))
					} else if d162.Loc == LocRegPair || d162.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d162, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d162)
						ctx.EmitStoreToStack(d162, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d72.Loc == LocReg {
						ctx.UnprotectReg(d72.Reg)
					} else if d72.Loc == LocRegPair {
						ctx.UnprotectReg(d72.Reg)
						ctx.UnprotectReg(d72.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ps163 := PhiState{General: true}
					ps163.OverlayValues = make([]JITValueDesc, 163)
					ps163.OverlayValues[1] = d1
					ps163.OverlayValues[2] = d2
					ps163.OverlayValues[3] = d3
					ps163.OverlayValues[4] = d4
					ps163.OverlayValues[5] = d5
					ps163.OverlayValues[6] = d6
					ps163.OverlayValues[9] = d9
					ps163.OverlayValues[12] = d12
					ps163.OverlayValues[22] = d22
					ps163.OverlayValues[23] = d23
					ps163.OverlayValues[24] = d24
					ps163.OverlayValues[25] = d25
					ps163.OverlayValues[27] = d27
					ps163.OverlayValues[28] = d28
					ps163.OverlayValues[29] = d29
					ps163.OverlayValues[30] = d30
					ps163.OverlayValues[32] = d32
					ps163.OverlayValues[33] = d33
					ps163.OverlayValues[35] = d35
					ps163.OverlayValues[36] = d36
					ps163.OverlayValues[37] = d37
					ps163.OverlayValues[38] = d38
					ps163.OverlayValues[39] = d39
					ps163.OverlayValues[42] = d42
					ps163.OverlayValues[70] = d70
					ps163.OverlayValues[71] = d71
					ps163.OverlayValues[72] = d72
					ps163.OverlayValues[73] = d73
					ps163.OverlayValues[74] = d74
					ps163.OverlayValues[76] = d76
					ps163.OverlayValues[78] = d78
					ps163.OverlayValues[81] = d81
					ps163.OverlayValues[118] = d118
					ps163.OverlayValues[119] = d119
					ps163.OverlayValues[120] = d120
					ps163.OverlayValues[121] = d121
					ps163.OverlayValues[122] = d122
					ps163.OverlayValues[123] = d123
					ps163.OverlayValues[124] = d124
					ps163.OverlayValues[125] = d125
					ps163.OverlayValues[127] = d127
					ps163.OverlayValues[128] = d128
					ps163.OverlayValues[132] = d132
					ps163.OverlayValues[134] = d134
					ps163.OverlayValues[135] = d135
					ps163.OverlayValues[136] = d136
					ps163.OverlayValues[137] = d137
					ps163.OverlayValues[138] = d138
					ps163.OverlayValues[139] = d139
					ps163.OverlayValues[140] = d140
					ps163.OverlayValues[141] = d141
					ps163.OverlayValues[142] = d142
					ps163.OverlayValues[143] = d143
					ps163.OverlayValues[144] = d144
					ps163.OverlayValues[145] = d145
					ps163.OverlayValues[146] = d146
					ps163.OverlayValues[147] = d147
					ps163.OverlayValues[148] = d148
					ps163.OverlayValues[149] = d149
					ps163.OverlayValues[150] = d150
					ps163.OverlayValues[152] = d152
					ps163.OverlayValues[153] = d153
					ps163.OverlayValues[155] = d155
					ps163.OverlayValues[156] = d156
					ps163.OverlayValues[157] = d157
					ps163.OverlayValues[159] = d159
					ps163.OverlayValues[161] = d161
					ps163.OverlayValues[162] = d162
					ps164 := PhiState{General: true}
					ps164.OverlayValues = make([]JITValueDesc, 163)
					ps164.OverlayValues[1] = d1
					ps164.OverlayValues[2] = d2
					ps164.OverlayValues[3] = d3
					ps164.OverlayValues[4] = d4
					ps164.OverlayValues[5] = d5
					ps164.OverlayValues[6] = d6
					ps164.OverlayValues[9] = d9
					ps164.OverlayValues[12] = d12
					ps164.OverlayValues[22] = d22
					ps164.OverlayValues[23] = d23
					ps164.OverlayValues[24] = d24
					ps164.OverlayValues[25] = d25
					ps164.OverlayValues[27] = d27
					ps164.OverlayValues[28] = d28
					ps164.OverlayValues[29] = d29
					ps164.OverlayValues[30] = d30
					ps164.OverlayValues[32] = d32
					ps164.OverlayValues[33] = d33
					ps164.OverlayValues[35] = d35
					ps164.OverlayValues[36] = d36
					ps164.OverlayValues[37] = d37
					ps164.OverlayValues[38] = d38
					ps164.OverlayValues[39] = d39
					ps164.OverlayValues[42] = d42
					ps164.OverlayValues[70] = d70
					ps164.OverlayValues[71] = d71
					ps164.OverlayValues[72] = d72
					ps164.OverlayValues[73] = d73
					ps164.OverlayValues[74] = d74
					ps164.OverlayValues[76] = d76
					ps164.OverlayValues[78] = d78
					ps164.OverlayValues[81] = d81
					ps164.OverlayValues[118] = d118
					ps164.OverlayValues[119] = d119
					ps164.OverlayValues[120] = d120
					ps164.OverlayValues[121] = d121
					ps164.OverlayValues[122] = d122
					ps164.OverlayValues[123] = d123
					ps164.OverlayValues[124] = d124
					ps164.OverlayValues[125] = d125
					ps164.OverlayValues[127] = d127
					ps164.OverlayValues[128] = d128
					ps164.OverlayValues[132] = d132
					ps164.OverlayValues[134] = d134
					ps164.OverlayValues[135] = d135
					ps164.OverlayValues[136] = d136
					ps164.OverlayValues[137] = d137
					ps164.OverlayValues[138] = d138
					ps164.OverlayValues[139] = d139
					ps164.OverlayValues[140] = d140
					ps164.OverlayValues[141] = d141
					ps164.OverlayValues[142] = d142
					ps164.OverlayValues[143] = d143
					ps164.OverlayValues[144] = d144
					ps164.OverlayValues[145] = d145
					ps164.OverlayValues[146] = d146
					ps164.OverlayValues[147] = d147
					ps164.OverlayValues[148] = d148
					ps164.OverlayValues[149] = d149
					ps164.OverlayValues[150] = d150
					ps164.OverlayValues[152] = d152
					ps164.OverlayValues[153] = d153
					ps164.OverlayValues[155] = d155
					ps164.OverlayValues[156] = d156
					ps164.OverlayValues[157] = d157
					ps164.OverlayValues[159] = d159
					ps164.OverlayValues[161] = d161
					ps164.OverlayValues[162] = d162
					ps164.PhiValues = make([]JITValueDesc, 1)
					d165 = d72
					ps164.PhiValues[0] = d165
					snap166 := d1
					snap167 := d2
					snap168 := d3
					snap169 := d4
					snap170 := d5
					snap171 := d6
					snap172 := d9
					snap173 := d12
					snap174 := d22
					snap175 := d23
					snap176 := d24
					snap177 := d25
					snap178 := d27
					snap179 := d28
					snap180 := d29
					snap181 := d30
					snap182 := d32
					snap183 := d33
					snap184 := d35
					snap185 := d36
					snap186 := d37
					snap187 := d38
					snap188 := d39
					snap189 := d42
					snap190 := d70
					snap191 := d71
					snap192 := d72
					snap193 := d73
					snap194 := d74
					snap195 := d76
					snap196 := d78
					snap197 := d81
					snap198 := d118
					snap199 := d119
					snap200 := d120
					snap201 := d121
					snap202 := d122
					snap203 := d123
					snap204 := d124
					snap205 := d125
					snap206 := d127
					snap207 := d128
					snap208 := d132
					snap209 := d134
					snap210 := d135
					snap211 := d136
					snap212 := d137
					snap213 := d138
					snap214 := d139
					snap215 := d140
					snap216 := d141
					snap217 := d142
					snap218 := d143
					snap219 := d144
					snap220 := d145
					snap221 := d146
					snap222 := d147
					snap223 := d148
					snap224 := d149
					snap225 := d150
					snap226 := d152
					snap227 := d153
					snap228 := d155
					snap229 := d156
					snap230 := d157
					snap231 := d159
					snap232 := d161
					snap233 := d162
					snap234 := d165
					alloc235 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps164)
					}
					ctx.RestoreAllocState(alloc235)
					d1 = snap166
					d2 = snap167
					d3 = snap168
					d4 = snap169
					d5 = snap170
					d6 = snap171
					d9 = snap172
					d12 = snap173
					d22 = snap174
					d23 = snap175
					d24 = snap176
					d25 = snap177
					d27 = snap178
					d28 = snap179
					d29 = snap180
					d30 = snap181
					d32 = snap182
					d33 = snap183
					d35 = snap184
					d36 = snap185
					d37 = snap186
					d38 = snap187
					d39 = snap188
					d42 = snap189
					d70 = snap190
					d71 = snap191
					d72 = snap192
					d73 = snap193
					d74 = snap194
					d76 = snap195
					d78 = snap196
					d81 = snap197
					d118 = snap198
					d119 = snap199
					d120 = snap200
					d121 = snap201
					d122 = snap202
					d123 = snap203
					d124 = snap204
					d125 = snap205
					d127 = snap206
					d128 = snap207
					d132 = snap208
					d134 = snap209
					d135 = snap210
					d136 = snap211
					d137 = snap212
					d138 = snap213
					d139 = snap214
					d140 = snap215
					d141 = snap216
					d142 = snap217
					d143 = snap218
					d144 = snap219
					d145 = snap220
					d146 = snap221
					d147 = snap222
					d148 = snap223
					d149 = snap224
					d150 = snap225
					d152 = snap226
					d153 = snap227
					d155 = snap228
					d156 = snap229
					d157 = snap230
					d159 = snap231
					d161 = snap232
					d162 = snap233
					d165 = snap234
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps163)
					}
					return result
					ctx.FreeDesc(&d156)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps236 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps236)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(48))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  97,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "connection_id",

		Fn: func(a ...Scmer) Scmer {
			if ss := GetCurrentSessionState(); ss != nil {
				return NewInt(int64(ss.ID))
			}
			return NewInt(0)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the process-list ID of the current session (MySQL CONNECTION_ID() equivalent)",
			Return: &TypeDescriptor{Kind: "int"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["connection_id"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
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
					d0 = ctx.EmitGoCallScalar(GoFuncAddr(GetCurrentSessionState), []JITValueDesc{}, 1)
					ctx.BindReg(d0.Reg, &d0)
					ctx.StabilizeDescForControlFlow(&d0)
					ctx.EnsureDesc(&d0)
					var d1 JITValueDesc
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d0)
						if d0.Loc != LocReg && d0.Loc != LocRegPair && d0.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d0.Reg)
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d1)
					}
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
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
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
					var d11 JITValueDesc
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						fieldAddr := uintptr(d0.Imm.Int()) + 0
						r1 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r1, fieldAddr)
						d11 = JITValueDesc{Loc: LocReg, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						off := int32(0)
						baseReg := d0.Reg
						r2 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r2, baseReg, off)
						d11 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d11)
					}
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					var d12 JITValueDesc
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d11.Imm.Int()))))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegReg(r3, d11.Reg)
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d12)
					}
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						ctx.EmitMakeInt(result, d12)
					} else {
						ctx.EmitMovToReg(result.Reg2, d12)
						d13 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d13)
						if d12.Loc == LocReg && d12.Reg != result.Reg2 {
							ctx.FreeReg(d12.Reg)
						}
					}
					result.Type = tagInt
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d14.Loc == LocImm {
						ctx.EmitMakeInt(result, d14)
					} else {
						ctx.EmitMovToReg(result.Reg2, d14)
						d15 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d15)
						if d14.Loc == LocReg && d14.Reg != result.Reg2 {
							ctx.FreeReg(d14.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps16 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps16)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITInlineCost: 10,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "kill_query",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(KillSession(uint64(a[0].Int())))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "cancel the running query in session id; returns true if a query was killed",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "int", Label: "id", Description: "session ID from SHOW PROCESSLIST"}},
			Return: &TypeDescriptor{Kind: "bool"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["kill_query"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				var d1 JITValueDesc
				if d0.Loc == LocImm {
					d1 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int())}
				} else if d0.Type == tagInt && d0.Loc == LocRegPair {
					ctx.FreeReg(d0.Reg)
					d1 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d0.Reg2}
					ctx.BindReg(d0.Reg2, &d1)
					ctx.BindReg(d0.Reg2, &d1)
				} else if d0.Type == tagInt && d0.Loc == LocReg {
					d1 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d0.Reg}
					ctx.BindReg(d0.Reg, &d1)
					ctx.BindReg(d0.Reg, &d1)
				} else {
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d0}, 1)
					d1.Type = tagInt
					ctx.BindReg(d1.Reg, &d1)
				}
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				var d2 JITValueDesc
				if d1.Loc == LocImm {
					d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(int64(d1.Imm.Int()))))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitMovRegReg(r0, d1.Reg)
					d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0}
					ctx.BindReg(r0, &d2)
				}
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				d3 := d2
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				lbl0 := ctx.ReserveLabel()
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_1 := int32(-1)
				_ = bbpos_1_1
				bbpos_1_2 := int32(-1)
				_ = bbpos_1_2
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(func(value uint64) any { return value }), []JITValueDesc{d3}, 2)
				ctx.ReclaimUntrackedRegs()
				d5 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&processList)))), NoHeapPointer: true, Rooted: true}
				if d5.Loc == LocRegPair || d5.Loc == LocStackPair || d5.Loc == LocRegTriple || d5.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d4.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d4 = tmpPair
				} else if d4.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
					switch d4.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d4)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d4)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d4)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d4)
					d4 = tmpPair
				}
				if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value ((*sync.Map).Load arg1)")
				}
				ctx.SyncDesc(&d5)
				ctx.SyncDesc(&d4)
				callResults6 := JITEmitGoCallResults(ctx, GoFuncAddr((*sync.Map).Load), []JITValueDesc{d5, d4}, []uint8{2, 1}, []uint8{3, 0})
				d7 := callResults6[0]
				_ = d7
				d8 := callResults6[1]
				_ = d8
				ctx.ReclaimUntrackedRegs()
				ctx.StabilizeDescForControlFlow(&d7)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d9 := d8
				ctx.EnsureDesc(&d9)
				if d9.Loc != LocImm && d9.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl1 := ctx.ReserveLabel()
				lbl2 := ctx.ReserveLabel()
				lbl3 := ctx.ReserveLabel()
				lbl4 := ctx.ReserveLabel()
				if d9.Loc == LocImm {
					if d9.Imm.Bool() {
						ctx.MarkLabel(lbl3)
						ctx.EmitJmp(lbl1)
					} else {
						ctx.MarkLabel(lbl4)
						ctx.EmitJmp(lbl2)
					}
				} else {
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl3)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl3)
					ctx.EmitJmp(lbl1)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
				}
				ctx.FreeDesc(&d8)
				bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r1 := ctx.AllocReg()
				d10 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r1, d10)
				}
				ctx.EmitJmp(lbl0)
				bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d7)
				d11 := ctx.EmitGoCallScalar(GoFuncAddr(func(value any) *SessionState { return value.(*SessionState) }), []JITValueDesc{d7}, 1)
				ctx.FreeDesc(&d7)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				ctx.EnsureDesc(&d11)
				if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d11)
				d12 := ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).Kill), []JITValueDesc{d11}, 1)
				ctx.EmitAndRegImm32(d12.Reg, 1)
				d12.Type = tagBool
				ctx.BindReg(d12.Reg, &d12)
				ctx.FreeDesc(&d11)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d12)
				if d12.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r1, d12)
				}
				ctx.EmitJmp(lbl0)
				ctx.MarkLabel(lbl0)
				d13 := JITValueDesc{Loc: LocReg, Reg: r1}
				ctx.BindReg(r1, &d13)
				ctx.BindReg(r1, &d13)
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d13)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d13.Loc == LocImm {
					ctx.EmitMakeBool(result, d13)
				} else {
					ctx.EmitMakeBool(result, d13)
					ctx.FreeReg(d13.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 16,
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
	now := time.Now().UnixNano()
	s.startedAt.Store(now)
	s.lastUsed.Store(now)
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
