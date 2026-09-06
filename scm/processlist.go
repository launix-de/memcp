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
	return s.SetQueryInfoPointer(seq, &info)
}

// SetQueryInfoPointer publishes the transaction-owned SQL string without
// creating another copy for SHOW FULL PROCESSLIST.
func (s *SessionState) SetQueryInfoPointer(seq uint64, info *string) bool {
	if seq == 0 || s.activeQuery.Load() != seq {
		return false
	}
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	if !s.active[seq] || s.activeQuery.Load() != seq {
		return false
	}
	s.Info.Store(info)
	return true
}

// FinishQueryExecution drops cancellation and process text as soon as SQL
// execution has unwound. EndQuery later performs connection-level accounting.
func (s *SessionState) FinishQueryExecution(seq uint64) {
	s.ClearCancel(seq)
	if s.activeQuery.CompareAndSwap(seq, 0) {
		empty := ""
		s.Info.Store(&empty)
		s.SetState("")
	}
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
	EmitTracePrint(formatKillLog(s, action))
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

// HasLocks reports whether this connection currently owns user-level table
// locks. The planner uses it to avoid publishing session-independent cache
// recipes whose source scan would have to re-enter the owner's lock.
func (s *SessionState) HasLocks() bool {
	if s == nil {
		return false
	}
	s.heldLocksMu.Lock()
	hasLocks := len(s.heldLocks) != 0
	s.heldLocksMu.Unlock()
	return hasLocks
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

func querySessionState(value Scmer) (*SessionState, uint64) {
	if value.IsNil() {
		return nil, 0
	}
	state, ok := value.Any().(interface {
		QuerySessionState() (*SessionState, uint64)
	})
	if !ok {
		return nil, 0
	}
	return state.QuerySessionState()
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
				declaration := declarations["show_processlist"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d11 JITValueDesc
				_ = d11
				var d14 JITValueDesc
				_ = d14
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d44 JITValueDesc
				_ = d44
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
				var d78 JITValueDesc
				_ = d78
				var d80 JITValueDesc
				_ = d80
				var d83 JITValueDesc
				_ = d83
				var d117 JITValueDesc
				_ = d117
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
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var stackArray129 int32
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
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
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d154 JITValueDesc
				_ = d154
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
				var bbs [9]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				bbs[3].PhiCount = uint16(1)
				bbs[7].PhiBase = int32(phiBase0) + int32(32)
				bbs[7].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				d3 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d3
				var d4 JITValueDesc
				if phiHomeOK2 {
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				}
				_ = d4
				d5 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(32))
				_ = d5
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
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d6)
					var d7 JITValueDesc
					if d6.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() > 0)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d6.Reg, 0)
						ctx.EmitSetcc(r1, CondSignedGreater)
						d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d7)
					}
					ctx.FreeDesc(&d6)
					d8 = d7
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocImm && d8.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d8.Loc == LocImm {
						if d8.Imm.Bool() {
							if ps.General {
							}
							ps9 := PhiState{General: ps.General}
							ps9.OverlayValues = make([]JITValueDesc, 9)
							ps9.OverlayValues[3] = d3
							ps9.OverlayValues[4] = d4
							ps9.OverlayValues[5] = d5
							ps9.OverlayValues[6] = d6
							ps9.OverlayValues[7] = d7
							ps9.OverlayValues[8] = d8
							return bbs[1].RenderPS(ps9)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps10 := PhiState{General: ps.General}
						ps10.OverlayValues = make([]JITValueDesc, 9)
						ps10.OverlayValues[3] = d3
						ps10.OverlayValues[4] = d4
						ps10.OverlayValues[5] = d5
						ps10.OverlayValues[6] = d6
						ps10.OverlayValues[7] = d7
						ps10.OverlayValues[8] = d8
						ps10.PhiValues = make([]JITValueDesc, 1)
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps10.PhiValues[0] = d11
						return bbs[2].RenderPS(ps10)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d8.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl11)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 12)
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[11] = d11
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 12)
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[5] = d5
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[11] = d11
					ps13.PhiValues = make([]JITValueDesc, 1)
					d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps13.PhiValues[0] = d14
					snap15 := d3
					snap16 := d4
					snap17 := d5
					snap18 := d6
					snap19 := d7
					snap20 := d8
					snap21 := d11
					snap22 := d14
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc23)
					d3 = snap15
					d4 = snap16
					d5 = snap17
					d6 = snap18
					d7 = snap19
					d8 = snap20
					d11 = snap21
					d14 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d7)
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					ctx.ReclaimUntrackedRegs()
					d24 = args[0]
					d24.ID = 0
					d26 = d24
					d26.ID = 0
					d25 = ctx.EmitBoolDesc(&d26, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d25)
					ctx.FreeDesc(&d24)
					if ps.General {
						ctx.SyncDesc(&d25)
						if d25.Loc == LocReg {
							ctx.ProtectReg(d25.Reg)
						} else if d25.Loc == LocRegPair {
							ctx.ProtectReg(d25.Reg)
							ctx.ProtectReg(d25.Reg2)
						}
						d27 = d25
						if d27.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d27)
						ctx.EmitStoreToStack(d27, int32(bbs[2].PhiBase)+int32(0))
						if d25.Loc == LocReg {
							ctx.UnprotectReg(d25.Reg)
						} else if d25.Loc == LocRegPair {
							ctx.UnprotectReg(d25.Reg)
							ctx.UnprotectReg(d25.Reg2)
						}
					}
					ps28 := PhiState{General: ps.General}
					ps28.OverlayValues = make([]JITValueDesc, 28)
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[6] = d6
					ps28.OverlayValues[7] = d7
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[11] = d11
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[24] = d24
					ps28.OverlayValues[25] = d25
					ps28.OverlayValues[26] = d26
					ps28.OverlayValues[27] = d27
					ps28.PhiValues = make([]JITValueDesc, 1)
					d29 = d25
					ps28.PhiValues[0] = d29
					if ps28.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps28)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d30 := ps.PhiValues[0]
							ctx.EnsureDesc(&d30)
							ctx.EmitStoreToStack(d30, int32(bbs[2].PhiBase)+int32(0))
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					d31 = ctx.EmitGoCallScalar(GoFuncAddr(Snapshot), []JITValueDesc{}, 3)
					d31.NoHeapPointer = false
					ctx.BindReg(d31.Reg, &d31)
					ctx.BindReg(d31.Reg2, &d31)
					ctx.BindReg(d31.Reg3, &d31)
					ctx.StabilizeDescForControlFlow(&d31)
					var d32 JITValueDesc
					if d31.SliceSizeKnown {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
					} else if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
					} else if d31.Loc == LocStackTriple {
						d32 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d31)
						if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
							d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
						} else if d31.Loc == LocReg {
							d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					callResults33 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d32, d32}, []uint8{3}, []uint8{1})
					d34 = callResults33[0]
					d34.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.FreeDesc(&d32)
					var d35 JITValueDesc
					if d31.SliceSizeKnown {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
					} else if d31.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
					} else if d31.Loc == LocStackTriple {
						d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d31)
						if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
						} else if d31.Loc == LocReg {
							d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d35)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
						}
					}
					ps36 := PhiState{General: ps.General}
					ps36.OverlayValues = make([]JITValueDesc, 36)
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[4] = d4
					ps36.OverlayValues[5] = d5
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[7] = d7
					ps36.OverlayValues[8] = d8
					ps36.OverlayValues[11] = d11
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[24] = d24
					ps36.OverlayValues[25] = d25
					ps36.OverlayValues[26] = d26
					ps36.OverlayValues[27] = d27
					ps36.OverlayValues[29] = d29
					ps36.OverlayValues[30] = d30
					ps36.OverlayValues[31] = d31
					ps36.OverlayValues[32] = d32
					ps36.OverlayValues[34] = d34
					ps36.OverlayValues[35] = d35
					ps36.PhiValues = make([]JITValueDesc, 1)
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps36.PhiValues[0] = d37
					if ps36.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps36)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d38 := ps.PhiValues[0]
							ctx.EnsureDesc(&d38)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d38)
							} else {
								ctx.EmitStoreToStack(d38, int32(bbs[3].PhiBase)+int32(0))
							}
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d4 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d4.Loc == LocReg {
						ctx.BindReg(r0, &d4)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					var d39 JITValueDesc
					if d4.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d39)
					}
					if d39.Loc == LocReg && d4.Loc == LocReg && d39.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d39)
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d35)
					ctx.EnsureDescsTogether(&d39, &d35)
					var d40 JITValueDesc
					if d39.Loc == LocImm && d35.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d39.Imm.Int() < d35.Imm.Int())}
					} else if d35.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d39.Reg)
						if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d39.Reg, int32(d35.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d35.Imm.Int()))
							ctx.EmitCmpInt64(d39.Reg, RegR11)
						}
						ctx.EmitSetcc(r2, CondSignedLess)
						d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d40)
					} else if d39.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d35.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d40)
					} else {
						r4 := ctx.AllocRegExcept(d39.Reg)
						ctx.EmitCmpInt64(d39.Reg, d35.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d40)
					}
					ctx.FreeDesc(&d35)
					d41 = d40
					ctx.EnsureDesc(&d41)
					if d41.Loc != LocImm && d41.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d41.Loc == LocImm {
						if d41.Imm.Bool() {
							if ps.General {
							}
							ps42 := PhiState{General: ps.General}
							ps42.OverlayValues = make([]JITValueDesc, 42)
							ps42.OverlayValues[3] = d3
							ps42.OverlayValues[4] = d4
							ps42.OverlayValues[5] = d5
							ps42.OverlayValues[6] = d6
							ps42.OverlayValues[7] = d7
							ps42.OverlayValues[8] = d8
							ps42.OverlayValues[11] = d11
							ps42.OverlayValues[14] = d14
							ps42.OverlayValues[24] = d24
							ps42.OverlayValues[25] = d25
							ps42.OverlayValues[26] = d26
							ps42.OverlayValues[27] = d27
							ps42.OverlayValues[29] = d29
							ps42.OverlayValues[30] = d30
							ps42.OverlayValues[31] = d31
							ps42.OverlayValues[32] = d32
							ps42.OverlayValues[34] = d34
							ps42.OverlayValues[35] = d35
							ps42.OverlayValues[37] = d37
							ps42.OverlayValues[38] = d38
							ps42.OverlayValues[39] = d39
							ps42.OverlayValues[40] = d40
							ps42.OverlayValues[41] = d41
							return bbs[4].RenderPS(ps42)
						}
						if ps.General {
						}
						ps43 := PhiState{General: ps.General}
						ps43.OverlayValues = make([]JITValueDesc, 42)
						ps43.OverlayValues[3] = d3
						ps43.OverlayValues[4] = d4
						ps43.OverlayValues[5] = d5
						ps43.OverlayValues[6] = d6
						ps43.OverlayValues[7] = d7
						ps43.OverlayValues[8] = d8
						ps43.OverlayValues[11] = d11
						ps43.OverlayValues[14] = d14
						ps43.OverlayValues[24] = d24
						ps43.OverlayValues[25] = d25
						ps43.OverlayValues[26] = d26
						ps43.OverlayValues[27] = d27
						ps43.OverlayValues[29] = d29
						ps43.OverlayValues[30] = d30
						ps43.OverlayValues[31] = d31
						ps43.OverlayValues[32] = d32
						ps43.OverlayValues[34] = d34
						ps43.OverlayValues[35] = d35
						ps43.OverlayValues[37] = d37
						ps43.OverlayValues[38] = d38
						ps43.OverlayValues[39] = d39
						ps43.OverlayValues[40] = d40
						ps43.OverlayValues[41] = d41
						return bbs[5].RenderPS(ps43)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d44 := ps.PhiValues[0]
							ctx.EnsureDesc(&d44)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d44)
							} else {
								ctx.EmitStoreToStack(d44, int32(bbs[3].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d41.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl6)
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 45)
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[6] = d6
					ps45.OverlayValues[7] = d7
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[24] = d24
					ps45.OverlayValues[25] = d25
					ps45.OverlayValues[26] = d26
					ps45.OverlayValues[27] = d27
					ps45.OverlayValues[29] = d29
					ps45.OverlayValues[30] = d30
					ps45.OverlayValues[31] = d31
					ps45.OverlayValues[32] = d32
					ps45.OverlayValues[34] = d34
					ps45.OverlayValues[35] = d35
					ps45.OverlayValues[37] = d37
					ps45.OverlayValues[38] = d38
					ps45.OverlayValues[39] = d39
					ps45.OverlayValues[40] = d40
					ps45.OverlayValues[41] = d41
					ps45.OverlayValues[44] = d44
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 45)
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[6] = d6
					ps46.OverlayValues[7] = d7
					ps46.OverlayValues[8] = d8
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[24] = d24
					ps46.OverlayValues[25] = d25
					ps46.OverlayValues[26] = d26
					ps46.OverlayValues[27] = d27
					ps46.OverlayValues[29] = d29
					ps46.OverlayValues[30] = d30
					ps46.OverlayValues[31] = d31
					ps46.OverlayValues[32] = d32
					ps46.OverlayValues[34] = d34
					ps46.OverlayValues[35] = d35
					ps46.OverlayValues[37] = d37
					ps46.OverlayValues[38] = d38
					ps46.OverlayValues[39] = d39
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[44] = d44
					snap47 := d3
					snap48 := d4
					snap49 := d5
					snap50 := d6
					snap51 := d7
					snap52 := d8
					snap53 := d11
					snap54 := d14
					snap55 := d24
					snap56 := d25
					snap57 := d26
					snap58 := d27
					snap59 := d29
					snap60 := d30
					snap61 := d31
					snap62 := d32
					snap63 := d34
					snap64 := d35
					snap65 := d37
					snap66 := d38
					snap67 := d39
					snap68 := d40
					snap69 := d41
					snap70 := d44
					alloc71 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps46)
					}
					ctx.RestoreAllocState(alloc71)
					d3 = snap47
					d4 = snap48
					d5 = snap49
					d6 = snap50
					d7 = snap51
					d8 = snap52
					d11 = snap53
					d14 = snap54
					d24 = snap55
					d25 = snap56
					d26 = snap57
					d27 = snap58
					d29 = snap59
					d30 = snap60
					d31 = snap61
					d32 = snap62
					d34 = snap63
					d35 = snap64
					d37 = snap65
					d38 = snap66
					d39 = snap67
					d40 = snap68
					d41 = snap69
					d44 = snap70
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps45)
					}
					return result
					ctx.FreeDesc(&d40)
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d39)
					d73 = ctx.EmitSliceElementAddress(&d31, &d39, 8)
					ctx.EnsureDesc(&d73)
					ctx.EmitMovRegMem(d73.Reg, d73.Reg, 0)
					d72 = d73
					ctx.StabilizeDescForControlFlow(&d72)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d74 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d72}, 2)
					d74.NoHeapPointer = false
					ctx.BindReg(d74.Reg, &d74)
					ctx.BindReg(d74.Reg2, &d74)
					ctx.StabilizeDescForControlFlow(&d74)
					d75 = d3
					ctx.EnsureDesc(&d75)
					if d75.Loc != LocImm && d75.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d75.Loc == LocImm {
						if d75.Imm.Bool() {
							if ps.General {
								ctx.SyncDesc(&d74)
								if d74.Loc == LocReg {
									ctx.ProtectReg(d74.Reg)
								} else if d74.Loc == LocRegPair {
									ctx.ProtectReg(d74.Reg)
									ctx.ProtectReg(d74.Reg2)
								}
								d76 = d74
								if d76.Loc == LocNone {
									panic("jit: phi source has no location")
								}
								ctx.SyncDesc(&d76)
								ctx.EmitStoreScmerToStack(d76, int32(bbs[7].PhiBase)+int32(0))
								if d74.Loc == LocReg {
									ctx.UnprotectReg(d74.Reg)
								} else if d74.Loc == LocRegPair {
									ctx.UnprotectReg(d74.Reg)
									ctx.UnprotectReg(d74.Reg2)
								}
							}
							ps77 := PhiState{General: ps.General}
							ps77.OverlayValues = make([]JITValueDesc, 77)
							ps77.OverlayValues[3] = d3
							ps77.OverlayValues[4] = d4
							ps77.OverlayValues[5] = d5
							ps77.OverlayValues[6] = d6
							ps77.OverlayValues[7] = d7
							ps77.OverlayValues[8] = d8
							ps77.OverlayValues[11] = d11
							ps77.OverlayValues[14] = d14
							ps77.OverlayValues[24] = d24
							ps77.OverlayValues[25] = d25
							ps77.OverlayValues[26] = d26
							ps77.OverlayValues[27] = d27
							ps77.OverlayValues[29] = d29
							ps77.OverlayValues[30] = d30
							ps77.OverlayValues[31] = d31
							ps77.OverlayValues[32] = d32
							ps77.OverlayValues[34] = d34
							ps77.OverlayValues[35] = d35
							ps77.OverlayValues[37] = d37
							ps77.OverlayValues[38] = d38
							ps77.OverlayValues[39] = d39
							ps77.OverlayValues[40] = d40
							ps77.OverlayValues[41] = d41
							ps77.OverlayValues[44] = d44
							ps77.OverlayValues[72] = d72
							ps77.OverlayValues[73] = d73
							ps77.OverlayValues[74] = d74
							ps77.OverlayValues[75] = d75
							ps77.OverlayValues[76] = d76
							ps77.PhiValues = make([]JITValueDesc, 1)
							d78 = d74
							ps77.PhiValues[0] = d78
							return bbs[7].RenderPS(ps77)
						}
						if ps.General {
						}
						ps79 := PhiState{General: ps.General}
						ps79.OverlayValues = make([]JITValueDesc, 79)
						ps79.OverlayValues[3] = d3
						ps79.OverlayValues[4] = d4
						ps79.OverlayValues[5] = d5
						ps79.OverlayValues[6] = d6
						ps79.OverlayValues[7] = d7
						ps79.OverlayValues[8] = d8
						ps79.OverlayValues[11] = d11
						ps79.OverlayValues[14] = d14
						ps79.OverlayValues[24] = d24
						ps79.OverlayValues[25] = d25
						ps79.OverlayValues[26] = d26
						ps79.OverlayValues[27] = d27
						ps79.OverlayValues[29] = d29
						ps79.OverlayValues[30] = d30
						ps79.OverlayValues[31] = d31
						ps79.OverlayValues[32] = d32
						ps79.OverlayValues[34] = d34
						ps79.OverlayValues[35] = d35
						ps79.OverlayValues[37] = d37
						ps79.OverlayValues[38] = d38
						ps79.OverlayValues[39] = d39
						ps79.OverlayValues[40] = d40
						ps79.OverlayValues[41] = d41
						ps79.OverlayValues[44] = d44
						ps79.OverlayValues[72] = d72
						ps79.OverlayValues[73] = d73
						ps79.OverlayValues[74] = d74
						ps79.OverlayValues[75] = d75
						ps79.OverlayValues[76] = d76
						ps79.OverlayValues[78] = d78
						return bbs[8].RenderPS(ps79)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d75.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.SyncDesc(&d74)
					if d74.Loc == LocReg {
						ctx.ProtectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.ProtectReg(d74.Reg)
						ctx.ProtectReg(d74.Reg2)
					}
					d80 = d74
					if d80.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d80)
					ctx.EmitStoreScmerToStack(d80, int32(bbs[7].PhiBase)+int32(0))
					if d74.Loc == LocReg {
						ctx.UnprotectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.UnprotectReg(d74.Reg)
						ctx.UnprotectReg(d74.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl9)
					ps81 := PhiState{General: true}
					ps81.OverlayValues = make([]JITValueDesc, 81)
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[8] = d8
					ps81.OverlayValues[11] = d11
					ps81.OverlayValues[14] = d14
					ps81.OverlayValues[24] = d24
					ps81.OverlayValues[25] = d25
					ps81.OverlayValues[26] = d26
					ps81.OverlayValues[27] = d27
					ps81.OverlayValues[29] = d29
					ps81.OverlayValues[30] = d30
					ps81.OverlayValues[31] = d31
					ps81.OverlayValues[32] = d32
					ps81.OverlayValues[34] = d34
					ps81.OverlayValues[35] = d35
					ps81.OverlayValues[37] = d37
					ps81.OverlayValues[38] = d38
					ps81.OverlayValues[39] = d39
					ps81.OverlayValues[40] = d40
					ps81.OverlayValues[41] = d41
					ps81.OverlayValues[44] = d44
					ps81.OverlayValues[72] = d72
					ps81.OverlayValues[73] = d73
					ps81.OverlayValues[74] = d74
					ps81.OverlayValues[75] = d75
					ps81.OverlayValues[76] = d76
					ps81.OverlayValues[78] = d78
					ps81.OverlayValues[80] = d80
					ps81.PhiValues = make([]JITValueDesc, 1)
					d83 = d74
					ps81.PhiValues[0] = d83
					ps82 := PhiState{General: true}
					ps82.OverlayValues = make([]JITValueDesc, 84)
					ps82.OverlayValues[3] = d3
					ps82.OverlayValues[4] = d4
					ps82.OverlayValues[5] = d5
					ps82.OverlayValues[6] = d6
					ps82.OverlayValues[7] = d7
					ps82.OverlayValues[8] = d8
					ps82.OverlayValues[11] = d11
					ps82.OverlayValues[14] = d14
					ps82.OverlayValues[24] = d24
					ps82.OverlayValues[25] = d25
					ps82.OverlayValues[26] = d26
					ps82.OverlayValues[27] = d27
					ps82.OverlayValues[29] = d29
					ps82.OverlayValues[30] = d30
					ps82.OverlayValues[31] = d31
					ps82.OverlayValues[32] = d32
					ps82.OverlayValues[34] = d34
					ps82.OverlayValues[35] = d35
					ps82.OverlayValues[37] = d37
					ps82.OverlayValues[38] = d38
					ps82.OverlayValues[39] = d39
					ps82.OverlayValues[40] = d40
					ps82.OverlayValues[41] = d41
					ps82.OverlayValues[44] = d44
					ps82.OverlayValues[72] = d72
					ps82.OverlayValues[73] = d73
					ps82.OverlayValues[74] = d74
					ps82.OverlayValues[75] = d75
					ps82.OverlayValues[76] = d76
					ps82.OverlayValues[78] = d78
					ps82.OverlayValues[80] = d80
					ps82.OverlayValues[83] = d83
					snap84 := d3
					snap85 := d4
					snap86 := d5
					snap87 := d6
					snap88 := d7
					snap89 := d8
					snap90 := d11
					snap91 := d14
					snap92 := d24
					snap93 := d25
					snap94 := d26
					snap95 := d27
					snap96 := d29
					snap97 := d30
					snap98 := d31
					snap99 := d32
					snap100 := d34
					snap101 := d35
					snap102 := d37
					snap103 := d38
					snap104 := d39
					snap105 := d40
					snap106 := d41
					snap107 := d44
					snap108 := d72
					snap109 := d73
					snap110 := d74
					snap111 := d75
					snap112 := d76
					snap113 := d78
					snap114 := d80
					snap115 := d83
					alloc116 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps81)
					}
					ctx.RestoreAllocState(alloc116)
					d3 = snap84
					d4 = snap85
					d5 = snap86
					d6 = snap87
					d7 = snap88
					d8 = snap89
					d11 = snap90
					d14 = snap91
					d24 = snap92
					d25 = snap93
					d26 = snap94
					d27 = snap95
					d29 = snap96
					d30 = snap97
					d31 = snap98
					d32 = snap99
					d34 = snap100
					d35 = snap101
					d37 = snap102
					d38 = snap103
					d39 = snap104
					d40 = snap105
					d41 = snap106
					d44 = snap107
					d72 = snap108
					d73 = snap109
					d74 = snap110
					d75 = snap111
					d76 = snap112
					d78 = snap113
					d80 = snap114
					d83 = snap115
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps82)
					}
					return result
					ctx.FreeDesc(&d3)
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d34)
					d117 = ctx.EmitNewSliceFromGoSlice(&d34)
					ctx.SyncDesc(&d117)
					if d117.Loc == LocRegPair || d117.Loc == LocStackPair || d117.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d117, &result)
						result.Type = d117.Type
					} else {
						switch d117.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d117)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d117)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d117)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d117, &result)
							result.Type = d117.Type
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					ctx.ReclaimUntrackedRegs()
					d118 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d119 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d118)
					ctx.EnsureDesc(&d119)
					var d121 JITValueDesc
					if d119.Loc == LocImm && d118.Loc == LocImm {
						d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d119.Imm.Int() - d118.Imm.Int())}
					} else {
						r5 := ctx.AllocReg()
						if d119.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d119.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d119.Reg)
						}
						if d118.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d118.Imm.Int()))
							ctx.EmitSubInt64(r5, RegR11)
						} else {
							ctx.EmitSubInt64(r5, d118.Reg)
						}
						d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d121)
					}
					var d122 JITValueDesc
					r6 := ctx.EmitSliceDataAfterLow(&d74, &d118, 1)
					d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
					ctx.BindReg(r6, &d122)
					ctx.BindReg(r6, &d122)
					var d123 JITValueDesc
					var r7 Reg
					var r8 Reg
					ctx.SyncDesc(&d122)
					ctx.EnsureDesc(&d122)
					if d122.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d122.Imm.Int()))
					} else {
						r7 = d122.Reg
					}
					ctx.ProtectReg(r7)
					ctx.SyncDesc(&d121)
					ctx.EnsureDesc(&d121)
					if d121.Loc == LocImm {
						r8 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r8, uint64(d121.Imm.Int()))
					} else {
						r8 = d121.Reg
					}
					ctx.ProtectReg(r8)
					ctx.UnprotectReg(r8)
					ctx.UnprotectReg(r7)
					d123 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d123)
					ctx.BindReg(r8, &d123)
					ctx.BindReg(r7, &d123)
					ctx.BindReg(r8, &d123)
					ctx.StabilizeDescForControlFlow(&d123)
					if ps.General {
						ctx.SyncDesc(&d123)
						if d123.Loc == LocReg {
							ctx.ProtectReg(d123.Reg)
						} else if d123.Loc == LocRegPair {
							ctx.ProtectReg(d123.Reg)
							ctx.ProtectReg(d123.Reg2)
						}
						d124 = d123
						if d124.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d124)
						ctx.EmitStoreScmerToStack(d124, int32(bbs[7].PhiBase)+int32(0))
						if d123.Loc == LocReg {
							ctx.UnprotectReg(d123.Reg)
						} else if d123.Loc == LocRegPair {
							ctx.UnprotectReg(d123.Reg)
							ctx.UnprotectReg(d123.Reg2)
						}
					}
					ps125 := PhiState{General: ps.General}
					ps125.OverlayValues = make([]JITValueDesc, 125)
					ps125.OverlayValues[3] = d3
					ps125.OverlayValues[4] = d4
					ps125.OverlayValues[5] = d5
					ps125.OverlayValues[6] = d6
					ps125.OverlayValues[7] = d7
					ps125.OverlayValues[8] = d8
					ps125.OverlayValues[11] = d11
					ps125.OverlayValues[14] = d14
					ps125.OverlayValues[24] = d24
					ps125.OverlayValues[25] = d25
					ps125.OverlayValues[26] = d26
					ps125.OverlayValues[27] = d27
					ps125.OverlayValues[29] = d29
					ps125.OverlayValues[30] = d30
					ps125.OverlayValues[31] = d31
					ps125.OverlayValues[32] = d32
					ps125.OverlayValues[34] = d34
					ps125.OverlayValues[35] = d35
					ps125.OverlayValues[37] = d37
					ps125.OverlayValues[38] = d38
					ps125.OverlayValues[39] = d39
					ps125.OverlayValues[40] = d40
					ps125.OverlayValues[41] = d41
					ps125.OverlayValues[44] = d44
					ps125.OverlayValues[72] = d72
					ps125.OverlayValues[73] = d73
					ps125.OverlayValues[74] = d74
					ps125.OverlayValues[75] = d75
					ps125.OverlayValues[76] = d76
					ps125.OverlayValues[78] = d78
					ps125.OverlayValues[80] = d80
					ps125.OverlayValues[83] = d83
					ps125.OverlayValues[117] = d117
					ps125.OverlayValues[118] = d118
					ps125.OverlayValues[119] = d119
					ps125.OverlayValues[120] = d120
					ps125.OverlayValues[121] = d121
					ps125.OverlayValues[122] = d122
					ps125.OverlayValues[123] = d123
					ps125.OverlayValues[124] = d124
					ps125.PhiValues = make([]JITValueDesc, 1)
					d126 = d123
					ps125.PhiValues[0] = d126
					if ps125.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps125)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d127 := ps.PhiValues[0]
							ctx.EnsureDesc(&d127)
							ctx.EmitStoreScmerToStack(d127, int32(bbs[7].PhiBase)+int32(0))
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
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
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d5 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d34)
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d72)
					d128 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d72}, 2)
					d128.NoHeapPointer = false
					ctx.BindReg(d128.Reg, &d128)
					ctx.BindReg(d128.Reg2, &d128)
					stackArray129 = ctx.AllocStack(int32(256))
					_ = stackArray129
					d130 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.SyncDesc(&d130)
					ctx.EmitStoreScmerToStack(d130, int32(stackArray129)+int32(0))
					var d131 JITValueDesc
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						fieldAddr := uintptr(d72.Imm.Int()) + 0
						r9 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r9, fieldAddr)
						d131 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d131)
					} else {
						off := int32(0)
						baseReg := d72.Reg
						r10 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r10, baseReg, off)
						d131 = JITValueDesc{Loc: LocReg, Reg: r10}
						ctx.BindReg(r10, &d131)
					}
					ctx.EnsureDesc(&d131)
					ctx.EnsureDesc(&d131)
					var d132 JITValueDesc
					if d131.Loc == LocImm {
						d132 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d131.Imm.Int()))))}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegReg(r11, d131.Reg)
						d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d132)
					}
					ctx.FreeDesc(&d131)
					ctx.EnsureDesc(&d132)
					ctx.SyncDesc(&d132)
					ctx.EnsureDesc(&d132)
					ctx.EmitStoreTypedScmerToStack(d132, tagInt, int32(stackArray129)+int32(16))
					d133 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.SyncDesc(&d133)
					ctx.EmitStoreScmerToStack(d133, int32(stackArray129)+int32(32))
					var d134 JITValueDesc
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						fieldAddr := uintptr(d72.Imm.Int()) + 8
						r12 := ctx.AllocReg()
						r13 := ctx.AllocRegExcept(r12)
						r14 := ctx.AllocRegExcept(r12, r13)
						ctx.EmitMovRegMem64(r12, fieldAddr)
						ctx.EmitMovRegMem64(r13, fieldAddr+8)
						ctx.EmitMovRegMem64(r14, fieldAddr+16)
						d134 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
						ctx.BindReg(r12, &d134)
						ctx.BindReg(r13, &d134)
						ctx.BindReg(r14, &d134)
					} else {
						off := int32(8)
						baseReg := d72.Reg
						r15 := ctx.AllocRegExcept(baseReg)
						r16 := ctx.AllocRegExcept(baseReg, r15)
						r17 := ctx.AllocRegExcept(baseReg, r15, r16)
						ctx.EmitMovRegMem(r15, baseReg, off)
						ctx.EmitMovRegMem(r16, baseReg, off+8)
						ctx.EmitMovRegMem(r17, baseReg, off+16)
						d134 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d134)
						ctx.BindReg(r16, &d134)
						ctx.BindReg(r17, &d134)
					}
					ctx.EnsureDesc(&d134)
					ctx.SyncDesc(&d134)
					ctx.EmitStoreScmerToStack(d134, int32(stackArray129)+int32(48))
					d135 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.SyncDesc(&d135)
					ctx.EmitStoreScmerToStack(d135, int32(stackArray129)+int32(64))
					var d136 JITValueDesc
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						fieldAddr := uintptr(d72.Imm.Int()) + 24
						r18 := ctx.AllocReg()
						r19 := ctx.AllocRegExcept(r18)
						r20 := ctx.AllocRegExcept(r18, r19)
						ctx.EmitMovRegMem64(r18, fieldAddr)
						ctx.EmitMovRegMem64(r19, fieldAddr+8)
						ctx.EmitMovRegMem64(r20, fieldAddr+16)
						d136 = JITValueDesc{Loc: LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
						ctx.BindReg(r18, &d136)
						ctx.BindReg(r19, &d136)
						ctx.BindReg(r20, &d136)
					} else {
						off := int32(24)
						baseReg := d72.Reg
						r21 := ctx.AllocRegExcept(baseReg)
						r22 := ctx.AllocRegExcept(baseReg, r21)
						r23 := ctx.AllocRegExcept(baseReg, r21, r22)
						ctx.EmitMovRegMem(r21, baseReg, off)
						ctx.EmitMovRegMem(r22, baseReg, off+8)
						ctx.EmitMovRegMem(r23, baseReg, off+16)
						d136 = JITValueDesc{Loc: LocRegTriple, Reg: r21, Reg2: r22, Reg3: r23}
						ctx.BindReg(r21, &d136)
						ctx.BindReg(r22, &d136)
						ctx.BindReg(r23, &d136)
					}
					ctx.EnsureDesc(&d136)
					ctx.SyncDesc(&d136)
					ctx.EmitStoreScmerToStack(d136, int32(stackArray129)+int32(80))
					d137 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.SyncDesc(&d137)
					ctx.EmitStoreScmerToStack(d137, int32(stackArray129)+int32(96))
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d138 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d72}, 2)
					d138.NoHeapPointer = false
					ctx.BindReg(d138.Reg, &d138)
					ctx.BindReg(d138.Reg2, &d138)
					ctx.EnsureDesc(&d138)
					ctx.SyncDesc(&d138)
					ctx.EmitStoreScmerToStack(d138, int32(stackArray129)+int32(112))
					d139 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.SyncDesc(&d139)
					ctx.EmitStoreScmerToStack(d139, int32(stackArray129)+int32(128))
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d140 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d72}, 2)
					d140.NoHeapPointer = false
					ctx.BindReg(d140.Reg, &d140)
					ctx.BindReg(d140.Reg2, &d140)
					ctx.EnsureDesc(&d140)
					ctx.SyncDesc(&d140)
					ctx.EmitStoreScmerToStack(d140, int32(stackArray129)+int32(144))
					d141 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.SyncDesc(&d141)
					ctx.EmitStoreScmerToStack(d141, int32(stackArray129)+int32(160))
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocRegPair || d72.Loc == LocStackPair || d72.Loc == LocRegTriple || d72.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d72)
					d142 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d72}, 1)
					d142.NoHeapPointer = true
					ctx.BindReg(d142.Reg, &d142)
					ctx.EnsureDesc(&d142)
					ctx.SyncDesc(&d142)
					ctx.EnsureDesc(&d142)
					ctx.EmitStoreTypedScmerToStack(d142, tagInt, int32(stackArray129)+int32(176))
					d143 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.SyncDesc(&d143)
					ctx.EmitStoreScmerToStack(d143, int32(stackArray129)+int32(192))
					ctx.EnsureDesc(&d128)
					ctx.SyncDesc(&d128)
					ctx.EmitStoreScmerToStack(d128, int32(stackArray129)+int32(208))
					d144 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.SyncDesc(&d144)
					ctx.EmitStoreScmerToStack(d144, int32(stackArray129)+int32(224))
					ctx.EnsureDesc(&d5)
					ctx.SyncDesc(&d5)
					ctx.EmitStoreScmerToStack(d5, int32(stackArray129)+int32(240))
					d145 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d145
					r24 := ctx.AllocReg()
					r25 := ctx.AllocRegExcept(r24)
					r26 := ctx.AllocRegExcept(r24, r25)
					d146 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r24, Reg2: r25, Reg3: r26}
					ctx.BindReg(r24, &d146)
					ctx.BindReg(r25, &d146)
					ctx.BindReg(r26, &d146)
					ctx.BindReg(r24, &d146)
					ctx.BindReg(r25, &d146)
					ctx.BindReg(r26, &d146)
					ctx.EmitLeaRegMem(d146.Reg, ctx.StackReg, int32(stackArray129))
					ctx.EmitMovRegImm64(d146.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d146.Reg3, uint64(16))
					callResults147 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d146}, []uint8{2}, []uint8{1})
					d148 = callResults147[0]
					ctx.EnsureDesc(&d39)
					ctx.SyncDesc(&d148)
					ctx.StabilizeDescAcrossNestedCall(&d39)
					d149 = d34
					d149.ID = 0
					d150 = d39
					d150.ID = 0
					d151 = ctx.EmitSliceElementAddress(&d149, &d150, int32(16))
					ctx.FreeDesc(&d150)
					ctx.EmitStoreScmerAt(&d151, &d148)
					ctx.FreeDesc(&d151)
					ctx.FreeDesc(&d148)
					if ps.General {
						ctx.SyncDesc(&d39)
						if d39.Loc == LocReg {
							ctx.ProtectReg(d39.Reg)
						} else if d39.Loc == LocRegPair {
							ctx.ProtectReg(d39.Reg)
							ctx.ProtectReg(d39.Reg2)
						}
						d152 = d39
						if d152.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d152)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d152)
						} else {
							ctx.EmitStoreToStack(d152, int32(bbs[3].PhiBase)+int32(0))
						}
						if d39.Loc == LocReg {
							ctx.UnprotectReg(d39.Reg)
						} else if d39.Loc == LocRegPair {
							ctx.UnprotectReg(d39.Reg)
							ctx.UnprotectReg(d39.Reg2)
						}
					}
					ps153 := PhiState{General: ps.General}
					ps153.OverlayValues = make([]JITValueDesc, 153)
					ps153.OverlayValues[3] = d3
					ps153.OverlayValues[4] = d4
					ps153.OverlayValues[5] = d5
					ps153.OverlayValues[6] = d6
					ps153.OverlayValues[7] = d7
					ps153.OverlayValues[8] = d8
					ps153.OverlayValues[11] = d11
					ps153.OverlayValues[14] = d14
					ps153.OverlayValues[24] = d24
					ps153.OverlayValues[25] = d25
					ps153.OverlayValues[26] = d26
					ps153.OverlayValues[27] = d27
					ps153.OverlayValues[29] = d29
					ps153.OverlayValues[30] = d30
					ps153.OverlayValues[31] = d31
					ps153.OverlayValues[32] = d32
					ps153.OverlayValues[34] = d34
					ps153.OverlayValues[35] = d35
					ps153.OverlayValues[37] = d37
					ps153.OverlayValues[38] = d38
					ps153.OverlayValues[39] = d39
					ps153.OverlayValues[40] = d40
					ps153.OverlayValues[41] = d41
					ps153.OverlayValues[44] = d44
					ps153.OverlayValues[72] = d72
					ps153.OverlayValues[73] = d73
					ps153.OverlayValues[74] = d74
					ps153.OverlayValues[75] = d75
					ps153.OverlayValues[76] = d76
					ps153.OverlayValues[78] = d78
					ps153.OverlayValues[80] = d80
					ps153.OverlayValues[83] = d83
					ps153.OverlayValues[117] = d117
					ps153.OverlayValues[118] = d118
					ps153.OverlayValues[119] = d119
					ps153.OverlayValues[120] = d120
					ps153.OverlayValues[121] = d121
					ps153.OverlayValues[122] = d122
					ps153.OverlayValues[123] = d123
					ps153.OverlayValues[124] = d124
					ps153.OverlayValues[126] = d126
					ps153.OverlayValues[127] = d127
					ps153.OverlayValues[128] = d128
					ps153.OverlayValues[130] = d130
					ps153.OverlayValues[131] = d131
					ps153.OverlayValues[132] = d132
					ps153.OverlayValues[133] = d133
					ps153.OverlayValues[134] = d134
					ps153.OverlayValues[135] = d135
					ps153.OverlayValues[136] = d136
					ps153.OverlayValues[137] = d137
					ps153.OverlayValues[138] = d138
					ps153.OverlayValues[139] = d139
					ps153.OverlayValues[140] = d140
					ps153.OverlayValues[141] = d141
					ps153.OverlayValues[142] = d142
					ps153.OverlayValues[143] = d143
					ps153.OverlayValues[144] = d144
					ps153.OverlayValues[145] = d145
					ps153.OverlayValues[146] = d146
					ps153.OverlayValues[148] = d148
					ps153.OverlayValues[149] = d149
					ps153.OverlayValues[150] = d150
					ps153.OverlayValues[151] = d151
					ps153.OverlayValues[152] = d152
					ps153.PhiValues = make([]JITValueDesc, 1)
					d154 = d39
					ps153.PhiValues[0] = d154
					if ps153.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps153)
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
					d3 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					if phiHomeOK2 {
						d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					}
					d5 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
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
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
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
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
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
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
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
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					ctx.ReclaimUntrackedRegs()
					var d155 JITValueDesc
					if d74.SliceSizeKnown {
						d155 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d74.KnownSliceLen))}
					} else if d74.Loc == LocImm {
						d155 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d74.Imm.String())))}
					} else if d74.Loc == LocStackTriple {
						d155 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d74.StackOff + 8, NoHeapPointer: true}
					} else if d74.Loc == LocStackPair {
						d155 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d74.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d74)
						if d74.Loc == LocRegPair || d74.Loc == LocRegTriple {
							d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d74.Reg2, ID: 0}
						} else if d74.Loc == LocReg {
							d155 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d74.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d155)
					var d156 JITValueDesc
					if d155.Loc == LocImm {
						d156 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d155.Imm.Int() > 100)}
					} else {
						r27 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d155.Reg, 100)
						ctx.EmitSetcc(r27, CondSignedGreater)
						d156 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d156)
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
							ps158.OverlayValues[3] = d3
							ps158.OverlayValues[4] = d4
							ps158.OverlayValues[5] = d5
							ps158.OverlayValues[6] = d6
							ps158.OverlayValues[7] = d7
							ps158.OverlayValues[8] = d8
							ps158.OverlayValues[11] = d11
							ps158.OverlayValues[14] = d14
							ps158.OverlayValues[24] = d24
							ps158.OverlayValues[25] = d25
							ps158.OverlayValues[26] = d26
							ps158.OverlayValues[27] = d27
							ps158.OverlayValues[29] = d29
							ps158.OverlayValues[30] = d30
							ps158.OverlayValues[31] = d31
							ps158.OverlayValues[32] = d32
							ps158.OverlayValues[34] = d34
							ps158.OverlayValues[35] = d35
							ps158.OverlayValues[37] = d37
							ps158.OverlayValues[38] = d38
							ps158.OverlayValues[39] = d39
							ps158.OverlayValues[40] = d40
							ps158.OverlayValues[41] = d41
							ps158.OverlayValues[44] = d44
							ps158.OverlayValues[72] = d72
							ps158.OverlayValues[73] = d73
							ps158.OverlayValues[74] = d74
							ps158.OverlayValues[75] = d75
							ps158.OverlayValues[76] = d76
							ps158.OverlayValues[78] = d78
							ps158.OverlayValues[80] = d80
							ps158.OverlayValues[83] = d83
							ps158.OverlayValues[117] = d117
							ps158.OverlayValues[118] = d118
							ps158.OverlayValues[119] = d119
							ps158.OverlayValues[120] = d120
							ps158.OverlayValues[121] = d121
							ps158.OverlayValues[122] = d122
							ps158.OverlayValues[123] = d123
							ps158.OverlayValues[124] = d124
							ps158.OverlayValues[126] = d126
							ps158.OverlayValues[127] = d127
							ps158.OverlayValues[128] = d128
							ps158.OverlayValues[130] = d130
							ps158.OverlayValues[131] = d131
							ps158.OverlayValues[132] = d132
							ps158.OverlayValues[133] = d133
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
							ps158.OverlayValues[148] = d148
							ps158.OverlayValues[149] = d149
							ps158.OverlayValues[150] = d150
							ps158.OverlayValues[151] = d151
							ps158.OverlayValues[152] = d152
							ps158.OverlayValues[154] = d154
							ps158.OverlayValues[155] = d155
							ps158.OverlayValues[156] = d156
							ps158.OverlayValues[157] = d157
							return bbs[6].RenderPS(ps158)
						}
						if ps.General {
							ctx.SyncDesc(&d74)
							if d74.Loc == LocReg {
								ctx.ProtectReg(d74.Reg)
							} else if d74.Loc == LocRegPair {
								ctx.ProtectReg(d74.Reg)
								ctx.ProtectReg(d74.Reg2)
							}
							d159 = d74
							if d159.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d159)
							ctx.EmitStoreScmerToStack(d159, int32(bbs[7].PhiBase)+int32(0))
							if d74.Loc == LocReg {
								ctx.UnprotectReg(d74.Reg)
							} else if d74.Loc == LocRegPair {
								ctx.UnprotectReg(d74.Reg)
								ctx.UnprotectReg(d74.Reg2)
							}
						}
						ps160 := PhiState{General: ps.General}
						ps160.OverlayValues = make([]JITValueDesc, 160)
						ps160.OverlayValues[3] = d3
						ps160.OverlayValues[4] = d4
						ps160.OverlayValues[5] = d5
						ps160.OverlayValues[6] = d6
						ps160.OverlayValues[7] = d7
						ps160.OverlayValues[8] = d8
						ps160.OverlayValues[11] = d11
						ps160.OverlayValues[14] = d14
						ps160.OverlayValues[24] = d24
						ps160.OverlayValues[25] = d25
						ps160.OverlayValues[26] = d26
						ps160.OverlayValues[27] = d27
						ps160.OverlayValues[29] = d29
						ps160.OverlayValues[30] = d30
						ps160.OverlayValues[31] = d31
						ps160.OverlayValues[32] = d32
						ps160.OverlayValues[34] = d34
						ps160.OverlayValues[35] = d35
						ps160.OverlayValues[37] = d37
						ps160.OverlayValues[38] = d38
						ps160.OverlayValues[39] = d39
						ps160.OverlayValues[40] = d40
						ps160.OverlayValues[41] = d41
						ps160.OverlayValues[44] = d44
						ps160.OverlayValues[72] = d72
						ps160.OverlayValues[73] = d73
						ps160.OverlayValues[74] = d74
						ps160.OverlayValues[75] = d75
						ps160.OverlayValues[76] = d76
						ps160.OverlayValues[78] = d78
						ps160.OverlayValues[80] = d80
						ps160.OverlayValues[83] = d83
						ps160.OverlayValues[117] = d117
						ps160.OverlayValues[118] = d118
						ps160.OverlayValues[119] = d119
						ps160.OverlayValues[120] = d120
						ps160.OverlayValues[121] = d121
						ps160.OverlayValues[122] = d122
						ps160.OverlayValues[123] = d123
						ps160.OverlayValues[124] = d124
						ps160.OverlayValues[126] = d126
						ps160.OverlayValues[127] = d127
						ps160.OverlayValues[128] = d128
						ps160.OverlayValues[130] = d130
						ps160.OverlayValues[131] = d131
						ps160.OverlayValues[132] = d132
						ps160.OverlayValues[133] = d133
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
						ps160.OverlayValues[148] = d148
						ps160.OverlayValues[149] = d149
						ps160.OverlayValues[150] = d150
						ps160.OverlayValues[151] = d151
						ps160.OverlayValues[152] = d152
						ps160.OverlayValues[154] = d154
						ps160.OverlayValues[155] = d155
						ps160.OverlayValues[156] = d156
						ps160.OverlayValues[157] = d157
						ps160.OverlayValues[159] = d159
						ps160.PhiValues = make([]JITValueDesc, 1)
						d161 = d74
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
					ctx.SyncDesc(&d74)
					if d74.Loc == LocReg {
						ctx.ProtectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.ProtectReg(d74.Reg)
						ctx.ProtectReg(d74.Reg2)
					}
					d162 = d74
					if d162.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d162)
					ctx.EmitStoreScmerToStack(d162, int32(bbs[7].PhiBase)+int32(0))
					if d74.Loc == LocReg {
						ctx.UnprotectReg(d74.Reg)
					} else if d74.Loc == LocRegPair {
						ctx.UnprotectReg(d74.Reg)
						ctx.UnprotectReg(d74.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ps163 := PhiState{General: true}
					ps163.OverlayValues = make([]JITValueDesc, 163)
					ps163.OverlayValues[3] = d3
					ps163.OverlayValues[4] = d4
					ps163.OverlayValues[5] = d5
					ps163.OverlayValues[6] = d6
					ps163.OverlayValues[7] = d7
					ps163.OverlayValues[8] = d8
					ps163.OverlayValues[11] = d11
					ps163.OverlayValues[14] = d14
					ps163.OverlayValues[24] = d24
					ps163.OverlayValues[25] = d25
					ps163.OverlayValues[26] = d26
					ps163.OverlayValues[27] = d27
					ps163.OverlayValues[29] = d29
					ps163.OverlayValues[30] = d30
					ps163.OverlayValues[31] = d31
					ps163.OverlayValues[32] = d32
					ps163.OverlayValues[34] = d34
					ps163.OverlayValues[35] = d35
					ps163.OverlayValues[37] = d37
					ps163.OverlayValues[38] = d38
					ps163.OverlayValues[39] = d39
					ps163.OverlayValues[40] = d40
					ps163.OverlayValues[41] = d41
					ps163.OverlayValues[44] = d44
					ps163.OverlayValues[72] = d72
					ps163.OverlayValues[73] = d73
					ps163.OverlayValues[74] = d74
					ps163.OverlayValues[75] = d75
					ps163.OverlayValues[76] = d76
					ps163.OverlayValues[78] = d78
					ps163.OverlayValues[80] = d80
					ps163.OverlayValues[83] = d83
					ps163.OverlayValues[117] = d117
					ps163.OverlayValues[118] = d118
					ps163.OverlayValues[119] = d119
					ps163.OverlayValues[120] = d120
					ps163.OverlayValues[121] = d121
					ps163.OverlayValues[122] = d122
					ps163.OverlayValues[123] = d123
					ps163.OverlayValues[124] = d124
					ps163.OverlayValues[126] = d126
					ps163.OverlayValues[127] = d127
					ps163.OverlayValues[128] = d128
					ps163.OverlayValues[130] = d130
					ps163.OverlayValues[131] = d131
					ps163.OverlayValues[132] = d132
					ps163.OverlayValues[133] = d133
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
					ps163.OverlayValues[148] = d148
					ps163.OverlayValues[149] = d149
					ps163.OverlayValues[150] = d150
					ps163.OverlayValues[151] = d151
					ps163.OverlayValues[152] = d152
					ps163.OverlayValues[154] = d154
					ps163.OverlayValues[155] = d155
					ps163.OverlayValues[156] = d156
					ps163.OverlayValues[157] = d157
					ps163.OverlayValues[159] = d159
					ps163.OverlayValues[161] = d161
					ps163.OverlayValues[162] = d162
					ps164 := PhiState{General: true}
					ps164.OverlayValues = make([]JITValueDesc, 163)
					ps164.OverlayValues[3] = d3
					ps164.OverlayValues[4] = d4
					ps164.OverlayValues[5] = d5
					ps164.OverlayValues[6] = d6
					ps164.OverlayValues[7] = d7
					ps164.OverlayValues[8] = d8
					ps164.OverlayValues[11] = d11
					ps164.OverlayValues[14] = d14
					ps164.OverlayValues[24] = d24
					ps164.OverlayValues[25] = d25
					ps164.OverlayValues[26] = d26
					ps164.OverlayValues[27] = d27
					ps164.OverlayValues[29] = d29
					ps164.OverlayValues[30] = d30
					ps164.OverlayValues[31] = d31
					ps164.OverlayValues[32] = d32
					ps164.OverlayValues[34] = d34
					ps164.OverlayValues[35] = d35
					ps164.OverlayValues[37] = d37
					ps164.OverlayValues[38] = d38
					ps164.OverlayValues[39] = d39
					ps164.OverlayValues[40] = d40
					ps164.OverlayValues[41] = d41
					ps164.OverlayValues[44] = d44
					ps164.OverlayValues[72] = d72
					ps164.OverlayValues[73] = d73
					ps164.OverlayValues[74] = d74
					ps164.OverlayValues[75] = d75
					ps164.OverlayValues[76] = d76
					ps164.OverlayValues[78] = d78
					ps164.OverlayValues[80] = d80
					ps164.OverlayValues[83] = d83
					ps164.OverlayValues[117] = d117
					ps164.OverlayValues[118] = d118
					ps164.OverlayValues[119] = d119
					ps164.OverlayValues[120] = d120
					ps164.OverlayValues[121] = d121
					ps164.OverlayValues[122] = d122
					ps164.OverlayValues[123] = d123
					ps164.OverlayValues[124] = d124
					ps164.OverlayValues[126] = d126
					ps164.OverlayValues[127] = d127
					ps164.OverlayValues[128] = d128
					ps164.OverlayValues[130] = d130
					ps164.OverlayValues[131] = d131
					ps164.OverlayValues[132] = d132
					ps164.OverlayValues[133] = d133
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
					ps164.OverlayValues[148] = d148
					ps164.OverlayValues[149] = d149
					ps164.OverlayValues[150] = d150
					ps164.OverlayValues[151] = d151
					ps164.OverlayValues[152] = d152
					ps164.OverlayValues[154] = d154
					ps164.OverlayValues[155] = d155
					ps164.OverlayValues[156] = d156
					ps164.OverlayValues[157] = d157
					ps164.OverlayValues[159] = d159
					ps164.OverlayValues[161] = d161
					ps164.OverlayValues[162] = d162
					ps164.PhiValues = make([]JITValueDesc, 1)
					d165 = d74
					ps164.PhiValues[0] = d165
					snap166 := d3
					snap167 := d4
					snap168 := d5
					snap169 := d6
					snap170 := d7
					snap171 := d8
					snap172 := d11
					snap173 := d14
					snap174 := d24
					snap175 := d25
					snap176 := d26
					snap177 := d27
					snap178 := d29
					snap179 := d30
					snap180 := d31
					snap181 := d32
					snap182 := d34
					snap183 := d35
					snap184 := d37
					snap185 := d38
					snap186 := d39
					snap187 := d40
					snap188 := d41
					snap189 := d44
					snap190 := d72
					snap191 := d73
					snap192 := d74
					snap193 := d75
					snap194 := d76
					snap195 := d78
					snap196 := d80
					snap197 := d83
					snap198 := d117
					snap199 := d118
					snap200 := d119
					snap201 := d120
					snap202 := d121
					snap203 := d122
					snap204 := d123
					snap205 := d124
					snap206 := d126
					snap207 := d127
					snap208 := d128
					snap209 := d130
					snap210 := d131
					snap211 := d132
					snap212 := d133
					snap213 := d134
					snap214 := d135
					snap215 := d136
					snap216 := d137
					snap217 := d138
					snap218 := d139
					snap219 := d140
					snap220 := d141
					snap221 := d142
					snap222 := d143
					snap223 := d144
					snap224 := d145
					snap225 := d146
					snap226 := d148
					snap227 := d149
					snap228 := d150
					snap229 := d151
					snap230 := d152
					snap231 := d154
					snap232 := d155
					snap233 := d156
					snap234 := d157
					snap235 := d159
					snap236 := d161
					snap237 := d162
					snap238 := d165
					alloc239 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps164)
					}
					ctx.RestoreAllocState(alloc239)
					d3 = snap166
					d4 = snap167
					d5 = snap168
					d6 = snap169
					d7 = snap170
					d8 = snap171
					d11 = snap172
					d14 = snap173
					d24 = snap174
					d25 = snap175
					d26 = snap176
					d27 = snap177
					d29 = snap178
					d30 = snap179
					d31 = snap180
					d32 = snap181
					d34 = snap182
					d35 = snap183
					d37 = snap184
					d38 = snap185
					d39 = snap186
					d40 = snap187
					d41 = snap188
					d44 = snap189
					d72 = snap190
					d73 = snap191
					d74 = snap192
					d75 = snap193
					d76 = snap194
					d78 = snap195
					d80 = snap196
					d83 = snap197
					d117 = snap198
					d118 = snap199
					d119 = snap200
					d120 = snap201
					d121 = snap202
					d122 = snap203
					d123 = snap204
					d124 = snap205
					d126 = snap206
					d127 = snap207
					d128 = snap208
					d130 = snap209
					d131 = snap210
					d132 = snap211
					d133 = snap212
					d134 = snap213
					d135 = snap214
					d136 = snap215
					d137 = snap216
					d138 = snap217
					d139 = snap218
					d140 = snap219
					d141 = snap220
					d142 = snap221
					d143 = snap222
					d144 = snap223
					d145 = snap224
					d146 = snap225
					d148 = snap226
					d149 = snap227
					d150 = snap228
					d151 = snap229
					d152 = snap230
					d154 = snap231
					d155 = snap232
					d156 = snap233
					d157 = snap234
					d159 = snap235
					d161 = snap236
					d162 = snap237
					d165 = snap238
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps163)
					}
					return result
					ctx.FreeDesc(&d156)
					return result
				}
				ps240 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps240)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  97,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "connection_id",

		Fn: func(a ...Scmer) Scmer {
			if len(a) > 0 {
				ss, _ := querySessionState(a[0])
				if ss == nil {
					return NewInt(0)
				}
				return NewInt(int64(ss.ID))
			}
			return NewInt(0)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the process-list ID of the current session (MySQL CONNECTION_ID() equivalent)",
			Params: []*TypeDescriptor{{Kind: "any", Label: "tx", Optional: true}},
			Return: &TypeDescriptor{Kind: "int"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["connection_id"]
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
				var d11 JITValueDesc
				_ = d11
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() > 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						ctx.EmitSetcc(r0, CondSignedGreater)
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
					d11 = args[0]
					d11.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					d11 = JITPrepareScmerGoArg(ctx, d11)
					ctx.SyncDesc(&d11)
					callResults12 := JITEmitGoCallResults(ctx, GoFuncAddr(querySessionState), []JITValueDesc{d11}, []uint8{1, 1}, []uint8{1, 0})
					d13 = callResults12[0]
					_ = d13
					d14 = callResults12[1]
					_ = d14
					ctx.FreeDesc(&d11)
					ctx.StabilizeDescForControlFlow(&d13)
					ctx.EnsureDesc(&d13)
					var d15 JITValueDesc
					if d13.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d13)
						if d13.Loc != LocReg && d13.Loc != LocRegPair && d13.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitSetcc(r1, CondEqual)
						d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d15)
					}
					d16 = d15
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							if ps.General {
							}
							ps17 := PhiState{General: ps.General}
							ps17.OverlayValues = make([]JITValueDesc, 17)
							ps17.OverlayValues[0] = d0
							ps17.OverlayValues[1] = d1
							ps17.OverlayValues[2] = d2
							ps17.OverlayValues[11] = d11
							ps17.OverlayValues[13] = d13
							ps17.OverlayValues[14] = d14
							ps17.OverlayValues[15] = d15
							ps17.OverlayValues[16] = d16
							return bbs[3].RenderPS(ps17)
						}
						if ps.General {
						}
						ps18 := PhiState{General: ps.General}
						ps18.OverlayValues = make([]JITValueDesc, 17)
						ps18.OverlayValues[0] = d0
						ps18.OverlayValues[1] = d1
						ps18.OverlayValues[2] = d2
						ps18.OverlayValues[11] = d11
						ps18.OverlayValues[13] = d13
						ps18.OverlayValues[14] = d14
						ps18.OverlayValues[15] = d15
						ps18.OverlayValues[16] = d16
						return bbs[4].RenderPS(ps18)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 17)
					ps19.OverlayValues[0] = d0
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					ps19.OverlayValues[16] = d16
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 17)
					ps20.OverlayValues[0] = d0
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					snap21 := d0
					snap22 := d1
					snap23 := d2
					snap24 := d11
					snap25 := d13
					snap26 := d14
					snap27 := d15
					snap28 := d16
					alloc29 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc29)
					d0 = snap21
					d1 = snap22
					d2 = snap23
					d11 = snap24
					d13 = snap25
					d14 = snap26
					d15 = snap27
					d16 = snap28
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d15)
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
					ctx.ReclaimUntrackedRegs()
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d30.Loc == LocImm {
						ctx.EmitMakeInt(result, d30)
					} else {
						ctx.EmitMovToReg(result.Reg2, d30)
						d31 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d31)
						if d30.Loc == LocReg && d30.Reg != result.Reg2 {
							ctx.FreeReg(d30.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					ctx.ReclaimUntrackedRegs()
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d32.Loc == LocImm {
						ctx.EmitMakeInt(result, d32)
					} else {
						ctx.EmitMovToReg(result.Reg2, d32)
						d33 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d33)
						if d32.Loc == LocReg && d32.Reg != result.Reg2 {
							ctx.FreeReg(d32.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
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
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
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
					var d34 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d34 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d34)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d34 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d34)
					}
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d34)
					ctx.EnsureDesc(&d34)
					var d35 JITValueDesc
					if d34.Loc == LocImm {
						d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d34.Imm.Int()))))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegReg(r4, d34.Reg)
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d35)
					}
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						ctx.EmitMakeInt(result, d35)
					} else {
						ctx.EmitMovToReg(result.Reg2, d35)
						d36 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d36)
						if d35.Loc == LocReg && d35.Reg != result.Reg2 {
							ctx.FreeReg(d35.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				ps37 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps37)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCost: 19,
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
				declaration := declarations["kill_query"]
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
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocRegPair || d2.Loc == LocStackPair || d2.Loc == LocRegTriple || d2.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(KillSession), []JITValueDesc{d2}, 1)
				d3.NoHeapPointer = true
				ctx.EmitAndRegImm32(d3.Reg, 1)
				d3.Type = tagBool
				ctx.BindReg(d3.Reg, &d3)
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d3)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d3.Loc == LocImm {
					ctx.EmitMakeBool(result, d3)
				} else {
					ctx.EmitMakeBool(result, d3)
					ctx.FreeReg(d3.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 7,
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
