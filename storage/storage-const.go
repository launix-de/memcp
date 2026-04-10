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
	"encoding/json"
	"fmt"
	"github.com/launix-de/memcp/scm"
	"io"
)

// StorageConst stores a column where every row has the same value.
// Zero per-element overhead: only the single constant value is stored.
type StorageConst struct {
	value scm.Scmer
	count uint64
}

func (s *StorageConst) String() string {
	return fmt.Sprintf("const[%s]", s.value.String())
}

func (s *StorageConst) ComputeSize() uint {
	return 48 + scm.ComputeSize(s.value)
}

func (s *StorageConst) GetValue(i uint32) scm.Scmer {
	return s.value
}

func (s *StorageConst) GetCachedReader() ColumnReader { return s }

func (s *StorageConst) prepare()                                  {}
func (s *StorageConst) scan(i uint32, value scm.Scmer)            {}
func (s *StorageConst) proposeCompression(i uint32) ColumnStorage { return nil }
func (s *StorageConst) init(i uint32)                             { s.count = uint64(i) }
func (s *StorageConst) build(i uint32, value scm.Scmer)           { s.value = value } // all rows identical; last assignment wins
func (s *StorageConst) finish()                                   {}

// Serialize: magic 41 + uint64 count + JSON-encoded value
func (s *StorageConst) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(41))
	binary.Write(f, binary.LittleEndian, s.count)
	b, _ := json.Marshal(s.value)
	binary.Write(f, binary.LittleEndian, uint32(len(b)))
	f.Write(b)
}

func (s *StorageConst) Deserialize(f io.Reader) uint {
	binary.Read(f, binary.LittleEndian, &s.count)
	var vlen uint32
	binary.Read(f, binary.LittleEndian, &vlen)
	buf := make([]byte, vlen)
	io.ReadFull(f, buf)
	var v any
	json.Unmarshal(buf, &v)
	s.value = scm.TransformFromJSON(v)
	return uint(s.count)
}

func (s *StorageConst) DistinctCount() uint { return 1 }
