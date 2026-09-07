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
import "strconv"
import "syscall"
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

func init() {
	BackendRegistry["filesystem"] = func(dbName string, raw json.RawMessage) PersistenceEngine {
		return (&FileFactory{Basepath: Basepath}).CreateDatabase(dbName)
	}
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
	jsonbytes, err := os.ReadFile(f.path + "schema.json")
	if err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure(f.BackendName(), f.path, "schema.read", err)
	}
	if len(jsonbytes) == 0 {
		// try to load backup (in case of failure while save)
		jsonbytes, err = os.ReadFile(f.path + "schema.json.old")
		if err != nil && !os.IsNotExist(err) {
			raisePersistenceFailure(f.BackendName(), f.path, "schema.read", err)
		}
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
		raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
	}
	tmp, err := os.CreateTemp(s.path, ".schema.json.tmp-")
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(jsonbytes); err != nil {
		tmp.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
	}
	if durable {
		if err := tmp.Sync(); err != nil {
			tmp.Close()
			raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
		}
	}
	if err := tmp.Close(); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
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
				raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
			}
		} else {
			reportPersistenceCleanupFailure(s.BackendName(), s.path, "schema.backup", err)
		}
	}
	if err := os.Rename(tmpName, current); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
	}
	if durable {
		dir, err := os.Open(s.path)
		if err != nil {
			raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
		}
		if err := dir.Sync(); err != nil {
			dir.Close()
			raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
		}
		if err := dir.Close(); err != nil {
			raisePersistenceFailure(s.BackendName(), s.path, "schema.write", err)
		}
	}
}

func (s *FileStorage) ReadColumn(shard string, column string) io.ReadCloser {
	//f, err := os.C
	f, err := os.Open(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		if !os.IsNotExist(err) {
			raisePersistenceFailure(s.BackendName(), s.path, "column.read.open", err)
		}
		// file does not exist -> no data available
		return ErrorReader{e: err, notFound: os.IsNotExist(err)}
	}
	return standardPersistenceReader(f, s.BackendName(), s.path, "column.read")
}

func (s *FileStorage) WriteColumn(shard string, column string) io.WriteCloser {
	if err := os.MkdirAll(s.path, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "column.write.open", err)
	}
	f, err := os.Create(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "column.write.open", err)
	}
	return &fileObjectWriter{File: f, database: s.path, operation: "column.write"}
}

func (s *FileStorage) RemoveColumn(shard string, column string) {
	if err := os.Remove(s.path + shard + "-" + ProcessColumnName(column)); err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure(s.BackendName(), s.path, "column.remove", err)
	}
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
		if !os.IsNotExist(err) {
			raisePersistenceFailure(s.BackendName(), s.path, "blob.read.open", err)
		}
		return ErrorReader{e: err, notFound: os.IsNotExist(err)}
	}
	return standardPersistenceReader(f, s.BackendName(), s.path, "blob.read")
}

func (s *FileStorage) WriteBlob(hash string) io.WriteCloser {
	p := s.blobPath(hash)
	dir := p[:strings.LastIndex(p, "/")]
	if err := os.MkdirAll(dir, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "blob.write.open", err)
	}
	f, err := os.CreateTemp(dir, ".blob-write-")
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "blob.write.open", err)
	}
	return &fileBlobWriter{File: f, finalPath: p, directory: dir, database: s.path}
}

type fileObjectWriter struct {
	*os.File
	database  string
	operation string
}

func (w *fileObjectWriter) Write(buffer []byte) (int, error) {
	n, err := w.File.Write(buffer)
	if err != nil || n != len(buffer) {
		if err == nil {
			err = io.ErrShortWrite
		}
		raisePersistenceFailure("filesystem", w.database, w.operation, err)
	}
	return n, nil
}

func (w *fileObjectWriter) Sync() error {
	if err := w.File.Sync(); err != nil {
		raisePersistenceFailure("filesystem", w.database, w.operation+".sync", err)
	}
	return nil
}

func (w *fileObjectWriter) Close() error {
	if err := w.File.Close(); err != nil {
		raisePersistenceFailure("filesystem", w.database, w.operation+".close", err)
	}
	return nil
}

// fileBlobWriter publishes a content-addressed blob only after the complete
// payload is durable. Linking is a no-replace operation: concurrent writers of
// the same hash keep the first complete object instead of truncating it.
type fileBlobWriter struct {
	*os.File
	finalPath string
	directory string
	database  string
}

func (w *fileBlobWriter) Write(buffer []byte) (int, error) {
	n, err := w.File.Write(buffer)
	if err != nil || n != len(buffer) {
		if err == nil {
			err = io.ErrShortWrite
		}
		raisePersistenceFailure("filesystem", w.database, "blob.write", err)
	}
	return n, nil
}

func (w *fileBlobWriter) Close() error {
	if err := w.File.Sync(); err != nil {
		w.File.Close()
		os.Remove(w.File.Name())
		raisePersistenceFailure("filesystem", w.database, "blob.write.sync", err)
	}
	if err := w.File.Close(); err != nil {
		os.Remove(w.File.Name())
		raisePersistenceFailure("filesystem", w.database, "blob.write.close", err)
	}
	err := os.Link(w.File.Name(), w.finalPath)
	if err != nil && !os.IsExist(err) {
		os.Remove(w.File.Name())
		raisePersistenceFailure("filesystem", w.database, "blob.write.publish", err)
	}
	os.Remove(w.File.Name())
	dir, err := os.Open(w.directory)
	if err != nil {
		raisePersistenceFailure("filesystem", w.database, "blob.write.sync", err)
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		raisePersistenceFailure("filesystem", w.database, "blob.write.sync", err)
	}
	if err := dir.Close(); err != nil {
		raisePersistenceFailure("filesystem", w.database, "blob.write.close", err)
	}
	return nil
}

func (s *FileStorage) DeleteBlob(hash string) {
	if err := os.Remove(s.blobPath(hash)); err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure(s.BackendName(), s.path, "blob.delete", err)
	}
}

func (s *FileStorage) WalkBlobs(fn func(hash string)) {
	err := filepath.Walk(s.path+"blob/", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".blob-write-") {
			return nil
		}
		fn(info.Name())
		return nil
	})
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "blob.walk", err)
	}
}

func (s *FileStorage) WalkShardFiles(fn func(name string)) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "shard.walk", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if n == "schema.json" || n == "schema.json.old" {
			continue
		}
		fn(n)
	}
}

func validShardFileName(name string) bool {
	return name != "" && filepath.Base(name) == name && name != "." && name != ".."
}

func (s *FileStorage) ReadShardFile(name string) io.ReadCloser {
	if !validShardFileName(name) {
		panic("invalid shard file name")
	}
	f, err := os.Open(s.path + name)
	if err != nil {
		if !os.IsNotExist(err) {
			raisePersistenceFailure(s.BackendName(), s.path, "shard.read.open", err)
		}
		return ErrorReader{e: err, notFound: os.IsNotExist(err)}
	}
	return standardPersistenceReader(f, s.BackendName(), s.path, "shard.read")
}

func (s *FileStorage) WriteShardFile(name string) io.WriteCloser {
	if !validShardFileName(name) {
		panic("invalid shard file name")
	}
	if err := os.MkdirAll(s.path, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "shard.write.open", err)
	}
	f, err := os.Create(s.path + name)
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "shard.write.open", err)
	}
	return &fileObjectWriter{File: f, database: s.path, operation: "shard.write"}
}

func (s *FileStorage) DeleteShardFile(name string) {
	if err := os.Remove(s.path + name); err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure(s.BackendName(), s.path, "shard.delete", err)
	}
}

func (s *FileStorage) OpenLog(shard string) PersistenceLogfile {
	if err := os.MkdirAll(s.path, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.open", err)
	}
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.open", err)
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "log.open", err)
	}
	return FileLogfile{w: f}
}

func (s *FileStorage) SwapLog(shard string, entries []interface{}, durable bool) PersistenceLogfile {
	if err := os.MkdirAll(s.path, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
	}
	tmp, err := os.CreateTemp(s.path, "."+shard+".log.tmp-")
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
	}
	tmpName := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpName)
		}
	}()

	for _, entry := range entries {
		if _, err := tmp.Write(encodeFileLogEntry(entry)); err != nil {
			_ = tmp.Close()
			raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
		}
	}
	if durable {
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
		}
	}
	if _, err := tmp.Seek(0, io.SeekEnd); err != nil {
		_ = tmp.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
	}
	var dir *os.File
	if durable {
		dir, err = os.Open(s.path)
		if err != nil {
			_ = tmp.Close()
			raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
		}
	}
	if err := os.Rename(tmpName, s.path+shard+".log"); err != nil {
		if dir != nil {
			_ = dir.Close()
		}
		_ = tmp.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "log.swap", err)
	}
	removeTmp = false
	if dir != nil {
		// PersistenceLogfile cannot report an ambiguous outcome after rename.
		// Match the existing best-effort Sync contract while still issuing the
		// directory barrier needed to persist the chosen complete generation.
		if err := dir.Sync(); err != nil {
			reportPersistenceAmbiguousFailure(s.BackendName(), s.path, "log.swap.sync", err)
		}
		if err := dir.Close(); err != nil {
			reportPersistenceCleanupFailure(s.BackendName(), s.path, "log.swap.close", err)
		}
	}
	return FileLogfile{w: tmp}
}

func (s *FileStorage) ReplayLog(shard string) (map[string]struct{}, chan interface{}, PersistenceLogfile) {
	if err := os.MkdirAll(s.path, 0750); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.replay", err)
	}
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.replay", err)
	}
	committed := fileCommittedTransactions(f)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "log.replay", err)
	}
	replay := make(chan interface{}, 64)
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		raisePersistenceFailure(s.BackendName(), s.path, "log.replay", err)
	}
	if fi.Size() > 0 {
		go func() {
			defer close(replay)
			reader := bufio.NewReaderSize(f, 256*1024)
			for {
				b, err := reader.ReadBytes('\n')
				if len(b) == 0 && errors.Is(err, io.EOF) {
					break
				}
				complete := len(b) > 0 && b[len(b)-1] == '\n'
				if errors.Is(err, io.EOF) && !complete {
					// A record without its delimiter may be a torn final write. It
					// was never a complete WAL frame and must not affect recovery.
					break
				}
				if complete {
					b = b[:len(b)-1]
				}
				if len(b) > 0 && b[len(b)-1] == '\r' {
					b = b[:len(b)-1]
				}
				if len(b) == 0 && err == nil {
					// nop
				} else if len(b) >= 10 && string(b[0:10]) == "commit-tx " {
					fields := strings.Fields(string(b[10:]))
					if len(fields) != 2 || fields[0] == "" || fields[1] != fileCommitChecksum(fields[0]) {
						panic("corrupt commit log: " + string(b))
					}
					replay <- LogEntryCommit{txID: fields[0]}
				} else if len(b) >= 10 && string(b[0:10]) == "delete-tx " {
					txID, idx := decodeFileDeleteTx(b[10:])
					replay <- LogEntryDelete{idx: idx, txID: txID}
				} else if len(b) >= 12 && string(b[0:12]) == "undelete-tx " {
					txID, idx := decodeFileDeleteTx(b[12:])
					replay <- LogEntryUndelete{idx: idx, txID: txID}
				} else if len(b) >= 17 && string(b[0:17]) == "insert-hidden-tx " {
					txID, payload := decodeFileTxPrefix(b[17:])
					cols, values := decodeFileInsertLog(payload)
					replay <- LogEntryInsertHidden{cols: cols, values: values, txID: txID}
				} else if len(b) >= 10 && string(b[0:10]) == "insert-tx " {
					txID, payload := decodeFileTxPrefix(b[10:])
					cols, values := decodeFileInsertLog(payload)
					replay <- LogEntryInsert{cols: cols, values: values, txID: txID}
				} else if len(b) >= 7 && string(b[0:7]) == "delete " {
					var idx uint32
					if decodeErr := json.Unmarshal(b[7:], &idx); decodeErr != nil {
						panic("corrupt delete log: " + string(b))
					}
					replay <- LogEntryDelete{idx: idx}
				} else if len(b) >= 9 && string(b[0:9]) == "undelete " {
					var idx uint32
					if decodeErr := json.Unmarshal(b[9:], &idx); decodeErr != nil {
						panic("corrupt undelete log: " + string(b))
					}
					replay <- LogEntryUndelete{idx: idx}
				} else if len(b) >= 14 && string(b[0:14]) == "insert-hidden " {
					cols, values := decodeFileInsertLog(b[14:])
					replay <- LogEntryInsertHidden{cols: cols, values: values}
				} else if len(b) >= 7 && string(b[0:7]) == "insert " {
					cols, values := decodeFileInsertLog(b[7:])
					replay <- LogEntryInsert{cols: cols, values: values}
				} else {
					panic("unknown log sequence: " + string(b))
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					raisePersistenceFailure(s.BackendName(), s.path, "log.replay", err)
				}
			}
		}()
	} else {
		close(replay)
	}
	return committed, replay, FileLogfile{w: f}
}

func fileCommittedTransactions(f *os.File) map[string]struct{} {
	committed := make(map[string]struct{})
	reader := bufio.NewReaderSize(f, 256*1024)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.EOF) && (len(line) == 0 || line[len(line)-1] != '\n') {
			break
		}
		line = bytes.TrimSuffix(line, []byte{'\n'})
		line = bytes.TrimSuffix(line, []byte{'\r'})
		if bytes.HasPrefix(line, []byte("commit-tx ")) {
			fields := strings.Fields(string(line[len("commit-tx "):]))
			if len(fields) != 2 || fields[0] == "" || fields[1] != fileCommitChecksum(fields[0]) {
				panic("corrupt commit log: " + string(line))
			}
			committed[fields[0]] = struct{}{}
		}
		if err != nil {
			raisePersistenceFailure("filesystem", f.Name(), "log.replay", err)
		}
	}
	return committed
}

func decodeFileTxPrefix(payload []byte) (string, []byte) {
	separator := bytes.IndexByte(payload, ' ')
	if separator <= 0 {
		panic("corrupt transactional WAL entry: " + string(payload))
	}
	txID := string(payload[:separator])
	if txID == "" {
		panic("corrupt transactional WAL id: " + string(payload))
	}
	return txID, payload[separator+1:]
}

func fileCommitChecksum(txID string) string {
	checksum := sha256.Sum256([]byte(txID))
	return fmt.Sprintf("%x", checksum[:])
}

func decodeFileDeleteTx(payload []byte) (string, uint32) {
	txID, recidPayload := decodeFileTxPrefix(payload)
	idx, err := strconv.ParseUint(string(recidPayload), 10, 32)
	if err != nil {
		panic("corrupt transactional visibility log: " + string(payload))
	}
	return txID, uint32(idx)
}

func decodeFileInsertLog(b []byte) ([]string, [][]scm.Scmer) {
	body := string(b)
	if pos := strings.Index(body, "]["); pos >= 0 {
		// new format: columns ][ values
		var cols []string
		var values [][]scm.Scmer
		if err := json.Unmarshal([]byte(body[:pos+1]), &cols); err != nil {
			panic("corrupt insert columns log: " + string(b))
		}
		if err := json.Unmarshal([]byte(body[pos+1:]), &values); err != nil {
			panic("corrupt insert values log: " + string(b))
		}
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
	if err := os.Remove(s.path + shard + ".log"); err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure(s.BackendName(), s.path, "log.remove", err)
	}
}

type FileLogfile struct {
	w *os.File
}

func (w FileLogfile) Write(logentry interface{}) {
	frame := encodeFileLogEntry(logentry)
	if len(frame) == 0 {
		return
	}
	start, err := w.w.Seek(0, io.SeekCurrent)
	if err != nil {
		raisePersistenceFailure("filesystem", w.w.Name(), "log.write", err)
	}
	n, err := w.w.Write(frame)
	if err == nil && n == len(frame) {
		return
	}
	if err == nil {
		err = io.ErrShortWrite
	}
	if truncateErr := w.w.Truncate(start); truncateErr != nil {
		err = fmt.Errorf("%w; WAL rollback failed: %v", err, truncateErr)
	}
	if _, seekErr := w.w.Seek(start, io.SeekStart); seekErr != nil {
		err = fmt.Errorf("%w; WAL seek rollback failed: %v", err, seekErr)
	}
	raisePersistenceFailure("filesystem", w.w.Name(), "log.write", err)
}

func (w FileLogfile) writePartial(logentry interface{}) error {
	frame := encodeFileLogEntry(logentry)
	if len(frame) == 0 {
		return syscall.ENOSPC
	}
	start, err := w.w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	prefix := len(frame) / 2
	if prefix == 0 {
		prefix = 1
	}
	if _, err := w.w.Write(frame[:prefix]); err != nil {
		return err
	}
	if err := w.w.Truncate(start); err != nil {
		return err
	}
	if _, err := w.w.Seek(start, io.SeekStart); err != nil {
		return err
	}
	return syscall.ENOSPC
}

func encodeFileLogEntry(logentry interface{}) []byte {
	var b bytes.Buffer
	switch l := logentry.(type) {
	case LogEntryDelete:
		if l.txID != "" {
			fmt.Fprintf(&b, "delete-tx %s %d\n", l.txID, l.idx)
			break
		}
		b.WriteString("delete ")
		tmp, _ := json.Marshal(l.idx)
		b.Write(tmp)
		b.WriteString("\n")
	case LogEntryUndelete:
		if l.txID != "" {
			fmt.Fprintf(&b, "undelete-tx %s %d\n", l.txID, l.idx)
			break
		}
		b.WriteString("undelete ")
		tmp, _ := json.Marshal(l.idx)
		b.Write(tmp)
		b.WriteString("\n")
	case LogEntryInsert:
		encodeFileInsert(&b, "insert ", l.txID, l.cols, l.values)
	case LogEntryInsertHidden:
		encodeFileInsert(&b, "insert-hidden ", l.txID, l.cols, l.values)
	case LogEntryCommit:
		fmt.Fprintf(&b, "commit-tx %s %s\n", l.txID, fileCommitChecksum(l.txID))
	}
	return b.Bytes()
}

func encodeFileInsert(b *bytes.Buffer, prefix string, txID string, cols []string, values [][]scm.Scmer) {
	if txID == "" {
		b.WriteString(prefix)
	} else {
		b.WriteString(strings.TrimSuffix(prefix, " "))
		b.WriteString("-tx ")
		b.WriteString(txID)
		b.WriteByte(' ')
	}
	tmp, _ := json.Marshal(cols)
	b.Write(tmp)
	tmp, _ = json.Marshal(values)
	b.Write(tmp)
	b.WriteString("\n")
}
func (w FileLogfile) Flush(durable bool) {
	if durable {
		if err := w.w.Sync(); err != nil {
			raisePersistenceFailure("filesystem", w.w.Name(), "log.sync", err)
		}
	}
}
func (w FileLogfile) Close() {
	if err := w.w.Close(); err != nil {
		raisePersistenceFailure("filesystem", w.w.Name(), "log.close", err)
	}
}

func (s *FileStorage) Remove() {
	if err := os.RemoveAll(s.path); err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "database.remove", err)
	}
}

func (s *FileStorage) BackendName() string {
	return "filesystem"
}

func (s *FileStorage) StorageIdentity() string {
	return "filesystem:" + filepath.Clean(s.path)
}
