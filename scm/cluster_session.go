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

func newClusterSession(handle ClusterHandle) Scmer {
	return NewClusterAssoc(handle, sessionCallableID, false)
}

// NewClusterAssoc prepares an immutable local read view. Session absence is nil;
// a cachemap miss must consult its owner (which may run a factory), never silently
// become nil. Prepared/warm reads remain atomic-load/map-only. Explicit factories
// forward their supplied execution scope; no per-row cancellation probe is added.
func NewClusterAssoc(handle ClusterHandle, callableType CallableTypeID, ownerOnMiss bool) Scmer {
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
			value, found := (*s.values.Load())[args[0].String()]
			if found || !ownerOnMiss {
				return value
			}
		}
		ctx := context.Background()
		if ownerOnMiss && len(args) == 3 || !ownerOnMiss && len(args) == 4 {
			ctx = executionContextFrom(args[1])
		} else if !ownerOnMiss && len(args) == 5 {
			ctx = executionContextFrom(args[3])
		}
		value, err := s.handle.Call(ctx, ClusterRequest{Method: "call", Arguments: args})
		if err != nil {
			panic(err)
		}
		// An ordinary key update or cachemap factory/miss publishes a private
		// successor view. Session's scoped 4/5-argument calls are not plain keys.
		if len(args) == 2 || ownerOnMiss && (len(args) == 1 || len(args) == 3) {
			s.publishValue(args[0].String(), value)
		}
		return value
	}
	return NewClusterFunc(call, s, callableType)
}

// Publication belongs to mutations/misses, never a prepared local read.
func (s *clusterSession) publishValue(key string, value Scmer) {
	for {
		old := s.values.Load()
		next := make(map[string]Scmer, len(*old)+1)
		for k, v := range *old {
			next[k] = v
		}
		next[key] = value
		if s.values.CompareAndSwap(old, &next) {
			return
		}
	}
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
