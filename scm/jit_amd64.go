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
	}
	return nil
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
