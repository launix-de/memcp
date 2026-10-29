/*
	Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; see <https://www.gnu.org/licenses/>.
*/
package main

import "fmt"
import "testing"
import "strings"
import "go/token"
import "go/constant"
import "golang.org/x/tools/go/ssa"

func TestRemainderFactsAtBBEdges(t *testing.T) {
	for _, sample := range []struct {
		name, source string
		folded       bool
	}{
		{"even_backedge", `func sample(n int) int { for i:=0;i<n;i+=2 {if i%2!=0 {return 7}};return 0 }`, true},
		{"signed_overflow", `func sample(n uint8) int {v:=int8(-4);for n>0 {if v%2!=0 {return 7};v+=2;n--};return 0}`, true},
		{"different_backedge", `func sample(n int) int {for i:=0;i<n;i++ {if i%2!=0 {return 7}};return 0}`, false},
		{"non_binary_overflow", `func sample(n uint8) int {v:=uint8(0);for n>0 {if v%3!=0 {return 7};v+=3;n--};return 0}`, false},
		{"dynamic_signed_product", `func sample(n int) int {if (n*8)%8!=0 {return 7};return 0}`, true},
		{"masked_signed", `func sample(n int) int {if (n&6)%2!=0 {return 7};return 0}`, true},
		{"negative_divisor", `func sample(n int) int {if (n*8)%-8!=0 {return 7};return 0}`, true},
		{"unit_divisor", `func sample(n int) int {if n%1!=0 {return 7};return 0}`, true},
		{"signed_nonzero_residue", `func sample(n int) int {if ((n*8)+1)%8!=0 {return 7};return 0}`, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			fn := buildTestSSAFunction(t, "package sample\n"+sample.source, "sample")
			proof := analyzeSSAReachability(fn, nil)
			seen := false
			for _, block := range fn.Blocks {
				for _, instruction := range block.Instrs {
					if v, ok := instruction.(*ssa.BinOp); ok && v.Op == token.REM {
						seen = true
						c := proof.constant(v)
						if sample.folded && (c == nil || c.Value == nil || constant.Sign(c.Value) != 0) {
							t.Fatalf("remainder not proven zero: %s", v)
						}
						if !sample.folded && c != nil {
							t.Fatalf("unsound modulo folding: %s => %v", v, c)
						}
					}
				}
			}
			if !seen {
				t.Fatal("fixture has no remainder")
			}
		})
	}
}

// These complete graphs let the induction proof see both entries immediately.
// A candidate residue must agree with EVERY incoming edge, regardless of order.
func TestRemainderRejectsDifferingPhiEntries(t *testing.T) {
	for _, initial := range []int{0, 1} {
		fn := buildTestSSAFunction(t, fmt.Sprintf(`package sample
 func sample(flag bool,n int) int {i:=%d;if flag {i=1};for i<n {if i%%2!=0 {return 7};i+=2};return 0}`, initial), "sample")
		proof := analyzeSSAReachability(fn, nil)
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				if v, ok := instruction.(*ssa.BinOp); ok && v.Op == token.REM && proof.constant(v) != nil && constant.Sign(proof.constant(v).Value) == 0 {
					t.Fatal("mixed or odd entry was proven even")
				}
			}
		}
	}
}

func TestPointerBranchKeepsMachineFlags(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct {ptr,aux uint64}
 func pointer(Scmer) *int
 func sample(args ...Scmer) Scmer {
  if pointer(args[0])!=nil {return args[0]};return args[1]
 }`, "sample")
	code, failure := generateClosure("sample", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if !strings.Contains(code, "Loc: LocFlags") || strings.Contains(code, "EmitSetcc") {
		t.Fatalf("pointer branch materializes a bool:\n%s", code)
	}
}

func TestRemainderFoldingRetainsOperandEffects(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct{ptr,aux uint64}
 func effect(Scmer) int64
 func sample(args ...Scmer) Scmer {
  if effect(args[0])%1!=0 {return args[1]};return args[0]
 }`, "sample")
	code, failure := generateClosure("sample", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if !strings.Contains(code, "GoFuncAddr(effect)") {
		t.Fatal("fold removed the evaluated operand's call/panic")
	}
}
