/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "context"
import "github.com/launix-de/memcp/scm"

func (*database) ClusterObjectType() string { return "database" }

// Database names resolve dynamically. This object owns source/schema policy;
// transport only decodes the envelope and selects its target database.
func (database *database) HandleClusterRequest(ctx context.Context, call scm.ClusterRequest) (any, error) {
	s, _ := ctx.Value(clusterObjectSessionKey{}).(*scanRPCSession)
	if s == nil {
		return nil, fmt.Errorf("missing authenticated scan invocation")
	}
	switch request := call.Payload.(type) {
	case remoteScanDescribe:
		if call.Method != "scan.describe" || request.Database != database.Name {
			return nil, fmt.Errorf("mismatched database request")
		}
		if request.Version != scanWireVersion || !validReplicaDatabase(request.Database) || request.Table == "" {
			return nil, fmt.Errorf("invalid remote scan source")
		}
		if sharedClusterDatabase(request.Database) {
			return nil, fmt.Errorf("shared cluster sources require automatic bounded placement")
		}
		db := database
		lease := db.GetRead(nil)
		defer lease.Release()
		tbl := db.GetTable(request.Table)
		if tbl == nil || tbl.isEphemeralQueryTable() {
			return nil, fmt.Errorf("unknown or ephemeral remote table")
		}
		result := make(remoteScanDescription, 0)
		var scanErr any
		var mu sync.Mutex
		done := tbl.iterateShardsParallel(nil, scanAccess{}, nil, func(shard *storageShard, _ bool, failure any) {
			if failure != nil {
				mu.Lock()
				scanErr = failure
				mu.Unlock()
				return
			}
			part := remoteScanShard{Schema: tbl.filterSchemaFingerprint(), Resource: mosiResource{Database: db.Name, Table: tbl.Name, Shard: shard.uuid.String()}, Horizon: shard.captureScanHorizon()}
			// Native GetRead is already complete. Register under cold persistence
			// ownership only now; never hold it while acquiring remote MOSI rights.
			db.persistenceLifecycle.RLock()
			defer db.persistenceLifecycle.RUnlock()
			view, err := scm.ClusterObjectValue(shard, ctx, scm.ClusterRequest{Method: "view"})
			if err != nil {
				mu.Lock()
				scanErr = err
				mu.Unlock()
				return
			}
			ref, err := s.service.exportObject(s.peer, shard, view)
			if err != nil {
				mu.Lock()
				scanErr = err
				mu.Unlock()
				return
			}
			mu.Lock()
			result = append(result, remoteScanDescriptor{part, &ref})
			mu.Unlock()
		})
		if done != nil {
			<-done
		}
		if scanErr != nil {
			s.service.cancelObjectOffers(s.peer, result.ClusterObjectOffers())
			return nil, fmt.Errorf("remote scan acquisition: %v", scanErr)
		}
		reply, err := boundedScanReply(result)
		if err != nil {
			s.service.cancelObjectOffers(s.peer, result.ClusterObjectOffers())
		}
		return reply, err

	case remoteScanTask:
		if call.Method != "scan.execute" || request.Shard.Resource.Database != database.Name {
			return nil, fmt.Errorf("mismatched database request")
		}
		if sharedClusterDatabase(request.Shard.Resource.Database) {
			return nil, fmt.Errorf("shared cluster tasks require RAM admission through cluster.reduce")
		}
		return s.executeScanTask(ctx, request)
	}
	return nil, fmt.Errorf("unsupported database message")
}

func (*database) Serialize(*scm.SerializationContext) (scm.Scmer, error) {
	return scm.NewNil(), fmt.Errorf("database is a routing endpoint, not a serializable runtime value")
}
