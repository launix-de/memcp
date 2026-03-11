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
package scm

// ActiveHTTPConnections tracks the current number of active HTTP connections.
// Incremented/decremented via ConnState callback — single atomic, no mutex.
var ActiveHTTPConnections int64

// TotalHTTPRequests is atomically incremented on each new HTTP request.
// The background sampler reads this to compute requests/sec without any hot-path mutex.
var TotalHTTPRequests int64
