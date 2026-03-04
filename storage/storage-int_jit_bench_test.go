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
func jitBuildRawFunc(tb testing.TB, s *StorageInt, constThisptr bool) (fn func(int64) scm.Scmer, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT benchmarks only on amd64")
	}

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		W:        w,
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	// Entry: Go ABI — RAX = int64 index argument
	// Move index to a free register so RAX is available for result
	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s))))}
	} else {
		ptrReg := ctx.AllocReg()
		w.EmitMovRegImm64(ptrReg, uint64(uintptr(unsafe.Pointer(s))))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: ptrReg}
	}

	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)
	ctx.EmitMovPairToResult(&desc, &result)
	w.EmitByte(0xC3) // RET

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
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

	tb.Logf("JIT code size: %d bytes (constThisptr=%v)", codeLen, constThisptr)

	return jitFn, func() {
		runtime.KeepAlive(ctx) // keeps SpillBuf-backed spill addresses valid for JIT code lifetime
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	}
}

// jitBuildSumFunc compiles a JIT function that loops 0..count-1, reads each
// value via JITEmit and accumulates SUM in R14. Returns func() int64.
func jitBuildSumFunc(tb testing.TB, s *StorageInt, count int64, constThisptr bool) (fn func() int64, cleanup func()) {
	tb.Helper()
	if runtime.GOARCH != "amd64" {
		tb.Skip("JIT benchmarks only on amd64")
	}

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	// R15 = loop counter, R14 = accumulator (SUM), R12 = slice base (unused but reserved)
	// R13 = thisptr for LocReg case
	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)))
	// R11 = scratch, R14 = accumulator, R15 = loop counter — all reserved
	ctx := &scm.JITContext{
		W:        w,
		FreeRegs: freeRegs,
		AllRegs:  freeRegs,
	}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s))))}
	} else {
		ctx.FreeRegs &^= 1 << uint(scm.RegR13)
		w.EmitMovRegImm64(scm.RegR13, uint64(uintptr(unsafe.Pointer(s))))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegR13}
	}

	// PUSH R14 (save Go's g pointer)
	w.EmitByte(0x41)
	w.EmitByte(0x56)
	// XOR R15, R15 (zero loop counter)
	w.EmitByte(0x4D)
	w.EmitByte(0x31)
	w.EmitByte(0xFF)
	// XOR R14, R14 (zero accumulator)
	w.EmitByte(0x4D)
	w.EmitByte(0x31)
	w.EmitByte(0xF6)

	lblTop := w.ReserveLabel()
	w.MarkLabel(lblTop)

	// Copy R15 to scratch for body consumption
	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegR15)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	// Emit JITEmit body — result goes wherever JITEmit chooses
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)

	// Scmer LocRegPair: Reg=sentinel ptr, Reg2=int value (aux field).
	// For null: EmitMakeNil zeroes both, so adding Reg2=0 is correct for SUM.
	// ADD R14, desc.Reg2
	w.EmitAddInt64(scm.RegR14, desc.Reg2)
	ctx.FreeDesc(&desc)

	// INC R15
	w.EmitByte(0x49)
	w.EmitByte(0xFF)
	w.EmitByte(0xC7)

	// CMP R15, count
	w.EmitCmpRegImm32(scm.RegR15, int32(count))

	// JL loopTop
	w.EmitJcc(scm.CcL, lblTop)

	// MOV RAX, R14 (return accumulator)
	w.EmitByte(0x4C)
	w.EmitByte(0x89)
	w.EmitByte(0xF0)
	// POP R14 (restore Go's g pointer)
	w.EmitByte(0x41)
	w.EmitByte(0x5E)
	// RET
	w.EmitByte(0xC3)

	w.ResolveFixups()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
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

	tb.Logf("JIT SUM code size: %d bytes (constThisptr=%v, count=%d)", codeLen, constThisptr, count)

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
		mapProc := scm.NewProcStruct(scm.Proc{
			Params:  scm.NewSlice([]scm.Scmer{scm.NewSymbol("x")}),
			Body:    scm.NewNthLocalVar(0),
			En:      &scm.Globalenv,
			NumVars: 1,
		})

		// reduce: (lambda (acc new) (+ acc new))
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

		// build recid list [0..benchN-1]
		recids := make([]uint32, benchN)
		for i := range recids {
			recids[i] = uint32(i)
		}

		// validate
		got := mr.Stream(scm.NewInt(0), recids)
		if got.Int() != expectedSum {
			b.Fatalf("MapReducer sum mismatch: got %d, want %d", got.Int(), expectedSum)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mr.Stream(scm.NewInt(0), recids)
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
