//go:build ceph

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

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ceph/go-ceph/rados"
	"github.com/launix-de/memcp/scm"
)

func init() {
	BackendRegistry["ceph"] = func(dbName string, raw json.RawMessage) PersistenceEngine {
		var cfg struct {
			UserName    string `json:"username"`
			ClusterName string `json:"cluster"`
			ConfFile    string `json:"conf_file"`
			Pool        string `json:"pool"`
			Prefix      string `json:"prefix"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			panic("ceph backend: invalid config: " + err.Error())
		}
		factory := &CephFactory{
			UserName:    cfg.UserName,
			ClusterName: cfg.ClusterName,
			ConfFile:    cfg.ConfFile,
			Pool:        cfg.Pool,
			Prefix:      cfg.Prefix,
		}
		return factory.CreateDatabase(dbName)
	}
}

// Ceph/RADOS layout
//  - schema:   <prefix>/schema.json
//  - column:   <prefix>/<shard>-<colhash>
//  - logs:     <prefix>/<shard>.log.<seg8>
//             segments are append-only, we always write to the last segment.
//
// Rationale:
//  - RADOS does not provide "append" API; but it does allow writes at an offset.
//  - segmenting avoids unbounded single-object growth, and allows quicker replay slicing.

type CephFactory struct {
	// These are intentionally minimal. You can extend with mon host list,
	// conf file path, keyring, etc.
	UserName    string // e.g. "client.admin" or "client.memcp"
	ClusterName string // often "ceph"
	ConfFile    string // optional
	Pool        string // e.g. "memcp"
	Prefix      string // base prefix; per database you can join with schema name
}

func (f *CephFactory) CreateDatabase(schema string) PersistenceEngine {
	pfx := path.Join(strings.TrimSuffix(f.Prefix, "/"), schema)
	return NewCephStorage(f, pfx)
}

type CephStorage struct {
	factory *CephFactory
	prefix  string

	mu     sync.Mutex
	conn   *rados.Conn
	ioctx  *rados.IOContext
	opened bool

}

func NewCephStorage(f *CephFactory, prefix string) *CephStorage {
	return &CephStorage{factory: f, prefix: prefix}
}

func (s *CephStorage) ensureOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opened {
		return
	}

	conn, err := rados.NewConnWithClusterAndUser(s.factory.ClusterName, s.factory.UserName)
	if err != nil {
		panic(err)
	}
	if s.factory.ConfFile != "" {
		if err := conn.ReadConfigFile(s.factory.ConfFile); err != nil {
			panic(err)
		}
	} else {
		// If no conf provided, caller must have CEPH_ARGS/CEPH_CONF env or defaults.
		_ = conn.ReadDefaultConfigFile()
	}

	if err := conn.Connect(); err != nil {
		panic(err)
	}

	ioctx, err := conn.OpenIOContext(s.factory.Pool)
	if err != nil {
		conn.Shutdown()
		panic(err)
	}

	s.conn = conn
	s.ioctx = ioctx
	s.opened = true
}

func (s *CephStorage) obj(name string) string {
	return path.Join(s.prefix, name)
}

func (s *CephStorage) ReadSchema() []byte {
	s.ensureOpen()
	obj := s.obj("schema.json")
	// First stat to get size
	stat, err := s.ioctx.Stat(obj)
	if err != nil {
		// Keep behavior similar to FileStorage: empty means "not found"
		return nil
	}
	data := make([]byte, stat.Size)
	n, err := s.ioctx.Read(obj, data, 0)
	if err != nil {
		return nil
	}
	return data[:n]
}

func (s *CephStorage) WriteSchema(schema []byte) {
	s.ensureOpen()
	obj := s.obj("schema.json")
	// atomic overwrite
	if err := s.ioctx.WriteFull(obj, schema); err != nil {
		panic(err)
	}
}

func (s *CephStorage) ReadColumn(shard string, column string) io.ReadCloser {
	s.ensureOpen()
	obj := s.obj(shard + "-" + ProcessColumnName(column))

	// We implement ReadCloser backed by in-memory buffer.
	// If you want streaming, you'd need chunked reads; for now columns are usually loaded fully anyway.
	stat, err := s.ioctx.Stat(obj)
	if err != nil {
		return ErrorReader{err}
	}
	data := make([]byte, stat.Size)
	n, err := s.ioctx.Read(obj, data, 0)
	if err != nil {
		return ErrorReader{err}
	}
	return io.NopCloser(bytes.NewReader(data[:n]))
}

type cephWriteCloser struct {
	s      *CephStorage
	obj    string
	buf    bytes.Buffer
	closed bool
}

func (w *cephWriteCloser) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.buf.Write(p)
}

func (w *cephWriteCloser) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	// atomic overwrite
	if err := w.s.ioctx.WriteFull(w.obj, w.buf.Bytes()); err != nil {
		return err
	}
	return nil
}

func (s *CephStorage) WriteColumn(shard string, column string) io.WriteCloser {
	s.ensureOpen()
	obj := s.obj(shard + "-" + ProcessColumnName(column))
	return &cephWriteCloser{s: s, obj: obj}
}

func (s *CephStorage) RemoveColumn(shard string, column string) {
	s.ensureOpen()
	_ = s.ioctx.Delete(s.obj(shard + "-" + ProcessColumnName(column)))
}

func (s *CephStorage) ReadBlob(hash string) io.ReadCloser {
	s.ensureOpen()
	obj := s.obj("blob/" + hash)
	stat, err := s.ioctx.Stat(obj)
	if err != nil {
		return ErrorReader{err}
	}
	data := make([]byte, stat.Size)
	n, err := s.ioctx.Read(obj, data, 0)
	if err != nil {
		return ErrorReader{err}
	}
	return io.NopCloser(bytes.NewReader(data[:n]))
}

func (s *CephStorage) WriteBlob(hash string) io.WriteCloser {
	s.ensureOpen()
	obj := s.obj("blob/" + hash)
	return &cephWriteCloser{s: s, obj: obj}
}

func (s *CephStorage) DeleteBlob(hash string) {
	s.ensureOpen()
	_ = s.ioctx.Delete(s.obj("blob/" + hash))
}

func (s *CephStorage) Remove() {
	// With plain librados we can't efficiently list "all objects under prefix"
	// unless we also maintain an index object / omap with object names.
	// For MVP: not implemented.
	// Recommendation: keep a "manifest" object listing all objects in the db prefix.
	panic("CephStorage.Remove not implemented: needs a manifest/index to enumerate objects")
}

// -------------------- Logs --------------------
//
// Important: RADOS has no append() primitive; we implement append by:
//  - stat object to get current size
//  - write at offset size (WriteOp with offset)
//  - update offset in-memory
//
// Bottleneck avoidance:
//  - batch entries in memory buffer
//  - write in larger chunks on Sync()
//  - segment logs so stat/read doesn't scale badly and replay can be chunked

type CephLogfile struct {
	s     *CephStorage
	shard string

	mu     sync.Mutex
	seg    uint32
	obj    string
	offset uint64 // current append offset within segment
	buf    bytes.Buffer

	// knobs
	flushEveryBytes int
}

func (s *CephStorage) OpenLog(shard string) PersistenceLogfile {
	s.ensureOpen()
	lf, err := openOrCreateCephLogfile(s, shard)
	if err != nil {
		panic(err)
	}
	return lf
}

func (s *CephStorage) ReplayLog(shard string) (chan interface{}, PersistenceLogfile) {
	s.ensureOpen()

	out := make(chan interface{}, 64)

	// Replay existing segments synchronously; caller consumes channel.
	go func() {
		defer close(out)
		segments, err := listLogSegments(s, shard)
		if err != nil {
			// no logs yet -> nothing to replay
			return
		}
		sort.Slice(segments, func(i, j int) bool { return segments[i].seg < segments[j].seg })

		for _, seg := range segments {
			stat, err := s.ioctx.Stat(seg.obj)
			if err != nil || stat.Size == 0 {
				continue
			}
			data := make([]byte, stat.Size)
			n, err := s.ioctx.Read(seg.obj, data, 0)
			if err != nil || n == 0 {
				continue
			}
			// decode framed records
			decodeLogStream(data[:n], out)
		}
	}()

	// Return an appendable logfile as second return, like FileStorage does.
	lf, err := openOrCreateCephLogfile(s, shard)
	if err != nil {
		panic(err)
	}
	return out, lf
}

func (s *CephStorage) RemoveLog(shard string) {
	s.ensureOpen()
	segments, _ := listLogSegments(s, shard)
	for _, seg := range segments {
		_ = s.ioctx.Delete(seg.obj)
	}
}

// --- log segment helpers

type logSegInfo struct {
	seg uint32
	obj string
}

func listLogSegments(s *CephStorage, shard string) ([]logSegInfo, error) {
	// librados enumeration is possible, but expensive and pool-wide.
	// For MVP we use a small manifest object per shard.
	//
	// shard log manifest: <prefix>/<shard>.log.manifest
	// contents: JSON array of segment numbers that exist.
	//
	// This keeps list operation O(segments), avoids pool-wide scans.

	manifestObj := s.obj(fmt.Sprintf("%s.log.manifest", shard))
	stat, err := s.ioctx.Stat(manifestObj)
	if err != nil || stat.Size == 0 {
		return nil, fmt.Errorf("no manifest")
	}
	raw := make([]byte, stat.Size)
	n, err := s.ioctx.Read(manifestObj, raw, 0)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("no manifest")
	}
	var segs []uint32
	if err := json.Unmarshal(raw[:n], &segs); err != nil {
		return nil, err
	}

	out := make([]logSegInfo, 0, len(segs))
	for _, seg := range segs {
		out = append(out, logSegInfo{
			seg: seg,
			obj: s.obj(fmt.Sprintf("%s.log.%08d", shard, seg)),
		})
	}
	return out, nil
}

func writeLogManifest(s *CephStorage, shard string, segs []uint32) error {
	manifestObj := s.obj(fmt.Sprintf("%s.log.manifest", shard))
	raw, _ := json.Marshal(segs)
	return s.ioctx.WriteFull(manifestObj, raw)
}

func openOrCreateCephLogfile(s *CephStorage, shard string) (*CephLogfile, error) {
	// Load manifest (or create with seg0)
	segs, _ := listLogSegments(s, shard)
	var seg uint32
	var all []uint32
	if len(segs) == 0 {
		seg = 0
		all = []uint32{0}
		if err := writeLogManifest(s, shard, all); err != nil {
			return nil, err
		}
	} else {
		// last segment
		sort.Slice(segs, func(i, j int) bool { return segs[i].seg < segs[j].seg })
		seg = segs[len(segs)-1].seg
		all = make([]uint32, 0, len(segs))
		for _, si := range segs {
			all = append(all, si.seg)
		}
	}

	obj := s.obj(fmt.Sprintf("%s.log.%08d", shard, seg))

	// Determine current size as append offset
	st, err := s.ioctx.Stat(obj)
	if err != nil {
		// object may not exist yet -> create empty using Truncate
		if err := s.ioctx.Truncate(obj, 0); err != nil {
			return nil, err
		}
		st, _ = s.ioctx.Stat(obj)
	}

	return &CephLogfile{
		s:               s,
		shard:           shard,
		seg:             seg,
		obj:             obj,
		offset:          uint64(st.Size),
		flushEveryBytes: 256 * 1024, // 256KB batches; tune
	}, nil
}

// --- Log encoding: framed JSON (MVP)
//
// Record format:
//   u32 len (little endian) + payload bytes
// Payload is JSON object: {"t":"delete","idx":...} or {"t":"insert","cols":[...],"values":[[...] ...]}
//
// Why framed instead of newline?
// - robust against embedded newlines
// - can be decoded with one pass
//
// Later optimization:
// - replace JSON with a binary codec (varint + column-id encoding + typed values).

type encDelete struct {
	T   string `json:"t"`
	Idx uint   `json:"idx"`
}
type encInsert struct {
	T      string        `json:"t"`
	Cols   []string      `json:"cols"`
	Values [][]scm.Scmer `json:"values"`
}

func encodeLogEntry(e interface{}) []byte {
	var payload []byte
	switch l := e.(type) {
	case LogEntryDelete:
		tmp, _ := json.Marshal(encDelete{T: "delete", Idx: l.idx})
		payload = tmp
	case LogEntryInsert:
		tmp, _ := json.Marshal(encInsert{T: "insert", Cols: l.cols, Values: l.values})
		payload = tmp
	default:
		// ignore unknown
		return nil
	}

	var frame bytes.Buffer
	_ = binary.Write(&frame, binary.LittleEndian, uint32(len(payload)))
	frame.Write(payload)
	return frame.Bytes()
}

func decodeLogStream(data []byte, out chan interface{}) {
	i := 0
	for {
		if i+4 > len(data) {
			return
		}
		n := int(binary.LittleEndian.Uint32(data[i : i+4]))
		i += 4
		if n <= 0 || i+n > len(data) {
			return // truncated/corrupt tail; ignore
		}
		payload := data[i : i+n]
		i += n

		// decode union
		var head struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal(payload, &head); err != nil {
			continue
		}
		switch head.T {
		case "delete":
			var d encDelete
			if json.Unmarshal(payload, &d) == nil {
				out <- LogEntryDelete{idx: uint(d.Idx)}
			}
		case "insert":
			var ins encInsert
			if json.Unmarshal(payload, &ins) == nil {
				// Transform values from JSON (same as FileStorage)
				for r := 0; r < len(ins.Values); r++ {
					for c := 0; c < len(ins.Values[r]); c++ {
						ins.Values[r][c] = scm.TransformFromJSON(ins.Values[r][c])
					}
				}
				out <- LogEntryInsert{cols: ins.Cols, values: ins.Values}
			}
		}
	}
}

func (w *CephLogfile) Write(logentry interface{}) {
	frame := encodeLogEntry(logentry)
	if len(frame) == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(frame)

	// optional auto-flush: prevents unbounded buffer in write-heavy workloads
	if w.buf.Len() >= w.flushEveryBytes {
		// best-effort flush without forcing caller to call Sync()
		_ = w.flushLocked(false)
	}
}

func (w *CephLogfile) Sync() {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.flushLocked(true)
}

func (w *CephLogfile) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.flushLocked(true)
	// no explicit handle to close for rados object
}

func (w *CephLogfile) flushLocked(force bool) error {
	if w.buf.Len() == 0 {
		return nil
	}

	// Segment roll-over (simple heuristic):
	// if segment exceeds ~64MB, start a new one.
	const maxSeg = 64 * 1024 * 1024
	if w.offset+uint64(w.buf.Len()) > maxSeg {
		// create next segment and update manifest
		next := w.seg + 1
		nextObj := w.s.obj(fmt.Sprintf("%s.log.%08d", w.shard, next))
		if err := w.s.ioctx.Truncate(nextObj, 0); err != nil {
			return err
		}
		// update manifest
		segs, _ := listLogSegments(w.s, w.shard)
		var all []uint32
		for _, si := range segs {
			all = append(all, si.seg)
		}
		all = append(all, next)
		_ = writeLogManifest(w.s, w.shard, all)

		w.seg = next
		w.obj = nextObj
		w.offset = 0
	}

	// write at current offset (append)
	payload := w.buf.Bytes()

	// Use a WriteOp so we can later extend with flags / op batching.
	op := rados.CreateWriteOp()
	defer op.Release()
	op.Write(payload, uint64(w.offset))
	// RADOS has no fsync; durability depends on replication and client ack.
	// For "Sync semantics" you mainly want to flush buffers and maybe block until op completes.
	if err := op.Operate(w.s.ioctx, w.obj, rados.OperationNoFlag); err != nil {
		// If op fails, keep buffer (caller can retry). We won't clear.
		return err
	}

	w.offset += uint64(len(payload))
	w.buf.Reset()

	// If "force" is requested, add a tiny sleep to coalesce? (optional)
	// In practice, you'd implement group commit above this layer.
	if force {
		time.Sleep(0) // placeholder, keep deterministic
	}
	return nil
}
