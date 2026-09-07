/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "io"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "strings"
import "github.com/google/uuid"

/*

persistence interface

MemCP allows multiple persistence interfaces for storage devices:
 - file system: in data/[dbname]
 - all other: in data/[dbname.json]

A storage interface must implement the following operations:
 - load schema.json
 - load a column
 - load all log entries
 - remove a shard (columns and log)
 - persist schema.json
 - persist a column (shard id, column name)
 - persist a log entry

*/

type PersistenceEngine interface {
	ReadSchema() []byte
	// WriteSchema must publish one complete schema generation atomically. A
	// reader may observe either the previous or the new generation, never a
	// missing, empty, or partial schema. Every backend I/O failure must panic
	// with *PersistenceFailure; absence is represented explicitly by a reader
	// whose Missing method returns true.
	WriteSchema(schema []byte)
	ReadColumn(shard string, column string) io.ReadCloser
	// WriteColumn returns a generation-private writer. Close must either publish
	// the complete column object or report an error; partial objects must never
	// become readable as a successfully completed rebuild generation.
	WriteColumn(shard string, column string) io.WriteCloser
	RemoveColumn(shard string, column string)
	ReadBlob(hash string) io.ReadCloser
	WriteBlob(hash string) io.WriteCloser
	DeleteBlob(hash string)
	// WalkBlobs calls fn for every blob hash stored on disk, one at a time.
	// Backend failures follow the common persistence contract and panic with
	// *PersistenceFailure.
	WalkBlobs(fn func(hash string))
	OpenLog(shard string) PersistenceLogfile // open for writing
	// SwapLog atomically replaces the visible log with entries and returns an
	// appendable handle for the replacement. A crash may leave private/orphaned
	// temporary storage, but ReplayLog must observe either the complete old log
	// or the complete replacement.
	SwapLog(shard string, entries []interface{}, durable bool) PersistenceLogfile
	// ReplayLog returns the transaction IDs committed in this WAL before it
	// streams entries. This keeps recovery memory proportional to commit count,
	// rather than retaining every row mutation until the final marker is known.
	ReplayLog(shard string) (map[string]struct{}, chan interface{}, PersistenceLogfile)
	RemoveLog(shard string)
	// WalkShardFiles calls fn for every shard-related file (column files, log files)
	// stored on disk, one at a time. The name passed to fn is exactly the value
	// expected by ReadShardFile, WriteShardFile, and DeleteShardFile. Backend
	// failures panic with *PersistenceFailure.
	WalkShardFiles(fn func(name string))
	ReadShardFile(name string) io.ReadCloser
	WriteShardFile(name string) io.WriteCloser
	// DeleteShardFile deletes a file previously yielded by WalkShardFiles.
	DeleteShardFile(name string)
	Remove()             // delete from storage
	BackendName() string // returns the backend type name (e.g. "filesystem", "s3", "ceph")
	// StorageIdentity names the physical namespace without exposing credentials.
	StorageIdentity() string
}

type PersistenceLogfile interface {
	Write(logentry interface{})
	// Flush publishes every accepted frame. durable additionally requests the
	// backend's strongest durability barrier (fsync for local files). Remote
	// backends must transmit buffered writes even when durable is false.
	Flush(durable bool)
	Close()
}

func finishColumnWrite(w io.WriteCloser, durable bool) {
	// Rebuild publication order is column data first, schema reference second.
	// SAFE columns must reach stable storage before schema.json can name their
	// shard generation. Close errors are publication failures too; callers must
	// retain the previously committed shard generation on every panic here.
	if durable {
		if syncer, ok := w.(interface{ Sync() error }); ok {
			if err := syncer.Sync(); err != nil {
				_ = w.Close()
				raisePersistenceFailure("unknown", "unknown", "column.write.sync", err)
			}
		}
	}
	if err := w.Close(); err != nil {
		raisePersistenceFailure("unknown", "unknown", "column.write.close", err)
	}
}

type LogEntryDelete struct {
	idx  uint32
	txID string
}
type LogEntryUndelete struct {
	idx  uint32
	txID string
}
type LogEntryInsert struct {
	cols   []string
	values [][]scm.Scmer
	txID   string
}
type LogEntryInsertHidden struct {
	cols   []string
	values [][]scm.Scmer
	txID   string
}
type LogEntryCommit struct {
	txID string
}

// for CREATE TABLE
type PersistenceFactory interface {
	CreateDatabase(schema string) PersistenceEngine
}

// BackendRegistry maps backend names (e.g. "ceph", "s3") to factory functions.
// Each backend registers itself here (typically via init()). The factory
// receives the database name and the raw JSON config and returns a
// PersistenceEngine ready for use.
var BackendRegistry = map[string]func(dbName string, raw json.RawMessage) PersistenceEngine{}

// Helper function to move databases between storages
func MoveDatabase(src PersistenceEngine, dst PersistenceEngine) {
	schema := src.ReadSchema()
	if len(schema) == 0 {
		panic("cannot move a database without a published schema")
	}

	src.WalkShardFiles(func(name string) {
		// WAL encodings are backend-specific (filesystem uses one appendable
		// file, remote stores use immutable segments plus a manifest). ALTER
		// DATABASE compacts them before MoveDatabase and opens native empty WALs.
		if isPersistenceLogObject(name) {
			return
		}
		copyPersistenceObject(src.ReadShardFile(name), dst.WriteShardFile(name), dst, "database.move.file")
	})
	src.WalkBlobs(func(hash string) {
		copyPersistenceObject(src.ReadBlob(hash), dst.WriteBlob(hash), dst, "database.move.blob")
	})

	// The schema is the commit record naming the copied shard generations. It
	// must be the final destination object so an interrupted move is never
	// mistaken for a complete database.
	dst.WriteSchema(schema)
}

func isPersistenceLogObject(name string) bool {
	logAt := strings.LastIndex(name, ".log")
	if logAt < 0 {
		return false
	}
	owner := name[:logAt]
	if owner != transactionLogName {
		if _, err := uuid.Parse(owner); err != nil {
			return false
		}
	}
	suffix := name[logAt+len(".log"):]
	if suffix == "" || suffix == ".manifest" {
		return true
	}
	if len(suffix) != 9 || suffix[0] != '.' {
		return false
	}
	for _, digit := range suffix[1:] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func copyPersistenceObject(reader io.ReadCloser, writer io.WriteCloser, dst PersistenceEngine, operation string) {
	if _, err := io.Copy(writer, reader); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		raisePersistenceFailure(dst.BackendName(), "database move", operation, err)
	}
	if err := reader.Close(); err != nil {
		_ = writer.Close()
		raisePersistenceFailure(dst.BackendName(), "database move", operation+".read.close", err)
	}
	if err := writer.Close(); err != nil {
		raisePersistenceFailure(dst.BackendName(), "database move", operation+".write.close", err)
	}
}

// ErrorReader implements io.ReadCloser
type ErrorReader struct {
	e        error
	notFound bool
}

// Missing reports whether the backend proved that the requested object does
// not exist. Callers must not equate arbitrary read failures with absence:
// doing so could turn a transient backend outage into destructive cleanup.
func (e ErrorReader) Missing() bool { return e.notFound }

func (e ErrorReader) Read([]byte) (int, error) {
	// reflects the error (e.g. file not found)
	return 0, e.e
}
func (e ErrorReader) Close() error {
	// closes without problem
	return nil
}
