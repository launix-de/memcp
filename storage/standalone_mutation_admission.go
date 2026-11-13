/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"sync"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// A standalone mutation borrows the existing database mode authority. Its
// transaction resource owns the pin until the SAME native decision/finisher
// retires resources. This is not a second mode registry or catalog phase.
type standaloneMutationAdmission struct {
	database    *database
	transaction *transactionState
	mode        *databaseLocalDeclarationLease
	bytes       uint
	once        sync.Once
}

func (admission *standaloneMutationAdmission) Close() error {
	admission.once.Do(func() {
		_ = admission.mode.Close()
		admission.database.adjustMetadataMemory(admission.bytes, 0)
	})
	return nil
}

// Resolve only at native operator setup, before shard/schema/UNIQUE locks or
// callbacks/defaults. A captured row callback must BORROW that operation's
// lifetime; it cannot construct a fresh admission after the operation ended.
// Closing that callback lifetime is an independent native owner prerequisite.
func (tx *TxContext) admitStandaloneMutation(db *database) *standaloneMutationAdmission {
	if tx == nil || tx.transactionState == nil {
		panic("standalone mutation requires an active native transaction")
	}
	state := tx.transactionState
	key := "native-standalone-mutation/" + db.Name
	value := tx.getOrComputeResource(key, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		// The existing factory construction count excludes decision/reset until
		// this producer either publishes its value or unwinds every prefix.
		mode := db.admitLocalCatalog(false)
		accepted := false
		charged := false
		bytes := uint(unsafe.Sizeof(standaloneMutationAdmission{})) + uint(unsafe.Sizeof(*mode))
		// Existing resource/header/key and map/channel bookkeeping estimate;
		// readers consume only the database's producer-updated numeric scalar.
		bytes += uint(unsafe.Sizeof(transactionResource{})) + uint(len(key)) + 128
		defer func() {
			if !accepted {
				_ = mode.Close()
				if charged {
					db.adjustMetadataMemory(bytes, 0)
				}
			}
		}()
		admission := &standaloneMutationAdmission{database: db, transaction: state, mode: mode, bytes: bytes}
		db.adjustMetadataMemory(0, bytes)
		charged = true
		result := scm.NewAny(admission)
		accepted = true
		return result
	}))
	// A string key is lookup, never authority. Attest the actual native owner,
	// including an old same-name database or a different transaction resource.
	admission, ok := value.Any().(*standaloneMutationAdmission)
	if !ok || admission.database != db || admission.transaction != state || GetDatabase(db.Name) != db {
		panic("standalone mutation resource belongs to another native owner")
	}
	return admission
}
