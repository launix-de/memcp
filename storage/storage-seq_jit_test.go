/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	"bytes"
	"math/rand"
	"runtime"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func buildStorageSeq(values []scm.Scmer) *StorageSeq {
	col := buildViaCompression(len(values), func(i int) scm.Scmer { return values[i] })
	if ss, ok := col.(*StorageSeq); ok {
		return ss
	}
	// Force StorageSeq
	s := new(StorageSeq)
	s.prepare()
	for i, v := range values {
		s.scan(uint32(i), v)
	}
	s.init(uint32(len(values)))
	for i, v := range values {
		s.build(uint32(i), v)
	}
	s.finish()
	return s
}

func TestStorageSeqJITEmitLinear(t *testing.T) {
	// Simple linear sequence: 0, 1, 2, 3, ..., 99
	n := 100
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageSeqJITEmitStride(t *testing.T) {
	// Sequence with stride: 10, 20, 30, ..., 500
	n := 50
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64((i + 1) * 10))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageSeqJITEmitMultipleSequences(t *testing.T) {
	// Two sequences: 0,1,2,3,4 then 100,200,300,400,500
	values := []scm.Scmer{
		scm.NewFloat(0), scm.NewFloat(1), scm.NewFloat(2), scm.NewFloat(3), scm.NewFloat(4),
		scm.NewFloat(100), scm.NewFloat(200), scm.NewFloat(300), scm.NewFloat(400), scm.NewFloat(500),
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageSeqJITEmitWithNull(t *testing.T) {
	// Two contiguous blocks: nulls then a stride-10 sequence.
	// Alternating nulls cannot be stored as a valid 2-sequence StorageSeq.
	values := []scm.Scmer{
		scm.NewNil(), scm.NewNil(), scm.NewFloat(10), scm.NewFloat(20), scm.NewFloat(30),
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v (nil=%v), GetValue got %v (nil=%v)",
				i, got, got.IsNil(), expected, expected.IsNil())
		}
	}
}

func TestStorageSeqJITEmitRegPtr(t *testing.T) {
	n := 100
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64(i * 3))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, false)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageSeqJITEmitConstants(t *testing.T) {
	// All same value (stride=0 sequence)
	n := 20
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(42)
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func makeRandomStorageSeqValues(r *rand.Rand, n int) []scm.Scmer {
	values := make([]scm.Scmer, n)
	pos := 0
	for pos < n {
		segLen := 1 + r.Intn(16)
		if pos+segLen > n {
			segLen = n - pos
		}
		if r.Intn(6) == 0 {
			for j := 0; j < segLen; j++ {
				values[pos+j] = scm.NewNil()
			}
		} else {
			start := int64(r.Intn(400) - 200)
			stride := int64(r.Intn(11) - 5)
			for j := 0; j < segLen; j++ {
				values[pos+j] = scm.NewFloat(float64(start + int64(j)*stride))
			}
		}
		pos += segLen
	}
	return values
}

func checkStorageSeqJITParity(t *testing.T, values []scm.Scmer, constThisptr bool, access []int64) {
	t.Helper()
	sExpected := buildStorageSeq(values)
	sJIT := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, sJIT, constThisptr)
	defer cleanup()

	for k, idx := range access {
		expected := sExpected.GetValue(uint32(idx))
		got := jitGet(idx)
		if !scmerEqual(got, expected) {
			t.Fatalf("step=%d idx=%d constThisptr=%v: JIT got %v (nil=%v), GetValue got %v (nil=%v)",
				k, idx, constThisptr, got, got.IsNil(), expected, expected.IsNil())
		}
	}
}

func TestStorageSeqJITEmitRandomizedParity(t *testing.T) {
	r := rand.New(rand.NewSource(20260302))
	for tc := 0; tc < 12; tc++ {
		n := 64 + r.Intn(192)
		values := makeRandomStorageSeqValues(r, n)

		seqAccess := make([]int64, 0, 3*n)
		for i := 0; i < n; i++ {
			seqAccess = append(seqAccess, int64(i))
		}
		for i := n - 1; i >= 0; i-- {
			seqAccess = append(seqAccess, int64(i))
		}
		for i := 0; i < n; i++ {
			seqAccess = append(seqAccess, int64(r.Intn(n)))
		}

		t.Run("const_ptr", func(t *testing.T) {
			checkStorageSeqJITParity(t, values, true, seqAccess)
		})
		t.Run("reg_ptr", func(t *testing.T) {
			checkStorageSeqJITParity(t, values, false, seqAccess)
		})
	}
}

const benchSeqN = 60000

func BenchmarkStorageSeqGetValue(b *testing.B) {
	values := make([]scm.Scmer, benchSeqN)
	for i := range values {
		values[i] = scm.NewFloat(float64(i * 7))
	}
	s := buildStorageSeq(values)

	// Pre-compute expected sum
	var expectedSum int64
	for i := uint32(0); i < benchSeqN; i++ {
		v := s.GetValue(i)
		if !v.IsNil() {
			expectedSum += int64(v.Float())
		}
	}
	b.Logf("expected SUM = %d", expectedSum)

	b.Run("Go", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var sum int64
			for j := uint32(0); j < benchSeqN; j++ {
				v := s.GetValue(j)
				if !v.IsNil() {
					sum += int64(v.Float())
				}
			}
			if sum != expectedSum {
				b.Fatalf("sum mismatch: got %d, want %d", sum, expectedSum)
			}
		}
	})

	b.Run("JIT_ConstFold", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitSum, cleanup := jitBuildSumFuncGeneric(b, s, benchSeqN, true)
		defer cleanup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitSum()
		}
	})

	b.Run("JIT_RegPtr", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitSum, cleanup := jitBuildSumFuncGeneric(b, s, benchSeqN, false)
		defer cleanup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitSum()
		}
	})
}

func TestStorageSeqNullTransitionPreservesInput(t *testing.T) {
	cases := [][]scm.Scmer{
		{scm.NewInt(1), scm.NewNil(), scm.NewInt(2), scm.NewInt(3), scm.NewNil(), scm.NewInt(4)},
		{scm.NewNil(), scm.NewNil(), scm.NewInt(-5), scm.NewInt(-4), scm.NewNil(), scm.NewInt(-3)},
		{scm.NewInt(10), scm.NewInt(20), scm.NewNil(), scm.NewInt(30), scm.NewInt(40), scm.NewNil()},
	}
	for ci, values := range cases {
		s := buildStorageSeq(values)
		var serialized bytes.Buffer
		s.Serialize(&serialized)
		if magic, err := serialized.ReadByte(); err != nil || magic != 11 {
			t.Fatalf("sequence magic = %d, err %v", magic, err)
		}
		var restored StorageSeq
		if count := restored.Deserialize(&serialized); count != uint(len(values)) {
			t.Fatalf("restored count = %d, want %d", count, len(values))
		}
		for i, want := range values {
			if got := restored.GetValue(uint32(i)); !scmerEqual(got, want) {
				t.Errorf("case %d restored row %d: got %v, want %v", ci, i, got, want)
			}
		}
		got := make([]scm.Scmer, len(values))
		s.GetValueRange(0, uint32(len(values)), got, 1)
		for i, want := range values {
			if scalar := s.GetValue(uint32(i)); !scmerEqual(scalar, want) {
				t.Errorf("case %d scalar row %d: got %v, want %v", ci, i, scalar, want)
			}
			if !scmerEqual(got[i], want) {
				t.Errorf("case %d range row %d: got %v, want %v", ci, i, got[i], want)
			}
		}
		ids := []uint32{uint32(len(values) - 1), 0, 2, 1, 2}
		multi := make([]scm.Scmer, len(ids)*2)
		s.GetValueMulti(ids, multi, 2)
		for i, id := range ids {
			if !scmerEqual(multi[i*2], values[id]) {
				t.Errorf("case %d multi row %d: got %v, want %v", ci, id, multi[i*2], values[id])
			}
		}
	}
}

func TestStorageSeqJITNullTransitionsMatchInput(t *testing.T) {
	values := []scm.Scmer{scm.NewInt(1), scm.NewNil(), scm.NewInt(2), scm.NewInt(3), scm.NewNil(), scm.NewInt(4)}
	s := buildStorageSeq(values)
	get, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()
	for i, want := range values {
		if got := get(int64(i)); !scmerEqual(got, want) {
			t.Errorf("JIT row %d: got %v, want %v", i, got, want)
		}
	}
}
