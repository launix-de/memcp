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
package main

import "archive/tar"
import "archive/zip"
import "bytes"
import "compress/gzip"
import "io"
import "os"
import "path/filepath"
import "testing"

func TestOpenStreamDecompressesFinalGzipFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gzipPath := filepath.Join(dir, "hello.txt.gz")
	writeGzipFile(t, gzipPath, []byte("hello gzip"))

	stream, err := openStream(gzipPath)
	if err != nil {
		t.Fatalf("openStream(%q): %v", gzipPath, err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll(%q): %v", gzipPath, err)
	}
	if string(data) != "hello gzip" {
		t.Fatalf("unexpected gzip payload: got %q", string(data))
	}
}

func TestOpenStreamReadsTarMemberInsideGzipPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "dump.gz")
	tarPayload := buildTarArchive(t, map[string]string{
		"restore.sql": "SELECT 1;\n",
		"3365.dat":    "1\tAlice\n",
	})
	writeGzipFile(t, archivePath, tarPayload)

	stream, err := openStream(filepath.Join(archivePath, "restore.sql"))
	if err != nil {
		t.Fatalf("openStream(restore.sql): %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll(restore.sql): %v", err)
	}
	if string(data) != "SELECT 1;\n" {
		t.Fatalf("unexpected restore.sql content: got %q", string(data))
	}
}

func TestOpenStreamReadsNestedArchiveMembers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outerZipPath := filepath.Join(dir, "outer.zip")
	innerTar := buildTarArchive(t, map[string]string{
		"nested/deep.txt": "nested archive payload",
	})
	writeZipFile(t, outerZipPath, map[string][]byte{
		"inner.tar": innerTar,
	})

	stream, err := openStream(filepath.Join(outerZipPath, "inner.tar", "nested", "deep.txt"))
	if err != nil {
		t.Fatalf("openStream(nested archive member): %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll(nested archive member): %v", err)
	}
	if string(data) != "nested archive payload" {
		t.Fatalf("unexpected nested archive content: got %q", string(data))
	}
}

func writeGzipFile(t *testing.T, path string, data []byte) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q): %v", path, err)
	}
	defer file.Close()

	writer := gzip.NewWriter(file)
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("gzip.Write(%q): %v", path, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip.Close(%q): %v", path, err)
	}
}

func buildTarArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatalf("tar.WriteHeader(%q): %v", name, err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatalf("tar.Write(%q): %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("tar.Close: %v", err)
	}
	return buf.Bytes()
}

func writeZipFile(t *testing.T, path string, files map[string][]byte) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q): %v", path, err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q): %v", name, err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatalf("zip.Write(%q): %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zip.Close(%q): %v", path, err)
	}
}
