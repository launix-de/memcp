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
	object   ClusterObject // strong while a transfer reservation or peer lease remains
	uri      string
	// Pending transfers pin the object from serialization until import/cancel.
	// Established peer leases are deduplicated across all subscans. They are
	// lifetime generations, NOT per-use counts; a late old GC release cannot
	// revoke the replacement lease of the same authenticated process.
	tokens map[string]clusterObjectPeer
	offers map[string]*mosiPeer
	leases map[string]clusterObjectPeer
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
	mu            sync.Mutex
	once          sync.Once
	endpoint      string
	entries       map[string]*clusterObjectEntry
	identities    map[ClusterObject]string
	releases      []clusterObjectRelease
	wake          chan struct{}
	stop          chan struct{}
	stopped       bool
	imported      map[string]weak.Pointer[clusterObjectImport] // weak import identity; never roots peer values
	remoteObjects map[string]weak.Pointer[clusterObjectLease]
	retired       []weak.Pointer[storageShard] // local in-flight owners survive last peer release
}

func (service *nodeScanService) objectRegistry() *clusterObjectRegistry {
	r := &service.objects
	r.once.Do(func() {
		// Wake/stop and cold diagnostics may run before the first export. Once
		// publishes to registry callers; these other observers use this mutex.
		r.mu.Lock()
		defer r.mu.Unlock()
		r.entries = make(map[string]*clusterObjectEntry)
		r.identities = make(map[ClusterObject]string)
		r.wake = make(chan struct{}, 1)
		r.stop = make(chan struct{})
		go func() {
			// Failed releases need their own retry source. A quiet peer can have
			// no later GC/reconnect event to wake us; never pin its old shard forever.
			retry := time.NewTimer(time.Hour)
			retry.Stop()
			defer retry.Stop()
			var retryC <-chan time.Time
			for {
				select {
				case <-r.wake:
				case <-retryC:
				case <-r.stop:
					return
				}
				retryC = nil
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
				r.mu.Lock()
				pending := len(r.releases) > 0
				r.mu.Unlock()
				if pending {
					retry.Reset(time.Second)
					retryC = retry.C
				}
			}
		}()
	})
	return r
}
func (service *nodeScanService) exportObject(peer *mosiPeer, object ClusterObject, view scm.Scmer, nesting ...int) (scm.ClusterReference, error) {
	if endpoint, ok := object.(interface{ ClusterEndpoint() scm.ClusterObject }); ok {
		object = endpoint.ClusterEndpoint()
	}
	if forwarded, ok := object.(interface{ ClusterForwardedHandle() *scm.ClusterHandle }); ok {
		if handle := forwarded.ClusterForwardedHandle(); handle != nil {
			return service.offerObject(context.Background(), handle.Reference, peer.objectPeer, clusterObjectPeer{service.authNodeID(), service.id})
		}
	}
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
	bindings := (&scm.SerializationContext{}).WithExport(func(child scm.ClusterObject, childView scm.Scmer) (scm.ClusterReference, error) {
		ref, err := service.exportObject(peer, child, childView, depth+1)
		if err == nil {
			children = append(children, ref)
		}
		return ref, err
	})
	encoded, err := freezeScanValue(view, bindings)
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
		r.entries[uri] = &clusterObjectEntry{identity: identity, object: object, uri: uri, tokens: make(map[string]clusterObjectPeer), leases: make(map[string]clusterObjectPeer), offers: make(map[string]*mosiPeer)}
		r.identities[object] = uri
	}
	token := uuid.NewString()
	r.entries[uri].tokens[token] = peer.objectPeer
	r.entries[uri].offers[token] = peer
	complete = true
	return scm.ClusterReference{URI: uri, Owner: service.authNodeID(), Incarnation: service.id, Token: token, Type: object.ClusterObjectType(), View: payload}, nil
}
func (service *nodeScanService) releaseObject(peer *mosiPeer, reference scm.ClusterReference) error {
	r := service.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	if reference.Owner != service.authNodeID() || reference.Incarnation != service.id {
		return nil
	}
	entry := r.entries[reference.URI]
	if entry == nil {
		return nil
	}
	if expected, exists := entry.leases[reference.Token]; exists && expected != peer.objectPeer {
		return fmt.Errorf("cluster object lease belongs to another peer")
	}
	// Idempotent, generation-specific removal. No per-subscan decrement and no
	// timeout: other live views/subscans retain their shared peer Go handle.
	delete(entry.leases, reference.Token)
	r.retireObjectLocked(entry)
	return nil
}
func (r *clusterObjectRegistry) retireObjectLocked(entry *clusterObjectEntry) {
	if len(entry.tokens) != 0 || len(entry.leases) != 0 {
		return
	}
	if shard := clusterObjectShard(entry.object); shard != nil {
		r.retired = append(r.retired, weak.Make(shard))
	}
	if inputs, ok := entry.object.(interface{ clusterInputShards() []*storageShard }); ok {
		for _, shard := range inputs.clusterInputShards() {
			if shard != nil {
				r.retired = append(r.retired, weak.Make(shard))
			}
		}
	}
	delete(r.entries, entry.uri)
	if r.identities[entry.object] == entry.uri {
		delete(r.identities, entry.object)
	}
}
func (r *clusterObjectRegistry) dropObjectTokenLocked(reference scm.ClusterReference) {
	entry := r.entries[reference.URI]
	if entry == nil {
		return
	}
	delete(entry.tokens, reference.Token)
	delete(entry.offers, reference.Token)
	r.retireObjectLocked(entry)
}

// The imported value owns this lease. The owner keeps the object, never this
// lease. Its GC cleanup only queues a release; it performs no network I/O.
// Disconnect alone does NOT release handles: a remote old-generation reader may
// still run. No timeout promotes MOSI rights or proves distributed deletion.
// One shared endpoint per owner/incarnation/URI. This is weakly indexed on the
// peer; ordinary Go reachability joins every concurrent subscan and snapshot.
// Only the last Go reference triggers the network release, outside the GC.
type clusterObjectLease struct{ reference scm.ClusterReference }

// Views and TX participants remain invocation-local: deduplicating endpoint
// identity must NOT share mutable query state or overwrite a frozen session
// snapshot. Every view retains the same endpoint lease through its Call closure.
type clusterObjectImport struct {
	ready chan struct{}
	value scm.Scmer
	err   error
	lease *clusterObjectLease
}
type clusterObjectImportCleanup struct {
	registry *clusterObjectRegistry
	key      string
	identity weak.Pointer[clusterObjectImport]
}

func cleanupClusterImport(cleanup clusterObjectImportCleanup) {
	r := cleanup.registry
	r.mu.Lock()
	if r.imported[cleanup.key] == cleanup.identity {
		delete(r.imported, cleanup.key)
	}
	r.mu.Unlock()
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
	if current := r.remoteObjects[cleanup.key].Value(); current == nil || current.reference.Token == cleanup.reference.Token {
		delete(r.remoteObjects, cleanup.key)
	}
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
	if reference.Type == "transaction" && reference.Token == "" {
		return service.importTransactionRoot(ctx, peer, reference)
	}
	if reference.Owner == service.authNodeID() && reference.Incarnation == service.id && reference.Token != "" {
		r := service.objectRegistry()
		r.mu.Lock()
		entry := r.entries[reference.URI]
		var native ClusterObject
		if entry != nil && entry.object.ClusterObjectType() == reference.Type && entry.leases[reference.Token] != (clusterObjectPeer{}) {
			native = entry.object
		}
		r.mu.Unlock()
		if native == nil {
			return scm.NewNil(), fmt.Errorf("expired returned cluster object")
		}
		return scm.NewAny(native), nil
	}
	if reference.Owner != peer.objectPeer.ID && service.cluster != nil {
		peer, err = service.cluster.ensurePeer(ctx, reference.Owner)
		if err != nil {
			return scm.NewNil(), err
		}
	}
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
		registry.imported = make(map[string]weak.Pointer[clusterObjectImport])
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
	identity := reference.Owner + "/" + reference.Incarnation + "/" + reference.URI
	if registry.remoteObjects == nil {
		registry.remoteObjects = make(map[string]weak.Pointer[clusterObjectLease])
	}
	lease := registry.remoteObjects[identity].Value()
	fresh := lease == nil
	if fresh {
		ref := reference
		ref.Token, ref.View = uuid.NewString(), nil
		lease = &clusterObjectLease{reference: ref}
		registry.remoteObjects[identity] = weak.Make(lease)
	}
	imported := &clusterObjectImport{ready: make(chan struct{}), lease: lease}
	registry.imported[key] = weak.Make(imported)
	registry.mu.Unlock()
	if fresh {
		runtime.AddCleanup(lease, finalizeClusterObject, clusterObjectCleanup{registry, peer.objectPeer, lease.reference, identity})
	}
	runtime.AddCleanup(imported, cleanupClusterImport, clusterObjectImportCleanup{registry, key, weak.Make(imported)})
	defer func() {
		if failure := recover(); failure != nil {
			result, err = scm.NewNil(), fmt.Errorf("cluster object reconstruction: %v", failure)
		}
		imported.value, imported.err = result, err
		close(imported.ready)
		runtime.KeepAlive(imported)
	}()
	// MUST happen while the import holds its shared Go lease. The reservation
	// becomes a lease atomically at the owner; even a never-called/empty-shard
	// lambda then owns a materialized handle with a working GC callback.
	if err := service.control(ctx, peer, "object.acquire", clusterObjectAcquisition{reference, lease.reference.Token}, nil, true); err != nil {
		return scm.NewNil(), err
	}
	var frozen schemeValue
	if err := json.Unmarshal(reference.View, &frozen); err != nil {
		return scm.NewNil(), err
	}
	decodedView, err := frozen.reconstruct(scm.ProcedureDecode{ResolveHandle: service.handleResolver(ctx, scm.NewNil()), Resolve: func(child scm.ClusterReference) (scm.Scmer, error) { return service.importObject(ctx, peer, child) }})
	if err != nil {
		return scm.NewNil(), err
	}
	handleReference := reference
	handleReference.Token = lease.reference.Token
	handle := scm.ClusterHandle{Reference: handleReference, View: decodedView, Call: service.objectCaller(peer, handleReference, imported)}
	value, failure := scm.ReconstructClusterObject(handle)
	if failure == nil {
		if source, ok := value.Any().(*remoteShardObject); ok {
			failure = source.prepareSource(ctx, peer)
		}
	}
	return value, failure
}

func (service *nodeScanService) objectCaller(peer *mosiPeer, reference scm.ClusterReference, retained any) func(context.Context, scm.ClusterRequest) (scm.Scmer, error) {
	handleReference := reference
	return func(callCtx context.Context, request scm.ClusterRequest) (scm.Scmer, error) {
		defer runtime.KeepAlive(retained)
		target := service.objectConnection(peer.objectPeer)
		if target == nil {
			slash := strings.LastIndex(reference.URI, "/")
			if reference.Type == "transaction" && reference.Token == "" {
				slash = strings.LastIndex(reference.URI, "/transactions/")
			}
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
		bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
			ref, err := service.exportObject(target, object, view)
			if err == nil {
				offered = append(offered, ref)
			}
			return ref, err
		})
		args := make([]schemeValue, len(request.Arguments))
		defer func() { service.cancelObjectOffers(target, offered) }()
		for i, arg := range request.Arguments {
			var err error
			args[i], err = freezeScanValue(arg, bindings)
			if err != nil {
				return scm.NewNil(), err
			}
		}
		err := service.control(operation, target, "object.call", clusterObjectCall{handleReference, request.Method, args}, &result, !request.CancelWithoutJoin)
		if err != nil {
			return scm.NewNil(), err
		}
		defer service.discardObjectOffers(target, result.Offers)
		return result.Value.reconstruct(scm.ProcedureDecode{ResolveHandle: service.handleResolver(callCtx, scm.NewNil()), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) { return service.importObject(callCtx, target, ref) }})
	}
}

type clusterObjectOffer struct {
	Reference scm.ClusterReference
	Target    clusterObjectPeer
	Issuer    clusterObjectPeer
}

// Re-export an imported native endpoint at its real owner. The final receiver
// acquires its own existing Go lease, so intermediate peers are not call relays.
func (service *nodeScanService) offerObject(ctx context.Context, ref scm.ClusterReference, target, issuer clusterObjectPeer) (scm.ClusterReference, error) {
	if ref.Owner == target.ID && ref.Incarnation == target.Boot {
		return ref, nil
	}
	if service.cluster == nil {
		return scm.ClusterReference{}, fmt.Errorf("forwarding native references requires cluster membership")
	}
	peer, err := service.cluster.ensurePeer(ctx, ref.Owner)
	if err != nil {
		return scm.ClusterReference{}, err
	}
	if peer.objectPeer.Boot != ref.Incarnation {
		return scm.ClusterReference{}, fmt.Errorf("expired forwarding incarnation")
	}
	var result scm.ClusterReference
	err = service.control(ctx, peer, "object.offer", clusterObjectOffer{ref, target, issuer}, &result, true)
	return result, err
}

type clusterObjectAcquisition struct {
	Reference scm.ClusterReference `json:"reference"`
	Lease     string               `json:"lease"`
}

type clusterObjectCall struct {
	Reference scm.ClusterReference `json:"reference"`
	Method    string               `json:"method"`
	Arguments []schemeValue        `json:"arguments,omitempty"`
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
	if method == "object.offer" {
		var request clusterObjectOffer
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		object, err := s.resolveObject(request.Reference)
		if err != nil {
			return nil, err
		}
		cluster := s.service.cluster
		if cluster == nil {
			return nil, fmt.Errorf("reference forwarding requires cluster membership")
		}
		if forwarded, ok := object.(interface{ ClusterForwardedHandle() *scm.ClusterHandle }); ok {
			if handle := forwarded.ClusterForwardedHandle(); handle != nil {
				return s.service.offerObject(ctx, handle.Reference, request.Target, request.Issuer)
			}
		}
		target, err := cluster.ensurePeer(ctx, request.Target.ID)
		if err != nil {
			return nil, err
		}
		issuer := target
		if request.Issuer.ID != s.service.authNodeID() {
			issuer, err = cluster.ensurePeer(ctx, request.Issuer.ID)
			if err != nil {
				return nil, err
			}
		} else if request.Issuer.Boot != s.service.id {
			return nil, fmt.Errorf("expired forwarding source")
		}
		if target.objectPeer != request.Target || (request.Issuer.ID != s.service.authNodeID() && issuer.objectPeer != request.Issuer) {
			return nil, fmt.Errorf("expired forwarded reference recipient")
		}
		view, err := scm.ClusterObjectValue(object, ctx, scm.ClusterRequest{Method: "view"})
		if err != nil {
			return nil, err
		}
		ref, err := s.service.exportObject(target, object, view)
		if err != nil {
			return nil, err
		}
		r := s.service.objectRegistry()
		r.mu.Lock()
		if entry := r.entries[ref.URI]; entry != nil {
			entry.offers[ref.Token] = issuer
		}
		r.mu.Unlock()
		return ref, nil
	}
	if method == "object.acquire" {
		var request clusterObjectAcquisition
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		ref := request.Reference
		if _, err := uuid.Parse(request.Lease); err != nil {
			return nil, err
		}
		r := s.service.objectRegistry()
		r.mu.Lock()
		defer r.mu.Unlock()
		entry := r.entries[ref.URI]
		if entry == nil || ref.Owner != s.service.authNodeID() || ref.Incarnation != s.service.id || entry.tokens[ref.Token] != s.peer.objectPeer || entry.object.ClusterObjectType() != ref.Type {
			return nil, fmt.Errorf("expired or foreign cluster object offer")
		}
		if previous, exists := entry.leases[request.Lease]; exists && previous != s.peer.objectPeer {
			return nil, fmt.Errorf("foreign cluster object lease")
		}
		// Same peer handle can import arbitrarily many subscan captures. Joining
		// an existing lease is idempotent; only transfer reservations multiply.
		entry.leases[request.Lease] = s.peer.objectPeer
		delete(entry.tokens, ref.Token)
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
	value, err := s.requestObject(ctx, call)
	if err != nil {
		return nil, err
	}
	var offered []scm.ClusterReference
	bindings := (&scm.SerializationContext{}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := s.service.exportObject(s.peer, object, view)
		if err == nil {
			offered = append(offered, ref)
		}
		return ref, err
	})
	frozen, err := freezeScanValue(value, bindings)
	refs := serializedReferences(frozen)
	if err != nil {
		s.service.cancelObjectOffers(s.peer, offered)
	}
	return clusterObjectReply{frozen, refs}, err
}

func (s *scanRPCSession) requestObject(ctx context.Context, call clusterObjectCall) (scm.Scmer, error) {
	object, err := s.resolveObject(call.Reference)
	if err != nil {
		return scm.NewNil(), err
	}
	// Native query ownership is established once for callable/helper requests.
	// Decoded nested scans borrow this cancellation/transaction context.
	id, _ := ctx.Value(remoteScanContextKey{}).(string)
	if id == "" {
		return scm.NewNil(), fmt.Errorf("object execution requires an operation envelope")
	}
	ss := &scm.SessionState{}
	seq := ss.BeginQuery("Cluster", "object request")
	ss.SetQueryContext(seq, ctx)
	tx := NewTxContext(TxCursorStability)
	tx.autoCommit = true
	tx.beginQuery(ss, seq, "object request")
	defer tx.endQuery(seq)
	stop := context.AfterFunc(ctx, func() { ss.KillQuery(seq) })
	defer stop()
	s.service.executions.Store(id, tx)
	defer s.service.executions.Delete(id)
	ctx = context.WithValue(ctx, clusterObjectQueryKey{}, tx)
	binding := scm.NewSlice([]scm.Scmer{scm.NewSymbol("node_query_context"), scm.NewString(id)})
	args := make([]scm.Scmer, len(call.Arguments))
	decode := scm.ProcedureDecode{ResolveHandle: s.service.handleResolver(ctx, scm.NewAny(tx)), Prepare: prepareScanProcedure(binding), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		value, err := s.service.importObject(ctx, s.peer, ref)
		if err != nil {
			return scm.NewNil(), err
		}
		return scm.BindClusterInvocation(value, scm.NewAny(tx))
	}}
	for i, arg := range call.Arguments {
		var err error
		args[i], err = arg.reconstruct(decode)
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

// Objects retaining physical input generations participate in the existing file
// maintenance directory, independently of their request methods or wire kind.
// GC of a registered run must not race deletion of its historical shard files.
func (shard *storageShard) clusterInputShard() *storageShard         { return shard }
func (tx *storageShardTransaction) clusterInputShard() *storageShard { return tx.clusterShard }
func (run *orderedScanRun) clusterInputShard() *storageShard         { return run.shard }
func clusterObjectShard(object ClusterObject) *storageShard {
	if owner, ok := object.(interface{ clusterInputShard() *storageShard }); ok {
		return owner.clusterInputShard()
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
		if owner, ok := entry.object.(interface{ clusterInputShards() []*storageShard }); ok {
			for _, shard := range owner.clusterInputShards() {
				add(shard)
			}
		}
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

func (s *scanRPCSession) resolveObject(ref scm.ClusterReference) (ClusterObject, error) {
	r := s.service.objectRegistry()
	r.mu.Lock()
	entry := r.entries[ref.URI]
	var object ClusterObject
	if entry != nil && ref.Owner == s.service.authNodeID() && ref.Incarnation == s.service.id && entry.leases[ref.Token] == s.peer.objectPeer && entry.object.ClusterObjectType() == ref.Type {
		object = entry.object
	}
	r.mu.Unlock()
	if object == nil && ref.Type == "transaction" && ref.Token == "" && ref.Owner == s.service.authNodeID() && ref.Incarnation == s.service.id {
		return resolveTransactionRoot(ref)
	}
	if object == nil && ref.Type == "table" && ref.Token == "" && ref.Owner == s.service.authNodeID() && ref.Incarnation == s.service.id {
		return resolveClusterCatalog(ref.URI)
	}
	if object == nil {
		return nil, fmt.Errorf("expired or foreign cluster object handle")
	}
	return object, nil
}

// Offers are pinned until the receiver acquires its handle. Failed encoding or
// a closed offering connection can retire ONLY unacquired offers. An acquired
// remote reader survives disconnect and is released by its GC notification.
func (service *nodeScanService) cancelObjectOffers(peer *mosiPeer, references []scm.ClusterReference) {
	var expanded []scm.ClusterReference
	for _, ref := range references {
		expanded = append(expanded, referenceOffers(ref)...)
	}
	references = expanded
	for _, ref := range references {
		if ref.Token == "" {
			continue
		}
		if ref.Owner != service.authNodeID() && service.cluster != nil {
			if owner := service.cluster.peer(ref.Owner); owner != nil {
				service.discardObjectOffers(owner, []scm.ClusterReference{ref})
			}
			continue
		}
		r := service.objectRegistry()
		r.mu.Lock()
		entry := r.entries[ref.URI]
		pending := entry != nil && (entry.tokens[ref.Token] == peer.objectPeer || (service.cluster != nil && ref.Owner == service.authNodeID() && ref.Incarnation == service.id && entry.tokens[ref.Token] != (clusterObjectPeer{}) && service.cluster.requireMember(peer.objectPeer.ID) == nil))
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
	Value  schemeValue            `json:"value"`
	Offers []scm.ClusterReference `json:"offers,omitempty"`
}

func (reply clusterObjectReply) ClusterObjectOffers() []scm.ClusterReference { return reply.Offers }
func (service *nodeScanService) discardObjectOffers(peer *mosiPeer, refs []scm.ClusterReference) {
	if len(refs) != 0 {
		_ = postScanMessage(context.Background(), peer, "object.discard", refs)
	}
}

// GC runs no network operations. Its queue worker uses the existing message
// collector to confirm release, retrying through the same process's reconnected
// peer. A temporary connection is needed for object callers
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
	return service.control(ctx, peer, "object.release", release.reference, nil, true) == nil
}

type clusterObjectQueryKey struct{}
