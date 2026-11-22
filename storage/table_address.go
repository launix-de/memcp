/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/launix-de/memcp/scm"
)

const nativeTableAddressVersion = "native-table-v1"

// This is an inert address, not a retained handle or a visibility grant. The
// table constructor reattests its database, physical key, UUID and actual Tx.
func (t *table) nativeAddress(tx *TxContext) scm.Scmer {
	if t.ScopedSchema {
		return scm.NewNil()
	}
	if t.NativeIdentity(tx).IsNil() {
		return scm.NewNil()
	}
	return scm.NewSlice([]scm.Scmer{scm.NewString(nativeTableAddressVersion), scm.NewString(t.schema.incarnationID()), scm.NewString(t.RuntimeIdentity), scm.NewString(t.Name)})
}

func (db *database) resolveNativeAddress(address scm.Scmer, tx *TxContext) *table {
	if !address.IsSlice() || len(address.Slice()) != 4 {
		panic("invalid native table address")
	}
	fields := address.Slice()
	for _, v := range fields {
		if !v.IsString() {
			panic("invalid native table address field")
		}
	}
	if fields[0].String() != nativeTableAddressVersion {
		panic("unsupported native table address version")
	}
	if fields[1].String() != db.incarnationID() {
		panic("native table address belongs to a replaced database")
	}
	if id, err := uuid.Parse(fields[2].String()); err != nil || id == uuid.Nil {
		panic("invalid native table address generation")
	}
	t := db.tables.Get(fields[3].String())
	if t == nil || t.RuntimeIdentity != fields[2].String() || !t.visibleIn(tx, db.catalogView().version) {
		panic("native table address generation is not visible")
	}
	return t
}

func (t *table) nativeAddressCode() string {
	if t.RuntimeIdentity == "" || t.ScopedSchema || t.schema.tables.Get(t.Name) != t {
		return strconv.Quote(t.clusterPublishedName())
	}
	return fmt.Sprintf("(list %s %s %s %s)", strconv.Quote(nativeTableAddressVersion), strconv.Quote(t.schema.incarnationID()), strconv.Quote(t.RuntimeIdentity), strconv.Quote(t.Name))
}

// Code-only cold dependency discovery accepts constant list/quote constructors.
// It never evaluates a callback, reads a row, or converts an address to a name.
func staticNativeTableAddress(value scm.Scmer) (scm.Scmer, bool) {
	value = value.WithoutSourceInfo()
	if !value.IsSlice() {
		return scm.Scmer{}, false
	}
	parts := value.Slice()
	if len(parts) == 2 && callHeadIs(parts[0], "quote") {
		value = parts[1].WithoutSourceInfo()
		if !value.IsSlice() {
			return scm.Scmer{}, false
		}
		parts = value.Slice()
	}
	if len(parts) == 5 && callHeadIs(parts[0], "list") {
		parts = parts[1:]
		value = scm.NewSlice(parts)
	}
	if len(parts) != 4 {
		return scm.Scmer{}, false
	}
	for _, field := range parts {
		if !field.IsString() {
			return scm.Scmer{}, false
		}
	}
	if parts[0].String() != nativeTableAddressVersion {
		return scm.Scmer{}, false
	}
	return value, true
}

func (ref tableRef) sameSource(other tableRef) bool {
	if ref.schema != other.schema || ref.table != other.table {
		return false
	}
	if ref.identity != "" || other.identity != "" {
		return ref.identity == other.identity && ref.incarnation == other.incarnation
	}
	return true
}

func (ref scanJoinInfo) sourceRef() tableRef {
	return tableRef{schema: ref.schema, table: ref.table, incarnation: ref.incarnation, identity: ref.identity}
}

func (ref tableRef) resolve(tx *TxContext) *table {
	db := GetDatabase(ref.schema)
	if db == nil {
		return nil
	}
	if ref.identity == "" {
		return db.GetTableForTx(ref.table, false, tx)
	}
	db.GetRead(tx).Release()
	if db.incarnationID() != ref.incarnation {
		return nil
	}
	t := db.tables.Get(ref.table)
	if t == nil || t.RuntimeIdentity != ref.identity || !t.visibleIn(tx, db.catalogView().version) {
		return nil
	}
	return t
}

// Native generated maintenance code retains a versioned generation address,
// never a physical key sent back through the public SQL-name constructor.
func (t *table) nativeAddressExpression() scm.Scmer {
	value := scm.NewSlice([]scm.Scmer{scm.NewString(nativeTableAddressVersion), scm.NewString(t.schema.incarnationID()), scm.NewString(t.RuntimeIdentity), scm.NewString(t.Name)})
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), value})
}
