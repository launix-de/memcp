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
	if !strings.Contains(code, "if (((len(args)-1)%2)!=0) {") {
		t.Fatalf("argument shape was not staged in emitter:\n%s", code)
	}
	if strings.Count(code, "GoFuncAddr(pick)") != 1 {
		t.Fatalf("only the unsupported arm should keep its native boundary:\n%s", code)
	}
	if !strings.Contains(code, "jitMaterializeVirtualGoSlice(ctx, args[1:])") {
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
