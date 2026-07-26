// Package gls implements goroutine-local storage.
package gls

import (
	"sync/atomic"
)

var mgrRegistry atomic.Pointer[[]*ContextManager]

// Values is simply a map of key types to value types. Used by SetValues to
// set multiple values at once.
type Values map[interface{}]interface{}

type contextFrame struct {
	values Values
	parent *contextFrame
}

type contextSlot struct {
	frame *contextFrame
}

// ContextManager is the main entrypoint for interacting with
// Goroutine-local-storage. You can have multiple independent ContextManagers
// at any given time. ContextManagers are usually declared globally for a given
// class of context variables. You should use NewContextManager for
// construction.
type ContextManager struct {
	values atomic.Pointer[map[uint]*contextSlot]
}

// NewContextManager returns a brand new ContextManager. It also registers the
// new ContextManager in the ContextManager registry which is used by the Go
// method. ContextManagers are typically defined globally at package scope.
func NewContextManager() *ContextManager {
	mgr := &ContextManager{}
	mgr.values.Store(new(map[uint]*contextSlot))
	registerManager(mgr)
	return mgr
}

func registerManager(mgr *ContextManager) {
	for {
		current := mgrRegistry.Load()
		if current == nil {
			empty := new([]*ContextManager)
			if !mgrRegistry.CompareAndSwap(nil, empty) {
				continue
			}
			current = empty
		}
		next := new([]*ContextManager)
		*next = make([]*ContextManager, 0, len(*current)+1)
		*next = append(*next, (*current)...)
		*next = append(*next, mgr)
		if mgrRegistry.CompareAndSwap(current, next) {
			return
		}
	}
}

// Unregister removes a ContextManager from the global registry, used by the
// Go method. Only intended for use when you're completely done with a
// ContextManager. Use of Unregister at all is rare.
func (m *ContextManager) Unregister() {
	for {
		current := mgrRegistry.Load()
		if current == nil {
			return
		}
		idx := -1
		for i, mgr := range *current {
			if mgr == m {
				idx = i
				break
			}
		}
		if idx == -1 {
			return
		}
		next := new([]*ContextManager)
		*next = make([]*ContextManager, 0, len(*current)-1)
		*next = append(*next, (*current)[:idx]...)
		*next = append(*next, (*current)[idx+1:]...)
		if mgrRegistry.CompareAndSwap(current, next) {
			return
		}
	}
}

func (m *ContextManager) getOrCreateSlot(gid uint) *contextSlot {
	for {
		current := m.values.Load()
		if slot := (*current)[gid]; slot != nil {
			return slot
		}

		slot := &contextSlot{}
		next := new(map[uint]*contextSlot)
		*next = make(map[uint]*contextSlot, len(*current)+1)
		for stateGID, stateSlot := range *current {
			(*next)[stateGID] = stateSlot
		}
		(*next)[gid] = slot
		if m.values.CompareAndSwap(current, next) {
			return slot
		}
	}
}

func (m *ContextManager) getSlot(gid uint) *contextSlot {
	return (*m.values.Load())[gid]
}

func (m *ContextManager) pushFrame(gid uint, frame *contextFrame, contextCall func()) {
	slot := m.getOrCreateSlot(gid)
	oldFrame := slot.frame
	frame.parent = oldFrame
	slot.frame = frame
	defer func() {
		slot.frame = oldFrame
	}()
	contextCall()
}

func (m *ContextManager) runWithFrame(gid uint, frame *contextFrame, contextCall func()) {
	if frame == nil {
		contextCall()
		return
	}
	slot := m.getOrCreateSlot(gid)
	oldFrame := slot.frame
	slot.frame = frame
	defer func() {
		slot.frame = oldFrame
	}()
	contextCall()
}

// SetValues takes a collection of values and a function to call for those
// values to be set in. Anything further down the stack will have the set
// values available through GetValue. SetValues will add new values or replace
// existing values of the same key and will not mutate or change values for
// previous stack frames.
func (m *ContextManager) SetValues(newValues Values, contextCall func()) {
	if len(newValues) == 0 {
		contextCall()
		return
	}
	EnsureGoroutineId(func(gid uint) {
		m.pushFrame(gid, &contextFrame{values: newValues}, contextCall)
	})
}

// GetValue will return a previously set value, provided that the value was set
// by SetValues somewhere higher up the stack. If the value is not found, ok
// will be false.
func (m *ContextManager) GetValue(key interface{}) (value interface{}, ok bool) {
	gid, ok := GetGoroutineId()
	if !ok {
		return nil, false
	}
	for frame := m.getFrameForGID(gid); frame != nil; frame = frame.parent {
		value, ok = frame.values[key]
		if ok {
			return value, true
		}
	}
	return nil, false
}

func (m *ContextManager) getFrameForGID(gid uint) *contextFrame {
	slot := m.getSlot(gid)
	if slot == nil {
		return nil
	}
	return slot.frame
}

func (m *ContextManager) getFrame() *contextFrame {
	gid, ok := GetGoroutineId()
	if !ok {
		return nil
	}
	return m.getFrameForGID(gid)
}

// Go preserves ContextManager values and Goroutine-local-storage across new
// goroutine invocations. Child goroutines share immutable context frames with
// their parent instead of copying value maps.
func Go(cb func()) {
	registry := mgrRegistry.Load()
	if registry != nil {
		for _, mgr := range *registry {
			frame := mgr.getFrame()
			if frame != nil {
				cb = func(mgr *ContextManager, frame *contextFrame, cb func()) func() {
					return func() {
						EnsureGoroutineId(func(gid uint) {
							mgr.runWithFrame(gid, frame, cb)
						})
					}
				}(mgr, frame, cb)
			}
		}
	}

	go cb()
}
