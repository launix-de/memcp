/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
package storage

import "os"
import "fmt"
import "bytes"
import "strings"
import "testing"
import "encoding/hex"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// Encode explicitly selected IDs, independently of rebuild's format choice.
// Adjacent sentinel characters expose accidental comparisons outside a value.
func comparisonCString(text string, format StringFormat, offset int) scm.Scmer {
	if cs := nibbleCharsetFor(format); cs != nil {
		prefix := ""
		if offset == 1 {
			prefix = string(cs.enc[len(scm.CStringAlphabet(uint8(format)))-1])
		}
		data, _ := appendNibbles(nil, prefix+text+prefix+"0", cs, 0)
		return scm.NewCString(&data[0], uint8(format), uint8(offset), len(text))
	}
	data := compressNonNibble(nil, text, format)
	return scm.NewCString(&data[0], uint8(format), 0, len(text))
}

func TestCStringComparisonMatrix(t *testing.T) {
	type value struct {
		text string
		v    scm.Scmer
	}
	values := []value{}
	for _, f := range []StringFormat{1, 2, 3, 8, 9, 10, 11, 12, 13, 14, 15, 16} {
		alphabet := scm.CStringAlphabet(uint8(f))
		texts := []string{"", "0", "00", "000", "01", "10", alphabet, strings.Repeat("0", 129) + "1"}
		for _, c := range alphabet {
			texts = append(texts, string(c), "0"+string(c))
		}
		for _, text := range texts {
			for off := 0; off < 2; off++ {
				values = append(values, value{text, comparisonCString(text, f, off)})
			}
		}
	}
	for _, f := range []StringFormat{6, 7} {
		for _, text := range []string{"00000000-0000-0000-0000-000000000000", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"} {
			if f == 7 {
				text = strings.ToUpper(text)
			}
			values = append(values, value{text, comparisonCString(text, f, 0)})
		}
	}
	for _, text := range []string{"", "0", "a", "A", "ä", "K", "ſ", "aK", "aſ", "aä", "0\x00", "00000000-0000-0000-0000-000000000000"} {
		values = append(values, value{text, scm.NewString(text)})
	}
	for _, a := range values {
		if a.v.String() != a.text {
			t.Fatalf("decode %q got %q", a.text, a.v.String())
		}
		for _, b := range values {
			if scm.Equal(a.v, b.v) != (a.text == b.text) {
				t.Fatalf("Equal %q %q", a.text, b.text)
			}
			if scm.Equal(a.v, scm.NewString(b.text)) != (a.text == b.text) || scm.Equal(scm.NewString(a.text), b.v) != (a.text == b.text) {
				t.Fatalf("mixed Equal %q %q", a.text, b.text)
			}
			if scm.Less(a.v, scm.NewString(b.text)) != (a.text < b.text) || scm.Less(scm.NewString(a.text), b.v) != (a.text < b.text) {
				t.Fatalf("mixed Less %q %q", a.text, b.text)
			}

			if scm.Less(a.v, b.v) != (a.text < b.text) {
				t.Fatalf("Less %q %q", a.text, b.text)
			}
			if scm.EqualSQL(a.v, b.v).Bool() != strings.EqualFold(a.text, b.text) {
				t.Fatalf("EqualSQL %q %q", a.text, b.text)
			}
		}
	}
}

func TestCStringCollationAndLike(t *testing.T) {
	like := scm.Globalenv.Vars[scm.Symbol("strlike")]
	texts := []string{"", "a", "A", "aa", "Aa", "ab", "aB", "0", "00", "01", "0A", strings.Repeat("a", 255) + "bc" + strings.Repeat("a", 300)}
	for _, f := range []StringFormat{2, 3, 11, 12} {
		for off := 0; off < 2; off++ {
			for _, text := range texts {
				if f == 3 || f == 12 {
					text = strings.ToUpper(text)
				} else {
					text = strings.ToLower(text)
				}
				v := comparisonCString(text, f, off)
				patterns := []string{"", "%", "%%", "_", "_%", "%_", "a%", "%a", "%ab%", "%bc%", "a%b%c", "%a_b%", "%a\\%", "%\\_%", "%ä%", "%K%", "%ſ%", "%" + strings.Repeat("a", 129) + "%"}
				for _, pattern := range patterns {
					for _, coll := range []string{"bin", "utf8mb4_general_ci"} {
						got := scm.Apply(like, v, scm.NewString(pattern), scm.NewString(coll)).Bool()
						cs := scm.Globalenv.Vars[scm.Symbol("strlike_cs")]
						if scm.Apply(cs, v, scm.NewString(pattern), scm.NewString(coll)).Bool() != scm.StrLike(text, pattern) {
							t.Fatal("strlike_cs changed case semantics")
						}
						if want := scm.StrLikeCollation(text, pattern, coll); got != want {
							t.Fatalf("LIKE %q %q %s got %v want %v", text, pattern, coll, got, want)
						}
					}
				}
				for _, coll := range []string{"bin", "utf8mb4_bin", "utf8mb4_general_ci", "utf8mb4_german2_ci"} {
					for _, reverse := range []bool{false, true} {
						fn := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(coll), scm.NewBool(reverse))
						less := scm.OrderRelationLess(fn.Func())
						for _, other := range append(texts, "ä", "aä", "aK") {
							plain := scm.NewString(text)
							b := scm.NewString(other)
							if less(v, b) != less(plain, b) || less(b, v) != less(b, plain) {
								t.Fatalf("collate %s reverse=%v %q %q", coll, reverse, text, other)
							}
						}
					}
				}
			}
		}
	}
}

func TestCStringLengthAndSubstring(t *testing.T) {
	for _, f := range []StringFormat{2, 11, 6} {
		text := "abcdef0123456789"
		if f == 6 {
			text = "550e8400-e29b-41d4-a716-446655440000"
		}
		v := comparisonCString(text, f, 0)
		if got := scm.Apply(scm.Globalenv.Vars[scm.Symbol("strlen")], v).Int(); got != int64(len(text)) {
			t.Fatal(got)
		}
		fn := scm.Globalenv.Vars[scm.Symbol("substr")]
		for i := 0; i <= len(text); i++ {
			for j := i; j <= len(text); j++ {
				got := scm.Apply(fn, v, scm.NewInt(int64(i)), scm.NewInt(int64(j-i))).String()
				if got != text[i:j] {
					t.Fatalf("substring %d:%d got %q", i, j, got)
				}
			}
		}
		sqlfn := scm.Globalenv.Vars[scm.Symbol("sql_substr")]
		for start := -2; start <= len(text)+2; start++ {
			for _, n := range []int{-1, 0, 1, 7, 100} {
				got := scm.Apply(sqlfn, v, scm.NewInt(int64(start)), scm.NewInt(int64(n))).String()
				want := scm.Apply(sqlfn, scm.NewString(text), scm.NewInt(int64(start)), scm.NewInt(int64(n))).String()
				if got != want {
					t.Fatalf("sql_substr %d %d got %q want %q", start, n, got, want)
				}
			}
		}

		for _, bounds := range [][2]int{{-1, 1}, {len(text) + 1, 0}, {1, -1}, {0, len(text) + 1}} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("invalid bounds accepted: %v", bounds)
					}
				}()
				scm.Apply(fn, v, scm.NewInt(int64(bounds[0])), scm.NewInt(int64(bounds[1])))
			}()
		}
	}
}

// Golden V1 bytes were written by ae5c20c5d, before ordered formats existed.
// V0 and the historical raw header use that same unchanged column body.
func TestCStringLegacyDiskFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/cstring-legacy.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Fixtures map[string]struct {
			Values []string
			Hex    string
			Format uint8
		}
	}
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	fixtures := document.Fixtures
	for name, f := range fixtures {
		t.Run(name, func(t *testing.T) {
			raw, err := hex.DecodeString(f.Hex)
			if err != nil {
				t.Fatal(err)
			}
			var column StorageString
			column.Deserialize(bytes.NewReader(raw[1:]))
			if uint8(column.format) != f.Format {
				t.Fatal("format reassigned")
			}
			verify := func(c *StorageString) {
				rangeValues := make([]scm.Scmer, len(f.Values))
				c.GetValueRange(0, uint32(len(f.Values)), rangeValues, 1)
				ids := make([]uint32, len(f.Values))
				for i := range ids {
					ids[i] = uint32(len(ids) - 1 - i)
				}
				multiValues := make([]scm.Scmer, len(f.Values))
				c.GetValueMulti(ids, multiValues, 1)
				for i, want := range f.Values {
					if rangeValues[i].String() != want || multiValues[len(f.Values)-1-i].String() != want {
						t.Fatalf("legacy bulk row %d", i)
					}
				}
				for i, want := range f.Values {
					v := c.GetValue(uint32(i))
					if v.String() != want || !scm.Equal(v, scm.NewString(want)) {
						t.Fatalf("row %d got %q want %q", i, v.String(), want)
					}
				}
			}
			verify(&column)
			var out bytes.Buffer
			column.Serialize(&out)
			if out.Bytes()[2] != f.Format {
				t.Fatal("serialization must not transcode loaded formats")
			}
			var reread StorageString
			reread.Deserialize(bytes.NewReader(out.Bytes()[1:]))
			verify(&reread)
			rebuilt := buildStringColumn(f.Values)
			verify(rebuilt)
			if isNibbleFormat(column.format) && rebuilt.format < 11 {
				t.Fatal("rebuild did not choose ordered format")
			}
		})
	}
	for _, format := range []byte{17, 48, 50, 255} {
		t.Run(fmt.Sprint(format), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("unknown format accepted")
				}
			}()
			var c StorageString
			c.Deserialize(bytes.NewReader([]byte{0, format, 2, 0, 0, 0, 0}))
		})
	}
}
