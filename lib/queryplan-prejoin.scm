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
_unn_* get_column terms and their canonicalized forms onto the prejoin's actual
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
(define materialized_source_dependency_tables (newsession))
(define append_dependency_table_unique (lambda (acc dep_entry)
	(if (or (nil? dep_entry)
		(reduce acc (lambda (found existing)
			(or found (and
				(equal?? (nth existing 0) (nth dep_entry 0))
				(equal?? (nth existing 1) (nth dep_entry 1)))))
			false))
		acc
		(merge acc (list dep_entry)))
))
(define collect_materialized_query_dependency_tables (lambda (query)
	(match query
		'(_ dep_tables _ _ _ _ _ _ _)
		(reduce dep_tables (lambda (acc td) (match td
			'(_ tschema ttbl _ _)
			(begin
				(define normalized_tbl (normalize-materialized-subquery-source ttbl))
				(if (materialized-subquery-source? normalized_tbl)
					(reduce (coalesceNil (materialized_source_dependency_tables normalized_tbl) '())
						append_dependency_table_unique
						acc)
					(begin
						(define base_tbl (planner_table_source_base normalized_tbl))
						(if (or (nil? base_tbl) (not (string? base_tbl)) (strlike base_tbl ".%"))
							acc
							(append_dependency_table_unique acc (list tschema base_tbl))))))
			_ acc))
			'())
		_ '())
))
(define merge_schema_fields_unique (lambda (field_lists)
	(reduce (merge field_lists) (lambda (acc coldef)
		(if (reduce acc (lambda (found existing)
			(or found (equal?? (existing "Field") (coldef "Field"))))
			false)
			acc
			(merge acc (list coldef))))
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
		(define normalized_ttbl (normalize-materialized-subquery-source ttbl))
		(define alias_cols_raw (if (or (nil? alias) (not (has_assoc? schemas alias))) nil (schemas alias)))
		(define alias_cols (if (list? alias_cols_raw) alias_cols_raw '()))
		(define planned_cols (coalesceNil (planned_materialized_fields normalized_ttbl) '()))
		(merge_schema_fields_unique (list alias_cols planned_cols)))))
(define materialized_source_physical_schema (lambda (tschema ttbl alias schemas)
	(begin
		(define normalized_ttbl (normalize-materialized-subquery-source ttbl))
		(define planned_cols (coalesceNil (planned_materialized_fields normalized_ttbl) '()))
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
(define normalize_visible_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) false col false)
		'((quote get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) false col false)
		(cons sym args)
		(cons sym (map args normalize_visible_aliases))
		expr
	)
))
(define normalize_canonical_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias false col false)
			_ (list (quote get_column) alias_ false col false))
		'((quote get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias false col false)
			_ (list (quote get_column) alias_ false col false))
		(cons sym args)
		(cons sym (map args normalize_canonical_aliases))
		expr
	)
)))
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
(define materialized-subquery-key (lambda (id subquery)
	(concat "__mat:" id ":" (sha1 (string (normalize_canonical_aliases subquery))))))
(define make_materialized-subquery-source (lambda (session_key)
	(list (quote materialized-subquery-source) session_key)))
(define legacy-materialized-subquery-source-key (lambda (table-source)
	(match table-source
		(cons (cons (symbol context) '("session")) (cons key '())) key
		(cons (cons '(quote context) '("session")) (cons key '())) key
		nil
)))
(define materialized-subquery-source? (lambda (table-source)
	(match table-source
		'(materialized-subquery-source _) true
		'((symbol materialized-subquery-source) _) true
		'((quote materialized-subquery-source) _) true
		false
)))
(define normalize-materialized-subquery-source (lambda (table-source)
	(match table-source
		'(materialized-subquery-source key) (make_materialized-subquery-source key)
		'((symbol materialized-subquery-source) key) (make_materialized-subquery-source key)
		'((quote materialized-subquery-source) key) (make_materialized-subquery-source key)
		'(materialized-subquery key) (make_materialized-subquery-source key)
		'((symbol materialized-subquery) key) (make_materialized-subquery-source key)
		'((quote materialized-subquery) key) (make_materialized-subquery-source key)
		_ (begin
			(define legacy_key (legacy-materialized-subquery-source-key table-source))
			(if (nil? legacy_key)
				table-source
				(make_materialized-subquery-source legacy_key)))
)))
(define materialized-subquery-source-key (lambda (table-source)
	(match (normalize-materialized-subquery-source table-source)
		'(materialized-subquery-source key) key
		'((symbol materialized-subquery-source) key) key
		'((quote materialized-subquery-source) key) key
		nil
)))
(define materialized-subquery-source-runtime (lambda (table-source)
	(begin
		(define key (materialized-subquery-source-key table-source))
		(if (nil? key)
			nil
			(list (list (quote context) "session")
				(if (string? key)
					key
					(list (quote quote) key))))
)))
(define materialized-subquery-source (lambda (id subquery)
	(make_materialized-subquery-source (materialized-subquery-key id subquery))))
(define materialized-subquery-init (lambda (id subquery rows_expr)
	(list (list (quote context) "session") (materialized-subquery-key id subquery) rows_expr)))
/* planner_collect_rows_ast: execute inner_plan through a sink callback and
persist produced rows in a session list. Keep this as the fallback bridge for
runtime materialization paths that still operate outside the logical IR. */
(define planner_collect_rows_ast (lambda (rows_sym sink_sym item_sym inner_plan limit_val cnt_sym) (begin
	(define append_row_ast (list rows_sym "rows"
		(list (quote merge) (list rows_sym "rows") (list (quote list) item_sym))))
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
		(list rows_sym "rows")))))
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
			cnt_sym))
	(materialized_source_dependency_tables mat_source
		(collect_materialized_query_dependency_tables subquery))
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
				(set cols (merge_unique (list
					(extract_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map prejoin_columns (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map prejoin_columns (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
					(extract_later_joinexpr_columns_for_tblvar tblvar rest)
				)))
				(match (split_scan_condition isOuter joinexpr scan_condition rest) '(now_condition later_condition) (begin
					(set filtercols (merge_unique (list
						(extract_columns_for_tblvar tblvar now_condition)
						(extract_outer_columns_for_tblvar tblvar now_condition))))
					(scan_wrapper 'scan schema tbl
						(cons list filtercols)
						'((quote lambda) (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
						(cons list cols)
						'((quote lambda) (map cols (lambda (col) (symbol (concat tblvar "." col)))) (build_materialize_scan rest later_condition false))
						/* reduce: merge sub-results */
						'('lambda '('acc 'sub) '('merge 'acc 'sub))
						'(list)
						/* reduce2: outermost inserts into prejoin, inner levels merge */
						(if is_outermost
							'('lambda '('acc 'shard_rows) '('insert '('table prejoin_schema prejointbl) (cons 'list prejoin_column_names) 'shard_rows '(list) '('lambda '() true) true))
							'('lambda '('acc 'shard_rows) '('merge 'acc 'shard_rows)))
						isOuter
					)
				))
			)
			'() /* base case: produce one row wrapped in a list */
			'('if (optimize (replace_columns_from_expr (coalesceNil scan_condition true)))
				(list (quote list) (cons (quote list) (map prejoin_columns (lambda (mc) (replace_columns_from_expr (cadr mc))))))
				'(list))
		)
	))
	(define prejoin_materialize_fields (merge (map prejoin_columns (lambda (mc) (list (car mc) (cadr mc))))))
	(define prejoin_materialize_rowplan (build_queryplan schema
		prejoin_source_tables
		prejoin_materialize_fields
		raw_condition
		covered_partition_stages
		schemas
		replace_find_column
		nil))
	(define _pj_prev_rr (symbol "__pj_prev_resultrow"))
	(define _pj_row_sym (symbol "__pj_row"))
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
		(list 'set (symbol "resultrow") _pj_prev_rr)
	)
)))
/* register_prejoin_materialized_metadata: isolate the lineage/name registration
for prejoin-backed materialized sources. This keeps the prejoin assembly focused
on plan wiring while preserving the existing materialized-source contracts. The
caller still owns the prejoin-local canonicalizer that defines the visible
source-expression namespace for this materialized source. */
(define register_prejoin_materialized_metadata (lambda (canonicalize_prejoin_source_expr prejointbl prejoin_columns prejoin_alias_map prejoin_source_tables prejoin_schema_def) (begin
	(define _td_alias_variants (lambda (tv tschema ttbl) (begin
		(define _raw_aliases (filter (list
			tv
			(match tv '(visible _) visible nil)
			(visible_occurrence_alias tv)
			(coalesce (resolve_source_alias prejoin_alias_map tv) nil)
			(if (equal? (visible_occurrence_alias tv) ttbl) (concat tschema "." ttbl) nil))
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
(define make_unnest_helper_table (lambda (schema_name base_table helper_kind)
	(if (nil? base_table)
		nil
		(list (quote unnest_helper_table) schema_name base_table helper_kind))
))
(define planner_table_source_base (lambda (table_source)
	(match table_source
		'(unnest_helper_table _ base_table _) (planner_table_source_base base_table)
		'((symbol unnest_helper_table) _ base_table _) (planner_table_source_base base_table)
		'((quote unnest_helper_table) _ base_table _) (planner_table_source_base base_table)
		'(scan-tagged-table base_table _ _ _ _ _) (planner_table_source_base base_table)
		'(scan-tagged-table base_table _ _ _ _ _ _) (planner_table_source_base base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _) (planner_table_source_base base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _ _) (planner_table_source_base base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _) (planner_table_source_base base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _ _) (planner_table_source_base base_table)
		table_source
)))
/* Planner-only wrapper: if it survives into emitted runtime code, it must stay
semantically neutral and evaluate to the underlying source. */
(define unnest_helper_table (lambda (schema_name base_table _helper_kind)
	(begin
		(define source_base (planner_table_source_base base_table))
		(if (string? source_base)
			(table schema_name source_base)
			source_base))
))
(define unnest_helper_table? (lambda (table_source)
	(match table_source
		'(unnest_helper_table _ _ _) true
		'((symbol unnest_helper_table) _ _ _) true
		'((quote unnest_helper_table) _ _ _) true
		false
)))
(define unnest_helper_table_base (lambda (table_source)
	(match table_source
		'(unnest_helper_table _ base_table _) (planner_table_source_base base_table)
		'((symbol unnest_helper_table) _ base_table _) (planner_table_source_base base_table)
		'((quote unnest_helper_table) _ base_table _) (planner_table_source_base base_table)
		table_source
)))
(define unnest_helper_table_schema (lambda (table_source)
	(match table_source
		'(unnest_helper_table schema_name _ _) schema_name
		'((symbol unnest_helper_table) schema_name _ _) schema_name
		'((quote unnest_helper_table) schema_name _ _) schema_name
		nil
)))
(define unnest_helper_table_runtime_source (lambda (table_source)
	(match table_source
		'(unnest_helper_table schema_name base_table _)
		(unnest_helper_table schema_name base_table nil)
		'((symbol unnest_helper_table) schema_name base_table _)
		(unnest_helper_table schema_name base_table nil)
		'((quote unnest_helper_table) schema_name base_table _)
		(unnest_helper_table schema_name base_table nil)
		table_source
)))
(define planner_runtime_table_source (lambda (schema table_source)
	(match (normalize-materialized-subquery-source table_source)
		'(unnest_helper_table _ base_table _)
		(planner_runtime_table_source schema base_table)
		'((symbol unnest_helper_table) _ base_table _)
		(planner_runtime_table_source schema base_table)
		'((quote unnest_helper_table) _ base_table _)
		(planner_runtime_table_source schema base_table)
		'(scan-tagged-table base_table _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'(scan-tagged-table base_table _ _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _ _)
		(planner_runtime_table_source schema base_table)
		'(materialized-subquery-source key) ((context "session") key)
		'((symbol materialized-subquery-source) key) ((context "session") key)
		'((quote materialized-subquery-source) key) ((context "session") key)
		'((context "session") key) ((context "session") key)
		'(((symbol context) "session") key) (((symbol context) "session") key)
		'(((quote context) "session") key) (((quote context) "session") key)
		'(table helper_schema helper_tbl) (table helper_schema helper_tbl)
		'((symbol table) helper_schema helper_tbl) (table helper_schema helper_tbl)
		'((quote table) helper_schema helper_tbl) (table helper_schema helper_tbl)
		(string? base_tbl) (table schema base_tbl)
		base_tbl base_tbl)))
(define planner_codegen_table_source (lambda (schema table_source)
	(match (normalize-materialized-subquery-source table_source)
		'(unnest_helper_table _ base_table _)
		(planner_codegen_table_source schema base_table)
		'((symbol unnest_helper_table) _ base_table _)
		(planner_codegen_table_source schema base_table)
		'((quote unnest_helper_table) _ base_table _)
		(planner_codegen_table_source schema base_table)
		'(scan-tagged-table base_table _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'(scan-tagged-table base_table _ _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'((symbol scan-tagged-table) base_table _ _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'((quote scan-tagged-table) base_table _ _ _ _ _ _)
		(planner_codegen_table_source schema base_table)
		'(materialized-subquery-source key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'((symbol materialized-subquery-source) key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'((quote materialized-subquery-source) key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'((context "session") key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'(((symbol context) "session") key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'(((quote context) "session") key) (list (list (quote context) "session")
			(if (string? key) key (list (quote quote) key)))
		'(table helper_schema helper_tbl) (list (quote table) helper_schema helper_tbl)
		'((symbol table) helper_schema helper_tbl) (list (quote table) helper_schema helper_tbl)
		'((quote table) helper_schema helper_tbl) (list (quote table) helper_schema helper_tbl)
		(string? base_tbl) (list (quote table) schema base_tbl)
		base_tbl base_tbl)))
(define unnest_helper_table_kind (lambda (table_source)
	(match table_source
		'(unnest_helper_table _ _ helper_kind) helper_kind
		'((symbol unnest_helper_table) _ _ helper_kind) helper_kind
		'((quote unnest_helper_table) _ _ helper_kind) helper_kind
		nil
)))
(define materialized-source? (lambda (table-source)
	(or
		(if (unnest_helper_table? table-source)
			(materialized-source? (unnest_helper_table_base table-source))
			false)
		(begin
			(define normalized_source (normalize-materialized-subquery-source table-source))
			(or
				(and (string? normalized_source) (>= (strlen normalized_source) 1) (equal? (substr normalized_source 0 1) "."))
				(materialized-subquery-source? normalized_source)
				(match normalized_source
					(cons (cons (symbol context) '("session")) _) true
					(cons (cons '(quote context) '("session")) _) true
					false
	)))))
))
(define planner-temp-source-name (lambda (tbl tblvar)
	(define base_tbl (normalize-materialized-subquery-source
		(if (unnest_helper_table? tbl) (unnest_helper_table_base tbl) tbl)))
	(if (string? base_tbl)
		base_tbl
		(if (materialized-source? base_tbl)
			(concat "mat_" (fnv_hash (string base_tbl)))
			(string tblvar)))))
