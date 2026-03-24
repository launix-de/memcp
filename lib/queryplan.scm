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

/* Registers invalidation triggers on src_table to drop pj_table on any DML.
Uses code-generator pattern: values baked into quoted lambda body at register time,
so no closure capture — the trigger body serializes cleanly as a self-contained expression. */
(define register_prejoin_invalidation (lambda (src_schema src_table pj_schema pj_table) (begin
	(define prefix (concat ".prejoin:" pj_table "|" src_table "|"))
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW) (list 'droptable pj_schema pj_table true))))
	(createtrigger src_schema src_table (concat prefix "after_insert")     "after_insert"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_update")     "after_update"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_delete")     "after_delete"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* Registers incremental maintenance triggers on src_table to keep pj_table in sync.
delete_fn/insert_fn/update_fn are code-generator-produced lambda expressions (no closures).
Lifecycle triggers use code-generator pattern for the drop body as well.
update_fn embeds delete_fn/insert_fn as proc literals in its body (no closure capture). */
(define register_prejoin_incremental (lambda (src_schema src_table pj_schema pj_table delete_fn insert_fn update_fn) (begin
	(define prefix (concat ".pj_incr:" pj_table "|" src_table "|"))
	(createtrigger src_schema src_table (concat prefix "after_delete") "after_delete" "" delete_fn false)
	(createtrigger src_schema src_table (concat prefix "after_insert") "after_insert" "" insert_fn false)
	(createtrigger src_schema src_table (concat prefix "after_update") "after_update" "" update_fn false)
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW) (list 'droptable pj_schema pj_table true))))
	(createtrigger src_schema src_table (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* returns a list of all tblvar aliases referenced via get_column in expr */
(define extract_tblvars (lambda (expr)
	(match expr
		'((symbol get_column) tblvar _ _ _) (if (nil? tblvar) '() '(tblvar))
		(cons sym args) (merge_unique (map args extract_tblvars))
		'()
	)
))

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
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
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
	(define result (scan_order schema tbl filtercols filterfn sortcols sortdirs 0 offset limit mapcols mapfn reduce neutral))
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

/* returns a list of all window function nodes (fn args over) in this expr */
(define extract_window_funcs (lambda (expr)
	(match expr
		(cons (symbol window_func) rest) (list rest)
		(cons sym args) /* function call */ (merge (map args extract_window_funcs))
		/* literal */ '()
	)
))

/* extract_all_get_columns: return all (get_column tblvar _ col _) refs as ("tblvar.col" expr) pairs */
(define extract_all_get_columns (lambda (expr)
	(match expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
		(cons sym args) (merge (map args extract_all_get_columns))
		'()
	)
))

/* extract_all_outer_columns: return all (outer tblvar.col) refs as ("tblvar.col" (get_column tblvar false col false)) pairs.
   Used alongside extract_all_get_columns to ensure prejoin materialization includes
   columns referenced by scalar subselects via outer scope. */
(define extract_all_outer_columns (lambda (expr)
	(match expr
		(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)) (equal? sym '(symbol outer)))
			(match args
				'(symname) (begin
					(define s (string symname))
					(define parts (split s "."))
					(if (> (count parts) 1)
						(begin
							(define tbl (car parts))
							/* rejoin remaining parts with . for column names containing dots */
							(define col (reduce (cdr parts) (lambda (acc p) (if (equal? acc "") p (concat acc "." p))) ""))
							(list (list (concat tbl "." col) (list (quote get_column) tbl false col false))))
						'()))
				_ '())
			(merge (map args extract_all_outer_columns)))
		'()
	)
))

/* extract_scanned_tables: walk an expression AST and return all (schema table) pairs from scan/scan_order calls.
Used to detect which tables a computor lambda reads from, so we can register invalidation triggers. */
(define extract_scanned_tables (lambda (expr)
	(match expr
		(cons (symbol scan) (cons schema (cons tbl rest))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scan_order) (cons schema (cons tbl rest))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scalar_scan) (cons schema (cons tbl rest))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scalar_scan_order) (cons schema (cons tbl rest))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons sym args) (merge (map args extract_scanned_tables))
		'()
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
				(define s (string symname))
				(define parts (split s "."))
				(if (> (count parts) 1)
					(begin
						(define tbl (car parts))
						(define col (reduce (cdr parts) (lambda (acc p) (if (equal? acc "") p (concat acc "." p))) ""))
						(if (equal?? tbl (string tblvar)) (list col) '()))
					'())
			)
			_ '()
		)
		(merge_unique (map args (lambda (arg) (extract_outer_columns_for_tblvar tblvar arg))))
	)
	'()
)))

/* symbols that canonicalize_columns must NOT recurse into — they have their own scope */
(define _is_opaque_scope_sym (lambda (sym) (match sym
	(symbol inner_select) true '(quote inner_select) true
	(symbol inner_select_in) true '(quote inner_select_in) true
	(symbol inner_select_exists) true '(quote inner_select_exists) true
	/* runtime code produced by build_scalar_subselect — has inner scan context */
	(symbol !begin) true '(quote !begin) true '!begin true
	(symbol scan) true '(quote scan) true 'scan true
	(symbol scan_order) true '(quote scan_order) true 'scan_order true
	(symbol scalar_scan) true '(quote scalar_scan) true 'scalar_scan true
	(symbol newpromise) true '(quote newpromise) true 'newpromise true
	(symbol newsession) true '(quote newsession) true 'newsession true
	_ false)))
/* canonicalize_columns resolves ti/ci flags to canonical casing.
all_schemas includes outer schemas (for qualified outer refs like src.ID).
Unqualified refs are only resolved against local tables (non-prefixed aliases)
to avoid matching outer tables which would break scope resolution. */
(define canonicalize_columns (lambda (expr all_schemas) (match expr
	'((symbol get_column) alias_ ti col ci) (if (or ti ci)
		(begin
			(define resolved_alias (if (nil? alias_)
				/* unqualified: search non-prefixed aliases only (local tables) */
				(reduce_assoc all_schemas (lambda (a alias cols)
					(if (and (equal? (replace (string alias) "\0" "") (string alias))
						(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
						alias a)) nil)
				/* qualified: search all schemas including outer */
				(reduce_assoc all_schemas (lambda (a alias cols)
					(if (and ((if ti equal?? equal?) alias_ alias)
						(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
						alias a)) nil)))
			(if (nil? resolved_alias)
				expr /* leave unresolved — replace_find_column will handle or error */
				(begin
					(define canonical_col (coalesce
						(reduce (all_schemas resolved_alias) (lambda (a coldef)
							(if (not (nil? a)) a
								(if ((if ci equal?? equal?) (coldef "Field") col) (coldef "Field") nil))) nil)
						col))
					(list (quote get_column) resolved_alias false canonical_col false))))
		expr /* ti=false ci=false: already canonical */
	)
	/* do not recurse into opaque scope nodes — inner_select, runtime code */
	(cons sym args) (if (_is_opaque_scope_sym sym) expr
		(cons (canonicalize_columns sym all_schemas) (map args (lambda (a) (canonicalize_columns a all_schemas)))))
	expr
)))
/* canonicalize all get_column markers in a group stage */
(define canonicalize_stage (lambda (stage all_schemas) (begin
	(define canon (lambda (expr) (canonicalize_columns expr all_schemas)))
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_having_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(if (stage_is_dedup stage)
		(make_dedup_stage (map sg canon))
		(make_group_stage
			(map sg canon)
			(canon sh)
			(map so (lambda (o) (match o '(c d) (list (canon c) d))))
			sl soff))
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
Returns (keytable_name init_code fk_pk_col) where init_code is plan-time code that ensures
the table exists at execution time (survives cache eviction of sloppy tables).
fk_pk_col is non-nil when FK→PK reuse is active (parent table used instead of temp keytable).
condition_suffix: if non-nil, appended to name (for dedup stages with WHERE) */
(define make_keytable (lambda (schema tbl keys tblvar condition_suffix) (begin
	/* FK→PK reuse: if single-column GROUP BY on a FK column without condition,
	reuse the parent (referenced) table instead of creating a temp keytable */
	(define fk_result (if (and (nil? condition_suffix) (equal? 1 (count keys)))
		(match (car keys)
			'('get_column (eval tblvar) false scol false) (begin
				(define fk_info (get_fk_target schema tbl scol))
				(if (not (nil? fk_info))
					(list (car fk_info) nil (car (cdr fk_info)))
					nil))
			nil)
		nil))
	(if (not (nil? fk_result))
		fk_result
		(begin
			(define alias_map (list (list tblvar (concat schema "." tbl))))
			(define key_names (map keys (lambda (k) (canonical_expr_name k '(list) '(list) alias_map))))
			(define condition_name (if (nil? condition_suffix) nil (canonical_expr_name condition_suffix '(list) '(list) alias_map)))
			(define key_name_at (lambda (i) (nth key_names i)))
			(define key_at (lambda (i) (nth keys i)))
			(define keytable_name (if (nil? condition_suffix)
				(concat "." tbl ":" key_names)
				(concat "." tbl ":" key_names "|" condition_name)))
			/* compute column definitions and partition spec at compile time */
			(define kt_cols (cons
				'("unique" "group" key_names)
				(map key_names (lambda (colname) '("column" colname "any" '() '())))))
			(define kt_partition (merge (map (produceN (count keys)) (lambda (i)
				(match (key_at i)
					'('get_column (eval tblvar) false scol false) (list (list (key_name_at i) (shardcolumn schema tbl scol)))
					'())))))
			/* create at compile time (needed for recursive build_queryplan) */
			(createtable schema keytable_name kt_cols '("engine" "sloppy") true)
			(partitiontable schema keytable_name kt_partition)
			/* build runtime init code to re-create after potential cache eviction (mirrors prejoin pattern) */
			(define kt_cols_code (cons 'list
				(cons
					(cons 'list (cons "unique" (cons "group" (list (cons 'list key_names)))))
					(map key_names (lambda (colname) (list 'list "column" colname "any" '(list) '(list)))))))
			(define kt_partition_code (cons 'list (merge (map (produceN (count keys)) (lambda (i)
				(match (key_at i)
					'('get_column (eval tblvar) false scol false) (list (list 'list (key_name_at i) (cons 'list (shardcolumn schema tbl scol))))
					'()))))))
			(define init_code (list 'begin
				(list 'createtable schema keytable_name kt_cols_code (list 'list "engine" "sloppy") true)
				(list 'partitiontable schema keytable_name kt_partition_code)
				(list 'touch_keytable schema keytable_name)))
			/* return (name init_code nil) — third element nil means no FK reuse */
			(list keytable_name init_code nil)))
)))

/* build_agg_window_plan: generates the full plan for aggregate window functions (SUM/COUNT/MIN/MAX OVER).
Uses keytable infrastructure (same as GROUP BY): make_keytable + collect + createcolumn + scalar fetch.
Result query runs on the BASE table; window_func expressions are replaced with scalar keytable scans. */
(define build_agg_window_plan (lambda (schema tbl tblvar tables over_partition wf_resolved condition groups schemas replace_find_column fields isOuter replace_columns_from_expr extract_columns_for_tblvar scan_wrapper) (begin
	(define has_partition (not (equal? over_partition '())))
	(define partition_exprs (map over_partition replace_find_column))
	(define group_keys (if has_partition partition_exprs '(1)))
	(define canon_alias_map (list (list tblvar (concat schema "." tbl))))
	(define expr_name (lambda (expr) (canonical_expr_name expr '(list) '(list) canon_alias_map)))
	(set condition (replace_find_column (coalesceNil condition true)))
	(define kt_result (make_keytable schema tbl group_keys tblvar nil))
	(match kt_result '(grouptbl keytable_init fk_pk_col) (begin
		(define is_fk_reuse (not (nil? fk_pk_col)))
		(define tblvar_cols (if has_partition (merge_unique (map group_keys (lambda (col) (extract_columns_for_tblvar tblvar col)))) '()))
		(set filtercols (if has_partition (extract_columns_for_tblvar tblvar condition) '()))
		/* collect plan */
		(define collect_plan (if (equal? group_keys '(1))
			'('insert schema grouptbl '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
			(begin
				(define keycols (merge_unique (map group_keys (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
				(scan_wrapper 'scan schema tbl
					(cons list filtercols)
					'('lambda (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
					(cons list keycols)
					'('lambda (map keycols (lambda (col) (symbol (concat tblvar "." col)))) (cons 'list (map group_keys (lambda (expr) (replace_columns_from_expr expr)))))
					'('lambda '('acc 'rowvals) '('set_assoc 'acc 'rowvals true))
					'(list)
					'('lambda '('acc 'sharddict) '('insert schema grouptbl (cons 'list (map group_keys expr_name)) '('extract_assoc 'sharddict '('lambda '('k 'v) 'k)) '(list) '('lambda '() true) true))
					isOuter))))
		/* aggregate descriptors */
		(define agg_col_name (lambda (ag) (concat (expr_name ag) "|" (expr_name condition))))
		(define fk_child_col (if is_fk_reuse (if has_partition (match (car group_keys) '('get_column _ false scol false) scol) nil) nil))
		(define ags (map wf_resolved (lambda (wf) (match wf '(fn args _) (begin
			/* args already resolved via replace_find_column in wf_resolved */
			(define map_expr (if (equal? fn "COUNT") 1 (if (nil? args) 1 (car args))))
			(define sep (if (and (equal? fn "GROUP_CONCAT") (> (count args) 1)) (cadr args) ","))
			(match fn "SUM" (list map_expr '+ 0) "COUNT" (list 1 '+ 0) "MIN" (list map_expr 'min nil) "MAX" (list map_expr 'max nil)
				"GROUP_CONCAT" (list '('concat map_expr) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil)
				(error (concat "unsupported aggregate window function: " fn))))))))
		/* createcolumn on KEYTABLE */
		(define agg_plans (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
			(define cols (extract_columns_for_tblvar tblvar expr))
			'('createcolumn schema grouptbl (agg_col_name ag) "any" '(list) '(list "temp" true)
				(cons list (map group_keys (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
				'('lambda (map group_keys (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
					(scan_wrapper 'scan schema tbl
						(cons list (merge tblvar_cols filtercols))
						'('lambda (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons 'and (cons (replace_columns_from_expr condition) (map group_keys (lambda (col) '('equal? (replace_columns_from_expr col) '('outer (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
						(cons list cols)
						'('lambda (map cols (lambda (col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr expr))
						reduce neutral nil isOuter))))))))
		(define compute_plan (cons 'parallel agg_plans))
		/* replace window_func with scalar fetch */
		(define replace_wf_with_fetch (lambda (expr) (match expr
			(cons (symbol window_func) wf_rest) (begin
				(define wf_fn (car wf_rest))
				(define wf_args (cadr wf_rest))
				(define map_expr (if (equal? wf_fn "COUNT") 1 (if (nil? wf_args) 1 (replace_find_column (car wf_args)))))
				(define sep (if (and (equal? wf_fn "GROUP_CONCAT") (> (count wf_args) 1)) (cadr wf_args) ","))
				(define ag_col (agg_col_name (match wf_fn "SUM" (list map_expr '+ 0) "COUNT" (list 1 '+ 0) "MIN" (list map_expr 'min nil) "MAX" (list map_expr 'max nil)
					"GROUP_CONCAT" (list '('concat map_expr) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil)
					(list map_expr '+ 0))))
				(if has_partition (begin
					(define kt_key_names (map group_keys (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
					/* outer refs need raw column names (tblvar.col), not canonical expr_name */
					(define raw_col_names (map group_keys (lambda (col) (match col '('get_column _ _ c _) c (expr_name col)))))
					(list 'scan schema grouptbl
						(cons 'list kt_key_names)
						/* filter: (equal? grouptbl.kt_key (outer tblvar.raw_col)) — zip kt_key_names with raw_col_names */
						(list 'lambda
							(map kt_key_names (lambda (kn) (symbol (concat grouptbl "." kn))))
							(cons 'and (map (produceN (count kt_key_names) (lambda (i) i)) (lambda (i)
								(list 'equal? (symbol (concat grouptbl "." (nth kt_key_names i))) (list 'outer (symbol (concat tblvar "." (nth raw_col_names i)))))))))
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false))
					(list 'scan schema grouptbl '(list) '('lambda '() true)
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false)))
			(cons sym args_) (cons sym (map args_ replace_wf_with_fetch))
			expr)))
		(define new_fields (map_assoc fields (lambda (k v) (replace_wf_with_fetch (replace_find_column v)))))
		(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil nil))
		(list 'begin keytable_init '('time collect_plan "collect") '('time compute_plan "compute") scan_plan)))
)))

/* make_col_replacer: create a function that rewrites column/aggregate references to point at a group table
is_dedup=true: leave aggregates intact (for dedup stages)
is_dedup=false: replace aggregates with column fetches (for normal group stages) */
(define make_col_replacer (lambda (grouptbl condition is_dedup expr_name src_tblvar) (begin
	(define colname (lambda (expr) (if (nil? expr_name) (concat expr) (expr_name expr))))
	(define replacer (lambda (expr) (match expr
		(cons (symbol aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (concat (colname rest) "|" (colname condition)) false))
		'((symbol get_column) src_tblvar ti col ci) '('get_column grouptbl ti (colname '('get_column src_tblvar ti col ci)) ci)
		/* rewrite (outer tblvar.col) inside scalar subselects to reference keytable column */
		'('outer sym) (begin
			(define symStr (concat sym))
			(define prefix (concat src_tblvar "."))
			(define prefixLen (strlen prefix))
			(if (and (>= (strlen symStr) prefixLen) (equal? (substr symStr 0 prefixLen) prefix))
				(begin
					(define col (substr symStr prefixLen (- (strlen symStr) prefixLen)))
					(define gc_expr '('get_column src_tblvar false col false))
					(define kt_col (colname gc_expr))
					'('outer (symbol (concat grouptbl "." kt_col))))
				expr))
		(cons sym args) (cons sym (map args replacer))
		expr
	)))
	replacer
)))

/* rewrite_for_prejoin: rewrite (get_column tblvar _ col _) -> (get_column pjvar false "tblvar.col" false) for prejoin materialization */
(define rewrite_for_prejoin (lambda (pjvar expr)
	(match expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) expr
			'('get_column pjvar false (concat tblvar "." col) false))
		(cons sym args) (cons sym (map args (lambda (a) (rewrite_for_prejoin pjvar a))))
		expr
	)
))

/* replace_tblvar_with_dict: replace (get_column tv _ col _) refs for a specific tv
with (list 'get_assoc dict_sym col) — for use in building trigger body S-expressions */
(define replace_tblvar_with_dict (lambda (tv dict_sym expr)
	(match expr
		'((symbol get_column) tblvar _ col _)
		(if (equal? tblvar tv)
			(list 'get_assoc dict_sym col)
			expr)
		'((quote get_column) tblvar _ col _)
		(if (equal? tblvar tv)
			(list 'get_assoc dict_sym col)
			expr)
		(cons sym args) (cons sym (map args (lambda (a) (replace_tblvar_with_dict tv dict_sym a))))
		expr
	)
))

/* build_pj_insert_scan: build the nested-scan S-expression for an INSERT trigger on trigger_tv.
Skips scanning trigger_tv (its cols come from (get_assoc NEW "col") at runtime),
scans all other tables, and inserts matching rows into pj_schema/pjtbl.
pj_schema, pjtbl, mat_cols, mat_col_names are passed explicitly to avoid free-variable capture issues.
Returns an S-expression that, when wrapped in (lambda (OLD NEW) ...) and eval'd, performs the insert. */
(define build_pj_insert_scan (lambda (scan_tables scan_condition trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
	(match scan_tables
		(cons '(tblvar schema tbl isOuter _) rest)
		(if (equal? tblvar trigger_tv)
			/* skip trigger table: replace its refs in condition with (get_assoc NEW col) and recurse */
			(build_pj_insert_scan rest
				(replace_tblvar_with_dict trigger_tv 'NEW scan_condition)
				trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
			/* scan this other table */
			(begin
				(set cols (merge_unique (list
					(extract_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc))))))))
				(match (split_condition (coalesceNil scan_condition true) rest) '(now_condition later_condition) (begin
					(set filtercols (merge_unique (list
						(extract_columns_for_tblvar tblvar now_condition)
						(extract_outer_columns_for_tblvar tblvar now_condition))))
					(list 'scan schema tbl
						(cons 'list filtercols)
						/* filter lambda: (lambda (tv.col ...) compiled_condition) */
						(list 'lambda (map filtercols (lambda (c) (symbol (concat tblvar "." c))))
							(optimize (replace_columns_from_expr now_condition)))
						(cons 'list cols)
						/* map lambda: (lambda (tv.col ...) recursive_inner_scan) */
						(list 'lambda (map cols (lambda (c) (symbol (concat tblvar "." c))))
							(build_pj_insert_scan rest later_condition trigger_tv false pj_schema pjtbl mat_cols mat_col_names))
						/* reduce: merge */
						(list 'lambda (list 'acc 'sub) (list 'merge 'acc 'sub))
						(list)
						/* reduce2: outermost inserts into pjtbl, inner levels merge */
						(if is_outermost
							(list 'lambda (list 'acc 'shard_rows)
								(list 'insert pj_schema pjtbl (cons 'list mat_col_names) 'shard_rows (list) (list 'lambda (list) true) true))
							(list 'lambda (list 'acc 'shard_rows) (list 'merge 'acc 'shard_rows)))
						isOuter)
				))
			)
		)
		/* base case: all tables processed. Produce one row with trigger_tv cols from NEW.
		replace_columns_from_expr converts remaining (get_column ...) refs to symbol variable refs. */
		(list 'if (optimize (replace_columns_from_expr (coalesceNil scan_condition true)))
			(list 'list (cons 'list
				(map mat_cols (lambda (mc)
					(match (cadr mc)
						'((symbol get_column) tv _ col _)
						(if (equal? tv trigger_tv)
							(list 'get_assoc 'NEW col)
							(symbol (concat tv "." col)))
						'((quote get_column) tv _ col _)
						(if (equal? tv trigger_tv)
							(list 'get_assoc 'NEW col)
							(symbol (concat tv "." col)))
						/* fallback: replace trigger_tv refs and convert to symbol */
						(replace_tblvar_with_dict trigger_tv 'NEW (replace_columns_from_expr (cadr mc))))))))
			(list))
	)
))

/*
=== CONTRACT: untangle_query ===

PURPOSE: Flatten a parsed SQL query into a logical structure (Neumann unnesting).
Recursively resolves derived tables, expression subqueries, and stages.
Goal: all correlated subqueries become LEFT JOIN LIMIT 1 table entries
so the output is a flat table list with predicates — no nested runtime code.

INPUT:  parsed query tuple (schema tables fields condition group having order limit offset outer_schemas_param)
OUTPUT: 7-tuple (schema tables fields condition groups schemas replace_find_column)

WHAT IT DOES:
- Flattens FROM (SELECT ...) derived tables into the parent tables list
- Unnests correlated scalar/IN/EXISTS subqueries into flat table entries
(currently via build_scalar_subselect which produces inline runtime code;
the goal is LEFT JOIN LIMIT 1 table entries instead)
- Pushes domain columns through GROUP BY barriers when decorrelating
(Neumann: D ⋈ Γ_A(T) == Γ_{A∪D}(D ⋈ T))
- Canonicalizes get_column markers (ti/ci → false/false)
- Collects group/having/order/limit/offset into a stages pipeline

WHAT IT MUST NOT DO:
- Choose join order (that is join_reorder's job)
- Create keytables or aggregate infrastructure (that is build_queryplan's job)
*/
(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas_param) (begin
	/* TODO: multiple group levels, limit+offset for each group level */
	(set rename_prefix (coalesce rename_prefix ""))
	(define outer_schemas_chain (coalesceNil outer_schemas_param '()))
	/* sq_cache: memoize scalar subselect plans */
	(define sq_cache (newsession))

	/* Accumulator for unnested subselect tables/schemas (Neumann unnesting) */
	(define unnest_acc (newsession))
	(unnest_acc "tables" '())
	(unnest_acc "schemas" '())
	(unnest_acc "counter" 0)

	/* unnest_subselect: converts a scalar subselect into a materialized derived table.
	Returns a (get_column $sqN false "col" false) substitution expression.
	The materialized table and its schema are accumulated in unnest_acc
	and merged into the outer query's table list after all inner_selects are processed.
	Neumann/Kemper BTW 2015: subquery → derived table with LEFT JOIN. */
	/* _has_outer_refs: check if an expression tree contains (outer ...) references
	or get_column references with case-insensitive flags that might resolve to outer tables */
	(define _has_outer_refs (lambda (expr outer_schemas) (match expr
		'((symbol outer) _) true
		'((quote outer) _) true
		'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci))
			/* check if the alias resolves in outer_schemas but not in inner schemas — delegate to caller */
			true
			false)
		(cons sym args) (if (not (nil? (inner_select_kind sym))) false /* don't descend into nested subselects */
			(reduce args (lambda (a b) (or a (_has_outer_refs b outer_schemas))) false))
		false
	)))
	/* _subquery_is_correlated: check if any part of a raw subquery references outer tables.
	Conservative: assumes correlated if uncertain (nested inner_selects, unqualified columns
	that might resolve to outer scope). False negatives (missing a correlation) cause panics,
	false positives (assuming correlated when not) only lose the optimization. */
	(define _subquery_is_correlated (lambda (subquery outer_schemas) (begin
		(define raw_fields (if (and (list? subquery) (>= (count subquery) 3)) (nth subquery 2) nil))
		(define raw_condition (if (and (list? subquery) (>= (count subquery) 4)) (nth subquery 3) nil))
		(define raw_tables (if (and (list? subquery) (>= (count subquery) 2)) (nth subquery 1) nil))
		/* collect inner table aliases */
		(define inner_aliases (if (list? raw_tables)
			(map raw_tables (lambda (t) (match t '(alias _ _ _ _) alias _ nil)))
			'()))
		/* check if fields or condition contain outer refs, nested subselects, or
		unqualified columns that might resolve to outer scope */
		(define _check_refs (lambda (expr) (match expr
			/* qualified get_column: correlated if alias NOT in inner tables */
			'((symbol get_column) alias_ ti col ci) (if (nil? alias_) false
				(not (reduce inner_aliases (lambda (a ia) (or a ((if ti equal?? equal?) alias_ ia))) false)))
			'((symbol outer) _) true
			'((quote outer) _) true
			/* nested inner_selects may reference outer tables at deeper levels;
			we can't cheaply check, so conservatively assume correlated */
			(cons sym args) (if (not (nil? (inner_select_kind sym))) true
				(if (or (equal?? sym (quote !begin)) (equal?? sym (symbol !begin))) false
					(reduce args (lambda (a b) (or a (_check_refs b))) false)))
			false
		)))
		(define fields_corr (if raw_fields (reduce_assoc raw_fields (lambda (a k v) (or a (_check_refs v))) false) false))
		(define cond_corr (if raw_condition (_check_refs raw_condition) false))
		/* also check for unqualified get_columns that might be outer references:
		if outer_schemas is non-empty and there are unqualified columns, be conservative */
		(define _has_unqualified (lambda (expr) (match expr
			'((symbol get_column) nil _ col _) true
			(cons sym args) (if (or (not (nil? (inner_select_kind sym))) (equal?? sym (quote !begin)) (equal?? sym (symbol !begin))) false
				(reduce args (lambda (a b) (or a (_has_unqualified b))) false))
			false
		)))
		(define has_unqual_cols (and (not (equal? outer_schemas '()))
			(or
				(if raw_fields (reduce_assoc raw_fields (lambda (a k v) (or a (_has_unqualified v))) false) false)
				(if raw_condition (_has_unqualified raw_condition) false))))
		(or fields_corr cond_corr has_unqual_cols)
	)))

	/* Domain extraction helpers for Neumann decorrelation */
	(define _is_gc_sym (lambda (sym) (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))))
	(define _gc_alias (lambda (expr) (match expr
		(cons sym '(alias_ ti col ci)) (if (and (_is_gc_sym sym) (not (nil? alias_))) alias_ nil)
		_ nil)))
	(define _gc_col (lambda (expr) (match expr
		(cons sym '(alias_ ti col ci)) (if (_is_gc_sym sym) col nil)
		_ nil)))
	(define _gc_in_aliases (lambda (expr aliases) (begin
		(define a (_gc_alias expr))
		(if (nil? a) false
			(reduce aliases (lambda (acc ia) (or acc (equal?? a ia))) false)))))
	(define _gc_not_in_aliases (lambda (expr aliases) (begin
		(define a (_gc_alias expr))
		(if (nil? a) false
			(not (reduce aliases (lambda (acc ia) (or acc (equal?? a ia))) false))))))
	(define _flat_and (lambda (expr) (match expr
		(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym '(symbol and)))
			(merge (map parts _flat_and))
			(list expr))
		(list expr))))
	(define _rebuild_and (lambda (parts) (match parts
		'() true (cons only '()) only _ (cons (quote and) parts))))
	(define _expr_has_outer_ref (lambda (expr inner_aliases) (match expr
		(cons sym args) (if (or (not (nil? (inner_select_kind sym))) (_is_precomputed sym)) false
			(if (_gc_not_in_aliases expr inner_aliases) true
				(reduce args (lambda (a b) (or a (_expr_has_outer_ref b inner_aliases))) false)))
		false)))

	/* _is_precomputed: check if a symbol is !begin (pre-computed subselect value, opaque to analysis) */
	(define _is_precomputed (lambda (sym) (match sym
		'!begin true '(quote !begin) true (symbol !begin) true _ false)))

	(define unnest_subselect (lambda (subquery outer_schemas) (begin
		/* Step 1: recursive unnesting — replace nested inner_selects BEFORE
		checking correlation. This flattens the subquery so that nested
		non-correlated scalars become inline !begin values, and the correlation
		check sees only direct references to outer tables. */
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			/* UNION ALL scalar subselect: not supported, fall back */
			(build_scalar_subselect subquery outer_schemas)
			(begin
				/* extract raw components */
				(define raw_schema (nth subquery 0))
				(define raw_tables (coalesceNil (nth subquery 1) '()))
				(define raw_fields (coalesceNil (nth subquery 2) '()))
				(define raw_condition (if (>= (count subquery) 4) (nth subquery 3) nil))

				/* build combined outer_schemas for recursive calls:
				inner_selects inside this subquery see both the actual outer tables
				AND this subquery's own tables as "outer" for correlation analysis */
				(define inner_aliases (if (list? raw_tables)
					(map raw_tables (lambda (t) (match t '(alias _ _ _ _) alias _ nil)))
					'()))
				(define combined_schemas (merge (coalesceNil outer_schemas '())
					(merge (map inner_aliases (lambda (alias) (list alias '()))))))

				/* walk expression tree and recursively unnest scalar inner_selects.
				TODO (Neumann next step): process ALL inner_selects including correlated.
				With full Neumann, walk_replace calls unnest_subselect unconditionally;
				the correlated case produces a domain-decorrelated derived table.
				This enables recursive decorrelation of arbitrarily nested correlated subqueries. */
				(define walk_replace (lambda (expr) (match expr
					(cons sym args) (begin
						(define kind (inner_select_kind sym))
						(if (equal?? kind (quote inner_select))
							(match args
								(cons sq '()) (if (_subquery_is_correlated sq combined_schemas)
									expr /* leave correlated inner_selects for build_scalar_subselect */
									(unnest_subselect sq combined_schemas))
								_ (cons sym (map args walk_replace)))
							/* leave IN/EXISTS for their own handlers */
							(if (not (nil? kind))
								expr
								(cons sym (map args walk_replace)))))
					expr)))

				(define new_fields (map_assoc raw_fields (lambda (k v) (walk_replace v))))
				(define new_condition (if (nil? raw_condition) nil (walk_replace raw_condition)))

				/* rebuild modified subquery with replaced inner_selects */
				(define modified_subquery (list raw_schema raw_tables new_fields new_condition
					(if (>= (count subquery) 5) (nth subquery 4) nil)
					(if (>= (count subquery) 6) (nth subquery 5) nil)
					(if (>= (count subquery) 7) (nth subquery 6) nil)
					(if (>= (count subquery) 8) (nth subquery 7) nil)
					(if (>= (count subquery) 9) (nth subquery 8) nil)))

				/* Step 2: correlation check on the MODIFIED subquery.
				Since nested inner_selects have been replaced with !begin (opaque),
				this only finds direct references to outer tables. */
				(if (_subquery_is_correlated modified_subquery outer_schemas)
					/* correlated: try Neumann domain decorrelation for aggregate scalars */
					(begin
						(define _val_keys (extract_assoc new_fields (lambda (k v) k)))
						(define _val_exprs (extract_assoc new_fields (lambda (k v) v)))
						(define _value_col (match _val_keys (cons only '()) only _ nil))
						(define _value_expr (if _value_col (car _val_exprs) nil))
						(define _is_agg_sym (lambda (sym) (or (equal? sym (quote aggregate)) (equal? sym '(quote aggregate)) (equal? sym '(symbol aggregate)))))
						(define _agg_info (if _value_expr (match _value_expr
							(cons sym '(item reduce neutral)) (if (_is_agg_sym sym) (list item reduce neutral) nil)
							_ nil) nil))
						/* classify AND-parts as domain equi-joins, inner conditions, or fail */
						(define _cond_parts (if (nil? new_condition) '() (_flat_and new_condition)))
						(define _is_eq_sym (lambda (sym) (or (equal?? (string sym) "equal??") (equal?? (string sym) "equal?"))))
						(define _classified (map _cond_parts (lambda (part) (match part
							(cons sym '(a b)) (if (_is_eq_sym sym)
								(if (and (_gc_in_aliases a inner_aliases) (_gc_not_in_aliases b inner_aliases))
									(list "domain" a b)
									(if (and (_gc_not_in_aliases a inner_aliases) (_gc_in_aliases b inner_aliases))
										(list "domain" b a)
										(if (_expr_has_outer_ref part inner_aliases) (list "fail" part) (list "inner" part))))
								(if (_expr_has_outer_ref part inner_aliases) (list "fail" part) (list "inner" part)))
							_ (if (_expr_has_outer_ref part inner_aliases) (list "fail" part) (list "inner" part))))))
						(define _has_fail (reduce _classified (lambda (a c) (or a (equal? (car c) "fail"))) false))
						(define _domain_joins (if _has_fail '()
							(filter (map _classified (lambda (c) (if (equal? (car c) "domain") (list (nth c 1) (nth c 2)) nil))) (lambda (x) (not (nil? x))))))
						/* can we decorrelate? need: aggregate, domain joins found, no failures,
						and if outer has GROUP BY, domain cols must be expressible via group keys
						(Neumann: D ⋈ Γ_{A;f} ≡ Γ_{A∪D;f}(D ⋈ T) requires D compatible with A) */
						(define _outer_has_incompatible_group (and (not (nil? group)) (not (equal? group '()))))
						(if (or (nil? _agg_info) _has_fail (equal? _domain_joins '()) (nil? _value_col) _outer_has_incompatible_group)
							(build_scalar_subselect subquery outer_schemas)
							/* TODO (Neumann next steps to eliminate this fallback):

							1. Non-aggregate correlated scalars (e.g. SELECT val FROM t2 WHERE owner = t1.id):
							Use promise as aggregate for scalar enforcement in the keytable/createcolumn path:
							neutral = (newpromise)
							reduce = (lambda (a b) (begin (a "once" b "scalar subselect returned more than one row") a))
							Read via (promise "value") in the substitution.
							This fits the existing createcolumn infrastructure: one promise per GROUP BY domain key.

							2. Incompatible outer GROUP BY (e.g. SELECT COUNT(*), (SELECT SUM(...) WHERE ...) GROUP BY customer):
							The domain table is joined BEFORE the GROUP BY in the scan pipeline, so there is no
							conflict. Remove the _outer_has_incompatible_group guard. The Neumann equivalence
							D |><| Gamma_{A;f}(T) = Gamma_{A+D;f}(D |><| T) handles this naturally.

							3. Equi-join symbol detection (_is_eq_sym):
							The Scheme symbol equal?? cannot be reliably compared using the equal?? function
							itself (self-reference/collision). Use match patterns like inner_select_kind does:
							'((symbol equal??) a b), '('equal?? a b), '((quote equal??) a b) etc.
							OR use (list? part) + (count part) + car/nth instead of (cons sym '(a b)) match.

							4. Unqualified column handling (_gc_in_aliases):
							Change (if (nil? a) false ...) to (if (nil? a) true ...) so that unqualified
							columns (no table alias) are assumed to be inner table columns. This is safe
							because unqualified columns resolve to inner tables during column resolution.

							5. ORDER BY + LIMIT 1 per domain (crop1 pattern):
							For subqueries like (SELECT file FROM rev WHERE doc = d.id ORDER BY created DESC LIMIT 1),
							the LIMIT 1 applies per domain value, not globally. Use GROUP BY domain with
							a promise-based aggregate (same as non-aggregate case) — the keytable/createcolumn
							scan_order per group key naturally produces the first value per domain.

							6. Once all cases are handled, remove build_scalar_subselect entirely
							(except the UNION ALL fallback). All scalar subselects go through unnest_subselect.
							*/
							(begin
								(define sq_num (unnest_acc "counter"))
								(unnest_acc "counter" (+ sq_num 1))
								(define sq_id (concat "$sq" sq_num))
								/* build remaining inner condition */
								(define _inner_parts (filter (map _classified (lambda (c) (if (equal? (car c) "inner") (nth c 1) nil))) (lambda (x) (not (nil? x)))))
								(define remaining_cond (_rebuild_and _inner_parts))
								/* add domain columns to fields and GROUP BY */
								(define domain_field_entries (merge (map _domain_joins (lambda (dj) (list (_gc_col (car dj)) (car dj))))))
								(define domain_group_cols (map _domain_joins (lambda (dj) (car dj))))
								(define raw_group (if (>= (count subquery) 5) (nth subquery 4) nil))
								(define new_group (merge (coalesceNil raw_group '()) domain_group_cols))
								/* build decorrelated subquery */
								(define decorrelated_subquery (list raw_schema raw_tables
									(merge new_fields domain_field_entries) remaining_cond
									new_group
									(if (>= (count subquery) 6) (nth subquery 5) nil)
									(if (>= (count subquery) 7) (nth subquery 6) nil)
									nil nil))
								/* materialize into real temp table (compatible with prejoin GROUP BY path) */
								(define temp_tbl_name (concat ".unnest:" sq_id))
								(define all_col_names (merge (list _value_col) (map _domain_joins (lambda (dj) (_gc_col (car dj))))))
								(createtable schema temp_tbl_name
									(map all_col_names (lambda (col) (list "column" col "any" '() '())))
									'("engine" "sloppy") true)
								(define resultrow_sym (symbol (concat "__unnest_rr:" sq_id)))
								(define materialize_code (list (quote begin)
									(list (quote set) resultrow_sym (symbol "resultrow"))
									(list (quote set) (symbol "resultrow")
										(list (quote lambda) (list (symbol "item"))
											(list (quote insert) schema temp_tbl_name
												(cons (quote list) all_col_names)
												(list (quote list) (cons (quote list)
													(map all_col_names (lambda (col) (list (quote get_assoc) (symbol "item") col)))))
												'(list) '('lambda '() true) true)))
									(build_queryplan_term decorrelated_subquery)
									(list (quote set) (symbol "resultrow") resultrow_sym)))
								/* joinexpr: match domain columns */
								(define domain_join_conds (map _domain_joins (lambda (dj)
									(list (quote equal??) (nth dj 1) (list (quote get_column) sq_id false (_gc_col (car dj)) false)))))
								(define joinexpr (match domain_join_conds (cons only '()) only _ (cons (quote and) domain_join_conds)))
								/* schema */
								(define sq_schema_cols (merge (list (list "Field" _value_col "Type" "any"))
									(map _domain_joins (lambda (dj) (list "Field" (_gc_col (car dj)) "Type" "any")))))
								/* register: use string table name, prepend materialize_code to unnest_acc init */
								(unnest_acc "tables" (merge (unnest_acc "tables") (list (list sq_id schema temp_tbl_name true joinexpr))))
								(unnest_acc "schemas" (merge (unnest_acc "schemas") (list sq_id sq_schema_cols)))
								/* register init code for materialization (executed before scan by build_queryplan) */
								(define guarded_init (list (quote if) (list (quote equal?) 0 (list (quote scan_estimate) schema temp_tbl_name))
									materialize_code nil))
								(unnest_acc "init" (merge (coalesceNil (unnest_acc "init") '()) (list guarded_init)))
								/* substitution: pure query term */
								(define agg_neutral (nth _agg_info 2))
								(list (quote coalesceNil) (list (quote get_column) sq_id false _value_col false) agg_neutral)
						))
					)
					(begin
						/* Step 3: non-correlated — pre-compute via build_queryplan_term */
						(define sq_num (unnest_acc "counter"))
						(unnest_acc "counter" (+ sq_num 1))
						(define sq_id (concat "$sq" sq_num))

						(define resultrow_sym (symbol (concat "__unnest_rr:" sq_id)))
						(define promise_sym (symbol (concat "__unnest_promise:" sq_id)))

						/* inline pre-computation: evaluate modified subquery, capture scalar value */
						(list (quote !begin)
							(list (quote set) promise_sym (list (quote newpromise)))
							(list (quote set) resultrow_sym (symbol "resultrow"))
							(list (quote set) (symbol "resultrow")
								(list (quote lambda) (list (symbol "item"))
									(list promise_sym "once"
										(list (quote nth) (symbol "item") 1)
										"scalar subselect returned more than one row")))
							(build_queryplan_term modified_subquery)
							(list (quote set) (symbol "resultrow") resultrow_sym)
							(list promise_sym "value"))
				))
		))
	)))


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
		/* force optimizer to retain both params by using them directly in the outer body */
		(define _s schemas2)
		(define _o outer_schemas)
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
		/* wrap_outer_leaves: replace get_column leaf nodes with (outer tblvar.col) symbol references
		so that derived-table computed columns are accessible via the optimizer's outer-scope mechanism */
		(define is_get_column_sym (lambda (sym)
			(or (equal? sym (quote get_column))
				(equal? sym '(quote get_column))
				(equal? sym '(symbol get_column))
			)
		))
		/* canonical_column_in_schema: returns the Field name as stored in the schema (canonical casing) */
		(define canonical_column_in_schema (lambda (schemas alias_name table_insensitive column_name column_insensitive)
			(reduce_assoc schemas (lambda (acc alias cols)
				(if (not (nil? acc)) acc
					(if (or (nil? alias_name) ((if table_insensitive equal?? equal?) alias_name alias))
						(reduce cols (lambda (found coldef)
							(if (not (nil? found)) found
								(if ((if column_insensitive equal?? equal?) (coldef "Field") column_name) (coldef "Field") nil))) nil)
						nil))
			) nil)
		))
		(define wrap_outer_leaves (lambda (expr) (match expr
			(cons sym args) (if (is_get_column_sym sym)
				(match args
					'(tblvar ti col ci) (if (nil? tblvar) expr (begin
						(define canonical (coalesce (canonical_column_in_schema _o tblvar ti col ci) col))
						(list (quote outer) (symbol (concat tblvar "." canonical)))))
					_ (cons (wrap_outer_leaves sym) (map args wrap_outer_leaves))
				)
				(cons (wrap_outer_leaves sym) (map args wrap_outer_leaves))
			)
			expr
		)))
		(define replace_get_column_subselect (lambda (alias_name table_insensitive column_name column_insensitive expr) (begin
			(define inner_alias (column_exists_in_schema _s alias_name table_insensitive column_name column_insensitive))
			(define inner_alias_exists (and (not (nil? alias_name)) (alias_exists_in_schema _s alias_name table_insensitive)))
			(if (and inner_alias_exists (nil? inner_alias))
				(error (concat "column " alias_name "." column_name " does not exist in subquery"))
				(if (not (nil? inner_alias))
					(if (or (nil? alias_name) table_insensitive column_insensitive)
						(begin
							(define inner_column (coalesce (canonical_column_in_schema _s alias_name table_insensitive column_name column_insensitive) column_name))
							'((quote get_column) inner_alias false inner_column false))
						expr)
					(begin
						(define outer_alias (column_exists_in_schema _o alias_name table_insensitive column_name column_insensitive))
						(if (nil? outer_alias)
							(if (nil? alias_name)
								(error (concat "column " column_name " does not exist in outer query"))
								expr)
							(begin
								/* check if the outer column is a computed expression (derived table) */
								(define outer_column (coalesce (canonical_column_in_schema _o alias_name table_insensitive column_name column_insensitive) column_name))
								(define outer_cols (_o outer_alias))
								(define outer_coldef (reduce outer_cols (lambda (a coldef) (if (and (nil? a) (equal? (coldef "Field") outer_column)) coldef a)) nil))
								(define outer_expr (if outer_coldef (outer_coldef "Expr") nil))
								(if outer_expr
									/* derived table computed column: inline expression with leaf get_column
									nodes replaced by (outer sym) references for optimizer resolution */
									(wrap_outer_leaves outer_expr)
									/* real table column: symbol lookup in outer scope */
									(list (quote outer) (symbol (concat outer_alias "." outer_column))))))
					)
				)
			)
		)))
		(define replace_find_column_subselect (lambda (expr) (match expr
			(cons sym args) (if (is_get_column_sym sym)
				(match args
					'(alias_name table_insensitive column_name column_insensitive) (replace_get_column_subselect alias_name table_insensitive column_name column_insensitive expr)
					_ (cons sym (map args replace_find_column_subselect))
				)
				/* canonicalize (outer tbl.col) symbols: normalize col to schema casing */
				(if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
					(match args
						(cons outer_sym '()) (begin
							(define _ps (split (string outer_sym) "."))
							(match _ps
								(list _tbl _col) (begin
									(define _canonical (coalesce (canonical_column_in_schema _o _tbl true _col true) _col))
									(if (equal? _col _canonical) expr
										(list (if (equal? sym (quote outer)) (quote outer) sym) (symbol (concat _tbl "." _canonical)))))
								_ (cons sym (map args replace_find_column_subselect))))
						_ (cons sym (map args replace_find_column_subselect)))
					(cons sym (map args replace_find_column_subselect)))
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
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2 _)
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
						(set fields2 (map_assoc fields2 (lambda (k v) (replace_find_column_subselect v))))
						(set condition2 (replace_find_column_subselect (coalesceNil condition2 true)))
						/* wrap remaining unresolved qualified get_column refs as (outer tbl.col).
						These are outer-outer refs that weren't in _s or _o — wrapping them
						preserves them through replace_columns_from_expr and allows
						replace_column_alias to prefix them during derived-table flattening. */
						(define wrap_unresolved_outer (lambda (e) (match e
							'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci))
								(list (quote outer) (symbol (concat alias_ "." col)))
								e)
							(cons sym args) (cons (wrap_unresolved_outer sym) (map args wrap_unresolved_outer))
							e
						)))
						(set fields2 (map_assoc fields2 (lambda (k v) (wrap_unresolved_outer v))))
						(set condition2 (wrap_unresolved_outer condition2))
						/* detect top-level aggregate for direct scan path */
						(define value_expr_rep (car (extract_assoc fields2 (lambda (k v) v))))
						(define _is_aggregate_sym (lambda (sym)
							(or (equal? sym (quote aggregate))
								(equal? sym '(quote aggregate))
								(equal? sym '(symbol aggregate))
						)))
						(define _agg_head (match value_expr_rep (cons sym _) sym _ nil))
						(define _agg_args (if (and _agg_head (_is_aggregate_sym _agg_head))
							(match value_expr_rep (cons _ args) args _ nil)
							nil))
						(define has_stage2 (and (not (nil? groups2)) (not (equal? groups2 '()))))
						(define stage2 (if has_stage2 (car groups2) nil))
						(define stage2_group (if stage2 (coalesceNil (stage_group_cols stage2) '()) '()))
						(define stage2_having (if stage2 (stage_having_expr stage2) nil))
						/* use direct scan: single top-level aggregate, no HAVING, no GROUP keys, has tables */
						(define use_direct_agg_scan (and
							(not (nil? _agg_args))
							(equal? (count _agg_args) 3)
							(nil? stage2_having)
							(or (nil? stage2_group) (equal? stage2_group '()) (equal? stage2_group '(1)))
							(not (nil? tables2))
							(not (equal? tables2 '()))
						))
						(if use_direct_agg_scan
							/* direct nested-scan aggregate: avoids build_queryplan keytable path */
							(begin
								(define agg_item (nth _agg_args 0))
								(define agg_reduce (nth _agg_args 1))
								(define agg_neutral (nth _agg_args 2))
								(define build_scalar_agg_scan (lambda (scan_tables scan_condition)
									(match scan_tables
										(cons '(tblvar schema3 tbl3 isOuter3 _) rest_tables) (begin
											(define cur_cols (merge_unique (list
												(extract_columns_for_tblvar tblvar scan_condition)
												(extract_columns_for_tblvar tblvar agg_item)
												(extract_outer_columns_for_tblvar tblvar scan_condition)
												(extract_outer_columns_for_tblvar tblvar agg_item)
											)))
											(match (split_condition (coalesceNil scan_condition true) rest_tables) '(now_condition later_condition) (begin
												(define filtercols (merge_unique (list
													(extract_columns_for_tblvar tblvar now_condition)
													(extract_outer_columns_for_tblvar tblvar now_condition)
												)))
												(define inner_body (build_scalar_agg_scan rest_tables later_condition))
												(scan_wrapper 'scan schema3 tbl3
													(cons list filtercols)
													(list (quote lambda)
														(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
														(replace_columns_from_expr now_condition)
													)
													(cons list cur_cols)
													(list (quote lambda)
														(map cur_cols (lambda (col) (symbol (concat tblvar "." col))))
														inner_body
													)
													(eval agg_reduce) agg_neutral (eval agg_reduce) isOuter3
												)
											))
										)
										'() (replace_columns_from_expr agg_item)
									)
								))
								(build_scalar_agg_scan tables2 condition2)
							)
							/* fallback: build_queryplan for non-aggregate or complex aggregate cases */
							(begin
								/* hash of inner query after column-resolution — used as dedup key and unique name suffix */
								(define _sq_hash (fnv_hash (concat tables2 "|" fields2 "|" condition2)))
								(define _sq_promise_name (concat "__scalar_promise_" _sq_hash))
								(define _sq_rr_name  (concat "__scalar_resultrow_" _sq_hash))
								(begin
									(define replace_resultrow (lambda (expr) (match expr
										(cons sym args) (if (equal? sym (quote resultrow))
											(cons (symbol _sq_rr_name) (map args replace_resultrow))
											(if (and (equal? sym (quote symbol)) (equal? args '("resultrow")))
												(list (quote symbol) _sq_rr_name)
												(cons (replace_resultrow sym) (map args replace_resultrow))
											)
										)
										expr
									)))
									(define subplan (replace_resultrow (build_queryplan schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column_subselect nil nil)))
									(define _init_stmts (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
									(cons (quote !begin) (merge _init_stmts (list
										(list (quote set) (symbol _sq_promise_name) (list (quote newpromise)))
										(list (quote set) (symbol _sq_rr_name)
											(list (quote lambda) (list (symbol "row"))
												(list (symbol _sq_promise_name) "once"
													(list (quote nth) (symbol "row") 1)
													"scalar subselect returned more than one row")
											)
										)
										subplan
										(list (symbol _sq_promise_name) "value")
									)))
								)
							)
						) /* close fallback begin */
					) /* close if use_direct_agg_scan */
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
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2 _)
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
								/* OR-reduce for semi-join membership test */
								(define in_reduce (list (quote lambda) (list (symbol "acc") (symbol "v"))
									(list (quote or) (quote acc) (quote v))
								))
								(define in_neutral false)
								(define use_ordered (or (and (not (nil? stage_order)) (not (equal? stage_order '()))) (not (nil? stage_limit)) (not (nil? stage_offset))))
								(if (and use_ordered (not (equal? (count tables2) 1)))
									(error "IN subselect ORDER BY with multiple tables not supported yet")
								)
								/* for single-table ordered case: extract sort columns from the one table */
								(define tblvar0 (nth (car tables2) 0))
								(define ordercols (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar0) (list col) '())
									'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar0) (list col) '())
									_ '()
								))))))
								(define dirs (merge (map stage_order (lambda (order_item) (match order_item '(col dir) (match col
									'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar0) (list dir) '())
									'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar0) (list dir) '())
									_ '()
								))))))
								(if (and use_ordered (not (equal? stage_order '())) (not (equal? (count ordercols) (count stage_order))))
									(error "IN subselect ORDER BY must use direct columns")
								)
								/* recursive nested-scan builder: push dependent semi-join down through tables (Neumann unnesting) */
								(define build_in_scan (lambda (scan_tables scan_condition)
									(match scan_tables
										(cons '(tblvar schema3 tbl3 isOuter3 _) rest_tables) (begin
											/* columns from this table needed at this and inner scan levels */
											(define cur_cols (merge_unique (list
												(merge_unique (cons
													(extract_columns_for_tblvar tblvar scan_condition)
													(list (extract_columns_for_tblvar tblvar value_expr))
												))
												(merge_unique (cons
													(extract_outer_columns_for_tblvar tblvar scan_condition)
													(list (extract_outer_columns_for_tblvar tblvar value_expr))
												))
											)))
											/* split condition: evaluate now vs defer to inner tables */
											(match (split_condition (coalesceNil scan_condition true) rest_tables) '(now_condition later_condition) (begin
												(define cur_filtercols (merge_unique (list
													(extract_columns_for_tblvar tblvar now_condition)
													(extract_outer_columns_for_tblvar tblvar now_condition)
												)))
												(if (and use_ordered (equal? rest_tables '()))
													/* single-table ordered path */
													(list (quote scan_order)
														schema3 tbl3
														(cons list cur_filtercols)
														(list (quote lambda)
															(map cur_filtercols (lambda (col) (symbol (concat tblvar "." col))))
															(optimize (replace_columns_from_expr now_condition))
														)
														(cons list ordercols)
														(cons list dirs)
														0
														(coalesceNil stage_offset 0)
														(coalesceNil stage_limit -1)
														(cons list cur_cols)
														(list (quote lambda)
															(map cur_cols (lambda (col) (symbol (concat tblvar "." col))))
															(list (quote equal??) (replace_columns_from_expr target_expr) (replace_columns_from_expr value_expr))
														)
														in_reduce
														in_neutral
													)
													/* unordered path: nest scans (Neumann dependent join push-down) */
													(scan_wrapper 'scan schema3 tbl3
														(cons list cur_filtercols)
														(list (quote lambda)
															(map cur_filtercols (lambda (col) (symbol (concat tblvar "." col))))
															(optimize (replace_columns_from_expr now_condition))
														)
														(cons list cur_cols)
														(list (quote lambda)
															(map cur_cols (lambda (col) (symbol (concat tblvar "." col))))
															(build_in_scan rest_tables later_condition)
														)
														in_reduce
														in_neutral
														in_reduce
														isOuter3
													)
												)
											))
										)
										/* base case: all tables visited, test membership equality */
										'() (list (quote equal??) (replace_columns_from_expr target_expr) (replace_columns_from_expr value_expr))
									)
								))
								(build_in_scan tables2 condition2)
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
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2 _)
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
								(define filtercols (merge_unique (list
									(extract_columns_for_tblvar tblvar condition2)
									(extract_outer_columns_for_tblvar tblvar condition2))))
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
										0
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
						(cons subquery '()) (unnest_subselect subquery outer_schemas)
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
			/* merge unnested subselect tables/schemas into result */
			(define extra_tables (unnest_acc "tables"))
			(define extra_schemas (unnest_acc "schemas"))
			(define _init_code (coalesceNil (unnest_acc "init") '()))
			(if (equal? extra_tables '())
				(list schema tables fields condition groups '() (lambda (expr) expr) _init_code nil)
				(list schema extra_tables fields condition groups extra_schemas (lambda (expr) expr) _init_code nil))
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
											(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
								)
								(build_queryplan_term subquery)
								(list (quote set) (symbol "resultrow") resultrow_sym)
								(list rows_sym "rows")
							))
							(define _mat_var (symbol (concat "__mat:" id)))
							(unnest_acc "init" (merge (coalesceNil (unnest_acc "init") '())
								(list (list (quote set) _mat_var materialized_rows))))
							(list
								(list (list id schemax _mat_var isOuter joinexpr))
								'()
								true
								(list id (map output_cols (lambda (col) '("Field" col "Type" "any"))))
							)
						))
						(match (apply untangle_query subquery) '(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2 _) (begin
							/* helper function add prefix to tblalias of every expression */
							(define replace_column_alias (lambda (expr) (match expr
								'((symbol get_column) nil ti col ci) (begin
									/* resolve unqualified column against inner schemas2; must match exactly one table.
									Skip aliases that contain \0 (null byte) — those are prefixed from flattened derived tables
									and should not participate in unqualified column resolution. */
									(define matches (reduce_assoc schemas2 (lambda (acc alias cols)
										(if (and (equal? (replace alias "\0" "") alias)
											(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
											(cons alias acc)
											acc)) '()))
									(match matches
										(cons only '()) '('get_column (concat id "\0" only) ti col ci)
										'() (begin
											/* column not in schemas2 - check if it's a SELECT alias in fields2 */
											(if (nil? (fields2 col))
												expr /* leave unresolved — inner subselect scope will handle it */
												/* found in fields2 - resolve to the underlying expression */
												(replace_column_alias (fields2 col))
											)
										)
										(cons _ _) (error (concat "ambiguous column " col " in subquery"))
									)
								)
								'((symbol get_column) alias_ ti col ci) (if (not (nil? (schemas2 alias_)))
									'('get_column (concat id "\0" alias_) ti col ci)
									expr) /* alias not in schemas2 → inner subselect scope, leave as-is */
								'((symbol outer) outer_arg) (begin
									/* prefix outer variable reference if it refers to a table in schemas2 */
									(define s (string outer_arg))
									(define parts (split s "."))
									(match parts
										(list tbl col) (if (not (nil? (schemas2 tbl)))
											(list (quote outer) (symbol (concat id "\0" tbl "." col)))
											(list (quote outer) outer_arg))
										_ (list (quote outer) (replace_column_alias outer_arg))
									)
								)
								(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym))) expr /* inner subselects resolved later by replace_inner_selects */ (cons (replace_column_alias sym) (map args replace_column_alias)))
								expr
							)))
							/* prefix all table aliases and transform their joinexprs */
							(set tablesPrefixed (map tables2 (lambda (x) (match x '(alias schema tbl a innerJoinexpr)
								(list (concat id "\0" alias) schema tbl a (if (nil? innerJoinexpr) nil (replace_column_alias innerJoinexpr)))))))
							/* helper function to transform joinexpr: only transform references to subquery alias id */
							(define transform_joinexpr (lambda (expr) (match expr
								'((symbol get_column) alias_ ti col ci) (if (equal?? alias_ id)
									/* reference to subquery alias -> resolve against inner schemas by passing nil alias */
									(replace_column_alias (list (quote get_column) nil ti col ci))
									/* reference to outer table -> keep as-is */
									expr)
								(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym))) expr /* inner subselects have their own scope */ (cons sym (map args transform_joinexpr)))
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
							/* window functions in subquery require materialization (cannot flatten because window needs its own ordering) */
							(define subquery_has_window (not (equal? (merge (extract_assoc fields2 (lambda (k v) (extract_window_funcs v)))) '())))
							/* TODO: group+order+limit+offset -> ordered scan list with aggregation layers (to avoid materialization) */
							/* Note: flat defines avoid nested begin scopes — (set) only updates the innermost Nodefine=false env */
							(define groups2_present (and (not (nil? groups2)) (not (equal? groups2 '()))))
							(define unsupported_groups (if groups2_present
								(reduce groups2 (lambda (acc stage)
									(or acc
										(begin
											(define g (stage_group_cols stage))
											(and (not (nil? g)) (not (equal? g '())))
										)
										(not (nil? (stage_having_expr stage)))
										(not (nil? (stage_limit_val stage)))
										(not (nil? (stage_offset_val stage)))
									)
								) false)
								false))
							(define use_materialize (or subquery_has_window unsupported_groups))
							/* Window-function LIMIT pushdown */
							(define mat_limit nil)
							(if subquery_has_window (begin
								(define _check_wf_limit (lambda (cond) (match cond
									'('<= '('get_column _ _ col _) n) (if (and (not (nil? (get_assoc fields2 col))) (not (equal? (extract_window_funcs (get_assoc fields2 col)) '())))
										(set mat_limit n) nil)
									'('< '('get_column _ _ col _) n) (if (and (not (nil? (get_assoc fields2 col))) (not (equal? (extract_window_funcs (get_assoc fields2 col)) '())))
										(set mat_limit (- n 1)) nil)
									'('and a b) (begin (_check_wf_limit a) (_check_wf_limit b))
									nil)))
								(_check_wf_limit condition)
							))
							/* if groups2 had only pass-through stages (no GROUP/HAVING/LIMIT/OFFSET), strip them for flattening */
							(if (and groups2_present (not unsupported_groups))
								(set groups2 nil))
							(if use_materialize
								(begin
									(define output_cols_sub (extract_assoc fields2 (lambda (k v) k)))
									(define rows_sym (symbol (concat "__from_subquery_rows:" id)))
									(define resultrow_sym (symbol (concat "__from_subquery_resultrow:" id)))
									/* Materialization: collect rows from build_queryplan_term into a list.
								   Use a unique resultrow name (__mat_rr:id) so that replace_resultrow in
								   build_scalar_subselect does NOT accidentally replace the collector —
								   otherwise correlated scalar subselects break because the inner
								   query's resultrow gets rewritten to the promise handler. */
								(define mat_rr_sym (symbol (concat "__mat_rr:" id)))
								(define mat_inner_plan (build_queryplan_term subquery))
								/* Replace resultrow → mat_rr_sym in the inner plan, so the inner
								   query feeds into our collector instead of the outer resultrow */
								(define replace_rr_mat (lambda (expr) (match expr
									(cons sym args) (if (equal? sym (quote resultrow))
										(cons mat_rr_sym (map args replace_rr_mat))
										(if (and (equal? sym (quote symbol)) (equal? args '("resultrow")))
											(list (quote symbol) (concat "__mat_rr:" id))
											(cons (replace_rr_mat sym) (map args replace_rr_mat))))
									expr)))
								(define mat_inner_plan (replace_rr_mat mat_inner_plan))
								(define materialized_rows (list (quote begin)
										(list (quote set) rows_sym (list (quote newsession)))
										(list rows_sym "rows" '())
										(define cnt_sym (symbol (concat "__from_subquery_cnt:" id)))
										(if (nil? mat_limit)
											/* no limit */
											(list (quote set) mat_rr_sym
												(list (quote lambda) (list (symbol "item"))
													(list rows_sym "rows"
														(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
											)
											/* with limit: stop collecting after mat_limit rows */
											(list (quote begin)
												(list (quote set) cnt_sym 0)
												(list (quote set) mat_rr_sym
													(list (quote lambda) (list (symbol "item"))
														(list (quote if) (list (quote <) cnt_sym mat_limit)
															(list (quote begin)
																(list (quote set) cnt_sym (list (quote +) cnt_sym 1))
																(list rows_sym "rows"
																	(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
															nil))))
										)
										mat_inner_plan
										(list rows_sym "rows")
									))
									/* Store materialized rows in a variable via init code.
									   The variable (a symbol) is used as tbl — at runtime,
									   scan evaluates the symbol to get the row list. */
									(define _mat_var (symbol (concat "__mat:" id)))
									(unnest_acc "init" (merge (coalesceNil (unnest_acc "init") '())
										(list (list (quote set) _mat_var materialized_rows))))
									(list
										(list (list id schemax _mat_var isOuter joinexpr))
										'()
										true
										(list id (map output_cols_sub (lambda (col) '("Field" col "Type" "any"))))
									)
								)
								(begin
									/* for LEFT JOIN: condition2 was integrated into joinexpr, so return true as global filter */
									/* for INNER JOIN: condition2 becomes global filter (can be reordered) */
									(set globalFilter (if isOuter true (replace_column_alias condition2)))
									(define _check_inner_select (lambda (expr) (match expr (cons sym args) (if (not (nil? (inner_select_kind sym))) true (reduce args (lambda (a b) (or a (_check_inner_select b))) false)) false)))
									(define wrap_outer_join_projection (lambda (expr)
										(if (and isOuter (not (equal? joinexpr true)) (not (nil? joinexpr2)) (not (equal? joinexpr2 true)) (not (_check_inner_select joinexpr2)))
											(list (quote if) joinexpr2 expr nil)
											expr)))
									(list tablesPrefixed (list id (map_assoc fields2 (lambda (k v) (wrap_outer_join_projection (replace_column_alias v))))) globalFilter (merge (list id (extract_assoc fields2 (lambda (k v) (list "Field" k "Type" "any" "Expr" (replace_column_alias v))))) (merge (extract_assoc schemas2 (lambda (k v) (list (concat id "\0" k) v))))))
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

			/* TODO: add rename_prefix to all table names and get_column expressions */
			/* TODO: apply renamelist to all expressions in fields condition group having order */

			/* at first: extract additional join exprs into condition list */
			/* Outer-to-inner join conversion: if the WHERE clause references any column
			   of a LEFT JOIN table, that LEFT JOIN can be safely converted to INNER JOIN
			   because NULL-padded rows would be rejected by the WHERE anyway. */
			(set tables (map tables (lambda (t) (match t
				'(a s tbl true je) (if (has? (extract_tblvars condition) a)
					(list a s tbl false je) /* WHERE references this table -> convert LEFT to INNER */
					t)
				t))))
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
						(define tblalias_str (if (string? tblalias) tblalias (string tblalias)))
						(define alias_lookup
							(coalesce
								(if (has_assoc? schemas tblalias_str) tblalias_str nil)
								nil))
						(define cols (if (nil? alias_lookup) nil (schemas alias_lookup)))
						(define coldef (if (list? cols)
							(reduce cols (lambda (a coldef)
								(if (or a (equal?? (coldef "Field") colname)) a coldef)
							) nil)
							nil))
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
					(begin
						(define resolved_alias (coalesce any_match (error (concat "column " col " does not exist in tables"))))
						(define canonical_col (if ci (coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col) col))
						'((quote get_column) resolved_alias false canonical_col false))
				)
				'((symbol get_column) alias_ ti col ci) (if (or ti ci)
					(begin
						(define resolved_alias (coalesce (reduce_assoc schemas (lambda (a alias cols) (if (and ((if ti equal?? equal?) alias_ alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)) alias a)) nil) (error (concat "column " alias_ "." col " does not exist in tables"))))
						(define canonical_col (if ci (coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col) col))
						'((quote get_column) resolved_alias false canonical_col false))
					expr) /* omit false false, otherwise freshly created columns wont be found */
				(cons sym args) /* function call */ (cons sym (map args replace_find_column))
				expr
			)))

			(set fields (map_assoc fields (lambda (k v) (replace_inner_selects v schemas))))
			/* Snapshot condition string BEFORE inner_select expansion for stable
			   prejoin/keytable naming. After expansion, the condition contains
			   promise/scan code that inflates canonical names. */
			(define condition_canonical_pre_expansion (concat condition))
			(set condition (replace_inner_selects condition schemas))
			(set group (map group (lambda (g) (replace_inner_selects g schemas))))
			(set having (replace_inner_selects having schemas))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col schemas) dir)))))
			/* Also resolve inner_selects in joinexprs so build_scan can use them directly */
			(set tables (map tables (lambda (t) (match t
				'(a s tbl io je) (list a s tbl io (if (nil? je) nil (replace_inner_selects je schemas)))
				t))))
			/* merge unnested subselect tables/schemas into outer query (Neumann unnesting) */
			(set tables (merge tables (unnest_acc "tables")))
			(set schemas (merge schemas (unnest_acc "schemas")))
			/* integrate joinexprs from unnested domain tables into condition */
			(set condition (reduce (unnest_acc "tables") (lambda (cond t) (match t '(_ _ _ _ je) (if (nil? je) cond (list (quote and) cond je)) cond)) condition))

			/* canonicalize_for_rename: resolve case-insensitive column names to canonical form,
			but ONLY for columns referencing derived table aliases (keys in renamelist).
			Uses schemas to find canonical column name without calling replace_find_column. */
			(define canonicalize_for_rename (lambda (expr) (match expr
				'((symbol get_column) alias_ ti col ci) (if (and ci (not (nil? alias_)))
					(if (has_assoc? renamelist (string alias_))
						(begin
							(define alias_cols (schemas (string alias_)))
							(define canonical_col (if (nil? alias_cols) col
								(coalesce (reduce alias_cols (lambda (found coldef)
									(if (not (nil? found)) found
										(if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col)))
							'((quote get_column) alias_ ti canonical_col ci))
						expr)
					expr)
				(cons sym args) (cons sym (map args canonicalize_for_rename))
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
				(list col (replace_find_column expr))
			)))))

			/* return parameter list for build_queryplan */
			(set conditionAll (cons 'and (filter (cons (replace_rename (canonicalize_for_rename condition)) conditionList) (lambda (x) (not (nil? x)))))) /* TODO: append inner conditions to condition */
			(set group (map group (lambda (g) (replace_rename (canonicalize_for_rename g)))))
			(set having (replace_rename (canonicalize_for_rename having)))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_rename (canonicalize_for_rename col)) dir)))))
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
			/* canonicalize all get_column markers: resolve ti/ci flags to canonical casing.
			After this, all get_column nodes have false false — no case ambiguity remains. */
			(define _canon (lambda (expr) (canonicalize_columns expr schemas)))
			(define _canon_fields (map_assoc fields (lambda (k v) (_canon (replace_rename v)))))
			(define _canon_condition (_canon conditionAll))
			(define _canon_groups (map (coalesceNil groups '()) (lambda (stage) (canonicalize_stage stage schemas))))
			/* eliminate unused LEFT JOINs: a LEFT JOIN is unused when none of its
			columns appear in fields or group stages. Tables whose WHERE references
			were converted to INNER JOIN above are no longer isOuter, so they won't
			be eliminated. Unnested table aliases must never be pruned. */
			(define _unnested_aliases (map (unnest_acc "tables") (lambda (t) (match t '(alias _ _ _ _) alias _ nil))))
			(define _used_tvs (merge_unique
				_unnested_aliases
				(merge (extract_assoc _canon_fields (lambda (k v) (extract_tblvars v))))
				(merge (map _canon_groups (lambda (stage)
					(merge_unique
						(merge (map (coalesceNil (stage_group_cols stage) '()) extract_tblvars))
						(extract_tblvars (coalesceNil (stage_having_expr stage) true))
						(merge (map (coalesceNil (stage_order_list stage) '()) (lambda (o) (match o '(col dir) (extract_tblvars col) (extract_tblvars o)))))))))))
			(define _pruned_tables (filter tables (lambda (t) (match t
				'(alias _ _ isOuter _) (if isOuter (has? _used_tvs alias) true)
				true))))
			/* rebuild condition: drop AND-parts that reference ONLY eliminated aliases */
			(define _elim_aliases (filter (map tables (lambda (t) (match t
				'(alias _ _ true _) (if (has? _used_tvs alias) nil alias)
				nil))) (lambda (x) (not (nil? x)))))
			(define _canon_condition (if (equal? (count _pruned_tables) (count tables)) _canon_condition
				(begin
					/* flatten nested (and ...) to get individual condition parts */
					(define _flatten_and (lambda (expr)
						(match expr (cons (symbol and) parts) (merge (map parts _flatten_and))
							(list expr))))
					(define _cond_parts (_flatten_and _canon_condition))
					/* drop condition parts that reference ANY eliminated alias */
					(define _kept_parts (filter _cond_parts (lambda (part)
						(not (reduce (extract_tblvars part) (lambda (acc tv) (or acc (has? _elim_aliases tv))) false)))))
					(if (equal? 0 (count _kept_parts)) true
						(if (equal? 1 (count _kept_parts)) (car _kept_parts)
							(cons 'and _kept_parts))))))
			(list schema _pruned_tables _canon_fields _canon_condition _canon_groups schemas replace_find_column (coalesceNil (unnest_acc "init") '()) condition_canonical_pre_expansion)
		)
	)
)
))

/*
=== CONTRACT: join_reorder ===

PURPOSE: Optimize table order for physical scan execution.
Determines which table to scan first in a nested-loop join based on
table sizes, available indexes, and predicate selectivity.
Pure physical optimization — does not change query semantics.

INPUT/OUTPUT: 7-tuple (schema tables fields condition groups schemas replace_find_column)

WHAT IT MAY DO:
- Reorder tables within a barrier-free scan segment

WHAT IT MUST NOT DO:
- Transform query structure (that is untangle_query's job)
- Decorrelate subqueries or create joins (that is untangle_query's job)
- Build physical scan plans (that is build_queryplan's job)
- Reorder tables across a join fence (LEFT/SEMI/ANTI JOIN boundary)
*/
/* currently a stub — preserves original table order */
(define join_reorder (lambda (schema tables fields condition groups schemas replace_find_column)
	(list schema tables fields condition groups schemas replace_find_column)))

(define build_queryplan_term (lambda (query) (begin
	(define union_parts (query_union_all_parts query))
	(if (nil? union_parts)
		(if (query_is_select_core query)
			(begin
				(define _uq_result (apply untangle_query (merge query (list nil))))
				(define _uq_init (if (>= (count _uq_result) 8) (nth _uq_result 7) '()))
				(define _uq_cond_pre (if (>= (count _uq_result) 9) (nth _uq_result 8) nil))
				(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
				(define _plan (apply build_queryplan (merge (apply join_reorder _uq_7tuple) (list nil _uq_cond_pre))))
				(if (equal? _uq_init '()) _plan (cons (quote begin) (merge _uq_init (list _plan)))))
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

/* build_dml_plan: route UPDATE/DELETE through the full query planner pipeline.
schema: target schema
target_tbl: target table name (the table being modified)
target_alias: alias of target table (or nil → uses target_tbl)
all_defs: list of table definitions ((alias schema tblname isOuter joinexpr) ...)
cols: flat assoc list (col1 expr1 col2 expr2 ...) for UPDATE, or nil/() for DELETE
condition: WHERE clause expression (raw, not pre-resolved)
order: ORDER BY list or nil
limit_val: LIMIT value or nil
offset_val: OFFSET value or nil
The pipeline resolves inner_selects in SET expressions, handles JOINs, subselects,
column resolution — then injects $update into the target table's scan. */
(define build_dml_plan (lambda (schema target_tbl target_alias all_defs cols condition order limit_val offset_val) (begin
	(define tgt (coalesce target_alias target_tbl))
	(define is_update (and (not (nil? cols)) (not (equal? cols '()))))
	/* For UPDATE: put SET expressions into synthetic fields so untangle_query processes them
	(including replace_inner_selects for scalar subselects).
	For DELETE: fields are empty — just the tables + condition. */
	(define set_fields (if is_update
		(begin
			(define col_names (extract_assoc cols (lambda (k v) k)))
			(define col_vals (extract_assoc cols (lambda (k v) v)))
			(merge (map (produceN (count col_names)) (lambda (i)
				(list (concat "$set:" (nth col_names i)) (nth col_vals i))))))
		'("$dml_dummy" 1))) /* need at least one field for the pipeline to work */
	/* Build synthetic SELECT 9-tuple: (schema tables fields condition group having order limit offset) */
	(define synthetic_select (list schema all_defs set_fields condition nil nil order limit_val offset_val))
	/* Run through untangle_query → join_reorder → build_queryplan */
	(define _dml_uq (apply untangle_query (merge synthetic_select (list nil))))
	(define _dml_init (if (>= (count _dml_uq) 8) (nth _dml_uq 7) '()))
	(define _dml_cond_pre (if (>= (count _dml_uq) 9) (nth _dml_uq 8) nil))
	(define pipeline_result (apply join_reorder (list (nth _dml_uq 0) (nth _dml_uq 1) (nth _dml_uq 2) (nth _dml_uq 3) (nth _dml_uq 4) (nth _dml_uq 5) (nth _dml_uq 6))))
	/* For UPDATE: reconstruct resolved cols from the pipeline's fields */
	(define resolved_target_cols (if is_update
		(begin
			(define resolved_fields (nth pipeline_result 2))
			(define cnames (extract_assoc cols (lambda (k v) k)))
			(merge (map cnames (lambda (cn) (begin
				(define set_key (concat "$set:" cn))
				/* Use a mutable flag (newsession) to track if match was found,
				avoiding equality check on the sentinel (0 == "__not_found__" is buggy) */
				(define _found (newsession))
				(_found "v" nil)
				(reduce_assoc resolved_fields (lambda (acc k v) (if (equal?? k set_key) (begin (_found "v" v) (_found "hit" true) v) acc)) nil)
				(list cn (if (_found "hit") (_found "v") (list (quote get_column) nil false cn false)))
		)))))
		'())) /* DELETE: empty cols signals deletion */
	/* Assemble final pipeline args: fields are dummy (not used for output), update_target has the real SET cols */
	(define final_pipeline (list
		(nth pipeline_result 0) /* schema */
		(nth pipeline_result 1) /* tables */
		'("$dml" 1) /* dummy fields — not used for output in DML mode */
		(nth pipeline_result 3) /* condition */
		(nth pipeline_result 4) /* groups */
		(nth pipeline_result 5) /* schemas */
		(nth pipeline_result 6) /* replace_find_column */
		(list tgt resolved_target_cols) /* update_target: (alias cols) — empty cols = DELETE */
		_dml_cond_pre /* condition_canonical_pre_expansion */
	))
	(define _dml_plan (apply build_queryplan final_pipeline))
	(if (equal? _dml_init '()) _dml_plan (cons (quote begin) (merge _dml_init (list _dml_plan))))
)))

/* Convenience wrapper for multi-table UPDATE (called from sql_update) */
(define build_multi_table_update (lambda (schema tbl tblalias all_defs cols condition)
	(build_dml_plan schema tbl tblalias all_defs cols condition nil nil nil)))

/*
=== CONTRACT: build_queryplan ===

PURPOSE: Generate physical execution plans from the logical structure.
Takes a flat, already-reordered table list and builds executable plans.

INPUT:  7-tuple (schema tables fields condition groups schemas replace_find_column)
After join_reorder, tables are in optimal scan order.

OUTPUT: executable Scheme expression (scan, keytable operations, resultrow, etc.)

WHAT IT DOES:
- Resolves get_column markers to variable references via replace_find_column
- Processes GROUP BY stages: creates keytables, collect/compute/grouped plans
- Processes ORDER BY / LIMIT: generates scan_order with offset/limit
- Generates nested scan loops via build_scan (follows table order from join_reorder)
- Handles window functions (ORC, aggregate, LAG/LEAD)

WHAT IT MUST NOT DO:
- Reorder tables (that is join_reorder's job)
- Flatten derived tables or unnest subqueries (that is untangle_query's job)

GROUP BY AGGREGATE PIPELINE:
1. collect_plan: extract unique group keys from base table into a keytable
2. compute_plan: for each aggregate, scan base table per group key,
store results as keytable columns named "expr|condition"
3. grouped_plan: scan populated keytable for final output (ORDER BY, HAVING, LIMIT)
*/
/* update_target: nil for SELECT, or (tblalias (col1 expr1 col2 expr2 ...)) for multi-table UPDATE.
   When set, the scan on tblalias includes $update in mapcols and the mapfn applies the SET expressions. */
(define build_queryplan (lambda (schema tables fields condition groups schemas replace_find_column update_target condition_canonical_pre_expansion) (begin
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

	/* window function detection */
	(define window_funcs_all (merge (extract_assoc fields (lambda (k v) (extract_window_funcs v)))))
	(define has_window (not (equal? window_funcs_all '())))
	/* Case 10: window functions in WHERE clause */
	(define window_in_condition (not (equal? (extract_window_funcs (coalesceNil condition true)) '())))
	(if window_in_condition (error "window functions not allowed in WHERE clause"))

	/* window functions with GROUP BY: strip window expressions to inner
	aggregates so the normal GROUP BY path processes them. Save original
	fields so we can inject promise values after compute_plan. */
	(define _wg_store (newsession))
	(_wg_store "fields" nil)
	(if (and has_window stage_group) (begin
		(_wg_store "fields" fields) /* save original fields with window_func */
		(define strip_window_inner (lambda (expr)
			(if (and (list? expr) (> (count expr) 0) (equal?? (car expr) (quote window_func)))
				(begin (define args (nth expr 2))
					(if (and (list? args) (> (count args) 0)) (car args) 1))
				(if (list? expr) (map expr strip_window_inner) expr))))
		(set fields (map_assoc fields (lambda (k v) (strip_window_inner v))))
		(set has_window false)))

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
				(define canon_alias_map (list (list tblvar (concat schema "." tbl))))
				(define expr_name (lambda (expr) (canonical_expr_name expr '(list) '(list) canon_alias_map)))

				(define kt_result (make_keytable schema tbl stage_group tblvar (if is_dedup condition nil)))
				(set grouptbl (car kt_result))
				(define keytable_init (car (cdr kt_result)))
				(define fk_pk_col (car (cdr (cdr kt_result))))
				(define is_fk_reuse (not (nil? fk_pk_col)))

				/* preparation */
				(define tblvar_cols (merge_unique (map stage_group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesceNil condition true)))
				(set filtercols (merge_unique (list
					(extract_columns_for_tblvar tblvar condition)
					(extract_outer_columns_for_tblvar tblvar condition))))

				/* make_collect: builds collect plan with optional WHERE filter
				with_filter=true: apply WHERE condition (for DEDUP)
				with_filter=false: collect ALL group keys (for NORMAL) */
				(define make_collect (lambda (with_filter)
					'('time '('begin
						/* If grouping is global (group='(1)), avoid base scan and insert one key row */
						(if (equal? stage_group '(1))
							'('insert schema grouptbl '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
							(begin
								/* key columns */
								(set keycols (merge_unique (map stage_group (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
								(scan_wrapper 'scan schema tbl
									(if with_filter (cons list filtercols) '(list))
									(if with_filter
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
										'((quote lambda) '() true))
									(cons list keycols)
									'((quote lambda)
										(map keycols (lambda (col) (symbol (concat tblvar "." col))))
										(cons (quote list) (map stage_group (lambda (expr) (replace_columns_from_expr expr))))) /* build records '(k1 k2 ...) */
									'((quote lambda) '('acc 'rowvals) '('set_assoc 'acc 'rowvals true)) /* add keys to assoc; each key is a dataset -> unique filtering */
									'(list) /* empty dict */
									'((quote lambda) '('acc 'sharddict)
										'('insert
											schema grouptbl
											(cons 'list (map stage_group expr_name))
											'('extract_assoc 'sharddict '('lambda '('k 'v) 'k)) /* turn keys from assoc into list */
											'(list) '('lambda '() true) true)
									)
									isOuter)
							)
						)
					) "collect")))

				(if is_dedup (begin
					/* DEDUP-ONLY stage: no aggregate computation, just collect unique keys and pass through to next stage */
					(define replace_col_for_dedup (make_col_replacer grouptbl condition true expr_name tblvar))
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
						replace_find_column
						nil nil))
					(list 'begin keytable_init (make_collect true) grouped_plan)
				) (begin
						/* NORMAL group stage: extract aggregates, compute, and continue.
						replace_agg_with_fetch rewrites (aggregate expr + 0) -> (get_column grouptbl "expr|cond")
						so ORDER BY SUM(amount) becomes ORDER BY on a keytable column. */
						(define replace_agg_with_fetch (make_col_replacer grouptbl condition false expr_name tblvar))
						(define agg_col_name (lambda (ag) (concat (expr_name ag) "|" (expr_name condition))))
						(define replace_group_key_or_fetch (lambda (expr) (if
							(reduce stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
							'('get_column grouptbl false (if is_fk_reuse fk_pk_col (expr_name expr)) false)
							(replace_agg_with_fetch expr)
						)))

						(define grouped_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (replace_group_key_or_fetch col) dir))))))
						(define next_groups (merge
							(if (coalesce grouped_order stage_limit stage_offset) (list (make_group_stage nil nil grouped_order stage_limit stage_offset)) '())
							rest_groups
						))
						/* FK reuse: extract child FK column name */
						(define fk_child_col (if is_fk_reuse
							(match (car stage_group) '('get_column _ false scol false) scol)
							nil))
						/* COUNT column replaces the old exists column: always add (1 + 0) to ags
						so we have a row count per group. HAVING filters on COUNT > 0 to exclude
						empty groups after DELETE. This also enables incremental DELETE maintenance. */
						(define needs_count (not (equal? stage_group '(1))))
						(define count_ag '(1 + 0))
						(define ags (if needs_count (merge_unique ags (list count_ag)) ags))
						(define count_col_name (if needs_count (agg_col_name count_ag) nil))

						/* AND count>0 into HAVING so empty/non-matching groups are excluded */
						(define effective_having (if needs_count
							(begin
								(define count_check '('> '('get_column grouptbl false count_col_name false) 0))
								(define replaced_having (replace_group_key_or_fetch stage_having))
								(if (or (nil? replaced_having) (equal? replaced_having true))
									count_check
									(list 'and replaced_having count_check)))
							(replace_group_key_or_fetch stage_having)))

						(define grouped_plan (build_queryplan schema '('(grouptbl schema grouptbl false nil))
							(map_assoc fields (lambda (k v) (replace_group_key_or_fetch v)))
							effective_having
							next_groups
							schemas
							replace_find_column
							nil nil))

						/* createcolumn options: filter by COUNT column so only groups with rows are computed */
						(define createcol_options (cons 'list (merge '("temp" true)
							(if needs_count
								(list "filtercols" (list 'list count_col_name)
									"filter" '((quote lambda) (list (symbol count_col_name)) '('> (symbol count_col_name) 0)))
								'()))))

						(define agg_plans (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
							(set cols (merge_unique (list
								(extract_columns_for_tblvar tblvar expr)
								(extract_outer_columns_for_tblvar tblvar expr)
							)))
							/* COUNT column itself must not filter by itself (circular); all others filter by COUNT>0 */
							(define this_options (if (and needs_count (equal? (agg_col_name ag) count_col_name)) '(list "temp" true) createcol_options))
							'((quote createcolumn) schema grouptbl (agg_col_name ag) "any" '(list) this_options
								(cons list (map stage_group (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
								'((quote lambda) (map stage_group (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
									(scan_wrapper 'scan schema tbl
										(cons list (merge tblvar_cols filtercols))
										/* check group equality AND WHERE-condition */
										'((quote lambda) (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons (quote and) (cons (replace_columns_from_expr condition) (map stage_group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
										(cons list cols)
										'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr expr))
										reduce
										neutral
										nil
										isOuter
									)
							))
						)))))

						(define compute_plan
							'('time (cons 'parallel agg_plans) "compute"))

						/* invalidation is handled by registerComputeTriggers in ComputeColumn:
						DML triggers on the base table invalidate computed columns automatically.
						No forced invalidation needed here — the createcolumn/ComputeColumn path
						skips recompute when the proxy is still valid (no DML since last compute). */
						(define invalidation_plan nil)

						/* build key column pairs for keytable cleanup triggers: ((base_col kt_col) ...) */
						(define key_pairs (map stage_group (lambda (expr)
							(match expr
								'((symbol get_column) _ _ col _) (list col (expr_name expr))
								'((quote get_column) _ _ col _) (list col (expr_name expr))
								(list (expr_name expr) (expr_name expr))
						))))
						(define cleanup_plan (if (or is_fk_reuse (equal? stage_group '(1))) nil
							(list 'register_keytable_cleanup schema tbl schema grouptbl tblvar
								(cons 'list (map key_pairs (lambda (p) (list 'list (car p) (cadr p))))))))

						(list 'begin keytable_init cleanup_plan
							(if is_fk_reuse nil
								(list 'if (list 'equal? 0 (list 'scan_estimate schema grouptbl))
									(make_collect false)
									nil))
							invalidation_plan compute_plan
							/* window+GROUP BY injection: after keytable is computed,
							scan it to fill promises with global totals, then wrap
							grouped_plan's resultrow to inject promise values. */
							(if (nil? (_wg_store "fields")) grouped_plan
								(begin
									(define _wg_ctr (newsession)) (_wg_ctr "n" 0)
									(define _wg_nn (lambda () (begin (_wg_ctr "n" (+ (_wg_ctr "n") 1)) (concat "__wgp_" (_wg_ctr "n")))))
									/* build promise info and output fields with promise refs */
									(define _wg_pl (newsession)) (_wg_pl "l" '())
									(define _wg_out_fields (map_assoc (_wg_store "fields") (lambda (k v)
										(if (and (list? v) (> (count v) 0) (equal?? (car v) (quote window_func)))
											(begin
												(define pn (_wg_nn))
												(define wfn (nth v 1))
												(define wargs (nth v 2))
												(define inner_agg (if (and (list? wargs) (> (count wargs) 0)) (car wargs) 1))
												(define agg_tuple (match inner_agg (cons (symbol aggregate) rest) rest (list inner_agg (quote +) 0)))
												(define acn (agg_col_name agg_tuple))
												(_wg_pl "l" (cons (list pn acn wfn) (_wg_pl "l")))
												(symbol pn))
											v))))
									/* scan keytable for each promise: aggregate the column globally */
									(define _wg_scans (map (_wg_pl "l") (lambda (pi) (match pi '(pn acn wfn)
										(begin
											(define reduce_op (match wfn "SUM" (quote +) "COUNT" (quote +) "MIN" (quote min) "MAX" (quote max) (quote +)))
											(define neutral (match wfn "SUM" 0 "COUNT" 0 "MIN" nil "MAX" nil 0))
											(list (quote set) (symbol pn)
												'('scan schema grouptbl
													'(list acn)
													'('lambda (list (symbol acn)) true)
													'(list acn)
													'('lambda (list (symbol acn)) (symbol acn))
													reduce_op
													neutral
													nil false)))))))
									/* wrap grouped_plan: replace resultrow to inject promise values */
									(define _wg_rr_body (cons (quote list) (merge (extract_assoc _wg_out_fields (lambda (k v)
										(if (not (list? v))
											(list k v)
											(list k (list (quote get_assoc) (symbol "__wgr") k))))))))
									(cons 'begin (merge _wg_scans (list
										(list (quote set) (symbol "__wg_orig_rr") (symbol "resultrow"))
										(list (quote set) (symbol "resultrow")
											(list (quote lambda) (list (symbol "__wgr"))
												(list (symbol "__wg_orig_rr") _wg_rr_body)))
										grouped_plan))))))
				))
			)
			(begin /* multi-table GROUP BY via prejoin materialization */
				(if is_dedup (error "DISTINCT on joined tables not yet supported"))
				/* resolve condition and fields */
				(set condition (replace_find_column (coalesceNil condition true)))
				(define resolved_fields (map_assoc fields (lambda (k v) (replace_find_column v))))
				/* extract all get_column AND outer refs from group, fields, having, order, condition.
			   outer refs come from scalar subselects that were expanded by replace_inner_selects. */
				(define mat_cols_raw (merge
					(merge (map stage_group extract_all_get_columns))
					(merge (map stage_group extract_all_outer_columns))
					(merge (extract_assoc resolved_fields (lambda (k v) (extract_all_get_columns v))))
					(merge (extract_assoc resolved_fields (lambda (k v) (extract_all_outer_columns v))))
					(if (nil? stage_having) '() (extract_all_get_columns stage_having))
					(if (nil? stage_having) '() (extract_all_outer_columns stage_having))
					(merge (map (coalesce stage_order '()) (lambda (o) (match o '(col dir) (extract_all_get_columns col)))))
					(merge (map (coalesce stage_order '()) (lambda (o) (match o '(col dir) (extract_all_outer_columns col)))))
					(extract_all_get_columns condition)
					(extract_all_outer_columns condition)
				))
				(define mat_cols (reduce mat_cols_raw (lambda (acc mc)
					(if (reduce acc (lambda (found mc2) (or found (equal? (car mc2) (car mc)))) false)
						acc
						(merge acc (list mc)))) '()))
				(define mat_col_names (map mat_cols car))
				/* compute prejoin table name and alias */
				(define pjvar ".pj")
				/* canonical prejoin key: source tables only (no alias), for maximal reuse across equivalent queries */
				(define prejoin_alias_map (map tables (lambda (t)
					(match t '(tv tschema ttbl _ _)
						(list tv (concat tschema "." (if (string? ttbl) ttbl tv))))))
				)
				(define prejoin_col_names (map mat_cols (lambda (mc) (canonical_expr_name (cadr mc) '(list) '(list) prejoin_alias_map))))
				/* Use pre-expansion condition for stable prejoin naming.
				   condition_canonical_pre_expansion is (concat condition) from BEFORE
				   replace_inner_selects — it doesn't contain promise/scan code. */
				(define prejoin_condition_name (if (nil? condition_canonical_pre_expansion)
					(canonical_expr_name condition '(list) '(list) prejoin_alias_map)
					(fnv_hash condition_canonical_pre_expansion)))
				(define prejointbl_full (concat ".prejoin:"
					(map tables (lambda (t) (match t '(tv tschema ttbl _ _) (concat tschema "." (if (string? ttbl) ttbl tv))))
					) ":" prejoin_col_names "|" prejoin_condition_name))
				/* Use a short hash-based name to prevent explosively long keytable column names
				   in the recursive build_queryplan call (canon_alias_map embeds the prejoin name
				   into every keytable column). The full name is used for debugging only. */
				(define prejointbl (concat ".pj:" (fnv_hash prejointbl_full)))
				/* capture outer schema and table name for trigger code generation */
				(define pj_schema schema)
				(define pjtbl prejointbl)
				/* create prejoin table at build time (needed for recursive build_queryplan -> make_keytable) */
				(createtable schema prejointbl
					(map mat_col_names (lambda (col) '("column" col "any" '() '())))
					'("engine" "sloppy") true)
				/* build materialization scan: nested-loop join populating prejoin table */
				(define build_mat_scan (lambda (scan_tables scan_condition is_outermost)
					(match scan_tables
						(cons '(tblvar schema tbl isOuter joinexpr) rest) (begin
							/* columns needed from this table for materialization + condition */
							(set cols (merge_unique (list
								(extract_columns_for_tblvar tblvar scan_condition)
								(merge_unique (map mat_cols (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
								(extract_outer_columns_for_tblvar tblvar scan_condition)
								(merge_unique (map mat_cols (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
							)))
							(match (split_condition (coalesceNil scan_condition true) rest) '(now_condition later_condition) (begin
								(set filtercols (merge_unique (list
									(extract_columns_for_tblvar tblvar now_condition)
									(extract_outer_columns_for_tblvar tblvar now_condition))))
								(scan_wrapper 'scan schema tbl
									(cons list filtercols)
									'((quote lambda) (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
									(cons list cols)
									'((quote lambda) (map cols (lambda (col) (symbol (concat tblvar "." col)))) (build_mat_scan rest later_condition false))
									'('lambda '('acc 'sub) '('merge 'acc 'sub))
									'(list)
									(if is_outermost
										'('lambda '('acc 'shard_rows) '('insert pj_schema prejointbl (cons 'list mat_col_names) 'shard_rows '(list) '('lambda '() true) true))
										'('lambda '('acc 'shard_rows) '('merge 'acc 'shard_rows)))
									isOuter
								)
							))
						)
						'() /* base case: produce one row wrapped in a list */
						'('if (coalesceNil scan_condition true)
							(list (quote list) (cons (quote list) (map mat_cols (lambda (mc) (replace_columns_from_expr (cadr mc))))))
							'(list))
					)
				))
				(define materialize_plan (build_mat_scan tables condition true))
				/* rewrite all column references to point at prejoin table */
				(define pj_rewrite (lambda (expr) (rewrite_for_prejoin pjvar expr)))
				(define pj_fields (map_assoc resolved_fields (lambda (k v) (pj_rewrite v))))
				(define pj_group (map stage_group pj_rewrite))
				(define pj_having (pj_rewrite stage_having))
				(define pj_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (pj_rewrite col) dir))))))
				/* rebuild group stage for recursive call */
				(define pj_stage (make_group_stage pj_group pj_having pj_order stage_limit stage_offset))
				(define pj_all_groups (cons pj_stage rest_groups))
				/* recursive call with single prejoin table */
				(define grouped_result (build_queryplan schema '('(pjvar schema prejointbl false nil))
					pj_fields
					nil /* condition already applied during materialization */
					pj_all_groups
					schemas
					replace_find_column
					nil nil))
				/* build per-source-table incremental trigger functions entirely in Scheme,
				then register them via a thin Go wrapper */
				(define pj_trigger_registrations
					(map tables (lambda (trigger_tbl)
						(match trigger_tbl '(trigger_tv src_schema src_tbl _ _)
							(begin
								/* collect (pj_col base_col) pairs for this source table */
								(define ti_col_pairs
									(reduce mat_cols (lambda (acc mc)
										(match (cadr mc)
											'((symbol get_column) tv _ col _)
											(if (equal? tv trigger_tv)
												(merge acc (list (list (car mc) col)))
												acc)
											'((quote get_column) tv _ col _)
											(if (equal? tv trigger_tv)
												(merge acc (list (list (car mc) col)))
												acc)
											acc)) (list)))
								/* DELETE trigger: scan pjtbl, filter by OLD values for T_i cols, delete matching rows */
								(define delete_fn
									(eval (list 'lambda (list 'OLD 'NEW)
										(list 'scan pj_schema pjtbl
											(cons 'list (map ti_col_pairs car))
											(list 'lambda (map ti_col_pairs (lambda (p) (symbol (concat "_pj." (car p)))))
												(if (equal? 1 (count ti_col_pairs))
													(list 'equal? (symbol (concat "_pj." (car (car ti_col_pairs))))
														(list 'get_assoc 'OLD (cadr (car ti_col_pairs))))
													(cons 'and (map ti_col_pairs (lambda (p)
														(list 'equal? (symbol (concat "_pj." (car p)))
															(list 'get_assoc 'OLD (cadr p))))))))
											(list 'list "$update")
											(list 'lambda (list '$update) (list '$update))
											'+ 0 'nil 'false))))
								/* INSERT trigger: scan other tables with T_i cols fixed to NEW, insert rows */
								(define insert_fn
									(eval (list 'lambda (list 'OLD 'NEW)
										(build_pj_insert_scan tables condition trigger_tv true pj_schema pjtbl mat_cols mat_col_names))))
								/* UPDATE trigger: delete old prejoin rows + insert new for any row change.
								Code-generator pattern: embed delete_fn/insert_fn as proc literals in body
								so no closure capture — serializes cleanly for persistence. */
								(define update_fn (eval (list 'lambda (list 'OLD 'NEW) (list 'begin (list delete_fn 'OLD 'NEW) (list insert_fn 'OLD 'NEW)))))
								/* emit the register call as an S-expression to be executed at query time */
								(list 'register_prejoin_incremental src_schema src_tbl pj_schema pjtbl
									delete_fn insert_fn update_fn))))))
				/* assemble: create (if not exists) + materialize if empty + register triggers + grouped result */
				(cons 'begin (merge
					(list
						'('createtable pj_schema prejointbl
							(cons 'list (map mat_col_names (lambda (col) (list 'list "column" col "any" '(list) '(list)))))
							'(list "engine" "sloppy") true)
						(list 'if (list 'equal? 0 (list 'scan_estimate pj_schema prejointbl))
							'('time materialize_plan "materialize")))
					pj_trigger_registrations
					(list grouped_result)))
			)
		)
	) (optimize (begin
			/* grouping has been removed; now to the real data: */
			(if (and (not (nil? rest_groups)) (not (equal? rest_groups '()))) (error "non-group stage must be last"))
			(if has_window (begin
				/* ========= Window function scan path (LAG/LEAD) ========= */
				/* Case 8: different OVER clauses */
				(define first_over (nth (car window_funcs_all) 2))
				(if (not (reduce window_funcs_all (lambda (ok wf) (and ok (equal? (nth wf 2) first_over))) true))
					(error "multiple window functions with different OVER clauses not yet supported"))
				/* extract and resolve OVER info */
				(define over_partition (map (car first_over) replace_find_column))
				(define over_order (map (cadr first_over) (lambda (o) (match o '(col dir) (list (replace_find_column col) dir)))))
				(define effective_sort (merge (map over_partition (lambda (pe) (list pe <))) over_order))
				(define stage_order_resolved (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (list (replace_find_column col) dir)))))
				(define wf_resolved (map window_funcs_all (lambda (wf) (match wf '(fn args over)
					(list fn (map args replace_find_column) over)))))
				/* ========= ORC window function descriptors ========= */
				/* Build a mapfn that passes $set + N extra values through as (list $set composite).
				For 1 col: composite = scalar; for N>1: composite = (list v0 v1 ...). */
				(define build_key_mapfn (lambda (col_names) (begin
					(define key_params (map (produceN (count col_names) (lambda (i) i)) (lambda (i) (symbol (concat "__k" i)))))
					(define key_expr (if (equal? (count key_params) 1) (car key_params) (cons (quote list) key_params)))
					(define mapfn_params (cons (symbol "$set") key_params))
					(define mapfn_body (list (quote list) (symbol "$set") key_expr))
					(eval (list (quote lambda) mapfn_params mapfn_body)))))
				/* Build a mapfn for aggregate window functions: $set + sort_cols + agg_col */
				(define build_agg_mapfn (lambda (agg_col_name sort_col_names) (begin
					(define all_cols (merge sort_col_names (list agg_col_name)))
					(define params (map (produceN (count all_cols) (lambda (i) i)) (lambda (i) (symbol (concat "__v" i)))))
					(define mapfn_params (cons (symbol "$set") params))
					(define mapfn_body (cons (quote list) (cons (symbol "$set") params)))
					(eval (list (quote lambda) mapfn_params mapfn_body)))))
				/* Extract column name from a resolved expression */
				(define extract_col_name (lambda (expr) (match expr
					'((symbol get_column) _ _ c _) c
					'((quote get_column) _ _ c _) c
					_ nil)))
			/* orc_window_descriptor: fn × args × sort_col_names → (extra_mapcols mapfn reducefn reduceinit)
				Returns nil for non-ORC functions (LAG/LEAD stay on window_mut path). */
				(define orc_window_descriptor (lambda (fn wf_args sort_col_names)
					(match fn
						"ROW_NUMBER" (list '()
							(lambda ($set) (list $set))
							(lambda (acc mapped) (begin ((car mapped) (+ acc 1)) (+ acc 1)))
							0)
						"RANK" (list sort_col_names
							(build_key_mapfn sort_col_names)
							(lambda (acc mapped)
								(begin
									(define setter (car mapped))
									(define key (cadr mapped))
									(define prev_rank (nth acc 0))
									(define prev_rownum (nth acc 1))
									(define new_rownum (+ prev_rownum 1))
									(define new_rank (if (equal? key (nth acc 2)) prev_rank new_rownum))
									(setter new_rank)
									(list new_rank new_rownum key)))
							(list 0 0 nil))
						"DENSE_RANK" (list sort_col_names
							(build_key_mapfn sort_col_names)
							(lambda (acc mapped)
								(begin
									(define setter (car mapped))
									(define key (cadr mapped))
									(define prev_rank (car acc))
									(define new_rank (if (equal? key (cadr acc)) prev_rank (+ prev_rank 1)))
									(setter new_rank)
									(list new_rank key)))
							(list 0 nil))
						/* registry-based ordered aggregates as running ORC (only if ordered=true) */
						_ (begin
							(define agg_desc (sql_aggregates fn))
							(if (or (nil? agg_desc) (not (nth agg_desc 2))) nil
								(if (nil? wf_args) nil
									(begin
										(define agg_col (extract_col_name (car wf_args)))
										(if (nil? agg_col) nil
											(begin
												(define agg_reduce (car agg_desc))
												(define agg_neutral (cadr agg_desc))
												/* GROUP_CONCAT: build reducer with separator from args */
												(if (equal? fn "GROUP_CONCAT")
													(begin
														(define sep (if (> (count wf_args) 1) (cadr wf_args) ","))
														(list (list agg_col)
															(lambda ($set v) (list $set v))
															(lambda (acc mapped) (begin
																(define v (cadr mapped))
																(define new_acc (if (nil? acc) (concat v) (concat acc sep v)))
																((car mapped) new_acc)
																new_acc))
															nil))
													(list (list agg_col)
														(lambda ($set v) (list $set v))
														(lambda (acc mapped) (begin
															(define new_acc (agg_reduce acc (cadr mapped)))
															((car mapped) new_acc)
															new_acc))
														agg_neutral))))))))
				)))
				(define is_orc_window (lambda (wf) (match wf '(fn args _) (not (nil? (orc_window_descriptor fn args '()))))))
				/* aggregate window: look up fn in sql_aggregates registry → (reduce neutral ordered) */
				(define is_agg_window (lambda (wf) (match wf '(fn _ _) (not (nil? (sql_aggregates fn))))))
				/* is_ordered_agg: true if the aggregate is order-sensitive (e.g. GROUP_CONCAT) */
				(define is_ordered_agg (lambda (wf) (match wf '(fn _ _) (begin
					(define reg (sql_aggregates fn))
					(if (nil? reg) false (nth reg 2))))))
				/* classify: ORC (has ORDER BY + ORC-eligible or ordered aggregate),
				aggregate (no ORDER BY, or non-ordered aggregate ignoring ORDER BY),
				LAG/LEAD (everything else) */
				(define has_over_order (not (equal? over_order '())))
				(define all_orc_window (and has_over_order (reduce wf_resolved (lambda (acc wf) (and acc (or (is_orc_window wf) (is_ordered_agg wf)))) true)))
				/* agg window: non-ordered aggs always, OR ordered aggs WITHOUT ORDER BY (keytable, not ORC) */
				(define all_agg_window (and (not all_orc_window) (reduce wf_resolved (lambda (acc wf) (and acc (is_agg_window wf) (or (not (is_ordered_agg wf)) (not has_over_order)))) true)))
				(if all_orc_window
					(match tables
						/* ========= ORC materialization (ROW_NUMBER, RANK, DENSE_RANK, ...) ========= */
						'('(tblvar schema tbl isOuter _)) (begin
							/* extract ORC sort columns from OVER ORDER BY */
							(define orc_sort_col_names (map over_order (lambda (o) (match o '(col dir) (match col
								'((symbol get_column) _ _ c _) c
								'((quote get_column) _ _ c _) c
								_ (match (replace_find_column col)
									'((symbol get_column) _ _ c _) c
									'((quote get_column) _ _ c _) c
									_ (error (concat "unsupported ORC sort expression: " col))))))))
							(define orc_sort_dirs_vals (map over_order (lambda (o) (match o '(col dir)
								(if (equal? dir >) true false)))))
							/* get descriptor for the first window function (all share same OVER) */
							(define first_wf (car wf_resolved))
							(define wf_fn (car first_wf))
							(define wf_args (cadr first_wf))
							(define descriptor (orc_window_descriptor wf_fn wf_args orc_sort_col_names))
							(define inner_extra_mapcols (nth descriptor 0))
							(define inner_mapfn (nth descriptor 1))
							(define inner_reducefn (nth descriptor 2))
							(define inner_reduceinit (nth descriptor 3))
							/* partition wrapper: prepend partition cols, wrap reducer with boundary reset */
							(define has_partition (not (equal? over_partition '())))
							(define partition_col_names (if has_partition
								(map over_partition (lambda (pe) (match pe
									'((symbol get_column) _ _ c _) c
									'((quote get_column) _ _ c _) c
									_ (match (replace_find_column pe)
										'((symbol get_column) _ _ c _) c
										'((quote get_column) _ _ c _) c
										_ (error (concat "unsupported partition expression: " pe))))))
								'()))
							(define extra_mapcols (if has_partition (merge partition_col_names inner_extra_mapcols) inner_extra_mapcols))
							(define orc_mapfn (if has_partition (begin
							/* build mapfn: ($set part_cols... inner_cols...) → (cons partition_key inner_mapped)
								The inner reducer sees (cdr mapped); wrapper sees (car mapped) as partition key. */
								(define n_part (count partition_col_names))
								(define n_inner (count inner_extra_mapcols))
								(define all_params (cons (symbol "$set")
									(map (produceN (+ n_part n_inner) (lambda (i) i)) (lambda (i) (symbol (concat "__p" i))))))
								(define part_syms (slice all_params 1 (+ 1 n_part)))
								(define inner_syms (slice all_params (+ 1 n_part) (+ 1 n_part n_inner)))
								(define pk_expr (if (equal? n_part 1) (car part_syms) (cons (quote list) part_syms)))
								(define inner_call (cons inner_mapfn (cons (symbol "$set") inner_syms)))
								(eval (list (quote lambda) all_params (list (quote cons) pk_expr inner_call))))
								inner_mapfn))
							(define orc_reducefn (if has_partition (begin
								/* wrap: acc = (list inner_acc prev_pk); mapped = (cons pk inner_mapped) */
								(lambda (acc mapped)
									(begin
										(define pk (car mapped))
										(define inner_mapped (cdr mapped))
										(define prev_pk (cadr acc))
										(define inner_acc (car acc))
										(define eff_acc (if (or (nil? prev_pk) (equal? pk prev_pk)) inner_acc inner_reduceinit))
										(define new_inner (inner_reducefn eff_acc inner_mapped))
										(list new_inner pk))))
								inner_reducefn))
							(define orc_reduceinit (if has_partition (list inner_reduceinit nil) inner_reduceinit))
							/* unique temp column name */
							(define orc_col_name (concat ".orc_" wf_fn "_" tbl))
							/* compile time: add bare column so the scan plan can reference it */
							(createcolumn schema tbl orc_col_name "any" '() '("temp" true))
							/* replace window_func references with ORC column read */
							(define replace_wf (lambda (expr) (match expr
								(cons (symbol window_func) _) '((quote get_column) (eval tblvar) false (eval orc_col_name) false)
								(cons sym args_) (cons sym (map args_ replace_wf))
								expr)))
							(define new_fields (map_assoc fields (lambda (k v) (replace_wf v))))
							/* runtime plan: createcolumn with ORC params, then the actual scan */
							/* sortcols: partition cols (ASC) first, then ORDER BY cols */
							(define full_sort_cols (if has_partition (merge partition_col_names orc_sort_col_names) orc_sort_col_names))
							(define full_sort_dirs (if has_partition
								(merge (map partition_col_names (lambda (_) false)) orc_sort_dirs_vals)
								orc_sort_dirs_vals))
							/* partitioncount is auto-detected from reduceinit shape: (list init nil) → 1 partition key */
							(define orc_setup (lambda ()
								(createcolumn schema tbl orc_col_name "any" '()
									(list "sortcols" full_sort_cols "sortdirs" full_sort_dirs
										"mapcols" extra_mapcols
										"mapfn" orc_mapfn "reducefn" orc_reducefn
										"reduceinit" orc_reduceinit "temp" true))))
							(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil nil))
							(list (quote begin) (list orc_setup) scan_plan)
						)
						(error "window functions on joined tables not yet supported"))
					(if all_agg_window
						(match tables
							'('(tblvar schema tbl isOuter _))
							(build_agg_window_plan schema tbl tblvar tables over_partition wf_resolved condition groups schemas replace_find_column fields isOuter replace_columns_from_expr extract_columns_for_tblvar scan_wrapper)
							(error "window functions on joined tables not yet supported"))
						(begin
							/* ========= LAG/LEAD scan path (unchanged) ========= */
							/* Case 3: conflicting ORDER BY */
							(if (and (not (equal? stage_order_resolved '())) (not (equal? effective_sort stage_order_resolved)))
								(error "window ORDER BY with outer ORDER BY not yet supported"))
							(if (reduce wf_resolved (lambda (acc wf) (match wf '(fn _ _)
								(or acc (and (not (equal? fn "LAG")) (not (equal? fn "LEAD")))))) false)
								(error (concat "unsupported window function in LAG/LEAD context: " (car (car wf_resolved)))))
							/* single table only */
							(match tables
								'('(tblvar schema tbl isOuter _)) (begin
									(set condition (replace_find_column (coalesceNil condition true)))
									(define has_partition (not (equal? over_partition '())))
									/* compute stride_cols: all columns needed in output and window args */
									(define non_window_cols (merge_unique (extract_assoc fields (lambda (k v)
										(extract_columns_for_tblvar tblvar (replace_find_column v))))))
									(define wf_arg_cols (merge_unique (map wf_resolved (lambda (wf) (match wf '(fn args _)
										(merge_unique (map args (lambda (a) (extract_columns_for_tblvar tblvar a)))))))))
									(define partition_col_names (merge_unique (map over_partition (lambda (pe) (match pe
										'((symbol get_column) _ _ col _) '(col)
										'((quote get_column) _ _ col _) '(col)
										'())))))
									(define stride_cols (merge_unique (list non_window_cols wf_arg_cols partition_col_names)))
									(define stride (count stride_cols))
									/* window parameters */
									(define max_lag (reduce wf_resolved (lambda (acc wf) (match wf '(fn args _)
										(if (equal? fn "LAG") (max acc (if (> (count args) 1) (cadr args) 1)) acc))) 0))
									(define max_lead (reduce wf_resolved (lambda (acc wf) (match wf '(fn args _)
										(if (equal? fn "LEAD") (max acc (if (> (count args) 1) (cadr args) 1)) acc))) 0))
									(define window_size (+ max_lag 1 max_lead))
									(define skip max_lead)
									(define flush_count skip)
									(define current_row_pos (- window_size 1 skip))
									/* emit_fn parameter symbols */
									(define num_emit_params (* window_size stride))
									(define emit_params (map (produceN num_emit_params (lambda (i) i)) (lambda (i) (symbol (concat "__w" i)))))
									/* helper: find column index in stride_cols */
									(define col_index (lambda (col) (car (reduce stride_cols (lambda (acc c) (match acc '(idx found)
										(if found acc (if (equal?? c col) (list idx true) (list (+ idx 1) false))))) (list 0 false)))))
									/* rewrite field expression for emit_fn */
									(define rewrite_for_emit (lambda (expr row_pos) (match expr
										(cons (symbol window_func) wf_rest) (begin
											(define fn (car wf_rest))
											(define wf_args (cadr wf_rest))
											(define wf_offset (if (> (count wf_args) 1) (cadr wf_args) 1))
											(define wf_pos (if (equal? fn "LAG") (- current_row_pos wf_offset) (+ current_row_pos wf_offset)))
											(rewrite_for_emit (replace_find_column (car wf_args)) wf_pos))
										'((symbol get_column) (eval tblvar) _ col _) (nth emit_params (+ (* row_pos stride) (col_index col)))
										'((quote get_column) (eval tblvar) _ col _) (nth emit_params (+ (* row_pos stride) (col_index col)))
										'((symbol get_column) nil _ col _) (rewrite_for_emit (replace_find_column expr) row_pos)
										(cons sym args_) (cons sym (map args_ (lambda (a) (rewrite_for_emit a row_pos))))
										expr)))
									/* build emit_fn: (lambda (__w0 __w1 ...) (resultrow (list field_rewrites...))) */
									(define emit_body '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (rewrite_for_emit v current_row_pos))))))
									(define emit_fn_ast (list (quote lambda) emit_params emit_body))
									/* build neutral */
									(define neutral_list (merge (list skip 0 stride) (produceN (* window_size stride) (lambda (_) nil))))
									(define neutral_ast (cons (quote list) neutral_list))
									/* sort cols/dirs from effective_sort */
									(define ordercols (merge (map effective_sort (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										_ '()))))))
									(define sort_dirs (merge (map effective_sort (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										_ '()))))))
									/* filter setup */
									(define filtercols (extract_columns_for_tblvar tblvar condition))
									/* symbols for emit_fn and fresh neutral */
									(define efn_sym (symbol "__emit_fn"))
									(define nfn_sym (symbol "__fresh_neutral"))
									(if has_partition (begin
										/* === Case 4: PARTITION BY + ORDER BY === */
										(define window_end (+ 3 (* window_size stride)))
										/* partition key expression in mapfn */
										(define partition_col_syms (map over_partition (lambda (pe) (match pe
											'((symbol get_column) _ _ col _) (symbol (concat tblvar "." col))
											'((quote get_column) _ _ col _) (symbol (concat tblvar "." col))))))
										(define pk_value_expr (if (equal? (count partition_col_syms) 1)
											(car partition_col_syms)
											(cons (quote list) partition_col_syms)))
										/* mapfn: returns (list stride_vals... partition_key) */
										(define mapfn_ast (list (quote lambda)
											(map stride_cols (lambda (col) (symbol (concat tblvar "." col))))
											(list (quote append)
												(cons (quote list) (map stride_cols (lambda (col) (symbol (concat tblvar "." col)))))
												pk_value_expr)))
										/* neutral with nil partition key */
										(define neutral_partition_ast (cons (quote list) (merge neutral_list (list nil))))
										/* partition-aware reducer */
										(define reducer_ast (list (quote lambda) '('acc 'mapped) (list (quote begin)
											'('define 'pk '('nth 'mapped stride))
											'('define 'vs '('slice 'mapped 0 stride))
											'('define 'prev_pk '('nth 'acc window_end))
											'('define 'win '('slice 'acc 0 window_end))
											(list (quote if) '('or '('nil? 'prev_pk) '('equal? 'pk 'prev_pk))
												'('append '('window_mut 'win efn_sym 'vs) 'pk)
												(list (quote begin)
													(if (> flush_count 0) '('window_flush 'win efn_sym flush_count) true)
													'('append '('window_mut nfn_sym efn_sym 'vs) 'pk))))))
										/* build scan with post-flush */
										(define scan_plan (list (quote begin)
											(list (quote define) efn_sym emit_fn_ast)
											(list (quote define) nfn_sym neutral_ast)
											(if (> flush_count 0) (begin
												(list (quote begin)
													(list (quote define) (symbol "__scan_result")
														(scan_wrapper 'scan_order schema tbl
															(cons list filtercols)
															'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
															(cons list ordercols)
															(cons list sort_dirs)
															0 0 -1
															(cons list stride_cols)
															mapfn_ast
															reducer_ast
															neutral_partition_ast
															isOuter))
													(list (quote window_flush) (list (quote slice) (symbol "__scan_result") 0 window_end) efn_sym flush_count)))
												(scan_wrapper 'scan_order schema tbl
													(cons list filtercols)
													'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
													(cons list ordercols)
													(cons list sort_dirs)
													0 0 -1
													(cons list stride_cols)
													mapfn_ast
													reducer_ast
													neutral_partition_ast
													isOuter))))
										scan_plan
									) (begin
											/* === Case 1: ORDER BY only, no partition === */
											/* mapfn: returns (list stride_vals...) */
											(define mapfn_ast '((quote lambda)
												(map stride_cols (lambda (col) (symbol (concat tblvar "." col))))
												(cons (quote list) (map stride_cols (lambda (col) (symbol (concat tblvar "." col)))))))
											/* simple reducer */
											(define reducer_ast '((quote lambda) '('acc 'mapped) '('window_mut 'acc efn_sym 'mapped)))
											/* build scan with post-flush */
											(define scan_plan (list (quote begin)
												(list (quote define) efn_sym emit_fn_ast)
												(if (> flush_count 0) (begin
													(list (quote begin)
														(list (quote define) (symbol "__scan_result")
															(scan_wrapper 'scan_order schema tbl
																(cons list filtercols)
																'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
																(cons list ordercols)
																(cons list sort_dirs)
																0 0 -1
																(cons list stride_cols)
																mapfn_ast
																reducer_ast
																neutral_ast
																isOuter))
														(list (quote window_flush) (symbol "__scan_result") efn_sym flush_count)))
													(scan_wrapper 'scan_order schema tbl
														(cons list filtercols)
														'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
														(cons list ordercols)
														(cons list sort_dirs)
														0 0 -1
														(cons list stride_cols)
														mapfn_ast
														reducer_ast
														neutral_ast
														isOuter))))
											scan_plan
									))
								)
								(error "window functions on joined tables not yet supported")
				))))
			) (if (coalesce stage_order stage_limit stage_offset) (begin
					/* ordered or limited scan */
					/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
					(set stage_order (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (list (replace_find_column col) dir)))))
					/* build_scan now takes is_first parameter to apply offset/limit only to outermost scan */
					(define build_scan (lambda (tables condition is_first)
						(match tables
							(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
								/* For LEFT JOIN: merge joinexpr into condition for extraction */
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

									/* check if this table is the DML target (ordered path) */
									(define is_update_target_ord (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
									(define ord_scan_mapcols (if is_update_target_ord (cons list (cons "$update" cols)) (cons list cols)))
									(define ord_scan_mapfn_params (if is_update_target_ord
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))

									(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
									(scan_wrapper 'scan_order schema tbl
										(cons list filtercols)
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
										(cons list ordercols)
										(cons list dirs)
										0
										scan_offset
										scan_limit
										ord_scan_mapcols
										(list (symbol "lambda") ord_scan_mapfn_params (build_scan tables later_condition false))
										(if is_update_target_ord (symbol "+") nil)
										(if is_update_target_ord 0 nil)
										isOuter
									)
								))
							)
							'() /* final inner */ (if (nil? update_target)
								'((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v)))))
								/* DML mode: emit $update call */
								(begin (define _ut_cols (nth update_target 1))
									(if (equal? _ut_cols '())
										'('if true '('$update) 0)
										'('if true '('$update (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))) 0))))
						)
					))
					(build_scan tables (replace_find_column condition) true)
				) (begin
						/* unordered unlimited scan */

						/* TODO: sort tables according to join plan */
						/* TODO: match tbl to inner query vs string */
						(define build_scan (lambda (tables condition)
							(match tables
								(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
									/* check if this table is the UPDATE target */
									(define is_update_target (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
									/* also extract cols needed for SET expressions in update_target */
									(define ut_extra_cols (if is_update_target
										(merge_unique (extract_assoc (nth update_target 1) (lambda (k v) (extract_columns_for_tblvar tblvar v))))
										'()))
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
											ut_extra_cols
										)
									))
									/* For UPDATE target: prepend $update to mapcols */
									(define scan_mapcols (if is_update_target (cons list (cons "$update" cols)) (cons list cols)))
									(define scan_mapfn_params (if is_update_target
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									/* split condition in those ANDs that still contain get_column from tables and those evaluatable now */
									(match (split_condition (coalesceNil condition true) tables) '(now_condition later_condition) (begin
										(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
										(scan_wrapper 'scan schema tbl
											/* condition */
											(cons list filtercols)
											'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
											/* extract columns and store them into variables */
											scan_mapcols
											(list (symbol "lambda") scan_mapfn_params (build_scan tables later_condition))
											(if is_update_target (symbol "+") nil)
											(if is_update_target 0 nil)
											nil
											isOuter
										)
									))
								)
								'() /* final inner (=scalar) */ (if (nil? update_target)
									'('if (coalesceNil condition true) '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v))))))
									/* DML mode */
									(begin (define _ut_cols (nth update_target 1))
										(if (equal? _ut_cols '())
											/* DELETE */
											'('if (coalesceNil condition true) '('$update) 0)
											/* UPDATE */
											'('if (coalesceNil condition true) '('$update (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))) 0))))
							)
						))
						(build_scan tables (replace_find_column condition))
			)))
	)))
)))
