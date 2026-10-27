/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package scm

import "sync"
import "unsafe"

// ProcedureBindings owns immutable session read snapshots for one statement's
// dispatched procedures. It never enters a stored plan. Snapshot construction
// and its mutex are outside row loops; the wire contains only demanded values.
// Mutating/dynamic session operations remain explicit runtime references, not
// automatically executable RPCs disguised as ordinary row-bound reads.
type ProcedureBindings struct {
	mu       sync.Mutex
	sessions map[Scmer]map[string]Scmer
}

func (bindings *ProcedureBindings) read(value Scmer, key string) (Scmer, bool) {
	if value.GetTag() != tagFunc || auxVal(value.aux) != uint64(sessionCallableID) {
		return NewNil(), false
	}
	bindings.mu.Lock()
	defer bindings.mu.Unlock()
	if bindings.sessions == nil {
		bindings.sessions = make(map[Scmer]map[string]Scmer)
	}
	snapshot, ok := bindings.sessions[value]
	if !ok {
		state := (*sessionCallable)(unsafe.Pointer(value.ptr)).state
		state.Mu.RLock()
		snapshot = make(map[string]Scmer, len(state.Map))
		for key, item := range state.Map {
			snapshot[key] = item
		}
		state.Mu.RUnlock()
		bindings.sessions[value] = snapshot
	}
	return snapshot[key], true
}
func (bindings *ProcedureBindings) freeze(value Scmer, env *Env, bound map[Symbol]struct{}) Scmer {
	value = value.WithoutSourceInfo()
	if value.IsProc() {
		proc := value.Proc().snapshot()
		proc.Body = bindings.freeze(proc.Body, proc.En, procedureWireBindings(proc.Params, proc.Body, nil))
		proc.JITCode = 0
		proc.Compiled = nil
		return NewProcStruct(proc)
	}
	if !value.IsSlice() {
		return value
	}
	items := value.Slice()
	if len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return value
	}
	if len(items) == 2 && items[1].IsString() {
		head := items[0]
		if head.IsSymbol() && env != nil {
			if _, local := bound[head.Symbol()]; !local {
				if owner := env.FindRead(head.Symbol()); owner != nil {
					head = owner.Vars[head.Symbol()]
				}
			}
		}
		if result, ok := bindings.read(head, items[1].String()); ok {
			return closedCaptureLiteral(result)
		}
	}
	if scmerIsSymbol(items[0], "lambda") && len(items) >= 3 {
		bound = procedureWireBindings(items[1], items[2], bound)
	}
	out := make([]Scmer, len(items))
	for i, item := range items {
		out[i] = bindings.freeze(item, env, bound)
	}
	return NewSlice(out)
}
