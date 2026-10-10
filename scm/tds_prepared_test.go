/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTDSPreparedClient(t *testing.T) {
	db, _ := startTDSTestServer(t, false)
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var handle int32
	_, err = conn.ExecContext(context.Background(), "sp_prepare", sql.Named("handle", sql.Out{Dest: &handle}), sql.Named("params", "@v bigint"), sql.Named("stmt", "SELECT prepared"), sql.Named("options", 0))
	if err != nil || handle <= 0 {
		t.Fatalf("prepare: handle=%d err=%v", handle, err)
	}
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []int64{9007199254740993, -42} {
		var actual int64
		err = conn.QueryRowContext(context.Background(), "sp_execute", sql.Named("handle", handle), sql.Named("v", value)).Scan(&actual)
		if err != nil || actual != value {
			t.Fatalf("execute %d: %d %v", value, actual, err)
		}
	}
	if _, err = conn.ExecContext(context.Background(), "sp_execute", sql.Named("handle", handle)); err == nil {
		t.Fatal("missing parameter accepted")
	}
	var prepexecHandle int32
	rows, err := conn.QueryContext(context.Background(), "sp_prepexec", sql.Named("handle", sql.Out{Dest: &prepexecHandle}), sql.Named("params", "@v bigint"), sql.Named("stmt", "SELECT prepared"), sql.Named("v", int64(73)))
	if err != nil {
		t.Fatal(err)
	}
	if !rows.Next() {
		t.Fatal("missing prepexec result")
	}
	var value int64
	if err = rows.Scan(&value); err != nil || value != 73 {
		t.Fatal(value, err)
	}
	for rows.Next() {
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	if prepexecHandle <= handle {
		t.Fatal("missing prepexec output handle", prepexecHandle)
	}
	for _, id := range []int32{handle, prepexecHandle} {
		if _, err = conn.ExecContext(context.Background(), "sp_unprepare", sql.Named("handle", id)); err != nil {
			t.Fatal(err)
		}
		if _, err = conn.ExecContext(context.Background(), "sp_execute", sql.Named("handle", id), sql.Named("v", 1)); err == nil {
			t.Fatal("released handle still executable")
		}
	}
	if _, err = conn.ExecContext(context.Background(), "sp_prepare", sql.Named("handle", sql.Out{Dest: &handle}), sql.Named("params", ""), sql.Named("stmt", "SELECT 1"), sql.Named("options", 1)); err == nil {
		t.Fatal("unavailable describe metadata accepted")
	}
	if err = conn.QueryRowContext(context.Background(), "SELECT 1").Scan(&value); err != nil || value != 1 {
		t.Fatal("RPC error broke connection", value, err)
	}
}

func TestTDSTypedResultsAndBinaryClient(t *testing.T) {
	db, _ := startTDSTestServer(t, false)
	rows, err := db.Query("SELECT typed empty")
	if err != nil {
		t.Fatal(err)
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"INT", "NVARCHAR", "BINARY", "DATE", "TIME", "DATETIME2"}
	for i, col := range types {
		if col.DatabaseTypeName() != expected[i] {
			t.Fatalf("column %d type %s, want %s", i, col.DatabaseTypeName(), expected[i])
		}
	}
	if rows.Next() {
		t.Fatal("empty result emitted a row")
	}
	rows.Close()
	var i int64
	var n any
	var binaryValue []byte
	var date, clock, datetime time.Time
	if err = db.QueryRow("SELECT temporal").Scan(&i, &n, &binaryValue, &date, &clock, &datetime); err != nil {
		t.Fatal(err)
	}
	if i != 1 || n != nil || string(binaryValue) != "\x00\xffabc\x00xy" || date.Format("2006-01-02") != "2024-02-29" || clock.Format("15:04:05") != "13:45:30" || datetime.Format("2006-01-02 15:04:05") != "2024-02-29 13:45:30" {
		t.Fatalf("corrupt typed values %d %v %x %v %v %v", i, n, binaryValue, date, clock, datetime)
	}
	var text string
	if err = db.QueryRow("SELECT character").Scan(&text); err != nil || text != "Grüße €" {
		t.Fatal(text, err)
	}
	if err = db.QueryRow("SELECT invalid typed").Scan(&text); err == nil {
		t.Fatal("invalid typed result accepted")
	}
	if err = db.QueryRow("SELECT 1").Scan(&i); err != nil || i != 1 {
		t.Fatal("conversion error desynchronized connection", i, err)
	}
	raw := []byte{0, 255, 128, 1, 0}
	if err = db.QueryRow("SELECT binary", sql.Named("v", raw)).Scan(&binaryValue); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, binaryValue) {
		t.Fatalf("binary parameter altered %x", binaryValue)
	}
}

func TestTDSPreparedRawLifecycle(t *testing.T) {
	server := &tdsServer{}
	tdsInstallTestFrontend(server)
	connection := &tdsConnection{server: server}
	out := tdsRPCParameter{"@handle", 1, NewNil()}
	input := func(value string) tdsRPCParameter { return tdsRPCParameter{"", 0, NewString(value)} }
	prepare := func(sql string) int32 {
		_, _, handle, _, _ := connection.prepareRPC(tdsRPC{procedure: 11, arguments: []tdsRPCParameter{out, input("@v int"), input(sql)}}, nil)
		return handle
	}
	handle := prepare("SELECT @v")
	query, parameters, _, _, _ := connection.prepareRPC(tdsRPC{procedure: 12, arguments: []tdsRPCParameter{{"", 0, NewInt(int64(handle))}, {"", 0, NewInt(27)}}}, nil)
	if query != "SELECT @v" || parameters["v"].Int() != 27 {
		t.Fatal(query, parameters)
	}
	for len(connection.prepared) < tdsPreparedLimit {
		prepare("SELECT @v")
	}
	expectTDSPanic(t, func() { prepare("SELECT @v") })
	connection.removePrepared(handle)
	if connection.preparedBytes < 0 || len(connection.prepared) != 127 {
		t.Fatal("invalid handle accounting")
	}
	expectTDSPanic(t, func() {
		connection.prepareRPC(tdsRPC{procedure: 12, arguments: []tdsRPCParameter{{"", 0, NewInt(int64(handle))}}}, nil)
	})
	connection = &tdsConnection{server: server}
	expectTDSPanic(t, func() { prepare(strings.Repeat("x", tdsPreparedBytesLimit+1)) })
	if len(connection.prepared) != 0 {
		t.Fatal("oversized prepare leaked handle")
	}
	for _, invalid := range []string{"@v int(4)", "@v nvarchar(0)", "@v nvarchar(max),@V int", "@v nchar(max)", "@v time(8)", "@v decimal(10,20)", "@v int output"} {
		if _, err := tdsDeclarations("fixture", invalid, server.declarations); err == nil {
			t.Fatal("invalid declarations accepted", invalid)
		}
	}
	var body bytes.Buffer
	tdsU32(&body, 22)
	tdsU32(&body, 18)
	tdsU16(&body, 2)
	tdsU64(&body, 0)
	tdsU32(&body, 1)
	tdsU16(&body, 0xffff)
	tdsU16(&body, 15)
	tdsU16(&body, 0)
	tdsBText(&body, "")
	body.WriteByte(0)
	body.Write([]byte{0x26, 4, 4})
	tdsU32(&body, 42)
	rpc, err := decodeTDSRPC(body.Bytes())
	if err != nil || rpc.procedure != 15 || rpc.arguments[0].value.Int() != 42 {
		t.Fatal("numeric RPC ID decode", rpc, err)
	}
	for _, id := range []uint16{11, 12, 13} {
		payload := append([]byte(nil), body.Bytes()...)
		payload[24] = byte(id)
		payload[25] = 0
		rpc, err = decodeTDSRPC(payload)
		if err != nil || rpc.procedure != id {
			t.Fatal(id, rpc, err)
		}
	}
	var tokens bytes.Buffer
	tdsReturnHandle(&tokens, "@handle", 42)
	d := tdsDecoder{data: tokens.Bytes()}
	if d.u8() != 0xac || d.u16() != 0 || d.text(int(d.u8())) != "@handle" || d.u8() != 1 || d.u32() != 0 || d.u16() != 0 || d.parameter().Int() != 42 || d.err != nil || len(d.data) != 0 {
		t.Fatal("invalid output handle token")
	}
}
func expectTDSPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected TDS rejection")
		}
	}()
	fn()
}

func TestTDSTemporalWireRoundtrip(t *testing.T) {
	for _, col := range []tdsColumn{{kind: 0x28}, {kind: 0x29, scale: 7}, {kind: 0x2a, scale: 0}, {kind: 0x2a, scale: 7}, {kind: 0x6f, size: 4}, {kind: 0x6f, size: 8}} {
		day, clock := int64(738944), int64(86340)
		switch col.kind {
		case 0x28:
			clock = 0
		case 0x29:
			day = 0
			clock *= int64(tdsTimeFactor(col.scale))
		case 0x2a:
			clock *= int64(tdsTimeFactor(col.scale))
		case 0x6f:
			day -= 693595
			if col.size == 4 {
				clock /= 60
			} else {
				clock *= 300
			}
		}
		payload := tdsTemporalPayload(day, clock, 0)
		metadata := []byte{col.kind}
		if col.kind == 0x29 || col.kind == 0x2a {
			metadata = append(metadata, col.scale)
		}
		if col.kind == 0x6f {
			metadata = append(metadata, byte(col.size))
		}
		decoder := tdsDecoder{data: append(metadata, tdsEncodeTemporal(col, payload)...)}
		actual := decoder.parameter()
		if decoder.err != nil || !Equal(actual, payload) {
			t.Fatal(col, decoder.err)
		}
	}
}

func TestTDSPrepareDescriptionIsPure(t *testing.T) {
	var queries, descriptions atomic.Int32
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		server.query = NewFunc(func(a ...Scmer) Scmer { queries.Add(1); panic("prepare executed SQL") })
		server.describe = NewFunc(func(a ...Scmer) Scmer {
			descriptions.Add(1)
			if a[1].String() != "SELECT @v" || a[2].String() != "@v int" {
				panic("wrong describe input")
			}
			return NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("name"), NewString("v"), NewString("sql_type"), NewString("INT")})})
		})
	})
	var handle int32
	_, err := db.Exec("sp_prepare", sql.Named("handle", sql.Out{Dest: &handle}), sql.Named("params", "@v int"), sql.Named("stmt", "SELECT @v"), sql.Named("options", 1))
	if err != nil || handle <= 0 || queries.Load() != 0 || descriptions.Load() != 1 {
		t.Fatal(handle, err, queries.Load(), descriptions.Load())
	}
}

type tdsRecordingConnection struct{ bytes.Buffer }

func (*tdsRecordingConnection) Read([]byte) (int, error)         { return 0, io.EOF }
func (*tdsRecordingConnection) Close() error                     { return nil }
func (*tdsRecordingConnection) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (*tdsRecordingConnection) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (*tdsRecordingConnection) SetDeadline(time.Time) error      { return nil }
func (*tdsRecordingConnection) SetReadDeadline(time.Time) error  { return nil }
func (*tdsRecordingConnection) SetWriteDeadline(time.Time) error { return nil }
func tdsRawRPC(procedure uint16, args ...tdsRPCParameter) []byte {
	var body bytes.Buffer
	tdsU32(&body, 22)
	tdsU32(&body, 18)
	tdsU16(&body, 2)
	tdsU64(&body, 0)
	tdsU32(&body, 1)
	tdsU16(&body, 0xffff)
	tdsU16(&body, procedure)
	tdsU16(&body, 0)
	for _, arg := range args {
		tdsBText(&body, arg.name)
		body.WriteByte(arg.flags)
		if arg.value.IsString() {
			data := tdsUTF16(arg.value.String())
			body.WriteByte(0xe7)
			tdsU16(&body, uint16(len(data)))
			body.Write([]byte{9, 4, 0xd0, 0, 0x34})
			tdsU16(&body, uint16(len(data)))
			body.Write(data)
		} else {
			body.Write([]byte{0x26, 4})
			if arg.value.IsNil() {
				body.WriteByte(0)
			} else {
				body.WriteByte(4)
				tdsU32(&body, uint32(arg.value.Int()))
			}
		}
	}
	return body.Bytes()
}
func TestTDSPreparedFailureResetAndTransaction(t *testing.T) {
	recorded := &tdsRecordingConnection{}
	server := &tdsServer{closeSession: NewFunc(func(a ...Scmer) Scmer { return a[0].Func()(NewString("transaction"), NewNil()) })}
	server.query = NewFunc(func(a ...Scmer) Scmer {
		if a[1].String() == "FAIL" {
			panic("fixture failure")
		}
		return NewInt(0)
	})
	connection := &tdsConnection{server: server, conn: recorded, database: "fixture", initialDatabase: "fixture", ss: RegisterSession("alice", "fixture", "fixture")}
	defer UnregisterSession(connection.ss.ID)
	connection.newSession("alice")
	output := tdsRPCParameter{"@handle", 1, NewNil()}
	text := func(v string) tdsRPCParameter { return tdsRPCParameter{"", 0, NewString(v)} }
	if err := connection.execute(tdsMessage{kind: 3, data: tdsRawRPC(13, output, text(""), text("FAIL"))}); err != nil {
		t.Fatal(err)
	}
	if len(connection.prepared) != 0 || connection.preparedBytes != 0 {
		t.Fatal("failed prepexec retained handle")
	}
	recorded.Reset()
	if err := connection.execute(tdsMessage{kind: 3, data: tdsRawRPC(11, output, text(""), text("SELECT 1"))}); err != nil {
		t.Fatal(err)
	}
	handle := connection.nextPrepared
	if len(connection.prepared) != 1 {
		t.Fatal("prepare did not publish handle")
	}
	connection.session.Func()(NewString("transaction"), NewBool(true))
	if err := connection.execute(tdsMessage{kind: 3, data: tdsRawRPC(12, tdsRPCParameter{"", 0, NewInt(int64(handle))})}); err != nil {
		t.Fatal(err)
	}
	connection.session.Func()(NewString("transaction"), NewNil())
	connection.transaction = 0
	if _, exists := connection.prepared[handle]; !exists {
		t.Fatal("transaction completion discarded prepare")
	}
	recorded.Reset()
	if err := connection.execute(tdsMessage{kind: 3, status: 8, data: tdsRawRPC(15, tdsRPCParameter{"", 0, NewInt(int64(handle))})}); err != nil {
		t.Fatal(err)
	}
	if len(connection.prepared) != 0 || connection.preparedBytes != 0 {
		t.Fatal("reset retained handles")
	}
	message, err := readTDSMessage(bytes.NewReader(recorded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(message.data, []byte{0xaa}) {
		t.Fatal("old handle accepted after reset")
	}
}

func TestTDSCharacterCodePages(t *testing.T) {
	if tdsCP1252 == nil {
		t.Fatal("CP1252 encoding unavailable")
	}
	for _, collation := range [][]byte{{7, 4, 0, 0, 0}, {9, 4, 0xd0, 0, 52}} {
		text, err := tdsDecodeCharacter([]byte{'G', 'r', 0xfc, 0xdf, 'e', ' ', 0x80}, collation)
		if err != nil || text != "Grüße €" {
			t.Fatal(text, err)
		}
	}
	utf8collation := []byte{9, 4, 0, 4, 0}
	text, err := tdsDecodeCharacter([]byte("Grüße 🐘"), utf8collation)
	if err != nil || text != "Grüße 🐘" {
		t.Fatal(text, err)
	}
	for _, fixture := range []struct{ data, collation []byte }{{[]byte{255}, utf8collation}, {[]byte{0xfc}, []byte{0, 0, 0, 0, 0}}, {[]byte{0x81}, []byte{7, 4, 0, 0, 0}}} {
		if _, err = tdsDecodeCharacter(fixture.data, fixture.collation); err == nil {
			t.Fatal("invalid or unknown encoding accepted")
		}
	}
	encoded := tdsStringBytes(tdsColumn{kind: 0xa7}, NewString("Grüße €"))
	if !bytes.Equal(encoded, []byte{'G', 'r', 0xfc, 0xdf, 'e', ' ', 0x80}) {
		t.Fatal(encoded)
	}
	expectTDSPanic(t, func() { tdsStringBytes(tdsColumn{kind: 0xa7}, NewString("🐘")) })
	expectTDSPanic(t, func() { tdsStringBytes(tdsColumn{kind: 0xe7, declared: true}, NewInt(1)) })
	expectTDSPanic(t, func() { tdsStringBytes(tdsColumn{kind: 0xad, declared: true}, NewInt(1)) })
}

func TestTDSDescriptionSessionDoesNotLeakFlags(t *testing.T) {
	base := NewSession()
	base.Func()(NewString("schema"), NewString("fixture"))
	server := &tdsServer{}
	tdsInstallTestFrontend(server)
	connection := &tdsConnection{server: server, session: base}
	declarations, err := tdsDeclarations("fixture", "@v int", server.declarations)
	if err != nil {
		t.Fatal(err)
	}
	described := connection.descriptionSession(declarations)
	described.Func()(NewString("tsql_describe_only"), NewBool(true))
	if !base.Func()(NewString("tsql_describe_only")).IsNil() {
		t.Fatal("describe flag leaked into live session")
	}
	if !described.Func()(NewString("tsql_bound:v")).Bool() || !described.Func()(NewString("tsql_param:v")).IsNil() || described.Func()(NewString("tsql_param_type:v")).Slice()[0].String() != "INT" {
		t.Fatal("incorrect description parameters")
	}
	if described.Func()(NewString("schema")).String() != "fixture" {
		t.Fatal("describe lost ordinary session state")
	}
	clone := NewSession()
	for _, key := range described.Func()().Slice() {
		clone.Func()(key, described.Func()(key))
	}
	if clone.Func()(NewString("tsql_param_type:v")).Slice()[0].String() != "INT" || !clone.Func()(NewString("tsql_bound:v")).Bool() {
		t.Fatal("session enumeration lost parameter declaration")
	}
	if !base.Func()(NewString("tsql_param_type:v")).IsNil() || !base.Func()(NewString("tsql_bound:v")).IsNil() {
		t.Fatal("enumeration persisted invocation parameters")
	}
}

func TestTDSPreparedDeclarationRecipeIsDatabaseBound(t *testing.T) {
	server := &tdsServer{}
	tdsInstallTestFrontend(server)
	var calls int
	server.declarations = NewFunc(func(a ...Scmer) Scmer {
		calls++
		if len(a) != 2 || a[0].String() != "original" || a[1].String() != "@v local_alias" {
			panic("declaration callback lost its database context")
		}
		return NewSlice([]Scmer{NewSlice([]Scmer{NewString("v"), NewSlice([]Scmer{NewString("resolved-recipe")})})})
	})
	server.convertParameter = NewFunc(func(a ...Scmer) Scmer {
		if a[2].Slice()[0].String() != "resolved-recipe" {
			panic("execution did not retain the bound declaration recipe")
		}
		return NewSlice([]Scmer{a[4], a[0]})
	})
	connection := &tdsConnection{server: server, database: "original", prepared: make(map[int32]tdsPreparedStatement)}
	_, _, handle, _, _ := connection.prepareRPC(tdsRPC{procedure: 11, arguments: []tdsRPCParameter{
		{"", 1, NewNil()}, {"", 0, NewString("@v local_alias")}, {"", 0, NewString("SELECT prepared")},
	}}, nil)
	connection.database = "later"
	for _, value := range []int64{1, 2} {
		_, params, _, _, _ := connection.prepareRPC(tdsRPC{procedure: 12, arguments: []tdsRPCParameter{
			{"", 0, NewInt(int64(handle))}, {"", 0, NewInt(value)},
		}}, nil)
		if params["v"].Int() != value {
			t.Fatal("bound parameter changed", params)
		}
	}
	if calls != 1 {
		t.Fatal("prepared declaration was resolved per execution", calls)
	}
}

func TestTDSTemporalRequiresBoundWirePayload(t *testing.T) {
	col := tdsColumn{kind: 0x2a, scale: 7}
	for _, value := range []Scmer{NewString("2024-02-29 13:45:30"), NewString("2024-02-29 13:45:30.123"), NewInt(1709214330), NewDate(1709214330)} {
		expectTDSPanic(t, func() { tdsEncodeTemporal(col, value) })
	}
}

func TestTDSTypedDiscoveryBufferIsBounded(t *testing.T) {
	var output bytes.Buffer
	sink := &tdsResult{w: &output}
	sink.fields(NewSlice([]Scmer{NewString("value")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("NVARCHAR"), NewString("size"), NewInt(-1)})}))
	row := NewSlice([]Scmer{NewString("value"), NewString(strings.Repeat("x", 600000))})
	sink.row(row)
	if sink.published {
		t.Fatal("premature discovery publication")
	}
	sink.row(row)
	if !sink.published || len(sink.rows) != 0 {
		t.Fatal("declared values escaped discovery byte limit")
	}
}

func TestTDSPreparedAttentionKeepsOnlyPublishedHandles(t *testing.T) {
	db, _ := startTDSTestServer(t, false)
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var unpublished int32
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(100*time.Millisecond, cancel)
	_, err = conn.ExecContext(ctx, "sp_prepexec", sql.Named("handle", sql.Out{Dest: &unpublished}), sql.Named("params", ""), sql.Named("stmt", "SELECT wait"))
	timer.Stop()
	cancel()
	if err == nil || unpublished != 0 {
		t.Fatal("cancelled prepexec published handle", unpublished, err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	_, err = conn.ExecContext(ctx, "sp_execute", sql.Named("handle", 1))
	cancel()
	if err == nil || !strings.Contains(err.Error(), "unknown TDS prepared handle") {
		t.Fatal("cancelled prepexec retained handle", err)
	}
	var published int32
	_, err = conn.ExecContext(context.Background(), "sp_prepare", sql.Named("handle", sql.Out{Dest: &published}), sql.Named("params", ""), sql.Named("stmt", "SELECT wait"), sql.Named("options", 0))
	if err != nil || published <= 0 {
		t.Fatal(published, err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	timer = time.AfterFunc(100*time.Millisecond, cancel)
	_, err = conn.ExecContext(ctx, "sp_execute", sql.Named("handle", published))
	timer.Stop()
	cancel()
	if err == nil {
		t.Fatal("cancelled execute succeeded")
	}
	if _, err = conn.ExecContext(context.Background(), "sp_unprepare", sql.Named("handle", published)); err != nil {
		t.Fatal("cancelled execute discarded published handle", err)
	}
}

func TestTDSLegacyLargeParameterTypes(t *testing.T) {
	for _, fixture := range []struct {
		kind  byte
		data  []byte
		value string
	}{{0x63, tdsUTF16("Grüße 🐘"), "Grüße 🐘"}, {0x23, []byte{0xfc, 0xdf, 0x80}, "üß€"}, {0x22, []byte{0, 255, 1}, "\x00\xff\x01"}} {
		var body bytes.Buffer
		body.WriteByte(fixture.kind)
		tdsU32(&body, uint32(len(fixture.data)))
		if fixture.kind != 0x22 {
			body.Write([]byte{9, 4, 0xd0, 0, 52})
		}
		tdsU32(&body, uint32(len(fixture.data)))
		body.Write(fixture.data)
		d := tdsDecoder{data: body.Bytes()}
		actual := d.parameter()
		if d.err != nil || actual.String() != fixture.value || len(d.data) != 0 {
			t.Fatal(fixture.kind, actual, d.err)
		}
	}
	for _, fixture := range [][]byte{{0x63, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255}, {0x22, 0, 0, 0, 0, 255, 255, 255, 255}} {
		d := tdsDecoder{data: fixture}
		value := d.parameter()
		if d.err != nil || !value.IsNil() {
			t.Fatal("legacy NULL", value, d.err)
		}
	}
	for _, fixture := range [][]byte{{0x63, 2, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0xd8}, {0x22, 0, 0, 0, 4, 0, 0, 0, 4}} {
		d := tdsDecoder{data: fixture}
		d.parameter()
		if d.err == nil {
			t.Fatal("malformed or oversized legacy TDS value accepted")
		}
	}
}
func TestTDSDeclaredNumbersAcceptExactCompressedRepresentations(t *testing.T) {
	for _, value := range []Scmer{NewInt(1), NewFloat(1), NewFloat(-9223372036854775808.0)} {
		if !tdsResultInteger(value).IsInt() {
			t.Fatal(value)
		}
	}
	for _, value := range []Scmer{NewFloat(1.5), NewFloat(9223372036854775808.0), NewString("1")} {
		expectTDSPanic(t, func() { tdsResultInteger(value) })
	}
	for _, value := range []Scmer{NewInt(0), NewInt(1), NewFloat(0), NewFloat(1), NewBool(true)} {
		if !tdsResultBit(value).IsBool() {
			t.Fatal(value)
		}
	}
	for _, value := range []Scmer{NewInt(-1), NewFloat(0.5), NewString("1")} {
		expectTDSPanic(t, func() { tdsResultBit(value) })
	}
	var output bytes.Buffer
	sink := &tdsResult{w: &output}
	sink.fields(NewSlice([]Scmer{NewString("i"), NewString("b")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("INT")}), tdsTestDescription([]Scmer{NewString("sql_type"), NewString("BIT")})}))
	sink.row(NewSlice([]Scmer{NewString("i"), NewFloat(1), NewString("b"), NewInt(1)}))
	sink.publish()
}

func TestTDSMissingWireDescriptorFailsExplicitly(t *testing.T) {
	var output bytes.Buffer
	sink := &tdsResult{w: &output}
	expectTDSPanic(t, func() {
		sink.fields(NewSlice([]Scmer{NewString("v")}), NewSlice([]Scmer{NewSlice([]Scmer{NewString("sql_type"), NewString("ANY")})}))
	})
	if output.Len() != 0 {
		t.Fatal("unbound type published invented metadata")
	}
}

func TestTDSPrepareNonqueryMetadataIsPureAndRequestLocal(t *testing.T) {
	var queries, descriptions atomic.Int32
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		server.query = NewFunc(func(a ...Scmer) Scmer {
			queries.Add(1)
			if a[1].String() != "INSERT fixture VALUES(@v)" || !a[4].Func()(NewString("tsql_prepare_describe")).IsNil() {
				panic("prepare mode leaked into execution")
			}
			if a[4].Func()(NewString("tsql_param:v")).Int() != 7 {
				panic("nonquery execution lost its binding")
			}
			return NewInt(1)
		})
		server.describe = NewFunc(func(a ...Scmer) Scmer {
			descriptions.Add(1)
			if a[1].String() != "INSERT fixture VALUES(@v)" || !a[3].Func()(NewString("tsql_prepare_describe")).Bool() {
				panic("nonquery prepare did not select its pure description mode")
			}
			return NewSlice(nil)
		})
	})
	connection, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	var handle int32
	_, err = connection.ExecContext(context.Background(), "sp_prepare", sql.Named("handle", sql.Out{Dest: &handle}), sql.Named("params", "@v int"), sql.Named("stmt", "INSERT fixture VALUES(@v)"), sql.Named("options", 1))
	if err != nil || handle <= 0 || queries.Load() != 0 || descriptions.Load() != 1 {
		t.Fatal(handle, err, queries.Load(), descriptions.Load())
	}
	result, err := connection.ExecContext(context.Background(), "sp_execute", sql.Named("handle", handle), sql.Named("v", 7))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 || queries.Load() != 1 {
		t.Fatal(rows, err, queries.Load())
	}
}
