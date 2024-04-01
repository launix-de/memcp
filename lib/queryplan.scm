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
	(cons (symbol aggregate) args) /* aggregates: keep unchanged */ (cons aggregate args)
	'((symbol get_column) tblvar col) (symbol col) /* TODO: rename in outer scans */
	(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
	expr /* literals */
)))

(define extract_aggregates (lambda (expr) (match expr
	(cons (symbol aggregate) args) '(args)
	(cons sym args) /* function call */ (merge (map args extract_aggregates))
	/* literal */ '()
)))
(define extract_aggregates_assoc (lambda (fields) (merge (extract_assoc fields (lambda (key expr) (extract_aggregates expr))))))

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

/* emulate metadata tables (TODO: information_schema.columns) */
(define get_schema (lambda (schema tbl) (match '(schema tbl)
	/* special tables */
	'((ignorecase "information_schema") (ignorecase "tables")) '('("name" "table_schema") '("name" "table_name") '("name" "table_type"))
	(show schema tbl) /* otherwise: fetch from metadata */
)))
(define scan_wrapper (lambda (schema tbl filter map reduce neutral) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "tables"))
		'((quote scan) schema 
			'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote list) "table_schema" (quote schema) "table_name" (quote tbl) "table_type" "BASE TABLE")))))) 
			filter map reduce neutral)
	'(schema tbl) /* normal case */
		'((quote scan) schema tbl filter map reduce neutral)
)))

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields condition group having order limit offset) (begin
	/* tables: '('(alias schema tbl) ...) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* TODO: GROUP, HAVING,  */
	/* expressions will use (get_column tblvar col) for reading from columns. we have to replace it with the correct variable */
	/* TODO: unnest arbitrary queries -> turn them into a big schema+tables+fields+condition */

	/*
		Query builder masterplan:
		1. make sure all optimizations are done (unnesting arbitrary queries, leave just one big table list with a field list, conditions, as well as a order+limit+offset)
		2. if order+limit+offset -> find or create all tables that have to be nestedly scanned. if two tables are clumsed together, create a prejoin. recurse over build_queryplan without the order clause.
		3. if group stage -> find or create the preaggregate table, scan over the preaggregate
		4. scan the rest of the tables

	*/

	/* tells whether there is an aggregate inside */
	(define expr_find_aggregate (lambda (expr) (match expr
		'((symbol aggregate) item reduce neutral) true
		(cons sym args) /* function call */ (reduce args (lambda (a b) (or a (expr_find_aggregate b))))
		false
	)))

	/* replace all aggregates with respected subtitutions */
	(define expr_replace_aggregate (lambda (expr indexmap) (match expr
		(cons (symbol aggregate) agg) (indexmap (string agg))
		(cons sym args) /* function call */ (cons sym (map args (lambda (x) (expr_replace_aggregate x indexmap))))
		expr
	)))

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

	(if (coalesce order limit offset) (begin
		/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
		(match order
			'('('((symbol get_column) tblalias "ORDINAL_POSITION") direction)) (build_queryplan schema tables fields condition group having nil nil nil) /* ignore ordering for some cases by now to use the dbeaver tool */
			(error "Ordered scan is not implemented yet")
		)
	) (begin
		/* set group to 1 if fields contain aggregates even if not */
		(define group (coalesce group (if (reduce_assoc fields (lambda (a key v) (or a (expr_find_aggregate v))) false) 1 nil)))

		(if group (begin
			/* TOOD: find or create preaggregate table, scan over preaggregate */
			(define ags (extract_aggregates_assoc fields))
			(define build_indexmap (lambda (expr ags) (match ags
				(cons head tail) (cons (string head) (cons '((quote car) expr) (build_indexmap '((quote cdr) expr) tail)))
				'()
			)))
			(define indexmap (build_indexmap (quote ags) ags))
			(if (equal? group 1) (begin
				/* one implemented corner case; TODO: recursively go through the scan tables */
				(set columns (merge (extract_assoc fields extract_columns)))
				(define build_reducer (lambda (ags) (match ags
					(cons '(val reduce neutral) rest) '((quote match) (quote p) '((quote list) '((quote cons) (quote xa) (quote a)) '((quote cons) (quote xb) (quote b))) '((quote cons) '(reduce (quote xa) (quote xb)) (build_reducer rest)))
					'() '((quote list))
				)))
				(define build_scan (lambda (tables)
					(match tables
						(cons '(alias schema tbl) tables) /* outer scan */
							(scan_wrapper schema tbl
								(build_condition schema tbl condition) /* TODO: conditions in multiple tables */
								/* todo filter columns for alias */
								'((quote lambda) (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))) (build_scan tables))
								/* reduce */ '((quote lambda) (quote p) (build_reducer ags))
								/* neutral */ (cons (quote list) (map ags (lambda (val) (match val '(expr reduce neutral) neutral))))
							)
						'() /* final inner */ (cons (quote list) (map ags (lambda (val) (match val '(expr reduce neutral) (replace_columns nil expr)))))
					)
				))
				'((quote begin)
					'((quote define) (quote ags) (build_scan tables))
					'((quote resultrow) (cons (quote list) (map_assoc fields (lambda (key value) (expr_replace_aggregate value indexmap)))))
				)
			) (begin
				(error "Grouping and aggregates are not implemented yet (Preaggregate tables)")
			))
		) (begin
			/* else: normal table scan */

			/* expand *-columns */
			(set fields (merge (extract_assoc fields (lambda (col expr) (match col
				"*" (merge (map tables (lambda (t) (match t '(alias schema tbl) /* all FROM-tables*/
					(merge (map (get_schema schema tbl) (lambda (coldesc) /* all columns of each table */
						'((coldesc "name") '((quote get_column) alias (coldesc "name")))
					)))
				))))
				'(col expr)
			)))))

			/* columns: '('(tblalias colname) ...) */
			(set columns (merge (extract_assoc fields extract_columns)))
			/* TODO: expand fields if it contains '(tblalias "*") or '("*" "*") */
				'((symbol get_column_all)) (merge (map tables (lambda (t) (match t '(alias schema tbl) (map (get_schema schema tbl) (lambda (col) '(tblvar (col "name"))))))))

			/* TODO: sort tables according to join plan */
			/* TODO: match tbl to inner query vs string */
			(define build_scan (lambda (tables)
				(match tables
					(cons '(alias schema tbl) tables) /* outer scan */
						(scan_wrapper schema tbl
							(build_condition schema tbl condition) /* TODO: conditions in multiple tables */
							/* todo filter columns for alias */
							'((quote lambda) (map columns (lambda(column) (match column '(tblvar colname) (symbol colname)))) (build_scan tables))
						)
					'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields replace_columns)))
				)
			))
			(build_scan tables)
		))
	))
)))
