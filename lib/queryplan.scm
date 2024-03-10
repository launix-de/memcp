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

/*

How MemCPs query plan builder works
-----------------------------------

MemCP will not implement any filtering or ordering on scheme lists directly since this will be very costly.
Instead, the storage engine is used to do these operations. The storage engine will automatically analyze a
lambda expression for filtering/ordering and will eventually create and use indexes.

Every filter and sort will be executed on a base table. Therefore, in GROUP BY clauses, a temporary table
has to be created. Also for cross joins (joins that either have no equality condition between the tables or
the equality is not on a unique column), there has to be a temporary cross-table.

when building a queryplan, there is a parameter `tables` which contains all tables that have to be joined.
Relevant for the iterator is now the "core". which is:
the list of tables in tables t1 that are not connected over a join t1,t2,t1.col1=t2.col2 where there is a unique key (t2.col2)
(helper function (unique? schema tbl col col col))

if the core consists of a single table, scan this table
if the core consists of two or more tables, create a temporary join table --> prejoins
if there is a group function, create a temporary preaggregate table
(helper function temptable(tbllist, collist) -> tbllist is the list of tables to be joined and collist is the list of (table, col) that will also be unique)

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
(define build_queryplan (lambda (schema tables fields condition group having order limit offset) (begin
	/* tables: '('(alias schema tbl) ...) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* TODO: GROUP, HAVING, ORDER, LIMIT, OFFSET */
	/* expressions will use (get_column tblvar col) for reading from columns. we have to replace it with the correct variable */
	/* TODO: unnest arbitrary queries -> turn them into a big schema+tables+fields+condition */

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
		"*" (merge (map tables (lambda (t) (match t '(alias schema tbl) /* all FROM-tables*/
			(merge (map (show schema tbl) (lambda (coldesc) /* all columns of each table */
				'((coldesc "name") '((quote get_column) alias (coldesc "name")))
			)))
		))))
		'(col expr)
	)))))

	/* columns: '('(tblalias colname) ...) */
	(set columns (merge (extract_assoc fields extract_columns)))
	/* TODO: expand fields if it contains '(tblalias "*") or '("*" "*") */
		'((symbol get_column_all)) (merge (map tables (lambda (t) (match t '(alias schema tbl) (map (show schema tbl) (lambda (col) '(tblvar (col "name"))))))))

	/* TODO: sort tables according to join plan */
	/* TODO: match tbl to inner query vs string */
	(define build_scan (lambda (tables)
		(match tables
			(cons '(alias schema tbl) tables) /* outer scan */
				'((quote scan) schema tbl /* TODO: scan vs scan_order when order or limit is present */
					(build_condition schema tbl condition) /* TODO: conditions in multiple tables */
					/* todo filter columns for alias */
					'((quote lambda) (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))) (build_scan tables))
					/* TODO: reduce+neutral */)
			'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields replace_columns)))
		)
	))
	(build_scan tables)
)))
