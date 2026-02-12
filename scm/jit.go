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
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"reflect"
	"runtime"
	"syscall"
	"unsafe"
)

/*
memcp JIT compiler
------------------
 - compiles faster JIT
 - composes functions from primitives
 - analyzes go functions and removes type checks, stack frames, jumps and so on

there are multiple approaches to speed up memcp with JIT:
 - the most promising is to instead of calling Apply() compose a func, so we can save memory allocations
 - a more advanced approach is to fuse the scan operator loop (getValue of a fixed Storage) with the parameter building and the calling of the map/aggregate function
 - helper functions like ToBool, ToFloat, ToString can be specialized to certain input parameters

 - either by parsing the assembly and peepholing it (my favourite)
 - or by reading the AST and regenerating the code

TODO:
 - create a type descriptor map for the most common types (int64, string, bool etc.)
 - helper functions to compose code (no-copy ideally) - platform specific

*/

// cache for specializations: key -> mmap pointer
func myAdd(a ...Scmer) Scmer {
	// disassemble /r 'github.com/launix-de/memcp/scm.myAdd'
	v := a[0]
	return v
}

func RunJitTest() {
	return // dont output that in production
	// disassemble /r 'github.com/launix-de/memcp/scm.RunJitTest'
	fmt.Println("run JIT test")
	fn2 := OptimizeForValues(myAdd, []int{2}, []Scmer{NewInt(4)})
	fmt.Println("fn=", fn2)
	fmt.Println("result", fn2(NewInt(3), NewInt(7))) // should return 4
}

/*
typical func call:
mov 16, %rax // size for one Scmer
call <runtime.newobject>
mov 8, (%rax) // code for int
mov 2, 0x8(%rax) // integer value 2
mov [fn], %rdx // func
mov (%rdx),%r8 // get fn ptr
mov $0x1,%rbx // slice size
mov $0x1,%rcx // slice capacity
call   *%r8 // eax=sliceptr, ecx=slicesize, ebx=slicecapacity, edx=context

receiver:
test rbx,rbx // slicesize != 0
jbe error handler

// load slice values from %rax into rcx+rbx
mov    (%rax),%rax // first return value (type descriptor)
mov    0x8(%rax),%rbx // second return value (value)


*/

// optimize takes a runtime function value `fn` (of type func(Scmer...) Scmer),
// a constMask where constMask[i] > 0 means the i-th parameter is a known
// compile-time constant, and constValues providing those constant values in
// the same order. It returns a new function func(Scmer...) Scmer which will
// call into specialized machine code if we recognized the pattern and
// successfully emitted native code. Otherwise it will return a wrapper that
// calls the original Go function.
func OptimizeForValues(fn func(...Scmer) Scmer, constMask []int /* 0=unknown, 1=type, 2=value */, constValues []Scmer) func(...Scmer) Scmer {
	// quick validation
	allVariable := true
	for _, c := range constMask {
		if c > 0 {
			allVariable = false
		}
	}
	if allVariable {
		// nothing to optimize
		return fn
	}

	//hash := fmt.Sprint("%p", fn) // TODO: cache targetDecl

	// locate source file from runtime info
	pc := reflect.ValueOf(fn).Pointer()
	f := runtime.FuncForPC(pc)
	fmt.Println("func", f)
	if f == nil {
		return fn
	}
	file, _ := f.FileLine(pc) // TODO: find relative path
	fmt.Println("file", file)
	// file may include ":<line>" from FileLine; but runtime.FuncForPC gives file path
	// attempt to read file
	src, err := ioutil.ReadFile(file)
	//fmt.Println("src", string(src))
	if err != nil {
		// fallback: try to use the function's name to locate file in GOPATH
		return fn
	}

	// parse file and find the func declaration that contains the runtime line
	fs := token.NewFileSet()
	fileAst, err := parser.ParseFile(fs, file, src, parser.ParseComments)
	if err != nil {
		return func(args ...Scmer) Scmer { return fn(args...) }
	}

	// get line number
	_, line := f.FileLine(pc)
	fmt.Println("line", line)

	var targetDecl *ast.FuncDecl
	for _, d := range fileAst.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		start := fs.Position(fd.Pos()).Line
		end := fs.Position(fd.End()).Line
		if line >= start && line <= end {
			targetDecl = fd
			break
		}
	}
	fmt.Println("targetDecl", targetDecl)
	if targetDecl == nil {
		return fn
	}
	// TODO: store targetDecl with hash

	// grab the first param name
	firstParam := targetDecl.Type.Params.List[0]
	fmt.Println("param: ----------")
	ast.Print(nil, firstParam)

	// statements
	stmts := targetDecl.Body.List
	fmt.Println("stmts", stmts)

	for i, s := range stmts {
		fmt.Println("-----", i, "-----")
		ast.Print(nil, s)
	}
	// TODO: recurse over the AST
	// ast.AssignStmt.Rhs[] ersetzen
	// ast.IndexExpr{X: firstParam, Index: ast.BasicLit}
	// ast.BasicLit{Kind: token.Int, Value: strconv.ParseInt(Value, 0, 64)}
	// *ast.ReturnStmt.Results[0]
	if stmts[0].(*ast.AssignStmt).Rhs[0].(*ast.IndexExpr).X.(*ast.Ident).Obj.Decl == firstParam {
		fmt.Println("bingo")
	}

	//code := jitReturnLiteral(constValues[0]) // TODO: compose the real code
	code := jitNthArgument(1) // TODO: compose the real code

	// allocate executable buffer
	buf, err := allocExec(len(code))
	if err != nil {
		return fn
	}
	// copy code
	dst := (*[1 << 30]byte)(buf.ptr)[:len(code):len(code)]
	copy(dst, code)
	if err := buf.makeRX(); err != nil {
		// free
		syscall.Munmap((*[1 << 30]byte)(buf.ptr)[:buf.n:buf.n])
		return fn
	}

	fn2 := unsafe.Pointer(&struct{ *byte }{&dst[0]})
	return *(*func(...Scmer) Scmer)(unsafe.Pointer(&fn2)) // struct { fnptr, closure1, closure2, ... }
}

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
			{"fn", "procedure", "the function to compile", nil},
		}, "procedure",
		jitCompile,
		false, false, nil, // not pure because it allocates executable memory
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

	switch tag {
	case tagFunc:
		// Already a native function - pass through
		return v

	case tagFuncEnv:
		// Already a native function with environment - pass through
		return v

	case tagProc:
		// Lambda/procedure - compile it
		// Use OptimizeProcToSerialFunction as fallback/baseline
		fn := OptimizeProcToSerialFunction(v)
		return NewFunc(fn)

	default:
		panic(fmt.Sprintf("jit: cannot compile %v (tag %d)", v, tag))
	}
}
