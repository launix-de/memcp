/*
Copyright (C) 2026  Carl-Philip Haensch

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
package main

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildTestSSAFunction(t *testing.T, source, name string) *ssa.Function {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("jitgen.test", "sample")
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, pkg, []*ast.File{file}, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := ssaPkg.Func(name)
	if fn == nil {
		t.Fatalf("SSA function %q not found", name)
	}
	return fn
}

func TestFallbackClosureCallsDeclaredBuiltinDirectly(t *testing.T) {
	got := generateFallbackClosure("json_encode")
	if _, err := parser.ParseExpr(got); err != nil {
		t.Fatalf("fallback is not valid Go: %v\n%s", err, got)
	}
	if !strings.Contains(got, `jitEmitGoVariadicCallFromDescs(ctx, declarations["json_encode"].Fn, args, result)`) {
		t.Fatalf("fallback does not call the declaration directly:\n%s", got)
	}
	if !strings.Contains(got, `ctx.Coverage.NativeCalls++`) {
		t.Fatalf("fallback does not account for the native declaration call:\n%s", got)
	}
}

func TestOperatorJITEmitMissing(t *testing.T) {
	if !operatorJITEmitMissing(operatorInfo{}) {
		t.Fatal("absent JITEmit field must be treated as missing")
	}
	nilExpr, err := parser.ParseExpr("nil")
	if err != nil || !operatorJITEmitMissing(operatorInfo{jitExpr: nilExpr}) {
		t.Fatal("nil JITEmit field must be treated as missing")
	}
	funcExpr, err := parser.ParseExpr("func() {}")
	if err != nil || operatorJITEmitMissing(operatorInfo{jitExpr: funcExpr}) {
		t.Fatal("existing JITEmit function must be preserved")
	}
}

func TestArithmeticUsesRequestedResultPayloadRegister(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
type Scmer struct{}
func NewInt(int64) Scmer
func (Scmer) Int() int64
func add(a ...Scmer) Scmer { return NewInt(a[0].Int() + a[1].Int()) }
`, "add")
	code, errMsg := generateClosure("add", fn, nil)
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	for _, want := range []string{
		`declaration := declarations["add"]`,
		"ctx.Coverage.InlinedCalls++",
		"= result.Reg2",
		"ctx.EmitAddInt64",
		"ctx.EmitMakeInt(result",
		"if !resultTarget",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("generated arithmetic emitter does not contain %q:\n%s", want, code)
		}
	}
}

func TestCollectOperatorsUsesRootFunctionTypeDescriptor(t *testing.T) {
	const source = `package sample
func init() {
	Declare(&env, &Declaration{
		Name: "nested",
		Fn: func(a ...Scmer) Scmer { return a[0] },
		Type: &TypeDescriptor{
			Kind: "func",
			Description: "root description",
			Params: []*TypeDescriptor{{
				Kind: "list",
				Label: "callbacks",
				Element: &TypeDescriptor{Kind: "func"},
			}},
			Return: &TypeDescriptor{Kind: "any"},
		},
	})
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	ops := collectOperators(fset, file, "sample.go")
	if len(ops) != 1 || ops[0].name != "nested" || !ops[0].jitInsertPos.IsValid() {
		t.Fatalf("collectOperators() = %#v, want one root declaration insertion", ops)
	}
}

func TestSlicePhiUsesThreeWordLayout(t *testing.T) {
	sliceType := types.NewSlice(types.Typ[types.Int64])
	if !isPhiTripleType(sliceType) {
		t.Fatal("Go slice phi was not classified as ptr/len/cap triple")
	}
	if isPhiPairType(sliceType) {
		t.Fatal("Go slice phi was also classified as a two-word value")
	}
}

func TestBoundedAppendStartsWithSpareCapacity(t *testing.T) {
	zero := ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
	one := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	bounded := &ssa.Phi{Edges: []ssa.Value{&ssa.MakeSlice{Len: zero, Cap: one}}}
	if !phiStartsWithBoundedEmptySlice(bounded) {
		t.Fatal("empty slice with separately bounded capacity was rejected")
	}
	unbounded := &ssa.Phi{Edges: []ssa.Value{&ssa.MakeSlice{Len: zero, Cap: zero}}}
	if phiStartsWithBoundedEmptySlice(unbounded) {
		t.Fatal("zero-capacity slice was accepted as non-growing append target")
	}
}

func TestFunctionCallsMultipleResults(t *testing.T) {
	fn := buildTestSSAFunction(t, `package sample
func pair() (bool, bool) { return true, true }
func caller() bool {
	matched, handled := pair()
	return matched && handled
}
`, "caller")
	if !functionCallsMultipleResults(fn) {
		t.Fatal("call returning a tuple was not detected")
	}
}

func TestCollectOperatorsRejectsNonFunctionRootType(t *testing.T) {
	const source = `package sample
func init() {
	Declare(&env, &Declaration{
		Name: "not-a-function-type",
		Fn: func(a ...Scmer) Scmer { return a[0] },
		Type: &TypeDescriptor{Kind: "list", Element: &TypeDescriptor{Kind: "func"}},
	})
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ops := collectOperators(fset, file, "sample.go"); len(ops) != 0 {
		t.Fatalf("collectOperators() found nested function type as an operator: %#v", ops)
	}
}
