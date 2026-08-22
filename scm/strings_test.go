/*
Copyright (C) 2026  MemCP Contributors

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

func TestLikePatternNeedsCaseFold(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"%123-456%", false},
		{"%straße%", true},
		{"%CAFÉ%", true},
		{"%12_34%", true},
	}
	for _, test := range tests {
		if got := likePatternNeedsCaseFold(test.pattern); got != test.want {
			t.Errorf("likePatternNeedsCaseFold(%q) = %v, want %v", test.pattern, got, test.want)
		}
	}
}

func TestStrLikeCollationUncasedPattern(t *testing.T) {
	if !StrLikeCollation("ABC 123 xyz", "%123%", "utf8mb4_general_ci") {
		t.Fatal("numeric pattern should match mixed-case text")
	}
	if StrLikeCollation("ABC 124 xyz", "%123%", "utf8mb4_general_ci") {
		t.Fatal("numeric pattern should preserve non-matches")
	}
	if !StrLikeCollation("Straße 123", "%straße%", "utf8mb4_general_ci") {
		t.Fatal("cased pattern should retain case-insensitive matching")
	}
}
