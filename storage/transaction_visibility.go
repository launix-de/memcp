/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "maps"
import "weak"
import "math/bits"
import "sync/atomic"
import NonLockingReadMap "github.com/launix-de/NonLockingReadMap"

// setOwnedVisibilityBit requires the bitmap's existing exclusive native writer:
// st.mu for actual journal/history masks, result.mu for private rebuild output,
// next.mu (or its preheld write owner) for compensation forwarding. Imported
// masks are initialized before the new input is published. Registration and
// retirement do not write words. Readers use atomic loads, never this helper.
//
// In-capacity words need no bitmap resize lock because the same writer also
// owns every grow. AtomicSet remains the library's allocation/publication path;
// an absent false bit needs neither a word nor a resize operation.
func setOwnedVisibilityBit(mask *NonLockingReadMap.NonBlockingBitMap, recid uint, value bool) {
	words := mask.DataPtr()
	wordIndex := recid >> 6
	if words == nil || wordIndex >= uint(len(*words)) {
		if value {
			mask.AtomicSet(recid, true)
		}
		return
	}
	word := &(*words)[wordIndex]
	previous := atomic.LoadUint64(word)
	bit := uint64(1) << (recid & 63)
	if value {
		previous |= bit
	} else {
		previous &^= bit
	}
	atomic.StoreUint64(word, previous)
}

// A rebuild projection supplies visibility only. Original native journals keep
// every undo/decision obligation; these masks never become participants. Weak
// targets let a live transaction discard obsolete compressed generations.
type transactionVisibilityProjection struct {
	target   weak.Pointer[storageShard]
	visible  NonLockingReadMap.NonBlockingBitMap
	hidden   NonLockingReadMap.NonBlockingBitMap
	released NonLockingReadMap.NonBlockingBitMap
}

type pendingVisibilityChange struct {
	ownerID uint64
	recid   uint32
}

var visibilityIdentityCounter atomic.Uint64

func (state *transactionState) visibilityProjection(shard *storageShard) *transactionVisibilityProjection {
	if identity := shard.visibilityIdentity.Load(); identity != 0 {
		if directory := state.visibilityProjections.Load(); directory != nil {
			if value := (*directory)[identity]; value != nil {
				return value
			}
		}
	}
	return nil
}

func (state *transactionState) retainVisibilityProjection(shard *storageShard) *transactionVisibilityProjection {
	if state.visibilityClosed.Load() {
		return nil
	}
	if projection := state.visibilityProjection(shard); projection != nil {
		return projection
	}
	identity := shard.visibilityIdentity.Load()
	if identity == 0 {
		identity = visibilityIdentityCounter.Add(1)
		if !shard.visibilityIdentity.CompareAndSwap(0, identity) {
			identity = shard.visibilityIdentity.Load()
		}
	}
	candidate := &transactionVisibilityProjection{target: weak.Make(shard)}
	var projection *transactionVisibilityProjection
	for {
		if state.visibilityClosed.Load() {
			return nil
		}
		previous := state.visibilityProjections.Load()
		if previous != nil {
			if value := (*previous)[identity]; value != nil {
				return value
			}
		}
		next := make(map[uint64]*transactionVisibilityProjection)
		if previous != nil {
			next = maps.Clone(*previous)
		}
		for key, value := range next {
			if value.target.Value() == nil {
				delete(next, key)
			}
		}
		next[identity] = candidate
		if state.visibilityProjections.CompareAndSwap(previous, &next) {
			projection = candidate
			break
		}
	}
	shard.visibilityProjections.Store(state, projection)
	// A decision seals admission before retiring registrations. No mutable
	// directory or sync.Map bookkeeping is reachable from row readers.
	if state.visibilityClosed.Load() {
		state.visibilityProjections.Store(nil)
		shard.visibilityProjections.CompareAndDelete(state, projection)
		shard.t.signalTransactionDrain()
		return nil
	}
	return projection
}

func (state *transactionState) releaseVisibilityProjections() {
	state.visibilityClosed.Store(true)
	if directory := state.visibilityProjections.Swap(nil); directory != nil {
		for _, projection := range *directory {
			if target := projection.target.Value(); target != nil {
				target.visibilityProjections.CompareAndDelete(state, projection)
				target.t.signalTransactionDrain()
			}
		}
	}
}

// Fixed native lookups merge inherited visibility with actual local mutations.
// stagedHistory indexes existing journal entries, including an explicit false
// bit after UnstageRow; absence must not resurrect an inherited private row.
func (tx *TxContext) projectedVisibility(shard *storageShard, recid uint32) (visible, hidden, released bool) {
	if projection := tx.visibilityProjection(shard); projection != nil {
		visible = projection.visible.AtomicGet(uint(recid))
		hidden = projection.hidden.AtomicGet(uint(recid))
		released = projection.released.AtomicGet(uint(recid))
	}
	if st := tx.getShardTx(shard); st != nil {
		if tx.Mode == TxCursorStability {
			return visible, hidden, released || st.DeletedMask.AtomicGet(uint(recid))
		}
		if st.DeleteMask.AtomicGet(uint(recid)) || st.hasStagedHistory(recid) {
			visible, hidden, released = false, true, true
		}
		if st.UndeleteMask.AtomicGet(uint(recid)) {
			visible, hidden, released = true, false, false
		}
	}
	return
}

func (st *storageShardTransaction) rebuildStagedHistory() {
	history := new(NonLockingReadMap.NonBlockingBitMap)
	for _, change := range st.UndeleteRecids {
		setOwnedVisibilityBit(history, uint(change.recid), true)
	}
	st.stagedHistory.Store(history)
}

func (st *storageShardTransaction) hasStagedHistory(recid uint32) bool {
	history := st.stagedHistory.Load()
	return history != nil && history.AtomicGet(uint(recid))
}

func (st *storageShardTransaction) markStagedHistory(recid uint32) {
	history := st.stagedHistory.Load()
	if history == nil {
		history = new(NonLockingReadMap.NonBlockingBitMap)
		st.stagedHistory.Store(history) // native journal mutation already owns st.mu
	}
	setOwnedVisibilityBit(history, uint(recid), true)
}

// The source mutation already owns source.mu. Private builders queue only
// changed coordinates; ready successors receive physical writes and undo through
// the existing native forwarding owners. No reader writes or walks ancestry.
func (tx *TxContext) forwardVisibilityLocked(source *storageShard, recid uint32) {
	if next := source.loadNext(); next != nil {
		if !source.nextReady.Load() {
			source.nextPendingVisibility = append(source.nextPendingVisibility,
				pendingVisibilityChange{tx.ID, recid})
			return
		}
		// Ready generations already receive the actual physical mutation under
		// their native writer. Its local journal overrides inherited masks. Undo
		// changes instead update projections in syncNextVisibilityLocked, using
		// the existing visibility-forwarding write owner.
	}
}

func (tx *TxContext) projectVisibilityRow(source *storageShard, recid uint32, target *storageShard, translated uint32) {
	projection := tx.retainVisibilityProjection(target)
	if projection == nil {
		return
	}
	visible, hidden, released := tx.projectedVisibility(source, recid)
	setOwnedVisibilityBit(&projection.visible, uint(translated), visible)
	setOwnedVisibilityBit(&projection.hidden, uint(translated), hidden)
	setOwnedVisibilityBit(&projection.released, uint(translated), released)
}

// A capture retains only fixed bitmap backing-slice headers. Writers use
// atomic word stores and queue changed coordinates while construction runs; traversal
// and merging are exclusively private builder work, never source.mu work.
type transactionVisibilityCapture struct {
	mode                                                           TxMode
	visible, hidden, released, deleted, staged, history, csDeleted *[]uint64
}

func capturedVisibilityBit(words *[]uint64, recid uint32) bool {
	return words != nil && int(recid>>6) < len(*words) &&
		atomic.LoadUint64(&(*words)[recid>>6])&(uint64(1)<<(recid&63)) != 0
}

func (source *storageShard) visibilityOwners(extra ...*storageShardTransaction) map[*transactionState]bool {
	owners := make(map[*transactionState]bool)
	source.overlays.Range(func(key, _ any) bool { owners[key.(*transactionState)] = true; return true })
	source.visibilityProjections.Range(func(key, _ any) bool { owners[key.(*transactionState)] = true; return true })
	for _, st := range extra {
		if st.clusterShard == source {
			owners[st.transaction] = true
		}
	}
	return owners
}

func (source *storageShard) captureVisibilityLocked(extra ...*storageShardTransaction) map[*transactionState]*transactionVisibilityCapture {
	owners := source.visibilityOwners(extra...)
	snapshot := make(map[*transactionState]*transactionVisibilityCapture, len(owners))
	for owner := range owners {
		if owner.visibilityClosed.Load() {
			continue
		}
		capture := &transactionVisibilityCapture{mode: owner.Mode}
		if inherited := owner.visibilityProjection(source); inherited != nil {
			capture.visible = inherited.visible.DataPtr()
			capture.hidden = inherited.hidden.DataPtr()
			capture.released = inherited.released.DataPtr()
		}
		if st := (&TxContext{transactionState: owner}).getShardTx(source); st != nil {
			capture.deleted = st.DeleteMask.DataPtr()
			capture.staged = st.UndeleteMask.DataPtr()
			capture.csDeleted = st.DeletedMask.DataPtr()
			if history := st.stagedHistory.Load(); history != nil {
				capture.history = history.DataPtr()
			}
		}
		snapshot[owner] = capture
	}
	return snapshot
}

func projectRebuildVisibility(target *storageShard, translation map[uint32]uint32, snapshot map[*transactionState]*transactionVisibilityCapture) {
	for owner, source := range snapshot {
		projection := owner.retainVisibilityProjection(target)
		if projection == nil {
			continue
		}
		coordinates := make(map[uint32]bool)
		for _, words := range []*[]uint64{source.visible, source.hidden, source.released,
			source.deleted, source.staged, source.history, source.csDeleted} {
			if words == nil {
				continue
			}
			for index := range *words {
				word := atomic.LoadUint64(&(*words)[index])
				for word != 0 {
					bit := bits.TrailingZeros64(word)
					coordinates[uint32(index*64+bit)] = true
					word &= word - 1
				}
			}
		}
		for recid := range coordinates {
			translated, found := translation[recid]
			if !found {
				continue
			}
			visible := capturedVisibilityBit(source.visible, recid)
			hidden := capturedVisibilityBit(source.hidden, recid)
			released := capturedVisibilityBit(source.released, recid)
			if source.mode == TxACID {
				if capturedVisibilityBit(source.deleted, recid) || capturedVisibilityBit(source.history, recid) {
					visible, hidden, released = false, true, true
				}
				if capturedVisibilityBit(source.staged, recid) {
					visible, hidden, released = true, false, false
				}
			} else {
				released = released || capturedVisibilityBit(source.csDeleted, recid)
			}
			setOwnedVisibilityBit(&projection.visible, uint(translated), visible)
			setOwnedVisibilityBit(&projection.hidden, uint(translated), hidden)
			setOwnedVisibilityBit(&projection.released, uint(translated), released)
		}
	}
}

// Existing live transfer already carries the original native journals and
// exact Next translations. Reconstruct non-owning masks from those same owners
// after every generation has decoded, without changing the transport format.
func restoreTransferredVisibility(grains map[string]*storageShard, overlays []*storageShardTransaction) {
	visited := make(map[*storageShard]bool)
	var visit func(*storageShard)
	visit = func(source *storageShard) {
		if source == nil || visited[source] {
			return
		}
		visited[source] = true
		if previous := source.rebuildSource.Value(); previous != nil {
			visit(previous)
		}
		next := source.loadNext()
		if next == nil || !source.nextReady.Load() {
			return
		}
		// Frozen remote views contribute headers, never mutation authority.
		snapshot := source.captureVisibilityLocked(overlays...)
		projectRebuildVisibility(next, source.nextTranslation, snapshot)
	}
	for _, source := range grains {
		visit(source)
	}
}
