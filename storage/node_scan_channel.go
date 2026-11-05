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
import "context"
import "strings"
import "strconv"
import "sync/atomic"
import "runtime/debug"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// Requests carry a registered return handle. Results and original panics are
// delivered as slot + payload through the ordinary handle registry. Cancelling never interrupts a shard's item
// loop: it fences new shard tasks and is inherited by nested scan invocations.
type scanCompletion struct {
	bindings *scm.SerializationContext
	Offers   []scm.ClusterReference `json:"offers,omitempty"`
	Data     []byte                 `json:"data,omitempty"`
	Panic    *scanPanic             `json:"panic,omitempty"`
}
type scanPanic struct {
	SQL     *sqldb.SQLError `json:"sql,omitempty"`
	Message string          `json:"message"`
	Value   []byte          `json:"value,omitempty"`
	Stack   string          `json:"stack,omitempty"`
}

func (p scanPanic) Error() string         { return p.Message }
func (p scanPanic) OriginalStack() string { return p.Stack }
func freezeScanPanic(value any) *scanPanic {
	result := &scanPanic{Message: fmt.Sprint(value), Stack: string(debug.Stack())}
	switch v := value.(type) {
	case scanPanic:
		copy := v
		return &copy
	case *scanPanic:
		copy := *v
		return &copy
	case scanError:
		result = freezeScanPanic(v.r)
		result.Stack = v.stack
	case *sqldb.SQLError:
		copy := *v
		result.SQL = &copy
	case scm.Scmer:
		encoded, _ := freezeScanValue(v)
		result.Value = []byte(encoded)
	case string:
		encoded, _ := freezeScanValue(scm.NewString(v))
		result.Value = []byte(encoded)
	}
	return result
}

// clusterReceiver orders caller-owned deliveries by admission to its FIFO. The
// mutex protects admission and completion only; callbacks never hold it.
// Object offers/leases own distributed lifetime independently, including an unlimited -1
// budget. The consumer must Close before releasing callback state. Close joins
// accepted deliveries and cancels active callback contexts. Callbacks must honor
// cancellation when blocking, and must not call Close on their own receiver.
type clusterReceiver struct {
	mu        sync.Mutex
	remaining atomic.Int64
	sealed    bool // production completed; zero during production is not completion
	closed    bool
	joined    bool
	done      chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	head      *receiverDelivery
	tail      *receiverDelivery
	callback  func(context.Context, scm.Scmer) (scm.Scmer, error)
}

// A delivery lives in its invoking handler, including while queued. Cancelling
// a waiter unlinks it without overtaking its active predecessor. No receiver
// worker or detached forwarding task outlives that handler.
type receiverDelivery struct {
	previous, next *receiverDelivery
	ready          chan struct{}
}

func newClusterReceiver(limit int64, callback func(context.Context, scm.Scmer) (scm.Scmer, error)) *clusterReceiver {
	if limit < -1 {
		panic("invalid receiver limit")
	}
	ctx, cancel := context.WithCancel(context.Background())
	receiver := &clusterReceiver{done: make(chan struct{}), ctx: ctx, cancel: cancel, callback: callback}
	receiver.remaining.Store(limit)
	return receiver
}

// Expect MUST succeed before dispatch. Sealing ends production, not ownership.
func (receiver *clusterReceiver) Expect(count int64) bool {
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if receiver.closed || receiver.sealed || count < 0 {
		return false
	}
	for {
		old := receiver.remaining.Load()
		if old == -1 {
			return true
		}
		if count > int64(^uint64(0)>>1)-old {
			return false
		}
		if receiver.remaining.CompareAndSwap(old, old+count) {
			return true
		}
	}
}
func (receiver *clusterReceiver) finishLocked() bool {
	if receiver.head == nil && !receiver.joined && (receiver.closed || receiver.sealed && receiver.remaining.Load() == 0) {
		receiver.closed = true
		receiver.joined = true
		receiver.callback = nil
		close(receiver.done)
		return true
	}
	return false
}
func (receiver *clusterReceiver) Seal() {
	receiver.mu.Lock()
	receiver.sealed = true
	finished := receiver.finishLocked()
	receiver.mu.Unlock()
	if finished {
		receiver.cancel()
	}
}
func (receiver *clusterReceiver) Close() {
	receiver.mu.Lock()
	receiver.closed = true
	receiver.finishLocked()
	receiver.mu.Unlock()
	receiver.cancel()
	<-receiver.done
}

func (receiver *clusterReceiver) complete(delivery *receiverDelivery) {
	receiver.mu.Lock()
	if delivery.previous != nil {
		delivery.previous.next = delivery.next
	} else {
		receiver.head = delivery.next
		if receiver.head != nil {
			close(receiver.head.ready)
		}
	}
	if delivery.next != nil {
		delivery.next.previous = delivery.previous
	} else {
		receiver.tail = delivery.previous
	}
	finished := receiver.finishLocked()
	receiver.mu.Unlock()
	if finished {
		receiver.cancel()
	}
}
func (*clusterReceiver) ClusterObjectType() string { return "receiver" }
func (receiver *clusterReceiver) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(receiver)
}
func (receiver *clusterReceiver) executeClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if request.Method == "view" {
		return scm.NewNil(), nil
	}
	if request.Method != "receive" || len(request.Arguments) != 1 {
		return nil, fmt.Errorf("invalid receiver request")
	}
	receiver.mu.Lock()
	if err := ctx.Err(); err != nil {
		receiver.mu.Unlock()
		return nil, err
	}
	if receiver.closed {
		receiver.mu.Unlock()
		return nil, fmt.Errorf("closed receiver")
	}
	for {
		old := receiver.remaining.Load()
		if old == 0 {
			receiver.mu.Unlock()
			return nil, fmt.Errorf("receiver limit exhausted")
		}
		if old == -1 || receiver.remaining.CompareAndSwap(old, old-1) {
			break
		}
	}
	delivery := receiverDelivery{previous: receiver.tail, ready: make(chan struct{})}
	if receiver.tail == nil {
		receiver.head = &delivery
		close(delivery.ready)
	} else {
		receiver.tail.next = &delivery
	}
	receiver.tail = &delivery
	receiver.mu.Unlock()
	defer receiver.complete(&delivery)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-receiver.ctx.Done():
		return nil, receiver.ctx.Err()
	case <-delivery.ready:
	}
	receiver.mu.Lock()
	callback, closed := receiver.callback, receiver.closed
	receiver.mu.Unlock()
	if closed {
		return nil, context.Canceled
	}
	callbackCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(receiver.ctx, cancel)
	defer stop()
	defer cancel()
	if err := callbackCtx.Err(); err != nil {
		return nil, err
	}
	if err := receiver.ctx.Err(); err != nil {
		return nil, err
	}
	return callback(callbackCtx, request.Arguments[0])
}

type remoteScanContextKey struct{}

type scanTaskScope struct {
	ctx    context.Context
	cancel context.CancelFunc
}
type scanTaskRegistry struct {
	sync.Mutex
	active map[string]*scanTaskScope
	// Bounded completion/cancellation history handles control messages racing
	// task dispatch, without retaining transaction or procedure objects.
	floor    string // retired UUIDv7 watermark: old operations can never execute again
	terminal map[string]struct{}
	order    []string
}

func (r *scanTaskRegistry) remember(id string) {
	if r.terminal == nil {
		r.terminal = make(map[string]struct{})
	}
	if _, exists := r.terminal[id]; !exists {
		r.order = append(r.order, id)
	}
	r.terminal[id] = struct{}{}
	if len(r.order) > 1024 {
		r.floor = max(r.floor, r.order[0])
		delete(r.terminal, r.order[0])
		r.order = r.order[1:]
	}
}
func (r *scanTaskRegistry) begin(parent context.Context, id string) (*scanTaskScope, error) {
	r.Lock()
	defer r.Unlock()
	if _, exists := r.terminal[id]; exists || id <= r.floor {
		return nil, fmt.Errorf("cancelled or completed scan channel")
	}
	if r.active == nil {
		r.active = make(map[string]*scanTaskScope)
	}
	if r.active[id] != nil || len(r.active) >= 1024 {
		return nil, fmt.Errorf("duplicate or excess scan channel")
	}
	ctx, cancel := context.WithCancel(context.WithValue(parent, remoteScanContextKey{}, id))
	scope := &scanTaskScope{ctx, cancel}
	r.active[id] = scope
	return scope, nil
}
func (r *scanTaskRegistry) finish(id string) {
	r.Lock()
	defer r.Unlock()
	if scope := r.active[id]; scope != nil {
		scope.cancel()
		delete(r.active, id)
	}
	r.remember(id)
}
func (r *scanTaskRegistry) cancelTask(id string) {
	r.Lock()
	defer r.Unlock()
	if scope := r.active[id]; scope != nil {
		scope.cancel()
	} else {
		if _, done := r.terminal[id]; !done {
			r.remember(id)
		}
	}
}
func postScanMessage(ctx context.Context, peer *mosiPeer, method string, value any) error {
	data, err := marshalNode(value)
	if err != nil {
		return err
	}
	message := mosiMessage{Channel: method, Payload: data}
	message.encoded, err = encodeMosiMessage(message)
	if err != nil {
		return err
	}
	return peer.send(ctx, message)
}

// Old frames cannot become executable after reconnect, even after the old
// connection's bounded history has gone away. This uses its existing nonce
// binding, not a persistent query registry or a second task/attempt identity.
func (s *scanRPCSession) validOperation(id string) bool {
	if s.peer == nil {
		return false
	}
	if s.peer.operationPrefix != "" {
		prefix, operation, ok := strings.Cut(id, "/")
		if !ok || prefix != s.peer.operationPrefix {
			return false
		}
		id = operation
	}
	operation, err := strconv.ParseUint(id, 16, 64)
	return len(id) == 16 && err == nil && operation != 0
}

func (s *scanRPCSession) channelMessage(ctx context.Context, method string, data []byte) (any, error) {
	switch method {

	case "scan.cancel":
		var id string
		if err := unmarshalNode(data, &id); err != nil {
			return nil, err
		}
		if !s.validOperation(id) {
			return nil, fmt.Errorf("invalid operation identity")
		}
		s.tasks.cancelTask(id)
		return nil, nil

	}
	return nil, fmt.Errorf("unknown scan channel message")
}

// Each registered operation supplies its own decoder; the channel, never an
// inner method string, selects execution. This helper only owns task lifetime.
func (s *scanRPCSession) receiveInvocation(ctx context.Context, method string, payload []byte) (any, error) {
	id, offers, data, err := decodeInvocation(method, payload)
	if err != nil {
		return nil, err
	}
	if id == "" || !s.validOperation(id) {
		return nil, fmt.Errorf("invalid operation identity")
	}
	completion := scanCompletion{}
	defer func() { completion.bindings.Queued(); queuedValuePublications(completion.Offers) }()
	defer s.service.discardObjectOffers(s.peer, offers)
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				completion.Panic = freezeScanPanic(failure)
			}
		}()
		scope, err := s.tasks.begin(ctx, id)
		if err != nil {
			panic(err)
		}
		defer s.tasks.finish(id)
		value, err := s.handle(scope.ctx, method, data)
		if err != nil {
			panic(err)
		}
		if source, ok := value.(interface {
			serializationBindings() *scm.SerializationContext
		}); ok {
			completion.bindings = source.serializationBindings()
		}
		if exported, ok := value.(interface{ ClusterObjectOffers() []scm.ClusterReference }); ok {
			completion.Offers = transferManifest(exported.ClusterObjectOffers())
		}
		// boundedScanReply already encoded and checked this response. Keep
		// those immutable bytes through the collector; do not marshal even
		// a RawMessage wrapper here (which scans and copies it again).
		if encoded, ok := value.(encodedReply); ok {
			completion.Data = encoded.data
			return
		}
		completion.Data, err = marshalNode(value)
		if err != nil {
			panic(err)
		}
	}()
	s.service.commitObjectOffers(s.peer, completion.Offers)
	payload, err = marshalNode(completion)
	if err != nil {
		return nil, err
	}
	if err := s.peer.send(ctx, mosiMessage{Channel: handleChannel(controlReturnSlot(id)), Payload: payload}); err != nil {
		s.service.refuseObjectOffers(s.peer, completion.Offers)
		return nil, err
	}
	return nil, nil

}

func mustNodeMetadata(value any) []byte {
	data, err := marshalNode(value)
	if err != nil {
		panic(err)
	}
	return data
}

func (executor *peerScanExecutor) dispatch(task remoteScanTask, offers ...scm.ClusterReference) remoteScanResult {
	service := executor.peer.scanService
	if service == nil {
		panic("missing scan return-channel service")
	}
	var result remoteScanResult
	if err := service.control(executor.ctx, executor.peer, "scan.execute", task, &result, true, offers...); err != nil {
		panicRemoteOperation(err)
	}
	if result.Rejected {
		panic(clusterCapacityRejection{})
	}
	if result.Redirect != nil {
		panic(*result.Redirect)
	}
	return result
}

// The authenticated stream namespace belongs to the return address, not to
// another operation identity. Only the numeric slot is sent on return delivery.
func controlReturnSlot(address string) uint64 {
	index := strings.LastIndexByte(address, '/')
	slot, _ := strconv.ParseUint(address[index+1:], 16, 64)
	return slot
}

func (s *scanRPCSession) operationID(slot uint64) string {
	// The replay registry compares addresses lexicographically. Fixed-width
	// big-endian slots preserve numeric order across 0xff/0x100 boundaries.
	id := fmt.Sprintf("%016x", slot)
	if s.peer.operationPrefix != "" {
		id = s.peer.operationPrefix + "/" + id
	}
	return id
}

// Return delivery uses the ordinary handle registry. The handler owns its
// payload format; transport sees only slot + payload and creates no RPC reply.
type controlResultHandle struct {
	service  *nodeScanService
	peer     *mosiPeer
	receiver *scm.ResultReceiver
	once     sync.Once
	offers   []scm.ClusterReference
}

// Cancellation can end a local wait before a needed remote result arrives.
// Keep only its payload disposer, without the query, waiter or callback state.
type abandonedControlHandle struct {
	service *nodeScanService
	peer    *mosiPeer
	slot    uint64
	entry   *handleSlot
	mu      sync.Mutex
	retired bool
	stop    func() bool
}

func (handle *abandonedControlHandle) retire() {
	handle.mu.Lock()
	handle.retired = true
	stop := handle.stop
	handle.stop = nil
	handle.mu.Unlock()
	if stop != nil {
		stop()
	}
	if handle.service.handles.CompareAndDelete(handle.slot, handle.entry) {
		handle.service.channels.Delete(handleChannel(handle.slot))
	}
}
func (handle *abandonedControlHandle) watch() {
	if handle.peer.ctx == nil {
		return
	}
	stop := context.AfterFunc(handle.peer.ctx, handle.retire)
	handle.mu.Lock()
	if handle.retired {
		handle.mu.Unlock()
		stop()
		return
	}
	handle.stop = stop
	handle.mu.Unlock()
}
func (handle *abandonedControlHandle) receive(payload []byte) error {
	defer handle.retire()
	var result scanCompletion
	if err := unmarshalNode(payload, &result); err != nil {
		panic(err)
	}
	handle.service.discardObjectOffers(handle.peer, result.Offers)
	return nil
}

func (handle *controlResultHandle) receive(payload []byte) error {
	var result scanCompletion
	if err := unmarshalNode(payload, &result); err != nil {
		panic(err)
	}
	accepted := false
	handle.once.Do(func() {
		handle.offers = result.Offers
		handle.receiver.Handle(scm.NewAny(result))
		accepted = true
	})
	if !accepted {
		// An exact duplicate shares the first delivery's grants. Releasing them
		// here could invalidate a result before its consumer reconstructs it.
		var unused []scm.ClusterReference
		for _, parent := range result.Offers {
			for _, ref := range referenceOffers(parent) {
				shared := false
				for _, previous := range handle.offers {
					for _, known := range referenceOffers(previous) {
						if ref.URI == known.URI && ref.Owner == known.Owner && ref.Incarnation == known.Incarnation && ref.Token == known.Token {
							shared = true
						}
					}
				}
				if !shared {
					unused = append(unused, ref)
				}
			}
		}
		handle.service.discardObjectOffers(handle.peer, unused)
	}
	return nil
}

// Operations use the same fire-and-forget transport and numeric handle
// registry. Waiting belongs to the control operation's collector, never a held
// send handle, native shard lock or long-lived per-call connection.
func (service *nodeScanService) control(ctx context.Context, peer *mosiPeer, method string, input, output any, joinOnCancel bool, offers ...scm.ClusterReference) error {
	defer queuedValuePublications(offers)
	if source, ok := input.(interface {
		serializationBindings() *scm.SerializationContext
	}); ok {
		defer source.serializationBindings().Queued()
	}
	queued := false
	defer func() {
		if !queued {
			service.refuseObjectOffers(peer, offers)
		}
	}()
	slot := service.allocateHandleSlot()
	id := (&scanRPCSession{peer: peer}).operationID(slot)
	receiver := scm.NewResultReceiver()
	entry := &handleSlot{peer: peer.objectPeer, receiver: &controlResultHandle{service: service, peer: peer, receiver: receiver}}
	service.storeHandle(slot, entry)
	received := false
	defer func() {
		var orphan *abandonedControlHandle
		live := true
		select {
		case <-peer.done:
			live = false
		default:
		}
		if queued && !received && live {
			orphan = &abandonedControlHandle{service: service, peer: peer, slot: slot}
			orphan.entry = &handleSlot{peer: peer.objectPeer, receiver: orphan}
			service.storeHandle(slot, orphan.entry)
		} else {
			service.deleteHandle(slot)
		}
		entry.close()
		if !received {
			select {
			case <-receiver.Done():
				abandoned := receiver.Wait().Any().(scanCompletion)
				service.discardObjectOffers(peer, abandoned.Offers)
				if orphan != nil {
					orphan.retire()
				}
			default:
			}
		}
		receiver.Close()
		if orphan != nil {
			orphan.watch()
		}
	}()
	offers = transferManifest(offers)
	call, objectCall := input.(clusterObjectCall)
	objectCall = objectCall && method == "object.call"
	var data []byte
	if objectCall {
		data = call.Payload
	} else {
		data = mustNodeMetadata(input)
	}
	payload, err := encodeInvocation(method, id, offers, data)
	if err != nil {
		return err
	}
	service.commitObjectOffers(peer, offers)
	channel := method
	if objectCall {
		channel = objectChannel(call.Reference.URI)
		payload = append([]byte{handleValue}, payload...)
	}
	sendErr := peer.send(ctx, mosiMessage{Channel: channel, Payload: payload})
	if sendErr != nil {
		return sendErr
	}
	queued = true
	completed := false
	defer func() {
		if !completed {
			// This invocation owns cancellation too; no detached AfterFunc may
			// remain blocked sending after its collector has returned. A failed
			// cancellation send closes the connection, whose context owns all
			// remote scopes. Never retry an ambiguously executed operation.
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := postScanMessage(cleanup, peer, "scan.cancel", id); err != nil {
				peer.close()
			}
		}
	}()
	select {
	case <-ctx.Done():
		// Cancellation ends admission, not the lifetime of already running
		// callbacks. Keep the return channel until the worker has unwound its
		// native invocation, including nested owner calls, and reports completion.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := postScanMessage(cleanup, peer, "scan.cancel", id)
		cancel()
		if err != nil {
			peer.close()
		}
		if joinOnCancel {
			select {
			case <-receiver.Done():
				result := receiver.Wait().Any().(scanCompletion)
				received = true
				service.discardObjectOffers(peer, result.Offers)
			case <-peer.done:
			}
		}
		completed = true
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf("Cluster network error")
	case <-receiver.Done():
		result := receiver.Wait().Any().(scanCompletion)
		received = true
		completed = true
		if result.Panic != nil {
			service.discardObjectOffers(peer, result.Offers)
			return result.Panic
		}
		if output != nil {
			if err := unmarshalNode(result.Data, output); err != nil {
				service.discardObjectOffers(peer, result.Offers)
				return err
			}
			if reply, ok := output.(*clusterObjectReply); ok {
				reply.Offers = result.Offers
			}
		} else {
			service.discardObjectOffers(peer, result.Offers)
		}
		return nil
	}
}

// Imported receivers are the same object type. Their callback forwards through
// the existing owner handle; its Go lifetime retains the ordinary peer lease.
func init() {
	scm.RegisterClusterObjectType("receiver", func(handle scm.ClusterHandle) scm.Scmer {
		receiver := newClusterReceiver(-1, func(ctx context.Context, value scm.Scmer) (scm.Scmer, error) {
			return handle.Call(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{value}, CancelWithoutJoin: true})
		})
		receiver.Seal()
		return scm.NewAny(receiver)
	})
}

func panicRemoteOperation(err error) {
	// Preserve the original SQL error/value and remote stack through the same
	// collector used by every object request. No scan-specific send handle.
	if failure, ok := err.(*scanPanic); ok {
		if failure.SQL != nil {
			panic(scanError{failure.SQL, failure.Stack})
		}
		if len(failure.Value) > 0 {
			value, err := schemeValue(failure.Value).reconstruct()
			if err == nil {
				if value.IsString() {
					panic(scanError{value.String(), failure.Stack})
				}
				panic(scanError{value, failure.Stack})
			}
		}
		panic(*failure)
	}
	panic(err)
}

func (receiver *clusterReceiver) HandleClusterRequest(ctx context.Context, payload []byte) {
	scm.ReceiveClusterValue(ctx, payload, func(request scm.ClusterRequest) (any, error) { return receiver.executeClusterRequest(ctx, request) })
}

func (receiver *clusterReceiver) ClusterView(ctx context.Context) (scm.Scmer, error) {
	value, err := receiver.executeClusterRequest(ctx, scm.ClusterRequest{Method: "view"})
	return scm.ClusterViewValue(value, err)
}

func (handle *controlResultHandle) HandleClusterRequest(_ context.Context, payload []byte) {
	if err := handle.receive(payload); err != nil {
		panic(err)
	}
}
func (handle *abandonedControlHandle) HandleClusterRequest(_ context.Context, payload []byte) {
	if err := handle.receive(payload); err != nil {
		panic(err)
	}
}

// Only scan/insert and object invocation protocols transfer capture grants.
// Membership and residency requests have just a return address and their own
// parameters. There is no generic method/fields map on the wire.
func invocationCaptures(method string) bool {
	return method == "scan.execute" || method == "insert.execute" || method == "object.call"
}
func encodeInvocation(method, address string, refs []scm.ClusterReference, data []byte) ([]byte, error) {
	if !invocationCaptures(method) && len(refs) != 0 {
		return nil, fmt.Errorf("operation does not transfer captures")
	}
	b := appendNodeBlob(make([]byte, 0, len(address)+len(data)+16+len(refs)*128), []byte(address))
	if invocationCaptures(method) {
		b = appendReferences(b, refs)
	}
	return append(b, data...), nil
}
func decodeInvocation(method string, data []byte) (address string, refs []scm.ClusterReference, body []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("invalid %s invocation: %v", method, failure)
		}
	}()
	r := nodeReader{data}
	address = string(r.blob())
	if invocationCaptures(method) {
		refs = r.references()
	}
	return address, refs, r.data, nil
}

// A result receiver owns this completion protocol. Ordinary success is two
// bytes followed by the raw result; only failures/capture transfers carry their
// extra parameters. The marker also lets the retired-channel disposer release
// grants without evaluating a late result's Scheme tail.
const scanCompletionMarker byte = 0xf3

func encodeScanCompletion(result scanCompletion) ([]byte, error) {
	flags := byte(0)
	if result.Panic != nil {
		flags |= 1
	}
	if len(result.Offers) != 0 {
		flags |= 2
	}
	b := make([]byte, 2, 2+len(result.Data)+len(result.Offers)*128)
	b[0], b[1] = scanCompletionMarker, flags
	if flags&2 != 0 {
		b = appendReferences(b, result.Offers)
	}
	if flags&1 != 0 {
		w := mosiWriter{data: b}
		failure := result.Panic
		bits := byte(0)
		if failure.SQL != nil {
			bits |= 1
		}
		if len(failure.Value) != 0 {
			bits |= 2
		}
		if failure.Stack != "" {
			bits |= 4
		}
		w.data = append(w.data, bits)
		w.text(failure.Message)
		if bits&1 != 0 {
			w.number(uint64(failure.SQL.Num))
			w.text(failure.SQL.State)
			w.text(failure.SQL.Message)
			w.text(failure.SQL.Query)
		}
		if bits&2 != 0 {
			w.blob(failure.Value)
		}
		if bits&4 != 0 {
			w.text(failure.Stack)
		}
		b = w.data
	}
	return append(b, result.Data...), nil
}
func decodeScanCompletion(data []byte, result *scanCompletion) (err error) {
	defer func() {
		if caught := recover(); caught != nil {
			err = fmt.Errorf("invalid scan completion: %v", caught)
		}
	}()
	r := nodeReader{data}
	if r.octet() != scanCompletionMarker {
		return fmt.Errorf("invalid scan completion")
	}
	flags := r.octet()
	if flags&^3 != 0 {
		return fmt.Errorf("invalid completion flags")
	}
	*result = scanCompletion{}
	if flags&2 != 0 {
		result.Offers = r.references()
	}
	if flags&1 != 0 {
		bits := r.octet()
		if bits&^7 != 0 {
			return fmt.Errorf("invalid error flags")
		}
		result.Panic = &scanPanic{Message: string(r.blob())}
		if bits&1 != 0 {
			number := r.uint()
			if number > 65535 {
				return fmt.Errorf("invalid SQL error number")
			}
			result.Panic.SQL = &sqldb.SQLError{Num: uint16(number), State: string(r.blob()), Message: string(r.blob()), Query: string(r.blob())}
		}
		if bits&2 != 0 {
			result.Panic.Value = r.blob()
			if err := scm.RetainClusterValues(result.Panic.Value); err != nil {
				return err
			}
		}
		if bits&4 != 0 {
			result.Panic.Stack = string(r.blob())
		}
	}
	result.Data = r.data
	return nil
}
