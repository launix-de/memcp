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

(define sql_identifier_unquoted (parser (not
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
	(parser '((atom "'" false) (define x (regex "(\\\\.|[^\\'])*" false false)) (atom "'" false false)) (replace x "\'" "'"))
	(parser '((atom "\"" false) (define x (regex "(\\\\.|[^\\\"])*" false false)) (atom "\"" false false)) (replace x "\\\"" "\""))
)))

(define parse_sql (lambda (schema s) (begin


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
		(parser '((define a sql_expression3) "==" (define b sql_expression2)) '((quote equal??) a b))
		(parser '((define a sql_expression3) "=" (define b sql_expression2)) '((quote equal??) a b))
		(parser '((define a sql_expression3) "<>" (define b sql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a sql_expression3) "!=" (define b sql_expression2)) '((quote not) '((quote equal?) a b)))
		(parser '((define a sql_expression3) "<=" (define b sql_expression2)) '((quote <=) a b))
		(parser '((define a sql_expression3) ">=" (define b sql_expression2)) '((quote >=) a b))
		(parser '((define a sql_expression3) "<" (define b sql_expression2)) '((quote <) a b))
		(parser '((define a sql_expression3) ">" (define b sql_expression2)) '((quote >) a b))
		(parser '((define a sql_expression3) (atom "LIKE" true) (define b sql_expression2)) '('strlike a b))
		(parser '((define a sql_expression3) (atom "IN" true) "(" (define b (+ sql_expression ",")) ")") '('contains? (cons list b) a))
		(parser '((define a sql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define b (+ sql_expression ",")) ")") '('not '('contains? (cons list b) a)))
		sql_expression3
	)))

	(define sql_expression3 (parser (or
		(parser '((define a sql_expression4) "+" (define b sql_expression3)) '((quote +) a b))
		(parser '((define a sql_expression4) "-" (define b sql_expression3)) '((quote -) a b))
		sql_expression4
	)))

	(define sql_expression4 (parser (or
		(parser '((define a sql_expression5) "*" (define b sql_expression4)) '((quote *) a b))
		(parser '((define a sql_expression5) "/" (define b sql_expression4)) '((quote /) a b))
		sql_expression5
	)))

	(define sql_expression5 (parser (or
		(parser '((atom "NOT" true) (define expr sql_expression6)) '('not expr))
		(parser '((define expr sql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr sql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		sql_expression6
	)))

	(define sql_expression6 (parser (or
		(parser '("(" (define a sql_expression) ")") a)

		(parser '((atom "CASE" true) (define conditions (* (parser '((atom "WHEN" true) (define a sql_expression) (atom "THEN" true) (define b sql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch sql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" "*" ")") '((quote aggregate) 1 (quote +) 0))
		(parser '((atom "COUNT" true) "(" sql_expression ")") '((quote aggregate) 1 (quote +) 0))
		(parser '((atom "SUM" true) "(" (define s sql_expression) ")") '('aggregate s (quote +) 0))
		(parser '((atom "AVG" true) "(" (define s sql_expression) ")") '((quote /) '('aggregate s (quote +) 0) '('aggregate 1 (quote +) 0)))
		(parser '((atom "MIN" true) "(" (define s sql_expression) ")") '('aggregate s 'min nil))
		(parser '((atom "MAX" true) "(" (define s sql_expression) ")") '('aggregate s 'max nil))

		(parser '((atom "DATABASE" true) "(" ")") schema)
		(parser '((atom "PASSWORD" true) "(" (define p sql_expression) ")") '('password p))
		(parser '((atom "UNIX_TIMESTAMP" true) "(" ")") '('now))
		(parser '((atom "UNIX_TIMESTAMP" true) "(" (define p sql_expression) ")") '('parse_date p))
		(parser '((atom "FLOOR" true) "(" (define p sql_expression) ")") '('floor p))
		(parser '((atom "CEIL" true) "(" (define p sql_expression) ")") '('ceil p))
		(parser '((atom "CEILING" true) "(" (define p sql_expression) ")") '('ceil p))
		(parser '((atom "ROUND" true) "(" (define p sql_expression) ")") '('round p))
		(parser '((atom "UPPER" true) "(" (define p sql_expression) ")") '('toUpper p))
		(parser '((atom "LOWER" true) "(" (define p sql_expression) ")") '('toLower p))
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "UNSIGNED" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "INTEGER" true) ")") '('simplify p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CAST" true) "(" (define p sql_expression) (atom "AS" true) (atom "CHAR" true) (atom "CHARACTER" true) (atom "SET" true) (atom "utf8" true) ")") '('concat p)) /* TODO: proper implement CAST; for now make vscode work */
		(parser '((atom "CONCAT" true) "(" (define p (+ sql_expression ",")) ")") (cons 'concat p)) /* TODO: proper implement CAST; for now make vscode work */
		/* TODO: function call */

		(parser '((atom "COALESCE" true) "(" (define args (* sql_expression ",")) ")") (cons (quote coalesce) args))
		(parser '((atom "VALUES" true) "(" (define e sql_identifier_unquoted) ")") '('get_column "VALUES" true e true)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */
		(parser '((atom "VALUES" true) "(" (define e sql_identifier_quoted) ")") '('get_column "VALUES" true e false)) /* passthrough VALUES for now, the extract_stupid and replace_stupid will do their job for now */

		(parser (atom "NULL" true) nil)
		(parser (atom "TRUE" true) true)
		(parser (atom "FALSE" true) false)
		(parser '((atom "@" true) (define var sql_identifier_unquoted)) '('session var))
		(parser '((atom "@@" true) (define var sql_identifier_unquoted)) '('globalvars var))
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
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) '(id schema tbl false nil))
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier) (define id sql_identifier)) '(id schema tbl false nil))
		(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier)) '(tbl schema tbl false nil))
		(parser '((define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) '(id schema tbl false nil))
		(parser '((define tbl sql_identifier) (define id sql_identifier)) '(id schema tbl false nil))
		(parser '((define tbl sql_identifier)) '(tbl schema tbl false nil))
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
			(parser (define e sql_expression) '((extract_title e) e))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
			(?
				(atom "WHERE" true)
				(define condition sql_expression)
			)
		)
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
				(parser '((define col sql_expression) (define direction_desc (or
					(parser (atom "DESC" true) true)
					(parser(atom "ASC" true) false)
					(parser empty false)
				))) '(col direction_desc))
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
	) (begin
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

	(define sql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)
		/* TODO: DELETE tbl FROM tbl, tbl, tbl */
		(define tbl sql_identifier) /* TODO: ignorecase */
		(? '(
			(atom "WHERE" true)
			(define condition sql_expression)
		))
	) (begin
		(define replace_find_column (lambda (expr) (match expr
			'((symbol get_column) nil _ col ci) '((quote get_column) tbl false col ci) /* TODO: case insensititive column */
			(cons sym args) /* function call */ (cons sym (map args replace_find_column))
			expr
		)))
		replace_find_column /* workaround for optimizer bug: variable bindings in parsers */
		(set condition (replace_find_column (coalesce condition true)))
		(set filtercols (extract_columns_for_tblvar tbl condition))
		'((quote scan)
			schema
			tbl
			(cons list filtercols)
			'((quote lambda) (map filtercols (lambda(col) (symbol (concat tbl "." col)))) (replace_columns_from_expr condition))
			'(list "$update")
			'((quote lambda) '((quote $update)) '((quote $update)))
		)
	)))

	(define sql_insert_into (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
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
		(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
		(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
		(define coldesc (coalesce coldesc (map (show schema tbl) (lambda (col) (col "name")))))
		'('insert schema tbl (cons list coldesc) (cons list (map datasets (lambda (dataset) (cons list dataset)))) (cons list updatecols) (if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))
	)))

	(define sql_insert_select (parser '(
		(atom "INSERT" true)
		(define ignoreexists (? (atom "IGNORE" true true true)))
		(atom "INTO" true)
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
		(set updaterows2 (if (nil? updaterows) nil (merge updaterows)))
		(set updatecols (if (nil? updaterows) '() (cons "$update" (merge_unique (extract_assoc updaterows2 (lambda (k v) (extract_stupid v)))))))
		(define coldesc (coalesce coldesc (map (show schema tbl) (lambda (col) (col "name")))))
		'('begin
			'('set 'resultrow '('lambda '('item) '('insert schema tbl (cons list coldesc) (cons list '((cons list (map (produceN (count coldesc)) (lambda (i) '('nth 'item (+ (* i 2) 1))))))) (cons list updatecols) (if ignoreexists '('lambda '() true) (if (nil? updaterows) nil '('lambda (map updatecols (lambda (c) (symbol c))) '('$update (cons 'list (map_assoc updaterows2 (lambda (k v) (replace_stupid v)))))))))))
			(apply build_queryplan (apply untangle_query inner))
		)
	)))

	(define sql_create_table (parser '(
		(atom "CREATE" true)
		(atom "TABLE" true)
		(define ifnotexists (parser (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)) true))
		(define id sql_identifier)
		"("
		(define cols (* (or
			(parser '((atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ sql_identifier ",")) ")") '((quote list) "unique" "PRIMARY" (cons (quote list) cols)))
			(parser '((atom "UNIQUE" true) (atom "KEY" true) (define id sql_identifier) "(" (define cols (+ sql_identifier ",")) ")" (? (atom "USING" true) (atom "BTREE" true))) '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "CONSTRAINT" true) (define id (? sql_identifier)) (atom "FOREIGN" true) (atom "KEY" true) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true))) (? (atom "ON" true) (atom "UPDATE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true)))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2)))
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
				(define typeparams (regex "[^,)]*")) /* TODO: rest */
			) '((quote list) "column" col type dimensions typeparams))
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
			(parser '((atom "DEFAULT" true) (atom "CHARSET" false) "=" (define id sql_identifier)) '("charset" id))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "COLLATE" true) (define collation (regex "[a-zA-Z0-9_]+"))) '("collation" collation))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define collation (regex "[0-9]+"))) '("auto_increment" collation))
		)))
	) '((quote createtable) schema id (cons (quote list) cols) (cons (quote list) (merge options)) ifnotexists)))

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
			(parser '((atom "ADD" true) (?(atom "COLUMN" true))
				(define col sql_identifier)
				(define type sql_identifier)
				(define dimensions (or
					(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
					(parser '("(" (define a sql_int) ")") '((quote list) a))
					(parser empty '((quote list)))
				))
				(define typeparams (regex "[^,)]*")) /* TODO: rest */
			) (lambda (id) '((quote createcolumn) schema id col type dimensions typeparams)))
			(parser '((atom "DROP" true) (? (atom "COLUMN" true)) (define col sql_identifier)) (lambda (id) '((quote altertable) schema id "drop" col)))
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) (lambda (id) '((quote altertable) schema id "engine" "memory")))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) (lambda (id) '((quote altertable) schema id "engine" "sloppy")))
			(parser '((atom "ENGINE" true) "=" (atom "LOGGING" true)) (lambda (id) '((quote altertable) schema id "engine" "logging")))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) (lambda (id) '((quote altertable) schema id "engine" "safe")))
			(parser '((atom "COLLATE" true) "=" (define collation (regex "[a-zA-Z0-9_]+"))) (lambda (id) '((quote altertable) schema id "collation" collation)))
			(parser '((atom "AUTO_INCREMENT" true) "=" (define ai (regex "[0-9]+"))) (lambda (id) '((quote altertable) schema id "auto_increment" ai)))
		) ","))
	) (cons '!begin (map alters (lambda (alter) (alter id))))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	(define p (parser (or
		(parser (define query sql_select) (apply build_queryplan (apply untangle_query query)))
		sql_insert_into
		sql_insert_select
		sql_create_table
		sql_alter_table
		sql_update
		sql_delete

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote createdatabase) id))
		(parser '((atom "CREATE" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			'('insert "system" "user" '('list "username" "password") '('list '('list username '('password password)))))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			'((quote scan) "system" "user" '('list "username") '((quote lambda) '('username) '((quote equal?) (quote username) username)) '('list "$update") '('lambda '('$update) '('$update '('list "password" '('password password))))))

		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true) (? (atom "FROM" true) (define schema sql_identifier))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '((quote resultrow) '((quote list) "Table" (quote tbl))))))
		(parser '((atom "SHOW" true) (atom "TABLE" true) (atom "STATUS" true) (? (atom "FROM" true) (define schema sql_identifier) (? (atom "LIKE" true) (define likepattern sql_expression)))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '('if '('strlike 'tbl '('coalesce 'likepattern "%")) '((quote resultrow) '('list "name" 'tbl "rows" "1")))))) /* TODO: engine version row_format avg_row_length data_length max_data_length index_length data_free auto_increment create_time update_time check_time collation checksum create_options comment max_index_length temporary */
		(parser '((atom "DESCRIBE" true) (define id sql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))

		(parser '((atom "SHOW" true) (atom "VARIABLES" true)) '((quote map_assoc) '((quote list) "version" "0.9") '((quote lambda) '((quote key) (quote value)) '((quote resultrow) '((quote list) "Variable_name" (quote key) "Value" (quote value))))))
		(parser '((atom "SET" true) (atom "NAMES" true) (define charset sql_expression)) (quote true)) /* ignore */


		(parser '((atom "DROP" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote dropdatabase) id))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define schema sql_identifier) (atom "." true) (define id sql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id sql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "SET" true) (? (atom "SESSION" true)) (define vars (* (parser '((? "@") (define key sql_identifier) "=" (define value sql_expression)) '((quote session) key value)) ","))) (cons '!begin vars))

		(parser '((atom "LOCK" true) (or (atom "TABLES" true) (atom "TABLE" true)) (+ (or sql_identifier '(sql_identifier (atom "AS" true) sql_identifier)) ",") (? (atom "READ" true)) (? (atom "LOCAL" true)) (? (atom "LOW_PRIORITY" true)) (? (atom "WRITE" true))) "ignore")
		(parser '((atom "UNLOCK" true) (or (atom "TABLES" true) (atom "TABLE" true))) "ignore")
		(parser '((atom "COMMIT" true) (or (atom "TABLES" true) (atom "TABLE" true))) "ignore")
		"" /* comment only command */
		))) 
	((parser (define command p) command "^(?:/\\*.*?\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
	)))

(define parse_sql_multi (lambda (schema s delimiter) (begin
	/* TODO: DELIMITER commands, version-specific meta commands usw */
	/* this implements a SQL preprocessor that separates multiple commands into an array and resolves SQL version macros */
	/* TODO: work on big file streams: detect incomplete SQL queries and return them as rest, so the caller can append more lines and reparse */

	/*(define parse_sql (lambda (schema s) '("SQL:" s)))*/
	(define tailrecursiveparser (lambda (pre scan delimiter) (match scan
		(regex "^--[^\n]+?(\n.*)" _ rest) (tailrecursiveparser pre rest delimiter)
		(regex "(?is)^DELIMITER([^\n]+)(.*)" _ del rest) (tailrecursiveparser pre rest del)
		(concat delimiter rest) (cons (parse_sql schema pre) (tailrecursiveparser "" rest delimiter))
		(regex "(?s)^('(?:''|\\'|[^'])*')(.*)" _ v rest) (tailrecursiveparser (concat pre v) rest delimiter)
		(regex "(?s)^([a-zA-Z0-9_]+)(.*)" _ v rest) (tailrecursiveparser (concat pre v) rest delimiter)
		(regex "(?s)^([^'a-zA-Z])(.*)" _ v rest) (tailrecursiveparser (concat pre v) rest delimiter)
		"" '((parse_sql schema pre))
		error("failed to parse compound SQL")
	)))
	(cons '!begin (tailrecursiveparser "" s ";"))
)))
/*
> (parse_sql_multi "sparse" "select * from a--man\n; 'b=4' ; moms ;\nDELIMITER foo\nafterwards" ";")


*/

