/*
Copyright (C) 2025  Carl-Philip Hänsch

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
	"bytes"
	"encoding/json"
	"testing"
)

func decodeAny(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

func TestScmerJSON_SymbolEncoding(t *testing.T) {
	s := NewSymbol("foo")
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal symbol: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode symbol json: %v", err)
	}
	if m["symbol"] != "foo" {
		t.Fatalf("expected {symbol:\"foo\"}, got %v", m)
	}
	// round-trip
	var s2 Scmer
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatalf("unmarshal symbol: %v", err)
	}
	if !s2.IsSymbol() || s2.String() != "foo" {
		t.Fatalf("roundtrip symbol mismatch: %v", s2)
	}
}

func TestScmerJSON_ListAndSymbol(t *testing.T) {
	v := NewSlice([]Scmer{NewInt(1), NewString("x"), NewSymbol("y")})
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal list: %v", err)
	}
	var arr []any
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 items, got %d", len(arr))
	}
	if _, ok := arr[0].(float64); !ok { // default json dec uses float64
		t.Fatalf("first item not number: %T", arr[0])
	}
	if arr[1] != "x" {
		t.Fatalf("second item mismatch: %v", arr[1])
	}
	if m, ok := arr[2].(map[string]any); !ok || m["symbol"] != "y" {
		t.Fatalf("third item not symbol encoding: %T %v", arr[2], arr[2])
	}
	// round-trip
	var v2 Scmer
	if err := json.Unmarshal(b, &v2); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if !v2.IsSlice() {
		t.Fatalf("roundtrip not a list: %v", v2)
	}
	l2 := v2.Slice()
	if len(l2) != 3 || l2[0].Int() != 1 || l2[1].String() != "x" || !l2[2].IsSymbol() || l2[2].String() != "y" {
		t.Fatalf("roundtrip list mismatch: %v", l2)
	}
}

func TestScmerJSON_LambdaEncodingRoundtrip(t *testing.T) {
	// lambda (a b) (+ a b)
	params := NewSlice([]Scmer{NewSymbol("a"), NewSymbol("b")})
	body := NewSlice([]Scmer{NewSymbol("+"), NewSymbol("a"), NewSymbol("b")})
	p := Proc{Params: params, Body: body, En: &Globalenv}
	s := NewProcStruct(p)
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal lambda: %v", err)
	}
	// structure check: [ {symbol:lambda}, params, body ]
	var outer []any
	if err := json.Unmarshal(b, &outer); err != nil {
		t.Fatalf("decode lambda: %v", err)
	}
	if len(outer) < 3 {
		t.Fatalf("lambda encoding too short: %v", outer)
	}
	head, ok := outer[0].(map[string]any)
	if !ok || head["symbol"] != "lambda" {
		t.Fatalf("lambda head mismatch: %v", outer[0])
	}
	// round-trip
	var s2 Scmer
	if err := json.Unmarshal(b, &s2); err != nil {
		t.Fatalf("unmarshal lambda: %v", err)
	}
	if !s2.IsProc() {
		t.Fatalf("roundtrip not a proc: %v", s2)
	}
	p2 := s2.Proc()
	if !p2.Params.IsSlice() {
		t.Fatalf("params not list: %v", p2.Params)
	}
	ps := p2.Params.Slice()
	if len(ps) != 2 || !ps[0].IsSymbol() || ps[0].String() != "a" || !ps[1].IsSymbol() || ps[1].String() != "b" {
		t.Fatalf("params mismatch: %v", ps)
	}
	if !p2.Body.IsSlice() {
		t.Fatalf("body not list: %v", p2.Body)
	}
	bs := p2.Body.Slice()
	if len(bs) != 3 || !bs[0].IsSymbol() || bs[0].String() != "+" || !bs[1].IsSymbol() || !bs[2].IsSymbol() {
		t.Fatalf("body mismatch: %v", bs)
	}
}

func TestScmerJSON_NativeFuncEncoding(t *testing.T) {
	// list function must marshal to {symbol:"list"}
	v := Globalenv.Vars[Symbol("list")]
	if v.GetTag() != tagFunc {
		t.Fatalf("global list not a native func: %v", v)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal func: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode func: %v", err)
	}
	if m["symbol"] != "list" {
		t.Fatalf("expected {symbol:list}, got %v", m)
	}
}

func TestScmerJSON_PrimitivesRoundTrip(t *testing.T) {
	cases := []Scmer{NewNil(), NewBool(true), NewInt(42), NewFloat(3.5), NewString("hi")}
	for _, c := range cases {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("marshal primitive %v: %v", c, err)
		}
		var r Scmer
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatalf("unmarshal primitive %v: %v", c, err)
		}
		// compare via Equal for non-proc types
		if !Equal(c, r) && !(c.IsNil() && !r.Bool()) { // tolerate nil encoding
			t.Fatalf("roundtrip mismatch: %v vs %v", SerializeToString(c, &Globalenv), SerializeToString(r, &Globalenv))
		}
	}
}
