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

import "fmt"
import "unsafe"
import "runtime"
import "strings"
import "testing"
import "github.com/launix-de/memcp/scm"

func compressedOpValues() []scm.Scmer {
	var values []scm.Scmer
	for _, f := range []StringFormat{1, 2, 3, 8, 9, 10, 11, 12, 13, 14, 15, 16, 19, 20} {
		alphabet := scm.CStringAlphabet(uint8(f))
		for _, text := range []string{"", "0", "00", alphabet, strings.Repeat(alphabet, 100)} {
			for offset := 0; offset < 2; offset++ {
				values = append(values, comparisonCString(text, f, offset))
			}
		}
	}
	for _, f := range []StringFormat{6, 7} {
		s := "550e8400-e29b-41d4-a716-446655440000"
		if f == 7 {
			s = strings.ToUpper(s)
		}
		values = append(values, comparisonCString(s, f, 0))
	}
	for _, s := range []string{"", "x", "xy", "xyz", "xyzw", "xyzwa", "\xfb\xff\xff", strings.Repeat("abcdef?\xff", 200)} {
		for _, url := range []bool{false, true} {
			for _, raw := range []bool{false, true} {
				values = append(values, scm.NewBString(unsafe.StringData(s), len(s), url, raw))
			}
		}
	}
	return values
}

func compressedOpOutcome(name string, args []scm.Scmer) (result string) {
	defer func() {
		if p := recover(); p != nil {
			result = "panic:" + fmt.Sprint(p)
			if name == "substr" {
				result = "panic"
			}
		}
	}()
	v := scm.Globalenv.Vars[scm.Symbol(name)].Func()(args...)
	return fmt.Sprintf("%t:%s", v.IsNil(), scm.String(v))
}

func TestCompressedStringOperators(t *testing.T) {
	for index, v := range compressedOpValues() {
		plain := scm.NewString(v.String())
		if got := scm.Eval(v, &scm.Globalenv); got.String() != plain.String() {
			t.Fatal("compressed literal")
		}
		check := func(name string, tail ...scm.Scmer) {
			t.Helper()
			a := append([]scm.Scmer{v}, tail...)
			b := append([]scm.Scmer{plain}, tail...)
			got, want := compressedOpOutcome(name, a), compressedOpOutcome(name, b)
			if got != want {
				t.Fatalf("value %d %s: got %q want %q", index, name, got, want)
			}
		}
		for _, name := range []string{"string?", "strlen", "toLower", "toUpper", "strtrim", "strltrim", "strrtrim", "sql_trim", "sql_ltrim", "sql_rtrim", "fnv_hash", "stable_structural_hash", "md5", "sha1", "sha256", "hex2bin", "bin2hex", "base64_encode", "base64_decode"} {
			check(name)
		}
		check("stable_structural_hash", scm.NewBool(true))
		check("concat", scm.NewString("!"), v)
		check("sql_concat", scm.NewString("!"), v)
		check("concat", scm.NewNil())
		for _, start := range []int{-1, 0, 1, 2, 3, len(plain.String())} {
			for _, n := range []int{-1, 0, 1, 2, 5} {
				check("substr", scm.NewInt(int64(start)), scm.NewInt(int64(n)))
				check("sql_substr", scm.NewInt(int64(start)), scm.NewInt(int64(n)))
			}
		}
		for _, pattern := range []string{"", "%", "_", "%a%", "%Z%", "%=%", "ab%", "%00", "%_a%", "%\\_%", "K%"} {
			check("strlike", scm.NewString(pattern))
			check("strlike_cs", scm.NewString(pattern))
		}
		if text := plain.String(); len(text) > 800 {
			for _, start := range []int{251, 252, 253, 254, 255, 256, 511} {
				for _, n := range []int{3, 127, 128, 129} {
					check("strlike_cs", scm.NewString("%"+text[start:start+n]+"%"))
				}
			}
		}
	}
}

func TestCompressedStringViewLifetime(t *testing.T) {
	v := comparisonCString("abcdef012345", 11, 1)
	sub := scm.Globalenv.Vars["substr"].Func()(v, scm.NewInt(1), scm.NewInt(7))
	upper := scm.Globalenv.Vars["toUpper"].Func()(sub)
	v = scm.NewNil()
	sub = scm.NewNil()
	runtime.GC()
	if upper.String() != "BCDEF01" {
		t.Fatal(upper.String())
	}
}

var compressedOpsSink scm.Scmer

func BenchmarkCompressedStringOperators(b *testing.B) {
	for _, size := range []int{16, 2048} {
		text := strings.Repeat("abcdef0123456789", size/16)
		for _, representation := range []string{"plain", "cstring", "bstring"} {
			v := scm.NewString(text)
			if representation == "cstring" {
				v = comparisonCString(text, 11, 1)
			}
			if representation == "bstring" {
				v = scm.NewBString(unsafe.StringData(text), len(text), false, false)
			}
			for _, op := range []struct {
				name string
				tail []scm.Scmer
			}{
				{"string?", nil}, {"strlen", nil}, {"concat", []scm.Scmer{scm.NewString("!")}}, {"substr", []scm.Scmer{scm.NewInt(1), scm.NewInt(8)}},
				{"toUpper", nil}, {"strtrim", nil}, {"base64_encode", nil}, {"bin2hex", nil}, {"strlike_cs", []scm.Scmer{scm.NewString("%~%")}}, {"fnv_hash", nil}, {"stable_structural_hash", nil}, {"sha256", nil}, {"base64_decode", nil},
			} {
				b.Run(fmt.Sprintf("%s/%d/%s", representation, size, op.name), func(b *testing.B) {
					args := append([]scm.Scmer{v}, op.tail...)
					fn := scm.Globalenv.Vars[scm.Symbol(op.name)].Func()
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						compressedOpsSink = fn(args...)
					}
				})
			}
		}
	}
}

func TestCompressedStringGenericCompatibility(t *testing.T) {
	for _, v := range []scm.Scmer{scm.NewAny("text"), scm.NewString("text"), scm.NewBString(nil, 0, false, false)} {
		if !scm.Globalenv.Vars["string?"].Func()(v).Bool() {
			t.Fatal("string type rejected")
		}
	}
	for _, v := range []scm.Scmer{scm.NewNil(), scm.NewInt(1), scm.NewSymbol("text")} {
		if scm.Globalenv.Vars["string?"].Func()(v).Bool() {
			t.Fatal("nonstring accepted")
		}
	}
	stream := scm.NewAny(strings.NewReader("stream"))
	if got := scm.Globalenv.Vars["concat"].Func()(scm.NewString("a"), stream).String(); got != "astream" {
		t.Fatal(got)
	}
}
