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
	"fmt"
	"regexp"
	"strconv"
	"testing"
)

var constantRegexInventory = []string{
	`(?:[^"])+`,
	`(\\.|[^\'])*`,
	`-?(?:[0-9]+\.?[0-9]*|\.[0-9]+)(?:e-?[0-9]+)?`,
	`-?[0-9]+`,
	`.*\.sql(?:\.gz)?$`,
	`[0-9]+`,
	`[a-zA-Z0-9_]+`,
	`[a-zA-Z_][a-zA-Z0-9_]*`,
	`^(.*)/[^/]+$`,
	`^[\r\n\t ]*ALTER DATABASE (?s:.*)\z`,
	`^[\r\n\t ]*ALTER FUNCTION (?s:.*)\z`,
	`^[\r\n\t ]*ALTER SCHEMA (?s:.*)\z`,
	`^[\r\n\t ]*ALTER SEQUENCE (?s:.*)\z`,
	`^[\r\n\t ]*ALTER STATISTICS (?s:.*)\z`,
	`^[\r\n\t ]*COMMENT ON EXTENSION (?s:.*)\z`,
	`^[\r\n\t ]*COPY (.*) FROM '([^']+)'\z`,
	`^[\r\n\t ]*CREATE EXTENSION (?s:.*)\z`,
	`^[\r\n\t ]*CREATE INDEX (?s:.*)\z`,
	`^[\r\n\t ]*CREATE SEQUENCE (?s:.*)\z`,
	`^[\r\n\t ]*CREATE STATISTICS (?s:.*)\z`,
	`^[\r\n\t ]*CREATE TRIGGER (?s:.*)\z`,
	`^[\r\n\t ]*CREATE UNIQUE INDEX (?s:.*)\z`,
	`^[^.]+\.(.*)_(.*?)_seq\z`,
	`^[a-zA-Z_][a-zA-Z0-9_]*$`,
	`(.*)_(.*?)_seq`,
	`(?:[^` + "`" + `]|` + "``" + `)+`,
	`(\\.|''|[^\'])*`,
	`(\\.|""|[^\"])*`,
	`0[xX](?:[0-9A-Fa-f]{2})*`,
	`^(?is:SELECT[\r\n\t ]+DISTINCT[\r\n\t ]+TABLESPACE_NAME.*FROM[\r\n\t ]+INFORMATION_SCHEMA\.FILES.*FILE_TYPE[\r\n\t ]*=[\r\n\t ]*'DATAFILE'.*)\z`,
	`^(?is:SELECT[\r\n\t ]+LOGFILE_GROUP_NAME.*FROM[\r\n\t ]+INFORMATION_SCHEMA\.FILES.*FILE_TYPE[\r\n\t ]*=[\r\n\t ]*'UNDO LOG'.*)\z`,
	`^/\*![0-9]+[\r\n\t ]+((?is:CREATE[\r\n\t ]+TRIGGER.*))[\r\n\t ]*\*/$`,
	`^/\*![0-9]+[\r\n\t ]+CREATE[\r\n\t ]*\*/[\r\n\t ]*/\*![0-9]+[\r\n\t ]+DEFINER=(?is:.*?)[\r\n\t ]*\*/[\r\n\t ]*/\*![0-9]+[\r\n\t ]+((?is:TRIGGER.*))[\r\n\t ]*\*/$`,
	`^\s*SELECT\b`,
	`(?s:.*)\bLIKE$`,
	`\b(?:LIKE|MATCH|JOIN|EXISTS)\b`,
	`^v[0-9]+$`,
	`^/sql/([^/]+)/(.*)$`,
	`^((?s:.*));\s*$`,
}

var nativeRegexCorpus = []string{
	"",
	"abc",
	"ABC_123",
	"-17",
	"-.25e-3",
	"0xCAFE",
	"a.sql",
	"dump.sql.gz",
	"dir/table_name_seq",
	"public.table_column_seq",
	"\n CREATE UNIQUE INDEX idx ON t (id)",
	"COPY public.t FROM '/tmp/data.csv'",
	"ALTER DATABASE app OWNER TO admin",
	"ALTER FUNCTION f() OWNER TO admin",
	"ALTER SCHEMA app OWNER TO admin",
	"ALTER SEQUENCE app.seq RESTART WITH 1",
	"ALTER STATISTICS app.stats OWNER TO admin",
	"COMMENT ON EXTENSION plpgsql IS 'x'",
	"CREATE EXTENSION pgcrypto",
	"CREATE INDEX idx ON t (id)",
	"CREATE SEQUENCE app.seq",
	"CREATE STATISTICS app.stats ON id FROM t",
	"CREATE TRIGGER trg BEFORE INSERT ON t EXECUTE FUNCTION f()",
	"SELECT DISTINCT TABLESPACE_NAME FROM INFORMATION_SCHEMA.FILES WHERE FILE_TYPE = 'DATAFILE'",
	"SELECT x FROM t WHERE x LIKE 'a%'",
	"/sql/example/SELECT%201",
	"SELECT 1; \t",
	"v123",
	"/*!50003 CREATE TRIGGER trg BEFORE INSERT ON t FOR EACH ROW SET @x=1 */",
	"quoted '' value",
	"quoted \\\" value",
	"Grüße",
}

func nativeRegexDifferentialCorpus() []string {
	corpus := append([]string(nil), nativeRegexCorpus...)
	alphabet := []rune("abcXYZ019_./'\"`-+*?! =;\r\n\tä世")
	state := uint64(0x709c0de)
	for range 192 {
		state = state*6364136223846793005 + 1
		length := int(state>>58) & 31
		value := make([]rune, length)
		for index := range value {
			state = state*6364136223846793005 + 1
			value[index] = alphabet[int(state%uint64(len(alphabet)))]
		}
		corpus = append(corpus, string(value))
	}
	return corpus
}

func compileNativeRegexpTest(t *testing.T, pattern string) Scmer {
	t.Helper()
	source := fmt.Sprintf("(lambda (value) (regexp_test value %s))", strconv.Quote(pattern))
	compiled := compileJITExpressionTestProc(t, source)
	requireNoDynamicJITCalls(t, compiled)
	if calls := compiled.Proc().Compiled.Coverage.NativeCalls; calls != 0 {
		t.Fatalf("constant regex emitted %d native builtin calls", calls)
	}
	return compiled
}

func TestJITNativeRegexSQLParserInventory(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	corpus := nativeRegexDifferentialCorpus()
	for _, pattern := range constantRegexInventory {
		t.Run(pattern, func(t *testing.T) {
			wantMatcher := regexp.MustCompile(pattern)
			compiled := compileNativeRegexpTest(t, pattern)
			for _, input := range corpus {
				got := Apply(compiled, NewString(input))
				want := wantMatcher.MatchString(input)
				if !got.IsBool() || got.Bool() != want {
					t.Fatalf("pattern %q input %q: JIT returned %s, want %t", pattern, input, String(got), want)
				}
			}
		})
	}
}

func TestJITNativeRegexpTestPreservesNil(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	compiled := compileNativeRegexpTest(t, `^[a-z]+$`)
	if got := Apply(compiled, NewNil()); !got.IsNil() {
		t.Fatalf("regexp_test(nil, pattern) returned %s, want nil", String(got))
	}
	compiled = compileNativeRegexpTest(t, `^42$`)
	if got := Apply(compiled, NewInt(42)); !got.IsBool() || !got.Bool() {
		t.Fatalf("regexp_test must preserve non-string conversion, got %s", String(got))
	}
}

func compileNativeRegexCaptures(t *testing.T, pattern string) Scmer {
	t.Helper()
	captureCount := regexp.MustCompile(pattern).NumSubexp() + 1
	variables := make([]string, captureCount)
	for index := range variables {
		variables[index] = fmt.Sprintf("capture%d", index)
	}
	source := fmt.Sprintf(
		"(lambda (value) (match value (regex %s %s) (list %s) _ false))",
		strconv.Quote(pattern), joinStrings(variables, " "), joinStrings(variables, " "))
	compiled := compileJITExpressionTestProc(t, source)
	requireNoDynamicJITCalls(t, compiled)
	return compiled
}

func joinStrings(values []string, separator string) string {
	result := ""
	for index, value := range values {
		if index != 0 {
			result += separator
		}
		result += value
	}
	return result
}

func TestJITNativeRegexCaptures(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	tests := []struct {
		pattern string
		inputs  []string
	}{
		{`^(.*)=(.*)$`, []string{"key=value", "=", "missing"}},
		{`(.*)_(.*?)_seq`, []string{"table_column_seq", "a_b_c_seq", "no"}},
		{`^[^.]+\.(.*)_(.*?)_seq\z`, []string{"public.table_column_seq", "table_column_seq"}},
		{`^(a)?(b*)$`, []string{"", "a", "bbb", "abbb", "x"}},
		{`^(?:(a)|(b))$`, []string{"a", "b", "x"}},
		{`^[\r\n\t ]*COPY (.*) FROM '([^']+)'\z`, []string{"COPY public.t FROM '/tmp/data.csv'", "COPY broken"}},
		{`^((?s:.*));\s*$`, []string{"SELECT 1;", "SELECT\n1; \t", "SELECT 1"}},
	}
	corpus := nativeRegexDifferentialCorpus()
	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			matcher := regexp.MustCompile(test.pattern)
			compiled := compileNativeRegexCaptures(t, test.pattern)
			inputs := append(append([]string(nil), test.inputs...), corpus...)
			for _, input := range inputs {
				want := matcher.FindStringSubmatch(input)
				got := Apply(compiled, NewString(input))
				if want == nil {
					if !got.IsBool() || got.Bool() {
						t.Fatalf("pattern %q input %q: JIT returned %s, want false", test.pattern, input, String(got))
					}
					continue
				}
				gotCaptures := got.Slice()
				if len(gotCaptures) != len(want) {
					t.Fatalf("pattern %q input %q: got %d captures, want %d", test.pattern, input, len(gotCaptures), len(want))
				}
				for index := range want {
					if !gotCaptures[index].IsString() || gotCaptures[index].String() != want[index] {
						t.Fatalf("pattern %q input %q capture %d: got %s, want %q", test.pattern, input, index, String(gotCaptures[index]), want[index])
					}
				}
			}
		})
	}
}
