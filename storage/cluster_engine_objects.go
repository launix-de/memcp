/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "sort"
import "time"
import "weak"
import "runtime"
import "context"
import "sync/atomic"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// Native objects keep their own authority/visibility rules. A registry reference
// retains an object; it is never an independent grant of current shard M.
func (tx *TxContext) ClusterObjectType() string { return tx.transactionState.ClusterObjectType() }
func (tx *TxContext) ClusterObjectID() string   { return tx.transactionState.ClusterObjectID() }
func (tx *TxContext) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	return tx.transactionState.HandleClusterRequest(ctx, r)
}
func (*transactionState) ClusterObjectType() string { return "transaction" }
func (tx *transactionState) ClusterObjectID() string {
	if tx.clusterID != "" {
		return tx.clusterID
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(walBootID+fmt.Sprint(tx.ID))).String()
}
func (tx *transactionState) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method == "enlist" {
		if tx.coordinator != nil {
			return tx.coordinator.Call(ctx, r)
		}
		if len(r.Arguments) != 2 {
			return scm.NewNil(), fmt.Errorf("transaction enlist expects one native shard participant")
		}
		participant, ok := r.Arguments[0].Any().(*storageShardTransaction)
		if !ok || participant.clusterTxID != tx.ClusterObjectID() || participant.remoteObject == nil {
			return scm.NewNil(), fmt.Errorf("participant belongs to another transaction")
		}
		tx.mu.Lock()
		defer tx.mu.Unlock()
		if tx.State != TxActive {
			return scm.NewNil(), fmt.Errorf("transaction is no longer active")
		}
		if tx.participants == nil {
			tx.participants = make(map[uuid.UUID]*storageShardTransaction)
		}
		participant.decisionToken = r.Arguments[1].String()
		tx.participants[participant.shardUUID] = participant
		return scm.NewBool(true), nil
	}
	if r.Method != "view" {
		return scm.NewNil(), fmt.Errorf("unsupported native transaction request %q", r.Method)
	}
	if registerTransactionState(tx) != tx {
		return scm.NewNil(), fmt.Errorf("conflicting native transaction identity")
	}
	return scm.NewSlice([]scm.Scmer{scm.NewInt(int64(tx.Mode)), scm.NewBool(tx.autoCommit),
		scm.NewInt(int64(tx.ID)), scm.NewString(tx.durableID()), scm.NewInt(int64(tx.SnapshotEpoch)),
		scm.NewInt(int64(tx.sourceSessionID)), scm.NewString(tx.ClusterObjectID())}), nil
}
func (*storageShard) ClusterObjectType() string { return "shard" }
func (s *storageShard) ClusterObjectID() string { return s.uuid.String() }
func (s *storageShard) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method == "view" {
		return scm.NewSlice([]scm.Scmer{scm.NewString(s.t.schema.Name), scm.NewString(s.t.Name), scm.NewString(s.uuid.String()), scm.NewInt(int64(s.t.filterSchemaFingerprint()))}), nil
	}
	if r.Method == "scan" {
		switch input := r.Payload.(type) {
		case remoteScanTask:
			session, _ := ctx.Value(clusterObjectSessionKey{}).(*scanRPCSession)
			if session == nil {
				return nil, fmt.Errorf("missing cluster scan invocation")
			}
			return s.t.schema.executeScanTask(ctx, session, input, s)
		case shardScanInvocation:
			// Advisory accounting follows the native generation, not a second
			// wire snapshot. Each kernel owns its actual fixed input boundary.
			inputCount := int64(s.main_count) + int64(s.plannerDeltaRows.Load())
			// This is the sole native reduction entry. Preparation already owns
			// the generation and TX scope; no second GetRead or decoder lives here.
			if input.tx == nil {
				return nil, fmt.Errorf("missing prepared scan transaction")
			}
			if input.operator == shardOrdered {
				run := s.prepareOrderedRun(input)
				return scanResult{res: scm.NewAny(run), outCount: int64(len(run.rows)), inputCount: int64(run.universe), candidateCount: run.candidates}, nil
			}
			if input.operator == shardExists {
				found := s.scanExists(input.access, input.conditionCols, input.condition, input.tx, SessionStateFromTx(input.tx), nil)
				var count int64
				if found {
					count = 1
				}
				return scanResult{res: scm.NewBool(found), outCount: count, inputCount: inputCount}, nil
			}
			if input.operator >= shardLookupOne {
				var value scm.Scmer
				var count int
				var stopped atomic.Bool
				column, hasValue := "", len(input.callbackCols) > 0
				if hasValue {
					column = input.callbackCols[0]
				}
				switch input.operator {
				case shardLookupOne:
					value, count = s.scanLookupOne(input.access, column, hasValue, input.tx)
				case shardLookupMany:
					value, count = s.scanLookupMany(input.access, column, hasValue, input.tx, &stopped)
				case shardLookupMapOne:
					var values []scm.Scmer
					values, count = s.scanLookupMapOne(input.access, input.callbackCols, input.tx)
					value = scm.NewSlice(values)
				case shardLookupMapMany:
					var values []scm.Scmer
					values, count = s.scanLookupMapMany(input.access, input.callbackCols, input.tx, &stopped)
					value = scm.NewSlice(values)
				}
				return scanResult{res: value, outCount: int64(count), inputCount: inputCount}, nil
			}
			value, count, candidates := s.scan(input.access, input.conditionCols, input.condition, input.callbackCols, input.mapReduce, input.neutral, input.stride, input.batchdata, input.tx, SessionStateFromTx(input.tx))
			return scanResult{res: value, outCount: count, inputCount: inputCount, candidateCount: candidates}, nil
		}
		return nil, fmt.Errorf("missing typed scan invocation")
	}
	return scm.NewNil(), fmt.Errorf("unsupported shard request %q", r.Method)
}

type clusterObjectSessionKey struct{}

// The imported generation is a regular native scan source. Its one-shard
// topology contains only a routing stub; GetRead delegates before payload use.
// The peer handle retained by the executor owns the historical generation.
type remoteShardObject struct {
	handle scm.ClusterHandle
	source *table
}

func (object *remoteShardObject) prepareSource(ctx context.Context, peer *mosiPeer) error {
	view := object.handle.View.Slice()
	if len(view) != 4 {
		return fmt.Errorf("invalid shard source view")
	}
	id, err := uuid.Parse(view[2].String())
	if err != nil {
		return err
	}
	resource := mosiResource{Database: view[0].String(), Table: view[1].String(), Shard: id.String()}
	tbl := &table{Name: resource.Table, schema: &database{Name: resource.Database}}
	shard := &storageShard{t: tbl, uuid: id}
	ref := object.handle.Reference
	executor := &peerScanExecutor{input: &ref, inputLease: scm.NewAny(object), peer: peer, ctx: context.Background(), prepared: remoteScanShard{Schema: uint64(view[3].Int()), Resource: resource}}
	shard.scanTarget.Store(&shardScanTarget{resource: resource, executor: executor})
	tbl.Shards = []*storageShard{shard}
	tbl.publishColumnNamesSnapshot()
	tbl.publishTopologyLocked()
	object.source = tbl
	return nil
}

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
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(st.clusterTxID+"/"+st.shardUUID.String())).String()
}
func (st *storageShardTransaction) HandleClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method != "view" {
		if st.remoteObject != nil {
			return st.remoteObject.Call(ctx, r)
		}
		if len(r.Arguments) < 1 || st.decisionToken == "" || r.Arguments[0].String() != st.decisionToken {
			return scm.NewNil(), fmt.Errorf("transaction decision requires the enlisted participant capability")
		}
		tx := &TxContext{transactionState: st.transaction}
		tx.decisionMu.Lock()
		defer tx.decisionMu.Unlock()
		if r.Method == "prepare" || r.Method == "commit" || r.Method == "rollback" {
			// A broken connection cancels the task, but its bounded shard loop
			// may still be finishing. Stop admission and join that native work
			// before publishing or undoing any of its transaction overlays.
			tx.mu.Lock()
			tx.remoteAdmissionClosed = true
			idle := tx.remoteIdle
			active := tx.remoteActive != 0
			tx.mu.Unlock()
			if active {
				select {
				case <-idle:
				case <-ctx.Done():
					return scm.NewNil(), ctx.Err()
				}
			}
		}
		switch r.Method {
		case "prepare":
			if tx.State != TxActive || tx.Mode != TxACID {
				return scm.NewNil(), fmt.Errorf("participant cannot prepare this transaction")
			}
			if err := tx.prepareACID(ctx); err != nil {
				return scm.NewNil(), err
			}
			return scm.NewBool(true), nil
		case "savepoint":
			sp := tx.CreateSavepoint()
			rows := make([]scm.Scmer, 0, len(sp.shardLens))
			for shard, lens := range sp.shardLens {
				rows = append(rows, scm.NewSlice([]scm.Scmer{scm.NewString(shard.uuid.String()),
					scm.NewInt(int64(lens.InsertLen)), scm.NewInt(int64(lens.DeletedLen)),
					scm.NewInt(int64(lens.DeleteLen)), scm.NewInt(int64(lens.UndeleteLen))}))
			}
			return scm.NewSlice([]scm.Scmer{scm.NewInt(int64(sp.Depth)), scm.NewInt(int64(sp.repartitionDeleteLen)), scm.NewSlice(rows)}), nil
		case "rollback-savepoint":
			if len(r.Arguments) != 2 {
				return scm.NewNil(), fmt.Errorf("participant rollback requires a native savepoint")
			}
			view := r.Arguments[1].Slice()
			if len(view) != 3 {
				return scm.NewNil(), fmt.Errorf("invalid native savepoint")
			}
			sp := Savepoint{Depth: uint32(view[0].Int()), repartitionDeleteLen: int(view[1].Int()), shardLens: make(map[*storageShard]shardSavepoint)}
			tx.mu.Lock()
			for _, row := range view[2].Slice() {
				parts := row.Slice()
				if len(parts) != 5 {
					tx.mu.Unlock()
					return scm.NewNil(), fmt.Errorf("invalid native shard savepoint")
				}
				for shard := range tx.shardTransactions() {
					if shard.uuid.String() == parts[0].String() {
						sp.shardLens[shard] = shardSavepoint{int(parts[1].Int()), int(parts[2].Int()), int(parts[3].Int()), int(parts[4].Int())}
						break
					}
				}
			}
			tx.mu.Unlock()
			tx.RollbackToSavepoint(sp)
			return scm.NewBool(true), nil
		case "commit":
			if tx.State == TxCommitted {
				return scm.NewBool(true), nil
			}
			if tx.State != TxActive {
				return scm.NewNil(), fmt.Errorf("participant transaction is not active")
			}
			if tx.prepared != nil {
				engine := sharedCommitBackend()
				if engine == nil || !readDistributedDecision(engine, tx.durableID()) {
					return scm.NewNil(), fmt.Errorf("participant has no shared commit decision")
				}
				tx.decisionStarted, tx.commitDecided = true, true
			}
			if err := tx.Commit(); err != nil {
				return scm.NewNil(), err
			}
			return scm.NewBool(true), nil
		case "rollback":
			if tx.State == TxActive {
				tx.Rollback()
			}
			return scm.NewBool(true), nil
		}
		return scm.NewNil(), fmt.Errorf("unsupported native shard participant request %q", r.Method)
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
	return scm.NewSlice([]scm.Scmer{scm.NewString(st.shardUUID.String()), scm.NewSlice(deletes), scm.NewSlice(undeletes), scm.NewString(st.clusterTxID)}), nil
}

// One endpoint per owner drives the peer's existing native transaction. Multiple
// shard participants on that peer retain their individual physical generations.
func (tx *TxContext) remoteParticipantOwners() map[string]*storageShardTransaction {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if len(tx.participants) == 0 {
		return nil
	}
	owners := make(map[string]*storageShardTransaction)
	for _, participant := range tx.participants {
		owners[participant.remoteObject.Reference.Owner] = participant
	}
	return owners
}

func (tx *TxContext) captureRemoteSavepoints() map[string]scm.Scmer {
	owners := tx.remoteParticipantOwners()
	if len(owners) == 0 {
		return nil
	}
	ctx := scanQueryContext(tx, context.Background())
	views := make(map[string]scm.Scmer, len(owners))
	for owner, participant := range owners {
		value, err := participant.remoteObject.Call(ctx, scm.ClusterRequest{Method: "savepoint", Arguments: []scm.Scmer{scm.NewString(participant.decisionToken)}})
		if err != nil {
			panic(err)
		}
		views[owner] = value
	}
	return views
}

func (tx *TxContext) rollbackRemoteSavepoints(sp Savepoint) {
	owners := tx.remoteParticipantOwners()
	if len(owners) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var failure error
	for owner, participant := range owners {
		view, found := sp.remote[owner]
		if !found {
			// An owner first enlisted after this savepoint has no earlier effects.
			view = scm.NewSlice([]scm.Scmer{scm.NewInt(int64(sp.Depth)), scm.NewInt(0), scm.NewSlice(nil)})
		}
		_, err := participant.remoteObject.Call(ctx, scm.ClusterRequest{Method: "rollback-savepoint", Arguments: []scm.Scmer{scm.NewString(participant.decisionToken), view}})
		if err != nil && failure == nil {
			failure = err
		}
	}
	if failure != nil {
		panic(failure)
	}
}

// Native transaction decisions address registered shard owners. Each owner
// runs its existing transaction implementation, including all local overlays,
// WAL barriers, rebuild forwarding and cleanup. No mutation is replayed here.
func (tx *TxContext) finishRemoteParticipants(method string) error {
	owners := tx.remoteParticipantOwners()
	participants := make([]*storageShardTransaction, 0, len(owners))
	for _, participant := range owners {
		participants = append(participants, participant)
	}
	if len(participants) == 0 {
		return nil
	}
	if method == "commit" && tx.Mode == TxACID {
		return fmt.Errorf("distributed ACID commit requires a shared prepare/publication decision")
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].remoteObject.Reference.Owner < participants[j].remoteObject.Reference.Owner
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var failure error
	completed := false
	for _, participant := range participants {
		_, err := participant.HandleClusterRequest(ctx, scm.ClusterRequest{Method: method, Arguments: []scm.Scmer{scm.NewString(participant.decisionToken)}})
		if err != nil && failure == nil {
			failure = err
		}
		if err == nil {
			completed = true
		}
	}
	if failure != nil {
		if method == "commit" && completed {
			return markCommitOutcomeUnknown(failure)
		}
		return failure
	}
	tx.mu.Lock()
	tx.participants = nil
	tx.mu.Unlock()
	return nil
}

// Enlist before the first mutation callback. The coordinator's imported native
// participant retains this overlay and generation across statement boundaries;
// the weak transaction directory alone never owns an uncommitted write.
func (st *storageShardTransaction) registerParticipant(ctx context.Context) error {
	if st.transaction.coordinator == nil {
		return nil
	}
	st.enlist.Do(func() {
		st.decisionToken = uuid.NewString()
		_, st.enlistError = st.transaction.coordinator.Call(ctx, scm.ClusterRequest{
			Method: "enlist", Arguments: []scm.Scmer{scm.NewAny(st), scm.NewString(st.decisionToken)},
		})
	})
	if st.enlistError != nil {
		// Enlistment precedes this participant's first callback. A failed setup
		// must release its empty native generation even if no owner received the
		// handle. Ambiguous delivery can retain the object, never a write effect.
		state := st.transaction
		state.mu.Lock()
		state.discardUnenlistedShardLocked(st.clusterShard, st)
		state.mu.Unlock()
	}
	return st.enlistError
}

// The native transaction owns its weak per-peer identity directory. Runtime
// imports may have distinct reference tokens or views without cloning overlays.
// A writing participant must additionally be retained by its registered owner.
var importedTransactions sync.Map // stable native transaction ID -> weak per-peer state

type importedTransactionCleanup struct {
	key   string
	state weak.Pointer[transactionState]
}

func importTransactionState(handle scm.ClusterHandle) scm.Scmer {
	view := handle.View.Slice()
	if len(view) != 7 {
		panic("invalid native transaction view")
	}
	state := &transactionState{ID: uint64(view[2].Int()), Mode: TxMode(view[0].Int()), State: TxActive,
		autoCommit: view[1].Bool(), SnapshotEpoch: uint64(view[4].Int()), sourceSessionID: uint64(view[5].Int()),
		clusterID: view[6].String(), coordinator: &handle}
	durableID := view[3].String()
	state.walTxID.Store(&durableID)
	return scm.NewAny(&TxContext{transactionState: registerTransactionState(state), remoteObject: &handle})
}

func registerTransactionState(state *transactionState) *transactionState {
	key := state.ClusterObjectID()
	for {
		entry := weak.Make(state)
		previous, loaded := importedTransactions.LoadOrStore(key, entry)
		if loaded {
			if existing := previous.(weak.Pointer[transactionState]).Value(); existing != nil {
				return existing
			}
			if !importedTransactions.CompareAndSwap(key, previous, entry) {
				continue
			}
		}
		runtime.AddCleanup(state, func(cleanup importedTransactionCleanup) {
			importedTransactions.CompareAndDelete(cleanup.key, cleanup.state)
		}, importedTransactionCleanup{key, entry})
		return state
	}
}

func init() {
	scm.ClusterObjectResolver = resolveRuntimeClusterObject
	scm.HandleResolver = func(owner, boot string, slot uint64) (scm.Handle, error) {
		service := nodeScans.Load()
		if service == nil {
			return nil, fmt.Errorf("cluster protocol is disabled")
		}
		return service.handleResolver(context.Background(), scm.NewNil())(owner, boot, slot)
	}
	scm.RegisterClusterObjectType("tx-shard", func(handle scm.ClusterHandle) scm.Scmer {
		view := handle.View.Slice()
		if len(view) != 4 {
			panic("invalid TX shard view")
		}
		id, err := uuid.Parse(view[0].String())
		if err != nil {
			panic(err)
		}
		st := &storageShardTransaction{shardUUID: id, clusterTxID: view[3].String(), remoteObject: &handle}
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
	scm.RegisterClusterObjectType("transaction", importTransactionState)
	scm.RegisterClusterObjectType("shard", func(handle scm.ClusterHandle) scm.Scmer { return scm.NewAny(&remoteShardObject{handle: handle}) })
}

func (origin *TxContext) BindClusterInvocation(value scm.Scmer) (scm.Scmer, error) {
	target, ok := value.Any().(*TxContext)
	if !ok {
		return scm.NewNil(), fmt.Errorf("missing native invocation transaction")
	}
	if origin.remoteObject == nil {
		return scm.NewNil(), fmt.Errorf("transaction is not an imported coordinator reference")
	}
	if target.remoteObject != nil && target.ClusterObjectID() != origin.ClusterObjectID() {
		return scm.NewNil(), fmt.Errorf("one invocation cannot mix independent transaction coordinators")
	}
	// Preparation owns target exclusively; callbacks have not started. Statement
	// identity/cancellation and native reader cleanup remain with that executor.
	target.transactionState = origin.transactionState
	target.remoteObject = origin.remoteObject
	if target.SessionState != nil {
		target.SessionState.ID = origin.sourceSessionID
	}
	return scm.NewAny(target), nil
}

func (tx *TxContext) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(tx.transactionState)
}

func (tx *transactionState) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
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
