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
Planner debug + EXPLAIN helpers.

Per Reflection 2026-05-04 finding F.2: tracing/EXPLAIN does not belong in the
compiler core. This module collects the planner debug session, the EXPLAIN IR
serialization, and the EXPLAIN REORDER inspection layer.

Loaded after queryplan.scm so untangle_query / join_reorder / build_queryplan
are visible to the EXPLAIN entry points. The planner_debug_record_scalar_event
hook in queryplan.scm resolves through lazy name binding inside lambda bodies.
*/

/* explain_emit_rows: render a list of column tuples back as resultrow forms */
(define explain_emit_rows (lambda (rows)
	(cons (quote begin) (map rows (lambda (row)
		(list (quote resultrow) (cons (quote list) row))))))
)

/* planner_debug_settings: opt-in toggles for compile-time tracing.
planner_debug_scalar_events captures the lowering decisions so EXPLAIN can
display them next to the plan root. */
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

(define explain_plan_root (lambda (plan)
	(match plan
		(cons sym _) (string sym)
		_ (string plan)
	)
))
(define explain_plan_root_with_scalar_debug (lambda (plan) (begin
	(define scalar_events (planner_debug_get_scalar_events))
	(if (equal? scalar_events '())
		(explain_plan_root plan)
		(concat (explain_plan_root plan) " scalar-events=" (serialize scalar_events))))))

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
	(define logical_term (untangle_query_term query nil))
	(if (strlike (serialize logical_term) "%inner_select_kind%")
		(error (concat "EXPLAIN_IR_LOGICAL_LEAK " (serialize logical_term)))
		nil)
	(match logical_term
		'(select_core_term schema tables fields condition groups schemas replace_find_column init) (begin
			(define _uq_7tuple (list schema tables fields condition groups schemas replace_find_column))
			(define _jr_result (apply join_reorder _uq_7tuple))
			(define _plan (apply build_queryplan (merge _jr_result (list nil))))
			(if (strlike (serialize _plan) "%inner_select_kind%")
				(error (concat "EXPLAIN_IR_PLAN_LEAK " (serialize _plan)))
				nil)
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
	(if (or (nil? query) (< (count query) 1) (not (string? (nth query 0))))
		(error (concat "EXPLAIN_REORDER_NONSTRING_QUERY_SCHEMA " (serialize query)))
		nil)
	(define _uq_result (apply untangle_query (merge query (list nil))))
	(if (not (equal? (count _uq_result) 8))
		(error (concat "EXPLAIN_REORDER_UQ_RESULT " (serialize _uq_result)))
		nil)
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
