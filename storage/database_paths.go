/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Database identities share the local backend-configuration namespace even
// when their data live remotely. They are single native components, never paths.
// Do not URL-decode or normalize names: existing percent and quoted identifiers
// must keep their exact catalog and on-disk identities.
func validDatabaseName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.IsLocal(name) &&
		!strings.ContainsAny(name, "/\\") && strings.IndexFunc(name, unicode.IsControl) < 0
}
func requireDatabaseName(name string) {
	if !validDatabaseName(name) {
		panic("invalid database name: expected a nonempty local component without separators or control characters")
	}
}
func databaseFilesystemDirectory(base, name string) string {
	requireDatabaseName(name)
	if strings.EqualFold(name, "settings.json") {
		panic("database directory conflicts with reserved server settings")
	}
	path := filepath.Join(base, name)
	// Discovery does not follow database-directory symlinks either. A logical
	// CREATE must not turn a preexisting link into authority outside its root.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			panic("database directory must not be a symbolic link")
		}
		if !info.IsDir() {
			panic("database directory conflicts with an existing non-directory entry")
		}
	} else if err != nil && !os.IsNotExist(err) {
		raisePersistenceFailure("filesystem", name, "database.path", err)
	}
	return path + string(os.PathSeparator)
}
func databaseBackendConfigPath(name string) string {
	requireDatabaseName(name)
	if name == "settings" {
		// settings.json belongs to the server, never to this database. An
		// ordinary filesystem database named settings remains supported.
		return ""
	}
	return filepath.Join(Basepath, name+".json")
}
func removeDatabaseBackendConfig(name string) {
	if path := databaseBackendConfigPath(name); path != "" {
		os.Remove(path)
	}
}
func defaultFilesystemBackendConfig(raw json.RawMessage) bool {
	var config map[string]json.RawMessage
	if json.Unmarshal(raw, &config) != nil || len(config) > 1 {
		return false
	}
	var backend string
	return (len(config) == 0 || json.Unmarshal(config["backend"], &backend) == nil) &&
		(backend == "" || backend == "filesystem")
}
