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

(import "sql-metadata.scm")

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields condition group having order limit offset) (begin
	/* tables: '('(alias schema tbl isOuter joinexpr) ...) */
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

	/* at first: extract additional join exprs into condition list */
	(set condition (cons 'and (filter (append (map tables (lambda (t) (match t '(alias schema tbl isOuter joinexpr) joinexpr nil))) condition) (lambda (x) (not (nil? x))))))

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
	(set schemas (merge (map tables (lambda (t) (match t '(alias schema tbl isOuter _) '(alias (get_schema schema tbl)))))))

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
			'((symbol get_column) nil "*")(merge (map tables (lambda (t) (match t '(alias schema tbl isOuter _) /* all FROM-tables*/
				(merge (map (get_schema schema tbl) (lambda (coldesc) /* all columns of each table */
					'((coldesc "name") '((quote get_column) alias (coldesc "name")))
				)))
			))))
			/* tbl.* */
			'((symbol get_column) tblvar "*")(merge (map tables (lambda (t) (match t '(alias schema tbl isOuter _) /* one FROM-table*/
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
		(define ags (merge_unique (extract_assoc fields (lambda (key expr) (extract_aggregates expr))))) /* aggregates in fields */
		(define ags (merge_unique ags (merge_unique (map (coalesce order '()) (lambda (x) (match x '(col dir) (extract_aggregates col))))))) /* aggregates in order */
		(define ags (merge_unique ags (extract_aggregates (coalesce having true)))) /* aggregates in having */

		/* TODO: replace (get_column nil col) in group, having and order with (coalesce (fields col) '('get_column nil col)) */

		(match tables
			/* TODO: allow for more than just group by single table */
			/* TODO: outer tables that only join on group */
			'('(tblvar schema tbl isOuter _)) (begin
				/* prepare preaggregate */

				/* TODO: check if there is a foreign key on tbl.groupcol and then reuse that table */
				(set grouptbl (concat "." tbl ":" group))
				(createtable schema grouptbl (cons
					/* unique key over all identiying columns */ '("unique" "group" (map group (lambda (col) (concat col))))
					/* all identifying columns */ (map group (lambda (col) '("column" (concat col) "any"/* TODO get type from schema */ '() '())))
				) '("engine" "sloppy") true)

				/* preparation */
				(define tblvar_cols (merge_unique (map group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesce condition true)))
				(set filtercols (extract_columns_for_tblvar tblvar condition))

				(define replace_agg_with_fetch (lambda (expr) (match expr
					(cons (symbol aggregate) rest) '('get_column grouptbl (concat rest "|" condition)) /* aggregate helper column */
					'((symbol get_column) tblvar col) '('get_column grouptbl (concat '('get_column tblvar col))) /* grouped col */
					(cons sym args) /* function call */ (cons sym (map args replace_agg_with_fetch))
					expr /* literals */
				)))

				(merge
					/* TODO: partitioning hint for insert -> same partitioning scheme as tables */
					'('begin
						/* INSERT IGNORE group cols into preaggregate */
						/* TODO: use bulk insert in scan reduce phase (and filter duplicates from a bulk!) */
						'('begin
							'('set 'resultrow '('lambda '('item) '('insert schema grouptbl (cons list (map group (lambda (col) (concat col)))) '(list '('extract_assoc 'item '('lambda '('key 'value) 'value))) true true)))
							(if (equal? group '(1)) '('resultrow '(list "1" 1)) (build_queryplan schema tables (merge (map group (lambda (expr) '((concat expr) expr)))) condition nil nil nil nil nil)) /* INSERT INTO grouptbl SELECT group-attributes FROM tbl */
						)
					)

					/* compute aggregates */
					(cons 'parallel (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
						(set cols (extract_columns_for_tblvar tblvar expr))
						/* TODO: name that column (concat ag "|" condition) */
						'((quote createcolumn) schema grouptbl (concat ag "|" condition) "any" '(list) "" '((quote lambda) (map group (lambda (col) (symbol (concat col))))
							(scan_wrapper 'scan schema tbl
								(cons list (merge tblvar_cols filtercols))
								/* check group equality AND WHERE-condition */
								'((quote lambda) (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (cons (quote and) (cons (replace_columns_from_expr condition) (map group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (concat col)))))))))
								(cons list cols)
								'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr expr))/* TODO: (build_scan tables condition)*/
								reduce
								neutral
								nil
								isOuter
							)
						))
					)))))

					/* build the queryplan for the ordered limited scan on the grouped table */
					'((build_queryplan schema '('(grouptbl schema grouptbl false)) (map_assoc fields (lambda (k v) (replace_agg_with_fetch v))) (replace_agg_with_fetch (replace_find_column having)) nil nil (if (nil? order) nil (map order (lambda (o) (match o '(col dir) '((replace_agg_with_fetch (replace_find_column col)) dir))))) limit offset))
				)
			)
			(error "Grouping and aggregates on joined tables is not implemented yet (prejoins)") /* TODO: construct grouptbl as join */
		)
	) (begin
		/* grouping has been removed; now to the real data: */

		(if (coalesce order limit offset) (begin
			/* ordered or limited scan */
			/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
			(set order (map (coalesce order '()) (lambda (x) (match x '(col dir) '((replace_find_column col) dir)))))
			(define build_scan (lambda (tables condition)
				(match tables
					(cons '(tblvar schema tbl isOuter _) tables) (begin /* outer scan */
						(set cols (merge_unique
							    (merge_unique (extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v))))
							    (extract_columns_for_tblvar tblvar condition)
						))
						/* TODO: split condition in those ANDs that still contain get_column from tables and those evaluatable now */
						(set rest_condition (match tables '() (coalesce condition true) true))
						(set filtercols (extract_columns_for_tblvar tblvar rest_condition))
						/* TODO: add columns from rest condition into cols list */

						/* extract order cols for this tblvar */
						(set ordercols (reduce order (lambda(a o) (match o '('((symbol get_column) (eval tblvar) col) dir) (cons col a) a)) '()))
						(set dirs      (reduce order (lambda(a o) (match o '('((symbol get_column) (eval tblvar) col) dir) (cons dir a) a)) '()))

						(scan_wrapper 'scan_order schema tbl
							/* condition */
							(cons list filtercols)
							'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr rest_condition))
							/* sortcols, sortdirs */
							(cons list ordercols)
							(cons list dirs)
							offset
							(coalesce limit -1)
							/* extract columns and store them into variables */
							(cons list cols)
							'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables condition))
							/* no reduce+neutral */
							nil
							nil
							isOuter
						)
					)
					'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v)))))
				)
			))
			(build_scan tables (replace_find_column condition))
		) (begin
			/* unordered unlimited scan */

			/* TODO: sort tables according to join plan */
			/* TODO: match tbl to inner query vs string */
			(define build_scan (lambda (tables condition)
				(match tables
					(cons '(tblvar schema tbl isOuter _) tables) (begin /* outer scan */
						(set cols (merge_unique
							    (merge_unique (extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v))))
							    (extract_columns_for_tblvar tblvar condition)
						))
						/* TODO: split condition in those ANDs that still contain get_column from tables and those evaluatable now */
						(set rest_condition (match tables '() (coalesce condition true) true))
						(set filtercols (extract_columns_for_tblvar tblvar rest_condition))
						/* TODO: add columns from rest condition into cols list */

						(scan_wrapper 'scan schema tbl
							/* condition */
							(cons list filtercols)
							'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr rest_condition))
							/* extract columns and store them into variables */
							(cons list cols)
							'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables condition))
							nil
							nil
							nil
							isOuter
						)
					)
					'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v)))))
				)
			))
			(build_scan tables (replace_find_column condition))
		))
	))
)))
