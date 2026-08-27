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

import "testing"

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
