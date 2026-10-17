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

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRegexpMatches(t *testing.T) {
	fn := Globalenv.Vars[Symbol("regexp_matches")]
	for _, test := range []struct {
		text, pattern string
		want          []Scmer
	}{
		{"one 23 ä", `[[:alpha:]]+|[0-9]+|ä`, []Scmer{NewString("one"), NewString("23"), NewString("ä")}},
		{"abc", `[0-9]+`, nil},
		{"", `.`, nil},
	} {
		got := Apply(fn, NewString(test.text), NewString(test.pattern))
		if !Equal(got, NewSlice(test.want)) {
			t.Fatalf("regexp_matches(%q, %q) = %v", test.text, test.pattern, got)
		}
		reader := NewSlice([]Scmer{NewSymbol("lambda"), NewSlice([]Scmer{NewSymbol("text")}),
			NewSlice([]Scmer{NewSymbol("regexp_matches"), NewSymbol("text"), NewString(test.pattern)})})
		compiled := Eval(Optimize(reader, &Globalenv, nil), &Globalenv)
		if got := Apply(compiled, NewString(test.text)); !Equal(got, NewSlice(test.want)) {
			t.Fatalf("precompiled regexp_matches(%q, %q) = %v", test.text, test.pattern, got)
		}
	}
	defer func() {
		if recover() == nil {
			t.Fatal("invalid regular expression must fail")
		}
	}()
	Apply(fn, NewString("abc"), NewString("["))
}

func TestBinaryCollationBooleanTextOrder(t *testing.T) {
	values := []Scmer{NewNil(), NewBool(false), NewBool(true),
		NewString(""), NewString("0"), NewString("1"), NewString("3"),
		NewString("false"), NewString("true"), NewString("view"), NewString("yes"),
		NewSymbol("true"), NewString("ä")}
	for _, name := range []string{"bin", "binary", "utf8", "utf8mb4", "utf8mb4_bin"} {
		for _, reverse := range []bool{false, true} {
			factory := Apply(Globalenv.Vars[Symbol("collate")], NewString(name), NewBool(reverse))
			less := OrderRelationLess(factory.Func())
			for i, a := range values {
				if less(a, a) {
					t.Fatalf("%s reverse=%v: irreflexivity at %d", name, reverse, i)
				}
				for j, b := range values {
					ab, ba := less(a, b), less(b, a)
					if ab && ba {
						t.Fatalf("%s reverse=%v: asymmetric order at %d,%d", name, reverse, i, j)
					}
					for k, c := range values {
						if ab && less(b, c) && !less(a, c) {
							t.Fatalf("%s reverse=%v: non-transitive order at %d,%d,%d", name, reverse, i, j, k)
						}
						if !ab && !ba && !less(b, c) && !less(c, b) && (less(a, c) || less(c, a)) {
							t.Fatalf("%s reverse=%v: non-transitive equivalence at %d,%d,%d", name, reverse, i, j, k)
						}
					}
				}
			}
		}
	}
}

func TestBinaryCollationPreservesNumericAndExpressionComparisons(t *testing.T) {
	// The index fix must not redefine coercing expression comparisons or
	// change the existing numeric/string search-key conversions.
	if !Less(NewBool(true), NewString("3")) {
		t.Fatal("coercing expression comparison changed")
	}
	values := []Scmer{NewNil(), NewInt(-1), NewInt(2), NewInt(10), NewFloat(2.5),
		NewString("2"), NewString("10"), NewDate(42)}
	for _, a := range values {
		for _, b := range values {
			if binaryCollationLess(a, b) != Less(a, b) {
				t.Fatalf("numeric/text compatibility changed for %v and %v", a, b)
			}
		}
	}
}

func TestUnicodeIndexCollationProvidesStableSortKeys(t *testing.T) {
	factory := Apply(Globalenv.Vars[Symbol("collate")],
		NewString("utf8mb4_unicode_ci"), NewBool(false))
	less := OrderRelationLess(factory.Func())
	left, right := NewString("alpha-9"), NewString("alpha-10")
	if !less(left, right) {
		t.Fatal("unicode collation order changed")
	}
	keyFn, reverse, ok := LookupCollationKey("utf8mb4_unicode_ci:asc")
	if !ok || reverse {
		t.Fatalf("unicode collation key lookup = (_, %t, %t), want (_, false, true)", reverse, ok)
	}
	leftKey, leftOK := keyFn(left)
	rightKey, rightOK := keyFn(right)
	if !leftOK || !rightOK || strings.Compare(leftKey, rightKey) >= 0 {
		t.Fatal("unicode collation keys do not preserve numeric ordering")
	}
}

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

func TestGeneralCIFoldCompareUnicode(t *testing.T) {
	values := []string{"", "A", "a", "École", "éCOLE", "Straße", "STRASSE", "İ", "i", "K", "K", "ſ", "S", "ẞ", "ß", "Ångström", "A\u030angström", "Σ", "σ", "ς", "中", "가", "😀", "𐐀", "𐐨", "\xff", "\xfe", "x\xc3", "x\ufffd", "ABC\xff", "abc\xfe"}
	for _, left := range values {
		for _, right := range values {
			want := strings.Compare(strings.ToLower(left), strings.ToLower(right))
			got := generalCIFoldCompare(left, right)
			if got != want {
				t.Fatalf("compare(%q,%q)=%d, want %d", left, right, got, want)
			}
		}
	}
	for _, pair := range [][2]string{{"École 123", "éCOLE 124"}, {"A\u030a", "Å"}, {"ABC\xff", "abc\ufffd"}, {strings.Repeat("ÉCOLE", 128), strings.Repeat("école", 128)}} {
		allocations := testing.AllocsPerRun(1000, func() { generalCIFoldCompare(pair[0], pair[1]) })
		if allocations != 0 {
			t.Errorf("compare(%q,%q) allocated %.2f objects, want 0", pair[0][:min(12, len(pair[0]))], pair[1][:min(12, len(pair[1]))], allocations)
		}
	}
}

func FuzzGeneralCIFoldCompare(f *testing.F) {
	for _, pair := range [][2]string{{"École", "éCOLE"}, {"ABC\xff", "abc\ufffd"}, {"K", "K"}, {"𐐀", "𐐨"}, {"ÉCOLE", "école"}} {
		f.Add(pair[0], pair[1])
	}
	f.Fuzz(func(t *testing.T, left, right string) {
		want := strings.Compare(strings.ToLower(left), strings.ToLower(right))
		if got := generalCIFoldCompare(left, right); got != want {
			t.Fatalf("compare(%q,%q)=%d, want %d", left, right, got, want)
		}
	})
}

func BenchmarkGeneralCIFoldCompare(b *testing.B) {
	for _, pair := range [][2]string{{"account-123", "ACCOUNT-124"}, {"École 123", "éCOLE 124"}, {"𐐀-123", "𐐨-124"}, {strings.Repeat("ÉCOLE", 128) + "1", strings.Repeat("école", 128) + "2"}} {
		b.Run(pair[0][:min(16, len(pair[0]))], func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if generalCIFoldCompare(pair[0], pair[1]) >= 0 {
					b.Fatal("unexpected ordering")
				}
			}
		})
	}
}

func TestComputeSizeBorrowedStrings(t *testing.T) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	os.Stdout = writer
	defer func() { os.Stdout = original }()
	data := []byte{0x12, 0x34}
	values := []Scmer{
		NewCString(&data[0], 11, 0, 4),
		NewCString(&data[0], 11, 1, 3),
		NewBString(&data[0], len(data), false, false),
		NewBString(&data[0], len(data), true, true),
		NewCString(nil, 0, 0, 0),
		NewBString(nil, 0, false, false),
	}
	for _, value := range values {
		if got := ComputeSize(value); got != scmerStructOverhead {
			t.Errorf("borrowed string tag %d: size %d, want slot size %d", value.GetTag(), got, scmerStructOverhead)
		}
	}
	writer.Close()
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(output) != 0 {
		t.Fatalf("valid string tags produced diagnostics: %s", output)
	}
}

func TestUTF16LengthAndPrefixContracts(t *testing.T) {
	for _, test := range []struct {
		text   string
		units  int64
		prefix []string
	}{
		{"", 0, []string{"", "", "", "", ""}},
		{"abcd", 4, []string{"", "a", "ab", "abc", "abcd"}},
		{"雪€", 2, []string{"", "雪", "雪€", "雪€", "雪€"}},
		{"a😀z", 4, []string{"", "a", "a", "a😀", "a😀z"}},
		{"😀x", 3, []string{"", "", "😀", "😀x", "😀x"}},
		{"e\u0301", 2, []string{"", "e", "e\u0301", "e\u0301", "e\u0301"}},
	} {
		buffer, offsets := BuildCompressedStringBuffer([]string{test.text})
		for _, value := range []Scmer{NewString(test.text), NewBufferString(buffer, offsets[0], len(test.text))} {
			if got := declarations["utf16_length"].Fn(value); !got.IsInt() || got.Int() != test.units {
				t.Fatalf("UTF16 length of %q = %v, want %d", test.text, got, test.units)
			}
			for limit, want := range test.prefix {
				if got := declarations["utf16_prefix"].Fn(value, NewInt(int64(limit))); got.String() != want {
					t.Fatalf("UTF16 prefix of %q at %d = %q, want %q", test.text, limit, got.String(), want)
				}
			}
			if got := declarations["utf16_prefix"].Fn(value, NewInt(9223372036854775807)); got.String() != test.text {
				t.Fatalf("unbounded prefix changed %q", test.text)
			}
		}
	}
	for _, name := range []string{"utf16_length", "utf16_prefix"} {
		if got := declarations[name].Fn(NewNil(), NewInt(2)); !got.IsNil() {
			t.Fatalf("%s lost nil", name)
		}
	}
	if got := declarations["utf16_prefix"].Fn(NewString("x"), NewNil()); !got.IsNil() {
		t.Fatal("nil prefix limit did not propagate")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("negative prefix limit was accepted")
		}
	}()
	declarations["utf16_prefix"].Fn(NewString("x"), NewInt(-1))
}

func TestUTF16CompiledStringContracts(t *testing.T) {
	length := CompileJIT(NewProcStruct(*calibrationProcedure("(lambda (text) (utf16_length text))")), true)
	prefix := CompileJIT(NewProcStruct(*calibrationProcedure("(lambda (text limit) (utf16_prefix text limit))")), true)
	text := strings.Repeat("雪😀a", 9000)
	buffer, offsets := BuildCompressedStringBuffer([]string{text})
	for _, value := range []Scmer{NewString(text), NewBufferString(buffer, offsets[0], len(text))} {
		if got := Apply(length, value); !got.IsInt() || got.Int() != 36000 {
			t.Fatalf("compiled Unicode unit count = %v", got)
		}
		if got := Apply(prefix, value, NewInt(2)); got.String() != "雪" {
			t.Fatalf("compiled prefix split supplementary scalar: %q", got.String())
		}
		if got := Apply(prefix, value, NewInt(4)); got.String() != "雪😀a" {
			t.Fatalf("compiled prefix lost complete scalars: %q", got.String())
		}
	}
	if !Apply(length, NewNil()).IsNil() || !Apply(prefix, NewString("x"), NewNil()).IsNil() {
		t.Fatal("compiled Unicode helpers lost nil")
	}
}

// Keep the generated hash loop equivalent to the standard FNV-1a contract,
// including the two compressed-text branches around the 128-byte boundary.
func fnvHashContractInputs() []Scmer {
	texts := []string{"", "a", "hello", "\x00\xff\x80", "äΣK", strings.Repeat("x", 127), strings.Repeat("x", 128), strings.Repeat("äΣ", 1024)}
	buffer, offsets := BuildCompressedStringBuffer(texts)
	values := []Scmer{NewNil(), NewBool(true), NewInt(-9223372036854775808), NewFloat(1.25)}
	for i, text := range texts {
		values = append(values, NewString(text), NewBufferString(buffer, offsets[i], len(text)))
	}
	return values
}

func fnvHashContractWant(value Scmer) Scmer {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(String(value)))
	return NewString(fmt.Sprintf("%016x", hash.Sum64()))
}

func TestFNVHashContract(t *testing.T) {
	for i, value := range fnvHashContractInputs() {
		if got, want := declarations["fnv_hash"].Fn(value), fnvHashContractWant(value); !Equal(got, want) {
			t.Fatalf("value %d (%v): got %s, want %s", i, value.GetTag(), String(got), String(want))
		}
	}
}

// Tests compile serially. Restore admission before executing the compiled code,
// including every panic/Goexit path; production costs and admission stay intact.
func compileFNVHashContract(t *testing.T, source string, forceBody bool) Scmer {
	t.Helper()
	if !jitEnabled {
		t.Skip("requires native JIT execution")
	}
	declaration := declarations["fnv_hash"]
	previous := declaration.RetainsCallArgs
	defer func() { declaration.RetainsCallArgs = previous }()
	if forceBody {
		declaration.RetainsCallArgs = true
	}
	compiled := compileJITExpressionTestProc(t, source)
	requireNoDynamicJITCalls(t, compiled)
	if forceBody && compiled.Proc().Compiled.Coverage.InlinedCalls == 0 {
		t.Fatal("hash contract did not exercise the generated inline body")
	}
	return compiled
}

func TestJITGeneratedFNVHashContract(t *testing.T) {
	for _, mode := range []struct {
		name      string
		forceBody bool
	}{{"boundary", false}, {"body", true}} {
		t.Run(mode.name, func(t *testing.T) {
			compiled := compileFNVHashContract(t, `(lambda (value) (fnv_hash value))`, mode.forceBody)
			for i, value := range fnvHashContractInputs() {
				if got, want := Apply(compiled, value), fnvHashContractWant(value); !Equal(got, want) {
					t.Fatalf("value %d: got %s, want %s", i, String(got), String(want))
				}
			}
		})
	}
}

func TestJITGeneratedFNVHashPreservesLiveValues(t *testing.T) {
	for _, mode := range []struct {
		name      string
		forceBody bool
	}{{"boundary", false}, {"body", true}} {
		t.Run(mode.name, func(t *testing.T) {
			compiled := compileFNVHashContract(t, `(lambda (value a b c d e f g h)
				(list a b c d (fnv_hash value) e f g h (fnv_hash value) a b c d e f g h))`, mode.forceBody)
			live := []Scmer{NewInt(-7), NewString("retained"), NewFloat(3.5), NewBool(true), NewNil(), NewInt(9223372036854775807), NewString("last"), NewBool(false)}
			for i, value := range fnvHashContractInputs() {
				hash := fnvHashContractWant(value)
				want := append([]Scmer{}, live[:4]...)
				want = append(want, hash)
				want = append(want, live[4:]...)
				want = append(want, hash)
				want = append(want, live...)
				args := append([]Scmer{value}, live...)
				if got := Apply(compiled, args...); !Equal(got, NewSlice(want)) {
					t.Fatalf("value %d: hash call changed live values: got %s, want %s", i, String(got), String(NewSlice(want)))
				}
			}
		})
	}
}
