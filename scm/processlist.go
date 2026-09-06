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
				var d109 JITValueDesc
				_ = d109
				var d111 JITValueDesc
				_ = d111
				var d144 JITValueDesc
				_ = d144
				var d147 JITValueDesc
				_ = d147
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
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var stackArray193 int32
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
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d218 JITValueDesc
				_ = d218
				var d219 JITValueDesc
				_ = d219
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d223 JITValueDesc
				_ = d223
				var d225 JITValueDesc
				_ = d225
				var d298 JITValueDesc
				_ = d298
				var d301 JITValueDesc
				_ = d301
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
					lbl11 := ctx.ReserveLabel()
					ctx.EmitJump(d8.Condition, lbl10)
					ctx.EmitJmp(lbl11)
					snap12 := d3
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d7
					snap17 := d8
					snap18 := d11
					alloc19 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc19)
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d7 = snap16
					d8 = snap17
					d11 = snap18
					ctx.MarkLabel(lbl11)
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
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitJump(d49.Condition, lbl12)
					ctx.EmitJmp(lbl13)
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
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl5)
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
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl6)
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
					d106 = ctx.EmitSliceElementAddress(&d39, &d47, 8)
					ctx.EnsureDesc(&d106)
					ctx.EmitMovRegMem(d106.Reg, d106.Reg, 0)
					d105 = d106
					d105.Type = JITTypeUnknown
					ctx.StabilizeDescForControlFlow(&d105)
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d107 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d107.NoHeapPointer = false
					ctx.BindReg(d107.Reg, &d107)
					ctx.BindReg(d107.Reg2, &d107)
					ctx.StabilizeDescForControlFlow(&d107)
					d108 = d3
					ctx.EnsureDesc(&d108)
					if d108.Loc != LocImm && d108.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d108.Loc == LocImm {
						if d108.Imm.Bool() {
							if ps.General {
								ctx.SyncDesc(&d107)
								if d107.Loc == LocReg {
									ctx.ProtectReg(d107.Reg)
								} else if d107.Loc == LocRegPair {
									ctx.ProtectReg(d107.Reg)
									ctx.ProtectReg(d107.Reg2)
								}
								d109 = d107
								if d109.Loc == LocNone {
									panic("jit: phi source has no location")
								}
								ctx.SyncDesc(&d109)
								if d109.Loc == LocStackPair {
									ctx.EmitCopyStackWords(d109, int32(bbs[7].PhiBase)+int32(0), 2)
								} else if d109.Loc == LocInputPair {
									ctx.EnsureDesc(&d109)
									ctx.EmitStoreScmerToStack(d109, int32(bbs[7].PhiBase)+int32(0))
								} else if d109.Loc == LocRegPair || d109.Loc == LocImm {
									ctx.EmitStoreScmerToStack(d109, int32(bbs[7].PhiBase)+int32(0))
								} else {
									ctx.EnsureDesc(&d109)
									ctx.EmitStoreToStack(d109, int32(bbs[7].PhiBase)+int32(0))
									ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
								}
								if d107.Loc == LocReg {
									ctx.UnprotectReg(d107.Reg)
								} else if d107.Loc == LocRegPair {
									ctx.UnprotectReg(d107.Reg)
									ctx.UnprotectReg(d107.Reg2)
								}
							}
							ps110 := PhiState{General: ps.General}
							ps110.OverlayValues = make([]JITValueDesc, 110)
							ps110.OverlayValues[3] = d3
							ps110.OverlayValues[4] = d4
							ps110.OverlayValues[5] = d5
							ps110.OverlayValues[6] = d6
							ps110.OverlayValues[7] = d7
							ps110.OverlayValues[8] = d8
							ps110.OverlayValues[11] = d11
							ps110.OverlayValues[22] = d22
							ps110.OverlayValues[32] = d32
							ps110.OverlayValues[33] = d33
							ps110.OverlayValues[34] = d34
							ps110.OverlayValues[35] = d35
							ps110.OverlayValues[37] = d37
							ps110.OverlayValues[38] = d38
							ps110.OverlayValues[39] = d39
							ps110.OverlayValues[40] = d40
							ps110.OverlayValues[42] = d42
							ps110.OverlayValues[43] = d43
							ps110.OverlayValues[45] = d45
							ps110.OverlayValues[46] = d46
							ps110.OverlayValues[47] = d47
							ps110.OverlayValues[48] = d48
							ps110.OverlayValues[49] = d49
							ps110.OverlayValues[52] = d52
							ps110.OverlayValues[105] = d105
							ps110.OverlayValues[106] = d106
							ps110.OverlayValues[107] = d107
							ps110.OverlayValues[108] = d108
							ps110.OverlayValues[109] = d109
							ps110.PhiValues = make([]JITValueDesc, 1)
							d111 = d107
							ps110.PhiValues[0] = d111
							return bbs[7].RenderPS(ps110)
						}
						if ps.General {
						}
						ps112 := PhiState{General: ps.General}
						ps112.OverlayValues = make([]JITValueDesc, 112)
						ps112.OverlayValues[3] = d3
						ps112.OverlayValues[4] = d4
						ps112.OverlayValues[5] = d5
						ps112.OverlayValues[6] = d6
						ps112.OverlayValues[7] = d7
						ps112.OverlayValues[8] = d8
						ps112.OverlayValues[11] = d11
						ps112.OverlayValues[22] = d22
						ps112.OverlayValues[32] = d32
						ps112.OverlayValues[33] = d33
						ps112.OverlayValues[34] = d34
						ps112.OverlayValues[35] = d35
						ps112.OverlayValues[37] = d37
						ps112.OverlayValues[38] = d38
						ps112.OverlayValues[39] = d39
						ps112.OverlayValues[40] = d40
						ps112.OverlayValues[42] = d42
						ps112.OverlayValues[43] = d43
						ps112.OverlayValues[45] = d45
						ps112.OverlayValues[46] = d46
						ps112.OverlayValues[47] = d47
						ps112.OverlayValues[48] = d48
						ps112.OverlayValues[49] = d49
						ps112.OverlayValues[52] = d52
						ps112.OverlayValues[105] = d105
						ps112.OverlayValues[106] = d106
						ps112.OverlayValues[107] = d107
						ps112.OverlayValues[108] = d108
						ps112.OverlayValues[109] = d109
						ps112.OverlayValues[111] = d111
						return bbs[8].RenderPS(ps112)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d108.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					snap113 := d3
					snap114 := d4
					snap115 := d5
					snap116 := d6
					snap117 := d7
					snap118 := d8
					snap119 := d11
					snap120 := d22
					snap121 := d32
					snap122 := d33
					snap123 := d34
					snap124 := d35
					snap125 := d37
					snap126 := d38
					snap127 := d39
					snap128 := d40
					snap129 := d42
					snap130 := d43
					snap131 := d45
					snap132 := d46
					snap133 := d47
					snap134 := d48
					snap135 := d49
					snap136 := d52
					snap137 := d105
					snap138 := d106
					snap139 := d107
					snap140 := d108
					snap141 := d109
					snap142 := d111
					alloc143 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl14)
					ctx.SyncDesc(&d107)
					if d107.Loc == LocReg {
						ctx.ProtectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.ProtectReg(d107.Reg)
						ctx.ProtectReg(d107.Reg2)
					}
					d144 = d107
					if d144.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d144)
					if d144.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d144, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d144.Loc == LocInputPair {
						ctx.EnsureDesc(&d144)
						ctx.EmitStoreScmerToStack(d144, int32(bbs[7].PhiBase)+int32(0))
					} else if d144.Loc == LocRegPair || d144.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d144, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d144)
						ctx.EmitStoreToStack(d144, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d107.Loc == LocReg {
						ctx.UnprotectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.UnprotectReg(d107.Reg)
						ctx.UnprotectReg(d107.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc143)
					d3 = snap113
					d4 = snap114
					d5 = snap115
					d6 = snap116
					d7 = snap117
					d8 = snap118
					d11 = snap119
					d22 = snap120
					d32 = snap121
					d33 = snap122
					d34 = snap123
					d35 = snap124
					d37 = snap125
					d38 = snap126
					d39 = snap127
					d40 = snap128
					d42 = snap129
					d43 = snap130
					d45 = snap131
					d46 = snap132
					d47 = snap133
					d48 = snap134
					d49 = snap135
					d52 = snap136
					d105 = snap137
					d106 = snap138
					d107 = snap139
					d108 = snap140
					d109 = snap141
					d111 = snap142
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl9)
					ctx.RestoreAllocState(alloc143)
					d3 = snap113
					d4 = snap114
					d5 = snap115
					d6 = snap116
					d7 = snap117
					d8 = snap118
					d11 = snap119
					d22 = snap120
					d32 = snap121
					d33 = snap122
					d34 = snap123
					d35 = snap124
					d37 = snap125
					d38 = snap126
					d39 = snap127
					d40 = snap128
					d42 = snap129
					d43 = snap130
					d45 = snap131
					d46 = snap132
					d47 = snap133
					d48 = snap134
					d49 = snap135
					d52 = snap136
					d105 = snap137
					d106 = snap138
					d107 = snap139
					d108 = snap140
					d109 = snap141
					d111 = snap142
					ps145 := PhiState{General: true}
					ps145.OverlayValues = make([]JITValueDesc, 145)
					ps145.OverlayValues[3] = d3
					ps145.OverlayValues[4] = d4
					ps145.OverlayValues[5] = d5
					ps145.OverlayValues[6] = d6
					ps145.OverlayValues[7] = d7
					ps145.OverlayValues[8] = d8
					ps145.OverlayValues[11] = d11
					ps145.OverlayValues[22] = d22
					ps145.OverlayValues[32] = d32
					ps145.OverlayValues[33] = d33
					ps145.OverlayValues[34] = d34
					ps145.OverlayValues[35] = d35
					ps145.OverlayValues[37] = d37
					ps145.OverlayValues[38] = d38
					ps145.OverlayValues[39] = d39
					ps145.OverlayValues[40] = d40
					ps145.OverlayValues[42] = d42
					ps145.OverlayValues[43] = d43
					ps145.OverlayValues[45] = d45
					ps145.OverlayValues[46] = d46
					ps145.OverlayValues[47] = d47
					ps145.OverlayValues[48] = d48
					ps145.OverlayValues[49] = d49
					ps145.OverlayValues[52] = d52
					ps145.OverlayValues[105] = d105
					ps145.OverlayValues[106] = d106
					ps145.OverlayValues[107] = d107
					ps145.OverlayValues[108] = d108
					ps145.OverlayValues[109] = d109
					ps145.OverlayValues[111] = d111
					ps145.OverlayValues[144] = d144
					ps145.PhiValues = make([]JITValueDesc, 1)
					d147 = d107
					ps145.PhiValues[0] = d147
					ps146 := PhiState{General: true}
					ps146.OverlayValues = make([]JITValueDesc, 148)
					ps146.OverlayValues[3] = d3
					ps146.OverlayValues[4] = d4
					ps146.OverlayValues[5] = d5
					ps146.OverlayValues[6] = d6
					ps146.OverlayValues[7] = d7
					ps146.OverlayValues[8] = d8
					ps146.OverlayValues[11] = d11
					ps146.OverlayValues[22] = d22
					ps146.OverlayValues[32] = d32
					ps146.OverlayValues[33] = d33
					ps146.OverlayValues[34] = d34
					ps146.OverlayValues[35] = d35
					ps146.OverlayValues[37] = d37
					ps146.OverlayValues[38] = d38
					ps146.OverlayValues[39] = d39
					ps146.OverlayValues[40] = d40
					ps146.OverlayValues[42] = d42
					ps146.OverlayValues[43] = d43
					ps146.OverlayValues[45] = d45
					ps146.OverlayValues[46] = d46
					ps146.OverlayValues[47] = d47
					ps146.OverlayValues[48] = d48
					ps146.OverlayValues[49] = d49
					ps146.OverlayValues[52] = d52
					ps146.OverlayValues[105] = d105
					ps146.OverlayValues[106] = d106
					ps146.OverlayValues[107] = d107
					ps146.OverlayValues[108] = d108
					ps146.OverlayValues[109] = d109
					ps146.OverlayValues[111] = d111
					ps146.OverlayValues[144] = d144
					ps146.OverlayValues[147] = d147
					snap148 := d3
					snap149 := d4
					snap150 := d5
					snap151 := d6
					snap152 := d7
					snap153 := d8
					snap154 := d11
					snap155 := d22
					snap156 := d32
					snap157 := d33
					snap158 := d34
					snap159 := d35
					snap160 := d37
					snap161 := d38
					snap162 := d39
					snap163 := d40
					snap164 := d42
					snap165 := d43
					snap166 := d45
					snap167 := d46
					snap168 := d47
					snap169 := d48
					snap170 := d49
					snap171 := d52
					snap172 := d105
					snap173 := d106
					snap174 := d107
					snap175 := d108
					snap176 := d109
					snap177 := d111
					snap178 := d144
					snap179 := d147
					alloc180 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps145)
					}
					ctx.RestoreAllocState(alloc180)
					d3 = snap148
					d4 = snap149
					d5 = snap150
					d6 = snap151
					d7 = snap152
					d8 = snap153
					d11 = snap154
					d22 = snap155
					d32 = snap156
					d33 = snap157
					d34 = snap158
					d35 = snap159
					d37 = snap160
					d38 = snap161
					d39 = snap162
					d40 = snap163
					d42 = snap164
					d43 = snap165
					d45 = snap166
					d46 = snap167
					d47 = snap168
					d48 = snap169
					d49 = snap170
					d52 = snap171
					d105 = snap172
					d106 = snap173
					d107 = snap174
					d108 = snap175
					d109 = snap176
					d111 = snap177
					d144 = snap178
					d147 = snap179
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps146)
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d42)
					d181 = ctx.EmitNewSliceFromGoSlice(&d42)
					ctx.SyncDesc(&d181)
					if d181.Loc == LocRegPair || d181.Loc == LocStackPair || d181.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d181, &result)
						result.Type = d181.Type
					} else {
						switch d181.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d181)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d181)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d181)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d181, &result)
							result.Type = d181.Type
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					ctx.ReclaimUntrackedRegs()
					d182 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d183 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d182)
					ctx.EnsureDesc(&d183)
					var d185 JITValueDesc
					if d183.Loc == LocImm && d182.Loc == LocImm {
						d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d183.Imm.Int() - d182.Imm.Int())}
					} else {
						r5 := ctx.AllocReg()
						if d183.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d183.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d183.Reg)
						}
						if d182.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d182.Imm.Int()))
							ctx.EmitSubInt64(r5, RegR11)
						} else {
							ctx.EmitSubInt64(r5, d182.Reg)
						}
						d185 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d185)
					}
					var d186 JITValueDesc
					r6 := ctx.EmitSliceDataAfterLow(&d107, &d182, 1)
					d186 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r6}
					ctx.BindReg(r6, &d186)
					ctx.BindReg(r6, &d186)
					var d187 JITValueDesc
					var r7 Reg
					var r8 Reg
					ctx.SyncDesc(&d186)
					ctx.EnsureDesc(&d186)
					if d186.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d186.Imm.Int()))
					} else {
						r7 = d186.Reg
					}
					ctx.ProtectReg(r7)
					ctx.SyncDesc(&d185)
					ctx.EnsureDesc(&d185)
					if d185.Loc == LocImm {
						r8 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r8, uint64(d185.Imm.Int()))
					} else {
						r8 = d185.Reg
					}
					ctx.ProtectReg(r8)
					ctx.UnprotectReg(r8)
					ctx.UnprotectReg(r7)
					d187 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d187)
					ctx.BindReg(r8, &d187)
					ctx.BindReg(r7, &d187)
					ctx.BindReg(r8, &d187)
					ctx.StabilizeDescForControlFlow(&d187)
					if ps.General {
						ctx.SyncDesc(&d187)
						if d187.Loc == LocReg {
							ctx.ProtectReg(d187.Reg)
						} else if d187.Loc == LocRegPair {
							ctx.ProtectReg(d187.Reg)
							ctx.ProtectReg(d187.Reg2)
						}
						d188 = d187
						if d188.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d188)
						if d188.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d188, int32(bbs[7].PhiBase)+int32(0), 2)
						} else if d188.Loc == LocInputPair {
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreScmerToStack(d188, int32(bbs[7].PhiBase)+int32(0))
						} else if d188.Loc == LocRegPair || d188.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d188, int32(bbs[7].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreToStack(d188, int32(bbs[7].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
						}
						if d187.Loc == LocReg {
							ctx.UnprotectReg(d187.Reg)
						} else if d187.Loc == LocRegPair {
							ctx.UnprotectReg(d187.Reg)
							ctx.UnprotectReg(d187.Reg2)
						}
					}
					ps189 := PhiState{General: ps.General}
					ps189.OverlayValues = make([]JITValueDesc, 189)
					ps189.OverlayValues[3] = d3
					ps189.OverlayValues[4] = d4
					ps189.OverlayValues[5] = d5
					ps189.OverlayValues[6] = d6
					ps189.OverlayValues[7] = d7
					ps189.OverlayValues[8] = d8
					ps189.OverlayValues[11] = d11
					ps189.OverlayValues[22] = d22
					ps189.OverlayValues[32] = d32
					ps189.OverlayValues[33] = d33
					ps189.OverlayValues[34] = d34
					ps189.OverlayValues[35] = d35
					ps189.OverlayValues[37] = d37
					ps189.OverlayValues[38] = d38
					ps189.OverlayValues[39] = d39
					ps189.OverlayValues[40] = d40
					ps189.OverlayValues[42] = d42
					ps189.OverlayValues[43] = d43
					ps189.OverlayValues[45] = d45
					ps189.OverlayValues[46] = d46
					ps189.OverlayValues[47] = d47
					ps189.OverlayValues[48] = d48
					ps189.OverlayValues[49] = d49
					ps189.OverlayValues[52] = d52
					ps189.OverlayValues[105] = d105
					ps189.OverlayValues[106] = d106
					ps189.OverlayValues[107] = d107
					ps189.OverlayValues[108] = d108
					ps189.OverlayValues[109] = d109
					ps189.OverlayValues[111] = d111
					ps189.OverlayValues[144] = d144
					ps189.OverlayValues[147] = d147
					ps189.OverlayValues[181] = d181
					ps189.OverlayValues[182] = d182
					ps189.OverlayValues[183] = d183
					ps189.OverlayValues[184] = d184
					ps189.OverlayValues[185] = d185
					ps189.OverlayValues[186] = d186
					ps189.OverlayValues[187] = d187
					ps189.OverlayValues[188] = d188
					ps189.PhiValues = make([]JITValueDesc, 1)
					d190 = d187
					ps189.PhiValues[0] = d190
					if ps189.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps189)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d191 := ps.PhiValues[0]
							ctx.EnsureDesc(&d191)
							ctx.EmitStoreScmerToStack(d191, int32(bbs[7].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
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
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
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
					d192 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d105}, 2)
					d192.NoHeapPointer = false
					ctx.BindReg(d192.Reg, &d192)
					ctx.BindReg(d192.Reg2, &d192)
					stackArray193 = ctx.AllocStack(int32(256))
					_ = stackArray193
					d194 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.SyncDesc(&d194)
					ctx.EmitStoreScmerToStack(d194, int32(stackArray193)+int32(0))
					var d195 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 0
						r9 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r9, fieldAddr)
						d195 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d195)
					} else {
						off := int32(0)
						baseReg := d105.Reg
						r10 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r10, baseReg, off)
						d195 = JITValueDesc{Loc: LocReg, Reg: r10}
						ctx.BindReg(r10, &d195)
					}
					ctx.EnsureDesc(&d195)
					ctx.EnsureDesc(&d195)
					var d196 JITValueDesc
					if d195.Loc == LocImm {
						d196 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d195.Imm.Int()))))}
					} else {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegReg(r11, d195.Reg)
						d196 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d196)
					}
					ctx.FreeDesc(&d195)
					ctx.EnsureDesc(&d196)
					ctx.SyncDesc(&d196)
					ctx.EnsureDesc(&d196)
					ctx.EmitStoreTypedScmerToStack(d196, tagInt, int32(stackArray193)+int32(16))
					d197 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.SyncDesc(&d197)
					ctx.EmitStoreScmerToStack(d197, int32(stackArray193)+int32(32))
					var d198 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 8
						r12 := ctx.AllocReg()
						r13 := ctx.AllocRegExcept(r12)
						r14 := ctx.AllocRegExcept(r12, r13)
						ctx.EmitMovRegMem64(r12, fieldAddr)
						ctx.EmitMovRegMem64(r13, fieldAddr+8)
						ctx.EmitMovRegMem64(r14, fieldAddr+16)
						d198 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
						ctx.BindReg(r12, &d198)
						ctx.BindReg(r13, &d198)
						ctx.BindReg(r14, &d198)
					} else {
						off := int32(8)
						baseReg := d105.Reg
						r15 := ctx.AllocRegExcept(baseReg)
						r16 := ctx.AllocRegExcept(baseReg, r15)
						r17 := ctx.AllocRegExcept(baseReg, r15, r16)
						ctx.EmitMovRegMem(r15, baseReg, off)
						ctx.EmitMovRegMem(r16, baseReg, off+8)
						ctx.EmitMovRegMem(r17, baseReg, off+16)
						d198 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d198)
						ctx.BindReg(r16, &d198)
						ctx.BindReg(r17, &d198)
					}
					ctx.EnsureDesc(&d198)
					ctx.SyncDesc(&d198)
					ctx.EmitStoreScmerToStack(d198, int32(stackArray193)+int32(48))
					d199 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.SyncDesc(&d199)
					ctx.EmitStoreScmerToStack(d199, int32(stackArray193)+int32(64))
					var d200 JITValueDesc
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocImm {
						fieldAddr := uintptr(d105.Imm.Int()) + 24
						r18 := ctx.AllocReg()
						r19 := ctx.AllocRegExcept(r18)
						r20 := ctx.AllocRegExcept(r18, r19)
						ctx.EmitMovRegMem64(r18, fieldAddr)
						ctx.EmitMovRegMem64(r19, fieldAddr+8)
						ctx.EmitMovRegMem64(r20, fieldAddr+16)
						d200 = JITValueDesc{Loc: LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
						ctx.BindReg(r18, &d200)
						ctx.BindReg(r19, &d200)
						ctx.BindReg(r20, &d200)
					} else {
						off := int32(24)
						baseReg := d105.Reg
						r21 := ctx.AllocRegExcept(baseReg)
						r22 := ctx.AllocRegExcept(baseReg, r21)
						r23 := ctx.AllocRegExcept(baseReg, r21, r22)
						ctx.EmitMovRegMem(r21, baseReg, off)
						ctx.EmitMovRegMem(r22, baseReg, off+8)
						ctx.EmitMovRegMem(r23, baseReg, off+16)
						d200 = JITValueDesc{Loc: LocRegTriple, Reg: r21, Reg2: r22, Reg3: r23}
						ctx.BindReg(r21, &d200)
						ctx.BindReg(r22, &d200)
						ctx.BindReg(r23, &d200)
					}
					ctx.EnsureDesc(&d200)
					ctx.SyncDesc(&d200)
					ctx.EmitStoreScmerToStack(d200, int32(stackArray193)+int32(80))
					d201 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.SyncDesc(&d201)
					ctx.EmitStoreScmerToStack(d201, int32(stackArray193)+int32(96))
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d202 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d202.NoHeapPointer = false
					ctx.BindReg(d202.Reg, &d202)
					ctx.BindReg(d202.Reg2, &d202)
					ctx.EnsureDesc(&d202)
					ctx.SyncDesc(&d202)
					ctx.EmitStoreScmerToStack(d202, int32(stackArray193)+int32(112))
					d203 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.SyncDesc(&d203)
					ctx.EmitStoreScmerToStack(d203, int32(stackArray193)+int32(128))
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d204 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d105}, 2)
					d204.NoHeapPointer = false
					ctx.BindReg(d204.Reg, &d204)
					ctx.BindReg(d204.Reg2, &d204)
					ctx.EnsureDesc(&d204)
					ctx.SyncDesc(&d204)
					ctx.EmitStoreScmerToStack(d204, int32(stackArray193)+int32(144))
					d205 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.SyncDesc(&d205)
					ctx.EmitStoreScmerToStack(d205, int32(stackArray193)+int32(160))
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					if d105.Loc == LocRegPair || d105.Loc == LocStackPair || d105.Loc == LocRegTriple || d105.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d105)
					d206 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d105}, 1)
					d206.NoHeapPointer = true
					ctx.BindReg(d206.Reg, &d206)
					ctx.EnsureDesc(&d206)
					ctx.SyncDesc(&d206)
					ctx.EnsureDesc(&d206)
					ctx.EmitStoreTypedScmerToStack(d206, tagInt, int32(stackArray193)+int32(176))
					d207 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.SyncDesc(&d207)
					ctx.EmitStoreScmerToStack(d207, int32(stackArray193)+int32(192))
					ctx.EnsureDesc(&d192)
					ctx.SyncDesc(&d192)
					ctx.EmitStoreScmerToStack(d192, int32(stackArray193)+int32(208))
					d208 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.SyncDesc(&d208)
					ctx.EmitStoreScmerToStack(d208, int32(stackArray193)+int32(224))
					ctx.EnsureDesc(&d5)
					ctx.SyncDesc(&d5)
					ctx.EmitStoreScmerToStack(d5, int32(stackArray193)+int32(240))
					d209 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d209
					r24 := ctx.AllocReg()
					r25 := ctx.AllocRegExcept(r24)
					r26 := ctx.AllocRegExcept(r24, r25)
					d210 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r24, Reg2: r25, Reg3: r26}
					ctx.BindReg(r24, &d210)
					ctx.BindReg(r25, &d210)
					ctx.BindReg(r26, &d210)
					ctx.BindReg(r24, &d210)
					ctx.BindReg(r25, &d210)
					ctx.BindReg(r26, &d210)
					ctx.EmitLeaRegMem(d210.Reg, ctx.StackReg, int32(stackArray193))
					ctx.EmitMovRegImm64(d210.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d210.Reg3, uint64(16))
					callResults211 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d210}, []uint8{2}, []uint8{1})
					d212 = callResults211[0]
					ctx.EnsureDesc(&d47)
					ctx.SyncDesc(&d212)
					ctx.StabilizeDescAcrossNestedCall(&d47)
					d213 = d42
					d213.ID = 0
					d214 = d47
					d214.ID = 0
					d215 = ctx.EmitSliceElementAddress(&d213, &d214, int32(16))
					ctx.FreeDesc(&d214)
					ctx.EmitStoreScmerAt(&d215, &d212)
					ctx.FreeDesc(&d215)
					ctx.FreeDesc(&d212)
					if ps.General {
						ctx.SyncDesc(&d47)
						if d47.Loc == LocReg {
							ctx.ProtectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.ProtectReg(d47.Reg)
							ctx.ProtectReg(d47.Reg2)
						}
						d216 = d47
						if d216.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d216)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d216)
						} else {
							ctx.EmitStoreToStack(d216, int32(bbs[3].PhiBase)+int32(0))
						}
						if d47.Loc == LocReg {
							ctx.UnprotectReg(d47.Reg)
						} else if d47.Loc == LocRegPair {
							ctx.UnprotectReg(d47.Reg)
							ctx.UnprotectReg(d47.Reg2)
						}
					}
					ps217 := PhiState{General: ps.General}
					ps217.OverlayValues = make([]JITValueDesc, 217)
					ps217.OverlayValues[3] = d3
					ps217.OverlayValues[4] = d4
					ps217.OverlayValues[5] = d5
					ps217.OverlayValues[6] = d6
					ps217.OverlayValues[7] = d7
					ps217.OverlayValues[8] = d8
					ps217.OverlayValues[11] = d11
					ps217.OverlayValues[22] = d22
					ps217.OverlayValues[32] = d32
					ps217.OverlayValues[33] = d33
					ps217.OverlayValues[34] = d34
					ps217.OverlayValues[35] = d35
					ps217.OverlayValues[37] = d37
					ps217.OverlayValues[38] = d38
					ps217.OverlayValues[39] = d39
					ps217.OverlayValues[40] = d40
					ps217.OverlayValues[42] = d42
					ps217.OverlayValues[43] = d43
					ps217.OverlayValues[45] = d45
					ps217.OverlayValues[46] = d46
					ps217.OverlayValues[47] = d47
					ps217.OverlayValues[48] = d48
					ps217.OverlayValues[49] = d49
					ps217.OverlayValues[52] = d52
					ps217.OverlayValues[105] = d105
					ps217.OverlayValues[106] = d106
					ps217.OverlayValues[107] = d107
					ps217.OverlayValues[108] = d108
					ps217.OverlayValues[109] = d109
					ps217.OverlayValues[111] = d111
					ps217.OverlayValues[144] = d144
					ps217.OverlayValues[147] = d147
					ps217.OverlayValues[181] = d181
					ps217.OverlayValues[182] = d182
					ps217.OverlayValues[183] = d183
					ps217.OverlayValues[184] = d184
					ps217.OverlayValues[185] = d185
					ps217.OverlayValues[186] = d186
					ps217.OverlayValues[187] = d187
					ps217.OverlayValues[188] = d188
					ps217.OverlayValues[190] = d190
					ps217.OverlayValues[191] = d191
					ps217.OverlayValues[192] = d192
					ps217.OverlayValues[194] = d194
					ps217.OverlayValues[195] = d195
					ps217.OverlayValues[196] = d196
					ps217.OverlayValues[197] = d197
					ps217.OverlayValues[198] = d198
					ps217.OverlayValues[199] = d199
					ps217.OverlayValues[200] = d200
					ps217.OverlayValues[201] = d201
					ps217.OverlayValues[202] = d202
					ps217.OverlayValues[203] = d203
					ps217.OverlayValues[204] = d204
					ps217.OverlayValues[205] = d205
					ps217.OverlayValues[206] = d206
					ps217.OverlayValues[207] = d207
					ps217.OverlayValues[208] = d208
					ps217.OverlayValues[209] = d209
					ps217.OverlayValues[210] = d210
					ps217.OverlayValues[212] = d212
					ps217.OverlayValues[213] = d213
					ps217.OverlayValues[214] = d214
					ps217.OverlayValues[215] = d215
					ps217.OverlayValues[216] = d216
					ps217.PhiValues = make([]JITValueDesc, 1)
					d218 = d47
					ps217.PhiValues[0] = d218
					if ps217.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps217)
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
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
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
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
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
						d213 = ps.OverlayValues[213]
					}
					if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
						d214 = ps.OverlayValues[214]
					}
					if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
						d215 = ps.OverlayValues[215]
					}
					if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != LocNone {
						d216 = ps.OverlayValues[216]
					}
					if len(ps.OverlayValues) > 218 && ps.OverlayValues[218].Loc != LocNone {
						d218 = ps.OverlayValues[218]
					}
					ctx.ReclaimUntrackedRegs()
					var d219 JITValueDesc
					if d107.SliceSizeKnown {
						d219 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d107.KnownSliceLen))}
					} else if d107.Loc == LocImm {
						d219 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d107.Imm.String())))}
					} else if d107.Loc == LocStackTriple {
						d219 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d107.StackOff + 8, NoHeapPointer: true}
					} else if d107.Loc == LocStackPair {
						d219 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d107.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d107)
						if d107.Loc == LocRegPair || d107.Loc == LocRegTriple {
							d219 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg2, ID: 0}
						} else if d107.Loc == LocReg {
							d219 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d219)
					var d220 JITValueDesc
					if d219.Loc == LocImm {
						d220 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d219.Imm.Int() > 100)}
					} else {
						r27 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d219.Reg, 100)
						d220 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedGreater}
						ctx.BindReg(r27, &d220)
					}
					ctx.FreeDesc(&d219)
					d221 = d220
					ctx.EnsureDesc(&d221)
					if d221.Loc != LocImm && d221.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d221.Loc == LocImm {
						if d221.Imm.Bool() {
							if ps.General {
							}
							ps222 := PhiState{General: ps.General}
							ps222.OverlayValues = make([]JITValueDesc, 222)
							ps222.OverlayValues[3] = d3
							ps222.OverlayValues[4] = d4
							ps222.OverlayValues[5] = d5
							ps222.OverlayValues[6] = d6
							ps222.OverlayValues[7] = d7
							ps222.OverlayValues[8] = d8
							ps222.OverlayValues[11] = d11
							ps222.OverlayValues[22] = d22
							ps222.OverlayValues[32] = d32
							ps222.OverlayValues[33] = d33
							ps222.OverlayValues[34] = d34
							ps222.OverlayValues[35] = d35
							ps222.OverlayValues[37] = d37
							ps222.OverlayValues[38] = d38
							ps222.OverlayValues[39] = d39
							ps222.OverlayValues[40] = d40
							ps222.OverlayValues[42] = d42
							ps222.OverlayValues[43] = d43
							ps222.OverlayValues[45] = d45
							ps222.OverlayValues[46] = d46
							ps222.OverlayValues[47] = d47
							ps222.OverlayValues[48] = d48
							ps222.OverlayValues[49] = d49
							ps222.OverlayValues[52] = d52
							ps222.OverlayValues[105] = d105
							ps222.OverlayValues[106] = d106
							ps222.OverlayValues[107] = d107
							ps222.OverlayValues[108] = d108
							ps222.OverlayValues[109] = d109
							ps222.OverlayValues[111] = d111
							ps222.OverlayValues[144] = d144
							ps222.OverlayValues[147] = d147
							ps222.OverlayValues[181] = d181
							ps222.OverlayValues[182] = d182
							ps222.OverlayValues[183] = d183
							ps222.OverlayValues[184] = d184
							ps222.OverlayValues[185] = d185
							ps222.OverlayValues[186] = d186
							ps222.OverlayValues[187] = d187
							ps222.OverlayValues[188] = d188
							ps222.OverlayValues[190] = d190
							ps222.OverlayValues[191] = d191
							ps222.OverlayValues[192] = d192
							ps222.OverlayValues[194] = d194
							ps222.OverlayValues[195] = d195
							ps222.OverlayValues[196] = d196
							ps222.OverlayValues[197] = d197
							ps222.OverlayValues[198] = d198
							ps222.OverlayValues[199] = d199
							ps222.OverlayValues[200] = d200
							ps222.OverlayValues[201] = d201
							ps222.OverlayValues[202] = d202
							ps222.OverlayValues[203] = d203
							ps222.OverlayValues[204] = d204
							ps222.OverlayValues[205] = d205
							ps222.OverlayValues[206] = d206
							ps222.OverlayValues[207] = d207
							ps222.OverlayValues[208] = d208
							ps222.OverlayValues[209] = d209
							ps222.OverlayValues[210] = d210
							ps222.OverlayValues[212] = d212
							ps222.OverlayValues[213] = d213
							ps222.OverlayValues[214] = d214
							ps222.OverlayValues[215] = d215
							ps222.OverlayValues[216] = d216
							ps222.OverlayValues[218] = d218
							ps222.OverlayValues[219] = d219
							ps222.OverlayValues[220] = d220
							ps222.OverlayValues[221] = d221
							return bbs[6].RenderPS(ps222)
						}
						if ps.General {
							ctx.SyncDesc(&d107)
							if d107.Loc == LocReg {
								ctx.ProtectReg(d107.Reg)
							} else if d107.Loc == LocRegPair {
								ctx.ProtectReg(d107.Reg)
								ctx.ProtectReg(d107.Reg2)
							}
							d223 = d107
							if d223.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d223)
							if d223.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d223, int32(bbs[7].PhiBase)+int32(0), 2)
							} else if d223.Loc == LocInputPair {
								ctx.EnsureDesc(&d223)
								ctx.EmitStoreScmerToStack(d223, int32(bbs[7].PhiBase)+int32(0))
							} else if d223.Loc == LocRegPair || d223.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d223, int32(bbs[7].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d223)
								ctx.EmitStoreToStack(d223, int32(bbs[7].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
							}
							if d107.Loc == LocReg {
								ctx.UnprotectReg(d107.Reg)
							} else if d107.Loc == LocRegPair {
								ctx.UnprotectReg(d107.Reg)
								ctx.UnprotectReg(d107.Reg2)
							}
						}
						ps224 := PhiState{General: ps.General}
						ps224.OverlayValues = make([]JITValueDesc, 224)
						ps224.OverlayValues[3] = d3
						ps224.OverlayValues[4] = d4
						ps224.OverlayValues[5] = d5
						ps224.OverlayValues[6] = d6
						ps224.OverlayValues[7] = d7
						ps224.OverlayValues[8] = d8
						ps224.OverlayValues[11] = d11
						ps224.OverlayValues[22] = d22
						ps224.OverlayValues[32] = d32
						ps224.OverlayValues[33] = d33
						ps224.OverlayValues[34] = d34
						ps224.OverlayValues[35] = d35
						ps224.OverlayValues[37] = d37
						ps224.OverlayValues[38] = d38
						ps224.OverlayValues[39] = d39
						ps224.OverlayValues[40] = d40
						ps224.OverlayValues[42] = d42
						ps224.OverlayValues[43] = d43
						ps224.OverlayValues[45] = d45
						ps224.OverlayValues[46] = d46
						ps224.OverlayValues[47] = d47
						ps224.OverlayValues[48] = d48
						ps224.OverlayValues[49] = d49
						ps224.OverlayValues[52] = d52
						ps224.OverlayValues[105] = d105
						ps224.OverlayValues[106] = d106
						ps224.OverlayValues[107] = d107
						ps224.OverlayValues[108] = d108
						ps224.OverlayValues[109] = d109
						ps224.OverlayValues[111] = d111
						ps224.OverlayValues[144] = d144
						ps224.OverlayValues[147] = d147
						ps224.OverlayValues[181] = d181
						ps224.OverlayValues[182] = d182
						ps224.OverlayValues[183] = d183
						ps224.OverlayValues[184] = d184
						ps224.OverlayValues[185] = d185
						ps224.OverlayValues[186] = d186
						ps224.OverlayValues[187] = d187
						ps224.OverlayValues[188] = d188
						ps224.OverlayValues[190] = d190
						ps224.OverlayValues[191] = d191
						ps224.OverlayValues[192] = d192
						ps224.OverlayValues[194] = d194
						ps224.OverlayValues[195] = d195
						ps224.OverlayValues[196] = d196
						ps224.OverlayValues[197] = d197
						ps224.OverlayValues[198] = d198
						ps224.OverlayValues[199] = d199
						ps224.OverlayValues[200] = d200
						ps224.OverlayValues[201] = d201
						ps224.OverlayValues[202] = d202
						ps224.OverlayValues[203] = d203
						ps224.OverlayValues[204] = d204
						ps224.OverlayValues[205] = d205
						ps224.OverlayValues[206] = d206
						ps224.OverlayValues[207] = d207
						ps224.OverlayValues[208] = d208
						ps224.OverlayValues[209] = d209
						ps224.OverlayValues[210] = d210
						ps224.OverlayValues[212] = d212
						ps224.OverlayValues[213] = d213
						ps224.OverlayValues[214] = d214
						ps224.OverlayValues[215] = d215
						ps224.OverlayValues[216] = d216
						ps224.OverlayValues[218] = d218
						ps224.OverlayValues[219] = d219
						ps224.OverlayValues[220] = d220
						ps224.OverlayValues[221] = d221
						ps224.OverlayValues[223] = d223
						ps224.PhiValues = make([]JITValueDesc, 1)
						d225 = d107
						ps224.PhiValues[0] = d225
						return bbs[7].RenderPS(ps224)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitJump(d221.Condition, lbl16)
					ctx.EmitJmp(lbl17)
					snap226 := d3
					snap227 := d4
					snap228 := d5
					snap229 := d6
					snap230 := d7
					snap231 := d8
					snap232 := d11
					snap233 := d22
					snap234 := d32
					snap235 := d33
					snap236 := d34
					snap237 := d35
					snap238 := d37
					snap239 := d38
					snap240 := d39
					snap241 := d40
					snap242 := d42
					snap243 := d43
					snap244 := d45
					snap245 := d46
					snap246 := d47
					snap247 := d48
					snap248 := d49
					snap249 := d52
					snap250 := d105
					snap251 := d106
					snap252 := d107
					snap253 := d108
					snap254 := d109
					snap255 := d111
					snap256 := d144
					snap257 := d147
					snap258 := d181
					snap259 := d182
					snap260 := d183
					snap261 := d184
					snap262 := d185
					snap263 := d186
					snap264 := d187
					snap265 := d188
					snap266 := d190
					snap267 := d191
					snap268 := d192
					snap269 := d194
					snap270 := d195
					snap271 := d196
					snap272 := d197
					snap273 := d198
					snap274 := d199
					snap275 := d200
					snap276 := d201
					snap277 := d202
					snap278 := d203
					snap279 := d204
					snap280 := d205
					snap281 := d206
					snap282 := d207
					snap283 := d208
					snap284 := d209
					snap285 := d210
					snap286 := d212
					snap287 := d213
					snap288 := d214
					snap289 := d215
					snap290 := d216
					snap291 := d218
					snap292 := d219
					snap293 := d220
					snap294 := d221
					snap295 := d223
					snap296 := d225
					alloc297 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc297)
					d3 = snap226
					d4 = snap227
					d5 = snap228
					d6 = snap229
					d7 = snap230
					d8 = snap231
					d11 = snap232
					d22 = snap233
					d32 = snap234
					d33 = snap235
					d34 = snap236
					d35 = snap237
					d37 = snap238
					d38 = snap239
					d39 = snap240
					d40 = snap241
					d42 = snap242
					d43 = snap243
					d45 = snap244
					d46 = snap245
					d47 = snap246
					d48 = snap247
					d49 = snap248
					d52 = snap249
					d105 = snap250
					d106 = snap251
					d107 = snap252
					d108 = snap253
					d109 = snap254
					d111 = snap255
					d144 = snap256
					d147 = snap257
					d181 = snap258
					d182 = snap259
					d183 = snap260
					d184 = snap261
					d185 = snap262
					d186 = snap263
					d187 = snap264
					d188 = snap265
					d190 = snap266
					d191 = snap267
					d192 = snap268
					d194 = snap269
					d195 = snap270
					d196 = snap271
					d197 = snap272
					d198 = snap273
					d199 = snap274
					d200 = snap275
					d201 = snap276
					d202 = snap277
					d203 = snap278
					d204 = snap279
					d205 = snap280
					d206 = snap281
					d207 = snap282
					d208 = snap283
					d209 = snap284
					d210 = snap285
					d212 = snap286
					d213 = snap287
					d214 = snap288
					d215 = snap289
					d216 = snap290
					d218 = snap291
					d219 = snap292
					d220 = snap293
					d221 = snap294
					d223 = snap295
					d225 = snap296
					ctx.MarkLabel(lbl17)
					ctx.SyncDesc(&d107)
					if d107.Loc == LocReg {
						ctx.ProtectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.ProtectReg(d107.Reg)
						ctx.ProtectReg(d107.Reg2)
					}
					d298 = d107
					if d298.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d298)
					if d298.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d298, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d298.Loc == LocInputPair {
						ctx.EnsureDesc(&d298)
						ctx.EmitStoreScmerToStack(d298, int32(bbs[7].PhiBase)+int32(0))
					} else if d298.Loc == LocRegPair || d298.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d298, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d298)
						ctx.EmitStoreToStack(d298, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d107.Loc == LocReg {
						ctx.UnprotectReg(d107.Reg)
					} else if d107.Loc == LocRegPair {
						ctx.UnprotectReg(d107.Reg)
						ctx.UnprotectReg(d107.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc297)
					d3 = snap226
					d4 = snap227
					d5 = snap228
					d6 = snap229
					d7 = snap230
					d8 = snap231
					d11 = snap232
					d22 = snap233
					d32 = snap234
					d33 = snap235
					d34 = snap236
					d35 = snap237
					d37 = snap238
					d38 = snap239
					d39 = snap240
					d40 = snap241
					d42 = snap242
					d43 = snap243
					d45 = snap244
					d46 = snap245
					d47 = snap246
					d48 = snap247
					d49 = snap248
					d52 = snap249
					d105 = snap250
					d106 = snap251
					d107 = snap252
					d108 = snap253
					d109 = snap254
					d111 = snap255
					d144 = snap256
					d147 = snap257
					d181 = snap258
					d182 = snap259
					d183 = snap260
					d184 = snap261
					d185 = snap262
					d186 = snap263
					d187 = snap264
					d188 = snap265
					d190 = snap266
					d191 = snap267
					d192 = snap268
					d194 = snap269
					d195 = snap270
					d196 = snap271
					d197 = snap272
					d198 = snap273
					d199 = snap274
					d200 = snap275
					d201 = snap276
					d202 = snap277
					d203 = snap278
					d204 = snap279
					d205 = snap280
					d206 = snap281
					d207 = snap282
					d208 = snap283
					d209 = snap284
					d210 = snap285
					d212 = snap286
					d213 = snap287
					d214 = snap288
					d215 = snap289
					d216 = snap290
					d218 = snap291
					d219 = snap292
					d220 = snap293
					d221 = snap294
					d223 = snap295
					d225 = snap296
					ps299 := PhiState{General: true}
					ps299.OverlayValues = make([]JITValueDesc, 299)
					ps299.OverlayValues[3] = d3
					ps299.OverlayValues[4] = d4
					ps299.OverlayValues[5] = d5
					ps299.OverlayValues[6] = d6
					ps299.OverlayValues[7] = d7
					ps299.OverlayValues[8] = d8
					ps299.OverlayValues[11] = d11
					ps299.OverlayValues[22] = d22
					ps299.OverlayValues[32] = d32
					ps299.OverlayValues[33] = d33
					ps299.OverlayValues[34] = d34
					ps299.OverlayValues[35] = d35
					ps299.OverlayValues[37] = d37
					ps299.OverlayValues[38] = d38
					ps299.OverlayValues[39] = d39
					ps299.OverlayValues[40] = d40
					ps299.OverlayValues[42] = d42
					ps299.OverlayValues[43] = d43
					ps299.OverlayValues[45] = d45
					ps299.OverlayValues[46] = d46
					ps299.OverlayValues[47] = d47
					ps299.OverlayValues[48] = d48
					ps299.OverlayValues[49] = d49
					ps299.OverlayValues[52] = d52
					ps299.OverlayValues[105] = d105
					ps299.OverlayValues[106] = d106
					ps299.OverlayValues[107] = d107
					ps299.OverlayValues[108] = d108
					ps299.OverlayValues[109] = d109
					ps299.OverlayValues[111] = d111
					ps299.OverlayValues[144] = d144
					ps299.OverlayValues[147] = d147
					ps299.OverlayValues[181] = d181
					ps299.OverlayValues[182] = d182
					ps299.OverlayValues[183] = d183
					ps299.OverlayValues[184] = d184
					ps299.OverlayValues[185] = d185
					ps299.OverlayValues[186] = d186
					ps299.OverlayValues[187] = d187
					ps299.OverlayValues[188] = d188
					ps299.OverlayValues[190] = d190
					ps299.OverlayValues[191] = d191
					ps299.OverlayValues[192] = d192
					ps299.OverlayValues[194] = d194
					ps299.OverlayValues[195] = d195
					ps299.OverlayValues[196] = d196
					ps299.OverlayValues[197] = d197
					ps299.OverlayValues[198] = d198
					ps299.OverlayValues[199] = d199
					ps299.OverlayValues[200] = d200
					ps299.OverlayValues[201] = d201
					ps299.OverlayValues[202] = d202
					ps299.OverlayValues[203] = d203
					ps299.OverlayValues[204] = d204
					ps299.OverlayValues[205] = d205
					ps299.OverlayValues[206] = d206
					ps299.OverlayValues[207] = d207
					ps299.OverlayValues[208] = d208
					ps299.OverlayValues[209] = d209
					ps299.OverlayValues[210] = d210
					ps299.OverlayValues[212] = d212
					ps299.OverlayValues[213] = d213
					ps299.OverlayValues[214] = d214
					ps299.OverlayValues[215] = d215
					ps299.OverlayValues[216] = d216
					ps299.OverlayValues[218] = d218
					ps299.OverlayValues[219] = d219
					ps299.OverlayValues[220] = d220
					ps299.OverlayValues[221] = d221
					ps299.OverlayValues[223] = d223
					ps299.OverlayValues[225] = d225
					ps299.OverlayValues[298] = d298
					ps300 := PhiState{General: true}
					ps300.OverlayValues = make([]JITValueDesc, 299)
					ps300.OverlayValues[3] = d3
					ps300.OverlayValues[4] = d4
					ps300.OverlayValues[5] = d5
					ps300.OverlayValues[6] = d6
					ps300.OverlayValues[7] = d7
					ps300.OverlayValues[8] = d8
					ps300.OverlayValues[11] = d11
					ps300.OverlayValues[22] = d22
					ps300.OverlayValues[32] = d32
					ps300.OverlayValues[33] = d33
					ps300.OverlayValues[34] = d34
					ps300.OverlayValues[35] = d35
					ps300.OverlayValues[37] = d37
					ps300.OverlayValues[38] = d38
					ps300.OverlayValues[39] = d39
					ps300.OverlayValues[40] = d40
					ps300.OverlayValues[42] = d42
					ps300.OverlayValues[43] = d43
					ps300.OverlayValues[45] = d45
					ps300.OverlayValues[46] = d46
					ps300.OverlayValues[47] = d47
					ps300.OverlayValues[48] = d48
					ps300.OverlayValues[49] = d49
					ps300.OverlayValues[52] = d52
					ps300.OverlayValues[105] = d105
					ps300.OverlayValues[106] = d106
					ps300.OverlayValues[107] = d107
					ps300.OverlayValues[108] = d108
					ps300.OverlayValues[109] = d109
					ps300.OverlayValues[111] = d111
					ps300.OverlayValues[144] = d144
					ps300.OverlayValues[147] = d147
					ps300.OverlayValues[181] = d181
					ps300.OverlayValues[182] = d182
					ps300.OverlayValues[183] = d183
					ps300.OverlayValues[184] = d184
					ps300.OverlayValues[185] = d185
					ps300.OverlayValues[186] = d186
					ps300.OverlayValues[187] = d187
					ps300.OverlayValues[188] = d188
					ps300.OverlayValues[190] = d190
					ps300.OverlayValues[191] = d191
					ps300.OverlayValues[192] = d192
					ps300.OverlayValues[194] = d194
					ps300.OverlayValues[195] = d195
					ps300.OverlayValues[196] = d196
					ps300.OverlayValues[197] = d197
					ps300.OverlayValues[198] = d198
					ps300.OverlayValues[199] = d199
					ps300.OverlayValues[200] = d200
					ps300.OverlayValues[201] = d201
					ps300.OverlayValues[202] = d202
					ps300.OverlayValues[203] = d203
					ps300.OverlayValues[204] = d204
					ps300.OverlayValues[205] = d205
					ps300.OverlayValues[206] = d206
					ps300.OverlayValues[207] = d207
					ps300.OverlayValues[208] = d208
					ps300.OverlayValues[209] = d209
					ps300.OverlayValues[210] = d210
					ps300.OverlayValues[212] = d212
					ps300.OverlayValues[213] = d213
					ps300.OverlayValues[214] = d214
					ps300.OverlayValues[215] = d215
					ps300.OverlayValues[216] = d216
					ps300.OverlayValues[218] = d218
					ps300.OverlayValues[219] = d219
					ps300.OverlayValues[220] = d220
					ps300.OverlayValues[221] = d221
					ps300.OverlayValues[223] = d223
					ps300.OverlayValues[225] = d225
					ps300.OverlayValues[298] = d298
					ps300.PhiValues = make([]JITValueDesc, 1)
					d301 = d107
					ps300.PhiValues[0] = d301
					snap302 := d3
					snap303 := d4
					snap304 := d5
					snap305 := d6
					snap306 := d7
					snap307 := d8
					snap308 := d11
					snap309 := d22
					snap310 := d32
					snap311 := d33
					snap312 := d34
					snap313 := d35
					snap314 := d37
					snap315 := d38
					snap316 := d39
					snap317 := d40
					snap318 := d42
					snap319 := d43
					snap320 := d45
					snap321 := d46
					snap322 := d47
					snap323 := d48
					snap324 := d49
					snap325 := d52
					snap326 := d105
					snap327 := d106
					snap328 := d107
					snap329 := d108
					snap330 := d109
					snap331 := d111
					snap332 := d144
					snap333 := d147
					snap334 := d181
					snap335 := d182
					snap336 := d183
					snap337 := d184
					snap338 := d185
					snap339 := d186
					snap340 := d187
					snap341 := d188
					snap342 := d190
					snap343 := d191
					snap344 := d192
					snap345 := d194
					snap346 := d195
					snap347 := d196
					snap348 := d197
					snap349 := d198
					snap350 := d199
					snap351 := d200
					snap352 := d201
					snap353 := d202
					snap354 := d203
					snap355 := d204
					snap356 := d205
					snap357 := d206
					snap358 := d207
					snap359 := d208
					snap360 := d209
					snap361 := d210
					snap362 := d212
					snap363 := d213
					snap364 := d214
					snap365 := d215
					snap366 := d216
					snap367 := d218
					snap368 := d219
					snap369 := d220
					snap370 := d221
					snap371 := d223
					snap372 := d225
					snap373 := d298
					snap374 := d301
					alloc375 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps300)
					}
					ctx.RestoreAllocState(alloc375)
					d3 = snap302
					d4 = snap303
					d5 = snap304
					d6 = snap305
					d7 = snap306
					d8 = snap307
					d11 = snap308
					d22 = snap309
					d32 = snap310
					d33 = snap311
					d34 = snap312
					d35 = snap313
					d37 = snap314
					d38 = snap315
					d39 = snap316
					d40 = snap317
					d42 = snap318
					d43 = snap319
					d45 = snap320
					d46 = snap321
					d47 = snap322
					d48 = snap323
					d49 = snap324
					d52 = snap325
					d105 = snap326
					d106 = snap327
					d107 = snap328
					d108 = snap329
					d109 = snap330
					d111 = snap331
					d144 = snap332
					d147 = snap333
					d181 = snap334
					d182 = snap335
					d183 = snap336
					d184 = snap337
					d185 = snap338
					d186 = snap339
					d187 = snap340
					d188 = snap341
					d190 = snap342
					d191 = snap343
					d192 = snap344
					d194 = snap345
					d195 = snap346
					d196 = snap347
					d197 = snap348
					d198 = snap349
					d199 = snap350
					d200 = snap351
					d201 = snap352
					d202 = snap353
					d203 = snap354
					d204 = snap355
					d205 = snap356
					d206 = snap357
					d207 = snap358
					d208 = snap359
					d209 = snap360
					d210 = snap361
					d212 = snap362
					d213 = snap363
					d214 = snap364
					d215 = snap365
					d216 = snap366
					d218 = snap367
					d219 = snap368
					d220 = snap369
					d221 = snap370
					d223 = snap371
					d225 = snap372
					d298 = snap373
					d301 = snap374
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps299)
					}
					return result
					return result
				}
				ps376 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps376)
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitJump(d2.Condition, lbl6)
					ctx.EmitJmp(lbl7)
					snap5 := d0
					snap6 := d1
					snap7 := d2
					alloc8 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc8)
					d0 = snap5
					d1 = snap6
					d2 = snap7
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl3)
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
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					snap23 := d0
					snap24 := d1
					snap25 := d2
					snap26 := d15
					snap27 := d17
					snap28 := d18
					snap29 := d19
					snap30 := d20
					alloc31 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc31)
					d0 = snap23
					d1 = snap24
					d2 = snap25
					d15 = snap26
					d17 = snap27
					d18 = snap28
					d19 = snap29
					d20 = snap30
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
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
