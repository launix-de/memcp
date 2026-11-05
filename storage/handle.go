/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "sync"
import "context"
import "github.com/launix-de/memcp/scm"

// Slots are never reused within a service incarnation. Their handlers, not the
// transport, define payload semantics and any reply channel.
type handleSlot struct {
	peer    clusterObjectPeer
	handler scm.Handle
	owner   scm.Scmer // local invocation lifetime; never part of a message
	mu      sync.Mutex
	closed  bool
	calls   sync.WaitGroup
}

func (slot *handleSlot) enter() bool {
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if slot.closed {
		return false
	}
	slot.calls.Add(1)
	return true
}
func (slot *handleSlot) close() {
	slot.mu.Lock()
	slot.closed = true
	slot.mu.Unlock()
	slot.calls.Wait()
}

type remoteHandle struct {
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
func (handle *remoteHandle) Handle(value scm.Scmer) {
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
	message := mosiMessage{Slot: handle.slot, Payload: payload}
	message.encoded, err = encodeMosiMessage(message)
	if err != nil {
		panic(err)
	}
	handle.service.commitObjectOffers(peer, offers)
	if err = peer.send(context.Background(), message); err != nil {
		panic(err)
	}
	queued = true
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
		return &remoteHandle{service: service, peer: identity, slot: slot, ctx: scanQueryContext(scmerToTxContext(scope), ctx), owner: scope}, nil
	}
}
func (service *nodeScanService) handleBindings(bindings *scm.SerializationContext, peer *mosiPeer) *scm.SerializationContext {
	owner := bindings.InvocationScope()
	return bindings.WithHandleExport(func(handler scm.Handle) (scm.Scmer, error) {
		slot := service.allocateHandleSlot()
		entry := &handleSlot{peer: peer.objectPeer, handler: handler, owner: owner}
		service.handles.Store(slot, entry)
		release := func() { service.handles.Delete(slot) }
		if receiver, ok := handler.(interface{ RetainHandleRelease(func()) }); ok {
			receiver.RetainHandleRelease(release)
			if result, ok := handler.(*scm.ResultReceiver); ok && peer.ctx != nil {
				stop := context.AfterFunc(peer.ctx, func() { result.Fail(fmt.Errorf("handle peer disconnected")) })
				result.RetainHandleRelease(func() { stop() })
			}
		} else if lifetime, ok := owner.Any().(interface{ RetainQueryWork(func()) bool }); !ok || !lifetime.RetainQueryWork(func() { release(); entry.close() }) {
			release()
			return scm.NewNil(), fmt.Errorf("exported handle requires a live invocation owner")
		}
		return scm.NewSlice([]scm.Scmer{scm.NewSymbol("cluster_handle"), scm.NewInt(int64(slot))}), nil
	})
}
func (s *scanRPCSession) handleValue(slot uint64, payload schemeValue) error {
	entry, ok := s.service.handles.Load(slot)
	if !ok {
		s.service.discardObjectOffers(s.peer, serializedReferences(payload))
		return nil
	} // a late delivery to a completed invocation is inert
	target := entry.(*handleSlot)
	if target.peer != s.peer.objectPeer {
		return fmt.Errorf("foreign handle slot")
	}
	if !target.enter() {
		// A cancelled waiter can hand its result slot to a state-free cleanup
		// handler. An already admitted delivery must follow that handoff.
		if current, exists := s.service.handles.Load(slot); exists && current != entry {
			return s.handleValue(slot, payload)
		}
		return nil
	}
	defer target.calls.Done()
	value, err := payload.reconstruct(scm.ProcedureDecode{ResolveHandle: s.service.handleResolver(s.peer.ctx, s.peer, target.owner), Resolve: func(ref scm.ClusterReference) (scm.Scmer, error) {
		return s.service.importObject(s.peer.ctx, s.peer, ref)
	}})
	if err != nil {
		return err
	}
	target.handler.Handle(value)
	return nil
}
