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
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/ssa"
)

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
