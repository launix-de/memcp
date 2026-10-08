/*
Copyright (C) 2025-2026  Carl-Philip Hänsch

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

import (
	"context"
	"runtime"
)

// global semaphore to limit concurrent disk-backed load operations
var loadSemaphore chan struct{}

func init() {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	loadSemaphore = make(chan struct{}, workers)
	// prefill with tokens
	for i := 0; i < workers; i++ {
		loadSemaphore <- struct{}{}
	}
}

// acquireLoadSlot waits for a disk-load slot or query cancellation. A nil
// context is used for recovery and maintenance, which must complete normally.
func acquireLoadSlot(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-loadSemaphore:
		// If both signals are ready, return the token and prefer cancellation
		// before starting a disk load.
		if err := ctx.Err(); err != nil {
			loadSemaphore <- struct{}{}
			return nil, err
		}
		return func() { loadSemaphore <- struct{}{} }, nil
	}
}
