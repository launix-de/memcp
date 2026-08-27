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
	"go/parser"
	"go/token"
	"testing"
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
