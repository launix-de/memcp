/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "time"
import "context"
import "strings"
import "sync/atomic"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

const (
	minOrderedBatch   = 16
	maxOrderedBatch   = 4096
	orderedQueueItems = 16 * 1024
	// An invocation owns this bounded I/O window, independently of CPU fanout.
	orderedFinishWindow = 32
)

// Divide one coordinator's target window across its shard jobs. The floor
// amortizes network messages for large fanouts; the wire byte ceiling can still
// shorten a batch of unusually wide rows. LIMIT bounds useful speculative work.
func orderedScanBatchSize(shards, limit int) int {
	size := min(maxOrderedBatch, max(minOrderedBatch, orderedQueueItems/max(1, shards)))
	if limit >= 0 {
		size = min(size, max(minOrderedBatch, limit))
	}
	return size
}

// Ordered delegation keeps the native shard sort and coordinator merge. The
// worker freezes a projected run, not a shard dump: only comparison keys and
// requested column values leave the owner. The serial accumulator remains at
// the coordinator and is invoked only for globally selected rows. In particular
// it is never run speculatively before OFFSET/LIMIT.
//
// A run is immutable query workspace, outside the resident-shard budget. It
// releases native payload/readers before replying, so preparing many runs cannot
// pin every input shard in RAM. Its ordinary registered object lifetime retains
// the physical generation until every imported peer handle becomes unreachable.
// Pages are bounded; neither an unbounded RPC reply nor per-row RPC is allowed.
type shardOrderInvocation struct {
	batchSize     int
	columns       []scm.Scmer
	directions    []func(...scm.Scmer) scm.Scmer
	acceptCols    []string
	accept        scm.Scmer
	projected     []string
	partitionCols int
	offset        int
	limit         int
}

type remoteScanOrder struct {
	BatchSize        int               `json:"batch_size"`
	Columns          []schemeValue     `json:"columns"`
	Directions       []json.RawMessage `json:"directions"`
	AcceptColumns    []string          `json:"accept_columns,omitempty"`
	Accept           json.RawMessage   `json:"accept,omitempty"`
	Projected        []string          `json:"projected"`
	PartitionColumns int               `json:"partition_columns"`
	Offset           int               `json:"offset"`
	Limit            int               `json:"limit"`
}

func freezeScanOrder(input *shardOrderInvocation, bindings *scm.SerializationContext) (*remoteScanOrder, error) {
	result := &remoteScanOrder{BatchSize: input.batchSize, AcceptColumns: input.acceptCols, Projected: input.projected, PartitionColumns: input.partitionCols, Offset: input.offset, Limit: input.limit}
	for _, column := range input.columns {
		frozen, err := freezeScanValue(column, bindings)
		if err != nil {
			return nil, err
		}
		result.Columns = append(result.Columns, frozen)
	}
	for _, direction := range input.directions {
		encoded, err := freezeScanDirection(direction, bindings)
		if err != nil {
			return nil, err
		}
		result.Directions = append(result.Directions, encoded)
	}

	if !input.accept.IsNil() {
		var err error
		result.Accept, err = scm.MarshalProcedure(input.accept, bindings)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func freezeScanDirection(direction func(...scm.Scmer) scm.Scmer, bindings *scm.SerializationContext) (json.RawMessage, error) {
	var expression scm.Scmer
	if name, reverse, ok := scm.LookupCollate(direction); ok {
		expression = scm.NewSlice([]scm.Scmer{scm.NewSymbol("collate"), scm.NewString(name), scm.NewBool(reverse)})
	} else if declaration := scm.DeclarationForValue(scm.NewFunc(direction)); declaration != nil {
		expression = scm.NewSymbol(declaration.Name)
	} else {
		return nil, fmt.Errorf("ordered comparison has no portable constructor")
	}

	// Construct the native collation once on the worker, preserving its identity
	// and index-order metadata as well as its value/NULL comparison semantics.
	proc := scm.Eval(scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(nil), expression}), &scm.Globalenv)
	return scm.MarshalProcedure(proc, bindings)
}

func (wire *remoteScanOrder) bind(decode scm.ProcedureDecode) (*shardOrderInvocation, error) {
	if wire.BatchSize < minOrderedBatch || wire.BatchSize > maxOrderedBatch || len(wire.Columns) != len(wire.Directions) || wire.PartitionColumns < 0 || wire.PartitionColumns > len(wire.Columns) || wire.Offset < 0 || wire.Limit < -1 {
		return nil, fmt.Errorf("invalid ordered scan contract")
	}
	result := &shardOrderInvocation{batchSize: wire.BatchSize, acceptCols: wire.AcceptColumns, projected: wire.Projected, partitionCols: wire.PartitionColumns, offset: wire.Offset, limit: wire.Limit}
	for _, column := range wire.Columns {
		value, err := column.reconstruct(decode)
		if err != nil {
			return nil, err
		}
		result.columns = append(result.columns, value)
	}
	var directions []scm.Scmer
	for _, encoded := range wire.Directions {
		value, err := scm.UnmarshalProcedure(encoded, decode)
		if err != nil {
			return nil, err
		}
		directions = append(directions, scm.Apply(value))
	}
	result.directions = scanSortDirections(directions)
	if len(wire.Accept) > 0 {
		value, err := scm.UnmarshalProcedure(wire.Accept, decode)
		if err != nil {
			return nil, err
		}
		result.accept = value
	}
	for _, name := range append(append([]string(nil), wire.AcceptColumns...), wire.Projected...) {
		if strings.HasPrefix(name, "$") {
			return nil, fmt.Errorf("ordered projection contains a mutation column")
		}
	}
	return result, nil
}

type orderedScanRun struct {
	mu         sync.Mutex // batch ACK/finish only; no row-loop synchronization
	batchSize  int
	sent       int
	lastStart  int
	closed     bool
	shard      *storageShard
	columns    []string
	keys       int
	universe   uint32
	candidates int64
	rows       []scm.Scmer // each row: physical recid, comparison keys, projection values
}

func (s *storageShard) prepareOrderedRun(input shardScanInvocation) *orderedScanRun {
	order := input.order
	queue := s.scan_order(input.access, input.conditionCols, input.condition, order.acceptCols, order.accept, order.columns, order.directions, order.partitionCols, order.offset, order.limit, nil, input.tx, SessionStateFromTx(input.tx))
	defer queue.closeReaders()
	run := &orderedScanRun{batchSize: order.batchSize, shard: s, columns: order.projected, keys: len(queue.scols), universe: queue.universe, candidates: queue.candidateCount, rows: make([]scm.Scmer, len(queue.items))}
	if run.batchSize == 0 {
		run.batchSize = orderedScanBatchSize(1, order.limit)
	}
	if len(queue.items) == 0 {
		return run
	} // an eliminated shard need not load projection columns
	identity := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return scm.NewSlice(append([]scm.Scmer(nil), values...)) })
	mapper := s.OpenMapper(order.projected, identity, false, input.tx)
	defer mapper.Close()
	for i, id := range queue.items {
		row := make([]scm.Scmer, 1+run.keys+len(order.projected))
		row[0] = scm.NewInt(int64(id))
		for k, get := range queue.scols {
			row[1+k] = get(id)
		}
		copy(row[1+run.keys:], mapper.MapOne(id).Slice())
		run.rows[i] = scm.NewSlice(row)
	}
	return run
}

func (*orderedScanRun) ClusterObjectType() string { return "ordered-run" }
func (run *orderedScanRun) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(run)
}
func (run *orderedScanRun) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if request.Method == "finish" {
		run.closed = true
		run.rows = nil
		return scm.NewNil(), nil
	}
	if run.closed {
		return nil, fmt.Errorf("ordered shard job is finished")
	}
	switch request.Method {
	case "view":
		columns := make([]scm.Scmer, len(run.columns))
		for i, name := range run.columns {
			columns[i] = scm.NewString(name)
		}
		// Small Top-N runs normally fit here. Piggyback the first bounded page so
		// those scans need no separate projection roundtrip after handle import.
		first, err := run.page(0, run.batchSize, 32*1024)
		if err != nil {
			return nil, err
		}
		if run.sent == 0 {
			run.sent = len(first.Slice())
		}
		return scm.NewSlice([]scm.Scmer{scm.NewInt(int64(len(run.rows))), scm.NewInt(int64(run.keys)), scm.NewInt(int64(run.universe)), scm.NewSlice(columns), first, scm.NewString(run.shard.uuid.String()), scm.NewInt(int64(run.batchSize))}), nil

	case "ack":
		if len(request.Arguments) != 2 {
			return nil, fmt.Errorf("ordered ACK requires consumed count and next offset")
		}
		consumed, start := int(request.Arguments[0].Int()), int(request.Arguments[1].Int())
		if start != run.sent || start >= len(run.rows) || consumed < run.lastStart+(run.sent-run.lastStart+1)/2 || consumed > start {
			return nil, fmt.Errorf("invalid ordered batch ACK")
		}
		page, err := run.page(start, run.batchSize, 96*1024)
		if err == nil {
			run.lastStart, run.sent = start, start+len(page.Slice())
		}
		return page, err
	}

	return nil, fmt.Errorf("unsupported ordered-run request %q", request.Method)
}

// Size the actual generic wire value, not a guessed row width. The smaller
// initial window leaves space for the reference and its transfer offer headers.
func (run *orderedScanRun) page(start, count, budget int) (scm.Scmer, error) {
	end := start + count
	if end > len(run.rows) {
		end = len(run.rows)
	}
	for {
		page := scm.NewSlice(run.rows[start:end])
		frozen, err := freezeScanValue(page)
		if err != nil {
			return scm.NewNil(), err
		}
		data, err := json.Marshal(frozen)
		if err != nil {
			return scm.NewNil(), err
		}
		if len(data) <= budget {
			return page, nil
		}
		if end-start <= 1 {
			if budget < 96*1024 {
				return scm.NewSlice(nil), nil
			} // fetch a larger regular page
			return scm.NewNil(), fmt.Errorf("ordered row exceeds bounded node frame")
		}
		end = start + (end-start)/2
	}
}

type remoteOrderedRun struct{ handle scm.ClusterHandle }

func (*remoteOrderedRun) ClusterObjectType() string { return "ordered-run" }
func (run *remoteOrderedRun) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(run)
}
func (run *remoteOrderedRun) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if request.Method == "view" {
		return run.handle.View, nil
	}
	return run.handle.Call(ctx, request)
}
func init() {
	scm.RegisterClusterObjectType("ordered-run", func(handle scm.ClusterHandle) scm.Scmer { return scm.NewAny(&remoteOrderedRun{handle}) })
}

func (response remoteScanResult) ClusterObjectOffers() []scm.ClusterReference {
	return scm.SerializedClusterReferences(response.Value)
}

// One queue owns its bounded receive window. Buffered selected IDs remain valid
// until the serial callback or record visitor consumes them, even if fetching the
// next comparison page preceded that flush. No shared row-loop map is mutated.
type remoteOrderQueue struct {
	// One serial worker owns these prepared callbacks and borrowed frames. A
	// shard switch may flush one row; it must not prepare a new program/frame.
	callback          scm.SerialProc
	post              scm.SerialProc
	args              []scm.Scmer
	postArgs          []scm.Scmer
	mapSlots          []int
	postSlots         []int
	generation        uuid.UUID // actual owner input, independent of the local routing stub
	run               *remoteOrderedRun
	ctx               context.Context
	cancel            context.CancelFunc
	pending           chan orderedBatchResult // at most one successor batch
	batchSize         int
	pageSize          int
	consumed          int
	total, next, keys int
	columns           map[string]int
	rows              map[uint32][]scm.Scmer
}

type orderedBatchResult struct {
	page       scm.Scmer
	err        error
	panicValue any
}

// The half-window ACK grants exactly one more batch to this shard job. The
// goroutine owns only the reply, never the consumer's row directory or cursor.
func (remote *remoteOrderQueue) prefetch() {
	if remote.pending != nil || remote.next == remote.total {
		return
	}
	reply := make(chan orderedBatchResult, 1)
	remote.pending = reply
	consumed, next := remote.consumed, remote.next
	go func() {
		result := orderedBatchResult{}
		defer func() {
			result.panicValue = recover()
			reply <- result
		}()
		result.page, result.err = remote.run.handle.Call(remote.ctx, scm.ClusterRequest{Method: "ack", Arguments: []scm.Scmer{scm.NewInt(int64(consumed)), scm.NewInt(int64(next))}})
	}()
}

// LIMIT, errors and cancellation all cancel and join a speculative receive
// before releasing the job. Finishing drops its buffered answers immediately;
// the existing GC lease continues to protect any retained physical identities.
func (remote *remoteOrderQueue) close(finishCtx context.Context) {
	if remote.cancel == nil {
		return
	}
	defer func() { remote.rows = nil }()
	remote.cancel()
	if remote.pending != nil {
		<-remote.pending
		remote.pending = nil
	}
	remote.cancel = nil
	if finishCtx == nil {
		var cancel context.CancelFunc
		finishCtx, cancel = context.WithTimeout(context.WithoutCancel(remote.ctx), 5*time.Second)
		defer cancel()
	}
	_, _ = remote.run.handle.Call(finishCtx, scm.ClusterRequest{Method: "finish"})
}

// The queue remains owned until its remote job and all borrowed readers join.
// Deferred release also covers a panic while decoding a finish response.
func closeOrderedQueue(queue *shardqueue, finishCtx context.Context) {
	if queue.postOrderMapper != nil {
		defer queue.postOrderMapper.Close()
	}
	if queue.mapper != nil {
		defer queue.mapper.Close()
	}
	if queue.remote == nil {
		queue.closeReaders()
		return
	}
	defer queue.closeReaders()
	queue.remote.close(finishCtx)
}

// Cancel every speculative fetch before waiting on any finish reply. A bounded
// set of collectors overlaps independent remote acknowledgements; it does not
// retain CPU fanout slots or create one goroutine per shard. All collectors join
// before returning, including when an individual queue's cleanup panics.
func closeOrderedQueues(queues []*shardqueue) {
	remoteCount := 0
	var base context.Context
	for _, queue := range queues {
		if remote := queue.remote; remote != nil && remote.cancel != nil {
			remote.cancel()
			remoteCount++
			base = remote.ctx
		}
	}
	if remoteCount < 2 {
		for _, queue := range queues {
			closeOrderedQueue(queue, nil)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(base), 5*time.Second)
	defer cancel()
	for _, queue := range queues {
		if queue.remote == nil || queue.remote.cancel == nil {
			closeOrderedQueue(queue, nil)
		}
	}
	var next atomic.Uint64
	var joined sync.WaitGroup
	failures := make(chan any, 1)
	for range min(remoteCount, orderedFinishWindow) {
		joined.Add(1)
		go func() {
			defer joined.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(queues) {
					return
				}
				queue := queues[i]
				if queue.remote == nil || queue.remote.cancel == nil {
					continue
				}
				func() {
					defer func() {
						if failure := recover(); failure != nil {
							select {
							case failures <- failure:
							default:
							}
						}
					}()
					closeOrderedQueue(queue, ctx)
				}()
			}
		}()
	}
	joined.Wait()
	select {
	case failure := <-failures:
		panic(failure)
	default:
	}
}

func (remote *remoteOrderQueue) fetch(queue *shardqueue) {
	if remote.next == remote.total {
		return
	}
	var rows []scm.Scmer
	if remote.next == 0 {
		rows = remote.run.handle.View.Slice()[4].Slice()
	}
	if len(rows) == 0 {
		remote.prefetch()
		result := <-remote.pending
		remote.pending = nil
		if result.panicValue != nil {
			panic(result.panicValue)
		}
		if result.err != nil {
			panic(result.err)
		}
		rows = result.page.Slice()
	}
	if len(rows) == 0 || len(rows) > remote.batchSize || len(rows) > remote.total-remote.next {
		panic("invalid ordered page length")
	}
	queue.items = make([]uint32, len(rows))
	for i, value := range rows {
		row := value.Slice()
		if len(row) != 1+remote.keys+len(remote.columns) || !row[0].IsInt() || row[0].Int() < 0 || row[0].Int() >= int64(queue.universe) {
			panic("invalid ordered page row")
		}
		id := uint32(row[0].Int())
		queue.items[i] = id
		remote.rows[id] = row
	}
	remote.next += len(rows)
	remote.pageSize = len(rows)
}

// Keep the native merge advance inlinable: its only extra work is a pointer
// read. Remote window maintenance never enters a local shard's item loop.
func (queue *shardqueue) consumeOrdered(n int, discard bool) {
	previous := queue.items
	queue.items = previous[n:]
	if remote := queue.remote; remote != nil {
		remote.consume(queue, previous[:n], discard)
	}
}
func (remote *remoteOrderQueue) consume(queue *shardqueue, consumed []uint32, discard bool) {
	remote.consumed += len(consumed)
	if discard {
		for _, id := range consumed {
			delete(remote.rows, id)
		}
	}
	if len(queue.items) <= remote.pageSize/2 {
		remote.prefetch()
	}
	if len(queue.items) == 0 {
		remote.fetch(queue)
	}
}

func (remote *remoteOrderQueue) loadProjection(dst []scm.Scmer, id uint32, slots []int) {
	row := remote.rows[id]
	if row == nil {
		panic("ordered projection lost physical row")
	}
	for i, slot := range slots {
		dst[i] = row[slot]
	}
}
func (remote *remoteOrderQueue) projectionSlots(columns []string) []int {
	result := make([]int, len(columns))
	for i, name := range columns {
		position, ok := remote.columns[name]
		if !ok {
			panic("ordered projection missing column " + name)
		}
		result[i] = 1 + remote.keys + position
	}
	return result
}

func delegateOrderedQueue(tx *TxContext, shard *storageShard, access scanAccess, spec *scanOrderTableSpec, dirs []func(...scm.Scmer) scm.Scmer, part, offset, limit int, failure any) *shardqueue {
	delegation, ok := failure.(shardReadDelegation)
	if !ok {
		panic(failure)
	}
	var columns []string
	seen := make(map[string]bool)
	for _, group := range [][]string{spec.callbackCols, spec.postOrderCols, spec.projectionCols} {
		for _, name := range group {
			if !seen[name] {
				columns = append(columns, name)
				seen[name] = true
			}
		}
	}

	batchSize := spec.remoteBatchSize
	if batchSize == 0 {
		batchSize = orderedScanBatchSize(1, limit)
	}
	order := &shardOrderInvocation{batchSize: batchSize, columns: spec.sortcols, directions: dirs, acceptCols: spec.acceptCols, accept: spec.accept, projected: columns, partitionCols: part, offset: offset, limit: limit}
	result := (shardScanInvocation{operator: shardOrdered, order: order, tx: tx, access: access, conditionCols: spec.conditionCols, condition: spec.condition}).delegate(shard, delegation)
	if result.err.r != nil {
		panic(result.err)
	}
	run, ok := result.res.Any().(*remoteOrderedRun)
	if !ok {
		panic("missing imported ordered run")
	}
	view := run.handle.View.Slice()
	if len(view) != 7 || !view[0].IsInt() || view[0].Int() < 0 || !view[1].IsInt() || view[1].Int() < 0 || view[1].Int() > int64(len(dirs)) || !view[2].IsInt() || view[2].Int() < 0 || view[2].Int() > int64(^uint32(0)) || !view[6].IsInt() || int(view[6].Int()) != batchSize {
		panic("invalid ordered run view")
	}
	generation, err := uuid.Parse(view[5].String())
	if err != nil {
		panic("invalid ordered input generation")
	}
	ctx, cancel := context.WithCancel(scanQueryContext(tx, context.Background()))
	remote := &remoteOrderQueue{generation: generation, run: run, ctx: ctx, cancel: cancel, batchSize: batchSize, total: int(view[0].Int()), keys: int(view[1].Int()), columns: make(map[string]int), rows: make(map[uint32][]scm.Scmer)}
	complete := false
	defer func() {
		if !complete {
			remote.close(nil)
		}
	}()
	for i, name := range view[3].Slice() {
		remote.columns[name.String()] = i
	}
	if spec.recordVisitor == nil {
		remote.callback = scm.PrepareSerialProc(spec.callback)
		remote.args = make([]scm.Scmer, 1+len(spec.callbackCols))
		remote.mapSlots = remote.projectionSlots(spec.callbackCols)
	}
	if !spec.postOrderFilter.IsNil() {
		remote.post = scm.PrepareSerialProc(spec.postOrderFilter)
		remote.postArgs = make([]scm.Scmer, len(spec.postOrderCols))
		remote.postSlots = remote.projectionSlots(spec.postOrderCols)
	}
	queue := &shardqueue{shard: shard, remote: remote, universe: uint32(view[2].Int()), candidateCount: result.candidateCount, sortdirs: dirs}
	for i := 0; i < remote.keys; i++ {
		k := i
		queue.scols = append(queue.scols, func(id uint32) scm.Scmer { return remote.rows[id][1+k] })
		queue.sortless = append(queue.sortless, scm.OrderRelationLess(dirs[i]))
	}
	remote.fetch(queue)
	complete = true
	return queue
}

func (queue *shardqueue) acceptOrderedPostFilter(spec *scanOrderTableSpec, id uint32) bool {
	if queue.remote != nil {
		if spec.postOrderFilter.IsNil() {
			return true
		}
		remote := queue.remote
		remote.loadProjection(remote.postArgs, id, remote.postSlots)
		return scm.ToBool(remote.post.Call(remote.postArgs))
	}
	return queue.postOrderMapper == nil || scm.ToBool(queue.postOrderMapper.MapOne(id))
}

func (remote *remoteOrderQueue) stream(spec *scanOrderTableSpec, queue *shardqueue, acc scm.Scmer, ids []uint32) (result scm.Scmer, broke bool) {
	defer func() {
		for _, id := range ids {
			delete(remote.rows, id)
		}
		if r := recover(); r != nil {
			if _, ok := r.(breakSentinel); ok {
				result = acc
				broke = true
			} else {
				panic(r)
			}
		}
	}()
	if spec.recordVisitor != nil {
		spec.recordVisitor(queue, ids)
		return acc, false
	}
	for _, id := range ids {
		remote.args[0] = acc
		remote.loadProjection(remote.args[1:], id, remote.mapSlots)
		acc = remote.callback.Call(remote.args)
	}
	return acc, false
}

// A materialized join input borrows immutable projected values, not its queue's
// moving receive window. The imported run handle keeps the physical owner alive
// until the last consumer of these old-generation row IDs has finished.
// For remote records the native shard pointer is routing metadata only; their
// IDs and values belong to owner.handle.View's immutable physical UUID.
type orderedRecordValues struct {
	owner   *remoteOrderedRun
	columns map[string]int
	values  []scm.Scmer
}

func (queue *shardqueue) captureOrderedValues(id uint32) *orderedRecordValues {
	if queue.remote == nil {
		return nil
	}
	remote := queue.remote
	return &orderedRecordValues{owner: remote.run, columns: remote.columns, values: remote.rows[id][1+remote.keys:]}
}
func (record *orderedRecordValues) project(columns []string) []scm.Scmer {
	result := make([]scm.Scmer, len(columns))
	for i, col := range columns {
		position, ok := record.columns[col]
		if !ok {
			panic("retained ordered row missing column " + col)
		}
		result[i] = record.values[position]
	}
	return result
}

// Local shard pointers are routing metadata for remote queues. Ordered physical
// IDs always belong to the owner's captured UUID, including a handover/rebuild
// which forwarded the request before it captured its result generation.
func (queue *shardqueue) orderedInputUUID() [16]byte {
	if queue.remote != nil {
		return queue.remote.generation
	}
	return queue.shard.uuid
}
