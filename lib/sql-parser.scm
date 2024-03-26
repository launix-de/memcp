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

/* TODO: make lower_identifiers settable */
(define sql_identifier_unquoted (parser (define id (regex "[a-zA-Z_][a-zA-Z0-9_]*")) (if lower_identifiers (toLower id) id))) /* raw -> toLower */
(define sql_identifier (parser (or
	(parser '("`" (define id (regex "(?:[^`]|``)+" false false)) "`") (replace id "``" "`")) /* with backtick */
	sql_identifier_unquoted
)))

(define sql_column (parser (or
	(parser '((define tbl sql_identifier) "." (define col sql_identifier)) '((quote get_column) tbl col))
	(parser (define col sql_identifier) '((quote get_column) nil col))
)))

(define sql_int (parser (define x (regex "[0-9]+")) (simplify x)))

(define sql_string (parser '("'" (define x (regex "(\\\\'|[^'])*" false false)) "'") (replace x "\'" "'")))

(define parse_sql (lambda (schema s) (begin


	/* derive the description of a column from its expression */
	(define extract_title (lambda (expr) (match expr
		'((symbol get_column) nil col) col
		'((symbol get_column) tblvar col) (concat tblvar "." col)
		(cons sym args) /* function call */ (concat (cons sym (map args extract_title)))
		(concat expr)
	)))

	/* merge two arrays into a dict */
	(define zip_cols (lambda(cols tuple) (match cols
		(cons col cols) (cons col (cons (car tuple) (zip_cols cols (cdr tuple))))
		'()
	)))
	
	/* TODO: (expr), a + b, a - b, a * b, a / b */
	(define sql_expression (parser (or
		(parser '((atom "@" true) (define var sql_identifier) (atom ":=" true) (define value sql_expression)) '((quote session) var value))
		(parser '((define a sql_expression1) (atom "OR" true) (define b (+ sql_expression1 (atom "OR" true)))) (cons (quote or) (cons a b)))
		sql_expression1
	)))
	(define sql_expression1 (parser (or
		(parser '((define a sql_expression2) (atom "AND" true) (define b (+ sql_expression2 (atom "AND" true)))) (cons (quote and) (cons a b)))
		sql_expression2
	)))

	(define sql_expression2 (parser (or
		(parser '((define a sql_expression3) "=" (define b sql_expression2)) '((quote equal?) a b))
		(parser '((define a sql_expression3) "<=" (define b sql_expression2)) '((quote <=) a b))
		(parser '((define a sql_expression3) ">=" (define b sql_expression2)) '((quote >=) a b))
		(parser '((define a sql_expression3) "<" (define b sql_expression2)) '((quote <) a b))
		(parser '((define a sql_expression3) ">" (define b sql_expression2)) '((quote >) a b))
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
		(parser '("(" (define a sql_expression) ")") a)
		(parser '((atom "DATABASE" true) "(" ")") schema)
		(parser '((atom "PASSWORD" true) "(" (define p sql_expression) ")") '((quote password) p))
		/* TODO: function call */
		(parser (atom "NULL" true) nil)
		(parser '((atom "@" true) (define var sql_identifier)) '((quote session) var))
		sql_int
		sql_string
		sql_column
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
			(parser "*" '("*" '((quote get_column) nil "*")))
			(parser '((define tbl sql_identifier) "." "*") '("*" '((quote get_column) tbl "*")))
			(parser '((define e sql_expression) (atom "AS" true) (define title sql_identifier)) '(title e))
			(parser (define e sql_expression) '((extract_title e) e))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+
				(or
					(parser '((atom "(" true) (define query sql_select) (atom ")" true) (atom "AS" true) (define id sql_identifier)) '(id schema query)) /* inner select as from */
					(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) '(id schema tbl))
					(parser '((define schema sql_identifier) (atom "." true) (define tbl sql_identifier)) '(tbl schema tbl))
					(parser '((define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) '(id schema tbl))
					(parser '((define tbl sql_identifier)) '(tbl schema tbl))
				)
			","))
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
				'((define limit sql_expression))
				'((define offset sql_expression) (atom "," true) (define limit sql_expression))
				'((define limit sql_expression) (atom "OFFSET" true) (define offset sql_expression))
			)
		)
	) '(schema (if (nil? from) '() from) (merge cols) condition group having order limit offset)))

	(define sql_update (parser '(
		(atom "UPDATE" true)
		/* TODO: UPDATE tbl FROM tbl, tbl, tbl */
		(define tbl sql_identifier)
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
	(set cols (merge cols))
	'((quote scan)
		schema
		tbl
		(build_condition schema tbl condition)
		'((quote lambda)
			(cons (quote $update) (merge (extract_assoc cols (lambda (col expr) (map (extract_columns_from_expr expr) (lambda (x) (match x '(tblvar col) (symbol col))))))))
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
		(define tbl sql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition sql_expression)
		))
	) '((quote scan) schema tbl (build_condition schema tbl condition) '((quote lambda) '((quote $update)) '((quote $update))))))

	(define sql_insert_into (parser '(
		(atom "INSERT" true)
		(atom "INTO" true)
		(define tbl sql_identifier)
		"("
		(define coldesc (*
			sql_identifier
		","))
		")"
		(atom "VALUES" true)
		(define datasets (* (parser '(
			"("
			(define dataset (* sql_expression ","))
			")"
		) dataset) ","))
	) (cons (quote begin) (map (map datasets (lambda (dataset) (zip_cols coldesc dataset))) (lambda (dataset) '((quote insert) schema tbl (cons (quote list) dataset)))))))

	(define sql_create_table (parser '(
		(atom "CREATE" true)
		(atom "TABLE" true)
		(define id sql_identifier)
		"("
		(define cols (* (or
			(parser '((atom "PRIMARY" true) (atom "KEY" true) "(" (define cols (+ sql_identifier ",")) ")") '((quote list) "unique" "PRIMARY" (cons (quote list) cols)))
			(parser '((atom "UNIQUE" true) (atom "KEY" true) (define id sql_identifier) "(" (define cols (+ sql_identifier ",")) ")") '((quote list) "unique" id (cons (quote list) cols)))
			(parser '((atom "FOREIGN" true) (atom "KEY" true) (define id (? sql_identifier)) "(" (define cols1 (+ sql_identifier ",")) ")" (atom "REFERENCES" true) (define tbl2 sql_identifier) "(" (define cols2 (+ sql_identifier ",")) ")" (? (atom "ON" true) (atom "DELETE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true))) (? (atom "ON" true) (atom "UPDATE" true) (or (atom "RESTRICT" true) (atom "CASCADE" true) (atom "SET NULL" true)))) '((quote list) "foreign" id (cons (quote list) cols1) tbl2 (cons (quote list) cols2)))
			(parser '((atom "KEY" true) sql_identifier "(" (+ sql_identifier ",") ")") '((quote list))) /* ignore index definitions */
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
			(parser '((atom "ENGINE" true) "=" (atom "MEMORY" true)) '("engine" "memory"))
			(parser '((atom "ENGINE" true) "=" (atom "SLOPPY" true)) '("engine" "sloppy"))
			(parser '((atom "ENGINE" true) "=" (atom "SAFE" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "MyISAM" true)) '("engine" "safe"))
			(parser '((atom "ENGINE" true) "=" (atom "InnoDB" true)) '("engine" "safe"))
			(parser '((atom "DEFAULT" true) (atom "CHARSET" false) "=" sql_identifier) '())
		)))
	) '((quote createtable) schema id (cons (quote list) cols) (cons (quote list) (merge options)))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	(define p (parser (or
		(parser (define query sql_select) (apply build_queryplan query))
		sql_insert_into
		sql_create_table
		sql_update
		sql_delete

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote createdatabase) id))
		(parser '((atom "CREATE" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			'((quote insert) "system" "user" '((quote list) "username" username "password" '((quote password) password))))
		(parser '((atom "ALTER" true) (atom "USER" true) (define username sql_identifier)
			(? '((atom "IDENTIFIED" true) (atom "BY" true) (define password sql_expression))))
			'((quote scan) "system" "user" '((quote lambda) '((quote username)) '((quote equal?) (quote username) username)) '((quote lambda) '((quote $update)) '((quote $update) '((quote list) "password" '((quote password) password))))))

		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true) (? (atom "FROM" true) (define schema sql_identifier))) '((quote map) '((quote show) schema) '((quote lambda) '((quote tbl)) '((quote resultrow) '((quote list) "Table" (quote tbl))))))
		(parser '((atom "DESCRIBE" true) (define id sql_identifier)) '((quote map) '((quote show) schema id) '((quote lambda) '((quote line)) '((quote resultrow) (quote line)))))

		(parser '((atom "SHOW" true) (atom "VARIABLES" true)) '((quote map_assoc) '((quote list) "version" "0.9") '((quote lambda) '((quote key) (quote value)) '((quote resultrow) '((quote list) "Variable_name" (quote key) "Value" (quote value))))))
		(parser '((atom "SET" true) (atom "NAMES" true) (define charset sql_expression)) (quote true)) /* ignore */


		(parser '((atom "DROP" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote dropdatabase) id))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define if_exists (? (atom "IF" true) (atom "EXISTS" true))) (define id sql_identifier)) '((quote droptable) schema id (if if_exists true false)))
		(parser '((atom "SET" true) (? (atom "SESSION" true)) (define vars (* (parser '((? "@") (define key sql_identifier) "=" (define value sql_expression)) '((quote session) key value)) ","))) (cons (quote begin) vars))
		empty
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
	(cons (quote begin) (tailrecursiveparser "" s ";"))
)))
/*
> (parse_sql_multi "sparse" "select * from a--man\n; 'b=4' ; moms ;\nDELIMITER foo\nafterwards" ";")


*/

