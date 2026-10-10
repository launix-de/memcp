/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Reuse the SQL parser and native metadata/DDL operators. Dialect types are
frontend names; no runtime tags, catalog tables or secondary allocators. */
(define tsql_identifier_unquoted (parser (not sql_identifier_unquoted
	(atom "TOP" true) (atom "OFFSET" true) (atom "FETCH" true)
	(atom "ROWS" true) (atom "ROW" true) (atom "ONLY" true))))
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
	(parser (atom "NULL" true) (sql_null_literal)) sql_hex_literal sql_number tsql_string)))
(define tsql_column_type (parser (define name tsql_identifier)
	(match (toUpper name) "NVARCHAR" "VARCHAR" "NCHAR" "CHAR" "BIT" "BOOLEAN" _ name)))
(define tsql_column_attributes (parser (define attrs (* (or
	(parser '((atom "PRIMARY" true) (atom "KEY" true)) '("primary" true "null" false))
	(parser (atom "UNIQUE" true) '("unique" true))
	(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
	(parser (atom "NULL" true) '("null" true))
	(parser '((atom "IDENTITY" true) (? "(" (define seed sql_int) "," (define increment sql_int) ")"))
		(if (and (or (nil? seed) (equal? seed 1)) (or (nil? increment) (equal? increment 1)))
			'("auto_increment" true) (error "IDENTITY currently requires seed and increment 1")))
	(parser '((atom "DEFAULT" true) (or
		'((atom "GETDATE" true) "(" ")")
		'("(" (atom "GETDATE" true) "(" ")" ")"))) '("default_expression" "CURRENT_TIMESTAMP"))
	(parser '((atom "DEFAULT" true) (define value (or
		(parser '("(" (define constant tsql_literal) ")") constant) tsql_literal))) '("default" value))
))) (merge attrs)))
(define tsql_select_prefix (parser (? (atom "TOP" true)
	(define top_count (or (parser '("(" (define n sql_int) ")") n) sql_int)))
	(if (and top_count (< top_count 0)) (error "TOP requires a nonnegative count") top_count)))
(define tsql_select_suffix (parser (? (atom "OFFSET" true) (define offset sql_int)
	(or (atom "ROW" true) (atom "ROWS" true))
	(? (atom "FETCH" true) (or (atom "FIRST" true) (atom "NEXT" true))
		(define limit sql_int) (or (atom "ROW" true) (atom "ROWS" true)) (atom "ONLY" true)))
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
(define tsql_dialect (lambda () (begin
	(define builtins (newsession))
	(map '("ABS" "CEILING" "FLOOR" "ROUND" "SQRT" "UPPER" "LOWER" "REPLACE" "SUBSTRING")
		(lambda (name) (builtins name (sql_builtins name))))
	(builtins "ISNULL" coalesceNil)
	(builtins "GETDATE" now)
	(builtins "LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value)))))
	(list "identifier_unquoted" tsql_identifier_unquoted "identifier_quoted" tsql_identifier_quoted
		"identifier" tsql_identifier "string" tsql_string "column" tsql_column "literal" tsql_literal
		"column_type" tsql_column_type "column_attributes" tsql_column_attributes
		"schema_name" (lambda (database owner) (if (equal?? owner "dbo") database owner))
		"select_prefix" tsql_select_prefix "select_suffix" tsql_select_suffix
		"fold_additive" tsql_fold_additive "builtins" builtins "result_columns" tsql_result_columns))))
(define parse_tsql (lambda (schema text policy session tx)
	(parse_sql_dialect schema text policy session tx (tsql_dialect))))
