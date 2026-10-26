/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "fmt"
import "strings"
import "sync/atomic"
import "github.com/launix-de/memcp/storage/mosi"

// The first cluster execution milestone mounts explicitly named immutable
// generations on shared storage. Every participant has S; none can grant M.
// This is not a substitute directory/lock protocol for live cluster writers.
// Write handover and mutable schema/GC publication will use the same MOSI core
// once connected; until then fail before modifying a shared namespace.
var sharedClusterMounts atomic.Pointer[map[string]bool]

func PrepareNodeCluster(args []string) error {
	file, names := "", ""
	for i := 0; i < len(args); i++ {
		for _, entry := range []struct {
			flag  string
			value *string
		}{{"--cluster-secret-file", &file}, {"--cluster-databases", &names}} {
			if strings.HasPrefix(args[i], entry.flag+"=") {
				*entry.value = strings.TrimPrefix(args[i], entry.flag+"=")
			} else if args[i] == entry.flag && i+1 < len(args) {
				i++
				*entry.value = args[i]
			}
		}
	}
	if file == "" {
		return nil
	}
	return prepareSharedClusterMounts(names)
}
func prepareSharedClusterMounts(names string) error {
	mounted := make(map[string]bool)
	for _, name := range strings.Split(names, ",") {
		if !validReplicaDatabase(name) {
			return fmt.Errorf("cluster-databases requires explicit shared database names")
		}
		mounted[name] = true
	}
	sharedClusterMounts.Store(&mounted)
	return nil
}
func sharedClusterDatabase(name string) bool {
	if mounts := sharedClusterMounts.Load(); mounts != nil {
		return (*mounts)[name]
	}
	return false
}

// Synthetic/unpublished table descriptors may not have a database. The
// optional cluster adapter must leave these native engine paths untouched.
func sharedClusterSource(table *table) bool {
	return table != nil && table.schema != nil && !table.isEphemeralQueryTable() && sharedClusterDatabase(table.schema.Name)
}

// Every schema/shard now acquires actual network MOSI permission. The
// immutable mount adapter has no authority to fabricate S from a backend file.
func acquireClusterSnapshot(resource mosiResource) *replicaCell {
	cluster := activeNodeCluster()
	if cluster == nil || !cluster.databases[resource.Database] {
		panic(mosi.NetworkError)
	}
	return cluster.manager.obtain(resource, mosi.S, "")
}
