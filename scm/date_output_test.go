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

func TestSQLTemporalOutputDateFromGenericRepresentations(t *testing.T) {
	const unix = int64(1718451045)
	for _, value := range []Scmer{NewDate(unix), NewInt(unix), NewFloat(float64(unix)), NewString("2024-06-15 10:30:45")} {
		if got := sqlTemporalOutput(value, "DATE").String(); got != "2024-06-15" {
			t.Fatalf("DATE output = %q, want 2024-06-15", got)
		}
	}
}

func TestSQLTemporalOutputPreservesNilAndUnknownType(t *testing.T) {
	if got := sqlTemporalOutput(NewNil(), "DATE"); !got.IsNil() {
		t.Fatalf("DATE NULL output = %v, want nil", got)
	}
	value := NewInt(42)
	if got := sqlTemporalOutput(value, "VARCHAR"); got != value {
		t.Fatalf("unknown temporal type changed value: got %v, want %v", got, value)
	}
}
