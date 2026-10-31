/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "errors"
import "context"

// A native callback retains process-local state, such as a connection writer.
// Scheme callers keep their normal local ABI; a captured callback addresses
// that state through its native owner endpoint instead of moving Go pointers.
type clusterCallback struct {
	source      *clusterCallback
	lifetime    *callbackLifetime
	call        func(...Scmer) Scmer
	ctx         context.Context
	handle      *ClusterHandle
	batchResult *Scmer
}

var clusterCallbackType = RegisterCallableType(&TypeDescriptor{
	Kind: "func", HasSideEffects: true,
	Params: []*TypeDescriptor{{Kind: "any", Variadic: true}},
	Return: &TypeDescriptor{Kind: "any"},
})

// batchResult declares an output-only callback with a constant result. Its
// remote invocations may send bounded batches; nil retains synchronous calls.
func NewClusterCallback(ctx context.Context, call func(...Scmer) Scmer, batchResult *Scmer) Scmer {
	callback := &clusterCallback{call: call, ctx: ctx, batchResult: batchResult}
	return NewClusterFunc(call, callback, clusterCallbackType)
}

func (*clusterCallback) ClusterObjectType() string { return "callback" }
func (callback *clusterCallback) Serialize(ctx *SerializationContext) (Scmer, error) {
	object, err := ctx.BindInvocationObject(callback, func(scope Scmer) (ClusterObject, error) {
		owner, ok := scope.Any().(interface{ RetainQueryWork(func()) bool })
		if !ok {
			return nil, fmt.Errorf("callback export requires a native invocation owner")
		}
		lifetime := new(callbackLifetime)
		if !owner.RetainQueryWork(lifetime.close) {
			return nil, fmt.Errorf("callback invocation has already finished")
		}
		return &clusterCallback{source: callback, lifetime: lifetime}, nil
	})
	if err != nil {
		return NewNil(), err
	}
	return ctx.ReferenceExpression(object)
}

// Admission and joining belong to the native callback endpoint, independently
// of the connection carrying its calls. Disconnect cannot release query state
// while a locally running invocation still borrows it.
type callbackLifetime struct {
	mu     sync.Mutex
	closed bool
	calls  sync.WaitGroup
}

func (lifetime *callbackLifetime) close() {
	lifetime.mu.Lock()
	lifetime.closed = true
	lifetime.mu.Unlock()
	lifetime.calls.Wait()
}

func (callback *clusterCallback) HandleClusterRequest(ctx context.Context, request ClusterRequest) (any, error) {
	if callback.lifetime != nil {
		lifetime := callback.lifetime
		lifetime.mu.Lock()
		if lifetime.closed {
			lifetime.mu.Unlock()
			return NewNil(), context.Canceled
		}
		lifetime.calls.Add(1)
		lifetime.mu.Unlock()
		defer lifetime.calls.Done()
		return callback.source.HandleClusterRequest(ctx, request)
	}
	if err := ctx.Err(); err != nil {
		return NewNil(), err
	}
	if err := callback.ctx.Err(); err != nil {
		return NewNil(), err
	}
	if request.Method == "view" {
		if callback.handle != nil {
			return callback.handle.View, nil
		}
		if callback.batchResult != nil {
			return NewSlice([]Scmer{*callback.batchResult}), nil
		}
		return NewNil(), nil
	}
	if request.Method != "call" && request.Method != "batch" {
		return NewNil(), fmt.Errorf("unsupported callback request %q", request.Method)
	}
	if callback.handle != nil {
		return callback.handle.Call(ctx, request)
	}
	if request.Method == "batch" {
		if callback.batchResult == nil {
			return NewNil(), fmt.Errorf("callback does not accept output batches")
		}
		for _, args := range request.Arguments {
			callback.call(args.Slice()...)
		}
		return NewNil(), nil
	}
	return callback.call(request.Arguments...), nil
}

func (callback *clusterCallback) BindClusterInvocation(scope Scmer) (Scmer, error) {
	if callback.handle == nil {
		return NewClusterFunc(callback.call, callback, clusterCallbackType), nil
	}
	ctx := executionContextFrom(scope)
	if callback.handle.View.IsSlice() && len(callback.handle.View.Slice()) == 1 {
		owner, ok := scope.Any().(interface{ RetainQueryWork(func()) bool })
		if !ok {
			return NewNil(), fmt.Errorf("output batches require a native invocation owner")
		}
		return callback.bindOutput(ctx, owner)
	}
	return newRemoteClusterCallback(*callback.handle, ctx), nil
}

// One invocation owns a bounded output queue and its sender. Producers only
// publish complete argument tuples; the sender owns batching and network I/O.
// Query cleanup closes production and joins before releasing captured state.
func (callback *clusterCallback) bindOutput(ctx context.Context, owner interface{ RetainQueryWork(func()) bool }) (Scmer, error) {
	const batchSize = 64
	queue := make(chan Scmer, batchSize)
	done := make(chan struct{})
	var failure any
	go func() {
		defer close(done)
		defer func() { failure = recover() }()
		batch := make([]Scmer, 0, batchSize)
		var send func([]Scmer)
		send = func(values []Scmer) {
			_, err := callback.handle.Call(ctx, ClusterRequest{Method: "batch", Arguments: values})
			if errors.Is(err, ErrClusterRequestTooLarge) && len(values) > 1 {
				middle := len(values) / 2
				send(values[:middle])
				send(values[middle:])
				return
			}
			if err != nil {
				panic(err)
			}
		}
		flush := func() {
			if len(batch) == 0 {
				return
			}
			send(batch)
			clear(batch)
			batch = batch[:0]
		}
		for {
			select {
			case args, ok := <-queue:
				if !ok {
					if ctx.Err() == nil {
						flush()
					}
					return
				}
				batch = append(batch, args)
				if len(batch) == batchSize {
					flush()
				}
			case <-ctx.Done():
				panic(ctx.Err())
			}
		}
	}()
	finish := func() {
		close(queue)
		<-done
		if failure != nil {
			panic(failure)
		}
	}
	if !owner.RetainQueryWork(finish) {
		finish()
		return NewNil(), fmt.Errorf("output invocation has already finished")
	}
	result := callback.handle.View.Slice()[0]
	call := func(args ...Scmer) Scmer {
		value := NewSlice(append([]Scmer(nil), args...))
		select {
		case queue <- value:
			return result
		case <-done:
			if failure != nil {
				panic(failure)
			}
			panic("output invocation has finished")
		}
	}
	return NewClusterFunc(call, &clusterCallback{handle: callback.handle, ctx: ctx}, clusterCallbackType), nil
}

func newRemoteClusterCallback(handle ClusterHandle, ctx context.Context) Scmer {
	callback := &clusterCallback{handle: &handle, ctx: ctx}
	call := func(args ...Scmer) Scmer {
		value, err := handle.Call(ctx, ClusterRequest{Method: "call", Arguments: args})
		if err != nil {
			panic(err)
		}
		return value
	}
	return NewClusterFunc(call, callback, clusterCallbackType)
}

func init() {
	RegisterClusterObjectType("callback", func(handle ClusterHandle) Scmer {
		return newRemoteClusterCallback(handle, context.Background())
	})
}
