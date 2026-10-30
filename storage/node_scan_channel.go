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
import "sync/atomic"
import "encoding/json"
import "runtime/debug"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/go-mysqlstack/sqldb"

// These messages have no RPC response handle. The reducer registers a return
// channel before enqueueing a task; the node reader delivers a completion (or
// the original panic) by that UUID. Cancelling never interrupts a shard's item
// loop: it fences new shard tasks and is inherited by nested scan invocations.
type scanCompletion struct {
	Offers []scm.ClusterReference `json:"offers,omitempty"`
	Data   json.RawMessage        `json:"data,omitempty"`
	Return string                 `json:"return"`
	Panic  *scanPanic             `json:"panic,omitempty"`
}
type scanPanic struct {
	SQL     *sqldb.SQLError `json:"sql,omitempty"`
	Message string          `json:"message"`
	Value   json.RawMessage `json:"value,omitempty"`
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
		result.Value = json.RawMessage(encoded)
	case string:
		encoded, _ := freezeScanValue(scm.NewString(v))
		result.Value = json.RawMessage(encoded)
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
	callback  func(context.Context, scm.Scmer) error
}

// A delivery lives in its invoking handler, including while queued. Cancelling
// a waiter unlinks it without overtaking its active predecessor. No receiver
// worker or detached forwarding task outlives that handler.
type receiverDelivery struct {
	previous, next *receiverDelivery
	ready          chan struct{}
}

func newClusterReceiver(limit int64, callback func(context.Context, scm.Scmer) error) *clusterReceiver {
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
func (receiver *clusterReceiver) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
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
	return scm.NewNil(), callback(callbackCtx, request.Arguments[0])
}

type receiverRoute struct {
	peer     *mosiPeer
	receiver *clusterReceiver
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
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	message := mosiMessage{Method: method, Data: data}
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
	operation, err := uuid.Parse(id)
	return err == nil && operation.Version() == 7
}

func (s *scanRPCSession) channelMessage(ctx context.Context, method string, data json.RawMessage) (any, error) {
	switch method {
	case "scan.result":
		var result scanCompletion
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		if entry, ok := s.service.returns.Load(result.Return); ok {
			route := entry.(*receiverRoute)
			if route.peer != s.peer {
				return nil, fmt.Errorf("completion from wrong peer")
			}
			_, err := route.receiver.HandleClusterRequest(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{scm.NewAny(result)}})
			if err != nil {
				s.service.discardObjectOffers(s.peer, result.Offers)
			}
		} else {
			s.service.discardObjectOffers(s.peer, result.Offers)
		}
		return nil, nil
	case "scan.cancel":
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, err
		}
		if !s.validOperation(id) {
			return nil, fmt.Errorf("invalid operation identity")
		}
		s.tasks.cancelTask(id)
		return nil, nil
	case "node.control":
		var request struct {
			Return string          `json:"return"`
			Method string          `json:"method"`
			Data   json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if !s.validOperation(request.Return) {
			return nil, fmt.Errorf("invalid operation identity")
		}
		completion := scanCompletion{Return: request.Return}
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					completion.Panic = freezeScanPanic(failure)
				}
			}()
			scope, err := s.tasks.begin(ctx, request.Return)
			if err != nil {
				panic(err)
			}
			defer s.tasks.finish(request.Return)
			if request.Method == "node.control" || request.Method == "scan.result" || request.Method == "scan.cancel" {
				panic("invalid nested control envelope")
			}
			value, err := s.handle(scope.ctx, request.Method, request.Data)
			if err != nil {
				panic(err)
			}
			if exported, ok := value.(interface{ ClusterObjectOffers() []scm.ClusterReference }); ok {
				completion.Offers = exported.ClusterObjectOffers()
			}
			// boundedScanReply already encoded and checked this response. Keep
			// those immutable bytes through the collector; do not marshal even
			// a RawMessage wrapper here (which scans and copies it again).
			if encoded, ok := value.(encodedReply); ok {
				completion.Data = encoded.data
				return
			}
			completion.Data, err = json.Marshal(value)
			if err != nil {
				panic(err)
			}
		}()
		return nil, postScanMessage(ctx, s.peer, "scan.result", completion)

	}
	return nil, fmt.Errorf("unknown scan channel message")
}
func mustScanJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func (executor *peerScanExecutor) dispatch(task remoteScanTask) remoteScanResult {
	service := executor.peer.scanService
	if service == nil {
		panic("missing scan return-channel service")
	}
	var result remoteScanResult
	if err := service.control(executor.ctx, executor.peer, "scan.execute", task, &result, true); err != nil {
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
	if result.Rejected {
		panic(clusterCapacityRejection{})
	}
	return result
}

// Control requests use the same fire-and-forget transport and UUID delivery
// registry. Waiting belongs to the control operation's collector, never a held
// send handle, native shard lock or long-lived per-call connection.
func (service *nodeScanService) control(ctx context.Context, peer *mosiPeer, method string, input, output any, joinOnCancel bool) error {
	operation, err := uuid.NewV7()
	if err != nil {
		return err
	}
	id := operation.String()
	if peer.operationPrefix != "" {
		id = peer.operationPrefix + "/" + id
	}
	results := make(chan scanCompletion, 1)
	receiver := newClusterReceiver(1, func(_ context.Context, value scm.Scmer) error { results <- value.Any().(scanCompletion); return nil })
	receiver.Seal()
	route := &receiverRoute{peer, receiver}
	service.returns.Store(id, route)
	defer func() {
		receiver.Close()
		service.returns.Delete(id)
		select {
		case abandoned := <-results:
			service.discardObjectOffers(peer, abandoned.Offers)
		default:
		}
	}()
	request := struct {
		Return string          `json:"return"`
		Method string          `json:"method"`
		Data   json.RawMessage `json:"data"`
	}{id, method, mustScanJSON(input)}
	if err := postScanMessage(ctx, peer, "node.control", request); err != nil {
		return err
	}
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
			case result := <-results:
				service.discardObjectOffers(peer, result.Offers)
			case <-peer.done:
			}
		}
		completed = true
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf("Cluster network error")
	case result := <-results:
		completed = true
		if result.Panic != nil {
			service.discardObjectOffers(peer, result.Offers)
			return result.Panic
		}
		if output != nil {
			if err := json.Unmarshal(result.Data, output); err != nil {
				service.discardObjectOffers(peer, result.Offers)
				return err
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
		receiver := newClusterReceiver(-1, func(ctx context.Context, value scm.Scmer) error {
			_, err := handle.Call(ctx, scm.ClusterRequest{Method: "receive", Arguments: []scm.Scmer{value}, CancelWithoutJoin: true})
			return err
		})
		receiver.Seal()
		return scm.NewAny(receiver)
	})
}
