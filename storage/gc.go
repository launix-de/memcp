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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
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
// safe. Missing legacy manifests are reconstructed from committed column files;
// corrupt manifests and all ambiguous I/O failures remain a fail-closed no-op.
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
			if readErr, failed := reader.(ErrorReader); failed {
				reader.Close()
				if !readErr.Missing() {
					return nil, false
				}
				legacyReferences, ok := backfillBlobManifest(shard)
				if !ok {
					return nil, false
				}
				for hash := range legacyReferences {
					references[hash] = struct{}{}
				}
				continue
			}
			manifestReferences, ok := readBlobManifest(reader)
			if !ok {
				return nil, false
			}
			for hash := range manifestReferences {
				references[hash] = struct{}{}
			}
		}
	}
	return references, true
}

func readBlobManifest(reader io.ReadCloser) (map[string]struct{}, bool) {
	defer reader.Close()
	references := make(map[string]struct{})
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() || scanner.Text()+"\n" != blobManifestHeader || !scanner.Scan() {
		return nil, false
	}
	expectedChecksum, err := hex.DecodeString(scanner.Text())
	if err != nil || len(expectedChecksum) != sha256.Size {
		return nil, false
	}
	hasher := sha256.New()
	for scanner.Scan() {
		hash := strings.TrimSpace(scanner.Text())
		if decoded, err := hex.DecodeString(hash); err != nil || len(decoded) != sha256.Size {
			return nil, false
		}
		hasher.Write([]byte(hash + "\n"))
		references[hash] = struct{}{}
	}
	if scanner.Err() != nil || !bytes.Equal(hasher.Sum(nil), expectedChecksum) {
		return nil, false
	}
	return references, true
}

// backfillBlobManifest upgrades one legacy active generation without loading
// blob payloads or forcing a rebuild. It inspects only committed column files,
// deserializes reference-bearing OverlayBlob/compute-proxy columns, and then
// publishes the same checksummed manifest used by new generations.
func backfillBlobManifest(shard *storageShard) (references map[string]struct{}, ok bool) {
	if shard == nil || shard.t == nil || shard.t.schema == nil {
		return nil, false
	}
	references = make(map[string]struct{})
	for _, column := range shard.t.Columns {
		if column.IsTemp {
			continue
		}
		reader := shard.t.schema.persistence.ReadColumn(shard.uuid.String(), column.Name)
		if readErr, failed := reader.(ErrorReader); failed {
			reader.Close()
			if readErr.Missing() {
				// The ordinary column loader represents an absent committed column
				// as sparse NULLs, so it cannot own an external blob.
				continue
			}
			return nil, false
		}
		var magic uint8
		if err := binary.Read(reader, binary.LittleEndian, &magic); err != nil {
			reader.Close()
			return nil, false
		}
		if magic != 31 && magic != 50 {
			reader.Close()
			continue
		}
		storageType, known := storages[magic]
		if !known {
			reader.Close()
			return nil, false
		}
		storage := reflect.New(storageType).Interface().(ColumnStorage)
		count, decoded := deserializeReferenceStorage(storage, reader)
		reader.Close()
		if !decoded {
			return nil, false
		}
		appendColumnBlobReferences(storage, references, count)
	}
	if !tryWriteBlobManifestReferences(shard, references) {
		return nil, false
	}
	return references, true
}

func deserializeReferenceStorage(storage ColumnStorage, reader io.Reader) (count uint32, ok bool) {
	defer func() {
		if recover() != nil {
			count, ok = 0, false
		}
	}()
	decoded := storage.Deserialize(reader)
	if uint(uint32(decoded)) != decoded {
		return 0, false
	}
	return uint32(decoded), true
}

func tryWriteBlobManifestReferences(shard *storageShard, references map[string]struct{}) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	writeBlobManifestReferences(shard, references)
	return true
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
		appendColumnBlobReferences(column, references, shard.main_count)
	}
	writeBlobManifestReferences(shard, references)
}

// appendColumnBlobReferences follows persisted storage wrappers. This keeps
// manifests complete when a computed column compresses to OverlayBlob rather
// than exposing that overlay as the shard's top-level storage.
func appendColumnBlobReferences(storage ColumnStorage, references map[string]struct{}, count uint32) {
	switch typed := storage.(type) {
	case *OverlayBlob:
		typed.appendBlobReferences(references, count)
	case *StorageComputeProxy:
		typed.mu.RLock()
		if typed.main != nil {
			appendColumnBlobReferences(typed.main, references, count)
		}
		typed.mu.RUnlock()
	}
}

func writeBlobManifestReferences(shard *storageShard, references map[string]struct{}) {
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
