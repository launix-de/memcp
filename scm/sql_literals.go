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
	"strings"
)

func declareSQLLiteralParameterizer() {
	Declare(&Globalenv, &Declaration{
		Name: "parameterize_sql_select_literals",
		Desc: "replaces safe literals in a top-level MySQL SELECT with positional runtime bindings",
		Fn: func(a ...Scmer) Scmer {
			normalized, bindings := parameterizeSQLSelectLiterals(String(a[0]))
			return NewSlice([]Scmer{NewString(normalized), NewSlice(bindings)})
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{{Kind: "string", ParamName: "query"}},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,
		},
	})
}

func parameterizeSQLSelectLiterals(query string) (string, []Scmer) {
	if !parameterizableSelectPrefix(query) {
		return query, nil
	}

	var out strings.Builder
	out.Grow(len(query))
	bindings := make([]Scmer, 0, 8)
	depth := 0
	typeDepth := -1
	orderDepth := -1
	previousWord := ""
	previousToken := ""
	seenSelect := false
	clause := ""

	for i := 0; i < len(query); {
		if isSQLSpace(query[i]) {
			out.WriteByte(query[i])
			i++
			continue
		}
		if query[i] == '#' || (query[i] == '-' && i+1 < len(query) && query[i+1] == '-') {
			end := i + 1
			for end < len(query) && query[end] != '\n' {
				end++
			}
			out.WriteString(query[i:end])
			i = end
			continue
		}
		if query[i] == '/' && i+1 < len(query) && query[i+1] == '*' {
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return query, nil
			}
			end += i + 4
			out.WriteString(query[i:end])
			i = end
			continue
		}
		if query[i] == '`' {
			end, ok := scanQuotedSQL(query, i, '`')
			if !ok {
				return query, nil
			}
			out.WriteString(query[i:end])
			i = end
			previousToken = "identifier"
			continue
		}
		if query[i] == '\'' || query[i] == '"' {
			end, ok := scanQuotedSQL(query, i, query[i])
			if !ok || previousWord == "AS" || previousWord == "DATE" || !parameterizableLiteralClause(clause) {
				if !ok {
					return query, nil
				}
				out.WriteString(query[i:end])
				i = end
				previousToken = "literal"
				continue
			}
			out.WriteByte('?')
			bindings = append(bindings, NewString(unescapeSQLLiteral(query[i+1:end-1])))
			i = end
			previousToken = "literal"
			continue
		}
		if query[i] == '?' {
			return query, nil
		}
		if isSQLIdentifierStart(query[i]) {
			end := i + 1
			for end < len(query) && isSQLIdentifierPart(query[end]) {
				end++
			}
			word := strings.ToUpper(query[i:end])
			if word == "SELECT" {
				if seenSelect {
					return query, nil
				}
				seenSelect = true
				clause = "SELECT"
			}
			if unsafeParameterizedSelectWord(word) {
				return query, nil
			}
			if word == "BY" && previousWord == "ORDER" {
				orderDepth = depth
			} else if orderDepth >= 0 && depth == orderDepth && (word == "LIMIT" || word == "FOR") {
				orderDepth = -1
			}
			if depth == 0 {
				switch word {
				case "FROM", "WHERE", "LIMIT", "OFFSET":
					clause = word
				case "ON":
					clause = "WHERE"
				}
			}
			out.WriteString(query[i:end])
			i = end
			previousWord = word
			previousToken = word
			continue
		}
		if query[i] == '(' {
			depth++
			if previousWord == "DECIMAL" || previousWord == "VARCHAR" {
				typeDepth = depth
			}
			out.WriteByte(query[i])
			i++
			previousToken = "("
			continue
		}
		if query[i] == ')' {
			if depth == typeDepth {
				typeDepth = -1
			}
			depth--
			out.WriteByte(query[i])
			i++
			previousToken = ")"
			continue
		}
		if query[i] == '-' && i+1 < len(query) && query[i+1] >= '0' && query[i+1] <= '9' && unarySQLSign(previousToken) && parameterizableLiteralClause(clause) {
			end := scanSQLNumber(query, i+1)
			if !isSQLIdentifierPartAt(query, end) {
				out.WriteByte('?')
				bindings = append(bindings, Simplify(query[i:end]))
				i = end
				previousToken = "literal"
				continue
			}
		}
		if isSQLNumberStart(query, i) {
			end := scanSQLNumber(query, i)
			if end > i && !isSQLIdentifierPartAt(query, end) {
				isOrderOrdinal := orderDepth == depth && (previousWord == "BY" || previousToken == ",")
				if typeDepth >= 0 || isOrderOrdinal || !parameterizableLiteralClause(clause) {
					out.WriteString(query[i:end])
				} else {
					out.WriteByte('?')
					bindings = append(bindings, Simplify(query[i:end]))
				}
				i = end
				previousToken = "literal"
				continue
			}
		}

		out.WriteByte(query[i])
		previousToken = query[i : i+1]
		i++
	}

	if len(bindings) == 0 {
		return query, nil
	}
	return out.String(), bindings
}

func parameterizableSelectPrefix(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "SELECT\n") || strings.HasPrefix(upper, "SELECT\t") {
		return true
	}
	if !strings.HasPrefix(upper, "EXPLAIN ") {
		return false
	}
	rest := strings.TrimSpace(upper[len("EXPLAIN "):])
	if strings.HasPrefix(rest, "COMPILE ") {
		return false
	}
	for _, modifier := range []string{"IR ", "REORDER "} {
		if strings.HasPrefix(rest, modifier) {
			rest = strings.TrimSpace(rest[len(modifier):])
		}
	}
	return strings.HasPrefix(rest, "SELECT ") || rest == "SELECT"
}

func scanQuotedSQL(query string, start int, quote byte) (int, bool) {
	for i := start + 1; i < len(query); i++ {
		if query[i] == '\\' {
			i++
			continue
		}
		if query[i] == quote {
			if quote == '`' && i+1 < len(query) && query[i+1] == '`' {
				i++
				continue
			}
			return i + 1, true
		}
	}
	return len(query), false
}

func unescapeSQLLiteral(value string) string {
	var out strings.Builder
	out.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			out.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case '0':
			out.WriteByte(0)
		case '\\', '\'', '"':
			out.WriteByte(value[i])
		default:
			out.WriteByte('\\')
			out.WriteByte(value[i])
		}
	}
	return out.String()
}

func scanSQLNumber(query string, start int) int {
	i := start
	for i < len(query) && query[i] >= '0' && query[i] <= '9' {
		i++
	}
	if i < len(query) && query[i] == '.' {
		i++
		for i < len(query) && query[i] >= '0' && query[i] <= '9' {
			i++
		}
	}
	if i < len(query) && (query[i] == 'e' || query[i] == 'E') {
		exponent := i
		i++
		if i < len(query) && (query[i] == '+' || query[i] == '-') {
			i++
		}
		digits := i
		for i < len(query) && query[i] >= '0' && query[i] <= '9' {
			i++
		}
		if i == digits {
			return exponent
		}
	}
	return i
}

func isSQLNumberStart(query string, i int) bool {
	if i >= len(query) || (query[i] < '0' || query[i] > '9') {
		return false
	}
	return i == 0 || !isSQLIdentifierPart(query[i-1])
}

func isSQLIdentifierStart(ch byte) bool {
	return ch == '_' || ch == '$' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z'
}

func isSQLIdentifierPart(ch byte) bool {
	return isSQLIdentifierStart(ch) || ch >= '0' && ch <= '9'
}

func isSQLIdentifierPartAt(query string, i int) bool {
	return i < len(query) && isSQLIdentifierPart(query[i])
}

func isSQLSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func unarySQLSign(previousToken string) bool {
	switch previousToken {
	case "", "(", ",", "=", ">", "<", ">=", "<=", "<>", "!=", "+", "-", "*", "/", "WHERE", "ON", "LIMIT", "OFFSET", "AND", "OR", "THEN", "ELSE":
		return true
	default:
		return false
	}
}

func parameterizableLiteralClause(clause string) bool {
	return clause == "WHERE" || clause == "LIMIT" || clause == "OFFSET"
}

func unsafeParameterizedSelectWord(word string) bool {
	switch word {
	case "GROUP", "HAVING", "UNION", "OR", "DISTINCT", "OVER",
		"COUNT", "SUM", "AVG", "MIN", "MAX", "GROUP_CONCAT":
		return true
	default:
		return false
	}
}
