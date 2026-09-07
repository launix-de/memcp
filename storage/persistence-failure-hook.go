/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "fmt"
import "os"
import "sync"
import "time"
import "sync/atomic"

import "github.com/launix-de/memcp/scm"

const persistenceFailureHookQueueSize = 64

type persistenceFailureEvent struct {
	class          string
	backend        string
	database       string
	operation      string
	err            error
	outcomeUnknown bool
}

type persistenceFailureFingerprint struct {
	class     string
	backend   string
	database  string
	operation string
}

type persistenceFailureCooldownState struct {
	next       time.Time
	suppressed int64
}

type persistenceFailureHookConfig struct {
	callback scm.Scmer
	cooldown time.Duration
}

type persistenceFailureHookTask struct {
	config     *persistenceFailureHookConfig
	event      persistenceFailureEvent
	occurredAt time.Time
	suppressed int64
}

var persistenceFailureHooks struct {
	config    atomic.Pointer[persistenceFailureHookConfig]
	running   atomic.Bool
	startOnce sync.Once
	mu        sync.Mutex
	states    map[persistenceFailureFingerprint]*persistenceFailureCooldownState
	queue     chan persistenceFailureHookTask
}

func registerPersistenceFailureHook(cooldown time.Duration, callback scm.Scmer) {
	if cooldown < 0 {
		panic("storage failure hook cooldown must not be negative")
	}
	config := &persistenceFailureHookConfig{callback: callback, cooldown: cooldown}
	persistenceFailureHooks.mu.Lock()
	persistenceFailureHooks.states = make(map[persistenceFailureFingerprint]*persistenceFailureCooldownState)
	if persistenceFailureHooks.queue == nil {
		persistenceFailureHooks.queue = make(chan persistenceFailureHookTask, persistenceFailureHookQueueSize)
	}
	persistenceFailureHooks.config.Store(config)
	persistenceFailureHooks.mu.Unlock()
	persistenceFailureHooks.startOnce.Do(func() { go runPersistenceFailureHooks() })
}

func clearPersistenceFailureHook() {
	persistenceFailureHooks.config.Store(nil)
	persistenceFailureHooks.mu.Lock()
	persistenceFailureHooks.states = nil
	persistenceFailureHooks.mu.Unlock()
}

func notifyPersistenceFailure(event persistenceFailureEvent) {
	notifyPersistenceFailureAt(event, time.Now())
}

func notifyPersistenceFailureAt(event persistenceFailureEvent, occurredAt time.Time) {
	config := persistenceFailureHooks.config.Load()
	if config == nil {
		return
	}
	fingerprint := persistenceFailureFingerprint{
		class: event.class, backend: event.backend,
		database: event.database, operation: event.operation,
	}
	persistenceFailureHooks.mu.Lock()
	if persistenceFailureHooks.config.Load() != config {
		persistenceFailureHooks.mu.Unlock()
		return
	}
	state := persistenceFailureHooks.states[fingerprint]
	if state == nil {
		state = new(persistenceFailureCooldownState)
		persistenceFailureHooks.states[fingerprint] = state
	}
	if persistenceFailureHooks.running.Load() {
		// Go has no goroutine-local storage with which to distinguish a hook's
		// own storage call from an unrelated concurrent failure. Suppress both
		// while the hook runs, but retain the count for the next notification.
		// This prevents recursion without silently losing concurrent failures.
		state.suppressed++
		persistenceFailureHooks.mu.Unlock()
		return
	}
	if occurredAt.Before(state.next) {
		state.suppressed++
		persistenceFailureHooks.mu.Unlock()
		return
	}
	task := persistenceFailureHookTask{
		config: config, event: event, occurredAt: occurredAt,
		suppressed: state.suppressed,
	}
	state.suppressed = 0
	state.next = occurredAt.Add(config.cooldown)
	select {
	case persistenceFailureHooks.queue <- task:
	default:
		// A blocked outage callback must never block storage error propagation.
		// Fold the dropped notification into the next event for this fingerprint.
		state.suppressed++
	}
	persistenceFailureHooks.mu.Unlock()
}

func runPersistenceFailureHooks() {
	for task := range persistenceFailureHooks.queue {
		if persistenceFailureHooks.config.Load() != task.config {
			continue
		}
		invokePersistenceFailureHook(task)
	}
}

func invokePersistenceFailureHook(task persistenceFailureHookTask) {
	persistenceFailureHooks.running.Store(true)
	defer persistenceFailureHooks.running.Store(false)
	defer func() {
		if recovered := recover(); recovered != nil {
			// Never use MemCP-backed logging here. The hook may have failed because
			// its own database operation reached the same unavailable storage.
			_, _ = fmt.Fprintf(os.Stderr, "CRITICAL: storage failure hook panicked: %v\n", recovered)
		}
	}()
	errText := ""
	if task.event.err != nil {
		errText = task.event.err.Error()
	}
	scm.Apply(task.config.callback, scm.NewSlice([]scm.Scmer{
		scm.NewString("backend"), scm.NewString(task.event.backend),
		scm.NewString("database"), scm.NewString(task.event.database),
		scm.NewString("operation"), scm.NewString(task.event.operation),
		scm.NewString("error"), scm.NewString(errText),
		scm.NewString("class"), scm.NewString(task.event.class),
		scm.NewString("outcome_unknown"), scm.NewBool(task.event.outcomeUnknown),
		scm.NewString("suppressed_count"), scm.NewInt(task.suppressed),
		scm.NewString("timestamp"), scm.NewFloat(float64(task.occurredAt.UnixNano()) / 1e9),
	}))
}
