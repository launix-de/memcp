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

(define extract_columns_from_expr (lambda (expr) (match expr
	'((symbol get_column) tblvar col) '('(tblvar col))
	(cons sym args) /* function call */ (merge (map args extract_columns_from_expr))
	'()
)))

(define replace_columns_from_expr (lambda (expr) (match expr
	'((symbol get_column) tblvar col) (symbol col) /* TODO: rename in outer scans */
	(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
	expr /* literals */
)))

/* condition for update/delete */
(define build_condition (lambda (schema table condition) (if
	(nil? condition)
	'((quote lambda) '() (quote true))
	(begin
		(set cols (extract_columns_from_expr condition))
		(set cols (map cols (lambda (x) (match x '(tblvar col) (symbol col))))) /* assume that tblvar always points to table (todo: pass tblvar and filter according to join order) */

		/* return lambda for tbl condition */
		'((quote lambda) cols (replace_columns_from_expr condition))
	)
)))

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields) (begin
	/* tables: '('(alias tbl) ...) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* TODO: WHERE, GROUP, HAVING, ORDER, LIMIT */
	/* expressions will use (get_column tblvar col) for reading from columns. we have to replace it with the correct variable */

	/* returns a list of '(tblvar col) */
	(define extract_columns (lambda (col expr) (match expr
		'((symbol get_column) tblvar col) '('(tblvar col))
		(cons sym args) /* function call */ (merge (map args extract_columns_from_expr)) /* TODO: use collector */
		'()
	)))

	/* changes (get_column tblvar col) into its counterpart */
	(define replace_columns (lambda (col expr) (match expr
		'((symbol get_column) tblvar col) (symbol col) /* TODO: rename in outer scans */
		(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
		expr /* literals */
	)))

	/* expand *-columns */
	(set fields (merge (extract_assoc fields (lambda (col expr) (match col
		"*" (merge (map tables (lambda (t) (match t '(alias tbl) /* all FROM-tables*/
			(merge (map (show schema tbl) (lambda (coldesc) /* all columns of each table */
				'((coldesc "name") '((quote get_column) alias (coldesc "name")))
			)))
		))))
		'(col expr)
	)))))

	/* columns: '('(tblalias colname) ...) */
	(set columns (merge (extract_assoc fields extract_columns)))
	/* TODO: expand fields if it contains '(tblalias "*") or '("*" "*") */
		'((symbol get_column_all)) (merge (map tables (lambda (t) (match t '(alias tbl) (map (show schema tbl) (lambda (col) '(tblvar (col "name"))))))))

	/* TODO: sort tables according to join plan */
	(define build_scan (lambda (tables)
		(match tables
			(cons '(alias tbl) tables) /* outer scan */
				'((quote scan) schema tbl
					'((quote lambda) '() (quote true)) /* TODO: filter */
					/* todo filter columns for alias */
					'((quote lambda) (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))) (build_scan tables))
					/* TODO: reduce+neutral */)
			'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields replace_columns)))
		)
	))
	(build_scan tables)
)))
