/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "context"

// A native callback retains process-local state, such as a connection writer.
// Scheme callers keep their normal local ABI; a captured callback addresses
// that state through its native owner endpoint instead of moving Go pointers.
type clusterCallback struct {
	call   func(...Scmer) Scmer
	ctx    context.Context
	handle *ClusterHandle
}

var clusterCallbackType = RegisterCallableType(&TypeDescriptor{
	Kind: "func", HasSideEffects: true,
	Params: []*TypeDescriptor{{Kind: "any", Variadic: true}},
	Return: &TypeDescriptor{Kind: "any"},
})

func NewClusterCallback(ctx context.Context, call func(...Scmer) Scmer) Scmer {
	callback := &clusterCallback{call: call, ctx: ctx}
	return NewClusterFunc(call, callback, clusterCallbackType)
}

func (*clusterCallback) ClusterObjectType() string { return "callback" }
func (callback *clusterCallback) Serialize(ctx *SerializationContext) (Scmer, error) {
	return ctx.ReferenceExpression(callback)
}
func (callback *clusterCallback) HandleClusterRequest(ctx context.Context, request ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return NewNil(), err
	}
	if err := callback.ctx.Err(); err != nil {
		return NewNil(), err
	}
	if request.Method == "view" {
		return NewNil(), nil
	}
	if request.Method != "call" {
		return NewNil(), fmt.Errorf("unsupported callback request %q", request.Method)
	}
	if callback.handle != nil {
		return callback.handle.Call(ctx, request)
	}
	return callback.call(request.Arguments...), nil
}

func (callback *clusterCallback) BindClusterInvocation(scope Scmer) (Scmer, error) {
	if callback.handle == nil {
		return NewClusterFunc(callback.call, callback, clusterCallbackType), nil
	}
	return newRemoteClusterCallback(*callback.handle, executionContextFrom(scope)), nil
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
