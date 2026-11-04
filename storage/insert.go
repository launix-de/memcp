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
import "slices"
import "unsafe"
import "runtime"
import "strings"
import "math/bits"
import "encoding/binary"
import "github.com/cespare/xxhash/v2"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"
import "github.com/launix-de/go-mysqlstack/sqldb"

// UNIQUE guards belong to the logical table, independently of physical shard
// generations. Equal keys use the same stripe across rebuild/repartition; an
// unrelated key does not hold a table-wide mutex during lookup, WAL or fsync.
// Large batches acquire their stripe set in ascending order, once per batch.
type uniqueWriteLocks [64]sync.Mutex

type uniqueWriteGuard struct {
	ownedShard   *storageShard // an UPDATE already owns this native MOSI writer
	reservations *uniqueReservationSet
	locks        *uniqueWriteLocks
	mask         uint64
	held         bool
	wrote        bool
	version      uint64 // advances when a collision callback temporarily releases guards
}

type uniqueReadRetry struct{}

func (g *uniqueWriteGuard) validReads() bool {
	return g.reservations == nil || g.reservations.valid()
}

// Forward UNIQUE existence checks before acquiring local shard payload. Warm
// readable local shards keep the direct path; other owners run the same native
// reservation/visibility kernel through the ordinary scan protocol.
func (s *storageShard) probeUnique(access scanAccess, tx *TxContext) bool {
	cluster := activeNodeCluster()
	resource := mosiResource{Database: s.t.schema.Name, Table: s.t.Name, Shard: s.uuid.String()}
	local := false
	if cell := s.coherence.Load(); cell != nil {
		state := cell.resolved().ownstate.Snapshot()
		local = state.Phase().Readable() && s.state() != COLD || state.Flight() != nil
	}
	owner := cluster.self.ID
	if !local {
		var err error
		owner, err = cluster.locate(scanQueryContext(tx, cluster.ctx), resource)
		if err != nil {
			panic(err)
		}
	}
	if local || owner == cluster.self.ID {
		delegated := false
		found := false
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					if _, capacity := failure.(shardReadDelegation); capacity {
						delegated = true
					} else {
						panic(failure)
					}
				}
			}()
			lease := s.getRead(tx, shardReadOptions{})
			defer lease.Release()
			_, found = lease.shard.GetRecordidForUnique(access, tx, true)
		}()
		if !delegated {
			return found
		}
	}
	var preferred *clusterLink
	for _, link := range cluster.view.Load().links {
		if link.member.ID == owner {
			preferred = link
			break
		}
	}
	executor := clusterScanExecutor{cluster: cluster, preferred: preferred, expected: remoteScanShard{Schema: s.t.filterSchemaFingerprint(), Resource: resource}}
	result := executor.ExecuteShardScan(resource, shardScanInvocation{operator: shardUniqueProbe, tx: tx, access: access})
	if result.err.r != nil {
		panic(result.err)
	}
	return result.res.Bool()
}

func (g *uniqueWriteGuard) lock() {
	for mask := g.mask; mask != 0; mask &= mask - 1 {
		g.locks[bits.TrailingZeros64(mask)].Lock()
	}
	g.held = true
}

func (g *uniqueWriteGuard) tryLock() bool {
	var acquired uint64
	for remaining := g.mask; remaining != 0; remaining &= remaining - 1 {
		i := bits.TrailingZeros64(remaining)
		if !g.locks[i].TryLock() {
			for acquired != 0 {
				j := bits.TrailingZeros64(acquired)
				g.locks[j].Unlock()
				acquired &= acquired - 1
			}
			return false
		}
		acquired |= uint64(1) << i
	}
	g.held = true
	return true
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

// A collision update may change UNIQUE columns. Borrow the existing key
// ownership and acquire any additional stripes without waiting: two callbacks
// changing keys in opposite directions must fail explicitly, never deadlock.
func (g *uniqueWriteGuard) extend(mask uint64) {
	var acquired uint64
	for remaining := mask &^ g.mask; remaining != 0; remaining &= remaining - 1 {
		bit := uint64(1) << bits.TrailingZeros64(remaining)
		if !g.locks[bits.TrailingZeros64(bit)].TryLock() {
			for acquired != 0 {
				i := bits.TrailingZeros64(acquired)
				g.locks[i].Unlock()
				acquired &^= uint64(1) << i
			}
			panic(sqldb.NewSQLError1(1213, "40001", "Concurrent UNIQUE key change; retry transaction"))
		}
		acquired |= bit
	}
	g.mask |= acquired
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
			return xxhash.Sum64String(text), true
		}
	default:
		return 0, false
	}
	number = math.Trunc(number)
	if number == 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	var encoded [8]byte
	binary.LittleEndian.PutUint64(encoded[:], math.Float64bits(number))
	return xxhash.Sum64(encoded[:]), true
}

func uniqueColumnValue(declaration *column, value scm.Scmer) scm.Scmer {
	if value.IsNil() {
		return value
	}
	if declaration.sanitizer != nil {
		return declaration.sanitizer(value)
	}
	switch strings.ToUpper(declaration.Typ) {
	case "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT", "VARCHAR", "CHAR":
		return scm.NewString(scm.String(value))
	}
	return value
}

// The column's index order owns text equivalence. Runtime representations
// (plain, compressed and UUID strings) must not change that equivalence.
func (t *table) uniqueColumnEqual(columnName string) func(scm.Scmer, scm.Scmer) bool {
	var declaration *column
	collation := "bin"
	for _, candidate := range t.columnDeclarations() {
		if candidate.Name == columnName {
			declaration = candidate
			collation = candidate.Collation
			break
		}
	}
	return func(a, b scm.Scmer) bool {
		typed := declaration != nil && declaration.Typ != "" && !strings.EqualFold(declaration.Typ, "any")
		if typed && (a.IsNil() || b.IsNil()) {
			return a.IsNil() && b.IsNil()
		}
		if typed {
			a, b = uniqueColumnValue(declaration, a), uniqueColumnValue(declaration, b)
		}
		if (a.IsString() || a.IsSymbol() || a.IsBufferString()) && (b.IsString() || b.IsSymbol() || b.IsBufferString()) {
			return scm.EqualCollated(a, b, collation).Bool()
		}
		return scm.Equal(a, b)
	}
}

func (t *table) uniqueColumnHash(column string, value scm.Scmer) (uint64, bool) {
	for _, declaration := range t.columnDeclarations() {
		if declaration.Name != column {
			continue
		}
		if declaration.Typ == "" || strings.EqualFold(declaration.Typ, "any") {
			if declaration.Collation == "" || declaration.Collation == "bin" || declaration.Collation == "binary" || declaration.Collation == "utf8" || declaration.Collation == "utf8mb4" || strings.HasSuffix(declaration.Collation, "_bin") {
				return uniqueGuardHash(value)
			}
			return 0, false // coercing ANY plus collation has no narrow general key
		}
		value = uniqueColumnValue(declaration, value)
		if value.IsNil() {
			return xxhash.Sum64String("NULL"), true
		}
		if declaration.sanitizer != nil && (value.IsInt() || value.IsDate() || value.IsFloat()) {
			var encoded [8]byte
			bits := uint64(value.Int())
			if value.IsFloat() {
				number := value.Float()
				if number == 0 {
					number = 0 // normalize negative zero for floating equality
				}
				bits = math.Float64bits(number)
			}
			binary.LittleEndian.PutUint64(encoded[:], bits)
			return xxhash.Sum64(encoded[:]), true
		}
		if !(value.IsString() || value.IsSymbol() || value.IsBufferString()) {
			return uniqueGuardHash(value)
		}
		collation := declaration.Collation
		if collation == "" || collation == "bin" || collation == "binary" || collation == "utf8" || collation == "utf8mb4" || strings.HasSuffix(collation, "_bin") {
			// Untyped/native text can still compare with numbers through Equal.
			// Preserve the conservative cross-type guard for those declarations.
			if declaration.Typ == "" || strings.EqualFold(declaration.Typ, "any") {
				return uniqueGuardHash(value)
			}
			return xxhash.Sum64String(scm.String(value)), true
		}
		if strings.Contains(collation, "_ci") {
			return xxhash.Sum64String(scm.CaseFoldKey(scm.String(value))), true
		}
		return xxhash.Sum64String(scm.String(value)), true
	}
	return uniqueGuardHash(value)
}

// UNIQUE probes use equality-compatible index ordering. Natural/language sort
// order can merge different numeric spellings or split Unicode fold classes;
// its sort keys must not decide equality membership or the guard hash.
func (t *table) uniqueAccess(access scanAccess) scanAccess {
	var schema []scm.Scmer
	for i := 0; i < access.len(); i++ {
		for _, declaration := range t.columnDeclarations() {
			if declaration.Name != access.boundaryColumn(i) {
				continue
			}
			switch strings.ToUpper(declaration.Typ) {
			case "TEXT", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT", "VARCHAR", "CHAR", "ENUM":
			default:
				continue
			}
			if schema == nil {
				schema = append([]scm.Scmer(nil), access.schema...)
			}
			box := *(*scanBoundaryBox)(schema[scanAccessSchemaHeaderSize+i].Custom(TagScanBoundary))
			orderName := "unicode-string"
			if strings.Contains(declaration.Collation, "_ci") {
				orderName = "unicode-simple-fold"
			}
			order := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(orderName), scm.NewBool(false))
			box.order, box.orderMetadata = order.Func(), orderName+":asc"
			schema[scanAccessSchemaHeaderSize+i] = NewScanBoundaryScmer(&box)
			break
		}
	}
	if schema != nil {
		access.schema = schema
		access.firstBoundary = (*scanBoundaryBox)(schema[scanAccessSchemaHeaderSize].Custom(TagScanBoundary))
	}
	return access
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
			for columnIndex, position := range positions {
				value := scm.NewNil()
				if position >= 0 && position < len(row) {
					value = row[position]
				}
				if value.IsNil() && !mergeNull && !key.NullsNotDistinct {
					skip = true
					break
				}
				part, precise := t.uniqueColumnHash(key.Cols[columnIndex], value)
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

// guardUniqueRow gives mutation mappers the same logical-key ownership as
// collision callbacks. It is invocation-owned, never stored per record. Resolve
// stale physical IDs and revalidate keys after waiting without holding shard.mu.
func (s *storageShard) guardUniqueRow(id uint32, tx *TxContext) (*uniqueWriteGuard, uint32, bool) {
	if len(s.t.Unique) == 0 || activeReplication(s.t.schema) != nil {
		return nil, id, true
	}
	var columns []string
	for _, constraint := range s.t.Unique {
		for _, column := range constraint.Cols {
			if !slices.Contains(columns, column) {
				columns = append(columns, column)
			}
		}
	}
	for _, column := range columns {
		s.getColumnStorageOrPanic(column, false, tx)
	}
	values := make([]scm.Scmer, len(columns))
	read := func() bool {
		if (tx == nil || tx.Mode != TxACID) && s.deletions.Get(uint(id)) {
			var found bool
			id, found = s.resolveVisiblePrimaryRecidLocked(id)
			if !found {
				return false
			}
		}
		for i, column := range columns {
			values[i] = s.rowValueByRecidLocked(id, column)
		}
		return true
	}
	for {
		s.mu.RLock()
		found := read()
		s.mu.RUnlock()
		if !found {
			return nil, id, false
		}
		guard := s.t.guardUniqueWrites(columns, [][]scm.Scmer{values}, false, 0)
		guard.lock()
		s.mu.RLock()
		found = read()
		current := s.t.guardUniqueWrites(columns, [][]scm.Scmer{values}, false, 0)
		s.mu.RUnlock()
		if !found {
			guard.unlock()
			return nil, id, false
		}
		if current.mask&^guard.mask == 0 {
			return &guard, id, true
		}
		guard.unlock()
	}
}

// Reservations include queued/in-flight batches until they finish, not just
// already appended rows. They belong to a physical generation and never move
// to a successor: the pinned operation and existing forwarding own its writes.
func reserveInsertRows(shard *storageShard, rows int64, size uint, overflow bool, expected int64) (func(), bool) {
	// Routing reserves a physical writer before UNIQUE preflight acquires its
	// write permission. Retain that generation's payload through the whole
	// reservation, using the same native lease as other prepared operators.
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
			// Publish the routing reservation before waiting for the payload
			// lease: other INSERTs can choose another lane while this writer
			// waits for the current shard operation to release its row lock.
			shard.retainCacheRead()
			return func() {
				shard.pendingInsertRows.Add(-rows)
				shard.releaseCacheRead()
			}, true
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
