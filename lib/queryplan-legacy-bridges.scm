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
Legacy runtime bridges (compatibility module).

Per Reflection 2026-05-04 finding F.1: planner_collect_rows_ast,
legacy_materialized_query_term_binding_ast and build_legacy_prejoin_materialize_plan
are session/resultrow-backed bridges that pre-date the structural
materialized-subquery-source wrapper. They are kept here as compat
helpers so the compiler core (lib/queryplan.scm) and the structural
backend (lib/queryplan-prejoin.scm) stay free of session-rows plumbing.

Loaded between queryplan-prejoin.scm (provides materialized-subquery-key /
-source / -init) and queryplan.scm (call sites). Names referenced inside
the lambda bodies (build_queryplan_term_with_sink, extract_columns_for_tblvar,
split_scan_condition, scan_wrapper, etc.) are resolved at call time through
lazy lambda-body name binding.

These helpers must NOT be extended for new planner work. Future
materialization needs go through the structural materialized-subquery-source
wrapper or, for FAQ §18-allowed group-cache / window-conflicting-order paths,
through the dedicated keytable / window-stage emitters.
*/

/* planner_collect_rows_ast: execute inner_plan through a sink callback and
persist produced rows in a session list. */
(define legacy_materialized_scalar_source_id? (lambda (id)
	(and (string? id)
		(>= (strlen id) 3)
		(or
			(equal? (substr id 0 3) "sq_")
			(equal? (substr id 0 3) "nq_"))
		(not (strlike id "%:%")))))

(define legacy_materialized_default_row_for_empty (lambda (id subquery) (match subquery
	'(schema tables fields condition group having order limit offset)
	(begin
		(define synthetic-single-row (and
			(equal? (count (coalesceNil tables '())) 1)
			(match (nth tables 0)
				'(_ _ tname _ _) (equal? tname ".(1)")
				false)))
		(define no-filter (or (nil? condition) (equal? condition true) (equal? condition (quote true))))
		(define no-group (or (nil? group) (equal? group '()) (and (list? group) (equal? (count group) 0))))
		(define scalar-static-domain (or no-group (equal? group '(1))))
		(if (or
			(and (legacy_materialized_scalar_source_id? id) scalar-static-domain)
			(and synthetic-single-row no-filter no-group (nil? having)))
			(merge (map (qpp-fields-to-pairs fields) (lambda (pair) (match pair
				'(name _expr) (list name nil)
				_ '()))))
			nil))
	_ nil)))

(define legacy_materialized_table_backed_id? (lambda (id)
	(and (string? id)
		(>= (strlen id) 3)
		(equal? (substr id 0 3) "nq_"))))

(define legacy_materialized_table_backed_binding_ast (lambda (id subquery sink_sym limit_val) (begin
	(if (not (nil? limit_val))
		nil
		(match subquery
			'(schema _tables fields _condition _group _having _order _limit _offset)
			(begin
				(define field_pairs (reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
					'(name _expr)
					(if (reduce acc (lambda (found existing_pair)
						(or found (equal?? (nth existing_pair 0) name)))
						false)
						acc
						(merge acc (list pair)))
					_ acc))
					'()))
				(define field_names (map field_pairs car))
				(if (equal? field_names '())
					nil
					(begin
						(define mat_key_base (string (normalize_canonical_aliases subquery)))
						(define mat_tbl (concat ".mat:" (fnv_hash (if (expr_uses_session_state subquery)
							(concat
								id ":"
								mat_key_base
								(user_session_runtime_cache_suffix_from_exprs (list subquery))
								(planner_current_user_session_snapshot_suffix))
							mat_key_base))))
						(define cache_scope (planner_analysis_cache_current))
						(define cache_key (concat "legacy-table-mat-binding:" schema ":" mat_tbl))
						(define cached_binding (if (nil? cache_scope) nil (cache_scope cache_key)))
						(if (not (nil? cached_binding))
							cached_binding
							(begin
								(define item_sym (symbol "__mat_item"))
								(define row_values
									(cons (quote list)
										(map field_names (lambda (col)
											(list (quote get_assoc) item_sym col)))))
								(define columns_code
									(cons (quote list)
										(map field_names (lambda (col)
											(list (quote list) "column" col "any" (list (quote list)) (list (quote list)))))))
								(define insert_row
									(list (quote insert)
										(list (quote table) schema mat_tbl)
										(cons (quote list) field_names)
										(list (quote list) row_values)
										(list (quote list))
										(list (quote lambda) (list) true)
										true))
								(define default_row (legacy_materialized_default_row_for_empty id subquery))
								(define default_insert
									(if (nil? default_row)
										nil
										(begin
											(define default_values
												(cons (quote list)
													(map field_names (lambda (col)
														(list (quote get_assoc) (list (quote quote) default_row) col)))))
											(list (quote if)
												(list (quote table_empty?) (list (quote table) schema mat_tbl))
												(list (quote insert)
													(list (quote table) schema mat_tbl)
													(cons (quote list) field_names)
													(list (quote list) default_values)
													(list (quote list))
													(list (quote lambda) (list) true)
													true)
												nil))))
									(define inner_plan
										(build_queryplan_term_with_sink subquery (list (quote callback) sink_sym)))
									(define mat_dependency_tables
										(collect_materialized_query_dependency_tables subquery))
								(define mat_dependency_invalidations
									(filter (map mat_dependency_tables (lambda (dep_td)
										(match dep_td '(dep_schema dep_tbl)
											(list (quote register_prejoin_invalidation) dep_schema dep_tbl schema mat_tbl)
											_ nil)))
										(lambda (x) (not (nil? x)))))
								(define init_code
									(cons (quote begin)
										(merge
											(list
												(list (quote createtable) schema mat_tbl columns_code query_temp_table_options_code true)
												(tbl-define-code schema mat_tbl))
											(merge mat_dependency_invalidations
												(list
													(list (quote if)
														(list (quote table_empty?) (list (quote table) schema mat_tbl))
														(list (quote begin)
															(list (quote define) sink_sym
																(list (quote lambda) (list item_sym) insert_row))
															inner_plan
															default_insert)
														nil))))))
								(planned_materialized_fields mat_tbl
									(map field_names (lambda (col)
										(list "Field" col "Type" "any"))))
								(materialized_source_dependency_tables mat_tbl
									mat_dependency_tables)
								(define binding (list mat_tbl init_code))
								(if (nil? cache_scope) nil (cache_scope cache_key binding))
								binding)))))
			_ nil)
))))

(define planner_collect_rows_ast (lambda (rows_sym sink_sym item_sym inner_plan limit_val cnt_sym default_row) (begin
	(define append_row_ast (list rows_sym "rows"
		(list (quote merge) (list rows_sym "rows") (list (quote list) item_sym))))
	(list (quote begin)
		(list (quote set) rows_sym (list (quote newsession)))
		(list rows_sym "rows" (list (quote list)))
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
		(if (nil? default_row)
			nil
			(list (quote if)
				(list (quote equal?) (list rows_sym "rows") (list (quote quote) '()))
				(list rows_sym "rows" (list (quote list) (list (quote quote) default_row)))
				nil))
		(list rows_sym "rows")))))

/* legacy_materialized_query_term_binding_ast: centralize the remaining
session-backed query-term materialization bridge. Callers stay responsible
for registering visible schema metadata. */
(define legacy_materialized_query_term_binding_ast (lambda (id subquery rows_sym sink_sym limit_val cnt_sym) (begin
	(define build_sink_plan (lambda ()
		(if (or
			(and (list? subquery) (> (count subquery) 0)
				(equal? (car subquery) (quote select_core_term)))
			(and (list? subquery) (> (count subquery) 0)
				(or
					(equal? (car subquery) (quote union_all_term))
					(equal? (car subquery) (quote union_distinct_term)))))
			(build_queryplan_term_from_logical_with_sink subquery (list (quote callback) sink_sym))
			(build_queryplan_term_with_sink subquery (list (quote callback) sink_sym)))))
	(if (legacy_materialized_table_backed_id? id)
		(coalesce
			(legacy_materialized_table_backed_binding_ast id subquery sink_sym limit_val)
			(materialized_query_term_binding_ast_from_sink_plan id subquery rows_sym sink_sym limit_val cnt_sym
				(build_sink_plan)))
		(materialized_query_term_binding_ast_from_sink_plan id subquery rows_sym sink_sym limit_val cnt_sym
			(build_sink_plan)))
)))

/* materialized_query_term_binding_ast_from_sink_plan: session-backed
materialization when the caller already compiled a callback-sink plan. */
(define materialized_query_term_binding_ast_from_sink_plan (lambda (id subquery rows_sym sink_sym limit_val cnt_sym sinked_inner_plan) (begin
	(define materialized_rows
		(planner_collect_rows_ast rows_sym sink_sym (symbol "item")
			sinked_inner_plan
			limit_val
			cnt_sym
			(legacy_materialized_default_row_for_empty id subquery)))
	(define runtime_id
		(concat
			id
			(user_session_runtime_cache_suffix_from_exprs (list subquery materialized_rows))))
		(define mat_source (materialized-subquery-source runtime_id subquery))
		(materialized_source_dependency_tables mat_source
			(collect_materialized_query_dependency_tables subquery))
	(list
		mat_source
		(materialized-subquery-init runtime_id subquery materialized_rows))
)))

/* dep_helper_keytable_binding: FAQ §32-compliant replacement for
legacy_materialized_query_term_binding_ast in the dep_helper code paths.

Produces a (kt_table_spec init_code) binding where:
- kt_table_spec: the keytable name (string) — usable as a regular table in scans
- init_code: createtable + createcolumn-per-aggregate, FAQ §31 canonical naming

Returns nil if the subquery shape is not yet supported by the keytable
backend; caller falls back to legacy_materialized_query_term_binding_ast.

Multi-session implementation plan (see memory: keytable_refactoring_spec.md (via Claude memory)):
- Step 1 (this commit): skeleton returns nil unconditionally (pure addition,
no behavior change). Ensures the function exists for the call sites to
feature-flag against without breakage.
- Step 2: implement simple-value-aggregate case (SUM/COUNT/MAX/MIN with
single-column domain D). Behind MEMCP_KEYTABLE_DEP_HELPER env flag.
- Step 3+: extend coverage, then enable by default, then remove legacy.

This is FAQ §32-compliant because the keytable IS a group cache (the
permitted form of materialization); cache columns are canonical
make_aggregate_cache_col_name (FAQ §31) which already disambiguates by
filter condition — solving the multi-scalar contamination case
(66_neumann_domain_col "multiple SUM with different domain joins") by
construction. */
(define dep_helper_keytable_binding (lambda (id subquery rows_sym sink_sym limit_val cnt_sym) (begin
	/* Dep-helper materialization must not go through session rows: the helper
	is a planned relation that the outer query joins repeatedly. Keep the
	physical representation table-backed so EXPLAIN/SELECT and sibling helpers
	can share normal table metadata and storage caches. */
	(if (not (nil? limit_val))
		nil
		(match subquery
			'(schema _tables fields _condition _group _having _order _limit _offset)
			(begin
				(define field_pairs (reduce (qpp-fields-to-pairs fields) (lambda (acc pair) (match pair
					'(name _expr)
					(if (reduce acc (lambda (found existing_pair)
						(or found (equal?? (nth existing_pair 0) name)))
						false)
						acc
						(merge acc (list pair)))
					_ acc))
					'()))
				(define field_names (map field_pairs car))
				(if (equal? field_names '())
					nil
					(begin
						(define helper_tbl (concat ".dephelper:" (fnv_hash id)))
						(define item_sym (symbol "__dep_helper_item"))
						(define row_values
							(cons (quote list)
								(map field_names (lambda (col)
									(list (quote get_assoc) item_sym col)))))
						(define columns_code
							(cons (quote list)
								(map field_names (lambda (col)
									(list (quote list) "column" col "any" (list (quote list)) (list (quote list)))))))
						(define insert_row
							(list (quote insert)
								(list (quote table) schema helper_tbl)
								(cons (quote list) field_names)
								(list (quote list) row_values)
								(list (quote list))
								(list (quote lambda) (list) true)
								true))
						(define inner_plan
							(build_queryplan_term_with_sink subquery (list (quote callback) sink_sym)))
						(define init_code
							(list (quote begin)
								(list (quote createtable) schema helper_tbl columns_code query_temp_table_options_code true)
								(tbl-define-code schema helper_tbl)
								(list (quote if)
									(list (quote table_empty?) (list (quote table) schema helper_tbl))
									(list (quote begin)
										(list (quote define) sink_sym
											(list (quote lambda) (list item_sym) insert_row))
										inner_plan)
									nil)))
						(planned_materialized_fields helper_tbl
							(map field_names (lambda (col)
								(list "Field" col "Type" "any"))))
						(list helper_tbl init_code))))
			_ nil)
))))

/* build_legacy_prejoin_materialize_plan: isolate the remaining
session/resultrow-backed prejoin filler used by trigger backfill paths.
Query-time prejoin filling stays on the canonical build_queryplan row
stream — this bridge only exists for the trigger-side backfill path. */
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
	(define pj_prev_rr (symbol "__pj_prev_resultrow"))
	(define pj_row_sym (symbol "__pj_row"))
	(list 'begin
		(list 'set pj_prev_rr (symbol "resultrow"))
		(list 'set (symbol "resultrow")
			(list 'lambda (list pj_row_sym)
				(list 'insert (list 'table prejoin_schema prejointbl)
					(cons 'list prejoin_column_names)
					(list 'list
						(cons 'list (map prejoin_column_names (lambda (col)
							(list 'get_assoc pj_row_sym col)))))
					(list)
					(list 'lambda (list) true)
					true)))
		prejoin_materialize_rowplan
		(list 'set (symbol "resultrow") pj_prev_rr)
	)
)))
