package storage

import (
	"encoding/hex"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

func TestStorageSeqJITDebugMulti(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("debug JIT emit skipped: %v", r)
		}
	}()
	if runtime.GOARCH != "amd64" {
		t.Skip("JIT only on amd64")
	}
	values := []scm.Scmer{
		scm.NewFloat(0), scm.NewFloat(1), scm.NewFloat(2), scm.NewFloat(3), scm.NewFloat(4),
		scm.NewFloat(100), scm.NewFloat(200), scm.NewFloat(300), scm.NewFloat(400), scm.NewFloat(500),
	}
	s := buildStorageSeq(values)
	t.Logf("seqCount=%d count=%d", s.seqCount, s.count)
	t.Logf("recordId: bitsize=%d offset=%d hasNull=%v", s.recordId.bitsize, s.recordId.offset, s.recordId.hasNull)
	t.Logf("start: bitsize=%d offset=%d hasNull=%v", s.start.bitsize, s.start.offset, s.start.hasNull)
	t.Logf("stride: bitsize=%d offset=%d hasNull=%v", s.stride.bitsize, s.stride.offset, s.stride.hasNull)

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

	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)
	ctx.EmitMovPairToResult(&desc, &result)
	w.EmitByte(0xC3) // RET
	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]
	t.Logf("JIT code size: %d bytes", codeLen)

	os.WriteFile("/tmp/jit_seq_multi.bin", code, 0644)
	t.Logf("Wrote /tmp/jit_seq_multi.bin — disassemble with: objdump -D -b binary -m i386:x86-64 /tmp/jit_seq_multi.bin")

	dumpLen := codeLen
	if dumpLen > 300 {
		dumpLen = 300
	}
	t.Logf("First %d bytes:\n%s", dumpLen, hex.Dump(code[:dumpLen]))

	// Execute the JIT code for all indices
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
	defer func() {
		runtime.KeepAlive(ctx)
		syscall.Munmap(b)
	}()
	type funcHeader struct{ fnptr *byte }
	hdr := &funcHeader{fnptr: &b[0]}
	hdrPtr := unsafe.Pointer(hdr)
	jitFn := *(*func(int64) scm.Scmer)(unsafe.Pointer(&hdrPtr))
	for i := 0; i < 10; i++ {
		s.lastValue.Store(0) // reset hint
		got := jitFn(int64(i))
		expected := s.GetValue(uint32(i))
		match := "OK"
		if got.Float() != expected.Float() {
			match = "WRONG"
		}
		t.Logf("idx=%d: got=%.0f expected=%.0f lastValue=%d %s",
			i, got.Float(), expected.Float(), s.lastValue.Load(), match)
	}
}

func TestStorageSeqJITDebug(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("debug JIT emit skipped: %v", r)
		}
	}()
	if runtime.GOARCH != "amd64" {
		t.Skip("JIT only on amd64")
	}
	n := 100
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	t.Logf("seqCount=%d count=%d", s.seqCount, s.count)
	t.Logf("recordId: bitsize=%d offset=%d hasNull=%v", s.recordId.bitsize, s.recordId.offset, s.recordId.hasNull)
	t.Logf("start: bitsize=%d offset=%d hasNull=%v", s.start.bitsize, s.start.offset, s.start.hasNull)
	t.Logf("stride: bitsize=%d offset=%d hasNull=%v", s.stride.bitsize, s.stride.offset, s.stride.hasNull)

	// Dump code for offline disassembly
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

	idxReg := ctx.AllocReg()
	w.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}

	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)
	ctx.EmitMovPairToResult(&desc, &result)
	w.EmitByte(0xC3) // RET
	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]
	t.Logf("JIT code size: %d bytes", codeLen)

	// Write binary for objdump
	os.WriteFile("/tmp/jit_seq.bin", code, 0644)
	t.Logf("Wrote /tmp/jit_seq.bin — disassemble with: objdump -D -b binary -m i386:x86-64 /tmp/jit_seq.bin")

	// Also dump first 200 bytes as hex
	dumpLen := codeLen
	if dumpLen > 200 {
		dumpLen = 200
	}
	t.Logf("First %d bytes:\n%s", dumpLen, hex.Dump(code[:dumpLen]))

	// Compute expected values BEFORE JIT (Go GetValue is broken after JIT corrupts lastValue)
	expectedVals := make([]scm.Scmer, 5)
	for i := 0; i < 5; i++ {
		expectedVals[i] = s.GetValue(uint32(i))
	}

	// Now test the JIT
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := int64(0); i < 5; i++ {
		s.lastValue.Store(0) // reset before each call
		got := jitGet(i)
		lv := s.lastValue.Load()
		exp := expectedVals[i]
		// RAX=ptr, RBX=aux in the JIT calling convention
		type scmerRaw struct {
			ptr uintptr
			aux uint64
		}
		rawGot := *(*scmerRaw)(unsafe.Pointer(&got))
		rawExp := *(*scmerRaw)(unsafe.Pointer(&exp))
		t.Logf("JIT(%d): ptr=0x%x aux=0x%x lastValue=%d | expected: ptr=0x%x aux=0x%x",
			i, rawGot.ptr, rawGot.aux, lv, rawExp.ptr, rawExp.aux)
	}
}
