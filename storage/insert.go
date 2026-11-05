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
import "context"
import "runtime"
import "strings"
import "math/bits"
import "sync/atomic"
import "encoding/binary"
import "github.com/cespare/xxhash/v2"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"
import "github.com/launix-de/go-mysqlstack/sqldb"

// One node-wide pool bounds synchronization metadata independently of table and
// row counts. GOMAXPROCS uses the runtime's container-aware CPU budget. Never
// resize a live pool: existing reservations must keep the same stripe mapping.
type UniqueLockPool struct {
	stripes []uniqueWriteStripe
}

var uniquePoolStartupFactor atomic.Int64

func init() { uniquePoolStartupFactor.Store(4) }

var nodeUniqueWriteLocks = sync.OnceValue(func() *UniqueLockPool {
	return &UniqueLockPool{stripes: make([]uniqueWriteStripe, uniqueWriteStripeCount(int(uniquePoolStartupFactor.Load())))}
})

// Padding separates unrelated contended mutexes onto separate cache lines.
// This storage is node-global, never per table, constraint or record.
type uniqueWriteStripe struct {
	sync.Mutex
	_ [64 - unsafe.Sizeof(sync.Mutex{})]byte
}

func uniqueWriteStripeCount(factor int) int {
	if factor == 0 {
		return 1
	}
	return factor * runtime.GOMAXPROCS(0)
}

// Keep ordinary CPU budgets allocation-free, while supporting pools larger
// than 128 stripes. This mask belongs to an invocation, never to a table.
type uniqueWriteMask struct {
	words [2]uint64
	extra []uint64
}

func (m uniqueWriteMask) word(i int) uint64 {
	if i < len(m.words) {
		return m.words[i]
	}
	if i -= len(m.words); i < len(m.extra) {
		return m.extra[i]
	}
	return 0
}

func (m *uniqueWriteMask) addWord(i int, bits uint64) {
	if bits == 0 {
		return
	}
	if i < len(m.words) {
		m.words[i] |= bits
		return
	}
	i -= len(m.words)
	if i >= len(m.extra) {
		extra := make([]uint64, (len(nodeUniqueWriteLocks().stripes)+63)/64-len(m.words))
		copy(extra, m.extra)
		m.extra = extra
	}
	m.extra[i] |= bits
}

func (m *uniqueWriteMask) add(i int) { m.addWord(i/64, uint64(1)<<uint(i%64)) }

func (m uniqueWriteMask) contains(other uniqueWriteMask) bool {
	for i := 0; i < len(other.words)+len(other.extra); i++ {
		if other.word(i)&^m.word(i) != 0 {
			return false
		}
	}
	return true
}

func (m uniqueWriteMask) empty() bool { return (uniqueWriteMask{}).contains(m) }

func (m uniqueWriteMask) intersects(other uniqueWriteMask) bool {
	for i := 0; i < len(m.words)+len(m.extra); i++ {
		if m.word(i)&other.word(i) != 0 {
			return true
		}
	}
	return false
}

func (m *uniqueWriteMask) include(other uniqueWriteMask) {
	for i := 0; i < len(other.words)+len(other.extra); i++ {
		m.addWord(i, other.word(i))
	}
}

type uniqueWriteGuard struct {
	afterTriggers []uniqueAfterTrigger
	ownedShard    *storageShard // an UPDATE already owns this native MOSI writer
	reservations  *uniqueReservationSet
	mutation      *mosi.Writer
	locks         *UniqueLockPool
	mask          uniqueWriteMask
	held          bool
	wrote         bool
	version       uint64 // advances when a collision callback temporarily releases guards
}

// AFTER triggers run after the mutation's key reservations are released. They
// may issue ordinary queries against the same keys; retaining a key stripe
// across that query would deadlock. The event owns row values, never recids.
type uniqueAfterTrigger struct {
	table    *table
	timing   TriggerTiming
	old, new dataset
	tx       *TxContext
}

func (g *uniqueWriteGuard) after(event uniqueAfterTrigger) {
	if g == nil {
		event.table.ExecuteTriggers(event.timing, event.old, event.new, event.tx)
		return
	}
	g.afterTriggers = append(g.afterTriggers, event)
}

func (g *uniqueWriteGuard) release() {
	g.unlock()
	if g.mutation != nil {
		g.mutation.Release()
		g.mutation = nil
	}
	events := g.afterTriggers
	g.afterTriggers = nil
	for _, event := range events {
		event.table.ExecuteTriggers(event.timing, event.old, event.new, event.tx)
	}
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
			lease := s.getRead(tx, shardReadOptions{mutation: true})
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
	for word := 0; word < len(g.mask.words)+len(g.mask.extra); word++ {
		for mask := g.mask.word(word); mask != 0; mask &= mask - 1 {
			g.locks.stripes[word*64+bits.TrailingZeros64(mask)].Lock()
		}
	}
	g.held = true
}

// Try additional stripes in ascending order. On conflict, release only the
// newly acquired prefix; existing ownership remains intact for error cleanup.
func (g *uniqueWriteGuard) tryAdditional(needed uniqueWriteMask) bool {
	for word := 0; word < len(needed.words)+len(needed.extra); word++ {
		for mask := needed.word(word) &^ g.mask.word(word); mask != 0; mask &= mask - 1 {
			i := word*64 + bits.TrailingZeros64(mask)
			if !g.locks.stripes[i].TryLock() {
				for previous := 0; previous <= word; previous++ {
					for acquired := needed.word(previous) &^ g.mask.word(previous); acquired != 0; acquired &= acquired - 1 {
						j := previous*64 + bits.TrailingZeros64(acquired)
						if j >= i {
							break
						}
						g.locks.stripes[j].Unlock()
					}
				}
				return false
			}
		}
	}
	return true
}

func (g *uniqueWriteGuard) tryLock() bool {
	needed := g.mask
	g.mask = uniqueWriteMask{}
	ok := g.tryAdditional(needed)
	g.mask = needed
	g.held = ok
	return ok
}

func (g *uniqueWriteGuard) unlock() {
	if !g.held {
		return
	}
	for word := len(g.mask.words) + len(g.mask.extra) - 1; word >= 0; word-- {
		for mask := g.mask.word(word); mask != 0; {
			i := bits.Len64(mask) - 1
			g.locks.stripes[word*64+i].Unlock()
			mask &^= uint64(1) << i
		}
	}
	g.held = false
	g.version++
}

// A collision update may change UNIQUE columns. Borrow the existing key
// ownership and acquire any additional stripes without waiting: two callbacks
// changing keys in opposite directions must fail explicitly, never deadlock.
func (g *uniqueWriteGuard) extend(mask uniqueWriteMask) {
	if !g.held {
		g.mask = mask
		if !g.tryLock() {
			panic(sqldb.NewSQLError1(1213, "40001", "Concurrent UNIQUE key change; retry transaction"))
		}
		return
	}
	if !g.tryAdditional(mask) {
		panic(sqldb.NewSQLError1(1213, "40001", "Concurrent UNIQUE key change; retry transaction"))
	}
	g.mask.include(mask)
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

// UNIQUE comparison and hashing share the declared type/collation contract,
// independently of plain, compressed and UUID string representations.
func (t *table) uniqueColumnDeclaration(columnName string) *column {
	for _, candidate := range t.columnDeclarations() {
		if candidate.Name == columnName {
			return candidate
		}
	}
	return nil
}

func (t *table) uniqueColumnEqual(columnName string) func(scm.Scmer, scm.Scmer) bool {
	declaration := t.uniqueColumnDeclaration(columnName)
	return func(a, b scm.Scmer) bool { return uniqueColumnValuesEqual(declaration, a, b) }
}

func uniqueColumnValuesEqual(declaration *column, a, b scm.Scmer) bool {
	typed := declaration != nil && declaration.Typ != "" && !strings.EqualFold(declaration.Typ, "any")
	if typed && (a.IsNil() || b.IsNil()) {
		return a.IsNil() && b.IsNil()
	}
	if typed {
		a, b = uniqueColumnValue(declaration, a), uniqueColumnValue(declaration, b)
	}
	if (a.IsString() || a.IsSymbol() || a.IsBufferString()) && (b.IsString() || b.IsSymbol() || b.IsBufferString()) {
		collation := "bin"
		if declaration != nil {
			collation = declaration.Collation
		}
		return scm.EqualCollated(a, b, collation).Bool()
	}
	return scm.Equal(a, b)
}

func (t *table) uniqueColumnHash(column string, value scm.Scmer) (uint64, bool) {
	for _, declaration := range t.columnDeclarations() {
		if declaration.Name != column {
			continue
		}
		if declaration.Typ == "" || strings.EqualFold(declaration.Typ, "any") {
			// Fold text within the declared equality classes before the native
			// coercion hash. Numeric prefixes still meet their numeric values;
			// nil, booleans and opaque bridges retain the all-stripes fallback.
			if strings.Contains(declaration.Collation, "_ci") && (value.IsString() || value.IsSymbol() || value.IsBufferString()) {
				value = scm.NewString(scm.CaseFoldKey(scm.String(value)))
			}
			return uniqueGuardHash(value)
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
			order := scm.Collate(scm.NewString(orderName), scm.NewBool(false))
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
	locks := nodeUniqueWriteLocks()
	var identity [8]byte
	binary.LittleEndian.PutUint64(identity[:], uint64(uintptr(unsafe.Pointer(t))))
	seed := xxhash.Sum64(identity[:])
	var mask uniqueWriteMask
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
			hash := (seed ^ uint64(idx+1)) * 1099511628211
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
				for stripe := range locks.stripes {
					mask.add(stripe)
				}
				return uniqueWriteGuard{locks: locks, mask: mask}
			}
			mask.add(int(hash % uint64(len(locks.stripes))))
		}
	}
	return uniqueWriteGuard{locks: locks, mask: mask}
}

// guardUniqueRow gives mutation mappers the same logical-key ownership as
// collision callbacks. It is invocation-owned, never stored per record. Resolve
// stale physical IDs and revalidate keys after waiting without holding shard.mu.
// Native mutation admission protects the read/modify/write lifetime. UNIQUE
// stripes protect key membership only; unchanged keys do not need them.
func (s *storageShard) localMutationState() *mosi.State {
	if current := s.localMutations.Load(); current != nil {
		return current
	}
	state := new(mosi.State)
	state.Initialize(mosi.M)
	if s.localMutations.CompareAndSwap(nil, state) {
		return state
	}
	return s.localMutations.Load()
}

func (s *storageShard) mutationGuard(tx *TxContext) *uniqueWriteGuard {
	guard := &uniqueWriteGuard{locks: nodeUniqueWriteLocks(), ownedShard: s}
	if activeReplication(s.t.schema) != nil {
		return guard
	}
	state := s.localMutationState()
	var owner any
	if tx != nil {
		owner = tx.transactionState
	}
	for {
		writer, done := state.TryWrite(owner, false)
		if writer != nil {
			guard.mutation = writer
			return guard
		}
		select {
		case <-done:
		case <-scanQueryContext(tx, context.Background()).Done():
			panic(scanQueryContext(tx, context.Background()).Err())
		}
	}
}

func (s *storageShard) guardUniqueRow(id uint32, tx *TxContext) (*uniqueWriteGuard, uint32, bool) {
	if activeReplication(s.t.schema) == nil && len(s.t.Triggers) != 0 && len(s.t.GetTriggers(BeforeUpdate)) != 0 {
		guard := s.mutationGuard(tx)
		s.mu.RLock()
		defer s.mu.RUnlock()
		if (tx == nil || tx.Mode != TxACID) && s.deletions.Get(uint(id)) {
			current, found := s.resolveVisiblePrimaryRecidLocked(id)
			if !found {
				guard.release()
				return nil, id, false
			}
			id = current
		}
		return guard, id, true
	}
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
		if guard.mask.contains(current.mask) {
			return &guard, id, true
		}
		guard.unlock()
	}
}

// A source's native writer stays reserved while BEFORE runs. Release key
// stripes on every retained peer before executing arbitrary trigger code.
func (g *uniqueWriteGuard) releaseKeys() {
	g.unlock()
	g.mask = uniqueWriteMask{}
	if g.reservations != nil {
		g.reservations.releaseKeys()
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
