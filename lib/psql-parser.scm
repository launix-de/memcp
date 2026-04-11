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

(define psql_identifier_unquoted (parser (not
	(regex "[a-zA-Z_][a-zA-Z0-9_]*")
	/* exceptions for things that can't be identifiers */
	(atom "NOT" true)
	(atom "IN" true)
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
	(atom "UNION" true)
	(atom "ALL" true)
	(atom "INSERT" true)
	(atom "ORDER" true)
	(atom "LIMIT" true)
	(atom "DELIMITER" true)
	(atom "TRIM" true)
	(atom "LTRIM" true)
	(atom "RTRIM" true)
	(atom "BETWEEN" true)
	(atom "INTERVAL" true)
)))
(define psql_identifier_quoted (parser '("\"" (define id (regex "(?:[^\"])+" false false)) "\"") (sql_unescape id))) /* with double quote */
(define psql_identifier (parser (define x (or psql_identifier_unquoted psql_identifier_quoted)) x))

(define psql_column (parser (or
	(parser '((define tbl psql_identifier_unquoted) "." (define col psql_identifier_unquoted)) '((quote get_column) tbl true col true))
	(parser '((define tbl psql_identifier_unquoted) "." (define col psql_identifier_quoted)) '((quote get_column) tbl true col false))
	(parser '((define tbl psql_identifier_quoted) "." (define col psql_identifier_unquoted)) '((quote get_column) tbl false col true))
	(parser '((define tbl psql_identifier_quoted) "." (define col psql_identifier_quoted)) '((quote get_column) tbl false col false))
	(parser (define col psql_identifier_quoted) '((quote get_column) nil true col false))
	(parser (define col psql_identifier_unquoted) '((quote get_column) nil true col true))
)))


(define psql_column_attributes (parser (define sub (* (or
	(parser '((atom "PRIMARY" true) (atom "KEY" true)) '("primary" true))
	(parser (atom "PRIMARY" true) '("primary" true))
	(parser '((atom "UNIQUE" true) (atom "KEY" true)) '("unique" true))
	(parser (atom "UNIQUE" true) '("unique" true))
	(parser (atom "AUTO_INCREMENT" true) '("auto_increment" true))
	(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
	(parser (atom "NULL" true) '("null" true))
	(parser '((atom "DEFAULT" true) (define default psql_expression)) '("default" default))
	(parser '((atom "ON" true) (atom "UPDATE" true) (define default psql_expression)) '("update" default))
	(parser '((atom "COMMENT" true) (define comment psql_expression)) '("comment" comment))
	(parser '((atom "COLLATE" true) (define comment sql_identifier)) '("collate" comment))
	(parser (atom "UNSIGNED" true) '()) /* ignore */
	/* TODO: GENERATED ALWAYS AS expr */
))) (merge sub)))

(define psql_int (parser (define x (regex "-?[0-9]+")) (simplify x)))
(define psql_number (parser (define x (regex "-?[0-9]+\.?[0-9]*(?:e-?[0-9]+)?" true)) (simplify x)))

(define psql_string (parser (or
	(parser '((atom "'" false) (define x (regex "(\\\\.|[^\\'])*" false false)) (atom "'" false false)) (sql_unescape x))
)))

/* SQL modulo expression: uses native mod builtin (NULL-safe, div-by-zero returns NULL) */
(define psql_mod_expr (lambda (a b) '('mod a b))
))

(define psql_type (parser (or
	(parser '((atom "character" true) (atom "varying" true)) "varying")
	(parser '((atom "double" true) (atom "precision" true)) "double")
	/* PostgreSQL timezone-aware types */
	(parser '((atom "TIMESTAMP" true) (atom "WITH" true) (atom "TIME" true) (atom "ZONE" true)) "TIMESTAMP")
	(parser '((atom "TIMESTAMP" true) (atom "WITHOUT" true) (atom "TIME" true) (atom "ZONE" true)) "DATETIME")
	(parser (atom "TIMESTAMPTZ" true) "TIMESTAMP")
	(parser '((atom "TIME" true) (atom "WITH" true) (atom "TIME" true) (atom "ZONE" true)) "TIME")
	(parser (atom "TIMETZ" true) "TIME")
	psql_identifier
)))

(define parse_psql (lambda (schema s policy) (begin

	/* counter for positional $N placeholders: each $N compiles to (session "vN") */
	(define placeholder_counter (newsession))
	(placeholder_counter "n" 0)

	/* derive the description of a column from its expression */
	(define extract_title (lambda (expr) (match expr
		'((symbol get_column) nil _ col _) col
		'((symbol get_column) tblvar _ col _) col /* x.y -> (concat tblvar "." col) */
		(cons sym args) /* function call */ (concat (cons sym (map args extract_title)))
		(concat expr)
	)))

	/* merge two arrays into a dict */
	(define zip_cols (lambda(cols tuple) (match cols
		(cons col cols) (cons col (cons (car tuple) (zip_cols cols (cdr tuple))))
		'()
	)))

	/* helper function for triggers and ON DUPLICATE: every column is just a symbol */
	(define replace_stupid (lambda (expr) (match expr
		'('get_column "VALUES" _ col _) (symbol (concat "NEW." col))
		'('get_column "excluded" _ col _) (symbol (concat "NEW." col))
		'('get_column _ _ col _) (symbol col) /* TODO: case matching */
		(cons head tail) (cons head (map tail replace_stupid))
		expr
	)))
	/* helper function for triggers and ON DUPLICATE: extract all used columns */
	(define extract_stupid (lambda (expr) (match expr
		'('get_column "VALUES" _ col _) '((concat "NEW." col))
		'('get_column "excluded" _ col _) '((concat "NEW." col))
		'('get_column _ _ col _) '(col)
		(cons head tail) (merge_unique (map tail extract_stupid))
		'()
	)))

	/* TODO: (expr), a + b, a - b, a * b, a / b */
	(define psql_expression (parser (or
		(parser '((atom "@" true) (define var psql_identifier_unquoted) (atom ":=" true) (define value psql_expression)) '((quote session) var value))
		(parser '((define a psql_expression1) (atom "OR" true) (define b (+ psql_expression1 (atom "OR" true)))) (cons (quote or) (cons a b)))
		psql_expression1
	)))
	(define psql_expression1 (parser (or
		(parser '((define a psql_expression2) (atom "AND" true) (define b (+ psql_expression2 (atom "AND" true)))) (cons (quote and) (cons a b)))
		psql_expression2
	)))

	(define psql_expression2 (parser (or
		/* IN (SELECT ...) and NOT IN (SELECT ...) -> pseudo operator, planner will lower or reject */
		(parser '((define a psql_expression3) (atom "IN" true) "(" (define sub psql_select) ")") '('inner_select_in a sub))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define sub psql_select) ")") (list (quote not) (list (quote inner_select_in) a sub)))
		(parser '((define a psql_expression3) "==" (define b psql_expression2)) '((quote equal??) a b))
		(parser '((define a psql_expression3) "=" (define b psql_expression2)) '((quote equal??) a b))
		(parser '((define a psql_expression3) "<>" (define b psql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a psql_expression3) "!=" (define b psql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a psql_expression3) "<=" (define b psql_expression2)) '((quote <=) a b))
		(parser '((define a psql_expression3) ">=" (define b psql_expression2)) '((quote >=) a b))
		(parser '((define a psql_expression3) "<" (define b psql_expression2)) '((quote <) a b))
		(parser '((define a psql_expression3) ">" (define b psql_expression2)) '((quote >) a b))
		/* ILIKE is Postgres case-insensitive LIKE. */
		(parser '((define a psql_expression3) (atom "ILIKE" true) (define b psql_expression2)) '('strlike a b "utf8mb4_general_ci"))
		(parser '((define a psql_expression3) (atom "COLLATE" true) (define collation psql_identifier) (atom "LIKE" true) (define b psql_expression2)) '('strlike_cs a b collation))
		/* Postgres LIKE is case-sensitive by default. */
		(parser '((define a psql_expression3) (atom "LIKE" true) (define b psql_expression2)) '('strlike_cs a b))
		/* REGEXP/RLIKE/~ operator: expr ~ 'pattern' -> regexp_test(expr, pattern) */
		(parser '((define a psql_expression3) "~" (define b psql_expression2)) '('regexp_test a b))
		(parser '((define a psql_expression3) (atom "REGEXP" true) (define b psql_expression2)) '('regexp_test a b))
		(parser '((define a psql_expression3) (atom "RLIKE" true) (define b psql_expression2)) '('regexp_test a b))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "REGEXP" true) (define b psql_expression2)) '('not '('regexp_test a b)))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "RLIKE" true) (define b psql_expression2)) '('not '('regexp_test a b)))
		(parser '((define a psql_expression3) (atom "IN" true) "(" (define b (+ psql_expression ",")) ")") '('contains? (cons list b) a))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define b (+ psql_expression ",")) ")") '('not (contains? (cons list b) a)))
		/* BETWEEN operator: expr BETWEEN low AND high -> a >= low AND a <= high */
		(parser '((define a psql_expression3) (atom "BETWEEN" true) (define low psql_expression3) (atom "AND" true) (define high psql_expression3)) (list (quote and) (list (quote >=) a low) (list (quote <=) a high)))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "BETWEEN" true) (define low psql_expression3) (atom "AND" true) (define high psql_expression3)) (list (quote not) (list (quote and) (list (quote >=) a low) (list (quote <=) a high))))
		psql_expression3
	)))

	(define psql_expression3 (parser (or
		/* date + INTERVAL n UNIT */
		(parser '((define a psql_expression4) "+" (atom "INTERVAL" true) (define n psql_expression4) (define unit psql_identifier_unquoted)) '('date_add a n unit))
		/* date - INTERVAL n UNIT */
		(parser '((define a psql_expression4) "-" (atom "INTERVAL" true) (define n psql_expression4) (define unit psql_identifier_unquoted)) '('date_sub a n unit))
		(parser '((define a psql_expression4) "+" (define b psql_expression3)) '((quote +) a b))
		(parser '((define a psql_expression4) "-" (define b psql_expression3)) '((quote -) a b))
		psql_expression4
	)))

	(define psql_expression4 (parser (or
		(parser '((define a psql_expression5) "*" (define b psql_expression4)) '((quote *) a b))
		(parser '((define a psql_expression5) "/" (define b psql_expression4)) '((quote /) a b))
		(parser '((define a psql_expression5) "%" (define b psql_expression4)) (psql_mod_expr a b))
		psql_expression5
	)))

	(define psql_expression5 (parser (or
		(parser '((atom "NOT" true) (define expr psql_expression6)) '('not expr))
		/* unary minus: -(expr) */
		(parser '("-" (define expr psql_expression6)) '((quote -) 0 expr))
		(parser '((define expr psql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr psql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		psql_expression6
	)))

	(define psql_expression7 (parser (or
		/* Scalar subselect in expressions: (SELECT ...) */
		(parser '("(" (define sub psql_select) ")") '('inner_select sub))
		(parser '("(" (define a psql_expression) ")") a)

		/* EXISTS (SELECT ...) */
		(parser '((atom "EXISTS" true) "(" (define sub psql_select) ")") '('inner_select_exists sub))
		(parser '((atom "CASE" true) (define conditions (+ (parser '((atom "WHEN" true) (define a psql_expression) (atom "THEN" true) (define b psql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch psql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" "*" ")") '((quote aggregate) 1 (quote +) 0))
		/* COUNT(DISTINCT expr): unique counting */
		(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define e psql_expression) ")") '('count_distinct e))
		/* COUNT(expr): count non-NULL values -> map to (if (nil? expr) 0 1), reduce +, neutral 0 */
		(parser '((atom "COUNT" true) "(" (define e psql_expression) ")") '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0))
		(parser '((atom "SUM" true) "(" (define s psql_expression) ")") '('aggregate s (quote +) 0))
		(parser '((atom "AVG" true) "(" (define s psql_expression) ")") '((quote /) '('aggregate s (quote +) 0) '('aggregate 1 (quote +) 0)))
		(parser '((atom "MIN" true) "(" (define s psql_expression) ")") '('aggregate s 'min nil))
		(parser '((atom "MAX" true) "(" (define s psql_expression) ")") '('aggregate s 'max nil))
		(parser '((atom "GROUP_CONCAT" true) "(" (define s psql_expression) (atom "SEPARATOR" true) (define sep psql_expression) ")") '('aggregate '('concat s) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil))
		(parser '((atom "GROUP_CONCAT" true) "(" (define s psql_expression) ")") '('aggregate '('concat s) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a "," 'b))) nil))

		(parser '((atom "DATABASE" true) "(" ")") schema)
		(parser '((atom "UNIX_TIMESTAMP" true) "(" ")") '('unix_timestamp))
		(parser '((atom "UNIX_TIMESTAMP" true) "(" (define p psql_expression) ")") '('unix_timestamp p))
		/* CURRENT_DATE / CURRENT_DATE() */
		(parser '((atom "CURRENT_DATE" true) "(" ")") '('current_date))
		(parser (atom "CURRENT_DATE" true) '('current_date))
		/* CURRENT_TIMESTAMP / CURRENT_TIMESTAMP() */
		(parser '((atom "CURRENT_TIMESTAMP" true) "(" ")") '('now))
		(parser (atom "CURRENT_TIMESTAMP" true) '('now))
		/* EXTRACT(field FROM expr) */
		(parser '((atom "EXTRACT" true) "(" (define field psql_identifier_unquoted) (atom "FROM" true) (define e psql_expression) ")") '('extract_date e field))
		/* TIMESTAMPDIFF(unit, dt1, dt2) — unit is a keyword, not a column */
		(parser '((atom "TIMESTAMPDIFF" true) "(" (define unit psql_identifier_unquoted) "," (define dt1 psql_expression) "," (define dt2 psql_expression) ")") '('timestampdiff unit dt1 dt2))
		/* DATE_ADD(expr, INTERVAL n UNIT) / DATE_SUB(expr, INTERVAL n UNIT) */
		(parser '((atom "DATE_ADD" true) "(" (define e psql_expression) "," (atom "INTERVAL" true) (define n psql_expression) (define unit psql_identifier_unquoted) ")") '('date_add e n unit))
		(parser '((atom "DATE_SUB" true) "(" (define e psql_expression) "," (atom "INTERVAL" true) (define n psql_expression) (define unit psql_identifier_unquoted) ")") '('date_sub e n unit))
		/* DATE('str') - parse date string; DATE(expr) - truncate to day */
		(parser '((atom "DATE" true) (define s psql_string)) '('parse_date s))
		(parser '((atom "DATE" true) "(" (define e psql_expression) ")") '('date_trunc_day e))
		/* SUBSTRING(expr FROM start FOR len) - SQL standard syntax */
		(parser '((atom "SUBSTRING" true) "(" (define s psql_expression) (atom "FROM" true) (define start psql_expression) (atom "FOR" true) (define len psql_expression) ")") '((quote sql_substr) s start len))
		/* IFNULL(val, default) - alias for COALESCE with 2 args */
		(parser '((atom "IFNULL" true) "(" (define a psql_expression) "," (define b psql_expression) ")") '((quote coalesceNil) a b))
		/* CONVERT(expr, type) */
		(parser '((atom "CONVERT" true) "(" (define p psql_expression) "," (atom "DECIMAL" true) (? "(" psql_int (? "," psql_int) ")") ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p psql_expression) "," (atom "UNSIGNED" true) ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p psql_expression) "," (atom "SIGNED" true) ")") '('simplify p))
		(parser '((atom "CONVERT" true) "(" (define p psql_expression) "," (atom "INTEGER" true) ")") '('simplify p))
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "UNSIGNED" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "INTEGER" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "VARCHAR" true) "(" psql_int ")" ")") '('concat p))
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "VARCHAR" true) ")") '('concat p))
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "CHAR" true) (atom "CHARACTER" true) (atom "SET" true) (atom "utf8" true) ")") '('concat p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CONCAT" true) "(" (define p (+ psql_expression ",")) ")") (cons 'sql_concat p))
		/* TRIM/LTRIM/RTRIM as explicit parser rules for reliable dispatch */
		(parser '((atom "TRIM" true) "(" (define e psql_expression) ")") '((quote sql_trim) e))
		(parser '((atom "LTRIM" true) "(" (define e psql_expression) ")") '((quote sql_ltrim) e))
		(parser '((atom "RTRIM" true) "(" (define e psql_expression) ")") '((quote sql_rtrim) e))

		(parser '((atom "COALESCE" true) "(" (define args (* psql_expression ",")) ")") (cons (quote coalesceNil) args))
		(parser '((atom "MOD" true) "(" (define a psql_expression) "," (define b psql_expression) ")") (psql_mod_expr a b))
		/* MySQL LAST_INSERT_ID(): direct session lookup to support session scoping */
		(parser '((atom "LAST_INSERT_ID" true) "(" ")") '('session "last_insert_id"))
		/* MySQL IF(condition, true_expr, false_expr) with short-circuit semantics */
		(parser '((atom "IF" true) "(" (define cond psql_expression) "," (define t psql_expression) "," (define f psql_expression) ")") '((quote if) cond t f))
		(parser '((atom "VALUES" true) "(" (define e psql_identifier_unquoted) ")") '('get_column "VALUES" true e true)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "VALUES" true) "(" (define e psql_identifier_quoted) ")") '('get_column "VALUES" true e false)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "pg_catalog" true) "." (atom "set_config" true) "(" psql_expression "," psql_expression "," psql_expression ")") nil) /* ignore */
		(parser '((atom "pg_catalog" true) "." (atom "setval" true) "(" "'" psql_identifier "." (define key psql_identifier) "'" "," (define val psql_expression) "," psql_expression ")") (match key (regex "(.*)_(.*?)_seq" _ tbl col) '('altercolumn '('table schema tbl) col "auto_increment" val) (error "unknown pg_catalog key: " key)))

		(parser (atom "NULL" true) 'nil)
		(parser (atom "TRUE" true) true)
		(parser (atom "FALSE" true) false)
		(parser (atom "ON" true) true)
		(parser (atom "OFF" true) false)
		(parser '((atom "@" true) (define var psql_identifier_unquoted)) '('session var))
		/* MySQL system variables: @@var, @@GLOBAL.var, @@SESSION.var
		@@GLOBAL.var reads globalvars directly; @@SESSION.var / @@var check session first */
		(parser '((atom "@@" true) (atom "GLOBAL" true) (atom "." true) (define var psql_identifier_unquoted)) '('globalvars var))
		(parser '((atom "@@" true) (? (atom "SESSION" true) (? (atom "." true))) (define var psql_identifier_unquoted)) '('session_globalvar var))
		(parser '((atom "@@" true) (define var psql_identifier_unquoted)) '('session_globalvar var))
		/* LEFT(str, n) -- special case because LEFT is a reserved keyword (LEFT JOIN) */
		(parser '((atom "LEFT" true) "(" (define s psql_expression) "," (define n psql_expression) ")") '((quote sql_substr) s 1 n))
		/* RIGHT(str, n) -- special case because RIGHT is a reserved keyword */
		(parser '((atom "RIGHT" true) "(" (define s psql_expression) "," (define n psql_expression) ")") '((quote if) '((quote nil?) s) nil '((quote sql_substr) s '((quote +) 1 '((quote -) '((quote strlen) s) n)) n)))
		(parser '((define fn sql_identifier_unquoted) "(" (define args (* psql_expression ",")) ")") (cons (coalesce (sql_builtins (toUpper fn)) (error "unknown function " fn)) args))
		/* PostgreSQL positional $N placeholder: compiles to (session "vN"), 1-indexed */
		(parser '((atom "$" false) (define n (regex "[0-9]+"))) (list (quote session) (concat "v" n)))
		psql_number
		psql_string
		psql_column
	)))

	/* Postgres cast syntax: expr::type (postfix operator; avoid left recursion) */
	(define psql_expression6 (parser (or
		(parser '((define a psql_expression7) "::" (atom "text" true)) '('concat a))
		(parser '((define a psql_expression7) "::" (atom "varchar" true)) '('concat a))
		(parser '((define a psql_expression7) "::" (atom "integer" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "int" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "bigint" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "float" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "double" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "numeric" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "boolean" true)) '('simplify a))
		(parser '((define a psql_expression7) "::" (atom "date" true)) '('concat a))
		(parser '((define a psql_expression7) "::" (atom "timestamp" true)) '('concat a))
		(parser '((define a psql_expression7) "::" (atom "timestamptz" true)) '('parse_date a))
		(parser '((define a psql_expression7) "::" psql_identifier) a) /* unknown cast types: pass through */
		/* PostgreSQL AT TIME ZONE postfix operator */
		(parser '((define a psql_expression7) (atom "AT" true) (atom "TIME" true) (atom "ZONE" true) (define tz psql_expression7)) '('at_time_zone a tz))
		psql_expression7
	)))

	(define tabledefs (parser (or
		/* TODO: left [outer] join, right [outer] join recursive buildup */
		(parser '((define l tabledefs) (define x (or
			(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e psql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl true e))))
			(parser '((atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e psql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl false e))))
			(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tabledefs)) r)
		))) (merge l x))
		(parser '((define l tabledef) (atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledefs) (atom "ON" true) (define e psql_expression)) (match l '(id schema tbl _ nil) (cons '(id schema tbl true e) r)))
		(parser (define t tabledef) '(t))
	)))
	(define tabledef (parser (or
		(parser '((atom "(" true) (define query psql_select) (atom ")" true) (atom "AS" true) (define id psql_identifier)) '(id schema query false nil)) /* inner select as from */
		(parser '((atom "(" true) (define query psql_select) (atom ")" true) (define id psql_identifier)) '(id schema query false nil)) /* inner select as from */
		/* TODO: case insensititive table search */
		(parser '((define schema psql_identifier) (atom "." true) (define tbl psql_identifier) (atom "AS" true) (define id psql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema psql_identifier) (atom "." true) (define tbl psql_identifier) (define id psql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema psql_identifier) (atom "." true) (define tbl psql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
		(parser '((define tbl psql_identifier) (atom "AS" true) (define id psql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl psql_identifier) (define id psql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl psql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
	)))

	/* bring those variables into a defined state */
	(define from nil)
	(define condition nil)
	(define group nil)
	(define having nil)
	(define order nil)
	(define limit nil)
	(define offset nil)
	(define psql_select_order (lambda (query) (match query
		'(schema tables fields condition group having order limit offset) order
		_ nil
	)))
	(define psql_select_limit (lambda (query) (match query
		'(schema tables fields condition group having order limit offset) limit
		_ nil
	)))
	(define psql_select_offset (lambda (query) (match query
		'(schema tables fields condition group having order limit offset) offset
		_ nil
	)))
	(define psql_select_clear_stage (lambda (query) (match query
		'(schema tables fields condition group having order limit offset) (list schema tables fields condition group having nil nil nil)
		_ query
	)))
	(define psql_union_all_parts (lambda (query) (match query
		'(union_all branches order limit offset) (list branches order limit offset)
		'((symbol union_all) branches order limit offset) (list branches order limit offset)
		'((quote union_all) branches order limit offset) (list branches order limit offset)
		_ nil
	)))
	(define psql_union_all_query (lambda (left right) (begin
		(define right_parts (psql_union_all_parts right))
		(if (nil? right_parts)
			(list (quote union_all)
				(list left (psql_select_clear_stage right))
				(psql_select_order right)
				(psql_select_limit right)
				(psql_select_offset right))
			(match right_parts '(branches order limit offset)
				(list (quote union_all) (cons left branches) order limit offset)))
	)))
	(define psql_select_core (parser '(
		(atom "SELECT" true)
		(? (atom "DISTINCT" true))
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil false "*" false)))
			(parser '((define tbl psql_identifier_quoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define tbl psql_identifier_unquoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define e psql_expression) (atom "AS" true) (define title psql_identifier)) '(title e))
			(parser '((define e psql_expression) (atom "AS" true) (define title psql_string)) '(title e))
			(parser (define e psql_expression) '((extract_title e) e))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
		)
		(define condition (or (parser '(
			(atom "WHERE" true)
			(define condition2 psql_expression)
		) condition2) (empty true)))
		/* GROUP BY + HAVING */
		(?
			(atom "GROUP" true)
			(atom "BY" true)
			(define group (+
				psql_expression
				(atom "," true)
			))
		)
		(?
			(atom "HAVING" true)
			(define having psql_expression)
		)
		/* ORDER BY + LIMIT */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col psql_expression)
					(? (atom "COLLATE" true) (define coll psql_identifier))
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
				'((define offset psql_expression) (atom "," true) (define limit psql_expression))
				'((define limit psql_expression) (atom "OFFSET" true) (define offset psql_expression))
				'((define limit psql_expression))
			)
		)
	) '(schema (if (nil? from) '() (merge from)) (merge cols) condition group having order limit offset)))
	(define psql_select (parser (or
		(parser '(
			(define left psql_select_core)
			(atom "UNION" true)
			(atom "ALL" true)
			(define right psql_select)
		) (psql_union_all_query left right))
		psql_select_core
	)))

	(define psql_update (parser '(
		(atom "UPDATE" true)
		/* TODO: UPDATE tbl FROM tbl, tbl, tbl */
		(define tables (+ psql_identifier ","))
		(atom "SET" true)
		(define cols (+ (or
			/* TODO: tbl.identifier */
			(parser '((define title psql_identifier) "=" (define e psql_expression)) '(title e))
		) ","))
		(? '(
			(atom "WHERE" true)
			(define condition psql_expression)
		))
	) (begin
			(define tbl (car tables))
			(if policy (policy schema tbl true) true)
			(define all_defs (map tables (lambda (t) (list t schema t false nil))))
			(build_dml_plan schema tbl nil all_defs (merge cols) (coalesceNil condition true) nil nil nil)
	)))

	(define psql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)
		/* schema-qualified */
		(? (define schema2 psql_identifier) ".") (define tbl psql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition psql_expression)
		))
		/* ORDER BY + LIMIT + OFFSET for partial deletes */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col psql_expression)
					(? (atom "COLLATE" true) (define coll psql_identifier))
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
				'((define limit psql_expression) (atom "OFFSET" true) (define offset psql_expression))
				'((define limit psql_expression))
			)
		)
		(? (atom "OFFSET" true) (define offset psql_expression))
	) (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define del_schema (coalesce schema2 schema))
			(build_dml_plan del_schema tbl nil (list (list tbl del_schema tbl false nil)) nil (coalesceNil condition true) order limit offset)
	)))

	/* TRUNCATE [TABLE] tbl — alias for DELETE FROM tbl without WHERE */
	(define psql_truncate (parser '(
		(atom "TRUNCATE" true) (? (atom "TABLE" true))
		/* schema-qualified */
		(? (define schema2 psql_identifier) ".") (define tbl psql_identifier)
	) (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define trunc_schema (coalesce schema2 schema))
			(build_dml_plan trunc_schema tbl nil (list (list tbl trunc_schema tbl false nil)) nil true nil nil nil)
	)))

	(define psql_insert_into (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
		(? (define schema2 psql_identifier) ".")
		(define tbl psql_identifier) /* TODO: ignorecase */
		(? "("
			(define coldesc (*
				psql_identifier
				","))
			")")
		(atom "VALUES" true)
		(define datasets (* (parser '(
			"("
			(define dataset (* psql_expression ","))
			")"
		) dataset) ","))
		(define updaterows (? (parser '(
			(atom "ON" true)
			(atom "DUPLICATE" true)
			(atom "KEY" true)
			(atom "UPDATE" true)
			/* TODO: ignorecase */
			(define updaterows (+ (parser '((define col psql_identifier) (atom "=" false) (define value psql_expression)) '(col value)) ","))
		) updaterows)))
		(define onconflict (? (parser '(
			(atom "ON" true)
			(atom "CONFLICT" true)
			"(" (define conflictcols (+ psql_identifier ",")) ")"
			(atom "DO" true)
			(define oc (or
				(parser '((atom "NOTHING" true)) '(do_nothing))
				(parser '((atom "UPDATE" true) (atom "SET" true)
					(define conflictupdates (+ (parser '((define col psql_identifier) (atom "=" false) (define value psql_expression)) '(col value)) ",")))
					'(do_update conflictupdates))
			))
		) oc)))
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(set do_nothing (match onconflict '(do_nothing) true _ false))
			(set conflictupdates (match onconflict '(do_update u) u _ nil))
			(set updaterows3 (coalesce conflictupdates updaterows))
			(set updaterows2 (if (nil? updaterows3) nil (merge updaterows3)))
			(set updatecols (if (nil? updaterows3) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
			(define coldesc (coalesce coldesc (map (show (coalesce schema2 schema) tbl) (lambda (col) (col "Field")))))
			'('insert '('table (coalesce schema2 schema) tbl) (cons list coldesc) (cons list (map datasets (lambda (dataset) (cons list dataset)))) (cons list updatecols)
				(if (or do_nothing (and ignoreexists (nil? updaterows3)))
					'((quote lambda) '() 0)
					(if ignoreexists '('lambda '() true) (if (nil? updaterows3) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))
				false '('lambda '('id) '('session "last_insert_id" 'id)))
	)))

	(define psql_insert_select (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
		(? (define schema2 psql_identifier) ".")
		(define tbl psql_identifier) /* TODO: ignorecase */
		(? "("
			(define coldesc (*
				psql_identifier
				","))
			")")
		(define inner psql_select) /* INNER SELECT */
		(define datasets (* (parser '(
			"("
			(define dataset (* psql_expression ","))
			")"
		) dataset) ","))
		(define updaterows (? (parser '(
			(atom "ON" true)
			(atom "DUPLICATE" true)
			(atom "KEY" true)
			(atom "UPDATE" true)
			/* TODO: ignorecase */
			(define updaterows (+ (parser '((define col psql_identifier) (atom "=" false) (define value psql_expression)) '(col value)) ","))
		) updaterows)))
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
			(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
			(define coldesc (coalesce coldesc (map (show (coalesce schema2 schema) tbl) (lambda (col) (col "Field")))))
			'('begin
				'('set 'resultrow '('lambda '('item) '('insert '('table (coalesce schema2 schema) tbl) (cons list coldesc) (cons list '((cons list (map (produceN (count coldesc)) (lambda (i) '('nth 'item (+ (* i 2) 1))))))) (cons list updatecols) (if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))) false '('lambda '('id) '('session "last_insert_id" 'id)))))
				(build_queryplan_term inner)
			)
	)))

	(define psql_foreign_key_mode (parser (or
		(parser (atom "RESTRICT" true) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser (atom "SET NULL" true) "set null")
	)))

	(define psql_create_table (parser '(
		(atom "CREATE" true)
		(atom "TABLE" true)
		(define ifnotexists (parser (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)) true))
		(define id (or (parser '((define schema2 psql_identifier) "." (define id psql_identifier)) id) psql_identifier))
		"("
		(define cols (* (or
			(parser '((atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ psql_identifier ",")) ")") '((quote list) "unique" "PRIMARY" (cons (quote list) cols)))
			(parser '((atom "UNIQUE" true) (atom "KEY" true) (define id psql_identifier) "(" (define cols (+ psql_identifier ",")) ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "CONSTRAINT" true) (define id (? psql_identifier)) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ psql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 psql_identifier) "(" (define cols2 (+ psql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (define deletemode psql_foreign_key_mode)) (? (atom "ON" true) (atom "UPDATE" true) (define updatemode psql_foreign_key_mode))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2) updatemode deletemode))
			(parser '((atom "FOREIGN" true) (atom "KEY" true) (define id (? psql_identifier)) "(" (define cols1 (+ psql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 psql_identifier) "(" (define cols2 (+ psql_identifier ",")) ")" (? (atom "ON" true) (atom "UPDATE" true) (define updatemode psql_foreign_key_mode)) (? (atom "ON" true) (atom "DELETE" true) (define deletemode psql_foreign_key_mode))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2)))
			(parser '((atom "KEY" true) psql_identifier "(" (+ psql_identifier ",") ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list))) /* ignore index definitions */
			(parser '(
				(define col psql_identifier)
				(define type psql_type)
				(define dimensions (or
					(parser '("(" (define a psql_int) "," (define b psql_int) ")") '((quote list) a b))
					(parser '("(" (define a psql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				/* column flags */
				(define typeparams (parser (define sub (* (or
					(parser '((atom "PRIMARY" true) (atom "KEY" true)) '("primary" true))
					(parser (atom "PRIMARY" true) '("primary" true))
					(parser '((atom "UNIQUE" true) (atom "KEY" true)) '("unique" true))
					(parser (atom "UNIQUE" true) '("unique" true))
					(parser (atom "AUTO_INCREMENT" true) '("auto_increment" true))
					(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
					(parser (atom "NULL" true) '("null" true))
					(parser '((atom "DEFAULT" true) (define default psql_expression)) '("default" default))
					(parser '((atom "COMMENT" true) (define comment psql_expression)) '("comment" comment))
					(parser '((atom "COLLATE" true) (define comment psql_identifier)) '("collate" comment))
					/* TODO: GENERATED ALWAYS AS expr */
				))) (merge sub)))
			) '((quote list) "column" col type dimensions (cons 'list typeparams)))
		) ","))
		")"
		(define options (* (or
			(parser '((atom "CHARACTER" true) (atom "SET" true) (define id psql_identifier)) '("charset" id))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) '("engine" "memory"))
			(parser '((atom "ENGINE" true) "=" (atom "CACHE" true)) '("engine" "cache"))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) '("engine" "sloppy"))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) '("engine" "logged"))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) '("engine" "safe"))
			(parser '((atom "DEFAULT" true) (atom "CHARSET" false) "=" (define id psql_identifier)) '("charset" id))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "COLLATE" true) (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define collation (regex "[0-9]+"))) '("auto_increment" collation))
		)))
	) '((quote createtable) (coalesce schema2 schema) id (cons (quote list) cols) (cons (quote list) (merge options)) ifnotexists)))

	(define psql_alter_table (parser '(
		(atom "ALTER" true)
		(atom "TABLE" true)
		(? (atom "ONLY" true))
		(define id (or (parser '(psql_identifier "." (define id psql_identifier)) id) psql_identifier))
		(define alters (+ (or
			/* TODO */
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define id psql_identifier) (atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ psql_identifier ",")) ")") (lambda (tbl) '('createkey '('table schema tbl) id true (cons (quote list) cols))))
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define id psql_identifier) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ psql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 (or (parser '(psql_identifier "." (define id psql_identifier)) id) psql_identifier)) "(" (define cols2 (+ psql_identifier ",")) ")" (? (atom "ON" true) (atom "UPDATE" true) (define updatemode psql_foreign_key_mode)) (? (atom "ON" true) (atom "DELETE" true) (define deletemode psql_foreign_key_mode))) (lambda (tbl) '('createforeignkey '('table schema tbl) id (cons (quote list) cols1) '('table schema tbl2) (cons (quote list) cols2) updatemode deletemode)))
			/*
			(parser '((atom "ADD" true) (atom "UNIQUE" true) (atom "KEY" true) (define id psql_identifier) "(" (define cols (+ psql_identifier ",")) ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "ADD" true) (atom "KEY" true) psql_identifier "(" (+ psql_identifier ",") ")" (? (atom "USING" true) (atom "BTREE" true))) nil) /* ignore index definitions */
			(parser '((atom "ADD" true) (?(atom "COLUMN" true))
				(define col psql_identifier)
				(define type psql_identifier)
				(define dimensions (or
					(parser '("(" (define a psql_int) "," (define b psql_int) ")") '((quote list) a b))
					(parser '("(" (define a psql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				(define typeparams psql_column_attributes)
			) (lambda (id) '((quote createcolumn) '('table schema id) col type dimensions (cons 'list typeparams))))
			(parser '((atom "OWNER" true) (atom "TO" true) (define owner psql_identifier)) (lambda (id) '((quote altertable) '('table schema id) "owner" owner)))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true) (define cname psql_identifier)) (lambda (id) true))
			(parser '((atom "DROP" true) (? (atom "COLUMN" true)) (define col psql_identifier)) (lambda (id) '((quote altertable) '('table schema id) "drop" col)))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "memory")))
			(parser '((atom "ENGINE" true) "=" (atom "CACHE" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "cache")))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "sloppy")))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "logged")))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) (lambda (id) '((quote altertable) '('table schema id) "engine" "safe")))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) (lambda (id) '((quote altertable) '('table schema id) "collation" collation)))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define ai (regex "[0-9]+"))) (lambda (id) '((quote altertable) '('table schema id) "auto_increment" ai)))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col psql_identifier) (define body (or /* ALTER COLUMN */
				(parser '((atom "ADD" true) (atom "GENERATED" true) (or '((atom "BY" true) (atom "DEFAULT" true)) (atom "ALWAYS" true)) (atom "AS" true) (atom "IDENTITY") "("
					(atom "SEQUENCE" true) (atom "NAME" true) psql_identifier "." psql_identifier
					(? (atom "START" true) (atom "WITH" true) psql_expression)
					(? (atom "INCREMENT" true) (atom "BY" true) psql_expression)
					(? (atom "NO" true) (atom "MINVALUE" true))
					(? (atom "NO" true) (atom "MAXVALUE" true))
					(? (atom "CACHE" true) psql_expression)
					")") (lambda (col) (lambda (id) '((quote altercolumn) '('table schema id) col "auto_increment" true))))
				/* Type and default changes */
				(parser '((atom "TYPE" true) (define type psql_identifier) (define dimensions (or (parser '("(" (define a psql_int) "," (define b psql_int) ")") '((quote list) a b)) (parser '("(" (define a psql_int) ")") '((quote list) a)) (parser empty '((quote list))))) ) (lambda (col) (lambda (id) '('!begin '((quote altercolumn) '('table schema id) col "type" type) '((quote altercolumn) '('table schema id) col "dimensions" dimensions)))))
				(parser '((atom "SET" true) (atom "DEFAULT" true) (define def psql_expression)) (lambda (col) (lambda (id) '((quote altercolumn) '('table schema id) col "default" def))))
				(parser '((atom "DROP" true) (atom "DEFAULT" true)) (lambda (col) (lambda (id) '((quote altercolumn) '('table schema id) col "default" nil))))
				(parser '((atom "COLLATE" true) (define coll psql_identifier)) (lambda (col) (lambda (id) '((quote altercolumn) '('table schema id) col "collation" coll))))
			))) (body col))
		) ","))
	) (cons '!begin (map alters (lambda (alter) (alter id))))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	(define p (parser (or
		(parser (atom "SHUTDOWN" true) (begin (if policy (policy "system" true true) true) '(shutdown)))
		(parser (define query psql_select) (build_queryplan_term query))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query psql_select)) (explain_queryplan_ir query))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query psql_select)) (explain_queryplan_reorder query))
		(parser '((atom "EXPLAIN" true) (define query psql_select)) '('resultrow '('list "code" (pretty_print (build_queryplan_term query) (settings "ExplainWidth")))))
		psql_insert_into
		psql_insert_select
		psql_create_table
		psql_alter_table
		psql_update
		psql_delete
		psql_truncate

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define ifnot (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true))) (define id psql_identifier)) (begin (if policy (policy "system" true true) true) '((quote createdatabase) id (if ifnot true false))) )
		/* CREATE USER/ROLE: support both MySQL (IDENTIFIED BY) and PostgreSQL (WITH PASSWORD / PASSWORD) syntax */
		(parser '((atom "CREATE" true) (or (atom "USER" true) (atom "ROLE" true)) (define username psql_identifier)
			(? (or
				'((atom "IDENTIFIED" true) (atom "BY" true) (define password psql_expression))
				'((? (atom "WITH" true)) (? (or (atom "SUPERUSER" true) (atom "LOGIN" true))) (atom "PASSWORD" true) (define password psql_expression))
		)))
			(begin (if policy (policy "system" true true) true)
				'('insert '('table "system" "user") '(list "username" "password" "admin") '(list '(list username '('password password) false)) '(list) '((quote lambda) '() '((quote error) "user already exists")))
		))
		/* ALTER USER password: MySQL (IDENTIFIED BY) and PostgreSQL (WITH PASSWORD / PASSWORD) */
		(parser '((atom "ALTER" true) (atom "USER" true) (define username psql_identifier)
			(? (atom "WITH" true))
			(atom "PASSWORD" true) (define password psql_expression))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "password" '('password password)))))
		))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username psql_identifier)
			(atom "IDENTIFIED" true) (atom "BY" true) (define password psql_expression))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "password" '('password password)))))
		))
		/* ALTER USER SUPERUSER / NOSUPERUSER — PostgreSQL admin grant */
		(parser '((atom "ALTER" true) (atom "USER" true) (define username psql_identifier) (atom "SUPERUSER" true))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" true))))
		))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username psql_identifier) (atom "NOSUPERUSER" true))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" false))))
		))
		/* DROP USER/ROLE [IF EXISTS] — cascade-deletes access entries then the user row */
		(parser '((atom "DROP" true) (or (atom "USER" true) (atom "ROLE" true)) (? (atom "IF" true) (atom "EXISTS" true)) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				(cons '!begin (list
					'((quote scan) '(session "__memcp_tx") '('table "system" "access")
						'('list "username")
						'((quote lambda) '('username) '((quote equal??) (quote username) username))
						'(list "$update")
						'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
						(quote +)
						0)
					'((quote scan) '(session "__memcp_tx") '('table "system" "user")
						'('list "username")
						'((quote lambda) '('username) '((quote equal??) (quote username) username))
						'(list "$update")
						'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
						(quote +)
						0)
				))
		))

		/* GRANT syntax (PostgreSQL-style) -> reflect only admin and database-level access */
		/* GRANT ALL [PRIVILEGES] ON *.* TO user -> set admin true */
		(parser '((atom "GRANT" true) (atom "ALL" true) (? (atom "PRIVILEGES" true)) (atom "ON" true) (atom "*" true) (atom "." true) (atom "*" true) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" true))))
		))
		/* REVOKE ALL [PRIVILEGES] ON *.* FROM user -> set admin false */
		(parser '((atom "REVOKE" true) (atom "ALL" true) (? (atom "PRIVILEGES" true)) (atom "ON" true) (atom "*" true) (atom "." true) (atom "*" true) (atom "FROM" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) '(session "__memcp_tx") '('table "system" "user") '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "admin" false))))
		))
		/* GRANT <any> ON DATABASE db TO user (idempotent) */
		(parser '((atom "GRANT" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "DATABASE" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert '('table "system" "access") '('list "username" "database") '('list '('list username db)) '(list) '((quote lambda) '() false))
		))
		/* GRANT <any> ON SCHEMA db TO user (idempotent) */
		(parser '((atom "GRANT" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert '('table "system" "access") '('list "username" "database") '('list '('list username db)) '(list) '((quote lambda) '() false))
		))
		/* GRANT ALL PRIVILEGES ON ALL DATABASES is non-standard; ignore */
		/* Treat GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA db TO user as db-level access (idempotent) */
		(parser '((atom "GRANT" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "ALL" true) (atom "TABLES" true) (atom "IN" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert '('table "system" "access") '('list "username" "database") '('list '('list username db)) '(list) '((quote lambda) '() false))
		))

		/* REVOKE syntax (PostgreSQL-style) -> mirror GRANT behavior */
		/* REVOKE <any> ON DATABASE db FROM user */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "DATABASE" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					'(session "__memcp_tx")
					'('table "system" "access")
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal??) (quote username) username) '((quote equal??) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))
		/* REVOKE <any> ON SCHEMA db FROM user */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					'(session "__memcp_tx")
					'('table "system" "access")
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal??) (quote username) username) '((quote equal??) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))
		/* REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA db FROM user -> treat as db-level */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier "," (atom "ALL" true) (atom "PRIVILEGES" true) (atom "SELECT" true) (atom "CONNECT" true) (atom "USAGE" true))) (atom "ON" true) (atom "ALL" true) (atom "TABLES" true) (atom "IN" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					'(session "__memcp_tx")
					'('table "system" "access")
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal?) (quote username) username) '((quote equal?) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))

		(parser '((atom "CREATE" true) (define unique (? (atom "UNIQUE" true))) (atom "INDEX" true) (define id psql_identifier) (atom "ON" true) (define tbl (or (parser '(psql_identifier "." (define id psql_identifier)) id) psql_identifier)) (? (atom "USING" true) psql_identifier) "(" (define cols (+ psql_identifier ",")) ")") (if unique '('createkey '('table schema tbl) id unique (cons (quote list) cols)) true))
		(parser '((atom "DROP" true) (atom "INDEX" true) (define id psql_identifier)) true)

		/* SHOW CREATE TABLE [schema.]table */
		(parser '((atom "SHOW" true) (atom "CREATE" true) (atom "TABLE" true) (define schema2 psql_identifier) (atom "." true) (define id psql_identifier))
			'((quote resultrow) '((quote list) "Table" id "Create Table" '((quote format_create_table) schema2 id))))
		(parser '((atom "SHOW" true) (atom "CREATE" true) (atom "TABLE" true) (define id psql_identifier))
			'((quote resultrow) '((quote list) "Table" id "Create Table" '((quote format_create_table) schema id))))
		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true) (? (atom "FROM" true) (define schema psql_identifier))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '((quote resultrow) '((quote list) "Table" (quote tbl))))))
		(parser '((atom "SHOW" true) (atom "TABLE" true) (atom "STATUS" true) (? (atom "FROM" true) (define schema psql_identifier) (? (atom "LIKE" true) (define likepattern psql_expression)))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '('if '('strlike 'tbl '('coalesce 'likepattern "%")) '((quote resultrow) '('list "name" 'tbl "rows" "1")))))) /* TODO: engine version row_format avg_row_length data_length max_data_length index_length data_free auto_increment create_time update_time check_time collation checksum create_options comment max_index_length temporary */
		(parser '((or (atom "DESCRIBE" true) (atom "DESC" true)) (define schema2 psql_identifier) (atom "." true) (define id psql_identifier)) '((quote map) '((quote show) schema2 id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))
		(parser '((or (atom "DESCRIBE" true) (atom "DESC" true)) (define id psql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))
		(parser '((atom "SHOW" true) (atom "FULL" true) (atom "COLUMNS" true) (atom "FROM" true) (define id psql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line))))) /* TODO: Field Type Collation Null Key Default Extra(auto_increment) Privileges Comment */

		/* SHOW TRIGGERS [FROM schema] */
		(parser '((atom "SHOW" true) (atom "TRIGGERS" true) (? (atom "FROM" true) (define tgtschema psql_identifier)))
			'((quote map) '((quote show_triggers) (coalesce tgtschema schema)) '((quote lambda) '((quote tr)) '((quote resultrow) (quote tr)))))

		/* SHOW ENGINES: list engines recognized by CREATE/ALTER TABLE */
		(parser '((atom "SHOW" true) (atom "ENGINES" true)) (cons '!begin '(
			'((quote resultrow) '((quote list) "Engine" "SAFE"    "Support" "DEFAULT" "Comment" "Safe durable engine"              "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "LOGGING" "Support" "YES"     "Comment" "Append-only logging engine"      "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "MEMORY"  "Support" "YES"     "Comment" "In-memory engine"               "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "CACHE"   "Support" "YES"     "Comment" "In-memory evictable engine"        "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "SLOPPY"  "Support" "YES"     "Comment" "Relaxed engine"                 "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "MyISAM"  "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "InnoDB"  "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
			'((quote resultrow) '((quote list) "Engine" "CSV"     "Support" "YES"     "Comment" "Alias of SAFE"                  "Transactions" "NO" "XA" "NO" "Savepoints" "NO"))
		)))
		/* SHOW {CHARSET|CHARACTER SET} [LIKE pattern] */
		(parser '((atom "SHOW" true) (or (atom "CHARSET" true) '((atom "CHARACTER" true) (atom "SET" true))) (? (atom "LIKE" true) (define likepattern psql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Charset" "utf8mb4" "Description" "UTF-8 Unicode" "Default collation" "utf8mb4_general_ci" "Maxlen" 4))
		)))
		/* SHOW COLLATION [LIKE pattern] */
		(parser '((atom "SHOW" true) (atom "COLLATION" true) (? (atom "LIKE" true) (define likepattern psql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Collation" "utf8mb4_general_ci" "Charset" "utf8mb4" "Id" 255 "Default" "YES" "Compiled" "YES" "Sortlen" 1))
			'((quote resultrow) '((quote list) "Collation" "utf8mb4_bin"        "Charset" "utf8mb4" "Id" 254 "Default" "NO"  "Compiled" "YES" "Sortlen" 1))
		)))
		/* SHOW PLUGINS: return empty set (ok for most clients) */
		(parser '((atom "SHOW" true) (atom "PLUGINS" true)) (quote true))

		/* SHOW [FULL] PROCESSLIST */
		(parser '((atom "SHOW" true) (atom "PROCESSLIST" true))
			'((quote map) '((quote show_processlist)) '((quote lambda) '((quote row)) '((quote resultrow) (quote row)))))
		(parser '((atom "SHOW" true) (atom "FULL" true) (atom "PROCESSLIST" true))
			'((quote map) '((quote show_processlist) true) '((quote lambda) '((quote row)) '((quote resultrow) (quote row)))))

		/* KILL [QUERY|CONNECTION] id */
		(parser '((atom "KILL" true) (? (or (atom "QUERY" true) (atom "CONNECTION" true))) (define id psql_expression))
			'((quote kill_query) id))

		/* SHOW [GLOBAL|SESSION] VARIABLES [LIKE pattern] */
		(parser '((atom "SHOW" true) (? (or (atom "GLOBAL" true) (atom "SESSION" true))) (atom "VARIABLES" true) (? (atom "LIKE" true) (define likepattern psql_expression))) (cons '!begin '(
			'((quote resultrow) '((quote list) "Variable_name" "version"               "Value" "0.9"))
			'((quote resultrow) '((quote list) "Variable_name" "character_set_server" "Value" "utf8mb4"))
			'((quote resultrow) '((quote list) "Variable_name" "collation_server"     "Value" "utf8mb4_general_ci"))
			'((quote resultrow) '((quote list) "Variable_name" "lower_case_table_names" "Value" 0))
		)))
		(parser '((atom "SHOW" true) (atom "VARIABLES" true)) '((quote map_assoc) '((quote list) "version" "0.9") '((quote lambda) '((quote key) (quote value)) '((quote resultrow) '((quote list) "Variable_name" (quote key) "Value" (quote value))))))
		(parser '((atom "SET" true) (atom "NAMES" true) (define charset psql_expression)) (quote true)) /* ignore */
		/* PostgreSQL SET TIME ZONE / SET timezone syntax */
		(parser '((atom "SET" true) (atom "TIME" true) (atom "ZONE" true) (atom "LOCAL" true)) '((quote session) "time_zone" "SYSTEM"))
		(parser '((atom "SET" true) (atom "TIME" true) (atom "ZONE" true) (atom "DEFAULT" true)) '((quote session) "time_zone" (globalvars "time_zone")))
		(parser '((atom "SET" true) (atom "TIME" true) (atom "ZONE" true) (define tz psql_expression)) '((quote session) "time_zone" tz))
		/* SET timezone = 'value' — PostgreSQL GUC style */
		(parser '((atom "SET" true) (atom "timezone" true) (or "=" (atom "TO" true)) (define tz psql_expression)) '((quote session) "time_zone" tz))


		(parser '((atom "DROP" true) (atom "DATABASE" true) (define id psql_identifier)) (begin (if policy (policy "system" true true) true) '((quote dropdatabase) id)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define schema psql_identifier) (atom "." true) (define id psql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id psql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "RENAME" true) (atom "TABLE" true) (define oldname psql_identifier) (atom "TO" true) (define newname psql_identifier)) '((quote renametable) schema oldname newname))
		(parser '((atom "SET" true) (? (atom "SESSION" true)) (define vars (* (parser '((? "@") (define key psql_identifier) "=" (define value (or
			(parser (atom "content" true) "content") /* quirks for SET xmloption = content */
			(parser (atom "warning" true) "warning") /* quirks for SET client_min_messages = warning */
			(parser (atom "heap" true) "heap") /* quirks for SET default_table_access_method = heap */
			psql_expression
		))) (list (list (quote context) "session") key value)) ","))) (cons '!begin vars))

		(parser '((atom "LOCK" true) (or (atom "TABLES" true) (atom "TABLE" true))
			(define locks (+ (parser '((define tbl psql_identifier) (? (atom "AS" true) (define alias psql_identifier)) (define mode (or (parser (atom "WRITE" true) true) (parser '((atom "LOW_PRIORITY" true) (atom "WRITE" true)) true) (parser '((atom "READ" true) (? (atom "LOCAL" true))) nil)))) (list tbl (not (nil? mode)))) ",")))
			(list (quote locktables) (cons (quote list) (map locks (lambda (l) (cons (quote list) (list schema (nth l 0) (nth l 1))))))))
		(parser '((atom "UNLOCK" true) (or (atom "TABLES" true) (atom "TABLE" true))) '((quote unlocktables)))

		/* SHOW INDEXES FROM t / SHOW INDEX FROM t / SHOW KEYS FROM t (no-op, returns empty) */
		(parser '((atom "SHOW" true) (or (atom "INDEXES" true) (atom "INDEX" true) (atom "KEYS" true)) (atom "FROM" true) psql_identifier (? (atom "WHERE" true) psql_expression)) "ignore")

		/* ANALYZE TABLE (no-op) */
		(parser '((atom "ANALYZE" true) (atom "TABLE" true) psql_identifier) "ignore")

		/* USE database - change current schema */
		(parser '((atom "USE" true) (define db psql_identifier)) (list (list (quote context) "session") "schema" db))

		/* transaction control */
		(parser '((atom "START" true) (atom "ACID" true) (atom "TRANSACTION" true)) (list (quote tx_begin_acid) (list (quote context) "session")))
		(parser '((atom "START" true) (atom "TRANSACTION" true)) (list (quote tx_begin) (list (quote context) "session")))
		(parser '((atom "BEGIN" true)) (list (quote tx_begin) (list (quote context) "session")))
		(parser '((atom "COMMIT" true)) (list (quote tx_commit) (list (quote context) "session")))
		(parser '((atom "ROLLBACK" true)) (list (quote tx_rollback) (list (quote context) "session")))
		"" /* comment only command */
	)))
	((parser (define command p) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))

(define psql_copy_def (parser '(psql_identifier /* ignore */ "." (define tbl psql_identifier) "(" (define columns (+ psql_identifier ",")) ")") '(tbl columns)))

(define load_psql (lambda (schema stream policy) (begin
	(set state (newsession))
	(set resultrow print)
	(set session (newsession))
	(define psql_line (lambda (line) (begin
		(match line
			(concat "--" b) /* comment */ false
			(concat "COPY " def " FROM stdin;\n") (begin
				/* public.cron (name, lastrun, medianruntime, id) */
				(match (psql_copy_def def) '(tbl columns) (begin
					/* (print "TODO: insert into " tbl columns) */
					/* TODO: escape b 8 f 12 n 10 r 13 t 9 v 11 \324 octal \xFF hex */
					(state "line" (lambda (line) (begin
						(match line
							"\\.\n" /* end of input */ (state "line" psql_line)
							(concat x "\n") (insert (table schema tbl) columns '((split x "\t")))
						)
					)))
				))
			)
			(concat start ";" rest) (begin
				/* command ended -> execute (at max one command per line) */
				(print (concat (state "sql") start))
				(set plan (parse_psql schema (concat (state "sql") start) policy))
				(print "SQL execute" plan)
				(eval plan)
				(state "sql" rest)
			)
			/* otherwise: append to cache */
			(state "sql" (concat (state "sql") line))
		)
	)))
	(state "line" psql_line)
	(state "sql" "")
	(load stream (lambda (line) (begin
		((state "line") line)
	)) "\n")
)))
