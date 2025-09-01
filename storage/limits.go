/*
Copyright (C) 2025  Carl-Philip Hänsch

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

import "runtime"

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

// acquireLoadSlot blocks until a load slot is available and returns a release func.
func acquireLoadSlot() func() {
	<-loadSemaphore
	return func() { loadSemaphore <- struct{}{} }
}

