/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "context"
import "sync/atomic"

func (*session) ClusterObjectType() string { return "session" }
func (s *session) HandleClusterRequest(ctx context.Context, request ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return NewNil(), err
	}
	switch request.Method {
	case "view":
		s.Mu.RLock()
		defer s.Mu.RUnlock()
		values := make([]Scmer, 0, len(s.Map)*2)
		for key, value := range s.Map {
			values = append(values, NewString(key), value)
		}
		return NewSlice(values), nil
	case "call":
		return s.call(request.Arguments...), nil
	}
	return NewNil(), fmt.Errorf("unsupported session request %q", request.Method)
}
func ProjectClusterAssoc(view Scmer, demands []Scmer) (Scmer, error) {
	values := view.Slice()
	selected := make([]Scmer, 0, len(demands)*2)
	for _, key := range demands {
		if key.IsNil() {
			return view, nil
		}
		value := NewNil()
		for i := 0; i < len(values); i += 2 {
			if values[i].String() == key.String() {
				value = values[i+1]
				break
			}
		}
		selected = append(selected, key, value)
	}
	return NewSlice(selected), nil
}

func (*session) ProjectClusterView(view Scmer, demands []Scmer) (Scmer, error) {
	return ProjectClusterAssoc(view, demands)
}

type clusterSession struct {
	handle ClusterHandle
	values atomic.Pointer[map[string]Scmer] // immutable S-view, lock-free local reads
}

func newClusterSession(handle ClusterHandle) Scmer { return NewClusterAssoc(handle, sessionCallableID) }

// NewClusterAssoc prepares an immutable local S-view; only owner mutations call
// the network. Session/cachemap owners supply their own captured view semantics.
func NewClusterAssoc(handle ClusterHandle, callableType CallableTypeID) Scmer {
	values := make(map[string]Scmer)
	pairs := handle.View.Slice()
	if len(pairs)%2 != 0 {
		panic("invalid session view")
	}
	for i := 0; i < len(pairs); i += 2 {
		values[pairs[i].String()] = pairs[i+1]
	}
	s := &clusterSession{handle: handle}
	s.values.Store(&values)
	call := func(args ...Scmer) Scmer {
		if len(args) == 0 {
			keys := make([]Scmer, 0, len(*s.values.Load()))
			for key := range *s.values.Load() {
				keys = append(keys, NewString(key))
			}
			return NewSlice(keys)
		}
		if len(args) == 1 {
			return (*s.values.Load())[args[0].String()]
		}
		if len(args) == 2 {
			value, err := s.handle.Call(context.Background(), ClusterRequest{Method: "call", Arguments: args})
			if err != nil {
				panic(err)
			}
			for {
				old := s.values.Load()
				next := make(map[string]Scmer, len(*old)+1)
				for k, v := range *old {
					next[k] = v
				}
				next[args[0].String()] = value
				if s.values.CompareAndSwap(old, &next) {
					break
				}
			}
			return value
		}
		value, err := s.handle.Call(context.Background(), ClusterRequest{Method: "call", Arguments: args})
		if err != nil {
			panic(err)
		}
		return value
	}
	return NewClusterFunc(call, s, callableType)
}
func (s *clusterSession) ClusterObjectType() string { return s.handle.Reference.Type }
func (s *clusterSession) HandleClusterRequest(ctx context.Context, r ClusterRequest) (any, error) {
	if r.Method == "view" {
		pairs := make([]Scmer, 0, len(*s.values.Load())*2)
		for k, v := range *s.values.Load() {
			pairs = append(pairs, NewString(k), v)
		}
		return NewSlice(pairs), nil
	}
	return s.handle.Call(ctx, r)
}
func (*clusterSession) ProjectClusterView(view Scmer, keys []Scmer) (Scmer, error) {
	return ProjectClusterAssoc(view, keys)
}
