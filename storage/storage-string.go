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

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"

type StorageString struct {
	// StorageInt for dictionary entries
	values StorageInt
	// the dictionary: bitcompress all start+end markers; use one big string for all values that is sliced of from
	dictionary string
	starts StorageInt
	lens StorageInt
	// helpers
	sb strings.Builder
	reverseMap map[string][3]uint
	count uint
	allsize int
	nodict bool // disable values array
}

func (s *StorageString) String() string {
	if s.nodict {
		return fmt.Sprintf("string-buffer[%d]", len(s.dictionary))
	}
	return fmt.Sprintf("string-dict[%d]", s.count)
}

func (s *StorageString) getValue(i uint) scm.Scmer {
	if s.nodict {
		start := s.starts.getValueUInt(i)
		if s.starts.hasNull && start == s.starts.null {
			return nil
		}
		len_ := s.lens.getValueUInt(i)
		return s.dictionary[start:start+len_]
	} else {
		idx := uint(s.values.getValueUInt(i))
		if s.values.hasNull && idx == uint(s.values.null) {
			return nil
		}
		start := s.starts.getValueUInt(idx)
		len_ := s.lens.getValueUInt(idx)
		return s.dictionary[start:start+len_]
	}
}

func (s *StorageString) prepare() {
	// set up scan
	s.starts.prepare()
	s.lens.prepare()
	s.values.prepare()
	s.reverseMap = make(map[string][3]uint)
}
func (s *StorageString) scan(i uint, value scm.Scmer) {
	// storage is so simple, dont need scan
	var v string
	switch v_ := value.(type) {
		case string:
			v = v_
		default:
			// NULL
			if !s.nodict {
				s.values.scan(i, nil)
			}
			return
	}
	if i == 100 && len(s.reverseMap) > 99 {
		// nearly no repetition in the first 100 items: save the time to build reversemap
		s.nodict = true
		s.reverseMap = nil
		s.sb.Reset()
		if s.values.hasNull {
			s.starts.scan(0, nil) // learn NULL
		}
		// build will fill our stringbuffer
	}
	s.allsize = s.allsize + len(v)
	if s.nodict {
		s.starts.scan(i, s.allsize)
		s.lens.scan(i, len(v))
	} else {
		start, ok := s.reverseMap[v]
		if ok {
			// reuse of string
		} else {
			// learn
			start[0] = s.count
			start[1] = uint(s.sb.Len())
			start[2] = uint(len(v))
			s.sb.WriteString(v)
			s.starts.scan(start[0], start[1])
			s.lens.scan(start[0], start[2])
			s.reverseMap[v] = start
			s.count = s.count + 1
		}
		s.values.scan(i, start[0])
	}
}
func (s *StorageString) init(i uint) {
	if s.nodict {
		// do not init values, sb andsoon
		s.starts.init(i)
		s.lens.init(i)
	} else {
		// allocate
		s.dictionary = s.sb.String() // extract one big slice with all strings (no extra memory structure)
		s.sb.Reset() // free the memory
		// prefixed strings are not accounted with that, but maybe this could be checked later??
		s.values.init(i)
		// take over dictionary
		s.starts.init(s.count)
		s.lens.init(s.count)
		for _, start := range s.reverseMap {
			// we read the value from dictionary, so we can free up all the single-strings
			s.starts.build(start[0], start[1])
			s.lens.build(start[0], start[2])
		}
	}
}
func (s *StorageString) build(i uint, value scm.Scmer) {
	// store
	var v string
	switch v_ := value.(type) {
		case string:
			v = v_
		default:
			// NULL = 1 1
			if s.nodict {
				s.starts.build(i, nil)
			} else {
				s.values.build(i, nil)
			}
			return
	}
	if s.nodict {
		s.starts.build(i, s.sb.Len())
		s.lens.build(i, len(v))
		s.sb.WriteString(v)
	} else {
		start := s.reverseMap[v]
		// write start+end into sub storage maps
		s.values.build(i, start[0])
	}
}
func (s *StorageString) finish() {
	if s.nodict {
		s.dictionary = s.sb.String()
		s.sb.Reset()
	} else {
		s.reverseMap = nil
		s.values.finish()
	}
	s.starts.finish()
	s.lens.finish()
}
func (s *StorageString) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}

