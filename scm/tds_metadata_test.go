/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"database/sql"
	"testing"
)

func TestTDSMetadataRPCPreservesOpaqueNamesAndValues(t *testing.T) {
	for _, name := range []string{"sp_tables", "[fixture]..[sp_tables]", "[fi]]xture].sys.sp_tables", "sp_custom", "sp_tables;drop"} {
		var wire bytes.Buffer
		wire.Write(tdsRawRPC(10)[:22])
		tdsU16(&wire, uint16(len(tdsUTF16(name))/2))
		wire.Write(tdsUTF16(name))
		tdsU16(&wire, 0)
		rpc, err := decodeTDSRPC(wire.Bytes())
		if err != nil || rpc.metadata != name {
			t.Fatal("wire adapter interpreted a frontend procedure name", name, err)
		}
	}
	parameters := tdsMetadataParameters([]tdsRPCParameter{{"@Table_Name", 0, NewString("'; DROP TABLE fixture; --")}, {"", 0, NewInt(3)}}).Slice()
	if parameters[0].String() != "@Table_Name" || parameters[1].String() != "'; DROP TABLE fixture; --" || parameters[2].String() != "1" || parameters[3].Int() != 3 {
		t.Fatal("opaque metadata parameter changed", parameters)
	}
	expectTDSPanic(t, func() { tdsMetadataParameters([]tdsRPCParameter{{"@x", 0, NewNil()}, {"@x", 0, NewNil()}}) })
	parameters = tdsMetadataParameters([]tdsRPCParameter{{"@x", 0, NewNil()}, {"@X", 0, NewNil()}}).Slice()
	if len(parameters) != 4 {
		t.Fatal("wire adapter applied frontend casing policy")
	}
}

func TestTDSMetadataRPCReleasedClient(t *testing.T) {
	db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
		server.metadata = NewFunc(func(a ...Scmer) Scmer {
			if len(a) != 8 || a[0].String() != "fixture" || a[1].String() != "sp_tables" {
				panic("invalid metadata callback contract")
			}
			parameters := a[2].Slice()
			if len(parameters) != 2 || parameters[0].String() != "@table_name" || parameters[1].String() != "quoted';value" {
				panic("metadata parameters corrupted")
			}
			a[4].Func()(NewSlice([]Scmer{NewString("TABLE_NAME")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("NVARCHAR"), NewString("size"), NewInt(128)})}))
			a[3].Func()(NewSlice([]Scmer{NewString("TABLE_NAME"), NewString("quoted';value")}))
			return NewInt(1)
		})
	})
	var value string
	if err := db.QueryRow("[fixture]..sp_tables", sql.Named("table_name", "quoted';value")).Scan(&value); err != nil || value != "quoted';value" {
		t.Fatal(value, err)
	}
	if err := db.QueryRow("[another]..sp_tables", sql.Named("table_name", "quoted';value")).Scan(&value); err == nil {
		t.Fatal("cross-database procedure accepted")
	}
	var integer int64
	if err := db.QueryRow("SELECT 1").Scan(&integer); err != nil || integer != 1 {
		t.Fatal("metadata error desynchronized connection", integer, err)
	}
}

func TestTDSKeyMetadataRPCReleasedClient(t *testing.T) {
	for _, procedure := range []string{"sp_pkeys", "sp_statistics", "sp_special_columns", "sp_fkeys"} {
		t.Run(procedure, func(t *testing.T) {
			db, _ := startTDSTestServer(t, false, func(server *tdsServer) {
				server.metadata = NewFunc(func(a ...Scmer) Scmer {
					if a[1].String() != procedure {
						panic("incorrect catalog procedure dispatch")
					}
					parameters := a[2].Slice()
					if len(parameters) != 2 || parameters[0].String() != "@table_name" || parameters[1].String() != "fixture_table" {
						panic("catalog parameters changed")
					}
					a[4].Func()(NewSlice([]Scmer{NewString("KEY_COLUMN")}), NewSlice([]Scmer{tdsTestDescription([]Scmer{NewString("sql_type"), NewString("NVARCHAR"), NewString("size"), NewInt(128)})}))
					a[3].Func()(NewSlice([]Scmer{NewString("KEY_COLUMN"), NewString("id")}))
					return NewInt(1)
				})
			})
			var value string
			if err := db.QueryRow(procedure, sql.Named("table_name", "fixture_table")).Scan(&value); err != nil || value != "id" {
				t.Fatal(value, err)
			}
		})
	}
}
