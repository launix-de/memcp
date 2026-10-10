/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"database/sql"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tdsdriver "github.com/denisenkom/go-mssqldb"
)

// Generic clients do not request ODBC/OLE browse tokens. Use the connector's
// public dialer to declare that mode; the external driver remains unmodified.
type tdsGenericClientDialer struct{}
type tdsGenericClientConnection struct{ net.Conn }

func (tdsGenericClientDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	connection, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &tdsGenericClientConnection{connection}, nil
}
func (connection *tdsGenericClientConnection) Write(data []byte) (int, error) {
	if len(data) >= 102 && data[0] == 0x10 && data[6] == 1 {
		data = append([]byte(nil), data...)
		data[33] &= ^byte(2)
		data[34] &= ^byte(0x10)
	}
	return connection.Conn.Write(data)
}
func startTDSGenericCursorClient(t *testing.T, configure func(*tdsServer)) *sql.DB {
	t.Helper()
	_, dsn := startTDSTestServer(t, false, configure)
	connector, err := tdsdriver.NewConnector(dsn)
	if err != nil {
		t.Fatal(err)
	}
	connector.Dialer = tdsGenericClientDialer{}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestTDSCursorPositions(t *testing.T) {
	cursor := &tdsCursor{rows: make([][]Scmer, 6)}
	for _, fixture := range []struct {
		kind, rownum, nrows  int32
		start, end, position int
		status               uint32
	}{{2, 0, 2, 0, 2, 1, 0}, {2, 0, 3, 2, 5, 3, 0}, {4, 0, 5, 0, 5, 1, 2}, {8, 0, 2, 4, 6, 5, 0}, {16, -2, 2, 4, 6, 5, 0}, {32, -3, 2, 1, 3, 2, 0}, {1, 0, 0, 0, 0, 0, 0}, {8, 0, 0, 0, 0, 7, 0}, {2, 0, 2, 6, 6, 7, 0}} {
		start, end, position, status := cursor.rangeForFetch(fixture.kind, fixture.rownum, fixture.nrows)
		if start != fixture.start || end != fixture.end || position != fixture.position || status != fixture.status {
			t.Fatalf("%+v: got %d,%d,%d,%d", fixture, start, end, position, status)
		}
		cursor.start, cursor.fetched = position, end-start
	}
	for _, kind := range []int32{64, 128, 512, 1024} {
		expectTDSPanic(t, func() { cursor.rangeForFetch(kind, 0, 1) })
	}
	expectTDSPanic(t, func() { cursor.rangeForFetch(2, 0, 0) })
	expectTDSPanic(t, func() { cursor.rangeForFetch(1, 0, tdsCursorFetchLimit+1) })
}

func TestTDSCursorSnapshotBoundsAndValues(t *testing.T) {
	var output bytes.Buffer
	sink := &tdsResult{w: &output, snapshotLimit: 4096}
	sink.fields(NewSlice([]Scmer{NewString("v")}))
	for i := 0; i < 30; i++ {
		sink.row(NewSlice([]Scmer{NewString("v"), NewInt(int64(i))}))
	}
	if sink.published || output.Len() != 0 || len(sink.rows) != 30 {
		t.Fatal("snapshot emitted rows before handle publication")
	}
	expectTDSPanic(t, func() { sink.row(NewSlice([]Scmer{NewString("v"), NewString(strings.Repeat("x", 4096))})) })
	sink = &tdsResult{w: &output, snapshotLimit: 4096}
	expectTDSPanic(t, func() { sink.row(NewSlice([]Scmer{NewString("v"), NewSlice([]Scmer{NewInt(1)})})) })
}

func TestTDSCursorCodecPreservesRetainedValue(t *testing.T) {
	var output bytes.Buffer
	original := NewInt(9007199254740993)
	row := []Scmer{original}
	var calls int
	sink := &tdsResult{w: &output, columns: []tdsColumn{{kind: 0x2a, scale: 7, declared: true,
		encode: NewFunc(func(a ...Scmer) Scmer {
			calls++
			if !a[0].IsInt() || a[0].Int() != original.Int() {
				panic("codec received its previously encoded tuple")
			}
			return tdsTemporalPayload(738944, 1234567, 0)
		}),
	}}}
	sink.emit(row)
	first := append([]byte(nil), output.Bytes()...)
	output.Reset()
	sink.emit(row)
	if calls != 2 || !row[0].IsInt() || row[0].Int() != original.Int() || !bytes.Equal(first, output.Bytes()) {
		t.Fatal("repeated fetch mutated its retained frontend scalar", calls, row)
	}
}

func TestTDSCursorReleasedClient(t *testing.T) {
	var executed atomic.Int32
	db := startTDSGenericCursorClient(t, func(server *tdsServer) {
		query := server.query
		server.query = NewFunc(func(a ...Scmer) Scmer {
			if a[1].String() == "SELECT many" {
				executed.Add(1)
			}
			return Apply(query, a...)
		})
		server.describe = NewFunc(func(a ...Scmer) Scmer {
			if a[1].String() != "SELECT many" {
				panic("cursor requires a SELECT")
			}
			return NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("name"), NewString("value"), NewString("sql_type"), NewString("BIGINT")})})
		})
	})
	connection, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	var handle, scroll, concurrency, count int32
	open := func(statement string, options int32) error {
		scroll, concurrency = options, 0x2001
		_, err := connection.ExecContext(context.Background(), "sp_cursoropen", sql.Named("cursor", sql.Out{Dest: &handle}), sql.Named("stmt", statement), sql.Named("scrollopt", sql.Out{Dest: &scroll, In: true}), sql.Named("ccopt", sql.Out{Dest: &concurrency, In: true}), sql.Named("rowcount", sql.Out{Dest: &count}))
		return err
	}
	if err := open("DELETE fixture", 8); err == nil || executed.Load() != 0 || handle != 0 {
		t.Fatal("cursor describe executed DML", err, executed.Load(), handle)
	}
	if err := open("SELECT many", 2); err == nil || executed.Load() != 0 {
		t.Fatal("DYNAMIC cursor silently downgraded", err, executed.Load())
	}
	if err := open("SELECT many", 8); err != nil || handle <= 0 || scroll != 8 || concurrency != 1 || count != 400 {
		t.Fatal("STATIC cursor open", handle, scroll, concurrency, count, err)
	}
	if _, err := connection.ExecContext(context.Background(), "sp_cursoroption", sql.Named("cursor", handle), sql.Named("code", int32(2)), sql.Named("value", "snapshot_name")); err != nil {
		t.Fatal("cursor naming", err)
	}
	if _, err := connection.ExecContext(context.Background(), "sp_cursoroption", sql.Named("cursor", handle), sql.Named("code", int32(2)), sql.Named("value", strings.Repeat("x", 129))); err == nil {
		t.Fatal("unbounded cursor name accepted")
	}
	var option int32
	if _, err := connection.ExecContext(context.Background(), "sp_cursoroption", sql.Named("cursor", handle), sql.Named("code", int32(6)), sql.Named("value", sql.Out{Dest: &option})); err != nil || option != 400 {
		t.Fatal("cursor option ROWCOUNT", option, err)
	}
	if _, err := connection.ExecContext(context.Background(), "sp_cursoroption", sql.Named("cursor", handle), sql.Named("code", int32(1)), sql.Named("value", int32(0))); err == nil {
		t.Fatal("unsupported text-pointer mode accepted")
	}
	tx, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		kind, rownum, nrows int32
		first, length       int
	}{{2, 0, 3, 0, 3}, {2, 0, 3, 3, 3}, {16, -2, 3, 398, 2}, {1, 0, 1, 0, 1}} {
		rows, err := connection.QueryContext(context.Background(), "sp_cursorfetch", sql.Named("cursor", handle), sql.Named("fetchtype", fixture.kind), sql.Named("rownum", fixture.rownum), sql.Named("nrows", fixture.nrows))
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil || len(columns) != 2 || columns[1] != "rowstat" {
			t.Fatal(columns, err)
		}
		n := 0
		for rows.Next() {
			var value, status int64
			if err := rows.Scan(&value, &status); err != nil || value != int64(fixture.first+n) || status != 1 {
				t.Fatal(value, status, err)
			}
			n++
		}
		if err := rows.Err(); err != nil || n != fixture.length {
			t.Fatal(n, err)
		}
		rows.Close()
	}
	var position, total int32
	if _, err := connection.ExecContext(context.Background(), "sp_cursorfetch", sql.Named("cursor", handle), sql.Named("fetchtype", int32(256)), sql.Named("rownum", sql.Out{Dest: &position}), sql.Named("nrows", sql.Out{Dest: &total})); err != nil || position != 1 || total != 400 {
		t.Fatal(position, total, err)
	}
	if _, err := connection.ExecContext(context.Background(), "sp_cursorclose", sql.Named("cursor", handle)); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), "sp_cursorfetch", sql.Named("cursor", handle)); err == nil {
		t.Fatal("closed cursor remained usable")
	}
	if executed.Load() != 1 {
		t.Fatal("cursor fetch reexecuted query", executed.Load())
	}
}

func TestTDSCursorAttentionDoesNotPublishSnapshot(t *testing.T) {
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		server.describe = NewFunc(func(a ...Scmer) Scmer { return NewSlice(nil) })
	})
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var handle, count int32
	scroll, concurrency := int32(8), int32(1)
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(100*time.Millisecond, cancel)
	_, err = conn.ExecContext(ctx, "sp_cursoropen", sql.Named("cursor", sql.Out{Dest: &handle}), sql.Named("stmt", "SELECT wait"), sql.Named("scrollopt", sql.Out{Dest: &scroll, In: true}), sql.Named("ccopt", sql.Out{Dest: &concurrency, In: true}), sql.Named("rowcount", sql.Out{Dest: &count}))
	timer.Stop()
	cancel()
	if err == nil || handle != 0 {
		t.Fatal("cancelled cursor was published", handle, err)
	}
	if _, err := conn.ExecContext(context.Background(), "sp_cursorclose", sql.Named("cursor", int32(1))); err == nil {
		t.Fatal("cancelled cursor handle leaked")
	}
	var value int64
	if err := conn.QueryRowContext(context.Background(), "SELECT 1").Scan(&value); err != nil || value != 1 {
		t.Fatal("cursor cancellation desynchronized connection", value, err)
	}
}

func TestTDSCursorCachedMetadataAndCloseFlags(t *testing.T) {
	for _, options := range []uint16{2, 4} {
		recorded := &tdsRecordingConnection{}
		response := &tdsResponse{w: recorded, kind: 4, id: 1, packetSize: 4096}
		cursor := &tdsCursor{columns: []tdsColumn{{name: "v", kind: 0x26, size: 8, declared: true}}, rows: [][]Scmer{{NewInt(1)}}, bytes: 128}
		connection := &tdsConnection{cursors: map[int32]*tdsCursor{1: cursor}, cursorBytes: 128, browseMetadata: true}
		request := &tdsRequest{ctx: context.Background(), cancel: func() {}}
		rpc := tdsRPC{procedure: 7, options: options, arguments: []tdsRPCParameter{{"", 0, NewInt(1)}, {"", 0, NewInt(2)}, {"", 0, NewNil()}, {"", 0, NewInt(1)}}}
		if err := connection.cursorRPC(rpc, request, response); err != nil {
			t.Fatal(err)
		}
		message, err := readTDSMessage(bytes.NewReader(recorded.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		if options == 2 && !bytes.Equal(message.data[:3], []byte{0x81, 255, 255}) {
			t.Fatalf("metadata not suppressed %x", message.data)
		}
		if options == 4 && !bytes.Equal(message.data[:3], []byte{0x81, 2, 0}) {
			t.Fatalf("reuse omitted required metadata %x", message.data)
		}
		if options == 4 && !bytes.Contains(message.data, []byte{0xa5, 6, 0, 1, 0, 4, 2, 0, 20}) {
			t.Fatalf("missing hidden cursor column browse metadata %x", message.data)
		}
		if cursor.start != 1 || cursor.fetched != 1 {
			t.Fatal("cached fetch position", cursor.start, cursor.fetched)
		}
		request = &tdsRequest{ctx: context.Background(), cancel: func() {}}
		rpc = tdsRPC{procedure: 9, options: 2, arguments: []tdsRPCParameter{{"", 0, NewInt(1)}}}
		if err := connection.cursorRPC(rpc, request, response); err != nil || connection.cursorBytes != 0 || len(connection.cursors) != 0 {
			t.Fatal("cached close leaked snapshot", err)
		}
	}
}

func TestTDSParameterizedStaticCursorReleasedClient(t *testing.T) {
	db := startTDSGenericCursorClient(t, func(server *tdsServer) {
		server.describe = NewFunc(func(a ...Scmer) Scmer {
			if a[1].String() != "SELECT prepared" || a[2].String() != "@v bigint" || a[3].Func()(NewString("tsql_param_type:v")).Slice()[0].String() != "BIGINT" {
				panic("parameterized cursor describe lost declarations")
			}
			return NewSlice(nil)
		})
	})
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var handle, count int32
	scroll, concurrency := int32(0x1008), int32(1)
	_, err = conn.ExecContext(context.Background(), "sp_cursoropen", sql.Named("cursor", sql.Out{Dest: &handle}), sql.Named("stmt", "SELECT prepared"), sql.Named("scrollopt", sql.Out{Dest: &scroll, In: true}), sql.Named("ccopt", sql.Out{Dest: &concurrency, In: true}), sql.Named("rowcount", sql.Out{Dest: &count}), sql.Named("paramdef", "@v bigint"), sql.Named("v", int64(9007199254740993)))
	if err != nil || handle <= 0 || scroll != 8 || count != 1 {
		t.Fatal(handle, scroll, count, err)
	}
	var value, status int64
	if err := conn.QueryRowContext(context.Background(), "sp_cursorfetch", sql.Named("cursor", handle)).Scan(&value, &status); err != nil || value != 9007199254740993 || status != 1 {
		t.Fatal(value, status, err)
	}
	if _, err := conn.ExecContext(context.Background(), "sp_cursorclose", sql.Named("cursor", handle)); err != nil {
		t.Fatal(err)
	}
}
