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

	ctx := &scm.JITContext{
		W: w,
		FreeRegs: (1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
			(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
			(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
			(1 << uint(scm.RegR11)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)),
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
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	}
}

// jitBuildLoopFunc compiles a JIT function that internally loops over 0..count-1,
// calling JITEmit for each index. Returns func() (no per-call overhead).
func jitBuildLoopFunc(tb testing.TB, s *StorageInt, count int64, constThisptr bool) (fn func(), cleanup func()) {
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

	// R15 = loop counter (not in free pool)
	// R13 = thisptr for LocReg case (not in free pool if used)
	ctx := &scm.JITContext{
		W: w,
		FreeRegs: (1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
			(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
			(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
			(1 << uint(scm.RegR11)) | (1 << uint(scm.RegR13)),
		// R15 reserved for loop counter, not in free pool
	}

	var thisptr scm.JITValueDesc
	if constThisptr {
		thisptr = scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s))))}
	} else {
		// Use R13 for thisptr (remove from free pool)
		ctx.FreeRegs &^= 1 << uint(scm.RegR13)
		w.EmitMovRegImm64(scm.RegR13, uint64(uintptr(unsafe.Pointer(s))))
		thisptr = scm.JITValueDesc{Loc: scm.LocReg, Reg: scm.RegR13}
	}

	// XOR R15, R15 (zero the loop counter)
	w.EmitByte(0x4D); w.EmitByte(0x31); w.EmitByte(0xFF)

	// Loop top label
	lblTop := w.ReserveLabel()
	w.MarkLabel(lblTop)

	// Copy R15 (loop counter) to a scratch register for the body to consume.
	// The body will destroy the scratch via IMUL etc., but R15 stays intact.
	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegR15)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	// Emit the JITEmit body — result goes to RAX+RBX (discarded each iteration)
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)
	// Free the result registers so the code-gen free pool is restored
	ctx.FreeDesc(&desc)

	// INC R15: REX.WB=0x49, FF /0, ModRM=0xC7 (reg=0, rm=R15&7=7)
	w.EmitByte(0x49); w.EmitByte(0xFF); w.EmitByte(0xC7)

	// CMP R15, count
	w.EmitCmpRegImm32(scm.RegR15, int32(count))

	// JL loopTop
	w.EmitJcc(scm.CcL, lblTop)

	// RET
	w.EmitByte(0xC3)

	w.ResolveFixups()

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
	jitFn := *(*func())(unsafe.Pointer(&hdrPtr))

	tb.Logf("JIT loop code size: %d bytes (constThisptr=%v, count=%d)", codeLen, constThisptr, count)

	return jitFn, func() {
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	}
}

// BenchmarkStorageIntGetValue — baseline: plain Go GetValue calls.
func BenchmarkStorageIntGetValue(b *testing.B) {
	s := buildBenchStorageInt(benchN)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := uint32(0); j < benchN; j++ {
			_ = s.GetValue(j)
		}
	}
}

// BenchmarkStorageIntJITLoop — JIT with internal loop (no Go call overhead per item).
func BenchmarkStorageIntJITLoop(b *testing.B) {
	s := buildBenchStorageInt(benchN)
	b.Run("ConstFold", func(b *testing.B) {
		jitLoop, cleanup := jitBuildLoopFunc(b, s, benchN, true)
		defer cleanup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitLoop()
		}
	})
	b.Run("RegPtr", func(b *testing.B) {
		jitLoop, cleanup := jitBuildLoopFunc(b, s, benchN, false)
		defer cleanup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			jitLoop()
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
