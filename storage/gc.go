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
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

const blobManifestColumn = ".blobrefs-v1"

const blobManifestHeader = "memcp-blob-references-v1\n"

// CleanDatabase removes only disk objects proven unowned by the complete active
// generation. It serializes against rebuild/repartition publication, but does
// not stop ordinary queries or DML.
func CleanDatabase(db *database) (blobsDeleted, shardsDeleted int) {
	// Startup cleanup runs immediately after the lazy database catalog is
	// discovered. An unloaded schema is not an empty schema: load its committed
	// topology before proving that any disk object is unowned.
	db.ensureLoaded()
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	blobsDeleted = cleanBlobs(db)
	shardsDeleted = cleanShards(db)
	return
}

// cleanBlobs deletes a blob only when every active persistent shard generation
// has supplied a valid ownership manifest. The refcount table is operational
// metadata and can lag after a crash, so it is never proof that deletion is
// safe. Legacy or incomplete generations make cleanup a conservative no-op.
func cleanBlobs(db *database) int {
	references, complete := activeBlobReferences(db)
	if !complete {
		return 0
	}

	deleted := 0
	db.persistence.WalkBlobs(func(hash string) error {
		defer db.lockBlobRef(hash)()
		if _, live := references[hash]; !live {
			db.persistence.DeleteBlob(hash)
			deleted++
		}
		return nil
	})
	return deleted
}

func activeBlobReferences(db *database) (map[string]struct{}, bool) {
	db.schemalock.RLock()
	defer db.schemalock.RUnlock()
	references := make(map[string]struct{})
	for _, table := range db.tables.GetAll() {
		if table.Name == ".blobs" || table.PersistencyMode == Memory || table.PersistencyMode == Cache {
			continue
		}
		for _, shard := range table.ActiveShards() {
			if shard == nil {
				continue
			}
			reader := db.persistence.ReadColumn(shard.uuid.String(), blobManifestColumn)
			if _, failed := reader.(ErrorReader); failed {
				reader.Close()
				return nil, false
			}
			scanner := bufio.NewScanner(reader)
			if !scanner.Scan() || scanner.Text()+"\n" != blobManifestHeader {
				reader.Close()
				return nil, false
			}
			if !scanner.Scan() {
				reader.Close()
				return nil, false
			}
			expectedChecksum, err := hex.DecodeString(scanner.Text())
			if err != nil || len(expectedChecksum) != sha256.Size {
				reader.Close()
				return nil, false
			}
			hasher := sha256.New()
			for scanner.Scan() {
				hash := strings.TrimSpace(scanner.Text())
				if decoded, err := hex.DecodeString(hash); err != nil || len(decoded) != sha256.Size {
					reader.Close()
					return nil, false
				}
				hasher.Write([]byte(hash + "\n"))
				references[hash] = struct{}{}
			}
			if err := scanner.Err(); err != nil {
				reader.Close()
				return nil, false
			}
			reader.Close()
			if !bytes.Equal(hasher.Sum(nil), expectedChecksum) {
				return nil, false
			}
		}
	}
	return references, true
}

func writeBlobManifest(shard *storageShard) {
	if shard == nil || shard.t == nil || shard.t.schema == nil || shard.uuid == uuid.Nil {
		return
	}
	if shard.t.PersistencyMode == Memory || shard.t.PersistencyMode == Cache || shard.t.Name == ".blobs" {
		return
	}
	references := make(map[string]struct{})
	for _, column := range shard.columns {
		if blobs, ok := column.(*OverlayBlob); ok {
			blobs.appendBlobReferences(references, shard.main_count)
		}
	}
	hashes := make([]string, 0, len(references))
	for hash := range references {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	body := ""
	if len(hashes) > 0 {
		body = strings.Join(hashes, "\n") + "\n"
	}
	checksum := sha256.Sum256([]byte(body))
	writer := shard.t.schema.persistence.WriteColumn(shard.uuid.String(), blobManifestColumn)
	manifest := blobManifestHeader + hex.EncodeToString(checksum[:]) + "\n" + body
	if _, err := writer.Write([]byte(manifest)); err != nil {
		_ = writer.Close()
		panic(err)
	}
	finishColumnWrite(writer, shard.t.PersistencyMode == Safe)
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
