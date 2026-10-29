/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "fmt"
import "maps"
import "sort"
import "sync"
import "time"
import "context"
import "net/http"
import "encoding/json"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

type replicaStatus struct {
	ID        string   `json:"id"`
	Connected bool     `json:"connected"`
	Ready     bool     `json:"ready"`
	Retired   bool     `json:"retired"`
	Databases []string `json:"databases"`
}

// ReclaimReplicationDatabases is the slave's periodic maintenance entry point.
// Call it before taking rebuild/persistence locks: ordinary MOSI handovers can
// need those same locks while the upstream supplies the current content.
// The cron owns serialization; failures affect only the current database.
func ReclaimReplicationDatabases() []string {
	r := replication.Load()
	if r == nil || r.master || r.retired.Load() {
		return nil
	}
	r.mu.Lock()
	names := make([]string, 0, len(r.directory.Initialized))
	for name, initialized := range r.directory.Initialized {
		if initialized && validReplicaDatabase(name) {
			names = append(names, name)
		}
	}
	r.mu.Unlock()
	sort.Strings(names)
	var failures []string
	for _, name := range names {
		if err := func() (err error) {
			defer func() {
				if failure := recover(); failure != nil {
					err = fmt.Errorf("replication reclaim %s: %v", name, failure)
				}
			}()
			if !r.reclaimDatabase(name) {
				return fmt.Errorf("replication reclaim %s: permissions were shared again during reclaim", name)
			}
			return nil
		}(); err != nil {
			failures = append(failures, err.Error())
		}
	}
	return failures
}

// Reclaim uses ordinary MOSI requests, including empty and internal durable
// shards. It never promotes an unavailable owner's backup. This is a point-in-
// time check, not a reservation against subsequent remote queries.
func (r *replicationManager) reclaimDatabase(name string) bool {
	if r.master || !validReplicaDatabase(name) {
		panic("database reclaim requires a replication slave and a user database")
	}
	schema := r.obtain(mosiResource{Database: name}, mosi.M, "", nil)
	db := databases.Get(name)
	if db == nil {
		panic("missing reclaimed database")
	}
	db.ensureLoaded()
	r.schemaShards(db, r.id, true)
	for _, table := range db.tables.GetAll() {
		if table.isEphemeralQueryTable() {
			continue
		}
		for _, shard := range table.ActiveShards() {
			if shard != nil {
				r.obtain(*r.shardCell(shard).resource.Load(), mosi.M, "", nil)
			}
		}
	}
	ownsM := func(cell *replicaCell) bool {
		state := cell.ownstate.Snapshot()
		return state.Phase() == mosi.M || state.Phase() == mosi.IS && state.Backend() == mosi.M
	}
	for _, table := range db.tables.GetAll() {
		if table.isEphemeralQueryTable() {
			continue
		}
		for _, shard := range table.ActiveShards() {
			if shard != nil && !ownsM(r.shardCell(shard)) {
				return false
			}
		}
	}
	return ownsM(schema)
}

func (r *replicationManager) statuses() []replicaStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	known := make(map[string]bool)
	for id := range r.peers {
		known[id] = true
	}
	for id := range r.directory.Retired {
		known[id] = true
	}
	for _, record := range r.directory.Records {
		known[record.Owner] = true
		for id := range record.Readers {
			known[id] = true
		}
	}
	delete(known, r.id)
	delete(known, "")
	result := make([]replicaStatus, 0, len(known))
	for id := range known {
		status := replicaStatus{ID: id, Ready: r.peerReady[id], Retired: r.directory.Retired[id], Databases: []string{}}
		if p := r.peers[id]; p != nil {
			select {
			case <-p.done:
			default:
				status.Connected = true
			}
		}
		databaseNames := make(map[string]bool)
		for _, record := range r.directory.Records {
			if record.Owner == id || record.Readers[id] {
				databaseNames[record.Resource.Database] = true
			}
		}
		for name := range databaseNames {
			status.Databases = append(status.Databases, name)
		}
		sort.Strings(status.Databases)
		result = append(result, status)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// retire is an explicit destructive administrative override, never a liveness
// timeout. Its tombstone is durable before any ownership is reassigned. The
// best available master backup becomes authoritative; offline changes may be lost.
func (r *replicationManager) retire(id string) error {
	if id == "" || id == r.id {
		return fmt.Errorf("invalid slave")
	}
	if err := r.storeConfig(replicaConfigEntry{Kind: "retired", Name: id, Value: true}); err != nil {
		return err
	}
	r.mu.Lock()
	p := r.peers[id]
	r.mu.Unlock()
	if p != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = p.call(ctx, "retire", struct{}{}, nil)
		cancel()
		p.close()
		// An inline import which passed its retirement check must finish before
		// backup promotion. RPC workers are not joined here: retirement itself
		// can run inside one of them. The connection owner cleans up imports.
		<-p.readDone
	}
	// Take the directory after the last inline import has completed. A completed
	// copy can also outlive reclamation of local M/O, so include already-exclusive
	// placements even when this peer is no longer present in their reader set.
	// Leaving such a marker behind makes restart demand an already-retired peer.
	r.mu.Lock()
	resources := make([]mosiResource, 0)
	for name, record := range r.directory.Records {
		exclusiveBackup := record.Owner == r.id && (record.State == "M" || record.State == "O") && r.backups[name].Content != ""
		for reader := range record.Readers {
			if reader != r.id && reader != id {
				exclusiveBackup = false
			}
		}
		if record.Owner == id || record.Readers[id] || exclusiveBackup {
			resources = append(resources, record.Resource)
		}
	}
	r.mu.Unlock()
	for _, resource := range resources {
		c := r.resourceCell(resource)
		c.directory.Lock()
		handover, err := c.ownstate.BeginHandover(context.Background(), c.drainWrites)
		if err != nil {
			c.directory.Unlock()
			return err
		}
		r.mu.Lock()
		record := r.directory.Records[r.resourceKey(resource)]
		r.mu.Unlock()
		if record.Owner == id {
			if err := r.restoreBackup(record); err != nil {
				handover.Release()
				c.directory.Unlock()
				return err
			}
		}
		record = r.updateRecord(record.Resource, func(record *replicaRecord) bool {
			delete(record.Readers, id)
			if record.Owner == id {
				record.Epoch++
				record.Owner, record.State = r.id, "M"
			}
			r.confirmed[r.resourceKey(record.Resource)] = record.Owner
			return true
		})
		err = nil
		if record.Owner == r.id {
			foreign := false
			for reader := range record.Readers {
				if reader != r.id {
					foreign = true
				}
			}
			if !foreign {
				err = r.clearBackup(record.Resource)
			}
		}
		if err == nil {
			// Explicit, fenced promotion can establish root-domain authority that was
			// unknown during recovery. It is not a timeout or a backend-existence proof.
			c.clusterstate.Publish(mosi.M)
			c.ownstate.Publish(mosi.ParsePhase(record.State))
		}
		handover.Release()
		c.directory.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}

// StopReplication runs after frontend draining and before the final local
// save. It closes and joins RPC handlers without surrendering peer ownership.
func StopReplication() {
	r := replication.Load()
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		r.mu.Lock()
		r.stopping = true
		close(r.done)
		peers := make([]*mosiPeer, 0, len(r.peers)+1)
		for _, p := range r.peers {
			peers = append(peers, p)
		}
		if r.upstream != nil {
			peers = append(peers, r.upstream)
		}
		r.mu.Unlock()
		if r.releaseRole != nil {
			r.releaseRole()
		}
		for _, p := range peers {
			p.close()
		}
		for _, p := range peers {
			p.wait()
		}
	})
}

type replicaAdminResult struct {
	Value  scm.Scmer   `json:"value"`
	Rows   []scm.Scmer `json:"rows"`
	Fields []scm.Scmer `json:"fields"`
}

// Admin commands are classified by the existing parser policy boundary. The
// master authenticates and checks its own administrator flag before evaluating
// them. Sessions, transactions and compiled closures stay local to each node.
func (r *replicationManager) executeAdmin(request replicaRequest) (result replicaAdminResult, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("admin command: %v", failure)
		}
	}()
	session := scm.NewSession()
	session.Func()(scm.NewString("username"), scm.NewString(request.User))
	session.Func()(scm.NewString("schema"), scm.NewString(request.Database))
	if request.PostgreSQL {
		session.Func()(scm.NewString("syntax"), scm.NewString("postgresql"))
	}
	execute := scm.Globalenv.FindRead("mysql_run_sql_statement").Vars["mysql_run_sql_statement"]
	var outputMu sync.Mutex
	row := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		outputMu.Lock()
		defer outputMu.Unlock()
		result.Rows = append(result.Rows, values[0])
		return scm.NewBool(true)
	})
	fields := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		outputMu.Lock()
		defer outputMu.Unlock()
		result.Fields = append(result.Fields, values[0])
		return scm.NewBool(true)
	})
	result.Value = WithAutocommit(session, nil, 0, "replicated admin command", scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.Apply(execute, values[0], scm.NewString(request.Database), scm.NewString(request.SQL), session, row, fields)
	}))
	return result, nil
}

func replicationAdmin(arguments []scm.Scmer) scm.Scmer {
	r := replication.Load()
	if r == nil || r.master {
		panic("admin forwarding requires a replication slave")
	}
	session := arguments[0]
	user := session.Func()(scm.NewString("username")).String()
	database := arguments[1].String()
	user, auth, err := r.proof(user, database)
	if err != nil || !auth.Admin {
		panic("administrator access required")
	}
	peer := r.peer("")
	if peer == nil {
		panic(mosi.NetworkError)
	}
	ctx, cancel := mosiPacketContext(peer)
	defer cancel()
	var result replicaAdminResult
	request := replicaRequest{User: user, Hash: auth.Hash, Database: database, SQL: arguments[2].String(), PostgreSQL: arguments[3].Bool()}
	if err := peer.call(ctx, "admin", request, &result); err != nil {
		panic(err)
	}
	for _, fields := range result.Fields {
		scm.Apply(arguments[5], fields)
	}
	for _, row := range result.Rows {
		scm.Apply(arguments[4], row)
	}
	return result.Value
}

// Runtime diagnostics are sampled on demand; they are never checkpoints or
// restart authority. Copy outer maps under mu, then encode immutable entries.
func (r *replicationManager) statusSnapshot() ([]byte, error) {
	r.mu.Lock()
	directory := r.directory
	directory.Records = maps.Clone(directory.Records)
	directory.Users = nil // diagnostics must not disclose cached credential proofs
	directory.Retired = maps.Clone(directory.Retired)
	directory.Initialized = maps.Clone(directory.Initialized)
	// These are on-demand diagnostic samples, not an atomic combined grant and
	// never a query-admission cache. Keep the legacy directory state for clients
	// while exposing the outward/local distinction to tests and reviewers.
	type authorityView struct {
		Resource     *mosiResource `json:"resource,omitempty"`
		Clusterstate string        `json:"clusterstate"`
		Ownstate     string        `json:"ownstate"`
		Current      string        `json:"current,omitempty"`
		MainRows     int64         `json:"main_rows,omitempty"`
		DeltaRows    int64         `json:"delta_rows,omitempty"`
	}
	authority := make(map[string]authorityView, len(r.cells))
	for key, cell := range r.cells {
		cell = cell.resolved()
		view := authorityView{Resource: cell.resource.Load(), Clusterstate: cell.clusterstate.Phase().String(), Ownstate: cell.ownstate.Phase().String()}
		if shard := cell.publishedShard(); shard != nil {
			view.Current, view.MainRows, view.DeltaRows = shard.uuid.String(), int64(shard.plannerMainRows.Load()), int64(shard.plannerDeltaRows.Load())
		}
		authority[key] = view
	}
	r.mu.Unlock()
	return json.Marshal(struct {
		ID        string                   `json:"id"`
		Boot      string                   `json:"boot"`
		Ready     bool                     `json:"ready"`
		Authority map[string]authorityView `json:"authority"`
		replicaDirectory
	}{ID: r.id, Boot: r.boot, Ready: r.ready.Load(), Authority: authority, replicaDirectory: directory})
}

func initReplicationAdmin(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "replication_status", Fn: func(...scm.Scmer) scm.Scmer {
		r := replication.Load()
		if r == nil {
			panic("replication is not enabled")
		}
		data, err := r.statusSnapshot()
		if err != nil {
			panic(err)
		}
		return scm.NewString(string(data))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "admin-only local RAM snapshot of replication resources; never persisted", Return: &scm.TypeDescriptor{Kind: "string"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_start", Fn: func(a ...scm.Scmer) scm.Scmer {
		shutdown := a[1]
		if err := StartReplication(a[0].String(), func() { scm.Apply(shutdown) }); err != nil {
			panic(err)
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "initializes replication from trusted Scheme startup; HTTP listeners and routes are constructed separately by Scheme", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "string", Label: "address"}, {Kind: "func", Label: "shutdown", Params: []*scm.TypeDescriptor{}, Return: &scm.TypeDescriptor{Kind: "any"}}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "node_protocol", Fn: func(...scm.Scmer) scm.Scmer {
		return scm.NewAny(http.HandlerFunc(serveMosiHTTP))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "returns the shared node HTTP handler for explicit http_route/serve composition; role selection grants no permissions", Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_reclaim", Fn: func(a ...scm.Scmer) scm.Scmer {
		r := replication.Load()
		if r == nil {
			panic("replication is not enabled")
		}
		return scm.NewBool(r.reclaimDatabase(a[0].String()))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "admin-only: request schema and every data shard as M through normal MOSI; false if concurrently shared again", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "string", Label: "database"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_slave", Fn: func(...scm.Scmer) scm.Scmer { r := replication.Load(); return scm.NewBool(r != nil && !r.master) }, Type: &scm.TypeDescriptor{Kind: "func", Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_admin", Fn: func(a ...scm.Scmer) scm.Scmer { return replicationAdmin(a) },
		Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
			Params: []*scm.TypeDescriptor{{Kind: "func", Label: "session"}, {Kind: "string", Label: "database"}, {Kind: "string", Label: "sql"}, {Kind: "bool", Label: "postgresql"}, {Kind: "func", Label: "resultrow"}, {Kind: "func", Label: "resultfields"}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_write", Fn: func(a ...scm.Scmer) scm.Scmer {
		if !a[0].IsNil() {
			if tx, ok := a[0].Any().(*TxContext); ok {
				replicationWrite(tx)
			}
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true,
		Params: []*scm.TypeDescriptor{{Kind: "any", Label: "tx"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_slaves", Fn: func(a ...scm.Scmer) scm.Scmer {
		r := replication.Load()
		var statuses []replicaStatus = []replicaStatus{}
		if r != nil && r.master {
			statuses = r.statuses()
		}
		data, err := json.Marshal(statuses)
		if err != nil {
			panic(err)
		}
		return scm.NewString(string(data))
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "admin-only dashboard snapshot of connected and offline replication slaves", Return: &scm.TypeDescriptor{Kind: "string"}}})
	scm.Declare(en, &scm.Declaration{Name: "replication_retire", Fn: func(a ...scm.Scmer) scm.Scmer {
		r := replication.Load()
		if r == nil || !r.master {
			panic("not a replication master")
		}
		if err := r.retire(a[0].String()); err != nil {
			panic(err)
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", Description: "explicitly retire a slave and use the last master backup; may lose offline changes", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "string", Label: "slave_uuid"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
}
