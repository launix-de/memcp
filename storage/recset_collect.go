/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "sort"
import "runtime"
import "math/bits"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// A collector consumes borrowed native record references, not column tuples.
// Its bitmap lanes are invocation-owned, deduplicate arbitrary producer order,
// and become an ordinary immutable RecSet only after the producer has drained.
// No callback retains a transaction, session or reader after this operator ends.
type recSetCollector struct {
	target *table
	tx     *TxContext // actual collection invocation; cleared before an escaped callback can survive
	parts  sync.Map   // actual source shard -> invocation-owned fixed bitmap chunks
	closed atomic.Bool
	active atomic.Int64
}

const recSetCollectorChunkWords = 256

type recSetCollectorPart struct {
	chunks sync.Map // uint32 word-chunk index -> *[recSetCollectorChunkWords]atomic.Uint64
}

func (collector *recSetCollector) accept(value scm.Scmer) {
	if collector.closed.Load() {
		panic("native RecSet collector has finished")
	}
	collector.active.Add(1)
	defer collector.active.Add(-1)
	if collector.closed.Load() {
		panic("native RecSet collector has finished")
	}
	// A NULL-extended source row has no physical identity to collect.
	if value.IsNil() {
		return
	}
	reference := recordRefFromScmer(value)
	if reference.shard == nil || reference.shard.t != collector.target {
		panic("native RecSet collector received another relation's record reference")
	}
	part, found := collector.parts.Load(reference.shard)
	if !found {
		// The native source callback still owns this exact generation. Enlist
		// the intended target once before its source reader returns, using the
		// existing transaction/generation owner. A later RecSet consumer must
		// not reinterpret these recids on a newly published successor shard.
		collector.tx.mu.Lock()
		collector.tx.getOrCreateShardTxLocked(reference.shard)
		collector.tx.mu.Unlock()
		part, _ = collector.parts.LoadOrStore(reference.shard, &recSetCollectorPart{})
	}
	word := reference.recid / 64
	chunkIndex := word / recSetCollectorChunkWords
	chunks := &part.(*recSetCollectorPart).chunks
	chunk, found := chunks.Load(chunkIndex)
	if !found {
		chunk, _ = chunks.LoadOrStore(chunkIndex, new([recSetCollectorChunkWords]atomic.Uint64))
	}
	chunk.(*[recSetCollectorChunkWords]atomic.Uint64)[word%recSetCollectorChunkWords].Or(uint64(1) << (reference.recid % 64))
}

func (collector *recSetCollector) seal() {
	collector.closed.Store(true)
	// Normal scan producers already join their workers before returning. An
	// accepted concurrent callback still owns its bitmap write until it exits;
	// a later escaped callback is rejected without touching collector payload.
	for collector.active.Load() != 0 {
		runtime.Gosched()
	}
}

func (collector *recSetCollector) clear() {
	collector.parts.Clear()
	collector.target = nil
	collector.tx = nil
}

func collectRecSet(tx *TxContext, target *table, producer scm.Scmer) *recSet {
	tx, finishData := target.writeDataContext(tx)
	defer finishData()

	if tx == nil || !tx.queryActive.Load() {
		panic("native record-reference collection requires its actual query invocation")
	}
	tx.requireMutationBinding()
	if sharedClusterSource(target) {
		panic("native record-reference collection currently requires a local target")
	}
	collector := &recSetCollector{target: target, tx: tx}
	defer func() {
		collector.seal()
		collector.clear()
	}()
	scm.Apply(producer, scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		if len(values) != 1 {
			panic("native RecSet collector requires one record reference")
		}
		collector.accept(values[0])
		return scm.NewNil()
	}))
	collector.seal()
	result := &recSet{table: target}
	collector.parts.Range(func(key, value any) bool {
		shard := key.(*storageShard)
		// Snapshot rights preserve the original generation and its recids,
		// exactly as RecMap.image; they must never follow a rebuild successor.
		release := shard.acquireSnapshotRead(tx)
		defer release.Release()
		shard.mu.RLock()
		universe := shard.main_count + uint32(len(shard.deltaRows()))
		shard.mu.RUnlock()
		part := value.(*recSetCollectorPart)
		var chunkIndexes []uint32
		part.chunks.Range(func(key, _ any) bool {
			chunkIndexes = append(chunkIndexes, key.(uint32))
			return true
		})
		sort.Slice(chunkIndexes, func(i, j int) bool { return chunkIndexes[i] < chunkIndexes[j] })
		var ids []uint32
		for _, chunkIndex := range chunkIndexes {
			chunk, _ := part.chunks.Load(chunkIndex)
			for index := range recSetCollectorChunkWords {
				word := chunk.(*[recSetCollectorChunkWords]atomic.Uint64)[index].Load()
				for word != 0 {
					id := (uint64(chunkIndex)*recSetCollectorChunkWords+uint64(index))*64 + uint64(bits.TrailingZeros64(word))
					if id >= uint64(universe) {
						panic("native RecSet collector received an invalid record identity")
					}
					ids = append(ids, uint32(id))
					word &= word - 1
				}
			}
		}
		selected := newRecSetShardFromSortedIDs(shard, universe, ids)
		result.count += selected.count
		result.shards = append(result.shards, selected)
		return true
	})
	return result
}
