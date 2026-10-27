/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "time"
import "weak"
import "strings"
import "context"
import "runtime"
import "net/url"
import "encoding/json"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"

// ClusterObject is shared with the runtime, avoiding a storage dependency in scm.
// Shards, transactions and runtime callables implement this same owner contract.
type ClusterObject = scm.ClusterObject

type clusterObjectEntry struct {
	identity string
	object   ClusterObject // strong while at least one remote token remains
	uri      string
	tokens   map[string]clusterObjectPeer
	views    map[string]scanWireValue
	acquired map[string]bool
	offers   map[string]*mosiPeer // unacquired offers belong to their offering connection
}
type clusterObjectPeer struct {
	ID   string
	Boot string
}
type clusterObjectRelease struct {
	peer      clusterObjectPeer
	reference scm.ClusterReference
}

// Only cold export/import/release paths take this mutex. Registered objects are
// not touched by ordinary local readers. Historical shard registration therefore
// pins the Go generation without adding a read counter to its element loop.
type clusterObjectRegistry struct {
	mu         sync.Mutex
	once       sync.Once
	endpoint   string
	entries    map[string]*clusterObjectEntry
	identities map[ClusterObject]string
	releases   []clusterObjectRelease
	wake       chan struct{}
	stop       chan struct{}
	stopped    bool
	imported   map[string]weak.Pointer[clusterObjectLease] // weak import identity; never roots peer values
	retired    []weak.Pointer[storageShard]                // local in-flight owners survive last peer release
}

func (service *nodeScanService) objectRegistry() *clusterObjectRegistry {
	r := &service.objects
	r.once.Do(func() {
		r.entries = make(map[string]*clusterObjectEntry)
		r.identities = make(map[ClusterObject]string)
		r.wake = make(chan struct{}, 1)
		r.stop = make(chan struct{})
		go func() {
			for {
				select {
				case <-r.wake:
					r.mu.Lock()
					batch := r.releases
					r.releases = nil
					r.mu.Unlock()
					for _, release := range batch {
						if !service.deliverObjectRelease(release) {
							r.mu.Lock()
							if !r.stopped {
								r.releases = append(r.releases, release)
							}
							r.mu.Unlock()
						}
					}
				case <-r.stop:
					return
				}
			}
		}()
	})
	return r
}
func (service *nodeScanService) exportObject(peer *mosiPeer, object ClusterObject, view scm.Scmer, nesting ...int) (scm.ClusterReference, error) {
	if peer == nil || object == nil {
		return scm.ClusterReference{}, fmt.Errorf("missing cluster object peer/owner")
	}
	depth := 0
	if len(nesting) > 0 {
		depth = nesting[0]
	}
	if depth > 64 {
		return scm.ClusterReference{}, fmt.Errorf("cyclic/excessive runtime object view")
	}
	var children []scm.ClusterReference
	complete := false
	defer func() {
		if !complete {
			service.cancelObjectOffers(peer, children)
		}
	}()
	bindings := (&scm.ClusterBindings{}).WithExport(func(child scm.ClusterObject, childView scm.Scmer) (scm.ClusterReference, error) {
		ref, err := service.exportObject(peer, child, childView, depth+1)
		if err == nil {
			children = append(children, ref)
		}
		return ref, err
	})
	encoded, err := freezeScanValue(view, 0, bindings)
	if err != nil {
		return scm.ClusterReference{}, err
	}
	payload, err := json.Marshal(encoded)
	if err != nil {
		return scm.ClusterReference{}, err
	}
	r := service.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return scm.ClusterReference{}, fmt.Errorf("cluster object service stopped")
	}
	if named, ok := object.(interface{ ClusterObjectPath() string }); ok {
		endpoint := r.endpoint
		if service.cluster != nil {
			endpoint = service.cluster.self.Address
		}
		if endpoint == "" {
			endpoint = "memcp://" + service.id + "/cluster"
		}
		complete = true
		return scm.ClusterReference{URI: strings.TrimRight(endpoint, "/") + "/" + named.ClusterObjectPath(), Owner: service.authNodeID(), Incarnation: service.id, Type: object.ClusterObjectType(), View: payload}, nil
	}
	identity := ""
	if named, ok := object.(interface{ ClusterObjectID() string }); ok {
		identity = named.ClusterObjectID()
	}
	uri := r.identities[object]
	if uri != "" && r.entries[uri].identity != identity {
		uri = ""
	}
	if uri == "" {
		id := uuid.NewString()
		if identity, ok := object.(interface{ ClusterObjectID() string }); ok {
			id = identity.ClusterObjectID()
		}
		endpoint := r.endpoint
		if service.cluster != nil {
			endpoint = service.cluster.self.Address
		}
		if endpoint == "" {
			endpoint = "memcp://" + service.id + "/cluster"
		}
		uri = strings.TrimRight(endpoint, "/") + "/" + id
		if r.entries[uri] != nil {
			return scm.ClusterReference{}, fmt.Errorf("conflicting cluster object identity")
		}
		r.entries[uri] = &clusterObjectEntry{identity: identity, object: object, uri: uri, tokens: make(map[string]clusterObjectPeer), views: make(map[string]scanWireValue), acquired: make(map[string]bool), offers: make(map[string]*mosiPeer)}
		r.identities[object] = uri
	}
	token := uuid.NewString()
	r.entries[uri].tokens[token] = peer.objectPeer
	r.entries[uri].views[token] = encoded
	r.entries[uri].offers[token] = peer
	complete = true
	return scm.ClusterReference{URI: uri, Owner: service.authNodeID(), Incarnation: service.id, Token: token, Type: object.ClusterObjectType(), View: payload}, nil
}
func (service *nodeScanService) releaseObject(peer *mosiPeer, reference scm.ClusterReference) error {
	r := service.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	if reference.Owner != service.authNodeID() || reference.Incarnation != service.id {
		return nil // stale GC releases cannot affect a new owner incarnation
	}
	entry := r.entries[reference.URI]
	if entry == nil {
		return nil
	} // idempotent GC delivery
	if expected, exists := entry.tokens[reference.Token]; exists {
		if expected != peer.objectPeer {
			return fmt.Errorf("cluster object handle belongs to another peer")
		}
	}
	r.dropObjectTokenLocked(reference)
	return nil
}
func (r *clusterObjectRegistry) dropObjectTokenLocked(reference scm.ClusterReference) {
	entry := r.entries[reference.URI]
	if entry == nil {
		return
	}
	delete(entry.tokens, reference.Token)
	delete(entry.views, reference.Token)
	delete(entry.acquired, reference.Token)
	delete(entry.offers, reference.Token)
	if len(entry.tokens) == 0 {
		if shard := clusterObjectShard(entry.object); shard != nil {
			r.retired = append(r.retired, weak.Make(shard))
		}
		delete(r.entries, reference.URI)
		if r.identities[entry.object] == entry.uri {
			delete(r.identities, entry.object)
		}
	}
}

// The imported value owns this lease. The owner keeps the object, never this
// lease. Its GC cleanup only queues a release; it performs no network I/O.
// Disconnect alone does NOT release handles: a remote old-generation reader may
// still run. No timeout promotes MOSI rights or proves distributed deletion.
type clusterObjectLease struct {
	ready chan struct{}
	value scm.Scmer
	err   error
}
type clusterObjectCleanup struct {
	registry  *clusterObjectRegistry
	peer      clusterObjectPeer
	reference scm.ClusterReference
	key       string
}

// AddCleanup is the GC callback for the imported handle. Unlike a finalizer it
// also works for a callable/closure cycle. Its argument contains no lease/value.
func finalizeClusterObject(cleanup clusterObjectCleanup) {
	r := cleanup.registry
	r.mu.Lock()
	delete(r.imported, cleanup.key)
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.releases = append(r.releases, clusterObjectRelease{cleanup.peer, cleanup.reference})
	r.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
}
func (service *nodeScanService) importObject(ctx context.Context, peer *mosiPeer, reference scm.ClusterReference) (result scm.Scmer, err error) {
	parsed, err := url.Parse(reference.URI)
	if err != nil || parsed.Host == "" || (reference.Token == "" && reference.Type != "table") || reference.Incarnation == "" || reference.Type == "" || reference.Owner != peer.objectPeer.ID {
		return scm.NewNil(), fmt.Errorf("invalid cluster object URI")
	}
	if reference.Type == "table" && reference.Token == "" {
		return scm.ReconstructClusterObject(scm.ClusterHandle{Reference: reference})
	}
	registry := service.objectRegistry()
	key := reference.Owner + "/" + reference.Incarnation + "/" + reference.URI + "/" + reference.Token
	registry.mu.Lock()
	if registry.imported == nil {
		registry.imported = make(map[string]weak.Pointer[clusterObjectLease])
	}
	if existing := registry.imported[key].Value(); existing != nil {
		registry.mu.Unlock()
		select {
		case <-existing.ready:
			return existing.value, existing.err
		case <-ctx.Done():
			return scm.NewNil(), ctx.Err()
		}
	}
	lease := &clusterObjectLease{ready: make(chan struct{})}
	registry.imported[key] = weak.Make(lease)
	registry.mu.Unlock()
	runtime.AddCleanup(lease, finalizeClusterObject, clusterObjectCleanup{registry, peer.objectPeer, reference, key})
	defer func() {
		if failure := recover(); failure != nil {
			result = scm.NewNil()
			err = fmt.Errorf("cluster object reconstruction: %v", failure)
		}
		lease.value, lease.err = result, err
		close(lease.ready)
		runtime.KeepAlive(lease)
	}()
	if err := service.control(ctx, peer, "object.acquire", reference, nil); err != nil {
		return scm.NewNil(), err
	}
	var frozen scanWireValue
	if err := json.Unmarshal(reference.View, &frozen); err != nil {
		return scm.NewNil(), err
	}
	decodedView, err := frozen.thaw(0, scm.ProcedureDecode{Resolve: func(child scm.ClusterReference) (scm.Scmer, error) { return service.importObject(ctx, peer, child) }})
	if err != nil {
		return scm.NewNil(), err
	}
	handle := scm.ClusterHandle{Reference: reference, View: decodedView, Call: func(callCtx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		defer runtime.KeepAlive(lease)
		target := service.objectConnection(peer.objectPeer)
		if target == nil {
			slash := strings.LastIndex(reference.URI, "/")
			if slash < 0 {
				return scm.NewNil(), fmt.Errorf("invalid object URI")
			}
			connected, connectErr := service.objectEndpoint(callCtx, reference.URI[:slash])
			if connectErr != nil {
				return scm.NewNil(), connectErr
			}
			target = connected
		}
		var result clusterObjectReply
		operation := callCtx
		var offered []scm.ClusterReference
		bindings := (&scm.ClusterBindings{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
			ref, err := service.exportObject(target, object, view)
			if err == nil {
				offered = append(offered, ref)
			}
			return ref, err
		})
		args := make([]scanWireValue, len(request.Arguments))
		defer func() { service.cancelObjectOffers(target, offered) }()
		for i, arg := range request.Arguments {
			var err error
			args[i], err = freezeScanValue(arg, 0, bindings)
			if err != nil {
				return scm.NewNil(), err
			}
		}
		err := service.control(operation, target, "object.call", clusterObjectCall{reference, request.Method, args}, &result)
		if err != nil {
			return scm.NewNil(), err
		}
		defer service.discardObjectOffers(target, result.Offers)
		return result.Value.thaw(0, scm.ProcedureDecode{Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return service.importObject(callCtx, target, ref) }})
	}}
	return scm.ReconstructClusterObject(handle)
}

type clusterObjectCall struct {
	Reference scm.ClusterReference `json:"reference"`
	Method    string               `json:"method"`
	Arguments []scanWireValue      `json:"arguments,omitempty"`
}

func (s *scanRPCSession) handleObject(ctx context.Context, method string, data []byte) (any, error) {
	if method == "object.discard" {
		var refs []scm.ClusterReference
		if err := json.Unmarshal(data, &refs); err != nil {
			return nil, err
		}
		s.service.cancelObjectOffers(s.peer, refs)
		return nil, nil
	}
	if method == "object.acquire" {
		var ref scm.ClusterReference
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, err
		}
		r := s.service.objectRegistry()
		r.mu.Lock()
		defer r.mu.Unlock()
		entry := r.entries[ref.URI]
		if entry == nil || ref.Owner != s.service.authNodeID() || ref.Incarnation != s.service.id || entry.tokens[ref.Token] != s.peer.objectPeer {
			return nil, fmt.Errorf("expired or foreign cluster object offer")
		}
		if entry.acquired[ref.Token] {
			return nil, fmt.Errorf("cluster object handle was already consumed")
		}
		entry.acquired[ref.Token] = true
		delete(entry.offers, ref.Token)
		return nil, nil
	}
	if method == "object.release" {
		var ref scm.ClusterReference
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, err
		}
		return nil, s.service.releaseObject(s.peer, ref)
	}
	var call clusterObjectCall
	if err := json.Unmarshal(data, &call); err != nil {
		return nil, err
	}
	if call.Method == "view" {
		_, view, err := s.resolveObject(call.Reference)
		return clusterObjectReply{view, scanWireReferences(view)}, err
	}
	value, err := s.requestObject(ctx, call)
	if err != nil {
		return nil, err
	}
	var offered []scm.ClusterReference
	bindings := (&scm.ClusterBindings{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := s.service.exportObject(s.peer, object, view)
		if err == nil {
			offered = append(offered, ref)
		}
		return ref, err
	})
	frozen, err := freezeScanValue(value, 0, bindings)
	refs := scanWireReferences(frozen)
	if err != nil {
		s.service.cancelObjectOffers(s.peer, offered)
	}
	return clusterObjectReply{frozen, refs}, err
}

func (s *scanRPCSession) requestObject(ctx context.Context, call clusterObjectCall) (scm.Scmer, error) {
	object, _, err := s.resolveObject(call.Reference)
	if err != nil {
		return scm.NewNil(), err
	}
	// Native query ownership is established once for callable/helper requests.
	// Decoded nested scans borrow this cancellation/transaction context.
	id := uuid.NewString()
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Cluster", "object request")
	ss.SetQueryContext(seq, ctx)
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	tx.beginQuery(ss, seq, "object request")
	defer tx.endQuery(seq)
	s.service.executions.Store(id, tx)
	defer s.service.executions.Delete(id)
	ctx = context.WithValue(ctx, clusterObjectQueryKey{}, tx)
	binding := scm.NewSlice([]scm.Scmer{scm.NewSymbol("node_query_context"), scm.NewString(id)})
	args := make([]scm.Scmer, len(call.Arguments))
	decode := scm.ProcedureDecode{QueryContext: binding, Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		value, err := s.service.importObject(ctx, s.peer, ref)
		if err != nil {
			return scm.NewNil(), err
		}
		return scm.BindClusterInvocation(value, scm.NewAny(tx))
	}}
	for i, arg := range call.Arguments {
		var err error
		args[i], err = arg.thaw(0, decode)
		if err != nil {
			return scm.NewNil(), err
		}
	}
	value, err := scm.ClusterObjectValue(object, context.WithValue(ctx, clusterObjectSessionKey{}, s), scm.ClusterRequest{Method: call.Method, Arguments: args})
	if err != nil {
		return scm.NewNil(), err
	}
	return value, nil
}

// Reconnect keeps handles of the SAME authenticated process incarnation usable.
// A different boot cannot release them. Connection loss itself proves nothing
// about old readers and never removes a registration or a MOSI permission.
func (service *nodeScanService) objectConnection(identity clusterObjectPeer) *mosiPeer {
	service.mu.Lock()
	defer service.mu.Unlock()
	for peer := range service.peers {
		if peer.objectPeer == identity {
			select {
			case <-peer.done:
				continue
			default:
				return peer
			}
		}
	}
	return nil
}

func (service *nodeScanService) stopObjects() {
	r := &service.objects
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stop != nil && !r.stopped {
		r.stopped = true
		close(r.stop)
		r.entries = nil
		r.identities = nil
		r.releases = nil
	}
}
func (service *nodeScanService) wakeObjectReleases() {
	r := &service.objects
	r.mu.Lock()
	wake := r.wake
	r.mu.Unlock()
	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

func clusterObjectShard(object ClusterObject) *storageShard {
	switch object := object.(type) {
	case *storageShard:
		return object
	case *storageShardTransaction:
		return object.clusterShard
	}
	return nil
}

// Cold maintenance obtains native pointers, not a second distributed refcount.
// Weak retired entries also cover a local request that outlives its peer's
// cancellation/release. Once Go has no owner, the next pass drops that entry.
func registeredClusterShards(name string) []*storageShard {
	service := nodeScans.Load()
	if service == nil {
		return nil
	}
	r := &service.objects
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := make(map[*storageShard]bool)
	var result []*storageShard
	add := func(shard *storageShard) {
		if shard != nil && shard.t != nil && shard.t.schema.Name == name && !seen[shard] {
			seen[shard] = true
			result = append(result, shard)
		}
	}
	for _, entry := range r.entries {
		add(clusterObjectShard(entry.object))
	}
	retained := r.retired[:0]
	for _, entry := range r.retired {
		if shard := entry.Value(); shard != nil {
			retained = append(retained, entry)
			add(shard)
		}
	}
	r.retired = retained
	return result
}
func clusterRetainsShard(shard *storageShard) bool {
	for _, registered := range registeredClusterShards(shard.t.schema.Name) {
		if registered.uuid == shard.uuid {
			return true
		}
	}
	return false
}

func (s *scanRPCSession) resolveObject(ref scm.ClusterReference) (ClusterObject, scanWireValue, error) {
	r := s.service.objectRegistry()
	r.mu.Lock()
	entry := r.entries[ref.URI]
	var object ClusterObject
	var view scanWireValue
	if entry != nil && ref.Owner == s.service.authNodeID() && ref.Incarnation == s.service.id && entry.tokens[ref.Token] == s.peer.objectPeer && entry.acquired[ref.Token] && entry.object.ClusterObjectType() == ref.Type {
		object = entry.object
		view = entry.views[ref.Token]
	}
	r.mu.Unlock()

	if object == nil && ref.Type == "table" && ref.Token == "" && ref.Owner == s.service.authNodeID() && ref.Incarnation == s.service.id {
		table, err := resolveClusterCatalog(ref.URI)
		if err != nil {
			return nil, scanWireValue{}, err
		}
		object = table
	}
	if object == nil {
		return nil, scanWireValue{}, fmt.Errorf("expired or foreign cluster object handle")
	}
	if entry == nil {
		value, err := scm.ClusterObjectValue(object, context.Background(), scm.ClusterRequest{Method: "view"})
		if err != nil {
			return nil, scanWireValue{}, err
		}
		view, err = freezeScanValue(value, 0)
		if err != nil {
			return nil, scanWireValue{}, err
		}
	}
	return object, view, nil
}

// Offers are pinned until the receiver acquires its handle. Failed encoding or
// a closed offering connection can retire ONLY unacquired offers. An acquired
// remote reader survives disconnect and is released by its GC notification.
func (service *nodeScanService) cancelObjectOffers(peer *mosiPeer, references []scm.ClusterReference) {
	var expanded []scm.ClusterReference
	for _, ref := range references {
		expanded = append(expanded, scanWireReferences(scanWireValue{Kind: "object", Reference: &ref})...)
	}
	references = expanded
	for _, ref := range references {
		if ref.Token == "" {
			continue
		}
		r := service.objectRegistry()
		r.mu.Lock()
		entry := r.entries[ref.URI]
		pending := entry != nil && !entry.acquired[ref.Token] && entry.tokens[ref.Token] == peer.objectPeer
		if pending {
			r.dropObjectTokenLocked(ref)
		}
		r.mu.Unlock()
	}
}
func (service *nodeScanService) cancelPeerObjectOffers(peer *mosiPeer) {
	r := service.objectRegistry()
	r.mu.Lock()
	var refs []scm.ClusterReference
	for uri, entry := range r.entries {
		for token, offering := range entry.offers {
			if offering == peer {
				refs = append(refs, scm.ClusterReference{URI: uri, Owner: service.authNodeID(), Incarnation: service.id, Token: token})
			}
		}
	}
	r.mu.Unlock()
	service.cancelObjectOffers(peer, refs)
}

type clusterObjectReply struct {
	Value  scanWireValue          `json:"value"`
	Offers []scm.ClusterReference `json:"offers,omitempty"`
}

func (reply clusterObjectReply) ClusterObjectOffers() []scm.ClusterReference { return reply.Offers }
func (service *nodeScanService) discardObjectOffers(peer *mosiPeer, refs []scm.ClusterReference) {
	if len(refs) != 0 {
		_ = postScanMessage(peer, "object.discard", refs)
	}
}

// GC runs no network operations. Its queue worker uses the existing message
// collector to confirm release, retrying through the same process's reconnected
// peer. A temporary connection is needed only for explicit scan_remote callers
// whose original query connection has already closed.
func (service *nodeScanService) deliverObjectRelease(release clusterObjectRelease) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	peer := service.objectConnection(release.peer)
	if peer == nil {
		endpoint, err := url.Parse(release.reference.URI)
		if err != nil {
			return false
		}
		if endpoint.Scheme != "http" && endpoint.Scheme != "https" && endpoint.Scheme != "ws" && endpoint.Scheme != "wss" {
			return false
		}
		slash := strings.LastIndex(endpoint.Path, "/")
		if slash < 0 {
			return false
		}
		endpoint.Path = endpoint.Path[:slash]
		endpoint.RawPath = ""
		connected, _, err := service.connect(ctx, endpoint.String())
		if err != nil {
			return false
		}
		peer = connected
		defer service.forget(peer)
	}
	return service.control(ctx, peer, "object.release", release.reference, nil) == nil
}

type clusterObjectQueryKey struct{}
