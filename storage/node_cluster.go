/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "os"
import "fmt"
import "sort"
import "sync"
import "time"
import "bytes"
import "errors"
import "context"
import "strings"
import "sync/atomic"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// The initial implementation keeps a full mesh for small clusters (<=10).
// For larger deployments retain only ten random neighbours and route messages
// over hops to their stable memcp:// peer URI. Address/connection/incarnation
// are separate from that URI; a reconnect must not change the logical target.
// Sparse multi-hop routing is deliberately not approximated by a larger mesh.
const clusterRosterLimit = 10

type clusterMember struct {
	ID      string `json:"id"`
	Boot    string `json:"boot"`
	Address string `json:"address"`
}
type clusterHello struct {
	Version int             `json:"version"`
	Self    clusterMember   `json:"self"`
	Members []clusterMember `json:"members"`
}

// A link belongs to one authenticated process connection. Each operation uses
// the common envelope identity for replies, cancellation and duplicate checks.
type clusterLink struct {
	member clusterMember
	peer   *mosiPeer
}
type clusterView struct {
	members []clusterMember
	links   []*clusterLink
}

// Membership is discovery, not MOSI authority. Losing a link changes placement
// hints only; it cannot promote a backup or reclaim an offline writer. The first
// contact has no special role: every node retains the complete authenticated
// roster and independently connects to discovered peers.
type nodeCluster struct {
	manager            *replicationManager     // shared MOSI engine adapter; no second state machine
	bootstrapResources map[string]mosiResource // immutable initialization only

	databases map[string]bool // immutable startup selection of shared-backend databases
	service   *nodeScanService
	self      clusterMember
	seed      string
	ctx       context.Context
	cancel    context.CancelFunc
	workers   sync.WaitGroup
	mu        sync.Mutex // cold roster publication only; local readers load view
	members   map[string]clusterMember
	links     map[string]*clusterLink
	view      atomic.Pointer[clusterView]
	idsMu     sync.Mutex
	nextID    uint64
	limitID   uint64
	received  atomic.Uint64
	delegated atomic.Uint64
	locality  atomic.Uint64
	capacity  atomic.Uint64
}

func activeNodeCluster() *nodeCluster {
	if service := nodeScans.Load(); service != nil {
		return service.cluster
	}
	return nil
}
func (cluster *nodeCluster) publishLocked() {
	view := &clusterView{}
	for _, member := range cluster.members {
		view.members = append(view.members, member)
	}
	for _, link := range cluster.links {
		view.links = append(view.links, link)
	}
	sort.Slice(view.members, func(i, j int) bool { return view.members[i].ID < view.members[j].ID })
	sort.Slice(view.links, func(i, j int) bool { return view.links[i].member.ID < view.links[j].member.ID })
	cluster.view.Store(view)
}
func validClusterMember(member clusterMember) bool {
	idErr := validateClusterURI(member.ID)
	_, bootErr := uuid.Parse(member.Boot)
	address, err := mosiEndpointURL(member.Address)
	return idErr == nil && bootErr == nil && err == nil && address == member.Address && len(address) <= 2048
}
func (cluster *nodeCluster) remember(source string, hello clusterHello) error {
	if hello.Version != scanWireVersion || hello.Self.ID != source || source == cluster.self.ID || !validClusterMember(hello.Self) || len(hello.Members) > clusterRosterLimit {
		return fmt.Errorf("invalid authenticated cluster roster")
	}
	for _, member := range hello.Members {
		if !validClusterMember(member) {
			return fmt.Errorf("invalid cluster member")
		}
	}
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	newIDs := map[string]bool{source: true}
	for _, member := range hello.Members {
		newIDs[member.ID] = true
	}
	additional := 0
	for id := range newIDs {
		if _, exists := cluster.members[id]; !exists {
			additional++
		}
	}
	if len(cluster.members)+additional > clusterRosterLimit {
		return fmt.Errorf("cluster roster limit")
	}
	// Gossip supplies contact hints. Only this peer's fresh challenge proof may
	// replace its own identity/boot entry; gossip cannot authenticate a connection.
	cluster.members[source] = hello.Self
	for _, member := range hello.Members {
		if member.ID == cluster.self.ID {
			continue
		}
		if _, exists := cluster.members[member.ID]; !exists {
			cluster.members[member.ID] = member
		}
	}
	cluster.publishLocked()
	return nil
}
func (cluster *nodeCluster) hello() clusterHello {
	return clusterHello{scanWireVersion, cluster.self, cluster.view.Load().members}
}
func (cluster *nodeCluster) handle(session *scanRPCSession, ctx context.Context, source, method string, data json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch method {
	case "cluster.try_read":
		return cluster.handleTryRead(ctx, source, data)
	case "cluster.location", "cluster.location_update", "cluster.schema_locations":
		return cluster.handleLocations(ctx, source, method, data)
	case "cluster.reserve_ids":
		return cluster.handleIDReservation(ctx, data)
	case "cluster.report":
		var resource mosiResource
		if err := json.Unmarshal(data, &resource); err != nil {
			return nil, err
		}
		if !resource.valid() || resource != globalResource && !cluster.databases[resource.Database] {
			return nil, fmt.Errorf("invalid cluster report")
		}
		return cluster.report(resource), nil
	case "cluster.hello":
		var hello clusterHello
		if err := json.Unmarshal(data, &hello); err != nil {
			return nil, err
		}
		if err := cluster.remember(source, hello); err != nil {
			return nil, err
		}
		return boundedScanReply(cluster.hello())
	default:
		return nil, fmt.Errorf("unknown cluster method")
	}
}
func (cluster *nodeCluster) exchange(link *clusterLink) error {
	ctx, cancel := context.WithTimeout(cluster.ctx, 3*time.Second)
	defer cancel()
	var hello clusterHello
	if err := cluster.service.control(ctx, link.peer, "cluster.hello", cluster.hello(), &hello, true); err != nil {
		return err
	}
	return cluster.remember(link.member.ID, hello)
}

// Publish one authenticated connection without closing a concurrently published
// live link underneath MOSI transfers. Discovery and demand acquisition share
// this publication owner.
func (cluster *nodeCluster) installLink(link *clusterLink) *mosiPeer {
	cluster.mu.Lock()
	prior := cluster.links[link.member.ID]
	if prior != nil {
		select {
		case <-prior.peer.done:
			prior = nil
		default:
		}
	}
	if prior == nil {
		link.member = cluster.members[link.member.ID]
		cluster.links[link.member.ID] = link
		cluster.publishLocked()
	}
	cluster.mu.Unlock()
	if prior != nil {
		cluster.service.forget(link.peer)
		return prior.peer
	}
	return link.peer
}

func (cluster *nodeCluster) ensurePeer(ctx context.Context, id string) (*mosiPeer, error) {
	if peer := cluster.peer(id); peer != nil {
		return peer, nil
	}
	var member clusterMember
	for _, candidate := range cluster.view.Load().members {
		if candidate.ID == id {
			member = candidate
			break
		}
	}
	if member.ID == "" {
		return nil, fmt.Errorf("unknown cluster peer %s", id)
	}
	peer, identity, err := cluster.service.connect(ctx, member.Address)
	if err != nil {
		return nil, err
	}
	if identity != id {
		cluster.service.forget(peer)
		return nil, fmt.Errorf("cluster contact identity changed")
	}
	link := &clusterLink{member: member, peer: peer}
	if err := cluster.exchange(link); err != nil {
		cluster.service.forget(peer)
		return nil, err
	}
	return cluster.installLink(link), nil
}
func (cluster *nodeCluster) loop() {
	defer cluster.workers.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	defer func() {
		for _, link := range cluster.view.Load().links {
			cluster.service.forget(link.peer)
		}
	}()
	for {
		view := cluster.view.Load()
		addresses := make(map[string]string)
		if cluster.seed != "" {
			addresses[cluster.seed] = ""
		}
		for _, member := range view.members {
			if member.ID != cluster.self.ID {
				addresses[member.Address] = member.ID
			}
		}
		for _, link := range view.links {
			delete(addresses, link.member.Address)
			select {
			case <-link.peer.done:
				cluster.service.forget(link.peer)
				cluster.mu.Lock()
				delete(cluster.links, link.member.ID)
				cluster.publishLocked()
				cluster.mu.Unlock()
			default:
				if err := cluster.exchange(link); err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
					fmt.Println("cluster roster exchange failed:", link.member.ID, err)
					link.peer.close()
				}
			}
		}
		for address, expectedID := range addresses {
			if address == cluster.self.Address || cluster.ctx.Err() != nil {
				continue
			}
			ctx, cancel := context.WithTimeout(cluster.ctx, 3*time.Second)
			peer, id, err := cluster.service.connect(ctx, address)
			cancel()
			if err != nil {
				continue
			}
			if id == cluster.self.ID || expectedID != "" && id != expectedID {
				cluster.service.forget(peer)
				continue
			}
			link := &clusterLink{member: clusterMember{ID: id, Address: address}, peer: peer}
			if err := cluster.exchange(link); err != nil {
				cluster.service.forget(peer)
				continue
			}
			cluster.installLink(link)
		}
		select {
		case <-cluster.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// A cluster starts from a local secret file, never a command-line secret or a
// secret returned by the seed. Authentication remains the existing mutual,
// direction-bound HMAC over fresh challenges. This adapter adds peer discovery.
func startNodeCluster(secretFile, address, seed, databases string, bootstrap bool) error {
	if nodeScans.Load() != nil {
		return fmt.Errorf("node protocol is already initialized")
	}
	secret, err := os.ReadFile(secretFile)
	if err != nil {
		return fmt.Errorf("reading cluster secret file: %w", err)
	}
	secret = bytes.TrimRight(secret, "\r\n")
	if len(secret) < 32 {
		return fmt.Errorf("cluster secret file must contain at least 32 bytes")
	}
	endpoint, err := mosiEndpointURL(address)
	if err != nil {
		return err
	}
	if seed != "" {
		seed, err = mosiEndpointURL(seed)
		if err != nil {
			return err
		}
	}
	if err := prepareSharedClusterMounts(databases); err != nil {
		return err
	}
	fresh := Settings.ReplicationUUID == ""
	id, err := localNodeIdentity()
	if err != nil {
		return err
	}
	service, err := newNodeScanService(string(secret))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cluster := &nodeCluster{service: service, self: clusterMember{"memcp://" + id, service.id, endpoint}, seed: seed, ctx: ctx, cancel: cancel, members: make(map[string]clusterMember), links: make(map[string]*clusterLink)}
	cluster.databases = make(map[string]bool)
	for _, name := range strings.Split(databases, ",") {
		if !validReplicaDatabase(name) {
			cancel()
			return fmt.Errorf("cluster-databases requires explicit shared database names")
		}
		cluster.databases[name] = true
	}
	cluster.initializeManager()
	if err := cluster.prepareBootstrap(fresh || bootstrap); err != nil {
		cancel()
		return err
	}
	if err := cluster.restoreGlobalContacts(); err != nil {
		cancel()
		return err
	}
	cluster.members[cluster.self.ID] = cluster.self
	cluster.publishLocked()
	service.cluster = cluster // immutable before the HTTP role becomes reachable
	service.slots = make(chan struct{}, clusterRosterLimit)
	if err := publishNodeScanService(service); err != nil {
		cancel()
		return err
	}
	cluster.workers.Add(1)
	go cluster.loop()
	return nil
}
func initNodeCluster(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "node_cluster_start", Fn: func(a ...scm.Scmer) scm.Scmer {
		if err := startNodeCluster(a[0].String(), a[1].String(), a[2].String(), a[3].String(), scm.ToBool(a[4])); err != nil {
			panic(err)
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "string", Label: "secret_file"}, {Kind: "string", Label: "advertise_url"}, {Kind: "string", Label: "join_url"}, {Kind: "string", Label: "databases"}, {Kind: "bool", Label: "bootstrap"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "cluster_status", Fn: func(a ...scm.Scmer) scm.Scmer {
		cluster := activeNodeCluster()
		if cluster == nil {
			return scm.NewNil()
		}
		view := cluster.view.Load()
		output := struct {
			Self      clusterMember   `json:"self"`
			Members   []clusterMember `json:"members"`
			Connected int             `json:"connected"`
			Received  uint64          `json:"received"`
			Delegated uint64          `json:"delegated"`
			Locality  uint64          `json:"locality"`
			Capacity  uint64          `json:"capacity"`
		}{cluster.self, view.members, len(view.links), cluster.received.Load(), cluster.delegated.Load(), cluster.locality.Load(), cluster.capacity.Load()}
		var resources json.RawMessage
		if len(a) > 0 && scm.ToBool(a[0]) {
			var err error
			resources, err = cluster.manager.statusSnapshot()
			if err != nil {
				panic(err)
			}
		}
		data, err := json.Marshal(struct {
			State     any             `json:"state"`
			Budget    CacheStat       `json:"budget"`
			Resources json.RawMessage `json:"resources,omitempty"`
		}{output, GlobalCache.Stat(), resources})
		if err != nil {
			panic(err)
		}
		return scm.NewString(string(data))
	}, Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{{Kind: "bool", Label: "resources", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "string"}}})
}

func validateClusterURI(id string) error {
	if !strings.HasPrefix(id, "memcp://") {
		return fmt.Errorf("expected memcp:// peer URI")
	}
	_, err := uuid.Parse(strings.TrimPrefix(id, "memcp://"))
	return err
}

// Membership/admission policy belongs to the node object, not the socket reader.
func (cluster *nodeCluster) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	session, _ := ctx.Value(clusterObjectSessionKey{}).(*scanRPCSession)
	data, ok := request.Payload.(json.RawMessage)
	if session == nil || !ok {
		return nil, fmt.Errorf("invalid native node request")
	}
	return cluster.handle(session, ctx, session.remoteID, request.Method, data)
}

// requireMember is computation admission, independent of residency and MOSI.
func (cluster *nodeCluster) requireMember(source string) error {
	if view := cluster.view.Load(); view != nil {
		for _, member := range view.members {
			if member.ID == source {
				return nil
			}
		}
	}
	return fmt.Errorf("authenticated cluster hello required before computation")
}
