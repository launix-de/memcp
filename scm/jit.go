/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

var JITLog bool

// execBuf is a small wrapper for mmap'd memory
type execBuf struct {
	ptr unsafe.Pointer
	n   int // size
}

func allocExec(size int) (*execBuf, error) {
	page := syscall.Getpagesize()
	n := (size + page - 1) & ^(page - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, err
	}
	return &execBuf{ptr: unsafe.Pointer(&b[0]), n: n}, nil
}

func (e *execBuf) makeRX() error {
	// change to PROT_READ|PROT_EXEC
	data := (*[1 << 30]byte)(e.ptr)[:e.n:e.n]
	return syscall.Mprotect(data, syscall.PROT_READ|syscall.PROT_EXEC)
}

func init_jit() {
	DeclareTitle("JIT Compilation")

	Declare(&Globalenv, &Declaration{
		"jit", "compiles a lambda to optimized native code; passes through already compiled functions",
		1, 1,
		[]DeclarationParameter{
			{"fn", "any", "the function to compile", nil},
		}, "any",
		jitCompile,
		false, false, nil, nil, // not pure because it allocates executable memory
	})
	Declare(&Globalenv, &Declaration{
		"jit?", "tells whether a value is a JIT-compiled function descriptor",
		1, 1,
		[]DeclarationParameter{
			{"value", "any", "value to inspect", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagJIT)
		},
		true, false, nil, func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(100))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 100)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps3)
			return result
		},
	})
}

// jitCompile compiles a Proc to a native function (tagFunc)
// Already compiled functions (tagFunc, tagFuncEnv) are passed through unchanged
func jitCompile(a ...Scmer) Scmer {
	if len(a) != 1 {
		panic("jit: expects exactly 1 argument")
	}

	v := a[0]
	tag := v.GetTag()
	if JITLog {
		fmt.Printf("JIT: compile %s\n", SerializeToString(v, &Globalenv))
	}

	switch tag {
	case tagJIT:
		// Already compiled
		return v
	case tagFunc:
		// Already a native function - pass through
		return v

	case tagFuncEnv:
		// Already a native function with environment - pass through
		return v

	case tagProc:
		// Lambda/procedure - attempt native compilation first
		proc := v.Proc()
		if code, roots := jitCompileProcWithRoots(proc); code != nil {
			if JITLog {
				fmt.Printf("%X\n", code)
			}
			buf, err := allocExec(len(code))
			if err == nil {
				dst := (*[1 << 30]byte)(buf.ptr)[:len(code):len(code)]
				copy(dst, code)
				if err2 := buf.makeRX(); err2 == nil {
					fn2 := unsafe.Pointer(&struct{ *byte }{&dst[0]})
					nativeFn := *(*func(...Scmer) Scmer)(unsafe.Pointer(&fn2))
					return NewJIT(&JITEntryPoint{
						Native:     nativeFn,
						ConstRoots: roots,
						Proc:       *proc,
						Arch:       runtime.GOARCH,
					})
				}
				syscall.Munmap((*[1 << 30]byte)(buf.ptr)[:buf.n:buf.n])
			}
		}
		if JITLog {
			fmt.Println("<fallback>")
		}
		// Fallback returns the original lambda/procedure unchanged.
		return v

	default:
		panic(fmt.Sprintf("jit: cannot compile %v (tag %d)", v, tag))
	}
}
