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

func TestStrLikeCollationASCIIFastPaths(t *testing.T) {
	tests := []struct {
		value   string
		pattern string
		want    bool
	}{
		{"A,C", "%a%", true},
		{"A,C", "%b%", false},
		{"Alpha", "a%", true},
		{"Alpha", "%HA", true},
		{"Alpha", "ALPHA", true},
		{"Straße", "%STRASSE%", false},
		{"Straße", "%straße%", true},
	}
	for _, test := range tests {
		if got := StrLikeCollation(test.value, test.pattern, "utf8mb4_general_ci"); got != test.want {
			t.Errorf("StrLikeCollation(%q, %q) = %v, want %v", test.value, test.pattern, got, test.want)
		}
	}
}

func TestStrLikeCollationASCIIFastPathAllocations(t *testing.T) {
	allocations := testing.AllocsPerRun(1000, func() {
		if !StrLikeCollation("A,C", "%a%", "utf8mb4_general_ci") {
			t.Fatal("ASCII contains pattern should match")
		}
	})
	if allocations != 0 {
		t.Fatalf("ASCII contains LIKE allocated %.2f objects per call, want 0", allocations)
	}
}

func BenchmarkStrLikeCollationDynamicCategory(b *testing.B) {
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if !StrLikeCollation("A,C", "%a%", "utf8mb4_general_ci") {
			b.Fatal("ASCII contains pattern should match")
		}
	}
}

func BenchmarkStrLikeCollationDynamicCategoryLegacy(b *testing.B) {
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if !StrLikeFold("A,C", "%a%") {
			b.Fatal("ASCII contains pattern should match")
		}
	}
}

func TestStrLikeEscapedWildcards(t *testing.T) {
	if !StrLike("_transient_sample", `\_transient\_%`) {
		t.Fatal("escaped underscores should match literal underscores")
	}
	if StrLike("xtransient_sample", `\_transient\_%`) {
		t.Fatal("escaped leading underscore must not act as a wildcard")
	}
	if !StrLike("discount%", `discount\%`) {
		t.Fatal("escaped percent should match a literal percent")
	}
}

func TestGeneralCIFoldCompare(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"Publish", "publish", 0},
		{"alpha", "BETA", -1},
		{"view_count", "VIEW_COUNTS", -1},
		{"Straße", "STRASSE", 1},
	}
	for _, test := range tests {
		got := generalCIFoldCompare(test.left, test.right)
		if got < 0 {
			got = -1
		} else if got > 0 {
			got = 1
		}
		if got != test.want {
			t.Fatalf("generalCIFoldCompare(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
	allocations := testing.AllocsPerRun(1000, func() {
		if generalCIFoldCompare("custom_color", "CUSTOM_COLOR") != 0 {
			t.Fatal("ASCII fold comparison changed equality")
		}
	})
	if allocations != 0 {
		t.Fatalf("ASCII general_ci comparison allocated %.2f objects per call, want 0", allocations)
	}
}
