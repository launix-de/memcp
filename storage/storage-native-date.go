/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "io"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// Exact scalar magic 3, version 2/encoding 2 stores native dates as three
// ordinary bit-packed integer columns: Unix seconds, nanoseconds and zone.
// Each component carries its complete magic-10/versioned storage frame and
// count. All three NULL masks must agree. No component reads past its frame.
// Mixed native values use the permanent private-JSON payload (encoding 1).
func (s *storageSCMERExact) serializeNativeDates(f io.Writer) bool {
	hasDate := false
	for _, value := range s.values {
		if value.IsDate() {
			hasDate = true
		} else if !value.IsNil() {
			return false
		}
	}
	if !hasDate {
		return false
	}
	if uint64(len(s.values)) > uint64(^uint32(0)) {
		panic("native date column count exceeds row identifier domain")
	}
	var components [3]StorageInt
	for i := range components {
		components[i].prepare()
	}
	parts := func(value scm.Scmer) [3]scm.Scmer {
		if value.IsNil() {
			return [3]scm.Scmer{}
		}
		return [3]scm.Scmer{scm.NewInt(value.Int()), scm.NewInt(int64(value.DateNanosecond())), scm.NewInt(int64(value.DateZone()))}
	}
	for row, value := range s.values {
		for part, field := range parts(value) {
			components[part].scan(uint32(row), field)
		}
	}
	for i := range components {
		components[i].init(uint32(len(s.values)))
	}
	for row, value := range s.values {
		for part, field := range parts(value) {
			components[part].build(uint32(row), field)
		}
	}
	if err := binary.Write(f, binary.LittleEndian, [3]byte{3, storageExactScalarVersion, 2}); err != nil {
		panic(err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint64(len(s.values))); err != nil {
		panic(err)
	}
	for i := range components {
		components[i].Serialize(f)
	}
	return true
}

func (s *storageSCMERExact) deserializeNativeDates(f io.Reader) uint {
	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		panic(err)
	}
	if count > uint64(^uint32(0)) {
		panic("native date column count exceeds row identifier domain")
	}
	var components [3]StorageInt
	for part := range components {
		var magic [1]byte
		readStorageBytes(f, magic[:])
		if magic[0] != 10 {
			panic("native date component must use integer storage")
		}
		if rows := components[part].Deserialize(f); uint64(rows) != count {
			panic("native date component row counts disagree")
		}
	}
	// All complete component frames and their common row count are validated.
	// Reserve the result once instead of retaining successive growing arrays
	// during cold query and index construction.
	s.values = make([]scm.Scmer, count)
	for row := uint32(0); uint64(row) < count; row++ {
		seconds := components[0].GetValue(row)
		nanoseconds := components[1].GetValue(row)
		zone := components[2].GetValue(row)
		if seconds.IsNil() || nanoseconds.IsNil() || zone.IsNil() {
			if !seconds.IsNil() || !nanoseconds.IsNil() || !zone.IsNil() {
				panic("native date component NULL masks disagree")
			}
			s.values[row] = scm.NewNil()
		} else {
			s.values[row] = scm.NewDateNanoWithZone(seconds.Int(), int(nanoseconds.Int()), int(zone.Int()))
		}
	}
	return uint(count)
}
