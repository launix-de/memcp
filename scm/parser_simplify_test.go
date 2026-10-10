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
package scm

import (
	"math"
	"testing"
)

func TestSimplifyParsesJSONObjectAndArray(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{`{"name":"Ada","active":true}`, `{"active":true,"name":"Ada"}`},
		{`[1,{"nested":true},null]`, `[1,{"nested":true},null]`},
	} {
		value := Simplify(test.input)
		if !value.IsBSON() {
			t.Fatalf("Simplify(%q) tag = %d, want BSON", test.input, value.GetTag())
		}
		if got := value.String(); got != test.want {
			t.Fatalf("Simplify(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestSimplifyLeavesInvalidJSONAsString(t *testing.T) {
	for _, input := range []string{"{not json}", "[1,]", " [1]"} {
		value := Simplify(input)
		if !value.IsString() || value.String() != input {
			t.Fatalf("Simplify(%q) = %s, want unchanged string", input, value.String())
		}
	}
}

func TestSimplifyStillParsesNumbers(t *testing.T) {
	value := Simplify("12.5")
	if !value.IsFloat() || value.Float() != 12.5 {
		t.Fatalf("Simplify number = %s, want 12.5", value.String())
	}
}

func TestReadIntegerLiterals(t *testing.T) {
	for _, test := range []struct {
		input string
		want  int64
	}{
		{"0", 0}, {"1", 1}, {"-1", -1}, {"00012", 12},
		{"9007199254740993", 9007199254740993},
		{"-9007199254740993", -9007199254740993},
		{"9223372036854775807", math.MaxInt64},
		{"-9223372036854775808", math.MinInt64},
	} {
		// Exercise both end-of-input and delimiter-terminated tokens.
		for _, suffix := range []string{"", " "} {
			value := Read(t.Name(), test.input+suffix)
			if !value.IsInt() || value.Int() != test.want {
				t.Fatalf("Read(%q) = %v (tag %d), want integer %d", test.input+suffix, value, value.GetTag(), test.want)
			}
		}
	}
}

func TestReadFloatAndOperatorTokens(t *testing.T) {
	for _, input := range []string{"1.0", "-0.0", "12.5", "9223372036854775808", "-9223372036854775809"} {
		for _, suffix := range []string{"", " "} {
			value := Read(t.Name(), input+suffix)
			if !value.IsFloat() {
				t.Fatalf("Read(%q) tag = %d, want float", input+suffix, value.GetTag())
			}
			if input == "-0.0" && !math.Signbit(value.Float()) {
				t.Fatal("floating negative zero lost its sign")
			}
		}
	}
	for _, test := range []struct{ input, want string }{{"-", "-"}, {"1.2.3", "NaN"}} {
		for _, suffix := range []string{"", " "} {
			value := Read(t.Name(), test.input+suffix)
			if !value.IsSymbol() || value.Symbol() != Symbol(test.want) {
				t.Fatalf("Read(%q) = %v, want symbol %s", test.input+suffix, value, test.want)
			}
		}
	}
}

func TestReadIntegerConstantFolding(t *testing.T) {
	for _, input := range []string{"(+ 9007199254740992 1)", "(intdiv 9007199254740993 1)"} {
		value := Optimize(Read(t.Name(), input), &Globalenv, nil)
		if !value.IsInt() || value.Int() != 9007199254740993 {
			t.Fatalf("Optimize(%s) = %v, want exact integer", input, value)
		}
	}
}

func TestJITReadIntegerLiteralDivision(t *testing.T) {
	fn := compileJITExpressionTestProc(t, "(lambda (value) (intdiv value 1))")
	for _, want := range []int64{math.MinInt64, -9007199254740993, 9007199254740993, math.MaxInt64} {
		value := Apply(fn, NewInt(want))
		if !value.IsInt() || value.Int() != want {
			t.Fatalf("integer division by literal 1 returned %v, want %d", value, want)
		}
	}
}

var numberTokenBenchmarkSink []Scmer

func BenchmarkTokenizeNumbers(b *testing.B) {
	for _, test := range []struct{ name, input string }{
		{"integers", "0 1 -1 12 255 1000 9007199254740993 -9223372036854775808"},
		{"floats", "0.0 1.0 -1.0 12.5 255.25 1000.5 9007199254740993.0 -9223372036854775808.0"},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				numberTokenBenchmarkSink = tokenize("benchmark", test.input)
			}
		})
	}
}
