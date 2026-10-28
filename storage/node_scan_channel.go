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
import "sync/atomic"
import "context"
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
		encoded, _ := freezeScanValue(v, 0)
		result.Value = json.RawMessage(encoded)
	case string:
		encoded, _ := freezeScanValue(scm.NewString(v), 0)
		result.Value = json.RawMessage(encoded)
	}
	return result
}

// clusterReceiver owns only acceptance and callbacks. Object offers/leases own
// distributed lifetime independently, including receivers with an unlimited -1
// budget. The consumer must Close before releasing callback state. Close joins
// accepted callbacks; a callback must not call Close on its own receiver.
type clusterReceiver struct {
	mu        sync.Mutex
	remaining atomic.Int64
	sealed    bool // production completed; zero during production is not completion
	closed    bool
	done      chan struct{}
	callback  func(context.Context, scm.Scmer) error
}

func newClusterReceiver(limit int64, callback func(context.Context, scm.Scmer) error) *clusterReceiver {
	if limit < -1 {
		panic("invalid receiver limit")
	}
	receiver := &clusterReceiver{done: make(chan struct{}), callback: callback}
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
func (receiver *clusterReceiver) finishLocked() {
	if receiver.sealed && receiver.remaining.Load() == 0 && !receiver.closed {
		receiver.closed = true
		receiver.callback = nil
		close(receiver.done)
	}
}
func (receiver *clusterReceiver) Seal() {
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	receiver.sealed = true
	receiver.finishLocked()
}
func (receiver *clusterReceiver) Close() {
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if !receiver.closed {
		receiver.closed = true
		receiver.callback = nil
		close(receiver.done)
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
	defer receiver.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if receiver.closed {
		return nil, fmt.Errorf("closed receiver")
	}
	for {
		old := receiver.remaining.Load()
		if old == 0 {
			return nil, fmt.Errorf("receiver limit exhausted")
		}
		if old == -1 || receiver.remaining.CompareAndSwap(old, old-1) {
			break
		}
	}
	defer receiver.finishLocked()
	return scm.NewNil(), receiver.callback(ctx, request.Arguments[0])
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
	terminal map[string]bool
	order    []string
}

func (r *scanTaskRegistry) remember(id string, cancelled bool) {
	if r.terminal == nil {
		r.terminal = make(map[string]bool)
	}
	if _, exists := r.terminal[id]; !exists {
		r.order = append(r.order, id)
	}
	r.terminal[id] = cancelled
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
	r.remember(id, false)
}
func (r *scanTaskRegistry) cancelTask(id string) {
	r.Lock()
	defer r.Unlock()
	if scope := r.active[id]; scope != nil {
		scope.cancel()
	} else {
		if _, done := r.terminal[id]; !done {
			r.remember(id, true)
		}
	}
}
func postScanMessage(peer *mosiPeer, method string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return peer.send(mosiMessage{Method: method, Data: data})
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
		if _, err := uuid.Parse(id); err != nil {
			return nil, err
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
		if operation, err := uuid.Parse(request.Return); err != nil || operation.Version() != 7 {
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
			completion.Data, err = json.Marshal(value)
			if err != nil {
				panic(err)
			}
		}()
		return nil, postScanMessage(s.peer, "scan.result", completion)

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
	if err := service.control(executor.ctx, executor.peer, "scan.execute", task, &result); err != nil {
		// Preserve the original SQL error/value and remote stack through the same
		// collector used by every object request. No scan-specific send handle.
		if failure, ok := err.(*scanPanic); ok {
			if failure.SQL != nil {
				panic(scanError{failure.SQL, failure.Stack})
			}
			if len(failure.Value) > 0 {
				value, err := schemeValue(failure.Value).thaw(0)
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
func (service *nodeScanService) control(ctx context.Context, peer *mosiPeer, method string, input, output any) error {
	operation, err := uuid.NewV7()
	if err != nil {
		return err
	}
	id := operation.String()
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
	if err := postScanMessage(peer, "node.control", request); err != nil {
		return err
	}
	cancelRemote := sync.OnceFunc(func() { _ = postScanMessage(peer, "scan.cancel", id) })
	stop := context.AfterFunc(ctx, cancelRemote)
	defer stop()
	select {
	case <-ctx.Done():
		// Same race as shard dispatch: returning must not suppress the very
		// cancellation which lets the remote nested scan release its scope.
		cancelRemote()
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf("Cluster network error")
	case result := <-results:
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

// Imported receivers use the same object-call serializer and offer/lease
// protocol as every other runtime object; they add no channel lifetime.
type remoteReceiver struct{ handle scm.ClusterHandle }

func (*remoteReceiver) ClusterObjectType() string { return "receiver" }
func (receiver *remoteReceiver) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	return ctx.ReferenceExpression(receiver)
}
func (receiver *remoteReceiver) HandleClusterRequest(ctx context.Context, request scm.ClusterRequest) (any, error) {
	if request.Method == "view" {
		return receiver.handle.View, nil
	}
	return receiver.handle.Call(ctx, request)
}
func init() {
	scm.RegisterClusterObjectType("receiver", func(handle scm.ClusterHandle) scm.Scmer { return scm.NewAny(&remoteReceiver{handle}) })
}
