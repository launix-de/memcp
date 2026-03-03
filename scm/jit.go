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
		// fallback: keep a serializable/executable JIT descriptor instead of
		// returning an unserializable native closure.
		fn := OptimizeProcToSerialFunction(v)
		return NewJIT(&JITEntryPoint{
			Native: fn,
			Proc:   *proc,
			Arch:   runtime.GOARCH,
		})

	default:
		panic(fmt.Sprintf("jit: cannot compile %v (tag %d)", v, tag))
	}
}
