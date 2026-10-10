/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type tdsServer struct {
	listener                                           net.Listener
	auth, query, closeSession                          Scmer
	describe                                           Scmer
	metadata                                           Scmer
	declarations, convertParameter, bindSession, event Scmer
	resolveMetadata                                    Scmer
	database                                           string
	tls                                                *tls.Config
}

var tdsListenersMu sync.Mutex
var tdsListeners []*tdsServer
var tdsConnections sync.Map // *tdsConnection -> *SessionState
var nextTDSTransaction atomic.Uint64

// TDSServe binds synchronously, then accepts after the shared SQL bootstrap.
// Arguments: port, auth(user,password,database), query callback, close(session),
// default database, optional certificate/private-key paths, pure describe and
// metadata callbacks, then declaration parser, parameter codec, immutable
// parameter-session binder, session-event handler and metadata-name resolver.
func TDSServe(a ...Scmer) Scmer {
	server := &tdsServer{auth: a[1], query: a[2], closeSession: a[3], database: a[4].String()}
	if len(a) > 7 {
		server.describe = a[7]
	}
	if len(a) > 8 {
		server.metadata = a[8]
	}
	if len(a) > 9 {
		server.declarations = a[9]
	}
	if len(a) > 10 {
		server.convertParameter = a[10]
	}
	if len(a) > 11 {
		server.bindSession = a[11]
	}
	if len(a) > 12 {
		server.event = a[12]
	}
	if len(a) > 13 {
		server.resolveMetadata = a[13]
	}
	certPath, keyPath := "", ""
	if len(a) > 5 && !a[5].IsNil() {
		certPath = a[5].String()
	}
	if len(a) > 6 && !a[6].IsNil() {
		keyPath = a[6].String()
	}
	if (certPath == "") != (keyPath == "") {
		panic("both TDS TLS certificate and key are required")
	}
	if certPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			panic(err)
		}
		server.tls = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("", a[0].String()))
	if err != nil {
		panic(err)
	}
	server.listener = listener
	tdsListenersMu.Lock()
	tdsListeners = append(tdsListeners, server)
	tdsListenersMu.Unlock()
	go func() {
		<-waitForMySQLInitialization()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.serve(conn)
		}
	}()
	return NewBool(true)
}

func closeTDSListeners() {
	tdsListenersMu.Lock()
	listeners := tdsListeners
	tdsListeners = nil
	tdsListenersMu.Unlock()
	for _, server := range listeners {
		_ = server.listener.Close()
	}
}
func closeTDSConnections() {
	tdsConnections.Range(func(key, value any) bool {
		value.(*SessionState).Kill()
		_ = key.(*tdsConnection).conn.Close()
		return true
	})
}
func hasActiveTDSQueries() bool {
	active := false
	tdsConnections.Range(func(_, value any) bool {
		if strPtr(&value.(*SessionState).Command) == "Query" {
			active = true
			return false
		}
		return true
	})
	return active
}

type tdsTLSHandshakeConn struct {
	net.Conn
	buffer []byte
	raw    bool
}

func (c *tdsTLSHandshakeConn) Read(data []byte) (int, error) {
	if c.raw {
		return c.Conn.Read(data)
	}
	if len(c.buffer) == 0 {
		message, err := readTDSMessage(c.Conn)
		if err != nil {
			return 0, err
		}
		if message.kind != 0x12 {
			return 0, fmt.Errorf("expected TLS-in-TDS PRELOGIN packet")
		}
		c.buffer = message.data
	}
	n := copy(data, c.buffer)
	c.buffer = c.buffer[n:]
	return n, nil
}
func (c *tdsTLSHandshakeConn) Write(data []byte) (int, error) {
	if c.raw {
		return c.Conn.Write(data)
	}
	if err := writeTDSMessage(c.Conn, 0x12, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func tdsPreloginEncryption(data []byte) (byte, error) {
	encryption := byte(2)
	seen := make(map[byte]bool)
	for i := 0; i < len(data); i += 5 {
		if data[i] == 0xff {
			return encryption, nil
		}
		if len(data)-i < 5 {
			return 0, fmt.Errorf("truncated PRELOGIN options")
		}
		offset := int(binary.BigEndian.Uint16(data[i+1:]))
		length := int(binary.BigEndian.Uint16(data[i+3:]))
		if offset > len(data) || length > len(data)-offset || seen[data[i]] {
			return 0, fmt.Errorf("invalid PRELOGIN option bounds")
		}
		seen[data[i]] = true
		if data[i] == 1 {
			if length != 1 || data[offset] > 3 {
				return 0, fmt.Errorf("unsupported PRELOGIN encryption option")
			}
			encryption = data[offset]
		}
		if data[i] == 4 && (length != 1 || data[offset] != 0) {
			return 0, fmt.Errorf("MARS is unsupported")
		}
	}
	return 0, fmt.Errorf("unterminated PRELOGIN options")
}

type tdsLogin struct {
	user, password, database string
	browseMetadata           bool
}

func decodeTDSLogin(data []byte) (tdsLogin, error) {
	var login tdsLogin
	if len(data) < 94 || int(binary.LittleEndian.Uint32(data)) != len(data) {
		return login, fmt.Errorf("invalid LOGIN7 length")
	}
	if binary.LittleEndian.Uint32(data[4:]) < 0x74000004 {
		return login, fmt.Errorf("TDS 7.4 or later client required")
	}
	if data[25]&0x80 != 0 || data[27]&1 != 0 {
		return login, fmt.Errorf("integrated authentication and password changes are not supported")
	}
	login.browseMetadata = data[25]&2 != 0 || data[26]&0x10 != 0
	get := func(pos int, password bool) (string, error) {
		offset := int(binary.LittleEndian.Uint16(data[pos:]))
		n := 2 * int(binary.LittleEndian.Uint16(data[pos+2:]))
		if offset > len(data) || n > len(data)-offset {
			return "", fmt.Errorf("invalid LOGIN7 field bounds")
		}
		field := data[offset : offset+n]
		if password {
			decoded := make([]byte, len(field))
			for i, ch := range field {
				ch ^= 0xa5
				decoded[i] = ch<<4 | ch>>4
			}
			field = decoded
		}
		return tdsText(field)
	}
	var err error
	if login.user, err = get(40, false); err != nil {
		return login, err
	}
	if login.password, err = get(44, true); err != nil {
		return login, err
	}
	if login.database, err = get(68, false); err != nil {
		return login, err
	}
	return login, nil
}

func (s *tdsServer) handshake(conn net.Conn) (net.Conn, tdsLogin, error) {
	var login tdsLogin
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	message, err := readTDSMessage(conn)
	if err != nil {
		return conn, login, err
	}
	if message.kind != 0x12 {
		return conn, login, fmt.Errorf("expected PRELOGIN")
	}
	encryption, err := tdsPreloginEncryption(message.data)
	if err != nil {
		return conn, login, err
	}
	answer := byte(2) // ENCRYPT_NOT_SUP when no certificate was configured
	if s.tls != nil {
		answer = 3
	} // ENCRYPT_REQ: never downgrade a TLS listener
	response := []byte{0, 0, 11, 0, 6, 1, 0, 17, 0, 1, 0xff, 16, 0, 0, 1, 0, 0, answer}
	if err := writeTDSMessage(conn, 4, response); err != nil {
		return conn, login, err
	}
	if s.tls != nil {
		if encryption == 2 {
			return conn, login, fmt.Errorf("this TDS listener requires encryption")
		}
		wrapper := &tdsTLSHandshakeConn{Conn: conn}
		encrypted := tls.Server(wrapper, s.tls)
		if err := encrypted.Handshake(); err != nil {
			return conn, login, err
		}
		wrapper.raw = true
		conn = encrypted
	} else if encryption == 1 || encryption == 3 {
		return conn, login, fmt.Errorf("TDS TLS certificate is not configured")
	}
	message, err = readTDSMessage(conn)
	if err != nil {
		return conn, login, err
	}
	if message.kind != 0x10 {
		return conn, login, fmt.Errorf("expected LOGIN7")
	}
	login, err = decodeTDSLogin(message.data)
	if err != nil {
		return conn, login, err
	}
	if login.database == "" {
		login.database = s.database
	}
	if !Apply(s.auth, NewString(login.user), NewString(login.password), NewString(login.database)).Bool() {
		var tokens bytes.Buffer
		tdsError(&tokens, 18456, "Login failed")
		tdsDone(&tokens, 0xfd, 2, 0)
		_ = writeTDSMessage(conn, 4, tokens.Bytes())
		return conn, login, fmt.Errorf("TDS login denied")
	}
	var tokens, ack bytes.Buffer
	ack.Write([]byte{1, 0x74, 0, 0, 4})
	tdsBText(&ack, "MemCP")
	ack.Write([]byte{16, 0, 0, 1})
	tdsToken(&tokens, 0xad, ack.Bytes())
	tdsDatabase(&tokens, login.database, "")
	var packetEnv bytes.Buffer
	packetEnv.WriteByte(4)
	tdsBText(&packetEnv, "4096")
	tdsBText(&packetEnv, "4096")
	tdsToken(&tokens, 0xe3, packetEnv.Bytes())
	tdsDone(&tokens, 0xfd, 0, 0)
	if err := writeTDSMessage(conn, 4, tokens.Bytes()); err != nil {
		return conn, login, err
	}
	_ = conn.SetDeadline(time.Time{})
	login.password = ""
	return conn, login, nil
}

// session/database/transaction are owned by serve's goroutine. The reader only
// accesses immutable conn/ss and the atomic current request/attention tokens.
type tdsConnection struct {
	conn            net.Conn
	server          *tdsServer
	ss              *SessionState
	session         Scmer
	database        string
	initialDatabase string
	seq             atomic.Uint64
	request         atomic.Pointer[tdsRequest]
	transaction     uint64
	prepared        map[int32]tdsPreparedStatement
	preparedBytes   int
	nextPrepared    int32
	cursors         map[int32]*tdsCursor
	cursorBytes     int
	nextCursor      int32
	browseMetadata  bool
}

// ATTENTION belongs to the complete batch, including parsing and the gaps
// between statements. Completion and cancellation share a mutex so an
// attention cannot be consumed after the final acknowledgement was decided.
type tdsRequest struct {
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	done, attention bool
}

func (r *tdsRequest) cancelQuery(ss *SessionState, seq uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return false
	}
	r.attention = true
	r.cancel()
	if seq != 0 {
		ss.KillQuery(seq)
	}
	return true
}

func (r *tdsRequest) finish() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.done = true
	return r.attention
}

func (s *tdsServer) serve(socket net.Conn) {
	defer socket.Close()
	defer func() {
		if r := recover(); r != nil {
			PrintError(r)
		}
	}()
	conn, login, err := s.handshake(socket)
	if err != nil {
		return
	}
	c := &tdsConnection{conn: conn, server: s, database: login.database, initialDatabase: login.database, browseMetadata: login.browseMetadata}
	c.ss = RegisterSession(login.user, conn.RemoteAddr().String(), login.database)
	c.newSession(login.user)
	tdsConnections.Store(c, c.ss)
	defer func() {
		_ = conn.Close()
		c.prepared = nil
		c.preparedBytes = 0
		c.cursors = nil
		c.cursorBytes = 0
		defer tdsConnections.Delete(c)
		defer UnregisterSession(c.ss.ID)
		defer c.ss.ReleaseAllLocks()
		Apply(s.closeSession, c.session)
	}()
	requests := make(chan tdsMessage, 1)
	done := make(chan struct{})
	defer close(done)
	go func() {
		defer close(requests)
		for {
			message, err := readTDSMessage(conn)
			if err != nil {
				c.ss.Kill()
				return
			}
			if message.kind == 6 {
				if request := c.request.Load(); request != nil && request.cancelQuery(c.ss, c.seq.Load()) {
					continue
				}
			}
			select {
			case requests <- message:
			case <-done:
				return
			}
		}
	}()
	for request := range requests {
		if err := c.execute(request); err != nil {
			return
		}
	}
}

func (c *tdsConnection) newSession(user string) {
	c.session = NewSession()
	fn := c.session.Func()
	fn(NewString("username"), NewString(user))
	fn(NewString("schema"), NewString(c.database))
	c.sessionEvent("initialize", c.session, NewNil())
}

// Split semicolon-delimited batches while preserving strings, quoted names and
// nested comments. GO is a client utility command, not a server statement.
func splitTDSBatch(sql string) ([]string, error) {
	var statements []string
	start, depth := 0, 0
	quote := byte(0)
	lineComment := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if lineComment {
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if depth > 0 {
			if i+1 < len(sql) && sql[i:i+2] == "/*" {
				depth++
				i++
				continue
			}
			if i+1 < len(sql) && sql[i:i+2] == "*/" {
				depth--
				i++
			}
			continue
		}
		if quote != 0 {
			if ch == quote {
				if i+1 < len(sql) && sql[i+1] == quote {
					i++
				} else {
					quote = 0
				}
			}
			continue
		}
		if i+1 < len(sql) && sql[i:i+2] == "--" {
			lineComment = true
			i++
			continue
		}
		if i+1 < len(sql) && sql[i:i+2] == "/*" {
			depth = 1
			i++
			continue
		}
		if ch == '[' {
			quote = ']'
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ';' {
			if statement := strings.TrimSpace(sql[start:i]); statement != "" {
				statements = append(statements, statement)
			}
			start = i + 1
		}
	}
	if quote != 0 || depth != 0 {
		return nil, fmt.Errorf("unterminated T-SQL string, identifier or comment")
	}
	if statement := strings.TrimSpace(sql[start:]); statement != "" {
		statements = append(statements, statement)
	}
	return statements, nil
}

func (c *tdsConnection) execute(message tdsMessage) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	request := &tdsRequest{ctx: ctx, cancel: cancel}
	c.request.Store(request)
	defer func() { request.finish(); cancel(); c.request.CompareAndSwap(request, nil) }()
	response := &tdsResponse{w: c.conn, kind: 4, id: 1, packetSize: tdsPacketSize}
	rpc := message.kind == 3
	var preparedHandle int32
	var preparedName string
	preparedPublished := false
	defer func() {
		if preparedHandle != 0 && !preparedPublished {
			c.removePrepared(preparedHandle)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			var tokens bytes.Buffer
			if c.transaction != 0 && c.session.Func()(NewString("transaction")).IsNil() {
				tdsTransaction(&tokens, 10, c.transaction)
				c.transaction = 0
			}
			if request.finish() {
				tdsDone(&tokens, 0xfd, 0x20, 0)
			} else {
				PrintError(r)
				tdsError(&tokens, 50000, fmt.Sprint(r))
				if rpc {
					tokens.WriteByte(0x79)
					tdsU32(&tokens, 1)
					tdsDone(&tokens, 0xfe, 2, 0)
				} else {
					tdsDone(&tokens, 0xfd, 2, 0)
				}
			}
			if _, writeErr := response.Write(tokens.Bytes()); writeErr != nil {
				err = writeErr
			} else {
				err = response.flush(true)
			}
		}
	}()
	if message.status&0x10 != 0 {
		panic("RESETCONNECTIONSKIPTRAN is not supported")
	}
	if message.status&8 != 0 {
		Apply(c.server.closeSession, c.session)
		previous := c.database
		c.database = c.initialDatabase
		c.transaction = 0
		c.prepared = nil
		c.preparedBytes = 0
		c.cursors = nil
		c.cursorBytes = 0
		c.newSession(c.ss.User)
		c.ss.SetDB(c.database)
		var tokens bytes.Buffer
		tdsDatabase(&tokens, c.database, previous)
		tdsToken(&tokens, 0xe3, []byte{18, 0, 0})
		if _, err := response.Write(tokens.Bytes()); err != nil {
			return err
		}
	}
	// A batch or RPC is one identity scope. Session identity survives it and
	// transaction rollback; pooling reset replaces the entire session above.
	c.sessionEvent("begin", c.session, NewNil())
	var query string
	var params map[string]Scmer
	var declarations []tdsDeclaration
	var transactionKind byte
	switch message.kind {
	case 1:
		c.validateHeaders(message.data)
		data, decodeErr := tdsStripHeaders(message.data)
		if decodeErr != nil {
			panic(decodeErr)
		}
		query, decodeErr = tdsText(data)
		if decodeErr != nil {
			panic(decodeErr)
		}
	case 3:
		c.validateHeaders(message.data)
		rpc = true
		decoded, decodeErr := decodeTDSRPC(message.data)
		if decodeErr != nil {
			panic(decodeErr)
		}
		if decoded.metadata != "" {
			return c.metadataRPC(decoded, request, response)
		}
		if decoded.procedure == 2 || decoded.procedure == 7 || decoded.procedure == 8 || decoded.procedure == 9 {
			return c.cursorRPC(decoded, request, response)
		}
		query, params, preparedHandle, preparedName, declarations = c.prepareRPC(decoded, response)
	case 14:
		c.validateHeaders(message.data)
		data, decodeErr := tdsStripHeaders(message.data)
		if decodeErr != nil {
			panic(decodeErr)
		}
		d := tdsDecoder{data: data}
		switch d.u16() {
		case 5:
			isolation := d.u8()
			name := d.take(int(d.u8()))
			if (isolation != 0 && isolation != 2 && isolation != 5) || len(name) != 0 {
				panic("unsupported transaction isolation or name")
			}
			query, transactionKind = "BEGIN TRANSACTION", 8
		case 7, 8:
			kind := binary.LittleEndian.Uint16(data)
			name := d.take(int(d.u8()))
			flags := d.u8()
			if len(name) != 0 || flags&^byte(1) != 0 {
				panic("named transactions or reserved transaction flags are unsupported")
			}
			if kind == 7 {
				query, transactionKind = "COMMIT", 9
			} else {
				query, transactionKind = "ROLLBACK", 10
			}
			if flags&1 != 0 {
				isolation := d.u8()
				name := d.take(int(d.u8()))
				if (isolation != 0 && isolation != 2 && isolation != 5) || len(name) != 0 {
					panic("unsupported successor transaction isolation or name")
				}
				// Validate the complete request before completing the current
				// transaction. ODBC keeps autocommit off by requesting a new
				// transaction in the same manager request. The normal batch
				// lifecycle emits the old completion descriptor, then the new
				// begin descriptor, and stops before BEGIN if completion fails.
				query += ";BEGIN TRANSACTION"
			}
		default:
			panic("unsupported TDS transaction request")
		}
		if d.err != nil || len(d.data) != 0 {
			panic("malformed TDS transaction request")
		}
	case 6:
		var tokens bytes.Buffer
		tdsDone(&tokens, 0xfd, 0x20, 0)
		_, err = response.Write(tokens.Bytes())
		if err != nil {
			return err
		}
		return response.flush(true)
	default:
		panic(fmt.Sprintf("unsupported TDS packet type 0x%02x", message.kind))
	}
	statements, splitErr := splitTDSBatch(query)
	if splitErr != nil {
		panic(splitErr)
	}
	fn := c.session.Func()
	// Keep bindings outside the persistent session map: clearing to nil would
	// otherwise retain every distinct parameter name for the connection lifetime.
	querySession := c.parameterSession(c.session, params, declarations)
	for i, statement := range statements {
		if err := request.ctx.Err(); err != nil {
			panic(err)
		}
		atomic.AddInt64(&TotalHTTPRequests, 1)
		seq := c.ss.BeginQuery("Query", statement)
		ctx, cancel := context.WithCancel(request.ctx)
		c.ss.SetCancel(seq, cancel)
		c.ss.SetQueryContext(seq, ctx)
		c.seq.Store(seq)
		var result Scmer
		sink := &tdsResult{w: response}
		func() {
			defer cancel()
			defer c.seq.Store(0)
			defer c.ss.EndQuery(seq, "Sleep", "")
			result = Apply(c.server.query, NewString(c.database), NewString(statement), NewFunc(sink.row), NewFunc(sink.fields), querySession, NewAny(c.ss), NewInt(int64(seq)))
		}()
		sink.publish()
		var tokens bytes.Buffer
		previous := c.database
		c.database = fn(NewString("schema")).String()
		c.ss.SetDB(c.database)
		if previous != c.database {
			tdsDatabase(&tokens, c.database, previous)
		}
		active := !fn(NewString("transaction")).IsNil()
		if transactionKind == 8 || (active && c.transaction == 0) {
			c.transaction = nextTDSTransaction.Add(1)
			tdsTransaction(&tokens, 8, c.transaction)
		} else if c.transaction != 0 && !active {
			kind := transactionKind
			if kind == 0 {
				kind = 9
				if strings.HasPrefix(strings.ToUpper(statement), "ROLLBACK") {
					kind = 10
				}
			}
			tdsTransaction(&tokens, kind, c.transaction)
			c.transaction = 0
		}
		status := uint16(0)
		if i < len(statements)-1 || rpc {
			status |= 1
		}
		if active {
			status |= 4
		}
		rows := sink.count
		if sink.columns == nil && (result.IsInt() || result.IsFloat()) {
			rows = uint64(result.Int())
		}
		if !c.sessionEvent("done", c.session, NewInt(int64(rows))).Bool() {
			status |= 0x10
		}
		token := byte(0xfd)
		if rpc {
			token = 0xff
		}
		tdsDone(&tokens, token, status, rows)
		if _, err := response.Write(tokens.Bytes()); err != nil {
			return err
		}
	}
	var tokens bytes.Buffer
	cancelled := request.finish()
	if cancelled {
		tdsDone(&tokens, 0xfd, 0x20, 0)
	} else if rpc {
		tokens.WriteByte(0x79)
		tdsU32(&tokens, 0)
		if preparedHandle != 0 {
			tdsReturnHandle(&tokens, preparedName, preparedHandle)
		}
		tdsDone(&tokens, 0xfe, 0, 0)
	} else if len(statements) == 0 {
		tdsDone(&tokens, 0xfd, 0, 0)
	}
	if _, err := response.Write(tokens.Bytes()); err != nil {
		return err
	}
	err = response.flush(true)
	if err == nil && !cancelled {
		preparedPublished = true
	}
	return err
}

// All scan callbacks borrow one immutable RPC parameter map. Ordinary session
// operations keep their existing synchronization; input parameter keys are read-only.
func (c *tdsConnection) sessionEvent(event string, session, value Scmer) Scmer {
	if c.server.event.IsNil() {
		return NewNil()
	}
	return Apply(c.server.event, NewString(event), session, value)
}
func (c *tdsConnection) parameterSession(session Scmer, params map[string]Scmer, declarations []tdsDeclaration) Scmer {
	if c.server.bindSession.IsNil() {
		if len(params) != 0 || len(declarations) != 0 {
			panic("TDS parameter session callback is unavailable")
		}
		return session
	}
	values := make([]Scmer, 0, 2*len(params))
	for name, value := range params {
		values = append(values, NewString(name), value)
	}
	specs := make([]Scmer, 0, 2*len(declarations))
	for _, declaration := range declarations {
		specs = append(specs, NewString(declaration.name), declaration.spec)
	}
	bound := Apply(c.server.bindSession, session, NewSlice(values), NewSlice(specs))
	if !scmerCallable(bound) {
		panic("TDS parameter session recipe must return a procedure")
	}
	if bound.GetTag() == tagFunc {
		return bound
	}
	// Engine session handles use the native function ABI. The frontend may
	// return a Scheme or compiled procedure; adapt that immutable invocation
	// recipe here while leaving ordinary session dispatch unchanged.
	return NewFunc(func(args ...Scmer) Scmer { return Apply(bound, args...) })
}

func (c *tdsConnection) validateHeaders(data []byte) {
	if _, err := tdsStripHeaders(data); err != nil {
		panic(err)
	}
	end := int(binary.LittleEndian.Uint32(data))
	for offset := 4; offset < end; {
		if binary.LittleEndian.Uint16(data[offset+4:]) == 2 && binary.LittleEndian.Uint64(data[offset+6:]) != c.transaction {
			panic("TDS transaction descriptor does not match this connection")
		}
		offset += int(binary.LittleEndian.Uint32(data[offset:]))
	}
}
