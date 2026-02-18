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
	"regexp"
	"runtime"
	"testing"
)

func stackGrow(depth int, v Scmer) {
	var scratch [64]byte
	scratch[0] = byte(depth)
	if depth == 0 {
		runtime.GC()
		runtime.KeepAlive(scratch)
		return
	}
	stackGrow(depth-1, v)
	runtime.KeepAlive(v)
	runtime.KeepAlive(scratch)
}

func TestScmerDoesNotCrashGCDuringStackGrowth(t *testing.T) {
	// Tags with ptr=nil (no pointer stored)
	stackGrow(2000, NewNil())
	stackGrow(2000, NewBool(true))
	stackGrow(2000, NewBool(false))
	stackGrow(2000, NewDate(1718451045))
	stackGrow(2000, NewNthLocalVar(7))

	// Tags with sentinel pointers
	stackGrow(2000, NewInt(1))
	stackGrow(2000, NewInt(-9999999))
	stackGrow(2000, NewFloat(3.14159))
	stackGrow(2000, NewFloat(0.0))

	// Tags with heap pointers to string backing store
	stackGrow(2000, NewString("hello world"))
	stackGrow(2000, NewString(""))
	stackGrow(2000, NewSymbol("my-symbol"))
	stackGrow(2000, NewSymbol(""))

	// Tags with heap pointers to slice backing arrays
	stackGrow(2000, NewSlice([]Scmer{NewInt(1), NewString("x"), NewFloat(2.5)}))
	stackGrow(2000, NewSlice([]Scmer{}))
	stackGrow(2000, NewVector([]float64{1.0, 2.0, 3.0}))
	stackGrow(2000, NewVector([]float64{}))

	// Tags with heap-allocated typed pointers
	stackGrow(2000, NewFunc(func(a ...Scmer) Scmer { return a[0] }))
	stackGrow(2000, NewFuncEnv(func(e *Env, a ...Scmer) Scmer { return a[0] }))
	stackGrow(2000, NewProcStruct(Proc{
		Params: NewSlice([]Scmer{NewSymbol("x")}),
		Body:   NewSymbol("x"),
		En:     &Globalenv,
	}))
	stackGrow(2000, NewProc(nil))
	stackGrow(2000, NewSourceInfo(SourceInfo{}))
	stackGrow(2000, NewAny("arbitrary value"))
	stackGrow(2000, NewRegex(regexp.MustCompile(`\d+`)))
	stackGrow(2000, NewRegex(nil))

	// FastDict with heap pointer
	fd := NewFastDictValue(4)
	fd.Set(NewString("key"), NewInt(42), nil)
	stackGrow(2000, NewFastDict(fd))
	stackGrow(2000, NewFastDict(nil))

	// Nested structures: slice containing strings, funcs, procs
	nested := NewSlice([]Scmer{
		NewString("nested"),
		NewSlice([]Scmer{NewInt(1), NewInt(2)}),
		NewFunc(func(a ...Scmer) Scmer { return NewNil() }),
		NewFastDict(fd),
	})
	stackGrow(2000, nested)
}
