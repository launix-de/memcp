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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["setTimeout"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
				declaration := declarations["clearTimeout"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [3]BBDescriptor
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d0 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() != 1)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d0.Reg, 1)
						d1 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondNotEqual}
						ctx.BindReg(r0, &d1)
					}
					ctx.FreeDesc(&d0)
					d2 = d1
					ctx.EnsureDesc(&d2)
					if d2.Loc != LocImm && d2.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d2.Loc == LocImm {
						if d2.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d2.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d1)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap3 := d0
						snap4 := d1
						snap5 := d2
						alloc6 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc6)
						d0 = snap3
						d1 = snap4
						d2 = snap5
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["clearTimeout"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					d7 = args[0]
					d7.ID = 0
					ctx.EnsureDesc(&d7)
					d8 = d7
					_ = d8
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl4 := ctx.ReserveLabel()
					_ = lbl4
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					if d8.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int())}
					} else if d8.Type == tagInt && d8.Loc == LocRegPair {
						ctx.FreeReg(d8.Reg)
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg2}
						ctx.BindReg(d8.Reg2, &d9)
						ctx.BindReg(d8.Reg2, &d9)
					} else if d8.Type == tagInt && d8.Loc == LocReg {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
						ctx.BindReg(d8.Reg, &d9)
						ctx.BindReg(d8.Reg, &d9)
					} else {
						d9 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d8}, 1)
						d9.Type = tagInt
						ctx.BindReg(d9.Reg, &d9)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(int64(d9.Imm.Int()))))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegReg(r1, d9.Reg)
						d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d11)
					}
					ctx.FreeDesc(&d9)
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&DefaultScheduler)))), NoHeapPointer: true, Rooted: true}
					if d12.Loc == LocRegPair || d12.Loc == LocStackPair || d12.Loc == LocRegTriple || d12.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d12)
					ctx.SyncDesc(&d11)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr((*Scheduler).Clear), []JITValueDesc{d12, d11}, 1)
					d13.NoHeapPointer = true
					ctx.EmitAndRegImm32(d13.Reg, 1)
					d13.Type = tagBool
					ctx.BindReg(d13.Reg, &d13)
					ctx.FreeDesc(&d11)
					ctx.SyncDesc(&d13)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d13) {
						return d13
					}
					if d13.Loc == LocImm {
						ctx.EmitMakeBool(result, d13)
					} else {
						ctx.EmitMovToReg(result.Reg2, d13)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d14)
						if d13.Loc == LocReg && d13.Reg != result.Reg2 {
							ctx.FreeReg(d13.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
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
