//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

// Package phpbridge registers the memcp PDO driver without replacing PDO or
// any other driver. SQL uses the same authenticated execution path as MySQL.
package phpbridge

// #include "bridge.h"
import "C"

import "fmt"
import "sync"
import "time"
import "unsafe"
import "context"
import "strconv"
import "runtime/cgo"
import "github.com/dunglas/frankenphp"
import "github.com/launix-de/memcp/scm"

var auth, schemaCheck, query, rollback scm.Scmer

// Register snapshots frontend callbacks after Scheme initialization, before
// starting PHP threads. Only the new memcp PDO driver is registered.
func Register(env *scm.Env, imapBinary string, memoryLimit int64) error {
	lookup := func(name string) scm.Scmer {
		e := env.FindRead(scm.Symbol(name))
		if e == nil {
			return scm.NewNil()
		}
		return e.Vars[scm.Symbol(name)]
	}
	auth, schemaCheck, query, rollback = lookup("mysql_auth"), lookup("mysql_schema"), lookup("mysql_handler"), lookup("tx_rollback")
	if auth.IsNil() || schemaCheck.IsNil() || query.IsNil() || rollback.IsNil() {
		return fmt.Errorf("PHP PDO requires the SQL frontend (lib/main.scm)")
	}
	// Snapshot the actually started Scheme listener before PHP workers start.
	C.memcp_set_mysql_port(0)
	if registry := lookup("service_registry"); !registry.IsNil() {
		if service := scm.Apply(registry, scm.NewString("MySQL Protocol")); !service.IsNil() {
			items := service.Slice()
			if len(items) > 0 {
				if port, err := strconv.ParseUint(items[0].String(), 10, 16); err == nil && port != 0 {
					C.memcp_set_mysql_port(C.uint(port))
				}
			}
		}
	}
	if err := registerIMAP(imapBinary, memoryLimit); err != nil {
		return err
	}
	frankenphp.RegisterExtension(C.memcp_module())
	frankenphp.RegisterExtension(C.memcp_locale_module())
	return nil
}

type connection struct {
	mu           sync.Mutex // one SQL execution or close per connection
	session      scm.Scmer
	state        *scm.SessionState
	schema       scm.Scmer
	stateValue   scm.Scmer
	buffer       resultBuffer
	fields, rows scm.Scmer
}

// Only the connection owner resets/releases buffers, after the synchronous
// SQL call has joined its workers. Result callbacks can run on different shard
// goroutines and serialize access with mu. No Go pointer is retained by C.
type resultBuffer struct {
	mu          sync.Mutex
	columns     map[string]int
	columnCount int
	metadata    bool
	cells       []C.memcp_cell
	data        []byte
	rowCount    C.size_t
}

const resultLimit = 64 << 20
const retainedResultBytes = 64 << 10

func (b *resultBuffer) release() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.columnCount > 256 {
		b.columns = nil
	} else {
		clear(b.columns)
	}
	if cap(b.cells)*int(unsafe.Sizeof(C.memcp_cell{})) > retainedResultBytes {
		b.cells = nil
	} else {
		b.cells = b.cells[:0]
	}
	if cap(b.data) > retainedResultBytes {
		b.data = nil
	} else {
		b.data = b.data[:0]
	}
	b.metadata, b.columnCount, b.rowCount = false, 0, 0
}

func (b *resultBuffer) appendValue(value scm.Scmer) C.memcp_cell {
	v := C.memcp_cell{}
	switch {
	case value.IsNil():
	case value.IsBool():
		v.kind = 4
		if value.Bool() {
			v.integer = 1
		}
	case value.IsInt():
		v.kind, v.integer = 1, C.int64_t(value.Int())
	case value.IsFloat():
		v.kind, v.number = 2, C.double(value.Float())
	default:
		s := value.String()
		if len(s) > resultLimit-len(b.data)-len(b.cells)*int(unsafe.Sizeof(v)) {
			panic("PDO result exceeds 64 MiB; use SQL LIMIT or pagination")
		}
		v.kind, v.offset, v.length = 3, C.size_t(len(b.data)), C.size_t(len(s))
		b.data = append(b.data, s...)
	}
	return v
}

// Callers hold mu. Metadata and rows share the flat C-cell layout, avoiding
// a second temporary allocation/copy for every returned row.
func (b *resultBuffer) growCells(count int) int {
	if count > (resultLimit-len(b.data))/int(unsafe.Sizeof(C.memcp_cell{}))-len(b.cells) {
		panic("PDO result exceeds 64 MiB; use SQL LIMIT or pagination")
	}
	start := len(b.cells)
	b.cells = append(b.cells, make([]C.memcp_cell, count)...)
	return start
}

func (b *resultBuffer) setColumns(titles []scm.Scmer) {
	b.metadata, b.columnCount = true, len(titles)
	if b.columns == nil {
		b.columns = make(map[string]int, len(titles))
	}
	b.growCells(len(titles))
	for i, title := range titles {
		b.columns[title.String()] = i
		b.cells[i] = b.appendValue(title)
	}
}

func (b *resultBuffer) captureFields(a ...scm.Scmer) scm.Scmer {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.metadata {
		panic("duplicate result metadata")
	}
	b.setColumns(a[0].Slice())
	return scm.NewBool(true)
}

func (b *resultBuffer) captureRow(a ...scm.Scmer) scm.Scmer {
	b.mu.Lock()
	defer b.mu.Unlock()
	row := a[0].Slice()
	if !b.metadata {
		titles := make([]scm.Scmer, 0, len(row)/2)
		for i := 0; i+1 < len(row); i += 2 {
			titles = append(titles, row[i])
		}
		b.setColumns(titles)
	}
	start := b.growCells(b.columnCount)
	for i := 0; i+1 < len(row); i += 2 {
		if j, ok := b.columns[row[i].String()]; ok {
			b.cells[start+j] = b.appendValue(row[i+1])
		}
	}
	b.rowCount++
	return scm.NewBool(true)
}

func result() *C.memcp_result {
	r := (*C.memcp_result)(C.calloc(1, C.size_t(unsafe.Sizeof(C.memcp_result{}))))
	return r
}

func fail(r *C.memcp_result, state, message string) {
	r.error = C.CString(message)
	for i := 0; i < 5; i++ {
		r.state[i] = C.char(state[i])
	}
}

// Each exported entry point contains Go panics; they must never cross Zend's
// C frames. PHP errors are raised by the C caller after Go has returned.
func catch(r *C.memcp_result) {
	if v := recover(); v != nil {
		fail(r, "HY000", fmt.Sprint(v))
	}
}

//export memcp_open
func memcp_open(db, username, password *C.char) (r *C.memcp_result) {
	r = result()
	defer catch(r)
	u, d := scm.NewString(C.GoString(username)), scm.NewString(C.GoString(db))
	p := scm.Apply(auth, u)
	if p.IsNil() || p.String() != scm.MySQLPassword(scm.NewString(C.GoString(password))).String() {
		fail(r, "28000", "MemCP authentication failed")
		return r
	}
	if !scm.Apply(schemaCheck, u, d).Bool() {
		fail(r, "3D000", "Unknown MemCP database")
		return r
	}
	s := scm.NewSession()
	s.Func()(scm.NewString("username"), u)
	s.Func()(scm.NewString("schema"), d)
	c := &connection{session: scm.NewFunc(s.Func()), state: scm.RegisterSession(u.String(), "PHP in-process", d.String()), schema: d}
	c.stateValue = scm.NewAny(c.state)
	c.fields = scm.NewFunc(c.buffer.captureFields)
	c.rows = scm.NewFunc(c.buffer.captureRow)
	r.handle = C.uintptr_t(cgo.NewHandle(c))
	return r
}

//export memcp_close
func memcp_close(handle C.uintptr_t) {
	defer func() {
		if v := recover(); v != nil {
			scm.PrintError(v)
		}
	}()
	h := cgo.Handle(handle)
	c := h.Value().(*connection)
	defer h.Delete()
	c.mu.Lock()
	defer c.mu.Unlock()
	defer scm.UnregisterSession(c.state.ID)
	defer c.state.ReleaseAllLocks()
	scm.Apply(rollback, c.session)
}

//export memcp_query
func memcp_query(handle C.uintptr_t, sql *C.char, length C.size_t) (r *C.memcp_result) {
	r = result()
	defer catch(r)
	c := cgo.Handle(handle).Value().(*connection)
	c.mu.Lock()
	defer c.mu.Unlock()
	statement := C.GoStringN(sql, C.int(length))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seq := c.state.BeginQuery("Query", statement)
	c.state.SetCancel(seq, cancel)
	c.state.SetQueryContext(seq, ctx)
	defer c.state.EndQuery(seq, "Sleep", "")
	b := &c.buffer
	defer b.release()
	ret := scm.Apply(query, c.schema, scm.NewString(statement), c.rows, c.fields, c.session, c.stateValue, scm.NewInt(int64(seq)))
	// The SQL frontend has completed all result callbacks before returning.
	b.mu.Lock()
	defer b.mu.Unlock()
	r.rows = b.rowCount
	if !b.metadata && !ret.IsNil() {
		r.affected = C.int64_t(ret.Int())
	} else {
		r.affected = C.int64_t(r.rows)
	}
	f := c.session.Func()
	r.insert_id = C.int64_t(f(scm.NewString("last_insert_id")).Int())
	if !f(scm.NewString("transaction")).IsNil() {
		r.transaction = 1
	}
	r.columns = C.size_t(b.columnCount)
	r.cells_length = C.size_t(len(b.cells))
	r.bytes_length = C.size_t(len(b.data))
	if len(b.cells) > 0 {
		r.cells = (*C.memcp_cell)(C.calloc(C.size_t(len(b.cells)), C.size_t(unsafe.Sizeof(C.memcp_cell{}))))
		copy(unsafe.Slice(r.cells, len(b.cells)), b.cells)
	}
	if len(b.data) > 0 {
		r.bytes = (*C.char)(C.CBytes(b.data))
	}
	return r
}
