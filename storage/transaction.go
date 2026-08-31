/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
package storage

import "context"
import "fmt"
import "github.com/carli2/hybridsort"
import "runtime"
import "sync"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import NonLockingReadMap "github.com/launix-de/NonLockingReadMap"

// TxMode selects the transaction isolation strategy.
type TxMode uint8

const (
	TxCursorStability TxMode = iota // default: direct writes + undo masks
	TxACID                          // snapshot isolation + OCC commit
)

// TxState tracks the lifecycle of a transaction.
type TxState uint8

const (
	TxActive TxState = iota
	TxCommitted
	TxAborted
)

// storageShardTransaction holds all per-shard state for a transaction.
//
// All four NonBlockingBitMaps are embedded as values — they are zero-alloc
// (lazy) until the first Set call, so allocating a storageShardTransaction
// for a shard that is only read-checked costs only the struct allocation.
//
// The bitmaps are written exclusively under st.mu (plain Set is safe).
// Reads in the hot-path visibility check (IsVisible) happen lock-free via Get.
//
// Fields by transaction mode:
//
//	CursorStability  InsertMask / InsertRecids  — inserted rows (undo = delete)
//	                 DeletedMask / DeletedRecids — deleted rows  (undo = undelete)
//	ACID             DeleteMask  / DeleteRecids  — rows to delete at commit
//	                 UndeleteMask/ UndeleteRecids — staged rows visible to this tx
type storageShardTransaction struct {
	// cursor-stability undo bitmaps
	InsertMask  NonLockingReadMap.NonBlockingBitMap
	DeletedMask NonLockingReadMap.NonBlockingBitMap
	// ACID overlay bitmaps
	DeleteMask   NonLockingReadMap.NonBlockingBitMap
	UndeleteMask NonLockingReadMap.NonBlockingBitMap

	// Recids for iteration at rollback/commit time.
	// Append-only; protected by mu.
	InsertRecids   []uint32
	DeletedRecids  []uint32
	DeleteRecids   []uint32
	UndeleteRecids []uint32
	generation     *tableShardTopology

	mu sync.Mutex
}

// shardSavepoint records the Recids slice lengths for one shard at a savepoint.
type shardSavepoint struct {
	InsertLen   int
	DeletedLen  int
	DeleteLen   int
	UndeleteLen int
}

// Savepoint captures the state of a transaction at a point in time.
// Used for nested transactions (trigger recovery, savepoints).
type Savepoint struct {
	shardLens            map[*storageShard]shardSavepoint
	Depth                uint32
	repartitionDeleteLen int
}

type repartitionDeleteAction struct {
	table    *table
	shard    *storageShard
	oldRecid uint32
}

// global transaction ID counter
var txIDCounter uint64

// GlobalCommitEpoch is advanced on each ACID commit.
var GlobalCommitEpoch uint64

// TxContext holds the state for one transaction.
//
// All per-shard state lives in a single shards map, keyed by *storageShard.
// The map is nil until the first write operation, so read-only transactions
// (the common case with with_autocommit) allocate nothing beyond the
// TxContext struct itself.
type TxContext struct {
	ID            uint64
	Mode          TxMode
	State         TxState
	SnapshotEpoch uint64 // ACID: snapshot boundary
	Depth         uint32 // nesting depth for savepoints / triggers
	Session       scm.Scmer
	SessionState  *scm.SessionState // owning connection; retained while the transaction object is parked
	querySeq      atomic.Uint64     // current statement generation; zero while parked
	queryInfo     atomic.Pointer[string]
	queryActive   atomic.Bool
	queryMu       sync.Mutex // serializes statements which reuse this transaction object
	// fanoutLimit/fanoutInUse bound only additional multi-shard workers. A
	// single relevant shard never reads or writes this cache line.
	fanoutLimit atomic.Int32
	fanoutInUse atomic.Int32

	// Per-shard state, nil until first write (zero-alloc for read-only transactions).
	shards map[*storageShard]*storageShardTransaction

	// Deferred sync: shards with pending log writes that need fsync at commit.
	touchedShards       sync.Map // map[*storageShard]bool
	autoCommit          bool
	writeHeld           map[*storageShard]uint32 // reentrant write-lock depth per shard
	repartitionDeletes  []repartitionDeleteAction
	invalidationDepth   uint32
	invalidationVisited map[string]bool

	mu sync.Mutex
}

// NewTxContext creates a new active transaction context with the given mode.
func NewTxContext(mode TxMode) *TxContext {
	tx := &TxContext{}
	tx.reset(mode)
	return tx
}

// reset starts another transaction in the same allocation. Commit and
// Rollback leave the object parked in its Scheme session so the next statement
// can reuse it without retaining any state from the completed transaction.
func (tx *TxContext) reset(mode TxMode) {
	// Normal commit/rollback releases every physical shard lock first. Clear a
	// stale ownership publication defensively before this allocation is reused.
	for shard := range tx.writeHeld {
		shard.writeOwner.CompareAndSwap(tx, nil)
	}
	tx.ID = atomic.AddUint64(&txIDCounter, 1)
	tx.Mode = mode
	tx.State = TxActive
	tx.SnapshotEpoch = 0
	if mode == TxACID {
		tx.SnapshotEpoch = atomic.LoadUint64(&GlobalCommitEpoch)
	}
	tx.Depth = 0
	tx.fanoutLimit.Store(int32(runtime.GOMAXPROCS(0)))
	tx.fanoutInUse.Store(0)
	tx.shards = nil
	tx.touchedShards = sync.Map{}
	tx.autoCommit = false
	tx.writeHeld = nil
	tx.repartitionDeletes = nil
	tx.invalidationDepth = 0
	tx.invalidationVisited = nil
}

// claimFanoutWorkers reserves at most half of the transaction's remaining
// fanout capacity. It never blocks: callers run synchronously when fewer than
// two workers can be claimed, preventing nested scans from waiting on their
// parents' reservations.
func (tx *TxContext) claimFanoutWorkers(maxWorkers int) int {
	if tx == nil || maxWorkers < 2 {
		return 0
	}
	for {
		limit := tx.fanoutLimit.Load()
		used := tx.fanoutInUse.Load()
		claim := (limit - used) / 2
		if claim > int32(maxWorkers) {
			claim = int32(maxWorkers)
		}
		if claim < 2 {
			return 0
		}
		if tx.fanoutInUse.CompareAndSwap(used, used+claim) {
			return int(claim)
		}
	}
}

func (tx *TxContext) releaseFanoutWorkers(workers int) {
	if workers > 0 {
		tx.fanoutInUse.Add(-int32(workers))
	}
}

func (tx *TxContext) SessionValue(key string) scm.Scmer {
	if tx == nil || tx.Session.IsNil() {
		return scm.NewNil()
	}
	return scm.Apply(tx.Session, scm.NewString(key))
}

// QuerySessionState exposes cancellation state only while this transaction is
// executing a statement. A parked explicit transaction is intentionally not a
// process and therefore has no current query generation.
func (tx *TxContext) QuerySessionState() (*scm.SessionState, uint64) {
	if tx == nil || !tx.queryActive.Load() {
		return nil, 0
	}
	return tx.SessionState, tx.querySeq.Load()
}

func (tx *TxContext) beginQuery(ss *scm.SessionState, seq uint64, info string) {
	tx.SessionState = ss
	tx.querySeq.Store(seq)
	infoPtr := &info
	tx.queryInfo.Store(infoPtr)
	tx.queryActive.Store(true)
	if ss != nil {
		ss.SetQueryInfoPointer(seq, infoPtr)
	}
}

func (tx *TxContext) endQuery(seq uint64) {
	if tx.querySeq.CompareAndSwap(seq, 0) {
		tx.queryActive.Store(false)
		tx.queryInfo.Store(nil)
		if tx.SessionState != nil {
			tx.SessionState.FinishQueryExecution(seq)
		}
	}
}

func txSessionScmer(tx *TxContext) scm.Scmer {
	if tx == nil || tx.Session.IsNil() {
		return scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewNil() })
	}
	return tx.Session
}

func bindExecutionEnv(env *scm.Env, session, tx scm.Scmer) *scm.Env {
	if env == nil {
		env = &scm.Globalenv
	}
	vars := make(scm.Vars, len(env.Vars)+2)
	for name, value := range env.Vars {
		vars[name] = value
	}
	vars[scm.Symbol("session")] = session
	vars[scm.Symbol("tx")] = tx
	return &scm.Env{
		Vars:         vars,
		VarsNumbered: env.VarsNumbered,
		Outer:        env.Outer,
		Nodefine:     env.Nodefine,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// getOrCreateShardTxLocked returns the storageShardTransaction for a shard,
// creating it if it does not exist. Must be called with tx.mu held.
func (tx *TxContext) getOrCreateShardTxLocked(shard *storageShard) *storageShardTransaction {
	if tx.shards == nil {
		tx.shards = make(map[*storageShard]*storageShardTransaction)
	}
	st := tx.shards[shard]
	if st == nil {
		st = new(storageShardTransaction)
		st.generation = shard.generation.Load()
		if st.generation != nil {
			st.generation.pinTransaction()
		}
		tx.shards[shard] = st
		shard.beginTransactionUse()
	}
	return st
}

func (tx *TxContext) addRepartitionDelete(table *table, shard *storageShard, oldRecid uint32) {
	tx.mu.Lock()
	tx.repartitionDeletes = append(tx.repartitionDeletes, repartitionDeleteAction{table: table, shard: shard, oldRecid: oldRecid})
	tx.mu.Unlock()
}

// finishRepartitionActions must run before activeTransactions is decremented:
// repartition publication waits for that counter and may otherwise pass its
// final pending-delete drain before the committed action is visible.
func (tx *TxContext) finishRepartitionActions(commit bool) {
	tx.mu.Lock()
	actions := tx.repartitionDeletes
	tx.repartitionDeletes = nil
	tx.mu.Unlock()
	if !commit {
		return
	}
	for _, action := range actions {
		action.table.dualWriteDelete(action.shard, action.oldRecid, nil)
	}
}

func (tx *TxContext) releaseActiveTransactionsLocked() {
	for shard, st := range tx.shards {
		shard.endTransactionUse()
		if st.generation != nil {
			st.generation.releaseTransaction()
		}
	}
}

// getShardTx returns the storageShardTransaction for a shard, or nil if none exists.
func (tx *TxContext) getShardTx(shard *storageShard) *storageShardTransaction {
	tx.mu.Lock()
	st := tx.shards[shard] // nil map returns nil safely
	tx.mu.Unlock()
	return st
}

// ---------------------------------------------------------------------------
// Write-lock tracking (reentrant depth counter per shard)
// ---------------------------------------------------------------------------

// EnterShardWrite marks that the current transaction holds a write lock on shard.
func (tx *TxContext) EnterShardWrite(shard *storageShard) {
	tx.mu.Lock()
	if tx.writeHeld == nil {
		tx.writeHeld = make(map[*storageShard]uint32)
	}
	depth := tx.writeHeld[shard]
	tx.writeHeld[shard] = depth + 1
	if depth == 0 {
		if owner := shard.writeOwner.Load(); owner != nil && owner != tx {
			tx.mu.Unlock()
			panic("shard write ownership published by a different transaction")
		}
		shard.writeOwner.Store(tx)
	}
	tx.mu.Unlock()
}

// ExitShardWrite decrements the write-hold depth for a shard.
func (tx *TxContext) ExitShardWrite(shard *storageShard) {
	tx.mu.Lock()
	if d := tx.writeHeld[shard]; d <= 1 {
		delete(tx.writeHeld, shard)
		if !shard.writeOwner.CompareAndSwap(tx, nil) {
			tx.mu.Unlock()
			panic("shard write ownership lost before release")
		}
	} else {
		tx.writeHeld[shard] = d - 1
	}
	tx.mu.Unlock()
}

// HasShardWrite is a read-only hot-path query. It intentionally performs one
// atomic load and no lock, CAS, counter update or other shared-memory write.
func (tx *TxContext) HasShardWrite(shard *storageShard) bool {
	return tx != nil && shard.writeOwner.Load() == tx
}

// ---------------------------------------------------------------------------
// Cursor-stability undo log
// ---------------------------------------------------------------------------

// LogInsert records that a row was inserted; on rollback it will be deleted.
// Cursor-stability only.
func (tx *TxContext) LogInsert(shard *storageShard, rowIndex uint32) {
	tx.mu.Lock()
	st := tx.getOrCreateShardTxLocked(shard)
	tx.mu.Unlock()
	st.mu.Lock()
	st.InsertMask.Set(uint(rowIndex), true)
	st.InsertRecids = append(st.InsertRecids, rowIndex)
	st.mu.Unlock()
}

// LogDelete records that a row was deleted; on rollback it will be undeleted.
// Cursor-stability only.
func (tx *TxContext) LogDelete(shard *storageShard, rowIndex uint32) {
	tx.mu.Lock()
	st := tx.getOrCreateShardTxLocked(shard)
	tx.mu.Unlock()
	st.mu.Lock()
	st.DeletedMask.Set(uint(rowIndex), true)
	st.DeletedRecids = append(st.DeletedRecids, rowIndex)
	st.mu.Unlock()
}

// ---------------------------------------------------------------------------
// ACID overlay masks
// ---------------------------------------------------------------------------

// AddToDeleteMask records that this ACID tx wants to delete a row at commit.
func (tx *TxContext) AddToDeleteMask(shard *storageShard, recid uint32) {
	tx.mu.Lock()
	st := tx.getOrCreateShardTxLocked(shard)
	tx.mu.Unlock()
	st.mu.Lock()
	st.DeleteMask.Set(uint(recid), true)
	st.DeleteRecids = append(st.DeleteRecids, recid)
	st.mu.Unlock()
}

// AddToUndeleteMask records that this ACID tx can see a staged (inserted) row.
func (tx *TxContext) AddToUndeleteMask(shard *storageShard, recid uint32) {
	tx.mu.Lock()
	st := tx.getOrCreateShardTxLocked(shard)
	tx.mu.Unlock()
	st.mu.Lock()
	st.UndeleteMask.Set(uint(recid), true)
	st.UndeleteRecids = append(st.UndeleteRecids, recid)
	st.mu.Unlock()
}

// UnstageRow removes recid from UndeleteMask (ACID UPDATE/DELETE of a staged row).
// Returns true if the row was staged by this tx and has been un-staged.
func (tx *TxContext) UnstageRow(shard *storageShard, recid uint32) bool {
	st := tx.getShardTx(shard)
	if st == nil || !st.UndeleteMask.Get(uint(recid)) {
		return false
	}
	// plain Set is safe: caller holds the shard write lock
	st.UndeleteMask.Set(uint(recid), false)
	return true
}

// ---------------------------------------------------------------------------
// Deferred fsync
// ---------------------------------------------------------------------------

// RegisterTouchedShard marks a shard as having pending writes for deferred sync.
// Only Safe-engine shards need an fsync; Memory/Cache/Sloppy shards are skipped.
func (tx *TxContext) RegisterTouchedShard(shard *storageShard) {
	if shard.t.PersistencyMode != Safe {
		return
	}
	tx.touchedShards.Store(shard, true)
}

// SyncTouchedShards flushes all pending log writes to durable storage.
func (tx *TxContext) SyncTouchedShards() {
	tx.touchedShards.Range(func(key, _ any) bool {
		shard := key.(*storageShard)
		if shard.t.PersistencyMode == Safe && shard.logfile != nil {
			shard.logfile.Sync()
		}
		return true
	})
	tx.touchedShards = sync.Map{}
}

// ---------------------------------------------------------------------------
// Visibility (ACID)
// ---------------------------------------------------------------------------

// IsVisible determines whether a row is visible to this ACID transaction.
//
//	UndeleteMask wins — it is the only way an ACID tx sees its own inserts.
//	Otherwise: not globally deleted AND not locally (tx-level) deleted.
func (tx *TxContext) IsVisible(shard *storageShard, recid uint32) bool {
	st := tx.getShardTx(shard)
	if st == nil {
		return !shard.deletions.Get(uint(recid))
	}
	if st.UndeleteMask.Get(uint(recid)) {
		return true
	}
	return !shard.deletions.Get(uint(recid)) && !st.DeleteMask.Get(uint(recid))
}

// ---------------------------------------------------------------------------
// Savepoints
// ---------------------------------------------------------------------------

// CreateSavepoint captures the current transaction state for later rollback.
func (tx *TxContext) CreateSavepoint() Savepoint {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	sp := Savepoint{Depth: tx.Depth}
	sp.repartitionDeleteLen = len(tx.repartitionDeletes)
	tx.Depth++
	if len(tx.shards) > 0 {
		sp.shardLens = make(map[*storageShard]shardSavepoint, len(tx.shards))
		for s, st := range tx.shards {
			st.mu.Lock()
			sp.shardLens[s] = shardSavepoint{
				InsertLen:   len(st.InsertRecids),
				DeletedLen:  len(st.DeletedRecids),
				DeleteLen:   len(st.DeleteRecids),
				UndeleteLen: len(st.UndeleteRecids),
			}
			st.mu.Unlock()
		}
	}
	return sp
}

// trackedShardSetLocked returns the rebuild generations whose transaction
// masks are applied independently. Caller must hold tx.mu.
func (tx *TxContext) trackedShardSetLocked() map[*storageShard]struct{} {
	tracked := make(map[*storageShard]struct{}, len(tx.shards))
	for shard := range tx.shards {
		tracked[shard] = struct{}{}
	}
	return tracked
}

// RollbackToSavepoint undoes all changes made since the savepoint was created.
func (tx *TxContext) RollbackToSavepoint(sp Savepoint) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.Depth = sp.Depth
	if sp.repartitionDeleteLen < len(tx.repartitionDeletes) {
		tx.repartitionDeletes = tx.repartitionDeletes[:sp.repartitionDeleteLen]
	}
	trackedShards := tx.trackedShardSetLocked()
	for shard, st := range tx.shards {
		lens := sp.shardLens[shard] // zero value (all zeros) if shard is new since savepoint
		st.mu.Lock()
		switch tx.Mode {
		case TxCursorStability:
			// Undo inserts: mark as globally deleted
			for i := len(st.InsertRecids) - 1; i >= lens.InsertLen; i-- {
				recid := st.InsertRecids[i]
				st.InsertMask.Set(uint(recid), false)
				shard.mu.Lock()
				shard.deletions.Set(uint(recid), true)
				if shard.logfile != nil {
					shard.logfile.Write(LogEntryDelete{recid})
				}
				shard.syncNextVisibilityLocked(recid, trackedShards)
				shard.mu.Unlock()
			}
			st.InsertRecids = st.InsertRecids[:lens.InsertLen]
			// Undo deletes: restore global visibility
			for i := len(st.DeletedRecids) - 1; i >= lens.DeletedLen; i-- {
				recid := st.DeletedRecids[i]
				st.DeletedMask.Set(uint(recid), false)
				shard.mu.Lock()
				shard.deletions.Set(uint(recid), false)
				shard.logVisibilityChangeLocked(recid, false)
				shard.rollbackProtected.Set(uint(recid), false)
				shard.syncNextVisibilityLocked(recid, trackedShards)
				shard.mu.Unlock()
			}
			st.DeletedRecids = st.DeletedRecids[:lens.DeletedLen]

		case TxACID:
			// Rollback DeleteMask additions
			for i := len(st.DeleteRecids) - 1; i >= lens.DeleteLen; i-- {
				st.DeleteMask.Set(uint(st.DeleteRecids[i]), false)
			}
			st.DeleteRecids = st.DeleteRecids[:lens.DeleteLen]
			// Rollback UndeleteMask additions (re-hide the staged rows)
			for i := len(st.UndeleteRecids) - 1; i >= lens.UndeleteLen; i-- {
				recid := st.UndeleteRecids[i]
				st.UndeleteMask.Set(uint(recid), false)
				ownsShardWrite := tx.writeHeld[shard] > 0
				if !ownsShardWrite {
					shard.mu.Lock()
				}
				shard.deletions.Set(uint(recid), true)
				shard.rollbackProtected.Set(uint(recid), false)
				shard.syncNextVisibilityLocked(recid, trackedShards)
				if !ownsShardWrite {
					shard.mu.Unlock()
				}
			}
			st.UndeleteRecids = st.UndeleteRecids[:lens.UndeleteLen]
		}
		st.mu.Unlock()
		if tx.Mode == TxCursorStability && shard.t.PersistencyMode == Safe && shard.logfile != nil {
			shard.logfile.Sync()
		}
	}
}

// ---------------------------------------------------------------------------
// Commit
// ---------------------------------------------------------------------------

// Commit finalizes the transaction.
func (tx *TxContext) Commit() error {
	switch tx.Mode {
	case TxCursorStability:
		tx.mu.Lock()
		trackedShards := tx.trackedShardSetLocked()
		for shard, st := range tx.shards {
			st.mu.Lock()
			shard.mu.Lock()
			for _, recid := range st.DeletedRecids {
				shard.rollbackProtected.Set(uint(recid), false)
				shard.syncNextVisibilityLocked(recid, trackedShards)
			}
			shard.mu.Unlock()
			st.mu.Unlock()
		}
		tx.mu.Unlock()
		tx.finishRepartitionActions(true)
		tx.mu.Lock()
		tx.releaseActiveTransactionsLocked()
		tx.State = TxCommitted
		tx.shards = nil
		tx.mu.Unlock()
		tx.SyncTouchedShards()
	case TxACID:
		if err := tx.commitACID(); err != nil {
			return err
		}
	}
	return nil
}

// commitACID locks touched shards in deterministic order, validates,
// and applies overlay masks to global state.
func (tx *TxContext) commitACID() error {
	tx.mu.Lock()
	shards := make([]*storageShard, 0, len(tx.shards))
	trackedShards := make(map[*storageShard]struct{}, len(tx.shards))
	for s := range tx.shards {
		shards = append(shards, s)
		trackedShards[s] = struct{}{}
	}
	tx.mu.Unlock()

	// Deterministic lock ordering prevents deadlocks.
	hybridsort.Slice(shards, func(i, j int) bool {
		return shards[i].uuid.String() < shards[j].uuid.String()
	})
	for _, s := range shards {
		s.mu.Lock()
	}

	// Validate: for each recid in DeleteMask, check it hasn't already been
	// globally deleted by another committed tx (write-write conflict → abort).
	for _, shard := range shards {
		st := tx.shards[shard]
		for _, recid := range st.DeleteRecids {
			if !st.DeleteMask.Get(uint(recid)) {
				continue // bit was rolled back via savepoint
			}
			if shard.deletions.Get(uint(recid)) {
				for _, s := range shards {
					s.mu.Unlock()
				}
				tx.finishRepartitionActions(false)
				tx.mu.Lock()
				tx.releaseActiveTransactionsLocked()
				tx.State = TxAborted
				tx.shards = nil
				tx.mu.Unlock()
				return fmt.Errorf("ACID commit conflict: row %d already deleted", recid)
			}
		}
	}

	// Apply DeleteMask → set global deletions + write log
	for _, shard := range shards {
		st := tx.shards[shard]
		for _, recid := range st.DeleteRecids {
			if !st.DeleteMask.Get(uint(recid)) {
				continue
			}
			shard.deletions.Set(uint(recid), true)
			if shard.logfile != nil {
				shard.logfile.Write(LogEntryDelete{recid})
			}
			shard.syncNextVisibilityLocked(recid, trackedShards)
		}
	}
	// Apply UndeleteMask → clear global deletions (make staged rows visible)
	for _, shard := range shards {
		st := tx.shards[shard]
		for _, recid := range st.UndeleteRecids {
			if !st.UndeleteMask.Get(uint(recid)) {
				continue // un-staged (row overwritten/deleted in same tx)
			}
			shard.deletions.Set(uint(recid), false)
			shard.rollbackProtected.Set(uint(recid), false)
			shard.logVisibilityChangeLocked(recid, false)
			shard.syncNextVisibilityLocked(recid, trackedShards)
		}
	}

	atomic.AddUint64(&GlobalCommitEpoch, 1)

	for _, s := range shards {
		s.mu.Unlock()
	}

	tx.finishRepartitionActions(true)
	tx.mu.Lock()
	tx.releaseActiveTransactionsLocked()
	tx.State = TxCommitted
	tx.shards = nil
	tx.mu.Unlock()

	tx.SyncTouchedShards()
	return nil
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

// Rollback undoes the transaction.
func (tx *TxContext) Rollback() {
	switch tx.Mode {
	case TxCursorStability:
		tx.rollbackCursorStability()
	case TxACID:
		tx.rollbackACID()
	}
}

// rollbackCursorStability replays undo masks in reverse to restore global state.
func (tx *TxContext) rollbackCursorStability() {
	tx.mu.Lock()
	trackedShards := tx.trackedShardSetLocked()
	for shard, st := range tx.shards {
		st.mu.Lock()
		// Undo inserts (reverse order): mark as globally deleted
		for i := len(st.InsertRecids) - 1; i >= 0; i-- {
			recid := st.InsertRecids[i]
			shard.mu.Lock()
			shard.deletions.Set(uint(recid), true)
			if shard.logfile != nil {
				shard.logfile.Write(LogEntryDelete{recid})
			}
			shard.syncNextVisibilityLocked(recid, trackedShards)
			shard.mu.Unlock()
		}
		// Undo deletes (reverse order): restore global visibility
		for i := len(st.DeletedRecids) - 1; i >= 0; i-- {
			recid := st.DeletedRecids[i]
			shard.mu.Lock()
			shard.deletions.Set(uint(recid), false)
			shard.logVisibilityChangeLocked(recid, false)
			shard.rollbackProtected.Set(uint(recid), false)
			shard.syncNextVisibilityLocked(recid, trackedShards)
			shard.mu.Unlock()
		}
		st.mu.Unlock()
		if shard.t.PersistencyMode == Safe && shard.logfile != nil {
			shard.logfile.Sync()
		}
	}
	tx.repartitionDeletes = nil
	tx.releaseActiveTransactionsLocked()
	tx.State = TxAborted
	tx.shards = nil
	tx.touchedShards = sync.Map{}
	tx.mu.Unlock()
}

// rollbackACID discards overlay masks. Staged rows that were globally hidden
// remain as garbage (collected by the next GC/compaction pass).
func (tx *TxContext) rollbackACID() {
	tx.mu.Lock()
	trackedShards := tx.trackedShardSetLocked()
	for shard, st := range tx.shards {
		st.mu.Lock()
		shard.mu.Lock()
		for _, recid := range st.UndeleteRecids {
			shard.rollbackProtected.Set(uint(recid), false)
			shard.syncNextVisibilityLocked(recid, trackedShards)
		}
		shard.mu.Unlock()
		st.mu.Unlock()
	}
	tx.repartitionDeletes = nil
	tx.releaseActiveTransactionsLocked()
	tx.State = TxAborted
	tx.shards = nil
	tx.mu.Unlock()
	tx.touchedShards = sync.Map{}
}

// SessionStateFromTx returns the SessionState from the given tx, falling back
// to nil when no explicit transaction belongs to the operation.
func SessionStateFromTx(tx *TxContext) *scm.SessionState {
	if tx == nil {
		return nil
	}
	return tx.SessionState
}

// querySeqFromTx returns the statement generation cached by an autocommit
// transaction. Explicit transactions can serve multiple concurrent statements;
// each statement updates the generation on its explicit transaction context.
func querySeqFromTx(tx *TxContext) uint64 {
	if tx == nil {
		return 0
	}
	return tx.querySeq.Load()
}

// WithAutocommit executes fn inside an implicit TxCursorStability transaction
// if no explicit transaction is already active in session, and commits it
// afterwards. If an explicit transaction is active (session["transaction"] != nil),
// fn is executed as-is without any wrapping.
//
// On panic inside fn the auto-commit transaction is rolled back and the panic
// is re-raised so the caller's error handler still fires. This guarantees that
// every SQL statement executed via the HTTP or MySQL frontend runs inside a
// transaction, enabling a single fsync per statement instead of one per write.
func transactionForSession(session scm.Scmer) *TxContext {
	sessionFn := session.Func()
	if value := sessionFn(scm.NewString("__memcp_tx")); !value.IsNil() {
		if tx, ok := value.Any().(*TxContext); ok {
			return tx
		}
	}
	tx := &TxContext{Session: session, State: TxCommitted}
	sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))
	return tx
}

// WithAutocommit runs a SQL statement with the session's reusable TxContext.
// Query identity and cancellation belong to the transaction only while fn is
// running; an explicit transaction remains parked after the statement ends.
func WithAutocommit(session scm.Scmer, ss *scm.SessionState, querySeq uint64, query string, fn scm.Scmer) scm.Scmer {
	sessionFn := session.Func()
	tx := transactionForSession(session)
	tx.queryMu.Lock()
	defer tx.queryMu.Unlock()
	tx.beginQuery(ss, querySeq, query)
	defer tx.endQuery(querySeq)
	txValue := scm.NewAny(tx)

	if !sessionFn(scm.NewString("transaction")).IsNil() {
		return scm.Apply(fn, txValue)
	}

	tx.reset(TxCursorStability)
	tx.autoCommit = true
	tx.Session = session

	var result scm.Scmer
	var panicVal any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
				// the stack is lost once we re-panic panicVal below (it gets
				// re-wrapped by the Scheme error layer), so log it here while
				// we still have the original panic's stack trace.
				scm.PrintError(r)
			}
		}()
		result = scm.Apply(fn, txValue)
	}()

	if panicVal != nil {
		if tx.State == TxActive {
			tx.Rollback()
		}
		panic(panicVal)
	}

	if !sessionFn(scm.NewString("transaction")).IsNil() {
		return result
	}

	if err := tx.Commit(); err != nil {
		panic("autocommit failed: " + err.Error())
	}
	return result
}

func initTransaction(en scm.Env) {
	scm.DeclareTitle("Transactions")
	scm.Declare(&en, &scm.Declaration{
		Name: "tx_query",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			return scm.NewInt(int64(querySeqFromTx(scmerToTxContext(a[0]))))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the current query generation of an active transaction",
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "int"}},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "tx_check",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			tx := scmerToTxContext(a[0])
			if tx == nil {
				return scm.NewBool(true)
			}
			ss, seq := tx.QuerySessionState()
			if ss == nil || seq == 0 {
				return scm.NewBool(true)
			}
			if ss.IsKilledSeq(seq) {
				panic(context.Canceled)
			}
			if ctx := ss.QueryContext(seq); ctx != nil && ctx.Err() != nil {
				panic(ctx.Err())
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "aborts when the transaction's current query was cancelled",
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "bool"}},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "tx_connection_id",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ss := SessionStateFromTx(scmerToTxContext(a[0]))
			if ss == nil {
				return scm.NewInt(0)
			}
			return scm.NewInt(int64(ss.ID))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the connection ID owning a transaction",
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "int"}},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "tx_begin",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			tx := transactionForSession(a[0])
			if len(a) > 1 && !a[1].IsNil() {
				if current, ok := a[1].Any().(*TxContext); ok {
					tx = current
				}
			}
			if tx.State == TxActive {
				if err := tx.Commit(); err != nil {
					panic("BEGIN failed to finish current statement: " + err.Error())
				}
			}
			tx.reset(TxCursorStability)
			tx.Session = a[0]
			sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))
			sessionFn(scm.NewString("transaction"), scm.NewInt(1))
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Begins a new cursor-stability transaction. Takes the session function as argument. Stores the transaction context in the session.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session", Description: "the session function to store tx state in", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}, {Kind: "any", Label: "tx", Optional: true}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "tx_begin_acid",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			tx := transactionForSession(a[0])
			if len(a) > 1 && !a[1].IsNil() {
				if current, ok := a[1].Any().(*TxContext); ok {
					tx = current
				}
			}
			if tx.State == TxActive {
				if err := tx.Commit(); err != nil {
					panic("BEGIN failed to finish current statement: " + err.Error())
				}
			}
			tx.reset(TxACID)
			tx.Session = a[0]
			sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))
			sessionFn(scm.NewString("transaction"), scm.NewInt(1))
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Begins a new ACID transaction with snapshot isolation and OCC commit. Takes the session function as argument.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session", Description: "the session function to store tx state in", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}, {Kind: "any", Label: "tx", Optional: true}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "tx_commit",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					if err := tx.Commit(); err != nil {
						sessionFn(scm.NewString("transaction"), scm.NewNil())
						panic("COMMIT failed: " + err.Error())
					}
				}
			}
			sessionFn(scm.NewString("transaction"), scm.NewNil())
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Commits the current transaction.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session", Description: "the session function that holds tx state", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "tx_rollback",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					tx.Rollback()
				}
			}
			sessionFn(scm.NewString("transaction"), scm.NewNil())
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Rolls back the current transaction.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session", Description: "the session function that holds tx state", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "with_autocommit",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			var ss *scm.SessionState
			if !a[1].IsNil() {
				ss, _ = a[1].Any().(*scm.SessionState)
			}
			return WithAutocommit(a[0], ss, uint64(a[2].Int()), a[3].String(), a[4])
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Executes fn inside an implicit TxCursorStability transaction if no explicit " +
			"transaction is active in session. Commits on success, rolls back on error, " +
			"and re-raises any panic so the caller's error handler still fires. " +
			"If an explicit transaction is active (session[\"transaction\"] != nil), " +
			"fn is executed without any wrapping.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "func", Label: "session", Description: "the session function holding tx state", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "key", Optional: true}, {Kind: "any", Label: "value", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "session_state", Description: "owning process-list session"},
				{Kind: "int", Label: "query_seq", Description: "current query generation"},
				{Kind: "string", Label: "query", Description: "current SQL text"},
				{Kind: "func", Label: "fn", Description: "function called with the active transaction", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
}
