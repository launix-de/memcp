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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d9 JITValueDesc
				_ = d9
				var d20 JITValueDesc
				_ = d20
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
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
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
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
				var d50 JITValueDesc
				_ = d50
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d108 JITValueDesc
				_ = d108
				var d140 JITValueDesc
				_ = d140
				var d143 JITValueDesc
				_ = d143
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
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
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var stackArray188 int32
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
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
				var d217 JITValueDesc
				_ = d217
				var d219 JITValueDesc
				_ = d219
				var d290 JITValueDesc
				_ = d290
				var d293 JITValueDesc
				_ = d293
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
					ctx.EmitJump(d6.Condition, lbl2)
					ctx.EmitJmp(lbl10)
					ctx.FreeDesc(&d5)
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d9
					alloc17 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc17)
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d9 = snap16
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d9 = snap16
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 10)
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[9] = d9
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 10)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[9] = d9
					ps19.PhiValues = make([]JITValueDesc, 1)
					d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps19.PhiValues[0] = d20
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d9
					snap28 := d20
					alloc29 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps19)
					}
					ctx.RestoreAllocState(alloc29)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d9 = snap27
					d20 = snap28
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps18)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					d30 = args[0]
					d30.ID = 0
					d32 = d30
					d32.ID = 0
					d31 = ctx.EmitBoolDesc(&d32, JITValueDesc{Loc: LocAny})
					ctx.StabilizeDescForControlFlow(&d31)
					ctx.FreeDesc(&d30)
					if ps.General {
						ctx.SyncDesc(&d31)
						if d31.Loc == LocReg {
							ctx.ProtectReg(d31.Reg)
						} else if d31.Loc == LocRegPair {
							ctx.ProtectReg(d31.Reg)
							ctx.ProtectReg(d31.Reg2)
						}
						d33 = d31
						if d33.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d33)
						ctx.EmitStoreToStack(d33, int32(bbs[2].PhiBase)+int32(0))
						if d31.Loc == LocReg {
							ctx.UnprotectReg(d31.Reg)
						} else if d31.Loc == LocRegPair {
							ctx.UnprotectReg(d31.Reg)
							ctx.UnprotectReg(d31.Reg2)
						}
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
					ps34.OverlayValues[20] = d20
					ps34.OverlayValues[30] = d30
					ps34.OverlayValues[31] = d31
					ps34.OverlayValues[32] = d32
					ps34.OverlayValues[33] = d33
					ps34.PhiValues = make([]JITValueDesc, 1)
					d35 = d31
					ps34.PhiValues[0] = d35
					if ps34.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps34)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d36 := ps.PhiValues[0]
							ctx.EnsureDesc(&d36)
							ctx.EmitStoreToStack(d36, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(Snapshot), []JITValueDesc{}, 3)
					d37.NoHeapPointer = false
					ctx.BindReg(d37.Reg, &d37)
					ctx.BindReg(d37.Reg2, &d37)
					ctx.BindReg(d37.Reg3, &d37)
					ctx.StabilizeDescForControlFlow(&d37)
					var d38 JITValueDesc
					if d37.SliceSizeKnown {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.KnownSliceLen))}
					} else if d37.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.StackOff))}
					} else if d37.Loc == LocStackTriple {
						d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d37.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d37)
						if d37.Loc == LocRegPair || d37.Loc == LocRegTriple {
							d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2, ID: 0}
						} else if d37.Loc == LocReg {
							d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					callResults39 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d38, d38}, []uint8{3}, []uint8{1})
					d40 = callResults39[0]
					d40.Type = tagSlice
					ctx.StabilizeDescForControlFlow(&d40)
					ctx.FreeDesc(&d38)
					var d41 JITValueDesc
					if d37.SliceSizeKnown {
						d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.KnownSliceLen))}
					} else if d37.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.StackOff))}
					} else if d37.Loc == LocStackTriple {
						d41 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d37.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d37)
						if d37.Loc == LocRegPair || d37.Loc == LocRegTriple {
							d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2, ID: 0}
						} else if d37.Loc == LocReg {
							d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d41)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[3].PhiBase)+int32(0))
					}
					ps42 := PhiState{General: ps.General}
					ps42.OverlayValues = make([]JITValueDesc, 42)
					ps42.OverlayValues[1] = d1
					ps42.OverlayValues[2] = d2
					ps42.OverlayValues[3] = d3
					ps42.OverlayValues[4] = d4
					ps42.OverlayValues[5] = d5
					ps42.OverlayValues[6] = d6
					ps42.OverlayValues[9] = d9
					ps42.OverlayValues[20] = d20
					ps42.OverlayValues[30] = d30
					ps42.OverlayValues[31] = d31
					ps42.OverlayValues[32] = d32
					ps42.OverlayValues[33] = d33
					ps42.OverlayValues[35] = d35
					ps42.OverlayValues[36] = d36
					ps42.OverlayValues[37] = d37
					ps42.OverlayValues[38] = d38
					ps42.OverlayValues[40] = d40
					ps42.OverlayValues[41] = d41
					ps42.PhiValues = make([]JITValueDesc, 1)
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps42.PhiValues[0] = d43
					if ps42.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps42)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d44 := ps.PhiValues[0]
							ctx.EnsureDesc(&d44)
							ctx.EmitStoreToStack(d44, int32(bbs[3].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d45 JITValueDesc
					if d2.Loc == LocImm {
						d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d45)
					}
					if d45.Loc == LocReg && d2.Loc == LocReg && d45.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d45)
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d45)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDescsTogether(&d45, &d41)
					var d46 JITValueDesc
					if d45.Loc == LocImm && d41.Loc == LocImm {
						d46 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d45.Imm.Int() < d41.Imm.Int())}
					} else if d41.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d45.Reg)
						if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d45.Reg, int32(d41.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d41.Imm.Int()))
							ctx.EmitCmpInt64(d45.Reg, RegR11)
						}
						d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d46)
					} else if d45.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d45.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d41.Reg)
						d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d46)
					} else {
						r3 := ctx.AllocRegExcept(d45.Reg)
						ctx.EmitCmpInt64(d45.Reg, d41.Reg)
						d46 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d46)
					}
					d47 = d46
					ctx.EnsureDesc(&d47)
					if d47.Loc != LocImm && d47.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d47.Loc == LocImm {
						if d47.Imm.Bool() {
							if ps.General {
							}
							ps48 := PhiState{General: ps.General}
							ps48.OverlayValues = make([]JITValueDesc, 48)
							ps48.OverlayValues[1] = d1
							ps48.OverlayValues[2] = d2
							ps48.OverlayValues[3] = d3
							ps48.OverlayValues[4] = d4
							ps48.OverlayValues[5] = d5
							ps48.OverlayValues[6] = d6
							ps48.OverlayValues[9] = d9
							ps48.OverlayValues[20] = d20
							ps48.OverlayValues[30] = d30
							ps48.OverlayValues[31] = d31
							ps48.OverlayValues[32] = d32
							ps48.OverlayValues[33] = d33
							ps48.OverlayValues[35] = d35
							ps48.OverlayValues[36] = d36
							ps48.OverlayValues[37] = d37
							ps48.OverlayValues[38] = d38
							ps48.OverlayValues[40] = d40
							ps48.OverlayValues[41] = d41
							ps48.OverlayValues[43] = d43
							ps48.OverlayValues[44] = d44
							ps48.OverlayValues[45] = d45
							ps48.OverlayValues[46] = d46
							ps48.OverlayValues[47] = d47
							return bbs[4].RenderPS(ps48)
						}
						if ps.General {
						}
						ps49 := PhiState{General: ps.General}
						ps49.OverlayValues = make([]JITValueDesc, 48)
						ps49.OverlayValues[1] = d1
						ps49.OverlayValues[2] = d2
						ps49.OverlayValues[3] = d3
						ps49.OverlayValues[4] = d4
						ps49.OverlayValues[5] = d5
						ps49.OverlayValues[6] = d6
						ps49.OverlayValues[9] = d9
						ps49.OverlayValues[20] = d20
						ps49.OverlayValues[30] = d30
						ps49.OverlayValues[31] = d31
						ps49.OverlayValues[32] = d32
						ps49.OverlayValues[33] = d33
						ps49.OverlayValues[35] = d35
						ps49.OverlayValues[36] = d36
						ps49.OverlayValues[37] = d37
						ps49.OverlayValues[38] = d38
						ps49.OverlayValues[40] = d40
						ps49.OverlayValues[41] = d41
						ps49.OverlayValues[43] = d43
						ps49.OverlayValues[44] = d44
						ps49.OverlayValues[45] = d45
						ps49.OverlayValues[46] = d46
						ps49.OverlayValues[47] = d47
						return bbs[5].RenderPS(ps49)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d50 := ps.PhiValues[0]
							ctx.EnsureDesc(&d50)
							ctx.EmitStoreToStack(d50, int32(bbs[3].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					ctx.EmitJump(d47.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d46)
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d9
					snap58 := d20
					snap59 := d30
					snap60 := d31
					snap61 := d32
					snap62 := d33
					snap63 := d35
					snap64 := d36
					snap65 := d37
					snap66 := d38
					snap67 := d40
					snap68 := d41
					snap69 := d43
					snap70 := d44
					snap71 := d45
					snap72 := d46
					snap73 := d47
					snap74 := d50
					alloc75 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc75)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d9 = snap57
					d20 = snap58
					d30 = snap59
					d31 = snap60
					d32 = snap61
					d33 = snap62
					d35 = snap63
					d36 = snap64
					d37 = snap65
					d38 = snap66
					d40 = snap67
					d41 = snap68
					d43 = snap69
					d44 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d50 = snap74
					ctx.RestoreAllocState(alloc75)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d9 = snap57
					d20 = snap58
					d30 = snap59
					d31 = snap60
					d32 = snap61
					d33 = snap62
					d35 = snap63
					d36 = snap64
					d37 = snap65
					d38 = snap66
					d40 = snap67
					d41 = snap68
					d43 = snap69
					d44 = snap70
					d45 = snap71
					d46 = snap72
					d47 = snap73
					d50 = snap74
					ps76 := PhiState{General: true}
					ps76.OverlayValues = make([]JITValueDesc, 51)
					ps76.OverlayValues[1] = d1
					ps76.OverlayValues[2] = d2
					ps76.OverlayValues[3] = d3
					ps76.OverlayValues[4] = d4
					ps76.OverlayValues[5] = d5
					ps76.OverlayValues[6] = d6
					ps76.OverlayValues[9] = d9
					ps76.OverlayValues[20] = d20
					ps76.OverlayValues[30] = d30
					ps76.OverlayValues[31] = d31
					ps76.OverlayValues[32] = d32
					ps76.OverlayValues[33] = d33
					ps76.OverlayValues[35] = d35
					ps76.OverlayValues[36] = d36
					ps76.OverlayValues[37] = d37
					ps76.OverlayValues[38] = d38
					ps76.OverlayValues[40] = d40
					ps76.OverlayValues[41] = d41
					ps76.OverlayValues[43] = d43
					ps76.OverlayValues[44] = d44
					ps76.OverlayValues[45] = d45
					ps76.OverlayValues[46] = d46
					ps76.OverlayValues[47] = d47
					ps76.OverlayValues[50] = d50
					ps77 := PhiState{General: true}
					ps77.OverlayValues = make([]JITValueDesc, 51)
					ps77.OverlayValues[1] = d1
					ps77.OverlayValues[2] = d2
					ps77.OverlayValues[3] = d3
					ps77.OverlayValues[4] = d4
					ps77.OverlayValues[5] = d5
					ps77.OverlayValues[6] = d6
					ps77.OverlayValues[9] = d9
					ps77.OverlayValues[20] = d20
					ps77.OverlayValues[30] = d30
					ps77.OverlayValues[31] = d31
					ps77.OverlayValues[32] = d32
					ps77.OverlayValues[33] = d33
					ps77.OverlayValues[35] = d35
					ps77.OverlayValues[36] = d36
					ps77.OverlayValues[37] = d37
					ps77.OverlayValues[38] = d38
					ps77.OverlayValues[40] = d40
					ps77.OverlayValues[41] = d41
					ps77.OverlayValues[43] = d43
					ps77.OverlayValues[44] = d44
					ps77.OverlayValues[45] = d45
					ps77.OverlayValues[46] = d46
					ps77.OverlayValues[47] = d47
					ps77.OverlayValues[50] = d50
					snap78 := d1
					snap79 := d2
					snap80 := d3
					snap81 := d4
					snap82 := d5
					snap83 := d6
					snap84 := d9
					snap85 := d20
					snap86 := d30
					snap87 := d31
					snap88 := d32
					snap89 := d33
					snap90 := d35
					snap91 := d36
					snap92 := d37
					snap93 := d38
					snap94 := d40
					snap95 := d41
					snap96 := d43
					snap97 := d44
					snap98 := d45
					snap99 := d46
					snap100 := d47
					snap101 := d50
					alloc102 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps77)
					}
					ctx.RestoreAllocState(alloc102)
					d1 = snap78
					d2 = snap79
					d3 = snap80
					d4 = snap81
					d5 = snap82
					d6 = snap83
					d9 = snap84
					d20 = snap85
					d30 = snap86
					d31 = snap87
					d32 = snap88
					d33 = snap89
					d35 = snap90
					d36 = snap91
					d37 = snap92
					d38 = snap93
					d40 = snap94
					d41 = snap95
					d43 = snap96
					d44 = snap97
					d45 = snap98
					d46 = snap99
					d47 = snap100
					d50 = snap101
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps76)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d45)
					d103 = ctx.EmitLoadScalarSliceElement(&d37, &d45, 8, JITTypeUnknown)
					ctx.StabilizeDescForControlFlow(&d103)
					if d103.Loc == LocRegPair || d103.Loc == LocStackPair || d103.Loc == LocRegTriple || d103.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d104 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d103}, 2)
					d104.NoHeapPointer = false
					ctx.BindReg(d104.Reg, &d104)
					ctx.BindReg(d104.Reg2, &d104)
					ctx.StabilizeDescForControlFlow(&d104)
					d105 = d1
					ctx.EnsureDesc(&d105)
					if d105.Loc != LocImm && d105.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d105.Loc == LocImm {
						if d105.Imm.Bool() {
							if ps.General {
								ctx.SyncDesc(&d104)
								if d104.Loc == LocReg {
									ctx.ProtectReg(d104.Reg)
								} else if d104.Loc == LocRegPair {
									ctx.ProtectReg(d104.Reg)
									ctx.ProtectReg(d104.Reg2)
								}
								d106 = d104
								if d106.Loc == LocNone {
									panic("jit: phi source has no location")
								}
								ctx.SyncDesc(&d106)
								if d106.Loc == LocStackPair {
									ctx.EmitCopyStackWords(d106, int32(bbs[7].PhiBase)+int32(0), 2)
								} else if d106.Loc == LocInputPair {
									ctx.EnsureDesc(&d106)
									ctx.EmitStoreScmerToStack(d106, int32(bbs[7].PhiBase)+int32(0))
								} else if d106.Loc == LocRegPair || d106.Loc == LocImm {
									ctx.EmitStoreScmerToStack(d106, int32(bbs[7].PhiBase)+int32(0))
								} else {
									ctx.EnsureDesc(&d106)
									ctx.EmitStoreToStack(d106, int32(bbs[7].PhiBase)+int32(0))
									ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
								}
								if d104.Loc == LocReg {
									ctx.UnprotectReg(d104.Reg)
								} else if d104.Loc == LocRegPair {
									ctx.UnprotectReg(d104.Reg)
									ctx.UnprotectReg(d104.Reg2)
								}
							}
							ps107 := PhiState{General: ps.General}
							ps107.OverlayValues = make([]JITValueDesc, 107)
							ps107.OverlayValues[1] = d1
							ps107.OverlayValues[2] = d2
							ps107.OverlayValues[3] = d3
							ps107.OverlayValues[4] = d4
							ps107.OverlayValues[5] = d5
							ps107.OverlayValues[6] = d6
							ps107.OverlayValues[9] = d9
							ps107.OverlayValues[20] = d20
							ps107.OverlayValues[30] = d30
							ps107.OverlayValues[31] = d31
							ps107.OverlayValues[32] = d32
							ps107.OverlayValues[33] = d33
							ps107.OverlayValues[35] = d35
							ps107.OverlayValues[36] = d36
							ps107.OverlayValues[37] = d37
							ps107.OverlayValues[38] = d38
							ps107.OverlayValues[40] = d40
							ps107.OverlayValues[41] = d41
							ps107.OverlayValues[43] = d43
							ps107.OverlayValues[44] = d44
							ps107.OverlayValues[45] = d45
							ps107.OverlayValues[46] = d46
							ps107.OverlayValues[47] = d47
							ps107.OverlayValues[50] = d50
							ps107.OverlayValues[103] = d103
							ps107.OverlayValues[104] = d104
							ps107.OverlayValues[105] = d105
							ps107.OverlayValues[106] = d106
							ps107.PhiValues = make([]JITValueDesc, 1)
							d108 = d104
							ps107.PhiValues[0] = d108
							return bbs[7].RenderPS(ps107)
						}
						if ps.General {
						}
						ps109 := PhiState{General: ps.General}
						ps109.OverlayValues = make([]JITValueDesc, 109)
						ps109.OverlayValues[1] = d1
						ps109.OverlayValues[2] = d2
						ps109.OverlayValues[3] = d3
						ps109.OverlayValues[4] = d4
						ps109.OverlayValues[5] = d5
						ps109.OverlayValues[6] = d6
						ps109.OverlayValues[9] = d9
						ps109.OverlayValues[20] = d20
						ps109.OverlayValues[30] = d30
						ps109.OverlayValues[31] = d31
						ps109.OverlayValues[32] = d32
						ps109.OverlayValues[33] = d33
						ps109.OverlayValues[35] = d35
						ps109.OverlayValues[36] = d36
						ps109.OverlayValues[37] = d37
						ps109.OverlayValues[38] = d38
						ps109.OverlayValues[40] = d40
						ps109.OverlayValues[41] = d41
						ps109.OverlayValues[43] = d43
						ps109.OverlayValues[44] = d44
						ps109.OverlayValues[45] = d45
						ps109.OverlayValues[46] = d46
						ps109.OverlayValues[47] = d47
						ps109.OverlayValues[50] = d50
						ps109.OverlayValues[103] = d103
						ps109.OverlayValues[104] = d104
						ps109.OverlayValues[105] = d105
						ps109.OverlayValues[106] = d106
						ps109.OverlayValues[108] = d108
						return bbs[8].RenderPS(ps109)
					}
					if !ps.General {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d105.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl9)
					snap110 := d1
					snap111 := d2
					snap112 := d3
					snap113 := d4
					snap114 := d5
					snap115 := d6
					snap116 := d9
					snap117 := d20
					snap118 := d30
					snap119 := d31
					snap120 := d32
					snap121 := d33
					snap122 := d35
					snap123 := d36
					snap124 := d37
					snap125 := d38
					snap126 := d40
					snap127 := d41
					snap128 := d43
					snap129 := d44
					snap130 := d45
					snap131 := d46
					snap132 := d47
					snap133 := d50
					snap134 := d103
					snap135 := d104
					snap136 := d105
					snap137 := d106
					snap138 := d108
					alloc139 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.SyncDesc(&d104)
					if d104.Loc == LocReg {
						ctx.ProtectReg(d104.Reg)
					} else if d104.Loc == LocRegPair {
						ctx.ProtectReg(d104.Reg)
						ctx.ProtectReg(d104.Reg2)
					}
					d140 = d104
					if d140.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d140)
					if d140.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d140, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d140.Loc == LocInputPair {
						ctx.EnsureDesc(&d140)
						ctx.EmitStoreScmerToStack(d140, int32(bbs[7].PhiBase)+int32(0))
					} else if d140.Loc == LocRegPair || d140.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d140, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d140)
						ctx.EmitStoreToStack(d140, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d104.Loc == LocReg {
						ctx.UnprotectReg(d104.Reg)
					} else if d104.Loc == LocRegPair {
						ctx.UnprotectReg(d104.Reg)
						ctx.UnprotectReg(d104.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc139)
					d1 = snap110
					d2 = snap111
					d3 = snap112
					d4 = snap113
					d5 = snap114
					d6 = snap115
					d9 = snap116
					d20 = snap117
					d30 = snap118
					d31 = snap119
					d32 = snap120
					d33 = snap121
					d35 = snap122
					d36 = snap123
					d37 = snap124
					d38 = snap125
					d40 = snap126
					d41 = snap127
					d43 = snap128
					d44 = snap129
					d45 = snap130
					d46 = snap131
					d47 = snap132
					d50 = snap133
					d103 = snap134
					d104 = snap135
					d105 = snap136
					d106 = snap137
					d108 = snap138
					ctx.RestoreAllocState(alloc139)
					d1 = snap110
					d2 = snap111
					d3 = snap112
					d4 = snap113
					d5 = snap114
					d6 = snap115
					d9 = snap116
					d20 = snap117
					d30 = snap118
					d31 = snap119
					d32 = snap120
					d33 = snap121
					d35 = snap122
					d36 = snap123
					d37 = snap124
					d38 = snap125
					d40 = snap126
					d41 = snap127
					d43 = snap128
					d44 = snap129
					d45 = snap130
					d46 = snap131
					d47 = snap132
					d50 = snap133
					d103 = snap134
					d104 = snap135
					d105 = snap136
					d106 = snap137
					d108 = snap138
					ps141 := PhiState{General: true}
					ps141.OverlayValues = make([]JITValueDesc, 141)
					ps141.OverlayValues[1] = d1
					ps141.OverlayValues[2] = d2
					ps141.OverlayValues[3] = d3
					ps141.OverlayValues[4] = d4
					ps141.OverlayValues[5] = d5
					ps141.OverlayValues[6] = d6
					ps141.OverlayValues[9] = d9
					ps141.OverlayValues[20] = d20
					ps141.OverlayValues[30] = d30
					ps141.OverlayValues[31] = d31
					ps141.OverlayValues[32] = d32
					ps141.OverlayValues[33] = d33
					ps141.OverlayValues[35] = d35
					ps141.OverlayValues[36] = d36
					ps141.OverlayValues[37] = d37
					ps141.OverlayValues[38] = d38
					ps141.OverlayValues[40] = d40
					ps141.OverlayValues[41] = d41
					ps141.OverlayValues[43] = d43
					ps141.OverlayValues[44] = d44
					ps141.OverlayValues[45] = d45
					ps141.OverlayValues[46] = d46
					ps141.OverlayValues[47] = d47
					ps141.OverlayValues[50] = d50
					ps141.OverlayValues[103] = d103
					ps141.OverlayValues[104] = d104
					ps141.OverlayValues[105] = d105
					ps141.OverlayValues[106] = d106
					ps141.OverlayValues[108] = d108
					ps141.OverlayValues[140] = d140
					ps141.PhiValues = make([]JITValueDesc, 1)
					d143 = d104
					ps141.PhiValues[0] = d143
					ps142 := PhiState{General: true}
					ps142.OverlayValues = make([]JITValueDesc, 144)
					ps142.OverlayValues[1] = d1
					ps142.OverlayValues[2] = d2
					ps142.OverlayValues[3] = d3
					ps142.OverlayValues[4] = d4
					ps142.OverlayValues[5] = d5
					ps142.OverlayValues[6] = d6
					ps142.OverlayValues[9] = d9
					ps142.OverlayValues[20] = d20
					ps142.OverlayValues[30] = d30
					ps142.OverlayValues[31] = d31
					ps142.OverlayValues[32] = d32
					ps142.OverlayValues[33] = d33
					ps142.OverlayValues[35] = d35
					ps142.OverlayValues[36] = d36
					ps142.OverlayValues[37] = d37
					ps142.OverlayValues[38] = d38
					ps142.OverlayValues[40] = d40
					ps142.OverlayValues[41] = d41
					ps142.OverlayValues[43] = d43
					ps142.OverlayValues[44] = d44
					ps142.OverlayValues[45] = d45
					ps142.OverlayValues[46] = d46
					ps142.OverlayValues[47] = d47
					ps142.OverlayValues[50] = d50
					ps142.OverlayValues[103] = d103
					ps142.OverlayValues[104] = d104
					ps142.OverlayValues[105] = d105
					ps142.OverlayValues[106] = d106
					ps142.OverlayValues[108] = d108
					ps142.OverlayValues[140] = d140
					ps142.OverlayValues[143] = d143
					snap144 := d1
					snap145 := d2
					snap146 := d3
					snap147 := d4
					snap148 := d5
					snap149 := d6
					snap150 := d9
					snap151 := d20
					snap152 := d30
					snap153 := d31
					snap154 := d32
					snap155 := d33
					snap156 := d35
					snap157 := d36
					snap158 := d37
					snap159 := d38
					snap160 := d40
					snap161 := d41
					snap162 := d43
					snap163 := d44
					snap164 := d45
					snap165 := d46
					snap166 := d47
					snap167 := d50
					snap168 := d103
					snap169 := d104
					snap170 := d105
					snap171 := d106
					snap172 := d108
					snap173 := d140
					snap174 := d143
					alloc175 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps141)
					}
					ctx.RestoreAllocState(alloc175)
					d1 = snap144
					d2 = snap145
					d3 = snap146
					d4 = snap147
					d5 = snap148
					d6 = snap149
					d9 = snap150
					d20 = snap151
					d30 = snap152
					d31 = snap153
					d32 = snap154
					d33 = snap155
					d35 = snap156
					d36 = snap157
					d37 = snap158
					d38 = snap159
					d40 = snap160
					d41 = snap161
					d43 = snap162
					d44 = snap163
					d45 = snap164
					d46 = snap165
					d47 = snap166
					d50 = snap167
					d103 = snap168
					d104 = snap169
					d105 = snap170
					d106 = snap171
					d108 = snap172
					d140 = snap173
					d143 = snap174
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps142)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d40)
					d176 = ctx.EmitNewSliceFromGoSlice(&d40)
					ctx.SyncDesc(&d176)
					if d176.Loc == LocRegPair || d176.Loc == LocStackPair || d176.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d176, &result)
						result.Type = d176.Type
					} else {
						switch d176.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d176)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d176)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d176)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d176, &result)
							result.Type = d176.Type
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					ctx.ReclaimUntrackedRegs()
					d177 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d178 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(100)}
					ctx.EnsureDesc(&d104)
					ctx.EnsureDesc(&d177)
					ctx.EnsureDesc(&d178)
					var d180 JITValueDesc
					if d178.Loc == LocImm && d177.Loc == LocImm {
						d180 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d178.Imm.Int() - d177.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d178.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d178.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d178.Reg)
						}
						if d177.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d177.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d177.Reg)
						}
						d180 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d180)
					}
					var d181 JITValueDesc
					r5 := ctx.EmitSliceDataAfterLow(&d104, &d177, 1)
					d181 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d181)
					ctx.BindReg(r5, &d181)
					var d182 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d181)
					ctx.EnsureDesc(&d181)
					if d181.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d181.Imm.Int()))
					} else {
						r6 = d181.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d180)
					ctx.EnsureDesc(&d180)
					if d180.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d180.Imm.Int()))
					} else {
						r7 = d180.Reg
					}
					ctx.ProtectReg(r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d182 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d182)
					ctx.BindReg(r7, &d182)
					ctx.BindReg(r6, &d182)
					ctx.BindReg(r7, &d182)
					ctx.StabilizeDescForControlFlow(&d182)
					if ps.General {
						ctx.SyncDesc(&d182)
						if d182.Loc == LocReg {
							ctx.ProtectReg(d182.Reg)
						} else if d182.Loc == LocRegPair {
							ctx.ProtectReg(d182.Reg)
							ctx.ProtectReg(d182.Reg2)
						}
						d183 = d182
						if d183.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d183)
						if d183.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d183, int32(bbs[7].PhiBase)+int32(0), 2)
						} else if d183.Loc == LocInputPair {
							ctx.EnsureDesc(&d183)
							ctx.EmitStoreScmerToStack(d183, int32(bbs[7].PhiBase)+int32(0))
						} else if d183.Loc == LocRegPair || d183.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d183, int32(bbs[7].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d183)
							ctx.EmitStoreToStack(d183, int32(bbs[7].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
						}
						if d182.Loc == LocReg {
							ctx.UnprotectReg(d182.Reg)
						} else if d182.Loc == LocRegPair {
							ctx.UnprotectReg(d182.Reg)
							ctx.UnprotectReg(d182.Reg2)
						}
					}
					ps184 := PhiState{General: ps.General}
					ps184.OverlayValues = make([]JITValueDesc, 184)
					ps184.OverlayValues[1] = d1
					ps184.OverlayValues[2] = d2
					ps184.OverlayValues[3] = d3
					ps184.OverlayValues[4] = d4
					ps184.OverlayValues[5] = d5
					ps184.OverlayValues[6] = d6
					ps184.OverlayValues[9] = d9
					ps184.OverlayValues[20] = d20
					ps184.OverlayValues[30] = d30
					ps184.OverlayValues[31] = d31
					ps184.OverlayValues[32] = d32
					ps184.OverlayValues[33] = d33
					ps184.OverlayValues[35] = d35
					ps184.OverlayValues[36] = d36
					ps184.OverlayValues[37] = d37
					ps184.OverlayValues[38] = d38
					ps184.OverlayValues[40] = d40
					ps184.OverlayValues[41] = d41
					ps184.OverlayValues[43] = d43
					ps184.OverlayValues[44] = d44
					ps184.OverlayValues[45] = d45
					ps184.OverlayValues[46] = d46
					ps184.OverlayValues[47] = d47
					ps184.OverlayValues[50] = d50
					ps184.OverlayValues[103] = d103
					ps184.OverlayValues[104] = d104
					ps184.OverlayValues[105] = d105
					ps184.OverlayValues[106] = d106
					ps184.OverlayValues[108] = d108
					ps184.OverlayValues[140] = d140
					ps184.OverlayValues[143] = d143
					ps184.OverlayValues[176] = d176
					ps184.OverlayValues[177] = d177
					ps184.OverlayValues[178] = d178
					ps184.OverlayValues[179] = d179
					ps184.OverlayValues[180] = d180
					ps184.OverlayValues[181] = d181
					ps184.OverlayValues[182] = d182
					ps184.OverlayValues[183] = d183
					ps184.PhiValues = make([]JITValueDesc, 1)
					d185 = d182
					ps184.PhiValues[0] = d185
					if ps184.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps184)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d186 := ps.PhiValues[0]
							ctx.EnsureDesc(&d186)
							ctx.EmitStoreScmerToStack(d186, int32(bbs[7].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
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
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d40)
					if d103.Loc == LocRegPair || d103.Loc == LocStackPair || d103.Loc == LocRegTriple || d103.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d103)
					d187 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).processListState), []JITValueDesc{d103}, 2)
					d187.NoHeapPointer = false
					ctx.BindReg(d187.Reg, &d187)
					ctx.BindReg(d187.Reg2, &d187)
					stackArray188 = ctx.AllocStack(int32(256))
					_ = stackArray188
					d189 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Id")}
					ctx.SyncDesc(&d189)
					ctx.EmitStoreScmerToStack(d189, int32(stackArray188)+int32(0))
					var d190 JITValueDesc
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						fieldAddr := uintptr(d103.Imm.Int()) + 0
						r8 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r8, fieldAddr)
						d190 = JITValueDesc{Loc: LocReg, Reg: r8}
						ctx.BindReg(r8, &d190)
					} else {
						off := int32(0)
						baseReg := d103.Reg
						r9 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r9, baseReg, off)
						d190 = JITValueDesc{Loc: LocReg, Reg: r9}
						ctx.BindReg(r9, &d190)
					}
					ctx.EnsureDesc(&d190)
					ctx.EnsureDesc(&d190)
					var d191 JITValueDesc
					if d190.Loc == LocImm {
						d191 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(int64(uint64(d190.Imm.Int()))))}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegReg(r10, d190.Reg)
						d191 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d191)
					}
					ctx.FreeDesc(&d190)
					ctx.EnsureDesc(&d191)
					ctx.SyncDesc(&d191)
					ctx.EnsureDesc(&d191)
					ctx.EmitStoreTypedScmerToStack(d191, tagInt, int32(stackArray188)+int32(16))
					d192 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("User")}
					ctx.SyncDesc(&d192)
					ctx.EmitStoreScmerToStack(d192, int32(stackArray188)+int32(32))
					var d193 JITValueDesc
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						fieldAddr := uintptr(d103.Imm.Int()) + 8
						r11 := ctx.AllocReg()
						r12 := ctx.AllocRegExcept(r11)
						r13 := ctx.AllocRegExcept(r11, r12)
						ctx.EmitMovRegMem64(r11, fieldAddr)
						ctx.EmitMovRegMem64(r12, fieldAddr+8)
						ctx.EmitMovRegMem64(r13, fieldAddr+16)
						d193 = JITValueDesc{Loc: LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
						ctx.BindReg(r11, &d193)
						ctx.BindReg(r12, &d193)
						ctx.BindReg(r13, &d193)
					} else {
						off := int32(8)
						baseReg := d103.Reg
						r14 := ctx.AllocRegExcept(baseReg)
						r15 := ctx.AllocRegExcept(baseReg, r14)
						r16 := ctx.AllocRegExcept(baseReg, r14, r15)
						ctx.EmitMovRegMem(r14, baseReg, off)
						ctx.EmitMovRegMem(r15, baseReg, off+8)
						ctx.EmitMovRegMem(r16, baseReg, off+16)
						d193 = JITValueDesc{Loc: LocRegTriple, Reg: r14, Reg2: r15, Reg3: r16}
						ctx.BindReg(r14, &d193)
						ctx.BindReg(r15, &d193)
						ctx.BindReg(r16, &d193)
					}
					ctx.EnsureDesc(&d193)
					ctx.SyncDesc(&d193)
					ctx.EmitStoreScmerToStack(d193, int32(stackArray188)+int32(48))
					d194 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Host")}
					ctx.SyncDesc(&d194)
					ctx.EmitStoreScmerToStack(d194, int32(stackArray188)+int32(64))
					var d195 JITValueDesc
					ctx.EnsureDesc(&d103)
					if d103.Loc == LocImm {
						fieldAddr := uintptr(d103.Imm.Int()) + 24
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d195 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d195)
						ctx.BindReg(r18, &d195)
						ctx.BindReg(r19, &d195)
					} else {
						off := int32(24)
						baseReg := d103.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d195 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d195)
						ctx.BindReg(r21, &d195)
						ctx.BindReg(r22, &d195)
					}
					ctx.EnsureDesc(&d195)
					ctx.SyncDesc(&d195)
					ctx.EmitStoreScmerToStack(d195, int32(stackArray188)+int32(80))
					d196 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("db")}
					ctx.SyncDesc(&d196)
					ctx.EmitStoreScmerToStack(d196, int32(stackArray188)+int32(96))
					if d103.Loc == LocRegPair || d103.Loc == LocStackPair || d103.Loc == LocRegTriple || d103.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d197 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d103}, 2)
					d197.NoHeapPointer = false
					ctx.BindReg(d197.Reg, &d197)
					ctx.BindReg(d197.Reg2, &d197)
					ctx.EnsureDesc(&d197)
					ctx.SyncDesc(&d197)
					ctx.EmitStoreScmerToStack(d197, int32(stackArray188)+int32(112))
					d198 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Command")}
					ctx.SyncDesc(&d198)
					ctx.EmitStoreScmerToStack(d198, int32(stackArray188)+int32(128))
					if d103.Loc == LocRegPair || d103.Loc == LocStackPair || d103.Loc == LocRegTriple || d103.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d199 = ctx.EmitGoCallScalar(GoFuncAddr(strPtr), []JITValueDesc{d103}, 2)
					d199.NoHeapPointer = false
					ctx.BindReg(d199.Reg, &d199)
					ctx.BindReg(d199.Reg2, &d199)
					ctx.EnsureDesc(&d199)
					ctx.SyncDesc(&d199)
					ctx.EmitStoreScmerToStack(d199, int32(stackArray188)+int32(144))
					d200 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Time")}
					ctx.SyncDesc(&d200)
					ctx.EmitStoreScmerToStack(d200, int32(stackArray188)+int32(160))
					if d103.Loc == LocRegPair || d103.Loc == LocStackPair || d103.Loc == LocRegTriple || d103.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d103)
					d201 = ctx.EmitGoCallScalar(GoFuncAddr((*SessionState).ElapsedSeconds), []JITValueDesc{d103}, 1)
					d201.NoHeapPointer = true
					ctx.BindReg(d201.Reg, &d201)
					ctx.EnsureDesc(&d201)
					ctx.SyncDesc(&d201)
					ctx.EnsureDesc(&d201)
					ctx.EmitStoreTypedScmerToStack(d201, tagInt, int32(stackArray188)+int32(176))
					d202 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("State")}
					ctx.SyncDesc(&d202)
					ctx.EmitStoreScmerToStack(d202, int32(stackArray188)+int32(192))
					ctx.EnsureDesc(&d187)
					ctx.SyncDesc(&d187)
					ctx.EmitStoreScmerToStack(d187, int32(stackArray188)+int32(208))
					d203 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("Info")}
					ctx.SyncDesc(&d203)
					ctx.EmitStoreScmerToStack(d203, int32(stackArray188)+int32(224))
					ctx.EnsureDesc(&d3)
					ctx.SyncDesc(&d3)
					ctx.EmitStoreScmerToStack(d3, int32(stackArray188)+int32(240))
					d204 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(16), KnownSliceCap: int32(16), SliceSizeKnown: true}
					_ = d204
					r23 := ctx.AllocReg()
					r24 := ctx.AllocRegExcept(r23)
					r25 := ctx.AllocRegExcept(r23, r24)
					d205 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r23, Reg2: r24, Reg3: r25}
					ctx.BindReg(r23, &d205)
					ctx.BindReg(r24, &d205)
					ctx.BindReg(r25, &d205)
					ctx.BindReg(r23, &d205)
					ctx.BindReg(r24, &d205)
					ctx.BindReg(r25, &d205)
					ctx.EmitLeaRegMem(d205.Reg, ctx.StackReg, int32(stackArray188))
					ctx.EmitMovRegImm64(d205.Reg2, uint64(16))
					ctx.EmitMovRegImm64(d205.Reg3, uint64(16))
					callResults206 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d205}, []uint8{2}, []uint8{1})
					d207 = callResults206[0]
					ctx.EnsureDesc(&d45)
					ctx.SyncDesc(&d207)
					d208 = d40
					d208.ID = 0
					d209 = d45
					d209.ID = 0
					if !ctx.TryEmitStoreScmerSliceElement(&d208, &d209, &d207, int32(16)) {
						ctx.StabilizeDescAcrossNestedCall(&d45)
						d209 = d45
						d209.ID = 0
						ctx.EmitStoreScmerSliceElement(&d208, &d209, &d207, int32(16))
					}
					ctx.FreeDesc(&d209)
					ctx.FreeDesc(&d207)
					if ps.General {
						ctx.SyncDesc(&d45)
						if d45.Loc == LocReg {
							ctx.ProtectReg(d45.Reg)
						} else if d45.Loc == LocRegPair {
							ctx.ProtectReg(d45.Reg)
							ctx.ProtectReg(d45.Reg2)
						}
						d210 = d45
						if d210.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d210)
						ctx.EmitStoreToStack(d210, int32(bbs[3].PhiBase)+int32(0))
						if d45.Loc == LocReg {
							ctx.UnprotectReg(d45.Reg)
						} else if d45.Loc == LocRegPair {
							ctx.UnprotectReg(d45.Reg)
							ctx.UnprotectReg(d45.Reg2)
						}
					}
					ps211 := PhiState{General: ps.General}
					ps211.OverlayValues = make([]JITValueDesc, 211)
					ps211.OverlayValues[1] = d1
					ps211.OverlayValues[2] = d2
					ps211.OverlayValues[3] = d3
					ps211.OverlayValues[4] = d4
					ps211.OverlayValues[5] = d5
					ps211.OverlayValues[6] = d6
					ps211.OverlayValues[9] = d9
					ps211.OverlayValues[20] = d20
					ps211.OverlayValues[30] = d30
					ps211.OverlayValues[31] = d31
					ps211.OverlayValues[32] = d32
					ps211.OverlayValues[33] = d33
					ps211.OverlayValues[35] = d35
					ps211.OverlayValues[36] = d36
					ps211.OverlayValues[37] = d37
					ps211.OverlayValues[38] = d38
					ps211.OverlayValues[40] = d40
					ps211.OverlayValues[41] = d41
					ps211.OverlayValues[43] = d43
					ps211.OverlayValues[44] = d44
					ps211.OverlayValues[45] = d45
					ps211.OverlayValues[46] = d46
					ps211.OverlayValues[47] = d47
					ps211.OverlayValues[50] = d50
					ps211.OverlayValues[103] = d103
					ps211.OverlayValues[104] = d104
					ps211.OverlayValues[105] = d105
					ps211.OverlayValues[106] = d106
					ps211.OverlayValues[108] = d108
					ps211.OverlayValues[140] = d140
					ps211.OverlayValues[143] = d143
					ps211.OverlayValues[176] = d176
					ps211.OverlayValues[177] = d177
					ps211.OverlayValues[178] = d178
					ps211.OverlayValues[179] = d179
					ps211.OverlayValues[180] = d180
					ps211.OverlayValues[181] = d181
					ps211.OverlayValues[182] = d182
					ps211.OverlayValues[183] = d183
					ps211.OverlayValues[185] = d185
					ps211.OverlayValues[186] = d186
					ps211.OverlayValues[187] = d187
					ps211.OverlayValues[189] = d189
					ps211.OverlayValues[190] = d190
					ps211.OverlayValues[191] = d191
					ps211.OverlayValues[192] = d192
					ps211.OverlayValues[193] = d193
					ps211.OverlayValues[194] = d194
					ps211.OverlayValues[195] = d195
					ps211.OverlayValues[196] = d196
					ps211.OverlayValues[197] = d197
					ps211.OverlayValues[198] = d198
					ps211.OverlayValues[199] = d199
					ps211.OverlayValues[200] = d200
					ps211.OverlayValues[201] = d201
					ps211.OverlayValues[202] = d202
					ps211.OverlayValues[203] = d203
					ps211.OverlayValues[204] = d204
					ps211.OverlayValues[205] = d205
					ps211.OverlayValues[207] = d207
					ps211.OverlayValues[208] = d208
					ps211.OverlayValues[209] = d209
					ps211.OverlayValues[210] = d210
					ps211.PhiValues = make([]JITValueDesc, 1)
					d212 = d45
					ps211.PhiValues[0] = d212
					if ps211.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps211)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
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
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
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
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
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
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
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
					ctx.ReclaimUntrackedRegs()
					var d213 JITValueDesc
					if d104.SliceSizeKnown {
						d213 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d104.KnownSliceLen))}
					} else if d104.Loc == LocImm {
						d213 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(d104.Imm.String())))}
					} else if d104.Loc == LocStackTriple {
						d213 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d104.StackOff + 8, NoHeapPointer: true}
					} else if d104.Loc == LocStackPair {
						d213 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d104.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d104)
						if d104.Loc == LocRegPair || d104.Loc == LocRegTriple {
							d213 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d104.Reg2, ID: 0}
						} else if d104.Loc == LocReg {
							d213 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d104.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d213)
					var d214 JITValueDesc
					if d213.Loc == LocImm {
						d214 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d213.Imm.Int() > 100)}
					} else {
						r26 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d213.Reg, 100)
						d214 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedGreater}
						ctx.BindReg(r26, &d214)
					}
					ctx.FreeDesc(&d213)
					d215 = d214
					ctx.EnsureDesc(&d215)
					if d215.Loc != LocImm && d215.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d215.Loc == LocImm {
						if d215.Imm.Bool() {
							if ps.General {
							}
							ps216 := PhiState{General: ps.General}
							ps216.OverlayValues = make([]JITValueDesc, 216)
							ps216.OverlayValues[1] = d1
							ps216.OverlayValues[2] = d2
							ps216.OverlayValues[3] = d3
							ps216.OverlayValues[4] = d4
							ps216.OverlayValues[5] = d5
							ps216.OverlayValues[6] = d6
							ps216.OverlayValues[9] = d9
							ps216.OverlayValues[20] = d20
							ps216.OverlayValues[30] = d30
							ps216.OverlayValues[31] = d31
							ps216.OverlayValues[32] = d32
							ps216.OverlayValues[33] = d33
							ps216.OverlayValues[35] = d35
							ps216.OverlayValues[36] = d36
							ps216.OverlayValues[37] = d37
							ps216.OverlayValues[38] = d38
							ps216.OverlayValues[40] = d40
							ps216.OverlayValues[41] = d41
							ps216.OverlayValues[43] = d43
							ps216.OverlayValues[44] = d44
							ps216.OverlayValues[45] = d45
							ps216.OverlayValues[46] = d46
							ps216.OverlayValues[47] = d47
							ps216.OverlayValues[50] = d50
							ps216.OverlayValues[103] = d103
							ps216.OverlayValues[104] = d104
							ps216.OverlayValues[105] = d105
							ps216.OverlayValues[106] = d106
							ps216.OverlayValues[108] = d108
							ps216.OverlayValues[140] = d140
							ps216.OverlayValues[143] = d143
							ps216.OverlayValues[176] = d176
							ps216.OverlayValues[177] = d177
							ps216.OverlayValues[178] = d178
							ps216.OverlayValues[179] = d179
							ps216.OverlayValues[180] = d180
							ps216.OverlayValues[181] = d181
							ps216.OverlayValues[182] = d182
							ps216.OverlayValues[183] = d183
							ps216.OverlayValues[185] = d185
							ps216.OverlayValues[186] = d186
							ps216.OverlayValues[187] = d187
							ps216.OverlayValues[189] = d189
							ps216.OverlayValues[190] = d190
							ps216.OverlayValues[191] = d191
							ps216.OverlayValues[192] = d192
							ps216.OverlayValues[193] = d193
							ps216.OverlayValues[194] = d194
							ps216.OverlayValues[195] = d195
							ps216.OverlayValues[196] = d196
							ps216.OverlayValues[197] = d197
							ps216.OverlayValues[198] = d198
							ps216.OverlayValues[199] = d199
							ps216.OverlayValues[200] = d200
							ps216.OverlayValues[201] = d201
							ps216.OverlayValues[202] = d202
							ps216.OverlayValues[203] = d203
							ps216.OverlayValues[204] = d204
							ps216.OverlayValues[205] = d205
							ps216.OverlayValues[207] = d207
							ps216.OverlayValues[208] = d208
							ps216.OverlayValues[209] = d209
							ps216.OverlayValues[210] = d210
							ps216.OverlayValues[212] = d212
							ps216.OverlayValues[213] = d213
							ps216.OverlayValues[214] = d214
							ps216.OverlayValues[215] = d215
							return bbs[6].RenderPS(ps216)
						}
						if ps.General {
							ctx.SyncDesc(&d104)
							if d104.Loc == LocReg {
								ctx.ProtectReg(d104.Reg)
							} else if d104.Loc == LocRegPair {
								ctx.ProtectReg(d104.Reg)
								ctx.ProtectReg(d104.Reg2)
							}
							d217 = d104
							if d217.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d217)
							if d217.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d217, int32(bbs[7].PhiBase)+int32(0), 2)
							} else if d217.Loc == LocInputPair {
								ctx.EnsureDesc(&d217)
								ctx.EmitStoreScmerToStack(d217, int32(bbs[7].PhiBase)+int32(0))
							} else if d217.Loc == LocRegPair || d217.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d217, int32(bbs[7].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d217)
								ctx.EmitStoreToStack(d217, int32(bbs[7].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
							}
							if d104.Loc == LocReg {
								ctx.UnprotectReg(d104.Reg)
							} else if d104.Loc == LocRegPair {
								ctx.UnprotectReg(d104.Reg)
								ctx.UnprotectReg(d104.Reg2)
							}
						}
						ps218 := PhiState{General: ps.General}
						ps218.OverlayValues = make([]JITValueDesc, 218)
						ps218.OverlayValues[1] = d1
						ps218.OverlayValues[2] = d2
						ps218.OverlayValues[3] = d3
						ps218.OverlayValues[4] = d4
						ps218.OverlayValues[5] = d5
						ps218.OverlayValues[6] = d6
						ps218.OverlayValues[9] = d9
						ps218.OverlayValues[20] = d20
						ps218.OverlayValues[30] = d30
						ps218.OverlayValues[31] = d31
						ps218.OverlayValues[32] = d32
						ps218.OverlayValues[33] = d33
						ps218.OverlayValues[35] = d35
						ps218.OverlayValues[36] = d36
						ps218.OverlayValues[37] = d37
						ps218.OverlayValues[38] = d38
						ps218.OverlayValues[40] = d40
						ps218.OverlayValues[41] = d41
						ps218.OverlayValues[43] = d43
						ps218.OverlayValues[44] = d44
						ps218.OverlayValues[45] = d45
						ps218.OverlayValues[46] = d46
						ps218.OverlayValues[47] = d47
						ps218.OverlayValues[50] = d50
						ps218.OverlayValues[103] = d103
						ps218.OverlayValues[104] = d104
						ps218.OverlayValues[105] = d105
						ps218.OverlayValues[106] = d106
						ps218.OverlayValues[108] = d108
						ps218.OverlayValues[140] = d140
						ps218.OverlayValues[143] = d143
						ps218.OverlayValues[176] = d176
						ps218.OverlayValues[177] = d177
						ps218.OverlayValues[178] = d178
						ps218.OverlayValues[179] = d179
						ps218.OverlayValues[180] = d180
						ps218.OverlayValues[181] = d181
						ps218.OverlayValues[182] = d182
						ps218.OverlayValues[183] = d183
						ps218.OverlayValues[185] = d185
						ps218.OverlayValues[186] = d186
						ps218.OverlayValues[187] = d187
						ps218.OverlayValues[189] = d189
						ps218.OverlayValues[190] = d190
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
						ps218.OverlayValues[207] = d207
						ps218.OverlayValues[208] = d208
						ps218.OverlayValues[209] = d209
						ps218.OverlayValues[210] = d210
						ps218.OverlayValues[212] = d212
						ps218.OverlayValues[213] = d213
						ps218.OverlayValues[214] = d214
						ps218.OverlayValues[215] = d215
						ps218.OverlayValues[217] = d217
						ps218.PhiValues = make([]JITValueDesc, 1)
						d219 = d104
						ps218.PhiValues[0] = d219
						return bbs[7].RenderPS(ps218)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitJump(d215.Condition, lbl7)
					ctx.EmitJmp(lbl12)
					ctx.FreeDesc(&d214)
					snap220 := d1
					snap221 := d2
					snap222 := d3
					snap223 := d4
					snap224 := d5
					snap225 := d6
					snap226 := d9
					snap227 := d20
					snap228 := d30
					snap229 := d31
					snap230 := d32
					snap231 := d33
					snap232 := d35
					snap233 := d36
					snap234 := d37
					snap235 := d38
					snap236 := d40
					snap237 := d41
					snap238 := d43
					snap239 := d44
					snap240 := d45
					snap241 := d46
					snap242 := d47
					snap243 := d50
					snap244 := d103
					snap245 := d104
					snap246 := d105
					snap247 := d106
					snap248 := d108
					snap249 := d140
					snap250 := d143
					snap251 := d176
					snap252 := d177
					snap253 := d178
					snap254 := d179
					snap255 := d180
					snap256 := d181
					snap257 := d182
					snap258 := d183
					snap259 := d185
					snap260 := d186
					snap261 := d187
					snap262 := d189
					snap263 := d190
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
					snap279 := d207
					snap280 := d208
					snap281 := d209
					snap282 := d210
					snap283 := d212
					snap284 := d213
					snap285 := d214
					snap286 := d215
					snap287 := d217
					snap288 := d219
					alloc289 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc289)
					d1 = snap220
					d2 = snap221
					d3 = snap222
					d4 = snap223
					d5 = snap224
					d6 = snap225
					d9 = snap226
					d20 = snap227
					d30 = snap228
					d31 = snap229
					d32 = snap230
					d33 = snap231
					d35 = snap232
					d36 = snap233
					d37 = snap234
					d38 = snap235
					d40 = snap236
					d41 = snap237
					d43 = snap238
					d44 = snap239
					d45 = snap240
					d46 = snap241
					d47 = snap242
					d50 = snap243
					d103 = snap244
					d104 = snap245
					d105 = snap246
					d106 = snap247
					d108 = snap248
					d140 = snap249
					d143 = snap250
					d176 = snap251
					d177 = snap252
					d178 = snap253
					d179 = snap254
					d180 = snap255
					d181 = snap256
					d182 = snap257
					d183 = snap258
					d185 = snap259
					d186 = snap260
					d187 = snap261
					d189 = snap262
					d190 = snap263
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
					d207 = snap279
					d208 = snap280
					d209 = snap281
					d210 = snap282
					d212 = snap283
					d213 = snap284
					d214 = snap285
					d215 = snap286
					d217 = snap287
					d219 = snap288
					ctx.MarkLabel(lbl12)
					ctx.SyncDesc(&d104)
					if d104.Loc == LocReg {
						ctx.ProtectReg(d104.Reg)
					} else if d104.Loc == LocRegPair {
						ctx.ProtectReg(d104.Reg)
						ctx.ProtectReg(d104.Reg2)
					}
					d290 = d104
					if d290.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d290)
					if d290.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d290, int32(bbs[7].PhiBase)+int32(0), 2)
					} else if d290.Loc == LocInputPair {
						ctx.EnsureDesc(&d290)
						ctx.EmitStoreScmerToStack(d290, int32(bbs[7].PhiBase)+int32(0))
					} else if d290.Loc == LocRegPair || d290.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d290, int32(bbs[7].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d290)
						ctx.EmitStoreToStack(d290, int32(bbs[7].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[7].PhiBase)+int32(0))+8)
					}
					if d104.Loc == LocReg {
						ctx.UnprotectReg(d104.Reg)
					} else if d104.Loc == LocRegPair {
						ctx.UnprotectReg(d104.Reg)
						ctx.UnprotectReg(d104.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc289)
					d1 = snap220
					d2 = snap221
					d3 = snap222
					d4 = snap223
					d5 = snap224
					d6 = snap225
					d9 = snap226
					d20 = snap227
					d30 = snap228
					d31 = snap229
					d32 = snap230
					d33 = snap231
					d35 = snap232
					d36 = snap233
					d37 = snap234
					d38 = snap235
					d40 = snap236
					d41 = snap237
					d43 = snap238
					d44 = snap239
					d45 = snap240
					d46 = snap241
					d47 = snap242
					d50 = snap243
					d103 = snap244
					d104 = snap245
					d105 = snap246
					d106 = snap247
					d108 = snap248
					d140 = snap249
					d143 = snap250
					d176 = snap251
					d177 = snap252
					d178 = snap253
					d179 = snap254
					d180 = snap255
					d181 = snap256
					d182 = snap257
					d183 = snap258
					d185 = snap259
					d186 = snap260
					d187 = snap261
					d189 = snap262
					d190 = snap263
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
					d207 = snap279
					d208 = snap280
					d209 = snap281
					d210 = snap282
					d212 = snap283
					d213 = snap284
					d214 = snap285
					d215 = snap286
					d217 = snap287
					d219 = snap288
					ps291 := PhiState{General: true}
					ps291.OverlayValues = make([]JITValueDesc, 291)
					ps291.OverlayValues[1] = d1
					ps291.OverlayValues[2] = d2
					ps291.OverlayValues[3] = d3
					ps291.OverlayValues[4] = d4
					ps291.OverlayValues[5] = d5
					ps291.OverlayValues[6] = d6
					ps291.OverlayValues[9] = d9
					ps291.OverlayValues[20] = d20
					ps291.OverlayValues[30] = d30
					ps291.OverlayValues[31] = d31
					ps291.OverlayValues[32] = d32
					ps291.OverlayValues[33] = d33
					ps291.OverlayValues[35] = d35
					ps291.OverlayValues[36] = d36
					ps291.OverlayValues[37] = d37
					ps291.OverlayValues[38] = d38
					ps291.OverlayValues[40] = d40
					ps291.OverlayValues[41] = d41
					ps291.OverlayValues[43] = d43
					ps291.OverlayValues[44] = d44
					ps291.OverlayValues[45] = d45
					ps291.OverlayValues[46] = d46
					ps291.OverlayValues[47] = d47
					ps291.OverlayValues[50] = d50
					ps291.OverlayValues[103] = d103
					ps291.OverlayValues[104] = d104
					ps291.OverlayValues[105] = d105
					ps291.OverlayValues[106] = d106
					ps291.OverlayValues[108] = d108
					ps291.OverlayValues[140] = d140
					ps291.OverlayValues[143] = d143
					ps291.OverlayValues[176] = d176
					ps291.OverlayValues[177] = d177
					ps291.OverlayValues[178] = d178
					ps291.OverlayValues[179] = d179
					ps291.OverlayValues[180] = d180
					ps291.OverlayValues[181] = d181
					ps291.OverlayValues[182] = d182
					ps291.OverlayValues[183] = d183
					ps291.OverlayValues[185] = d185
					ps291.OverlayValues[186] = d186
					ps291.OverlayValues[187] = d187
					ps291.OverlayValues[189] = d189
					ps291.OverlayValues[190] = d190
					ps291.OverlayValues[191] = d191
					ps291.OverlayValues[192] = d192
					ps291.OverlayValues[193] = d193
					ps291.OverlayValues[194] = d194
					ps291.OverlayValues[195] = d195
					ps291.OverlayValues[196] = d196
					ps291.OverlayValues[197] = d197
					ps291.OverlayValues[198] = d198
					ps291.OverlayValues[199] = d199
					ps291.OverlayValues[200] = d200
					ps291.OverlayValues[201] = d201
					ps291.OverlayValues[202] = d202
					ps291.OverlayValues[203] = d203
					ps291.OverlayValues[204] = d204
					ps291.OverlayValues[205] = d205
					ps291.OverlayValues[207] = d207
					ps291.OverlayValues[208] = d208
					ps291.OverlayValues[209] = d209
					ps291.OverlayValues[210] = d210
					ps291.OverlayValues[212] = d212
					ps291.OverlayValues[213] = d213
					ps291.OverlayValues[214] = d214
					ps291.OverlayValues[215] = d215
					ps291.OverlayValues[217] = d217
					ps291.OverlayValues[219] = d219
					ps291.OverlayValues[290] = d290
					ps292 := PhiState{General: true}
					ps292.OverlayValues = make([]JITValueDesc, 291)
					ps292.OverlayValues[1] = d1
					ps292.OverlayValues[2] = d2
					ps292.OverlayValues[3] = d3
					ps292.OverlayValues[4] = d4
					ps292.OverlayValues[5] = d5
					ps292.OverlayValues[6] = d6
					ps292.OverlayValues[9] = d9
					ps292.OverlayValues[20] = d20
					ps292.OverlayValues[30] = d30
					ps292.OverlayValues[31] = d31
					ps292.OverlayValues[32] = d32
					ps292.OverlayValues[33] = d33
					ps292.OverlayValues[35] = d35
					ps292.OverlayValues[36] = d36
					ps292.OverlayValues[37] = d37
					ps292.OverlayValues[38] = d38
					ps292.OverlayValues[40] = d40
					ps292.OverlayValues[41] = d41
					ps292.OverlayValues[43] = d43
					ps292.OverlayValues[44] = d44
					ps292.OverlayValues[45] = d45
					ps292.OverlayValues[46] = d46
					ps292.OverlayValues[47] = d47
					ps292.OverlayValues[50] = d50
					ps292.OverlayValues[103] = d103
					ps292.OverlayValues[104] = d104
					ps292.OverlayValues[105] = d105
					ps292.OverlayValues[106] = d106
					ps292.OverlayValues[108] = d108
					ps292.OverlayValues[140] = d140
					ps292.OverlayValues[143] = d143
					ps292.OverlayValues[176] = d176
					ps292.OverlayValues[177] = d177
					ps292.OverlayValues[178] = d178
					ps292.OverlayValues[179] = d179
					ps292.OverlayValues[180] = d180
					ps292.OverlayValues[181] = d181
					ps292.OverlayValues[182] = d182
					ps292.OverlayValues[183] = d183
					ps292.OverlayValues[185] = d185
					ps292.OverlayValues[186] = d186
					ps292.OverlayValues[187] = d187
					ps292.OverlayValues[189] = d189
					ps292.OverlayValues[190] = d190
					ps292.OverlayValues[191] = d191
					ps292.OverlayValues[192] = d192
					ps292.OverlayValues[193] = d193
					ps292.OverlayValues[194] = d194
					ps292.OverlayValues[195] = d195
					ps292.OverlayValues[196] = d196
					ps292.OverlayValues[197] = d197
					ps292.OverlayValues[198] = d198
					ps292.OverlayValues[199] = d199
					ps292.OverlayValues[200] = d200
					ps292.OverlayValues[201] = d201
					ps292.OverlayValues[202] = d202
					ps292.OverlayValues[203] = d203
					ps292.OverlayValues[204] = d204
					ps292.OverlayValues[205] = d205
					ps292.OverlayValues[207] = d207
					ps292.OverlayValues[208] = d208
					ps292.OverlayValues[209] = d209
					ps292.OverlayValues[210] = d210
					ps292.OverlayValues[212] = d212
					ps292.OverlayValues[213] = d213
					ps292.OverlayValues[214] = d214
					ps292.OverlayValues[215] = d215
					ps292.OverlayValues[217] = d217
					ps292.OverlayValues[219] = d219
					ps292.OverlayValues[290] = d290
					ps292.PhiValues = make([]JITValueDesc, 1)
					d293 = d104
					ps292.PhiValues[0] = d293
					snap294 := d1
					snap295 := d2
					snap296 := d3
					snap297 := d4
					snap298 := d5
					snap299 := d6
					snap300 := d9
					snap301 := d20
					snap302 := d30
					snap303 := d31
					snap304 := d32
					snap305 := d33
					snap306 := d35
					snap307 := d36
					snap308 := d37
					snap309 := d38
					snap310 := d40
					snap311 := d41
					snap312 := d43
					snap313 := d44
					snap314 := d45
					snap315 := d46
					snap316 := d47
					snap317 := d50
					snap318 := d103
					snap319 := d104
					snap320 := d105
					snap321 := d106
					snap322 := d108
					snap323 := d140
					snap324 := d143
					snap325 := d176
					snap326 := d177
					snap327 := d178
					snap328 := d179
					snap329 := d180
					snap330 := d181
					snap331 := d182
					snap332 := d183
					snap333 := d185
					snap334 := d186
					snap335 := d187
					snap336 := d189
					snap337 := d190
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
					snap353 := d207
					snap354 := d208
					snap355 := d209
					snap356 := d210
					snap357 := d212
					snap358 := d213
					snap359 := d214
					snap360 := d215
					snap361 := d217
					snap362 := d219
					snap363 := d290
					snap364 := d293
					alloc365 := ctx.SnapshotAllocState()
					if !bbs[7].Rendered {
						bbs[7].RenderPS(ps292)
					}
					ctx.RestoreAllocState(alloc365)
					d1 = snap294
					d2 = snap295
					d3 = snap296
					d4 = snap297
					d5 = snap298
					d6 = snap299
					d9 = snap300
					d20 = snap301
					d30 = snap302
					d31 = snap303
					d32 = snap304
					d33 = snap305
					d35 = snap306
					d36 = snap307
					d37 = snap308
					d38 = snap309
					d40 = snap310
					d41 = snap311
					d43 = snap312
					d44 = snap313
					d45 = snap314
					d46 = snap315
					d47 = snap316
					d50 = snap317
					d103 = snap318
					d104 = snap319
					d105 = snap320
					d106 = snap321
					d108 = snap322
					d140 = snap323
					d143 = snap324
					d176 = snap325
					d177 = snap326
					d178 = snap327
					d179 = snap328
					d180 = snap329
					d181 = snap330
					d182 = snap331
					d183 = snap332
					d185 = snap333
					d186 = snap334
					d187 = snap335
					d189 = snap336
					d190 = snap337
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
					d207 = snap353
					d208 = snap354
					d209 = snap355
					d210 = snap356
					d212 = snap357
					d213 = snap358
					d214 = snap359
					d215 = snap360
					d217 = snap361
					d219 = snap362
					d290 = snap363
					d293 = snap364
					if !bbs[6].Rendered {
						return bbs[6].RenderPS(ps291)
					}
					return result
					return result
				}
				ps366 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps366)
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
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d1)
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
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
					}
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
