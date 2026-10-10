/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Independent T-SQL grammar. Statements emit the canonical query AST and
existing native DDL/DML operations, as the other SQL frontends do. */
(define tsql_int (parser (define value (regex "-?[0-9]+")) (simplify value)))
(define tsql_number (parser (define value (regex "-?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:e-?[0-9]+)?" true)) (simplify value)))
(define tsql_identifier_unquoted (parser (not
	(regex "[a-zA-Z_][a-zA-Z0-9_]*")
	(atom "NOT" true)
	(atom "IN" true)
	(atom "BETWEEN" true)
	(atom "AS" true)
	(atom "ON" true)
	(atom "WHERE" true)
	(atom "GROUP" true)
	(atom "BY" true)
	(atom "VALUES" true)
	(atom "FROM" true)
	(atom "LEFT" true)
	(atom "RIGHT" true)
	(atom "INNER" true)
	(atom "OUTER" true)
	(atom "CROSS" true)
	(atom "JOIN" true)
	(atom "SELECT" true)
	(atom "UNION" true)
	(atom "ALL" true)
	(atom "INSERT" true)
	(atom "SET" true)
	(atom "ORDER" true)
	(atom "LIMIT" true)
	(atom "TRIM" true)
	(atom "LTRIM" true)
	(atom "RTRIM" true)
	(atom "TOP" true)
	(atom "OFFSET" true)
	(atom "FETCH" true)
	(atom "ROWS" true)
	(atom "ROW" true)
	(atom "ONLY" true)
	(atom "CASE" true)
	(atom "WHEN" true)
	(atom "THEN" true)
	(atom "ELSE" true)
	(atom "END" true)
	(atom "HAVING" true)
)))
(define tsql_identifier_quoted (parser (or
	(parser '("[" (define id (regex "(?:[^\\]]|\\]\\])+" false false)) "]") (replace id "]]" "]"))
	(parser '("\"" (define id (regex "(?:[^\"]|\"\")+" false false)) "\"") (replace id "\"\"" "\"")))))
(define tsql_identifier (parser (or tsql_identifier_quoted tsql_identifier_unquoted)))
(define tsql_string (parser '((? (atom "N" true false)) (atom "'" false)
	(define value (regex "(?:''|[^'])*" false false)) (atom "'" false false))
	(replace value "''" "'")))
(define tsql_column (parser (or
	(parser '((define alias tsql_identifier) "." (define name tsql_identifier)) '('get_column alias true name true))
	(parser (define name tsql_identifier) '('get_column nil true name true)))))
(define tsql_literal (parser (or
	(parser (atom "NULL" true) (sql_null_literal)) sql_hex_literal tsql_number tsql_string)))
(define tsql_column_type (parser (define name tsql_identifier)
	(match (toUpper name) "NVARCHAR" "VARCHAR" "NCHAR" "CHAR" "BIT" "BOOLEAN" _ name)))
(define tsql_column_attributes (parser (define attrs (* (or
	(parser '((atom "PRIMARY" true) (atom "KEY" true)) '("primary" true "null" false))
	(parser (atom "UNIQUE" true) '("unique" true))
	(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
	(parser (atom "NULL" true) '("null" true))
	(parser '((atom "IDENTITY" true) (? "(" (define seed tsql_int) "," (define increment tsql_int) ")"))
		(if (and (or (nil? seed) (equal? seed 1)) (or (nil? increment) (equal? increment 1)))
			'("auto_increment" true) (error "IDENTITY currently requires seed and increment 1")))
	(parser '((atom "DEFAULT" true) (or
		'((atom "GETDATE" true) "(" ")")
		'("(" (atom "GETDATE" true) "(" ")" ")"))) '("default_expression" "CURRENT_TIMESTAMP"))
	(parser '((atom "DEFAULT" true) (define value (or
		(parser '("(" (define constant tsql_literal) ")") constant) tsql_literal))) '("default" value))
))) (merge attrs)))
(define tsql_select_prefix (parser (? (atom "TOP" true)
	(define top_count (or (parser '("(" (define n tsql_int) ")") n) sql_int)))
	(if (and top_count (< top_count 0)) (error "TOP requires a nonnegative count") top_count)))
(define tsql_select_suffix (parser (? (atom "OFFSET" true) (define offset tsql_int)
	(or (atom "ROW" true) (atom "ROWS" true))
	(? (atom "FETCH" true) (or (atom "FIRST" true) (atom "NEXT" true))
		(define limit tsql_int) (or (atom "ROW" true) (atom "ROWS" true)) (atom "ONLY" true)))
	(if (nil? offset) nil (if (or (< offset 0) (and limit (< limit 0)))
		(error "OFFSET and FETCH require nonnegative counts") (list limit offset)))))
(define tsql_add (lambda (left right)
	(if (or (nil? left) (nil? right)) nil
		(if (and (string? left) (string? right)) (concat left right) (+ left right)))))
(define tsql_fold_additive (lambda (acc term) (match term
	'("add" value) (list tsql_add acc value)
	_ (sql_fold_additive_term acc term))))
/* Explicit wire descriptors come from the existing SQL expression metadata,
never from the first row. Unknown expression types stay text in this first slice. */
(define tsql_result_columns (lambda (query) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(extract_assoc (expand_query_block_fields sources fields) (lambda (name expression) (begin
		(define type (sql_info_type (sql_expr_info sources expression)))
		(define kind (if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) 38
			(if (has? '("FLOAT" "DOUBLE" "REAL" "DECIMAL" "NUMERIC") type) 109
				(if (has? '("BOOLEAN" "BOOL" "BIT") type) 104 231))))
		(list "name" name "kind" kind "size" (if (equal? kind 231) 8000 (if (equal? kind 104) 1 8)) "flags" 1))))
	((symbol union-block) _mode branches _order _limit _offset _facts) (tsql_result_columns (car branches))
	_ (error "unsupported T-SQL result metadata shape"))))

(define tsql_builtins (lambda (name) (match name
	"ISNULL" coalesceNil
	"GETDATE" now
	"LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value))))
	_ (if (has? '("ABS" "CEILING" "FLOOR" "ROUND" "SQRT" "UPPER" "LOWER" "REPLACE" "SUBSTRING" "CONCAT") name)
		(sql_builtins name) nil))))

(define parse_tsql (lambda (database s policy planning_session tx) (begin
	(define schema database)
	(define parse_started_ns (nanotime))

	(define placeholder_counter (newsession))
	(placeholder_counter "n" 0)

	(define extract_title_or_sql (lambda (captured_result)
		(match (car (cdr captured_result))
			'((symbol get_column) nil _ col _) col
			'((symbol get_column) tblvar _ col _) col
			_ (begin
				(define raw_sql (car captured_result))

				(if (> (strlen raw_sql) 256)
					(concat (substr raw_sql 0 200) "..." (fnv_hash raw_sql))
					raw_sql))
		)
	))

	(define tsql_expression (parser (or
		(parser '((define a tsql_expression1) (atom "OR" true) (define b (+ tsql_expression1 (atom "OR" true)))) (cons (quote or) (cons a b)))
		tsql_expression1
	)))

	(define tsql_expression1 (parser (or
		(parser '((define a tsql_expression2) (atom "AND" true) (define b (+ tsql_expression2 (atom "AND" true)))) (cons (quote and) (cons a b)))
		tsql_expression2
	)))

	(define tsql_in_values (parser (or
		(parser '("(" "?*" ")") (sql_in_array_parameter placeholder_counter))
		(parser '("(" (define values (+ tsql_expression ",")) ")") (cons list values)))))

	(define tsql_expression2 (parser (or

		(parser '((atom "NOT" true) (atom "EXISTS" true) "(" (define sub tsql_select) ")") (list (quote not) (list (quote inner_select_exists) sub)))
		(parser '((atom "NOT" true) (define expr tsql_expression2)) '('sql_not expr))

		(parser '((define a tsql_expression3) (atom "IN" true) "(" (define sub tsql_select) ")") '('inner_select_in a sub))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define sub tsql_select) ")") (list (quote not) (list (quote inner_select_in) a sub)))

		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "=" (define b tsql_expression2)) '('equal_collate a b collation))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "<>" (define b tsql_expression2)) '('notequal_collate a b collation))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "!=" (define b tsql_expression2)) '('notequal_collate a b collation))
		(parser '((define a tsql_expression3) (define op (or
			(parser "=" "eq")
			(parser "<>" "ne")
			(parser "!=" "ne")
			(parser "<=" "le")
			(parser ">=" "ge")
			(parser "<" "lt")
			(parser ">" "gt")
		)) (define b tsql_expression2)) (sql_comparison_expr op a b))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) (atom "LIKE" true) (define b tsql_expression2)) '('strlike a b collation))

		(parser '((define a tsql_expression3) (atom "LIKE" true) (define b tsql_expression2)) '('strlike a b "utf8mb4_general_ci"))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "LIKE" true) (define b tsql_expression2)) '('sql_not '('strlike a b "utf8mb4_general_ci")))

		(parser '((define a tsql_expression3) (atom "IN" true) (define b tsql_in_values)) '('sql_in b a))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) (define b tsql_in_values)) (list (quote sql_not) (list (quote sql_in) b a)))

		(parser '((define a tsql_expression3) (atom "BETWEEN" true) (define low tsql_expression3) (atom "AND" true) (define high tsql_expression3)) (list (quote and) (list (quote >=) a low) (list (quote <=) a high)))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "BETWEEN" true) (define low tsql_expression3) (atom "AND" true) (define high tsql_expression3)) (list (quote sql_not) (list (quote and) (list (quote >=) a low) (list (quote <=) a high))))
		tsql_expression3
	)))

	(define tsql_expression4 (parser '(
		(define a tsql_expression5)
		(define terms (* (parser '(
			(define op (or
				(parser "*" "multiply")
				(parser "/" "divide")

				(parser "%" "modulo")
			))
			(define value tsql_expression5)
		) (list op value)) empty true))
	) (reduce terms sql_fold_multiplicative_term a)))

	(define tsql_expression5 (parser (or

		(parser '("-" (define expr tsql_expression6)) '((quote -) 0 expr))

		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		tsql_expression6
	)))

	(define tsql_window_orderby_item (parser '(
		(define col tsql_expression)
		(define dir (or
			(parser (atom "DESC" true) >)
			(parser (atom "ASC" true) <)
			(parser empty <)
		))
	) (list col dir)))

	(define tsql_window_spec (parser '(
		(? (atom "PARTITION" true) (atom "BY" true) (define partition_by (+ tsql_expression ",")))
		(? (atom "ORDER" true) (atom "BY" true) (define order_by (+ tsql_window_orderby_item ",")))
	) (list (coalesce partition_by '()) (coalesce order_by '()))))

	(define tsql_expression6 (parser (or

		(parser '("(" (define sub tsql_select) ")") '('inner_select sub))
		(parser '("(" (define a tsql_expression) ")") a)

		(parser '((atom "EXISTS" true) "(" (define sub tsql_select) ")") '('inner_select_exists sub))

		(parser '((atom "CASE" true) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "CASE" true) (define expr tsql_expression) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (merge '((quote if)) (merge (extract_assoc (merge conditions) (lambda (a b) '('('equal?? expr a) b)))) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" "*" ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "COUNT" '() _over))
		(parser '((atom "COUNT" true) "(" (define e tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "COUNT" (list e) _over))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "SUM" (list s) _over))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "AVG" (list s) _over))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "MIN" (list s) _over))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func "MAX" (list s) _over))

		(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define e tsql_expression) ")") '('count_distinct e))
		(parser '((atom "COUNT" true) "(" "*" ")") '((quote aggregate) 1 (quote +) 0))
		(parser '((atom "COUNT" true) "(" (define e tsql_expression) ")") '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "SUM")) '('aggregate s (car d) (cadr d))))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")")
			(sql_avg_expr s (sql_aggregates "SUM") (sql_aggregates "COUNT")))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MIN")) '('aggregate s (car d) (cadr d))))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MAX")) '('aggregate s (car d) (cadr d))))

		(parser '((define fn tsql_identifier_unquoted) "(" (define arg tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")")
			'('window_func (toUpper fn) (list arg) _over))

		(parser '((atom "SUBSTRING" true) "(" (define s tsql_expression) "," (define start tsql_expression) "," (define len tsql_expression) ")") '((quote sql_substr) s start len))

		(parser '((atom "CONCAT" true) "(" (define p (+ tsql_expression ",")) ")") (cons 'sql_concat p))

		(parser '((atom "TRIM" true) "(" (define e tsql_expression) ")") '((quote sql_trim) e))
		(parser '((atom "LTRIM" true) "(" (define e tsql_expression) ")") '((quote sql_ltrim) e))
		(parser '((atom "RTRIM" true) "(" (define e tsql_expression) ")") '((quote sql_rtrim) e))

		(parser '((atom "COALESCE" true) "(" (define args (* tsql_expression ",")) ")") (cons (quote coalesceNil) args))

		(parser '((atom "NULLIF" true) "(" (define a tsql_expression) "," (define b tsql_expression) ")") '((quote if) '((quote equal??) a b) nil a))

		(parser (atom "NULL" true) (sql_null_literal))

		(parser '((atom "@" true) (define var tsql_identifier_unquoted)) '('session (toLower var)))

		(parser '((atom "LEFT" true) "(" (define s tsql_expression) "," (define n tsql_expression) ")") '((quote sql_substr) s 1 n))

		(parser '((atom "RIGHT" true) "(" (define s tsql_expression) "," (define n tsql_expression) ")") '((quote if) '((quote nil?) s) nil '((quote sql_substr) s '((quote +) 1 '((quote -) '((quote strlen) s) n)) n)))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") '('window_func (toUpper fn) args _over))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")")
			(begin (define d (sql_aggregates (toUpper fn)))
				(if (not (nil? d))
					'('aggregate (car args) (car d) (cadr d))
					(cons (coalesce (tsql_builtins (toUpper fn)) (error "unknown function " fn)) args))))

		(parser "?" (begin
			(define n (placeholder_counter "n"))
			(placeholder_counter "n" (+ n 1))
			(list (quote session) (concat "v" (string (+ n 1))))))
		sql_hex_literal
		tsql_number
		tsql_string
		tsql_column
	)))

	(define tsql_rename_derived_fields (lambda (fields aliases) (match aliases
		(cons alias remaining_aliases) (match fields
			(cons _title (cons expr remaining_fields))
			(cons alias (cons expr (tsql_rename_derived_fields remaining_fields remaining_aliases)))
			_ (error "derived table has more column aliases than columns"))
		_ fields
	)))

	(define tsql_apply_derived_column_aliases (lambda (query aliases) (match query
		((symbol query-block) schema2 tables fields condition group having order limit offset hidden stages facts)
		(list (quote query-block) schema2 tables
			(tsql_rename_derived_fields fields aliases)
			condition group having order limit offset hidden stages facts)
		((symbol union-block) mode branches order limit offset facts)
		(list (quote union-block) mode
			(map branches (lambda (branch) (tsql_apply_derived_column_aliases branch aliases)))
			order limit offset facts)
		_ (error "derived table column aliases require a SELECT query")
	)))

	(define tabledefs (parser (or

		(parser '((define l tabledefs) (define x (or
			(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e tsql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl true e))))
			(parser '((? (atom "INNER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e tsql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl false e))))
			(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tabledefs)) r)
		))) (merge l x))

		(parser '((define l tabledef) (atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledefs) (atom "ON" true) (define e tsql_expression))
			(match l '(id schema tbl _ nil)
				(merge r '('(id schema tbl true e)))))
		(parser (define t tabledef) '(t))
	)))

	(define tabledef (parser (or

		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier)) '(id schema query false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier)) '(id schema query false nil))

		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
		(parser '((define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl tsql_identifier) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
	)))

	(define from nil)

	(define condition nil)

	(define group nil)

	(define having nil)

	(define order nil)

	(define limit nil)

	(define offset nil)

	(define tsql_select_order (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) order
		_ nil
	)))

	(define tsql_select_limit (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) limit
		_ nil
	)))

	(define tsql_select_offset (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) offset
		_ nil
	)))

	(define tsql_select_clear_stage (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts)
		(list (quote query-block) schema tables fields condition group having nil nil nil '() '() '())
		_ query
	)))

	(define tsql_union_all_parts (lambda (query)
		(match query
			((symbol union-block) (symbol all) branches order limit offset facts) (list branches order limit offset)
			_ nil)))

	(define tsql_union_distinct_parts (lambda (query)
		(match query
			((symbol union-block) (symbol distinct) branches order limit offset facts) (list branches order limit offset)
			_ nil)))

	(define tsql_union_all_query (lambda (left right) (begin
		(define right_parts (tsql_union_all_parts right))
		(if (nil? right_parts)
			(list (quote union-block)
				(quote all)
				(list left (tsql_select_clear_stage right))
				(tsql_select_order right)
				(tsql_select_limit right)
				(tsql_select_offset right)
				'())
			(match right_parts '(branches order limit offset)
				(list (quote union-block) (quote all) (cons left branches) order limit offset '())))
	)))

	(define tsql_union_distinct_query (lambda (left right) (begin
		(define right_parts (tsql_union_distinct_parts right))
		(if (nil? right_parts)
			(list (quote union-block)
				(quote distinct)
				(list left (tsql_select_clear_stage right))
				(tsql_select_order right)
				(tsql_select_limit right)
				(tsql_select_offset right)
				'())
			(match right_parts '(branches order limit offset)
				(list (quote union-block) (quote distinct) (cons left branches) order limit offset '())))
	)))

	(define tsql_inner_select_kind (lambda (sym) (begin
		(if (equal?? sym "inner_select")
			(quote inner_select)
			(if (equal?? sym "inner_select_in")
				(quote inner_select_in)
				(if (equal?? sym "inner_select_exists")
					(quote inner_select_exists)
					(match sym
						(symbol inner_select) (quote inner_select)
						'inner_select (quote inner_select)
						'(quote inner_select) (quote inner_select)
						(symbol inner_select_in) (quote inner_select_in)
						'inner_select_in (quote inner_select_in)
						'(quote inner_select_in) (quote inner_select_in)
						(symbol inner_select_exists) (quote inner_select_exists)
						'inner_select_exists (quote inner_select_exists)
						'(quote inner_select_exists) (quote inner_select_exists)
						_ nil)
				)
			)
		)
	)))

	(define tsql_expr_contains_inner_select (lambda (expr) (match expr
		(cons sym args) (or
			(not (nil? (tsql_inner_select_kind sym)))
			(reduce args (lambda (a b) (or a (tsql_expr_contains_inner_select b))) false))
		_ false
	)))

	(define tsql_dataset_contains_inner_select (lambda (dataset)
		(reduce dataset (lambda (a b) (or a (tsql_expr_contains_inner_select b))) false)
	))

	(define tsql_values_row_query (lambda (database columns row)
		(make_query_block database '()
			(merge (map (produceN (count columns)) (lambda (i) (list (nth columns i) (nth row i)))))
			true nil nil nil nil nil '() '() '())))

	(define tsql_values_to_select_query (lambda (schema2 coldesc datasets)
		(match datasets
			(cons only '()) (tsql_values_row_query schema2 coldesc only)
			(cons first rest) (reduce rest (lambda (acc row)
				(tsql_union_all_query acc (tsql_values_row_query schema2 coldesc row)))
				(tsql_values_row_query schema2 coldesc first))
			_ (error "INSERT VALUES requires at least one row")
		)
	))

	(define tsql_select_core (parser '(
		(atom "SELECT" true)

		(define distinct (? (atom "DISTINCT" true)))
		(define prefix_limit tsql_select_prefix)
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil false "*" false)))
			(parser '((define tbl tsql_identifier_quoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define tbl tsql_identifier_unquoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_string)) '(title e))

			(parser '((define e tsql_expression) (define title tsql_identifier)) '(title e))

			(parser (define captured (capture tsql_expression)) '((extract_title_or_sql captured) (car (cdr captured))))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
		)
		(define condition (or (parser '(
			(atom "WHERE" true)
			(define condition2 tsql_expression)
		) condition2) (empty true)))

		(?
			(atom "GROUP" true)
			(atom "BY" true)
			(define group (+
				tsql_expression
				(atom "," true)
			))
		)
		(?
			(atom "HAVING" true)
			(define having tsql_expression)
		)

		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col tsql_expression)
					(? (atom "COLLATE" true) (define coll tsql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))

				(atom "," true)
			))
		)

		(define suffix_stage tsql_select_suffix)

	) (begin
			(if (and suffix_stage (nil? order)) (error "OFFSET requires ORDER BY") true)
			(if (and suffix_stage (not (nil? prefix_limit))) (error "TOP cannot be combined with OFFSET") true)
			(define projected_exprs (extract_assoc (merge cols) (lambda (_title expr) expr)))
			(define sources (if (nil? from) '() (merge from)))

			(define distinct_preserved_by_group (and distinct
				(and (not (nil? group))
					(and (not (empty_list? group))
						(reduce group (lambda (preserved expr)
							(and preserved (contains? projected_exprs expr))) true)))))
			(list (quote query-block) schema sources (merge cols) condition
				(if (and distinct (not distinct_preserved_by_group)) projected_exprs group)
				having order (if (nil? prefix_limit) (if suffix_stage (car suffix_stage) limit) prefix_limit)
				(if suffix_stage (cadr suffix_stage) offset) '() '()
				(merge (list

					(if distinct (list (list (quote select_distinct) true)) '())))))))

	(define tsql_select (parser (or
		(parser '(
			(define left tsql_select_core)
			(atom "UNION" true)
			(atom "ALL" true)
			(define right tsql_select)
		) (tsql_union_all_query left right))
		(parser '(
			(define left tsql_select_core)
			(atom "UNION" true)
			(define right tsql_select)
		) (tsql_union_distinct_query left right))
		tsql_select_core
	)))

	(define tsql_update (parser '(
		(atom "UPDATE" true)
		(define tbldefs (+ tabledefs ","))
		(atom "SET" true)
		(define cols (+ (or
			(parser '(tsql_identifier "." (define title tsql_identifier) "=" (define e tsql_expression)) '(title e))
			(parser '((define title tsql_identifier) "=" (define e tsql_expression)) '(title e))
		) ","))
		(? '(
			(atom "WHERE" true)
			(define condition tsql_expression)
		))

	) (begin
			(define all_defs (merge tbldefs))
			(define first_def (car all_defs))
			(define tbl (match first_def '(_ _ t _ _) t))
			(define tblalias (match first_def '(id _ _ _ _) id))

			(if policy (policy schema tbl true) true)

			(build_dml_plan schema tbl tblalias all_defs (merge cols) (coalesceNil condition true) order limit offset planning_session tx)
	)))

	(define tsql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)

		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition tsql_expression)
		))

	) (begin

			(if policy (policy (coalesce schema2 schema) tbl true) true)

			(define del_schema (coalesce schema2 schema))
			(define del_defs (list (list tbl del_schema tbl false nil)))
			(build_dml_plan del_schema tbl nil del_defs nil (coalesceNil condition true) order limit offset planning_session tx)
	)))

	(define tsql_truncate (parser '(
		(atom "TRUNCATE" true) (? (atom "TABLE" true))

		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
	) (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define trunc_schema (coalesce schema2 schema))
			(cons '!begin (list
				(list (quote checktablemaintenance) trunc_schema tbl "truncate")
				(build_dml_plan trunc_schema tbl nil (list (list tbl trunc_schema tbl false nil)) nil true nil nil nil planning_session tx)))
	)))

	(define tsql_insert_literal_cell (parser (or
		(parser "?" (quote sql_insert_placeholder))
		tsql_literal)))

	(define tsql_insert_values_row (parser (or
		(parser '("(" (define dataset (* tsql_insert_literal_cell ",")) ")")
			(map dataset (lambda (value)
				(if (and (symbol? value) (equal?? value (quote sql_insert_placeholder)))
					(begin
						(define n (placeholder_counter "n"))
						(placeholder_counter "n" (+ n 1))
						(list (quote session) (concat "v" (string (+ n 1)))))
					value))))
		(parser '("(" (define dataset (* tsql_expression ",")) ")") dataset))))

	(define tsql_create_view (parser '(
		(atom "CREATE" true)
		(define replace (? (atom "OR" true) (atom "ALTER" true)))
		(atom "VIEW" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(define target (or
			(parser '((define schema2 tsql_schema_identifier) "." (define id tsql_identifier)) '(schema2 id))
			(parser (define id tsql_identifier) '(nil id))))
		(define aliases (? (parser '("(" (define columns (+ tsql_identifier ",")) ")") columns)))
		(atom "AS" true)
		(define captured (capture tsql_select))
	) (match target '(schema2 id) (begin
			(define view_schema (coalesce schema2 schema))
			(if policy (policy view_schema id true) true)
			(define query (sql_apply_view_column_aliases (nth captured 1) aliases))
			(list (quote create_sql_view)
				(list (quote session) "__memcp_tx")
				view_schema id "tsql" (nth captured 0) (list (quote quote) query)
				(if replace "replace" (if ifnotexists "ignore" "error")))))))

	(define tsql_schema_identifier (parser (define owner tsql_identifier)
		(if (equal?? owner "dbo") database owner)))
	(define tsql_expression3 (parser '(
		(define a tsql_expression4)
		(define terms (* (parser '(
			(define op (or (parser "+" "add") (parser "-" "sub")))
			(define value tsql_expression4)
		) (list op value)) empty true))
	) (reduce terms tsql_fold_additive a)))
	(define tsql_build_select_plan (lambda (query) (begin
		(define expanded (sql_expand_views query policy))
		(list (quote !begin)
			(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded))
				(list (quote quote) (tsql_result_columns expanded)))
			(build_queryplan_term expanded planning_session tx)))))
	/* INSERT SELECT consumes the existing planner's result stream and the native
	insert operator. Identity allocation and constraints remain native. */
	(define tsql_insert_select_plan (lambda (database tbl columns query) (begin
		(define insert_expr '('insert '('table database tbl) (cons list columns)
			(cons list '((cons list (map (produceN (count columns))
				(lambda (i) '('nth 'item (+ (* i 2) 1)))))))
			'(list) nil false '('lambda '('id) '('session "last_insert_id" 'id)) (quote tx)))
		(define plan (build_queryplan_term (sql_expand_views query policy) planning_session tx))
		(define ordered (or
			(and (query_block? query) (not (empty_list? (coalesceNil (qb_order query) '()))))
			(and (union_block? query) (not (empty_list? (coalesceNil (union_order query) '()))))))
		(if ordered
			(list (quote begin)
				(list (quote set) (quote __insert_count) (list (quote newsession)))
				(list (quote __insert_count) "count" 0)
				(list (quote set) (quote resultrow) (list (quote lambda) (list (quote item))
					(list (quote begin)
						(list (quote set) (quote __inserted) insert_expr)
						(list (quote __insert_count) "count" (list (quote +) (list (quote __insert_count) "count") (quote __inserted)))
						(quote __inserted))))
				plan (list (quote __insert_count) "count"))
			(list (quote begin)
				(list (quote set) (quote resultrow) (list (quote lambda) (list (quote item)) insert_expr)) plan)))))
	(define tsql_insert_into (parser '(
		(atom "INSERT" true) (atom "INTO" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ tsql_insert_values_row ","))
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			(define columns (coalesce columns (table_insertable_columns database tbl)))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count columns)))) true)
				true (error "INSERT column count does not match value count"))
			(if (reduce rows (lambda (found row) (or found (tsql_dataset_contains_inner_select row))) false)
				(tsql_insert_select_plan database tbl columns (tsql_values_to_select_query database columns rows))
				'('insert '('table database tbl) (cons list columns) (sql_insert_values_expr rows)
					'(list) nil false '('lambda '('id) '('session "last_insert_id" 'id)) (quote tx)))
	)))
	(define tsql_insert_select (parser '(
		(atom "INSERT" true) (atom "INTO" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(define query tsql_select)
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			(tsql_insert_select_plan database tbl (coalesce columns (table_insertable_columns database tbl)) query)
	)))
	(define tsql_foreign_key_mode (parser (or
		(parser '((atom "NO" true) (atom "ACTION" true)) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser '((atom "SET" true) (atom "NULL" true)) "set null"))))
	(define column_dimensions (parser (or
		(parser '("(" (define precision tsql_int) "," (define scale tsql_int) ")") (list (quote list) precision scale))
		(parser '("(" (define length tsql_int) ")") (list (quote list) length))
		(parser empty '(list)))))
	(define column_definition (parser '(
		(define name tsql_identifier) (define type tsql_column_type)
		(define dimensions column_dimensions) (define attributes tsql_column_attributes)
	) (list name type dimensions attributes)))
	(define foreign_key_definition (parser '(
		(? (atom "CONSTRAINT" true) (define name tsql_identifier))
		(atom "FOREIGN" true) (atom "KEY" true) "(" (define columns (+ tsql_identifier ",")) ")"
		(atom "REFERENCES" true) (? tsql_schema_identifier ".") (define target tsql_identifier)
		"(" (define target_columns (+ tsql_identifier ",")) ")"
		(? (atom "ON" true) (atom "DELETE" true) (define deletemode tsql_foreign_key_mode))
		(? (atom "ON" true) (atom "UPDATE" true) (define updatemode tsql_foreign_key_mode))
	) (list (quote list) "foreign" name (cons (quote list) columns) target
			(cons (quote list) target_columns) updatemode deletemode)))
	(define tsql_create_table (parser '(
		(atom "CREATE" true) (atom "TABLE" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier)
		"(" (define columns (+ (or
			(parser '((atom "PRIMARY" true) (atom "KEY" true) "(" (define keys (+ tsql_identifier ",")) ")")
				(list (quote list) "unique" "PRIMARY" (cons (quote list) keys)))
			(parser '((atom "CONSTRAINT" true) (define key tsql_identifier) (atom "UNIQUE" true)
				"(" (define keys (+ tsql_identifier ",")) ")")
				(list (quote list) "unique" key (cons (quote list) keys)))
			foreign_key_definition
			(parser (define column column_definition) (match column '(name type dimensions attributes)
				(list (quote list) "column" name type dimensions (cons (quote list) attributes))))
		) ",")) ")"
	) (begin
			(if policy (policy (coalesce schema2 schema) name true) true)
			(list (quote sql_create_table) (coalesce schema2 schema) name (cons (quote list) columns)
				'(list) (if ifnotexists true false) (quote tx)))))
	(define tsql_alter_table (parser '(
		(atom "ALTER" true) (atom "TABLE" true)
		(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier)
		(define changes (+ (or
			(parser '((atom "ADD" true) (define column column_definition))
				(match column '(col type dimensions attributes)
					(lambda (database name)
						(list (quote sql_create_column) (list (quote table) database name)
							col type dimensions (cons (quote list) (sql_add_column_attributes attributes))))))
			(parser '((atom "DROP" true) (atom "COLUMN" true) (define col tsql_identifier))
				(lambda (database name)
					(list (quote altertable) (list (quote table) database name) "drop" col)))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true) (define key tsql_identifier))
				(lambda (database name) (list (quote or)
					(list (quote dropforeignkey) (list (quote table) database name) key)
					(list (quote dropkey) (list (quote table) database name) key))))
		) ","))
	) (begin
			(if policy (policy (coalesce schema2 schema) name true) true)
			(cons (quote !begin) (map changes (lambda (change) (change (coalesce schema2 schema) name)))))))
	(define p (parser (or
		(parser (define query tsql_select) (tsql_build_select_plan query))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_select)) (explain_queryplan_ir (sql_expand_views query policy)))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query tsql_select)) (explain_queryplan_reorder (sql_expand_views query policy) planning_session))
		(parser '((atom "EXPLAIN" true) (atom "COMPILE" true) (define query tsql_select)) (explain_queryplan_compile (sql_expand_views query policy) parse_started_ns (strlen s) planning_session))
		(parser '((atom "EXPLAIN" true) (atom "PHYSICAL" true) (define query tsql_select)) (explain_queryplan_physical (sql_expand_views query policy) planning_session))
		(parser '((atom "EXPLAIN" true) (define query tsql_select))
			'('resultrow '('list "code" (pretty_print (optimize (build_queryplan_term (sql_expand_views query policy) planning_session tx)) (settings "ExplainWidth")))))
		tsql_insert_into tsql_insert_select tsql_update tsql_delete tsql_truncate
		tsql_create_view tsql_create_table tsql_alter_table
		(parser '((atom "CREATE" true) (atom "DATABASE" true)
			(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
			(define name tsql_identifier)) (begin
				(if policy (policy "system" true true) true)
				(list (quote createdatabase) name (if ifnotexists true false) nil nil)))
		(parser '((atom "DROP" true) (atom "DATABASE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true))) (define name tsql_identifier))
			(begin (if policy (policy "system" true true) true)
				(list (quote drop_sql_database) (list (quote session) "__memcp_tx") name (if ifexists true false))))
		(parser '((atom "DROP" true) (atom "TABLE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(begin (if policy (policy (coalesce schema2 schema) name true) true)
				(list (quote droptable) (coalesce schema2 schema) name (if ifexists true false))))
		(parser '((atom "DROP" true) (atom "VIEW" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(begin (if policy (policy (coalesce schema2 schema) name true) true)
				(list (quote drop_sql_view) (list (quote session) "__memcp_tx")
					(coalesce schema2 schema) name (if ifexists true false))))
		(parser '((atom "USE" true) (define name tsql_identifier)) (begin
			(if policy (policy name true false) true)
			(list (quote session) "schema" name)))
		(parser '((atom "BEGIN" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_begin) (quote session) (quote tx)))
		(parser '((atom "COMMIT" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_commit) (quote session)))
		(parser '((atom "ROLLBACK" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_rollback) (quote session)))
		""
	)))

	((parser (define command p) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))
