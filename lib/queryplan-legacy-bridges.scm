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
session-backed query-term materialization bridge. Callers stay responsible
for registering visible schema metadata. */
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
