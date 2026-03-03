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
package scm

import (
	"os"
	"testing"
	"unsafe"
)

func requireTmpDisasm(t *testing.T) {
	if os.Getenv("MEMCP_TMP_DISASM") != "1" {
		t.Skip("set MEMCP_TMP_DISASM=1 to run temporary disassembly tests")
	}
}

func TestTmpDisasmPlus(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Body: NewSlice([]Scmer{
			NewSymbol("+"),
			NewInt(3),
			NewInt(4),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Fatal("jitCompileProc returned no code")
	}
	out := "/tmp/jit_plus.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmConcatConst(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Body: NewSlice([]Scmer{
			NewSymbol("concat"),
			NewString("ab"),
			NewString("cd"),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Skip("jitCompileProc returned no code (expected for constant-folded concat)")
	}
	out := "/tmp/jit_concat_const.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmPlusStringConst(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Body: NewSlice([]Scmer{
			NewSymbol("+"),
			NewString("ab"),
			NewString("cd"),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Skip("jitCompileProc returned no code")
	}
	out := "/tmp/jit_plus_string_const.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmPlusStringParamConst(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Params: NewSlice([]Scmer{NewSymbol("x")}),
		Body: NewSlice([]Scmer{
			NewSymbol("+"),
			NewSymbol("x"),
			NewString("cd"),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Skip("jitCompileProc returned no code")
	}
	out := "/tmp/jit_plus_string_param_const.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmDeclPlusConstStrings(t *testing.T) {
	requireTmpDisasm(t)
	decl := declarations["+"]
	if decl == nil || decl.JITEmit == nil {
		t.Fatal("declaration + has no JIT emitter")
	}

	codeBuf := make([]byte, 4096)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	freeRegs := uint64((1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
		(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	ctx := &JITContext{
		W:         w,
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: RegR12,
	}

	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	args := []JITValueDesc{
		{Loc: LocImm, Type: tagString, Imm: NewString("a")},
		{Loc: LocImm, Type: tagString, Imm: NewString("b")},
	}

	start := uintptr(w.Ptr)
	desc := decl.JITEmit(ctx, args, result)
	emitted := uintptr(w.Ptr) - start
	t.Logf("decl emitter emitted %d bytes before return materialization", emitted)

	if desc.Loc == LocImm {
		ptr, aux := desc.Imm.RawWords()
		w.EmitMovRegImm64(RegRAX, uint64(ptr))
		w.EmitMovRegImm64(RegRBX, aux)
	} else {
		ctx.EmitMovPairToResult(&desc, &result)
	}
	w.EmitByte(0xC3)
	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]
	out := "/tmp/jit_decl_plus_const_strings.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmStringQConst(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Body: NewSlice([]Scmer{
			NewSymbol("string?"),
			NewString("ab"),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Fatal("jitCompileProc returned no code")
	}
	out := "/tmp/jit_stringq_const.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmStringQParam(t *testing.T) {
	requireTmpDisasm(t)
	proc := &Proc{
		Body: NewSlice([]Scmer{
			NewSymbol("string?"),
			NewNthLocalVar(0),
		}),
	}
	code := jitCompileProc(proc)
	if len(code) == 0 {
		t.Fatal("jitCompileProc returned no code")
	}
	out := "/tmp/jit_stringq_param.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}

func TestTmpDisasmDeclStringQKnownTypeUnknownValue(t *testing.T) {
	requireTmpDisasm(t)
	decl := declarations["string?"]
	if decl == nil || decl.JITEmit == nil {
		t.Fatal("declaration string? has no JIT emitter")
	}

	codeBuf := make([]byte, 4096)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	freeRegs := uint64((1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
		(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	ctx := &JITContext{
		W:         w,
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: RegR12,
	}

	// "known string, unknown content": value lives in regs, but static type is string.
	arg := JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: RegRDX, Reg2: RegRSI}
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := decl.JITEmit(ctx, []JITValueDesc{arg}, result)
	ctx.EmitMovPairToResult(&desc, &result)
	w.EmitByte(0xC3)
	w.ResolveFixupsFinal()

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	code := codeBuf[:codeLen]
	out := "/tmp/jit_stringq_known_type_unknown_value.bin"
	if err := os.WriteFile(out, code, 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote %d bytes to %s", len(code), out)
}
