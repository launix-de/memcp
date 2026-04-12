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
import "io"
import "unsafe"
import "github.com/carli2/hybridsort"
import "sync"
import "sync/atomic"
import "time"
import "strconv"
import "reflect"
import "strings"
import units "github.com/docker/go-units"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// ColumnReader provides sequential-access-optimized reads. Returned by
// ColumnStorage.GetCachedReader(). Must not be shared between goroutines.
type ColumnReader interface {
	GetValue(uint32) scm.Scmer
}

type ColumnReaderFunc func(uint32) scm.Scmer

func (f ColumnReaderFunc) GetValue(idx uint32) scm.Scmer {
	return f(idx)
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
			panic("invalid partition column pair")
		}
		normalized = append(normalized, pair[0], pair[1])
	}
	return normalized
}

// THE basic storage pattern
type ColumnStorage interface {
	// info
	GetValue(uint32) scm.Scmer     // read function (concurrent-safe, no mutable state)
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
	if v.IsSlice() {
		return v.Slice(), true
	}
	return nil, false
}

func mustScmerSlice(v scm.Scmer, ctx string) []scm.Scmer {
	if slice, ok := scmerSlice(v); ok {
		return slice
	}
	panic(ctx + ": expected list")
}

func scmerSliceToStrings(list []scm.Scmer) []string {
	out := make([]string, len(list))
	for i, item := range list {
		out[i] = scm.String(item)
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

// lockTable acquires a user-level read or write lock on the named table.
// The session's State is updated while waiting, and the unlock callback is
// registered with the session so that ReleaseAllLocks() can free it later.
// Acquiring any lock requires an exclusive wait (drain other owners), then
// drains in-flight shard readers by briefly acquiring each shard's write lock.
func lockTable(schema, name string, write bool, ss *scm.SessionState) {
	db := GetDatabase(schema)
	if db == nil {
		panic("LOCK TABLES: unknown database: " + schema)
	}
	t := db.GetTable(name)
	if t == nil {
		panic("LOCK TABLES: unknown table: " + schema + "." + name)
	}
	// LOCK TABLES itself is serialized FIFO per table. This avoids a thundering
	// herd when many sessions repeatedly lock the same hot table (e.g. cron).
	cond := t.getTableLockCond()
	if ss != nil {
		ss.BeginLockWait()
		defer ss.EndLockWait()
		ss.SetState("Waiting for table lock")
	}
	t.tableLockMu.Lock()
	myTicket := t.tableLockNext
	t.tableLockNext++
	for myTicket != t.tableLockServe || t.tableLockOwner.Load() != nil {
		cond.Wait()
	}
	t.tableLockMu.Unlock()
	// Drain in-flight shard readers by acquiring each shard's write lock.
	// We MUST set tableLockOwner while still holding the last shard write lock
	// so that any new scan that does RLock → tableLockOwner.Load() sees the
	// owner set before it can proceed past its own RLock.
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
	for _, s := range shards {
		s.mu.Lock()
	}
	t.tableLockWrite.Store(write)
	t.tableLockOwner.Store(ss)
	for _, s := range shards {
		s.mu.Unlock()
	}
	acquired = true
	if ss != nil {
		ss.AddLock(t.unlockTable)
	}
}

func Init(en scm.Env) {
	scm.DeclareTitle("Storage")

	// Register TagTable serializer for the printer.
	scm.CustomStringer[TagTable] = func(ptr unsafe.Pointer) string {
		return (*table)(ptr).String()
	}

	scm.Declare(&en, &scm.Declaration{
		Name: "table",
		Desc: "resolves a schema+table name pair into a table handle",
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
		Type: &scm.TypeDescriptor{
			Kind: "func",
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema"},
				{Kind: "string", ParamName: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "table"},
			Optimize: func(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
				// Do NOT fold to a constant pointer — DDL (DROP/CREATE TABLE)
				// invalidates the pointer and cached query plans would reference
				// a stale table. Runtime evaluation via GetTable is lock-free.
				for i := 1; i < len(v); i++ {
					v[i], _ = oc.OptimizeSub(v[i], true)
				}
				return scm.NewSlice(v), nil
			},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "scan_estimate",
		Desc: "estimate output row count for a table scan",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewInt(int64(t.CountEstimate()))
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "int"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "table_empty?",
		Desc: "returns true if a table currently has no rows",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewBool(t.CountEstimate() == 0)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "scan",
		Desc: "does an unordered parallel filter-map-reduce pass on a single table and returns the reduced result",
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

			t := TableFromScmer(a[layout.tableIdx])

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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "any", ParamName: "tx", ParamDesc: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list", ParamName: "table", ParamDesc: "table handle or a list for temporary data"},
				{Kind: "list", ParamName: "filterColumns", ParamDesc: "list of columns that are fed into filter"},
				{Kind: "func", ParamName: "filter", ParamDesc: "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}},
				{Kind: "list", ParamName: "mapColumns", ParamDesc: "list of columns that are fed into map"},
				{Kind: "func", ParamName: "map", ParamDesc: "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns.", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "func", Params: []*scm.TypeDescriptor{{Transfer: true}, nil}, ParamName: "reduce", ParamDesc: "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass.", Optional: true},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				{Kind: "func", Params: []*scm.TypeDescriptor{{Transfer: true}, nil}, ParamName: "reduce2", ParamDesc: "(optional) second stage reduce function that will apply a result of reduce to the neutral element/accumulator", Optional: true},
				{Kind: "bool", ParamName: "isOuter", ParamDesc: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
			},
			Return:   &scm.TypeDescriptor{Kind: "any"},
			Optimize: optimizeScan,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_batch",
		Desc: "does an unordered parallel filter-map-reduce pass on a single table using batchdata-backed #N pseudo columns and returns the reduced result",
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

			t := TableFromScmer(a[layout.tableIdx])

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
			return t.scanWithBatch(layout.tx, filtercols, a[layout.filterFnIdx], mapcols, a[layout.mapFnIdx], aggregate, neutral, reduce2, isOuter, stride, batchdata)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "any", ParamName: "tx", ParamDesc: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list", ParamName: "table", ParamDesc: "table handle or a list for temporary data"},
				{Kind: "list", ParamName: "filterColumns", ParamDesc: "list of columns that are fed into filter; #0, #1, ... address batchdata slots"},
				{Kind: "func", ParamName: "filter", ParamDesc: "lambda function that decides whether a dataset is passed to the map phase", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}},
				{Kind: "list", ParamName: "mapColumns", ParamDesc: "list of columns that are fed into map; #0, #1, ... address batchdata slots"},
				{Kind: "func", ParamName: "map", ParamDesc: "lambda function to extract data from the dataset", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "int", ParamName: "stride", ParamDesc: "number of batchdata entries per batch row"},
				{Kind: "list", ParamName: "batchdata", ParamDesc: "flat batch buffer accessed via #N pseudo columns"},
				{Kind: "func", Params: []*scm.TypeDescriptor{{Transfer: true}, nil}, ParamName: "reduce", ParamDesc: "(optional) lambda function to aggregate the map results", Optional: true},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				{Kind: "func", Params: []*scm.TypeDescriptor{{Transfer: true}, nil}, ParamName: "reduce2", ParamDesc: "(optional) second stage reduce function that will apply a result of reduce to the neutral element/accumulator", Optional: true},
				{Kind: "bool", ParamName: "isOuter", ParamDesc: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
			},
			Return:   &scm.TypeDescriptor{Kind: "any"},
			Optimize: optimizeScanBatch,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_order",
		Desc: "does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result",
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

			if list, ok := scmerSlice(tableArg); ok {
				result := neutral
				filterfn := scm.OptimizeProcToSerialFunction(a[layout.filterFnIdx])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[layout.limitIdx+2])
				mapparams := make([]scm.Scmer, len(mapcols))
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
				for idx, val := range filtered {
					if idx < offset {
						continue
					}
					if limit >= 0 && count >= limit {
						break
					}
					row := mustScmerSlice(val, "scan_order row")
					ds := dataset(row)
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
				return result
			}

			t := TableFromScmer(a[layout.tableIdx])

			return t.scan_order(layout.tx, filtercols, a[layout.filterFnIdx], sortcolsVals, sortdirs, limitPartitionCols, scm.ToInt(a[layout.offsetIdx]), scm.ToInt(a[layout.limitIdx]), mapcols, a[layout.limitIdx+2], aggregate, neutral, isOuter)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "any", ParamName: "tx", ParamDesc: "transaction context to use for visibility and mutations; usually ((context \"session\") \"__memcp_tx\")"},
				{Kind: "table|list", ParamName: "table", ParamDesc: "table handle or a list for temporary data"},
				{Kind: "list", ParamName: "filterColumns", ParamDesc: "list of columns that are fed into filter"},
				{Kind: "func", ParamName: "filter", ParamDesc: "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}},
				{Kind: "list", ParamName: "sortcols", ParamDesc: "list of columns to sort. Each column is either a string to point to an existing column or a func(cols...)->any to compute a sortable value"},
				{Kind: "list", ParamName: "sortdirs", ParamDesc: "list of column directions to sort. Must be same length as sortcols. < means ascending, > means descending, (collate ...) will add collations"},
				{Kind: "number", ParamName: "limitPartitionCols", ParamDesc: "number of leading sort columns that form the partition key for per-partition offset/limit. 0 (default) means global offset/limit."},
				{Kind: "number", ParamName: "offset", ParamDesc: "number of items to skip before the first one is fed into map"},
				{Kind: "number", ParamName: "limit", ParamDesc: "max number of items to read"},
				{Kind: "list", ParamName: "mapColumns", ParamDesc: "list of columns that are fed into map"},
				{Kind: "func", ParamName: "map", ParamDesc: "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns.", Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "func", ParamName: "reduce", ParamDesc: "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass.", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "acc", Transfer: true}, {Kind: "any", ParamName: "val"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) neutral element for the reduce phase, otherwise nil is assumed", Optional: true},
				{Kind: "bool", ParamName: "isOuter", ParamDesc: "(optional) if true, in case of no hits, call map once anyway with NULL values", Optional: true},
			},
			Return:   &scm.TypeDescriptor{Kind: "any"},
			Optimize: optimizeScanOrder,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "scan_order_multi",
		Desc: "does an ordered parallel filter and serial map-reduce pass across multiple tables simultaneously, merging results into a single sorted stream",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			// Parameters:
			// 0: tx
			// 1: tables list (table handles)
			// 2: filterColumns per table (list of lists)
			// 3: filterFns per table (list of lambdas)
			// 4: sortcols per table (list of lists)
			// 5: sortdirs (shared list)
			// 6: limitPartitionCols
			// 7: offset
			// 8: limit
			// 9: mapColumns per table (list of lists)
			// 10: mapFns per table (list of lambdas)
			// 11: reduce (optional)
			// 12: neutral (optional)
			// 13: isOuter (optional)
			currentTx := scmerToTxContext(a[0])
			tables := mustScmerSlice(a[1], "tables")
			filterColsArr := mustScmerSlice(a[2], "filterColumns")
			filterFnArr := mustScmerSlice(a[3], "filterFns")
			sortcolsArr := mustScmerSlice(a[4], "sortcols")
			sortdirsVals := mustScmerSlice(a[5], "sortdirs")
			limitPartitionCols := scm.ToInt(a[6])
			offset := scm.ToInt(a[7])
			limit := scm.ToInt(a[8])
			mapColsArr := mustScmerSlice(a[9], "mapColumns")
			mapFnArr := mustScmerSlice(a[10], "mapFns")

			aggregate := scm.NewNil()
			if len(a) > 11 {
				aggregate = a[11]
			}
			neutral := scm.NewNil()
			if len(a) > 12 {
				neutral = a[12]
			}
			isOuter := len(a) > 13 && scm.ToBool(a[13])

			n := len(tables)
			if len(filterColsArr) != n || len(filterFnArr) != n || len(sortcolsArr) != n || len(mapColsArr) != n || len(mapFnArr) != n {
				panic("scan_order_multi: all per-table arrays must have the same length")
			}

			sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortdirsVals))
			for i, dir := range sortdirsVals {
				sortdirs[i] = scm.OptimizeProcToSerialFunction(dir)
			}

			specs := make([]scanOrderTableSpec, n)
			for i := 0; i < n; i++ {
				t := TableFromScmer(tables[i])
				specs[i] = scanOrderTableSpec{
					table:         t,
					conditionCols: scmerSliceToStrings(mustScmerSlice(filterColsArr[i], "filterColumns[i]")),
					condition:     filterFnArr[i],
					sortcols:      mustScmerSlice(sortcolsArr[i], "sortcols[i]"),
					callbackCols:  scmerSliceToStrings(mustScmerSlice(mapColsArr[i], "mapColumns[i]")),
					callback:      mapFnArr[i],
				}
			}

			return scanOrderMulti(currentTx, specs, sortdirs, int(limitPartitionCols), int(offset), int(limit), aggregate, neutral, isOuter)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "any", ParamName: "tx", ParamDesc: "transaction context"},
				{Kind: "list", ParamName: "tables", ParamDesc: "list of table handles"},
				{Kind: "list", ParamName: "filterColumns", ParamDesc: "list of filter column lists, one per table"},
				{Kind: "list", ParamName: "filterFns", ParamDesc: "list of filter lambdas, one per table"},
				{Kind: "list", ParamName: "sortcols", ParamDesc: "list of sort column lists, one per table"},
				{Kind: "list", ParamName: "sortdirs", ParamDesc: "list of sort direction comparators (shared)"},
				{Kind: "number", ParamName: "limitPartitionCols", ParamDesc: "number of leading sort columns forming partition key"},
				{Kind: "number", ParamName: "offset", ParamDesc: "number of items to skip"},
				{Kind: "number", ParamName: "limit", ParamDesc: "max number of items to read"},
				{Kind: "list", ParamName: "mapColumns", ParamDesc: "list of map column lists, one per table"},
				{Kind: "list", ParamName: "mapFns", ParamDesc: "list of map lambdas, one per table"},
				{Kind: "func", ParamName: "reduce", ParamDesc: "(optional) aggregation function", Optional: true},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) neutral element for reduce", Optional: true},
				{Kind: "bool", ParamName: "isOuter", ParamDesc: "(optional) if true, emit null row when no hits", Optional: true},
			},
			Return:   &scm.TypeDescriptor{Kind: "any"},
			Optimize: optimizeScanOrderMulti,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createdatabase",
		Desc: "creates a new database",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ignoreexists := len(a) > 1 && scm.ToBool(a[1])
			return scm.NewBool(CreateDatabase(scm.String(a[0]), ignoreexists))
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the new database"},
				{Kind: "bool", ParamName: "ignoreexists", ParamDesc: "if true, return false instead of throwing an error", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropdatabase",
		Desc: "drops a database",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifexists := len(a) > 1 && scm.ToBool(a[1])
			return scm.NewBool(DropDatabase(scm.String(a[0]), ifexists))
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database"},
				{Kind: "bool", ParamName: "ifexists", ParamDesc: "if true, don't throw an error if it doesn't exist", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createtable",
		Desc: "creates a new database",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifnotexists := len(a) > 4 && scm.ToBool(a[4])
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
					atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
					return scm.NewBool(false)
				}
			}

			// parse options only after the fast existing-table probe
			options := mustScmerSlice(a[3], "options")
			var autoIncrement uint64
			engine := Settings.DefaultEngine
			collation := ""
			charset := ""
			comment := ""
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
					autoIncrement, _ = strconv.ParseUint(scm.String(val), 0, 64)
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
			newTable.Auto_increment = 1
			newTable.Collation = collation
			newTable.Charset = charset
			newTable.Comment = comment
			newTable.Auto_increment = autoIncrement

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

			db.schemalock.Lock()
			existing := db.tables.Get(tblName)
			if existing != nil {
				if !ifnotexists {
					db.schemalock.Unlock()
					panic("Table " + tblName + " already exists")
				}
				// Keep the hot ifnotexists path free of schema saves. Planner-created
				// helper tables deliberately re-issue createtable on every query; if
				// the table already exists, "created=false" is the only signal the
				// caller needs to skip collect/materialization work.
				atomic.StoreUint64(&existing.lastAccessed, uint64(time.Now().UnixNano()))
				db.schemalock.Unlock()
				return scm.NewBool(false)
			}

			if prev := db.tables.Set(newTable); prev != nil {
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
			db.saveLockedWithDurabilityAndUnlock(newTable.PersistencyMode == Safe)
			registerCreatedTable(newTable)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database"},
				{Kind: "string", ParamName: "table", ParamDesc: "name of the new table"},
				{Kind: "list", ParamName: "cols", ParamDesc: "list of columns and constraints, each '(\"column\" colname typename dimensions typeparams) where dimensions is a list of 0-2 numeric items or '(\"primary\" cols) or '(\"unique\" cols) or '(\"foreign\" cols tbl2 cols2 updatemode deletemode of 'restrict'|'cascade'|'set null')"},
				{Kind: "list", ParamName: "options", ParamDesc: "further options like engine=safe|sloppy|memory"},
				{Kind: "bool", ParamName: "ifnotexists", ParamDesc: "don't throw an error if table already exists", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createcolumn",
		Desc: "creates a new column in table",
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
			var orcMapCols []string
			var orcMapFn, orcReduceFn, orcReduceInit scm.Scmer
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
			// 4. filtercols/filter further narrow which keys need eager materialization;
			//    they do not change the identity of the canonical temp column itself.
			if len(orcSortCols) > 0 {
				t.computeOrderedColumnDDLLocked(colname, orcSortCols, orcSortDirs, 0, orcMapCols, orcMapFn, orcReduceFn, orcReduceInit)
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
				t.computeColumnDDLLocked(colname, paramNames, a[6], filterCols, filter)
				return scm.NewBool(true)
			}

			return scm.NewBool(created)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "colname", ParamDesc: "name of the new column"},
				{Kind: "string", ParamName: "type", ParamDesc: "name of the basetype"},
				{Kind: "list", ParamName: "dimensions", ParamDesc: "dimensions of the type (e.g. for decimal)"},
				{Kind: "list", ParamName: "options", ParamDesc: "assoc list: primary, unique, auto_increment, null, comment, default, collate; ORC: sortcols, sortdirs, partitioncount, mapcols, mapfn, reducefn, reduceinit"},
				{Kind: "list", ParamName: "computorCols", ParamDesc: "list of columns that is passed into params of computor", Optional: true},
				{Kind: "func", ParamName: "computor", ParamDesc: "lambda expression that can take other column values and computes the value of that column", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "any", ParamName: "columns", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any"}},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createkey",
		Desc: "creates a new key on a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])

			if !scm.ToBool(a[2]) {
				return scm.NewBool(true)
			}

			cols := scmerSliceToStrings(mustScmerSlice(a[3], "unique columns"))
			name := scm.String(a[1])

			t.schema.schemalock.Lock()
			for _, u := range t.Unique {
				if u.Id == name {
					t.schema.schemalock.Unlock()
					return scm.NewBool(false)
				}
			}

			t.Unique = append(t.Unique, uniqueKey{name, cols})
			t.schema.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)

			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "keyname", ParamDesc: "name of the new key"},
				{Kind: "bool", ParamName: "unique", ParamDesc: "whether the key is unique"},
				{Kind: "list", ParamName: "columns", ParamDesc: "list of columns to include"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "createforeignkey",
		Desc: "creates a new foreign key on a table",
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

			db.saveLockedWithDurabilityAndUnlock(t1.PersistencyMode == Safe || t2.PersistencyMode == Safe)

			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table1"},
				{Kind: "string", ParamName: "keyname", ParamDesc: "name of the new key"},
				{Kind: "list", ParamName: "columns1", ParamDesc: "list of columns to include"},
				{Kind: "table", ParamName: "table2"},
				{Kind: "list", ParamName: "columns2", ParamDesc: "list of columns to include"},
				{Kind: "string", ParamName: "updatemode", ParamDesc: "restrict|cascade|set null"},
				{Kind: "string", ParamName: "deletemode", ParamDesc: "restrict|cascade|set null"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "shardcolumn",
		Desc: "tells us how it would partition a column according to their values. Returns a list of pivot elements.",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "colname", ParamDesc: "name of the column"},
				{Kind: "number", ParamName: "numpartitions", ParamDesc: "number of partitions; optional. leave 0 if you want to detect the partiton number automatically or copy the partition schema of the table", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "list"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "partitiontable",
		Desc: "suggests a partition scheme for a table. If the table has no partition scheme yet, it will immediately apply that scheme and return true. If the table already has a partition scheme, it will alter the partitioning score such that the partitioning scheme is considered in the next repartitioning and return false.",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "list", ParamName: "columns", ParamDesc: "associative list of string -> list representing column name -> pivots. You can compute pivots by (shardcolumn ...)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "altertable",
		Desc: "alters a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			db := t.schema

			switch scm.String(a[1]) {
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
			default:
				panic("unimplemented alter table operation: " + scm.String(a[1]))
			}
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "operation", ParamDesc: "one of owner|drop|engine|collation"},
				{Kind: "any", ParamName: "parameter", ParamDesc: "name of the column to drop or value of the parameter"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "altercolumn",
		Desc: "alters a column",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			db := t.schema
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
							t.Auto_increment = uint64(ai)
							db.save()
							return scm.NewBool(true)
						}
						t.Columns[i].AutoIncrement = scm.ToBool(a[3])
						db.save()
						return scm.NewBool(true)
					default:
						ok := t.Columns[i].Alter(scm.String(a[2]), a[3])
						db.save()
						return scm.NewBool(scm.ToBool(ok))
					}
				}
			}
			panic("column " + t.schema.Name + "." + t.Name + "." + scm.String(a[1]) + " does not exist")
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "column", ParamDesc: "name of the column"},
				{Kind: "string", ParamName: "operation", ParamDesc: "one of drop|type|collation|auto_increment|comment"},
				{Kind: "any", ParamName: "parameter", ParamDesc: "name of the column to drop or value of the parameter"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "droptable",
		Desc: "removes a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ifexists := len(a) > 2 && scm.ToBool(a[2])
			DropTable(scm.String(a[0]), scm.String(a[1]), ifexists)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema"},
				{Kind: "string", ParamName: "table"},
				{Kind: "bool", ParamName: "ifexists", ParamDesc: "if true, don't throw an error if it already exists", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "dropcolumn",
		Desc: "drops a column from a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			return scm.NewBool(t.DropColumn(scm.String(a[1])))
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "column", ParamDesc: "name of the column to drop"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "invalidatecolumn",
		Desc: "marks all values of a computed column as stale",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			t := TableFromScmer(a[0])
			colName := scm.String(a[1])
			for _, s := range t.maintenanceShards() {
				s.mu.RLock()
				col := s.columns[colName]
				s.mu.RUnlock()
				if proxy, ok := col.(*StorageComputeProxy); ok {
					proxy.InvalidateAll()
				}
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "column", ParamDesc: "name of the computed column"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "invalidateorc",
		Desc: "invalidates ORC column rows from a sort key onwards via validMask scan",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "column", ParamDesc: "name of the ORC column"},
				{Kind: "list", ParamName: "sortkeys", ParamDesc: "composite sort key values from which to invalidate"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "register_keytable_cleanup",
		Desc: "registers triggers on a base table to maintain keytable entries (insert/delete group keys)",
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

			// Helper: build (get_assoc <sym> "col") list
			getAssocs := func(sym string, cols []string) []scm.Scmer {
				result := make([]scm.Scmer, len(cols))
				for i, col := range cols {
					result[i] = fkGetAssocExpr(sym, col)
				}
				return result
			}

			// Build count-scan: (scan base_schema base_table (list base_cols...) (lambda (tblvar.col...) (and (equal? tblvar.col (get_assoc OLD "col")) ...)) () (lambda () 1) + 0 nil)
			buildCountScan := func(dictSym string) scm.Scmer {
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("scan"),
					scm.NewSymbol("session"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(baseSchema), scm.NewString(baseTable.Name)}),
					scanFilterCols(baseCols),
					scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("lambda"), scanFilterParams(tblvar, baseCols)},
						buildAndEquals(scanParamSyms(tblvar, baseCols), getAssocs(dictSym, baseCols)))),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice([]scm.Scmer{}), scm.NewInt(1)}),
					scm.NewSymbol("+"), scm.NewInt(0), scm.NewNil(),
				})
			}

			// Build delete-scan: (scan kt_schema kt_name (list kt_cols...) (lambda (kt.col...) (and (equal? kt.col (get_assoc OLD "base_col")) ...)) (list "$update") (lambda ($update) ($update)) + 0 nil)
			buildDeleteScan := func(dictSym string) scm.Scmer {
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("scan"),
					scm.NewSymbol("session"),
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
				return scm.NewSlice([]scm.Scmer{
					scm.NewSymbol("insert"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("table"), scm.NewString(ktSchema), scm.NewString(ktName)}),
					scanFilterCols(ktCols),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list"),
						fkValListExpr(dictSym, baseCols)}),
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
					continue
				}
				baseTable.AddTrigger(TriggerDescription{
					Name:     triggerName,
					Timing:   td.timing,
					IsSystem: true,
					Priority: 90, // run before invalidatecolumn (100) so keys are current when values recompute
					Func:     buildFKProc(td.body),
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
					continue
				}
				baseTable.AddTrigger(TriggerDescription{
					Name:     triggerName,
					Timing:   timing,
					IsSystem: true,
					Priority: 90,
					Func:     buildFKProc(dropBody),
				})
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "base_table"},
				{Kind: "table", ParamName: "kt_table"},
				{Kind: "string", ParamName: "tblvar", ParamDesc: "table alias used in scan column prefixes"},
				{Kind: "list", ParamName: "key_pairs", ParamDesc: "list of (base_col kt_col) pairs"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "touch_keytable",
		Desc: "extends the lease on a keytable so CacheManager defers eviction",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "locktables",
		Desc: "acquires WRITE or READ user-level locks on a list of tables (LOCK TABLES); implicitly releases any previously held locks",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			ss := scm.GetCurrentSessionState()
			if ss != nil {
				ss.ReleaseAllLocks() // LOCK TABLES implicitly releases prior locks
			}
			for _, item := range a[0].Slice() {
				triple := item.Slice()
				schema := scm.String(triple[0])
				tbl := scm.String(triple[1])
				write := triple[2].Bool()
				lockTable(schema, tbl, write, ss)
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "list", ParamName: "locks", ParamDesc: "flat list of schema, table, write? triples"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "unlocktables",
		Desc: "releases all user-level table locks held by this session",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if ss := scm.GetCurrentSessionState(); ss != nil {
				ss.ReleaseAllLocks()
			}
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "get_fk_target",
		Desc: "returns (ref_table ref_column) if a single-column FK exists for the given column, nil otherwise",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "column", ParamDesc: "column name"},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "renametable",
		Desc: "renames a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			RenameTable(scm.String(a[0]), scm.String(a[1]), scm.String(a[2]))
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database"},
				{Kind: "string", ParamName: "oldname", ParamDesc: "current name of the table"},
				{Kind: "string", ParamName: "newname", ParamDesc: "new name of the table"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "insert",
		Desc: "inserts a new dataset into table and returns the number of successful items",
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
			inserted := t.Insert(cols, rows, onCollisionCols, onCollision, mergeNull, onFirst)
			return scm.NewInt(int64(inserted))
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "list", ParamName: "columns", ParamDesc: "list of column names, e.g. '(\"ID\", \"value\")"},
				{Kind: "list", ParamName: "datasets", ParamDesc: "list of list of column values, e.g. '('(1 10) '(2 15))"},
				{Kind: "list", ParamName: "onCollisionCols", ParamDesc: "list of columns of the old dataset that have to be passed to onCollision. Can also request $update.", Optional: true},
				{Kind: "func", ParamName: "onCollision", ParamDesc: "the function that is called on each collision dataset. The first parameter is filled with the $update function, the second parameter is the dataset as associative list. If not set, an error is thrown in case of a collision.", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "func", ParamName: "$update"}, {Kind: "list", ParamName: "dataset"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
				{Kind: "bool", ParamName: "mergeNull", ParamDesc: "if true, it will handle NULL values as equal according to SQL 2003's definition of DISTINCT (https://en.wikipedia.org/wiki/Null_(SQL)#When_two_nulls_are_equal:_grouping,_sorting,_and_some_set_operations)", Optional: true},
				{Kind: "func", ParamName: "onInsertid", ParamDesc: "(optional) callback (id)->any; called once with the first auto_increment id assigned for this INSERT", Optional: true, Params: []*scm.TypeDescriptor{{Kind: "number", ParamName: "id"}}, Return: &scm.TypeDescriptor{Kind: "any"}},
			},
			Return: &scm.TypeDescriptor{Kind: "number"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "stat",
		Desc: "return system statistics as assoc: mem_available, mem_total, process_memory, shard_memory, shard_budget, persisted_memory, persisted_budget, cache_entry_count, cache_entry_size.\n(stat schema) and (stat schema tbl) return a string with detailed memory usage.",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				memTotal, memAvail := ReadMemInfo()
				processMem := ReadProcessRSS()
				cs := GlobalCache.Stat()
				return scm.NewSlice([]scm.Scmer{
					scm.NewString("mem_available"), scm.NewInt(memAvail),
					scm.NewString("mem_total"), scm.NewInt(memTotal),
					scm.NewString("process_memory"), scm.NewInt(processMem),
					scm.NewString("shard_memory"), scm.NewInt(cs.CurrentMemory),
					scm.NewString("shard_budget"), scm.NewInt(cs.MemoryBudget),
					scm.NewString("persisted_memory"), scm.NewInt(cs.PersistedMemory),
					scm.NewString("persisted_budget"), scm.NewInt(cs.PersistedBudget),
					scm.NewString("cache_entry_count"), scm.NewInt(cs.CountByType[TypeCacheEntry]),
					scm.NewString("cache_entry_size"), scm.NewInt(cs.SizeByType[TypeCacheEntry]),
				})
			} else if len(a) == 1 {
				return scm.NewString(GetDatabase(scm.String(a[0])).PrintMemUsage())
			} else if len(a) == 1 && a[0].IsCustom(TagTable) {
				return scm.NewString(TableFromScmer(a[0]).PrintMemUsage())
			} else if len(a) == 2 {
				return scm.NewString(GetDatabase(scm.String(a[0])).GetTable(scm.String(a[1])).PrintMemUsage())
			}
			return scm.NewNil()
		},
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "(optional) database name for detailed string output", Optional: true},
				{Kind: "string", ParamName: "table", ParamDesc: "(optional) table name for detailed string output", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "totalmem",
		Desc: "Returns total physical memory in bytes (from /proc/meminfo)",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			return scm.NewInt(totalMemoryBytes())
		},
		Type: &scm.TypeDescriptor{
			Return: &scm.TypeDescriptor{Kind: "number"},
			Const:  true,
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "show",
		Desc: "show databases/tables/columns/shards\n\n(show) lists database names\n(show schema) lists table names\n(show schema true) lists tables with full info: [{name,engine,row_count,size_bytes,collation,comment},...]\n(show schema tbl) lists column defs\n(show schema tbl true) returns assoc {columns,meta,shards}\n(show schema tbl N) returns shard N overview assoc {shard,state,main_count,delta,deletions,size_bytes}\n(show schema tbl N true) returns shard N full assoc adding columns and indexes\n(show schema tbl \"statistics\") returns index statistics (used by INFORMATION_SCHEMA)",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			// table-based overloads: (show table) → columns, (show table "statistics") → index stats, etc.
			if len(a) >= 1 && a[0].IsCustom(TagTable) {
				t := TableFromScmer(a[0])
				if len(a) == 1 {
					return t.ShowColumns()
				}
				if len(a) == 2 {
					if a[1].IsString() && scm.String(a[1]) == "statistics" {
						return showBuildMeta(t.schema, t) // reuse existing statistics builder
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
						var rowCount int64
						var sizeBytes int64
						for _, s := range t.ActiveShards() {
							if s == nil {
								continue
							}
							s.mu.RLock()
							rowCount += int64(s.main_count) + int64(len(s.inserts)) - int64(s.deletions.Count())
							sizeBytes += int64(s.ComputeSize())
							s.mu.RUnlock()
						}
						rows = append(rows, scm.NewSlice([]scm.Scmer{
							scm.NewString("name"), scm.NewString(t.Name),
							scm.NewString("engine"), scm.NewString(engine),
							scm.NewString("row_count"), scm.NewInt(rowCount),
							scm.NewString("size_bytes"), scm.NewInt(sizeBytes),
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
				// (show schema tbl "statistics") → index statistics (INFORMATION_SCHEMA)
				if a[2].IsString() && scm.String(a[2]) == "statistics" {
					var result []scm.Scmer
					for _, uk := range t.Unique {
						for seq, col := range uk.Cols {
							result = append(result, scm.NewSlice([]scm.Scmer{
								scm.NewString("table_catalog"), scm.NewString("def"),
								scm.NewString("table_schema"), scm.NewString(db.Name),
								scm.NewString("table_name"), scm.NewString(t.Name),
								scm.NewString("non_unique"), scm.NewInt(0),
								scm.NewString("index_schema"), scm.NewString(db.Name),
								scm.NewString("index_name"), scm.NewString(uk.Id),
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
							idxSlice = append(idxSlice, scm.NewSlice([]scm.Scmer{
								scm.NewString("cols"), scm.NewString(ix.String()),
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "(optional) database name", Optional: true},
				{Kind: "string|bool", ParamName: "table", ParamDesc: "(optional) table name, or true for full table listing", Optional: true},
				{Kind: "int|bool", ParamName: "property", ParamDesc: "(optional) shard index (int), true for full table info, or \"statistics\"", Optional: true},
				{Kind: "bool", ParamName: "full", ParamDesc: "(optional) true to include columns and indexes in shard detail", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})
	// show_triggers(schema, table): returns a list of triggers for a table (non-system triggers only)
	scm.Declare(&en, &scm.Declaration{
		Name: "show_triggers",
		Desc: "show triggers for a given table",
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "database name"},
				{Kind: "string", ParamName: "table", ParamDesc: "(optional) table name, if omitted shows all triggers in schema", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "rebuild",
		Desc: "rebuilds all main storages and returns the amount of time it took",
		Fn: func(a ...scm.Scmer) scm.Scmer {
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "bool", ParamName: "all", ParamDesc: "if true, rebuild all shards, even if nothing has changed (default: false)", Optional: true},
				{Kind: "bool", ParamName: "repartition", ParamDesc: "if true, also repartition (default: true)", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	// clean() is intentionally not exposed as a SQL function.
	// It runs automatically at startup (in a background goroutine) after LoadDatabases().

	scm.Declare(&en, &scm.Declaration{
		Name: "loadCSV",
		Desc: "loads a CSV stream into a table and returns the amount of time it took.\nThe first line of the file must be the headlines. The headlines must match the table's columns exactly.",
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
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database"},
				{Kind: "string", ParamName: "table", ParamDesc: "name of the table"},
				{Kind: "stream", ParamName: "stream", ParamDesc: "CSV file, load with: (stream filename)"},
				{Kind: "string", ParamName: "delimiter", ParamDesc: "(optional) delimiter defaults to \";\"", Optional: true},
				{Kind: "bool", ParamName: "firstline", ParamDesc: "(optional) if the first line contains the column names (otherwise, the tables column order is used)", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "loadJSON",
		Desc: "loads a .jsonl file from stream into a database and returns the amount of time it took.\nJSONL is a linebreak separated file of JSON objects. Each JSON object is one dataset in the database. Before you add rows, you must declare the table in a line '#table <tablename>'. All other lines starting with # are comments. Columns are created dynamically as soon as they occur in a json object.",
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
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database where you want to put the tables in"},
				{Kind: "stream", ParamName: "stream", ParamDesc: "stream of the .jsonl file, read with: (stream filename)"},
			},
			Return: &scm.TypeDescriptor{Kind: "string"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "settings",
		Desc: "reads or writes a global settings value. This modifies your data/settings.json.",
		Fn:   ChangeSettings,
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "key", ParamDesc: "name of the key to set or get (for reference, rts)", Optional: true},
				{Kind: "any", ParamName: "value", ParamDesc: "new value of that setting", Optional: true},
			},
			Return: &scm.TypeDescriptor{Kind: "any"},
		},
	})

	// Trigger management
	scm.Declare(&en, &scm.Declaration{
		Name: "createtrigger",
		Desc: "creates a new trigger on a table",
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
			db.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)
			return scm.NewBool(true)
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "table", ParamName: "table"},
				{Kind: "string", ParamName: "name", ParamDesc: "name of the trigger"},
				{Kind: "string", ParamName: "timing", ParamDesc: "one of: before_insert, after_insert, before_update, after_update, before_delete, after_delete"},
				{Kind: "string", ParamName: "source_sql", ParamDesc: "original SQL body text (for SHOW TRIGGERS)"},
				{Kind: "any", ParamName: "body", ParamDesc: "trigger body (parsed Scheme expression)"},
				{Kind: "bool", ParamName: "visible", ParamDesc: "true = user trigger (shown in SHOW TRIGGERS), false = internal trigger (hidden)"},
			},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
	scm.Declare(&en, &scm.Declaration{
		Name: "droptrigger",
		Desc: "removes a trigger from a table",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				if scm.ToBool(a[2]) {
					return scm.NewBool(false)
				}
				panic("database " + scm.String(a[0]) + " does not exist")
			}

			name := scm.String(a[1])
			tables := db.tables.GetAll()
			// Search all tables for the trigger. Take the table-local DDL lock
			// before the database schemalock to keep the DDL lock order stable.
			for _, t := range tables {
				t.ddlMu.Lock()
				db.schemalock.Lock()
				if t.RemoveTrigger(name) {
					db.saveLockedWithDurabilityAndUnlock(t.PersistencyMode == Safe)
					t.ddlMu.Unlock()
					return scm.NewBool(true)
				}
				db.schemalock.Unlock()
				t.ddlMu.Unlock()
			}

			if scm.ToBool(a[2]) {
				return scm.NewBool(false)
			}
			panic("trigger " + name + " does not exist")
		},
		Type: &scm.TypeDescriptor{HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "name of the database"},
				{Kind: "string", ParamName: "name", ParamDesc: "name of the trigger"},
				{Kind: "bool", ParamName: "ifexists", ParamDesc: "don't throw error if trigger doesn't exist"},
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
		Desc: "Creates a new cachemap. Returns a threadsafe key-value function with LRU eviction under memory pressure: (cachemap key value) sets, (cachemap key) gets, (cachemap) lists keys.",
		Fn:   NewCacheMap,
		Type: &scm.TypeDescriptor{
			Return: &scm.TypeDescriptor{Kind: "func"},
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
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	var b strings.Builder
	if db.srState == COLD {
		b.WriteString("State: COLD (no schema loaded)\n")
		return b.String()
	}
	b.WriteString("Table                    \tColumns\tShards\tDims\tSize/Bytes\n")
	var dsize uint
	for _, t := range db.tables.GetAll() {
		var size uint = 10*8 + 32*uint(len(t.Columns))
		for _, s := range t.Shards {
			size += s.ComputeSize()
		}
		for _, s := range t.PShards {
			if s != nil {
				size += s.ComputeSize()
			}
		}
		b.WriteString(fmt.Sprintf("%-25s\t%d\t%d\t%d\t%s\n", t.Name, len(t.Columns), len(t.Shards)+len(t.PShards), len(t.PDimensions), units.BytesSize(float64(size))))
		dsize += size
	}
	b.WriteString(fmt.Sprintf("\ntotal size = %s\n", units.BytesSize(float64(dsize))))
	return b.String()
}

func (t *table) PrintMemUsage() string {
	var b strings.Builder
	var dsize uint = 0
	shards := t.ActiveShards()
	if t.ShardMode == ShardModePartition {
		b.WriteString(fmt.Sprint("Partitioning Schema:", t.PDimensions) + "\n\n")
	}
	for i, s := range shards {
		var ssz uint = 14 * 8 // overhead
		if s.srState == COLD {
			b.WriteString(fmt.Sprintf("Shard %d [COLD] (no content loaded)\n---\n\n", i))
			dsize += ssz
			continue
		}
		b.WriteString(fmt.Sprintf("Shard %d [%s]\n---\n", i, sharedStateStr(s.srState)))
		b.WriteString(fmt.Sprintf("main count: %d, delta count: %d, deletions: %d\n", s.main_count, len(s.inserts), s.deletions.Count()))
		for c, v := range s.columns {
			if v == nil {
				b.WriteString(fmt.Sprintf(" %s: COLD\n", c))
				continue
			}
			sz := v.ComputeSize()
			b.WriteString(fmt.Sprintf(" %s: %s, size = %s\n", c, v.String(), units.BytesSize(float64(sz))))
			ssz += sz
		}
		b.WriteString(" ---\n")
		for _, idx := range s.Indexes {
			indexSize := idx.ComputeSize()
			b.WriteString(fmt.Sprintf(" index %s: %s\n", idx.String(), units.BytesSize(float64(indexSize))))
			ssz += indexSize
		}
		b.WriteString(" ---\n")
		insertionSize := scm.ComputeSize(scm.NewAny(s.inserts))
		deletionSize := s.deletions.ComputeSize()
		ssz += insertionSize
		ssz += deletionSize
		b.WriteString(fmt.Sprintf(" + insertions %s\n", units.BytesSize(float64(insertionSize))))
		b.WriteString(fmt.Sprintf(" + deletions %s\n", units.BytesSize(float64(deletionSize))))
		b.WriteString(" ---\n")
		b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(ssz))))
		dsize += ssz
	}
	b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(dsize))))
	return b.String()
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
		Desc: "check that FK values exist in the parent table, panic if not",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := CurrentTx()
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "database name"},
				{Kind: "string", ParamName: "parent_table", ParamDesc: "parent table name"},
				{Kind: "list", ParamName: "parent_cols", ParamDesc: "parent column names"},
				{Kind: "list", ParamName: "values", ParamDesc: "FK values to check"},
				{Kind: "string", ParamName: "fk_id", ParamDesc: "FK constraint name"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "__fk_on_parent_delete",
		Desc: "enforce FK constraint when parent row is deleted",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := CurrentTx()
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "database name"},
				{Kind: "string", ParamName: "child_table", ParamDesc: "child table name"},
				{Kind: "list", ParamName: "child_cols", ParamDesc: "child FK column names"},
				{Kind: "list", ParamName: "parent_vals", ParamDesc: "old parent PK values"},
				{Kind: "string", ParamName: "fk_id", ParamDesc: "FK constraint name"},
				{Kind: "string", ParamName: "mode", ParamDesc: "RESTRICT, CASCADE, or SETNULL"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

	scm.Declare(&en, &scm.Declaration{
		Name: "__fk_on_parent_update",
		Desc: "enforce FK constraint when parent PK is updated",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			currentTx := CurrentTx()
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{
				{Kind: "string", ParamName: "schema", ParamDesc: "database name"},
				{Kind: "string", ParamName: "child_table", ParamDesc: "child table name"},
				{Kind: "list", ParamName: "child_cols", ParamDesc: "child FK column names"},
				{Kind: "list", ParamName: "old_vals", ParamDesc: "old parent PK values"},
				{Kind: "list", ParamName: "new_vals", ParamDesc: "new parent PK values"},
				{Kind: "string", ParamName: "fk_id", ParamDesc: "FK constraint name"},
				{Kind: "string", ParamName: "mode", ParamDesc: "RESTRICT, CASCADE, or SETNULL"},
			},
			Return:    &scm.TypeDescriptor{Kind: "nil"},
			Forbidden: true,
		},
	})

}

// buildFKProc constructs a serializable Proc that calls a builtin with the given args.
// body is the Scheme expression as an S-expression (a Scmer list).
func buildFKProc(body scm.Scmer) scm.Scmer {
	return scm.NewProc(&scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("OLD"), scm.NewSymbol("NEW"), scm.NewSymbol("session")}),
		Body:   body,
		En:     &scm.Globalenv,
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
				scm.NewString(fk.Id),
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
				scm.NewString(fk.Id),
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
			scm.NewString(fk.Id), scm.NewString(modeStr),
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
				scm.NewString(fk.Id), scm.NewString(updateModeStr),
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

// showBuildMeta builds the meta assoc for a table: Name, Engine, Collation, Charset, Comment, Unique, Partitions.
func showBuildMeta(db *database, t *table) scm.Scmer {
	engine := showEngineStr(t)
	uniques := make([]scm.Scmer, len(t.Unique))
	for i, uk := range t.Unique {
		cols := make([]scm.Scmer, len(uk.Cols))
		for j, c := range uk.Cols {
			cols[j] = scm.NewString(c)
		}
		uniques[i] = scm.NewSlice([]scm.Scmer{
			scm.NewString("Id"), scm.NewString(uk.Id),
			scm.NewString("Cols"), scm.NewSlice(cols),
		})
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
		scm.NewString("Unique"), scm.NewSlice(uniques),
		scm.NewString("Partitions"), scm.NewSlice(partitions),
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
	s.mu.RLock()
	mainCount := s.main_count
	delta := len(s.inserts)
	deletions := s.deletions.Count()
	state := sharedStateStr(s.srState)
	size := s.ComputeSize()
	s.mu.RUnlock()
	return scm.NewSlice([]scm.Scmer{
		scm.NewString("shard"), scm.NewInt(int64(i)),
		scm.NewString("state"), scm.NewString(state),
		scm.NewString("main_count"), scm.NewInt(int64(mainCount)),
		scm.NewString("delta"), scm.NewInt(int64(delta)),
		scm.NewString("deletions"), scm.NewInt(int64(deletions)),
		scm.NewString("size_bytes"), scm.NewInt(int64(size)),
	})
}
