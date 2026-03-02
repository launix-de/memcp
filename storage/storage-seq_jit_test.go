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
