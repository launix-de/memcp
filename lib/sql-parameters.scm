/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.
You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

/* This is the lexical front-cache pass, not a second SQL parser. Its policy
belongs with SQL scopes and literal syntax. The ordinary parser still validates
the complete statement. Tokenization uses the runtime's generic regex engine. */
(define sql_parameter_tokens (lambda (query)
	(regexp_matches query
		"[ \\t\\r\\n]+|--[^\\n]*|#[^\\n]*|/\\*(?s:.*?)\\*/|`(?:\\\\.|``|[^`\\\\])*`|'(?:\\\\.|''|[^'\\\\])*'|\"(?:\\\\.|\"\"|[^\"\\\\])*\"|[a-zA-Z_$][a-zA-Z0-9_$]*|[0-9]+(?:\\.[0-9]*)?(?:[eE][+-]?[0-9]+)?|(?s:.)")))

(define sql_parameter_prefix (lambda (query)
	(regexp_test (toUpper (strtrim query))
		"^(?:SELECT[ \\t\\n]|EXPLAIN +(?:IR +)?(?:REORDER +)?SELECT(?:[ \\t\\n]|$)|DELETE[ \\t\\n]+FROM[ \\t\\n]|UPDATE[ \\t\\n]|INSERT[ \\t\\n]+INTO[ \\t\\n])")))

/* WHERE/HAVING gate a filter-affecting literal (needs the safe-scope check every
other literal gets). SET (UPDATE's assignment list) and VALUES (INSERT's row
tuples) are added here too: a literal there is a value being WRITTEN, not a
filter or grouping key -- it can never change plan shape or selectivity, only
the safe-scope check that already applies to every candidate here matters. */
(define sql_parameter_scope_allows (lambda (scope)
	(and (not (nil? scope))
		(or (equal? (scope "depth") 0) (scope "derived"))
		(if (has? '("LIMIT" "OFFSET") (scope "clause"))
			(equal? (scope "depth") 0)
			(has? '("WHERE" "HAVING" "SET" "VALUES") (scope "clause"))))))

(define sql_parameter_unary (lambda (token)
	(has? '("" "(" "," "=" ">" "<" ">=" "<=" "<>" "!=" "+" "-" "*" "/"
		"WHERE" "ON" "LIMIT" "OFFSET" "AND" "OR" "THEN" "ELSE") token)))

(define sql_parameter_new_scope (lambda (depth previous_word statement_word)
	(begin
		(define scope (newsession))
		(scope "depth" depth)
		(scope "clause" statement_word)
		(scope "order_depth" -1)
		(scope "unsafe" false)
		(scope "derived" (or (equal? depth 0) (has? '("FROM" "JOIN") previous_word)))
		scope)))

(define sql_parameter_scope_word (lambda (scope word previous_word depth)
	(if (and (not (nil? scope)) (equal? depth (scope "depth")))
		(begin
			(if (has? '("GROUP" "HAVING" "UNION" "DISTINCT" "OVER"
				"COUNT" "SUM" "AVG" "MIN" "MAX" "GROUP_CONCAT") word)
				(scope "unsafe" true))
			(if (and (equal? word "BY") (equal? previous_word "ORDER"))
				(scope "order_depth" depth)
				(if (has? '("LIMIT" "FOR") word) (scope "order_depth" -1)))
			/* SET (UPDATE ... SET a=1, b=2 ...) and VALUES (INSERT ... VALUES (1,2), (3,4))
			enter the same allow-listed-clause tracking as WHERE/HAVING -- see
			sql_parameter_scope_allows. ON DUPLICATE KEY UPDATE reuses the existing
			ON->WHERE mapping below: its assignment list is exactly as safe to fold as
			a WHERE literal, and giving it a dedicated clause name would add a state
			with no different behavior. */
			(if (has? '("FROM" "WHERE" "HAVING" "LIMIT" "OFFSET" "GROUP" "ORDER" "UNION" "ON" "SET" "VALUES") word)
				(scope "clause" (if (equal? word "ON") "WHERE" word)))))))

(define sql_parameter_next_significant (lambda (tokens i)
	(if (>= i (count tokens)) nil
		(if (regexp_test (nth tokens i) "^(?:[ \\t\\r\\n]|--|#|/\\*)")
			(sql_parameter_next_significant tokens (+ i 1))
			(nth tokens i)))))

/* One item of an all-constant projection row, e.g. each literal in
`SELECT 0 AS i, 1704067200 AS s, 1789037871 AS e` (the shape a generated query
emits for a UNION of synthetic period rows). Such a literal is the whole
select-list expression: it only appears verbatim in the result and never
influences plan shape, selectivity or grouping, so it is a forced candidate that
survives an unsafe scope. Recognised lexically: it opens a select item (previous
significant token is SELECT or a comma), sits at the scope's own paren depth and
is directly AS-aliased. The caller gates this on const_row_ok, which additionally
requires the entire select list to be constant-AS items. */
(define sql_parameter_select_const_item (lambda (const_row_ok scope depth previous_token idx tokens)
	(and const_row_ok
		(not (nil? scope))
		(equal? depth (scope "depth"))
		(equal? (scope "clause") "SELECT")
		(has? '("SELECT" ",") previous_token)
		(match (sql_parameter_next_significant tokens (+ idx 1))
			next (equal? (toUpper next) "AS")
			_ false))))

/* Candidates retain their owning scope until all tokens have been visited:
a later GROUP/HAVING may disqualify an earlier literal in that scope. The
sessions and token buffers belong to this one lexical compilation only. */
(define parameterize_sql_select_literals (lambda (query)
	(if (not (sql_parameter_prefix query)) (list query '() (fnv_hash query))
		(begin
			(define tokens (sql_parameter_tokens query))
			/* Only fold constant projection items when the whole SELECT list is a
			constant row (`SELECT <num> AS a, <num> AS b [...] FROM|UNION|)`), the shape a
			generated query produces for a UNION of synthetic period rows. A lone
			`1 AS _flag` mixed with real columns stays exact -- it is a match carrier. */
			(define const_row_ok (regexp_test query "(?is)SELECT\\s+-?(?:0[xX][0-9a-fA-F]+|[0-9]+(?:\\.[0-9]*)?(?:[eE][+-]?[0-9]+)?)\\s+AS\\s+(?:`[^`]+`|[A-Za-z_$][A-Za-z0-9_$]*)\\s*(?:,\\s*-?(?:0[xX][0-9a-fA-F]+|[0-9]+(?:\\.[0-9]*)?(?:[eE][+-]?[0-9]+)?)\\s+AS\\s+(?:`[^`]+`|[A-Za-z_$][A-Za-z0-9_$]*)\\s*)*(?:FROM\\b|UNION\\b|\\)|;|$)"))
			(define candidates (newsession))
			(define pieces (newsession))
			(define result (for (list 0 0 -1 "" "" '() false 0 0)
				(lambda (idx depth type_depth previous_word previous_token scopes invalid candidate_count piece_count)
					(and (not invalid) (< idx (count tokens))))
				(lambda (idx depth type_depth previous_word previous_token scopes invalid candidate_count piece_count)
					(begin
						(define token (nth tokens idx))
						(define scope (if (equal? scopes '()) nil (car scopes)))
						(if (regexp_test token "^(?:[ \\t\\r\\n]|--|#|/\\*)")
							(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word previous_token scopes false candidate_count (+ piece_count 1)))
							(if (regexp_test token "^[a-zA-Z_$]")
								(begin
									(define word (toUpper token))
									/* SELECT re-enters the same scope at the same depth (a UNION branch or
									the SELECT half of INSERT ... SELECT); a nested SELECT at a deeper depth
									gets its own scope, same as before. DELETE/UPDATE/INSERT only ever open
									the outermost scope, once, as literally the first token sql_parameter_prefix
									already required -- scopes is still empty at that point. */
									(define next_scopes (if (equal? word "SELECT")
										(if (and (not (nil? scope)) (equal? (scope "depth") depth))
											(begin (scope "clause" "SELECT") (scope "order_depth" -1) scopes)
											(cons (sql_parameter_new_scope depth previous_word word) scopes))
										(if (and (equal? scopes '()) (equal? depth 0) (has? '("DELETE" "UPDATE" "INSERT") word))
											(cons (sql_parameter_new_scope depth previous_word word) scopes)
											scopes)))
									(sql_parameter_scope_word (if (equal? next_scopes '()) nil (car next_scopes)) word previous_word depth)
									(begin (pieces piece_count token) (list (+ idx 1) depth type_depth word word next_scopes (equal? word "OVER") candidate_count (+ piece_count 1))))
								(if (equal? token "(")
									(begin (pieces piece_count token) (list (+ idx 1) (+ depth 1) (if (has? '("DECIMAL" "VARCHAR") previous_word) (+ depth 1) type_depth) previous_word token scopes false candidate_count (+ piece_count 1)))
									(if (equal? token ")")
										(begin (pieces piece_count token) (list (+ idx 1) (- depth 1) (if (equal? depth type_depth) -1 type_depth) previous_word token (if (and (> (count scopes) 1) (equal? (scope "depth") depth)) (cdr scopes) scopes) false candidate_count (+ piece_count 1)))
										(if (regexp_test token "^['\"]")
											(if (equal? (strlen token) 1)
												(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word token scopes true candidate_count (+ piece_count 1)))
												(begin
													(define forced (and (not (nil? scope)) (> (scope "depth") 0) (has? '("LIKE" "AGAINST") previous_word)))
													(if (and (not (has? '("AS" "DATE") previous_word)) (or forced (sql_parameter_scope_allows scope)))
														(begin (candidates candidate_count (list piece_count scope forced (regexp_replace (substr token 1 (- (strlen token) 2)) "\\\\[\\\\'\"nr0]" sql_string_unescape))) (begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word "literal" scopes false (+ candidate_count 1) (+ piece_count 1))))
														(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word "literal" scopes false candidate_count (+ piece_count 1))))))
											(if (regexp_test token "^`")
												(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word "identifier" scopes (equal? (strlen token) 1) candidate_count (+ piece_count 1)))
												(if (and (equal? token "-") (< (+ idx 1) (count tokens))
													(regexp_test (nth tokens (+ idx 1)) "^[0-9]") (sql_parameter_unary previous_token)
													(sql_parameter_scope_allows scope))
													(begin
														(define number (nth tokens (+ idx 1)))
														(if (and (< (+ idx 2) (count tokens)) (regexp_test (nth tokens (+ idx 2)) "^[a-zA-Z0-9_$]"))
															(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word token scopes false candidate_count (+ piece_count 1))) (begin (candidates candidate_count (list piece_count scope false (simplify (concat "-" number)))) (begin (pieces piece_count (concat "-" number)) (list (+ idx 2) depth type_depth previous_word "literal" scopes false (+ candidate_count 1) (+ piece_count 1))))))
													(if (regexp_test token "^[0-9]")
														(if (or (>= type_depth 0)
															(and (not (sql_parameter_scope_allows scope)) (not (sql_parameter_select_const_item const_row_ok scope depth previous_token idx tokens)))
															(and (not (nil? scope)) (equal? (scope "order_depth") depth) (or (equal? previous_word "BY") (equal? previous_token ",")))
															(and (< (+ idx 1) (count tokens)) (regexp_test (nth tokens (+ idx 1)) "^[a-zA-Z0-9_$]")))
															(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word "literal" scopes false candidate_count (+ piece_count 1))) (begin (candidates candidate_count (list piece_count scope (sql_parameter_select_const_item const_row_ok scope depth previous_token idx tokens) (simplify token))) (begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word "literal" scopes false (+ candidate_count 1) (+ piece_count 1)))))
														(begin (pieces piece_count token) (list (+ idx 1) depth type_depth previous_word token scopes (or (equal? token "?") (and (equal? token "/") (< (+ idx 1) (count tokens)) (equal? (nth tokens (+ idx 1)) "*"))) candidate_count (+ piece_count 1)))))))))))))))
			(match result '(_idx _depth _type _word _token scopes invalid candidate_count piece_count)
				/* Unchanged from the pre-existing master logic: an unsafe OUTER query
				block (an aggregate/UNION/DISTINCT at the top scope's own depth, e.g.
				`SELECT COUNT(*) ... WHERE x IN (SELECT ... UNION ...)`) still bails the
				whole statement to exact, no exceptions -- including constant-projection-
				row candidates. They never need that exception: a const-row derived table
				is always its own nested scope, and reaching an unsafe outer scope at all
				requires an aggregate/UNION token at the outer scope's own depth, which the
				const-row shape (bare `SELECT <num> AS x, ... UNION ...` with no aggregate
				at that depth) does not produce. Only the candidate's OWN scope being
				unsafe is exempted per-candidate below, exactly as before. */
				(if (or invalid (equal? scopes '()) ((car scopes) "unsafe") (equal? candidate_count 0))
					(list query '() (fnv_hash query))
					(begin
						(define accepted (filter (map (produceN candidate_count) (lambda (idx) (candidates idx)))
							(lambda (candidate) (or (nth candidate 2) (not ((cadr candidate) "unsafe"))))))
						(if (equal? accepted '())
							(list query '() (fnv_hash query))
							(begin
								(reduce accepted (lambda (_ candidate) (pieces (car candidate) "?")) nil)
								(define normalized (apply concat (map (produceN piece_count) (lambda (idx) (pieces idx)))))
								(list normalized (map accepted (lambda (candidate) (nth candidate 3))) (fnv_hash normalized)))))))))))

/* Keep exact SQL variants out of the parser while sharing their compiled plan.
Only parameterized results enter the small front cache; exact-only statements
continue to occupy just their existing query-plan entry. The third result item
is the normalized shape hash. Keeping it beside the bindings avoids traversing
the same normalized query again on every warm literal-specialization hit. */
(define sql_parameterized_shape_result (lambda (result)
	(match result
		'(normalized bindings shape_hash) result
		'(normalized bindings) (list normalized bindings (fnv_hash normalized)))))

(define sql_parameterize_select_literals_cached (lambda (cache query enabled)
	(if (not enabled)
		(list query '() (fnv_hash query))
		(begin
			(define cached (cache query))
			(if cached
				cached
				(match (parameterize_sql_select_literals query) '(normalized bindings shape_hash)
					(begin
						(define result (if (equal? bindings '())
							(sql_parameterized_shape_result (sql_parameterize_select_like_strings query enabled))
							(list normalized bindings shape_hash)))
						(if (equal? (cadr result) '())
							result
							(cache query result)))))))))

(define sql_parameterize_select_literals (lambda (query enabled)
	(sql_parameterize_select_literals_cached sql_literal_shape_cache query enabled)))


/* sql_parameterize_select_like_strings: query-plan-cache helper for ad-hoc
fulltext-ish SELECTs. It replaces string literals directly following LIKE or
inside MATCH...AGAINST(...) with ? placeholders and returns (normalized-query bindings). Other string
literals, DDL/DML and already-parameterized statements keep exact cache keys. */
(define sql_parameterize_select_like_strings (lambda (query enabled) (begin
	(define starts_like_select (lambda (q)
		(match q (regex "^\\s*SELECT\\b" _) true false)))
	(define parameterized_rhs_literal? (lambda (q pos) (begin
		(define prefix (toUpper (strrtrim (substr q 0 pos))))
		(or
			(match prefix (regex "(?s:.*)\\bLIKE$" _) true false)
			(match prefix (regex "(?s:.*)\\bAGAINST\\s*\\($" _) true false)))))
	(define read_string_literal (lambda (q start quote_ch) (begin
		(define len (strlen q))
		/* Store each piece under its own session key (O(1) amortized per
		write, like any dict built incrementally in a loop) instead of
		concat-in-a-loop, which would copy the whole prefix on every
		character (O(n^2) for a literal of length n). cons/append aren't an
		O(1) alternative here either: both are implemented as a full copy of
		the existing list on this slice-backed representation. */
		(define pieces (newsession))
		(match (for (list (+ start 1) 0 false)
			(lambda (i count done) (and (not done) (< i len)))
			(lambda (i count done) (begin
				(define ch (substr q i 1))
				(if (equal? ch "\\")
					(if (< (+ i 1) len)
						(begin (pieces count (substr q (+ i 1) 1)) (list (+ i 2) (+ count 1) false))
						(list (+ i 1) count false))
					(if (equal? ch quote_ch)
						(list (+ i 1) count true)
						(begin (pieces count ch) (list (+ i 1) (+ count 1) false)))))))
			'(next_i count done)
			(list next_i (apply concat (map (produceN count) (lambda (idx) (pieces idx)))) done)))))
	(if (or
		(not enabled)
		(not (starts_like_select query))
		(match (toUpper query)
			(regex "(?:\\b(?:COUNT|SUM|AVG|MIN|MAX|GROUP_CONCAT)\\s*\\(|\\bGROUP\\s+BY\\b|\\bHAVING\\b)" _)
			true false)
		(not (or
			(match (toUpper query) (regex "\\bLIKE\\b" _) true false)
			(match (toUpper query) (regex "\\bAGAINST\\b" _) true false)))
		(match query (regex "\\?" _) true false))
		(list query '())
		(begin
			(define len (strlen query))
			/* Same session-backed accumulation as read_string_literal: out
			collects pieces (single chars or a "?" placeholder) by index
			instead of concatenating on every character. */
			(define out (newsession))
			(define state (for (list 0 0 '() false)
				(lambda (i out_count bindings invalid) (and (not invalid) (< i len)))
				(lambda (i out_count bindings invalid) (begin
					(define ch (substr query i 1))
					(if (and (or (equal? ch "'") (equal? ch "\"")) (parameterized_rhs_literal? query i))
						(match (read_string_literal query i ch) '(next_i value done)
							(if done
								(begin (out out_count "?") (list next_i (+ out_count 1) (merge bindings (list value)) false))
								(list len out_count '() true)))
						(begin (out out_count ch) (list (+ i 1) (+ out_count 1) bindings false)))))))
			(match state '(end_i out_count bindings invalid)
				(if (or invalid (equal? bindings '()))
					(list query '())
					(list (apply concat (map (produceN out_count) (lambda (idx) (out idx)))) bindings))))))))

