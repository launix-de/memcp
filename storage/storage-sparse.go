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
	values map[uint]scm.Scmer
}

func (s *StorageSparse) Size() uint {
	return 8 + 24 * uint(len(s.values)) // heuristic
}

func (s *StorageSparse) String() string {
	return "SCMER-sparse"
}
func (s *StorageSparse) Serialize(f *os.File) {
	defer f.Close()
	binary.Write(f, binary.LittleEndian, uint8(2)) // 2 = StorageSparse
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for k, v := range s.values {
		binary.Write(f, binary.LittleEndian, uint64(k))
		vbytes, err := json.Marshal(v)
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
	s.values = make(map[uint]scm.Scmer)
	scanner := bufio.NewScanner(f)
	for {
		var k uint64
		err := binary.Read(f, binary.LittleEndian, &k)
		if err != nil || !scanner.Scan() {
			break
		}
		var v scm.Scmer
		json.Unmarshal(scanner.Bytes(), &v)
		s.values[uint(k)] = v
	}
	return uint(l)
}

func (s *StorageSparse) GetValue(i uint) scm.Scmer {
	return s.values[i]
}

func (s *StorageSparse) scan(i uint, value scm.Scmer) {
}
func (s *StorageSparse) prepare() {
}
func (s *StorageSparse) init(i uint) {
	s.values = make(map[uint]scm.Scmer)
}
func (s *StorageSparse) build(i uint, value scm.Scmer) {
	// store
	if value != nil {
		s.values[i] = value
	}
}
func (s *StorageSparse) finish() {
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint) ColumnStorage {
	return nil
}
