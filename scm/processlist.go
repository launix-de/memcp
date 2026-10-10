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
// Storage execution contract: scheduling and cold-column preflight waits may
// observe cancellation before row execution starts. Shard mutations remain
// atomic; recovery and write-locked loads must finish. Index, batch, and row
// loops must not contain cancellation checks.
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
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
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d77 JITValueDesc
				_ = d77
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var stackArray110 int32
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
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
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d129 JITValueDesc
				_ = d129
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
				var d194 JITValueDesc
				_ = d194
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
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
				var bbs [9]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiBase = int32(phiBase0) + int32(16)
				bbs[7].PhiBase = int32(phiBase0) + int32(32)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(32))
				_ = d3
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() > 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d4.Reg, 0)
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d5)
					}
					ctx.FreeDesc(&d4)
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							return bbs[1].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d6.Condition, lbl2)
					ctx.EmitJmp(lbl10)
					ctx.FreeDesc(&d5)
					snap7 := d1
					snap8 := d2
					snap9 := d3
					snap10 := d4
					snap11 := d5
					snap12 := d6
					alloc13 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc13)
					d1 = snap7
					d2 = snap8
					d3 = snap9
					d4 = snap10
					d5 = snap11
					d6 = snap12
					if !bbs[2].Rendered {
						snap14 := d1
						snap15 := d2
						snap16 := d3
						snap17 := d4
						snap18 := d5
						snap19 := d6
						alloc20 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc20)
						d1 = snap14
						d2 = snap15
						d3 = snap16
						d4 = snap17
						d5 = snap18
						d6 = snap19
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d21 = args[0]
					d21.ID = 0
					d23 = d21
					d23.ID = 0
					d22 = ctx.EmitBoolDesc(&d23, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d22)
					ctx.FreeDesc(&d21)
					ctx.SyncDesc(&d22)
					if d22.Loc == LocReg || d22.Loc == LocFPReg {
						ctx.ProtectReg(d22.Reg)
					} else if d22.Loc == LocRegPair {
						ctx.ProtectReg(d22.Reg)
						ctx.ProtectReg(d22.Reg2)
					}
					d24 = d22
					if d24.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d24)
					ctx.EmitStoreToStack(d24, int32(bbs[2].PhiBase)+int32(0))
					if d22.Loc == LocReg || d22.Loc == LocFPReg {
						ctx.UnprotectReg(d22.Reg)
					} else if d22.Loc == LocRegPair {
						ctx.UnprotectReg(d22.Reg)
						ctx.UnprotectReg(d22.Reg2)
					}
					return bbs[2].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d25 = ctx.EmitGoCallScalar(GoFuncAddr(Snapshot), []JITValueDesc{}, 3)
					d25.NoHeapPointer = false
					ctx.BindReg(d25.Reg, &d25)
					ctx.BindReg(d25.Reg2, &d25)
					ctx.BindReg(d25.Reg3, &d25)
					ctx.StabilizeDescForControlFlow(&d25)
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
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					callResults27 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d26, d26}, []uint8{3}, []uint8{1})
					d28 = callResults27[0]
					d28.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d28)
					ctx.FreeDesc(&d26)
					if d25.SliceSizeKnown {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
					} else if d25.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
					} else if d25.Loc == LocStackTriple {
						d29 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d25)
						if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
						} else if d25.Loc == LocReg {
							d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d29)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					return bbs[3].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d30)
					}
					if d30.Loc == LocReg && d2.Loc == LocReg && d30.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d30)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d30)
					ctx.EnsureDesc(&d29)
					ctx.EnsureDescsTogether(&d30, &d29)
					if d30.Loc == LocImm && d29.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d30.Imm.Int() < d29.Imm.Int())}
					} else if d29.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d30.Reg)
						if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d30.Reg, int32(d29.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d29.Imm.Int()))
							ctx.EmitCmpInt64(d30.Reg, ctx.ScratchReg)
						}
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d31)
					} else if d30.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d30.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d29.Reg)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d31)
					} else {
						r3 := ctx.AllocRegExcept(d30.Reg)
						ctx.EmitCmpInt64(d30.Reg, d29.Reg)
						d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d31)
					}
					d32 = d31
					ctx.EnsureDesc(&d32)
					if d32.Loc != LocImm && d32.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d32.Loc == LocImm {
						if d32.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d32.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d31)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap33 := d1
						snap34 := d2
						snap35 := d3
						snap36 := d4
						snap37 := d5
						snap38 := d6
						snap39 := d21
						snap40 := d22
						snap41 := d23
						snap42 := d24
						snap43 := d25
						snap44 := d26
						snap45 := d28
						snap46 := d29
						snap47 := d30
						snap48 := d31
						snap49 := d32
						alloc50 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc50)
						d1 = snap33
						d2 = snap34
						d3 = snap35
						d4 = snap36
						d5 = snap37
						d6 = snap38
						d21 = snap39
						d22 = snap40
						d23 = snap41
						d24 = snap42
						d25 = snap43
						d26 = snap44
						d28 = snap45
						d29 = snap46
						d30 = snap47
						d31 = snap48
						d32 = snap49
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					d51 = ctx.EmitLoadScalarSliceElement(&d25, &d30, 8, JITTypeUnknown)
					ctx.StabilizeDescForControlFlow(&d51)
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d52 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d51}, 2)
					d52.NoHeapPointer = false
					ctx.BindReg(d52.Reg, &d52)
					ctx.BindReg(d52.Reg2, &d52)
					ctx.StabilizeDescForControlFlow(&d52)
					d53 = d1
					ctx.EnsureDesc(&d53)
					if d53.Loc != LocImm && d53.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d53.Loc == LocImm {
						if d53.Imm.Bool() {
							ctx.SyncDesc(&d52)
							if d52.Loc == LocReg || d52.Loc == LocFPReg {
								ctx.ProtectReg(d52.Reg)
							} else if d52.Loc == LocRegPair {
								ctx.ProtectReg(d52.Reg)
								ctx.ProtectReg(d52.Reg2)
							}
							d54 = d52
							if d54.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d54)
							if d54.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d54, int32(bbs[7].PhiBase)+int32(0), 2)
							} else if d54.Loc == LocInputPair {
								ctx.EnsureDesc(&d54)
								ctx.EmitStoreScmerToStack(d54, int32(bbs[7].PhiBase)+int32(0))
							} else if d54.Loc == LocRegPair || d54.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d54, int32(bbs[7].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d54)
								ctx.EmitStoreToStack(d54, int32(bbs[7].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
							}
							if d52.Loc == LocReg || d52.Loc == LocFPReg {
								ctx.UnprotectReg(d52.Reg)
							} else if d52.Loc == LocRegPair {
								ctx.UnprotectReg(d52.Reg)
								ctx.UnprotectReg(d52.Reg2)
							}
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d53.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl9)
					snap55 := d1
					snap56 := d2
					snap57 := d3
					snap58 := d4
					snap59 := d5
					snap60 := d6
					snap61 := d21
					snap62 := d22
					snap63 := d23
					snap64 := d24
					snap65 := d25
					snap66 := d26
					snap67 := d28
					snap68 := d29
					snap69 := d30
					snap70 := d31
					snap71 := d32
					snap72 := d51
					snap73 := d52
					snap74 := d53
					snap75 := d54
					alloc76 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d52)
					if d52.Loc == LocReg || d52.Loc == LocFPReg {
						ctx.ProtectReg(d52.Reg)
					} else if d52.Loc == LocRegPair {
						ctx.ProtectReg(d52.Reg)
						ctx.ProtectReg(d52.Reg2)
					}
					d77 = d52
					if d77.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d77)
					if d77.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d77, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d77.Loc == LocInputPair {
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreScmerToStack(d77, int32(bbs[7].PhiBase)+int32(0))
					} else if d77.Loc == LocRegPair || d77.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d77, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d52.Loc == LocReg || d52.Loc == LocFPReg {
						ctx.UnprotectReg(d52.Reg)
					} else if d52.Loc == LocRegPair {
						ctx.UnprotectReg(d52.Reg)
						ctx.UnprotectReg(d52.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc76)
					d1 = snap55
					d2 = snap56
					d3 = snap57
					d4 = snap58
					d5 = snap59
					d6 = snap60
					d21 = snap61
					d22 = snap62
					d23 = snap63
					d24 = snap64
					d25 = snap65
					d26 = snap66
					d28 = snap67
					d29 = snap68
					d30 = snap69
					d31 = snap70
					d32 = snap71
					d51 = snap72
					d52 = snap73
					d53 = snap74
					d54 = snap75
					if !bbs[7].Rendered {
						snap78 := d1
						snap79 := d2
						snap80 := d3
						snap81 := d4
						snap82 := d5
						snap83 := d6
						snap84 := d21
						snap85 := d22
						snap86 := d23
						snap87 := d24
						snap88 := d25
						snap89 := d26
						snap90 := d28
						snap91 := d29
						snap92 := d30
						snap93 := d31
						snap94 := d32
						snap95 := d51
						snap96 := d52
						snap97 := d53
						snap98 := d54
						snap99 := d77
						alloc100 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc100)
						d1 = snap78
						d2 = snap79
						d3 = snap80
						d4 = snap81
						d5 = snap82
						d6 = snap83
						d21 = snap84
						d22 = snap85
						d23 = snap86
						d24 = snap87
						d25 = snap88
						d26 = snap89
						d28 = snap90
						d29 = snap91
						d30 = snap92
						d31 = snap93
						d32 = snap94
						d51 = snap95
						d52 = snap96
						d53 = snap97
						d54 = snap98
						d77 = snap99
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d28)
					d101 = ctx.EmitNewSliceFromGoSlice(&d28)
					ctx.SyncDesc(&d101)
					if d101.Loc == LocRegPair || d101.Loc == LocStackPair || d101.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d101, &result)
						result.Type = d101.Type
					} else {
						switch d101.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d101)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d101)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d101)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d101, &result)
							result.Type = d101.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d102)
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm && d102.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d103.Imm.Int() - d102.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d103.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d103.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d103.Reg)
						}
						if d102.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d102.Imm.Int()))
							ctx.EmitSubInt64(r4, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r4, d102.Reg)
						}
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d105)
					}
					r5 := ctx.EmitSliceDataAfterLow(&d52, &d102, 1)
					d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d106)
					ctx.BindReg(r5, &d106)
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d106)
					ctx.EnsureDesc(&d106)
					if d106.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d106.Imm.Int()))
					} else {
						r6 = d106.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d105)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d105.Imm.Int()))
					} else {
						r7 = d105.Reg
					}
					ctx.ProtectReg(r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d107 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d107)
					ctx.BindReg(r7, &d107)
					ctx.BindReg(r6, &d107)
					ctx.BindReg(r7, &d107)
					ctx.StabilizeDescForControlFlow(&d107)
					ctx.SyncDesc(&d107)
					if d107.Loc == LocReg || d107.Loc == LocFPReg {
						ctx.ProtectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.ProtectReg(d107.Reg)
						ctx.ProtectReg(d107.Reg2)
					}
					d108 = d107
					if d108.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d108)
					if d108.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d108, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d108.Loc == LocInputPair {
						ctx.EnsureDesc(&d108)
						ctx.EmitStoreScmerToStack(d108, int32(bbs[7].PhiBase)+int32(0))
					} else if d108.Loc == LocRegPair || d108.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d108, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d108)
						ctx.EmitStoreToStack(d108, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d107.Loc == LocReg || d107.Loc == LocFPReg {
						ctx.UnprotectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.UnprotectReg(d107.Reg)
						ctx.UnprotectReg(d107.Reg2)
					}
					return bbs[7].Render()
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d28)
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d51)
					d109 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d51}, 2)
					d109.NoHeapPointer = false
					ctx.BindReg(d109.Reg, &d109)
					ctx.BindReg(d109.Reg2, &d109)
					stackArray110 = ctx.AllocStack(int32(256))
					_ = stackArray110
					d111 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.SyncDesc(&d111)
					ctx.EmitStoreScmerToStack(d111, int32(stackArray110)+int32(0))
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						fieldAddr := uintptr(d51.Imm.Int()) + 0
						r8 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r8, fieldAddr)
						d112 = JITValueDesc{Loc: LocReg, Reg: r8}
						ctx.BindReg(r8, &d112)
					} else {
						off := int32(0)
						baseReg := d51.Reg
						r9 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r9, baseReg, off)
						d112 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d112)
					}
					ctx.EnsureDesc(&d112)
					ctx.EnsureDesc(&d112)
					if d112.Loc == LocImm {
						d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d112.Imm.Int()))))}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegReg(r10, d112.Reg)
						d113 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d113)
					}
					ctx.FreeDesc(&d112)
					ctx.EnsureDesc(&d113)
					ctx.SyncDesc(&d113)
					ctx.EnsureDesc(&d113)
					ctx.EmitStoreTypedScmerToStack(d113, tagInt, int32(stackArray110)+int32(16))
					d114 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.SyncDesc(&d114)
					ctx.EmitStoreScmerToStack(d114, int32(stackArray110)+int32(32))
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						fieldAddr := uintptr(d51.Imm.Int()) + 8
						r11 := ctx.AllocReg()
						r12 := ctx.AllocRegExcept(r11)
						r13 := ctx.AllocRegExcept(r11, r12)
						ctx.EmitMovRegMem64(r11, fieldAddr)
						ctx.EmitMovRegMem64(r12, fieldAddr+8)
						ctx.EmitMovRegMem64(r13, fieldAddr+16)
						d115 = JITValueDesc{Loc: LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
						ctx.BindReg(r11, &d115)
						ctx.BindReg(r12, &d115)
						ctx.BindReg(r13, &d115)
					} else {
						off := int32(8)
						baseReg := d51.Reg
						r14 := ctx.AllocRegExcept(baseReg)
						r15 := ctx.AllocRegExcept(baseReg, r14)
						r16 := ctx.AllocRegExcept(baseReg, r14, r15)
						ctx.EmitMovRegMem(r14, baseReg, off)
						ctx.EmitMovRegMem(r15, baseReg, off+8)
						ctx.EmitMovRegMem(r16, baseReg, off+16)
						d115 = JITValueDesc{Loc: LocRegTriple, Reg: r14, Reg2: r15, Reg3: r16}
						ctx.BindReg(r14, &d115)
						ctx.BindReg(r15, &d115)
						ctx.BindReg(r16, &d115)
					}
					ctx.EnsureDesc(&d115)
					ctx.SyncDesc(&d115)
					ctx.EmitStoreScmerToStack(d115, int32(stackArray110)+int32(48))
					d116 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.SyncDesc(&d116)
					ctx.EmitStoreScmerToStack(d116, int32(stackArray110)+int32(64))
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocImm {
						fieldAddr := uintptr(d51.Imm.Int()) + 24
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d117 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d117)
						ctx.BindReg(r18, &d117)
						ctx.BindReg(r19, &d117)
					} else {
						off := int32(24)
						baseReg := d51.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d117 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d117)
						ctx.BindReg(r21, &d117)
						ctx.BindReg(r22, &d117)
					}
					ctx.EnsureDesc(&d117)
					ctx.SyncDesc(&d117)
					ctx.EmitStoreScmerToStack(d117, int32(stackArray110)+int32(80))
					d118 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.SyncDesc(&d118)
					ctx.EmitStoreScmerToStack(d118, int32(stackArray110)+int32(96))
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d119 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d51}, 2)
					d119.NoHeapPointer = false
					ctx.BindReg(d119.Reg, &d119)
					ctx.BindReg(d119.Reg2, &d119)
					ctx.EnsureDesc(&d119)
					ctx.SyncDesc(&d119)
					ctx.EmitStoreScmerToStack(d119, int32(stackArray110)+int32(112))
					d120 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.SyncDesc(&d120)
					ctx.EmitStoreScmerToStack(d120, int32(stackArray110)+int32(128))
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d121 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d51}, 2)
					d121.NoHeapPointer = false
					ctx.BindReg(d121.Reg, &d121)
					ctx.BindReg(d121.Reg2, &d121)
					ctx.EnsureDesc(&d121)
					ctx.SyncDesc(&d121)
					ctx.EmitStoreScmerToStack(d121, int32(stackArray110)+int32(144))
					d122 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.SyncDesc(&d122)
					ctx.EmitStoreScmerToStack(d122, int32(stackArray110)+int32(160))
					if d51.Loc == LocRegPair || d51.Loc == LocStackPair || d51.Loc == LocRegTriple || d51.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d51)
					d123 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d51}, 1)
					d123.NoHeapPointer = true
					ctx.BindReg(d123.Reg, &d123)
					ctx.EnsureDesc(&d123)
					ctx.SyncDesc(&d123)
					ctx.EnsureDesc(&d123)
					ctx.EmitStoreTypedScmerToStack(d123, tagInt, int32(stackArray110)+int32(176))
					d124 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.SyncDesc(&d124)
					ctx.EmitStoreScmerToStack(d124, int32(stackArray110)+int32(192))
					ctx.EnsureDesc(&d109)
					ctx.SyncDesc(&d109)
					ctx.EmitStoreScmerToStack(d109, int32(stackArray110)+int32(208))
					d125 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.SyncDesc(&d125)
					ctx.EmitStoreScmerToStack(d125, int32(stackArray110)+int32(224))
					ctx.EnsureDesc(&d3)
					ctx.SyncDesc(&d3)
					ctx.EmitStoreScmerToStack(d3, int32(stackArray110)+int32(240))
					d126 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d126
					r23 := ctx.AllocReg()
					r24 := ctx.AllocRegExcept(r23)
					r25 := ctx.AllocRegExcept(r23, r24)
					d127 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r23, Reg2: r24, Reg3: r25}
					ctx.BindReg(r23, &d127)
					ctx.BindReg(r24, &d127)
					ctx.BindReg(r25, &d127)
					ctx.BindReg(r23, &d127)
					ctx.BindReg(r24, &d127)
					ctx.BindReg(r25, &d127)
					ctx.EmitLeaRegMem(d127.Reg, ctx.StackReg, int32(stackArray110))
					ctx.EmitMovRegImm64(d127.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d127.Reg3, uint64(16))
					callResults128 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d127}, []uint8{2}, []uint8{1})
					d129 = callResults128[0]
					ctx.EnsureDesc(&d30)
					ctx.SyncDesc(&d129)
					d130 = d28
					d130.ID = 0
					d131 = d30
					d131.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d130, &d131, &d129, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d30)
						d131 = d30
						d131.ID = 0
						ctx.EmitStoreScmerSliceElement(&d130, &d131, &d129, int32(16))
					}
					ctx.FreeDesc(&d131)
					ctx.FreeDesc(&d129)
					ctx.SyncDesc(&d30)
					if d30.Loc == LocReg || d30.Loc == LocFPReg {
						ctx.ProtectReg(d30.Reg)
					} else if d30.Loc == LocRegPair {
						ctx.ProtectReg(d30.Reg)
						ctx.ProtectReg(d30.Reg2)
					}
					d132 = d30
					if d132.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d132)
					ctx.EmitStoreToStack(d132, int32(bbs[3].PhiBase)+int32(0))
					if d30.Loc == LocReg || d30.Loc == LocFPReg {
						ctx.UnprotectReg(d30.Reg)
					} else if d30.Loc == LocRegPair {
						ctx.UnprotectReg(d30.Reg)
						ctx.UnprotectReg(d30.Reg2)
					}
					return bbs[3].Render()
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					if d52.SliceSizeKnown {
						d133 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d52.KnownSliceLen))}
					} else if d52.Loc == LocImm {
						d133 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d52.Imm.String())))}
					} else if d52.Loc == LocStackTriple {
						d133 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d52.StackOff + 8, NoHeapPointer: true}
					} else if d52.Loc == LocStackPair {
						d133 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d52.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d52)
						if d52.Loc == LocRegPair || d52.Loc == LocRegTriple {
							d133 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg2, ID: 0}
						} else if d52.Loc == LocReg {
							d133 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d133)
					if d133.Loc == LocImm {
						d134 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d133.Imm.Int() > 100)}
					} else {
						r26 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d133.Reg, 100)
						d134 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedGreater}
						ctx.BindReg(r26, &d134)
					}
					ctx.FreeDesc(&d133)
					d135 = d134
					ctx.EnsureDesc(&d135)
					if d135.Loc != LocImm && d135.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d135.Loc == LocImm {
						if d135.Imm.Bool() {
							return bbs[6].Render()
						}
						ctx.SyncDesc(&d52)
						if d52.Loc == LocReg || d52.Loc == LocFPReg {
							ctx.ProtectReg(d52.Reg)
						} else if d52.Loc == LocRegPair {
							ctx.ProtectReg(d52.Reg)
							ctx.ProtectReg(d52.Reg2)
						}
						d136 = d52
						if d136.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d136)
						if d136.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d136, int32(bbs[7].PhiBase)+int32(0), 2)
						} else if d136.Loc == LocInputPair {
							ctx.EnsureDesc(&d136)
							ctx.EmitStoreScmerToStack(d136, int32(bbs[7].PhiBase)+int32(0))
						} else if d136.Loc == LocRegPair || d136.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d136, int32(bbs[7].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d136)
							ctx.EmitStoreToStack(d136, int32(bbs[7].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
						}
						if d52.Loc == LocReg || d52.Loc == LocFPReg {
							ctx.UnprotectReg(d52.Reg)
						} else if d52.Loc == LocRegPair {
							ctx.UnprotectReg(d52.Reg)
							ctx.UnprotectReg(d52.Reg2)
						}
						return bbs[7].Render()
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitJump(d135.Condition, lbl7)
					ctx.EmitJmp(lbl12)
					ctx.FreeDesc(&d134)
					snap137 := d1
					snap138 := d2
					snap139 := d3
					snap140 := d4
					snap141 := d5
					snap142 := d6
					snap143 := d21
					snap144 := d22
					snap145 := d23
					snap146 := d24
					snap147 := d25
					snap148 := d26
					snap149 := d28
					snap150 := d29
					snap151 := d30
					snap152 := d31
					snap153 := d32
					snap154 := d51
					snap155 := d52
					snap156 := d53
					snap157 := d54
					snap158 := d77
					snap159 := d101
					snap160 := d102
					snap161 := d103
					snap162 := d104
					snap163 := d105
					snap164 := d106
					snap165 := d107
					snap166 := d108
					snap167 := d109
					snap168 := d111
					snap169 := d112
					snap170 := d113
					snap171 := d114
					snap172 := d115
					snap173 := d116
					snap174 := d117
					snap175 := d118
					snap176 := d119
					snap177 := d120
					snap178 := d121
					snap179 := d122
					snap180 := d123
					snap181 := d124
					snap182 := d125
					snap183 := d126
					snap184 := d127
					snap185 := d129
					snap186 := d130
					snap187 := d131
					snap188 := d132
					snap189 := d133
					snap190 := d134
					snap191 := d135
					snap192 := d136
					alloc193 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d52)
					if d52.Loc == LocReg || d52.Loc == LocFPReg {
						ctx.ProtectReg(d52.Reg)
					} else if d52.Loc == LocRegPair {
						ctx.ProtectReg(d52.Reg)
						ctx.ProtectReg(d52.Reg2)
					}
					d194 = d52
					if d194.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d194)
					if d194.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d194, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d194.Loc == LocInputPair {
						ctx.EnsureDesc(&d194)
						ctx.EmitStoreScmerToStack(d194, int32(bbs[7].PhiBase)+int32(0))
					} else if d194.Loc == LocRegPair || d194.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d194, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d194)
						ctx.EmitStoreToStack(d194, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d52.Loc == LocReg || d52.Loc == LocFPReg {
						ctx.UnprotectReg(d52.Reg)
					} else if d52.Loc == LocRegPair {
						ctx.UnprotectReg(d52.Reg)
						ctx.UnprotectReg(d52.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc193)
					d1 = snap137
					d2 = snap138
					d3 = snap139
					d4 = snap140
					d5 = snap141
					d6 = snap142
					d21 = snap143
					d22 = snap144
					d23 = snap145
					d24 = snap146
					d25 = snap147
					d26 = snap148
					d28 = snap149
					d29 = snap150
					d30 = snap151
					d31 = snap152
					d32 = snap153
					d51 = snap154
					d52 = snap155
					d53 = snap156
					d54 = snap157
					d77 = snap158
					d101 = snap159
					d102 = snap160
					d103 = snap161
					d104 = snap162
					d105 = snap163
					d106 = snap164
					d107 = snap165
					d108 = snap166
					d109 = snap167
					d111 = snap168
					d112 = snap169
					d113 = snap170
					d114 = snap171
					d115 = snap172
					d116 = snap173
					d117 = snap174
					d118 = snap175
					d119 = snap176
					d120 = snap177
					d121 = snap178
					d122 = snap179
					d123 = snap180
					d124 = snap181
					d125 = snap182
					d126 = snap183
					d127 = snap184
					d129 = snap185
					d130 = snap186
					d131 = snap187
					d132 = snap188
					d133 = snap189
					d134 = snap190
					d135 = snap191
					d136 = snap192
					if !bbs[7].Rendered {
						snap195 := d1
						snap196 := d2
						snap197 := d3
						snap198 := d4
						snap199 := d5
						snap200 := d6
						snap201 := d21
						snap202 := d22
						snap203 := d23
						snap204 := d24
						snap205 := d25
						snap206 := d26
						snap207 := d28
						snap208 := d29
						snap209 := d30
						snap210 := d31
						snap211 := d32
						snap212 := d51
						snap213 := d52
						snap214 := d53
						snap215 := d54
						snap216 := d77
						snap217 := d101
						snap218 := d102
						snap219 := d103
						snap220 := d104
						snap221 := d105
						snap222 := d106
						snap223 := d107
						snap224 := d108
						snap225 := d109
						snap226 := d111
						snap227 := d112
						snap228 := d113
						snap229 := d114
						snap230 := d115
						snap231 := d116
						snap232 := d117
						snap233 := d118
						snap234 := d119
						snap235 := d120
						snap236 := d121
						snap237 := d122
						snap238 := d123
						snap239 := d124
						snap240 := d125
						snap241 := d126
						snap242 := d127
						snap243 := d129
						snap244 := d130
						snap245 := d131
						snap246 := d132
						snap247 := d133
						snap248 := d134
						snap249 := d135
						snap250 := d136
						snap251 := d194
						alloc252 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc252)
						d1 = snap195
						d2 = snap196
						d3 = snap197
						d4 = snap198
						d5 = snap199
						d6 = snap200
						d21 = snap201
						d22 = snap202
						d23 = snap203
						d24 = snap204
						d25 = snap205
						d26 = snap206
						d28 = snap207
						d29 = snap208
						d30 = snap209
						d31 = snap210
						d32 = snap211
						d51 = snap212
						d52 = snap213
						d53 = snap214
						d54 = snap215
						d77 = snap216
						d101 = snap217
						d102 = snap218
						d103 = snap219
						d104 = snap220
						d105 = snap221
						d106 = snap222
						d107 = snap223
						d108 = snap224
						d109 = snap225
						d111 = snap226
						d112 = snap227
						d113 = snap228
						d114 = snap229
						d115 = snap230
						d116 = snap231
						d117 = snap232
						d118 = snap233
						d119 = snap234
						d120 = snap235
						d121 = snap236
						d122 = snap237
						d123 = snap238
						d124 = snap239
						d125 = snap240
						d126 = snap241
						d127 = snap242
						d129 = snap243
						d130 = snap244
						d131 = snap245
						d132 = snap246
						d133 = snap247
						d134 = snap248
						d135 = snap249
						d136 = snap250
						d194 = snap251
					}
					if !bbs[6].Rendered {
						return bbs[6].Render()
					}
					return result
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
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() > 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 0)
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
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
					d7 = args[0]
					d7.ID = 0
					d7 = JITPrepareScmerGoArg(ctx, d7)
					ctx.SyncDesc(&d7)
					callResults8 := JITEmitGoCallResults(ctx, GoFuncAddr(querySessionState), []JITValueDesc{d7}, []uint8{1, 1}, []uint8{1, 0})
					d9 = callResults8[0]
					_ = d9
					d10 = callResults8[1]
					_ = d10
					ctx.FreeDesc(&d7)
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d9)
						if d9.Loc != LocReg && d9.Loc != LocRegPair && d9.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpRegImm32(d9.Reg, 0)
						ctx.EmitSetcc(r1, CondEqual)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d11)
					}
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[4].Render()
					}
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
					ctx.FlushRegisterMoves()
					if !bbs[4].Rendered {
						snap13 := d0
						snap14 := d1
						snap15 := d2
						snap16 := d7
						snap17 := d9
						snap18 := d10
						snap19 := d11
						snap20 := d12
						alloc21 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc21)
						d0 = snap13
						d1 = snap14
						d2 = snap15
						d7 = snap16
						d9 = snap17
						d10 = snap18
						d11 = snap19
						d12 = snap20
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d11)
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
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d22.Loc == LocImm {
						ctx.EmitMakeInt(result, d22)
					} else {
						ctx.EmitMovToReg(result.Reg2, d22)
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d23)
						if d22.Loc == LocReg && d22.Reg != result.Reg2 {
							ctx.FreeReg(d22.Reg)
						}
					}
					result.Type = tagInt
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
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d24.Loc == LocImm {
						ctx.EmitMakeInt(result, d24)
					} else {
						ctx.EmitMovToReg(result.Reg2, d24)
						d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d25)
						if d24.Loc == LocReg && d24.Reg != result.Reg2 {
							ctx.FreeReg(d24.Reg)
						}
					}
					result.Type = tagInt
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						fieldAddr := uintptr(d9.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d26 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d26)
					} else {
						off := int32(0)
						baseReg := d9.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d26 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d26)
					}
					ctx.EnsureDesc(&d26)
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d26.Imm.Int()))))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegReg(r4, d26.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d27)
					}
					ctx.FreeDesc(&d26)
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocImm {
						ctx.EmitMakeInt(result, d27)
					} else {
						ctx.EmitMovToReg(result.Reg2, d27)
						d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d28)
						if d27.Loc == LocReg && d27.Reg != result.Reg2 {
							ctx.FreeReg(d27.Reg)
						}
					}
					result.Type = tagInt
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
				ctx.SyncDesc(&d3)
				if ctx.hasBooleanFlags(d3) {
					return d3
				}
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
