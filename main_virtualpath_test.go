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
import "fmt"
import "strings"
import "encoding/hex"
import "os"
import "path/filepath"
import "testing"
import "github.com/launix-de/memcp/scm"

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

func TestWatchInitialErrorsPropagate(t *testing.T) {
	for _, missing := range []bool{true, false} {
		t.Run(map[bool]string{true: "missing-file", false: "callback-panic"}[missing], func(t *testing.T) {
			dir := t.TempDir()
			if !missing {
				if err := os.WriteFile(filepath.Join(dir, "watched"), []byte("initial"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			defer func() {
				if recover() == nil {
					t.Error("initial watch error was swallowed")
				}
			}()
			getWatch(dir)(scm.NewString("watched"), scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic("initial callback failed") }))
		})
	}
}

// loadTestStream records ownership and deliberately returns one byte per read.
// A decoder cannot assume that a BOM or surrogate pair arrives in one Read.
type loadTestStream struct {
	io.Reader
	closed bool
}

func (s *loadTestStream) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return s.Reader.Read(p)
}

func (s *loadTestStream) Close() error { s.closed = true; return nil }

func TestLoadTextEncoding(t *testing.T) {
	for _, tc := range []struct {
		name, hex, encoding, want, failure string
	}{
		{"utf8", "41c3bcf09f9098", "bom", "Aü🐘", ""},
		{"utf8-bom-retained", "efbbbf41", "bom", "\ufeffA", ""},
		{"no-endian-guess", "41004200", "bom", "A\x00B\x00", ""},
		{"utf16le", "fffe4100fc003dd818dc", "bom", "Aü🐘", ""},
		{"utf16be", "feff004100fcd83ddc18", "bom", "Aü🐘", ""},
		{"explicit-le", "4100fc003dd818dc", "utf-16le", "Aü🐘", ""},
		{"explicit-be", "004100fcd83ddc18", "utf-16be", "Aü🐘", ""},
		{"explicit-le-bom", "fffe4100", "utf-16le", "A", ""},
		{"explicit-be-bom", "feff0041", "utf-16be", "A", ""},
		{"embedded-bom", "fffe4100fffe4200", "bom", "A\ufeffB", ""},
		{"replacement-is-data", "fffefdff", "bom", "�", ""},
		{"empty", "", "bom", "", ""},
		{"bom-only", "fffe", "bom", "", ""},
		{"raw-default", "fffe4100", "", "\xff\xfeA\x00", ""},
		{"odd-byte", "fffe410042", "bom", "", "odd UTF-16"},
		{"high-surrogate", "fffe00d8", "bom", "", "surrogate"},
		{"low-surrogate", "fffe00dc", "bom", "", "surrogate"},
		{"nonpair", "fffe00d84100", "bom", "", "surrogate"},
		{"reversed-pair", "fffe00dc00d8", "bom", "", "surrogate"},
		{"utf32le", "fffe000041000000", "bom", "", "UTF-32"},
		{"utf32be", "0000feff00000041", "bom", "", "UTF-32"},
		{"wrong-le-bom", "feff0041", "utf-16le", "", "byte order"},
		{"wrong-be-bom", "fffe4100", "utf-16be", "", "byte order"},
		{"unknown-encoding", "41", "latin-guess", "", "encoding"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := hex.DecodeString(tc.hex)
			if err != nil {
				t.Fatal(err)
			}
			stream := &loadTestStream{Reader: bytes.NewReader(data)}
			defer func() {
				failure := recover()
				if tc.failure == "" && failure != nil {
					t.Fatalf("unexpected failure: %v", failure)
				}
				if tc.failure != "" && (failure == nil || !strings.Contains(fmt.Sprint(failure), tc.failure)) {
					t.Errorf("failure = %v, want %q", failure, tc.failure)
				}
				if stream.closed {
					t.Error("load closed a borrowed reader")
				}
			}()
			got := getLoad("")(scm.NewAny(stream), scm.NewNil(), scm.NewNil(), scm.NewString(tc.encoding))
			if tc.failure == "" && got.String() != tc.want {
				t.Errorf("got %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestLoadDecodedHandlers(t *testing.T) {
	for _, delimiter := range []scm.Scmer{scm.NewNil(), scm.NewString("\n")} {
		t.Run(fmt.Sprint(delimiter), func(t *testing.T) {
			data := []byte{0xff, 0xfe, 'A', 0, '\n', 0, 'B', 0}
			var lines []string
			callback := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { lines = append(lines, a[0].String()); return scm.NewBool(true) })
			getLoad("")(scm.NewAny(bytes.NewReader(data)), callback, delimiter, scm.NewString("bom"))
			want := []string{"A\nB"}
			if !delimiter.IsNil() {
				want = []string{"A\n", "B"}
			}
			if fmt.Sprint(lines) != fmt.Sprint(want) {
				t.Errorf("lines = %q, want %q", lines, want)
			}
		})
	}
	for _, data := range [][]byte{{0xff, 0xfe, 'A', 0, '\n', 0, 0}, {0xff, 0xfe, 0, 0xd8}} {
		t.Run(fmt.Sprintf("malformed-%x", data), func(t *testing.T) {
			calls := 0
			defer func() {
				if recover() == nil {
					t.Error("malformed input succeeded")
				}
				if calls != 0 {
					t.Errorf("handler ran %d times before decode failure", calls)
				}
			}()
			getLoad("")(scm.NewAny(bytes.NewReader(data)), scm.NewFunc(func(...scm.Scmer) scm.Scmer { calls++; return scm.NewBool(true) }), scm.NewString("\n"), scm.NewString("bom"))
		})
	}
}

func TestLoadFileOwnership(t *testing.T) {
	// Inspect only this test's filename; other parallel tests may open descriptors.
	if _, err := os.ReadDir("/proc/self/fd"); err != nil {
		t.Skip("descriptor inspection unavailable")
	}
	dir := t.TempDir()
	filename := filepath.Join(dir, "owned.txt.gz")
	writeGzipFile(t, filename, []byte{0xff, 0xfe, 'A', 0})
	for _, panics := range []bool{false, true} {
		func() {
			defer func() {
				if (recover() != nil) != panics {
					t.Error("unexpected callback panic state")
				}
			}()
			callback := scm.NewFunc(func(...scm.Scmer) scm.Scmer {
				if panics {
					panic("callback failed")
				}
				return scm.NewBool(true)
			})
			getLoad(dir)(scm.NewString("owned.txt.gz"), callback, scm.NewNil(), scm.NewString("bom"))
		}()
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name, _ := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if name == filename {
				t.Errorf("load leaked owned file descriptor %s", entry.Name())
			}
		}
	}
}

func TestLoadGzipDecodeFailure(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "broken.sql.gz")
	writeGzipFile(t, filename, []byte{0xff, 0xfe, 'A', 0})
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-8] ^= 1 // corrupt the checksum, retaining a readable payload
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	defer func() {
		if recover() == nil {
			t.Error("corrupt gzip succeeded")
		}
		if calls != 0 {
			t.Error("handler saw partial input before gzip failure")
		}
	}()
	getLoad(dir)(scm.NewString("broken.sql.gz"), scm.NewFunc(func(...scm.Scmer) scm.Scmer { calls++; return scm.NewBool(true) }), scm.NewNil(), scm.NewString("bom"))
}

func TestLoadLegacyCalls(t *testing.T) {
	for _, count := range []int{1, 2, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			stream := &loadTestStream{Reader: strings.NewReader("\xff\xfeA\x00\nB")}
			var got []string
			args := []scm.Scmer{scm.NewAny(stream)}
			if count > 1 {
				args = append(args, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { got = append(got, a[0].String()); return scm.NewBool(true) }))
			}
			if count > 2 {
				args = append(args, scm.NewString("\n"))
			}
			result := getLoad("")(args...)
			want := []string{"\xff\xfeA\x00\nB"}
			if count == 1 {
				got = []string{result.String()}
			}
			if count == 3 {
				want = []string{"\xff\xfeA\x00\n", "B"}
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("got %q, want %q", got, want)
			}
			if stream.closed {
				t.Error("legacy load closed a borrowed reader")
			}
		})
	}
}

type loadFailReader struct{}

func (loadFailReader) Read([]byte) (int, error) { return 0, fmt.Errorf("input read failed") }

func TestLoadInputErrorBeforeHandler(t *testing.T) {
	called := false
	defer func() {
		if failure := recover(); failure == nil || !strings.Contains(fmt.Sprint(failure), "input read failed") {
			t.Errorf("failure = %v", failure)
		}
		if called {
			t.Error("handler called after read failure")
		}
	}()
	getLoad("")(scm.NewAny(loadFailReader{}), scm.NewFunc(func(...scm.Scmer) scm.Scmer { called = true; return scm.NewBool(true) }), scm.NewNil(), scm.NewString("bom"))
}

func TestLoadBorrowedReaderHandlerPanic(t *testing.T) {
	stream := &loadTestStream{Reader: strings.NewReader("A")}
	defer func() {
		if failure := recover(); fmt.Sprint(failure) != "handler panic" {
			t.Errorf("failure = %v", failure)
		}
		if stream.closed {
			t.Error("load closed borrowed reader during handler panic")
		}
	}()
	getLoad("")(scm.NewAny(stream), scm.NewFunc(func(...scm.Scmer) scm.Scmer { panic("handler panic") }), scm.NewNil(), scm.NewString("bom"))
}
