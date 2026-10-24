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
import "time"
import "context"
import "strconv"
import "strings"
import "net/http"
import "sync/atomic"
import "encoding/json"
import "path/filepath"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

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
	peers         map[string]*mosiPeer
	upstream      *mosiPeer
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

// StartReplication accepts a port (master) or host:port (slave). A slave uses
// only outbound connections, so reverse transfers also work behind NAT.
func StartReplication(address string, shutdown func()) error {
	if address == "" {
		return nil
	}
	r := &replicationManager{address: address, cells: make(map[string]*replicaCell), peers: make(map[string]*mosiPeer), done: make(chan struct{})}
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
		c.rights.RestoreBackend(mosi.ParsePhase(record.State))
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
	p := newMosiPeer(conn, func(ctx context.Context, method string, data json.RawMessage) (any, error) {
		return r.handleMaster(ctx, id, method, data)
	}, func(packet *mosiPacket) error { return r.receivePacket(id, packet) })
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
			p := newMosiPeer(conn, r.handleSlave, func(packet *mosiPacket) error { return r.receivePacket("", packet) })
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

func (r *replicationManager) peer(id string) *mosiPeer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == "" {
		return r.upstream
	}
	return r.peers[id]
}

func validReplicaDatabase(name string) bool {
	return name != "" && name != "system" && !strings.HasPrefix(name, "system_") && !strings.ContainsAny(name, "/\\\x00") && name != "." && name != ".."
}

func replicationLock(ctx context.Context, gate *sync.RWMutex) error {
	for !gate.TryLock() {
		select {
		case <-ctx.Done():
			return fmt.Errorf(mosi.NetworkError)
		case <-time.After(time.Millisecond):
		}
	}
	if ctx.Err() != nil {
		gate.Unlock()
		return fmt.Errorf(mosi.NetworkError)
	}
	return nil
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
