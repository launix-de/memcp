/*
	Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; see <https://www.gnu.org/licenses/>.
*/
package main

import "strings"
import "testing"
import "go/types"
import "go/constant"
import "golang.org/x/tools/go/ssa"

func TestStructuralSliceConditionSpecializesHelper(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func unsupported(values []Scmer) Scmer
 func pick(values []Scmer) Scmer {
  if len(values)%2!=0 { return unsupported(values[:len(values)-1]) }
  return values[0]
 }
 func builtin(args ...Scmer) Scmer { return pick(args[1:]) }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if !strings.Contains(code, "if ((len(virtualArgs0)%2)!=0) {") {
		t.Fatalf("argument shape was not staged in emitter:\n%s", code)
	}
	if strings.Contains(code, "GoFuncAddr(pick)") || strings.Count(code, "GoFuncAddr(unsupported)") != 1 {
		t.Fatalf("the helper must inline both arms while retaining the unsupported leaf call:\n%s", code)
	}
	if !strings.Contains(code, "jitMaterializeVirtualGoSlice(ctx, virtualArgs") {
		t.Fatal("native arm lost the virtual slice boundary")
	}
}

func TestReachabilityIdentityPhiPreservesVirtualSlice(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 func pick(values []int, trim bool) int {
  if trim { values=values[:len(values)-1] }
  return len(values)
 }
`, "pick")
	proof := analyzeSSAReachability(fn, map[ssa.Value]*ssa.Const{fn.Params[1]: ssa.NewConst(constant.MakeBool(false), types.Typ[types.Bool])})
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			if phi, ok := instruction.(*ssa.Phi); ok && proof.blocks[block] {
				if proof.replacement(phi) != fn.Params[0] {
					t.Fatalf("slice identity was lost: %s", phi)
				}
			}
		}
	}
}

func TestWritableMethodUsesOrdinaryNilSpecialization(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 type Holder struct { Value Scmer }
 func pointer(value Scmer) *Holder
 func (h *Holder) Set(value Scmer, merge func(Scmer,Scmer) Scmer) {
  if merge!=nil { h.Value=merge(h.Value,value) } else { h.Value=value }
 }
 func builtin(args ...Scmer) Scmer {
  h:=pointer(args[0]); h.Set(args[1],nil); return args[1]
 }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	for _, forbidden := range []string{"GoFuncAddr((*Holder).Set)", "jitInvokeMergeCallback"} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("method bypassed nil specialization (%s):\n%s", forbidden, code)
		}
	}
}

func TestVariadicTailViewsPreserveDescriptorDirectory(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func tail(values []Scmer) Scmer {
  last := values[len(values)-1]
  values = values[:len(values)-1]
  if len(values) == 0 { return last }
  return values[0]
 }
 func builtin(args ...Scmer) Scmer { return tail(args[1:]) }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if strings.Contains(code, "GoFuncAddr(tail)") || strings.Contains(code, "jitMaterializeVirtualGoSlice") {
		t.Fatalf("arity-only views crossed a native slice boundary:\n%s", code)
	}
	if !strings.Contains(code, "variadic slice bound is not known during emission") || !strings.Contains(code, ".Imm.Int())]") {
		t.Fatalf("tail bounds were not retained as compiler values:\n%s", code)
	}
}

func TestKnownGoClosureNilComparison(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func call(value Scmer, fn func(Scmer) Scmer) Scmer {
  if fn != nil { return fn(value) }
  return value
 }
 func builtin(args ...Scmer) Scmer {
  value := args[0]
  return call(args[1], func(Scmer) Scmer { return value })
 }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if strings.Contains(code, "if .Loc") || strings.Contains(code, "GoFuncAddr(call)") {
		t.Fatalf("known Go closure was lost at helper boundary:\n%s", code)
	}
	if !strings.Contains(code, "Imm: NewBool(true)") {
		t.Fatal("closure's non-nil proof was not folded")
	}
}

func TestNewSliceUsesBorrowedSubview(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func NewSlice(values []Scmer) Scmer
 func builtin(args ...Scmer) Scmer { return NewSlice(args[1:]) }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if !strings.Contains(code, "Virtual: append([]JITValueDesc(nil), virtualArgs0...)") {
		t.Fatalf("constructor discarded its subview: %s", code)
	}
}

func TestHelperParameterNameDoesNotSelectRootArguments(t *testing.T) {
	for _, helper := range []string{
		"func head(args []Scmer) Scmer { return args[0] }",
		"func head(args []Scmer) Scmer { return NewInt(int64(len(args))) }",
	} {
		fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func NewInt(value int64) Scmer
 `+helper+`
 func builtin(args ...Scmer) Scmer { return head(args[1:]) }
`, "builtin")
		code, failure := generateClosure("builtin", fn, nil)
		if failure != "" {
			t.Fatal(failure)
		}
		if strings.Contains(code, "GoFuncAddr(head)") {
			t.Fatalf("helper was not inlined: %s", helper)
		}
		if strings.Contains(helper, "args[0]") && !strings.Contains(code, "= virtualArgs0[0]") {
			t.Fatalf("helper selected root arguments: %s", code)
		}
		if strings.Contains(helper, "len(args)") && strings.Contains(code, "int64(len(args))") {
			t.Fatalf("helper used root length: %s", code)
		}
	}
}

func TestClosureBindingWritesDifferFromPointeeWrites(t *testing.T) {
	for _, test := range []struct {
		body    string
		rebinds bool
	}{
		{`value:=args[0]; fn:=func() Scmer { return value }; return fn()`, false},
		{`value:=args[0]; fn:=func() Scmer { value=args[1]; return value }; return fn()`, true},
		{`var values [1]Scmer; fn:=func() Scmer { values[0]=args[0]; return values[0] }; return fn()`, false},
	} {
		fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func builtin(args ...Scmer) Scmer { `+test.body+` }
`, "builtin")
		found := false
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				if closure, ok := instruction.(*ssa.MakeClosure); ok {
					found = true
					if got := closureRebindsCapture(closure.Fn.(*ssa.Function)); got != test.rebinds {
						t.Fatalf("%s: capture rebinding=%v", test.body, got)
					}
				}
			}
		}
		if !found {
			t.Fatal("fixture has no closure")
		}
	}
}
