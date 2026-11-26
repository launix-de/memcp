/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"testing"
)

func TestSQLTemporalOutputDateFromGenericRepresentations(t *testing.T) {
	const unix = int64(1718451045)
	for _, value := range []Scmer{NewDate(unix), NewInt(unix), NewFloat(float64(unix)), NewString("2024-06-15 10:30:45")} {
		if got := sqlTemporalOutput(value, "DATE", NewString("UTC")).String(); got != "2024-06-15" {
			t.Fatalf("DATE output = %q, want 2024-06-15", got)
		}
	}
}

func TestSQLTemporalOutputPreservesNilAndUnknownType(t *testing.T) {
	if got := sqlTemporalOutput(NewNil(), "DATE", NewString("UTC")); !got.IsNil() {
		t.Fatalf("DATE NULL output = %v, want nil", got)
	}
	value := NewInt(42)
	if got := sqlTemporalOutput(value, "VARCHAR", NewString("UTC")); got != value {
		t.Fatalf("unknown temporal type changed value: got %v, want %v", got, value)
	}
}

func TestSQLTemporalOutputPreservesMySQLZeroDates(t *testing.T) {
	for _, test := range []struct {
		input   string
		sqlType string
		want    string
	}{
		{input: "0000-00-00", sqlType: "DATE", want: "0000-00-00"},
		{input: "0000-00-00 00:00:00", sqlType: "DATETIME", want: "0000-00-00 00:00:00"},
	} {
		unix, ok := ParseDateString(test.input)
		if !ok {
			t.Fatalf("ParseDateString(%q) rejected a MySQL zero date", test.input)
		}
		for _, value := range []Scmer{NewDate(unix), NewString(test.input)} {
			if got := sqlTemporalOutput(value, test.sqlType, NewString("UTC")).String(); got != test.want {
				t.Fatalf("%s output (tag %d) = %q, want %q", test.sqlType, value.GetTag(), got, test.want)
			}
		}
	}
}

func TestDateStringConversionsPreserveTemporalValues(t *testing.T) {
	for _, input := range []string{"1970-01-01 00:00:00", "2024-06-15 10:30:45", "0000-00-00 00:00:00"} {
		t.Run(input, func(t *testing.T) {
			unix, ok := ParseDateString(input)
			if !ok {
				t.Fatal("invalid test date")
			}
			value := NewDate(unix)
			if got := String(value); got != input {
				t.Errorf("String(date) = %q, want %q", got, input)
			}
			var buffer bytes.Buffer
			WriteStringValue(bufferTextWriter(&buffer), value)
			if got := buffer.String(); got != input {
				t.Errorf("WriteStringValue(date) = %q, want %q", got, input)
			}
		})
	}
}

// Literal predecessor payloads establish compatibility independently of this
// runtime's encoder and of the zero-fraction upgrade fixtures.
func TestHistoricalDateNanoseconds(t *testing.T) {
	decode := func(payload string) Scmer {
		var value Scmer
		if err := UnmarshalPersistentJSON([]byte(payload), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	first := decode(`{"scmer_date_v1":[1700000000,100000001,17]}`)
	second := decode(`{"scmer_date_v1":[1700000000,100000002,17]}`)
	if Equal(first, second) || EqualSQL(first, second).Bool() || !Less(first, second) || Less(second, first) {
		t.Fatal("adjacent native nanoseconds lost exact ordering")
	}
	if DateToDisplay(first, nil) != "2023-11-14 22:13:20.100000001" {
		t.Fatal("fraction was lost in display", first)
	}
	for _, value := range []Scmer{first, second, decode(`{"scmer_date_v1":[-1,999999999,2047]}`), decode(`{"scmer_date_v1":[1700000000,0,17]}`)} {
		check := func(restored Scmer) {
			want, err := MarshalPersistentJSON(value)
			if err != nil {
				t.Fatal(err)
			}
			got, err := MarshalPersistentJSON(restored)
			if err != nil || string(want) != string(got) {
				t.Fatalf("date identity changed: %s -> %s (%v)", want, got, err)
			}
		}
		check(Eval(Read(t.Name(), SerializeToString(value, &Globalenv), true), &Globalenv))
		quoted := NewSlice([]Scmer{NewSymbol("quote"), NewSlice([]Scmer{value})})
		check(Eval(Read(t.Name(), SerializeToString(quoted, &Globalenv), true), &Globalenv).Slice()[0])
		env := &Env{Vars: Vars{Symbol("captured"): value}, Outer: &Globalenv}
		proc := Eval(Read(t.Name(), `(lambda () captured)`, true), env)
		wire, err := MarshalProcedure(proc)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := UnmarshalProcedure(wire)
		if err != nil {
			t.Fatal(err)
		}
		check(Apply(restored))
		if HashKey(value) != HashKey(NewFloat(value.Float())) {
			t.Fatal("equal numeric dates hash differently")
		}
	}
}
