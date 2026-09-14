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
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFileBlobWriteDoesNotExposePartialReplacement(t *testing.T) {
	dir := t.TempDir()
	store := &FileStorage{path: dir + "/"}
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	first := store.WriteBlob(hash)
	if _, err := io.WriteString(first, "published"); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	interrupted := store.WriteBlob(hash)
	if _, err := io.WriteString(interrupted, "partial"); err != nil {
		t.Fatal(err)
	}
	// Before Close publishes the new generation, readers must still see the
	// complete content-addressed object which was already present.
	reader := store.ReadBlob(hash)
	got, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "published" {
		t.Fatalf("partial blob write became visible: got %q", got)
	}
	if err := interrupted.Close(); err != nil {
		t.Fatal(err)
	}

	got, err = os.ReadFile(filepath.Clean(store.blobPath(hash)))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "published" {
		t.Fatalf("content-addressed blob was overwritten: got %q", got)
	}
}

func TestConcurrentFileBlobWritersPublishOneCompleteObject(t *testing.T) {
	dir := t.TempDir()
	store := &FileStorage{path: dir + "/"}
	hash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	payloads := []string{"complete-left", "complete-right"}
	var ready sync.WaitGroup
	ready.Add(len(payloads))
	release := make(chan struct{})
	var writers sync.WaitGroup
	for _, payload := range payloads {
		writers.Add(1)
		go func(payload string) {
			defer writers.Done()
			writer := store.WriteBlob(hash)
			if _, err := io.WriteString(writer, payload); err != nil {
				t.Errorf("write blob: %v", err)
				return
			}
			ready.Done()
			<-release
			if err := writer.Close(); err != nil {
				t.Errorf("close blob: %v", err)
			}
		}(payload)
	}
	ready.Wait()
	close(release)
	writers.Wait()
	got, err := os.ReadFile(store.blobPath(hash))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payloads[0] && string(got) != payloads[1] {
		t.Fatalf("published partial/interleaved blob %q", got)
	}
}

func blobWriteFault(engine PersistenceEngine, operation, phase string) *faultPersistence {
	return &faultPersistence{PersistenceEngine: engine, database: "blob-abort-test", faults: &persistenceFaultInjector{
		rng: rand.New(rand.NewSource(1)), probability: 1, remaining: 1,
		operations: map[string]struct{}{operation: {}}, phase: phase,
	}}
}

func assertNoBlobTempfiles(t *testing.T, dir string) {
	t.Helper()
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(entry.Name(), ".blob-write-") {
			t.Errorf("abandoned tempfile: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestBlobWritePanicAbortsPrivateFile(t *testing.T) {
	for _, phase := range []string{"before", "partial", "close"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", phase, existing), func(t *testing.T) {
				dir := t.TempDir()
				store := &FileStorage{path: dir + "/"}
				hash := strings.Repeat("ab", 32)
				if existing {
					writeBlobContents(store, hash, "committed")
				}
				op, faultPhase := "blob.write", phase
				if phase == "close" {
					op, faultPhase = "blob.write.close", "before"
				}
				var caught any
				func() {
					defer func() { caught = recover() }()
					writeBlobContents(blobWriteFault(store, op, faultPhase), hash, "uncommitted")
				}()
				if caught == nil {
					t.Fatal("expected injected write panic")
				}
				assertNoBlobTempfiles(t, dir)
				got, err := os.ReadFile(store.blobPath(hash))
				if existing {
					if err != nil || string(got) != "committed" {
						t.Fatalf("committed object changed: %q, %v", got, err)
					}
				} else if !os.IsNotExist(err) {
					t.Fatalf("partial object published: %q, %v", got, err)
				}
			})
		}
	}
}

func TestFileBlobAbortClosesDescriptor(t *testing.T) {
	store := &FileStorage{path: t.TempDir() + "/"}
	writer := store.WriteBlob(strings.Repeat("cd", 32)).(*fileBlobWriter)
	if _, err := writer.Write([]byte("private")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Abort(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.File.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("descriptor still open: %v", err)
	}
	if err := writer.Abort(); err != nil {
		t.Fatalf("second abort: %v", err)
	}
	assertNoBlobTempfiles(t, store.path)
}

func TestFileBlobCloseFailureAbortsPrivateFile(t *testing.T) {
	store := &FileStorage{path: t.TempDir() + "/"}
	writer := store.WriteBlob(strings.Repeat("ef", 32)).(*fileBlobWriter)
	writer.File.Close() // inject a sync failure without touching committed data
	var caught any
	func() { defer func() { caught = recover() }(); writer.Close() }()
	if caught == nil {
		t.Fatal("expected sync panic")
	}
	assertNoBlobTempfiles(t, store.path)
	if _, err := os.Stat(writer.finalPath); !os.IsNotExist(err) {
		t.Fatalf("failed write published: %v", err)
	}
}

func TestBlobStartupRemovesOnlyAbandonedWrites(t *testing.T) {
	defer setupGCTest(t)()
	CreateDatabase("gcdb", false)
	db := GetDatabase("gcdb")
	store := db.persistence.(*FileStorage)
	hash := strings.Repeat("ab", 32)
	writeBlobContents(store, hash, "committed")
	writer := store.WriteBlob(hash).(*fileBlobWriter)
	writer.Write([]byte("interrupted"))
	writer.File.Close() // model process death: file survives, no live descriptor
	other := filepath.Join(writer.directory, "unrelated-file")
	if err := os.WriteFile(other, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	databases.Remove("gcdb")
	LoadDatabases()
	assertNoBlobTempfiles(t, store.path)
	got, err := os.ReadFile(store.blobPath(hash))
	if err != nil || string(got) != "committed" {
		t.Fatalf("committed blob changed: %q, %v", got, err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("unrelated file removed: %v", err)
	}
	// Runtime GC must not sweep a live writer's tempfile.
	active := store.WriteBlob(strings.Repeat("cd", 32)).(*fileBlobWriter)
	defer active.Abort()
	CleanDatabase(GetDatabase("gcdb"))
	if _, err := os.Stat(active.File.Name()); err != nil {
		t.Fatalf("active tempfile removed: %v", err)
	}
}

// The reader supplies bytes before failing, as an interrupted backend transfer
// can. A destination must not publish that prefix under a complete object's hash.
type interruptedBlobReader struct{ sent bool }

func (r *interruptedBlobReader) Read(buffer []byte) (int, error) {
	if r.sent {
		return 0, io.ErrUnexpectedEOF
	}
	r.sent = true
	return copy(buffer, "partial"), nil
}
func (*interruptedBlobReader) Close() error { return nil }

func TestBlobMoveReadFailureDoesNotPublishPrefix(t *testing.T) {
	store := &FileStorage{path: t.TempDir() + "/"}
	hash := strings.Repeat("ab", 32)
	writer := store.WriteBlob(hash).(*fileBlobWriter)
	var caught any
	func() {
		defer func() { caught = recover() }()
		copyPersistenceObject(&interruptedBlobReader{}, writer, store, "database.move.blob")
	}()
	if caught == nil {
		t.Fatal("expected interrupted transfer failure")
	}
	if _, err := os.Stat(store.blobPath(hash)); !os.IsNotExist(err) {
		t.Fatalf("partial blob published: %v", err)
	}
	if _, err := writer.File.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("descriptor still open: %v", err)
	}
	assertNoBlobTempfiles(t, store.path)
}

func TestBlobRemoteAbortDoesNotPublishBufferedContents(t *testing.T) {
	writer := &s3WriteCloser{} // no client: any attempted publication would panic
	writer.Write([]byte("private"))
	if err := writer.Abort(); err != nil {
		t.Fatal(err)
	}
	if writer.buf.Len() != 0 {
		t.Fatal("abort retained payload buffer")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("late")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("aborted writer accepted data: %v", err)
	}
}
