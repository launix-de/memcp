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
import "sync"
import "time"
import "strings"
import "context"
import "net/http"
import "crypto/hmac"
import "sync/atomic"
import "encoding/hex"
import "crypto/sha256"
import "github.com/google/uuid"
import "github.com/gorilla/websocket"
import "github.com/launix-de/memcp/scm"

// nodeScanService is startup-owned. Its secret never enters replication auth or
// an HTTP response. Incarnations and connection/replay state are process-local.
var nodeScans atomic.Pointer[nodeScanService]
var nodeServiceConstructing bool // guarded by the existing database namespace owner

type nodeScanService struct {
	channelOnce sync.Once
	channels    sync.Map // shared registered channels survive peer reconnection
	handles     sync.Map // fresh slot -> invocation-owned handle
	nextHandle  atomic.Uint64
	objects     clusterObjectRegistry
	executions  sync.Map // invocation UUID -> worker TxContext, never persisted or serialized
	id          string
	cluster     *nodeCluster // immutable startup-owned adapter; nil for standalone object endpoints
	secret      []byte
	release     func()
	mu          sync.Mutex
	peers       map[*mosiPeer]bool
	stopping    bool
	connections sync.WaitGroup
	slots       chan struct{}
}

type scanChallenge struct {
	OperationPrefix string `json:"-"` // derived from the existing authenticated nonce pair
	Boot            string `json:"boot"`
	Version         int    `json:"version"`
	Node            string `json:"node"`
	Nonce           string `json:"nonce"`
	Proof           string `json:"proof,omitempty"`
	Return          string `json:"return,omitempty"`
}

func scanAuthProof(secret []byte, role string, server, client scanChallenge) string {
	mac := hmac.New(sha256.New, secret)
	for _, item := range []string{mosiSubprotocol + "/objects.channels.v11", role, server.Node, server.Boot, server.Nonce, server.Return, client.Node, client.Boot, client.Nonce, client.Return} {
		mac.Write([]byte(item))
		mac.Write([]byte{0})
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func validScanChallenge(c scanChallenge) bool {
	_, nodeErr := uuid.Parse(c.Node)
	if strings.HasPrefix(c.Node, "memcp://") {
		nodeErr = validateClusterURI(c.Node)
	}
	_, nonceErr := uuid.Parse(c.Nonce)
	_, bootErr := uuid.Parse(c.Boot)
	return c.Version == scanWireVersion && nodeErr == nil && nonceErr == nil && bootErr == nil && len(c.Return) > 0 && len(c.Return) <= 255
}

func (service *nodeScanService) authenticate(peer *mosiPeer, incoming bool) (scanChallenge, error) {
	peer.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	local := scanChallenge{Version: scanWireVersion, Node: service.authNodeID(), Boot: service.id, Nonce: uuid.NewString(), Return: "scan.accept"}
	var remote scanChallenge
	if incoming {
		local.Return = "scan.proof"
		if err := postScanMessage(peer.ctx, peer, "scan.challenge", local); err != nil {
			return scanChallenge{}, err
		}
		if err := peer.readAuth(local.Return, &remote); err != nil {
			return scanChallenge{}, err
		}
		if !validScanChallenge(remote) || !hmac.Equal([]byte(remote.Proof), []byte(scanAuthProof(service.secret, "client", local, remote))) {
			return scanChallenge{}, fmt.Errorf("node authentication failed")
		}
		if err := postScanMessage(peer.ctx, peer, remote.Return, scanChallenge{Version: scanWireVersion, Proof: scanAuthProof(service.secret, "server", local, remote)}); err != nil {
			return scanChallenge{}, err
		}
		remote.OperationPrefix = scanAuthProof(service.secret, "operation", local, remote)
		return remote, nil
	}
	if err := peer.readAuth("scan.challenge", &remote); err != nil {
		return scanChallenge{}, err
	}
	if !validScanChallenge(remote) {
		return scanChallenge{}, fmt.Errorf("invalid node challenge")
	}
	local.Proof = scanAuthProof(service.secret, "client", remote, local)
	if err := postScanMessage(peer.ctx, peer, remote.Return, local); err != nil {
		return scanChallenge{}, err
	}
	var accepted scanChallenge
	if err := peer.readAuth(local.Return, &accepted); err != nil {
		return scanChallenge{}, err
	}
	if accepted.Version != scanWireVersion || !hmac.Equal([]byte(accepted.Proof), []byte(scanAuthProof(service.secret, "server", remote, local))) {
		return scanChallenge{}, fmt.Errorf("node authentication failed")
	}
	remote.OperationPrefix = scanAuthProof(service.secret, "operation", remote, local)
	return remote, nil
}

func (service *nodeScanService) beginConnection() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.stopping {
		return false
	}
	service.connections.Add(1)
	return true
}

func (service *nodeScanService) track(peer *mosiPeer) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.stopping {
		return false
	}
	service.peers[peer] = true
	service.wakeObjectReleases()
	return true
}
func (service *nodeScanService) forget(peer *mosiPeer) {
	defer service.connections.Done()
	peer.close()
	peer.wait()
	service.mu.Lock()
	delete(service.peers, peer)
	service.mu.Unlock()
}
func (service *nodeScanService) accept(w http.ResponseWriter, req *http.Request) {
	if !service.beginConnection() {
		http.Error(w, "node stopping", http.StatusServiceUnavailable)
		return
	}
	handedOff := false
	defer func() {
		if !handedOff {
			service.connections.Done()
		}
	}()
	if req.Method != http.MethodGet {
		http.Error(w, "node protocol requires GET", http.StatusMethodNotAllowed)
		return
	}
	select {
	case service.slots <- struct{}{}:
		defer func() { <-service.slots }()
	default:
		http.Error(w, "node connection limit", http.StatusServiceUnavailable)
		return
	}
	supported := false
	for _, protocol := range websocket.Subprotocols(req) {
		if protocol == mosiSubprotocol {
			supported = true
		}
	}
	if !supported {
		http.Error(w, "unsupported node protocol", http.StatusUpgradeRequired)
		return
	}
	conn, err := (&websocket.Upgrader{Subprotocols: []string{mosiSubprotocol}, CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" }}).Upgrade(w, req, nil)
	if err != nil {
		return
	}
	var remote scanChallenge
	var authErr error
	remoteID := ""
	session := newScanRPCSession()
	session.service = service
	var packets func(*mosiPacket) error
	if service.cluster != nil {
		packets = func(packet *mosiPacket) error {
			<-session.peerReady
			return service.cluster.receive(remoteID, packet, session.peer)
		}
	}
	peer := newMosiPeer(conn, func(peer *mosiPeer) {
		remote, authErr = service.authenticate(peer, true)
		if authErr != nil {
			peer.close()
			return
		}
		// Only an authenticated peer may name this node's object endpoint; an
		// arbitrary Host header would redirect peers into a relayable handshake.
		registry := service.objectRegistry()
		registry.mu.Lock()
		if registry.endpoint == "" {
			scheme := "http"
			if req.TLS != nil {
				scheme = "https"
			}
			registry.endpoint = scheme + "://" + req.Host + req.URL.Path
		}
		registry.mu.Unlock()
		remoteID = remote.Node
		session.remoteID = remoteID
		peer.scanService = service
		registerScanChannels(peer, session)
		registerMosiChannel(peer, packets)
		peer.operationPrefix = remote.OperationPrefix
		peer.objectPeer = clusterObjectPeer{remote.Node, remote.Boot}
		session.bindPeer(peer)
		// Application decoding can resolve callback handles immediately after
		// initialization. Publish this authenticated connection before the
		// transport starts its permanent reader, not after its constructor returns.
		if !service.track(peer) {
			peer.close()
		}
	})
	if authErr != nil {
		peer.wait()
		return
	}
	handedOff = true
	defer service.forget(peer)
	<-peer.readDone
}
func (service *nodeScanService) connect(ctx context.Context, address string) (*mosiPeer, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	endpoint, err := mosiEndpointURL(address)
	if err != nil {
		return nil, "", err
	}
	// A successful dial publishes a shared service connection. Reuse it for
	// later setup/release work instead of accumulating per-call transports.
	service.mu.Lock()
	if !service.stopping {
		for peer := range service.peers {
			if peer.objectEndpoint == endpoint {
				select {
				case <-peer.done:
				default:
					service.mu.Unlock()
					return peer, peer.objectPeer.ID, nil
				}
			}
		}
	}
	service.mu.Unlock()
	if !service.beginConnection() {
		return nil, "", fmt.Errorf("node stopping")
	}
	handedOff := false
	defer func() {
		if !handedOff {
			service.connections.Done()
		}
	}()
	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = []string{mosiSubprotocol}
	conn, response, err := dialer.DialContext(ctx, endpoint, http.Header{"X-Memcp-Role": []string{"scan"}})
	if err != nil && response != nil && response.Body != nil {
		response.Body.Close()
	}
	if err != nil {
		return nil, "", err
	}
	if conn.Subprotocol() != mosiSubprotocol {
		conn.Close()
		return nil, "", fmt.Errorf("unsupported node protocol")
	}
	var remote scanChallenge
	var authErr error
	tracked := false
	remoteID := ""
	session := newScanRPCSession()
	session.service = service
	var packets func(*mosiPacket) error
	if service.cluster != nil {
		packets = func(packet *mosiPacket) error {
			<-session.peerReady
			return service.cluster.receive(remoteID, packet, session.peer)
		}
	}
	peer := newMosiPeer(conn, func(peer *mosiPeer) {
		remote, authErr = service.authenticate(peer, false)
		if authErr != nil {
			peer.close()
			return
		}
		remoteID = remote.Node
		session.remoteID = remoteID
		peer.objectEndpoint = endpoint
		peer.scanService = service
		registerScanChannels(peer, session)
		registerMosiChannel(peer, packets)
		peer.operationPrefix = remote.OperationPrefix
		peer.objectPeer = clusterObjectPeer{remote.Node, remote.Boot}
		session.bindPeer(peer)
		// Inbound callback decoding must see the same published connection
		// before newMosiPeer admits its first application frame.
		tracked = service.track(peer)
		if !tracked {
			peer.close()
		}
	})
	if authErr != nil {
		peer.wait()
		return nil, "", authErr
	}
	if !tracked {
		peer.close()
		peer.wait()
		return nil, "", fmt.Errorf("node stopping")
	}
	handedOff = true
	// Publication in service.peers makes this connection available to every
	// native handle caller. Its lifetime belongs to the service, not the dialer.
	go func() {
		<-peer.readDone
		service.forget(peer)
	}()
	return peer, remoteID, nil
}

// Construct before publication so the optional cluster adapter is immutable to
// readers. Object endpoints and cluster membership share this service,
// route, mutual challenge proof and bounded binary transport.
func newNodeScanService(secret string) (*nodeScanService, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("cluster secret must contain at least 16 bytes")
	}
	return &nodeScanService{id: uuid.NewString(), secret: []byte(secret), peers: make(map[*mosiPeer]bool), slots: make(chan struct{}, 32)}, nil
}
func publishNodeScanService(service *nodeScanService) error {
	if nodeScans.Load() != nil {
		return fmt.Errorf("node scans already initialized")
	}
	release, err := registerMosiRole("scan", http.HandlerFunc(service.accept))
	if err != nil {
		return err
	}
	service.release = release
	nodeScans.Store(service)
	return nil
}
func startNodeScans(secret string) error {
	finishStart, err := beginNodeServiceStart()
	if err != nil {
		return err
	}
	defer finishStart()
	service, err := newNodeScanService(secret)
	if err != nil {
		return err
	}
	return publishNodeScanService(service)
}
func (service *nodeScanService) authNodeID() string {
	if service.cluster != nil {
		return service.cluster.self.ID
	}
	return service.id
}

func StopNodeScans() {
	if service := nodeScans.Load(); service != nil {
		service.stop()
	}
}

func (service *nodeScanService) stop() {
	if service.cluster != nil {
		service.cluster.cancel()
	}
	service.mu.Lock()
	service.stopping = true
	peers := make([]*mosiPeer, 0, len(service.peers))
	for peer := range service.peers {
		peers = append(peers, peer)
	}
	service.mu.Unlock()
	if service.release != nil {
		service.release()
	}
	for _, peer := range peers {
		peer.close()
	}
	for _, peer := range peers {
		peer.wait()
	}
	if service.cluster != nil {
		service.cluster.workers.Wait()
	}
	service.connections.Wait()
	service.stopObjects()
}
func initNodeScans(en *scm.Env) {
	initNodeCluster(en)
	initClusterObjectAPI(en)
	scm.Declare(en, &scm.Declaration{Name: "node_query_context", Fn: func(a ...scm.Scmer) scm.Scmer {
		if service := nodeScans.Load(); service != nil {
			if value, ok := service.executions.Load(a[0].String()); ok {
				return scm.NewAny(value.(*TxContext))
			}
		}
		panic("expired remote query context")
	}, Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{{Kind: "string", Label: "channel"}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "node_scan_start", Fn: func(a ...scm.Scmer) scm.Scmer {
		if err := startNodeScans(a[0].String()); err != nil {
			panic(err)
		}
		return scm.NewBool(true)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "string", Label: "secret"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "node_id", Fn: func(...scm.Scmer) scm.Scmer {
		if service := nodeScans.Load(); service != nil {
			return scm.NewString(service.id)
		}
		return scm.NewNil()
	}, Type: &scm.TypeDescriptor{Kind: "func", Return: &scm.TypeDescriptor{Kind: "string"}}})
}
