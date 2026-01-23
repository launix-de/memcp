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
- (build_queryplan schema tables fields condition groups schemas) builds a lisp expression that runs the query and calls resultrow for each result tuple
- (build_scan schema tables cols map reduce neutral neutral2 condition groups) builds a lisp expression that scans the tables
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
	'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (error (concat "column " col " not found")) (symbol (concat tblvar "." col)))
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

/* helper to check list membership */
(define list_contains (lambda (lst item) (reduce lst (lambda (acc x) (or acc (equal? x item))) false)))

/* helper to collect all column references in an expression */
(define collect_all_column_refs (lambda (expr) (match expr
	'((symbol get_column) tblvar _ col _) (list (list tblvar col))
	(cons sym args) (merge_unique (map args collect_all_column_refs))
	'()
)))

(import "sql-metadata.scm")

/* recursively preprocess a query and return the flattened query. The returned parameterset will be passed to build_queryplan */
(define untangle_query (lambda (schema tables fields condition group having order limit offset) (begin
	/* TODO: unnest arbitrary queries -> turn them into a left join limit 1 */
	/* TODO: when FROM: spill tables and conditions into main query but rename all tables and columns with a prepended rename_prefix
	/* TODO: multiple group levels, limit+offset for each group level */
	(set rename_prefix (coalesce rename_prefix ""))
	(define make_group_stage (lambda (group having order limit offset)
		(list
			(cons (quote group-cols) (coalesce group '()))
			(list (quote having) having)
			(list (quote order) (coalesce order '()))
			(list (quote limit) limit)
			(list (quote offset) offset)
		)
	))
	(define stage_group_cols (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol group-cols) cols) cols
			_ nil
		) acc)
	) nil)))
	(define stage_having_expr (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol having) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))
	(define stage_limit_val (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol limit) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))
	(define stage_offset_val (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol offset) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))

	/* TODO(memcp): Unnesting strategy
	- Untangle/flatten FROM-subselects first (alias prefixing + column rewrite).
	- Implement inner_select{,_in,_exists} as plan-level joins using map/reduce + isOuter.
	- Add free_vars(expr) with schema-backed resolution (based on replace_find_column).
	- Replace order/limit params with group-stages to support per-group order/limit.
	*/

	/* check if we have FROM selects -> returns '(tables renamelist condition schemas) */
	(match (zip (map tables (lambda (tbldesc) (match tbldesc
		'(alias schema (string? tbl) _ _) '('(tbldesc) '() true '(alias (get_schema schema tbl))) /* leave primary tables as is and load their schema definition */
		'(id schemax subquery _ _) (match (apply untangle_query subquery) '(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2) (begin
			/* prefix all table aliases */	
			(set tablesPrefixed (map tables2 (lambda (x) (match x '(alias schema tbl a b) '((concat id ":" alias) schema tbl a b)))))
			/* helper function add prefix to tblalias of every expression */
			(define replace_column_alias (lambda (expr) (match expr
				'((symbol get_column) nil ti col ci) (begin
					/* resolve unqualified column against inner schemas2; must match exactly one table */
					(define matches (reduce_assoc schemas2 (lambda (acc alias cols)
						(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)
							(cons alias acc)
							acc)) '()))
					(match matches
						(cons only '()) '('get_column (concat id ":" only) ti col ci)
						'() (error (concat "column " col " does not exist in subquery"))
						(cons _ _) (error (concat "ambiguous column " col " in subquery"))
					)
				)
				'((symbol get_column) alias_ ti col ci) '('get_column (concat id ":" alias_) ti col ci)
				(cons sym args) /* function call */ (cons sym (map args replace_column_alias))
				expr
			)))
			/* TODO: group+order+limit+offset -> ordered scan list with aggregation layers (to avoid materialization) */
			(if (and (not (nil? groups2)) (not (equal? groups2 '())))
				(begin
					(define unsupported (reduce groups2 (lambda (acc stage)
						(or acc
							(begin
								(define g (stage_group_cols stage))
								(and (not (nil? g)) (not (equal? g '())))
							)
							(not (nil? (stage_having_expr stage)))
							(not (nil? (stage_limit_val stage)))
							(not (nil? (stage_offset_val stage)))
						)
					) false))
					(if unsupported (error "group/order/limit is not supported yet in subqueries"))
					(set groups2 nil)
				)
			)
			'(tablesPrefixed '(id (map_assoc fields2 (lambda (k v) (replace_column_alias v)))) (replace_column_alias condition2) (merge '(alias (extract_assoc fields2 (lambda (k v) '("Field" k "Type" "any")))) (merge (extract_assoc schemas2 (lambda (k v) '((concat id ":" k) v))))))
		) (error "non matching return value for untangle_query"))
		(error (concat "unknown tabledesc: " tbldesc))
	))))
		'(tablesList renameList conditionList schemasList) (begin /* schemas is an assoc array from alias -> columnlist */
			/* rewrite a flat table list according to inner selects */
			(set renamelist (merge renameList))
			(set tables (merge tablesList))
			(set schemas (merge schemasList))
			/*(print "tables=" tables)*/
			/*(print "schemas=" schemas)*/

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
				/* Ensure MySQL LIKE uses a collation at compile time:
				- If lhs is a text column, take collation from schema metadata.
				- Otherwise default to utf8mb4_general_ci (MySQL default in this project). */
				'((symbol strlike) a b c) (begin
					(define default_collation "utf8mb4_general_ci")
					(define find_column_collation (lambda (tblalias colname) (begin
						(define cols (schemas tblalias))
						(define coldef (reduce cols (lambda (a coldef)
							(if (or a (equal?? (coldef "Field") colname)) a coldef)
						) nil))
						(coalesce (and coldef (coldef "Collation")) default_collation)
					)))
					(match a
						'((symbol get_column) nil _ col ci)
						(cons (quote strlike)
							(cons
								(replace_find_column a)
								(cons (replace_find_column b) (cons default_collation '()))))
						'((symbol get_column) alias_ ti col ci)
						(begin
							(define resolved
								(coalesce
									(reduce_assoc schemas (lambda (a alias cols)
										(if (and ((if ti equal?? equal?) alias_ alias)
											(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
											alias
											a)
									) nil)
									alias_))
							(cons (quote strlike)
								(cons
									(replace_find_column a)
									(cons
										(replace_find_column b)
										(cons
											(if (equal?? c default_collation) (find_column_collation resolved col) c)
											'())))))
						_
						(cons (quote strlike)
							(cons (replace_find_column a) (cons (replace_find_column b) (cons c '()))))
					)
				)
				'((symbol get_column) nil _ col ci) '((quote get_column) (coalesce (reduce_assoc schemas (lambda (a alias cols) (if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false) alias a)) nil) (error (concat "column " col " does not exist in tables"))) false col false)
				'((symbol get_column) alias_ ti col ci) (if (or ti ci) '((quote get_column) (coalesce (reduce_assoc schemas (lambda (a alias cols) (if (and ((if ti equal?? equal?) alias_ alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)) alias a)) nil) (error (concat "column " alias_ "." col " does not exist in tables"))) false col false) expr) /* omit false false, otherwise freshly created columns wont be found */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))

			/* apply renamelist (assoc of assoc of expr) */
			(define replace_rename (lambda (expr) (match expr
				'((symbol get_column) alias_ ti col ci) (if (nil? alias_)
					/* no tblalias -> search the field in all tables */
					(reduce_assoc renamelist (lambda (a k v) (coalesce (v col) a)) expr)
					/* tblalias -> look up the field */
					(begin
						(define alias_str (string alias_))
						(define alias_sym (symbol alias_str))
						(define rename_fn (if (has_assoc? renamelist alias_)
							(renamelist alias_)
							(if (has_assoc? renamelist alias_str)
								(renamelist alias_str)
								(if (has_assoc? renamelist alias_sym)
									(renamelist alias_sym)
									nil))))
						(if (nil? rename_fn) expr (rename_fn col))
					)
				)
				(cons sym args) /* function call */ (cons sym (map args replace_rename))
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
			/* Append inner conditions (from derived tables) while avoiding nil short-circuit. */
			(set conditionAll (cons 'and (filter (cons (replace_rename condition) conditionList) (lambda (x) (not (nil? x)))))) /* TODO: append inner conditions to condition */
			(set group (map group replace_rename))
			(set having (replace_rename having))
			(set order (map order (lambda (o) (match o '(col dir) '((replace_rename col) dir)))))
			(set groups (if (or group having order limit offset) (list (make_group_stage group having order limit offset)) nil))
			'(schema tables (map_assoc fields (lambda (k v) (replace_rename v))) conditionAll groups schemas replace_find_column)
		)
		/* else: empty tables list */
		(begin
			(set groups (if (or group having order limit offset) (list (make_group_stage group having order limit offset)) nil))
			'(schema tables fields (optimize condition) groups '() (lambda (expr) expr))
		)
	)
)))

/* build queryplan from parsed query */
(define build_queryplan (lambda (schema tables fields condition groups schemas replace_find_column) (begin
	/* tables: '('(alias schema tbl isOuter joinexpr) ...), tbl might be string or '(schema tables fields condition groups) */
	/* fields: '(colname expr ...) (colname=* -> SELECT *) */
	/* expressions will use (get_column tblvar ti col ci) for reading from columns. we have to replace it with the correct variable */
	/*(print "build queryplan " '(schema tables fields condition groups schemas))*/
	/*
	Query builder masterplan:
	1. make sure all optimizations are done (unnesting arbitrary queries, leave just one big table list with fields, conditions, and group-stages)
	2. process group-stages: split queryplan into filling grouped table(s) and scanning them
	3. order/limit stages become ordered scans on the current table set
	4. scan the rest of the tables

	*/

	/* TODO: order tables: outer joins behind */
	(define make_group_stage (lambda (group having order limit offset)
		(list
			(cons (quote group-cols) (coalesce group '()))
			(list (quote having) having)
			(list (quote order) (coalesce order '()))
			(list (quote limit) limit)
			(list (quote offset) offset)
		)
	))
	(define stage_group_cols (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol group-cols) cols) cols
			_ nil
		) acc)
	) nil)))
	(define stage_having_expr (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol having) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))
	(define stage_order_list (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol order) rest) (if (nil? rest) '() (car rest))
			_ nil
		) acc)
	) nil)))
	(define stage_limit_val (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol limit) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))
	(define stage_offset_val (lambda (stage) (reduce stage (lambda (acc item)
		(if (nil? acc) (match item
			(cons (symbol offset) rest) (if (nil? rest) nil (car rest))
			_ nil
		) acc)
	) nil)))
	(set groups (coalesceNil groups '()))
	(define groups_present (and (not (nil? groups)) (not (equal? groups '()))))
	(define stage (if groups_present (car groups) nil))
	(define rest_groups (if groups_present (cdr groups) nil))
	(set rest_groups (coalesceNil rest_groups '()))
	(define stage_group (if stage (stage_group_cols stage) nil))
	(define stage_having (if stage (stage_having_expr stage) nil))
	(define stage_order (if stage (stage_order_list stage) nil))
	(define stage_limit (if stage (stage_limit_val stage) nil))
	(define stage_offset (if stage (stage_offset_val stage) nil))

	(if stage_group (begin
		/* group: extract aggregate clauses and split the query into two parts: gathering the aggregates and outputting them */
		(set stage_group (map stage_group replace_find_column))
		(set stage_having (replace_find_column stage_having))
		(set stage_order (map stage_order (lambda (o) (match o '(col dir) '((replace_find_column col) dir)))))
		(define ags (merge_unique (extract_assoc fields (lambda (key expr) (extract_aggregates expr))))) /* aggregates in fields */
		(define ags (merge_unique ags (merge_unique (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (extract_aggregates col))))))) /* aggregates in order */
		(define ags (merge_unique ags (extract_aggregates (coalesce stage_having true)))) /* aggregates in having */

		/* TODO: replace (get_column nil ti col ci) in group, having and order with (coalesce (fields col) '('get_column nil false col false)) */

		(match tables
			/* TODO: allow for more than just group by single table */
			/* TODO: outer tables that only join on group */
			'('(tblvar schema tbl isOuter _)) (begin
				/* prepare preaggregate */

				/* TODO: check if there is a foreign key on tbl.groupcol and then reuse that table */
				(set grouptbl (concat "." tbl ":" stage_group))
				(createtable schema grouptbl (cons
					/* unique key over all identiying columns */ '("unique" "group" (map stage_group (lambda (col) (concat col))))
					/* all identifying columns */ (map stage_group (lambda (col) '("column" (concat col) "any"/* TODO get type from schema */ '() '())))
				) '("engine" "sloppy") true)

				/* prepare a fitting repartitioning for that table from the beginning: copy parititioning schema from the source tbl */
				(partitiontable schema grouptbl (merge (map stage_group (lambda (col) (match col '('get_column (eval tblvar) false scol false) '((concat col) (shardcolumn schema tbl scol)) '())))))

				/* preparation */
				(define tblvar_cols (merge_unique (map stage_group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesceNil condition true)))
				(set filtercols (extract_columns_for_tblvar tblvar condition))

				(define replace_agg_with_fetch (lambda (expr) (match expr
					(cons (symbol aggregate) rest) '('get_column grouptbl false (concat rest "|" condition) false) /* aggregate helper column */
					'((symbol get_column) tblvar ti col ci) '('get_column grouptbl ti (concat '('get_column tblvar ti col ci)) ci) /* grouped col */
					(cons sym args) /* function call */ (cons sym (map args replace_agg_with_fetch))
					expr /* literals */
				)))

				(define grouped_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (replace_agg_with_fetch col) dir))))))
				(define next_groups (merge
					(if (coalesce grouped_order stage_limit stage_offset) (list (make_group_stage nil nil grouped_order stage_limit stage_offset)) '())
					rest_groups
				))
				(define grouped_plan (build_queryplan schema '('(grouptbl schema grouptbl false nil))
					(map_assoc fields (lambda (k v) (replace_agg_with_fetch v)))
					(replace_agg_with_fetch stage_having)
					next_groups
					schemas
					replace_find_column))

				(define collect_plan
					'('time '('begin
						/* If grouping is global (group='(1)), avoid base scan and insert one key row */
						(if (equal? stage_group '(1))
							'('insert schema grouptbl '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
							(begin
								/* key columns */
								(set keycols (merge_unique (map stage_group (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
								(scan_wrapper 'scan schema tbl
									(cons list filtercols)
									'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
									(cons list keycols)
									'((quote lambda)
										(map keycols (lambda (col) (symbol (concat tblvar "." col))))
										(cons (quote list) (map stage_group (lambda (expr) (replace_columns_from_expr expr))))) /* build records '(k1 k2 ...) */
									'((quote lambda) '('acc 'rowvals) '('set_assoc 'acc 'rowvals true)) /* add keys to assoc; each key is a dataset -> unique filtering */
									'(list) /* empty dict */
									'((quote lambda) '('acc 'sharddict)
										'('insert
											schema grouptbl
											(cons 'list (map stage_group (lambda (col) (concat col))))
											'('extract_assoc 'sharddict '('lambda '('k 'v) 'k)) /* turn keys from assoc into list */
											'(list) '('lambda '() true) true)
									)
									isOuter)
							)
						)
					) "collect"))

				(define compute_plan
					'('time (cons 'parallel (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
						(set cols (extract_columns_for_tblvar tblvar expr))
						/* TODO: name that column (concat ag "|" condition) */
						'((quote createcolumn) schema grouptbl (concat ag "|" condition) "any" '(list) '(list "temp" true) (cons list (map stage_group (lambda (col) (concat col)))) '((quote lambda) (map stage_group (lambda (col) (symbol (concat col))))
							(scan_wrapper 'scan schema tbl
								(cons list (merge tblvar_cols filtercols))
								/* check group equality AND WHERE-condition */
								'((quote lambda) (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons (quote and) (cons (replace_columns_from_expr condition) (map stage_group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (concat col))))))))))
								(cons list cols)
								'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr expr))/* TODO: (build_scan tables condition)*/
								reduce
								neutral
								nil
								isOuter
							)
						))
					))))) "compute"))

				(list 'begin collect_plan compute_plan grouped_plan)
			)
			(error "Grouping and aggregates on joined tables is not implemented yet (prejoins)") /* TODO: construct grouptbl as join */
		)
	) (begin
			/* grouping has been removed; now to the real data: */
			(if (and (not (nil? rest_groups)) (not (equal? rest_groups '()))) (error "non-group stage must be last"))
			(if (coalesce stage_order stage_limit stage_offset) (begin
				/* ordered or limited scan */
				/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
				(set stage_order (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) '((replace_find_column col) dir)))))
				(define build_scan (lambda (tables condition)
					(match tables
						(cons '(tblvar schema tbl isOuter _) tables) (begin /* outer scan */
							(set cols (merge_unique
								(merge_unique (extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v))))
								(extract_columns_for_tblvar tblvar condition)
							))
							(match (split_condition (coalesceNil condition true) tables) '(now_condition later_condition) (begin
								(set filtercols (extract_columns_for_tblvar tblvar now_condition))
								/* TODO: add columns from rest condition into cols list */

								/* extract order cols for this tblvar */
								/* TODO: match case insensitive column */
								/* TODO: non-trivial columns to computed columns */
								/* preserve ORDER BY key order (first key has highest priority) */
								(set ordercols (merge (map stage_order (lambda (o) (match o '('((symbol get_column) (eval tblvar) _ col _) dir) (list col) '())))))
								(set dirs      (merge (map stage_order (lambda (o) (match o '('((symbol get_column) (eval tblvar) _ col _) dir) (list dir) '())))))

								(scan_wrapper 'scan_order schema tbl
									/* condition */
									(cons list filtercols)
									'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
									/* sortcols, sortdirs */
									(cons list ordercols)
									(cons list dirs)
									stage_offset
									(coalesceNil stage_limit -1)
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
								(match (split_condition (coalesceNil condition true) tables) '(now_condition later_condition) (begin
									(set filtercols (extract_columns_for_tblvar tblvar now_condition))

									(scan_wrapper 'scan schema tbl
										/* condition */
										(cons list filtercols)
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
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
							'() /* final inner (=scalar) */ '('if (coalesceNil condition true) '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v))))))
						)
					))
					(build_scan tables (replace_find_column condition))
			))
	))
)))
