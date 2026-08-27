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

type sqlShapeBuilder struct {
	text strings.Builder
	hash uint64
}

type sqlSelectScope struct {
	depth      int
	clause     string
	orderDepth int
	unsafe     bool
	derived    bool
}

type sqlLiteralCandidate struct {
	start int
	end   int
	value Scmer
	scope *sqlSelectScope
	force bool
}

func newSQLShapeBuilder(capacity int) *sqlShapeBuilder {
	result := &sqlShapeBuilder{hash: fnv64Offset}
	result.text.Grow(capacity)
	return result
}

func (builder *sqlShapeBuilder) WriteByte(value byte) {
	builder.text.WriteByte(value)
	builder.hash = (builder.hash ^ uint64(value)) * fnv64Prime
}

func (builder *sqlShapeBuilder) WriteString(value string) {
	builder.text.WriteString(value)
	for i := 0; i < len(value); i++ {
		builder.hash = (builder.hash ^ uint64(value[i])) * fnv64Prime
	}
}

func (builder *sqlShapeBuilder) Result() (string, string) {
	return builder.text.String(), formatStructuralHash(builder.hash)
}

func declareSQLLiteralParameterizer() {
	Declare(&Globalenv, &Declaration{
		Name: "parameterize_sql_select_literals",

		Fn: func(a ...Scmer) Scmer {
			normalized, bindings, shapeHash := parameterizeSQLSelectLiterals(String(a[0]))
			return NewSlice([]Scmer{NewString(normalized), NewSlice(bindings), NewString(shapeHash)})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "replaces safe literals in a top-level MySQL SELECT and returns normalized SQL, positional runtime bindings, and its stable shape hash",
			Params: []*TypeDescriptor{{Kind: "string", Label: "query"}},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,
		},
	})
}

func parameterizeSQLSelectLiterals(query string) (string, []Scmer, string) {
	if !parameterizableSelectPrefix(query) {
		return query, nil, fnvHashString(query)
	}

	out := newSQLShapeBuilder(len(query))
	bindings := make([]Scmer, 0, 8)
	depth := 0
	typeDepth := -1
	previousWord := ""
	previousToken := ""
	structuralQuery := false
	selectScopes := make([]*sqlSelectScope, 0, 4)
	candidates := make([]sqlLiteralCandidate, 0, 8)

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
				return query, nil, fnvHashString(query)
			}
			end += i + 4
			out.WriteString(query[i:end])
			i = end
			continue
		}
		if query[i] == '`' {
			end, ok := scanQuotedSQL(query, i, '`')
			if !ok {
				return query, nil, fnvHashString(query)
			}
			out.WriteString(query[i:end])
			i = end
			previousToken = "identifier"
			continue
		}
		if query[i] == '\'' || query[i] == '"' {
			end, ok := scanQuotedSQL(query, i, query[i])
			clause := ""
			var scope *sqlSelectScope
			if len(selectScopes) > 0 {
				scope = selectScopes[len(selectScopes)-1]
				clause = scope.clause
			}
			forcedPattern := scope != nil && scope.depth > 0 && (previousWord == "LIKE" || previousWord == "AGAINST")
			parameterizable := parameterizableLiteralAtScope(clause, scope) || forcedPattern
			if !ok || previousWord == "AS" || previousWord == "DATE" || !parameterizable {
				if !ok {
					return query, nil, fnvHashString(query)
				}
				out.WriteString(query[i:end])
				i = end
				previousToken = "literal"
				continue
			}
			out.WriteByte('?')
			value := NewString(unescapeSQLLiteral(query[i+1 : end-1]))
			bindings = append(bindings, value)
			candidates = append(candidates, sqlLiteralCandidate{i, end, value, selectScopes[len(selectScopes)-1], forcedPattern})
			i = end
			previousToken = "literal"
			continue
		}
		if query[i] == '?' {
			return query, nil, fnvHashString(query)
		}
		if isSQLIdentifierStart(query[i]) {
			end := i + 1
			for end < len(query) && isSQLIdentifierPart(query[end]) {
				end++
			}
			word := strings.ToUpper(query[i:end])
			if word == "OVER" {
				structuralQuery = true
			}
			if word == "SELECT" {
				if len(selectScopes) > 0 && selectScopes[len(selectScopes)-1].depth == depth {
					selectScopes[len(selectScopes)-1].clause = "SELECT"
					selectScopes[len(selectScopes)-1].orderDepth = -1
				} else {
					selectScopes = append(selectScopes, &sqlSelectScope{
						depth:      depth,
						clause:     "SELECT",
						orderDepth: -1,
						derived:    depth == 0 || previousWord == "FROM" || previousWord == "JOIN",
					})
				}
			}
			if len(selectScopes) > 0 && depth == selectScopes[len(selectScopes)-1].depth {
				scope := selectScopes[len(selectScopes)-1]
				switch word {
				case "GROUP", "HAVING", "UNION", "DISTINCT", "OVER",
					"COUNT", "SUM", "AVG", "MIN", "MAX", "GROUP_CONCAT":
					scope.unsafe = true
				}
				if word == "BY" && previousWord == "ORDER" {
					scope.orderDepth = depth
				} else if scope.orderDepth >= 0 && (word == "LIMIT" || word == "FOR") {
					scope.orderDepth = -1
				}
				switch word {
				case "FROM", "WHERE", "HAVING", "LIMIT", "OFFSET":
					scope.clause = word
				case "ON":
					scope.clause = "WHERE"
				case "GROUP", "ORDER", "UNION":
					scope.clause = word
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
			if len(selectScopes) > 1 && selectScopes[len(selectScopes)-1].depth == depth {
				selectScopes = selectScopes[:len(selectScopes)-1]
			}
			depth--
			out.WriteByte(query[i])
			i++
			previousToken = ")"
			continue
		}
		clause := ""
		orderDepth := -1
		var scope *sqlSelectScope
		if len(selectScopes) > 0 {
			scope = selectScopes[len(selectScopes)-1]
			clause = scope.clause
			orderDepth = scope.orderDepth
		}
		if query[i] == '-' && i+1 < len(query) && query[i+1] >= '0' && query[i+1] <= '9' && unarySQLSign(previousToken) && parameterizableLiteralAtScope(clause, scope) {
			end := scanSQLNumber(query, i+1)
			if !isSQLIdentifierPartAt(query, end) {
				out.WriteByte('?')
				value := Simplify(query[i:end])
				bindings = append(bindings, value)
				candidates = append(candidates, sqlLiteralCandidate{i, end, value, selectScopes[len(selectScopes)-1], false})
				i = end
				previousToken = "literal"
				continue
			}
		}
		if isSQLNumberStart(query, i) {
			end := scanSQLNumber(query, i)
			if end > i && !isSQLIdentifierPartAt(query, end) {
				isOrderOrdinal := orderDepth == depth && (previousWord == "BY" || previousToken == ",")
				if typeDepth >= 0 || isOrderOrdinal || !parameterizableLiteralAtScope(clause, scope) {
					out.WriteString(query[i:end])
				} else {
					out.WriteByte('?')
					value := Simplify(query[i:end])
					bindings = append(bindings, value)
					candidates = append(candidates, sqlLiteralCandidate{i, end, value, selectScopes[len(selectScopes)-1], false})
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

	if len(candidates) == 0 {
		return query, nil, fnvHashString(query)
	}
	if structuralQuery || len(selectScopes) == 0 || selectScopes[0].unsafe {
		return query, nil, fnvHashString(query)
	}
	accepted := make([]sqlLiteralCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.force || !candidate.scope.unsafe {
			accepted = append(accepted, candidate)
		}
	}
	if len(accepted) == 0 {
		return query, nil, fnvHashString(query)
	}
	if len(accepted) != len(candidates) {
		out = newSQLShapeBuilder(len(query))
		bindings = make([]Scmer, 0, len(accepted))
		position := 0
		for _, candidate := range accepted {
			out.WriteString(query[position:candidate.start])
			out.WriteByte('?')
			bindings = append(bindings, candidate.value)
			position = candidate.end
		}
		out.WriteString(query[position:])
	}
	normalized, shapeHash := out.Result()
	return normalized, bindings, shapeHash
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

func parameterizableLiteralAtScope(clause string, scope *sqlSelectScope) bool {
	if scope == nil || (scope.depth > 0 && !scope.derived) {
		return false
	}
	if clause == "LIMIT" || clause == "OFFSET" {
		return scope.depth == 0
	}
	return clause == "WHERE" || clause == "HAVING"
}
