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
	"encoding/json"
	"fmt"
	"testing"
)

// roundTripJSON marshals to JSON and back.
func roundTripJSON(v Scmer) (Scmer, []byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return NewNil(), nil, fmt.Errorf("marshal: %w", err)
	}
	var out Scmer
	if err := json.Unmarshal(b, &out); err != nil {
		return NewNil(), b, fmt.Errorf("unmarshal: %w", err)
	}
	return out, b, nil
}

func TestScmerJSON_RoundTripMany(t *testing.T) {
	cases := make([]Scmer, 0, 140)
	// nil and booleans
	cases = append(cases, NewNil(), NewBool(false), NewBool(true))
	// integers: -50..49 (100 examples)
	for i := -50; i < 50; i++ {
		cases = append(cases, NewInt(int64(i)))
	}
	// floats
	floats := []float64{0, -0, 0.5, -0.5, 1.0, -1.0, 2.5, 3.1415926535, 2.7182818284, 1e6, -1e6, 1e-6, -1e-6}
	for _, f := range floats {
		cases = append(cases, NewFloat(f))
	}
	// strings
	for i := 0; i < 15; i++ {
		cases = append(cases, NewString(fmt.Sprintf("s%d", i)))
	}
	// symbols
	for i := 0; i < 15; i++ {
		cases = append(cases, NewSymbol(fmt.Sprintf("sym%d", i)))
	}
	// short lists
	for i := 0; i < 15; i++ {
		lst := []Scmer{NewSymbol("a"), NewInt(int64(i)), NewString("x")}
		cases = append(cases, NewSlice(lst))
	}
	// nested lists
	for i := 0; i < 10; i++ {
		inner := NewSlice([]Scmer{NewInt(int64(i)), NewSymbol("z")})
		cases = append(cases, NewSlice([]Scmer{NewSymbol("pair"), inner}))
	}
	// lambdas (10 examples): (lambda (a b) (+ a b)) with variations in body
	for i := 0; i < 10; i++ {
		params := NewSlice([]Scmer{NewSymbol("a"), NewSymbol("b")})
		body := NewSlice([]Scmer{NewSymbol("+"), NewSymbol("a"), NewInt(int64(i))})
		p := Proc{Params: params, Body: body, En: &Globalenv}
		cases = append(cases, NewProcStruct(p))
	}
	// native funcs: ensure some are present
	cases = append(cases, Globalenv.Vars[Symbol("list")])
	// maybe more if available in globals
	if v, ok := Globalenv.Vars[Symbol("append")]; ok {
		cases = append(cases, v)
	}

	// Execute round-trips and validate
	for idx, in := range cases {
		got, b, err := roundTripJSON(in)
		if err != nil {
			t.Fatalf("case %d failed: %v (json %s)", idx, err, string(b))
		}
		// Special shape checks
		switch auxTag(in.aux) {
		case tagSymbol:
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("case %d symbol decode: %v", idx, err)
			}
			if m["symbol"] != in.String() {
				t.Fatalf("case %d symbol wrong encoding: %v", idx, m)
			}
		case tagFunc:
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatalf("case %d func decode: %v", idx, err)
			}
			if _, ok := m["symbol"]; !ok {
				t.Fatalf("case %d func should encode to {symbol:name}", idx)
			}
		case tagProc:
			var arr []any
			if err := json.Unmarshal(b, &arr); err != nil {
				t.Fatalf("case %d proc decode: %v", idx, err)
			}
			if len(arr) < 3 {
				t.Fatalf("case %d proc encoding too short: %v", idx, arr)
			}
			head, ok := arr[0].(map[string]any)
			if !ok || head["symbol"] != "lambda" {
				t.Fatalf("case %d proc head not lambda: %v", idx, arr[0])
			}
		}
		// Prefer semantic equality (tolerates int<->float where values match)
		if !Equal(in, got) {
			// Fallback to structural/string comparison for procs and complex values
			strIn := SerializeToString(in, &Globalenv)
			strGot := SerializeToString(got, &Globalenv)
			if strIn != strGot {
				t.Fatalf("case %d roundtrip mismatch:\n in:  %s\n got: %s\n json: %s", idx, strIn, strGot, string(b))
			}
		}
	}
}
