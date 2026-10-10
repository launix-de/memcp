/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	tdsdriver "github.com/denisenkom/go-mssqldb"
)

func startTDSTestServer(t *testing.T, encrypted bool, configure ...func(*tdsServer)) (*sql.DB, string) {
	t.Helper()
	s := &tdsServer{database: "fixture"}
	tdsInstallTestFrontend(s)
	s.auth = NewFunc(func(a ...Scmer) Scmer {
		return NewBool(a[0].String() == "alice" && a[1].String() == "secret" && a[2].String() == "fixture")
	})
	s.closeSession = NewFunc(func(a ...Scmer) Scmer { return a[0].Func()(NewString("transaction"), NewNil()) })
	s.query = NewFunc(func(a ...Scmer) Scmer {
		query, session := strings.TrimSpace(a[1].String()), a[4].Func()
		fields, row := a[3].Func(), a[2].Func()
		emit := func(name string, value Scmer) {
			fields(NewSlice([]Scmer{NewString(name)}))
			row(NewSlice([]Scmer{NewString(name), value}))
		}
		switch query {
		case "SELECT 1", "select 1":
			emit("value", NewInt(1))
		case "SELECT empty":
			fields(NewSlice([]Scmer{NewString("value")}))
		case "SELECT types":
			fields(NewSlice([]Scmer{NewString("i"), NewString("f"), NewString("b"), NewString("s"), NewString("n")}))
			row(NewSlice([]Scmer{NewString("i"), NewInt(9007199254740993), NewString("f"), NewFloat(2.5), NewString("b"), NewBool(true), NewString("s"), NewString("Grüße 🐘"), NewString("n"), NewNil()}))
		case "SELECT params":
			fields(NewSlice([]Scmer{NewString("i"), NewString("s"), NewString("n"), NewString("b"), NewString("f")}))
			values := []Scmer{}
			for _, name := range []string{"i", "s", "n", "b", "f"} {
				if !session(NewString("tsql_bound:" + name)).Bool() {
					panic("parameter was not bound")
				}
				values = append(values, NewString(name), session(NewString("tsql_param:"+name)))
			}
			row(NewSlice(values))
		case "SELECT prepared":
			emit("value", session(NewString("tsql_param:v")))
		case "SELECT character":
			fields(NewSlice([]Scmer{NewString("value")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("VARCHAR"), NewString("size"), NewInt(32)})}))
			row(NewSlice([]Scmer{NewString("value"), NewString("Grüße €")}))
		case "SELECT invalid typed":
			fields(NewSlice([]Scmer{NewString("value")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("TIME")})}))
			row(NewSlice([]Scmer{NewString("value"), NewString("not a time")}))
		case "SELECT binary":
			fields(NewSlice([]Scmer{NewString("value")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("VARBINARY"), NewString("size"), NewInt(8)})}))
			row(NewSlice([]Scmer{NewString("value"), session(NewString("tsql_param:v"))}))
		case "SELECT typed empty", "SELECT temporal":
			names := []string{"i", "n", "rv", "date", "time", "dt"}
			kinds := []string{"INT", "NVARCHAR", "ROWVERSION", "DATE", "TIME", "DATETIME2"}
			titles, descriptors := []Scmer{}, []Scmer{}
			for i, name := range names {
				titles = append(titles, NewString(name))
				descriptors = append(descriptors, tdsTestDescription([]Scmer{NewString("name"), NewString(name), NewString("sql_type"), NewString(kinds[i]), NewString("size"), NewInt(8), NewString("scale"), NewInt(0)}))
			}
			fields(NewSlice(titles), NewSlice(descriptors))
			if query == "SELECT temporal" {
				row(NewSlice([]Scmer{NewString("i"), NewInt(1), NewString("n"), NewNil(), NewString("rv"), NewString("\x00\xffabc\x00xy"), NewString("date"), tdsTemporalPayload(738944, 0, 0), NewString("time"), tdsTemporalPayload(0, 49530, 0), NewString("dt"), tdsTemporalPayload(738944, 49530, 0)}))
			}
		case "SELECT many":
			fields(NewSlice([]Scmer{NewString("value")}))
			for i := 0; i < 400; i++ {
				row(NewSlice([]Scmer{NewString("value"), NewInt(int64(i))}))
			}
		case "SELECT long":
			emit("value", NewString(strings.Repeat("ä🐘", 30000)))
		case "BEGIN TRANSACTION":
			session(NewString("transaction"), NewBool(true))
		case "COMMIT", "ROLLBACK":
			session(NewString("transaction"), NewNil())
		case "SELECT wait":
			ss, seq := a[5].Any().(*SessionState), uint64(a[6].Int())
			<-ss.QueryContext(seq).Done()
			panic("query cancelled")
		default:
			panic("unsupported test query")
		}
		return NewInt(0)
	})
	if encrypted {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		s.tls = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}
	}
	for _, setup := range configure {
		setup(s)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.listener = listener
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	t.Cleanup(func() { listener.Close(); <-done })
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	dsn := fmt.Sprintf("server=%s;port=%s;user id=alice;password=secret;database=fixture;encrypt=disable;connection timeout=5", host, port)
	if encrypted {
		dsn = strings.Replace(dsn, "encrypt=disable", "encrypt=true;TrustServerCertificate=true", 1)
	}
	connector, err := tdsdriver.NewConnector(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db, dsn
}

func TestTDSClient(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		t.Run(fmt.Sprintf("tls=%v", encrypted), func(t *testing.T) {
			db, dsn := startTDSTestServer(t, encrypted)
			if err := db.Ping(); err != nil {
				t.Fatal(err)
			}
			var i int64
			var f float64
			var b bool
			var s string
			var n any
			if err := db.QueryRow("SELECT types").Scan(&i, &f, &b, &s, &n); err != nil {
				t.Fatal(err)
			}
			if i != 9007199254740993 || f != 2.5 || !b || s != "Grüße 🐘" || n != nil {
				t.Fatalf("incorrect values: %v %v %v %q %v", i, f, b, s, n)
			}
			injection := "'; DROP TABLE t; -- 🐘"
			if err := db.QueryRow("SELECT params", sql.Named("i", int64(9007199254740993)), sql.Named("s", injection), sql.Named("n", nil), sql.Named("b", true), sql.Named("f", 2.5)).Scan(&i, &s, &n, &b, &f); err != nil {
				t.Fatal(err)
			}
			if i != 9007199254740993 || s != injection || n != nil || !b || f != 2.5 {
				t.Fatal("parameter corruption")
			}
			if err := db.QueryRow("SELECT long").Scan(&s); err != nil {
				t.Fatal(err)
			}
			if s != strings.Repeat("ä🐘", 30000) {
				t.Fatal("fragmented Unicode result corrupted")
			}
			rows, err := db.Query("SELECT many")
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for rows.Next() {
				if err := rows.Scan(&i); err != nil {
					t.Fatal(err)
				}
				if i != int64(count) {
					t.Fatal("stream order")
				}
				count++
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
			if count != 400 {
				t.Fatalf("got %d rows", count)
			}
			rows, err = db.Query("SELECT 1;SELECT 1")
			if err != nil {
				t.Fatal(err)
			}
			for set := 0; set < 2; set++ {
				if !rows.Next() {
					t.Fatal("missing batch row")
				}
				if err := rows.Scan(&i); err != nil {
					t.Fatal(err)
				}
				if i != 1 {
					t.Fatal(i)
				}
				if rows.Next() {
					t.Fatal("extra batch row")
				}
				if set == 0 && !rows.NextResultSet() {
					t.Fatal("missing result set")
				}
			}
			rows.Close()
			if err := db.QueryRow("SELECT empty").Scan(&i); err != sql.ErrNoRows {
				t.Fatalf("empty result: %v", err)
			}
			tx, err := db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow("SELECT 1").Scan(&i); err != nil {
				t.Fatal(err)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			tx, err = db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("unsupported"); err == nil {
				t.Fatal("expected SQL error")
			}
			if err := db.QueryRow("SELECT 1").Scan(&i); err != nil {
				t.Fatal("connection did not recover:", err)
			}
			badConnector, err := tdsdriver.NewConnector(strings.Replace(dsn, "secret", "wrong", 1))
			if err != nil {
				t.Fatal(err)
			}
			bad := sql.OpenDB(badConnector)
			defer bad.Close()
			if err := bad.Ping(); err == nil {
				t.Fatal("invalid credentials accepted")
			}
		})
	}
}

func TestTDSAttention(t *testing.T) {
	db, _ := startTDSTestServer(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := db.ExecContext(ctx, "SELECT wait"); err == nil {
		t.Fatal("cancelled query succeeded")
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var value int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&value); err != nil {
		t.Fatal("connection failed after ATTENTION:", err)
	}
}

func TestTDSOffsetReleasedClientRoundtrip(t *testing.T) {
	for _, offset := range []int{-840, -90, 0, 150, 840} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			input := time.Date(2024, 2, 29, 0, 15, 30, 123456700, time.FixedZone("fixture", offset*60))
			utc := input.UTC()
			day := (time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).Unix() - tdsDateEpoch.Unix()) / 86400
			clock := int64(utc.Hour()*3600+utc.Minute()*60+utc.Second())*10000000 + int64(utc.Nanosecond()/100)
			expected := tdsTemporalPayload(day, clock, int64(offset))
			db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
				convert := server.convertParameter
				server.convertParameter = NewFunc(func(a ...Scmer) Scmer {
					if tdsDescriptorValue(a[1], "wire_kind").Int() != 0x2b || !Equal(a[0], expected) {
						panic("released client offset parameter lost its wire payload")
					}
					return Apply(convert, a...)
				})
				server.query = NewFunc(func(a ...Scmer) Scmer {
					if a[1].String() != "SELECT offset" {
						panic("unexpected offset fixture query")
					}
					value := Apply(a[4], NewString("tsql_param:v"))
					Apply(a[3], NewSlice([]Scmer{NewString("v")}), NewSlice([]Scmer{NewSlice([]Scmer{
						NewString("wire_kind"), NewInt(0x2b), NewString("scale"), NewInt(7),
					})}))
					Apply(a[2], NewSlice([]Scmer{NewString("v"), value}))
					return NewInt(1)
				})
			})
			var actual time.Time
			if err := db.QueryRow("SELECT offset", sql.Named("v", tdsdriver.DateTimeOffset(input))).Scan(&actual); err != nil {
				t.Fatal(err)
			}
			_, actualOffset := actual.Zone()
			if !actual.Equal(input) || actualOffset != offset*60 {
				t.Fatal("offset parameter/result changed", input, actual, actualOffset)
			}
		})
	}
}

func TestTDSParametersAreInvocationLocal(t *testing.T) {
	base := NewSession()
	server := &tdsServer{}
	tdsInstallTestFrontend(server)
	connection := &tdsConnection{server: server}
	first := connection.parameterSession(base, map[string]Scmer{"v": NewInt(1), "null": NewNil()}, nil)
	second := connection.parameterSession(base, map[string]Scmer{"v": NewInt(2)}, nil)
	if first.Func()(NewString("tsql_param:v")).Int() != 1 || second.Func()(NewString("tsql_param:v")).Int() != 2 {
		t.Fatal("bindings leaked across requests")
	}
	if !first.Func()(NewString("tsql_bound:null")).Bool() || second.Func()(NewString("tsql_bound:null")).Bool() {
		t.Fatal("NULL and absent bindings confused")
	}
	if len(base.Func()().Slice()) != 0 {
		t.Fatal("parameter names retained in session")
	}
	first.Func()(NewString("schema"), NewString("other"))
	if base.Func()(NewString("schema")).String() != "other" {
		t.Fatal("ordinary session state not forwarded")
	}
}

func TestTDSFrontendSessionProcedureUsesNativeABI(t *testing.T) {
	env := &Env{Vars: make(map[Symbol]Scmer), Outer: &Globalenv}
	binder := EvalAll(t.Name(), `(lambda (base params specs)
		(lambda args (if (and (> (count args) 0) (equal? (car args) "parameter"))
			(get_assoc params "v") (apply base args))))`, env)
	base := NewSession()
	if Apply(binder, base, NewSlice(nil), NewSlice(nil)).GetTag() == tagFunc {
		t.Fatal("fixture must exercise a frontend procedure rather than a native session")
	}
	connection := &tdsConnection{server: &tdsServer{bindSession: binder}}
	first := connection.parameterSession(base, map[string]Scmer{"v": NewInt(1)}, nil)
	second := connection.parameterSession(base, map[string]Scmer{"v": NewInt(2)}, nil)
	if first.Func()(NewString("parameter")).Int() != 1 || second.Func()(NewString("parameter")).Int() != 2 {
		t.Fatal("frontend procedure bindings leaked between invocations")
	}
	if !base.Func()(NewString("parameter")).IsNil() {
		t.Fatal("frontend input parameter leaked into the connection session")
	}
	transaction := NewAny(&struct{}{})
	first.Func()(NewString("__memcp_tx"), transaction)
	if second.Func()(NewString("__memcp_tx")) != transaction {
		t.Fatal("frontend procedure adapter did not preserve the base transaction handle")
	}
	second.Func()(NewString("schema"), NewString("other"))
	if first.Func()(NewString("schema")).String() != "other" {
		t.Fatal("ordinary connection state was not forwarded through the procedure")
	}
	connection.server.bindSession = NewFunc(func(...Scmer) Scmer { return NewNil() })
	expectTDSPanic(t, func() { connection.parameterSession(base, nil, nil) })
}

func TestTDSAttentionDuringBatchPreparation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &tdsRequest{ctx: ctx, cancel: cancel}
	if !request.cancelQuery(nil, 0) || ctx.Err() != context.Canceled {
		t.Fatal("attention did not cancel batch preparation")
	}
	if !request.finish() {
		t.Fatal("attention acknowledgement lost")
	}
	if request.cancelQuery(nil, 0) {
		t.Fatal("late attention consumed after completion")
	}
}

func TestTDSMalformedInput(t *testing.T) {
	for _, data := range [][]byte{nil, {1, 1, 0, 7, 0, 0, 0, 0}, {1, 0, 0, 8, 0, 0, 0, 0}, {1, 1, 0, 9, 0, 0, 0, 0}} {
		if _, err := readTDSMessage(bytes.NewReader(data)); err == nil {
			t.Fatalf("accepted malformed packet %x", data)
		}
	}
	for _, data := range [][]byte{nil, {0, 0, 0, 0}, {255, 255, 255, 255}} {
		if _, err := tdsStripHeaders(data); err == nil {
			t.Fatal("accepted invalid headers")
		}
	}
	var packet bytes.Buffer
	if err := writeTDSMessage(&packet, 1, tdsUTF16("SELECT '😀'")); err != nil {
		t.Fatal(err)
	}
	message, err := readTDSMessage(&packet)
	if err != nil {
		t.Fatal(err)
	}
	text, err := tdsText(message.data)
	if err != nil || text != "SELECT '😀'" {
		t.Fatal("UTF16 roundtrip", text, err)
	}
	if _, err := decodeTDSLogin(make([]byte, 94)); err == nil {
		t.Fatal("invalid login accepted")
	}
	d := tdsDecoder{data: []byte{0xff}}
	d.parameter()
	if d.err == nil {
		t.Fatal("unknown parameter type accepted")
	}
	var headers [4]byte
	binary.LittleEndian.PutUint32(headers[:], 4)
	if _, err := decodeTDSRPC(headers[:]); err == nil {
		t.Fatal("truncated RPC accepted")
	}
	parts, err := splitTDSBatch("SELECT N';';SELECT [a;b];/* x; /* y */ */SELECT 1 -- ;\n;")
	if err != nil || len(parts) != 3 {
		t.Fatal(parts, err)
	}
	if _, err := splitTDSBatch("SELECT 'unterminated"); err == nil {
		t.Fatal("invalid batch accepted")
	}
}

func TestTSQLScriptReader(t *testing.T) {
	script := "\ufeffSET ANSI_NULLS ON\nGO\nINSERT t VALUES (N'first\nGO\nlast;🐘')\nINSERT t VALUES (NULL); -- end\nGO\n"
	var statements []string
	ReadTSQLScript(strings.NewReader(script), func(s string) { statements = append(statements, s) }, nil)
	if len(statements) != 3 || !strings.Contains(statements[1], "\nGO\n") || !strings.Contains(statements[1], "last;🐘") {
		t.Fatalf("incorrect script boundaries: %#v", statements)
	}
	ReadTSQLScript(strings.NewReader("INSERT t VALUES ('first\r\nGO\r\nlast');\r\n"), func(s string) {
		if !strings.Contains(s, "first\r\nGO\r\nlast") {
			t.Fatal("literal CRLF bytes were changed")
		}
	}, nil)
	for _, bigEndian := range []bool{false, true} {
		encoded := tdsUTF16("SELECT N'ä🐘';\nGO\n")
		bom := []byte{0xff, 0xfe}
		if bigEndian {
			bom = []byte{0xfe, 0xff}
			for i := 0; i < len(encoded); i += 2 {
				encoded[i], encoded[i+1] = encoded[i+1], encoded[i]
			}
		}
		ReadTSQLScript(bytes.NewReader(append(bom, encoded...)), func(s string) {
			if s != "SELECT N'ä🐘'" {
				t.Fatalf("bad UTF16 script: %q", s)
			}
		}, nil)
	}
	for _, invalid := range []string{"SELECT 'unterminated", "/* comment", "GO 2\n", "\xff\xfe\x00\xd8", "SELECT \xff"} {
		t.Run(fmt.Sprintf("invalid=%q", invalid), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("invalid script accepted")
				}
			}()
			ReadTSQLScript(strings.NewReader(invalid), func(string) {}, nil)
		})
	}
}

func TestTSQLScriptConditionalLineBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		want         []string
	}{
		{"guard", "IF OBJECT_ID(N'dbo.item',N'U') IS NULL\nCREATE TABLE dbo.item(id INT)\nSET ANSI_NULLS ON\nGO\n", []string{"IF OBJECT_ID(N'dbo.item',N'U') IS NULL\nCREATE TABLE dbo.item(id INT)", "SET ANSI_NULLS ON"}},
		{"else", "IF 1=1\nINSERT dbo.item VALUES(1)\nELSE\nINSERT dbo.item VALUES(2)\nSET ANSI_NULLS ON\n", []string{"IF 1=1\nINSERT dbo.item VALUES(1)\nELSE\nINSERT dbo.item VALUES(2)", "SET ANSI_NULLS ON"}},
		{"quoted keywords", "/* CREATE */ IF N'DROP ELSE CREATE'=N'DROP ELSE CREATE'\n-- INSERT\nDROP TABLE dbo.item;\n", []string{"IF N'DROP ELSE CREATE'=N'DROP ELSE CREATE'\n \nDROP TABLE dbo.item"}},
		{"case else", "IF 1=1 SELECT CASE WHEN 1=0 THEN N'ELSE' ELSE N'value' END AS value\nCREATE TABLE dbo.other(id INT)\n", []string{"IF 1=1 SELECT CASE WHEN 1=0 THEN N'ELSE' ELSE N'value' END AS value", "CREATE TABLE dbo.other(id INT)"}},
		{"inline guard", "IF 1=0 DROP TABLE dbo.item\nCREATE TABLE dbo.other(id INT)\n", []string{"IF 1=0 DROP TABLE dbo.item", "CREATE TABLE dbo.other(id INT)"}},
		{"parameter keywords", "IF @create=1 AND @else=0\nCREATE TABLE dbo.item(id INT);\n", []string{"IF @create=1 AND @else=0\nCREATE TABLE dbo.item(id INT)"}},
		{"consecutive guards", "IF 1=0 DROP TABLE dbo.first\nIF 1=0 DROP TABLE dbo.second\nGO\n", []string{"IF 1=0 DROP TABLE dbo.first", "IF 1=0 DROP TABLE dbo.second"}},
		{"ordinary ddl", "CREATE TABLE dbo.item(id INT)\nCREATE TABLE dbo.other(id INT)\n", []string{"CREATE TABLE dbo.item(id INT)", "CREATE TABLE dbo.other(id INT)"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			ReadTSQLScript(strings.NewReader(tc.script), func(statement string) { got = append(got, statement) }, nil)
			if len(got) != len(tc.want) {
				t.Fatalf("conditional statement split incorrectly: %#v", got)
			}
			for i := range got {
				if strings.Join(strings.Fields(got[i]), " ") != strings.Join(strings.Fields(tc.want[i]), " ") {
					t.Fatalf("statement %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestTSQLScriptBatchScopes(t *testing.T) {
	var events []string
	ReadTSQLScript(strings.NewReader("INSERT t VALUES ('GO'); SELECT 1;\nGO\nSELECT 2;\n"), func(statement string) {
		events = append(events, statement)
	}, func() { events = append(events, "scope") })
	want := []string{"scope", "INSERT t VALUES ('GO')", "SELECT 1", "scope", "SELECT 2"}
	if fmt.Sprint(events) != fmt.Sprint(want) {
		t.Fatal("statement boundaries were confused with batch scopes", events)
	}
}

func TestTDSTransactionManagerSuccessor(t *testing.T) {
	for _, kind := range []uint16{7, 8} {
		t.Run(fmt.Sprintf("request=%d", kind), func(t *testing.T) {
			recorded := &tdsRecordingConnection{}
			var statements []string
			server := &tdsServer{}
			server.query = NewFunc(func(a ...Scmer) Scmer {
				statement := a[1].String()
				statements = append(statements, statement)
				active := statement == "BEGIN TRANSACTION"
				value := NewNil()
				if active {
					value = NewBool(true)
				}
				return a[4].Func()(NewString("transaction"), value)
			})
			connection := &tdsConnection{server: server, conn: recorded, database: "fixture", ss: RegisterSession("alice", "fixture", "fixture")}
			defer UnregisterSession(connection.ss.ID)
			connection.newSession("alice")
			send := func(payload []byte, descriptor uint64) []byte {
				t.Helper()
				recorded.Reset()
				headers := append([]byte(nil), tdsRawRPC(10)[:22]...)
				binary.LittleEndian.PutUint64(headers[10:], descriptor)
				if err := connection.execute(tdsMessage{kind: 14, data: append(headers, payload...)}); err != nil {
					t.Fatal(err)
				}
				message, err := readTDSMessage(bytes.NewReader(recorded.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				return message.data
			}
			send([]byte{5, 0, 2, 0}, 0)
			original := connection.transaction
			if original == 0 {
				t.Fatal("begin did not publish a descriptor")
			}
			statements = nil
			data := send([]byte{byte(kind), 0, 0, 1, 2, 0}, original)
			completion, envKind := "COMMIT", byte(9)
			if kind == 8 {
				completion, envKind = "ROLLBACK", 10
			}
			if len(statements) != 2 || statements[0] != completion || statements[1] != "BEGIN TRANSACTION" {
				t.Fatal("successor ran in the wrong order", statements)
			}
			current := connection.transaction
			if current == 0 || current == original || connection.session.Func()(NewString("transaction")).IsNil() {
				t.Fatal("successor descriptor/session missing", original, current)
			}
			var expected bytes.Buffer
			tdsTransaction(&expected, envKind, original)
			tdsDone(&expected, 0xfd, 0x11, 0)
			tdsTransaction(&expected, 8, current)
			tdsDone(&expected, 0xfd, 0x14, 0)
			if !bytes.Equal(data, expected.Bytes()) {
				t.Fatalf("completion/begin descriptor stream: %x, want %x", data, expected.Bytes())
			}
			// The old descriptor must be rejected after publication, without
			// completing or replacing the successor transaction.
			statements = nil
			data = send([]byte{byte(kind), 0, 0, 0}, original)
			if len(statements) != 0 || connection.transaction != current || !bytes.Contains(data, []byte{0xaa}) {
				t.Fatal("stale descriptor changed the transaction")
			}
			// Invalid successor payloads are rejected before the current
			// transaction is committed or rolled back.
			for _, payload := range [][]byte{
				{byte(kind), 0, 0, 2},
				{byte(kind), 0, 0, 1},
				{byte(kind), 0, 0, 1, 2},
				{byte(kind), 0, 0, 1, 4, 0},
				{byte(kind), 0, 0, 1, 2, 2, 'x', 0},
				{byte(kind), 0, 0, 0, 2, 0},
			} {
				data = send(payload, current)
				if len(statements) != 0 || connection.transaction != current || connection.session.Func()(NewString("transaction")).IsNil() || !bytes.Contains(data, []byte{0xaa}) {
					t.Fatal("invalid request changed the current transaction", payload)
				}
			}
		})
	}
}

func TestTDSTransactionManagerFailureDoesNotBeginSuccessor(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprintf("completion_clears_transaction=%v", rollback), func(t *testing.T) {
			recorded := &tdsRecordingConnection{}
			var statements []string
			server := &tdsServer{}
			server.query = NewFunc(func(a ...Scmer) Scmer {
				statements = append(statements, a[1].String())
				if rollback {
					a[4].Func()(NewString("transaction"), NewNil())
				}
				panic("transaction completion failed")
			})
			connection := &tdsConnection{server: server, conn: recorded, database: "fixture", transaction: 123, ss: RegisterSession("alice", "fixture", "fixture")}
			defer UnregisterSession(connection.ss.ID)
			connection.newSession("alice")
			connection.session.Func()(NewString("transaction"), NewBool(true))
			headers := append([]byte(nil), tdsRawRPC(10)[:22]...)
			binary.LittleEndian.PutUint64(headers[10:], 123)
			if err := connection.execute(tdsMessage{kind: 14, data: append(headers, 7, 0, 0, 1, 2, 0)}); err != nil {
				t.Fatal(err)
			}
			message, err := readTDSMessage(bytes.NewReader(recorded.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if len(statements) != 1 || statements[0] != "COMMIT" {
				t.Fatal("completion error began a successor", statements)
			}
			wantDescriptor := uint64(123)
			if rollback {
				wantDescriptor = 0
			}
			if connection.transaction != wantDescriptor {
				t.Fatal("completion error descriptor did not reflect session state", connection.transaction)
			}
			if !bytes.Contains(message.data, []byte{0xaa}) {
				t.Fatal("completion error not returned")
			}
			var begin bytes.Buffer
			tdsTransaction(&begin, 8, 123)
			if bytes.Contains(message.data, begin.Bytes()) {
				t.Fatal("failure published a begin descriptor")
			}
		})
	}
}
