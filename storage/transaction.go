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

import "fmt"
import "github.com/carli2/hybridsort"
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
	shardLens map[*storageShard]shardSavepoint
	Depth     uint32
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

	// Per-shard state, nil until first write (zero-alloc for read-only transactions).
	shards map[*storageShard]*storageShardTransaction

	// Deferred sync: shards with pending log writes that need fsync at commit.
	touchedShards sync.Map // map[*storageShard]bool
	autoCommit    bool
	writeHeld     map[*storageShard]uint32 // reentrant write-lock depth per shard

	mu sync.Mutex
}

// NewTxContext creates a new active transaction context with the given mode.
func NewTxContext(mode TxMode) *TxContext {
	tx := &TxContext{
		ID:    atomic.AddUint64(&txIDCounter, 1),
		State: TxActive,
		Mode:  mode,
	}
	if mode == TxACID {
		tx.SnapshotEpoch = atomic.LoadUint64(&GlobalCommitEpoch)
	}
	return tx
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
		tx.shards[shard] = st
	}
	return st
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
	tx.writeHeld[shard]++
	tx.mu.Unlock()
}

// ExitShardWrite decrements the write-hold depth for a shard.
func (tx *TxContext) ExitShardWrite(shard *storageShard) {
	tx.mu.Lock()
	if d := tx.writeHeld[shard]; d <= 1 {
		delete(tx.writeHeld, shard)
	} else {
		tx.writeHeld[shard] = d - 1
	}
	tx.mu.Unlock()
}

// HasShardWrite returns true when this tx context currently holds a write lock
// on the shard. Used to avoid re-entering shard read locks from nested scans.
func (tx *TxContext) HasShardWrite(shard *storageShard) bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	return tx.writeHeld[shard] > 0
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

// RollbackToSavepoint undoes all changes made since the savepoint was created.
func (tx *TxContext) RollbackToSavepoint(sp Savepoint) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.Depth = sp.Depth
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
				shard.mu.Unlock()
			}
			st.InsertRecids = st.InsertRecids[:lens.InsertLen]
			// Undo deletes: restore global visibility
			for i := len(st.DeletedRecids) - 1; i >= lens.DeletedLen; i-- {
				recid := st.DeletedRecids[i]
				st.DeletedMask.Set(uint(recid), false)
				shard.mu.Lock()
				shard.deletions.Set(uint(recid), false)
				if shard.logfile != nil && recid >= shard.main_count {
					deltaIdx := int(recid - shard.main_count)
					if deltaIdx < len(shard.inserts) {
						row := shard.inserts[deltaIdx]
						cols := make([]string, 0, len(shard.deltaColumns))
						vals := make([]scm.Scmer, 0, len(shard.deltaColumns))
						for name, idx := range shard.deltaColumns {
							if idx < len(row) {
								cols = append(cols, name)
								vals = append(vals, row[idx])
							}
						}
						shard.logfile.Write(LogEntryInsert{cols, [][]scm.Scmer{vals}})
					}
				}
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
				shard.deletions.Set(uint(recid), true)
			}
			st.UndeleteRecids = st.UndeleteRecids[:lens.UndeleteLen]
		}
		st.mu.Unlock()
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
	for s := range tx.shards {
		shards = append(shards, s)
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
				tx.mu.Lock()
				tx.State = TxAborted
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
		}
	}

	atomic.AddUint64(&GlobalCommitEpoch, 1)

	for _, s := range shards {
		s.mu.Unlock()
	}

	tx.mu.Lock()
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
	defer tx.mu.Unlock()
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
			shard.mu.Unlock()
		}
		// Undo deletes (reverse order): restore global visibility
		for i := len(st.DeletedRecids) - 1; i >= 0; i-- {
			recid := st.DeletedRecids[i]
			shard.mu.Lock()
			shard.deletions.Set(uint(recid), false)
			if shard.logfile != nil && recid >= shard.main_count {
				deltaIdx := int(recid - shard.main_count)
				if deltaIdx < len(shard.inserts) {
					row := shard.inserts[deltaIdx]
					cols := make([]string, 0, len(shard.deltaColumns))
					vals := make([]scm.Scmer, 0, len(shard.deltaColumns))
					for name, idx := range shard.deltaColumns {
						if idx < len(row) {
							cols = append(cols, name)
							vals = append(vals, row[idx])
						}
					}
					shard.logfile.Write(LogEntryInsert{cols, [][]scm.Scmer{vals}})
				}
			}
			shard.mu.Unlock()
		}
		st.mu.Unlock()
	}
	tx.State = TxAborted
	tx.shards = nil
	tx.touchedShards = sync.Map{}
}

// rollbackACID discards overlay masks. Staged rows that were globally hidden
// remain as garbage (collected by the next GC/compaction pass).
func (tx *TxContext) rollbackACID() {
	tx.mu.Lock()
	tx.State = TxAborted
	tx.shards = nil
	tx.mu.Unlock()
	tx.touchedShards = sync.Map{}
}

// ---------------------------------------------------------------------------
// GLS / session helpers
// ---------------------------------------------------------------------------

// CurrentTx returns the active TxContext from goroutine-local storage, or nil.
func CurrentTx() *TxContext {
	txAny := scm.GetCurrentTx()
	if txAny == nil {
		return nil
	}
	tx, _ := txAny.(*TxContext)
	return tx
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
func WithAutocommit(sessionFn func(...scm.Scmer) scm.Scmer, fn scm.Scmer) scm.Scmer {
	if !sessionFn(scm.NewString("transaction")).IsNil() {
		return scm.Apply(fn)
	}

	tx := NewTxContext(TxCursorStability)
	sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))

	var result scm.Scmer
	var panicVal any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
			}
		}()
		result = scm.Apply(fn)
	}()

	if panicVal != nil {
		if tx.State == TxActive {
			tx.Rollback()
		}
		sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
		panic(panicVal)
	}

	if !sessionFn(scm.NewString("transaction")).IsNil() {
		return result
	}

	if err := tx.Commit(); err != nil {
		sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
		panic("autocommit failed: " + err.Error())
	}
	sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
	return result
}

func initTransaction(en scm.Env) {
	scm.DeclareTitle("Transactions")

	scm.Declare(&en, &scm.Declaration{
		Name:         "tx_begin",
		Desc:         "Begins a new cursor-stability transaction. Takes the session function as argument. Stores the transaction context in the session.",
		MinParameter: 1,
		MaxParameter: 1,
		Params: []scm.DeclarationParameter{
			{Name: "session", Type: "func", Desc: "the session function to store tx state in"},
		},
		Returns: "bool",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					tx.Commit()
				}
			}
			tx := NewTxContext(TxCursorStability)
			sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))
			sessionFn(scm.NewString("transaction"), scm.NewInt(1))
			return scm.NewBool(true)
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name:         "tx_begin_acid",
		Desc:         "Begins a new ACID transaction with snapshot isolation and OCC commit. Takes the session function as argument.",
		MinParameter: 1,
		MaxParameter: 1,
		Params: []scm.DeclarationParameter{
			{Name: "session", Type: "func", Desc: "the session function to store tx state in"},
		},
		Returns: "bool",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					tx.Commit()
				}
			}
			tx := NewTxContext(TxACID)
			sessionFn(scm.NewString("__memcp_tx"), scm.NewAny(tx))
			sessionFn(scm.NewString("transaction"), scm.NewInt(1))
			return scm.NewBool(true)
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name:         "tx_commit",
		Desc:         "Commits the current transaction.",
		MinParameter: 1,
		MaxParameter: 1,
		Params: []scm.DeclarationParameter{
			{Name: "session", Type: "func", Desc: "the session function that holds tx state"},
		},
		Returns: "bool",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					if err := tx.Commit(); err != nil {
						sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
						sessionFn(scm.NewString("transaction"), scm.NewNil())
						panic("COMMIT failed: " + err.Error())
					}
				}
			}
			sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
			sessionFn(scm.NewString("transaction"), scm.NewNil())
			return scm.NewBool(true)
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name:         "tx_rollback",
		Desc:         "Rolls back the current transaction.",
		MinParameter: 1,
		MaxParameter: 1,
		Params: []scm.DeclarationParameter{
			{Name: "session", Type: "func", Desc: "the session function that holds tx state"},
		},
		Returns: "bool",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			sessionFn := a[0].Func()
			existingTx := sessionFn(scm.NewString("__memcp_tx"))
			if !existingTx.IsNil() {
				if tx, ok := existingTx.Any().(*TxContext); ok && tx.State == TxActive {
					tx.Rollback()
				}
			}
			sessionFn(scm.NewString("__memcp_tx"), scm.NewNil())
			sessionFn(scm.NewString("transaction"), scm.NewNil())
			return scm.NewBool(true)
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "with_autocommit",
		Desc: "Executes fn inside an implicit TxCursorStability transaction if no explicit " +
			"transaction is active in session. Commits on success, rolls back on error, " +
			"and re-raises any panic so the caller's error handler still fires. " +
			"If an explicit transaction is active (session[\"transaction\"] != nil), " +
			"fn is executed without any wrapping.",
		MinParameter: 2,
		MaxParameter: 2,
		Params: []scm.DeclarationParameter{
			{Name: "session", Type: "func", Desc: "the session function holding tx state"},
			{Name: "fn", Type: "func", Desc: "zero-argument function to execute"},
		},
		Returns: "any",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			return WithAutocommit(a[0].Func(), a[1])
		},
	})
}
