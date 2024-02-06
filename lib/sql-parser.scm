/*
Copyright (C) 2023  Carl-Philip Hänsch

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

(define sql_string (parser '( "'" (define x (regex "(\\\\'|[^'])*" false false)) "'") (replace x "\'" "'")))

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
		sql_int
		sql_string
		sql_column
	)))

	(define sql_select (parser '(
		(atom "SELECT" true)
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil "*")))
			(parser '((define tbl sql_identifier) "." "*") '("*" '((quote get_column) tbl "*")))
			(parser '((define e sql_expression) (atom "AS" true) (define title sql_identifier)) '(title e))
			(parser (define e sql_expression) '((extract_title e) e))
		) ","))
		(? '(
			(atom "FROM" true)
			(define from (+
				(or
					/* TODO: inner select as from */
					(parser '((define tbl sql_identifier) (atom "AS" true) (define id sql_identifier)) '(id tbl))
					(parser '((define tbl sql_identifier)) '(tbl tbl))
				)
			","))
			(? '(
				(atom "WHERE" true)
				(define condition sql_expression)
			))
		))
		/* TODO: GROUP BY + HAVING */
		/* TODO: ORDER BY + LIMIT */
	) (build_queryplan schema (if (nil? from) '() from) (merge cols) condition)))

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
		(define cols (* (parser '(
			(define col sql_identifier)
			(define type sql_identifier)
			(define dimensions (or
				(parser '("(" (define a sql_int) "," (define b sql_int) ")") '((quote list) a b))
				(parser '("(" (define a sql_int) ")") '((quote list) a))
				(parser empty '((quote list)))
			))
			(define typeparams (regex "[^,)]*")) /* TODO: rest */
		) '((quote list) col type dimensions typeparams)) ","))
		")"
	) '((quote createtable) schema id (cons (quote list) cols))))

	/* TODO: ignore comments wherever they occur --> Lexer */
	((parser (or
		sql_select
		sql_insert_into
		sql_create_table
		sql_update
		sql_delete

		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote createdatabase) id))

		(parser '((atom "SHOW" true) (atom "DATABASES" true)) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Database" (quote schema))))))
		(parser '((atom "SHOW" true) (atom "TABLES" true)) '((quote map) '((quote show) schema) '((quote lambda) '((quote schema)) '((quote resultrow) '((quote list) "Table" (quote schema))))))

		(parser '((atom "DROP" true) (atom "DATABASE" true) (define id sql_identifier)) '((quote dropdatabase) id))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define id sql_identifier)) '((quote droptable) schema id))
	)) s)
)))

