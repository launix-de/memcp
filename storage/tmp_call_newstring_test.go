/*
Copyright (C) 2024-2026  Carl-Philip Haensch
*/
package storage

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

func TestTmpJITCallNewStringBridge(t *testing.T) {
	if runtime.GOARCH != "amd64" { t.Skip("amd64") }
	in := "hello"
	inPtr := uintptr(unsafe.Pointer(unsafe.StringData(in)))
	inLen := len(in)

	codeBuf := make([]byte, 4096)
	w := &scm.JITWriter{Ptr: unsafe.Pointer(&codeBuf[0]), Start: unsafe.Pointer(&codeBuf[0]), End: unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-128)}
	freeRegs := uint64((1 << uint(scm.RegRAX)) | (1 << uint(scm.RegRBX)) | (1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) | (1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) | (1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) | (1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{W: w, FreeRegs: freeRegs, AllRegs: freeRegs}

	rPtr := ctx.AllocReg()
	ctx.W.EmitMovRegImm64(rPtr, uint64(inPtr))
	rLen := ctx.AllocRegExcept(rPtr)
	ctx.W.EmitMovRegImm64(rLen, uint64(inLen))
	arg := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: rPtr, Reg2: rLen}
	res := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{arg}, 2)
	out := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	ctx.EmitMovPairToResult(&res, &out)
	w.EmitByte(0xC3)
	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]
	pageSize := syscall.Getpagesize()
	n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil { t.Fatal(err) }
	copy(b, code)
	if err := syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); err != nil { t.Fatal(err) }
	defer syscall.Munmap(b)
	type fh struct{ fnptr *byte }
	h := &fh{fnptr: &b[0]}
	hp := unsafe.Pointer(h)
	fn := *(*func() scm.Scmer)(unsafe.Pointer(&hp))
	got := fn()
	gp, ga := got.RawWords()
	ep, ea := scm.NewString(in).RawWords()
	if gp != inPtr || ga != ea {
		t.Fatalf("bridge mismatch got=%#x/%#x exp=%#x/%#x", gp, ga, ep, ea)
	}
}
