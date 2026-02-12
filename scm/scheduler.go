package scm

import (
	"container/heap"
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

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
		return
	}
	s.stopped = true
	close(s.stopCh)
	s.mu.Unlock()
	s.signal()
	s.wg.Wait()
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
	DeclareTitle("Scheduler")
	Declare(&Globalenv, &Declaration{
		"setTimeout", "Schedules a callback to run after the given delay in milliseconds (fractional values allowed for sub-millisecond precision).",
		2, 1000,
		[]DeclarationParameter{
			{"callback", "func", "function to execute once the timeout expires", nil},
			{"milliseconds", "number", "milliseconds until execution", nil},
			{"args...", "any", "optional arguments forwarded to the callback", nil},
		}, "int",
		setTimeout, false, false, nil,
	})
	Declare(&Globalenv, &Declaration{
		"clearTimeout", "Cancels a timeout created with setTimeout.",
		1, 1,
		[]DeclarationParameter{
			{"id", "number", "identifier returned by setTimeout", nil},
		}, "bool",
		clearTimeout, false, false, nil,
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
		NewContext(context.TODO(), func() {
			Apply(callback, callbackArgs...)
		})
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
