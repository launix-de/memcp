/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "sync"
import "context"

// Handle receives one value. Delivery has no transport-level result or method.
// A handler defines its own payload and may receive another handle as a reply path.
type Handle interface{ Handle(Scmer) }
type HandleFunc func(Scmer)

func (fn HandleFunc) Handle(value Scmer) { fn(value) }

// HandleSerializer is implemented by imported handles, which retain their peer.
type HandleSerializer interface {
	SerializeHandle(*SerializationContext) (Scmer, error)
}

// ResultReceiver is a one-shot return channel. Sending remains fire-and-forget;
// only the consumer chooses whether and where to wait.
type ResultReceiver struct {
	mu       sync.Mutex
	done     chan struct{}
	value    Scmer
	closed   bool
	releases []func()
}

func NewResultReceiver() *ResultReceiver { return &ResultReceiver{done: make(chan struct{})} }
func (receiver *ResultReceiver) Handle(value Scmer) {
	receiver.mu.Lock()
	if receiver.closed {
		receiver.mu.Unlock()
		return
	}
	receiver.value = value
	receiver.closed = true
	releases := receiver.releases
	receiver.releases = nil
	close(receiver.done)
	receiver.mu.Unlock()
	for _, release := range releases {
		release()
	}
}
func (receiver *ResultReceiver) RetainHandleRelease(release func()) {
	receiver.mu.Lock()
	if !receiver.closed {
		receiver.releases = append(receiver.releases, release)
		receiver.mu.Unlock()
		return
	}
	receiver.mu.Unlock()
	release()
}
func (receiver *ResultReceiver) Close()      { receiver.Handle(NewNil()) }
func (receiver *ResultReceiver) Wait() Scmer { return receiver.WaitContext(context.Background()) }
func (receiver *ResultReceiver) WaitContext(ctx context.Context) Scmer {
	defer receiver.Close()
	select {
	case <-receiver.done:
		if receiver.value.GetTag() == tagAny {
			if failure, ok := receiver.value.Any().(*handleFailure); ok {
				panic(failure)
			}
		}
		return receiver.value
	case <-ctx.Done():
		panic(ctx.Err())
	}
}

type handleFailure struct{ message string }

func (failure *handleFailure) Error() string { return failure.message }

// ClosureHandle interprets (parameters return-handle). The result is sent raw;
// error values have their own reconstruction constructor, not a transport field.
func ClosureHandle(ctx context.Context, call func(...Scmer) Scmer) Handle {
	return HandleFunc(func(payload Scmer) {
		values := payload.Slice()
		var reply Handle
		if len(values) != 2 {
			panic("closure payload requires parameters and return handle")
		}
		if !values[1].IsNil() {
			reply = values[1].Any().(Handle)
		}
		defer func() {
			if failure := recover(); failure != nil {
				if reply == nil {
					panic(failure)
				}
				reply.Handle(NewAny(&handleFailure{fmt.Sprint(failure)}))
			}
		}()
		if err := ctx.Err(); err != nil {
			panic(err)
		}
		result := call(values[0].Slice()...)
		if reply != nil {
			reply.Handle(result)
		}
	})
}
func callClosure(ctx context.Context, handle Handle, args []Scmer) Scmer {
	if err := ctx.Err(); err != nil {
		panic(err)
	}
	receiver := NewResultReceiver()
	defer receiver.Close()
	handle.Handle(NewSlice([]Scmer{NewSlice(args), NewAny(receiver)}))
	return receiver.WaitContext(ctx)
}

func (bindings *SerializationContext) HandleExpression(handle Handle) (Scmer, error) {
	if serializable, ok := handle.(HandleSerializer); ok {
		return serializable.SerializeHandle(bindings)
	}
	if bindings == nil || bindings.ExportHandle == nil {
		return NewNil(), fmt.Errorf("handle requires an export context")
	}
	return bindings.ExportHandle(handle)
}
func (bindings *SerializationContext) WithHandleExport(export func(Handle) (Scmer, error)) *SerializationContext {
	root := bindings
	if bindings.root != nil {
		root = bindings.root
	}
	return &SerializationContext{Export: bindings.Export, ExportHandle: export, root: root}
}
func (bindings *SerializationContext) InvocationScope() Scmer {
	if bindings.root != nil {
		return bindings.root.Invocation
	}
	return bindings.Invocation
}

var HandleResolver func(string, string, uint64) (Handle, error)

func initHandles() {
	Declare(&Globalenv, &Declaration{Name: "cluster_handle", Fn: func(args ...Scmer) Scmer {
		if HandleResolver == nil {
			panic("handle resolver is unavailable")
		}
		handle, err := HandleResolver(args[0].String(), args[1].String(), uint64(args[2].Int()))
		if err != nil {
			panic(err)
		}
		return NewAny(handle)
	}, Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "string"}, {Kind: "string"}, {Kind: "int"}}, Return: &TypeDescriptor{Kind: "any"}}})
	Declare(&Globalenv, &Declaration{Name: "handle_failure", Fn: func(args ...Scmer) Scmer { return NewAny(&handleFailure{args[0].String()}) }, Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "string"}}, Return: &TypeDescriptor{Kind: "any"}}})
	Declare(&Globalenv, &Declaration{Name: "remote_closure", Fn: func(args ...Scmer) Scmer {
		callback := &clusterCallback{endpoint: args[1].Any().(Handle), ctx: context.Background()}
		if endpoint, ok := callback.endpoint.(interface{ Context() context.Context }); ok {
			callback.ctx = endpoint.Context()
		}
		if !args[2].IsNil() {
			result := args[2].Slice()[0]
			callback.batchResult = &result
		}
		if !args[0].IsNil() {
			value, err := callback.BindClusterInvocation(args[0])
			if err != nil {
				panic(err)
			}
			return value
		}
		return callback.value()
	}, Type: &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "any", Label: "tx"}, {Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "func"}}})
}
