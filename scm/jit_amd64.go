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

/*
alu: int?
   0x000000000072f9c0 <+0>:	55                 	push   %rbp
   0x000000000072f9c1 <+1>:	48 89 e5           	mov    %rsp,%rbp
   0x000000000072f9c4 <+4>:	48 83 ec 10        	sub    $0x10,%rsp
   0x000000000072f9c8 <+8>:	48 89 44 24 20     	mov    %rax,0x20(%rsp)
   0x000000000072f9cd <+13>:	48 85 db           	test   %rbx,%rbx
   0x000000000072f9d0 <+16>:	76 28              	jbe    0x72f9fa <github.com/launix-de/memcp/scm.init_alu.func1+58>
   0x000000000072f9d2 <+18>:	48 8d 0d 47 4e 07 00	lea    0x74e47(%rip),%rcx        # 0x7a4820 == descriptor for int64
   0x000000000072f9d9 <+25>:	48 39 08           	cmp    %rcx,(%rax)
   0x000000000072f9dc <+28>:	0f 94 c1           	sete   %cl
   0x000000000072f9df <+31>:	0f b6 c9           	movzbl %cl,%ecx
   0x000000000072f9e2 <+34>:	48 8d 15 f7 0c 3f 00	lea    0x3f0cf7(%rip),%rdx        # 0xb206e0 <runtime.staticuint64s>
   0x000000000072f9e9 <+41>:	48 8d 1c ca        	lea    (%rdx,%rcx,8),%rbx => lookup table true/false (+0/1 * 8)
   0x000000000072f9ed <+45>:	48 8d 05 6c 50 07 00	lea    0x7506c(%rip),%rax        # 0x7a4a60 == descriptor for bool
   0x000000000072f9f4 <+52>:	48 83 c4 10        	add    $0x10,%rsp
   0x000000000072f9f8 <+56>:	5d                 	pop    %rbp
   0x000000000072f9f9 <+57>:	c3                 	ret
   0x000000000072f9fa <+58>:	31 c0              	xor    %eax,%eax
   0x000000000072f9fc <+60>:	48 89 c1           	mov    %rax,%rcx
   0x000000000072f9ff <+63>:	90                 	nop
   0x000000000072fa00 <+64>:	e8 3b 20 d4 ff     	call   0x471a40 <runtime.panicIndex>
   0x000000000072fa05 <+69>:	90                 	nop

optimized:
   48 8d 0d 47 4e 07 00	lea    0x74e47(%rip),%rcx        # 0x7a4820 == descriptor for int64
   48 39 08           	cmp    %rcx,(%rax)
   0f 94 c1           	sete   %cl
   0f b6 c8           	movzbl %cl,%eax
   48 8d 05 6c 50 07 00	lea    0x7506c(%rip),%rax        # 0x7a4a60 == descriptor for bool
   c3                 	ret

*/

/* alu: add

   0x000000000072faa0 <+0>:	49 3b 66 10        	cmp    0x10(%r14),%rsp -> check stack frame size
   0x000000000072faa4 <+4>:	76 7b              	jbe    0x72fb21 <github.com/launix-de/memcp/scm.init_alu.func3+129>
   0x000000000072faa6 <+6>:	55                 	push   %rbp
   0x000000000072faa7 <+7>:	48 89 e5           	mov    %rsp,%rbp
   0x000000000072faaa <+10>:	48 83 ec 28        	sub    $0x28,%rsp
   0x000000000072faae <+14>:	48 89 44 24 38     	mov    %rax,0x38(%rsp)
   0x000000000072fab3 <+19>:	0f 57 c0           	xorps  %xmm0,%xmm0 -> v = float64(0)
   0x000000000072fab6 <+22>:	eb 37              	jmp    0x72faef <github.com/launix-de/memcp/scm.init_alu.func3+79> -> while check

   // loop head
   0x000000000072fab8 <+24>:	48 89 5c 24 18     	mov    %rbx,0x18(%rsp) -> save vars to stack
   0x000000000072fabd <+29>:	48 89 44 24 20     	mov    %rax,0x20(%rsp)
   0x000000000072fac2 <+34>:	f2 0f 11 44 24 10  	movsd  %xmm0,0x10(%rsp)
   0x000000000072fac8 <+40>:	48 8b 58 08        	mov    0x8(%rax),%rbx -> load value to rb
   0x000000000072facc <+44>:	48 89 c8           	mov    %rcx,%rax -> type descriptor to rax
   0x000000000072facf <+47>:	e8 ac 05 fd ff     	call   0x700080 <github.com/launix-de/memcp/scm.ToFloat>
   0x000000000072fad4 <+52>:	48 8b 44 24 20     	mov    0x20(%rsp),%rax -> restore array
   0x000000000072fad9 <+57>:	48 83 c0 10        	add    $0x10,%rax -> move on 16 bytes in array
   0x000000000072fadd <+61>:	48 8b 5c 24 18     	mov    0x18(%rsp),%rbx
   0x000000000072fae2 <+66>:	48 ff cb           	dec    %rbx -> recrease slice len
   0x000000000072fae5 <+69>:	f2 0f 10 4c 24 10  	movsd  0x10(%rsp),%xmm1
   0x000000000072faeb <+75>:	f2 0f 58 c1        	addsd  %xmm1,%xmm0 -> add

   // while check
   0x000000000072faef <+79>:	48 85 db           	test   %rbx,%rbx -> check a length of a
   0x000000000072faf2 <+82>:	7e 12              	jle    0x72fb06 <github.com/launix-de/memcp/scm.init_alu.func3+102> -> go out
   0x000000000072faf4 <+84>:	48 8b 08           	mov    (%rax),%rcx -> type descriptor
   0x000000000072faf7 <+87>:	48 85 c9           	test   %rcx,%rcx -> i == nil?
   0x000000000072fafa <+90>:	75 bc              	jne    0x72fab8 <github.com/launix-de/memcp/scm.init_alu.func3+24> -> loop head
   0x000000000072fafc <+92>:	31 c0              	xor    %eax,%eax -> return nil
   0x000000000072fafe <+94>:	31 db              	xor    %ebx,%ebx
   0x000000000072fb00 <+96>:	48 83 c4 28        	add    $0x28,%rsp
   0x000000000072fb04 <+100>:	5d                 	pop    %rbp
   0x000000000072fb05 <+101>:	c3                 	ret

   // go out
   0x000000000072fb06 <+102>:	66 48 0f 7e c0     	movq   %xmm0,%rax -> float64 to rax (convert)
   0x000000000072fb0b <+107>:	e8 90 e4 cd ff     	call   0x40dfa0 <runtime.convT64> -> ???
   0x000000000072fb10 <+112>:	48 89 c3           	mov    %rax,%rbx -> value
   0x000000000072fb13 <+115>:	48 8d 05 06 4f 07 00	lea    0x74f06(%rip),%rax        # 0x7a4a20 -> descriptor
   0x000000000072fb1a <+122>:	48 83 c4 28        	add    $0x28,%rsp
   0x000000000072fb1e <+126>:	5d                 	pop    %rbp
   0x000000000072fb1f <+127>:	90                 	nop
   0x000000000072fb20 <+128>:	c3                 	ret
   0x000000000072fb21 <+129>:	48 89 44 24 08     	mov    %rax,0x8(%rsp)
   0x000000000072fb26 <+134>:	48 89 5c 24 10     	mov    %rbx,0x10(%rsp)
   0x000000000072fb2b <+139>:	48 89 4c 24 18     	mov    %rcx,0x18(%rsp)
   0x000000000072fb30 <+144>:	e8 6b fc d3 ff     	call   0x46f7a0 <runtime.morestack_noctxt>
   0x000000000072fb35 <+149>:	48 8b 44 24 08     	mov    0x8(%rsp),%rax
   0x000000000072fb3a <+154>:	48 8b 5c 24 10     	mov    0x10(%rsp),%rbx
   0x000000000072fb3f <+159>:	48 8b 4c 24 18     	mov    0x18(%rsp),%rcx
   0x000000000072fb44 <+164>:	e9 57 ff ff ff     	jmp    0x72faa0 <github.com/launix-de/memcp/scm.init_alu.func3>

// optimized:

   0x000000000072faa6 <+6>:	55                 	push   %rbp
   0x000000000072faa7 <+7>:	48 89 e5           	mov    %rsp,%rbp
   0x000000000072faaa <+10>:	48 83 ec 28        	sub    $0x28,%rsp
   0x000000000072faae <+14>:	48 89 44 24 38     	mov    %rax,0x38(%rsp)
   0x000000000072fab3 <+19>:	0f 57 c0           	xorps  %xmm0,%xmm0 -> v = float64(0)

   // while check
   0x000000000072faef <+79>:	48 85 db           	test   %rbx,%rbx -> check a length of a
   0x000000000072faf2 <+82>:	7e 12              	jle    0x72fb06 <github.com/launix-de/memcp/scm.init_alu.func3+102> -> go out
   0x000000000072faf4 <+84>:	48 8b 08           	mov    (%rax),%rcx -> type descriptor
   0x000000000072faf7 <+87>:	48 85 c9           	test   %rcx,%rcx -> i == nil?
   0x000000000072fafa <+90>:	75x xx              	je    0x72fab8 <github.com/launix-de/memcp/scm.init_alu.func3+24> -> return nil

   // loop head
   0x000000000072fab8 <+24>:	48 89 5c 24 18     	mov    %rbx,0x18(%rsp) -> save vars to stack (len)
   0x000000000072fabd <+29>:	48 89 44 24 20     	mov    %rax,0x20(%rsp) (array)
   0x000000000072fac2 <+34>:	f2 0f 11 44 24 10  	movsd  %xmm0,0x10(%rsp) (sum)
   0x000000000072fac8 <+40>:	48 8b 58 08        	mov    0x8(%rax),%rbx -> load value to rb
   0x000000000072facc <+44>:	48 89 c8           	mov    %rcx,%rax -> type descriptor to rax
   0x000000000072facf <+47>:	e8 ac 05 fd ff     	call   0x700080 <github.com/launix-de/memcp/scm.ToFloat> (result in xmm0)
   0x000000000072fad4 <+52>:	48 8b 44 24 20     	mov    0x20(%rsp),%rax -> restore array
   0x000000000072fad9 <+57>:	48 83 c0 10        	add    $0x10,%rax -> move on 16 bytes in array
   0x000000000072fadd <+61>:	48 8b 5c 24 18     	mov    0x18(%rsp),%rbx -> restore len
   0x000000000072fae2 <+66>:	48 ff cb           	dec    %rbx -> decrease slice len
   0x000000000072fae5 <+69>:	f2 0f 10 4c 24 10  	movsd  0x10(%rsp),%xmm1 -> load sum to xmm1
   0x000000000072faeb <+75>:	f2 0f 58 c1        	addsd  %xmm1,%xmm0 -> add
   jmp <+79>

   // go out
   0x000000000072fb06 <+102>:	66 48 0f 7e c0     	movq   %xmm0,%rax -> float64 to rax (convert)
   0x000000000072fb0b <+107>:	e8 90 e4 cd ff     	call   0x40dfa0 <runtime.convT64> -> ???
   0x000000000072fb10 <+112>:	48 89 c3           	mov    %rax,%rbx -> value
   0x000000000072fb13 <+115>:	48 8d 05 06 4f 07 00	lea    0x74f06(%rip),%rax        # 0x7a4a20 -> descriptor
   0x000000000072fb1a <+122>:	48 83 c4 28        	add    $0x28,%rsp
   0x000000000072fb1e <+126>:	5d                 	pop    %rbp
   0x000000000072fb1f <+127>:	90                 	nop
   0x000000000072fb20 <+128>:	c3                 	ret
   0x000000000072fb21 <+129>:	48 89 44 24 08     	mov    %rax,0x8(%rsp)
   0x000000000072fb26 <+134>:	48 89 5c 24 10     	mov    %rbx,0x10(%rsp)
   0x000000000072fb2b <+139>:	48 89 4c 24 18     	mov    %rcx,0x18(%rsp)
   0x000000000072fb30 <+144>:	e8 6b fc d3 ff     	call   0x46f7a0 <runtime.morestack_noctxt>
   0x000000000072fb35 <+149>:	48 8b 44 24 08     	mov    0x8(%rsp),%rax
   0x000000000072fb3a <+154>:	48 8b 5c 24 10     	mov    0x10(%rsp),%rbx
   0x000000000072fb3f <+159>:	48 8b 4c 24 18     	mov    0x18(%rsp),%rcx
   0x000000000072fb44 <+164>:	e9 57 ff ff ff     	jmp    0x72faa0 <github.com/launix-de/memcp/scm.init_alu.func3>

   // return nil
   0x000000000072fafc <+92>:	31 c0              	xor    %eax,%eax -> return nil
   0x000000000072fafe <+94>:	31 db              	xor    %ebx,%ebx
   0x000000000072fb00 <+96>:	48 83 c4 28        	add    $0x28,%rsp
   0x000000000072fb04 <+100>:	5d                 	pop    %rbp
   0x000000000072fb05 <+101>:	c3                 	ret
*/
