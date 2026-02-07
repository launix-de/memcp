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
	IncrBlobRefcount(hash string)
	DecrBlobRefcount(hash string)                                  // deletes blob when RC <= 0
	FlushBlobRefcounts()                                           // persist refcount changes to disk
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
	idx uint
}
type LogEntryInsert struct {
	cols   []string
	values [][]scm.Scmer
}

// for CREATE TABLE
type PersistenceFactory interface {
	CreateDatabase(schema string) PersistenceEngine
}

// BackendConfig describes the configuration for a remote storage backend.
// These are stored as JSON files in the data folder (e.g. data/mydb.json).
type BackendConfig struct {
	Backend string `json:"backend"` // "ceph", "s3", etc.

	// Ceph-specific fields
	UserName    string `json:"username,omitempty"`  // Ceph: e.g. "client.admin"
	ClusterName string `json:"cluster,omitempty"`   // Ceph: often "ceph"
	ConfFile    string `json:"conf_file,omitempty"` // Ceph: optional config path
	Pool        string `json:"pool,omitempty"`      // Ceph: e.g. "memcp"
	Prefix      string `json:"prefix,omitempty"`    // Object prefix (Ceph and S3)

	// S3-specific fields
	AccessKeyID     string `json:"access_key_id,omitempty"`     // S3: AWS or S3-compatible access key
	SecretAccessKey string `json:"secret_access_key,omitempty"` // S3: AWS or S3-compatible secret key
	Region          string `json:"region,omitempty"`            // S3: AWS region (e.g., "us-east-1")
	Endpoint        string `json:"endpoint,omitempty"`          // S3: Custom endpoint (MinIO, etc.)
	Bucket          string `json:"bucket,omitempty"`            // S3: Bucket name
	ForcePathStyle  bool   `json:"force_path_style,omitempty"`  // S3: Use path-style URLs (for MinIO)
}

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
