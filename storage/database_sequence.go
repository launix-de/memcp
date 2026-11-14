/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sort"
import "math"
import "strings"
import "context"
import "encoding/json"
import "crypto/sha256"
import "encoding/base64"
import "unicode/utf8"
import "github.com/launix-de/memcp/scm"

// Sequence highwaters are database authority, not row mutations or transaction
// decisions. Their versioned frames survive every decision-log checkpoint.
// Earlier row/commit formats retain their existing readers and encodings.
type LogEntrySequence struct {
	name string
	high int64
}

type sequenceLogRecord struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	High    int64  `json:"high"`
}

func validSequenceName(name string) bool {
	return name != "" && len(name) <= 1024 && utf8.ValidString(name) && !strings.ContainsRune(name, 0)
}

func encodeSequenceRecord(entry LogEntrySequence) string {
	if !validSequenceName(entry.name) || entry.high <= 0 {
		panic("invalid native sequence record")
	}
	raw, err := json.Marshal(sequenceLogRecord{Version: 1, Name: entry.name, High: entry.high})
	if err != nil {
		panic(err)
	}
	checksum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + fmt.Sprintf("%x", checksum[:])
}

func decodeSequenceRecord(encoded string) LogEntrySequence {
	payload, checksum, found := strings.Cut(encoded, ".")
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if !found || err != nil || strings.Contains(checksum, ".") {
		panic("invalid native sequence frame")
	}
	actual := sha256.Sum256(raw)
	if checksum != fmt.Sprintf("%x", actual[:]) {
		panic("corrupt native sequence frame")
	}
	var record sequenceLogRecord
	if err := json.Unmarshal(raw, &record); err != nil || record.Version != 1 || !validSequenceName(record.Name) || record.High <= 0 {
		panic("invalid native sequence record")
	}
	return LogEntrySequence{name: record.Name, high: record.High}
}

// All callers own transactionMu, or the unpublished transactionOnce producer.
func (db *database) acceptDatabaseWALFrame(entry interface{}, committed map[string]uint64, generation uint64) error {
	switch record := entry.(type) {
	case LogEntryCommit:
		if record.txID != "" {
			return db.acceptNativeCommitFrame(record.txID, committed, generation)
		}
	case LogEntrySequence:
		if validSequenceName(record.name) && record.high > 0 {
			if db.sequences == nil {
				db.sequences = make(map[string]int64)
			}
			if record.high > db.sequences[record.name] {
				db.sequences[record.name] = record.high
			}
			return nil
		}
	}
	return fmt.Errorf("invalid entry in database authority log")
}

func (db *database) sequenceLogEntries() []interface{} {
	names := make([]string, 0, len(db.sequences))
	for name := range db.sequences {
		names = append(names, name)
	}
	sort.Strings(names)
	entries := make([]interface{}, 0, len(names))
	for _, name := range names {
		entries = append(entries, LogEntrySequence{name: name, high: db.sequences[name]})
	}
	return entries
}

func sequenceDatabase(database, name scm.Scmer, transaction scm.Scmer) (*database, *TxContext) {
	if !database.IsString() || !name.IsString() || !validSequenceName(name.String()) {
		panic("sequence requires a database and a nonempty UTF-8 name of at most 1024 bytes")
	}
	tx := scmerToTxContext(transaction)
	if !transaction.IsNil() && tx == nil {
		panic("sequence requires a native transaction capability or nil")
	}
	if err := scanQueryContext(tx, context.Background()).Err(); err != nil {
		panic(err)
	}
	db := GetDatabase(database.String())
	if db == nil {
		panic("sequence database does not exist")
	}
	db.requireLocalSequences()
	return db, tx
}

func (db *database) requireLocalSequences() {
	if sharedClusterDatabase(db.Name) || activeReplication(db) != nil {
		panic("native sequences do not yet support distributed databases")
	}
	if GetDatabase(db.Name) != db {
		panic("sequence database was retired")
	}
}

func (db *database) reserveSequence(name string, count int64, tx *TxContext) (int64, int64) {
	// Sequence authority does not mutate schema. The lifecycle lease excludes
	// backend movement/removal without taking catalogPublicationMu. A scalar
	// caller can own a shard lock, so never wait for lifecycle maintenance which
	// may itself need that shard. The transaction supplies admission/cancellation
	// only; rollback never rewinds reservations.
	defer db.GetRead(tx).Release()
	if !db.persistenceLifecycle.TryRLock() {
		panic("database maintenance in progress; retry sequence operation")
	}
	defer db.persistenceLifecycle.RUnlock()
	db.requireLocalSequences()
	db.initializeTransactionLog()
	db.transactionMu.Lock()
	defer db.transactionMu.Unlock()
	if db.sequenceFailure != nil {
		panic(db.sequenceFailure)
	}
	previous := db.sequences[name]
	if count <= 0 || previous > math.MaxInt64-count {
		panic("sequence reservation exceeds the positive signed 64-bit range")
	}
	last := previous + count
	if db.sequences == nil {
		db.sequences = make(map[string]int64)
	}
	// A failed append/barrier can have reached durable storage. Consume the
	// proposed range in memory and fail closed until restart; never return it.
	db.sequences[name] = last
	if err := persistenceCall(func() {
		db.transactionLog.Write(LogEntrySequence{name: name, high: last})
		db.transactionLog.Flush(true)
	}); err != nil {
		db.sequenceFailure = err
		panic(err)
	}
	return previous + 1, last
}

func initDatabaseSequences(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{
		Name: "sequence_reserve",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) != 4 || !a[2].IsInt() || a[2].Int() <= 0 {
				panic("sequence_reserve expects database, name, positive integer count and native transaction")
			}
			db, tx := sequenceDatabase(a[0], a[1], a[3])
			first, last := db.reserveSequence(a[1].String(), a[2].Int(), tx)
			return scm.NewSlice([]scm.Scmer{scm.NewInt(first), scm.NewInt(last)})
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "Durably reserves an inclusive positive signed-64-bit interval from one named database sequence. Reservations survive rollback, savepoints and restart; gaps are allowed. Distributed databases are explicitly unsupported.",
			Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "database"}, {Kind: "string", Label: "name"}, {Kind: "int", Label: "count"}, {Kind: "any", Label: "transaction"}},
			Return:      &scm.TypeDescriptor{Kind: "list"}},
	})
	scm.Declare(en, &scm.Declaration{
		Name: "sequence_current",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			if len(a) != 3 {
				panic("sequence_current expects database, name and native transaction")
			}
			db, tx := sequenceDatabase(a[0], a[1], a[2])
			defer db.GetRead(tx).Release()
			if !db.persistenceLifecycle.TryRLock() {
				panic("database maintenance in progress; retry sequence operation")
			}
			defer db.persistenceLifecycle.RUnlock()
			db.requireLocalSequences()
			db.transactionMu.RLock()
			defer db.transactionMu.RUnlock()
			if db.sequenceFailure != nil {
				panic(db.sequenceFailure)
			}
			return scm.NewInt(db.sequences[a[1].String()])
		},
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "Reads a named database sequence's highwater, initially zero, without reserving values. Distributed databases are explicitly unsupported.",
			Params:      []*scm.TypeDescriptor{{Kind: "string", Label: "database"}, {Kind: "string", Label: "name"}, {Kind: "any", Label: "transaction"}},
			Return:      &scm.TypeDescriptor{Kind: "int"}},
	})
}
