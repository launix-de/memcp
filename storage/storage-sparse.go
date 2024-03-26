/*
Copyright (C) 2023  Carl-Philip Hänsch

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
import "bufio"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type StorageSparse struct {
	i, count uint64
	recids StorageInt
	values []scm.Scmer // TODO: embed other formats as values (ColumnStorage with a proposeCompression loop)
}

func (s *StorageSparse) Size() uint {
	return 8 + 24 + s.recids.Size() + 16 * uint(len(s.values)) // heuristic
}

func (s *StorageSparse) String() string {
	return "SCMER-sparse"
}
func (s *StorageSparse) Serialize(f *os.File) {
	defer f.Close()
	binary.Write(f, binary.LittleEndian, uint8(2)) // 2 = StorageSparse
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for k, v := range s.values {
		vbytes, err := json.Marshal(uint64(s.recids.GetValueUInt(uint(k)) + uint64(s.recids.offset)))
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file beginning at byte 9
		vbytes, err = json.Marshal(v)
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file beginning at byte 9
	}
}
func (s *StorageSparse) Deserialize(f *os.File) uint {
	defer f.Close()
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.count = l
	var l2 uint64
	binary.Read(f, binary.LittleEndian, &l2)
	s.values = make([]scm.Scmer, l2)
	s.i = l2
	scanner := bufio.NewScanner(f)
	s.recids.prepare()
	s.recids.scan(0, 0)
	s.recids.scan(uint(l2-1), l-1)
	s.recids.init(uint(l2))
	i := 0
	for {
		var k uint64
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &k)
		if !scanner.Scan() {
			break
		}
		var v scm.Scmer
		json.Unmarshal(scanner.Bytes(), &v)
		s.recids.build(uint(i), k)
		s.values[i] = v
		i++
	}
	s.recids.finish()
	return uint(l)
}

func (s *StorageSparse) GetValue(i uint) scm.Scmer {
	var lower uint = 0
	var upper uint = uint(s.i)
	for {
		if lower == upper {
			return nil // sparse value
		}
		pivot := uint((lower + upper) / 2)
		recid := s.recids.GetValueUInt(pivot) + uint64(s.recids.offset)
		if recid == uint64(i) {
			return s.values[pivot] // found the value
		}
		if recid < uint64(i) {
			lower = pivot + 1
		} else {
			upper = pivot
		}

	}
}

func (s *StorageSparse) scan(i uint, value scm.Scmer) {
	if value != nil {
		s.recids.scan(uint(s.i), i)
		s.i++
	}
}
func (s *StorageSparse) prepare() {
	s.i = 0
}
func (s *StorageSparse) init(i uint) {
	s.values = make([]scm.Scmer, s.i)
	s.count = uint64(i)
	s.recids.init(uint(s.i))
	s.i = 0
}
func (s *StorageSparse) build(i uint, value scm.Scmer) {
	// store
	if value != nil {
		s.recids.build(uint(s.i), i)
		s.values[s.i] = value
		s.i++
	}
}
func (s *StorageSparse) finish() {
	s.recids.finish()
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint) ColumnStorage {
	return nil
}
