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
	values      []scm.Scmer
	minIntScale int8 // power-of-ten exponent of finest granularity
	//   0 = pure ints, -2 = 2 decimal places, 2 = multiples of 100
	//   math.MinInt8 = not representable as scaled int
	hasString    bool
	longStrings  int
	null         uint  // amount of NULL values (sparse map!)
	numSeq       uint  // sequence statistics
	last1, last2 int64 // sequence statistics

	// enum detection: collect up to enumMaxSymbols distinct values
	enumVals  [enumMaxSymbols]scm.Scmer
	enumFreqs [enumMaxSymbols]uint64
	enumK     uint8 // number of distinct values seen so far (0xFF = abandoned)
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

func (s *StorageSCMER) GetCachedReader() ColumnReader { return s }

func (s *StorageSCMER) GetValue(i uint32) scm.Scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint32, value scm.Scmer) {
	// enum detection: track up to enumMaxSymbols distinct values with frequencies
	if s.enumK != 0xFF {
		found := false
		for j := uint8(0); j < s.enumK; j++ {
			// Use strict comparison: NULL is only equal to NULL
			// (scm.Equal treats NULL == 0 == false per Scheme semantics,
			// but storage needs them distinguished)
			if value.IsNil() == s.enumVals[j].IsNil() && (value.IsNil() || scm.Equal(s.enumVals[j], value)) {
				s.enumFreqs[j]++
				found = true
				break
			}
		}
		if !found {
			if s.enumK < enumMaxSymbols {
				s.enumVals[s.enumK] = value
				s.enumFreqs[s.enumK] = 1
				s.enumK++
			} else {
				s.enumK = 0xFF // abandon enum tracking
			}
		}
	}

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
		// scale detection: only if not yet abandoned
		if s.minIntScale > math.MinInt8 {
			exp := trailingZeroPow10(v2)
			if exp < s.minIntScale {
				s.minIntScale = exp
			}
		}
		if v2-s.last1 == s.last1-s.last2 {
			s.numSeq++
		}
		s.last2 = s.last1
		s.last1 = v2
		return
	}
	if value.IsFloat() {
		f := value.Float()
		// scale detection: only if not yet abandoned
		if s.minIntScale > math.MinInt8 {
			exp := detectFloatScale(f)
			if exp < s.minIntScale {
				s.minIntScale = exp
			}
		}
		// sequence statistics for integer-valued floats
		if _, frac := math.Modf(f); frac == 0.0 {
			v := int64(f)
			if v-s.last1 == s.last1-s.last2 {
				s.numSeq++
			}
			s.last2 = s.last1
			s.last1 = v
		}
		return
	}
	// non-numeric → no integer scaling possible
	s.minIntScale = math.MinInt8
	if value.IsString() {
		s.hasString = true
		if len(value.String()) > 255 {
			s.longStrings++
		}
	}
}
func (s *StorageSCMER) prepare() {
	s.minIntScale = math.MaxInt8 // neutral, gets driven down by scan
	s.hasString = false
	s.enumK = 0
}
func (s *StorageSCMER) init(i uint32) {
	// allocate
	s.values = make([]scm.Scmer, i)
}
func (s *StorageSCMER) build(i uint32, value scm.Scmer) {
	// store
	s.values[i] = value
}
func (s *StorageSCMER) finish() {
}

// soley to StorageSCMER
func (s *StorageSCMER) proposeCompression(i uint32) ColumnStorage {
	if s.null*100 > uint(i)*13 {
		// sparse payoff against bitcompressed is at ~13%
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageSparse)
			return b
		}
		return new(StorageSparse)
	}
	// enum detection: if <=8 distinct values and not abandoned, propose StorageEnum
	// but only when the distribution is skewed enough to beat PFOR's ceil(log2(k)) bits/element
	if s.enumK != 0xFF && s.enumK >= 2 && i > 0 {
		// compute Shannon entropy in bits
		n := float64(i)
		entropy := 0.0
		for j := uint8(0); j < s.enumK; j++ {
			if s.enumFreqs[j] > 0 {
				p := float64(s.enumFreqs[j]) / n
				entropy -= p * math.Log2(p)
			}
		}
		// PFOR cost: ceil(log2(k)) bits/element
		pforBits := math.Ceil(math.Log2(float64(s.enumK)))
		// only propose rANS when entropy is significantly below PFOR
		if entropy < pforBits*0.8 {
			return new(StorageEnum)
		}
	}
	if s.hasString {
		if s.longStrings > 2 {
			b := new(OverlayBlob)
			b.Base = new(StorageString)
			return b
		}
		return new(StorageString)
	}
	// scalable numerics (replaces onlyInt and onlyFloat)
	if s.minIntScale > math.MinInt8 {
		if s.minIntScale == 0 || s.minIntScale == math.MaxInt8 {
			// pure integers (MaxInt8 = all values were 0 or no non-NULL numerics)
			if i > 5 && 2*(uint(i)-s.numSeq) < uint(i) {
				return new(StorageSeq)
			}
			return new(StorageInt)
		}
		// scaled integers: multiples of 10^n or decimal places
		return &StorageDecimal{scaleExp: s.minIntScale}
	}
	// arbitrary floats (minIntScale == MinInt8 && !hasString)
	if !s.hasString {
		return new(StorageFloat)
	}
	if s.null*2 > uint(i) {
		// sparse payoff against StorageSCMER is at 2.1
		return new(StorageSparse)
	}
	// don't propose another pass
	return nil
}
