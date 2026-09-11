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
)

// A variable-width greedy repeat lowers to a native byte walk either way now:
// when the body provably cannot swallow the continuation's leading delimiter
// byte (the backslash-escaped quoted-string / block-comment token shapes),
// emitComplexTailRepeat commits without a stack; a doubled-delimiter escape
// ('', ``) needs emitBacktrackingRepeat's real position-backtracking stack
// (exercised against actual JIT-compiled code, not just this classification,
// by TestJITRegexBacktrackingRepeat).
func TestJITRegexVariableWidthRepeat(t *testing.T) {
	cases := []struct {
		name       string
		pat        string
		wantNative bool
		inputs     []string
	}{
		{"sqstr-backslash", `'(?:\\.|[^'\\])*'`, true,
			[]string{`'abc' x`, `'a\'b' y`, `'unterminated`, `''`, `not a string`}},
		{"dqstr-backslash", `"(?:\\.|[^"\\])*"`, true,
			[]string{`"abc"`, `"a\"b"`, `"unterminated`}},
		{"block-comment", `/\*(?s:.*?)\*/`, true,
			[]string{`/* a */`, `/* * / */ x`, `/* unterminated`, `/ not`}},
		{"number", `[0-9]+(?:\.[0-9]*)?(?:[eE][+-]?[0-9]+)?`, true,
			[]string{`12`, `12.5`, `12.5e-3`, `12.e9`, `x`}},
		{"sqstr-doubled", `'(?:''|[^'])*'`, true,
			[]string{`'a''b'`, `'ab'cd'`, `'unterminated`}},
		{"backtick-doubled", "`(?:``|[^`])*`", true,
			[]string{"`a``b`", "`a`b`", "`unterminated"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			re := regexp.MustCompile("^(?:" + tc.pat + ")")
			program, goRegex := jitCompileRegexProgramOrGo(re)
			if tc.wantNative && program == nil {
				t.Fatalf("expected native lowering, got Go regexp fallback")
			}
			if !tc.wantNative && (program != nil || goRegex == nil) {
				t.Fatalf("expected Go regexp fallback, got native program")
			}
			for _, in := range tc.inputs {
				got := jitConstantRegexpTest(NewRegex(re), NewString(in)).Bool()
				if want := re.MatchString(in); got != want {
					t.Errorf("%q on %q: got %v, want %v", tc.pat, in, got, want)
				}
			}
		})
	}
}
