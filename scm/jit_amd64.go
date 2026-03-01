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
		0x48, 0x8b, 0x08,       // mov rcx, [rax]
		0x48, 0x8b, 0x58, 0x08, // mov rbx, [rax+8]
		0x48, 0x89, 0xc8,       // mov rax, rcx
		0xC3,                    // ret
	)
	return code
}

// jitCompileProc compiles a Proc body to amd64 machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	// Trivial patterns: literal returns, parameter passthrough
	switch body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return jitReturnLiteral(body)
	case tagNthLocalVar:
		return jitNthArgument(int(body.NthLocalVar()))
	}
	// Expression compilation via JITEmit
	if body.GetTag() == tagSlice {
		return jitCompileExprBody(body)
	}
	return nil
}

// jitCompileExprBody compiles a Scheme expression body to machine code
// using Declaration.JITEmit callbacks. Returns nil if any sub-expression
// is not JIT-compilable.
func jitCompileExprBody(body Scmer) (code []byte) {
	defer func() {
		if r := recover(); r != nil {
			code = nil
		}
	}()

	// Allocate temp buffer for code emission
	codeBuf := make([]byte, 16384)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	// Free registers: all GPRs except RAX (result ptr), RBX (result aux),
	// RSP, RBP, R11 (scratch), R12 (slice base)
	ctx := &JITContext{
		W: w,
		FreeRegs: (1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
			(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
			(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
			(1 << uint(RegR13)) | (1 << uint(RegR14)) | (1 << uint(RegR15)),
	}

	// Emit: MOV R12, RAX (save slice base pointer)
	w.emitMovRegReg(RegR12, RegRAX)

	// Compile body, place result into RAX+RBX (Scmer return registers)
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := jitCompileExpr(ctx, body, RegR12, result)

	// If result came back as LocImm, materialize into RAX+RBX
	if desc.Loc == LocImm {
		switch desc.Imm.GetTag() {
		case tagBool:
			w.EmitReturnBool(desc)
		case tagInt:
			w.EmitReturnInt(desc)
		case tagFloat:
			w.EmitReturnFloat(desc)
		case tagNil:
			w.EmitReturnNil()
		default:
			return nil
		}
	} else {
		// Result already materialized in RAX+RBX by the emitter
		w.emitByte(0xC3) // RET
	}

	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	return codeBuf[:codeLen]
}

// jitCompileExpr recursively compiles a Scheme expression to machine code.
// sliceBase is the GPR holding the variadic args slice pointer.
// result tells the emitter where to place the output.
// Panics on unsupported expressions (caught by jitCompileExprBody).
func jitCompileExpr(ctx *JITContext, expr Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	switch expr.GetTag() {
	case tagNil:
		return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: expr}
	case tagBool:
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: expr}
	case tagInt:
		return JITValueDesc{Loc: LocImm, Type: tagInt, Imm: expr}
	case tagFloat:
		return JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: expr}
	case tagString:
		return JITValueDesc{Loc: LocImm, Type: tagString, Imm: expr}
	case tagNthLocalVar:
		// Load parameter from args slice: ptr at [base+i*16], aux at [base+i*16+8]
		idx := int(expr.NthLocalVar())
		ptrReg := ctx.AllocReg()
		auxReg := ctx.AllocReg()
		ctx.W.emitMovRegMem(ptrReg, sliceBase, int32(idx*16))
		ctx.W.emitMovRegMem(auxReg, sliceBase, int32(idx*16+8))
		return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
	case tagSlice:
		list := expr.Slice()
		if len(list) == 0 {
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
		}
		// Resolve operator
		if !list[0].IsSymbol() {
			panic("jit: non-symbol in call position")
		}
		name := string(list[0].Symbol())
		decl, ok := declarations[name]
		if !ok || decl.JITEmit == nil {
			panic("jit: no JITEmit for " + name)
		}
		// Compile arguments (intermediate results use LocAny)
		args := make([]JITValueDesc, len(list)-1)
		for i := 1; i < len(list); i++ {
			args[i-1] = jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
		}
		// Call the JITEmit callback
		return decl.JITEmit(ctx, args, result)
	default:
		panic("jit: unsupported expression tag")
	}
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
