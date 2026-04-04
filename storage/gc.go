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
// No global lock needed: IncrBlobRefcount is always called BEFORE WriteBlob,
// so any file on disk has (or will have) a refcount entry.  Only truly
// orphaned files (from incomplete writes) are deleted.
// Queries the .blobs table per blob file instead of building an in-memory map.
func cleanBlobs(db *database) int {
	bt := db.GetTable(".blobs")
	if bt == nil {
		return 0
	}
	aggr := sumProc()

	// Walk disk blobs one by one; check refcount via scan on .blobs table.
	deleted := 0
	db.persistence.WalkBlobs(func(hash string) error {
		hashVal := scm.NewString(hash)
		// scan .blobs WHERE hash = <hash>, sum refcount
		rc := bt.scan(
			nil,
			[]string{"hash"}, blobCondition(hashVal),
			[]string{"refcount"},
			scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return a[0] }),
			aggr, scm.NewInt(0), aggr, false,
		)
		if scm.ToInt(rc) <= 0 {
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
