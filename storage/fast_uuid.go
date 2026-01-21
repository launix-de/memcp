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

import (
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var uuidCounter uint64 = uint64(time.Now().UnixNano())

// newUUID returns a UUIDv4-like value without relying on crypto/rand.
// It is not suitable for cryptographic use but avoids startup stalls on low-entropy systems.
func newUUID() uuid.UUID {
	ctr := atomic.AddUint64(&uuidCounter, 1)
	now := uint64(time.Now().UnixNano())
	var b [16]byte
	binary.LittleEndian.PutUint64(b[0:8], ctr)
	binary.LittleEndian.PutUint64(b[8:16], ctr^now^(now<<17))
	// RFC4122 variant + version 4
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return uuid.UUID(b)
}
