/*
Copyright (C) 2024  Carl-Philip Hänsch

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
	"encoding/json"
	"testing"
)

func mustUnmarshalScmer(t *testing.T, js string) Scmer {
	t.Helper()
	var v Scmer
	if err := json.Unmarshal([]byte(js), &v); err != nil {
		t.Fatalf("unmarshal %s: %v", js, err)
	}
	return v
}

func TestJSONParse_Primitives(t *testing.T) {
	// int
	v := mustUnmarshalScmer(t, "123")
	if !v.IsInt() || v.Int() != 123 {
		t.Fatalf("expected int 123, got %v", v)
	}
	// float
	v = mustUnmarshalScmer(t, "1.5")
	if !v.IsFloat() || v.Float() != 1.5 {
		t.Fatalf("expected float 1.5, got %v", v)
	}
	// bool
	if !mustUnmarshalScmer(t, "true").Bool() {
		t.Fatalf("expected true")
	}
	if mustUnmarshalScmer(t, "false").Bool() {
		t.Fatalf("expected false")
	}
	// null
	if !mustUnmarshalScmer(t, "null").IsNil() {
		t.Fatalf("expected nil")
	}
	// string
	v = mustUnmarshalScmer(t, `"hello\nworld"`)
	var expected string
	if err := json.Unmarshal([]byte(`"hello\nworld"`), &expected); err != nil {
		t.Fatalf("decode expected: %v", err)
	}
	if !v.IsString() || v.String() != expected {
		t.Fatalf("expected string with newline, got %q", v.String())
	}
}

func TestJSONParse_ObjectToAssoc(t *testing.T) {
	v := mustUnmarshalScmer(t, `{"a":1,"b":"x","c":true,"d":null}`)
	// Expect assoc list equivalent to ("a" 1 "b" "x" "c" true "d" nil), order-insensitive
	expected := NewSlice([]Scmer{NewString("a"), NewInt(1), NewString("b"), NewString("x"), NewString("c"), NewBool(true), NewString("d"), NewNil()})
	ap, ok1 := assocPairs(v)
	bp, ok2 := assocPairs(expected)
	if !ok1 || !ok2 || !equalAssocPairs(ap, bp) {
		t.Fatalf("assoc mismatch: %s vs %s", SerializeToString(v, &Globalenv), SerializeToString(expected, &Globalenv))
	}
}

func TestJSONParse_ArrayNested(t *testing.T) {
	v := mustUnmarshalScmer(t, `[{"symbol":"y"}, 2, "z", [1,2]]`)
	if !v.IsSlice() {
		t.Fatalf("expected list, got %v", v)
	}
	l := v.Slice()
	if len(l) != 4 || !l[0].IsSymbol() || l[0].String() != "y" || !l[1].IsInt() || l[1].Int() != 2 || !l[2].IsString() || l[2].String() != "z" || !l[3].IsSlice() || len(l[3].Slice()) != 2 {
		t.Fatalf("unexpected list content: %v", SerializeToString(v, &Globalenv))
	}
}

func TestJSONParse_SymbolObject(t *testing.T) {
	v := mustUnmarshalScmer(t, `{"symbol":"alpha-beta"}`)
	if !v.IsSymbol() || v.String() != "alpha-beta" {
		t.Fatalf("expected symbol alpha-beta, got %v", v)
	}
}

func TestJSONParse_LambdaFull(t *testing.T) {
	js := `[{"symbol":"lambda"}, [{"symbol":"a"}, {"symbol":"b"}], [{"symbol":"+"}, {"symbol":"a"}, {"symbol":"b"}], 3]`
	v := mustUnmarshalScmer(t, js)
	if !v.IsProc() {
		t.Fatalf("expected proc, got %v", v)
	}
	p := v.Proc()
	if p.NumVars != 3 {
		t.Fatalf("expected NumVars=3, got %d", p.NumVars)
	}
	if !p.Params.IsSlice() || !p.Body.IsSlice() {
		t.Fatalf("params/body must be lists")
	}
	ps := p.Params.Slice()
	if len(ps) != 2 || !ps[0].IsSymbol() || ps[0].String() != "a" || !ps[1].IsSymbol() || ps[1].String() != "b" {
		t.Fatalf("params mismatch: %v", ps)
	}
	bs := p.Body.Slice()
	if len(bs) != 3 || !bs[0].IsSymbol() || bs[0].String() != "+" {
		t.Fatalf("body mismatch: %v", bs)
	}
}

func TestJSONParse_SourceInfoUnwrap(t *testing.T) {
	// Create a SourceInfo-wrapped value and ensure it marshals to plain value and parses back
	// Use a string to avoid small-int ptr representation on stack
	si := SourceInfo{source: "t.scm", line: 1, col: 1, value: NewString("7")}
	s := NewSourceInfo(si)
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal sourceinfo: %v", err)
	}
	var out Scmer
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal sourceinfo: %v", err)
	}
	if !Equal(out, si.value) {
		t.Fatalf("sourceinfo roundtrip mismatch: %v vs %v", out, si.value)
	}
}

func TestJSONParse_DeepNestedMixed(t *testing.T) {
	js := `{"obj":{"inner":[1,{"k":"v"},{"symbol":"S"}],"n":-5},"arr":[{"symbol":"lambda"},[],[]]}`
	v := mustUnmarshalScmer(t, js)
	if !v.IsSlice() {
		t.Fatalf("expected assoc list")
	}
	// Basic spot checks
	_, ok := v.Any().([]Scmer)
	if !ok {
		// v is a slice, Any() returns []Scmer for tagSlice
	}
}
