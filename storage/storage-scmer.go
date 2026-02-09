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

import "io"
import "math"
import "bufio"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// main type for storage: can store any value, is inefficient but does type analysis how to optimize
type StorageSCMER struct {
	values       []scm.Scmer
	onlyInt      bool
	onlyFloat    bool
	hasString    bool
	longStrings  int
	null         uint  // amount of NULL values (sparse map!)
	numSeq       uint  // sequence statistics
	last1, last2 int64 // sequence statistics
}

func (s *StorageSCMER) ComputeSize() uint {
	// ! size of Scmer values is not considered
	var sz uint = 80 + 24
	for _, v := range s.values {
		sz += scm.ComputeSize(v)
	}
	return sz
}

func (s *StorageSCMER) String() string {
	return "SCMER"
}

func (s *StorageSCMER) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(1)) // 1 = StorageSCMER
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for i := 0; i < len(s.values); i++ {
		v, err := json.Marshal(s.values[i])
		if err != nil {
			panic(err)
		}
		f.Write(v)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file beginning at byte 9
	}
}
func (s *StorageSCMER) Deserialize(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values = make([]scm.Scmer, l)
	scanner := bufio.NewScanner(f)
	for i := uint64(0); i < l; i++ {
		if scanner.Scan() {
			var v any
			json.Unmarshal(scanner.Bytes(), &v)
			s.values[i] = scm.TransformFromJSON(v)
		}
	}
	return uint(l)
}

func (s *StorageSCMER) GetValue(i uint) scm.Scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint, value scm.Scmer) {
	if value.IsNil() {
		s.null++
		return
	}
	if value.GetTag() == scm.TagDate {
		v2 := value.Int()
		if v2-s.last1 == s.last1-s.last2 {
			s.numSeq++
		}
		s.last2 = s.last1
		s.last1 = v2
		return
	}
	if value.IsInt() {
		v2 := value.Int()
		if v2-s.last1 == s.last1-s.last2 {
			s.numSeq++
		}
		s.last2 = s.last1
		s.last1 = v2
		return
	}
	if value.IsFloat() {
		f := value.Float()
		if _, frac := math.Modf(f); frac != 0.0 {
			s.onlyInt = false
		} else {
			v := int64(f)
			if v-s.last1 == s.last1-s.last2 {
				s.numSeq++
			}
			s.last2 = s.last1
			s.last1 = v
		}
		return
	}
	s.onlyInt = false
	if value.IsString() {
		s.onlyFloat = false
		s.hasString = true
		if len(value.String()) > 255 {
			s.longStrings++
		}
	} else {
		s.onlyFloat = false
	}
}
func (s *StorageSCMER) prepare() {
	s.onlyInt = true
	s.onlyFloat = true
	s.hasString = false
}
func (s *StorageSCMER) init(i uint) {
	// allocate
	s.values = make([]scm.Scmer, i)
}
func (s *StorageSCMER) build(i uint, value scm.Scmer) {
	// store
	s.values[i] = value
}
func (s *StorageSCMER) finish() {
}

// soley to StorageSCMER
func (s *StorageSCMER) proposeCompression(i uint) ColumnStorage {
	if s.null*100 > i*13 {
		// sparse payoff against bitcompressed is at ~13%
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageSparse)
			return b
		}
		return new(StorageSparse)
	}
	if s.hasString {
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageString)
			return b
		}
		return new(StorageString)
	}
	if s.onlyInt { // TODO: OverlaySCMER?
		// propose sequence compression in the form (recordid, startvalue, length, stride) using binary search on recordid for reading
		if i > 5 && 2*(i-s.numSeq) < i {
			return new(StorageSeq)
		}
		return new(StorageInt)
	}
	if s.onlyFloat {
		// tight float packing
		return new(StorageFloat)
	}
	if s.null*2 > i {
		// sparse payoff against StorageSCMER is at 2.1
		return new(StorageSparse)
	}
	// dont't propose another pass
	return nil
}
