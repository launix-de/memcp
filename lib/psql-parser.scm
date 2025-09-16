/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

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
	(atom "INSERT" true)
	(atom "ORDER" true)
	(atom "LIMIT" true)
	(atom "DELIMITER" true)
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

(define psql_type (parser (or
	(parser '((atom "character" true) (atom "varying" true)) "varying")
	(parser '((atom "double" true) (atom "precision" true)) "double")
	psql_identifier
)))

(define parse_psql (lambda (schema s policy) (begin


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
		(parser '((define a psql_expression3) "==" (define b psql_expression2)) '((quote equal??) a b))
		(parser '((define a psql_expression3) "=" (define b psql_expression2)) '((quote equal??) a b))
		(parser '((define a psql_expression3) "<>" (define b psql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a psql_expression3) "!=" (define b psql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a psql_expression3) "<=" (define b psql_expression2)) '((quote <=) a b))
		(parser '((define a psql_expression3) ">=" (define b psql_expression2)) '((quote >=) a b))
		(parser '((define a psql_expression3) "<" (define b psql_expression2)) '((quote <) a b))
		(parser '((define a psql_expression3) ">" (define b psql_expression2)) '((quote >) a b))
		(parser '((define a psql_expression3) (atom "COLLATE" true) (define collation psql_identifier) (atom "LIKE" true) (define b psql_expression2)) '('strlike a b collation))
		(parser '((define a psql_expression3) (atom "LIKE" true) (define b psql_expression2)) '('strlike a b))
		(parser '((define a psql_expression3) (atom "IN" true) "(" (define b (+ psql_expression ",")) ")") '('contains? (cons list b) a))
		(parser '((define a psql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define b (+ psql_expression ",")) ")") '('not '('contains? (cons list b) a)))
		psql_expression3
	)))

	(define psql_expression3 (parser (or
		(parser '((define a psql_expression4) "+" (define b psql_expression3)) '((quote +) a b))
		(parser '((define a psql_expression4) "-" (define b psql_expression3)) '((quote -) a b))
		psql_expression4
	)))

	(define psql_expression4 (parser (or
		(parser '((define a psql_expression5) "*" (define b psql_expression4)) '((quote *) a b))
		(parser '((define a psql_expression5) "/" (define b psql_expression4)) '((quote /) a b))
		psql_expression5
	)))

	(define psql_expression5 (parser (or
		(parser '((atom "NOT" true) (define expr psql_expression6)) '('not expr))
		(parser '((define expr psql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr psql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		psql_expression6
	)))

	(define psql_expression6 (parser (or
		(parser '("(" (define a psql_expression) ")") a)

		(parser '((atom "CASE" true) (define conditions (* (parser '((atom "WHEN" true) (define a psql_expression) (atom "THEN" true) (define b psql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch psql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" "*" ")") '((quote aggregate) 1 (quote +) 0))
		/* COUNT(expr): count non-NULL values -> map to (if (nil? expr) 0 1), reduce +, neutral 0 */
		(parser '((atom "COUNT" true) "(" (define e psql_expression) ")") '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0))
		(parser '((atom "SUM" true) "(" (define s psql_expression) ")") '('aggregate s (quote +) 0))
		(parser '((atom "AVG" true) "(" (define s psql_expression) ")") '((quote /) '('aggregate s (quote +) 0) '('aggregate 1 (quote +) 0)))
		(parser '((atom "MIN" true) "(" (define s psql_expression) ")") '('aggregate s 'min nil))
		(parser '((atom "MAX" true) "(" (define s psql_expression) ")") '('aggregate s 'max nil))

		(parser '((atom "DATABASE" true) "(" ")") schema)
		(parser '((atom "UNIX_TIMESTAMP" true) "(" ")") '('now))
		(parser '((atom "UNIX_TIMESTAMP" true) "(" (define p psql_expression) ")") '('parse_date p))
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "UNSIGNED" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "INTEGER" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p psql_expression) (atom "AS" true) (atom "CHAR" true) (atom "CHARACTER" true) (atom "SET" true) (atom "utf8" true) ")") '('concat p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CONCAT" true) "(" (define p (+ psql_expression ",")) ")") (cons 'concat p)) /* TODO: proper implement CAST; for now make vscode work */
		/* TODO: function call */

		(parser '((atom "COALESCE" true) "(" (define args (* psql_expression ",")) ")") (cons (quote coalesce) args))
		(parser '((atom "VALUES" true) "(" (define e psql_identifier_unquoted) ")") '('get_column "VALUES" true e true)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "VALUES" true) "(" (define e psql_identifier_quoted) ")") '('get_column "VALUES" true e false)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "pg_catalog" true) "." (atom "set_config" true) "(" psql_expression "," psql_expression "," psql_expression ")") nil) /* ignore */
		(parser '((atom "pg_catalog" true) "." (atom "setval" true) "(" "'" psql_identifier "." (define key psql_identifier) "'" "," (define val psql_expression) "," psql_expression ")") (match key (regex "(.*)_(.*?)_seq" _ tbl col) '('altercolumn schema tbl col "auto_increment" val) (error "unknown pg_catalog key: " key)))

		(parser (atom "NULL" true) nil)
		(parser (atom "TRUE" true) true)
		(parser (atom "FALSE" true) false)
		(parser (atom "ON" true) true)
		(parser (atom "OFF" true) false)
		(parser '((atom "@" true) (define var psql_identifier_unquoted)) '('session var))
		(parser '((atom "@@" true) (define var psql_identifier_unquoted)) '('globalvars var))
		(parser '((define fn sql_identifier_unquoted) "(" (define args (* psql_expression ",")) ")") (cons (coalesce (sql_builtins (toUpper fn)) (error "unknown function " fn)) args))
		psql_number
		psql_string
		psql_column
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
	(define psql_select (parser '(
		(atom "SELECT" true)
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
			(?
				(atom "WHERE" true)
				(define condition psql_expression)
			)
		)
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

	(define psql_update (parser '(
		(atom "UPDATE" true)
		/* TODO: UPDATE tbl FROM tbl, tbl, tbl */
		(define tbl psql_identifier) /* TODO: ignorecase */
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
			/* policy: write access check */
			(if policy (policy schema tbl true) true)
			(define replace_find_column (lambda (expr) (match expr
				'((symbol get_column) nil _ col ci) '((quote get_column) tbl false col ci) /* TODO: case insensitive column */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))
			replace_find_column /* workaround for optimizer bug: variable bindings in parsers */
			(set cols (map_assoc (merge cols) (lambda (col expr) (replace_find_column expr))))
			(set condition (replace_find_column (coalesce condition true)))
			(set filtercols (extract_columns_for_tblvar tbl condition))
			(set scancols (merge_unique (extract_assoc cols (lambda (col expr) (extract_columns_for_tblvar tbl expr)))))
			'((quote scan)
				schema
				tbl
				(cons list filtercols)
				'((quote lambda) (map filtercols (lambda(col) (symbol (concat tbl "." col)))) (replace_columns_from_expr condition))
				(cons list (cons "$update" scancols))
				'('lambda
					(cons (quote $update) (map scancols (lambda (col) (symbol (concat tbl "." col)))))
					'((quote if) '((quote $update) (cons (quote list) (map_assoc cols (lambda (col expr) (replace_columns_from_expr expr))))) 1 0)
				)
				(quote +)
				0
			)
	)))

	(define psql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)
		/* schema-qualified */
		(? (define schema2 psql_identifier) ".") (define tbl psql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition psql_expression)
	))) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define replace_find_column (lambda (expr) (match expr
				'((symbol get_column) nil _ col ci) '((quote get_column) tbl false col ci) /* TODO: case insensititive column */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))
			replace_find_column /* workaround for optimizer bug: variable bindings in parsers */
			(set condition (replace_find_column (coalesce condition true)))
			(set filtercols (extract_columns_for_tblvar tbl condition))
			'((quote scan)
				(coalesce schema2 schema)
				tbl
				(cons list filtercols)
				'((quote lambda) (map filtercols (lambda(col) (symbol (concat tbl "." col)))) (replace_columns_from_expr condition))
				'(list "$update")
				'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
				(quote +)
				0
			)
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
	) (begin
			/* policy: write access check */
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
			(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
			(define coldesc (coalesce coldesc (map (show (coalesce schema2 schema) tbl) (lambda (col) (col "Field")))))
			'('insert (coalesce schema2 schema) tbl (cons list coldesc) (cons list (map datasets (lambda (dataset) (cons list dataset)))) (cons list updatecols) (if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))
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
				'('set 'resultrow '('lambda '('item) '('insert (coalesce schema2 schema) tbl (cons list coldesc) (cons list '((cons list (map (produceN (count coldesc)) (lambda (i) '('nth 'item (+ (* i 2) 1))))))) (cons list updatecols) (if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))))
				(apply build_queryplan (apply untangle_query inner))
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
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) '("engine" "sloppy"))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) '("engine" "logging"))
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
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define id psql_identifier) (atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ psql_identifier ",")) ")") (lambda (tbl) '('createkey schema tbl id true (cons (quote list) cols))))
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define id psql_identifier) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ psql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 (or (parser '(psql_identifier "." (define id psql_identifier)) id) psql_identifier)) "(" (define cols2 (+ psql_identifier ",")) ")" (? (atom "ON" true) (atom "UPDATE" true) (define updatemode psql_foreign_key_mode)) (? (atom "ON" true) (atom "DELETE" true) (define deletemode psql_foreign_key_mode))) (lambda (tbl) '('createforeignkey schema id tbl (cons (quote list) cols1) tbl2 (cons (quote list) cols2) updatemode deletemode)))
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
			) (lambda (id) '((quote createcolumn) schema id col type dimensions (cons 'list typeparams))))
			(parser '((atom "OWNER" true) (atom "TO" true) (define owner psql_identifier)) (lambda (id) '((quote altertable) schema id "owner" owner)))
			(parser '((atom "DROP" true) (? (atom "COLUMN" true)) (define col psql_identifier)) (lambda (id) '((quote altertable) schema id "drop" col)))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) (lambda (id) '((quote altertable) schema id "engine" "memory")))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) (lambda (id) '((quote altertable) schema id "engine" "sloppy")))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) (lambda (id) '((quote altertable) schema id "engine" "logging")))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) (lambda (id) '((quote altertable) schema id "collation" collation)))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define ai (regex "[0-9]+"))) (lambda (id) '((quote altertable) schema id "auto_increment" ai)))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define col psql_identifier) (define body (or /* ALTER COLUMN */
				(parser '((atom "ADD" true) (atom "GENERATED" true) (or '((atom "BY" true) (atom "DEFAULT" true)) (atom "ALWAYS" true)) (atom "AS" true) (atom "IDENTITY") "("
					(atom "SEQUENCE" true) (atom "NAME" true) psql_identifier "." psql_identifier
					(? (atom "START" true) (atom "WITH" true) psql_expression)
					(? (atom "INCREMENT" true) (atom "BY" true) psql_expression)
					(? (atom "NO" true) (atom "MINVALUE" true))
					(? (atom "NO" true) (atom "MAXVALUE" true))
					(? (atom "CACHE" true) psql_expression)
					")") (lambda (col) (lambda (id) '((quote altercolumn) schema id col "auto_increment" true))))
				/* Type and default changes */
				(parser '((atom "TYPE" true) (define type psql_identifier) (define dimensions (or (parser '("(" (define a psql_int) "," (define b psql_int) ")") '((quote list) a b)) (parser '("(" (define a psql_int) ")") '((quote list) a)) (parser empty '((quote list))))) ) (lambda (col) (lambda (id) '('!begin '((quote altercolumn) schema id col "type" type) '((quote altercolumn) schema id col "dimensions" dimensions)))))
				(parser '((atom "SET" true) (atom "DEFAULT" true) (define def psql_expression)) (lambda (col) (lambda (id) '((quote altercolumn) schema id col "default" def))))
				(parser '((atom "DROP" true) (atom "DEFAULT" true)) (lambda (col) (lambda (id) '((quote altercolumn) schema id col "default" nil))))
				(parser '((atom "COLLATE" true) (define coll psql_identifier)) (lambda (col) (lambda (id) '((quote altercolumn) schema id col "collation" coll))))
			))) (body col))
		) ","))
	) (cons '!begin (map alters (lambda (alter) (alter id))))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	(define p (parser (or
		(parser (atom "SHUTDOWN" true) (begin (if policy (policy "system" true true) true) '(shutdown)))
		(parser (define query psql_select) (apply build_queryplan (apply untangle_query query)))
		psql_insert_into
		psql_insert_select
		psql_create_table
		psql_alter_table
		psql_update
		psql_delete

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define ifnot (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true))) (define id psql_identifier)) (begin (if policy (policy "system" true true) true) '((quote createdatabase) id (if ifnot true false))) )
		(parser '((atom "CREATE" true) (atom "USER" true) (define username psql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password psql_expression))))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "user" '(list "username" "password" "admin") '(list '(list username '('password password) false)) '(list) '((quote lambda) '() '((quote error) "user already exists")))
		))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username psql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password psql_expression))))
			(begin (if policy (policy "system" true true) true)
				'((quote scan) "system" "user" '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "password" '('password password)))))
		))

		/* GRANT syntax (PostgreSQL-style) -> reflect only admin and database-level access */
		/* GRANT <any> ON DATABASE db TO user */
		(parser '((atom "GRANT" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "DATABASE" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "access" '('list "username" "database") '('list '('list username db)))
		))
		/* GRANT <any> ON SCHEMA db TO user */
		(parser '((atom "GRANT" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "access" '('list "username" "database") '('list '('list username db)))
		))
		/* GRANT ALL PRIVILEGES ON ALL DATABASES is non-standard; ignore */
		/* Treat GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA db TO user as db-level access */
		(parser '((atom "GRANT" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "ALL" true) (atom "TABLES" true) (atom "IN" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "TO" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'('insert "system" "access" '('list "username" "database") '('list '('list username db)))
		))

		/* REVOKE syntax (PostgreSQL-style) -> mirror GRANT behavior */
		/* REVOKE <any> ON DATABASE db FROM user */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "DATABASE" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
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
		/* REVOKE <any> ON SCHEMA db FROM user */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
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
		/* REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA db FROM user -> treat as db-level */
		(parser '((atom "REVOKE" true) (+ (or psql_identifier ",")) (atom "ON" true) (atom "ALL" true) (atom "TABLES" true) (atom "IN" true) (atom "SCHEMA" true) (define db psql_identifier) (atom "FROM" true) (define username psql_identifier))
			(begin (if policy (policy "system" true true) true)
				'((quote scan)
					"system"
					"access"
					'(list "username" "database")
					'((quote lambda) '('username 'database) '((quote and) '((quote equal?) (quote username) username) '((quote equal?) (quote database) db)))
					'(list "$update")
					'((quote lambda) '((quote $update)) '((quote if) '((quote $update)) 1 0))
					(quote +)
					0
		)))

		(parser '((atom "CREATE" true) (define unique (? (atom "UNIQUE" true))) (atom "INDEX" true) (define id psql_identifier) (atom "ON" true) (define tbl (or (parser '(psql_identifier "." (define id psql_identifier)) id) psql_identifier)) (atom "USING" true) psql_identifier "(" (define cols (+ psql_identifier ",")) ")") '('createkey schema tbl id unique (cons (quote list) cols)))

		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true) (? (atom "FROM" true) (define schema psql_identifier))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '((quote resultrow) '((quote list) "Table" (quote tbl))))))
		(parser '((atom "SHOW" true) (atom "TABLE" true) (atom "STATUS" true) (? (atom "FROM" true) (define schema psql_identifier) (? (atom "LIKE" true) (define likepattern psql_expression)))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '('if '('strlike 'tbl '('coalesce 'likepattern "%")) '((quote resultrow) '('list "name" 'tbl "rows" "1")))))) /* TODO: engine version row_format avg_row_length data_length max_data_length index_length data_free auto_increment create_time update_time check_time collation checksum create_options comment max_index_length temporary */
		(parser '((atom "DESCRIBE" true) (define id psql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))
		(parser '((atom "SHOW" true) (atom "FULL" true) (atom "COLUMNS" true) (atom "FROM" true) (define id psql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line))))) /* TODO: Field Type Collation Null Key Default Extra(auto_increment) Privileges Comment */

		(parser '((atom "SHOW" true) (atom "VARIABLES" true)) '((quote map_assoc) '((quote list) "version" "0.9") '((quote lambda) '((quote key) (quote value)) '((quote resultrow) '((quote list) "Variable_name" (quote key) "Value" (quote value))))))
		(parser '((atom "SET" true) (atom "NAMES" true) (define charset psql_expression)) (quote true)) /* ignore */


		(parser '((atom "DROP" true) (atom "DATABASE" true) (define id psql_identifier)) (begin (if policy (policy "system" true true) true) '((quote dropdatabase) id)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define schema psql_identifier) (atom "." true) (define id psql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id psql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "SET" true) (? (atom "SESSION" true)) (define vars (* (parser '((? "@") (define key psql_identifier) "=" (define value (or
			(parser (atom "content" true) "content") /* quirks for SET xmloption = content */
			(parser (atom "warning" true) "warning") /* quirks for SET client_min_messages = warning */
			(parser (atom "heap" true) "heap") /* quirks for SET default_table_access_method = heap */
			psql_expression
		))) '((quote session) key value)) ","))) (cons '!begin vars))

		(parser '((atom "LOCK" true) (or (atom "TABLES" true) (atom "TABLE" true)) (+ (or psql_identifier '(psql_identifier (atom "AS" true) psql_identifier)) ",") (? (atom "READ" true)) (? (atom "LOCAL" true)) (? (atom "LOW_PRIORITY" true)) (? (atom "WRITE" true))) "ignore")
		(parser '((atom "UNLOCK" true) (or (atom "TABLES" true) (atom "TABLE" true))) "ignore")

		/* TODO: draw transaction number, commit */
		(parser '((atom "START" true) (atom "TRANSACTION" true)) '('session "transaction" 1))
		(parser '((atom "COMMIT" true)) '('session "transaction" nil))
		(parser '((atom "ROLLBACK" true)) '('session "transaction" nil))
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
							(concat x "\n") (insert schema tbl columns '((split x "\t")))
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
)
