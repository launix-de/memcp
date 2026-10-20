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

import "io"
import "os"
import "fmt"
import "net"
import "sync"
import "time"
import "bytes"
import "context"
import "strconv"
import "strings"
import "net/http"
import "crypto/sha1"
import "sync/atomic"
import "compress/gzip"
import "crypto/subtle"
import "encoding/json"
import "path/filepath"
import "encoding/base64"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

const replicationNetworkError = "Cluster network error"

// Replication is published once at startup, after frontend registration. It is
// nil for ordinary instances. Directory updates are serialized per database;
// its journal is fsynced before a new owner may execute. Local admission holds
// a read lease per transaction, never per row. The first implementation groups
// a schema and its shards into one ownership unit; storage backends are unchanged.
var replication atomic.Pointer[replicationManager]

type replicaAuth struct {
	Hash      string          `json:"hash"`
	Admin     bool            `json:"admin"`
	Databases map[string]bool `json:"databases"`
}

type replicaRecord struct {
	Epoch   uint64          `json:"epoch"`
	State   string          `json:"state"`
	Owner   string          `json:"owner,omitempty"`
	Readers map[string]bool `json:"readers,omitempty"`
}

type replicaJournal struct {
	Records map[string]replicaRecord `json:"records"`
	Users   map[string]replicaAuth   `json:"users"`
	Retired map[string]bool          `json:"retired"`
}

type replicaCell struct {
	// gate protects active local statements against an incoming ownership
	// change. Normal statements share the lease, including local writes.
	gate       sync.RWMutex
	state      atomic.Pointer[string]
	directory  sync.Mutex
	checkpoint sync.Mutex
	acquire    sync.Mutex
}

type replicationManager struct {
	master        bool
	address       string
	id            string
	mu            sync.Mutex
	authMu        sync.Mutex
	authInstalled map[string]string
	cells         map[string]*replicaCell
	journal       replicaJournal
	peers         map[string]*replicationPeer
	upstream      *replicationPeer
	server        *http.Server
	done          chan struct{}
	password      scm.Scmer
	policy        scm.Scmer
	shutdown      func()
	stopping      bool
	stopOnce      sync.Once
}

type replicaRequest struct {
	Epoch    uint64 `json:"epoch,omitempty"`
	Database string `json:"database"`
	User     string `json:"user,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Read     bool   `json:"read,omitempty"`
	State    string `json:"state,omitempty"`
	Dump     []byte `json:"dump,omitempty"`
	Salt     []byte `json:"salt,omitempty"`
	Scramble []byte `json:"scramble,omitempty"`
}

type replicaDump struct {
	Schema []byte            `json:"schema"`
	Files  map[string][]byte `json:"files"`
	Blobs  map[string][]byte `json:"blobs"`
}

type replicaGrant struct {
	Epoch uint64 `json:"epoch"`
	State string `json:"state"`
	Dump  []byte `json:"dump,omitempty"`
}

// StartReplication accepts a port (master) or host:port (slave). A slave uses
// only outbound connections, so reverse transfers also work behind NAT.
func StartReplication(address string, shutdown func()) error {
	if address == "" {
		return nil
	}
	r := &replicationManager{address: address, cells: make(map[string]*replicaCell), peers: make(map[string]*replicationPeer), done: make(chan struct{})}
	r.shutdown = shutdown
	r.master = !strings.Contains(address, ":")
	if r.master {
		port, err := strconv.Atoi(address)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid replication port")
		}
	} else if _, _, err := net.SplitHostPort(address); err != nil {
		return fmt.Errorf("invalid replication address: %w", err)
	}
	r.password = scm.Globalenv.FindRead("mysql_auth").Vars["mysql_auth"]
	r.policy = scm.Globalenv.FindRead("sql_policy").Vars["sql_policy"]
	if r.password.IsNil() || r.policy.IsNil() {
		return fmt.Errorf("replication requires the SQL authentication frontend")
	}
	if Settings.ReplicationUUID == "" {
		Settings.ReplicationUUID = uuid.NewString()
		data, err := json.Marshal(Settings)
		if err != nil {
			return err
		}
		if err := writeReplicationJournal(filepath.Join(Basepath, "settings.json"), data); err != nil {
			return err
		}
	}
	r.id = Settings.ReplicationUUID
	if _, err := uuid.Parse(r.id); err != nil {
		return fmt.Errorf("invalid saved replication identity")
	}
	r.journal = replicaJournal{Records: make(map[string]replicaRecord), Users: make(map[string]replicaAuth), Retired: make(map[string]bool)}
	data, err := os.ReadFile(filepath.Join(Basepath, ".replication-state"))
	if err == nil {
		if err := json.Unmarshal(data, &r.journal); err != nil {
			return fmt.Errorf("invalid replication journal: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if r.journal.Records == nil || r.journal.Users == nil || r.journal.Retired == nil {
		return fmt.Errorf("invalid replication journal directories")
	}
	for name, record := range r.journal.Records {
		r.cell(name).state.Store(&record.State)
	}
	if r.master {
		mux := http.NewServeMux()
		mux.HandleFunc("/replication", r.accept)
		listener, err := net.Listen("tcp", ":"+address)
		if err != nil {
			return err
		}
		r.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		replication.Store(r)
		go func() {
			if err := r.server.Serve(listener); err != nil && err != http.ErrServerClosed {
				scm.PrintError(err)
			}
		}()
	} else {
		replication.Store(r)
		go r.connect()
	}
	return nil
}

func writeReplicationJournal(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".replication-tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (r *replicationManager) saveLocked() error {
	data, err := json.Marshal(r.journal)
	if err != nil {
		return err
	}
	return writeReplicationJournal(filepath.Join(Basepath, ".replication-state"), data)
}

func (r *replicationManager) cell(name string) *replicaCell {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.cells[name]
	if c == nil {
		c = &replicaCell{}
		r.cells[name] = c
	}
	return c
}

func (r *replicationManager) setLocal(name, state string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.journal.Records[name]
	record.State = state
	r.journal.Records[name] = record
	if err := r.saveLocked(); err != nil {
		return err
	}
	r.cells[name].state.Store(&state)
	return nil
}

func (r *replicationManager) accept(w http.ResponseWriter, req *http.Request) {
	id := req.Header.Get("X-MemCP-Replica")
	if id == r.id {
		http.Error(w, "master identity cannot be used by a slave", 400)
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid replica identity", 400)
		return
	}
	r.mu.Lock()
	retired := r.journal.Retired[id]
	stopping := r.stopping
	old := r.peers[id]
	r.mu.Unlock()
	if retired || stopping {
		http.Error(w, "replica retired by administrator", 403)
		return
	}
	if old != nil {
		http.Error(w, "replica already connected", 409)
		return
	}
	conn, err := (&websocket.Upgrader{CheckOrigin: func(req *http.Request) bool { return req.Header.Get("Origin") == "" }}).Upgrade(w, req, nil)
	if err != nil {
		return
	}
	r.mu.Lock()
	if r.peers[id] != nil || r.stopping {
		r.mu.Unlock()
		conn.Close()
		return
	}
	p := newReplicationPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) {
		return r.handleMaster(ctx, id, method, data)
	})
	r.peers[id] = p
	r.mu.Unlock()
	<-p.done
	p.wait()
	r.mu.Lock()
	if r.peers[id] == p {
		delete(r.peers, id)
	}
	r.mu.Unlock()
}

func (r *replicationManager) connect() {
	for {
		select {
		case <-r.done:
			return
		default:
		}
		header := http.Header{"X-Memcp-Replica": []string{r.id}}
		conn, _, err := websocket.DefaultDialer.Dial("ws://"+r.address+"/replication", header)
		if err == nil {
			r.mu.Lock()
			if r.stopping {
				r.mu.Unlock()
				conn.Close()
				return
			}
			p := newReplicationPeer(conn, r.handleSlave)
			r.upstream = p
			r.mu.Unlock()
			go func() {
				r.mu.Lock()
				names := make([]string, 0, len(r.journal.Records))
				for name, record := range r.journal.Records {
					if record.State == "M" {
						names = append(names, name)
					}
				}
				users := make(map[string]replicaAuth, len(r.journal.Users))
				for user, auth := range r.journal.Users {
					users[user] = auth
				}
				r.mu.Unlock()
				for _, name := range names {
					for user, auth := range users {
						if auth.Admin || auth.Databases[name] {
							replicationCheckpoint(name, user)
							break
						}
					}
				}
			}()
			<-p.done
			p.wait()
			r.mu.Lock()
			if r.upstream == p {
				r.upstream = nil
			}
			r.mu.Unlock()
		}
		select {
		case <-r.done:
			return
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (r *replicationManager) peer(id string) *replicationPeer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == "" {
		return r.upstream
	}
	return r.peers[id]
}

func (r *replicationManager) authenticate(user, hash string, salt, scramble []byte, requested string) (auth replicaAuth, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("authentication failed")
		}
	}()
	password := scm.Apply(r.password, scm.NewString(user))
	if password.IsNil() || !replicationPasswordProof([]byte(password.String()), hash, salt, scramble) {
		return auth, fmt.Errorf("authentication failed")
	}
	auth = replicaAuth{Hash: base64.StdEncoding.EncodeToString([]byte(password.String())), Databases: make(map[string]bool)}
	policy := scm.Apply(r.policy, scm.NewString(user))
	auth.Admin = func() (allowed bool) {
		defer func() { _ = recover() }()
		return scm.Apply(policy, scm.NewString("system"), scm.NewBool(true), scm.NewBool(true)).Bool()
	}()
	if auth.Admin {
		return auth, nil
	}
	if requested != "" {
		if scm.Apply(policy, scm.NewString(requested), scm.NewString("*"), scm.NewBool(true)).Bool() {
			auth.Databases[requested] = true
		}
		return auth, nil
	}
	// Query just this user's grants, not every tenant in the catalog.
	access := GetDatabase("system").GetTable("access")
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		newScanBoundarySpec("username", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false),
	})
	collect := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1]))
	})
	combine := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1].Slice()...))
	})
	rows := access.scan(nil, schema, []scm.Scmer{scm.NewString(user)}, nil, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"database"}, collect, scm.NewSlice(nil), combine, false)
	for _, name := range rows.Slice() {
		auth.Databases[name.String()] = true
	}
	return auth, nil
}

func (r *replicationManager) handleMaster(ctx context.Context, id, method string, data json.RawMessage) (any, error) {
	select {
	case <-r.done:
		return nil, fmt.Errorf(replicationNetworkError)
	default:
	}
	var req replicaRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	r.mu.Lock()
	retired := r.journal.Retired[id]
	r.mu.Unlock()
	if retired {
		return nil, fmt.Errorf("replica retired by administrator")
	}
	auth, err := r.authenticate(req.User, req.Hash, req.Salt, req.Scramble, req.Database)
	if err != nil {
		return nil, err
	}
	if method == "login" {
		return auth, nil
	}
	if !auth.Admin && !auth.Databases[req.Database] {
		return nil, fmt.Errorf("access denied for database %s", req.Database)
	}
	switch method {
	case "claim":
		return r.claim(ctx, id, req)
	case "backup":
		c := r.cell(req.Database)
		c.directory.Lock()
		defer c.directory.Unlock()
		r.mu.Lock()
		record := r.journal.Records[req.Database]
		r.mu.Unlock()
		if record.Owner != id || record.Epoch != req.Epoch {
			return nil, fmt.Errorf("replica is no longer the modifier")
		}
		c.gate.Lock()
		defer c.gate.Unlock()
		return nil, importReplicaDatabase(req.Database, req.Dump)
	default:
		return nil, fmt.Errorf("unknown replication method")
	}
}

func (r *replicationManager) handleSlave(ctx context.Context, method string, data json.RawMessage) (any, error) {
	if method == "retire" {
		r.mu.Lock()
		r.journal.Retired[r.id] = true
		err := r.saveLocked()
		r.mu.Unlock()
		if err != nil {
			return nil, err
		}
		if r.shutdown != nil {
			time.AfterFunc(50*time.Millisecond, r.shutdown)
		}
		return true, nil
	}
	var req replicaRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	if !validReplicaDatabase(req.Database) {
		return nil, fmt.Errorf("invalid replication database")
	}
	c := r.cell(req.Database)
	if err := replicationLock(ctx, &c.gate); err != nil {
		return nil, err
	}
	defer c.gate.Unlock()
	switch method {
	case "offer":
		r.mu.Lock()
		epoch := r.journal.Records[req.Database].Epoch
		r.mu.Unlock()
		if epoch != req.Epoch {
			return nil, fmt.Errorf("stale ownership transfer")
		}
		dump, err := exportReplicaDatabase(req.Database)
		if err != nil {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf(replicationNetworkError)
		}
		if err := r.setLocal(req.Database, req.State); err != nil {
			return nil, err
		}
		return replicaGrant{State: req.State, Dump: dump}, nil
	case "invalidate":
		return nil, r.setLocal(req.Database, "I")
	default:
		return nil, fmt.Errorf("unknown replication callback")
	}
}

func validReplicaDatabase(name string) bool {
	return name != "" && name != "system" && !strings.HasPrefix(name, "system_") && !strings.ContainsAny(name, "/\\\x00") && name != "." && name != ".."
}

func replicationLock(ctx context.Context, gate *sync.RWMutex) error {
	for !gate.TryLock() {
		select {
		case <-ctx.Done():
			return fmt.Errorf(replicationNetworkError)
		case <-time.After(time.Millisecond):
		}
	}
	if ctx.Err() != nil {
		gate.Unlock()
		return fmt.Errorf(replicationNetworkError)
	}
	return nil
}

func (r *replicationManager) claim(ctx context.Context, id string, req replicaRequest) (any, error) {
	if !validReplicaDatabase(req.Database) {
		return nil, fmt.Errorf("invalid replication database")
	}
	c := r.cell(req.Database)
	c.directory.Lock()
	defer c.directory.Unlock()
	r.mu.Lock()
	record, exists := r.journal.Records[req.Database]
	readers := make(map[string]bool, len(record.Readers))
	for reader, present := range record.Readers {
		readers[reader] = present
	}
	record.Readers = readers
	r.mu.Unlock()
	if !exists {
		record = replicaRecord{State: "M", Owner: r.id, Epoch: 1}
		r.mu.Lock()
		r.journal.Records[req.Database] = record
		err := r.saveLocked()
		r.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	if record.Readers == nil {
		record.Readers = make(map[string]bool)
	}
	var grant replicaGrant
	grant.State = "M"
	if req.Read {
		grant.State = "S"
	}
	if record.Owner != id {
		oldState := "I"
		if req.Read {
			oldState = "O"
		}
		if record.Owner == r.id {
			if err := replicationLock(ctx, &c.gate); err != nil {
				return nil, err
			}
			dump, err := exportReplicaDatabase(req.Database)
			if err == nil {
				err = r.setLocal(req.Database, oldState)
			}
			c.gate.Unlock()
			if err != nil {
				return nil, err
			}
			grant.Dump = dump
		} else {
			peer := r.peer(record.Owner)
			if peer == nil {
				return nil, fmt.Errorf(replicationNetworkError)
			}
			if err := peer.call(ctx, "offer", replicaRequest{Database: req.Database, State: oldState, Epoch: record.Epoch}, &grant); err != nil {
				return nil, fmt.Errorf(replicationNetworkError)
			}
			grant.State = "M"
			if req.Read {
				grant.State = "S"
			}
			// The master keeps the complete transferred generation, regardless
			// of whether it currently holds a read capability.
			if err := replicationLock(ctx, &c.gate); err != nil {
				return nil, err
			}
			err := importReplicaDatabase(req.Database, grant.Dump)
			c.gate.Unlock()
			if err != nil {
				return nil, err
			}
		}
	} else if id != r.id && req.Epoch == record.Epoch {
		// A timed-out request may invalidate the modifier after the requester
		// has abandoned its reply. The still-current epoch identifies the
		// modifier's own disk as authoritative, even if its local flag is I.
		// Never overwrite that disk from the master's older backup.
		peer := r.peer(id)
		if peer == nil {
			return nil, fmt.Errorf(replicationNetworkError)
		}
		if err := peer.call(ctx, "offer", replicaRequest{Database: req.Database, State: "I", Epoch: record.Epoch}, &grant); err != nil {
			return nil, err
		}
		if err := replicationLock(ctx, &c.gate); err != nil {
			return nil, err
		}
		err := importReplicaDatabase(req.Database, grant.Dump)
		c.gate.Unlock()
		if err != nil {
			return nil, err
		}
		grant.State = "M"
	} else {
		// An uninstalled grant has a newer epoch than the receiver's journal;
		// its complete payload is recoverable from the master backup.
		if err := replicationLock(ctx, &c.gate); err != nil {
			return nil, err
		}
		dump, err := exportReplicaDatabase(req.Database)
		c.gate.Unlock()
		if err != nil {
			return nil, err
		}
		grant.Dump = dump
	}
	if req.Read {
		record.Readers[id] = true
	} else {
		for reader := range record.Readers {
			if reader == id || reader == r.id {
				continue
			}
			p := r.peer(reader)
			if p == nil || p.call(ctx, "invalidate", replicaRequest{Database: req.Database}, nil) != nil {
				return nil, fmt.Errorf(replicationNetworkError)
			}
		}
		record.Readers = make(map[string]bool)
		record.Epoch++
		record.Owner = id
	}
	r.mu.Lock()
	if r.journal.Retired[id] {
		r.mu.Unlock()
		return nil, fmt.Errorf("replica retired by administrator")
	}
	record.State = "I"
	if record.Owner == r.id {
		record.State = "M"
		if req.Read {
			record.State = "O"
		}
	} else if req.Read && id == r.id {
		record.State = "S"
	}
	r.journal.Records[req.Database] = record
	grant.Epoch = record.Epoch
	err := r.saveLocked()
	if err == nil {
		c.state.Store(&record.State)
	}
	r.mu.Unlock()
	return grant, err
}

// replicationEnter conservatively acquires write ownership for the schema and
// its physical shards before planning. This prevents cached plans retaining a
// replaced catalog. The lease spans an explicit transaction, including pauses
// between statements; a remote handover must not serialize uncommitted rows.
func replicationEnter(tx *TxContext, session scm.Scmer) {
	r := replication.Load()
	if r == nil {
		return
	}
	name := session.Func()(scm.NewString("schema")).String()
	if tx.replicationRelease != nil {
		if tx.replicationDatabase != name {
			panic("cross-database transactions are not supported by replication")
		}
		return
	}
	r.mu.Lock()
	retired := !r.master && r.journal.Retired[r.id]
	r.mu.Unlock()
	if retired {
		panic("replica retired by administrator")
	}
	if !validReplicaDatabase(name) {
		return
	}
	c := r.cell(name)
	for {
		c.gate.RLock()
		state := c.state.Load()
		if state != nil && *state == "M" {
			tx.replicationDatabase = name
			tx.replicationRelease = func() { c.gate.RUnlock(); tx.replicationDatabase = "" }
			return
		}
		// A fresh master owns its existing databases until explicitly delegated.
		if state == nil && r.master {
			c.gate.RUnlock()
			c.gate.Lock()
			if c.state.Load() == nil {
				r.mu.Lock()
				r.journal.Records[name] = replicaRecord{State: "M", Owner: r.id, Epoch: 1}
				err := r.saveLocked()
				r.mu.Unlock()
				if err != nil {
					c.gate.Unlock()
					panic(err)
				}
				value := "M"
				c.state.Store(&value)
			}
			c.gate.Unlock()
			continue
		}
		c.gate.RUnlock()
		c.acquire.Lock()
		if state := c.state.Load(); state != nil && *state == "M" {
			c.acquire.Unlock()
			continue
		}
		user := session.Func()(scm.NewString("username")).String()
		var grant replicaGrant
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var err error
		r.mu.Lock()
		epoch := r.journal.Records[name].Epoch
		r.mu.Unlock()
		if r.master {
			var value any
			value, err = r.claim(ctx, r.id, replicaRequest{Database: name, Epoch: epoch})
			if err == nil {
				grant = value.(replicaGrant)
			}
		} else {
			r.mu.Lock()
			auth, ok := r.journal.Users[user]
			r.mu.Unlock()
			if !ok || (!auth.Admin && !auth.Databases[name]) {
				cancel()
				c.acquire.Unlock()
				panic("replication database access denied")
			}
			peer := r.peer("")
			if peer == nil {
				cancel()
				c.acquire.Unlock()
				panic(replicationNetworkError)
			}
			err = peer.call(ctx, "claim", replicaRequest{Database: name, User: user, Hash: auth.Hash, Epoch: epoch}, &grant)
		}
		cancel()
		if err != nil {
			c.acquire.Unlock()
			panic(replicationNetworkError)
		}
		c.gate.Lock()
		if err := importReplicaDatabase(name, grant.Dump); err != nil {
			c.gate.Unlock()
			c.acquire.Unlock()
			panic(err)
		}
		r.mu.Lock()
		localRecord := r.journal.Records[name]
		localRecord.Epoch = grant.Epoch
		r.journal.Records[name] = localRecord
		r.mu.Unlock()
		if err := r.setLocal(name, grant.State); err != nil {
			c.gate.Unlock()
			c.acquire.Unlock()
			panic(err)
		}
		c.gate.Unlock()
		c.acquire.Unlock()
	}
}

func replicationCheckTable(t *table, tx *TxContext) {
	if replication.Load() != nil && tx != nil && tx.replicationRelease != nil && validReplicaDatabase(t.schema.Name) && tx.replicationDatabase != t.schema.Name {
		panic("cross-database statements are not supported by replication")
	}
}

func replicationOpenDatabase(user, database string) (err error) {
	if replication.Load() == nil || !validReplicaDatabase(database) {
		return nil
	}
	tx := &TxContext{}
	defer func() {
		if tx.replicationRelease != nil {
			tx.replicationRelease()
		}
		if value := recover(); value != nil {
			err = fmt.Errorf("%v", value)
		}
	}()
	session := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		switch a[0].String() {
		case "username":
			return scm.NewString(user)
		case "schema":
			return scm.NewString(database)
		}
		return scm.NewNil()
	})
	replicationEnter(tx, session)
	return nil
}

func replicationSessionClosed(session scm.Scmer) {
	if replication.Load() == nil {
		return
	}
	value := session.Func()(scm.NewString("__memcp_tx"))
	if value.IsNil() {
		return
	}
	tx, ok := value.Any().(*TxContext)
	if !ok {
		return
	}
	tx.queryMu.Lock()
	defer tx.queryMu.Unlock()
	defer func() {
		if tx.replicationRelease != nil {
			tx.replicationRelease()
			tx.replicationRelease = nil
		}
		if err := recover(); err != nil {
			scm.PrintError(err)
		}
	}()
	if tx.State == TxActive {
		tx.Rollback()
	}
}

func exportReplicaDatabase(name string) (data []byte, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("replication export: %v", p)
		}
	}()
	db := GetDatabase(name)
	if db == nil {
		return nil, fmt.Errorf("database does not exist")
	}
	db.ensureLoaded()
	for {
		errors, complete := rebuildDatabaseAndCompact(db, true, false, false)
		if len(errors) != 0 {
			return nil, fmt.Errorf("replication rebuild: %s", strings.Join(errors, "; "))
		}
		if complete {
			break
		}
		// A best-effort maintenance pass may skip a table already rebuilding.
		// Wait without DDL ownership, then force a complete generation. Never
		// drop its still-authoritative WAL from an incomplete transfer.
		for _, t := range db.tables.GetAll() {
			t.maintenanceMu.Lock()
			t.maintenanceMu.Unlock()
		}
	}
	db.persistenceLifecycle.Lock()
	defer db.persistenceLifecycle.Unlock()
	dump := replicaDump{Schema: db.persistence.ReadSchema(), Files: make(map[string][]byte), Blobs: make(map[string][]byte)}
	read := func(reader io.ReadCloser) []byte {
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			panic(err)
		}
		return data
	}
	db.persistence.WalkShardFiles(func(name string) {
		if !isPersistenceLogObject(name) {
			dump.Files[name] = read(db.persistence.ReadShardFile(name))
		}
	})
	db.persistence.WalkBlobs(func(hash string) { dump.Blobs[hash] = read(db.persistence.ReadBlob(hash)) })
	var packed bytes.Buffer
	writer := gzip.NewWriter(&packed)
	if err := json.NewEncoder(writer).Encode(dump); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return packed.Bytes(), nil
}

func importReplicaDatabase(name string, data []byte) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("replication import: %v", p)
		}
	}()
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer reader.Close()
	var dump replicaDump
	if err := json.NewDecoder(reader).Decode(&dump); err != nil {
		return err
	}
	var schema struct{ Name string }
	if err := json.Unmarshal(dump.Schema, &schema); err != nil || schema.Name != name {
		return fmt.Errorf("invalid replication schema")
	}
	db := GetDatabase(name)
	var engine PersistenceEngine
	if db != nil {
		db.persistenceLifecycle.Lock()
		defer db.persistenceLifecycle.Unlock()
		db.save() // drain queued buffered schema publications before replacement
		db.closeTransactionLog()
		engine = db.persistence
	} else {
		// A preconfigured destination selects its own backend and credentials.
		engine = createPersistenceFromConfig(name, databaseBackendConfig(name))
	}
	write := func(w io.WriteCloser, data []byte) {
		if _, err := w.Write(data); err != nil {
			abortPersistenceWrite(w)
			panic(err)
		}
		finishColumnWrite(w, true)
	}
	for object, data := range dump.Files {
		if strings.ContainsAny(object, "/\\\x00") || strings.HasPrefix(object, ".") {
			return fmt.Errorf("invalid replication object")
		}
		write(engine.WriteShardFile(object), data)
	}
	for hash, data := range dump.Blobs {
		if strings.ContainsAny(hash, "/\\\x00") || strings.HasPrefix(hash, ".") {
			return fmt.Errorf("invalid replication blob")
		}
		write(engine.WriteBlob(hash), data)
	}
	// New shard UUIDs have no source WALs. Keep old coordinator records until
	// ordinary compaction: deleting them before atomic schema publication could
	// destroy recovery of the previous generation after a process crash.
	engine.WriteSchema(dump.Schema)
	replacement := newDatabase()
	replacement.Name, replacement.persistence, replacement.srState = name, engine, COLD
	replacement.ensureLoaded()
	databases.Set(name, replacement)
	return nil
}

// replicationCheckpoint is best effort while the master is offline. Local
// SAFE commit remains durable independently. Reconnection uploads the newest
// complete rebuild again; an unavailable master never revokes local M.
func replicationCheckpoint(name, user string) {
	r := replication.Load()
	if r == nil || r.master || !validReplicaDatabase(name) {
		return
	}
	peer := r.peer("")
	if peer == nil {
		return
	}
	c := r.cell(name)
	if !c.checkpoint.TryLock() {
		return
	}
	defer c.checkpoint.Unlock()
	if !c.gate.TryLock() {
		return
	}
	state := c.state.Load()
	if state == nil || *state != "M" {
		c.gate.Unlock()
		return
	}
	dump, err := exportReplicaDatabase(name)
	r.mu.Lock()
	epoch := r.journal.Records[name].Epoch
	r.mu.Unlock()
	c.gate.Unlock()
	if err != nil {
		scm.PrintError(err)
		return
	}
	r.mu.Lock()
	auth := r.journal.Users[user]
	r.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := peer.call(ctx, "backup", replicaRequest{Database: name, User: user, Hash: auth.Hash, Dump: dump, Epoch: epoch}, nil); err != nil {
		scm.PrintError(fmt.Errorf("replication checkpoint deferred: %w", err))
	}
}

// replicationHTTPAuthentication forwards an online login to the master before
// the ordinary frontend sees it. Its cache is private to this installation;
// the master never trusts locally edited access rows.
func replicationHTTPAuthentication(user, password string) bool {
	if r := replication.Load(); r == nil || r.master {
		return true
	}
	hash := base64.StdEncoding.EncodeToString([]byte(scm.MySQLPassword(scm.NewString(password)).String()))
	return replicationAuthentication(replicaRequest{User: user, Hash: hash})
}

func replicationMySQLAuthentication(user string, salt, scramble []byte) bool {
	return replicationAuthentication(replicaRequest{User: user, Salt: salt, Scramble: scramble})
}

func replicationPasswordProof(raw []byte, hash string, salt, scramble []byte) bool {
	if hash != "" {
		return subtle.ConstantTimeCompare([]byte(base64.StdEncoding.EncodeToString(raw)), []byte(hash)) == 1
	}
	if len(raw) != sha1.Size || len(scramble) != sha1.Size || len(salt) != 20 {
		return false
	}
	second := sha1.Sum(raw)
	challenge := sha1.New()
	challenge.Write(salt)
	challenge.Write(second[:])
	expected := challenge.Sum(nil)
	for i := range expected {
		expected[i] ^= raw[i]
	}
	return subtle.ConstantTimeCompare(expected, scramble) == 1
}

func replicationAuthentication(proof replicaRequest) bool {
	r := replication.Load()
	if r == nil || r.master {
		return true
	}
	var auth replicaAuth
	if peer := r.peer(""); peer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := peer.call(ctx, "login", proof, &auth)
		cancel()
		if err != nil {
			if _, denied := err.(replicationRemoteError); denied {
				return false
			}
			r.mu.Lock()
			auth = r.journal.Users[proof.User]
			r.mu.Unlock()
		} else {
			r.mu.Lock()
			previous, exists := r.journal.Users[proof.User]
			r.journal.Users[proof.User] = auth
			old, _ := json.Marshal(previous)
			current, _ := json.Marshal(auth)
			if !exists || !bytes.Equal(old, current) {
				err = r.saveLocked()
			}
			r.mu.Unlock()
			if err != nil {
				return false
			}
		}
	} else {
		r.mu.Lock()
		auth = r.journal.Users[proof.User]
		r.mu.Unlock()
	}
	rawHash, err := base64.StdEncoding.DecodeString(auth.Hash)
	if err != nil || !replicationPasswordProof(rawHash, proof.Hash, proof.Salt, proof.Scramble) {
		return false
	}
	r.authMu.Lock()
	defer r.authMu.Unlock()
	encoded, _ := json.Marshal(auth)
	if r.authInstalled == nil {
		r.authInstalled = make(map[string]string)
	}
	if r.authInstalled[proof.User] == string(encoded) {
		return true
	}
	// Publish only this proven user's credentials through the existing native
	// table APIs, so SQL/HTTP/MySQL use the ordinary authentication callbacks.
	db := GetDatabase("system")
	users := db.GetTable("user")
	row := []scm.Scmer{scm.NewString(proof.User), scm.NewString(string(rawHash)), scm.NewBool(auth.Admin)}
	update := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Apply(a[0], scm.NewSlice([]scm.Scmer{scm.NewString("password"), row[1], scm.NewString("admin"), row[2]}))
	})
	users.Insert([]string{"username", "password", "admin"}, [][]scm.Scmer{row}, []string{"$update"}, update, false, nil)
	access := db.GetTable("access")
	for name := range auth.Databases {
		access.Insert([]string{"username", "database"}, [][]scm.Scmer{{scm.NewString(proof.User), scm.NewString(name)}}, nil, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewInt(0) }), false, nil)
	}
	r.authInstalled[proof.User] = string(encoded)
	return true
}
