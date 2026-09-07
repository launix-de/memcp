/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

type failingPersistenceReader struct {
	readErr  error
	closeErr error
}

func failureHookValue(t *testing.T, event scm.Scmer, key string) scm.Scmer {
	t.Helper()
	values := event.Slice()
	for index := 0; index+1 < len(values); index += 2 {
		if values[index].String() == key {
			return values[index+1]
		}
	}
	t.Fatalf("failure hook event has no %q: %s", key, event.String())
	return scm.NewNil()
}

func waitForPersistenceFailureHook(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for persistenceFailureHooks.running.Load() {
		if time.Now().After(deadline) {
			t.Fatal("storage failure hook did not return")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestPersistenceFailureHookDeliversStructuredEvent(t *testing.T) {
	clearPersistenceFailureHooks()
	t.Cleanup(clearPersistenceFailureHooks)
	received := make(chan scm.Scmer, 1)
	registerPersistenceFailureHook("structured", []string{"*"}, 30*time.Second, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		received <- args[0]
		return scm.NewNil()
	}))

	notifyPersistenceFailureAt(persistenceFailureEvent{
		class: "io", backend: "s3", database: "bucket/prefix",
		operation: "log.write", err: syscall.ENOSPC,
	}, time.Unix(100, 0))

	select {
	case event := <-received:
		if got := failureHookValue(t, event, "class").String(); got != "io" {
			t.Fatalf("class = %q, want io", got)
		}
		if got := failureHookValue(t, event, "backend").String(); got != "s3" {
			t.Fatalf("backend = %q, want s3", got)
		}
		if got := failureHookValue(t, event, "database").String(); got != "bucket/prefix" {
			t.Fatalf("database = %q, want bucket/prefix", got)
		}
		if got := failureHookValue(t, event, "operation").String(); got != "log.write" {
			t.Fatalf("operation = %q, want log.write", got)
		}
		if got := failureHookValue(t, event, "error").String(); got != syscall.ENOSPC.Error() {
			t.Fatalf("error = %q, want %q", got, syscall.ENOSPC.Error())
		}
		if failureHookValue(t, event, "outcome_unknown").Bool() {
			t.Fatal("ordinary I/O failure reported an unknown commit outcome")
		}
		if got := failureHookValue(t, event, "suppressed_count").Int(); got != 0 {
			t.Fatalf("suppressed_count = %d, want 0", got)
		}
	case <-time.After(time.Second):
		t.Fatal("storage failure hook was not called")
	}
	waitForPersistenceFailureHook(t)
}

func TestPersistenceFailureHookCooldownCoalescesFingerprint(t *testing.T) {
	clearPersistenceFailureHooks()
	t.Cleanup(clearPersistenceFailureHooks)
	received := make(chan scm.Scmer, 2)
	registerPersistenceFailureHook("cooldown", []string{"io"}, 10*time.Second, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		received <- args[0]
		return scm.NewNil()
	}))
	event := persistenceFailureEvent{class: "io", backend: "filesystem", database: "db", operation: "log.sync", err: syscall.EIO}
	start := time.Unix(200, 0)
	notifyPersistenceFailureAt(event, start)
	first := <-received
	waitForPersistenceFailureHook(t)
	notifyPersistenceFailureAt(event, start.Add(time.Second))
	notifyPersistenceFailureAt(event, start.Add(2*time.Second))
	notifyPersistenceFailureAt(event, start.Add(11*time.Second))

	second := <-received
	if got := failureHookValue(t, first, "suppressed_count").Int(); got != 0 {
		t.Fatalf("first suppressed_count = %d, want 0", got)
	}
	if got := failureHookValue(t, second, "suppressed_count").Int(); got != 2 {
		t.Fatalf("second suppressed_count = %d, want 2", got)
	}
	waitForPersistenceFailureHook(t)
	select {
	case extra := <-received:
		t.Fatalf("cooldown emitted an extra event: %s", extra.String())
	case <-time.After(20 * time.Millisecond):
	}
}

func TestPersistenceFailureHookDoesNotReenter(t *testing.T) {
	clearPersistenceFailureHooks()
	t.Cleanup(clearPersistenceFailureHooks)
	calls := make(chan struct{}, 2)
	registerPersistenceFailureHook("reentrant", []string{"io"}, 0, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		calls <- struct{}{}
		notifyPersistenceFailureAt(persistenceFailureEvent{
			class: "io", backend: "filesystem", database: "hook",
			operation: "log.write", err: syscall.ENOSPC,
		}, time.Now())
		panic("hook failed")
	}))
	notifyPersistenceFailureAt(persistenceFailureEvent{
		class: "io", backend: "filesystem", database: "db",
		operation: "schema.write", err: syscall.EIO,
	}, time.Now())

	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("storage failure hook was not called")
	}
	waitForPersistenceFailureHook(t)
	select {
	case <-calls:
		t.Fatal("storage failure hook recursively invoked itself")
	case <-time.After(20 * time.Millisecond):
	}

	notifyPersistenceFailureAt(persistenceFailureEvent{
		class: "io", backend: "filesystem", database: "db",
		operation: "schema.sync", err: syscall.EIO,
	}, time.Now())
	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("hook panic disabled later notifications")
	}
	waitForPersistenceFailureHook(t)
}

func TestPersistenceFailureHooksFilterClassesAndDispatchMultipleRules(t *testing.T) {
	clearPersistenceFailureHooks()
	t.Cleanup(clearPersistenceFailureHooks)
	ioCalls := make(chan scm.Scmer, 1)
	cleanupCalls := make(chan scm.Scmer, 1)
	registerPersistenceFailureHook("io-only", []string{"io"}, 0, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		ioCalls <- args[0]
		return scm.NewNil()
	}))
	registerPersistenceFailureHook("cleanup-only", []string{"cleanup"}, 0, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		cleanupCalls <- args[0]
		return scm.NewNil()
	}))

	notifyPersistenceFailure(persistenceFailureEvent{
		class: "ambiguous_commit", backend: "filesystem", database: "db",
		operation: "log.sync", err: syscall.EIO, outcomeUnknown: true,
	})
	select {
	case <-ioCalls:
		t.Fatal("io-only hook received an ambiguous commit event")
	case <-cleanupCalls:
		t.Fatal("cleanup-only hook received an ambiguous commit event")
	case <-time.After(20 * time.Millisecond):
	}

	notifyPersistenceFailure(persistenceFailureEvent{
		class: "io", backend: "filesystem", database: "db",
		operation: "log.write", err: syscall.ENOSPC,
	})
	select {
	case event := <-ioCalls:
		if got := failureHookValue(t, event, "class").String(); got != "io" {
			t.Fatalf("io hook class = %q, want io", got)
		}
	case <-time.After(time.Second):
		t.Fatal("io-only hook did not receive matching event")
	}
	waitForPersistenceFailureHook(t)

	notifyPersistenceFailure(persistenceFailureEvent{
		class: "cleanup", backend: "s3", database: "db",
		operation: "log.swap.cleanup", err: syscall.EIO,
	})
	select {
	case event := <-cleanupCalls:
		if got := failureHookValue(t, event, "class").String(); got != "cleanup" {
			t.Fatalf("cleanup hook class = %q, want cleanup", got)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup-only hook did not receive matching event")
	}
	waitForPersistenceFailureHook(t)
}

func TestPersistenceFailureHookNameReplacesAndUnregistersRule(t *testing.T) {
	clearPersistenceFailureHooks()
	t.Cleanup(clearPersistenceFailureHooks)
	oldCalls := make(chan struct{}, 1)
	newCalls := make(chan struct{}, 1)
	registerPersistenceFailureHook("replaceable", []string{"io"}, 0, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		oldCalls <- struct{}{}
		return scm.NewNil()
	}))
	registerPersistenceFailureHook("replaceable", []string{"cleanup"}, 0, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		newCalls <- struct{}{}
		return scm.NewNil()
	}))
	notifyPersistenceFailure(persistenceFailureEvent{class: "io", backend: "filesystem", database: "db", operation: "log.write"})
	select {
	case <-oldCalls:
		t.Fatal("replaced hook callback was called")
	case <-newCalls:
		t.Fatal("replacement hook ignored its class mask")
	case <-time.After(20 * time.Millisecond):
	}
	notifyPersistenceFailure(persistenceFailureEvent{class: "cleanup", backend: "filesystem", database: "db", operation: "log.swap.cleanup"})
	select {
	case <-newCalls:
	case <-time.After(time.Second):
		t.Fatal("replacement hook was not called")
	}
	waitForPersistenceFailureHook(t)
	unregisterPersistenceFailureHook("replaceable")
	notifyPersistenceFailure(persistenceFailureEvent{class: "cleanup", backend: "filesystem", database: "db", operation: "log.swap.cleanup"})
	select {
	case <-newCalls:
		t.Fatal("unregistered hook was called")
	case <-time.After(20 * time.Millisecond):
	}
}

func (r failingPersistenceReader) Read([]byte) (int, error) { return 0, r.readErr }
func (r failingPersistenceReader) Close() error             { return r.closeErr }

func TestPersistenceObjectReaderStandardizesStreamErrors(t *testing.T) {
	for _, reader := range []io.ReadCloser{
		standardPersistenceReader(failingPersistenceReader{readErr: syscall.EIO}, "test", "db", "column.read"),
		standardPersistenceReader(failingPersistenceReader{readErr: io.EOF, closeErr: syscall.EIO}, "test", "db", "column.read"),
	} {
		func() {
			defer func() {
				if _, ok := recover().(*PersistenceFailure); !ok {
					t.Fatal("stream error did not panic with *PersistenceFailure")
				}
			}()
			_, err := reader.Read(make([]byte, 1))
			if err == io.EOF {
				_ = reader.Close()
			}
		}()
	}
}

func TestRemoteRetryIsBounded(t *testing.T) {
	want := errors.New("remote unavailable")
	attempts := 0
	err := remoteRetry(func() error {
		attempts++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("remoteRetry returned %v, want %v", err, want)
	}
	if attempts != remoteStorageAttempts {
		t.Fatalf("remoteRetry made %d attempts, want %d", attempts, remoteStorageAttempts)
	}
}

func TestRemoteRetryStopsAfterSuccess(t *testing.T) {
	attempts := 0
	err := remoteRetry(func() error {
		attempts++
		if attempts < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("remoteRetry made %d attempts, want 2", attempts)
	}
}

func TestRemoteRetryValueReturnsSuccessfulAttempt(t *testing.T) {
	attempts := 0
	value, err := remoteRetryValue(func() (int, error) {
		attempts++
		if attempts < 2 {
			return 0, errors.New("transient")
		}
		return 42, nil
	})
	if err != nil || value != 42 || attempts != 2 {
		t.Fatalf("remoteRetryValue = (%d, %v) after %d attempts", value, err, attempts)
	}
}

func TestFaultInjectorRejectsPostPublicationFailure(t *testing.T) {
	t.Setenv("MEMCP_IO_FAULT_PROBABILITY", "1")
	t.Setenv("MEMCP_IO_FAULT_PHASE", "after")
	defer func() {
		if recover() == nil {
			t.Fatal("after-publication fault phase was accepted")
		}
	}()
	_ = persistenceFaultInjectorFromEnv("test")
}

func TestPartialFilesystemWALWriteLeavesNoTornFrame(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fault-db") + "/"
	raw := &FileStorage{path: dir}
	t.Setenv("MEMCP_IO_FAULT_PROBABILITY", "1")
	t.Setenv("MEMCP_IO_FAULT_OPERATIONS", "log.write")
	t.Setenv("MEMCP_IO_FAULT_PHASE", "partial")
	t.Setenv("MEMCP_IO_FAULT_AFTER", "1")
	t.Setenv("MEMCP_IO_FAULT_LIMIT", "1")
	logfile := instrumentPersistence("fault-db", raw).OpenLog("shard")
	logfile.Write(LogEntryDelete{idx: 1})

	func() {
		defer func() {
			recovered := recover()
			if _, ok := recovered.(*PersistenceFailure); !ok {
				t.Fatalf("partial write panic = %T, want *PersistenceFailure", recovered)
			}
		}()
		logfile.Write(LogEntryDelete{idx: 2})
	}()
	logfile.Write(LogEntryDelete{idx: 3})
	logfile.Close()

	_, entries, replayLog := raw.ReplayLog("shard")
	var recids []uint32
	for entry := range entries {
		if deletion, ok := entry.(LogEntryDelete); ok {
			recids = append(recids, deletion.idx)
		}
	}
	replayLog.Close()
	if len(recids) != 2 || recids[0] != 1 || recids[1] != 3 {
		t.Fatalf("replayed delete recids = %v, want [1 3]", recids)
	}
}

func TestFilesystemLogErrorsUsePersistenceFailure(t *testing.T) {
	raw := &FileStorage{path: filepath.Join(t.TempDir(), "closed-log") + "/"}
	logfile := raw.OpenLog("shard")
	logfile.Close()
	defer func() {
		if _, ok := recover().(*PersistenceFailure); !ok {
			t.Fatal("closed WAL write did not panic with *PersistenceFailure")
		}
	}()
	logfile.Write(LogEntryDelete{idx: 1})
}

func TestFilesystemWalkErrorsUsePersistenceFailure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "blob"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	raw := &FileStorage{path: dir + "/"}
	for _, walk := range []func(){
		func() { raw.WalkBlobs(func(string) {}) },
		func() { (&FileStorage{path: filepath.Join(dir, "missing") + "/"}).WalkShardFiles(func(string) {}) },
	} {
		func() {
			defer func() {
				if _, ok := recover().(*PersistenceFailure); !ok {
					t.Fatal("filesystem walk did not panic with *PersistenceFailure")
				}
			}()
			walk()
		}()
	}
}
