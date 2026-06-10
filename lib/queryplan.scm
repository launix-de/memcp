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
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW 'session) (list 'droptable pj_schema pj_table true))))
	(createtrigger (table src_schema src_table) (concat prefix "after_insert")     "after_insert"     "" drop_body false)
	(createtrigger (table src_schema src_table) (concat prefix "after_update")     "after_update"     "" drop_body false)
	(createtrigger (table src_schema src_table) (concat prefix "after_delete")     "after_delete"     "" drop_body false)
	(createtrigger (table src_schema src_table) (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger (table src_schema src_table) (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* Registers incremental maintenance triggers on src_table to keep pj_table in sync.
delete_fn/insert_fn/update_fn are code-generator-produced lambda expressions (no closures).
Lifecycle triggers use code-generator pattern for the drop body as well.
update_fn embeds delete_fn/insert_fn as proc literals in its body (no closure capture). */
(define register_prejoin_incremental (lambda (src_schema src_table pj_schema pj_table delete_fn insert_fn update_fn) (begin
	(define prefix (concat ".pj_incr:" pj_table "|" src_table "|"))
	(createtrigger (table src_schema src_table) (concat prefix "after_delete") "after_delete" "" delete_fn false)
	(createtrigger (table src_schema src_table) (concat prefix "after_insert") "after_insert" "" insert_fn false)
	(createtrigger (table src_schema src_table) (concat prefix "after_update") "after_update" "" update_fn false)
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW 'session) (list 'droptable pj_schema pj_table true))))
	(createtrigger (table src_schema src_table) (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger (table src_schema src_table) (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* prejoin_canonical_sources maps a materialized prejoin table name to an assoc
of physical prejoin column name -> source expression. make_keytable uses this
to canonicalize get_column markers on prejoin temps back to their original
source expressions, instead of baking the prejoin table name into the key name. */
(define prejoin_canonical_sources (newsession))
/* materialized_source_expr_lookup maps a temp table to an assoc of source-expression
string -> physical field name. Later GROUP stages can then rewrite both original
domain_scalar_* get_column terms and their canonicalized forms onto the prejoin's actual
physical columns without guessing from aliases or suffixes. */
(define materialized_source_expr_lookup (newsession))
/* session-sensitive runtime predicate columns must not be reused across plan
builds, because their truth value depends on current session state. */
(define session_runtime_plan_counter (newsession))
(define alias_lookup_variants (lambda (alias_)
	(reduce (filter (list
		alias_
		(visible_occurrence_alias alias_)
		(if (string? alias_) (sanitize_temp_name alias_) nil)
		(if (string? (visible_occurrence_alias alias_)) (sanitize_temp_name (visible_occurrence_alias alias_)) nil)
		(if (string? alias_) (symbol alias_) nil)
		(if (string? (visible_occurrence_alias alias_)) (symbol (visible_occurrence_alias alias_)) nil)
		(if (string? alias_) nil (string alias_))
		(if (string? (visible_occurrence_alias alias_)) nil
			(if (nil? (visible_occurrence_alias alias_)) nil (string (visible_occurrence_alias alias_)))))
		(lambda (x) (not (nil? x))))
		(lambda (acc alias_v) (append_unique acc alias_v))
		'())))
(define assoc_lookup_variants (lambda (assoc variants)
	(reduce variants (lambda (found key_v)
		(if (not (nil? found))
			found
			(if (nil? assoc) nil (get_assoc assoc key_v))))
		nil)
))
(define alias_variants_match (lambda (left right insensitive)
	(reduce (alias_lookup_variants left) (lambda (matched left_v)
		(or matched
			(reduce (alias_lookup_variants right) (lambda (matched2 right_v)
				(or matched2 ((if insensitive equal?? equal?) left_v right_v)))
				false)))
		false)
))
(define materialized_expr_alias_variants alias_lookup_variants)
(define materialized_source_expr_keys (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci) (reduce
			(reduce (materialized_expr_alias_variants alias_) (lambda (variants alias_v)
				(merge variants (list
					(list (symbol get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (symbol get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (symbol get_column) alias_v ti col ci))
					(list (quote get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (quote get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (quote get_column) alias_v ti col ci)))))
				'())
			(lambda (acc expr_variant)
				(merge acc (list
					(string expr_variant)
					(sanitize_temp_name (string expr_variant)))))
			'())
		'((quote get_column) alias_ ti col ci) (reduce
			(reduce (materialized_expr_alias_variants alias_) (lambda (variants alias_v)
				(merge variants (list
					(list (quote get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (quote get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (quote get_column) alias_v ti col ci))
					(list (symbol get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (symbol get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (symbol get_column) alias_v ti col ci)))))
				'())
			(lambda (acc expr_variant)
				(merge acc (list
					(string expr_variant)
					(sanitize_temp_name (string expr_variant)))))
			'())
		_ (list
			(string expr)
			(sanitize_temp_name (string expr))
			(string (normalize_canonical_aliases expr))
			(sanitize_temp_name (string (normalize_canonical_aliases expr))))
	)
))
(define planned_materialized_fields (newsession))
(define merge_schema_fields_unique (lambda (field_lists)
	(reduce (merge field_lists) (lambda (acc coldef)
		(if (reduce acc (lambda (found existing)
			(or found (equal?? (existing "Field") (coldef "Field"))))
			false)
			acc
			(merge acc (list coldef))))
		'())))
(define schema_bindings_to_flat_list (lambda (schema_bindings)
	(merge (extract_assoc (coalesceNil schema_bindings '()) (lambda (alias cols)
		(list alias cols))))))
(define merge_schema_binding_groups (lambda (schema_groups)
	(reduce (coalesceNil schema_groups '()) (lambda (acc bindings)
		(merge acc (schema_bindings_to_flat_list bindings)))
		'())))
(define expand_star_fields_with_schemas (lambda (fields schemas) (begin
	(define _expand_alias_cols (lambda (alias def)
		/* Visible schema exports may merge alias/planned/shown column descriptors.
		Star expansion must dedupe by field name here so wrappers over materialized
		subqueries do not emit the same visible field twice. */
		(merge (map (merge_schema_fields_unique (list def)) (lambda (coldesc)
			'((coldesc "Field") '((quote get_column) alias false (coldesc "Field") false))
	)))))
	(define _schema_matches_alias (lambda (candidate target ignorecase)
		(or ((if ignorecase equal?? equal?) candidate target)
			((if ignorecase equal?? equal?) (visible_occurrence_alias candidate) target))))
	(define _latest_schema_for_alias (lambda (target ignorecase)
		(begin
			(define latest (newsession))
			(extract_assoc schemas (lambda (alias def)
				(if (_schema_matches_alias alias target ignorecase)
					(latest "v" def)
					nil)))
			(coalesceNil (latest "v") nil))))
	(merge (extract_assoc fields (lambda (col expr) (match expr
		'((symbol get_column) nil _ "*" _) (merge (extract_assoc schemas _expand_alias_cols))
		'((quote get_column) nil _ "*" _) (merge (extract_assoc schemas _expand_alias_cols))
		'((symbol get_column) tblvar ignorecase "*" _) (begin
			(define latest_def (_latest_schema_for_alias tblvar ignorecase))
			(if (nil? latest_def) '() (_expand_alias_cols tblvar latest_def)))
		'((quote get_column) tblvar ignorecase "*" _) (begin
			(define latest_def (_latest_schema_for_alias tblvar ignorecase))
			(if (nil? latest_def) '() (_expand_alias_cols tblvar latest_def)))
		(list col expr)
)))))))
/* materialized_source_schema: resolve schema for a materialized temp source
(keytable, prejoin) using planner-internal metadata only. No storage access --
keytables/prejoins may not exist at compile time (runtime-only creation). */
(define materialized_source_schema (lambda (tschema ttbl alias schemas)
	(begin
		(define alias_cols (if (or (nil? alias) (not (has_assoc? schemas alias))) '() (coalesceNil (schemas alias) '())))
		(define planned_cols (coalesceNil (planned_materialized_fields ttbl) '()))
		(merge_schema_fields_unique (list alias_cols planned_cols)))))
(define materialized_source_physical_schema (lambda (tschema ttbl alias schemas)
	(begin
		(define planned_cols (coalesceNil (planned_materialized_fields ttbl) '()))
		/* Design contract: only columns that are part of the explicit materialized
		stage schema count as physical planner inputs. Dynamic show()/compute-column
		metadata may expose virtual cache columns that are not safe scan inputs for
		a later stage and would reintroduce early aggregate/get_column substitution.
		Visible wrappers may still consult materialized_source_schema, but scan-time
		lowering must stay on stable planned columns only. */
		(merge_schema_fields_unique (list planned_cols)))))
(define materialized_field_from_get_column_name (lambda (materialized_cols expr)
	(match expr
		'((symbol get_column) _ _ col _) (find_materialized_field_by_name materialized_cols col)
		'((quote get_column) _ _ col _) (find_materialized_field_by_name materialized_cols col)
		nil
	)
))
(define register_materialized_subquery_metadata (lambda (mat_source fields_assoc preserve_visible_boundary)
	(begin
		(define planned_schema_def (extract_assoc fields_assoc (lambda (k v)
			(list "Field" k "Type" "any" "Expr" v))))
		(define visible_schema_def (if preserve_visible_boundary
			(extract_assoc fields_assoc (lambda (k v)
				(list "Field" k "Type" "any")))
			planned_schema_def))
		(planned_materialized_fields mat_source planned_schema_def)
		(prejoin_canonical_sources mat_source
			(merge (extract_assoc fields_assoc (lambda (k v)
				(list
					(list k v)
					(list (sanitize_temp_name k) v))))))
		(materialized_source_expr_lookup mat_source
			(merge (extract_assoc fields_assoc (lambda (k v)
				(map (materialized_source_expr_keys v) (lambda (key) (list key k)))))))
		visible_schema_def
	)
))
/* Some rewrite paths carry alias provenance as (visible_alias canonical_source).
For physical scans the visible alias matters; for canonical naming / temp reuse the
canonical source side must be used so equivalent queries share the same temp cols. */
(define visible_occurrence_alias (lambda (alias_)
	(match alias_
		'(visible_alias _) visible_alias
		_ (if (string? alias_)
			(begin
				(define _parts (split alias_ "\0"))
				(if (> (count _parts) 1) (nth _parts (- (count _parts) 1)) alias_))
			alias_))))
(define scalar_helper_root_alias? (lambda (alias_)
	(begin
		(define alias_text (string alias_))
		(and
			(>= (strlen alias_text) 14)
			(equal? (substr alias_text 0 14) "domain_scalar_")
			(equal? (visible_occurrence_alias alias_text) alias_text)))))
(define scalar_helper_nested_alias? (lambda (alias_)
	(begin
		(define alias_text (string alias_))
		(and
			(>= (strlen alias_text) 14)
			(equal? (substr alias_text 0 14) "domain_scalar_")
			(not (equal? (visible_occurrence_alias alias_text) alias_text))))))
(define normalize_visible_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) ti col ci)
		'((quote get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) ti col ci)
		(cons sym args)
		(cons sym (map args normalize_visible_aliases))
		expr
	)
))
(define normalize_canonical_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias ti col ci)
			_ expr)
		'((quote get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias ti col ci)
			_ expr)
		(cons sym args)
		(cons sym (map args normalize_canonical_aliases))
		expr
	)
)))
(define planner_name_clear_case_flags (lambda (expr) (match expr
	'((symbol get_column) alias_ _ col _) (list (quote get_column) alias_ false col false)
	'((quote get_column) alias_ _ col _) (list (quote get_column) alias_ false col false)
	(cons sym args) (cons (planner_name_clear_case_flags sym) (map args planner_name_clear_case_flags))
	expr)))
/* temp table / keytable names must not embed NUL alias separators from flattened
derived tables. Keep the canonical structure, but drop the separator byte in the
physical storage name so partition files remain valid on disk. */
(define sanitize_temp_name (lambda (name)
	(if (string? name) (replace name "\0" "") name)
)))
(define query_temp_table_options '("engine" "cache"))
(define query_temp_table_options_code '(list "engine" "cache"))
/* Design contract: get_column / aggregate / window sentinels stay logical for as
long as possible and are only lowered to physical scan symbols at the final
build_scan boundary. Materialized derived tables therefore must not be keyed by
their visible SQL alias alone (`t`, `x`, ...), because later keytable names are
derived from that materialized source identity. If two unrelated wrappers reuse
the same alias on a shared server, alias-only temp identities would let stale
createcolumn results bleed across queries. The rows themselves are session-bound
so stored compute lambdas can still resolve them after the surrounding lexical
scope is gone. */
(define materialized-subquery-key (lambda (id subquery) (begin
	(define key_hash (fnv_hash (concat id ":" (string (normalize_canonical_aliases subquery)))))
	(if (scalar_helper_root_alias? id)
		(concat "__mat:domain_scalar_" key_hash)
		(concat "__mat:" key_hash)))))
(define materialized-subquery-source (lambda (id subquery)
	(list (list (quote context) "session") (materialized-subquery-key id subquery))))
(define materialized-init-head-is? (lambda (head sym)
	(or
		(equal? head sym)
		(equal? head (symbol sym))
		(equal? head (list (quote symbol) sym))
		(equal? head (list (quote quote) sym)))))
(define materialized-init-context-head? (lambda (head)
	(and (list? head)
		(equal? (count head) 2)
		(or
			(equal? (car head) (quote context))
			(equal? (car head) (symbol "context"))
			(equal? (car head) (list (quote symbol) (quote context)))
			(equal? (car head) (list (quote quote) (quote context))))
		(equal? (cadr head) "session"))))
(define materialized-init-statement? (lambda (expr) (match expr
	(cons head args)
	(and
		(materialized-init-context-head? head)
		(equal? (count args) 2)
		(string? (car args))
		(>= (strlen (car args)) 6)
		(equal? (substr (car args) 0 6) "__mat:"))
	false)))
(define strip-nested-materialized-inits (lambda (expr) (match expr
	(cons head args)
	(if (materialized-init-statement? expr)
		nil
		(if (or
			(materialized-init-head-is? head (quote begin))
			(materialized-init-head-is? head (quote !begin)))
			(cons head (filter (map args strip-nested-materialized-inits)
				(lambda (arg) (not (nil? arg)))))
			(cons head (map args strip-nested-materialized-inits))))
	expr)))
(define materialized-subquery-init (lambda (id subquery rows_expr)
	(list (list (quote context) "session")
		(materialized-subquery-key id subquery)
		(strip-nested-materialized-inits rows_expr))))
(define compact-planner-temp-column-name (lambda (name)
	(if (and (string? name) (> (strlen name) 48))
		(concat "__pc:" (sha1 name))
		name)))
(define compact-prejoin-table-name (lambda (prejoin_source_tables prejoin_col_names prejoin_condition_name)
	(concat ".prejoin:" (fnv_hash (string (list
		prejoin_source_tables
		prejoin_col_names
		prejoin_condition_name))))))
(define compact-keytable-table-name (lambda (keytable_source_name key_names condition_name)
	(begin
		(define raw_name (if (nil? condition_name)
			(concat "." keytable_source_name ":" key_names)
			(concat "." keytable_source_name ":" key_names "|" condition_name)))
		(if (> (strlen raw_name) 96)
			(concat ".keytable:" (fnv_hash raw_name))
			raw_name))))
/* planner_collect_rows_ast: execute inner_plan through a sink callback and
persist produced rows in a session list. Keep this as the fallback bridge for
runtime materialization paths that still operate outside the logical IR. */
(define planner_collect_rows_ast (lambda (rows_sym sink_sym item_sym inner_plan limit_val cnt_sym row_filter_expr) (begin
	(define append_row_ast (list rows_sym "rows"
		(list (quote merge) (list rows_sym "rows") (list (quote cons) item_sym '()))))
	(define filtered_append_row_ast
		(list (quote if) (coalesceNil row_filter_expr true)
			append_row_ast
			nil))
	(list (quote begin)
		(list (quote set) rows_sym (list (quote newsession)))
		(list rows_sym "rows" '())
		(if (nil? limit_val)
			(list (quote define) sink_sym
				(list (quote lambda) (list item_sym)
					filtered_append_row_ast))
			(list (quote begin)
				(list (quote set) cnt_sym 0)
				(list (quote define) sink_sym
					(list (quote lambda) (list item_sym)
						(list (quote if) (list (quote <) cnt_sym limit_val)
							(list (quote begin)
								(list (quote set) cnt_sym (list (quote +) cnt_sym 1))
								filtered_append_row_ast)
							nil)))))
		inner_plan
		(list rows_sym "rows")))))

(define planner_collect_rows_default_ast (lambda (rows_sym sink_sym item_sym inner_plan limit_val cnt_sym default_rows_expr) (begin
	(define append_row_ast (list rows_sym "rows"
		(list (quote merge) (list rows_sym "rows") (list (quote cons) item_sym '()))))
	(list (quote begin)
		(list (quote set) rows_sym (list (quote newsession)))
		(list rows_sym "rows" '())
		(if (nil? limit_val)
			(list (quote define) sink_sym
				(list (quote lambda) (list item_sym)
					append_row_ast))
			(list (quote begin)
				(list (quote set) cnt_sym 0)
				(list (quote define) sink_sym
					(list (quote lambda) (list item_sym)
						(list (quote if) (list (quote <) cnt_sym limit_val)
							(list (quote begin)
								(list (quote set) cnt_sym (list (quote +) cnt_sym 1))
								append_row_ast)
							nil)))))
		inner_plan
		(list (quote if)
			(list (quote equal?) (list rows_sym "rows") '())
			default_rows_expr
			(list rows_sym "rows"))))))
/* legacy_materialized_query_term_binding_ast: centralize the remaining
session-backed query-term materialization bridge. This is intentionally a
legacy fallback wrapper around planner_collect_rows_ast, not a new planner
primitive: callers stay responsible for registering visible schema metadata. */
(define legacy_materialized_query_term_binding_ast (lambda (id subquery rows_sym sink_sym limit_val cnt_sym) (begin
	(define mat_source (materialized-subquery-source id subquery))
	(define materialized_rows
		(planner_collect_rows_ast rows_sym sink_sym (symbol "item")
			(build_queryplan_term_with_sink subquery (list (quote callback) sink_sym))
			limit_val
			cnt_sym
			true))
	(list
		mat_source
		(materialized-subquery-init id subquery materialized_rows))
)))
/* build_legacy_prejoin_materialize_plan: isolate the remaining session/resultrow-
backed prejoin filler used by trigger backfill paths. This is intentionally a
legacy fallback wrapper; query-time prejoin filling stays on the canonical
build_queryplan row stream. */
(define build_legacy_prejoin_materialize_plan (lambda (schema prejoin_schema prejointbl prejoin_columns prejoin_column_names prejoin_source_tables raw_condition covered_partition_stages schemas replace_find_column) (begin
	(define build_materialize_scan (lambda (scan_tables scan_condition is_outermost)
		(match scan_tables
			(cons '(tblvar schema tbl isOuter joinexpr) rest) (begin
				/* columns needed from this table for materialization + condition */
				(define base_tbl (scan_tagged_table_base tbl))
				(define tagged_scan (scan_tagged_table_needs_scan_order tbl))
				(define tbl_scan_order (scan_tagged_table_order tbl))
				(define tbl_scan_limit (scan_tagged_table_limit tbl))
				(define tbl_scan_offset (scan_tagged_table_offset tbl))
				(define tbl_scan_partcols (scan_tagged_table_partition_cols tbl))
				(define tbl_scan_effective_partcols
					(if (and (nil? tbl_scan_limit) (nil? tbl_scan_offset))
						0
						tbl_scan_partcols))
				(define tbl_once_limit (scan_tagged_table_once_limit tbl))
				(define tblvar_is_scalar_helper (or (scalar_helper_root_alias? tblvar) (strlike (string tblvar) "domain_scalar_%")))
				(define tblvar_is_nested_scalar_helper (scalar_helper_nested_alias? tblvar))
				(set cols (merge_unique (list
					(extract_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map prejoin_columns (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map prejoin_columns (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
					(extract_later_joinexpr_columns_for_tblvar tblvar rest)
				)))
				(define split_is_outer (and
					isOuter
					(not tblvar_is_nested_scalar_helper)
					(or (nil? tbl_once_limit) tblvar_is_scalar_helper)))
				(match (split_scan_condition split_is_outer joinexpr scan_condition rest) '(now_condition later_condition) (begin
					(set filtercols (merge_unique (list
						(extract_columns_for_tblvar tblvar now_condition)
						(extract_outer_columns_for_tblvar tblvar now_condition))))
					(define no_op_materialize_scan (and
						(not is_outermost)
						(equal? cols '())
						(equal? filtercols '())
						(or (nil? now_condition) (equal? now_condition true))
						(or (nil? joinexpr) (equal? joinexpr true))))
					(define tblvar_schema_cols (coalesceNil (schemas tblvar) '()))
					(define tblvar_has_physical_col? (lambda (col)
						(schema_has_column? tblvar_schema_cols col false)))
					(define scalar_value_source_col
						(if (and tblvar_is_scalar_helper
							(not (tblvar_has_physical_col? "value"))
							(has? cols "value"))
							(match (filter cols (lambda (col)
								(and
									(not (equal? col "value"))
									(not (has? filtercols col))
									(tblvar_has_physical_col? col))))
								(cons first _) first
								_ nil)
							nil))
					(define physical_col_for_scan (lambda (col)
						(begin
							(define materialized_col (if (materialized-source? base_tbl)
								(begin
									(define expr_lookup (materialized_source_expr_lookup base_tbl))
									(coalesce
										(if (nil? expr_lookup) nil (expr_lookup col))
										(find_materialized_field_by_name
											(materialized_source_physical_schema schema base_tbl tblvar schemas)
											col)))
								nil))
							(coalesce materialized_col
								(if (and (equal? col "value") (not (nil? scalar_value_source_col)))
									scalar_value_source_col
									(scan_physical_col_for_tblvar schemas tblvar col))))))
					(define physical_cols (map cols physical_col_for_scan))
					(define scan_filtercols filtercols)
					(define scan_now_condition (strip_outer_scalar_helper_ref_terms
						(if (or tblvar_is_scalar_helper tblvar_is_nested_scalar_helper)
							(scalar_helper_outer_join_terms tblvar now_condition)
							now_condition)))
					(define physical_filtercols (map scan_filtercols physical_col_for_scan))
					(define materialize_lower_scan_expr (lambda (expr) (match expr
						(cons (symbol aggregate) _) expr
						'((symbol get_column) alias_ _ col _)
						(if (equal?? alias_ tblvar)
							(symbol (concat alias_ "." col))
							(list (quote outer) (symbol (concat alias_ "." col))))
						'((quote get_column) alias_ _ col _)
						(if (equal?? alias_ tblvar)
							(symbol (concat alias_ "." col))
							(list (quote outer) (symbol (concat alias_ "." col))))
						(cons sym args) (cons (materialize_lower_scan_expr sym)
							(map args materialize_lower_scan_expr))
						expr)))
					(define materialize_map_body
						(build_materialize_scan rest later_condition false))
					(define materialize_mapfn
						(list (quote lambda)
							(map cols (lambda (col) (symbol (concat tblvar "." col))))
							materialize_map_body))
					(define materialize_reduce
						(list (quote lambda) (list (quote acc) (quote sub))
							(list (quote merge) (quote acc) (quote sub))))
					(define materialize_insert_rows (lambda (rows_expr)
						(list (quote insert)
							(list (quote table) prejoin_schema prejointbl)
							(cons (quote list) prejoin_column_names)
							rows_expr
							(list)
							(list (quote lambda) (list) true)
							true)))
					(define materialize_reduce2 (if is_outermost
						(list (quote lambda) (list (quote acc) (quote shard_rows))
							(materialize_insert_rows (quote shard_rows)))
						(list (quote lambda) (list (quote acc) (quote shard_rows))
							(list (quote merge) (quote acc) (quote shard_rows)))))
					(if no_op_materialize_scan
						(build_materialize_scan rest later_condition false)
						(begin
							(define materialize_scan_ast (if tagged_scan
								(begin
									(define ordercols (extract_scan_order_cols_for_tblvar tbl_scan_order tblvar))
									(define physical_ordercols (map ordercols physical_col_for_scan))
									(define dirs (extract_scan_order_dirs_for_tblvar tbl_scan_order tblvar))
									(define materialize_order_scan_ast
										(scan_wrapper 'scan_order schema base_tbl
											(cons list physical_filtercols)
											'((quote lambda) (map scan_filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (materialize_lower_scan_expr scan_now_condition)))
											(cons list physical_ordercols)
											(cons list dirs)
											tbl_scan_effective_partcols
											(coalesceNil tbl_scan_offset 0)
											(coalesceNil tbl_scan_limit -1)
											(cons list physical_cols)
											materialize_mapfn
											materialize_reduce
											'(list)
											isOuter))
									(if is_outermost
										(begin
											(define materialize_order_rows_sym
												(symbol (concat "__prejoin_order_rows_" (fnv_hash prejointbl))))
											(list (quote begin)
												(list (quote define) materialize_order_rows_sym materialize_order_scan_ast)
												(materialize_insert_rows materialize_order_rows_sym)))
										materialize_order_scan_ast))
								(scan_wrapper 'scan schema base_tbl
									(cons list physical_filtercols)
									'((quote lambda) (map scan_filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (materialize_lower_scan_expr scan_now_condition)))
									(cons list physical_cols)
									materialize_mapfn
									materialize_reduce
									'(list)
									materialize_reduce2
									isOuter)))
							materialize_scan_ast))
				))
			)
			'() /* base case: produce one row wrapped in a list */
			(list (quote if)
				(optimize (replace_columns_from_expr (coalesceNil scan_condition true)))
				(list (quote cons)
					(runtime_list_ast (map prejoin_columns (lambda (mc) (replace_columns_from_expr (cadr mc)))))
					(list (quote list)))
				(list (quote list)))
		)
	))
	(define prejoin_materialize_fields (merge (map prejoin_columns (lambda (mc) (list (car mc) (cadr mc))))))
	(define prejoin_has_outer_join (reduce prejoin_source_tables (lambda (found td)
		(or found (match td '(_ _ _ isOuter _) isOuter false)))
		false))
	(define prejoin_use_outer_scan_shortcut
		(and prejoin_has_outer_join (equal? (coalesceNil covered_partition_stages '()) '())))
	(define prejoin_materialize_rowplan (if prejoin_use_outer_scan_shortcut
		(build_materialize_scan prejoin_source_tables raw_condition true)
		(build_queryplan schema
			prejoin_source_tables
			prejoin_materialize_fields
			raw_condition
			covered_partition_stages
			schemas
			replace_find_column
			nil)))
	(define _pj_prev_rr (symbol "__pj_prev_resultrow"))
	(define _pj_row_sym (symbol "__pj_row"))
	(if prejoin_use_outer_scan_shortcut
		prejoin_materialize_rowplan
		(list 'begin
			(list 'set _pj_prev_rr (symbol "resultrow"))
			(list 'set (symbol "resultrow")
				(list 'lambda (list _pj_row_sym)
					(list 'insert (list 'table prejoin_schema prejointbl)
						(cons 'list prejoin_column_names)
						(list 'list
							(cons 'list (map prejoin_column_names (lambda (col)
								(list 'get_assoc _pj_row_sym col)))))
						(list)
						(list 'lambda (list) true)
						true)))
			prejoin_materialize_rowplan
			(list 'set (symbol "resultrow") _pj_prev_rr)))
)))
/* register_prejoin_materialized_metadata: isolate the lineage/name registration
for prejoin-backed materialized sources. This keeps the prejoin assembly focused
on plan wiring while preserving the existing materialized-source contracts. The
caller still owns the prejoin-local canonicalizer that defines the visible
source-expression namespace for this materialized source. */
(define register_prejoin_materialized_metadata (lambda (canonicalize_prejoin_source_expr prejointbl prejoin_columns prejoin_alias_map prejoin_source_tables prejoin_schema_def) (begin
	(define _td_alias_variants (lambda (tv tschema ttbl) (begin
		(define base_ttbl (scan_tagged_table_base ttbl))
		(define _raw_aliases (filter (list
			tv
			(match tv '(visible _) visible nil)
			(visible_occurrence_alias tv)
			(coalesce (resolve_source_alias prejoin_alias_map tv) nil)
			(if (equal? (visible_occurrence_alias tv) base_ttbl) (concat tschema "." base_ttbl) nil))
			(lambda (x) (not (nil? x)))))
		(reduce (merge _raw_aliases
			(merge (map _raw_aliases (lambda (alias_v)
				(if (string? alias_v) (list (sanitize_temp_name alias_v)) '())))))
			(lambda (acc alias_v)
				(if (or (nil? alias_v) (has? acc alias_v))
					acc
					(merge acc (list alias_v))))
			'()))))
	(define prejoin_variant_exprs (lambda (expr) (match expr
		'((symbol get_column) alias_ ti col ci) (merge
			(list expr
				(canonicalize_prejoin_source_expr expr)
				(rewrite_source_aliases prejoin_alias_map expr))
			(merge (map prejoin_source_tables (lambda (td) (match td '(tv tschema ttbl _ _)
				(if (has? (_td_alias_variants tv tschema ttbl) alias_)
					(map (_td_alias_variants tv tschema ttbl) (lambda (alias_v)
						(list (quote get_column) alias_v ti col ci)))
					'())
				'())))))
		'((quote get_column) alias_ ti col ci) (merge
			(list expr
				(canonicalize_prejoin_source_expr expr)
				(rewrite_source_aliases prejoin_alias_map expr))
			(merge (map prejoin_source_tables (lambda (td) (match td '(tv tschema ttbl _ _)
				(if (has? (_td_alias_variants tv tschema ttbl) alias_)
					(map (_td_alias_variants tv tschema ttbl) (lambda (alias_v)
						(list (quote get_column) alias_v ti col ci)))
					'())
				'())))))
		_ (list expr
			(canonicalize_prejoin_source_expr expr)
			(rewrite_source_aliases prejoin_alias_map expr)))))
	(define prejoin_variant_names (lambda (expr)
		(reduce (map (prejoin_variant_exprs expr) (lambda (variant_expr)
			(sanitize_temp_name
				(serialize_canonical_expr
					(canonicalize_expr
						(normalize_canonical_aliases (canonicalize_prejoin_source_expr variant_expr))
						prejoin_alias_map)))))
			(lambda (acc variant_name) (append_unique acc variant_name))
			'())))
	(prejoin_canonical_sources prejointbl
		(merge (map prejoin_columns (lambda (mc) (begin
			(define source_expr (canonicalize_prejoin_source_expr (cadr mc)))
			(map (reduce (cons (car mc) (prejoin_variant_names (cadr mc)))
				(lambda (acc variant_name) (append_unique acc variant_name))
				'())
				(lambda (variant_name) (list variant_name source_expr))))))))
	(materialized_source_expr_lookup prejointbl
		(merge (map prejoin_columns (lambda (mc) (begin
			(define variant_exprs (reduce (prejoin_variant_exprs (cadr mc))
				(lambda (acc variant_expr) (append_unique acc variant_expr))
				'()))
			(merge (map variant_exprs (lambda (variant_expr)
				(map (materialized_source_expr_keys variant_expr) (lambda (k) (list k (car mc))))))))))))
	(planned_materialized_fields prejointbl prejoin_schema_def)
	true
)))
(define materialized-source? (lambda (table-source)
	(or
		(and (string? table-source) (>= (strlen table-source) 1) (equal? (substr table-source 0 1) "."))
		(match table-source
			(cons (cons (symbol context) '("session")) _) true
			(cons (cons '(quote context) '("session")) _) true
			false
	))
))
(define planner-temp-source-name (lambda (tbl tblvar)
	(if (string? tbl)
		tbl
		(if (materialized-source? tbl)
			(concat "mat_" (fnv_hash (string tbl)))
			(string tblvar)))))
/* rewrite_source_aliases: replace get_column table aliases according to alias_map.
Used to store prejoin lineage in the same canonical source namespace that also
defines the physical prejoin column names. */
(define resolve_source_alias (lambda (alias_map alias_)
	(coalesce
		(assoc_lookup_variants alias_map (map (alias_lookup_variants alias_) string))
		alias_)))
(define rewrite_source_aliases (lambda (alias_map expr)
	(match (normalize_canonical_aliases expr)
		'((symbol get_column) alias_ ti col ci)
		(list (quote get_column) (resolve_source_alias alias_map alias_) ti col ci)
		'((quote get_column) alias_ ti col ci)
		(list (quote get_column) (resolve_source_alias alias_map alias_) ti col ci)
		(cons sym args)
		(cons sym (map args (lambda (arg) (rewrite_source_aliases alias_map arg))))
		expr
		(if (equal? expr (symbol (string expr)))
			(begin
				(define resolved (resolve_source_alias alias_map expr))
				(if (equal? resolved expr)
					expr
					(symbol (string resolved))))
			expr)
	)
))
/* Planner contract: schema-based case repair is an untangle_query concern.
Once a logical expression leaves untangle_query, every get_column in planner IR
must already be `(get_column exact_alias false exact_field false)`.
canonicalize_expr/serialize_canonical_expr and later physical plan stages therefore work strictly
case-sensitively: they may rewrite aliases in a canonical source namespace, but
they must not guess/fix alias or column casing anymore. */
(define logical_expr_has_case_flags (lambda (expr) (match expr
	'((symbol get_column) _ ti _ ci) (or ti ci)
	'((quote get_column) _ ti _ ci) (or ti ci)
	(cons sym args) (if (_is_opaque_scope_sym sym)
		false
		(or (logical_expr_has_case_flags sym)
			(reduce args (lambda (found arg) (or found (logical_expr_has_case_flags arg))) false)))
	false
)))
(define require_canonical_logical_expr (lambda (context expr)
	(if (logical_expr_has_case_flags expr)
		(error (concat "planner contract violated: " context " still contains case-insensitive get_column markers in " (serialize expr)))
		expr)
))
/* Naming contract:
- canonicalize_expr rewrites planner-local aliases into the stable canonical
source namespace, but only for already-canonical logical expressions
- serialize_canonical_expr turns that canonical logical IR into a stable key
Callers that need the canonical expression for more than one downstream use must
keep it in a local define instead of rebuilding it from scratch. */
(define canonicalize_expr (lambda (expr alias_map)
	(rewrite_source_aliases alias_map
		(require_canonical_logical_expr "canonicalize_expr"
			(planner_name_clear_case_flags expr)))
))
(define serialize_canonical_expr (lambda (expr)
	(serialize (require_canonical_logical_expr "serialize_canonical_expr" expr))
))
/* Explain helpers: keep planner debugging on a stable, compact serialization
surface so tests can assert planner structure without depending on pretty-print
layout. */
(define explain_emit_rows (lambda (rows)
	(cons (quote begin) (map rows (lambda (row)
		(list (quote resultrow) (cons (quote list) row))))))
)
(define planner_debug_settings (newsession))
(define planner_debug_scalar_events (newsession))
(define planner_debug_scalar_trace_enabled (lambda ()
	(equal? (planner_debug_settings "scalar-trace") true)))
(define planner_debug_reset_scalar_events (lambda ()
	(planner_debug_scalar_events "rows" '())))
(define planner_debug_record_scalar_event (lambda (kind reason)
	(if (planner_debug_scalar_trace_enabled)
		(planner_debug_scalar_events "rows" (merge
			(coalesceNil (planner_debug_scalar_events "rows") '())
			(list (list kind reason))))
		nil)))
(define planner_debug_get_scalar_events (lambda ()
	(coalesceNil (planner_debug_scalar_events "rows") '())))
(define explain_plan_root_with_scalar_debug (lambda (plan) (begin
	(define scalar_events (planner_debug_get_scalar_events))
	(if (equal? scalar_events '())
		(explain_plan_root plan)
		(concat (explain_plan_root plan) " scalar-events=" (serialize scalar_events))))))
(define explain_queryplan_code (lambda (query) (begin
	(define plan (build_queryplan_term query))
	(pretty_print plan (size plan)))))
(define scalar_subselect_inline_reason (lambda (_agg_args direct_agg_stages_simple raw_contains_skip_level_nested_outer_ref scalar_uses_session_state stage2_post_group_condition stage2_group tables2 scalar_has_outer_ref)
	(if (nil? _agg_args)
		(quote legacy-fallback-non-aggregate)
		(if (not (equal? (count _agg_args) 3))
			(quote legacy-fallback-non-trivial-aggregate)
			(if (not direct_agg_stages_simple)
				(quote legacy-fallback-complex-group-stage)
				(if (and raw_contains_skip_level_nested_outer_ref (not scalar_uses_session_state))
					(quote legacy-fallback-skip-level-outer-ref)
					(if (not (nil? stage2_post_group_condition))
						(quote legacy-fallback-post-group-filter)
						(if (not (or (nil? stage2_group) (equal? stage2_group '()) (equal? stage2_group '(1))))
							(quote legacy-fallback-explicit-group-keys)
							(if (or (nil? tables2) (equal? tables2 '()))
								(quote legacy-fallback-no-inner-tables)
								(if (not (or scalar_has_outer_ref scalar_uses_session_state))
									(quote legacy-fallback-uncorrelated-aggregate)
									(quote inline-direct-agg-scan))))))))))))
(define scalar_subselect_inline_strategy scalar_subselect_inline_reason)
(define scalar_subselect_lowering_reason_from_facts (lambda (_has_outer _has_agg_or_stage _outer_refs_are_direct_columns _outer_has_group _contains_inner_select_marker _value_expr _value_expr_is_direct_column _domain_preserving_outer_refs _allow_grouped_direct_non_equality_outer)
	(if (not _has_outer)
		(quote prefer-unnest)
		(if (nil? _value_expr)
			(quote inline-missing-value-expr)
			(if _has_agg_or_stage
				(if (not (or _domain_preserving_outer_refs _allow_grouped_direct_non_equality_outer))
					(quote inline-grouped-non-domain-correlation)
					(if (and _contains_inner_select_marker (not _allow_grouped_direct_non_equality_outer))
						(quote inline-grouped-inner-select-marker)
						(quote prefer-unnest)))
				(if (not _outer_refs_are_direct_columns)
					(quote inline-non-grouped-non-direct-outer-refs)
					(if _outer_has_group
						(quote prefer-unnest)
						(if _contains_inner_select_marker
							(quote prefer-unnest)
							(quote prefer-unnest)))))))))
(define planner_scalar_subselect_inline_reason scalar_subselect_inline_reason)
(define planner_scalar_subselect_inline_strategy scalar_subselect_inline_strategy)
(define planner_scalar_subselect_lowering_reason_from_facts scalar_subselect_lowering_reason_from_facts)
(define explain_plan_root (lambda (plan)
	(match plan
		(cons sym _) (string sym)
		_ (string plan)
	)
))
(define explain_normalize_stage (lambda (stage)
	(list
		(cons (quote group-cols) (coalesceNil (stage_group_cols stage) '()))
		(list (quote having) (stage_having_expr stage))
		(list (quote order) (coalesceNil (stage_order_list stage) '()))
		(list (quote limit-partition-cols) (coalesceNil (stage_limit_partition_cols stage) 0))
		(list (quote limit) (stage_limit_val stage))
		(list (quote offset) (stage_offset_val stage))
		(list (quote group-alias) nil)
		(list (quote dedup) (stage_is_dedup stage))
		(list (quote partition-aliases) (stage_partition_aliases stage))
		(list (quote init) (stage_init_code stage))
	)
))
(define explain_normalize_stages (lambda (stages)
	(map stages explain_normalize_stage)
))
/* explain_queryplan_ir: expose planner IR around the logical query-term planner.
Returns compact stage/kind/value rows for stable SQL-level inspection. */
(define explain_queryplan_ir (lambda (query) (begin
	(planner_debug_settings "scalar-trace" true)
	(planner_debug_reset_scalar_events)
	(define logical_term (prepare_queryplan_term query))
	(match logical_term
		'(select_core_term schema tables fields condition groups schemas replace_find_column init) (begin
			(define _uq_7tuple (list schema tables fields condition groups schemas replace_find_column))
			(define _jr_result (apply join_reorder _uq_7tuple))
			(define _plan (apply build_queryplan (merge _jr_result (list nil))))
			(define _rows (list
				(list "stage" "untangle" "kind" "tables" "value" (serialize tables))
				(list "stage" "untangle" "kind" "fields" "value" (serialize fields))
				(list "stage" "untangle" "kind" "condition" "value" (serialize condition))
				(list "stage" "untangle" "kind" "groups" "value" (serialize (explain_normalize_stages groups)))
				(list "stage" "untangle" "kind" "init" "value" (serialize init))
				(list "stage" "reorder" "kind" "tables" "value" (serialize (nth _jr_result 1)))
				(list "stage" "reorder" "kind" "changed" "value" (not (equal? tables (nth _jr_result 1))))
				(list "stage" "plan" "kind" "root" "value" (explain_plan_root_with_scalar_debug _plan))))
			(planner_debug_settings "scalar-trace" false)
			(explain_emit_rows _rows))
		'(union_all_term branches order limit offset)
		(begin
			(planner_debug_settings "scalar-trace" false)
			(explain_emit_rows (list
				(list "stage" "term" "kind" "root" "value" "union_all")
				(list "stage" "term" "kind" "branches" "value" (count branches))
				(list "stage" "term" "kind" "order" "value" (serialize (coalesceNil order '())))
				(list "stage" "term" "kind" "limit" "value" (serialize limit))
				(list "stage" "term" "kind" "offset" "value" (serialize offset)))))
		_ (error "invalid logical query term for EXPLAIN IR")
	)
)))
/* explain_queryplan_reorder: focused view for join-reorder work. */
(define explain_queryplan_reorder (lambda (query) (begin
	(define _uq_result (apply untangle_query (merge query (list nil))))
	(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
	(define _jr_result (apply join_reorder _uq_7tuple))
	(define table_rows_for_stage (lambda (stage_name tables)
		(map (produceN (count tables)) (lambda (idx) (match (nth tables idx)
			'(alias schema tbl isOuter joinexpr)
			(list
				"stage" stage_name
				"position" idx
				"alias" (string alias)
				"schema" (string schema)
				"table" (string tbl)
				"outer" isOuter
				"joinexpr" (serialize (coalesceNil joinexpr true)))
			_ (list
				"stage" stage_name
				"position" idx
				"alias" ""
				"schema" ""
				"table" (serialize (nth tables idx))
				"outer" nil
				"joinexpr" "true"
			)
	))))
))
	(explain_emit_rows (merge
		(table_rows_for_stage "untangle" (nth _uq_result 1))
		(table_rows_for_stage "reorder" (nth _jr_result 1))
	))
)))
/* Compatibility wrapper for older call sites. New planner code should keep the
canonical expression in a local define and only serialize it at the edge. */
(define canonical_expr_name (lambda (expr columns params alias_map)
	(serialize_canonical_expr (canonicalize_expr (planner_name_clear_case_flags expr) alias_map))
))
/* build_occurrence_alias_map: assign a stable canonical source namespace to query
aliases. Single occurrences keep the physical table name for maximal reuse.
If the same physical table appears multiple times in one query tree, append an
occurrence index so self-joins do not collapse distinct roles. */
(define build_occurrence_alias_map (lambda (tables) (begin
	(define total_counts (newsession))
	(map tables (lambda (td) (match td
		'(_ tschema ttbl _ _) (begin
			(define base_ttbl (scan_tagged_table_base ttbl))
			(define src (concat tschema "." base_ttbl))
			(total_counts src (+ 1 (coalesceNil (total_counts src) 0)))
			nil)
		nil)))
	(define seen_counts (newsession))
	(define alias_pairs (map tables (lambda (td) (match td
		'(tv tschema ttbl _ _) (begin
			(define base_ttbl (scan_tagged_table_base ttbl))
			(define src (concat tschema "." base_ttbl))
			(define idx (coalesceNil (seen_counts src) 0))
			(define canon (if (> (coalesceNil (total_counts src) 0) 1)
				(concat src "#" idx)
				src))
			(seen_counts src (+ idx 1))
			(list tv canon))
		(list "" "")))))
	(reduce alias_pairs
		(lambda (acc pair) (match pair
			'(tv canon) (begin
				(define visible (visible_occurrence_alias tv))
				(define tv_sanitized (if (string? tv) (sanitize_temp_name tv) tv))
				(define visible_sanitized (if (string? visible) (sanitize_temp_name visible) visible))
				(reduce (filter (list tv visible tv_sanitized visible_sanitized canon)
					(lambda (alias_v) (not (nil? alias_v))))
					(lambda (acc2 alias_v) (set_assoc acc2 (string alias_v) canon))
					acc))
			(list "" "") acc))
		'()))
)))

(define rewrite_materialized_source_columns (lambda (tbl tblvar expr)
	(begin
		(define source_alias_map (prejoin_canonical_sources tbl))
		(if (or (nil? source_alias_map) (not (list? source_alias_map)))
			expr
			(match expr
				'((symbol get_column) (eval tblvar) ti col ci)
				(begin
					(coalesce (source_alias_map col) expr))
				'((quote get_column) (eval tblvar) ti col ci)
				(begin
					(coalesce (source_alias_map col) expr))
				(cons sym args)
				(cons sym (map args (lambda (arg) (rewrite_materialized_source_columns tbl tblvar arg))))
				expr)))))

/* lower_materialized_source_expr: deterministically lower expressions on a
materialized temp source onto the original source-expression namespace recorded
for that temp source. This keeps keytable/cache names stable and avoids leaking
raw domain_scalar_* occurrence aliases into physical temp column names. */
(define lower_materialized_source_expr (lambda (tbl tblvar expr)
	(begin
		(define expr_lookup (materialized_source_expr_lookup tbl))
		(define source_alias_map (prejoin_canonical_sources tbl))
		(define planned_cols (coalesceNil (planned_materialized_fields tbl) '()))
		(define lower_node (lambda (node) (begin
			(define normalized_node (normalize_canonical_aliases node))
			(define node_keys (materialized_source_expr_keys node))
			(define planned_source_expr
				(match node
					'((symbol get_column) _ _ col _)
					(reduce planned_cols (lambda (found coldef)
						(if (not (nil? found))
							found
							(if (equal? (coldef "Field") col)
								(coalesceNil (coldef "Expr") nil)
								nil)))
						nil)
					'((quote get_column) _ _ col _)
					(reduce planned_cols (lambda (found coldef)
						(if (not (nil? found))
							found
							(if (equal? (coldef "Field") col)
								(coalesceNil (coldef "Expr") nil)
								nil)))
						nil)
					nil))
			(define planned_field
				(reduce planned_cols (lambda (found coldef)
					(if (not (nil? found))
						found
						(begin
							(define source_expr (coalesceNil (coldef "Expr") nil))
							(if (and (not (nil? source_expr))
								(or (equal? (normalize_canonical_aliases source_expr) normalized_node)
									(reduce node_keys (lambda (matched key)
										(or matched (has? (materialized_source_expr_keys source_expr) key)))
										false)))
								(coldef "Field")
								nil))))
					nil))
			(define direct_source_expr
				(match node
					'((symbol get_column) _ _ col _)
					(coalesce planned_source_expr
						(if (nil? source_alias_map) nil (source_alias_map col)))
					'((quote get_column) _ _ col _)
					(coalesce planned_source_expr
						(if (nil? source_alias_map) nil (source_alias_map col)))
					nil))
			(define direct_field
				(coalesce planned_field
					(if (nil? expr_lookup)
						nil
						(reduce (materialized_source_expr_keys node) (lambda (found key)
							(if (not (nil? found))
								found
								(coalesce (expr_lookup key) nil)))
							nil))))
			(if (not (nil? direct_source_expr))
				direct_source_expr
				(if (not (nil? direct_field))
					(coalesce
						(if (nil? source_alias_map) nil (source_alias_map direct_field))
						node)
					(match node
						(cons sym args)
						(cons sym (map args lower_node))
						_
						(rewrite_materialized_source_columns tbl tblvar node)))))))
		(lower_node expr)
)))

/* preserve_current_materialized_field_refs: for naming/group identity on the
current materialized source, keep direct output-field references logical instead
of inlining their full source expression. This prevents wrapper GROUP stages
from serializing nested window/subquery materializations into keytable/agg names
while still recursing into deeper helper lineage when needed. */
(define preserve_current_materialized_field_refs (lambda (tbl tblvar expr)
	(begin
		(define planned_cols (coalesceNil (planned_materialized_fields tbl) '()))
		(define preserve_node (lambda (node) (match node
			'((symbol get_column) (eval tblvar) _ col _)
			(if (nil? (find_materialized_field_by_name planned_cols col))
				(lower_materialized_source_expr tbl tblvar node)
				(list (quote get_column) tblvar false col false))
			'((quote get_column) (eval tblvar) _ col _)
			(if (nil? (find_materialized_field_by_name planned_cols col))
				(lower_materialized_source_expr tbl tblvar node)
				(list (quote get_column) tblvar false col false))
			(cons sym args) (cons sym (map args preserve_node))
			_ node)))
		(if (materialized-source? tbl)
			(preserve_node expr)
			expr)
)))

/* returns a list of all tblvar aliases referenced via get_column in expr */
(define extract_tblvars (lambda (expr)
	(match expr
		'((symbol get_column) tblvar _ _ _) (if (nil? tblvar) '() (list tblvar))
		'((quote get_column) tblvar _ _ _) (if (nil? tblvar) '() (list tblvar))
		(cons sym args) (reduce args (lambda (acc arg)
			(merge_unique acc (extract_tblvars arg)))
			'())
		'()
	)
))

/* returns a list of '(string...) */
(define extract_columns_for_tblvar (lambda (tblvar expr)
	(match expr
		'((symbol get_column) (eval tblvar) _ col _) (if (equal? col "*") '() (list col)) /* TODO: case matching */
		'((quote get_column) (eval tblvar) _ col _) (if (equal? col "*") '() (list col))
		(cons sym args) /* function call */ (reduce args (lambda (acc arg)
			(merge_unique acc (extract_columns_for_tblvar tblvar arg)))
			'())
		'()
	)
))

/* extracts unqualified column references (get_column nil ...) from an expression.
Used by derived-table flattening so wrapper columns referenced without an alias
still keep their projected field alive. */
(define extract_unqualified_columns (lambda (expr)
	(match expr
		'((symbol get_column) nil _ col _) (if (equal? col "*") '() (list col))
		'((quote get_column) nil _ col _) (if (equal? col "*") '() (list col))
		(cons sym args) (reduce args (lambda (acc arg)
			(merge_unique acc (extract_unqualified_columns arg)))
			'())
		'()
	)
))

/* true iff expr contains a direct tblvar.* wildcard reference.
Used by derived-table flattening: an empty referenced-column set can mean either
"outer query needs nothing from this wrapper" (good, prune all inner fields) or
"outer query asked for tblvar.*" (must retain the full projected column set). */
(define expr_has_tblvar_wildcard_ref (lambda (tblvar expr)
	(match expr
		'((symbol get_column) (eval tblvar) _ col _) (equal? col "*")
		'((quote get_column) (eval tblvar) _ col _) (equal? col "*")
		(cons sym args) (reduce args (lambda (found arg) (or found (expr_has_tblvar_wildcard_ref tblvar arg))) false)
		false
	)
))

/* true iff expr contains an unqualified * reference.
Used by derived-table flattening: SELECT * over a wrapper still needs the full
projected column set even when no alias-qualified t.* appears. */
(define expr_has_unqualified_wildcard_ref (lambda (expr)
	(match expr
		'((symbol get_column) nil _ col _) (equal? col "*")
		'((quote get_column) nil _ col _) (equal? col "*")
		(cons sym args) (reduce args (lambda (found arg) (or found (expr_has_unqualified_wildcard_ref arg))) false)
		false
	)
))

/* changes (get_column tblvar ti col ci) into its symbol */
(define replace_columns_from_expr (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) /* aggregates: don't dive in */ expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		'((quote get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
		expr /* literals */
	)
))

(define replace_columns_from_expr_for_scan (lambda (current_tblvar expr) (begin
	(define scan_expr (if (strlike (string current_tblvar) "domain_scalar_%")
		(scalar_helper_outer_join_terms current_tblvar expr)
		expr))
	(match scan_expr
		(cons (symbol aggregate) args) expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		'((quote get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		'((symbol outer) symname) (match (split (string symname) ".")
			(list outer_tbl outer_col) (if (equal?? outer_tbl current_tblvar)
				(symbol (concat outer_tbl "." outer_col))
				expr)
			(list (quote outer) (replace_columns_from_expr_for_scan current_tblvar symname)))
		'((quote outer) symname) (match (split (string symname) ".")
			(list outer_tbl outer_col) (if (equal?? outer_tbl current_tblvar)
				(symbol (concat outer_tbl "." outer_col))
				expr)
			(list (quote outer) (replace_columns_from_expr_for_scan current_tblvar symname)))
		(cons sym args) (cons sym (map args (lambda (arg) (replace_columns_from_expr_for_scan current_tblvar arg))))
		scan_expr
	)
)))

/* scan-tagged tables: keep once-limit/order metadata on the table entry so
build_scan can lower scalar subselects without extra once-limit stages. */
(define make_scan_tagged_table_parts (lambda (base order limit offset partition_cols once_limit outer_sources)
	(if (and (equal? (coalesceNil order '()) '())
		(nil? limit)
		(nil? offset)
		(equal? (coalesceNil partition_cols 0) 0)
		(nil? once_limit)
		(or (nil? outer_sources) (equal? outer_sources '())))
		base
		(if (or (nil? outer_sources) (equal? outer_sources '()))
			(list (quote scan-tagged-table) base (coalesceNil order '()) limit offset (coalesceNil partition_cols 0) once_limit)
			(list (quote scan-tagged-table) base (coalesceNil order '()) limit offset (coalesceNil partition_cols 0) once_limit outer_sources)))
))
(define make_scan_tagged_table (lambda (base order limit offset partition_cols once_limit)
	(make_scan_tagged_table_parts base order limit offset partition_cols once_limit nil)
))
(define scan_tagged_table_base (lambda (tbl) (match tbl
	'(scan-tagged-table base _ _ _ _ _) base
	'(scan-tagged-table base _ _ _ _ _ _) base
	'((symbol scan-tagged-table) base _ _ _ _ _) base
	'((symbol scan-tagged-table) base _ _ _ _ _ _) base
	'((quote scan-tagged-table) base _ _ _ _ _) base
	'((quote scan-tagged-table) base _ _ _ _ _ _) base
	tbl
)))
(define scan_tagged_table_order (lambda (tbl) (match tbl
	'(scan-tagged-table _ order _ _ _ _) (coalesceNil order '())
	'(scan-tagged-table _ order _ _ _ _ _) (coalesceNil order '())
	'((symbol scan-tagged-table) _ order _ _ _ _) (coalesceNil order '())
	'((symbol scan-tagged-table) _ order _ _ _ _ _) (coalesceNil order '())
	'((quote scan-tagged-table) _ order _ _ _ _) (coalesceNil order '())
	'((quote scan-tagged-table) _ order _ _ _ _ _) (coalesceNil order '())
	'()
)))
(define scan_tagged_table_limit (lambda (tbl) (match tbl
	'(scan-tagged-table _ _ limit _ _ _) limit
	'(scan-tagged-table _ _ limit _ _ _ _) limit
	'((symbol scan-tagged-table) _ _ limit _ _ _) limit
	'((symbol scan-tagged-table) _ _ limit _ _ _ _) limit
	'((quote scan-tagged-table) _ _ limit _ _ _) limit
	'((quote scan-tagged-table) _ _ limit _ _ _ _) limit
	nil
)))
(define scan_tagged_table_offset (lambda (tbl) (match tbl
	'(scan-tagged-table _ _ _ offset _ _) offset
	'(scan-tagged-table _ _ _ offset _ _ _) offset
	'((symbol scan-tagged-table) _ _ _ offset _ _) offset
	'((symbol scan-tagged-table) _ _ _ offset _ _ _) offset
	'((quote scan-tagged-table) _ _ _ offset _ _) offset
	'((quote scan-tagged-table) _ _ _ offset _ _ _) offset
	nil
)))
(define scan_tagged_table_partition_cols (lambda (tbl) (match tbl
	'(scan-tagged-table _ _ _ _ partition_cols _) (coalesceNil partition_cols 0)
	'(scan-tagged-table _ _ _ _ partition_cols _ _) (coalesceNil partition_cols 0)
	'((symbol scan-tagged-table) _ _ _ _ partition_cols _) (coalesceNil partition_cols 0)
	'((symbol scan-tagged-table) _ _ _ _ partition_cols _ _) (coalesceNil partition_cols 0)
	'((quote scan-tagged-table) _ _ _ _ partition_cols _) (coalesceNil partition_cols 0)
	'((quote scan-tagged-table) _ _ _ _ partition_cols _ _) (coalesceNil partition_cols 0)
	0
)))
(define scan_tagged_table_once_limit (lambda (tbl) (match tbl
	'(scan-tagged-table _ _ _ _ _ once_limit) once_limit
	'(scan-tagged-table _ _ _ _ _ once_limit _) once_limit
	'((symbol scan-tagged-table) _ _ _ _ _ once_limit) once_limit
	'((symbol scan-tagged-table) _ _ _ _ _ once_limit _) once_limit
	'((quote scan-tagged-table) _ _ _ _ _ once_limit) once_limit
	'((quote scan-tagged-table) _ _ _ _ _ once_limit _) once_limit
	nil
)))
(define scan_tagged_table_outer_sources (lambda (tbl) (match tbl
	'(scan-tagged-table _ _ _ _ _ _ outer_sources) (coalesceNil outer_sources '())
	'((symbol scan-tagged-table) _ _ _ _ _ _ outer_sources) (coalesceNil outer_sources '())
	'((quote scan-tagged-table) _ _ _ _ _ _ outer_sources) (coalesceNil outer_sources '())
	'()
)))
(define scan_tagged_table_with_outer_sources (lambda (tbl outer_sources)
	(if (or (nil? outer_sources) (equal? outer_sources '()))
		tbl
		(make_scan_tagged_table_parts
			(scan_tagged_table_base tbl)
			(scan_tagged_table_order tbl)
			(scan_tagged_table_limit tbl)
			(scan_tagged_table_offset tbl)
			(scan_tagged_table_partition_cols tbl)
			(scan_tagged_table_once_limit tbl)
			outer_sources)
)))
(define scan_tagged_table_needs_scan_order (lambda (tbl)
	(or (not (equal? (scan_tagged_table_order tbl) '()))
		(not (nil? (scan_tagged_table_limit tbl)))
		(not (nil? (scan_tagged_table_offset tbl)))
		(not (equal? (scan_tagged_table_partition_cols tbl) 0)))
))
/* Ordered scalar scans currently lower physically via partitioned scan_order.
The later logical ORDER/LIMIT normalization should keep targeting these small
helpers instead of rebuilding alias/order logic inline inside untangle_query. */
(define scalar_scan_domain_order (lambda (domain_cols rewrite_inner_expr scalar_alias)
	(filter
		(map domain_cols (lambda (dc) (list (rewrite_inner_expr (nth dc 0)) '<)))
		(lambda (oi) (match oi '(col _)
			(match col
				'((symbol get_column) a _ _ _) (equal? a scalar_alias)
				'((quote get_column) a _ _ _) (equal? a scalar_alias)
				false)
			false))
)))
(define scalar_scan_rewrite_order (lambda (order_list rewrite_expr)
	(map (coalesceNil order_list '()) (lambda (oi) (match oi
		'(col dir) (list (rewrite_expr col) dir)
		oi)))
))
(define scalar_scan_order_supported (lambda (order_list scalar_alias)
	(reduce order_list (lambda (acc oi) (and acc (match oi
		'(col _dir) (match col
			'((symbol get_column) a _ _ _) (equal? a scalar_alias)
			'((quote get_column) a _ _ _) (equal? a scalar_alias)
			false)
		false)))
		true)
))
/* Logical scalar-partition stage: encode partition-topk semantics on the
stage itself and leave promise/runtime details to build_scan. The stage still
records scalar cardinality via once-limit, but no once-limit contract or
runtime promise name is created during unnesting anymore. */
(define make_scalar_partition_stage (lambda (order_list limit_value offset_value partition_cols aliases outer_sources) (begin
	(define stage_order (coalesceNil order_list '()))
	(define stage_partition_cols (coalesceNil partition_cols 0))
	(define stage_offset (coalesceNil offset_value 0))
	(define stage_once_limit (if (and (not (nil? limit_value)) (<= limit_value 1))
		1
		2))
	(define stage_limit (if (nil? limit_value)
		stage_once_limit
		(if (<= limit_value 1) limit_value 2)))
	(stage_with_outer_sources
		(make_stage '() nil stage_order stage_partition_cols stage_limit stage_offset false aliases nil nil stage_once_limit)
		outer_sources)
)))
(define scalar_subselect_alias_map (lambda (base_tables single_tbl scalar_prefix)
	(map base_tables (lambda (td) (match td
		'(alias _ _ _ _) (list alias (if single_tbl scalar_prefix (concat scalar_prefix "\0" alias)))
		(list "" "")))))
)
(define scalar_subselect_lookup_alias (lambda (alias_map alias_name)
	(reduce alias_map (lambda (acc pair)
		(if (nil? acc)
			(if (equal?? alias_name (nth pair 0)) (nth pair 1) nil)
			acc))
		nil)
))
(define scalar_subselect_lookup_original_alias (lambda (alias_map renamed_alias)
	(reduce alias_map (lambda (acc pair)
		(if (nil? acc)
			(if (equal? (nth pair 1) renamed_alias) (nth pair 0) nil)
			acc))
		nil)
))
(define scalar_subselect_rewrite_prefixed_expr (lambda (expr lookup_alias) (match expr
	'((symbol get_column) alias_ ti col ci) (begin
		(define na (lookup_alias alias_))
		(if (nil? na) expr (list (quote get_column) na false col false)))
	'((quote get_column) alias_ ti col ci) (begin
		(define na (lookup_alias alias_))
		(if (nil? na) expr (list (quote get_column) na false col false)))
	(cons sym args) (cons (scalar_subselect_rewrite_prefixed_expr sym lookup_alias) (map args (lambda (arg)
		(scalar_subselect_rewrite_prefixed_expr arg lookup_alias))))
	expr
)))
(define scalar_subselect_prefixed_tables (lambda (tables lookup_alias rewrite_expr)
	(map tables (lambda (td) (match td
		'(a s t io je) (list (coalesceNil (lookup_alias a) a) s t io
			(if (nil? je) nil (rewrite_expr je)))
		td)))
))
(define scalar_subselect_rewrite_alias (lambda (lookup_alias alias_name)
	(coalesceNil (lookup_alias alias_name) alias_name)
))
(define scalar_subselect_rewrite_tables (lambda (tables rewrite_expr)
	(map tables (lambda (td) (match td
		'(a s t io je) (list a s t io (if (nil? je) nil (rewrite_expr je)))
		td)))
))
(define scalar_subselect_rewrite_stages (lambda (stages rewrite_expr rewrite_alias)
	(map stages (lambda (stage)
		(rewrite_stage_for_flattened_aliases stage rewrite_expr rewrite_alias)))
))
(define scalar_subselect_rewrite_stages_with_lookup (lambda (stages rewrite_expr lookup_alias)
	(scalar_subselect_rewrite_stages stages rewrite_expr (lambda (alias_name)
		(scalar_subselect_rewrite_alias lookup_alias alias_name)))
))
(define scalar_subselect_collect_stage_outer_sources (lambda (stages)
	(merge_unique (map stages (lambda (stage)
		(coalesceNil (stage_outer_sources stage) '()))))
))
(define scalar_subselect_schema_entry (lambda (alias_name schemas2_fn) (begin
	(define schema_cols (schemas2_fn alias_name))
	(if (nil? schema_cols)
		'()
		(list alias_name schema_cols))
)))
(define scalar_subselect_passthrough_schemas (lambda (tables schemas2_fn)
	(merge (map tables (lambda (td) (match td
		'(alias_name _ _ _ _) (scalar_subselect_schema_entry alias_name schemas2_fn)
		'()))))
))
(define scalar_subselect_prefixed_schemas (lambda (prefixed_tables alias_map schemas2_fn)
	(merge (map prefixed_tables (lambda (td) (match td
		'(alias_name _ _ _ _) (begin
			(define original_alias (scalar_subselect_lookup_original_alias alias_map alias_name))
			(if (nil? original_alias)
				'()
				(begin
					(define schema_cols (schemas2_fn original_alias))
					(if (nil? schema_cols) '() (list alias_name schema_cols)))))
		'()))))
))
(define scalar_subselect_table_aliases (lambda (tables)
	(map tables (lambda (td) (match td
		'(alias_name _ _ _ _) alias_name
		"")))
))
(define scalar_subselect_correlation_info (lambda (condition_expr inner_aliases rewrite_outer_expr) (begin
	(define _ss_has_outer (lambda (expr) (unnest_expr_has_outer_ref expr inner_aliases)))
	(define _ss_cond_parts (flatten_and_terms condition_expr))
	(define _ss_inner_parts (filter _ss_cond_parts (lambda (part) (not (_ss_has_outer part)))))
	(define _ss_outer_parts (filter _ss_cond_parts (lambda (part) (_ss_has_outer part))))
	(define _ss_domain_cols (filter
		(map _ss_outer_parts (lambda (part) (unnest_correlated_domain_col part _ss_has_outer rewrite_outer_expr)))
		(lambda (x) (not (nil? x)))))
	(define _ss_extra_inner_parts (filter _ss_outer_parts (lambda (part)
		(unnest_correlated_residual_part? part _ss_has_outer))))
	(define _ss_inner_parts_combined (merge _ss_inner_parts (map _ss_extra_inner_parts rewrite_outer_expr)))
	(define _ss_inner_cond_raw
		(if (equal? (count _ss_inner_parts_combined) 0) nil
			(if (equal? (count _ss_inner_parts_combined) 1) (car _ss_inner_parts_combined)
				(cons (quote and) _ss_inner_parts_combined))))
	(list _ss_outer_parts _ss_domain_cols _ss_inner_cond_raw)
)))
(define make_once_limit_promise_name (lambda (limit_value once_limit tblvar condition joinexpr tbl) (begin
	(define contract_once_limit (if (nil? once_limit)
		(if (and (not (nil? limit_value)) (<= limit_value 1))
			1
			2)
		once_limit))
	(if (and (not (nil? tblvar)) (not (nil? tbl)) (>= contract_once_limit 2))
		(concat "__once_limit_" tblvar "_" (fnv_hash (concat condition "|" joinexpr "|" tbl)))
		nil)
)))
(define make_once_limit_scan_contract (lambda (limit_value offset_value partition_cols once_limit tblvar condition joinexpr tbl) (begin
	(define contract_once_limit (if (nil? once_limit)
		(if (and (not (nil? limit_value)) (<= limit_value 1))
			1
			2)
		once_limit))
	(define contract_limit (if (nil? once_limit)
		(if (nil? limit_value)
			2
			(if (<= limit_value 1) limit_value 2))
		(coalesceNil limit_value -1)))
	(define contract_offset (coalesceNil offset_value 0))
	(define contract_partition_cols (coalesceNil partition_cols 0))
	(define contract_promise_name (make_once_limit_promise_name limit_value once_limit tblvar condition joinexpr tbl))
	(list contract_limit contract_offset contract_partition_cols contract_once_limit contract_promise_name)
)))
(define once_limit_scan_contract_limit (lambda (contract) (nth contract 0)))
(define once_limit_scan_contract_offset (lambda (contract) (nth contract 1)))
(define once_limit_scan_contract_partition_cols (lambda (contract) (nth contract 2)))
(define once_limit_scan_contract_once_limit (lambda (contract) (nth contract 3)))
(define once_limit_scan_contract_promise_name (lambda (contract) (nth contract 4)))
(define wrap_once_limit_body (lambda (promise_name body)
	(if (nil? promise_name)
		body
		(list (quote !begin)
			(list (symbol promise_name) "once" true "Subquery returns more than 1 row")
			body))
))
(define wrap_once_limit_scan (lambda (promise_name scan_expr)
	(if (nil? promise_name)
		scan_expr
		(list (quote !begin)
			(list (quote set) (symbol promise_name) (list (quote newpromise)))
			scan_expr))
))
/* scan-codegen-table: generates a table expression for codegen */
(define scan-codegen-table (lambda (schema tbl) (match (scan_tagged_table_base tbl)
	'(materialized-subquery key) (list (list (quote context) "session") key)
	'((symbol materialized-subquery) key) (list (list (quote context) "session") key)
	'((quote materialized-subquery) key) (list (list (quote context) "session") key)
	base_tbl (list (quote table) schema base_tbl)
)))

/* tbl-define-code: generates (define tbl:schema:name (table schema name)) for init blocks */
(define tbl-define-code (lambda (schema tbl)
	(list 'define (symbol (concat "tbl:" schema ":" tbl)) (list 'table schema tbl))))

/* scan_batch peephole optimization has been moved to the Go optimizer hook
in storage/scan_batch_rewrite.go (optimizeScan → tryScanBatchRewrite). */

/* returns a list of all aggregates in this expr */
(define extract_aggregates (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) '(args)
		(cons '(quote aggregate) args) '(args)
		(cons sym args) /* function call */ (merge (map args extract_aggregates))
		/* literal */ '()
	)
))

/* session-sensitive COUNT/EXISTS stages must be classified before the physical
planner starts splitting aggregate conditions apart. */
(define expr_uses_session_state (lambda (expr)
	(match expr
		(symbol session) true
		'(quote session) true
		(cons (symbol session) _) true
		(cons '(quote session) _) true
		(cons (symbol context) '("session")) true
		(cons (cons (symbol context) '("session")) _) true
		(cons (symbol ?) '("__memcp_tx")) true
		(cons '(quote ?) '("__memcp_tx")) true
		(cons sym args) (reduce args (lambda (acc arg) (or acc (expr_uses_session_state arg))) false)
		false
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
		'((quote get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
		(cons sym args) (merge (map args extract_all_get_columns))
		'()
	)
))

/* extract_all_table_aliases: return a flat list of all table aliases referenced
via get_column nodes in an expression.  Used by LEFT JOIN pruning to detect
which tables are actually read. */
(define extract_all_table_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list (string alias_)))
		'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list (string alias_)))
		(cons sym args) (merge (map args extract_all_table_aliases))
		'()
	)
))

/* extract_scanned_tables: walk an expression AST and return all (schema table) pairs from scan/scan_order calls.
Used to detect which tables a computor lambda reads from, so we can register invalidation triggers. */
(define extract_scanned_tables (lambda (expr)
	(match expr
		(cons (symbol scan) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scan_order) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons sym args) (merge (map args extract_scanned_tables))
		'()
	)
))

/* expr_has_scan: returns true if an AST expression contains any scan node. */
(define expr_has_scan (lambda (expr)
	(match expr
		(cons head rest) (match (string head)
			"scan" true "scan_order" true "scan_batch" true
			_ (reduce rest (lambda (acc child) (or acc (expr_has_scan child))) false))
		false)))

/* expr_refs_outer_var: returns true if an expression references (var N) at the
top level (outside any nested lambda). */
(define expr_refs_outer_var (lambda (expr)
	(match expr
		(cons head rest) (match (string head)
			"var" true
			"lambda" false
			_ (reduce rest (lambda (acc child) (or acc (expr_refs_outer_var child))) false))
		false)))

/* expr_is_parallelizable: safe to wrap in a parallel thunk if it contains
a scan AND does not reference outer-scope vars. */
(define expr_is_parallelizable (lambda (expr)
	(and (expr_has_scan expr) (not (expr_refs_outer_var expr)))))

(define resultrow_call_head? (lambda (head)
	(or
		(equal? head (quote resultrow))
		(equal? (string head) "resultrow")
		(match head
			'((symbol symbol) "resultrow") true
			'((quote symbol) "resultrow") true
			'(symbol "resultrow") true
			false))))

(define scalar_resultrow_call_head? (lambda (head)
	(or
		(strlike (string head) "__scalar_resultrow_%")
		(match head
			'((symbol symbol) name) (strlike name "__scalar_resultrow_%")
			'((quote symbol) name) (strlike name "__scalar_resultrow_%")
			'(symbol name) (strlike name "__scalar_resultrow_%")
			false))))

/* parallelize_resultrows: post-processing pass over the finished query plan AST.
Rewrites (resultrow (list k1 v1 k2 v2 ...)) nodes: if >=2 value expressions
are parallelizable, wrap them in parallel_map for concurrent evaluation.
Only recurses into transparent wrappers (begin/if/define/time). */
(define parallelize_resultrows (lambda (ast)
	(match ast
		(cons head rest)
		(if (scalar_resultrow_call_head? head)
			(match rest
				(cons (cons quote_sym quoted_args) tail)
				(if (is_quote_scope_sym quote_sym)
					(begin
						(define row_items (if (and (equal? (count quoted_args) 1) (list? (car quoted_args)))
							(car quoted_args)
							(if (> (count quoted_args) 1) quoted_args nil)))
						(if (nil? row_items)
							(cons head (map rest parallelize_resultrows))
							(cons head (cons (runtime_heap_list_ast row_items) tail))))
					(cons head (map rest parallelize_resultrows)))
				(cons head (map rest parallelize_resultrows)))
			(if (resultrow_call_head? head)
				(match rest
					(cons (cons list_head kv_pairs) rr_rest)
					(begin
						(define vals (extract_assoc kv_pairs (lambda (k v) v)))
						(define complex_count (reduce vals (lambda (acc v) (+ acc (if (expr_is_parallelizable v) 1 0))) 0))
						(if (< complex_count 2)
							ast
							(begin
								(define keys (extract_assoc kv_pairs (lambda (k v) k)))
								(define thunks (map vals (lambda (v) (list (quote lambda) '() v))))
								(define pmap_call (list (symbol "parallel_map")
									(cons (symbol "list") thunks)
									(list (quote lambda) (list (symbol "__pf")) (list (symbol "__pf")))))
								(define reassembled (cons list_head
									(merge (mapIndex keys (lambda (i k) (list k (list (symbol "nth") (symbol "__pr") i)))))))
								(list (quote begin)
									(list (quote define) (symbol "__pr") pmap_call)
									(cons head (cons reassembled rr_rest))))))
					ast)
				(cons (cons quote_sym quoted_args) rr_rest)
				(if (is_quote_scope_sym quote_sym)
					(begin
						(define row_items (if (and (equal? (count quoted_args) 1) (list? (car quoted_args)))
							(car quoted_args)
							(if (> (count quoted_args) 1) quoted_args nil)))
						(if (nil? row_items)
							ast
							(cons head (cons (runtime_heap_list_ast row_items) rr_rest))))
					ast)
				(match (string head)
					"begin" (cons head (map rest parallelize_resultrows))
					"!begin" (cons head (map rest parallelize_resultrows))
					"if" (cons head (map rest parallelize_resultrows))
					"define" (cons head (map rest parallelize_resultrows))
					"time" (cons head (map rest parallelize_resultrows))
					ast)))
		ast)))

(define normalize_quoted_scalar_resultrow_calls (lambda (ast)
	(match ast
		(cons head rest)
		(if (is_quote_scope_sym head)
			(match rest
				(cons quoted_expr '())
				(match quoted_expr
					(cons lambda_head _)
					(if (or
						(equal? lambda_head (quote lambda))
						(equal? lambda_head (symbol lambda))
						(equal? lambda_head '(quote lambda)))
						(list head (normalize_quoted_scalar_resultrow_calls quoted_expr))
						ast)
					ast)
				ast)
			(if (scalar_resultrow_call_head? head)
				(match rest
					(cons (cons quote_sym quoted_args) tail)
					(if (is_quote_scope_sym quote_sym)
						(begin
							(define row_items (if (and (equal? (count quoted_args) 1) (list? (car quoted_args)))
								(car quoted_args)
								(if (> (count quoted_args) 1) quoted_args nil)))
							(if (nil? row_items)
								(cons head (map rest normalize_quoted_scalar_resultrow_calls))
								(cons head (cons (runtime_heap_list_ast row_items) tail))))
						(cons head (map rest normalize_quoted_scalar_resultrow_calls)))
					(cons head (map rest normalize_quoted_scalar_resultrow_calls)))
				(cons (normalize_quoted_scalar_resultrow_calls head)
					(if (resultrow_call_head? head)
						(match rest
							(cons (cons quote_sym quoted_args) tail)
							(if (is_quote_scope_sym quote_sym)
								(begin
									(define row_items (if (and (equal? (count quoted_args) 1) (list? (car quoted_args)))
										(car quoted_args)
										(if (> (count quoted_args) 1) quoted_args nil)))
									(if (nil? row_items)
										(map rest normalize_quoted_scalar_resultrow_calls)
										(cons (runtime_heap_list_ast row_items) tail)))
								(map rest normalize_quoted_scalar_resultrow_calls))
							(map rest normalize_quoted_scalar_resultrow_calls))
						(map rest normalize_quoted_scalar_resultrow_calls)))))
		ast)))

/* split_condition: selection pushdown for nested-loop join planning.
Splits an AND-condition into (now, later): predicates evaluatable with currently
bound tables vs predicates that must wait for inner tables to be scanned.
Enables index-based filtering in scan/scan_order by pushing predicates down. */
(define split_condition_outer_alias (lambda (symname)
	(match symname
		'(symbol inner) (split_condition_outer_alias inner)
		'(quote inner) (split_condition_outer_alias inner)
		(match (split (string symname) ".")
			'(alias_ _) alias_
			nil))))
(define split_condition_tables_contain_alias? (lambda (tables alias_)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(or found
			(match td
				'(a _ _ _ _) (equal?? a alias_)
				false)))
		false)))
(define split_condition_scalar_helper_alias? (lambda (alias_)
	(and
		(not (nil? alias_))
		(not (list? alias_))
		(begin
			(define alias-str (string alias_))
			(and
				(>= (strlen alias-str) 14)
				(equal? (substr alias-str 0 14) "domain_scalar_"))))))
(define split_condition (lambda (expr tables) (match expr
	'((symbol get_column) tblvar _ col _) /* a column */ (match tables
		'() '(expr true) /* last condition: compute now */
		(cons (cons (eval tblvar) _) _) '(true expr) /* col depends on tblvar */
		(cons _ tablesrest) (split_condition expr tablesrest) /* check next table in join plan */
		(error "invalid tables list")
	)
	'((quote get_column) tblvar _ col _) /* a column */ (match tables
		'() '(expr true) /* last condition: compute now */
		(cons (cons (eval tblvar) _) _) '(true expr) /* col depends on tblvar */
		(cons _ tablesrest) (split_condition expr tablesrest) /* check next table in join plan */
		(error "invalid tables list")
	)
	'((symbol aggregate) _ _ _) (if (equal? tables '()) '(expr true) '(true expr))
	'((quote aggregate) _ _ _) (if (equal? tables '()) '(expr true) '(true expr))
		'((symbol outer) symname)
			(begin
				(define outer-alias (split_condition_outer_alias symname))
			(if (split_condition_scalar_helper_alias? outer-alias)
				'(true true)
				(if (split_condition_tables_contain_alias?
					tables outer-alias)
					'(true expr)
					'(expr true))))
		'((quote outer) symname)
			(begin
				(define outer-alias (split_condition_outer_alias symname))
			(if (split_condition_scalar_helper_alias? outer-alias)
				'(true true)
				(if (split_condition_tables_contain_alias?
					tables outer-alias)
					'(true expr)
					'(expr true))))
		(cons (symbol outer) args)
			(begin
				(define symname (car args))
				(define outer-alias (split_condition_outer_alias symname))
			(if (split_condition_scalar_helper_alias? outer-alias)
				'(true true)
				(if (split_condition_tables_contain_alias?
					tables outer-alias)
					'(true expr)
					'(expr true))))
		(cons (symbol and) conditions) /* splittable and */ (split_condition_and conditions tables)
	/* Scope contract: runtime subplans and other opaque scope nodes carry their
	own alias domain. split_condition must not recurse into them, otherwise inner
	get_column refs from correlated scalar subqueries get misclassified as later
	join refs of the surrounding group stage and leak into keytable lowering. */
	(cons sym args) /* non-splittable function call */ (if (_is_opaque_scope_sym sym)
		'(expr true)
		(split_condition_combine sym args tables))
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

	(define split_condition_has_outer_scalar_helper? (lambda (expr)
		(match expr
			'((symbol outer) symname)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias symname))
			'((quote outer) symname)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias symname))
			(cons (symbol outer) args)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias (car args)))
			(cons head args)
			(or
				(split_condition_has_outer_scalar_helper? head)
				(reduce (coalesceNil args '()) (lambda (found arg)
					(or found
						(split_condition_has_outer_scalar_helper? arg)))
					false))
			false)))

	(define split_condition_outer_scalar_helper_ref? (lambda (expr)
		(match expr
			'((symbol get_column) alias_ _ _ _)
			(split_condition_scalar_helper_alias? alias_)
			'((quote get_column) alias_ _ _ _)
			(split_condition_scalar_helper_alias? alias_)
			'((symbol outer) symname)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias symname))
			'((quote outer) symname)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias symname))
			(cons (symbol outer) args)
			(split_condition_scalar_helper_alias?
				(split_condition_outer_alias (car args)))
			false)))

	(define strip_outer_scalar_helper_ref_terms (lambda (expr)
		(combine_and_terms
			(filter (flatten_and_terms (coalesceNil expr true))
				(lambda (part)
					(not (split_condition_outer_scalar_helper_ref? part)))))))

	(define split_condition_combine (lambda (sym args tables)
		(if (reduce (coalesceNil args '()) (lambda (found arg)
			(or found
				(split_condition_has_outer_scalar_helper? arg)))
			false)
			'(true true)
			(if
				(reduce args (lambda (other arg) (match (split_condition arg tables) '(_ true) other true)) false) /* if one of the args is later, everything is later */
				'(true (cons sym args))
				'((cons sym args) true)
		))))

	(define flatten_and_terms (lambda (expr) (match expr
	(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
		(merge (map parts flatten_and_terms))
		(if (or (nil? expr) (equal? expr true)) '() (list expr)))
	_ (if (or (nil? expr) (equal? expr true)) '() (list expr))
)))

(define combine_and_terms (lambda (parts) (begin
	(define _parts
		(reduce (filter parts (lambda (x) (and (not (nil? x)) (not (equal? x true)))))
			(lambda (acc part) (append_unique acc part))
			'()))
	(if (equal? _parts '()) true
		(if (equal? 1 (count _parts)) (car _parts)
			(cons (quote and) _parts)))
)))

(define equality_op_symbol? (lambda (op)
	(or
		(equal? op (quote equal?))
		(equal? op (quote equal??))
		(equal? op '(quote equal?))
		(equal? op '(quote equal??)))))

(define expr_has_runtime_outer_ref? (lambda (expr) (match expr
	'((symbol outer) _) true
	'((quote outer) _) true
	(cons sym args)
	(or
		(expr_has_runtime_outer_ref? sym)
		(reduce args (lambda (found arg)
			(or found (expr_has_runtime_outer_ref? arg)))
			false))
	false)))

(define planner_expr_head_is? (lambda (head sym)
	(or
		(equal? head sym)
		(equal? head (symbol sym))
		(equal? head (list (quote symbol) sym))
		(equal? head (list (quote quote) sym)))))

(define scalar_helper_value_ref? (lambda (expr tblvar)
	(match expr
		'((symbol get_column) alias_ _ col _) (and (equal?? alias_ tblvar) (equal? (string col) "value"))
		'((quote get_column) alias_ _ col _) (and (equal?? alias_ tblvar) (equal? (string col) "value"))
		false)))

(define scalar_helper_count_positive_pred? (lambda (expr tblvar)
	(match expr
		'(op lhs rhs)
		(if (and (planner_expr_head_is? op (quote >)) (equal? rhs 0))
			(match lhs
				'(cop inner default)
				(and
					(or
						(planner_expr_head_is? cop (quote coalesce))
						(planner_expr_head_is? cop (quote coalesceNil)))
					(equal? default 0)
					(scalar_helper_value_ref? inner tblvar))
				false)
			false)
		false)))

(define scalar_helper_count_positive_pred_any? (lambda (expr)
	(or
		(reduce (extract_tblvars expr) (lambda (found tblvar)
			(or found
				(and
					(split_condition_scalar_helper_alias? tblvar)
					(scalar_helper_count_positive_pred? expr tblvar))))
			false)
		(match expr
			(cons head args)
			(or
				(scalar_helper_count_positive_pred_any? head)
				(reduce (coalesceNil args '()) (lambda (found arg)
					(or found
						(scalar_helper_count_positive_pred_any? arg)))
					false))
			false))))

(define scalar_helper_zero_literal? (lambda (value)
	(or (equal? value 0) (equal? value 0.0))))

(define scalar_helper_zero_coalesced_value? (lambda (expr tblvar)
	(match expr
		'(op inner default)
		(or
			(and
				(or
					(planner_expr_head_is? op (quote coalesce))
					(planner_expr_head_is? op (quote coalesceNil)))
				(scalar_helper_zero_literal? default)
				(has? (extract_tblvars inner) tblvar))
			(scalar_helper_zero_coalesced_value? inner tblvar))
		(or
			(scalar_helper_value_ref? expr tblvar)
			(and
				(not (equal? (extract_aggregates expr) '()))
				(has? (extract_tblvars expr) tblvar))))))

(define scalar_helper_count_zero_pred? (lambda (expr tblvar)
	(match expr
		'(op lhs rhs)
		(and
			(or
				(planner_expr_head_is? op (quote equal?))
				(planner_expr_head_is? op (quote equal??))
				(planner_expr_head_is? op (quote =)))
			(or
				(and
					(scalar_helper_zero_literal? rhs)
					(scalar_helper_zero_coalesced_value? lhs tblvar))
				(and
					(scalar_helper_zero_literal? lhs)
					(scalar_helper_zero_coalesced_value? rhs tblvar))))
		false)))

(define scalar_helper_count_anti_pred? (lambda (expr tblvar) (begin
	(define zero_literal? (lambda (value)
		(or (equal? value 0) (equal? value 0.0))))
	(define zero_coalesced_helper? (lambda (value)
		(match value
			'(op inner default)
			(and
				(or
					(planner_expr_head_is? op (quote coalesce))
					(planner_expr_head_is? op (quote coalesceNil)))
				(zero_literal? default)
				(has? (extract_tblvars inner) tblvar))
			false)))
	(define zero_pred? (lambda (value)
		(match value
			'(op lhs rhs)
			(and
				(or
					(planner_expr_head_is? op (quote equal?))
					(planner_expr_head_is? op (quote equal??))
					(planner_expr_head_is? op (quote =)))
				(or
					(and (zero_literal? rhs) (has? (extract_tblvars lhs) tblvar))
					(and (zero_literal? lhs) (has? (extract_tblvars rhs) tblvar))))
			false)))
	(or
		(zero_pred? expr)
		(match expr
			'(head inner)
			(and
				(planner_expr_head_is? head (quote not))
				(scalar_helper_count_positive_pred? inner tblvar))
			false)))))

(define outer_scan_join_filter_part? (lambda (tblvar materialized_source part) (begin
	(or
		(match part
			'(op _ _) (equality_op_symbol? op)
			false)
		(and
			(not (expr_has_runtime_outer_ref? part))
			(equal? (has_only_tblvar_refs part tblvar) true))))))

/* split_scan_condition: keep joinexpr separate from global WHERE.
Returns (now later) for one scan level:
- INNER scans: joinexpr parts evaluatable now are pushed into now; later parts stay deferred.
- OUTER scans: only joinexpr stays on the scan (ON semantics); global WHERE terms are deferred. */
(define split_scan_condition (lambda (isOuter joinexpr scan_condition rest_tables) (begin
	(match (split_condition (coalesceNil scan_condition true) rest_tables) '(raw_now_condition raw_later_condition)
		(match (split_condition (coalesceNil joinexpr true) rest_tables) '(join_now_condition join_later_condition)
			(if (not isOuter)
				(list
					(strip_outer_scalar_helper_ref_terms
						(combine_and_terms (merge (flatten_and_terms raw_now_condition) (flatten_and_terms join_now_condition))))
					(combine_and_terms (merge (flatten_and_terms raw_later_condition) (flatten_and_terms join_later_condition))))
				(begin
					(define join_now_parts (flatten_and_terms join_now_condition))
					(define delayed_join_now_parts
						(filter join_now_parts
							scalar_helper_count_positive_pred_any?))
					(define immediate_join_now_parts
						(filter join_now_parts (lambda (part)
							(not (scalar_helper_count_positive_pred_any? part)))))
					(list
						(strip_outer_scalar_helper_ref_terms
							(combine_and_terms immediate_join_now_parts))
						(combine_and_terms (merge
							(flatten_and_terms raw_now_condition)
							(flatten_and_terms raw_later_condition)
							delayed_join_now_parts
							(flatten_and_terms join_later_condition)))))))))))

(define scalar_helper_outer_local_scan_condition (lambda (enabled tblvar base_tbl scan_condition rest_tables) (if enabled
	(match (split_condition (coalesceNil scan_condition true) rest_tables) '(raw_now_condition _raw_later_condition)
		(combine_and_terms
			(filter
				(flatten_and_terms (coalesceNil raw_now_condition true))
				(lambda (part)
					(not (scalar_helper_count_anti_pred? part tblvar))))))
	true)))

(define scalar_helper_outer_join_terms (lambda (tblvar condition)
	condition))

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
	(cons sym args) (reduce args (lambda (acc arg)
		(merge_unique acc (collect_all_column_refs arg)))
		'())
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
		(reduce args (lambda (acc arg)
			(merge_unique acc (extract_outer_columns_for_tblvar tblvar arg)))
			'())
	)
	'()
)))

/* columns of tblvar that are needed only because later table-local joinexprs reference them.
These columns must still be mapped by the current scan so nested join filters can see them. */
(define extract_later_joinexpr_columns_for_tblvar (lambda (tblvar tables)
	(merge_unique (map tables (lambda (td) (match td
		'(_ _ _ _ je) (if (nil? je) '() (merge_unique (list
			(extract_columns_for_tblvar tblvar je)
			(extract_outer_columns_for_tblvar tblvar je))))
		'()))))
))

/* Some scalar helper stages carry outer correlation keys only in stage metadata
(outer-sources). The enclosing scan still has to map those columns so nested
helper filters can close over them at runtime. */
(define extract_stage_outer_source_cols_for_tblvar (lambda (tblvar stages)
	(merge_unique (map (coalesceNil stages '()) (lambda (stage)
		(merge_unique (map (coalesceNil (stage_outer_sources stage) '()) (lambda (src)
			(match src
				'(outer_tv outer_col _inner_expr) (if (stage_outer_source_expr_tuple? src)
					(extract_columns_for_tblvar tblvar (nth src 1))
					(if (equal?? outer_tv tblvar) (list outer_col) '()))
				_ '()))))))))
))

(define collect_scan_base_cols (lambda (tblvar scan_condition visible_fields tables partition_stages extra_cols)
	(merge_unique
		(list
			(merge_unique
				(cons
					(extract_columns_for_tblvar tblvar scan_condition)
					(extract_assoc visible_fields (lambda (k v) (extract_columns_for_tblvar tblvar v)))
				)
			)
			(merge_unique
				(cons
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(extract_assoc visible_fields (lambda (k v) (extract_outer_columns_for_tblvar tblvar v)))
				)
			)
			(extract_later_joinexpr_columns_for_tblvar tblvar tables)
			(extract_stage_outer_source_cols_for_tblvar tblvar partition_stages)
			extra_cols
		)
	)
))

(define extend_scan_cols_for_later_condition (lambda (tblvar cols effective_later_condition)
	(merge_unique (list
		cols
		(extract_columns_for_tblvar tblvar effective_later_condition)
		(extract_outer_columns_for_tblvar tblvar effective_later_condition)
	))
))

(define find_partition_stage_for_alias (lambda (stages tblvar)
	(reduce (coalesceNil stages '()) (lambda (found stage)
		(if (nil? found)
			(if (has? (coalesceNil (stage_partition_aliases stage) '()) tblvar) stage nil)
			found))
		nil)
))

(define _extract_scan_order_terms_for_tblvar (lambda (order_items tblvar pick_col)
	(merge (map (coalesceNil order_items '()) (lambda (order_item) (match order_item '(col dir) (match col
		'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list (if pick_col col dir)) '())
		'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list (if pick_col col dir)) '())
		_ '()
)))))))

(define extract_scan_order_cols_for_tblvar (lambda (order_items tblvar)
	(_extract_scan_order_terms_for_tblvar order_items tblvar true)
))

(define extract_scan_order_dirs_for_tblvar (lambda (order_items tblvar)
	(_extract_scan_order_terms_for_tblvar order_items tblvar false)
))

/* symbols that canonicalize_columns must NOT recurse into — they have their own scope */
(define _is_opaque_scope_sym (lambda (sym) (match sym
	/* inner_select markers are NOT opaque — they are logical markers that
	must be transparent for outer-ref detection during Neumann decorrelation.
	Only physical runtime code (scan, !begin, etc.) is opaque. */
	(symbol !begin) true '(quote !begin) true '!begin true
	(symbol scan) true '(quote scan) true 'scan true
	(symbol scan_order) true '(quote scan_order) true 'scan_order true
	(symbol newpromise) true '(quote newpromise) true 'newpromise true
	(symbol newsession) true '(quote newsession) true 'newsession true
	_ false)))
(define is_quote_scope_sym (lambda (sym) (match sym
	(symbol quote) true '(quote quote) true 'quote true
	(symbol quasiquote) true '(quote quasiquote) true 'quasiquote true
	_ false)))
(define expr_has_opaque_scope (lambda (expr) (match expr
	(cons sym args) (or
		(_is_opaque_scope_sym sym)
		(reduce args (lambda (found arg) (or found (expr_has_opaque_scope arg))) false))
	false)))
/* extract runtime-sensitive subexpressions that do not depend on table rows.
Those terms must affect cache identity (temp column names) even if the
relational key/domain stays unchanged. */
(define runtime_cache_normalize_term (lambda (term)
	(if (and (list? term) (equal? (count term) 1) (expr_uses_session_state (car term)))
		(car (extract_session_lookup_terms (car term)))
		term)))
(define runtime_cache_unique_terms (lambda (terms)
	(reduce (coalesceNil terms '()) (lambda (acc raw_term)
		(define term (runtime_cache_normalize_term raw_term))
		(if (or (nil? term) (equal? term '())
			(reduce acc (lambda (found existing) (or found (equal? existing term))) false))
			acc
			(merge acc (list term))))
		'())))
(define runtime_cache_merge_term_lists (lambda (term_lists)
	(runtime_cache_unique_terms (merge term_lists))))
(define runtime_cache_session_key (lambda (key)
	(if (string? key) key (string key))))
(define runtime_cache_normalize_session_term (lambda (expr) (match expr
	(cons (symbol session) args) (match args
		(cons key rest) (cons (quote session) (cons (runtime_cache_session_key key) rest))
		expr)
	(cons '(quote session) args) (match args
		(cons key rest) (cons (quote session) (cons (runtime_cache_session_key key) rest))
		expr)
	(cons (cons (symbol context) '("session")) args) (match args
		(cons key rest) (cons (list (quote context) "session") (cons (runtime_cache_session_key key) rest))
		expr)
	(cons (cons '(quote context) '("session")) args) (match args
		(cons key rest) (cons (list (quote context) "session") (cons (runtime_cache_session_key key) rest))
		expr)
	expr)))
(define extract_runtime_cache_terms (lambda (expr) (match expr
	(cons only '()) (if (expr_uses_session_state only)
		(extract_runtime_cache_terms only)
		(if (and (expr_uses_session_state expr) (equal? (extract_tblvars expr) '()))
			(list expr)
			'()))
	(cons (symbol session) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms)))
	(cons '(quote session) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms)))
	(cons (cons (symbol context) '("session")) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms)))
	(cons (cons '(quote context) '("session")) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms)))
	(cons sym args) (if (_is_opaque_scope_sym sym)
		(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms))
		(if (and (expr_uses_session_state expr) (equal? (extract_tblvars expr) '()))
			(list expr)
			(runtime_cache_merge_term_lists (map args extract_runtime_cache_terms))))
	'()
)))
(define extract_session_lookup_terms (lambda (expr) (match expr
	(cons only '()) (if (expr_uses_session_state only)
		(extract_session_lookup_terms only)
		'())
	(cons (symbol session) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_session_lookup_terms)))
	(cons '(quote session) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_session_lookup_terms)))
	(cons (cons (symbol context) '("session")) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_session_lookup_terms)))
	(cons (cons '(quote context) '("session")) args) (if (equal? (count args) 1)
		(list (runtime_cache_normalize_session_term expr))
		(runtime_cache_merge_term_lists (map args extract_session_lookup_terms)))
	(cons sym args) (runtime_cache_merge_term_lists (map args extract_session_lookup_terms))
	'()
)))
/* runtime_cache_suffix_from_exprs: derive a stable, value-sensitive appendix for
temp column names from session-/context-dependent terms. The suffix is computed
at plan-build time, so repeated queries with the same runtime values reuse the
same cache column while different session values get separate temp columns. */
(define planner_eval_runtime_term (lambda (expr)
	(define _bind_context_session (lambda (node) (match node
		(cons (symbol session) args) (match args
			(cons key rest) (cons '(context "session") (cons (runtime_cache_session_key key) rest))
			node)
		(cons '(quote session) args) (match args
			(cons key rest) (cons '(context "session") (cons (runtime_cache_session_key key) rest))
			node)
		(cons sym args) (cons (_bind_context_session sym) (map args _bind_context_session))
		node)))
	(eval (_bind_context_session expr))
))
(define runtime_cache_terms_from_exprs (lambda (exprs)
	(runtime_cache_unique_terms (merge
		(map exprs extract_runtime_cache_terms)
		(map exprs extract_session_lookup_terms)))
))
(define runtime_cache_suffix_from_terms (lambda (terms) (begin
	(if (equal? terms '())
		""
		(concat "|rt:"
			(serialize_canonical_expr
				(canonicalize_expr
					(map terms (lambda (term) (list term (planner_eval_runtime_term term))))
					'(list)))))
)))
(define runtime_cache_suffix_from_exprs (lambda (exprs)
	(runtime_cache_suffix_from_terms (runtime_cache_terms_from_exprs exprs))
))
(define aggregate_count_descriptor '(1 + 0))
(define aggregate_cache_condition_suffix (lambda (expr_name condition_expr runtime_suffix)
	(fnv_hash (concat (expr_name (planner_name_clear_case_flags condition_expr)) (coalesceNil runtime_suffix "")))
))
(define aggregate_cache_col_name_with_suffix (lambda (expr_name ag suffix)
	(if (equal? ag aggregate_count_descriptor)
		(concat "COUNT(*)|" suffix)
		(concat (expr_name ag) "|" suffix))
))
(define make_aggregate_cache_col_name (lambda (expr_name condition_expr runtime_suffix) (begin
	(define suffix (aggregate_cache_condition_suffix expr_name condition_expr runtime_suffix))
	(lambda (ag) (aggregate_cache_col_name_with_suffix expr_name ag suffix))
)))
(define find_materialized_field_by_name (lambda (materialized_cols target_col)
	(reduce materialized_cols (lambda (found coldef)
		(if (not (nil? found))
			found
			(if (equal? (coldef "Field") target_col)
				target_col
				nil)))
		nil)
))
(define assoc_key_less (lambda (a b)
	(if (and (list? a) (list? b))
		(begin
			(define n (min (count a) (count b)))
			(define idx (reduce (produceN n) (lambda (found i)
				(if (nil? found)
					(if (equal? (nth a i) (nth b i)) nil i)
					found))
				nil))
			(if (nil? idx)
				(< (count a) (count b))
				(< (nth a idx) (nth b idx))))
		(< a b))))
(define assoc_keys_as_dataset_rows (lambda (dict width)
	(map (sort (extract_assoc dict (lambda (k v) k)) assoc_key_less)
		(lambda (k)
			(if (list? k)
				k
				(if (<= width 1)
					(list k)
					(map (produceN width) (lambda (_) nil))))
))))

(define runtime_list_ast (lambda (items)
	(match items
		'() (list (quote list))
		(cons head tail) (list (quote cons) head (runtime_list_ast tail)))))
(define runtime_heap_list_ast (lambda (items)
	(cons (quote heap_list) items)))
(define assoc_items_as_dataset_rows (lambda (dict width)
	(map (sort (extract_assoc dict (lambda (k v) k)) assoc_key_less)
		(lambda (k)
			(merge
				(if (list? k)
					k
					(if (<= width 1)
						(list k)
						(map (produceN width) (lambda (_) nil))))
				(list (get_assoc dict k))))
)))
/* Column-resolution contract:
- parser-level get_column markers may still carry ti/ci flags inside untangle_query
- they must be resolved against schema metadata exactly once before the logical IR
crosses into build_queryplan
- later planner stages operate strictly case-sensitively on canonical aliases and
field names and must not re-run schema repair
resolve_schema_column_ref_scoped is the shared lookup primitive for that boundary:
it canonicalizes alias/field casing from schema metadata and keeps the preferred
search order for unqualified refs (main tables before helper/unnested aliases). */
(define main_scope_alias? (lambda (alias)
	(begin
		(define s (string alias))
		(and (equal? (replace s "\0" "") s)
			(not (and (>= (strlen s) 14) (equal? (substr s 0 14) "domain_scalar_")))))))
/* Shared schema-resolution contract:
- all alias/column lookups flow through these helpers
- callers choose whether they want the first visible match or require uniqueness
- the resolver itself is the only place that may interpret alias variants or
schema-driven column casing inside queryplan.scm */
(define schema_alias_matches (lambda (query_alias schema_alias ti)
	((if ti equal?? equal?) query_alias schema_alias)
))
(define resolve_schema_alias_scoped (lambda (schemas alias_ ti)
	(if (nil? alias_) nil
		(reduce_assoc schemas (lambda (found alias cols)
			(if (and (nil? found) (schema_alias_matches alias_ alias ti))
				alias
				found))
			nil)
	)
))
(define schema_column_def (lambda (cols col ci)
	(reduce cols (lambda (found coldef)
		(if (not (nil? found))
			found
			(if ((if ci equal?? equal?) (coldef "Field") col) coldef nil)))
		nil)
))
(define schema_has_column? (lambda (cols col ci)
	(not (nil? (schema_column_def cols col ci)))
))
(define direct_schema_projection_source_col (lambda (expr tblvar)
	(match expr
		'((symbol get_column) alias_ _ source_col _) (if (equal?? alias_ tblvar) source_col nil)
		'((quote get_column) alias_ _ source_col _) (if (equal?? alias_ tblvar) source_col nil)
		nil)))
(define scan_physical_col_for_tblvar (lambda (schemas tblvar col)
	(begin
		(define projected_source_col_from_cols (lambda (cols)
			(begin
				(define coldef (schema_column_def (coalesceNil cols '()) col false))
				(if (nil? coldef)
					nil
					(direct_schema_projection_source_col (coalesceNil (coldef "Expr") nil) tblvar)))))
		(define source_col_direct (projected_source_col_from_cols (coalesceNil (schemas tblvar) '())))
		(define source_col_any
			(if (not (nil? source_col_direct))
				source_col_direct
				(reduce_assoc schemas (lambda (found _alias cols)
					(if (not (nil? found))
						found
						(projected_source_col_from_cols cols)))
					nil)))
		(define source_col (if (nil? source_col_any)
			nil
			source_col_any))
		(coalesce source_col col))))
(define scan_physical_cols_for_tblvar (lambda (schemas tblvar cols)
	(map cols (lambda (col) (scan_physical_col_for_tblvar schemas tblvar col)))))
(define canonical_schema_column_name (lambda (cols col ci)
	(begin
		(define coldef (if (nil? cols) nil (schema_column_def cols col ci)))
		(coalesce (if (nil? coldef) nil (coldef "Field")) col))
))
(define collect_schema_column_matches_scoped (lambda (local_schemas visible_schemas alias_ ti col ci) (begin
	(define collect_matches (lambda (schemas alias_pred)
		(reduce_assoc schemas (lambda (acc alias cols)
			(if (and (alias_pred alias) (schema_has_column? cols col ci))
				(merge acc (list (list alias (canonical_schema_column_name cols col ci))))
				acc))
			'())))
	(if (nil? alias_)
		(begin
			(define _main (collect_matches local_schemas main_scope_alias?))
			(if (equal? _main '())
				(collect_matches local_schemas (lambda (alias) true))
				_main))
		(collect_matches visible_schemas (lambda (alias) (schema_alias_matches alias_ alias ti))))
)))
(define first_schema_column_match_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(match (collect_schema_column_matches_scoped local_schemas visible_schemas alias_ ti col ci)
		(cons head _) head
		nil
)))
(define unique_schema_column_match_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(match (collect_schema_column_matches_scoped local_schemas visible_schemas alias_ ti col ci)
		(cons head '()) head
		nil
)))
(define resolve_schema_column_ref_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (first_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved) nil
			(list (nth resolved 0) (nth resolved 1)))
	)
))
(define resolve_unique_schema_column_ref_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (unique_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved) nil
			(list (nth resolved 0) (nth resolved 1)))
	)
))
(define resolve_schema_column_expr_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (first_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved)
			nil
			(list (quote get_column) (nth resolved 0) false (nth resolved 1) false))
	)
))
/* canonicalize_columns_scoped resolves ti/ci flags to canonical casing.
local_schemas are the aliases visible in the current scope, while visible_schemas
also contains outer aliases needed for qualified outer refs like src.ID.
Unqualified refs must only match local aliases so recursive untangling keeps
free get_columns free instead of accidentally binding them to an outer table. */
(define canonicalize_columns_scoped (lambda (expr local_schemas visible_schemas) (match expr
	'((symbol get_column) alias_ ti col ci) (if (or ti ci)
		(begin
			(define resolved (resolve_schema_column_expr_scoped local_schemas visible_schemas alias_ ti col ci))
			(if (nil? resolved)
				expr /* leave unresolved — replace_find_column will handle or error */
				resolved))
		expr /* ti=false ci=false: already canonical */
	)
	'((quote get_column) alias_ ti col ci) (if (or ti ci)
		(begin
			(define resolved (resolve_schema_column_expr_scoped local_schemas visible_schemas alias_ ti col ci))
			(if (nil? resolved)
				expr /* leave unresolved — replace_find_column will handle or error */
				resolved))
		expr /* ti=false ci=false: already canonical */
	)
	/* do not recurse into opaque scope nodes — inner_select, runtime code */
	(cons sym args) (if (_is_opaque_scope_sym sym) expr
		(cons (canonicalize_columns_scoped sym local_schemas visible_schemas)
			(map args (lambda (a) (canonicalize_columns_scoped a local_schemas visible_schemas)))))
	expr
)))
/* canonicalize_columns keeps the old single-schema API for callers that do not
cross a scope boundary. */
(define canonicalize_columns (lambda (expr all_schemas)
	(canonicalize_columns_scoped expr all_schemas all_schemas)
))
/* finalize_logical_expr is the only normalization gate from untangle_query into
the downstream planner.
Order matters:
1. canonicalize_columns: resolve parser-level ti/ci flags against schemas
2. rewrite_expr: lower visible derived-table aliases to their logical source expr
3. canonicalize_columns again: any get_column introduced by rewrite_expr must
also leave untangle_query in exact schema casing
After this helper, later planner stages must only see exact/case-sensitive
get_column markers and may no longer run schema-based repair heuristics. */
(define finalize_logical_expr_scoped (lambda (expr local_schemas visible_schemas rewrite_expr enforce_contract) (begin
	(define finalized
		(canonicalize_columns_scoped
			(rewrite_expr (canonicalize_columns_scoped expr local_schemas visible_schemas))
			local_schemas
			visible_schemas))
	(if enforce_contract
		(require_canonical_logical_expr "untangle_query output" finalized)
		finalized))
))
(define finalize_logical_expr (lambda (expr all_schemas rewrite_expr enforce_contract)
	(finalize_logical_expr_scoped expr all_schemas all_schemas rewrite_expr enforce_contract)
))
(define finalize_logical_stage_scoped (lambda (stage local_schemas visible_schemas rewrite_expr enforce_contract) (begin
	(define fin (lambda (expr) (finalize_logical_expr_scoped expr local_schemas visible_schemas rewrite_expr enforce_contract)))
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_post_group_condition_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(define spa (stage_partition_aliases stage))
	(define sc (stage_condition stage))
	(define sonce (stage_once_limit stage))
	(if (stage_is_dedup stage)
		(stage_rebuild_with_meta
			stage
			(make_dedup_stage (map sg fin) spa)
			fin
			(lambda (a) a))
		(if (and (not (nil? spa)) (or (nil? sg) (equal? sg '())))
			(stage_rebuild_with_meta
				stage
				(make_stage
					'()
					nil
					(map so (lambda (o) (match o '(c d) (list (fin c) d))))
					(coalesceNil (stage_limit_partition_cols stage) 0)
					sl
					soff
					false
					spa
					(stage_init_code stage)
					(if (nil? sc) nil (fin sc))
					sonce)
				fin
				(lambda (a) a))
			(stage_rebuild_with_meta
				stage
				(if (nil? sc)
					(make_group_stage
						(map sg fin)
						(fin sh)
						(map so (lambda (o) (match o '(c d) (list (fin c) d))))
						sl
						soff
						spa
						(stage_init_code stage))
					(make_group_stage_with_condition
						(map sg fin)
						(fin sh)
						(map so (lambda (o) (match o '(c d) (list (fin c) d))))
						sl
						soff
						spa
						(stage_init_code stage)
						(fin sc)))
				fin
				(lambda (a) a))))))
)
(define finalize_logical_stage (lambda (stage all_schemas rewrite_expr enforce_contract)
	(finalize_logical_stage_scoped stage all_schemas all_schemas rewrite_expr enforce_contract)
))
/* canonicalize all get_column markers in a group stage */
(define canonicalize_stage (lambda (stage all_schemas) (begin
	(define canon (lambda (expr) (canonicalize_columns expr all_schemas)))
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_having_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(define spa (stage_partition_aliases stage))
	(define sc (stage_condition stage))
	(define sonce (stage_once_limit stage))
	(if (stage_is_dedup stage)
		(stage_rebuild_with_meta stage (make_dedup_stage (map sg canon) spa) canon (lambda (a) a))
		(if (and (not (nil? spa)) (or (nil? sg) (equal? sg '())))
			/* partition stage (aliases but no group): preserve partition-aliases and limit-partition-cols */
			(stage_rebuild_with_meta stage
				(make_stage
					'()
					nil
					(map so (lambda (o) (match o '(c d) (list (canon c) d))))
					(coalesceNil (stage_limit_partition_cols stage) 0)
					sl
					soff
					false
					spa
					(stage_init_code stage)
					(if (nil? sc) nil (canon sc))
					sonce)
				canon
				(lambda (a) a))
			/* group stage (possibly scoped with aliases) */
			(stage_rebuild_with_meta stage
				(if (nil? sc)
					(make_group_stage
						(map sg canon)
						(canon sh)
						(map so (lambda (o) (match o '(c d) (list (canon c) d))))
						sl soff spa (stage_init_code stage))
					(make_group_stage_with_condition
						(map sg canon)
						(canon sh)
						(map so (lambda (o) (match o '(c d) (list (canon c) d))))
						sl soff spa (stage_init_code stage)
						(canon sc)))
				canon
				(lambda (a) a)))))
))

(import "sql-metadata.scm")

/* group stage constructors and accessors - shared between untangle_query and build_queryplan
All stages have partition-aliases (scope): nil = global (all tables), list = scoped to those tables.
All stages have init: nil = no init code, or code to run before the scan. */
(define normalize_stage_aliases (lambda (aliases)
	(if (nil? aliases)
		nil
		(if (list? aliases)
			aliases
			(list aliases)))))
(define make_stage (lambda (group having order limit_partition_cols limit offset dedup aliases init cond once_limit)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) having)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) (coalesce limit_partition_cols 0))
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) (coalesce dedup false))
		(list (quote partition-aliases) (normalize_stage_aliases aliases))
		(list (quote init) init)
		(list (quote stage-condition) cond)
		(list (quote once-limit) once_limit)
	)
))
(define make_group_stage (lambda (group having order limit offset aliases init)
	(make_stage group having order 0 limit offset false aliases init nil nil)
))
/* make_group_stage_with_condition: like make_group_stage but carries the inner
subquery's WHERE condition scoped to this stage's tables. build_queryplan merges
it into the local condition when the stage is processed, preventing cross-stage
condition leakage. Uses canonical column names for keytable cache reuse. */
(define make_group_stage_with_condition (lambda (group having order limit offset aliases init cond)
	(make_stage group having order 0 limit offset false aliases init cond nil)
))
(define make_once_per_partition_stage (lambda (group having order once_limit aliases init cond)
	(make_stage group having order 0 nil nil false aliases init cond once_limit)
))
(define make_partition_stage (lambda (aliases order partition_cols limit offset init)
	(make_stage '() nil order partition_cols limit offset false aliases init nil nil)
))
(define make_dedup_stage (lambda (group aliases)
	(make_stage group nil '() 0 nil nil true aliases nil nil nil)
))
(define stage_get (lambda (stage key default)
	(reduce stage (lambda (acc item)
		(if (nil? acc)
			(if (and (list? item) (> (count item) 0) (equal? (car item) key))
				(if (> (count item) 1) (nth item 1) default)
				nil)
			acc)
	) nil)
))
(define stage_get_rest (lambda (stage key default)
	(reduce stage (lambda (acc item)
		(if (nil? acc)
			(if (and (list? item) (> (count item) 0) (equal? (car item) key))
				(cdr item)
				nil)
			acc)
	) default)
))
(define stage_without_key (lambda (stage key)
	(filter stage (lambda (item)
		(not (and (list? item) (> (count item) 0) (equal? (car item) key)))))
))
(define stage_set (lambda (stage key value)
	(if (nil? value)
		(stage_without_key stage key)
		(cons (list key value) (stage_without_key stage key)))
))
(define stage_group_cols (lambda (stage)
	(coalesceNil (stage_get_rest stage (quote group-cols) nil) nil)))
(define stage_having_expr (lambda (stage)
	(stage_get stage (quote having) nil)))
/* Compatibility alias: older unnesting logic still refers to the logical
post-group predicate under this name. On current master it is the HAVING expr. */
(define stage_post_group_condition_expr stage_having_expr)
(define stage_order_list (lambda (stage)
	(coalesceNil (stage_get stage (quote order) '()) '())))
(define stage_limit_val (lambda (stage)
	(stage_get stage (quote limit) nil)))
(define stage_offset_val (lambda (stage)
	(stage_get stage (quote offset) nil)))
(define stage_limit_partition_cols (lambda (stage)
	(coalesceNil (stage_get stage (quote limit-partition-cols) 0) 0)))
(define stage_partition_aliases (lambda (stage)
	(define raw_aliases (stage_get stage (quote partition-aliases) nil))
	(if (nil? raw_aliases) nil (normalize_stage_aliases raw_aliases))))
(define stage_init_code (lambda (stage)
	(stage_get stage (quote init) nil)))
(define stage_condition (lambda (stage)
	(stage_get stage (quote stage-condition) nil)))
(define stage_once_limit (lambda (stage)
	(stage_get stage (quote once-limit) nil)))
(define stage_cache_policy (lambda (stage)
	(stage_get stage (quote cache-policy) nil)))
(define stage_cache_query (lambda (stage)
	(stage_get stage (quote cache-query) nil)))
(define stage_is_dedup (lambda (stage)
	(equal? (stage_get stage (quote dedup) false) true)))
(define stage_kind (lambda (stage) (begin
	(define sk_aliases (stage_partition_aliases stage))
	(define sk_group (coalesceNil (stage_group_cols stage) '()))
	(if (stage_is_dedup stage) (quote dedup)
		(if (not (nil? (stage_once_limit stage))) (quote once-per-partition)
			(if (and (not (nil? sk_aliases)) (equal? sk_group '())) (quote partition)
				(if (not (nil? sk_aliases)) (quote scoped-group)
					(if (nil? sk_aliases) (quote global-group)
						nil)))))
)))
(define stage_is_scoped? (lambda (stage)
	(not (nil? (stage_partition_aliases stage)))))
(define stage_with_cache_policy (lambda (stage policy)
	(stage_set stage (quote cache-policy) policy)
))
(define stage_with_cache_query (lambda (stage query)
	(stage_set stage (quote cache-query) query)
))
/* stage_outer_sources: list of correlation tuples carried by scalar/partition
stages so the post-reorder anti-pass fixup can null-extend outer rows whose
correlation key has no match in the inner helper (per FAQ point 34 /
once-limit-rework.md). Entries are either direct
(outer_tblvar outer_colname inner_expr) or expression-valued
(expr outer_expr inner_expr) for correlations like `inner = outer_col - 90`.
Present only on stages where Path B/C extracted us_domain_cols; nil elsewhere.
Design: assoc-style add-on; make_stage signature stays stable. Rebuilders must
preserve via stage_preserve_cache_meta. No consumer reads this field yet —
inject_anti_passes (next PR) will annotate stages when join_reorder places the
helper alias above its correlation source, and build_queryplan will emit the
companion anti-pass scan. */
(define normalize_stage_outer_sources_value (lambda (sources)
	(if (stage_outer_source_expr_tuple? sources)
		(list sources)
		(if (and (list? sources)
			(equal? (count sources) 3)
			(not (list? (car sources)))
			(not (list? (cadr sources))))
			(list sources)
			sources))))
(define stage_outer_source_expr_tuple? (lambda (src)
	(and
		(list? src)
		(equal? (count src) 3)
		(equal? (nth src 0) (quote expr)))))
(define stage_outer_sources (lambda (stage)
	(normalize_stage_outer_sources_value (stage_get stage (quote outer-sources) nil))))
(define stage_with_outer_sources (lambda (stage sources)
	(if (or (nil? sources) (equal? sources '()))
		(stage_set stage (quote outer-sources) nil)
		(stage_set stage (quote outer-sources) sources)
)))
(define stage_preserve_cache_meta (lambda (old_stage new_stage)
	(stage_with_outer_sources
		(stage_with_cache_query
			(stage_with_cache_policy new_stage (stage_cache_policy old_stage))
			(stage_cache_query old_stage))
		(stage_outer_sources old_stage)
	)
))
(define stage_rewrite_outer_sources (lambda (stage rewrite_expr rewrite_alias) (begin
	(define _sos_sources (coalesceNil (stage_outer_sources stage) '()))
	(if (equal? _sos_sources '())
		stage
		(stage_with_outer_sources
			stage
			(map _sos_sources (lambda (src) (match src
				'(outer_tv outer_col inner_expr)
				(if (stage_outer_source_expr_tuple? src)
					(list (quote expr)
						(rewrite_expr (nth src 1))
						(rewrite_expr (nth src 2)))
					(list
						(coalesceNil (rewrite_alias outer_tv) outer_tv)
						outer_col
						(rewrite_expr inner_expr)))
				_ src))))))
))
(define stage_rebuild_with_meta (lambda (old_stage new_stage rewrite_expr rewrite_alias)
	(stage_rewrite_outer_sources
		(stage_preserve_cache_meta old_stage new_stage)
		rewrite_expr
		rewrite_alias)
))
/* Extract anti-pass correlation metadata from correlated equality pairs.
Each domain pair is (inner_expr outer_expr); only direct outer get_column refs
become outer-sources entries because later anti-pass injection needs a stable
outer table alias plus column name. Session/runtime bindings stay out of this
carrier until session domains are modeled explicitly. */
(define domain_outer_sources_from_correlation_cols (lambda (domain_cols rewrite_inner_expr)
	(filter (map domain_cols (lambda (dc) (match (nth dc 1)
		'((symbol get_column) outer_tv _ outer_col _) (list outer_tv outer_col (rewrite_inner_expr (nth dc 0)))
		'((quote get_column) outer_tv _ outer_col _) (list outer_tv outer_col (rewrite_inner_expr (nth dc 0)))
		outer_expr (if (equal? (extract_tblvars outer_expr) '())
			nil
			(list (quote expr) outer_expr (rewrite_inner_expr (nth dc 0)))))))
		(lambda (src) (not (nil? src))))
))
(define unnest_expr_outer_refs (lambda (expr inner_aliases) (match expr
	(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
		(match args
			(cons sym_arg '()) (list (string sym_arg))
			'())
		(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
			(match args
				'(alias_ _ col _) (if (and (not (nil? alias_))
					(not (reduce inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
					(list (concat alias_ "." col))
					'())
				'())
			(if (_is_opaque_scope_sym sym)
				'()
				(merge_unique (map args (lambda (arg) (unnest_expr_outer_refs arg inner_aliases)))))))
	'())))
(define unnest_expr_has_outer_ref (lambda (expr inner_aliases) (match expr
	(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
		true
		(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
			(match args
				'(alias_ _ _ _) (and (not (nil? alias_))
					(not (reduce inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
				false)
			(if (_is_opaque_scope_sym sym)
				false
				(reduce args (lambda (a b) (or a (unnest_expr_has_outer_ref b inner_aliases))) false))))
	false)))
(define unnest_runtime_outer_ref_expr (lambda (expr) (match expr
	(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
		(match args
			(cons sym_arg '()) (begin
				(define ps (split (string sym_arg) "."))
				(match ps
					(list tbl col) (list (quote get_column) tbl false col false)
					_ expr))
			_ expr)
		(cons (unnest_runtime_outer_ref_expr sym) (map args unnest_runtime_outer_ref_expr)))
	expr)))
(define unnest_rewrite_inner_aliases (lambda (expr alias_lookup) (match expr
	'((symbol get_column) alias_ ti col ci) (begin
		(define na (alias_lookup alias_))
		(if (nil? na) expr (list (quote get_column) na false col false)))
	'((quote get_column) alias_ ti col ci) (begin
		(define na (alias_lookup alias_))
		(if (nil? na) expr (list (quote get_column) na false col false)))
	(cons sym args) (cons (unnest_rewrite_inner_aliases sym alias_lookup) (map args (lambda (arg)
		(unnest_rewrite_inner_aliases arg alias_lookup))))
	expr)))
(define unnest_correlated_domain_col (lambda (part has_outer_ref resolve_outer_ref) (match part
	'((symbol equal??) a b) (if (has_outer_ref a)
		(if (not (has_outer_ref b)) (list b (resolve_outer_ref a)) nil)
		(if (has_outer_ref b) (list a (resolve_outer_ref b)) nil))
	'((quote equal??) a b) (if (has_outer_ref a)
		(if (not (has_outer_ref b)) (list b (resolve_outer_ref a)) nil)
		(if (has_outer_ref b) (list a (resolve_outer_ref b)) nil))
	nil)))
(define unnest_correlated_residual_part? (lambda (part has_outer_ref) (match part
	'((symbol equal??) a b) (if (has_outer_ref a) (has_outer_ref b) (not (has_outer_ref b)))
	'((quote equal??) a b) (if (has_outer_ref a) (has_outer_ref b) (not (has_outer_ref b)))
	true)))
(define stage_has_group_boundary (lambda (stage) (begin
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(or
		(stage_is_dedup stage)
		(and (not (nil? sg)) (not (equal? sg '())))
		(not (nil? (stage_having_expr stage)))
		(not (nil? (stage_condition stage)))
	)
)))
(define rewrite_stage_for_flattened_aliases (lambda (stage rewrite_expr rewrite_alias) (begin
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_having_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(define spa_raw (stage_partition_aliases stage))
	(define spa (coalesceNil spa_raw '()))
	(define sc (stage_condition stage))
	(define sonce (stage_once_limit stage))
	(define init (stage_init_code stage))
	(define fin_order (map so (lambda (o) (match o '(c d) (list (rewrite_expr c) d) o))))
	(define fin_aliases (if (or (nil? spa_raw) (equal? spa '())) nil (map spa rewrite_alias)))
	(if (stage_is_dedup stage)
		(stage_rebuild_with_meta
			stage
			(make_dedup_stage (map sg rewrite_expr) fin_aliases)
			rewrite_expr
			rewrite_alias)
		(if (and (not (nil? fin_aliases)) (or (nil? sg) (equal? sg '())))
			(stage_rebuild_with_meta
				stage
				(make_stage
					'()
					nil
					fin_order
					(coalesceNil (stage_limit_partition_cols stage) 0)
					sl
					soff
					false
					fin_aliases
					init
					(if (nil? sc) nil (rewrite_expr sc))
					sonce)
				rewrite_expr
				rewrite_alias)
			(stage_rebuild_with_meta
				stage
				(if (nil? sc)
					(make_group_stage
						(map sg rewrite_expr)
						(if (nil? sh) nil (rewrite_expr sh))
						fin_order
						sl
						soff
						fin_aliases
						init)
					(make_group_stage_with_condition
						(map sg rewrite_expr)
						(if (nil? sh) nil (rewrite_expr sh))
						fin_order
						sl
						soff
						fin_aliases
						init
						(rewrite_expr sc)))
				rewrite_expr
				rewrite_alias)))))
)

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
(define make_select_core_term (lambda (uq_result)
	(list
		(quote select_core_term)
		(nth uq_result 0)
		(nth uq_result 1)
		(nth uq_result 2)
		(nth uq_result 3)
		(nth uq_result 4)
		(nth uq_result 5)
		(nth uq_result 6)
		(if (>= (count uq_result) 8) (nth uq_result 7) '())
	)
))
(define logical_query_term_is_select_core (lambda (term) (match term
	'(select_core_term _ _ _ _ _ _ _ _) true
	false
)))
(define logical_query_term_is_union_all (lambda (term) (match term
	'(union_all_term _ _ _ _) true
	false
)))
(define expr_has_any_wildcard_ref (lambda (expr) (match expr
	'((symbol get_column) _ _ "*" _) true
	'((quote get_column) _ _ "*" _) true
	(cons sym args) (or (expr_has_any_wildcard_ref sym) (reduce args (lambda (found arg) (or found (expr_has_any_wildcard_ref arg))) false))
	false
)))
(define lookup_query_field_expr (lambda (fields col col_insensitive)
	(reduce_assoc fields (lambda (acc k v)
		(if (not (nil? acc)) acc
			(if ((if col_insensitive equal?? equal?) k col) v nil)))
		nil)
))
(define rewrite_union_wrapper_expr (lambda (expr wrapper_alias branch_fields) (match expr
	'((symbol get_column) alias_ ti col ci) (if (or (equal?? alias_ wrapper_alias) (nil? alias_))
		(coalesce (lookup_query_field_expr branch_fields col ci) expr)
		expr)
	'((quote get_column) alias_ ti col ci) (if (or (equal?? alias_ wrapper_alias) (nil? alias_))
		(coalesce (lookup_query_field_expr branch_fields col ci) expr)
		expr)
	(cons sym args) (cons (rewrite_union_wrapper_expr sym wrapper_alias branch_fields)
		(map args (lambda (arg) (rewrite_union_wrapper_expr arg wrapper_alias branch_fields))))
	expr
)))
(define query_union_branch_is_simple_select (lambda (branch) (match branch
	'(_ _ fields _ group having order limit offset)
	(and
		(or (nil? group) (equal? group '()))
		(nil? having)
		(nil? order)
		(nil? limit)
		(nil? offset)
		(reduce_assoc fields (lambda (acc _k v) (and acc (equal? (extract_aggregates v) '()))) true))
	false
)))
	(define rewrite_query_term (lambda (query) (begin
		(define query_table_source_is_base? (lambda (source)
			(or
				(string? source)
				(not (equal? (scan_tagged_table_base source) source)))))
		(define select_has_from_subquery (lambda (query2) (match query2
			'(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2)
			(reduce tables2 (lambda (acc td) (or acc (match td
				'(_ _ tbl _ _) (not (query_table_source_is_base? tbl))
				false))) false)
			false
		)))
	(define rewrite_select_core_over_union_from (lambda (query2) (match query2
		'(schema tables fields condition group having order limit offset)
		(if (not (equal? (count tables) 1))
			query2
			(match (car tables)
				'(id schemax subquery false nil) (begin
					(define union_parts (query_union_all_parts subquery))
					(if (or
						(nil? union_parts)
						(not (or (nil? group) (equal? group '())))
						(not (nil? having))
						(reduce_assoc fields (lambda (acc _k v) (or acc (not (equal? (extract_aggregates v) '())))) false)
						(expr_has_any_wildcard_ref fields)
						(expr_has_any_wildcard_ref condition)
						(reduce (coalesceNil order '()) (lambda (acc item) (or acc (match item
							'(col _dir) (expr_has_any_wildcard_ref col)
							false))) false))
						query2
						(match union_parts '(branches union_order union_limit union_offset)
							(if (or (not (nil? union_order)) (not (nil? union_limit)) (not (nil? union_offset)))
								query2
								(if (not (reduce branches (lambda (acc branch) (and acc (query_union_branch_is_simple_select branch))) true))
									query2
									(list (quote union_all)
										(map branches (lambda (branch) (match branch
											'(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2) (begin
												(define rewritten_condition (rewrite_union_wrapper_expr (coalesceNil condition true) id fields2))
												(list schema2 tables2
													(map_assoc fields (lambda (k v) (rewrite_union_wrapper_expr v id fields2)))
													(if (or (nil? rewritten_condition) (equal? rewritten_condition true))
														condition2
														(if (or (nil? condition2) (equal? condition2 true))
															rewritten_condition
															(list (quote and) condition2 rewritten_condition)))
													nil nil nil nil nil)
											)
											_ branch)))
										order limit offset)))))
					query2))
			query2))))
	(define top_union_parts (query_union_all_parts query))
	(if (not (nil? top_union_parts))
		(match top_union_parts '(branches order limit offset)
			(list (quote union_all) (map branches rewrite_query_term) order limit offset))
		(if (not (select_has_from_subquery query))
			query
			(rewrite_select_core_over_union_from (match query
					'(schema tables fields condition group having order limit offset)
					(list schema
						(map tables (lambda (tbldesc) (match tbldesc
							'(alias schema2 tbl isOuter joinexpr) (if (query_table_source_is_base? tbl)
								tbldesc
								(begin
									(define rewritten_subquery (rewrite_query_term tbl))
									(list alias schema2
										(if (and (nil? rewritten_subquery) (query_is_select_core tbl))
											tbl
											rewritten_subquery)
										isOuter joinexpr)))
							tbldesc)))
					fields condition group having order limit offset)
				query))))
)))
(define logical_query_term_output_cols (lambda (term) (match term
	'(select_core_term _ _ fields _ _ _ _ _) (extract_assoc fields (lambda (k v) k))
	'(union_all_term branches _ _ _) (if (or (nil? branches) (equal? branches '()))
		'()
		(logical_query_term_output_cols (car branches)))
	_ (error "invalid logical query term")
)))
(define untangle_query_term (lambda (query outer_schemas) (begin
	(define rewritten_query (rewrite_query_term query))
	(define union_parts (query_union_all_parts rewritten_query))
	(if (nil? union_parts)
		(if (query_is_select_core rewritten_query)
			(begin
				(define uq_result (apply untangle_query (merge rewritten_query (list outer_schemas))))
				(make_select_core_term uq_result))
				(error "invalid SELECT query term"))
		(match union_parts '(branches order limit offset) (begin
			(if (or (nil? branches) (equal? branches '()))
				(error "UNION ALL requires at least one branch"))
			(list (quote union_all_term)
				(map branches (lambda (branch) (untangle_query_term branch outer_schemas)))
				order limit offset)
))))
)))
(define query_has_from_subquery (lambda (query) (match query
	'(schema tables fields condition group having order limit offset)
	(reduce tables (lambda (acc td) (or acc (match td
		'(_ _ (string? _tbl) _ _) false
		'(_ _ _ _ _) true
		false))) false)
	false
)))

/* make_keytable_schema: compute keytable name and schema without creating the table.
Used by untangle to predict the keytable name for HAVING subselect decorrelation.
Returns (keytable_name key_col_names schema_def) where schema_def is a list of
column descriptors suitable for the schemas assoc in untangle_query.
Does NOT handle FK→PK reuse (returns nil for that case — caller must check). */
(define make_keytable_schema (lambda (schema tbl keys tblvar) (begin
	(define keytable_source_name (planner-temp-source-name tbl tblvar))
	(if (equal? keytable_source_name "")
		(error (concat "make_keytable_schema: empty source name for tbl=" tbl " tblvar=" tblvar)))
	(define alias_map (list (list tblvar (concat schema "." keytable_source_name))))
	(define key_names (map keys (lambda (k)
		(sanitize_temp_name
			(canonical_expr_name (normalize_canonical_aliases (preserve_current_materialized_field_refs tbl tblvar k)) '(list) '(list) alias_map)))))
	(define keytable_name (compact-keytable-table-name keytable_source_name key_names nil))
	(define schema_def (map key_names (lambda (colname) (list "Field" colname "Type" "any"))))
	(list keytable_name key_names schema_def)
)))

/* make_keytable: create a canonically named group/key table with sloppy engine
Returns (keytable_name init_code fk_pk_col) where init_code is plan-time code that ensures
the table exists at execution time (survives cache eviction of sloppy tables).
fk_pk_col is non-nil when FK→PK reuse is active (parent table used instead of temp keytable).
condition_suffix: if non-nil, appended to name (for dedup stages with WHERE) */
/* make_keytable: compute keytable name, schema, and idempotent runtime init_code.
Returns (keytable_name kt_schema_def init_code).
- keytable_name: canonical dot-prefixed table name for the group/dedup keytable
- kt_schema_def: column schema for the planner (so build_scan can resolve get_columns)
- init_code: runtime expression — createtable returns true on first creation (caller
uses this as guard for initial collect + trigger deploy), false on subsequent calls
(incremental maintenance via triggers). partitiontable + touch_keytable are idempotent.
For FK→PK reuse: returns (parent_tbl parent_schema fk_init_code) where parent_schema
comes from show() on the existing parent table and fk_init_code creates the alias column. */
(define make_keytable (lambda (schema tbl keys tblvar condition_suffix) (begin
	/* physical_tbl: true for real user tables, false for planner-internal temps
	(dot-prefixed keytables/prejoins) that may not exist in storage at compile time */
	(define physical_tbl (and (string? tbl) (> (strlen tbl) 0) (not (equal? (substr tbl 0 1) "."))))
	(define keytable_source_name (planner-temp-source-name tbl tblvar))
	/* FK→PK reuse: if single-column GROUP BY on a FK column without condition,
	reuse the parent (referenced) table instead of creating a temp keytable. */
	(define fk_result (if (and physical_tbl (nil? condition_suffix) (equal? 1 (count keys)))
		(match (car keys)
			'('get_column (eval tblvar) false scol false) (begin
				(define fk_info (get_fk_target (table schema tbl) scol))
				(if (not (nil? fk_info))
					(begin
						(define alias_map (list (list tblvar (concat schema "." tbl))))
						(define key_name
							(sanitize_temp_name
								(canonical_expr_name
									(normalize_canonical_aliases
										(preserve_current_materialized_field_refs tbl tblvar (car keys)))
									'(list) '(list) alias_map)))
						(define parent_tbl (car fk_info))
						(define parent_col (car (cdr fk_info)))
						(define parent_schema (show schema parent_tbl))
						/* FK-reuse createcolumn on parent: safe at compile time (parent is a physical table) */
						(if (not (equal? key_name parent_col))
							(createcolumn (table schema parent_tbl) key_name "any" '() '("temp" true)
								(list parent_col)
								(eval (list 'lambda (list (symbol parent_col)) (symbol parent_col)))))
						(define fk_init (if (equal? key_name parent_col) nil
							(list 'createcolumn (list 'table schema parent_tbl) key_name "any"
								(list 'quote '())
								(list 'quote '("temp" true))
								(list 'quote (list parent_col))
								(list 'lambda (list (symbol parent_col)) (symbol parent_col)))))
						(list parent_tbl parent_schema fk_init key_name))
					nil))
			nil)
		nil))
	(if (not (nil? fk_result))
		fk_result
		(begin
			(define alias_map (list (list tblvar (concat schema "." keytable_source_name))))
			(define key_names (map keys (lambda (k)
				(sanitize_temp_name
					(canonical_expr_name (normalize_canonical_aliases (preserve_current_materialized_field_refs tbl tblvar k)) '(list) '(list) alias_map)))))
			(define condition_name (if (nil? condition_suffix) nil
				(fnv_hash (concat
					(canonical_expr_name (normalize_canonical_aliases (preserve_current_materialized_field_refs tbl tblvar condition_suffix)) '(list) '(list) alias_map)
					(runtime_cache_suffix_from_exprs (list condition_suffix))))))
			(define key_name_at (lambda (i) (nth key_names i)))
			(define key_at (lambda (i) (nth keys i)))
			(define keytable_name (compact-keytable-table-name keytable_source_name key_names condition_name))
			/* column definitions for runtime createtable */
			(define kt_cols_code (cons 'list
				(cons
					(cons 'list (cons "unique" (cons "group" (list (cons 'list key_names)))))
					(map key_names (lambda (colname) (list 'list "column" colname "any" '(list) '(list)))))))
			/* partition spec: shardcolumn deferred to runtime (table may not exist at compile time) */
			(define kt_partition_code (cons 'list (if physical_tbl
				(merge (map (produceN (count keys)) (lambda (i)
					(match (key_at i)
						'('get_column (eval tblvar) false scol false) (list (list 'list (key_name_at i) (list 'shardcolumn (list 'table schema tbl) scol)))
						'()))))
				'())))
			/* init_code: idempotent runtime keytable creation.
			createtable returns true on first creation — caller wraps collect + trigger deploy
			in this guard. On subsequent calls createtable returns false (table already exists,
			incrementally maintained by triggers). partitiontable and touch_keytable are idempotent. */
			(define init_code (list '!begin
				(list 'if (list 'createtable schema keytable_name kt_cols_code query_temp_table_options_code true)
					(list 'partitiontable (list 'table schema keytable_name) kt_partition_code)
					nil)
				(list 'touch_keytable (list 'table schema keytable_name))
				/* pre-resolve keytable pointer for inner loops */
				(tbl-define-code schema keytable_name)
				/* returns true when collect + trigger deploy is needed.
				createtable already handled the existence check.  Dot-prefixed
				keytables are hidden from (show schema), so using show here would
				force a full recollect on every query. */
				(list 'table_empty? (list 'table schema keytable_name))))
			(define kt_schema_def (map key_names (lambda (colname) (list "Field" colname "Type" "any"))))
			(list keytable_name kt_schema_def init_code nil)))
)))

/* build_agg_window_plan: generates the full plan for aggregate window functions (SUM/COUNT/MIN/MAX OVER).
Uses keytable infrastructure (same as GROUP BY): make_keytable + collect + createcolumn + scalar fetch.
Result query runs on the BASE table; window_func expressions are replaced with scalar keytable scans. */
(define build_agg_window_plan (lambda (schema tbl tblvar tables over_partition wf_resolved condition groups schemas replace_find_column fields isOuter replace_columns_from_expr extract_columns_for_tblvar scan_wrapper) (begin
	(define has_partition (not (equal? over_partition '())))
	(define partition_exprs (map over_partition replace_find_column))
	(define group_keys (if has_partition partition_exprs '(1)))
	(define canon_alias_map (list (list tblvar (concat schema "." tbl))))
	(define materialized_source (materialized-source? tbl))
	(define expr_name (lambda (expr)
		(canonical_expr_name (normalize_canonical_aliases (rewrite_materialized_source_columns tbl tblvar expr)) '(list) '(list) canon_alias_map)))
	(set condition (replace_find_column (coalesceNil condition true)))
	(define window_runtime_suffix (runtime_cache_suffix_from_exprs (merge
		(list condition)
		partition_exprs
		(merge (map wf_resolved (lambda (wf) (match wf '(fn args _) args '())))))))
	(define kt_result (make_keytable schema tbl group_keys tblvar nil))
	(match kt_result '(grouptbl kt_schema_def keytable_init fk_pk_col) (begin
		(define is_fk_reuse (not (nil? fk_pk_col)))
		/* register keytable schema so planner can resolve columns */
		(if (not (nil? kt_schema_def))
			(set schemas (set_assoc schemas grouptbl kt_schema_def)))
		(define tblvar_cols (if has_partition (merge_unique (map group_keys (lambda (col) (extract_columns_for_tblvar tblvar col)))) '()))
		(define materialized_cols (if materialized_source
			(materialized_source_physical_schema schema tbl tblvar schemas)
			'()))
		/* Design contract:
		Keep aggregate/window sentinels logical while naming and wiring stages.
		Only the scan expression of the current materialized source may lower a
		nested aggregate marker to the already materialized column that computes it.
		This prevents raw (aggregate ...) nodes from leaking into build_scan while
		still avoiding early physical substitution in the logical stage graph. */
		(define lower_window_runtime_expr (lambda (expr) (match expr
			(cons (symbol aggregate) agg_args) (begin
				(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
				(define match_col (if materialized_source
					(reduce materialized_cols (lambda (found coldef)
						(if (not (nil? found)) found
							(begin
								(define field_name (coldef "Field"))
								(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
									(equal? (substr field_name 0 (strlen agg_name)) agg_name)
									(equal? (substr field_name (strlen agg_name) 1) "|"))
									field_name
									nil))))
						nil)
					nil))
				(if (nil? match_col)
					(match agg_args
						'(agg_expr agg_reduce agg_neutral)
						(list (quote aggregate) (lower_window_runtime_expr agg_expr) agg_reduce agg_neutral)
						_ expr)
					(list (quote get_column) tblvar false match_col false)))
			(cons '(quote aggregate) agg_args) (begin
				(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
				(define match_col (if materialized_source
					(reduce materialized_cols (lambda (found coldef)
						(if (not (nil? found)) found
							(begin
								(define field_name (coldef "Field"))
								(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
									(equal? (substr field_name 0 (strlen agg_name)) agg_name)
									(equal? (substr field_name (strlen agg_name) 1) "|"))
									field_name
									nil))))
						nil)
					nil))
				(if (nil? match_col)
					(match agg_args
						'(agg_expr agg_reduce agg_neutral)
						(list (quote aggregate) (lower_window_runtime_expr agg_expr) agg_reduce agg_neutral)
						_ expr)
					(list (quote get_column) tblvar false match_col false)))
			(cons sym args) (cons sym (map args lower_window_runtime_expr))
			expr)))
		(set filtercols (if has_partition
			(merge_unique (list
				(extract_columns_for_tblvar tblvar condition)
				(extract_outer_columns_for_tblvar tblvar condition)))
			'()))
		/* collect plan */
		(define collect_plan (if (equal? group_keys '(1))
			'('insert '('table schema grouptbl) '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
			(begin
				(define keycols (merge_unique (map group_keys (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
				(scan_wrapper 'scan schema tbl
					(cons list filtercols)
					(list 'lambda (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
					(cons list keycols)
					(list 'lambda (map keycols (lambda (col) (symbol (concat tblvar "." col)))) (runtime_list_ast (map group_keys (lambda (expr) (replace_columns_from_expr expr)))))
					'('lambda '('acc 'rowvals) '('set_assoc 'acc 'rowvals true))
					'(list)
					'('lambda '('acc 'sharddict) '('insert '('table schema grouptbl) (cons 'list (map group_keys expr_name)) '('extract_assoc 'sharddict '('lambda '('k 'v) 'k)) '(list) '('lambda '() true) true))
					isOuter))))
		/* aggregate descriptors */
		(define agg_col_name (make_aggregate_cache_col_name expr_name condition window_runtime_suffix))
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
			(define runtime_expr (lower_window_runtime_expr expr))
			(define cols (extract_columns_for_tblvar tblvar runtime_expr))
			'('createcolumn '('table schema grouptbl) (agg_col_name ag) "any" '(list) '(list "temp" true)
				(cons list (map group_keys (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
				'('lambda (map group_keys (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
					(scan_wrapper 'scan schema tbl
						(cons list (merge tblvar_cols filtercols))
						'('lambda (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons 'and (cons (replace_columns_from_expr condition) (map group_keys (lambda (col) '('equal? (replace_columns_from_expr col) '('outer (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
						(cons list cols)
						'('lambda (map cols (lambda (col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr runtime_expr))
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
					(list 'scan '(context_session_get "__memcp_tx") (list 'table schema grouptbl)
						(cons 'list kt_key_names)
						/* filter: (equal? grouptbl.kt_key (outer tblvar.raw_col)) — zip kt_key_names with raw_col_names */
						(list 'lambda
							(map kt_key_names (lambda (kn) (symbol (concat grouptbl "." kn))))
							(cons 'and (map (produceN (count kt_key_names) (lambda (i) i)) (lambda (i)
								(list 'equal? (symbol (concat grouptbl "." (nth kt_key_names i))) (list 'outer (symbol (concat tblvar "." (nth raw_col_names i)))))))))
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false))
					(list 'scan '(context_session_get "__memcp_tx") (list 'table schema grouptbl) '(list) '('lambda '() true)
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false)))
			(cons sym args_) (cons sym (map args_ replace_wf_with_fetch))
			expr)))
		(define new_fields (map_assoc fields (lambda (k v) (replace_wf_with_fetch (replace_find_column v)))))
		(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil))
		/* Key collection is guarded by createtable, but aggregate columns are
		query-local requirements: a later COUNT/MAX/GROUP_CONCAT OVER may reuse an
		existing partition keytable created by SUM OVER and still needs its own
		createcolumn/compute step. */
		(list 'begin
			'('if keytable_init '('time collect_plan "collect") nil)
			'('time compute_plan "compute")
			scan_plan)))
)))

/* make_col_replacer: create a function that rewrites column/aggregate references to point at a group table
is_dedup=true: leave aggregates intact (for dedup stages)
is_dedup=false: replace aggregates with column fetches (for normal group stages) */
(define make_col_replacer (lambda (grouptbl condition is_dedup expr_name src_tblvar agg_col_name) (begin
	(define colname (lambda (expr) (if (nil? expr_name) (concat expr) (expr_name expr))))
	(define replacer (lambda (expr) (match expr
		(cons (symbol aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (agg_col_name rest) false))
		(cons '(quote aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (agg_col_name rest) false))
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

/* rewrite_for_prejoin: rewrite only columns coming from the prejoin source scope.
Outer tables must stay untouched so scoped GROUP stages can still join the
materialized prejoin/keytable back to the surrounding row stream. */
(define rewrite_for_prejoin (lambda (pjvar alias_map expr)
	(match expr
		'((symbol get_column) tblvar _ col _) (if (or (nil? tblvar) (nil? (alias_map tblvar))) expr
			'('get_column pjvar false (canonical_expr_name (normalize_canonical_aliases expr) '(list) '(list) alias_map) false))
		'((quote get_column) tblvar _ col _) (if (or (nil? tblvar) (nil? (alias_map tblvar))) expr
			'('get_column pjvar false (canonical_expr_name (normalize_canonical_aliases expr) '(list) '(list) alias_map) false))
		(cons sym args) (cons sym (map args (lambda (a) (rewrite_for_prejoin pjvar alias_map a))))
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
Returns an S-expression that, when wrapped in (lambda (OLD NEW session) ...) and eval'd, performs the insert. */
(define build_pj_insert_scan (lambda (scan_tables scan_condition trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
	(match scan_tables
		(cons '(tblvar schema tbl isOuter joinexpr) rest)
		(if (equal? tblvar trigger_tv)
			/* skip trigger table: replace its refs in both the carried scan_condition
			and any joinexprs that still reference this table. Also fold the skipped
			stage's own joinexpr into scan_condition, otherwise that join predicate is
			lost completely when the trigger table itself is not scanned. */
			(begin
				(define rewritten_condition
					(replace_tblvar_with_dict trigger_tv 'NEW scan_condition))
				(define rewritten_joinexpr
					(if (nil? joinexpr) true
						(replace_tblvar_with_dict trigger_tv 'NEW joinexpr)))
				(define combined_condition
					(if (equal? rewritten_joinexpr true)
						rewritten_condition
						(combine_and_terms (list rewritten_condition rewritten_joinexpr))))
				(define rewritten_rest
					(map rest (lambda (td) (match td
						'(rest_tblvar rest_schema rest_tbl rest_isOuter rest_joinexpr)
						(list rest_tblvar rest_schema rest_tbl rest_isOuter
							(if (nil? rest_joinexpr) nil
								(replace_tblvar_with_dict trigger_tv 'NEW rest_joinexpr)))
						td))))
				(build_pj_insert_scan rewritten_rest
					combined_condition
					trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
			)
			/* scan this other table */
			(begin
				(define base_tbl (scan_tagged_table_base tbl))
				(define tagged_scan (scan_tagged_table_needs_scan_order tbl))
				(define tbl_scan_order (scan_tagged_table_order tbl))
				(define tbl_scan_limit (scan_tagged_table_limit tbl))
				(define tbl_scan_offset (scan_tagged_table_offset tbl))
				(define tbl_scan_partcols (scan_tagged_table_partition_cols tbl))
				(define tbl_once_limit (scan_tagged_table_once_limit tbl))
				(define tblvar_is_scalar_helper (or (scalar_helper_root_alias? tblvar) (strlike (string tblvar) "domain_scalar_%")))
				(define tblvar_is_nested_scalar_helper (scalar_helper_nested_alias? tblvar))
				(set cols (merge_unique (list
					(extract_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
					(extract_later_joinexpr_columns_for_tblvar tblvar rest))))
				(define split_is_outer (and
					isOuter
					(not tblvar_is_nested_scalar_helper)
					(or (nil? tbl_once_limit) tblvar_is_scalar_helper)))
				(match (split_scan_condition split_is_outer joinexpr scan_condition rest) '(now_condition later_condition) (begin
					(define scan_now_condition (strip_outer_scalar_helper_ref_terms
						(if (and isOuter (or tblvar_is_scalar_helper tblvar_is_nested_scalar_helper))
							(scalar_helper_outer_join_terms tblvar now_condition)
							now_condition)))
					(set filtercols (merge_unique (list
						(extract_columns_for_tblvar tblvar scan_now_condition)
						(extract_outer_columns_for_tblvar tblvar scan_now_condition))))
					(define pj_filtercols (if tagged_scan
						(merge_unique filtercols cols)
						filtercols))
					(define pj_mapfn
						(list 'lambda (map cols (lambda (c) (symbol (concat tblvar "." c))))
							(build_pj_insert_scan rest later_condition trigger_tv false pj_schema pjtbl mat_cols mat_col_names)))
					(define pj_reduce (list 'lambda (list 'acc 'sub) (list 'merge 'acc 'sub)))
					(define pj_reduce2 (if is_outermost
						(list 'lambda (list 'acc 'shard_rows)
							(list 'insert (list 'table pj_schema pjtbl) (cons 'list mat_col_names) 'shard_rows (list) (list 'lambda (list) true) true))
						(list 'lambda (list 'acc 'shard_rows) (list 'merge 'acc 'shard_rows))))
					(if tagged_scan
						(begin
							(define ordercols (extract_scan_order_cols_for_tblvar tbl_scan_order tblvar))
							(define dirs (extract_scan_order_dirs_for_tblvar tbl_scan_order tblvar))
							(list 'scan_order '(context_session_get "__memcp_tx") (list 'table schema base_tbl)
								(cons 'list pj_filtercols)
								(list 'lambda (map pj_filtercols (lambda (c) (symbol (concat tblvar "." c))))
									(optimize (replace_columns_from_expr_for_scan tblvar scan_now_condition)))
								(cons 'list ordercols)
								(cons 'list dirs)
								tbl_scan_partcols
								(coalesceNil tbl_scan_offset 0)
								(coalesceNil tbl_scan_limit -1)
								(cons 'list cols)
								pj_mapfn
								pj_reduce
								(list)
								pj_reduce2
								isOuter))
						(list 'scan '(context_session_get "__memcp_tx") (list 'table schema base_tbl)
							(cons 'list pj_filtercols)
							/* filter lambda: (lambda (tv.col ...) compiled_condition) */
							(list 'lambda (map pj_filtercols (lambda (c) (symbol (concat tblvar "." c))))
								(optimize (replace_columns_from_expr_for_scan tblvar scan_now_condition)))
							(cons 'list cols)
							/* map lambda: (lambda (tv.col ...) recursive_inner_scan) */
							pj_mapfn
							/* reduce: merge */
							pj_reduce
							(list)
							/* reduce2: outermost inserts into pjtbl, inner levels merge */
							pj_reduce2
							isOuter))
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

/* build_prejoin_delete_plan: route prejoin helper row removal through the
normal DELETE planner. The old bespoke scan+$update builder duplicated DML
codegen and drifted out of sync with the shared mutation path. */
(define build_prejoin_delete_plan (lambda (pj_schema pjtbl ti_col_pairs) (begin
	(define delete_alias "_pj")
	(define delete_condition
		(if (equal? 1 (count ti_col_pairs))
			(list 'equal?
				(list 'get_column delete_alias false (car (car ti_col_pairs)) false)
				(list 'get_assoc 'OLD (cadr (car ti_col_pairs))))
			(cons 'and
				(map ti_col_pairs (lambda (p)
					(list 'equal?
						(list 'get_column delete_alias false (car p) false)
						(list 'get_assoc 'OLD (cadr p))))))))
	(build_dml_plan pj_schema pjtbl delete_alias
		(list (list delete_alias pj_schema pjtbl false nil))
		nil
		delete_condition
		nil nil nil))))

/*
=== untangle_query: logical rewrite / Neumann decorrelation ===

Implements the algebraic unnesting transformation from Neumann/Kemper (BTW 2015)
and the holistic top-down extension (Neumann BTW 2025). Transforms a parsed SQL
query with arbitrarily nested correlated subqueries into a flat relational IR:

INPUT:  parsed query (schema tables fields condition group having order limit offset)
OUTPUT: (schema tables fields condition groups schemas replace_find_column)

The output is a single flat table list where every correlated subquery has been
replaced by a LEFT JOIN table entry. Dependencies between nesting levels are
expressed as join conditions; aggregation boundaries are expressed as group-stages
with partition-aliases (scoping). There is no nested runtime code in the output.
The IR must stay purely logical: no inner_select, subscan, or derived-source
materialization model may remain after untangle_query.

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
FROM (SELECT ...) must be inlined here by renaming/term replacement; aggregate
window functions without a true physical ORDER requirement also belong here as
ordinary group/keytable rewrites, not as later physical planner semantics.
*/

/* Derived-table flattening must not recurse blindly into opaque runtime scopes,
because those already contain lowered var/resultrow shapes. Rewrite only outer
refs inside scan filter/map lambdas so wrapped correlated scalar subselects keep
seeing the correctly prefixed outer alias. */
(define prefix_flattened_outer_ref (lambda (flatten_id inner_schemas outer_arg) (begin
	(define s (string outer_arg))
	(define parts (split s "."))
	(match parts
		(list tbl col) (if (not (nil? (inner_schemas tbl)))
			(list (quote outer) (symbol (concat flatten_id "\0" tbl "." col)))
			(list (quote outer) outer_arg))
		_ (list (quote outer) outer_arg))
)))
(define opaque_expr_has_outer_ref (lambda (expr) (match expr
	'((symbol outer) _) true
	'((quote outer) _) true
	(cons sym args) (if (is_quote_scope_sym sym)
		false
		(reduce args (lambda (found arg) (or found (opaque_expr_has_outer_ref arg))) false))
	false
)))
(define rewrite_opaque_outer_expr_for_flatten (lambda (flatten_id inner_schemas expr) (match expr
	'((symbol outer) outer_arg) (prefix_flattened_outer_ref flatten_id inner_schemas outer_arg)
	'((quote outer) outer_arg) (prefix_flattened_outer_ref flatten_id inner_schemas outer_arg)
	(cons sym args) (if (is_quote_scope_sym sym)
		expr
		(if (reduce args (lambda (found arg) (or found (opaque_expr_has_outer_ref arg))) false)
			(cons sym (map args (lambda (arg)
				(if (opaque_expr_has_outer_ref arg)
					(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas arg)
					arg))))
			expr))
	expr
)))
(define rewrite_opaque_outer_lambda_for_flatten (lambda (flatten_id inner_schemas fn) (match fn
	'((symbol lambda) params body)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body))
	'((symbol lambda) params body numvars)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body) numvars)
	'((quote lambda) params body)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body))
	'((quote lambda) params body numvars)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body) numvars)
	fn
)))
(define rewrite_opaque_outer_alias_for_flatten (lambda (flatten_id inner_schemas expr) (match expr
	(cons (symbol !begin) forms)
	(cons (quote !begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons '(quote !begin) forms)
	(cons '(quote !begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons (symbol begin) forms)
	(cons (quote begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons '(quote begin) forms)
	(cons '(quote begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons (symbol set) (cons lhs (cons rhs tail)))
	(cons (quote set) (cons lhs (cons (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas rhs) tail)))
	(cons '(quote set) (cons lhs (cons rhs tail)))
	(cons '(quote set) (cons lhs (cons (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas rhs) tail)))
	(cons scanhead (cons tx (cons schema3 (cons tbl3 rest))))
	(match (string scanhead)
		"scan" (match rest
			(cons filtercols (cons filterfn (cons mapcols (cons mapfn tail))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons mapcols
										(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail))))))))
			expr)
		"scan_order" (match rest
			(cons filtercols (cons filterfn (cons sortcols (cons sortdirs (cons sortpartcols (cons offset (cons limit (cons mapcols (cons mapfn tail)))))))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons sortcols
										(cons sortdirs
											(cons sortpartcols
												(cons offset
													(cons limit
														(cons mapcols
															(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail)))))))))))))
			expr)
		"scan_batch" (match rest
			(cons filtercols (cons filterfn (cons mapcols (cons mapfn tail))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons mapcols
										(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail))))))))
			expr)
		_ expr)
	expr
)))

(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas_param) (begin
	(set rename_prefix (coalesce rename_prefix ""))
	(define outer_schema_bindings (schema_bindings_to_flat_list outer_schemas_param))
	(define scalar_subquery_cache (newsession))
	(scalar_subquery_cache "init" '())
	(define dep_scalar_cache (newsession))

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

	(define make_replace_find_column_subselect (lambda (schemas2 outer_schemas preserve_grouped_outer_domain) (begin
		/* force optimizer to retain both params by using them directly in the outer body */
		(define _s schemas2)
		(define _o outer_schemas)
		(define _preserve_grouped_outer_domain preserve_grouped_outer_domain)
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
		(define outer_expr_is_domain_safe (lambda (expr)
			(and
				(not (expr_has_opaque_scope expr))
				(equal? (extract_aggregates expr) '())
				(equal? (extract_window_funcs expr) '()))))
		(define outer_alias_requires_domain_preservation (lambda (outer_alias) (begin
			(define outer_cols (if (has_assoc? _o outer_alias) (_o outer_alias) nil))
			(and _preserve_grouped_outer_domain
				(not (nil? outer_cols))
				(reduce outer_cols (lambda (needs_preserve coldef)
					(or needs_preserve
						(begin
							(define expr (coalesceNil (coldef "Expr") nil))
							(and (not (nil? expr))
								(not (outer_expr_is_domain_safe expr))))))
					false)))))
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
								(define outer_column (coalesce (canonical_column_in_schema _o alias_name table_insensitive column_name column_insensitive) column_name))
								(define outer_cols (_o outer_alias))
								(define outer_coldef (reduce outer_cols (lambda (a coldef) (if (and (nil? a) (equal? (coldef "Field") outer_column)) coldef a)) nil))
								(define outer_expr (if outer_coldef (outer_coldef "Expr") nil))
								(if (or _preserve_grouped_outer_domain
									(outer_alias_requires_domain_preservation outer_alias))
									/* grouped/windowed outer aliases define the visible correlation
									domain; do not collapse them back into base-table refs */
									(list (quote outer) (symbol (concat outer_alias "." outer_column)))
									(if (and (not (nil? outer_expr)) (outer_expr_is_domain_safe outer_expr))
										/* simple pass-through/computed wrapper columns may still inline */
										(wrap_outer_leaves outer_expr)
										(list (quote outer) (symbol (concat outer_alias "." outer_column)))))))
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
	(define _raw_query_local_aliases (lambda (query) (match query
		'(_ raw_tables _ _ _ _ _ _ _) (reduce raw_tables (lambda (acc td)
			(match td
				'(alias _ _ _ _) (append_unique acc alias)
				acc))
			'())
		'())))
	(define _alias_in_list (lambda (aliases alias_name)
		(reduce aliases (lambda (acc alias_) (or acc (equal?? alias_ alias_name))) false)))
	(define _raw_query_uses_alias_outside_current (lambda (query current_aliases) (match query
		'(_ raw_tables raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
			(define nested_local_aliases (_raw_query_local_aliases query))
			(define raw_expr_uses_alias_outside_current (lambda (expr) (match expr
				'((symbol get_column) alias_ _ _ _) (and (not (nil? alias_))
					(not (_alias_in_list nested_local_aliases alias_))
					(not (_alias_in_list current_aliases alias_)))
				'((quote get_column) alias_ _ _ _) (and (not (nil? alias_))
					(not (_alias_in_list nested_local_aliases alias_))
					(not (_alias_in_list current_aliases alias_)))
				(cons sym args) (reduce args (lambda (acc arg) (or acc (raw_expr_uses_alias_outside_current arg))) false)
				false)))
			(or
				(reduce_assoc raw_fields (lambda (acc _k v) (or acc (raw_expr_uses_alias_outside_current v))) false)
				(raw_expr_uses_alias_outside_current (coalesceNil raw_condition true))
				(reduce (coalesceNil raw_group '()) (lambda (acc gexpr) (or acc (raw_expr_uses_alias_outside_current gexpr))) false)
				(raw_expr_uses_alias_outside_current (coalesceNil raw_having true))
				(reduce (coalesceNil raw_order '()) (lambda (acc order_item)
					(or acc (match order_item
						'(col _dir) (raw_expr_uses_alias_outside_current col)
						false)))
					false)))
		false)))
	(define _raw_query_contains_skip_level_nested_outer_ref (lambda (query current_aliases) (match query
		'(_ _ raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
			(define nested_current_aliases (append_unique current_aliases (_raw_query_local_aliases query)))
			(define raw_expr_contains_skip_level_nested_outer_ref (lambda (expr) (match expr
				(cons sym args) (begin
					(define kind (inner_select_kind sym))
					(define nested_subquery (if (nil? kind) nil
						(match kind
							(quote inner_select) (match args
								(cons inner_subquery '()) inner_subquery
								nil)
							(quote inner_select_in) (match args
								(cons _target_expr (cons inner_subquery '())) inner_subquery
								nil)
							(quote inner_select_exists) (match args
								(cons inner_subquery '()) inner_subquery
								nil)
							nil)))
					(or
						(and (not (nil? nested_subquery))
							(or
								(_raw_query_uses_alias_outside_current nested_subquery nested_current_aliases)
								(_raw_query_contains_skip_level_nested_outer_ref nested_subquery nested_current_aliases)))
						(reduce args (lambda (acc arg) (or acc (raw_expr_contains_skip_level_nested_outer_ref arg))) false)))
				false)))
			(or
				(reduce_assoc raw_fields (lambda (acc _k v) (or acc (raw_expr_contains_skip_level_nested_outer_ref v))) false)
				(raw_expr_contains_skip_level_nested_outer_ref (coalesceNil raw_condition true))
				(reduce (coalesceNil raw_group '()) (lambda (acc gexpr) (or acc (raw_expr_contains_skip_level_nested_outer_ref gexpr))) false)
				(raw_expr_contains_skip_level_nested_outer_ref (coalesceNil raw_having true))
				(reduce (coalesceNil raw_order '()) (lambda (acc order_item)
					(or acc (match order_item
						'(col _dir) (raw_expr_contains_skip_level_nested_outer_ref col)
						false)))
					false)))
		false)))
	(define scalar_subselect_shape_facts (lambda (subquery outer_schemas) (match subquery
		'(_ _ flds _ g h o l off) (begin
			(define value_expr (match flds
				(cons _ (cons v _)) v
				nil))
			(define has_outer (_subquery_has_outer_refs subquery outer_schemas))
			(list
				g h o l off
				value_expr
				has_outer
				(if has_outer
					(_subquery_outer_refs_are_direct_columns subquery outer_schemas)
					true)
				(_contains_inner_select_marker subquery)
				(not (equal? (if (nil? value_expr) '() (extract_aggregates value_expr)) '()))
				(expr_uses_session_state subquery)
				(_raw_query_contains_skip_level_nested_outer_ref subquery (_raw_query_local_aliases subquery))))
		nil)))
	(define scalar_subselect_inline_raw_flags (lambda (subquery) (match subquery
		'(_ _ _ _ g h o l off)
		(list
			g h o l off
			(expr_uses_session_state subquery)
			(_raw_query_contains_skip_level_nested_outer_ref subquery (_raw_query_local_aliases subquery)))
		nil)))
	(define scalar_subselect_lowering_facts (lambda (subquery outer_schemas) (match subquery
		'(_ _ flds _ g h o _ _) (begin
			(define value_expr (match flds
				(cons _ (cons v _)) v
				nil))
			(define has_outer (_subquery_has_outer_refs subquery outer_schemas))
			(list
				g h o
				value_expr
				has_outer
				(if has_outer
					(_subquery_outer_refs_are_direct_columns subquery outer_schemas)
					true)
				(_contains_inner_select_marker subquery)
				(not (equal? (if (nil? value_expr) '() (extract_aggregates value_expr)) '()))))
		nil)))
	(define scalar_subselect_inline_reason planner_scalar_subselect_inline_reason)
	(define scalar_subselect_inline_strategy planner_scalar_subselect_inline_strategy)
	(define scalar_subselect_lowering_reason_from_facts planner_scalar_subselect_lowering_reason_from_facts)
	(define untangle_scalar_subquery_scope (lambda (subquery outer_schemas raw_group raw_having raw_order raw_limit raw_offset) (begin
		/* Shared logical scope preparation for scalar subqueries.
		This keeps recursive untangle + default stage synthesis in one place so the
		future top-down dependent-join pass can replace exactly this boundary. */
		(match (apply untangle_query (merge subquery (list outer_schemas)))
			'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2)
			(begin
				(define groups2 (coalesceNil groups2 '()))
				(define groups2 (if (or (nil? groups2) (equal? groups2 '()))
					(if (or raw_group raw_having raw_order raw_limit raw_offset)
						(list (make_group_stage raw_group raw_having raw_order raw_limit raw_offset nil nil))
						groups2)
					groups2))
				(list schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2))
			nil))))
	(define prepare_scalar_subselect_inline_scope (lambda (subquery outer_schemas raw_group raw_having raw_order raw_limit raw_offset) (begin
		/* This is the logical scope-normalization boundary for scalar inline lowering.
		It must stay free of runtime scan/promise construction so a future top-down
		dependent-join pass can hook in here without re-walking the old fallback code. */
		(match (untangle_scalar_subquery_scope subquery outer_schemas raw_group raw_having raw_order raw_limit raw_offset)
			'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2)
			(begin
				(define replace_find_column_subselect (make_replace_find_column_subselect schemas2 outer_schemas false))
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
					'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci)
						/* Keep qualified non-local refs as get_column markers here.
						replace_columns_from_expr lowers them to symbols in the
						actual scan lambda where the optimizer can derive the
						correct number of (outer ...) hops from real nesting. */
						(not (nil? (reduce_assoc outer_schemas (lambda (a k v) (or a (equal?? k alias_))) false))))
						e
						e)
					(cons sym args) (cons (wrap_unresolved_outer sym) (map args wrap_unresolved_outer))
					e
				)))
				(set fields2 (map_assoc fields2 (lambda (k v) (wrap_unresolved_outer v))))
				(set condition2 (wrap_unresolved_outer condition2))
				(list schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column_subselect _init2 value_expr))
			nil))))
	(define build_scalar_subselect_inline_with_strategy (lambda (subquery outer_schemas) (begin
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			(begin
				(planner_debug_record_scalar_event (quote inline-strategy) (quote inline-union-all-not-supported))
				(error "scalar subselect UNION ALL is not supported yet"))
			(begin
				(match (scalar_subselect_inline_raw_flags subquery)
					'(raw_group raw_having raw_order raw_limit raw_offset scalar_uses_session_state raw_contains_skip_level_nested_outer_ref)
					(begin
						(match (prepare_scalar_subselect_inline_scope
							subquery outer_schemas
							raw_group raw_having raw_order raw_limit raw_offset)
							'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column_subselect _init2 value_expr)
							(begin
								/* Software contract: scalar aggregates are split by canonical
								correlation, not by raw parser shape.
								- uncorrelated aggregates go through the helper-table/keytable path
								and may be globally memoized
								- correlated aggregates stay on the per-row direct scan path until
								the helper-table path can safely carry row-local promises
								The correlation test therefore has to run on resolved planner
								expressions so derived-table aliases and wrapped outer refs are
								classified correctly. */
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
								(define direct_agg_stages_simple (or (equal? groups2 '())
									(and (equal? (count groups2) 1)
										(not (stage_is_dedup stage2)))))
								(define stage2_group (if stage2 (coalesceNil (stage_group_cols stage2) '()) '()))
								(define stage2_post_group_condition (if stage2 (stage_post_group_condition_expr stage2) nil))
								(define contains_noncolumn_outer_ref (lambda (expr) (match expr
									'((quote outer) outer_sym) (equal? 1 (count (split (string outer_sym) ".")))
									'((symbol outer) outer_sym) (equal? 1 (count (split (string outer_sym) ".")))
									(cons sym args) (or (contains_noncolumn_outer_ref sym) (reduce args (lambda (a arg) (or a (contains_noncolumn_outer_ref arg))) false))
									false
								)))
								(define has_noncolumn_outer_ref (or
									(contains_noncolumn_outer_ref value_expr)
									(contains_noncolumn_outer_ref condition2)
								))
								(define contains_inner_select_marker (lambda (expr) (match expr
									(cons sym args) (or
										(not (nil? (inner_select_kind sym)))
										(contains_inner_select_marker sym)
										(reduce args (lambda (found arg) (or found (contains_inner_select_marker arg))) false))
									false)))
								(define contains_outer_ref (lambda (expr) (match expr
									'((quote outer) _) true
									'((symbol outer) _) true
									(cons sym args) (or
										(contains_outer_ref sym)
										(reduce args (lambda (found arg) (or found (contains_outer_ref arg))) false))
									false)))
								(define collapse_runtime_outer_refs (lambda (expr) (match expr
									'((quote outer) inner_expr) (match inner_expr
										(symbol inner_sym) (if (equal? 1 (count (split (string inner_sym) ".")))
											inner_expr
											expr)
										'((symbol var) _) inner_expr
										'((quote var) _) inner_expr
										'((quote outer) _) (collapse_runtime_outer_refs inner_expr)
										'((symbol outer) _) (collapse_runtime_outer_refs inner_expr)
										_ expr)
									'((symbol outer) inner_expr) (match inner_expr
										(symbol inner_sym) (if (equal? 1 (count (split (string inner_sym) ".")))
											inner_expr
											expr)
										'((symbol var) _) inner_expr
										'((quote var) _) inner_expr
										'((quote outer) _) (collapse_runtime_outer_refs inner_expr)
										'((symbol outer) _) (collapse_runtime_outer_refs inner_expr)
										_ expr)
									(cons sym args) (cons sym (map args collapse_runtime_outer_refs))
									expr)))
								(define stage_contains_outer_ref (lambda (stage)
									(or
										(reduce (coalesceNil (stage_group_cols stage) '()) (lambda (found expr) (or found (contains_outer_ref expr))) false)
										(contains_outer_ref (coalesceNil (stage_post_group_condition_expr stage) true))
										(reduce (coalesceNil (stage_order_list stage) '()) (lambda (found order_item)
											(or found (match order_item
												'(col _dir) (contains_outer_ref col)
												(contains_outer_ref order_item)))) false))))
								(define scalar_has_outer_ref (or
									(reduce_assoc fields2 (lambda (found _k v) (or found (contains_outer_ref v))) false)
									(contains_outer_ref condition2)
									(reduce (coalesceNil groups2 '()) (lambda (found stage) (or found (stage_contains_outer_ref stage))) false)))
								(define scalar_subselect_fallback_take_first_without_pushdown (lambda ()
									(and
										(not (nil? raw_limit))
										(<= raw_limit 1)
										(or (nil? raw_offset) (equal? raw_offset 0))
										(equal? (coalesceNil raw_order '()) '()))))
								(define build_scalar_subselect_via_legacy_fallback (lambda () (begin
									(define scalar_subquery_hash (fnv_hash (concat tables2 "|" fields2 "|" condition2)))
									(define scalar_subquery_idx (coalesceNil (scalar_subquery_cache "idx") 0))
									(scalar_subquery_cache "idx" (+ scalar_subquery_idx 1))
									(define scalar_subquery_name_suffix
										(concat scalar_subquery_hash "_" (string scalar_subquery_idx)))
									(define scalar_subquery_promise_name (concat "__scalar_promise_" scalar_subquery_name_suffix))
									(define scalar_subquery_resultrow_name (concat "__scalar_resultrow_" scalar_subquery_name_suffix))
									(define scalar_subquery_take_first_without_pushdown (scalar_subselect_fallback_take_first_without_pushdown))
									(define scalar_resultrow_head? (lambda (sym)
										(or
											(equal? sym (quote resultrow))
											(equal? (string sym) "resultrow")
											(strlike (string sym) "%resultrow%")
											(match sym
												'((symbol symbol) "resultrow") true
												'((quote symbol) "resultrow") true
												'(symbol "resultrow") true
												false))))
									(define scalar_resultrow_args (lambda (args)
										(map args (lambda (arg) (match arg
											(cons quote_sym quoted_args)
											(if (is_quote_scope_sym quote_sym)
												(begin
													(define row_items (if (and (equal? (count quoted_args) 1) (list? (car quoted_args)))
														(car quoted_args)
														(if (> (count quoted_args) 1) quoted_args nil)))
													(if (nil? row_items)
														(replace_resultrow arg)
														(runtime_heap_list_ast row_items)))
												(replace_resultrow arg))
											(replace_resultrow arg))))))
									(define scalar_resultrow_symbol? (lambda (sym)
										(or
											(equal? sym (quote resultrow))
											(equal? sym (symbol resultrow))
											(equal? (string sym) "resultrow"))))
									(define resultrow_set_stmt? (lambda (stmt) (match stmt
										'(set sym _) (scalar_resultrow_symbol? sym)
										'((quote set) sym _) (scalar_resultrow_symbol? sym)
										'((symbol set) sym _) (scalar_resultrow_symbol? sym)
										false)))
									(define begin_symbol? (lambda (sym)
										(or
											(equal? sym (quote begin))
											(equal? sym (symbol begin))
											(equal? sym '(quote begin))
											(equal? sym (quote !begin))
											(equal? sym (symbol !begin))
											(equal? sym '(quote !begin)))))
									(define resultrow_rebinding_scope? (lambda (sym args)
										(and
											(begin_symbol? sym)
											(reduce args (lambda (found stmt)
												(or found (resultrow_set_stmt? stmt)))
												false))))
									(define replace_resultrow (lambda (expr) (match expr
										(cons sym args) (if (resultrow_rebinding_scope? sym args)
											expr
											(if (scalar_resultrow_head? sym)
												(cons (symbol scalar_subquery_resultrow_name)
													(scalar_resultrow_args args))
												(if (and (equal? sym (quote symbol)) (equal? args '("resultrow")))
													(list (quote symbol) scalar_subquery_resultrow_name)
													(begin
														(define replaced_sym (replace_resultrow sym))
														(if (equal? replaced_sym (symbol scalar_subquery_resultrow_name))
															(cons replaced_sym (scalar_resultrow_args args))
															(cons replaced_sym (map args replace_resultrow))))
												)
										))
										expr
									)))
									(define fallback_groups (if scalar_subquery_take_first_without_pushdown
										(map groups2 (lambda (stage)
											(if (or (stage_is_dedup stage) (not (nil? (stage_partition_aliases stage))))
												stage
												(stage_preserve_cache_meta stage
													(make_group_stage_with_condition
														(coalesceNil (stage_group_cols stage) '())
														(stage_having_expr stage)
														(coalesceNil (stage_order_list stage) '())
														nil nil
														(stage_partition_aliases stage)
														(stage_init_code stage)
														(stage_condition stage))))))
										groups2))
									(define subplan (normalize_quoted_scalar_resultrow_calls
										(replace_resultrow (build_queryplan schema2 tables2 fields2 condition2 fallback_groups schemas2 replace_find_column_subselect nil))))
									(define init_stmts (if (or (nil? _init2) (equal? _init2 '())) '()
										(map _init2 normalize_quoted_scalar_resultrow_calls)))
									(cons (quote !begin) (merge init_stmts (list
										(list (quote set) (symbol scalar_subquery_promise_name) (list (quote newpromise)))
										(list (quote set) (symbol scalar_subquery_resultrow_name)
											(list (quote lambda) (list (symbol "row"))
												(if scalar_subquery_take_first_without_pushdown
													(list (quote if)
														(list (quote nil?) (list (symbol scalar_subquery_promise_name) "state"))
														(list (symbol scalar_subquery_promise_name) "value" (list (quote nth) (symbol "row") 1))
														0)
													(list (symbol scalar_subquery_promise_name) "once"
														(list (quote nth) (symbol "row") 1)
														"scalar subselect returned more than one row"))
											)
										)
										subplan
										(list (symbol scalar_subquery_promise_name) "value")
									)))
								)))
								(define build_scalar_subselect_via_direct_agg_scan (lambda () (begin
									(define agg_item (nth _agg_args 0))
									(define agg_reduce (nth _agg_args 1))
									(define agg_neutral (nth _agg_args 2))
									(define build_scalar_agg_scan (lambda (scan_tables scan_condition)
										(match scan_tables
											(cons '(tblvar schema3 tbl3 isOuter3 joinexpr3) rest_tables) (begin
												(define tbl3_once_limit (scan_tagged_table_once_limit tbl3))
												(define tblvar_is_scalar_helper (or (scalar_helper_root_alias? tblvar) (strlike (string tblvar) "domain_scalar_%")))
												(define tblvar_is_nested_scalar_helper (scalar_helper_nested_alias? tblvar))
												(define direct_agg_lower_scan_expr (lambda (expr) (match expr
													'((symbol aggregate) _) expr
													'((symbol get_column) alias_ _ col _)
													(if (equal?? alias_ tblvar)
														(symbol (concat alias_ "." col))
														(list (quote outer) (symbol (concat alias_ "." col))))
													'((quote get_column) alias_ _ col _)
													(if (equal?? alias_ tblvar)
														(symbol (concat alias_ "." col))
														(list (quote outer) (symbol (concat alias_ "." col))))
													(cons sym args) (cons (direct_agg_lower_scan_expr sym)
														(map args direct_agg_lower_scan_expr))
													expr)))
												(define cur_cols (merge_unique (list
													(extract_columns_for_tblvar tblvar scan_condition)
													(extract_columns_for_tblvar tblvar agg_item)
													(extract_outer_columns_for_tblvar tblvar scan_condition)
													(extract_outer_columns_for_tblvar tblvar agg_item)
													(extract_later_joinexpr_columns_for_tblvar tblvar rest_tables)
												)))
												(define split_is_outer (and
													isOuter3
													(not tblvar_is_nested_scalar_helper)
													(or (nil? tbl3_once_limit) tblvar_is_scalar_helper)))
												(match (split_scan_condition split_is_outer joinexpr3 scan_condition rest_tables) '(now_condition later_condition) (begin
													(define scan_now_condition (strip_outer_scalar_helper_ref_terms
														(if (or tblvar_is_scalar_helper tblvar_is_nested_scalar_helper)
															(scalar_helper_outer_join_terms tblvar now_condition)
															now_condition)))
													(define filtercols (merge_unique (list
														(extract_columns_for_tblvar tblvar scan_now_condition)
														(extract_outer_columns_for_tblvar tblvar scan_now_condition)
													)))
													(define inner_body (build_scalar_agg_scan rest_tables later_condition))
													(define filterbody (direct_agg_lower_scan_expr scan_now_condition))
													(scan_wrapper 'scan schema3 tbl3
														(cons list filtercols)
														(list (quote lambda)
															(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
															filterbody
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
											'() (collapse_runtime_outer_refs (replace_columns_from_expr agg_item))
										)
									))
									(define init_stmts_agg (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
									(if (equal? init_stmts_agg '())
										(build_scalar_agg_scan tables2 condition2)
										(cons (quote !begin) (merge init_stmts_agg (list (build_scalar_agg_scan tables2 condition2)))))
								)))
								(define scalar_strategy (scalar_subselect_inline_strategy
									_agg_args
									direct_agg_stages_simple
									raw_contains_skip_level_nested_outer_ref
									scalar_uses_session_state
									stage2_post_group_condition
									stage2_group
									tables2
									scalar_has_outer_ref))
								(planner_debug_record_scalar_event (quote inline-strategy) scalar_strategy)
								(list scalar_strategy
									(if (equal? scalar_strategy (quote inline-direct-agg-scan))
										(build_scalar_subselect_via_direct_agg_scan)
										(build_scalar_subselect_via_legacy_fallback)))
							)
						)
				))
			)
		)
	)
	)
	)
	(define build_scalar_subselect_inline (lambda (subquery outer_schemas) (begin
		(match (build_scalar_subselect_inline_with_strategy subquery outer_schemas)
			'(_ lowered_expr) lowered_expr
			nil)
	)))
	(define build_exists_subselect (lambda (subquery outer_schemas) (match subquery
		'(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2)
		(begin
			(define local_aliases (_raw_query_local_aliases subquery))
			(define condition_parts (flatten_and_terms (coalesceNil condition2 true)))
			(define part_has_unqualified_column? (lambda (expr) (match expr
				'((symbol get_column) nil _ _ _) true
				'((quote get_column) nil _ _ _) true
				(cons sym args) (reduce args (lambda (found arg) (or found (part_has_unqualified_column? arg))) false)
				false)))
			(define part_tblvars_outside_inner_selects (lambda (expr) (match expr
				'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
				'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
				(cons sym args)
				(if (not (nil? (inner_select_kind sym)))
					'()
					(merge_unique (map args part_tblvars_outside_inner_selects)))
				'())))
			(define raw_query_refs_aliases? (lambda (query aliases) (match query
				'(_ raw_tables raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
					(define nested_aliases (_raw_query_local_aliases query))
					(define raw_expr_refs_aliases? (lambda (expr) (match expr
						'((symbol get_column) alias_ _ _ _) (and
							(not (nil? alias_))
							(not (_alias_in_list nested_aliases alias_))
							(_alias_in_list aliases alias_))
						'((quote get_column) alias_ _ _ _) (and
							(not (nil? alias_))
							(not (_alias_in_list nested_aliases alias_))
							(_alias_in_list aliases alias_))
						(cons expr_sym expr_args)
						(reduce expr_args (lambda (found expr_arg)
							(or found (raw_expr_refs_aliases? expr_arg))) false)
						false)))
					(or
						(reduce_assoc raw_fields (lambda (found _k field_expr)
							(or found (raw_expr_refs_aliases? field_expr))) false)
						(raw_expr_refs_aliases? (coalesceNil raw_condition true))
						(reduce (coalesceNil raw_group '()) (lambda (found group_expr)
							(or found (raw_expr_refs_aliases? group_expr))) false)
						(raw_expr_refs_aliases? (coalesceNil raw_having true))
						(reduce (coalesceNil raw_order '()) (lambda (found order_item)
							(or found (match order_item
								'(order_expr _dir) (raw_expr_refs_aliases? order_expr)
								false))) false)))
				false)))
			(define part_inner_select_refs_local? (lambda (expr) (match expr
				(cons sym args)
				(if (not (nil? (inner_select_kind sym)))
					(match args
						(cons nested_query _) (raw_query_refs_aliases? nested_query local_aliases)
						false)
					(reduce args (lambda (found arg)
						(or found (part_inner_select_refs_local? arg))) false))
				false)))
			(define part_refs_local? (lambda (part)
				(or
					(part_has_unqualified_column? part)
					(part_inner_select_refs_local? part)
					(reduce (part_tblvars_outside_inner_selects part) (lambda (found tv)
						(or found (_alias_in_list local_aliases tv))) false))))
			(define local_parts (filter condition_parts part_refs_local?))
			(define guard_parts (filter condition_parts (lambda (part)
				(not (part_refs_local? part)))))
			(define replace_guard_inner_selects (lambda (expr) (match expr
				(cons sym args) (begin
					(define kind (inner_select_kind sym))
					(if (equal?? kind (quote inner_select))
						(match args
							(cons guard_subquery '())
							(coalesce
								(_unnest_scalar_subselect guard_subquery outer_schemas)
								(replace_inner_selects expr outer_schemas))
							_ (replace_inner_selects expr outer_schemas))
						(cons sym (map args replace_guard_inner_selects))))
				expr)))
			(define boolean_scalar_inner_selects_to_exists (lambda (expr) (begin
				(define nil_symbol? (lambda (sym) (match sym
					(symbol nil?) true
					'nil? true
					'(quote nil?) true
					false)))
				(define scalar_value_exists (lambda (bool_subquery value_condition original_expr) (match bool_subquery
					'(bs bt bf bc bg bh bo bl boff)
					(begin
						(define bool_value_expr (match bf
							(cons _ (cons v _)) v
							nil))
						(define value_condition_expr
							(if (equal?? value_condition (quote not-nil))
								(list (quote not) (list (quote nil?) bool_value_expr))
								bool_value_expr))
						(if (and (not (nil? bool_value_expr))
							(equal? (coalesceNil bg '()) '())
							(nil? bh)
							(equal? (coalesceNil bo '()) '())
							(or (nil? bl) (<= bl 1))
							(or (nil? boff) (equal? boff 0)))
							(list (quote inner_select_exists)
								(list bs bt bf
									(combine_and_terms (list bc value_condition_expr))
									bg bh bo bl boff))
							original_expr))
					original_expr)))
				(define direct_scalar_exists (lambda (inner_expr original_expr)
					(match inner_expr
						(cons inner_sym inner_args)
						(if (equal?? (inner_select_kind inner_sym) (quote inner_select))
							(match inner_args
								(cons bool_subquery '())
								(scalar_value_exists bool_subquery (quote truthy) original_expr)
								original_expr)
							original_expr)
						original_expr)))
				(define non_nil_scalar_exists (lambda (inner_expr original_expr)
					(match inner_expr
						(cons inner_sym inner_args)
						(if (equal?? (inner_select_kind inner_sym) (quote inner_select))
							(match inner_args
								(cons bool_subquery '())
								(scalar_value_exists bool_subquery (quote not-nil) original_expr)
								original_expr)
							original_expr)
						original_expr)))
				(match expr
					(cons sym args) (if (not_symbol sym)
						(match args
							(cons inner_expr '())
							(match inner_expr
								(cons nil_sym nil_args)
								(if (nil_symbol? nil_sym)
									(match nil_args
										(cons maybe_scalar '())
										(begin
											(define exists_expr (non_nil_scalar_exists maybe_scalar expr))
											(if (equal? exists_expr expr)
												(list sym (boolean_scalar_inner_selects_to_exists inner_expr))
												exists_expr))
										(list sym (boolean_scalar_inner_selects_to_exists inner_expr)))
									(list sym (boolean_scalar_inner_selects_to_exists inner_expr)))
								(list sym (boolean_scalar_inner_selects_to_exists inner_expr)))
							(cons sym (map args boolean_scalar_inner_selects_to_exists)))
						(if (nil_symbol? sym)
							(match args
								(cons maybe_scalar '())
								(begin
									(define exists_expr (non_nil_scalar_exists maybe_scalar expr))
									(if (equal? exists_expr expr)
										(cons sym (map args boolean_scalar_inner_selects_to_exists))
										(list (quote not) exists_expr)))
								(cons sym (map args boolean_scalar_inner_selects_to_exists)))
							(begin
								(define kind (inner_select_kind sym))
								(if (equal?? kind (quote inner_select))
									(direct_scalar_exists expr expr)
									(cons sym (map args boolean_scalar_inner_selects_to_exists)))))))
				expr)
			))
			(define condition2_exists (boolean_scalar_inner_selects_to_exists condition2))
			(define hoisted_expr
				(if (or (equal? guard_parts '()) (equal? local_parts condition_parts))
					nil
					(begin
						(define local_condition (combine_and_terms local_parts))
						(define guard_condition (combine_and_terms guard_parts))
						(if (not (_contains_inner_select_marker guard_condition))
							nil
							(begin
								(define local_exists (_unnest_count_subselect
									(list schema2 tables2 fields2 local_condition group2 having2 order2 limit2 offset2)
									outer_schemas nil (quote >)))
								(if (nil? local_exists)
									nil
									(list (quote and)
										(replace_guard_inner_selects guard_condition)
										local_exists)))))))
			(coalesce hoisted_expr
				(_unnest_count_subselect
					(list schema2 tables2
						(list "__exists" true)
						condition2_exists group2 having2 order2 (coalesceNil limit2 1) offset2)
					outer_schemas nil (quote >))
				(match (build_scalar_subselect_with_strategy
					(list schema2 tables2
						(list "__exists" true)
						condition2_exists group2 having2 order2 (coalesceNil limit2 1) offset2)
					outer_schemas)
					'(_ lowered_exists_expr)
					(list (quote coalesceNil) lowered_exists_expr false)
					(list (quote coalesceNil)
						(build_scalar_subselect_inline
							(list schema2 tables2
								(list "__exists" true)
								condition2_exists group2 having2 order2 (coalesceNil limit2 1) offset2)
							outer_schemas)
						false))))
		false
	)))

	/* unnest_subselect: core Neumann decorrelation for a single subquery.
	Transforms a correlated scalar subquery into a LEFT JOIN table entry,
	eliminating the dependent join. Returns (substitution tables) or nil.

	Three paths based on subquery shape:
	Path A (aggregate): adds domain columns to GROUP BY (Neumann Γ_{A∪D}),
	flattens inner tables with scoped GROUP-stage. Handles COUNT/SUM/AVG/etc.
	Path B/C (non-agg): attaches a scan tag to the direct LEFT JOIN helper table.
	build_scan lowers that tag into scan/scan_order with scalar once-limit semantics.

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
				/* pass outer_schemas chain to recursive untangle so grandparent refs resolve.
				Use the shared logical scope preparation so inline and unnest paths stay on
				the same recursive normalization boundary. */
				(match (untangle_scalar_subquery_scope
					subquery outer_schemas
					raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us)
					'(schema2_us tables2_us fields2_us condition2_us groups2_us schemas2_us rfcol2_us _init2_us) (begin
						(if (and (not (nil? _init2_us)) (not (equal? _init2_us '())))
							(scalar_subquery_cache "init" (merge (coalesceNil (scalar_subquery_cache "init") '()) _init2_us)))
						/* no-table subselect without aggregates: return field expression directly */
						(if (and (or (nil? tables2_us) (equal? tables2_us '()))
							(not (reduce_assoc fields2_us (lambda (a k v) (or a
								(begin (define _nta (lambda (e) (match e (cons (symbol aggregate) _) true (cons s args) (reduce args (lambda (a2 b) (or a2 (_nta b))) false) false))) (_nta v)))) false)))
							(list (car (extract_assoc fields2_us (lambda (k v) v))) '())
							(begin
								/* no-table with aggregates: inject virtual "(1)" one-row table.
								Only mutate tables2_us and schemas2_us — groups2_us is set below. */
								(define _nt_virtual_init (list (quote begin)
									(list (quote createtable) schema2_us "(1)"
										(list (list "unique" "group" (list "1")) (list "column" "1" "any" '() '()))
										(list "engine" "sloppy") true)
									(list (quote insert) (list (quote table) schema2_us "(1)") (list "1") (list (list 1)) '() (list (quote lambda) '() true) true)))
								(if (or (nil? tables2_us) (equal? tables2_us '()))
									(begin
										(set tables2_us (list (list "(1)" schema2_us "(1)" false nil)))
										(set schemas2_us (list "(1)" (list (list "Field" "1" "Type" "any"))))))
								(define groups2_us (coalesceNil groups2_us '()))
								(define groups2_us (if (or (nil? groups2_us) (equal? groups2_us '()))
									(if (or raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us)
										(list (make_group_stage raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us nil _nt_virtual_init))
										groups2_us)
									groups2_us))
								(define _us_has_field_aggregate (reduce_assoc fields2_us (lambda (found _k v)
									(or found (not (equal? (extract_aggregates v) '())))) false))
								/* resolve columns against inner and outer schemas */
								(define rfcs_us (make_replace_find_column_subselect schemas2_us outer_schemas
									(or
										_us_has_field_aggregate
										raw_group_us raw_having_us raw_order_us
										(_raw_query_uses_alias_outside_current subquery (_raw_query_local_aliases subquery)))))
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
								(define us_inner_aliases (map tables2_us (lambda (td) (match td '(a _ _ _ _) a ""))))
								(define us_outer_refs (merge_unique
									(merge (extract_assoc fields2_us (lambda (k v) (unnest_expr_outer_refs v us_inner_aliases))))
									(unnest_expr_outer_refs condition2_us us_inner_aliases)))
								/* feasibility checks */
								(define us_has_outer (not (equal? us_outer_refs '())))
								/* separate own stages from inner scoped stages (from nested decorrelation) —
								must be defined BEFORE _us_inner_aliases which depends on _us_inner_stages */
								(define _us_own_stages (filter (coalesceNil groups2_us '()) (lambda (s) (nil? (stage_partition_aliases s)))))
								(define _us_inner_stages (filter (coalesceNil groups2_us '()) (lambda (s) (not (nil? (stage_partition_aliases s))))))
								/* count only OWN tables (not inner scoped ones from nested decorrelation) */
								(define _us_inner_aliases (merge (map _us_inner_stages (lambda (s) (coalesceNil (stage_partition_aliases s) '())))))
								(define _us_own_tables (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (not (has? _us_inner_aliases a)) true))))
								(define _us_generated_unnest_alias (lambda (alias_name)
									(and (not (nil? alias_name))
										(begin
											(define alias_text (string alias_name))
											(and
												(>= (strlen alias_text) 14)
												(equal? (substr alias_text 0 14) "domain_scalar_"))))))
								(define _us_base_tables (filter _us_own_tables (lambda (t) (match t
									'(a _ _ _ _) (not (_us_generated_unnest_alias a))
									true))))
								(define _us_nested_direct_tbls (filter _us_own_tables (lambda (t) (match t
									'(a _ _ _ _) (_us_generated_unnest_alias a)
									false))))
								(define _us_base_aliases (map _us_base_tables (lambda (td) (match td '(a _ _ _ _) a nil))))
								(define us_single_tbl (and (list? _us_base_tables) (equal? (count _us_base_tables) 1)))
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
								(define us_simple_agg_stages (or (equal? _us_own_stages '())
									(and (equal? (count _us_own_stages) 1)
										(not (stage_is_dedup (car _us_own_stages))))))
								/* check for outer refs in fields (not just condition) — these need
								more complex handling, fall back for now */
								(define us_outer_in_fields (not (equal?
									(merge (extract_assoc fields2_us (lambda (k v) (unnest_expr_outer_refs v us_inner_aliases)))) '())))
								(if us_outer_in_fields nil /* outer refs in fields: not handled yet */
									(begin
										/* === Neumann unnesting: nD domain, single or multi-table === */
										/* generate unique alias using fnv_hash to avoid collisions across nesting levels */
										(define us_scalar_subquery_idx (coalesceNil (scalar_subquery_cache "idx") 0))
										(scalar_subquery_cache "idx" (+ us_scalar_subquery_idx 1))
										(define us_scalar_subquery_prefix (concat "domain_scalar_" us_scalar_subquery_idx))
										/* build alias rename map: only OWN tables get prefixed.
										Inner-scoped tables (from nested decorrelation) keep their alias. */
										(define us_alias_map (scalar_subselect_alias_map _us_base_tables us_single_tbl us_scalar_subquery_prefix))
										(define _us_lookup (lambda (a) (scalar_subselect_lookup_alias us_alias_map a)))
										(define _us_local_aliases_raw (_raw_query_local_aliases subquery))
										/* scalar subqueries still expose a single projected value expr */
										(define us_value_expr (car (extract_assoc fields2_us (lambda (k v) v))))
										(define _us_ror unnest_runtime_outer_ref_expr)
										(define _us_ria (lambda (expr) (unnest_rewrite_inner_aliases expr _us_lookup)))
										(define _us_rewrite_current_base_refs (lambda (expr) (match expr
											'((symbol get_column) alias_ ti col ci) (if (and (has? _us_base_aliases alias_) (has? _us_local_aliases_raw alias_))
												(begin
													(define na (_us_lookup alias_))
													(if (nil? na) expr (list (quote get_column) na false col false)))
												expr)
											'((quote get_column) alias_ ti col ci) (if (and (has? _us_base_aliases alias_) (has? _us_local_aliases_raw alias_))
												(begin
													(define na (_us_lookup alias_))
													(if (nil? na) expr (list (quote get_column) na false col false)))
												expr)
											(cons sym args) (cons (_us_rewrite_current_base_refs sym) (map args _us_rewrite_current_base_refs))
											expr)))
										(match (scalar_subselect_correlation_info condition2_us us_inner_aliases _us_ror)
											'(us_outer_parts us_domain_cols us_inner_cond_raw)
											(begin
												(define us_build_aggregate_path (lambda () (begin
													/* === A: Aggregate → flatten inner tables + scoped GROUP stage ===
													Neumann Γ_{A∪D;f}: add domain cols to GROUP BY, flatten inner tables
													with prefix into outer table list. No materialization. */
													(define _us_prefix_ria (lambda (expr)
														(scalar_subselect_rewrite_prefixed_expr expr _us_lookup)))
													(define us_prefixed_tables_raw (scalar_subselect_prefixed_tables _us_base_tables _us_lookup _us_prefix_ria))
													(define us_prefixed_tables (match us_prefixed_tables_raw
														(cons first_tbl rest_tbls)
														(cons (match first_tbl
															'(a s t _ je) (list a s t true je)
															first_tbl) rest_tbls)
														us_prefixed_tables_raw))
													(define _us_inner_tbls_for_group
														(filter tables2_us (lambda (td) (match td
															'(a _ _ _ _) (has? _us_inner_aliases a)
															false))))
													(define _us_localize_outer_refs (lambda (expr) (match expr
														'((symbol outer) symname) (begin
															(define parts (split (string symname) "."))
															(match parts
																'(alias_ col) (begin
																	(define na (_us_lookup alias_))
																	(if (nil? na)
																		(list (quote outer) (list (quote outer) (list (quote get_column) alias_ false col false)))
																		(list (quote get_column) na false col false)))
																_ expr))
														'((quote outer) symname) (begin
															(define parts (split (string symname) "."))
															(match parts
																'(alias_ col) (begin
																	(define na (_us_lookup alias_))
																	(if (nil? na)
																		(list (quote outer) (list (quote outer) (list (quote get_column) alias_ false col false)))
																		(list (quote get_column) na false col false)))
																_ expr))
														(cons sym args) (cons sym (map args _us_localize_outer_refs))
														expr)))
													(define _us_rewrite_inner_expr_for_group (lambda (expr)
														(_us_localize_outer_refs (_us_ria expr))))
													(define _us_inner_tbls_rewritten_for_group
														(scalar_subselect_rewrite_tables _us_inner_tbls_for_group _us_rewrite_inner_expr_for_group))
													(define _us_nested_direct_tbls_rewritten_for_group
														(scalar_subselect_rewrite_tables _us_nested_direct_tbls _us_rewrite_inner_expr_for_group))
													(define us_inner_cond_prefixed (if (nil? us_inner_cond_raw) nil
														(_us_localize_outer_refs (_us_prefix_ria us_inner_cond_raw))))
													(define us_orig_group (if us_has_stages (coalesceNil (stage_group_cols (car _us_own_stages)) '()) '()))
													(define us_orig_having (if us_has_stages (stage_having_expr (car _us_own_stages)) nil))
													(define us_cache_policy (count_subquery_cache_policy subquery target_expr))
													(define us_nested_domain_cols (reduce _us_inner_stages (lambda (acc s)
														(merge acc
															(filter (map (coalesceNil (stage_outer_sources s) '()) (lambda (src)
																(match src
																	'(outer_tv outer_col inner_expr)
																	(if (stage_outer_source_expr_tuple? src)
																		(list (nth src 2) (nth src 1))
																		(list inner_expr (list (quote get_column) outer_tv false outer_col false)))
																	_ nil)))
																(lambda (x) (not (nil? x)))))) '()))
													(define _us_direct_col (lambda (expr) (match expr
														'((symbol get_column) tv _ col _) (list tv col expr)
														'((quote get_column) tv _ col _) (list tv col expr)
														nil)))
													(define _us_scalar_match_inner_expr (lambda (domain_cols)
														(reduce (coalesceNil domain_cols '()) (lambda (found dc)
															(if (not (nil? found))
																found
																(begin
																	(define inner_info (_us_direct_col (nth dc 0)))
																	(if (and
																		(not (nil? inner_info))
																		(has? _us_base_aliases (nth inner_info 0)))
																		(nth dc 0)
																		nil))))
															nil)))
													(define _us_wrap_scalar_no_match_nil (lambda (subst domain_cols rewrite_fn)
														(begin
															(define match_inner_expr (_us_scalar_match_inner_expr domain_cols))
															(if (or
																(nil? match_inner_expr)
																(and us_has_agg (not us_has_grp)))
																subst
																(list (quote if)
																	(list (quote not) (list (quote nil?) (rewrite_fn match_inner_expr)))
																	subst
																	nil)))))
													(define _us_nested_limit_any_shape
														(and (not us_has_agg)
															(not us_has_grp)
															(or
																(not (equal? _us_inner_stages '()))
																(not (equal? _us_nested_direct_tbls '())))
															(not (nil? raw_limit_us))
															(<= raw_limit_us 1)
															(or (nil? raw_offset_us) (equal? raw_offset_us 0))
															(equal? (coalesceNil raw_order_us '()) '())))
													(define _us_nested_helper_domain_cols
														(if _us_nested_limit_any_shape
															(reduce (qpu-and-conjuncts us_inner_cond_raw) (lambda (acc part) (begin
																(define _eq_args (match part
																	'((symbol equal??) a b) (list a b)
																	'((quote equal??) a b) (list a b)
																	'((symbol =) a b) (list a b)
																	'((quote =) a b) (list a b)
																	nil))
																(if (nil? _eq_args) acc
																	(begin
																		(define _li (_us_direct_col (nth _eq_args 0)))
																		(define _ri (_us_direct_col (nth _eq_args 1)))
																		(define _domain_pair
																			(if (and (not (nil? _li)) (has? _us_base_aliases (nth _li 0))
																				(not (nil? _ri)) (_us_generated_unnest_alias (nth _ri 0)))
																				(list (nth _li 2) (nth _ri 2))
																				(if (and (not (nil? _ri)) (has? _us_base_aliases (nth _ri 0))
																					(not (nil? _li)) (_us_generated_unnest_alias (nth _li 0)))
																					(list (nth _ri 2) (nth _li 2))
																					nil)))
																		(if (nil? _domain_pair) acc (merge acc (list _domain_pair))))))) '())
															'()))
													(define _us_nested_direct_expr_refs_generated? (lambda (expr) (match expr
														'((symbol get_column) tv _ _ _) (_us_generated_unnest_alias tv)
														'((quote get_column) tv _ _ _) (_us_generated_unnest_alias tv)
														(cons sym args)
														(or (_us_nested_direct_expr_refs_generated? sym)
															(reduce args (lambda (found arg)
																(or found (_us_nested_direct_expr_refs_generated? arg)))
																false))
														false)))
													(define _us_nested_direct_original_joinexpr (lambda (td) (match td
														'(alias_ _ _ _ joinexpr_)
														(coalesceNil
															(get_assoc (coalesceNil (scalar_subquery_cache "scalar_table_original_joinexprs") '()) alias_)
															joinexpr_)
														nil)))
													(define _us_expr_ref_aliases (lambda (expr) (match expr
														'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
														'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
														'((symbol outer) symname) (if (list? symname)
															(_us_expr_ref_aliases symname)
															(match (split (string symname) ".")
																'(tv _col) (list tv)
																_ '()))
														'((quote outer) symname) (if (list? symname)
															(_us_expr_ref_aliases symname)
															(match (split (string symname) ".")
																'(tv _col) (list tv)
																_ '()))
														(cons sym args)
														(merge_unique (list
															(_us_expr_ref_aliases sym)
															(map args _us_expr_ref_aliases)))
														'())))
													(define _us_group_table_cache_key (if (not us_has_agg) "scalar_tables" "tables"))
													(define _us_cached_nested_domain_tbls
														(filter (merge
															(coalesceNil (scalar_subquery_cache _us_group_table_cache_key) '())
															(coalesceNil (scalar_subquery_cache "scalar_tables") '()))
															(lambda (td) (match td
																'(alias_ _ _ _ _)
																(and
																	(_us_generated_unnest_alias alias_)
																	(reduce (_us_expr_ref_aliases (_us_nested_direct_original_joinexpr td)) (lambda (found ref_alias)
																		(or found (has? _us_base_aliases ref_alias)))
																		false))
																false))))
													(define _us_cached_nested_domain_aliases
														(map _us_cached_nested_domain_tbls (lambda (td) (match td
															'(alias_ _ _ _ _) alias_
															nil))))
													(define _us_nested_direct_domain_cols
														(reduce (merge _us_inner_tbls_for_group _us_nested_direct_tbls _us_cached_nested_domain_tbls) (lambda (acc td)
															(reduce (qpu-and-conjuncts (_us_nested_direct_original_joinexpr td)) (lambda (acc2 part) (begin
																(define _eq_args (match part
																	'((symbol equal??) a b) (list a b)
																	'((quote equal??) a b) (list a b)
																	'((symbol =) a b) (list a b)
																	'((quote =) a b) (list a b)
																	nil))
																(if (nil? _eq_args) acc2
																	(begin
																		(define _li (_us_direct_col (nth _eq_args 0)))
																		(define _ri (_us_direct_col (nth _eq_args 1)))
																		(define _domain_pair
																			(if (and (not (nil? _li))
																				(_us_generated_unnest_alias (nth _li 0))
																				(not (_us_nested_direct_expr_refs_generated? (nth _eq_args 1))))
																				(list (nth _li 2) (nth _eq_args 1))
																				(if (and (not (nil? _ri))
																					(_us_generated_unnest_alias (nth _ri 0))
																					(not (_us_nested_direct_expr_refs_generated? (nth _eq_args 0))))
																					(list (nth _ri 2) (nth _eq_args 0))
																					nil)))
																		(if (nil? _domain_pair) acc2 (merge acc2 (list _domain_pair)))))))
																acc))
															'()))
													(define _us_nested_domain_inner_exprs
														(map us_nested_domain_cols (lambda (dc) (nth dc 0))))
													(define _us_nested_direct_domain_cols_effective
														(filter _us_nested_direct_domain_cols (lambda (dc)
															(not (reduce _us_nested_domain_inner_exprs (lambda (found inner_expr)
																(or found (equal? inner_expr (nth dc 0))))
																false)))))
													(define us_domain_cols_all (reduce (merge us_domain_cols us_nested_domain_cols _us_nested_helper_domain_cols _us_nested_direct_domain_cols_effective) (lambda (acc dc)
														(if (reduce acc (lambda (found existing) (or found (equal? existing dc))) false)
															acc
															(merge acc (list dc)))) '()))
													(define _us_dom_group_cols (map us_domain_cols_all (lambda (dc) (_us_prefix_ria (nth dc 0)))))
													/* A complex scalar LIMIT 1 value is payload, not domain. Carry it
													through the normal grouped-value aggregate path; adding a boolean/CASE
													filter value to GROUP BY turns it into a reattach key and breaks WHERE
													scalar helpers after materialization. Direct column values are still
													stable domain keys and are needed by existing NULL-or-equals scalar
													COUNT patterns. */
													(define _us_value_group_cols
														(if (and _us_nested_limit_any_shape (not (nil? (_us_direct_col us_value_expr))))
															(list (_us_prefix_ria us_value_expr))
															'()))
													(define us_prefixed_aliases (scalar_subselect_table_aliases us_prefixed_tables))
													(define us_new_group (merge _us_dom_group_cols _us_value_group_cols
														(if (or (equal? us_orig_group '()) (equal? us_orig_group '(1)))
															(if (equal? _us_dom_group_cols '()) us_orig_group '())
															(map us_orig_group _us_prefix_ria))))
													(define us_new_having (if (nil? us_orig_having) nil (_us_prefix_ria us_orig_having)))
													(define _us_nested_direct_aliases
														(map _us_nested_direct_tbls (lambda (td) (match td '(a _ _ _ _) a nil))))
													(define us_stage_aliases
														(if (equal? _us_dom_group_cols '()) nil
															(merge_unique (list
																_us_inner_aliases
																_us_nested_direct_aliases
																_us_cached_nested_domain_aliases
																us_prefixed_aliases))))
													(define _us_nested_domain_joinexpr_for_alias (lambda (alias_) (begin
														(define parts
															(filter (map us_nested_domain_cols (lambda (dc) (begin
																(define inner_info (_us_direct_col (nth dc 0)))
																(if (and (not (nil? inner_info))
																	(equal? (nth inner_info 0) alias_))
																	(list (quote equal??) (nth dc 0) (nth dc 1))
																	nil))))
																(lambda (p) (not (nil? p)))))
														(combine_and_terms parts))))
													(define us_orig_order_a (if (and us_has_grp us_has_stages) (coalesceNil (stage_order_list (car _us_own_stages)) '()) '()))
													(define us_orig_limit_a (if (and us_has_grp us_has_stages) (stage_limit_val (car _us_own_stages)) nil))
													(define us_orig_offset_a (if (and us_has_grp us_has_stages) (stage_offset_val (car _us_own_stages)) nil))
													(define us_new_order (map us_orig_order_a (lambda (oi) (match oi '(col dir) (list (_us_prefix_ria col) dir) oi))))
													(define us_group_stage_base
														(if (and (equal? us_cache_policy (quote uncached-count))
															(not (nil? us_inner_cond_prefixed)))
															(make_group_stage_with_condition us_new_group us_new_having us_new_order us_orig_limit_a us_orig_offset_a us_stage_aliases nil us_inner_cond_prefixed)
															(make_group_stage us_new_group us_new_having us_new_order us_orig_limit_a us_orig_offset_a us_stage_aliases nil)))
													(define us_group_outer_sources (domain_outer_sources_from_correlation_cols us_domain_cols_all _us_prefix_ria))
													(define us_group_stage
														(stage_with_outer_sources
															(stage_with_cache_query
																(stage_with_cache_policy
																	us_group_stage_base
																	us_cache_policy)
																(if (nil? us_cache_policy) nil subquery))
															us_group_outer_sources))
													(define _us_prefixed_inner_stages (scalar_subselect_rewrite_stages_with_lookup
														_us_inner_stages
														_us_prefix_ria
														_us_lookup))
													(define _us_group_tables (merge
														_us_inner_tbls_rewritten_for_group
														_us_nested_direct_tbls_rewritten_for_group
														_us_cached_nested_domain_tbls
														us_prefixed_tables))
													(define _us_group_tables
														(map _us_group_tables (lambda (td) (match td
															'(alias_ schema_ table_ is_outer_ joinexpr_)
															(begin
																(define nested_joinexpr
																	(_us_nested_domain_joinexpr_for_alias alias_))
																(if (or (nil? nested_joinexpr)
																	(equal? nested_joinexpr true))
																	td
																	(list alias_ schema_ table_ is_outer_ nested_joinexpr)))
															td))))
													(if (not (equal? _us_cached_nested_domain_aliases '()))
														(scalar_subquery_cache "scalar_tables"
															(filter (coalesceNil (scalar_subquery_cache "scalar_tables") '())
																(lambda (td) (match td
																	'(alias_ _ _ _ _) (not (has? _us_cached_nested_domain_aliases alias_))
																	true)))))
													(define _us_cached_table_refs_current_base? (lambda (td)
														(match td
															'(alias_ _ _ _ joinexpr_)
															(and
																(_us_generated_unnest_alias alias_)
																(not (nil? joinexpr_))
																(reduce (_us_expr_ref_aliases joinexpr_) (lambda (found ref_alias)
																	(or found (has? _us_base_aliases ref_alias)))
																	false))
															false)))
													(define _us_rewrite_cached_nested_table_joinrefs (lambda (td)
														(if (_us_cached_table_refs_current_base? td)
															(match td
																'(alias_ schema_ table_ is_outer_ joinexpr_)
																(begin
																	(define original_joinexpr (coalesceNil
																		(get_assoc (coalesceNil (scalar_subquery_cache "scalar_table_original_joinexprs") '()) alias_)
																		joinexpr_))
																	(define rewritten_joinexpr (_us_rewrite_current_base_refs original_joinexpr))
																	(list alias_ schema_ table_ is_outer_ rewritten_joinexpr))
																td)
															td)))
													(scalar_subquery_cache _us_group_table_cache_key
														(map (coalesceNil (scalar_subquery_cache _us_group_table_cache_key) '())
															_us_rewrite_cached_nested_table_joinrefs))
													(if (not us_has_agg)
														(scalar_subquery_cache "scalar_tables" (merge
															_us_group_tables
															(coalesceNil (scalar_subquery_cache "scalar_tables") '())))
														(scalar_subquery_cache "tables" (merge
															_us_group_tables
															(coalesceNil (scalar_subquery_cache "tables") '()))))
													(define us_prefixed_schemas (scalar_subselect_prefixed_schemas us_prefixed_tables us_alias_map schemas2_us))
													(define _us_passthrough_schemas_for_group
														(scalar_subselect_passthrough_schemas
															(merge _us_inner_tbls_for_group _us_nested_direct_tbls)
															schemas2_us))
													(scalar_subquery_cache "schemas" (merge
														us_prefixed_schemas
														_us_passthrough_schemas_for_group
														(coalesceNil (scalar_subquery_cache "schemas") '())))
													(define us_dom_je_parts (map us_domain_cols_all (lambda (dc)
														(list (quote equal??) (_us_prefix_ria (nth dc 0)) (nth dc 1)))))
													(define us_dom_je_refs_generated_helper? (lambda (part)
														(reduce (extract_tblvars part) (lambda (found tv)
															(or found (_us_generated_unnest_alias tv))) false)))
													(define us_dom_je_nested_parts
														(filter us_dom_je_parts us_dom_je_refs_generated_helper?))
													(define us_dom_je_outer_parts
														(filter us_dom_je_parts (lambda (part)
															(not (us_dom_je_refs_generated_helper? part)))))
													(define us_dom_je (if (equal? (count us_dom_je_outer_parts) 0) true
														(if (equal? (count us_dom_je_outer_parts) 1) (car us_dom_je_outer_parts)
															(cons (quote and) us_dom_je_outer_parts))))
													(define _us_inner_parts_list (merge
														us_dom_je_nested_parts
														(if (nil? us_inner_cond_prefixed) '()
															(match us_inner_cond_prefixed
																(cons (symbol and) parts) parts
																(cons (quote and) parts) parts
																(list us_inner_cond_prefixed)))))
													(define _us_expr_refs (lambda (expr) (match expr
														'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
														'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
														(cons _ args) (reduce args (lambda (acc a) (merge acc (_us_expr_refs a))) '())
														_ (if (or (nil? expr) (string? expr) (list? expr))
															'()
															(begin
																(define _atom_parts (split (string expr) "."))
																(if (> (count _atom_parts) 1) (list (car _atom_parts)) '()))))))
													(define _us_generated_helper_only_part? (lambda (part) (begin
														(define _refs (_us_expr_refs part))
														(and
															(not (equal? _refs '()))
															(reduce _refs (lambda (ok tv)
																(and ok (_us_generated_unnest_alias tv)))
																true)))))
													(define _us_stage_filter_parts
														(filter _us_inner_parts_list _us_generated_helper_only_part?))
													(define _us_join_parts_list
														(filter _us_inner_parts_list (lambda (part)
															(not (_us_generated_helper_only_part? part)))))
													(define _us_stage_filter_condition (combine_and_terms _us_stage_filter_parts))
													(if (not (equal? _us_stage_filter_condition true))
														(set us_group_stage
															(stage_rebuild_with_meta
																us_group_stage
																(make_group_stage_with_condition
																	us_new_group us_new_having us_new_order
																	us_orig_limit_a us_orig_offset_a
																	us_stage_aliases nil
																	(combine_and_terms (list (stage_condition us_group_stage) _us_stage_filter_condition)))
																(lambda (expr) expr)
																(lambda (alias_) alias_))))
													(define _us_last_alias (lambda (part) (begin
														(define _refs (_us_expr_refs part))
														(reduce us_prefixed_aliases (lambda (best al)
															(if (reduce _refs (lambda (found r) (or found (equal?? r al))) false)
																al best)) nil))))
													(define _us_parts_for (lambda (alias) (begin
														(define _my (filter _us_join_parts_list (lambda (p) (equal?? (_us_last_alias p) alias))))
														(if (equal? (count _my) 0) nil
															(if (equal? (count _my) 1) (car _my)
																(cons (quote and) _my))))))
													(if (not (nil? us_prefixed_tables))
														(scalar_subquery_cache _us_group_table_cache_key (begin
															(define _all_tbls (scalar_subquery_cache _us_group_table_cache_key))
															(define _first_alias (match (car us_prefixed_tables) '(a _ _ _ _) a ""))
															(map _all_tbls (lambda (td) (match td
																'(a s t io je) (if (not (reduce us_prefixed_aliases (lambda (f al) (or f (equal?? al a))) false)) td
																	(begin
																		(define _my_cond (_us_parts_for a))
																		(if (equal? a _first_alias)
																			(list a s t true (if (nil? _my_cond) us_dom_je
																				(if (equal? us_dom_je true) _my_cond
																					(list (quote and) us_dom_je _my_cond))))
																			(list a s t io (if (nil? _my_cond) je
																				(if (nil? je) _my_cond
																					(list (quote and) je _my_cond)))))))
																td))))))
													(scalar_subquery_cache "groups" (merge
														(list us_group_stage)
														_us_prefixed_inner_stages
														(coalesceNil (scalar_subquery_cache "groups") '())))
													(define us_subst_raw (_us_prefix_ria us_value_expr))
													(define us_is_count (match us_value_expr
														'((symbol aggregate) _ (symbol +) 0) true
														'((quote aggregate) _ (symbol +) 0) true
														'((quote aggregate) _ '(symbol +) 0) true
														false))
													(define us_positive_count_comparison? (lambda (expr) (match expr
														'(op left right)
														(and
															(or
																(equal? op (quote >))
																(equal? op (symbol >))
																(equal? op '(quote >)))
															(or (equal? right 0) (equal? right 0.0))
															(match left
																'((symbol aggregate) _ (symbol +) 0) true
																'((quote aggregate) _ (symbol +) 0) true
																'(coalesce_op inner fallback)
																(and
																	(or
																		(equal? coalesce_op (quote coalesceNil))
																		(equal? coalesce_op (symbol coalesceNil))
																		(equal? coalesce_op '(quote coalesceNil)))
																	(or (equal? fallback 0) (equal? fallback 0.0))
																	(us_positive_count_comparison? (list (quote >) inner 0)))
																false))
														false)))
													(define us_count_zero_on_empty (and us_is_count (not us_has_grp)))
													(define us_count_bool_false_on_empty (and (not us_has_grp) (us_positive_count_comparison? us_value_expr)))
													(define us_subst
														(if us_count_zero_on_empty
															(list (quote coalesceNil) us_subst_raw 0)
															(if us_count_bool_false_on_empty
																(list (quote coalesceNil) us_subst_raw false)
																(_us_wrap_scalar_no_match_nil us_subst_raw us_domain_cols_all _us_prefix_ria))))
													(list us_subst '())
												)))
												(define us_build_scalar_scan_path (lambda () (begin
													(define _us_nested_direct_refs_base_aliases (reduce _us_nested_direct_tbls (lambda (acc td) (match td
														'(_ _ _ _ je) (or acc
															(and (not (nil? je))
																(reduce (extract_tblvars je) (lambda (found tv)
																	(or found (has? _us_base_aliases tv))) false)))
														_ acc))
														false))
													(if (and us_single_tbl
														(equal? _us_nested_direct_tbls '())
														(equal? _us_inner_stages '())
														(not _us_nested_direct_refs_base_aliases))
														(begin
															(define us_tdesc (car _us_base_tables))
															(define us_tblvar (nth us_tdesc 0))
															(define us_tbl_schema (nth us_tdesc 1))
															(define us_tbl_name (nth us_tdesc 2))
															(define us_orig_order (if us_has_stages (coalesceNil (stage_order_list (car _us_own_stages)) '()) '()))
															(define us_orig_limit (if us_has_stages (stage_limit_val (car _us_own_stages)) nil))
															(define us_orig_offset (if us_has_stages (stage_offset_val (car _us_own_stages)) nil))
															(define _us_inner_tbls (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (has? _us_inner_aliases a) false))))
															(define _us_inner_tbls_rewritten (scalar_subselect_rewrite_tables _us_inner_tbls _us_ria))
															(define us_simple_uncorrelated_cache_key (if (and
																(not us_has_outer)
																(equal? _us_inner_tbls '())
																(equal? _us_inner_stages '()))
																(serialize subquery)
																nil))
															(define us_cached_subst (if (nil? us_simple_uncorrelated_cache_key)
																nil
																(get_assoc (coalesceNil (scalar_subquery_cache "scalar_helper_cache") '()) us_simple_uncorrelated_cache_key)))
															(if (not (nil? us_cached_subst))
																(list us_cached_subst '())
																(begin
																	(if (not (equal? _us_inner_tbls_rewritten '()))
																		(scalar_subquery_cache "tables" (merge _us_inner_tbls_rewritten (coalesceNil (scalar_subquery_cache "tables") '()))))
																	(define us_dom_order (scalar_scan_domain_order us_domain_cols _us_ria us_scalar_subquery_prefix))
																	(define us_renamed_order (scalar_scan_rewrite_order us_orig_order _us_ria))
																	(define us_order_supported (scalar_scan_order_supported us_renamed_order us_scalar_subquery_prefix))
																	(if (not us_order_supported)
																		nil
																		(begin
																			(define us_part_order (merge us_dom_order us_renamed_order))
																			(define us_dom_count (count us_dom_order))
																			(define us_outer_sources (domain_outer_sources_from_correlation_cols us_domain_cols _us_ria))
																			(define _us_inner_stages_rewritten (scalar_subselect_rewrite_stages_with_lookup
																				_us_inner_stages
																				_us_ria
																				_us_lookup))
																			(define _us_nested_outer_sources (scalar_subselect_collect_stage_outer_sources _us_inner_stages_rewritten))
																			(define us_part_stage (make_scalar_partition_stage
																				us_part_order
																				us_orig_limit
																				us_orig_offset
																				us_dom_count
																				(list us_scalar_subquery_prefix)
																				(merge_unique (list us_outer_sources _us_nested_outer_sources))))
																			(scalar_subquery_cache "groups" (merge
																				(list us_part_stage)
																				_us_inner_stages_rewritten
																				(coalesceNil (scalar_subquery_cache "groups") '())))
																			(define us_join_lim (map us_outer_parts (lambda (p) (_us_ria p))))
																			(define us_inner_lim (_us_ria us_inner_cond_raw))
																			(define us_full_lim (if (nil? us_inner_lim)
																				(if (equal? (count us_join_lim) 0) true (if (equal? (count us_join_lim) 1) (car us_join_lim) (cons (quote and) us_join_lim)))
																				(cons (quote and) (merge us_join_lim (list us_inner_lim)))))
																			(define _us_localize_outer_refs_for_scan (lambda (expr) (match expr
																				'((symbol outer) symname) (begin
																					(define parts (split (string symname) "."))
																					(match parts
																						'(alias_ col) (begin
																							(define na (_us_lookup alias_))
																							(if (nil? na)
																								(list (quote outer) (list (quote outer) (list (quote get_column) alias_ false col false)))
																								(list (quote get_column) na false col false)))
																						_ expr))
																				'((quote outer) symname) (begin
																					(define parts (split (string symname) "."))
																					(match parts
																						'(alias_ col) (begin
																							(define na (_us_lookup alias_))
																							(if (nil? na)
																								(list (quote outer) (list (quote outer) (list (quote get_column) alias_ false col false)))
																								(list (quote get_column) na false col false)))
																						_ expr))
																				(cons sym args) (cons sym (map args _us_localize_outer_refs_for_scan))
																				expr)))
																			(define _us_nested_direct_tbls_rewritten (scalar_subselect_rewrite_tables _us_nested_direct_tbls (lambda (expr)
																				(_us_localize_outer_refs_for_scan (_us_ria expr)))))
																			(define us_tbl_entries (merge _us_nested_direct_tbls_rewritten (list (list us_scalar_subquery_prefix us_tbl_schema us_tbl_name true us_full_lim))))
																			(define _us_inner_schema (schemas2_us us_tblvar))
																			(define us_subst_raw (_us_ria us_value_expr))
																			(define _us_scan_direct_col (lambda (expr) (match expr
																				'((symbol get_column) tv _ col _) (list tv col expr)
																				'((quote get_column) tv _ col _) (list tv col expr)
																				nil)))
																			(define _us_scan_match_inner_expr
																				(reduce (coalesceNil us_domain_cols '()) (lambda (found dc)
																					(if (not (nil? found))
																						found
																						(begin
																							(define inner_info (_us_scan_direct_col (nth dc 0)))
																							(if (and
																								(not (nil? inner_info))
																								(has? _us_base_aliases (nth inner_info 0)))
																								(nth dc 0)
																								nil))))
																					nil))
																			(define us_subst
																				(if (or (nil? _us_scan_match_inner_expr)
																					(and us_has_agg (not us_has_grp)))
																					us_subst_raw
																					(list (quote if)
																						(list (quote not) (list (quote nil?) (_us_ria _us_scan_match_inner_expr)))
																						us_subst_raw
																						nil)))
																			(define _us_projected_schema
																				(if (nil? _us_inner_schema)
																					(list (list "Field" "value" "Type" "any" "Expr" us_subst))
																					(merge_schema_fields_unique (list
																						(list (list "Field" "value" "Type" "any" "Expr" us_subst))
																						_us_inner_schema))))
																			(define _us_passthrough_schemas (merge
																				(if (not (nil? _us_projected_schema)) (list us_scalar_subquery_prefix _us_projected_schema) '())
																				(scalar_subselect_passthrough_schemas (merge _us_inner_tbls _us_nested_direct_tbls) schemas2_us)))
																			(if (not (equal? _us_passthrough_schemas '()))
																				(scalar_subquery_cache "schemas"
																					(reduce_assoc _us_passthrough_schemas (lambda (acc alias cols)
																						(set_assoc acc alias cols))
																						(coalesceNil (scalar_subquery_cache "schemas") '()))))
																			(if (not (nil? us_simple_uncorrelated_cache_key))
																				(scalar_subquery_cache "scalar_helper_cache"
																					(set_assoc (coalesceNil (scalar_subquery_cache "scalar_helper_cache") '())
																						us_simple_uncorrelated_cache_key
																						us_subst)))
																			(list us_subst us_tbl_entries))))))
														nil
													)
												)))
												/* If a non-aggregate LIMIT 1 scalar depends on a nested
												decorrelated helper, table-local LIMIT would apply before the
												helper join and therefore globally. With no ORDER BY, SQL allows
												any matching row, so route through the domain GROUP path and let
												the grouped projection pick one value per carried outer source. */
												(define us_nested_limit_any_path
													(and (not us_has_agg)
														(not us_has_grp)
														(or
															(not (equal? _us_inner_stages '()))
															(not (equal? _us_nested_direct_tbls '())))
														(not (nil? raw_limit_us))
														(<= raw_limit_us 1)
														(or (nil? raw_offset_us) (equal? raw_offset_us 0))
														(equal? (coalesceNil raw_order_us '()) '())))
												/* === Three-way branch: aggregate/domain GROUP / non-agg+LIMIT / non-agg-no-LIMIT === */
												(if (or us_has_agg us_has_grp us_nested_limit_any_path)
													(if (not us_simple_agg_stages)
														nil
														(us_build_aggregate_path))
													/* === B/C: Non-aggregate === */
													(us_build_scalar_scan_path)
												)
											)
										)
									)
								)
					)))
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
	(define dependent_scalar_compile_marker (lambda (idx)
		(list (quote dependent_scalar_compile) idx)))
	(define dependent_scalar_compile_marker_id (lambda (expr) (match expr
		'((quote dependent_scalar_compile) idx) idx
		'((symbol dependent_scalar_compile) idx) idx
		'(dependent_scalar_compile idx) idx
		_ nil)))
	(define not_symbol (lambda (sym) (match sym
		(symbol not) true
		'not true
		'(quote not) true
		_ false
	)))
	(define _contains_inner_select_marker (lambda (expr) (match expr
		(cons sym args) (or
			(not (nil? (inner_select_kind sym)))
			(_contains_inner_select_marker sym)
			(reduce args (lambda (found arg) (or found (_contains_inner_select_marker arg))) false))
		false)))
	(define exists_subquery_uses_session_state_for_row_existence (lambda (query)
		(match query
			'(schema2 tables2 _fields2 condition2 group2 having2 _order2 limit2 offset2)
			(expr_uses_session_state
				(list schema2 tables2 '() condition2 group2 having2 nil limit2 offset2))
			(expr_uses_session_state query))))
	(define subquery_has_unresolved_qualified_refs (lambda (query outer_schemas)
		(match query
			'(_ tables2 fields2 condition2 group2 having2 order2 _ _)
			(begin
				(define local_aliases (_raw_query_local_aliases query))
				(define unresolved_expr? (lambda (expr) (match expr
					'((symbol get_column) alias_ ti _ ci)
					(and (not (nil? alias_)) (or ti ci)
						(not (_alias_in_list local_aliases alias_))
						(not (has_assoc? outer_schemas alias_)))
					'((quote get_column) alias_ ti _ ci)
					(and (not (nil? alias_)) (or ti ci)
						(not (_alias_in_list local_aliases alias_))
						(not (has_assoc? outer_schemas alias_)))
					(cons sym args)
					(if (not (nil? (inner_select_kind sym)))
						false
						(or (unresolved_expr? sym)
							(reduce args (lambda (found arg)
								(or found (unresolved_expr? arg))) false)))
					false)))
				(or
					(reduce_assoc fields2 (lambda (found _k v)
						(or found (unresolved_expr? v))) false)
					(unresolved_expr? condition2)
					(reduce (coalesceNil group2 '()) (lambda (found g)
						(or found (unresolved_expr? g))) false)
					(unresolved_expr? having2)
					(reduce (coalesceNil order2 '()) (lambda (found o)
						(or found (match o
							'(col _dir) (unresolved_expr? col)
							(unresolved_expr? o)))) false)))
			false)))
	(define count_subquery_cache_policy (lambda (query target_expr)
		(match query
			'(s t f c g h o l off) (begin
				(define only_count (match f
					'("__cnt" ((quote aggregate) 1 op 0)) (equal?? op (quote +))
					'("__cnt" ((symbol aggregate) 1 op 0)) (equal?? op (quote +))
					false))
				(define session_sensitive_count
					(if (nil? target_expr)
						(exists_subquery_uses_session_state_for_row_existence query)
						(expr_uses_session_state query)))
				(if (and only_count session_sensitive_count)
					(quote uncached-count)
					nil))
			nil)))
	/* _unnest_count_subselect: shared helper for IN/EXISTS/NOT IN/NOT EXISTS rewrite.
	Rewrites semi-joins (EXISTS/IN) and anti-joins (NOT EXISTS/NOT IN) as COUNT(*)
	aggregates instead of direct semi/anti-join operators. This is intentional:
	the COUNT-based approach produces a keytable computed column that benefits from
	MemCP's incremental aggregate cache — DML triggers invalidate only affected
	groups, so subsequent queries skip recomputation for unchanged partitions.
	Caching policy roadmap for session-sensitive predicates:
	1. First iteration: if the COUNT/EXISTS condition depends on volatile session
	state (for example @current_user_id, @cutoff_time, snapshot_time, username),
	build_queryplan must prefer a cache-free execution path where GROUP ==
	current subselect domain and the predicate is evaluated on the current row
	stream instead of a reusable keytable cache.
	2. Second iteration: enable memoizing caches for predicates that depend on a
	stable session key (for example fixed user-id). The cache key must then
	include both the logical domain D and the memoized session value, i.e.
	semantically the cache lives on D x SessionKey rather than on D alone.
	If those entries are managed independently, cache eviction may also need
	row-wise cleanup hooks so the cache manager can register a domain plus
	memory budget together with a callback/DELETE plan for affected rows only.
	3. Third iteration: add cache-aware iterative rescans for monotone/session
	predicates by reasoning in SQL predicate algebra instead of application
	semantics. Example: x < @y is disjoint from x >= @y, and if @y < @z then
	the cached result for x < @y can be reused while only the delta predicate
	x >= @y AND x < @z has to be scanned; their union is equivalent to x < @z.
	build_queryplan can therefore decompose a broader query into reusable cache
	fragments plus a catch-up scan over the previously uncovered range.
	Until those stages are implemented, volatile session-dependent predicates must
	not be treated as freely reusable aggregate caches.
	Builds a COUNT(*) subquery from the original, optionally adding an equality condition
	(for IN/NOT IN: first_field = target_expr). Returns (substitution tables) or nil.
	comparison: (quote >) for positive match, (quote equal?) for negated match */
	(define _subquery_outer_refs (lambda (query outer_schemas) (begin
		(match (apply untangle_query (merge query (list outer_schemas)))
			'(_ tables2 fields2 condition2 _groups2 schemas2 _rfcol2 _init2) (begin
				(define _inner_aliases (map tables2 (lambda (td) (match td '(a _ _ _ _) a ""))))
				(define _eor (lambda (expr) (match expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						(match args (cons sym_arg '()) (list (string sym_arg)) '())
						(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
							(match args '(alias_ _ col _)
								(if (and (not (nil? alias_))
									(not (reduce _inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
									(list (concat alias_ "." col))
									'())
								'())
							(if (_is_opaque_scope_sym sym) '()
								(merge_unique (map args _eor)))))
					'())))
				(merge_unique
					(merge (extract_assoc fields2 (lambda (_k v) (_eor v))))
					(_eor (coalesceNil condition2 true))))
			'()))))
	(define _subquery_has_outer_refs (lambda (query outer_schemas)
		(not (equal? (_subquery_outer_refs query outer_schemas) '()))))
	(define _outer_ref_is_domain_column (lambda (outer_schemas ref) (match (split ref ".")
		(list alias col) (begin
			(define cols (if (has_assoc? outer_schemas alias) (outer_schemas alias) nil))
			(define coldef (if (nil? cols)
				nil
				(reduce cols (lambda (found cd)
					(if (or (not (nil? found)) (not (equal?? (cd "Field") col)))
						found
						cd))
					nil)))
			(if (nil? coldef)
				false
				(begin
					(define expr (coalesceNil (coldef "Expr") nil))
					(match expr
						nil true
						'((quote get_column) _ _ _ _) true
						'((symbol get_column) _ _ _ _) true
						_ (not (expr_has_opaque_scope expr))))))
		_ false)))
	(define _outer_ref_is_direct_column (lambda (outer_schemas ref) (match (split ref ".")
		(list alias col) (begin
			(define cols (if (has_assoc? outer_schemas alias) (outer_schemas alias) nil))
			(define coldef (if (nil? cols)
				nil
				(reduce cols (lambda (found cd)
					(if (or (not (nil? found)) (not (equal?? (cd "Field") col)))
						found
						cd))
					nil)))
			(if (nil? coldef)
				false
				(begin
					(define expr (coalesceNil (coldef "Expr") nil))
					(match expr
						nil true
						'((quote get_column) _ _ _ _) true
						'((symbol get_column) _ _ _ _) true
						_ false))))
		_ false)))
	(define _subquery_outer_refs_are_direct_columns (lambda (query outer_schemas)
		(reduce (_subquery_outer_refs query outer_schemas) (lambda (all_ok ref)
			(and all_ok (_outer_ref_is_direct_column outer_schemas ref)))
			true)))
	(define _subquery_outer_refs_are_domain_columns (lambda (query outer_schemas)
		(reduce (_subquery_outer_refs query outer_schemas) (lambda (all_ok ref)
			(and all_ok (_outer_ref_is_domain_column outer_schemas ref)))
			true)))
	(define _subquery_outer_refs_need_domain_preservation (lambda (query outer_schemas)
		(and
			(not (_subquery_outer_refs_are_direct_columns query outer_schemas))
			(_subquery_outer_refs_are_domain_columns query outer_schemas))))
	(define _raw_subquery_has_non_equality_outer_condition (lambda (query outer_schemas) (match query
		'(_ _ _ raw_condition _ _ _ _ _) (begin
			(define local_aliases (_raw_query_local_aliases query))
			(define raw_expr_has_outer_ref (lambda (expr) (match expr
				'((symbol get_column) alias_ _ _ _) (and (not (nil? alias_))
					(not (_alias_in_list local_aliases alias_))
					(has_assoc? outer_schemas alias_))
				'((quote get_column) alias_ _ _ _) (and (not (nil? alias_))
					(not (_alias_in_list local_aliases alias_))
					(has_assoc? outer_schemas alias_))
				(cons sym args) (if (not (nil? (inner_select_kind sym)))
					false
					(reduce args (lambda (acc arg) (or acc (raw_expr_has_outer_ref arg))) false))
				false)))
			(define raw_condition_parts (flatten_and_terms (coalesceNil raw_condition true)))
			(reduce raw_condition_parts (lambda (found part)
				(or found
					(and (raw_expr_has_outer_ref part)
						(match part
							'((symbol equal??) a b) (if (raw_expr_has_outer_ref a)
								(raw_expr_has_outer_ref b)
								(not (raw_expr_has_outer_ref b)))
							'((quote equal??) a b) (if (raw_expr_has_outer_ref a)
								(raw_expr_has_outer_ref b)
								(not (raw_expr_has_outer_ref b)))
							true))))
				false))
		false)))
	(define scalar_subselect_lowering_reason (lambda (subquery outer_schemas)
		(match (scalar_subselect_lowering_facts subquery outer_schemas)
			'(_g h _o _value_expr _has_outer _outer_refs_are_direct_columns _contains_inner_select_marker _has_aggregate) (begin
				(define _contains_ordered_limited_inner_select_marker (lambda (expr) (match expr
					(cons sym args) (begin
						(define marker_kind (inner_select_kind sym))
						(or
							(and (equal?? marker_kind (quote inner_select))
								(match args
									(cons marker_subquery '())
									(match marker_subquery
										'(_ _ _ _ _ _ marker_order marker_limit marker_offset)
										(or
											(not (equal? (coalesceNil marker_order '()) '()))
											(not (nil? marker_limit))
											(not (nil? marker_offset)))
										false)
									false))
							(_contains_ordered_limited_inner_select_marker sym)
							(reduce args (lambda (found arg)
								(or found (_contains_ordered_limited_inner_select_marker arg))) false)))
					false)))
					/* ORDER/LIMIT-only correlated scalars lower through the normal
					non-aggregate partition-topk path. Nested markers are handled by
					the top-down pipeline before the scalar value is materialized. */
				(define _has_grouped_semantics (or
					_has_aggregate
					(not (nil? h))
					(not (equal? (coalesceNil _g '()) '()))
						(and
							(not (equal? (coalesceNil _o '()) '()))
							(_contains_ordered_limited_inner_select_marker subquery))))
				(define _value_expr_is_direct_column (match _value_expr
					'((symbol get_column) _ _ _ _) true
					'((quote get_column) _ _ _ _) true
					false))
				/* uncorrelated + outer GROUP BY: defer to group-barrier refactoring
				(prejoin scoping bug when unnested table meets GROUP stage) */
				(define _outer_has_group (or group having _cd_has))
				(define _raw_limit_for_lowering (match subquery
					'(_ _ _ _ _ _ _ l _) l
					nil))
				(define _raw_offset_for_lowering (match subquery
					'(_ _ _ _ _ _ _ _ off) off
					nil))
				(define _ordered_limited_direct_scalar (and
						_has_outer
						(not _has_aggregate)
						(or
							(not (equal? (coalesceNil _o '()) '()))
							(not (nil? _raw_limit_for_lowering))
						(not (nil? _raw_offset_for_lowering)))
					(or (nil? h) (equal? h true))
					(or (nil? _g) (equal? (coalesceNil _g '()) '()))
					_value_expr_is_direct_column
					(not _outer_has_group)))
				(define _allow_grouped_direct_non_equality_outer
					(and
						(not _outer_has_group)
						_outer_refs_are_direct_columns
						(_raw_subquery_has_non_equality_outer_condition subquery outer_schemas)))
				(if _ordered_limited_direct_scalar
					(quote prefer-unnest)
					(scalar_subselect_lowering_reason_from_facts
						_has_outer
						_has_grouped_semantics
						_outer_refs_are_direct_columns
						_outer_has_group
						_contains_inner_select_marker
						_value_expr
						_value_expr_is_direct_column
						(_subquery_outer_refs_need_domain_preservation subquery outer_schemas)
						_allow_grouped_direct_non_equality_outer)))
			nil)))
	(define scalar_subselect_unnest_applicable (lambda (subquery outer_schemas)
		(equal? (scalar_subselect_lowering_reason subquery outer_schemas) (quote prefer-unnest))))
	(define _unnest_scalar_subselect (lambda (subquery outer_schemas) (begin
		(match (unnest_subselect subquery outer_schemas)
			'(subst tbls) (begin
				/* Scalar subselect unnesting yields null-preserving LEFT JOIN helper
				tables for SELECT/expr projection. Keep them separate from COUNT/IN/EXISTS
				helper tables so their joinexpr stays attached to the table entry and is
				not re-applied globally as a filter later. */
				(scalar_subquery_cache "scalar_table_original_joinexprs"
					(reduce tbls (lambda (acc td) (match td
						'(alias_ _ _ _ joinexpr_) (set_assoc acc alias_ joinexpr_)
						acc))
						(coalesceNil (scalar_subquery_cache "scalar_table_original_joinexprs") '())))
				(scalar_subquery_cache "scalar_tables" (merge tbls (coalesceNil (scalar_subquery_cache "scalar_tables") '())))
				subst)
			nil)
	)))
	(define scalar_subselect_lowering_policy (lambda (subquery outer_schemas) (begin
		(define lowering_reason (scalar_subselect_lowering_reason subquery outer_schemas))
		(planner_debug_record_scalar_event (quote lowering) lowering_reason)
		(if (equal? lowering_reason (quote prefer-unnest))
			(quote prefer-unnest)
			(quote inline-only)))))
	(define build_scalar_subselect_with_strategy (lambda (subquery outer_schemas) (begin
		(match (scalar_subselect_lowering_policy subquery outer_schemas)
			(quote prefer-unnest) (begin
				(define lowered_expr (_unnest_scalar_subselect subquery outer_schemas))
					(if (nil? lowered_expr)
						(error (concat "unable to unnest scalar subselect: " (serialize subquery)))
						(list (quote unnest) lowered_expr)))
			(build_scalar_subselect_inline_with_strategy subquery outer_schemas))
	)))
	(define scalar_subquery_projects_exists_value? (lambda (subquery)
		(match subquery
			'(_ _ scalar_fields _ _ _ _ _ _)
			(begin
				(define expr_contains_exists_marker? (lambda (expr) (match expr
					(cons first_sym first_args)
					(or
						(equal?? (inner_select_kind first_sym) (quote inner_select_exists))
						(expr_contains_exists_marker? first_sym)
						(reduce first_args (lambda (found arg)
							(or found (expr_contains_exists_marker? arg)))
							false))
					false)))
				(define first_expr (match scalar_fields
					(cons _ (cons value_expr _)) value_expr
					nil))
				(expr_contains_exists_marker? first_expr))
			false)))
	(define build_scalar_subselect (lambda (subquery outer_schemas) (begin
		(match (build_scalar_subselect_with_strategy subquery outer_schemas)
			'(_ lowered_expr)
			(if (scalar_subquery_projects_exists_value? subquery)
				(list (quote coalesceNil) lowered_expr false)
				lowered_expr)
			nil)
	)))
	(define _unnest_count_subselect (lambda (subquery outer_schemas target_expr comparison) (begin
		(define _resolve_outer (lambda (expr) (match expr
			'((symbol get_column) nil ti col ci) (begin
				(define _resolved (reduce_assoc outer_schemas (lambda (a alias cols)
					(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false) alias a)) nil))
				(if (nil? _resolved) expr
					(list (quote get_column) _resolved false col false)))
			(cons sym args) (cons (_resolve_outer sym) (map args _resolve_outer))
			expr)))
		(define resolved_target_expr (if (nil? target_expr) nil (_resolve_outer target_expr)))
		/* UNION ALL: recurse into each branch, combine with OR (positive) or AND (negated) */
		(define _union_parts (query_union_all_parts subquery))
		(if (not (nil? _union_parts))
			(match _union_parts '(branches order limit offset)
				(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
					nil /* UNION ALL with ORDER/LIMIT/OFFSET: not supported */
					(begin
						(define _first_cols (match branches
							(cons first_branch _) (query_branch_field_names first_branch)
							_ '()))
						(if (not (reduce branches (lambda (acc branch)
							(and acc (equal? (count (query_branch_field_names branch)) (count _first_cols)))) true))
							(error "UNION ALL branches must project the same number of columns")
							nil)
						(define _branch_exists_expr (lambda (branch) (match branch
							'(s t f c g h o l off) (begin
								(define _first_field (if (nil? resolved_target_expr) nil
									(match f (cons _ (cons v _)) v nil)))
								(if (and (not (nil? resolved_target_expr)) (nil? _first_field))
									nil
									(begin
										(define _branch_condition (if (nil? resolved_target_expr) c
											(if (or (nil? c) (equal? c true))
												(list (quote equal??) _first_field resolved_target_expr)
												(list (quote and) c (list (quote equal??) _first_field resolved_target_expr)))))
										(define _exists_expr (build_exists_subselect
											(list s t f _branch_condition g h o l off)
											outer_schemas))
										(if (equal?? comparison (quote >))
											_exists_expr
											(list (quote not) _exists_expr)))))
							nil)))
						(define _branch_results (filter (map branches _branch_exists_expr)
							(lambda (r) (not (nil? r)))))
						(if (or (equal? _branch_results '()) (not (equal? (count _branch_results) (count branches))))
							nil
							(if (equal? 1 (count _branch_results)) (car _branch_results)
								(cons (if (equal?? comparison (quote >)) (quote or) (quote and)) _branch_results))))))
			/* single subquery (non-UNION) path */
			(begin
				(define hoist_exists_outer_guards (lambda (query) (match query
					'(s t f c g h o l off)
					(if (or (not (nil? target_expr)) (not (equal?? comparison (quote >))))
						nil
						(begin
							(define local_aliases (_raw_query_local_aliases query))
							(define condition_parts (flatten_and_terms (coalesceNil c true)))
							(define part_has_unqualified_column? (lambda (expr) (match expr
								'((symbol get_column) nil _ _ _) true
								'((quote get_column) nil _ _ _) true
								(cons sym args) (reduce args (lambda (found arg) (or found (part_has_unqualified_column? arg))) false)
								false)))
							(define part_tblvars_outside_inner_selects (lambda (expr) (match expr
								'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
								'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
								(cons sym args)
								(if (not (nil? (inner_select_kind sym)))
									'()
									(merge_unique (map args part_tblvars_outside_inner_selects)))
								'())))
							(define raw_query_refs_aliases? (lambda (query aliases) (match query
								'(_ raw_tables raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
									(define nested_aliases (_raw_query_local_aliases query))
									(define raw_expr_refs_aliases? (lambda (expr) (match expr
										'((symbol get_column) alias_ _ _ _) (and
											(not (nil? alias_))
											(not (_alias_in_list nested_aliases alias_))
											(_alias_in_list aliases alias_))
										'((quote get_column) alias_ _ _ _) (and
											(not (nil? alias_))
											(not (_alias_in_list nested_aliases alias_))
											(_alias_in_list aliases alias_))
										(cons expr_sym expr_args)
										(reduce expr_args (lambda (found expr_arg)
											(or found (raw_expr_refs_aliases? expr_arg))) false)
										false)))
									(or
										(reduce_assoc raw_fields (lambda (found _k field_expr)
											(or found (raw_expr_refs_aliases? field_expr))) false)
										(raw_expr_refs_aliases? (coalesceNil raw_condition true))
										(reduce (coalesceNil raw_group '()) (lambda (found group_expr)
											(or found (raw_expr_refs_aliases? group_expr))) false)
										(raw_expr_refs_aliases? (coalesceNil raw_having true))
										(reduce (coalesceNil raw_order '()) (lambda (found order_item)
											(or found (match order_item
												'(order_expr _dir) (raw_expr_refs_aliases? order_expr)
												false))) false)))
								false)))
							(define part_inner_select_refs_local? (lambda (expr) (match expr
								(cons sym args)
								(if (not (nil? (inner_select_kind sym)))
									(match args
										(cons nested_query _) (raw_query_refs_aliases? nested_query local_aliases)
										false)
									(reduce args (lambda (found arg)
										(or found (part_inner_select_refs_local? arg))) false))
								false)))
							(define part_refs_local? (lambda (part)
								(or
									(part_has_unqualified_column? part)
									(part_inner_select_refs_local? part)
									(reduce (part_tblvars_outside_inner_selects part) (lambda (found tv)
										(or found (_alias_in_list local_aliases tv))) false))))
							(define local_parts (filter condition_parts part_refs_local?))
							(define guard_parts (filter condition_parts (lambda (part)
								(not (part_refs_local? part)))))
							(if (or (equal? guard_parts '()) (equal? local_parts condition_parts))
								nil
								(begin
									(define local_condition (combine_and_terms local_parts))
									(define guard_condition (combine_and_terms guard_parts))
									(if (not (_contains_inner_select_marker guard_condition))
										nil
										(begin
											(define local_expr (_unnest_count_subselect
												(list s t f local_condition g h o l off)
												outer_schemas target_expr comparison))
											(if (nil? local_expr)
												nil
												(list (quote and)
													(replace_inner_selects guard_condition outer_schemas)
													local_expr))))))))
					nil)))
				(define hoisted_exists_expr (hoist_exists_outer_guards subquery))
				(define _first_field (if (nil? target_expr) nil
					(match subquery '(_ _ flds _ _ _ _ _ _) (match flds (cons _ (cons v _)) v nil) nil)))
				(define target_expr resolved_target_expr)
				(if (not (nil? hoisted_exists_expr))
					hoisted_exists_expr
					(if (and (nil? target_expr) (not (_subquery_has_outer_refs subquery outer_schemas)))
						(begin
							(define _count_sq (match subquery
								'(s t f c g h o l off) (list s t
									(list "__cnt" (list (quote aggregate) 1 (symbol "+") 0))
									c
									nil nil nil nil nil)
								nil))
							(if (nil? _count_sq)
								nil
								(begin
									(define _count_idx (coalesceNil (scalar_subquery_cache "idx") 0))
									(scalar_subquery_cache "idx" (+ _count_idx 1))
									(define _count_alias (concat "_uncorr_cnt_" _count_idx))
									(define _count_rows_sym (symbol (concat "__uncorr_count_rows:" _count_idx)))
									(define _count_sink_sym (symbol (concat "__uncorr_count_sink:" _count_idx)))
									(define _count_materialized
										(legacy_materialized_query_term_binding_ast
											_count_alias _count_sq _count_rows_sym _count_sink_sym nil nil))
									(define mat_source (nth _count_materialized 0))
									(define mat_init (nth _count_materialized 1))
									/* D = ∅: materialize the helper once and expose it as a normal
									one-row relation with visible column __cnt. The outer query still
									sees a regular table input, not a nested runtime subquery. */
									(scalar_subquery_cache "init" (merge (coalesceNil (scalar_subquery_cache "init") '())
										(list mat_init)))
									(scalar_subquery_cache "tables" (merge
										(list (list _count_alias schema mat_source false nil))
										(coalesceNil (scalar_subquery_cache "tables") '())))
									(scalar_subquery_cache "schemas" (merge
										(list _count_alias (list (list "Field" "__cnt" "Type" "any")))
										(coalesceNil (scalar_subquery_cache "schemas") '())))
									(list comparison
										(list (quote coalesceNil)
											(list (quote get_column) _count_alias false "__cnt" false)
											0)
										0))))
						(if (and (not (nil? target_expr)) (nil? _first_field))
							nil
							(begin
								(define _count_local_aliases (_raw_query_local_aliases subquery))
								(define _count_tblvars_outside_inner_selects (lambda (expr) (match expr
									'(get_column alias_ _ _ _) (if (nil? alias_) '() (list alias_))
									'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
									'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
									(cons sym args)
									(if (not (nil? (inner_select_kind sym)))
										'()
										(merge_unique (map args _count_tblvars_outside_inner_selects)))
									'())))
								(define _count_part_local_only? (lambda (part)
									(reduce (_count_tblvars_outside_inner_selects part) (lambda (ok ref_alias)
										(and ok (_alias_in_list _count_local_aliases ref_alias)))
										true)))
								(define _count_hidden_helper_alias? (lambda (alias_)
									(and (string? alias_)
										(or
											(and (> (strlen alias_) 0) (equal? (substr alias_ 0 1) "."))
											(strlike alias_ "domain_scalar_%")))))
								(define _count_hidden_helper_join_terms
									(merge (map (match subquery '(_ ts _ _ _ _ _ _ _) ts '()) (lambda (td) (match td
										'(tv _ ttbl _ tjoinexpr)
										(if (_count_hidden_helper_alias? (if (nil? tv) ttbl tv))
											(flatten_and_terms (coalesceNil tjoinexpr true))
											'())
										'())))))
								(define _count_base_condition
									(coalesceNil (match subquery '(_ _ _ c _ _ _ _ _) c true) true))
								(define _count_target_condition
									(if (nil? target_expr) nil
										(list (quote equal??) _first_field target_expr)))
								(define _count_effective_condition
									(combine_and_terms (list _count_base_condition _count_target_condition)))
								(define _count_domain_condition
									(combine_and_terms
										(filter (flatten_and_terms _count_effective_condition)
											(lambda (part)
												(and
													(not (equal?? part true))
													(not (_count_part_local_only? part)))))))
								(define _count_local_filter
									(combine_and_terms
										(merge
											(filter (flatten_and_terms _count_effective_condition)
												(lambda (part)
													(and
														(not (equal?? part true))
														(_count_part_local_only? part))))
											(filter _count_hidden_helper_join_terms (lambda (part)
												(not (equal?? part true)))))))
								(define _count_input_expr
									(if (or (nil? _count_local_filter) (equal?? _count_local_filter true))
										1
										(list (quote if) _count_local_filter 1 0)))
								(define _count_sq (match subquery
									'(s t f c g h o l off) (list s t
										(list "__cnt" (list (quote aggregate) _count_input_expr (symbol "+") 0))
										_count_domain_condition
										(list 1) nil nil nil nil)
									nil))
								(if (nil? _count_sq)
									nil
									(begin
										(define _result (unnest_subselect _count_sq outer_schemas))
										(if (nil? _result)
											nil
											(match _result '(_subst _tbls) (begin
												(scalar_subquery_cache "tables" (merge _tbls (coalesceNil (scalar_subquery_cache "tables") '())))
												(list comparison (list (quote coalesceNil) _subst 0) 0))))))))))))
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
			(define union_exists_expr (lambda (subquery negated) (begin
				(define union_parts (query_union_all_parts subquery))
				(if (nil? union_parts)
					nil
					(match union_parts '(branches union_order union_limit union_offset)
						(if (or (not (nil? union_order)) (not (nil? union_limit)) (not (nil? union_offset)))
							nil
							(begin
								(define empty_query_part? (lambda (part)
									(or (nil? part) (equal? part false) (equal? part '()))))
								(define branch_exprs (map branches (lambda (branch) (begin
									(define resolved_branch (match branch
										'(b_schema _b_tables _b_fields _b_condition b_group b_having b_order b_limit b_offset)
										(match (untangle_scalar_subquery_scope branch outer_schemas b_group b_having b_order b_limit b_offset)
											'(rb_schema rb_tables rb_fields rb_condition rb_groups rb_schemas _rb_rf _rb_init)
											(begin
												(define rb_resolve (make_replace_find_column_subselect rb_schemas outer_schemas false))
												(list rb_schema rb_tables
													(map_assoc rb_fields (lambda (k v) (rb_resolve v)))
													(rb_resolve (coalesceNil rb_condition true))
													b_group b_having b_order b_limit b_offset))
											branch)
										branch))
									(define direct_exists_probe (match resolved_branch
										'(rb_schema rb_tables rb_fields rb_condition rb_group rb_having rb_order rb_limit rb_offset)
										(if (and
											(equal? (count rb_tables) 1)
											(empty_query_part? rb_group)
											(empty_query_part? rb_having)
											(empty_query_part? rb_order)
											(empty_query_part? rb_limit)
											(empty_query_part? rb_offset))
											(match (car rb_tables)
												'(old_alias tbl_schema tbl_name _old_outer old_joinexpr) (begin
													(define ex_idx (coalesceNil (scalar_subquery_cache "idx") 0))
													(scalar_subquery_cache "idx" (+ ex_idx 1))
													(define ex_alias (concat "domain_exists_" old_alias "_" ex_idx))
													(define ex_alias_map (list old_alias ex_alias))
													(define ex_outer_to_get_column (lambda (expr) (match expr
														'((quote outer) outer_sym) (match (split (string outer_sym) ".")
															(list outer_tbl outer_col) (list (quote get_column) outer_tbl false outer_col false)
															expr)
														'((symbol outer) outer_sym) (match (split (string outer_sym) ".")
															(list outer_tbl outer_col) (list (quote get_column) outer_tbl false outer_col false)
															expr)
														(cons sym args) (cons sym (map args ex_outer_to_get_column))
														expr)))
													(define ex_join (ex_outer_to_get_column
														(rewrite_source_aliases ex_alias_map (coalesceNil rb_condition true))))
													(scalar_subquery_cache "scalar_tables" (merge
														(list (list ex_alias tbl_schema tbl_name true ex_join))
														(coalesceNil (scalar_subquery_cache "scalar_tables") '())))
													(scalar_subquery_cache "schemas" (merge
														(list ex_alias (get_schema tbl_schema tbl_name))
														(coalesceNil (scalar_subquery_cache "schemas") '())))
													(match rb_fields
														(cons _ (cons first_field _))
														(rewrite_source_aliases ex_alias_map first_field)
														nil))
												nil)
											nil)
										nil))
									(define scalar_probe (if (nil? direct_exists_probe)
										(_unnest_scalar_subselect resolved_branch outer_schemas)
										direct_exists_probe))
									(if (not (nil? scalar_probe))
										(if negated
											(list (quote nil?) scalar_probe)
											(list (quote not) (list (quote nil?) scalar_probe)))
										(begin
											(define branch_count_expr (match resolved_branch
												'(_ _ b_fields _ _ _ _ _ _)
												(match b_fields
													(cons _ (cons first_field _))
													(list (quote if) (list (quote nil?) first_field) 1 1)
													1)
												1))
											(define branch_count_query (match resolved_branch
												'(b_schema b_tables _b_fields b_condition _b_group _b_having _b_order _b_limit _b_offset)
												(list b_schema b_tables
													(list "__cnt" (list (quote aggregate) branch_count_expr (quote +) 0))
													b_condition
													(list 1) nil nil nil nil)
												nil))
											(define branch_count_result (if (or (nil? branch_count_query) (subquery_has_unresolved_qualified_refs resolved_branch outer_schemas))
												nil
												(unnest_subselect branch_count_query outer_schemas)))
											(coalesce
												(match branch_count_result
													'(branch_subst branch_tbls) (begin
														(scalar_subquery_cache "tables" (merge branch_tbls (coalesceNil (scalar_subquery_cache "tables") '())))
														(list (if negated (quote equal?) (quote >))
															(list (quote coalesceNil) branch_subst 0)
															0))
													nil)
												(if negated
													(list (quote not) (build_exists_subselect resolved_branch outer_schemas))
													(build_exists_subselect resolved_branch outer_schemas)))))))))
								(if (equal? branch_exprs '())
									nil
									(if (equal? (count branch_exprs) 1)
										(car branch_exprs)
										(cons (if negated (quote and) (quote or)) branch_exprs))))))))))
			(define union_in_expr (lambda (target_expr subquery negated) (begin
				(define union_parts (query_union_all_parts subquery))
				(if (nil? union_parts)
					nil
					(match union_parts '(branches union_order union_limit union_offset)
						(if (or (not (nil? union_order)) (not (nil? union_limit)) (not (nil? union_offset)))
							nil
							(begin
								(if (not (reduce branches (lambda (ok branch)
									(and ok (equal? 1 (count (query_branch_field_names branch)))))
									true))
									(error "UNION ALL subquery must project exactly one column for IN")
									nil)
								(define normalize_union_in_branch (lambda (branch)
									(match branch
										'(b_schema b_tables b_fields b_condition b_group b_having b_order b_limit b_offset)
										(begin
											(define first_field_expr (match b_fields
												(cons _ (cons v _)) v
												nil))
											(if (or (nil? first_field_expr) (not (or (nil? b_condition) (equal? b_condition true))))
												branch
												(list b_schema b_tables b_fields
													(list (quote equal??) first_field_expr first_field_expr)
													b_group b_having b_order b_limit b_offset)))
										branch)))
								(define rewritten_expr
									(if (equal? (count branches) 1)
										(if negated
											(list (quote not) (list (quote inner_select_in) target_expr (normalize_union_in_branch (car branches))))
											(list (quote inner_select_in) target_expr (normalize_union_in_branch (car branches))))
										(cons (if negated (quote and) (quote or))
											(map branches (lambda (branch)
												(if negated
													(list (quote not) (list (quote inner_select_in) target_expr (normalize_union_in_branch branch)))
													(list (quote inner_select_in) target_expr (normalize_union_in_branch branch))))))))
								(replace_inner_selects rewritten_expr outer_schemas))))))))
			(define not_expr (if (not_symbol sym)
				(match args
					(cons inner_expr '()) (match inner_expr
						(cons inner_sym inner_args) (begin
							(define inner_kind (inner_select_kind inner_sym))
							(if (equal?? inner_kind (quote inner_select_in))
								(match inner_args
									(cons target_expr (cons subquery '()))
									(coalesce
										(union_in_expr target_expr subquery true)
										(if (subquery_has_unresolved_qualified_refs subquery outer_schemas)
											nil
											(_unnest_count_subselect subquery outer_schemas target_expr (quote equal?)))
										(match subquery
											'(nis_schema nis_tables nis_fields nis_condition nis_group nis_having nis_order nis_limit nis_offset)
											(begin
												(define nis_first_field (match nis_fields
													(cons _ (cons v _)) v
													nil))
												(if (nil? nis_first_field)
													expr
													(list (quote not)
														(build_exists_subselect
															(list nis_schema nis_tables nis_fields
																(if (or (nil? nis_condition) (equal? nis_condition true))
																	(list (quote equal??) nis_first_field target_expr)
																	(list (quote and) nis_condition (list (quote equal??) nis_first_field target_expr)))
																nis_group nis_having nis_order nis_limit nis_offset)
															outer_schemas))))
											expr))
									_ nil)
								(if (equal?? inner_kind (quote inner_select_exists))
									(match inner_args
										(cons subquery '())
										(begin
											(define _union_exists_neg (union_exists_expr subquery true))
											(if (not (nil? _union_exists_neg))
												_union_exists_neg
												(if (or
													(exists_subquery_uses_session_state_for_row_existence subquery)
													(_contains_inner_select_marker subquery))
													(list (quote not) (build_exists_subselect subquery outer_schemas))
													(begin
														(define _count_exists_neg
															(if (subquery_has_unresolved_qualified_refs subquery outer_schemas)
																nil
																(_unnest_count_subselect subquery outer_schemas nil (quote equal?))))
														(if (not (nil? _count_exists_neg))
															_count_exists_neg
															(list (quote not) (build_exists_subselect subquery outer_schemas)))))))
										_ nil)
									nil)))
						_ nil)
					_ nil)
				nil))
			(define scalar_value_comparison_op? (lambda (op)
				(or (equal? op (quote equal?)) (equal? op (symbol equal?))
					(equal? op (quote equal??)) (equal? op (symbol equal??))
					(equal? op (quote =)) (equal? op (symbol =))
					(equal? op (quote >)) (equal? op (symbol >))
					(equal? op (quote <)) (equal? op (symbol <))
					(equal? op (quote >=)) (equal? op (symbol >=))
					(equal? op (quote <=)) (equal? op (symbol <=)))))
			(define scalar_count_zero_comparison_expr (lambda ()
				(begin
					(define equality_comparison_op? (lambda (op)
						(or (equal? op (quote equal?)) (equal? op (symbol equal?))
							(equal? op (quote equal??)) (equal? op (symbol equal??))
							(equal? op (quote =)) (equal? op (symbol =)))))
					(define zero_literal? (lambda (value)
						(and (number? value) (equal? value 0))))
					(define plus_reduce? (lambda (op)
						(or (equal? op (quote +)) (equal? op (symbol +)))))
					(define aggregate_symbol? (lambda (head)
						(or
							(equal? head (quote aggregate))
							(equal? head (symbol aggregate))
							(equal? head '(quote aggregate)))))
					(define if_symbol? (lambda (head)
						(or
							(equal? head (quote if))
							(equal? head (symbol if))
							(equal? head '(quote if)))))
					(define nil_symbol? (lambda (head)
						(or
							(equal? head (quote nil?))
							(equal? head (symbol nil?))
							(equal? head '(quote nil?)))))
					(define count_presence_condition (lambda (count_expr)
						(match count_expr
							1 true
							(cons if_head if_args)
							(if (if_symbol? if_head)
								(match if_args
									'(nil_check zero_value one_value)
									(if (and (zero_literal? zero_value) (equal? one_value 1))
										(match nil_check
											(cons nil_head nil_args)
											(if (nil_symbol? nil_head)
												(match nil_args
													(cons nullable_expr '())
													(if (number? nullable_expr)
														true
														(list (quote not) (list (quote nil?) nullable_expr)))
													nil)
												nil)
											nil)
										nil)
									nil)
								nil)
							nil)))
					(define count_zero_subquery (lambda (subquery)
						(match subquery
							'(count_schema count_tables count_fields count_condition count_group count_having count_order count_limit count_offset)
							(if (or
								(not (or (nil? count_group) (equal? count_group '())))
								(not (nil? count_having))
								(not (or (nil? count_order) (equal? count_order '())))
								(not (nil? count_limit))
								(not (nil? count_offset)))
								nil
								(begin
									(define count_field_expr
										(match count_fields
											(cons _ (cons first_expr _)) first_expr
											nil))
									(match count_field_expr
										(cons agg_head agg_args)
										(if (aggregate_symbol? agg_head)
											(match agg_args
												'(count_expr count_reduce count_neutral)
												(if (and (plus_reduce? count_reduce) (zero_literal? count_neutral))
													(begin
														(define presence_condition (count_presence_condition count_expr))
														(if (nil? presence_condition)
															nil
															(list count_schema count_tables
																(list "__exists" true)
																(combine_and_terms (list count_condition presence_condition))
																nil nil nil nil nil)))
													nil)
												nil)
											nil)
										nil)))
							nil)))
					(if (and (equality_comparison_op? sym) (equal? (count args) 2))
						(begin
							(define count_zero_inner_arg (lambda (candidate other)
								(if (zero_literal? other)
									(match candidate
										(cons inner_sym inner_args)
										(if (equal?? (inner_select_kind inner_sym) (quote inner_select))
											(match inner_args
												(cons scalar_subquery '())
												(count_zero_subquery scalar_subquery)
												nil)
											nil)
										nil)
									nil)))
							(define exists_query
								(coalesce
									(count_zero_inner_arg (nth args 0) (nth args 1))
									(count_zero_inner_arg (nth args 1) (nth args 0))))
							(if (nil? exists_query)
								nil
								(_unnest_count_subselect exists_query outer_schemas nil (quote equal?))))
						nil))))
				(define single_value_limited_subquery? (lambda (subquery)
					(match subquery
						'(_ _ _ _ _ _ _ scalar_limit _)
						(and (not (nil? scalar_limit)) (<= scalar_limit 1))
						false)))
				(define scalar_value_comparison_expr (if (scalar_value_comparison_op? sym)
					(begin
						(define replace_scalar_value_arg (lambda (arg) (match arg
							(cons inner_sym inner_args) (if (equal?? (inner_select_kind inner_sym) (quote inner_select))
								(match inner_args
									(cons scalar_subquery '())
									(match (build_scalar_subselect_with_strategy scalar_subquery outer_schemas)
										'(_ lowered_scalar_expr) lowered_scalar_expr
										(build_scalar_subselect_inline scalar_subquery outer_schemas))
									_ (replace_inner_selects arg outer_schemas))
								(replace_inner_selects arg outer_schemas))
							_ (replace_inner_selects arg outer_schemas))))
					(cons sym (map args replace_scalar_value_arg)))
				nil))
			(if (nil? not_expr)
				(coalesce (scalar_count_zero_comparison_expr)
					scalar_value_comparison_expr
					(match kind
						(quote inner_select) (match args
							(cons subquery '())
							(if (and
								(or
									(_contains_inner_select_marker subquery)
									(scalar_subquery_projects_exists_value? subquery))
								(single_value_limited_subquery? subquery))
								(begin
									(define inline_scalar_expr
										(build_scalar_subselect_inline subquery outer_schemas))
									(if (scalar_subquery_projects_exists_value? subquery)
										(list (quote coalesceNil) inline_scalar_expr false)
										inline_scalar_expr))
								(match (build_scalar_subselect_with_strategy subquery outer_schemas)
									'(_ lowered_expr)
									(if (scalar_subquery_projects_exists_value? subquery)
										(list (quote coalesceNil) lowered_expr false)
										lowered_expr)
									nil))
							_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
						(quote inner_select_in) (match args
							(cons target_expr (cons subquery '()))
							(coalesce
								(union_in_expr target_expr subquery false)
								(if (single_value_limited_subquery? subquery)
									(replace_inner_selects (list (quote equal??) target_expr (list (quote inner_select) subquery)) outer_schemas)
									nil)
								(if (subquery_has_unresolved_qualified_refs subquery outer_schemas)
									nil
									(_unnest_count_subselect subquery outer_schemas target_expr (quote >)))
								expr)
							_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
						(quote inner_select_exists) (match args
							(cons subquery '())
							(begin
								(define _union_exists_pos (union_exists_expr subquery false))
								(if (not (nil? _union_exists_pos))
									_union_exists_pos
									(begin
										(if (or
											(exists_subquery_uses_session_state_for_row_existence subquery)
											(_contains_inner_select_marker subquery))
											(build_exists_subselect subquery outer_schemas)
											(begin
												(define _count_exists_pos
													(if (subquery_has_unresolved_qualified_refs subquery outer_schemas)
														nil
														(_unnest_count_subselect subquery outer_schemas nil (quote >))))
												(if (not (nil? _count_exists_pos))
													_count_exists_pos
													(build_exists_subselect subquery outer_schemas)))))))
							_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas))))))
				not_expr))
		expr
	)))
	/* Compile-only scalar markers keep large expr trees free of eager scalar
	lowering while the current scope is still being normalized. The marker never
	escapes the compiler; it is resolved back into the normal logical plan before
	scalar subquery helper integration. */
	(define nil_test_of_inner_select (lambda (expr) (match expr
		(cons nil_sym (cons inner_expr '()))
		(and
			(or
				(equal?? nil_sym (symbol nil?))
				(equal?? nil_sym (quote nil?))
				(equal?? nil_sym (quote (quote nil?))))
			(match inner_expr
				(cons inner_sym (cons _ '()))
				(equal?? (inner_select_kind inner_sym) (quote inner_select))
				_ false))
		_ false)))
	(define collect_dependent_scalar_compile_markers (lambda (expr outer_schemas)
		(if (nil_test_of_inner_select expr)
			(replace_inner_selects expr outer_schemas)
			(match expr
				(cons sym args) (begin
					(define kind (inner_select_kind sym))
					(if (equal?? kind (quote inner_select))
						(match args
							(cons subquery '()) (begin
								(if (scalar_subselect_unnest_applicable subquery outer_schemas)
									(replace_inner_selects expr outer_schemas)
									(begin
										(define dep_id (coalesceNil (dep_scalar_cache "idx") 1))
										(dep_scalar_cache "idx" (+ dep_id 1))
										(dep_scalar_cache dep_id subquery)
										(dependent_scalar_compile_marker dep_id))))
							_ (replace_inner_selects expr outer_schemas))
						(if (nil? kind)
							(cons sym (map args (lambda (arg) (collect_dependent_scalar_compile_markers arg outer_schemas))))
							(replace_inner_selects expr outer_schemas))))
				_ expr))))
	(define resolve_dependent_scalar_compile_markers (lambda (expr outer_schemas)
		(match expr
			(cons sym args) (begin
				(define dep_id (dependent_scalar_compile_marker_id expr))
				(if (nil? dep_id)
					(if (_is_opaque_scope_sym sym)
						expr
						(cons sym (map args (lambda (arg)
							(resolve_dependent_scalar_compile_markers arg outer_schemas)))))
					(begin
						(define subquery (dep_scalar_cache dep_id))
						(coalesce
							(build_scalar_subselect subquery outer_schemas)
							(replace_inner_selects (list (quote inner_select) subquery) outer_schemas)
							expr))))
			_ expr)))
	/* no-FROM rewrite: inject virtual one-row table ".(1)" (like Oracle DUAL).
	Dot prefix hides from SHOW TABLES. Eliminates the no-table special case.
	set tables= must wrap the if (set is scope-local in this Scheme dialect). */
	(set tables (if (or (nil? tables) (equal? tables '()))
		(begin
			(createdatabase schema true)
			/* ".(1)" is the synthetic one-row DUAL table for no-FROM queries.
			It may already exist while being empty after cache eviction / restart /
			other recovery paths. Re-check runtime emptiness instead of inserting
			only on first CREATE, otherwise scalar/EXISTS no-FROM subqueries become
			silently empty and collapse to NULL/FALSE. */
			(begin
				(createtable schema ".(1)"
					(list (list "unique" "group" (list "1")) (list "column" "1" "any" (list) (list)))
					(list "engine" "sloppy") true)
				(if (table_empty? (table schema ".(1)"))
					(insert (table schema ".(1)") (list "1") (list (list 1)) (list) (lambda () true) true)
					nil))
			(list (list ".(1)" schema ".(1)" false nil)))
		tables))
	(set zipped (zip (map tables (lambda (tbldesc) (begin
		/* PRE-CHECK: scan-tagged-table is a TAGGED base table per FAQ §20
		inline-flat lowering for scalar context — NOT a derived sub-query.
		Use the existing scan_tagged_table_needs_scan_order predicate
		(handles all quoting variants). Treat as base so its order/limit
		slots aren't misinterpreted as a 9-tuple SELECT shape (which
		breaks reduce_assoc on the order list as "fields"). */
		(define _tbl_at_3 (if (and (list? tbldesc) (>= (count tbldesc) 3))
			(nth tbldesc 2) nil))
		(if (and (not (nil? _tbl_at_3))
			(scan_tagged_table_needs_scan_order _tbl_at_3))
			(begin
				(define _alias (nth tbldesc 0))
				(define _schema (nth tbldesc 1))
				(define _base (scan_tagged_table_base _tbl_at_3))
				(list (list tbldesc) '() true (list _alias (get_schema _schema _base))))
			(match tbldesc
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
							(define row_sink_sym (symbol (concat "__from_union_sink:" id)))
							(define materialized_binding
								(legacy_materialized_query_term_binding_ast
									id subquery rows_sym row_sink_sym nil nil))
							(define mat_source (nth materialized_binding 0))
							(define mat_init (nth materialized_binding 1))
							(planned_materialized_fields mat_source
								(map output_cols (lambda (col) (list "Field" col "Type" "any"))))
							(scalar_subquery_cache "init" (merge (coalesceNil (scalar_subquery_cache "init") '())
								(list mat_init)))
							(list
								(list (list id schemax mat_source isOuter joinexpr))
								'()
								true
								(list id (map output_cols (lambda (col) (list "Field" col "Type" "any"))))
							)
						))
						(match (prepare_queryplan_term subquery) '(select_core_term schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2) (begin
							(if (and (not (nil? _init2)) (not (equal? _init2 '())))
								(scalar_subquery_cache "init" (merge (coalesceNil (scalar_subquery_cache "init") '()) _init2)))
							(define fields2_lookup (lambda (col)
								(coalesce
									(fields2 col)
									(fields2 (string col))
									(if (string? col) (fields2 (symbol col)) nil))))
							(define same_flatten_alias? (lambda (a b)
								(or (equal?? a b)
									(equal? (string a) (string b)))))
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
									(define matches (reduce matches (lambda (acc alias) (append_unique acc alias)) '()))
									(match matches
										(cons only '()) '('get_column (concat id "\0" only) ti col ci)
										'() (begin
											/* column not in schemas2 - check if it's a SELECT alias in fields2 */
											(define field_expr (fields2_lookup col))
											(if (nil? field_expr)
												expr /* leave unresolved — inner subselect scope will handle it */
												/* found in fields2 - resolve to the underlying expression */
												(replace_column_alias field_expr)
											)
										)
										(cons _ _) (error (concat "ambiguous column " col " in subquery"))
									)
								)
								'((symbol get_column) alias_ ti col ci) (if (same_flatten_alias? alias_ id)
									(begin
										(define field_expr (fields2_lookup col))
										(if (nil? field_expr)
											(replace_column_alias (list (quote get_column) nil ti col ci))
											(replace_column_alias field_expr)))
									(if (not (nil? (schemas2 alias_)))
										'('get_column (concat id "\0" alias_) ti col ci)
										expr)) /* alias not in schemas2 → inner subselect scope, leave as-is */
								'((quote get_column) alias_ ti col ci) (if (same_flatten_alias? alias_ id)
									(begin
										(define field_expr (fields2_lookup col))
										(if (nil? field_expr)
											(replace_column_alias (list (quote get_column) nil ti col ci))
											(replace_column_alias field_expr)))
									(if (not (nil? (schemas2 alias_)))
										'('get_column (concat id "\0" alias_) ti col ci)
										expr)) /* alias not in schemas2 → inner subselect scope, leave as-is */
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
								(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym)))
									expr /* inner subselects resolved later by replace_inner_selects */
									(cons (replace_column_alias sym) (map args replace_column_alias)))
								expr
							)))
							/* prefix all table aliases and transform their joinexprs */
							(define replace_column_alias_table_ref (lambda (tbl)
								(if (scan_tagged_table_needs_scan_order tbl)
									(scan_tagged_table_with_outer_sources
										(make_scan_tagged_table
											(scan_tagged_table_base tbl)
											(map (scan_tagged_table_order tbl) (lambda (o) (match o
												'(col dir) (list (replace_column_alias col) dir)
												o)))
											(scan_tagged_table_limit tbl)
											(scan_tagged_table_offset tbl)
											(scan_tagged_table_partition_cols tbl)
											(scan_tagged_table_once_limit tbl))
										(scan_tagged_table_outer_sources tbl))
									tbl)))
							(set tablesPrefixed (map tables2 (lambda (x) (match x '(alias schema tbl a innerJoinexpr)
								(list (concat id "\0" alias) schema (replace_column_alias_table_ref tbl) a
									(if (nil? innerJoinexpr) nil (replace_column_alias innerJoinexpr)))))))
							/* helper function to transform joinexpr: only transform references to subquery alias id */
							(define transform_joinexpr (lambda (expr) (match expr
								'((symbol get_column) alias_ ti col ci) (if (same_flatten_alias? alias_ id)
									/* reference to subquery alias -> prefer the derived projection
									boundary, then fall back to inner schemas by passing nil alias */
									(begin
										(define field_expr (fields2_lookup col))
										(if (nil? field_expr)
											(replace_column_alias (list (quote get_column) nil ti col ci))
											(replace_column_alias field_expr)))
									/* reference to outer table -> keep as-is */
									expr)
								'((quote get_column) alias_ ti col ci) (if (same_flatten_alias? alias_ id)
									(begin
										(define field_expr (fields2_lookup col))
										(if (nil? field_expr)
											(replace_column_alias (list (quote get_column) nil ti col ci))
											(replace_column_alias field_expr)))
									expr)
								(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym))) expr /* inner subselects have their own scope */ (cons sym (map args transform_joinexpr)))
								expr
							)))
							(define flatten_scalar_helper_alias? (lambda (alias)
								(scalar_helper_root_alias? alias)))
							(define flatten_nested_scalar_helper_alias? (lambda (alias)
								(and
									(string? alias)
									(> (count (split alias "\0")) 2))))
							(define table_aliases_order (map tablesPrefixed (lambda (td)
								(match td '(a _ _ _ _) a nil))))
							(define flatten_atom_ref_aliases (lambda (expr)
								(if (or (nil? expr) (string? expr) (list? expr))
									'()
									(begin
										(define parts (split (string expr) "."))
										(if (> (count parts) 1)
											(begin
												(define alias_part (car parts))
												(if (has? table_aliases_order alias_part) (list alias_part) '()))
											'())))))
							(define flatten_expr_ref_aliases (lambda (expr) (match expr
								'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
								'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list alias_))
								(cons sym args) (reduce args (lambda (acc arg)
									(merge_unique acc (flatten_expr_ref_aliases arg)))
									'())
								_ (flatten_atom_ref_aliases expr))))
							(define expr_refs_alias? (lambda (expr alias)
								(reduce (flatten_expr_ref_aliases expr) (lambda (acc ref_alias)
									(or acc (same_flatten_alias? ref_alias alias)))
									false)))
							(define flatten_extract_columns_for_alias (lambda (alias expr) (match expr
								'((symbol get_column) (eval alias) _ col _) (if (equal? col "*") '() (list col))
								'((quote get_column) (eval alias) _ col _) (if (equal? col "*") '() (list col))
								(cons sym args) (reduce args (lambda (acc arg)
									(merge_unique acc (flatten_extract_columns_for_alias alias arg)))
									'())
								_ (if (or (nil? expr) (string? expr) (list? expr))
									'()
									(begin
										(define parts (split (string expr) "."))
										(if (and (> (count parts) 1) (same_flatten_alias? (car parts) alias))
											(list (nth parts 1))
											'()))))))
							(define flatten_equality_conjunct? (lambda (expr)
								(match expr
									'(op _ _) (or (equal? op (quote equal?)) (equal? op (symbol equal?))
										(equal? op (quote equal??)) (equal? op (symbol equal??))
										(equal? op (quote =)) (equal? op (symbol =)))
									false)))
							(define add_join_part (lambda (old part)
								(if (or (nil? old) (equal? old true))
									part
									(if (or (nil? part) (equal? part true))
										old
										(list (quote and) old part)))))
							(define condition_target_alias (lambda (expr)
								(reduce table_aliases_order (lambda (acc alias)
									(if (expr_refs_alias? expr alias) alias acc))
									nil)))
							(define flatten_part_has_external_ref? (lambda (part target)
								(reduce (flatten_expr_ref_aliases part) (lambda (found ref_alias)
									(or found
										(and
											(not (same_flatten_alias? ref_alias target))
											(not (has? table_aliases_order ref_alias)))))
									false)))
							(define flatten_part_has_other_ref? (lambda (part target)
								(reduce (flatten_expr_ref_aliases part) (lambda (found ref_alias)
									(or found (not (same_flatten_alias? ref_alias target))))
									false)))
							(define flatten_part_has_scalar_helper_ref? (lambda (part target)
								(reduce (flatten_expr_ref_aliases part) (lambda (found ref_alias)
									(or found
										(and
											(not (same_flatten_alias? ref_alias target))
											(string? ref_alias)
											(>= (strlen ref_alias) 14)
											(equal? (substr ref_alias 0 14) "domain_scalar_"))))
									false)))
							(define flatten_scalar_join_filter_split (lambda (expr) (begin
								(define split_result
									(reduce (flatten_and_terms (coalesceNil expr true)) (lambda (state part)
										(match state '(join_parts filter_parts)
											(begin
												(define target (condition_target_alias part))
												(define target_local_part (and
													(not (nil? target))
													(not (flatten_part_has_other_ref? part target))))
												(define target_correlation_part (and
													(not (nil? target))
													(flatten_equality_conjunct? part)
													(flatten_part_has_other_ref? part target)))
												(if (and
													(or target_local_part target_correlation_part)
													(not (_contains_inner_select_marker part))
													(not (flatten_part_has_scalar_helper_ref? part target)))
													(list (merge join_parts (list part)) filter_parts)
													(list join_parts (merge filter_parts (list part)))))))
										(list '() '())))
								(match split_result '(join_parts filter_parts)
									(list (combine_and_terms join_parts) (combine_and_terms filter_parts))))))
							/* transform and attach joinexpr to first table in tablesPrefixed */
							(set joinexpr2 (if (nil? joinexpr) nil (transform_joinexpr joinexpr)))
							(set condition2_transformed (replace_column_alias condition2))
							/* For scalar-helper LEFT flattening, the subquery's local WHERE
							predicate belongs to the helper's ON condition, not the global WHERE:
							it must run before an ordered/limited helper scan chooses its row while
							still preserving the outer row through LEFT JOIN null extension. */
							(define projection_joinexpr2 (if isOuter
								/* merge condition2 into joinexpr for outer joins */
								(if (nil? joinexpr2)
									condition2_transformed
									(if (or (nil? condition2_transformed) (equal? condition2_transformed true))
										joinexpr2
										(list (quote and) joinexpr2 condition2_transformed)))
								joinexpr2))
							(define scalar_projection_join_filter
								(if (and isOuter (flatten_scalar_helper_alias? id))
									(flatten_scalar_join_filter_split projection_joinexpr2)
									(list projection_joinexpr2 true)))
							(define scalar_projection_joinexpr2 (nth scalar_projection_join_filter 0))
							(define scalar_projection_filter2 (nth scalar_projection_join_filter 1))
							(define flatten_scalar_filter_only_for_alias (lambda (expr alias)
								(filter (flatten_and_terms (coalesceNil expr true)) (lambda (part)
									(and
										(expr_refs_alias? part alias)
										(not (flatten_part_has_other_ref? part alias)))))))
							(define scalar_table_filter2
								(if (and isOuter (flatten_scalar_helper_alias? id))
									(combine_and_terms (merge (map tablesPrefixed (lambda (td) (match td
										'(a _ _ _ je) (if (nil? je) '() (flatten_scalar_filter_only_for_alias je a))
										'())))))
									true))
							(define flatten_scalar_join_only_for_alias (lambda (expr alias) (begin
								(define join_parts (filter (flatten_and_terms (coalesceNil expr true)) (lambda (part)
									(not (and
										(expr_refs_alias? part alias)
										(not (flatten_part_has_other_ref? part alias)))))))
								(combine_and_terms join_parts))))
							(if (and isOuter (flatten_scalar_helper_alias? id))
								(set tablesPrefixed (map tablesPrefixed (lambda (td) (match td
									'(a s t io je)
									(list a s t io (if (nil? je) nil (flatten_scalar_join_only_for_alias je a)))
									td)))))
							(set joinexpr2 (if (and isOuter (flatten_scalar_helper_alias? id))
								scalar_projection_joinexpr2
								projection_joinexpr2))
							(define scalar_projection_refs_nested_helper (reduce
								(flatten_expr_ref_aliases (coalesceNil projection_joinexpr2 true))
								(lambda (found ref_alias)
									(or found (flatten_nested_scalar_helper_alias? ref_alias)))
								false))
							(define scalar_first_table_is_outer (and
								isOuter
								(not (and
									(flatten_scalar_helper_alias? id)
									scalar_projection_refs_nested_helper))))
							(if (and (not (nil? joinexpr2)) (not (nil? tablesPrefixed)))
								(set tablesPrefixed (cons
									/* inherit isOuter from the subquery's join type, not from inner table */
									(match (car tablesPrefixed) '(a s t io je) (list a s t scalar_first_table_is_outer joinexpr2))
									(cdr tablesPrefixed)))
							)
							(define flatten_order_scalar_helpers_by_local_dependencies (lambda (tbls) (begin
								(define tbl_aliases (map tbls (lambda (td) (match td '(a _ _ _ _) a nil))))
								(define td_alias (lambda (td) (match td '(a _ _ _ _) a nil)))
								(define nested_helper_alias? (lambda (alias_)
									(and (string? alias_) (> (count (split alias_ "\0")) 2))))
								(define reorder_leading_nested_scalar_cycle (lambda (items) (match items
									(cons parent_td (cons helper_td rest_items))
									(match parent_td
										'(parent_a parent_s parent_t parent_io parent_je)
										(match helper_td
											'(helper_a helper_s helper_t helper_io helper_je)
											(if (and
												(nested_helper_alias? helper_a)
												(not (nested_helper_alias? parent_a))
												(not (nil? parent_je))
												(not (nil? helper_je))
												(expr_refs_alias? parent_je helper_a)
												(not (expr_refs_alias? parent_je parent_a))
												(expr_refs_alias? helper_je parent_a))
												(cons
													(list helper_a helper_s helper_t helper_io parent_je)
													(cons
														(list parent_a parent_s parent_t parent_io helper_je)
														rest_items))
												items)
											items)
										items)
									_ items)))
								(define td_local_deps (lambda (td)
									(match td
										'(a _ _ _ je)
										(if (nil? je)
											'()
											(filter (flatten_expr_ref_aliases je) (lambda (ref_alias)
												(and
													(not (same_flatten_alias? ref_alias a))
													(has? tbl_aliases ref_alias)))))
										'())))
								(define deps_satisfied? (lambda (td emitted_aliases)
									(reduce (td_local_deps td) (lambda (ok dep)
										(and ok (has? emitted_aliases dep)))
										true)))
								(define topo_state (reduce tbls (lambda (state _)
									(match state
										'(ordered remaining emitted_aliases)
										(begin
											(define ready (filter remaining (lambda (td)
												(deps_satisfied? td emitted_aliases))))
											(if (equal? ready '())
												state
												(list
													(merge ordered ready)
													(filter remaining (lambda (td) (not (has? ready td))))
													(merge emitted_aliases (map ready td_alias)))))
										state))
									(list '() tbls '())))
								(match topo_state
									'(ordered remaining _)
									(if (equal? remaining '())
										ordered
										(merge ordered (reorder_leading_nested_scalar_cycle remaining)))
									tbls))))
							(define tablesPrefixed (if (and isOuter (flatten_scalar_helper_alias? id))
								(flatten_order_scalar_helpers_by_local_dependencies tablesPrefixed)
								tablesPrefixed))
							(define table_aliases_order (if (and isOuter (flatten_scalar_helper_alias? id))
								(map tablesPrefixed (lambda (td) (match td '(a _ _ _ _) a nil)))
								table_aliases_order))
							(define flatten_scalar_corr_cols (lambda (expr alias)
								(merge_unique (filter (map (flatten_and_terms (coalesceNil expr true)) (lambda (part)
									(match part
										'(op lhs rhs)
										(if (flatten_equality_conjunct? part)
											(begin
												(define lhs_refs_alias (expr_refs_alias? lhs alias))
												(define rhs_refs_alias (expr_refs_alias? rhs alias))
												(if (and lhs_refs_alias (not rhs_refs_alias))
													(flatten_extract_columns_for_alias alias lhs)
													(if (and rhs_refs_alias (not lhs_refs_alias))
														(flatten_extract_columns_for_alias alias rhs)
														'())))
											'())
										'())))
									(lambda (cols) (not (equal? cols '())))))))
							(define tag_scalar_no_limit_table (lambda (td)
								(match td
									'(a s t io je)
									(begin
										(define raw_limit_for_tag (match subquery
											'(_ _ _ _ _ _ _ l _) l
											nil))
										(if (and
											isOuter
											(flatten_scalar_helper_alias? id)
											(nil? raw_limit_for_tag)
											(not (qpp-tuple? t))
											(not (scan_tagged_table_needs_scan_order t)))
											(begin
												(define corr_cols (flatten_scalar_corr_cols scalar_projection_joinexpr2 a))
												(define corr_order (map corr_cols (lambda (col)
													(list (list (quote get_column) a false col false) (quote <)))))
												(list a s
													(make_scan_tagged_table t corr_order 2 nil (count corr_cols) 2)
													io je))
											td))
									td)))
							(define attach_join_part_to_alias (lambda (tbls target part)
								(map tbls (lambda (td) (match td
									'(a s t io je)
									(if (same_flatten_alias? a target)
										(list a s t io (add_join_part je part))
										td)
									td)))))
							(if (and isOuter (flatten_scalar_helper_alias? id) (not (nil? scalar_projection_joinexpr2)) (not (equal? scalar_projection_joinexpr2 true)))
								(set tablesPrefixed
									(reduce (flatten_and_terms scalar_projection_joinexpr2) (lambda (acc part)
										(if (not (flatten_equality_conjunct? part))
											acc
											(begin
												(define target (condition_target_alias part))
												(if (nil? target) acc (attach_join_part_to_alias acc target part)))))
										tablesPrefixed)))
							(if (and isOuter (flatten_scalar_helper_alias? id) (not (nil? tablesPrefixed)))
								(set tablesPrefixed (cons
									(tag_scalar_no_limit_table (car tablesPrefixed))
									(cdr tablesPrefixed))))
							(define flattened_table_aliases (map tablesPrefixed (lambda (td) (match td '(alias _ _ _ _) alias ""))))
							/* check for dangling get_column refs that point to a
							flattened alias prefix but not an actual flattened table.
							Skip opaque scopes — their inner get_column refs belong
							to the lowered scan, not the relational alias domain. */
							(define extract_visible_get_columns (lambda (expr)
								(match expr
									'((symbol get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
									'((quote get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
									(cons sym args) (if (_is_opaque_scope_sym sym) '()
										(merge (map args extract_visible_get_columns)))
									'()
								)
							))
							(define has_dangling_flatten_ref (lambda (expr)
								(reduce (extract_visible_get_columns expr) (lambda (acc mc)
									(or acc (match mc
										'(name '((symbol get_column) alias_ _ _ _))
										(begin
											(define alias_str (string alias_))
											(and (strlike alias_str (concat id "\0%"))
												(not (has? flattened_table_aliases alias_str))))
										'(name '((quote get_column) alias_ _ _ _))
										(begin
											(define alias_str (string alias_))
											(and (strlike alias_str (concat id "\0%"))
												(not (has? flattened_table_aliases alias_str))))
										false)))
									false)))
							(define flatten_referenced_cols (merge_unique (list
								(extract_columns_for_tblvar id fields)
								(extract_columns_for_tblvar id condition)
								(extract_columns_for_tblvar id (coalesceNil having true))
								(merge (map (coalesceNil order '()) (lambda (o) (extract_columns_for_tblvar id o))))
								(merge (map (coalesceNil group '()) (lambda (gexpr) (extract_columns_for_tblvar id gexpr))))
								(extract_unqualified_columns fields)
								(extract_unqualified_columns condition)
								(extract_unqualified_columns (coalesceNil having true))
								(merge (map (coalesceNil order '()) (lambda (o) (extract_unqualified_columns o))))
								(merge (map (coalesceNil group '()) (lambda (gexpr) (extract_unqualified_columns gexpr))))
								(merge (map tables (lambda (td) (match td
									'(_ _ _ _ outer_joinexpr) (if (nil? outer_joinexpr) '()
										(extract_columns_for_tblvar id outer_joinexpr))
									'())))))))
							(define flatten_uses_subquery_wildcard (or
								(expr_has_unqualified_wildcard_ref fields)
								(expr_has_tblvar_wildcard_ref id fields)
								(expr_has_tblvar_wildcard_ref id condition)
								(expr_has_tblvar_wildcard_ref id (coalesceNil having true))
								(reduce (coalesceNil order '()) (lambda (acc o) (or acc (expr_has_tblvar_wildcard_ref id o))) false)
								(reduce (coalesceNil group '()) (lambda (acc gexpr) (or acc (expr_has_tblvar_wildcard_ref id gexpr))) false)))
							(define pruned_fields2 (if flatten_uses_subquery_wildcard
								fields2
								(filter_assoc fields2 (lambda (k v)
									(reduce flatten_referenced_cols (lambda (keep refcol)
										(or keep (equal?? refcol k)))
										false)))))
							(define flatten_has_dangling_output_ref
								(reduce_assoc pruned_fields2 (lambda (acc _k v)
									(or acc (has_dangling_flatten_ref (replace_column_alias v))))
									false))
							(define flatten_has_dangling_join_ref
								(or
									(has_dangling_flatten_ref projection_joinexpr2)
									(has_dangling_flatten_ref condition2_transformed)))
							(define expr_contains_materialized_helper (lambda (expr) (match expr
								_ (if (materialized-source? expr)
									true
									(match expr
										(cons sym args) (or
											(expr_contains_materialized_helper sym)
											(reduce args (lambda (acc arg) (or acc (expr_contains_materialized_helper arg))) false))
										false)))))
							(define flatten_helper_projection_aliases (map (filter tables2 (lambda (td) (match td
								'(alias _ ttbl _ _) (scan_tagged_table_needs_scan_order ttbl)
								false))) (lambda (td) (match td '(alias _ _ _ _) alias ""))))
							(define flatten_materialized_helper_aliases (map (filter tables2 (lambda (td) (match td
								'(alias _ ttbl _ _) (materialized-source? ttbl)
								false))) (lambda (td) (match td '(alias _ _ _ _) alias ""))))
							(define expr_contains_scan_tagged_helper (lambda (expr)
								(reduce (extract_tblvars expr) (lambda (acc alias_)
									(or acc
										(has? flatten_helper_projection_aliases alias_)
										(has? flatten_materialized_helper_aliases alias_)))
									false)))
							(define flatten_has_helper_backed_projection
								(reduce_assoc pruned_fields2 (lambda (acc _k v)
									(or acc
										(expr_contains_materialized_helper v)
										(expr_contains_scan_tagged_helper v)))
									false))
							(define groups2_present (and (not (nil? groups2)) (not (equal? groups2 '()))))
							(define aggregate_refs_subquery_alias (lambda (expr)
								(reduce (extract_aggregates expr) (lambda (acc ag)
									(or acc (match ag
										'(agg_expr _ _)
										(reduce (extract_tblvars agg_expr) (lambda (found tv) (or found (equal?? tv id))) false)
										false)))
									false)))
							(define correlated_inner_select_refs_subquery_alias (lambda (expr)
								(and
									(_contains_inner_select_marker expr)
									(reduce (extract_tblvars expr) (lambda (found tv)
										(or found (equal?? tv id)))
										false))))
							(define subquery_contains_inner_select_marker
								(or
									(reduce_assoc fields2 (lambda (acc _k v)
										(or acc (_contains_inner_select_marker v))) false)
									(_contains_inner_select_marker condition2)
									(reduce (coalesceNil groups2 '()) (lambda (acc stage)
										(or acc
											(reduce (coalesceNil (stage_group_cols stage) '()) (lambda (found gexpr)
												(or found (_contains_inner_select_marker gexpr))) false)
											(_contains_inner_select_marker (stage_having_expr stage))
											(reduce (coalesceNil (stage_order_list stage) '()) (lambda (found order_item)
												(or found (match order_item
													'(col _dir) (_contains_inner_select_marker col)
													(_contains_inner_select_marker order_item)))) false)))
										false)))
							(define raw_subquery_contains_inner_select_marker
								(_contains_inner_select_marker subquery))
							(define subquery_contains_inner_select_marker_outside_condition
								(or
									(reduce_assoc fields2 (lambda (acc _k v)
										(or acc (_contains_inner_select_marker v))) false)
									(reduce (coalesceNil groups2 '()) (lambda (acc stage)
										(or acc
											(reduce (coalesceNil (stage_group_cols stage) '()) (lambda (found gexpr)
												(or found (_contains_inner_select_marker gexpr))) false)
											(_contains_inner_select_marker (stage_having_expr stage))
											(reduce (coalesceNil (stage_order_list stage) '()) (lambda (found order_item)
												(or found (match order_item
													'(col _dir) (_contains_inner_select_marker col)
													(_contains_inner_select_marker order_item)))) false)))
										false)))
							(define grouped_or_helper_boundary
								(or groups2_present flatten_has_helper_backed_projection))
							(define outer_uses_subquery_group_boundary (or
								(reduce_assoc fields (lambda (acc _k v) (or acc
									(aggregate_refs_subquery_alias v)
									(and grouped_or_helper_boundary (correlated_inner_select_refs_subquery_alias v)))) false)
								(and grouped_or_helper_boundary (correlated_inner_select_refs_subquery_alias (coalesceNil condition true)))
								(reduce (coalesceNil group '()) (lambda (acc gexpr)
									(or acc (reduce (extract_tblvars gexpr) (lambda (found tv) (or found (equal?? tv id))) false)))
									false)
								(reduce (coalesceNil order '()) (lambda (acc o) (or acc (match o
									'(col _dir) (or
										(aggregate_refs_subquery_alias col)
										(and grouped_or_helper_boundary (correlated_inner_select_refs_subquery_alias col)))
									false)))
									false)
								(or
									(aggregate_refs_subquery_alias (coalesceNil having true))
									(and grouped_or_helper_boundary outer_has_group_stage
										(reduce (extract_tblvars (coalesceNil having true))
											(lambda (found tv) (or found (equal?? tv id))) false))
									(and grouped_or_helper_boundary (correlated_inner_select_refs_subquery_alias (coalesceNil having true))))))
							/* window functions in subquery require materialization (cannot flatten because window needs its own ordering).
							The Neumann lowerer may wrap a window-derived source in a scalar helper, so inspect nested
							FROM-derived tuples as well, not only this wrapper's direct projection fields. */
							(define subquery_has_window (qpp-tuple-contains-window-recursive? subquery))
							/* TODO: group+order+limit+offset -> ordered scan list with aggregation layers (to avoid materialization) */
							/* Note: flat defines avoid nested begin scopes — (set) only updates the innermost Nodefine=false env */
							(define flatten_groups2 (if groups2_present
								(filter groups2 (lambda (stage)
									(or
										(not (nil? (stage_limit_val stage)))
										(not (nil? (stage_offset_val stage)))
										(not (nil? (stage_partition_aliases stage))))))
								'()))
							(define unsupported_groups (if groups2_present
								(reduce groups2 (lambda (acc stage)
									(or acc (stage_has_group_boundary stage))
								) false)
								false))
							/* Nested derived-table flattening must not descend into precomputed runtime blocks
							from scalar subselects when they appear in JOIN conditions; those scopes carry lowered
							scan structures that break under alias-renaming. */
							(define subquery_has_runtime_joinexpr (reduce tables2 (lambda (acc tbl_desc) (or acc (match tbl_desc
								'(_ _ _ _ inner_joinexpr) (if (nil? inner_joinexpr) false (expr_has_opaque_scope (replace_column_alias inner_joinexpr)))
								_ false))) false))
							(define outer_has_non_group_stage (or
								(not (equal? (coalesceNil order '()) '()))
								(not (nil? limit))
								(not (nil? offset))))
							(define outer_has_group_stage (or
								(not (nil? group))
								(not (nil? having))
								(reduce_assoc fields (lambda (acc _k v) (or acc (not (equal? (extract_aggregates v) '())))) false)))
							/* Force materialize when ANOTHER derived at this outer level
							has ALREADY been flattened (and pushed its stage into
							scalar_subquery_cache "groups"). The legacy build_queryplan_inner
							recursion only supports ONE non-group "tail" stage; two
							flattened sibling scalars produce two such stages and trip
							the "non-group stage must be last" assertion at line 8077.
							By materializing the second+ sibling, we avoid pushing its
							stage up — the inner becomes a temp table whose scan is
							a self-contained block. */
							(define previously_flattened_stages_present
								(not (equal? (coalesceNil (scalar_subquery_cache "groups") '()) '())))
							(define scalar_helper_alias? (lambda (alias)
								(scalar_helper_root_alias? alias)))
							(define scalar_helper_referenced_by_outer_expr
								(or
									(reduce_assoc fields (lambda (acc _k v)
										(or acc (has? (extract_tblvars v) id))) false)
									(has? (extract_tblvars (coalesceNil condition true)) id)
									(reduce (coalesceNil group '()) (lambda (acc gexpr)
										(or acc (has? (extract_tblvars gexpr) id))) false)
									(reduce (coalesceNil order '()) (lambda (acc o) (or acc (match o
										'(col _dir) (has? (extract_tblvars col) id)
										false))) false)
									(has? (extract_tblvars (coalesceNil having true)) id)))
							(define scalar_helper_limit_one_no_group
								(and groups2_present
									(reduce (coalesceNil groups2 '()) (lambda (ok stage)
										(and ok
											(equal? (coalesceNil (stage_group_cols stage) '()) '())
											(nil? (stage_having_expr stage))
											(not (stage_is_dedup stage))
											(equal? (stage_limit_val stage) 1)))
										true)))
							(define scalar_helper_group_can_flatten
								(and (scalar_helper_alias? id)
									scalar_helper_referenced_by_outer_expr
									(or outer_has_group_stage scalar_helper_limit_one_no_group)
									(not flatten_has_dangling_join_ref)
									(not subquery_has_window)
									(not unsupported_groups)))
							(define use_materialize (or
								subquery_has_window
								(and groups2_present (not scalar_helper_group_can_flatten))
								unsupported_groups
								flatten_has_dangling_output_ref
								flatten_has_dangling_join_ref
								(and subquery_has_runtime_joinexpr (not scalar_helper_group_can_flatten))
								(and (not (equal? flatten_groups2 '())) outer_has_non_group_stage)
								(and (not (equal? flatten_groups2 '())) outer_has_group_stage (not scalar_helper_group_can_flatten))
								(and (not (equal? flatten_groups2 '())) (not (equal? (coalesceNil outer_schemas '()) '())))
								(and subquery_contains_inner_select_marker outer_has_non_group_stage)
								(and outer_has_group_stage subquery_contains_inner_select_marker
									subquery_contains_inner_select_marker_outside_condition
									(reduce_assoc fields (lambda (acc _k v)
										(or acc (aggregate_refs_subquery_alias v))) false))
								(and flatten_has_helper_backed_projection
									subquery_contains_inner_select_marker_outside_condition
									outer_uses_subquery_group_boundary)
								(and (not (equal? flatten_groups2 '())) previously_flattened_stages_present)))
							(define use_materialize
								(if scalar_helper_group_can_flatten false use_materialize))
							(define condition_only_inner_select_can_flatten
								(and
									subquery_contains_inner_select_marker
									(not subquery_contains_inner_select_marker_outside_condition)
									(not subquery_has_window)
									(not flatten_has_dangling_output_ref)
									(not flatten_has_dangling_join_ref)))
							(define condition_only_non_scalar_derived
								(and
									(not (scalar_helper_alias? id))
									(not raw_subquery_contains_inner_select_marker)
									(not subquery_contains_inner_select_marker_outside_condition)
									(not subquery_has_window)
									(not flatten_has_dangling_output_ref)
									(not flatten_has_dangling_join_ref)))
							(define use_materialize
								(if condition_only_inner_select_can_flatten false use_materialize))
							(define use_materialize
								(if (and
									(not (scalar_helper_alias? id))
									outer_has_group_stage
									(not (or
										(nil? (match subquery '(_ _ _ raw_condition _ _ _ _ _) raw_condition true))
										(equal?? (match subquery '(_ _ _ raw_condition _ _ _ _ _) raw_condition true) true)
										(equal? (match subquery '(_ _ _ raw_condition _ _ _ _ _) raw_condition true) (quote true))))
									(not subquery_has_window))
									true
									use_materialize))
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
							(define flatten_stage_alias (lambda (alias)
								(if (nil? alias) nil
									(if (or (not (nil? (schemas2 alias)))
										(and (string? alias) (not (nil? (schemas2 (symbol alias))))))
										(concat id "\0" alias)
										alias))))
							(define scalar_helper_flatten_aliases (map tablesPrefixed (lambda (td)
								(match td '(alias _ _ _ _) alias nil))))
							(define scalar_helper_scope_flatten_stage (lambda (stage) (begin
								(define _sg (coalesceNil (stage_group_cols stage) '()))
								(define _spa (stage_partition_aliases stage))
								(if (and
									scalar_helper_group_can_flatten
									(equal? (count scalar_helper_flatten_aliases) 1)
									(or (nil? _spa) (equal? (coalesceNil _spa '()) '()))
									(or (nil? _sg) (equal? _sg '())))
									(stage_rebuild_with_meta
										stage
										(make_stage
											'()
											nil
											(coalesceNil (stage_order_list stage) '())
											(coalesceNil (stage_limit_partition_cols stage) 0)
											(stage_limit_val stage)
											(stage_offset_val stage)
											false
											scalar_helper_flatten_aliases
											(stage_init_code stage)
											(stage_condition stage)
											(stage_once_limit stage))
										(lambda (expr) expr)
										(lambda (alias) alias))
									stage))))
							/* pass-through stage semantics stay on the flattened plan.
							Pure inner ORDER BY without LIMIT/OFFSET is dropped; the outer ORDER BY wins. */
							(if (and groups2_present (not use_materialize) (not (equal? flatten_groups2 '())))
								(scalar_subquery_cache "groups" (merge
									(map flatten_groups2 (lambda (stage)
										(rewrite_stage_for_flattened_aliases
											(scalar_helper_scope_flatten_stage stage)
											replace_column_alias
											flatten_stage_alias)))
									(coalesceNil (scalar_subquery_cache "groups") '()))))
							(if use_materialize
								(begin
									(define output_cols_sub (extract_assoc fields2 (lambda (k v) k)))
									(define mat_symbol_suffix (if (scalar_helper_alias? id)
										(fnv_hash id)
										id))
									(define rows_sym (symbol (concat "__mr_" mat_symbol_suffix)))
									(define row_sink_sym (symbol (concat "__ms_" mat_symbol_suffix)))
									(define cnt_sym (symbol (concat "__mc_" mat_symbol_suffix)))
									(define scalar_materialize_default?
										(and (string? id)
											(>= (strlen id) 14)
											(equal? (substr id 0 14) "domain_scalar_")
											(equal? (count output_cols_sub) 1)))
									(define scalar_default_row
										(cons (quote list)
											(reduce output_cols_sub (lambda (acc col)
												(merge acc (list col nil))) '())))
									/* Build the materialized inner plan from the already untangled IR of
									this subquery. Replanning from the raw AST here can drift from the
									current alias/scope environment and reintroduce wrapper-specific
									regressions. */
									(define direct_get_column_alias (lambda (expr) (match expr
										'((symbol get_column) tv _ _ _) tv
										'((quote get_column) tv _ _ _) tv
										nil)))
									(define scalar_materialize_key_exprs
										(if (scalar_helper_alias? id)
											(reduce_assoc fields2 (lambda (acc k v)
												(if (and
													(not (nil? (direct_get_column_alias v)))
													(or
														(and (string? k) (>= (strlen k) 4)
															(equal? (substr k 0 4) "__kt"))
														(not (equal? k "value"))))
													(merge acc (list v))
													acc)) '())
											'()))
									(define scalar_materialize_key_alias
										(if (equal? (count scalar_materialize_key_exprs) 0)
											nil
											(direct_get_column_alias (car scalar_materialize_key_exprs))))
									(define scalar_materialize_keys_same_alias
										(reduce scalar_materialize_key_exprs (lambda (acc key_expr)
											(and acc (equal? (direct_get_column_alias key_expr)
												scalar_materialize_key_alias))) true))
									(define scalar_materialize_order_aliases (lambda (order_items)
										(merge_unique (map (coalesceNil order_items '()) (lambda (order_item)
											(match order_item
												'(order_expr order_dir) (extract_tblvars order_expr)
												(extract_tblvars order_item)))))))
									(define scalar_materialize_partition_stage (lambda (stage) (begin
										(define stage_order_items (coalesceNil (stage_order_list stage) '()))
										(define stage_order_aliases (scalar_materialize_order_aliases stage_order_items))
										(define stage_has_no_order (equal? stage_order_items '()))
										(define stage_order_matches_key
											(and (> (count stage_order_items) 0)
												(equal? (count stage_order_aliases) 1)
												(equal? (car stage_order_aliases) scalar_materialize_key_alias)))
										(if (and
											(not (nil? scalar_materialize_key_alias))
											scalar_materialize_keys_same_alias
											(equal? (coalesceNil (stage_group_cols stage) '()) '())
											(nil? (stage_having_expr stage))
											(not (stage_is_dedup stage))
											(equal? (stage_limit_val stage) 1)
											(nil? (stage_offset_val stage))
											(or stage_has_no_order stage_order_matches_key))
											(if stage_has_no_order
												(make_stage
													scalar_materialize_key_exprs
													nil
													'()
													0
													nil
													nil
													false
													(stage_partition_aliases stage)
													(stage_init_code stage)
													(stage_condition stage)
													nil)
												(make_stage
													'()
													nil
													(merge
														(map scalar_materialize_key_exprs (lambda (key_expr)
															(list key_expr (quote <))))
														stage_order_items)
													(count scalar_materialize_key_exprs)
													(stage_limit_val stage)
													(stage_offset_val stage)
													false
													(stage_partition_aliases stage)
													(stage_init_code stage)
													(stage_condition stage)
													(stage_once_limit stage)))
											stage))))
									(define raw_materialized_subquery_limit
										(match subquery
											'(_ _ _ _ _ _ _ raw_limit _) raw_limit
											nil))
									(define raw_materialized_subquery_offset
										(match subquery
											'(_ _ _ _ _ _ _ _ raw_offset) raw_offset
											nil))
									(define materialized_subquery_has_own_limit
										(and
											(not raw_subquery_contains_inner_select_marker)
											(or (not (nil? raw_materialized_subquery_limit))
												(not (nil? raw_materialized_subquery_offset)))))
									(define materialized_subquery_contains_scalar_helper
										(strlike (serialize tables2) "%domain_scalar_%"))
									(define raw_inner_select_local_aliases_from_query (lambda (query) (match query
										'(_ _ raw_fields raw_condition raw_group raw_having raw_order _ _)
										(merge_unique (list
											(reduce_assoc raw_fields (lambda (acc _field_name field_expr)
												(merge acc (raw_inner_select_local_aliases_from_expr field_expr))) '())
											(raw_inner_select_local_aliases_from_expr raw_condition)
											(reduce (coalesceNil raw_group '()) (lambda (acc group_expr)
												(merge acc (raw_inner_select_local_aliases_from_expr group_expr))) '())
											(raw_inner_select_local_aliases_from_expr raw_having)
											(reduce (coalesceNil raw_order '()) (lambda (acc order_item)
												(merge acc (match order_item
													'(order_expr _order_dir) (raw_inner_select_local_aliases_from_expr order_expr)
													(raw_inner_select_local_aliases_from_expr order_item)))) '())))
										'())))
									(define raw_inner_select_local_aliases_from_expr (lambda (expr) (match expr
										(cons sym args) (begin
											(define kind (inner_select_kind sym))
											(if (nil? kind)
												(reduce args (lambda (acc arg)
													(merge acc (raw_inner_select_local_aliases_from_expr arg))) '())
												(begin
													(define nested_subquery (match kind
														(quote inner_select) (match args
															(cons inner_subquery '()) inner_subquery
															nil)
														(quote inner_select_exists) (match args
															(cons inner_subquery '()) inner_subquery
															nil)
														(quote inner_select_in) (match args
															(cons _target_expr (cons inner_subquery '())) inner_subquery
															nil)
														nil))
													(define marker_extra_exprs (match kind
														(quote inner_select_in) (match args
															(cons target_expr (cons _inner_subquery '())) (list target_expr)
															'())
														'()))
													(if (nil? nested_subquery)
														(reduce args (lambda (acc arg)
															(merge acc (raw_inner_select_local_aliases_from_expr arg))) '())
														(merge_unique (list
															(_raw_query_local_aliases nested_subquery)
															(raw_inner_select_local_aliases_from_query nested_subquery)
															(reduce marker_extra_exprs (lambda (acc marker_expr)
																(merge acc (raw_inner_select_local_aliases_from_expr marker_expr))) '())))))))
										'())))
									(define raw_inner_select_local_aliases
										(raw_inner_select_local_aliases_from_query subquery))
									(define materialized_expr_refs_any_alias (lambda (expr aliases)
										(reduce (extract_tblvars expr) (lambda (found alias_)
											(or found (_alias_in_list aliases alias_)))
											false)))
									(define materialized_trim_nested_scalar_group_stage (lambda (stage) (begin
										(define outer_sources (coalesceNil (stage_outer_sources stage) '()))
										(define nested_inner_sources (filter outer_sources (lambda (src) (match src
											'(outer_alias _outer_col _inner_expr)
											(if (stage_outer_source_expr_tuple? src)
												(materialized_expr_refs_any_alias (nth src 1) raw_inner_select_local_aliases)
												(_alias_in_list raw_inner_select_local_aliases outer_alias))
											false))))
										(define kept_outer_sources (filter outer_sources (lambda (src) (match src
											'(outer_alias _outer_col _inner_expr)
											(if (stage_outer_source_expr_tuple? src)
												(not (materialized_expr_refs_any_alias (nth src 1) raw_inner_select_local_aliases))
												(not (_alias_in_list raw_inner_select_local_aliases outer_alias)))
											true))))
										(if (or (equal? nested_inner_sources '()) (equal? kept_outer_sources '()))
											stage
											(begin
												(define nested_inner_aliases
													(merge_unique (map nested_inner_sources (lambda (src) (match src
														'(_outer_alias _outer_col inner_expr)
														(if (stage_outer_source_expr_tuple? src)
															(extract_tblvars (nth src 2))
															(extract_tblvars inner_expr))
														'())))))
												(define trimmed_group_cols
													(filter (coalesceNil (stage_group_cols stage) '()) (lambda (group_expr)
														(not (materialized_expr_refs_any_alias group_expr nested_inner_aliases)))))
												(if (equal? trimmed_group_cols (coalesceNil (stage_group_cols stage) '()))
													stage
													(stage_with_outer_sources
														(stage_with_cache_query
															(stage_with_cache_policy
																(make_stage
																	trimmed_group_cols
																	(stage_having_expr stage)
																	(stage_order_list stage)
																	(stage_limit_partition_cols stage)
																	(stage_limit_val stage)
																	(stage_offset_val stage)
																	(stage_is_dedup stage)
																	(stage_partition_aliases stage)
																	(stage_init_code stage)
																	(stage_condition stage)
																	(stage_once_limit stage))
																(stage_cache_policy stage))
															(stage_cache_query stage))
														kept_outer_sources)))))))
									(define groups2_materialized_boundary
										(coalesceNil groups2 '()))
									(define materialized_output_aliases
										(merge_unique (reduce_assoc fields2 (lambda (acc _field_name field_expr)
											(merge acc (extract_tblvars field_expr))) '())))
									(define materialized_strip_outer_table_limit_leak (lambda (td) (match td
										'(alias_ schema_ table_ is_outer_ join_expr_)
										(if (and
											(has? materialized_output_aliases alias_)
											(not (_alias_in_list raw_inner_select_local_aliases alias_))
											(or (nil? join_expr_)
												(equal? join_expr_ true)
												(equal? join_expr_ (quote true)))
											(not (scalar_helper_alias? alias_))
											(scan_tagged_table_needs_scan_order table_)
											(not (nil? (scan_tagged_table_limit table_)))
											(<= (scan_tagged_table_limit table_) 1)
											(or (nil? (scan_tagged_table_offset table_))
												(equal? (scan_tagged_table_offset table_) 0))
											(equal? (coalesceNil (scan_tagged_table_partition_cols table_) 0) 0))
											(list alias_ schema_ (scan_tagged_table_base table_) is_outer_ join_expr_)
											td)
										td)))
									(define tables2_materialized
										(if (and
											(not (scalar_helper_alias? id))
											(not materialized_subquery_has_own_limit)
											raw_subquery_contains_inner_select_marker)
											(map tables2 materialized_strip_outer_table_limit_leak)
											tables2))
									(define groups2_materialized
										(if (and (scalar_helper_alias? id)
											(> (count scalar_materialize_key_exprs) 0))
											(map groups2_materialized_boundary scalar_materialize_partition_stage)
											groups2_materialized_boundary))
									(define groups2_materialized
										(if (and
											(not (scalar_helper_alias? id))
											(not materialized_subquery_has_own_limit)
											raw_subquery_contains_inner_select_marker)
											(map (coalesceNil groups2_materialized '()) materialized_trim_nested_scalar_group_stage)
											groups2_materialized))
									(define materialized_strip_nested_scalar_stage_limit (lambda (stage)
										(if (and
											(not (nil? (stage_limit_val stage)))
											(<= (stage_limit_val stage) 1)
											(or (nil? (stage_offset_val stage))
												(equal? (stage_offset_val stage) 0))
											(equal? (coalesceNil (stage_group_cols stage) '()) '())
											(nil? (stage_having_expr stage))
											(equal? (coalesceNil (stage_order_list stage) '()) '())
											(nil? (stage_partition_aliases stage)))
											(stage_rebuild_with_meta
												stage
												(make_stage
													(stage_group_cols stage)
													(stage_having_expr stage)
													(stage_order_list stage)
													(stage_limit_partition_cols stage)
													nil
													nil
													(stage_is_dedup stage)
													(stage_partition_aliases stage)
													(stage_init_code stage)
													(stage_condition stage)
													nil)
												(lambda (expr) expr)
												(lambda (alias) alias))
											stage)))
									(define groups2_materialized
										(if (and
											(not (scalar_helper_alias? id))
											(not materialized_subquery_has_own_limit)
											materialized_subquery_contains_scalar_helper)
											(map (coalesceNil groups2_materialized '()) materialized_strip_nested_scalar_stage_limit)
											groups2_materialized))
									(define materialized_order_only_stage? (lambda (stage)
										(and
											(equal? (coalesceNil (stage_group_cols stage) '()) '())
											(nil? (stage_having_expr stage))
											(not (equal? (coalesceNil (stage_order_list stage) '()) '()))
											(nil? (stage_limit_val stage))
											(or (nil? (stage_offset_val stage))
												(equal? (stage_offset_val stage) 0))
											(not (stage_is_dedup stage))
											(nil? (stage_partition_aliases stage))
											(nil? (stage_condition stage)))))
									(define groups2_materialized
										(if (and
											(not (scalar_helper_alias? id))
											(not materialized_subquery_has_own_limit))
											(filter (coalesceNil groups2_materialized '()) (lambda (stage)
												(not (materialized_order_only_stage? stage))))
											groups2_materialized))
									(define groups2_materialized
										(if groups2_present
											(map (coalesceNil groups2_materialized '()) (lambda (stage)
												(stage_set stage (quote materialized-output-boundary) true)))
											groups2_materialized))
									(define mat_fields2_base (map_assoc fields2 (lambda (k v) (replace_find_column2 v))))
									(define scalar_payload_passthrough_fields
										(if (scalar_helper_alias? id)
											(reduce tables2_materialized (lambda (acc td) (match td
												'(alias_ _ source_ _ _)
												(if (or (scalar_helper_alias? alias_) (scalar_helper_alias? source_))
													(reduce (coalesceNil (schemas2 alias_) '()) (lambda (acc2 coldef) (begin
														(define col (coldef "Field"))
														(if (and
															(string? col)
															(>= (strlen col) 10)
															(equal? (substr col 0 10) "__payload_")
															(not (has_assoc? acc2 col))
															(not (has_assoc? mat_fields2_base col)))
															(set_assoc acc2 col (list (quote get_column) alias_ false col false))
															acc2)))
														acc)
													acc)
												_ acc))
												'())
											'()))
									(define mat_fields2 (merge mat_fields2_base scalar_payload_passthrough_fields))
									(define mat_condition2 (replace_find_column2 (coalesceNil condition2 true)))
									(define mat_init_stmts (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
									(define materialized_logical_has_runtime_scalar
										(strlike (serialize mat_fields2) "%__scalar_promise_%"))
									(define mat_logical_term (list
										(quote select_core_term)
										schema2 tables2_materialized mat_fields2 mat_condition2 groups2_materialized schemas2 replace_find_column2 mat_init_stmts))
									(define materialized_subquery_has_scalar_helper_table
										(or
											materialized_subquery_contains_scalar_helper
											(reduce tables2_materialized (lambda (found td)
												(or found (match td
													'(alias_ _ source_ _ _)
													(or
														(scalar_helper_alias? alias_)
														(scalar_helper_alias? source_))
													false)))
												false)))
									(define raw_materialized_subquery_order
										(match subquery
											'(_ _ _ _ _ _ raw_order _ _) raw_order
											nil))
									(define raw_materialized_subquery_condition
										(match subquery
											'(_ _ _ raw_condition _ _ _ _ _) raw_condition
											true))
									(define materialized_stage_order_items
										raw_materialized_subquery_order)
									(define materialized_order_expr_to_output (lambda (expr)
										(coalesce
											(reduce_assoc mat_fields2 (lambda (found k v)
												(if (not (nil? found))
													found
													(if (equal?? expr v)
														(list (quote get_column) id false k false)
														nil)))
												nil)
											(match expr
												'((symbol get_column) _ _ col _)
												(if (nil? (fields2_lookup col))
													nil
													(list (quote get_column) id false col false))
												'((quote get_column) _ _ col _)
												(if (nil? (fields2_lookup col))
													nil
													(list (quote get_column) id false col false))
												nil))))
									(define materialized_output_order
										(if (nil? materialized_stage_order_items)
											nil
											(filter (map materialized_stage_order_items (lambda (order_item) (match order_item
												'(order_expr order_dir)
												(begin
													(define output_expr (materialized_order_expr_to_output order_expr))
													(if (nil? output_expr) nil (list output_expr order_dir)))
												nil)))
												(lambda (x) (not (nil? x))))))
									(define materialized_output_order_valid
										(and
											(not (scalar_helper_alias? id))
											(not (nil? materialized_stage_order_items))
											(equal? (count materialized_output_order) (count materialized_stage_order_items))))
									(define mat_row_plan
										(if (or
											(and
												condition_only_non_scalar_derived
												(not raw_subquery_contains_inner_select_marker)
												(not subquery_contains_inner_select_marker)
												(not materialized_logical_has_runtime_scalar)
												(not materialized_subquery_has_scalar_helper_table)))
											(build_queryplan_term_with_sink subquery
												(list (quote callback) row_sink_sym))
											(build_queryplan_term_from_logical_with_sink mat_logical_term
												(list (quote callback) row_sink_sym))))
									(define mat_source (materialized-subquery-source id mat_logical_term))
									(define mat_scan_source
										(if materialized_output_order_valid
											(make_scan_tagged_table mat_source materialized_output_order nil nil 0 nil)
											mat_source))
									(define materialized_row_nonempty_terms
										(map output_cols_sub (lambda (col)
											(list (quote nil?) (list (quote get_assoc) (symbol "item") col)))))
									(define materialized_row_nonempty_filter
										(if (or
											(scalar_helper_alias? id)
											(nil? raw_materialized_subquery_condition)
											(equal?? raw_materialized_subquery_condition true)
											(equal? materialized_row_nonempty_terms '()))
											true
											(list (quote not)
												(if (equal? (count materialized_row_nonempty_terms) 1)
													(car materialized_row_nonempty_terms)
													(cons (quote and) materialized_row_nonempty_terms)))))
									(define mat_init (materialized-subquery-init id mat_logical_term
										(if scalar_materialize_default?
											(planner_collect_rows_default_ast rows_sym row_sink_sym (symbol "item")
												mat_row_plan
												mat_limit
												cnt_sym
												(list (quote list) scalar_default_row))
											(planner_collect_rows_ast rows_sym row_sink_sym (symbol "item")
												mat_row_plan
												mat_limit
												cnt_sym
												materialized_row_nonempty_filter))))
									(scalar_subquery_cache "init" (merge (coalesceNil (scalar_subquery_cache "init") '())
										(list mat_init)))
									(define mat_schema_def
										(register_materialized_subquery_metadata mat_source mat_fields2
											(or outer_uses_subquery_group_boundary groups2_present)))
									(list
										(list (list id schemax mat_scan_source isOuter joinexpr))
										'()
										true
										(list id (merge_schema_fields_unique (list mat_schema_def)))
									)
								)
								(begin
									/* for LEFT JOIN: condition2 was integrated into joinexpr, so return true as global filter */
									/* for INNER JOIN: condition2 becomes global filter (can be reordered) */
									(set globalFilter (if isOuter
										(if (flatten_scalar_helper_alias? id)
											(combine_and_terms (list scalar_projection_filter2 scalar_table_filter2))
											true)
										(replace_column_alias condition2)))
									(define _check_inner_select (lambda (expr) (match expr (cons sym args) (if (not (nil? (inner_select_kind sym))) true (reduce args (lambda (a b) (or a (_check_inner_select b))) false)) false)))
									(define planner_key_projection? (lambda (name)
										(and (string? name) (>= (strlen name) 4)
											(equal? (substr name 0 4) "__kt"))))
									(define comparison_op? (lambda (op)
										(or (equal? op (quote >)) (equal? op (symbol >))
											(equal? op (quote <)) (equal? op (symbol <))
											(equal? op (quote >=)) (equal? op (symbol >=))
											(equal? op (quote <=)) (equal? op (symbol <=))
											(equal? op (quote =)) (equal? op (symbol =))
											(equal? op (quote equal?)) (equal? op (symbol equal?))
											(equal? op (quote equal??)) (equal? op (symbol equal??)))))
									(define has_coalesce? (lambda (expr) (match expr
										'((symbol coalesce) _ _) true
										'((quote coalesce) _ _) true
										(cons _ args) (reduce args (lambda (found arg)
											(or found (has_coalesce? arg))) false)
										false)))
									(define null_tolerant_projection? (lambda (expr) (match expr
										'((symbol coalesce) _ _) true
										'((quote coalesce) _ _) true
										'((symbol not) arg) (null_tolerant_projection? arg)
										'((quote not) arg) (null_tolerant_projection? arg)
										'(op _ _) (and (comparison_op? op) (has_coalesce? expr))
										false)))
									(define wrap_outer_join_projection (lambda (name expr)
										(if (and (not (planner_key_projection? name))
											(not (null_tolerant_projection? expr))
											isOuter (not (equal? joinexpr true)) (not (nil? projection_joinexpr2)) (not (equal? projection_joinexpr2 true)) (not (_check_inner_select projection_joinexpr2)))
											(list (quote if) projection_joinexpr2 expr nil)
											expr)))
									(list tablesPrefixed (list id (map_assoc pruned_fields2 (lambda (k v) (wrap_outer_join_projection k (replace_column_alias v))))) globalFilter (merge (list id (extract_assoc pruned_fields2 (lambda (k v) (list "Field" k "Type" "any" "Expr" (replace_column_alias v))))) (merge (extract_assoc schemas2 (lambda (k v) (list (concat id "\0" k) v))))))
								)
							)
						) (error "non matching return value for untangle_query"))
					)
				)
				(error (concat "unknown tabledesc: " tbldesc))
	)))))))
	(set tablesList (car zipped))
	(set renameList (car (cdr zipped)))
	(set conditionList (car (cdr (cdr zipped))))
	(set schemasList (car (cdr (cdr (cdr zipped)))))
	/* schemas is an assoc array from alias -> columnlist */
	/* rewrite a flat table list according to inner selects */
	(set renamelist (merge renameList))
	(set tables (merge tablesList))
	(set schemas (merge_schema_binding_groups schemasList))

	/* global WHERE stays separate from per-table joinexpr (ON). */
	(set condition (coalesceNil condition true))

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
	(define schema_alias_variants (lambda (alias)
		(reduce (filter (list
			alias
			(visible_occurrence_alias alias)
			(if (string? alias) (sanitize_temp_name alias) nil)
			(if (string? (visible_occurrence_alias alias)) (sanitize_temp_name (visible_occurrence_alias alias)) nil))
			(lambda (x) (not (nil? x))))
			(lambda (acc alias_v) (append_unique acc alias_v))
			'())))
	(define schema_alias_matches (lambda (query_alias schema_alias ti)
		(reduce (schema_alias_variants schema_alias) (lambda (matched alias_v)
			(or matched ((if ti equal?? equal?) query_alias alias_v)))
			false)))
	(define replace_outer_get_columns (lambda (expr) (match expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar)
			(symbol (concat "__unresolved__." col))
			(symbol (concat tblvar "." col)))
		'((quote get_column) tblvar _ col _) (if (nil? tblvar)
			(symbol (concat "__unresolved__." col))
			(symbol (concat tblvar "." col)))
		(cons sym args) (cons sym (map args replace_outer_get_columns))
		expr)))
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
								(if (and (schema_alias_matches alias_ alias ti)
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
		Main tables have no ':' prefix and no domain_scalar_ prefix in their alias. */
		'((symbol get_column) nil _ "*" _) expr
		'((quote get_column) nil _ "*" _) expr
		'((symbol get_column) _ _ "*" _) expr
		'((quote get_column) _ _ "*" _) expr
		'((symbol outer) inner) (list (quote outer) (replace_outer_get_columns inner))
		'((quote outer) inner) (list (quote outer) (replace_outer_get_columns inner))
		'((symbol get_column) nil _ col ci) (begin
			/* First try main tables (aliases without ':' or domain_scalar_ prefix) */
			(define _is_main_alias (lambda (alias) (begin
				(define s (string alias))
				(and (not (strlike s "%:%"))
					(not (strlike s "%\0%"))
					(not (and (>= (strlen s) 14) (equal? (substr s 0 14) "domain_scalar_")))))))
			(define main_match (reduce_assoc schemas (lambda (a alias cols)
				(if (and (_is_main_alias alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias a)) nil))
			/* If not found in main tables, try subquery tables (aliases with ':') */
			(define any_match (if (nil? main_match)
				(reduce_assoc schemas (lambda (a alias cols)
					(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)
						alias a)) nil)
				main_match))
			(define _kt_source_col (if (and (string? col) (>= (strlen col) 5)
				(equal? (substr col 0 5) "__kt_"))
				(substr col 5 (- (strlen col) 5))
				col))
			(define _kt_unqualified_match_state
				(if (or (not (nil? any_match)) (equal? _kt_source_col col))
					(list any_match 0)
					(reduce_assoc schemas (lambda (state alias cols)
						(match state
							'(found count)
							(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") _kt_source_col))) false)
								(list alias (+ count 1))
								state)
							_ state))
						(list nil 0))))
			(define any_match (if (or (not (nil? any_match)) (equal? _kt_source_col col))
				any_match
				(match _kt_unqualified_match_state
					'(found count) (if (equal? count 1) found nil)
					_ nil)))
			(begin
				(define resolved_alias (coalesce any_match (error (concat "column " col " does not exist in tables"))))
				(define canonical_col (if ci (coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col) col))
				(define canonical_col (if (and (equal? canonical_col col)
					(not (equal? _kt_source_col col))
					(reduce (schemas resolved_alias) (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") _kt_source_col))) false))
					_kt_source_col
					canonical_col))
				'((quote get_column) resolved_alias false canonical_col false))
		)
		'((symbol get_column) alias_ ti col ci) (begin
			(define _kt_source_col (if (and (string? col) (>= (strlen col) 5)
				(equal? (substr col 0 5) "__kt_"))
				(substr col 5 (- (strlen col) 5))
				col))
			(define _alias_is_scalar_helper (lambda (alias)
				(and (string? alias)
					(>= (strlen alias) 14)
					(equal? (substr alias 0 14) "domain_scalar_"))))
			(define resolved_alias (reduce_assoc schemas (lambda (a alias cols)
				(if (and (schema_alias_matches alias_ alias ti)
					(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias
					a))
				nil))
			(define resolved_alias (if (or (not (nil? resolved_alias))
				(equal? _kt_source_col col)
				(not (_alias_is_scalar_helper alias_)))
				resolved_alias
				(reduce_assoc schemas (lambda (a alias cols)
					(if (and (schema_alias_matches alias_ alias ti)
						(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") _kt_source_col))) false))
						alias
						a))
					nil)))
			(if (nil? resolved_alias)
				expr
				(begin
					(define canonical_col (if ci
						(coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil)
							(reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") _kt_source_col) (coldef "Field") nil))) nil)
							col)
						(if (and (not (equal? _kt_source_col col))
							(reduce (schemas resolved_alias) (lambda (a coldef) (or a (equal? (coldef "Field") _kt_source_col))) false))
							_kt_source_col
							col)))
					'((quote get_column) resolved_alias false canonical_col false))))
		/* omit strict failure for false/false refs: freshly created temp columns are
		allowed to pass through unresolved until their stage materializes them */
		'((quote get_column) alias_ ti col ci) (begin
			(define _kt_source_col (if (and (string? col) (>= (strlen col) 5)
				(equal? (substr col 0 5) "__kt_"))
				(substr col 5 (- (strlen col) 5))
				col))
			(define _alias_is_scalar_helper (lambda (alias)
				(and (string? alias)
					(>= (strlen alias) 14)
					(equal? (substr alias 0 14) "domain_scalar_"))))
			(define resolved_alias (reduce_assoc schemas (lambda (a alias cols)
				(if (and (schema_alias_matches alias_ alias ti)
					(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias
					a))
				nil))
			(define resolved_alias (if (or (not (nil? resolved_alias))
				(equal? _kt_source_col col)
				(not (_alias_is_scalar_helper alias_)))
				resolved_alias
				(reduce_assoc schemas (lambda (a alias cols)
					(if (and (schema_alias_matches alias_ alias ti)
						(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") _kt_source_col))) false))
						alias
						a))
					nil)))
			(if (nil? resolved_alias)
				expr
				(begin
					(define canonical_col (if ci
						(coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil)
							(reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") _kt_source_col) (coldef "Field") nil))) nil)
							col)
						(if (and (not (equal? _kt_source_col col))
							(reduce (schemas resolved_alias) (lambda (a coldef) (or a (equal? (coldef "Field") _kt_source_col))) false))
							_kt_source_col
							col)))
					'((quote get_column) resolved_alias false canonical_col false))))
		(cons sym args) /* function call */ (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args replace_find_column)))
		expr
	)))

	/* pass full schema chain (current + ancestors) so nested subselects can resolve grandparent refs */
	(define _ris_schemas (merge (schema_bindings_to_flat_list schemas) outer_schema_bindings))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl toisOuter tje)
		(list tv tschema ttbl toisOuter
			(if (nil? tje) nil (replace_inner_selects tje _ris_schemas)))
		td))))
	(set fields (map_assoc fields (lambda (k v) (collect_dependent_scalar_compile_markers v _ris_schemas))))
	(set condition (replace_inner_selects condition _ris_schemas))
	(set group (map group (lambda (g) (replace_inner_selects g _ris_schemas))))
	(set having (begin
		(define _hv_resolved (replace_inner_selects having _ris_schemas))
		/* check if any inner_select nodes remain — HAVING with subqueries
		requires post-group processing which is not yet implemented */
		(define _hv_check (lambda (expr) (match expr
			(cons sym args) (if (not (nil? (inner_select_kind sym))) true
				(reduce args (lambda (a b) (or a (_hv_check b))) false))
			false)))
		(if (and (not (nil? _hv_resolved)) (_hv_check _hv_resolved))
			(error "HAVING with subqueries not yet supported")
			_hv_resolved)))
	(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col _ris_schemas) dir)))))
	/* Freeze visible top-level field refs against the currently visible tables
	before unnested helper tables are merged into schemas. This prevents later
	helper/keytable columns from stealing unrelated outer output bindings. */
	(define freeze_visible_field_refs (lambda (expr) (match expr
		'((symbol get_column) nil _ _ _) (replace_find_column expr)
		'((quote get_column) nil _ _ _) (replace_find_column expr)
		'((symbol get_column) alias_ _ _ _) (if (has_assoc? schemas alias_) (replace_find_column expr) expr)
		'((quote get_column) alias_ _ _ _) (if (has_assoc? schemas alias_) (replace_find_column expr) expr)
		(cons sym args) (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args freeze_visible_field_refs)))
		expr)))
	(set fields (map_assoc fields (lambda (k v) (freeze_visible_field_refs v))))
	(set fields (map_assoc fields (lambda (k v)
		(resolve_dependent_scalar_compile_markers v _ris_schemas))))
	/* integrate unnested scalar subselects from Neumann unnesting.
	Tables from non-aggregate path (direct LEFT JOIN) do NOT need schema updates.
	Tables from aggregate path (materialized derived) DO need schemas for build_queryplan. */
	(define _scalar_subquery_tables (coalesceNil (scalar_subquery_cache "tables") '()))
	(define _scalar_subquery_scalar_tables (coalesceNil (scalar_subquery_cache "scalar_tables") '()))
	(define scalar_generated_helper_alias? (lambda (alias_)
		(and (string? alias_)
			(>= (strlen alias_) 14)
			(equal? (substr alias_ 0 14) "domain_scalar_"))))
	(define scalar_helper_external_join_refs (lambda (td) (match td
		'(tv _ _ _ joinexpr)
		(filter (extract_tblvars joinexpr) (lambda (alias_)
			(and
				(not (equal?? alias_ tv))
				(not (scalar_generated_helper_alias? alias_)))))
		'())))
	(define scalar_query_has_skip_level_scalar_helper (reduce _scalar_subquery_scalar_tables (lambda (found td)
		(or found (> (count (scalar_helper_external_join_refs td)) 1)))
		false))
	/* Deduplication is only sound while the query still carries purely logical
	scalar expressions. Legacy inline fallbacks embed physical runtime code
	(scan, scan_order, !begin, promises) directly into field expressions; alias
	rewrite across those opaque subplans is not semantics-preserving yet. Keep
	those queries on the undeduped helper path until the fallback code is gone. */
	(define scalar_query_has_opaque_runtime (or
		(reduce_assoc fields (lambda (found _k v) (or found (expr_has_opaque_scope v))) false)
		(expr_has_opaque_scope condition)
		(reduce (coalesceNil group '()) (lambda (found expr) (or found (expr_has_opaque_scope expr))) false)
		(expr_has_opaque_scope having)
		(reduce (coalesceNil order '()) (lambda (found order_item) (or found (expr_has_opaque_scope order_item))) false)))
	/* Deduplicate identical scalar projection LEFT JOIN helpers before they
	enter the normal table pipeline. join_reorder can move helpers, but it cannot
	recognize that two domain_scalar_ aliases describe the same LEFT JOIN relation once
	they have already been materialized as distinct table entries. */
	(define scalar_left_join_dedup_key (lambda (td) (match td
		'(tv tschema ttbl isOuter joinexpr) (begin
			(define _dedup_alias_map
				(reduce (alias_lookup_variants tv) (lambda (acc alias_v)
					(set_assoc acc (string alias_v) "__scalar_left_join__"))
					'()))
			(serialize (list
				tschema
				(rewrite_source_aliases _dedup_alias_map ttbl)
				isOuter
				(rewrite_source_aliases _dedup_alias_map (coalesceNil joinexpr true)))))
		nil)))
	(define _scalar_subquery_scalar_dedup_state (if (or scalar_query_has_opaque_runtime scalar_query_has_skip_level_scalar_helper)
		(list _scalar_subquery_scalar_tables '() '())
		(reduce _scalar_subquery_scalar_tables (lambda (state td) (match state
			'(kept key_map alias_map) (match td
				'(tv _ _ _ _) (begin
					(define _dedup_key (scalar_left_join_dedup_key td))
					(define _canonical_alias (get_assoc key_map _dedup_key))
					(if (nil? _canonical_alias)
						(list
							(merge kept (list td))
							(set_assoc key_map _dedup_key tv)
							alias_map)
						(list
							kept
							key_map
							(reduce (alias_lookup_variants tv) (lambda (acc alias_v)
								(set_assoc acc (string alias_v) _canonical_alias))
								alias_map))))
				_ td)
			_ state))
			(list '() '() '()))))
	(define _scalar_subquery_scalar_tables (nth _scalar_subquery_scalar_dedup_state 0))
	(define _scalar_subquery_scalar_alias_map (nth _scalar_subquery_scalar_dedup_state 2))
	(define rewrite_scalar_left_join_alias (lambda (alias_)
		(coalesceNil (get_assoc _scalar_subquery_scalar_alias_map (string alias_)) alias_)))
	(define rewrite_scalar_left_join_aliases (lambda (expr)
		(if (equal? _scalar_subquery_scalar_alias_map '())
			expr
			(rewrite_source_aliases _scalar_subquery_scalar_alias_map expr))))
	(define dedupe_logical_stages (lambda (stages)
		(match (reduce (coalesceNil stages '()) (lambda (state stage) (match state
			'(out seen) (begin
				(define skey (serialize stage))
				(if (has? seen skey)
					(list out seen)
					(list (merge out (list stage)) (merge seen (list skey)))))
			_ state))
			(list '() '()))
			'(out _) out
			_ stages)))
	(if (not (equal? _scalar_subquery_scalar_alias_map '()))
		(begin
			(set tables (map tables (lambda (td) (match td
				'(tv tschema ttbl toisOuter tje)
				(list tv tschema ttbl toisOuter
					(if (nil? tje) nil (rewrite_scalar_left_join_aliases tje)))
				td))))
			(set fields (map_assoc fields (lambda (k v) (rewrite_scalar_left_join_aliases v))))
			(set condition (rewrite_scalar_left_join_aliases condition))
			(set group (map group rewrite_scalar_left_join_aliases))
			(set having (rewrite_scalar_left_join_aliases having))
			(set order (map order (lambda (o) (match o
				'(col dir) (list (rewrite_scalar_left_join_aliases col) dir)
				o))))
			(set renamelist (reduce_assoc renamelist (lambda (acc alias_ field_map) (begin
				(define rewritten_alias (rewrite_scalar_left_join_alias alias_))
				(define rewritten_field_map (map_assoc field_map (lambda (col expr)
					(rewrite_scalar_left_join_aliases expr))))
				(set_assoc acc rewritten_alias
					(if (has_assoc? acc rewritten_alias)
						(merge (acc rewritten_alias) rewritten_field_map)
						rewritten_field_map))
			)) '()))))
	(define scalar_subquery_helper_alias (lambda (t) (match t
		'(tv _ ttbl _ _)
		(if (nil? tv) ttbl tv)
		nil)))
	(define scalar_joinexpr_ref_aliases (lambda (expr) (begin
		(define outer_ref_aliases (lambda (e) (match e
			'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
			'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
			'((symbol outer) symname) (match (split (string symname) ".")
				(list tbl _col) (list tbl)
				_ (if (list? symname) (outer_ref_aliases symname) '()))
			'((quote outer) symname) (match (split (string symname) ".")
				(list tbl _col) (list tbl)
				_ (if (list? symname) (outer_ref_aliases symname) '()))
			(cons sym args)
			(if (or (equal? sym (quote outer)) (equal? sym '(quote outer)) (equal? sym '(symbol outer)))
				(match args
					'(symname) (match (split (string symname) ".")
						(list tbl _col) (list tbl)
						_ (if (list? symname) (outer_ref_aliases symname) '()))
					'())
				(merge_unique (list (outer_ref_aliases sym) (map args outer_ref_aliases))))
			'())))
		(merge_unique (list (extract_tblvars expr) (outer_ref_aliases expr))))))
	/* Contract: scalar helper tables used only for SELECT/expr projection keep
	their LEFT JOIN joinexpr local so NULL-preserving semantics survive.
	When the current WHERE references such a helper, it is no longer a pure
	projection helper: its joinexpr belongs to the row-domain and must be merged
	into the normal table/condition pipeline before grouping. */
	(define condition_ref_aliases (extract_tblvars condition))
	(define scalar_subquery_condition_tables (filter _scalar_subquery_scalar_tables (lambda (t) (match t
		'(tv _ ttbl _ _)
		(has? condition_ref_aliases (if (nil? tv) ttbl tv))
		false))))
	(define base_table_aliases_for_scalar_helpers (map tables (lambda (td) (match td
		'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
		nil))))
	(define scalar_helper_joinexpr_aliases (lambda (td) (match td
		'(_ _ _ _ je) (if (nil? je) '() (scalar_joinexpr_ref_aliases je))
		'())))
	(define scalar_helper_has_base_join_source? (lambda (td)
		(reduce (scalar_helper_joinexpr_aliases td) (lambda (found alias_)
			(or found (has? base_table_aliases_for_scalar_helpers alias_)))
			false)))
	(define scalar_helper_keep_correlation_joinexpr (lambda (td) (match td
		'(alias_ schema_ table_ is_outer_ joinexpr_)
		(if (nil? joinexpr_)
			td
			(begin
				(define keep_parts (filter (flatten_and_terms joinexpr_) (lambda (part)
					(begin
						(define part_refs (scalar_joinexpr_ref_aliases part))
						(define has_base_join_ref (reduce part_refs (lambda (found ref_alias)
							(or found (and
								(not (equal?? ref_alias alias_))
								(has? base_table_aliases_for_scalar_helpers ref_alias))))
							false))
						(define local_helper_ref (and
							(not (equal? part_refs '()))
							(reduce part_refs (lambda (ok ref_alias)
								(and ok (or
									(equal?? ref_alias alias_)
									(scalar_generated_helper_alias? ref_alias))))
								true)))
						(or has_base_join_ref local_helper_ref)))))
				(list alias_ schema_ table_ is_outer_ (combine_and_terms keep_parts))))
		td)))
	(define scalar_condition_tables_with_join_source
		(filter scalar_subquery_condition_tables scalar_helper_has_base_join_source?))
	(define scalar_condition_tables_without_join_source
		(filter scalar_subquery_condition_tables (lambda (td)
			(not (scalar_helper_has_base_join_source? td)))))
	(define tables_with_condition_scalar_helpers
		(match (reduce tables (lambda (state td) (match state
			'(out remaining) (begin
				(define td_alias (match td '(tv _ ttbl _ _) (if (nil? tv) ttbl tv) nil))
				(define emit_helpers_raw (filter remaining (lambda (helper_td)
					(has? (scalar_helper_joinexpr_aliases helper_td) td_alias))))
				(define emit_helpers (map emit_helpers_raw
					scalar_helper_keep_correlation_joinexpr))
				(define emit_aliases (map emit_helpers scalar_subquery_helper_alias))
				(list
					(merge out (list td) emit_helpers)
					(filter remaining (lambda (helper_td)
						(not (has? emit_aliases (scalar_subquery_helper_alias helper_td)))))))
			state))
			(list '() scalar_condition_tables_with_join_source))
			'(ordered remaining) (merge ordered remaining)
			tables))
	(define scalar_subquery_projection_tables (filter _scalar_subquery_scalar_tables (lambda (t)
		(not (has? scalar_subquery_condition_tables t)))))
	/* Helpers referenced from a sibling JOIN ... ON must appear before the
	dependent table so split_scan_condition keeps ON semantics intact. */
	(define joinexpr_ref_aliases (merge_unique (map tables_with_condition_scalar_helpers (lambda (td) (match td
		'(_ _ _ _ je) (if (nil? je) '() (extract_tblvars je))
		'())))))
	(define scalar_subquery_joinexpr_tables (filter scalar_subquery_projection_tables (lambda (t)
		(has? joinexpr_ref_aliases (scalar_subquery_helper_alias t)))))
	(define scalar_subquery_tail_projection_tables (filter scalar_subquery_projection_tables (lambda (t)
		(not (has? scalar_subquery_joinexpr_tables t)))))
	(define tables_with_joinexpr_scalar_helpers
		(match (reduce tables_with_condition_scalar_helpers (lambda (state td) (match state
			'(out remaining) (begin
				(define je_refs (match td '(_ _ _ _ je) (if (nil? je) '() (scalar_joinexpr_ref_aliases je)) '()))
				(define emit_helpers (filter remaining (lambda (ht)
					(has? je_refs (scalar_subquery_helper_alias ht)))))
				(define emit_aliases (map emit_helpers scalar_subquery_helper_alias))
				(list
					(merge out emit_helpers (list td))
					(filter remaining (lambda (ht)
						(not (has? emit_aliases (scalar_subquery_helper_alias ht)))))))
			state))
			(list '() scalar_subquery_joinexpr_tables))
			'(ordered remaining) (merge ordered remaining)
			tables))
	(define order_scalar_tables_by_join_dependencies (lambda (tbls) (begin
		(define table_aliases (map tbls (lambda (td) (match td
			'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
			nil))))
		(define table_alias (lambda (td) (match td
			'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
			nil)))
		(define dotted_ref_aliases (lambda (expr)
			(if (list? expr)
				(match expr
					(cons head args) (merge_unique (list
						(dotted_ref_aliases head)
						(map args dotted_ref_aliases)))
					'())
				(match (split (string expr) ".")
					(list ref_alias _col) (if (has? table_aliases ref_alias)
						(list ref_alias)
						'())
					'()))))
		(define table_deps (lambda (td) (match td
			'(tv _ ttbl _ joinexpr)
			(begin
				(define alias_ (if (nil? tv) ttbl tv))
				(if (nil? joinexpr)
					'()
					(filter (merge_unique (list
						(scalar_joinexpr_ref_aliases joinexpr)
						(dotted_ref_aliases joinexpr))) (lambda (ref_alias)
							(and
								(not (equal?? ref_alias alias_))
								(has? table_aliases ref_alias))))))
			'())))
		(define deps_satisfied? (lambda (td emitted)
			(reduce (table_deps td) (lambda (ok dep)
				(and ok (has? emitted dep)))
				true)))
		(define ordered_state (reduce tbls (lambda (state _)
			(match state
				'(ordered remaining emitted)
				(begin
					(define ready (filter remaining (lambda (td)
						(deps_satisfied? td emitted))))
					(if (equal? ready '())
						state
						(list
							(merge ordered ready)
							(filter remaining (lambda (td) (not (has? ready td))))
							(merge emitted (map ready table_alias)))))
				state))
			(list '() tbls '())))
		(match ordered_state
			'(ordered remaining _)
			(if (equal? remaining '())
				ordered
				(merge ordered remaining))
			tbls))))
	(set tables (order_scalar_tables_by_join_dependencies
		(merge scalar_condition_tables_without_join_source tables_with_joinexpr_scalar_helpers _scalar_subquery_tables scalar_subquery_tail_projection_tables)))
	(define _scalar_subquery_schemas (coalesceNil (scalar_subquery_cache "schemas") '()))
	(if (not (equal? _scalar_subquery_schemas '()))
		(set schemas
			(reduce_assoc _scalar_subquery_schemas (lambda (acc alias cols)
				(set_assoc acc alias
					(merge_schema_fields_unique (list cols (coalesceNil (acc alias) '())))))
				schemas)))
	/* ensure materialized temp sources have a visible schema under their current alias.
	This keeps later planner passes from guessing temp columns via ad-hoc name heuristics. */
	(set schemas (reduce tables (lambda (acc td) (match td
		'(tv tschema ttbl _ _)
		(begin
			(define _existing (if (has_assoc? acc tv) (acc tv) nil))
			(define _resolved (coalesce _existing (materialized_source_schema tschema ttbl tv acc)))
			(if (nil? _resolved) acc
				(set_assoc acc tv _resolved)))
		acc)) schemas))
	(define scalar_helper_join_alias? (lambda (alias_)
		(and (string? alias_)
			(>= (strlen alias_) 14)
			(equal? (substr alias_ 0 14) "domain_scalar_"))))
	(define current_table_aliases (map tables (lambda (td) (match td
		'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
		nil))))
	(define current_alias_has_column? (lambda (alias_ col ci)
		(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
			(or found ((if ci equal?? equal?) (coldef "Field") col)))
			false)))
	(define unique_current_alias_for_column (lambda (current_alias col ci)
		(match (filter current_table_aliases (lambda (alias_)
			(and
				(not (nil? alias_))
				(not (equal?? alias_ current_alias))
				(current_alias_has_column? alias_ col ci))))
			(cons only_alias rest_aliases)
			(if (equal? rest_aliases '()) only_alias nil)
			nil)))
	(define rewrite_dangling_scalar_join_aliases (lambda (current_alias expr) (match expr
		'((symbol get_column) alias_ ti col ci)
		(if (or (nil? alias_) (has? current_table_aliases alias_))
			expr
			(begin
				(define resolved_alias (unique_current_alias_for_column current_alias col ci))
				(if (nil? resolved_alias)
					expr
					(list (quote get_column) resolved_alias false col false))))
		'((quote get_column) alias_ ti col ci)
		(if (or (nil? alias_) (has? current_table_aliases alias_))
			expr
			(begin
				(define resolved_alias (unique_current_alias_for_column current_alias col ci))
				(if (nil? resolved_alias)
					expr
					(list (quote get_column) resolved_alias false col false))))
		(cons sym args) (cons sym (map args (lambda (arg)
			(rewrite_dangling_scalar_join_aliases current_alias arg))))
		expr)))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl is_outer joinexpr)
		(if (and (scalar_helper_join_alias? tv) (not (nil? joinexpr)))
			(list tv tschema ttbl is_outer (rewrite_dangling_scalar_join_aliases tv joinexpr))
			td)
		td))))
	(define scalar_helper_local_filter_parts (lambda (alias_)
		(filter (flatten_and_terms (coalesceNil condition true)) (lambda (part)
			(begin
				(define refs (extract_tblvars part))
				(and
					(not (equal? refs '()))
					(reduce refs (lambda (ok ref_alias)
						(and ok (equal?? ref_alias alias_)))
						true)))))))
	(define scalar_helper_positive_aggregate_match? (lambda (alias_ part) (match part
		'(op left right)
		(and
			(or
				(equal?? op (quote >))
				(equal?? op (symbol >)))
			(or (equal? right 0) (equal? right 0.0))
			(not (equal? (extract_aggregates left) '()))
			(has? (extract_tblvars left) alias_))
		false)))
	(define scalar_helper_positive_aggregate_local_filter (lambda (alias_ part) (begin
		(define positive_op? (lambda (op)
			(or
				(equal?? op (quote >))
				(equal?? op (symbol >)))))
		(define unwrap_positive_value (lambda (expr) (match expr
			'((symbol coalesceNil) inner fallback)
			(if (or (equal? fallback 0) (equal? fallback 0.0))
				(unwrap_positive_value inner)
				expr)
			'((quote coalesceNil) inner fallback)
			(if (or (equal? fallback 0) (equal? fallback 0.0))
				(unwrap_positive_value inner)
				expr)
			'((symbol coalesce) inner fallback)
			(if (or (equal? fallback 0) (equal? fallback 0.0))
				(unwrap_positive_value inner)
				expr)
			'((quote coalesce) inner fallback)
			(if (or (equal? fallback 0) (equal? fallback 0.0))
				(unwrap_positive_value inner)
				expr)
			_ expr)))
		(define aggregate_input (lambda (expr) (match (unwrap_positive_value expr)
			'((symbol aggregate) input _ _ ) input
			'((quote aggregate) input _ _) input
			nil)))
		(define count_input_filter (lambda (expr) (match expr
			'((symbol if) pred 1 0) pred
			'((quote if) pred 1 0) pred
			'((symbol if) pred true false) pred
			'((quote if) pred true false) pred
			nil)))
		(match part
			'(op left right)
			(if (and (positive_op? op) (or (equal? right 0) (equal? right 0.0)))
				(begin
					(define pred (count_input_filter (aggregate_input left)))
					(if (and (not (nil? pred))
						(reduce (extract_tblvars pred) (lambda (ok ref_alias)
							(and ok (equal?? ref_alias alias_))) true))
						pred
						nil))
				nil)
			nil))))
	(define scalar_helper_filter_requires_match? (lambda (alias_ part) (match part
		'(op left right)
		(and
			(or
				(equal?? op (quote equal?))
				(equal?? op (quote equal??))
				(equal?? op (quote =))
				(equal?? op (quote >))
				(equal?? op (quote <))
				(equal?? op (quote >=))
				(equal?? op (quote <=)))
			(or
				(has? (extract_tblvars left) alias_)
				(has? (extract_tblvars right) alias_)))
		false)))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl is_outer joinexpr)
		(if (scalar_helper_join_alias? tv)
			(begin
				(define helper_filter_parts (scalar_helper_local_filter_parts tv))
				(define helper_join_filter_parts (filter helper_filter_parts (lambda (part)
					(and
						(equal? (extract_aggregates part) '())
						(not (scalar_helper_count_anti_pred? part tv))))))
				(define helper_positive_filters
					(filter (map helper_filter_parts (lambda (part)
						(scalar_helper_positive_aggregate_local_filter tv part)))
						(lambda (part) (not (nil? part)))))
				(define helper_filter (combine_and_terms
					(merge helper_join_filter_parts helper_positive_filters)))
				(define helper_filter_requires_match
					(reduce helper_filter_parts (lambda (found part)
						(or found
							(if (equal? (extract_aggregates part) '())
								(scalar_helper_filter_requires_match? tv part)
								(scalar_helper_positive_aggregate_match? tv part))))
						false))
				(list tv tschema ttbl
					(if helper_filter_requires_match false is_outer)
					(combine_and_terms (list joinexpr helper_filter))))
			td)
		td))))
	(define scalar_required_aliases (lambda (tbls) (begin
		(define initial_required
			(filter (map tbls (lambda (td) (match td
				'(tv _ _ is_outer _)
				(if (and (scalar_helper_join_alias? tv) (not is_outer)) tv nil)
				nil)))
				(lambda (alias_) (not (nil? alias_)))))
		(define step_required (lambda (required)
			(merge_unique (list required
				(filter (map tbls (lambda (td) (match td
					'(tv _ _ _ _)
					(if (and
						(scalar_helper_join_alias? tv)
						(reduce tbls (lambda (found dep_td) (or found (match dep_td
							'(dep_tv _ _ _ dep_joinexpr)
							(and
								(has? required dep_tv)
								(has? (extract_tblvars dep_joinexpr) tv))
							false))) false))
						tv
						nil)
					nil)))
					(lambda (alias_) (not (nil? alias_))))))))
		(reduce (produceN (count tbls)) (lambda (required _)
			(step_required required))
			initial_required))))
	(define required_scalar_aliases (scalar_required_aliases tables))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl is_outer joinexpr)
		(if (and (scalar_helper_join_alias? tv) (has? required_scalar_aliases tv))
			(list tv tschema ttbl false joinexpr)
			td)
		td))))
	(set tables (order_scalar_tables_by_join_dependencies tables))
	/* Design contract: logical get_column/aggregate/window sentinels should stay
	as long as possible and join semantics must stay attached to their stage.
	COUNT/IN helper tables still expose their correlation predicates here as
	global condition terms. LEFT/MARK scalar helpers keep joinexpr local even
	when WHERE references their computed value; otherwise NULL-preserving
	semantics are lost. */
	(define _scalar_subquery_joinexprs (filter (map (merge _scalar_subquery_tables scalar_subquery_condition_tables) (lambda (t) (match t
		'(_ _ _ isOuter je) (if isOuter nil je)
		nil))) (lambda (x) (not (nil? x)))))
	(set condition (if (equal? _scalar_subquery_joinexprs '()) condition (cons (quote and) (cons condition _scalar_subquery_joinexprs))))
	(define inner_scalar_helper_tables (filter tables (lambda (td) (match td
		'(tv _ _ is_outer joinexpr)
		(and
			(scalar_helper_join_alias? tv)
			(not is_outer)
			(not (nil? joinexpr)))
		false))))
	(define scalar_helper_alias_for_stage (lambda (td) (match td
		'(tv _ _ _ _) tv
		nil)))
	(define scalar_helper_depends_on_stage_alias? (lambda (stage_aliases td) (match td
		'(_ _ _ _ joinexpr)
		(reduce (extract_tblvars joinexpr) (lambda (found ref_alias)
			(or found (has? stage_aliases ref_alias)))
			false)
		false)))
	(define expand_scoped_group_stage_dependencies (lambda (stage) (begin
		(define stage_aliases (stage_partition_aliases stage))
		(define stage_groups (coalesceNil (stage_group_cols stage) '()))
		(if (or (nil? stage_aliases) (equal? stage_groups '()))
			stage
			(begin
				(define dependent_helper_aliases (filter (map inner_scalar_helper_tables (lambda (td)
					(if (scalar_helper_depends_on_stage_alias? stage_aliases td)
						(scalar_helper_alias_for_stage td)
						nil)))
					(lambda (alias_) (not (nil? alias_)))))
				(define expanded_aliases (merge_unique (list stage_aliases dependent_helper_aliases)))
				(if (equal? expanded_aliases stage_aliases)
					stage
					(stage_rebuild_with_meta
						stage
						(make_stage
							stage_groups
							(stage_having_expr stage)
							(stage_order_list stage)
							(stage_limit_partition_cols stage)
							(stage_limit_val stage)
							(stage_offset_val stage)
							(stage_is_dedup stage)
							expanded_aliases
							(stage_init_code stage)
							(stage_condition stage)
							(stage_once_limit stage))
						(lambda (expr) expr)
						(lambda (alias_) alias_))))))))
	(define _scalar_subquery_propagated_groups_raw
		(if (equal? _scalar_subquery_scalar_alias_map '())
			(coalesceNil (scalar_subquery_cache "groups") '())
			(map (coalesceNil (scalar_subquery_cache "groups") '()) (lambda (stage)
				(rewrite_stage_for_flattened_aliases
					stage
					rewrite_scalar_left_join_aliases
					rewrite_scalar_left_join_alias)))))
	(define _scalar_subquery_propagated_groups
		(dedupe_logical_stages
			(map _scalar_subquery_propagated_groups_raw expand_scoped_group_stage_dependencies)))
	(set groups (if (equal? _scalar_subquery_propagated_groups '()) groups (merge _scalar_subquery_propagated_groups (coalesceNil groups '()))))
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
		(cons sym args) (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args canonicalize_for_rename)))
		expr
	)))

	/* apply renamelist (assoc of assoc of expr) */
	(define live_table_aliases (map tables (lambda (td) (match td
		'(alias _ _ _ _) alias
		nil))))
	(define replace_rename (lambda (expr) (match expr
		'((symbol get_column) alias_ ti col ci) (if (nil? alias_)
			/* no tblalias -> search the field in all tables */
			(reduce_assoc renamelist (lambda (a k v) (coalesce (v col) a)) expr)
			/* tblalias -> look up the field */
			(begin
				(define alias_str (string alias_))
				(define alias_sym (symbol alias_str))
				(define rename_fn (if (or (has? live_table_aliases alias_) (has? live_table_aliases alias_str) (has? live_table_aliases alias_sym))
					nil
					(if (has_assoc? renamelist alias_)
						(renamelist alias_)
						(if (has_assoc? renamelist alias_str)
							(renamelist alias_str)
							(if (has_assoc? renamelist alias_sym)
								(renamelist alias_sym)
								nil)))))
				(if (nil? rename_fn) expr (rename_fn col))
			)
		)
		(cons sym args) /* function call */ (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args replace_rename)))
		expr
	)))


	(define planner_visible_schemas (merge (schema_bindings_to_flat_list schemas) outer_schema_bindings))
	(define finalize_visible_expr (lambda (expr)
		(finalize_logical_expr_scoped
			(resolve_dependent_scalar_compile_markers expr planner_visible_schemas)
			schemas planner_visible_schemas replace_rename enforce_planner_contract)))
	(define finalize_visible_table_ref (lambda (tbl)
		(if (scan_tagged_table_needs_scan_order tbl)
			(scan_tagged_table_with_outer_sources
				(make_scan_tagged_table
					(scan_tagged_table_base tbl)
					(map (scan_tagged_table_order tbl) (lambda (o) (match o
						'(col dir) (list (finalize_visible_expr col) dir)
						o)))
					(scan_tagged_table_limit tbl)
					(scan_tagged_table_offset tbl)
					(scan_tagged_table_partition_cols tbl)
					(scan_tagged_table_once_limit tbl))
				(scan_tagged_table_outer_sources tbl))
			tbl)))


	/* Contract boundary for user-visible expressions:
	fields, WHERE, GROUP/HAVING/ORDER and JOIN conditions all go through the same
	finalize_visible_expr gate exactly once here. After that the planner must only
	see exact get_column markers and may no longer re-run schema casing repair. */
	(set fields (map_assoc (expand_star_fields_with_schemas fields schemas) (lambda (col expr)
		(finalize_visible_expr (replace_find_column expr)))))
	(if (not (equal? _scalar_subquery_scalar_alias_map '()))
		(set fields (map_assoc fields (lambda (k v) (rewrite_scalar_left_join_aliases v)))))

	/* return parameter list for build_queryplan */
	(set conditionAll (cons 'and (filter
		(cons (finalize_visible_expr condition) (map conditionList finalize_visible_expr))
		(lambda (x) (not (nil? x))))))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl toisOuter tje)
		(list tv tschema (finalize_visible_table_ref ttbl) toisOuter
			(if (nil? tje) nil (finalize_visible_expr tje)))
		td))))
	(set group (map group finalize_visible_expr))
	(set order (map order (lambda (o) (match o '(col dir) (list (finalize_visible_expr col) dir)))))

	(set having (finalize_visible_expr having))

	/* LEFT JOIN pruning: remove LEFT JOINed tables that are not referenced
	anywhere in the query (fields, condition, having, order, or sibling
	joinexprs). A LEFT JOIN that is never read contributes only NULL columns
	and cannot filter rows, so it is safe to drop entirely. */
	(define _all_referenced_aliases_base (merge_unique (list
		(extract_all_table_aliases fields)
		(extract_all_table_aliases conditionAll)
		(extract_all_table_aliases (coalesceNil having true))
		(merge (map (coalesceNil order '()) (lambda (o) (extract_all_table_aliases o))))
		(merge (map (coalesceNil groups '()) (lambda (stage)
			(merge_unique
				(merge (map (coalesceNil (stage_group_cols stage) '()) extract_all_table_aliases))
				(extract_all_table_aliases (coalesceNil (stage_having_expr stage) true))
				(merge (map (coalesceNil (stage_order_list stage) '()) (lambda (o) (match o
					'(col dir) (extract_all_table_aliases col)
					(extract_all_table_aliases o)))))
				(coalesceNil (stage_partition_aliases stage) '())))))
		(merge (map tables (lambda (td) (match td '(_ _ _ _ je) (if (nil? je) '() (extract_all_table_aliases je)) '())))))))
	(define _flatten_alias_matches_wrapper_ref (lambda (alias_str ref_alias)
		(begin
			(define ref_str (string ref_alias))
			(and
				(> (strlen alias_str) (+ (strlen ref_str) 1))
				(equal? (substr alias_str 0 (strlen ref_str)) ref_str)
				(equal? (substr alias_str (strlen ref_str) 1) "\0")))))
	(define _flattened_scalar_alias? (lambda (alias_)
		(begin
			(define alias_str (string alias_))
			(and
				(>= (strlen alias_str) 14)
				(equal? (substr alias_str 0 14) "domain_scalar_")))))
	(define _flatten_aliases_referenced_via_wrapper (filter
		(map tables (lambda (td) (match td
			'(alias _ _ _ _)
			(begin
				(define alias_str (string alias))
				(if (reduce _all_referenced_aliases_base (lambda (found ref_alias)
					(or found (_flatten_alias_matches_wrapper_ref alias_str ref_alias))) false)
					alias_str
					nil))
			nil)))
		(lambda (x) (not (nil? x)))))
	(define _all_referenced_aliases (merge_unique
		_all_referenced_aliases_base
		_flatten_aliases_referenced_via_wrapper))
	(set tables (filter tables (lambda (td) (match td
		'(alias _ _ isOuter _) (or (not isOuter) (has? _all_referenced_aliases (string alias)) (_flattened_scalar_alias? alias))
		true))))

	(define groups (merge
		(coalesceNil _scalar_subquery_propagated_groups '())
		(if (coalesce _cd_distinct_exprs false)
			/* COUNT(DISTINCT): two group stages - first dedup, then aggregate */
			(list
				(make_dedup_stage
					(merge
						(map (coalesce _cd_user_group '()) finalize_visible_expr)
						(map _cd_distinct_exprs (lambda (e) (replace_find_column (finalize_visible_expr e)))))
					nil)
				(make_group_stage
					(if (equal? (coalesceNil _cd_user_group '()) '())
						'(1)
						(map _cd_user_group (lambda (e) (replace_find_column (finalize_visible_expr e)))))
					(_cd_replace (finalize_visible_expr _cd_having))
					(map (coalesce _cd_order '()) (lambda (o) (match o '(col dir) (list (_cd_replace (finalize_visible_expr col)) dir))))
					_cd_limit _cd_offset nil nil))
			/* normal: single group stage */
			(if (or group having order limit offset) (list (make_group_stage group having order limit offset nil nil)) '()))))
	/* Contract boundary: untangle_query returns canonical logical IR.
	All case-insensitive parser markers are resolved here, before build_queryplan
	starts creating keytables/prejoins or serializing canonical expression names. */
	(define _canon_fields fields)
	(define _canon_condition conditionAll)
	(define _canon_groups (map (coalesceNil groups '()) (lambda (stage)
		(finalize_logical_stage_scoped stage schemas planner_visible_schemas replace_rename enforce_planner_contract))))
	/* eliminate unused LEFT JOINs: a LEFT JOIN is unused when none of its
	columns appear in fields or group stages. Join predicates reference the
	JOIN alias by construction and must not keep it alive. Only unnested
	aliases are protected explicitly because they may be referenced indirectly. */
	(define _unnested_aliases (map _scalar_subquery_tables (lambda (t) (match t '(alias _ _ _ _) alias _ nil))))
	(define _joinexpr_dependency_aliases (merge_unique (map tables (lambda (td) (match td
		'(alias _ _ _ je) (filter (if (nil? je) '() (extract_tblvars je)) (lambda (ref_alias)
			(not (equal?? ref_alias alias))))
		'())))))
	(define _used_tvs_base (merge_unique
		_unnested_aliases
		_joinexpr_dependency_aliases
		(merge (extract_assoc _canon_fields (lambda (k v) (extract_tblvars v))))
		(extract_tblvars _canon_condition)
		(merge (map _canon_groups (lambda (stage)
			(merge_unique
				(merge (map (coalesceNil (stage_group_cols stage) '()) extract_tblvars))
				(extract_tblvars (coalesceNil (stage_having_expr stage) true))
				(merge (map (coalesceNil (stage_order_list stage) '()) (lambda (o) (match o '(col dir) (extract_tblvars col) (extract_tblvars o)))))
				(coalesceNil (stage_partition_aliases stage) '())))))))
	(define _used_flatten_aliases_via_wrapper (filter
		(map tables (lambda (td) (match td
			'(alias _ _ _ _)
			(begin
				(define alias_str (string alias))
				(if (reduce _used_tvs_base (lambda (found ref_alias)
					(or found (_flatten_alias_matches_wrapper_ref alias_str ref_alias))) false)
					alias
					nil))
			nil)))
		(lambda (x) (not (nil? x)))))
	(define _used_tvs (merge_unique _used_tvs_base _used_flatten_aliases_via_wrapper))
	/* prune unused LEFT JOINs and unreferenced .(1) DUAL tables.
	.(1) is only pruned if other tables remain (it's the scan driver otherwise). */
	(define _has_non_dual (reduce tables (lambda (a t) (or a (match t '(_ _ tbl _ _) (not (equal? tbl ".(1)")) true))) false))
	(define _pruned_tables (filter tables (lambda (t) (match t
		'(alias _ tbl isOuter _) (if isOuter (or (has? _used_tvs alias) (_flattened_scalar_alias? alias))
			(if (and _has_non_dual (equal? tbl ".(1)")) (has? _used_tvs alias) true))
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
	(list schema _pruned_tables _canon_fields _canon_condition _canon_groups schemas replace_find_column (coalesceNil (scalar_subquery_cache "init") '()))
)
)
)

/*
=== CONTRACT: join_reorder ===

PURPOSE: Decide the physical table order for scan execution AND attach the
correctness fixups that the chosen order requires.
Tables are scored by estimated row count (from statistics) with local
predicate count as tiebreaker; the cheapest table drives the scan. When a
scalar LEFT-JOIN helper ends up placed above its outer correlation source
(helper_pos < outer_pos), join_reorder annotates the helper's partition
stage with an anti-pass-needed marker so build_queryplan can emit the
companion null-extension scan (FAQ-unnesting point 35).

INPUT/OUTPUT: 7-tuple (schema tables fields condition groups schemas replace_find_column)

WHAT IT MAY DO:
- Reorder tables within a barrier-free scan segment
- Augment the groups list with anti-pass-needed markers on partition stages
whose helper was lifted above its correlation source by the reordering

WHAT IT MUST NOT DO:
- Transform query structure (that is untangle_query's job)
- Decorrelate subqueries or create joins (that is untangle_query's job)
- Build physical scan plans (that is build_queryplan's job)
- Reorder tables across a join fence (LEFT/SEMI/ANTI JOIN boundary)
*/
/* conservative first pass: only reorder two-table INNER segments when the
second table carries strictly more local WHERE predicates than the first. */
(define jqr_td_alias (lambda (td) (nth td 0)))
(define jqr_td_schema (lambda (td) (nth td 1)))
(define jqr_td_table (lambda (td) (nth td 2)))
(define jqr_td_outer (lambda (td) (nth td 3)))
(define jqr_td_joinexpr (lambda (td) (nth td 4)))
(define jqr_td_with_joinexpr (lambda (td joinexpr)
	(list (jqr_td_alias td) (jqr_td_schema td) (jqr_td_table td) (jqr_td_outer td) joinexpr)))
(define jqr_flatten_join_terms (lambda (tables_)
	(merge (map tables_ (lambda (td)
		(flatten_and_terms (coalesceNil (jqr_td_joinexpr td) true)))))))
(define jqr_local_term_count (lambda (alias terms)
	(reduce terms
		(lambda (acc term) (begin
			(define refs (extract_tblvars term))
			(if (and (not (equal? refs '()))
				(reduce refs (lambda (ok tv) (and ok (equal?? tv alias))) true))
				(+ acc 1)
				acc)))
		0)))
(define jqr_has_order_sensitive_stage (lambda (groups)
	(reduce groups
		(lambda (acc stage)
			(or acc
				(not (equal? (coalesceNil (stage_order_list stage) '()) '()))
				(not (nil? (stage_limit_val stage)))
				(not (nil? (stage_offset_val stage)))))
		false)))
/* jqr_estimate_rows: estimate filtered row count using cached statistics */
(define jqr_estimate_rows (lambda (td condition schemas) (begin
	(define alias (jqr_td_alias td))
	(define cols (if (has_assoc? schemas alias) (schemas alias) '()))
	(define base_rows (reduce cols (lambda (acc coldef)
		(if (> acc 0) acc
			(begin (define re (coldef "RowEstimate"))
				(if (and (not (nil? re)) (> re 0)) re 0)))) 0))
	(if (equal? base_rows 0) 1000000
		(begin
			(define condition_terms (flatten_and_terms (coalesceNil condition true)))
			(define selectivity (reduce condition_terms (lambda (sel term)
				(match term
					'((symbol equal??) left right) (begin
						(define col_name (match left
							'((symbol get_column) (eval alias) _ c _) c
							'((quote get_column) (eval alias) _ c _) c
							nil))
						(define col_name (if (nil? col_name) (match right
							'((symbol get_column) (eval alias) _ c _) c
							'((quote get_column) (eval alias) _ c _) c
							nil) col_name))
						(if (nil? col_name) sel
							(begin
								(define distinct (reduce cols (lambda (acc coldef)
									(if (> acc 0) acc
										(if (equal?? (coldef "Field") col_name)
											(begin (define de (coldef "DistinctEstimate"))
												(if (and (not (nil? de)) (> de 0)) de 0))
											0))) 0))
								(if (> distinct 1) (* sel (/ 1.0 distinct)) sel))))
					_ sel)) 1.0))
			(max 1 (* base_rows selectivity)))))))
(define jqr_reorder_inner_segment (lambda (segment condition schemas) (begin
	(if (< (count segment) 2)
		segment
		(begin
			(if (reduce segment (lambda (acc td) (or acc (jqr_td_outer td))) false)
				segment
				(begin
					/* score each table: estimated rows (from statistics) with predicate count as tiebreaker */
					(define condition_terms (flatten_and_terms (coalesceNil condition true)))
					(define scored (map segment (lambda (td) (list
						(jqr_estimate_rows td condition schemas)
						(- 0 (jqr_local_term_count (jqr_td_alias td) condition_terms)) /* negate: more predicates = lower score = scanned first */
						td))))
					(define sorted (sort scored (lambda (a b)
						(if (equal? (car a) (car b))
							(< (cadr a) (cadr b))  /* tiebreaker: more local predicates first */
							(< (car a) (car b))))))
					(define all_join_terms (jqr_flatten_join_terms segment))
					(define combined_je (combine_and_terms all_join_terms))
					(map (produceN (count sorted)) (lambda (i) (begin
						(define td (nth (nth sorted i) 2))
						(if (equal? i 0)
							(jqr_td_with_joinexpr td true)
							(jqr_td_with_joinexpr td combined_je))))))))))))

(define jqr_reorder_segments (lambda (tables_ condition schemas) (begin
	(match (reduce tables_
		(lambda (state td) (match state
			'(out seg)
			(if (jqr_td_outer td)
				(list (merge out (jqr_reorder_inner_segment seg condition schemas) (list td)) '())
				(list out (merge seg (list td))))
			state))
		(list '() '()))
		'(out seg) (merge out (jqr_reorder_inner_segment seg condition schemas))
		tables_))))
(define jqr_dotted_ref_aliases (lambda (expr aliases)
	(if (list? expr)
		(match expr
			(cons head args) (merge_unique (list
				(jqr_dotted_ref_aliases head aliases)
				(map args (lambda (arg) (jqr_dotted_ref_aliases arg aliases)))))
			'())
		(match (split (string expr) ".")
			(list ref_alias _col) (if (has? aliases ref_alias)
				(list ref_alias)
				'())
			'()))))
(define jqr_outer_ref_aliases (lambda (expr aliases)
	(match expr
		'((symbol get_column) tv _ _ _) (if (and (not (nil? tv)) (has? aliases tv))
			(list tv)
			'())
		'((quote get_column) tv _ _ _) (if (and (not (nil? tv)) (has? aliases tv))
			(list tv)
			'())
		(cons sym args)
		(if (or (equal? sym (quote outer)) (equal? sym '(quote outer)) (equal? sym '(symbol outer)))
			(match args
				(cons first _)
				(if (list? first)
					(jqr_outer_ref_aliases first aliases)
					(match (split (string first) ".")
						(list ref_alias _col) (if (has? aliases ref_alias)
							(list ref_alias)
							'())
						'()))
				'())
			(merge_unique (list
				(jqr_outer_ref_aliases sym aliases)
				(map args (lambda (arg) (jqr_outer_ref_aliases arg aliases))))))
		'())))
(define jqr_expr_dependency_aliases (lambda (expr self_alias aliases)
	(filter (merge_unique (list
		(extract_tblvars expr)
		(jqr_dotted_ref_aliases expr aliases)
		(jqr_outer_ref_aliases expr aliases))) (lambda (ref_alias)
			(not (equal?? ref_alias self_alias))))))
(define jqr_join_dependency_aliases (lambda (td aliases)
	(match td
		'(alias _ _ _ je)
		(if (nil? je)
			'()
			(jqr_expr_dependency_aliases je alias aliases))
		'())))
(define jqr_order_join_dependencies (lambda (tables_) (begin
	(define aliases (map tables_ jqr_td_alias))
	(define deps-satisfied? (lambda (td emitted)
		(reduce (jqr_join_dependency_aliases td aliases) (lambda (ok dep)
			(and ok (has? emitted dep)))
			true)))
	(define ordered-state (reduce tables_ (lambda (state _)
		(match state
			'(ordered remaining emitted)
			(begin
				(define ready (filter remaining (lambda (td)
					(deps-satisfied? td emitted))))
				(if (equal? ready '())
					state
					(list
						(merge ordered ready)
						(filter remaining (lambda (td) (not (has? ready td))))
						(merge emitted (map ready jqr_td_alias)))))
			state))
		(list '() tables_ '())))
	(match ordered-state
		'(ordered remaining _)
		(if (equal? remaining '())
			ordered
			(merge ordered remaining))
		tables_))))
(define join_reorder (lambda (schema tables fields condition groups schemas replace_find_column) (begin
	(define jqr_original_outer_domain_aliases (filter (map tables (lambda (td) (match td
		'(alias _ _ is_outer _)
		(if (and
			is_outer
			(strlike (string alias) "domain_scalar_%"))
			alias
			nil)
		nil))) (lambda (alias) (not (nil? alias)))))
	(define jqr_constant_scalar_aliases (reduce (coalesceNil groups '()) (lambda (acc stage)
		(begin
			(define _spa (stage_partition_aliases stage))
			(define _sg (coalesceNil (stage_group_cols stage) '()))
			(define _spc (coalesceNil (stage_limit_partition_cols stage) 0))
			(define _sos (coalesceNil (stage_outer_sources stage) '()))
			(if (or
				(nil? _spa)
				(not (equal? _sg '()))
				(not (equal? _spc 0))
				(not (equal? _sos '())))
				acc
				(merge acc _spa))))
		'()))
	(define jqr_has_external_join_refs (lambda (td) (begin
		(define alias (jqr_td_alias td))
		(define collect_join_refs (lambda (expr) (match expr
			'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
			'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
			(cons _ args) (merge (map args collect_join_refs))
			'())))
		(reduce (collect_join_refs (coalesceNil (jqr_td_joinexpr td) true))
			(lambda (found tv)
				(or found (not (equal?? tv alias))))
			false))))
	(define jqr_constant_scalar_tables (filter tables (lambda (td) (match td
		'(alias _ _ _ _) (and
			(has? jqr_constant_scalar_aliases alias)
			(not (jqr_has_external_join_refs td)))
		false))))
	(define jqr_constant_scalar_table_aliases (map jqr_constant_scalar_tables (lambda (td)
		(match td '(alias _ _ _ _) alias nil))))
	(define jqr_alias_referenced_by_other_joinexpr (lambda (target_alias)
		(reduce tables (lambda (found td) (or found (match td
			'(alias _ _ _ je)
			(and
				(not (equal?? alias target_alias))
				(not (nil? je))
				(has? (extract_tblvars je) target_alias))
			false))) false)))
	(define jqr_regular_tables_base (filter tables (lambda (td) (match td
		'(alias _ _ _ _)
		(if (has? jqr_constant_scalar_aliases alias)
			(and
				(not (has? jqr_constant_scalar_table_aliases alias))
				(jqr_alias_referenced_by_other_joinexpr alias))
			true)
		true))))
	(define jqr_condition_refs (merge_unique (list
		(extract_tblvars condition)
		(jqr_dotted_ref_aliases condition (map jqr_regular_tables_base jqr_td_alias)))))
	(define jqr_condition_refs_alias? (lambda (alias_)
		(or
			(has? jqr_condition_refs alias_)
			(has? jqr_condition_refs (string alias_))
			(has? jqr_condition_refs (symbol (string alias_))))))
	(define jqr_regular_aliases (map jqr_regular_tables_base jqr_td_alias))
	(define jqr_domain_scalar_alias? (lambda (alias_)
		(begin
			(define alias_str (string alias_))
			(and
				(>= (strlen alias_str) 14)
				(equal? (substr alias_str 0 14) "domain_scalar_")))))
	(define jqr_helper_ref? (lambda (expr helper_alias)
		(match expr
			'((symbol get_column) tv _ _ _) (equal?? tv helper_alias)
			'((quote get_column) tv _ _ _) (equal?? tv helper_alias)
			false)))
	(define jqr_coalesced_helper_ref? (lambda (expr helper_alias)
		(match expr
			'(head value fallback)
			(and
				(or (equal? head (quote coalesce)) (equal? head (symbol coalesce))
					(equal? head '(quote coalesce)))
				(jqr_helper_ref? value helper_alias)
				(equal? fallback 0))
			_ (jqr_helper_ref? expr helper_alias))))
	(define jqr_positive_helper_count? (lambda (expr helper_alias)
		(match expr
			'(head left right)
			(and
				(or (equal? head (quote >)) (equal? head (symbol >))
					(equal? head '(quote >)))
				(jqr_coalesced_helper_ref? left helper_alias)
				(equal? right 0))
			false)))
	(define jqr_not_positive_helper_count? (lambda (expr helper_alias)
		(match expr
			'(head inner)
			(and
				(or (equal? head (quote not)) (equal? head (symbol not))
					(equal? head '(quote not)))
				(jqr_positive_helper_count? inner helper_alias))
			false)))
	(define jqr_zero_literal? (lambda (value)
		(or (equal? value 0) (equal? value 0.0))))
	(define jqr_zero_coalesced_helper? (lambda (expr helper_alias)
		(match expr
			'(head inner fallback)
			(and
				(or (equal? head (quote coalesce)) (equal? head (symbol coalesce))
					(equal? head '(quote coalesce))
					(equal? head (quote coalesceNil)) (equal? head (symbol coalesceNil))
					(equal? head '(quote coalesceNil)))
				(jqr_zero_literal? fallback)
				(has? (extract_tblvars inner) helper_alias))
			false)))
	(define jqr_zero_helper_count? (lambda (expr helper_alias)
		(match expr
			'(head left right)
			(and
				(or (equal? head (quote equal?)) (equal? head (symbol equal?))
					(equal? head '(quote equal?))
					(equal? head (quote equal??)) (equal? head (symbol equal??))
					(equal? head '(quote equal??))
					(equal? head (quote =)) (equal? head (symbol =))
					(equal? head '(quote =)))
				(or
					(and (jqr_zero_literal? right) (has? (extract_tblvars left) helper_alias))
					(and (jqr_zero_literal? left) (has? (extract_tblvars right) helper_alias))))
			false)))
	(define jqr_condition_keeps_scalar_misses? (lambda (expr helper_alias)
		(match expr
			(cons head args)
			(or
				(jqr_not_positive_helper_count? expr helper_alias)
				(jqr_zero_helper_count? expr helper_alias)
				(reduce (coalesceNil args '()) (lambda (found arg)
					(or found (jqr_condition_keeps_scalar_misses? arg helper_alias))) false))
			false)))
	(define jqr_strip_condition_scalar_joinexpr (lambda (td) (match td
		'(alias_ schema_ table_ is_outer_ joinexpr_)
		(if (and
			(jqr_domain_scalar_alias? alias_)
			(not (nil? joinexpr_)))
			(begin
				(define keep_parts (filter (flatten_and_terms joinexpr_) (lambda (part)
					(not (equal? (jqr_expr_dependency_aliases part alias_ jqr_regular_aliases) '())))))
				(list alias_ schema_ table_
					is_outer_
					(combine_and_terms keep_parts)))
			td)
		td)))
	(define jqr_regular_tables
		(map jqr_regular_tables_base jqr_strip_condition_scalar_joinexpr))
	(define jqr_dependency_ordered_tables
		(jqr_order_join_dependencies jqr_regular_tables))
	(define jqr_final_tables_raw (merge
		jqr_constant_scalar_tables
		(if (jqr_has_order_sensitive_stage groups)
			jqr_dependency_ordered_tables
			(jqr_reorder_segments jqr_dependency_ordered_tables condition schemas))))
	(define jqr_final_tables
		(map jqr_final_tables_raw (lambda (td) (match (jqr_strip_condition_scalar_joinexpr td)
			'(alias schema_ table_ is_outer joinexpr)
			(list alias schema_ table_
				(if (has? jqr_original_outer_domain_aliases alias) true is_outer)
				joinexpr)
			td))))
	/* FAQ-unnesting point 35: after the physical reorder decision is made,
	hand the 7-tuple to inject_anti_passes so stages whose helper alias was
	lifted above its correlation source (helper_pos < outer_pos) pick up an
	anti-pass-needed marker. build_queryplan consumes the marker to emit a
	companion null-extension scan. Safe no-op when nothing was lifted. */
	(inject_anti_passes (list schema jqr_final_tables
		fields condition groups schemas replace_find_column)))))

/* inject_anti_passes: post-reorder correctness fixup for scalar LEFT-JOIN
helpers. When join_reorder places a scalar helper carrying a partition-stage
with outer-sources ABOVE its outer correlation source, scan_order's per-call
isOuter fallback cannot null-extend outer rows whose correlation key has no
inner match — the helper scan emits only matched partitions, and plain LEFT
JOIN semantics drop the unmatched outer rows. Rather than constraining the
reorderer, we detect the lifting here and annotate the stage with an
anti-pass-needed marker that build_queryplan consumes to emit a companion
anti-pass scan over the outer table.

Marker shape (prepended to the stage assoc list):
(anti-pass-needed helper_tv outer_tv outer_col inner_expr)
- helper_tv:  alias of the scalar helper whose partition-stage carries outer-sources
- outer_tv:   alias of the outer table that supplies the correlation key
- outer_col:  column on outer_tv used as the correlation key
- inner_expr: the helper-side expression the outer_col was bound to (for plan legibility)

If no stage is lifted, the 7-tuple flows through unchanged. */
(define iap_build_pos_map (lambda (iap_tables) (begin
	/* Walk tables in order with an explicit index counter in a 2-element
	accumulator (next_index, current_map) so reduce stays pure and robust
	to table-descriptor shape variants — we only extract the alias via a
	single match pattern. */
	(define iap_acc (reduce iap_tables (lambda (iap_state iap_td) (begin
		(define iap_i (nth iap_state 0))
		(define iap_m (nth iap_state 1))
		(define iap_tv (match iap_td '(tv _ _ _ _) tv _ nil))
		(if (nil? iap_tv)
			(list (+ iap_i 1) iap_m)
			(list (+ iap_i 1) (merge iap_m (list (list iap_tv iap_i))))))) (list 0 '())))
	(nth iap_acc 1))))

(define iap_pos_of (lambda (iap_map iap_tv)
	(reduce iap_map (lambda (iap_acc iap_entry)
		(if (nil? iap_acc)
			(if (equal? (nth iap_entry 0) iap_tv) (nth iap_entry 1) nil)
			iap_acc)) nil)))

(define inject_anti_passes (lambda (iapreorder_result) (match iapreorder_result
	'(iapschema iaptables iapfields iapcondition iapgroups iapschemas iaprfcol) (begin
		(define iap_pos_map (iap_build_pos_map iaptables))
		(define iap_augmented (map (coalesceNil iapgroups '()) (lambda (iap_stage)
			(begin
				(define iap_os (stage_outer_sources iap_stage))
				(define iap_aliases (stage_partition_aliases iap_stage))
				(if (or (nil? iap_os) (equal? iap_os '())
					(nil? iap_aliases) (equal? iap_aliases '()))
					iap_stage
					(begin
						(define iap_helper_tv (car iap_aliases))
						(define iap_src (car iap_os))
						(define iap_outer_tv (nth iap_src 0))
						(define iap_hp (iap_pos_of iap_pos_map iap_helper_tv))
						(define iap_op (iap_pos_of iap_pos_map iap_outer_tv))
						(if (and (not (nil? iap_hp)) (not (nil? iap_op)) (< iap_hp iap_op))
							(cons (list (quote anti-pass-needed) iap_helper_tv iap_outer_tv (nth iap_src 1) (nth iap_src 2)) iap_stage)
							iap_stage))))
		)))
		(list iapschema iaptables iapfields iapcondition iap_augmented iapschemas iaprfcol))
	_ iapreorder_result)))

/* Accessor for the anti-pass-needed marker attached by inject_anti_passes.
Marker shape: (anti-pass-needed helper_tv outer_tv outer_col inner_expr) */
(define stage_anti_pass_marker (lambda (iapstage) (reduce iapstage (lambda (iapacc iapitem)
	(if (nil? iapacc) (match iapitem
		(cons (quote anti-pass-needed) iaprest) iaprest
		_ nil)
		iapacc)) nil)))

/* Collect all anti-pass-needed markers from a groups list. Each marker is a
4-tuple (helper_tv outer_tv outer_col inner_expr). */
(define iap_collect_markers (lambda (iap_groups)
	(reduce (coalesceNil iap_groups '()) (lambda (iap_acc iap_stage) (begin
		(define iap_m (stage_anti_pass_marker iap_stage))
		(if (nil? iap_m) iap_acc (merge iap_acc (list iap_m))))) '())))

/* Replace get_column refs on helper alias iap_tv with nil so the anti-pass
emits NULL-extended rows for helper fields. */
(define iap_nullify_helper_refs (lambda (iap_expr iap_tv) (match iap_expr
	'((symbol get_column) a _ _ _) (if (equal?? a iap_tv) nil iap_expr)
	'((quote get_column) a _ _ _) (if (equal?? a iap_tv) nil iap_expr)
	(cons iap_sym iap_args) (cons (iap_nullify_helper_refs iap_sym iap_tv)
		(map iap_args (lambda (iap_a) (iap_nullify_helper_refs iap_a iap_tv))))
	_ iap_expr)))

/* Extract referenced column names for a single alias from an expression. */
(define iap_collect_alias_cols_raw (lambda (iap_expr iap_tv) (match iap_expr
	'((symbol get_column) a _ c _) (if (equal?? a iap_tv) (list c) '())
	'((quote get_column) a _ c _) (if (equal?? a iap_tv) (list c) '())
	(cons iap_sym iap_args) (merge (iap_collect_alias_cols_raw iap_sym iap_tv)
		(reduce (map iap_args (lambda (a) (iap_collect_alias_cols_raw a iap_tv))) merge '()))
	_ '())))
(define iap_collect_alias_cols (lambda (iap_expr iap_tv)
	(reduce (iap_collect_alias_cols_raw iap_expr iap_tv)
		(lambda (acc c) (if (has? acc c) acc (merge acc (list c)))) '())))

/* Lower get_column refs for scalar_scan's filter lambda: helper refs become
lambda params, other refs become (outer alias.col) closure captures. */
(define iap_lower_filter_expr (lambda (iap_expr iap_helper_tv) (match iap_expr
	'((symbol get_column) a _ c _) (if (equal?? a iap_helper_tv)
		(symbol (concat a "." c))
		(list (quote outer) (symbol (concat a "." c))))
	'((quote get_column) a _ c _) (if (equal?? a iap_helper_tv)
		(symbol (concat a "." c))
		(list (quote outer) (symbol (concat a "." c))))
	(cons iap_sym iap_args) (cons (iap_lower_filter_expr iap_sym iap_helper_tv)
		(map iap_args (lambda (ia) (iap_lower_filter_expr ia iap_helper_tv))))
	_ iap_expr)))

/* Build "no helper row matched" predicate for the companion anti-pass plan. */
(define iap_build_antifilter (lambda (iap_helper_schema iap_helper_tbl iap_helper_je iap_helper_tv) (begin
	(define iap_filter_cols (iap_collect_alias_cols iap_helper_je iap_helper_tv))
	(define iap_param_names (map iap_filter_cols (lambda (c) (symbol (concat iap_helper_tv "." c)))))
	(define iap_filter_body (iap_lower_filter_expr iap_helper_je iap_helper_tv))
	(list (quote nil?)
		(list (quote scalar_scan)
			iap_helper_schema iap_helper_tbl
			(cons (quote list) iap_filter_cols)
			(list (quote lambda) iap_param_names iap_filter_body)
			(list (quote list))
			(list (quote lambda) (list) 1)
			(list (quote lambda) (list (quote acc) (quote item)) 1)
			nil
			nil)))))

/* Locate a table descriptor by alias in the reordered tables list. */
(define iap_find_td (lambda (iap_tables iap_alias)
	(reduce iap_tables (lambda (iap_acc iap_td)
		(if (nil? iap_acc)
			(match iap_td '(tv _ _ _ _) (if (equal?? tv iap_alias) iap_td nil) _ nil)
			iap_acc)) nil)))

(define build_queryplan_term_from_logical_with_sink (lambda (logical_term sink_mode) (begin
	(define term_sink_row_expr (lambda (row_expr) (match row_expr
		(cons quote_sym quoted_args)
		(if (and
			(is_quote_scope_sym quote_sym)
			(equal? (count quoted_args) 1)
			(list? (car quoted_args)))
			(runtime_list_ast (car quoted_args))
			row_expr)
		row_expr)))
	(define term_sink_emit_row (lambda (row_expr) (match sink_mode
		'(callback sink_fn) (list sink_fn (term_sink_row_expr row_expr))
		_ (list (symbol "resultrow") row_expr))))
	(define materialized_init_key? (lambda (key)
		(and (string? key)
			(>= (strlen key) 6)
			(equal? (substr key 0 6) "__mat:"))))
	(define materialized_init_context_head? (lambda (head)
		(and (list? head)
			(equal? (count head) 2)
			(or
				(equal? (car head) (quote context))
				(equal? (car head) (list (quote symbol) (quote context)))
				(equal? (car head) (list (quote quote) (quote context))))
			(equal? (cadr head) "session"))))
	(define statement_temp_table_key (lambda (head args) (if (and
		(planner_expr_head_is? head (quote begin))
		(>= (count args) 3))
		(match (car args)
			'(drop_sym schema_expr tbl_expr true)
			(if (and
				(planner_expr_head_is? drop_sym (quote droptable))
				(string? tbl_expr)
				(or
					(strlike tbl_expr ".prejoin:%")
					(strlike tbl_expr ".keytable:%")))
				(concat "temp:" tbl_expr)
				nil)
			nil)
		nil)))
	(define dedupe_materialized_init_ast (lambda (expr seen) (match expr
		(cons head args)
		(if (is_quote_scope_sym head)
			expr
			(begin
				(define temp_key (statement_temp_table_key head args))
				(if (not (nil? temp_key))
					(if (seen temp_key)
						(dedupe_materialized_init_ast (nth args (- (count args) 1)) seen)
						(begin
							(seen temp_key true)
							(cons
								(dedupe_materialized_init_ast head seen)
								(map args (lambda (arg)
									(dedupe_materialized_init_ast arg seen))))))
					(if (and
						(materialized_init_context_head? head)
						(equal? (count args) 2)
						(materialized_init_key? (car args)))
						(begin
							(define key (car args))
							(if (seen key)
								nil
								(begin
									(seen key true)
									(cons head (list key
										(dedupe_materialized_init_ast (cadr args) seen))))))
						(cons
							(dedupe_materialized_init_ast head seen)
							(map args (lambda (arg)
								(dedupe_materialized_init_ast arg seen))))))))
		expr)))
	(define dedupe_materialized_inits (lambda (init seen)
		(dedupe_materialized_init_ast init seen)))
	(define materialized_init_entry_key (lambda (expr) (match expr
		(cons head args)
		(if (and
			(materialized_init_context_head? head)
			(equal? (count args) 2)
			(materialized_init_key? (car args)))
			(car args)
			nil)
		nil)))
	(define collect_materialized_key_refs (lambda (expr) (match expr
		(cons head args)
		(merge_unique (merge
			(collect_materialized_key_refs head)
			(merge (map args collect_materialized_key_refs))))
		_ (if (materialized_init_key? expr) (list expr) '()))))
	(define materialized_init_entry_deps (lambda (expr) (begin
		(define own_key (materialized_init_entry_key expr))
		(filter (collect_materialized_key_refs expr) (lambda (key)
			(not (equal? key own_key)))))))
	(define materialized_init_ready? (lambda (expr done_keys remaining_keys) (begin
		(define deps (materialized_init_entry_deps expr))
		(reduce deps (lambda (ok dep)
			(and ok (or
				(has? done_keys dep)
				(not (has? remaining_keys dep)))))
			true))))
	(define materialized_init_wave_expr (lambda (wave)
		(if (equal? (count wave) 2)
			(cons (quote parallel) wave)
			(if (equal? (count wave) 1)
				(car wave)
				(cons (quote begin) wave)))))
	(define schedule_materialized_inits_loop (lambda (remaining done_keys out) (begin
		(if (equal? remaining '())
			out
			(begin
				(define remaining_keys
					(filter (map remaining materialized_init_entry_key)
						(lambda (key) (not (nil? key)))))
				(define ready
					(filter remaining (lambda (expr)
						(materialized_init_ready? expr done_keys remaining_keys))))
				(if (equal? ready '())
					(merge out remaining)
					(begin
						(define ready_keys
							(filter (map ready materialized_init_entry_key)
								(lambda (key) (not (nil? key)))))
						(define rest
							(filter remaining (lambda (expr)
								(not (has? ready_keys (materialized_init_entry_key expr))))))
						(schedule_materialized_inits_loop
							rest
							(merge_unique (list done_keys ready_keys))
							(merge out (list (materialized_init_wave_expr ready)))))))))))
	(define schedule_materialized_inits (lambda (init)
		(if (or (nil? init) (equal? init '()))
			'()
			(schedule_materialized_inits_loop init '() '()))))
	(if (logical_query_term_is_select_core logical_term)
		(match logical_term '(select_core_term schema tables fields condition groups schemas replace_find_column init) (begin
			(define _uq_7tuple (list schema tables fields condition groups schemas replace_find_column))
			(define _reorder (apply join_reorder _uq_7tuple))
			(define _plan (apply build_queryplan (merge _reorder (list nil))))
			(define _plan (match _reorder
				'(_r_schema _r_tables _r_fields _r_condition _r_groups _r_schemas _r_rfcol) (begin
					(define _r_markers (iap_collect_markers _r_groups))
					(if (equal? _r_markers '()) _plan
						(begin
							(define _anti_plans (map _r_markers (lambda (_ap_m) (begin
								(define _ap_helper_tv (nth _ap_m 0))
								(define _ap_helper_td (iap_find_td _r_tables _ap_helper_tv))
								(if (nil? _ap_helper_td) nil
									(begin
										(define _ap_helper_schema (nth _ap_helper_td 1))
										(define _ap_helper_tbl (nth _ap_helper_td 2))
										(define _ap_helper_je (nth _ap_helper_td 4))
										(define _ap_antifilter (iap_build_antifilter _ap_helper_schema _ap_helper_tbl _ap_helper_je _ap_helper_tv))
										(define _ap_tables (filter _r_tables (lambda (_ap_td) (not (equal?? (nth _ap_td 0) _ap_helper_tv)))))
										(define _ap_fields (map_assoc _r_fields (lambda (_ap_k _ap_v) (iap_nullify_helper_refs _ap_v _ap_helper_tv))))
										(define _ap_groups (filter (coalesceNil _r_groups '()) (lambda (_ap_s) (nil? (stage_anti_pass_marker _ap_s)))))
										(define _ap_condition_raw (iap_nullify_helper_refs (coalesceNil _r_condition true) _ap_helper_tv))
										(define _ap_condition (if (or (nil? _ap_condition_raw) (equal? _ap_condition_raw true))
											_ap_antifilter
											(list (quote and) _ap_condition_raw _ap_antifilter)))
										(apply build_queryplan (list _r_schema _ap_tables _ap_fields _ap_condition _ap_groups _r_schemas _r_rfcol nil))))))))
							(define _valid_anti_plans (filter _anti_plans (lambda (_ap_p) (not (nil? _ap_p)))))
							(if (equal? _valid_anti_plans '()) _plan
								(cons (quote begin) (cons _plan _valid_anti_plans))))))
				_ _plan))
			(define _plan (normalize_quoted_scalar_resultrow_calls _plan))
			(define _dedupe_seen (newsession))
			(define _normalized_init
				(schedule_materialized_inits
					(dedupe_materialized_inits
						(map init normalize_quoted_scalar_resultrow_calls)
						_dedupe_seen)))
			(define _plan (dedupe_materialized_init_ast _plan _dedupe_seen))
			(define _plan_no_init (parallelize_resultrows _plan))
			(define _finalize_statement_plan (lambda (_stmt_plan)
				(dedupe_materialized_init_ast
					(parallelize_resultrows _stmt_plan)
					(newsession))))
			(define _full_plan (_finalize_statement_plan
				(if (equal? _normalized_init '()) _plan (cons (quote begin) (merge _normalized_init (list _plan))))))
			(match sink_mode
				'(callback sink_fn) (_finalize_statement_plan (cons (quote begin) (merge _normalized_init (list
					(list (quote set) (symbol "__term_prev_resultrow") (symbol "resultrow"))
					(list (quote set) (symbol "resultrow")
						(list (quote lambda) (list (symbol "item"))
							(list sink_fn (symbol "item"))))
					_plan_no_init
					(list (quote set) (symbol "resultrow") (symbol "__term_prev_resultrow"))))))
				_ _full_plan)))
		(if (logical_query_term_is_union_all logical_term)
			(match logical_term '(union_all_term branches order limit offset) (begin
				(if (or (nil? branches) (equal? branches '()))
					(error "UNION ALL requires at least one branch"))
				(define branch_meta (map branches (lambda (branch) (begin
					(define branch_cols (logical_query_term_output_cols branch))
					(list branch branch_cols (count branch_cols))
				))))
				(define expected_cols (match branch_meta
					(cons first_meta _) (nth first_meta 2)
					_ 0))
				(define output_cols (match branch_meta
					(cons first_meta _) (nth first_meta 1)
					_ '()))
				(if (not (reduce branch_meta (lambda (ok meta) (and ok (equal? (nth meta 2) expected_cols))) true))
					(error "UNION ALL branches must project the same number of columns"))
				(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
					/* === UNION ALL with ORDER BY / LIMIT / OFFSET ===
					Emit scan_order_multi for materialization-free sorted merge across tables. */
					(begin
						/* Resolve each branch through join_reorder on the already logical select_core. */
						(define resolved_branches (map branches (lambda (branch) (begin
							(if (not (logical_query_term_is_select_core branch))
								(error "UNION ALL ORDER BY requires SELECT branches"))
							(match branch '(select_core_term schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 init2) (begin
								(if (not (equal? (coalesceNil groups2 '()) '()))
									(error "UNION ALL ORDER BY with staged branches not yet supported"))
								(if (not (equal? (coalesceNil init2 '()) '()))
									(error "UNION ALL ORDER BY with initialized branches not yet supported"))
								(define _uq7 (list schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2))
								(define _jr (apply join_reorder _uq7))
								(define jr_tables (nth _jr 1))
								(if (not (equal? (count jr_tables) 1))
									(error "UNION ALL ORDER BY requires single-table branches (no joins)"))
								(define tbldef (car jr_tables))
								(define jr_fields (nth _jr 2))
								(define jr_condition ((nth _jr 6) (coalesceNil (nth _jr 3) true)))
								(list tbldef jr_fields jr_condition))
								_ (error "UNION ALL ORDER BY requires SELECT branches"))
						))))

						/* Parse ORDER BY: resolve each item to position in output_cols */
						(define order_items (map order (lambda (item) (match item '(col dir) (begin
							(define col_name (match col
								'((symbol get_column) _ _ cn _) cn
								'((quote get_column) _ _ cn _) cn
								_ (if (number? col) nil (to_string col))))
							/* Try name match first, then positional */
							(define pos (reduce (produceN expected_cols (lambda (i) i)) (lambda (found i)
								(if (not (nil? found)) found
									(if (equal?? col_name (nth output_cols i)) i nil))) nil))
							(set pos (if (nil? pos)
								(if (and (number? col) (> col 0) (<= col expected_cols))
									(- col 1)
									nil)
								pos))
							(if (nil? pos) (error (concat "UNION ALL ORDER BY: column not found: " col)))
							(list pos dir))
						))))

						/* Build per-branch scan parameters */
						(define scan_specs (map resolved_branches (lambda (rb) (begin
							(define tbldef (nth rb 0))
							(define fields (nth rb 1))
							(define condition (nth rb 2))
							(match tbldef '(tblvar tbl_schema tbl isOuter joinexpr) (begin
								/* filter: columns from condition */
								(define filtercols (merge_unique (list
									(extract_columns_for_tblvar tblvar condition)
									(extract_outer_columns_for_tblvar tblvar condition))))
								(define filter_ast (list (quote lambda)
									(map filtercols (lambda (c) (symbol (concat tblvar "." c))))
									(optimize (replace_columns_from_expr condition))))

								/* fields by position */
								(define field_names (extract_assoc fields (lambda (k v) k)))
								(define field_exprs (extract_assoc fields (lambda (k v) v)))

								/* sort columns for this branch: map ORDER BY positions to physical columns */
								(define sortcols (map order_items (lambda (oi) (match oi '(pos _dir) (begin
									(define expr (nth field_exprs pos))
									(match expr
										'((symbol get_column) (eval tblvar) _ col _) col
										'((quote get_column) (eval tblvar) _ col _) col
										_ (begin
											/* complex expression: emit lambda-based sort column */
											(define sort_expr_cols (extract_columns_for_tblvar tblvar expr))
											(list (quote lambda)
												(map sort_expr_cols (lambda (c) (symbol (concat tblvar "." c))))
												(replace_columns_from_expr expr)))))))))

								/* map: all columns needed for output field expressions + sort cols */
								(define all_output_cols (reduce (extract_assoc fields (lambda (k v)
									(extract_columns_for_tblvar tblvar v)))
									(lambda (acc cols) (merge_unique acc cols))
									'()))
								(define sort_phys_cols (merge_unique (map sortcols (lambda (sc)
									(if (string? sc) (list sc)
										(match sc
											'((quote lambda) params body) (extract_columns_for_tblvar tblvar body)
											'((symbol lambda) params body) (extract_columns_for_tblvar tblvar body)
											'()))))))
								(define mapcols (merge_unique (list all_output_cols sort_phys_cols)))

								/* map lambda: emit rows with normalized output aliases */
								(define map_ast (list (quote lambda)
									(map mapcols (lambda (c) (symbol (concat tblvar "." c))))
									(term_sink_emit_row
										(cons (symbol "list")
											(merge (map (produceN expected_cols (lambda (i) i)) (lambda (i)
												(list (nth output_cols i) (replace_columns_from_expr (nth field_exprs i))))))))))

								(list tbl_schema tbl filtercols filter_ast sortcols mapcols map_ast))
								_ (error "invalid table definition in UNION ALL branch"))
						))))

						/* Sort directions (shared): extract from order_items */
						(define sort_dirs (map order_items (lambda (oi) (match oi '(_pos dir) dir))))

						(define limit_val (if (nil? limit) -1 limit))
						(define offset_val (if (nil? offset) 0 offset))

						/* Emit scan_order_multi call. Per-table offset/limit are nil here
						(no per-branch ORDER+LIMIT in this codepath); if a branch ever carries
						its own order+limit, populate the nil lists with per-branch ints. */
						(merge (list (symbol "scan_order_multi") '(context_session_get "__memcp_tx"))
							(list
								(cons (symbol "list") (map scan_specs (lambda (s) (list (symbol "table") (nth s 0) (nth s 1)))))
								(cons (symbol "list") (map scan_specs (lambda (s) (cons (symbol "list") (nth s 2)))))
								(cons (symbol "list") (map scan_specs (lambda (s) (nth s 3))))
								(cons (symbol "list") (map scan_specs (lambda (s) (cons (symbol "list") (nth s 4)))))
								(cons (symbol "list") sort_dirs)
								nil
								nil
								0
								offset_val
								limit_val
								(cons (symbol "list") (map scan_specs (lambda (s) (cons (symbol "list") (nth s 5)))))
								(cons (symbol "list") (map scan_specs (lambda (s) (nth s 6))))
						))
					)
					/* === UNION ALL without ORDER BY === */
					(begin
						(define branch_plans (map (produceN (count branch_meta)) (lambda (branch_idx) (begin
							(define meta (nth branch_meta branch_idx))
							(define branch (nth meta 0))
							(define branch_sink_sym (symbol (concat "__union_branch_sink:" branch_idx)))
							(define branch_plan (build_queryplan_term_from_logical_with_sink branch (list (quote callback) branch_sink_sym)))
							(define normalized_row (cons (quote list) (merge (map (produceN expected_cols) (lambda (idx)
								(list (nth output_cols idx) (list (quote nth) (symbol "row") (+ (* idx 2) 1)))
							)))))
							(list (quote begin)
								(list (quote define) branch_sink_sym
									(list (quote lambda) (list (symbol "row"))
										(term_sink_emit_row normalized_row)))
								branch_plan)
						))))
						(cons (quote begin) branch_plans))
				)
			))
			(error "invalid logical query term"))
	)
))))
(define build_queryplan_term_from_logical (lambda (logical_term)
	(build_queryplan_term_from_logical_with_sink logical_term '(resultrow))
))
(define qpp-tuple-contains-window? (lambda (tuple)
	(if (not (qpp-tuple? tuple)) false
		(begin
			(define field-has-window
				(reduce (qpp-fields-to-pairs (qpp-tuple-fields tuple)) (lambda (acc pair) (match pair
					'(_ expr) (or acc (not (equal? (extract_window_funcs expr) '())))
					acc)) false))
			(define order-has-window
				(reduce (coalesceNil (qpp-tuple-order tuple) '()) (lambda (acc item) (match item
					'(expr _dir) (or acc (not (equal? (extract_window_funcs expr) '())))
					acc)) false))
			(or field-has-window
				(not (equal? (extract_window_funcs (coalesceNil (qpp-tuple-condition tuple) true)) '()))
				(not (equal? (extract_window_funcs (coalesceNil (qpp-tuple-having tuple) true)) '()))
				order-has-window)))))

(define qpp-tuple-contains-window-recursive? (lambda (tuple)
	(if (not (qpp-tuple? tuple)) false
		(or
			(qpp-tuple-contains-window? tuple)
			(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (acc td)
				(or acc (match td
					'(_ _ subquery _ _)
					(qpp-tuple-contains-window-recursive? subquery)
					false))) false)))))

(define qpp-expr-inner-select-contains-window? (lambda (expr)
	(match expr
		(cons sym args)
		(begin
			(define kind (qpl-marker-kind expr))
			(or
				(match kind
					'inner_select
					(match args
						'(subquery)
						(qpp-tuple-contains-window-recursive? subquery)
						false)
					'inner_select_exists
					(match args
						'(subquery)
						(qpp-tuple-contains-window-recursive? subquery)
						false)
					'inner_select_in
					(match args
						'(_target subquery)
						(qpp-tuple-contains-window-recursive? subquery)
						false)
					false)
				(reduce (coalesceNil args '()) (lambda (acc arg)
					(or acc (qpp-expr-inner-select-contains-window? arg))) false)))
		false)))

(define qpp-tuple-inner-select-contains-window? (lambda (tuple)
	(if (not (qpp-tuple? tuple)) false
		(or
			(reduce (qpp-fields-to-pairs (qpp-tuple-fields tuple)) (lambda (acc pair)
				(or acc (match pair
					'(_ expr) (qpp-expr-inner-select-contains-window? expr)
					false))) false)
			(qpp-expr-inner-select-contains-window? (qpp-tuple-condition tuple))
			(reduce (coalesceNil (qpp-tuple-group tuple) '()) (lambda (acc expr)
				(or acc (qpp-expr-inner-select-contains-window? expr))) false)
			(qpp-expr-inner-select-contains-window? (qpp-tuple-having tuple))
			(reduce (coalesceNil (qpp-tuple-order tuple) '()) (lambda (acc item)
				(or acc (match item
					'(expr _dir) (qpp-expr-inner-select-contains-window? expr)
					(qpp-expr-inner-select-contains-window? item)))) false)
			(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (acc td)
				(or acc (match td
					'(_ _ subquery _ _)
					(qpp-tuple-inner-select-contains-window? subquery)
					false))) false)))))

(define qpp-tuple-has-inner-select-marker-recursive? (lambda (tuple)
	(if (not (qpp-tuple? tuple)) false
		(or
			(qpp-tuple-has-inner-select-marker? tuple)
			(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (acc td)
				(or acc (match td
					'(_ _ subquery _ _)
					(if (and (qpp-tuple? subquery) (qpp-tuple-contains-window? subquery))
						false
						(qpp-tuple-has-inner-select-marker-recursive? subquery))
					false))) false)))))

(define qpp-expr-has-inner-select-marker-outside-window? (lambda (expr)
	(match expr
		(cons sym args)
		(if (or (equal? sym (symbol window_func))
			(equal? sym (quote window_func))
			(equal? sym '(quote window_func))
			(equal? sym 'window_func))
			false
			(or
				(match sym
					(symbol inner_select) true
					(quote inner_select) true
					'(quote inner_select) true
					'inner_select true
					(symbol inner_select_in) true
					(quote inner_select_in) true
					'(quote inner_select_in) true
					'inner_select_in true
					(symbol inner_select_exists) true
					(quote inner_select_exists) true
					'(quote inner_select_exists) true
					'inner_select_exists true
					false)
				(qpp-expr-has-inner-select-marker-outside-window? sym)
				(reduce (coalesceNil args '()) (lambda (acc a)
					(or acc (qpp-expr-has-inner-select-marker-outside-window? a))) false)))
		false)))

(define qpp-tuple-has-inner-select-marker-outside-window? (lambda (sub)
	(if (not (qpp-tuple? sub))
		false
		(or
			(reduce (qpp-fields-to-pairs (qpp-tuple-fields sub)) (lambda (acc pair) (match pair
				'(_ expr) (or acc (qpp-expr-has-inner-select-marker-outside-window? expr))
				acc)) false)
			(qpp-expr-has-inner-select-marker-outside-window? (qpp-tuple-condition sub))
			(reduce (coalesceNil (qpp-tuple-group sub) '()) (lambda (acc expr)
				(or acc (qpp-expr-has-inner-select-marker-outside-window? expr))) false)
			(qpp-expr-has-inner-select-marker-outside-window? (qpp-tuple-having sub))
			(reduce (coalesceNil (qpp-tuple-order sub) '()) (lambda (acc item) (match item
				'(expr _dir) (or acc (qpp-expr-has-inner-select-marker-outside-window? expr))
				(or acc (qpp-expr-has-inner-select-marker-outside-window? item)))) false)))))

(define qpp-tuple-has-inner-select-marker-outside-window-recursive? (lambda (tuple)
	(if (not (qpp-tuple? tuple)) false
		(or
			(qpp-tuple-has-inner-select-marker-outside-window? tuple)
			(reduce (coalesceNil (qpp-tuple-tables tuple) '()) (lambda (acc td)
				(or acc (match td
					'(_ _ subquery _ _)
					(qpp-tuple-has-inner-select-marker-outside-window-recursive? subquery)
					false))) false)))))

(define neumann_assert_marker_free_tuple (lambda (label tuple)
	(if (qpp-tuple-has-inner-select-marker-recursive? tuple)
		(error (concat label ": residual inner_select marker after Neumann top-down decorrelation"))
		tuple)))

(define neumann_prepare_select_logical_term (lambda (query) (begin
	/* Subquery decorrelation is not allowed to fall back to the legacy
	replace_inner_selects/materialization path. The Neumann pipeline must
	remove every marker first; the old planner may only translate the
	marker-free SELECT shape into its physical stage representation. */
	(define has-window (qpp-tuple-contains-window? query))
	(define has-marker (qpp-tuple-has-inner-select-marker-recursive? query))
	(define has-outside-marker
		(qpp-tuple-has-inner-select-marker-outside-window-recursive? query))
	(if (and has-window (not has-outside-marker))
		(untangle_query_term query nil)
		(begin
			(define neu (neumann_compile_select query))
			(if neumann_pipeline_trace (begin
				(print "[neumann] input:   " query)
				(print "[neumann] lowered: " neu)) nil)
			(neumann_assert_marker_free_tuple "neumann_prepare_select_logical_term" neu)
			(untangle_query_term neu nil))))))

	(define prepare_queryplan_term (lambda (query) (if (nil? query)
		nil
		(begin
		(define rewritten_query (rewrite_query_term query))
		(define union_parts (query_union_all_parts rewritten_query))
	(if (nil? union_parts)
		(if (query_is_select_core rewritten_query)
			(if (or
				(qpp-tuple-contains-window? rewritten_query)
				(qpp-tuple-inner-select-contains-window? rewritten_query))
				(untangle_query_term rewritten_query nil)
				(neumann_prepare_select_logical_term rewritten_query))
				(error "invalid SELECT query term"))
			(match union_parts '(branches order limit offset) (begin
				(if (or (nil? branches) (equal? branches '()))
					(error "UNION ALL requires at least one branch"))
				(list (quote union_all_term)
					(map branches prepare_queryplan_term)
					order limit offset))))))))

(define route_queryplan_term (lambda (query) (begin
	/* Compatibility wrapper for older call sites. New code should call
	prepare_queryplan_term and work on logical terms directly. */
	(define logical_term (prepare_queryplan_term query))
	(match logical_term
		'(select_core_term schema tables fields condition groups schemas replace_find_column init)
		(list schema tables fields condition groups schemas replace_find_column)
		_ logical_term))))

	(define build_queryplan_term_with_sink (lambda (query sink_mode)
		(if (nil? query)
			nil
			(build_queryplan_term_from_logical_with_sink (prepare_queryplan_term query) sink_mode))
	))

(define build_queryplan_term (lambda (query) (begin
	(build_queryplan_term_with_sink query '(resultrow))
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
column resolution — then projects $update through the target table's scan.
The actual mutation is executed only via a temporary resultrow wrapper after the
full WHERE/join pipeline reached its final leaf. Keep this contract: inner scans
must stay pure row/filter pipelines, and DML side effects happen only at the
same boundary where SELECT would emit result rows. */
(define build_dml_plan (lambda (schema target_tbl target_alias all_defs cols condition order limit_val offset_val) (begin
	(define tgt (coalesce target_alias target_tbl))
	(define is_update (and (not (nil? cols)) (not (equal? cols '()))))
	(define dml_table_aliases (lambda (defs)
		(map defs (lambda (td) (match td
			'(alias_ _ tbl_ _ _) (string (if (nil? alias_) tbl_ alias_))
			nil)))))
	(define dml_inner_select_kind (lambda (sym)
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
				_ nil))))
	(define dml_inner_select_subquery (lambda (sym args) (begin
		(define kind (dml_inner_select_kind sym))
		(match kind
			(quote inner_select) (match args (cons subquery '()) subquery nil)
			(quote inner_select_exists) (match args (cons subquery '()) subquery nil)
			(quote inner_select_in) (match args (cons _target (cons subquery '())) subquery nil)
			nil))))
	(define dml_query_local_aliases (lambda (query) (match query
		'(_ qtables _ _ _ _ _ _ _)
		(dml_table_aliases qtables)
		'())))
	(define dml_get_column_head? (lambda (sym)
		(or
			(equal? sym (quote get_column))
			(equal? sym '(quote get_column))
			(equal? sym '(symbol get_column))
			(equal? (string sym) "get_column"))))
	(define dml_free_aliases_expr (lambda (expr local_aliases) (match expr
		(cons sym args) (begin
			(define subquery (dml_inner_select_subquery sym args))
			(if (dml_get_column_head? sym)
				(match args
					'(alias_ _ _ _)
					(if (and (not (nil? alias_)) (not (has? local_aliases (string alias_)))) (list (string alias_)) '())
					'())
				(if (nil? subquery)
					(merge_unique (map args (lambda (arg) (dml_free_aliases_expr arg local_aliases))))
					(match subquery
						'(_ _ fields2 condition2 group2 having2 order2 _ _)
						(begin
							(define nested_local_aliases (merge_unique (list local_aliases (dml_query_local_aliases subquery))))
							(merge_unique (list
								(merge (extract_assoc fields2 (lambda (_k v) (dml_free_aliases_expr v nested_local_aliases))))
								(dml_free_aliases_expr condition2 nested_local_aliases)
								(merge (map (coalesceNil group2 '()) (lambda (g) (dml_free_aliases_expr g nested_local_aliases))))
								(dml_free_aliases_expr having2 nested_local_aliases)
								(merge (map (coalesceNil order2 '()) (lambda (o) (match o
									'(col _dir) (dml_free_aliases_expr col nested_local_aliases)
									(dml_free_aliases_expr o nested_local_aliases)))))))
							'())))))
		'())))
	(define dml_base_aliases (dml_table_aliases all_defs))
	(define dml_condition_serialized (serialize condition))
	(define dml_local_alias_in_condition? (lambda (alias_)
		(strlike dml_condition_serialized
			(concat "%\"" alias_ "\" \"" schema "\"%"))))
	(define dml_extra_alias_candidates
		(if (strlike dml_condition_serialized "%(or %")
			(merge_unique (list
				(dml_free_aliases_expr condition dml_base_aliases)
				(extract_tblvars condition)))
			(dml_free_aliases_expr condition dml_base_aliases)))
	(define dml_extra_aliases
		(filter dml_extra_alias_candidates
			(lambda (alias_) (and
				(not (has? dml_base_aliases alias_))
				(string? alias_)
				(not (nil? (show schema alias_)))))))
	(define dml_all_defs (merge all_defs
		(map dml_extra_aliases (lambda (alias_)
			(list alias_ schema alias_ false nil)))))
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
	(define synthetic_select (list schema dml_all_defs set_fields condition nil nil order limit_val offset_val))
	/* Run through untangle_query → join_reorder → build_queryplan */
	(define _uq_result (apply untangle_query (merge synthetic_select (list nil))))
	(define _uq_init (if (>= (count _uq_result) 8) (nth _uq_result 7) '()))
	(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
	(define pipeline_result (apply join_reorder _uq_7tuple))
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
	/* Assemble final pipeline args.
	For UPDATE we must keep the real resolved SET expressions in fields so the
	planner still pulls all helper/materialized columns needed by those
	expressions through later scans. DML output is still suppressed by
	update_target; the fields only preserve planning dependencies. */
	(define final_fields (if is_update
		(nth pipeline_result 2)
		'("$dml" 1)))
	(define dml_tag (concat "__dml:" (fnv_hash (concat schema "|" target_tbl "|" tgt "|" cols "|" condition "|" order "|" limit_val "|" offset_val))))
	(define final_pipeline (list
		(nth pipeline_result 0) /* schema */
		(nth pipeline_result 1) /* tables */
		final_fields
		(nth pipeline_result 3) /* condition */
		(nth pipeline_result 4) /* groups */
		(nth pipeline_result 5) /* schemas */
		(nth pipeline_result 6) /* replace_find_column */
		(list tgt resolved_target_cols dml_tag) /* update_target: (alias cols dml_tag) — empty cols = DELETE */
	))
	(define dml_plan (apply build_queryplan final_pipeline))
	(define dml_prev_rr (symbol "__dml_prev_resultrow"))
	(define dml_rc (symbol "__dml_result_count"))
	(define wrapped_plan (list (quote begin)
		(list (quote set) dml_prev_rr (symbol "resultrow"))
		(list (quote set) (symbol "resultrow")
			(list (quote lambda) (list (symbol "item"))
				(list (quote if) (list (quote or)
					(list (quote nil?) (list (quote get_assoc) (symbol "item") "__update"))
					(list (quote not) (list (quote equal??) (list (quote get_assoc) (symbol "item") "__dml_tag") dml_tag)))
					0
					(list (quote if) (list (quote nil?) (list (quote get_assoc) (symbol "item") "__values"))
						(list (quote if) (list (quote apply) (list (quote get_assoc) (symbol "item") "__update") nil) 1 0)
						(list (quote if) (list (quote apply) (list (quote get_assoc) (symbol "item") "__update") (list (quote list) (list (quote get_assoc) (symbol "item") "__values"))) 1 0)))))
		(list (quote define) dml_rc dml_plan)
		(list (quote set) (symbol "resultrow") dml_prev_rr)
		dml_rc))
	(if (equal? _uq_init '()) wrapped_plan (cons (quote begin) (merge _uq_init (list wrapped_plan))))
)))

/* Convenience wrapper for multi-table UPDATE (called from sql_update) */
(define build_multi_table_update (lambda (schema tbl tblalias all_defs cols condition)
	(build_dml_plan schema tbl tblalias all_defs cols condition nil nil nil)))

/*
=== CONTRACT: build_queryplan ===

PURPOSE: Generate physical execution plans from the logical IR.
Takes a flat, already-reordered table list and translates it into executable SCM.

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
- Re-introduce logical subquery semantics. If build_queryplan still needs
inner_select/subscan/materialized-derived-source behavior, untangle_query has
not finished its job.

GROUP BY AGGREGATE PIPELINE:
1. collect_plan: extract unique group keys from base table into a keytable
2. compute_plan: for each aggregate, scan base table per group key,
store results as keytable columns named "expr|condition"
3. grouped_plan: scan populated keytable for final output (ORDER BY, HAVING, LIMIT)
*/
/*
=== build_queryplan: physical plan generation ===

Translates the flat logical IR from untangle_query into executable SCM scan code.
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
(define _build_queryplan_inner (lambda (schema tables fields condition groups schemas replace_find_column update_target) (begin

	/* TODO: order tables: outer joins behind */
	(define schemas (schema_bindings_to_flat_list schemas))
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
	(define _group_stages (filter groups (lambda (s) (begin
		(define _sg (stage_group_cols s))
		(and (not (nil? _sg)) (not (equal? _sg '())))))))
	(define _non_group_stages (filter groups (lambda (s) (begin
		(define _sg (stage_group_cols s))
		(or (nil? _sg) (equal? _sg '()))))))
	(set groups (merge _group_stages _non_group_stages))
	(define redundant_global_scalar_limit_stage? (lambda (s)
		(and
			(equal? (coalesceNil (stage_group_cols s) '()) '())
			(equal? (coalesceNil (stage_order_list s) '()) '())
			(not (nil? (stage_limit_val s)))
			(<= (stage_limit_val s) 1)
			(or (nil? (stage_offset_val s)) (equal? (stage_offset_val s) 0))
			(nil? (stage_partition_aliases s))
			(nil? (stage_condition s))
			(nil? (stage_once_limit s)))))
	(set groups (match groups
		(cons first_stage rest_stage_list)
		(if (and
			(redundant_global_scalar_limit_stage? first_stage)
			(not (nil? rest_stage_list))
			(not (equal? rest_stage_list '())))
			rest_stage_list
			groups)
		groups))
	(define groups_present (and (not (nil? groups)) (not (equal? groups '()))))
	(define stage (if groups_present (car groups) nil))
	(define rest_groups (if groups_present (cdr groups) nil))
	(set rest_groups (coalesceNil rest_groups '()))
	(define stage_is_scoped (and stage (not (nil? (stage_partition_aliases stage)))))
	(define stage_group (if stage (stage_group_cols stage) nil))
	(define stage_having (if stage (stage_having_expr stage) nil))
	(define stage_order (if (and stage (not stage_is_scoped)) (stage_order_list stage) nil))
	(define stage_partcols (if (and stage (not stage_is_scoped)) (coalesceNil (stage_limit_partition_cols stage) 0) 0))
	(define stage_limit (if (and stage (not stage_is_scoped)) (stage_limit_val stage) nil))
	(define stage_offset (if (and stage (not stage_is_scoped)) (stage_offset_val stage) nil))

	/* window function detection */
	(define window_funcs_all
		(reduce_assoc fields (lambda (acc _k v)
			(merge acc (extract_window_funcs v)))
			'()))
	(define has_window (not (equal? window_funcs_all '())))
	(define grouped_navigation_window
		(and stage_group
			(reduce window_funcs_all (lambda (found wf) (match wf
				'(fn _ _)
				(or found (or (equal?? fn "LEAD") (equal?? fn "LAG")))
				found))
				false)))
	(if grouped_navigation_window
		(error "navigation window functions over grouped rows are not supported")
		nil)
	/* Case 10: window functions in WHERE clause */
	(define window_in_condition (not (equal? (extract_window_funcs (coalesceNil condition true)) '())))
	/* Design contract:
	Materialized temp sources may expose aggregate results as physical temp
	columns, but logical aggregate sentinels must survive until the scan stage
	that actually reads that temp source. Lower them exactly once here. */
	(define lower_materialized_scan_expr (lambda (scan_schema scan_tbl scan_tblvar scan_expr agg_name_context) (begin
		(define materialized_source (materialized-source? scan_tbl))
		(if (not materialized_source)
			scan_expr
			(begin
				(define canon_alias_map (list (list scan_tblvar (concat scan_schema "." scan_tbl))))
				(define scan_expr_name (lambda (expr)
					(canonical_expr_name (normalize_canonical_aliases (preserve_current_materialized_field_refs scan_tbl scan_tblvar expr)) '(list) '(list) canon_alias_map)))
				(define agg_col_name (make_aggregate_cache_col_name scan_expr_name agg_name_context nil))
				(define materialized_cols (materialized_source_physical_schema scan_schema scan_tbl scan_tblvar schemas))
				(define kt_source_col (lambda (col)
					(if (and (string? col) (>= (strlen col) 5)
						(equal? (substr col 0 5) "__kt_"))
						(substr col 5 (- (strlen col) 5))
						col)))
				(define materialized_positive_count_bool_expr? (lambda (expr) (match expr
					'(op left right)
					(and
						(or
							(equal? op (quote >))
							(equal? op (symbol >))
							(equal? op '(quote >)))
						(or (equal? right 0) (equal? right 0.0))
						(match left
							'((symbol aggregate) _ (symbol +) 0) true
							'((quote aggregate) _ (symbol +) 0) true
							'(coalesce_op inner fallback)
							(and
								(or
									(equal? coalesce_op (quote coalesceNil))
									(equal? coalesce_op (symbol coalesceNil))
									(equal? coalesce_op '(quote coalesceNil)))
								(or (equal? fallback 0) (equal? fallback 0.0))
								(materialized_positive_count_bool_expr? (list (quote >) inner 0)))
							'((symbol get_column) _ _ col _)
							(and (string? col) (strlike col "%COUNT(*)%"))
							'((quote get_column) _ _ col _)
							(and (string? col) (strlike col "%COUNT(*)%"))
							false))
					false)))
				(define materialized_col_false_on_empty? (lambda (col)
					(reduce materialized_cols (lambda (found coldef)
						(or found
							(and
								(equal? (coldef "Field") col)
								(materialized_positive_count_bool_expr? (coalesceNil (coldef "Expr") nil)))))
						false)))
				(define materialized_read_with_empty_default (lambda (col node)
					(if (materialized_col_false_on_empty? col)
						(list (quote coalesceNil) node false)
						node)))
				(define lower_materialized_key_ref (lambda (node) (match node
					'((symbol get_column) (eval scan_tblvar) _ col _) (begin
						(define source_col (kt_source_col col))
						(materialized_read_with_empty_default source_col
							(if (and (not (equal? source_col col))
								(nil? (find_materialized_field_by_name materialized_cols col))
								(not (nil? (find_materialized_field_by_name materialized_cols source_col))))
								(list (quote get_column) scan_tblvar false source_col false)
								node)))
					'((quote get_column) (eval scan_tblvar) _ col _) (begin
						(define source_col (kt_source_col col))
						(materialized_read_with_empty_default source_col
							(if (and (not (equal? source_col col))
								(nil? (find_materialized_field_by_name materialized_cols col))
								(not (nil? (find_materialized_field_by_name materialized_cols source_col))))
								(list (quote get_column) scan_tblvar false source_col false)
								node)))
					_ node)))
				(define lookup_expr_field (lambda (expr) (begin
					(define expr_lookup (materialized_source_expr_lookup scan_tbl))
					(define expr_keys (materialized_source_expr_keys expr))
					(define direct_field (if (nil? expr_lookup) nil
						(reduce expr_keys (lambda (found key)
							(if (not (nil? found))
								found
								(coalesce (expr_lookup key) nil)))
							nil)))
					(if (not (nil? direct_field))
						direct_field
						(begin
							(define normalized_expr (normalize_canonical_aliases expr))
							(reduce materialized_cols (lambda (found coldef)
								(if (not (nil? found))
									found
									(begin
										(define source_expr (coalesceNil (coldef "Expr") nil))
										(if (and (not (nil? source_expr))
											(or (equal? (normalize_canonical_aliases source_expr) normalized_expr)
												(reduce expr_keys (lambda (matched key)
													(or matched (has? (materialized_source_expr_keys source_expr) key)))
													false)))
											(coldef "Field")
											nil))))
								nil))))))
				(define current_scan_agg_field (lambda (expr agg_args)
					(coalesce
						(lookup_expr_field expr)
						(begin
							(find_materialized_field_by_name materialized_cols (agg_col_name agg_args))))))
				(define lower_aggs (lambda (expr) (match expr
					'((symbol get_column) (eval scan_tblvar) _ _ _)
					(lower_materialized_key_ref expr)
					'((quote get_column) (eval scan_tblvar) _ _ _)
					(lower_materialized_key_ref expr)
					(cons (symbol aggregate) agg_args) (begin
						(define match_col (current_scan_agg_field expr agg_args))
						(if (nil? match_col)
							expr
							(list (quote get_column) scan_tblvar false match_col false)))
					(cons '(quote aggregate) agg_args) (begin
						(define match_col (current_scan_agg_field expr agg_args))
						(if (nil? match_col)
							expr
							(list (quote get_column) scan_tblvar false match_col false)))
					(cons sym args) (cons sym (map args lower_aggs))
					expr)))
				(lower_aggs scan_expr))))))
	(define lower_materialized_scan_condition (lambda (scan_schema scan_tbl scan_tblvar scan_condition)
		(lower_materialized_scan_expr scan_schema scan_tbl scan_tblvar scan_condition scan_condition)))
	(define lower_materialized_emit_expr (lambda (scan_schema scan_tbl scan_tblvar scan_expr)
		(lower_materialized_scan_expr scan_schema scan_tbl scan_tblvar scan_expr true)))
	(define lower_materialized_emit_assoc (lambda (scan_schema scan_tbl scan_tblvar exprs)
		(map_assoc exprs (lambda (k v) (lower_materialized_emit_expr scan_schema scan_tbl scan_tblvar v)))))
	(if window_in_condition (error "window functions not allowed in WHERE clause"))

	/* window functions with GROUP BY: strip window expressions to inner
	aggregates so the normal GROUP BY path processes them. Save original
	fields so we can inject promise values after compute_plan. */
	(define _wg_store (newsession))
	(_wg_store "fields" nil)
	(_wg_store "stripped-fields" nil)
	(if (and has_window stage_group) (begin
		(_wg_store "fields" fields) /* save original fields with window_func */
		(define strip_window_inner (lambda (expr)
			(match expr
				(cons (symbol window_func) wf_rest)
				(begin
					(define args (cadr wf_rest))
					(if (and (list? args) (> (count args) 0)) (car args) 1))
				(cons (quote window_func) wf_rest)
				(begin
					(define args (cadr wf_rest))
					(if (and (list? args) (> (count args) 0)) (car args) 1))
				(cons sym args) (cons sym (map args strip_window_inner))
				expr)))
		(_wg_store "stripped-fields" (map_assoc fields (lambda (k v) (strip_window_inner v))))))
	(define fields (coalesceNil (_wg_store "stripped-fields") fields))
	(define has_window (if (nil? (_wg_store "stripped-fields")) has_window false))

	(if stage_group (begin
		/* group: extract aggregate clauses and split the query into two parts: gathering the aggregates and outputting them */
		/* Design contract:
		Keep get_column / aggregate / window sentinels logical until the final scan
		code is emitted. A GROUP stage may resolve expressions for its own
		materialization, but when it wraps itself into a recursive prejoin/materialized
		stage it must forward the original logical AST into the next stage. Otherwise
		physical temp field names from an earlier materialization become part of the
		next stage's logical keys/fields and explode into nested "(get_column ...)"
		temp names. */
		(define raw_stage_group stage_group)
		(define raw_stage_having stage_having)
		/* Compatibility name: current planner stores the logical post-group
		predicate in HAVING. Keep an explicit raw alias here so recursive
		prejoin/group planning cannot silently drop it by referring to an
		undefined symbol. */
		(define raw_stage_post_group_condition raw_stage_having)
		(define raw_stage_order stage_order)
		(define raw_fields fields)
		(define raw_stage_condition (stage_condition stage))
		(define condition (if (or (nil? raw_stage_condition) (equal? raw_stage_condition true))
			condition
			(combine_and_terms (list (coalesceNil condition true) raw_stage_condition))))
		(set stage_group (map stage_group replace_find_column))
		(set stage_having (replace_find_column stage_having))
		(set stage_order (map stage_order (lambda (o) (match o '(col dir) (list (replace_find_column col) dir)))))
		(define is_dedup (stage_is_dedup stage))
		(define _scoped_stage (not (nil? (stage_partition_aliases stage))))
		(define _field_agg_has_nested_agg (lambda (args)
			(reduce args (lambda (acc arg)
				(or acc (not (equal? (extract_aggregates arg) '()))))
				false)))
		(define _needs_outer_group_expr (lambda (expr) (match expr
			(cons (symbol aggregate) args)
			(and (equal? (extract_tblvars expr) '()) (_field_agg_has_nested_agg args))
			(cons '(quote aggregate) args)
			(and (equal? (extract_tblvars expr) '()) (_field_agg_has_nested_agg args))
			(cons _ args) (reduce args (lambda (acc arg) (or acc (_needs_outer_group_expr arg))) false)
			false)))
		(define _has_existing_later_group_stage (reduce rest_groups (lambda (acc s)
			(or acc (begin
				(define _later_sg (stage_group_cols s))
				(and (not (nil? _later_sg)) (not (equal? _later_sg '()))))))
			false))
		(define _needs_synthetic_outer_group (and _scoped_stage
			(not _has_existing_later_group_stage)
			(or
				(reduce_assoc fields (lambda (acc _k expr) (or acc (_needs_outer_group_expr expr))) false)
				(_needs_outer_group_expr (coalesce stage_having true))
				(reduce (coalesce stage_order '()) (lambda (acc o)
					(or acc (match o '(col _dir) (_needs_outer_group_expr col) false))) false))))
		(define _has_later_group_stage (or _has_existing_later_group_stage _needs_synthetic_outer_group))
		(define _defer_field_agg (lambda (expr args)
			(and (equal? (extract_tblvars expr) '())
				_scoped_stage
				_has_later_group_stage
				(_field_agg_has_nested_agg args))))
		(define extract_stage_field_aggregates (lambda (expr deferred_outer) (match expr
			(cons (symbol aggregate) args)
			(if (and (not deferred_outer) (_defer_field_agg expr args))
				(merge (map args (lambda (arg) (extract_stage_field_aggregates arg true))))
				(list args))
			(cons '(quote aggregate) args)
			(if (and (not deferred_outer) (_defer_field_agg expr args))
				(merge (map args (lambda (arg) (extract_stage_field_aggregates arg true))))
				(list args))
			(cons sym args) (merge (map args (lambda (arg) (extract_stage_field_aggregates arg deferred_outer))))
			'())))
		/* collect all unique aggregate tuples (expr reduce neutral) from fields, ORDER BY, and HAVING.
		Each tuple becomes a computed column on the keytable, e.g. SUM(amount) -> ((get_column t amount) + 0).
		ORDER BY SUM(x) requires SUM(x) to be pre-computed here even if not in SELECT. */
		(define ags_raw (if is_dedup '() (extract_assoc fields (lambda (key expr) (extract_stage_field_aggregates expr false)))))
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
		(define materialized_output_boundary (equal? (stage_get stage (quote materialized-output-boundary) false) true))
		(define _grp_tables_raw_base (if (not (nil? _stage_scope))
			/* scoped GROUP: only the tables listed in the stage's aliases */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (has? _stage_scope tv) false)))
			/* global GROUP: all tables except partition-staged */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (not (has? _grp_ps_aliases tv)) true)))))
		(define _stage_positive_match_condition? (lambda (expr) (match expr
			'(op left right) (and
				(or
					(equal? op (quote >))
					(equal? op '(quote >))
					(equal? op (symbol ">"))
					(equal? (string op) ">"))
				(not (equal? (extract_aggregates left) '()))
				(or (equal? right 0) (equal? right 0.0)))
			(cons sym args) (reduce args (lambda (found arg)
				(or found (_stage_positive_match_condition? arg))) false)
			false)))
		(define _stage_match_serialized (concat (serialize condition) " " (serialize stage_having)))
		(define _stage_match_has_agg (or
			(not (equal? (extract_aggregates (coalesceNil condition true)) '()))
			(not (equal? (extract_aggregates (coalesceNil stage_having true)) '()))))
		(define _stage_positive_match_filter (or
			(_stage_positive_match_condition? (coalesceNil condition true))
			(_stage_positive_match_condition? (coalesceNil stage_having true))
			(and
				_stage_match_has_agg
				(not (regexp_test _stage_match_serialized "equal"))
				(not (regexp_test _stage_match_serialized "= *0")))))
		(define _grp_tables_raw
			(if (and _scoped_stage _stage_positive_match_filter)
				(map _grp_tables_raw_base (lambda (td) (match td
					'(tv tschema ttbl _ joinexpr) (list tv tschema ttbl false joinexpr)
					td)))
				_grp_tables_raw_base))
		(define _grp_ps_tables_raw_base (filter tables (lambda (t) (match t '(tv _ _ _ _)
			(and (not (has? (coalesceNil _stage_scope '()) tv))
				(or (has? _grp_ps_aliases tv) (not (nil? _stage_scope))))
			false))))
		(define _grp_ps_tables_raw
			(if (and _scoped_stage _stage_positive_match_filter)
				(map _grp_ps_tables_raw_base (lambda (td) (match td
					'(tv tschema ttbl _ joinexpr) (list tv tschema ttbl false joinexpr)
					td)))
				_grp_ps_tables_raw_base))
		(define _grp_ps_visible_aliases (merge_unique (map _grp_ps_tables_raw (lambda (td) (match td
			'(tv tschema ttbl _ _)
			(filter (list
				tv
				(visible_occurrence_alias tv)
				(if (equal? (visible_occurrence_alias tv) ttbl) (concat tschema "." ttbl) nil))
				(lambda (alias_) (not (nil? alias_))))
			'())))))
		(define _grp_table_visible_aliases (merge_unique (map _grp_tables_raw (lambda (td) (match td
			'(tv tschema ttbl _ _)
			(filter (list
				tv
				(visible_occurrence_alias tv)
				(if (equal? (visible_occurrence_alias tv) ttbl) (concat tschema "." ttbl) nil))
				(lambda (alias_) (not (nil? alias_))))
			'())))))
		(define _expr_refs_outside_grp_tables (lambda (expr)
			(reduce (extract_tblvars expr) (lambda (acc alias_)
				(or acc
					(and
						(not (has? _grp_table_visible_aliases alias_))
						(not (has? _grp_table_visible_aliases (visible_occurrence_alias alias_))))))
				false)))
		(define _expr_refs_grp_ps_table (lambda (expr)
			(reduce (extract_all_get_columns expr) (lambda (acc mc)
				(or acc (match mc
					'(name '((symbol get_column) alias_ _ _ _))
					(or (has? _grp_ps_visible_aliases alias_)
						(has? _grp_ps_visible_aliases (visible_occurrence_alias alias_)))
					'(name '((quote get_column) alias_ _ _ _))
					(or (has? _grp_ps_visible_aliases alias_)
						(has? _grp_ps_visible_aliases (visible_occurrence_alias alias_)))
					false)))
				false)))
		(define _resolved_expr_refs_grp_ps_table (lambda (expr)
			(_expr_refs_grp_ps_table (replace_find_column expr))))
		(define _scalar_helper_group_table? (lambda (td) (match td
			'(tv _ ttbl _ _)
			(or
				(and (string? tv)
					(>= (strlen tv) 14)
					(equal? (substr tv 0 14) "domain_scalar_"))
				(strlike (serialize ttbl) "%:((get_column \\\"domain_scalar_%"))
			false)))
		(define _scalar_helper_materialized_keytable? (lambda (td) (match td
			'(tv _ ttbl _ _)
			(and
				(not (and (string? tv)
					(>= (strlen tv) 14)
					(equal? (substr tv 0 14) "domain_scalar_")))
				(strlike (serialize ttbl) "%:((get_column \\\"domain_scalar_%")
				(not (strlike (serialize ttbl) "%|%")))
			false)))
		(define _grp_ps_only_scalar_helpers
			(and
				(not (equal? _grp_ps_tables_raw '()))
				(reduce _grp_ps_tables_raw (lambda (acc td)
					(and acc (_scalar_helper_group_table? td)))
					true)))
		(define _grp_tables_only_scalar_helpers
			(and
				(not (equal? _grp_tables_raw '()))
				(reduce _grp_tables_raw (lambda (acc td)
					(and acc (_scalar_helper_group_table? td)))
					true)))
		(define _grp_output_refs_ps_table
			(reduce_assoc raw_fields (lambda (acc _k expr)
				(or acc
					(_expr_refs_grp_ps_table expr)
					(_resolved_expr_refs_grp_ps_table expr)
					(_expr_refs_outside_grp_tables expr)
					(_expr_refs_outside_grp_tables (replace_find_column expr))))
				false))
		(define _grp_output_refs_ps_table_specific
			(reduce_assoc raw_fields (lambda (acc _k expr)
				(or acc
					(_expr_refs_grp_ps_table expr)
					(_resolved_expr_refs_grp_ps_table expr)))
				false))
		(define _grp_stage_refs_ps_table_specific (or
			(reduce stage_group (lambda (acc expr) (or acc (_resolved_expr_refs_grp_ps_table expr))) false)
			(reduce ags (lambda (acc ag) (or acc (_resolved_expr_refs_grp_ps_table ag))) false)
			(_resolved_expr_refs_grp_ps_table (coalesce condition true))
			(_resolved_expr_refs_grp_ps_table (coalesce stage_having true))
			(reduce (coalesce stage_order '()) (lambda (acc o) (or acc (match o '(col _dir) (_resolved_expr_refs_grp_ps_table col) false))) false)))
		(define _must_prejoin_outer_group_tables (and
			(not _grp_tables_only_scalar_helpers)
			(not (equal? _grp_ps_tables_raw '()))
			(or
				(not _grp_ps_only_scalar_helpers)
				_grp_output_refs_ps_table_specific
				_grp_stage_refs_ps_table_specific)
			(or
				_grp_output_refs_ps_table
				_grp_stage_refs_ps_table_specific)))
		(define _grp_tables (if _must_prejoin_outer_group_tables
			(merge _grp_tables_raw _grp_ps_tables_raw)
			_grp_tables_raw))
		(define _grp_ps_tables (if _must_prejoin_outer_group_tables
			'()
			_grp_ps_tables_raw))
		(define _group_table_alias_variants (lambda (td) (match td
			'(tv tschema ttbl _ _)
			(begin
				(define base_ttbl (scan_tagged_table_base ttbl))
				(filter (list
					tv
					(visible_occurrence_alias tv)
					(if (string? base_ttbl) base_ttbl nil)
					(if (string? base_ttbl) (concat tschema "." base_ttbl) nil))
					(lambda (alias_) (not (nil? alias_)))))
			'())))
		(define _raw_fields_ref_group_table? (lambda (td) (begin
			(define aliases (_group_table_alias_variants td))
			(reduce_assoc raw_fields (lambda (found _k expr)
				(or found
					(reduce (extract_tblvars expr) (lambda (has_ref alias_)
						(or has_ref (has? aliases alias_)))
						false)))
				false))))
		(define _mixed_group_scalar_guard_tables
			(if _grp_tables_only_scalar_helpers
				'()
				(filter _grp_tables (lambda (td)
					(and
						(_scalar_helper_materialized_keytable? td)
						(match td '(_ _ _ _ je) (not (equal? je true)) false))))))
		(define _mixed_group_scalar_guard_join_domain (and
			(not (equal? _mixed_group_scalar_guard_tables '()))
			(not _grp_output_refs_ps_table)))
		(define _mixed_group_scalar_guard_keys (map _mixed_group_scalar_guard_tables serialize))
		(define _grp_tables (filter _grp_tables (lambda (td)
			(not (has? _mixed_group_scalar_guard_keys (serialize td))))))
		(define _grp_ps_tables (merge _grp_ps_tables _mixed_group_scalar_guard_tables))
		(define _split_scalar_group_expr? (lambda (expr)
			(and
				(strlike (serialize expr) "%domain_scalar_%")
				(reduce _mixed_group_scalar_guard_tables (lambda (found td)
					(or found (strlike (serialize td) "%domain_scalar_%")))
					false))))
		(define stage_group (if (not (equal? _mixed_group_scalar_guard_tables '()))
			(filter stage_group (lambda (expr)
				(not (_split_scalar_group_expr? expr))))
			stage_group))
		(define raw_stage_group (if (not (equal? _mixed_group_scalar_guard_tables '()))
			(filter raw_stage_group (lambda (expr)
				(not (_split_scalar_group_expr? expr))))
			raw_stage_group))
		(match _grp_tables
			/* TODO: allow for more than just group by single table */
			/* TODO: outer tables that only join on group */
			'('(tblvar schema tbl isOuter _)) (begin
				(define base_tbl (scan_tagged_table_base tbl))
				(define tagged_scan (scan_tagged_table_needs_scan_order tbl))
				(define tbl_scan_order (scan_tagged_table_order tbl))
				(define tbl_scan_limit (scan_tagged_table_limit tbl))
				(define tbl_scan_offset (scan_tagged_table_offset tbl))
				(define tbl_scan_partcols (scan_tagged_table_partition_cols tbl))
				(define ags (filter ags (lambda (ag) (match ag
					'(agg_expr _ _)
					(begin
						(define refs (extract_tblvars agg_expr))
						(or (materialized-source? base_tbl)
							(equal? refs '())
							(reduce refs (lambda (ok ref)
								(and ok (equal?? ref tblvar)))
								true)))
					false))))
				/* prepare preaggregate */
				(define canon_alias_map (list (list tblvar (concat schema "." base_tbl))))
				(define materialized_source (materialized-source? base_tbl))
				(define tbl_source_expr (if (string? base_tbl)
					(list (quote table) schema base_tbl)
					tbl))
				(define group_value_local_materializable (and materialized_source (string? base_tbl)))
				(define group_value_refs_only_current_source? (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (ok ref)
						(and ok (equal?? ref tblvar)))
						true)))
				(define expr_name (lambda (expr)
					(sanitize_temp_name
						(canonical_expr_name (normalize_canonical_aliases (preserve_current_materialized_field_refs tbl tblvar expr)) '(list) '(list) canon_alias_map))))
				(define group_runtime_terms (runtime_cache_unique_terms (merge
					(runtime_cache_terms_from_exprs (merge (list condition) ags))
					(extract_session_lookup_terms (stage_cache_query stage)))))
				(define group_runtime_suffix (runtime_cache_suffix_from_terms group_runtime_terms))
				(define agg_col_name (make_aggregate_cache_col_name expr_name condition group_runtime_suffix))
				(define rewrite_materialized_source_aggs_single (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args) (begin
						(define target_col (agg_col_name agg_args))
						(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
						(define match_col (if (group_value_refs_only_current_source? agg_args)
							(find_materialized_field_by_name materialized_cols target_col)
							nil))
						(if (nil? match_col)
							expr
							(list (quote get_column) tblvar false match_col false)))
					(cons '(quote aggregate) agg_args) (begin
						(define target_col (agg_col_name agg_args))
						(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
						(define match_col (if (group_value_refs_only_current_source? agg_args)
							(find_materialized_field_by_name materialized_cols target_col)
							nil))
						(if (nil? match_col)
							expr
							(list (quote get_column) tblvar false match_col false)))
					(cons sym args) (cons sym (map args rewrite_materialized_source_aggs_single))
					expr)))
				(define rewrite_materialized_source_cols_single (lambda (expr) (match expr
					'((symbol get_column) _ _ _ _) (begin
						(define expr_lookup (materialized_source_expr_lookup tbl))
						(define visible_field_expr (match expr
							'((symbol get_column) alias_ _ col _)
							(if (or (nil? alias_) (not (has_assoc? schemas alias_))) nil
								(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
									(if (not (nil? found))
										found
										(if (and (equal? (coldef "Field") col) (has_assoc? coldef "Expr"))
											(coalesceNil (coldef "Expr") nil)
											nil)))
									nil))
							nil))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce (materialized_source_expr_keys expr) (lambda (found key)
								(if (not (nil? found)) found
									(coalesce (expr_lookup key) nil)))
								nil)))
						(define direct_field (coalesce direct_field
							(materialized_field_from_get_column_name
								(materialized_source_physical_schema schema tbl tblvar schemas)
								expr)))
						(if (not (nil? direct_field))
							(list (quote get_column) tblvar false direct_field false)
							(begin
								(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
								(define normalized_expr (normalize_canonical_aliases expr))
								(define match_col (reduce materialized_cols (lambda (found coldef)
									(if (not (nil? found))
										found
										(begin
											(define source_expr (coalesceNil (coldef "Expr") nil))
											(if (and (not (nil? source_expr))
												(equal? (normalize_canonical_aliases source_expr) normalized_expr))
												(coldef "Field")
												nil))))
									nil))
								(if (nil? match_col)
									(if (nil? visible_field_expr)
										expr
										(rewrite_materialized_source_cols_single visible_field_expr))
									(list (quote get_column) tblvar false match_col false)))))
					'((quote get_column) _ _ _ _) (begin
						(define expr_lookup (materialized_source_expr_lookup tbl))
						(define visible_field_expr (match expr
							'((quote get_column) alias_ _ col _)
							(if (or (nil? alias_) (not (has_assoc? schemas alias_))) nil
								(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
									(if (not (nil? found))
										found
										(if (and (equal? (coldef "Field") col) (has_assoc? coldef "Expr"))
											(coalesceNil (coldef "Expr") nil)
											nil)))
									nil))
							nil))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce (materialized_source_expr_keys expr) (lambda (found key)
								(if (not (nil? found)) found
									(coalesce (expr_lookup key) nil)))
								nil)))
						(define direct_field (coalesce direct_field
							(materialized_field_from_get_column_name
								(materialized_source_physical_schema schema tbl tblvar schemas)
								expr)))
						(if (not (nil? direct_field))
							(list (quote get_column) tblvar false direct_field false)
							(begin
								(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
								(define normalized_expr (normalize_canonical_aliases expr))
								(define match_col (reduce materialized_cols (lambda (found coldef)
									(if (not (nil? found))
										found
										(begin
											(define source_expr (coalesceNil (coldef "Expr") nil))
											(if (and (not (nil? source_expr))
												(equal? (normalize_canonical_aliases source_expr) normalized_expr))
												(coldef "Field")
												nil))))
									nil))
								(if (nil? match_col)
									(if (nil? visible_field_expr)
										expr
										(rewrite_materialized_source_cols_single visible_field_expr))
									(list (quote get_column) tblvar false match_col false)))))
					(cons sym args) (cons sym (map args rewrite_materialized_source_cols_single))
					expr)))
				(define _visible_materialized_agg_match_col (lambda (td agg_args agg_name)
					(match td
						'(tv tschema ttbl _ _)
						(begin
							(define td_aliases (_group_table_alias_variants td))
							(if (reduce (extract_tblvars agg_args) (lambda (ok ref_alias)
								(and ok (has? td_aliases ref_alias)))
								true)
								(begin
									(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
									(reduce source_cols (lambda (found coldef)
										(if (not (nil? found)) found
											(begin
												(define field_name (coldef "Field"))
												(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
													(equal? (substr field_name 0 (strlen agg_name)) agg_name)
													(equal? (substr field_name (strlen agg_name) 1) "|"))
													(list tv field_name)
													nil))))
										nil))
								nil))
						nil)))
				(define lower_visible_materialized_aggs_single (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args) (begin
						(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
						(define match_col (if (equal? (extract_tblvars expr) '())
							nil
							(reduce tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(_visible_materialized_agg_match_col td agg_args agg_name)
										nil)))
								nil)))
						(if (nil? match_col)
							(match agg_args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_visible_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
								_ expr)
							(list (quote get_column) (car match_col) false (cadr match_col) false)))
					(cons '(quote aggregate) agg_args) (begin
						(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
						(define match_col (if (equal? (extract_tblvars expr) '())
							nil
							(reduce tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(_visible_materialized_agg_match_col td agg_args agg_name)
										nil)))
								nil)))
						(if (nil? match_col)
							(match agg_args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_visible_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
								_ expr)
							(list (quote get_column) (car match_col) false (cadr match_col) false)))
					(cons sym args) (cons sym (map args lower_visible_materialized_aggs_single))
					expr)))
				(define resolved_stage_group (if materialized_source
					(map stage_group rewrite_materialized_source_cols_single)
					stage_group))
				(if materialized_source
					(set ags (map ags rewrite_materialized_source_cols_single)))
				/* MySQL-style grouped projections may still contain row-local expressions
				that are neither GROUP keys nor explicit aggregates. Those expressions must
				not be rewritten into fictitious keytable columns by recursively replacing
				their inner get_column markers. Instead, synthesize a stable "pick any
				non-nil" aggregate for the whole expression and fetch that aggregate like
				any other grouped value. This keeps grouped output generic and avoids
				phantom columns such as (get_column "...") on the keytable. */
				(define _group_any_reduce
					(list (quote lambda)
						(list (quote acc) (quote item))
						(list (quote if)
							(list (quote nil?) (quote item))
							(quote acc)
							(list (quote if)
								(list (quote nil?) (quote acc))
								(quote item)
								(quote acc)))))
				(define group_value_reduce _group_any_reduce)
				(define _group_value_ag (lambda (expr)
					(list expr group_value_reduce nil)))
				(define _group_value_ag_expr (lambda (expr)
					(list (quote aggregate) expr group_value_reduce nil)))
				(define _outer_stage_aliases (merge_unique (merge (map _grp_ps_tables _group_table_alias_variants))))
				(define _refs_only_outer_stage (lambda (expr)
					(begin
						(define _column_refs (extract_all_table_aliases expr))
						(define _refs (merge_unique (merge (list (extract_tblvars expr) _column_refs))))
						(define _refs_current_group_source (reduce _refs (lambda (found tv)
							(or found
								(equal? tv tblvar)
								(equal? (visible_occurrence_alias tv) (visible_occurrence_alias tblvar))))
							false))
						(define _refs_outer_stage_source (reduce _refs (lambda (found tv)
							(or found
								(has? _outer_stage_aliases tv)
								(has? _outer_stage_aliases (visible_occurrence_alias tv))))
							false))
						(and _refs_outer_stage_source (not _refs_current_group_source)))))
				(define _matches_group_expr (lambda (expr)
					(or
						(reduce stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
						(reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
						(reduce resolved_stage_group (lambda (acc group_expr)
							(or acc (equal? group_expr (rewrite_materialized_source_cols_single expr))))
							false))))
				(define _field_has_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (_field_has_agg_expr b))) false)
					false)))
				(define _expr_has_non_group_column_refs (lambda (expr)
					(reduce (extract_all_get_columns expr) (lambda (acc mc)
						(or acc
							(not (_matches_group_expr (cadr mc)))))
						false)))
				(define _field_needs_group_value_agg (lambda (expr)
					(and (not (_field_has_agg_expr expr))
						(not (_matches_group_expr expr))
						(not (and
							(or
								(not (nil? _stage_scope))
								_mixed_group_scalar_guard_join_domain)
							(_refs_only_outer_stage expr)))
						(_expr_has_non_group_column_refs expr))))
				/* Materialized sources may expose grouped pass-through fields as
				row-local LEFT-JOIN wrapper expressions. Aggregate them only after
				materializing that wrapper once as a temp source column; otherwise the
				later createcolumn/cache path can mistake the aggregate name for a
				physical source column on the underlying temp table. */
				(define group_value_local_key_expr (lambda (expr)
					(if group_value_local_materializable
						(rewrite_materialized_source_cols_single expr)
						expr)))
				(define group_value_local_col_name (lambda (expr)
					(begin
						(define logical_expr (if group_value_local_materializable
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(define ref_key (sha1 (string
							(map (extract_all_get_columns logical_expr) (lambda (mc)
								(sanitize_temp_name (string (normalize_canonical_aliases (cadr mc)))))))))
						(define head_key (match logical_expr
							'((symbol get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							'((quote get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							(cons sym _) (sanitize_temp_name (string sym))
							_ (sanitize_temp_name (string logical_expr))))
						(concat ".group_value|" head_key "|" ref_key))))
				(define group_value_local_lookup (newsession))
				(define group_value_local_head_lookup (newsession))
				(define group_value_local_head_count (newsession))
				(define group_value_local_head_key (lambda (expr)
					(begin
						(define logical_expr (if group_value_local_materializable
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(match logical_expr
							'((symbol get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							'((quote get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							(cons sym _) (sanitize_temp_name (string sym))
							_ (sanitize_temp_name (string logical_expr))))))
				(define group_value_local_expr (lambda (expr)
					(if (and group_value_local_materializable
						(group_value_refs_only_current_source? expr)
						(_field_needs_group_value_agg expr))
						(begin
							(define key_expr (group_value_local_key_expr expr))
							(define logical_expr (if group_value_local_materializable
								(lower_materialized_source_expr tbl tblvar expr)
								expr))
							(define match_col (reduce
								(merge (materialized_source_expr_keys expr)
									(materialized_source_expr_keys key_expr)
									(materialized_source_expr_keys logical_expr))
								(lambda (found key)
									(if (not (nil? found))
										found
										(coalesce (group_value_local_lookup key) nil)))
								nil))
							(define head_key (group_value_local_head_key expr))
							(define head_col (if (equal? (coalesceNil (group_value_local_head_count head_key) 0) 1)
								(coalesce (group_value_local_head_lookup head_key) nil)
								nil))
							(list (quote get_column) tblvar false (coalesce match_col head_col (group_value_local_col_name expr)) false))
						expr)))
				(define group_value_local_setup_expr (lambda (expr) (begin
					(define lowered_expr_base (group_value_local_key_expr expr))
					(define group_value_refs_helper (reduce (extract_tblvars expr) (lambda (found alias_)
						(begin
							(define alias_s (if (nil? alias_) "" (string alias_)))
							(or
								found
								(scalar_helper_root_alias? alias_s)
								(scalar_helper_nested_alias? alias_s)
								(and (> (strlen alias_s) 0) (equal? (substr alias_s 0 1) ".")))))
						false))
					(define materialized_where_guard_col ".materialized_where_condition")
					(define materialized_where_guard_available (and
						group_value_local_materializable
						group_value_refs_helper
						(schema_has_column?
							(materialized_source_physical_schema schema tbl tblvar schemas)
							materialized_where_guard_col
							false)))
					(define lowered_expr (if materialized_where_guard_available
						(list (quote if)
							(list (quote coalesceNil)
								(list (quote get_column) tblvar false materialized_where_guard_col false)
								true)
							lowered_expr_base
							nil)
						lowered_expr_base))
					(define col_name (group_value_local_col_name expr))
					(define cols (extract_columns_for_tblvar tblvar lowered_expr))
					(list (quote createcolumn) tbl_source_expr col_name "any" '(list) '(list "temp" true)
						(cons (quote list) cols)
						(list (quote lambda) (map cols (lambda (col) (symbol (concat tblvar "." col))))
							(replace_columns_from_expr lowered_expr))))))
				(define _append_group_value_exprs (lambda (acc exprs)
					(reduce (coalesceNil exprs '()) (lambda (acc2 expr)
						(if (reduce acc2 (lambda (found existing)
							(or found (equal? existing expr))) false)
							acc2
							(merge acc2 (list expr))))
						acc)))
				(define _merge_group_value_expr_lists (lambda (expr_lists)
					(reduce (coalesceNil expr_lists '()) (lambda (acc exprs)
						(_append_group_value_exprs acc exprs))
						'())))
				(define _collect_group_value_exprs (lambda (expr)
					(if (_field_needs_group_value_agg expr)
						(match expr
							(cons sym args) (_append_group_value_exprs
								(list expr)
								(_merge_group_value_expr_lists
									(map args _collect_group_value_exprs)))
							(list expr))
						(match expr
							(cons (symbol aggregate) _) '()
							(cons '(quote aggregate) _) '()
							(cons sym args) (_merge_group_value_expr_lists
								(map args _collect_group_value_exprs))
							'()))))
				(define _collect_current_group_value_exprs (lambda (expr)
					(if (_field_needs_group_value_agg expr)
						(if (group_value_refs_only_current_source? expr)
							(match expr
								(cons sym args) (_append_group_value_exprs
									(list expr)
									(_merge_group_value_expr_lists
										(map args _collect_current_group_value_exprs)))
								(list expr))
							(match expr
								(cons (symbol aggregate) _) '()
								(cons '(quote aggregate) _) '()
								(cons sym args) (_merge_group_value_expr_lists
									(map args _collect_current_group_value_exprs))
								'()))
						(match expr
							(cons (symbol aggregate) _) '()
							(cons '(quote aggregate) _) '()
							(cons sym args) (_merge_group_value_expr_lists
								(map args _collect_current_group_value_exprs))
							'()))))
				(define _field_group_value_expr_lists
					(reduce_assoc fields (lambda (acc _key expr)
						(merge acc (list (_collect_group_value_exprs expr))))
						'()))
				(define _field_group_value_exprs_flat
					(_merge_group_value_expr_lists _field_group_value_expr_lists))
				(define _order_group_value_exprs (_merge_group_value_expr_lists
					(map (coalesceNil stage_order '()) (lambda (o)
						(match o '(col _dir) (_collect_group_value_exprs col) '())))))
				(define _join_group_value_exprs (_merge_group_value_expr_lists
					(map (coalesceNil tables '()) (lambda (td)
						(match td
							'(_ _ _ _ joinexpr)
							(_collect_current_group_value_exprs (coalesceNil joinexpr true))
							'())))))
				(define group_value_candidate_exprs (_merge_group_value_expr_lists
					(merge
						_field_group_value_expr_lists
						(list
							(_collect_group_value_exprs (coalesceNil condition true))
							(_collect_group_value_exprs (coalesceNil stage_having true))
							_order_group_value_exprs
							_join_group_value_exprs))))
				(define group_value_synthetic_candidate_exprs
					(filter group_value_candidate_exprs
						group_value_refs_only_current_source?))
				(define group_value_local_fields (if group_value_local_materializable
					/* Keep each row-local grouped projection as one logical AST. `merge`
					would flatten list-valued expressions like `(if ...)` to their head
					symbol and arguments, which then materializes nonsense temp columns
					such as `(lambda () if)`. */
					group_value_synthetic_candidate_exprs
					'()))
				(if group_value_local_materializable
					(map group_value_local_fields (lambda (expr) (begin
						(define col_name (group_value_local_col_name expr))
						(define key_expr (group_value_local_key_expr expr))
						(define logical_expr (if group_value_local_materializable
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(define head_key (group_value_local_head_key expr))
						(group_value_local_head_count head_key (+ 1 (coalesceNil (group_value_local_head_count head_key) 0)))
						(if (nil? (group_value_local_head_lookup head_key))
							(group_value_local_head_lookup head_key col_name)
							nil)
						(map (merge (materialized_source_expr_keys expr)
							(materialized_source_expr_keys key_expr)
							(materialized_source_expr_keys logical_expr))
							(lambda (key) (group_value_local_lookup key col_name)))
						nil))))
				(define group_value_local_compute_plan (if (equal? group_value_local_fields '()) nil
					(list (quote time)
						(cons (quote parallel) (map group_value_local_fields group_value_local_setup_expr))
						"group-value")))
				(define synthetic_field_ags (if is_dedup '()
					(map group_value_synthetic_candidate_exprs (lambda (expr)
						(_group_value_ag (group_value_local_expr expr))))))
				(define ags (if is_dedup ags (merge_unique ags synthetic_field_ags)))
				(define lower_materialized_agg_tuple (lambda (ag) (match ag
					'(ag_expr ag_reduce ag_neutral)
					(list
						(rewrite_materialized_source_cols_single
							(rewrite_materialized_source_aggs_single ag_expr))
						ag_reduce
						ag_neutral)
					_ ag)))
				(if materialized_source
					(set ags (map ags lower_materialized_agg_tuple)))

				/* preparation */
				(define tblvar_cols (merge_unique (map resolved_stage_group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesceNil condition true)))
				(set condition (lower_visible_materialized_aggs_single condition))
				(if materialized_source
					(set condition (rewrite_materialized_source_aggs_single condition)))
				(define _flatten_and_parts (lambda (expr) (match expr
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
						(merge (map parts _flatten_and_parts))
						(list expr))
					(list expr))))
				(define _projection_materialized_where_guard? (lambda (expr)
					(match expr
						'((symbol get_column) alias_ _ col _)
						(and (equal?? alias_ tblvar) (equal?? col ".materialized_where_condition"))
						'((quote get_column) alias_ _ col _)
						(and (equal?? alias_ tblvar) (equal?? col ".materialized_where_condition"))
						'(op inner fallback)
						(and
							(or
								(equal? op (quote coalesceNil))
								(equal? op '(quote coalesceNil))
								(equal? op (symbol "coalesceNil")))
							(or (equal? fallback true) (equal? fallback (quote true)))
							(_projection_materialized_where_guard? inner))
						false)))
				(define _projection_materialized_where_guard_derived? (lambda (expr)
					(if (_projection_materialized_where_guard? expr)
						true
						(match expr
							'(op guard value fallback)
							(and
								(or
									(equal? op (quote if))
									(equal? op '(quote if))
									(equal? op (symbol "if")))
								(_projection_materialized_where_guard? guard)
								(or (nil? fallback) (equal? fallback (quote nil)))
								(_projection_materialized_where_guard_derived? value))
							false))))
				(define _has_non_guard_group_value_field
					(reduce _field_group_value_exprs_flat (lambda (found expr)
						(or found (not (_projection_materialized_where_guard_derived? expr))))
						false))
				(define _strip_projection_materialized_where_guard (lambda (expr)
					(begin
						(define kept_parts
							(filter (_flatten_and_parts expr) (lambda (part)
								(not (_projection_materialized_where_guard? part)))))
						(if (equal? kept_parts '())
							true
							(if (equal? (count kept_parts) 1)
								(car kept_parts)
								(cons (quote and) kept_parts))))))
				(if (and
					materialized_source
					(nil? _stage_scope)
					_has_non_guard_group_value_field
					(not (equal? group_value_local_fields '())))
					(set condition (_strip_projection_materialized_where_guard condition))
					nil)
				(define _condition_parts0 (_flatten_and_parts condition))
				/* Old runtime-local temp-column materialization pushed session-sensitive
				row predicates into createcolumn lambdas. That leaks the query-only
				session scope into storage compute code. Keep the predicate in the
				normal grouped scan and let the tx-aware compute/cache layer handle
				session variants instead of precomputing .runtime_pred temp columns. */
				(define runtime_local_compute_plan nil)
				/* 2-phase condition split:
				Phase 1: separate aggregate-containing AND-parts from non-aggregate parts.
				Aggregates cannot be evaluated as row filters — they need the keytable.
				Phase 2 (after keytable creation): replace aggregates with get_column refs,
				then split by table references for pushdown. */
				(define _has_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (_has_agg_expr b))) false)
					false)))
				(define _cond_parts (_flatten_and_parts condition))
				(define _cond_agg_parts (filter _cond_parts _has_agg_expr))
				(define _cond_non_agg (filter _cond_parts (lambda (p) (not (_has_agg_expr p)))))
				(define _grp_refs_src_tbl (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (acc tv) (or acc (equal?? tv tblvar))) false)))
				(define ags (if materialized_source
					(merge_unique ags (merge (extract_assoc fields (lambda (key expr) (extract_aggregates expr)))))
					ags))
				(define _grp_has_explicit_outer (lambda (expr) (match expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						true
						(reduce args (lambda (acc arg) (or acc (_grp_has_explicit_outer arg))) false))
					false)))
				(define _grp_equality_op? (lambda (op)
					(or
						(equal? op (quote equal?))
						(equal? op (quote equal??))
						(equal? op '(quote equal?))
						(equal? op '(quote equal??)))))
				/* scoped GROUPs must not keep domain/key correlation equalities in the
				aggregate compute filter. Those equalities are represented by the keytable
				group key plus LEFT JOIN ON-clause; leaving them here leaks outer refs
				into the cache formula and breaks skip-level COUNT reuse. Immediate
				correlations to the current outer row still stay in the compute path. */
				(define _grp_key_corr_part (lambda (part) (match part
					'(op left right) (if (_grp_equality_op? op)
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
							(and (not (_grp_refs_src_tbl right)) (_grp_has_explicit_outer right))
							(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
								(and (not (_grp_refs_src_tbl left)) (_grp_has_explicit_outer left))
								false))
						false)
					false)))
				(define _cond_key_corr (if (nil? _stage_scope) '()
					(filter _cond_non_agg _grp_key_corr_part)))
				(define _cond_non_agg_effective (if (nil? _stage_scope) _cond_non_agg
					(filter _cond_non_agg (lambda (p) (not (_grp_key_corr_part p))))))
				/* non-aggregate condition = keytable scan filter */
				(set condition (if (equal? 0 (count _cond_non_agg_effective)) true
					(if (equal? 1 (count _cond_non_agg_effective)) (car _cond_non_agg_effective)
						(cons (quote and) _cond_non_agg_effective))))
				/* split non-aggregate condition: parts referencing partition-staged tables go to grouped_plan */
				(define _grp_cond_split (split_condition condition _grp_ps_tables))
				(define _grp_ps_condition (match _grp_cond_split '(_ later) later))
				(set condition (match _grp_cond_split '(now _) now))
				/* Scope contract: split_condition may conservatively classify a term as
				"later" because it traversed into nested runtime/scalar-subquery code and
				saw inner aliases there. If the resulting later-part no longer contains a
				real reference to one of the current partition-stage aliases at this plan
				level, it still belongs to the row-domain filter of the current group. */
				(define grp_ps_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _)
						(if (nil? tv) ttbl tv)
						"")))))
				(define grp_ps_condition_refs_stage (lambda (expr) (match expr
					'((symbol get_column) alias_ _ _ _) (and (not (nil? alias_)) (has? grp_ps_aliases alias_))
					'((quote get_column) alias_ _ _ _) (and (not (nil? alias_)) (has? grp_ps_aliases alias_))
					(cons sym args) (if (_is_opaque_scope_sym sym)
						false
						(reduce args (lambda (found arg) (or found (grp_ps_condition_refs_stage arg))) false))
					false)))
				(if (and (not (nil? _grp_ps_condition))
					(not (equal? _grp_ps_condition true))
					(not (grp_ps_condition_refs_stage _grp_ps_condition)))
					(begin
						(set condition (combine_and_terms (list condition _grp_ps_condition)))
						(set _grp_ps_condition true))
					nil)
				(define _grp_outer_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
						"")))))
				(define _resolve_outer_group_field (lambda (expr) (match expr
					'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (has? _grp_outer_aliases alias_))
						expr
						(if (nil? alias_)
							(begin
								(define matches (filter _grp_ps_tables (lambda (td) (match td
									'(tv _ ttbl _ _)
									(begin
										(define lookup_alias (if (nil? tv) ttbl tv))
										(reduce (coalesceNil (schemas lookup_alias) '()) (lambda (found coldef)
											(or found ((if ci equal?? equal?) (coldef "Field") col))) false))
									false))))
								(if (equal? 1 (count matches))
									(match (car matches)
										'(tv _ ttbl _ _)
										(list (quote get_column) (if (nil? tv) ttbl tv) false col false)
										nil)
									nil))
							nil))
					'((quote get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (has? _grp_outer_aliases alias_))
						expr
						(if (nil? alias_)
							(begin
								(define matches (filter _grp_ps_tables (lambda (td) (match td
									'(tv _ ttbl _ _)
									(begin
										(define lookup_alias (if (nil? tv) ttbl tv))
										(reduce (coalesceNil (schemas lookup_alias) '()) (lambda (found coldef)
											(or found ((if ci equal?? equal?) (coldef "Field") col))) false))
									false))))
								(if (equal? 1 (count matches))
									(match (car matches)
										'(tv _ ttbl _ _)
										(list (quote get_column) (if (nil? tv) ttbl tv) false col false)
										nil)
									nil))
							nil))
					nil)))
				(define collect_condition (if (and is_dedup materialized_source)
					(begin
						(define _dedup_collect_parts (filter (_flatten_and_parts condition) (lambda (part)
							(and (equal? (has_only_tblvar_refs part tblvar) true)
								(not (_has_agg_expr part))))))
						(if (equal? 0 (count _dedup_collect_parts)) true
							(if (equal? 1 (count _dedup_collect_parts)) (car _dedup_collect_parts)
								(cons (quote and) _dedup_collect_parts))))
					condition))
				(define _stage_join_outer_expr (lambda (expr) (match expr
					'((symbol get_column) _ _ _ _) expr
					'((quote get_column) _ _ _ _) expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						expr
						(cons sym (map args _stage_join_outer_expr)))
					_ (begin
						(define _parts (split (string expr) "."))
						(match _parts
							(list _tbl _col) (list (quote get_column) _tbl false _col false)
							_ expr)))))
				(define _dedup_join_term (lambda (part) (match part
					'(op left right) (if (_grp_equality_op? op)
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
							(if (_grp_refs_src_tbl right) nil
								(list (quote equal??) (replace_col_for_dedup left) (_stage_join_outer_expr right)))
							(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
								(if (_grp_refs_src_tbl left) nil
									(list (quote equal??) (_stage_join_outer_expr left) (replace_col_for_dedup right)))
								nil))
						nil)
					nil)))
				(set filtercols (merge_unique (list
					(extract_columns_for_tblvar tblvar collect_condition)
					(extract_outer_columns_for_tblvar tblvar collect_condition))))
				(define session_sensitive_group_domain (expr_uses_session_state collect_condition))
				(define group_cache_identity (stage_cache_query stage))
				(define keytable_condition_suffix (if (nil? group_cache_identity)
					collect_condition
					(list (quote cache-key) collect_condition group_cache_identity)))
				/* Include condition suffix for keytable naming: dedup stages without
				session-sensitive domains, scoped stages with a local condition, and
				session-sensitive group domains. make_keytable folds runtime session
				values into that suffix, so helper tables do not leak one session's row
				domain into another session. */
				(define kt_result (make_keytable schema base_tbl resolved_stage_group tblvar
					(if (or (and is_dedup (not session_sensitive_group_domain))
						(and (or _scoped_stage session_sensitive_group_domain)
							(not (or (nil? collect_condition) (equal? collect_condition true))))
						(not (nil? group_cache_identity)))
						keytable_condition_suffix nil)))
				(set grouptbl (car kt_result))
				(define kt_schema_def (nth kt_result 1))
				(define keytable_init (nth kt_result 2))
				(define fk_pk_col (nth kt_result 3))
				(define is_fk_reuse (not (nil? fk_pk_col)))
				/* register keytable schema so build_scan can resolve get_columns */
				(if (not (nil? kt_schema_def))
					(set schemas (set_assoc schemas grouptbl kt_schema_def)))

				/* make_collect: builds collect plan with optional WHERE filter
				with_filter=true: apply WHERE condition (for DEDUP)
				with_filter=false: collect ALL group keys (for NORMAL) */
				(define make_collect (lambda (with_filter)
					'('time '('begin
						/* If grouping is global (group='(1)), avoid base scan and insert one key row */
						(if (equal? resolved_stage_group '(1))
							'('insert '('table schema grouptbl) '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
							(begin
								/* key columns */
								(set keycols (merge_unique (map resolved_stage_group (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
								(if tagged_scan
									(begin
										(define ordercols (extract_scan_order_cols_for_tblvar tbl_scan_order tblvar))
										(define dirs (extract_scan_order_dirs_for_tblvar tbl_scan_order tblvar))
										(define collect_filtercols (if with_filter (merge_unique filtercols keycols) keycols))
										(scan_wrapper 'scan_order schema base_tbl
											(cons list collect_filtercols)
											(if with_filter
												'((quote lambda) (map collect_filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr collect_condition)))
												'((quote lambda) (map collect_filtercols (lambda(col) (symbol (concat tblvar "." col)))) true))
											(cons list ordercols)
											(cons list dirs)
											tbl_scan_partcols
											(coalesceNil tbl_scan_offset 0)
											(coalesceNil tbl_scan_limit -1)
											(cons list keycols)
											(list (quote lambda)
												(map keycols (lambda (col) (symbol (concat tblvar "." col))))
												(runtime_list_ast (map resolved_stage_group (lambda (expr) (replace_columns_from_expr expr)))))
											'((quote lambda) '('acc 'rowvals) '('set_assoc 'acc 'rowvals true))
											'(list)
											'((quote lambda) '('acc 'sharddict)
												'('insert
													'('table schema grouptbl)
													(cons 'list (map resolved_stage_group expr_name))
													'('assoc_keys_as_dataset_rows 'sharddict (count resolved_stage_group))
													'(list) '('lambda '() true) true)
											)
											isOuter))
									(scan_wrapper 'scan schema base_tbl
										(if with_filter (cons list filtercols) '(list))
										(if with_filter
											'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr collect_condition)))
											'((quote lambda) '() true))
										(cons list keycols)
										(list (quote lambda)
											(map keycols (lambda (col) (symbol (concat tblvar "." col))))
											(runtime_list_ast (map resolved_stage_group (lambda (expr) (replace_columns_from_expr expr))))) /* build records '(k1 k2 ...) */
										'((quote lambda) '('acc 'rowvals) '('set_assoc 'acc 'rowvals true)) /* add keys to assoc; each key is a dataset -> unique filtering */
										'(list) /* empty dict */
										'((quote lambda) '('acc 'sharddict)
											'('insert
												'('table schema grouptbl)
												(cons 'list (map resolved_stage_group expr_name))
												'('assoc_keys_as_dataset_rows 'sharddict (count resolved_stage_group)) /* turn keys from assoc into dataset rows */
												'(list) '('lambda '() true) true)
										)
										isOuter))
							)
						)
					) "collect")))

				(if is_dedup (begin
					/* DEDUP-ONLY stage: no aggregate computation, just collect unique keys and pass through to next stage */
					(define replace_col_for_dedup (make_col_replacer grouptbl collect_condition true expr_name tblvar agg_col_name))
					(define dedup_schema_def (map resolved_stage_group (lambda (expr)
						(list "Field" (if is_fk_reuse fk_pk_col (expr_name expr)) "Type" "any"))))
					(planned_materialized_fields grouptbl dedup_schema_def)
					/* transform rest_groups to reference grouptbl columns instead of source table columns;
					first resolve nil -> tblvar via replace_find_column, then map tblvar -> grouptbl */
					(define _dedup_refs_current (lambda (e)
						(reduce (extract_tblvars e) (lambda (acc tv) (or acc (equal?? tv tblvar))) false)))
					(define _dedup_resolve (lambda (e)
						(if (_dedup_refs_current e)
							(replace_col_for_dedup (replace_find_column e))
							(coalesce (_resolve_outer_group_field e) e))))
					(define _dedup_kt_is_outer (not (nil? _stage_scope)))
					(define _dedup_kt_je (if _dedup_kt_is_outer
						(begin
							(define _dedup_terms (filter (map _cond_key_corr _dedup_join_term) (lambda (x) (not (nil? x)))))
							(if (equal? _dedup_terms '()) nil
								(if (equal? 1 (count _dedup_terms)) (car _dedup_terms)
									(cons (quote and) _dedup_terms))))
						nil))
					(define transformed_rest_groups (map rest_groups (lambda (s)
						(stage_rebuild_with_meta s (make_group_stage
							(map (stage_group_cols s) _dedup_resolve)
							(_dedup_resolve (stage_having_expr s))
							(map (coalesce (stage_order_list s) '()) (lambda (o) (match o '(col dir) (list (_dedup_resolve col) dir))))
							(stage_limit_val s)
							(stage_offset_val s)
							(stage_partition_aliases s)
							(stage_init_code s))
							_dedup_resolve
							(lambda (a) a))
					)))
					(define dedup_output_fields
						(map_assoc fields (lambda (k v) (_dedup_resolve v))))
					(define dedup_visible_schema
						(extract_assoc dedup_output_fields (lambda (k v)
							(list "Field" k "Type" "any" "Expr" v))))
					(define grouped_plan (build_queryplan schema
						(if _dedup_kt_is_outer
							(merge _grp_ps_tables (list (list grouptbl schema grouptbl true _dedup_kt_je)))
							(list (list grouptbl schema grouptbl false nil)))
						dedup_output_fields
						nil /* condition already applied in collect */
						transformed_rest_groups
						(set_assoc schemas grouptbl dedup_visible_schema)
						replace_find_column
						update_target))
					/* init_code guard: collect dedup keys on first keytable creation */
					(cons 'begin (merge
						(list (list 'if keytable_init (make_collect true) nil))
						(if (nil? runtime_local_compute_plan) '() (list runtime_local_compute_plan))
						(list grouped_plan)))
				) (begin
						/* NORMAL group stage: extract aggregates, compute, and continue.
						replace_agg_with_fetch rewrites (aggregate expr + 0) -> (get_column grouptbl "expr|cond")
						so ORDER BY SUM(amount) becomes ORDER BY on a keytable column. */
						(define agg_col_name (make_aggregate_cache_col_name expr_name condition group_runtime_suffix))
						(define replace_agg_with_fetch (make_col_replacer grouptbl condition false expr_name tblvar agg_col_name))
						(define replace_group_key_or_fetch (lambda (expr) (if
							(reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
							'('get_column grouptbl false (if is_fk_reuse fk_pk_col (expr_name expr)) false)
							(replace_agg_with_fetch expr)
						)))
						/* scoped GROUP stages from unnesting must not eagerly rewrite later
						outer aggregates like COUNT(*) in the SELECT list. Those belong to
						subsequent global group stages and carry no refs to the current
						scoped source table. */
						(define replace_deferred_mixed_aggregate_item (lambda (agg_expr) (match agg_expr
							'((symbol get_column) _ _ _ _)
							(if (group_value_refs_only_current_source? agg_expr)
								(replace_group_field_expr agg_expr)
								agg_expr)
							'((quote get_column) _ _ _ _)
							(if (group_value_refs_only_current_source? agg_expr)
								(replace_group_field_expr agg_expr)
								agg_expr)
							(cons sym args)
							(cons sym (map args replace_deferred_mixed_aggregate_item))
							agg_expr)))
						(define replace_deferred_aggregate_item (lambda (agg_expr)
							(if (equal? (extract_tblvars agg_expr) '())
								agg_expr
								(if (group_value_refs_only_current_source? agg_expr)
									(replace_group_field_expr agg_expr)
									(replace_deferred_mixed_aggregate_item agg_expr)))))
						(define replace_group_field_expr (lambda (expr)
							/* Design contract: scoped GROUP stages may only lower current-stage
							keys/aggregates here. Outer-pass-through expressions stay logical
							and are resolved later by the recursive grouped_plan build, not by
							synthesizing ad-hoc keytable aggregates in this stage. */
							(if (and (not (nil? _stage_scope)) (_refs_only_outer_stage expr))
								expr
								(match expr
									'((symbol get_column) _ _ _ _) (if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(replace_group_key_or_fetch (rewrite_materialized_source_cols_single expr)))
									'((quote get_column) _ _ _ _) (if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(replace_group_key_or_fetch (rewrite_materialized_source_cols_single expr)))
									(cons (symbol aggregate) agg_rest)
									(if (or (and (not (nil? _stage_scope)) _has_later_group_stage (equal? (extract_tblvars expr) '()) (not (equal? agg_rest aggregate_count_descriptor)))
										(and (not materialized_source) (_field_agg_has_nested_agg agg_rest) (equal? (extract_tblvars expr) '()))
										(and (not materialized_source) (not (group_value_refs_only_current_source? expr))))
										(match agg_rest
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (replace_deferred_aggregate_item agg_expr) agg_reduce agg_neutral)
											_ expr)
										(replace_group_key_or_fetch expr))
									(cons '(quote aggregate) agg_rest)
									(if (or (and (not (nil? _stage_scope)) _has_later_group_stage (equal? (extract_tblvars expr) '()) (not (equal? agg_rest aggregate_count_descriptor)))
										(and (not materialized_source) (_field_agg_has_nested_agg agg_rest) (equal? (extract_tblvars expr) '()))
										(and (not materialized_source) (not (group_value_refs_only_current_source? expr))))
										(match agg_rest
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (replace_deferred_aggregate_item agg_expr) agg_reduce agg_neutral)
											_ expr)
										(replace_group_key_or_fetch expr))
									(cons sym args)
									(if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(if (_matches_group_expr expr)
											(replace_group_key_or_fetch expr)
											(if (_is_opaque_scope_sym sym)
												(replace_group_key_or_fetch expr)
												(cons sym (map args replace_group_field_expr)))))
									(replace_group_key_or_fetch expr)
						))))
						/* Materialized derived GROUP outputs are a semantic boundary: a
						SELECT-list aggregate that has no refs to the current scoped helper
						must remain a logical aggregate for the later outer GROUP pass. This is
						intentionally output-only; HAVING/EXISTS predicates still need the local
						helper COUNT/SUM values from this stage. */
						(define replace_group_output_expr (lambda (expr)
							(if (and (not (nil? _stage_scope)) materialized_output_boundary _has_later_group_stage (equal? (extract_tblvars expr) '()))
								(match expr
									(cons (symbol aggregate) agg_rest)
									(match agg_rest
										'(agg_expr agg_reduce agg_neutral)
										(list (quote aggregate) (replace_deferred_aggregate_item agg_expr) agg_reduce agg_neutral)
										_ expr)
									(cons '(quote aggregate) agg_rest)
									(match agg_rest
										'(agg_expr agg_reduce agg_neutral)
										(list (quote aggregate) (replace_deferred_aggregate_item agg_expr) agg_reduce agg_neutral)
										_ expr)
									(cons sym args) (cons sym (map args replace_group_output_expr))
									(replace_group_field_expr expr))
								(match expr
									(cons (symbol aggregate) _)
									(replace_group_field_expr expr)
									(cons '(quote aggregate) _)
									(replace_group_field_expr expr)
									(cons sym args) (if (_field_needs_group_value_agg expr)
										(replace_group_field_expr expr)
										(if (_matches_group_expr expr)
											(replace_group_field_expr expr)
											(if (_is_opaque_scope_sym sym)
												(replace_group_field_expr expr)
												(cons sym (map args replace_group_output_expr)))))
									(replace_group_field_expr expr)))))
						/* normalize outer-side join expressions into column AST so scan
						planning can request the needed outer columns even if they are not
						part of the current projection/order list. */
						(define _grp_join_outer_expr (lambda (expr) (match expr
							'((symbol get_column) _ _ _ _) expr
							'((quote get_column) _ _ _ _) expr
							(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
								expr
								(cons sym (map args _grp_join_outer_expr)))
							_ (begin
								(define _parts (split (string expr) "."))
								(match _parts
									(list _tbl _col) (list (quote get_column) _tbl false _col false)
									_ expr)))))
						(define _grp_join_term (lambda (part) (match part
							'(op left right) (if (_grp_equality_op? op)
								(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
									(if (_grp_refs_src_tbl right) nil
										(list (quote equal??) (replace_group_key_or_fetch left) (_grp_join_outer_expr right)))
									(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
										(if (_grp_refs_src_tbl left) nil
											(list (quote equal??) (_grp_join_outer_expr left) (replace_group_key_or_fetch right)))
										nil))
								nil)
							nil)))
						/* scoped GROUPs join the keytable back via LEFT JOIN. Any correlation
						equality already emitted into that ON-clause must not survive as an
						additional post-group filter, otherwise NOT IN / NOT EXISTS lose the
						empty-match row (NULL keytable side fails the redundant equality). */
						/* Design contract: every scoped GROUP stage re-attaches its keytable to
						the preserved outer row stream, even for global `(1)` groups, but only
						when such a real outer stream actually exists. The synthetic no-FROM
						DUAL `.(1)` row is just a planner helper; treating it as a preserved
						outer stream here would force unnecessary LEFT JOIN semantics and can
						drop top-level NOT IN / EXISTS filters. */
						(define _real_outer_ps_tables (filter _grp_ps_tables (lambda (td) (match td
							'(_ _ ttbl _ _) (not (equal? ttbl ".(1)"))
							true))))
						(define _grp_positive_match_condition? (lambda (expr) (match expr
							'(op left right) (and
								(or
									(equal? op (quote >))
									(equal? op '(quote >))
									(equal? op (symbol ">"))
									(equal? (string op) ">"))
								(_has_agg_expr left)
								(or (equal? right 0) (equal? right 0.0)))
							(cons sym args) (reduce args (lambda (found arg)
								(or found (_grp_positive_match_condition? arg))) false)
							false)))
						(define _kt_positive_match_filter
							(reduce _cond_agg_parts (lambda (found part)
								(or found (_grp_positive_match_condition? part))) false))
						(define _kt_is_outer (and
							(or
								(not (nil? _stage_scope))
								_mixed_group_scalar_guard_join_domain)
							(or (not _kt_positive_match_filter) (not (nil? update_target)))
							(not (equal? _real_outer_ps_tables '()))))
						(define _kt_outer_source_terms (if _kt_is_outer
							(filter (map (coalesceNil (stage_outer_sources stage) '()) (lambda (src)
								(match src
									'(outer_tv outer_col inner_expr)
									(if (stage_outer_source_expr_tuple? src)
										(begin
											(define outer_expr (nth src 1))
											(define inner_expr (nth src 2))
											(if (and
												(reduce (extract_tblvars outer_expr) (lambda (found outer_tv)
													(or found (has? grp_ps_aliases outer_tv))) false)
												(reduce resolved_stage_group (lambda (found group_expr)
													(or found (equal? group_expr inner_expr)))
													false))
												(list (quote equal??)
													(replace_group_key_or_fetch inner_expr)
													outer_expr)
												nil))
										(if (and
											(has? grp_ps_aliases outer_tv)
											(reduce resolved_stage_group (lambda (found group_expr)
												(or found (equal? group_expr inner_expr)))
												false))
											(list (quote equal??)
												(replace_group_key_or_fetch inner_expr)
												(list (quote get_column) outer_tv false outer_col false))
											nil))
									_ nil)))
								(lambda (x) (not (nil? x))))
							'()))
						(define _kt_condition_parts (merge
							_cond_non_agg
							(if (or (nil? _grp_ps_condition) (equal? _grp_ps_condition true))
								'()
								(_flatten_and_parts _grp_ps_condition))))
						(define _kt_terms (if _kt_is_outer
							(merge_unique (list
								(filter (map _kt_condition_parts _grp_join_term) (lambda (x) (not (nil? x))))
								_kt_outer_source_terms))
							'()))
						(define detached_materialized_scoped_group_output (and
							_kt_is_outer
							materialized_output_boundary
							(not _has_later_group_stage)
							(equal? fields '())
							(equal? _kt_terms '())))
						(define _strip_kt_term (lambda (part)
							(not (reduce _kt_terms (lambda (acc kt_part) (or acc (equal? part kt_part))) false))))
						(define _flatten_gp_part (lambda (expr) (match expr
							(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
								(merge (map parts _flatten_gp_part))
								(list expr))
							(list expr))))
						(define _positive_match_key_parts (if _kt_positive_match_filter
							(map resolved_stage_group (lambda (group_expr)
								(list (quote not) (list (quote nil?) (replace_group_key_or_fetch group_expr)))))
							'()))

						(define grouped_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (replace_group_key_or_fetch col) dir))))))
						(define scoped_no_order_scalar_limit_redundant
							(and _scoped_stage
								(equal? (coalesceNil grouped_order '()) '())
								(not (nil? stage_limit))
								(<= stage_limit 1)
								(or (nil? stage_offset) (equal? stage_offset 0))
								(not (equal? (coalesceNil resolved_stage_group '()) '()))))
						(define next_groups (merge
							(if (and (coalesce grouped_order stage_limit stage_offset)
								(not scoped_no_order_scalar_limit_redundant))
								(list (make_group_stage nil nil grouped_order stage_limit stage_offset nil nil))
								'())
							(if _needs_synthetic_outer_group (list (make_group_stage '(1) nil nil nil nil nil nil)) '())
							rest_groups
						))
						/* FK reuse: extract child FK column name */
						(define fk_child_col (if is_fk_reuse
							(match (car resolved_stage_group) '('get_column _ false scol false) scol)
							nil))
						/* COUNT payload is needed whenever later logic fetches an aggregate-backed
						existence/anti result from this stage. Global `(1)` groups still need it
						if the deferred post-group condition contains aggregate terms. Only the
						COUNT>0 empty-group filter itself stays restricted to true global groups. */
						(define _marker_in_fields (reduce_assoc fields (lambda (acc _k v) (or acc (_has_agg_expr v))) false))
						(define _marker_in_having (if (nil? stage_having) false (_has_agg_expr stage_having)))
						(define _marker_in_order (reduce (coalesce stage_order '()) (lambda (acc o) (or acc (match o '(col _dir) (_has_agg_expr col) false))) false))
						(define _condition_unrestricted (or (nil? condition) (equal? condition true)))
						/* SQL GROUP BY semantics: unscoped non-global groups only exist for row
						keys that survive the pre-group row domain. Keep the logical aggregate
						sentinels until build_scan, but enforce this domain invariant here via
						COUNT(*) > 0 instead of materializing helper-side phantom groups.
						Global helper stages still use the narrower suppression rule so
						user-visible SELECT COUNT(*) FROM ... on empty input keeps its single
						neutral row. Scoped GROUP stages must not suppress empty matches here,
						because NOT EXISTS / NOT IN rely on the later LEFT JOIN + coalesceNil. */
						(define filter_empty_groups (and
							(nil? _stage_scope)
							(or
								(and
									(not (equal? resolved_stage_group '(1)))
									(or
										is_fk_reuse
										(not _condition_unrestricted)))
								(and
									(equal? resolved_stage_group '(1))
									(not _marker_in_fields)
									(not _marker_in_having)
									(not _marker_in_order)))))
						(define needs_count (or
							filter_empty_groups
							(not (equal? _cond_agg_parts '()))))
						(define ags (if needs_count (merge_unique ags (list aggregate_count_descriptor)) ags))
						(define count_col_name (if needs_count (agg_col_name aggregate_count_descriptor) nil))
						(define keytable_schema_def (merge
							(map resolved_stage_group (lambda (expr)
								(list "Field" (if is_fk_reuse fk_pk_col (expr_name expr)) "Type" "any")))
							(map ags (lambda (ag)
								(list "Field" (agg_col_name ag) "Type" "any")))))
						(planned_materialized_fields grouptbl keytable_schema_def)
						/* AND count>0 into HAVING so empty/non-matching groups are excluded */
						(define effective_having (if (and needs_count filter_empty_groups)
							(begin
								(define count_check '('> '('get_column grouptbl false count_col_name false) 0))
								(define replaced_having (replace_group_field_expr stage_having))
								(if (or (nil? replaced_having) (equal? replaced_having true))
									count_check
									(list 'and replaced_having count_check)))
							(replace_group_field_expr stage_having)))

						/* Phase 2: replace aggregates in the separated agg-condition parts,
						then combine everything: HAVING + replaced agg-parts + ps-table conditions */
						(define _replaced_agg_parts (map _cond_agg_parts replace_group_field_expr))
						(define _expr_contains_group_key? (lambda (expr)
							(or
								(reduce resolved_stage_group (lambda (found group_expr)
									(or found (equal? expr group_expr))) false)
								(match expr
									(cons _ args) (reduce args (lambda (found arg)
										(or found (_expr_contains_group_key? arg))) false)
									false))))
						(define _post_group_non_agg_parts (if (equal? resolved_stage_group '(1))
							'()
							(filter
								(map
									(filter _cond_non_agg_effective _expr_contains_group_key?)
									replace_group_field_expr)
								(lambda (part)
									(and
										(not (nil? part))
										(not (equal? part true))
										(not (_grp_refs_src_tbl part)))))))
						/* partition-staged table predicates stay global filters.
						The keytable LEFT JOIN must only use correlations against group/domain
						keys, otherwise unrelated outer filters get attached to the wrong side. */
						(define _gp_parts (filter (merge (map (merge
							(if (or (nil? effective_having) (equal? effective_having true)) '() (list effective_having))
							_replaced_agg_parts
							_post_group_non_agg_parts
							_positive_match_key_parts
							(if (equal? _grp_ps_condition true) '() (list (replace_group_key_or_fetch _grp_ps_condition))))
							_flatten_gp_part))
							(lambda (x) (and (not (nil? x)) (not (equal? x true)) (_strip_kt_term x)))))
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
						/* keytable join condition: only keep equalities that bind a group/domain
						key to an outer expression. Filters on outer/prejoin tables stay in the
						global condition; they are not ON-conditions of the keytable join. */
						(define _kt_join_only_terms (filter _kt_terms (lambda (term)
							(match term
								'(op _ _) (_grp_equality_op? op)
								false))))
						(define _kt_je (if _kt_is_outer
							(begin
								(if (equal? _kt_join_only_terms '()) nil
									(if (equal? 1 (count _kt_join_only_terms)) (car _kt_join_only_terms)
										(cons (quote and) _kt_join_only_terms))))
							nil))
						(define grouped_output_fields
							(map_assoc fields (lambda (k v) (replace_group_output_expr v))))
						(define grouped_visible_schema
							(extract_assoc grouped_output_fields (lambda (k v)
								(list "Field" k "Type" "any" "Expr" v))))
						(define grouped_plan (if detached_materialized_scoped_group_output
							true
							(build_queryplan schema
								(if _kt_is_outer
									(merge _grp_ps_tables (list (list grouptbl schema grouptbl true _kt_je)))
									(list (list grouptbl schema grouptbl false nil)))
								grouped_output_fields
								_gp_condition
								(merge next_groups _remaining_pstages)
								(set_assoc schemas grouptbl grouped_visible_schema)
								replace_find_column
								update_target)))
						/* Software contract for filtered grouped aggregates:
						- keep a single canonical aggregate temp column per helper table
						- drive eager materialization via filtercols/filter on the canonical
						COUNT helper column, not via ad-hoc one-shot aggregate scans
						- this keeps the aggregate cache persistent and incrementally
						maintainable while still skipping empty groups */
						/* createcolumn options: filter by COUNT column so only groups with rows are computed.
						Session-sensitive helper columns also get an explicit neutral filter marker so
						StorageComputeProxy can keep per-session variants even when the computor has been
						lowered through nested scans. */
						(define session_variant_filter_body (lambda (base_body)
							(if (equal? group_runtime_terms '())
								base_body
								(cons (quote begin)
									(merge
										(map group_runtime_terms (lambda (term) (list (quote quote) term)))
										(list base_body))))))
						(define session_variant_filter (lambda (cols body)
							(list (quote lambda)
								(map cols symbol)
								(session_variant_filter_body body))))
						(define session_marker_options (lambda (base_options cols body)
							(if (equal? group_runtime_terms '())
								base_options
								(cons 'list (merge '("temp" true)
									(list "filtercols" (cons 'list cols)
										"filter" (session_variant_filter cols body)))))))
						(define count_createcol_options
							(session_marker_options '(list "temp" true) '() true))
						(define stage_order_group_keys_only (reduce (coalesceNil stage_order '()) (lambda (acc order_item)
							(and acc (match order_item
								'(order_expr _dir)
								(reduce resolved_stage_group (lambda (found group_expr)
									(or found (equal? group_expr order_expr))) false)
								false)))
							true))
						(define lazy_group_aggregate_compute (and
							(not needs_count)
							(not (nil? stage_limit))
							(or
								(equal? (coalesceNil stage_order '()) '())
								stage_order_group_keys_only)
							(nil? effective_having)))
						(define collect_single_aggregate_compute (and
							lazy_group_aggregate_compute
							(not tagged_scan)
							(not materialized_source)
							(equal? (count ags) 1)
							(or (nil? collect_condition) (equal? collect_condition true))
							(equal? group_runtime_terms '())))
						(define aggregate_createcol_base_options
							(if lazy_group_aggregate_compute
								'(list "temp" true "lazy" true)
								'(list "temp" true)))
						(define createcol_options
							(if (and needs_count filter_empty_groups)
								(cons 'list (merge '("temp" true)
									(list "filtercols" (list 'list count_col_name)
										"filter" (session_variant_filter (list count_col_name) (list (quote >) (symbol count_col_name) 0)))))
								(session_marker_options aggregate_createcol_base_options '() true)))

						(define match_runtime_materialized_agg_col (lambda (target_col agg_name)
							(if materialized_source
								(begin
									(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
									(coalesce
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define field_name (coldef "Field"))
													(if (equal? field_name target_col) field_name nil))))
											nil)
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define field_name (coldef "Field"))
													(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
														(equal? (substr field_name 0 (strlen agg_name)) agg_name)
														(equal? (substr field_name (strlen agg_name) 1) "|"))
														field_name
														nil))))
											nil)
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define source_expr (coalesceNil (coldef "Expr") nil))
													(match source_expr
														'((symbol get_column) _ _ source_col _)
														(if (or (equal? source_col target_col)
															(and (>= (strlen source_col) (+ (strlen agg_name) 1))
																(equal? (substr source_col 0 (strlen agg_name)) agg_name)
																(equal? (substr source_col (strlen agg_name) 1) "|")))
															(coldef "Field")
															nil)
														'((quote get_column) _ _ source_col _)
														(if (or (equal? source_col target_col)
															(and (>= (strlen source_col) (+ (strlen agg_name) 1))
																(equal? (substr source_col 0 (strlen agg_name)) agg_name)
																(equal? (substr source_col (strlen agg_name) 1) "|")))
															(coldef "Field")
															nil)
														nil))))
											nil)))
								nil)))
						(define lower_runtime_materialized_aggs_single (lambda (expr) (match expr
							(cons (symbol aggregate) agg_args) (begin
								(define target_col (agg_col_name agg_args))
								(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
								(define visible_expr (if materialized_source
									(lower_visible_materialized_aggs_single expr)
									expr))
								(define match_col (match_runtime_materialized_agg_col target_col agg_name))
								(if (not (equal? visible_expr expr))
									visible_expr
									(if (nil? match_col)
										(match agg_args
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (lower_runtime_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
											_ expr)
										(list (quote get_column) tblvar false match_col false))))
							(cons '(quote aggregate) agg_args) (begin
								(define target_col (agg_col_name agg_args))
								(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
								(define visible_expr (if materialized_source
									(lower_visible_materialized_aggs_single expr)
									expr))
								(define match_col (match_runtime_materialized_agg_col target_col agg_name))
								(if (not (equal? visible_expr expr))
									visible_expr
									(if (nil? match_col)
										(match agg_args
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (lower_runtime_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
											_ expr)
										(list (quote get_column) tblvar false match_col false))))
							(cons sym args) (cons sym (map args lower_runtime_materialized_aggs_single))
							expr)))
						(define make_collect_single_aggregate (lambda (ag) (match ag '(expr reduce neutral) (begin
							(define runtime_expr
								(rewrite_materialized_source_cols_single
									(rewrite_materialized_source_aggs_single
										(lower_runtime_materialized_aggs_single expr))))
							(define keycols (merge_unique (map resolved_stage_group (lambda (gexpr)
								(extract_columns_for_tblvar tblvar gexpr)))))
							(define aggcols (merge_unique (list
								(extract_columns_for_tblvar tblvar runtime_expr)
								(extract_outer_columns_for_tblvar tblvar runtime_expr))))
							(define mapcols (merge_unique (list keycols aggcols)))
							(define key_expr (if (equal? (count resolved_stage_group) 1)
								(replace_columns_from_expr (car resolved_stage_group))
								(cons 'list (map resolved_stage_group (lambda (gexpr)
									(replace_columns_from_expr gexpr))))))
							(define value_expr (replace_columns_from_expr runtime_expr))
							(list 'time
								(list 'begin
									(list 'createcolumn
										(list 'table schema grouptbl)
										(agg_col_name ag)
										"any"
										'(list)
										'(list "temp" true))
									(scan_wrapper 'scan schema base_tbl
										'(list)
										'((quote lambda) '() true)
										(cons list mapcols)
										(list (quote lambda)
											(map mapcols (lambda (col) (symbol (concat tblvar "." col))))
											(list 'cons key_expr (list 'cons value_expr (list 'list))))
										(list (quote lambda) (list (quote acc) (quote rowvals))
											(list 'set_assoc
												'acc
												(list 'nth 'rowvals 0)
												(list 'nth 'rowvals 1)
												(list (quote lambda) (list (quote old) (quote new))
													(list reduce 'old 'new))))
										'(list)
										(list (quote lambda) (list (quote acc) (quote sharddict))
											(list 'insert
												(list 'table schema grouptbl)
												(cons 'list (merge (map resolved_stage_group expr_name) (list (agg_col_name ag))))
												(list 'assoc_items_as_dataset_rows 'sharddict (count resolved_stage_group))
												'(list)
												'((quote lambda) '() true)
												true))
										isOuter))
								"collect")))))
						(define agg_plans (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
							(define runtime_expr
								(rewrite_materialized_source_cols_single
									(rewrite_materialized_source_aggs_single
										(lower_runtime_materialized_aggs_single expr))))
							(set cols (merge_unique (list
								(extract_columns_for_tblvar tblvar runtime_expr)
								(extract_outer_columns_for_tblvar tblvar runtime_expr)
							)))
							/* COUNT column itself must not filter by itself (circular); it may still carry a session marker. */
							(define this_options (if (and needs_count (equal? (agg_col_name ag) count_col_name)) count_createcol_options createcol_options))
							(define create_agg_col_plan '('createcolumn '('table schema grouptbl) (agg_col_name ag) "any" '(list) this_options
								(cons list (map resolved_stage_group (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
								'((quote lambda) (map resolved_stage_group (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
									(if tagged_scan
										(begin
											(define ordercols (extract_scan_order_cols_for_tblvar tbl_scan_order tblvar))
											(define dirs (extract_scan_order_dirs_for_tblvar tbl_scan_order tblvar))
											(define agg_filtercols (merge_unique (merge tblvar_cols filtercols cols)))
											(scan_wrapper 'scan_order schema base_tbl
												(cons list agg_filtercols)
												/* check group equality AND WHERE-condition */
												'((quote lambda) (map agg_filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons (quote and) (cons (replace_columns_from_expr condition) (map resolved_stage_group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
												(cons list ordercols)
												(cons list dirs)
												tbl_scan_partcols
												(coalesceNil tbl_scan_offset 0)
												(coalesceNil tbl_scan_limit -1)
												(cons list cols)
												'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr runtime_expr))
												reduce
												neutral
												nil
												false))
										(scan_wrapper 'scan schema base_tbl
											(cons list (merge tblvar_cols filtercols))
											/* check group equality AND WHERE-condition */
											'((quote lambda) (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons (quote and) (cons (replace_columns_from_expr condition) (map resolved_stage_group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
											(cons list cols)
											'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr runtime_expr))
											reduce
											neutral
											nil
											false /* never isOuter in createcolumn: COUNT=0 for empty matches, not COUNT=1 */
									))
							)))
							create_agg_col_plan
						)))))
						/* COUNT is a dependency for filtered aggregates: non-count keytable
						columns may filter on COUNT>0 and therefore must not race the COUNT
						createcolumn itself. Keep COUNT synchronous, then parallelize the
						remaining independent aggregate columns. */
						(define agg_plan_indices (produceN (count ags)))
						(define count_plan (if needs_count
							(reduce agg_plan_indices (lambda (found i)
								(if (not (nil? found))
									found
									(if (equal? (agg_col_name (nth ags i)) count_col_name)
										(nth agg_plans i)
										nil)))
								nil)
							nil))
						(define non_count_agg_plans (reduce agg_plan_indices (lambda (acc i)
							(if (and needs_count (equal? (agg_col_name (nth ags i)) count_col_name))
								acc
								(merge acc (list (nth agg_plans i)))))
							'()))
						(define compute_plan
							(if collect_single_aggregate_compute
								nil
								(if (nil? count_plan)
									'('time (cons 'parallel agg_plans) "compute")
									(if (equal? non_count_agg_plans '())
										(list 'time count_plan "compute")
										(list 'begin
											(list 'time count_plan "compute-count")
											(list 'time (cons 'parallel non_count_agg_plans) "compute"))))))

						/* invalidation is handled by registerComputeTriggers in ComputeColumn:
						DML triggers on the base table invalidate computed columns automatically.
						No forced invalidation needed here — the createcolumn/ComputeColumn path
						skips recompute when the proxy is still valid (no DML since last compute). */
						(define invalidation_plan nil)

						/* build key column pairs for keytable cleanup triggers: ((base_col kt_col) ...) */
						(define key_pairs (map resolved_stage_group (lambda (expr)
							(match expr
								'((symbol get_column) _ _ col _) (list col (expr_name expr))
								'((quote get_column) _ _ col _) (list col (expr_name expr))
								(list (expr_name expr) (expr_name expr))
						))))
						(define cleanup_plan (if (or is_fk_reuse (equal? resolved_stage_group '(1)) (not (string? tbl))) nil
							(list 'register_keytable_cleanup tbl_source_expr (list 'table schema grouptbl) tblvar
								(cons 'list (map key_pairs (lambda (p) (list 'list (car p) (cadr p))))))))
						/* collect + trigger deploy on first keytable creation only.
						createtable inside init_code returns true on first creation.
						Materialized planner sources are rebuilt by the surrounding query, so
						keytables over them must be rebuilt with that source instead of
						reusing rows/aggregate columns from an earlier session value. */
						(define transient_group_source (and materialized_source (not is_fk_reuse)))
						(define collect_uses_materialized_where_condition
							(and
								materialized_source
								(reduce (extract_all_get_columns condition) (lambda (found mc)
									(or found (match mc
										'(_ col_expr)
										(match col_expr
											'((symbol get_column) alias_ _ col _)
											(and (equal?? alias_ tblvar) (equal?? col ".materialized_where_condition"))
											'((quote get_column) alias_ _ col _)
											(and (equal?? alias_ tblvar) (equal?? col ".materialized_where_condition"))
											false)
										false)))
									false)))
						(define guarded_keytable_collect (lambda (body)
							(if transient_group_source
								(list (quote begin)
									(list (quote droptable) schema grouptbl true)
									(list (quote if) keytable_init body nil))
								(list (quote if) keytable_init body nil))))
						(define collect_plan (if is_fk_reuse '()
							(if collect_single_aggregate_compute
								(list (if transient_group_source
									(guarded_keytable_collect
										(list 'begin
											(make_collect_single_aggregate (car ags))
											cleanup_plan))
									(list 'if keytable_init
										(list 'begin
											(make_collect_single_aggregate (car ags))
											cleanup_plan)
										(make_collect_single_aggregate (car ags)))))
								(list (guarded_keytable_collect
									(list 'begin
										(make_collect collect_uses_materialized_where_condition)
										cleanup_plan))))))
						(cons 'begin (merge
							(if (nil? runtime_local_compute_plan) '() (list runtime_local_compute_plan))
							(if (nil? group_value_local_compute_plan) '() (list group_value_local_compute_plan))
							collect_plan
							(if (nil? invalidation_plan) '() (list invalidation_plan))
							(list compute_plan)
							(list
								/* window+GROUP BY injection: after keytable is computed,
								scan it to fill promises with global totals, then wrap
								grouped_plan's resultrow to inject promise values. */
								(if (nil? (_wg_store "fields")) grouped_plan
									(begin
										(define _wg_ctr (newsession)) (_wg_ctr "n" 0)
										(define _wg_nn (lambda () (begin (_wg_ctr "n" (+ (_wg_ctr "n") 1)) (concat "__wgp_" (_wg_ctr "n")))))
										/* Fields with nested window functions must be rebuilt from the
										grouped row plus global window promises. Pure non-window fields can
										still pass straight through as grouped row lookups. */
										(define _wg_pl (newsession)) (_wg_pl "l" '())
										(define _wg_row_fields (newsession))
										(map_assoc fields (lambda (fk fv)
											(begin
												(map (materialized_source_expr_keys fv) (lambda (key)
													(_wg_row_fields key fk)))
												nil)))
										(define _wg_find_row_field (lambda (expr)
											(reduce (materialized_source_expr_keys expr) (lambda (found key)
												(if (not (nil? found))
													found
													(coalesce (_wg_row_fields key) nil)))
												nil)))
										(define _wg_promises (newsession))
										(define _wg_window_value (lambda (expr) (begin
											(define existing (reduce (materialized_source_expr_keys expr) (lambda (found key)
												(if (not (nil? found))
													found
													(coalesce (_wg_promises key) nil)))
												nil))
											(if (not (nil? existing))
												(symbol existing)
												(begin
													(define pn (_wg_nn))
													(define wfn (nth expr 1))
													(define wargs (nth expr 2))
													(define inner_agg (if (and (list? wargs) (> (count wargs) 0)) (car wargs) 1))
													(define agg_tuple (match inner_agg (cons (symbol aggregate) rest) rest (list inner_agg (quote +) 0)))
													(define acn (agg_col_name agg_tuple))
													(map (materialized_source_expr_keys expr) (lambda (key)
														(_wg_promises key pn)))
													(_wg_pl "l" (cons (list pn acn wfn) (_wg_pl "l")))
													(symbol pn))))))
										(define _wg_emit_window_expr (lambda (expr) (begin
											(define has_nested_window (not (equal? (extract_window_funcs expr) '())))
											(if has_nested_window
												(match expr
													(cons (symbol window_func) _) (_wg_window_value expr)
													(cons (quote window_func) _) (_wg_window_value expr)
													(cons sym args) (cons sym (map args _wg_emit_window_expr))
													expr)
												(begin
													(define row_field (_wg_find_row_field expr))
													(if (not (nil? row_field))
														(list (quote get_assoc) (symbol "__wgr") row_field)
														(match expr
															(cons sym args) (cons sym (map args _wg_emit_window_expr))
															expr)))))))
										(define _wg_out_fields (map_assoc (_wg_store "fields") (lambda (k v)
											(if (equal? (extract_window_funcs v) '())
												(list (quote get_assoc) (symbol "__wgr") k)
												(_wg_emit_window_expr v)))))
										/* scan keytable for each promise: aggregate the column globally */
										(define _wg_scans (map (_wg_pl "l") (lambda (pi) (match pi '(pn acn wfn)
											(begin
												(define reduce_op (match wfn "SUM" (quote +) "COUNT" (quote +) "MIN" (quote min) "MAX" (quote max) (quote +)))
												(define neutral (match wfn "SUM" 0 "COUNT" 0 "MIN" nil "MAX" nil 0))
												(list (quote set) (symbol pn)
													(list (quote scan)
														'(context_session_get "__memcp_tx")
														(scan-codegen-table schema grouptbl)
														(list (quote list) acn)
														(list (quote lambda) (list (symbol acn)) true)
														(list (quote list) acn)
														(list (quote lambda) (list (symbol acn)) (symbol acn))
														reduce_op
														neutral
														nil
														false)))))))
										/* wrap grouped_plan: preserve field/value pairs so outer
										materialization and result serialization keep the visible column
										names attached to the rebuilt expressions. */
										(define _wg_rr_body (cons (quote list) (merge (extract_assoc _wg_out_fields (lambda (k v)
											(list k v))))))
										(cons 'begin (merge _wg_scans (list
											(list (quote set) (symbol "__wg_orig_rr") (symbol "resultrow"))
											(list (quote set) (symbol "resultrow")
												(list (quote lambda) (list (symbol "__wgr"))
													(list (symbol "__wg_orig_rr") _wg_rr_body)))
											grouped_plan))))))))
				))
			)
			(begin /* multi-table GROUP BY via prejoin materialization */
				/* Scoped groups only materialize the tables inside their domain. Outer
				tables stay outside so the recursive single-table GROUP path can keep the
				keytable LEFT-joined to the surrounding row stream. Global multi-table
				GROUPs still materialize all participating tables. */
				(define _prejoin_order_tables_by_local_dependencies (lambda (tbls) (begin
					(define tbl_aliases (map tbls (lambda (td) (match td '(tv _ _ _ _) tv nil))))
					(define td_alias (lambda (td) (match td '(tv _ _ _ _) tv nil)))
					(define hidden_prejoin_source? (lambda (td)
						(match td
							'(tv _ ttbl _ _)
							(begin
								(define alias_text (string (if (nil? tv) ttbl tv)))
								(and (> (strlen alias_text) 0)
									(or
										(equal? (substr alias_text 0 1) ".")
										(strlike alias_text "domain_scalar_%"))))
							false)))
					(define td_local_deps (lambda (td)
						(match td
							'(a _ _ _ je)
							(if (nil? je)
								'()
								(filter (extract_tblvars je) (lambda (ref_alias)
									(and
										(not (equal? ref_alias a))
										(has? tbl_aliases ref_alias)))))
							'())))
					(define deps_satisfied? (lambda (td emitted_aliases)
						(reduce (td_local_deps td) (lambda (ok dep)
							(and ok (has? emitted_aliases dep)))
							true)))
					(define topo_state (reduce tbls (lambda (state _)
						(match state
							'(ordered remaining emitted_aliases)
							(begin
								(define ready (filter remaining (lambda (td)
									(deps_satisfied? td emitted_aliases))))
								(define ready_ordered (merge
									(filter ready (lambda (td) (not (hidden_prejoin_source? td))))
									(filter ready hidden_prejoin_source?)))
								(if (equal? ready '())
									state
									(list
										(merge ordered ready_ordered)
										(filter remaining (lambda (td) (not (has? ready_ordered td))))
										(merge emitted_aliases (map ready_ordered td_alias)))))
							state))
						(list '() tbls '())))
					(match topo_state
						'(ordered remaining _)
						(if (equal? remaining '())
							ordered
							(merge ordered remaining))
						tbls))))
				(define prejoin_group_tables (_prejoin_order_tables_by_local_dependencies _grp_tables))
				(define prejoin_group_aliases (map prejoin_group_tables (lambda (t) (match t '(tv _ _ _ _) tv ""))))
				(define prejoin_table_has_local_join_source? (lambda (td)
					(match td
						'(tv _ _ _ je)
						(if (nil? je)
							false
							(reduce (extract_tblvars je) (lambda (found ref_alias)
								(or found
									(and
										(not (equal? ref_alias tv))
										(has? prejoin_group_aliases ref_alias))))
								false))
						false)))
				(define prejoin_source_group_tables (map prejoin_group_tables (lambda (td)
					(match td
						'(tv tschema ttbl tisOuter tjoinexpr)
						(if (prejoin_table_has_local_join_source? td)
							(list tv tschema ttbl false tjoinexpr)
							td)
						td))))
				(define _grp_table_aliases (map prejoin_source_group_tables (lambda (t) (match t '(tv _ _ _ _) tv ""))))
				(define _prejoin_local_joinexpr_part (lambda (part)
					(reduce (extract_tblvars part) (lambda (acc tv)
						(and acc (has? _grp_table_aliases tv)))
						true)))
				(define _split_prejoin_joinexpr (lambda (expr)
					(begin
						(define _parts (flatten_and_terms (coalesceNil expr true)))
						(list
							(combine_and_terms (filter _parts _prejoin_local_joinexpr_part))
							(combine_and_terms (filter _parts (lambda (part) (not (_prejoin_local_joinexpr_part part)))))))))
				(define _prejoin_joinexpr_split (reduce prejoin_source_group_tables (lambda (acc td)
					(match acc '(tables_acc raw_deferred_acc deferred_acc)
						(match td '(tv tschema ttbl tisOuter tjoinexpr)
							(match (_split_prejoin_joinexpr tjoinexpr) '(local_joinexpr deferred_joinexpr)
								(list
									(merge tables_acc (list (list tv tschema ttbl tisOuter local_joinexpr)))
									(merge raw_deferred_acc (flatten_and_terms deferred_joinexpr))
									(merge deferred_acc (flatten_and_terms (replace_find_column deferred_joinexpr))))))))
					(list '() '() '())))
				(define prejoin_source_tables (car _prejoin_joinexpr_split))
				(define deferred_prejoin_joinexpr_parts_raw (cadr _prejoin_joinexpr_split))
				(define deferred_prejoin_joinexpr_parts (cadr (cdr _prejoin_joinexpr_split)))
				(define prejoin_alias_map (build_occurrence_alias_map prejoin_source_tables))
				(define _prejoin_alias_variants (lambda (tv tschema ttbl)
					(begin
						(define base_ttbl (scan_tagged_table_base ttbl))
						(merge
							(list tv)
							(if (equal? (visible_occurrence_alias tv) tv) '() (list (visible_occurrence_alias tv)))
							(if (equal? (visible_occurrence_alias tv) base_ttbl) (list (concat tschema "." base_ttbl)) '())))))
				(define _prejoin_td_alias_variants (lambda (td)
					(match td
						'(tv tschema ttbl _ _)
						(merge_unique (merge
							(list (_prejoin_alias_variants tv tschema ttbl))
							(map (materialized_source_physical_schema tschema ttbl tv schemas) (lambda (coldef)
								(extract_tblvars (coalesceNil (coldef "Expr") nil))))))
						'())))
				(define known_table_aliases (merge (map prejoin_source_tables (lambda (t) (match t
					'(tv tschema ttbl _ _) (_prejoin_alias_variants tv tschema ttbl)
					'())))))
				(define _prejoin_agg_refs_only_td? (lambda (td agg_args)
					(match td
						'(tv tschema ttbl _ _)
						(begin
							(define td_aliases (_prejoin_alias_variants tv tschema ttbl))
							(reduce (extract_tblvars agg_args) (lambda (ok ref_alias)
								(and ok (has? td_aliases ref_alias)))
								true))
						false)))
				(define _prejoin_materialized_agg_match_col (lambda (td agg_args agg_name)
					(match td
						'(tv tschema ttbl _ _)
						(if (_prejoin_agg_refs_only_td? td agg_args)
							(begin
								(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
								(reduce source_cols (lambda (found coldef)
									(if (not (nil? found)) found
										(begin
											(define field_name (coldef "Field"))
											(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
												(equal? (substr field_name 0 (strlen agg_name)) agg_name)
												(equal? (substr field_name (strlen agg_name) 1) "|"))
												(list tv field_name)
												nil))))
									nil))
							nil)
						nil)))
				(define rewrite_materialized_source_aggs (lambda (expr nested_agg) (match expr
					(cons (symbol aggregate) agg_args)
					(if nested_agg
						(begin
							(define canonical_agg_args (canonicalize_expr
								(planner_name_clear_case_flags agg_args)
								prejoin_alias_map))
							(define agg_name (serialize_canonical_expr canonical_agg_args))
							(define match_col (reduce prejoin_source_tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(_prejoin_materialized_agg_match_col td agg_args agg_name)
										nil)))
								nil))
							(if (nil? match_col)
								(match agg_args
									'(agg_expr agg_reduce agg_neutral)
									(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
									_ expr)
								(list (quote get_column) (car match_col) false (cadr match_col) false)))
						(match agg_args
							'(agg_expr agg_reduce agg_neutral)
							(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
							_ expr))
					(cons '(quote aggregate) agg_args)
					(if nested_agg
						(begin
							(define canonical_agg_args (canonicalize_expr
								(planner_name_clear_case_flags agg_args)
								prejoin_alias_map))
							(define agg_name (serialize_canonical_expr canonical_agg_args))
							(define match_col (reduce prejoin_source_tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(_prejoin_materialized_agg_match_col td agg_args agg_name)
										nil)))
								nil))
							(if (nil? match_col)
								(match agg_args
									'(agg_expr agg_reduce agg_neutral)
									(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
									_ expr)
								(list (quote get_column) (car match_col) false (cadr match_col) false)))
						(match agg_args
							'(agg_expr agg_reduce agg_neutral)
							(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
							_ expr))
					(cons sym args) (cons sym (map args (lambda (arg) (rewrite_materialized_source_aggs arg nested_agg))))
					expr)))
				(define post_group_tables '())
				/* resolve condition and fields */
				(define raw_condition (coalesceNil condition true))
				(define raw_condition_for_materialized_where raw_condition)
				(set condition (replace_find_column (coalesceNil condition true)))
				(define condition_for_materialized_where condition)
				(define post_group_condition (combine_and_terms deferred_prejoin_joinexpr_parts))
				(define raw_post_group_condition (combine_and_terms deferred_prejoin_joinexpr_parts_raw))
				/* 2-phase: separate aggregate-containing parts from materialize condition.
				Aggregates belong in the grouped_plan (evaluated after GROUP BY keytable),
				not in the prejoin_materialize_plan (which just fills the prejoin table). */
				(define contains_aggregate (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (contains_aggregate b))) false)
					false)))
				(define materialized_where_condition_field ".materialized_where_condition")
				(define materialized_where_condition_needed false)
				(define raw_condition_for_materialize_split
					(if materialized_where_condition_needed true raw_condition_for_materialized_where))
				(define condition_for_materialize_split
					(if materialized_where_condition_needed true condition_for_materialized_where))
				(define raw_condition_parts (match raw_condition_for_materialize_split
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)))
						parts (list raw_condition_for_materialize_split))
					(list raw_condition_for_materialize_split)))
				(define condition_parts (match condition_for_materialize_split
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)))
						parts (list condition_for_materialize_split))
					(list condition_for_materialize_split)))
				(define raw_aggregate_condition_parts (filter raw_condition_parts contains_aggregate))
				(define raw_materialize_condition_parts (filter raw_condition_parts (lambda (p) (not (contains_aggregate p)))))
				(define raw_materialize_condition (if (equal? 0 (count raw_materialize_condition_parts)) true
					(if (equal? 1 (count raw_materialize_condition_parts)) (car raw_materialize_condition_parts)
						(cons (quote and) raw_materialize_condition_parts))))
				(define aggregate_condition_parts (filter condition_parts contains_aggregate))
				(define materialize_condition_parts (filter condition_parts (lambda (p) (not (contains_aggregate p)))))
				(set condition (if (equal? 0 (count materialize_condition_parts)) true
					(if (equal? 1 (count materialize_condition_parts)) (car materialize_condition_parts)
						(cons (quote and) materialize_condition_parts))))
				/* Scoped prejoins must not consume outer-table predicates at materialize
				time. Keep local predicates on the prejoin source and defer the rest to the
				recursive grouped plan, where the keytable joins back to the outer row. */
				(define _grp_condition_split (split_condition condition _grp_ps_tables))
				(define condition (match _grp_condition_split '(prejoin_condition _) prejoin_condition))
				(define post_group_condition (match _grp_condition_split '(_ deferred_condition)
					(if (equal? deferred_condition true) post_group_condition
						(if (equal? post_group_condition true)
							deferred_condition
							(cons (quote and) (cons post_group_condition (list deferred_condition)))))))
				(define _raw_grp_condition_split (split_condition raw_materialize_condition _grp_ps_tables))
				(define raw_condition (match _raw_grp_condition_split '(raw_prejoin_condition _) raw_prejoin_condition))
				(define raw_post_group_condition (match _raw_grp_condition_split '(_ raw_deferred_condition)
					(if (equal? raw_deferred_condition true) raw_post_group_condition
						(if (equal? raw_post_group_condition true)
							raw_deferred_condition
							(cons (quote and) (cons raw_post_group_condition (list raw_deferred_condition)))))))
				/* Only true deferred condition aggregates belong in grouped_plan condition.
				Local HAVING on the recursive grouped prejoin stage must stay on the stage
				itself instead of being collapsed into the later count-cache condition path. */
				(define post_group_condition (if (or (nil? _grp_ps_tables) (equal? 0 (count aggregate_condition_parts))) post_group_condition
					(if (equal? post_group_condition true)
						(if (equal? 1 (count aggregate_condition_parts)) (car aggregate_condition_parts) (cons (quote and) aggregate_condition_parts))
						(cons (quote and) (cons post_group_condition aggregate_condition_parts)))))
				(define raw_post_group_condition (if (or (nil? _grp_ps_tables) (equal? 0 (count raw_aggregate_condition_parts))) raw_post_group_condition
					(if (equal? raw_post_group_condition true)
						(if (equal? 1 (count raw_aggregate_condition_parts)) (car raw_aggregate_condition_parts) (cons (quote and) raw_aggregate_condition_parts))
						(cons (quote and) (cons raw_post_group_condition raw_aggregate_condition_parts)))))
				(define materialized_where_equal_head? (lambda (head) (match head
					(symbol equal??) true
					'equal?? true
					'(quote equal??) true
					_ false)))
				(define materialized_where_equal_term? (lambda (term) (match term
					(cons head args) (and (materialized_where_equal_head? head) (equal? (count args) 2))
					false)))
				(define materialized_where_helper_alias? (lambda (alias_) (begin
					(define alias_s (if (nil? alias_) "" (string alias_)))
					(or
						(and (> (strlen alias_s) 0) (equal? (substr alias_s 0 1) "."))
						(strlike alias_s "domain_scalar_%")))))
				(define materialized_where_outer_aliases (if (nil? _grp_ps_tables) '()
					(filter (map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
						"")))
						(lambda (alias_) (if (materialized_where_helper_alias? alias_) false true)))))
				(define materialized_where_outer_aliases (if (or (nil? materialized_where_outer_aliases) (equal? materialized_where_outer_aliases '()))
					(filter (map tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
						"")))
						(lambda (alias_) (if (materialized_where_helper_alias? alias_) false true)))
					materialized_where_outer_aliases))
				(define materialized_where_refs_outer? (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (found alias_)
						(or found (has? materialized_where_outer_aliases alias_)))
						false)))
				(define materialized_where_refs_helper? (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (found alias_)
						(or found (materialized_where_helper_alias? alias_)))
						false)))
				(define materialized_where_refs_any? (lambda (expr)
					(not (equal? (extract_tblvars expr) '()))))
				(define materialized_where_equal_term? (lambda (term) (match term
					'(op left right)
					(and
						(materialized_where_equal_head? op)
						(materialized_where_refs_any? left)
						(materialized_where_refs_any? right)
						(or
							(materialized_where_refs_helper? left)
							(materialized_where_refs_helper? right)))
					false)))
				(define materialized_where_join_key_from_term (lambda (term) (match term
					'((symbol equal??) left right)
					(if (materialized_where_refs_outer? left)
						(if (materialized_where_refs_outer? right) nil right)
						(if (materialized_where_refs_outer? right) left nil))
					'((quote equal??) left right)
					(if (materialized_where_refs_outer? left)
						(if (materialized_where_refs_outer? right) nil right)
						(if (materialized_where_refs_outer? right) left nil))
					'(equal?? left right)
					(if (materialized_where_refs_outer? left)
						(if (materialized_where_refs_outer? right) nil right)
						(if (materialized_where_refs_outer? right) left nil))
					nil)))
				(define materialized_where_flatten_and_terms (lambda (expr) (match expr
					(cons sym parts) (if (or (equal?? sym (quote and)) (equal?? sym '(quote and)) (equal?? sym 'and))
						(merge (map parts materialized_where_flatten_and_terms))
						(if (or (nil? expr) (equal?? expr true)) '() (list expr)))
					_ (if (or (nil? expr) (equal?? expr true)) '() (list expr)))))
				(define materialized_where_combine_and_terms (lambda (parts) (begin
					(define _parts (filter parts (lambda (x) (and (not (nil? x)) (not (equal?? x true))))))
					(if (equal? _parts '()) true
						(if (equal? 1 (count _parts)) (car _parts)
							(cons (quote and) _parts))))))
				(define materialized_where_source_join_parts
					(merge (map tables (lambda (td) (match td
						'(tv _ ttbl _ tjoinexpr)
						(begin
							(define alias_ (if (nil? tv) ttbl tv))
							(if (materialized_where_helper_alias? alias_)
								(materialized_where_flatten_and_terms (coalesceNil tjoinexpr true))
								'()))
						'())))))
				(define materialized_where_raw_parts
					(merge
						(materialized_where_flatten_and_terms (coalesceNil raw_condition_for_materialized_where true))
						(materialized_where_flatten_and_terms (coalesceNil raw_condition true))
						materialized_where_source_join_parts))
				(define materialized_where_condition_parts
					(filter materialized_where_raw_parts (lambda (part)
						(and
							(not (equal?? part true))
							(not (materialized_where_equal_term? part))))))
				(define materialized_where_join_terms
					(filter
						(merge
							(materialized_where_flatten_and_terms (coalesceNil raw_post_group_condition true))
							materialized_where_raw_parts)
						materialized_where_equal_term?))
				(define materialized_where_join_key
					(reduce materialized_where_join_terms (lambda (found term)
						(if (not (nil? found))
							found
							(materialized_where_join_key_from_term term)))
						nil))
				(define materialized_where_condition_value
					(materialized_where_combine_and_terms
						materialized_where_condition_parts))
				(define materialized_where_join_key_from_helper (and
					(not (nil? materialized_where_join_key))
					(reduce (extract_tblvars materialized_where_join_key) (lambda (found alias_)
						(or found (materialized_where_helper_alias? alias_)))
						false)))
				(define materialized_where_condition_refs_helper
					(reduce (extract_tblvars materialized_where_condition_value) (lambda (found alias_)
						(or found (materialized_where_helper_alias? alias_)))
						false))
				(define materialized_where_required_helper_alias? (lambda (alias_)
					(and
						(materialized_where_helper_alias? alias_)
						(reduce tables (lambda (found td)
							(or found (match td
								'(tv _ ttbl is_outer _)
								(and
									(not is_outer)
									(or
										(equal?? alias_ tv)
										(and (nil? tv) (equal?? alias_ ttbl))))
								false)))
							false))))
				(define materialized_where_join_key_from_required_helper (and
					(not (nil? materialized_where_join_key))
					(reduce (extract_tblvars materialized_where_join_key) (lambda (found alias_)
						(or found (materialized_where_required_helper_alias? alias_)))
						false)))
				(define materialized_where_condition_missing_defaults_true? (lambda (expr) (match expr
					true true
					'true true
					'(quote true) true
					'((symbol coalesceNil) value fallback)
					(or
						(equal? fallback true)
						(equal? fallback (quote true)))
					'((quote coalesceNil) value fallback)
					(or
						(equal? fallback true)
						(equal? fallback (quote true)))
					(cons sym parts)
					(if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
						(reduce parts (lambda (ok part)
							(and ok (materialized_where_condition_missing_defaults_true? part)))
							true)
						false)
					false)))
				(define materialized_where_expr_has_anti_count_for_alias? (lambda (expr alias_) (match expr
					(cons sym args)
					(or
						(scalar_helper_count_anti_pred? expr alias_)
						(materialized_where_expr_has_anti_count_for_alias? sym alias_)
						(reduce (coalesceNil args '()) (lambda (found arg)
							(or found
								(materialized_where_expr_has_anti_count_for_alias? arg alias_)))
							false))
					false)))
				(define materialized_where_has_anti_count_projection?
					(reduce (extract_tblvars (coalesceNil materialized_where_join_key true)) (lambda (found alias_)
						(or found
							(and
								(materialized_where_helper_alias? alias_)
								(reduce_assoc raw_fields (lambda (field_found _field expr)
									(or field_found
										(materialized_where_expr_has_anti_count_for_alias? expr alias_)))
									false))))
						false))
				(define materialized_where_missing_join_key_value
					(if materialized_where_join_key_from_required_helper
						(materialized_where_condition_missing_defaults_true? materialized_where_condition_value)
						(if (and materialized_where_join_key_from_helper (not materialized_where_condition_refs_helper))
							materialized_where_has_anti_count_projection?
							true)))
				(define materialized_where_condition_expr
					(if (nil? materialized_where_join_key)
						materialized_where_condition_value
						(list (quote if)
							(list (quote nil?) materialized_where_join_key)
							materialized_where_missing_join_key_value
							materialized_where_condition_value)))
				(define materialized_where_has_outer_ref? (lambda (expr) (match expr
					'((symbol outer) _) true
					'((quote outer) _) true
					(cons sym args) (or
						(materialized_where_has_outer_ref? sym)
						(reduce args (lambda (found arg)
							(or found (materialized_where_has_outer_ref? arg))) false))
					false)))
				(define materialized_where_no_join_key_safe (and
					(nil? _stage_scope)
					(nil? materialized_where_join_key)
					(not (reduce materialized_where_source_join_parts (lambda (found part)
						(or found (materialized_where_refs_outer? part))) false))))
				(define materialized_where_source_join_refs_outer
					(reduce materialized_where_source_join_parts (lambda (found part)
						(or found (materialized_where_refs_outer? part))) false))
				(define materialized_where_condition_needed (and
					(not (nil? materialized_where_outer_aliases))
					(not (or (nil? materialized_where_condition_value) (equal?? materialized_where_condition_value true)))
					(not (contains_aggregate materialized_where_condition_value))
					(or
						(not materialized_where_source_join_refs_outer)
						(not (equal? (coalesceNil raw_stage_group '()) '(1))))
					(or
						(not (nil? materialized_where_join_key))
						materialized_where_no_join_key_safe)))
				(define raw_condition (if materialized_where_condition_needed
					true
					raw_condition))
				(define condition (if materialized_where_condition_needed
					true
					condition))
				(define _outer_visible_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl (visible_occurrence_alias tv))
						"")))))
				(define _outer_field_default_expr (lambda (alias_ col)
					(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
						(if (not (nil? found))
							found
							(if (equal? (coldef "Field") col)
								(match (coalesceNil (coldef "Expr") nil)
									'(coalesceNil _ fallback) fallback
									'((symbol coalesceNil) _ fallback) fallback
									'((quote coalesceNil) _ fallback) fallback
									nil)
								nil)))
						nil)))
				(define _projection_null_fallback_expr (lambda (expr)
					(match expr
						'(coalesceNil _ fallback) fallback
						'((symbol coalesceNil) _ fallback) fallback
						'((quote coalesceNil) _ fallback) fallback
						nil)))
				(define _keep_outer_column_or_default (lambda (expr alias_ col)
					(begin
						(define fallback (_outer_field_default_expr alias_ col))
						(if (nil? fallback)
							expr
							(list (quote coalesceNil) expr fallback)))))
				(define _keep_outer_field_early (lambda (expr)
					(match expr
						'((symbol get_column) alias_ _ col _)
						(begin
							(define fallback (_outer_field_default_expr alias_ col))
							(if (not (nil? fallback))
								(list (quote coalesceNil) expr fallback)
								(if (and (not (nil? _stage_scope)) (has? _outer_visible_aliases alias_)) expr nil)))
						'((quote get_column) alias_ _ col _)
						(begin
							(define fallback (_outer_field_default_expr alias_ col))
							(if (not (nil? fallback))
								(list (quote coalesceNil) expr fallback)
								(if (and (not (nil? _stage_scope)) (has? _outer_visible_aliases alias_)) expr nil)))
						_ nil)))
				(define resolved_fields (map_assoc fields (lambda (k v)
					(rewrite_materialized_source_aggs
						(coalesce (_keep_outer_field_early v) (replace_find_column v))
						false))))
				(define materialized_where_condition_fields
					(if materialized_where_condition_needed
						(list materialized_where_condition_field materialized_where_condition_expr)
						'()))
				(define resolved_prejoin_fields (map_assoc
					(merge fields materialized_where_condition_fields)
					(lambda (k v)
						(rewrite_materialized_source_aggs
							(coalesce (_keep_outer_field_early v) (replace_find_column v))
							false))))
				/* extract all get_column refs from group, fields, having, order, AND condition
				(condition may reference domain_scalar_ tables whose columns must be in the prejoin) */
				(define all_referenced_columns (merge
					(merge (map stage_group extract_all_get_columns))
					(merge (extract_assoc resolved_prejoin_fields (lambda (k v) (extract_all_get_columns v))))
					(if (nil? stage_post_group_condition) '() (extract_all_get_columns stage_post_group_condition))
					(extract_all_get_columns (coalesceNil raw_stage_having true))
					(extract_all_get_columns (coalesceNil stage_having true))
					(merge (map ags extract_all_get_columns))
					(merge (map (coalesce stage_order '()) (lambda (o) (match o '(col dir) (extract_all_get_columns col)))))
					(extract_all_get_columns (coalesceNil raw_condition true))
					(extract_all_get_columns (coalesceNil raw_post_group_condition true))
					(merge (map tables (lambda (td) (match td
						'(_ _ _ _ je) (if (nil? je) '() (extract_all_get_columns je))
						'()))))
				))
				/* filter out columns from partition-staged tables (they're not part of the prejoin) */
				(define _prejoin_materialized_ref_physical? (lambda (alias_ col)
					(reduce prejoin_source_tables (lambda (ok td)
						(match td
							'(tv tschema ttbl _ _)
							(if (has? (_prejoin_td_alias_variants td) alias_)
								(if (materialized-source? ttbl)
									(begin
										(define td_aliases (_prejoin_td_alias_variants td))
										(define expr_lookup (materialized_source_expr_lookup ttbl))
										(define source_alias_map (prejoin_canonical_sources ttbl))
										(define source_expr (if (nil? source_alias_map) nil
											(source_alias_map col)))
										(define source_refs_only_td (lambda (expr)
											(if (nil? expr)
												true
												(reduce (extract_tblvars expr) (lambda (ok ref_alias)
													(and ok
														(or
															(has? td_aliases ref_alias)
															(has? td_aliases (visible_occurrence_alias ref_alias)))))
													true))))
										(or
											(and
												(not (nil? (if (nil? expr_lookup) nil (expr_lookup col))))
												(source_refs_only_td source_expr))
											(not (nil? (find_materialized_field_by_name
												(materialized_source_physical_schema tschema ttbl tv schemas)
												col)))))
									true)
								ok)
							ok))
						true)))
				(define all_referenced_columns (filter all_referenced_columns (lambda (mc) (match mc
					'(name '((symbol get_column) alias_ _ col _))
					(and (has? known_table_aliases alias_)
						(_prejoin_materialized_ref_physical? alias_ col))
					'(name '((quote get_column) alias_ _ col _))
					(and (has? known_table_aliases alias_)
						(_prejoin_materialized_ref_physical? alias_ col))
					true))))
				/* compute prejoin table name and alias */
				(define prejoin_alias ".pj")
				(define lower_prejoin_lineage_expr (lambda (expr) (begin
					(define _lower_once (lambda (cur)
						(reduce prejoin_source_tables (lambda (inner td) (match td
							'(tv tschema ttbl _ _)
							(if (materialized-source? ttbl)
								(begin
									(define base_ttbl (scan_tagged_table_base ttbl))
									(define alias_pairs (filter
										(list
											(list (concat tschema "." base_ttbl) tv)
											(list base_ttbl tv)
											(list (concat tschema "." (string ttbl)) tv)
											(list (string ttbl) tv))
										(lambda (pair) (match pair
											'(src dst) (and (string? src) (not (equal? src dst)))
											false))))
									(define rewrite_mat_token_alias (lambda (node) (match node
										'((symbol get_column) alias_ ti col ci)
										(if (and (string? alias_) (strlike alias_ (concat "%__mat:" tv ":%")))
											(list (quote get_column) tv ti col ci)
											node)
										'((quote get_column) alias_ ti col ci)
										(if (and (string? alias_) (strlike alias_ (concat "%__mat:" tv ":%")))
											(list (quote get_column) tv ti col ci)
											node)
										(cons sym args) (cons sym (map args rewrite_mat_token_alias))
										node)))
									(preserve_current_materialized_field_refs ttbl tv
										(rewrite_mat_token_alias (rewrite_source_aliases alias_pairs inner))))
								inner)
							inner))
							cur)))
					(define expr2 (_lower_once expr))
					(define expr3 (_lower_once expr2))
					(define expr4 (_lower_once expr3))
					(if (equal? expr4 expr3)
						expr4
						expr4))))
				(define canonicalize_prejoin_source_expr (lambda (expr)
					(rewrite_source_aliases prejoin_alias_map
						(normalize_visible_aliases
							(lower_prejoin_lineage_expr expr)))))
				/* canonical prejoin key: source tables only (no alias), for maximal reuse across equivalent queries */
				(define prejoin_columns_base (reduce all_referenced_columns (lambda (acc mc)
					(begin
						(define source_lineage_expr (lower_prejoin_lineage_expr (cadr mc)))
						(define canonical_lineage_expr (canonicalize_expr
							(normalize_canonical_aliases (canonicalize_prejoin_source_expr source_lineage_expr))
							prejoin_alias_map))
						(define canon_name (serialize_canonical_expr canonical_lineage_expr))
						(if (reduce acc (lambda (found mc2) (or found (equal? (car mc2) canon_name))) false)
							acc
							(merge acc (list (list canon_name source_lineage_expr)))))) '()))
				/* Later wrapper groups must be able to reuse derived output fields by their
				stable alias instead of re-serializing nested scalar/window expressions into
				prejoin/keytable column names. Materialize only row-local projected fields on
				the prejoin. Logical aggregate/window sentinels must stay deferred until the
				later group/window stage; otherwise raw `(aggregate ...)` / `(window_func ...)`
				nodes leak into the prejoin row materializer and become executable runtime
				code instead of logical planner markers. */
				(define prejoin_materializable_projection? (lambda (field_expr)
					(and
						(equal? (extract_aggregates field_expr) '())
						(equal? (extract_window_funcs field_expr) '()))))
				(define prejoin_columns_projected (reduce (extract_assoc resolved_prejoin_fields (lambda (field_name field_expr)
					(match field_expr
						'((symbol get_column) _ _ _ _) nil
						'((quote get_column) _ _ _ _) nil
						(if (prejoin_materializable_projection? field_expr)
							(list field_name field_expr)
							nil))))
					(lambda (acc mc)
						(if (or (nil? mc)
							(reduce acc (lambda (found mc2) (or found (equal? (car mc2) (car mc)))) false))
							acc
							(merge acc (list mc))))
					prejoin_columns_base))
				(define prejoin_columns prejoin_columns_projected)
				(define prejoin_column_names (map prejoin_columns car))
				(define prejoin_col_names prejoin_column_names)
				(define prejoin_has_computed_projection (reduce prejoin_columns (lambda (found mc)
					(or found
						(match (cadr mc)
							'((symbol get_column) _ _ _ _) false
							'((quote get_column) _ _ _ _) false
							true)))
					false))
				(define prejoin_schema_def (map prejoin_columns (lambda (mc)
					(list "Field" (car mc) "Type" "any" "Expr" (cadr mc)))))
				(define post_group_term_prejoin_local? (lambda (term)
					(match term
						(cons head _)
						(and
							(or
								(not (list? head))
								(match head
									'(quote _) true
									_ false))
							(not (contains_aggregate term))
							(reduce (extract_tblvars term) (lambda (ok alias_)
								(and ok
									(or
										(has? known_table_aliases alias_)
										(has? known_table_aliases (visible_occurrence_alias alias_)))))
								true))
						false)))
				(define local_post_group_condition_raw
					(if (nil? _stage_scope)
						(combine_and_terms
							(filter (flatten_and_terms (coalesceNil raw_post_group_condition true))
								(lambda (term) (not (contains_aggregate term)))))
						(combine_and_terms
							(filter (flatten_and_terms (coalesceNil raw_post_group_condition true))
								post_group_term_prejoin_local?))))
				(define local_post_group_condition_raw
					(if (and (nil? _stage_scope)
						(or (nil? local_post_group_condition_raw) (equal? local_post_group_condition_raw true)))
						(combine_and_terms
							(filter (flatten_and_terms (coalesceNil post_group_condition true))
								(lambda (term) (not (contains_aggregate term)))))
						local_post_group_condition_raw))
				(define materialized_where_post_join_condition_raw
					(if materialized_where_condition_needed
						(materialized_where_combine_and_terms
							(filter
								(materialized_where_flatten_and_terms (coalesceNil raw_post_group_condition true))
								materialized_where_equal_term?))
						true))
				(define local_post_group_condition_raw
					(materialized_where_combine_and_terms
						(list local_post_group_condition_raw materialized_where_post_join_condition_raw)))
				(define raw_condition_for_prejoin
					(if materialized_where_condition_needed
						(materialized_where_combine_and_terms (list raw_condition local_post_group_condition_raw))
						(combine_and_terms (list raw_condition local_post_group_condition_raw))))
				(define prejoin_source_aliases
					(merge_unique (merge (map prejoin_source_tables _prejoin_td_alias_variants))))
				(define prejoin_part_refs_only_sources? (lambda (part)
					(reduce (extract_tblvars part) (lambda (ok ref_alias)
						(and ok (has? prejoin_source_aliases ref_alias)))
						true)))
				(define hidden_prejoin_source_alias? (lambda (alias_) (begin
					(define alias_s (if (nil? alias_) "" (string alias_)))
					(or
						(and (> (strlen alias_s) 0)
							(equal? (substr alias_s 0 1) "."))
						(strlike alias_s "domain_scalar_%")))))
				(define prejoin_expr_refs_alias? (lambda (alias_ expr)
					(has? (extract_tblvars expr) alias_)))
				(define prejoin_expr_refs_only_alias? (lambda (alias_ expr) (begin
					(define refs (extract_tblvars expr))
					(and
						(not (equal? refs '()))
						(reduce refs (lambda (ok ref_alias)
							(and ok (equal?? ref_alias alias_)))
							true)))))
				(define prejoin_expr_refs_visible_only? (lambda (expr) (begin
					(define refs (extract_tblvars expr))
					(and
						(not (equal? refs '()))
						(reduce refs (lambda (ok ref_alias)
							(and ok (not (hidden_prejoin_source_alias? ref_alias))))
							true)))))
				(define prejoin_pure_hidden_visible_equality? (lambda (alias_ part) (match part
					'(op left right)
					(and
						(materialized_where_equal_head? op)
						(or
							(and
								(prejoin_expr_refs_only_alias? alias_ left)
								(prejoin_expr_refs_visible_only? right))
							(and
								(prejoin_expr_refs_only_alias? alias_ right)
								(prejoin_expr_refs_visible_only? left))))
					false)))
				(define prejoin_filter_requires_match? (lambda (alias_ expr)
					(reduce (materialized_where_flatten_and_terms (coalesceNil expr true)) (lambda (found part)
						(or found
							(begin
								(and
									(prejoin_expr_refs_alias? alias_ part)
									(not (prejoin_pure_hidden_visible_equality? alias_ part))
									(match part
										'(op _ _)
										(or
											(equal?? op (quote equal?))
											(equal?? op (quote equal??))
											(equal?? op (quote =))
											(equal?? op (quote >))
											(equal?? op (quote <))
											(equal?? op (quote >=))
											(equal?? op (quote <=)))
										false)))))
						false)))
				(define prejoin_join_part_target (lambda (part) (begin
					(define refs (extract_tblvars part))
					(define hidden_refs (filter refs hidden_prejoin_source_alias?))
					(define visible_refs (filter refs (lambda (ref_alias)
						(if (hidden_prejoin_source_alias? ref_alias) false true))))
					(if (and (> (count hidden_refs) 0) (> (count visible_refs) 0))
						(car hidden_refs)
						nil))))
				(define prejoin_raw_condition_parts
					(if materialized_where_condition_needed
						(merge
							materialized_where_join_terms
							(filter (flatten_and_terms (coalesceNil raw_condition_for_prejoin true))
								prejoin_part_refs_only_sources?))
						(flatten_and_terms (coalesceNil raw_condition_for_prejoin true))))
				(define prejoin_materialize_join_parts
					(merge
						(filter prejoin_raw_condition_parts (lambda (part)
							(not (nil? (prejoin_join_part_target part)))))
						(merge (map prejoin_source_tables (lambda (td)
							(match td
								'(_ _ _ _ tjoinexpr)
								(filter
									(materialized_where_flatten_and_terms (coalesceNil tjoinexpr true))
									(lambda (part)
										(not (nil? (prejoin_join_part_target part)))))
								'()))))))
				(define raw_condition_for_prejoin_materialize
					(if materialized_where_condition_needed
						(materialized_where_combine_and_terms (filter prejoin_raw_condition_parts (lambda (part)
							(and
								(nil? (prejoin_join_part_target part))
								(prejoin_part_refs_only_sources? part)))))
						(combine_and_terms (filter prejoin_raw_condition_parts (lambda (part)
							(nil? (prejoin_join_part_target part)))))))
				(define prejoin_join_part_available_for_aliases? (lambda (part aliases) (begin
					(define target_alias (prejoin_join_part_target part))
					(define refs (extract_tblvars part))
					(define local_visible_refs (filter refs (lambda (ref_alias)
						(and
							(not (has? aliases ref_alias))
							(not (hidden_prejoin_source_alias? ref_alias))
							(or
								(has? prejoin_source_aliases ref_alias)
								(has? prejoin_source_aliases (visible_occurrence_alias ref_alias)))))))
					(and
						(not (nil? target_alias))
						(has? aliases target_alias)
						(not (equal? local_visible_refs '()))
						(reduce refs (lambda (ok ref_alias)
							(and ok
								(or
									(has? aliases ref_alias)
									(has? prejoin_source_aliases ref_alias)
									(has? prejoin_source_aliases (visible_occurrence_alias ref_alias)))))
							true)))))
				(define prejoin_join_part_for_aliases (lambda (aliases)
					(combine_and_terms (filter prejoin_materialize_join_parts (lambda (part)
						(prejoin_join_part_available_for_aliases? part aliases))))))
				(define prejoin_bare_get_column_expr? (lambda (expr) (match expr
					'((symbol get_column) _ _ _ _) true
					'((quote get_column) _ _ _ _) true
					false)))
				(define prejoin_expr_refs_local_visible? (lambda (expr aliases)
					(reduce (extract_tblvars expr) (lambda (found ref_alias)
						(or found
							(and
								(not (has? aliases ref_alias))
								(not (hidden_prejoin_source_alias? ref_alias))
								(or
									(has? prejoin_source_aliases ref_alias)
									(has? prejoin_source_aliases (visible_occurrence_alias ref_alias))))))
						false)))
				(define prejoin_materialized_join_part_allowed? (lambda (aliases)
					(reduce prejoin_materialize_join_parts (lambda (found part)
						(or found
							(prejoin_join_part_available_for_aliases? part aliases)))
						false)))
				(define add_prejoin_join_part (lambda (old part)
					(if (or (nil? part) (equal? part true))
						old
						(if (or (nil? old) (equal? old true))
							part
							(list (quote and) old part)))))
				(define prejoin_source_tables_ordered_for_materialized_where
					(if materialized_where_condition_needed
						(merge
							(filter prejoin_source_tables (lambda (td) (match td
								'(tv _ ttbl _ _)
								(if (hidden_prejoin_source_alias? (if (nil? tv) ttbl tv)) false true)
								true)))
							(filter prejoin_source_tables (lambda (td) (match td
								'(tv _ ttbl _ _)
								(hidden_prejoin_source_alias? (if (nil? tv) ttbl tv))
								false))))
						prejoin_source_tables))
				(define prejoin_source_tables_for_materialize
					(map prejoin_source_tables_ordered_for_materialized_where (lambda (td)
						(match td
							'(tv tschema ttbl tisOuter tjoinexpr)
							(begin
								(define alias_ (if (nil? tv) ttbl tv))
								(define alias_variants (_prejoin_td_alias_variants td))
								(define join_part
									(if (and (materialized-source? ttbl)
										(not (prejoin_materialized_join_part_allowed? alias_variants)))
										true
										(prejoin_join_part_for_aliases alias_variants)))
								(define raw_base_joinexpr (if (and
									materialized_where_condition_needed
									(hidden_prejoin_source_alias? alias_))
									(materialized_where_combine_and_terms
										(filter
											(materialized_where_flatten_and_terms (coalesceNil tjoinexpr true))
											materialized_where_equal_term?))
									tjoinexpr))
								(define base_joinexpr
									(materialized_where_combine_and_terms
										(filter
											(materialized_where_flatten_and_terms (coalesceNil raw_base_joinexpr true))
											(lambda (part)
												(begin
													(define target_alias (prejoin_join_part_target part))
													(or
														(nil? target_alias)
														(has? alias_variants target_alias)))))))
								(define hidden_source_projected
									(reduce (extract_assoc resolved_prejoin_fields (lambda (_field_name field_expr)
										(has? (extract_tblvars field_expr) alias_)))
										(lambda (found uses_alias) (or found uses_alias))
										false))
								(define hidden_source_projected_domain_key
									(reduce (extract_assoc resolved_prejoin_fields (lambda (_field_name field_expr)
										(match field_expr
											'((symbol get_column) ref_alias _ ref_col _)
											(and (equal?? ref_alias alias_)
												(string? ref_col)
												(>= (strlen ref_col) 5)
												(equal? (substr ref_col 0 5) "__kt_"))
											'((quote get_column) ref_alias _ ref_col _)
											(and (equal?? ref_alias alias_)
												(string? ref_col)
												(>= (strlen ref_col) 5)
												(equal? (substr ref_col 0 5) "__kt_"))
											false)))
										(lambda (found uses_alias) (or found uses_alias))
										false))
								(define hidden_source_referenced_later
									(or
										hidden_source_projected
										(has? (extract_tblvars (coalesceNil condition true)) alias_)
										(has? (extract_tblvars (coalesceNil raw_condition true)) alias_)
										(has? (extract_tblvars (coalesceNil post_group_condition true)) alias_)
										(has? (extract_tblvars (coalesceNil raw_post_group_condition true)) alias_)
										(has? (extract_tblvars (coalesceNil stage_post_group_condition true)) alias_)
										(has? (extract_tblvars (coalesceNil stage_having true)) alias_)))
								(define hidden_source_direct_domain_match
									(and
										hidden_source_projected_domain_key
										(not materialized_where_condition_needed)
										(not (prejoin_filter_requires_match? alias_ base_joinexpr))))
								(define base_is_outer (or tisOuter
									(and
										(hidden_prejoin_source_alias? alias_)
										hidden_source_referenced_later
										(not hidden_source_direct_domain_match)
										(or
											hidden_source_projected
											(not (prejoin_filter_requires_match? alias_ base_joinexpr))))))
								(if (or (nil? join_part) (equal? join_part true))
									(list tv tschema ttbl base_is_outer base_joinexpr)
									(list tv tschema ttbl base_is_outer (add_prejoin_join_part base_joinexpr join_part))))
							td))))
				(define prejoin_row_domain_raw (combine_and_terms
					(merge
						(if (or (nil? raw_condition_for_prejoin) (equal? raw_condition_for_prejoin true)) '()
							(list raw_condition_for_prejoin))
						(merge (map prejoin_source_tables (lambda (td) (match td
							'(_ _ _ _ tjoinexpr)
							(if (or (nil? tjoinexpr) (equal? tjoinexpr true)) '()
								(list tjoinexpr))
							'())))))))
				(define prejoin_row_domain_name_expr
					(planner_name_clear_case_flags (replace_find_column prejoin_row_domain_raw)))
				(define prejoin_condition_name (serialize_canonical_expr
					(canonicalize_expr
						(normalize_canonical_aliases (canonicalize_prejoin_source_expr prejoin_row_domain_name_expr))
						prejoin_alias_map)))
				(define prejoin_materialized_where_condition_name (if materialized_where_condition_needed
					(serialize_canonical_expr
						(canonicalize_expr
							(normalize_canonical_aliases
								(canonicalize_prejoin_source_expr
									(planner_name_clear_case_flags
										(replace_find_column materialized_where_condition_expr))))
							prejoin_alias_map))
					nil))
				(define prejoin_runtime_terms (runtime_cache_unique_terms (merge
					(runtime_cache_terms_from_exprs (list
						prejoin_row_domain_raw
						raw_materialize_condition
						raw_post_group_condition
						materialized_where_condition_expr))
					(extract_session_lookup_terms (stage_cache_query stage)))))
				(define prejoin_runtime_suffix
					(runtime_cache_suffix_from_terms prejoin_runtime_terms))
				(define prejoin_condition_name
					(concat prejoin_condition_name
						(if (nil? prejoin_materialized_where_condition_name)
							""
							(concat "|" prejoin_materialized_where_condition_name))
						prejoin_runtime_suffix))
				(define prejointbl (compact-prejoin-table-name
					prejoin_source_tables
					prejoin_col_names
					prejoin_condition_name))
				/* capture outer schema and table name for trigger code generation */
				(define prejoin_schema schema)
				(define pj_schema schema) /* needed in quoted runtime code below */
				(define prejoin_table_name prejointbl)
				(define temp_source_table? materialized-source?)
				(define prejoin_has_temp_source (reduce prejoin_source_tables (lambda (found td)
					(or found (match td '(_ _ ttbl _ _) (temp_source_table? ttbl) false)))
					false))
				(register_prejoin_materialized_metadata
					canonicalize_prejoin_source_expr
					prejointbl
					prejoin_columns
					prejoin_alias_map
					prejoin_source_tables
					prejoin_schema_def)
				/* prejoin table creation deferred to runtime (guard at plan assembly below) */
				(define covered_partition_stages (filter partition_stages (lambda (ps)
					(reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
						(or acc (has? known_table_aliases a))) false))))
				(define prejoin_materialize_plan
					(build_legacy_prejoin_materialize_plan
						schema
						prejoin_schema
						prejointbl
						prejoin_columns
						prejoin_column_names
						prejoin_source_tables_for_materialize
						raw_condition_for_prejoin_materialize
						covered_partition_stages
						schemas
						replace_find_column))
				/* Design contract:
				Keep get_column / aggregate / window sentinels logical for as long as
				possible. Materialized stages may register lineage and visible schemas,
				but must not eagerly bake physical .prejoin/.cache field names into the
				next logical stage. The final get_column -> scan symbol substitution
				should happen only when building the actual scan/map/filter code path.
				The helper below is therefore only for scan-time lowering via the
				recursive replace_find_column of the next stage, not for pre-rewriting
				group keys/fields/having/order themselves. */
				/* rewrite all column references from the materialized source scope to
				the physical prejoin columns. Outer-scope references must stay intact. */
				(define rewrite_as_prejoin_column (lambda (expr) (match expr
					'((symbol get_column) src_alias ti col ci) (begin
						(define _visible_alias (visible_occurrence_alias src_alias))
						(define _scope_match (or (has? known_table_aliases src_alias)
							(has? known_table_aliases _visible_alias)))
						(define expr_lookup (materialized_source_expr_lookup prejointbl))
						(define _rewritten_source_expr (rewrite_source_aliases prejoin_alias_map (normalize_visible_aliases expr)))
						(define _logical_source_expr (lower_prejoin_lineage_expr _rewritten_source_expr))
						(define _lookup_exprs (list
							expr
							(normalize_visible_aliases expr)
							(normalize_canonical_aliases expr)
							_rewritten_source_expr
							(normalize_canonical_aliases _rewritten_source_expr)
							_logical_source_expr
							(normalize_canonical_aliases _logical_source_expr)))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce _lookup_exprs (lambda (found lookup_expr)
								(if (not (nil? found))
									found
									(reduce (materialized_source_expr_keys lookup_expr) (lambda (found2 key)
										(if (not (nil? found2)) found2
											(coalesce (expr_lookup key) nil)))
										nil)))
								nil)))
						(define direct_field (coalesce direct_field
							(materialized_field_from_get_column_name
								(materialized_source_physical_schema prejoin_schema prejointbl prejoin_alias schemas)
								expr)))
						(define fallback_field (sanitize_temp_name
							(serialize_canonical_expr (canonicalize_expr _logical_source_expr prejoin_alias_map))))
						(define fallback_field_exists
							(reduce prejoin_columns (lambda (found mc)
								(or found (equal? (car mc) fallback_field)))
								false))
						(if (not (nil? direct_field))
							(list (quote get_column) prejoin_alias false direct_field false)
							(if (and _scope_match fallback_field_exists)
								(list (quote get_column) prejoin_alias false fallback_field false)
								expr)))
					'((quote get_column) src_alias ti col ci) (begin
						(define _visible_alias (visible_occurrence_alias src_alias))
						(define _scope_match (or (has? known_table_aliases src_alias)
							(has? known_table_aliases _visible_alias)))
						(define expr_lookup (materialized_source_expr_lookup prejointbl))
						(define _rewritten_source_expr (rewrite_source_aliases prejoin_alias_map (normalize_visible_aliases expr)))
						(define _logical_source_expr (lower_prejoin_lineage_expr _rewritten_source_expr))
						(define _lookup_exprs (list
							expr
							(normalize_visible_aliases expr)
							(normalize_canonical_aliases expr)
							_rewritten_source_expr
							(normalize_canonical_aliases _rewritten_source_expr)
							_logical_source_expr
							(normalize_canonical_aliases _logical_source_expr)))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce _lookup_exprs (lambda (found lookup_expr)
								(if (not (nil? found))
									found
									(reduce (materialized_source_expr_keys lookup_expr) (lambda (found2 key)
										(if (not (nil? found2)) found2
											(coalesce (expr_lookup key) nil)))
										nil)))
								nil)))
						(define direct_field (coalesce direct_field
							(materialized_field_from_get_column_name
								(materialized_source_physical_schema prejoin_schema prejointbl prejoin_alias schemas)
								expr)))
						(define fallback_field (sanitize_temp_name
							(serialize_canonical_expr (canonicalize_expr _logical_source_expr prejoin_alias_map))))
						(define fallback_field_exists
							(reduce prejoin_columns (lambda (found mc)
								(or found (equal? (car mc) fallback_field)))
								false))
						(if (not (nil? direct_field))
							(list (quote get_column) prejoin_alias false direct_field false)
							(if (and _scope_match fallback_field_exists)
								(list (quote get_column) prejoin_alias false fallback_field false)
								expr)))
					(cons sym args) (cons sym (map args rewrite_as_prejoin_column))
					expr)))
				/* Preserve logical lineage into the recursive stage.
				Do not carry physical .prejoin/.cache column names forward; instead lower
				the raw stage expressions back onto their logical source lineage first.
				build_scan stays the only place that finally substitutes onto the current
				stage's physical scan symbols. */
				(define prejoin_group_expr_match? (lambda (left right) (begin
					(define normalize_for_group_match (lambda (node)
						(replace_columns_from_expr
							(normalize_canonical_aliases
								(rewrite_source_aliases prejoin_alias_map
									(normalize_visible_aliases
										(lower_prejoin_lineage_expr node)))))))
					(or
						(equal? left right)
						(equal? (normalize_for_group_match left) (normalize_for_group_match right))))))
				(define prejoin_group_expr_has_ref? (lambda (expr) (match expr
					'((symbol get_column) _ _ _ _) true
					'((quote get_column) _ _ _ _) true
					'((symbol outer) _) true
					'((quote outer) _) true
					(cons sym args)
					(or
						(prejoin_group_expr_has_ref? sym)
						(reduce args (lambda (found arg)
							(or found (prejoin_group_expr_has_ref? arg)))
							false))
					_ (if (or (nil? expr) (string? expr) (number? expr))
						false
						(match (split (string expr) ".")
							'(_ _) true
							false)))))
				(define prejoin_projected_column_for_expr (lambda (expr) (begin
					(define lowered (lower_prejoin_lineage_expr expr))
					(if (not (prejoin_group_expr_has_ref? lowered))
						nil
						(reduce prejoin_columns (lambda (found mc)
							(if (not (nil? found))
								found
								(if (prejoin_group_expr_match? lowered (lower_prejoin_lineage_expr (cadr mc)))
									mc
									nil)))
							nil)))))
				(define prejoin_group_key_representative_from_pair (lambda (expr left right) (begin
					(define left_column (prejoin_projected_column_for_expr left))
					(define right_column (prejoin_projected_column_for_expr right))
					(if (not (prejoin_group_expr_has_ref? expr))
						nil
						(if (and
							(prejoin_group_expr_match? expr left)
							(nil? left_column)
							(not (nil? right_column)))
							right
							(if (and
								(prejoin_group_expr_match? expr right)
								(nil? right_column)
								(not (nil? left_column)))
								left
								nil))))))
				(define prejoin_group_key_representative_from_term (lambda (expr term) (match term
					'((symbol equal??) left right)
					(prejoin_group_key_representative_from_pair expr left right)
					'((quote equal??) left right)
					(prejoin_group_key_representative_from_pair expr left right)
					'(equal?? left right)
					(prejoin_group_key_representative_from_pair expr left right)
					nil)))
				(define prejoin_group_key_representative_expr (lambda (expr)
					(coalesce
						(reduce materialized_where_join_terms (lambda (found term)
							(if (not (nil? found))
								found
								(prejoin_group_key_representative_from_term expr term)))
							nil)
						expr)))
				(define lower_prejoin_group_expr_with_representatives (lambda (expr use_join_representatives) (begin
					(define prejoin_positive_op? (lambda (op)
						(or
							(equal? op (quote >))
							(equal? op '(quote >))
							(equal? op (symbol ">"))
							(equal? (string op) ">"))))
					(define prejoin_coalesce_op? (lambda (op)
						(or
							(equal? op (quote coalesce))
							(equal? op '(quote coalesce))
							(equal? op (symbol "coalesce"))
							(equal? op (quote coalesceNil))
							(equal? op '(quote coalesceNil))
							(equal? op (symbol "coalesceNil")))))
					(define prejoin_count_value_read? (lambda (node) (match node
						'((symbol get_column) alias_ _ col _)
						(and (equal? alias_ prejoin_alias)
							(string? col)
							(or (equal? col "value") (strlike col "%\"value\"%")))
						'((quote get_column) alias_ _ col _)
						(and (equal? alias_ prejoin_alias)
							(string? col)
							(or (equal? col "value") (strlike col "%\"value\"%")))
						false)))
					(define normalize_prejoin_count_guard (lambda (node) (match node
						'(op left right)
						(if (prejoin_positive_op? op)
							(match left
								'(cop val fallback)
								(if (and
									(prejoin_coalesce_op? cop)
									(equal? fallback right)
									(prejoin_count_value_read? val))
									(list (quote >) (list (quote coalesceNil) val 0) 0)
									(list op (normalize_prejoin_count_guard left) (normalize_prejoin_count_guard right)))
								(list op (normalize_prejoin_count_guard left) (normalize_prejoin_count_guard right)))
							(list op (normalize_prejoin_count_guard left) (normalize_prejoin_count_guard right)))
						(cons sym args) (cons sym (map args normalize_prejoin_count_guard))
						node)))
					(define lower_inner (lambda (node preserve_aggregate) (match node
						(cons (symbol aggregate) args)
						(if preserve_aggregate
							(match args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_inner agg_expr false) agg_reduce agg_neutral)
								(cons (quote aggregate) (map args (lambda (arg) (lower_inner arg false)))))
							(lower_inner_projected node preserve_aggregate))
						(cons '(quote aggregate) args)
						(if preserve_aggregate
							(match args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_inner agg_expr false) agg_reduce agg_neutral)
								(cons (quote aggregate) (map args (lambda (arg) (lower_inner arg false)))))
							(lower_inner_projected node preserve_aggregate))
						_ (lower_inner_projected node preserve_aggregate))))
					(define lower_inner_projected (lambda (node preserve_aggregate) (begin
						(define equality_lowered (match node
							'(op left right)
							(if (and use_join_representatives (materialized_where_equal_head? op))
								(list op
									(lower_prejoin_group_expr_with_representatives left false)
									(lower_prejoin_group_expr_with_representatives right false))
								nil)
							nil))
						(if (not (nil? equality_lowered))
							equality_lowered
							(begin
								(define representative_node (if use_join_representatives
									(prejoin_group_key_representative_expr node)
									node))
								(define lowered (lower_prejoin_lineage_expr representative_node))
								(define projected_column (prejoin_projected_column_for_expr representative_node))
								(if (not (nil? projected_column))
									(begin
										(define projected_read (list (quote get_column) prejoin_alias false (car projected_column) false))
										(define projected_fallback (if use_join_representatives
											nil
											(_projection_null_fallback_expr (cadr projected_column))))
										(if (nil? projected_fallback)
											projected_read
											(list (quote coalesceNil) projected_read projected_fallback)))
									(match lowered
										'((symbol get_column) _ _ _ _) lowered
										'((quote get_column) _ _ _ _) lowered
										(cons sym args) (cons sym (map args (lambda (arg) (lower_inner arg preserve_aggregate))))
										lowered)))))))
					(normalize_prejoin_count_guard (lower_inner expr true)))))
				(define lower_prejoin_group_expr (lambda (expr)
					(lower_prejoin_group_expr_with_representatives expr false)))
				(define lower_prejoin_group_key_expr (lambda (expr)
					(lower_prejoin_group_expr_with_representatives expr true)))
				(define grouped_fields (map_assoc raw_fields (lambda (k v)
					(lower_prejoin_group_key_expr v))))
				(define grouped_keys (map (coalesce raw_stage_group '()) lower_prejoin_group_key_expr))
				(define grouped_hidden_key_not_nil_terms
					(if materialized_where_condition_needed
						'()
						(filter
							(map grouped_keys (lambda (group_key)
								(if (reduce (extract_tblvars group_key) (lambda (found ref_alias)
									(or found (hidden_prejoin_source_alias? ref_alias)))
									false)
									(list (quote not) (list (quote nil?) group_key))
									nil)))
							(lambda (term) (not (nil? term))))))
				(define grouped_stage_alias_result (if (nil? grouped_keys)
					nil
					(make_keytable_schema schema prejointbl grouped_keys prejoin_alias)))
				(define grouped_stage_alias (if (nil? grouped_stage_alias_result) nil
					(car grouped_stage_alias_result)))
				(define grouped_stage_key_names (if (nil? grouped_stage_alias_result) '()
					(car (cdr grouped_stage_alias_result))))
				(define rewrite_group_key_to_group_alias (lambda (expr)
					(coalesce
						(reduce (produceN (count grouped_keys)) (lambda (found i)
							(if (not (nil? found))
								found
								(if (equal? expr (nth grouped_keys i))
									(list (quote get_column) grouped_stage_alias false (nth grouped_stage_key_names i) false)
									nil)))
							nil)
						(match expr
							(cons sym args) (cons sym (map args rewrite_group_key_to_group_alias))
							expr))))
				(define grouped_outer_condition_aliases (map _grp_ps_tables (lambda (td) (match td
					'(tv _ _ _ _) (if (nil? tv) "" tv)
					""))))
				(define grouped_outer_condition_term? (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (acc tv)
						(or acc (has? grouped_outer_condition_aliases tv)))
						false)))
				(define countlike_prejoin_aggregate_expr? (lambda (expr) (match expr
					1 true
					'(if _ 1 0) true
					'(if _ true false) true
					'(if _ false true) true
					'(if _ 0 1) true
					_ false)))
				(define rewrite_local_prejoin_count_term (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args)
					(match agg_args
						'(agg_expr + 0)
						(if (countlike_prejoin_aggregate_expr? agg_expr)
							(list (quote aggregate) 1 + 0)
							expr)
						_ expr)
					(cons '(quote aggregate) agg_args)
					(match agg_args
						'(agg_expr + 0)
						(if (countlike_prejoin_aggregate_expr? agg_expr)
							(list (quote aggregate) 1 + 0)
							expr)
						_ expr)
					(cons sym args) (cons sym (map args rewrite_local_prejoin_count_term))
					expr)))
				(define keep_grouped_post_group_term (lambda (expr)
					(if (or
						(grouped_outer_condition_term? expr)
						(post_group_term_prejoin_local? expr))
						expr
						(if (equal? (extract_aggregates expr) '())
							nil
							(rewrite_local_prejoin_count_term expr)))))
				(define grouped_having_source (if (and
					materialized_where_condition_needed
					(not (equal? (coalesceNil raw_stage_group '()) '()))
					(not (equal? group_value_local_fields '())))
					(begin
						(define grouped_having_source_parts
							(filter
								(materialized_where_flatten_and_terms (coalesceNil raw_stage_post_group_condition true))
								(lambda (part)
									(or
										(contains_aggregate part)
										(not (materialized_where_refs_helper? part))))))
						(define grouped_having_source_combined
							(materialized_where_combine_and_terms grouped_having_source_parts))
						(if (equal? grouped_having_source_combined true) nil grouped_having_source_combined))
					raw_stage_post_group_condition))
				(define grouped_having
					(rewrite_group_key_to_group_alias (lower_prejoin_group_key_expr grouped_having_source)))
				(define grouped_order (if (nil? raw_stage_order) nil
					(map raw_stage_order (lambda (o) (match o '(col dir)
						(list (lower_prejoin_group_key_expr col) dir))))))
				(define grouped_outer_tables (map _grp_ps_tables (lambda (td) (match td
					'(tv tschema ttbl toisOuter je)
					(list (if (nil? tv) ttbl tv) tschema ttbl toisOuter je)
					td))))
				(define grouped_outer_aliases (map grouped_outer_tables (lambda (td) (match td '(tv _ _ _ _) tv ""))))
				(define grouped_outer_schema_bindings (merge (map grouped_outer_tables (lambda (td) (match td
					'(tv tschema ttbl _ _)
					(list tv (materialized_source_schema tschema ttbl tv schemas))
					'())))))
				/* recursive call with single prejoin table.
				Contract:
				1. The prejoin materializes the complete row-domain that can be decided
				before grouping: local join predicates, row-local filters and
				lineage columns needed later.
				2. The grouped cache built on top of that prejoin must only see
				group-domain conditions. Terms containing aggregates are rewritten
				against keytable/temp columns and evaluated on the grouped table.
				3. Purely local prejoin predicates must not be copied into the grouped
				cache suffix again, otherwise unrelated grouped plans alias the same
				prejoin differently and cache names/filters drift apart.
				Scoped groups keep their outer tables outside the prejoin so later
				field expressions can still read them after the keytable LEFT JOIN. */
				(define grouped_plan_condition_base (if (nil? _grp_ps_tables)
					nil
					(begin
						(define grouped_plan_condition_base_raw (combine_and_terms
							(filter (map (flatten_and_terms (coalesceNil raw_post_group_condition true))
								keep_grouped_post_group_term)
								(lambda (x) (and (not (nil? x)) (not (equal? x true)))))))
						(if (or (nil? grouped_plan_condition_base_raw) (equal? grouped_plan_condition_base_raw true))
							nil
							(lower_prejoin_group_key_expr grouped_plan_condition_base_raw)))))
				(define grouped_plan_condition_base
					(combine_and_terms (merge
						(if (or (nil? grouped_plan_condition_base) (equal? grouped_plan_condition_base true))
							'()
							(list grouped_plan_condition_base))
						grouped_hidden_key_not_nil_terms
						(if (and
							materialized_where_condition_needed
							(not (equal? (coalesceNil raw_stage_group '()) '())))
							(begin
								(define materialized_where_group_parts
									(filter materialized_where_condition_parts (lambda (part)
										(not (materialized_where_refs_helper? part)))))
								(define materialized_where_group_condition
									(materialized_where_combine_and_terms materialized_where_group_parts))
								(if (or (nil? materialized_where_group_condition) (equal? materialized_where_group_condition true))
									'()
									(list (lower_prejoin_group_key_expr materialized_where_group_condition))))
							'())
						(if (and
							materialized_where_condition_needed
							(or
								(equal? (coalesceNil raw_stage_group '()) '())
								(equal? group_value_local_fields '())))
							(list (list (quote coalesceNil)
								(list (quote get_column) prejoin_alias false materialized_where_condition_field false)
								true))
							'()))))
				(define recursive_replace_find_column (lambda (expr)
					(match expr
						'((symbol get_column) alias_ _ _ _) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								'((quote get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								_ resolved))
						'((quote get_column) alias_ _ _ _) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								'((quote get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								_ resolved))
						(cons sym args) (cons sym (map args recursive_replace_find_column))
						expr)))
				(define recursive_replace_find_column_condition (lambda (expr)
					(match expr
						'((symbol get_column) alias_ ti col ci) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								'((quote get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								_ (recursive_replace_find_column resolved)))
						'((quote get_column) alias_ ti col ci) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								'((quote get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								_ (recursive_replace_find_column resolved)))
						(cons sym args) (cons sym (map args recursive_replace_find_column_condition))
						expr)))
				(define grouped_having_for_recursive (if (nil? grouped_having) nil
					(if (nil? _grp_ps_tables)
						grouped_having
						(recursive_replace_find_column_condition grouped_having))))
				(define grouped_plan_condition grouped_plan_condition_base)
				(define grouped_stage_limit_redundant
					(and
						(equal? (coalesceNil grouped_order '()) '())
						(not (nil? stage_limit))
						(<= stage_limit 1)
						(or (nil? stage_offset) (equal? stage_offset 0))
						(not (equal? (coalesceNil grouped_keys '()) '()))))
				(define grouped_stage_limit (if grouped_stage_limit_redundant nil stage_limit))
				(define grouped_stage_offset (if grouped_stage_limit_redundant nil stage_offset))
				/* rebuild group stage for recursive call.
				HAVING stays attached to the recursive grouped stage. Only deferred
				post-group outer predicates continue as condition. */
				(define grouped_stage (if is_dedup
					(make_dedup_stage grouped_keys
						(if (nil? _stage_scope) nil (list prejoin_alias)))
					(make_group_stage grouped_keys grouped_having_for_recursive grouped_order grouped_stage_limit grouped_stage_offset
						(if (nil? _stage_scope) nil (list prejoin_alias))
						nil)))
				(define grouped_fields_for_recursive (if is_dedup
					(map_assoc raw_fields (lambda (k v)
						(recursive_replace_find_column v)))
					grouped_fields))
				(define transform_recursive_stage (lambda (s)
					(begin
						(define _sg (coalesceNil (stage_group_cols s) '()))
						(define _so (coalesceNil (stage_order_list s) '()))
						(define _spa (stage_partition_aliases s))
						(define _sonce (stage_once_limit s))
						(define _sc (stage_condition s))
						(if (stage_is_dedup s)
							(stage_rebuild_with_meta s
								(make_dedup_stage
									(map _sg recursive_replace_find_column)
									_spa)
								recursive_replace_find_column
								(lambda (a) a))
							(if (and (not (nil? _spa)) (or (nil? _sg) (equal? _sg '())))
								(stage_rebuild_with_meta s
									(make_stage
										'()
										nil
										(map _so (lambda (o) (match o '(col dir) (list (recursive_replace_find_column col) dir))))
										(coalesceNil (stage_limit_partition_cols s) 0)
										(stage_limit_val s)
										(stage_offset_val s)
										false
										_spa
										(stage_init_code s)
										(if (nil? _sc) nil (recursive_replace_find_column _sc))
										_sonce)
									recursive_replace_find_column
									(lambda (a) a))
								(stage_rebuild_with_meta s
									(make_group_stage
										(map _sg recursive_replace_find_column)
										(recursive_replace_find_column (stage_having_expr s))
										(map _so (lambda (o) (match o '(col dir) (list (recursive_replace_find_column col) dir))))
										(stage_limit_val s)
										(stage_offset_val s)
										_spa
										(stage_init_code s))
									recursive_replace_find_column
									(lambda (a) a)))))))
				/* drop partition stages covered by the prejoin (all tables materialized) */
				(define partition_stage_covered_by_prejoin? (lambda (ps)
					(reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
						(or acc (has? known_table_aliases a))) false)))
				(define remaining_partition_stages (filter partition_stages (lambda (ps)
					(not (partition_stage_covered_by_prejoin? ps)))))
				(define rest_groups_for_recursive (filter rest_groups (lambda (s)
					(not (or
						(partition_stage_covered_by_prejoin? s)
						(and
							(not (nil? (stage_partition_aliases s)))
							(equal? (coalesceNil (stage_group_cols s) '()) '())))))))
				(define grouped_all_stages (cons grouped_stage
					(map rest_groups_for_recursive transform_recursive_stage)))
				(define schemas_for_recursive_base
					(list prejoin_alias prejoin_schema_def))
				(define grouped_result (if (nil? _grp_ps_tables)
					(begin
						(define no_outer_group_condition_raw (combine_and_terms
							(filter (flatten_and_terms (coalesceNil raw_post_group_condition true))
								contains_aggregate)))
						(define no_outer_group_condition (if (or (nil? no_outer_group_condition_raw) (equal? no_outer_group_condition_raw true))
							nil
							no_outer_group_condition_raw))
						/* no outer-scope aliases remain here, so the recursive call is a
						plain single-table GROUP BY over the materialized prejoin table.
						Only aggregate-dependent terms survive into the grouped filter. */
						(define no_outer_group_stage (if is_dedup
							(make_dedup_stage raw_stage_group nil)
							(make_group_stage grouped_keys grouped_having grouped_order grouped_stage_limit grouped_stage_offset nil nil)))
						(build_queryplan schema
							(list (list prejoin_alias schema prejointbl false nil))
							raw_fields
							no_outer_group_condition
							(merge (list no_outer_group_stage) (map rest_groups_for_recursive transform_recursive_stage) remaining_partition_stages)
							schemas_for_recursive_base
							recursive_replace_find_column
							update_target))
					(build_queryplan schema
						(merge (list (list prejoin_alias schema prejointbl false nil)) grouped_outer_tables)
						grouped_fields_for_recursive
						grouped_plan_condition
						(merge grouped_all_stages remaining_partition_stages)
						(merge schemas_for_recursive_base grouped_outer_schema_bindings)
						recursive_replace_find_column
						update_target)))
				/* Register compact invalidation hooks instead of serializing full
				incremental prejoin maintenance plans into every query. Rebuilding a
				cache helper after source DML is cheaper to compile and avoids embedding
				recursive materialized-source scans inside trigger bodies. */
				(define seen_trigger_tables (newsession))
				(define prejoin_alias_table_lookup (merge
					(map tables (lambda (td) (match td
						'(tv src_schema src_tbl _ _) (list (string tv) (list src_schema (scan_tagged_table_base src_tbl)))
						'())))
					(map prejoin_source_tables (lambda (td) (match td
						'(tv src_schema src_tbl _ _) (list (string tv) (list src_schema (scan_tagged_table_base src_tbl)))
						'())))))
				(define prejoin_expr_dependency_tables (lambda (expr) (match expr
					'((symbol table) dep_schema dep_tbl) (list (list dep_schema (scan_tagged_table_base dep_tbl)))
					'((quote table) dep_schema dep_tbl) (list (list dep_schema (scan_tagged_table_base dep_tbl)))
					'((symbol get_column) dep_alias _ _ _) (begin
						(define dep_tbl_info (if (nil? dep_alias) nil (coalesce (prejoin_alias_table_lookup (string dep_alias)) nil)))
						(if (nil? dep_tbl_info) '() (list dep_tbl_info)))
					'((quote get_column) dep_alias _ _ _) (begin
						(define dep_tbl_info (if (nil? dep_alias) nil (coalesce (prejoin_alias_table_lookup (string dep_alias)) nil)))
						(if (nil? dep_tbl_info) '() (list dep_tbl_info)))
					(cons sym args) (merge_unique
						(merge
							(prejoin_expr_dependency_tables sym)
							(merge (map args prejoin_expr_dependency_tables))))
					'())))
				(define prejoin_dependency_exprs (merge
					(map prejoin_columns (lambda (mc) (cadr mc)))
					(list raw_condition raw_post_group_condition)))
				(define prejoin_dependency_tables (merge_unique
					(merge
						(map tables (lambda (trigger_tbl) (match trigger_tbl
							'(_ src_schema src_tbl _ _) (list src_schema (scan_tagged_table_base src_tbl))
							'())))
						(merge (map prejoin_dependency_exprs prejoin_expr_dependency_tables)))))
				(define prejoin_dependency_tables (filter prejoin_dependency_tables (lambda (td)
					(match td
						'(dep_schema dep_tbl) (and (string? dep_schema) (string? dep_tbl))
						false))))
				(define pj_trigger_registrations
					(filter (map prejoin_dependency_tables (lambda (trigger_tbl)
						(match trigger_tbl '(src_schema src_base_tbl)
							(begin
								(define trigger_table_key (concat src_schema "." src_base_tbl))
								(if (or (temp_source_table? src_base_tbl) (seen_trigger_tables trigger_table_key)) nil
									(begin (seen_trigger_tables trigger_table_key true)
										(list 'register_prejoin_invalidation src_schema src_base_tbl prejoin_schema prejoin_table_name))))))) (lambda (x) (not (nil? x)))))
					/* assemble: createtable returns true on first creation -> materialize + deploy triggers.
					Subsequent calls: table exists; source DML drops it so the next query rebuilds. */
					(cons 'begin (merge
						(list (list 'droptable pj_schema prejointbl true))
						(list
							/* createtable true = new table, table_empty = restart recovery shell */
							(list 'if (list 'or
							(list 'createtable pj_schema prejointbl
								(cons 'list (map prejoin_column_names (lambda (col) (list 'list "column" col "any" '(list) '(list)))))
								query_temp_table_options_code true)
							(list 'table_empty? (list 'table pj_schema prejointbl)))
							(cons 'begin (cons (list 'time prejoin_materialize_plan "materialize") pj_trigger_registrations))
							nil))
					(list grouped_result)))
			)
		)
	) (optimize (begin
			/* grouping has been removed; now to the real data: */
			(if (and (not (nil? rest_groups)) (not (equal? rest_groups '())))
				(error (concat "non-group stage must be last: " (serialize rest_groups))))
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
							/* register column in planner schema so downstream plan can reference it
							without the column existing in storage at compile time */
							(planned_materialized_fields tbl (list (list "Field" orc_col_name "Type" "any")))
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
								(createcolumn (table schema tbl) orc_col_name "any" '()
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
									(define non_window_cols (reduce (extract_assoc fields (lambda (k v)
										(extract_columns_for_tblvar tblvar (replace_find_column v))))
										(lambda (acc cols) (merge_unique acc cols))
										'()))
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
					/* If ORDER BY targets a single inner-joined table, drive the join from that
					table so LIMIT/OFFSET applies on the actual sort source instead of the
					original parser order. This keeps semantics correct for cases like
					ORDER BY derived_tbl.col DESC LIMIT n over inner joins. */
					(define _order_driver_aliases (reduce (coalesce stage_order '()) (lambda (acc order_item) (match order_item
						'(col _dir) (match col
							'((symbol get_column) alias_ _ _ _) (append_unique acc alias_)
							'((quote get_column) alias_ _ _ _) (append_unique acc alias_)
							_ acc)
						acc))
						'()))
					(define _all_inner_tables (reduce tables (lambda (acc td) (match td
						'(_ _ _ isOuter _) (and acc (not isOuter))
						acc))
						true))
					(define _order_driver_alias (if (and _all_inner_tables (equal? 1 (count _order_driver_aliases)))
						(car _order_driver_aliases)
						nil))
					(define ordered_tables (if (nil? _order_driver_alias)
						tables
						(merge
							(filter tables (lambda (td) (match td '(tblvar _ _ _ _) (equal?? tblvar _order_driver_alias) false)))
							(filter tables (lambda (td) (match td '(tblvar _ _ _ _) (not (equal?? tblvar _order_driver_alias)) true))))))
					/* build_scan now takes is_first parameter to apply offset/limit only to outermost scan */
					(define build_scan (lambda (tables condition is_first last_scan_ctx pending_once_name)
						(match tables
							(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
								(define base_tbl (scan_tagged_table_base tbl))
								(define tbl_scan_order (scan_tagged_table_order tbl))
								(define tbl_scan_limit (scan_tagged_table_limit tbl))
								(define tbl_scan_offset (scan_tagged_table_offset tbl))
								(define tbl_scan_partcols (scan_tagged_table_partition_cols tbl))
								(define tbl_once_limit (scan_tagged_table_once_limit tbl))
								(define tblvar_is_scalar_helper_ord (or (scalar_helper_root_alias? tblvar) (strlike (string tblvar) "domain_scalar_%")))
								(define tblvar_is_nested_scalar_helper_ord (scalar_helper_nested_alias? tblvar))
								(define scan_condition (lower_materialized_scan_condition schema base_tbl tblvar condition))
								(define scan_joinexpr (lower_materialized_scan_condition schema base_tbl tblvar (coalesceNil joinexpr true)))
								(define visible_fields (lower_materialized_emit_assoc schema base_tbl tblvar fields))
								(define is_update_target_ord (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
								(define visible_ut_cols_ord (if is_update_target_ord
									(lower_materialized_emit_assoc schema base_tbl tblvar (nth update_target 1))
									'()))
								(define ut_extra_cols_ord (if is_update_target_ord
									(reduce (extract_assoc visible_ut_cols_ord (lambda (k v) (extract_columns_for_tblvar tblvar v)))
										(lambda (acc cols) (merge_unique acc cols))
										'())
									'()))
								(set cols (collect_scan_base_cols tblvar scan_condition visible_fields tables partition_stages ut_extra_cols_ord))
								(define _ps_ord_split (if (not (nil? tbl_once_limit)) nil
									(find_partition_stage_for_alias partition_stages tblvar)))
								(define split_is_outer_ord isOuter)
								(define resolved_scan_joinexpr_ord (replace_find_column scan_joinexpr))
								(match (split_scan_condition split_is_outer_ord resolved_scan_joinexpr_ord scan_condition tables) '(now_condition later_condition) (begin
									(define effective_later_condition (if (and isOuter (equal? now_condition later_condition)) true later_condition))
									(define scalar_helper_local_now_condition_ord (scalar_helper_outer_local_scan_condition (and isOuter tblvar_is_scalar_helper_ord) tblvar base_tbl scan_condition tables))
									(define scan_now_condition (strip_outer_scalar_helper_ref_terms
										(if (or tblvar_is_scalar_helper_ord tblvar_is_nested_scalar_helper_ord)
											(scalar_helper_outer_join_terms tblvar now_condition)
											(if isOuter
												(combine_and_terms (merge
													(flatten_and_terms scalar_helper_local_now_condition_ord)
													(filter (flatten_and_terms (coalesceNil now_condition true)) (lambda (part)
														(outer_scan_join_filter_part? tblvar (materialized-source? base_tbl) part)))))
												now_condition))))
									(set cols (extend_scan_cols_for_later_condition tblvar cols effective_later_condition))
									(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar scan_now_condition) (extract_outer_columns_for_tblvar tblvar scan_now_condition))))
									/* check partition_stages for this table. Tagged scans still override the
									local stage config, but scoped partition stages must now also work when
									this helper is the driver after join_reorder. */
									(define _ps_ord _ps_ord_split)
									(define _ps_once_limit (if (nil? _ps_ord) nil (stage_once_limit _ps_ord)))
									/* tagged helper scans override the local scan config; otherwise use
									partition-stage order first and the outer ORDER only on the driver scan. */
									(define _eff_order (if (not (nil? tbl_once_limit))
										tbl_scan_order
										(if (nil? _ps_ord) stage_order (coalesceNil (stage_order_list _ps_ord) '()))))
									/* extract order cols for this tblvar */
									(set ordercols (extract_scan_order_cols_for_tblvar _eff_order tblvar))
									(set dirs (extract_scan_order_dirs_for_tblvar _eff_order tblvar))

									/* offset/limit: tagged helper scans carry their own local limits. */
									(define ord_raw_scan_offset (if (not (nil? tbl_once_limit))
										(coalesceNil tbl_scan_offset 0)
										(if (not (nil? _ps_ord)) (coalesceNil (stage_offset_val _ps_ord) 0)
											(if is_first stage_offset 0))))
									(define ord_raw_scan_limit (if (not (nil? tbl_once_limit))
										(coalesceNil tbl_scan_limit -1)
										(if (not (nil? _ps_ord)) (coalesceNil (stage_limit_val _ps_ord) -1)
											(if is_first (coalesceNil stage_limit -1) -1))))
									(define ord_raw_scan_partcols (if (not (nil? tbl_once_limit))
										tbl_scan_partcols
										(if (not (nil? _ps_ord)) (coalesceNil (stage_limit_partition_cols _ps_ord) 0)
											(if is_first stage_partcols 0))))
									(define ord_effective_once_limit (coalesce tbl_once_limit _ps_once_limit))
									(define ord_can_push_once_limit (and (equal? effective_later_condition true) (equal? tables '())))
									(define ord_defer_tagged_once_limit (and (not ord_can_push_once_limit) (not (nil? tbl_once_limit)) (>= tbl_once_limit 2)))
									(define ord_scan_input_offset (if ord_defer_tagged_once_limit 0 ord_raw_scan_offset))
									(define ord_scan_input_limit (if ord_defer_tagged_once_limit -1 ord_raw_scan_limit))
									(define ord_scan_input_partcols (if ord_defer_tagged_once_limit 0 ord_raw_scan_partcols))
									(define ord_once_contract (if (or (nil? ord_effective_once_limit) (not ord_can_push_once_limit))
										nil
										(make_once_limit_scan_contract ord_scan_input_limit ord_scan_input_offset ord_scan_input_partcols ord_effective_once_limit tblvar condition joinexpr tbl)))
									(define ord_scan_offset (if (nil? ord_once_contract) ord_scan_input_offset (once_limit_scan_contract_offset ord_once_contract)))
									(define ord_scan_limit (if (nil? ord_once_contract) ord_scan_input_limit (once_limit_scan_contract_limit ord_once_contract)))
									(define ord_scan_partcols (if (nil? ord_once_contract) ord_scan_input_partcols (once_limit_scan_contract_partition_cols ord_once_contract)))

									(define ord_scan_mapcols (if is_update_target_ord (cons list (cons "$update" cols)) (cons list cols)))
									(define ord_scan_mapfn_params (if is_update_target_ord
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									(define ord_once_name (if (nil? ord_effective_once_limit)
										nil
										(if (nil? ord_once_contract)
											(make_once_limit_promise_name ord_scan_input_limit ord_effective_once_limit tblvar condition joinexpr tbl)
											(once_limit_scan_contract_promise_name ord_once_contract))))
									(define ord_child_body (build_scan tables effective_later_condition false (list schema base_tbl tblvar) (coalesce pending_once_name ord_once_name)))
									(define ord_scan_body ord_child_body)
									/* emit init code from partition stage if present */
									(define _ps_init (if (nil? _ps_ord) nil (stage_init_code _ps_ord)))
									(define _ord_scan_core (scan_wrapper 'scan_order schema base_tbl
										/* condition */
										(cons list filtercols)
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr_for_scan tblvar scan_now_condition)))
										/* sortcols, sortdirs */
										(cons list ordercols)
										(cons list dirs)
										ord_scan_partcols
										ord_scan_offset
										ord_scan_limit
										/* extract columns and store them into variables */
										ord_scan_mapcols
										(list (symbol "lambda") ord_scan_mapfn_params ord_scan_body)
										/* reduce+neutral for DML */
										(if is_update_target_ord (symbol "+") nil)
										(if is_update_target_ord 0 nil)
										isOuter
									))
									(define _ord_scan (wrap_once_limit_scan ord_once_name _ord_scan_core))
									(if (nil? _ps_init) _ord_scan (list (quote begin) _ps_init _ord_scan))
								))
							)
							'() /* final inner */ (if (nil? update_target)
								(begin
									(define emit_fields (if (nil? last_scan_ctx) fields
										(match last_scan_ctx
											'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar fields)
											fields)))
									(define emit_replace (if (nil? last_scan_ctx)
										replace_columns_from_expr
										(match last_scan_ctx
											'(_ _ scan_tblvar) (lambda (expr)
												(replace_columns_from_expr_for_scan scan_tblvar expr))
											replace_columns_from_expr)))
									(define result_body (list (symbol "resultrow") (runtime_heap_list_ast (map_assoc emit_fields (lambda (k v) (emit_replace v))))))
									(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body)))
								/* DML mode: emit mutation payload; actual DELETE/UPDATE runs in build_dml_plan's resultrow wrapper */
								(begin (define _ut_cols (nth update_target 1))
									(define _ut_tag (nth update_target 2))
									(define _ut_cols (if (nil? last_scan_ctx) _ut_cols
										(match last_scan_ctx
											'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar _ut_cols)
											_ut_cols)))
									(if (equal? _ut_cols '())
										(begin
											(define result_body (list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" (symbol "$update") "__values" nil)))
											(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body) 0))
										(begin
											(define result_body (list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" (symbol "$update") "__values" (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v)))))))
											(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body) 0)))))
						)
					))
					(build_scan ordered_tables (replace_find_column condition) true nil nil)
				) (begin
						/* unordered unlimited scan */

						/* TODO: sort tables according to join plan */
						/* TODO: match tbl to inner query vs string */
						(define build_scan (lambda (tables condition last_scan_ctx bound_update_expr pending_once_name)
							(match tables
								(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
									(define base_tbl (scan_tagged_table_base tbl))
									(define tbl_scan_order (scan_tagged_table_order tbl))
									(define tbl_scan_limit (scan_tagged_table_limit tbl))
									(define tbl_scan_offset (scan_tagged_table_offset tbl))
									(define tbl_scan_partcols (scan_tagged_table_partition_cols tbl))
									(define tbl_once_limit (scan_tagged_table_once_limit tbl))
									(define tblvar_is_scalar_helper (or (scalar_helper_root_alias? tblvar) (strlike (string tblvar) "domain_scalar_%")))
									(define tblvar_is_nested_scalar_helper (scalar_helper_nested_alias? tblvar))
									(define scan_condition (lower_materialized_scan_condition schema base_tbl tblvar condition))
									(define scan_joinexpr (lower_materialized_scan_condition schema base_tbl tblvar (coalesceNil joinexpr true)))
									(define visible_fields (lower_materialized_emit_assoc schema base_tbl tblvar fields))
									/* check if this table is the UPDATE target */
									(define is_update_target (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
									(define visible_ut_cols (if is_update_target
										(lower_materialized_emit_assoc schema base_tbl tblvar (nth update_target 1))
										'()))
									/* also extract cols needed for SET expressions in update_target */
									(define ut_extra_cols (if is_update_target
										(reduce (extract_assoc visible_ut_cols (lambda (k v) (extract_columns_for_tblvar tblvar v)))
											(lambda (acc cols) (merge_unique acc cols))
											'())
										'()))
									(set cols (collect_scan_base_cols tblvar scan_condition visible_fields tables partition_stages ut_extra_cols))
									/* For UPDATE target: prepend $update to mapcols */
									(define scan_mapcols (if is_update_target (cons list (cons "$update" cols)) (cons list cols)))
									(define scan_mapfn_params (if is_update_target
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									/* split condition in those ANDs that still contain get_column from tables and those evaluatable now */
									(define _ps_split (if (not (nil? tbl_once_limit))
										nil
										(find_partition_stage_for_alias partition_stages tblvar)))
									(define split_is_outer isOuter)
									(define resolved_scan_joinexpr (replace_find_column scan_joinexpr))
									(match (split_scan_condition split_is_outer resolved_scan_joinexpr scan_condition tables) '(now_condition later_condition) (begin
										(define effective_later_condition (if (and isOuter (equal? now_condition later_condition)) true later_condition))
										(define scalar_helper_local_now_condition (scalar_helper_outer_local_scan_condition (and isOuter tblvar_is_scalar_helper) tblvar base_tbl scan_condition tables))
										(define scan_now_condition (strip_outer_scalar_helper_ref_terms
											(if (or tblvar_is_scalar_helper tblvar_is_nested_scalar_helper)
												(scalar_helper_outer_join_terms tblvar now_condition)
												(if isOuter
													(combine_and_terms (merge
														(flatten_and_terms scalar_helper_local_now_condition)
														(filter (flatten_and_terms (coalesceNil now_condition true)) (lambda (part)
															(outer_scan_join_filter_part? tblvar (materialized-source? base_tbl) part)))))
													now_condition))))
										(set cols (extend_scan_cols_for_later_condition tblvar cols effective_later_condition))
										(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar scan_now_condition) (extract_outer_columns_for_tblvar tblvar scan_now_condition))))
										/* optimize: skip .(1) DUAL scan when no columns needed (1 row, no data) */
										(if (and (equal? base_tbl ".(1)") (equal? cols (list)) (equal? filtercols (list)) (equal? tables '()))
											(begin
												/* The skipped DUAL row still carries any constant/materialized
												predicate that split_scan_condition classified as now_condition.
												Forward it explicitly, otherwise no-FROM predicates like
												NOT IN (subselect) vanish before the materialized temp scan
												can lower their aggregate sentinel. */
												(define deferred_condition (combine_and_terms (merge
													(flatten_and_terms now_condition)
													(flatten_and_terms effective_later_condition))))
												(build_scan tables deferred_condition last_scan_ctx bound_update_expr pending_once_name))
											(begin
												(define next_update_expr (if is_update_target (symbol "__dml_update_bound") bound_update_expr))
												/* check partition_stages: does this table have a per-table partition limit? */
												(define _ps _ps_split)
												(define _ps_once_limit (if (nil? _ps) nil (stage_once_limit _ps)))
												(define _tagged_scan (scan_tagged_table_needs_scan_order tbl))
												(define scan_raw_partcols (if _tagged_scan
													tbl_scan_partcols
													(if (nil? _ps) 0 (coalesceNil (stage_limit_partition_cols _ps) 0))))
												(define scan_raw_limit (if _tagged_scan
													(coalesceNil tbl_scan_limit -1)
													(if (nil? _ps) -1 (coalesceNil (stage_limit_val _ps) -1))))
												(define scan_raw_offset (if _tagged_scan
													(coalesceNil tbl_scan_offset 0)
													(if (nil? _ps) 0 (coalesceNil (stage_offset_val _ps) 0))))
												(define scan_effective_once_limit (coalesce tbl_once_limit _ps_once_limit))
												(define scan_can_push_once_limit (and (equal? effective_later_condition true) (equal? tables '())))
												(define scan_defer_tagged_once_limit (and (not scan_can_push_once_limit) (not (nil? tbl_once_limit)) (>= tbl_once_limit 2)))
												(define scan_input_offset (if scan_defer_tagged_once_limit 0 scan_raw_offset))
												(define scan_input_limit (if scan_defer_tagged_once_limit -1 scan_raw_limit))
												(define scan_input_partcols (if scan_defer_tagged_once_limit 0 scan_raw_partcols))
												(define scan_once_contract (if (or (nil? scan_effective_once_limit) (not scan_can_push_once_limit))
													nil
													(make_once_limit_scan_contract scan_input_limit scan_input_offset scan_input_partcols scan_effective_once_limit tblvar condition joinexpr tbl)))
												(define scan_once_name (if (nil? scan_effective_once_limit)
													nil
													(if (nil? scan_once_contract)
														(make_once_limit_promise_name scan_input_limit scan_effective_once_limit tblvar condition joinexpr tbl)
														(once_limit_scan_contract_promise_name scan_once_contract))))
												(define child_scan (build_scan tables effective_later_condition (list schema base_tbl tblvar) next_update_expr (coalesce pending_once_name scan_once_name)))
												(define scan_body (if is_update_target
													(list (list (symbol "lambda") (list (symbol "__dml_update_bound")) child_scan) (symbol "$update"))
													(if (nil? bound_update_expr) child_scan
														(list (list (symbol "lambda") (list (symbol "__dml_update_bound")) child_scan) bound_update_expr))))
												(if (or _tagged_scan (not (nil? _ps)))
													/* === table-local scan_order === */
													(begin
														(define _ps_filtercols (merge_unique (list (extract_columns_for_tblvar tblvar scan_now_condition) (extract_outer_columns_for_tblvar tblvar scan_now_condition))))
														(define _ps_order (if _tagged_scan tbl_scan_order (coalesceNil (stage_order_list _ps) '())))
														(define _ps_partcols (if (nil? scan_once_contract)
															scan_input_partcols
															(once_limit_scan_contract_partition_cols scan_once_contract)))
														(define _ps_limit (if (nil? scan_once_contract)
															scan_input_limit
															(once_limit_scan_contract_limit scan_once_contract)))
														(define _ps_offset (if (nil? scan_once_contract)
															scan_input_offset
															(once_limit_scan_contract_offset scan_once_contract)))
														(define _ps_ordercols (extract_scan_order_cols_for_tblvar _ps_order tblvar))
														(define _ps_dirs (extract_scan_order_dirs_for_tblvar _ps_order tblvar))
														/* emit init code from partition stage if present */
														(define _ps_init2 (if _tagged_scan nil (stage_init_code _ps)))
														(define _ps_scan_core (scan_wrapper 'scan_order schema base_tbl
															(cons list (merge_unique _ps_filtercols cols))
															'((quote lambda) (map (merge_unique _ps_filtercols cols) (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr_for_scan tblvar scan_now_condition)))
															(cons list _ps_ordercols)
															(cons list _ps_dirs)
															_ps_partcols _ps_offset _ps_limit
															scan_mapcols
															(list (symbol "lambda") scan_mapfn_params scan_body)
															nil nil isOuter))
														(define _ps_scan (wrap_once_limit_scan scan_once_name _ps_scan_core))
														(if (nil? _ps_init2) _ps_scan (list (quote begin) _ps_init2 _ps_scan)))
													/* === regular scan === */
													(scan_wrapper 'scan schema base_tbl
														(cons list filtercols)
														'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr_for_scan tblvar scan_now_condition)))
														scan_mapcols
														(list (symbol "lambda") scan_mapfn_params scan_body)
														(if is_update_target (symbol "+") nil)
														(if is_update_target 0 nil)
														nil
														isOuter
										))))
									))
								)
								'() /* final inner (=scalar) */ (if (nil? update_target)
									(begin
										(define emit_fields (if (nil? last_scan_ctx) fields
											(match last_scan_ctx
												'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar fields)
												fields)))
										(define emit_replace (if (nil? last_scan_ctx)
											replace_columns_from_expr
											(match last_scan_ctx
												'(_ _ scan_tblvar) (lambda (expr)
													(replace_columns_from_expr_for_scan scan_tblvar expr))
												replace_columns_from_expr)))
										(define result_body (list (symbol "resultrow") (runtime_heap_list_ast (map_assoc emit_fields (lambda (k v) (emit_replace v))))))
										(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body)))
									/* DML mode: emit mutation payload; actual DELETE/UPDATE runs in build_dml_plan's resultrow wrapper */
									(begin (define _ut_cols (nth update_target 1))
										(define _ut_tag (nth update_target 2))
										(define _ut_cols (if (nil? last_scan_ctx) _ut_cols
											(match last_scan_ctx
												'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar _ut_cols)
												_ut_cols)))
										(if (equal? _ut_cols '())
											/* DELETE */
											(begin
												(define result_body (list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" bound_update_expr "__values" nil)))
												(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body) 0))
											/* UPDATE */
											(begin
												(define result_body (list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" bound_update_expr "__values" (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v)))))))
												(list (quote if) (list (quote optimize) (replace_columns_from_expr condition)) (wrap_once_limit_body pending_once_name result_body) 0)))))
							)
						))
						(build_scan tables (replace_find_column condition) nil nil nil)
			)))
	)))
)))
/* _replace_table_with_sym: replaces (table schema name) AST nodes with tbl:schema:name symbols
for tables that have pre-resolved defines. tbl_map is an assoc list of (schema.name . symbol). */
(define _replace_table_with_sym (lambda (expr tbl_map)
	(if (not (list? expr)) expr
		(if (and (equal? (count expr) 3) (equal? (car expr) 'table) (string? (nth expr 1)) (string? (nth expr 2)))
			(begin
				(define key (concat (nth expr 1) ":" (nth expr 2)))
				(define sym (get_assoc tbl_map key))
				(if (nil? sym) expr sym))
			(map expr (lambda (e) (_replace_table_with_sym e tbl_map)))))))

/* build_queryplan: wraps _build_queryplan_inner with table-pointer pre-resolution */
(define build_queryplan (lambda (schema tables fields condition groups schemas replace_find_column update_target) (begin
	/* Collect base tables that can be pre-resolved */
	(define _tbl_entries (filter (map tables (lambda (td) (match td
		'(_ tschema tname _ _) (begin
			(define _base_tname (scan_tagged_table_base tname))
			(if (string? _base_tname) (list tschema _base_tname) nil))
		nil))) (lambda (d) (not (nil? d)))))
	(define _tbl_defines (map _tbl_entries (lambda (e) (tbl-define-code (car e) (car (cdr e))))))
	(define _tbl_map (merge (map _tbl_entries (lambda (e)
		(list (concat (car e) ":" (car (cdr e))) (symbol (concat "tbl:" (car e) ":" (car (cdr e)))))))))
	(define _plan (_build_queryplan_inner schema tables fields condition groups schemas replace_find_column update_target))
	(if (equal? _tbl_defines '()) _plan
		(begin
			/* Replace (table schema name) calls with pre-resolved symbols in the plan */
			(define _opt_plan (_replace_table_with_sym _plan _tbl_map))
			(cons '!begin (merge _tbl_defines (list _opt_plan)))))
)))
