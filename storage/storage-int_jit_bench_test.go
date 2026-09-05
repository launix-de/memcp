/*
Copyright (C) 2026  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package storage

import (
	"math/rand"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

const benchN = 60000

// buildBenchStorageInt creates a StorageInt with n random values (0..999).
func buildBenchStorageInt(n int) *StorageInt {
	rng := rand.New(rand.NewSource(42))
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewInt(int64(rng.Intn(1000)))
	}
	return buildStorageInt(values)
}

// jitBuildRawFunc compiles JIT code for StorageInt and returns func(int64) scm.Scmer.
// Go ABI: int64 arg in RAX, Scmer result in RAX+RBX. Zero allocations per call.
func jitBuildRawFunc(tb testing.TB, s *StorageInt, _ bool) (fn func(int64) scm.Scmer, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT benchmarks only on amd64")
	}

	codeBuf := make([]byte, 65536)
	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		Ptr:      unsafe.Pointer(&codeBuf[0]),
		Start:    unsafe.Pointer(&codeBuf[0]),
		End:      unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	// Entry: Go ABI — RAX = int64 index argument
	// Move index to a free register so RAX is available for result
	idxReg := ctx.AllocReg()
	ctx.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, idx, result)
	ctx.EmitMovPairToResult(&desc, &result)
	ctx.EmitByte(0xC3) // RET

	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	code := codeBuf[:codeLen]

	// Allocate executable memory
	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		tb.Fatalf("mmap failed: %v", err)
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		tb.Fatalf("mprotect failed: %v", err)
	}

	type funcHeader struct {
		fnptr *byte
	}
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	jitFn := *(*func(int64) scm.Scmer)(unsafe.Pointer(&hdrPtr))

	tb.Logf("JIT code size: %d bytes (storage-specialized)", codeLen)

	return jitFn, func() {
		runtime.KeepAlive(ctx) // keeps compile-time roots alive through the native call
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	}
}

// jitBuildSumFunc compiles a JIT function that loops 0..count-1, reads each
// value via JITEmit and accumulates SUM in R14. Returns func() int64.
func jitBuildSumFunc(tb testing.TB, s *StorageInt, count int64, _ bool) (fn func() int64, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT benchmarks only on amd64")
	}

	codeBuf := make([]byte, 65536)
	// R15 = loop counter, R14 = accumulator (SUM), R12 = slice base (unused but reserved)
	// R13 remains available to the specialized storage emitter.
	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)))
	// R11 = scratch, R14 = accumulator, R15 = loop counter — all reserved
	ctx := &scm.JITContext{
		Ptr:      unsafe.Pointer(&codeBuf[0]),
		Start:    unsafe.Pointer(&codeBuf[0]),
		End:      unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	// PUSH R14 (save Go's g pointer)
	ctx.EmitByte(0x41)
	ctx.EmitByte(0x56)
	// XOR R15, R15 (zero loop counter)
	ctx.EmitByte(0x4D)
	ctx.EmitByte(0x31)
	ctx.EmitByte(0xFF)
	// XOR R14, R14 (zero accumulator)
	ctx.EmitByte(0x4D)
	ctx.EmitByte(0x31)
	ctx.EmitByte(0xF6)

	lblTop := ctx.ReserveLabel()
	ctx.MarkLabel(lblTop)

	// Copy R15 to scratch for body consumption
	idxReg := ctx.AllocReg()
	ctx.EmitMovRegReg(idxReg, scm.RegR15)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	// Emit JITEmit body — result goes wherever JITEmit chooses
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, idx, result)

	// Scmer LocRegPair: Reg=sentinel ptr, Reg2=int value (aux field).
	// For null: EmitMakeNil zeroes both, so adding Reg2=0 is correct for SUM.
	// ADD R14, desc.Reg2
	ctx.EmitAddInt64(scm.RegR14, desc.Reg2)
	ctx.FreeDesc(&desc)

	// INC R15
	ctx.EmitByte(0x49)
	ctx.EmitByte(0xFF)
	ctx.EmitByte(0xC7)

	// CMP R15, count
	ctx.EmitCmpRegImm32(scm.RegR15, int32(count))

	// JL loopTop
	ctx.EmitJcc(scm.CcL, lblTop)

	// MOV RAX, R14 (return accumulator)
	ctx.EmitByte(0x4C)
	ctx.EmitByte(0x89)
	ctx.EmitByte(0xF0)
	// POP R14 (restore Go's g pointer)
	ctx.EmitByte(0x41)
	ctx.EmitByte(0x5E)
	// RET
	ctx.EmitByte(0xC3)

	ctx.ResolveFixups()

	codeLen := int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	code := codeBuf[:codeLen]

	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		tb.Fatalf("mmap failed: %v", err)
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		tb.Fatalf("mprotect failed: %v", err)
	}

	type funcHeader struct {
		fnptr *byte
	}
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	jitFn := *(*func() int64)(unsafe.Pointer(&hdrPtr))

	tb.Logf("JIT SUM code size: %d bytes (storage-specialized, count=%d)", codeLen, count)

	return jitFn, func() {
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	}
}

// buildBenchShard creates a minimal storageShard with a single "x" column backed by s.
func buildBenchShard(s *StorageInt, count uint32) *storageShard {
	t := &table{
		Columns: []*column{{Name: "x", Typ: "int"}},
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

// BenchmarkStorageIntSum — SUM(x) over 60k items across 4 implementations.
func BenchmarkStorageIntSum(b *testing.B) {
	s := buildBenchStorageInt(benchN)

	// Pre-compute expected sum for validation
	var expectedSum int64
	for i := uint32(0); i < benchN; i++ {
		v := s.GetValue(i)
		if !v.IsNil() {
			expectedSum += v.Int()
		}
	}
	b.Logf("expected SUM = %d", expectedSum)

	// 1) Go baseline: plain GetValue + accumulate
	b.Run("Go", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var sum int64
			for j := uint32(0); j < benchN; j++ {
				v := s.GetValue(j)
				if !v.IsNil() {
					sum += v.Int()
				}
			}
			if sum != expectedSum {
				b.Fatalf("sum mismatch: got %d, want %d", sum, expectedSum)
			}
		}
	})

	// 2) JIT ConstFold: thisptr baked as immediate
	b.Run("JIT_ConstFold", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitSum, cleanup := jitBuildSumFunc(b, s, benchN, true)
		defer cleanup()
		// validate
		if got := jitSum(); got != expectedSum {
			b.Fatalf("JIT ConstFold sum mismatch: got %d, want %d", got, expectedSum)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitSum()
		}
	})

	// 3) JIT RegPtr: thisptr in register
	b.Run("JIT_RegPtr", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitSum, cleanup := jitBuildSumFunc(b, s, benchN, false)
		defer cleanup()
		if got := jitSum(); got != expectedSum {
			b.Fatalf("JIT RegPtr sum mismatch: got %d, want %d", got, expectedSum)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitSum()
		}
	})

	// 4) MapReducer: Proc-based map=(lambda (x) x), reduce=(lambda (acc new) (+ acc new))
	b.Run("MapReducer", func(b *testing.B) {
		shard := buildBenchShard(s, benchN)

		// identity map: (lambda (x) x) — body = (var 0), NumVars=1
		mapReduceProc := scm.NewProcStruct(scm.Proc{
			Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc"), scm.NewSymbol("x")}),
			Body: scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("+"),
				scm.NewNthLocalVar(0),
				scm.NewNthLocalVar(1),
			}),
			En:      &scm.Globalenv,
			NumVars: 2,
		})

		mr := shard.OpenMapReducer([]string{"x"}, mapReduceProc, false, 1, nil, nil)
		defer mr.Close()

		// build recid list [0..benchN-1]
		recids := make([]uint32, benchN)
		for i := range recids {
			recids[i] = uint32(i)
		}

		// validate
		got := mr.Stream(scm.NewInt(0), recids, nil)
		if got.Int() != expectedSum {
			b.Fatalf("MapReducer sum mismatch: got %d, want %d", got.Int(), expectedSum)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mr.Stream(scm.NewInt(0), recids, nil)
		}
	})
}

// BenchmarkStorageIntJITCompile — measures JIT compilation time only.
func BenchmarkStorageIntJITCompile(b *testing.B) {
	s := buildBenchStorageInt(benchN)
	b.Run("ConstFold", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildRawFunc(b, s, true)
			cleanup()
		}
	})
	b.Run("RegPtr", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, cleanup := jitBuildRawFunc(b, s, false)
			cleanup()
		}
	})
}

// BenchmarkStorageIntFinishedReaders compares the normal immutable storage
// methods with the three typed functions installed by finish(). The fixtures,
// record order, targets, warmup and sample size are identical within each pair.
func BenchmarkStorageIntFinishedReaders(b *testing.B) {
	if !scm.JITEnabled() {
		b.Skip("requires the JIT experiment")
	}
	s := buildBenchStorageInt(benchN)
	scalar := s.GetJITGetValue()
	rangeReader := s.GetJITGetValueRange()
	multiReader := s.GetJITGetValueMulti()
	if scalar == nil || rangeReader == nil || multiReader == nil {
		b.Fatal("finish did not install every JIT reader")
	}

	b.Run("Scalar/Go", func(b *testing.B) {
		var sum int64
		b.ReportAllocs()
		b.ResetTimer()
		for sample := 0; sample < b.N; sample++ {
			for recid := uint32(0); recid < benchN; recid++ {
				sum += s.GetValue(recid).Int()
			}
		}
		runtime.KeepAlive(sum)
	})
	b.Run("Scalar/JIT", func(b *testing.B) {
		var sum int64
		b.ReportAllocs()
		b.ResetTimer()
		for sample := 0; sample < b.N; sample++ {
			for recid := uint32(0); recid < benchN; recid++ {
				sum += scalar(recid).Int()
			}
		}
		runtime.KeepAlive(sum)
	})

	target := make([]scm.Scmer, benchN)
	b.Run("Range/Go", func(b *testing.B) {
		b.ReportAllocs()
		for sample := 0; sample < b.N; sample++ {
			s.GetValueRange(0, benchN, target, 1)
		}
	})
	b.Run("Range/JIT", func(b *testing.B) {
		b.ReportAllocs()
		for sample := 0; sample < b.N; sample++ {
			rangeReader(0, benchN, target, 1)
		}
	})

	recids := make([]uint32, benchN)
	for index := range recids {
		recids[index] = uint32((index * 7919) % benchN)
	}
	b.Run("Multi/Go", func(b *testing.B) {
		b.ReportAllocs()
		for sample := 0; sample < b.N; sample++ {
			s.GetValueMulti(recids, target, 1)
		}
	})
	b.Run("Multi/JIT", func(b *testing.B) {
		b.ReportAllocs()
		for sample := 0; sample < b.N; sample++ {
			multiReader(recids, target, 1)
		}
	})
}
