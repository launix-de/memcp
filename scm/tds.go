/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "io"
import "fmt"
import "net"
import "context"
import "sync/atomic"
import tds "github.com/launix-de/go-tdsstack"

// This file only bridges native transport payloads and Scheme callbacks. SQL,
// authentication policy, parameter binding, types and transactions live in lib/.
type tdsSession struct {
	value Scmer
	state *SessionState
}

type tdsCloser struct{ server *tds.Server }

func (c tdsCloser) Close() { _ = c.server.Close() }

func tdsAssoc(value Scmer, key string) Scmer {
	pairs, dict := asAssoc(value, "TDS callback payload")
	if dict != nil {
		v, _ := dict.Get(NewString(key))
		return v
	}
	for i := 0; i < len(pairs); i += 2 {
		if pairs[i].String() == key {
			return pairs[i+1]
		}
	}
	return NewNil()
}

func tdsTypeData(t tds.WireType) Scmer {
	return NewSlice([]Scmer{NewString("kind"), NewInt(int64(t.Kind)), NewString("size"), NewInt(int64(t.Size)),
		NewString("precision"), NewInt(int64(t.Precision)), NewString("scale"), NewInt(int64(t.Scale)),
		NewString("collation"), NewString(string(t.Collation[:]))})
}

// Preserve protocol descriptors explicitly across the language boundary. In
// particular, SQL_VARIANT owns a base descriptor; never infer it from a value.
func tdsTypeFromData(data Scmer) tds.WireType {
	t := tds.WireType{Kind: byte(tdsAssoc(data, "kind").Int()), Size: int(tdsAssoc(data, "size").Int()),
		Precision: byte(tdsAssoc(data, "precision").Int()), Scale: byte(tdsAssoc(data, "scale").Int())}
	if collation := tdsAssoc(data, "collation"); !collation.IsNil() {
		copy(t.Collation[:], collation.String())
	}
	return t
}
func tdsValueData(v any) Scmer {
	switch x := v.(type) {
	case tds.Variant:
		return NewSlice([]Scmer{NewString("variant"), tdsTypeData(x.Type), NewString("value"), tdsValueData(x.Value)})
	case []byte:
		return NewString(string(x))
	case tds.Decimal:
		return NewSlice([]Scmer{NewString("coefficient"), NewString(x.Coefficient)})
	case tds.Temporal:
		return NewSlice([]Scmer{NewString("days"), NewInt(x.Days), NewString("ticks"), NewInt(x.Ticks), NewString("offset"), NewInt(x.Offset)})
	default:
		return FromAny(v)
	}
}
func tdsRequestData(r tds.Request) Scmer {
	params := make([]Scmer, len(r.Parameters))
	for i, p := range r.Parameters {
		params[i] = NewSlice([]Scmer{NewString("name"), NewString(p.Name), NewString("flags"), NewInt(int64(p.Flags)),
			NewString("type"), tdsTypeData(p.Type), NewString("value"), tdsValueData(p.Value)})
	}
	return NewSlice([]Scmer{NewString("database"), NewString(r.Database), NewString("text"), NewString(r.Text),
		NewString("declarations"), NewString(r.Declarations), NewString("procedure"), NewString(r.Procedure),
		NewString("parameters"), NewSlice(params), NewString("cursor"), NewBool(r.Cursor), NewString("browse"), NewBool(r.BrowseMetadata)})
}
func tdsColumns(data Scmer) []tds.Column {
	columns := make([]tds.Column, len(data.Slice()))
	for i, item := range data.Slice() {
		c := &columns[i]
		c.Name = tdsAssoc(item, "name").String()
		c.WireType = tdsTypeFromData(item)
		c.Flags = uint16(tdsAssoc(item, "flags").Int())
		c.UserType = uint32(tdsAssoc(item, "user_type").Int())
		c.Key = tdsAssoc(item, "key").Bool()
		c.Browse = tdsAssoc(item, "browse").Bool()
		source := tdsAssoc(item, "source")
		if !source.IsNil() {
			if len(source.Slice()) != 4 {
				panic("TDS source needs four components")
			}
			for j, s := range source.Slice() {
				c.Source[j] = s.String()
			}
		}
	}
	return columns
}

// tdsMessage validates only transport field widths, without SQL severity policy.
func tdsMessage(data Scmer) tds.Message {
	integer := func(key string, max uint64) uint32 {
		value := tdsAssoc(data, key)
		if !value.IsInt() || value.Int() < 0 || uint64(value.Int()) > max {
			panic("invalid TDS message integer: " + key)
		}
		return uint32(value.Int())
	}
	text := func(key string) string {
		value := tdsAssoc(data, key)
		if !value.IsString() {
			panic("invalid TDS message text: " + key)
		}
		return value.String()
	}
	return tds.Message{Number: integer("number", 1<<32-1), State: byte(integer("state", 255)), Severity: byte(integer("severity", 255)), Text: text("text"), Server: text("server"), Procedure: text("procedure"), Line: integer("line", 1<<32-1)}
}

// The frontend explicitly selects a token kind. Missing kind retains the
// existing INFO callback contract; severity and SQL control flow are opaque.
func tdsWriteDiagnostic(writer tds.ResultWriter, data Scmer) error {
	kind := tdsAssoc(data, "kind")
	if kind.IsNil() {
		return writer.Info(tdsMessage(data))
	}
	if !kind.IsString() {
		return fmt.Errorf("invalid TDS diagnostic kind")
	}
	switch kind.String() {
	case "info":
		return writer.Info(tdsMessage(data))
	case "error":
		sink, ok := writer.(tds.DiagnosticWriter)
		if !ok {
			return fmt.Errorf("TDS result sink does not support explicit ERROR diagnostics")
		}
		return sink.Diagnostic(tds.KindError, tdsMessage(data))
	default:
		return fmt.Errorf("invalid TDS diagnostic kind")
	}
}

// TDSDiagnosticError constructs invocation-owned transport data. SQL numbers,
// severity and the decision to throw it remain in the calling frontend.
func TDSDiagnosticError(a ...Scmer) Scmer {
	if len(a) != 1 {
		panic("tds_diagnostic_error requires one message payload")
	}
	return NewAny(&tds.DiagnosticError{Message: tdsMessage(a[0])})
}

func tdsCompletion(data Scmer) tds.Completion {
	return tds.Completion{CountRows: tdsAssoc(data, "count_rows").Bool(), Rows: uint64(tdsAssoc(data, "rows").Int()), NoCount: tdsAssoc(data, "no_count").Bool(),
		More: tdsAssoc(data, "more").Bool(), Transaction: tds.TransactionState(tdsAssoc(data, "transaction").Int()),
		Database: func() string {
			d := tdsAssoc(data, "database")
			if d.IsNil() {
				return ""
			}
			return d.String()
		}()}
}
func tdsWireValue(v Scmer) any {
	if v.IsNil() {
		return nil
	}
	if v.IsSlice() {
		if descriptor := tdsAssoc(v, "variant"); !descriptor.IsNil() {
			return tds.Variant{Type: tdsTypeFromData(descriptor), Value: tdsWireValue(tdsAssoc(v, "value"))}
		}
		if bytes := tdsAssoc(v, "bytes"); !bytes.IsNil() {
			return []byte(bytes.String())
		}
		if floating := tdsAssoc(v, "floating"); !floating.IsNil() {
			return floating.Float()
		}
		if coefficient := tdsAssoc(v, "coefficient"); !coefficient.IsNil() {
			return tds.Decimal{Coefficient: coefficient.String()}
		}
		if tdsAssoc(v, "days").IsNil() || tdsAssoc(v, "ticks").IsNil() || tdsAssoc(v, "offset").IsNil() {
			panic("unsupported TDS result payload")
		}
		return tds.Temporal{Days: tdsAssoc(v, "days").Int(), Ticks: tdsAssoc(v, "ticks").Int(), Offset: tdsAssoc(v, "offset").Int()}
	}
	return v.Any()
}
func tdsCatch(fn func()) (err error) {
	defer func() {
		if p := recover(); p != nil {
			if cause, ok := p.(error); ok {
				err = cause
			} else {
				err = fmt.Errorf("%v", p)
			}
		}
	}()
	fn()
	return
}

// TDSServe invokes one frontend callback with (event, session, payload, row,
// fields, message, state, query_seq). All descriptor/conversion decisions stay in Scheme.
func TDSServe(a ...Scmer) Scmer {
	callback := a[1]
	server := &tds.Server{Product: "MemCP", Database: a[2].String()}
	collation := a[3].String()
	if len(collation) != len(server.Collation) {
		panic("TDS session collation must contain five bytes")
	}
	copy(server.Collation[:], collation)
	call := func(ctx context.Context, event string, s *tdsSession, payload, row, fields, message Scmer) Scmer {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		seq := s.state.BeginQuery("Query", event)
		s.state.SetCancel(seq, cancel)
		s.state.SetQueryContext(seq, ctx)
		defer s.state.EndQuery(seq, "Sleep", "")
		return Apply(callback, NewString(event), s.value, payload, row, fields, message, NewAny(s.state), NewInt(int64(seq)))
	}
	server.Handler.Authenticate = func(ctx context.Context, l tds.Login) (session tds.Session, err error) {
		err = tdsCatch(func() {
			payload := NewSlice([]Scmer{NewString("username"), NewString(l.User), NewString("password"), NewString(l.Password), NewString("database"), NewString(l.Database), NewString("hostname"), NewString(l.HostName), NewString("appname"), NewString(l.AppName)})
			value := Apply(callback, NewString("login"), NewNil(), payload)
			if value.IsNil() || value.IsBool() {
				panic("login denied")
			}
			session = &tdsSession{value: value, state: RegisterSession(l.User, l.RemoteAddress, l.Database)}
		})
		return
	}
	execute := func(event string) func(context.Context, tds.Session, tds.Request, tds.ResultWriter) error {
		return func(ctx context.Context, session tds.Session, r tds.Request, w tds.ResultWriter) error {
			// Count admitted client executions, as MySQL ComQuery does. A
			// describe/cursor preparation callback is not another execution.
			atomic.AddInt64(&TotalHTTPRequests, 1)
			return tdsCatch(func() {
				row := NewFunc(func(a ...Scmer) Scmer {
					if err := ctx.Err(); err != nil {
						panic(err)
					}
					values := make([]any, len(a[0].Slice()))
					for i, v := range a[0].Slice() {
						values[i] = tdsWireValue(v)
					}
					if err := w.Row(values); err != nil {
						panic(err)
					}
					return NewBool(true)
				})
				fields := NewFunc(func(a ...Scmer) Scmer {
					if len(a) > 1 && !a[1].IsNil() {
						if err := w.Done(tdsCompletion(a[1])); err != nil {
							panic(err)
						}
					}
					if err := w.Fields(tdsColumns(a[0])); err != nil {
						panic(err)
					}
					return NewBool(true)
				})
				message := NewFunc(func(a ...Scmer) Scmer {
					if err := tdsWriteDiagnostic(w, a[0]); err != nil {
						panic(err)
					}
					return NewBool(true)
				})
				completion := call(ctx, event, session.(*tdsSession), tdsRequestData(r), row, fields, message)
				if !completion.IsNil() {
					if err := w.Done(tdsCompletion(completion)); err != nil {
						panic(err)
					}
				}
			})
		}
	}
	server.Handler.Execute = execute("execute")
	server.Handler.Metadata = execute("metadata")
	server.Handler.Bulk = func(ctx context.Context, session tds.Session, bulk *tds.BulkReader) (completion tds.Completion, err error) {
		// The stream is one admission, independent of its row count/lifetime.
		atomic.AddInt64(&TotalHTTPRequests, 1)
		err = tdsCatch(func() {
			columns := bulk.Columns()
			metadata := make([]Scmer, len(columns))
			for i, column := range columns {
				fields := append([]Scmer(nil), tdsTypeData(column.WireType).Slice()...)
				metadata[i] = NewSlice(append(fields, NewString("name"), NewString(column.Name), NewString("flags"), NewInt(int64(column.Flags)), NewString("user_type"), NewInt(int64(column.UserType))))
			}
			var active, consumed atomic.Bool
			active.Store(true)
			defer active.Store(false)
			consume := NewFunc(func(a ...Scmer) Scmer {
				if !active.Load() || !consumed.CompareAndSwap(false, true) {
					panic("bulk consumer is outside its request or already consumed")
				}
				var count int64
				for {
					values, readErr := bulk.Next()
					if readErr == io.EOF {
						return NewInt(count)
					}
					if readErr != nil {
						panic(readErr)
					}
					row := make([]Scmer, len(values))
					for i, value := range values {
						row[i] = tdsValueData(value)
					}
					Apply(a[0], NewSlice(row))
					count++
				}
			})
			payload := NewSlice([]Scmer{NewString("columns"), NewSlice(metadata), NewString("consume"), consume})
			completion = tdsCompletion(call(ctx, "bulk", session.(*tdsSession), payload, NewNil(), NewNil(), NewNil()))
		})
		return
	}
	server.Handler.State = func(ctx context.Context, s tds.Session) (completion tds.Completion, err error) {
		err = tdsCatch(func() {
			ss := s.(*tdsSession)
			completion = tdsCompletion(Apply(callback, NewString("state"), ss.value, NewNil(), NewNil(), NewNil(), NewNil(), NewAny(ss.state), NewInt(0)))
		})
		return
	}
	server.Handler.Describe = func(ctx context.Context, s tds.Session, r tds.Request) (columns []tds.Column, err error) {
		err = tdsCatch(func() {
			columns = tdsColumns(call(ctx, "describe", s.(*tdsSession), tdsRequestData(r), NewNil(), NewNil(), NewNil()))
		})
		return
	}
	server.Handler.Transaction = func(ctx context.Context, s tds.Session, r tds.TransactionRequest) (completion tds.Completion, err error) {
		err = tdsCatch(func() {
			completion = tdsCompletion(call(ctx, "transaction", s.(*tdsSession), NewSlice([]Scmer{NewString("action"), NewInt(int64(r.Action)), NewString("isolation"), NewInt(int64(r.Isolation))}), NewNil(), NewNil(), NewNil()))
		})
		return
	}
	server.Handler.Reset = func(ctx context.Context, s tds.Session) (tds.Session, error) {
		err := tdsCatch(func() {
			ss := s.(*tdsSession)
			replacement := call(ctx, "reset", ss, NewNil(), NewNil(), NewNil(), NewNil())
			if SessionCleanupHook != nil {
				SessionCleanupHook(ss.value)
			}
			ss.value = replacement
		})
		return s, err
	}
	server.Handler.Close = func(s tds.Session) {
		ss := s.(*tdsSession)
		defer UnregisterSession(ss.state.ID)
		defer ss.state.ReleaseAllLocks()
		_ = tdsCatch(func() { Apply(callback, NewString("close"), ss.value) })
		if SessionCleanupHook != nil {
			_ = tdsCatch(func() { SessionCleanupHook(ss.value) })
		}
	}
	listener, err := net.Listen("tcp", ":"+a[0].String())
	if err != nil {
		panic(err)
	}
	mysqlListenersMu.Lock()
	mysqlListeners = append(mysqlListeners, tdsCloser{server})
	mysqlListenersMu.Unlock()
	go func() {
		defer server.Close()
		defer listener.Close()
		<-waitForMySQLInitialization()
		_ = server.Serve(listener)
	}()
	return NewBool(true)
}
