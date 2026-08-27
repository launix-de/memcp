/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

import "io"
import "os"
import "fmt"
import "bufio"
import "bytes"
import "strings"
import "errors"
import "path/filepath"
import "crypto/sha256"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type FileStorage struct {
	path string
}

type FileFactory struct {
	Basepath string
}

// helper for long column names
func ProcessColumnName(col string) string {
	if len(col) < 64 {
		return col
	} else {
		hashsum := sha256.Sum256([]byte(col))
		return fmt.Sprintf("%x", hashsum[:8])
	}
}

func (f *FileFactory) CreateDatabase(schema string) PersistenceEngine {
	return &FileStorage{path: f.Basepath + "/" + schema + "/"}
}

func (f *FileStorage) ReadSchema() []byte {
	jsonbytes, _ := os.ReadFile(f.path + "schema.json")
	if len(jsonbytes) == 0 {
		// try to load backup (in case of failure while save)
		jsonbytes, _ = os.ReadFile(f.path + "schema.json.old")
	}
	return jsonbytes
}

func (s *FileStorage) WriteSchema(jsonbytes []byte) {
	s.WriteSchemaWithMode(jsonbytes, true)
}

// WriteSchemaWithMode publishes filesystem schema generations atomically. The
// live path is never renamed away or truncated: a complete same-directory
// temporary file replaces it with one atomic rename. Durable writes sync the
// file before rename and the directory afterwards. The backup is a hard link
// to the previously committed generation and cannot create a live-path gap.
func (s *FileStorage) WriteSchemaWithMode(jsonbytes []byte, durable bool) {
	if err := os.MkdirAll(s.path, 0750); err != nil {
		panic(err)
	}
	tmp, err := os.CreateTemp(s.path, ".schema.json.tmp-")
	if err != nil {
		panic(err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(jsonbytes); err != nil {
		tmp.Close()
		panic(err)
	}
	if durable {
		if err := tmp.Sync(); err != nil {
			tmp.Close()
			panic(err)
		}
	}
	if err := tmp.Close(); err != nil {
		panic(err)
	}

	current := s.path + "schema.json"
	backup := s.path + "schema.json.old"
	if stat, err := os.Stat(current); err == nil && stat.Size() > 0 {
		// Publish the rescue link under a temporary name first. Failure to make
		// a backup must not disturb the still-live current generation.
		backupTmp := s.path + ".schema.json.old.tmp"
		_ = os.Remove(backupTmp)
		if err := os.Link(current, backupTmp); err == nil {
			if err := os.Rename(backupTmp, backup); err != nil {
				_ = os.Remove(backupTmp)
				panic(err)
			}
		}
	}
	if err := os.Rename(tmpName, current); err != nil {
		panic(err)
	}
	if durable {
		dir, err := os.Open(s.path)
		if err != nil {
			panic(err)
		}
		if err := dir.Sync(); err != nil {
			dir.Close()
			panic(err)
		}
		if err := dir.Close(); err != nil {
			panic(err)
		}
	}
}

func (s *FileStorage) ReadColumn(shard string, column string) io.ReadCloser {
	//f, err := os.C
	f, err := os.Open(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		// file does not exist -> no data available
		return ErrorReader{e: err, notFound: os.IsNotExist(err)}
	}
	return f
}

func (s *FileStorage) WriteColumn(shard string, column string) io.WriteCloser {
	os.MkdirAll(s.path, 0750)
	f, err := os.Create(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		panic(err)
	}
	return f
}

func (s *FileStorage) RemoveColumn(shard string, column string) {
	os.Remove(s.path + shard + "-" + ProcessColumnName(column))
}

func (s *FileStorage) blobPath(hash string) string {
	if len(hash) >= 4 {
		return s.path + "blob/" + hash[:2] + "/" + hash[2:4] + "/" + hash
	}
	return s.path + "blob/" + hash
}

func (s *FileStorage) ReadBlob(hash string) io.ReadCloser {
	f, err := os.Open(s.blobPath(hash))
	if err != nil {
		return ErrorReader{e: err, notFound: os.IsNotExist(err)}
	}
	return f
}

func (s *FileStorage) WriteBlob(hash string) io.WriteCloser {
	p := s.blobPath(hash)
	dir := p[:strings.LastIndex(p, "/")]
	os.MkdirAll(dir, 0750)
	f, err := os.CreateTemp(dir, ".blob-write-")
	if err != nil {
		panic(err)
	}
	return &fileBlobWriter{File: f, finalPath: p, directory: dir}
}

// fileBlobWriter publishes a content-addressed blob only after the complete
// payload is durable. Linking is a no-replace operation: concurrent writers of
// the same hash keep the first complete object instead of truncating it.
type fileBlobWriter struct {
	*os.File
	finalPath string
	directory string
}

func (w *fileBlobWriter) Close() error {
	if err := w.File.Sync(); err != nil {
		w.File.Close()
		os.Remove(w.File.Name())
		return err
	}
	if err := w.File.Close(); err != nil {
		os.Remove(w.File.Name())
		return err
	}
	err := os.Link(w.File.Name(), w.finalPath)
	if err != nil && !os.IsExist(err) {
		os.Remove(w.File.Name())
		return err
	}
	os.Remove(w.File.Name())
	dir, err := os.Open(w.directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (s *FileStorage) DeleteBlob(hash string) {
	os.Remove(s.blobPath(hash))
}

func (s *FileStorage) WalkBlobs(fn func(hash string) error) error {
	return filepath.Walk(s.path+"blob/", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.HasPrefix(info.Name(), ".blob-write-") {
			return nil
		}
		return fn(info.Name())
	})
}

func (s *FileStorage) WalkShardFiles(fn func(name string) error) error {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if n == "schema.json" || n == "schema.json.old" {
			continue
		}
		if err := fn(n); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) DeleteShardFile(name string) {
	os.Remove(s.path + name)
}

func (s *FileStorage) OpenLog(shard string) PersistenceLogfile {
	os.MkdirAll(s.path, 0750)
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	return FileLogfile{f}
}

func (s *FileStorage) ReplayLog(shard string) (chan interface{}, PersistenceLogfile) {
	os.MkdirAll(s.path, 0750)
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	replay := make(chan interface{}, 64)
	fi, _ := f.Stat()
	if fi.Size() > 0 {
		go func() {
			defer close(replay)
			reader := bufio.NewReaderSize(f, 256*1024)
			for {
				b, err := reader.ReadBytes('\n')
				if len(b) == 0 && errors.Is(err, io.EOF) {
					break
				}
				if len(b) > 0 && b[len(b)-1] == '\n' {
					b = b[:len(b)-1]
				}
				if len(b) > 0 && b[len(b)-1] == '\r' {
					b = b[:len(b)-1]
				}
				if len(b) == 0 && err == nil {
					// nop
				} else if len(b) >= 7 && string(b[0:7]) == "delete " {
					var idx uint32
					json.Unmarshal(b[7:], &idx)
					replay <- LogEntryDelete{idx}
				} else if len(b) >= 9 && string(b[0:9]) == "undelete " {
					var idx uint32
					json.Unmarshal(b[9:], &idx)
					replay <- LogEntryUndelete{idx}
				} else if len(b) >= 14 && string(b[0:14]) == "insert-hidden " {
					cols, values := decodeFileInsertLog(b[14:])
					replay <- LogEntryInsertHidden{cols, values}
				} else if len(b) >= 7 && string(b[0:7]) == "insert " {
					cols, values := decodeFileInsertLog(b[7:])
					replay <- LogEntryInsert{cols, values}
				} else {
					panic("unknown log sequence: " + string(b))
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					panic(err)
				}
			}
		}()
	} else {
		close(replay)
	}
	return replay, FileLogfile{f}
}

func decodeFileInsertLog(b []byte) ([]string, [][]scm.Scmer) {
	body := string(b)
	if pos := strings.Index(body, "]["); pos >= 0 {
		// new format: columns ][ values
		var cols []string
		var values [][]scm.Scmer
		json.Unmarshal([]byte(body[:pos+1]), &cols)
		json.Unmarshal([]byte(body[pos+1:]), &values)
		for i := 0; i < len(values); i++ {
			for j := 0; j < len(values[i]); j++ {
				values[i][j] = scm.TransformFromJSON(values[i][j])
			}
		}
		return cols, values
	} else {
		// fallback/old format: flat array of alternating key/value pairs -> single row
		var flat []interface{}
		if err := json.Unmarshal([]byte(body), &flat); err != nil {
			panic("unknown log sequence: " + string(b))
		}
		if len(flat)%2 != 0 {
			panic("corrupt insert log (odd items): " + string(b))
		}
		cols := make([]string, 0, len(flat)/2)
		row := make([]scm.Scmer, 0, len(flat)/2)
		for i := 0; i < len(flat); i += 2 {
			cols = append(cols, flat[i].(string))
			row = append(row, scm.TransformFromJSON(flat[i+1]))
		}
		return cols, [][]scm.Scmer{row}
	}
}

func (s *FileStorage) RemoveLog(shard string) {
	os.Remove(s.path + shard + ".log")
}

type FileLogfile struct {
	w *os.File
}

func (w FileLogfile) Write(logentry interface{}) {
	switch l := logentry.(type) {
	case LogEntryDelete:
		var b bytes.Buffer
		b.WriteString("delete ")
		tmp, _ := json.Marshal(l.idx)
		b.Write(tmp)
		b.WriteString("\n")
		w.w.Write(b.Bytes())
	case LogEntryUndelete:
		var b bytes.Buffer
		b.WriteString("undelete ")
		tmp, _ := json.Marshal(l.idx)
		b.Write(tmp)
		b.WriteString("\n")
		w.w.Write(b.Bytes())
	case LogEntryInsert:
		w.writeInsert("insert ", l.cols, l.values)
	case LogEntryInsertHidden:
		w.writeInsert("insert-hidden ", l.cols, l.values)
	}
}

func (w FileLogfile) writeInsert(prefix string, cols []string, values [][]scm.Scmer) {
	var b bytes.Buffer
	b.WriteString(prefix)
	tmp, _ := json.Marshal(cols)
	b.Write(tmp)
	tmp, _ = json.Marshal(values)
	b.Write(tmp)
	b.WriteString("\n")
	w.w.Write(b.Bytes())
}
func (w FileLogfile) Sync() {
	w.w.Sync()
}
func (w FileLogfile) Close() {
	w.w.Close()
}

func (s *FileStorage) Remove() {
	os.RemoveAll(s.path)
}

func (s *FileStorage) BackendName() string {
	return "filesystem"
}
