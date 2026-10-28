/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "strings"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

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
	Columns          []scanWireValue   `json:"columns"`
	Directions       []json.RawMessage `json:"directions"`
	AcceptColumns    []string          `json:"accept_columns,omitempty"`
	Accept           json.RawMessage   `json:"accept,omitempty"`
	Projected        []string          `json:"projected"`
	PartitionColumns int               `json:"partition_columns"`
	Offset           int               `json:"offset"`
	Limit            int               `json:"limit"`
}

func freezeScanOrder(input *shardOrderInvocation, bindings *scm.SerializationContext) (*remoteScanOrder, error) {
	result := &remoteScanOrder{AcceptColumns: input.acceptCols, Projected: input.projected, PartitionColumns: input.partitionCols, Offset: input.offset, Limit: input.limit}
	for _, column := range input.columns {
		frozen, err := freezeScanValue(column, 0, bindings)
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
	if len(wire.Columns) != len(wire.Directions) || wire.PartitionColumns < 0 || wire.PartitionColumns > len(wire.Columns) || wire.Offset < 0 || wire.Limit < -1 {
		return nil, fmt.Errorf("invalid ordered scan contract")
	}
	result := &shardOrderInvocation{acceptCols: wire.AcceptColumns, projected: wire.Projected, partitionCols: wire.PartitionColumns, offset: wire.Offset, limit: wire.Limit}
	for _, column := range wire.Columns {
		value, err := column.thaw(0, decode)
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
		if err := validateRemoteScanReadOnly(value); err != nil {
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
		if err := validateRemoteScanReadOnly(value); err != nil {
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
	run := &orderedScanRun{shard: s, columns: order.projected, keys: len(queue.scols), universe: queue.universe, candidates: queue.candidateCount, rows: make([]scm.Scmer, len(queue.items))}
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
	switch request.Method {
	case "view":
		columns := make([]scm.Scmer, len(run.columns))
		for i, name := range run.columns {
			columns[i] = scm.NewString(name)
		}
		// Small Top-N runs normally fit here. Piggyback the first bounded page so
		// those scans need no separate projection roundtrip after handle import.
		first, err := run.page(0, 256, 32*1024)
		if err != nil {
			return nil, err
		}
		return scm.NewSlice([]scm.Scmer{scm.NewInt(int64(len(run.rows))), scm.NewInt(int64(run.keys)), scm.NewInt(int64(run.universe)), scm.NewSlice(columns), first, scm.NewString(run.shard.uuid.String())}), nil

	case "page":
		if len(request.Arguments) != 2 {
			return nil, fmt.Errorf("ordered page requires offset and count")
		}
		start, count := int(request.Arguments[0].Int()), int(request.Arguments[1].Int())
		if start < 0 || start > len(run.rows) || count <= 0 || count > 256 {
			return nil, fmt.Errorf("invalid ordered page range")
		}
		return run.page(start, count, 96*1024)
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
		frozen, err := freezeScanValue(page, 0)
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
	if response.Object == nil {
		return nil
	}
	return []scm.ClusterReference{*response.Object}
}

// One queue owns its bounded receive window. Buffered selected IDs remain valid
// until the serial callback or record visitor consumes them, even if fetching the
// next comparison page preceded that flush. No shared row-loop map is mutated.
type remoteOrderQueue struct {
	generation        uuid.UUID // actual owner input, independent of the local routing stub
	run               *remoteOrderedRun
	ctx               context.Context
	total, next, keys int
	columns           map[string]int
	rows              map[uint32][]scm.Scmer
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
		page, err := remote.run.handle.Call(remote.ctx, scm.ClusterRequest{Method: "page", Arguments: []scm.Scmer{scm.NewInt(int64(remote.next)), scm.NewInt(256)}})
		if err != nil {
			panic(err)
		}
		rows = page.Slice()
	}
	if len(rows) == 0 || len(rows) > 256 || len(rows) > remote.total-remote.next {
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
	if discard {
		for _, id := range consumed {
			delete(remote.rows, id)
		}
	}
	if len(queue.items) == 0 {
		remote.fetch(queue)
	}
}

func (remote *remoteOrderQueue) project(id uint32, columns []string) []scm.Scmer {
	row := remote.rows[id]
	if row == nil {
		panic("ordered projection lost physical row")
	}
	values := make([]scm.Scmer, len(columns))
	for i, name := range columns {
		position, ok := remote.columns[name]
		if !ok {
			panic("ordered projection missing column " + name)
		}
		values[i] = row[1+remote.keys+position]
	}
	return values
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

	order := &shardOrderInvocation{columns: spec.sortcols, directions: dirs, acceptCols: spec.acceptCols, accept: spec.accept, projected: columns, partitionCols: part, offset: offset, limit: limit}
	result := (shardScanInvocation{operator: shardOrdered, order: order, tx: tx, access: access, conditionCols: spec.conditionCols, condition: spec.condition}).delegate(shard, delegation)
	if result.err.r != nil {
		panic(result.err)
	}
	run, ok := result.res.Any().(*remoteOrderedRun)
	if !ok {
		panic("missing imported ordered run")
	}
	view := run.handle.View.Slice()
	if len(view) != 6 || !view[0].IsInt() || view[0].Int() < 0 || !view[1].IsInt() || view[1].Int() < 0 || view[1].Int() > int64(len(dirs)) || !view[2].IsInt() || view[2].Int() < 0 || view[2].Int() > int64(^uint32(0)) {
		panic("invalid ordered run view")
	}
	generation, err := uuid.Parse(view[5].String())
	if err != nil {
		panic("invalid ordered input generation")
	}
	remote := &remoteOrderQueue{generation: generation, run: run, ctx: scanQueryContext(tx, context.Background()), total: int(view[0].Int()), keys: int(view[1].Int()), columns: make(map[string]int), rows: make(map[uint32][]scm.Scmer)}
	for i, name := range view[3].Slice() {
		remote.columns[name.String()] = i
	}
	queue := &shardqueue{shard: shard, remote: remote, universe: uint32(view[2].Int()), candidateCount: result.candidateCount, sortdirs: dirs}
	for i := 0; i < remote.keys; i++ {
		k := i
		queue.scols = append(queue.scols, func(id uint32) scm.Scmer { return remote.rows[id][1+k] })
		queue.sortless = append(queue.sortless, scm.OrderRelationLess(dirs[i]))
	}
	remote.fetch(queue)
	return queue
}

func (queue *shardqueue) acceptOrderedPostFilter(spec *scanOrderTableSpec, id uint32) bool {
	if queue.remote != nil {
		if spec.postOrderFilter.IsNil() {
			return true
		}
		return scm.ToBool(scm.Apply(spec.postOrderFilter, queue.remote.project(id, spec.postOrderCols)...))
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
	program := scm.PrepareSerialProc(spec.callback)
	for _, id := range ids {
		args := append([]scm.Scmer{acc}, remote.project(id, spec.callbackCols)...)
		acc = program.Call(args)
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
