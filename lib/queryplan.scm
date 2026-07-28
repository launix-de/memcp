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
Neumann compiler rebuild scaffold
---------------------------------

The logical compiler IR intentionally uses few combined operators:

	query-block  select/join/filter/project/order/limit work unit
	group-stage  domain-D/keytable/aggregate/EXISTS/HAVING work unit
	union-block  set operator work unit

There is no logical scan operator.  Physical scans, bounds, indexes,
materialisation and fused loops are decisions of build_queryplan after
untangle_query and join_reorder have produced a decorrelated logical program.
*/

/* ------------------------------------------------------------------------- */
/* Small assoc helpers                                                        */

(define qassoc_get (lambda (xs key default)
	(coalesceNil
		(reduce (coalesceNil xs '()) (lambda (found entry)
			(if (not (nil? found))
				found
				(match entry
					(cons k rest) (if (equal? k key)
						(if (equal? (count rest) 1) (car rest) rest)
						nil)
					nil)))
			nil)
		default)))

(define qassoc_set (lambda (xs key value)
	(cons (list key value)
		(filter (coalesceNil xs '()) (lambda (entry) (match entry
			(cons k _) (not (equal? k key))
			true))))))

(define empty_list? (lambda (xs)
	(or (nil? xs) (equal? xs '()))))

/* ------------------------------------------------------------------------- */
/* Combined logical operators                                                 */

(define make_query_block (lambda (schema sources fields where group having order limit offset hidden stages facts)
	(list (quote query-block)
		schema
		(coalesceNil sources '())
		(coalesceNil fields '())
		(coalesceNil where true)
		group
		having
		order
		limit
		offset
		(coalesceNil hidden '())
		(coalesceNil stages '())
		(coalesceNil facts '()))))

(define make_group_stage (lambda (id input domain keys aggregates having output order limit offset facts)
	(list (quote group-stage)
		id input
		(coalesceNil domain '())
		(coalesceNil keys '())
		(coalesceNil aggregates '())
		having
		(coalesceNil output '())
		(coalesceNil order '())
		limit
		offset
		(coalesceNil facts '()))))

(define make_union_block (lambda (mode branches order limit offset facts)
	(list (quote union-block)
		mode
		(coalesceNil branches '())
		order limit offset
		(coalesceNil facts '()))))

(define logical_op (lambda (node) (if (and (list? node) (not (nil? node))) (car node) nil)))
(define query_block? (lambda (node) (equal? (logical_op node) (quote query-block))))
(define group_stage? (lambda (node) (equal? (logical_op node) (quote group-stage))))
(define union_block? (lambda (node) (equal? (logical_op node) (quote union-block))))
(define orc_stage? (lambda (node) (equal? (logical_op node) (quote orc-stage))))

(define qb_schema (lambda (node) (nth node 1)))
(define qb_sources (lambda (node) (nth node 2)))
(define qb_fields (lambda (node) (nth node 3)))
(define qb_where (lambda (node) (nth node 4)))
(define qb_group (lambda (node) (nth node 5)))
(define qb_having (lambda (node) (nth node 6)))
(define qb_order (lambda (node) (nth node 7)))
(define qb_limit (lambda (node) (nth node 8)))
(define qb_offset (lambda (node) (nth node 9)))
(define qb_hidden (lambda (node) (nth node 10)))
(define qb_stages (lambda (node) (nth node 11)))
(define qb_facts (lambda (node) (nth node 12)))

(define union_mode (lambda (node) (nth node 1)))
(define union_branches (lambda (node) (nth node 2)))
(define union_order (lambda (node) (nth node 3)))
(define union_limit (lambda (node) (nth node 4)))
(define union_offset (lambda (node) (nth node 5)))
(define union_facts (lambda (node) (nth node 6)))

(define gs_id (lambda (node) (nth node 1)))
(define gs_input (lambda (node) (nth node 2)))
(define gs_domain (lambda (node) (nth node 3)))
(define gs_keys (lambda (node) (nth node 4)))
(define gs_aggregates (lambda (node) (nth node 5)))
(define gs_having (lambda (node) (nth node 6)))
(define gs_output (lambda (node) (nth node 7)))
(define gs_order (lambda (node) (nth node 8)))
(define gs_limit (lambda (node) (nth node 9)))
(define gs_offset (lambda (node) (nth node 10)))
(define gs_facts (lambda (node) (nth node 11)))

(define make_orc_stage (lambda (id source column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit)
	(list (quote orc-stage)
		id source column
		(coalesceNil sortcols '())
		(coalesceNil sortdirs '())
		(coalesceNil partitioncount 0)
		(coalesceNil mapcols '())
		mapfn
		reducefn
		reduceinit)))

(define os_id (lambda (node) (nth node 1)))
(define os_source (lambda (node) (nth node 2)))
(define os_column (lambda (node) (nth node 3)))
(define os_sortcols (lambda (node) (nth node 4)))
(define os_sortdirs (lambda (node) (nth node 5)))
(define os_partitioncount (lambda (node) (nth node 6)))
(define os_mapcols (lambda (node) (nth node 7)))
(define os_mapfn (lambda (node) (nth node 8)))
(define os_reducefn (lambda (node) (nth node 9)))
(define os_reduceinit (lambda (node) (nth node 10)))

(define source_alias (lambda (src)
	(match src '(alias _schema _relation _outer _join) alias _ nil)))
(define source_schema (lambda (src)
	(match src '(_alias schema _relation _outer _join) schema _ nil)))
(define source_relation (lambda (src)
	(match src '(_alias _schema relation _outer _join) relation _ nil)))
(define source_outer? (lambda (src)
	(match src '(_alias _schema _relation outer _join) outer _ nil)))
(define source_join_expr (lambda (src)
	(match src '(_alias _schema _relation _outer join) join _ nil)))

/* ------------------------------------------------------------------------- */
/* IR envelope                                                                */

(define make_uctx (lambda (parent attrs)
	(list (quote uctx) parent (coalesceNil attrs '()))))

(define uctx_get (lambda (ctx key default)
	(match ctx
		((symbol uctx) parent attrs) (coalesceNil (qassoc_get attrs key nil)
			(if (nil? parent) default (uctx_get parent key default)))
		_ default)))

(define make_ir (lambda (kind root stages context return_mode)
	(list (quote neumann-ir)
		kind
		root
		(coalesceNil stages '())
		context
		(coalesceNil return_mode (quote rows)))))

(define neumann_ir? (lambda (ir) (equal? (logical_op ir) (quote neumann-ir))))
(define ir_kind (lambda (ir) (nth ir 1)))
(define ir_root (lambda (ir) (nth ir 2)))
(define ir_stages (lambda (ir) (nth ir 3)))
(define ir_context_of (lambda (ir) (nth ir 4)))
(define ir_return (lambda (ir) (nth ir 5)))
(define ir_context_get uctx_get)
(define ir_output_fields (lambda (ir)
	(if (query_block? (ir_root ir)) (qb_fields (ir_root ir)) '())))
(define ir_hidden_fields (lambda (ir)
	(if (query_block? (ir_root ir)) (qb_hidden (ir_root ir)) '())))

(define ir_with_return (lambda (ir return_mode)
	(make_ir (ir_kind ir) (ir_root ir) (ir_stages ir) (ir_context_of ir) return_mode)))

/* ------------------------------------------------------------------------- */
/* Normalisation                                                              */

(define normalize_query_ast (lambda (query)
	(match query
		((symbol query-block) schema sources fields where group having order limit offset hidden stages facts)
			query
		((symbol union-block) mode branches order limit offset facts)
			query
		((symbol union_all) branches order limit offset)
			(make_union_block (quote all) branches order limit offset '())
		((symbol union_distinct) branches order limit offset)
			(make_union_block (quote distinct) branches order limit offset '())
		'(schema sources fields where group having order limit offset)
			(make_query_block schema sources fields where group having order limit offset '() '() '())
		_ query)))

(define query_block_no_from? (lambda (node)
	(and (query_block? node)
		(empty_list? (qb_sources node))
		(empty_list? (qb_group node))
		(nil? (qb_having node))
		(empty_list? (qb_order node))
		(nil? (qb_limit node))
		(nil? (qb_offset node)))))

(define query_block_first_expr (lambda (node)
	(match (qb_fields node)
		(cons _title (cons expr _rest)) expr
		_ nil)))

(define split_and_terms (lambda (expr)
	(match (coalesceNil expr true)
		((symbol and) a b) (merge (list (split_and_terms a) (split_and_terms b)))
		((quote and) a b) (merge (list (split_and_terms a) (split_and_terms b)))
		_ (list expr))))

(define neumann_fail (lambda (phase msg)
	(error (concat phase ": " msg))))

/* ------------------------------------------------------------------------- */
/* Subquery detection and zero-domain rewrites                                */

(define subquery_head? (lambda (head)
	(or (equal? head (quote inner_select))
		(or (equal? head (quote inner_select_in))
			(equal? head (quote inner_select_exists))))))

(define expr_contains_subquery? (lambda (expr)
	(match expr
		(cons head tail) (or
			(subquery_head? head)
			(reduce tail (lambda (a b) (or a (expr_contains_subquery? b))) false))
		_ false)))

(define fields_contains_subquery? (lambda (fields)
	(reduce (coalesceNil fields '()) (lambda (found item)
		(or found (expr_contains_subquery? item))) false)))

(define query_contains_subquery? (lambda (node)
	(if (query_block? node)
		(or (fields_contains_subquery? (qb_fields node))
			(or (expr_contains_subquery? (qb_where node))
				(or (fields_contains_subquery? (qb_hidden node))
					(or (fields_contains_subquery? (qb_group node))
						(or (expr_contains_subquery? (qb_having node))
							(fields_contains_subquery? (qb_order node)))))))
		(if (union_block? node)
			(reduce (union_branches node) (lambda (a b) (or a (query_contains_subquery? b))) false)
			false))))

(define untangle_zero_domain_subquery (lambda (kind probe subquery ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define inner (untangle_query normalized ctx))
		(if (not (query_block_no_from? inner))
			(neumann_fail "untangle_query" "table-backed subquery unnesting must become group-stage(D) before lowering")
			(begin
				(define where (coalesceNil (qb_where inner) true))
				(match kind
					(symbol inner_select) (if (equal? where true)
						(query_block_first_expr inner)
						(list (quote if) where (query_block_first_expr inner) nil))
					(symbol inner_select_exists) where
					(symbol inner_select_in) (list (quote and) where (list (quote equal??) probe (query_block_first_expr inner)))
					_ (neumann_fail "untangle_query" "unknown subquery expression")))))))

(define source_aliases (lambda (sources)
	(map (coalesceNil sources '()) source_alias)))

(define expr_refs_alias? (lambda (default_alias alias expr)
	(match expr
		((symbol get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		((quote get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		(cons _head tail) (reduce tail (lambda (found item) (or found (expr_refs_alias? default_alias alias item))) false)
		_ false)))

(define expr_refs_any_alias? (lambda (default_alias aliases expr)
	(reduce (coalesceNil aliases '()) (lambda (found alias)
		(or found (expr_refs_alias? default_alias alias expr))) false)))

(define direct_column_ref? (lambda (expr)
	(match expr
		((symbol get_column) _ _ _ _) true
		((quote get_column) _ _ _ _) true
		_ false)))

(define exists_correlation_pair (lambda (inner_default inner_aliases outer_aliases term)
	(match term
		((symbol equal??) a b) (begin
			(define a_inner (expr_refs_any_alias? inner_default inner_aliases a))
			(define b_inner (expr_refs_any_alias? inner_default inner_aliases b))
			(define a_outer (and (not a_inner) (expr_refs_any_alias? nil outer_aliases a)))
			(define b_outer (and (not b_inner) (expr_refs_any_alias? nil outer_aliases b)))
			(if (and a_inner b_outer)
				(list a b)
				(if (and b_inner a_outer)
					(list b a)
					nil)))
		((quote equal??) a b) (exists_correlation_pair inner_default inner_aliases outer_aliases (list (quote equal??) a b))
		((symbol equal?) a b) (exists_correlation_pair inner_default inner_aliases outer_aliases (list (quote equal??) a b))
		((quote equal?) a b) (exists_correlation_pair inner_default inner_aliases outer_aliases (list (quote equal??) a b))
		_ nil)))

(define exists_stage_alias (lambda (stage_id)
	(concat "__exists_" (fnv_hash stage_id))))

(define exists_stage_table_name (lambda (stage)
	(begin
		(define src (gs_input stage))
		(group_table_name
			(source_schema src)
			(source_relation src)
			(source_alias src)
			(if (empty_list? (gs_keys stage)) '(1) (gs_keys stage))
			(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)
			(gs_aggregates stage)))))

(define make_exists_stage_join_condition (lambda (stage_alias key_names outer_domain)
	(if (empty_list? outer_domain)
		nil
		(combine_where_terms
			(map (produceN (count outer_domain)) (lambda (i)
				(list (quote equal??)
					(list (quote get_column) stage_alias false (nth key_names i) false)
					(nth outer_domain i))))
			true))))

(define exists_inner_supported? (lambda (inner)
	(and (query_block? inner)
		(and (single_source? (qb_sources inner))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (empty_list? (qb_order inner))
						(and (nil? (qb_limit inner))
							(and (nil? (qb_offset inner))
								(empty_list? (qb_stages inner)))))))))))

(define scalar_once_supported? (lambda (inner)
	(and (query_block? inner)
		(and (single_source? (qb_sources inner))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (nil? (qb_offset inner))
						(and (empty_list? (qb_stages inner))
							(equal? (qb_limit inner) 1)))))))))

(define scalar_once_reduce_first (lambda ()
	(list (quote lambda)
		(list (quote a) (quote b))
		(list (quote if) (list (quote nil?) (quote a)) (quote b) (quote a)))))

(define scalar_once_descriptor (lambda (value_expr order_items)
	(if (empty_list? order_items)
		(list value_expr (scalar_once_reduce_first) nil)
		(match order_items
			(cons item '()) (match item
				'(order_expr dir) (if (equal? order_expr value_expr)
					(list value_expr (if (equal? dir <) (quote min) (quote max)) nil)
					(neumann_fail "untangle_query" "scalar once_limit ORDER BY different expression needs ORC stage"))
				_ (neumann_fail "untangle_query" "malformed scalar once_limit ORDER BY"))
			_ (neumann_fail "untangle_query" "scalar once_limit supports one ORDER BY item")))))

(define make_exists_stage_rewrite (lambda (inner outer_sources subquery ctx)
	(begin
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "EXISTS group-stage(D) currently supports one plain inner query-block")
			true)
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "EXISTS group-stage(D) requires a base inner source after FROM flattening")
			true)
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? corr_pairs)
			'(1)
			(map corr_pairs (lambda (pair) (canonical_column_expr_for_alias inner_default (nth pair 0))))))
		(define outer_domain (if (empty_list? corr_pairs)
			'()
			(map corr_pairs (lambda (pair) (nth pair 1)))))
		(define condition (combine_where_terms local_terms true))
		(define stage_id (concat "exists:" (fnv_hash (string (list subquery keys outer_domain condition)))))
		(define stage (make_group_stage
			stage_id
			inner_src
			outer_domain
			keys
			(list aggregate_count_descriptor)
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) condition)
				(list (quote purpose) (quote exists-cache)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define grouptbl (exists_stage_table_name stage))
		(define source (list
			stage_alias
			(source_schema inner_src)
			grouptbl
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(list
			(list (quote >) (list (quote get_column) stage_alias false count_col false) 0)
			(list stage)
			(list source)))))

(define make_in_stage_rewrite (lambda (probe inner outer_sources subquery ctx)
	(begin
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "IN group-stage(D) currently supports one plain inner query-block")
			true)
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "IN group-stage(D) requires a base inner source after FROM flattening")
			true)
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define rhs_expr (canonical_column_expr_for_alias inner_default (query_block_first_expr inner)))
		(define keys (cons rhs_expr
			(map corr_pairs (lambda (pair) (canonical_column_expr_for_alias inner_default (nth pair 0))))))
		(define outer_domain (cons probe
			(map corr_pairs (lambda (pair) (nth pair 1)))))
		(define condition (combine_where_terms local_terms true))
		(define stage_id (concat "in:" (fnv_hash (string (list subquery probe keys outer_domain condition)))))
		(define stage (make_group_stage
			stage_id
			inner_src
			outer_domain
			keys
			(list aggregate_count_descriptor)
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) condition)
				(list (quote purpose) (quote in-cache)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define grouptbl (exists_stage_table_name stage))
		(define source (list
			stage_alias
			(source_schema inner_src)
			grouptbl
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(list
			(list (quote >) (list (quote get_column) stage_alias false count_col false) 0)
			(list stage)
			(list source)))))

(define make_scalar_aggregate_stage_rewrite (lambda (inner outer_sources subquery ctx)
	(begin
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "scalar aggregate group-stage(D) currently supports one plain inner query-block")
			true)
		(define value_expr (query_block_first_expr inner))
		(define ags (dedupe_aggregates_by_col (extract_aggregates value_expr)))
		(if (empty_list? ags)
			(neumann_fail "untangle_query" "table-backed scalar subquery without aggregate needs once_limit stage")
			true)
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "scalar aggregate group-stage(D) requires a base inner source after FROM flattening")
			true)
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? corr_pairs)
			'(1)
			(map corr_pairs (lambda (pair) (canonical_column_expr_for_alias inner_default (nth pair 0))))))
		(define outer_domain (if (empty_list? corr_pairs)
			'()
			(map corr_pairs (lambda (pair) (nth pair 1)))))
		(define condition (combine_where_terms local_terms true))
		(define stage_id (concat "scalar-agg:" (fnv_hash (string (list subquery keys outer_domain condition ags)))))
		(define stage (make_group_stage
			stage_id
			inner_src
			outer_domain
			keys
			ags
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) condition)
				(list (quote purpose) (quote scalar-aggregate-cache)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define grouptbl (exists_stage_table_name stage))
		(define source (list
			stage_alias
			(source_schema inner_src)
			grouptbl
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(list
			(replace_group_expr inner_default stage_alias keys key_names ags value_expr)
			(list stage)
			(list source)))))

(define make_scalar_once_stage_rewrite (lambda (inner outer_sources subquery ctx)
	(begin
		(if (not (scalar_once_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without aggregate needs once_limit stage")
			true)
		(define value_expr (query_block_first_expr inner))
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "scalar once_limit stage requires a base inner source after FROM flattening")
			true)
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? corr_pairs)
			'(1)
			(map corr_pairs (lambda (pair) (canonical_column_expr_for_alias inner_default (nth pair 0))))))
		(define outer_domain (if (empty_list? corr_pairs)
			'()
			(map corr_pairs (lambda (pair) (nth pair 1)))))
		(define condition (combine_where_terms local_terms true))
		(define value_for_inner (canonical_column_expr_for_alias inner_default value_expr))
		(define order_for_inner (map (coalesceNil (qb_order inner) '()) (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias inner_default expr) dir)))))
		(define ag (scalar_once_descriptor value_for_inner order_for_inner))
		(define ags (list ag))
		(define stage_id (concat "scalar-once:" (fnv_hash (string (list subquery keys outer_domain condition ag)))))
		(define stage (make_group_stage
			stage_id
			inner_src
			outer_domain
			keys
			ags
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) condition)
				(list (quote once_limit) 1)
				(list (quote purpose) (quote scalar-once-cache)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define grouptbl (exists_stage_table_name stage))
		(define source (list
			stage_alias
			(source_schema inner_src)
			grouptbl
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(list
			(list (quote get_column) stage_alias false (aggregate_col_name ag) false)
			(list stage)
			(list source)))))

(define window_aggregate_descriptor (lambda (fn args)
	(match fn
		"COUNT" (list aggregate_count_descriptor)
		"SUM" (list (list (car args) (quote +) 0))
		"MAX" (list (list (car args) (quote max) nil))
		"MIN" (list (list (car args) (quote min) nil))
		_ (neumann_fail "untangle_query" "window function needs ORC stage"))))

(define window_aggregate_value_expr (lambda (fn args ags stage_alias)
	(match fn
		"COUNT" (list (quote get_column) stage_alias false (aggregate_col_name aggregate_count_descriptor) false)
		"SUM" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"MAX" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"MIN" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		_ (neumann_fail "untangle_query" "window function needs ORC stage"))))

(define window_order_sort_dir (lambda (dir)
	(if (equal? dir >) true false)))

(define window_order_exprs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) expr
			_ (neumann_fail "untangle_query" "malformed window ORDER BY item"))))))

(define window_order_dirs_for_orc (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) (window_order_sort_dir dir)
			_ (neumann_fail "untangle_query" "malformed window ORDER BY item"))))))

(define make_row_number_orc_stage_rewrite (lambda (args over outer_sources ctx)
	(begin
		(if (not (empty_list? args))
			(neumann_fail "untangle_query" "ROW_NUMBER does not accept arguments")
			true)
		(if (not (single_source? outer_sources))
			(neumann_fail "untangle_query" "ROW_NUMBER ORC stage currently supports one base source")
			true)
		(define src (car outer_sources))
		(if (not (source_is_base_table? src))
			(neumann_fail "untangle_query" "ROW_NUMBER ORC stage requires a base source")
			true)
		(define alias (source_alias src))
		(define partition_exprs (nth over 0))
		(define order_items (nth over 1))
		(if (empty_list? order_items)
			(neumann_fail "untangle_query" "ROW_NUMBER requires ORDER BY for ORC lowering")
			true)
		(define partition_cols (map partition_exprs (lambda (expr)
			(order_column_for_alias alias (canonical_column_expr_for_alias alias expr)))))
		(define order_cols (map (window_order_exprs order_items) (lambda (expr)
			(order_column_for_alias alias (canonical_column_expr_for_alias alias expr)))))
		(define sortcols (merge (list partition_cols order_cols)))
		(define sortdirs (merge (list
			(map partition_exprs (lambda (_expr) false))
			(window_order_dirs_for_orc order_items))))
		(define col (concat "__orc_row_number_" (fnv_hash (string (list (source_relation src) alias sortcols sortdirs (count partition_exprs))))))
		(define stage (make_orc_stage
			(concat "orc-row-number:" col)
			src
			col
			sortcols
			sortdirs
			(count partition_exprs)
			partition_cols
			(if (empty_list? partition_exprs)
				(list (quote lambda) (list (quote $set)) (list (quote list) (quote $set)))
				(list (quote lambda)
					(cons (quote $set) (map partition_cols (lambda (colname) (symbol colname))))
					(list (quote cons)
						(if (single_source? partition_cols)
							(symbol (car partition_cols))
							(cons (quote list) (map partition_cols (lambda (colname) (symbol colname)))))
						(list (quote list) (quote $set)))))
			(if (empty_list? partition_exprs)
				(list (quote lambda) (list (quote acc) (quote mapped))
					(list (quote begin)
						(list (list (quote car) (quote mapped)) (list (quote +) (quote acc) 1))
						(list (quote +) (quote acc) 1)))
				(list (quote lambda) (list (quote acc) (quote mapped))
					(list (quote begin)
						(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
						(list (quote define) (quote writer) (list (quote car) (list (quote cdr) (quote mapped))))
						(list (quote define) (quote prev_partition) (list (quote cadr) (quote acc)))
						(list (quote define) (quote inner_acc) (list (quote car) (quote acc)))
						(list (quote define) (quote eff_acc) (list (quote if)
							(list (quote or)
								(list (quote nil?) (quote prev_partition))
								(list (quote equal?) (quote row_partition) (quote prev_partition)))
							(quote inner_acc)
							0))
						(list (quote writer) (list (quote +) (quote eff_acc) 1))
						(list (quote list) (list (quote +) (quote eff_acc) 1) (quote row_partition)))))
			(if (empty_list? partition_exprs) 0 (list (quote list) 0 nil))))
		(list
			(list (quote get_column) alias false col false)
			(list stage)
			'()))))

(define make_window_aggregate_stage_rewrite (lambda (fn args over outer_sources ctx)
	(begin
		(if (not (single_source? outer_sources))
			(neumann_fail "untangle_query" "window aggregate stage currently supports one base source")
			true)
		(define src (car outer_sources))
		(if (not (source_is_base_table? src))
			(neumann_fail "untangle_query" "window aggregate stage requires a base source")
			true)
		(define alias (source_alias src))
		(define partition_exprs (nth over 0))
		(define canonical_args (map (coalesceNil args '()) (lambda (arg) (canonical_column_expr_for_alias alias arg))))
		(define ags (dedupe_aggregates_by_col (window_aggregate_descriptor fn canonical_args)))
		(define keys (if (empty_list? partition_exprs)
			'(1)
			(map partition_exprs (lambda (expr) (canonical_column_expr_for_alias alias expr)))))
		(define outer_domain (if (empty_list? partition_exprs)
			'()
			partition_exprs))
		(define stage_id (concat "window-agg:" (fnv_hash (string (list fn canonical_args keys)))))
		(define stage (make_group_stage
			stage_id
			src
			outer_domain
			keys
			ags
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) true)
				(list (quote purpose) (quote window-aggregate-cache)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define grouptbl (exists_stage_table_name stage))
		(define source (list
			stage_alias
			(source_schema src)
			grouptbl
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(list
			(window_aggregate_value_expr fn canonical_args ags stage_alias)
			(list stage)
			(list source)))))

(define combine_stage_rewrite_results (lambda (head rewritten_args)
	(begin
		(define expr (cons head (map rewritten_args (lambda (item) (nth item 0)))))
		(define stages (merge (map rewritten_args (lambda (item) (nth item 1)))))
		(define sources (merge (map rewritten_args (lambda (item) (nth item 2)))))
		(list expr stages sources))))

(define untangle_expr_with_stages (lambda (expr outer_sources ctx)
	(match expr
		((symbol inner_select) subquery)
			(begin
				(define normalized (normalize_query_ast subquery))
				(define inner (untangle_query normalized ctx))
				(if (query_block_no_from? inner)
					(list (untangle_zero_domain_subquery (quote inner_select) nil subquery ctx) '() '())
					(if (empty_list? (extract_aggregates (query_block_first_expr inner)))
						(make_scalar_once_stage_rewrite inner outer_sources subquery ctx)
						(make_scalar_aggregate_stage_rewrite inner outer_sources subquery ctx))))
		((symbol inner_select_exists) subquery)
			(begin
				(define normalized (normalize_query_ast subquery))
				(define inner (untangle_query normalized ctx))
				(if (query_block_no_from? inner)
					(list (untangle_zero_domain_subquery (quote inner_select_exists) nil subquery ctx) '() '())
					(make_exists_stage_rewrite inner outer_sources subquery ctx)))
		((symbol inner_select_in) probe subquery)
			(begin
				(define normalized (normalize_query_ast subquery))
				(define inner (untangle_query normalized ctx))
				(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
				(if (query_block_no_from? inner)
					(list (untangle_zero_domain_subquery (quote inner_select_in) rewritten_probe subquery ctx) '() '())
					(make_in_stage_rewrite rewritten_probe inner outer_sources subquery ctx)))
		((symbol window_func) fn args over)
			(if (equal? fn "ROW_NUMBER")
				(make_row_number_orc_stage_rewrite args over outer_sources ctx)
				(make_window_aggregate_stage_rewrite fn args over outer_sources ctx))
		(cons head tail) (combine_stage_rewrite_results head (map tail (lambda (item) (untangle_expr_with_stages item outer_sources ctx))))
		_ (list expr '() '()))))

(define untangle_expr (lambda (expr ctx)
	(nth (untangle_expr_with_stages expr '() ctx) 0)))

(define untangle_fields_with_stages (lambda (fields outer_sources ctx)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (begin
			(define rewritten (untangle_expr_with_stages expr outer_sources ctx))
			(define tail (untangle_fields_with_stages rest outer_sources ctx))
			(list
				(cons title (cons (nth rewritten 0) (nth tail 0)))
				(merge (list (nth rewritten 1) (nth tail 1)))
				(merge (list (nth rewritten 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define untangle_fields (lambda (fields ctx)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
			(cons title (cons (untangle_expr expr ctx) (untangle_fields rest ctx)))
		_ '())))

(define field_expr_by_title (lambda (fields title)
	(match (coalesceNil fields '())
		(cons current_title (cons expr rest)) (if (equal? current_title title)
			expr
			(field_expr_by_title rest title))
		_ nil)))

(define derived_star_ref? (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		((quote get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		_ false)))

(define rewrite_derived_ref (lambda (alias projection expr)
	(match expr
		((symbol get_column) tblvar _ col _) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col) expr)
			expr)
		((quote get_column) tblvar _ col _) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col) expr)
			expr)
		(cons head tail) (cons head (map tail (lambda (item) (rewrite_derived_ref alias projection item))))
		_ expr)))

(define rewrite_derived_fields (lambda (alias projection fields)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (if (derived_star_ref? alias expr)
			(merge (list projection (rewrite_derived_fields alias projection rest)))
			(cons title (cons (rewrite_derived_ref alias projection expr) (rewrite_derived_fields alias projection rest))))
		_ '())))

(define rewrite_derived_order (lambda (alias projection order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (list (rewrite_derived_ref alias projection expr) dir)
			_ item)))))

(define rewrite_source_join_for_derived (lambda (alias projection src)
	(list
		(source_alias src)
		(source_schema src)
		(source_relation src)
		(source_outer? src)
		(rewrite_derived_ref alias projection (source_join_expr src)))))

(define rewrite_sources_join_for_derived (lambda (alias projection sources)
	(map (coalesceNil sources '()) (lambda (src)
		(rewrite_source_join_for_derived alias projection src)))))

(define rewrite_derived_ref_chain (lambda (rewrites expr)
	(reduce (coalesceNil rewrites '()) (lambda (acc rewrite)
		(match rewrite
			'(alias projection) (rewrite_derived_ref alias projection acc)
			_ acc))
		expr)))

(define rewrite_derived_fields_chain (lambda (rewrites fields)
	(reduce (coalesceNil rewrites '()) (lambda (acc rewrite)
		(match rewrite
			'(alias projection) (rewrite_derived_fields alias projection acc)
			_ acc))
		fields)))

(define rewrite_derived_order_chain (lambda (rewrites order_items)
	(reduce (coalesceNil rewrites '()) (lambda (acc rewrite)
		(match rewrite
			'(alias projection) (rewrite_derived_order alias projection acc)
			_ acc))
		order_items)))

(define requalify_single_source_expr (lambda (old_alias new_alias expr)
	(match expr
		((symbol get_column) tblvar ignorecase col json_path)
			(list (quote get_column) (if (or (nil? tblvar) (equal? tblvar old_alias)) new_alias tblvar) ignorecase col json_path)
		((quote get_column) tblvar ignorecase col json_path)
			(list (quote get_column) (if (or (nil? tblvar) (equal? tblvar old_alias)) new_alias tblvar) ignorecase col json_path)
		(cons head tail) (cons head (map tail (lambda (item) (requalify_single_source_expr old_alias new_alias item))))
		_ expr)))

(define requalify_single_source_fields (lambda (old_alias new_alias fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(requalify_single_source_expr old_alias new_alias expr)))))

(define combine_where (lambda (a b)
	(begin
		(define aa (coalesceNil a true))
		(define bb (coalesceNil b true))
		(if (equal? aa true)
			bb
			(if (equal? bb true)
				aa
				(list (quote and) aa bb)))))))

(define derived_block_needs_operator? (lambda (block)
	(or (not (empty_list? (qb_group block)))
		(or (not (nil? (qb_having block)))
			(or (not (nil? (qb_limit block)))
				(or (not (nil? (qb_offset block)))
					(or (not (empty_list? (qb_stages block)))
						(query_block_has_aggregates? block))))))))

(define untangle_flattened_base_source (lambda (src ctx)
	(list
		(source_alias src)
		(source_schema src)
		(normalize_query_ast (source_relation src))
		(source_outer? src)
		(untangle_expr (source_join_expr src) ctx))))

(define flatten_source_list (lambda (sources ctx)
	(if (empty_list? sources)
		(list '() '() '())
		(begin
			(define src (car sources))
			(define rest (cdr sources))
			(define relation (normalize_query_ast (source_relation src)))
				(define tail (flatten_source_list rest ctx))
				(define tail_sources (nth tail 0))
				(define tail_rewrites (nth tail 1))
				(define tail_wheres (nth tail 2))
				(if (string? relation)
					(begin
						(list
							(cons (untangle_flattened_base_source src ctx) tail_sources)
							tail_rewrites
							tail_wheres))
					(if (union_block? relation)
						(neumann_fail "untangle_query" "FROM union-block needs union logical lowering before source flattening")
						(begin
						(define alias (source_alias src))
						(define inner (untangle_query relation ctx))
						(if (derived_block_needs_operator? inner)
							(neumann_fail "untangle_query" "derived table with group/limit/stage must become an explicit logical operator before flattening")
							(begin
								(define inner_sources (coalesceNil (qb_sources inner) '()))
								(if (empty_list? inner_sources)
									(if (or (source_outer? src) (not (nil? (source_join_expr src))))
									(neumann_fail "untangle_query" "zero-source derived JOIN needs constant-relation support")
									(list
										(rewrite_sources_join_for_derived alias (qb_fields inner) tail_sources)
										(cons (list alias (qb_fields inner)) tail_rewrites)
										(cons (qb_where inner) tail_wheres)))
									(if (equal? (count inner_sources) 1)
										(begin
											(define only_inner (car inner_sources))
											(define inner_alias (source_alias only_inner))
											(define projection (requalify_single_source_fields inner_alias alias (qb_fields inner)))
											(define inner_where (requalify_single_source_expr inner_alias alias (qb_where inner)))
											(define outer_join (rewrite_derived_ref alias projection (source_join_expr src)))
											(define joined_condition (if (or (source_outer? src) (not (nil? outer_join)))
												(combine_where inner_where outer_join)
												nil))
											(define parent_condition (if (nil? joined_condition) inner_where nil))
											(list
												(cons
													(list
														alias
														(source_schema only_inner)
														(source_relation only_inner)
														(source_outer? src)
														joined_condition)
													(rewrite_sources_join_for_derived alias projection tail_sources))
												(cons (list alias projection) tail_rewrites)
												(if (nil? parent_condition) tail_wheres (cons parent_condition tail_wheres))))
										(if (or (source_outer? src) (not (nil? (source_join_expr src))))
											(neumann_fail "untangle_query" "multi-source derived JOIN needs relation-unit lowering")
											(list
												(merge (list inner_sources (rewrite_sources_join_for_derived alias (qb_fields inner) tail_sources)))
												(cons (list alias (qb_fields inner)) tail_rewrites)
												(cons (qb_where inner) tail_wheres)))))))))
						))))))))

(define combine_where_terms (lambda (terms seed)
	(reduce (coalesceNil terms '()) combine_where seed)))

(define untangle_source (lambda (src ctx)
	(begin
		(define relation (normalize_query_ast (source_relation src)))
		(if (or (query_block? relation) (union_block? relation))
			(neumann_fail "untangle_query" "FROM-subquery flattening must be implemented inside query-block, not left as source")
			(list
				(source_alias src)
				(source_schema src)
				relation
				(source_outer? src)
				(untangle_expr (source_join_expr src) ctx))))))

(define untangle_sources (lambda (sources ctx)
	(map (coalesceNil sources '()) (lambda (src) (untangle_source src ctx)))))

/* ------------------------------------------------------------------------- */
/* Top-down untangle                                                          */

(define untangle_query_block (lambda (block ctx)
	(begin
		(define child_ctx (make_uctx ctx
			(list
				(list (quote compile-budget-ms) 1000)
				(list (quote operator-model) (quote combined)))))
		(define flattened_sources (flatten_source_list (qb_sources block) child_ctx))
		(define sources (nth flattened_sources 0))
		(define rewrites (nth flattened_sources 1))
		(define source_where_terms (nth flattened_sources 2))
		(define rewritten_where (combine_where_terms source_where_terms (rewrite_derived_ref_chain rewrites (qb_where block))))
		(define where_result (untangle_expr_with_stages rewritten_where sources child_ctx))
		(define field_result (untangle_fields_with_stages (rewrite_derived_fields_chain rewrites (qb_fields block)) sources child_ctx))
		(define stage_sources (merge (list (nth where_result 2) (nth field_result 2))))
		(make_query_block
			(qb_schema block)
			(merge (list sources stage_sources))
			(nth field_result 0)
			(nth where_result 0)
			(map (coalesceNil (qb_group block) '()) (lambda (item) (untangle_expr (rewrite_derived_ref_chain rewrites item) child_ctx)))
			(untangle_expr (rewrite_derived_ref_chain rewrites (qb_having block)) child_ctx)
			(map (rewrite_derived_order_chain rewrites (qb_order block)) (lambda (item) (untangle_expr item child_ctx)))
			(qb_limit block)
			(qb_offset block)
			(untangle_fields (rewrite_derived_fields_chain rewrites (qb_hidden block)) child_ctx)
			(merge (list (qb_stages block) (nth where_result 1) (nth field_result 1)))
			(qb_facts block)))))

(define untangle_union_block (lambda (block ctx)
	(make_union_block
		(union_mode block)
		(map (union_branches block) (lambda (branch) (untangle_query (normalize_query_ast branch) ctx)))
		(union_order block)
		(union_limit block)
		(union_offset block)
		(union_facts block))))

(define untangle_query (lambda (query ctx)
	(begin
		(define normalized (normalize_query_ast query))
		(match (logical_op normalized)
			(symbol query-block) (untangle_query_block normalized ctx)
			(symbol union-block) (untangle_union_block normalized ctx)
			_ (neumann_fail "untangle_query" "expected query-block or union-block")))))

(define require_unnested_node (lambda (phase node)
	(begin
		(if (query_contains_subquery? node)
			(neumann_fail phase "subquery marker survived untangle_query")
			true)
		node)))

(define untangle_query_term (lambda (query ctx)
	(begin
		(define root (require_unnested_node "untangle_query" (untangle_query query ctx)))
		(make_ir (if (union_block? root) (quote union) (quote select))
			root
			(if (query_block? root) (qb_stages root) '())
			(make_uctx ctx (list
				(list (quote compile-budget-ms) 1000)
				(list (quote operator-model) (quote combined))))
			(quote rows)))))

/* ------------------------------------------------------------------------- */
/* Reorder/optimise scaffold                                                  */

(define join_reorder (lambda (ir)
	ir))

/* ------------------------------------------------------------------------- */
/* Physical lowering scaffold                                                 */

(define quoted_runtime_list (lambda (xs)
	(list (quote quote) (coalesceNil xs '()))))

(define append_unique (lambda (xs item)
	(if (contains? (coalesceNil xs '()) item)
		(coalesceNil xs '())
		(merge (coalesceNil xs '()) (list item)))))

(define merge_unique (lambda (parts)
	(reduce (merge (coalesceNil parts '())) (lambda (acc item)
		(append_unique acc item))
		'())))

(define single_source? (lambda (sources)
	(match (coalesceNil sources '())
		(cons _ '()) true
		_ false)))

(define resolve_column_alias (lambda (alias default_alias)
	(if (nil? alias) default_alias alias)))

(define extract_columns_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar alias) alias) (list col) '())
		((quote get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar alias) alias) (list col) '())
		(cons head tail) (merge_unique (map tail (lambda (item) (extract_columns_for_alias alias item))))
		_ '())))

(define lower_column_expr_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (symbol (concat (resolve_column_alias tblvar alias) "." col))
		((quote get_column) tblvar _ col _) (symbol (concat (resolve_column_alias tblvar alias) "." col))
		(cons head tail) (cons head (map tail (lambda (item) (lower_column_expr_for_alias alias item))))
		_ expr)))

(define extract_columns_for_join_alias (lambda (default_alias alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar default_alias) alias) (list col) '())
		((quote get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar default_alias) alias) (list col) '())
		(cons head tail) (merge_unique (map tail (lambda (item) (extract_columns_for_join_alias default_alias alias item))))
		_ '())))

(define lower_column_expr_for_join (lambda (default_alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (symbol (concat (resolve_column_alias tblvar default_alias) "." col))
		((quote get_column) tblvar _ col _) (symbol (concat (resolve_column_alias tblvar default_alias) "." col))
		(cons head tail) (cons head (map tail (lambda (item) (lower_column_expr_for_join default_alias item))))
		_ expr)))

(define canonical_column_expr_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (list (quote get_column) (resolve_column_alias tblvar alias) false col false)
		((quote get_column) tblvar _ col _) (list (quote get_column) (resolve_column_alias tblvar alias) false col false)
		(cons head tail) (cons head (map tail (lambda (item) (canonical_column_expr_for_alias alias item))))
		_ expr)))

(define order_column_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar alias) alias)
			col
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		((quote get_column) tblvar _ col _) (if (equal?? (resolve_column_alias tblvar alias) alias)
			col
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		_ (neumann_fail "build_queryplan" "ORDER BY expression lowering needs a computed sort column"))))

(define order_cols_for_alias (lambda (alias order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (order_column_for_alias alias expr)
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define order_dirs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) dir
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define aggregate_expr? (lambda (expr)
	(match expr
		((symbol aggregate) _expr _reduce _neutral) true
		((quote aggregate) _expr _reduce _neutral) true
		_ false)))

(define aggregate_count_descriptor (list 1 (quote +) 0))

(define extract_aggregates (lambda (expr)
	(match expr
		((symbol aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		((quote aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		(cons head tail) (merge_unique (map tail extract_aggregates))
		_ '())))

(define stage_aggregates_for_fields (lambda (fields)
	(merge_unique (extract_assoc fields (lambda (_title expr) (extract_aggregates expr))))))

(define group_key_col_name (lambda (i)
	(concat "k" i)))

(define aggregate_col_name (lambda (ag)
	(concat "agg_" (fnv_hash (string ag)))))

(define dedupe_aggregates_by_col (lambda (ags)
	(reduce (coalesceNil ags '()) (lambda (acc ag)
		(begin
			(define col (aggregate_col_name ag))
			(if (reduce acc (lambda (found existing)
				(or found (equal? col (aggregate_col_name existing))))
				false)
				acc
				(merge acc (list ag)))))
		'())))

(define group_table_name (lambda (schema tbl alias keys condition ags)
	(concat ".grp:" tbl ":" (fnv_hash (string (list "neumann-clean-groups-v2" schema tbl alias keys condition ags))))))

(define group_key_cols (lambda (keys)
	(map (produceN (count keys)) group_key_col_name)))

(define assoc_keys_as_dataset_rows (lambda (dict width)
	(map (extract_assoc dict (lambda (k v) k))
		(lambda (k)
			(if (list? k)
				k
				(if (<= width 1)
					(list k)
					(map (produceN width) (lambda (_) nil))))))))

(define make_group_stage_for_block (lambda (block src)
	(begin
		(define visible_ags (stage_aggregates_for_fields (qb_fields block)))
		(define ags (dedupe_aggregates_by_col (if (empty_list? (qb_group block))
			visible_ags
			(merge_unique (list visible_ags (list aggregate_count_descriptor))))))
		(define alias (source_alias src))
		(make_group_stage
			(concat "group:" (source_relation src) ":" (fnv_hash (string (list (qb_group block) ags))))
			src
			'()
			(map (coalesceNil (qb_group block) '()) (lambda (expr) (canonical_column_expr_for_alias alias expr)))
			ags
			(qb_having block)
			(qb_fields block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(list
				(list (quote condition) (coalesceNil (qb_where block) true)))))))

(define group_key_index (lambda (alias keys expr)
	(if (or (nil? expr) (or (equal? expr true) (equal? expr false)))
		nil
		(begin
			(define resolved (canonical_column_expr_for_alias alias expr))
			(reduce (produceN (count keys)) (lambda (found i)
				(if (not (nil? found))
					found
					(if (equal? resolved (nth keys i)) i nil)))
				nil)))))

(define replace_group_expr (lambda (alias grouptbl keys key_names ags expr)
	(begin
		(define key_idx (group_key_index alias keys expr))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
					(list (quote get_column) grouptbl false (aggregate_col_name (list agg_expr agg_reduce agg_neutral)) false)
				((quote aggregate) agg_expr agg_reduce agg_neutral)
					(list (quote get_column) grouptbl false (aggregate_col_name (list agg_expr agg_reduce agg_neutral)) false)
				((symbol get_column) _tblvar _ _col _)
					(if (equal?? (resolve_column_alias _tblvar alias) alias)
						(neumann_fail "build_queryplan" "non-aggregate output must be a GROUP BY key")
						expr)
				((quote get_column) _tblvar _ _col _)
					(if (equal?? (resolve_column_alias _tblvar alias) alias)
						(neumann_fail "build_queryplan" "non-aggregate output must be a GROUP BY key")
						expr)
				(cons head tail) (cons head (map tail (lambda (item) (replace_group_expr alias grouptbl keys key_names ags item))))
				_ expr)))))

(define group_key_equality_terms (lambda (alias key_names keys)
	(map (produceN (count keys)) (lambda (i)
		(list (quote equal?)
			(lower_column_expr_for_alias alias (nth keys i))
			(list (quote outer) (symbol (nth key_names i))))))))

(define build_group_collect_plan (lambda (schema tbl alias grouptbl keys key_names condition)
	(if (equal? keys '(1))
		(list (quote insert)
			(list (quote table) schema grouptbl)
			(quoted_runtime_list (list "k0"))
			(list (quote list) (list (quote list) 1))
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			true)
		(begin
			(define keycols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias alias expr)))))
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(quoted_runtime_list '())
				(list (quote lambda) '() true)
				(cons (quote list) keycols)
				(list (quote lambda)
					(map keycols (lambda (col) (symbol (concat alias "." col))))
					(cons (quote list) (map keys (lambda (expr) (lower_column_expr_for_alias alias expr)))))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc) (quote acc) (quote rowvals) true))
				(quoted_runtime_list '())
				(list (quote lambda) (list (quote acc) (quote sharddict))
					(list (quote insert)
						(list (quote table) schema grouptbl)
						(cons (quote list) key_names)
						(list (quote assoc_keys_as_dataset_rows) (quote sharddict) (count keys))
						(quoted_runtime_list '())
						(list (quote lambda) '() true)
						true))
				false)))))

(define build_group_aggregate_column (lambda (schema tbl alias grouptbl keys key_names condition ag)
	(match ag '(agg_expr agg_reduce agg_neutral) (begin
		(define agg_col (aggregate_col_name ag))
		(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias alias expr)))))
		(define condition_cols (extract_columns_for_alias alias condition))
		(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols)))
		(define aggcols (extract_columns_for_alias alias agg_expr))
		(list (quote createcolumn)
			(list (quote table) schema grouptbl)
			agg_col
			"any"
			(quoted_runtime_list '())
			(quoted_runtime_list '("temp" true))
			(cons (quote list) key_names)
			(list (quote lambda)
				(map key_names (lambda (col) (symbol col)))
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(list (quote optimize)
							(cons (quote and) (cons
								(lower_column_expr_for_alias alias condition)
								(group_key_equality_terms alias key_names keys)))))
					(cons (quote list) aggcols)
					(list (quote lambda)
						(map aggcols (lambda (col) (symbol (concat alias "." col))))
						(lower_column_expr_for_alias alias agg_expr))
					agg_reduce
					agg_neutral
					nil
					false)))))))

(define build_group_keytable_cleanup (lambda (schema tbl alias grouptbl keys key_names)
	(begin
		(define pairs (map (produceN (count keys)) (lambda (i)
			(match (nth keys i)
				((symbol get_column) _ _ col _) (list col (nth key_names i))
				((quote get_column) _ _ col _) (list col (nth key_names i))
				_ nil))))
		(if (reduce pairs (lambda (bad p) (or bad (nil? p))) false)
			nil
			(list (quote register_keytable_cleanup)
				(list (quote table) schema tbl)
				(list (quote table) schema grouptbl)
				alias
				(cons (quote list) (map pairs (lambda (p) (cons (quote list) p)))))))))

(define group_stage_keytable_block (lambda (stage grouptbl key_names ags having_expr)
	(make_query_block
		(source_schema (gs_input stage))
		(list (list grouptbl (source_schema (gs_input stage)) grouptbl false nil))
		(gs_output stage)
		having_expr
		nil nil
		(map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr dir) (list (replace_group_expr (source_alias (gs_input stage)) grouptbl (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)) key_names ags expr) dir))))
		(gs_limit stage)
		(gs_offset stage)
		'() '() '())))

(define rewrite_source_for_group_domain (lambda (alias grouptbl keys key_names ags src)
	(list
		(source_alias src)
		(source_schema src)
		(source_relation src)
		(source_outer? src)
		(replace_group_expr alias grouptbl keys key_names ags (source_join_expr src)))))

(define group_stage_final_block (lambda (stage extra_sources)
	(begin
		(define src (gs_input stage))
		(define schema (source_schema src))
		(define tbl (source_relation src))
		(define alias (source_alias src))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(define grouptbl (group_table_name schema tbl alias keys condition ags))
		(define output_fields (map_assoc (gs_output stage) (lambda (_title expr)
			(replace_group_expr alias grouptbl keys key_names ags expr))))
		(define replaced_having (replace_group_expr alias grouptbl keys key_names ags (coalesceNil (gs_having stage) true)))
		(define count_col_name (aggregate_col_name aggregate_count_descriptor))
		(define count_check (list (quote >) (list (quote get_column) grouptbl false count_col_name false) 0))
		(define needs_count_filter (and (not (equal? keys '(1))) (not (equal? condition true))))
		(define having_expr (if (not needs_count_filter)
			replaced_having
			(if (or (nil? replaced_having) (equal? replaced_having true))
				count_check
				(list (quote and) replaced_having count_check))))
		(make_query_block
			schema
			(cons
				(list grouptbl schema grouptbl false nil)
				(map (coalesceNil extra_sources '()) (lambda (extra)
					(rewrite_source_for_group_domain alias grouptbl keys key_names ags extra))))
			output_fields
			having_expr
			nil nil
			(map (coalesceNil (gs_order stage) '()) (lambda (item)
				(match item '(expr dir) (list (replace_group_expr alias grouptbl keys key_names ags expr) dir))))
			(gs_limit stage)
			(gs_offset stage)
			'() '() '()))))

(define lower_group_stage_prepare (lambda (stage)
	(begin
		(define src (gs_input stage))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "group-stage lowering only supports base tables")
			true)
		(define schema (source_schema src))
		(define tbl (source_relation src))
		(define alias (source_alias src))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(define grouptbl (group_table_name schema tbl alias keys condition ags))
		(define key_columns (map key_names (lambda (col) (list (quote list) "column" col "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))
		(define agg_columns (map ags (lambda (ag) (list (quote list) "column" (aggregate_col_name ag) "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))
		(define create_cols (cons (quote list)
			(cons (cons (quote list) (cons "unique" (cons "group" (list (cons (quote list) key_names)))))
				(merge (list key_columns agg_columns)))))
		(define keytable_init (list (quote !begin)
			(list (quote if)
				(list (quote createtable) schema grouptbl create_cols (quoted_runtime_list '("engine" "sloppy")) true)
				nil
				nil)
			(list (quote touch_keytable) (list (quote table) schema grouptbl))
			(list (quote or)
				(list (quote not) (list (quote has?) (list (quote show) schema) grouptbl))
				(list (quote table_empty?) (list (quote table) schema grouptbl)))))
		(define collect_plan (build_group_collect_plan schema tbl alias grouptbl keys key_names condition))
		(define cleanup_plan (build_group_keytable_cleanup schema tbl alias grouptbl keys key_names))
		(define agg_plans (map ags (lambda (ag) (build_group_aggregate_column schema tbl alias grouptbl keys key_names condition ag))))
		(list (quote begin)
			(if (nil? cleanup_plan)
				(list (quote begin)
					keytable_init
					collect_plan)
				(list (quote if) keytable_init
					(list (quote begin)
						collect_plan
						cleanup_plan)
					nil))
			(cons (quote parallel) agg_plans)))))

(define lower_orc_stage_prepare (lambda (stage)
	(begin
		(define src (os_source stage))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "ORC stage lowering only supports base tables")
			true)
		(list (quote createcolumn)
			(list (quote table) (source_schema src) (source_relation src))
			(os_column stage)
			"any"
			(quoted_runtime_list '())
			(list (quote list)
				"temp" true
				"sortcols" (quoted_runtime_list (os_sortcols stage))
				"sortdirs" (cons (quote list) (os_sortdirs stage))
				"partitioncount" (os_partitioncount stage)
				"mapcols" (quoted_runtime_list (os_mapcols stage))
				"mapfn" (os_mapfn stage)
				"reducefn" (os_reducefn stage)
				"reduceinit" (os_reduceinit stage))))))

(define lower_stage_prepare (lambda (stage)
	(if (group_stage? stage)
		(lower_group_stage_prepare stage)
		(if (orc_stage? stage)
			(lower_orc_stage_prepare stage)
			(neumann_fail "build_queryplan" "unknown logical stage")))))

(define lower_group_stage (lambda (stage)
	(begin
		(define src (gs_input stage))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "group-stage lowering only supports base tables")
			true)
		(define schema (source_schema src))
		(define tbl (source_relation src))
		(define alias (source_alias src))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(define grouptbl (group_table_name schema tbl alias keys condition ags))
		(list (quote begin)
			(lower_group_stage_prepare stage)
			(lower_query_block_core (group_stage_final_block stage '())))))))

(define lower_grouped_query_block_with_stages (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define base_src (car sources))
		(if (not (source_is_base_table? base_src))
			(neumann_fail "build_queryplan" "group-stage with subquery stages requires a base driver source")
			true)
		(define stage_sources (cdr sources))
		(define base_block (make_query_block
			(qb_schema block)
			(list base_src)
			(qb_fields block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(qb_hidden block)
			'()
			(qb_facts block)))
		(define main_stage (make_group_stage_for_block base_block base_src))
		(cons (quote begin)
			(merge (list
				(list (lower_group_stage_prepare main_stage))
				(map (qb_stages block) lower_stage_prepare)
				(list (lower_query_block_core (group_stage_final_block main_stage stage_sources)))))))))

(define query_block_without_stages (lambda (block)
	(make_query_block
		(qb_schema block)
		(qb_sources block)
		(qb_fields block)
		(qb_where block)
		(qb_group block)
		(qb_having block)
		(qb_order block)
		(qb_limit block)
		(qb_offset block)
		(qb_hidden block)
		'()
		(qb_facts block))))

(define lower_query_block_core (lambda (block)
	(if (empty_list? (qb_sources block))
		(lower_zero_source_query_block block)
		(if (single_source? (qb_sources block))
			(lower_single_source_query_block block)
			(lower_multi_source_query_block block)))))

(define lower_query_block_with_stages (lambda (block)
	(if (empty_list? (qb_stages block))
		(lower_query_block_core block)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(lower_grouped_query_block_with_stages block)
			(cons (quote begin)
				(merge (list
					(map (qb_stages block) lower_stage_prepare)
					(list (lower_query_block_core (query_block_without_stages block))))))))))

(define source_is_base_table? (lambda (src)
	(string? (source_relation src))))

(define query_block_has_aggregates? (lambda (block)
	(not (empty_list? (stage_aggregates_for_fields (qb_fields block))))))

(define table_column_names (lambda (schema tbl)
	(map (show schema tbl) (lambda (col) (col "Field")))))

(define star_expr_alias (lambda (expr)
	(match expr
		((symbol get_column) tblvar _ "*" _) tblvar
		((quote get_column) tblvar _ "*" _) tblvar
		_ false)))

(define star_expr? (lambda (expr)
	(match expr
		((symbol get_column) _ _ "*" _) true
		((quote get_column) _ _ "*" _) true
		_ false)))

(define expand_star_for_sources (lambda (sources requested_alias)
	(merge (map (coalesceNil sources '()) (lambda (src)
		(if (or (nil? requested_alias) (equal? requested_alias (source_alias src)))
			(merge (map (table_column_names (source_schema src) (source_relation src)) (lambda (col)
				(list col (list (quote get_column) (source_alias src) false col false)))))
			'()))))))

(define expand_query_block_fields (lambda (sources fields)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (begin
			(define requested_alias (star_expr_alias expr))
			(if (star_expr? expr)
				(merge (list (expand_star_for_sources sources requested_alias) (expand_query_block_fields sources rest)))
				(cons title (cons expr (expand_query_block_fields sources rest)))))
		_ '())))

(define lower_single_source_query_block (lambda (block)
	(begin
		(define src (car (qb_sources block)))
		(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "single-source query-block lowering only supports base tables")
			true)
		(if (or (source_outer? src) (not (nil? (source_join_expr src))))
			(neumann_fail "build_queryplan" "single-source query-block lowering does not support joins yet")
			true)
		(if (not (empty_list? (qb_stages block)))
			(neumann_fail "build_queryplan" "pre-existing stages are not implemented yet")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(lower_group_stage (make_group_stage_for_block block src))
			(begin
		(if (and (empty_list? (qb_order block)) (or (not (nil? (qb_limit block))) (not (nil? (qb_offset block)))))
			(neumann_fail "build_queryplan" "LIMIT/OFFSET without ORDER BY is not implemented yet")
			true)
		(define alias (source_alias src))
		(define condition (coalesceNil (qb_where block) true))
		(define order_items (coalesceNil (qb_order block) '()))
		(define filtercols (extract_columns_for_alias alias condition))
		(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
			(extract_columns_for_alias alias expr)))))
		(define ordercols (if (empty_list? order_items) '() (order_cols_for_alias alias order_items)))
		(define mapcols (merge_unique (list filtercols fieldcols)))
		(define table_expr (list (quote table) (source_schema src) (source_relation src)))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (symbol (concat alias "." col))))
			(list (quote optimize) (lower_column_expr_for_alias alias condition))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (symbol (concat alias "." col))))
			(list (quote resultrow)
				(cons (quote list) (map_assoc fields (lambda (title expr)
					(lower_column_expr_for_alias alias expr)))))))
		(if (empty_list? order_items)
			(list (quote scan)
				'(session "__memcp_tx")
				table_expr
				(cons (quote list) filtercols)
				filter_expr
				(cons (quote list) mapcols)
				map_expr
				nil
				nil
				nil
				false)
			(list (quote scan_order)
				'(session "__memcp_tx")
				table_expr
				(cons (quote list) filtercols)
				filter_expr
				(cons (quote list) ordercols)
				(cons (quote list) (order_dirs order_items))
				0
				(coalesceNil (qb_offset block) 0)
				(coalesceNil (qb_limit block) -1)
				(cons (quote list) mapcols)
				map_expr
				nil
				nil
				false)))))))

(define order_exprs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) expr
			_ true)))))

(define source_join_exprs (lambda (sources)
	(map (coalesceNil sources '()) (lambda (src) (coalesceNil (source_join_expr src) true)))))

(define join_cols_for_alias (lambda (default_alias alias needed_exprs)
	(merge_unique (map (coalesceNil needed_exprs '()) (lambda (expr)
		(extract_columns_for_join_alias default_alias alias expr))))))

(define join_filter_cols_for_alias (lambda (default_alias alias condition)
	(extract_columns_for_join_alias default_alias alias condition)))

(define lower_join_result_fields (lambda (default_alias fields)
	(map_assoc fields (lambda (_title expr)
		(lower_column_expr_for_join default_alias expr)))))

(define build_join_scan_rows (lambda (schema sources default_alias needed_exprs final_condition fields order_items offset_value limit_value)
	(if (empty_list? sources)
		(if (equal? (coalesceNil final_condition true) true)
			(list (quote list)
				(list (quote resultrow)
					(cons (quote list) (lower_join_result_fields default_alias fields))))
			(list (quote if)
				(list (quote optimize) (lower_column_expr_for_join default_alias final_condition))
				(list (quote list)
					(list (quote resultrow)
						(cons (quote list) (lower_join_result_fields default_alias fields))))
				(list (quote list))))
		(begin
			(define src (car sources))
			(if (not (source_is_base_table? src))
				(neumann_fail "build_queryplan" "multi-source query-block lowering only supports base tables after untangle")
				true)
			(define alias (source_alias src))
			(define condition (coalesceNil (source_join_expr src) true))
			(define filtercols (join_filter_cols_for_alias default_alias alias condition))
			(define mapcols (join_cols_for_alias default_alias alias needed_exprs))
			(define table_expr (list (quote table) (source_schema src) (source_relation src)))
			(define filter_expr (list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat alias "." col))))
				(list (quote optimize) (lower_column_expr_for_join default_alias condition))))
			(define map_expr (list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat alias "." col))))
				(build_join_scan_rows schema (cdr sources) default_alias needed_exprs final_condition fields '() 0 -1)))
			(define reduce_expr (list (quote lambda) (list (quote acc) (quote subrows))
				(list (quote merge) (quote acc) (quote subrows))))
			(if (empty_list? order_items)
				(list (quote scan)
					'(session "__memcp_tx")
					table_expr
					(cons (quote list) filtercols)
					filter_expr
					(cons (quote list) mapcols)
					map_expr
					reduce_expr
					(list (quote list))
					nil
					(source_outer? src))
				(list (quote scan_order)
					'(session "__memcp_tx")
					table_expr
					(cons (quote list) filtercols)
					filter_expr
					(cons (quote list) (order_cols_for_alias alias order_items))
					(cons (quote list) (order_dirs order_items))
					0
					(coalesceNil offset_value 0)
					(coalesceNil limit_value -1)
					(cons (quote list) mapcols)
					map_expr
					reduce_expr
					(list (quote list))
					(source_outer? src)))))))

(define lower_multi_source_query_block (lambda (block)
	(begin
		(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
		(if (not (empty_list? (qb_stages block)))
			(neumann_fail "build_queryplan" "pre-existing stages are not implemented yet")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(neumann_fail "build_queryplan" "group-stage over joins is not implemented yet")
			true)
		(if (or (not (nil? (qb_limit block))) (not (nil? (qb_offset block))))
			(if (empty_list? (qb_order block))
				(neumann_fail "build_queryplan" "LIMIT/OFFSET without ORDER BY is not implemented yet")
				true)
			true)
		(define sources (qb_sources block))
		(define first_alias (source_alias (car sources)))
		(define needed_exprs (merge (list
			(extract_assoc fields (lambda (_title expr) expr))
			(list (coalesceNil (qb_where block) true))
			(order_exprs (qb_order block))
			(source_join_exprs sources))))
		(define rows_plan (build_join_scan_rows (qb_schema block) sources first_alias needed_exprs (coalesceNil (qb_where block) true) fields (qb_order block) (qb_offset block) (qb_limit block)))
		rows_plan)))

(define lower_zero_source_query_block (lambda (block)
	(if (equal? (coalesceNil (qb_where block) true) true)
		(list (quote resultrow) (cons (quote list) (qb_fields block)))
		(list (quote if)
			(qb_where block)
			(list (quote resultrow) (cons (quote list) (qb_fields block)))
			(list (quote resultrow) (list (quote list)))))))

(define build_queryplan (lambda (ir)
	(begin
		(require_unnested_node "build_queryplan input" (ir_root ir))
		(match (ir_return ir)
			(symbol rows) (match (logical_op (ir_root ir))
				(symbol query-block) (lower_query_block_with_stages (ir_root ir))
				(symbol union-block) (neumann_fail "build_queryplan" "union-block lowering is intentionally not scaffolded yet")
				_ (neumann_fail "build_queryplan" "unknown logical root"))
			_ (neumann_fail "build_queryplan" "DML lowering is intentionally not scaffolded yet")))))

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
		(ir_with_return (untangle_query_term query nil) sink_mode))))

(define build_queryplan_term_from_logical (lambda (logical_ir)
	(neumann_compile_ir_pipeline logical_ir)))

(define build_queryplan_term_from_logical_with_sink (lambda (logical_ir sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with_return logical_ir sink_mode))))

(define build_dml_plan (lambda (_schema _tbl _tblalias _all_defs _cols _condition _order _limit _offset)
	(neumann_fail "build_dml_plan" "DML lowering is outside the combined-operator scaffold")))

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
