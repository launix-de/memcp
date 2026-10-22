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
import "sort"
import "sync"
import "time"
import "context"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type replicaStatus struct {
	ID        string   `json:"id"`
	Connected bool     `json:"connected"`
	Retired   bool     `json:"retired"`
	Databases []string `json:"databases"`
}

func (r *replicationManager) statuses() []replicaStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	known := make(map[string]bool)
	for id := range r.peers {
		known[id] = true
	}
	for id := range r.journal.Retired {
		known[id] = true
	}
	for _, record := range r.journal.Records {
		known[record.Owner] = true
		for id := range record.Readers {
			known[id] = true
		}
	}
	delete(known, r.id)
	delete(known, "")
	result := make([]replicaStatus, 0, len(known))
	for id := range known {
		status := replicaStatus{ID: id, Retired: r.journal.Retired[id], Databases: []string{}}
		if p := r.peers[id]; p != nil {
			select {
			case <-p.done:
			default:
				status.Connected = true
			}
		}
		databaseNames := make(map[string]bool)
		for _, record := range r.journal.Records {
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
	r.mu.Lock()
	if id == "" || id == r.id {
		r.mu.Unlock()
		return fmt.Errorf("invalid slave")
	}
	r.journal.Retired[id] = true
	err := r.saveLocked()
	p := r.peers[id]
	names := make([]string, 0)
	for name, record := range r.journal.Records {
		if record.Owner == id || record.Readers[id] {
			names = append(names, name)
		}
	}
	r.mu.Unlock()
	if err != nil {
		return err
	}
	if p != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = p.call(ctx, "retire", struct{}{}, nil)
		cancel()
		p.close()
	}
	for _, name := range names {
		c := r.cell(name)
		c.directory.Lock()
		c.gate.Lock()
		r.mu.Lock()
		record := r.journal.Records[name]
		r.mu.Unlock()
		if record.Owner == id {
			if err := r.restoreBackup(record); err != nil {
				c.gate.Unlock()
				c.directory.Unlock()
				return err
			}
		}
		r.mu.Lock()
		delete(record.Readers, id)
		if record.Owner == id {
			record.Epoch++
			record.Owner, record.State = r.id, "M"
		}
		r.journal.Records[name] = record
		err := r.saveLocked()
		if err == nil {
			c.rights.publish(parseMosiPhase(record.State))
		}
		r.mu.Unlock()
		c.gate.Unlock()
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
		peers := make([]*replicationPeer, 0, len(r.peers)+1)
		for _, p := range r.peers {
			peers = append(peers, p)
		}
		if r.upstream != nil {
			peers = append(peers, r.upstream)
		}
		r.mu.Unlock()
		for _, p := range peers {
			p.close()
		}
		if r.server != nil {
			r.server.Shutdown(context.Background())
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
		panic(replicationNetworkError)
	}
	ctx, cancel := replicaPacketContext(peer)
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

func initReplicationAdmin(en *scm.Env) {
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
