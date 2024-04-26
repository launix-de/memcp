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

/* helper functions:
 - (build_queryplan schema tables fields condition group having order limit offset) builds a lisp expression that runs the query and calls resultrow for each result tuple
 - (build_scan schema tables cols map reduce neutral neutral2 condition group having order limit offset) builds a lisp expression that scans the tables
 - (extract_columns_for_tblvar expr tblvar) extracts a list of used columns for each tblvar '(tblvar col)
 - (replace_columns expr) replaces all (get_column ...) and (aggregate ...) with values

*/

/* returns a list of '(string...) */
(define extract_columns_for_tblvar (lambda (tblvar expr) (match expr
	'((symbol get_column) (eval tblvar) col) '(col)
	(cons sym args) /* function call */ (merge_unique (map args (lambda (arg) (extract_columns_for_tblvar tblvar arg))))
	'()
)))

/* changes (get_column tblvar col) into its symbol */
(define replace_columns_from_expr (lambda (expr) (match expr
	(cons (symbol aggregate) args) /* aggregates: don't dive in */ (cons aggregate args)
	'((symbol get_column) tblvar col) (symbol (concat tblvar "." col))
	(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
	expr /* literals */
)))

/* returns a list of all aggregates in this expr */
(define extract_aggregates (lambda (expr) (match expr
	(cons (symbol aggregate) args) '(args)
	(cons sym args) /* function call */ (merge (map args extract_aggregates))
	/* literal */ '()
)))

/* emulate metadata tables (TODO: information_schema.columns) */
(define get_schema (lambda (schema tbl) (match '(schema tbl)
	/* special tables */
	'((ignorecase "information_schema") (ignorecase "tables")) '('("name" "table_schema") '("name" "table_name") '("name" "table_type"))
	(show schema tbl) /* otherwise: fetch from metadata */
)))
(define scan_wrapper (lambda (schema tbl filtercols filter mapcols map reduce neutral) (match '(schema tbl)
	'((ignorecase "information_schema") (ignorecase "tables"))
		'((quote scan) schema 
			'((quote merge) '((quote map) '((quote show)) '((quote lambda) '((quote schema)) '((quote map) '((quote show) (quote schema)) '((quote lambda) '((quote tbl)) '((quote list) "table_schema" (quote schema) "table_name" (quote tbl) "table_type" "BASE TABLE")))))) 
			filtercols filter mapcols map reduce neutral)
	'(schema tbl) /* normal case */
		'((quote scan) schema tbl filtercols filter mapcols map reduce neutral)
)))

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields condition group having order limit offset) (begin
	/* tables: '('(alias schema tbl) ...) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* expressions will use (get_column tblvar col) for reading from columns. we have to replace it with the correct variable */
	/* TODO: unnest arbitrary queries -> turn them into a big schema+tables+fields+condition */

	/*
		Query builder masterplan:
		1. make sure all optimizations are done (unnesting arbitrary queries, leave just one big table list with a field list, conditions, as well as a order+limit+offset)
		2. if group is present: split the queryplan into filling the grouped table and scanning it -> find or create the preaggregate table, scan over the preaggregate
		3. if order+limit+offset is present: split the queryplan into providing a scannable tableset and a ordered scan on that tableset
		   -> find or create all tables that have to be nestedly scanned. if two tables are clumsed together, create a prejoin. recurse over build_queryplan without the order clause.
		4. scan the rest of the tables

	*/

	/* tells whether there is an aggregate inside */
	(define expr_find_aggregate (lambda (expr) (match expr
		'((symbol aggregate) item reduce neutral) 5
		(cons sym args) /* function call */ (reduce args (lambda (a b) (or a (expr_find_aggregate b))) false)
		false
	)))

	/* replace all aggregates with respected subtitutions */
	(define expr_replace_aggregate (lambda (expr indexmap) (match expr
		(cons (symbol aggregate) agg) (indexmap (string agg))
		(cons sym args) /* function call */ (cons sym (map args (lambda (x) (expr_replace_aggregate x indexmap))))
		expr
	)))

	/* put all schemas of corresponding tables into an assoc */
	(set schemas (merge (map tables (lambda (t) (match t '(alias schema tbl) '(alias (get_schema schema tbl)))))))

	/* find those columns that have no table */
	(define replace_find_column (lambda (expr) (match expr
		'((symbol get_column) nil col) '((quote get_column) (reduce_assoc schemas (lambda (a alias cols) (if (reduce cols (lambda (a coldef) (or a (equal? (coldef "name") col))) false) alias a)) (lambda () (error (concat "column " col " does not exist in tables")))) col)
		(cons sym args) /* function call */ (cons sym (map args replace_find_column))
		expr
	)))

	/* expand *-columns */
	(set fields (merge (extract_assoc fields (lambda (col expr) (match col
		"*" (match expr
			/* *.* */
			'((symbol get_column) nil "*")(merge (map tables (lambda (t) (match t '(alias schema tbl) /* all FROM-tables*/
				(merge (map (get_schema schema tbl) (lambda (coldesc) /* all columns of each table */
					'((coldesc "name") '((quote get_column) alias (coldesc "name")))
				)))
			))))
			/* tbl.* */
			'((symbol get_column) tblvar "*")(merge (map tables (lambda (t) (match t '(alias schema tbl) /* one FROM-table*/
				(if (equal? alias tblvar)
					(merge (map (get_schema schema tbl) (lambda (coldesc) /* all columns of each table */
						'((coldesc "name") '((quote get_column) alias (coldesc "name")))
					)))
					'())
			))))
		)
		'(col (replace_find_column expr))
	)))))

	/* set group to 1 if fields contain aggregates even if not */
	(define group (coalesce group (if (reduce_assoc fields (lambda (a key v) (or a (expr_find_aggregate v))) false) '(1) nil)))

	(if group (begin
		/* group: extract aggregate clauses and split the query into two parts: gathering the aggregates and outputting them */
		(set group (map group replace_find_column))
		(define ags (merge (extract_assoc fields (lambda (key expr) (extract_aggregates expr)))))

		(match tables
			/* TODO: allow for more than just group by single table */
			'('(tblvar schema tbl)) (begin
				/* prepare preaggregate */

				/* TODO: check if there is a foreign key on tbl.groupcol and then reuse that table */
				(set grouptbl (concat "." tbl ":" group))
				(createtable schema grouptbl (cons
					/* unique key over all identiying columns */ '("unique" "group" (map group (lambda (col) (concat col))))
					/* all identifying columns */ (map group (lambda (col) '("column" (concat col) "any"/* TODO get type from schema */ '() '())))
				) '("engine" "sloppy") true)

				/* preparation */
				/* changes (get_column tblvar col) into its counterpart */
				(define replace_columns_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) rest) (symbol (concat rest)) /* aggregate helper column */
					'((symbol get_column) tblvar col) (symbol (concat expr)) /* grouped col */
					(cons sym args) /* function call */ (cons sym (map args replace_columns_agg_expr))
					expr /* literals */
				)))

				(define tblvar_cols (merge_unique (map group (lambda (col) (extract_columns_for_tblvar tblvar col)))))

				/* HAVING */
				(set rest_condition (replace_find_column (coalesce having true)))
				(set filtercols (extract_columns_for_tblvar tblvar rest_condition))

				(merge
					/* TODO: partitioning hint for insert -> same partitioning scheme as tables */
					'((quote begin)
						/* INSERT IGNORE group cols into preaggregate */
						/* TODO: use bulk insert in scan reduce phase (and filter duplicates from a bulk!) */
						'((quote begin)
							'((quote set) (quote resultrow) '((quote lambda) '((quote item)) '((quote insert) schema grouptbl (cons list (map group (lambda (col) (concat col)))) '(list '((quote extract_assoc) (quote item) '((quote lambda) '((quote key) (quote value)) (quote value)))) true true)))
							(build_queryplan schema tables (merge (map group (lambda (expr) '((concat expr) expr)))) condition nil nil nil nil nil) /* INSERT INTO grouptbl SELECT group-attributes FROM tbl */
						)
					)

					/* compute aggregates */
					(map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
						(set cols (extract_columns_for_tblvar tblvar expr))
						/* TODO: name that column (concat ag "|" condition) */
						'((quote createcolumn) schema grouptbl (concat ag) "any" '(list) "" '((quote lambda) (map group (lambda (col) (symbol (concat col))))
							/* TODO: recurse build_queryplan? */
							(scan_wrapper schema tbl
								(cons list tblvar_cols)
								/* TODO: AND WHERE */
								'((quote lambda) (map tblvar_cols (lambda (col) (symbol (concat tblvar "." col)))) (cons (quote and) (map group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (concat col))))))))
								(cons list cols)
								'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr expr))/* TODO: (build_scan tables condition)*/
								reduce
								neutral
							)
						))
					))))

					/* scan preaggregate (TODO: recurse over build_queryplan with group=nil over the preagg table) */
					/* TODO: build_queryplan with order limit offset */
					'((scan_wrapper schema grouptbl
						/* HAVING */
						(cons list filtercols)
						'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr rest_condition)) /* TODO: filter count|condition > 0 */
						(cons list (merge
							/* group columns */
							(map group (lambda (col) (concat col)))
							/* aggregates */
							(map ags (lambda (ag) (concat ag)))
						))
						'((quote lambda) (merge
							/* group columns */
							(map group (lambda (col) (symbol (concat col))))
							/* aggregates */
							(map ags (lambda (ag) (symbol (concat ag))))
						) '((quote resultrow)
							(cons (quote list) (map_assoc fields (lambda (col expr) (replace_columns_agg_expr expr))))
						))
					))
				)
			)
			(error "Grouping and aggregates on joined tables is not implemented yet (prejoins)") /* TODO: construct grouptbl as join */
		)
	) (begin
		/* grouping has been removed; now to the real data: */

		(if (coalesce order /*limit offset*/) (begin
			/* ordered or limited scan */
			/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
			(match order
				'('('((symbol get_column) tblalias "ORDINAL_POSITION") direction)) (build_queryplan schema tables fields condition group having nil nil nil) /* ignore ordering for some cases by now to use the dbeaver tool */
				(error "Ordered scan is not implemented yet")
			)
		) (begin
			/* unordered unlimited scan */

			/* TODO: sort tables according to join plan */
			/* TODO: match tbl to inner query vs string */
			(define build_scan (lambda (tables condition)
				(match tables
					(cons '(tblvar schema tbl) tables) (begin /* outer scan */
						(set cols (merge_unique
							    (merge_unique (extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v))))
							    (extract_columns_for_tblvar tblvar condition)
						))
						/* TODO: split condition in those ANDs that still contain get_column from tables and those evaluatable now */
						(set rest_condition (match tables '() (coalesce condition true) true))
						(set filtercols (extract_columns_for_tblvar tblvar rest_condition))
						/* TODO: add columns from rest condition into cols list */

						(scan_wrapper schema tbl
							/* condition */
							(cons list filtercols)
							'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr rest_condition))
							/* extract columns and store them into variables */
							(cons list cols)
							'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables condition))
						)
					)
					'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v)))))
				)
			))
			(build_scan tables (replace_find_column condition))
		))
	))
)))
