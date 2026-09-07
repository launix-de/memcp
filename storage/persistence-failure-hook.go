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

import "os"
import "fmt"
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
	hook      string
	class     string
	backend   string
	database  string
	operation string
}

type persistenceFailureCooldownState struct {
	next       time.Time
	suppressed int64
}

type persistenceFailureHookRule struct {
	name     string
	classes  map[string]struct{}
	callback scm.Scmer
	cooldown time.Duration
}

type persistenceFailureHookConfig struct {
	rules []*persistenceFailureHookRule
}

type persistenceFailureHookTask struct {
	rule       *persistenceFailureHookRule
	event      persistenceFailureEvent
	occurredAt time.Time
	suppressed int64
}

// Published configs, rules, and class maps are immutable. The atomic config
// pointer serves error reporters without locking when no hooks exist; mu owns
// cooldown state and config replacement, while running guards callback-induced
// failures across the single dispatcher goroutine.
var persistenceFailureHooks struct {
	config    atomic.Pointer[persistenceFailureHookConfig]
	running   atomic.Bool
	startOnce sync.Once
	mu        sync.Mutex
	states    map[persistenceFailureFingerprint]*persistenceFailureCooldownState
	queue     chan persistenceFailureHookTask
}

func registerPersistenceFailureHook(name string, classes []string, cooldown time.Duration, callback scm.Scmer) {
	if name == "" {
		panic("storage failure hook name must not be empty")
	}
	if len(classes) == 0 {
		panic("storage failure hook must select at least one failure class")
	}
	if cooldown < 0 {
		panic("storage failure hook cooldown must not be negative")
	}
	if !callback.IsProc() && !callback.IsNativeFunc() && !callback.IsJIT() {
		panic("storage failure hook callback must be callable")
	}
	classSet := make(map[string]struct{}, len(classes))
	for _, class := range classes {
		if class == "" {
			panic("storage failure hook class must not be empty")
		}
		classSet[class] = struct{}{}
	}
	rule := &persistenceFailureHookRule{
		name: name, classes: classSet, callback: callback, cooldown: cooldown,
	}
	persistenceFailureHooks.mu.Lock()
	config := persistenceFailureHooks.config.Load()
	rules := make([]*persistenceFailureHookRule, 0, 1)
	if config != nil {
		rules = make([]*persistenceFailureHookRule, 0, len(config.rules)+1)
		for _, existing := range config.rules {
			if existing.name != name {
				rules = append(rules, existing)
			}
		}
	}
	rules = append(rules, rule)
	for fingerprint := range persistenceFailureHooks.states {
		if fingerprint.hook == name {
			delete(persistenceFailureHooks.states, fingerprint)
		}
	}
	if persistenceFailureHooks.queue == nil {
		persistenceFailureHooks.queue = make(chan persistenceFailureHookTask, persistenceFailureHookQueueSize)
	}
	if persistenceFailureHooks.states == nil {
		persistenceFailureHooks.states = make(map[persistenceFailureFingerprint]*persistenceFailureCooldownState)
	}
	persistenceFailureHooks.config.Store(&persistenceFailureHookConfig{rules: rules})
	persistenceFailureHooks.mu.Unlock()
	persistenceFailureHooks.startOnce.Do(func() { go runPersistenceFailureHooks() })
}

func unregisterPersistenceFailureHook(name string) {
	persistenceFailureHooks.mu.Lock()
	config := persistenceFailureHooks.config.Load()
	if config == nil {
		persistenceFailureHooks.mu.Unlock()
		return
	}
	rules := make([]*persistenceFailureHookRule, 0, len(config.rules))
	for _, rule := range config.rules {
		if rule.name != name {
			rules = append(rules, rule)
		}
	}
	for fingerprint := range persistenceFailureHooks.states {
		if fingerprint.hook == name {
			delete(persistenceFailureHooks.states, fingerprint)
		}
	}
	if len(rules) == 0 {
		persistenceFailureHooks.config.Store(nil)
	} else {
		persistenceFailureHooks.config.Store(&persistenceFailureHookConfig{rules: rules})
	}
	persistenceFailureHooks.mu.Unlock()
}

func clearPersistenceFailureHooks() {
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
	persistenceFailureHooks.mu.Lock()
	if persistenceFailureHooks.config.Load() != config {
		persistenceFailureHooks.mu.Unlock()
		return
	}
	for _, rule := range config.rules {
		if !persistenceFailureHookMatchesClass(rule, event.class) {
			continue
		}
		fingerprint := persistenceFailureFingerprint{
			hook: rule.name, class: event.class, backend: event.backend,
			database: event.database, operation: event.operation,
		}
		state := persistenceFailureHooks.states[fingerprint]
		if state == nil {
			state = new(persistenceFailureCooldownState)
			persistenceFailureHooks.states[fingerprint] = state
		}
		if persistenceFailureHooks.running.Load() {
			// Go has no goroutine-local storage with which to distinguish a hook's
			// own storage call from an unrelated concurrent failure. Suppress both
			// while a hook runs, but retain the count for the next notification.
			// This prevents both direct and cross-hook recursion.
			state.suppressed++
			continue
		}
		if occurredAt.Before(state.next) {
			state.suppressed++
			continue
		}
		task := persistenceFailureHookTask{
			rule: rule, event: event, occurredAt: occurredAt,
			suppressed: state.suppressed,
		}
		state.suppressed = 0
		state.next = occurredAt.Add(rule.cooldown)
		select {
		case persistenceFailureHooks.queue <- task:
		default:
			// A blocked outage callback must never block storage error propagation.
			// Fold the dropped notification into the next event for this fingerprint.
			state.suppressed++
		}
	}
	persistenceFailureHooks.mu.Unlock()
}

func persistenceFailureHookMatchesClass(rule *persistenceFailureHookRule, class string) bool {
	if _, matches := rule.classes[class]; matches {
		return true
	}
	_, matchesAll := rule.classes["*"]
	return matchesAll
}

func runPersistenceFailureHooks() {
	for task := range persistenceFailureHooks.queue {
		if !persistenceFailureHookIsCurrent(task.rule) {
			continue
		}
		invokePersistenceFailureHook(task)
	}
}

func persistenceFailureHookIsCurrent(wanted *persistenceFailureHookRule) bool {
	config := persistenceFailureHooks.config.Load()
	if config == nil {
		return false
	}
	for _, rule := range config.rules {
		if rule == wanted {
			return true
		}
	}
	return false
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
	scm.Apply(task.rule.callback, scm.NewSlice([]scm.Scmer{
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
