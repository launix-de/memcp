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
	WriteSchema(schema []byte)
	ReadColumn(shard string, column string) io.ReadCloser
	WriteColumn(shard string, column string) io.WriteCloser
	RemoveColumn(shard string, column string)
	ReadBlob(hash string) io.ReadCloser
	WriteBlob(hash string) io.WriteCloser
	DeleteBlob(hash string)
	OpenLog(shard string) PersistenceLogfile                       // open for writing
	ReplayLog(shard string) (chan interface{}, PersistenceLogfile) // replay existing log
	RemoveLog(shard string)
	Remove() // delete from storage
}

type PersistenceLogfile interface {
	Write(logentry interface{})
	Sync()
	Close()
}
type LogEntryDelete struct {
	idx uint32
}
type LogEntryInsert struct {
	cols   []string
	values [][]scm.Scmer
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
	// TODO: read schema.json
	// TODO: for each shard: read columns, read log, transfer to dst
}

// ErrorReader implements io.ReadCloser
type ErrorReader struct {
	e error
}

func (e ErrorReader) Read([]byte) (int, error) {
	// reflects the error (e.g. file not found)
	return 0, e.e
}
func (e ErrorReader) Close() error {
	// closes without problem
	return nil
}
