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

func TestSimplifyIntegerLiterals(t *testing.T) {
	for _, test := range []struct {
		input string
		want  int64
	}{
		{"0", 0}, {"+0", 0},
		{"1", 1}, {"-1", -1}, {"+12", 12},
		{"00012", 12}, {"-00012", -12},
		{"9007199254740993", 9007199254740993},
		{"-9007199254740993", -9007199254740993},
		{"9223372036854775807", math.MaxInt64},
		{"-9223372036854775808", math.MinInt64},
	} {
		t.Run(test.input, func(t *testing.T) {
			value := Simplify(test.input)
			if !value.IsInt() || value.Int() != test.want {
				t.Fatalf("Simplify(%q) = %v (tag %d), want integer %d", test.input, value, value.GetTag(), test.want)
			}
		})
	}
}

func TestSimplifyFloatLiteralForms(t *testing.T) {
	for _, test := range []struct {
		input string
		want  float64
	}{
		{"1.0", 1}, {"1.", 1}, {".25", .25}, {"12.5", 12.5},
		{"1e2", 100}, {"1E-2", .01}, {"0x1p4", 16},
		{"-0", math.Copysign(0, -1)}, {"-00", math.Copysign(0, -1)},
		{"-0000", math.Copysign(0, -1)},
		{"-0.0", math.Copysign(0, -1)}, {"-0e0", math.Copysign(0, -1)},
		{"9223372036854775808", 9223372036854775808.0},
		{"-9223372036854775809", -9223372036854775809.0},
		{"NaN", math.NaN()}, {"Inf", math.Inf(1)}, {"-Inf", math.Inf(-1)},
	} {
		t.Run(test.input, func(t *testing.T) {
			value := Simplify(test.input)
			if !value.IsFloat() {
				t.Fatalf("Simplify(%q) tag = %d, want float", test.input, value.GetTag())
			}
			got := value.Float()
			if math.IsNaN(test.want) {
				if !math.IsNaN(got) {
					t.Fatalf("Simplify(%q) = %v, want NaN", test.input, got)
				}
			} else if got != test.want || math.Signbit(got) != math.Signbit(test.want) {
				t.Fatalf("Simplify(%q) = %v, want %v with the same sign", test.input, got, test.want)
			}
		})
	}
}

func TestSimplifyLeavesOtherTextUnchanged(t *testing.T) {
	for _, input := range []string{"", "text", "+", "-", "1e", "0x10", " 12", "12 ", "1,2", "１２"} {
		value := Simplify(input)
		if !value.IsString() || value.String() != input {
			t.Fatalf("Simplify(%q) = %v (tag %d), want unchanged text", input, value, value.GetTag())
		}
	}
}

func TestReadIntegerLiterals(t *testing.T) {
	for _, test := range []struct {
		input string
		want  int64
	}{
		{"0", 0}, {"-0", 0}, {"-00", 0}, {"1", 1}, {"-1", -1}, {"00012", 12},
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
			if value.Float() == 0 && input[0] == '-' && !math.Signbit(value.Float()) {
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

func TestJITSimplifySignedZero(t *testing.T) {
	fn := compileJITExpressionTestProc(t, "(lambda (text) (simplify text))")
	for _, text := range []string{"-0", "-00", "-0000", "-0.0", "-0e0"} {
		value := Apply(fn, NewString(text))
		if !value.IsFloat() || value.Float() != 0 || !math.Signbit(value.Float()) {
			t.Fatalf("compiled Simplify(%q) = %v (tag %d), want signed floating zero", text, value, value.GetTag())
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

func testSimplifiedLiteralArithmetic(t *testing.T, add, subtract func(Scmer, Scmer) Scmer) {
	t.Helper()
	for _, test := range []struct {
		name     string
		left     string
		right    string
		subtract bool
		want     Scmer
	}{
		{"positive wide plus zero", "9007199254740993", "0", false, NewInt(9007199254740993)},
		{"positive wide plus one", "9007199254740993", "1", false, NewInt(9007199254740994)},
		{"negative wide plus zero", "-9007199254740993", "0", false, NewInt(-9007199254740993)},
		{"negative wide minus one", "-9007199254740993", "1", true, NewInt(-9007199254740994)},
		{"positive wide minus zero", "9007199254740993", "0", true, NewInt(9007199254740993)},
		{"positive wide minus one", "9007199254740993", "1", true, NewInt(9007199254740992)},
		{"negative wide minus zero", "-9007199254740993", "0", true, NewInt(-9007199254740993)},
		{"negative wide minus negative one", "-9007199254740993", "-1", true, NewInt(-9007199254740992)},
		{"maximum plus zero", "9223372036854775807", "0", false, NewInt(math.MaxInt64)},
		{"minimum plus zero", "-9223372036854775808", "0", false, NewInt(math.MinInt64)},
		{"maximum minus zero", "9223372036854775807", "0", true, NewInt(math.MaxInt64)},
		{"minimum minus zero", "-9223372036854775808", "0", true, NewInt(math.MinInt64)},
		{"opposite endpoints add", "9223372036854775807", "-9223372036854775808", false, NewInt(-1)},
		{"opposite endpoints reversed", "-9223372036854775808", "9223372036854775807", false, NewInt(-1)},
		{"maximum cancellation", "9223372036854775807", "9223372036854775807", true, NewInt(0)},
		{"minimum cancellation", "-9223372036854775808", "-9223372036854775808", true, NewInt(0)},
		{"maximum inward addition", "9223372036854775807", "-1", false, NewInt(math.MaxInt64 - 1)},
		{"minimum inward addition", "-9223372036854775808", "1", false, NewInt(math.MinInt64 + 1)},
		{"maximum inward subtraction", "9223372036854775807", "1", true, NewInt(math.MaxInt64 - 1)},
		{"minimum inward subtraction", "-9223372036854775808", "-1", true, NewInt(math.MinInt64 + 1)},
		{"addition positive overflow", "9223372036854775807", "1", false, NewFloat(float64(math.MaxInt64))},
		{"addition negative overflow", "-9223372036854775808", "-1", false, NewFloat(float64(math.MinInt64))},
		{"subtraction positive overflow", "9223372036854775807", "-1", true, NewFloat(float64(math.MaxInt64))},
		{"subtraction negative overflow", "-9223372036854775808", "1", true, NewFloat(float64(math.MinInt64))},
		{"subtract minimum overflows", "0", "-9223372036854775808", true, NewFloat(-float64(math.MinInt64))},
		{"decimal addition residue", ".06", ".01", false, NewFloat(.07)},
		{"decimal subtraction residue", ".06", ".01", true, NewFloat(.05)},
		{"negative decimal residue", "-.06", ".01", true, NewFloat(-.07)},
		{"mixed decimal addition", "1", ".5", false, NewFloat(1.5)},
	} {
		t.Run(test.name, func(t *testing.T) {
			operation := add
			if test.subtract {
				operation = subtract
			}
			got := operation(Simplify(test.left), Simplify(test.right))
			if test.want.IsInt() {
				if !got.IsInt() || got.Int() != test.want.Int() {
					t.Fatalf("got %v (tag %d), want integer %d", got, got.GetTag(), test.want.Int())
				}
			} else if !got.IsFloat() || got.Float() != test.want.Float() {
				t.Fatalf("got %v (tag %d), want float %v", got, got.GetTag(), test.want.Float())
			}
		})
	}
}

func TestSimplifiedLiteralArithmetic(t *testing.T) {
	testSimplifiedLiteralArithmetic(t,
		func(a, b Scmer) Scmer { return declarations["sql_add_numeric_literals"].Fn(a, b) },
		func(a, b Scmer) Scmer { return declarations["sql_sub_numeric_literals"].Fn(a, b) })
}

func TestJITSimplifiedLiteralArithmetic(t *testing.T) {
	add := compileJITExpressionTestProc(t, "(lambda (left right) (sql_add_numeric_literals left right))")
	subtract := compileJITExpressionTestProc(t, "(lambda (left right) (sql_sub_numeric_literals left right))")
	testSimplifiedLiteralArithmetic(t,
		func(a, b Scmer) Scmer { return Apply(add, a, b) },
		func(a, b Scmer) Scmer { return Apply(subtract, a, b) })
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
