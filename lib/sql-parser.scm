/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Hänsch

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

(define sql_builtins (coalesce sql_builtins (newsession)))

(define sql_identifier_unquoted (parser (not
	(regex "[a-zA-Z_][a-zA-Z0-9_]*")
	/* exceptions for things that can't be identifiers */
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
	(atom "FROM" true)
	(atom "LEFT" true)
	(atom "RIGHT" true)
	(atom "INNER" true)
	(atom "OUTER" true)
	(atom "CROSS" true)
	(atom "JOIN" true)
	(atom "SELECT" true)
	(atom "INSERT" true)
	(atom "ORDER" true)
	(atom "LIMIT" true)
	(atom "DELIMITER" true)
	(atom "TRIM" true)
	(atom "LTRIM" true)
	(atom "RTRIM" true)
)))
(define sql_identifier_quoted (parser '("`" (define id (regex "(?:[^`]|``)+" false false)) "`") (replace id "``" "`"))) /* with backtick */
(define sql_identifier (parser (define x (or sql_identifier_unquoted sql_identifier_quoted)) x))

(define sql_column (parser (or
	(parser '((define tbl sql_identifier_unquoted) "." (define col sql_identifier_unquoted)) '((quote get_column) tbl true col true))
	(parser '((define tbl sql_identifier_unquoted) "." (define col sql_identifier_quoted)) '((quote get_column) tbl true col false))
	(parser '((define tbl sql_identifier_quoted) "." (define col sql_identifier_unquoted)) '((quote get_column) tbl false col true))
	(parser '((define tbl sql_identifier_quoted) "." (define col sql_identifier_quoted)) '((quote get_column) tbl false col false))
	(parser (define col sql_identifier_quoted) '((quote get_column) nil true col false))
	(parser (define col sql_identifier_unquoted) '((quote get_column) nil true col true))
)))

(define sql_int (parser (define x (regex "-?[0-9]+")) (simplify x)))
(define sql_number (parser (define x (regex "-?[0-9]+\.?[0-9]*(?:e-?[0-9]+)?" true)) (simplify x)))

(define sql_string (parser (or
	(parser '((atom "'" false) (define x (regex "(\\\\.|[^\\'])*" false false)) (atom "'" false false)) (sql_unescape x))
	(parser '((atom "\"" false) (define x (regex "(\\\\.|[^\\\"])*" false false)) (atom "\"" false false)) (sql_unescape x))
)))

/* SQL modulo expression: NULL-safe, division-by-zero-safe, truncates quotient toward zero */
(define sql_mod_expr (lambda (a b)
	'((quote if)
		'((quote or) '((quote nil?) a) '((quote nil?) b) '((quote equal??) b 0))
		nil
		'((quote -) a '((quote *) b '((quote if) '((quote <) '((quote /) a b) 0) '((quote ceil) '((quote /) a b)) '((quote floor) '((quote /) a b)))))
	)
))

/* lightweight literal parser for top-level contexts (before sql_expression is defined) */
(define sql_literal (parser (or
	(parser (atom "NULL" true) 'nil)
	(parser (atom "TRUE" true) true)
	(parser (atom "FALSE" true) false)
	sql_number
	sql_string
)))
(define sql_index_column (parser (or
	(parser '((define col sql_identifier) "(" (define len sql_int) ")") col)
	(parser (define col sql_identifier) col)
)))

(define parse_sql (lambda (schema s policy) (begin


	/* derive the description of a column from its expression */
	/* For single column: use raw column name (most apps rely on this) */
	/* For complex expressions: use captured SQL text */
	/* captured_result is (raw_sql expr) from capture wrapper */
	(define extract_title_or_sql (lambda (captured_result)
		(match (car (cdr captured_result))
			'((symbol get_column) nil _ col _) col
			'((symbol get_column) tblvar _ col _) col /* x.y -> col */
			_ (car captured_result) /* for complex expressions, use captured SQL */
		)
	))

	/* merge two arrays into a dict */
	(define zip_cols (lambda(cols tuple) (match cols
		(cons col cols) (cons col (cons (car tuple) (zip_cols cols (cdr tuple))))
		'()
	)))

	/* helper function for triggers and ON DUPLICATE: every column is just a symbol */
	(define replace_stupid (lambda (expr) (match expr
		'('get_column "VALUES" _ col _) (symbol (concat "NEW." col))
		'('get_column _ _ col _) (symbol col) /* TODO: case matching */
		(cons head tail) (cons head (map tail replace_stupid))
		expr
	)))
	/* helper function for triggers and ON DUPLICATE: extract all used columns */
	(define extract_stupid (lambda (expr) (match expr
		'('get_column "VALUES" _ col _) '((concat "NEW." col))
		'('get_column _ _ col _) '(col)
		(cons head tail) (merge_unique (map tail extract_stupid))
		'()
	)))

	/* helper function for triggers: transform get_column to dict access */
	/* (get_column "NEW" _ col _) -> (get_assoc NEW col) */
	/* (get_column "OLD" _ col _) -> (get_assoc OLD col) */
	/* (get_column nil _ col _) -> (get_assoc NEW col) for unqualified columns in trigger context */
	(define transform_trigger_expr (lambda (expr) (match expr
		'('get_column "NEW" _ col _) (list (symbol "get_assoc") (symbol "NEW") col)
		'('get_column "OLD" _ col _) (list (symbol "get_assoc") (symbol "OLD") col)
		'('get_column nil _ col _) (list (symbol "get_assoc") (symbol "NEW") col)
		(cons head tail) (cons (transform_trigger_expr head) (map tail transform_trigger_expr))
		expr
	)))

	/* helper function to build nested set_assoc calls at compile time */
	/* (build_trigger_sets NEW ((col1 expr1) (col2 expr2))) -> (set_assoc (set_assoc NEW "col1" expr1) "col2" expr2) */
	(define build_trigger_sets (lambda (acc assignments)
		(match assignments
			'() acc
			(cons assignment rest) (match assignment '(col expr)
				(build_trigger_sets (list (symbol "set_assoc") acc col (transform_trigger_expr expr)) rest)))))

	/* compile_trigger_body: compile parsed trigger body to executable lambda */
	/* Input: schema (database name for INSERT/UPDATE/DELETE statements) */
	/*        timing (string like "before_insert", "after_insert", etc.) */
	/*        body from sql_trigger_body */
	/*   - (!begin stmt1 stmt2 ...) for BEGIN...END blocks */
	/*   - (list (col1 expr1) (col2 expr2) ...) for SET statements */
	/* Output: (lambda (OLD NEW) ...) that can be applied by ExecuteTriggers */
	/* Uses set_assoc approach: (set changed_rows (set_assoc changed_rows key value)) */
	(define compile_trigger_body (lambda (schema timing body) (begin
		(define params (list (symbol "OLD") (symbol "NEW")))
		(define is_after (or (equal? timing "after_insert") (equal? timing "after_update") (equal? timing "after_delete")))
		(define changed_rows_sym (symbol "changed_rows"))

		/* Helper to compile SET assignments into (set changed_rows ...) statements */
		/* Returns a list of set statements for each assignment */
		(define compile_set_assignments (lambda (assignments)
			(map assignments (lambda (a) (match a '(col expr)
				(list (symbol "set") changed_rows_sym
					(list (symbol "set_assoc") changed_rows_sym col (transform_trigger_expr expr))))))))

		/* Helper to compile a single statement */
		/* For BEFORE triggers with SET: returns (set changed_rows ...) statements */
		/* For other statements: returns the statement as-is */
		(define compile_stmt (lambda (stmt)
			(if (not (list? stmt)) nil
				(begin
					(define tag (car stmt))
					(if (equal? tag '!set)
						/* SET NEW.col = expr - stmt is (!set a1 a2 ...) where ai are (col expr) pairs */
						(if is_after
							(error "SET NEW is not allowed in AFTER triggers")
							(compile_set_assignments (cdr stmt)))
						(if (equal? tag '!if)
							/* IF condition THEN stmts END IF - stmt is (!if condition then_stmts) */
							(begin
								(define condition (car (cdr stmt)))
								(define then_stmts (car (cdr (cdr stmt))))
								/* Compile statements inside IF - flatten nested lists from SET */
								(define compiled_then (merge (map then_stmts compile_stmt)))
								(define valid_then (filter compiled_then (lambda (s) (not (nil? s)))))
								/* Use !begin (no new scope) so set writes to outer changed_rows */
								(define then_body (if (> (count valid_then) 1)
									(cons '!begin valid_then)
									(if (> (count valid_then) 0) (car valid_then) nil)))
								/* Return as single-element list to be flattened by caller */
								(list (list (symbol "if") (transform_trigger_expr condition) then_body)))
							(if (equal? tag '!insert)
								/* INSERT INTO table - stmt is (!insert tbl cols vals) */
								(begin
									(define tbl (car (cdr stmt)))
									(define cols (car (cdr (cdr stmt))))
									(define vals (car (cdr (cdr (cdr stmt)))))
									(list (list (symbol "insert") schema tbl
										(cons (symbol "list") cols)
										(cons (symbol "list") (map vals (lambda (row) (cons (symbol "list") (map row transform_trigger_expr)))))
										(list (symbol "list")) nil false nil)))
								(if (equal? tag '!update)
									/* UPDATE table SET ... WHERE ... - stmt is (!update tbl assignments where) */
									(begin
										(define tbl (car (cdr stmt)))
										(define assignments (car (cdr (cdr stmt))))
										(define where (car (cdr (cdr (cdr stmt)))))
										(list (list (symbol "update") schema tbl
											(list (symbol "lambda") '() (if (nil? where) true (transform_trigger_expr where)))
											(cons (symbol "list") (map assignments (lambda (a) (match a '(col expr) (list (symbol "list") col (transform_trigger_expr expr))))))
											false)))
									(if (equal? tag '!delete)
										/* DELETE FROM table WHERE ... - stmt is (!delete tbl where) */
										(begin
											(define tbl (car (cdr stmt)))
											(define where (car (cdr (cdr stmt))))
											(list (list (symbol "delete") schema tbl
												(list (symbol "lambda") '() (if (nil? where) true (transform_trigger_expr where))))))
										/* Unknown statement type */
										nil
					)))))
				)
		)))

		(match body
			/* BEGIN...END block - compile all statements */
			(cons '!begin stmts) (begin
				/* Compile all statements - each may return a list of statements */
				(define compiled_stmts (merge (map stmts compile_stmt)))
				/* Filter out nil statements */
				(define valid_stmts (filter compiled_stmts (lambda (s) (not (nil? s)))))
				/* For BEFORE triggers: init changed_rows from NEW, return it at end */
				/* For AFTER triggers: just execute statements */
				(if is_after
					(if (> (count valid_stmts) 0)
						(list (symbol "lambda") params (cons (symbol "begin") valid_stmts))
						(list (symbol "lambda") params nil))
					/* BEFORE trigger: wrap with changed_rows handling */
					/* Use outer begin for define, inner !begin (no new scope) for statements */
					/* Wrap in begin: define changed_rows, execute stmts, return changed_rows */
					(list (symbol "lambda") params
						(list (symbol "begin")
							(list (symbol "define") changed_rows_sym (symbol "NEW"))
							(cons '!begin valid_stmts)
							changed_rows_sym)))
			)
			/* SET assignments (legacy format) - body is AST (list (col1 expr1) ...), eval to get actual list */
			(cons (symbol list) assignments) (begin
				/* Validate: SET NEW is not allowed in AFTER triggers */
				(if is_after
					(error "SET NEW is not allowed in AFTER triggers - the row has already been written")
					(list (symbol "lambda") params (build_trigger_sets (symbol "NEW") assignments)))
			)
			/* Unknown body type - return no-op */
			_ (list (symbol "lambda") params nil)
		)
	)))

	/* Simple trigger statements (non-IF) */
	(define sql_trigger_simple_stmt (parser (or
		/* SET NEW.col = expr; */
		(parser '(
			(atom "SET" true)
			(define assignments (+ (parser '(
				(atom "NEW" true) "." (define col sql_identifier) "=" (define expr sql_expression)
			) '(col expr)) ","))
			(atom ";" false)
		) (cons '!set assignments))
		/* INSERT INTO table (...) VALUES (...); */
		(parser '(
			(atom "INSERT" true) (atom "INTO" true) (define tbl sql_identifier)
			"(" (define cols (+ sql_identifier ",")) ")"
			(atom "VALUES" true)
			(define values (+ (parser '("(" (define v (+ sql_expression ",")) ")") v) ","))
			(atom ";" false)
		) (list '!insert tbl cols values))
		/* UPDATE table SET col=val WHERE condition; */
		(parser '(
			(atom "UPDATE" true) (define tbl sql_identifier)
			(atom "SET" true) (define assignments (+ (parser '((define col sql_identifier) "=" (define expr sql_expression)) '(col expr)) ","))
			(? (atom "WHERE" true) (define where sql_expression))
			(atom ";" false)
		) (list '!update tbl assignments where))
		/* DELETE FROM table WHERE condition; */
		(parser '(
			(atom "DELETE" true) (atom "FROM" true) (define tbl sql_identifier)
			(? (atom "WHERE" true) (define where sql_expression))
			(atom ";" false)
		) (list '!delete tbl where))
	)))

	/* Full trigger statement parser including IF...THEN...END IF */
	/* IF body can contain simple statements (no nested IF for now) */
	(define sql_trigger_stmt (parser (or
		/* IF condition THEN stmts END IF; */
		(parser '(
			(atom "IF" true)
			(define condition sql_expression)
			(atom "THEN" true)
			(define then_stmts (* sql_trigger_simple_stmt))
			(atom "END" true) (atom "IF" true)
			(atom ";" false)
		) (list '!if condition then_stmts))
		sql_trigger_simple_stmt
	)))

	/* Trigger body parser: handles BEGIN...END or single statement */
	/* Uses (capture ...) to get both raw SQL text and parsed result */
	/* Result is (raw_sql parsed_body) where parsed_body is the semantic representation */
	(define sql_trigger_body (parser (capture (or
		/* BEGIN...END block - statements must end with semicolon */
		(parser '(
			(atom "BEGIN" true)
			(define stmts (* sql_trigger_stmt))
			(atom "END" true)
		) (cons '!begin stmts))
		/* Single SET statement for BEFORE triggers (no semicolon needed) */
		(parser '(
			(atom "SET" true)
			(define assignments (+ (parser '(
				(atom "NEW" true) "." (define col sql_identifier) "=" (define expr sql_expression)
			) '(col expr)) ","))
		) (cons (quote list) assignments))
	))))

	(define sql_column_attributes (parser (define sub (* (or
		(parser '((atom "PRIMARY" true) (atom "KEY" true)) '("primary" true))
		(parser (atom "PRIMARY" true) '("primary" true))
		(parser '((atom "UNIQUE" true) (atom "KEY" true)) '("unique" true))
		(parser (atom "UNIQUE" true) '("unique" true))
		(parser (atom "AUTO_INCREMENT" true) '("auto_increment" true))
		(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
		(parser (atom "NULL" true) '("null" true))
		(parser '((atom "DEFAULT" true) (define default sql_literal)) '("default" default))
		(parser '((atom "DEFAULT" true) (atom "CURRENT_TIMESTAMP" true)) '("default" '('now)))
		(parser '((atom "ON" true) (atom "UPDATE" true) (define default sql_literal)) '("update" default))
		(parser '((atom "COMMENT" true) (define comment sql_expression)) '("comment" comment))
		(parser '((atom "COLLATE" true) (define comment sql_identifier)) '("collate" comment))
		(parser (atom "UNSIGNED" true) '()) /* ignore */
		/* TODO: GENERATED ALWAYS AS expr */
	))) (merge sub)))

	(define sql_expression (parser (or
		(parser '((atom "@" true) (define var sql_identifier_unquoted) (atom ":=" true) (define value sql_expression)) '((quote session) var value))
		(parser '((define a sql_expression1) (atom "OR" true) (define b (+ sql_expression1 (atom "OR" true)))) (cons (quote or) (cons a b)))
		sql_expression1
	)))
	(define sql_expression1 (parser (or
		(parser '((define a sql_expression2) (atom "AND" true) (define b (+ sql_expression2 (atom "AND" true)))) (cons (quote and) (cons a b)))
		sql_expression2
	)))

	(define sql_expression2 (parser (or
		/* IN (SELECT ...) and NOT IN (SELECT ...) -> pseudo operator, planner will lower or reject */
		(parser '((define a sql_expression3) (atom "IN" true) "(" (define sub sql_select) ")") '('inner_select_in a sub))
		(parser '((define a sql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define sub sql_select) ")") (list (quote not) (list (quote inner_select_in) a sub)))
		/* Collation-aware comparisons (MySQL): enforce given collation on string comparisons */
		(parser '((define a sql_expression3) (atom "COLLATE" true) (define collation sql_identifier) "=" (define b sql_expression2)) '('equal_collate a b collation))
		(parser '((define a sql_expression3) (atom "COLLATE" true) (define collation sql_identifier) "==" (define b sql_expression2)) '('equal_collate a b collation))
		(parser '((define a sql_expression3) (atom "COLLATE" true) (define collation sql_identifier) "<>" (define b sql_expression2)) '('notequal_collate a b collation))
		(parser '((define a sql_expression3) (atom "COLLATE" true) (define collation sql_identifier) "!=" (define b sql_expression2)) '('notequal_collate a b collation))
		(parser '((define a sql_expression3) "==" (define b sql_expression2)) '((quote equal??) a b))
		(parser '((define a sql_expression3) "=" (define b sql_expression2)) '((quote equal??) a b))
		(parser '((define a sql_expression3) "<>" (define b sql_expression2)) '((quote not) '((quote equal??) a b)))
		(parser '((define a sql_expression3) "!=" (define b sql_expression2)) '((quote not) '((quote equal??) a b)))
		(parser '((define a sql_expression3) "<=" (define b sql_expression2)) '((quote <=) a b))
		(parser '((define a sql_expression3) ">=" (define b sql_expression2)) '((quote >=) a b))
		(parser '((define a sql_expression3) "<" (define b sql_expression2)) '((quote <) a b))
		(parser '((define a sql_expression3) ">" (define b sql_expression2)) '((quote >) a b))
		(parser '((define a sql_expression3) (atom "COLLATE" true) (define collation sql_identifier) (atom "LIKE" true) (define b sql_expression2)) '('strlike a b collation))
		/* MySQL default collation is case-insensitive in this project (utf8mb4_general_ci). */
		(parser '((define a sql_expression3) (atom "LIKE" true) (define b sql_expression2)) '('strlike a b "utf8mb4_general_ci"))
		(parser '((define a sql_expression3) (atom "NOT" true) (atom "LIKE" true) (define b sql_expression2)) '('not '('strlike a b "utf8mb4_general_ci")))
		(parser '((define a sql_expression3) (atom "IN" true) "(" (define b (+ sql_expression ",")) ")") '('contains? (cons list b) a))
		(parser '((define a sql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define b (+ sql_expression ",")) ")") (list (quote not) (cons (quote contains?) (cons (cons (quote list) b) (list a)))))
		/* BETWEEN operator: expr BETWEEN low AND high -> a >= low AND a <= high */
		(parser '((define a sql_expression3) (atom "BETWEEN" true) (define low sql_expression3) (atom "AND" true) (define high sql_expression3)) (list (quote and) (list (quote >=) a low) (list (quote <=) a high)))
		(parser '((define a sql_expression3) (atom "NOT" true) (atom "BETWEEN" true) (define low sql_expression3) (atom "AND" true) (define high sql_expression3)) (list (quote not) (list (quote and) (list (quote >=) a low) (list (quote <=) a high))))
		sql_expression3
	)))

	(define sql_expression3 (parser (or
		/* date + INTERVAL n UNIT */
		(parser '((define a sql_expression4) "+" (atom "INTERVAL" true) (define n sql_expression4) (define unit sql_identifier_unquoted)) '('date_add a n unit))
		/* date - INTERVAL n UNIT */
		(parser '((define a sql_expression4) "-" (atom "INTERVAL" true) (define n sql_expression4) (define unit sql_identifier_unquoted)) '('date_sub a n unit))
		(parser '((define a sql_expression4) "+" (define b sql_expression3)) '((quote +) a b))
		(parser '((define a sql_expression4) "-" (define b sql_expression3)) '((quote -) a b))
		sql_expression4
	)))

	(define sql_expression4 (parser (or
		(parser '((define a sql_expression5) "*" (define b sql_expression4)) '((quote *) a b))
		(parser '((define a sql_expression5) "/" (define b sql_expression4)) '((quote /) a b))
		(parser '((define a sql_expression5) "%" (define b sql_expression4)) (sql_mod_expr a b))
		sql_expression5
	)))

	(define sql_expression5 (parser (or
		(parser '((atom "NOT" true) (define expr sql_expression6)) '('not expr))
		(parser '((define expr sql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr sql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		sql_expression6
	)))

	(define sql_expression6 (parser (or
		/* Scalar subselect in expressions: (SELECT ...) */
		(parser '("(" (define sub sql_select) ")") '('inner_select sub))
		(parser '("(" (define a sql_expression) ")") a)

		/* EXISTS (SELECT ...) */
		(parser '((atom "EXISTS" true) "(" (define sub sql_select) ")") '('inner_select_exists sub))
		(parser '((atom "CASE" true) (define conditions (* (parser '((atom "WHEN" true) (define a sql_expression) (atom "THEN" true) (define b sql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch sql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define e sql_expression) ")") '('count_distinct e))
		(parser '((atom "COUNT" true) "(" "*" ")") '((quote aggregate) 1 (quote +) 0))
		/* COUNT(expr): count non-NULL values -> map to (if (nil? expr) 0 1), reduce +, neutral 0 */
		(parser '((atom "COUNT" true) "(" (define e sql_expression) ")") '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0))
		(parser '((atom "SUM" true) "(" (define s sql_expression) ")") '('aggregate s (quote +) 0))
		(parser '((atom "AVG" true) "(" (define s sql_expression) ")") '((quote /) '('aggregate s (quote +) 0) '('aggregate 1 (quote +) 0)))
		(parser '((atom "MIN" true) "(" (define s sql_expression) ")") '('aggregate s 'min nil))
		(parser '((atom "MAX" true) "(" (define s sql_expression) ")") '('aggregate s 'max nil))
		(parser '((atom "GROUP_CONCAT" true) "(" (define s sql_expression) (atom "SEPARATOR" true) (define sep sql_expression) ")") '('aggregate '('concat s) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil))
		(parser '((atom "GROUP_CONCAT" true) "(" (define s sql_expression) ")") '('aggregate '('concat s) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a "," 'b))) nil))

		(parser '((atom "DATABASE" true) "(" ")") schema)
		(parser '((atom "UNIX_TIMESTAMP" true) "(" ")") '('now))
		(parser '((atom "UNIX_TIMESTAMP" true) "(" (define p psql_expression) ")") '('parse_date p))

		/* DATE literal: DATE 'yyyy-mm-dd' */
		(parser '((atom "DATE" true) (define s sql_string)) '('parse_date s))

		/* CURRENT_DATE / CURRENT_DATE() */
		(parser '((atom "CURRENT_DATE" true) "(" ")") '('current_date))
		(parser (atom "CURRENT_DATE" true) '('current_date))

		/* EXTRACT(field FROM expr) */
		(parser '((atom "EXTRACT" true) "(" (define field sql_identifier_unquoted) (atom "FROM" true) (define e sql_expression) ")") '('extract_date e field))

		/* DATE_ADD(expr, INTERVAL n UNIT) */
		(parser '((atom "DATE_ADD" true) "(" (define e sql_expression) "," (atom "INTERVAL" true) (define n sql_expression) (define unit sql_identifier_unquoted) ")") '('date_add e n unit))
		/* DATE_SUB(expr, INTERVAL n UNIT) */
		(parser '((atom "DATE_SUB" true) "(" (define e sql_expression) "," (atom "INTERVAL" true) (define n sql_expression) (define unit sql_identifier_unquoted) ")") '('date_sub e n unit))

		/* SUBSTRING(expr FROM start FOR len) - SQL standard syntax */
		(parser '((atom "SUBSTRING" true) "(" (define s sql_expression) (atom "FROM" true) (define start sql_expression) (atom "FOR" true) (define len sql_expression) ")") '((quote sql_substr) s start len))

		/* DATE(expr) - extract date part */
		(parser '((atom "DATE" true) "(" (define e sql_expression) ")") '('date_trunc_day e))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "UNSIGNED" true) ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "SIGNED" true) ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "INTEGER" true) ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "DECIMAL" true) "(" sql_int "," sql_int ")" ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "DECIMAL" true) "(" sql_int ")" ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "DECIMAL" true) ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "CHAR" true) (atom "CHARACTER" true) (atom "SET" true) (atom "utf8" true) ")") '('concat p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "DECIMAL" true) "(" sql_int "," sql_int ")" ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "DECIMAL" true) "(" sql_int ")" ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "DECIMAL" true) ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "UNSIGNED" true) ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "SIGNED" true) ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p sql_expression) "," (atom "INTEGER" true) ")") '('simplify p))
		(parser '((atom "CONCAT" true) "(" (define p (+ sql_expression ",")) ")") (cons 'concat p))
		/* TRIM/LTRIM/RTRIM as explicit parser rules for reliable dispatch */
		(parser '((atom "TRIM" true) "(" (define e sql_expression) ")") '((quote sql_trim) e))
		(parser '((atom "LTRIM" true) "(" (define e sql_expression) ")") '((quote sql_ltrim) e))
		(parser '((atom "RTRIM" true) "(" (define e sql_expression) ")") '((quote sql_rtrim) e))

		(parser '((atom "COALESCE" true) "(" (define args (* sql_expression ",")) ")") (cons (quote coalesceNil) args))
		/* MySQL IFNULL(val, default) — alias for COALESCE with 2 args */
		(parser '((atom "IFNULL" true) "(" (define a sql_expression) "," (define b sql_expression) ")") '((quote coalesceNil) a b))
		(parser '((atom "MOD" true) "(" (define a sql_expression) "," (define b sql_expression) ")") (sql_mod_expr a b))
		/* MySQL LAST_INSERT_ID(): direct session lookup to support session scoping */
		(parser '((atom "LAST_INSERT_ID" true) "(" ")") '('session "last_insert_id"))
		/* MySQL IF(condition, true_expr, false_expr) with short-circuit semantics */
		(parser '((atom "IF" true) "(" (define cond sql_expression) "," (define t sql_expression) "," (define f sql_expression) ")") '((quote if) cond t f))
		(parser '((atom "VALUES" true) "(" (define e sql_identifier_unquoted) ")") '('get_column "VALUES" true e true)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "VALUES" true) "(" (define e sql_identifier_quoted) ")") '('get_column "VALUES" true e false)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "pg_catalog" true) "." (atom "set_config" true) "(" sql_expression "," sql_expression "," sql_expression ")") nil) /* ignore */
		(parser '((atom "pg_catalog" true) "." (atom "setval" true) "(" "'" sql_identifier "." (define key sql_identifier) "'" "," (define val sql_expression) "," sql_expression ")") (match key (regex "(.*)_(.*?)_seq" _ tbl col) '('altercolumn schema tbl col "auto_increment" val) (error "unknown pg_catalog key: " key)))

		(parser (atom "NULL" true) 'nil)
		(parser (atom "TRUE" true) true)
		(parser (atom "FALSE" true) false)
		(parser (atom "ON" true) true)
		(parser (atom "OFF" true) false)
		(parser '((atom "@" true) (define var sql_identifier_unquoted)) '('session var))
		/* MySQL system variables: @@var, @@GLOBAL.var, @@SESSION.var */
		(parser '((atom "@@" true) (? (or (atom "GLOBAL" true) (atom "SESSION" true)) (? (atom "." true))) (define var sql_identifier_unquoted)) '('globalvars var))
		(parser '((atom "@@" true) (define var sql_identifier_unquoted)) '('globalvars var))
		/* LEFT(str, n) -- special case because LEFT is a reserved keyword (LEFT JOIN) */
		(parser '((atom "LEFT" true) "(" (define s sql_expression) "," (define n sql_expression) ")") '((quote sql_substr) s 1 n))
		/* RIGHT(str, n) -- special case because RIGHT is a reserved keyword */
		(parser '((atom "RIGHT" true) "(" (define s sql_expression) "," (define n sql_expression) ")") '((quote if) '((quote nil?) s) nil '((quote sql_substr) s '((quote +) 1 '((quote -) '((quote strlen) s) n)) n)))
		(parser '((define fn sql_identifier_unquoted) "(" (define args (* sql_expression ",")) ")") (cons (coalesce (sql_builtins (toUpper fn)) (error "unknown function " fn)) args))
		sql_number
		sql_string
		sql_column
	)))

	(define tabledefs (parser (or
		/* TODO: left [outer] join, right [outer] join recursive buildup */
		(parser '((define l tabledefs) (define x (or
			(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e sql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl true e))))
			(parser '((atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e sql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl false e))))
			(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tabledefs)) r)
		))) (merge l x))
		(parser '((define l tabledef) (atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledefs) (atom "ON" true) (define e sql_expression)) (match l '(id schema tbl _ nil) (cons '(id schema tbl true e) r)))
		(parser (define t tabledef) '(t))
	)))
	(define tabledef (parser (or
		(parser '((atom "(" true) (define query sql_select) (atom ")" true) (atom "AS" true) (define id sql_identifier)) '(id schema query false nil)) /* inner select as from */
		(parser '((atom "(" true) (define query sql_select) (atom ")" true) (define id sql_identifier)) '(id schema query false nil)) /* inner select as from */
		/* TODO: case insensititive table search */
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier) (define id sql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
		(parser '((define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl sql_identifier) (define id sql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl sql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
	)))

	/* bring those variables into a defined state */
	(define from nil)
	(define condition nil)
	(define group nil)
	(define having nil)
	(define order nil)
	(define limit nil)
	(define offset nil)
	(define sql_select (parser '(
		(atom "SELECT" true)
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil false "*" false)))
			(parser '((define tbl sql_identifier_quoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define tbl sql_identifier_unquoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define e sql_expression) (atom "AS" true) (define title sql_identifier)) '(title e))
			(parser '((define e sql_expression) (atom "AS" true) (define title sql_string)) '(title e))
			/* capture sql_expression to get raw SQL text for column naming */
			(parser (define captured (capture sql_expression)) '((extract_title_or_sql captured) (car (cdr captured))))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
		)
		(define condition (or (parser '(
			(atom "WHERE" true)
			(define condition2 sql_expression)
		) condition2) (empty true)))
		/* GROUP BY + HAVING */
		(?
			(atom "GROUP" true)
			(atom "BY" true)
			(define group (+
				sql_expression
				(atom "," true)
			))
		)
		(?
			(atom "HAVING" true)
			(define having sql_expression)
		)
		/* ORDER BY + LIMIT */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col sql_expression)
					(? (atom "COLLATE" true) (define coll sql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))

				(atom "," true)
			))
		)
		(?
			(atom "LIMIT" true)
			(or
				'((define offset sql_expression) (atom "," true) (define limit sql_expression))
				'((define limit sql_expression) (atom "OFFSET" true) (define offset sql_expression))
				'((define limit sql_expression))
			)
		)
	) '(schema (if (nil? from) '() (merge from)) (merge cols) condition group having order limit offset)))

	(define sql_update (parser '(
		(atom "UPDATE" true)
		/* TODO: UPDATE tbl FROM tbl, tbl, tbl */
		(define tbl sql_identifier) /* TODO: ignorecase */
		(atom "SET" true)
		(define cols (+ (or
			/* TODO: tbl.identifier */
			(parser '((define title sql_identifier) "=" (define e sql_expression)) '(title e))
		) ","))
		(? '(
			(atom "WHERE" true)
			(define condition sql_expression)
		))
		/* ORDER BY + LIMIT for partial updates */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col sql_expression)
					(? (atom "COLLATE" true) (define coll sql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))
				(atom "," true)
			))
		)
		(?
			(atom "LIMIT" true)
			(or
				'((define offset sql_expression) (atom "," true) (define limit sql_expression))
				'((define limit sql_expression) (atom "OFFSET" true) (define offset sql_expression))
				'((define limit sql_expression))
			)
		)
	) (begin
			/* policy: write access check */
			(if policy (policy schema tbl true) true)
			(define replace_find_column (lambda (expr) (match expr
				'((symbol get_column) nil _ col ci) '((quote get_column) tbl false col ci) /* TODO: case insensitive column */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))
			replace_find_column /* workaround for optimizer bug: variable bindings in parsers */
			(set cols (map_assoc (merge cols) (lambda (col expr) (replace_find_column expr))))
			(set condition (replace_find_column (coalesceNil condition true)))
			(set filtercols (extract_columns_for_tblvar tbl condition))
			(set scancols (merge_unique (extract_assoc cols (lambda (col expr) (extract_columns_for_tblvar tbl expr)))))
			(define mapcols (cons list (cons "$update" scancols)))
			(define mapfn '('lambda
				(cons (quote $update) (map scancols (lambda (col) (symbol (concat tbl "." col)))))
				'((quote if) '((quote $update) (cons (quote list) (map_assoc cols (lambda (col expr) (replace_columns_from_expr expr))))) 1 0)
			))
			(define filterfn '((quote lambda) (map filtercols (lambda(col) (symbol (concat tbl "." col)))) (replace_columns_from_expr condition)))
			(if (or (not (nil? order)) (not (nil? limit)))
				(begin
					(define ordercols (if (nil? order) '() (map order (lambda (x)
						(car (cdr (cdr (cdr (replace_find_column (car x))))))
					))))
					(define dirs (if (nil? order) '() (map order (lambda (x) (car (cdr x))))))
					'((quote scan_order)
						schema
						tbl
						(cons list filtercols)
						filterfn
						(cons list ordercols)
						(cons list dirs)
						(coalesceNil offset 0)
						(coalesceNil limit -1)
						mapcols
						mapfn
						(quote +)
						0
					)
				)
				'((quote scan)
					schema
					tbl
					(cons list filtercols)
					filterfn
					mapcols
					mapfn
					(quote +)
					0
				)
			)
	)))

	(define sql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)
		/* schema-qualified */
		(? (define schema2 sql_identifier) ".") (define tbl sql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition sql_expression)
		))
		/* ORDER BY + LIMIT for partial deletes */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col sql_expression)
					(? (atom "COLLATE" true) (define coll sql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))
				(atom "," true)
			))
		)
		(?
			(atom "LIMIT" true)
			(or
				'((define offset sql_expression) (atom "," true) (define limit sql_expression))
				'((define limit sql_expression) (atom "OFFSET" true) (define offset sql_expression))
				'((define limit sql_expression))
			)
		)
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define replace_find_column (lambda (expr) (match expr
				'((symbol get_column) nil _ col ci) '((quote get_column) tbl false col ci) /* TODO: case insensititive column */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))
			replace_find_column /* workaround for optimizer bug: variable bindings in parsers */
			(set condition (replace_find_column (coalesceNil condition true)))
			(set filtercols (extract_columns_for_tblvar tbl condition))
			(define filterfn '((quote lambda) (map filtercols (lambda(col) (symbol (concat tbl "." col)))) (replace_columns_from_expr condition)))
			(if (or (not (nil? order)) (not (nil? limit)))
				(begin
					(define ordercols (if (nil? order) '() (map order (lambda (x)
						(car (cdr (cdr (cdr (replace_find_column (car x))))))
					))))
					(define dirs (if (nil? order) '() (map order (lambda (x) (car (cdr x))))))
					'((quote scan_order)
						(coalesce schema2 schema)
						tbl
						(cons list filtercols)
						filterfn
						(cons list ordercols)
						(cons list dirs)
						(coalesceNil offset 0)
						(coalesceNil limit -1)
						'(list "$update")
						'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
						(quote +)
						0
					)
				)
				'((quote scan)
					(coalesce schema2 schema)
					tbl
					(cons list filtercols)
					filterfn
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
				)
			)
	)))

	(define sql_insert_into (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
		(? (define schema2 sql_identifier) ".")
		(define tbl sql_identifier) /* TODO: ignorecase */
		(? "("
			(define coldesc (*
				sql_identifier
				","))
			")")
		(atom "VALUES" true)
		(define datasets (* (parser '(
			"("
			(define dataset (* sql_expression ","))
			")"
		) dataset) ","))
		(define updaterows (? (parser '(
			(atom "ON" true)
			(atom "DUPLICATE" true)
			(atom "KEY" true)
			(atom "UPDATE" true)
			/* TODO: ignorecase */
			(define updaterows (+ (parser '((define col sql_identifier) (atom "=" false) (define value sql_expression)) '(col value)) ","))
		) updaterows)))
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
			(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
			(define coldesc (coalesce coldesc (map (show (coalesce schema2 schema) tbl) (lambda (col) (col "Field")))))
			'('insert (coalesce schema2 schema) tbl (cons list coldesc) (cons list (map datasets (lambda (dataset) (cons list dataset)))) (cons list updatecols)
				(if (and ignoreexists (nil? updaterows))
					'((quote lambda) '() 0)
					(if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))
				false '('lambda '('id) '('session "last_insert_id" 'id)))
	)))

	(define sql_insert_select (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
		(? (define schema2 sql_identifier) ".")
		(define tbl sql_identifier) /* TODO: ignorecase */
		(? "("
			(define coldesc (*
				sql_identifier
				","))
			")")
		(define inner sql_select) /* INNER SELECT */
		(define datasets (* (parser '(
			"("
			(define dataset (* sql_expression ","))
			")"
		) dataset) ","))
		(define updaterows (? (parser '(
			(atom "ON" true)
			(atom "DUPLICATE" true)
			(atom "KEY" true)
			(atom "UPDATE" true)
			/* TODO: ignorecase */
			(define updaterows (+ (parser '((define col sql_identifier) (atom "=" false) (define value sql_expression)) '(col value)) ","))
		) updaterows)))
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
			(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
			(define coldesc (coalesce coldesc (map (show (coalesce schema2 schema) tbl) (lambda (col) (col "Field")))))
			'('begin
				'('set 'resultrow '('lambda '('item) '('insert (coalesce schema2 schema) tbl (cons list coldesc) (cons list '((cons list (map (produceN (count coldesc)) (lambda (i) '('nth 'item (+ (* i 2) 1))))))) (cons list updatecols)
					(if (and ignoreexists (nil? updaterows))
						'((quote lambda) '() 0)
						(if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))
					'('lambda '('id) '('session "last_insert_id" 'id)))))
				(apply build_queryplan (apply untangle_query inner))
			)
	)))

	(define sql_foreign_key_mode (parser (or
		(parser (atom "RESTRICT" true) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser (atom "SET NULL" true) "set null")
	)))

	(define sql_create_table (parser '(
		(atom "CREATE" true)
		(atom "TABLE" true)
		(define ifnotexists (parser (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)) true))
		(define id (or (parser '((define schema2 sql_identifier) "." (define id sql_identifier)) id) sql_identifier))
		"("
		(define cols (* (or
			(parser '((atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ sql_identifier ",")) ")") '((quote list) "unique" "PRIMARY" (cons (quote list) cols)))
			(parser '((atom "UNIQUE" true) (atom "KEY" true) (define id sql_identifier) "(" (define cols (+ sql_identifier ",")) ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "CONSTRAINT" true) (define id (? sql_identifier)) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (define deletemode sql_foreign_key_mode)) (? (atom "ON" true) (atom "UPDATE" true) (define updatemode sql_foreign_key_mode))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2) updatemode deletemode))
			(parser '((atom "FOREIGN" true) (atom "KEY" true) (define id (? sql_identifier)) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true))) (? (atom "ON" true) (atom "UPDATE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true)))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2)))
			(parser '((atom "KEY" true) sql_identifier "(" (+ sql_identifier ",") ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list))) /* ignore index definitions */
			(parser '(
				(define col sql_identifier)
				(define type sql_identifier)
				(define dimensions (or
					(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
					(parser '("(" (define a sql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				/* column flags */
				(define typeparams sql_column_attributes)
			) '((quote list) "column" col type dimensions (cons 'list typeparams)))
		) ","))
		")"
		(define options (* (or
			(parser '((atom "CHARACTER" true) (atom "SET" true) (define id sql_identifier)) '("charset" id))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) '("engine" "memory"))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) '("engine" "sloppy"))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) '("engine" "logging"))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "CSV" true)) '("engine" "safe"))
			(parser '((atom "COMMENT" true) "=" (define value sql_string)) '("comment" value))
			(parser '((atom "DEFAULT" true) (atom "CHARSET" false) "=" (define id sql_identifier)) '("charset" id))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "COLLATE" true) (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define collation (regex "[0-9]+"))) '("auto_increment" collation))
		)))
	) '((quote createtable) (coalesce schema2 schema) id (cons (quote list) cols) (cons (quote list) (merge options)) ifnotexists)))

	(define sql_alter_table (parser '(
		(atom "ALTER" true)
		(atom "TABLE" true)
		(define id sql_identifier) /* TODO: ignorecase */
		(define alters (+ (or
			/* TODO
			(parser '((atom "ADD" true) (atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ sql_identifier ",")) ")") '((quote list) "unique" "PRIMARY" (cons (quote list) cols)))
			(parser '((atom "ADD" true) (atom "UNIQUE" true) (atom "KEY" true) (define id sql_identifier) "(" (define cols (+ sql_identifier ",")) ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "ADD" true) (atom "FOREIGN" true) (atom "KEY" true) (define id (? sql_identifier)) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true))) (? (atom "ON" true) (atom "UPDATE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true)))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2))) */
			(parser '((atom "ADD" true) (atom "KEY" true) sql_identifier "(" (+ sql_identifier ",") ")" (? (atom "USING" true) (atom "BTREE" true))) nil) /* ignore index definitions */
			(parser '((atom "ADD" true) (atom "FULLTEXT" true) (? (atom "INDEX" true)) (? sql_identifier) "(" (+ sql_identifier ",") ")") (lambda (id) true)) /* ignore fulltext index */
			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define cname (? sql_identifier))) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "UPDATE" true) (define updatemode sql_foreign_key_mode)) (? (atom "ON" true) (atom "DELETE" true) (define deletemode sql_foreign_key_mode))) (lambda (id) true))
			(parser '((atom "DROP" true) (atom "FOREIGN" true) (atom "KEY" true) (define fk sql_identifier)) (lambda (id) true))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true) (define fk sql_identifier)) (lambda (id) true))
			(parser '((atom "DROP" true) (atom "PRIMARY" true) (atom "KEY" true)) (lambda (id) true))
			(parser '((atom "DROP" true) (atom "INDEX" true) (define idx sql_identifier)) (lambda (id) true))
			(parser '((atom "ADD" true) (?(atom "COLUMN" true))
				(define col sql_identifier)
				(define type sql_identifier)
				(define dimensions (or
					(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
					(parser '("(" (define a sql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				(define typeparams sql_column_attributes)
			) (lambda (id) '((quote createcolumn) schema id col type dimensions (cons 'list typeparams))))
			(parser '((atom "MODIFY" true) (?(atom "COLUMN" true))
				(define col sql_identifier)
				(define type sql_identifier)
				(define dimensions (or
					(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
					(parser '("(" (define a sql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				(define typeparams sql_column_attributes)
			) (lambda (id) '('!begin '((quote altercolumn) schema id col "type" type) '((quote altercolumn) schema id col "dimensions" dimensions) 1)))
			(parser '((atom "CHANGE" true)
				(define oldcol sql_identifier)
				(define col sql_identifier)
				(define type sql_identifier)
				(define dimensions (or
					(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
					(parser '("(" (define a sql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				(define typeparams sql_column_attributes)
			) (lambda (id) '('!begin '((quote altercolumn) schema id col "type" type) '((quote altercolumn) schema id col "dimensions" dimensions) 1)))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col sql_identifier) (atom "TYPE" true) (define type sql_identifier) (define dimensions (or
				(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
				(parser '("(" (define a sql_int) ")") '((quote list) a))
				(parser empty '((quote list)))
			))) (lambda (id) '('!begin '((quote altercolumn) schema id col "type" type) '((quote altercolumn) schema id col "dimensions" dimensions) 1)))
			(parser '((atom "CONVERT" true) (atom "TO" true) (atom "CHARACTER" true) (atom "SET" true) (define charset sql_identifier) (? (atom "COLLATE" true) (define coll sql_identifier))) (lambda (id) true))
			(parser '((atom "DROP" true) (? (atom "COLUMN" true)) (define col sql_identifier)) (lambda (id) '((quote altertable) schema id "drop" col)))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) (lambda (id) '((quote altertable) schema id "engine" "memory")))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) (lambda (id) '((quote altertable) schema id "engine" "sloppy")))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) (lambda (id) '((quote altertable) schema id "engine" "logging")))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) (lambda (id) '((quote altertable) schema id "collation" collation)))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define ai (regex "[0-9]+"))) (lambda (id) '((quote altertable) schema id "auto_increment" ai)))
			/* ALTER COLUMN operations for defaults */
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col sql_identifier) (atom "SET" true) (atom "DEFAULT" true) (define def sql_expression)) (lambda (id) '((quote altercolumn) schema id col "default" def)))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col sql_identifier) (atom "DROP" true) (atom "DEFAULT" true)) (lambda (id) '((quote altercolumn) schema id col "default" nil)))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col sql_identifier) (atom "COLLATE" true) (define coll sql_identifier)) (lambda (id) '((quote altercolumn) schema id col "collation" coll)))
		) ","))
	) (cons '!begin (map alters (lambda (alter) (alter id))))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	(define p (parser (or
		(parser (atom "SHUTDOWN" true) (begin (if policy (policy "system" true true) true) '(shutdown)))
		(parser (define query sql_select) (apply build_queryplan (apply untangle_query query)))
		(parser '((atom "EXPLAIN" true) (define query sql_select)) '('resultrow '('list "code" (serialize (apply build_queryplan (apply untangle_query query))))))
		sql_insert_into
		sql_insert_select
		sql_create_table
		sql_alter_table
		sql_update
		sql_delete

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define ifnot (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true))) (define id sql_identifier)) (begin (if policy (policy "system" true true) true) '((quote createdatabase) id (if ifnot true false))))
		(parser '((atom "CREATE" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "user" '(list "username" "password" "admin") '(list '(list username '('password password) false)) '(list) '((quote lambda) '() '((quote error) "user already exists")))
		))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) "system" "user" '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "password" '('password password)))))
		))

		/* GRANT syntax (MySQL-style) -> reflect only admin and database-level access */
		/* GRANT ALL [PRIVILEGES] ON *.* TO user -> set admin true */
		(parser '((atom "GRANT" true) (atom "ALL" true) (? (atom "PRIVILEGES" true)) (atom "ON" true) (atom "*" true) (atom "." true) (atom "*" true) (atom "TO" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) "system" "user" '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" true))))
		))
		/* GRANT <anything> ON db.* TO user -> insert access */
		(parser '((atom "GRANT" true) (+ (or sql_identifier "," (atom "SELECT" true))) (atom "ON" true) (define db sql_identifier) (atom "." true) (or (atom "*" true) sql_identifier) (atom "TO" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "access" '('list "username" "database") '('list '('list username db)))
		))
		/* GRANT <anything> ON db.table TO user -> also insert access at db level */
		(parser '((atom "GRANT" true) (+ (or sql_identifier "," (atom "SELECT" true))) (atom "ON" true) (define db sql_identifier) (atom "." true) sql_identifier (atom "TO" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "access" '('list "username" "database") '('list '('list username db)))
		))

		/* REVOKE syntax (MySQL-style) -> mirror GRANT behavior */
		/* REVOKE ALL [PRIVILEGES] ON *.* FROM user -> set admin false */
		(parser '((atom "REVOKE" true) (atom "ALL" true) (? (atom "PRIVILEGES" true)) (atom "ON" true) (atom "*" true) (atom "." true) (atom "*" true) (atom "FROM" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) "system" "user" '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" false))))
		))
		/* REVOKE <anything> ON db.* FROM user -> delete access entry */
		(parser '((atom "REVOKE" true) (+ (or sql_identifier "," (atom "SELECT" true))) (atom "ON" true) (define db sql_identifier) (atom "." true) (or (atom "*" true) sql_identifier) (atom "FROM" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					"system"
					"access"
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal??) (quote username) username) '((quote equal??) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))
		/* REVOKE <anything> ON db.table FROM user -> treat as db-level and delete access entry */
		(parser '((atom "REVOKE" true) (+ (or sql_identifier "," (atom "SELECT" true))) (atom "ON" true) (define db sql_identifier) (atom "." true) sql_identifier (atom "FROM" true) (define username sql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					"system"
					"access"
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal??) (quote username) username) '((quote equal??) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))

		/* SHOW CREATE TABLE [schema.]table */
		(parser '((atom "SHOW" true) (atom "CREATE" true) (atom "TABLE" true) (define schema2 sql_identifier) (atom "." true) (define id sql_identifier))
			'((quote resultrow) '((quote list) "Table" id "Create Table" '((quote format_create_table) schema2 id))))
		(parser '((atom "SHOW" true) (atom "CREATE" true) (atom "TABLE" true) (define id sql_identifier))
			'((quote resultrow) '((quote list) "Table" id "Create Table" '((quote format_create_table) schema id))))
		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true) (? (atom "FROM" true) (define schema sql_identifier))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '((quote resultrow) '((quote list) "Table" (quote tbl))))))
		(parser '((atom "SHOW" true) (atom "TABLE" true) (atom "STATUS" true) (? (atom "FROM" true) (define schema sql_identifier) (? (atom "LIKE" true) (define likepattern sql_expression)))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '('if '('strlike 'tbl '('coalesce 'likepattern "%")) '((quote resultrow) '('list "name" 'tbl "rows" "1")))))) /* TODO: engine version row_format avg_row_length data_length max_data_length index_length data_free auto_increment create_time update_time check_time collation checksum create_options comment max_index_length temporary */
		(parser '((atom "DESCRIBE" true) (define schema2 sql_identifier) (atom "." true) (define id sql_identifier)) '((quote map) '((quote show) schema2 id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))
		(parser '((atom "DESCRIBE" true) (define id sql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))
		(parser '((atom "SHOW" true) (atom "FULL" true) (atom "COLUMNS" true) (atom "FROM" true) (define id sql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line))))) /* TODO: Field Type Collation Null Key Default Extra(auto_increment) Privileges Comment */

		/* SHOW TRIGGERS [FROM schema] */
		(parser '((atom "SHOW" true) (atom "TRIGGERS" true) (? (atom "FROM" true) (define tgtschema sql_identifier)))
			'((quote map) '((quote show_triggers) (coalesce tgtschema schema)) '((quote lambda) '((quote tr)) '((quote resultrow) (quote tr)))))

		/* SHOW INDEXES FROM t / SHOW INDEX FROM t / SHOW KEYS FROM t (no-op, returns empty) */
		(parser '((atom "SHOW" true) (or (atom "INDEXES" true) (atom "INDEX" true) (atom "KEYS" true)) (atom "FROM" true) sql_identifier (? (atom "WHERE" true) sql_expression)) "ignore")

		/* SHOW ENGINES: list engines recognized by CREATE/ALTER TABLE */
		(parser '((atom "SHOW" true) (atom "ENGINES" true)) (cons '!begin '(
			'((quote resultrow) '((quote list) "Engine" "SAFE"    "Support" "DEFAULT" "Comment" "Safe durable engine"              "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "LOGGING" "Support" "YES"     "Comment" "Append-only logging engine"      "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "MEMORY"  "Support" "YES"     "Comment" "In-memory engine"               "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "SLOPPY"  "Support" "YES"     "Comment" "Relaxed engine"                 "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "MyISAM"  "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "InnoDB"  "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "CSV"     "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
		)))
		/* SHOW {CHARSET|CHARACTER SET} [LIKE pattern] */
		(parser '((atom "SHOW" true) (or (atom "CHARSET" true) '((atom "CHARACTER" true) (atom "SET" true))) (? (atom "LIKE" true) (define likepattern sql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Charset" "utf8mb4" "Description" "UTF-8 Unicode" "Default collation" "utf8mb4_general_ci" "Maxlen" 4))
		)))
		/* SHOW COLLATION [LIKE pattern] */
		(parser '((atom "SHOW" true) (atom "COLLATION" true) (? (atom "LIKE" true) (define likepattern sql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Collation" "utf8mb4_general_ci" "Charset" "utf8mb4" "Id" 255 "Default" "YES" "Compiled" "YES" "Sortlen" 1))
			'((quote resultrow) '((quote list) "Collation" "utf8mb4_bin"        "Charset" "utf8mb4" "Id" 254 "Default" "NO"  "Compiled" "YES" "Sortlen" 1))
		)))
		/* SHOW PLUGINS: return empty set (ok for most clients) */
		(parser '((atom "SHOW" true) (atom "PLUGINS" true)) (quote true))

		/* SHOW [GLOBAL|SESSION] VARIABLES [LIKE pattern] */
		(parser '((atom "SHOW" true) (? (or (atom "GLOBAL" true) (atom "SESSION" true))) (atom "VARIABLES" true) (? (atom "LIKE" true) (define likepattern sql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Variable_name" "version"               "Value" "0.9"))
			'((quote resultrow) '((quote list) "Variable_name" "character_set_server" "Value" "utf8mb4"))
			'((quote resultrow) '((quote list) "Variable_name" "collation_server"     "Value" "utf8mb4_general_ci"))
			'((quote resultrow) '((quote list) "Variable_name" "lower_case_table_names" "Value" 0))
		)))
		(parser '((atom "SET" true) (atom "NAMES" true) (define charset sql_expression)) (quote true)) /* ignore */


		(parser '((atom "DROP" true) (atom "DATABASE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id sql_identifier)) (begin (if policy (policy "system" true true) true) '((quote dropdatabase) id (if if_exists true false))))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define schema sql_identifier) (atom "." true) (define id sql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id sql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "RENAME" true) (atom "TABLE" true) (define oldname sql_identifier) (atom "TO" true) (define newname sql_identifier)) '((quote renametable) schema oldname newname))
		(parser '((atom "SET" true) (? (atom "SESSION" true)) (define vars (* (parser '((? "@") (define key sql_identifier) "=" (define value sql_expression)) '((quote session) key value)) ","))) (cons '!begin vars))

		(parser '((atom "LOCK" true) (or (atom "TABLES" true) (atom "TABLE" true)) (+ (or sql_identifier '(sql_identifier (atom "AS" true) sql_identifier)) ",") (? (atom "READ" true)) (? (atom "LOCAL" true)) (? (atom "LOW_PRIORITY" true)) (? (atom "WRITE" true))) "ignore")
		(parser '((atom "UNLOCK" true) (or (atom "TABLES" true) (atom "TABLE" true))) "ignore")

		/* CREATE INDEX syntax acceptance (no-op unless UNIQUE; MemCP auto-indexes) */
		(parser '((atom "CREATE" true)
			(atom "UNIQUE" true) (atom "INDEX" true)
			(define idx sql_identifier)
			(atom "ON" true)
			(define tbl (or
				(parser '((define schema2 sql_identifier) (atom "." true) (define t sql_identifier)) '(schema2 t))
				(parser (define t sql_identifier) '(nil t))
			))
			"(" (define cols (+ sql_index_column ",")) ")"
			(? (atom "USING" true) (atom "BTREE" true))
		) (match tbl '(schema2 t) '('createkey (coalesce schema2 schema) t idx true (cons (quote list) cols))))
		(parser '((atom "CREATE" true)
			(atom "INDEX" true)
			(define idx sql_identifier)
			(atom "ON" true)
			(define tbl (or
				(parser '((define schema2 sql_identifier) (atom "." true) (define t sql_identifier)) '(schema2 t))
				(parser (define t sql_identifier) '(nil t))
			))
			"(" (define cols (+ sql_index_column ",")) ")"
			(? (atom "USING" true) (atom "BTREE" true))
		) "ignore")
		(parser '((atom "CREATE" true)
			(atom "UNIQUE" true) (atom "INDEX" true)
			(define idx sql_identifier)
			(atom "ON" true)
			(define tbl (or
				(parser '((define schema2 sql_identifier) (atom "." true) (define t sql_identifier)) '(schema2 t))
				(parser (define t sql_identifier) '(nil t))
			))
			(atom "USING" true) (atom "BTREE" true)
			"(" (define cols (+ sql_index_column ",")) ")"
		) (match tbl '(schema2 t) '('createkey (coalesce schema2 schema) t idx true (cons (quote list) cols))))
		(parser '((atom "DROP" true) (atom "INDEX" true) (define idx sql_identifier) (? (atom "ON" true) (define tbl sql_identifier))) "ignore")

		/* CREATE TRIGGER syntax */
		/* body from sql_trigger_body is (raw_sql parsed_body) due to capture wrapper */
		(parser '(
			(atom "CREATE" true)
			(atom "TRIGGER" true)
			(define name sql_identifier)
			(define timing (or
				(parser '((atom "BEFORE" true) (atom "INSERT" true)) "before_insert")
				(parser '((atom "AFTER" true) (atom "INSERT" true)) "after_insert")
				(parser '((atom "BEFORE" true) (atom "UPDATE" true)) "before_update")
				(parser '((atom "AFTER" true) (atom "UPDATE" true)) "after_update")
				(parser '((atom "BEFORE" true) (atom "DELETE" true)) "before_delete")
				(parser '((atom "AFTER" true) (atom "DELETE" true)) "after_delete")
			))
			(atom "ON" true)
			(define tbl sql_identifier)
			(atom "FOR" true) (atom "EACH" true) (atom "ROW" true)
			(define body sql_trigger_body)
		) (begin
				(define compiled (compile_trigger_body schema timing (car (cdr body))))
				(list 'createtrigger schema tbl name timing (car body) (eval compiled))
		))

		/* DROP TRIGGER syntax */
		(parser '((atom "DROP" true) (atom "TRIGGER" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define name sql_identifier))
			'((quote droptrigger) schema name (if if_exists true false)))

		/* USE database - change current schema */
		(parser '((atom "USE" true) (define db sql_identifier)) '('session "schema" db))

		/* ANALYZE TABLE (no-op) */
		(parser '((atom "ANALYZE" true) (atom "TABLE" true) sql_identifier) "ignore")

		/* transaction control */
		(parser '((atom "START" true) (atom "ACID" true) (atom "TRANSACTION" true)) '('tx_begin_acid 'session))
		(parser '((atom "START" true) (atom "TRANSACTION" true)) '('tx_begin 'session))
		(parser '((atom "BEGIN" true)) '('tx_begin 'session))
		(parser '((atom "COMMIT" true)) '('tx_commit 'session))
		(parser '((atom "ROLLBACK" true)) '('tx_rollback 'session))
		"" /* comment only command */
	)))
	((parser (define command p) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))

(define load_sql (lambda (schema stream) (begin
	(set state (newsession))
	(set resultrow print)
	(set session (newsession))
	(define sql_line (lambda (line) (begin
		(match line
			(concat "--" b) /* comment */ false
			(concat "DELIMITER " d "\r\n") /* delimiter change */ (state "delimiter" d)
			(concat "DELIMITER " d "\n") /* delimiter change */ (state "delimiter" d)
			(concat start (eval (state "delimiter")) rest) (begin
				/* command ended -> execute (at max one command per line) */
				(print (concat (state "sql") start))
				(set plan (parse_sql schema (concat (state "sql") start) nil))
				(print "SQL execute" plan)
				(eval plan)
				(state "sql" rest)
			)
			/* otherwise: append to cache */
			(state "sql" (concat (state "sql") line))
		)
	)))
	(state "delimiter" ";")
	(state "line" sql_line)
	(state "sql" "")
	(load stream (lambda (line) (begin
		((state "line") line)
	)) "\n")
)))
