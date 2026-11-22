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
		panic("native table address requires durable generation identity")
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
