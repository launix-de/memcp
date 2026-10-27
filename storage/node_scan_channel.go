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
type scanDispatch struct {
	Return  string         `json:"return"`
	Cluster bool           `json:"cluster"`
	Task    remoteScanTask `json:"task"`
}
type scanCompletion struct {
	Offers   []scm.ClusterReference `json:"offers,omitempty"`
	Data     json.RawMessage        `json:"data,omitempty"`
	Return   string                 `json:"return"`
	Result   *remoteScanResult      `json:"result,omitempty"`
	Rejected bool                   `json:"rejected,omitempty"`
	Panic    *scanPanic             `json:"panic,omitempty"`
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
		result.Value, _ = json.Marshal(v)
	case string:
		result.Value, _ = json.Marshal(scm.NewString(v))
	}
	return result
}

type scanReturnChannel struct {
	mu     sync.Mutex // delivery/collector-close, outside scan loops
	closed bool
	peer   *mosiPeer
	result chan scanCompletion
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
		delete(r.terminal, r.order[0])
		r.order = r.order[1:]
	}
}
func (r *scanTaskRegistry) begin(parent context.Context, id string) (*scanTaskScope, error) {
	r.Lock()
	defer r.Unlock()
	if _, exists := r.terminal[id]; exists {
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
			channel := entry.(*scanReturnChannel)
			if channel.peer != s.peer {
				return nil, fmt.Errorf("scan completion from wrong peer")
			}
			channel.mu.Lock()
			delivered := false
			if !channel.closed {
				select {
				case channel.result <- result:
					delivered = true
				default:
				}
			}
			channel.mu.Unlock()
			if !delivered {
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
		if _, err := uuid.Parse(request.Return); err != nil {
			return nil, err
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
			if request.Method == "node.control" || request.Method == "scan.task" || request.Method == "scan.result" || request.Method == "scan.cancel" {
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
	case "scan.task":
		var request scanDispatch
		if err := json.Unmarshal(data, &request); err != nil {
			return nil, err
		}
		if _, err := uuid.Parse(request.Return); err != nil {
			return nil, err
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
			// The bounded active-channel registry limits in-flight tasks. Do not
			// retain a second per-peer compute semaphore across user callbacks:
			// nested scans may need this peer to complete their parent shard.
			// Native shard scheduling and cold RAM admission remain authoritative.
			var value any
			if request.Cluster {
				if s.service.cluster == nil {
					panic("cluster membership is disabled")
				}
				value, err = s.service.cluster.compute(s, scope.ctx, "cluster.reduce", mustScanJSON(request.Task))
			} else {
				value, err = s.executeScanTask(scope.ctx, request.Task)
			}
			if err != nil {
				panic(err)
			}
			switch reply := value.(type) {
			case clusterReduceReply:
				completion.Result, completion.Rejected = reply.Result, reply.Rejected
			case remoteScanResult:
				completion.Result = &reply
			default:
				panic("invalid scan completion")
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
	id := uuid.NewString()
	channel := &scanReturnChannel{peer: executor.peer, result: make(chan scanCompletion, 1)}
	service.returns.Store(id, channel)
	defer func() { channel.close(service); service.returns.Delete(id) }()
	if err := postScanMessage(executor.peer, "scan.task", scanDispatch{id, executor.cluster, task}); err != nil {
		panic(err)
	}
	stop := context.AfterFunc(executor.ctx, func() { _ = postScanMessage(executor.peer, "scan.cancel", id) })
	defer stop()
	select {
	case <-executor.ctx.Done():
		panic(executor.ctx.Err())
	case <-executor.peer.done:
		panic("Cluster network error")
	case result := <-channel.result:
		if result.Panic != nil {
			if result.Panic.SQL != nil {
				panic(scanError{result.Panic.SQL, result.Panic.Stack})
			}
			if len(result.Panic.Value) > 0 {
				var value scm.Scmer
				if json.Unmarshal(result.Panic.Value, &value) == nil {
					if value.IsString() {
						panic(scanError{value.String(), result.Panic.Stack})
					}
					panic(scanError{value, result.Panic.Stack})
				}
			}
			panic(*result.Panic)
		}
		if result.Rejected {
			panic(clusterCapacityRejection{})
		}
		if result.Result == nil {
			panic("missing scan result")
		}
		return *result.Result
	}
}

// Control requests use the same fire-and-forget transport and UUID delivery
// registry. Waiting belongs to the control operation's collector, never a held
// send handle, native shard lock or long-lived per-call connection.
func (service *nodeScanService) control(ctx context.Context, peer *mosiPeer, method string, input, output any) error {
	id := uuid.NewString()
	channel := &scanReturnChannel{peer: peer, result: make(chan scanCompletion, 1)}
	service.returns.Store(id, channel)
	defer func() { channel.close(service); service.returns.Delete(id) }()
	request := struct {
		Return string          `json:"return"`
		Method string          `json:"method"`
		Data   json.RawMessage `json:"data"`
	}{id, method, mustScanJSON(input)}
	if err := postScanMessage(peer, "node.control", request); err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { _ = postScanMessage(peer, "scan.cancel", id) })
	defer stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-peer.done:
		return fmt.Errorf("Cluster network error")
	case result := <-channel.result:
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

func (channel *scanReturnChannel) close(service *nodeScanService) {
	channel.mu.Lock()
	channel.closed = true
	var abandoned []scm.ClusterReference
	select {
	case result := <-channel.result:
		abandoned = result.Offers
	default:
	}
	channel.mu.Unlock()
	service.discardObjectOffers(channel.peer, abandoned)
}
