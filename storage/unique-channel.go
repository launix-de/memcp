/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "sort"
import "sync"
import "slices"
import "strings"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

type uniqueNativeShard struct {
	shard *storageShard
	guard *uniqueWriteGuard
	tx    *TxContext
}

type uniqueReservationBusy struct {
	shard    remoteScanShard
	owner    string
	capacity bool
}

func (uniqueReservationBusy) Error() string { return "UNIQUE writer admission is busy" }

func (t *table) admitUnique(busy uniqueReservationBusy, tx *TxContext) {
	cluster := activeNodeCluster()
	owner, err := cluster.locate(scanQueryContext(tx, cluster.ctx), busy.shard.Resource)
	if err != nil {
		panic(err)
	}
	if owner != busy.owner {
		return // regroup by the new owner; never recall data through a stale hint
	}
	if owner == cluster.self.ID && !busy.capacity {
		shard := cluster.manager.findShard(busy.shard.Resource)
		if shard == nil {
			return
		}
		loaded := false
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					if _, capacity := failure.(shardReadDelegation); !capacity {
						panic(failure)
					}
				}
			}()
			write := shard.GetWrite(tx)
			write.Release()
			loaded = true
		}()
		if loaded {
			return
		}
		busy.capacity = true
	}
	var preferred *clusterLink
	for _, link := range cluster.view.Load().links {
		if link.member.ID == owner {
			preferred = link
			break
		}
	}
	executor := clusterScanExecutor{cluster: cluster, preferred: preferred, expected: busy.shard}
	if busy.capacity {
		executor.preferred = nil
		executor.exclude = owner
	}
	result := executor.ExecuteShardScan(busy.shard.Resource, shardScanInvocation{operator: shardUniqueAdmit, tx: tx})
	if result.err.r != nil {
		panic(result.err)
	}
}

type uniqueReservationClient struct {
	ctx      context.Context
	cancel   context.CancelFunc
	commands *clusterReceiver
	done     chan scanResult
}

type uniqueReservationSet struct {
	ctx      context.Context
	table    *table
	topology *tableShardTopology
	local    map[string]*uniqueNativeShard
	clients  map[string]*uniqueReservationClient
	ordered  []*uniqueReservationClient
	updates  sync.Mutex // invocation-owned; serializes nested cross-peer updates
}

// Every known partition dimension narrows the probe domain. Text dimensions
// with a nonbinary equality contract cannot use the existing raw partition hash
// as an absence proof; their other dimensions can still prune the group.
func (t *table) uniqueProbeShards(topology *tableShardTopology, columns []string, key []scm.Scmer) []*storageShard {
	if sources := t.repartitionSources.Load(); t.repartitionDualWriteActive.Load() && sources != nil {
		return sources.shards
	}
	if topology.mode != ShardModePartition {
		return topology.shards
	}
	positions := make([]int, len(topology.dimensions))
	for i, dimension := range topology.dimensions {
		positions[i] = -1
		for j, column := range columns {
			if column != dimension.Column || j >= len(key) {
				continue
			}
			binary := true
			if key[j].IsString() || key[j].IsSymbol() || key[j].IsBufferString() {
				for _, definition := range t.columnDeclarations() {
					if definition.Name == column {
						collation := definition.Collation
						binary = collation == "" || collation == "bin" || collation == "binary" || collation == "utf8" || collation == "utf8mb4" || strings.HasSuffix(collation, "_bin")
						break
					}
				}
			}
			if binary {
				positions[i] = computeShardDimensionIndex(dimension, key[j])
			}
			break
		}
	}
	var selected []*storageShard
	for index, shard := range topology.shards {
		if shard == nil {
			continue
		}
		matches := true
		for i := len(positions) - 1; i >= 0; i-- {
			partition := index % topology.dimensions[i].NumPartitions
			index /= topology.dimensions[i].NumPartitions
			if positions[i] >= 0 && positions[i] != partition {
				matches = false
				break
			}
		}
		if matches {
			selected = append(selected, shard)
		}
	}
	return selected
}

func uniqueKeyValues(columns []string, row []scm.Scmer, constraint uniqueKey, mergeNull bool) ([]scm.Scmer, bool) {
	key := make([]scm.Scmer, len(constraint.Cols))
	for i, name := range constraint.Cols {
		for j, column := range columns {
			if column == name && j < len(row) {
				key[i] = row[j]
				break
			}
		}
		if key[i].IsNil() && !mergeNull && !constraint.NullsNotDistinct {
			return nil, false
		}
	}
	return key, true
}

// Complete the read-only prefix before reserving writers. The first miss is
// rechecked under reservations; already delivered collision callbacks are never
// repeated. In particular an all-duplicate IGNORE needs no writer admission.
func (t *table) uniqueReadOnlyPrefix(columns []string, rows [][]scm.Scmer, mergeNull bool, tx *TxContext, failure func(string, []scm.Scmer)) [][]scm.Scmer {
	for index, row := range rows {
		topology := t.pinActiveTopology()
		found := false
		func() {
			defer topology.releaseOperation()
			for constraintIndex := t.nextUniqueConstraint(columns, 0); constraintIndex < len(t.Unique); constraintIndex = t.nextUniqueConstraint(columns, constraintIndex+1) {
				constraint := t.Unique[constraintIndex]
				key, comparable := uniqueKeyValues(columns, row, constraint, mergeNull)
				if !comparable {
					continue
				}
				access := exactScanAccess(newExactScanAccessSchema(constraint.Cols), key)
				for _, shard := range t.uniqueProbeShards(topology, constraint.Cols, key) {
					if shard != nil && shard.probeUnique(access, tx) {
						failure(constraint.Id, nil)
						found = true
						return
					}
				}
			}
		}()
		if !found {
			return rows[index:]
		}
	}
	return nil
}

func uniqueStringArray(columns []string) scm.Scmer {
	values := make([]scm.Scmer, len(columns))
	for i, column := range columns {
		values[i] = scm.NewString(column)
	}
	return scm.NewSlice(values)
}

func uniqueRowArray(rows [][]scm.Scmer) scm.Scmer {
	values := make([]scm.Scmer, len(rows))
	for i, row := range rows {
		values[i] = scm.NewSlice(row)
	}
	return scm.NewSlice(values)
}

func (set *uniqueReservationSet) valid() bool {
	return set.table.topology.Load().routing == set.topology.routing && set.ctx.Err() == nil
}

func (set *uniqueReservationSet) call(shard *storageShard, operation string, arguments ...scm.Scmer) scm.Scmer {
	client := set.clients[shard.uuid.String()]
	if client == nil {
		if set.local == nil {
			// A concurrent lane append/rebuild can publish another topology
			// while the initial owner channels are being acquired. The outer
			// native boundary retries only if this batch has published no effects.
			panic(uniqueReadRetry{})
		}
		panic(sqldb.NewSQLError1(1213, "40001", "UNIQUE mutation extends its reserved shard domain (%s, %d reserved shards); retry transaction", shard.uuid.String(), len(set.clients)))
	}
	message := append([]scm.Scmer{scm.NewString(operation), scm.NewString(shard.uuid.String())}, arguments...)
	value, err := callUniqueChannel(client.ctx, client.commands, message...)
	if err != nil {
		panicRemoteOperation(err)
	}
	return value
}

func (set *uniqueReservationSet) probe(shard *storageShard, columns []string, key []scm.Scmer, projection []string) (uint32, bool, []scm.Scmer) {
	if part := set.local[shard.uuid.String()]; part != nil {
		id, found := part.shard.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema(columns), key), part.tx, true)
		values := make([]scm.Scmer, len(projection))
		if found {
			for i, column := range projection {
				if column == "$update" || len(column) > 4 && column[:4] == "NEW." {
					continue
				}
				reader := part.shard.ColumnReaderTx(part.tx, column, false, ReadAuto)
				values[i] = reader.GetValue(id)
				ReleaseColumnReader(reader)
			}
		}
		return id, found, values
	}
	value := set.call(shard, "probe", uniqueStringArray(columns), scm.NewSlice(key), uniqueStringArray(projection)).Slice()
	return uint32(value[1].Int()), value[0].Bool(), value[2].Slice()
}

func (set *uniqueReservationSet) update(shard *storageShard, id uint32, changes scm.Scmer) scm.Scmer {
	set.updates.Lock()
	defer set.updates.Unlock()
	if part := set.local[shard.uuid.String()]; part != nil {
		update := part.shard.UpdateFunction(id, true, false, part.tx, part.guard)
		if changes.IsNil() {
			return update()
		}
		return update(changes)
	}
	return set.call(shard, "update", scm.NewInt(int64(id)), changes)
}

func (set *uniqueReservationSet) insert(shard *storageShard, columns []string, rows [][]scm.Scmer, firstID func(int64)) {
	value := set.call(shard, "insert", uniqueStringArray(columns), uniqueRowArray(rows))
	if firstID != nil && !value.IsNil() {
		firstID(value.Int())
	}
}

func (set *uniqueReservationSet) close(commit bool) {
	var failure any
	for i := len(set.ordered) - 1; i >= 0; i-- {
		client := set.ordered[i]
		func() {
			defer func() {
				if value := recover(); value != nil && failure == nil {
					failure = value
				}
			}()
			defer client.cancel()
			operation := "abort"
			if commit {
				operation = "finish"
			}
			_, err := callUniqueChannel(client.ctx, client.commands, scm.NewString(operation))
			if err != nil {
				client.cancel()
			}
			result := <-client.done
			if commit && result.err.r != nil {
				panic(result.err)
			}
			if commit && err != nil {
				panicRemoteOperation(err)
			}
		}()
	}
	set.ordered = nil
	if commit && failure != nil {
		panic(failure)
	}
}

// Reserve one channel per owner, in the same owner/shard order on every node.
// Requests retain native worker scopes; the coordinator never imports payload.
func (t *table) reserveUniqueDomain(columns []string, rows [][]scm.Scmer, mergeNull bool, tx *TxContext, targets []*storageShard, borrowed *uniqueWriteGuard) *uniqueReservationSet {
	cluster := activeNodeCluster()
	ctx := scanQueryContext(tx, cluster.ctx)
	topology := t.activeTopology()
	set := &uniqueReservationSet{ctx: ctx, table: t, topology: topology, clients: make(map[string]*uniqueReservationClient)}
	complete := false
	defer func() {
		if !complete {
			set.close(false)
		}
	}()
	groups := make(map[string][]*storageShard)
	domain := append([]*storageShard(nil), targets...)
	for index := t.nextUniqueConstraint(columns, 0); index < len(t.Unique); index = t.nextUniqueConstraint(columns, index+1) {
		for _, row := range rows {
			if key, comparable := uniqueKeyValues(columns, row, t.Unique[index], mergeNull); comparable {
				domain = append(domain, t.uniqueProbeShards(topology, t.Unique[index].Cols, key)...)
			}
		}
	}
	seen := make(map[string]bool)
	for _, shard := range domain {
		if shard == nil || seen[shard.uuid.String()] {
			continue
		}
		seen[shard.uuid.String()] = true
		owner, err := cluster.locate(ctx, mosiResource{Database: t.schema.Name, Table: t.Name, Shard: shard.uuid.String()})
		if err != nil {
			panic(err)
		}
		groups[owner] = append(groups[owner], shard)
	}
	owners := make([]string, 0, len(groups))
	for owner := range groups {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	// Reservation setup needs key values only. Payload columns and collision
	// closures stay with the coordinator until a probe finds an actual match.
	var keyColumns []string
	var keyPositions []int
	for position, column := range columns {
		for _, constraint := range t.Unique {
			if slices.Contains(constraint.Cols, column) {
				keyColumns = append(keyColumns, column)
				keyPositions = append(keyPositions, position)
				break
			}
		}
	}
	keyRows := make([][]scm.Scmer, len(rows))
	for i, row := range rows {
		keyRows[i] = make([]scm.Scmer, len(keyPositions))
		for j, position := range keyPositions {
			keyRows[i][j] = row[position]
		}
	}
	for _, owner := range owners {
		shards := groups[owner]
		sort.Slice(shards, func(i, j int) bool { return shards[i].uuid.String() < shards[j].uuid.String() })
		if borrowed.ownedShard != nil && owner == cluster.self.ID {
			for i, shard := range shards {
				if shard == borrowed.ownedShard {
					shards[0], shards[i] = shards[i], shards[0]
					break
				}
			}
		}
		ids := make([]string, len(shards))
		for i, shard := range shards {
			ids[i] = shard.uuid.String()
		}
		client := t.startUniqueReservation(ctx, owner, shards[0], ids, keyColumns, keyRows, mergeNull, tx, borrowed)
		set.ordered = append(set.ordered, client)
		for _, id := range ids {
			set.clients[id] = client
		}
	}
	bindings := make([]scm.Scmer, 0, len(set.clients))
	for id, client := range set.clients {
		bindings = append(bindings, scm.NewSlice([]scm.Scmer{scm.NewString(id), scm.NewAny(client.commands)}))
	}
	for _, client := range set.ordered {
		if _, err := callUniqueChannel(ctx, client.commands, scm.NewString("bind"), scm.NewSlice(bindings)); err != nil {
			panicRemoteOperation(err)
		}
	}
	complete = true
	return set
}

func (t *table) startUniqueReservation(parent context.Context, owner string, first *storageShard, ids, columns []string, rows [][]scm.Scmer, mergeNull bool, tx *TxContext, borrowed *uniqueWriteGuard) *uniqueReservationClient {
	ctx, cancel := context.WithCancel(parent)
	client := &uniqueReservationClient{ctx: ctx, cancel: cancel, done: make(chan scanResult, 1)}
	ready := make(chan *clusterReceiver, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) (scm.Scmer, error) {
		ready <- value.Any().(*clusterReceiver)
		return scm.NewNil(), nil
	})
	receiver.Seal()
	defer receiver.Close()
	invocation := shardScanInvocation{operator: shardUniqueReserve, tx: tx, writeAcquired: true, neutral: scm.NewSlice([]scm.Scmer{
		scm.NewAny(receiver), uniqueStringArray(columns), uniqueRowArray(rows), scm.NewBool(mergeNull), scm.NewBool(tx == nil), uniqueStringArray(ids),
	})}
	cluster := activeNodeCluster()
	go func() {
		defer func() {
			if failure := recover(); failure != nil {
				client.done <- scanResult{err: scanError{r: failure}}
			}
		}()
		if owner == cluster.self.ID {
			if borrowed.ownedShard == first {
				invocation.uniqueGuard = borrowed
			}
			committed, err := first.serveUniqueScan(ctx, invocation)
			if busy, ok := err.(uniqueReservationBusy); ok {
				client.done <- scanResult{res: scm.NewSlice([]scm.Scmer{scm.NewString("busy"), scm.NewString(busy.shard.Resource.Shard), scm.NewBool(busy.capacity)})}
				return
			}
			if err != nil {
				panic(err)
			}
			client.done <- scanResult{res: scm.NewBool(committed)}
			return
		}
		var preferred *clusterLink
		for _, link := range cluster.view.Load().links {
			if link.member.ID == owner {
				preferred = link
				break
			}
		}
		resource := mosiResource{Database: t.schema.Name, Table: t.Name, Shard: first.uuid.String()}
		executor := clusterScanExecutor{cluster: cluster, preferred: preferred, expected: remoteScanShard{Schema: t.filterSchemaFingerprint(), Resource: resource}}
		client.done <- executor.ExecuteShardScan(resource, invocation)
	}()
	select {
	case client.commands = <-ready:
		return client
	case result := <-client.done:
		cancel()
		if result.err.r != nil {
			panic(result.err)
		}
		if result.res.IsSlice() && len(result.res.Slice()) == 3 && scm.String(result.res.Slice()[0]) == "busy" {
			panic(uniqueReservationBusy{shard: remoteScanShard{Schema: t.filterSchemaFingerprint(), Resource: mosiResource{Database: t.schema.Name, Table: t.Name, Shard: scm.String(result.res.Slice()[1])}}, owner: owner, capacity: result.res.Slice()[2].Bool()})
		}
		panic("UNIQUE reservation ended before its channel was ready")
	case <-ctx.Done():
		cancel()
		<-client.done
		panic(ctx.Err())
	}
}

func uniqueStrings(value scm.Scmer) ([]string, error) {
	if !value.IsSlice() {
		return nil, fmt.Errorf("UNIQUE columns must be an array")
	}
	columns := make([]string, len(value.Slice()))
	for i, item := range value.Slice() {
		if !item.IsString() {
			return nil, fmt.Errorf("UNIQUE columns must be strings")
		}
		columns[i] = item.String()
	}
	return columns, nil
}

func uniqueRows(value scm.Scmer, columns int) ([][]scm.Scmer, error) {
	if !value.IsSlice() {
		return nil, fmt.Errorf("UNIQUE rows must be an array")
	}
	rows := make([][]scm.Scmer, len(value.Slice()))
	for i, item := range value.Slice() {
		if !item.IsSlice() || len(item.Slice()) != columns {
			return nil, fmt.Errorf("UNIQUE row width differs from its columns")
		}
		rows[i] = item.Slice()
	}
	return rows, nil
}

// The prepared native shard invocation owns MOSI admission and transaction
// decoding until its command channel is finished and all callbacks have joined.
func (s *storageShard) serveUniqueScan(ctx context.Context, input shardScanInvocation) (committed bool, err error) {
	preparing := true
	defer func() {
		if value := recover(); value != nil {
			if capacity, ok := value.(shardReadDelegation); ok && preparing {
				committed = false
				err = uniqueReservationBusy{shard: remoteScanShard{Schema: s.t.filterSchemaFingerprint(), Resource: capacity.target.resource}, capacity: true}
			} else {
				panic(value)
			}
		}
	}()
	if !input.neutral.IsSlice() || len(input.neutral.Slice()) != 6 {
		return false, fmt.Errorf("invalid prepared UNIQUE reservation")
	}
	parameters := input.neutral.Slice()
	ready, ok := parameters[0].Any().(*clusterReceiver)
	if !ok {
		return false, fmt.Errorf("UNIQUE reservation requires a ready channel")
	}
	columns, err := uniqueStrings(parameters[1])
	if err != nil {
		return false, err
	}
	rows, err := uniqueRows(parameters[2], len(columns))
	if err != nil {
		return false, err
	}
	guard := s.t.guardUniqueWrites(columns, rows, parameters[3].Bool(), 0)
	guardPointer := &guard
	if input.uniqueGuard != nil {
		guardPointer = input.uniqueGuard
	}
	tx := input.tx
	if parameters[4].Bool() {
		// The invocation supplies cancellation and MOSI ownership, while a
		// native nil-TX mutation keeps its immediate publication semantics.
		tx = nil
	}
	ids, err := uniqueStrings(parameters[5])
	if err != nil || len(ids) == 0 {
		return false, fmt.Errorf("UNIQUE reservation requires its shard group")
	}
	if input.uniqueGuard == nil {
		firstWrite, admitted := s.getWrite(tx, false)
		if !admitted {
			return false, uniqueReservationBusy{shard: remoteScanShard{Resource: mosiResource{Database: s.t.schema.Name, Table: s.t.Name, Shard: ids[0]}}}
		}
		defer firstWrite.Release()
		s = firstWrite.shard
	}
	scope := &uniqueReservationSet{ctx: ctx, table: s.t, topology: s.t.activeTopology(), local: make(map[string]*uniqueNativeShard), clients: make(map[string]*uniqueReservationClient)}
	scope.local[ids[0]] = &uniqueNativeShard{shard: s, guard: guardPointer, tx: tx}
	guardPointer.reservations = scope
	for _, id := range ids[1:] {
		resource := mosiResource{Database: s.t.schema.Name, Table: s.t.Name, Shard: id}
		part := findScanShard(s.t, resource)
		if part == nil {
			// A retained UUID can already route to a rebuild successor. Resolve
			// it through the same native directory used by writer admission.
			part = activeNodeCluster().manager.findShard(resource)
		}
		if part == nil {
			return false, uniqueReservationBusy{shard: remoteScanShard{Resource: resource}}
		}
		// Several retained UUIDs may now address one successor. A native nil-TX
		// writer cannot reacquire itself; retain one admission for that resource.
		var existing *uniqueNativeShard
		manager := activeNodeCluster().manager
		for _, owned := range scope.local {
			if manager.shardCell(owned.shard) == manager.shardCell(part) {
				existing = owned
				break
			}
		}
		if existing != nil {
			scope.local[id] = existing
			continue
		}
		access, admitted := part.getWrite(tx, false)
		if !admitted {
			return false, uniqueReservationBusy{shard: remoteScanShard{Resource: resource}}
		}
		defer access.Release()
		scope.local[id] = &uniqueNativeShard{shard: access.shard, guard: guardPointer, tx: tx}
	}
	if !guardPointer.held {
		if !guardPointer.tryLock() {
			return false, uniqueReservationBusy{shard: remoteScanShard{Resource: mosiResource{Database: s.t.schema.Name, Table: s.t.Name, Shard: ids[0]}}}
		}
		defer guardPointer.unlock()
	}
	preparing = false
	return serveUniqueChannel(ctx, guardPointer, ready, func(_ context.Context, command []scm.Scmer) (scm.Scmer, error) {
		if scm.String(command[0]) == "bind" {
			if len(command) != 2 || !command[1].IsSlice() {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE peer bindings")
			}
			for _, binding := range command[1].Slice() {
				if !binding.IsSlice() || len(binding.Slice()) != 2 {
					return scm.NewNil(), fmt.Errorf("invalid UNIQUE peer binding")
				}
				id := scm.String(binding.Slice()[0])
				channel, ok := binding.Slice()[1].Any().(*clusterReceiver)
				if !ok {
					return scm.NewNil(), fmt.Errorf("invalid UNIQUE peer channel")
				}
				if scope.local[id] == nil {
					scope.clients[id] = &uniqueReservationClient{ctx: ctx, commands: channel}
				}
			}
			return scm.NewBool(true), nil
		}
		if len(command) < 2 {
			return scm.NewNil(), fmt.Errorf("UNIQUE command requires its shard")
		}
		part := scope.local[scm.String(command[1])]
		if part == nil {
			return scm.NewNil(), fmt.Errorf("UNIQUE command addresses an unreserved shard")
		}
		s := part.shard
		command = append([]scm.Scmer{command[0]}, command[2:]...)
		switch scm.String(command[0]) {
		case "probe":
			if len(command) != 4 || !command[2].IsSlice() {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE probe command")
			}
			keyColumns, err := uniqueStrings(command[1])
			if err != nil || len(keyColumns) != len(command[2].Slice()) {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE key")
			}
			projection, err := uniqueStrings(command[3])
			if err != nil {
				return scm.NewNil(), err
			}
			id, found := s.GetRecordidForUnique(exactScanAccess(newExactScanAccessSchema(keyColumns), command[2].Slice()), tx, true)
			values := make([]scm.Scmer, len(projection))
			if found {
				for i, column := range projection {
					if column == "$update" || len(column) > 4 && column[:4] == "NEW." {
						continue // invocation-owned inputs/handles are bound by the caller
					}
					reader := s.ColumnReaderTx(tx, column, false, ReadAuto)
					values[i] = reader.GetValue(id)
					ReleaseColumnReader(reader)
				}
			}
			return scm.NewSlice([]scm.Scmer{scm.NewBool(found), scm.NewInt(int64(id)), scm.NewSlice(values)}), nil
		case "update":
			if len(command) != 3 || !command[1].IsInt() || (!command[2].IsSlice() && !command[2].IsNil()) {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE update command")
			}
			if command[1].Int() < 0 || uint64(command[1].Int()) >= uint64(s.main_count)+uint64(len(s.deltaRows())) {
				return scm.NewNil(), fmt.Errorf("UNIQUE record outside its retained generation")
			}
			update := s.UpdateFunction(uint32(command[1].Int()), true, false, tx, guardPointer)
			if command[2].IsNil() {
				return update(), nil
			}
			return update(command[2]), nil
		case "insert":
			if len(command) != 3 {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE insert command")
			}
			insertColumns, err := uniqueStrings(command[1])
			if err != nil {
				return scm.NewNil(), err
			}
			insertRows, err := uniqueRows(command[2], len(insertColumns))
			if err != nil {
				return scm.NewNil(), err
			}
			firstID := scm.NewNil()
			s.Insert(insertColumns, insertRows, false, nil, func(id int64) { firstID = scm.NewInt(id) }, false, tx)
			return firstID, nil
		}
		return scm.NewNil(), fmt.Errorf("unknown UNIQUE command")
	})
}

// serveUniqueChannel borrows the native worker's lifetime. It holds ownership
// until finish/abort, cancellation or a failed command has joined all admitted
// channel callbacks. No goroutine, receiver or lock is retained per stored row.
// Commands and replies use ordinary receiver handles and Scheme arrays.
func serveUniqueChannel(ctx context.Context, guard *uniqueWriteGuard, ready *clusterReceiver, dispatch func(context.Context, []scm.Scmer) (scm.Scmer, error)) (bool, error) {
	if !guard.held {
		guard.lock()
		defer guard.unlock()
	}
	done := make(chan struct{})
	finished, commit := false, false // accessed only by the receiver's FIFO
	commands := newClusterReceiver(-1, func(commandCtx context.Context, message scm.Scmer) (_ scm.Scmer, err error) {
		defer func() {
			if value := recover(); value != nil {
				if !finished {
					finished = true
					close(done)
				}
				panic(value)
			}
		}()
		if finished || !message.IsSlice() || len(message.Slice()) < 1 {
			return scm.NewNil(), fmt.Errorf("closed or invalid UNIQUE reservation command")
		}
		parts := message.Slice()
		var value scm.Scmer
		switch scm.String(parts[0]) {
		case "finish", "abort":
			if len(parts) != 1 {
				return scm.NewNil(), fmt.Errorf("invalid UNIQUE completion")
			}
			finished, commit = true, scm.String(parts[0]) == "finish"
			defer close(done)
			value = scm.NewBool(commit)
		default:
			value, err = dispatch(commandCtx, parts)
			if err != nil {
				finished = true
				defer close(done)
				return scm.NewNil(), err
			}
		}
		return value, nil
	})
	commands.Seal()
	defer commands.Close() // join commands before releasing the borrowed guard
	if _, err := ready.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewAny(commands)}}); err != nil {
		return false, err
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-done:
		// Close joins the terminal callback before reading its result.
		commands.Close()
		return commit, nil
	}
}

func callUniqueChannel(ctx context.Context, commands *clusterReceiver, parts ...scm.Scmer) (scm.Scmer, error) {
	value, err := commands.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewSlice(parts)}})
	if err != nil {
		return scm.NewNil(), err
	}
	return value.(scm.Scmer), nil
}
