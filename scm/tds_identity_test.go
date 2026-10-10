/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"testing"
)

func TestTDSIdentityBatchAndRPCScopes(t *testing.T) {
	recorded := &tdsRecordingConnection{}
	server := &tdsServer{closeSession: NewFunc(func(a ...Scmer) Scmer { return NewBool(true) })}
	tdsInstallTestFrontend(server)
	var scopes []bool
	server.query = NewFunc(func(a ...Scmer) Scmer {
		session := a[4].Func()
		switch a[1].String() {
		case "ALLOCATE":
			session(NewString("tsql_scope_identity"), NewInt(9))
			session(NewString("tsql_last_identity"), NewInt(9))
		case "CHECK":
			scopes = append(scopes, !session(NewString("tsql_scope_identity")).IsNil())
			if session(NewString("tsql_last_identity")).Int() != 9 {
				panic("session identity did not survive batch boundary")
			}
		}
		return NewInt(0)
	})
	connection := &tdsConnection{server: server, conn: recorded, database: "fixture", initialDatabase: "fixture", ss: RegisterSession("alice", "fixture", "fixture")}
	defer UnregisterSession(connection.ss.ID)
	connection.newSession("alice")
	for _, message := range []tdsMessage{
		{kind: 1, data: append(tdsRawRPC(10)[:22], tdsUTF16("ALLOCATE; CHECK")...)},
		{kind: 1, data: append(tdsRawRPC(10)[:22], tdsUTF16("CHECK")...)},
		{kind: 3, data: tdsRawRPC(10, tdsRPCParameter{"", 0, NewString("ALLOCATE; CHECK")})},
		{kind: 3, data: tdsRawRPC(10, tdsRPCParameter{"", 0, NewString("CHECK")})},
	} {
		recorded.Reset()
		if err := connection.execute(message); err != nil {
			t.Fatal(err)
		}
		response, err := readTDSMessage(bytes.NewReader(recorded.Bytes()))
		if err != nil || bytes.Contains(response.data, []byte{0xaa}) {
			t.Fatal("identity scope execution failed", err, response.data)
		}
	}
	if len(scopes) != 4 || !scopes[0] || scopes[1] || !scopes[2] || scopes[3] {
		t.Fatal("batch/RPC scope leaked or reset between statements", scopes)
	}
	connection.newSession("alice")
	if !connection.session.Func()(NewString("tsql_last_identity")).IsNil() {
		t.Fatal("pooling reset retained identity")
	}
}

func TestTDSIdentityMetadataFlag(t *testing.T) {
	var encoded bytes.Buffer
	result := &tdsResult{w: &encoded}
	result.fields(NewSlice([]Scmer{NewString("id")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("BIGINT"), NewString("identity"), NewBool(true)})}))
	result.publish()
	data := encoded.Bytes()
	if len(data) < 10 || data[7]&0x10 == 0 {
		t.Fatal("auto identity was not advertised in column flags", data)
	}
}
