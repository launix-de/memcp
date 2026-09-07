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

import "io"
import "os"
import "fmt"
import "sync"
import "math/rand"
import "strconv"
import "strings"
import "syscall"

type persistenceFaultInjector struct {
	mu          sync.Mutex
	rng         *rand.Rand
	probability float64
	operations  map[string]struct{}
	phase       string
	skip        uint64
	remaining   int64
}

func persistenceFaultInjectorFromEnv(database string) *persistenceFaultInjector {
	probabilityText := os.Getenv("MEMCP_IO_FAULT_PROBABILITY")
	if probabilityText == "" {
		return nil
	}
	probability, err := strconv.ParseFloat(probabilityText, 64)
	if err != nil || probability <= 0 || probability > 1 {
		panic(fmt.Sprintf("invalid MEMCP_IO_FAULT_PROBABILITY %q", probabilityText))
	}
	if filter := os.Getenv("MEMCP_IO_FAULT_DATABASE"); filter != "" && filter != database {
		return nil
	}
	seed := int64(1)
	if text := os.Getenv("MEMCP_IO_FAULT_SEED"); text != "" {
		parsed, parseErr := strconv.ParseInt(text, 10, 64)
		if parseErr != nil {
			panic(fmt.Sprintf("invalid MEMCP_IO_FAULT_SEED %q", text))
		}
		seed = parsed
	}
	operations := make(map[string]struct{})
	for _, operation := range strings.Split(os.Getenv("MEMCP_IO_FAULT_OPERATIONS"), ",") {
		operation = strings.TrimSpace(operation)
		if operation != "" {
			operations[operation] = struct{}{}
		}
	}
	if len(operations) == 0 {
		operations["*"] = struct{}{}
	}
	phase := os.Getenv("MEMCP_IO_FAULT_PHASE")
	if phase == "" {
		phase = "before"
	}
	if phase != "before" && phase != "partial" {
		panic(fmt.Sprintf("invalid MEMCP_IO_FAULT_PHASE %q", phase))
	}
	remaining := int64(-1)
	if text := os.Getenv("MEMCP_IO_FAULT_LIMIT"); text != "" {
		parsed, parseErr := strconv.ParseInt(text, 10, 64)
		if parseErr != nil || parsed < 1 {
			panic(fmt.Sprintf("invalid MEMCP_IO_FAULT_LIMIT %q", text))
		}
		remaining = parsed
	}
	var skip uint64
	if text := os.Getenv("MEMCP_IO_FAULT_AFTER"); text != "" {
		parsed, parseErr := strconv.ParseUint(text, 10, 64)
		if parseErr != nil {
			panic(fmt.Sprintf("invalid MEMCP_IO_FAULT_AFTER %q", text))
		}
		skip = parsed
	}
	return &persistenceFaultInjector{
		rng: rand.New(rand.NewSource(seed)), probability: probability,
		operations: operations, phase: phase, skip: skip, remaining: remaining,
	}
}

func (f *persistenceFaultInjector) trigger(operation string, phase string) bool {
	if f == nil || f.phase != phase {
		return false
	}
	if _, all := f.operations["*"]; !all {
		if _, selected := f.operations[operation]; !selected {
			return false
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.remaining == 0 {
		return false
	}
	if f.skip > 0 {
		f.skip--
		return false
	}
	if f.rng.Float64() >= f.probability {
		return false
	}
	if f.remaining > 0 {
		f.remaining--
	}
	return true
}

type faultPersistence struct {
	PersistenceEngine
	database string
	faults   *persistenceFaultInjector
}

func instrumentPersistence(database string, engine PersistenceEngine) PersistenceEngine {
	faults := persistenceFaultInjectorFromEnv(database)
	if faults == nil {
		return engine
	}
	return &faultPersistence{PersistenceEngine: engine, database: database, faults: faults}
}

func (p *faultPersistence) fail(operation string, phase string) {
	if p.faults.trigger(operation, phase) {
		raisePersistenceFailure(p.BackendName(), p.database, operation, syscall.ENOSPC)
	}
}

func (p *faultPersistence) around(operation string, fn func()) {
	p.fail(operation, "before")
	// Non-stream operations have no meaningful partial public state. Their
	// backend implementation remains responsible for atomic publication.
	p.fail(operation, "partial")
	fn()
}

func (p *faultPersistence) ReadSchema() (result []byte) {
	p.around("schema.read", func() { result = p.PersistenceEngine.ReadSchema() })
	return result
}

func (p *faultPersistence) WriteSchema(schema []byte) {
	p.around("schema.write", func() { p.PersistenceEngine.WriteSchema(schema) })
}

func (p *faultPersistence) WriteSchemaWithMode(schema []byte, durable bool) {
	p.around("schema.write", func() {
		if writer, ok := p.PersistenceEngine.(schemaWriteOptions); ok {
			writer.WriteSchemaWithMode(schema, durable)
		} else {
			p.PersistenceEngine.WriteSchema(schema)
		}
	})
}

func (p *faultPersistence) ReadColumn(shard string, column string) io.ReadCloser {
	var result io.ReadCloser
	p.around("column.read.open", func() { result = p.PersistenceEngine.ReadColumn(shard, column) })
	return &faultReadCloser{ReadCloser: result, owner: p, operation: "column.read"}
}

func (p *faultPersistence) WriteColumn(shard string, column string) io.WriteCloser {
	var result io.WriteCloser
	p.around("column.write.open", func() { result = p.PersistenceEngine.WriteColumn(shard, column) })
	return &faultWriteCloser{WriteCloser: result, owner: p, operation: "column.write"}
}

func (p *faultPersistence) RemoveColumn(shard string, column string) {
	p.around("column.remove", func() { p.PersistenceEngine.RemoveColumn(shard, column) })
}

func (p *faultPersistence) ReadBlob(hash string) io.ReadCloser {
	var result io.ReadCloser
	p.around("blob.read.open", func() { result = p.PersistenceEngine.ReadBlob(hash) })
	return &faultReadCloser{ReadCloser: result, owner: p, operation: "blob.read"}
}

func (p *faultPersistence) WriteBlob(hash string) io.WriteCloser {
	var result io.WriteCloser
	p.around("blob.write.open", func() { result = p.PersistenceEngine.WriteBlob(hash) })
	return &faultWriteCloser{WriteCloser: result, owner: p, operation: "blob.write"}
}

func (p *faultPersistence) DeleteBlob(hash string) {
	p.around("blob.delete", func() { p.PersistenceEngine.DeleteBlob(hash) })
}

func (p *faultPersistence) WalkBlobs(fn func(hash string)) {
	p.around("blob.walk", func() { p.PersistenceEngine.WalkBlobs(fn) })
}

func (p *faultPersistence) OpenLog(shard string) (result PersistenceLogfile) {
	p.around("log.open", func() { result = p.PersistenceEngine.OpenLog(shard) })
	return &faultLogfile{PersistenceLogfile: result, owner: p}
}

func (p *faultPersistence) SwapLog(shard string, entries []interface{}, durable bool) (result PersistenceLogfile) {
	p.around("log.swap", func() { result = p.PersistenceEngine.SwapLog(shard, entries, durable) })
	return &faultLogfile{PersistenceLogfile: result, owner: p}
}

func (p *faultPersistence) ReplayLog(shard string) (committed map[string]struct{}, entries chan interface{}, result PersistenceLogfile) {
	p.around("log.replay", func() {
		committed, entries, result = p.PersistenceEngine.ReplayLog(shard)
	})
	return committed, entries, &faultLogfile{PersistenceLogfile: result, owner: p}
}

func (p *faultPersistence) RemoveLog(shard string) {
	p.around("log.remove", func() { p.PersistenceEngine.RemoveLog(shard) })
}

func (p *faultPersistence) WalkShardFiles(fn func(name string)) {
	p.around("shard.walk", func() { p.PersistenceEngine.WalkShardFiles(fn) })
}

func (p *faultPersistence) ReadShardFile(name string) io.ReadCloser {
	var result io.ReadCloser
	p.around("shard.read.open", func() { result = p.PersistenceEngine.ReadShardFile(name) })
	return &faultReadCloser{ReadCloser: result, owner: p, operation: "shard.read"}
}

func (p *faultPersistence) WriteShardFile(name string) io.WriteCloser {
	var result io.WriteCloser
	p.around("shard.write.open", func() { result = p.PersistenceEngine.WriteShardFile(name) })
	return &faultWriteCloser{WriteCloser: result, owner: p, operation: "shard.write"}
}

func (p *faultPersistence) DeleteShardFile(name string) {
	p.around("shard.delete", func() { p.PersistenceEngine.DeleteShardFile(name) })
}

func (p *faultPersistence) Remove() {
	p.around("database.remove", p.PersistenceEngine.Remove)
}

func (p *faultPersistence) StorageIdentity() string {
	return p.PersistenceEngine.StorageIdentity()
}

type faultReadCloser struct {
	io.ReadCloser
	owner     *faultPersistence
	operation string
}

func (r *faultReadCloser) Missing() bool {
	if missing, ok := r.ReadCloser.(interface{ Missing() bool }); ok {
		return missing.Missing()
	}
	return false
}

func (r *faultReadCloser) Read(buffer []byte) (n int, err error) {
	r.owner.fail(r.operation, "before")
	if r.owner.faults.trigger(r.operation, "partial") && len(buffer) > 1 {
		n, err = r.ReadCloser.Read(buffer[:len(buffer)/2])
		raisePersistenceFailure(r.owner.BackendName(), r.owner.database, r.operation, syscall.EIO)
	}
	n, err = r.ReadCloser.Read(buffer)
	if err != nil && err != io.EOF {
		if r.Missing() {
			return n, err
		}
		raisePersistenceFailure(r.owner.BackendName(), r.owner.database, r.operation, err)
	}
	return n, err
}

func (r *faultReadCloser) Close() error {
	operation := r.operation + ".close"
	r.owner.fail(operation, "before")
	if err := r.ReadCloser.Close(); err != nil {
		raisePersistenceFailure(r.owner.BackendName(), r.owner.database, operation, err)
	}
	return nil
}

type faultWriteCloser struct {
	io.WriteCloser
	owner     *faultPersistence
	operation string
}

func (w *faultWriteCloser) Write(buffer []byte) (n int, err error) {
	w.owner.fail(w.operation, "before")
	if w.owner.faults.trigger(w.operation, "partial") && len(buffer) > 0 {
		prefix := len(buffer) / 2
		if prefix == 0 {
			prefix = 1
		}
		_, _ = w.WriteCloser.Write(buffer[:prefix])
		raisePersistenceFailure(w.owner.BackendName(), w.owner.database, w.operation, syscall.ENOSPC)
	}
	n, err = w.WriteCloser.Write(buffer)
	if err != nil || n != len(buffer) {
		if err == nil {
			err = io.ErrShortWrite
		}
		raisePersistenceFailure(w.owner.BackendName(), w.owner.database, w.operation, err)
	}
	return n, nil
}

func (w *faultWriteCloser) Close() error {
	w.owner.fail(w.operation+".close", "before")
	err := w.WriteCloser.Close()
	if err != nil {
		raisePersistenceFailure(w.owner.BackendName(), w.owner.database, w.operation+".close", err)
	}
	return nil
}

type partialPersistenceLogfile interface {
	writePartial(logentry interface{}) error
}

type faultLogfile struct {
	PersistenceLogfile
	owner *faultPersistence
}

func (w *faultLogfile) Write(logentry interface{}) {
	w.owner.fail("log.write", "before")
	if w.owner.faults.trigger("log.write", "partial") {
		if partial, ok := w.PersistenceLogfile.(partialPersistenceLogfile); ok {
			if err := partial.writePartial(logentry); err != nil {
				raisePersistenceFailure(w.owner.BackendName(), w.owner.database, "log.write", err)
			}
		}
		raisePersistenceFailure(w.owner.BackendName(), w.owner.database, "log.write", syscall.ENOSPC)
	}
	w.PersistenceLogfile.Write(logentry)
}

func (w *faultLogfile) Flush(durable bool) {
	operation := "log.flush"
	if durable {
		operation = "log.sync"
	}
	w.owner.around(operation, func() { w.PersistenceLogfile.Flush(durable) })
}

func (w *faultLogfile) Close() {
	w.owner.around("log.close", w.PersistenceLogfile.Close)
}
