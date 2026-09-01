/*
Copyright (C) 2026  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import (
	"container/heap"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)
import "unsafe"

type Task func()

type task struct {
	runAt time.Time
	fn    Task
	id    uint64
}

type taskHeap []task

func (h taskHeap) Len() int { return len(h) }

func (h taskHeap) Less(i, j int) bool {
	if h[i].runAt.Equal(h[j].runAt) {
		return h[i].id < h[j].id
	}
	return h[i].runAt.Before(h[j].runAt)
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *taskHeap) Push(x any) {
	*h = append(*h, x.(task))
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type Scheduler struct {
	mu       sync.Mutex
	tasks    taskHeap
	wakeCh   chan struct{}
	stopCh   chan struct{}
	cancel   map[uint64]struct{}
	active   map[uint64]struct{}
	stopped  bool
	nextID   uint64
	initOnce sync.Once
	wg       sync.WaitGroup
	taskWg   sync.WaitGroup // tracks in-flight task goroutines
}

var DefaultScheduler Scheduler

func init() {
	DefaultScheduler.init()
}

func (s *Scheduler) init() {
	s.initOnce.Do(func() {
		s.wakeCh = make(chan struct{}, 1)
		s.stopCh = make(chan struct{})
		s.cancel = make(map[uint64]struct{})
		s.active = make(map[uint64]struct{})
		heap.Init(&s.tasks)
		s.wg.Add(1)
		go s.run()
	})
}

func (s *Scheduler) ScheduleAt(t time.Time, fn Task) (uint64, bool) {
	if fn == nil {
		return 0, false
	}
	s.init()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return 0, false
	}
	s.nextID++
	id := s.nextID
	newTask := task{runAt: t, fn: fn, id: id}
	heap.Push(&s.tasks, newTask)
	s.active[id] = struct{}{}
	delete(s.cancel, id)
	shouldWake := len(s.tasks) > 0 && s.tasks[0].id == id
	if shouldWake {
		s.signalLocked()
	}
	return id, true
}

func (s *Scheduler) ScheduleAfter(delay time.Duration, fn Task) (uint64, bool) {
	if delay < 0 {
		delay = 0
	}
	return s.ScheduleAt(time.Now().Add(delay), fn)
}

func (s *Scheduler) Clear(id uint64) bool {
	s.init()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return false
	}
	if _, ok := s.active[id]; !ok {
		return false
	}
	s.cancel[id] = struct{}{}
	delete(s.active, id)
	s.signalLocked()
	return true
}

func (s *Scheduler) Stop() {
	s.init()
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		s.wg.Wait()
		s.taskWg.Wait()
		return
	}
	s.stopped = true
	close(s.stopCh)
	s.mu.Unlock()
	s.signal()
	s.wg.Wait()
	s.taskWg.Wait()
}

func (s *Scheduler) signalLocked() {
	select {
	case s.wakeCh <- struct{}{}:
	default:
	}
}

func (s *Scheduler) signal() {
	signalC := s.wakeCh
	if signalC == nil {
		return
	}
	select {
	case signalC <- struct{}{}:
	default:
	}
}

func (s *Scheduler) runTask(fn Task) {
	defer s.taskWg.Done()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("scheduler: task panic: %v\n", r)
			debug.PrintStack()
		}
	}()
	fn()
}

func (s *Scheduler) drainTimer(timer *time.Timer) {
	if timer != nil && !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func (s *Scheduler) run() {
	defer s.wg.Done()
	var timer *time.Timer
	for {
		s.mu.Lock()
		if len(s.tasks) == 0 {
			if s.stopped {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
			select {
			case <-s.stopCh:
				return
			case <-s.wakeCh:
			}
			continue
		}
		next := s.tasks[0]
		if _, cancelled := s.cancel[next.id]; cancelled {
			heap.Pop(&s.tasks)
			delete(s.cancel, next.id)
			delete(s.active, next.id)
			s.mu.Unlock()
			continue
		}
		wait := time.Until(next.runAt)
		if wait <= 0 {
			heap.Pop(&s.tasks)
			delete(s.active, next.id)
			delete(s.cancel, next.id)
			s.taskWg.Add(1)
			s.mu.Unlock()
			go s.runTask(next.fn)
			continue
		}
		if timer == nil {
			timer = time.NewTimer(wait)
		} else {
			timer.Reset(wait)
		}
		s.mu.Unlock()
		select {
		case <-timer.C:
		case <-s.wakeCh:
			s.drainTimer(timer)
		case <-s.stopCh:
			s.drainTimer(timer)
			return
		}
	}
}

func init_scheduler() {
	Declare(&Globalenv, &Declaration{
		Name: "setTimeout",

		Fn: setTimeout,
		Type: &TypeDescriptor{Kind: "func", Description: "Schedules a callback to run after the given delay in milliseconds (fractional values allowed for sub-millisecond precision).",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "func", Label: "callback", Description: "function to execute once the timeout expires", Params: []*TypeDescriptor{{Kind: "any", Label: "args", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}, &TypeDescriptor{Kind: "number", Label: "milliseconds", Description: "milliseconds until execution"}, &TypeDescriptor{Kind: "any", Label: "args...", Description: "optional arguments forwarded to the callback", Variadic: true}},
			Return: &TypeDescriptor{Kind: "int"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["setTimeout"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "clearTimeout",

		Fn: clearTimeout,
		Type: &TypeDescriptor{Kind: "func", Description: "Cancels a timeout created with setTimeout.",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", Label: "id", Description: "identifier returned by setTimeout"}},
			Return: &TypeDescriptor{Kind: "bool"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["clearTimeout"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
					var d1 JITValueDesc
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() != 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 1)
						ctx.EmitSetcc(r0, CondNotEqual)
						d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							if ps.General {
							}
							ps3 := PhiState{General: ps.General}
							ps3.OverlayValues = make([]JITValueDesc, 3)
							ps3.OverlayValues[0] = d0
							ps3.OverlayValues[1] = d1
							ps3.OverlayValues[2] = d2
							return bbs[1].RenderPS(ps3)
						}
						if ps.General {
						}
						ps4 := PhiState{General: ps.General}
						ps4.OverlayValues = make([]JITValueDesc, 3)
						ps4.OverlayValues[0] = d0
						ps4.OverlayValues[1] = d1
						ps4.OverlayValues[2] = d2
						return bbs[2].RenderPS(ps4)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps5 := PhiState{General: true}
					ps5.OverlayValues = make([]JITValueDesc, 3)
					ps5.OverlayValues[0] = d0
					ps5.OverlayValues[1] = d1
					ps5.OverlayValues[2] = d2
					ps6 := PhiState{General: true}
					ps6.OverlayValues = make([]JITValueDesc, 3)
					ps6.OverlayValues[0] = d0
					ps6.OverlayValues[1] = d1
					ps6.OverlayValues[2] = d2
					snap7 := d0
					snap8 := d1
					snap9 := d2
					alloc10 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps6)
					}
					ctx.RestoreAllocState(alloc10)
					d0 = snap7
					d1 = snap8
					d2 = snap9
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps5)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["clearTimeout"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d11 = args[0]
					d11.ID = 0
					ctx.EnsureDesc(&d11)
					d12 = d11
					_ = d12
					ctx.StabilizeDescForControlFlow(&d12)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d13 JITValueDesc
					if d12.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int())}
					} else if d12.Type == tagInt && d12.Loc == LocRegPair {
						ctx.FreeReg(d12.Reg)
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2}
						ctx.BindReg(d12.Reg2, &d13)
						ctx.BindReg(d12.Reg2, &d13)
					} else if d12.Type == tagInt && d12.Loc == LocReg {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
						ctx.BindReg(d12.Reg, &d13)
						ctx.BindReg(d12.Reg, &d13)
					} else {
						d13 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d12}, 1)
						d13.Type = tagInt
						ctx.BindReg(d13.Reg, &d13)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.FreeDesc(&d11)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					var d15 JITValueDesc
					if d13.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(int64(d13.Imm.Int()))))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegReg(r1, d13.Reg)
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d15)
					}
					ctx.FreeDesc(&d13)
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&DefaultScheduler)))), NoHeapPointer: true, Rooted: true}
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d15)
					d17 = ctx.EmitGoCallScalar(GoFuncAddr((*Scheduler).Clear), []JITValueDesc{d16, d15}, 1)
					ctx.EmitAndRegImm32(d17.Reg, 1)
					d17.Type = tagBool
					ctx.BindReg(d17.Reg, &d17)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						ctx.EmitMakeBool(result, d17)
					} else {
						ctx.EmitMovToReg(result.Reg2, d17)
						d18 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d18)
						if d17.Loc == LocReg && d17.Reg != result.Reg2 {
							ctx.FreeReg(d17.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps19 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps19)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  15,
		},
	})
}

func setTimeout(a ...Scmer) Scmer {
	if len(a) < 2 {
		panic("setTimeout expects at least a callback and delay")
	}

	callback := a[0]
	millis := ToFloat(a[1])
	if millis < 0 {
		millis = 0
	}

	duration := time.Duration(millis * float64(time.Millisecond))
	callbackArgs := append([]Scmer(nil), a[2:]...)
	id, ok := DefaultScheduler.ScheduleAfter(duration, func() {
		Apply(callback, callbackArgs...)
	})
	if !ok {
		return NewBool(false)
	}
	return NewInt(int64(id))
}

func clearTimeout(a ...Scmer) Scmer {
	if len(a) != 1 {
		panic("clearTimeout expects one argument")
	}
	id := uint64(ToInt(a[0]))
	return NewBool(DefaultScheduler.Clear(id))
}
