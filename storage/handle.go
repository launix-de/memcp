/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "weak"
import "context"
import "strconv"
import "runtime"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

// Slots are never reused within a service incarnation. Their handlers, not the
// transport, define payload semantics and any reply channel.
type handleSlot struct {
	peer     clusterObjectPeer
	receiver scm.ClusterRequestHandler
	owner    scm.Scmer // local invocation lifetime; never part of a message
	activity atomic.Uint64
	drained  chan struct{}
}

const handleStopped uint64 = 1 << 63

func (slot *handleSlot) enter() bool {
	for {
		state := slot.activity.Load()
		if state&handleStopped != 0 {
			return false
		}
		if slot.activity.CompareAndSwap(state, state+1) {
			return true
		}
	}
}
func (slot *handleSlot) leave() {
	if slot.activity.Add(^uint64(0)) == handleStopped {
		close(slot.drained)
	}
}
func (slot *handleSlot) stop() {
	for {
		state := slot.activity.Load()
		if state&handleStopped != 0 {
			return
		}
		if slot.activity.CompareAndSwap(state, state|handleStopped) {
			if state == 0 {
				close(slot.drained)
			}
			return
		}
	}
}
func (slot *handleSlot) close() { slot.stop(); <-slot.drained }

const (
	handleValue      byte = 0xf0
	handleClose      byte = 0xf1
	handleFinalValue byte = 0xf2
)

// Each imported callback owns one local GC lease, shared by duplicate imports.
// The registry contains weak pointers only. Cleanup arguments never root the
// imported handle, its invocation, or its receiver.
type remoteHandleKey struct {
	peer clusterObjectPeer
	slot uint64
}
type remoteHandleLease struct {
	key      remoteHandleKey
	cleanup  runtime.Cleanup
	finished atomic.Bool
}
type remoteHandleCleanup struct {
	registry *clusterObjectRegistry
	key      remoteHandleKey
	identity weak.Pointer[remoteHandleLease]
}

func cleanupRemoteHandle(cleanup remoteHandleCleanup) {
	r := cleanup.registry
	r.mu.Lock()
	if current := r.remoteHandles[cleanup.key]; current != cleanup.identity && current.Value() != nil {
		r.mu.Unlock()
		return
	}
	delete(r.remoteHandles, cleanup.key)
	if !r.stopped {
		r.releases = append(r.releases, clusterObjectRelease{peer: cleanup.key.peer, slot: cleanup.key.slot})
	}
	r.mu.Unlock()
	select {
	case r.wake <- struct{}{}:
	default:
	}
}
func (service *nodeScanService) importHandleLease(peer clusterObjectPeer, slot uint64) *remoteHandleLease {
	r := service.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	key := remoteHandleKey{peer, slot}
	if r.remoteHandles == nil {
		r.remoteHandles = make(map[remoteHandleKey]weak.Pointer[remoteHandleLease])
	}
	if lease := r.remoteHandles[key].Value(); lease != nil {
		return lease
	}
	lease := &remoteHandleLease{key: key}
	identity := weak.Make(lease)
	r.remoteHandles[key] = identity
	lease.cleanup = runtime.AddCleanup(lease, cleanupRemoteHandle, remoteHandleCleanup{r, key, identity})
	return lease
}

type weakResultHandle struct {
	receiver weak.Pointer[scm.ResultReceiver]
}

func (handle weakResultHandle) Handle(value scm.Scmer) {
	if receiver := handle.receiver.Value(); receiver != nil {
		receiver.Handle(value)
	}
}

type remoteHandle struct {
	lease   *remoteHandleLease
	service *nodeScanService
	peer    clusterObjectPeer
	slot    uint64
	ctx     context.Context
	owner   scm.Scmer
}

func (handle *remoteHandle) Context() context.Context { return handle.ctx }
func (handle *remoteHandle) WithInvocation(owner scm.Scmer, ctx context.Context) scm.Handle {
	bound := *handle
	bound.ctx = ctx
	bound.owner = owner
	return &bound
}
func (handle *remoteHandle) SerializeHandle(bindings *scm.SerializationContext) (scm.Scmer, error) {
	// Re-exporting registers a forwarding handler with the new invocation's
	// lifetime. A third peer never mistakes a sender-relative slot for its own.
	if bindings.ExportHandle == nil {
		return scm.NewNil(), fmt.Errorf("handle requires an export context")
	}
	return bindings.ExportHandle(handle)
}
func (handle *remoteHandle) Handle(value scm.Scmer)       { handle.send(value, false) }
func (handle *remoteHandle) HandleResult(value scm.Scmer) { handle.send(value, true) }
func (handle *remoteHandle) send(value scm.Scmer, final bool) {
	if handle.lease.finished.Load() {
		panic("handle result already sent")
	}
	defer runtime.KeepAlive(handle)
	peer := handle.service.objectConnection(handle.peer)
	if peer == nil {
		panic("handle peer is disconnected")
	}
	var offers []scm.ClusterReference
	bindings := (&scm.SerializationContext{Invocation: handle.owner}).WithExport(func(object scm.ClusterObject, view scm.Scmer) (scm.ClusterReference, error) {
		ref, err := handle.service.exportObject(peer, object, view)
		if err == nil {
			offers = append(offers, ref)
		}
		return ref, err
	})
	queued := false
	defer func() {
		if !queued {
			handle.service.refuseObjectOffers(peer, offers)
		}
	}()
	bindings = handle.service.handleBindings(bindings, peer)
	payload, err := freezeScanValue(value, bindings)
	if err != nil {
		panic(err)
	}
	kind := handleValue
	if final {
		kind = handleFinalValue
	}
	message := mosiMessage{Channel: handleChannel(handle.slot), Payload: append([]byte{kind}, payload...)}
	message.encoded, err = encodeMosiMessage(message)
	if err != nil {
		panic(err)
	}
	handle.service.commitObjectOffers(peer, offers)
	if err = peer.send(context.Background(), message); err != nil {
		panic(err)
	}
	queued = true
	if final {
		// A final delivery itself relinquishes this return-channel reference. Stop
		// its GC notification so it cannot send a redundant standalone Close.
		handle.lease.finished.Store(true)
		handle.lease.cleanup.Stop()
		r := handle.service.objectRegistry()
		r.mu.Lock()
		if r.remoteHandles[handle.lease.key].Value() == handle.lease {
			delete(r.remoteHandles, handle.lease.key)
		}
		r.mu.Unlock()
	}
}

func handleChannel(slot uint64) string { return strconv.FormatUint(slot, 10) }
func (service *nodeScanService) storeHandle(slot uint64, entry *handleSlot) {
	entry.drained = make(chan struct{})
	service.handles.Store(slot, entry)
	work := workerChannel(func(peer *mosiPeer, payload []byte) error {
		return (&scanRPCSession{service: service, peer: peer}).deliverHandle(entry, payload)
	}, entry.leave)
	service.channels.Store(handleChannel(slot), scm.ClusterRequestHandlerFunc(func(ctx context.Context, payload []byte) {
		delivery := ctx.(*channelDelivery)
		if entry.peer != delivery.peer.objectPeer {
			delivery.accepted = false
			return
		}
		if len(payload) == 1 && payload[0] == handleClose {
			entry.stop()
			if service.handles.CompareAndDelete(slot, entry) {
				service.channels.Delete(handleChannel(slot))
			}
			return
		}
		// Admission happens in stream order, before asynchronous execution. Close
		// cannot discard an already accepted fire-and-forget invocation.
		if !entry.enter() {
			if current, exists := service.handles.Load(slot); exists && current != entry {
				if replacement, exists := service.channels.Load(handleChannel(slot)); exists {
					replacement.(scm.ClusterRequestHandler).HandleClusterRequest(ctx, payload)
					return
				}
			}
			// Admission can race cancellation after the reader looked up this
			// receiver. Return any grants without reconstructing expired query data.
			delivery.accepted = service.discardHandlePayload(delivery.peer, payload) == nil
			return
		}
		work.HandleClusterRequest(ctx, payload)
		if !delivery.accepted {
			entry.leave()
		} else if len(payload) > 0 && payload[0] == handleFinalValue {
			entry.stop()
			if service.handles.CompareAndDelete(slot, entry) {
				service.channels.Delete(handleChannel(slot))
			}
		}
	}))
}
func (service *nodeScanService) deleteHandle(slot uint64) {
	if entry, exists := service.handles.LoadAndDelete(slot); exists {
		entry.(*handleSlot).stop()
	}
	service.channels.Delete(handleChannel(slot))
}

func (service *nodeScanService) allocateHandleSlot() uint64 {
	slot := service.nextHandle.Add(1)
	if slot == 0 || slot>>63 != 0 {
		panic("handle address space exhausted")
	}
	return slot
}
func (service *nodeScanService) handleResolver(ctx context.Context, peer *mosiPeer, scope scm.Scmer) func(uint64) (scm.Handle, error) {
	return func(slot uint64) (scm.Handle, error) {
		identity := peer.objectPeer
		if slot == 0 || service.objectConnection(identity) == nil {
			return nil, fmt.Errorf("unknown handle peer or slot")
		}
		return &remoteHandle{lease: service.importHandleLease(identity, slot), service: service, peer: identity, slot: slot, ctx: scanQueryContext(scmerToTxContext(scope), ctx), owner: scope}, nil
	}
}
func (service *nodeScanService) handleBindings(bindings *scm.SerializationContext, peer *mosiPeer) *scm.SerializationContext {
	owner := bindings.InvocationScope()
	return bindings.WithHandleExport(func(handler scm.Handle) (scm.Scmer, error) {
		slot := service.allocateHandleSlot()
		entry := &handleSlot{peer: peer.objectPeer, owner: owner}
		decodeHandler := handler
		_, resultReceiver := handler.(*scm.ResultReceiver)
		if receiver, ok := handler.(*scm.ResultReceiver); ok {
			decodeHandler = weakResultHandle{weak.Make(receiver)}
		}
		entry.receiver = service.schemeHandleReceiver(decodeHandler, owner)
		service.storeHandle(slot, entry)
		release := func() { service.deleteHandle(slot) }
		if receiver, ok := handler.(interface{ RetainHandleRelease(func()) }); ok {
			receiver.RetainHandleRelease(release)
			if result, ok := handler.(*scm.ResultReceiver); ok && peer.ctx != nil {
				weakResult := weak.Make(result)
				stop := context.AfterFunc(peer.ctx, func() {
					if receiver := weakResult.Value(); receiver != nil {
						receiver.Fail(fmt.Errorf("handle peer disconnected"))
					}
				})
				result.RetainHandleRelease(func() { stop() })
			}
		}
		// Producers register their finish hook before exporting reply slots. Join
		// admitted reply decoding after those producers, including canceled waits.
		// The hook is weak so abandoning a callback can still release it early.
		weakEntry := weak.Make(entry)
		if lifetime, ok := owner.Any().(interface{ RetainQueryWork(func()) bool }); ok {
			retained := lifetime.RetainQueryWork(func() {
				release()
				if current := weakEntry.Value(); current != nil {
					current.close()
				}
			})
			if !retained && !resultReceiver {
				release()
				return scm.NewNil(), fmt.Errorf("exported handle requires a live invocation owner")
			}
			// An already registered producer may flush its final batch while closing.
			// It owns and joins those late one-shot replies itself.
		}
		return scm.NewSlice([]scm.Scmer{scm.NewSymbol("cluster_handle"), scm.NewInt(int64(slot))}), nil
	})
}

// The scan role owns both payload codecs. Serializer source never begins with
// the binary scan-completion marker; control completions are binary records, not Scheme.
// Inspect only inert manifests, never reconstruct a cancelled query's values.
func (service *nodeScanService) discardHandlePayload(peer *mosiPeer, payload []byte) error {
	if len(payload) > 0 && payload[0] == handleClose {
		return nil
	}
	if len(payload) > 0 && (payload[0] == handleValue || payload[0] == handleFinalValue) {
		payload = payload[1:]
	}
	var refs []scm.ClusterReference
	if len(payload) > 0 && payload[0] == scanCompletionMarker {
		var completion scanCompletion
		if err := unmarshalNode(payload, &completion); err != nil {
			return err
		}
		refs = completion.Offers
	} else {
		refs = serializedReferences(schemeValue(payload))
	}
	service.discardObjectOffers(peer, refs)
	return nil
}

func (s *scanRPCSession) handleValue(slot uint64, payload schemeValue) error {
	entry, ok := s.service.handles.Load(slot)
	if !ok {
		return s.service.discardHandlePayload(s.peer, payload)
	} // a late delivery to a completed invocation is inert
	target := entry.(*handleSlot)
	if target.peer != s.peer.objectPeer {
		return fmt.Errorf("foreign handle slot")
	}
	if len(payload) == 1 && payload[0] == handleClose {
		target.stop()
		if s.service.handles.CompareAndDelete(slot, target) {
			s.service.channels.Delete(handleChannel(slot))
		}
		return nil
	}

	if !target.enter() {
		// A cancelled waiter can hand its result slot to a state-free cleanup
		// handler. An already admitted delivery must follow that handoff.
		if current, exists := s.service.handles.Load(slot); exists && current != entry {
			return s.handleValue(slot, payload)
		}
		return nil
	}
	defer target.leave()
	return s.deliverHandle(target, payload)
}

func (s *scanRPCSession) deliverHandle(target *handleSlot, payload []byte) error {
	target.receiver.HandleClusterRequest(&channelDelivery{Context: s.peer.ctxOrBackground(), peer: s.peer}, payload)
	return nil
}

// A Scheme handle is a language-level adapter over the same raw receiver.
// Only this payload protocol knows that its tail is one serialized Scmer.
func (service *nodeScanService) schemeHandleReceiver(handler scm.Handle, owner scm.Scmer) scm.ClusterRequestHandler {
	return scm.ClusterRequestHandlerFunc(func(ctx context.Context, payload []byte) {
		if len(payload) == 0 || (payload[0] != handleValue && payload[0] != handleFinalValue) {
			panic("invalid handle delivery")
		}
		peer := ctx.(*channelDelivery).peer
		value, err := schemeValue(payload[1:]).reconstruct(scm.ProcedureDecode{ResolveHandle: service.handleResolver(peer.ctx, peer, owner), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
			return service.importObject(peer.ctx, peer, ref)
		}})
		if err != nil {
			panic(err)
		}
		handler.Handle(value)
	})
}
