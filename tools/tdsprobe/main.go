/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

// This probe is a client integration test, not a server or SQL implementation.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode/utf16"

	tdsclient "github.com/denisenkom/go-mssqldb"
	"github.com/golang-sql/civil"
	"github.com/golang-sql/sqlexp"
)

type config struct {
	HostName         *string     `json:"hostname"`
	AppName          *string     `json:"appname"`
	Host             string      `json:"host"`
	Port             int         `json:"port"`
	Database         string      `json:"database"`
	Username         string      `json:"username"`
	Password         string      `json:"password"`
	ExpectLoginError bool        `json:"expect_login_error"`
	Statements       []statement `json:"statements"`
	RPCPackets       []rpcPacket `json:"rpc_packets"`
}

// Explicit packet fixtures exercise descriptor/length cases which a released
// SQL driver normalizes before transmission. The YAML owns every payload byte;
// this bounded plain-TCP path adds framing/login only, never SQL interpretation.
type rpcPacket struct {
	PacketType         byte     `json:"packet_type"`
	ContinuationStatus byte     `json:"continuation_status"`
	PayloadHex         string   `json:"payload_hex"`
	ContainsHex        []string `json:"contains_hex"`
	Contains           []string `json:"contains"`
	Error              bool     `json:"error"`
	DoneCount          *bool    `json:"done_count"`
	DoneRows           *uint64  `json:"done_rows"`
}

type statement struct {
	Connection   string       `json:"connection"`
	Reset        bool         `json:"reset"`
	Close        bool         `json:"close"`
	SQL          string       `json:"sql"`
	Parameters   []parameter  `json:"parameters"`
	Prepare      bool         `json:"prepare"`
	PrepareOnly  bool         `json:"prepare_only"`
	Declarations string       `json:"declarations"`
	Bulk         *bulkRequest `json:"bulk"`
	Expect       expectation  `json:"expect"`
}

type bulkRequest struct {
	Table            string   `json:"table"`
	Columns          []string `json:"columns"`
	Rows             [][]any  `json:"rows"`
	KeepNulls        bool     `json:"keep_nulls"`
	FireTriggers     bool     `json:"fire_triggers"`
	CheckConstraints bool     `json:"check_constraints"`
}

type parameter struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type expectation struct {
	ResultSets  []resultSetExpectation  `json:"result_sets"`
	Messages    []map[string]any        `json:"messages"`
	Error       bool                    `json:"error"`
	Contains    []string                `json:"contains"`
	Rows        *int                    `json:"rows"`
	Data        []map[string]any        `json:"data"`
	Columns     []string                `json:"columns"`
	ColumnTypes []columnTypeExpectation `json:"column_types"`
}

type resultSetExpectation struct {
	Rows        *int                    `json:"rows"`
	Data        []map[string]any        `json:"data"`
	Columns     []string                `json:"columns"`
	ColumnTypes []columnTypeExpectation `json:"column_types"`
}

type columnTypeExpectation struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Length    *int64 `json:"length"`
	Nullable  *bool  `json:"nullable"`
	Precision *int64 `json:"precision"`
	Scale     *int64 `json:"scale"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("TDS client probe passed")
}

func probeUTF16(text string) []byte {
	units := utf16.Encode([]rune(text))
	data := make([]byte, 2*len(units))
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2*i:], unit)
	}
	return data
}

func probePackets(cfg config) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)), 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	exchange := func(kind byte, payload []byte, continuationStatus ...byte) ([]byte, error) {
		if len(payload) > 65536 {
			return nil, fmt.Errorf("raw TDS fixture exceeds bounded message size")
		}
		for offset, id := 0, byte(1); ; id++ {
			end := min(offset+4088, len(payload))
			status := byte(1)
			if end < len(payload) {
				status = 0
				if len(continuationStatus) != 0 {
					status = continuationStatus[0]
				}
			}
			packet := make([]byte, 8+end-offset)
			packet[0], packet[1], packet[6] = kind, status, id
			binary.BigEndian.PutUint16(packet[2:], uint16(len(packet)))
			copy(packet[8:], payload[offset:end])
			if _, err := io.Copy(conn, bytes.NewReader(packet)); err != nil {
				return nil, err
			}
			if end == len(payload) {
				break
			}
			offset = end
		}
		var answer []byte
		for {
			var header [8]byte
			if _, err := io.ReadFull(conn, header[:]); err != nil {
				return nil, err
			}
			size := int(binary.BigEndian.Uint16(header[2:])) - 8
			if header[0] != 4 || size < 0 || len(answer)+size > 65536 {
				return nil, fmt.Errorf("invalid or oversized raw TDS response")
			}
			start := len(answer)
			answer = append(answer, make([]byte, size)...)
			if _, err := io.ReadFull(conn, answer[start:]); err != nil {
				return nil, err
			}
			if header[1]&1 != 0 {
				return answer, nil
			}
		}
	}
	if _, err := exchange(0x12, []byte{1, 0, 6, 0, 1, 0xff, 2}); err != nil {
		return err
	}
	login := make([]byte, 94)
	binary.LittleEndian.PutUint32(login[4:], 0x74000004)
	binary.LittleEndian.PutUint32(login[8:], 4096)
	for _, field := range []struct {
		offset int
		text   string
	}{{40, cfg.Username}, {44, cfg.Password}, {68, cfg.Database}} {
		data := probeUTF16(field.text)
		if len(login)+len(data) > 4088 {
			return fmt.Errorf("raw TDS login exceeds fixture bounds")
		}
		binary.LittleEndian.PutUint16(login[field.offset:], uint16(len(login)))
		binary.LittleEndian.PutUint16(login[field.offset+2:], uint16(len(data)/2))
		if field.offset == 44 {
			for i, value := range data {
				data[i] = (value<<4 | value>>4) ^ 0xa5
			}
		}
		login = append(login, data...)
	}
	binary.LittleEndian.PutUint32(login, uint32(len(login)))
	answer, err := exchange(0x10, login)
	if err != nil || len(answer) == 0 || answer[0] != 0xad {
		return fmt.Errorf("raw TDS login failed: %v", err)
	}
	for index, fixture := range cfg.RPCPackets {
		payload, err := hex.DecodeString(fixture.PayloadHex)
		if err != nil {
			return fmt.Errorf("TDS fixture %d: %w", index+1, err)
		}
		kind := fixture.PacketType
		if kind == 0 {
			kind = 3 // Existing fixtures are RPC requests.
		}
		if kind != 1 && kind != 3 && kind != 7 {
			return fmt.Errorf("unsupported fixture packet type %d", kind)
		}
		answer, err := exchange(kind, payload, fixture.ContinuationStatus)
		if err != nil {
			return fmt.Errorf("TDS fixture %d: %w", index+1, err)
		}
		if len(answer) < 13 || (answer[len(answer)-13] != 0xfd && answer[len(answer)-13] != 0xfe && answer[len(answer)-13] != 0xff) {
			return fmt.Errorf("TDS fixture %d lacks a final completion", index+1)
		}
		status := binary.LittleEndian.Uint16(answer[len(answer)-12:])
		if fixture.DoneCount != nil && (status&0x10 != 0) != *fixture.DoneCount {
			return fmt.Errorf("TDS fixture %d DONE_COUNT=%v, want %v", index+1, status&0x10 != 0, *fixture.DoneCount)
		}
		if fixture.DoneRows != nil {
			count := binary.LittleEndian.Uint64(answer[len(answer)-8:])
			if count != *fixture.DoneRows {
				return fmt.Errorf("TDS fixture %d final row count %d, want %d", index+1, count, *fixture.DoneRows)
			}
		}
		failed := status&2 != 0
		if failed != fixture.Error {
			return fmt.Errorf("TDS fixture %d expected error=%v, got error=%v", index+1, fixture.Error, failed)
		}
		for _, text := range fixture.Contains {
			if !bytes.Contains(answer, probeUTF16(text)) {
				return fmt.Errorf("TDS fixture %d lacks expected UTF16 content %q", index+1, text)
			}
		}
		for _, text := range fixture.ContainsHex {
			pattern, err := hex.DecodeString(text)
			if err != nil || len(pattern) == 0 || !bytes.Contains(answer, pattern) {
				return fmt.Errorf("TDS fixture %d lacks expected hex content %q", index+1, text)
			}
		}
	}
	return nil
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: tdsprobe <json-config>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	var cfg config
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&cfg); err != nil {
		return err
	}
	if len(cfg.RPCPackets) != 0 {
		if len(cfg.Statements) != 0 || cfg.ExpectLoginError {
			return fmt.Errorf("raw TDS fixtures require their own probe")
		}
		return probePackets(cfg)
	}
	u := &url.URL{Scheme: "sqlserver", Host: net.JoinHostPort(cfg.Host, fmt.Sprint(cfg.Port)), User: url.UserPassword(cfg.Username, cfg.Password)}
	q := url.Values{"database": {cfg.Database}, "encrypt": {"disable"}, "connection timeout": {"10"}}
	if cfg.HostName != nil {
		q.Set("workstation id", *cfg.HostName)
	}
	if cfg.AppName != nil {
		q.Set("app name", *cfg.AppName)
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if cfg.ExpectLoginError {
		if err == nil {
			return fmt.Errorf("login unexpectedly succeeded")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "login") {
			return fmt.Errorf("expected login rejection, got: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	connections := make(map[string]*sql.Conn)
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()
	for i, stmt := range cfg.Statements {
		conn := connections[stmt.Connection]
		if stmt.Close {
			if conn == nil || stmt.Reset || stmt.SQL != "" {
				return fmt.Errorf("statement %d: close requires an open connection and no SQL/reset", i+1)
			}
			// Discard the underlying driver connection instead of returning it to
			// the idle pool, so this action exercises actual socket-close cleanup.
			err := conn.Raw(func(any) error { return driver.ErrBadConn })
			if err != nil && !errors.Is(err, driver.ErrBadConn) {
				return fmt.Errorf("statement %d close: %w", i+1, err)
			}
			conn.Close()
			delete(connections, stmt.Connection)
			continue
		}
		if conn == nil {
			conn, err = db.Conn(ctx)
			if err != nil {
				return fmt.Errorf("statement %d connection: %w", i+1, err)
			}
			connections[stmt.Connection] = conn
		}
		if stmt.Reset {
			if err := conn.Raw(func(native any) error {
				resetter, ok := native.(driver.SessionResetter)
				if !ok {
					return fmt.Errorf("driver does not support session reset")
				}
				return resetter.ResetSession(ctx)
			}); err != nil {
				return fmt.Errorf("statement %d reset: %w", i+1, err)
			}
		}
		if stmt.SQL == "" && stmt.Bulk == nil {
			continue
		}
		if err := execute(ctx, conn, stmt); err != nil {
			return fmt.Errorf("statement %d: %w\nSQL: %s", i+1, err, stmt.SQL)
		}
	}
	return nil
}

// These explicit client contracts ask the driver to send actual temporal RPC
// fields. Plain strings would exercise only text-to-calendar SQL conversion.
func temporalParameter(encoded map[string]any) (any, error) {
	kind, ok := encoded["temporal_type"].(string)
	text, textOK := encoded["value"].(string)
	if !ok || !textOK || len(encoded) != 2 {
		return nil, fmt.Errorf("invalid temporal probe parameter")
	}
	layout := time.RFC3339Nano
	switch kind {
	case "date":
		layout = "2006-01-02"
	case "time":
		layout = "15:04:05.999999999"
	case "datetime", "datetime2", "datetimeoffset":
	default:
		return nil, fmt.Errorf("unsupported temporal probe contract: %s", kind)
	}
	value, err := time.Parse(layout, text)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "date":
		return civil.DateOf(value), nil
	case "time":
		return civil.TimeOf(value), nil
	case "datetime":
		return tdsclient.DateTime1(value), nil
	case "datetime2":
		return civil.DateTimeOf(value), nil
	default:
		return tdsclient.DateTimeOffset(value), nil
	}
}

func probeValue(v any) (any, error) {
	if encoded, ok := v.(map[string]any); ok {
		if _, temporal := encoded["temporal_type"]; temporal {
			return temporalParameter(encoded)
		}
		text, ok := encoded["binary_hex"].(string)
		if !ok || len(encoded) != 1 {
			return nil, fmt.Errorf("unsupported probe payload")
		}
		return hex.DecodeString(text)
	}
	if n, ok := v.(json.Number); ok {
		if strings.ContainsAny(string(n), ".eE") {
			return n.Float64()
		}
		return n.Int64()
	}
	return v, nil
}

// The YAML owns table, column mappings, source values and expectations. This
// drives the released client's real bulk protocol, not an INSERT SQL surrogate.
func executeBulk(ctx context.Context, conn *sql.Conn, stmt statement) error {
	b := stmt.Bulk
	prepared, err := conn.PrepareContext(ctx, tdsclient.CopyIn(b.Table, tdsclient.BulkOptions{
		KeepNulls: b.KeepNulls, FireTriggers: b.FireTriggers, CheckConstraints: b.CheckConstraints,
	}, b.Columns...))
	if err != nil {
		return checkEarlyError(stmt.Expect, err)
	}
	defer prepared.Close()
	for _, row := range b.Rows {
		values := make([]any, len(row))
		for i, v := range row {
			values[i], err = probeValue(v)
			if err != nil {
				return err
			}
		}
		if _, err = prepared.ExecContext(ctx, values...); err != nil {
			return checkEarlyError(stmt.Expect, err)
		}
	}
	result, err := prepared.ExecContext(ctx)
	if err != nil {
		return checkEarlyError(stmt.Expect, err)
	}
	if stmt.Expect.Error {
		return fmt.Errorf("expected bulk failure")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if stmt.Expect.Rows != nil && count != int64(*stmt.Expect.Rows) {
		return fmt.Errorf("bulk affected %d rows, want %d", count, *stmt.Expect.Rows)
	}
	return nil
}

func execute(ctx context.Context, conn *sql.Conn, stmt statement) error {
	if stmt.Bulk != nil {
		return executeBulk(ctx, conn, stmt)
	}
	if stmt.PrepareOnly {
		if stmt.Expect.ResultSets != nil {
			return fmt.Errorf("result_sets requires an executed query")
		}
		// Use the same driver message loop for pure Describe as for Execute.
		var handle int64
		zero := 0
		want := stmt.Expect
		want.Rows, want.Data, want.Columns, want.ColumnTypes = &zero, nil, nil, nil
		request := statement{SQL: "sp_prepare", Expect: want, Parameters: []parameter{
			{Name: "handle", Value: sql.Out{Dest: &handle}},
			{Name: "params", Value: stmt.Declarations},
			{Name: "stmt", Value: stmt.SQL},
		}}
		if err := execute(ctx, conn, request); err != nil {
			return err
		}
		if stmt.Expect.Error {
			return nil
		}
		if handle <= 0 {
			return fmt.Errorf("pure prepare did not return a handle")
		}
		_, err := conn.ExecContext(ctx, "sp_unprepare", sql.Named("handle", handle))
		return err
	}

	args := make([]any, len(stmt.Parameters))
	for i, p := range stmt.Parameters {
		v, err := probeValue(p.Value)
		if err != nil {
			return fmt.Errorf("parameter %s: %w", p.Name, err)
		}
		args[i] = sql.Named(p.Name, v)
	}
	var messageQueue *sqlexp.ReturnMessage
	if stmt.Expect.Messages != nil {
		messageQueue = &sqlexp.ReturnMessage{}
		args = append(args, messageQueue)
	}
	query := conn.QueryContext
	if stmt.Prepare {
		prepared, err := conn.PrepareContext(ctx, stmt.SQL)
		if err != nil {
			return checkEarlyError(stmt.Expect, err)
		}
		defer prepared.Close()
		query = func(ctx context.Context, _ string, args ...any) (*sql.Rows, error) {
			return prepared.QueryContext(ctx, args...)
		}
	}
	rows, err := query(ctx, stmt.SQL, args...)
	if err != nil {
		return checkEarlyError(stmt.Expect, err)
	}
	defer rows.Close()
	var data []map[string]any
	var messages []map[string]any
	var messageError error
	type messageBarrier struct{}
	type messageResult struct {
		messages []map[string]any
		err      error
	}
	var messageDone chan messageResult
	if messageQueue != nil {
		messageContext, cancelMessages := context.WithCancel(ctx)
		defer cancelMessages()
		messageDone = make(chan messageResult, 1)
		go func() {
			result := messageResult{}
			for {
				message := messageQueue.Message(messageContext)
				if messageContext.Err() != nil {
					messageDone <- result
					return
				}
				switch message := message.(type) {
				case sqlexp.MsgNotice:
					info, ok := message.Message.(tdsclient.Error)
					if !ok {
						result.err = fmt.Errorf("unknown driver informational payload: %T", message.Message)
						continue
					}
					result.messages = append(result.messages, map[string]any{"number": int64(uint32(info.Number)), "state": int64(info.State), "severity": int64(info.Class), "text": info.Message, "server": info.ServerName, "procedure": info.ProcName, "line": int64(uint32(info.LineNo))})
				case sqlexp.MsgError:
					if result.err == nil {
						result.err = message.Error
					}
				case messageBarrier:
					messageDone <- result
					return
				}
			}
		}()
	}
	setIndex := 0
	for {
		columns, err := rows.Columns()
		if err != nil {
			return checkEarlyError(stmt.Expect, err)
		}
		if err := checkResultColumns(rows, columns, stmt.Expect.Columns, stmt.Expect.ColumnTypes); err != nil {
			return err
		}
		if stmt.Expect.ResultSets != nil {
			if setIndex >= len(stmt.Expect.ResultSets) {
				return fmt.Errorf("unexpected result set %d with columns %v", setIndex+1, columns)
			}
			want := stmt.Expect.ResultSets[setIndex]
			if err := checkResultColumns(rows, columns, want.Columns, want.ColumnTypes); err != nil {
				return fmt.Errorf("result set %d: %w", setIndex+1, err)
			}
		}
		setStart := len(data)
		for rows.Next() {
			values := make([]any, len(columns))
			ptrs := make([]any, len(columns))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return err
			}
			row := make(map[string]any, len(columns))
			for i, name := range columns {
				row[name] = values[i]
			}
			data = append(data, row)
		}
		if err := rows.Err(); err != nil {
			if messageQueue == nil {
				return checkEarlyError(stmt.Expect, err)
			}
			if messageError == nil {
				messageError = err
			}
		}
		if stmt.Expect.ResultSets != nil {
			want := stmt.Expect.ResultSets[setIndex]
			if err := checkResultRows(data[setStart:], want.Rows, want.Data); err != nil {
				return fmt.Errorf("result set %d: %w", setIndex+1, err)
			}
		}
		setIndex++
		if !rows.NextResultSet() {
			break
		}
	}
	if stmt.Expect.ResultSets != nil && setIndex != len(stmt.Expect.ResultSets) {
		return fmt.Errorf("expected %d result sets, got %d", len(stmt.Expect.ResultSets), setIndex)
	}
	if messageQueue != nil {
		// All INFO/ERROR tokens precede the final DONE observed by Rows. An
		// explicit FIFO barrier drains them without depending on the driver's
		// optional final MsgNextResultSet, which cancellation may suppress.
		if err := sqlexp.ReturnMessageEnqueue(ctx, messageQueue, messageBarrier{}); err != nil {
			return err
		}
		result := <-messageDone
		messages = result.messages
		if messageError == nil {
			messageError = result.err
		}
	}
	if err := rows.Err(); err != nil && messageError == nil {
		messageError = err
	}
	if messageError != nil {
		if err := checkError(stmt.Expect, messageError); err != nil {
			return err
		}
	} else if stmt.Expect.Error {
		return fmt.Errorf("expected an error, query succeeded")
	}
	if stmt.Expect.Messages != nil {
		if len(messages) != len(stmt.Expect.Messages) {
			return fmt.Errorf("expected %d INFO messages, got %d: %v", len(stmt.Expect.Messages), len(messages), messages)
		}
		for i, expected := range stmt.Expect.Messages {
			for key, value := range expected {
				actual, ok := messages[i][key]
				if !ok || !equalValue(actual, value) {
					return fmt.Errorf("INFO message %d field %s: got %v, expected %v", i, key, actual, value)
				}
			}
		}
	}
	return checkResultRows(data, stmt.Expect.Rows, stmt.Expect.Data)
}

func checkResultColumns(rows *sql.Rows, columns []string, wantColumns []string, wantTypes []columnTypeExpectation) error {
	if wantColumns != nil && !reflect.DeepEqual(columns, wantColumns) {
		return fmt.Errorf("expected columns %v, got %v", wantColumns, columns)
	}
	if wantTypes != nil {
		types, err := rows.ColumnTypes()
		if err != nil {
			return err
		}
		if len(types) != len(wantTypes) {
			return fmt.Errorf("expected %d column contracts, got %d", len(wantTypes), len(types))
		}
		for i, expected := range wantTypes {
			actual := types[i]
			if expected.Name != actual.Name() || expected.Type != actual.DatabaseTypeName() {
				return fmt.Errorf("column %d expected %s %s, got %s %s", i, expected.Name, expected.Type, actual.Name(), actual.DatabaseTypeName())
			}
			if expected.Length != nil {
				length, ok := actual.Length()
				if !ok || length != *expected.Length {
					return fmt.Errorf("column %s length: expected %d, got %d (known=%t)", actual.Name(), *expected.Length, length, ok)
				}
			}
			if expected.Nullable != nil {
				nullable, ok := actual.Nullable()
				if !ok || nullable != *expected.Nullable {
					return fmt.Errorf("column %s nullable: expected %t, got %t (known=%t)", actual.Name(), *expected.Nullable, nullable, ok)
				}
			}
			// DecimalSize is the driver's decimal/numeric contract, not a
			// temporal scale API: go-mssqldb v0.12.3 reports DATETIME2 as unknown.
			// Temporal scale assertions use raw TYPEINFO fixtures; never turn
			// unknown driver metadata into an assumed expected value here.
			precision, scale, known := actual.DecimalSize()
			if expected.Precision != nil && (!known || precision != *expected.Precision) {
				return fmt.Errorf("column %s precision: expected %d, got %d (known=%t)", actual.Name(), *expected.Precision, precision, known)
			}
			if expected.Scale != nil && (!known || scale != *expected.Scale) {
				return fmt.Errorf("column %s scale: expected %d, got %d (known=%t)", actual.Name(), *expected.Scale, scale, known)
			}
		}
	}
	return nil
}

func checkResultRows(data []map[string]any, wantRows *int, wantData []map[string]any) error {
	if wantRows != nil && len(data) != *wantRows {
		return fmt.Errorf("expected %d rows, got %d: %v", *wantRows, len(data), data)
	}
	if wantData != nil {
		if len(data) != len(wantData) {
			return fmt.Errorf("expected %d data rows, got %d", len(wantData), len(data))
		}
		for i, want := range wantData {
			for name, expected := range want {
				actual, found := data[i][name]
				if !found || !equalValue(actual, expected) {
					return fmt.Errorf("row %d column %s: got %v (%T), expected %v (%T)", i, name, actual, actual, expected, expected)
				}
			}
		}
	}
	return nil
}

// A direct driver failure has no fully consumed token stream. Never claim INFO
// assertions passed merely because an expected error happened early.
func checkEarlyError(want expectation, err error) error {
	if want.ResultSets != nil {
		return fmt.Errorf("driver failed before result-set assertions could be evaluated: %w", err)
	}
	if want.Messages != nil {
		return fmt.Errorf("driver failed before INFO assertions could be evaluated: %w", err)
	}
	return checkError(want, err)
}

func checkError(want expectation, err error) error {
	if !want.Error {
		return err
	}
	for _, text := range want.Contains {
		if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(text)) {
			return fmt.Errorf("error %q does not contain %q", err, text)
		}
	}
	return nil
}

func equalValue(actual, expected any) bool {
	if encoded, ok := expected.(map[string]any); ok {
		if kind, ok := encoded["temporal_type"].(string); ok {
			text, textOK := encoded["value"].(string)
			got, typed := actual.(time.Time)
			if !textOK || !typed || len(encoded) != 2 {
				return false
			}
			switch kind {
			case "date":
				want, err := time.Parse("2006-01-02", text)
				return err == nil && got.Equal(want)
			case "time":
				want, err := time.Parse("15:04:05.999999999", text)
				return err == nil && civil.TimeOf(got) == civil.TimeOf(want)
			case "datetime", "datetime2", "smalldatetime":
				want, err := time.Parse(time.RFC3339Nano, text)
				return err == nil && got.Equal(want)
			default:
				return false
			}
		}
		text, ok := encoded["binary_hex"].(string)
		if !ok || len(encoded) != 1 {
			return false
		}
		want, err := hex.DecodeString(text)
		got, ok := actual.([]byte)
		return err == nil && ok && bytes.Equal(got, want)
	}
	if n, ok := expected.(json.Number); ok {
		switch v := actual.(type) {
		case int64:
			i, err := n.Int64()
			return err == nil && v == i
		case float64:
			want, validWant := new(big.Rat).SetString(string(n))
			got, validGot := new(big.Rat).SetString(fmt.Sprint(v))
			return validWant && validGot && want.Cmp(got) == 0
		default:
			return false
		}
	}
	if b, ok := actual.([]byte); ok {
		actual = string(b)
	}
	return reflect.DeepEqual(actual, expected)
}
