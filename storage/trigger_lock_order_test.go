/*
Copyright (C) 2023-2026 Launix GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/launix-de/NonLockingReadMap"
)

func TestPersistedKeytableTriggerWaitsForRuntimeTarget(t *testing.T) {
	original := TriggerDescription{
		Name:     ".kt_cleanup:.grp:query:test|items|AFTER DELETE",
		Timing:   AfterDelete,
		IsSystem: true,
		Acquire:  func(*TxContext) bool { return true },
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var restored TriggerDescription
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.acquireTarget(nil) {
		t.Fatal("persisted keytable trigger ran before its ephemeral target was rebound")
	}

	tbl := &table{Triggers: []TriggerDescription{restored}}
	if !tbl.SetTriggerTarget(original.Name, func(*TxContext) bool { return true }, func() {}) {
		t.Fatal("restored keytable trigger was not reusable")
	}
	if !tbl.Triggers[0].acquireTarget(nil) {
		t.Fatal("rebound keytable trigger did not acquire its runtime target")
	}
}

func TestLegacyPersistedCacheTriggerWaitsForRuntimeTarget(t *testing.T) {
	encoded := []byte(`{"name":".cache:.grp:query:test:agg|scan0|items|AFTER DELETE","timing":5,"is_system":true}`)
	var restored TriggerDescription
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.acquireTarget(nil) {
		t.Fatal("legacy cache trigger ran before its ephemeral target was rebound")
	}
}

func TestDropTriggerDoesNotHoldSchemaLockWhileWaitingForTableDDL(t *testing.T) {
	db := &database{Name: "trigger-lock-order", srState: COLD}
	db.tables = NonLockingReadMap.New[table, string]()
	table := &table{Name: "items", schema: db}
	table.Triggers = []TriggerDescription{{Name: "items_after_drop"}}
	db.tables.Set(table)

	table.ddlMu.Lock()
	dropped := make(chan bool, 1)
	go func() { dropped <- db.dropTrigger("items_after_drop") }()

	// Give dropTrigger time to take its catalog snapshot and wait on ddlMu.
	time.Sleep(20 * time.Millisecond)
	schemaAvailable := make(chan struct{}, 1)
	go func() {
		db.schemalock.Lock()
		db.schemalock.Unlock()
		schemaAvailable <- struct{}{}
	}()

	select {
	case <-schemaAvailable:
	case <-time.After(2 * time.Second):
		table.ddlMu.Unlock()
		t.Fatal("dropTrigger held schemalock while waiting for table ddlMu")
	}

	table.ddlMu.Unlock()
	select {
	case ok := <-dropped:
		if !ok {
			t.Fatal("trigger was not removed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dropTrigger did not finish after ddlMu was released")
	}
}

func TestReadTableLockPublicationDoesNotWaitForShardReaders(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	acquired := make(chan func(), 1)
	go func() {
		acquired <- lockTablePublicationShards([]*storageShard{shard}, false, false)
	}()

	select {
	case unlock := <-acquired:
		unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("READ table lock publication waited for an existing shard reader")
	}
}

func TestSnapshotReadTableLockPublicationDoesNotDeadlockBehindQueuedWriter(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()

	writerAcquired := make(chan struct{})
	releaseWriter := make(chan struct{})
	go func() {
		shard.mu.Lock()
		close(writerAcquired)
		<-releaseWriter
		shard.mu.Unlock()
	}()
	// Give the writer time to queue. Go's RWMutex blocks later readers once a
	// writer is waiting, which reproduces the cache-in-a-reader lock cycle.
	time.Sleep(20 * time.Millisecond)

	readPublished := make(chan func(), 1)
	go func() {
		readPublished <- lockTablePublicationShards([]*storageShard{shard}, false, true)
	}()

	var unlockPublication func()
	blocked := false
	select {
	case unlockPublication = <-readPublished:
	case <-time.After(100 * time.Millisecond):
		blocked = true
	}
	shard.mu.RUnlock()
	<-writerAcquired
	close(releaseWriter)
	if blocked {
		unlockPublication = <-readPublished
	}
	unlockPublication()
	if blocked {
		t.Fatal("READ table lock publication deadlocked behind a queued shard writer")
	}
}

func TestWriteTableLockPublicationWaitsForShardReaders(t *testing.T) {
	shard := &storageShard{}
	shard.mu.RLock()

	acquired := make(chan func(), 1)
	go func() {
		acquired <- lockTablePublicationShards([]*storageShard{shard}, true, false)
	}()

	select {
	case unlock := <-acquired:
		unlock()
		shard.mu.RUnlock()
		t.Fatal("WRITE table lock publication passed an existing shard reader")
	case <-time.After(20 * time.Millisecond):
	}

	shard.mu.RUnlock()
	select {
	case unlock := <-acquired:
		unlock()
	case <-time.After(2 * time.Second):
		t.Fatal("WRITE table lock publication did not continue after reader release")
	}
}

func TestTriggerUnlockTemporarilyWithdrawsWriteOwnership(t *testing.T) {
	shard := &storageShard{}
	tx := NewTxContext(TxCursorStability)
	shard.mu.Lock()
	tx.EnterShardWrite(shard)

	txOwnerVisible := false
	nestedAcquired := false
	shard.runWithWriteLockReleased(tx, func() {
		txOwnerVisible = tx.HasShardWrite(shard)
		acquired := make(chan struct{}, 1)
		go func() {
			shard.mu.Lock()
			shard.mu.Unlock()
			acquired <- struct{}{}
		}()
		select {
		case <-acquired:
			nestedAcquired = true
		case <-time.After(2 * time.Second):
		}
	})

	if txOwnerVisible {
		t.Fatal("trigger callback observed stale shard write ownership")
	}
	if !nestedAcquired {
		t.Fatal("nested trigger query could not acquire the released shard lock")
	}
	if !tx.HasShardWrite(shard) {
		t.Fatal("write ownership was not restored after trigger callback")
	}
	tx.ExitShardWrite(shard)
	shard.mu.Unlock()
}
