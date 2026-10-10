/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "sync"
import "time"
import "testing"
import "runtime"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type controlledSchemaWrite struct {
	data    []byte
	durable bool
	release chan any
}

type controlledSchemaPersistence struct {
	PersistenceEngine
	writes chan controlledSchemaWrite
}

func (p *controlledSchemaPersistence) WriteSchemaWithMode(data []byte, durable bool) {
	write := controlledSchemaWrite{append([]byte(nil), data...), durable, make(chan any)}
	p.writes <- write
	if failure := <-write.release; failure != nil {
		panic(failure)
	}
	p.PersistenceEngine.(schemaWriteOptions).WriteSchemaWithMode(data, durable)
}

func captureTestSchema(db *database) schemaSnapshot {
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	return db.captureSchemaSnapshotLocked()
}

func submitTestSchema(db *database, snapshot schemaSnapshot, durable bool) <-chan any {
	done := make(chan any, 1)
	go func() {
		var failure any
		func() {
			defer func() { failure = recover() }()
			db.commitSchemaSnapshot(snapshot, durable)
		}()
		done <- failure
	}()
	return done
}

func awaitSchemaSubmission(t *testing.T, done <-chan any) any {
	t.Helper()
	select {
	case failure := <-done:
		return failure
	case <-time.After(3 * time.Second):
		t.Fatal("schema submission did not finish")
		return nil
	}
}

func awaitSchemaWrite(t *testing.T, writes <-chan controlledSchemaWrite) controlledSchemaWrite {
	t.Helper()
	select {
	case write := <-writes:
		return write
	case <-time.After(3 * time.Second):
		t.Fatal("schema publication did not reach persistence")
		return controlledSchemaWrite{}
	}
}

func awaitDurableSchemaPending(t *testing.T, db *database) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		db.saveMu.Lock()
		pending := db.savePending.data != nil && db.savePendingSync
		db.saveMu.Unlock()
		if pending {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("superseded fsync request did not upgrade the latest snapshot")
}

func assertSchemaWater(t *testing.T, data []byte, expected uint64) {
	t.Helper()
	var metadata struct {
		HighWater uint64 `json:"sequence_high_water"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil || metadata.HighWater != expected {
		t.Fatalf("snapshot water=%d, expected %d, decode=%v", metadata.HighWater, expected, err)
	}
}

func TestSchemaSnapshotLateCaptureCannotRegressDurableCatalog(t *testing.T) {
	for _, durable := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsynced", true: "synced"}[durable], func(t *testing.T) {
			db := sequenceTestDatabase(t)
			db.loadOnce.Do(func() {})
			old := captureTestSchema(db)
			resumeOld := make(chan struct{})
			done := make(chan any, 1)
			go func() {
				<-resumeOld
				done <- <-submitTestSchema(db, old, durable)
			}()
			db.schemalock.Lock()
			tbl, _ := db.createTableLocked("KeptTable", Memory, false)
			db.schemalock.Unlock()
			db.schemalock.Lock()
			db.metadataValue = scm.NewSlice([]scm.Scmer{scm.NewString("retained_alias"), scm.NewInt(256)})
			tbl.Metadata = scm.NewInt(42)
			db.schemalock.Unlock()
			db.sequenceNext.Store(sequenceReservation - 1)
			db.prepareSequence()
			first := db.allocateSequence(2)
			close(resumeOld)
			if failure := awaitSchemaSubmission(t, done); failure != nil {
				t.Fatal(failure)
			}
			assertSchemaWater(t, db.persistence.ReadSchema(), 2*sequenceReservation)
			restored := newDatabase()
			restored.Name, restored.persistence, restored.srState = db.Name, db.persistence, COLD
			restored.ensureLoaded()
			t.Cleanup(func() { restored.transactionLog.Close() })
			// Cold loading reserves one new interval before publishing an enabled
			// sequence; both the previous fsync and the new lease must survive.
			if restored.SequenceHighWater != 3*sequenceReservation {
				t.Fatal("restored sequence did not preserve and extend the durable interval")
			}
			if kept := restored.GetTable("KeptTable"); kept == nil || kept.Metadata.Int() != tbl.Metadata.Int() {
				t.Fatal("late old snapshot removed a table or rewound its logical identity counter")
			}
			if restored.metadataValue.Slice()[1].Int() != 256 {
				t.Fatal("late old snapshot removed a durable type alias")
			}
			restored.sequenceEnabled.Store(true)
			restored.prepareSequence()
			if next := restored.allocateSequence(1); next <= first+1 {
				t.Fatal("restart reused a value from the acknowledged reservation")
			}
		})
	}
}

func TestSchemaSnapshotSupersededFsyncWaitsForDurableLatest(t *testing.T) {
	for _, completed := range []bool{false, true} {
		t.Run(map[bool]string{false: "inflight", true: "completed"}[completed], func(t *testing.T) {
			db := newDatabase()
			db.Name = "snapshot-test"
			backend := &controlledSchemaPersistence{PersistenceEngine: &FileStorage{path: t.TempDir() + "/"}, writes: make(chan controlledSchemaWrite)}
			db.persistence, db.srState = backend, SHARED
			old := captureTestSchema(db)
			db.schemalock.Lock()
			db.SequenceHighWater = 42
			db.schemalock.Unlock()
			latest := captureTestSchema(db)
			newDone := submitTestSchema(db, latest, false)
			first := awaitSchemaWrite(t, backend.writes)
			if first.durable {
				t.Fatal("fixture write unexpectedly used fsync")
			}
			if completed {
				first.release <- nil
				if failure := awaitSchemaSubmission(t, newDone); failure != nil {
					t.Fatal(failure)
				}
			}
			oldDone := submitTestSchema(db, old, true)
			if !completed {
				awaitDurableSchemaPending(t, db)
				first.release <- nil
			}
			second := awaitSchemaWrite(t, backend.writes)
			if !second.durable {
				t.Fatal("superseded fsync request returned without upgrading persistence")
			}
			assertSchemaWater(t, second.data, 42)
			select {
			case <-oldDone:
				t.Fatal("superseded fsync returned before latest durable I/O completed")
			default:
			}
			second.release <- nil
			if failure := awaitSchemaSubmission(t, oldDone); failure != nil {
				t.Fatal(failure)
			}
			if !completed {
				if failure := awaitSchemaSubmission(t, newDone); failure != nil {
					t.Fatal(failure)
				}
			}
			assertSchemaWater(t, backend.ReadSchema(), 42)
		})
	}
}

func TestSchemaSnapshotConcurrentCapturesRemainOrdered(t *testing.T) {
	db := sequenceTestDatabase(t)
	var submissions sync.WaitGroup
	for i := uint64(1); i <= 64; i++ {
		db.schemalock.Lock()
		db.SequenceHighWater = sequenceReservation + i
		snapshot := db.captureSchemaSnapshotLocked()
		db.schemalock.Unlock()
		submissions.Add(1)
		go func() {
			defer submissions.Done()
			db.commitSchemaSnapshot(snapshot, true)
		}()
	}
	submissions.Wait()
	assertSchemaWater(t, db.persistence.ReadSchema(), sequenceReservation+64)
}

func TestSchemaSnapshotFailedWriteNotifiesWaitersAndRetriesLatest(t *testing.T) {
	db := newDatabase()
	db.Name, db.srState = "snapshot-failure", SHARED
	backend := &controlledSchemaPersistence{PersistenceEngine: &FileStorage{path: t.TempDir() + "/"}, writes: make(chan controlledSchemaWrite)}
	db.persistence = backend
	first := captureTestSchema(db)
	firstDone := submitTestSchema(db, first, false)
	failedWrite := awaitSchemaWrite(t, backend.writes)
	db.schemalock.Lock()
	db.SequenceHighWater = 99
	db.schemalock.Unlock()
	latest := captureTestSchema(db)
	latestDone := submitTestSchema(db, latest, true)
	awaitDurableSchemaPending(t, db)
	failedWrite.release <- "injected schema write failure"
	if failure := awaitSchemaSubmission(t, firstDone); failure == nil {
		t.Fatal("publisher lost its backend failure")
	}
	// A new request may retry immediately. The original coalesced waiter must
	// still observe its failed generation, even when the retry succeeds first.
	retryDone := submitTestSchema(db, captureTestSchema(db), true)
	retryWrite := awaitSchemaWrite(t, backend.writes)
	assertSchemaWater(t, retryWrite.data, 99)
	retryWrite.release <- nil
	if failure := awaitSchemaSubmission(t, retryDone); failure != nil {
		t.Fatal(failure)
	}
	if failure := awaitSchemaSubmission(t, latestDone); failure == nil {
		t.Fatal("immediate successful retry hid a failure from its existing waiter")
	}
	assertSchemaWater(t, backend.ReadSchema(), 99)
}
