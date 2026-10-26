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
import "net"
import "sync"
import "time"
import "context"
import "strconv"
import "strings"
import "net/url"
import "net/http"
import "sync/atomic"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

// Replication is published once at startup, after frontend registration. It is
// nil for ordinary instances. Directory updates are serialized per resource;
// complete data installation precedes ownership publication. Only write transactions
// use MOSI local writer admission; reads use ordinary snapshots. Each database schema and
// each data shard have independent outward/local MOSI pairs; storage backends
// are unchanged. The current master flag selects the root/leaf replication
// adapter, not query permission. Cluster membership/routing will supply outer
// authority separately; engine admission must continue to use only ownstate.
var replication atomic.Pointer[replicationManager]
var replicationStarting atomic.Bool

// PrepareReplication prevents startup GC from deleting safety copies before
// runtime ownership is reconstructed. Called before catalog discovery starts.
func PrepareReplication() { replicationStarting.Store(true) }

// Session getters borrow immutable arguments; reusing the key avoids a
// variadic argument allocation at every statement boundary.
var replicationSchemaArgument = []scm.Scmer{scm.NewString("schema")}
var replicationUsernameArgument = []scm.Scmer{scm.NewString("username")}

type replicationManager struct {
	master          bool
	address         string
	id              string
	mu              sync.Mutex
	authMu          sync.Mutex
	authInstalled   map[string]string
	cells           map[string]*replicaCell
	cellSnapshot    atomic.Pointer[map[string]*replicaCell]
	retired         atomic.Bool
	directory       replicaDirectory
	config          *table
	backups         map[string]replicaBackup
	boot            string
	ready           atomic.Bool
	peersBoot       map[string]string
	peerReady       map[string]bool
	recoveryMu      sync.Mutex
	recoveredUsers  map[string]*mosiPeer
	restoring       map[string]bool
	confirmed       map[string]string
	initialClaims   map[string]map[string]bool
	recoveryReports map[string]map[string]replicaRecord
	peers           map[string]*mosiPeer
	upstream        *mosiPeer
	releaseRole     func() // startup-owned role; stopOnce removes it before peer shutdown
	done            chan struct{}
	password        scm.Scmer
	policy          scm.Scmer
	shutdown        func()
	stopping        bool
	stopOnce        sync.Once
	steps           sync.Map // transfer/invalidations; values are *replicaStep
	transfers       sync.Map // active outgoing *replicaTransfer -> database; cold GC lifetime only
	imports         sync.Map // connection ID + transfer ID; reader-owned *replicaImport
}

// StartReplication initializes engine adapters from trusted Scheme startup.
// Scheme constructs listeners and mounts node_protocol explicitly. A slave
// uses one outbound bidirectional connection, including behind NAT.
func StartReplication(address string, shutdown func()) error {
	if address == "" {
		return nil
	}
	if replication.Load() != nil {
		return fmt.Errorf("replication is already initialized")
	}
	r := &replicationManager{address: address, cells: make(map[string]*replicaCell), peers: make(map[string]*mosiPeer), done: make(chan struct{})}
	r.shutdown = shutdown
	r.master = address == "listen" || !strings.Contains(address, ":")
	if r.master && address != "listen" {
		port, err := strconv.Atoi(address)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid replication port")
		}
	} else if !r.master {
		var err error
		r.address, err = mosiEndpointURL(address)
		if err != nil {
			return err
		}
	}
	r.password = scm.Globalenv.FindRead("mysql_auth").Vars["mysql_auth"]
	r.policy = scm.Globalenv.FindRead("sql_policy").Vars["sql_policy"]
	if r.password.IsNil() || r.policy.IsNil() {
		return fmt.Errorf("replication requires the SQL authentication frontend")
	}
	PrepareReplication()
	var identityErr error
	r.id, identityErr = localNodeIdentity()
	if identityErr != nil {
		return identityErr
	}
	r.boot = uuid.NewString()
	r.backups = make(map[string]replicaBackup)
	r.restoring = make(map[string]bool)
	r.confirmed = make(map[string]string)
	r.initialClaims = make(map[string]map[string]bool)
	r.recoveryReports = make(map[string]map[string]replicaRecord)
	r.peersBoot = make(map[string]string)
	r.peerReady = make(map[string]bool)
	r.directory = replicaDirectory{Records: make(map[string]replicaRecord), Users: make(map[string]replicaAuth), Retired: make(map[string]bool), Initialized: make(map[string]bool)}
	if err := r.openConfig(); err != nil {
		return err
	}
	r.retired.Store(r.directory.Retired[r.id])
	for _, backup := range r.backups {
		r.restoring[backup.Resource.Database] = true
	}
	if r.master {
		r.ready.Store(true)
	}
	snapshot := make(map[string]*replicaCell)
	r.cellSnapshot.Store(&snapshot)
	if r.master {
		release, err := registerMosiRole("replication", http.HandlerFunc(r.accept))
		if err != nil {
			return err
		}
		r.releaseRole = release
		replication.Store(r)
	} else {
		replication.Store(r)
		go r.connect()
	}
	return nil
}

func mosiEndpointURL(address string) (string, error) {
	if !strings.Contains(address, "://") {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return "", fmt.Errorf("invalid node address")
		}
		address = "ws://" + address
	}
	u, err := url.Parse(address)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", fmt.Errorf("invalid node URL")
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("node URL requires HTTP(S) or WS(S)")
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = mosiHTTPPath
	}
	return u.String(), nil
}

func (r *replicationManager) accept(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "node protocol requires GET", http.StatusMethodNotAllowed)
		return
	}
	supported := false
	for _, protocol := range websocket.Subprotocols(req) {
		if protocol == mosiSubprotocol {
			supported = true
		}
	}
	if !supported {
		w.Header().Set("Sec-WebSocket-Protocol", mosiSubprotocol)
		http.Error(w, "unsupported node protocol", http.StatusUpgradeRequired)
		return
	}
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
	retired := r.directory.Retired[id]
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
	conn, err := (&websocket.Upgrader{Subprotocols: []string{mosiSubprotocol}, CheckOrigin: func(req *http.Request) bool { return req.Header.Get("Origin") == "" }}).Upgrade(w, req, nil)
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
	r.peerReady[id] = false
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
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{mosiSubprotocol}
	for {
		select {
		case <-r.done:
			return
		default:
		}
		header := http.Header{"X-Memcp-Replica": []string{r.id}, "X-Memcp-Role": []string{"replication"}}
		conn, response, err := dialer.Dial(r.address, header)
		if err != nil && response != nil {
			response.Body.Close()
			if response.StatusCode == http.StatusForbidden {
				_, _ = r.handleSlave(context.Background(), "retire", nil)
				return
			}
		}
		if err == nil && conn.Subprotocol() != mosiSubprotocol {
			conn.Close()
			err = fmt.Errorf("unsupported node protocol")
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
			go r.recoverCached(p)
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

// Engine access fails closed while configured replication has not published
// its manager. Frontend bootstrap may still use the excluded system catalog.
func activeReplication(db *database) *replicationManager {
	if db == nil {
		return nil
	}
	r := replication.Load()
	if r == nil && replicationStarting.Load() && validReplicaDatabase(db.Name) {
		panic(mosi.NetworkError)
	}
	return r
}

func validReplicaDatabase(name string) bool {
	return name != "" && name != "system" && !strings.HasPrefix(name, "system_") && !strings.ContainsAny(name, "/\\\x00") && name != "." && name != ".."
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

// The installation UUID identifies a node across reconnects. Cluster membership
// requires the independent shared-secret proof; knowing this ID grants no role.
func localNodeIdentity() (string, error) {
	settingsMu.Lock()
	createdIdentity := Settings.ReplicationUUID == ""
	if createdIdentity {
		Settings.ReplicationUUID = uuid.NewString()
	}
	id := Settings.ReplicationUUID
	settingsMu.Unlock()
	if _, err := uuid.Parse(id); err != nil {
		return "", fmt.Errorf("invalid saved replication identity")
	}
	if createdIdentity {
		if err := saveSettings(); err != nil {
			settingsMu.Lock()
			Settings.ReplicationUUID = ""
			settingsMu.Unlock()
			return "", err
		}
	}
	return id, nil
}
