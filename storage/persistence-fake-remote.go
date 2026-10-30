/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
This program is free software under the GNU General Public License, version 3
or later, WITHOUT ANY WARRANTY. See <https://www.gnu.org/licenses/>.
*/
package storage

import "io"
import "os"
import "fmt"
import "time"
import "bufio"
import "context"
import "strconv"
import "path/filepath"
import "encoding/json"

// FakeRemote keeps the ordinary filesystem format and durability protocol.
// Only backend operations acquire simulated request latency and bandwidth.
// No additional decoded cache, row state or persisted timing state is introduced.
type FakeRemote struct {
	*FileStorage
	simulation *ioSimulation
}

func init() {
	BackendRegistry["FakeRemote"] = func(dbName string, raw json.RawMessage) PersistenceEngine {
		var cfg struct {
			Backend   string          `json:"backend"`
			Prefix    string          `json:"prefix"`
			Latency   json.RawMessage `json:"latency_ms"`
			Bandwidth json.RawMessage `json:"bandwidth_mbps"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			panic("FakeRemote config: " + err.Error())
		}
		number := func(raw json.RawMessage) float64 {
			if len(raw) == 0 {
				return 0
			}
			text := string(raw)
			if raw[0] == '"' {
				if err := json.Unmarshal(raw, &text); err != nil {
					panic(err)
				}
			}
			n, err := strconv.ParseFloat(text, 64)
			if err != nil {
				panic("FakeRemote numeric option: " + err.Error())
			}
			return n
		}
		profile, err := newIOSimulation(number(cfg.Bandwidth), number(cfg.Latency), &fakeRemoteCounters)
		if err != nil {
			panic(err)
		}
		prefix := cfg.Prefix
		if prefix == "" {
			prefix = dbName
		}
		if !filepath.IsAbs(prefix) {
			prefix = filepath.Join(Basepath, prefix)
		}
		return &FakeRemote{FileStorage: &FileStorage{path: filepath.Clean(prefix) + string(os.PathSeparator)}, simulation: profile}
	}
}

func (s *FakeRemote) BackendName() string { return "FakeRemote" }
func (s *FakeRemote) ReadSchema() []byte {
	s.simulation.request()
	started := time.Now()
	data := s.FileStorage.ReadSchema()
	s.simulation.transfer(context.Background(), len(data), started)
	return data
}
func (s *FakeRemote) WriteSchema(data []byte) { s.WriteSchemaWithMode(data, true) }
func (s *FakeRemote) WriteSchemaWithMode(data []byte, durable bool) {
	s.simulation.request()
	started := time.Now()
	s.FileStorage.WriteSchemaWithMode(data, durable)
	s.simulation.transfer(context.Background(), len(data), started)
}

type fakeRemoteReader struct {
	io.ReadCloser
	buffer *bufio.Reader
}

func (r *fakeRemoteReader) Read(p []byte) (int, error) { return r.buffer.Read(p) }
func (r *fakeRemoteReader) Missing() bool {
	if missing, ok := r.ReadCloser.(interface{ Missing() bool }); ok {
		return missing.Missing()
	}
	return false
}

type fakeRemoteReadStream struct {
	io.Reader
	simulation *ioSimulation
}

func (r *fakeRemoteReadStream) Read(p []byte) (int, error) {
	started := time.Now()
	n, err := r.Reader.Read(p)
	r.simulation.transfer(context.Background(), n, started)
	return n, err
}
func (s *FakeRemote) reader(reader io.ReadCloser) io.ReadCloser {
	return &fakeRemoteReader{ReadCloser: reader, buffer: bufio.NewReaderSize(&fakeRemoteReadStream{reader, s.simulation}, 64*1024)}
}
func (s *FakeRemote) ReadColumn(shard, column string) io.ReadCloser {
	s.simulation.request()
	return s.reader(s.FileStorage.ReadColumn(shard, column))
}
func (s *FakeRemote) ReadBlob(hash string) io.ReadCloser {
	s.simulation.request()
	return s.reader(s.FileStorage.ReadBlob(hash))
}
func (s *FakeRemote) ReadShardFile(name string) io.ReadCloser {
	s.simulation.request()
	return s.reader(s.FileStorage.ReadShardFile(name))
}

type fakeRemoteWriteStream struct {
	io.Writer
	simulation *ioSimulation
}

func (w *fakeRemoteWriteStream) Write(p []byte) (int, error) {
	started := time.Now()
	n, err := w.Writer.Write(p)
	w.simulation.transfer(context.Background(), n, started)
	return n, err
}

type fakeRemoteWriter struct {
	io.WriteCloser
	buffer *bufio.Writer
}

func (w *fakeRemoteWriter) Write(p []byte) (int, error) { return w.buffer.Write(p) }
func (w *fakeRemoteWriter) Sync() error {
	if err := w.buffer.Flush(); err != nil {
		return err
	}
	if syncer, ok := w.WriteCloser.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}
func (w *fakeRemoteWriter) Close() error {
	if err := w.buffer.Flush(); err != nil {
		abortPersistenceWrite(w.WriteCloser)
		return err
	}
	return w.WriteCloser.Close()
}
func (w *fakeRemoteWriter) Abort() error {
	// Discard private buffered bytes. Never flush or publish on abort.
	w.buffer.Reset(io.Discard)
	if aborter, ok := w.WriteCloser.(interface{ Abort() error }); ok {
		return aborter.Abort()
	}
	return w.WriteCloser.Close()
}
func (s *FakeRemote) writer(writer io.WriteCloser) io.WriteCloser {
	return &fakeRemoteWriter{WriteCloser: writer, buffer: bufio.NewWriterSize(&fakeRemoteWriteStream{writer, s.simulation}, 64*1024)}
}
func (s *FakeRemote) WriteColumn(shard, column string) io.WriteCloser {
	s.simulation.request()
	return s.writer(s.FileStorage.WriteColumn(shard, column))
}
func (s *FakeRemote) WriteBlob(hash string) io.WriteCloser {
	s.simulation.request()
	return s.writer(s.FileStorage.WriteBlob(hash))
}
func (s *FakeRemote) WriteShardFile(name string) io.WriteCloser {
	s.simulation.request()
	return s.writer(s.FileStorage.WriteShardFile(name))
}
func (s *FakeRemote) RemoveColumn(shard, column string) {
	s.simulation.request()
	s.FileStorage.RemoveColumn(shard, column)
}
func (s *FakeRemote) DeleteBlob(hash string) { s.simulation.request(); s.FileStorage.DeleteBlob(hash) }
func (s *FakeRemote) DeleteShardFile(name string) {
	s.simulation.request()
	s.FileStorage.DeleteShardFile(name)
}
func (s *FakeRemote) WalkBlobs(fn func(string)) { s.simulation.request(); s.FileStorage.WalkBlobs(fn) }
func (s *FakeRemote) WalkShardFiles(fn func(string)) {
	s.simulation.request()
	s.FileStorage.WalkShardFiles(fn)
}
func (s *FakeRemote) RemoveLog(shard string) { s.simulation.request(); s.FileStorage.RemoveLog(shard) }
func (s *FakeRemote) Remove()                { s.simulation.request(); s.FileStorage.Remove() }

type fakeRemoteLog struct {
	native     FileLogfile
	simulation *ioSimulation
	pending    int
}

func (l *fakeRemoteLog) Write(entry interface{}) {
	frame := encodeFileLogEntry(entry)
	l.native.writeFrame(frame)
	l.pending += len(frame)
}
func (l *fakeRemoteLog) Flush(durable bool) {
	l.simulation.request()
	started := time.Now()
	l.native.Flush(durable)
	l.simulation.transfer(context.Background(), l.pending, started)
	l.pending = 0
}
func (l *fakeRemoteLog) Close() { l.Flush(false); l.native.Close() }
func (s *FakeRemote) log(log PersistenceLogfile) PersistenceLogfile {
	return &fakeRemoteLog{native: log.(FileLogfile), simulation: s.simulation}
}
func (s *FakeRemote) OpenLog(shard string) PersistenceLogfile {
	s.simulation.request()
	return s.log(s.FileStorage.OpenLog(shard))
}
func (s *FakeRemote) logSize(shard string) int {
	stat, err := os.Stat(s.path + shard + ".log")
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		raisePersistenceFailure(s.BackendName(), s.path, "log.stat", err)
	}
	if stat.Size() > int64(int(^uint(0)>>1)) {
		raisePersistenceFailure(s.BackendName(), s.path, "log.stat", fmt.Errorf("FakeRemote WAL too large"))
	}
	return int(stat.Size())
}
func (s *FakeRemote) SwapLog(shard string, entries []interface{}, durable bool) PersistenceLogfile {
	s.simulation.request()
	started := time.Now()
	log := s.FileStorage.SwapLog(shard, entries, durable)
	s.simulation.transfer(context.Background(), s.logSize(shard), started)
	return s.log(log)
}
func (s *FakeRemote) ReplayLog(shard string) (map[string]struct{}, chan interface{}, PersistenceLogfile) {
	s.simulation.request()
	// Model one object GET before native bounded replay, retaining its commit
	// discovery and torn-frame recovery rules without a second log encoding.
	s.simulation.transfer(context.Background(), s.logSize(shard), time.Now())
	committed, entries, log := s.FileStorage.ReplayLog(shard)
	return committed, entries, s.log(log)
}

func (s *FakeRemote) CleanupAbandonedBlobWrites() {
	s.simulation.request()
	s.FileStorage.CleanupAbandonedBlobWrites()
}
