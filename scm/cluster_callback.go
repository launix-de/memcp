/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "errors"
import "context"

// A native callback retains process-local state, such as a connection writer.
// Scheme callers keep their normal local ABI; a captured callback addresses
// that state through its native owner endpoint instead of moving Go pointers.
type clusterCallback struct {
	call        func(...Scmer) Scmer
	ctx         context.Context
	endpoint    Handle
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
	if callback.endpoint != nil {
		expression, err := ctx.HandleExpression(callback.endpoint)
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
			var err error
			func() {
				defer func() {
					if failure := recover(); failure != nil {
						if e, ok := failure.(error); ok {
							err = e
						} else {
							panic(failure)
						}
					}
				}()
				callClosure(ctx, callback.endpoint, []Scmer{NewSlice(values)})
			}()
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
	result := *callback.batchResult
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
	return NewClusterFunc(call, &clusterCallback{endpoint: callback.endpoint, ctx: ctx, batchResult: callback.batchResult}, clusterCallbackType), nil
}
