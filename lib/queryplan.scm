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
 - (build_queryplan schema tables fields condition group having order limit offset schemas) builds a lisp expression that runs the query and calls resultrow for each result tuple
 - (build_scan schema tables cols map reduce neutral neutral2 condition group having order limit offset) builds a lisp expression that scans the tables
 - (extract_columns_for_tblvar expr tblvar) extracts a list of used columns for each tblvar '(tblvar col)
 - (replace_columns expr) replaces all (get_column ...) and (aggregate ...) with values

*/

/* returns a list of '(string...) */
(define extract_columns_for_tblvar (lambda (tblvar expr) (match expr
	'((symbol get_column) (eval tblvar) _ col _) '(col) /* TODO: case matching */
	(cons sym args) /* function call */ (merge_unique (map args (lambda (arg) (extract_columns_for_tblvar tblvar arg))))
	'()
)))

/* changes (get_column tblvar ti col ci) into its symbol */
(define replace_columns_from_expr (lambda (expr) (match expr
	(cons (symbol aggregate) args) /* aggregates: don't dive in */ (cons aggregate args)
	'((symbol get_column) tblvar _ col _) (symbol (concat tblvar "." col)) /* TODO: case insensitive matching */
	(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
	expr /* literals */
)))

/* returns a list of all aggregates in this expr */
(define extract_aggregates (lambda (expr) (match expr
	(cons (symbol aggregate) args) '(args)
	(cons sym args) /* function call */ (merge (map args extract_aggregates))
	/* literal */ '()
)))

/* split_condition returns a tuple (now, later) according to what can be checked now and what has to be waited for tables '('(tblvar ...) ...) */
(define split_condition (lambda (expr tables) (match expr
	'((symbol get_column) tblvar _ col _) /* a column */ (match tables
		'() '(expr true) /* last condition: compute now */
		(cons (cons (eval tblvar) _) _) '(true expr) /* col depends on tblvar */
		(cons _ tablesrest) (split_condition expr tablesrest) /* check next table in join plan */
		(error "invalid tables list")
	)
	(cons (symbol and) conditions) /* splittable and */ (split_condition_and conditions tables)
	(cons sym args) /* non-splittable function call */ (split_condition_combine sym args tables)
	/* literal */ '(expr true)
)))
(define split_condition_combine (lambda (sym args tables) (if
	(reduce args (lambda (other arg) (match (split_condition arg tables) '(_ true) other true)) false) /* if one of the args is later, everything is later */
	'(true (cons sym args))
	'((cons sym args) true)
)))
(define split_condition_and (lambda (l tables) (match l
	'() '(true true)
	(cons head tail) (match '((split_condition head tables) (split_condition_and tail tables))
		'('(true true) '(x y)) '(x y)
		'('(true y) '(x true)) '(x y)
		'('(x true) '(true y)) '(x y)
		'('(x y) '(true true)) '(x y)
		'('(x1 y) '(x2 true)) '('('and x1 x2) y)
		'('(x1 true) '(x2 y)) '('('and x1 x2) y)
		'('(true y1) '(x y2)) '(x '('and y1 y2))
		'('(x y1) '(true y2)) '(x '('and y1 y2))
		'('(x1 y1) '(x2 y2)) '('('and x1 x2) '('and y1 y2))
	)
)))

(import "sql-metadata.scm")

/* recursively preprocess a query and return the flattened query */
(define untangle_query (lambda (schema tables fields condition group having order limit offset) (begin
	/* TODO: unnest arbitrary queries -> turn them into a left join limit 1 */
	/* TODO: when FROM: spill tables and conditions into main query but rename all tables and columns with a prepended rename_prefix
	/* TODO: multiple group levels, limit+offset for each group level */
	(set rename_prefix (coalesce rename_prefix ""))

	/* check if we have FROM selects -> returns '(tables renamelist) */
	(match (zip (map tables (lambda (tbldesc) (match tbldesc
		'(alias schema (string? tbl) _ _) '('(tbldesc) '() '(alias (get_schema schema tbl))) /* leave primary tables as is and load their schema definition */
		'(id schemax subquery _ _) (match (apply untangle_query subquery) '(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2 schemas2 replace_find_column2) (begin
			/* TODO: helper function add prefix to tblalias of every expression */
			/* TODO: integrate tables into main query and add fields to a renamelist */
			/* TODO: tables -> rename with prefix */
			/* TODO: fields -> add to renamelist + rename with prefix */
			/* TODO: condition -> add to main condition list + rename with prefix */
			/* TODO: group+order+limit+offset -> ordered scan list with aggregation layers */
			(print "TODO: " '(schema2 tables2 fields2 condition2 group2 order2 limit2 offset2 schemas2))
			'(tables2 '(id fields2) '(alias (extract_assoc fields2 (lambda (k v) '("Field" k "Type" "any")))))
		) (error "non matching return value for untangle_query"))
		(error (concat "unknown tabledesc: " tbldesc))
	))))
	'(tablesList renamelist schemasList) (begin /* schemas is an assoc array from alias -> columnlist */
		/* rewrite a flat table list according to inner selects */
		(print "renamelist = " (merge renamelist))
		(set tables (merge tablesList))
		(set schemas (merge schemasList))

		/* TODO: add rename_prefix to all table names and get_column expressions */
		/* TODO: apply renamelist to all expressions in fields condition group having order */

		/* at first: extract additional join exprs into condition list */
		(set condition (cons 'and (coalesce (filter (append (map tables (lambda (t) (match t '(alias schema tbl isOuter joinexpr) joinexpr nil))) condition) (lambda (x) (not (nil? x)))) true)))

		/* tells whether there is an aggregate inside */
		(define expr_find_aggregate (lambda (expr) (match expr
			'((symbol aggregate) item reduce neutral) true
			(cons sym args) /* function call */ (reduce args (lambda (a b) (or a (expr_find_aggregate b))) false)
			false
		)))

		/* set group to 1 if fields contain aggregates even if not */
		(define group (coalesce group (if (reduce_assoc fields (lambda (a key v) (or a (expr_find_aggregate v))) false) '(1) nil)))

		/* find those columns that have no table */
		(define replace_find_column (lambda (expr) (match expr
			'((symbol get_column) nil _ col ci) '((quote get_column) (reduce_assoc schemas (lambda (a alias cols) (if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false) alias a)) (lambda () (error (concat "column " col " does not exist in tables")))) false col false)
			'((symbol get_column) alias_ ti col ci) (if (or ti ci) '((quote get_column) (reduce_assoc schemas (lambda (a alias cols) (if (and ((if ti equal?? equal?) alias_ alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)) alias a)) (lambda () (error (concat "column " alias "." col " does not exist in tables")))) false col false) expr) /* omit false false), otherwise freshly created columns wont be found */
			(cons sym args) /* function call */ (cons sym (map args replace_find_column))
			expr
		)))

		/* expand *-columns */
		(set fields (merge (extract_assoc fields (lambda (col expr) (match col
			"*" (match expr
				/* *.* */
				'((symbol get_column) nil _ "*" _) (merge (extract_assoc schemas (lambda (alias def) (merge (map def (lambda (coldesc) /* all columns of each table */
						'((coldesc "Field") '((quote get_column) alias false (coldesc "Field") false))
					)))
				)))
				/* tbl.* */
				'((symbol get_column) tblvar ignorecase "*" _) (merge (extract_assoc schemas (lambda (alias def) (if ((if ignorecase equal?? equal?) alias tblvar) (merge (map def (lambda (coldesc) /* all columns of each table */
						'((coldesc "Field") '((quote get_column) alias false (coldesc "Field") false))
					))) '())
				)))
			)
			'(col (replace_find_column expr))
		)))))

		/* return parameter list for build_queryplan */
		'(schema tables fields condition group having order limit offset schemas replace_find_column)
	))
)))

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields condition group having order limit offset schemas replace_find_column) (begin
	/* tables: '('(alias schema tbl isOuter joinexpr) ...), tbl might be string or '(schema tables fields condition group having order limit offset) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* expressions will use (get_column tblvar ti col ci) for reading from columns. we have to replace it with the correct variable */

	/*
		Query builder masterplan:
		1. make sure all optimizations are done (unnesting arbitrary queries, leave just one big table list with a field list, conditions, as well as a order+limit+offset)
		2. if group is present: split the queryplan into filling the grouped table and scanning it -> find or create the preaggregate table, scan over the preaggregate
		3. if order+limit+offset is present: split the queryplan into providing a scannable tableset and a ordered scan on that tableset
		   -> find or create all tables that have to be nestedly scanned. if two tables are clumsed together, create a prejoin. recurse over build_queryplan without the order clause.
		4. scan the rest of the tables

	*/

  	/* TODO: order tables: outer joins behind */

	(if group (begin
		/* group: extract aggregate clauses and split the query into two parts: gathering the aggregates and outputting them */
		(set group (map group replace_find_column))
		(set having (replace_find_column having))
		(set order (map order (lambda (o) (match o '(col dir) '((replace_find_column col) dir)))))
		(define ags (merge_unique (extract_assoc fields (lambda (key expr) (extract_aggregates expr))))) /* aggregates in fields */
		(define ags (merge_unique ags (merge_unique (map (coalesce order '()) (lambda (x) (match x '(col dir) (extract_aggregates col))))))) /* aggregates in order */
		(define ags (merge_unique ags (extract_aggregates (coalesce having true)))) /* aggregates in having */

		/* TODO: replace (get_column nil ti col ci) in group, having and order with (coalesce (fields col) '('get_column nil false col false)) */

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

				/* prepare a fitting repartitioning for that table from the beginning: copy parititioning schema from the source tbl */
				(partitiontable schema grouptbl (merge (map group (lambda (col) (match col '('get_column (eval tblvar) false scol false) '((concat col) (shardcolumn schema tbl scol)) '())))))

				/* preparation */
				(define tblvar_cols (merge_unique (map group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesce condition true)))
				(set filtercols (extract_columns_for_tblvar tblvar condition))

				(define replace_agg_with_fetch (lambda (expr) (match expr
					(cons (symbol aggregate) rest) '('get_column grouptbl false (concat rest "|" condition) false) /* aggregate helper column */
					'((symbol get_column) tblvar ti col ci) '('get_column grouptbl ti (concat '('get_column tblvar ti col ci)) ci) /* grouped col */
					(cons sym args) /* function call */ (cons sym (map args replace_agg_with_fetch))
					expr /* literals */
				)))

				'('begin
					/* TODO: partitioning hint for insert -> same partitioning scheme as tables */
					/* INSERT IGNORE group cols into preaggregate */
					/* TODO: use bulk insert in scan reduce phase (and filter duplicates from a bulk!) */
					'('time '('begin
						/* the optimizer will optimize and group this */
						'('set 'resultrow '('lambda '('item) '('insert schema grouptbl (cons list (map group (lambda (col) (concat col)))) '(list '('extract_assoc 'item '('lambda '('key 'value) 'value))) '(list) '('lambda '() true) true)))
						(if (equal? group '(1)) '('resultrow '(list "1" 1)) (build_queryplan schema tables (merge (map group (lambda (expr) '((concat expr) expr)))) condition nil nil nil nil nil schemas replace_find_column)) /* INSERT INTO grouptbl SELECT group-attributes FROM tbl */
					) "collect")

					/* compute aggregates */
					'('time (cons 'parallel (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
						(set cols (extract_columns_for_tblvar tblvar expr))
						/* TODO: name that column (concat ag "|" condition) */
						'((quote createcolumn) schema grouptbl (concat ag "|" condition) "any" '(list) "" (cons list (map group (lambda (col) (concat col)))) '((quote lambda) (map group (lambda (col) (symbol (concat col))))
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
					))))) "compute")

					/* build the queryplan for the ordered limited scan on the grouped table */
					(build_queryplan schema '('(grouptbl schema grouptbl false nil)) (map_assoc fields (lambda (k v) (replace_agg_with_fetch v))) (replace_agg_with_fetch having) nil nil (if (nil? order) nil (map order (lambda (o) (match o '(col dir) '((replace_agg_with_fetch col) dir))))) limit offset schemas replace_find_column)
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
						(match (split_condition (coalesce condition true) tables) '(now_condition later_condition) (begin
							(set filtercols (extract_columns_for_tblvar tblvar now_condition))
							/* TODO: add columns from rest condition into cols list */

							/* extract order cols for this tblvar */
							/* TODO: match case insensitive column */
							(set ordercols (reduce order (lambda(a o) (match o '('((symbol get_column) (eval tblvar) _ col _) dir) (cons col a) a)) '()))
							(set dirs      (reduce order (lambda(a o) (match o '('((symbol get_column) (eval tblvar) _ col _) dir) (cons dir a) a)) '()))

							(scan_wrapper 'scan_order schema tbl
								/* condition */
								(cons list filtercols)
								'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr now_condition))
								/* sortcols, sortdirs */
								(cons list ordercols)
								(cons list dirs)
								offset
								(coalesce limit -1)
								/* extract columns and store them into variables */
								(cons list cols)
								'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables later_condition))
								/* no reduce+neutral */
								nil
								nil
								isOuter
							)
						))
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
						/* split condition in those ANDs that still contain get_column from tables and those evaluatable now */
						(match (split_condition (coalesce condition true) tables) '(now_condition later_condition) (begin
							(set filtercols (extract_columns_for_tblvar tblvar now_condition))

							(scan_wrapper 'scan schema tbl
								/* condition */
								(cons list filtercols)
								'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr now_condition))
								/* extract columns and store them into variables */
								(cons list cols)
								'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables later_condition))
								nil
								nil
								nil
								isOuter
							)
						))
					)
					'() /* final inner */ '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v)))))
				)
			))
			(build_scan tables (replace_find_column condition))
		))
	))
)))
