/*
Copyright (C) 2024-2026  Carl-Philip Haensch
*/
package storage

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

func TestStorageStringJITDebugDisasm(t *testing.T) {
	if runtime.GOARCH != "amd64" { t.Skip("amd64") }
	values := []scm.Scmer{scm.NewString("hello"), scm.NewString("world"), scm.NewString("foo")}
	s := buildStorageString(values)
	t.Logf("nodict=%v dict_len=%d", s.nodict, len(s.dictionary))

	codeBuf := make([]byte, 65536)
	w := &scm.JITWriter{Ptr: unsafe.Pointer(&codeBuf[0]), Start: unsafe.Pointer(&codeBuf[0]), End: unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256)}
	freeRegs := uint64((1 << uint(scm.RegRAX)) | (1 << uint(scm.RegRBX)) | (1 << uint(scm.RegRCX)) | (1 << uint(scm.RegRDX)) | (1 << uint(scm.RegRSI)) | (1 << uint(scm.RegRDI)) | (1 << uint(scm.RegR8)) | (1 << uint(scm.RegR9)) | (1 << uint(scm.RegR10)) | (1 << uint(scm.RegR12)) | (1 << uint(scm.RegR13)) | (1 << uint(scm.RegR15)))
	ctx := &scm.JITContext{W: w, FreeRegs: freeRegs, AllRegs: freeRegs}
	ctx.FreeRegs &^= 1 << uint(scm.RegR15)
	ctx.AllRegs &^= 1 << uint(scm.RegR15)
	ctx.FreeRegs &^= 1 << uint(scm.RegR12)
	ctx.AllRegs &^= 1 << uint(scm.RegR12)
	idxReg := ctx.AllocReg(); w.EmitMovRegReg(idxReg, scm.RegRAX)
	idx := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxReg}
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(extractDataPtr(s))}
	result := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: scm.RegRAX, Reg2: scm.RegRBX}
	desc := s.JITEmit(ctx, thisptr, idx, result)
	ctx.EmitMovPairToResult(&desc, &result)
	w.EmitByte(0xC3); w.ResolveFixupsFinal()
	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start)); code := codeBuf[:codeLen]
	_ = os.WriteFile("/tmp/jit_string.bin", code, 0644)

	pageSize := syscall.Getpagesize(); n := (len(code) + pageSize - 1) &^ (pageSize - 1)
	b, _ := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	copy(b, code); _ = syscall.Mprotect(b, syscall.PROT_READ|syscall.PROT_EXEC); defer syscall.Munmap(b)
	type fh struct{ fnptr *byte }; hdr := &fh{fnptr: &b[0]}; hp := unsafe.Pointer(hdr)
	jitFn := *(*func(int64) scm.Scmer)(unsafe.Pointer(&hp))
	for i:=0;i<3;i++ { got:=jitFn(int64(i)); exp:=s.GetValue(uint32(i)); gp,ga:=got.RawWords(); ep,ea:=exp.RawWords(); t.Logf("i=%d got=%#x/%#x exp=%#x/%#x",i,gp,ga,ep,ea)}
}
