/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "errors"
import "sync"
import "context"

// A native callback retains process-local state, such as a connection writer.
// Scheme callers keep their normal local ABI; a captured callback addresses
// that state through its native owner endpoint instead of moving Go pointers.
type clusterCallback struct {
	call        func(...Scmer) Scmer
	ctx         context.Context
	endpoint    Handle
	batchResult *Scmer
	output      *callbackOutputQueue
}

// Serialize access to an imperative sink by admission, without executing the
// callback or its I/O under a publication lock. Network queues stay independent.
type callbackOutputQueue struct {
	mu   sync.Mutex
	tail chan struct{}
}

func (queue *callbackOutputQueue) enter() func() {
	done := make(chan struct{})
	queue.mu.Lock()
	previous := queue.tail
	queue.tail = done
	queue.mu.Unlock()
	if previous != nil {
		<-previous
	}
	return func() {
		close(done)
		queue.mu.Lock()
		if queue.tail == done {
			queue.tail = nil
		}
		queue.mu.Unlock()
	}
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
	if batchResult != nil {
		callback.output = new(callbackOutputQueue)
	}
	return NewClusterFunc(call, callback, clusterCallbackType)
}

func (*clusterCallback) ClusterObjectType() string { return "callback" }
func (callback *clusterCallback) Serialize(ctx *SerializationContext) (Scmer, error) {
	if callback.endpoint != nil {
		endpoint, err := callback.forwardClosure(ctx)
		if err != nil {
			return NewNil(), err
		}
		expression, err := ctx.HandleExpression(endpoint)
		if err != nil {
			return NewNil(), err
		}
		return callback.expression(expression), nil
	}
	endpoint := ClosureHandle(callback.ctx, func(args ...Scmer) Scmer {
		method := "call"
		if callback.batchResult != nil {
			method = "batch"
			args = args[0].Slice()
		}
		value, err := ClusterObjectValue(callback, callback.ctx, ClusterRequest{Method: method, Arguments: args})
		if err != nil {
			panic(err)
		}
		return value
	})
	expression, err := ctx.HandleExpression(endpoint)
	if err != nil {
		return NewNil(), err
	}
	return callback.expression(expression), nil
}

// The callable owns the (arguments return-handle) protocol. Relaying a call
// substitutes a local one-shot receiver, so its return slot is retired at the
// reply instead of retaining one forwarding slot per output batch until EOF.
// Publication never waits; the invocation owner joins all accepted replies.
func (callback *clusterCallback) forwardClosure(bindings *SerializationContext) (Handle, error) {
	owner, ok := bindings.InvocationScope().Any().(interface{ RetainQueryWork(func()) bool })
	if !ok {
		return nil, fmt.Errorf("forwarded closure requires an invocation owner")
	}
	var mu sync.Mutex
	var pending sync.WaitGroup
	closed := false
	var failure any
	finish := func() {
		mu.Lock()
		closed = true
		mu.Unlock()
		pending.Wait()
		mu.Lock()
		err := failure
		mu.Unlock()
		if err != nil {
			panic(err)
		}
	}
	if !owner.RetainQueryWork(finish) {
		return nil, fmt.Errorf("forwarded closure owner has finished")
	}
	return HandleFunc(func(payload Scmer) {
		values := payload.Slice()
		if len(values) != 2 {
			panic("closure payload requires parameters and return handle")
		}
		var reply Handle
		if !values[1].IsNil() {
			reply = values[1].Any().(Handle)
		}
		mu.Lock()
		if closed {
			mu.Unlock()
			if reply != nil {
				reply.Handle(NewAny(&handleFailure{"forwarded closure owner has finished"}))
			}
			return
		}
		pending.Add(1)
		mu.Unlock()
		receiver := NewResultReceiver()
		receiver.onResult = func(value Scmer) {
			defer pending.Done()
			defer func() {
				if err := recover(); err != nil {
					mu.Lock()
					if failure == nil {
						failure = err
					}
					mu.Unlock()
				}
			}()
			if reply != nil {
				reply.Handle(value)
			}
		}
		defer func() {
			if err := recover(); err != nil {
				receiver.Fail(fmt.Errorf("%v", err))
			}
		}()
		if err := callback.ctx.Err(); err != nil {
			panic(err)
		}
		callback.endpoint.Handle(NewSlice([]Scmer{values[0], NewAny(receiver)}))
	}), nil
}

func (callback *clusterCallback) expression(handle Scmer) Scmer {
	view := NewNil()
	if callback.batchResult != nil {
		view = NewSlice([]Scmer{NewSymbol("list"), *callback.batchResult})
	}
	return NewSlice([]Scmer{NewSymbol("remote_closure"), NewNil(), handle, view})
}
func (callback *clusterCallback) value() Scmer {
	call := callback.call
	if callback.endpoint != nil {
		call = func(args ...Scmer) Scmer {
			if callback.batchResult != nil {
				// A closure passed as a value may have no streaming invocation.
				// Keep its normal call ABI while addressing the batched endpoint.
				callClosure(callback.ctx, callback.endpoint, []Scmer{NewSlice([]Scmer{NewSlice(args)})})
				return *callback.batchResult
			}
			return callClosure(callback.ctx, callback.endpoint, args)
		}
	}
	return NewClusterFunc(call, callback, clusterCallbackType)
}

func (callback *clusterCallback) HandleClusterRequest(ctx context.Context, request ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return NewNil(), err
	}
	if err := callback.ctx.Err(); err != nil {
		return NewNil(), err
	}
	if request.Method == "view" {
		if callback.batchResult != nil {
			return NewSlice([]Scmer{*callback.batchResult}), nil
		}
		return NewNil(), nil
	}
	if request.Method != "call" && request.Method != "batch" {
		return NewNil(), fmt.Errorf("unsupported callback request %q", request.Method)
	}
	if request.Method == "batch" {
		if callback.batchResult == nil {
			return NewNil(), fmt.Errorf("callback does not accept output batches")
		}
		release := callback.output.enter()
		defer release()
		if err := ctx.Err(); err != nil {
			return NewNil(), err
		}
		if err := callback.ctx.Err(); err != nil {
			return NewNil(), err
		}
		for _, args := range request.Arguments {
			callback.call(args.Slice()...)
		}
		return NewNil(), nil
	}
	return callback.call(request.Arguments...), nil
}

func (callback *clusterCallback) BindClusterInvocation(scope Scmer) (Scmer, error) {
	if callback.endpoint == nil {
		return callback.value(), nil
	}
	bound := *callback
	bound.ctx = executionContextFrom(scope)
	if endpoint, ok := bound.endpoint.(interface {
		WithInvocation(Scmer, context.Context) Handle
	}); ok {
		bound.endpoint = endpoint.WithInvocation(scope, bound.ctx)
	}
	if bound.batchResult != nil {
		owner, ok := scope.Any().(interface{ RetainQueryWork(func()) bool })
		if !ok {
			return NewNil(), fmt.Errorf("output batches require a native invocation owner")
		}
		return bound.bindOutput(bound.ctx, owner)
	}
	return bound.value(), nil
}

// Producers publish argument tuples; only producer admission may park at the
// bounded credit window. A send never waits for its return handle. Completion
// handlers return credits and wake workers; cleanup joins every accepted call.
func (callback *clusterCallback) bindOutput(ctx context.Context, owner interface{ RetainQueryWork(func()) bool }) (Scmer, error) {
	const batchSize = 64
	const window = 4
	var mu sync.Mutex
	batch := make([]Scmer, 0, batchSize)
	pending := 0
	closing := false
	var failure any
	wake := make(chan struct{})
	// Broadcast credits: coalescing a single notification could strand parallel
	// producers even when several credits became available together.
	signal := func() { close(wake); wake = make(chan struct{}) }
	var send func([]Scmer)
	send = func(values []Scmer) {
		for {
			mu.Lock()
			if failure != nil {
				err := failure
				mu.Unlock()
				panic(err)
			}
			if pending < window {
				pending++
				mu.Unlock()
				break
			}
			ready := wake
			mu.Unlock()
			select {
			case <-ready:
			case <-ctx.Done():
				panic(ctx.Err())
			}
		}
		receiver := NewResultReceiver()
		receiver.onResult = func(value Scmer) {
			mu.Lock()
			pending--
			if value.GetTag() == tagAny {
				if err, ok := value.Any().(*handleFailure); ok && failure == nil {
					failure = err
				}
			}
			signal()
			mu.Unlock()
		}
		var err error
		func() {
			defer func() {
				if p := recover(); p != nil {
					if e, ok := p.(error); ok {
						err = e
					} else {
						err = fmt.Errorf("%v", p)
					}
				}
			}()
			callback.endpoint.Handle(NewSlice([]Scmer{NewSlice([]Scmer{NewSlice(values)}), NewAny(receiver)}))
		}()
		if err != nil {
			receiver.Close()
			if errors.Is(err, ErrClusterRequestTooLarge) && len(values) > 1 {
				middle := len(values) / 2
				send(values[:middle])
				send(values[middle:])
				return
			}
			panic(err)
		}
	}
	finish := func() {
		mu.Lock()
		closing = true
		tail := batch
		batch = nil
		mu.Unlock()
		var sendFailure any
		func() {
			defer func() { sendFailure = recover() }()
			if ctx.Err() == nil && len(tail) > 0 {
				send(tail)
			}
		}()
		for {
			mu.Lock()
			left, err, ready := pending, failure, wake
			mu.Unlock()
			if left == 0 {
				if sendFailure != nil {
					panic(sendFailure)
				}
				if err != nil {
					panic(err)
				}
				return
			}
			<-ready
		}
	}
	if !owner.RetainQueryWork(finish) {
		return NewNil(), fmt.Errorf("output invocation has already finished")
	}
	result := *callback.batchResult
	call := func(args ...Scmer) Scmer {
		tuple := NewSlice(append([]Scmer(nil), args...))
		mu.Lock()
		if closing || failure != nil {
			err := failure
			mu.Unlock()
			if err != nil {
				panic(err)
			}
			panic("output invocation has finished")
		}
		batch = append(batch, tuple)
		var ready []Scmer
		if len(batch) == batchSize {
			ready = batch
			batch = make([]Scmer, 0, batchSize)
		}
		mu.Unlock()
		if ready != nil {
			send(ready)
		}
		return result
	}
	return NewClusterFunc(call, &clusterCallback{endpoint: callback.endpoint, ctx: ctx, batchResult: callback.batchResult}, clusterCallbackType), nil
}
