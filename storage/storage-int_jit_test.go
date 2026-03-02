package storage

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// jitExecStorageInt builds JIT code via s.JITEmit and returns a callable
// func(...Scmer) Scmer. Call it with a single Scmer arg (the index as int).
func jitExecStorageInt(t *testing.T, s *StorageInt) func(...scm.Scmer) scm.Scmer {
	t.Helper()
	if runtime.GOARCH != "amd64" {
		t.Skip("JIT tests only on amd64")
	}

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	// Same register setup as jitCompileExprBody (R11 reserved as scratch)
	freeRegs := uint64((1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) |
		(1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) |
		(1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) |
		(1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{
		W:         w,
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: scm.RegR12,
	}

	// Entry: func(...Scmer) Scmer calling convention
	// RAX = pointer to args slice data (*Scmer base), RBX = len, RCX = cap
	// Save slice base in R12
	w.EmitMovRegReg(scm.RegR12, scm.RegRAX)

	// Load args[0] as Scmer pair (the index)
	idxReg := ctx.AllocReg()
	idxReg2 := ctx.AllocReg()
	w.EmitLoadArgPair(idxReg, idxReg2, scm.RegR12, 0)
	idx := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: idxReg, Reg2: idxReg2}

	// thisptr is compile-time known (the *StorageInt pointer as LocImm)
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s))))}

	// Result into RAX+RBX
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)

	// Move result to RAX+RBX if not already there
	ctx.EmitMovPairToResult(&desc, &result)

	// RET
	w.EmitByte(0xC3)

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]

	// Allocate executable memory
	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatalf("mmap failed: %v", err)
	}
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil {
		syscall.Munmap(b)
		t.Fatalf("mprotect failed: %v", err)
	}

	// Build func(...Scmer) Scmer — closure is an indirect pointer:
	// Go func value = pointer to struct { fnptr *byte }
	type funcHeader struct {
		fnptr *byte
	}
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	jitFn := *(*func(...scm.Scmer) scm.Scmer)(unsafe.Pointer(&hdrPtr))

	t.Cleanup(func() {
		runtime.KeepAlive(hdr)
		syscall.Munmap(b)
	})

	return jitFn
}

func TestStorageIntJITEmit(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(10), scm.NewInt(20), scm.NewInt(30), scm.NewInt(40), scm.NewInt(50),
	}
	s := buildStorageInt(values)
	jitGet := jitExecStorageInt(t, s)

	for i := uint32(0); i < uint32(len(values)); i++ {
		got := jitGet(scm.NewInt(int64(i)))
		expected := s.GetValue(i)
		if expected.IsNil() != got.IsNil() {
			t.Errorf("idx=%d: nil mismatch: JIT=%v GetValue=%v", i, got, expected)
		} else if !expected.IsNil() && got.Int() != expected.Int() {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageIntJITEmitWithNull(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(5), scm.NewNil(), scm.NewInt(10), scm.NewNil(), scm.NewInt(15),
	}
	s := buildStorageInt(values)
	jitGet := jitExecStorageInt(t, s)

	for i := uint32(0); i < uint32(len(values)); i++ {
		got := jitGet(scm.NewInt(int64(i)))
		expected := s.GetValue(i)
		if expected.IsNil() != got.IsNil() {
			t.Errorf("idx=%d: nil mismatch: JIT=%v GetValue=%v", i, got, expected)
		} else if !expected.IsNil() && got.Int() != expected.Int() {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageIntJITEmitWithOffset(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(-100), scm.NewInt(-50), scm.NewInt(0), scm.NewInt(50), scm.NewInt(100),
	}
	s := buildStorageInt(values)
	jitGet := jitExecStorageInt(t, s)

	for i := uint32(0); i < uint32(len(values)); i++ {
		got := jitGet(scm.NewInt(int64(i)))
		expected := s.GetValue(i)
		if expected.IsNil() != got.IsNil() {
			t.Errorf("idx=%d: nil mismatch: JIT=%v GetValue=%v", i, got, expected)
		} else if !expected.IsNil() && got.Int() != expected.Int() {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

// jitExecRawStorageInt builds JIT code with func(int64) Scmer signature (LocReg thisptr).
func jitExecRawStorageInt(t *testing.T, s *StorageInt) func(int64) scm.Scmer {
	t.Helper()
	fn, cleanup := jitBuildRawFunc(t, s, false)
	t.Cleanup(cleanup)
	return fn
}

func TestStorageIntJITEmitRegPtr(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(10), scm.NewInt(20), scm.NewInt(30), scm.NewInt(40), scm.NewInt(50),
	}
	s := buildStorageInt(values)
	jitGet := jitExecRawStorageInt(t, s)

	for i := uint32(0); i < uint32(len(values)); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(i)
		if expected.IsNil() != got.IsNil() {
			t.Errorf("idx=%d: nil mismatch: JIT=%v GetValue=%v", i, got, expected)
		} else if !expected.IsNil() && got.Int() != expected.Int() {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageIntJITEmitRegPtrWithNull(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(5), scm.NewNil(), scm.NewInt(10), scm.NewNil(), scm.NewInt(15),
	}
	s := buildStorageInt(values)
	jitGet := jitExecRawStorageInt(t, s)

	for i := uint32(0); i < uint32(len(values)); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(i)
		if expected.IsNil() != got.IsNil() {
			t.Errorf("idx=%d: nil mismatch: JIT=%v GetValue=%v", i, got, expected)
		} else if !expected.IsNil() && got.Int() != expected.Int() {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}
