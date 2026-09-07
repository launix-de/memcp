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
)

type failingPersistenceReader struct {
	readErr  error
	closeErr error
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
