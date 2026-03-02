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
	"runtime"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func buildBenchStorageSeq(n int) *StorageSeq {
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64(i * 7))
	}
	return buildStorageSeq(values)
}

func buildBenchShardSeq(s *StorageSeq, count uint32) *storageShard {
	t := &table{
		Columns: []*column{{Name: "x", Typ: "float"}},
	}
	shard := &storageShard{
		t:            t,
		columns:      map[string]ColumnStorage{"x": s},
		deltaColumns: make(map[string]int),
		main_count:   count,
	}
	shard.deletions.Reset()
	return shard
}

func BenchmarkStorageSeqJITCompileTmp(b *testing.B) {
	s := buildBenchStorageSeq(benchSeqN)
	b.Run("GetValue_ConstFold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildGetValueFunc(b, s, true)
			cleanup()
		}
	})
	b.Run("GetValue_RegPtr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildGetValueFunc(b, s, false)
			cleanup()
		}
	})
	b.Run("SumLoop_ConstFold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildSumFuncGeneric(b, s, benchSeqN, true)
			cleanup()
		}
	})
	b.Run("SumLoop_RegPtr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildSumFuncGeneric(b, s, benchSeqN, false)
			cleanup()
		}
	})
}

func BenchmarkStorageSeqSumTmp(b *testing.B) {
	s := buildBenchStorageSeq(benchSeqN)

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
		if got := jitSum(); got != expectedSum {
			b.Fatalf("JIT ConstFold sum mismatch: got %d, want %d", got, expectedSum)
		}
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
		if got := jitSum(); got != expectedSum {
			b.Fatalf("JIT RegPtr sum mismatch: got %d, want %d", got, expectedSum)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitSum()
		}
	})

	b.Run("MapReducer", func(b *testing.B) {
		shard := buildBenchShardSeq(s, benchSeqN)

		mapProc := scm.NewProcStruct(scm.Proc{
			Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("x")}),
			Body:    scm.NewNthLocalVar(0),
			En:      &scm.Globalenv,
			NumVars: 1,
		})

		reduceProc := scm.NewProcStruct(scm.Proc{
			Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc"), scm.NewSymbol("new")}),
			Body: scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("+"),
				scm.NewNthLocalVar(0),
				scm.NewNthLocalVar(1),
			}),
			En:      &scm.Globalenv,
			NumVars: 2,
		})

		mr := shard.OpenMapReducer([]string{"x"}, mapProc, reduceProc)
		defer mr.Close()

		recids := make([]uint32, benchSeqN)
		for i := range recids {
			recids[i] = uint32(i)
		}

		got := mr.Stream(scm.NewInt(0), recids)
		if int64(got.Float()) != expectedSum {
			b.Fatalf("MapReducer sum mismatch: got %v, want %d", got, expectedSum)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mr.Stream(scm.NewInt(0), recids)
		}
	})
}

func BenchmarkStorageSeqEvalInterpreterTmp(b *testing.B) {
	s := buildBenchStorageSeq(benchSeqN)

	var expectedSum int64
	for i := uint32(0); i < benchSeqN; i++ {
		v := s.GetValue(i)
		if !v.IsNil() {
			expectedSum += int64(v.Float())
		}
	}
	b.Logf("expected SUM = %d", expectedSum)

	expr := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("+"),
		scm.NewNthLocalVar(0),
		scm.NewNthLocalVar(1),
	})
	env := &scm.Env{
		Vars:         make(scm.Vars),
		VarsNumbered: make([]scm.Scmer, 2),
		Outer:        &scm.Globalenv,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum := scm.NewInt(0)
		for j := uint32(0); j < benchSeqN; j++ {
			v := s.GetValue(j)
			if v.IsNil() {
				continue
			}
			env.VarsNumbered[0] = sum
			env.VarsNumbered[1] = v
			sum = scm.Eval(expr, env)
		}
		if got := int64(sum.Float()); got != expectedSum {
			b.Fatalf("Eval sum mismatch: got %d, want %d", got, expectedSum)
		}
	}
}
