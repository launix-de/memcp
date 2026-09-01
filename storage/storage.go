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
import "io"
import "math/rand"
import "sort"
import "unsafe"
import "github.com/carli2/hybridsort"
import "sync"
import "sync/atomic"
import "time"
import "strconv"
import "reflect"
import "strings"
import "unicode/utf8"
import units "github.com/docker/go-units"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// ColumnReader provides sequential-access-optimized reads. Returned by
// ColumnStorage.GetCachedReader(). Must not be shared between goroutines.
//
// GetValueMulti and GetValueRange are bulk counterparts to GetValue that
// amortize per-call overhead (interface dispatch, binary search, shared
// decode state) across many rows in one call:
//
//   - GetValueMulti(recids, target, stride) gathers values at arbitrary,
//     caller-supplied row ids (e.g. an index-probe batch or a sort
//     permutation). Implementations should detect runs of ascending recids
//     and take a sequential fast path where their underlying encoding
//     benefits from it (rANS chunk decode, run-length sequences, sparse
//     merge), falling back to per-element lookup only for genuine jumps.
//   - GetValueRange(recid, count, target, stride) reads count consecutive
//     rows starting at recid. Implementations should use this to skip
//     repeated binary search / bit-position division entirely.
//
// Both write results into target starting at target[0], target[stride],
// target[2*stride], ... so a caller building an interleaved multi-column
// row buffer can pass stride=rowWidth and a per-column offset slice
// (target[colOffset:]). stride<=0 is treated as 1.
type ColumnReader interface {
	GetValue(uint32) scm.Scmer
	GetValueMulti(recids []uint32, target []scm.Scmer, stride int)
	GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int)
}

type ColumnReaderFunc func(uint32) scm.Scmer

func (f ColumnReaderFunc) GetValue(idx uint32) scm.Scmer {
	return f(idx)
}

func (f ColumnReaderFunc) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for _, recid := range recids {
		target[idx] = f(recid)
		idx += stride
	}
}

func (f ColumnReaderFunc) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = f(recid + k)
		idx += stride
	}
}

// TxColumnReaderProvider optionally exposes a transaction-bound reader.
// Storages that do not depend on tx/session context can ignore it and rely on
// the legacy GetCachedReader path.
type TxColumnReaderProvider interface {
	GetCachedReaderTx(*TxContext) ColumnReader
}

func scmerToTxContext(v scm.Scmer) *TxContext {
	if v.IsNil() {
		return nil
	}
	tx, _ := v.Any().(*TxContext)
	return tx
}

type scanArgLayout struct {
	tx            *TxContext
	tableIdx      int
	filterColsIdx int
	filterFnIdx   int
	mapColsIdx    int
	mapFnIdx      int
	reduceIdx     int
	neutralIdx    int
	reduce2Idx    int
	outerIdx      int
	sortColsIdx   int
	sortDirsIdx   int
	partColsIdx   int
	offsetIdx     int
	limitIdx      int
	strideIdx     int
	batchDataIdx  int
}

func scanLayout(a []scm.Scmer) scanArgLayout {
	return scanArgLayout{
		tx:            scmerToTxContext(a[0]),
		tableIdx:      1,
		filterColsIdx: 2,
		filterFnIdx:   3,
		mapColsIdx:    4,
		mapFnIdx:      5,
		reduceIdx:     6,
		neutralIdx:    7,
		reduce2Idx:    8,
		outerIdx:      9,
		sortColsIdx:   4,
		sortDirsIdx:   5,
		partColsIdx:   6,
		offsetIdx:     7,
		limitIdx:      8,
		strideIdx:     6,
		batchDataIdx:  7,
	}
}

func showColumnsForRows(rows []scm.Scmer) scm.Scmer {
	if len(rows) == 0 {
		return scm.NewSlice([]scm.Scmer{})
	}
	firstRow, ok := scmerSlice(rows[0])
	if !ok {
		return scm.NewSlice([]scm.Scmer{})
	}
	columns := make([]scm.Scmer, 0, len(firstRow)/2)
	for i := 0; i+1 < len(firstRow); i += 2 {
		columns = append(columns, scm.NewSlice([]scm.Scmer{
			scm.NewString("Field"), scm.NewString(scm.String(firstRow[i])),
			scm.NewString("Type"), scm.NewString("any"),
			scm.NewString("Collation"), scm.NewString(""),
			scm.NewString("RawType"), scm.NewString("any"),
			scm.NewString("Dimensions"), scm.NewSlice([]scm.Scmer{}),
			scm.NewString("Null"), scm.NewBool(true),
			scm.NewString("Key"), scm.NewString(""),
			scm.NewString("Default"), scm.NewNil(),
			scm.NewString("Extra"), scm.NewString(""),
		}))
	}
	return scm.NewSlice(columns)
}

func normalizePartitionDataset(arg scm.Scmer) dataset {
	raw := mustScmerSlice(arg, "partition columns")
	if len(raw) == 0 {
		return dataset(raw)
	}
	flat := true
	for _, item := range raw {
		pair, ok := scmerSlice(item)
		if !ok {
			continue
		}
		if len(pair) == 2 && (pair[0].IsString() || pair[0].GetTag() == scm.TagSymbol) {
			flat = false
			break
		}
	}
	if flat {
		return dataset(raw)
	}
	normalized := make(dataset, 0, len(raw)*2)
	for _, item := range raw {
		pair := mustScmerSlice(item, "partition column pair")
		if len(pair) != 2 {
			panic(fmt.Sprintf("invalid partition column pair: expected (column value), got %s", describeScmerValue(item)))
		}
		normalized = append(normalized, pair[0], pair[1])
	}
	return normalized
}

// THE basic storage pattern
type ColumnStorage interface {
	// info
	GetValue(uint32) scm.Scmer // read function (concurrent-safe, no mutable state)
	// GetValueMulti and GetValueRange are the bulk counterparts of GetValue;
	// see the ColumnReader doc comment for the exact contract. Every storage
	// format must implement both with a real bulk strategy for its own
	// encoding (sequential cursor, cached decode, merge-scan, ...) rather
	// than a naive per-element loop through GetValue, wherever that
	// encoding has state worth amortizing across a batch.
	GetValueMulti(recids []uint32, target []scm.Scmer, stride int)
	GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int)
	GetCachedReader() ColumnReader // returns a per-goroutine cached reader for O(1) sequential access
	String() string                // self-description
	scm.Sizable

	// buildup functions 1) prepare 2) scan, 3) proposeCompression(), if != nil repeat at 1, 4) init, 5) build; all values are passed through twice
	// analyze
	prepare()
	scan(uint32, scm.Scmer)
	proposeCompression(i uint32) ColumnStorage
	// store
	init(uint32)
	build(uint32, scm.Scmer)
	finish()

	// statistics — collected at rebuild time, cheap O(1) access for query planning
	DistinctCount() uint // estimated number of distinct values in this shard column

	// JIT compilation
	JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc

	// persistency (the callee takes ownership of the file handle, so he can close it immediately or set a finalizer)
	Serialize(io.Writer)        // write content to Writer
	Deserialize(io.Reader) uint // read from Reader (note that first byte is already read, so the reader starts at the second byte)
}

// storages maps the on-disk magic byte to the Go type used for deserialization.
//
// ⚠️  STORAGE FORMAT VERSIONING — READ BEFORE CHANGING ANY Serialize/Deserialize METHOD ⚠️
//
// Two-level versioning scheme:
//
//	Level 1 — magic byte (this map):
//	  Identifies the storage TYPE.  Every persisted column file starts with one
//	  magic byte; the runtime dispatches to the matching type via this map.
//	  The magic byte must NEVER change for an existing type.
//
//	Level 2 — per-type version byte (inside each Serialize/Deserialize pair):
//	  Identifies the LAYOUT VERSION within a type.  Each type reads a version
//	  byte immediately after the magic byte (or reuses an existing padding byte)
//	  and dispatches via a switch to the appropriate deserializeXxxV* helper.
//
// Per-type versioning rules (enforced in each Serialize/Deserialize pair):
//  1. Serialize always writes the CURRENT version constant as the first byte.
//  2. Deserialize reads the version byte first, then switches on it.
//  3. NEVER delete an old deserializeXxxV* helper — old on-disk data must stay
//     readable forever.
//  4. When changing binary layout: increment the version constant and add a new
//     deserializeXxxV* method.  Leave the old one untouched.
//  5. Data written before this versioning scheme was introduced is "version 0"
//     (or a named legacy constant).  Each type documents what version 0 means.
//
// Exception — magic bytes 1, 2, 13, 40 (StorageSCMER, StorageSparse, StorageDecimal, StorageEnum):
//
//	These types existed before the versioning scheme and had NO padding byte in
//	their original layout, so there is no safe location for an inline version
//	byte without corrupting existing data.  They read their first field directly
//	with NO version byte.  If any of their formats must change, register a NEW
//	magic byte for the new layout and keep the old magic as a read-only legacy
//	reader forever.
//
// Current magic byte assignments:
//
//	 1  StorageSCMER   – generic Scmer values        (no version byte — see above)
//	 2  StorageSparse  – sparse/NULL-only column     (no version byte — see above)
//	10  StorageInt     – bit-packed integer
//	11  StorageSeq     – sequential/auto-increment integer
//	12  StorageFloat   – 64-bit float
//	13  StorageDecimal – fixed-precision decimal      (no version byte — see above)
//	20  StorageString  – dictionary-compressed or buffer string
//	21  StoragePrefix  – prefix-compressed string (experimental)
//	31  OverlayBlob    – large binary/blob overlay
//	40  StorageEnum    – rANS-entropy-coded enum         (no version byte — see above)
//	41  StorageConst   – single constant value column
//	50  StorageComputeProxy – computed/cached column
var storages = map[uint8]reflect.Type{
	1:  reflect.TypeOf(StorageSCMER{}),
	2:  reflect.TypeOf(StorageSparse{}),
	10: reflect.TypeOf(StorageInt{}),
	11: reflect.TypeOf(StorageSeq{}),
	12: reflect.TypeOf(StorageFloat{}),
	13: reflect.TypeOf(StorageDecimal{}),
	20: reflect.TypeOf(StorageString{}),
	21: reflect.TypeOf(StoragePrefix{}),
	//30: reflect.TypeOf(OverlaySCMER{}),
	31: reflect.TypeOf(OverlayBlob{}),
	40: reflect.TypeOf(StorageEnum{}),
	41: reflect.TypeOf(StorageConst{}),
	50: reflect.TypeOf(StorageComputeProxy{}),
}

func scmerSlice(v scm.Scmer) ([]scm.Scmer, bool) {
	v = v.WithoutSourceInfo()
	if v.IsSlice() {
		return v.Slice(), true
	}
	return nil, false
}

// describeScmerValue renders v for use in a panic message. Long values
// (entire codegen'd expressions) are truncated at a UTF-8 rune boundary
// so the panic stays readable and never leaves a half-encoded code point.
//
// Uses AppendString with a heap-backed 256-byte scratch buffer so primitive
// values (string/symbol/int/float/bool/nil) render without an extra heap
// allocation. Larger values (slices, dicts) still allocate inside
// AppendString — panics are rare, so we accept that cost.
func describeScmerValue(v scm.Scmer) string {
	const maxBytes = 200
	if v.IsNil() {
		return "nil"
	}
	// make'd slice is heap-allocated so the unsafe.String view returned by
	// AppendString for tagInt / tagFloat stays live as long as the result.
	buf := make([]byte, 0, 256)
	s, _ := v.AppendString(buf)
	if len(s) <= maxBytes {
		return s
	}
	// Back off from maxBytes to the previous rune boundary; max UTF-8 rune
	// is 4 bytes so this costs at most 3 iterations.
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

func mustScmerSlice(v scm.Scmer, ctx string) []scm.Scmer {
	if slice, ok := scmerSlice(v); ok {
		return slice
	}
	panic(fmt.Sprintf("%s: expected list, got %s", ctx, describeScmerValue(v)))
}

func scmerSliceToStrings(list []scm.Scmer) []string {
	out := make([]string, len(list))
	for i, item := range list {
		out[i] = scm.String(item)
	}
	return out
}

// decodePerTableInts accepts nil (= all -1), or a list of length n of ints.
// Sentinel -1 disables per-table offset/limit for that entry.
func decodePerTableInts(v scm.Scmer, n int, ctx string) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = -1
	}
	if v.IsNil() {
		return out
	}
	slice, ok := scmerSlice(v)
	if !ok {
		panic(fmt.Sprintf("%s: expected list or nil, got %s", ctx, describeScmerValue(v)))
	}
	if len(slice) != n {
		panic(fmt.Sprintf("%s: expected length %d, got %d", ctx, n, len(slice)))
	}
	for i, entry := range slice {
		if entry.IsNil() {
			out[i] = -1
		} else {
			out[i] = int(scm.ToInt(entry))
		}
	}
	return out
}

func parseBatchPseudoColName(name string) (int, bool) {
	if len(name) < 2 || name[0] != '#' {
		return 0, false
	}
	n, err := strconv.Atoi(name[1:])
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func isScanPseudoColName(name string) bool {
	if name == "$recset_contains" {
		return true
	}
	_, isBatch := parseBatchPseudoColName(name)
	return isBatch
}

// lockTablePublicationShards closes the publication race with shard users.
// User-visible locks drain incompatible work before returning. Cache
// initialization is different: its READ lock is immediately followed by a
// complete snapshot scan, which serializes any writer already past the atomic
// table-lock recheck. Taking a recursive shard RLock for that path can deadlock
// behind a queued writer when initialization runs inside a parallel reader.
func lockTablePublicationShards(shards []*storageShard, write bool, snapshotFollows bool) func() {
	if !write && snapshotFollows {
		return func() {}
	}
	if write {
		for _, shard := range shards {
			shard.mu.Lock()
		}
		return func() {
			for _, shard := range shards {
				shard.mu.Unlock()
			}
		}
	}
	for _, shard := range shards {
		shard.mu.RLock()
	}
	return func() {
		for _, shard := range shards {
			shard.mu.RUnlock()
		}
	}
}

// lockTable acquires a user-level read or write lock on the named table.
// The session's State is updated while waiting, and the unlock callback is
// registered with the session so that ReleaseAllLocks() can free it later.
// A run of FIFO-adjacent READ requests shares the lock. The first reader, or an
// exclusive writer, drains in-flight shard readers before publishing the lock.
func acquireTableLock(schema, name string, write bool, snapshotFollows bool, ss *scm.SessionState, querySeq uint64) func() {
	if ss == nil {
		panic("LOCK TABLES requires a query session")
	}
	db := GetDatabase(schema)
	if db == nil {
		panic("LOCK TABLES: unknown database: " + schema)
	}
	t := db.GetTable(name)
	if t == nil {
		panic("LOCK TABLES: unknown table: " + schema + "." + name)
	}
	cond := t.getTableLockCond()
	ctx := tableLockQueryContext(ss, querySeq)
	if ctx != nil {
		stopWake := context.AfterFunc(ctx, func() {
			t.tableLockMu.Lock()
			cond.Broadcast()
			t.tableLockMu.Unlock()
		})
		defer stopWake()
	}
	ss.BeginLockWait()
	defer ss.EndLockWait()
	t.tableLockMu.Lock()
	if t.tableLockOwner.Load() == ss {
		t.tableLockMu.Unlock()
		return func() {}
	}
	if t.tableLockReadOwners[ss] != 0 {
		t.tableLockMu.Unlock()
		if write {
			panic("LOCK TABLES: cannot upgrade an existing READ lock")
		}
		return func() {}
	}
	myTicket := t.tableLockNext
	t.tableLockNext++
	for myTicket != t.tableLockServe || t.tableLockState.Load() < 0 || (write && t.tableLockState.Load() != 0) {
		if ctx != nil && ctx.Err() != nil {
			t.tableLockMu.Unlock()
			panic("query killed")
		}
		cond.Wait()
	}
	if !write && t.tableLockState.Load() > 0 {
		if t.tableLockReadOwners == nil {
			t.tableLockReadOwners = make(map[*scm.SessionState]uint32)
		}
		t.tableLockReadOwners[ss]++
		t.tableLockState.Add(1)
		t.tableLockServe++
		cond.Broadcast()
		t.tableLockMu.Unlock()
		return func() {
			t.tableLockMu.Lock()
			if t.tableLockReadOwners[ss] == 1 {
				delete(t.tableLockReadOwners, ss)
			} else {
				t.tableLockReadOwners[ss]--
			}
			if t.tableLockState.Add(-1) < 0 {
				t.tableLockMu.Unlock()
				panic("table READ lock count underflow")
			}
			cond.Broadcast()
			t.tableLockMu.Unlock()
		}
	}
	t.tableLockMu.Unlock()
	// User locks drain incompatible shard users. Cache snapshot READ publication
	// instead relies on lockForMutation's post-shard-lock recheck; an already-
	// running writer is serialized by the initializer's immediately following scan.
	acquired := false
	defer func() {
		if acquired {
			return
		}
		t.tableLockMu.Lock()
		if t.tableLockServe == myTicket {
			t.tableLockServe++
		}
		cond.Broadcast()
		t.tableLockMu.Unlock()
	}()
	shards := t.ActiveShards()
	unlockShards := lockTablePublicationShards(shards, write, snapshotFollows)
	if write {
		t.tableLockOwner.Store(ss)
		t.tableLockState.Store(-1)
	} else {
		t.tableLockMu.Lock()
		if t.tableLockReadOwners == nil {
			t.tableLockReadOwners = make(map[*scm.SessionState]uint32)
		}
		t.tableLockReadOwners[ss]++
		t.tableLockState.Add(1)
		t.tableLockServe++
		cond.Broadcast()
		t.tableLockMu.Unlock()
	}
	unlockShards()
	acquired = true
	if write {
		return t.unlockTableWrite
	}
	return func() {
		t.tableLockMu.Lock()
		if t.tableLockReadOwners[ss] == 1 {
			delete(t.tableLockReadOwners, ss)
		} else {
			t.tableLockReadOwners[ss]--
		}
		if t.tableLockState.Add(-1) < 0 {
			t.tableLockMu.Unlock()
			panic("table READ lock count underflow")
		}
		cond.Broadcast()
		t.tableLockMu.Unlock()
	}
}

func lockTable(schema, name string, write bool, ss *scm.SessionState, querySeq uint64) {
	unlock := acquireTableLock(schema, name, write, false, ss, querySeq)
	if ss != nil {
		ss.AddLock(unlock)
	}
}

func withComputeInvalidationWave(t *table, colName string, tx *TxContext, invalidate func() bool) {
	key := t.schema.Name + "\x00" + t.Name + "\x00" + colName
	if tx != nil {
		tx.mu.Lock()
		if tx.invalidationDepth == 0 {
			tx.invalidationVisited = make(map[string]bool)
		}
		if tx.invalidationVisited[key] {
			tx.mu.Unlock()
			return
		}
		tx.invalidationVisited[key] = true
		tx.invalidationDepth++
		tx.mu.Unlock()
		defer func() {
			tx.mu.Lock()
			tx.invalidationDepth--
			if tx.invalidationDepth == 0 {
				tx.invalidationVisited = nil
			}
			tx.mu.Unlock()
		}()
	}
	if invalidate() {
		t.ExecuteTriggers(AfterInvalidate, nil, nil, tx)
	}
}

// invalidateComputedColumn propagates invalidation through computed-cache
// dependency edges. A transaction-local visited set makes one synchronous invalidation
// wave idempotent and prevents malformed cyclic computed definitions from
// recursing forever without introducing a global lock or cross-query state.
func invalidateComputedColumn(t *table, colName string, tx *TxContext) {
	withComputeInvalidationWave(t, colName, tx, func() bool {
		invalidated := false
		for _, s := range t.maintenanceShards() {
			s.mu.RLock()
			col := s.columns[colName]
			s.mu.RUnlock()
			if proxy, isProxy := col.(*StorageComputeProxy); isProxy {
				proxy.InvalidateAll()
				invalidated = true
			}
		}
		return invalidated
	})
}

// invalidateComputedRows preserves the exact subset found by an analyzed
// lookup-maintenance scan. The AfterInvalidate edge stays column-level, so a
// nested cache remains correct even when its own lookup relation cannot be
// narrowed safely. The common invalidation-wave guard prevents dependency
// cycles exactly as for invalidateComputedColumn.
func invalidateComputedRows(proxy *StorageComputeProxy, recids map[uint32]struct{}, currentTx *TxContext) {
	if proxy == nil || proxy.shard == nil || len(recids) == 0 {
		return
	}
	withComputeInvalidationWave(proxy.shard.t, proxy.colName, currentTx, func() bool {
		proxy.InvalidateRowsTx(currentTx, recids)
		return true
	})
}

func Init(en scm.Env) {
	const scanFilterColumnsDesc = "physical columns passed to filter before map/reduce; $recset_contains supplies a row-bound RecSet membership closure"
	const scanMapColumnsDesc = "physical columns passed to map after filtering; pseudo columns are $update (update/delete current row), $recset_contains (row-bound RecSet membership), $set:<column>, $increment:<column>, and $invalidate:<column> (computed-column maintenance), plus NEW.<column> in trigger plans"
	const scanOrderMapColumnsDesc = scanMapColumnsDesc + "; $break is reserved for internal ORC convergence and must not implement SQL OFFSET/LIMIT, which belong in the native offset and limit arguments"
	columnList := func(label, description string) *scm.TypeDescriptor {
		return &scm.TypeDescriptor{
			Kind:        "list",
			Label:       label,
			Description: description,
			Element: &scm.TypeDescriptor{
				Kind:        "string",
				Label:       "column",
				Description: "column name passed to the corresponding callback parameter",
			},
		}
	}
	rowCallback := func(label, description, returnKind, returnDescription string) *scm.TypeDescriptor {
		return &scm.TypeDescriptor{
			Kind:        "func",
			Label:       label,
			Description: description,
			Params: []*scm.TypeDescriptor{{
				Kind:        "any",
				Label:       "columns",
				Description: "one value for each entry in the matching column list, in the same order",
				Variadic:    true,
			}},
			Return: &scm.TypeDescriptor{Kind: returnKind, Label: "result", Description: returnDescription},
		}
	}
	reducer := func(label, description string) *scm.TypeDescriptor {
		return &scm.TypeDescriptor{
			Kind:        "func",
			Label:       label,
			Description: description,
			Optional:    true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "accumulator", Description: "current aggregate, initially the neutral value"},
				{Kind: "any", Label: "value", Description: "next mapped or partially reduced value"},
			},
			Return: &scm.TypeDescriptor{Kind: "any", Label: "accumulator", Description: "aggregate passed to the next reducer call or returned by the scan"},
		}
	}
	sortColumnList := func(label, description string) *scm.TypeDescriptor {
		return &scm.TypeDescriptor{
			Kind:        "list",
			Label:       label,
			Description: description,
			Element: &scm.TypeDescriptor{
				Kind:        "string|func",
				Label:       "sort column",
				Description: "a column name, or a function of row-column values that returns the sortable value",
				Params: []*scm.TypeDescriptor{{
					Kind: "any", Label: "columns", Description: "column values used to compute the sort key", Variadic: true,
				}},
				Return: &scm.TypeDescriptor{Kind: "any", Label: "sort key", Description: "value compared at this sort position"},
			},
		}
	}
	sortDirectionList := func(label, description string) *scm.TypeDescriptor {
		return &scm.TypeDescriptor{
			Kind:        "list",
			Label:       label,
			Description: description,
			Element: &scm.TypeDescriptor{
				Kind:        "func",
				Label:       "direction",
				Description: "strict ordering relation such as <, >, or a collate relation",
				Params: []*scm.TypeDescriptor{
					{Kind: "any", Label: "left", Description: "left sort value"},
					{Kind: "any", Label: "right", Description: "right sort value"},
				},
				Return: &scm.TypeDescriptor{Kind: "bool", Label: "ordered", Description: "true when left belongs before right"},
			},
		}
	}
	tableOptions := &scm.TypeDescriptor{
		Kind:        "list|assoc",
		Label:       "options",
		Description: "table options as an alternating key/value list",
		Keys: map[string]*scm.TypeDescriptor{
			"auto_increment": {Kind: "int", Label: "auto_increment", Description: "first automatically assigned value; must be non-negative"},
			"charset":        {Kind: "string", Label: "charset", Description: "default character set name"},
			"collation":      {Kind: "string", Label: "collation", Description: "default collation name"},
			"comment":        {Kind: "string", Label: "comment", Description: "user-visible table comment"},
			"engine":         {Kind: "string", Label: "engine", Description: "storage engine: safe, logged, sloppy, memory, or cache"},
			"oninit": {
				Kind:        "func",
				Label:       "oninit",
				Description: "closed zero-argument initializer run synchronously once per data generation; concurrent if-not-exists callers wait for completion",
				Params:      []*scm.TypeDescriptor{},
				Return:      &scm.TypeDescriptor{Kind: "any", Label: "result", Description: "ignored initializer result"},
			},
		},
	}
	columnOptions := &scm.TypeDescriptor{
		Kind:        "list|assoc",
		Label:       "options",
		Description: "column properties and computed-column configuration as an alternating key/value list",
		Keys: map[string]*scm.TypeDescriptor{
			"auto_increment":     {Kind: "bool", Label: "auto_increment", Description: "assign increasing values automatically"},
			"collate":            {Kind: "string", Label: "collate", Description: "collation used for this column"},
			"comment":            {Kind: "string", Label: "comment", Description: "user-visible column comment"},
			"default":            {Kind: "any", Label: "default", Description: "literal value used when an insert omits the column"},
			"default_expression": {Kind: "string", Label: "default_expression", Description: "expression evaluated when an insert omits the column"},
			"filtercols":         columnList("filtercols", "columns supplied to filter before computing a value"),
			"filter":             rowCallback("filter", "predicate limiting which rows are computed", "bool", "true when the row should be computed"),
			"mapcols":            columnList("mapcols", "columns supplied to mapfn for ordered-reduce computation"),
			"mapfn":              rowCallback("mapfn", "maps one source row into a value for reducefn", "any", "value passed to reducefn"),
			"null":               {Kind: "bool", Label: "null", Description: "whether the column accepts nil values"},
			"partitioncount":     {Kind: "int", Label: "partitioncount", Description: "number of leading sort columns that define independent reducer partitions"},
			"primary":            {Kind: "bool", Label: "primary", Description: "whether this column belongs to the primary key"},
			"reducefn":           reducer("reducefn", "combines ordered mapped values into the computed-column aggregate"),
			"reduceinit":         {Kind: "any", Label: "reduceinit", Description: "initial accumulator supplied to reducefn"},
			"sortcols":           sortColumnList("sortcols", "columns or expressions defining ordered-reduce input order"),
			"sortdirs":           sortDirectionList("sortdirs", "one ordering relation for every sortcols entry"),
			"temp":               {Kind: "bool", Label: "temp", Description: "whether this is a query-local temporary computed column"},
			"unique":             {Kind: "bool", Label: "unique", Description: "whether values must be unique"},
			"update":             {Kind: "any", Label: "update", Description: "expression evaluated when a row is updated"},
		},
	}
	scm.DeclareTitle("Storage")

	// Register TagTable serializer for the printer.
	scm.CustomStringer[TagTable] = func(ptr unsafe.Pointer) string {
		return (*table)(ptr).String()
	}
	scm.CustomStringer[TagRecSet] = func(ptr unsafe.Pointer) string {
		return (*recSet)(ptr).String()
	}

	scm.Declare(&en, &scm.Declaration{
		Name: "table",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				return scm.NewNil()
			}
			t := db.GetTable(scm.String(a[1]))
			if t == nil {
				return scm.NewNil()
			}
			return NewTableScmer(t)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "resolves a schema+table name pair into a table handle",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema"},
				{Kind: "string", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "table"},
		},
		Optimize: func(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
			// Do NOT fold to a constant pointer — DDL (DROP/CREATE TABLE)
			// invalidates the pointer and cached query plans would reference
			// a stale table. Runtime evaluation via GetTable is lock-free.
			for i := 1; i < len(v); i++ {
				v[i], _ = oc.OptimizeSub(v[i], true)
			}
			return scm.NewSlice(v), nil
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "scan_estimate",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewInt(int64(t.CountEstimate()))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "estimate output row count for a table scan",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "int"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_planner_statistics",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			return TableFromScmer(a[0]).PlannerStatistics()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "return the immutable O(1) planner-statistics snapshot for a table",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_planner_statistics_token",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			if a[0].IsNil() {
				return scm.NewNil()
			}
			return scm.NewInt(int64(TableFromScmer(a[0]).PlannerStatsToken()))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "return the process-unique dependency token of a table's immutable planner-statistics snapshot",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "int"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_planner_statistics_fingerprint",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			if a[0].IsNil() {
				return scm.NewNil()
			}
			return scm.NewInt(int64(TableFromScmer(a[0]).PlannerStatisticsFingerprint()))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "return the coarse cost-class fingerprint of a table's immutable planner-statistics snapshot",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "int"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_order_partitioned?",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			column := scm.String(a[1])
			topology := t.activeTopology()
			return scm.NewBool(topology.mode == ShardModePartition &&
				len(topology.dimensions) == 1 && topology.dimensions[0].Column == column)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "reports whether a table has one range-partition dimension matching the leading ORDER BY column; this immutable-topology check lets physical costing account for ordered shard pruning",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_selectivity_estimate",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			t := TableFromScmer(a[1])
			conditionCols := scmerSliceToStrings(mustScmerSlice(a[2], "condition columns"))
			condition := a[3]
			limit := scm.ToInt(a[4])
			if limit <= 0 {
				limit = 1024
			}
			shards := t.ActiveShards()
			input := int64(t.CountEstimate())
			if len(shards) == 0 || input == 0 {
				return scm.NewSlice([]scm.Scmer{
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("rows"), scm.NewInt(0)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("capped"), scm.NewBool(false)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("sampled"), scm.NewInt(0)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("input"), scm.NewInt(input)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("population"), scm.NewSymbol("table_rows")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("coverage"), scm.NewSymbol("exact")}),
				})
			}
			start := 0
			if len(shards) > 1 {
				start = rand.Intn(len(shards))
			}
			// Costing must never depend on whether an auto-index already exists and
			// must not turn planning into a scan of every shard. Sample one randomly
			// selected non-empty shard. An installed index hook makes the same call
			// constant-time and more precise; the generated crossover guard then
			// invalidates a cached choice if that new selectivity changes the winner.
			for offset := range shards {
				shard := shards[(start+offset)%len(shards)]
				if shard == nil {
					continue
				}
				estimate := func() filteredRowEstimate {
					release := shard.GetRead()
					defer release()
					return shard.EstimateFilteredRows(conditionCols, condition, limit, currentTx)
				}()
				if estimate.examined == 0 {
					continue
				}
				if estimate.population == "index_hook_candidates" && estimate.examined > 0 {
					estimatedRows := input * estimate.rows / estimate.examined
					return scm.NewSlice([]scm.Scmer{
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("rows"), scm.NewInt(estimatedRows)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("capped"), scm.NewBool(false)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("sampled"), scm.NewInt(estimate.examined)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("input"), scm.NewInt(input)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("population"), scm.NewSymbol(estimate.population)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("coverage"), scm.NewSymbol(estimate.coverage)}),
					})
				}
				// An ordinary index range examines only matching candidates. Its
				// candidate count is the numerator of a shard sample, not the sample
				// population itself. Scale it by the selected shard's visible row
				// universe; using examined here turns every exact equality range into
				// an apparent 100%-selective predicate.
				if estimate.population == "index_candidates" && estimate.universe > 0 {
					coverage := estimate.coverage
					if len(shards) > 1 {
						coverage = "sampled"
					}
					return scm.NewSlice([]scm.Scmer{
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("rows"), scm.NewInt(estimate.rows)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("capped"), scm.NewBool(estimate.capped)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("sampled"), scm.NewInt(estimate.universe)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("input"), scm.NewInt(input)}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("population"), scm.NewSymbol("table_rows")}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("coverage"), scm.NewSymbol(coverage)}),
					})
				}
				return scm.NewSlice([]scm.Scmer{
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("rows"), scm.NewInt(estimate.rows)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("capped"), scm.NewBool(estimate.capped)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("sampled"), scm.NewInt(estimate.examined)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("input"), scm.NewInt(input)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("population"), scm.NewSymbol("table_rows")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("coverage"), scm.NewSymbol("sampled")}),
				})
			}
			// All active shards were empty snapshots. The table estimate may include
			// concurrent deltas; keep the estimate unknown rather than claiming zero.
			return scm.NewSlice([]scm.Scmer{
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("rows"), scm.NewInt(0)}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("capped"), scm.NewBool(true)}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("sampled"), scm.NewInt(0)}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("input"), scm.NewInt(input)}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("population"), scm.NewSymbol("table_rows")}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("coverage"), scm.NewSymbol("lower_bound")}),
			})
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "bounded estimate of visible rows matching a table filter; stops at max_rows and does not log scan telemetry",
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table", Label: "table"},
				columnList("condition_cols", "columns passed to the selectivity predicate"),
				rowCallback("condition", "predicate sampled to estimate matching rows", "bool", "true when the sampled row matches"),
				{Kind: "int", Label: "max_rows"},
			},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_empty?",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewBool(t.CountExact() == 0)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns true if a table currently has no rows",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_recset",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			filtercols := scmerSliceToStrings(mustScmerSlice(a[2], "filterColumns"))
			if a[1].IsCustom(TagRecSet) {
				return NewRecSetScmer(RecSetFromScmer(a[1]).filterToRecSet(currentTx, filtercols, a[3]))
			}
			t := TableFromScmer(a[1])
			return NewRecSetScmer(t.scanRecSet(currentTx, filtercols, a[3]))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "builds a query-local record-set handle from one table scan, or -- when given an existing recset instead of a table -- narrows that recset to the members which also satisfy filter, re-evaluating filter only over its existing membership. The latter is the cheap way to AND a further (possibly subscan-heavy) condition onto an already-narrowed recset without re-touching rows outside it (e.g. evaluating an expensive correlated check only over the rows a cheap selective filter already narrowed a table down to). The returned value is not persisted and can be scanned like a table",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "any", Label: "table", Description: "a table, or an existing recset to narrow further"},
				columnList("filterColumns", scanFilterColumnsDesc),
				rowCallback("filter", "lambda function that decides whether a row enters the recset", "bool", "true when the row belongs in the recset"),
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_count",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			return scm.NewInt(RecSetFromScmer(a[0]).count)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the number of currently stored recids in a query-local recset",
			Params: []*scm.TypeDescriptor{
				{Kind: "recset", Label: "recset"},
			},
			Return: &scm.TypeDescriptor{Kind: "int"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_project_join",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			source := RecSetFromScmer(a[1])
			sourceKeyCols := scmerSliceToStrings(mustScmerSlice(a[2], "sourceKeyColumns"))
			target := TableFromScmer(a[3])
			targetKeyCols := scmerSliceToStrings(mustScmerSlice(a[4], "targetKeyColumns"))
			return NewRecSetScmer(source.projectJoin(currentTx, sourceKeyCols, target, targetKeyCols))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "projects a source recset through key columns into a query-local target-table recset",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "recset", Label: "source_recset"},
				{Kind: "list", Label: "source_key_columns"},
				{Kind: "table", Label: "target_table"},
				{Kind: "list", Label: "target_key_columns"},
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_key_index",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			source := RecSetFromScmer(a[1])
			sourceKeyCols := scmerSliceToStrings(mustScmerSlice(a[2], "sourceKeyColumns"))
			if len(sourceKeyCols) == 0 {
				panic("recset_key_index requires at least one key column")
			}
			keys := source.collectProjectJoinKeys(currentTx, sourceKeyCols, SessionStateFromTx(currentTx))
			return scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
				if len(values) != keys.width {
					panic("recset key lookup received the wrong number of key values")
				}
				return scm.NewBool(keys.contains(values))
			})
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "builds an immutable lookup function for key columns of the rows contained in a query-local recset",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context used while reading source keys"},
				{Kind: "recset", Label: "source_recset"},
				{Kind: "list", Label: "source_key_columns"},
			},
			Return: &scm.TypeDescriptor{Kind: "func", Label: "lookup", Description: "tests whether the recset contains a row with the supplied composite key", Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "key", Description: "one value for each source key column, in the same order", Variadic: true},
			}, Return: &scm.TypeDescriptor{Kind: "bool", Label: "present", Description: "whether the composite key occurs in the recset"}},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_union",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			values := mustScmerSlice(a[0], "recsets")
			recsets := make([]*recSet, 0, len(values))
			for _, value := range values {
				recsets = append(recsets, RecSetFromScmer(value))
			}
			return NewRecSetScmer(recSetUnion(recsets))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "combines query-local recsets from the same table and removes duplicate record IDs",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "list", Label: "recsets"},
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_intersect",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			values := mustScmerSlice(a[0], "recsets")
			recsets := make([]*recSet, 0, len(values))
			for _, value := range values {
				recsets = append(recsets, RecSetFromScmer(value))
			}
			return NewRecSetScmer(recSetIntersect(recsets))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "intersects query-local recsets from the same table",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "list", Label: "recsets"},
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_difference",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			values := mustScmerSlice(a[0], "recsets")
			recsets := make([]*recSet, 0, len(values))
			for _, value := range values {
				recsets = append(recsets, RecSetFromScmer(value))
			}
			return NewRecSetScmer(recSetDifference(recsets))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the records from the first query-local recset which occur in none of the following same-table recsets",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "list", Label: "recsets"},
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "recset_not",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			return NewRecSetScmer(recSetNot(RecSetFromScmer(a[0])))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the complement of a query-local recset relative to the currently visible rows of its base table",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "recset", Label: "recset"},
			},
			Return: &scm.TypeDescriptor{Kind: "recset"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_exists",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			filtercols := scmerSliceToStrings(mustScmerSlice(a[2], "filterColumns"))
			tableArg := a[1]
			if list, ok := scmerSlice(tableArg); ok {
				filterfn := scm.OptimizeProcToSerialFunction(a[3])
				filterparams := make([]scm.Scmer, len(filtercols))
				for _, val := range list {
					row := mustScmerSlice(val, "scan_exists list row")
					ds := dataset(row)
					for i, col := range filtercols {
						filterparams[i], _ = ds.GetI(col)
					}
					if scm.ToBool(filterfn(filterparams...)) {
						return scm.NewBool(true)
					}
				}
				return scm.NewBool(false)
			}
			if tableArg.IsCustom(TagRecSet) {
				return scm.NewBool(RecSetFromScmer(tableArg).scanExists(currentTx, filtercols, a[3]))
			}
			t := TableFromScmer(tableArg)
			return scm.NewBool(t.scanExists(currentTx, filtercols, a[3]))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns true if a table contains at least one visible row matching the given filter; uses scan boundary analysis without map/reduce setup",
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list|recset", Label: "table"},
				columnList("filterColumns", scanFilterColumnsDesc),
				rowCallback("filter", "lambda function that decides whether a row exists", "bool", "true when the row satisfies the existence test"),
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "scan",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			layout := scanLayout(a)
			filtercols := scmerSliceToStrings(mustScmerSlice(a[layout.filterColsIdx], "filterColumns"))
			mapcols := scmerSliceToStrings(mustScmerSlice(a[layout.mapColsIdx], "mapColumns"))
			tableArg := a[layout.tableIdx]
			isOuter := len(a) > layout.outerIdx && scm.ToBool(a[layout.outerIdx])

			if list, ok := scmerSlice(tableArg); ok {
				neutral := scm.NewNil()
				if len(a) > layout.neutralIdx {
					neutral = a[layout.neutralIdx]
				}
				result := neutral
				filterfn := scm.OptimizeProcToSerialFunction(a[layout.filterFnIdx])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[layout.mapFnIdx])
				mapparams := make([]scm.Scmer, len(mapcols))
				reducefn := func(args ...scm.Scmer) scm.Scmer { return args[1] }
				if len(a) > layout.reduceIdx {
					reducefn = scm.OptimizeProcToSerialFunction(a[layout.reduceIdx])
				}
				hadValue := false
				for _, val := range list {
					row := mustScmerSlice(val, "scan list row")
					ds := dataset(row)
					for i, col := range filtercols {
						filterparams[i], _ = ds.GetI(col)
					}
					if !scm.ToBool(filterfn(filterparams...)) {
						continue
					}
					hadValue = true
					for i, col := range mapcols {
						mapparams[i], _ = ds.GetI(col)
					}
					result = reducefn(result, mapfn(mapparams...))
				}
				if !hadValue && isOuter {
					for i := range mapparams {
						mapparams[i] = scm.NewNil()
					}
					result = reducefn(result, mapfn(mapparams...))
				}
				if len(a) > layout.reduce2Idx && !a[layout.reduce2Idx].IsNil() {
					reduce2fn := scm.OptimizeProcToSerialFunction(a[layout.reduce2Idx])
					base := neutral
					if len(a) > layout.neutralIdx {
						base = a[layout.neutralIdx]
					}
					result = reduce2fn(base, result)
				}
				return result
			}

			if tableArg.IsCustom(TagRecSet) {
				rs := RecSetFromScmer(tableArg)
				aggregate := scm.NewNil()
				if len(a) > layout.reduceIdx {
					aggregate = a[layout.reduceIdx]
				}
				neutral := scm.NewNil()
				if len(a) > layout.neutralIdx {
					neutral = a[layout.neutralIdx]
				}
				reduce2 := scm.NewNil()
				if len(a) > layout.reduce2Idx {
					reduce2 = a[layout.reduce2Idx]
				}
				return rs.scan(layout.tx, filtercols, a[layout.filterFnIdx], mapcols, a[layout.mapFnIdx], aggregate, neutral, reduce2, isOuter)
			}

			t := TableFromScmer(tableArg)

			aggregate := scm.NewNil()
			if len(a) > layout.reduceIdx {
				aggregate = a[layout.reduceIdx]
			}
			neutral := scm.NewNil()
			if len(a) > layout.neutralIdx {
				neutral = a[layout.neutralIdx]
			}
			reduce2 := scm.NewNil()
			if len(a) > layout.reduce2Idx {
				reduce2 = a[layout.reduce2Idx]
			}
			return t.scan(layout.tx, filtercols, a[layout.filterFnIdx], mapcols, a[layout.mapFnIdx], aggregate, neutral, reduce2, isOuter)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "does an unordered parallel filter-map-reduce pass on a single table and returns the reduced result",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list|recset", Label: "table", Description: "table handle, query-local recset, or a list for temporary data"},
				columnList("filterColumns", scanFilterColumnsDesc),
				rowCallback("filter", "lambda function that decides whether a dataset is passed to the map phase. Equality and range comparisons may be translated into indexed scans", "bool", "true when the row proceeds to map"),
				columnList("mapColumns", scanMapColumnsDesc),
				rowCallback("map", "lambda function that extracts or produces one value from the row; it may also use documented pseudo columns for mutations or result output", "any", "value passed to reduce, or returned directly when no reducer is supplied"),
				reducer("reduce", "optional aggregation function used first within shards and then to combine shard results"),
				{Kind: "any", Label: "neutral", Description: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				reducer("reduce2", "optional final reducer that combines the neutral value with the result produced by reduce"),
				{Kind: "bool", Label: "isOuter", Description: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
		Optimize: optimizeScan,
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_batch",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			layout := scanLayout(a)
			filtercols := scmerSliceToStrings(mustScmerSlice(a[layout.filterColsIdx], "filterColumns"))
			mapcols := scmerSliceToStrings(mustScmerSlice(a[layout.mapColsIdx], "mapColumns"))
			stride := int(scm.ToInt(a[layout.strideIdx]))
			batchdata := mustScmerSlice(a[layout.batchDataIdx], "batchdata")
			tableArg := a[layout.tableIdx]
			// scan_batch inserts stride+batchdata (2 slots) between mapfn and
			// reduce, so reduce/neutral/reduce2/isOuter sit at scanLayout's
			// reduceIdx/neutralIdx/reduce2Idx/outerIdx + 2.
			const sbShift = 2
			isOuter := len(a) > layout.outerIdx+sbShift && scm.ToBool(a[layout.outerIdx+sbShift])

			if list, ok := scmerSlice(tableArg); ok {
				neutral := scm.NewNil()
				if len(a) > layout.neutralIdx+sbShift {
					neutral = a[layout.neutralIdx+sbShift]
				}
				result := neutral
				filterfn := scm.OptimizeProcToSerialFunction(a[layout.filterFnIdx])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[layout.mapFnIdx])
				mapparams := make([]scm.Scmer, len(mapcols))
				reducefn := func(args ...scm.Scmer) scm.Scmer { return args[1] }
				if len(a) > layout.reduceIdx+sbShift {
					reducefn = scm.OptimizeProcToSerialFunction(a[layout.reduceIdx+sbShift])
				}
				hadValue := false
				batchCount := 0
				if stride > 0 {
					batchCount = len(batchdata) / stride
				}
				for batchid := 0; batchid < batchCount; batchid++ {
					for _, val := range list {
						row := mustScmerSlice(val, "scan_batch list row")
						ds := dataset(row)
						for i, col := range filtercols {
							if subidx, ok := parseBatchPseudoColName(col); ok {
								filterparams[i] = batchdata[batchid*stride+subidx]
							} else {
								filterparams[i], _ = ds.GetI(col)
							}
						}
						if !scm.ToBool(filterfn(filterparams...)) {
							continue
						}
						hadValue = true
						for i, col := range mapcols {
							if subidx, ok := parseBatchPseudoColName(col); ok {
								mapparams[i] = batchdata[batchid*stride+subidx]
							} else {
								mapparams[i], _ = ds.GetI(col)
							}
						}
						result = reducefn(result, mapfn(mapparams...))
					}
				}
				if !hadValue && isOuter {
					for i := range mapparams {
						mapparams[i] = scm.NewNil()
					}
					result = reducefn(result, mapfn(mapparams...))
				}
				if len(a) > layout.reduce2Idx+sbShift && !a[layout.reduce2Idx+sbShift].IsNil() {
					reduce2fn := scm.OptimizeProcToSerialFunction(a[layout.reduce2Idx+sbShift])
					base := neutral
					if len(a) > layout.neutralIdx+sbShift {
						base = a[layout.neutralIdx+sbShift]
					}
					result = reduce2fn(base, result)
				}
				return result
			}

			var source *recSet
			var t *table
			if tableArg.IsCustom(TagRecSet) {
				source = RecSetFromScmer(tableArg)
				t = source.table
			} else {
				t = TableFromScmer(tableArg)
			}

			aggregate := scm.NewNil()
			if len(a) > layout.reduceIdx+sbShift {
				aggregate = a[layout.reduceIdx+sbShift]
			}
			neutral := scm.NewNil()
			if len(a) > layout.neutralIdx+sbShift {
				neutral = a[layout.neutralIdx+sbShift]
			}
			reduce2 := scm.NewNil()
			if len(a) > layout.reduce2Idx+sbShift {
				reduce2 = a[layout.reduce2Idx+sbShift]
			}
			return t.scanWithBatchFrom(layout.tx, source, filtercols, a[layout.filterFnIdx], mapcols, a[layout.mapFnIdx], aggregate, neutral, reduce2, isOuter, stride, batchdata, nil)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "does an unordered parallel filter-map-reduce pass on a single table using batchdata-backed #N pseudo columns and returns the reduced result",
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list|recset", Label: "table", Description: "table handle, query-local recset, or a list for temporary data"},
				columnList("filterColumns", "columns passed to filter; #0, #1, ... address batchdata slots"),
				rowCallback("filter", "lambda function that decides whether a dataset is passed to the map phase", "bool", "true when this table row and batch row proceed to map"),
				columnList("mapColumns", "columns passed to map; #0, #1, ... address batchdata slots"),
				rowCallback("map", "lambda function that extracts data from the table row and batch row", "any", "value passed to reduce or returned directly"),
				{Kind: "int", Label: "stride", Description: "number of batchdata entries per batch row"},
				{Kind: "list", Label: "batchdata", Description: "flat batch buffer accessed via #N pseudo columns", Element: &scm.TypeDescriptor{Kind: "any", Label: "slot", Description: "one batch value; every stride consecutive slots form a batch row"}},
				reducer("reduce", "optional lambda function that aggregates mapped values"),
				{Kind: "any", Label: "neutral", Description: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				reducer("reduce2", "optional final reducer that combines the neutral value with the result produced by reduce"),
				{Kind: "bool", Label: "isOuter", Description: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
		Optimize: optimizeScanBatch,
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_order_batch_accept",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			tableArg := a[1]
			batchFilter := a[2]
			sortcolsVals := mustScmerSlice(a[3], "sortcols")
			sortdirsVals := mustScmerSlice(a[4], "sortdirs")
			sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortdirsVals))
			for i, dir := range sortdirsVals {
				sortdirs[i] = scm.OptimizeProcToSerialFunction(dir)
			}
			limitPartitionCols := scm.ToInt(a[5])
			offset := scm.ToInt(a[6])
			limit := scm.ToInt(a[7])
			mapcols := scmerSliceToStrings(mustScmerSlice(a[8], "mapColumns"))
			callback := a[9]
			aggregate := scm.NewNil()
			if len(a) > 10 {
				aggregate = a[10]
			}
			neutral := scm.NewNil()
			if len(a) > 11 {
				neutral = a[11]
			}
			isOuter := len(a) > 12 && scm.ToBool(a[12])
			notFoundValue := neutral
			if len(a) > 13 {
				notFoundValue = a[13]
			}
			source := scanOrderTableSpec{}
			if tableArg.IsCustom(TagRecSet) {
				source.recset = RecSetFromScmer(tableArg)
			} else {
				source.table = TableFromScmer(tableArg)
			}
			return scanOrderBatchAccept(currentTx, source, batchFilter, sortcolsVals, sortdirs,
				limitPartitionCols, offset, limit, mapcols, callback, aggregate, neutral, isOuter, notFoundValue)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "incrementally scans a table or existing RecSet in scan_order order and applies a RecSet batch filter before OFFSET/LIMIT and map/reduce. The first candidate RecSet contains offset+limit rows; if too few rows are accepted, subsequent disjoint batches contain twice as many candidates until the accepted limit is satisfied or the input is exhausted. batchFilter is called as (batchFilter input_recset) and must return an exact subset RecSet of the same base table and transaction. A simple batchFilter may call (scan_recset tx input_recset filterColumns realFilter); complex filters may project input_recset to another table, apply search/ACL scans and project the result back to the input table. The returned RecSet is used only as a membership mask against the already ordered candidate vector, so output order is preserved without scanning the unordered RecSet again. For non-unique ORDER BY values, include an explicit unique tie-breaker. sortcols/sortdirs may both be empty; that path greedily collects candidates without sorting. limitPartitionCols is present for scan_order signature compatibility and currently must be 0",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context used consistently by the candidate scan and every batch filter operation; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|recset", Label: "table_or_recset", Description: "base table or complete existing query-local RecSet from which ordered candidate batches are drawn"},
				{Kind: "func", Label: "batchFilter", Description: "function (lambda (input_recset) accepted_recset). It may naively narrow input_recset with scan_recset, or run arbitrary RecSet projections/search/ACL operations and project back. It must return a same-table, same-transaction subset of input_recset", Params: []*scm.TypeDescriptor{{Kind: "recset", Label: "input_recset"}}, Return: &scm.TypeDescriptor{Kind: "recset"}},
				sortColumnList("sortcols", "same as scan_order: columns or computed sort functions. Include a unique tie-breaker for a total repeatable order; use an empty list for greedy unsorted collection"),
				sortDirectionList("sortdirs", "same as scan_order: one relation per sort column; must also be empty when sortcols is empty"),
				{Kind: "number", Label: "limitPartitionCols", Description: "reserved for scan_order signature compatibility; currently must be 0"},
				{Kind: "number", Label: "offset", Description: "number of batch-filter-accepted rows to skip; it is not the number of driver candidates already examined"},
				{Kind: "number", Label: "limit", Description: "finite maximum number of accepted rows passed to map; the initial candidate batch size is offset+limit and doubles for every subsequent batch"},
				columnList("mapColumns", scanOrderMapColumnsDesc),
				rowCallback("map", "same map callback contract as scan_order; accepted record IDs are passed to its shard mapper in batches", "any", "value passed to reduce or returned directly"),
				reducer("reduce", "optional serial reducer over mapped accepted rows, with the same accumulator contract as scan_order"),
				{Kind: "any", Label: "neutral", Description: "optional neutral element for reduce; defaults to nil", Optional: true},
				{Kind: "bool", Label: "isOuter", Description: "optional scan_order-compatible outer behavior: map one NULL row when no accepted row reaches map", Optional: true},
				{Kind: "any", Label: "notFoundValue", Description: "optional result when no accepted row reaches map and isOuter is false; defaults to neutral", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
		Optimize: optimizeScanOrderBatchAccept,
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_order",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			layout := scanLayout(a)
			filtercols := scmerSliceToStrings(mustScmerSlice(a[layout.filterColsIdx], "filterColumns"))
			sortcolsVals := mustScmerSlice(a[layout.sortColsIdx], "sortcols")
			sortdirsVals := mustScmerSlice(a[layout.sortDirsIdx], "sortdirs")
			limitPartitionCols := scm.ToInt(a[layout.partColsIdx])
			mapcols := scmerSliceToStrings(mustScmerSlice(a[layout.limitIdx+1], "mapColumns"))
			tableArg := a[layout.tableIdx]

			aggregate := scm.NewNil()
			if len(a) > layout.limitIdx+3 {
				aggregate = a[layout.limitIdx+3]
			}
			neutral := scm.NewNil()
			if len(a) > layout.limitIdx+4 {
				neutral = a[layout.limitIdx+4]
			}

			sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortcolsVals))
			for i, dir := range sortdirsVals {
				sortdirs[i] = scm.OptimizeProcToSerialFunction(dir)
			}

			isOuter := len(a) > layout.limitIdx+5 && scm.ToBool(a[layout.limitIdx+5])
			notFoundValue := neutral
			if len(a) > layout.limitIdx+6 {
				notFoundValue = a[layout.limitIdx+6]
			}
			postOrderCols := []string(nil)
			postOrderFilter := scm.NewNil()
			if len(a) > layout.limitIdx+8 {
				postOrderCols = scmerSliceToStrings(mustScmerSlice(a[layout.limitIdx+7], "postOrderFilterColumns"))
				postOrderFilter = a[layout.limitIdx+8]
			}

			// TODO(planner-scalability): remove list-backed relational scans after
			// metadata/RDF callers use physical scan sources. Query plans must never
			// materialize cardinality-dependent rows into SCM lists.
			if list, ok := scmerSlice(tableArg); ok {
				result := neutral
				filterfn := scm.OptimizeProcToSerialFunction(a[layout.filterFnIdx])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[layout.limitIdx+2])
				mapparams := make([]scm.Scmer, len(mapcols))
				var postOrderFn func(...scm.Scmer) scm.Scmer
				postOrderParams := make([]scm.Scmer, len(postOrderCols))
				if !postOrderFilter.IsNil() {
					postOrderFn = scm.OptimizeProcToSerialFunction(postOrderFilter)
				}
				reducefn := func(args ...scm.Scmer) scm.Scmer { return args[1] }
				if !aggregate.IsNil() {
					reducefn = scm.OptimizeProcToSerialFunction(aggregate)
				}
				var filtered []scm.Scmer
				for _, val := range list {
					row := mustScmerSlice(val, "scan_order list row")
					ds := dataset(row)
					for i, col := range filtercols {
						filterparams[i], _ = ds.GetI(col)
					}
					if scm.ToBool(filterfn(filterparams...)) {
						filtered = append(filtered, val)
					}
				}
				scols := make([]func(uint32) scm.Scmer, len(sortcolsVals))
				for i, scol := range sortcolsVals {
					if scol.IsString() {
						colname := scol.String()
						scols[i] = func(idx uint32) scm.Scmer {
							row := mustScmerSlice(filtered[idx], "sort row")
							ds := dataset(row)
							val, _ := ds.GetI(colname)
							return val
						}
						continue
					}
					proc := scm.OptimizeProcToSerialFunction(scol)
					var params []scm.Scmer
					if slice, ok := scmerSlice(scol); ok {
						params = slice
					}
					scols[i] = func(idx uint32) scm.Scmer {
						row := mustScmerSlice(filtered[idx], "sort row")
						ds := dataset(row)
						args := make([]scm.Scmer, len(params))
						for j, p := range params {
							args[j], _ = ds.GetI(scm.String(p))
						}
						return proc(args...)
					}
				}
				hybridsort.Slice(filtered, func(i, j int) bool {
					for c := 0; c < len(scols); c++ {
						a := scols[c](uint32(i))
						b := scols[c](uint32(j))
						if scm.ToBool(sortdirs[c](a, b)) {
							return true
						} else if scm.ToBool(sortdirs[c](b, a)) {
							return false
						}
					}
					return false
				})
				offset := int(scm.ToInt(a[layout.offsetIdx]))
				limit := int(scm.ToInt(a[layout.limitIdx]))
				hadValue := false
				count := 0
				accepted := 0
				for _, val := range filtered {
					row := mustScmerSlice(val, "scan_order row")
					ds := dataset(row)
					if postOrderFn != nil {
						for i, col := range postOrderCols {
							postOrderParams[i], _ = ds.GetI(col)
						}
						if !scm.ToBool(postOrderFn(postOrderParams...)) {
							continue
						}
					}
					if accepted < offset {
						accepted++
						continue
					}
					if limit >= 0 && count >= limit {
						break
					}
					for i, col := range mapcols {
						mapparams[i], _ = ds.GetI(col)
					}
					result = reducefn(result, mapfn(mapparams...))
					hadValue = true
					count++
				}
				if !hadValue && isOuter {
					for i := range mapparams {
						mapparams[i] = scm.NewNil()
					}
					result = reducefn(result, mapfn(mapparams...))
				}
				if !hadValue && !isOuter {
					result = notFoundValue
				}
				return result
			}

			if tableArg.IsCustom(TagRecSet) {
				return RecSetFromScmer(tableArg).scan_order(layout.tx, filtercols, a[layout.filterFnIdx], sortcolsVals, sortdirs, limitPartitionCols, scm.ToInt(a[layout.offsetIdx]), scm.ToInt(a[layout.limitIdx]), mapcols, a[layout.limitIdx+2], aggregate, neutral, isOuter, notFoundValue, postOrderCols, postOrderFilter)
			}

			t := TableFromScmer(a[layout.tableIdx])

			return t.scan_order(layout.tx, filtercols, a[layout.filterFnIdx], sortcolsVals, sortdirs, limitPartitionCols, scm.ToInt(a[layout.offsetIdx]), scm.ToInt(a[layout.limitIdx]), mapcols, a[layout.limitIdx+2], aggregate, neutral, isOuter, notFoundValue, postOrderCols, postOrderFilter)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result",
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list|recset", Label: "table", Description: "table handle, query-local RecSet, or a list for temporary data"},
				columnList("filterColumns", scanFilterColumnsDesc),
				rowCallback("filter", "lambda function that decides whether a dataset is passed to the map phase. Equality and range comparisons may be translated into indexed scans", "bool", "true when the row proceeds to ordering and map"),
				sortColumnList("sortcols", "columns used for ordering; each entry corresponds to one relation in sortdirs"),
				sortDirectionList("sortdirs", "one ordering relation per entry in sortcols; < is ascending and > is descending"),
				{Kind: "number", Label: "limitPartitionCols", Description: "number of leading sort columns that form the partition key for per-partition offset/limit. 0 (default) means global offset/limit."},
				{Kind: "number", Label: "offset", Description: "number of globally ordered, filter-accepted items to skip before map; apply SQL OFFSET here rather than in map"},
				{Kind: "number", Label: "limit", Description: "maximum globally ordered, filter-accepted items passed to map; -1 means unlimited; apply SQL LIMIT here so shard-local Top-K and the global merge can brake early"},
				columnList("mapColumns", scanOrderMapColumnsDesc),
				rowCallback("map", "lambda function that extracts or produces one value from each accepted row", "any", "value passed to reduce or returned directly"),
				reducer("reduce", "optional serial aggregation function over mapped values"),
				{Kind: "any", Label: "neutral", Description: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				{Kind: "bool", Label: "isOuter", Description: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
				{Kind: "any", Label: "notFoundValue", Description: "(optional) result for no hits when isOuter is false; defaults to neutral", Optional: true},
				func() *scm.TypeDescriptor {
					value := columnList("postOrderFilterColumns", "optional columns for a predicate evaluated in global order before OFFSET/LIMIT are counted; use for expensive acceptance checks that cannot participate in index boundaries")
					value.Optional = true
					return value
				}(),
				func() *scm.TypeDescriptor {
					value := rowCallback("postOrderFilter", "optional late acceptance predicate. Rejected rows do not count toward OFFSET/LIMIT and never reach map", "bool", "true when the ordered row counts toward OFFSET/LIMIT and reaches map")
					value.Optional = true
					return value
				}(),
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
		Optimize: optimizeScanOrder,
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_order_multi",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			// Parameters:
			// 0:  tx
			// 1:  tables list (table handles)
			// 2:  filterColumns per table (list of lists)
			// 3:  filterFns per table (list of lambdas)
			// 4:  sortcols per table (list of lists)
			// 5:  sortdirs (shared list — used for both per-table top-K and outer merge)
			// 6:  perTableOffset (list of int, or nil; -1 per entry = disable for that table)
			// 7:  perTableLimit  (list of int, or nil; -1 per entry = disable for that table)
			// 8:  limitPartitionCols (global Top-K-per-partition across merge)
			// 9:  offset (global)
			// 10: limit  (global; -1 = unlimited)
			// 11: mapColumns per table (list of lists)
			// 12: mapFns per table (list of lambdas)
			// 13: reduce (optional)
			// 14: neutral (optional)
			// 15: isOuter (optional)
			// 16: notFoundValue (optional)
			currentTx := scmerToTxContext(a[0])
			tables := mustScmerSlice(a[1], "tables")
			filterColsArr := mustScmerSlice(a[2], "filterColumns")
			filterFnArr := mustScmerSlice(a[3], "filterFns")
			sortcolsArr := mustScmerSlice(a[4], "sortcols")
			sortdirsVals := mustScmerSlice(a[5], "sortdirs")
			n := len(tables)
			perTableOffsets := decodePerTableInts(a[6], n, "perTableOffset")
			perTableLimits := decodePerTableInts(a[7], n, "perTableLimit")
			limitPartitionCols := scm.ToInt(a[8])
			offset := scm.ToInt(a[9])
			limit := scm.ToInt(a[10])
			mapColsArr := mustScmerSlice(a[11], "mapColumns")
			mapFnArr := mustScmerSlice(a[12], "mapFns")

			aggregate := scm.NewNil()
			if len(a) > 13 {
				aggregate = a[13]
			}
			neutral := scm.NewNil()
			if len(a) > 14 {
				neutral = a[14]
			}
			isOuter := len(a) > 15 && scm.ToBool(a[15])
			notFoundValue := neutral
			if len(a) > 16 {
				notFoundValue = a[16]
			}

			if len(filterColsArr) != n || len(filterFnArr) != n || len(sortcolsArr) != n || len(mapColsArr) != n || len(mapFnArr) != n {
				panic("scan_order_multi: all per-table arrays must have the same length")
			}

			sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortdirsVals))
			for i, dir := range sortdirsVals {
				sortdirs[i] = scm.OptimizeProcToSerialFunction(dir)
			}

			specs := make([]scanOrderTableSpec, n)
			for i := 0; i < n; i++ {
				specs[i] = scanOrderTableSpec{
					conditionCols:  scmerSliceToStrings(mustScmerSlice(filterColsArr[i], "filterColumns[i]")),
					condition:      filterFnArr[i],
					sortcols:       mustScmerSlice(sortcolsArr[i], "sortcols[i]"),
					callbackCols:   scmerSliceToStrings(mustScmerSlice(mapColsArr[i], "mapColumns[i]")),
					callback:       mapFnArr[i],
					perTableOffset: perTableOffsets[i],
					perTableLimit:  perTableLimits[i],
				}
				if tables[i].IsCustom(TagRecSet) {
					specs[i].recset = RecSetFromScmer(tables[i])
				} else {
					specs[i].table = TableFromScmer(tables[i])
				}
			}

			return scanOrderMulti(currentTx, specs, sortdirs, int(limitPartitionCols), int(offset), int(limit), aggregate, neutral, isOuter, notFoundValue)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "does an ordered parallel filter and serial map-reduce pass across multiple tables simultaneously, merging results into a single sorted stream",
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context"},
				{Kind: "list", Label: "tables", Description: "scan sources; all per-table lists must have this length", Element: &scm.TypeDescriptor{Kind: "table|recset", Label: "source", Description: "base table or query-local record set for one input stream"}},
				{Kind: "list", Label: "filterColumns", Description: "filter column lists, one per table", Element: &scm.TypeDescriptor{Kind: "list", Label: "table filter columns", Description: "columns supplied to the matching filterFns entry", Element: &scm.TypeDescriptor{Kind: "string", Label: "column", Description: "column name in the corresponding table"}}},
				{Kind: "list", Label: "filterFns", Description: "filter lambdas, one per table", Element: rowCallback("table filter", "predicate for the corresponding table and filterColumns entry", "bool", "true when the row enters that table's ordered stream")},
				{Kind: "list", Label: "sortcols", Description: "sort column lists, one per table; every inner list must match sortdirs in length and result domains", Element: sortColumnList("table sort columns", "sort expressions for the corresponding table")},
				sortDirectionList("sortdirs", "shared ordering relations used for every table stream and for the outer merge"),
				{Kind: "list|nil", Label: "perTableOffset", Description: "optional per-table offsets; nil disables all per-table offsets", Element: &scm.TypeDescriptor{Kind: "int", Label: "offset", Description: "rows skipped in the corresponding table before the outer merge; -1 disables the offset"}},
				{Kind: "list|nil", Label: "perTableLimit", Description: "optional per-table limits; nil disables all per-table limits", Element: &scm.TypeDescriptor{Kind: "int", Label: "limit", Description: "maximum rows retained from the corresponding table before the outer merge; -1 disables the limit"}},
				{Kind: "number", Label: "limitPartitionCols", Description: "number of leading sort columns forming partition key"},
				{Kind: "number", Label: "offset", Description: "number of items to skip (global)"},
				{Kind: "number", Label: "limit", Description: "max number of items to read (global; -1 = unlimited)"},
				{Kind: "list", Label: "mapColumns", Description: "map column lists, one per table", Element: &scm.TypeDescriptor{Kind: "list", Label: "table map columns", Description: "columns supplied to the matching mapFns entry", Element: &scm.TypeDescriptor{Kind: "string", Label: "column", Description: "column name in the corresponding table"}}},
				{Kind: "list", Label: "mapFns", Description: "map lambdas, one per table", Element: rowCallback("table map", "mapper for the corresponding table and mapColumns entry", "any", "value inserted into the merged stream and passed to reduce")},
				reducer("reduce", "optional aggregation function over mapped values from the merged stream"),
				{Kind: "any", Label: "neutral", Description: "(optional) neutral element for reduce", Optional: true},
				{Kind: "bool", Label: "isOuter", Description: "(optional) if true, emit null row when no hits", Optional: true},
				{Kind: "any", Label: "notFoundValue", Description: "(optional) result for no hits when isOuter is false; defaults to neutral", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
		Optimize: optimizeScanOrderMulti,
	})
	declareScanJoinOrder(&en)
	scm.Declare(&en, &scm.Declaration{
		Name: "createdatabase",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			ignoreexists := len(a) > 1 && scm.ToBool(a[1])
			return scm.NewBool(CreateDatabase(scm.String(a[0]), ignoreexists))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a new database", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the new database"},
				{Kind: "bool", Label: "ignoreexists", Description: "if true, return false instead of throwing an error", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropdatabase",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifexists := len(a) > 1 && scm.ToBool(a[1])
			return scm.NewBool(DropDatabase(scm.String(a[0]), ifexists))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "drops a database", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "bool", Label: "ifexists", Description: "if true, don't throw an error if it doesn't exist", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "checktablemaintenance",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			operation := maintenanceOperation(scm.String(a[2]))
			requireTableMaintenance(scm.String(a[0]), scm.String(a[1]), operation)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "checks whether a user-initiated maintenance operation is allowed for a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema"},
				{Kind: "string", Label: "table"},
				{Kind: "string", Label: "operation"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "maintenance_capabilities",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) > 1 {
				return maintenanceCapabilitiesScmer(tableMaintenanceCapabilities(scm.String(a[0]), scm.String(a[1])))
			}
			return maintenanceCapabilitiesScmer(databaseMaintenanceCapabilities(scm.String(a[0])))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the server-side maintenance capabilities for a database or table",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema"},
				{Kind: "string", Label: "table", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createtable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifnotexists := len(a) > 4 && scm.ToBool(a[4])
			currentTx := scm.NewNil()
			if len(a) > 5 {
				currentTx = a[5]
			}
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			db.ensureLoaded()
			tblName := scm.String(a[1])

			// Software contract:
			// createtable(..., ifnotexists=true) is the hot-path guard used by
			// planner-generated keytable/prejoin initializers. When the table is
			// already present, the fast path must stay extremely cheap:
			// - no schema save
			// - no table-definition rebuild
			// - no options/columns parsing
			//
			// We still confirm under schemalock after the optimistic probe so the
			// result stays race-free under concurrent creators.
			if ifnotexists {
				if existing := db.tables.Get(tblName); existing != nil {
					if existing.OnInit != nil || existing.onInitComplete {
						// This also reruns persisted oninit for data-empty MEMORY/CACHE
						// tables after restart. The local barrier affects no other table.
						existing.awaitCreationInitialization(currentTx)
						atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
						return scm.NewBool(false)
					}
					// Legacy CACHE/MEMORY schemas did not persist oninit. Parse this
					// invocation so its current closed callback can repair the table.
				}
			}

			// parse options only after the fast existing-table probe
			options := mustScmerSlice(a[3], "options")
			var autoIncrement uint64
			engine := Settings.DefaultEngine
			collation := ""
			charset := ""
			comment := ""
			oninit := scm.NewNil()
			for i := 0; i+1 < len(options); i += 2 {
				key := scm.String(options[i])
				val := options[i+1]
				switch key {
				case "engine":
					engine = scm.String(val)
				case "collation":
					collation = scm.String(val)
				case "charset":
					charset = scm.String(val)
				case "comment":
					comment = scm.String(val)
				case "auto_increment":
					nextAutoIncrement, _ := strconv.ParseUint(scm.String(val), 0, 64)
					if nextAutoIncrement > 0 {
						autoIncrement = nextAutoIncrement - 1
					}
				case "oninit":
					oninit = val
				default:
					panic("unknown option: " + key)
				}
			}
			pm := parsePersistencyMode(engine)

			newTable := new(table)
			newTable.schema = db
			newTable.Name = tblName
			newTable.PersistencyMode = pm
			newTable.ShardMode = ShardModeFree
			newTable.lastAccessed = uint64(time.Now().UnixNano())
			newTable.Shards = make([]*storageShard, 1)
			newTable.Shards[0] = NewShard(newTable)
			newTable.publishTopologyLocked()
			newTable.Collation = collation
			newTable.Charset = charset
			newTable.Comment = comment
			newTable.Auto_increment = autoIncrement
			if !oninit.IsNil() {
				closedOnInit := scm.CloseProcedure(oninit)
				newTable.OnInit = &closedOnInit
			}

			for _, coldef := range mustScmerSlice(a[2], "columns") {
				def := mustScmerSlice(coldef, "column definition")
				if len(def) == 0 {
					continue
				}
				head := scm.String(def[0])
				switch head {
				case "unique":
					cols := scmerSliceToStrings(mustScmerSlice(def[2], "unique columns"))
					newTable.Unique = append(newTable.Unique, uniqueKey{scm.String(def[1]), cols})
				case "foreign":
					cols1 := scmerSliceToStrings(mustScmerSlice(def[2], "foreign cols1"))
					cols2 := scmerSliceToStrings(mustScmerSlice(def[4], "foreign cols2"))
					var updatemode foreignKeyMode
					if len(def) > 5 {
						updatemode = getForeignKeyMode(def[5])
					}
					var deletemode foreignKeyMode
					if len(def) > 6 {
						deletemode = getForeignKeyMode(def[6])
					}
					newTable.Foreign = append(newTable.Foreign, foreignKey{
						Id:         scm.String(def[1]),
						Tbl1:       newTable.Name,
						Cols1:      cols1,
						Tbl2:       scm.String(def[3]),
						Cols2:      cols2,
						Updatemode: updatemode,
						Deletemode: deletemode,
					})
				case "column":
					colname := scm.String(def[1])
					typename := scm.String(def[2])
					dimVals := mustScmerSlice(def[3], "column dimensions")
					dimensions := make([]int, len(dimVals))
					for i, d := range dimVals {
						dimensions[i] = scm.ToInt(d)
					}
					typeparams := mustScmerSlice(def[4], "column typeparams")
					if _, ok := newTable.createColumnLocked(colname, typename, dimensions, typeparams); !ok {
						panic("column " + newTable.Name + "." + colname + " already exists")
					}
				default:
					panic("unknown column definition: " + head)
				}
			}
			newTable.publishShowColumnsSnapshot()

			db.schemalock.Lock()
			existing := db.tables.Get(tblName)
			if existing != nil {
				// Keep the hot ifnotexists path free of schema saves. Planner-created
				// helper tables deliberately re-issue createtable on every query; if
				// the table already exists, "created=false" is the only signal the
				// caller needs to skip collect/materialization work.
				atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
				if existing.OnInit == nil && !oninit.IsNil() {
					closedOnInit := scm.CloseProcedure(oninit)
					existing.OnInit = &closedOnInit
					db.saveLockedAndUnlock(schemaSaveBuffered)
				} else {
					db.schemalock.Unlock()
				}
				// The competing creator may have published the table after our
				// optimistic probe. Its table-local barrier includes oninit.
				existing.awaitCreationInitialization(currentTx)
				if !ifnotexists {
					panic("Table " + tblName + " already exists")
				}
				return scm.NewBool(false)
			}

			// Lock before publication. Every if-not-exists observer therefore waits
			// until oninit and registered create-table triggers have both completed.
			newTable.creationMu.Lock()
			if prev := db.tables.Set(newTable); prev != nil {
				newTable.creationMu.Unlock()
				db.schemalock.Unlock()
				panic("Table " + tblName + " already exists")
			}

			for _, fk := range newTable.Foreign {
				if t2 := newTable.schema.GetTable(fk.Tbl2); t2 != nil {
					t2.Foreign = append(t2.Foreign, fk)
					installFKTriggers(newTable.schema, newTable, t2, fk)
				}
			}
			// add constraints that are added onto us (forward-declared FKs)
			for _, t2 := range newTable.schema.tables.GetAll() {
				if t2 != newTable {
					for _, foreign := range t2.Foreign {
						if foreign.Tbl2 == newTable.Name {
							newTable.Foreign = append(newTable.Foreign, foreign)
							installFKTriggers(newTable.schema, t2, newTable, foreign)
						}
					}
				}
			}
			mode := newTable.schemaSaveMode()
			if newTable.isEphemeralQueryTable() {
				mode = schemaSaveBuffered
			}
			db.saveLockedAndUnlock(mode)
			registerCreatedTable(newTable)
			func() {
				defer func() {
					// Publish the shared outcome before releasing concurrent creators.
					newTable.creationPanic = recover()
					newTable.creationMu.Unlock()
				}()
				if !oninit.IsNil() {
					scm.Apply(oninit, currentTx)
				}
				newTable.onInitComplete = true
				executeRegisteredCreateTableTriggers(newTable, nil)
			}()
			if newTable.creationPanic != nil {
				panic(newTable.creationPanic)
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a table, runs its oninit option and registered after-create-table lifecycle triggers synchronously, and returns only after initialization completes; concurrent if-not-exists callers wait for that same completion", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the existing database that will contain the table"},
				{Kind: "string", Label: "table", Description: "name of the table to create"},
				{Kind: "list", Label: "cols", Description: "column and constraint definitions", Element: &scm.TypeDescriptor{Kind: "list", Label: "definition", Description: "one of (\"column\" name type dimensions typeparams), (\"unique\" name columns), or (\"foreign\" name local_columns referenced_table referenced_columns update_mode delete_mode). Column lists contain strings; foreign-key modes are restrict, cascade, or set null. A column definition's dimensions contains integers and its typeparams uses the same fields documented by createcolumn options"}},
				tableOptions,
				{Kind: "bool", Label: "ifnotexists", Description: "when true, return false instead of failing if the table exists; if another caller is still creating it, wait for that caller's after-create-table initialization before returning false", Optional: true},
				{Kind: "any", Label: "tx", Description: "explicit transaction context for oninit", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool", Description: "true when this call created and initialized the table, false when ifnotexists reused an initialized table"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createcolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])

			// normal column
			colname := scm.String(a[1])
			typename := scm.String(a[2])
			dimensionsVals := mustScmerSlice(a[3], "dimensions")
			dimensions := make([]int, len(dimensionsVals))
			for i, d := range dimensionsVals {
				dimensions[i] = scm.ToInt(d)
			}
			typeparams := mustScmerSlice(a[4], "typeparams")

			// ORC column: sortcols in options signals ordered-reduce computed column.
			// Extract ORC params from the options assoc list.
			var orcSortCols []string
			var orcSortDirs []bool
			var orcPartCount int
			var orcFilterCols []string
			var orcMapCols []string
			var orcFilterFn, orcMapFn, orcReduceFn, orcReduceInit scm.Scmer
			for i := 0; i+1 < len(typeparams); i += 2 {
				key := scm.String(typeparams[i])
				val := typeparams[i+1]
				switch key {
				case "sortcols":
					orcSortCols = scmerSliceToStrings(mustScmerSlice(val, "sortcols"))
				case "sortdirs":
					dirs := mustScmerSlice(val, "sortdirs")
					orcSortDirs = make([]bool, len(dirs))
					for j, d := range dirs {
						orcSortDirs[j] = scm.ToBool(d)
					}
				case "partitioncount":
					orcPartCount = scm.ToInt(val)
				case "filtercols":
					orcFilterCols = scmerSliceToStrings(mustScmerSlice(val, "filtercols"))
				case "filter":
					orcFilterFn = val
				case "mapcols":
					orcMapCols = scmerSliceToStrings(mustScmerSlice(val, "mapcols"))
				case "mapfn":
					orcMapFn = val
				case "reducefn":
					orcReduceFn = val
				case "reduceinit":
					orcReduceInit = val
				}
			}
			t.ddlMu.Lock()
			defer t.ddlMu.Unlock()
			created := t.createColumnDDLLocked(colname, typename, dimensions, typeparams)

			// Software contract:
			// createcolumn is the table-local DDL entrypoint for both "create a new
			// physical column" and "upgrade/configure an existing temp/base column".
			//
			// Planner/cache contract:
			// 1. Query plans may predeclare canonical temp columns and later call
			//    createcolumn again with the real computor/ORC metadata.
			// 2. Reissuing createcolumn for the SAME canonical temp column must be
			//    idempotent: if the cache/proxy is already valid, the call must not
			//    eagerly recompute or destroy the cached values.
			// 3. "Always materialize, but correctly" means the runtime path may reuse
			//    an already-populated temp column; it must not silently fall back to a
			//    throwaway one-shot computation because the column already exists.
			// 4. filtercols/filter define an ordered column's input domain. The
			//    planner therefore includes the predicate recipe in its canonical
			//    temp-column name; this entrypoint must preserve that identity.
			if len(orcSortCols) > 0 {
				t.computeOrderedColumnDDLLocked(colname, orcSortCols, orcSortDirs, orcPartCount, orcFilterCols, orcFilterFn, orcMapCols, orcMapFn, orcReduceFn, orcReduceInit)
				return scm.NewBool(true)
			}

			// Regular per-row computed column.
			if len(a) > 6 && !a[6].IsNil() {
				paramNames := scmerSliceToStrings(mustScmerSlice(a[5], "computor param names"))
				// extract filter from options
				var filterCols []string
				var filter scm.Scmer
				for i := 0; i < len(typeparams); i += 2 {
					key := scm.String(typeparams[i])
					if key == "filtercols" {
						filterCols = scmerSliceToStrings(mustScmerSlice(typeparams[i+1], "filter column names"))
					} else if key == "filter" {
						filter = typeparams[i+1]
					}
				}
				var currentTx *TxContext
				if len(a) > 7 {
					currentTx = scmerToTxContext(a[7])
				}
				t.computeColumnDDLLocked(colname, paramNames, a[6], filterCols, filter, currentTx)
				return scm.NewBool(true)
			}

			return scm.NewBool(created)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a new column in table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "colname", Description: "name of the new column"},
				{Kind: "string", Label: "type", Description: "name of the basetype"},
				{Kind: "list", Label: "dimensions", Description: "dimensions of the type, for example precision and scale for decimal", Element: &scm.TypeDescriptor{Kind: "int", Label: "dimension", Description: "one type-specific dimension"}},
				columnOptions,
				func() *scm.TypeDescriptor {
					value := columnList("computorCols", "columns passed to computor in this order")
					value.Optional = true
					return value
				}(),
				func() *scm.TypeDescriptor {
					value := rowCallback("computor", "lambda expression that computes this column from the values selected by computorCols", "any", "computed column value")
					value.Optional = true
					return value
				}(),
				{Kind: "any", Label: "tx", Description: "explicit transaction used while materializing computed values", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createkey",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			var currentTx *TxContext
			if len(a) > 4 {
				currentTx = scmerToTxContext(a[4])
			}

			if !scm.ToBool(a[2]) {
				return scm.NewBool(true)
			}

			cols := scmerSliceToStrings(mustScmerSlice(a[3], "unique columns"))
			name := scm.String(a[1])
			requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)

			// SQL DDL runs with a query session. Its exclusive table lock closes the
			// race with writers which selected the no-UNIQUE insert path before the
			// metadata was published. Boot-time catalog creation is single-threaded
			// and therefore does not require a user-level lock.
			unlockTable := func() {}
			if ss := SessionStateFromTx(currentTx); ss != nil {
				unlockTable = acquireTableLock(t.schema.Name, t.Name, true, false, ss, querySeqFromTx(currentTx))
			}
			defer unlockTable()

			// Validate under the table-local schema lock, but do not hold the
			// database-wide catalog lock while scanning table data.
			alreadyExists, hasDuplicates := func() (bool, bool) {
				t.ddlMu.Lock()
				defer t.ddlMu.Unlock()
				for _, u := range t.Unique {
					if strings.EqualFold(u.Id, name) {
						return true, false
					}
				}
				return false, t.hasDuplicateUniqueValues(cols, currentTx)
			}()
			if alreadyExists {
				return scm.NewBool(false)
			}
			if hasDuplicates {
				panic(sqldb.NewSQLError1(1062, "23000", "Duplicate entry in table %s prevents unique key %s", t.Name, name))
			}

			// Publication follows the documented database -> table DDL lock order.
			// Recheck the name because another internal DDL operation may have
			// published metadata between validation and catalog publication.
			t.schema.schemalock.Lock()
			t.ddlMu.Lock()
			defer t.ddlMu.Unlock()
			for _, u := range t.Unique {
				if strings.EqualFold(u.Id, name) {
					t.schema.schemalock.Unlock()
					return scm.NewBool(false)
				}
			}
			t.Unique = append(t.Unique, uniqueKey{name, cols})
			t.publishShowColumnsSnapshot()
			t.schema.saveLockedAndUnlock(t.schemaSaveMode())

			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a new key on a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "keyname", Description: "name of the new key"},
				{Kind: "bool", Label: "unique", Description: "whether the key is unique"},
				{Kind: "list", Label: "columns", Description: "list of columns to include"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropkey",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			name := scm.String(a[1])
			var currentTx *TxContext
			if len(a) > 2 {
				currentTx = scmerToTxContext(a[2])
			}
			requireTableMaintenance(t.schema.Name, t.Name, maintenanceAlter)

			unlockTable := func() {}
			if ss := SessionStateFromTx(currentTx); ss != nil {
				unlockTable = acquireTableLock(t.schema.Name, t.Name, true, false, ss, querySeqFromTx(currentTx))
			}
			defer unlockTable()
			t.schema.schemalock.Lock()
			t.ddlMu.Lock()
			defer t.ddlMu.Unlock()
			for i, key := range t.Unique {
				if strings.EqualFold(key.Id, name) {
					t.Unique = append(t.Unique[:i], t.Unique[i+1:]...)
					t.publishShowColumnsSnapshot()
					t.schema.saveLockedAndUnlock(t.schemaSaveMode())
					return scm.NewBool(true)
				}
			}
			t.schema.schemalock.Unlock()
			return scm.NewBool(false)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "drops a named unique key from a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "keyname", Description: "name of the unique key"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createforeignkey",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t1 := TableFromScmer(a[0])
			id := scm.String(a[1])
			cols1 := scmerSliceToStrings(mustScmerSlice(a[2], "foreign cols1"))
			t2 := TableFromScmer(a[3])
			cols2 := scmerSliceToStrings(mustScmerSlice(a[4], "foreign cols2"))

			db := t1.schema
			db.schemalock.Lock()
			for _, u := range t1.Foreign {
				if u.Id == id {
					db.schemalock.Unlock()
					return scm.NewBool(false)
				}
			}

			k := foreignKey{id, t1.Name, cols1, t2.Name, cols2, getForeignKeyMode(a[5]), getForeignKeyMode(a[6])}
			t1.Foreign = append(t1.Foreign, k)
			t2.Foreign = append(t2.Foreign, k)

			// auto-generate system triggers for FK enforcement
			installFKTriggers(db, t1, t2, k)

			db.saveLockedAndUnlock(schemaSaveModeForDurability(t1.PersistencyMode == Safe || t2.PersistencyMode == Safe))

			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a new foreign key on a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table1"},
				{Kind: "string", Label: "keyname", Description: "name of the new key"},
				{Kind: "list", Label: "columns1", Description: "list of columns to include"},
				{Kind: "table", Label: "table2"},
				{Kind: "list", Label: "columns2", Description: "list of columns to include"},
				{Kind: "string", Label: "updatemode", Description: "restrict|cascade|set null"},
				{Kind: "string", Label: "deletemode", Description: "restrict|cascade|set null"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "shardcolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			numPartitions := 0
			if len(a) > 2 {
				numPartitions = scm.ToInt(a[2])
			}
			if numPartitions == 0 {
				// check if that paritition dimension already exists
				if t.ShardMode == ShardModePartition {
					for _, sd := range t.PDimensions {
						if sd.Column == scm.String(a[1]) {
							return scm.NewSlice(sd.Pivots) // found the column in partition schema: return exactly the same pivots as we found already
						}
					}
				}
				// otherwise: no partition schema yet: find out the best number of partitions
				// normally, we put ~60,000 items per shard, but to parallelize grouping, we should do less?
				numPartitions = int(1 + ((2 * t.Count()) / Settings.ShardSize))
			}
			// calculate them anew
			return scm.NewSlice(t.NewShardDimension(scm.String(a[1]), numPartitions).Pivots)

		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "tells us how it would partition a column according to their values. Returns a list of pivot elements.",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "colname", Description: "name of the column"},
				{Kind: "number", Label: "numpartitions", Description: "number of partitions; optional. leave 0 if you want to detect the partiton number automatically or copy the partition schema of the table", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "partitiontable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			cols := normalizePartitionDataset(a[1])
			// Contract: an empty partition schema carries no physical partitioning
			// information. Keep the table in free-shard mode instead of forcing a
			// degenerate "partitioned with one shard" rebuild for scratch/key tables.
			if len(cols) == 0 {
				return scm.NewBool(false)
			}
			if t.ShardMode == ShardModeFree {
				// apply partitioning schema
				ps := make([]shardDimension, len(cols)/2)
				for i := 0; i < len(ps); i++ {
					ps[i].Column = scm.String(cols[2*i])
					ps[i].Pivots = mustScmerSlice(cols[2*i+1], "partition pivots")
					ps[i].NumPartitions = len(ps[i].Pivots) + 1
				}
				trimmed := make([]shardDimension, 0, len(ps))
				for _, dim := range ps {
					if dim.NumPartitions > 1 {
						trimmed = append(trimmed, dim)
					}
				}
				ps = trimmed
				if len(ps) == 0 {
					return scm.NewBool(false)
				}
				if len(ps) > Settings.PartitionMaxDimensions {
					ps = ps[:Settings.PartitionMaxDimensions]
				}
				if !t.beginManualRepartition() {
					return scm.NewBool(false)
				}
				t.repartition(ps) // perform repartitioning immediately
				return scm.NewBool(true)
			} else {
				// early exit if all requested columns are already partitioned
				allPresent := true
				for i := 0; i < len(cols)/2; i++ {
					colName := scm.String(cols[2*i])
					found := false
					for _, dim := range t.PDimensions {
						if dim.Column == colName {
							found = true
							break
						}
					}
					if !found {
						allPresent = false
						break
					}
				}
				if allPresent {
					return scm.NewBool(false)
				}
				// increase partitioning scores
				for i, c := range t.Columns {
					if pivots, ok := cols.Get(c.Name); ok {
						// that column is in the parititoning schema -> increase score
						t.Columns[i].PartitioningScore = c.PartitioningScore + len(mustScmerSlice(pivots, "partition pivots"))
					}
				}
				return scm.NewBool(false)
			}
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "suggests a partition scheme for a table. If the table has no partition scheme yet, it will immediately apply that scheme and return true. If the table already has a partition scheme, it will alter the partitioning score such that the partitioning scheme is considered in the next repartitioning and return false.",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "list", Label: "columns", Description: "associative list of string -> list representing column name -> pivots. You can compute pivots by (shardcolumn ...)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "altertable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			db := t.schema
			operation := scm.String(a[1])

			switch operation {
			case "drop":
				return scm.NewBool(t.DropColumn(scm.String(a[2])))
			case "drop_if_exists":
				return scm.NewBool(t.DropColumnIfExists(scm.String(a[2])))
			case "engine":
				newMode := parsePersistencyMode(scm.String(a[2]))
				oldMode := t.PersistencyMode
				if oldMode == newMode {
					return scm.NewBool(true) // no-op
				}
				requireTableMaintenance(db.Name, t.Name, maintenanceChangeEngine)

				t.mu.Lock()

				if (oldMode == Memory || oldMode == Cache) && newMode != Memory && newMode != Cache {
					// Memory → Persisted: ensure all columns are loaded
					// while PersistencyMode is still Memory (so they get
					// initialized as StorageSparse instead of reading
					// non-existent disk files). Then switch mode and
					// rebuild to flush everything to disk.
					shards := t.ActiveShards()
					for _, s := range shards {
						s.mu.Lock()
						for _, col := range t.Columns {
							s.ensureColumnLoaded(col.Name, true)
						}
						s.mu.Unlock()
					}
					t.PersistencyMode = newMode
					t.mu.Unlock()
					for i, s := range shards {
						shards[i] = s.rebuild(true)
					}
				} else {
					t.PersistencyMode = newMode
					// All other transitions can be done in-place.
					for _, s := range t.ActiveShards() {
						s.mu.Lock()
						transitionShardEngine(s, oldMode, newMode)
						s.mu.Unlock()
					}
					t.mu.Unlock()
				}

				db.save()
				return scm.NewBool(true)
			case "owner":
				return scm.NewBool(false) // ignore
			case "auto_increment":
				requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
				next := uint64(scm.ToInt(a[2]))
				if next > 0 {
					t.mu.Lock()
					if next-1 > t.Auto_increment {
						t.Auto_increment = next - 1
					}
					t.mu.Unlock()
					db.save()
				}
				return scm.NewBool(true)
			default:
				panic("unimplemented alter table operation: " + operation)
			}
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "alters a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "operation", Description: "one of owner|drop|engine|collation|auto_increment"},
				{Kind: "any", Label: "parameter", Description: "name of the column to drop or value of the parameter"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "altercolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			db := t.schema
			requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
			for i, c := range t.Columns {
				if c.Name == scm.String(a[1]) {
					switch scm.String(a[2]) {
					case "drop":
						ok := t.DropColumn(scm.String(a[1]))
						db.save()
						return scm.NewBool(ok)
					case "auto_increment":
						ai := scm.ToInt(a[3])
						if ai > 1 {
							t.mu.Lock()
							t.Auto_increment = uint64(ai)
							t.mu.Unlock()
							db.save()
							return scm.NewBool(true)
						}
						t.Columns[i].AutoIncrement = scm.ToBool(a[3])
						t.publishShowColumnsSnapshot()
						db.save()
						return scm.NewBool(true)
					default:
						ok := t.Columns[i].Alter(scm.String(a[2]), a[3])
						t.publishShowColumnsSnapshot()
						db.save()
						return scm.NewBool(scm.ToBool(ok))
					}
				}
			}
			panic("column " + t.schema.Name + "." + t.Name + "." + scm.String(a[1]) + " does not exist")
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "alters a column",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "name of the column"},
				{Kind: "string", Label: "operation", Description: "one of drop|type|collation|auto_increment|comment"},
				{Kind: "any", Label: "parameter", Description: "name of the column to drop or value of the parameter"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "droptable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifexists := len(a) > 2 && scm.ToBool(a[2])
			DropTable(scm.String(a[0]), scm.String(a[1]), ifexists)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "removes a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema"},
				{Kind: "string", Label: "table"},
				{Kind: "bool", Label: "ifexists", Description: "if true, don't throw an error if it already exists", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropcolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewBool(t.DropColumn(scm.String(a[1])))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "drops a column from a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "name of the column to drop"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "migratedropcolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewBool(t.dropColumnForMigration(scm.String(a[1])))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "drops a legacy system column during startup migration", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "legacy column name"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "invalidatecolumn",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			colName := scm.String(a[1])
			var currentTx *TxContext
			if len(a) > 2 {
				currentTx = scmerToTxContext(a[2])
			}
			invalidateComputedColumn(t, colName, currentTx)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "marks all values of a computed column as stale",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "name of the computed column"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "invalidateorc",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			// Accept both a single value and a list of sort key values
			var sortKeys []scm.Scmer
			if a[2].IsSlice() {
				sortKeys = a[2].Slice()
			} else {
				sortKeys = []scm.Scmer{a[2]}
			}
			t.invalidateORCFromSortKey(scm.String(a[1]), sortKeys)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "invalidates ORC column rows from a sort key onwards via validMask scan",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "name of the ORC column"},
				{Kind: "list", Label: "sortkeys", Description: "composite sort key values from which to invalidate"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "register_keytable_cleanup",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			if a[0].IsNil() {
				return scm.NewBool(false)
			}
			baseTable := TableFromScmer(a[0])
			ktTable := TableFromScmer(a[1])
			ktSchema := ktTable.schema.Name
			ktName := ktTable.Name
			tblvar := scm.String(a[2])
			pairs := a[3].Slice()

			// Extract column name pairs
			var baseCols, ktCols []string
			for _, p := range pairs {
				pp := p.Slice()
				baseCols = append(baseCols, scm.String(pp[0]))
				ktCols = append(ktCols, scm.String(pp[1]))
			}

			baseSchema := baseTable.schema.Name
			type computedKey struct {
				computor scm.Scmer
				inputs   []string
			}
			computedKeys := make(map[string]computedKey)
			baseTable.ddlMu.RLock()
			for _, col := range baseTable.Columns {
				if col.Computor.IsNil() || len(col.OrcSortCols) > 0 {
					continue
				}
				computedKeys[col.Name] = computedKey{
					computor: col.Computor,
					inputs:   append([]string(nil), col.ComputorInputCols...),
				}
			}
			baseTable.ddlMu.RUnlock()

			// Helper: build (and (equal? x1 y1) (equal? x2 y2) ...) or just (equal? x y) for single key
			buildAndEquals := func(xs, ys []scm.Scmer) scm.Scmer {
				if len(xs) == 1 {
					return scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), xs[0], ys[0]})
				}
				parts := make([]scm.Scmer, 1+len(xs))
				parts[0] = scm.NewSymbol("and")
				for i := range xs {
					parts[1+i] = scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), xs[i], ys[i]})
				}
				return scm.NewSlice(parts)
			}

			// Helper: build scan filter lambda params as tblvar.col symbols
			scanFilterParams := func(prefix string, cols []string) scm.Scmer {
				params := make([]scm.Scmer, len(cols))
				for i, col := range cols {
					params[i] = scm.NewSymbol(prefix + "." + col)
				}
				return scm.NewSlice(params)
			}

			// Helper: build scan filter column list (list "col1" "col2" ...)
			scanFilterCols := func(cols []string) scm.Scmer {
				elems := make([]scm.Scmer, 1+len(cols))
				elems[0] = scm.NewSymbol("list")
				for i, col := range cols {
					elems[1+i] = scm.NewString(col)
				}
				return scm.NewSlice(elems)
			}

			// Helper: build symbols for scan param references
			scanParamSyms := func(prefix string, cols []string) []scm.Scmer {
				syms := make([]scm.Scmer, len(cols))
				for i, col := range cols {
					syms[i] = scm.NewSymbol(prefix + "." + col)
				}
				return syms
			}

			// DELETE captures a complete OLD row, including computed values. INSERT
			// NEW rows only contain stored columns, so evaluate missing computed keys
			// from their physical inputs without rescanning the mutated base row.
			valueFromDict := func(sym, col string) scm.Scmer {
				computed, ok := computedKeys[col]
				if !ok || sym == "OLD" {
					return fkGetAssocExpr(sym, col)
				}
				args := make([]scm.Scmer, 1, 1+len(computed.inputs))
				args[0] = scm.NewSymbol("list")
				for _, input := range computed.inputs {
					args = append(args, fkGetAssocExpr(sym, input))
				}
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("apply"),
					computed.computor,
					scm.NewSlice(args),
				})
			}

			// Helper: build row-dictionary value expressions for columns.
			getAssocs := func(sym string, cols []string) []scm.Scmer {
				result := make([]scm.Scmer, len(cols))
				for i, col := range cols {
					result[i] = valueFromDict(sym, col)
				}
				return result
			}

			// Build count-scan: (scan tx (table base_schema base_table) (list base_cols...) (lambda (tblvar.col...) (and (equal? tblvar.col (get_assoc OLD "col")) ...)) () (lambda () 1) + 0 nil)
			buildCountScan := func(dictSym string) scm.Scmer {
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("scan"),
					scm.NewSymbol("tx"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(baseSchema), scm.NewString(baseTable.Name)}),
					scanFilterCols(baseCols),
					scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("lambda"), scanFilterParams(tblvar, baseCols)},
						buildAndEquals(scanParamSyms(tblvar, baseCols), getAssocs(dictSym, baseCols)))),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{}), scm.NewInt(1)}),
					scm.NewSymbol("+"), scm.NewInt(0), scm.NewNil(),
				})
			}

			// Build delete-scan: (scan tx (table kt_schema kt_name) (list kt_cols...) (lambda (kt.col...) (and (equal? kt.col (get_assoc OLD "base_col")) ...)) (list "$update") (lambda ($update) ($update)) + 0 nil)
			buildDeleteScan := func(dictSym string) scm.Scmer {
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("scan"),
					scm.NewSymbol("tx"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(ktSchema), scm.NewString(ktName)}),
					scanFilterCols(ktCols),
					scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("lambda"), scanFilterParams(ktName, ktCols)},
						buildAndEquals(scanParamSyms(ktName, ktCols), getAssocs(dictSym, baseCols)))),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"), scm.NewString("$update")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("$update")}),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("$update")})}),
					scm.NewSymbol("+"), scm.NewInt(0), scm.NewNil(),
				})
			}

			// Build insert: (insert kt_schema kt_name (list kt_cols...) (list (list vals...)) (list) (lambda () true) true)
			buildInsert := func(dictSym string) scm.Scmer {
				values := append([]scm.Scmer{scm.NewSymbol("list")}, getAssocs(dictSym, baseCols)...)
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("insert"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(ktSchema), scm.NewString(ktName)}),
					scanFilterCols(ktCols),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"),
						scm.NewSlice(values)}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{}), scm.NewBool(true)}),
					scm.NewBool(true),
				})
			}

			// Keytable membership is a set, not a multiset. Always attempt an
			// idempotent INSERT IGNORE on AFTER INSERT / UPDATE instead of trying
			// to detect the first row via COUNT==1: a single logical source-row
			// trigger may insert multiple base rows of the same group in one batch,
			// so all rows are already visible when the first AFTER INSERT trigger
			// fires. COUNT==1 would then miss the new group entirely.
			buildInsertIfMissing := func(dictSym string) scm.Scmer {
				return buildInsert(dictSym)
			}

			// AfterDelete body: if count=0 then delete from keytable
			deleteBody := scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), scm.NewInt(0), buildCountScan("OLD")}),
				buildDeleteScan("OLD"),
			})

			// AfterInsert body: only add a key when this INSERT created the first
			// row for the group. Existing groups already have their keytable row.
			insertBody := buildInsertIfMissing("NEW")

			// AfterUpdate body: if key changed, clean up old + insert new
			keyChangedCheck := buildAndEquals(getAssocs("OLD", baseCols), getAssocs("NEW", baseCols))
			updateBody := scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("not"), keyChangedCheck}),
				scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("begin"),
					scm.NewSlice([]scm.Scmer{
						scm.NewSymbol("if"),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), scm.NewInt(0), buildCountScan("OLD")}),
						buildDeleteScan("OLD"),
					}),
					buildInsertIfMissing("NEW"),
				}),
			})

			// Register DML triggers with idempotency
			triggerDefs := []struct {
				timing TriggerTiming
				body   scm.Scmer
			}{
				{AfterDelete, deleteBody},
				{AfterInsert, insertBody},
				{AfterUpdate, updateBody},
			}
			for _, td := range triggerDefs {
				triggerName := ".kt_cleanup:" + ktName + "|" + baseTable.Name + "|" + td.timing.String()
				exists := false
				for _, tr := range baseTable.Triggers {
					if tr.Name == triggerName {
						exists = true
						break
					}
				}
				if exists {
					baseTable.SetTriggerTarget(triggerName, ktTable.acquireCacheUseForTrigger, ktTable.releaseCacheUse)
					continue
				}
				baseTable.AddTrigger(TriggerDescription{
					Name:     triggerName,
					Timing:   td.timing,
					IsSystem: true,
					Priority: 90, // run before invalidatecolumn (100) so keys are current when values recompute
					Func:     buildFKProc(td.body),
					Acquire:  ktTable.acquireCacheUseForTrigger,
					Release:  ktTable.releaseCacheUse,
				})
			}
			// Lifecycle cleanup: when the base table is dropped/shape-changed, the keytable
			// must be dropped as well, otherwise stale keytables can be reused by a later
			// table recreation with the same name and cause cross-suite flakes.
			dropBody := scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("droptable"),
				scm.NewString(ktSchema),
				scm.NewString(ktName),
				scm.NewBool(true),
			})
			for _, timing := range []TriggerTiming{AfterDropTable, AfterDropColumn} {
				triggerName := ".kt_cleanup:" + ktName + "|" + baseTable.Name + "|" + timing.String()
				exists := false
				for _, tr := range baseTable.Triggers {
					if tr.Name == triggerName {
						exists = true
						break
					}
				}
				if exists {
					baseTable.SetTriggerTarget(triggerName, ktTable.acquireCacheUseForTrigger, ktTable.releaseCacheUse)
					continue
				}
				baseTable.AddTrigger(TriggerDescription{
					Name:     triggerName,
					Timing:   timing,
					IsSystem: true,
					Priority: 90,
					Func:     buildFKProc(dropBody),
					Acquire:  ktTable.acquireCacheUseForTrigger,
					Release:  ktTable.releaseCacheUse,
				})
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "registers triggers on a base table to maintain keytable entries (insert/delete group keys)",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "base_table"},
				{Kind: "table", Label: "kt_table"},
				{Kind: "string", Label: "tblvar", Description: "table alias used in scan column prefixes"},
				{Kind: "list", Label: "key_pairs", Description: "list of (base_col kt_col) pairs"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "initialize_cache_table",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[0])
			tbl := TableFromScmer(a[1])
			// Client-query cache preparation carries its lock owner through the
			// transaction. An internal transactionless builder receives a local,
			// unregistered lock owner from initializeCache instead.
			ss := SessionStateFromTx(currentTx)
			initialized := tbl.initializeCache(ss, func(lockOwner *scm.SessionState) {
				sources := mustScmerSlice(a[2], "source tables")
				sourceTables := make([]*table, len(sources))
				for i, source := range sources {
					sourceTables[i] = TableFromScmer(source)
				}
				sort.Slice(sourceTables, func(i, j int) bool {
					left := sourceTables[i].schema.Name + "\x00" + sourceTables[i].Name
					right := sourceTables[j].schema.Name + "\x00" + sourceTables[j].Name
					return left < right
				})

				unlocks := make([]func(), 0, len(sourceTables))
				defer func() {
					for i := len(unlocks) - 1; i >= 0; i-- {
						unlocks[i]()
					}
				}()
				for _, source := range sourceTables {
					unlocks = append(unlocks, acquireTableLock(source.schema.Name, source.Name, false, true, lockOwner, querySeqFromTx(currentTx)))
				}
				// Install maintenance while source writes are blocked. Once the
				// locks are released, every later mutation observes the triggers.
				scm.Apply(a[3], a[0])
				scm.Apply(a[4], a[0])
				if len(a) > 5 {
					scm.Apply(a[5], a[0])
				}
			})
			return scm.NewBool(initialized)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "registers maintenance, locks source tables for a consistent snapshot, and runs a canonical planner-cache initializer exactly once", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "transaction", Description: "explicit transaction context carrying query-session ownership"},
				{Kind: "table", Label: "table"},
				{Kind: "list", Label: "source_tables"},
				{Kind: "func", Label: "register_maintenance", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "initializer", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "finalizer", Description: "optional finalizer run under the same source-table locks after initialization", Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}}, Return: &scm.TypeDescriptor{Kind: "any"}, Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "touch_keytable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			tbl := TableFromScmer(a[0])
			now := time.Now()
			nowNs := uint64(now.UnixNano())
			atomic.StoreUint64(&tbl.lastAccessed, nowNs)
			for _, c := range tbl.Columns {
				if c.IsTemp {
					atomic.StoreInt64(&c.lastAccessed, now.UnixNano())
				}
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "extends the lease on a keytable so CacheManager defers eviction",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "locktables",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			tx := scmerToTxContext(a[1])
			ss := SessionStateFromTx(tx)
			querySeq := querySeqFromTx(tx)
			if ss != nil {
				ss.ReleaseAllLocks() // LOCK TABLES implicitly releases prior locks
			}
			for _, item := range a[0].Slice() {
				triple := item.Slice()
				schema := scm.String(triple[0])
				tbl := scm.String(triple[1])
				write := triple[2].Bool()
				lockTable(schema, tbl, write, ss, querySeq)
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "acquires WRITE or READ user-level locks on a list of tables (LOCK TABLES); implicitly releases any previously held locks", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "list", Label: "locks", Description: "flat list of schema, table, write? triples"},
				{Kind: "any", Label: "tx", Description: "explicit request context"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "unlocktables",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			ss := SessionStateFromTx(scmerToTxContext(a[0]))
			if ss != nil {
				ss.ReleaseAllLocks()
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "releases all user-level table locks held by this session", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx", Description: "explicit request context"}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "get_fk_target",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			tbl := TableFromScmer(a[0])
			col := scm.String(a[1])
			for _, fk := range tbl.Foreign {
				if fk.Tbl1 == tbl.Name && len(fk.Cols1) == 1 && fk.Cols1[0] == col {
					return scm.NewSlice([]scm.Scmer{scm.NewString(fk.Tbl2), scm.NewString(fk.Cols2[0])})
				}
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "returns (ref_table ref_column) if a single-column FK exists for the given column, nil otherwise",
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "column", Description: "column name"},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "renametable",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			RenameTable(scm.String(a[0]), scm.String(a[1]), scm.String(a[2]))
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "renames a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "string", Label: "oldname", Description: "current name of the table"},
				{Kind: "string", Label: "newname", Description: "new name of the table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "insert",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			var onCollisionCols []string
			onCollision := scm.NewNil()
			if len(a) > 4 {
				onCollisionColsVals := mustScmerSlice(a[3], "onCollision columns")
				onCollisionCols = make([]string, len(onCollisionColsVals))
				for i, c := range onCollisionColsVals {
					onCollisionCols[i] = scm.String(c)
				}
				onCollision = a[4]
			}
			mergeNull := len(a) > 5 && scm.ToBool(a[5])
			// optional onInsertid callback
			var onFirst func(int64)
			if len(a) > 6 && !a[6].IsNil() {
				cb := a[6]
				var once sync.Once
				onFirst = func(id int64) {
					once.Do(func() { scm.Apply(cb, scm.NewInt(id)) })
				}
			}
			colsVals := mustScmerSlice(a[1], "column names")
			cols := make([]string, len(colsVals))
			for i, col := range colsVals {
				cols[i] = scm.String(col)
			}
			rowVals := mustScmerSlice(a[2], "dataset rows")
			rows := make([][]scm.Scmer, len(rowVals))
			for i, row := range rowVals {
				rows[i] = mustScmerSlice(row, "insert row")
			}
			var currentTx *TxContext
			if len(a) > 7 {
				currentTx = scmerToTxContext(a[7])
			}
			inserted := t.Insert(cols, rows, onCollisionCols, onCollision, mergeNull, onFirst, currentTx)
			return scm.NewInt(int64(inserted))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "inserts a new dataset into table and returns the number of successful items", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "list", Label: "columns", Description: "list of column names, e.g. '(\"ID\", \"value\")"},
				{Kind: "list", Label: "datasets", Description: "list of list of column values, e.g. '('(1 10) '(2 15))"},
				{Kind: "list", Label: "onCollisionCols", Description: "list of columns of the old dataset that have to be passed to onCollision. Can also request $update, $set:<computed-column>, or NEW.<insert-column>.", Optional: true},
				{Kind: "func", Label: "onCollision", Description: "function called for each collision. Its positional parameters are the values requested by onCollisionCols, in the same order. If omitted, collisions raise an error.", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "any", Label: "column values", Description: "one value for each onCollisionCols entry", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any", Label: "result"}},
				{Kind: "bool", Label: "mergeNull", Description: "if true, it will handle NULL values as equal according to SQL 2003's definition of DISTINCT (https://en.wikipedia.org/wiki/Null_(SQL)#When_two_nulls_are_equal:_grouping,_sorting,_and_some_set_operations)", Optional: true},
				{Kind: "func", Label: "onInsertid", Description: "called once with the first auto_increment id assigned for this INSERT", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "number", Label: "id", Description: "first assigned auto_increment id"}}, Return: &scm.TypeDescriptor{Kind: "any", Label: "result", Description: "ignored callback result"}},
				{Kind: "any", Label: "tx", Description: "explicit transaction context", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "number"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "stat",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				memTotal, memAvail := ReadMemInfo()
				processMem := ReadProcessRSS()
				cs := GlobalCache.Stat()
				nonEvictable := processMem - cs.CurrentMemory
				if nonEvictable < 0 {
					nonEvictable = 0
				}
				return scm.NewSlice([]scm.Scmer{
					scm.NewString("mem_available"), scm.NewInt(memAvail),
					scm.NewString("mem_total"), scm.NewInt(memTotal),
					scm.NewString("process_memory"), scm.NewInt(processMem),
					scm.NewString("evictable_memory"), scm.NewInt(cs.CurrentMemory),
					scm.NewString("non_evictable_process_memory"), scm.NewInt(nonEvictable),
					scm.NewString("shard_memory"), scm.NewInt(cs.CurrentMemory),
					scm.NewString("shard_budget"), scm.NewInt(cs.MemoryBudget),
					scm.NewString("persisted_memory"), scm.NewInt(cs.PersistedMemory),
					scm.NewString("persisted_budget"), scm.NewInt(cs.PersistedBudget),
					scm.NewString("cache_entry_count"), scm.NewInt(cs.CountByType[TypeCacheEntry]),
					scm.NewString("cache_entry_size"), scm.NewInt(cs.SizeByType[TypeCacheEntry]),
					scm.NewString("shard_column_size"), scm.NewInt(cs.SizeByType[TypeShard]),
					scm.NewString("index_size"), scm.NewInt(cs.SizeByType[TypeIndex]),
					scm.NewString("temp_column_size"), scm.NewInt(cs.SizeByType[TypeTempColumn]),
					scm.NewString("temp_column_count"), scm.NewInt(cs.CountByType[TypeTempColumn]),
					scm.NewString("temp_keytable_size"), scm.NewInt(cs.SizeByType[TypeTempKeytable]),
					scm.NewString("temp_keytable_count"), scm.NewInt(cs.CountByType[TypeTempKeytable]),
					scm.NewString("string_dictionary_size"), scm.NewInt(cs.SizeByType[TypeStringDict]),
					scm.NewString("string_dictionary_count"), scm.NewInt(cs.CountByType[TypeStringDict]),
				})
			} else if len(a) == 1 && a[0].IsCustom(TagTable) {
				return scm.NewString(TableFromScmer(a[0]).PrintMemUsage())
			} else if len(a) == 1 {
				return scm.NewString(GetDatabase(scm.String(a[0])).PrintMemUsage())
			} else if len(a) == 2 {
				return scm.NewString(GetDatabase(scm.String(a[0])).GetTable(scm.String(a[1])).PrintMemUsage())
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "return system statistics as assoc. process_memory is exact process RSS; evictable_memory and its per-owner size/count fields are disjoint estimated Go payload ownership; non_evictable_process_memory is the RSS remainder including allocator, runtime, stacks, and untracked shared overhead. shard_memory remains a compatibility alias for evictable_memory.\n(stat schema) and (stat schema tbl) return disjoint owner-payload estimates, not per-schema RSS.",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "(optional) database name for detailed string output", Optional: true},
				{Kind: "string", Label: "table", Description: "(optional) table name for detailed string output", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "totalmem",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			return scm.NewInt(totalMemoryBytes())
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Returns total physical memory in bytes (from /proc/meminfo)",
			Return: &scm.TypeDescriptor{Kind: "number"},
			Const:  true,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "resolve_column_name",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				return scm.NewNil()
			}
			t := db.GetTable(scm.String(a[1]))
			if t == nil {
				return scm.NewNil()
			}
			name, ok := t.ResolveColumnName(scm.String(a[2]), scm.ToBool(a[3]))
			if !ok {
				return scm.NewNil()
			}
			return scm.NewString(name)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "resolve a physical column name from immutable table metadata",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "database name"},
				{Kind: "string", Label: "table", Description: "table name"},
				{Kind: "string", Label: "column", Description: "column name"},
				{Kind: "bool", Label: "ignorecase", Description: "whether identifier case is ignored"},
			},
			Return: &scm.TypeDescriptor{Kind: "string|nil", Transfer: false},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "show",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			// table-based overloads: (show table) / (show recset) → columns,
			// (show table "statistics") / (show recset "statistics") → index stats, etc.
			if len(a) >= 1 && (a[0].IsCustom(TagTable) || a[0].IsCustom(TagRecSet)) {
				t := (*table)(nil)
				if a[0].IsCustom(TagRecSet) {
					t = RecSetFromScmer(a[0]).table
				} else {
					t = TableFromScmer(a[0])
				}
				if len(a) == 1 {
					return t.ShowColumns()
				}
				if len(a) == 2 {
					if a[1].IsString() && scm.String(a[1]) == "statistics" {
						return showBuildIndexRows(t.schema, t, false)
					}
					if a[1].IsString() && scm.String(a[1]) == "indexes" {
						return showBuildIndexRows(t.schema, t, true)
					}
					if a[1].IsBool() && a[1].Bool() {
						return showBuildMeta(t.schema, t)
					}
				}
				return t.ShowColumns()
			}
			if len(a) == 0 {
				// list databases
				dbs := databases.GetAll()
				result := make([]scm.Scmer, len(dbs))
				for i, db := range dbs {
					result[i] = scm.NewString(db.Name)
				}
				return scm.NewSlice(result)
			} else if len(a) == 1 {
				// list table names
				db := GetDatabase(scm.String(a[0]))
				if db == nil {
					return scm.NewNil() // use this to check if a database exists
				}
				return db.ShowTables()
			} else if len(a) == 2 {
				db := GetDatabase(scm.String(a[0]))
				if db == nil {
					panic("database " + scm.String(a[0]) + " does not exist")
				}
				// (show schema true) → full table listing
				if a[1].IsBool() && a[1].Bool() {
					db.ensureLoaded()
					tables := db.tables.GetAll()
					rows := make([]scm.Scmer, 0, len(tables))
					for _, t := range tables {
						if t.isHiddenFromShowTables() {
							continue
						}
						engine := showEngineStr(t)
						stats := t.statistics()
						rows = append(rows, scm.NewSlice([]scm.Scmer{
							scm.NewString("name"), scm.NewString(t.Name),
							scm.NewString("engine"), scm.NewString(engine),
							scm.NewString("row_count"), scm.NewInt(stats.rowCount),
							scm.NewString("size_bytes"), scm.NewInt(stats.sizeBytes),
							scm.NewString("collation"), scm.NewString(t.Collation),
							scm.NewString("comment"), scm.NewString(t.Comment),
						}))
					}
					return scm.NewSlice(rows)
				}
				// (show schema tbl) → column defs
				tableArg := a[1]
				if rows, ok := scmerSlice(tableArg); ok {
					return showColumnsForRows(rows)
				}
				t := db.GetTable(scm.String(tableArg))
				if t == nil {
					if len(scm.String(tableArg)) > 0 && scm.String(tableArg)[0] == '.' {
						// temp table does not exist yet - return empty schema
						return scm.NewSlice(nil)
					}
					panic("table " + scm.String(a[0]) + "." + scm.String(tableArg) + " does not exist")
				}
				return t.ShowColumns()
			} else if len(a) == 3 {
				db := GetDatabase(scm.String(a[0]))
				if db == nil {
					panic("database " + scm.String(a[0]) + " does not exist")
				}
				t := db.GetTable(scm.String(a[1]))
				if t == nil {
					panic("show3: table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
				}
				// (show schema tbl "statistics"|"indexes") → index metadata
				if a[2].IsString() && scm.String(a[2]) == "statistics" {
					return showBuildIndexRows(db, t, false)
				}
				if a[2].IsString() && scm.String(a[2]) == "indexes" {
					return showBuildIndexRows(db, t, true)
				}
				// (show schema tbl true) → full table info {columns, meta, shards}
				if a[2].IsBool() && a[2].Bool() {
					shards := t.ActiveShards()
					shardRows := make([]scm.Scmer, 0, len(shards))
					for i, s := range shards {
						shardRows = append(shardRows, showBuildShardRow(t, i, s))
					}
					// build trigger info
					triggerRows := make([]scm.Scmer, 0, len(t.Triggers))
					for _, tr := range t.Triggers {
						triggerRows = append(triggerRows, scm.NewSlice([]scm.Scmer{
							scm.NewString("name"), scm.NewString(tr.Name),
							scm.NewString("timing"), scm.NewString(string(tr.Timing)),
							scm.NewString("hidden"), scm.NewBool(tr.Hidden),
							scm.NewString("system"), scm.NewBool(tr.IsSystem),
							scm.NewString("priority"), scm.NewInt(int64(tr.Priority)),
						}))
					}
					return scm.NewSlice([]scm.Scmer{
						scm.NewString("columns"), t.ShowColumns(),
						scm.NewString("meta"), showBuildMeta(db, t),
						scm.NewString("shards"), scm.NewSlice(shardRows),
						scm.NewString("triggers"), scm.NewSlice(triggerRows),
					})
				}
				// (show schema tbl N) → shard N overview
				if a[2].IsInt() || a[2].IsFloat() {
					shards := t.ActiveShards()
					idx := int(scm.ToInt(a[2]))
					if idx < 0 || idx >= len(shards) {
						panic("shard index out of range")
					}
					return showBuildShardRow(t, idx, shards[idx])
				}
				panic("invalid call of show")
			} else if len(a) == 4 {
				// (show schema tbl N true) → full shard info with columns and indexes
				if (a[2].IsInt() || a[2].IsFloat()) && a[3].IsBool() && a[3].Bool() {
					db := GetDatabase(scm.String(a[0]))
					if db == nil {
						panic("database " + scm.String(a[0]) + " does not exist")
					}
					t := db.GetTable(scm.String(a[1]))
					if t == nil {
						panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
					}
					shards := t.ActiveShards()
					idx := int(scm.ToInt(a[2]))
					if idx < 0 || idx >= len(shards) {
						panic("shard index out of range")
					}
					s := shards[idx]
					// build shard overview fields
					overview := showBuildShardRow(t, idx, s)
					// build columns detail
					var colRows scm.Scmer
					var indexRows scm.Scmer
					if s == nil {
						colRows = scm.NewSlice([]scm.Scmer{})
						indexRows = scm.NewSlice([]scm.Scmer{})
					} else {
						s.mu.RLock()
						deltaCount := len(s.inserts)
						colSlice := make([]scm.Scmer, 0, len(t.Columns))
						for _, col := range t.Columns {
							cs := s.columns[col.Name]
							var typStr string
							var colSize uint
							if cs != nil {
								typStr = cs.String()
								colSize = cs.ComputeSize()
							} else {
								typStr = "unloaded"
								colSize = 0
							}
							var deltaSize uint
							if dIdx, ok := s.deltaColumns[col.Name]; ok {
								for _, row := range s.inserts {
									if dIdx < len(row) {
										deltaSize += row[dIdx].ComputeSize()
									}
								}
							}
							colSlice = append(colSlice, scm.NewSlice([]scm.Scmer{
								scm.NewString("name"), scm.NewString(col.Name),
								scm.NewString("compression"), scm.NewString(typStr),
								scm.NewString("size_bytes"), scm.NewInt(int64(colSize)),
								scm.NewString("delta_count"), scm.NewInt(int64(deltaCount)),
								scm.NewString("delta_size_bytes"), scm.NewInt(int64(deltaSize)),
							}))
						}
						colRows = scm.NewSlice(colSlice)
						idxSlice := make([]scm.Scmer, 0, len(s.Indexes))
						for _, ix := range s.Indexes {
							orders := make([]scm.Scmer, len(ix.ColOrderMeta))
							for i, order := range ix.ColOrderMeta {
								orders[i] = scm.NewString(order)
							}
							idxSlice = append(idxSlice, scm.NewSlice([]scm.Scmer{
								scm.NewString("cols"), scm.NewString(ix.String()),
								scm.NewString("orders"), scm.NewSlice(orders),
								scm.NewString("active"), scm.NewBool(ix.baseState.active),
								scm.NewString("native"), scm.NewBool(ix.Native),
								scm.NewString("savings"), scm.NewFloat(ix.Savings),
								scm.NewString("size_bytes"), scm.NewInt(int64(ix.ComputeSize())),
							}))
						}
						indexRows = scm.NewSlice(idxSlice)
						s.mu.RUnlock()
					}
					// merge overview fields with columns and indexes
					overviewSlice := overview.Slice()
					result := make([]scm.Scmer, 0, len(overviewSlice)+4)
					result = append(result, overviewSlice...)
					result = append(result, scm.NewString("columns"), colRows)
					result = append(result, scm.NewString("indexes"), indexRows)
					return scm.NewSlice(result)
				}
				panic("invalid call of show")
			}
			panic("invalid call of show")
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "show databases/tables/columns/shards\n\n(show) lists database names\n(show schema) lists table names\n(show table_handle) lists the memoized column defs\n(show table_handle true) returns table metadata\n(show table_handle \"statistics\") returns index statistics\n(show schema true) lists tables with full info: [{name,engine,row_count,size_bytes,collation,comment},...]\n(show schema tbl) lists column defs\n(show schema tbl true) returns assoc {columns,meta,shards}\n(show schema tbl N) returns shard N overview assoc {shard,state,main_count,delta,deletions,size_bytes}\n(show schema tbl N true) returns shard N full assoc adding columns and indexes\n(show schema tbl \"statistics\") returns INFORMATION_SCHEMA index statistics\n(show schema tbl \"indexes\") returns MySQL SHOW INDEX rows",
			Params: []*scm.TypeDescriptor{
				{Kind: "string|table|recset", Label: "schema_or_table", Description: "(optional) database name or resolved table/recset handle", Optional: true},
				{Kind: "string|bool", Label: "table_or_property", Description: "(optional) table name, true for full info, or \"statistics\" for a handle", Optional: true},
				{Kind: "int|bool|string", Label: "property", Description: "(optional) shard index (int), true for full table info, or \"statistics\"", Optional: true},
				{Kind: "bool", Label: "full", Description: "(optional) true to include columns and indexes in shard detail", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any", Transfer: false},
		},
	})
	// show_triggers(schema, table): returns a list of triggers for a table (non-system triggers only)
	scm.Declare(&en, &scm.Declaration{
		Name: "show_triggers",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			rows := make([]scm.Scmer, 0)
			tables := db.tables.GetAll()
			for _, t := range tables {
				// If table name specified, filter
				if len(a) >= 2 && scm.String(a[1]) != t.Name {
					continue
				}
				for _, tr := range t.Triggers {
					// Skip system/internal triggers — only show user-visible ones
					if tr.IsSystem || tr.Hidden {
						continue
					}
					// MySQL SHOW TRIGGERS format:
					// Trigger, Event, Table, Statement, Timing, Created, sql_mode, Definer, character_set_client, collation_connection, Database Collation
					timing := "BEFORE"
					event := "INSERT"
					switch tr.Timing {
					case BeforeInsert:
						timing, event = "BEFORE", "INSERT"
					case AfterInsert:
						timing, event = "AFTER", "INSERT"
					case BeforeUpdate:
						timing, event = "BEFORE", "UPDATE"
					case AfterUpdate:
						timing, event = "AFTER", "UPDATE"
					case BeforeDelete:
						timing, event = "BEFORE", "DELETE"
					case AfterDelete:
						timing, event = "AFTER", "DELETE"
					}
					funcStr := ""
					if !tr.Func.IsNil() {
						funcStr = scm.String(tr.Func)
					}
					rows = append(rows, scm.NewSlice([]scm.Scmer{
						scm.NewString("Trigger"), scm.NewString(tr.Name),
						scm.NewString("Event"), scm.NewString(event),
						scm.NewString("Table"), scm.NewString(t.Name),
						scm.NewString("Statement"), scm.NewString(tr.SourceSQL),
						scm.NewString("Timing"), scm.NewString(timing),
						scm.NewString("Created"), scm.NewNil(),
						scm.NewString("sql_mode"), scm.NewString(""),
						scm.NewString("Definer"), scm.NewString(""),
						scm.NewString("character_set_client"), scm.NewString("utf8mb4"),
						scm.NewString("collation_connection"), scm.NewString("utf8mb4_general_ci"),
						scm.NewString("Database Collation"), scm.NewString("utf8mb4_general_ci"),
						scm.NewString("FuncStr"), scm.NewString(funcStr),
					}))
				}
			}
			return scm.NewSlice(rows)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "show triggers for a given table",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "database name"},
				{Kind: "string", Label: "table", Description: "(optional) table name, if omitted shows all triggers in schema", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "rebuild",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) > 0 && a[0].IsCustom(TagTable) {
				all := len(a) > 1 && scm.ToBool(a[1])
				repartition := true
				if len(a) > 2 {
					repartition = scm.ToBool(a[2])
				}
				return scm.NewString(RebuildTable(TableFromScmer(a[0]), all, repartition))
			}
			if len(a) > 2 {
				panic("global rebuild accepts at most all and repartition")
			}
			all := false
			if len(a) > 0 && scm.ToBool(a[0]) {
				all = true
			}
			repartition := true
			if len(a) > 1 {
				repartition = scm.ToBool(a[1])
			}

			return scm.NewString(Rebuild(all, repartition))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "rebuilds main storages and returns the amount of time it took; with a table handle, rebuilds only that table",
			Params: []*scm.TypeDescriptor{
				{Kind: "bool|table", Label: "table_or_all", Description: "table handle for a table-local rebuild; otherwise whether to rebuild unchanged shards globally (default: false)", Optional: true},
				{Kind: "bool", Label: "all_or_repartition", Description: "with a table: whether to rebuild unchanged shards; globally: whether to repartition (default: true)", Optional: true},
				{Kind: "bool", Label: "repartition", Description: "with a table handle, whether to repartition that table (default: true)", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	// clean() is intentionally not exposed as a SQL function.
	// It runs automatically at startup (in a background goroutine) after LoadDatabases().

	scm.Declare(&en, &scm.Declaration{
		Name: "loadCSV",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			// schema, table, filename, delimiter
			start := time.Now()

			delimiter := ";"
			if len(a) > 3 {
				delimiter = scm.String(a[3])
			}
			firstline := true
			if len(a) > 4 {
				firstline = scm.ToBool(a[4])
			}
			stream, ok := a[2].Any().(io.Reader)
			if !ok {
				panic("loadCSV expects a stream")
			}
			LoadCSV(scm.String(a[0]), scm.String(a[1]), stream, delimiter, firstline)

			return scm.NewString(fmt.Sprint(time.Since(start)))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "loads a CSV stream into a table and returns the amount of time it took.\nThe first line of the file must be the headlines. The headlines must match the table's columns exactly.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "string", Label: "table", Description: "name of the table"},
				{Kind: "stream", Label: "stream", Description: "CSV file, load with: (stream filename)"},
				{Kind: "string", Label: "delimiter", Description: "(optional) delimiter defaults to \";\"", Optional: true},
				{Kind: "bool", Label: "firstline", Description: "(optional) if the first line contains the column names (otherwise, the tables column order is used)", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "loadJSON",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			// schema, filename
			start := time.Now()

			stream, ok := a[1].Any().(io.Reader)
			if !ok {
				panic("loadJSON expects a stream")
			}
			LoadJSON(scm.String(a[0]), stream)

			return scm.NewString(fmt.Sprint(time.Since(start)))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "loads a .jsonl file from stream into a database and returns the amount of time it took.\nJSONL is a linebreak separated file of JSON objects. Each JSON object is one dataset in the database. Before you add rows, you must declare the table in a line '#table <tablename>'. All other lines starting with # are comments. Columns are created dynamically as soon as they occur in a json object.", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database where you want to put the tables in"},
				{Kind: "stream", Label: "stream", Description: "stream of the .jsonl file, read with: (stream filename)"},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "settings",

		Fn: ChangeSettings,
		Type: &scm.TypeDescriptor{Kind: "func", Description: "reads or writes a global settings value. This modifies your data/settings.json.",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "key", Description: "name of the key to set or get (for reference, rts)", Optional: true},
				{Kind: "any", Label: "value", Description: "new value of that setting", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})

	// Trigger management
	scm.Declare(&en, &scm.Declaration{
		Name: "createcreatetabletrigger",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			body, deferredPlan := unwrapDeferredTriggerBody(a[4])
			if triggerScmerMissing(body) && !triggerScmerMissing(deferredPlan) {
				body = scm.Eval(deferredPlan, &scm.Globalenv)
			}
			if triggerScmerMissing(body) {
				panic("create-table trigger body must not be empty")
			}
			trigger := TriggerDescription{
				Name:      scm.String(a[2]),
				Timing:    AfterCreateTable,
				Func:      body,
				SourceSQL: scm.String(a[3]),
				Hidden:    !scm.ToBool(a[5]),
			}
			finalizeTriggerCompilation(&trigger)
			registerCreateTableTrigger(CreateTableTriggerRegistration{
				Schema:    scm.String(a[0]),
				Table:     scm.String(a[1]),
				Name:      trigger.Name,
				SourceSQL: trigger.SourceSQL,
				Hidden:    trigger.Hidden,
				Priority:  trigger.Priority,
				Async:     trigger.Async,
				Func:      trigger.Func,
			})
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "registers a lifecycle trigger that fires synchronously after a future createtable for the given schema/table succeeds", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "string", Label: "table", Description: "name of the table to watch for creation"},
				{Kind: "string", Label: "name", Description: "name of the trigger"},
				{Kind: "string", Label: "source_sql", Description: "original SQL body text (for diagnostics)"},
				{Kind: "any", Label: "body", Description: "trigger body (Scheme procedure or deferred trigger expression)"},
				{Kind: "bool", Label: "visible", Description: "true = user trigger, false = internal trigger"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropcreatetabletrigger",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			schema := scm.String(a[0])
			table := scm.String(a[1])
			name := scm.String(a[2])
			if dropCreateTableTrigger(schema, table, name) {
				return scm.NewBool(true)
			}
			if scm.ToBool(a[3]) {
				return scm.NewBool(false)
			}
			panic("create-table trigger " + schema + "." + table + ":" + name + " does not exist")
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "removes a registered create-table lifecycle trigger", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "string", Label: "table", Description: "name of the table watched for creation"},
				{Kind: "string", Label: "name", Description: "name of the trigger"},
				{Kind: "bool", Label: "ifexists", Description: "don't throw error if trigger doesn't exist"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createtrigger",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			db := t.schema

			name := scm.String(a[1])
			timingStr := scm.String(a[2])
			var timing TriggerTiming
			switch timingStr {
			case "before_insert":
				timing = BeforeInsert
			case "after_insert":
				timing = AfterInsert
			case "before_update":
				timing = BeforeUpdate
			case "after_update":
				timing = AfterUpdate
			case "before_delete":
				timing = BeforeDelete
			case "after_delete":
				timing = AfterDelete
			case "after_drop_table":
				timing = AfterDropTable
			case "after_drop_column":
				timing = AfterDropColumn
			case "after_invalidate":
				timing = AfterInvalidate
			default:
				panic("invalid trigger timing: " + timingStr)
			}

			sourceSQL := scm.String(a[3])
			body, deferredPlan := unwrapDeferredTriggerBody(a[4])
			visible := scm.ToBool(a[5])
			if visible {
				requireTableMaintenance(db.Name, t.Name, maintenanceAlter)
			}

			trigger := TriggerDescription{
				Name:      name,
				Timing:    timing,
				Func:      body,
				FuncPlan:  deferredPlan,
				SourceSQL: sourceSQL,
				Hidden:    !visible,
				Priority:  0,
			}
			t.ddlMu.Lock()
			defer t.ddlMu.Unlock()
			db.schemalock.Lock()
			// Idempotent: replace any existing trigger with the same name
			t.RemoveTrigger(name)
			t.AddTrigger(trigger)
			db.saveLockedAndUnlock(t.schemaSaveMode())
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "creates a new trigger on a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", Label: "table"},
				{Kind: "string", Label: "name", Description: "name of the trigger"},
				{Kind: "string", Label: "timing", Description: "one of: before_insert, after_insert, before_update, after_update, before_delete, after_delete"},
				{Kind: "string", Label: "source_sql", Description: "original SQL body text (for SHOW TRIGGERS)"},
				{Kind: "any", Label: "body", Description: "trigger body (parsed Scheme expression)"},
				{Kind: "bool", Label: "visible", Description: "true = user trigger (shown in SHOW TRIGGERS), false = internal trigger (hidden)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "droptrigger",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				if scm.ToBool(a[2]) {
					return scm.NewBool(false)
				}
				panic("database " + scm.String(a[0]) + " does not exist")
			}

			name := scm.String(a[1])
			if db.dropTrigger(name) {
				return scm.NewBool(true)
			}
			if scm.ToBool(a[2]) {
				return scm.NewBool(false)
			}
			panic("trigger " + name + " does not exist")
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "removes a trigger from a table", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "name of the database"},
				{Kind: "string", Label: "name", Description: "name of the trigger"},
				{Kind: "bool", Label: "ifexists", Description: "don't throw error if trigger doesn't exist"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	initMySQLImport(en)
	initPSQLImport(en)
	initDashboard(en)
	initMetricsDeclarations(en)
	scm.DeclareInSection("Sync", &en, &scm.Declaration{
		Name: "newcachemap",

		Fn: NewCacheMap,
		Type: &scm.TypeDescriptor{Kind: "func", Description: "Creates a new cachemap. Returns a threadsafe key-value function with LRU eviction under memory pressure: (cachemap key value) sets, (cachemap key) gets, (cachemap) lists keys, (cachemap \"get_or_compute\" key producer) computes one value for concurrent misses of the same key.",
			Return: cacheMapCallableType,
		},
	})
	initTransaction(en)
	initFKBuiltins(en)
}

func PrintMemUsage() string {
	m := scm.CachedMemStats()
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", units.BytesSize(float64(m.Alloc)), units.BytesSize(float64(m.TotalAlloc)), units.BytesSize(float64(m.Sys)), m.NumGC))

	// CacheManager evictable memory breakdown
	b.WriteString("\n\nCache\n======\n")
	b.WriteString(GlobalCache.Stat().FormatStat())

	for _, db := range databases.GetAll() {
		b.WriteString("\n\n" + db.Name + " [" + sharedStateStr(db.srState) + "]\n======\n")
		b.WriteString(db.PrintMemUsage())
	}
	return b.String()
}

func sharedStateStr(s SharedState) string {
	switch s {
	case COLD:
		return "COLD"
	case SHARED:
		return "SHARED"
	case WRITE:
		return "WRITE"
	default:
		return "UNKNOWN"
	}
}

func (db *database) PrintMemUsage() string {
	var b strings.Builder
	if db.srState == COLD {
		b.WriteString("State: COLD (no schema loaded)\n")
		return b.String()
	}
	b.WriteString("Disjoint owner-payload estimate (not RSS; excludes Go runtime, allocator slack, stacks, and shared process overhead)\n")
	b.WriteString("Table                    \tColumns\tShards\tBase\tIndexes\tTemp columns\tString dicts\tMetadata\tTotal\n")
	var total memoryOwnerSnapshot
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	for _, t := range db.tables.GetAll() {
		snapshot := t.memoryOwnerSnapshotLocked()
		b.WriteString(fmt.Sprintf("%-25s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			t.Name, len(t.Columns), len(t.ActiveShards()),
			units.BytesSize(float64(snapshot.base)),
			units.BytesSize(float64(snapshot.indexes)),
			units.BytesSize(float64(snapshot.tempColumns)),
			units.BytesSize(float64(snapshot.stringDictionaries)),
			units.BytesSize(float64(snapshot.metadata)),
			units.BytesSize(float64(snapshot.total()))))
		total.add(snapshot)
	}
	b.WriteString(fmt.Sprintf("\ntotal owner payload estimate = %s\n", units.BytesSize(float64(total.total()))))
	return b.String()
}

func (t *table) PrintMemUsage() string {
	var b strings.Builder
	t.schema.schemalock.RLock()
	snapshot := t.memoryOwnerSnapshotLocked()
	shardCount := len(t.ActiveShards())
	columnCount := len(t.Columns)
	partitioned := t.ShardMode == ShardModePartition
	partitioningSchema := fmt.Sprint(t.PDimensions)
	t.schema.schemalock.RUnlock()
	if partitioned {
		b.WriteString("Partitioning Schema:" + partitioningSchema + "\n\n")
	}
	b.WriteString("Disjoint owner-payload estimate (not RSS)\n")
	b.WriteString(fmt.Sprintf("columns: %d, shards: %d\n", columnCount, shardCount))
	b.WriteString(fmt.Sprintf("base shard payload: %s\n", units.BytesSize(float64(snapshot.base))))
	b.WriteString(fmt.Sprintf("indexes: %s\n", units.BytesSize(float64(snapshot.indexes))))
	b.WriteString(fmt.Sprintf("temporary columns: %s\n", units.BytesSize(float64(snapshot.tempColumns))))
	b.WriteString(fmt.Sprintf("materialized string dictionaries: %s\n", units.BytesSize(float64(snapshot.stringDictionaries))))
	b.WriteString(fmt.Sprintf("table metadata estimate: %s\n", units.BytesSize(float64(snapshot.metadata))))
	b.WriteString(fmt.Sprintf("total owner payload estimate: %s\n", units.BytesSize(float64(snapshot.total()))))
	return b.String()
}

type memoryOwnerSnapshot struct {
	base               uint
	indexes            uint
	tempColumns        uint
	stringDictionaries uint
	metadata           uint
}

func (m memoryOwnerSnapshot) total() uint {
	return m.base + m.indexes + m.tempColumns + m.stringDictionaries + m.metadata
}

func (m *memoryOwnerSnapshot) add(other memoryOwnerSnapshot) {
	m.base += other.base
	m.indexes += other.indexes
	m.tempColumns += other.tempColumns
	m.stringDictionaries += other.stringDictionaries
	m.metadata += other.metadata
}

// memoryOwnerSnapshotLocked attributes every loaded storage payload to exactly
// one owner. CacheManager weights and partial/full eviction policy are not part
// of size ownership and remain independently applied by cache.go. The caller
// holds the schema read lock so table topology and column ownership are stable.
func (t *table) memoryOwnerSnapshotLocked() memoryOwnerSnapshot {
	snapshot := memoryOwnerSnapshot{metadata: 10*8 + 32*uint(len(t.Columns))}
	for _, shard := range t.ActiveShards() {
		if shard == nil {
			continue
		}
		shard.mu.RLock()
		snapshot.base += shard.computeSizeLocked()
		for name, storage := range shard.columns {
			if storage == nil {
				continue
			}
			if !shard.ownsColumnMemory(name) {
				snapshot.tempColumns += ownedColumnMemory(storage)
			}
			snapshot.stringDictionaries += materializedDictionaryMemory(storage)
		}
		for _, index := range shard.Indexes {
			if index != nil {
				snapshot.indexes += index.ComputeSize()
			}
		}
		shard.mu.RUnlock()
	}
	return snapshot
}

// fkExistenceCheck checks if values exist in tbl[filterCols]. Returns true if found or all NULL.
func fkExistenceCheck(currentTx *TxContext, tbl *table, filterCols []string, vals []scm.Scmer) bool {
	for _, v := range vals {
		if v.IsNil() {
			return true // NULL FK is always valid
		}
	}
	condition := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		for i := range filterCols {
			if !scm.Equal(a[i], vals[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	reduceFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		if scm.ToBool(a[0]) || scm.ToBool(a[1]) {
			return scm.NewBool(true)
		}
		return scm.NewBool(false)
	})
	return scm.ToBool(tbl.scan(currentTx, filterCols, condition, filterCols[:0], mapFn, reduceFn, scm.NewBool(false), reduceFn, false))
}

// fkCascadeDelete deletes rows in childTbl where cols match vals.
func fkCascadeDelete(currentTx *TxContext, childTbl *table, cols []string, vals []scm.Scmer) {
	condition := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		for i := range cols {
			if !scm.Equal(a[i], vals[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	mapCols := make([]string, len(cols)+1)
	copy(mapCols, cols)
	mapCols[len(cols)] = "$update"
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		scm.Apply(a[len(cols)]) // $update() with no args = delete
		return scm.NewNil()
	})
	childTbl.scan(currentTx, cols, condition, mapCols, mapFn, scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
}

// fkCascadeSetNull sets FK cols to NULL in childTbl where cols match vals.
func fkCascadeSetNull(currentTx *TxContext, childTbl *table, cols []string, vals []scm.Scmer) {
	condition := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		for i := range cols {
			if !scm.Equal(a[i], vals[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	payload := make([]scm.Scmer, len(cols)*2)
	for i, col := range cols {
		payload[i*2] = scm.NewString(col)
		payload[i*2+1] = scm.NewNil()
	}
	mapCols := make([]string, len(cols)+1)
	copy(mapCols, cols)
	mapCols[len(cols)] = "$update"
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		scm.Apply(a[len(cols)], scm.NewSlice(payload))
		return scm.NewNil()
	})
	childTbl.scan(currentTx, cols, condition, mapCols, mapFn, scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
}

// fkCascadeUpdate updates FK cols in childTbl from oldVals to newVals.
func fkCascadeUpdate(currentTx *TxContext, childTbl *table, cols []string, oldVals, newVals []scm.Scmer) {
	condition := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		for i := range cols {
			if !scm.Equal(a[i], oldVals[i]) {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	payload := make([]scm.Scmer, len(cols)*2)
	for i, col := range cols {
		payload[i*2] = scm.NewString(col)
		payload[i*2+1] = newVals[i]
	}
	mapCols := make([]string, len(cols)+1)
	copy(mapCols, cols)
	mapCols[len(cols)] = "$update"
	mapFn := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		scm.Apply(a[len(cols)], scm.NewSlice(payload))
		return scm.NewNil()
	})
	childTbl.scan(currentTx, cols, condition, mapCols, mapFn, scm.NewNil(), scm.NewNil(), scm.NewNil(), false)
}

// initFKBuiltins declares the FK enforcement builtins used by trigger Procs.
func initFKBuiltins(en scm.Env) {
	scm.Declare(&en, &scm.Declaration{
		Name: "__fk_check_ref",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[5])
			schema := scm.String(a[0])
			parentTable := scm.String(a[1])
			parentCols := scmerSliceToStrings(mustScmerSlice(a[2], "parent_cols"))
			values := mustScmerSlice(a[3], "values")
			fkId := scm.String(a[4])
			// NULL FK values are always valid
			for _, v := range values {
				if v.IsNil() {
					return scm.NewNil()
				}
			}
			db := GetDatabase(schema)
			if db == nil {
				panic("foreign key " + fkId + ": database " + schema + " does not exist")
			}
			tbl := db.GetTable(parentTable)
			if tbl == nil {
				panic("foreign key " + fkId + ": parent table " + schema + "." + parentTable + " does not exist")
			}
			if !fkExistenceCheck(currentTx, tbl, parentCols, values) {
				panic(sqldb.NewSQLError1(1452, "23000", "foreign key constraint %s failed: value does not exist in %s", fkId, parentTable))
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "check that FK values exist in the parent table, panic if not",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "database name"},
				{Kind: "string", Label: "parent_table", Description: "parent table name"},
				{Kind: "list", Label: "parent_cols", Description: "parent column names"},
				{Kind: "list", Label: "values", Description: "FK values to check"},
				{Kind: "string", Label: "fk_id", Description: "FK constraint name"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "__fk_on_parent_delete",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[6])
			schema := scm.String(a[0])
			childTable := scm.String(a[1])
			childCols := scmerSliceToStrings(mustScmerSlice(a[2], "child_cols"))
			parentVals := mustScmerSlice(a[3], "parent_vals")
			fkId := scm.String(a[4])
			mode := scm.String(a[5])
			db := GetDatabase(schema)
			if db == nil {
				return scm.NewNil()
			}
			tbl := db.GetTable(childTable)
			if tbl == nil {
				return scm.NewNil()
			}
			if !fkExistenceCheck(currentTx, tbl, childCols, parentVals) {
				return scm.NewNil() // no references
			}
			switch mode {
			case "RESTRICT":
				panic(sqldb.NewSQLError1(1451, "23000", "foreign key constraint %s failed: cannot delete because rows in %s reference it", fkId, childTable))
			case "CASCADE":
				fkCascadeDelete(currentTx, tbl, childCols, parentVals)
			case "SETNULL":
				fkCascadeSetNull(currentTx, tbl, childCols, parentVals)
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "enforce FK constraint when parent row is deleted",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "database name"},
				{Kind: "string", Label: "child_table", Description: "child table name"},
				{Kind: "list", Label: "child_cols", Description: "child FK column names"},
				{Kind: "list", Label: "parent_vals", Description: "old parent PK values"},
				{Kind: "string", Label: "fk_id", Description: "FK constraint name"},
				{Kind: "string", Label: "mode", Description: "RESTRICT, CASCADE, or SETNULL"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "__fk_on_parent_update",

		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := scmerToTxContext(a[7])
			schema := scm.String(a[0])
			childTable := scm.String(a[1])
			childCols := scmerSliceToStrings(mustScmerSlice(a[2], "child_cols"))
			oldVals := mustScmerSlice(a[3], "old_vals")
			newVals := mustScmerSlice(a[4], "new_vals")
			fkId := scm.String(a[5])
			mode := scm.String(a[6])
			// check if PK actually changed
			if len(oldVals) == len(newVals) {
				changed := false
				for i := range oldVals {
					if !scm.Equal(oldVals[i], newVals[i]) {
						changed = true
						break
					}
				}
				if !changed {
					return scm.NewNil()
				}
			}
			db := GetDatabase(schema)
			if db == nil {
				return scm.NewNil()
			}
			tbl := db.GetTable(childTable)
			if tbl == nil {
				return scm.NewNil()
			}
			switch mode {
			case "RESTRICT":
				if fkExistenceCheck(currentTx, tbl, childCols, oldVals) {
					panic(sqldb.NewSQLError1(1451, "23000", "foreign key constraint %s failed: cannot update because rows in %s reference it", fkId, childTable))
				}
			case "CASCADE":
				fkCascadeUpdate(currentTx, tbl, childCols, oldVals, newVals)
			case "SETNULL":
				fkCascadeSetNull(currentTx, tbl, childCols, oldVals)
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{Kind: "func", Description: "enforce FK constraint when parent PK is updated",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", Label: "schema", Description: "database name"},
				{Kind: "string", Label: "child_table", Description: "child table name"},
				{Kind: "list", Label: "child_cols", Description: "child FK column names"},
				{Kind: "list", Label: "old_vals", Description: "old parent PK values"},
				{Kind: "list", Label: "new_vals", Description: "new parent PK values"},
				{Kind: "string", Label: "fk_id", Description: "FK constraint name"},
				{Kind: "string", Label: "mode", Description: "RESTRICT, CASCADE, or SETNULL"},
				{Kind: "any", Label: "tx", Description: "explicit transaction context"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

}

// buildFKProc constructs a serializable Proc that calls a builtin with the given args.
// body is the Scheme expression as an S-expression (a Scmer list).
func buildFKProc(body scm.Scmer) scm.Scmer {
	// Generated trigger bodies may contain optimized child closures which address
	// these four trigger arguments through outer numbered slots. Persist the
	// complete frame shape so the closures keep the same lexical layout after a
	// schema reload, while NumberedOnly remains false for symbolic trigger bodies.
	return scm.NewProc(&scm.Proc{
		Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("OLD"), scm.NewSymbol("NEW"), scm.NewSymbol("session"), scm.NewSymbol("tx")}),
		Body:    body,
		En:      &scm.Globalenv,
		NumVars: 4,
	})
}

// fkGetAssocExpr builds (get_assoc <sym> <colName>) expression
func fkGetAssocExpr(sym string, col string) scm.Scmer {
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("get_assoc"), scm.NewSymbol(sym), scm.NewString(col)})
}

// fkValListExpr builds (list (get_assoc sym col1) (get_assoc sym col2) ...) expression
func fkValListExpr(sym string, cols []string) scm.Scmer {
	elems := make([]scm.Scmer, 1+len(cols))
	elems[0] = scm.NewSymbol("list")
	for i, col := range cols {
		elems[1+i] = fkGetAssocExpr(sym, col)
	}
	return scm.NewSlice(elems)
}

// fkQuotedList builds a quoted literal list: (quote ("col1" "col2" ...))
func fkQuotedList(cols []string) scm.Scmer {
	elems := make([]scm.Scmer, len(cols))
	for i, col := range cols {
		elems[i] = scm.NewString(col)
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(elems)})
}

func fkTxExpr() scm.Scmer {
	return scm.NewSymbol("tx")
}

// installFKTriggers creates system triggers on child (t1) and parent (t2) tables
// to enforce the foreign key constraint. All trigger functions are serializable Procs
// that call declared builtins (__fk_check_ref, __fk_on_parent_delete, __fk_on_parent_update).
func installFKTriggers(db *database, t1, t2 *table, fk foreignKey) {
	triggerPrefix := "__fk_" + fk.Id + "_"
	dbName := db.Name

	// 1) BEFORE INSERT on child: (lambda (OLD NEW) (begin (__fk_check_ref ...) NEW))
	t1.AddTrigger(TriggerDescription{
		Name:     triggerPrefix + "child_insert",
		Timing:   BeforeInsert,
		IsSystem: true,
		Priority: -100,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("__fk_check_ref"),
				scm.NewString(dbName), scm.NewString(fk.Tbl2),
				fkQuotedList(fk.Cols2), fkValListExpr("NEW", fk.Cols1),
				scm.NewString(fk.Id), fkTxExpr(),
			}),
			scm.NewSymbol("NEW"),
		})),
	})

	// 2) BEFORE UPDATE on child: (lambda (OLD NEW) (begin (__fk_check_ref ...) NEW))
	t1.AddTrigger(TriggerDescription{
		Name:     triggerPrefix + "child_update",
		Timing:   BeforeUpdate,
		IsSystem: true,
		Priority: -100,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("__fk_check_ref"),
				scm.NewString(dbName), scm.NewString(fk.Tbl2),
				fkQuotedList(fk.Cols2), fkValListExpr("NEW", fk.Cols1),
				scm.NewString(fk.Id), fkTxExpr(),
			}),
			scm.NewSymbol("NEW"),
		})),
	})

	// 3) BEFORE DELETE on parent: (lambda (OLD NEW) (__fk_on_parent_delete ...))
	modeStr := "RESTRICT"
	switch fk.Deletemode {
	case CASCADE:
		modeStr = "CASCADE"
	case SETNULL:
		modeStr = "SETNULL"
	}
	t2.AddTrigger(TriggerDescription{
		Name:     triggerPrefix + "parent_delete",
		Timing:   BeforeDelete,
		IsSystem: true,
		Priority: -100,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("__fk_on_parent_delete"),
			scm.NewString(dbName), scm.NewString(fk.Tbl1),
			fkQuotedList(fk.Cols1), fkValListExpr("OLD", fk.Cols2),
			scm.NewString(fk.Id), scm.NewString(modeStr), fkTxExpr(),
		})),
	})

	// 4) Parent UPDATE: RESTRICT uses BEFORE UPDATE, CASCADE/SET NULL use AFTER UPDATE
	// (BEFORE UPDATE triggers run inside shard write lock; cascaded child updates
	// that scan back to the parent would deadlock)
	updateModeStr := "RESTRICT"
	switch fk.Updatemode {
	case CASCADE:
		updateModeStr = "CASCADE"
	case SETNULL:
		updateModeStr = "SETNULL"
	}
	timing := BeforeUpdate
	if fk.Updatemode != RESTRICT {
		timing = AfterUpdate
	}
	t2.AddTrigger(TriggerDescription{
		Name:     triggerPrefix + "parent_update",
		Timing:   timing,
		IsSystem: true,
		Priority: -100,
		Func: buildFKProc(scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("__fk_on_parent_update"),
				scm.NewString(dbName), scm.NewString(fk.Tbl1),
				fkQuotedList(fk.Cols1),
				fkValListExpr("OLD", fk.Cols2), fkValListExpr("NEW", fk.Cols2),
				scm.NewString(fk.Id), scm.NewString(updateModeStr), fkTxExpr(),
			}),
			scm.NewSymbol("NEW"),
		})),
	})
}

// showEngineStr returns the engine name string for a table (matches dashboard dropdown values).
func showEngineStr(t *table) string {
	switch t.PersistencyMode {
	case Logged:
		return "logged"
	case Sloppy:
		return "sloppy"
	case Memory:
		return "memory"
	default:
		return "safe"
	}
}

// showBuildIndexRows is the single metadata source for INFORMATION_SCHEMA and
// MySQL SHOW INDEX. Keeping the two wire spellings over the same immutable key
// snapshot prevents schema synchronizers from seeing a different constraint
// catalog than the SQL planner and insert path.
func showBuildIndexRows(db *database, t *table, mysqlNames bool) scm.Scmer {
	t.ddlMu.RLock()
	keys := make([]uniqueKey, len(t.Unique))
	for i, key := range t.Unique {
		keys[i] = uniqueKey{Id: key.Id, Cols: append([]string(nil), key.Cols...)}
	}
	t.ddlMu.RUnlock()

	result := make([]scm.Scmer, 0, len(keys))
	for _, key := range keys {
		for seq, col := range key.Cols {
			if mysqlNames {
				result = append(result, scm.NewSlice([]scm.Scmer{
					scm.NewString("Table"), scm.NewString(t.Name),
					scm.NewString("Non_unique"), scm.NewInt(0),
					scm.NewString("Key_name"), scm.NewString(key.Id),
					scm.NewString("Seq_in_index"), scm.NewInt(int64(seq + 1)),
					scm.NewString("Column_name"), scm.NewString(col),
					scm.NewString("Collation"), scm.NewString("A"),
					scm.NewString("Cardinality"), scm.NewNil(),
					scm.NewString("Sub_part"), scm.NewNil(),
					scm.NewString("Packed"), scm.NewNil(),
					scm.NewString("Null"), scm.NewString(""),
					scm.NewString("Index_type"), scm.NewString("BTREE"),
					scm.NewString("Comment"), scm.NewString(""),
					scm.NewString("Index_comment"), scm.NewString(""),
					scm.NewString("Visible"), scm.NewString("YES"),
					scm.NewString("Expression"), scm.NewNil(),
				}))
				continue
			}
			result = append(result, scm.NewSlice([]scm.Scmer{
				scm.NewString("table_catalog"), scm.NewString("def"),
				scm.NewString("table_schema"), scm.NewString(db.Name),
				scm.NewString("table_name"), scm.NewString(t.Name),
				scm.NewString("non_unique"), scm.NewInt(0),
				scm.NewString("index_schema"), scm.NewString(db.Name),
				scm.NewString("index_name"), scm.NewString(key.Id),
				scm.NewString("seq_in_index"), scm.NewInt(int64(seq + 1)),
				scm.NewString("column_name"), scm.NewString(col),
				scm.NewString("collation"), scm.NewString("A"),
				scm.NewString("cardinality"), scm.NewNil(),
				scm.NewString("sub_part"), scm.NewNil(),
				scm.NewString("packed"), scm.NewNil(),
				scm.NewString("nullable"), scm.NewString(""),
				scm.NewString("index_type"), scm.NewString("BTREE"),
				scm.NewString("comment"), scm.NewString(""),
				scm.NewString("index_comment"), scm.NewString(""),
			}))
		}
	}
	return scm.NewSlice(result)
}

func showStringSlice(values []string) scm.Scmer {
	items := make([]scm.Scmer, len(values))
	for i, value := range values {
		items[i] = scm.NewString(value)
	}
	return scm.NewSlice(items)
}

func showForeignKeyMode(mode foreignKeyMode) string {
	switch mode {
	case CASCADE:
		return "cascade"
	case SETNULL:
		return "set null"
	default:
		return "restrict"
	}
}

func plannerDistinctForColumns(t *table, columns []string) (float64, float64, string) {
	rows := float64(t.CountEstimate())
	if len(columns) == 0 {
		return 0, 0, "unknown"
	}
	estimate := 1.0
	confidence := 1.0
	for _, columnName := range columns {
		var descriptor *column
		for _, candidate := range t.Columns {
			if candidate.Name == columnName {
				descriptor = candidate
				break
			}
		}
		if descriptor == nil || descriptor.PlannerStats.Load() == nil {
			return 0, 0, "unknown"
		}
		distinct := float64(atomic.LoadUint64(&descriptor.DistinctEstimate))
		if distinct <= 0 {
			return 0, 0, "unknown"
		}
		estimate *= distinct
		confidence *= descriptor.PlannerStats.Load().Confidence
	}
	if estimate > rows {
		estimate = rows
	}
	return estimate, confidence, "rebuild_independence"
}

// showBuildMeta builds table metadata including key, relationship, and
// multi-column planner statistics. All statistics are immutable snapshots.
func showBuildMeta(db *database, t *table) scm.Scmer {
	engine := showEngineStr(t)
	maintenance := tableMaintenanceCapabilities(db.Name, t.Name)
	t.mu.Lock()
	nextAutoIncrement := t.Auto_increment + 1
	t.mu.Unlock()
	uniques := make([]scm.Scmer, len(t.Unique))
	for i, uk := range t.Unique {
		uniques[i] = scm.NewSlice([]scm.Scmer{
			scm.NewString("Id"), scm.NewString(uk.Id),
			scm.NewString("Cols"), showStringSlice(uk.Cols),
		})
	}
	multiColumnDistinct := make([]scm.Scmer, 0, len(t.Unique))
	for _, uk := range t.Unique {
		if len(uk.Cols) < 2 {
			continue
		}
		confidence := 0.8
		source := "unique_constraint_upper_bound"
		if uk.Id == "PRIMARY" {
			confidence = 1
			source = "primary_key"
		}
		multiColumnDistinct = append(multiColumnDistinct, scm.NewSlice([]scm.Scmer{
			scm.NewString("Columns"), showStringSlice(uk.Cols),
			scm.NewString("Estimate"), scm.NewInt(int64(t.CountEstimate())),
			scm.NewString("Confidence"), scm.NewFloat(confidence),
			scm.NewString("Source"), scm.NewString(source),
		}))
	}
	foreignKeys := make([]scm.Scmer, 0, len(t.Foreign))
	fanouts := make([]scm.Scmer, 0, len(t.Foreign))
	for _, fk := range t.Foreign {
		role := "referenced"
		localColumns := fk.Cols2
		otherTable := fk.Tbl1
		otherColumns := fk.Cols1
		if fk.Tbl1 == t.Name {
			role = "referencing"
			localColumns = fk.Cols1
			otherTable = fk.Tbl2
			otherColumns = fk.Cols2
		}
		foreignKeys = append(foreignKeys, scm.NewSlice([]scm.Scmer{
			scm.NewString("Id"), scm.NewString(fk.Id),
			scm.NewString("Role"), scm.NewString(role),
			scm.NewString("LocalColumns"), showStringSlice(localColumns),
			scm.NewString("OtherTable"), scm.NewString(otherTable),
			scm.NewString("OtherColumns"), showStringSlice(otherColumns),
			scm.NewString("UpdateMode"), scm.NewString(showForeignKeyMode(fk.Updatemode)),
			scm.NewString("DeleteMode"), scm.NewString(showForeignKeyMode(fk.Deletemode)),
		}))
		fanoutEstimate := scm.NewNil()
		confidenceValue := 0.0
		sourceValue := "unknown"
		if role == "referencing" {
			distinct, confidence, source := plannerDistinctForColumns(t, localColumns)
			if distinct > 0 {
				fanoutEstimate = scm.NewFloat(float64(t.CountEstimate()) / distinct)
				confidenceValue = confidence
				sourceValue = source
			}
		}
		fanouts = append(fanouts, scm.NewSlice([]scm.Scmer{
			scm.NewString("Id"), scm.NewString(fk.Id),
			scm.NewString("Role"), scm.NewString(role),
			scm.NewString("EstimatedFanout"), fanoutEstimate,
			scm.NewString("Confidence"), scm.NewFloat(confidenceValue),
			scm.NewString("Source"), scm.NewString(sourceValue),
		}))
	}
	partitions := make([]scm.Scmer, 0)
	if t.ShardMode == ShardModePartition {
		for _, sd := range t.PDimensions {
			partitions = append(partitions, scm.NewSlice([]scm.Scmer{
				scm.NewString("Column"), scm.NewString(sd.Column),
				scm.NewString("NumPartitions"), scm.NewInt(int64(sd.NumPartitions)),
				scm.NewString("Pivots"), scm.NewSlice(sd.Pivots),
			}))
		}
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("Name"), scm.NewString(t.Name),
		scm.NewString("Engine"), scm.NewString(engine),
		scm.NewString("Collation"), scm.NewString(t.Collation),
		scm.NewString("Charset"), scm.NewString(t.Charset),
		scm.NewString("Comment"), scm.NewString(t.Comment),
		scm.NewString("AutoIncrement"), scm.NewInt(int64(nextAutoIncrement)),
		scm.NewString("Unique"), scm.NewSlice(uniques),
		scm.NewString("ForeignKeys"), scm.NewSlice(foreignKeys),
		scm.NewString("Fanout"), scm.NewSlice(fanouts),
		scm.NewString("MultiColumnDistinct"), scm.NewSlice(multiColumnDistinct),
		scm.NewString("Partitions"), scm.NewSlice(partitions),
		scm.NewString("MaintenanceClass"), scm.NewString(maintenance.class),
		scm.NewString("CanDrop"), scm.NewBool(maintenance.canDrop),
		scm.NewString("CanTruncate"), scm.NewBool(maintenance.canTruncate),
		scm.NewString("CanRename"), scm.NewBool(maintenance.canRename),
		scm.NewString("CanAlter"), scm.NewBool(maintenance.canAlter),
		scm.NewString("CanChangeEngine"), scm.NewBool(maintenance.canChangeEngine),
	})
}

// showBuildShardRow builds the overview assoc for a single shard (nil shard is represented as all-zero/nil state).
func showBuildShardRow(t *table, i int, s *storageShard) scm.Scmer {
	if s == nil {
		return scm.NewSlice([]scm.Scmer{
			scm.NewString("shard"), scm.NewInt(int64(i)),
			scm.NewString("state"), scm.NewString("nil"),
			scm.NewString("main_count"), scm.NewInt(0),
			scm.NewString("delta"), scm.NewInt(0),
			scm.NewString("deletions"), scm.NewInt(0),
			scm.NewString("size_bytes"), scm.NewInt(0),
		})
	}
	stats := s.statsSnapshot()
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("shard"), scm.NewInt(int64(i)),
		scm.NewString("state"), scm.NewString(sharedStateStr(stats.state)),
		scm.NewString("main_count"), scm.NewInt(int64(stats.mainCount)),
		scm.NewString("delta"), scm.NewInt(int64(stats.delta)),
		scm.NewString("deletions"), scm.NewInt(int64(stats.deletions)),
		scm.NewString("size_bytes"), scm.NewInt(int64(stats.size)),
	})
}
