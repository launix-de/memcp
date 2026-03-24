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

/* split_condition: selection pushdown for nested-loop join planning.
Splits an AND-condition into (now, later): predicates evaluatable with currently
bound tables vs predicates that must wait for inner tables to be scanned.
Enables index-based filtering in scan/scan_order by pushing predicates down. */
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

/* has_only_tblvar_refs: returns true if expr contains get_column refs and ALL of them
reference only the given tblvar. Returns false if any get_column references another alias,
or if expr has no get_column refs at all (pure literal → not a tblvar-only condition). */
(define has_only_tblvar_refs (lambda (expr tblvar) (match expr
	'((symbol get_column) alias_ _ _ _) (equal? alias_ tblvar)
	'((quote get_column) alias_ _ _ _) (equal? alias_ tblvar)
	(cons sym args) (reduce args (lambda (acc arg) (begin
		(define child (has_only_tblvar_refs arg tblvar))
		(if (nil? acc) child (if (nil? child) acc (and acc child))))) nil)
	nil /* literal: no refs → nil (unknown) */
)))

/* extract_pure_tblvar_conditions: from an AND expression, extract parts that
reference ONLY tblvar columns (no outer refs). Returns the AND of those parts, or true. */
(define extract_pure_tblvar_conditions (lambda (expr tblvar) (match expr
	(cons (symbol and) parts) (reduce parts (lambda (acc part)
		(if (equal? (has_only_tblvar_refs part tblvar) true)
			(if (equal? acc true) part (list (quote and) acc part))
			acc)) true)
	_ (if (equal? (has_only_tblvar_refs expr tblvar) true) expr true)
)))

/* extract_non_pure_tblvar_conditions: from an AND expression, extract parts that
reference OTHER tables too (not only tblvar). Complement of extract_pure_tblvar_conditions. */
(define extract_non_pure_tblvar_conditions (lambda (expr tblvar) (match expr
	(cons (symbol and) parts) (reduce parts (lambda (acc part)
		(if (not (equal? (has_only_tblvar_refs part tblvar) true))
			(if (equal? acc true) part (list (quote and) acc part))
			acc)) true)
	_ (if (not (equal? (has_only_tblvar_refs expr tblvar) true)) expr true)
)))

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

/* symbols that canonicalize_columns must NOT recurse into — they have their own scope */
(define _is_opaque_scope_sym (lambda (sym) (match sym
	/* inner_select markers are NOT opaque — they are logical markers that
	must be transparent for outer-ref detection during Neumann decorrelation.
	Only physical runtime code (scan, !begin, etc.) is opaque. */
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
	(define spa (stage_partition_aliases stage))
	(if (stage_is_dedup stage)
		(make_dedup_stage (map sg canon) spa)
		(if (and (not (nil? spa)) (or (nil? sg) (equal? sg '())))
			/* partition stage (aliases but no group): preserve partition-aliases and limit-partition-cols */
			(make_partition_stage spa
				(map so (lambda (o) (match o '(c d) (list (canon c) d))))
				(coalesceNil (stage_limit_partition_cols stage) 0) sl soff (stage_init_code stage))
			/* group stage (possibly scoped with aliases) */
			(make_group_stage
				(map sg canon)
				(canon sh)
				(map so (lambda (o) (match o '(c d) (list (canon c) d))))
				sl soff spa (stage_init_code stage))))
)))

(import "sql-metadata.scm")

/* group stage constructors and accessors - shared between untangle_query and build_queryplan
All stages have partition-aliases (scope): nil = global (all tables), list = scoped to those tables.
All stages have init: nil = no init code, or code to run before the scan. */
(define make_group_stage (lambda (group having order limit offset aliases init)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) having)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) 0)
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
		(list (quote partition-aliases) aliases)
		(list (quote init) init)
	)
))
(define make_partition_stage (lambda (aliases order partition_cols limit offset init)
	(list
		(cons (quote group-cols) '())
		(list (quote having) nil)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) partition_cols)
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
		(list (quote partition-aliases) aliases)
		(list (quote init) init)
	)
))
(define make_dedup_stage (lambda (group aliases)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) nil)
		(list (quote order) '())
		(list (quote limit-partition-cols) 0)
		(list (quote limit) nil)
		(list (quote offset) nil)
		(list (quote dedup) true)
		(list (quote partition-aliases) aliases)
		(list (quote init) nil)
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
(define stage_limit_partition_cols (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote limit-partition-cols) rest) (if (nil? rest) 0 (car rest))
		_ nil
	) acc)
) nil)))
(define stage_partition_aliases (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote partition-aliases) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_init_code (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote init) rest) (if (nil? rest) nil (car rest))
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

/* make_keytable_schema: compute keytable name and schema without creating the table.
Used by untangle to predict the keytable name for HAVING subselect decorrelation.
Returns (keytable_name key_col_names schema_def) where schema_def is a list of
column descriptors suitable for the schemas assoc in untangle_query.
Does NOT handle FK→PK reuse (returns nil for that case — caller must check). */
(define make_keytable_schema (lambda (schema tbl keys tblvar) (begin
	(define alias_map (list (list tblvar (concat schema "." tbl))))
	(define key_names (map keys (lambda (k) (canonical_expr_name k '(list) '(list) alias_map))))
	(define keytable_name (concat "." tbl ":" key_names))
	(define schema_def (map key_names (lambda (colname) (list "Field" colname "Type" "any"))))
	(list keytable_name key_names schema_def)
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
		(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil))
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
		'((symbol get_column) (eval src_tblvar) ti col ci) '('get_column grouptbl ti (colname '('get_column src_tblvar ti col ci)) ci)
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
=== untangle_query: Neumann query decorrelation ===

Implements the algebraic unnesting transformation from Neumann/Kemper (BTW 2015)
and the holistic top-down extension (Neumann BTW 2025). Transforms a parsed SQL
query with arbitrarily nested correlated subqueries into a flat relational IR:

INPUT:  parsed query (schema tables fields condition group having order limit offset)
OUTPUT: (schema tables fields condition groups schemas replace_find_column)

The output is a single flat table list where every correlated subquery has been
replaced by a LEFT JOIN table entry. Dependencies between nesting levels are
expressed as join conditions; aggregation boundaries are expressed as group-stages
with partition-aliases (scoping). There is no nested runtime code in the output.

Key transformations:
- Derived tables (FROM subqueries): flattened into parent table list with column renaming
- Scalar subselects: decorrelated via unnest_subselect into LEFT JOIN + partition-stage (Path B)
or LEFT JOIN + scoped GROUP-stage (Path A for aggregates)
- IN/EXISTS/NOT IN/NOT EXISTS: rewritten to COUNT(*) aggregates, then decorrelated via Path A
- Domain column extension: Neumann Γ_{A∪D;f} — outer correlation columns added to GROUP BY
- Condition merging: WHERE and JOIN ON conditions unified into a single condition list
- Unused LEFT JOIN pruning: tables not referenced in output are eliminated

Does NOT: choose join order (join_reorder), create keytables (build_queryplan),
or generate runtime scan code (build_queryplan).
*/
(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas_param) (begin
	(set rename_prefix (coalesce rename_prefix ""))
	(define outer_schemas_chain (coalesceNil outer_schemas_param '()))
	(define sq_cache (newsession))

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

	/* unnest_subselect: core Neumann decorrelation for a single subquery.
	Transforms a correlated scalar subquery into a LEFT JOIN table entry,
	eliminating the dependent join. Returns (substitution tables) or nil.

	Three paths based on subquery shape:
	Path A (aggregate): adds domain columns to GROUP BY (Neumann Γ_{A∪D}),
	flattens inner tables with scoped GROUP-stage. Handles COUNT/SUM/AVG/etc.
	Path B (non-agg + LIMIT): creates partition-stage for LIMIT per outer row,
	direct LEFT JOIN table entry. Handles ORDER BY + LIMIT 1 pattern.
	Path C (non-agg, no LIMIT): returns nil (fallback to inline evaluation).

	Recursive nesting: inner subqueries are decorrelated first by untangle_query.
	Their tables become "inner-scoped" (identified via partition-aliases) and are
	passed through to the outer level with joinexpr rewriting. Dependencies on
	tables outside the current scope stay as bare get_column references. */
	(define unnest_subselect (lambda (subquery outer_schemas) (begin
		(define union_parts_us (query_union_all_parts subquery))
		(if (not (nil? union_parts_us))
			nil /* UNION ALL not handled yet */
			(begin
				(define raw_vals_us (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)))
				(define raw_group_us (nth raw_vals_us 0))
				(define raw_having_us (nth raw_vals_us 1))
				(define raw_order_us (nth raw_vals_us 2))
				(define raw_limit_us (nth raw_vals_us 3))
				(define raw_offset_us (nth raw_vals_us 4))
				(match (apply untangle_query subquery)
					'(schema2_us tables2_us fields2_us condition2_us groups2_us schemas2_us rfcol2_us) (begin
						(define groups2_us (coalesceNil groups2_us '()))
						(define groups2_us (if (or (nil? groups2_us) (equal? groups2_us '()))
							(if (or raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us)
								(list (make_group_stage raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us nil nil))
								groups2_us)
							groups2_us))
						/* resolve columns against inner and outer schemas */
						(define rfcs_us (make_replace_find_column_subselect schemas2_us outer_schemas))
						(set fields2_us (map_assoc fields2_us (lambda (k v) (rfcs_us v))))
						(set condition2_us (rfcs_us (coalesceNil condition2_us true)))
						/* wrap remaining unresolved qualified refs as (outer tbl.col) */
						(define _us_wrap (lambda (e) (match e
							'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci))
								(list (quote outer) (symbol (concat alias_ "." col)))
								e)
							(cons sym args) (cons (_us_wrap sym) (map args _us_wrap))
							e)))
						(set fields2_us (map_assoc fields2_us (lambda (k v) (_us_wrap v))))
						(set condition2_us (_us_wrap condition2_us))
						/* extract all outer references from fields and condition.
						Detects both explicit (outer tbl.col) AND bare (get_column tbl false col false)
						where tbl is NOT in the inner table set (from nested unnesting).
						Skip opaque scopes (!begin, scan, etc.). */
						(define us_inner_aliases (map tables2_us (lambda (td) (match td '(a _ _ _ _) a ""))))
						(define _us_eor (lambda (expr) (match expr
							(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
								(match args (cons sym_arg '()) (list (string sym_arg)) '())
								(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
									(match args '(alias_ _ col _) (if (and (not (nil? alias_)) (not (reduce us_inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
										(list (concat alias_ "." col)) '())
										'())
									(if (_is_opaque_scope_sym sym) '()
										(merge_unique (map args _us_eor)))))
							'())))
						(define us_outer_refs (merge_unique
							(merge (extract_assoc fields2_us (lambda (k v) (_us_eor v))))
							(_us_eor condition2_us)))
						/* feasibility checks */
						(define us_has_outer (not (equal? us_outer_refs '())))
						/* separate own stages from inner scoped stages (from nested decorrelation) —
						must be defined BEFORE _us_inner_aliases which depends on _us_inner_stages */
						(define _us_own_stages (filter (coalesceNil groups2_us '()) (lambda (s) (nil? (stage_partition_aliases s)))))
						(define _us_inner_stages (filter (coalesceNil groups2_us '()) (lambda (s) (not (nil? (stage_partition_aliases s))))))
						/* count only OWN tables (not inner scoped ones from nested decorrelation) */
						(define _us_inner_aliases (merge (map _us_inner_stages (lambda (s) (coalesceNil (stage_partition_aliases s) '())))))
						(define _us_own_tables (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (not (has? _us_inner_aliases a)) true))))
						(define us_single_tbl (and (list? _us_own_tables) (equal? (count _us_own_tables) 1)))
						/* check for aggregates in fields */
						(define _us_agg (lambda (expr) (match expr
							'((symbol aggregate) _ _ _) true
							(cons sym args) (reduce args (lambda (a b) (or a (_us_agg b))) false)
							false)))
						(define us_has_agg (reduce_assoc fields2_us (lambda (a k v) (or a (_us_agg v))) false))
						/* check for GROUP/HAVING in OWN stages only */
						(define us_has_stages (not (equal? _us_own_stages '())))
						(define us_has_grp (if us_has_stages
							(reduce _us_own_stages (lambda (acc stage) (or acc
								(begin
									(define g (stage_group_cols stage))
									(or (and (not (nil? g)) (not (equal? g '())) (not (equal? g '(1))))
										(not (nil? (stage_having_expr stage))))))) false)
							false))
						/* check for LIMIT/ORDER/OFFSET stages — deferred until 1-row constraint handling */
						(define us_has_limit (if us_has_stages
							(reduce _us_own_stages (lambda (acc stage) (or acc
								(not (nil? (stage_limit_val stage)))
								(not (nil? (stage_offset_val stage)))
								(begin
									(define o (coalesceNil (stage_order_list stage) '()))
									(and (not (nil? o)) (not (equal? o '())))))) false)
							false))
						/* check for outer refs in fields (not just condition) — these need
						more complex handling, fall back for now */
						(define us_outer_in_fields (not (equal?
							(merge (extract_assoc fields2_us (lambda (k v) (_us_eor v)))) '())))
						(if us_outer_in_fields nil /* outer refs in fields: not handled yet */
							(begin
								/* === Neumann unnesting: nD domain, single or multi-table === */
								/* generate unique alias using fnv_hash to avoid collisions across nesting levels */
								(define us_sq_idx (coalesceNil (sq_cache "idx") 0))
								(sq_cache "idx" (+ us_sq_idx 1))
								(define _us_own_tblname (match (car _us_own_tables) '(_ _ t _ _) (if (string? t) t "x") "x"))
								(define us_sq_prefix (concat "_unn_" _us_own_tblname "_" us_sq_idx))
								/* build alias rename map: only OWN tables get prefixed.
								Inner-scoped tables (from nested decorrelation) keep their alias. */
								(define us_alias_map (map _us_own_tables (lambda (td) (match td
									'(alias _ _ _ _) (list alias (if us_single_tbl us_sq_prefix (concat us_sq_prefix "\0" alias)))
									(list "" "")))))
								(define _us_lookup (lambda (a) (reduce us_alias_map (lambda (acc p) (if (nil? acc) (if (equal?? a (nth p 0)) (nth p 1) nil) acc)) nil)))
								/* value column and its source alias */
								(define us_value_key (car (extract_assoc fields2_us (lambda (k v) k))))
								(define us_value_expr (car (extract_assoc fields2_us (lambda (k v) v))))
								(define us_value_src (match us_value_expr '((symbol get_column) a _ _ _) a '((quote get_column) a _ _ _) a nil))
								(define us_value_new (if (nil? us_value_src) us_sq_prefix (coalesceNil (_us_lookup us_value_src) us_sq_prefix)))
								/* helper: does expr contain outer refs? Detects both (outer ...) and
								bare get_column refs to non-inner tables. Skip opaque scopes. */
								(define _us_hor (lambda (expr) (match expr
									(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
										true
										(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
											(match args '(alias_ _ _ _) (and (not (nil? alias_)) (not (reduce us_inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false))) false)
											(if (_is_opaque_scope_sym sym) false
												(reduce args (lambda (a b) (or a (_us_hor b))) false))))
									false)))
								/* split condition into AND-parts */
								(define _us_fap (lambda (expr) (match expr
									(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
										(merge (map parts _us_fap))
										(list expr))
									(list expr))))
								(define us_cond_parts (_us_fap condition2_us))
								(define us_inner_parts (filter us_cond_parts (lambda (p) (not (_us_hor p)))))
								(define us_outer_parts (filter us_cond_parts (lambda (p) (_us_hor p))))
								/* resolve (outer tbl.col) -> (get_column tbl false col false).
								Runtime resolves bare symbols via scope chain (multi-level lookup). */
								(define _us_ror (lambda (expr) (match expr
									(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
										(match args
											(cons sym_arg '()) (begin
												(define ps (split (string sym_arg) "."))
												(match ps
													(list tbl col) (list (quote get_column) tbl false col false)
													_ expr))
											_ expr)
										(cons (_us_ror sym) (map args _us_ror)))
									expr)))
								/* replace ANY inner alias with its renamed version */
								(define _us_ria (lambda (expr) (match expr
									'((symbol get_column) alias_ ti col ci) (begin
										(define na (_us_lookup alias_))
										(if (nil? na) expr (list (quote get_column) na false col false)))
									'((quote get_column) alias_ ti col ci) (begin
										(define na (_us_lookup alias_))
										(if (nil? na) expr (list (quote get_column) na false col false)))
									(cons sym args) (cons (_us_ria sym) (map args _us_ria))
									expr)))
								/* inner condition (non-correlated) - kept with original aliases for
								aggregate path; renamed for non-aggregate path */
								(define us_inner_cond_raw (if (equal? (count us_inner_parts) 0) nil
									(if (equal? (count us_inner_parts) 1) (car us_inner_parts)
										(cons (quote and) us_inner_parts))))
								/* extract domain columns from correlated equalities:
								(equal?? inner_expr outer_expr) → (inner_expr resolved_outer_expr) */
								(define us_domain_cols (filter (map us_outer_parts (lambda (part) (match part
									'((symbol equal??) a b) (if (_us_hor a) (if (not (_us_hor b)) (list b (_us_ror a)) nil) (if (_us_hor b) (list a (_us_ror b)) nil))
									'((quote equal??) a b) (if (_us_hor a) (if (not (_us_hor b)) (list b (_us_ror a)) nil) (if (_us_hor b) (list a (_us_ror b)) nil))
									nil))) (lambda (x) (not (nil? x)))))
								/* === Three-way branch: aggregate / non-agg+LIMIT / non-agg-no-LIMIT === */
								(if (or us_has_agg us_has_grp)
									/* === A: Aggregate → flatten inner tables + scoped GROUP stage ===
									Neumann Γ_{A∪D;f}: add domain cols to GROUP BY, flatten inner tables
									with prefix into outer table list. No materialization. */
									(begin
										/* rename inner aliases: alias → _sq0\0alias (recursive in all exprs) */
										(define _us_prefix_ria (lambda (expr) (match expr
											'((symbol get_column) alias_ ti col ci) (begin
												(define na (_us_lookup alias_))
												(if (nil? na) expr (list (quote get_column) na false col false)))
											'((quote get_column) alias_ ti col ci) (begin
												(define na (_us_lookup alias_))
												(if (nil? na) expr (list (quote get_column) na false col false)))
											(cons sym args) (cons (_us_prefix_ria sym) (map args _us_prefix_ria))
											expr)))
										/* prefix inner tables: alias → _sq0\0alias, tbl stays string */
										(define us_prefixed_tables (map tables2_us (lambda (td) (match td
											'(a s t io je) (list (coalesceNil (_us_lookup a) a) s t io
												(if (nil? je) nil (_us_prefix_ria je)))
											td))))
										/* inner condition (non-correlated), prefixed */
										(define us_inner_cond_prefixed (if (nil? us_inner_cond_raw) nil (_us_prefix_ria us_inner_cond_raw)))
										/* domain columns + original GROUP BY → scoped GROUP stage */
										(define us_orig_group (if us_has_stages (coalesceNil (stage_group_cols (car _us_own_stages)) '()) '()))
										(define us_orig_having (if us_has_stages (stage_having_expr (car _us_own_stages)) nil))
										(define _us_dom_group_cols (map us_domain_cols (lambda (dc) (_us_prefix_ria (nth dc 0)))))
										(define us_new_group (merge _us_dom_group_cols
											(if (or (equal? us_orig_group '()) (equal? us_orig_group '(1)))
												/* keep (1) for static aggregation if no domain cols */
												(if (equal? _us_dom_group_cols '()) us_orig_group '())
												(map us_orig_group _us_prefix_ria))))
										(define us_new_having (if (nil? us_orig_having) nil (_us_prefix_ria us_orig_having)))
										/* scoped GROUP stage: partition-aliases = prefixed inner table aliases */
										(define us_inner_aliases (map us_prefixed_tables (lambda (td) (match td '(a _ _ _ _) a ""))))
										(define us_group_stage (make_group_stage us_new_group us_new_having '() nil nil us_inner_aliases nil))
										/* propagate inner scoped stages with prefix */
										(define _us_prefixed_inner_stages (map _us_inner_stages (lambda (s) (begin
											(define _psg (map (coalesceNil (stage_group_cols s) '()) _us_prefix_ria))
											(define _psh (if (nil? (stage_having_expr s)) nil (_us_prefix_ria (stage_having_expr s))))
											(define _pso (map (coalesceNil (stage_order_list s) '()) (lambda (o) (match o '(c d) (list (_us_prefix_ria c) d) o))))
											(define _psa (map (coalesceNil (stage_partition_aliases s) '()) (lambda (a) (coalesceNil (_us_lookup a) a))))
											(make_group_stage _psg _psh _pso (stage_limit_val s) (stage_offset_val s) _psa (stage_init_code s))))))
										/* register prefixed tables */
										(sq_cache "tables" (merge us_prefixed_tables (coalesceNil (sq_cache "tables") '())))
										/* register scoped GROUP stage + propagated inner stages */
										(sq_cache "groups" (merge (list us_group_stage) _us_prefixed_inner_stages (coalesceNil (sq_cache "groups") '())))
										/* register schemas for prefixed aliases */
										(define us_prefixed_schemas (merge (map us_prefixed_tables (lambda (td) (match td
											'(a _ _ _ _) (begin
												(define _orig_a (reduce us_alias_map (lambda (acc p) (if (nil? acc) (if (equal? (nth p 1) a) (nth p 0) nil) acc)) nil))
												(define _s_cols (if (nil? _orig_a) nil (schemas2_us _orig_a)))
												(if (nil? _s_cols) '() (list a _s_cols)))
											'())))))
										(sq_cache "schemas" (merge us_prefixed_schemas (coalesceNil (sq_cache "schemas") '())))
										/* join condition: domain equalities (outer_expr = prefixed_inner_expr) */
										(define us_dom_je_parts (map us_domain_cols (lambda (dc)
											(list (quote equal??) (_us_prefix_ria (nth dc 0)) (nth dc 1)))))
										(define us_dom_je (if (equal? (count us_dom_je_parts) 0) true
											(if (equal? (count us_dom_je_parts) 1) (car us_dom_je_parts)
												(cons (quote and) us_dom_je_parts))))
										/* set first inner table to LEFT JOIN with domain+inner condition as joinexpr */
										(define _us_full_je (if (nil? us_inner_cond_prefixed) us_dom_je
											(if (equal? us_dom_je true) us_inner_cond_prefixed
												(list (quote and) us_dom_je us_inner_cond_prefixed))))
										(if (not (nil? us_prefixed_tables))
											(sq_cache "tables" (begin
												(define _all_tbls (sq_cache "tables"))
												(define _first_inner (car us_prefixed_tables))
												(define _first_alias (match _first_inner '(a _ _ _ _) a ""))
												(map _all_tbls (lambda (td) (match td
													'(a s t io je) (if (equal? a _first_alias)
														(list a s t true _us_full_je)
														td)
													td))))))
										/* substitution: reference the prefixed value column */
										(define us_subst_raw (_us_prefix_ria us_value_expr))
										(define us_is_count (match us_value_expr
											'((symbol aggregate) _ (symbol +) 0) true
											'((quote aggregate) _ (symbol +) 0) true
											'((quote aggregate) _ '(symbol +) 0) true
											false))
										(define us_subst (if us_is_count (list (quote coalesceNil) us_subst_raw 0) us_subst_raw))
										/* return substitution + empty table entries (tables already in sq_cache) */
										(list us_subst '()))
									/* === B/C: Non-aggregate === */
									(begin
										/* value must be a simple column (not computed expression) for direct table entry */
										(define _us_val_is_col (match us_value_expr
											'((symbol get_column) _ _ _ _) true '((quote get_column) _ _ _ _) true false))
										(if (and us_single_tbl _us_val_is_col)
											/* === B/C: Non-agg → direct LEFT JOIN table entry ===
											Path B (has LIMIT): adds partition-stage for ORDER BY + LIMIT per outer row.
											Path C (no LIMIT): plain LEFT JOIN, no partition-stage. */
											(begin
												(define us_tdesc (car tables2_us))
												(define us_tblvar (nth us_tdesc 0))
												(define us_tbl_schema (nth us_tdesc 1))
												(define us_tbl_name (nth us_tdesc 2))
												(define us_orig_order (if us_has_stages (coalesceNil (stage_order_list (car _us_own_stages)) '()) '()))
												(define us_orig_limit (if us_has_stages (stage_limit_val (car _us_own_stages)) nil))
												(define us_orig_offset (if us_has_stages (stage_offset_val (car _us_own_stages)) nil))
												/* pass through inner-scoped tables (from nested decorrelation) with joinexpr rewriting */
												(define _us_inner_tbls (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (has? _us_inner_aliases a) false))))
												(define _us_inner_tbls_rewritten (map _us_inner_tbls (lambda (td) (match td
													'(a s t io je) (list a s t io (if (nil? je) nil (_us_ria je)))
													td))))
												(if (not (equal? _us_inner_tbls_rewritten '()))
													(sq_cache "tables" (merge _us_inner_tbls_rewritten (coalesceNil (sq_cache "tables") '()))))
												/* Always register partition stage: Path B uses explicit LIMIT,
												Path C uses implicit LIMIT 1 (scalar subselect = at most one row) */
												(begin
													(define us_dom_order (map us_domain_cols (lambda (dc) (list (_us_ria (nth dc 0)) '<))))
													(define us_renamed_order (map (coalesceNil us_orig_order '()) (lambda (oi) (match oi '(col dir) (list (_us_ria col) dir) oi))))
													(define us_part_order (merge us_dom_order us_renamed_order))
													(define us_dom_count (count us_domain_cols))
													(define us_part_stage (make_partition_stage (list us_sq_prefix) us_part_order us_dom_count (coalesceNil us_orig_limit 1) (coalesceNil us_orig_offset 0) nil))
													(sq_cache "partition_stages" (cons us_part_stage (coalesceNil (sq_cache "partition_stages") '()))))
												/* propagate inner scoped stages with renaming */
												(if (not (equal? _us_inner_stages '()))
													(sq_cache "groups" (merge
														(map _us_inner_stages (lambda (s) (begin
															(define _psg (map (coalesceNil (stage_group_cols s) '()) _us_ria))
															(define _psh (if (nil? (stage_having_expr s)) nil (_us_ria (stage_having_expr s))))
															(define _pso (map (coalesceNil (stage_order_list s) '()) (lambda (o) (match o '(c d) (list (_us_ria c) d) o))))
															(define _psa (map (coalesceNil (stage_partition_aliases s) '()) (lambda (a) (coalesceNil (_us_lookup a) a))))
															(make_group_stage _psg _psh _pso (stage_limit_val s) (stage_offset_val s) _psa (stage_init_code s)))))
														(coalesceNil (sq_cache "groups") '()))))
												/* direct table entry with join condition (like non-agg non-LIMIT path) */
												(define us_join_lim (map us_outer_parts (lambda (p) (_us_ria (_us_ror p)))))
												(define us_inner_lim (_us_ria us_inner_cond_raw))
												(define us_full_lim (if (nil? us_inner_lim)
													(if (equal? (count us_join_lim) 0) true (if (equal? (count us_join_lim) 1) (car us_join_lim) (cons (quote and) us_join_lim)))
													(cons (quote and) (merge us_join_lim (list us_inner_lim)))))
												(define us_tbl_entries (list (list us_sq_prefix us_tbl_schema us_tbl_name true us_full_lim)))
												/* register schema for own table + pass through inner-scoped schemas */
												(define _us_inner_schema (schemas2_us us_tblvar))
												(define _us_passthrough_schemas (merge
													(if (not (nil? _us_inner_schema)) (list us_sq_prefix _us_inner_schema) '())
													(merge (map _us_inner_tbls (lambda (td) (match td
														'(a _ _ _ _) (begin
															(define _isch (schemas2_us a))
															(if (nil? _isch) '() (list a _isch)))
														'()))))))
												(if (not (equal? _us_passthrough_schemas '()))
													(sq_cache "schemas" (merge _us_passthrough_schemas (coalesceNil (sq_cache "schemas") '()))))
												/* substitution: apply _us_ria to the value expression.
												If value comes from own table, _us_ria renames it to us_sq_prefix.
												If value comes from inner-scoped table, it stays unchanged. */
												(define us_subst (_us_ria us_value_expr))
												(list us_subst us_tbl_entries))
											nil /* multi-table or computed value: not yet handled */
									))
								)
							)
						)
					)
					nil /* untangle failed */
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
	/* _unnest_count_subselect: shared helper for IN/EXISTS/NOT IN/NOT EXISTS rewrite.
	Builds a COUNT(*) subquery from the original, optionally adding an equality condition
	(for IN/NOT IN: first_field = target_expr). Returns (substitution tables) or nil.
	comparison: (quote >) for positive match, (quote equal?) for negated match */
	(define _unnest_count_subselect (lambda (subquery outer_schemas target_expr comparison) (begin
		(define _has_tables (match subquery '(_ t _ _ _ _ _ _ _) (and (not (nil? t)) (not (equal? t '()))) false))
		(define _first_field (if (nil? target_expr) nil
			(match subquery '(_ _ flds _ _ _ _ _ _) (match flds (cons k (cons v _)) v nil) nil)))
		(if (and (not _has_tables) (nil? target_expr)) nil
			(if (and (not (nil? target_expr)) (nil? _first_field)) nil
				(begin
					(define _count_sq (match subquery
						'(s t f c g h o l off) (list s t
							(list "__cnt" (list (quote aggregate) 1 (symbol "+") 0))
							(if (nil? target_expr) c
								(if (or (nil? c) (equal? c true))
									(list (quote equal??) _first_field target_expr)
									(list (quote and) c (list (quote equal??) _first_field target_expr))))
							(list 1) nil nil nil nil)
						nil))
					(if (nil? _count_sq) nil
						(begin
							(define _result (unnest_subselect _count_sq outer_schemas))
							(if (nil? _result) nil
								(match _result '(_subst _tbls) (begin
									(sq_cache "tables" (merge _tbls (coalesceNil (sq_cache "tables") '())))
									(list comparison (list (quote coalesceNil) _subst 0) 0)))))))))
	)))
	/* replace_inner_selects: walks an expression tree and replaces inner_select markers
	with their Neumann-decorrelated equivalents. Scalar subselects go through
	unnest_subselect directly; IN/EXISTS/NOT IN/NOT EXISTS are first rewritten to
	COUNT(*) aggregates via _unnest_count_subselect, then decorrelated via Path A.
	Returns the rewritten expression with subselects replaced by get_column refs
	or comparison expressions on the unnested aggregate columns. */
	(define replace_inner_selects (lambda (expr outer_schemas) (match expr
		(cons sym args) (begin
			(define kind (inner_select_kind sym))
			/* handle NOT IN / NOT EXISTS */
			(define not_expr (if (not_symbol sym)
				(match args
					(cons inner_expr '()) (match inner_expr
						(cons inner_sym inner_args) (begin
							(define inner_kind (inner_select_kind inner_sym))
							(if (equal?? inner_kind (quote inner_select_in))
								(match inner_args
									(cons target_expr (cons subquery '()))
									(coalesce (_unnest_count_subselect subquery outer_schemas target_expr (quote equal?)) expr)
									_ nil)
								(if (equal?? inner_kind (quote inner_select_exists))
									(match inner_args
										(cons subquery '())
										(coalesce (_unnest_count_subselect subquery outer_schemas nil (quote equal?)) expr)
										_ nil)
									nil)))
						_ nil)
					_ nil)
				nil))
			(if (nil? not_expr)
				(match kind
					(quote inner_select) (match args
						(cons subquery '()) (begin
							(define _us_r (unnest_subselect subquery outer_schemas))
							(if (nil? _us_r)
								expr
								(match _us_r '(_us_subst _us_tbls) (begin
									(sq_cache "tables" (merge _us_tbls (coalesceNil (sq_cache "tables") '())))
									_us_subst))))
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					(quote inner_select_in) (match args
						(cons target_expr (cons subquery '()))
						(coalesce (_unnest_count_subselect subquery outer_schemas target_expr (quote >)) expr)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					(quote inner_select_exists) (match args
						(cons subquery '())
						(coalesce (_unnest_count_subselect subquery outer_schemas nil (quote >)) expr)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
				not_expr))
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
			(set groups (if (or group having order limit offset) (list (make_group_stage group having order limit offset nil nil)) nil))
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
											(list (quote cons) (symbol "item") (list rows_sym "rows"))))
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
									(define materialized_rows (list (quote begin)
										(list (quote set) rows_sym (list (quote newsession)))
										(list rows_sym "rows" '())
										(list (quote set) resultrow_sym (symbol "resultrow"))
										(define cnt_sym (symbol (concat "__from_subquery_cnt:" id)))
										(if (nil? mat_limit)
											/* no limit */
											(list (quote set) (symbol "resultrow")
												(list (quote lambda) (list (symbol "item"))
													(list rows_sym "rows"
														(list (quote cons) (symbol "item") (list rows_sym "rows"))))
											)
											/* with limit: stop collecting after mat_limit rows */
											(list (quote begin)
												(list (quote set) cnt_sym 0)
												(list (quote set) (symbol "resultrow")
													(list (quote lambda) (list (symbol "item"))
														(list (quote if) (list (quote <) cnt_sym mat_limit)
															(list (quote begin)
																(list (quote set) cnt_sym (list (quote +) cnt_sym 1))
																(list rows_sym "rows"
																	(list (quote cons) (symbol "item") (list rows_sym "rows"))))
															nil))))
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

			/* extract additional join exprs into condition list */
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
				/* Unqualified column: prefer main tables over unnested/subquery tables.
				Main tables have no ':' prefix and no '_unn_' prefix in their alias. */
				'((symbol get_column) nil _ col ci) (begin
					/* First try main tables (aliases without ':' or '_unn_' prefix) */
					(define _is_main_alias (lambda (alias) (begin
						(define s (string alias))
						(and (not (strlike s "%:%"))
							(not (and (>= (strlen s) 5) (equal? (substr s 0 5) "_unn_")))))))
					(define main_match (reduce_assoc schemas (lambda (a alias cols)
						(if (and (_is_main_alias alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
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
			(set condition (replace_inner_selects condition schemas))
			(set group (map group (lambda (g) (replace_inner_selects g schemas))))
			(set having (begin
				(define _hv_resolved (replace_inner_selects having schemas))
				/* check if any inner_select nodes remain — HAVING with subqueries
				requires post-group processing which is not yet implemented */
				(define _hv_check (lambda (expr) (match expr
					(cons sym args) (if (not (nil? (inner_select_kind sym))) true
						(reduce args (lambda (a b) (or a (_hv_check b))) false))
					false)))
				(if (and (not (nil? _hv_resolved)) (_hv_check _hv_resolved))
					(error "HAVING with subqueries not yet supported")
					_hv_resolved)))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col schemas) dir)))))
			/* integrate unnested scalar subselects from Neumann unnesting.
			Tables from non-aggregate path (direct LEFT JOIN) do NOT need schema updates.
			Tables from aggregate path (materialized derived) DO need schemas for build_queryplan. */
			(define _sq_tbls (coalesceNil (sq_cache "tables") '()))
			(set tables (merge tables _sq_tbls))
			(define _sq_schs (coalesceNil (sq_cache "schemas") '()))
			(if (not (equal? _sq_schs '())) (set schemas (merge schemas _sq_schs)))
			(define _sq_jes (filter (map _sq_tbls (lambda (t) (match t '(_ _ _ _ je) je nil))) (lambda (x) (not (nil? x)))))
			(set condition (if (equal? _sq_jes '()) condition (cons (quote and) (cons condition _sq_jes))))
			/* integrate partition stages from non-aggregate LIMIT unnesting */
			(define _sq_pstages (coalesceNil (sq_cache "partition_stages") '()))
			(define _sq_prop_groups (coalesceNil (sq_cache "groups") '()))
			(set groups (if (equal? _sq_pstages '()) groups (merge _sq_pstages (coalesceNil groups '()))))
			(set groups (if (equal? _sq_prop_groups '()) groups (merge _sq_prop_groups (coalesceNil groups '()))))
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
			(set conditionAll (cons 'and (filter (cons (replace_rename (canonicalize_for_rename condition)) conditionList) (lambda (x) (not (nil? x))))))
			(set group (map group (lambda (g) (replace_rename (canonicalize_for_rename g)))))
			(set order (map order (lambda (o) (match o '(col dir) (list (replace_rename (canonicalize_for_rename col)) dir)))))

			(set having (replace_rename (canonicalize_for_rename having)))

			(define groups (merge
				(coalesceNil _sq_pstages '())
				(coalesceNil _sq_prop_groups '())
				(if (coalesce _cd_distinct_exprs false)
					/* COUNT(DISTINCT): two group stages - first dedup, then aggregate */
					(list
						(make_dedup_stage
							(merge (map (coalesce _cd_user_group '()) replace_rename) (map _cd_distinct_exprs (lambda (e) (replace_find_column (replace_rename e))))) nil)
						(make_group_stage
							(if (nil? _cd_user_group) '(1) (map _cd_user_group (lambda (e) (replace_find_column (replace_rename e)))))
							(_cd_replace (replace_rename _cd_having))
							(map (coalesce _cd_order '()) (lambda (o) (match o '(col dir) (list (_cd_replace (replace_rename col)) dir))))
							_cd_limit _cd_offset nil nil))
					/* normal: single group stage */
					(if (or group having order limit offset) (list (make_group_stage group having order limit offset nil nil)) '()))))
			/* canonicalize all get_column markers: resolve ti/ci flags to canonical casing.
			After this, all get_column nodes have false false — no case ambiguity remains. */
			(define _canon (lambda (expr) (canonicalize_columns expr schemas)))
			(define _canon_fields (map_assoc fields (lambda (k v) (_canon (replace_rename v)))))
			(define _canon_condition (_canon conditionAll))
			(define _canon_groups (map (coalesceNil groups '()) (lambda (stage) (canonicalize_stage stage schemas))))
			/* eliminate unused LEFT JOINs: if no column of a LEFT JOIN table
			is referenced in fields, condition, or group stages, the join
			cannot affect row count and can be safely removed */
			/* eliminate unused LEFT JOINs: a LEFT JOIN is unused when none of its
			columns appear in fields or group stages. The merged condition includes
			joinexprs which reference the JOIN itself — those must not prevent
			elimination. We only check fields + groups (the query's output). */
			(define _used_tvs (merge_unique
				(merge (extract_assoc _canon_fields (lambda (k v) (extract_tblvars v))))
				(merge (map _canon_groups (lambda (stage)
					(merge_unique
						(merge (map (coalesceNil (stage_group_cols stage) '()) extract_tblvars))
						(extract_tblvars (coalesceNil (stage_having_expr stage) true))
						(merge (map (coalesceNil (stage_order_list stage) '()) (lambda (o) (match o '(col dir) (extract_tblvars col) (extract_tblvars o)))))
						/* scoped stages: their partition-aliases are "used" tables */
						(coalesceNil (stage_partition_aliases stage) '())))))
				/* include condition-referenced tables: unnested subqueries may create
				cross-table dependencies (e.g., d.did = _sq0.did) that must prevent pruning */
				(extract_tblvars _canon_condition)))
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
			(list schema _pruned_tables _canon_fields _canon_condition _canon_groups schemas replace_find_column)
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
			(apply build_queryplan (merge (apply join_reorder (apply untangle_query (merge query (list nil)))) (list nil)))
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
	(define pipeline_result (apply join_reorder (apply untangle_query (merge synthetic_select (list nil)))))
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
	))
	(apply build_queryplan final_pipeline)
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
/*
=== build_queryplan: physical plan generation ===

Translates the flat relational IR from untangle_query into executable scan code.
Consumes the table list, conditions, and group-stages and produces nested scan/scan_order
calls, keytable materialization (GROUP BY), and prejoin materialization (multi-table GROUP).

Processing order (recursive — each stage peels off one layer):
1. Group-stages with partition-aliases (scoped): separate into keytable fill + post-group scan
- Single-table group: make_keytable + collect keys + createcolumn per aggregate
- Multi-table group: prejoin materialization + keytable on the prejoin
- Aggregates are discovered in fields, order, having, AND condition (Neumann EXISTS/IN rewrite)
2. Partition-stages (LIMIT per partition): scan_order with partition columns
3. ORDER BY / LIMIT / OFFSET: scan_order on the remaining tables
4. Unordered scan: nested-loop scan over remaining tables

Key helpers:
- make_keytable: creates sloppy temp table for group keys + computed aggregate columns
- split_condition: selection pushdown — splits AND-parts by which tables they reference
- replace_columns_from_expr: rewrites get_column markers to runtime variable references
- scan_wrapper: generates scan/scan_order calls with filter/map/reduce structure
*/
/* update_target: nil for SELECT, or (tblalias (col1 expr1 col2 expr2 ...)) for multi-table UPDATE.
When set, the scan on tblalias includes $update in mapcols and the mapfn applies the SET expressions. */
(define build_queryplan (lambda (schema tables fields condition groups schemas replace_find_column update_target) (begin
	/*(print "build queryplan " '(schema tables fields condition groups schemas))*/

	/* TODO: order tables: outer joins behind */
	(set groups (coalesceNil groups '()))
	/* separate partition stages (have partition-aliases) from regular stages */
	/* separate partition stages (have aliases but NO group-cols) from regular/scoped group stages */
	(define partition_stages (filter groups (lambda (s) (begin
		(define _spa (stage_partition_aliases s))
		(define _sg (stage_group_cols s))
		(and (not (nil? _spa)) (or (nil? _sg) (equal? _sg '())))))))
	(set groups (filter groups (lambda (s) (begin
		(define _spa (stage_partition_aliases s))
		(define _sg (stage_group_cols s))
		(or (nil? _spa) (and (not (nil? _sg)) (not (equal? _sg '()))))))))
	(define groups_present (and (not (nil? groups)) (not (equal? groups '()))))
	(define stage (if groups_present (car groups) nil))
	(define rest_groups (if groups_present (cdr groups) nil))
	(set rest_groups (coalesceNil rest_groups '()))
	(define stage_group (if stage (stage_group_cols stage) nil))
	(define stage_having (if stage (stage_having_expr stage) nil))
	(define stage_order (if stage (stage_order_list stage) nil))
	(define stage_partcols (if stage (coalesceNil (stage_limit_partition_cols stage) 0) 0))
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
		(define ags (if is_dedup ags (merge_unique ags (extract_aggregates (coalesce condition true))))) /* aggregates in condition (from Neumann EXISTS/IN rewrite) */

		/* TODO: replace (get_column nil ti col ci) in group, having and order with (coalesce (fields col) '('get_column nil false col false)) */

		/* determine which tables the GROUP BY applies to:
		- if stage has partition-aliases (scoped): only those tables
		- otherwise (global): all tables except partition-staged ones */
		(define _grp_ps_aliases (merge (map partition_stages (lambda (s) (coalesceNil (stage_partition_aliases s) '())))))
		(define _stage_scope (stage_partition_aliases stage))
		(define _grp_tables (if (not (nil? _stage_scope))
			/* scoped GROUP: only the tables listed in the stage's aliases */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (has? _stage_scope tv) false)))
			/* global GROUP: all tables except partition-staged */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (not (has? _grp_ps_aliases tv)) true)))))
		(define _grp_ps_tables (filter tables (lambda (t) (match t '(tv _ _ _ _)
			(and (not (has? (coalesceNil _stage_scope '()) tv))
				(or (has? _grp_ps_aliases tv) (not (nil? _stage_scope))))
			false))))
		(match _grp_tables
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
				/* 2-phase condition split:
				Phase 1: separate aggregate-containing AND-parts from non-aggregate parts.
				Aggregates cannot be evaluated as row filters — they need the keytable.
				Phase 2 (after keytable creation): replace aggregates with get_column refs,
				then split by table references for pushdown. */
				(define _has_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (_has_agg_expr b))) false)
					false)))
				(define _flatten_and_parts (lambda (expr) (match expr
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
						(merge (map parts _flatten_and_parts))
						(list expr))
					(list expr))))
				(define _cond_parts (_flatten_and_parts condition))
				(define _cond_agg_parts (filter _cond_parts _has_agg_expr))
				(define _cond_non_agg (filter _cond_parts (lambda (p) (not (_has_agg_expr p)))))
				/* non-aggregate condition = keytable scan filter */
				(set condition (if (equal? 0 (count _cond_non_agg)) true
					(if (equal? 1 (count _cond_non_agg)) (car _cond_non_agg)
						(cons (quote and) _cond_non_agg))))
				/* split non-aggregate condition: parts referencing partition-staged tables go to grouped_plan */
				(define _grp_cond_split (split_condition condition _grp_ps_tables))
				(define _grp_ps_condition (match _grp_cond_split '(_ later) later))
				(set condition (match _grp_cond_split '(now _) now))
				(set filtercols (extract_columns_for_tblvar tblvar condition))

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
							(stage_offset_val s)
							(stage_partition_aliases s)
							(stage_init_code s))
					)))
					(define grouped_plan (build_queryplan schema '('(grouptbl schema grouptbl false nil))
						(map_assoc fields (lambda (k v) (_dedup_resolve v)))
						nil /* condition already applied in collect */
						transformed_rest_groups
						schemas
						replace_find_column
						nil))
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
							(if (coalesce grouped_order stage_limit stage_offset) (list (make_group_stage nil nil grouped_order stage_limit stage_offset nil nil)) '())
							rest_groups
						))
						/* FK reuse: extract child FK column name */
						(define fk_child_col (if is_fk_reuse
							(match (car stage_group) '('get_column _ false scol false) scol)
							nil))
						/* COUNT column replaces the old exists column: always add (1 + 0) to ags
						so we have a row count per group. HAVING filters on COUNT > 0 to exclude
						empty groups after DELETE. This also enables incremental DELETE maintenance.
						Scoped GROUPs (from Neumann unnesting) are temporary — no stale groups,
						and NOT EXISTS needs COUNT=0 groups to survive. */
						(define needs_count (and (not (equal? stage_group '(1))) (nil? _stage_scope)))
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

						/* Phase 2: replace aggregates in the separated agg-condition parts,
						then combine everything: HAVING + replaced agg-parts + ps-table conditions */
						(define _replaced_agg_parts (map _cond_agg_parts replace_group_key_or_fetch))
						(define _gp_parts (filter (merge
							(if (or (nil? effective_having) (equal? effective_having true)) '() (list effective_having))
							_replaced_agg_parts
							(if (equal? _grp_ps_condition true) '() (list (replace_group_key_or_fetch _grp_ps_condition))))
							(lambda (x) (and (not (nil? x)) (not (equal? x true))))))
						(define _gp_condition (if (equal? 0 (count _gp_parts)) nil
							(if (equal? 1 (count _gp_parts)) (car _gp_parts)
								(cons (quote and) _gp_parts))))
						/* drop partition-stages covered by this scoped GROUP: the keytable
						guarantees 1 row per group key, making the partition LIMIT redundant */
						(define _remaining_pstages (filter partition_stages (lambda (ps)
							(not (reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
								(or acc (has? (coalesceNil _stage_scope '()) a))) false)))))
						/* scoped GROUPs: outer tables come FIRST, keytable is LEFT JOINed
						AFTER them. This ensures outer rows without keytable matches still
						appear (with NULL aggregates → coalesceNil → 0).
						Essential for NOT EXISTS / NOT IN semantics. */
						(define _kt_is_outer (and (not (nil? _stage_scope)) (not (equal? stage_group '(1)))))
						(define _kt_je (if _kt_is_outer
							/* build join condition: keytable group-key columns = outer domain expressions */
							(begin
								(define _kt_je_parts (map stage_group (lambda (g) (list (quote equal??) (replace_group_key_or_fetch g) g))))
								(if (equal? 1 (count _kt_je_parts)) (car _kt_je_parts)
									(if (> (count _kt_je_parts) 1) (cons (quote and) _kt_je_parts) true)))
							nil))
						(define grouped_plan (build_queryplan schema
							(if _kt_is_outer
								(merge _grp_ps_tables (list (list grouptbl schema grouptbl true _kt_je)))
								(merge (list (list grouptbl schema grouptbl false nil)) _grp_ps_tables))
							(map_assoc fields (lambda (k v) (replace_group_key_or_fetch v)))
							_gp_condition
							(merge next_groups _remaining_pstages)
							schemas
							replace_find_column
							nil))

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
				/* exclude partition-staged tables from prejoin (same as single-table path) */
				(define _pj_tables (filter tables (lambda (t) (match t '(tv _ _ _ _) (not (has? _grp_ps_aliases tv)) true))))
				(define _pj_ps_tables (filter tables (lambda (t) (match t '(tv _ _ _ _) (has? _grp_ps_aliases tv) false))))
				/* resolve condition and fields */
				(set condition (replace_find_column (coalesceNil condition true)))
				/* split condition: partition-staged table refs go to grouped_plan */
				(define _pj_cond_split (split_condition condition _pj_ps_tables))
				(define _pj_ps_condition (match _pj_cond_split '(_ later) later))
				(set condition (match _pj_cond_split '(now _) now))
				(define resolved_fields (map_assoc fields (lambda (k v) (replace_find_column v))))
				/* extract all get_column refs from group, fields, having, order */
				(define mat_cols_raw (merge
					(merge (map stage_group extract_all_get_columns))
					(merge (extract_assoc resolved_fields (lambda (k v) (extract_all_get_columns v))))
					(if (nil? stage_having) '() (extract_all_get_columns stage_having))
					(merge (map (coalesce stage_order '()) (lambda (o) (match o '(col dir) (extract_all_get_columns col)))))
				))
				/* filter out columns from partition-staged tables (they're not part of the prejoin) */
				(define _pj_tblvar_list (map _pj_tables (lambda (t) (match t '(tv _ _ _ _) tv ""))))
				(define mat_cols_raw (filter mat_cols_raw (lambda (mc) (match mc
					'(name '((symbol get_column) alias_ _ _ _)) (has? _pj_tblvar_list alias_)
					'(name '((quote get_column) alias_ _ _ _)) (has? _pj_tblvar_list alias_)
					true))))
				(define mat_cols (reduce mat_cols_raw (lambda (acc mc)
					(if (reduce acc (lambda (found mc2) (or found (equal? (car mc2) (car mc)))) false)
						acc
						(merge acc (list mc)))) '()))
				(define mat_col_names (map mat_cols car))
				/* compute prejoin table name and alias */
				(define pjvar ".pj")
				/* canonical prejoin key: source tables only (no alias), for maximal reuse across equivalent queries */
				(define prejoin_alias_map (map _pj_tables (lambda (t)
					(match t '(tv tschema ttbl _ _)
						(list tv (concat tschema "." ttbl)))))
				)
				(define prejoin_col_names (map mat_cols (lambda (mc) (canonical_expr_name (cadr mc) '(list) '(list) prejoin_alias_map))))
				(define prejoin_condition_name (canonical_expr_name condition '(list) '(list) prejoin_alias_map))
				(define prejointbl (concat ".prejoin:"
					(map _pj_tables (lambda (t) (match t '(_ tschema ttbl _ _) (concat tschema "." ttbl)))
					) ":" prejoin_col_names "|" prejoin_condition_name))
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
						(cons '(tblvar schema tbl isOuter _) rest) (begin
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
									/* reduce: merge sub-results */
									'('lambda '('acc 'sub) '('merge 'acc 'sub))
									'(list)
									/* reduce2: outermost inserts into prejoin, inner levels merge */
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
				(define pj_stage (make_group_stage pj_group pj_having pj_order stage_limit stage_offset nil nil))
				(define pj_all_groups (cons pj_stage rest_groups))
				/* recursive call with single prejoin table */
				/* combine partition-staged table conditions with prejoin output */
				(define _pj_gp_condition (if (equal? _pj_ps_condition true) nil
					(pj_rewrite _pj_ps_condition)))
				(define grouped_result (build_queryplan schema
					(merge (list (list pjvar schema prejointbl false nil)) _pj_ps_tables)
					pj_fields
					_pj_gp_condition
					(merge pj_all_groups partition_stages)
					schemas
					replace_find_column
					nil))
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
							(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil))
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
									/* LEFT JOIN filter split (ordered path): move pure-tblvar conditions to post-filter */
									(define effective_later (if isOuter (begin
										(define _pure_cond (extract_pure_tblvar_conditions now_condition tblvar))
										(if (equal? _pure_cond true) later_condition
											(if (equal? later_condition true) _pure_cond
												(list (quote and) later_condition _pure_cond))))
										later_condition))
									(define now_condition (if isOuter
										(extract_non_pure_tblvar_conditions now_condition tblvar)
										now_condition))
									(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
									/* check partition_stages for this table (non-first tables may have per-table partition limits) */
									(define _ps_ord (if is_first nil
										(reduce partition_stages (lambda (a s) (if (nil? a) (if (has? (coalesceNil (stage_partition_aliases s) '()) tblvar) s nil) a)) nil)))
									/* use partition stage's order if available, otherwise use stage_order */
									(define _eff_order (if (nil? _ps_ord) stage_order (coalesceNil (stage_order_list _ps_ord) '())))
									/* extract order cols for this tblvar */
									(set ordercols (merge (map _eff_order (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										_ '()
									))))))
									(set dirs (merge (map _eff_order (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										_ '()
									))))))

									/* offset/limit: for partition-staged tables use their limits, else only outermost */
									(define scan_offset (if (not (nil? _ps_ord)) (coalesceNil (stage_offset_val _ps_ord) 0)
										(if is_first stage_offset 0)))
									(define scan_limit (if (not (nil? _ps_ord)) (coalesceNil (stage_limit_val _ps_ord) -1)
										(if is_first (coalesceNil stage_limit -1) -1)))
									(define scan_partcols (if (not (nil? _ps_ord)) (coalesceNil (stage_limit_partition_cols _ps_ord) 0)
										(if is_first stage_partcols 0)))

									/* check if this table is the DML target (ordered path) */
									(define is_update_target_ord (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
									(define ord_scan_mapcols (if is_update_target_ord (cons list (cons "$update" cols)) (cons list cols)))
									(define ord_scan_mapfn_params (if is_update_target_ord
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									/* emit init code from partition stage if present */
									(define _ps_init (if (nil? _ps_ord) nil (stage_init_code _ps_ord)))
									(define _ord_scan (scan_wrapper 'scan_order schema tbl
										/* condition */
										(cons list filtercols)
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
										/* sortcols, sortdirs */
										(cons list ordercols)
										(cons list dirs)
										scan_partcols
										scan_offset
										scan_limit
										/* extract columns and store them into variables */
										ord_scan_mapcols
										(list (symbol "lambda") ord_scan_mapfn_params (build_scan tables effective_later false))
										/* reduce+neutral for DML */
										(if is_update_target_ord (symbol "+") nil)
										(if is_update_target_ord 0 nil)
										isOuter
									))
									(if (nil? _ps_init) _ord_scan (list (quote begin) _ps_init _ord_scan))
								))
							)
							'() /* final inner */ (if (nil? update_target)
								'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v))))))
								/* DML mode: emit $update call */
								(begin (define _ut_cols (nth update_target 1))
									(if (equal? _ut_cols '())
										'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '('$update) 0)
										'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '('$update (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))) 0))))
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
										/* LEFT JOIN filter split: for isOuter tables, conditions that reference ONLY
										this table's columns (no outer refs) must be MOVED to the post-filter.
										The scan filter keeps only join-correlation conditions (with outer refs).
										This ensures hadValue correctly reflects join partner existence. */
										(define effective_later (if isOuter (begin
											(define _pure_cond (extract_pure_tblvar_conditions now_condition tblvar))
											(if (equal? _pure_cond true) later_condition
												(if (equal? later_condition true) _pure_cond
													(list (quote and) later_condition _pure_cond))))
											later_condition))
										(define now_condition (if isOuter
											(extract_non_pure_tblvar_conditions now_condition tblvar)
											now_condition))
										(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
										/* check partition_stages: does this table have a per-table partition limit? */
										(define _ps (reduce partition_stages (lambda (a s) (if (nil? a) (if (has? (coalesceNil (stage_partition_aliases s) '()) tblvar) s nil) a)) nil))
										(if (not (nil? _ps))
											/* === partition-limited scan_order === */
											(begin
												(define _ps_order (coalesceNil (stage_order_list _ps) '()))
												(define _ps_partcols (coalesceNil (stage_limit_partition_cols _ps) 0))
												(define _ps_limit (coalesceNil (stage_limit_val _ps) -1))
												(define _ps_offset (coalesceNil (stage_offset_val _ps) 0))
												(define _ps_ordercols (merge (map _ps_order (lambda (oi) (match oi '(col dir) (match col
													'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
													'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
													_ '()))))))
												(define _ps_dirs (merge (map _ps_order (lambda (oi) (match oi '(col dir) (match col
													'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
													'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
													_ '()))))))
												/* emit init code from partition stage if present */
												(define _ps_init2 (stage_init_code _ps))
												(define _ps_scan (scan_wrapper 'scan_order schema tbl
													(cons list (merge_unique filtercols cols))
													'((quote lambda) (map (merge_unique filtercols cols) (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
													(cons list _ps_ordercols)
													(cons list _ps_dirs)
													_ps_partcols _ps_offset _ps_limit
													scan_mapcols
													(list (symbol "lambda") scan_mapfn_params (build_scan tables effective_later))
													nil nil isOuter))
												(if (nil? _ps_init2) _ps_scan (list (quote begin) _ps_init2 _ps_scan)))
											/* === regular scan === */
											(scan_wrapper 'scan schema tbl
												(cons list filtercols)
												'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
												scan_mapcols
												(list (symbol "lambda") scan_mapfn_params (build_scan tables effective_later))
												(if is_update_target (symbol "+") nil)
												(if is_update_target 0 nil)
												nil
												isOuter
										))
									))
								)
								'() /* final inner (=scalar) */ (if (nil? update_target)
									'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (replace_columns_from_expr v))))))
									/* DML mode */
									(begin (define _ut_cols (nth update_target 1))
										(if (equal? _ut_cols '())
											/* DELETE */
											'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '('$update) 0)
											/* UPDATE */
											'('if (optimize (replace_columns_from_expr (coalesceNil condition true))) '('$update (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))) 0))))
							)
						))
						(build_scan tables (replace_find_column condition))
			)))
	)))
)))
