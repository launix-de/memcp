//go:build amd64

/*
Copyright (C) 2024  Carl-Philip Hänsch

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

import "unsafe"

// TODO: create this file for other architectures, too

// all code snippets fill rax+rbx with the return value
func jitReturnLiteral(value Scmer) []byte {
	code := []byte{
		0x48, 0xB8, 7, 0, 0, 0, 0, 0, 0, 0, // mov rax, 7
		0x48, 0xBB, 7, 0, 0, 0, 0, 0, 0, 0, // mov rbx, 7
		0xC3,
	}
	// insert the literal into the immediate values
	*(*unsafe.Pointer)(unsafe.Pointer(&code[2])) = *(*unsafe.Pointer)(unsafe.Pointer(&value))
	*(*unsafe.Pointer)(unsafe.Pointer(&code[12])) = *((*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(&value), 8)))
	return code
}

func jitNthArgument(idx int) []byte { // up to 16 params
	var code []byte
	if idx > 0 {
		code = append(code, 0x48, 0x83, 0xC0, byte(idx*16)) // add rax, 16*idx
	}
	code = append(code,
		0x48, 0x8b, 0x08, // mov rcx, [rax]
		0x48, 0x8b, 0x58, 0x08, // mov rbx, [rax+8]
		0x48, 0x89, 0xc8, // mov rax, rcx
		0xC3, // ret
	)
	return code
}

// jitCompileProc pattern-matches a Proc body and returns amd64 machine code or nil.
//
// Native code emission is limited to frameless patterns (literal returns, param
// returns) because Go's stack walker cannot handle invisible JIT frames:
//   - CALL from JIT into Go causes "unexpected return pc" (Go doesn't know JIT PCs)
//   - JMP with JIT frame left on stack confuses frame-size-based stack walking
//   - No-frame approach has no safe memory for the variadic arg slice
//
// Function calls fall through to OptimizeProcToSerialFunction (Go closure).
func jitCompileProc(proc *Proc) []byte {
	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	switch body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return jitReturnLiteral(body)
	case tagNthLocalVar:
		return jitNthArgument(int(body.NthLocalVar()))
	case tagSlice:
		// Function call: (fn arg0 arg1 ...)
		return jitCompileCall(body.Slice())
	}
	return nil
}

// jitCompileCall compiles a function call expression using Declaration.JITEmit.
func jitCompileCall(call []Scmer) []byte {
	if len(call) < 1 {
		return nil
	}
	// Look up the Declaration for the head
	decl := DeclarationForValue(call[0])
	if decl == nil || decl.JITEmit == nil {
		return nil
	}

	// Allocate a scratch buffer for code generation
	buf := make([]byte, 4096)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&buf[0]),
		Start: unsafe.Pointer(&buf[0]),
		End:   unsafe.Pointer(&buf[len(buf)-1]),
	}

	args := call[1:]

	// Emit stack frame prologue (required by Go calling convention)
	// PUSH RBP; MOV RBP, RSP; SUB RSP, frameSize
	frameSize := byte(0x10)                  // 16 bytes for saving RAX (args slice ptr)
	w.emitByte(0x55)                         // PUSH RBP
	w.emitBytes(0x48, 0x89, 0xE5)            // MOV RBP, RSP
	w.emitBytes(0x48, 0x83, 0xEC, frameSize) // SUB RSP, frameSize

	// Save RAX (args slice pointer) on stack before clobbering it
	// MOV [RSP], RAX
	w.emitBytes(0x48, 0x89, 0x04, 0x24) // MOV [RSP], RAX

	// Available GPRs (excluding RAX/RBX=return, RSP, RBP, R14=g)
	ctx := &JITContext{
		W: w,
		FreeRegs: (1 << RegRCX) | (1 << RegRDX) | (1 << RegRSI) | (1 << RegRDI) |
			(1 << RegR8) | (1 << RegR9) | (1 << RegR10) | (1 << RegR11) |
			(1 << RegR12) | (1 << RegR13) | (1 << RegR15),
	}

	// Load each argument from the Scmer slice into register pairs
	// Use RSP-saved RAX as base for arg loads
	descs := make([]JITValueDesc, len(args))
	for i := range args {
		ptrReg := ctx.AllocReg()
		auxReg := ctx.AllocReg()
		// MOV ptrReg, [RAX + i*16]     (Scmer.ptr)
		w.emitMovRegMem(ptrReg, RegRAX, int32(i*16))
		// MOV auxReg, [RAX + i*16 + 8] (Scmer.aux)
		w.emitMovRegMem(auxReg, RegRAX, int32(i*16+8))
		descs[i] = JITValueDesc{Type: JITTypeUnknown, Loc: LocRegPair, Reg: ptrReg, Reg2: auxReg}
	}

	// Result goes into RAX (ptr) + RBX (aux) — the Go return registers
	resultDesc := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	decl.JITEmit(ctx, args, descs, resultDesc)

	// Emit stack frame epilogue + RET
	w.emitBytes(0x48, 0x83, 0xC4, frameSize) // ADD RSP, frameSize
	w.emitByte(0x5D)                         // POP RBP
	w.emitByte(0xC3)                         // RET

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	return buf[:codeLen]
}

func jitStackFrame(size uint8) []byte {
	return []byte{
		0x55,             //push   %rbp
		0x48, 0x89, 0xe5, //mov    %rsp,%rbp
		0x48, 0x83, 0xec, size, //sub    $0x10,%rsp
		// TODO: inner code
		// TODO: getter/setter mov    %rax,0x20(%rsp)
		0x48, 0x83, 0xc4, size, //add    $0x10,%rsp
		0x5d, //pop    %rbp
		0xc3, //ret
	}
}

/* TODO: peephole optimizer:
- remove argument checks (test rbx,rbx 48 85 db 76 xx)
- shorten immediate values
- constant-fold operations
- inline functions
- jump to other functions
*/
