/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "maps"
import "sort"
import "context"
import "math"
import "encoding/json"
import "crypto/sha256"
import "github.com/launix-de/memcp/storage/mosi"

var globalResource = mosiResource{Object: "global"}

const clusterIDBlockSize = uint64(10000)
const clusterGlobalLog = ".cluster-global-v1"

// This is ordinary persistent data, not saved MOSI authority. The same native
// cell/writer/handover implementation used by shards protects this object.
type clusterGlobalData struct {
	Version int                      `json:"version"`
	NextID  uint64                   `json:"next_id"`
	Peers   map[string]clusterMember `json:"peers"`
}

func (cluster *nodeCluster) globalBackend() PersistenceEngine {
	names := make([]string, 0, len(cluster.databases))
	for name := range cluster.databases {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 || databases.Get(names[0]) == nil {
		panic("global object requires a shared backend")
	}
	return databases.Get(names[0]).persistence
}

func globalContent(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func decodeGlobal(data []byte) (*clusterGlobalData, error) {
	var value clusterGlobalData
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value.Version != 1 || value.NextID == 0 || len(value.Peers) > clusterRosterLimit {
		return nil, fmt.Errorf("invalid global object")
	}
	for id, member := range value.Peers {
		if id != member.ID || !validClusterMember(member) {
			return nil, fmt.Errorf("invalid global peer")
		}
	}
	return &value, nil
}

// Called only after authority was acquired (or from an explicitly bootstrapped
// cell). A complete native SwapLog generation is the durable high-water mark.
func (cluster *nodeCluster) prepareGlobal() (string, error) {
	cell := cluster.manager.resourceCell(globalResource)
	value := cell.global.Load()
	if value == nil {
		_, entries, log := cluster.globalBackend().ReplayLog(clusterGlobalLog)
		defer log.Close()
		value = &clusterGlobalData{Version: 1, NextID: 1, Peers: map[string]clusterMember{}}
		var failure error
		for entry := range entries {
			commit, ok := entry.(LogEntryCommit)
			if !ok {
				failure = fmt.Errorf("invalid global log entry")
				continue
			}
			next, err := decodeGlobal([]byte(commit.txID))
			if err != nil {
				failure = err
				continue
			}
			value = next
		}
		if failure != nil {
			return "", failure
		}
		cell.global.CompareAndSwap(nil, value)
		value = cell.global.Load()
	}
	data, err := json.Marshal(value)
	return globalContent(data), err
}

func (cluster *nodeCluster) installGlobal(cell *replicaCell, packet mosiPacket) error {
	data := []byte(packet.Object)
	if globalContent(data) != packet.Content {
		return fmt.Errorf("global payload changed")
	}
	value, err := decodeGlobal(data)
	if err != nil {
		return err
	}
	cell.global.Store(value)
	return nil
}

// Blocks are burned on a lost reply or restart. Every numeric ID is allocated
// locally from its reserved block; only replenishment acquires network M.
func (cluster *nodeCluster) nextTransactionID() uint64 {
	cluster.idsMu.Lock()
	defer cluster.idsMu.Unlock()
	if cluster.nextID == cluster.limitID {
		cluster.reserveIDBlock(cluster.ctx)
	}
	id := cluster.nextID
	cluster.nextID++
	return id
}

func (cluster *nodeCluster) reserveIDBlock(ctx context.Context) {
	for {
		cell := cluster.obtain(ctx, globalResource, mosi.M)
		writer, wait := cell.ownstate.TryWrite(nil, false)
		if writer == nil {
			if wait != nil {
				select {
				case <-wait:
				case <-ctx.Done():
					panic(ctx.Err())
				}
			}
			continue
		}
		defer writer.Release()
		if _, err := cluster.prepareGlobal(); err != nil {
			panic(err)
		}
		old := cell.global.Load()
		if old.NextID > math.MaxInt64-clusterIDBlockSize {
			panic("cluster ID space exhausted")
		}
		value := *old
		value.Peers = maps.Clone(old.Peers)
		for _, member := range cluster.view.Load().members {
			if _, exists := value.Peers[member.ID]; !exists || member.ID == cluster.self.ID {
				value.Peers[member.ID] = member
			}
		}
		value.NextID += clusterIDBlockSize
		data, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		log := cluster.globalBackend().SwapLog(clusterGlobalLog, []any{LogEntryCommit{txID: string(data)}}, true)
		log.Close()
		cell.global.Store(&value)
		cluster.nextID, cluster.limitID = old.NextID, value.NextID
		return
	}
}

// Startup WAL replay runs before the live cluster service is published. The
// shared mounts select the data backend only; this supplies no MOSI rights.
func sharedCommitBackend() PersistenceEngine {
	if cluster := activeNodeCluster(); cluster != nil {
		return cluster.globalBackend()
	}
	mounts := sharedClusterMounts.Load()
	if mounts == nil {
		return nil
	}
	names := make([]string, 0, len(*mounts))
	for name := range *mounts {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}
	if db := databases.Get(names[0]); db != nil {
		return db.persistence
	}
	return nil
}
