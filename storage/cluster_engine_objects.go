/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Native objects keep their own authority/visibility rules. A registry reference
// retains an object; it is never an independent grant of current shard M.
func (tx *TxContext) ClusterObjectType() string { return "transaction" }
func (tx *TxContext) ClusterObjectID() string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(walBootID+fmt.Sprint(tx.ID))).String()
}
func (tx *TxContext) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method != "view" {
		return scm.NewNil(), fmt.Errorf("transaction decisions remain at the native coordinator")
	}
	return scm.NewSlice([]scm.Scmer{scm.NewInt(int64(tx.Mode)), scm.NewBool(tx.autoCommit)}), nil
}
func (*storageShard) ClusterObjectType() string { return "shard" }
func (s *storageShard) ClusterObjectID() string { return s.uuid.String() }
func (s *storageShard) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method == "view" {
		return scm.NewSlice([]scm.Scmer{scm.NewString(s.t.schema.Name), scm.NewString(s.t.Name), scm.NewString(s.uuid.String())}), nil
	}
	if r.Method == "reduce" {
		if len(r.Arguments) != 5 {
			return scm.NewNil(), fmt.Errorf("retained reduction needs filter/map columns, callbacks and neutral")
		}
		tx, _ := ctx.Value(clusterObjectQueryKey{}).(*TxContext)
		if tx == nil {
			return scm.NewNil(), fmt.Errorf("missing native object query context")
		}
		columns := func(value scm.Scmer) []string {
			result := make([]string, len(value.Slice()))
			for i, item := range value.Slice() {
				result[i] = item.String()
			}
			return result
		}
		lease := s.getRead(tx, true)
		defer lease.Release()
		access := runtimeScanAccess(nil)
		horizon := s.captureScanHorizon()
		access.ensureRuntime().horizon = &horizon
		for _, callback := range []scm.Scmer{r.Arguments[1], r.Arguments[3]} {
			if err := validateRemoteScanReadOnly(callback); err != nil {
				return scm.NewNil(), err
			}
		}
		value, _, _ := s.scan(access, columns(r.Arguments[0]), r.Arguments[1], columns(r.Arguments[2]), r.Arguments[3], r.Arguments[4], 0, nil, tx, SessionStateFromTx(tx))
		return value, nil
	}
	if r.Method == "scan" {
		session, _ := ctx.Value(clusterObjectSessionKey{}).(*scanRPCSession)
		if session == nil || len(r.Arguments) != 1 {
			return scm.NewNil(), fmt.Errorf("missing cluster scan invocation")
		}
		var task remoteScanTask
		if err := json.Unmarshal([]byte(r.Arguments[0].String()), &task); err != nil {
			return scm.NewNil(), err
		}
		result, err := session.executeScanTask(ctx, task, s)
		if err != nil {
			return scm.NewNil(), err
		}
		data, err := json.Marshal(result)
		return scm.NewString(string(data)), err
	}
	return scm.NewNil(), fmt.Errorf("unsupported shard request %q", r.Method)
}

type clusterObjectSessionKey struct{}

type remoteShardObject struct{ handle scm.ClusterHandle }

func (*remoteShardObject) ClusterObjectType() string { return "shard" }
func (s *remoteShardObject) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if r.Method == "view" {
		return s.handle.View, nil
	}
	return s.handle.Call(ctx, r)
}

// The overlay is a native TX participant, not a serialized TxContext. Its
// read view is tied to the exact physical UUID and reconstructed privately.
// This exports visibility, not private column payload or commit authority.
func (*storageShardTransaction) ClusterObjectType() string { return "tx-shard" }
func (st *storageShardTransaction) ClusterObjectID() string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(walBootID+fmt.Sprint(st.clusterTxID)+"/"+st.shardUUID.String())).String()
}
func (st *storageShardTransaction) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method != "view" {
		return scm.NewNil(), fmt.Errorf("overlay mutations remain at their native participant")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	deletes, undeletes := make([]scm.Scmer, 0, len(st.DeleteRecids)), make([]scm.Scmer, 0, len(st.UndeleteRecids))
	for _, id := range st.DeleteRecids {
		if st.DeleteMask.Get(uint(id)) {
			deletes = append(deletes, scm.NewInt(int64(id)))
		}
	}
	for _, id := range st.UndeleteRecids {
		if st.UndeleteMask.Get(uint(id)) {
			undeletes = append(undeletes, scm.NewInt(int64(id)))
		}
	}
	return scm.NewSlice([]scm.Scmer{scm.NewString(st.shardUUID.String()), scm.NewSlice(deletes), scm.NewSlice(undeletes)}), nil
}

func init() {
	scm.ClusterObjectResolver = resolveRuntimeClusterObject
	scm.RegisterClusterObjectType("tx-shard", func(handle scm.ClusterHandle) scm.Scmer {
		view := handle.View.Slice()
		if len(view) != 3 {
			panic("invalid TX shard view")
		}
		id, err := uuid.Parse(view[0].String())
		if err != nil {
			panic(err)
		}
		st := &storageShardTransaction{shardUUID: id, remoteObject: &handle}
		for _, value := range view[1].Slice() {
			recid := uint32(value.Int())
			st.DeleteMask.Set(uint(recid), true)
			st.DeleteRecids = append(st.DeleteRecids, recid)
		}
		for _, value := range view[2].Slice() {
			recid := uint32(value.Int())
			st.UndeleteMask.Set(uint(recid), true)
			st.UndeleteRecids = append(st.UndeleteRecids, recid)
		}
		return scm.NewAny(st)
	})
	scm.RegisterClusterObjectType("transaction", func(handle scm.ClusterHandle) scm.Scmer {
		view := handle.View.Slice()
		if len(view) != 2 {
			return scm.NewNil()
		}
		tx := NewTxContext(TxMode(view[0].Int()))
		tx.autoCommit = view[1].Bool()
		tx.remoteObject = &handle
		return scm.NewAny(tx)
	})
	scm.RegisterClusterObjectType("shard", func(handle scm.ClusterHandle) scm.Scmer { return scm.NewAny(&remoteShardObject{handle}) })
}

func (origin *TxContext) BindClusterInvocation(value scm.Scmer) (scm.Scmer, error) {
	target, ok := value.Any().(*TxContext)
	if !ok {
		return scm.NewNil(), fmt.Errorf("missing native invocation transaction")
	}
	if origin.remoteObject == nil {
		return scm.NewNil(), fmt.Errorf("transaction is not an imported coordinator reference")
	}
	if target.remoteObject != nil && target.remoteObject.Reference.URI != origin.remoteObject.Reference.URI {
		return scm.NewNil(), fmt.Errorf("one invocation cannot mix independent transaction coordinators")
	}
	// Preparation owns target exclusively; callbacks have not started. Statement
	// identity/cancellation and native reader cleanup remain with that executor.
	target.Mode = origin.Mode
	target.autoCommit = origin.autoCommit
	target.SnapshotEpoch = origin.SnapshotEpoch
	target.remoteObject = origin.remoteObject
	return scm.NewAny(target), nil
}

func (tx *TxContext) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(tx)
}

func (s *storageShard) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(s)
}

func (s *remoteShardObject) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(s)
}

func (st *storageShardTransaction) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(st)
}
