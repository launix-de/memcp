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
import "sort"
import "github.com/launix-de/memcp/scm"

type StorageSeq struct {
	// data
	recordId,
	start,
	stride StorageInt
	seqCount uint // number of sequences

	// analysis
	lastValue, lastStride int64
	lastValueNil bool
	lastValueFirst bool
}

func (s *StorageSeq) String() string {
	return fmt.Sprintf("seq[%dx %s/%s]", s.seqCount, s.start.String(), s.stride.String())
}

func (s *StorageSeq) getValue(i uint) scm.Scmer {
	// bisect to the correct index where to find (lowest idx to find our sequence)
	idx := sort.Search(int(s.seqCount), func (idx int) bool {
		recid := s.recordId.getValueUInt(uint(idx))
		return uint64(i) < recid // return true as long as we are bigger than searched index
	}) - 1
	var value, stride int64
	if s.start.hasNegative {
		value = s.start.getValueInt(uint(idx))
	} else {
		value = int64(s.start.getValueUInt(uint(idx)))
	}
	if s.start.hasNull && value == int64(s.start.null) {
		return nil
	}
	if s.stride.hasNegative {
		stride = s.stride.getValueInt(uint(idx))
	} else {
		stride = int64(s.stride.getValueUInt(uint(idx)))
	}
	recid := s.recordId.getValueUInt(uint(idx))
	return float64(value + int64(uint64(i) - recid) * stride)

}

func (s *StorageSeq) prepare() {
	// set up scan
	s.recordId.prepare()
	s.start.prepare()
	s.stride.prepare()
}
func (s *StorageSeq) scan(i uint, value scm.Scmer) {
	if value == nil {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, i)
			s.start.scan(s.seqCount-1, nil)
			s.stride.scan(s.seqCount-1, 0)
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, i)
			s.start.scan(s.seqCount-1, nil)
			s.stride.scan(s.seqCount-1, 0)
		}
	} else {
		// integer
		v := toInt(value)
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue
			s.lastValue = v
			s.stride.scan(s.seqCount-1, s.lastStride)
		} else if i != 0 && v == s.lastValue + s.lastStride {
			// sequence stays the same
			s.lastValue = v
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue = v
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.scan(s.seqCount-1, i)
			s.start.scan(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) init(i uint) {
	s.recordId.init(s.seqCount)
	s.start.init(s.seqCount)
	s.stride.init(s.seqCount)
	s.lastValueNil = false
	s.lastValueFirst = false
	s.seqCount = 0
}
func (s *StorageSeq) build(i uint, value scm.Scmer) {
	// store
	if value == nil {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, i)
			s.start.build(s.seqCount-1, nil)
			s.stride.build(s.seqCount-1, 0)
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, i)
			s.start.build(s.seqCount-1, nil)
			s.stride.build(s.seqCount-1, 0)
		}
	} else {
		// integer
		v := toInt(value)
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue
			s.lastValue = v
			s.stride.build(s.seqCount-1, s.lastStride)
		} else if i != 0 && v == s.lastValue + s.lastStride {
			// sequence stays the same
			s.lastValue = v
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue = v
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.build(s.seqCount-1, i)
			s.start.build(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) finish() {
	s.recordId.finish()
	s.start.finish()
	s.stride.finish()

	/* debug output of the sequence:
	for i := uint(0); i < s.seqCount; i++ {
		fmt.Println(s.recordId.getValue(i),":",s.start.getValue(i),":",s.stride.getValue(i))
	}*/
}
func (s *StorageSeq) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}

