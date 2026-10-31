/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

package scm

import "net"
import "time"
import "runtime"
import "strings"
import "testing"
import "github.com/launix-de/go-mysqlstack/xlog"
import "github.com/launix-de/go-mysqlstack/driver"
import querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"

func TestMySQLInitializationGate(t *testing.T) {
	BeginMySQLInitialization()
	ready := waitForMySQLInitialization()
	select {
	case <-ready:
		t.Fatal("MySQL initialization gate opened before bootstrap completed")
	default:
	}

	CompleteMySQLInitialization()
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("MySQL initialization gate stayed closed after bootstrap completed")
	}
}

func TestMySQLClientErrorMessageDropsInternalStackAndFitsCommandBuffer(t *testing.T) {
	stack := strings.Repeat("runtime stack frame\n", 1000)
	got := mysqlClientErrorMessage("NthLocalVar(2) out of range (len=0)\n" + stack)
	if got != "NthLocalVar(2) out of range (len=0)" {
		t.Fatalf("unexpected client error %q", got)
	}
	if len(got) > mysqlClientErrorMessageLimit {
		t.Fatalf("client error uses %d bytes, limit is %d", len(got), mysqlClientErrorMessageLimit)
	}
}

func TestMySQLClientErrorMessageBoundsSingleLineErrors(t *testing.T) {
	got := mysqlClientErrorMessage(strings.Repeat("x", mysqlClientErrorMessageLimit+100))
	if len(got) != mysqlClientErrorMessageLimit {
		t.Fatalf("client error uses %d bytes, want %d", len(got), mysqlClientErrorMessageLimit)
	}
}

func TestAppendMySQLResultRowDuplicateAliasUsesLastValueType(t *testing.T) {
	var fields []*querypb.Field
	colmap := map[string]int{}

	row, unknown := prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("x"), NewInt(1),
		NewString("x"), NewString("EUR"),
	}, nil, false, true)

	if unknown {
		t.Fatal("first row unexpectedly reported an unknown column")
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].Type != querypb.Type_VARCHAR {
		t.Fatalf("expected varchar metadata, got %v", fields[0].Type)
	}
	if fields[0].Charset != 45 {
		t.Fatalf("expected utf8mb4 charset, got %d", fields[0].Charset)
	}
	if got := row[0].String(); got != "EUR" {
		t.Fatalf("expected last duplicate value, got %q", got)
	}
}

func TestPrepareMySQLResultRowPadsMissingAndRejectsNewPublishedColumns(t *testing.T) {
	fields := []*querypb.Field{{Name: "a", Type: querypb.Type_INT64}, {Name: "b", Type: querypb.Type_VARCHAR}}
	colmap := map[string]int{"a": 0, "b": 1}
	row := []Scmer{NewInt(1), NewString("old")}

	row, unknown := prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("b"), NewString("new"),
		NewString("c"), NewString("ignored"),
	}, row, true, false)

	if !unknown {
		t.Fatal("new column after publishing fields was not reported")
	}
	if !row[0].IsNil() {
		t.Fatalf("missing column was not padded with NULL: %v", row[0])
	}
	if got := row[1].String(); got != "new" {
		t.Fatalf("known column got %q, want new", got)
	}
}

func TestPrepareMySQLResultRowRefinesInitiallyNullMetadata(t *testing.T) {
	var fields []*querypb.Field
	colmap := map[string]int{}

	row, _ := prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("value"), NewNil(),
	}, nil, false, true)
	if fields[0].Type != querypb.Type_NULL_TYPE {
		t.Fatalf("initial NULL has type %v, want NULL_TYPE", fields[0].Type)
	}

	row, _ = prepareMySQLResultRow(&fields, colmap, []Scmer{
		NewString("value"), NewInt(42),
	}, row, true, true)
	if fields[0].Type != querypb.Type_INT64 {
		t.Fatalf("later integer has type %v, want INT64", fields[0].Type)
	}
}

func BenchmarkPrepareMySQLResultRow10PublishedColumns(b *testing.B) {
	fields := make([]*querypb.Field, 10)
	colmap := make(map[string]int, 10)
	item := make([]Scmer, 0, 20)
	row := make([]Scmer, 10)
	for i := 0; i < 10; i++ {
		name := string(rune('a' + i))
		fields[i] = &querypb.Field{Name: name, Type: querypb.Type_INT64}
		colmap[name] = i
		item = append(item, NewString(name), NewInt(int64(i)))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		row, _ = prepareMySQLResultRow(&fields, colmap, item, row, true, false)
	}
}

func TestMySQLServerVersionHasClientCompatiblePrefix(t *testing.T) {
	if got := (&MySQLWrapper{}).ServerVersion(); got != "5.7.44-MemCP" {
		t.Fatalf("unexpected MySQL protocol version %q", got)
	}
}

func TestPrepareMySQLResultFieldsPreservesCompilerOrderAndDuplicates(t *testing.T) {
	fields, colmap, row := prepareMySQLResultFields([]Scmer{
		NewString("id"), NewString("value"), NewString("value"),
	})
	if len(fields) != 3 || len(row) != 3 {
		t.Fatalf("prepared %d fields and %d row slots, want 3 each", len(fields), len(row))
	}
	if fields[0].Name != "id" || fields[1].Name != "value" || fields[2].Name != "value" {
		t.Fatalf("compiler field order was not preserved: %+v", fields)
	}
	for _, field := range fields {
		if field.Type != querypb.Type_NULL_TYPE {
			t.Fatalf("unobserved field %q has type %v, want NULL_TYPE", field.Name, field.Type)
		}
	}
	if colmap["id"] != 0 || colmap["value"] != 2 {
		t.Fatalf("unexpected fallback column map: %+v", colmap)
	}
}

// mysqlLifecycleProbe observes real driver events while keeping the production
// wrapper responsible for Scheme sessions, locks and process-list ownership.
type mysqlLifecycleProbe struct {
	*MySQLWrapper
	opened chan *driver.Session
	closed chan *driver.Session
}

func (p *mysqlLifecycleProbe) NewSession(session *driver.Session) {
	p.MySQLWrapper.NewSession(session)
	p.opened <- session
}

func (p *mysqlLifecycleProbe) SessionClosed(session *driver.Session) {
	p.MySQLWrapper.SessionClosed(session)
	p.closed <- session
}

func TestMySQLConnectionCloseReleasesSession(t *testing.T) {
	log := xlog.NewStdLog(xlog.Level(xlog.ERROR))
	probe := &mysqlLifecycleProbe{
		MySQLWrapper: &MySQLWrapper{
			log: log,
			authcallback: NewFunc(func(...Scmer) Scmer {
				return NewString(string(driver.CreatePassword("admin")))
			}),
		},
		opened: make(chan *driver.Session, 1),
		closed: make(chan *driver.Session, 1),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := driver.NewListenerFromNetListener(log, listener, probe)
	// Protocol listeners must preserve the embedding application's scheduler.
	// Choose a limit different from NumCPU so the old driver bug is visible.
	configuredProcs := 1
	if runtime.NumCPU() == configuredProcs {
		configuredProcs = 2
	}
	procs := runtime.GOMAXPROCS(configuredProcs)
	stopped := make(chan struct{})
	go func() { defer close(stopped); server.Accept() }()
	t.Cleanup(func() { server.Close(); <-stopped; runtime.GOMAXPROCS(procs) })

	httpKey := t.Name()
	httpSession := RegisterSession("root", "http", "")
	httpStates.Store(httpKey, httpSession)
	t.Cleanup(func() { EvictHTTPSession(httpKey) })

	for _, mode := range []string{"quit", "disconnect", "auth failure"} {
		t.Run(mode, func(t *testing.T) {
			password := "admin"
			if mode == "auth failure" {
				password = "wrong"
			}
			client, err := driver.NewConn("root", password, listener.Addr().String(), "", "")
			if client != nil {
				t.Cleanup(client.Cleanup)
			}
			if mode == "auth failure" {
				if err == nil {
					client.Close()
					t.Fatal("invalid password was accepted")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			var session *driver.Session
			select {
			case session = <-probe.opened:
				t.Cleanup(session.Close)
			case <-time.After(3 * time.Second):
				t.Fatal("MySQL session was not registered")
			}
			if got := runtime.GOMAXPROCS(0); got != configuredProcs {
				t.Fatalf("MySQL listener changed GOMAXPROCS to %d; want %d", got, configuredProcs)
			}
			var state *SessionState
			unlocked := make(chan struct{}, 1)
			if mode != "auth failure" {
				value, ok := mysqlStates.Load(session)
				if !ok {
					t.Fatal("live MySQL session is missing from the process list")
				}
				state = value.(*SessionState)
				state.AddLock(func() { unlocked <- struct{}{} })
				if mode == "disconnect" {
					client.Cleanup()
				} else {
					client.Close()
				}
			}
			select {
			case closed := <-probe.closed:
				if closed != session {
					t.Fatal("another MySQL session was closed")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("MySQL session was not unregistered")
			}
			select {
			case <-session.Done():
			default:
				t.Fatal("application session was removed but driver session remained alive")
			}
			if _, ok := mysqlsessions.Load(session); ok {
				t.Fatal("closed connection retained its Scheme session")
			}
			if _, ok := mysqlStates.Load(session); ok {
				t.Fatal("closed connection retained its process-list mapping")
			}
			if state != nil {
				if _, ok := processList.Load(state.ID); ok {
					t.Fatal("closed connection remained in the process list")
				}
				select {
				case <-unlocked:
				default:
					t.Fatal("closed connection retained its table locks")
				}
			}
			if value, ok := httpStates.Load(httpKey); !ok || value != httpSession {
				t.Fatal("MySQL connection close removed an independent persistent HTTP session")
			}
			if _, ok := processList.Load(httpSession.ID); !ok {
				t.Fatal("persistent HTTP session was prematurely unregistered")
			}
		})
	}
}
