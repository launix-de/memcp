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
	"io"
	"os"
	"path/filepath"
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
