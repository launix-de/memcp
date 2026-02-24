/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Hänsch

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
(define extract_columns_for_tblvar (lambda (tblvar expr)
	(match expr
		'((symbol get_column) (eval tblvar) _ col _) (if (equal? col "*") '() '(col)) /* TODO: case matching */
		(cons sym args) /* function call */ (merge_unique (map args (lambda (arg) (extract_columns_for_tblvar tblvar arg))))
		'()
	)
))

/* changes (get_column tblvar ti col ci) into its symbol */
(define replace_columns_from_expr (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) /* aggregates: don't dive in */ (cons aggregate args)
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (error (concat "column " col " not found")) (symbol (concat tblvar "." col)))
		(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
		expr /* literals */
	)
))

/* scalar subselect helper wrappers */
(define scalar_scan (lambda (schema tbl filtercols filterfn mapcols mapfn reduce neutral reduce2) (begin
	(define result (scan schema tbl filtercols filterfn mapcols mapfn reduce neutral reduce2))
	(if (equal? result neutral) nil result)
)))
(define scalar_scan_order (lambda (schema tbl filtercols filterfn sortcols sortdirs offset limit mapcols mapfn reduce neutral) (begin
	(define result (scan_order schema tbl filtercols filterfn sortcols sortdirs offset limit mapcols mapfn reduce neutral))
	(if (equal? result neutral) nil result)
)))

/* returns a list of all aggregates in this expr */
(define extract_aggregates (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) '(args)
		(cons sym args) /* function call */ (merge (map args extract_aggregates))
		/* literal */ '()
	)
))

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

(define extract_outer_columns_for_tblvar (lambda (tblvar expr) (match expr
	(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)) (equal? sym '(symbol outer)))
		(match args
			'(symname) (begin
				(define parts (split (string symname) "."))
				(match parts
					(list tbl col) (if (equal?? tbl (string tblvar)) (list col) '())
					_ '()
				)
			)
			_ '()
		)
		(merge_unique (map args (lambda (arg) (extract_outer_columns_for_tblvar tblvar arg))))
	)
	'()
)))

(import "sql-metadata.scm")

/* group stage constructors and accessors - shared between untangle_query and build_queryplan */
(define make_group_stage (lambda (group having order limit offset)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) having)
		(list (quote order) (coalesce order '()))
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
	)
))
(define make_dedup_stage (lambda (group)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) nil)
		(list (quote order) '())
		(list (quote limit) nil)
		(list (quote offset) nil)
		(list (quote dedup) true)
	)
))
(define stage_group_cols (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote group-cols) cols) cols
		_ nil
	) acc)
) nil)))
(define stage_having_expr (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote having) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_order_list (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote order) rest) (if (nil? rest) '() (car rest))
		_ nil
	) acc)
) nil)))
(define stage_limit_val (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote limit) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_offset_val (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote offset) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_is_dedup (lambda (stage) (reduce stage (lambda (acc item)
	(if acc acc (match item
		'((quote dedup) true) true
		_ false
	))
) false)))

/* query term helpers */
(define query_union_all_parts (lambda (query) (match query
	'(union_all branches order limit offset) (list branches order limit offset)
	'((symbol union_all) branches order limit offset) (list branches order limit offset)
	'((quote union_all) branches order limit offset) (list branches order limit offset)
	_ nil
)))
(define query_is_select_core (lambda (query) (and (list? query) (>= (count query) 9))))
(define query_branch_field_names (lambda (query) (match query
	'(schema tables fields condition group having order limit offset) (extract_assoc fields (lambda (k v) k))
	_ '()
)))

/* make_keytable: create a canonically named group/key table with sloppy engine
Returns the keytable name. Creates table + partitioning as side effect.
condition_suffix: if non-nil, appended to name (for dedup stages with WHERE) */
(define make_keytable (lambda (schema tbl keys tblvar condition_suffix) (begin
	(define keytable_name (if (nil? condition_suffix)
		(concat "." tbl ":" keys)
		(concat "." tbl ":" keys "|" condition_suffix)))
	(createtable schema keytable_name (cons
		'("unique" "group" (map keys (lambda (col) (concat col))))
		(map keys (lambda (col) '("column" (concat col) "any" '() '())))
	) '("engine" "sloppy") true)
	(partitiontable schema keytable_name (merge (map keys (lambda (col) (match col '('get_column (eval tblvar) false scol false) '((concat col) (shardcolumn schema tbl scol)) '())))))
	keytable_name
)))

/* make_col_replacer: create a function that rewrites column/aggregate references to point at a group table
is_dedup=true: leave aggregates intact (for dedup stages)
is_dedup=false: replace aggregates with column fetches (for normal group stages) */
(define make_col_replacer (lambda (grouptbl condition is_dedup) (begin
	(define replacer (lambda (expr) (match expr
		(cons (symbol aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (concat rest "|" condition) false))
		'((symbol get_column) tblvar ti col ci) '('get_column grouptbl ti (concat '('get_column tblvar ti col ci)) ci)
		(cons sym args) (cons sym (map args replacer))
		expr
	)))
	replacer
)))

/* recursively preprocess a query and return the flattened query. The returned parameterset will be passed to build_queryplan */
(define untangle_query (lambda (schema tables fields condition group having order limit offset) (begin
	/* TODO: unnest arbitrary queries -> turn them into a left join limit 1 */
	/* TODO: multiple group levels, limit+offset for each group level */
	(set rename_prefix (coalesce rename_prefix ""))

	/* COUNT(DISTINCT) rewrite helpers - do not descend into inner_select nodes (subqueries are processed separately) */
	(define _cd_is_subquery (lambda (sym) (match sym
		'inner_select true '(quote inner_select) true (symbol inner_select) true
		'inner_select_in true '(quote inner_select_in) true (symbol inner_select_in) true
		'inner_select_exists true '(quote inner_select_exists) true (symbol inner_select_exists) true
		_ false)))
	(define _cd_find (lambda (expr) (match expr
		'((symbol count_distinct) _) true
		(cons sym args) (if (_cd_is_subquery sym) false (reduce args (lambda (a b) (or a (_cd_find b))) false))
		false)))
	(define _cd_extract (lambda (expr) (match expr
		'((symbol count_distinct) e) (list e)
		(cons sym args) (if (_cd_is_subquery sym) '() (merge (map args _cd_extract)))
		'())))
	(define _cd_replace (lambda (expr) (match expr
		'((symbol count_distinct) e) '((quote aggregate) 1 (quote +) 0)
		(cons sym args) (if (_cd_is_subquery sym) expr (cons sym (map args _cd_replace)))
		expr)))
	(define _cd_has (reduce_assoc fields (lambda (a k v) (or a (_cd_find v))) false))
	/* if count_distinct present: save original having/order/limit/offset, replace fields,
	clear having/order/limit/offset (they belong to the outer/final group stage) */
	(define _cd_distinct_exprs (if _cd_has (reduce_assoc fields (lambda (a k v) (merge a (_cd_extract v))) '()) nil))
	(define _cd_having (if _cd_has having nil))
	(define _cd_order (if _cd_has order nil))
	(define _cd_limit (if _cd_has limit nil))
	(define _cd_offset (if _cd_has offset nil))
	(define _cd_user_group group)
	(define fields (if _cd_has (map_assoc fields (lambda (k v) (_cd_replace v))) fields))
	(define having (if _cd_has nil having))
	(define order (if _cd_has nil order))
	(define limit (if _cd_has nil limit))
	(define offset (if _cd_has nil offset))

	(define make_replace_find_column_subselect (lambda (schemas2 outer_schemas) (begin
		(define alias_exists_in_schema (lambda (schemas alias_name table_insensitive) (reduce_assoc schemas (lambda (acc alias cols)
			(or acc ((if table_insensitive equal?? equal?) alias_name alias))
		) false)))
		(define column_exists_in_schema (lambda (schemas alias_name table_insensitive column_name column_insensitive) (begin
			(define matches (reduce_assoc schemas (lambda (acc alias cols)
				(if (and (or (nil? alias_name) ((if table_insensitive equal?? equal?) alias_name alias))
					(reduce cols (lambda (found coldef) (or found ((if column_insensitive equal?? equal?) (coldef "Field") column_name))) false))
					(cons alias acc)
					acc)
			) '()))
			(match matches
				'() nil
				(cons only _) only
			)
		)))
		(define replace_get_column_subselect (lambda (alias_name table_insensitive column_name column_insensitive expr) (begin
			(define inner_alias (column_exists_in_schema schemas2 alias_name table_insensitive column_name column_insensitive))
			(define inner_alias_exists (and (not (nil? alias_name)) (alias_exists_in_schema schemas2 alias_name table_insensitive)))
			(if (and inner_alias_exists (nil? inner_alias))
				(error (concat "column " alias_name "." column_name " does not exist in subquery"))
				(if (not (nil? inner_alias))
					(if (or (nil? alias_name) table_insensitive column_insensitive)
						'((quote get_column) inner_alias false column_name false)
						expr)
					(begin
						(define outer_alias (column_exists_in_schema outer_schemas alias_name table_insensitive column_name column_insensitive))
						(if (nil? outer_alias)
							(if (nil? alias_name)
								(error (concat "column " column_name " does not exist in outer query"))
								expr)
							(list (quote outer) (symbol (concat outer_alias "." column_name))))
					)
				)
			)
		)))
		(define is_get_column_sym (lambda (sym)
			(or (equal? sym (quote get_column))
				(equal? sym '(quote get_column))
				(equal? sym '(symbol get_column))
			)
		))
		(define replace_find_column_subselect (lambda (expr) (match expr
			(cons sym args) (if (is_get_column_sym sym)
				(match args
					'(alias_name table_insensitive column_name column_insensitive) (replace_get_column_subselect alias_name table_insensitive column_name column_insensitive expr)
					_ (cons sym (map args replace_find_column_subselect))
				)
				(cons sym (map args replace_find_column_subselect))
			)
			expr
		)))
		replace_find_column_subselect
	)))

	(define build_scalar_subselect (lambda (subquery outer_schemas) (begin
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			(error "scalar subselect UNION ALL is not supported yet")
			(begin
				(define raw_vals (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)
				))
				(define raw_group (nth raw_vals 0))
				(define raw_having (nth raw_vals 1))
				(define raw_order (nth raw_vals 2))
				(define raw_limit (nth raw_vals 3))
				(define raw_offset (nth raw_vals 4))
				(match (apply untangle_query subquery)
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2)
					(begin
						(define groups2 (coalesceNil groups2 '()))
						(define groups2 (if (or (nil? groups2) (equal? groups2 '()))
							(if (or raw_group raw_having raw_order raw_limit raw_offset)
								(list (make_group_stage raw_group raw_having raw_order raw_limit raw_offset))
								groups2)
							groups2))
						(define replace_find_column_subselect (make_replace_find_column_subselect schemas2 outer_schemas))
						(define field_exprs (extract_assoc fields2 (lambda (k v) v)))
						(define value_expr (match field_exprs
							(cons only '()) only
							_ (error "scalar subselect must return single column")
						))
						(define scalar_neutral (list (quote quote) (quote __scalar_empty)))
						(define scalar_reduce (list (quote lambda) (list (symbol "acc") (symbol "v"))
							(list (quote if)
								(list (quote equal?) (quote acc) scalar_neutral)
								(list (quote if)
									(list (quote equal?) (quote v) scalar_neutral)
									(quote acc)
									(quote v)
								)
								(list (quote if)
									(list (quote equal?) (quote v) scalar_neutral)
									(quote acc)
									(list (quote error) "scalar subselect returned more than one row")
								)
							)
						))
						(set fields2 (map_assoc fields2 (lambda (k v) (replace_find_column_subselect v))))
						(set condition2 (replace_find_column_subselect (coalesceNil condition2 true)))
						(define replace_resultrow (lambda (expr) (match expr
							(cons sym args) (if (equal? sym (quote resultrow))
								(cons (symbol "__scalar_resultrow") (map args replace_resultrow))
								(if (and (equal? sym (quote symbol)) (equal? args '("resultrow")))
									(list (quote symbol) "__scalar_resultrow")
									(cons (replace_resultrow sym) (map args replace_resultrow))
								)
							)
							expr
						)))
						(define subplan (replace_resultrow (build_queryplan schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column_subselect)))
						(list (quote begin)
							(list (quote set) (symbol "accsess") (list (quote newsession)))
							(list (symbol "accsess") "acc" scalar_neutral)
							(list (quote set) (symbol "__scalar_resultrow")
								(list (quote lambda) (list (symbol "row"))
									(list (quote begin)
										(list (symbol "accsess") "acc"
											(list scalar_reduce
												(list (symbol "accsess") "acc")
												(list (quote nth) (symbol "row") 1)))
										true
									)
								)
							)
							subplan
							(list (quote if)
								(list (quote equal?) (list (symbol "accsess") "acc") scalar_neutral)
								nil
								(list (symbol "accsess") "acc"))
						)
					)
				)
			)
		)
	)))

	(define build_in_subselect (lambda (target_expr subquery outer_schemas) (begin
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			(match union_parts '(branches order limit offset) (begin
				(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
					(error "UNION ALL with ORDER/LIMIT/OFFSET is not supported yet in IN subselects"))
				(if (or (nil? branches) (equal? branches '()))
					false
					(cons (quote or) (map branches (lambda (branch) (build_in_subselect target_expr branch outer_schemas)))))
			))
			(begin
				(define raw_vals (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)
				))
				(define raw_group (nth raw_vals 0))
				(define raw_having (nth raw_vals 1))
				(define raw_order (nth raw_vals 2))
				(define raw_limit (nth raw_vals 3))
				(define raw_offset (nth raw_vals 4))
				(match (apply untangle_query subquery)
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2)
					(begin
						(define groups2 (coalesceNil groups2 '()))
						(define groups2 (if (or (nil? groups2) (equal? groups2 '()))
							(if (or raw_group raw_having raw_order raw_limit raw_offset)
								(list (make_group_stage raw_group raw_having raw_order raw_limit raw_offset))
								groups2)
							groups2))
						(define has_stage (and (not (nil? groups2)) (not (equal? groups2 '()))))
						(if (and has_stage (not (equal? (cdr groups2) '()))) (error "multiple group stages not supported yet in IN subselects"))
						(define stage (if has_stage (car groups2) nil))
						(define stage_group (if stage (coalesceNil (stage_group_cols stage) '()) nil))
						(define stage_having (if stage (stage_having_expr stage) nil))
						(define stage_order (if stage (coalesceNil (stage_order_list stage) '()) nil))
						(define stage_limit (if stage (stage_limit_val stage) nil))
						(define stage_offset (if stage (stage_offset_val stage) nil))
						(if (or (and (not (nil? stage_group)) (not (equal? stage_group '()))) (not (nil? stage_having)))
							(error "group/having is not supported yet in IN subselects")
						)
						(define replace_find_column_subselect (make_replace_find_column_subselect schemas2 outer_schemas))
						(define replace_find_column_outer (make_replace_find_column_subselect '() outer_schemas))
						(define field_exprs (extract_assoc fields2 (lambda (key value) value)))
						(define value_expr (match field_exprs
							(cons only '()) only
							_ (error "IN subselect must return single column")
						))
						(set target_expr (replace_find_column_outer target_expr))
						(set value_expr (replace_find_column_subselect value_expr))
						(set condition2 (replace_find_column_subselect (coalesceNil condition2 true)))
						(define in_expr (if (or (nil? tables2) (equal? tables2 '()))
							(begin
								(define limit_zero (and (not (nil? stage_limit)) (equal? stage_limit 0)))
								(define offset_positive (and (not (nil? stage_offset)) (> stage_offset 0)))
								(if (or limit_zero offset_positive)
									false
									(list (quote and) condition2 (list (quote equal??) target_expr value_expr)))
							)
							(begin
								(if (not (and (list? tables2) (equal? (count tables2) 1)))
									(error "IN subselect with multiple tables not supported yet")
								)
								(define tdesc (car tables2))
								(if (not (and (list? tdesc) (equal? (count tdesc) 5)))
									(error "IN subselect with multiple tables not supported yet")
								)
								(define tblvar (nth tdesc 0))
								(define schema3 (nth tdesc 1))
								(define tbl (nth tdesc 2))
								(define isOuter (nth tdesc 3))
								(define joinexpr (nth tdesc 4))
								(if (not (nil? joinexpr)) (error "IN subselect joins not supported yet"))
								(define filtercols (extract_columns_for_tblvar tblvar condition2))
								(define mapcols (extract_columns_for_tblvar tblvar value_expr))
								(define use_ordered (or (and (not (nil? stage_order)) (not (equal? stage_order '()))) (not (nil? stage_limit)) (not (nil? stage_offset))))
								(define ordercols (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									_ '()
								))))))
								(define dirs (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									_ '()
								))))))
								(if (and use_ordered (not (equal? stage_order '())) (not (equal? (count ordercols) (count stage_order))))
									(error "IN subselect ORDER BY must use direct columns")
								)
								(define in_reduce (list (quote lambda) (list (symbol "acc") (symbol "v"))
									(list (quote or) (quote acc) (quote v))
								))
								(define in_neutral false)
								(define map_expr (list (quote equal??) (replace_columns_from_expr target_expr) (replace_columns_from_expr value_expr)))
								(if use_ordered
									(list (quote scan_order)
										schema3
										tbl
										(cons list filtercols)
										(list (quote lambda)
											(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
											(optimize (replace_columns_from_expr condition2))
										)
										(cons list ordercols)
										(cons list dirs)
										(coalesceNil stage_offset 0)
										(coalesceNil stage_limit -1)
										(cons list mapcols)
										(list (quote lambda)
											(map mapcols (lambda (col) (symbol (concat tblvar "." col))))
											map_expr
										)
										in_reduce
										in_neutral
									)
									(list (quote scan)
										schema3
										tbl
										(cons list filtercols)
										(list (quote lambda)
											(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
											(optimize (replace_columns_from_expr condition2))
										)
										(cons list mapcols)
										(list (quote lambda)
											(map mapcols (lambda (col) (symbol (concat tblvar "." col))))
											map_expr
										)
										in_reduce
										in_neutral
										in_reduce
									)
								)
							)
						))
						in_expr
					)
				)
			)
		)
	)))

	(define build_exists_subselect (lambda (subquery outer_schemas) (begin
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			(match union_parts '(branches order limit offset) (begin
				(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
					(error "UNION ALL with ORDER/LIMIT/OFFSET is not supported yet in EXISTS subselects"))
				(if (or (nil? branches) (equal? branches '()))
					false
					(cons (quote or) (map branches (lambda (branch) (build_exists_subselect branch outer_schemas)))))
			))
			(begin
				(define raw_vals (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)
				))
				(define raw_group (nth raw_vals 0))
				(define raw_having (nth raw_vals 1))
				(define raw_order (nth raw_vals 2))
				(define raw_limit (nth raw_vals 3))
				(define raw_offset (nth raw_vals 4))
				(match (apply untangle_query subquery)
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2)
					(begin
						(define groups2 (coalesceNil groups2 '()))
						(define groups2 (if (or (nil? groups2) (equal? groups2 '()))
							(if (or raw_group raw_having raw_order raw_limit raw_offset)
								(list (make_group_stage raw_group raw_having raw_order raw_limit raw_offset))
								groups2)
							groups2))
						(define has_stage (and (not (nil? groups2)) (not (equal? groups2 '()))))
						(if (and has_stage (not (equal? (cdr groups2) '()))) (error "multiple group stages not supported yet in EXISTS subselects"))
						(define stage (if has_stage (car groups2) nil))
						(define stage_group (if stage (coalesceNil (stage_group_cols stage) '()) nil))
						(define stage_having (if stage (stage_having_expr stage) nil))
						(define stage_order (if stage (coalesceNil (stage_order_list stage) '()) nil))
						(define stage_limit (if stage (stage_limit_val stage) nil))
						(define stage_offset (if stage (stage_offset_val stage) nil))
						(if (or (and (not (nil? stage_group)) (not (equal? stage_group '()))) (not (nil? stage_having)))
							(error "group/having is not supported yet in EXISTS subselects")
						)
						(define replace_find_column_subselect (make_replace_find_column_subselect schemas2 outer_schemas))
						(set condition2 (replace_find_column_subselect (coalesceNil condition2 true)))
						(define exists_expr (if (or (nil? tables2) (equal? tables2 '()))
							(begin
								(define limit_zero (and (not (nil? stage_limit)) (equal? stage_limit 0)))
								(define offset_positive (and (not (nil? stage_offset)) (> stage_offset 0)))
								(if (or limit_zero offset_positive)
									false
									condition2)
							)
							(begin
								(if (not (and (list? tables2) (equal? (count tables2) 1)))
									(error "EXISTS subselect with multiple tables not supported yet")
								)
								(define tdesc (car tables2))
								(if (not (and (list? tdesc) (equal? (count tdesc) 5)))
									(error "EXISTS subselect with multiple tables not supported yet")
								)
								(define tblvar (nth tdesc 0))
								(define schema3 (nth tdesc 1))
								(define tbl (nth tdesc 2))
								(define isOuter (nth tdesc 3))
								(define joinexpr (nth tdesc 4))
								(if (not (nil? joinexpr)) (error "EXISTS subselect joins not supported yet"))
								(define filtercols (extract_columns_for_tblvar tblvar condition2))
								(define use_ordered (or (and (not (nil? stage_order)) (not (equal? stage_order '()))) (not (nil? stage_limit)) (not (nil? stage_offset))))
								(define ordercols (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									_ '()
								))))))
								(define dirs (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									_ '()
								))))))
								(if (and use_ordered (not (equal? stage_order '())) (not (equal? (count ordercols) (count stage_order))))
									(error "EXISTS subselect ORDER BY must use direct columns")
								)
								(define exists_reduce (list (quote lambda) (list (symbol "acc") (symbol "v"))
									(list (quote or) (quote acc) (quote v))
								))
								(define exists_neutral false)
								(if use_ordered
									(list (quote scan_order)
										schema3
										tbl
										(cons list filtercols)
										(list (quote lambda)
											(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
											(optimize (replace_columns_from_expr condition2))
										)
										(cons list ordercols)
										(cons list dirs)
										(coalesceNil stage_offset 0)
										(coalesceNil stage_limit -1)
										(cons list '())
										(list (quote lambda) '() true)
										exists_reduce
										exists_neutral
									)
									(list (quote scan)
										schema3
										tbl
										(cons list filtercols)
										(list (quote lambda)
											(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
											(optimize (replace_columns_from_expr condition2))
										)
										(cons list '())
										(list (quote lambda) '() true)
										exists_reduce
										exists_neutral
										exists_reduce
									)
								)
							)
						))
						exists_expr
					)
				)
			)
		)
	)))

	(define inner_select_kind (lambda (sym) (begin
		(if (string? sym)
			(if (equal?? sym "inner_select")
				(quote inner_select)
				(if (equal?? sym "inner_select_in")
					(quote inner_select_in)
					(if (equal?? sym "inner_select_exists")
						(quote inner_select_exists)
						nil)))
			(match sym
				(symbol inner_select) (quote inner_select)
				'inner_select (quote inner_select)
				'(quote inner_select) (quote inner_select)
				(symbol inner_select_in) (quote inner_select_in)
				'inner_select_in (quote inner_select_in)
				'(quote inner_select_in) (quote inner_select_in)
				(symbol inner_select_exists) (quote inner_select_exists)
				'inner_select_exists (quote inner_select_exists)
				'(quote inner_select_exists) (quote inner_select_exists)
				_ nil
			)
		)
	)))
	(define not_symbol (lambda (sym) (match sym
		(symbol not) true
		'not true
		'(quote not) true
		_ false
	)))
	(define replace_inner_selects (lambda (expr outer_schemas) (match expr
		(cons sym args) (begin
			(define kind (inner_select_kind sym))
			(define not_expr (if (not_symbol sym)
				(match args
					(cons inner_expr '()) (match inner_expr
						(cons inner_sym inner_args) (begin
							(define inner_kind (inner_select_kind inner_sym))
							(if (equal?? inner_kind (quote inner_select_in))
								(match inner_args
									(cons target_expr (cons subquery '())) (list (quote not) (build_in_subselect target_expr subquery outer_schemas))
									_ nil
								)
								nil)
						)
						_ nil
					)
					_ nil
				)
				nil))
			(if (nil? not_expr)
				(match kind
					(quote inner_select) (match args
						(cons subquery '()) (build_scalar_subselect subquery outer_schemas)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas))))
					)
					(quote inner_select_in) (match args
						(cons target_expr (cons subquery '())) (build_in_subselect target_expr subquery outer_schemas)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas))))
					)
					(quote inner_select_exists) (match args
						(cons subquery '()) (build_exists_subselect subquery outer_schemas)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas))))
					)
					_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas))))
				)
				not_expr
			)
		)
		expr
	)))

	/* check if we have FROM selects -> returns '(tables renamelist condition schemas) */
	(if (or (nil? tables) (equal? tables '()))
		(begin
			(set fields (map_assoc fields (lambda (k v) (replace_inner_selects v '()))))
			(set condition (replace_inner_selects condition '()))
			(set group (map group (lambda (g) (replace_inner_selects g '()))))
			(set having (replace_inner_selects having '()))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col '()) dir)))))
			(set groups (if (or group having order limit offset) (list (make_group_stage group having order limit offset)) nil))
			(list schema tables fields condition groups '() (lambda (expr) expr))
		)
		(begin
			(set zipped (zip (map tables (lambda (tbldesc) (match tbldesc
				'(alias schema (string? tbl) _ _) '('(tbldesc) '() true '(alias (get_schema schema tbl))) /* leave primary tables as is and load their schema definition */
				'(id schemax subquery isOuter joinexpr) (begin
					(define union_parts_from (query_union_all_parts subquery))
					(if (not (nil? union_parts_from))
						(match union_parts_from '(branches union_order union_limit union_offset) (begin
							(define output_cols (match branches
								(cons first_branch _) (query_branch_field_names first_branch)
								_ '()))
							(if (or (nil? output_cols) (equal? output_cols '()))
								(error "UNION ALL subquery must project at least one column"))
							(define rows_sym (symbol (concat "__from_union_rows:" id)))
							(define resultrow_sym (symbol (concat "__from_union_resultrow:" id)))
							(define materialized_rows (list (quote begin)
								(list (quote set) rows_sym (list (quote newsession)))
								(list rows_sym "rows" '())
								(list (quote set) resultrow_sym (symbol "resultrow"))
								(list (quote set) (symbol "resultrow")
									(list (quote lambda) (list (symbol "item"))
										(list rows_sym "rows"
											(list (quote append)
												(list rows_sym "rows")
												(list (quote list) (symbol "item")))))
								)
								(build_queryplan_term subquery)
								(list (quote set) (symbol "resultrow") resultrow_sym)
								(list rows_sym "rows")
							))
							(list
								(list (list id schemax materialized_rows isOuter joinexpr))
								'()
								true
								(list id (map output_cols (lambda (col) '("Field" col "Type" "any"))))
							)
						))
						(match (apply untangle_query subquery) '(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2) (begin
							/* helper function add prefix to tblalias of every expression */
							(define replace_column_alias (lambda (expr) (match expr
								'((symbol get_column) nil ti col ci) (begin
									/* resolve unqualified column against inner schemas2; must match exactly one table.
									Skip aliases that contain ':' — those are prefixed from flattened derived tables
									and should not participate in unqualified column resolution. */
									(define matches (reduce_assoc schemas2 (lambda (acc alias cols)
										(if (and (equal? (replace alias ":" "") alias)
											(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
											(cons alias acc)
											acc)) '()))
									(match matches
										(cons only '()) '('get_column (concat id ":" only) ti col ci)
										'() (begin
											/* column not in schemas2 - check if it's a SELECT alias in fields2 */
											(if (nil? (fields2 col))
												(error (concat "column " col " does not exist in subquery"))
												/* found in fields2 - resolve to the underlying expression */
												(replace_column_alias (fields2 col))
											)
										)
										(cons _ _) (error (concat "ambiguous column " col " in subquery"))
									)
								)
								'((symbol get_column) alias_ ti col ci) '('get_column (concat id ":" alias_) ti col ci)
								'((symbol outer) outer_arg) (begin
									/* prefix outer variable reference if it refers to a table in schemas2 */
									(define s (string outer_arg))
									(define parts (split s "."))
									(match parts
										(list tbl col) (if (not (nil? (schemas2 tbl)))
											(list (quote outer) (symbol (concat id ":" tbl "." col)))
											(list (quote outer) outer_arg))
										_ (list (quote outer) (replace_column_alias outer_arg))
									)
								)
								(cons sym args) /* function call */ (cons sym (map args replace_column_alias))
								expr
							)))
							/* prefix all table aliases and transform their joinexprs */
							(set tablesPrefixed (map tables2 (lambda (x) (match x '(alias schema tbl a innerJoinexpr)
								(list (concat id ":" alias) schema tbl a (if (nil? innerJoinexpr) nil (replace_column_alias innerJoinexpr)))))))
							/* helper function to transform joinexpr: only transform references to subquery alias id */
							(define transform_joinexpr (lambda (expr) (match expr
								'((symbol get_column) alias_ ti col ci) (if (equal?? alias_ id)
									/* reference to subquery alias -> resolve against inner schemas by passing nil alias */
									(replace_column_alias (list (quote get_column) nil ti col ci))
									/* reference to outer table -> keep as-is */
									expr)
								(cons sym args) /* function call */ (cons sym (map args transform_joinexpr))
								expr
							)))
							/* transform and attach joinexpr to first table in tablesPrefixed */
							(set joinexpr2 (if (nil? joinexpr) nil (transform_joinexpr joinexpr)))
							/* for LEFT JOIN (isOuter=true), integrate condition2 into joinexpr to preserve LEFT JOIN semantics */
							(set condition2_transformed (replace_column_alias condition2))
							(set joinexpr2 (if isOuter
								/* merge condition2 into joinexpr for outer joins */
								(if (nil? joinexpr2)
									condition2_transformed
									(if (or (nil? condition2_transformed) (equal? condition2_transformed true))
										joinexpr2
										(list (quote and) joinexpr2 condition2_transformed)))
								joinexpr2))
							(if (and (not (nil? joinexpr2)) (not (nil? tablesPrefixed)))
								(set tablesPrefixed (cons
									/* inherit isOuter from the subquery's join type, not from inner table */
									(match (car tablesPrefixed) '(a s t io je) (list a s t isOuter joinexpr2))
									(cdr tablesPrefixed)))
							)
							(define use_materialize false)
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
									(if unsupported
										(set use_materialize true)
										(set groups2 nil))
								)
							)
							(if use_materialize
								(begin
									(define output_cols_sub (extract_assoc fields2 (lambda (k v) k)))
									(define rows_sym (symbol (concat "__from_subquery_rows:" id)))
									(define resultrow_sym (symbol (concat "__from_subquery_resultrow:" id)))
									(define materialized_rows (list (quote begin)
										(list (quote set) rows_sym (list (quote newsession)))
										(list rows_sym "rows" '())
										(list (quote set) resultrow_sym (symbol "resultrow"))
										(list (quote set) (symbol "resultrow")
											(list (quote lambda) (list (symbol "item"))
												(list rows_sym "rows"
													(list (quote append)
														(list rows_sym "rows")
														(list (quote list) (symbol "item")))))
										)
										(build_queryplan_term subquery)
										(list (quote set) (symbol "resultrow") resultrow_sym)
										(list rows_sym "rows")
									))
									(list
										(list (list id schemax materialized_rows isOuter joinexpr))
										'()
										true
										(list id (map output_cols_sub (lambda (col) '("Field" col "Type" "any"))))
									)
								)
								(begin
									/* for LEFT JOIN: condition2 was integrated into joinexpr, so return true as global filter */
									/* for INNER JOIN: condition2 becomes global filter (can be reordered) */
									(set globalFilter (if isOuter true (replace_column_alias condition2)))
									(list tablesPrefixed (list id (map_assoc fields2 (lambda (k v) (replace_column_alias v)))) globalFilter (merge (list alias (extract_assoc fields2 (lambda (k v) '("Field" k "Type" "any")))) (merge (extract_assoc schemas2 (lambda (k v) (list (concat id ":" k) v))))))
								)
							)
						) (error "non matching return value for untangle_query"))
					)
				)
				(error (concat "unknown tabledesc: " tbldesc))
			)))))
			(set tablesList (car zipped))
			(set renameList (car (cdr zipped)))
			(set conditionList (car (cdr (cdr zipped))))
			(set schemasList (car (cdr (cdr (cdr zipped)))))
			/* schemas is an assoc array from alias -> columnlist */
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
				(cons sym args) /* function call */ (if (nil? (inner_select_kind sym))
					(reduce args (lambda (a b) (or a (expr_find_aggregate b))) false)
					false)
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
				/* Unqualified column: prefer main tables (no ':' prefix) over subquery tables (prefixed with ':') */
				'((symbol get_column) nil _ col ci) (begin
					/* First try main tables (aliases without ':') */
					(define main_match (reduce_assoc schemas (lambda (a alias cols)
						(if (and (not (strlike (string alias) "%:%")) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
							alias a)) nil))
					/* If not found in main tables, try subquery tables (aliases with ':') */
					(define any_match (if (nil? main_match)
						(reduce_assoc schemas (lambda (a alias cols)
							(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)
								alias a)) nil)
						main_match))
					'((quote get_column) (coalesce any_match (error (concat "column " col " does not exist in tables"))) false col false)
				)
				'((symbol get_column) alias_ ti col ci) (if (or ti ci) '((quote get_column) (coalesce (reduce_assoc schemas (lambda (a alias cols) (if (and ((if ti equal?? equal?) alias_ alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)) alias a)) nil) (error (concat "column " alias_ "." col " does not exist in tables"))) false col false) expr) /* omit false false, otherwise freshly created columns wont be found */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))

			(set fields (map_assoc fields (lambda (k v) (replace_inner_selects v schemas))))
			(set condition (replace_inner_selects condition schemas))
			(set group (map group (lambda (g) (replace_inner_selects g schemas))))
			(set having (replace_inner_selects having schemas))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col schemas) dir)))))

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
				(list col (replace_find_column expr))
			)))))

			/* return parameter list for build_queryplan */
			(set conditionAll (cons 'and (filter (cons (replace_rename condition) conditionList) (lambda (x) (not (nil? x)))))) /* TODO: append inner conditions to condition */
			(set group (map group replace_rename))
			(set having (replace_rename having))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_rename col) dir)))))
			(define groups (if (coalesce _cd_distinct_exprs false)
				/* COUNT(DISTINCT): two group stages - first dedup, then aggregate */
				(list
					(make_dedup_stage
						(merge (map (coalesce _cd_user_group '()) replace_rename) (map _cd_distinct_exprs (lambda (e) (replace_find_column (replace_rename e))))))
					(make_group_stage
						(if (nil? _cd_user_group) '(1) (map _cd_user_group (lambda (e) (replace_find_column (replace_rename e)))))
						(_cd_replace (replace_rename _cd_having))
						(map (coalesce _cd_order '()) (lambda (o) (match o '(col dir) (list (_cd_replace (replace_rename col)) dir))))
						_cd_limit _cd_offset))
				/* normal: single group stage */
				(if (or group having order limit offset) (list (make_group_stage group having order limit offset)) nil)))
			(list schema tables (map_assoc fields (lambda (k v) (replace_rename v))) conditionAll groups schemas replace_find_column)
		)
	)
)
))

(define build_queryplan_term (lambda (query) (begin
	(define union_parts (query_union_all_parts query))
	(if (nil? union_parts)
		(if (query_is_select_core query)
			(apply build_queryplan (apply untangle_query query))
			(error "invalid SELECT query term"))
		(match union_parts '(branches order limit offset) (begin
			(if (or (nil? branches) (equal? branches '()))
				(error "UNION ALL requires at least one branch"))
			(define branch_meta (map branches (lambda (branch) (begin
				(if (not (query_is_select_core branch))
					(error "UNION ALL branch must be a SELECT query"))
				(match branch '(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2) (begin
					(if (or (not (nil? order2)) (not (nil? limit2)) (not (nil? offset2)))
						(error "UNION ALL branch ORDER/LIMIT/OFFSET is not supported yet"))
					(define branch_cols (query_branch_field_names branch))
					(list branch branch_cols (count branch_cols) schema2))
					_ (error "UNION ALL branch must be a SELECT query"))
			))))
			(define expected_cols (match branch_meta
				(cons first_meta _) (nth first_meta 2)
				_ 0))
			(define output_cols (match branch_meta
				(cons first_meta _) (nth first_meta 1)
				_ '()))
			(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
				(error "UNION ALL with global ORDER BY/LIMIT/OFFSET is not supported yet"))
			(if (not (reduce branch_meta (lambda (ok meta) (and ok (equal? (nth meta 2) expected_cols))) true))
				(error "UNION ALL branches must project the same number of columns"))
			(define branch_plans (map branch_meta (lambda (meta) (begin
				(define branch (nth meta 0))
				(define branch_plan (build_queryplan_term branch))
				(define normalized_row (cons (quote list) (merge (map (produceN expected_cols) (lambda (idx)
					(list (nth output_cols idx) (list (quote nth) (symbol "row") (+ (* idx 2) 1)))
				)))))
				(list (quote begin)
					(list (quote set) (symbol "__union_prev_resultrow") (symbol "resultrow"))
					(list (quote set) (symbol "resultrow")
						(list (quote lambda) (list (symbol "row"))
							(list (symbol "__union_prev_resultrow") normalized_row)))
					branch_plan
					(list (quote set) (symbol "resultrow") (symbol "__union_prev_resultrow")))
			))))
			(cons (quote begin) branch_plans)
		))
	)
)))

/* build queryplan from parsed query
GROUP BY aggregate pipeline:
When a GROUP BY query has aggregates (SUM, COUNT, etc.), three phases run:
1. collect_plan: extract unique group keys from base table into a keytable
2. compute_plan: for each aggregate in 'ags', scan base table per group key
and store results as keytable columns named "expr|condition"
3. grouped_plan: scan populated keytable for final output (ORDER BY, HAVING, LIMIT)
All aggregates from fields, ORDER BY, and HAVING are collected into 'ags' so that
e.g. ORDER BY SUM(amount) works even if SUM(amount) only appears in ORDER BY.
*/
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
		(set stage_order (map stage_order (lambda (o) (match o '(col dir) (list (replace_find_column col) dir)))))
		(define is_dedup (stage_is_dedup stage))
		/* collect all unique aggregate tuples (expr reduce neutral) from fields, ORDER BY, and HAVING.
		Each tuple becomes a computed column on the keytable, e.g. SUM(amount) -> ((get_column t amount) + 0).
		ORDER BY SUM(x) requires SUM(x) to be pre-computed here even if not in SELECT. */
		(define ags_raw (if is_dedup '() (extract_assoc fields (lambda (key expr) (extract_aggregates expr)))))
		(define ags (if is_dedup '() (merge_unique ags_raw))) /* aggregates in fields */
		(define ags (if is_dedup ags (merge_unique ags (merge_unique (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (extract_aggregates col)))))))) /* aggregates in order */
		(define ags (if is_dedup ags (merge_unique ags (extract_aggregates (coalesce stage_having true))))) /* aggregates in having */

		/* TODO: replace (get_column nil ti col ci) in group, having and order with (coalesce (fields col) '('get_column nil false col false)) */

		(match tables
			/* TODO: allow for more than just group by single table */
			/* TODO: outer tables that only join on group */
			'('(tblvar schema tbl isOuter _)) (begin
				/* prepare preaggregate */

				(set grouptbl (make_keytable schema tbl stage_group tblvar (if is_dedup condition nil)))

				/* preparation */
				(define tblvar_cols (merge_unique (map stage_group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesceNil condition true)))
				(set filtercols (extract_columns_for_tblvar tblvar condition))

				/* collect plan: insert unique group keys into group table (shared by dedup and normal stages) */
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

				(if is_dedup (begin
					/* DEDUP-ONLY stage: no aggregate computation, just collect unique keys and pass through to next stage */
					(define replace_col_for_dedup (make_col_replacer grouptbl condition true))
					/* transform rest_groups to reference grouptbl columns instead of source table columns;
					first resolve nil -> tblvar via replace_find_column, then map tblvar -> grouptbl */
					(define _dedup_resolve (lambda (e) (replace_col_for_dedup (replace_find_column e))))
					(define transformed_rest_groups (map rest_groups (lambda (s)
						(make_group_stage
							(map (stage_group_cols s) _dedup_resolve)
							(_dedup_resolve (stage_having_expr s))
							(map (coalesce (stage_order_list s) '()) (lambda (o) (match o '(col dir) (list (_dedup_resolve col) dir))))
							(stage_limit_val s)
							(stage_offset_val s))
					)))
					(define grouped_plan (build_queryplan schema '('(grouptbl schema grouptbl false nil))
						(map_assoc fields (lambda (k v) (_dedup_resolve v)))
						nil /* condition already applied in collect */
						transformed_rest_groups
						schemas
						replace_find_column))
					(list 'begin collect_plan grouped_plan)
				) (begin
						/* NORMAL group stage: extract aggregates, compute, and continue.
						replace_agg_with_fetch rewrites (aggregate expr + 0) -> (get_column grouptbl "expr|cond")
						so ORDER BY SUM(amount) becomes ORDER BY on a keytable column. */
						(define replace_agg_with_fetch (make_col_replacer grouptbl condition false))
						(define replace_group_key_or_fetch (lambda (expr) (if
							(reduce stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
							'('get_column grouptbl false (concat expr) false)
							(replace_agg_with_fetch expr)
						)))

						(define grouped_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (replace_group_key_or_fetch col) dir))))))
						(define next_groups (merge
							(if (coalesce grouped_order stage_limit stage_offset) (list (make_group_stage nil nil grouped_order stage_limit stage_offset)) '())
							rest_groups
						))
						(define grouped_plan (build_queryplan schema '('(grouptbl schema grouptbl false nil))
							(map_assoc fields (lambda (k v) (replace_group_key_or_fetch v)))
							(replace_group_key_or_fetch stage_having)
							next_groups
							schemas
							replace_find_column))

						(define compute_plan
							'('time (cons 'parallel (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
								(set cols (extract_columns_for_tblvar tblvar expr))
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
				))
			)
			(error "Grouping and aggregates on joined tables is not implemented yet (prejoins)") /* TODO: construct grouptbl as join */
		)
	) (optimize (begin
			/* grouping has been removed; now to the real data: */
			(if (and (not (nil? rest_groups)) (not (equal? rest_groups '()))) (error "non-group stage must be last"))
			(if (coalesce stage_order stage_limit stage_offset) (begin
				/* ordered or limited scan */
				/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
				(set stage_order (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (list (replace_find_column col) dir)))))
				/* build_scan now takes is_first parameter to apply offset/limit only to outermost scan */
				(define build_scan (lambda (tables condition is_first)
					(match tables
						(cons '(tblvar schema tbl isOuter _) tables) (begin /* outer scan */
							(set cols (merge_unique
								(list
									(merge_unique
										(cons
											(extract_columns_for_tblvar tblvar condition)
											(extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v)))
										)
									)
									(merge_unique
										(cons
											(extract_outer_columns_for_tblvar tblvar condition)
											(extract_assoc fields (lambda (k v) (extract_outer_columns_for_tblvar tblvar v)))
										)
									)
								)
							))
							(match (split_condition (coalesceNil condition true) tables) '(now_condition later_condition) (begin
								(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
								/* TODO: add columns from rest condition into cols list */

								/* extract order cols for this tblvar */
								/* TODO: match case insensitive column */
								/* TODO: non-trivial columns to computed columns */
								/* preserve ORDER BY key order (first key has highest priority) */
								(set ordercols (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
									_ '()
								))))))
								(set dirs (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
									_ '()
								))))))

								/* offset/limit only apply to the outermost scan, not to nested JOINs */
								(define scan_offset (if is_first stage_offset 0))
								(define scan_limit (if is_first (coalesceNil stage_limit -1) -1))

								(scan_wrapper 'scan_order schema tbl
									/* condition */
									(cons list filtercols)
									'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
									/* sortcols, sortdirs */
									(cons list ordercols)
									(cons list dirs)
									scan_offset
									scan_limit
									/* extract columns and store them into variables */
									(cons list cols)
									'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (build_scan tables later_condition false))
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
				(build_scan tables (replace_find_column condition) true)
			) (begin
					/* unordered unlimited scan */

					/* TODO: sort tables according to join plan */
					/* TODO: match tbl to inner query vs string */
					(define build_scan (lambda (tables condition)
						(match tables
							(cons '(tblvar schema tbl isOuter _) tables) (begin /* outer scan */
								(set cols (merge_unique
									(list
										(merge_unique
											(cons
												(extract_columns_for_tblvar tblvar condition)
												(extract_assoc fields (lambda (k v) (extract_columns_for_tblvar tblvar v)))
											)
										)
										(merge_unique
											(cons
												(extract_outer_columns_for_tblvar tblvar condition)
												(extract_assoc fields (lambda (k v) (extract_outer_columns_for_tblvar tblvar v)))
											)
										)
									)
								))
								/* split condition in those ANDs that still contain get_column from tables and those evaluatable now */
								(match (split_condition (coalesceNil condition true) tables) '(now_condition later_condition) (begin
									(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))

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
	)))
)))
