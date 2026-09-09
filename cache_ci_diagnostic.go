// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import "os"
import "fmt"
import "time"
import "runtime/pprof"
import "github.com/launix-de/memcp/storage"

// Temporary CI diagnosis for #875; remove after obtaining the stalled stack.
// It neither cancels requests nor changes eviction or test outcomes.
func init() {
	if os.Getenv("MEMCP_CACHE_DIAGNOSTIC") != "1" {
		return
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			done := make(chan struct{})
			go func() { storage.GlobalCache.Stat(); close(done) }()
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				fmt.Fprintln(os.Stderr, "CI CacheManager stalled: goroutine snapshot")
				pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
				return
			}
		}
	}()
}
