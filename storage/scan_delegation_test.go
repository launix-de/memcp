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
			lease := remote.GetRead(nil)
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

func TestScanDelegationRejectsUnsupportedAndStaleTasks(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tscandelegationunsupported")
	shard := tbl.ActiveShards()[0]
	calls := 0
	setTestScanTarget(shard, func(mosiResource, shardScanInvocation) scanResult {
		calls++
		return scanResult{}
	})
	for _, test := range []struct {
		stride  int
		columns []string
		want    string
	}{
		{stride: 1, want: "remote shard scan does not support physical source sets or batches"},
		{columns: []string{"$update"}, want: "remote shard scan mutations require a transaction participant protocol"},
		{columns: []string{"$increment:id"}, want: "remote shard scan mutations require a transaction participant protocol"},
	} {
		var failure any
		func() {
			defer func() { failure = recover() }()
			tbl.scanWithBatch(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil, scm.NewNil(), test.columns, scm.NewNil(), scm.NewInt(0), scm.NewNil(), false, test.stride, nil)
		}()
		if fmt.Sprint(failure) != test.want {
			t.Fatalf("unsupported scan failure = %v, want %s", failure, test.want)
		}
	}
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
