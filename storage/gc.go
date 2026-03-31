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

import (
	"fmt"

	"github.com/launix-de/memcp/scm"
)

// CleanDatabase removes orphaned blob files and shard column/log files for db.
// Orphaned blobs are those on disk with no entry (or refcount=0) in the .blobs table.
// Orphaned shard files are those whose UUID-prefix is not in any active shard.
// Safe to call on a live, loaded database.
func CleanDatabase(db *database) (blobsDeleted, shardsDeleted int) {
	blobsDeleted = cleanBlobs(db)
	shardsDeleted = cleanShards(db)
	return
}

// cleanBlobs deletes blob files that are not referenced in the .blobs table.
// Runs under db.blobMu to serialize with IncrBlobRefcount/FlushBlobRefcounts,
// eliminating the race between WriteBlob and IncrBlobRefcount.
func cleanBlobs(db *database) int {
	db.blobMu.Lock()
	defer db.blobMu.Unlock()

	// Build set of active hashes from the .blobs table (small — unique blobs only).
	active := map[string]bool{}
	if bt := db.GetTable(".blobs"); bt != nil {
		bt.scan(
			[]string{},
			scm.NewProcStruct(scm.Proc{Body: scm.NewBool(true), En: &scm.Globalenv}),
			[]string{"hash", "refcount"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
				if scm.ToInt(a[1]) > 0 {
					active[scm.String(a[0])] = true
				}
				return scm.NewNil()
			}),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
			scm.NewNil(),
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[1] }),
			false, 0, nil,
		)
	}

	// Walk disk blobs one by one; delete those without an active refcount.
	deleted := 0
	db.persistence.WalkBlobs(func(hash string) error {
		if !active[hash] {
			db.persistence.DeleteBlob(hash)
			deleted++
		}
		return nil
	})
	return deleted
}

// cleanShards deletes shard column/log files whose UUID is not in any active shard.
// Reads the active UUID set from in-memory table structures (always current),
// under schemalock.RLock to prevent concurrent DDL from modifying the table map.
func cleanShards(db *database) int {
	// Collect active UUIDs from in-memory shards under read-lock.
	db.schemalock.RLock()
	activeUUIDs := map[string]bool{}
	for _, t := range db.tables.GetAll() {
		for _, s := range t.ActiveShards() {
			if s != nil {
				activeUUIDs[s.uuid.String()] = true
			}
		}
	}
	db.schemalock.RUnlock()

	// Walk disk shard files one by one; delete those with an unknown UUID.
	deleted := 0
	db.persistence.WalkShardFiles(func(name string) error {
		uuid := extractShardUUID(name)
		if uuid != "" && !activeUUIDs[uuid] {
			db.persistence.DeleteShardFile(name)
			deleted++
		}
		return nil
	})
	return deleted
}

// extractShardUUID returns the UUID prefix from a shard filename.
// Column files: "<uuid>-<colhash>" — the 37th character is '-'.
// Log files:    "<uuid>.log*"      — the 37th character is '.'.
// UUID is always exactly 36 chars (8-4-4-4-12 with hyphens).
// Returns "" if the name doesn't look like a shard file.
func extractShardUUID(name string) string {
	if len(name) <= 36 {
		return ""
	}
	switch name[36] {
	case '-', '.':
		return name[:36]
	}
	return ""
}

// Clean runs CleanDatabase on all loaded databases and returns a summary string.
func Clean() string {
	totalBlobs, totalShards := 0, 0
	for _, db := range databases.GetAll() {
		b, s := CleanDatabase(db)
		totalBlobs += b
		totalShards += s
	}
	return fmt.Sprintf("cleaned %d orphaned blobs, %d orphaned shard files", totalBlobs, totalShards)
}
