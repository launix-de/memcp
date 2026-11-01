/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "io"
import "fmt"
import "maps"
import "sort"
import "sync"
import "bufio"
import "reflect"
import "context"
import "encoding/json"
import "encoding/base64"
import "encoding/binary"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import NonLockingReadMap "github.com/launix-de/NonLockingReadMap"

// A live transfer retains the exact native generations, journals and physical
// RecIDs. Rebuilding would renumber private rows, so no rebuild occurs here.
// The stream is bounded; it does not materialize a relation as a Scheme list.
type liveShardTransfer struct {
	id                 string
	manager            *replicationManager
	cell               *replicaCell
	shards             []*storageShard
	accesses           []ReadAccess
	migrations         []func()
	overlays           []*storageShardTransaction
	peer               *mosiPeer
	offers             []scm.ClusterReference
	mu                 sync.Mutex
	nextMu             sync.Mutex
	reader             *io.PipeReader
	done               chan struct{}
	closed             bool
	maintenanceRelease func()
	maintenanceOnce    sync.Once
}

type liveShardCopy struct{ accesses []ReadAccess }

func (copy *liveShardCopy) release() {
	for _, access := range copy.accesses {
		access.Release()
	}
}

type liveGrainHeader struct {
	ID               string
	Next             string
	NextReady        bool
	Translation      map[uint32]uint32
	PendingDeletes   []uint32
	Main             uint32
	Columns          []string
	ColumnReferences map[string]bool
	DeltaColumns     map[string]int
	DeltaRows        int
	Overlays         int
}
type liveOverlayHeader struct {
	Tx          scm.ClusterReference
	Depth       uint32
	Repartition []uint32
}

func (r *replicationManager) captureLiveShard(cell *replicaCell, current *storageShard) (capture *liveShardTransfer, err error) {
	capture = &liveShardTransfer{id: uuid.NewString(), manager: r, cell: cell}
	ownedCapture := capture
	complete := false
	defer func() {
		if !complete {
			ownedCapture.release()
		}
	}()
	seen := map[*storageShard]bool{}
	add := func(shard *storageShard) {
		for shard != nil && !seen[shard] {
			seen[shard] = true
			capture.shards = append(capture.shards, shard)
			shard = shard.loadNext()
		}
	}
	add(current)
	if previous := cell.retiredShards.Load(); previous != nil {
		for _, entry := range *previous {
			if shard := entry.Value(); shard != nil && shard.activeTransactions.Load() != 0 {
				add(shard)
			}
		}
	}
	states := map[*transactionState]bool{}
	for _, shard := range capture.shards {
		shard.overlays.Range(func(key, value any) bool { states[key.(*transactionState)] = true; return true })
	}
	ordered := make([]*transactionState, 0, len(states))
	for state := range states {
		ordered = append(ordered, state)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ClusterObjectID() < ordered[j].ClusterObjectID() })
	for _, state := range ordered {
		release, err := state.beginMigration()
		if err != nil {
			return nil, err
		}
		capture.migrations = append(capture.migrations, release)
	}
	for _, shard := range capture.shards {
		capture.accesses = append(capture.accesses, shard.getRead(nil, true, false))
		shard.overlays.Range(func(_, value any) bool {
			capture.overlays = append(capture.overlays, value.(*storageShardTransaction))
			return true
		})
		if shard.logfile != nil {
			shard.logfile.Flush(shard.t.PersistencyMode == Safe)
		}
	}
	complete = true
	return capture, nil
}

func (*liveShardTransfer) ClusterObjectType() string { return "shard-transfer" }
func (capture *liveShardTransfer) Serialize(bindings *scm.SerializationContext) (scm.Scmer, error) {
	return bindings.ReferenceExpression(capture)
}
func (capture *liveShardTransfer) ClusterObjectID() string { return capture.id }
func (capture *liveShardTransfer) export(peer *mosiPeer) (scm.ClusterReference, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	if capture.closed {
		return scm.ClusterReference{}, fmt.Errorf("live shard transfer has finished")
	}
	capture.peer = peer
	return capture.manager.cluster.service.exportObject(peer, capture, scm.NewNil())
}

func (capture *liveShardTransfer) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if request.Method == "view" {
		return scm.NewNil(), nil
	}
	if request.Method == "enlist" {
		return capture.enlist(ctx, request)
	}
	if request.Method != "next" {
		return nil, fmt.Errorf("unsupported live transfer request")
	}
	capture.nextMu.Lock()
	defer capture.nextMu.Unlock()
	capture.mu.Lock()
	if capture.closed {
		capture.mu.Unlock()
		return scm.NewNil(), nil
	}
	if capture.reader == nil {
		reader, writer := io.Pipe()
		capture.reader, capture.done = reader, make(chan struct{})
		go func() {
			defer close(capture.done)
			var failure error
			defer func() {
				if value := recover(); value != nil {
					failure = fmt.Errorf("live transfer: %v", value)
				}
				writer.CloseWithError(failure)
			}()
			buffered := bufio.NewWriterSize(writer, 64*1024)
			failure = capture.write(buffered)
			if failure == nil {
				failure = buffered.Flush()
			}
		}()
	}
	reader := capture.reader
	capture.mu.Unlock()
	stop := context.AfterFunc(ctx, func() { reader.CloseWithError(ctx.Err()) })
	defer stop()
	buffer := make([]byte, 8*1024)
	n, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n == 0 {
		return scm.NewNil(), nil
	}
	return scm.NewString(base64.RawStdEncoding.EncodeToString(buffer[:n])), nil
}

func (capture *liveShardTransfer) clusterInputShards() []*storageShard {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return append([]*storageShard(nil), capture.shards...)
}

func (capture *liveShardTransfer) releaseMaintenance() {
	capture.maintenanceOnce.Do(func() {
		if capture.maintenanceRelease != nil {
			capture.maintenanceRelease()
		}
	})
}

func (capture *liveShardTransfer) release() {
	capture.mu.Lock()
	if capture.closed {
		capture.mu.Unlock()
		return
	}
	capture.closed = true
	capture.releaseMaintenance()
	if capture.reader != nil {
		capture.reader.Close()
	}
	done := capture.done
	capture.mu.Unlock()
	if done != nil {
		<-done
	}
	for _, access := range capture.accesses {
		access.Release()
	}
	for _, release := range capture.migrations {
		release()
	}
	if capture.peer != nil {
		capture.manager.cluster.service.cancelObjectOffers(capture.peer, capture.offers)
	}
	capture.mu.Lock()
	capture.shards, capture.overlays, capture.accesses, capture.migrations = nil, nil, nil, nil
	capture.mu.Unlock()
}

// The native column decoders may buffer ahead. A frame reader ends at exactly
// one column, so those buffers can never consume the following generation data.
type columnFrameWriter struct{ io.Writer }

func (writer columnFrameWriter) Write(data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n := min(len(data), mosiPacketBytes)
		if err := binary.Write(writer.Writer, binary.LittleEndian, uint32(n)); err != nil {
			return total, err
		}
		if _, err := writer.Writer.Write(data[:n]); err != nil {
			return total, err
		}
		total += n
		data = data[n:]
	}
	return total, nil
}

type columnFrameReader struct {
	reader    io.Reader
	remaining uint32
	ended     bool
}

func (reader *columnFrameReader) Read(data []byte) (int, error) {
	if reader.ended {
		return 0, io.EOF
	}
	if reader.remaining == 0 {
		if err := binary.Read(reader.reader, binary.LittleEndian, &reader.remaining); err != nil {
			return 0, err
		}
		if reader.remaining == 0 {
			reader.ended = true
			return 0, io.EOF
		}
		if reader.remaining > mosiPacketBytes {
			return 0, fmt.Errorf("oversized column frame")
		}
	}
	n, err := reader.reader.Read(data[:min(len(data), int(reader.remaining))])
	reader.remaining -= uint32(n)
	return n, err
}

func writeLiveMask(writer io.Writer, mask *NonLockingReadMap.NonBlockingBitMap) error {
	if err := binary.Write(writer, binary.LittleEndian, uint64(mask.Count())); err != nil {
		return err
	}
	var failure error
	mask.Iterate(func(id uint) {
		if failure == nil {
			failure = binary.Write(writer, binary.LittleEndian, uint32(id))
		}
	})
	return failure
}
func readLiveMask(reader io.Reader, mask *NonLockingReadMap.NonBlockingBitMap) error {
	var count uint64
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return err
	}
	for range count {
		var id uint32
		if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
			return err
		}
		mask.Set(uint(id), true)
	}
	return nil
}
func writeLiveJournal(writer io.Writer, ids []uint32, mask *NonLockingReadMap.NonBlockingBitMap) error {
	if err := binary.Write(writer, binary.LittleEndian, uint64(len(ids))); err != nil {
		return err
	}
	for _, id := range ids {
		if err := binary.Write(writer, binary.LittleEndian, id); err != nil {
			return err
		}
		var bit uint8
		if mask.Get(uint(id)) {
			bit = 1
		}
		if err := binary.Write(writer, binary.LittleEndian, bit); err != nil {
			return err
		}
	}
	return nil
}
func readLiveJournal(reader io.Reader, mask *NonLockingReadMap.NonBlockingBitMap) ([]uint32, error) {
	var count uint64
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	ids := make([]uint32, 0)
	for range count {
		var id uint32
		var bit uint8
		if err := binary.Read(reader, binary.LittleEndian, &id); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &bit); err != nil {
			return nil, err
		}
		if bit > 1 {
			return nil, fmt.Errorf("invalid overlay bit")
		}
		ids = append(ids, id)
		mask.Set(uint(id), bit == 1)
	}
	return ids, nil
}
func readLiveJSON(reader *bufio.Reader, value any) error {
	data, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func (capture *liveShardTransfer) write(writer io.Writer) error {
	encode := json.NewEncoder(writer)
	if err := encode.Encode(struct {
		Version     int
		Generations int
	}{1, len(capture.shards)}); err != nil {
		return err
	}
	for _, shard := range capture.shards {
		header := liveGrainHeader{ColumnReferences: make(map[string]bool), ID: shard.uuid.String(), Main: shard.main_count, DeltaColumns: maps.Clone(shard.deltaColumns), DeltaRows: len(shard.deltaRows())}
		if next := shard.loadNext(); next != nil {
			header.Next = next.uuid.String()
			header.NextReady = shard.nextReady.Load()
			shard.nextTranslationMu.RLock()
			header.Translation = maps.Clone(shard.nextTranslation)
			shard.nextTranslationMu.RUnlock()
			header.PendingDeletes = append([]uint32(nil), shard.nextPendingDeletes...)
		}
		for _, column := range shard.t.columnDeclarations() {
			if !isRuntimeComputedColumn(column) && column.OrcMapReduceFn.IsNil() {
				header.Columns = append(header.Columns, column.Name)
				if shard.t.PersistencyMode == Safe || shard.t.PersistencyMode == Logged {
					reader := shard.t.schema.persistence.ReadColumn(shard.uuid.String(), column.Name)
					missing, known := reader.(interface{ Missing() bool })
					header.ColumnReferences[column.Name] = !known || !missing.Missing()
					reader.Close()
				}
			}
		}
		for _, st := range capture.overlays {
			if st.clusterShard == shard {
				header.Overlays++
			}
		}
		if err := encode.Encode(header); err != nil {
			return err
		}
		for _, name := range header.Columns {
			if header.ColumnReferences[name] {
				continue
			}
			shard.getColumnStorageOrPanic(name, false, nil).Serialize(columnFrameWriter{writer})
			if err := binary.Write(writer, binary.LittleEndian, uint32(0)); err != nil {
				return err
			}
		}
		for _, row := range shard.deltaRows() {
			value, err := freezeScanValue(scm.NewSlice(row))
			if err != nil {
				return err
			}
			if err := encode.Encode(value); err != nil {
				return err
			}
		}
		if err := writeLiveMask(writer, &shard.deletions); err != nil {
			return err
		}
		if err := writeLiveMask(writer, &shard.rollbackProtected); err != nil {
			return err
		}
		for _, st := range capture.overlays {
			if st.clusterShard != shard {
				continue
			}
			tx := &TxContext{transactionState: st.transaction}
			view, err := scm.ClusterObjectValue(tx, context.Background(), scm.ClusterRequest{Method: "view"})
			if err != nil {
				return err
			}
			ref := scm.ClusterReference{}
			if tx.coordinator != nil {
				ref = tx.coordinator.Reference
			} else {
				ref, err = capture.manager.cluster.service.exportObject(capture.peer, tx.transactionState, view)
			}
			if err != nil {
				return err
			}
			capture.offers = append(capture.offers, ref)
			meta := liveOverlayHeader{Tx: ref, Depth: tx.Depth}
			tx.mu.Lock()
			for _, action := range tx.repartitionDeletes {
				if action.shard == shard {
					meta.Repartition = append(meta.Repartition, action.oldRecid)
				}
			}
			tx.mu.Unlock()
			if err := encode.Encode(meta); err != nil {
				return err
			}
			err = writeLiveJournal(writer, st.InsertRecids, &st.InsertMask)
			if err == nil {
				err = writeLiveJournal(writer, st.DeletedRecids, &st.DeletedMask)
			}
			if err == nil {
				err = writeLiveJournal(writer, st.DeleteRecids, &st.DeleteMask)
			}
			if err == nil {
				err = writeLiveJournal(writer, st.UndeleteRecids, &st.UndeleteMask)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type remoteLiveTransfer struct{ handle scm.ClusterHandle }

func (*remoteLiveTransfer) ClusterObjectType() string { return "shard-transfer" }
func (remote *remoteLiveTransfer) Serialize(bindings *scm.SerializationContext) (scm.Scmer, error) {
	return bindings.ReferenceExpression(remote)
}
func (remote *remoteLiveTransfer) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	return remote.handle.Call(ctx, request)
}

type liveTransferReader struct {
	ctx    context.Context
	remote *remoteLiveTransfer
	buffer []byte
	ended  bool
}

func (reader *liveTransferReader) Read(data []byte) (int, error) {
	if len(reader.buffer) == 0 && !reader.ended {
		value, err := reader.remote.handle.Call(reader.ctx, scm.ClusterRequest{Method: "next"})
		if err != nil {
			return 0, err
		}
		if value.IsNil() {
			reader.ended = true
		} else {
			reader.buffer, err = base64.RawStdEncoding.DecodeString(value.String())
			if err != nil {
				return 0, err
			}
		}
	}
	if len(reader.buffer) == 0 {
		return 0, io.EOF
	}
	n := copy(data, reader.buffer)
	reader.buffer = reader.buffer[n:]
	return n, nil
}

func init() {
	scm.RegisterClusterObjectType("shard-transfer", func(handle scm.ClusterHandle) scm.Scmer { return scm.NewAny(&remoteLiveTransfer{handle}) })
}

func (cluster *nodeCluster) installLiveShard(cell *replicaCell, packet mosiPacket) error {
	peer := cluster.peer(packet.Capture.Owner)
	if peer == nil {
		return fmt.Errorf("live transfer peer is disconnected")
	}
	value, err := cluster.service.importObject(peer.ctx, peer, *packet.Capture)
	if err != nil {
		return err
	}
	remote, ok := value.Any().(*remoteLiveTransfer)
	if !ok {
		return fmt.Errorf("invalid live shard source")
	}
	reader := bufio.NewReader(&liveTransferReader{ctx: peer.ctx, remote: remote})
	var stream struct {
		Version     int
		Generations int
	}
	if err := readLiveJSON(reader, &stream); err != nil {
		return err
	}
	if stream.Version != 1 || stream.Generations < 1 {
		return fmt.Errorf("invalid live shard header")
	}
	db := databases.Get(packet.Resource.Database)
	if db == nil {
		return fmt.Errorf("missing live shard database")
	}
	table := db.tables.Get(packet.Resource.Table)
	if table == nil {
		return fmt.Errorf("missing live shard table")
	}
	owned := packet.State == "M"
	grains := make(map[string]*storageShard)
	nextIDs := make(map[*storageShard]string)
	var overlays []*storageShardTransaction
	var staged []*storageShard
	complete := false
	defer func() {
		if !complete {
			for _, st := range overlays {
				if st.readOnly {
					state := st.transaction
					state.mu.Lock()
					if old := state.readViews.Load(); old != nil {
						views := maps.Clone(*old)
						if views[st.clusterShard] == st {
							delete(views, st.clusterShard)
							state.readViews.Store(&views)
						}
					}
					state.mu.Unlock()
				} else {
					state := st.transaction
					state.mu.Lock()
					state.discardUnenlistedShardLocked(st.clusterShard, st)
					state.mu.Unlock()
				}
			}
		}
	}()
	for range stream.Generations {
		var header liveGrainHeader
		if err := readLiveJSON(reader, &header); err != nil {
			return err
		}
		id, err := uuid.Parse(header.ID)
		if err != nil {
			return err
		}
		shard := &storageShard{t: table, uuid: id, main_count: header.Main, columns: make(map[string]ColumnStorage), deltaColumns: header.DeltaColumns}
		for _, name := range header.Columns {
			if shard.schemaColumn(name) == nil {
				return fmt.Errorf("unknown live shard column")
			}
			if header.ColumnReferences[name] {
				shard.columns[name] = nil
				continue
			}
			framed := &columnFrameReader{reader: reader}
			var magic uint8
			if err := binary.Read(framed, binary.LittleEndian, &magic); err != nil {
				return err
			}
			typ, valid := storages[magic]
			if !valid {
				return fmt.Errorf("unknown live column storage")
			}
			storage := reflect.New(typ).Interface().(ColumnStorage)
			count, valid := deserializeReferenceStorage(storage, framed)
			if _, err := io.Copy(io.Discard, framed); err != nil {
				return err
			}
			if !valid || count != header.Main {
				return fmt.Errorf("live column row count changed")
			}
			shard.columns[name] = shard.attachColumnRuntime(name, storage)
		}
		rows := make([][]scm.Scmer, 0)
		for range header.DeltaRows {
			var raw schemeValue
			if err := readLiveJSON(reader, &raw); err != nil {
				return err
			}
			row, err := raw.reconstruct(scm.ProcedureDecode{})
			if err != nil {
				return err
			}
			rows = append(rows, row.Slice())
		}
		shard.publishDeltaRows(rows)
		shard.plannerMainRows.Store(header.Main)
		shard.plannerDeltaRows.Store(uint64(len(rows)))
		if err := readLiveMask(reader, &shard.deletions); err != nil {
			return err
		}
		if err := readLiveMask(reader, &shard.rollbackProtected); err != nil {
			return err
		}
		shard.coherence.Store(cell)
		shard.setState(WRITE)
		grains[header.ID] = shard
		nextIDs[shard] = header.Next
		shard.nextTranslation = header.Translation
		shard.nextPendingDeletes = header.PendingDeletes
		shard.nextReady.Store(header.NextReady)
		staged = append(staged, shard)
		for range header.Overlays {
			var meta liveOverlayHeader
			if err := readLiveJSON(reader, &meta); err != nil {
				return err
			}
			value, err := cluster.service.importObject(peer.ctx, peer, meta.Tx)
			if err != nil {
				return err
			}
			tx := value.Any().(*TxContext)
			st := &storageShardTransaction{clusterShard: shard, transaction: tx.transactionState, clusterTxID: tx.ClusterObjectID(), shardUUID: id, readOnly: !owned}
			st.InsertRecids, err = readLiveJournal(reader, &st.InsertMask)
			if err != nil {
				return err
			}
			st.DeletedRecids, err = readLiveJournal(reader, &st.DeletedMask)
			if err != nil {
				return err
			}
			st.DeleteRecids, err = readLiveJournal(reader, &st.DeleteMask)
			if err != nil {
				return err
			}
			st.UndeleteRecids, err = readLiveJournal(reader, &st.UndeleteMask)
			if err != nil {
				return err
			}
			tx.mu.Lock()
			tx.Depth = meta.Depth
			if owned {
				tx.bindShardEndpointLocked(st)
				directory := tx.shards.Load()
				if directory == nil {
					directory = new(sync.Map)
					tx.shards.Store(directory)
				}
				directory.Store(shard, st)
				shard.overlays.Store(tx.transactionState, st)
				shard.beginTransactionUse()
				for _, recid := range meta.Repartition {
					tx.repartitionDeletes = append(tx.repartitionDeletes, repartitionDeleteAction{table, shard, recid})
				}
			} else {
				tx.publishReadViewLocked(shard, st)
			}
			tx.mu.Unlock()
			if owned {
				tx.RegisterTouchedShard(shard)
			}
			overlays = append(overlays, st)
		}
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return fmt.Errorf("trailing or incomplete live shard stream: %v", err)
	}
	current := grains[packet.Content]
	if current == nil {
		return fmt.Errorf("live transfer omitted current generation")
	}
	for shard, next := range nextIDs {
		if next != "" {
			successor := grains[next]
			if successor == nil {
				return fmt.Errorf("missing live successor")
			}
			shard.storeNext(successor)
		}
	}
	if owned {
		// A successful enlist transfers ownership of the fully decoded native
		// state. From here it must survive even a lost subsequent grant/ACK.
		complete = true
		for _, st := range overlays {
			st.decisionToken = uuid.NewString()
			view := scm.NewSlice([]scm.Scmer{scm.NewString(st.shardUUID.String()), scm.NewString(st.clusterTxID), scm.NewString(cluster.self.ID)})
			ref, err := cluster.service.exportObject(peer, st, view)
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(ref)
			if err != nil {
				return err
			}
			if _, err := remote.handle.Call(peer.ctx, scm.ClusterRequest{Method: "enlist", Arguments: []scm.Scmer{scm.NewString(string(encoded)), scm.NewString(st.decisionToken)}}); err != nil {
				return err
			}
		}
	}
	previous := cell.current.Load()
	if err := cluster.manager.replaceShard(packet.Resource, previous, current); err != nil {
		return err
	}
	cell.current.Store(current)
	copy := &liveShardCopy{}
	for _, shard := range staged {
		if shard != current {
			cell.retainShard(shard)
		}
		shard.registerLoadedShard()
		if !owned {
			copy.accesses = append(copy.accesses, shard.getRead(nil, true, false))
		}
	}
	if old := cell.liveCopy.Swap(copy); old != nil {
		old.release()
	}
	complete = true
	return nil
}

// Enlistment acknowledges the complete native payload, before the MOSI grant.
// The sender retains immutable read views; only the receiver may later mutate.
func (capture *liveShardTransfer) enlist(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if len(request.Arguments) != 2 {
		return nil, fmt.Errorf("invalid live participant binding")
	}
	session, _ := ctx.Value(clusterObjectSessionKey{}).(*scanRPCSession)
	if session == nil || session.peer.objectPeer != capture.peer.objectPeer {
		return nil, fmt.Errorf("foreign live participant")
	}
	var ref scm.ClusterReference
	if err := json.Unmarshal([]byte(request.Arguments[0].String()), &ref); err != nil {
		return nil, err
	}
	value, err := capture.manager.cluster.service.importObject(ctx, session.peer, ref)
	if err != nil {
		return nil, err
	}
	child, ok := value.Any().(*storageShardTransaction)
	if !ok {
		return nil, fmt.Errorf("invalid live participant type")
	}
	child.decisionToken = request.Arguments[1].String()
	for _, st := range capture.overlays {
		if st.shardUUID != child.shardUUID || st.clusterTxID != child.clusterTxID {
			continue
		}
		tx := &TxContext{transactionState: st.transaction}
		st.forwarded.Store(child)
		if tx.coordinator != nil {
			endpoint := st.ClusterEndpoint()
			if _, err := tx.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "enlist", Arguments: []scm.Scmer{scm.NewAny(endpoint), scm.NewString(child.decisionToken)}}); err != nil {
				return nil, err
			}
		}
		tx.mu.Lock()
		defer tx.mu.Unlock()
		directory := tx.shards.Load()
		if directory == nil {
			return nil, fmt.Errorf("live participant is no longer active")
		}
		if _, present := directory.Load(st.clusterShard); !present {
			return scm.NewBool(true), nil
		}
		if tx.coordinator == nil {
			if tx.participants == nil {
				tx.participants = make(map[uuid.UUID]*storageShardTransaction)
			}
			tx.participants[st.shardUUID] = child
			tx.participantEpoch++
		}
		tx.publishReadViewLocked(st.clusterShard, st)
		directory.Delete(st.clusterShard)
		tx.touchedShards.Delete(st.clusterShard)
		st.clusterShard.overlays.Delete(tx.transactionState)
		st.clusterShard.endTransactionUse()
		if st.generation != nil {
			st.generation.releaseTransaction()
		}
		remaining := tx.repartitionDeletes[:0]
		for _, action := range tx.repartitionDeletes {
			if action.shard != st.clusterShard {
				remaining = append(remaining, action)
			}
		}
		tx.repartitionDeletes = remaining
		return scm.NewBool(true), nil
	}
	return nil, fmt.Errorf("live participant identity changed")
}

func (tx *TxContext) publishReadViewLocked(shard *storageShard, st *storageShardTransaction) {
	views := make(map[*storageShard]*storageShardTransaction)
	if old := tx.readViews.Load(); old != nil {
		views = maps.Clone(*old)
	}
	views[shard] = st
	tx.readViews.Store(&views)
}
