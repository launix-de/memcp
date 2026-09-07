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
				var d22 JITValueDesc
				_ = d22
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
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
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d52 JITValueDesc
				_ = d52
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d110 JITValueDesc
				_ = d110
				var d142 JITValueDesc
				_ = d142
				var d145 JITValueDesc
				_ = d145
				var d178 JITValueDesc
				_ = d178
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var d185 JITValueDesc
				_ = d185
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var stackArray190 int32
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var d219 JITValueDesc
				_ = d219
				var d221 JITValueDesc
				_ = d221
				var d292 JITValueDesc
				_ = d292
				var d295 JITValueDesc
				_ = d295
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
						d7 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedGreater}
						ctx.BindReg(r1, &d7)
					}
					ctx.FreeDesc(&d6)
					d8 = d7
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocImm && d8.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d8.Condition, lbl2)
					ctx.EmitJmp(lbl10)
					snap12 := d3
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d7
					snap17 := d8
					snap18 := d11
					alloc19 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc19)
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					d8 = snap17
					d11 = snap18
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc19)
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					d8 = snap17
					d11 = snap18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 12)
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
					ps20.OverlayValues[11] = d11
					ps21 := PhiState{General: true}
					ps21.OverlayValues = make([]JITValueDesc, 12)
					ps21.OverlayValues[3] = d3
					ps21.OverlayValues[4] = d4
					ps21.OverlayValues[5] = d5
					ps21.OverlayValues[6] = d6
					ps21.OverlayValues[7] = d7
					ps21.OverlayValues[8] = d8
					ps21.OverlayValues[11] = d11
					ps21.PhiValues = make([]JITValueDesc, 1)
					d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps21.PhiValues[0] = d22
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d7
					snap28 := d8
					snap29 := d11
					snap30 := d22
					alloc31 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps21)
					}
					ctx.RestoreAllocState(alloc31)
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d7 = snap27
					d8 = snap28
					d11 = snap29
					d22 = snap30
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps20)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					ctx.ReclaimUntrackedRegs()
					d32 = args[0]
					d32.ID = 0
					d34 = d32
					d34.ID = 0
					d33 = ctx.EmitBoolDesc(&d34, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d33)
					ctx.FreeDesc(&d32)
					if ps.General {
						ctx.SyncDesc(&d33)
						if d33.Loc == LocReg {
							ctx.ProtectReg(d33.Reg)
						} else if d33.Loc == LocRegPair {
							ctx.ProtectReg(d33.Reg)
							ctx.ProtectReg(d33.Reg2)
						}
						d35 = d33
						if d35.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d35)
						ctx.EmitStoreToStack(d35, int32(bbs[2].PhiBase)+int32(0))
						if d33.Loc == LocReg {
							ctx.UnprotectReg(d33.Reg)
						} else if d33.Loc == LocRegPair {
							ctx.UnprotectReg(d33.Reg)
							ctx.UnprotectReg(d33.Reg2)
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
					ps36.OverlayValues[22] = d22
					ps36.OverlayValues[32] = d32
					ps36.OverlayValues[33] = d33
					ps36.OverlayValues[34] = d34
					ps36.OverlayValues[35] = d35
					ps36.PhiValues = make([]JITValueDesc, 1)
					d37 = d33
					ps36.PhiValues[0] = d37
					if ps36.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps36)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d38 := ps.PhiValues[0]
							ctx.EnsureDesc(&d38)
							ctx.EmitStoreToStack(d38, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					d39 = ctx.EmitGoCallScalar(GoFuncAddr(Snapshot), []JITValueDesc{}, 3)
					d39.NoHeapPointer = false
					ctx.BindReg(d39.Reg, &d39)
					ctx.BindReg(d39.Reg2, &d39)
					ctx.BindReg(d39.Reg3, &d39)
					ctx.StabilizeDescForControlFlow(&d39)
					var d40 JITValueDesc
					if d39.SliceSizeKnown {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.KnownSliceLen))}
					} else if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.StackOff))}
					} else if d39.Loc == LocStackTriple {
						d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d39.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d39)
						if d39.Loc == LocRegPair || d39.Loc == LocRegTriple {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg2, ID: 0}
						} else if d39.Loc == LocReg {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					callResults41 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d40, d40}, []uint8{3}, []uint8{1})
					d42 = callResults41[0]
					d42.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.FreeDesc(&d40)
					var d43 JITValueDesc
					if d39.SliceSizeKnown {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.KnownSliceLen))}
					} else if d39.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.StackOff))}
					} else if d39.Loc == LocStackTriple {
						d43 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d39.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d39)
						if d39.Loc == LocRegPair || d39.Loc == LocRegTriple {
							d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg2, ID: 0}
						} else if d39.Loc == LocReg {
							d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d43)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
						}
					}
					ps44 := PhiState{General: ps.General}
					ps44.OverlayValues = make([]JITValueDesc, 44)
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[6] = d6
					ps44.OverlayValues[7] = d7
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[22] = d22
					ps44.OverlayValues[32] = d32
					ps44.OverlayValues[33] = d33
					ps44.OverlayValues[34] = d34
					ps44.OverlayValues[35] = d35
					ps44.OverlayValues[37] = d37
					ps44.OverlayValues[38] = d38
					ps44.OverlayValues[39] = d39
					ps44.OverlayValues[40] = d40
					ps44.OverlayValues[42] = d42
					ps44.OverlayValues[43] = d43
					ps44.PhiValues = make([]JITValueDesc, 1)
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps44.PhiValues[0] = d45
					if ps44.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps44)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d46 := ps.PhiValues[0]
							ctx.EnsureDesc(&d46)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d46)
							} else {
								ctx.EmitStoreToStack(d46, int32(bbs[3].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
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
					var d47 JITValueDesc
					if d4.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d47)
					}
					if d47.Loc == LocReg && d4.Loc == LocReg && d47.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d47)
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d43)
					ctx.EnsureDescsTogether(&d47, &d43)
					var d48 JITValueDesc
					if d47.Loc == LocImm && d43.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Int() < d43.Imm.Int())}
					} else if d43.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d47.Reg)
						if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d47.Reg, int32(d43.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
							ctx.EmitCmpInt64(d47.Reg, RegR11)
						}
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d48)
					} else if d47.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d43.Reg)
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d48)
					} else {
						r4 := ctx.AllocRegExcept(d47.Reg)
						ctx.EmitCmpInt64(d47.Reg, d43.Reg)
						d48 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d48)
					}
					d49 = d48
					ctx.EnsureDesc(&d49)
					if d49.Loc != LocImm && d49.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d49.Loc == LocImm {
						if d49.Imm.Bool() {
							if ps.General {
							}
							ps50 := PhiState{General: ps.General}
							ps50.OverlayValues = make([]JITValueDesc, 50)
							ps50.OverlayValues[3] = d3
							ps50.OverlayValues[4] = d4
							ps50.OverlayValues[5] = d5
							ps50.OverlayValues[6] = d6
							ps50.OverlayValues[7] = d7
							ps50.OverlayValues[8] = d8
							ps50.OverlayValues[11] = d11
							ps50.OverlayValues[22] = d22
							ps50.OverlayValues[32] = d32
							ps50.OverlayValues[33] = d33
							ps50.OverlayValues[34] = d34
							ps50.OverlayValues[35] = d35
							ps50.OverlayValues[37] = d37
							ps50.OverlayValues[38] = d38
							ps50.OverlayValues[39] = d39
							ps50.OverlayValues[40] = d40
							ps50.OverlayValues[42] = d42
							ps50.OverlayValues[43] = d43
							ps50.OverlayValues[45] = d45
							ps50.OverlayValues[46] = d46
							ps50.OverlayValues[47] = d47
							ps50.OverlayValues[48] = d48
							ps50.OverlayValues[49] = d49
							return bbs[4].RenderPS(ps50)
						}
						if ps.General {
						}
						ps51 := PhiState{General: ps.General}
						ps51.OverlayValues = make([]JITValueDesc, 50)
						ps51.OverlayValues[3] = d3
						ps51.OverlayValues[4] = d4
						ps51.OverlayValues[5] = d5
						ps51.OverlayValues[6] = d6
						ps51.OverlayValues[7] = d7
						ps51.OverlayValues[8] = d8
						ps51.OverlayValues[11] = d11
						ps51.OverlayValues[22] = d22
						ps51.OverlayValues[32] = d32
						ps51.OverlayValues[33] = d33
						ps51.OverlayValues[34] = d34
						ps51.OverlayValues[35] = d35
						ps51.OverlayValues[37] = d37
						ps51.OverlayValues[38] = d38
						ps51.OverlayValues[39] = d39
						ps51.OverlayValues[40] = d40
						ps51.OverlayValues[42] = d42
						ps51.OverlayValues[43] = d43
						ps51.OverlayValues[45] = d45
						ps51.OverlayValues[46] = d46
						ps51.OverlayValues[47] = d47
						ps51.OverlayValues[48] = d48
						ps51.OverlayValues[49] = d49
						return bbs[5].RenderPS(ps51)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d52 := ps.PhiValues[0]
							ctx.EnsureDesc(&d52)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d52)
							} else {
								ctx.EmitStoreToStack(d52, int32(bbs[3].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					ctx.EmitJump(d49.Condition, lbl5)
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d8
					snap59 := d11
					snap60 := d22
					snap61 := d32
					snap62 := d33
					snap63 := d34
					snap64 := d35
					snap65 := d37
					snap66 := d38
					snap67 := d39
					snap68 := d40
					snap69 := d42
					snap70 := d43
					snap71 := d45
					snap72 := d46
					snap73 := d47
					snap74 := d48
					snap75 := d49
					snap76 := d52
					alloc77 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc77)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d8 = snap58
					d11 = snap59
					d22 = snap60
					d32 = snap61
					d33 = snap62
					d34 = snap63
					d35 = snap64
					d37 = snap65
					d38 = snap66
					d39 = snap67
					d40 = snap68
					d42 = snap69
					d43 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d48 = snap74
					d49 = snap75
					d52 = snap76
					ctx.RestoreAllocState(alloc77)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d8 = snap58
					d11 = snap59
					d22 = snap60
					d32 = snap61
					d33 = snap62
					d34 = snap63
					d35 = snap64
					d37 = snap65
					d38 = snap66
					d39 = snap67
					d40 = snap68
					d42 = snap69
					d43 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d48 = snap74
					d49 = snap75
					d52 = snap76
					ps78 := PhiState{General: true}
					ps78.OverlayValues = make([]JITValueDesc, 53)
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[7] = d7
					ps78.OverlayValues[8] = d8
					ps78.OverlayValues[11] = d11
					ps78.OverlayValues[22] = d22
					ps78.OverlayValues[32] = d32
					ps78.OverlayValues[33] = d33
					ps78.OverlayValues[34] = d34
					ps78.OverlayValues[35] = d35
					ps78.OverlayValues[37] = d37
					ps78.OverlayValues[38] = d38
					ps78.OverlayValues[39] = d39
					ps78.OverlayValues[40] = d40
					ps78.OverlayValues[42] = d42
					ps78.OverlayValues[43] = d43
					ps78.OverlayValues[45] = d45
					ps78.OverlayValues[46] = d46
					ps78.OverlayValues[47] = d47
					ps78.OverlayValues[48] = d48
					ps78.OverlayValues[49] = d49
					ps78.OverlayValues[52] = d52
					ps79 := PhiState{General: true}
					ps79.OverlayValues = make([]JITValueDesc, 53)
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[6] = d6
					ps79.OverlayValues[7] = d7
					ps79.OverlayValues[8] = d8
					ps79.OverlayValues[11] = d11
					ps79.OverlayValues[22] = d22
					ps79.OverlayValues[32] = d32
					ps79.OverlayValues[33] = d33
					ps79.OverlayValues[34] = d34
					ps79.OverlayValues[35] = d35
					ps79.OverlayValues[37] = d37
					ps79.OverlayValues[38] = d38
					ps79.OverlayValues[39] = d39
					ps79.OverlayValues[40] = d40
					ps79.OverlayValues[42] = d42
					ps79.OverlayValues[43] = d43
					ps79.OverlayValues[45] = d45
					ps79.OverlayValues[46] = d46
					ps79.OverlayValues[47] = d47
					ps79.OverlayValues[48] = d48
					ps79.OverlayValues[49] = d49
					ps79.OverlayValues[52] = d52
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d7
					snap85 := d8
					snap86 := d11
					snap87 := d22
					snap88 := d32
					snap89 := d33
					snap90 := d34
					snap91 := d35
					snap92 := d37
					snap93 := d38
					snap94 := d39
					snap95 := d40
					snap96 := d42
					snap97 := d43
					snap98 := d45
					snap99 := d46
					snap100 := d47
					snap101 := d48
					snap102 := d49
					snap103 := d52
					alloc104 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps79)
					}
					ctx.RestoreAllocState(alloc104)
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d7 = snap84
					d8 = snap85
					d11 = snap86
					d22 = snap87
					d32 = snap88
					d33 = snap89
					d34 = snap90
					d35 = snap91
					d37 = snap92
					d38 = snap93
					d39 = snap94
					d40 = snap95
					d42 = snap96
					d43 = snap97
					d45 = snap98
					d46 = snap99
					d47 = snap100
					d48 = snap101
					d49 = snap102
					d52 = snap103
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps78)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d47)
					d105 = ctx.EmitLoadScalarSliceElement(&d39, &d47, 8, JITTypeUnknown)
					ctx.StabilizeDescForControlFlow(&d105)
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d106 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d106.NoHeapPointer = false
					ctx.BindReg(d106.Reg, &d106)
					ctx.BindReg(d106.Reg2, &d106)
					ctx.StabilizeDescForControlFlow(&d106)
					d107 = d3
					ctx.EnsureDesc(&d107)
					if d107.Loc != LocImm && d107.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d107.Loc == LocImm {
						if d107.Imm.Bool() {
							if ps.General {
								ctx.SyncDesc(&d106)
								if d106.Loc == LocReg {
									ctx.ProtectReg(d106.Reg)
								} else if d106.Loc == LocRegPair {
									ctx.ProtectReg(d106.Reg)
									ctx.ProtectReg(d106.Reg2)
								}
								d108 = d106
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
								if d106.Loc == LocReg {
									ctx.UnprotectReg(d106.Reg)
								} else if d106.Loc == LocRegPair {
									ctx.UnprotectReg(d106.Reg)
									ctx.UnprotectReg(d106.Reg2)
								}
							}
							ps109 := PhiState{General: ps.General}
							ps109.OverlayValues = make([]JITValueDesc, 109)
							ps109.OverlayValues[3] = d3
							ps109.OverlayValues[4] = d4
							ps109.OverlayValues[5] = d5
							ps109.OverlayValues[6] = d6
							ps109.OverlayValues[7] = d7
							ps109.OverlayValues[8] = d8
							ps109.OverlayValues[11] = d11
							ps109.OverlayValues[22] = d22
							ps109.OverlayValues[32] = d32
							ps109.OverlayValues[33] = d33
							ps109.OverlayValues[34] = d34
							ps109.OverlayValues[35] = d35
							ps109.OverlayValues[37] = d37
							ps109.OverlayValues[38] = d38
							ps109.OverlayValues[39] = d39
							ps109.OverlayValues[40] = d40
							ps109.OverlayValues[42] = d42
							ps109.OverlayValues[43] = d43
							ps109.OverlayValues[45] = d45
							ps109.OverlayValues[46] = d46
							ps109.OverlayValues[47] = d47
							ps109.OverlayValues[48] = d48
							ps109.OverlayValues[49] = d49
							ps109.OverlayValues[52] = d52
							ps109.OverlayValues[105] = d105
							ps109.OverlayValues[106] = d106
							ps109.OverlayValues[107] = d107
							ps109.OverlayValues[108] = d108
							ps109.PhiValues = make([]JITValueDesc, 1)
							d110 = d106
							ps109.PhiValues[0] = d110
							return bbs[7].RenderPS(ps109)
						}
						if ps.General {
						}
						ps111 := PhiState{General: ps.General}
						ps111.OverlayValues = make([]JITValueDesc, 111)
						ps111.OverlayValues[3] = d3
						ps111.OverlayValues[4] = d4
						ps111.OverlayValues[5] = d5
						ps111.OverlayValues[6] = d6
						ps111.OverlayValues[7] = d7
						ps111.OverlayValues[8] = d8
						ps111.OverlayValues[11] = d11
						ps111.OverlayValues[22] = d22
						ps111.OverlayValues[32] = d32
						ps111.OverlayValues[33] = d33
						ps111.OverlayValues[34] = d34
						ps111.OverlayValues[35] = d35
						ps111.OverlayValues[37] = d37
						ps111.OverlayValues[38] = d38
						ps111.OverlayValues[39] = d39
						ps111.OverlayValues[40] = d40
						ps111.OverlayValues[42] = d42
						ps111.OverlayValues[43] = d43
						ps111.OverlayValues[45] = d45
						ps111.OverlayValues[46] = d46
						ps111.OverlayValues[47] = d47
						ps111.OverlayValues[48] = d48
						ps111.OverlayValues[49] = d49
						ps111.OverlayValues[52] = d52
						ps111.OverlayValues[105] = d105
						ps111.OverlayValues[106] = d106
						ps111.OverlayValues[107] = d107
						ps111.OverlayValues[108] = d108
						ps111.OverlayValues[110] = d110
						return bbs[8].RenderPS(ps111)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d107.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl9)
					snap112 := d3
					snap113 := d4
					snap114 := d5
					snap115 := d6
					snap116 := d7
					snap117 := d8
					snap118 := d11
					snap119 := d22
					snap120 := d32
					snap121 := d33
					snap122 := d34
					snap123 := d35
					snap124 := d37
					snap125 := d38
					snap126 := d39
					snap127 := d40
					snap128 := d42
					snap129 := d43
					snap130 := d45
					snap131 := d46
					snap132 := d47
					snap133 := d48
					snap134 := d49
					snap135 := d52
					snap136 := d105
					snap137 := d106
					snap138 := d107
					snap139 := d108
					snap140 := d110
					alloc141 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d106)
					if d106.Loc == LocReg {
						ctx.ProtectReg(d106.Reg)
					} else if d106.Loc == LocRegPair {
						ctx.ProtectReg(d106.Reg)
						ctx.ProtectReg(d106.Reg2)
					}
					d142 = d106
					if d142.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d142)
					if d142.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d142, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d142.Loc == LocInputPair {
						ctx.EnsureDesc(&d142)
						ctx.EmitStoreScmerToStack(d142, int32(bbs[7].PhiBase)+int32(0))
					} else if d142.Loc == LocRegPair || d142.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d142, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d142)
						ctx.EmitStoreToStack(d142, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d106.Loc == LocReg {
						ctx.UnprotectReg(d106.Reg)
					} else if d106.Loc == LocRegPair {
						ctx.UnprotectReg(d106.Reg)
						ctx.UnprotectReg(d106.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc141)
					d3 = snap112
					d4 = snap113
					d5 = snap114
					d6 = snap115
					d7 = snap116
					d8 = snap117
					d11 = snap118
					d22 = snap119
					d32 = snap120
					d33 = snap121
					d34 = snap122
					d35 = snap123
					d37 = snap124
					d38 = snap125
					d39 = snap126
					d40 = snap127
					d42 = snap128
					d43 = snap129
					d45 = snap130
					d46 = snap131
					d47 = snap132
					d48 = snap133
					d49 = snap134
					d52 = snap135
					d105 = snap136
					d106 = snap137
					d107 = snap138
					d108 = snap139
					d110 = snap140
					ctx.RestoreAllocState(alloc141)
					d3 = snap112
					d4 = snap113
					d5 = snap114
					d6 = snap115
					d7 = snap116
					d8 = snap117
					d11 = snap118
					d22 = snap119
					d32 = snap120
					d33 = snap121
					d34 = snap122
					d35 = snap123
					d37 = snap124
					d38 = snap125
					d39 = snap126
					d40 = snap127
					d42 = snap128
					d43 = snap129
					d45 = snap130
					d46 = snap131
					d47 = snap132
					d48 = snap133
					d49 = snap134
					d52 = snap135
					d105 = snap136
					d106 = snap137
					d107 = snap138
					d108 = snap139
					d110 = snap140
					ps143 := PhiState{General: true}
					ps143.OverlayValues = make([]JITValueDesc, 143)
					ps143.OverlayValues[3] = d3
					ps143.OverlayValues[4] = d4
					ps143.OverlayValues[5] = d5
					ps143.OverlayValues[6] = d6
					ps143.OverlayValues[7] = d7
					ps143.OverlayValues[8] = d8
					ps143.OverlayValues[11] = d11
					ps143.OverlayValues[22] = d22
					ps143.OverlayValues[32] = d32
					ps143.OverlayValues[33] = d33
					ps143.OverlayValues[34] = d34
					ps143.OverlayValues[35] = d35
					ps143.OverlayValues[37] = d37
					ps143.OverlayValues[38] = d38
					ps143.OverlayValues[39] = d39
					ps143.OverlayValues[40] = d40
					ps143.OverlayValues[42] = d42
					ps143.OverlayValues[43] = d43
					ps143.OverlayValues[45] = d45
					ps143.OverlayValues[46] = d46
					ps143.OverlayValues[47] = d47
					ps143.OverlayValues[48] = d48
					ps143.OverlayValues[49] = d49
					ps143.OverlayValues[52] = d52
					ps143.OverlayValues[105] = d105
					ps143.OverlayValues[106] = d106
					ps143.OverlayValues[107] = d107
					ps143.OverlayValues[108] = d108
					ps143.OverlayValues[110] = d110
					ps143.OverlayValues[142] = d142
					ps143.PhiValues = make([]JITValueDesc, 1)
					d145 = d106
					ps143.PhiValues[0] = d145
					ps144 := PhiState{General: true}
					ps144.OverlayValues = make([]JITValueDesc, 146)
					ps144.OverlayValues[3] = d3
					ps144.OverlayValues[4] = d4
					ps144.OverlayValues[5] = d5
					ps144.OverlayValues[6] = d6
					ps144.OverlayValues[7] = d7
					ps144.OverlayValues[8] = d8
					ps144.OverlayValues[11] = d11
					ps144.OverlayValues[22] = d22
					ps144.OverlayValues[32] = d32
					ps144.OverlayValues[33] = d33
					ps144.OverlayValues[34] = d34
					ps144.OverlayValues[35] = d35
					ps144.OverlayValues[37] = d37
					ps144.OverlayValues[38] = d38
					ps144.OverlayValues[39] = d39
					ps144.OverlayValues[40] = d40
					ps144.OverlayValues[42] = d42
					ps144.OverlayValues[43] = d43
					ps144.OverlayValues[45] = d45
					ps144.OverlayValues[46] = d46
					ps144.OverlayValues[47] = d47
					ps144.OverlayValues[48] = d48
					ps144.OverlayValues[49] = d49
					ps144.OverlayValues[52] = d52
					ps144.OverlayValues[105] = d105
					ps144.OverlayValues[106] = d106
					ps144.OverlayValues[107] = d107
					ps144.OverlayValues[108] = d108
					ps144.OverlayValues[110] = d110
					ps144.OverlayValues[142] = d142
					ps144.OverlayValues[145] = d145
					snap146 := d3
					snap147 := d4
					snap148 := d5
					snap149 := d6
					snap150 := d7
					snap151 := d8
					snap152 := d11
					snap153 := d22
					snap154 := d32
					snap155 := d33
					snap156 := d34
					snap157 := d35
					snap158 := d37
					snap159 := d38
					snap160 := d39
					snap161 := d40
					snap162 := d42
					snap163 := d43
					snap164 := d45
					snap165 := d46
					snap166 := d47
					snap167 := d48
					snap168 := d49
					snap169 := d52
					snap170 := d105
					snap171 := d106
					snap172 := d107
					snap173 := d108
					snap174 := d110
					snap175 := d142
					snap176 := d145
					alloc177 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps143)
					}
					ctx.RestoreAllocState(alloc177)
					d3 = snap146
					d4 = snap147
					d5 = snap148
					d6 = snap149
					d7 = snap150
					d8 = snap151
					d11 = snap152
					d22 = snap153
					d32 = snap154
					d33 = snap155
					d34 = snap156
					d35 = snap157
					d37 = snap158
					d38 = snap159
					d39 = snap160
					d40 = snap161
					d42 = snap162
					d43 = snap163
					d45 = snap164
					d46 = snap165
					d47 = snap166
					d48 = snap167
					d49 = snap168
					d52 = snap169
					d105 = snap170
					d106 = snap171
					d107 = snap172
					d108 = snap173
					d110 = snap174
					d142 = snap175
					d145 = snap176
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps144)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d42)
					d178 = ctx.EmitNewSliceFromGoSlice(&d42)
					ctx.SyncDesc(&d178)
					if d178.Loc == LocRegPair || d178.Loc == LocStackPair || d178.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d178, &result)
						result.Type = d178.Type
					} else {
						switch d178.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d178)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d178)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d178)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d178, &result)
							result.Type = d178.Type
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					ctx.ReclaimUntrackedRegs()
					d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d179)
					ctx.EnsureDesc(&d180)
					var d182 JITValueDesc
					if d180.Loc == LocImm && d179.Loc == LocImm {
						d182 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d180.Imm.Int() - d179.Imm.Int())}
					} else {
						r5 := ctx.AllocReg()
						if d180.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d180.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d180.Reg)
						}
						if d179.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d179.Imm.Int()))
							ctx.EmitSubInt64(r5, RegR11)
						} else {
							ctx.EmitSubInt64(r5, d179.Reg)
						}
						d182 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d182)
					}
					var d183 JITValueDesc
					r6 := ctx.EmitSliceDataAfterLow(&d106, &d179, 1)
					d183 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
					ctx.BindReg(r6, &d183)
					ctx.BindReg(r6, &d183)
					var d184 JITValueDesc
					var r7 Reg
					var r8 Reg
					ctx.SyncDesc(&d183)
					ctx.EnsureDesc(&d183)
					if d183.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d183.Imm.Int()))
					} else {
						r7 = d183.Reg
					}
					ctx.ProtectReg(r7)
					ctx.SyncDesc(&d182)
					ctx.EnsureDesc(&d182)
					if d182.Loc == LocImm {
						r8 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r8, uint64(d182.Imm.Int()))
					} else {
						r8 = d182.Reg
					}
					ctx.ProtectReg(r8)
					ctx.UnprotectReg(r8)
					ctx.UnprotectReg(r7)
					d184 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d184)
					ctx.BindReg(r8, &d184)
					ctx.BindReg(r7, &d184)
					ctx.BindReg(r8, &d184)
					ctx.StabilizeDescForControlFlow(&d184)
					if ps.General {
						ctx.SyncDesc(&d184)
						if d184.Loc == LocReg {
							ctx.ProtectReg(d184.Reg)
						} else if d184.Loc == LocRegPair {
							ctx.ProtectReg(d184.Reg)
							ctx.ProtectReg(d184.Reg2)
						}
						d185 = d184
						if d185.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d185)
						if d185.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d185, int32(bbs[7].PhiBase)+int32(0), 2)
						} else if d185.Loc == LocInputPair {
							ctx.EnsureDesc(&d185)
							ctx.EmitStoreScmerToStack(d185, int32(bbs[7].PhiBase)+int32(0))
						} else if d185.Loc == LocRegPair || d185.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d185, int32(bbs[7].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d185)
							ctx.EmitStoreToStack(d185, int32(bbs[7].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
						}
						if d184.Loc == LocReg {
							ctx.UnprotectReg(d184.Reg)
						} else if d184.Loc == LocRegPair {
							ctx.UnprotectReg(d184.Reg)
							ctx.UnprotectReg(d184.Reg2)
						}
					}
					ps186 := PhiState{General: ps.General}
					ps186.OverlayValues = make([]JITValueDesc, 186)
					ps186.OverlayValues[3] = d3
					ps186.OverlayValues[4] = d4
					ps186.OverlayValues[5] = d5
					ps186.OverlayValues[6] = d6
					ps186.OverlayValues[7] = d7
					ps186.OverlayValues[8] = d8
					ps186.OverlayValues[11] = d11
					ps186.OverlayValues[22] = d22
					ps186.OverlayValues[32] = d32
					ps186.OverlayValues[33] = d33
					ps186.OverlayValues[34] = d34
					ps186.OverlayValues[35] = d35
					ps186.OverlayValues[37] = d37
					ps186.OverlayValues[38] = d38
					ps186.OverlayValues[39] = d39
					ps186.OverlayValues[40] = d40
					ps186.OverlayValues[42] = d42
					ps186.OverlayValues[43] = d43
					ps186.OverlayValues[45] = d45
					ps186.OverlayValues[46] = d46
					ps186.OverlayValues[47] = d47
					ps186.OverlayValues[48] = d48
					ps186.OverlayValues[49] = d49
					ps186.OverlayValues[52] = d52
					ps186.OverlayValues[105] = d105
					ps186.OverlayValues[106] = d106
					ps186.OverlayValues[107] = d107
					ps186.OverlayValues[108] = d108
					ps186.OverlayValues[110] = d110
					ps186.OverlayValues[142] = d142
					ps186.OverlayValues[145] = d145
					ps186.OverlayValues[178] = d178
					ps186.OverlayValues[179] = d179
					ps186.OverlayValues[180] = d180
					ps186.OverlayValues[181] = d181
					ps186.OverlayValues[182] = d182
					ps186.OverlayValues[183] = d183
					ps186.OverlayValues[184] = d184
					ps186.OverlayValues[185] = d185
					ps186.PhiValues = make([]JITValueDesc, 1)
					d187 = d184
					ps186.PhiValues[0] = d187
					if ps186.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps186)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d188 := ps.PhiValues[0]
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreScmerToStack(d188, int32(bbs[7].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d5 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d105)
					d189 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d105}, 2)
					d189.NoHeapPointer = false
					ctx.BindReg(d189.Reg, &d189)
					ctx.BindReg(d189.Reg2, &d189)
					stackArray190 = ctx.AllocStack(int32(256))
					_ = stackArray190
					d191 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.SyncDesc(&d191)
					ctx.EmitStoreScmerToStack(d191, int32(stackArray190)+int32(0))
					var d192 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 0
						r9 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r9, fieldAddr)
						d192 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d192)
					} else {
						off := int32(0)
						baseReg := d105.Reg
						r10 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r10, baseReg, off)
						d192 = JITValueDesc{Loc: LocReg, Reg: r10}
						ctx.BindReg(r10, &d192)
					}
					ctx.EnsureDesc(&d192)
					ctx.EnsureDesc(&d192)
					var d193 JITValueDesc
					if d192.Loc == LocImm {
						d193 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d192.Imm.Int()))))}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegReg(r11, d192.Reg)
						d193 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d193)
					}
					ctx.FreeDesc(&d192)
					ctx.EnsureDesc(&d193)
					ctx.SyncDesc(&d193)
					ctx.EnsureDesc(&d193)
					ctx.EmitStoreTypedScmerToStack(d193, tagInt, int32(stackArray190)+int32(16))
					d194 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.SyncDesc(&d194)
					ctx.EmitStoreScmerToStack(d194, int32(stackArray190)+int32(32))
					var d195 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 8
						r12 := ctx.AllocReg()
						r13 := ctx.AllocRegExcept(r12)
						r14 := ctx.AllocRegExcept(r12, r13)
						ctx.EmitMovRegMem64(r12, fieldAddr)
						ctx.EmitMovRegMem64(r13, fieldAddr+8)
						ctx.EmitMovRegMem64(r14, fieldAddr+16)
						d195 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
						ctx.BindReg(r12, &d195)
						ctx.BindReg(r13, &d195)
						ctx.BindReg(r14, &d195)
					} else {
						off := int32(8)
						baseReg := d105.Reg
						r15 := ctx.AllocRegExcept(baseReg)
						r16 := ctx.AllocRegExcept(baseReg, r15)
						r17 := ctx.AllocRegExcept(baseReg, r15, r16)
						ctx.EmitMovRegMem(r15, baseReg, off)
						ctx.EmitMovRegMem(r16, baseReg, off+8)
						ctx.EmitMovRegMem(r17, baseReg, off+16)
						d195 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d195)
						ctx.BindReg(r16, &d195)
						ctx.BindReg(r17, &d195)
					}
					ctx.EnsureDesc(&d195)
					ctx.SyncDesc(&d195)
					ctx.EmitStoreScmerToStack(d195, int32(stackArray190)+int32(48))
					d196 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.SyncDesc(&d196)
					ctx.EmitStoreScmerToStack(d196, int32(stackArray190)+int32(64))
					var d197 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 24
						r18 := ctx.AllocReg()
						r19 := ctx.AllocRegExcept(r18)
						r20 := ctx.AllocRegExcept(r18, r19)
						ctx.EmitMovRegMem64(r18, fieldAddr)
						ctx.EmitMovRegMem64(r19, fieldAddr+8)
						ctx.EmitMovRegMem64(r20, fieldAddr+16)
						d197 = JITValueDesc{Loc: LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
						ctx.BindReg(r18, &d197)
						ctx.BindReg(r19, &d197)
						ctx.BindReg(r20, &d197)
					} else {
						off := int32(24)
						baseReg := d105.Reg
						r21 := ctx.AllocRegExcept(baseReg)
						r22 := ctx.AllocRegExcept(baseReg, r21)
						r23 := ctx.AllocRegExcept(baseReg, r21, r22)
						ctx.EmitMovRegMem(r21, baseReg, off)
						ctx.EmitMovRegMem(r22, baseReg, off+8)
						ctx.EmitMovRegMem(r23, baseReg, off+16)
						d197 = JITValueDesc{Loc: LocRegTriple, Reg: r21, Reg2: r22, Reg3: r23}
						ctx.BindReg(r21, &d197)
						ctx.BindReg(r22, &d197)
						ctx.BindReg(r23, &d197)
					}
					ctx.EnsureDesc(&d197)
					ctx.SyncDesc(&d197)
					ctx.EmitStoreScmerToStack(d197, int32(stackArray190)+int32(80))
					d198 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.SyncDesc(&d198)
					ctx.EmitStoreScmerToStack(d198, int32(stackArray190)+int32(96))
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d199 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d199.NoHeapPointer = false
					ctx.BindReg(d199.Reg, &d199)
					ctx.BindReg(d199.Reg2, &d199)
					ctx.EnsureDesc(&d199)
					ctx.SyncDesc(&d199)
					ctx.EmitStoreScmerToStack(d199, int32(stackArray190)+int32(112))
					d200 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.SyncDesc(&d200)
					ctx.EmitStoreScmerToStack(d200, int32(stackArray190)+int32(128))
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d201 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d201.NoHeapPointer = false
					ctx.BindReg(d201.Reg, &d201)
					ctx.BindReg(d201.Reg2, &d201)
					ctx.EnsureDesc(&d201)
					ctx.SyncDesc(&d201)
					ctx.EmitStoreScmerToStack(d201, int32(stackArray190)+int32(144))
					d202 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.SyncDesc(&d202)
					ctx.EmitStoreScmerToStack(d202, int32(stackArray190)+int32(160))
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d105)
					d203 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d105}, 1)
					d203.NoHeapPointer = true
					ctx.BindReg(d203.Reg, &d203)
					ctx.EnsureDesc(&d203)
					ctx.SyncDesc(&d203)
					ctx.EnsureDesc(&d203)
					ctx.EmitStoreTypedScmerToStack(d203, tagInt, int32(stackArray190)+int32(176))
					d204 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.SyncDesc(&d204)
					ctx.EmitStoreScmerToStack(d204, int32(stackArray190)+int32(192))
					ctx.EnsureDesc(&d189)
					ctx.SyncDesc(&d189)
					ctx.EmitStoreScmerToStack(d189, int32(stackArray190)+int32(208))
					d205 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.SyncDesc(&d205)
					ctx.EmitStoreScmerToStack(d205, int32(stackArray190)+int32(224))
					ctx.EnsureDesc(&d5)
					ctx.SyncDesc(&d5)
					ctx.EmitStoreScmerToStack(d5, int32(stackArray190)+int32(240))
					d206 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d206
					r24 := ctx.AllocReg()
					r25 := ctx.AllocRegExcept(r24)
					r26 := ctx.AllocRegExcept(r24, r25)
					d207 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r24, Reg2: r25, Reg3: r26}
					ctx.BindReg(r24, &d207)
					ctx.BindReg(r25, &d207)
					ctx.BindReg(r26, &d207)
					ctx.BindReg(r24, &d207)
					ctx.BindReg(r25, &d207)
					ctx.BindReg(r26, &d207)
					ctx.EmitLeaRegMem(d207.Reg, ctx.StackReg, int32(stackArray190))
					ctx.EmitMovRegImm64(d207.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d207.Reg3, uint64(16))
					callResults208 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d207}, []uint8{2}, []uint8{1})
					d209 = callResults208[0]
					ctx.EnsureDesc(&d47)
					ctx.SyncDesc(&d209)
					d210 = d42
					d210.ID = 0
					d211 = d47
					d211.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d210, &d211, &d209, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d47)
						d211 = d47
						d211.ID = 0
						ctx.EmitStoreScmerSliceElement(&d210, &d211, &d209, int32(16))
					}
					ctx.FreeDesc(&d211)
					ctx.FreeDesc(&d209)
					if ps.General {
						ctx.SyncDesc(&d47)
						if d47.Loc == LocReg {
							ctx.ProtectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.ProtectReg(d47.Reg)
							ctx.ProtectReg(d47.Reg2)
						}
						d212 = d47
						if d212.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d212)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d212)
						} else {
							ctx.EmitStoreToStack(d212, int32(bbs[3].PhiBase)+int32(0))
						}
						if d47.Loc == LocReg {
							ctx.UnprotectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.UnprotectReg(d47.Reg)
							ctx.UnprotectReg(d47.Reg2)
						}
					}
					ps213 := PhiState{General: ps.General}
					ps213.OverlayValues = make([]JITValueDesc, 213)
					ps213.OverlayValues[3] = d3
					ps213.OverlayValues[4] = d4
					ps213.OverlayValues[5] = d5
					ps213.OverlayValues[6] = d6
					ps213.OverlayValues[7] = d7
					ps213.OverlayValues[8] = d8
					ps213.OverlayValues[11] = d11
					ps213.OverlayValues[22] = d22
					ps213.OverlayValues[32] = d32
					ps213.OverlayValues[33] = d33
					ps213.OverlayValues[34] = d34
					ps213.OverlayValues[35] = d35
					ps213.OverlayValues[37] = d37
					ps213.OverlayValues[38] = d38
					ps213.OverlayValues[39] = d39
					ps213.OverlayValues[40] = d40
					ps213.OverlayValues[42] = d42
					ps213.OverlayValues[43] = d43
					ps213.OverlayValues[45] = d45
					ps213.OverlayValues[46] = d46
					ps213.OverlayValues[47] = d47
					ps213.OverlayValues[48] = d48
					ps213.OverlayValues[49] = d49
					ps213.OverlayValues[52] = d52
					ps213.OverlayValues[105] = d105
					ps213.OverlayValues[106] = d106
					ps213.OverlayValues[107] = d107
					ps213.OverlayValues[108] = d108
					ps213.OverlayValues[110] = d110
					ps213.OverlayValues[142] = d142
					ps213.OverlayValues[145] = d145
					ps213.OverlayValues[178] = d178
					ps213.OverlayValues[179] = d179
					ps213.OverlayValues[180] = d180
					ps213.OverlayValues[181] = d181
					ps213.OverlayValues[182] = d182
					ps213.OverlayValues[183] = d183
					ps213.OverlayValues[184] = d184
					ps213.OverlayValues[185] = d185
					ps213.OverlayValues[187] = d187
					ps213.OverlayValues[188] = d188
					ps213.OverlayValues[189] = d189
					ps213.OverlayValues[191] = d191
					ps213.OverlayValues[192] = d192
					ps213.OverlayValues[193] = d193
					ps213.OverlayValues[194] = d194
					ps213.OverlayValues[195] = d195
					ps213.OverlayValues[196] = d196
					ps213.OverlayValues[197] = d197
					ps213.OverlayValues[198] = d198
					ps213.OverlayValues[199] = d199
					ps213.OverlayValues[200] = d200
					ps213.OverlayValues[201] = d201
					ps213.OverlayValues[202] = d202
					ps213.OverlayValues[203] = d203
					ps213.OverlayValues[204] = d204
					ps213.OverlayValues[205] = d205
					ps213.OverlayValues[206] = d206
					ps213.OverlayValues[207] = d207
					ps213.OverlayValues[209] = d209
					ps213.OverlayValues[210] = d210
					ps213.OverlayValues[211] = d211
					ps213.OverlayValues[212] = d212
					ps213.PhiValues = make([]JITValueDesc, 1)
					d214 = d47
					ps213.PhiValues[0] = d214
					if ps213.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps213)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
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
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != LocNone {
						d211 = ps.OverlayValues[211]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					ctx.ReclaimUntrackedRegs()
					var d215 JITValueDesc
					if d106.SliceSizeKnown {
						d215 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d106.KnownSliceLen))}
					} else if d106.Loc == LocImm {
						d215 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d106.Imm.String())))}
					} else if d106.Loc == LocStackTriple {
						d215 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d106.StackOff + 8, NoHeapPointer: true}
					} else if d106.Loc == LocStackPair {
						d215 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d106.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d106)
						if d106.Loc == LocRegPair || d106.Loc == LocRegTriple {
							d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d106.Reg2, ID: 0}
						} else if d106.Loc == LocReg {
							d215 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d106.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d215)
					var d216 JITValueDesc
					if d215.Loc == LocImm {
						d216 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d215.Imm.Int() > 100)}
					} else {
						r27 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d215.Reg, 100)
						d216 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedGreater}
						ctx.BindReg(r27, &d216)
					}
					ctx.FreeDesc(&d215)
					d217 = d216
					ctx.EnsureDesc(&d217)
					if d217.Loc != LocImm && d217.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d217.Loc == LocImm {
						if d217.Imm.Bool() {
							if ps.General {
							}
							ps218 := PhiState{General: ps.General}
							ps218.OverlayValues = make([]JITValueDesc, 218)
							ps218.OverlayValues[3] = d3
							ps218.OverlayValues[4] = d4
							ps218.OverlayValues[5] = d5
							ps218.OverlayValues[6] = d6
							ps218.OverlayValues[7] = d7
							ps218.OverlayValues[8] = d8
							ps218.OverlayValues[11] = d11
							ps218.OverlayValues[22] = d22
							ps218.OverlayValues[32] = d32
							ps218.OverlayValues[33] = d33
							ps218.OverlayValues[34] = d34
							ps218.OverlayValues[35] = d35
							ps218.OverlayValues[37] = d37
							ps218.OverlayValues[38] = d38
							ps218.OverlayValues[39] = d39
							ps218.OverlayValues[40] = d40
							ps218.OverlayValues[42] = d42
							ps218.OverlayValues[43] = d43
							ps218.OverlayValues[45] = d45
							ps218.OverlayValues[46] = d46
							ps218.OverlayValues[47] = d47
							ps218.OverlayValues[48] = d48
							ps218.OverlayValues[49] = d49
							ps218.OverlayValues[52] = d52
							ps218.OverlayValues[105] = d105
							ps218.OverlayValues[106] = d106
							ps218.OverlayValues[107] = d107
							ps218.OverlayValues[108] = d108
							ps218.OverlayValues[110] = d110
							ps218.OverlayValues[142] = d142
							ps218.OverlayValues[145] = d145
							ps218.OverlayValues[178] = d178
							ps218.OverlayValues[179] = d179
							ps218.OverlayValues[180] = d180
							ps218.OverlayValues[181] = d181
							ps218.OverlayValues[182] = d182
							ps218.OverlayValues[183] = d183
							ps218.OverlayValues[184] = d184
							ps218.OverlayValues[185] = d185
							ps218.OverlayValues[187] = d187
							ps218.OverlayValues[188] = d188
							ps218.OverlayValues[189] = d189
							ps218.OverlayValues[191] = d191
							ps218.OverlayValues[192] = d192
							ps218.OverlayValues[193] = d193
							ps218.OverlayValues[194] = d194
							ps218.OverlayValues[195] = d195
							ps218.OverlayValues[196] = d196
							ps218.OverlayValues[197] = d197
							ps218.OverlayValues[198] = d198
							ps218.OverlayValues[199] = d199
							ps218.OverlayValues[200] = d200
							ps218.OverlayValues[201] = d201
							ps218.OverlayValues[202] = d202
							ps218.OverlayValues[203] = d203
							ps218.OverlayValues[204] = d204
							ps218.OverlayValues[205] = d205
							ps218.OverlayValues[206] = d206
							ps218.OverlayValues[207] = d207
							ps218.OverlayValues[209] = d209
							ps218.OverlayValues[210] = d210
							ps218.OverlayValues[211] = d211
							ps218.OverlayValues[212] = d212
							ps218.OverlayValues[214] = d214
							ps218.OverlayValues[215] = d215
							ps218.OverlayValues[216] = d216
							ps218.OverlayValues[217] = d217
							return bbs[6].RenderPS(ps218)
						}
						if ps.General {
							ctx.SyncDesc(&d106)
							if d106.Loc == LocReg {
								ctx.ProtectReg(d106.Reg)
							} else if d106.Loc == LocRegPair {
								ctx.ProtectReg(d106.Reg)
								ctx.ProtectReg(d106.Reg2)
							}
							d219 = d106
							if d219.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d219)
							if d219.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d219, int32(bbs[7].PhiBase)+int32(0), 2)
							} else if d219.Loc == LocInputPair {
								ctx.EnsureDesc(&d219)
								ctx.EmitStoreScmerToStack(d219, int32(bbs[7].PhiBase)+int32(0))
							} else if d219.Loc == LocRegPair || d219.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d219, int32(bbs[7].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d219)
								ctx.EmitStoreToStack(d219, int32(bbs[7].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
							}
							if d106.Loc == LocReg {
								ctx.UnprotectReg(d106.Reg)
							} else if d106.Loc == LocRegPair {
								ctx.UnprotectReg(d106.Reg)
								ctx.UnprotectReg(d106.Reg2)
							}
						}
						ps220 := PhiState{General: ps.General}
						ps220.OverlayValues = make([]JITValueDesc, 220)
						ps220.OverlayValues[3] = d3
						ps220.OverlayValues[4] = d4
						ps220.OverlayValues[5] = d5
						ps220.OverlayValues[6] = d6
						ps220.OverlayValues[7] = d7
						ps220.OverlayValues[8] = d8
						ps220.OverlayValues[11] = d11
						ps220.OverlayValues[22] = d22
						ps220.OverlayValues[32] = d32
						ps220.OverlayValues[33] = d33
						ps220.OverlayValues[34] = d34
						ps220.OverlayValues[35] = d35
						ps220.OverlayValues[37] = d37
						ps220.OverlayValues[38] = d38
						ps220.OverlayValues[39] = d39
						ps220.OverlayValues[40] = d40
						ps220.OverlayValues[42] = d42
						ps220.OverlayValues[43] = d43
						ps220.OverlayValues[45] = d45
						ps220.OverlayValues[46] = d46
						ps220.OverlayValues[47] = d47
						ps220.OverlayValues[48] = d48
						ps220.OverlayValues[49] = d49
						ps220.OverlayValues[52] = d52
						ps220.OverlayValues[105] = d105
						ps220.OverlayValues[106] = d106
						ps220.OverlayValues[107] = d107
						ps220.OverlayValues[108] = d108
						ps220.OverlayValues[110] = d110
						ps220.OverlayValues[142] = d142
						ps220.OverlayValues[145] = d145
						ps220.OverlayValues[178] = d178
						ps220.OverlayValues[179] = d179
						ps220.OverlayValues[180] = d180
						ps220.OverlayValues[181] = d181
						ps220.OverlayValues[182] = d182
						ps220.OverlayValues[183] = d183
						ps220.OverlayValues[184] = d184
						ps220.OverlayValues[185] = d185
						ps220.OverlayValues[187] = d187
						ps220.OverlayValues[188] = d188
						ps220.OverlayValues[189] = d189
						ps220.OverlayValues[191] = d191
						ps220.OverlayValues[192] = d192
						ps220.OverlayValues[193] = d193
						ps220.OverlayValues[194] = d194
						ps220.OverlayValues[195] = d195
						ps220.OverlayValues[196] = d196
						ps220.OverlayValues[197] = d197
						ps220.OverlayValues[198] = d198
						ps220.OverlayValues[199] = d199
						ps220.OverlayValues[200] = d200
						ps220.OverlayValues[201] = d201
						ps220.OverlayValues[202] = d202
						ps220.OverlayValues[203] = d203
						ps220.OverlayValues[204] = d204
						ps220.OverlayValues[205] = d205
						ps220.OverlayValues[206] = d206
						ps220.OverlayValues[207] = d207
						ps220.OverlayValues[209] = d209
						ps220.OverlayValues[210] = d210
						ps220.OverlayValues[211] = d211
						ps220.OverlayValues[212] = d212
						ps220.OverlayValues[214] = d214
						ps220.OverlayValues[215] = d215
						ps220.OverlayValues[216] = d216
						ps220.OverlayValues[217] = d217
						ps220.OverlayValues[219] = d219
						ps220.PhiValues = make([]JITValueDesc, 1)
						d221 = d106
						ps220.PhiValues[0] = d221
						return bbs[7].RenderPS(ps220)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitJump(d217.Condition, lbl7)
					ctx.EmitJmp(lbl12)
					snap222 := d3
					snap223 := d4
					snap224 := d5
					snap225 := d6
					snap226 := d7
					snap227 := d8
					snap228 := d11
					snap229 := d22
					snap230 := d32
					snap231 := d33
					snap232 := d34
					snap233 := d35
					snap234 := d37
					snap235 := d38
					snap236 := d39
					snap237 := d40
					snap238 := d42
					snap239 := d43
					snap240 := d45
					snap241 := d46
					snap242 := d47
					snap243 := d48
					snap244 := d49
					snap245 := d52
					snap246 := d105
					snap247 := d106
					snap248 := d107
					snap249 := d108
					snap250 := d110
					snap251 := d142
					snap252 := d145
					snap253 := d178
					snap254 := d179
					snap255 := d180
					snap256 := d181
					snap257 := d182
					snap258 := d183
					snap259 := d184
					snap260 := d185
					snap261 := d187
					snap262 := d188
					snap263 := d189
					snap264 := d191
					snap265 := d192
					snap266 := d193
					snap267 := d194
					snap268 := d195
					snap269 := d196
					snap270 := d197
					snap271 := d198
					snap272 := d199
					snap273 := d200
					snap274 := d201
					snap275 := d202
					snap276 := d203
					snap277 := d204
					snap278 := d205
					snap279 := d206
					snap280 := d207
					snap281 := d209
					snap282 := d210
					snap283 := d211
					snap284 := d212
					snap285 := d214
					snap286 := d215
					snap287 := d216
					snap288 := d217
					snap289 := d219
					snap290 := d221
					alloc291 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc291)
					d3 = snap222
					d4 = snap223
					d5 = snap224
					d6 = snap225
					d7 = snap226
					d8 = snap227
					d11 = snap228
					d22 = snap229
					d32 = snap230
					d33 = snap231
					d34 = snap232
					d35 = snap233
					d37 = snap234
					d38 = snap235
					d39 = snap236
					d40 = snap237
					d42 = snap238
					d43 = snap239
					d45 = snap240
					d46 = snap241
					d47 = snap242
					d48 = snap243
					d49 = snap244
					d52 = snap245
					d105 = snap246
					d106 = snap247
					d107 = snap248
					d108 = snap249
					d110 = snap250
					d142 = snap251
					d145 = snap252
					d178 = snap253
					d179 = snap254
					d180 = snap255
					d181 = snap256
					d182 = snap257
					d183 = snap258
					d184 = snap259
					d185 = snap260
					d187 = snap261
					d188 = snap262
					d189 = snap263
					d191 = snap264
					d192 = snap265
					d193 = snap266
					d194 = snap267
					d195 = snap268
					d196 = snap269
					d197 = snap270
					d198 = snap271
					d199 = snap272
					d200 = snap273
					d201 = snap274
					d202 = snap275
					d203 = snap276
					d204 = snap277
					d205 = snap278
					d206 = snap279
					d207 = snap280
					d209 = snap281
					d210 = snap282
					d211 = snap283
					d212 = snap284
					d214 = snap285
					d215 = snap286
					d216 = snap287
					d217 = snap288
					d219 = snap289
					d221 = snap290
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d106)
					if d106.Loc == LocReg {
						ctx.ProtectReg(d106.Reg)
					} else if d106.Loc == LocRegPair {
						ctx.ProtectReg(d106.Reg)
						ctx.ProtectReg(d106.Reg2)
					}
					d292 = d106
					if d292.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d292)
					if d292.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d292, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d292.Loc == LocInputPair {
						ctx.EnsureDesc(&d292)
						ctx.EmitStoreScmerToStack(d292, int32(bbs[7].PhiBase)+int32(0))
					} else if d292.Loc == LocRegPair || d292.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d292, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d292)
						ctx.EmitStoreToStack(d292, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d106.Loc == LocReg {
						ctx.UnprotectReg(d106.Reg)
					} else if d106.Loc == LocRegPair {
						ctx.UnprotectReg(d106.Reg)
						ctx.UnprotectReg(d106.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc291)
					d3 = snap222
					d4 = snap223
					d5 = snap224
					d6 = snap225
					d7 = snap226
					d8 = snap227
					d11 = snap228
					d22 = snap229
					d32 = snap230
					d33 = snap231
					d34 = snap232
					d35 = snap233
					d37 = snap234
					d38 = snap235
					d39 = snap236
					d40 = snap237
					d42 = snap238
					d43 = snap239
					d45 = snap240
					d46 = snap241
					d47 = snap242
					d48 = snap243
					d49 = snap244
					d52 = snap245
					d105 = snap246
					d106 = snap247
					d107 = snap248
					d108 = snap249
					d110 = snap250
					d142 = snap251
					d145 = snap252
					d178 = snap253
					d179 = snap254
					d180 = snap255
					d181 = snap256
					d182 = snap257
					d183 = snap258
					d184 = snap259
					d185 = snap260
					d187 = snap261
					d188 = snap262
					d189 = snap263
					d191 = snap264
					d192 = snap265
					d193 = snap266
					d194 = snap267
					d195 = snap268
					d196 = snap269
					d197 = snap270
					d198 = snap271
					d199 = snap272
					d200 = snap273
					d201 = snap274
					d202 = snap275
					d203 = snap276
					d204 = snap277
					d205 = snap278
					d206 = snap279
					d207 = snap280
					d209 = snap281
					d210 = snap282
					d211 = snap283
					d212 = snap284
					d214 = snap285
					d215 = snap286
					d216 = snap287
					d217 = snap288
					d219 = snap289
					d221 = snap290
					ps293 := PhiState{General: true}
					ps293.OverlayValues = make([]JITValueDesc, 293)
					ps293.OverlayValues[3] = d3
					ps293.OverlayValues[4] = d4
					ps293.OverlayValues[5] = d5
					ps293.OverlayValues[6] = d6
					ps293.OverlayValues[7] = d7
					ps293.OverlayValues[8] = d8
					ps293.OverlayValues[11] = d11
					ps293.OverlayValues[22] = d22
					ps293.OverlayValues[32] = d32
					ps293.OverlayValues[33] = d33
					ps293.OverlayValues[34] = d34
					ps293.OverlayValues[35] = d35
					ps293.OverlayValues[37] = d37
					ps293.OverlayValues[38] = d38
					ps293.OverlayValues[39] = d39
					ps293.OverlayValues[40] = d40
					ps293.OverlayValues[42] = d42
					ps293.OverlayValues[43] = d43
					ps293.OverlayValues[45] = d45
					ps293.OverlayValues[46] = d46
					ps293.OverlayValues[47] = d47
					ps293.OverlayValues[48] = d48
					ps293.OverlayValues[49] = d49
					ps293.OverlayValues[52] = d52
					ps293.OverlayValues[105] = d105
					ps293.OverlayValues[106] = d106
					ps293.OverlayValues[107] = d107
					ps293.OverlayValues[108] = d108
					ps293.OverlayValues[110] = d110
					ps293.OverlayValues[142] = d142
					ps293.OverlayValues[145] = d145
					ps293.OverlayValues[178] = d178
					ps293.OverlayValues[179] = d179
					ps293.OverlayValues[180] = d180
					ps293.OverlayValues[181] = d181
					ps293.OverlayValues[182] = d182
					ps293.OverlayValues[183] = d183
					ps293.OverlayValues[184] = d184
					ps293.OverlayValues[185] = d185
					ps293.OverlayValues[187] = d187
					ps293.OverlayValues[188] = d188
					ps293.OverlayValues[189] = d189
					ps293.OverlayValues[191] = d191
					ps293.OverlayValues[192] = d192
					ps293.OverlayValues[193] = d193
					ps293.OverlayValues[194] = d194
					ps293.OverlayValues[195] = d195
					ps293.OverlayValues[196] = d196
					ps293.OverlayValues[197] = d197
					ps293.OverlayValues[198] = d198
					ps293.OverlayValues[199] = d199
					ps293.OverlayValues[200] = d200
					ps293.OverlayValues[201] = d201
					ps293.OverlayValues[202] = d202
					ps293.OverlayValues[203] = d203
					ps293.OverlayValues[204] = d204
					ps293.OverlayValues[205] = d205
					ps293.OverlayValues[206] = d206
					ps293.OverlayValues[207] = d207
					ps293.OverlayValues[209] = d209
					ps293.OverlayValues[210] = d210
					ps293.OverlayValues[211] = d211
					ps293.OverlayValues[212] = d212
					ps293.OverlayValues[214] = d214
					ps293.OverlayValues[215] = d215
					ps293.OverlayValues[216] = d216
					ps293.OverlayValues[217] = d217
					ps293.OverlayValues[219] = d219
					ps293.OverlayValues[221] = d221
					ps293.OverlayValues[292] = d292
					ps294 := PhiState{General: true}
					ps294.OverlayValues = make([]JITValueDesc, 293)
					ps294.OverlayValues[3] = d3
					ps294.OverlayValues[4] = d4
					ps294.OverlayValues[5] = d5
					ps294.OverlayValues[6] = d6
					ps294.OverlayValues[7] = d7
					ps294.OverlayValues[8] = d8
					ps294.OverlayValues[11] = d11
					ps294.OverlayValues[22] = d22
					ps294.OverlayValues[32] = d32
					ps294.OverlayValues[33] = d33
					ps294.OverlayValues[34] = d34
					ps294.OverlayValues[35] = d35
					ps294.OverlayValues[37] = d37
					ps294.OverlayValues[38] = d38
					ps294.OverlayValues[39] = d39
					ps294.OverlayValues[40] = d40
					ps294.OverlayValues[42] = d42
					ps294.OverlayValues[43] = d43
					ps294.OverlayValues[45] = d45
					ps294.OverlayValues[46] = d46
					ps294.OverlayValues[47] = d47
					ps294.OverlayValues[48] = d48
					ps294.OverlayValues[49] = d49
					ps294.OverlayValues[52] = d52
					ps294.OverlayValues[105] = d105
					ps294.OverlayValues[106] = d106
					ps294.OverlayValues[107] = d107
					ps294.OverlayValues[108] = d108
					ps294.OverlayValues[110] = d110
					ps294.OverlayValues[142] = d142
					ps294.OverlayValues[145] = d145
					ps294.OverlayValues[178] = d178
					ps294.OverlayValues[179] = d179
					ps294.OverlayValues[180] = d180
					ps294.OverlayValues[181] = d181
					ps294.OverlayValues[182] = d182
					ps294.OverlayValues[183] = d183
					ps294.OverlayValues[184] = d184
					ps294.OverlayValues[185] = d185
					ps294.OverlayValues[187] = d187
					ps294.OverlayValues[188] = d188
					ps294.OverlayValues[189] = d189
					ps294.OverlayValues[191] = d191
					ps294.OverlayValues[192] = d192
					ps294.OverlayValues[193] = d193
					ps294.OverlayValues[194] = d194
					ps294.OverlayValues[195] = d195
					ps294.OverlayValues[196] = d196
					ps294.OverlayValues[197] = d197
					ps294.OverlayValues[198] = d198
					ps294.OverlayValues[199] = d199
					ps294.OverlayValues[200] = d200
					ps294.OverlayValues[201] = d201
					ps294.OverlayValues[202] = d202
					ps294.OverlayValues[203] = d203
					ps294.OverlayValues[204] = d204
					ps294.OverlayValues[205] = d205
					ps294.OverlayValues[206] = d206
					ps294.OverlayValues[207] = d207
					ps294.OverlayValues[209] = d209
					ps294.OverlayValues[210] = d210
					ps294.OverlayValues[211] = d211
					ps294.OverlayValues[212] = d212
					ps294.OverlayValues[214] = d214
					ps294.OverlayValues[215] = d215
					ps294.OverlayValues[216] = d216
					ps294.OverlayValues[217] = d217
					ps294.OverlayValues[219] = d219
					ps294.OverlayValues[221] = d221
					ps294.OverlayValues[292] = d292
					ps294.PhiValues = make([]JITValueDesc, 1)
					d295 = d106
					ps294.PhiValues[0] = d295
					snap296 := d3
					snap297 := d4
					snap298 := d5
					snap299 := d6
					snap300 := d7
					snap301 := d8
					snap302 := d11
					snap303 := d22
					snap304 := d32
					snap305 := d33
					snap306 := d34
					snap307 := d35
					snap308 := d37
					snap309 := d38
					snap310 := d39
					snap311 := d40
					snap312 := d42
					snap313 := d43
					snap314 := d45
					snap315 := d46
					snap316 := d47
					snap317 := d48
					snap318 := d49
					snap319 := d52
					snap320 := d105
					snap321 := d106
					snap322 := d107
					snap323 := d108
					snap324 := d110
					snap325 := d142
					snap326 := d145
					snap327 := d178
					snap328 := d179
					snap329 := d180
					snap330 := d181
					snap331 := d182
					snap332 := d183
					snap333 := d184
					snap334 := d185
					snap335 := d187
					snap336 := d188
					snap337 := d189
					snap338 := d191
					snap339 := d192
					snap340 := d193
					snap341 := d194
					snap342 := d195
					snap343 := d196
					snap344 := d197
					snap345 := d198
					snap346 := d199
					snap347 := d200
					snap348 := d201
					snap349 := d202
					snap350 := d203
					snap351 := d204
					snap352 := d205
					snap353 := d206
					snap354 := d207
					snap355 := d209
					snap356 := d210
					snap357 := d211
					snap358 := d212
					snap359 := d214
					snap360 := d215
					snap361 := d216
					snap362 := d217
					snap363 := d219
					snap364 := d221
					snap365 := d292
					snap366 := d295
					alloc367 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps294)
					}
					ctx.RestoreAllocState(alloc367)
					d3 = snap296
					d4 = snap297
					d5 = snap298
					d6 = snap299
					d7 = snap300
					d8 = snap301
					d11 = snap302
					d22 = snap303
					d32 = snap304
					d33 = snap305
					d34 = snap306
					d35 = snap307
					d37 = snap308
					d38 = snap309
					d39 = snap310
					d40 = snap311
					d42 = snap312
					d43 = snap313
					d45 = snap314
					d46 = snap315
					d47 = snap316
					d48 = snap317
					d49 = snap318
					d52 = snap319
					d105 = snap320
					d106 = snap321
					d107 = snap322
					d108 = snap323
					d110 = snap324
					d142 = snap325
					d145 = snap326
					d178 = snap327
					d179 = snap328
					d180 = snap329
					d181 = snap330
					d182 = snap331
					d183 = snap332
					d184 = snap333
					d185 = snap334
					d187 = snap335
					d188 = snap336
					d189 = snap337
					d191 = snap338
					d192 = snap339
					d193 = snap340
					d194 = snap341
					d195 = snap342
					d196 = snap343
					d197 = snap344
					d198 = snap345
					d199 = snap346
					d200 = snap347
					d201 = snap348
					d202 = snap349
					d203 = snap350
					d204 = snap351
					d205 = snap352
					d206 = snap353
					d207 = snap354
					d209 = snap355
					d210 = snap356
					d211 = snap357
					d212 = snap358
					d214 = snap359
					d215 = snap360
					d216 = snap361
					d217 = snap362
					d219 = snap363
					d221 = snap364
					d292 = snap365
					d295 = snap366
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps293)
					}
					return result
					return result
				}
				ps368 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps368)
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
				var d15 JITValueDesc
				_ = d15
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
					ctx.EmitJump(d2.Condition, lbl2)
					snap5 := d0
					snap6 := d1
					snap7 := d2
					alloc8 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 3)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 3)
					ps10.OverlayValues[0] = d0
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					snap11 := d0
					snap12 := d1
					snap13 := d2
					alloc14 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc14)
					d0 = snap11
					d1 = snap12
					d2 = snap13
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
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
					d15 = args[0]
					d15.ID = 0
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					d15 = JITPrepareScmerGoArg(ctx, d15)
					ctx.SyncDesc(&d15)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(querySessionState), []JITValueDesc{d15}, []uint8{1, 1}, []uint8{1, 0})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.FreeDesc(&d15)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.EnsureDesc(&d17)
					var d19 JITValueDesc
					if d17.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc != LocReg && d17.Loc != LocRegPair && d17.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r1 := ctx.AllocRegExcept(d17.Reg)
						ctx.EmitCmpRegImm32(d17.Reg, 0)
						ctx.EmitSetcc(r1, CondEqual)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[0] = d0
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[2] = d2
							ps21.OverlayValues[15] = d15
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[3].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[0] = d0
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[2] = d2
						ps22.OverlayValues[15] = d15
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[4].RenderPS(ps22)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					snap23 := d0
					snap24 := d1
					snap25 := d2
					snap26 := d15
					snap27 := d17
					snap28 := d18
					snap29 := d19
					snap30 := d20
					alloc31 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc31)
					d0 = snap23
					d1 = snap24
					d2 = snap25
					d15 = snap26
					d17 = snap27
					d18 = snap28
					d19 = snap29
					d20 = snap30
					ctx.RestoreAllocState(alloc31)
					d0 = snap23
					d1 = snap24
					d2 = snap25
					d15 = snap26
					d17 = snap27
					d18 = snap28
					d19 = snap29
					d20 = snap30
					ps32 := PhiState{General: true}
					ps32.OverlayValues = make([]JITValueDesc, 21)
					ps32.OverlayValues[0] = d0
					ps32.OverlayValues[1] = d1
					ps32.OverlayValues[2] = d2
					ps32.OverlayValues[15] = d15
					ps32.OverlayValues[17] = d17
					ps32.OverlayValues[18] = d18
					ps32.OverlayValues[19] = d19
					ps32.OverlayValues[20] = d20
					ps33 := PhiState{General: true}
					ps33.OverlayValues = make([]JITValueDesc, 21)
					ps33.OverlayValues[0] = d0
					ps33.OverlayValues[1] = d1
					ps33.OverlayValues[2] = d2
					ps33.OverlayValues[15] = d15
					ps33.OverlayValues[17] = d17
					ps33.OverlayValues[18] = d18
					ps33.OverlayValues[19] = d19
					ps33.OverlayValues[20] = d20
					snap34 := d0
					snap35 := d1
					snap36 := d2
					snap37 := d15
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					alloc42 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps33)
					}
					ctx.RestoreAllocState(alloc42)
					d0 = snap34
					d1 = snap35
					d2 = snap36
					d15 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps32)
					}
					return result
					ctx.FreeDesc(&d19)
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					ctx.ReclaimUntrackedRegs()
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d43.Loc == LocImm {
						ctx.EmitMakeInt(result, d43)
					} else {
						ctx.EmitMovToReg(result.Reg2, d43)
						d44 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d44)
						if d43.Loc == LocReg && d43.Reg != result.Reg2 {
							ctx.FreeReg(d43.Reg)
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					ctx.ReclaimUntrackedRegs()
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					if d45.Loc == LocImm {
						ctx.EmitMakeInt(result, d45)
					} else {
						ctx.EmitMovToReg(result.Reg2, d45)
						d46 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d46)
						if d45.Loc == LocReg && d45.Reg != result.Reg2 {
							ctx.FreeReg(d45.Reg)
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					ctx.ReclaimUntrackedRegs()
					var d47 JITValueDesc
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						fieldAddr := uintptr(d17.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d47 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d47)
					} else {
						off := int32(0)
						baseReg := d17.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d47 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d47)
					}
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d47)
					var d48 JITValueDesc
					if d47.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d47.Imm.Int()))))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegReg(r4, d47.Reg)
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d48)
					}
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						ctx.EmitMakeInt(result, d48)
					} else {
						ctx.EmitMovToReg(result.Reg2, d48)
						d49 := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeInt(result, d49)
						if d48.Loc == LocReg && d48.Reg != result.Reg2 {
							ctx.FreeReg(d48.Reg)
						}
					}
					result.Type = tagInt
					ctx.EmitJmp(lbl0)
					return result
				}
				ps50 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps50)
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
