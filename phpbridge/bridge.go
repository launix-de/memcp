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
import "runtime/cgo"
import "github.com/dunglas/frankenphp"
import "github.com/launix-de/memcp/scm"

var auth, schemaCheck, query, rollback scm.Scmer

// Register snapshots frontend callbacks after Scheme initialization, before
// starting PHP threads. Only the new memcp PDO driver is registered.
func Register(env *scm.Env) error {
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
	frankenphp.RegisterExtension(C.memcp_module())
	return nil
}

type connection struct {
	mu      sync.Mutex // one SQL execution or close per connection
	session scm.Scmer
	state   *scm.SessionState
	schema  string
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
	c := &connection{session: scm.NewFunc(s.Func()), state: scm.RegisterSession(u.String(), "PHP in-process", d.String()), schema: d.String()}
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
	var mu sync.Mutex
	var names []string
	var cells []C.memcp_cell
	var data []byte
	var columns map[string]int
	const maxBytes = 64 << 20
	appendValue := func(value scm.Scmer) C.memcp_cell {
		v := C.memcp_cell{}
		switch {
		case value.IsNil():
		case value.IsBool():
			v.kind = 4
			if value.Bool() {
				v.integer = 1
			}
		case value.IsInt():
			v.kind = 1
			v.integer = C.int64_t(value.Int())
		case value.IsFloat():
			v.kind = 2
			v.number = C.double(value.Float())
		default:
			s := value.String()
			v.kind = 3
			v.offset = C.size_t(len(data))
			v.length = C.size_t(len(s))
			data = append(data, s...)
		}
		return v
	}
	setColumns := func(titles []scm.Scmer) {
		columns = make(map[string]int, len(titles))
		for i, title := range titles {
			names = append(names, title.String())
			columns[title.String()] = i
			cells = append(cells, appendValue(title))
		}
	}
	fields := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		mu.Lock()
		defer mu.Unlock()
		if columns != nil {
			panic("duplicate result metadata")
		}
		setColumns(a[0].Slice())
		return scm.NewBool(true)
	})
	rows := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		mu.Lock()
		defer mu.Unlock()
		row := a[0].Slice()
		if columns == nil {
			titles := make([]scm.Scmer, 0, len(row)/2)
			for i := 0; i < len(row); i += 2 {
				titles = append(titles, row[i])
			}
			setColumns(titles)
		}
		values := make([]C.memcp_cell, len(names))
		for i := 0; i+1 < len(row); i += 2 {
			if j, ok := columns[row[i].String()]; ok {
				values[j] = appendValue(row[i+1])
			}
		}
		cells = append(cells, values...)
		if len(data)+len(cells)*int(unsafe.Sizeof(C.memcp_cell{})) > maxBytes {
			panic("PDO result exceeds 64 MiB; use SQL LIMIT or pagination")
		}
		r.rows++
		return scm.NewBool(true)
	})
	ret := scm.Apply(query, scm.NewString(c.schema), scm.NewString(statement), rows, fields, c.session, scm.NewAny(c.state), scm.NewInt(int64(seq)))
	if columns == nil && !ret.IsNil() {
		r.affected = C.int64_t(ret.Int())
	} else {
		r.affected = C.int64_t(r.rows)
	}
	f := c.session.Func()
	r.insert_id = C.int64_t(f(scm.NewString("last_insert_id")).Int())
	if !f(scm.NewString("transaction")).IsNil() {
		r.transaction = 1
	}
	r.columns = C.size_t(len(names))
	if len(cells) > 0 {
		r.cells = (*C.memcp_cell)(C.calloc(C.size_t(len(cells)), C.size_t(unsafe.Sizeof(C.memcp_cell{}))))
		copy(unsafe.Slice(r.cells, len(cells)), cells)
	}
	if len(data) > 0 {
		r.bytes = (*C.char)(C.CBytes(data))
	}
	return r
}
