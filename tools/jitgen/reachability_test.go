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
import "go/token"
import "go/types"
import "go/constant"
import "golang.org/x/tools/go/ssa"

// These helpers deliberately contain code the generator cannot lower. A proof
// of unreachability must eliminate that code before testing lowering support;
// silently retaining a whole native helper boundary is not constant folding.
func TestReachabilityPrunesInlinedHelperBlocks(t *testing.T) {
	for _, sample := range []struct{ name, source string }{
		{"nil_callback", `package sample
 type Scmer struct { ptr, aux uint64 }
 func pick(value Scmer, merge func(Scmer, Scmer) Scmer) Scmer {
  if merge == nil { return value }; return merge(value, value)
 }
 func builtin(a ...Scmer) Scmer { return pick(a[0], nil) }
`},
		{"forwarded_bool", `package sample
 type Scmer struct { ptr, aux uint64 }
 func unknown(value Scmer) Scmer
 func pick(value Scmer, skip bool) Scmer {
  if skip { return value }; return unknown(value)
 }
 func forward(value Scmer, skip bool) Scmer { return pick(value, skip) }
 func builtin(a ...Scmer) Scmer { return forward(a[0], true) }
`},
		{"dead_tuple_call", `package sample
 type Scmer struct { ptr, aux uint64 }
 func tuple(value Scmer) (Scmer,bool)
 func pick(value Scmer, skip bool) Scmer {
  if skip { return value }; other,_:=tuple(value); return other
 }
 func builtin(a ...Scmer) Scmer { return pick(a[0], true) }
`},
		{"nil_phi", `package sample
 type Scmer struct { ptr, aux uint64 }
 func pick(value Scmer, count int) Scmer {
  var merge func(Scmer, Scmer) Scmer
  if count % 2 != 0 { merge = func(old, next Scmer) Scmer { return next } }
  if merge == nil { return value }; return merge(value, value)
 }
 func builtin(a ...Scmer) Scmer { return pick(a[0], 2) }
`},
	} {
		t.Run(sample.name, func(t *testing.T) {
			fn := buildTestSSAFunction(t, sample.source, "builtin")
			code, failure := generateClosure("builtin", fn, nil)
			if failure != "" {
				t.Fatal(failure)
			}
			for _, forbidden := range []string{"GoFuncAddr(pick)", "GoFuncAddr(forward)", "GoFuncAddr(unknown)", "GoFuncAddr(tuple)", "jitInvokeMergeCallback"} {
				if strings.Contains(code, forbidden) {
					t.Fatalf("dead helper code survived (%s):\n%s", forbidden, code)
				}
			}
		})
	}
}

func TestReachabilityRetainsDynamicHelperBranch(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 type Scmer struct { ptr, aux uint64 }
 func unknown(value Scmer) Scmer
 func (Scmer) Int() int64
 func pick(value Scmer, skip bool) Scmer {
  if skip { return value }; return unknown(value)
 }
 func builtin(a ...Scmer) Scmer { return pick(a[0], a[1].Int()!=0) }
`, "builtin")
	code, failure := generateClosure("builtin", fn, nil)
	if failure != "" {
		t.Fatal(failure)
	}
	if !strings.Contains(code, "GoFuncAddr(unknown)") {
		t.Fatal("unproven helper branch was removed")
	}
}

func TestReachabilityLoopBackedgeWidensConstants(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 func sum(n int) int { result:=0; for i:=0; i<n; i++ { result+=i }; return result }
`, "sum")
	for _, bound := range []int64{0, 3} {
		proof := analyzeSSAReachability(fn, map[ssa.Value]*ssa.Const{fn.Params[0]: ssa.NewConst(constant.MakeInt64(bound), types.Typ[types.Int])})
		var returned *ssa.Const
		for _, b := range fn.Blocks {
			if !proof.blocks[b] {
				continue
			}
			if ret, ok := b.Instrs[len(b.Instrs)-1].(*ssa.Return); ok {
				returned = proof.constant(ret.Results[0])
			}
		}
		if bound == 0 {
			if returned == nil || constant.Sign(returned.Value) != 0 || len(proof.blocks) == len(fn.Blocks) {
				t.Fatal("zero-iteration loop not eliminated")
			}
		} else if returned != nil || len(proof.blocks) != len(fn.Blocks) {
			t.Fatal("loop's first iteration was mistaken for a permanent constant")
		}
	}
	// Bindings and executable edges belong to the call, not to shared SSA.
	generic := analyzeSSAReachability(fn, nil)
	if len(generic.blocks) != len(fn.Blocks) {
		t.Fatal("a prior specialization mutated the shared function")
	}
}

func TestReachabilityFoldingPreservesIntegerAndPanicSemantics(t *testing.T) {
	for _, test := range []struct {
		typ        types.BasicKind
		op         token.Token
		a, b, want int64
	}{
		{types.Uint8, token.ADD, 255, 1, 0},
		{types.Int8, token.ADD, 127, 1, -128},
		{types.Int64, token.QUO, -7, 2, -3},
		{types.Int8, token.SHL, 1, 8, 0},
		{types.Int8, token.SHR, -1, 100, -1},
	} {
		typ := types.Typ[test.typ]
		got := foldSSAConstant(typ, test.op, ssa.NewConst(constant.MakeInt64(test.a), typ), ssa.NewConst(constant.MakeInt64(test.b), typ))
		if got.dynamic || got.value == nil || !constant.Compare(got.value.Value, token.EQL, constant.MakeInt64(test.want)) {
			t.Fatalf("%v %v %v: %+v, want %d", test.a, test.op, test.b, got, test.want)
		}
	}
	typ := types.Typ[types.Int]
	for _, test := range []struct {
		op    token.Token
		right int64
	}{{token.QUO, 0}, {token.REM, 0}, {token.SHL, -1}} {
		got := foldSSAConstant(typ, test.op, ssa.NewConst(constant.MakeInt64(1), typ), ssa.NewConst(constant.MakeInt64(test.right), typ))
		if !got.dynamic {
			t.Fatalf("fold removed a required panic: %v %+v", test.op, got)
		}
	}
}

func TestReachabilityDistinguishesParallelPhiEdges(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
 func pick(flag bool) int { value:=1; if flag { value=2 }; return value }
`, "pick")
	var branch *ssa.BasicBlock
	var join *ssa.BasicBlock
	var phi *ssa.Phi
	for _, b := range fn.Blocks {
		if _, ok := b.Instrs[len(b.Instrs)-1].(*ssa.If); ok {
			branch = b
		}
		if p, ok := b.Instrs[0].(*ssa.Phi); ok {
			join, phi = b, p
		}
	}
	if branch == nil || phi == nil {
		t.Fatal("missing branch/phi fixture")
	}
	// Parallel CFG edges have distinct phi inputs, even though both ends match.
	// This graph is private to the test; production analysis never edits SSA.
	branch.Succs = []*ssa.BasicBlock{join, join}
	join.Preds = []*ssa.BasicBlock{branch, branch}
	phi.Edges = []ssa.Value{ssa.NewConst(constant.MakeInt64(10), types.Typ[types.Int]), ssa.NewConst(constant.MakeInt64(20), types.Typ[types.Int])}
	for _, selected := range []bool{true, false} {
		proof := analyzeSSAReachability(fn, map[ssa.Value]*ssa.Const{fn.Params[0]: ssa.NewConst(constant.MakeBool(selected), types.Typ[types.Bool])})
		want := int64(20)
		if selected {
			want = 10
		}
		got := proof.constant(phi)
		if got == nil || !constant.Compare(got.Value, token.EQL, constant.MakeInt64(want)) {
			t.Fatalf("edge selection %v: constant=%v want=%d", selected, got, want)
		}
	}
}

func TestReachabilityPreservesUnsignedConstantWord(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
func step(value uint64) uint64 { return value + 1 }
`, "step")
	input := constant.BinaryOp(constant.Shift(constant.MakeInt64(1), token.SHL, 63), token.SUB, constant.MakeInt64(1))
	proof := analyzeSSAReachability(fn, map[ssa.Value]*ssa.Const{fn.Params[0]: ssa.NewConst(input, types.Typ[types.Uint64])})
	ret := fn.Blocks[0].Instrs[len(fn.Blocks[0].Instrs)-1].(*ssa.Return)
	value := proof.constant(ret.Results[0])
	if value == nil || uint64(constWordValue(value.Value)) != uint64(1)<<63 {
		t.Fatal("folded unsigned high bit was lost")
	}
}
