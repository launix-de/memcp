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

import "testing"

func TestRoundSQLDecimalOutput(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		scale int
		want  float64
	}{
		{name: "addition residue", value: 0.30000000000000004, scale: 2, want: 0.3},
		{name: "positive half below boundary", value: 62668.72499999999, scale: 2, want: 62668.73},
		{name: "negative half above boundary", value: -1.0049999999999999, scale: 2, want: -1.01},
		{name: "ordinary value below half", value: 1.004, scale: 2, want: 1.0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := roundSQLDecimalOutput(test.value, test.scale); got != test.want {
				t.Fatalf("roundSQLDecimalOutput(%v, %d) = %v, want %v", test.value, test.scale, got, test.want)
			}
		})
	}
}
