/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "fmt"
import "net"
import "context"
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
func tdsValueData(v any) Scmer {
	switch x := v.(type) {
	case []byte:
		return NewString(string(x))
	case tds.Decimal:
		return NewSlice([]Scmer{NewString("coefficient"), NewString(x.Coefficient)})
	case tds.Temporal:
		return NewSlice([]Scmer{NewString("days"), NewInt(x.Days), NewString("ticks"), NewInt(x.Ticks), NewString("offset"), NewInt(x.Offset)})
	default:
		return NewAny(v)
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
		c.Kind = byte(tdsAssoc(item, "kind").Int())
		c.Size = int(tdsAssoc(item, "size").Int())
		c.Scale = byte(tdsAssoc(item, "scale").Int())
		c.Precision = byte(tdsAssoc(item, "precision").Int())
		c.Flags = uint16(tdsAssoc(item, "flags").Int())
		c.UserType = uint32(tdsAssoc(item, "user_type").Int())
		c.Key = tdsAssoc(item, "key").Bool()
		c.Browse = tdsAssoc(item, "browse").Bool()
		collation := tdsAssoc(item, "collation")
		if !collation.IsNil() {
			copy(c.Collation[:], collation.String())
		}
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
func tdsCompletion(data Scmer) tds.Completion {
	return tds.Completion{Rows: uint64(tdsAssoc(data, "rows").Int()), NoCount: tdsAssoc(data, "no_count").Bool(),
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
			err = fmt.Errorf("%v", p)
		}
	}()
	fn()
	return
}

// TDSServe invokes one frontend callback with (event, session, payload, row,
// fields, state, query_seq). All descriptor/conversion decisions stay in Scheme.
func TDSServe(a ...Scmer) Scmer {
	callback := a[1]
	server := &tds.Server{Product: "MemCP", Database: a[2].String()}
	call := func(ctx context.Context, event string, s *tdsSession, payload, row, fields Scmer) Scmer {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		seq := s.state.BeginQuery("Query", event)
		s.state.SetCancel(seq, cancel)
		s.state.SetQueryContext(seq, ctx)
		defer s.state.EndQuery(seq, "Sleep", "")
		return Apply(callback, NewString(event), s.value, payload, row, fields, NewAny(s.state), NewInt(int64(seq)))
	}
	server.Handler.Authenticate = func(ctx context.Context, l tds.Login) (session tds.Session, err error) {
		err = tdsCatch(func() {
			payload := NewSlice([]Scmer{NewString("username"), NewString(l.User), NewString("password"), NewString(l.Password), NewString("database"), NewString(l.Database)})
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
					if err := w.Fields(tdsColumns(a[0])); err != nil {
						panic(err)
					}
					return NewBool(true)
				})
				completion := call(ctx, event, session.(*tdsSession), tdsRequestData(r), row, fields)
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
	server.Handler.Describe = func(ctx context.Context, s tds.Session, r tds.Request) (columns []tds.Column, err error) {
		err = tdsCatch(func() {
			columns = tdsColumns(call(ctx, "describe", s.(*tdsSession), tdsRequestData(r), NewNil(), NewNil()))
		})
		return
	}
	server.Handler.Transaction = func(ctx context.Context, s tds.Session, r tds.TransactionRequest) (completion tds.Completion, err error) {
		err = tdsCatch(func() {
			completion = tdsCompletion(call(ctx, "transaction", s.(*tdsSession), NewSlice([]Scmer{NewString("action"), NewInt(int64(r.Action)), NewString("isolation"), NewInt(int64(r.Isolation))}), NewNil(), NewNil()))
		})
		return
	}
	server.Handler.Reset = func(ctx context.Context, s tds.Session) (tds.Session, error) {
		err := tdsCatch(func() { ss := s.(*tdsSession); ss.value = call(ctx, "reset", ss, NewNil(), NewNil(), NewNil()) })
		return s, err
	}
	server.Handler.Close = func(s tds.Session) {
		ss := s.(*tdsSession)
		defer UnregisterSession(ss.state.ID)
		defer ss.state.ReleaseAllLocks()
		_ = tdsCatch(func() { Apply(callback, NewString("close"), ss.value) })
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
