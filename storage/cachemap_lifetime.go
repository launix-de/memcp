/*
Copyright (C) 2026 MemCP Contributors

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import (
	"context"
	"sync"
)

// Cache factories can outlive the request which first waited for them. Their
// process lifetime therefore owns admission and completion independently of
// each cache's readers and entries. Context children retain only live factories.
type cacheMapFactoryLifetime struct {
	mu        sync.Mutex
	closed    bool
	context   context.Context
	cancel    context.CancelFunc
	producers sync.WaitGroup
}

func newCacheMapFactoryLifetime() *cacheMapFactoryLifetime {
	ctx, cancel := context.WithCancel(context.Background())
	return &cacheMapFactoryLifetime{context: ctx, cancel: cancel}
}

var processCacheMapFactories = newCacheMapFactoryLifetime()

func (owner *cacheMapFactoryLifetime) admit() (context.Context, context.CancelFunc, bool) {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closed {
		return nil, nil, false
	}
	ctx, cancel := context.WithCancel(owner.context)
	// Shutdown closes admission under the same mutex before it can wait.
	owner.producers.Add(1)
	return ctx, cancel, true
}

func (owner *cacheMapFactoryLifetime) shutdown() {
	owner.mu.Lock()
	owner.closed = true
	owner.mu.Unlock()
	owner.cancel()
	// Cancellation is not completion: a factory may still be compiling or
	// unwinding. Every shutdown caller joins the same admitted producers.
	owner.producers.Wait()
}

// ShutdownCacheMapFactories is terminal. Stop admitting factories and join
// their publication/unwind before storage resources or executable code retire.
func ShutdownCacheMapFactories() {
	processCacheMapFactories.shutdown()
}
