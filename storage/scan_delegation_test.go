/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "fmt"
import "testing"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

type testShardScanExecutor func(mosiResource, shardScanInvocation) scanResult

func (execute testShardScanExecutor) ExecuteShardScan(resource mosiResource, invocation shardScanInvocation) scanResult {
	return execute(resource, invocation)
}

func setTestScanTarget(shard *storageShard, execute testShardScanExecutor) {
	shard.scanTarget.Store(&shardScanTarget{
		resource: mosiResource{Database: shard.t.schema.Name, Table: shard.t.Name, Shard: shard.uuid.String()},
		executor: execute,
	})
}

func delegationTestScan(tbl *table, tx *TxContext, callback scm.Scmer) scm.Scmer {
	condition := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) })
	return tbl.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, condition, []string{"id"}, callback, scm.NewInt(0), combine, false)
}

func TestScanDelegationColdSingleAndMixedShards(t *testing.T) {
	for _, multi := range []bool{false, true} {
		t.Run(fmt.Sprint(multi), func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "tscandelegation", 4)
			RebuildTable(tbl, true, false)
			if multi {
				if !tbl.beginManualRepartition() {
					t.Fatal("manual repartition was not claimed")
				}
				tbl.repartition([]shardDimension{tbl.NewShardDimension("id", 2)})
			}
			shards := tbl.ActiveShards()
			if multi && len(shards) < 2 {
				t.Fatal("fixture did not partition")
			}
			remote := shards[0]
			// Determine the partial result before detaching its payload. The fake
			// executor returns only that reduction, never loads the remote stub.
			var sum int64
			var count int64
			lease := remote.GetRead(nil, false)
			reader := remote.ColumnReaderTx(nil, "id", false, ReadAuto)
			for id := uint32(0); id < remote.main_count; id++ {
				sum += reader.GetValue(id).Int()
				count++
			}
			ReleaseColumnReader(reader)
			lease.Release()
			var freed [numEvictableTypes]int64
			if !shardCleanup(remote, &freed) || remote.state() != COLD {
				t.Fatal("fixture did not become cold")
			}
			var calls atomic.Int32
			tx := NewTxContext(TxACID)
			setTestScanTarget(remote, func(resource mosiResource, invocation shardScanInvocation) scanResult {
				calls.Add(1)
				if resource.Shard != remote.uuid.String() || invocation.tx != tx {
					t.Error("delegation lost resource, generation or invocation binding")
				}
				return scanResult{res: scm.NewInt(sum), outCount: count, inputCount: count, candidateCount: count}
			})
			callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) })
			if got := delegationTestScan(tbl, tx, callback).Int(); got != 10 {
				t.Fatalf("combined sum = %d, want 10", got)
			}
			if calls.Load() != 1 || remote.state() != COLD || remote.cacheReaders.Load() != 0 {
				t.Fatalf("delegation calls=%d state=%v readers=%d", calls.Load(), remote.state(), remote.cacheReaders.Load())
			}
			for _, shard := range shards {
				if shard.activeScanners.Load() != 0 || shard.cacheReaders.Load() != 0 {
					t.Fatal("scan leaked a shard registration or payload lease")
				}
			}
			if tbl.activeTopology().operations.Load() != 0 {
				t.Fatal("scan leaked topology registration")
			}
		})
	}
}

func TestScanDelegationDoesNotRetryExecutorFailure(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscandelegationfailure")
	shard := tbl.ActiveShards()[0]
	calls := 0
	setTestScanTarget(shard, func(mosiResource, shardScanInvocation) scanResult {
		calls++
		panic("ambiguous remote completion")
	})
	var failure any
	func() {
		defer func() { failure = recover() }()
		delegationTestScan(tbl, nil, scm.NewNil())
	}()
	if fmt.Sprint(failure) != "ambiguous remote completion" || calls != 1 {
		t.Fatalf("failure=%v calls=%d", failure, calls)
	}
	if shard.activeScanners.Load() != 0 || tbl.activeTopology().operations.Load() != 0 {
		t.Fatal("failed delegation leaked registrations")
	}
}

func TestScanDelegationOnlyConsumesAcquisitionPanic(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "tscandelegationcallback", 1)
	calls := 0
	target := &shardScanTarget{executor: testShardScanExecutor(func(mosiResource, shardScanInvocation) scanResult {
		calls++
		return scanResult{}
	})}
	callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic(shardReadDelegation{target}) })
	var failure any
	func() {
		defer func() { failure = recover() }()
		delegationTestScan(tbl, nil, callback)
	}()
	if failure == nil || calls != 0 {
		t.Fatalf("callback panic caused delegation: failure=%v calls=%d", failure, calls)
	}
}

func TestScanDelegationPreservesNativeMutationInvocation(t *testing.T) {
	for _, column := range []string{"$update", "$increment:id"} {
		t.Run(column, func(t *testing.T) {
			tbl := setupScanParallelTestTable(t, "tscandelegationmutation")
			shard := tbl.ActiveShards()[0]
			tx := NewTxContext(TxACID)
			defer tx.Rollback()
			calls := 0
			setTestScanTarget(shard, func(_ mosiResource, invocation shardScanInvocation) scanResult {
				calls++
				if invocation.tx != tx || len(invocation.callbackCols) != 1 || invocation.callbackCols[0] != column {
					t.Fatal("delegation lost the native transaction or mutation callback")
				}
				return scanResult{res: scm.NewInt(4), outCount: 4, inputCount: 4, candidateCount: 4}
			})
			combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewInt(args[0].Int() + args[1].Int()) })
			got := tbl.scanWithBatch(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, scm.NewNil(), []string{column}, scm.NewNil(), scm.NewInt(0), combine, false, 0, nil)
			if got.Int() != 4 || calls != 1 {
				t.Fatalf("result=%v delegation count=%d", got, calls)
			}
		})
	}
}

func TestScanDelegationRejectsStaleTasks(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscandelegationunsupported")
	shard := tbl.ActiveShards()[0]
	calls := 0
	setTestScanTarget(shard, func(mosiResource, shardScanInvocation) scanResult {
		calls++
		return scanResult{}
	})
	stale := *shard.scanTarget.Load()
	stale.resource.Shard = uuid.NewString()
	shard.scanTarget.Store(&stale)
	var staleFailure any
	func() {
		defer func() { staleFailure = recover() }()
		delegationTestScan(tbl, nil, scm.NewNil())
	}()
	if fmt.Sprint(staleFailure) != "invalid remote shard scan target" {
		t.Fatalf("stale target failure = %v", staleFailure)
	}
	if calls != 0 {
		t.Fatal("rejected task reached the executor")
	}
	var failure any
	func() {
		defer func() { failure = recover() }()
		shard.GetWrite(nil)
	}()
	if failure == nil || shard.cacheReaders.Load() != 0 {
		t.Fatal("stub admitted a local write")
	}
}

func TestScanDelegationPreservesNativeExistsAndBatchInvocation(t *testing.T) {
	for _, exists := range []bool{true, false} {
		t.Run(fmt.Sprint("exists=", exists), func(t *testing.T) {
			table, _ := createDurabilityTestTable(t, "t_scan_family_delegate", 4)
			RebuildTable(table, true, false)
			shard := table.ActiveShards()[0]
			var freed [numEvictableTypes]int64
			if !shardCleanup(shard, &freed) {
				t.Fatal("fixture remained resident")
			}
			calls := 0
			batch := []scm.Scmer{scm.NewInt(2), scm.NewInt(10), scm.NewInt(4), scm.NewInt(20)}
			setTestScanTarget(shard, func(_ mosiResource, invocation shardScanInvocation) scanResult {
				calls++
				if exists {
					if invocation.operator != shardExists {
						t.Fatal("EXISTS was lowered to a full reduction")
					}
					return scanResult{res: scm.NewBool(true), outCount: 1}
				}
				if invocation.operator != shardReduce || invocation.stride != 2 || !scm.Equal(scm.NewSlice(invocation.batchdata), scm.NewSlice(batch)) {
					t.Fatal("delegation discarded its batch bindings", invocation)
				}
				return scanResult{res: scm.NewInt(36), outCount: 2}
			})
			condition := scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
			if exists {
				if !table.scanExists(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, condition) {
					t.Fatal("remote EXISTS result was lost")
				}
			} else {
				value := table.scanWithBatch(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, condition, nil, scm.Globalenv.Vars[scm.Symbol("+")], scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false, 2, batch)
				if value.Int() != 36 {
					t.Fatal("remote batch result was lost", value)
				}
			}
			if calls != 1 || shard.state() != COLD || shard.cacheReaders.Load() != 0 || shard.activeScanners.Load() != 0 || table.activeTopology().operations.Load() != 0 {
				t.Fatal("delegation loaded its source or leaked native ownership", calls, shard.state())
			}
		})
	}
}
