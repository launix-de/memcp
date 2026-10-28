/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "math"
import "sync"
import "unsafe"
import "runtime"
import "math/bits"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// UNIQUE guards belong to the logical table, independently of physical shard
// generations. Equal keys use the same stripe across rebuild/repartition; an
// unrelated key does not hold a table-wide mutex during lookup, WAL or fsync.
// Large batches acquire their stripe set in ascending order, once per batch.
type uniqueWriteLocks [64]sync.Mutex

type uniqueWriteGuard struct {
	locks      *uniqueWriteLocks
	mask       uint64
	held       bool
	readProofs []uniqueReadProof
	wrote      bool
	version    uint64 // advances when a collision callback temporarily releases guards
}

// Read validation adds no reader registration. The existing generation and MOSI
// publication prove that another node has not invalidated a UNIQUE input. The
// success callback validates only after acquiring its native destination writer.
type uniqueReadProof struct {
	table    *table
	topology *tableShardTopology
	shard    *storageShard
}

type uniqueReadRetry struct{}

func (g *uniqueWriteGuard) validReads() bool {
	for _, proof := range g.readProofs {
		if proof.table != nil {
			if proof.table.topology.Load().routing != proof.topology.routing || !activeNodeCluster().manager.resourceCell(mosiResource{Database: proof.table.schema.Name}).ownstate.Phase().Readable() {
				return false
			}
		} else {
			cell := proof.shard.coherence.Load()
			if cell == nil {
				return false
			}
			cell = cell.resolved()
			if !cell.ownstate.Phase().Readable() || cell.publishedShard() != proof.shard {
				return false
			}
		}
	}
	return true
}

func (g *uniqueWriteGuard) lock() {
	for mask := g.mask; mask != 0; mask &= mask - 1 {
		g.locks[bits.TrailingZeros64(mask)].Lock()
	}
	g.held = true
}

func (g *uniqueWriteGuard) unlock() {
	if !g.held {
		return
	}
	for mask := g.mask; mask != 0; {
		i := bits.Len64(mask) - 1
		g.locks[i].Unlock()
		mask &^= uint64(1) << i
	}
	g.held = false
	g.version++
}

// Equal is deliberately coercing: ints can equal numeric strings/dates and
// floats can equal rounded ints. Truncation groups those numeric representations
// conservatively. Booleans, zero, NULL and opaque values bridge otherwise
// separate equality domains, so they must intersect every stripe. Collisions
// merely reduce parallelism; hash equality never establishes UNIQUE membership.
func uniqueGuardHash(value scm.Scmer) (uint64, bool) {
	var number float64
	switch {
	case value.IsInt(), value.IsFloat():
		number = scm.ToFloat(value)
	case value.IsString(), value.IsSymbol():
		text := scm.String(value)
		// Use the runtime's numeric-prefix conversion ("12x" equals 12).
		number = value.Float()
		if number == 0 {
			return scm.HashKey(scm.NewString(text)), true
		}
	default:
		return 0, false
	}
	number = math.Trunc(number)
	if number == 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return scm.HashKey(scm.NewFloat(number)), true
}

func (t *table) guardUniqueWrites(columns []string, values [][]scm.Scmer, mergeNull bool, first int) uniqueWriteGuard {
	var mask uint64
	for idx := t.nextUniqueConstraint(columns, first); idx < len(t.Unique); idx = t.nextUniqueConstraint(columns, idx+1) {
		key := t.Unique[idx]
		var inlinePositions [8]int
		positions := inlinePositions[:]
		if len(key.Cols) <= len(positions) {
			positions = positions[:len(key.Cols)]
		} else {
			positions = make([]int, len(key.Cols))
		}
		for i, name := range key.Cols {
			positions[i] = -1
			for j, column := range columns {
				if column == name {
					positions[i] = j
					break
				}
			}
		}
		for _, row := range values {
			hash := uint64(idx + 1)
			skip, broad := false, false
			for _, position := range positions {
				value := scm.NewNil()
				if position >= 0 && position < len(row) {
					value = row[position]
				}
				if value.IsNil() && !mergeNull && !key.NullsNotDistinct {
					skip = true
					break
				}
				part, precise := uniqueGuardHash(value)
				broad = broad || !precise
				hash = (hash ^ part) * 1099511628211
			}
			if skip {
				continue
			}
			if broad {
				mask = ^uint64(0)
				break
			}
			mask |= uint64(1) << (hash & 63)
			if mask == ^uint64(0) {
				break
			}
		}
		if mask == ^uint64(0) {
			break
		}
	}
	locks := t.uniqueWrites.Load()
	if mask != 0 && locks == nil {
		locks = new(uniqueWriteLocks)
		if !t.uniqueWrites.CompareAndSwap(nil, locks) {
			locks = t.uniqueWrites.Load()
		} else if t.isEphemeralQueryTable() {
			GlobalCache.UpdateSizeAsync(t, int64(unsafe.Sizeof(*locks)))
		}
	}
	return uniqueWriteGuard{locks: locks, mask: mask}
}

// Reservations include queued/in-flight batches until they finish, not just
// already appended rows. They belong to a physical generation and never move
// to a successor: the pinned operation and existing forwarding own its writes.
func reserveInsertRows(shard *storageShard, rows int64, size uint, overflow bool, expected int64) (func(), bool) {
	for {
		pending := shard.pendingInsertRows.Load()
		if expected >= 0 && pending != expected {
			return nil, false
		}
		physical := uint64(shard.plannerMainRows.Load()) + shard.plannerDeltaRows.Load()
		if !overflow && physical+uint64(pending)+uint64(rows) > uint64(size) {
			return nil, false
		}
		if shard.pendingInsertRows.CompareAndSwap(pending, pending+rows) {
			return func() { shard.pendingInsertRows.Add(-rows) }, true
		}
	}
}

func (t *table) routeFreeInsert(topology *tableShardTopology, rows int64) (*storageShard, func(), *tableShardTopology) {
	if topology.mode != ShardModeFree || len(topology.shards) == 0 {
		return nil, nil, topology
	}
	// Inspect only the recent writer window; old full shards never turn routing
	// into a table-sized traversal. Concurrency is bounded by the Go CPU budget.
	budget := min(runtime.GOMAXPROCS(0), maxRepartitionShards)
	writers := topology.shards
	if sharedClusterSource(t) {
		// Foreign appends must not push this node's still-open writer lanes out
		// of the recent window. Collect at most the local CPU budget.
		writers = make([]*storageShard, 0, budget)
		for i := len(topology.shards) - 1; i >= 0 && len(writers) < budget; i-- {
			if t.localInsertShard(topology.shards[i]) {
				writers = append(writers, topology.shards[i])
			}
		}
	}
	window := min(len(writers), budget)
	start := len(writers) - window
	ticket := t.insertRoute.Add(1)
	var candidate *storageShard
	var leastPending int64
	writable := 0
	for i := 0; i < window; i++ {
		shard := writers[start+(int(ticket%uint64(window))+i)%window]
		if !t.localInsertShard(shard) {
			continue
		}
		pending := shard.pendingInsertRows.Load()
		physical := uint64(shard.plannerMainRows.Load()) + shard.plannerDeltaRows.Load()
		if physical+uint64(pending)+uint64(rows) > uint64(Settings.ShardSize) {
			continue
		}
		writable++
		if candidate == nil || pending < leastPending {
			candidate, leastPending = shard, pending
		}
	}
	if candidate != nil && (leastPending == 0 || writable >= budget) {
		if release, ok := reserveInsertRows(candidate, rows, Settings.ShardSize, false, leastPending); ok {
			return candidate, release, topology
		}
		return nil, nil, nil // another writer won; reload and rebalance
	}
	// Under pressure publish another independent writer lane. If maintenance is
	// already running, keep writing to the live generation instead of waiting
	// behind its compression/schema I/O. Existing forwarding preserves handover.
	source := topology.shards[len(topology.shards)-1]
	next, _ := t.appendFreeShardDurably(topology, source, uint(rows), candidate != nil || !t.localInsertShard(source))
	if next != nil && next != topology {
		return nil, nil, next
	}
	if candidate == nil {
		if !t.localInsertShard(source) {
			// Maintenance must never make the overflow fallback steal a foreign
			// writer. Retry allocation after its existing owner releases it.
			runtime.Gosched()
			return nil, nil, nil
		}
		candidate = source
	}
	release, _ := reserveInsertRows(candidate, rows, Settings.ShardSize, true, -1)
	return candidate, release, topology
}

// O is still local ownership: upgrading it invalidates readers, rather than
// transferring a foreign writer's shard. IS retains its explicit backend proof.
func (t *table) localInsertShard(shard *storageShard) bool {
	if !sharedClusterSource(t) {
		return true
	}
	cluster := activeNodeCluster()
	if cluster == nil {
		panic(mosi.NetworkError)
	}
	publication := cluster.manager.shardCell(shard).ownstate.Snapshot()
	phase := publication.Phase()
	if phase == mosi.IS {
		phase = publication.Backend()
	}
	return phase == mosi.M || phase == mosi.O
}

// Expand volatile defaults before partition routing and UNIQUE checks. The
// generated value becomes explicit invocation data, reused by WAL and forwarding.
func (t *table) prepareInsertDefaults(columns []string, values [][]scm.Scmer) ([]string, [][]scm.Scmer) {
	var missing []*column
	for _, c := range t.columnDeclarations() {
		if c.DefaultExpression == "" {
			continue
		}
		found := false
		for _, name := range columns {
			if name == c.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, c)
		}
	}
	if len(missing) == 0 {
		return columns, values
	}
	cols := append([]string(nil), columns...)
	for _, c := range missing {
		cols = append(cols, c.Name)
	}
	rows := make([][]scm.Scmer, len(values))
	for i, row := range values {
		rows[i] = make([]scm.Scmer, len(cols))
		copy(rows[i], row)
		for j, c := range missing {
			rows[i][len(columns)+j] = c.defaultValue()
		}
	}
	return cols, rows
}

// Reserve before shard ownership when clustering or hash routing requires the
// generated values. The shard sink then receives explicit IDs.
func (t *table) prepareInsertIDs(columns []string, values [][]scm.Scmer, notify func(int64), dimensions []shardDimension) ([]string, [][]scm.Scmer, bool) {
	needed := sharedClusterSource(t)
	for _, d := range dimensions {
		needed = needed || d.Method != ""
	}
	if !needed {
		return columns, values, false
	}
	for _, c := range t.columnDeclarations() {
		if !c.AutoIncrement {
			continue
		}
		position := -1
		for i, name := range columns {
			if name == c.Name {
				position = i
				break
			}
		}
		generated := uint64(0)
		explicit := uint64(0)
		for _, row := range values {
			if position < 0 || position >= len(row) || row[position].IsNil() {
				generated++
				continue
			}
			if id := scm.ToInt(row[position]); id > 0 {
				explicit = max(explicit, uint64(id))
			}
		}
		base := t.reserveAutoIncrement(explicit, generated)
		if generated > 0 && notify != nil {
			notify(int64(base + 1))
		}
		cols := append([]string(nil), columns...)
		if position < 0 {
			position = len(cols)
			cols = append(cols, c.Name)
		}
		rows := make([][]scm.Scmer, len(values))
		for i, row := range values {
			rows[i] = make([]scm.Scmer, len(cols))
			copy(rows[i], row)
			if rows[i][position].IsNil() {
				base++
				rows[i][position] = scm.NewInt(int64(base))
			}
		}
		return cols, rows, true
	}
	return columns, values, false
}
