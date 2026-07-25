/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Haensch

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
Neumann compiler rebuild
------------------------

The only legal planner pipeline is:

	parser AST -> untangle_query -> join_reorder -> build_queryplan

untangle_query is the Neumann/Kemper unnesting compiler.  It owns arbitrary
subqueries, including scalar subqueries, IN/EXISTS, nested dependent joins and
FROM (SELECT ...).  Its result must be a flat relational IR.  It must never emit
physical runtime scans, promises, materialized subquery sources, or scalar
fallback code.

join_reorder may only reorder an already-flat IR.

build_queryplan may only lower an already-flat, ordered IR to executable Scheme.
If build_queryplan ever sees subquery semantics, untangle_query is incomplete.

The previous master planner and the topdown-cleanup worktree remain quarry via
git/worktree history.  No legacy implementation is kept in this file.
*/

/* ------------------------------------------------------------------------- */
/* IR                                                                         */

/*
Canonical query IR:

(neumann-query
	(kind select|update|delete|insert)
	(schema schema-name)
	(sources ((alias schema source join-kind join-condition) ...))
	(fields ("name" expr ...))
	(predicate expr)
	(stages (stage ...))
	(return return-mode)
	(metadata assoc))

source is either a physical table name or a future logical operator produced by
FROM-subquery flattening.  It is never a runtime materialized-subquery handle.

Subquery unnesting adds relational sources/stages:
- dependent joins become ordinary joins plus domain columns D
- scalar cardinality becomes a relational cardinality stage, not a promise
- IN/EXISTS become semi/anti relational operators or aggregate stages
- GROUP/HAVING/ORDER/LIMIT become explicit stages
*/

(define ir_query (lambda (kind schema sources fields predicate stages return metadata)
	(list (quote neumann-query)
		(list (quote kind) kind)
		(list (quote schema) schema)
		(list (quote sources) (coalesceNil sources '()))
		(list (quote fields) (coalesceNil fields '()))
		(list (quote predicate) (coalesceNil predicate true))
		(list (quote stages) (coalesceNil stages '()))
		(list (quote return) return)
		(list (quote metadata) (coalesceNil metadata '())))))

(define ir_get (lambda (ir key default)
	(coalesceNil (get_assoc (cdr ir) key) default)))

(define ir_with (lambda (ir key value)
	(cons (car ir)
		(cons (list key value)
			(filter (cdr ir) (lambda (entry) (match entry
				'(k _) (not (equal? k key))
				true)))))))

(define ir_kind (lambda (ir) (ir_get ir (quote kind) nil)))
(define ir_schema (lambda (ir) (ir_get ir (quote schema) nil)))
(define ir_sources (lambda (ir) (ir_get ir (quote sources) '())))
(define ir_fields (lambda (ir) (ir_get ir (quote fields) '())))
(define ir_predicate (lambda (ir) (ir_get ir (quote predicate) true)))
(define ir_stages (lambda (ir) (ir_get ir (quote stages) '())))
(define ir_return (lambda (ir) (ir_get ir (quote return) nil)))
(define ir_metadata (lambda (ir) (ir_get ir (quote metadata) '())))

(define ir_stage (lambda (kind keys payload metadata)
	(list (quote neumann-stage)
		(list (quote kind) kind)
		(list (quote keys) (coalesceNil keys '()))
		(list (quote payload) (coalesceNil payload '()))
		(list (quote metadata) (coalesceNil metadata '())))))

(define ir_source (lambda (alias schema source join_kind join_condition)
	(list alias schema source join_kind (coalesceNil join_condition true))))

/* ------------------------------------------------------------------------- */
/* Subquery/fallback guards                                                   */

(define neumann_fail (lambda (where detail)
	(error (concat "NEUMANN_REBUILD_UNIMPLEMENTED: " where ": " detail))))

(define inner_select_head? (lambda (sym)
	(or
		(equal? sym (quote inner_select))
		(equal? sym '(quote inner_select))
		(equal? sym '(symbol inner_select))
		(equal? sym (quote inner_select_in))
		(equal? sym '(quote inner_select_in))
		(equal? sym '(symbol inner_select_in))
		(equal? sym (quote inner_select_exists))
		(equal? sym '(quote inner_select_exists))
		(equal? sym '(symbol inner_select_exists)))))

(define expr_contains_subquery? (lambda (expr) (match expr
	(cons sym args)
	(or
		(inner_select_head? sym)
		(expr_contains_subquery? sym)
		(reduce args (lambda (found arg)
			(or found (expr_contains_subquery? arg)))
			false))
	false)))

(define table_desc_contains_from_subquery? (lambda (td) (match td
	'(_ _ (string? _) _ _) false
	'(_ _ _ _ _) true
	false)))

(define select_ast_contains_subquery? (lambda (tables fields condition group having order)
	(or
		(reduce (coalesceNil tables '()) (lambda (found td)
			(or found (table_desc_contains_from_subquery? td)))
			false)
		(reduce_assoc (coalesceNil fields '()) (lambda (found _key expr)
			(or found (expr_contains_subquery? expr)))
			false)
		(expr_contains_subquery? condition)
		(reduce (coalesceNil group '()) (lambda (found expr)
			(or found (expr_contains_subquery? expr)))
			false)
		(expr_contains_subquery? having)
		(reduce (coalesceNil order '()) (lambda (found item)
			(or found (expr_contains_subquery? item)))
			false))))

(define ir_contains_forbidden_subquery? (lambda (ir)
	(or
		(reduce (ir_sources ir) (lambda (found source)
			(or found (table_desc_contains_from_subquery? source)))
			false)
		(reduce_assoc (ir_fields ir) (lambda (found _key expr)
			(or found (expr_contains_subquery? expr)))
			false)
		(expr_contains_subquery? (ir_predicate ir))
		(reduce (ir_stages ir) (lambda (found stage)
			(or found (expr_contains_subquery? stage)))
			false))))

(define require_flat_ir (lambda (where ir)
	(if (ir_contains_forbidden_subquery? ir)
		(error (concat "NEUMANN_INVARIANT_BROKEN: " where " produced subquery-bearing IR"))
		ir)))

/* ------------------------------------------------------------------------- */
/* untangle_query                                                             */

(define select_sources_from_parser_tables (lambda (tables)
	(map (coalesceNil tables '()) (lambda (td) (match td
		'(alias schema (string? tbl) is_outer join_expr)
		(ir_source alias schema tbl
			(if is_outer (quote left) (quote inner))
			join_expr)
		_ (neumann_fail "untangle_query" "FROM-subquery flattening not ported yet"))))))

(define select_stages_from_parser_clauses (lambda (group having order limit offset)
	(if (or
		(not (equal? (coalesceNil group '()) '()))
		(not (nil? having))
		(not (equal? (coalesceNil order '()) '()))
		(not (nil? limit))
		(not (nil? offset)))
		(list (ir_stage (quote select-stage) (coalesceNil group '())
			(list
				(list (quote having) having)
				(list (quote order) (coalesceNil order '()))
				(list (quote limit) limit)
				(list (quote offset) offset))
			'()))
		'())))

(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas)
	(if (select_ast_contains_subquery? tables fields condition group having order)
		(neumann_fail "untangle_query" "Neumann arbitrary-query unnesting not ported yet")
		(require_flat_ir "untangle_query"
			(ir_query
				(quote select)
				schema
				(select_sources_from_parser_tables tables)
				fields
				(coalesceNil condition true)
				(select_stages_from_parser_clauses group having order limit offset)
				(quote rows)
				(list
					(list (quote outer-schemas) (coalesceNil outer_schemas '()))))))))

(define untangle_dml (lambda (kind schema target_table target_alias tables fields condition order limit offset)
	(require_flat_ir "untangle_dml"
		(ir_with
			(untangle_query schema tables fields condition nil nil order limit offset nil)
			(quote return)
			(list kind target_table target_alias fields)))))

(define untangle_query_term (lambda (query outer_schemas) (match query
	'(schema tables fields condition group having order limit offset)
	(untangle_query schema tables fields condition group having order limit offset outer_schemas)
	_ (neumann_fail "untangle_query_term" "query term kind not ported yet"))))

/* ------------------------------------------------------------------------- */
/* reorder                                                                    */

(define join_reorder (lambda (ir)
	(require_flat_ir "join_reorder" ir)))

/* ------------------------------------------------------------------------- */
/* build_queryplan                                                            */

(define build_queryplan (lambda (ir)
	(require_flat_ir "build_queryplan input" ir)
	(neumann_fail "build_queryplan" "physical lowering not ported yet")))

(define neumann_compile_pipeline (lambda (ast)
	(build_queryplan
		(join_reorder
			(untangle_query_term ast nil)))))

(define neumann_compile_ir_pipeline (lambda (ir)
	(build_queryplan
		(join_reorder ir))))

/* ------------------------------------------------------------------------- */
/* Parser-facing adapters                                                     */

(define build_queryplan_term (lambda (query)
	(neumann_compile_pipeline query)))

(define build_queryplan_term_with_sink (lambda (query sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with (untangle_query_term query nil) (quote return) sink_mode))))

(define build_queryplan_term_from_logical (lambda (logical_ir)
	(neumann_compile_ir_pipeline logical_ir)))

(define build_queryplan_term_from_logical_with_sink (lambda (logical_ir sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with logical_ir (quote return) sink_mode))))

(define build_dml_plan (lambda (schema tbl tblalias all_defs cols condition order limit offset)
	(neumann_compile_ir_pipeline
		(untangle_dml
			(if (nil? cols) (quote delete) (quote update))
			schema tbl tblalias all_defs
			(if (nil? cols) (list "__dml_row" 1) cols)
			(coalesceNil condition true)
			order limit offset))))

(define sql_truncate (lambda (schema tbl)
	(build_dml_plan schema tbl nil
		(list (list tbl schema tbl false nil))
		nil true nil nil nil)))

(define explain_queryplan_ir (lambda (query)
	(list (quote resultrow)
		(list (quote list)
			"ir"
			(pretty_print (untangle_query_term query nil) (settings "ExplainWidth"))))))

(define explain_queryplan_reorder (lambda (query)
	(list (quote resultrow)
		(list (quote list)
			"reorder"
			(pretty_print
				(join_reorder (untangle_query_term query nil))
				(settings "ExplainWidth"))))))
