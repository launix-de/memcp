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

import (
	"regexp"
	"testing"
	"unsafe"
)

func regexpMatchStrings(v Scmer) []string {
	s, _ := scmerSlice(v)
	o := make([]string, len(s))
	for i := range s {
		o[i] = String(s[i])
	}
	return o
}

// (regexp_matches s "<const>") lowers to a native scan that JIT-compiles and
// returns results equal to RE2's FindAllString.
func TestJITRegexpMatchesNative(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	cases := []string{`[a-z]+|[0-9]+|.`, `[ ]+|[a-zA-Z_$][a-zA-Z0-9_$]*|[0-9]+|.`, `x`, `[a-z_]+`}
	inputs := []string{"ab12 c", "SELECT x FROM t WHERE a = 5", "", "  yes_1  ", "()"}
	for _, pat := range cases {
		re := regexp.MustCompile(pat)
		f := Eval(Read("f", `(jit (eval (optimize (quote (lambda (s) (regexp_matches s "`+pat+`"))))))`), &Globalenv)
		if !(f.IsProc() && f.Proc() != nil && f.Proc().Compiled != nil) {
			t.Errorf("regexp_matches %q did NOT JIT-compile", pat)
		}
		for _, in := range inputs {
			got := regexpMatchStrings(Apply(f, NewString(in)))
			want := re.FindAllString(in, -1)
			if len(got) != len(want) {
				t.Fatalf("%q on %q: got %v want %v", pat, in, got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("%q on %q [%d]: %q != %q", pat, in, i, got[i], want[i])
				}
			}
		}
	}
}

// The escape-aware SQL tokenizer alternation (interpreter Fn == JIT fallback).
func TestJITRegexpMatchesTokenizer(t *testing.T) {
	tok := regexp.MustCompile(`[ \t\r\n]+|--[^\n]*|/\*(?s:.*?)\*/|` +
		"`(?:\\\\.|``|[^`\\\\])*`" +
		`|'(?:\\.|[^'\\])*'|[a-zA-Z_$][a-zA-Z0-9_$]*|[0-9]+(?:\.[0-9]*)?|(?s:.)`)
	in := `SELECT id, 'a\'b' FROM ` + "`t``x`" + ` WHERE n = 12.5 /* c */ -- e`
	got := regexpMatchStrings(jitConstantRegexpMatches(NewRegex(tok), NewString(in)))
	want := tok.FindAllString(in, -1)
	if len(got) != len(want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] %q != %q", i, got[i], want[i])
		}
	}
}

// Every match string is a view into the scanned input, not a copy.
func TestJITRegexpMatchesNoCopy(t *testing.T) {
	in := "alpha beta gamma"
	base := uintptr(unsafe.Pointer(unsafe.StringData(in)))
	v := jitConstantRegexpMatches(NewRegex(regexp.MustCompile(`[a-z]+`)), NewString(in))
	s, _ := scmerSlice(v)
	for i, m := range s {
		p := uintptr(unsafe.Pointer(unsafe.StringData(String(m))))
		if p < base || p >= base+uintptr(len(in)) {
			t.Errorf("match %d %q is not a view into the input backing", i, String(m))
		}
	}
}
