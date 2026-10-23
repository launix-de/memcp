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

import "os"
import "fmt"
import "net"
import "sync"
import "weak"
import "bytes"
import "time"
import "context"
import "strconv"
import "strings"
import "net/http"
import "crypto/sha1"
import "sync/atomic"
import "crypto/subtle"
import "encoding/json"
import "path/filepath"
import "encoding/base64"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

const replicationNetworkError = "Cluster network error"

// Replication is published once at startup, after frontend registration. It is
// nil for ordinary instances. Directory updates are serialized per resource;
// its journal is fsynced before a new owner may execute. Only write transactions
// retain a local gate; reads use ordinary snapshots. Each database schema and
// each data shard have independent MOSI slots; storage backends are unchanged.
var replication atomic.Pointer[replicationManager]

// Session getters borrow immutable arguments; reusing the key avoids a
// variadic argument allocation at every statement boundary.
var replicationSchemaArgument = []scm.Scmer{scm.NewString("schema")}
var replicationUsernameArgument = []scm.Scmer{scm.NewString("username")}

type replicaAuth struct {
	Hash      string          `json:"hash"`
	Admin     bool            `json:"admin"`
	Databases map[string]bool `json:"databases"`
}

type replicaRecord struct {
	InitialClaimed bool                          `json:"initial_claimed,omitempty"`
	Epoch          uint64                        `json:"epoch"`
	State          string                        `json:"state"`
	Owner          string                        `json:"owner,omitempty"`
	Readers        map[string]bool               `json:"readers,omitempty"`
	Resource       replicaResource               `json:"resource"`
	Generation     string                        `json:"generation,omitempty"`
	Requests       map[string]replicaRequestMark `json:"requests,omitempty"`
}

type replicaRequestMark struct {
	Sequence uint64 `json:"sequence"`
	Write    bool   `json:"write"`
}

type replicaJournal struct {
	Version     int                      `json:"version"`
	Records     map[string]replicaRecord `json:"records"`
	Users       map[string]replicaAuth   `json:"users"`
	Retired     map[string]bool          `json:"retired"`
	Initialized map[string]bool          `json:"initialized,omitempty"`
	Sequence    uint64                   `json:"sequence,omitempty"`
}

type replicaCell struct {
	// gate protects uncommitted local writes against ownership transfer.
	// Analytics never participate: their existing snapshots remain valid.
	gate              sync.RWMutex // write transactions only; reads use existing snapshots
	rights            mosiState
	resource          atomic.Pointer[replicaResource]
	current           atomic.Pointer[storageShard]
	bootstrap         sync.Mutex
	initialized       atomic.Bool
	bootstrapStarted  bool
	directory         sync.Mutex
	checkpoint        sync.Mutex
	importPublication sync.Mutex // schema cell: import registration/publication versus GC
	acquire           sync.Mutex
	retiredShards     atomic.Pointer[[]weak.Pointer[storageShard]]
}

type replicationManager struct {
	master        bool
	address       string
	id            string
	mu            sync.Mutex
	authMu        sync.Mutex
	authInstalled map[string]string
	cells         map[string]*replicaCell
	cellSnapshot  atomic.Pointer[map[string]*replicaCell]
	retired       atomic.Bool
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
	steps         sync.Map // transfer/invalidations; values are *replicaStep
	imports       sync.Map // connection ID + transfer ID; reader-owned *replicaImport
}

type replicaRequest struct {
	SQL        string `json:"sql,omitempty"`
	PostgreSQL bool   `json:"postgresql,omitempty"`
	Epoch      uint64 `json:"epoch,omitempty"`
	Database   string `json:"database"`
	User       string `json:"user,omitempty"`
	Hash       string `json:"hash,omitempty"`
	Read       bool   `json:"read,omitempty"`
	State      string `json:"state,omitempty"`
	Salt       []byte `json:"salt,omitempty"`
	Scramble   []byte `json:"scramble,omitempty"`
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
	r.journal = replicaJournal{Version: 2, Records: make(map[string]replicaRecord), Users: make(map[string]replicaAuth), Retired: make(map[string]bool)}
	data, err := os.ReadFile(filepath.Join(Basepath, ".replication-state"))
	if err == nil {
		r.journal = replicaJournal{}
		if err := json.Unmarshal(data, &r.journal); err != nil {
			return fmt.Errorf("invalid replication journal: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if r.journal.Version != 2 {
		return fmt.Errorf("unsupported replication journal version: resynchronize the experimental database-granular installation explicitly")
	}
	if r.journal.Records == nil || r.journal.Users == nil || r.journal.Retired == nil {
		return fmt.Errorf("invalid replication journal directories")
	}
	r.retired.Store(r.journal.Retired[r.id])
	for name, record := range r.journal.Records {
		c := &replicaCell{}
		resource := record.Resource
		if !resource.valid() || resource.key() != name {
			return fmt.Errorf("invalid replication journal resource")
		}
		c.resource.Store(&resource)
		c.initialized.Store(r.journal.Initialized[resource.Database])
		c.rights.restoreBackend(parseMosiPhase(record.State))
		r.cells[name] = c
	}
	// Startup constructs the directory unpublished; avoid copying it once per
	// tenant. Runtime additions publish one new immutable lookup snapshot.
	snapshot := make(map[string]*replicaCell, len(r.cells))
	for name, c := range r.cells {
		snapshot[name] = c
	}
	r.cellSnapshot.Store(&snapshot)
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
	if snapshot := r.cellSnapshot.Load(); snapshot != nil {
		if c := (*snapshot)[name]; c != nil {
			return c
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.cells[name]
	if c == nil {
		c = &replicaCell{}
		r.cells[name] = c
		snapshot := make(map[string]*replicaCell, len(r.cells))
		for name, cell := range r.cells {
			snapshot[name] = cell
		}
		r.cellSnapshot.Store(&snapshot)
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
	r.cells[name].rights.publish(parseMosiPhase(state))
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
	if retired {
		http.Error(w, "replica retired by administrator", http.StatusForbidden)
		return
	}
	if stopping {
		http.Error(w, "replication master is stopping", http.StatusServiceUnavailable)
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
	}, func(packet *replicaPacket) error { return r.receivePacket(id, packet) })
	r.peers[id] = p
	r.mu.Unlock()
	<-p.done
	p.wait()
	r.abortImports(id)
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
		conn, response, err := websocket.DefaultDialer.Dial("ws://"+r.address+"/replication", header)
		if err != nil && response != nil {
			response.Body.Close()
			if response.StatusCode == http.StatusForbidden {
				_, _ = r.handleSlave(context.Background(), "retire", nil)
				return
			}
		}
		if err == nil {
			r.mu.Lock()
			if r.stopping {
				r.mu.Unlock()
				conn.Close()
				return
			}
			p := newReplicationPeer(conn, r.handleSlave, func(packet *replicaPacket) error { return r.receivePacket("", packet) })
			r.upstream = p
			r.mu.Unlock()
			go func() {
				r.mu.Lock()
				names := make(map[string]bool)
				for _, record := range r.journal.Records {
					if record.State == "M" {
						names[record.Resource.Database] = true
					}
				}
				users := make(map[string]replicaAuth, len(r.journal.Users))
				for user, auth := range r.journal.Users {
					users[user] = auth
				}
				r.mu.Unlock()
				for name := range names {
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
			r.abortImports("")
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
	if method == "admin" {
		if !auth.Admin {
			return nil, fmt.Errorf("administrator access required")
		}
		return r.executeAdmin(req)
	}
	if !auth.Admin && !auth.Databases[req.Database] {
		return nil, fmt.Errorf("access denied for database %s", req.Database)
	}
	return nil, fmt.Errorf("unknown authentication method")
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
		r.retired.Store(true)
		if r.shutdown != nil {
			time.AfterFunc(50*time.Millisecond, r.shutdown)
		}
		return true, nil
	}
	return nil, fmt.Errorf("unknown administrative callback")
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

// Reads use the engine's ordinary immutable shard/catalog snapshots. They do
// not register admission, increment a counter, hold a lock, or rebuild/checkpoint.
// Only a write transaction retains the local write-side gate: handover must not
// serialize uncommitted deltas. Shared ownership alone never blocks analytics.
func replicationEnter(tx *TxContext, session scm.Scmer) {
	replicationEnterMode(tx, session, false)
}

func replicationLeave(tx *TxContext) {
	tx.mu.Lock()
	cells := tx.replicationCells
	tx.replicationCells = nil
	tx.mu.Unlock()
	for cell := range cells {
		cell.gate.RUnlock()
	}
	tx.replicationWriting, tx.replicationActive = false, false
}

func replicaCanRead(state *string) bool {
	return state != nil && (*state == "M" || *state == "O" || *state == "S")
}

func replicationEnterMode(tx *TxContext, session scm.Scmer, write bool) {
	r := replication.Load()
	if r == nil {
		return
	}
	name := session.Func()(replicationSchemaArgument...).String()
	if !validReplicaDatabase(name) {
		return
	}
	r.firstDatabase(name, session)
	tx.replicationActive = true
	if write {
		tx.replicationWriting = true
	}
}

// Called before a parsed write plan executes, including on plan-cache hits.
func replicationWrite(tx *TxContext) {
	if replication.Load() != nil && tx != nil {
		tx.replicationWriting = true
	}
}

func replicationCheckWriteTable(t *table, tx *TxContext) {
	if replication.Load() == nil || !validReplicaDatabase(t.schema.Name) || t.isEphemeralQueryTable() {
		return
	}
	if t.schema.tables.Get(t.Name) != t {
		panic("replication schema changed; retry statement")
	}
}

// Weak retirement metadata is updated only at generation publication. A query
// retaining a historical shard keeps its files alive without registering reads.
func replicationRetainsCatalogs(name string) bool {
	if r := replication.Load(); r != nil {
		if cells := r.cellSnapshot.Load(); cells != nil {
			for _, cell := range *cells {
				resource := cell.resource.Load()
				if resource == nil || resource.Database != name {
					continue
				}
				if history := cell.retiredShards.Load(); history != nil {
					for _, previous := range *history {
						if previous.Value() != nil {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (c *replicaCell) retainGeneration(shard *storageShard) {
	if shard == nil {
		return
	}
	for {
		previous := c.retiredShards.Load()
		next := make([]weak.Pointer[storageShard], 0)
		if previous != nil {
			for _, entry := range *previous {
				if entry.Value() != nil {
					next = append(next, entry)
				}
			}
		}
		next = append(next, weak.Make(shard))
		if c.retiredShards.CompareAndSwap(previous, &next) {
			return
		}
	}
}

func replicationOpenDatabase(user, database string) (err error) {
	if replication.Load() == nil || !validReplicaDatabase(database) {
		return nil
	}
	tx := &TxContext{}
	defer func() {
		if tx.replicationActive {
			replicationLeave(tx)
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
		if tx.replicationActive {
			replicationLeave(tx)
			tx.replicationActive = false
		}
		if err := recover(); err != nil {
			scm.PrintError(err)
		}
	}()
	if tx.State == TxActive {
		tx.Rollback()
	}
}

// replicationCheckpoint is best effort while the master is offline. Local
// SAFE commit remains durable independently. Reconnection uploads the newest
// complete rebuild again; an unavailable master never revokes local M.
func replicationCheckpoint(name, user string) {
	r := replication.Load()
	if r != nil && !r.master && validReplicaDatabase(name) {
		r.backupDatabase(name, user)
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
