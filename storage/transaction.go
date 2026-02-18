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
import "sort"
import "sync"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

// TxMode selects the transaction isolation strategy.
type TxMode uint8

const (
	TxCursorStability TxMode = iota // default: direct writes + undo log
	TxACID                          // snapshot isolation + OCC commit
)

// TxState tracks the lifecycle of a transaction.
type TxState uint8

const (
	TxActive TxState = iota
	TxCommitted
	TxAborted
)

// UndoType identifies the kind of undo operation (cursor-stability only).
type UndoType int

const (
	UndoInsert UndoType = iota // rollback: mark inserted row as deleted
	UndoDelete                 // rollback: undelete the deleted row
)

// UndoEntry records one reversible storage operation (cursor-stability only).
type UndoEntry struct {
	Type     UndoType
	Shard    *storageShard
	RowIndex uint32
}

// shardOverlay provides O(1) visibility checks via bitmap and an iteration
// list of recids for commit-time application.
type shardOverlay struct {
	Bitmap NonLockingReadMap.NonBlockingBitMap // O(1) visibility check in scan hot path
	Recids []uint32                            // for commit-time iteration
	mu     sync.Mutex                          // protects Recids append from parallel workers
}

// Add records a recid in both the bitmap and the iteration list.
func (o *shardOverlay) Add(recid uint32) {
	o.Bitmap.Set(recid, true)
	o.mu.Lock()
	o.Recids = append(o.Recids, recid)
	o.mu.Unlock()
}

// Has checks whether a recid is in this overlay (O(1) via bitmap).
func (o *shardOverlay) Has(recid uint32) bool {
	return o.Bitmap.Get(recid)
}

// Savepoint captures the state of a transaction at a point in time.
// Used for nested transactions (trigger recovery, savepoints).
type Savepoint struct {
	UndoLogLen       int                   // cursor-stability: UndoLog length at savepoint
	DeleteMaskLens   map[*storageShard]int // ACID: Recids length per shard in DeleteMask
	UndeleteMaskLens map[*storageShard]int // ACID: Recids length per shard in UndeleteMask
	Depth            uint32                // nesting depth at creation
}

// global transaction ID counter
var txIDCounter uint64

// GlobalCommitEpoch is advanced on each ACID commit.
var GlobalCommitEpoch uint64

// TxContext holds the state for one transaction.
type TxContext struct {
	ID            uint64
	Mode          TxMode
	State         TxState
	SnapshotEpoch uint64 // ACID: snapshot boundary
	Depth         uint32 // for savepoint/trigger nesting (future)

	// Cursor-stability: undo log
	UndoLog []UndoEntry

	// ACID: per-shard overlays
	DeleteMask   map[*storageShard]*shardOverlay // recids this tx wants to delete
	UndeleteMask map[*storageShard]*shardOverlay // staged insert recids visible to this tx

	// Deferred sync (§10)
	touchedShards sync.Map // map[*storageShard]bool
	autoCommit    bool

	mu sync.Mutex
}

// NewTxContext creates a new active transaction context with the given mode.
func NewTxContext(mode TxMode) *TxContext {
	tx := &TxContext{
		ID:    atomic.AddUint64(&txIDCounter, 1),
		State: TxActive,
		Mode:  mode,
	}
	switch mode {
	case TxCursorStability:
		tx.UndoLog = make([]UndoEntry, 0, 16)
	case TxACID:
		tx.SnapshotEpoch = atomic.LoadUint64(&GlobalCommitEpoch)
		tx.DeleteMask = make(map[*storageShard]*shardOverlay)
		tx.UndeleteMask = make(map[*storageShard]*shardOverlay)
	}
	return tx
}

// CreateSavepoint captures the current transaction state for later rollback.
func (tx *TxContext) CreateSavepoint() Savepoint {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	sp := Savepoint{Depth: tx.Depth}
	tx.Depth++
	switch tx.Mode {
	case TxCursorStability:
		sp.UndoLogLen = len(tx.UndoLog)
	case TxACID:
		sp.DeleteMaskLens = make(map[*storageShard]int, len(tx.DeleteMask))
		for s, overlay := range tx.DeleteMask {
			overlay.mu.Lock()
			sp.DeleteMaskLens[s] = len(overlay.Recids)
			overlay.mu.Unlock()
		}
		sp.UndeleteMaskLens = make(map[*storageShard]int, len(tx.UndeleteMask))
		for s, overlay := range tx.UndeleteMask {
			overlay.mu.Lock()
			sp.UndeleteMaskLens[s] = len(overlay.Recids)
			overlay.mu.Unlock()
		}
	}
	return sp
}

// RollbackToSavepoint undoes all changes made since the savepoint was created.
func (tx *TxContext) RollbackToSavepoint(sp Savepoint) {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	tx.Depth = sp.Depth
	switch tx.Mode {
	case TxCursorStability:
		// Replay undo entries from current position back to savepoint
		for i := len(tx.UndoLog) - 1; i >= sp.UndoLogLen; i-- {
			entry := tx.UndoLog[i]
			entry.Shard.mu.Lock()
			switch entry.Type {
			case UndoInsert:
				entry.Shard.deletions.Set(entry.RowIndex, true)
				if entry.Shard.logfile != nil {
					entry.Shard.logfile.Write(LogEntryDelete{entry.RowIndex})
				}
			case UndoDelete:
				entry.Shard.deletions.Set(entry.RowIndex, false)
			}
			entry.Shard.mu.Unlock()
		}
		tx.UndoLog = tx.UndoLog[:sp.UndoLogLen]
	case TxACID:
		// Rollback DeleteMask entries added since savepoint
		for shard, overlay := range tx.DeleteMask {
			savedLen := sp.DeleteMaskLens[shard] // 0 if shard was new
			overlay.mu.Lock()
			for i := len(overlay.Recids) - 1; i >= savedLen; i-- {
				recid := overlay.Recids[i]
				overlay.Bitmap.Set(recid, false)
			}
			overlay.Recids = overlay.Recids[:savedLen]
			overlay.mu.Unlock()
		}
		// Rollback UndeleteMask entries added since savepoint
		for shard, overlay := range tx.UndeleteMask {
			savedLen := sp.UndeleteMaskLens[shard] // 0 if shard was new
			overlay.mu.Lock()
			for i := len(overlay.Recids) - 1; i >= savedLen; i-- {
				recid := overlay.Recids[i]
				overlay.Bitmap.Set(recid, false)
				// Re-hide the row globally (it was un-hidden by AddToUndeleteMask)
				shard.deletions.Set(recid, true)
			}
			overlay.Recids = overlay.Recids[:savedLen]
			overlay.mu.Unlock()
		}
	}
}

// LogInsert records that a row was inserted; on rollback it will be deleted.
// Cursor-stability only.
func (tx *TxContext) LogInsert(shard *storageShard, rowIndex uint32) {
	tx.mu.Lock()
	tx.UndoLog = append(tx.UndoLog, UndoEntry{
		Type:     UndoInsert,
		Shard:    shard,
		RowIndex: rowIndex,
	})
	tx.mu.Unlock()
}

// LogDelete records that a row was deleted; on rollback it will be undeleted.
// Cursor-stability only.
func (tx *TxContext) LogDelete(shard *storageShard, rowIndex uint32) {
	tx.mu.Lock()
	tx.UndoLog = append(tx.UndoLog, UndoEntry{
		Type:     UndoDelete,
		Shard:    shard,
		RowIndex: rowIndex,
	})
	tx.mu.Unlock()
}

// AddToDeleteMask records that this ACID tx wants to delete a row.
func (tx *TxContext) AddToDeleteMask(shard *storageShard, recid uint32) {
	tx.mu.Lock()
	overlay, ok := tx.DeleteMask[shard]
	if !ok {
		overlay = new(shardOverlay)
		tx.DeleteMask[shard] = overlay
	}
	tx.mu.Unlock()
	overlay.Add(recid)
}

// AddToUndeleteMask records that this ACID tx can see a staged row.
func (tx *TxContext) AddToUndeleteMask(shard *storageShard, recid uint32) {
	tx.mu.Lock()
	overlay, ok := tx.UndeleteMask[shard]
	if !ok {
		overlay = new(shardOverlay)
		tx.UndeleteMask[shard] = overlay
	}
	tx.mu.Unlock()
	overlay.Add(recid)
}

// RegisterTouchedShard marks a shard as having pending writes for deferred sync.
func (tx *TxContext) RegisterTouchedShard(shard *storageShard) {
	tx.touchedShards.Store(shard, true)
}

// SyncTouchedShards flushes all pending log writes to durable storage.
func (tx *TxContext) SyncTouchedShards() {
	tx.touchedShards.Range(func(key, _ any) bool {
		shard := key.(*storageShard)
		if shard.t.PersistencyMode == Safe {
			shard.logfile.Sync()
		}
		return true
	})
	tx.touchedShards = sync.Map{} // clear for next query/transaction
}

// IsVisible determines whether a row is visible to this ACID transaction.
// Formula: (!shard->delete[i] && !tx->delete[i]) || tx->undelete[i]
// UndeleteMask always wins — it is the only way an ACID tx sees its own inserts.
func (tx *TxContext) IsVisible(shard *storageShard, recid uint32) bool {
	tx.mu.Lock()
	dm := tx.DeleteMask[shard]
	um := tx.UndeleteMask[shard]
	tx.mu.Unlock()
	// undelete mask overrides everything — this tx staged this row
	if um != nil && um.Has(recid) {
		return true
	}
	// otherwise: not globally deleted AND not locally deleted
	return !shard.deletions.Get(recid) && (dm == nil || !dm.Has(recid))
}

// Commit finalizes the transaction. For cursor-stability it discards the undo
// log. For ACID it runs OCC validation and applies overlay masks.
func (tx *TxContext) Commit() error {
	switch tx.Mode {
	case TxCursorStability:
		tx.mu.Lock()
		tx.State = TxCommitted
		tx.UndoLog = nil
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
	// Collect all shards that have overlays
	shardSet := make(map[*storageShard]bool)
	for s := range tx.DeleteMask {
		shardSet[s] = true
	}
	for s := range tx.UndeleteMask {
		shardSet[s] = true
	}
	tx.mu.Unlock()

	// Sort shards by UUID string for deterministic lock ordering
	shards := make([]*storageShard, 0, len(shardSet))
	for s := range shardSet {
		shards = append(shards, s)
	}
	sort.Slice(shards, func(i, j int) bool {
		return shards[i].uuid.String() < shards[j].uuid.String()
	})

	// Lock all touched shards
	for _, s := range shards {
		s.mu.Lock()
	}

	// Validate: for each recid in DeleteMask, check it's not already
	// globally deleted (another tx committed first → conflict → abort)
	// Note: DeleteMask bits are never cleared, so no bitmap skip needed.
	for shard, overlay := range tx.DeleteMask {
		for _, recid := range overlay.Recids {
			if shard.deletions.Get(recid) {
				// Conflict: row was already deleted by another committed tx
				// Unlock and abort
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

	// Apply: merge DeleteMask → set global deletions + log
	for shard, overlay := range tx.DeleteMask {
		for _, recid := range overlay.Recids {
			shard.deletions.Set(recid, true)
			if shard.logfile != nil {
				shard.logfile.Write(LogEntryDelete{recid})
			}
		}
	}
	// Apply: merge UndeleteMask → clear global deletions (make staged rows visible)
	for shard, overlay := range tx.UndeleteMask {
		for _, recid := range overlay.Recids {
			if !overlay.Bitmap.Get(recid) {
				continue // removed (e.g., staged row superseded by UPDATE/DELETE in same tx)
			}
			shard.deletions.Set(recid, false)
			// The row data was already inserted globally; just making it visible.
			// The logfile already has the Insert entry from when the row was staged.
		}
	}

	// Advance global commit epoch
	atomic.AddUint64(&GlobalCommitEpoch, 1)

	// Unlock shards
	for _, s := range shards {
		s.mu.Unlock()
	}

	tx.mu.Lock()
	tx.State = TxCommitted
	tx.DeleteMask = nil
	tx.UndeleteMask = nil
	tx.mu.Unlock()

	tx.SyncTouchedShards()
	return nil
}

// Rollback undoes the transaction. For cursor-stability it replays the undo
// log. For ACID it discards overlay masks (staged rows stay as garbage).
func (tx *TxContext) Rollback() {
	switch tx.Mode {
	case TxCursorStability:
		tx.rollbackCursorStability()
	case TxACID:
		tx.rollbackACID()
	}
}

// rollbackCursorStability replays the undo log in reverse order.
func (tx *TxContext) rollbackCursorStability() {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	for i := len(tx.UndoLog) - 1; i >= 0; i-- {
		entry := tx.UndoLog[i]
		entry.Shard.mu.Lock()
		switch entry.Type {
		case UndoInsert:
			// undo an insert: mark the inserted row as deleted
			entry.Shard.deletions.Set(entry.RowIndex, true)
			if entry.Shard.logfile != nil {
				entry.Shard.logfile.Write(LogEntryDelete{entry.RowIndex})
			}
		case UndoDelete:
			// undo a delete: undelete the row
			entry.Shard.deletions.Set(entry.RowIndex, false)
			if entry.Shard.logfile != nil {
				t := entry.Shard
				if entry.RowIndex >= t.main_count {
					deltaIdx := int(entry.RowIndex - t.main_count)
					if deltaIdx < len(t.inserts) {
						row := t.inserts[deltaIdx]
						cols := make([]string, 0, len(t.deltaColumns))
						vals := make([]scm.Scmer, 0, len(t.deltaColumns))
						for name, idx := range t.deltaColumns {
							if idx < len(row) {
								cols = append(cols, name)
								vals = append(vals, row[idx])
							}
						}
						t.logfile.Write(LogEntryInsert{cols, [][]scm.Scmer{vals}})
					}
				}
				// For main storage rows, persistence after rollback of
				// main-row deletes is imperfect across restarts.
			}
		}
		entry.Shard.mu.Unlock()
	}
	tx.State = TxAborted
	tx.UndoLog = nil
	// No sync needed for rollback — discard touched shards
	tx.touchedShards = sync.Map{}
}

// rollbackACID discards overlay masks. Staged rows that were globally hidden
// remain as garbage for GC.
func (tx *TxContext) rollbackACID() {
	tx.mu.Lock()
	tx.State = TxAborted
	tx.DeleteMask = nil
	tx.UndeleteMask = nil
	tx.mu.Unlock()
	// No sync needed — discard touched shards
	tx.touchedShards = sync.Map{}
}

// CurrentTx returns the active TxContext from the goroutine-local storage,
// or nil if no transaction is active.
func CurrentTx() *TxContext {
	txAny := scm.GetCurrentTx()
	if txAny == nil {
		return nil
	}
	tx, _ := txAny.(*TxContext)
	return tx
}

// initTransaction registers the tx_begin, tx_begin_acid, tx_commit,
// tx_rollback builtins.
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
			// Check if there's already an active transaction — implicit commit
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
			// Check if there's already an active transaction — implicit commit
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
		Desc:         "Commits the current transaction. For cursor-stability: discards undo log. For ACID: validates and applies overlay masks.",
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
		Desc:         "Rolls back the current transaction. For cursor-stability: replays undo log. For ACID: discards overlay masks.",
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
}
