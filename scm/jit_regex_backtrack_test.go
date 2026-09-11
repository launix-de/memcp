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
	"strings"
	"testing"
)

// schemePatternLiteral renders a Go regex-pattern string as a Scheme string
// literal. The Scheme reader collapses "\\" to "\" inside a string literal
// (see the doubled-backslash pitfall noted elsewhere in this package's
// tests), so every backslash in the pattern must be doubled in the source to
// come back out as one backslash in the runtime string value RE2 sees.
func schemePatternLiteral(pat string) string {
	return `"` + strings.ReplaceAll(pat, `\`, `\\`) + `"`
}

// A doubled-delimiter escape ('', ``) now lowers natively via
// emitBacktrackingRepeat instead of falling back to a Go regexp call - this
// exercises the real JIT-compiled machine code (jitCompileRegexProgramOrGo,
// checked by TestJITRegexVariableWidthRepeat, only classifies the pattern
// without running it).
func TestJITRegexBacktrackingRepeat(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	cases := []struct {
		name   string
		pat    string
		inputs []string
	}{
		{"sqstr-doubled", `^'(?:''|[^'])*'`,
			[]string{
				`'a''b'`, `'ab'cd'`, `'unterminated`, `''`, `'''`, `''''`,
				`'''''''''`, // 9 quotes: many consecutive backtrack pops
				`'a''''b'`, `not a string`, `''x''y''`,
			}},
		{"backtick-doubled", "^`(?:``|[^`])*`",
			[]string{
				"`a``b`", "`a`b`", "`unterminated", "``", "```", "````",
				"```````" /* 7 backticks */, "`a````b`", "no backticks",
			}},
		// mixed: backslash escape AND doubled-delimiter escape in the same
		// alternation, `#` closing - stresses the alt-choice-per-iteration
		// staying correct while the iteration *count* backtracks.
		{"mixed-escape", `^#(?:\\.|##|[^#\\])*#`,
			[]string{
				`#a##b#`, `#a\#b#`, `#a\\#`, `#unterminated`, `##`, `###`,
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			re := regexp.MustCompile(tc.pat)
			program, goRegex := jitCompileRegexProgramOrGo(re)
			if program == nil {
				t.Fatalf("expected native lowering (backtracking stack), got Go regexp fallback for %q", goRegex.String())
			}

			testFn := Eval(Read("f", `(jit (eval (optimize (quote (lambda (s) (regexp_test s `+schemePatternLiteral(tc.pat)+`))))))`), &Globalenv)
			if !(testFn.IsProc() && testFn.Proc() != nil && testFn.Proc().Compiled != nil) {
				t.Fatalf("regexp_test %q did NOT JIT-compile", tc.pat)
			}
			matchesFn := Eval(Read("f", `(jit (eval (optimize (quote (lambda (s) (regexp_matches s `+schemePatternLiteral(tc.pat)+`))))))`), &Globalenv)
			if !(matchesFn.IsProc() && matchesFn.Proc() != nil && matchesFn.Proc().Compiled != nil) {
				t.Fatalf("regexp_matches %q did NOT JIT-compile", tc.pat)
			}

			for _, in := range tc.inputs {
				wantTest := re.MatchString(in)
				gotTest := Apply(testFn, NewString(in)).Bool()
				if gotTest != wantTest {
					t.Errorf("regexp_test %q on %q: got %v, want %v", tc.pat, in, gotTest, wantTest)
				}

				wantMatches := re.FindAllString(in, -1)
				gotMatches := regexpMatchStrings(Apply(matchesFn, NewString(in)))
				if len(gotMatches) != len(wantMatches) {
					t.Fatalf("regexp_matches %q on %q: got %v want %v", tc.pat, in, gotMatches, wantMatches)
				}
				for i := range wantMatches {
					if gotMatches[i] != wantMatches[i] {
						t.Errorf("regexp_matches %q on %q [%d]: %q != %q", tc.pat, in, i, gotMatches[i], wantMatches[i])
					}
				}
			}
		})
	}
}

// The real SQL tokenizer alternation, doubled-quote/backtick escapes
// included, now lowers end to end (jit? true) instead of falling back.
func TestJITRegexBacktrackingTokenizer(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	pat := `[ \t\r\n]+|--[^\n]*|/\*(?s:.*?)\*/|` +
		"`(?:``|[^`])*`" +
		`|'(?:''|[^'])*'|[a-zA-Z_$][a-zA-Z0-9_$]*|[0-9]+(?:\.[0-9]*)?|(?s:.)`
	re := regexp.MustCompile(pat)
	program, goRegex := jitCompileRegexProgramOrGo(re)
	if program == nil {
		t.Fatalf("expected native lowering, got Go regexp fallback for %q", goRegex.String())
	}
	f := Eval(Read("f", `(jit (eval (optimize (quote (lambda (s) (regexp_matches s `+schemePatternLiteral(pat)+`))))))`), &Globalenv)
	if !(f.IsProc() && f.Proc() != nil && f.Proc().Compiled != nil) {
		t.Fatalf("regexp_matches tokenizer did NOT JIT-compile")
	}
	inputs := []string{
		`SELECT id, 'a''b' FROM ` + "`t``x`" + ` WHERE n = 12.5 /* c */ -- e`,
		`'unterminated`,
		"`unterminated",
		"'' " + "``" + " x",
	}
	for _, in := range inputs {
		want := re.FindAllString(in, -1)
		got := regexpMatchStrings(Apply(f, NewString(in)))
		if len(got) != len(want) {
			t.Fatalf("on %q: got %v\nwant %v", in, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("on %q [%d]: %q != %q", in, i, got[i], want[i])
			}
		}
	}
}
