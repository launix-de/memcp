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

This file is the clean query compiler pipeline:

parser AST -> untangle_query -> join_reorder -> build_queryplan

The logical IR deliberately uses a small set of combined operators instead of
textbook one-operation nodes:

query-block  SELECT/FROM/JOIN/WHERE/project/ORDER/LIMIT work unit
group-stage  domain-D/keytable/aggregate/EXISTS/HAVING work unit
union-block  set operator work unit
orc-stage    ordered-reduced computed column barrier for incompatible windows

There is no logical scan operator.  scan, scan_order, scan_order_multi, index
bounds, update callbacks, temp tables and fused loops are physical choices made
by build_queryplan after the logical program is decorrelated and optimized.

The Neumann/top-down invariant is strict: untangle_query must remove logical
subquery expressions.  Correlated scalar/EXISTS/IN forms become explicit domain
and keytable work; they must not survive as scalar fallback calls hidden inside
expression lowering.  Unsupported forms should fail early and loudly until a
relational rewrite exists.

Logical trees may be deep.  That depth is compiler structure, not a license to
materialize.  The lowering goal is to fuse filter, projection, joins, ordering,
limits and DML sinks into the strongest physical scan available.  Materialization
is reserved for real barriers: keytables/groups, ORC columns, unions with global
ordering/deduplication, deduplicated domains and once-limit/scalar caches.
Group stages are referenced from query-block sources through logical
stage-output relations.  Those aliases are stable enough for expression rewrites
such as SUM(x) -> stage_alias.agg, but they are not physical table names.
build_queryplan resolves a stage-output through a group-carrier decision.  The
default carrier is a generated keytable; later optimizers may choose an existing
referenced table when the final group/domain keys match a foreign-key entity.
Window functions are not ORCs by default.  If the query is unordered and has a
single window domain, build_queryplan should run the main scan in that window
order and compute the value in the scan accumulator.  ORCs are only needed when
multiple window domains need incompatible orders, or when the final query order
conflicts with the window order.  Pure partition aggregates use group-stage /
keytable lookup, not ORC.  In hot paths, avoid using runtime row lists as a
general adapter; prefer emitting a single fused scan/keytable builder once the
logical rules are clear.

Keep the code small and explicit.  Do not reintroduce legacy fallback branches,
parser-specific operator variants, or early subselect recompilation.  SQL and
PostgreSQL parsers should both lower to the same combined operators.
*/

/* ------------------------------------------------------------------------- */
/* Small assoc helpers                                                        */

(define get_schema (lambda (schema tbl)
	(try
		(lambda () (show schema tbl))
		(lambda (_e) '()))))

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

(define qp_any (lambda (x) x))

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

(define make_stage_output_relation (lambda (stage_id)
	(list (quote stage-output) stage_id)))

(define logical_op (lambda (node) (if (and (list? node) (not (nil? node))) (car node) nil)))
(define query_block? (lambda (node) (equal? (logical_op node) (quote query-block))))
(define group_stage? (lambda (node) (equal? (logical_op node) (quote group-stage))))
(define union_block? (lambda (node) (equal? (logical_op node) (quote union-block))))
(define orc_stage? (lambda (node) (equal? (logical_op node) (quote orc-stage))))
(define window_stage? (lambda (node) (equal? (logical_op node) (quote window-stage))))
(define stage_output_relation? (lambda (relation) (equal? (logical_op relation) (quote stage-output))))
(define stage_output_relation_id (lambda (relation)
	(match relation
		'(_ stage_id) stage_id
		_ nil)))

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

(define make_window_stage (lambda (id source column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit facts)
	(list (quote window-stage)
		id source column
		(coalesceNil sortcols '())
		(coalesceNil sortdirs '())
		(coalesceNil partitioncount 0)
		(coalesceNil mapcols '())
		mapfn
		reducefn
		reduceinit
		(coalesceNil facts '()))))

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

(define source_with_relation (lambda (src relation)
	(match src
		'(alias schema _relation outer join) (list alias schema relation outer join)
		_ src)))

(define source_with_join_expr (lambda (src join_expr)
	(match src
		'(alias schema relation outer _join) (list alias schema relation outer join_expr)
		_ src)))

(define source_with_schema_relation (lambda (src schema relation)
	(match src
		'(alias _schema _relation outer join) (list alias schema relation outer join)
		_ src)))

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

(define make_dependent_subquery_marker (lambda (kind probe subquery outer_sources)
	(list
		(quote dependent-subquery)
		kind
		probe
		subquery
		(coalesceNil outer_sources '()))))

(define dependent_subquery_marker? (lambda (expr)
	(match expr
		((symbol dependent-subquery) _kind _probe _subquery _outer_sources) true
		((quote dependent-subquery) _kind _probe _subquery _outer_sources) true
		_ false)))

(define dep_subquery_kind (lambda (expr) (nth expr 1)))
(define dep_subquery_probe (lambda (expr) (nth expr 2)))
(define dep_subquery_query (lambda (expr) (nth expr 3)))
(define dep_subquery_outer_sources (lambda (expr) (nth expr 4)))

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
		((quote union_all) branches order limit offset)
		(make_union_block (quote all) branches order limit offset '())
		((symbol union_distinct) branches order limit offset)
		(make_union_block (quote union_distinct) branches order limit offset '())
		((quote union_distinct) branches order limit offset)
		(make_union_block (quote union_distinct) branches order limit offset '())
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
	(if (contains? (list
		(quote inner_select)
		(quote inner_select_in)
		(quote inner_select_exists)
		(quote neumann_scalar)
		(quote neumann_exists)
		(quote neumann_in)
		(quote dependent-subquery)
		(quote exists_probe)) head)
		true
		(match head
			((quote quote) sym) (subquery_head? sym)
			_ false))))

(define expr_contains_subquery? (lambda (expr)
	(match expr
		(cons head tail) (or
			(subquery_head? head)
			(reduce tail (lambda (a b) (or a (expr_contains_subquery? b))) false))
		_ false)))

(define expr_contains_window? (lambda (expr)
	(match expr
		((symbol window_func) _fn _args _over) true
		((quote window_func) _fn _args _over) true
		(cons _head tail) (reduce tail (lambda (a b) (or a (expr_contains_window? b))) false)
		_ false)))

(define expr_contains_orc_column? (lambda (expr)
	(match expr
		((symbol get_column) _tblvar _ignorecase col _json_path)
		(and (string? col) (and (>= (strlen col) 6) (equal? (substr col 0 6) "__orc_")))
		((quote get_column) _tblvar _ignorecase col _json_path)
		(and (string? col) (and (>= (strlen col) 6) (equal? (substr col 0 6) "__orc_")))
		(cons _head tail) (reduce tail (lambda (a b) (or a (expr_contains_orc_column? b))) false)
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
							(or (fields_contains_subquery? (qb_order node))
								(or (reduce (qb_sources node) (lambda (found src)
									(or found
										(or (expr_contains_subquery? (source_join_expr src))
											(expr_contains_subquery? (source_relation src)))))
									false)
									(reduce (qb_stages node) (lambda (found stage)
										(or found (query_contains_subquery? stage)))
										false))))))))
		(if (union_block? node)
			(reduce (union_branches node) (lambda (a b) (or a (query_contains_subquery? b))) false)
			(if (group_stage? node)
				(or (query_contains_subquery? (gs_input node))
					(or (fields_contains_subquery? (gs_domain node))
						(or (fields_contains_subquery? (gs_keys node))
							(or (expr_contains_subquery? (gs_having node))
								(or (fields_contains_subquery? (gs_output node))
									(fields_contains_subquery? (gs_order node)))))))
				false)))))

(define physical_helper_relation? (lambda (relation)
	(and (string? relation) (strlike relation ".grp:%"))))

(define node_contains_physical_helper? (lambda (node)
	(if (query_block? node)
		(or (reduce (qb_sources node) (lambda (found src)
			(or found
				(or (physical_helper_relation? (source_relation src))
					(node_contains_physical_helper? (source_relation src)))))
			false)
			(reduce (qb_stages node) (lambda (found stage)
				(or found (node_contains_physical_helper? stage)))
				false))
		(if (union_block? node)
			(reduce (union_branches node) (lambda (found branch)
				(or found (node_contains_physical_helper? branch)))
				false)
			(if (group_stage? node)
				(node_contains_physical_helper? (gs_input node))
				false)))))

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
					(symbol inner_select_in) (begin
						(define rhs (query_block_first_expr inner))
						(define compare
							(list (quote if)
								(list (quote nil?) probe)
								nil
								(list (quote if)
									(list (quote nil?) rhs)
									nil
									(list (quote equal??) probe rhs))))
						(if (equal? where true)
							compare
							(list (quote if) where compare false)))
					_ (neumann_fail "untangle_query" "unknown subquery expression")))))))

(define untangle_zero_domain_not_in_subquery (lambda (probe subquery ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define inner (untangle_query normalized ctx))
		(if (not (query_block_no_from? inner))
			(neumann_fail "untangle_query" "table-backed NOT IN subquery unnesting must become group-stage(D) before lowering")
			(begin
				(define where (coalesceNil (qb_where inner) true))
				(define rhs (query_block_first_expr inner))
				(define compare
					(list (quote if)
						(list (quote nil?) probe)
						nil
						(list (quote if)
							(list (quote nil?) rhs)
							nil
							(list (quote not) (list (quote equal??) probe rhs)))))
				(if (equal? where true)
					compare
					(list (quote if) where compare true)))))))

(define source_aliases (lambda (sources)
	(map (coalesceNil sources '()) source_alias)))

(define window_stage_sources (lambda (sources)
	(filter (coalesceNil sources '()) (lambda (src)
		(and (not (source_outer? src)) (source_is_base_table? src))))))

(define expr_refs_alias? (lambda (default_alias alias expr)
	(match expr
		((symbol get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		((quote get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		(cons _head tail) (reduce tail (lambda (found item) (or found (expr_refs_alias? default_alias alias item))) false)
		_ false)))

(define expr_refs_any_alias? (lambda (default_alias aliases expr)
	(reduce (coalesceNil aliases '()) (lambda (found alias)
		(or found (expr_refs_alias? default_alias alias expr))) false)))

(define btw2025_expr_accessing_aliases (lambda (expr outer_aliases)
	(merge_unique (map (coalesceNil outer_aliases '()) (lambda (alias)
		(if (expr_refs_any_alias? nil (list alias) expr)
			(list alias)
			'()))))))

(define btw2025_fields_accessing_aliases (lambda (fields outer_aliases)
	(merge_unique (extract_assoc (coalesceNil fields '()) (lambda (_title expr)
		(btw2025_expr_accessing_aliases expr outer_aliases))))))

(define btw2025_order_accessing_aliases (lambda (order_items outer_aliases)
	(merge_unique (map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (btw2025_expr_accessing_aliases expr outer_aliases)
			_ '()))))))

(define btw2025_sources_accessing_aliases (lambda (sources outer_aliases)
	(merge_unique (map (coalesceNil sources '()) (lambda (src)
		(btw2025_expr_accessing_aliases (coalesceNil (source_join_expr src) true) outer_aliases))))))

(define btw2025_terms_accessing_aliases (lambda (terms outer_aliases)
	(merge_unique (map (coalesceNil terms '()) (lambda (term)
		(btw2025_expr_accessing_aliases term outer_aliases))))))

(define btw2025_query_block_accessing_aliases (lambda (block outer_sources)
	(if (not (query_block? block))
		'()
		(begin
			(define outer_aliases (source_aliases outer_sources))
			(merge_unique (list
				(btw2025_sources_accessing_aliases (qb_sources block) outer_aliases)
				(btw2025_fields_accessing_aliases (qb_fields block) outer_aliases)
				(btw2025_fields_accessing_aliases (qb_hidden block) outer_aliases)
				(btw2025_expr_accessing_aliases (qb_where block) outer_aliases)
				(btw2025_expr_accessing_aliases (coalesceNil (qb_having block) true) outer_aliases)
				(btw2025_order_accessing_aliases (qb_order block) outer_aliases)))))))

(define expr_refs_alias_after_group? (lambda (default_alias alias expr)
	(match expr
		((symbol aggregate) _agg_expr _agg_reduce _agg_neutral) false
		((quote aggregate) _agg_expr _agg_reduce _agg_neutral) false
		((symbol count_distinct) _agg_expr) false
		((quote count_distinct) _agg_expr) false
		((symbol get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		((quote get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		(cons _head tail) (reduce tail (lambda (found item) (or found (expr_refs_alias_after_group? default_alias alias item))) false)
		_ false)))

(define fields_ref_alias? (lambda (default_alias alias fields)
	(reduce (extract_assoc (coalesceNil fields '()) (lambda (_title expr) expr))
		(lambda (found expr)
			(or found (expr_refs_alias_after_group? default_alias alias expr)))
		false)))

(define order_ref_alias? (lambda (default_alias alias order_items)
	(reduce (coalesceNil order_items '()) (lambda (found item)
		(or found
			(match item
				'(expr _dir) (expr_refs_alias_after_group? default_alias alias expr)
				_ false)))
		false)))

(define source_needed_after_group? (lambda (default_alias block src)
	(and (stage_output_relation? (source_relation src))
		(or
			(fields_ref_alias? default_alias (source_alias src) (qb_fields block))
			(or
				(fields_ref_alias? default_alias (source_alias src) (qb_hidden block))
				(or
					(expr_refs_alias_after_group? default_alias (source_alias src) (coalesceNil (qb_having block) true))
					(order_ref_alias? default_alias (source_alias src) (qb_order block))))))))

(define source_needed_after_group_stage? (lambda (default_alias stage src)
	(and (stage_output_relation? (source_relation src))
		(or
			(fields_ref_alias? default_alias (source_alias src) (gs_output stage))
			(or
				(expr_refs_alias_after_group? default_alias (source_alias src) (coalesceNil (gs_having stage) true))
				(order_ref_alias? default_alias (source_alias src) (gs_order stage)))))))

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

(define unique_correlation_pairs (lambda (pairs)
	(reduce (coalesceNil pairs '()) (lambda (acc pair)
		(if (contains? acc pair)
			acc
			(merge acc (list pair))))
		'())))

(define correlation_inner_keys (lambda (inner_default pairs)
	(map (coalesceNil pairs '()) (lambda (pair)
		(canonical_column_expr_for_alias inner_default (nth pair 0))))))

(define correlation_lookup_keys (lambda (pairs)
	(map (coalesceNil pairs '()) (lambda (pair) (nth pair 1)))))

(define correlation_domain (lambda (pairs)
	(merge_unique (list (correlation_lookup_keys pairs)))))

(define btw2025_cclasses_for_pairs (lambda (pairs)
	(map (coalesceNil pairs '()) (lambda (pair)
		(list (nth pair 0) (nth pair 1))))))

(define btw2025_repr_for_pairs (lambda (inner_default pairs)
	(map (coalesceNil pairs '()) (lambda (pair)
		(list
			(nth pair 1)
			(canonical_column_expr_for_alias inner_default (nth pair 0)))))))

(define make_btw2025_unnesting_info (lambda (parent outer_refs domain accessing accessing_after_simple cclasses repr)
	(list
		(quote btw2025-unnesting-info)
		parent
		(coalesceNil outer_refs '())
		(coalesceNil domain '())
		(coalesceNil accessing '())
		(coalesceNil accessing_after_simple '())
		(coalesceNil cclasses '())
		(coalesceNil repr '()))))

(define btw2025_info_parent (lambda (info) (nth info 1)))
(define btw2025_info_outer_refs (lambda (info) (nth info 2)))
(define btw2025_info_domain (lambda (info) (nth info 3)))
(define btw2025_info_accessing (lambda (info) (nth info 4)))
(define btw2025_info_accessing_after_simple (lambda (info) (nth info 5)))
(define btw2025_info_cclasses (lambda (info) (nth info 6)))
(define btw2025_info_repr (lambda (info) (nth info 7)))

(define group_keys_for_correlations (lambda (inner_default pairs explicit_group_keys)
	(begin
		(define corr_keys (correlation_inner_keys inner_default pairs))
		(if (and (empty_list? corr_keys) (empty_list? explicit_group_keys))
			'(1)
			(reduce explicit_group_keys (lambda (acc key)
				(if (contains? acc key)
					acc
					(merge acc (list key))))
				corr_keys)))))

(define source_join_correlation_pairs (lambda (inner_default inner_aliases outer_aliases sources)
	(merge
		(map
			(coalesceNil sources '())
			(lambda (src)
				(filter
					(map
						(split_and_terms (coalesceNil (source_join_expr src) true))
						(lambda (term)
							(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
					(lambda (pair)
						(not (nil? pair)))))))))

(define source_without_outer_join_terms (lambda (inner_default inner_aliases outer_aliases src)
	(begin
		(define terms (split_and_terms (coalesceNil (source_join_expr src) true)))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
			(list
				(source_alias src)
				(source_schema src)
				(source_relation src)
				(source_outer? src)
				(combine_where_terms local_terms true)))))

(define sources_without_outer_join_terms (lambda (inner_default inner_aliases outer_aliases sources)
	(map (coalesceNil sources '()) (lambda (src)
		(source_without_outer_join_terms inner_default inner_aliases outer_aliases src)))))

(define btw2025_local_where_terms_after_simple (lambda (inner_default inner_aliases outer_aliases block)
	(filter
		(split_and_terms (coalesceNil (qb_where block) true))
		(lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term))))))

(define btw2025_accessing_after_simple (lambda (block outer_sources)
	(if (not (query_block? block))
		'()
		(begin
			(define inner_default (if (empty_list? (qb_sources block)) nil (source_alias (car (qb_sources block)))))
			(define inner_aliases (source_aliases (qb_sources block)))
			(define outer_aliases (source_aliases outer_sources))
			(define local_terms (btw2025_local_where_terms_after_simple inner_default inner_aliases outer_aliases block))
			(define local_sources (sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources block)))
			(merge_unique (list
				(btw2025_sources_accessing_aliases local_sources outer_aliases)
				(btw2025_fields_accessing_aliases (qb_fields block) outer_aliases)
				(btw2025_fields_accessing_aliases (qb_hidden block) outer_aliases)
				(btw2025_terms_accessing_aliases local_terms outer_aliases)
				(btw2025_expr_accessing_aliases (coalesceNil (qb_having block) true) outer_aliases)
				(btw2025_order_accessing_aliases (qb_order block) outer_aliases)))))))

(define btw2025_stage_facts (lambda (block outer_sources lookup_pairs)
	(begin
		(define inner_default (if (or (not (query_block? block)) (empty_list? (qb_sources block))) nil (source_alias (car (qb_sources block)))))
		(define accessing (btw2025_query_block_accessing_aliases block outer_sources))
		(define accessing_after_simple (btw2025_accessing_after_simple block outer_sources))
		(define domain (correlation_domain lookup_pairs))
		(define cclasses (btw2025_cclasses_for_pairs lookup_pairs))
		(define repr (btw2025_repr_for_pairs inner_default lookup_pairs))
		(define info (make_btw2025_unnesting_info nil domain domain accessing accessing_after_simple cclasses repr))
		(list
			(list (quote btw2025_accessing) accessing)
			(list (quote btw2025_accessing_after_simple) accessing_after_simple)
			(list (quote btw2025_simple_d_eliminated) (and (not (empty_list? accessing)) (empty_list? accessing_after_simple)))
			(list (quote btw2025_domain) domain)
			(list (quote btw2025_cclasses) cclasses)
			(list (quote btw2025_repr) repr)
			(list (quote btw2025_info) info)
			(list (quote btw2025_lookup_keys) (correlation_lookup_keys lookup_pairs))))))

(define exists_stage_alias (lambda (stage_id)
	(concat "__exists_" (fnv_hash stage_id))))

(define stage_source_outer? (lambda (outer_sources)
	(not (empty_list? outer_sources))))

(define make_exists_stage_join_condition (lambda (stage_alias key_names outer_domain)
	(if (empty_list? outer_domain)
		true
		(combine_where_terms
			(map (produceN (count outer_domain)) (lambda (i)
				(list (quote equal??)
					(list (quote get_column) stage_alias false (nth key_names i) false)
					(nth outer_domain i))))
			true))))

(define make_positive_in_join_condition (lambda (stage_alias key_names lookup_keys probe match_ag)
	(combine_where_terms
		(list
			(make_exists_stage_join_condition stage_alias key_names lookup_keys)
			(list (quote not) (list (quote nil?) probe))
			(list (quote >) (membership_count_expr stage_alias match_ag) 0))
		true)))

(define in_null_count_descriptor (lambda (rhs_expr)
	(list (list (quote if) (list (quote nil?) rhs_expr) 1 0) (quote +) 0)))

(define membership_count_expr (lambda (stage_alias ag)
	(list (quote coalesceNil)
		(list (quote get_column) stage_alias false (aggregate_col_name ag) false)
		0)))

(define in_membership_expr (lambda (probe match_alias match_ag null_alias null_ag)
	(begin
		(define match_count (membership_count_expr match_alias match_ag))
		(define null_count (membership_count_expr null_alias null_ag))
		(list (quote if)
			(list (quote nil?) probe)
			nil
			(list (quote if)
				(list (quote >) match_count 0)
				true
				(list (quote if)
					(list (quote >) null_count 0)
					nil
					false))))))

(define not_in_membership_expr (lambda (probe match_alias match_ag null_alias null_ag)
	(begin
		(define match_count (membership_count_expr match_alias match_ag))
		(define null_count (membership_count_expr null_alias null_ag))
		(list (quote if)
			(list (quote nil?) probe)
			nil
			(list (quote if)
				(list (quote >) match_count 0)
				false
				(list (quote if)
					(list (quote >) null_count 0)
					nil
					true))))))

(define scalar_first_probe_expr (lambda (stage col)
	(list (quote scalar_first_probe) stage col)))

(define make_stage_lookup_condition (lambda (stage_alias key_names outer_domain post_condition)
	(combine_where
		(make_exists_stage_join_condition stage_alias key_names outer_domain)
		post_condition)))

(define source_is_stage_output? (lambda (src)
	(stage_output_relation? (source_relation src))))

(define source_is_not_stage_output? (lambda (src)
	(not (source_is_stage_output? src))))

(define scalar_source_shape_supported? (lambda (sources)
	(match (coalesceNil sources '())
		(cons base helpers)
		(and (source_is_base_table? base)
			(reduce helpers (lambda (ok src)
				(and ok (source_is_stage_output? src)))
				true))
		_ false)))

(define exists_inner_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (empty_list? (qb_order inner))
						(and (or (nil? (qb_limit inner)) (equal? (qb_limit inner) 1))
							(nil? (qb_offset inner))))))))))

(define scalar_once_supported? (lambda (inner)
	(and (query_block? inner)
		(and (scalar_source_shape_supported? (qb_sources inner))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (or (nil? (qb_offset inner)) (not (empty_list? (qb_order inner))))
						(equal? (qb_limit inner) 1))))))))

(define scalar_aggregate_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_order inner))
				(and (or (nil? (qb_limit inner)) (equal? (qb_limit inner) 1))
					(nil? (qb_offset inner))))))))

(define scalar_single_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (empty_list? (qb_order inner))
						(and (nil? (qb_limit inner))
							(nil? (qb_offset inner))))))))))

(define grouped_scalar_top_supported? (lambda (inner outer_sources)
	(and (empty_list? outer_sources)
		(and (query_block? inner)
			(and (scalar_source_shape_supported? (qb_sources inner))
				(and (not (empty_list? (qb_group inner)))
					(and (nil? (qb_having inner))
						(and (not (empty_list? (qb_order inner)))
							(and (equal? (qb_limit inner) 1)
								(nil? (qb_offset inner)))))))))))

(define scalar_once_reduce_first (lambda ()
	(list (quote lambda)
		(list (quote a) (quote b))
		(list (quote if) (list (quote nil?) (quote a)) (quote b) (quote a)))))

(define scalar_first_payload (lambda (order_key value)
	(list order_key value)))

(define scalar_first_order_reduce (lambda (dir)
	(list (quote lambda)
		(list (quote a) (quote b))
		(list (quote if)
			(list (quote nil?) (list (quote car) (quote a)))
			(quote b)
			(list (quote if)
				(list dir (list (quote car) (quote b)) (list (quote car) (quote a)))
				(quote b)
				(quote a))))))

(define scalar_once_ordered_payload? (lambda (ag)
	(match ag
		'(((symbol scalar_first_payload) _order_expr _value_expr) _reduce _neutral) true
		'(((quote scalar_first_payload) _order_expr _value_expr) _reduce _neutral) true
		_ false)))

(define scalar_once_value_expr (lambda (stage_alias ag)
	(begin
		(define stored (list (quote get_column) stage_alias false (aggregate_col_name ag) false))
		(if (scalar_once_ordered_payload? ag)
			(list (quote if)
				(list (quote or)
					(list (quote nil?) stored)
					(list (quote nil?) (list (quote car) stored)))
				nil
				(list (quote cadr) stored))
			stored))))

(define scalar_single_value_expr (lambda (stage_alias value_ag count_ag)
	(begin
		(define count_expr (list (quote coalesceNil)
			(list (quote get_column) stage_alias false (aggregate_col_name count_ag) false)
			0))
		(define value_expr (list (quote get_column) stage_alias false (aggregate_col_name value_ag) false))
		(list (quote if)
			(list (quote >) count_expr 1)
			(list (quote error) "scalar subselect returned more than one row")
			value_expr))))

(define scalar_single_aggregates (lambda (value_expr)
	(list
		(scalar_once_descriptor value_expr '() nil)
		aggregate_count_descriptor)))

(define expr_refs_stage_output_alias? (lambda (expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(and (string? tblvar) (strlike tblvar "__exists_%"))
		((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(and (string? tblvar) (strlike tblvar "__exists_%"))
		(cons _head tail) (reduce tail (lambda (found item)
			(or found (expr_refs_stage_output_alias? item)))
			false)
		_ false)))

(define scalar_first_probe_stage? (lambda (stage)
	(and (group_stage? stage)
		(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single))
			(and (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote first))
				(and (not (empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
					(and (empty_list? (qassoc_get (gs_facts stage) (quote btw2025_accessing_after_simple) '()))
						(and (not (reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (found key)
							(or found (expr_refs_stage_output_alias? key)))
							false))
							(and (source_is_base_table? (gs_input stage))
								(and (equal? (count (gs_aggregates stage)) 1)
									(and (equal? (nth (car (gs_aggregates stage)) 1) (scalar_once_reduce_first))
										(not (scalar_once_ordered_payload? (car (gs_aggregates stage)))))))))))))))

(define scalar_once_descriptor (lambda (value_expr order_items offset_value)
	(if (empty_list? order_items)
		(list value_expr (scalar_once_reduce_first) nil)
		(begin
			(define order_exprs (map order_items (lambda (item)
				(match item
					'(order_expr _dir) order_expr
					_ (neumann_fail "untangle_query" "malformed scalar once_limit ORDER BY")))))
			(define dirs (map order_items (lambda (item)
				(match item
					'(_order_expr dir) dir
					_ (neumann_fail "untangle_query" "malformed scalar once_limit ORDER BY")))))
			(if (and (equal? (coalesceNil offset_value 0) 0)
				(and (single_source? order_exprs) (equal? (car order_exprs) value_expr)))
				(list value_expr (if (equal? (car dirs) <) (quote min) (quote max)) nil)
				(list
					(list (quote scalar_order_value) value_expr order_exprs dirs (coalesceNil offset_value 0))
					(scalar_once_reduce_first)
					nil))))))

(define order_item_exprs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) expr
			_ true)))))

(define make_grouped_scalar_top_rewrite (lambda (inner subquery)
	(begin
		(define inner_src (car (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define visible_ags (stage_aggregates_for_fields (qb_fields inner)))
		(define order_ags (merge_unique (map (order_item_exprs (qb_order inner)) extract_aggregates)))
		(define ags (dedupe_aggregates_by_col (merge_unique (list visible_ags order_ags (list aggregate_count_descriptor)))))
		(define keys (map (coalesceNil (qb_group inner) '()) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define stage_id (concat "scalar-group-top:" (fnv_hash (serialize (list subquery keys ags (qb_order inner))))))
		(define stage (make_group_stage
			stage_id
			inner_src
			'()
			keys
			ags
			nil
			(qb_fields inner)
			(qb_order inner)
			1
			nil
			(list
				(list (quote condition) (coalesceNil (qb_where inner) true))
				(list (quote purpose) (quote scalar_aggregate))
				(list (quote domain) '())
				(list (quote lookup-keys) '())
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote scalar))
				(list (quote cardinality_mode) (quote first)))))
		(list
			(list (quote grouped_scalar_top) stage)
			(list stage)
			'()))))

(define make_exists_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "EXISTS group-stage(D) currently supports one plain inner query-block")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_aliases outer_aliases (qb_sources inner)))
		(define lookup_pairs (unique_correlation_pairs (merge (list corr_pairs source_corr_pairs))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define stage_input (if (and (single_source? (qb_sources inner)) (empty_list? (qb_stages inner)))
			inner_src
			(make_query_block
				(qb_schema inner)
				(sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources inner))
				'()
				condition
				'() nil '() nil nil
				'()
				(qb_stages inner)
				(qb_facts inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "exists:" (fnv_hash (string (list subquery keys outer_domain condition)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			(list aggregate_count_descriptor)
			nil
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote exists))
					(list (quote presence_only) true)
					(list (quote max_needed_per_domain) 1)
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) false)
					(list (quote null_semantics) (quote exists))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define source (list
				stage_alias
				(group_stage_schema stage)
				(make_stage_output_relation stage_id)
				(stage_source_outer? outer_sources)
				(make_exists_stage_join_condition stage_alias key_names lookup_keys)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(list
			(list (quote >)
				(list (quote coalesceNil)
					(list (quote get_column) stage_alias false count_col false)
					0)
				0)
			(list stage)
			(list source)))))

(define combine_exists_union_results (lambda (results)
	(match (coalesceNil results '())
		(cons item rest) (begin
			(define tail (combine_exists_union_results rest))
			(list
				(if (empty_list? rest)
					(nth item 0)
					(list (quote or) (nth item 0) (nth tail 0)))
				(merge (list (nth item 1) (nth tail 1)))
				(merge (list (nth item 2) (nth tail 2)))))
		_ (list false '() '()))))

(define make_exists_union_stage_rewrite (lambda (inner args)
	(if (not (union_block? inner))
		(neumann_fail "untangle_query" "EXISTS UNION lowering expects union-block")
		(if (or (not (empty_list? (union_order inner)))
			(or (not (nil? (union_limit inner))) (not (nil? (union_offset inner)))))
			(neumann_fail "untangle_query" "EXISTS over UNION currently supports plain unordered branches")
			(combine_exists_union_results
				(map (union_branches inner) (lambda (branch)
					(make_exists_stage_rewrite branch args))))))))

(define combine_in_union_results (lambda (results negate)
	(match (coalesceNil results '())
		(cons item rest) (begin
			(define tail (combine_in_union_results rest negate))
			(list
				(if (empty_list? rest)
					(nth item 0)
					(list (if negate (quote and) (quote or)) (nth item 0) (nth tail 0)))
				(merge (list (nth item 1) (nth tail 1)))
				(merge (list (nth item 2) (nth tail 2)))))
		_ (list false '() '()))))

(define make_in_union_stage_rewrite (lambda (probe inner args)
	(if (not (union_block? inner))
		(neumann_fail "untangle_query" "IN UNION lowering expects union-block")
		(if (or (not (empty_list? (union_order inner)))
			(or (not (nil? (union_limit inner))) (not (nil? (union_offset inner)))))
			(neumann_fail "untangle_query" "IN over UNION currently supports plain unordered branches")
			(if (not (in_union_single_column? inner))
				(neumann_fail "untangle_query" "IN UNION subquery branches must expose exactly one column")
				(if (and (not (if (>= (count args) 3) (nth args 2) false))
					(in_union_candidate_supported? inner (nth args 0)))
					(make_in_union_candidate_stage_rewrite probe inner args)
					(combine_in_union_results
						(map (union_branches inner) (lambda (branch)
							(make_in_stage_rewrite probe branch args)))
						(if (>= (count args) 3) (nth args 2) false))))))))

(define in_union_single_column? (lambda (inner)
	(reduce (union_branches inner) (lambda (ok branch)
		(and ok
			(and (query_block? branch)
				(equal? (count (qb_fields branch)) 2))))
		true)))

(define in_union_candidate_supported? (lambda (inner outer_sources)
	(and (union_block? inner)
		(or (equal? (union_mode inner) (quote all))
			(or (equal? (union_mode inner) (quote distinct))
				(equal? (union_mode inner) (quote union_distinct))))
		(reduce (union_branches inner) (lambda (ok branch)
			(and ok
				(and (exists_inner_supported? branch)
					(and (equal? (count (qb_fields branch)) 2)
						(empty_list? (btw2025_query_block_accessing_aliases branch outer_sources))))))
			true))))

(define make_in_union_candidate_branch (lambda (branch)
	(begin
		(define inner_src (car (qb_sources branch)))
		(define inner_default (source_alias inner_src))
		(define rhs_expr (canonical_column_expr_for_alias inner_default (query_block_first_expr branch)))
		(make_query_block
			(qb_schema branch)
			(qb_sources branch)
			(list "v" rhs_expr)
			(qb_where branch)
			'() nil '() nil nil
			'()
			(qb_stages branch)
			(qb_facts branch)))))

(define make_in_union_candidate_stage_rewrite (lambda (probe inner args)
	(begin
		(define semijoin_where (if (>= (count args) 4) (nth args 3) false))
		(define candidate_alias (concat "__in_candidate_" (fnv_hash (string (list probe inner)))))
		(define union_input (make_union_block
			(union_mode inner)
			(map (union_branches inner) make_in_union_candidate_branch)
			'() nil nil
			(list
				(list (quote purpose) (quote in_candidate_union))
				(list (quote alias) candidate_alias))))
		(define candidate_key (list (quote get_column) candidate_alias false "v" false))
		(define keys (list candidate_key))
		(define stage_id (concat "in-candidate:" (fnv_hash (string (list probe inner)))))
		(define null_stage_id (concat "in-candidate-null:" (fnv_hash (string inner))))
		(define null_ag (in_null_count_descriptor candidate_key))
		(define stage (make_group_stage
			stage_id
			union_input
			'()
			keys
			(list aggregate_count_descriptor)
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) true)
				(list (quote purpose) (quote in_candidate))
				(list (quote candidate) true)
				(list (quote domain) '())
				(list (quote lookup-keys) (list probe))
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote in))
				(list (quote cardinality_mode) (quote many)))))
		(define null_stage (make_group_stage
			null_stage_id
			union_input
			'()
			'(1)
			(list null_ag)
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) true)
				(list (quote purpose) (quote in_candidate_rhs_nulls))
				(list (quote candidate) true)
				(list (quote domain) '())
				(list (quote lookup-keys) '())
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote in))
				(list (quote cardinality_mode) (quote many)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define null_stage_alias (exists_stage_alias null_stage_id))
		(define key_names (group_key_cols keys))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			false
			(if semijoin_where
				(make_positive_in_join_condition stage_alias key_names (list probe) probe aggregate_count_descriptor)
				(make_exists_stage_join_condition stage_alias key_names (list probe)))))
		(define null_source (list
			null_stage_alias
			(group_stage_schema null_stage)
			(make_stage_output_relation null_stage_id)
			false
			true))
		(if semijoin_where
			(list true (list stage) (list source))
			(list
				(in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
				(list stage null_stage)
				(list source null_source))))))

(define make_in_stage_rewrite (lambda (probe inner args)
	(begin
		(define outer_sources (nth args 0))
		(define negate (if (>= (count args) 3) (nth args 2) false))
		(define semijoin_where (and (not negate) (if (>= (count args) 4) (nth args 3) false)))
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "IN group-stage(D) currently supports one plain inner query-block")
			true)
		(if (not (equal? (count (qb_fields inner)) 2))
			(neumann_fail "untangle_query" "IN subquery must expose exactly one column")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define lookup_pairs (unique_correlation_pairs corr_pairs))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define rhs_expr (canonical_column_expr_for_alias inner_default (query_block_first_expr inner)))
		(define keys (cons rhs_expr
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define null_keys (if (empty_list? (correlation_inner_keys inner_default lookup_pairs))
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define domain_lookup_keys (correlation_lookup_keys lookup_pairs))
		(define lookup_keys (cons probe domain_lookup_keys))
		(define condition (combine_where_terms local_terms true))
		(define stage_input (if (and (source_is_base_table? inner_src) (empty_list? (qb_stages inner)))
			inner_src
			(make_query_block
				(qb_schema inner)
				(sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources inner))
				'()
				condition
				'() nil '() nil nil
				'()
				(qb_stages inner)
				(qb_facts inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "in:" (fnv_hash (string (list probe keys lookup_keys condition)))))
		(define null_ag (in_null_count_descriptor rhs_expr))
		(define null_stage_id (concat "in-null:" (fnv_hash (string (list outer_domain condition rhs_expr)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			(list aggregate_count_descriptor)
			nil
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote in_membership))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) false)
					(list (quote null_semantics) (quote in))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs)))))
		(define null_stage (make_group_stage
			null_stage_id
			stage_input
			outer_domain
			null_keys
			(list null_ag)
			nil
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote in_rhs_nulls))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) domain_lookup_keys)
					(list (quote preserve_empty_domain) false)
					(list (quote null_semantics) (quote in))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define null_stage_alias (exists_stage_alias null_stage_id))
		(define key_names (group_key_cols keys))
		(define null_key_names (group_key_cols null_keys))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(if semijoin_where false (stage_source_outer? outer_sources))
			(if semijoin_where
				(make_positive_in_join_condition stage_alias key_names lookup_keys probe aggregate_count_descriptor)
				(make_exists_stage_join_condition stage_alias key_names lookup_keys))))
		(define null_source (list
			null_stage_alias
			(group_stage_schema null_stage)
			(make_stage_output_relation null_stage_id)
			(stage_source_outer? outer_sources)
			(make_exists_stage_join_condition null_stage_alias null_key_names domain_lookup_keys)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(if semijoin_where
			(list true (list stage) (list source))
			(list
				(if negate
					(not_in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
					(in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag))
				(list stage null_stage)
				(list source null_source))))))

(define make_scalar_aggregate_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(if (not (scalar_aggregate_supported? inner))
			(neumann_fail "untangle_query" "scalar aggregate group-stage(D) currently supports one plain inner query-block")
			true)
		(define value_expr (query_block_first_expr inner))
		(define ags (dedupe_aggregates_by_col (merge (list (extract_aggregates value_expr) (list aggregate_count_descriptor)))))
		(if (empty_list? ags)
			(neumann_fail "untangle_query" "table-backed scalar subquery without aggregate needs cardinality_mode single_or_error lowering")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define having_terms (split_and_terms (coalesceNil (qb_having inner) true)))
		(define having_corr_pairs (filter (map having_terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_having_terms (filter having_terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_aliases outer_aliases (qb_sources inner)))
		(define all_corr_pairs (unique_correlation_pairs (merge (list corr_pairs having_corr_pairs source_corr_pairs))))
		(define explicit_group_keys (map (coalesceNil (qb_group inner) '()) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define keys (group_keys_for_correlations inner_default all_corr_pairs explicit_group_keys))
		(define outer_domain (correlation_domain all_corr_pairs))
		(define lookup_keys (correlation_lookup_keys all_corr_pairs))
		(define condition (combine_where_terms local_terms true))
		(define local_having (combine_where_terms local_having_terms true))
		(define stage_input (if (and (single_source? (qb_sources inner)) (empty_list? (qb_stages inner)))
			inner_src
			(make_query_block
				(qb_schema inner)
				(sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources inner))
				'()
				condition
				'() nil '() nil nil
				'()
				(qb_stages inner)
				(qb_facts inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "scalar-agg:" (fnv_hash (string (list subquery keys outer_domain condition ags)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			ags
			local_having
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote scalar_aggregate))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) true)
					(list (quote null_semantics) (quote aggregate))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources all_corr_pairs)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define post_condition (replace_group_expr inner_default stage_alias keys key_names ags local_having))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(or (stage_source_outer? outer_sources) (not (equal? local_having true)))
			(make_stage_lookup_condition stage_alias key_names lookup_keys post_condition)))
		(define presence_expr (list (quote get_column) stage_alias false (aggregate_col_name aggregate_count_descriptor) false))
		(define scalar_value_expr (replace_group_expr inner_default stage_alias keys key_names ags value_expr))
		(list
			(if (equal? local_having true)
				scalar_value_expr
				(list (quote if) (list (quote nil?) presence_expr) nil scalar_value_expr))
			(list stage)
			(list source)))))

(define make_scalar_once_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(if (not (scalar_once_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
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
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_aliases outer_aliases (qb_sources inner)))
		(define lookup_pairs (unique_correlation_pairs (merge (list corr_pairs source_corr_pairs))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define value_for_inner (canonical_column_expr_for_alias inner_default value_expr))
		(define order_for_inner (map (coalesceNil (qb_order inner) '()) (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias inner_default expr) dir)))))
		(define ag (scalar_once_descriptor value_for_inner order_for_inner (qb_offset inner)))
		(define ags (list ag))
		(define stage_input (if (empty_list? (qb_stages inner))
			inner_src
			(make_query_block
				(qb_schema inner)
				(sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources inner))
				'()
				condition
				'() nil '() nil nil
				'()
				(qb_stages inner)
				(qb_facts inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "scalar-once:" (fnv_hash (string (list subquery keys outer_domain condition ag)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			ags
			nil
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote scalar_single))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) true)
					(list (quote null_semantics) (quote scalar))
					(list (quote cardinality_mode) (quote first))
					(list (quote partition_by) outer_domain)
					(list (quote physical_max_rows) 1)
					(list (quote on_overflow) (quote ignore)))
				(btw2025_stage_facts inner outer_sources lookup_pairs)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(stage_source_outer? outer_sources)
			(make_exists_stage_join_condition stage_alias key_names lookup_keys)))
		(list
			(scalar_once_value_expr stage_alias ag)
			(list stage)
			(list source)))))

(define make_limited_derived_stage_source (lambda (alias inner original_relation)
	(begin
		(if (not (scalar_once_supported? inner))
			(neumann_fail "untangle_query" "derived relation stage currently supports ORDER/LIMIT scalar shape only")
			true)
		(if (not (equal? (count (qb_fields inner)) 2))
			(neumann_fail "untangle_query" "derived relation stage currently supports one projected column")
			true)
		(define value_expr (query_block_first_expr inner))
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "derived relation stage requires a base inner source")
			true)
		(define inner_default (source_alias inner_src))
		(define keys '(1))
		(define condition (coalesceNil (qb_where inner) true))
		(define value_for_inner (canonical_column_expr_for_alias inner_default value_expr))
		(define order_for_inner (map (coalesceNil (qb_order inner) '()) (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias inner_default expr) dir)))))
		(define ag (scalar_once_descriptor value_for_inner order_for_inner (qb_offset inner)))
		(define stage_id (concat "derived-once:" (fnv_hash (string (list original_relation alias condition ag)))))
		(define stage (make_group_stage
			stage_id
			inner_src
			'()
			keys
			(list ag)
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) condition)
				(list (quote purpose) (quote scalar_single))
				(list (quote domain) '())
				(list (quote lookup-keys) '())
				(list (quote preserve_empty_domain) true)
				(list (quote null_semantics) (quote scalar))
				(list (quote cardinality_mode) (quote first))
				(list (quote partition_by) '())
				(list (quote physical_max_rows) 1)
				(list (quote on_overflow) (quote ignore)))))
		(define projection (match (qb_fields inner)
			'(title _expr) (list title (scalar_once_value_expr alias ag))
			_ (neumann_fail "untangle_query" "malformed derived relation projection")))
		(list
			(list alias (group_stage_schema stage) (make_stage_output_relation stage_id) false nil)
			projection
			stage))))

(define grouped_derived_relation? (lambda (inner)
	(and (query_block? inner)
		(or (not (empty_list? (qb_group inner)))
			(or (not (nil? (qb_having inner)))
				(query_block_has_aggregates? inner))))))

(define grouped_derived_projection (lambda (stage alias)
	(begin
		(define input_alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define key_names (group_key_cols keys))
		(define ags (gs_aggregates stage))
		(map_assoc (gs_output stage) (lambda (_title expr)
			(replace_group_order_expr input_alias alias keys key_names ags expr))))))

(define make_grouped_derived_stage_source (lambda (src alias inner)
	(begin
		(define stage (make_group_stage_for_query_block inner))
		(define projection (grouped_derived_projection stage alias))
		(define input_alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define key_names (group_key_cols keys))
		(define ags (gs_aggregates stage))
		(define logical_extra_sources (if (query_block? (gs_input stage))
			(map
				(filter (cdr (qb_sources (gs_input stage))) (lambda (extra)
					(source_needed_after_group_stage? input_alias stage extra)))
				(lambda (extra)
					(rewrite_source_for_group_domain input_alias alias keys key_names ags extra)))
			'()))
		(define source (list
			alias
			(group_stage_schema stage)
			(make_stage_output_relation (gs_id stage))
			(source_outer? src)
			(rewrite_derived_ref alias projection (source_join_expr src))))
		(list (cons source logical_extra_sources) projection stage))))

(define ordered_limited_derived_supported? (lambda (inner src)
	(and (query_block? inner)
		(and (not (empty_list? (qb_order inner)))
			(and (not (nil? (qb_limit inner)))
				(and (nil? (source_join_expr src))
					(and (not (source_outer? src))
						(and (empty_list? (qb_group inner))
							(and (nil? (qb_having inner))
								(and (not (query_block_has_aggregates? inner))
									(scalar_source_shape_supported? (qb_sources inner))))))))))))

(define make_ordered_limited_derived_rewrite (lambda (src alias inner)
	(begin
		(if (not (ordered_limited_derived_supported? inner src))
			(neumann_fail "untangle_query" "derived relation stage currently supports grouped relations or single-source ORDER/LIMIT")
			true)
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "ordered derived relation requires a base inner source")
			true)
		(define inner_alias (source_alias inner_src))
		(define derived_src (list alias (source_schema inner_src) (source_relation inner_src) false nil))
		(define helper_sources (requalify_source_join_exprs inner_alias alias (cdr (qb_sources inner))))
		(define order_items (map (qb_order inner) (lambda (item)
			(match item '(expr dir) (list (requalify_single_source_expr inner_alias alias expr) dir)))))
		(define sortcols (order_cols_for_alias derived_src order_items))
		(define sortdirs (window_order_dirs_for_orc order_items))
		(define rn_col (concat "__derived_row_number_" (fnv_hash (serialize (list alias sortcols sortdirs (qb_limit inner) (qb_offset inner))))))
		(define rn_stage (make_window_stage
			(concat "window-row-number:" rn_col)
			derived_src
			rn_col
			sortcols
			sortdirs
			0
			'()
			(list (quote lambda) (list (quote $set))
				(list (quote cons) (symbol "$set") (list (quote list))))
			(list (quote lambda) (list (quote acc) (quote mapped))
				(list (quote begin)
					(list (list (quote car) (quote mapped)) (list (quote +) (quote acc) 1))
					(list (quote +) (quote acc) 1)))
			0
			(list (list (quote kind) (quote ordered-window)))))
		(define rn_expr (list (quote get_column) alias false rn_col false))
		(define offset_value (coalesceNil (qb_offset inner) 0))
		(define upper_bound (+ offset_value (qb_limit inner)))
		(define lower_filter (if (> offset_value 0) (list (quote >) rn_expr offset_value) true))
		(define limit_filter (combine_where lower_filter (list (quote <=) rn_expr upper_bound)))
		(define inner_where (requalify_single_source_expr inner_alias alias (qb_where inner)))
		(list
			(cons derived_src helper_sources)
			(requalify_single_source_fields inner_alias alias (qb_fields inner))
			(combine_where inner_where limit_filter)
			rn_stage))))

(define make_scalar_single_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(if (not (scalar_single_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
			true)
		(define value_expr (query_block_first_expr inner))
		(define inner_src (car (qb_sources inner)))
		(define inner_aliases (source_aliases (qb_sources inner)))
		(define inner_default (source_alias inner_src))
		(define outer_aliases (source_aliases outer_sources))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_aliases outer_aliases term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_aliases outer_aliases (qb_sources inner)))
		(define lookup_pairs (unique_correlation_pairs (merge (list corr_pairs source_corr_pairs))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_aliases outer_aliases term)))))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define value_for_inner (canonical_column_expr_for_alias inner_default value_expr))
		(define ags (scalar_single_aggregates value_for_inner))
		(define value_ag (car ags))
		(define count_ag (cadr ags))
		(define stage_input (if (and (single_source? (qb_sources inner)) (empty_list? (qb_stages inner)))
			inner_src
			(make_query_block
				(qb_schema inner)
				(sources_without_outer_join_terms inner_default inner_aliases outer_aliases (qb_sources inner))
				'()
				condition
				'() nil '() nil nil
				'()
				(qb_stages inner)
				(qb_facts inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "scalar-single:" (fnv_hash (string (list subquery keys outer_domain condition ags)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			ags
			nil
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) stage_condition)
					(list (quote purpose) (quote scalar_single))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) true)
					(list (quote null_semantics) (quote scalar))
					(list (quote cardinality_mode) (quote single_or_error))
					(list (quote partition_by) outer_domain)
					(list (quote physical_max_rows) 2)
					(list (quote on_overflow) (quote error)))
				(btw2025_stage_facts inner outer_sources lookup_pairs)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(stage_source_outer? outer_sources)
			(make_exists_stage_join_condition stage_alias key_names lookup_keys)))
		(list
			(scalar_single_value_expr stage_alias value_ag count_ag)
			(list stage)
			(list source)))))

(define window_aggregate_descriptor (lambda (fn args)
	(match fn
		"COUNT" (list aggregate_count_descriptor)
		"SUM" (match (car args)
			((symbol aggregate) agg_expr (symbol +) 0) (list (list agg_expr (quote +) 0))
			((quote aggregate) agg_expr (quote +) 0) (list (list agg_expr (quote +) 0))
			_ (list (list (car args) (quote +) 0)))
		"MAX" (list (list (car args) (quote max) nil))
		"MIN" (list (list (car args) (quote min) nil))
		"GROUP_CONCAT" (list (list
			(list (quote concat) (car args))
			(list (quote lambda) (list (quote a) (quote b))
				(list (quote if) (list (quote nil?) (quote a))
					(quote b)
					(list (quote concat) (quote a) (nth args 1) (quote b))))
			nil))
		_ (neumann_fail "untangle_query" "window function needs ORC stage"))))

(define window_aggregate_value_expr (lambda (fn args ags stage_alias)
	(match fn
		"COUNT" (list (quote get_column) stage_alias false (aggregate_col_name aggregate_count_descriptor) false)
		"SUM" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"MAX" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"MIN" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"GROUP_CONCAT" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
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

(define invert_window_order_dirs (lambda (dirs)
	(map (coalesceNil dirs '()) (lambda (dir) (not dir)))))

(define window_offset_arg (lambda (args)
	(begin
		(if (> (count args) 2)
			(neumann_fail "untangle_query" "window offset function accepts at most two arguments")
			true)
		(define offset (if (> (count args) 1) (nth args 1) 1))
		(if (not (number? offset))
			(neumann_fail "untangle_query" "window offset must be a numeric literal")
			true)
		(if (< offset 1)
			(neumann_fail "untangle_query" "window offset must be positive")
			true)
		offset)))

(define make_window_offset_mapfn (lambda (partition_cols value_col)
	(if (empty_list? partition_cols)
		(list (quote lambda)
			(list (quote $set) (quote $value))
			(runtime_cons_list_expr (list (quote $set) (quote $value))))
		(list (quote lambda)
			(merge (list (list (quote $set)) (map partition_cols (lambda (colname) (symbol colname))) (list (quote $value))))
			(list (quote cons)
				(if (single_source? partition_cols)
					(symbol (car partition_cols))
					(cons (quote list) (map partition_cols (lambda (colname) (symbol colname)))))
				(runtime_cons_list_expr (list (quote $set) (quote $value))))))))

(define make_window_offset_reducefn (lambda (offset partitioned)
	(if partitioned
		(list (quote lambda) (list (quote acc) (quote mapped))
			(list (quote begin)
				(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
				(list (quote define) (quote payload) (list (quote cdr) (quote mapped)))
				(list (quote define) (quote writer) (list (quote car) (quote payload)))
				(list (quote define) (quote value) (list (quote cadr) (quote payload)))
				(list (quote define) (quote old_queue) (list (quote car) (quote acc)))
				(list (quote define) (quote prev_partition) (list (quote cadr) (quote acc)))
				(list (quote define) (quote same_partition) (list (quote or)
					(list (quote nil?) (quote prev_partition))
					(list (quote equal?) (quote row_partition) (quote prev_partition))))
				(list (quote define) (quote queue) (list (quote if) (quote same_partition) (quote old_queue) (list (quote list))))
				(list (quote define) (quote shifted) (list (quote if) (list (quote <) (list (quote count) (quote queue)) offset) nil (list (quote car) (quote queue))))
				(list (quote writer) (quote shifted))
				(list (quote define) (quote next_queue) (list (quote append) (quote queue) (quote value)))
				(list (quote define) (quote trimmed) (list (quote if) (list (quote >) (list (quote count) (quote next_queue)) offset) (list (quote cdr) (quote next_queue)) (quote next_queue)))
				(list (quote list) (quote trimmed) (quote row_partition))))
		(list (quote lambda) (list (quote acc) (quote mapped))
			(list (quote begin)
				(list (quote define) (quote writer) (list (quote car) (quote mapped)))
				(list (quote define) (quote value) (list (quote cadr) (quote mapped)))
				(list (quote define) (quote shifted) (list (quote if) (list (quote <) (list (quote count) (quote acc)) offset) nil (list (quote car) (quote acc))))
				(list (quote writer) (quote shifted))
				(list (quote define) (quote next_queue) (list (quote append) (quote acc) (quote value)))
				(list (quote if) (list (quote >) (list (quote count) (quote next_queue)) offset) (list (quote cdr) (quote next_queue)) (quote next_queue)))))))

(define make_window_offset_orc_stage_rewrite (lambda (fn args over outer_sources ctx)
	(begin
		(if (or (not (equal? fn "LAG")) (empty_list? args))
			(if (or (not (equal? fn "LEAD")) (empty_list? args))
				(neumann_fail "untangle_query" "window offset function requires a value argument")
				true)
			true)
		(define local_sources (window_stage_sources outer_sources))
		(if (not (single_source? local_sources))
			(neumann_fail "untangle_query" "window offset ORC stage currently supports one base source")
			true)
		(match local_sources
			(cons src _rest) (match src '(alias _schema relation _outer _join)
				(if (not (string? relation))
					(neumann_fail "untangle_query" "window offset ORC stage requires a base source")
					(match over '(partition_exprs order_items) (begin
						(if (empty_list? order_items)
							(neumann_fail "untangle_query" "window offset function requires ORDER BY")
							true)
						(define offset (window_offset_arg args))
						(define value_col (order_column_for_alias src (canonical_column_expr_for_alias alias (car args))))
						(define partition_cols (map partition_exprs (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(if (contains? partition_cols value_col)
							(neumann_fail "untangle_query" "window offset value column must not duplicate partition columns yet")
							true)
						(define order_cols (map (window_order_exprs order_items) (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(define order_dirs (window_order_dirs_for_orc order_items))
						(define sortdirs (merge (list
							(map partition_exprs (lambda (_expr) false))
							(if (equal? fn "LEAD") (invert_window_order_dirs order_dirs) order_dirs))))
						(define sortcols (merge (list partition_cols order_cols)))
						(define col (concat "__orc_" (toLower fn) "_" (fnv_hash (string (list relation alias value_col sortcols sortdirs offset (count partition_exprs))))))
						(define stage (make_window_stage
							(concat "window-" (toLower fn) ":" col)
							src
							col
							sortcols
							sortdirs
							(count partition_exprs)
							(merge (list partition_cols (list value_col)))
							(make_window_offset_mapfn partition_cols value_col)
							(make_window_offset_reducefn offset (not (empty_list? partition_exprs)))
							(if (empty_list? partition_exprs) (list (quote list)) (list (quote list) (list (quote list)) nil))
							(list (list (quote kind) (quote ordered-window)))))
						(qp_any (list
							(list (quote get_column) alias false col false)
							(list stage)
							'())))))
				_ (neumann_fail "untangle_query" "window offset ORC stage requires one source"))))))

(define window_rank_initial_state (lambda (fn)
	(if (equal? fn "DENSE_RANK")
		(list (quote list) 0 nil)
		(list (quote list) 0 0 nil))))

(define make_window_rank_mapfn (lambda (partition_cols order_cols)
	(begin
		(define order_key (if (single_source? order_cols)
			(symbol (car order_cols))
			(cons (quote list) (map order_cols (lambda (colname) (symbol colname))))))
		(if (empty_list? partition_cols)
			(list (quote lambda)
				(cons (quote $set) (map order_cols (lambda (colname) (symbol colname))))
				(runtime_cons_list_expr (list (quote $set) order_key)))
			(list (quote lambda)
				(merge (list
					(list (quote $set))
					(map partition_cols (lambda (colname) (symbol colname)))
					(map order_cols (lambda (colname) (symbol colname)))))
				(list (quote cons)
					(if (single_source? partition_cols)
						(symbol (car partition_cols))
						(cons (quote list) (map partition_cols (lambda (colname) (symbol colname)))))
					(runtime_cons_list_expr (list (quote $set) order_key))))))))

(define make_window_rank_inner_body (lambda (fn inner_state_expr order_key_expr writer_expr)
	(if (equal? fn "DENSE_RANK")
		(list
			(list (quote define) (quote dense_rank) (list (quote car) inner_state_expr))
			(list (quote define) (quote prev_key) (list (quote cadr) inner_state_expr))
			(list (quote define) (quote same_key) (list (quote and)
				(list (quote not) (list (quote nil?) (quote prev_key)))
				(list (quote equal?) order_key_expr (quote prev_key))))
			(list (quote define) (quote new_rank) (list (quote if) (quote same_key) (quote dense_rank) (list (quote +) (quote dense_rank) 1)))
			(list writer_expr (quote new_rank))
			(list (quote list) (quote new_rank) order_key_expr))
		(list
			(list (quote define) (quote rank_value) (list (quote car) inner_state_expr))
			(list (quote define) (quote rownum) (list (quote cadr) inner_state_expr))
			(list (quote define) (quote prev_key) (list (quote car) (list (quote cdr) (list (quote cdr) inner_state_expr))))
			(list (quote define) (quote same_key) (list (quote and)
				(list (quote not) (list (quote nil?) (quote prev_key)))
				(list (quote equal?) order_key_expr (quote prev_key))))
			(list (quote define) (quote new_rownum) (list (quote +) (quote rownum) 1))
			(list (quote define) (quote new_rank) (list (quote if) (quote same_key) (quote rank_value) (quote new_rownum)))
			(list writer_expr (quote new_rank))
			(list (quote list) (quote new_rank) (quote new_rownum) order_key_expr)))))

(define make_window_rank_reducefn (lambda (fn partitioned)
	(begin
		(define init (window_rank_initial_state fn))
		(if partitioned
			(list (quote lambda) (list (quote acc) (quote mapped))
				(cons (quote begin)
					(merge (list
						(list
							(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
							(list (quote define) (quote payload) (list (quote cdr) (quote mapped)))
							(list (quote define) (quote writer) (list (quote car) (quote payload)))
							(list (quote define) (quote order_key) (list (quote cadr) (quote payload)))
							(list (quote define) (quote old_inner) (list (quote car) (quote acc)))
							(list (quote define) (quote prev_partition) (list (quote cadr) (quote acc)))
							(list (quote define) (quote same_partition) (list (quote or)
								(list (quote nil?) (quote prev_partition))
								(list (quote equal?) (quote row_partition) (quote prev_partition))))
							(list (quote define) (quote inner_state) (list (quote if) (quote same_partition) (quote old_inner) init)))
						(make_window_rank_inner_body fn (quote inner_state) (quote order_key) (quote writer))
						(list (list (quote list)
							(if (equal? fn "DENSE_RANK")
								(list (quote list) (quote new_rank) (quote order_key))
								(list (quote list) (quote new_rank) (quote new_rownum) (quote order_key)))
							(quote row_partition)))))))
			(list (quote lambda) (list (quote acc) (quote mapped))
				(cons (quote begin)
					(merge (list
						(list
							(list (quote define) (quote writer) (list (quote car) (quote mapped)))
							(list (quote define) (quote order_key) (list (quote cadr) (quote mapped))))
						(make_window_rank_inner_body fn (quote acc) (quote order_key) (quote writer))))))))))

(define make_window_rank_orc_stage_rewrite (lambda (fn args over outer_sources ctx)
	(begin
		(if (not (empty_list? args))
			(neumann_fail "untangle_query" "rank window function does not accept arguments")
			true)
		(define local_sources (window_stage_sources outer_sources))
		(if (not (single_source? local_sources))
			(neumann_fail "untangle_query" "rank ORC stage currently supports one base source")
			true)
		(match local_sources
			(cons src _rest) (match src '(alias _schema relation _outer _join)
				(if (not (string? relation))
					(neumann_fail "untangle_query" "rank ORC stage requires a base source")
					(match over '(partition_exprs order_items) (begin
						(if (empty_list? order_items)
							(neumann_fail "untangle_query" "rank window function requires ORDER BY")
							true)
						(define partition_cols (map partition_exprs (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(define order_cols (map (window_order_exprs order_items) (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(define sortcols (merge (list partition_cols order_cols)))
						(define sortdirs (merge (list
							(map partition_exprs (lambda (_expr) false))
							(window_order_dirs_for_orc order_items))))
						(define col (concat "__orc_" (toLower fn) "_" (fnv_hash (string (list relation alias sortcols sortdirs (count partition_exprs))))))
						(define stage (make_window_stage
							(concat "window-" (toLower fn) ":" col)
							src
							col
							sortcols
							sortdirs
							(count partition_exprs)
							(merge (list partition_cols order_cols))
							(make_window_rank_mapfn partition_cols order_cols)
							(make_window_rank_reducefn fn (not (empty_list? partition_exprs)))
							(if (empty_list? partition_exprs)
								(window_rank_initial_state fn)
								(list (quote list) (window_rank_initial_state fn) nil))
							(list (list (quote kind) (quote ordered-window)))))
						(qp_any (list
							(list (quote get_column) alias false col false)
							(list stage)
							'())))))
				_ (neumann_fail "untangle_query" "rank ORC stage requires one source"))))))

(define make_row_number_orc_stage_rewrite (lambda (args over outer_sources ctx)
	(begin
		(if (not (empty_list? args))
			(neumann_fail "untangle_query" "ROW_NUMBER does not accept arguments")
			true)
		(define local_sources (window_stage_sources outer_sources))
		(if (not (single_source? local_sources))
			(neumann_fail "untangle_query" "ROW_NUMBER ORC stage currently supports one base source")
			true)
		(match local_sources
			(cons src _rest) (match src '(alias _schema relation _outer _join)
				(if (not (string? relation))
					(neumann_fail "untangle_query" "ROW_NUMBER ORC stage requires a base source")
					(match over '(partition_exprs order_items) (begin
						(if (empty_list? order_items)
							(neumann_fail "untangle_query" "ROW_NUMBER requires ORDER BY for ORC lowering")
							true)
						(define partition_cols (map partition_exprs (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(define order_cols (map (window_order_exprs order_items) (lambda (expr)
							(order_column_for_alias src (canonical_column_expr_for_alias alias expr)))))
						(define sortcols (merge (list partition_cols order_cols)))
						(define sortdirs (merge (list
							(map partition_exprs (lambda (_expr) false))
							(window_order_dirs_for_orc order_items))))
						(define col (concat "__orc_row_number_" (fnv_hash (string (list relation alias sortcols sortdirs (count partition_exprs))))))
						(define stage (make_window_stage
							(concat "window-row-number:" col)
							src
							col
							sortcols
							sortdirs
							(count partition_exprs)
							partition_cols
							(if (empty_list? partition_exprs)
								(list (quote lambda) (list (quote $set))
									(list (quote cons) (symbol "$set") (list (quote list))))
								(list (quote lambda)
									(cons (quote $set) (map partition_cols (lambda (colname) (symbol colname))))
									(list (quote cons)
										(if (single_source? partition_cols)
											(symbol (car partition_cols))
											(cons (quote list) (map partition_cols (lambda (colname) (symbol colname)))))
										(list (quote cons) (symbol "$set") (list (quote list))))))
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
										(list (quote cons)
											(list (quote +) (quote eff_acc) 1)
											(list (quote cons) (quote row_partition) (list (quote list)))))))
							(if (empty_list? partition_exprs) 0 (list (quote list) 0 nil))
							(list (list (quote kind) (quote ordered-window)))))
						(list
							(list (quote get_column) alias false col false)
							(list stage)
							'())))))
			_ (neumann_fail "untangle_query" "ROW_NUMBER ORC stage requires one source")))))

(define make_window_aggregate_stage_rewrite (lambda (fn args over outer_sources ctx)
	(begin
		(define local_sources (window_stage_sources outer_sources))
		(if (not (single_source? local_sources))
			(neumann_fail "untangle_query" "window aggregate stage currently supports one base source")
			true)
		(define src (car local_sources))
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
				(list (quote purpose) (quote window_partition_aggregate))
				(list (quote domain) outer_domain)
				(list (quote lookup-keys) outer_domain)
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote aggregate))
				(list (quote cardinality_mode) (quote many)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define source (list
			stage_alias
			(source_schema src)
			(make_stage_output_relation stage_id)
			true
			(make_exists_stage_join_condition stage_alias key_names outer_domain)))
		(list
			(window_aggregate_value_expr fn canonical_args ags stage_alias)
			(list stage)
			(list source)))))

(define combine_stage_rewrite_results (lambda (head rewritten_args)
	(begin
		(define expr (cons head (map rewritten_args (lambda (item) (nth item 0)))))
		(define stages (merge_unique (map rewritten_args (lambda (item) (nth item 1)))))
		(define sources (merge_unique (map rewritten_args (lambda (item) (nth item 2)))))
		(list expr stages sources))))

(define untangle_scalar_subquery_with_stages (lambda (subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define inner (untangle_query normalized sub_ctx))
		(if (query_block_no_from? inner)
			(list (untangle_zero_domain_subquery (quote inner_select) nil subquery ctx) '() '())
			(if (grouped_scalar_top_supported? inner outer_sources)
				(make_grouped_scalar_top_rewrite inner subquery)
				(if (empty_list? (extract_aggregates (query_block_first_expr inner)))
					(if (scalar_once_supported? inner)
						(make_scalar_once_stage_rewrite inner (list outer_sources subquery))
						(make_scalar_single_stage_rewrite inner (list outer_sources subquery)))
					(make_scalar_aggregate_stage_rewrite inner (list outer_sources subquery))))))))

(define untangle_exists_subquery_with_stages (lambda (subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define inner (untangle_query normalized sub_ctx))
		(if (union_block? inner)
			(make_exists_union_stage_rewrite inner (list outer_sources subquery))
				(if (query_block_no_from? inner)
					(list (untangle_zero_domain_subquery (quote inner_select_exists) nil subquery ctx) '() '())
					(make_exists_stage_rewrite inner (list outer_sources subquery)))))))

(define untangle_not_exists_subquery_with_stages (lambda (subquery outer_sources ctx)
	(begin
		(define rewritten (untangle_exists_subquery_with_stages subquery outer_sources ctx))
		(list
			(list (quote not) (nth rewritten 0))
			(nth rewritten 1)
			(nth rewritten 2)))))

(define untangle_in_subquery_with_stages (lambda (probe subquery outer_sources ctx)
	(begin
		(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
		(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx false))))

(define resolve_in_subquery_with_stages (lambda (rewritten_probe subquery outer_sources ctx semijoin_where)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define inner (untangle_query normalized sub_ctx))
		(if (union_block? inner)
			(make_in_union_stage_rewrite rewritten_probe inner (list outer_sources subquery false semijoin_where))
			(if (query_block_no_from? inner)
				(list (untangle_zero_domain_subquery (quote inner_select_in) rewritten_probe subquery ctx) '() '())
				(make_in_stage_rewrite rewritten_probe inner (list outer_sources subquery false semijoin_where)))))))

(define untangle_not_in_subquery_with_stages (lambda (probe subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define inner (untangle_query normalized sub_ctx))
		(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
		(if (union_block? inner)
			(make_in_union_stage_rewrite rewritten_probe inner (list outer_sources subquery true))
			(if (query_block_no_from? inner)
				(list (untangle_zero_domain_not_in_subquery rewritten_probe subquery ctx) '() '())
				(make_in_stage_rewrite rewritten_probe inner (list outer_sources subquery true)))))))

(define direct_positive_in_term_rewrite (lambda (expr outer_sources ctx)
	(match expr
		((symbol inner_select_in) probe subquery)
		(begin
			(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
			(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx true))
		((quote inner_select_in) probe subquery)
		(begin
			(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
			(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx true))
		_ nil)))

(define where_conjunct_with_stages (lambda (expr outer_sources ctx)
	(begin
		(define direct_in (direct_positive_in_term_rewrite expr outer_sources ctx))
		(if (nil? direct_in)
			(untangle_expr_with_stages expr outer_sources ctx)
			direct_in))))

(define untangle_where_with_stages (lambda (expr outer_sources ctx)
	(begin
		(define terms (split_and_terms (coalesceNil expr true)))
		(define rewritten_terms (map terms (lambda (term)
			(where_conjunct_with_stages term outer_sources ctx))))
		(list
			(combine_where_terms (map rewritten_terms (lambda (item) (nth item 0))) true)
			(merge_unique (map rewritten_terms (lambda (item) (nth item 1))))
			(merge_unique (map rewritten_terms (lambda (item) (nth item 2))))))))

(define btw2025_normalized_subquery_accessing_aliases (lambda (subquery outer_sources)
	(if (query_block? subquery)
		(btw2025_query_block_accessing_aliases subquery outer_sources)
		(if (union_block? subquery)
			(merge_unique (map (union_branches subquery) (lambda (branch)
				(btw2025_subquery_accessing_aliases branch outer_sources))))
			'()))))

(define btw2025_subquery_accessing_aliases (lambda (subquery outer_sources)
	(if (empty_list? outer_sources)
		'()
		(if (or (query_block? subquery) (union_block? subquery))
			(btw2025_normalized_subquery_accessing_aliases subquery outer_sources)
			(btw2025_normalized_subquery_accessing_aliases (normalize_query_ast subquery) outer_sources)))))

(define btw2025_defer_subquery_rewrite? (lambda (subquery outer_sources ctx)
	(and (uctx_get ctx (quote defer-subquery-rewrites) false)
		(not (empty_list? (btw2025_subquery_accessing_aliases subquery outer_sources))))))

(define untangle_expr_with_stages (lambda (expr outer_sources ctx)
	(match expr
		((symbol inner_select) subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote scalar) nil subquery outer_sources) '() '())
			(untangle_scalar_subquery_with_stages subquery outer_sources ctx))
		((quote inner_select) subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote scalar) nil subquery outer_sources) '() '())
			(untangle_scalar_subquery_with_stages subquery outer_sources ctx))
		((symbol inner_select_exists) subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote exists) nil subquery outer_sources) '() '())
			(untangle_exists_subquery_with_stages subquery outer_sources ctx))
		((quote inner_select_exists) subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote exists) nil subquery outer_sources) '() '())
			(untangle_exists_subquery_with_stages subquery outer_sources ctx))
		((symbol inner_select_in) probe subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote in) probe subquery outer_sources) '() '())
			(untangle_in_subquery_with_stages probe subquery outer_sources ctx))
			((quote inner_select_in) probe subquery)
			(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
				(list (make_dependent_subquery_marker (quote in) probe subquery outer_sources) '() '())
				(untangle_in_subquery_with_stages probe subquery outer_sources ctx))
			((symbol not) ((symbol inner_select_exists) subquery))
			(untangle_not_exists_subquery_with_stages subquery outer_sources ctx)
			((quote not) ((quote inner_select_exists) subquery))
			(untangle_not_exists_subquery_with_stages subquery outer_sources ctx)
			((symbol not) ((quote inner_select_exists) subquery))
			(untangle_not_exists_subquery_with_stages subquery outer_sources ctx)
			((quote not) ((symbol inner_select_exists) subquery))
			(untangle_not_exists_subquery_with_stages subquery outer_sources ctx)
			((symbol not) ((symbol inner_select_in) probe subquery))
			(untangle_not_in_subquery_with_stages probe subquery outer_sources ctx)
		((quote not) ((quote inner_select_in) probe subquery))
		(untangle_not_in_subquery_with_stages probe subquery outer_sources ctx)
		((symbol not) ((quote inner_select_in) probe subquery))
		(untangle_not_in_subquery_with_stages probe subquery outer_sources ctx)
		((quote not) ((symbol inner_select_in) probe subquery))
		(untangle_not_in_subquery_with_stages probe subquery outer_sources ctx)
		((symbol window_func) fn args over)
		(begin
			(define window_sources (uctx_get ctx (quote local-sources) outer_sources))
			(if (equal? fn "ROW_NUMBER")
				(make_row_number_orc_stage_rewrite args over window_sources ctx)
				(if (or (equal? fn "LAG") (equal? fn "LEAD"))
					(make_window_offset_orc_stage_rewrite fn args over window_sources ctx)
					(if (or (equal? fn "RANK") (equal? fn "DENSE_RANK"))
						(make_window_rank_orc_stage_rewrite fn args over window_sources ctx)
						(make_window_aggregate_stage_rewrite fn args over window_sources ctx)))))
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

(define untangle_expr_list_with_stages (lambda (exprs outer_sources ctx)
	(match (coalesceNil exprs '())
		(cons expr rest) (begin
			(define rewritten (untangle_expr_with_stages expr outer_sources ctx))
			(define tail (untangle_expr_list_with_stages rest outer_sources ctx))
			(list
				(cons (nth rewritten 0) (nth tail 0))
				(merge (list (nth rewritten 1) (nth tail 1)))
				(merge (list (nth rewritten 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define untangle_order_with_stages (lambda (order_items outer_sources ctx)
	(match (coalesceNil order_items '())
		(cons item rest) (begin
			(define rewritten_item (match item
				'(expr dir) (begin
					(define rewritten_expr (untangle_expr_with_stages expr outer_sources ctx))
					(list (list (nth rewritten_expr 0) dir) (nth rewritten_expr 1) (nth rewritten_expr 2)))
				_ (list item '() '())))
			(define tail (untangle_order_with_stages rest outer_sources ctx))
			(list
				(cons (nth rewritten_item 0) (nth tail 0))
				(merge (list (nth rewritten_item 1) (nth tail 1)))
				(merge (list (nth rewritten_item 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define untangle_fields (lambda (fields ctx)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
		(cons title (cons (untangle_expr expr ctx) (untangle_fields rest ctx)))
		_ '())))

(define btw2025_resolve_dependent_subquery (lambda (marker ctx)
	(begin
		(define kind (dep_subquery_kind marker))
		(define probe (dep_subquery_probe marker))
		(define subquery (dep_subquery_query marker))
		(define outer_sources (dep_subquery_outer_sources marker))
		(match kind
			(symbol scalar)
			(untangle_scalar_subquery_with_stages subquery outer_sources ctx)
			(symbol exists)
			(untangle_exists_subquery_with_stages subquery outer_sources ctx)
			(symbol in)
			(begin
				(define probe_result (btw2025_decorrelate_expr_with_stages probe ctx))
				(define in_result (resolve_in_subquery_with_stages (nth probe_result 0) subquery outer_sources ctx false))
				(list
					(nth in_result 0)
					(merge_unique (list (nth probe_result 1) (nth in_result 1)))
					(merge_unique (list (nth probe_result 2) (nth in_result 2)))))
			_ (neumann_fail "untangle_query" "unknown dependent subquery marker")))))

(define btw2025_decorrelate_expr_with_stages (lambda (expr ctx)
	(match expr
		((symbol dependent-subquery) _kind _probe _subquery _outer_sources)
		(btw2025_resolve_dependent_subquery expr ctx)
		((quote dependent-subquery) _kind _probe _subquery _outer_sources)
		(btw2025_resolve_dependent_subquery expr ctx)
		(cons head tail)
		(combine_stage_rewrite_results head (map tail (lambda (item)
			(btw2025_decorrelate_expr_with_stages item ctx))))
		_ (list expr '() '()))))

(define btw2025_decorrelate_fields_with_stages (lambda (fields ctx)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (begin
			(define rewritten (btw2025_decorrelate_expr_with_stages expr ctx))
			(define tail (btw2025_decorrelate_fields_with_stages rest ctx))
			(list
				(cons title (cons (nth rewritten 0) (nth tail 0)))
				(merge_unique (list (nth rewritten 1) (nth tail 1)))
				(merge_unique (list (nth rewritten 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define btw2025_decorrelate_order_with_stages (lambda (order_items ctx)
	(match (coalesceNil order_items '())
		(cons item rest) (begin
			(define rewritten_item (match item
				'(expr dir) (begin
					(define rewritten_expr (btw2025_decorrelate_expr_with_stages expr ctx))
					(list (list (nth rewritten_expr 0) dir) (nth rewritten_expr 1) (nth rewritten_expr 2)))
				_ (list item '() '())))
			(define tail (btw2025_decorrelate_order_with_stages rest ctx))
			(list
				(cons (nth rewritten_item 0) (nth tail 0))
				(merge_unique (list (nth rewritten_item 1) (nth tail 1)))
				(merge_unique (list (nth rewritten_item 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define btw2025_decorrelate_expr_list_with_stages (lambda (exprs ctx)
	(match (coalesceNil exprs '())
		(cons expr rest) (begin
			(define rewritten (btw2025_decorrelate_expr_with_stages expr ctx))
			(define tail (btw2025_decorrelate_expr_list_with_stages rest ctx))
			(list
				(cons (nth rewritten 0) (nth tail 0))
				(merge_unique (list (nth rewritten 1) (nth tail 1)))
				(merge_unique (list (nth rewritten 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define btw2025_decorrelate_source_with_stages (lambda (src ctx)
	(begin
		(define rewritten_join (btw2025_decorrelate_expr_with_stages (source_join_expr src) ctx))
		(list
			(merge_unique (list (nth rewritten_join 2) (list (source_with_join_expr src (nth rewritten_join 0)))))
			(nth rewritten_join 1)
			'()))))

(define btw2025_decorrelate_sources_with_stages (lambda (sources ctx)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define rewritten_src (btw2025_decorrelate_source_with_stages src ctx))
			(define tail (btw2025_decorrelate_sources_with_stages rest ctx))
			(list
				(merge_unique (list (nth rewritten_src 0) (nth tail 0)))
				(merge_unique (list (nth rewritten_src 1) (nth tail 1)))
				(merge_unique (list (nth rewritten_src 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define btw2025_decorrelate_query_block (lambda (block ctx)
	(begin
		(define source_result (btw2025_decorrelate_sources_with_stages (qb_sources block) ctx))
		(define sources (nth source_result 0))
		(define source_stages (nth source_result 1))
		(define source_stage_sources (nth source_result 2))
		(define where_result (btw2025_decorrelate_expr_with_stages (qb_where block) ctx))
		(define field_result (btw2025_decorrelate_fields_with_stages (qb_fields block) ctx))
		(define group_result (btw2025_decorrelate_expr_list_with_stages (qb_group block) ctx))
		(define having_result (btw2025_decorrelate_expr_with_stages (qb_having block) ctx))
		(define order_result (btw2025_decorrelate_order_with_stages (qb_order block) ctx))
		(define hidden_result (btw2025_decorrelate_fields_with_stages (qb_hidden block) ctx))
		(make_query_block
			(qb_schema block)
			(merge_unique (list sources source_stage_sources (nth where_result 2) (nth field_result 2) (nth group_result 2) (nth having_result 2) (nth order_result 2) (nth hidden_result 2)))
			(nth field_result 0)
			(nth where_result 0)
			(nth group_result 0)
			(nth having_result 0)
			(nth order_result 0)
			(qb_limit block)
			(qb_offset block)
			(nth hidden_result 0)
			(merge_unique (list (qb_stages block) source_stages (nth where_result 1) (nth field_result 1) (nth group_result 1) (nth having_result 1) (nth order_result 1) (nth hidden_result 1)))
			(qb_facts block)))))

(define field_expr_by_title (lambda (fields title ignorecase)
	(match (coalesceNil fields '())
		(cons current_title (cons expr rest)) (if (if ignorecase (equal?? current_title title) (equal? current_title title))
			expr
			(field_expr_by_title rest title ignorecase))
		_ nil)))

(define derived_star_ref? (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		((quote get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		_ false)))

(define rewrite_derived_ref (lambda (alias projection expr)
	(match expr
		((symbol get_column) tblvar ignorecase col _json_path) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col ignorecase) expr)
			expr)
		((quote get_column) tblvar ignorecase col _json_path) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col ignorecase) expr)
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

(define requalify_source_join_expr (lambda (old_alias new_alias src)
	(source_with_join_expr src
		(requalify_single_source_expr old_alias new_alias (source_join_expr src)))))

(define requalify_source_join_exprs (lambda (old_alias new_alias sources)
	(map (coalesceNil sources '()) (lambda (src)
		(requalify_source_join_expr old_alias new_alias src)))))

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

(define first_projection_expr (lambda (fields)
	(match (coalesceNil fields '())
		(cons _title (cons expr _rest)) expr
		_ nil)))

(define null_extend_projection_fields (lambda (fields)
	(begin
		(define presence (first_projection_expr fields))
		(if (nil? presence)
			fields
			(map_assoc (coalesceNil fields '()) (lambda (_title expr)
				(list (quote if) (list (quote nil?) presence) nil expr)))))))

(define combine_where (lambda (a b)
	(begin
		(define aa (coalesceNil a true))
		(define bb (coalesceNil b true))
		(if (equal? aa true)
			bb
			(if (equal? bb true)
				aa
				(list (quote and) aa bb))))))

(define derived_block_needs_operator? (lambda (block)
	(or (not (empty_list? (qb_group block)))
		(or (not (nil? (qb_having block)))
			(or (not (nil? (qb_limit block)))
				(or (not (nil? (qb_offset block)))
					(query_block_has_aggregates? block)))))))

(define untangle_flattened_base_source (lambda (src ctx)
	(list
		(source_alias src)
		(source_schema src)
		(normalize_query_ast (source_relation src))
		(source_outer? src)
		(source_join_expr src))))

(define flatten_source_list (lambda (sources ctx)
	(if (empty_list? sources)
		(list '() '() '() '())
		(begin
			(define src (car sources))
			(define rest (cdr sources))
			(define relation (normalize_query_ast (source_relation src)))
			(define tail (flatten_source_list rest ctx))
			(define tail_sources (nth tail 0))
			(define tail_rewrites (nth tail 1))
			(define tail_wheres (nth tail 2))
			(define tail_stages (nth tail 3))
			(if (string? relation)
				(begin
					(list
						(cons (untangle_flattened_base_source src ctx) tail_sources)
						tail_rewrites
						tail_wheres
						tail_stages))
				(if (union_block? relation)
					(neumann_fail "untangle_query" "FROM union-block needs union logical lowering before source flattening")
					(begin
						(define alias (source_alias src))
						(define inner (untangle_query relation ctx))
						(if (derived_block_needs_operator? inner)
							(if (grouped_derived_relation? inner)
								(begin
									(define staged (make_grouped_derived_stage_source src alias inner))
									(list
										(merge (list (nth staged 0) tail_sources))
										(cons (list alias (nth staged 1)) tail_rewrites)
										tail_wheres
										(cons (nth staged 2) (merge (list (qb_stages inner) tail_stages)))))
								(if (ordered_limited_derived_supported? inner src)
									(begin
										(define staged (make_ordered_limited_derived_rewrite src alias inner))
										(list
											(merge (list (nth staged 0) tail_sources))
											(cons (list alias (nth staged 1)) tail_rewrites)
											(cons (nth staged 2) tail_wheres)
											(cons (nth staged 3) (merge (list (qb_stages inner) tail_stages)))))
									(if (or (source_outer? src) (not (nil? (source_join_expr src))))
										(neumann_fail "untangle_query" "derived relation stage with outer join needs relation-unit lowering")
										(begin
											(define staged (make_limited_derived_stage_source alias inner relation))
											(list
												(cons (nth staged 0) tail_sources)
												(cons (list alias (nth staged 1)) tail_rewrites)
												tail_wheres
												(cons (nth staged 2) (merge (list (qb_stages inner) tail_stages))))))))
							(begin
								(define inner_sources (coalesceNil (qb_sources inner) '()))
								(if (empty_list? inner_sources)
									(if (or (source_outer? src) (not (nil? (source_join_expr src))))
										(neumann_fail "untangle_query" "zero-source derived JOIN needs constant-relation support")
										(list
											(rewrite_sources_join_for_derived alias (qb_fields inner) tail_sources)
											(cons (list alias (qb_fields inner)) tail_rewrites)
											(cons (qb_where inner) tail_wheres)
											(merge (list (qb_stages inner) tail_stages))))
									(if (equal? (count inner_sources) 1)
										(begin
											(define only_inner (car inner_sources))
											(define inner_alias (source_alias only_inner))
											(define projection (requalify_single_source_fields inner_alias alias (qb_fields inner)))
											(define effective_projection (if (source_outer? src)
												(null_extend_projection_fields projection)
												projection))
											(define inner_where (requalify_single_source_expr inner_alias alias (qb_where inner)))
											(define outer_join (rewrite_derived_ref alias effective_projection (source_join_expr src)))
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
													(rewrite_sources_join_for_derived alias effective_projection tail_sources))
												(cons (list alias effective_projection) tail_rewrites)
												(if (nil? parent_condition) tail_wheres (cons parent_condition tail_wheres))
												(merge (list (qb_stages inner) tail_stages))))
										(if (or (source_outer? src) (not (nil? (source_join_expr src))))
											(if (scalar_source_shape_supported? inner_sources)
												(begin
													(define only_inner (car inner_sources))
													(define inner_alias (source_alias only_inner))
													(define projection (requalify_single_source_fields inner_alias alias (qb_fields inner)))
													(define effective_projection (if (source_outer? src)
														(null_extend_projection_fields projection)
														projection))
													(define inner_where (requalify_single_source_expr inner_alias alias (qb_where inner)))
													(define outer_join (rewrite_derived_ref alias effective_projection (source_join_expr src)))
													(define joined_condition (combine_where inner_where outer_join))
													(define derived_base_source (list
														alias
														(source_schema only_inner)
														(source_relation only_inner)
														(source_outer? src)
														joined_condition))
													(define derived_stage_sources (requalify_source_join_exprs inner_alias alias (cdr inner_sources)))
													(list
														(merge (list
															(list derived_base_source)
															derived_stage_sources
															(rewrite_sources_join_for_derived alias effective_projection tail_sources)))
														(cons (list alias effective_projection) tail_rewrites)
														tail_wheres
														(merge (list (qb_stages inner) tail_stages))))
												(neumann_fail "untangle_query" "multi-source derived JOIN needs relation-unit lowering"))
											(list
												(merge (list inner_sources (rewrite_sources_join_for_derived alias (qb_fields inner) tail_sources)))
												(cons (list alias (qb_fields inner)) tail_rewrites)
												(cons (qb_where inner) tail_wheres)
												(merge (list (qb_stages inner) tail_stages)))))))))
))))))

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

(define untangle_source_join_expr_with_stages (lambda (src outer_sources ctx)
	(match (source_join_expr src)
		nil (list src '() '())
		join_expr (begin
			(define rewritten (untangle_expr_with_stages join_expr outer_sources ctx))
			(list
				(source_with_join_expr src (nth rewritten 0))
				(nth rewritten 1)
				(nth rewritten 2))))))

(define untangle_source_join_exprs_with_stages (lambda (sources outer_sources ctx)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define head (untangle_source_join_expr_with_stages src outer_sources ctx))
			(define tail (untangle_source_join_exprs_with_stages rest outer_sources ctx))
			(list
				(merge (list (nth head 2) (list (nth head 0)) (nth tail 0)))
				(merge_unique (list (nth head 1) (nth tail 1)))
				(merge_unique (list (nth head 2) (nth tail 2)))))
		_ (list '() '() '()))))

/* ------------------------------------------------------------------------- */
/* Top-down untangle                                                          */

(define single_union_source (lambda (block)
	(match (coalesceNil (qb_sources block) '())
		'(src) (begin
			(define relation (normalize_query_ast (source_relation src)))
			(if (union_block? relation)
				(if (or (source_outer? src) (not (nil? (source_join_expr src))))
					nil
					src)
				nil))
		_ nil)))

(define first_embedded_union_source (lambda (sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(begin
				(define relation (normalize_query_ast (source_relation src)))
				(if (and (union_block? relation)
					(and (not (source_outer? src)) (nil? (source_join_expr src))))
					src
					nil))))
		nil)))

(define union_wrapper_rewrite_allowed? (lambda (block)
	(and (empty_list? (qb_group block))
		(and (nil? (qb_having block))
			(and (not (query_block_has_aggregates? block))
				(empty_list? (qb_hidden block)))))))

(define union_count_projection_title (lambda (fields)
	(match (coalesceNil fields '())
		'(title expr) (match expr
			'(fn seed reduce init) (if (and (equal? fn (quote aggregate))
				(and (equal? seed 1)
					(and (equal? reduce (quote +)) (equal? init 0))))
				title
				nil)
			_ nil)
		_ nil)))

(define rewrite_query_block_union_count (lambda (block src ctx)
	(begin
		(define title (union_count_projection_title (qb_fields block)))
		(if (or (nil? title)
			(or (not (equal? (coalesceNil (qb_where block) true) true))
				(or (not (empty_list? (qb_order block)))
					(or (not (nil? (qb_limit block)))
						(or (not (nil? (qb_offset block)))
							(not (empty_list? (qb_hidden block))))))))
			nil
			(make_query_block
				(qb_schema block)
				'()
				(list title (list (quote union_count) (untangle_union_block (normalize_query_ast (source_relation src)) ctx)))
				true
				'() nil '() nil nil '() '() '())))))

(define wrap_union_branch_query (lambda (outer alias branch)
	(begin
		(define inner (normalize_query_ast branch))
		(if (not (query_block? inner))
			(neumann_fail "untangle_query" "FROM UNION wrapper expects query-block branches")
			true)
		(define projection (qb_fields inner))
		(make_query_block
			(qb_schema inner)
			(qb_sources inner)
			(rewrite_derived_fields alias projection (qb_fields outer))
			(combine_where (qb_where inner) (rewrite_derived_ref alias projection (qb_where outer)))
			(qb_group inner)
			(qb_having inner)
			'()
			(qb_limit inner)
			(qb_offset inner)
			(qb_hidden inner)
			(qb_stages inner)
			(qb_facts inner)))))

(define rewrite_query_block_over_union_source (lambda (block src)
	(begin
		(if (not (union_wrapper_rewrite_allowed? block))
			nil
			(begin
				(define relation (normalize_query_ast (source_relation src)))
				(make_union_block
					(union_mode relation)
					(map (union_branches relation) (lambda (branch)
						(wrap_union_branch_query block (source_alias src) branch)))
					(if (empty_list? (qb_order block)) (union_order relation) (qb_order block))
					(coalesceNil (qb_limit block) (union_limit relation))
					(coalesceNil (qb_offset block) (union_offset relation))
					(union_facts relation)))))))

(define replace_union_source_branch (lambda (sources union_src branch)
	(map (coalesceNil sources '()) (lambda (src)
		(if (equal?? (source_alias src) (source_alias union_src))
			(source_with_relation src branch)
			src)))))

(define wrap_embedded_union_branch_query (lambda (outer union_src branch)
	(make_query_block
		(qb_schema outer)
		(replace_union_source_branch (qb_sources outer) union_src (normalize_query_ast branch))
		(qb_fields outer)
		(qb_where outer)
		(qb_group outer)
		(qb_having outer)
		(qb_order outer)
		(qb_limit outer)
		(qb_offset outer)
		(qb_hidden outer)
		(qb_stages outer)
		(qb_facts outer))))

(define rewrite_query_block_over_embedded_union_source (lambda (block src)
	(begin
		(if (not (union_wrapper_rewrite_allowed? block))
			nil
			(begin
				(define relation (normalize_query_ast (source_relation src)))
				(make_union_block
					(union_mode relation)
					(map (union_branches relation) (lambda (branch)
						(wrap_embedded_union_branch_query block src branch)))
					(union_order relation)
					(union_limit relation)
					(union_offset relation)
					(union_facts relation)))))))

(define untangle_query_block (lambda (block ctx)
	(begin
		(define child_ctx (make_uctx ctx
			(list
				(list (quote compile-budget-ms) 1000)
				(list (quote operator-model) (quote combined)))))
		(define union_src (single_union_source block))
		(define union_rewrite (if (nil? union_src) nil (rewrite_query_block_over_union_source block union_src)))
		(if (not (nil? union_rewrite))
			(untangle_union_block union_rewrite child_ctx)
			(begin
				(define union_count_rewrite (if (nil? union_src) nil (rewrite_query_block_union_count block union_src child_ctx)))
				(if (not (nil? union_count_rewrite))
					union_count_rewrite
					(begin
						(define embedded_union_src (first_embedded_union_source (qb_sources block)))
						(define embedded_union_rewrite (if (nil? embedded_union_src) nil (rewrite_query_block_over_embedded_union_source block embedded_union_src)))
						(if (not (nil? embedded_union_rewrite))
							(untangle_union_block embedded_union_rewrite child_ctx)
							(begin
								(define flattened_sources (flatten_source_list (qb_sources block) child_ctx))
								(define sources (nth flattened_sources 0))
								(define rewrites (nth flattened_sources 1))
								(define source_where_terms (nth flattened_sources 2))
								(define source_stages (nth flattened_sources 3))
								(define inherited_outer_sources (uctx_get child_ctx (quote outer-sources) '()))
								(define expr_outer_sources (merge (list inherited_outer_sources sources)))
								(define expr_ctx (make_uctx child_ctx (list
									(list (quote outer-sources) expr_outer_sources)
									(list (quote local-sources) sources))))
								(define source_join_result (untangle_source_join_exprs_with_stages sources expr_outer_sources expr_ctx))
								(define untangled_sources (nth source_join_result 0))
								(define source_join_stage_sources (nth source_join_result 2))
								(define joined_expr_outer_sources (merge (list inherited_outer_sources untangled_sources source_join_stage_sources)))
								(define joined_expr_ctx (make_uctx child_ctx (list
									(list (quote outer-sources) joined_expr_outer_sources)
									(list (quote local-sources) (merge_unique (list untangled_sources source_join_stage_sources))))))
								(define rewritten_where (combine_where_terms source_where_terms (rewrite_derived_ref_chain rewrites (qb_where block))))
								(if (expr_contains_window? rewritten_where)
									(neumann_fail "untangle_query" "window function is not allowed in WHERE")
									true)
								(define where_result (untangle_where_with_stages rewritten_where joined_expr_outer_sources joined_expr_ctx))
								(define field_result (untangle_fields_with_stages (rewrite_derived_fields_chain rewrites (qb_fields block)) joined_expr_outer_sources joined_expr_ctx))
								(define having_result (untangle_expr_with_stages (rewrite_derived_ref_chain rewrites (qb_having block)) joined_expr_outer_sources joined_expr_ctx))
								(define stage_sources (merge_unique (list (nth where_result 2) (nth field_result 2) (nth having_result 2))))
								(define group_result (untangle_expr_list_with_stages
									(map (coalesceNil (qb_group block) '()) (lambda (item) (rewrite_derived_ref_chain rewrites item)))
									joined_expr_outer_sources
									joined_expr_ctx))
								(define order_result (untangle_order_with_stages
									(rewrite_derived_order_chain rewrites (qb_order block))
									joined_expr_outer_sources
									joined_expr_ctx))
								(define hidden_result (untangle_fields_with_stages (rewrite_derived_fields_chain rewrites (qb_hidden block)) joined_expr_outer_sources joined_expr_ctx))
								(define delayed_block (make_query_block
									(qb_schema block)
									(merge_unique (list untangled_sources source_join_stage_sources stage_sources (nth group_result 2) (nth order_result 2) (nth hidden_result 2)))
									(nth field_result 0)
									(nth where_result 0)
									(nth group_result 0)
									(nth having_result 0)
									(nth order_result 0)
									(qb_limit block)
									(qb_offset block)
									(nth hidden_result 0)
									(merge_unique (list source_stages (qb_stages block) (nth source_join_result 1) (nth where_result 1) (nth field_result 1) (nth group_result 1) (nth having_result 1) (nth order_result 1) (nth hidden_result 1)))
									(qb_facts block)))
								(btw2025_decorrelate_query_block delayed_block child_ctx))))))))))

(define untangle_union_block (lambda (block ctx)
	(make_union_block
		(if (equal? (union_mode block) (quote distinct)) (quote union_distinct) (union_mode block))
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
		(if (node_contains_physical_helper? node)
			(neumann_fail phase "physical helper relation survived before build_queryplan")
			true)
		node)))

(define untangle_query_term (lambda (query ctx)
	(begin
		(define root (require_unnested_node "untangle_query" (untangle_query query ctx)))
		(define ir (make_ir (if (union_block? root) (quote union) (quote select))
			root
			(if (query_block? root) (qb_stages root) '())
			(make_uctx ctx (list
				(list (quote compile-budget-ms) 1000)
				(list (quote operator-model) (quote combined))))
			(quote rows)))
		(require_flat_stage_dependencies "untangle_query" (normalize_stage_dependencies ir)))))

/* ------------------------------------------------------------------------- */
/* Reorder/optimise scaffold                                                  */

(define planner_literal_value (lambda (expr)
	(match expr
		((symbol session) key) (try
			(lambda () (session key))
			(lambda (_e) nil))
		((quote session) key) (planner_literal_value (list (quote session) key))
		_ expr)))

(define planner_string_expr_value (lambda (expr)
	(begin
		(define value (planner_literal_value expr))
		(if (string? value)
			value
			(match expr
				((symbol concat) a b) (begin
					(define av (planner_string_expr_value a))
					(define bv (planner_string_expr_value b))
					(if (and (string? av) (string? bv)) (concat av bv) nil))
				((quote concat) a b) (planner_string_expr_value (list (quote concat) a b))
				_ nil)))))

(define like_pattern_core (lambda (pattern)
	(replace (replace (coalesceNil pattern "") "%" "") "_" "")))

(define broad_like_pattern? (lambda (pattern)
	(if (not (string? pattern))
		false
		(begin
			(define core (like_pattern_core pattern))
			(<= (strlen core) 1)))))

(define expr_contains_broad_text_match? (lambda (expr)
	(match expr
		((symbol strlike) _value pattern _collation)
		(broad_like_pattern? (planner_string_expr_value pattern))
		((quote strlike) _value pattern _collation)
		(expr_contains_broad_text_match? (list (quote strlike) _value pattern _collation))
		(cons head tail)
		(reduce tail (lambda (found item)
			(or found (expr_contains_broad_text_match? item)))
			(expr_contains_broad_text_match? head))
		_ false)))

(define stage_input_contains_broad_membership_filter? (lambda (input)
	(if (union_block? input)
		(reduce (union_branches input) (lambda (found branch)
			(or found (stage_input_contains_broad_membership_filter? branch)))
			false)
		(if (query_block? input)
			(expr_contains_broad_text_match? (qb_where input))
			false))))

(define candidate_stage_broad? (lambda (stage)
	(or (equal? (qassoc_get (gs_facts stage) (quote selectivity_class) nil) (quote broad))
		(stage_input_contains_broad_membership_filter? (gs_input stage)))))

(define source_order_limit_driver? (lambda (src order_items limit_value)
	(and (query_limit_active? nil limit_value)
		(order_items_belong_to_source? src order_items))))

(define planner_table_row_count (lambda (schema relation)
	(try
		(lambda ()
			(define live_count (reduce (show schema true) (lambda (found row)
				(if (not (nil? found))
					found
					(if (equal?? (row "name") relation)
						(row "row_count")
						nil)))
				nil))
			(if (and (not (nil? live_count)) (> live_count 0))
				live_count
				(match (show schema relation)
					(cons col _rest) (col "RowEstimate")
					_ live_count)))
		(lambda (_e) nil))))

(define planner_source_filter_estimate (lambda (src condition max_rows)
	(if (not (source_is_base_table? src))
		nil
		(try
			(lambda ()
				(begin
					(define alias (source_alias src))
					(define filtercols (extract_columns_for_alias src condition))
					(define filter_expr (list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(list (quote optimize) (lower_column_expr_for_alias src condition))))
					(scan_selectivity_estimate
						(session "__memcp_tx")
						(table (source_schema src) (source_relation src))
						filtercols
						(eval filter_expr)
						max_rows)))
			(lambda (_e) nil)))))

(define planner_source_row_count (lambda (src)
	(if (source_is_base_table? src)
		(planner_table_row_count (source_schema src) (source_relation src))
		nil)))

(define source_join_present? (lambda (src)
	(begin
		(define join_expr (source_join_expr src))
		(not (or (nil? join_expr) (equal? join_expr true))))))

(define source_reorder_estimate (lambda (src)
	(list
		(source_alias src)
		(list (quote relation) (source_relation src))
		(list (quote row_count) (planner_source_row_count src))
		(list (quote outer_join) (source_outer? src))
		(list (quote join_filter) (source_join_present? src)))))

(define left_join_strategy_options (lambda (src)
	(if (not (source_outer? src))
		nil
		(begin
			(define rows (planner_source_row_count src))
			(list
				(source_alias src)
				(list (quote preferred) (if (and (not (nil? rows)) (< rows 1000))
					(quote subscan)
					(quote tempcol_materialize_reusable)))
				(list (quote alternatives) (list
					(quote subscan)
					(quote group_cache_read)
					(quote tempcol_materialize_reusable)))
				(list (quote row_count) rows))))))

(define query_block_reorder_telemetry (lambda (block)
	(list
		(list (quote source_estimates) (map (qb_sources block) source_reorder_estimate))
		(list (quote left_join_plan_options) (filter
			(map (qb_sources block) left_join_strategy_options)
			(lambda (item) (not (nil? item))))))))

(define query_block_needs_reorder_facts? (lambda (block)
	(or (not (empty_list? (qb_stages block)))
		(reduce (qb_sources block) (lambda (needed src)
			(or needed
				(or (source_outer? src)
					(source_join_present? src))))
			false))))

(define planner_add_estimates (lambda (values)
	(reduce (coalesceNil values '()) (lambda (acc value)
		(if (nil? value) acc (+ acc value)))
		0)))

(define planner_query_block_input_rows (lambda (block)
	(planner_add_estimates (map (qb_sources block) planner_source_row_count))))

(define planner_stage_input_rows (lambda (input)
	(if (union_block? input)
		(planner_add_estimates (map (union_branches input) planner_stage_input_rows))
		(if (query_block? input)
			(planner_query_block_input_rows input)
			nil))))

(define planner_stage_filter_estimate (lambda (input max_rows)
	(if (union_block? input)
		(begin
			(define branch_estimates (map (union_branches input) (lambda (branch)
				(planner_stage_filter_estimate branch max_rows))))
			(define available (filter branch_estimates (lambda (estimate) (not (nil? estimate)))))
			(if (empty_list? available)
				nil
				(begin
					(define rows (planner_add_estimates (map available (lambda (estimate)
						(qassoc_get estimate (quote rows) nil)))))
					(define input_rows (planner_add_estimates (map available (lambda (estimate)
						(qassoc_get estimate (quote input) nil)))))
					(define capped (or (>= rows max_rows)
						(reduce available (lambda (found estimate)
							(or found (qassoc_get estimate (quote capped) false)))
							false)))
					(list
						(list (quote rows) rows)
						(list (quote capped) capped)
						(list (quote input) input_rows)))))
		(if (query_block? input)
			(if (single_source? (qb_sources input))
				(begin
					(define src (car (qb_sources input)))
					(planner_source_filter_estimate src (combine_where (qb_where input) (source_join_expr src)) max_rows))
				nil)
			nil))))

(define candidate_reorder_telemetry (lambda (stage sources block)
	(begin
		(define driver (if (empty_list? sources) nil (car sources)))
		(define driver_rows (if (nil? driver) nil (planner_source_row_count driver)))
		(define candidate_rows (planner_stage_input_rows (gs_input stage)))
		(define candidate_estimate (planner_stage_filter_estimate (gs_input stage) 512))
		(define estimate_rows (qassoc_get candidate_estimate (quote rows) nil))
		(define estimate_capped (qassoc_get candidate_estimate (quote capped) false))
		(define estimate_input (qassoc_get candidate_estimate (quote input) nil))
		(define estimate_ratio_broad (and
			(and (not (nil? estimate_rows)) (and (not (nil? estimate_input)) (> estimate_input 0)))
			(>= (* estimate_rows 4) estimate_input)))
		(define class (if (or estimate_capped (or estimate_ratio_broad (and (nil? candidate_estimate) (candidate_stage_broad? stage)))) (quote broad) (quote selective)))
		(define ordered_driver (if (nil? driver) false (source_order_limit_driver? driver (qb_order block) (qb_limit block))))
		(list
			(list (quote membership_selectivity_class) class)
			(list (quote membership_driver_rows) driver_rows)
			(list (quote membership_candidate_input_rows) candidate_rows)
			(list (quote membership_candidate_estimated_rows) estimate_rows)
			(list (quote membership_candidate_estimate_capped) estimate_capped)
			(list (quote membership_candidate_estimate_input) estimate_input)
			(list (quote membership_order_limit) (qb_limit block))
			(list (quote membership_order_limit_driver) ordered_driver)
			(list (quote membership_cost_reason) (if (and (equal? class (quote broad)) ordered_driver)
				(quote broad_membership_preserve_driver_order_limit)
				(quote selective_membership_build_candidate_keyset)))))))

(define stage_reorder_strategy (lambda (stage)
	(match (qassoc_get (gs_facts stage) (quote purpose) nil)
		(symbol exists) (quote group_cache_read)
		(symbol not_exists) (quote group_cache_read)
		(symbol in_membership) (quote group_cache_read)
		(symbol in_candidate) (quote candidate_keyset)
		(symbol scalar_aggregate) (quote group_cache_read)
		(symbol scalar_single) (if (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote first))
			(quote subscan_partition_limit)
			(quote group_cache_read))
		_ (quote group_cache_read))))

(define stage_reorder_telemetry (lambda (stage)
	(list
		(list (quote group_stage_strategy) (stage_reorder_strategy stage))
		(list (quote group_input_rows) (planner_stage_input_rows (gs_input stage)))
		(list (quote group_domain_count) (count (gs_domain stage)))
		(list (quote group_key_count) (count (gs_keys stage)))
		(list (quote group_plan_options) (list
			(quote subscan)
			(quote group_cache_read)
			(quote tempcol_materialize_reusable))))))

(define group_stage_with_reorder_facts (lambda (stage)
	(make_group_stage
		(gs_id stage)
		(gs_input stage)
		(gs_domain stage)
		(gs_keys stage)
		(gs_aggregates stage)
		(gs_having stage)
		(gs_output stage)
		(gs_order stage)
		(gs_limit stage)
		(gs_offset stage)
		(merge (list (stage_reorder_telemetry stage) (gs_facts stage))))))

(define candidate_reorder_strategy (lambda (stage sources block)
	(begin
		(define driver (if (empty_list? sources) nil (car sources)))
		(define candidate_estimate (planner_stage_filter_estimate (gs_input stage) 512))
		(define estimate_rows (qassoc_get candidate_estimate (quote rows) nil))
		(define estimate_input (qassoc_get candidate_estimate (quote input) nil))
		(define estimate_ratio_broad (and
			(and (not (nil? estimate_rows)) (and (not (nil? estimate_input)) (> estimate_input 0)))
			(>= (* estimate_rows 4) estimate_input)))
		(define broad (or
			(qassoc_get candidate_estimate (quote capped) false)
			(or estimate_ratio_broad
				(and (nil? candidate_estimate) (candidate_stage_broad? stage)))))
		(if (and (not (nil? driver))
			(source_order_limit_driver? driver (qb_order block) (qb_limit block)))
			(quote driver_order_membership_probe)
			(quote candidate_keyset)))))

(define query_block_with_reorder_facts (lambda (block facts)
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
		(qb_stages block)
		(merge (list facts (qb_facts block))))))

(define driver_membership_probe_expr (lambda (stage probe)
	(list (quote and)
		(list (quote not) (list (quote nil?) probe))
		(list (quote driver_membership_probe) stage probe))))

(define dml_preserve_driver_membership_probe (lambda (fallback_schema expr)
	(match expr
		((symbol driver_membership_probe) stage probe)
		(list (quote dml_driver_membership_probe) fallback_schema stage probe)
		((quote driver_membership_probe) stage probe)
		(list (quote dml_driver_membership_probe) fallback_schema stage probe)
		(cons head tail) (cons head (map tail (lambda (item) (dml_preserve_driver_membership_probe fallback_schema item))))
		_ expr)))

(define candidate_stage_without_source (lambda (stages stage_id)
	(filter (coalesceNil stages '()) (lambda (stage)
		(not (and (group_stage? stage) (equal? (gs_id stage) stage_id)))))))

(define candidate_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(and (not (nil? stage))
				(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote in_candidate)))))))

(define exists_recset_stage? (lambda (stage)
	(and (group_stage? stage)
		(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote exists))
			(and (equal? (qassoc_get (gs_facts stage) (quote presence_only) false) true)
				(and (single_source? (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
					(and (single_source? (gs_keys stage))
						(source_is_base_table? (gs_input stage)))))))))

(define exists_recset_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(exists_recset_stage? stage)))))

(define first_candidate_source (lambda (stages sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (candidate_stage_output_source? stages src) src nil)))
		nil)))

(define negated_term? (lambda (term)
	(match term
		((symbol not) _expr) true
		((quote not) _expr) true
		_ false)))

(define positive_condition_refs_source? (lambda (default_alias alias condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(or found
			(and (not (negated_term? term))
				(expr_refs_alias_after_group? default_alias alias term))))
		false)))

(define first_exists_recset_source (lambda (stages sources default_alias condition)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (and (exists_recset_stage_output_source? stages src)
				(positive_condition_refs_source? default_alias (source_alias src) condition))
				src
				nil)))
		nil)))

(define without_source_alias (lambda (sources alias)
	(filter (coalesceNil sources '()) (lambda (src)
		(not (equal? (source_alias src) alias))))))

(define attach_candidate_join_condition (lambda (default_alias candidate_alias condition sources)
	(match (coalesceNil sources '())
		(cons src rest) (if (and (not (equal? (source_alias src) candidate_alias))
			(expr_refs_alias_after_group? default_alias (source_alias src) condition))
			(cons (source_with_join_expr src (combine_where (source_join_expr src) condition)) rest)
			(cons src (attach_candidate_join_condition default_alias candidate_alias condition rest)))
		_ '())))

(define strip_condition_for_source_alias (lambda (default_alias alias condition)
	(combine_where_terms
		(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
			(not (expr_refs_alias_after_group? default_alias alias term))))
		true)))

(define reorder_candidate_sources (lambda (stages sources)
	(begin
		(define candidate (first_candidate_source stages sources))
		(if (nil? candidate)
			sources
			(begin
				(define candidate_alias (source_alias candidate))
				(define candidate_join (coalesceNil (source_join_expr candidate) true))
				(define rest (without_source_alias sources candidate_alias))
				(if (equal? candidate_join true)
					(cons candidate rest)
					(cons
						(source_with_join_expr candidate true)
						(attach_candidate_join_condition
							(if (empty_list? rest) candidate_alias (source_alias (car rest)))
							candidate_alias
							candidate_join
							rest))))))))

(define reorder_query_block_with_candidate_strategy (lambda (block)
	(begin
		(if (equal? (qassoc_get (qb_facts block) (quote dml) false) true)
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
				(map (qb_stages block) join_reorder_stage)
				(qb_facts block))
			(begin
		(define sources (qb_sources block))
		(define candidate (first_candidate_source (qb_stages block) sources))
		(if (nil? candidate)
			(begin
				(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
				(define exists_src (first_exists_recset_source (qb_stages block) sources default_alias (qb_where block)))
				(if (nil? exists_src)
					(if (query_block_needs_reorder_facts? block)
						(query_block_with_reorder_facts
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
								(map (qb_stages block) join_reorder_stage)
								(qb_facts block))
							(query_block_reorder_telemetry block))
						block)
					(begin
						(define stage (stage_by_id (qb_stages block) (stage_output_relation_id (source_relation exists_src))))
						(define stage_id (gs_id stage))
						(define probe (car (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
						(define base_where (strip_condition_for_source_alias
							default_alias
							(source_alias exists_src)
							(qb_where block)))
						(query_block_with_reorder_facts
							(make_query_block
								(qb_schema block)
								(without_source_alias sources (source_alias exists_src))
								(qb_fields block)
								(combine_where base_where (driver_membership_probe_expr stage probe))
								(qb_group block)
								(qb_having block)
								(qb_order block)
								(qb_limit block)
								(qb_offset block)
								(qb_hidden block)
								(map (candidate_stage_without_source (qb_stages block) stage_id) join_reorder_stage)
								(qb_facts block))
							(merge (list
								(query_block_reorder_telemetry block)
								(list (list (quote exists_plan_strategy) (quote recset_project_join)))))))))
			(begin
				(define stage (stage_by_id (qb_stages block) (stage_output_relation_id (source_relation candidate))))
				(define strategy (candidate_reorder_strategy stage sources block))
				(define facts (merge (list
					(query_block_reorder_telemetry block)
					(cons
						(list (quote membership_plan_strategy) strategy)
						(candidate_reorder_telemetry stage sources block)))))
				(match strategy
					(symbol driver_order_membership_probe) (begin
						(define stage_id (gs_id stage))
						(define probe (car (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
						(query_block_with_reorder_facts
							(make_query_block
								(qb_schema block)
								(without_source_alias sources (source_alias candidate))
								(qb_fields block)
								(combine_where (qb_where block) (driver_membership_probe_expr stage probe))
								(qb_group block)
								(qb_having block)
								(qb_order block)
								(qb_limit block)
								(qb_offset block)
								(qb_hidden block)
								(map (candidate_stage_without_source (qb_stages block) stage_id) join_reorder_stage)
								(qb_facts block))
							facts))
					_ (make_query_block
						(qb_schema block)
						(reorder_candidate_sources (qb_stages block) sources)
						(qb_fields block)
						(qb_where block)
						(qb_group block)
						(qb_having block)
						(qb_order block)
						(qb_limit block)
						(qb_offset block)
						(qb_hidden block)
						(map (qb_stages block) join_reorder_stage)
						(merge (list facts (qassoc_set (qb_facts block) (quote default_alias) (source_alias (car sources)))))))))))))))

(define join_reorder_node (lambda (node)
	(if (query_block? node)
		(reorder_query_block_with_candidate_strategy node)
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) join_reorder_node)
				(union_order node)
				(union_limit node)
				(union_offset node)
				(union_facts node))
			node))))

(define join_reorder_stage (lambda (stage)
	(if (group_stage? stage)
		(group_stage_with_reorder_facts
			(make_group_stage
				(gs_id stage)
				(join_reorder_node (gs_input stage))
				(gs_domain stage)
				(gs_keys stage)
				(gs_aggregates stage)
				(gs_having stage)
				(gs_output stage)
				(gs_order stage)
				(gs_limit stage)
				(gs_offset stage)
				(gs_facts stage)))
		stage)))

(define query_block_without_logical_stages (lambda (block)
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

(define normalize_stage_dependencies_node (lambda (node)
	(if (query_block? node)
		(normalize_stage_dependencies_query_block node)
		(if (union_block? node)
			(normalize_stage_dependencies_union_block node)
			(list node '())))))

(define normalize_stage_dependencies_stage (lambda (stage)
	(if (group_stage? stage)
		(begin
			(define input_result (normalize_stage_dependencies_node (gs_input stage)))
			(define input_node (nth input_result 0))
			(define nested_stages (nth input_result 1))
			(define normalized_input (if (query_block? input_node)
				(query_block_without_logical_stages input_node)
				input_node))
			(list
				(make_group_stage
					(gs_id stage)
					normalized_input
					(gs_domain stage)
					(gs_keys stage)
					(gs_aggregates stage)
					(gs_having stage)
					(gs_output stage)
					(gs_order stage)
					(gs_limit stage)
					(gs_offset stage)
					(gs_facts stage))
				nested_stages))
		(list stage '()))))

(define normalize_stage_dependencies_stages (lambda (stages)
	(match (coalesceNil stages '())
		(cons stage rest) (begin
			(define head (normalize_stage_dependencies_stage stage))
			(define tail (normalize_stage_dependencies_stages rest))
			(merge_unique (list (nth head 1) (list (nth head 0)) tail)))
		_ '())))

(define normalize_stage_dependencies_query_block (lambda (block)
	(if (empty_list? (qb_stages block))
		(list block '())
		(begin
			(define normalized_stages (normalize_stage_dependencies_stages (qb_stages block)))
			(define normalized_block (make_query_block
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
				normalized_stages
				(qb_facts block)))
			(list normalized_block normalized_stages)))))

(define normalize_stage_dependencies_union_block (lambda (block)
	(begin
		(define branch_results (map (union_branches block) normalize_stage_dependencies_node))
		(list
			(make_union_block
				(union_mode block)
				(map branch_results (lambda (item) (nth item 0)))
				(union_order block)
				(union_limit block)
				(union_offset block)
				(union_facts block))
			(merge_unique (map branch_results (lambda (item) (nth item 1))))))))

(define normalize_stage_dependencies (lambda (ir)
	(begin
		(define root_result (normalize_stage_dependencies_node (ir_root ir)))
		(make_ir
			(ir_kind ir)
			(nth root_result 0)
			(if (query_block? (nth root_result 0)) (qb_stages (nth root_result 0)) (nth root_result 1))
			(ir_context_of ir)
			(ir_return ir)))))

(define node_contains_nested_stage_input? (lambda (node)
	(if (query_block? node)
		(reduce (qb_stages node) (lambda (found stage)
			(or found (node_contains_nested_stage_input? stage)))
			false)
		(if (union_block? node)
			(reduce (union_branches node) (lambda (found branch)
				(or found (node_contains_nested_stage_input? branch)))
				false)
			(if (group_stage? node)
				(begin
					(define input (gs_input node))
					(or
						(and (query_block? input) (not (empty_list? (qb_stages input))))
						(node_contains_nested_stage_input? input)))
				false)))))

(define require_flat_stage_dependencies (lambda (phase ir)
	(begin
		(if (or (node_contains_nested_stage_input? (ir_root ir))
			(reduce (ir_stages ir) (lambda (found stage)
				(or found (node_contains_nested_stage_input? stage)))
				false))
			(neumann_fail phase "nested stage input survived recursive decorrelation")
			true)
		ir)))

(define join_reorder (lambda (ir)
	(make_ir
		(ir_kind ir)
		(join_reorder_node (ir_root ir))
		(map (ir_stages ir) join_reorder_stage)
		(ir_context_of ir)
		(ir_return ir))))

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

(define source_alias_matches? (lambda (src default_alias tblvar tbl_ignorecase)
	(begin
		(define ref_alias (resolve_column_alias tblvar default_alias))
		(if tbl_ignorecase
			(equal?? ref_alias (source_alias src))
			(equal? ref_alias (source_alias src))))))

(define source_for_alias (lambda (sources default_alias tblvar tbl_ignorecase)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (source_alias_matches? src default_alias tblvar tbl_ignorecase) src nil)))
		nil)))

(define resolve_physical_column_name (lambda (src col col_ignorecase)
	(if (not (string? col))
		col
		(if (not (source_is_base_table? src))
			col
			(begin
				(define cols (get_schema (source_schema src) (source_relation src)))
				(define exact (reduce cols (lambda (found row)
					(if (not (nil? found))
						found
						(if (equal? (row "Field") col) (row "Field") nil)))
					nil))
				(if (not (nil? exact))
					exact
					(coalesceNil
						(reduce cols (lambda (found row)
							(if (not (nil? found))
								found
								(if (equal?? (row "Field") col) (row "Field") nil)))
							nil)
						col)))))))

(define extract_columns_for_alias (lambda (src expr)
	(match expr
		((symbol dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_alias src probe)
		((quote dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_alias src probe)
		((symbol scalar_first_probe) stage _requested_col)
		(merge_unique (map (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (key)
			(extract_columns_for_alias src key))))
		((quote scalar_first_probe) stage requested_col)
		(extract_columns_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase) (list (resolve_physical_column_name src col col_ignorecase)) '())
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase) (list (resolve_physical_column_name src col col_ignorecase)) '())
		(cons head tail) (merge_unique (map tail (lambda (item) (extract_columns_for_alias src item))))
		_ '())))

(define lower_column_expr_for_alias (lambda (src expr)
	(match expr
		((symbol dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((quote dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col)
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (symbol (concat (resolve_column_alias tblvar (source_alias src)) "." (resolve_physical_column_name src col col_ignorecase)))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (symbol (concat (resolve_column_alias tblvar (source_alias src)) "." (resolve_physical_column_name src col col_ignorecase)))
		(cons head tail) (cons head (map tail (lambda (item) (lower_column_expr_for_alias src item))))
		_ expr)))

(define scalar_first_probe_parts (lambda (ag)
	(match ag
		'(((symbol scalar_order_value) value_expr order_exprs dirs offset_value) _reduce _neutral)
		(list value_expr order_exprs dirs offset_value)
		'(((quote scalar_order_value) value_expr order_exprs dirs offset_value) _reduce _neutral)
		(list value_expr order_exprs dirs offset_value)
		'(((symbol scalar_order_value) value_expr order_expr dir) _reduce _neutral)
		(list value_expr (list order_expr) (list dir) 0)
		'(((quote scalar_order_value) value_expr order_expr dir) _reduce _neutral)
		(list value_expr (list order_expr) (list dir) 0)
		'(value_expr _reduce _neutral)
		(list value_expr '() '() 0)
		_ (neumann_fail "build_queryplan" "scalar-first probe expects aggregate descriptor"))))

(define scalar_first_probe_aggregate (lambda (stage requested_col)
	(reduce (gs_aggregates stage) (lambda (found ag)
		(if (not (nil? found))
			found
			(if (equal? (aggregate_col_name ag) requested_col) ag nil)))
		nil)))

(define scalar_first_probe_key_terms (lambda (sources default_alias src keys lookup_keys)
	(begin
		(define alias (source_alias src))
		(map (produceN (count keys)) (lambda (i)
			(list (quote equal??)
				(lower_column_expr_for_alias src (nth keys i))
				(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))))

(define lower_scalar_first_probe_expr (lambda (sources default_alias stage requested_col)
	(begin
		(if (not (scalar_first_probe_stage? stage))
			(neumann_fail "build_queryplan" "scalar-first probe requires scalar_single first stage")
			true)
		(define src (gs_input stage))
		(define keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(if (not (equal? (count keys) (count lookup_keys)))
			(neumann_fail "build_queryplan" "scalar-first probe key/domain mismatch")
			true)
		(define ag (scalar_first_probe_aggregate stage requested_col))
		(if (nil? ag)
			(neumann_fail "build_queryplan" "scalar-first probe references unknown aggregate column")
			true)
		(define parts (scalar_first_probe_parts ag))
		(define value_expr (nth parts 0))
		(define order_exprs (nth parts 1))
		(define dirs (nth parts 2))
		(define offset_value (nth parts 3))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define condition_cols (extract_columns_for_alias src condition))
		(define key_cols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define order_cols (merge_unique (map order_exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define value_cols (extract_columns_for_alias src value_expr))
		(define filtercols (merge_unique (list condition_cols key_cols order_cols)))
		(define mapcols (merge_unique (list value_cols)))
		(list (quote scan_order)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(list (quote optimize)
					(cons (quote and)
						(cons
							(lower_column_expr_for_alias src condition)
							(scalar_first_probe_key_terms sources default_alias src keys lookup_keys)))))
			(cons (quote list) order_cols)
			(cons (quote list) dirs)
			0
			(coalesceNil offset_value 0)
			1
			(cons (quote list) mapcols)
			(list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(lower_column_expr_for_alias src value_expr))
			(scalar_once_reduce_first)
			nil
			false))))

(define extract_columns_for_join_alias (lambda (sources default_alias alias expr)
	(match expr
		((symbol driver_membership_probe) _stage probe)
		(extract_columns_for_join_alias sources default_alias alias probe)
		((quote driver_membership_probe) _stage probe)
		(extract_columns_for_join_alias sources default_alias alias probe)
		((symbol dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_join_alias sources default_alias alias probe)
		((quote dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_join_alias sources default_alias alias probe)
		((symbol scalar_first_probe) stage _requested_col)
		(merge_unique (map (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (key)
			(extract_columns_for_join_alias sources default_alias alias key))))
		((quote scalar_first_probe) stage requested_col)
		(extract_columns_for_join_alias sources default_alias alias (list (quote scalar_first_probe) stage requested_col))
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (source_for_alias sources default_alias tblvar tbl_ignorecase))
			(if (and (not (nil? src)) (equal?? (source_alias src) alias))
				(list (resolve_physical_column_name src col col_ignorecase))
				'()))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (source_for_alias sources default_alias tblvar tbl_ignorecase))
			(if (and (not (nil? src)) (equal?? (source_alias src) alias))
				(list (resolve_physical_column_name src col col_ignorecase))
				'()))
		(cons head tail) (merge_unique (map tail (lambda (item) (extract_columns_for_join_alias sources default_alias alias item))))
		_ '())))

(define lower_column_expr_for_join (lambda (sources default_alias expr)
	(match expr
		((symbol driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((quote driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((symbol dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr sources default_alias fallback_schema stage probe)
		((quote dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr sources default_alias fallback_schema stage probe)
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col)
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (source_for_alias sources default_alias tblvar tbl_ignorecase))
			(if (nil? src)
				(symbol (concat (resolve_column_alias tblvar default_alias) "." col))
				(symbol (concat (source_alias src) "." (resolve_physical_column_name src col col_ignorecase)))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (source_for_alias sources default_alias tblvar tbl_ignorecase))
			(if (nil? src)
				(symbol (concat (resolve_column_alias tblvar default_alias) "." col))
				(symbol (concat (source_alias src) "." (resolve_physical_column_name src col col_ignorecase)))))
		(cons head tail) (cons head (map tail (lambda (item) (lower_column_expr_for_join sources default_alias item))))
		_ expr)))

(define canonical_column_expr_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase) (list (quote get_column) (resolve_column_alias tblvar alias) false col false)
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase) (list (quote get_column) (resolve_column_alias tblvar alias) false col false)
		(cons head tail) (cons head (map tail (lambda (item) (canonical_column_expr_for_alias alias item))))
		_ expr)))

(define order_column_for_alias (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(resolve_physical_column_name src col col_ignorecase)
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(resolve_physical_column_name src col col_ignorecase)
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		_ (neumann_fail "build_queryplan" "ORDER BY expression lowering needs a computed sort column"))))

(define order_cols_for_alias (lambda (src order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (order_column_for_alias src expr)
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define lower_scan_order_sort_expr_for_alias (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(symbol (resolve_physical_column_name src col col_ignorecase))
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(lower_scan_order_sort_expr_for_alias src (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons head (map tail (lambda (item) (lower_scan_order_sort_expr_for_alias src item))))
		_ expr)))

(define scan_order_sort_column_for_alias (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(resolve_physical_column_name src col col_ignorecase)
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(scan_order_sort_column_for_alias src (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		_ (begin
			(define cols (extract_columns_for_alias src expr))
			(list (quote lambda)
				(map cols (lambda (col) (symbol col)))
				(lower_scan_order_sort_expr_for_alias src expr))))))

(define scan_order_sort_columns_for_alias (lambda (src order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (scan_order_sort_column_for_alias src expr)
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define order_dirs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) dir
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define order_expr_belongs_to_source? (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase _col _col_ignorecase) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
		((quote get_column) tblvar tbl_ignorecase _col _col_ignorecase) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
		(cons _head tail) (reduce tail (lambda (ok item) (and ok (order_expr_belongs_to_source? src item))) true)
		_ true)))

(define order_items_belong_to_source? (lambda (src order_items)
	(reduce (coalesceNil order_items '()) (lambda (ok item)
		(and ok (match item
			'(expr _dir) (order_expr_belongs_to_source? src expr)
			_ false)))
		true)))

(define direct_order_expr_for_source? (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase _col _col_ignorecase) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
		((quote get_column) tblvar tbl_ignorecase _col _col_ignorecase) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
		_ false)))

(define direct_order_items_for_source? (lambda (src order_items)
	(reduce (coalesceNil order_items '()) (lambda (ok item)
		(and ok (match item
			'(expr _dir) (direct_order_expr_for_source? src expr)
			_ false)))
		true)))

(define lower_grouped_scalar_top_expr (lambda (stage)
	(begin
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define key_names (group_key_cols keys))
		(define carrier (group_stage_carrier stage))
		(define schema (group_carrier_schema carrier))
		(define grouptbl (group_carrier_relation carrier))
		(define group_src (list grouptbl schema grouptbl false nil))
		(define value_expr (replace_group_expr alias grouptbl keys key_names ags (first_projection_expr (gs_output stage))))
		(define replaced_order (map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr dir) (list (replace_group_order_expr alias grouptbl keys key_names ags expr) dir)))))
		(define ordercols (order_cols_for_alias group_src replaced_order))
		(define valuecols (extract_columns_for_alias group_src value_expr))
		(list (quote scan_order)
			'(session "__memcp_tx")
			(list (quote table) schema grouptbl)
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) ordercols)
			(cons (quote list) (order_dirs replaced_order))
			0
			0
			1
			(cons (quote list) valuecols)
			(list (quote lambda)
				(map valuecols (lambda (col) (symbol (concat grouptbl "." col))))
				(lower_column_expr_for_alias group_src value_expr))
			(scalar_once_reduce_first)
			nil
			false))))

(define lower_union_count_branch_expr (lambda (branch)
	(begin
		(if (not (union_ordered_branch_supported? branch))
			(neumann_fail "build_queryplan" "COUNT over UNION currently supports simple single-source branches")
			true)
		(define src (car (qb_sources branch)))
		(define alias (source_alias src))
		(define condition (combine_where (qb_where branch) (source_join_expr src)))
		(define filtercols (extract_columns_for_alias src condition))
		(list (quote scan)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat alias "." col))))
				(list (quote optimize) (lower_column_expr_for_alias src condition)))
			(quoted_runtime_list '())
			(list (quote lambda) '() 1)
			(quote +)
			0
			nil
			false))))

(define lower_union_count_expr (lambda (block)
	(if (not (equal? (union_mode block) (quote all)))
		(neumann_fail "build_queryplan" "COUNT over UNION currently supports UNION ALL only")
		(cons (quote +) (map (union_branches block) lower_union_count_branch_expr)))))

(define driver_membership_probe_branch_expr (lambda (sources default_alias branch probe)
	(begin
		(if (not (and (query_block? branch) (single_source? (qb_sources branch))))
			(neumann_fail "build_queryplan" "driver membership probe expects simple query-block branches")
			true)
		(define src (car (qb_sources branch)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "driver membership probe expects base table branches")
			true)
		(define rhs_expr (query_block_first_expr branch))
		(define condition (combine_where (qb_where branch) (source_join_expr src)))
		(define filtercols (merge_unique (list
			(extract_columns_for_alias src condition)
			(extract_columns_for_alias src rhs_expr))))
		(define lowered_condition (list (quote and)
			(lower_column_expr_for_alias src condition)
			(list (quote equal??)
				(lower_column_expr_for_alias src rhs_expr)
				(lower_column_expr_for_join sources default_alias probe))))
		(list (quote scan)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(list (quote optimize) lowered_condition))
			(quoted_runtime_list '())
			(list (quote lambda) '() 1)
			(quote +)
			0
			nil
			false))))

(define lower_driver_membership_probe_expr (lambda (sources default_alias stage probe)
	(begin
		(define input (gs_input stage))
		(if (not (union_block? input))
			(neumann_fail "build_queryplan" "driver membership probe currently expects UNION candidate input")
			true)
		(list (quote >)
			(cons (quote +) (map (union_branches input) (lambda (branch)
				(driver_membership_probe_branch_expr sources default_alias branch probe))))
			0))))

(define lower_dml_driver_membership_probe_expr (lambda (sources default_alias fallback_schema stage _probe)
	(begin
		(define keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(if (< (count keys) (count lookup_keys))
			(neumann_fail "build_queryplan" "DML membership probe key/domain mismatch")
			true)
		(define key_names (group_key_cols keys))
		(define filter_key_names (map (produceN (count lookup_keys)) (lambda (i) (nth key_names i))))
		(define carrier_schema (coalesceNil (group_stage_carrier_schema stage) fallback_schema))
		(define key_terms (map (produceN (count lookup_keys)) (lambda (i)
			(list (quote equal??)
				(symbol (nth key_names i))
				(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))
		(list (quote scan_exists)
			'(session "__memcp_tx")
			(list (quote table) carrier_schema (group_stage_carrier_relation stage))
			(cons (quote list) filter_key_names)
			(list (quote lambda)
				(map filter_key_names symbol)
				(list (quote optimize) (if (empty_list? key_terms) true (cons (quote and) key_terms))))))))

(define direct_column_name_for_alias (lambda (src expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(resolve_physical_column_name src col col_ignorecase)
			nil)
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (direct_column_name_for_alias src (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		_ nil)))

(define driver_membership_probe_term (lambda (expr)
	(match expr
		((symbol driver_membership_probe) stage probe) (list stage probe)
		((quote driver_membership_probe) stage probe) (list stage probe)
		_ nil)))

(define driver_membership_nil_guard? (lambda (probe expr)
	(match expr
		((symbol not) ((symbol nil?) guarded)) (equal? guarded probe)
		((quote not) ((quote nil?) guarded)) (equal? guarded probe)
		((symbol not) ((quote nil?) guarded)) (equal? guarded probe)
		((quote not) ((symbol nil?) guarded)) (equal? guarded probe)
		_ false)))

(define driver_membership_for_source (lambda (src condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(if (not (nil? found))
			found
			(begin
				(define probe_term (driver_membership_probe_term term))
				(if (nil? probe_term)
					nil
					(begin
						(define probe_col (direct_column_name_for_alias src (nth probe_term 1)))
						(if (nil? probe_col)
							nil
							(list (nth probe_term 0) (nth probe_term 1) probe_col term)))))))
		nil)))

(define strip_driver_membership_for_source (lambda (src condition membership)
	(if (nil? membership)
		condition
		(begin
			(define probe (nth membership 1))
			(define marker_term (nth membership 3))
			(combine_where_terms
				(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
					(and (not (equal? term marker_term))
						(not (driver_membership_nil_guard? probe term)))))
				true)))))

(define recset_project_join_branch_expr (lambda (target_src branch target_col)
	(begin
		(if (not (and (query_block? branch) (single_source? (qb_sources branch))))
			(neumann_fail "build_queryplan" "recset project membership expects simple query-block branches")
			true)
		(define src (car (qb_sources branch)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "recset project membership expects base table branches")
			true)
		(define source_col (direct_column_name_for_alias src (query_block_first_expr branch)))
		(if (nil? source_col)
			(neumann_fail "build_queryplan" "recset project membership expects direct RHS column")
			true)
		(define alias (source_alias src))
		(define condition (combine_where (qb_where branch) (source_join_expr src)))
		(define filtercols (extract_columns_for_alias src condition))
		(list (quote recset_project_join)
			'(session "__memcp_tx")
			(list (quote scan_recset)
				'(session "__memcp_tx")
				(source_table_expr src)
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols (lambda (col) (symbol (concat alias "." col))))
					(list (quote optimize) (lower_column_expr_for_alias src condition))))
			(quoted_runtime_list (list source_col))
			(source_table_expr target_src)
			(quoted_runtime_list (list target_col))))))

(define recset_project_join_expr_for_membership (lambda (src membership)
	(begin
		(define stage (nth membership 0))
		(define target_col (nth membership 2))
		(define input (gs_input stage))
		(if (union_block? input)
			(begin
				(define projected (map (union_branches input) (lambda (branch)
					(recset_project_join_branch_expr src branch target_col))))
				(if (single_source? projected)
					(car projected)
					(list (quote recset_union) (cons (quote list) projected))))
			(if (exists_recset_stage? stage)
				(begin
					(define source_col (direct_column_name_for_alias input (car (gs_keys stage))))
					(if (nil? source_col)
						nil
						(begin
							(define alias (source_alias input))
							(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
							(define filtercols (extract_columns_for_alias input condition))
							(list (quote recset_project_join)
								'(session "__memcp_tx")
								(list (quote scan_recset)
									'(session "__memcp_tx")
									(source_table_expr input)
									(cons (quote list) filtercols)
									(list (quote lambda)
										(map filtercols (lambda (col) (symbol (concat alias "." col))))
										(list (quote optimize) (lower_column_expr_for_alias input condition))))
								(quoted_runtime_list (list source_col))
								(source_table_expr src)
								(quoted_runtime_list (list target_col))))))
				nil)))))

(define lower_scalar_marker_expr (lambda (expr)
	(match expr
		((symbol grouped_scalar_top) stage) (lower_grouped_scalar_top_expr stage)
		((quote grouped_scalar_top) stage) (lower_grouped_scalar_top_expr stage)
		((symbol union_count) block) (lower_union_count_expr block)
		((quote union_count) block) (lower_union_count_expr block)
		(cons head tail) (cons head (map tail lower_scalar_marker_expr))
		_ expr)))

(define aggregate_expr? (lambda (expr)
	(match expr
		((symbol aggregate) _expr _reduce _neutral) true
		((quote aggregate) _expr _reduce _neutral) true
		_ false)))

(define aggregate_count_descriptor (list 1 (quote +) 0))

(define count_distinct_reduce (lambda ()
	(list (quote lambda) (list (quote a) (quote b))
		(list (quote begin)
			(list (quote define) (quote aa) (list (quote if)
				(list (quote list?) (quote a))
				(quote a)
				(list (quote if) (list (quote nil?) (quote a)) (list (quote list)) (list (quote cons) (quote a) (list (quote list))))))
			(list (quote define) (quote bb) (list (quote if)
				(list (quote list?) (quote b))
				(quote b)
				(list (quote if) (list (quote nil?) (quote b)) (list (quote list)) (list (quote cons) (quote b) (list (quote list))))))
			(list (quote merge_unique)
				(list (quote cons) (quote aa)
					(list (quote cons) (quote bb) (list (quote list)))))))))

(define count_distinct_combine (lambda ()
	(count_distinct_reduce)))

(define count_distinct_count_reduce (lambda ()
	(list (quote lambda) (list (quote a) (quote b))
		(list (quote count)
			(list (quote begin)
				(list (quote define) (quote aa) (list (quote if)
					(list (quote list?) (quote a))
					(quote a)
					(list (quote if) (list (quote nil?) (quote a)) (list (quote list)) (list (quote cons) (quote a) (list (quote list))))))
				(list (quote define) (quote bb) (list (quote if)
					(list (quote list?) (quote b))
					(quote b)
					(list (quote if) (list (quote nil?) (quote b)) (list (quote list)) (list (quote cons) (quote b) (list (quote list))))))
				(list (quote merge_unique)
					(list (quote cons) (quote aa)
						(list (quote cons) (quote bb) (list (quote list))))))))))

(define count_distinct_descriptor (lambda (expr)
	(list expr (count_distinct_reduce) (list (quote list)))))

(define count_distinct_descriptor? (lambda (ag)
	(match ag
		'(_expr reduce _neutral) (equal? reduce (count_distinct_reduce))
		_ false)))

(define aggregate_shard_combine (lambda (ag)
	(if (count_distinct_descriptor? ag)
		(count_distinct_combine)
		nil)))

(define query_aggregate_shard_combine (lambda (ag)
	(if (count_distinct_descriptor? ag)
		(count_distinct_count_reduce)
		(aggregate_shard_combine ag))))

(define aggregate_map_value_expr (lambda (ag expr)
	(if (count_distinct_descriptor? ag)
		(list (quote cons) expr (list (quote list)))
		expr)))

(define extract_aggregates (lambda (expr)
	(match expr
		((symbol count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((quote count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((symbol aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		((quote aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		(cons head tail) (merge_unique (map tail extract_aggregates))
		_ '())))

(define stage_aggregates_for_fields (lambda (fields)
	(merge_unique (extract_assoc fields (lambda (_title expr) (extract_aggregates expr))))))

(define expr_has_aggregates? (lambda (expr)
	(not (empty_list? (extract_aggregates expr)))))

(define external_column_refs_for_alias (lambda (default_alias expr)
	(match expr
		((symbol count_distinct) _agg_expr) '()
		((quote count_distinct) _agg_expr) '()
		((symbol aggregate) _agg_expr _agg_reduce _agg_neutral) '()
		((quote aggregate) _agg_expr _agg_reduce _agg_neutral) '()
		((symbol get_column) tblvar ignorecase col col_ignorecase)
		(if (or (nil? tblvar) (equal? (resolve_column_alias tblvar default_alias) default_alias))
			'()
			(list (list (quote get_column) tblvar ignorecase col col_ignorecase)))
		((quote get_column) tblvar ignorecase col col_ignorecase)
		(if (or (nil? tblvar) (equal? (resolve_column_alias tblvar default_alias) default_alias))
			'()
			(list (list (quote get_column) tblvar ignorecase col col_ignorecase)))
		(cons _head tail) (merge_unique (map tail (lambda (item) (external_column_refs_for_alias default_alias item))))
		_ '())))

(define field_passthrough_keys_for_alias (lambda (default_alias fields)
	(merge_unique (extract_assoc (coalesceNil fields '()) (lambda (_title expr)
		(if (or (expr_has_aggregates? expr)
			(empty_list? (external_column_refs_for_alias default_alias expr)))
			'()
			(list (canonical_column_expr_for_alias default_alias expr))))))))

(define group_key_col_name (lambda (i)
	(concat "k" i)))

(define aggregate_col_name (lambda (ag)
	(concat "agg_" (fnv_hash (serialize ag)))))

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
	(concat ".grp:" tbl ":" (fnv_hash (serialize (list "neumann-clean-groups-v2" schema tbl alias keys condition ags))))))

(define make_group_keytable_carrier (lambda (schema relation)
	(list (quote group-keytable) schema relation)))

(define group_carrier_kind (lambda (carrier)
	(if (list? carrier) (nth carrier 0) nil)))

(define group_carrier_schema (lambda (carrier)
	(if (list? carrier) (nth carrier 1) nil)))

(define group_carrier_relation (lambda (carrier)
	(if (list? carrier) (nth carrier 2) nil)))

(define group_stage_schema (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(qb_schema (car (union_branches input)))
			(if (query_block? input)
				(qb_schema input)
				(source_schema input))))))

(define group_stage_input_alias (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(qassoc_get (union_facts input) (quote alias) "__union")
			(if (query_block? input)
				(source_alias (car (qb_sources input)))
				(source_alias input))))))

(define group_stage_input_name (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(concat "union:" (fnv_hash (string input)))
			(if (query_block? input)
				(concat "query:" (fnv_hash (string input)))
				(source_relation input))))))

(define group_stage_default_carrier (lambda (stage)
	(begin
		(define schema (group_stage_schema stage))
		(define tbl (group_stage_input_name stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(make_group_keytable_carrier schema (group_table_name schema tbl alias keys condition ags)))))

(define group_stage_carrier (lambda (stage)
	(coalesceNil
		(qassoc_get (gs_facts stage) (quote carrier) nil)
		(group_stage_default_carrier stage))))

(define group_stage_carrier_relation (lambda (stage)
	(group_carrier_relation (group_stage_carrier stage))))

(define group_stage_carrier_schema (lambda (stage)
	(group_carrier_schema (group_stage_carrier stage))))

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

(define assoc_keys_values_as_dataset_rows (lambda (dict width)
	(map (extract_assoc dict (lambda (k v)
		(merge (list
			(if (list? k)
				k
				(if (<= width 1)
					(list k)
					(map (produceN width) (lambda (_) nil))))
			v))))
		(lambda (row) row))))

(define runtime_cons_list_expr (lambda (exprs)
	(if (empty_list? exprs)
		(quoted_runtime_list '())
		(list (quote cons) (car exprs) (runtime_cons_list_expr (cdr exprs))))))

(define make_group_stage_for_block (lambda (block src)
	(begin
		(define visible_ags (stage_aggregates_for_fields (qb_fields block)))
		(define having_ags (extract_aggregates (coalesceNil (qb_having block) true)))
		(define ags (dedupe_aggregates_by_col (if (empty_list? (qb_group block))
			(merge_unique (list visible_ags having_ags))
			(merge_unique (list visible_ags having_ags (list aggregate_count_descriptor))))))
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

(define make_group_stage_for_query_block (lambda (block)
	(begin
		(define visible_ags (stage_aggregates_for_fields (qb_fields block)))
		(define having_ags (extract_aggregates (coalesceNil (qb_having block) true)))
		(define ags (dedupe_aggregates_by_col (if (empty_list? (qb_group block))
			(merge_unique (list visible_ags having_ags))
			(merge_unique (list visible_ags having_ags (list aggregate_count_descriptor))))))
		(define alias (source_alias (car (qb_sources block))))
		(define group_keys (map (coalesceNil (qb_group block) '()) (lambda (expr) (canonical_column_expr_for_alias alias expr))))
		(define field_passthrough_keys (field_passthrough_keys_for_alias alias (qb_fields block)))
		(define passthrough_keys (external_column_refs_for_alias alias (coalesceNil (qb_having block) true)))
		(define input (make_query_block
			(qb_schema block)
			(qb_sources block)
			'()
			(qb_where block)
			'() nil '() nil nil
			'()
			(qb_stages block)
			(qb_facts block)))
		(make_group_stage
			(concat "group:query:" (fnv_hash (serialize (list (qb_sources block) (qb_where block) (qb_group block) ags))))
			input
			'()
			(merge_unique (list group_keys field_passthrough_keys passthrough_keys))
			ags
			(qb_having block)
			(qb_fields block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(list
				(list (quote condition) true))))))

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

(define aggregate_count_like? (lambda (ag)
	(match ag
		'(expr (symbol +) 0) true
		'(expr (quote +) 0) true
		_ false)))

(define group_aggregate_read_expr (lambda (grouptbl ag)
	(begin
		(define col (aggregate_col_name ag))
		(define read_expr (list (quote get_column) grouptbl false col false))
		(if (aggregate_count_like? ag)
			(list (quote coalesceNil) read_expr 0)
			read_expr))))

(define group_aggregate_order_read_expr (lambda (grouptbl ag)
	(list (quote get_column) grouptbl false (aggregate_col_name ag) false)))

(define count_distinct_read_expr (lambda (grouptbl agg_expr)
	(begin
		(define read_expr (list (quote get_column) grouptbl false (aggregate_col_name (count_distinct_descriptor agg_expr)) false))
		(list (quote if)
			(list (quote list?) read_expr)
			(list (quote count) read_expr)
			(list (quote coalesceNil) read_expr 0)))))

(define replace_group_expr (lambda (alias grouptbl keys key_names ags expr)
	(begin
		(define key_idx (group_key_index alias keys expr))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol count_distinct) agg_expr)
				(count_distinct_read_expr grouptbl agg_expr)
				((quote count_distinct) agg_expr)
				(count_distinct_read_expr grouptbl agg_expr)
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_read_expr grouptbl (list agg_expr agg_reduce agg_neutral))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_read_expr grouptbl (list agg_expr agg_reduce agg_neutral))
				((symbol get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				((quote get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				(cons head tail) (cons head (map tail (lambda (item) (replace_group_expr alias grouptbl keys key_names ags item))))
				_ expr)))))

(define replace_group_order_expr (lambda (alias grouptbl keys key_names ags expr)
	(begin
		(define key_idx (group_key_index alias keys expr))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol count_distinct) agg_expr)
				(count_distinct_read_expr grouptbl agg_expr)
				((quote count_distinct) agg_expr)
				(count_distinct_read_expr grouptbl agg_expr)
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_order_read_expr grouptbl (list agg_expr agg_reduce agg_neutral))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_order_read_expr grouptbl (list agg_expr agg_reduce agg_neutral))
				(cons head tail) (cons head (map tail (lambda (item) (replace_group_order_expr alias grouptbl keys key_names ags item))))
				_ expr)))))

(define direct_group_order_expr? (lambda (expr)
	(match expr
		((symbol get_column) _ _ _ _) true
		((quote get_column) _ _ _ _) true
		_ false)))

(define group_computed_order_col_name (lambda (expr)
	(concat "ord_" (fnv_hash (serialize expr)))))

(define group_order_physical_expr (lambda (grouptbl expr)
	(if (direct_group_order_expr? expr)
		expr
		(list (quote get_column) grouptbl false (group_computed_order_col_name expr) false))))

(define lower_group_computed_order_expr (lambda (expr)
	(match expr
		((symbol get_column) _tblvar _ col _) (symbol col)
		((quote get_column) _tblvar _ col _) (symbol col)
		(cons head tail) (cons head (map tail lower_group_computed_order_expr))
		_ expr)))

(define build_group_computed_order_column (lambda (schema grouptbl expr)
	(begin
		(define src (list grouptbl schema grouptbl false nil))
		(define cols (extract_columns_for_alias src expr))
		(list (quote createcolumn)
			(list (quote table) schema grouptbl)
			(group_computed_order_col_name expr)
			"any"
			(quoted_runtime_list '())
			(quoted_runtime_list '("temp" true))
			(cons (quote list) cols)
			(list (quote lambda)
				(map cols (lambda (col) (symbol col)))
				(lower_group_computed_order_expr expr))))))

(define group_key_equality_terms (lambda (alias key_names keys)
	(begin
		(define src (list alias nil nil false nil))
		(map (produceN (count keys)) (lambda (i)
			(list (quote equal?)
				(lower_column_expr_for_alias src (nth keys i))
				(list (quote outer) (symbol (nth key_names i)))))))))

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
			(define src (list alias schema tbl false nil))
			(define keycols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
			(define condition_cols (extract_columns_for_alias src condition))
			(define filtercols (merge_unique (list keycols condition_cols)))
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols (lambda (col) (symbol (concat alias "." col))))
					(list (quote optimize) (lower_column_expr_for_alias src condition)))
				(cons (quote list) keycols)
				(list (quote lambda)
					(map keycols (lambda (col) (symbol (concat alias "." col))))
					(cons (quote list) (map keys (lambda (expr) (lower_column_expr_for_alias src expr)))))
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

(define build_query_group_collect_plan (lambda (input grouptbl keys key_names)
	(begin
		(define schema (qb_schema input))
		(define key_fields (merge (map (produceN (count keys)) (lambda (i)
			(list (nth key_names i) (nth keys i))))))
		(define rows_plan (lower_query_block_as_dataset_rows input key_fields))
		(list (quote scan)
			'(session "__memcp_tx")
			rows_plan
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) key_names)
			(list (quote lambda)
				(map key_names (lambda (col) (symbol col)))
				(runtime_cons_list_expr (map key_names (lambda (col) (symbol col)))))
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
			false))))

(define build_group_ordered_scalar_column (lambda (schema tbl alias grouptbl keys key_names condition ag value_expr order_exprs dirs offset_value agg_reduce agg_neutral)
	(begin
		(define src (list alias schema tbl false nil))
		(define agg_col (aggregate_col_name ag))
		(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define condition_cols (extract_columns_for_alias src condition))
		(define order_cols (map order_exprs (lambda (order_expr) (order_column_for_alias src order_expr))))
		(define valuecols (extract_columns_for_alias src value_expr))
		(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols order_cols)))
		(define mapcols (merge_unique (list valuecols)))
		(list (quote createcolumn)
			(list (quote table) schema grouptbl)
			agg_col
			"any"
			(quoted_runtime_list '())
			(quoted_runtime_list '("temp" true))
			(cons (quote list) key_names)
			(list (quote lambda)
				(map key_names (lambda (col) (symbol col)))
				(list (quote scan_order)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(list (quote optimize)
							(cons (quote and) (cons
								(lower_column_expr_for_alias src condition)
								(group_key_equality_terms alias key_names keys)))))
					(quoted_runtime_list order_cols)
					(cons (quote list) dirs)
					0
					(coalesceNil offset_value 0)
					1
					(cons (quote list) mapcols)
					(list (quote lambda)
						(map mapcols (lambda (col) (symbol (concat alias "." col))))
						(lower_column_expr_for_alias src value_expr))
					agg_reduce
					agg_neutral
					false))))))

(define build_group_aggregate_column (lambda (schema tbl alias grouptbl keys key_names condition ag)
	(match ag
		'(((symbol scalar_order_value) value_expr order_exprs dirs offset_value) agg_reduce agg_neutral)
		(build_group_ordered_scalar_column schema tbl alias grouptbl keys key_names condition ag value_expr order_exprs dirs offset_value agg_reduce agg_neutral)
		'(((quote scalar_order_value) value_expr order_exprs dirs offset_value) agg_reduce agg_neutral)
		(build_group_ordered_scalar_column schema tbl alias grouptbl keys key_names condition ag value_expr order_exprs dirs offset_value agg_reduce agg_neutral)
		'(((symbol scalar_order_value) value_expr order_expr dir) agg_reduce agg_neutral)
		(build_group_ordered_scalar_column schema tbl alias grouptbl keys key_names condition ag value_expr (list order_expr) (list dir) 0 agg_reduce agg_neutral)
		'(((quote scalar_order_value) value_expr order_expr dir) agg_reduce agg_neutral)
		(build_group_ordered_scalar_column schema tbl alias grouptbl keys key_names condition ag value_expr (list order_expr) (list dir) 0 agg_reduce agg_neutral)
		'(agg_expr agg_reduce agg_neutral) (begin
			(define src (list alias schema tbl false nil))
			(define agg_col (aggregate_col_name ag))
			(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
			(define condition_cols (extract_columns_for_alias src condition))
			(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols)))
			(define aggcols (extract_columns_for_alias src agg_expr))
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
									(lower_column_expr_for_alias src condition)
									(group_key_equality_terms alias key_names keys)))))
						(cons (quote list) aggcols)
						(list (quote lambda)
							(map aggcols (lambda (col) (symbol (concat alias "." col))))
							(aggregate_map_value_expr ag (lower_column_expr_for_alias src agg_expr)))
						agg_reduce
						agg_neutral
						(aggregate_shard_combine ag)
						false)))))))

(define build_query_group_aggregate_column (lambda (input grouptbl keys key_names ag)
	(match ag '(agg_expr agg_reduce agg_neutral) (begin
		(define schema (qb_schema input))
		(define agg_col (aggregate_col_name ag))
		(define value_col "__agg")
		(define row_key_names (map key_names (lambda (col) (concat "__row_" col))))
		(define row_identity (if (count_distinct_descriptor? ag)
			(list "__row_identity" (cons (quote list) (merge (list keys (list agg_expr)))))
			'()))
		(define row_fields (merge (list
			row_identity
			(merge (map (produceN (count keys)) (lambda (i)
				(list (nth row_key_names i) (nth keys i)))))
			(list value_col agg_expr))))
		(define rows_plan (lower_query_block_as_dataset_rows input row_fields))
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
					rows_plan
					(quoted_runtime_list '())
					(list (quote lambda) '() true)
					(cons (quote list) (merge (list row_key_names (list value_col))))
					(list (quote lambda)
						(merge (list
							(map row_key_names (lambda (col) (symbol col)))
							(list (symbol value_col))))
						(list (quote if)
							(list (quote optimize)
								(cons (quote and) (map (produceN (count key_names)) (lambda (i)
									(list (quote equal?) (symbol (nth row_key_names i)) (list (quote outer) (symbol (nth key_names i))))))))
							(aggregate_map_value_expr ag (symbol value_col))
							agg_neutral))
					agg_reduce
					agg_neutral
					(query_aggregate_shard_combine ag)
					false)))))))

(define build_query_group_aggregate_insert_plan (lambda (input grouptbl keys key_names ag)
	(match ag '(agg_expr agg_reduce agg_neutral) (begin
		(define schema (qb_schema input))
		(define agg_col (aggregate_col_name ag))
		(define value_col "__agg")
		(define row_key_names (map key_names (lambda (col) (concat "__row_" col))))
		(define row_fields (merge (list
			(merge (map (produceN (count keys)) (lambda (i)
				(list (nth row_key_names i) (nth keys i)))))
			(list value_col agg_expr))))
		(define rows_plan (lower_query_block_as_dataset_rows input row_fields))
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define value_symbol (symbol value_col))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define mapped_value (aggregate_map_value_expr ag value_symbol))
		(define payload_expr (runtime_cons_list_expr (list mapped_value)))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(runtime_cons_list_expr (list (list agg_reduce (list (quote car) (quote old)) (list (quote car) (quote new)))))))
		(list (quote scan)
			'(session "__memcp_tx")
			rows_plan
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) (merge (list row_key_names (list value_col))))
			(list (quote lambda)
				(merge (list key_symbols (list value_symbol)))
				(runtime_cons_list_expr (list key_expr payload_expr)))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(quoted_runtime_list '())
			(list (quote lambda) (list (quote acc) (quote grouped))
				(group_insert_finish_expr schema grouptbl key_names (list agg_col)))
			false))
		_ (neumann_fail "build_queryplan" "query-input aggregate insert expects aggregate descriptor"))))

(define direct_group_assoc_expr (lambda (key_names ags)
	(cons (quote list)
		(merge (list
			(merge (map (produceN (count key_names)) (lambda (i)
				(list (nth key_names i) (list (quote nth) (quote row) i)))))
			(merge (map (produceN (count ags)) (lambda (i)
				(list (aggregate_col_name (nth ags i)) (list (quote nth) (quote row) (+ (count key_names) i)))))))))))

(define direct_group_aggregate_read_expr (lambda (ag)
	(begin
		(define read_expr (list (quote get_assoc) (quote rowassoc) (aggregate_col_name ag)))
		(if (aggregate_count_like? ag)
			(list (quote coalesceNil) read_expr 0)
			read_expr))))

(define replace_direct_group_expr (lambda (alias keys key_names ags expr)
	(begin
		(define key_idx (group_key_index alias keys expr))
		(if (not (nil? key_idx))
			(list (quote get_assoc) (quote rowassoc) (nth key_names key_idx))
			(match expr
				((symbol count_distinct) agg_expr)
				(count_distinct_read_expr (quote rowassoc) agg_expr)
				((quote count_distinct) agg_expr)
				(count_distinct_read_expr (quote rowassoc) agg_expr)
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(direct_group_aggregate_read_expr (list agg_expr agg_reduce agg_neutral))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(direct_group_aggregate_read_expr (list agg_expr agg_reduce agg_neutral))
				((symbol get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				((quote get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				(cons head tail) (cons head (map tail (lambda (item) (replace_direct_group_expr alias keys key_names ags item))))
				_ expr))))))

(define direct_group_result_assoc_expr (lambda (alias keys key_names ags fields)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
		(list (quote cons)
			(string title)
			(list (quote cons)
				(replace_direct_group_expr alias keys key_names ags expr)
				(direct_group_result_assoc_expr alias keys key_names ags rest)))
		_ (list (quote list)))))

(define build_base_group_scan_assoc_plan (lambda (schema tbl alias keys condition ags)
	(begin
		(define src (list alias schema tbl false nil))
		(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define condition_cols (extract_columns_for_alias src condition))
		(define agg_value_cols (merge_unique (map ags (lambda (ag)
			(match ag
				'(agg_expr _agg_reduce _agg_neutral)
				(if (equal? ag aggregate_count_descriptor)
					'()
					(extract_columns_for_alias src agg_expr))
				_ (neumann_fail "build_queryplan" "base group scan expects aggregate descriptor"))))))
		(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols)))
		(define mapcols (merge_unique (list group_key_cols_for_scan agg_value_cols)))
		(define key_expr (runtime_cons_list_expr (map keys (lambda (expr) (lower_column_expr_for_alias src expr)))))
		(define payload_expr (runtime_cons_list_expr (map ags (lambda (ag)
			(match ag
				'(agg_expr _agg_reduce _agg_neutral)
				(if (equal? ag aggregate_count_descriptor)
					1
					(aggregate_map_value_expr ag (lower_column_expr_for_alias src agg_expr)))
				_ (neumann_fail "build_queryplan" "base group scan expects aggregate descriptor"))))))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(aggregate_payload_merge_expr ags 0)))
		(define merge_groups (list (quote lambda) (list (quote acc) (quote grouped))
			(list (quote merge_assoc_mut) (quote acc) (quote grouped) merge_payload)))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) schema tbl)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat alias "." col))))
				(list (quote optimize) (lower_column_expr_for_alias src condition)))
			(cons (quote list) mapcols)
			(list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat alias "." col))))
				(runtime_cons_list_expr (list key_expr payload_expr)))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc_mut)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(quoted_runtime_list '())
			merge_groups
			false))))

(define lower_direct_base_group_stage (lambda (stage fields offset_value limit_value)
	(begin
		(define src (gs_input stage))
		(define schema (source_schema src))
		(define tbl (source_relation src))
		(define alias (source_alias src))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define key_names (group_key_cols keys))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define offset_expr (coalesceNil offset_value 0))
		(define limit_expr (coalesceNil limit_value -1))
		(list
			(list (quote lambda) (list (quote grouped))
				(list
					(list (quote lambda) (list (quote rows))
						(list (quote map)
							(list (quote slice)
								(quote rows)
								offset_expr
								(list (quote if)
									(list (quote equal?) limit_expr -1)
									(list (quote count) (quote rows))
									(list (quote min) (list (quote count) (quote rows)) (list (quote +) offset_expr limit_expr))))
							(list (quote lambda) (list (quote row))
								(list
									(list (quote lambda) (list (quote rowassoc))
										(list (quote resultrow)
											(direct_group_result_assoc_expr alias keys key_names ags fields)))
									(direct_group_assoc_expr key_names ags)))))
					(list (quote assoc_keys_values_as_dataset_rows) (quote grouped) (count key_names))))
			(build_base_group_scan_assoc_plan schema tbl alias keys condition ags)))))

(define aggregate_payload_merge_expr (lambda (ags idx)
	(if (>= idx (count ags))
		(quoted_runtime_list '())
		(match (nth ags idx)
			'(_agg_expr agg_reduce _agg_neutral)
			(list (quote cons)
				(list agg_reduce
					(list (quote nth) (quote old) idx)
					(list (quote nth) (quote new) idx))
				(aggregate_payload_merge_expr ags (+ idx 1)))
			_ (neumann_fail "build_queryplan" "aggregate payload merge expects aggregate descriptor")))))

(define group_upsert_collision_cols (lambda (value_cols)
	(cons (quote list)
		(cons "$update"
			(map value_cols (lambda (col) (concat "NEW." col)))))))

(define group_upsert_collision_lambda (lambda (value_cols)
	(begin
		(define new_params (map (produceN (count value_cols)) (lambda (i) (symbol (concat "__new_group_value_" i)))))
		(define update_pairs (merge (map (produceN (count value_cols)) (lambda (i)
			(list (nth value_cols i) (nth new_params i))))))
		(list (quote lambda)
			(cons (quote $update) new_params)
			(list (quote $update) (cons (quote list) update_pairs))))))

(define group_cleanup_missing_keys_plan (lambda (schema grouptbl key_names)
	(begin
		(define key_symbols (map key_names (lambda (col) (symbol col))))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) schema grouptbl)
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) (cons "$update" key_names))
			(list (quote lambda)
				(cons (quote $update) key_symbols)
				(list (quote if)
					(list (quote has_assoc?) (quote grouped) key_expr)
					true
					(list (quote $update))))
			nil
			nil
			nil
			false))))

(define group_insert_finish_expr (lambda (schema grouptbl key_names value_cols)
	(list (quote !begin)
		(group_cleanup_missing_keys_plan schema grouptbl key_names)
		(list (quote insert)
			(list (quote table) schema grouptbl)
			(cons (quote list) (merge (list key_names value_cols)))
			(list (quote assoc_keys_values_as_dataset_rows) (quote grouped) (count key_names))
			(group_upsert_collision_cols value_cols)
			(group_upsert_collision_lambda value_cols)
			true))))

(define build_query_group_aggregates_insert_plan (lambda (input grouptbl keys key_names ags)
	(begin
		(define schema (qb_schema input))
		(define row_key_names (map key_names (lambda (col) (concat "__row_" col))))
		(define value_cols (map (produceN (count ags)) (lambda (i) (concat "__agg" i))))
		(define row_fields (merge (list
			(merge (map (produceN (count keys)) (lambda (i)
				(list (nth row_key_names i) (nth keys i)))))
			(merge (map (produceN (count ags)) (lambda (i)
				(match (nth ags i)
					'(agg_expr _agg_reduce _agg_neutral)
					(list (nth value_cols i)
						(if (equal? (nth ags i) aggregate_count_descriptor)
							(car keys)
							agg_expr))
					_ (neumann_fail "build_queryplan" "query-input aggregate insert expects aggregate descriptor"))))))))
		(define rows_plan (lower_query_block_as_dataset_rows input row_fields))
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define value_symbols (map value_cols (lambda (col) (symbol col))))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define payload_expr (runtime_cons_list_expr (map (produceN (count ags)) (lambda (i)
			(if (equal? (nth ags i) aggregate_count_descriptor)
				1
				(aggregate_map_value_expr (nth ags i) (nth value_symbols i)))))))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(aggregate_payload_merge_expr ags 0)))
		(define finish_expr (group_insert_finish_expr schema grouptbl key_names (map ags aggregate_col_name)))
		(list (quote scan)
			'(session "__memcp_tx")
			rows_plan
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) (merge (list row_key_names value_cols)))
			(list (quote lambda)
				(merge (list key_symbols value_symbols))
				(runtime_cons_list_expr (list key_expr payload_expr)))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(quoted_runtime_list '())
			(list (quote lambda) (list (quote acc) (quote grouped))
				finish_expr)
			false))))

(define union_branch_group_row_fields (lambda (candidate_alias branch keys key_names ags value_cols)
	(begin
		(define projection (qb_fields branch))
		(define key_fields (map (produceN (count keys)) (lambda (i)
			(list (nth key_names i)
				(rewrite_derived_ref candidate_alias projection (nth keys i))))))
		(define value_fields (map (produceN (count ags)) (lambda (i)
			(match (nth ags i)
				'(agg_expr _agg_reduce _agg_neutral)
				(list (nth value_cols i)
					(if (equal? (nth ags i) aggregate_count_descriptor)
						(if (empty_list? keys) 1 (rewrite_derived_ref candidate_alias projection (car keys)))
						(rewrite_derived_ref candidate_alias projection agg_expr)))
				_ (neumann_fail "build_queryplan" "union-input aggregate insert expects aggregate descriptor")))))
		(merge (list (merge key_fields) (merge value_fields))))))

(define lower_union_block_as_dataset_rows (lambda (block keys key_names ags value_cols)
	(begin
		(define candidate_alias (qassoc_get (union_facts block) (quote alias) "__union"))
		(define branches (union_branches block))
		(if (empty_list? branches)
			(quoted_runtime_list '())
			(list (quote merge)
				(cons (quote list) (map branches (lambda (branch)
					(lower_query_block_as_dataset_rows branch
						(union_branch_group_row_fields candidate_alias branch keys key_names ags value_cols))))))))))

(define build_union_group_aggregates_insert_plan (lambda (input grouptbl keys key_names ags)
	(begin
		(define schema (qb_schema (car (union_branches input))))
		(define row_key_names key_names)
		(define value_cols (map (produceN (count ags)) (lambda (i) (concat "__agg" i))))
		(define rows_plan (lower_union_block_as_dataset_rows input keys row_key_names ags value_cols))
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define value_symbols (map value_cols (lambda (col) (symbol col))))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define payload_expr (runtime_cons_list_expr (map (produceN (count ags)) (lambda (i)
			(if (equal? (nth ags i) aggregate_count_descriptor)
				1
				(aggregate_map_value_expr (nth ags i) (nth value_symbols i)))))))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(aggregate_payload_merge_expr ags 0)))
		(define finish_expr (group_insert_finish_expr schema grouptbl key_names (map ags aggregate_col_name)))
		(list (quote scan)
			'(session "__memcp_tx")
			rows_plan
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) (merge (list row_key_names value_cols)))
			(list (quote lambda)
				(merge (list key_symbols value_symbols))
				(runtime_cons_list_expr (list key_expr payload_expr)))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(quoted_runtime_list '())
			(list (quote lambda) (list (quote acc) (quote grouped))
				finish_expr)
			false))))

(define build_scalar_single_query_stage_fill_plan (lambda (input grouptbl keys key_names value_ag count_ag)
	(match value_ag '(value_expr _value_reduce _value_neutral) (begin
		(define schema (qb_schema input))
		(define value_col (aggregate_col_name value_ag))
		(define count_col (aggregate_col_name count_ag))
		(define payload_col "__agg")
		(define row_key_names (map key_names (lambda (col) (concat "__row_" col))))
		(define row_fields (merge (list
			(merge (map (produceN (count keys)) (lambda (i)
				(list (nth row_key_names i) (nth keys i)))))
			(list payload_col value_expr))))
		(define rows_plan (lower_query_block_as_dataset_rows input row_fields))
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define payload_symbol (symbol payload_col))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define payload_expr (runtime_cons_list_expr (list payload_symbol 1)))
		(define merge_payload (list (quote lambda)
			(list (quote old) (quote new))
			(list (quote list)
				(list (quote car) (quote old))
				(list (quote +)
					(list (quote cadr) (quote old))
					(list (quote cadr) (quote new))))))
		(list (quote scan)
			'(session "__memcp_tx")
			rows_plan
			(quoted_runtime_list '())
			(list (quote lambda) '() true)
			(cons (quote list) (merge (list row_key_names (list payload_col))))
			(list (quote lambda)
				(merge (list key_symbols (list payload_symbol)))
				(runtime_cons_list_expr (list key_expr payload_expr)))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(quoted_runtime_list '())
			(list (quote lambda) (list (quote acc) (quote grouped))
				(group_insert_finish_expr schema grouptbl key_names (list value_col count_col)))
			false))
		_ (neumann_fail "build_queryplan" "scalar-single stage expects aggregate descriptor"))))

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
		(group_stage_schema stage)
		(list (list grouptbl (group_stage_schema stage) grouptbl false nil))
		(gs_output stage)
		having_expr
		nil nil
		(map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr dir) (list (replace_group_expr (group_stage_input_alias stage) grouptbl (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)) key_names ags expr) dir))))
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

(define stage_by_id (lambda (stages stage_id)
	(reduce (coalesceNil stages '()) (lambda (found stage)
		(if (not (nil? found))
			found
			(if (and (group_stage? stage) (equal? (gs_id stage) stage_id))
				stage
				nil)))
		nil)))

(define physicalize_stage_output_source (lambda (stages src)
	(begin
		(define relation (source_relation src))
		(if (not (stage_output_relation? relation))
			src
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id relation)))
				(if (nil? stage)
					(neumann_fail "build_queryplan" "stage-output source references unknown stage")
					true)
				(source_with_schema_relation
					src
					(group_stage_carrier_schema stage)
					(group_stage_carrier_relation stage)))))))

(define physicalize_stage_output_sources (lambda (stages sources)
	(map (coalesceNil sources '()) (lambda (src)
		(physicalize_stage_output_source stages src)))))

(define stage_outputs_from_sources_using (lambda (stages sources)
	(merge_unique (list (filter (map (coalesceNil sources '()) (lambda (src)
		(match src
			'(_alias _schema relation _outer _join_expr)
			(if (not (stage_output_relation? relation))
				nil
				(begin
					(define stage (stage_by_id stages (stage_output_relation_id relation)))
					(if (nil? stage)
						(neumann_fail "build_queryplan" "stage-output source references unknown stage")
						stage)))
			_ nil)))
		(lambda (stage) (not (nil? stage))))))))

(define scalar_first_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(scalar_first_probe_stage? stage)))))

(define scalar_first_stage_for_alias (lambda (stages sources default_alias tblvar tbl_ignorecase)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (and (scalar_first_stage_output_source? stages src)
				(source_alias_matches? src default_alias tblvar tbl_ignorecase))
				(stage_by_id stages (stage_output_relation_id (source_relation src)))
				nil)))
		nil)))

(define rewrite_scalar_first_probe_expr (lambda (stages sources default_alias expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define stage (scalar_first_stage_for_alias stages sources default_alias tblvar tbl_ignorecase))
			(if (nil? stage)
				expr
				(scalar_first_probe_expr stage col)))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(rewrite_scalar_first_probe_expr stages sources default_alias (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_scalar_first_probe_expr stages sources default_alias item))))
		_ expr)))

(define rewrite_scalar_first_probe_fields (lambda (stages sources default_alias fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(rewrite_scalar_first_probe_expr stages sources default_alias expr)))))

(define rewrite_scalar_first_probe_order (lambda (stages sources default_alias order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (list (rewrite_scalar_first_probe_expr stages sources default_alias expr) dir)
			_ item)))))

(define rewrite_scalar_first_probe_sources (lambda (stages sources default_alias)
	(map (coalesceNil sources '()) (lambda (src)
		(list
			(source_alias src)
			(source_schema src)
			(source_relation src)
			(source_outer? src)
			(rewrite_scalar_first_probe_expr stages sources default_alias (source_join_expr src)))))))

(define sources_without_scalar_first_outputs (lambda (stages sources)
	(filter (coalesceNil sources '()) (lambda (src)
		(not (scalar_first_stage_output_source? stages src))))))

(define stages_without_scalar_first_probes (lambda (stages)
	(filter (coalesceNil stages '()) (lambda (stage)
		(not (scalar_first_probe_stage? stage))))))

(define query_block_with_scalar_first_probes_using (lambda (stages block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define rewritten_sources (rewrite_scalar_first_probe_sources stages sources default_alias))
		(make_query_block
			(qb_schema block)
			(sources_without_scalar_first_outputs stages rewritten_sources)
			(rewrite_scalar_first_probe_fields stages sources default_alias (qb_fields block))
			(rewrite_scalar_first_probe_expr stages sources default_alias (qb_where block))
			(qb_group block)
			(rewrite_scalar_first_probe_expr stages sources default_alias (qb_having block))
			(rewrite_scalar_first_probe_order stages sources default_alias (qb_order block))
			(qb_limit block)
			(qb_offset block)
			(rewrite_scalar_first_probe_fields stages sources default_alias (qb_hidden block))
			(stages_without_scalar_first_probes (qb_stages block))
			(qb_facts block)))))

(define query_block_without_stages_after_prepare_using (lambda (stages block)
	(begin
		(define rewritten (query_block_with_scalar_first_probes_using stages block))
		(make_query_block
			(qb_schema rewritten)
			(physicalize_stage_output_sources stages (qb_sources rewritten))
			(qb_fields rewritten)
			(qb_where rewritten)
			(qb_group rewritten)
			(qb_having rewritten)
			(qb_order rewritten)
			(qb_limit rewritten)
			(qb_offset rewritten)
			(qb_hidden rewritten)
			'()
			(qb_facts rewritten)))))

(define query_block_without_stages_after_prepare (lambda (block)
	(query_block_without_stages_after_prepare_using (qb_stages block) block)))

(define query_block_without_stages_after_eager_prepare_using (lambda (stages block)
	(make_query_block
		(qb_schema block)
		(physicalize_stage_output_sources stages (qb_sources block))
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

(define query_block_with_prepared_sources_using (lambda (stages block)
	(begin
		(define rewritten (query_block_with_scalar_first_probes_using stages block))
		(make_query_block
			(qb_schema rewritten)
			(physicalize_stage_output_sources stages (qb_sources rewritten))
			(qb_fields rewritten)
			(qb_where rewritten)
			(qb_group rewritten)
			(qb_having rewritten)
			(qb_order rewritten)
			(qb_limit rewritten)
			(qb_offset rewritten)
			(qb_hidden rewritten)
			(qb_stages rewritten)
			(qb_facts rewritten)))))

(define query_block_with_prepared_sources (lambda (block)
	(query_block_with_prepared_sources_using (qb_stages block) block)))

(define group_stage_final_block (lambda (stage extra_sources)
	(begin
		(define src (gs_input stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(define carrier (group_stage_carrier stage))
		(define schema (group_carrier_schema carrier))
		(define grouptbl (group_carrier_relation carrier))
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
				(match item '(expr dir) (begin
					(define replaced_order_expr (replace_group_order_expr alias grouptbl keys key_names ags expr))
					(list (group_order_physical_expr grouptbl replaced_order_expr) dir)))))
			(gs_limit stage)
			(gs_offset stage)
			'() '() '()))))

(define group_stage_final_extra_sources_using (lambda (stages stage)
	(begin
		(define src (gs_input stage))
		(if (query_block? src)
			(physicalize_stage_output_sources stages
				(filter (cdr (qb_sources src)) (lambda (extra)
					(source_needed_after_group_stage? (group_stage_input_alias stage) stage extra))))
			'()))))

(define group_stage_final_extra_sources (lambda (stage)
	(group_stage_final_extra_sources_using (if (query_block? (gs_input stage)) (qb_stages (gs_input stage)) '()) stage)))

(define lower_group_stage_prepare_using (lambda (all_stages stage)
	(begin
		(define src (gs_input stage))
		(if (and (not (union_block? src)) (and (not (query_block? src)) (not (source_is_base_table? src))))
			(neumann_fail "build_queryplan" "group-stage lowering expects a base table, query-block, or union-block input")
			true)
		(define carrier (group_stage_carrier stage))
		(if (not (equal? (group_carrier_kind carrier) (quote group-keytable)))
			(neumann_fail "build_queryplan" "foreign-key backed group carriers are not lowered yet")
			true)
		(define schema (group_carrier_schema carrier))
		(define tbl (group_stage_input_name stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define query_input_carrier (or (query_block? src) (union_block? src)))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(define grouptbl (group_carrier_relation carrier))
		(define scalar_query_stage (and (query_block? src)
			(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single))
				(equal? (count ags) 2))))
		(define prepared_src (if (query_block? src) (query_block_without_stages_after_eager_prepare_using all_stages src) src))
		(define nested_stages (if (query_block? src)
			(merge_unique (list
				(qb_stages src)
				(stage_outputs_from_sources_using all_stages (qb_sources src))))
			'()))
		(define nested_prepare (if (query_block? src) (map nested_stages (lambda (nested_stage)
			(lower_stage_prepare_using all_stages nested_stage))) '()))
		(define nested_materialize (if (query_block? src) (lower_stage_materialize_all nested_stages) '()))
		(define nested_prepare_expr (if (empty_list? nested_prepare)
			nil
			(cons (quote !begin) (merge (list nested_prepare nested_materialize)))))
		(define key_columns (map key_names (lambda (col) (list (quote list) "column" col "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))
		(define agg_columns (if query_input_carrier
			(map ags (lambda (ag) (list (quote list) "column" (aggregate_col_name ag) "any" (quoted_runtime_list '()) (quoted_runtime_list '()))))
			'()))
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
		(define collect_plan (if query_input_carrier
			(if (union_block? src)
				(build_union_group_aggregates_insert_plan prepared_src grouptbl keys key_names (list aggregate_count_descriptor))
				(build_query_group_collect_plan prepared_src grouptbl keys key_names))
			(build_group_collect_plan schema tbl alias grouptbl keys key_names condition)))
		(define cleanup_plan (if (query_block? src)
			nil
			(build_group_keytable_cleanup schema tbl alias grouptbl keys key_names)))
		(define agg_plans (if query_input_carrier
			(if (empty_list? ags)
				'()
				(list (if (union_block? src)
					(build_union_group_aggregates_insert_plan prepared_src grouptbl keys key_names ags)
					(build_query_group_aggregates_insert_plan prepared_src grouptbl keys key_names ags))))
			(map ags (lambda (ag) (build_group_aggregate_column schema tbl alias grouptbl keys key_names condition ag)))))
		(define computed_order_exprs (merge_unique (map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr _dir) (begin
				(define replaced_order_expr (replace_group_order_expr alias grouptbl keys key_names ags expr))
				(if (direct_group_order_expr? replaced_order_expr) '() (list replaced_order_expr))))))))
		(define computed_order_plans (map computed_order_exprs (lambda (expr)
			(build_group_computed_order_column schema grouptbl expr))))
		(define aggregate_prepare_expr (cons
			(quote !begin)
			(merge (list agg_plans computed_order_plans))))
		(if scalar_query_stage
			(list (quote !begin)
				nested_prepare_expr
				keytable_init
				(build_scalar_single_query_stage_fill_plan prepared_src grouptbl keys key_names (car ags) (cadr ags)))
			(if query_input_carrier
				(cons (quote !begin)
					(merge (list
						nested_prepare
						nested_materialize
						(list keytable_init)
						(if (empty_list? ags) (list collect_plan) '())
						agg_plans
						computed_order_plans)))
				(list (quote !begin)
					nested_prepare_expr
					(if (nil? cleanup_plan)
						(list (quote !begin)
							keytable_init
							collect_plan)
						(list (quote if) keytable_init
							(list (quote !begin)
								collect_plan
								cleanup_plan)
							nil))
					aggregate_prepare_expr))))))

(define lower_orc_stage_prepare (lambda (stage)
	(begin
		(define src (os_source stage))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "ORC stage lowering only supports base tables")
			true)
		(list (quote createcolumn)
			(source_table_expr src)
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

(define lower_window_stage_prepare (lambda (stage)
	(match stage
		'(_ id source column sortcols sortdirs partitioncount mapcols mapfn reducefn reduceinit _facts)
		(lower_orc_stage_prepare (make_orc_stage
			id source column
			sortcols sortdirs partitioncount
			mapcols mapfn reducefn reduceinit))
		_ (neumann_fail "build_queryplan" "malformed window-stage"))))

(define lower_orc_stage_materialize (lambda (stage)
	(match stage
		'(_ _id src col _sortcols _sortdirs _partitioncount _mapcols _mapfn _reducefn _reduceinit _facts)
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "ORC materialization expects base table source")
			(list (quote scan)
				'(session "__memcp_tx")
				(source_table_expr src)
				(quoted_runtime_list '())
				(list (quote lambda) '() true)
				(quoted_runtime_list (list col))
				(list (quote lambda) (list (quote __orc_value))
					(list (quote if) (list (quote nil?) (quote __orc_value)) 0 1))
				(quote +)
				0
				nil
				false))
		_ nil)))

(define lower_stage_materialize (lambda (stage)
	(if (or (orc_stage? stage) (window_stage? stage))
		(lower_orc_stage_materialize stage)
		nil)))

(define lower_stage_materialize_all (lambda (stages)
	(filter (map (coalesceNil stages '()) lower_stage_materialize)
		(lambda (plan) (not (nil? plan))))))

(define lower_stage_prepare_using (lambda (all_stages stage)
	(if (group_stage? stage)
		(lower_group_stage_prepare_using all_stages stage)
		(if (orc_stage? stage)
			(lower_orc_stage_prepare stage)
			(if (window_stage? stage)
				(lower_window_stage_prepare stage)
				(neumann_fail "build_queryplan" "unknown logical stage"))))))

(define lower_stage_prepare (lambda (stage)
	(lower_stage_prepare_using (list stage) stage)))

(define query_block_has_only_window_stages? (lambda (block)
	(and
		(reduce (qb_sources block) (lambda (ok src) (and ok (source_is_base_table? src))) true)
		(and (not (empty_list? (qb_stages block)))
			(reduce (qb_stages block) (lambda (ok stage) (and ok (window_stage? stage))) true)))))

(define lower_group_stage (lambda (stage)
	(begin
		(define src (gs_input stage))
		(if (and (not (query_block? src)) (not (source_is_base_table? src)))
			(neumann_fail "build_queryplan" "group-stage lowering expects a base table or query-block input")
			true)
		(define tbl (group_stage_input_name stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define ags (gs_aggregates stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_names (group_key_cols keys))
		(list (quote begin)
			(lower_group_stage_prepare_using (list stage) stage)
			(lower_query_block_core (group_stage_final_block stage (group_stage_final_extra_sources stage)))))))

(define row_number_get_column? (lambda (col expr)
	(match expr
		'(fn _tblvar _ignore column _json_path)
		(and (equal? fn (quote get_column)) (equal? column col))
		_ false)))

(define row_number_stage_filter_from_condition (lambda (col condition)
	(match condition
		'(op expr limit) (if (row_number_get_column? col expr)
			(if (equal? op (quote <=))
				(list (quote le) limit)
				(if (or (equal? op (quote equal??)) (equal? op (quote equal?)))
					(list (quote eq) limit)
					nil))
			nil)
		_ nil)))

(define window_scan_dirs (lambda (sortdirs)
	(map (coalesceNil sortdirs '()) (lambda (desc) (if desc (quote >) (quote <))))))

(define count_star_field_title (lambda (fields)
	(match (coalesceNil fields '())
		'(title expr) (match expr
			'(fn seed reduce init) (if (and (equal? fn (quote aggregate))
				(and (equal? seed 1)
					(and (equal? reduce (quote +)) (equal? init 0))))
				title
				nil)
			_ nil)
		_ nil)))

(define row_number_count_match_expr (lambda (mode rownum limit)
	(if (equal? mode (quote eq))
		(list (quote equal?) rownum limit)
		(list (quote <=) rownum limit))))

(define row_number_count_mapfn (lambda (mapcols)
	(list (quote lambda)
		(map (coalesceNil mapcols '()) (lambda (col) (symbol col)))
		(if (empty_list? mapcols)
			true
			(if (single_source? mapcols)
				(symbol (car mapcols))
				(cons (quote list) (map mapcols (lambda (col) (symbol col)))))))))

(define row_number_count_reducefn (lambda (mode limit)
	(list (quote lambda) (list (quote state) (quote partition_key))
		(list (quote begin)
			(list (quote define) (quote cnt) (list (quote car) (quote state)))
			(list (quote define) (quote prev_partition) (list (quote cadr) (quote state)))
			(list (quote define) (quote prev_rownum) (list (quote car) (list (quote cdr) (list (quote cdr) (quote state)))))
			(list (quote define) (quote same_partition) (list (quote and)
				(list (quote not) (list (quote equal?) (quote prev_rownum) 0))
				(list (quote equal?) (quote partition_key) (quote prev_partition))))
			(list (quote define) (quote next_rownum) (list (quote if) (quote same_partition) (list (quote +) (quote prev_rownum) 1) 1))
			(list (quote define) (quote include_row) (row_number_count_match_expr mode (quote next_rownum) limit))
			(list (quote list)
				(list (quote +) (quote cnt) (list (quote if) (quote include_row) 1 0))
				(quote partition_key)
				(quote next_rownum))))))

(define row_number_stage? (lambda (stage)
	(match stage
		'(_ id _source _column _sortcols _sortdirs _partitioncount _mapcols _mapfn _reducefn _reduceinit _facts)
		(and (window_stage? stage)
			(and (string? id)
				(equal? (substr id 0 18) "window-row-number:")))
		_ false)))

(define replace_row_number_expr (lambda (src col expr)
	(match expr
		((symbol if) presence nil_expr inner) (list (quote if) presence nil_expr (replace_row_number_expr src col inner))
		((quote if) presence nil_expr inner) (list (quote if) presence nil_expr (replace_row_number_expr src col inner))
		((symbol get_column) tblvar tbl_ignorecase column col_ignorecase) (if (and (equal? column col) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase))
			(quote __row_number)
			expr)
		((quote get_column) tblvar tbl_ignorecase column col_ignorecase) (if (and (equal? column col) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase))
			(quote __row_number)
			expr)
		(cons head tail) (cons head (map tail (lambda (item) (replace_row_number_expr src col item))))
		_ expr)))

(define lower_row_number_result_fields (lambda (src col fields)
	(map_assoc fields (lambda (_title expr)
		(lower_column_expr_for_alias src (replace_row_number_expr src col expr))))))

(define lower_fused_row_number_sorted_block (lambda (block stage src fields order_items condition row_number_filter)
	(match stage
		'(_ _id _stage_src col sortcols sortdirs _partitioncount mapcols _mapfn _reducefn _reduceinit _facts)
		(begin
			(define pre_condition (nth row_number_filter 2))
			(define filtercols (extract_columns_for_alias src pre_condition))
			(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
				(extract_columns_for_alias src expr)))))
			(define ordercols (merge_unique (map (order_exprs order_items) (lambda (expr)
				(extract_columns_for_alias src expr)))))
			(define scan_mapcols (merge_unique (list (without_col (merge_unique (list fieldcols ordercols)) col) mapcols)))
			(define row_number_output_titles (merge (extract_assoc fields (lambda (title expr)
				(if (row_number_condition_expr? src col expr) (list (string title)) '())))))
			(define row_number_order_keys (merge (map (produceN (count (coalesceNil order_items '()))) (lambda (i)
				(match (nth order_items i)
					'(expr _dir) (if (row_number_condition_expr? src col expr) (list (concat "__order_" i)) '())
					_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))
			(define output_pairs (merge (extract_assoc fields (lambda (title expr)
				(list (string title) (if (row_number_condition_expr? src col expr)
					nil
					(lower_column_expr_for_alias src expr)))))))
			(define sort_pairs (merge (map (produceN (count (coalesceNil order_items '()))) (lambda (i)
				(match (nth order_items i)
					'(expr _dir) (list (concat "__order_" i) (if (row_number_condition_expr? src col expr)
						nil
						(lower_column_expr_for_alias src expr)))
					_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))
			(define row_assoc (cons (quote list) (merge (list output_pairs sort_pairs))))
			(define filter_expr (list (quote lambda)
				(map filtercols (lambda (filter_col) (symbol (concat (source_alias src) "." filter_col))))
				(list (quote optimize) (lower_column_expr_for_alias src pre_condition))))
			(define map_expr (list (quote lambda)
				(map scan_mapcols (lambda (map_col) (symbol (concat (source_alias src) "." map_col))))
				row_assoc))
			(define reduce_expr (list (quote lambda) (list (quote state) (quote row))
				(list (quote begin)
					(list (quote define) (quote rows) (list (quote car) (quote state)))
					(list (quote define) (quote rownum) (list (quote cadr) (quote state)))
					(list (quote define) (quote next_rownum) (list (quote +) (quote rownum) 1))
					(list (quote define) (quote row_with_rownum)
						(list (quote reduce)
							(quoted_runtime_list row_number_output_titles)
							(list (quote lambda) (list (quote acc) (quote title))
								(list (quote set_assoc) (quote acc) (quote title) (quote next_rownum)))
							(quote row)))
					(list (quote define) (quote sorted_row)
						(list (quote reduce)
							(quoted_runtime_list row_number_order_keys)
							(list (quote lambda) (list (quote acc) (quote key))
								(list (quote set_assoc) (quote acc) (quote key) (quote next_rownum)))
							(quote row_with_rownum)))
					(list (quote list)
						(list (quote if)
							(row_number_count_match_expr (nth row_number_filter 0) (quote next_rownum) (nth row_number_filter 1))
							(list (quote append) (quote rows) (quote sorted_row))
							(quote rows))
						(quote next_rownum)))))
			(define rows_plan (list
				(list (quote lambda) (list (quote state)) (list (quote car) (quote state)))
				(list (quote scan_order)
					'(session "__memcp_tx")
					(source_table_expr src)
					(cons (quote list) filtercols)
					filter_expr
					(quoted_runtime_list sortcols)
					(cons (quote list) (window_scan_dirs sortdirs))
					0
					0
					-1
					(cons (quote list) scan_mapcols)
					map_expr
					reduce_expr
					(list (quote list) (list (quote list)) 0)
					(source_outer? src))))
			(join_sorted_rows_plan rows_plan fields order_items (qb_offset block) (qb_limit block)))
		_ nil)))

(define row_number_order_compatible? (lambda (src order_items sortcols sortdirs)
	(if (empty_list? order_items)
		true
		(and
			(equal? (order_cols_for_alias src order_items) sortcols)
			(equal? (serialize (order_dirs order_items)) (serialize (window_scan_dirs sortdirs)))))))

(define lower_fused_row_number_block (lambda (block)
	(match (qb_stages block)
		(cons stage rest) (if (or (not (empty_list? rest)) (not (row_number_stage? stage)))
			nil
			(match (qb_sources block)
				(cons src src_rest) (if (or (not (empty_list? src_rest)) (not (source_is_base_table? src)))
					nil
					(match stage
						'(_ _id stage_src col sortcols sortdirs _partitioncount mapcols _mapfn _reducefn _reduceinit _facts)
						(if (or (not (equal? (source_schema stage_src) (source_schema src)))
							(not (equal? (source_relation stage_src) (source_relation src))))
							nil
							(begin
								(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
								(define condition (combine_where (qb_where block) (source_join_expr src)))
								(define row_number_filter (extract_row_number_filter src col condition))
								(if (not (row_number_order_compatible? src (coalesceNil (qb_order block) '()) sortcols sortdirs))
									(if (nil? row_number_filter)
										nil
										(lower_fused_row_number_sorted_block block stage src fields (coalesceNil (qb_order block) '()) condition row_number_filter))
									(begin
										(define pre_condition (if (nil? row_number_filter) condition (nth row_number_filter 2)))
										(define rewritten_condition (replace_row_number_expr src col condition))
										(define filtercols (extract_columns_for_alias src pre_condition))
										(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
											(extract_columns_for_alias src expr)))))
										(define scan_mapcols (merge_unique (list (without_col fieldcols col) mapcols)))
										(define filter_expr (list (quote lambda)
											(map filtercols (lambda (filter_col) (symbol (concat (source_alias src) "." filter_col))))
											(list (quote optimize) (lower_column_expr_for_alias src pre_condition))))
										(define row_expr (list (quote resultrow)
											(cons (quote list) (lower_row_number_result_fields src col fields))))
										(define continuation_expr (list (quote lambda) (list (quote __row_number))
											(list (quote if)
												(list (quote optimize) (lower_column_expr_for_alias src rewritten_condition))
												row_expr
												nil)))
										(define map_expr (list (quote lambda)
											(map scan_mapcols (lambda (map_col) (symbol (concat (source_alias src) "." map_col))))
											(list (quote list)
												(row_number_scan_partition_expr (source_alias src) mapcols)
												continuation_expr)))
										(define reduce_expr (list (quote lambda) (list (quote state) (quote mapped))
											(list (quote begin)
												(list (quote define) (quote prev_partition) (list (quote car) (quote state)))
												(list (quote define) (quote prev_rownum) (list (quote cadr) (quote state)))
												(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
												(list (quote define) (quote continuation) (list (quote cadr) (quote mapped)))
												(list (quote define) (quote same_partition) (list (quote and)
													(list (quote not) (list (quote equal?) (quote prev_rownum) 0))
													(list (quote equal?) (quote row_partition) (quote prev_partition))))
												(list (quote define) (quote next_rownum) (list (quote if) (quote same_partition) (list (quote +) (quote prev_rownum) 1) 1))
												(list (quote continuation) (quote next_rownum))
												(list (quote list) (quote row_partition) (quote next_rownum)))))
										(list (quote scan_order)
											'(session "__memcp_tx")
											(source_table_expr src)
											(cons (quote list) filtercols)
											filter_expr
											(quoted_runtime_list sortcols)
											(cons (quote list) (window_scan_dirs sortdirs))
											0
											(coalesceNil (qb_offset block) 0)
											(coalesceNil (qb_limit block) -1)
											(cons (quote list) scan_mapcols)
											map_expr
											reduce_expr
											(list (quote list) nil 0)
											(source_outer? src))))))
						_ nil))
				_ nil))
		_ nil)))

(define lower_row_number_count_scan (lambda (title schema relation sortcols sortdirs mapcols filter)
	(match filter '(mode limit)
		(list (quote resultrow)
			(list (quote list) title
				(list (quote car)
					(list (quote scan_order)
						'(session "__memcp_tx")
						(list (quote table) schema relation)
						(quoted_runtime_list '())
						(list (quote lambda) '() true)
						(quoted_runtime_list sortcols)
						(cons (quote list) (window_scan_dirs sortdirs))
						0
						0
						-1
						(quoted_runtime_list mapcols)
						(row_number_count_mapfn mapcols)
						(row_number_count_reducefn mode limit)
						(list (quote list) 0 nil 0)
						false))))
		_ nil)))

(define lower_row_number_top_count_block (lambda (block)
	(match (qb_stages block)
		'(stage) (if (not (window_stage? stage))
			nil
			(match stage
				'(_ _id src col sortcols sortdirs _partitioncount mapcols _mapfn _reducefn _reduceinit _facts)
				(match src '(alias schema relation _outer _join)
					(begin
						(define title (count_star_field_title (qb_fields block)))
						(define filter (row_number_stage_filter_from_condition col (qb_where block)))
						(if (or (nil? title) (or (nil? filter) (not (number? (cadr filter)))))
							nil
							(lower_row_number_count_scan title schema relation sortcols sortdirs mapcols filter)))
					_ nil)
				_ nil))
		_ nil)))

(define lower_grouped_query_block_with_stages (lambda (block)
	(begin
		(define fused_top_count (lower_row_number_top_count_block block))
		(if (not (nil? fused_top_count))
			fused_top_count
			(begin
				(define sources (qb_sources block))
				(define base_src (car sources))
				(if (not (source_is_base_table? base_src))
					(neumann_fail "build_queryplan" "group-stage with subquery stages requires a base driver source")
					true)
				(define stage_sources (physicalize_stage_output_sources (qb_stages block) (cdr sources)))
				(define final_stage_sources (physicalize_stage_output_sources (qb_stages block)
					(filter (cdr sources) (lambda (src)
						(source_needed_after_group? (source_alias base_src) block src)))))
				(define grouped_input_block (make_query_block
					(qb_schema block)
					(cons base_src stage_sources)
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
				(define main_stage (make_group_stage_for_query_block grouped_input_block))
				(cons (quote !begin)
					(merge (list
						(map (qb_stages block) (lambda (stage)
							(lower_stage_prepare_using (qb_stages block) stage)))
						(lower_stage_materialize_all (qb_stages block))
						(list (lower_group_stage_prepare_using (cons main_stage (qb_stages block)) main_stage))
						(list (lower_query_block_core (group_stage_final_block main_stage final_stage_sources)))))))))))

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

(define row_number_stage_consumed_by_source? (lambda (stage src)
	(match stage
		'(_ _id stage_src col _sortcols _sortdirs _partitioncount _mapcols _mapfn _reducefn _reduceinit _facts)
		(and (row_number_stage? stage)
			(and (equal? (source_schema stage_src) (source_schema src))
				(and (equal? (source_relation stage_src) (source_relation src))
					(not (nil? (extract_row_number_filter src col (source_join_expr src)))))))
		_ false)))

(define row_number_stage_consumed_by_join? (lambda (stage sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(or found (row_number_stage_consumed_by_source? stage src)))
		false)))

(define query_block_stages_to_prepare (lambda (block)
	(filter
		(if (single_source? (qb_sources block))
			(qb_stages block)
			(filter (qb_stages block) (lambda (stage)
				(not (row_number_stage_consumed_by_join? stage (qb_sources block))))))
		(lambda (stage)
			(not (scalar_first_probe_stage? stage))))))

(define lower_query_block_with_stages (lambda (block)
	(if (empty_list? (qb_stages block))
		(lower_query_block_core block)
		(begin
			(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
				(lower_grouped_query_block_with_stages block)
				(begin
					(define fused_row_number (lower_fused_row_number_block block))
					(if (not (nil? fused_row_number))
						fused_row_number
						(cons (quote !begin)
							(merge (list
								(map (query_block_stages_to_prepare block) (lambda (stage)
									(lower_stage_prepare_using (qb_stages block) stage)))
								(list (lower_query_block_core (if (single_source? (qb_sources block))
									(query_block_without_stages_after_prepare_using (qb_stages block) block)
									(query_block_with_prepared_sources_using (qb_stages block) block))))))))))))))

(define source_is_base_table? (lambda (src)
	(string? (source_relation src))))

(define information_schema_source? (lambda (schema relation)
	(and (string? schema)
		(and (string? relation)
			(equal?? schema "information_schema")))))

(define source_table_expr (lambda (src)
	(if (information_schema_source? (source_schema src) (source_relation src))
		(list (quote information_schema_rows) (source_schema src) (source_relation src))
		(list (quote table) (source_schema src) (source_relation src)))))

(define query_block_has_aggregates? (lambda (block)
	(not (empty_list? (stage_aggregates_for_fields (qb_fields block))))))

(define direct_base_group_stage_supported? (lambda (block stage)
	(and (not (empty_list? (qb_group block)))
		(and (nil? (qb_having block))
			(and (empty_list? (qb_order block))
				(and (query_block_has_aggregates? block)
					(not (reduce (gs_aggregates stage)
						(lambda (found ag) (or found (count_distinct_descriptor? ag)))
						false))))))))

(define table_column_names (lambda (schema tbl)
	(map (get_schema schema tbl) (lambda (col) (col "Field")))))

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

(define query_limit_active? (lambda (offset_value limit_value)
	(or (and (not (nil? offset_value)) (not (equal? offset_value 0)))
		(and (not (nil? limit_value)) (not (equal? limit_value -1))))))

(define lower_single_source_query_block (lambda (block)
	(begin
		(define src (car (qb_sources block)))
		(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "single-source query-block lowering only supports base tables")
			true)
		(if (not (empty_list? (qb_stages block)))
			(neumann_fail "build_queryplan" "pre-existing stages are not implemented yet")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(begin
				(define group_stage (make_group_stage_for_block block src))
				(if (direct_base_group_stage_supported? block group_stage)
					(lower_direct_base_group_stage group_stage fields (qb_offset block) (qb_limit block))
					(lower_group_stage group_stage)))
				(begin
					(define alias (source_alias src))
					(define condition (combine_where (qb_where block) (source_join_expr src)))
					(define membership (driver_membership_for_source src condition))
					(define membership_table_expr (if (nil? membership) nil (recset_project_join_expr_for_membership src membership)))
					(define effective_condition (strip_driver_membership_for_source src condition membership))
					(define order_items (coalesceNil (qb_order block) '()))
					(define scan_order_supported (order_items_belong_to_source? src order_items))
					(define bounded (query_limit_active? (qb_offset block) (qb_limit block)))
					(define filtercols (extract_columns_for_alias src effective_condition))
					(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
						(extract_columns_for_alias src expr)))))
					(define ordercols (if (empty_list? order_items) '() (scan_order_sort_columns_for_alias src order_items)))
					(define mapcols (merge_unique (list filtercols fieldcols)))
					(define table_expr (coalesceNil membership_table_expr (source_table_expr src)))
					(define filter_expr (list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(list (quote optimize) (lower_column_expr_for_alias src effective_condition))))
				(define map_expr (list (quote lambda)
					(map mapcols (lambda (col) (symbol (concat alias "." col))))
					(list (quote resultrow)
						(cons (quote list) (map_assoc fields (lambda (title expr)
							(lower_column_expr_for_alias src expr)))))))
				(if (and (empty_list? order_items) (not bounded))
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
						(source_outer? src))
					(if scan_order_supported
						(list (quote scan_order)
							'(session "__memcp_tx")
							table_expr
							(cons (quote list) filtercols)
							filter_expr
							(cons (quote list) ordercols)
							(cons (quote list) (if (empty_list? order_items) '() (order_dirs order_items)))
							0
							(coalesceNil (qb_offset block) 0)
							(coalesceNil (qb_limit block) -1)
							(cons (quote list) mapcols)
							map_expr
							nil
							nil
							(source_outer? src))
						(join_sorted_rows_plan
							(build_join_scan_rows_with_mapper
								(qb_schema block)
								(qb_sources block)
								(qb_sources block)
								alias
								(merge (list
										(extract_assoc fields (lambda (_title expr) expr))
										(list effective_condition)
										(order_exprs order_items)))
									effective_condition
								(lower_join_result_row_assoc (qb_sources block) alias fields order_items)
								'()
								0
								-1
								'())
							fields
							order_items
							(qb_offset block)
							(qb_limit block)))))))))

(define order_exprs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) expr
			_ true)))))

(define source_join_exprs (lambda (sources)
	(map (coalesceNil sources '()) (lambda (src) (coalesceNil (source_join_expr src) true)))))

(define join_cols_for_alias (lambda (all_sources default_alias alias needed_exprs)
	(merge_unique (map (coalesceNil needed_exprs '()) (lambda (expr)
		(extract_columns_for_join_alias all_sources default_alias alias expr))))))

(define join_filter_cols_for_alias (lambda (all_sources default_alias alias condition)
	(extract_columns_for_join_alias all_sources default_alias alias condition)))

(define row_number_condition_expr? (lambda (src col expr)
	(match expr
		((symbol if) _presence nil_expr inner) (if (nil? nil_expr) (row_number_condition_expr? src col inner) false)
		((quote if) _presence nil_expr inner) (if (nil? nil_expr) (row_number_condition_expr? src col inner) false)
		((symbol get_column) tblvar tbl_ignorecase column _col_ignorecase) (and (equal? column col) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase))
		((quote get_column) tblvar tbl_ignorecase column _col_ignorecase) (and (equal? column col) (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase))
		_ false)))

(define extract_row_number_filter (lambda (src col condition)
	(match (coalesceNil condition true)
		((symbol and) left right) (begin
			(define left_match (extract_row_number_filter src col left))
			(if (nil? left_match)
				(begin
					(define right_match (extract_row_number_filter src col right))
					(if (nil? right_match)
						nil
						(list (nth right_match 0) (nth right_match 1) (combine_where left (nth right_match 2)))))
				(list (nth left_match 0) (nth left_match 1) (combine_where (nth left_match 2) right))))
		((quote and) left right)
		(extract_row_number_filter src col (list (quote and) left right))
		'(op expr limit) (if (and (number? limit) (row_number_condition_expr? src col expr))
			(if (equal? op (quote <=))
				(list (quote le) limit true)
				(if (or (equal? op (quote equal??)) (equal? op (quote equal?)))
					(list (quote eq) limit true)
					nil))
			nil)
		_ nil)))

(define row_number_stage_for_source (lambda (stages src condition)
	(reduce (coalesceNil stages '()) (lambda (found stage)
		(if (not (nil? found))
			found
			(match stage
				'(_ id stage_src col sortcols sortdirs partitioncount mapcols _mapfn _reducefn _reduceinit facts)
				(if (and (row_number_stage? stage)
					(and (equal? (qassoc_get facts (quote kind) nil) (quote ordered-window))
						(and (equal? (source_schema stage_src) (source_schema src))
							(equal? (source_relation stage_src) (source_relation src)))))
					(begin
						(define filter (extract_row_number_filter src col condition))
						(if (nil? filter)
							nil
							(list stage filter)))
					nil)
				_ nil)))
		nil)))

(define lower_join_result_fields (lambda (all_sources default_alias fields)
	(map_assoc fields (lambda (_title expr)
		(lower_column_expr_for_join all_sources default_alias expr)))))

(define lower_join_result_row_assoc (lambda (all_sources default_alias fields order_items)
	(begin
		(define output_pairs (merge (extract_assoc fields (lambda (title expr)
			(list (string title) (lower_column_expr_for_join all_sources default_alias expr))))))
		(define sort_pairs (merge (map (produceN (count (coalesceNil order_items '()))) (lambda (i)
			(match (nth order_items i)
				'(expr _dir) (list (concat "__order_" i) (lower_column_expr_for_join all_sources default_alias expr))
				_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))
		(cons (quote list) (merge (list output_pairs sort_pairs))))))

(define lower_dataset_row_assoc (lambda (all_sources default_alias fields)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
		(list (quote cons)
			(string title)
			(list (quote cons)
				(lower_column_expr_for_join all_sources default_alias expr)
				(lower_dataset_row_assoc all_sources default_alias rest)))
		_ (list (quote list)))))

(define safe_get_assoc_expr (lambda (row key)
	(list (quote if)
		(list (quote or)
			(list (quote nil?) row)
			(list (quote list?) row))
		(list (quote get_assoc) row key)
		nil)))

(define sorted_row_key_expr (lambda (row key)
	(list (quote coalesceNil)
		(safe_get_assoc_expr row key)
		(list (quote if)
			(list (quote and)
				(list (quote list?) row)
				(list (quote >) (list (quote count) row) 0))
			(safe_get_assoc_expr (list (quote car) row) key)
			nil))))

(define join_order_compare_expr (lambda (order_items idx)
	(if (>= idx (count (coalesceNil order_items '())))
		false
		(match (nth order_items idx)
			'(_expr dir) (begin
				(define key (concat "__order_" idx))
				(define left (sorted_row_key_expr (symbol "a") key))
				(define right (sorted_row_key_expr (symbol "b") key))
				(list (quote if)
					(list (quote equal??) left right)
					(join_order_compare_expr order_items (+ idx 1))
					(list (quote apply) dir (list (quote list) left right))))
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item")))))

(define sorted_row_result_assoc_expr (lambda (fields)
	(match (coalesceNil fields '())
		(cons title (cons _expr rest))
		(list (quote cons)
			(string title)
			(list (quote cons)
				(list (quote coalesceNil)
					(safe_get_assoc_expr (symbol "row") (string title))
					(list (quote coalesceNil)
						(safe_get_assoc_expr (symbol "row") title)
						(list (quote if)
							(list (quote and)
								(list (quote list?) (symbol "row"))
								(list (quote >) (list (quote count) (symbol "row")) 0))
							(safe_get_assoc_expr (list (quote car) (symbol "row")) (string title))
							nil)))
				(sorted_row_result_assoc_expr rest)))
		_ (list (quote list)))))

(define join_sorted_rows_plan (lambda (rows_plan fields order_items offset_value limit_value)
	(begin
		(define offset_expr (coalesceNil offset_value 0))
		(define limit_expr (coalesceNil limit_value -1))
		(define end_expr (list (quote if)
			(list (quote equal?) limit_expr -1)
			(list (quote count) (quote sorted))
			(list (quote min) (list (quote count) (quote sorted)) (list (quote +) offset_expr limit_expr))))
		(list
			(list (quote lambda) (list (quote sorted))
				(list (quote map)
					(list (quote slice) (quote sorted) offset_expr end_expr)
					(list (quote lambda) (list (quote row))
						(list (quote resultrow)
							(sorted_row_result_assoc_expr fields)))))
			(list (quote sort) rows_plan
				(list (quote lambda) (list (quote a) (quote b))
					(join_order_compare_expr order_items 0)))))))

(define without_col (lambda (cols col)
	(filter (coalesceNil cols '()) (lambda (item) (not (equal? item col))))))

(define row_number_partition_expr (lambda (mapcols)
	(if (empty_list? mapcols)
		true
		(if (single_source? mapcols)
			(symbol (car mapcols))
			(cons (quote list) (map mapcols (lambda (col) (symbol col))))))))

(define row_number_scan_partition_expr (lambda (alias mapcols)
	(if (empty_list? mapcols)
		true
		(if (single_source? mapcols)
			(symbol (concat alias "." (car mapcols)))
			(cons (quote list) (map mapcols (lambda (col) (symbol (concat alias "." col)))))))))

(define replace_lowered_row_number_symbol (lambda (alias col expr)
	(if (equal? expr (symbol (concat alias "." col)))
		(quote __row_number)
		(match expr
			(cons head tail) (cons (replace_lowered_row_number_symbol alias col head)
				(map tail (lambda (item) (replace_lowered_row_number_symbol alias col item))))
			_ expr))))

(define build_join_row_number_scan_rows (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr stage_filter)
	(begin
		(define src (car sources))
		(define alias (source_alias src))
		(define stage (nth stage_filter 0))
		(define filter (nth stage_filter 1))
		(define mode (nth filter 0))
		(define limit (nth filter 1))
		(define stripped_condition (nth filter 2))
		(match stage
			'(_ _id _stage_src col sortcols sortdirs _partitioncount mapcols _mapfn _reducefn _reduceinit _facts)
			(begin
				(define filtercols (join_filter_cols_for_alias all_sources default_alias alias stripped_condition))
				(define raw_mapcols (join_cols_for_alias all_sources default_alias alias needed_exprs))
				(define scan_mapcols (merge_unique (list (without_col raw_mapcols col) mapcols)))
				(define table_expr (source_table_expr src))
				(define rewritten_row_expr (replace_lowered_row_number_symbol alias col row_expr))
				(define filter_expr (list (quote lambda)
					(map filtercols (lambda (filter_col) (symbol (concat alias "." filter_col))))
					(list (quote optimize) (lower_column_expr_for_join all_sources default_alias stripped_condition))))
				(define continuation_expr (list (quote lambda) (list (quote __row_number))
					(build_join_scan_rows_with_mapper schema all_sources (cdr sources) default_alias needed_exprs final_condition rewritten_row_expr '() 0 -1 '())))
				(define map_expr (list (quote lambda)
					(map scan_mapcols (lambda (map_col) (symbol (concat alias "." map_col))))
					(list (quote list)
						(row_number_scan_partition_expr alias mapcols)
						continuation_expr)))
				(define reduce_expr (list (quote lambda) (list (quote state) (quote mapped))
					(list (quote begin)
						(list (quote define) (quote rows) (list (quote car) (quote state)))
						(list (quote define) (quote prev_partition) (list (quote cadr) (quote state)))
						(list (quote define) (quote prev_rownum) (list (quote car) (list (quote cdr) (list (quote cdr) (quote state)))))
						(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
						(list (quote define) (quote continuation) (list (quote cadr) (quote mapped)))
						(list (quote define) (quote same_partition) (list (quote and)
							(list (quote not) (list (quote equal?) (quote prev_rownum) 0))
							(list (quote equal?) (quote row_partition) (quote prev_partition))))
						(list (quote define) (quote next_rownum) (list (quote if) (quote same_partition) (list (quote +) (quote prev_rownum) 1) 1))
						(list (quote define) (quote next_rows) (list (quote if)
							(row_number_count_match_expr mode (quote next_rownum) limit)
							(list (quote merge) (quote rows) (list (quote continuation) (quote next_rownum)))
							(quote rows)))
						(list (quote list) (quote next_rows) (quote row_partition) (quote next_rownum)))))
				(list (quote car)
					(list (quote scan_order)
						'(session "__memcp_tx")
						table_expr
						(cons (quote list) filtercols)
						filter_expr
						(quoted_runtime_list sortcols)
						(cons (quote list) (window_scan_dirs sortdirs))
						0
						0
						-1
						(cons (quote list) scan_mapcols)
						map_expr
						reduce_expr
						(list (quote list) (list (quote list)) nil 0)
						(source_outer? src))))
			_ (neumann_fail "build_queryplan" "malformed ROW_NUMBER stage")))))

(define build_join_scan_rows_with_mapper (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value stages)
	(if (empty_list? sources)
		(if (equal? (coalesceNil final_condition true) true)
			(runtime_cons_list_expr (list row_expr))
			(list (quote if)
				(list (quote optimize) (lower_column_expr_for_join all_sources default_alias final_condition))
				(runtime_cons_list_expr (list row_expr))
				(list (quote list))))
		(begin
			(define src (car sources))
			(if (not (source_is_base_table? src))
				(neumann_fail "build_queryplan" "multi-source query-block lowering only supports base tables after untangle")
				true)
			(define alias (source_alias src))
			(define condition (coalesceNil (source_join_expr src) true))
			(define membership (driver_membership_for_source src final_condition))
			(define membership_table_expr (if (nil? membership) nil (recset_project_join_expr_for_membership src membership)))
			(define effective_final_condition (strip_driver_membership_for_source src final_condition membership))
			(define row_number_stage_filter (row_number_stage_for_source stages src condition))
			(define filtercols (join_filter_cols_for_alias all_sources default_alias alias condition))
			(define mapcols (join_cols_for_alias all_sources default_alias alias needed_exprs))
			(define table_expr (coalesceNil membership_table_expr (source_table_expr src)))
			(define filter_expr (list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat alias "." col))))
				(list (quote optimize) (lower_column_expr_for_join all_sources default_alias condition))))
			(define map_expr (list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat alias "." col))))
				(build_join_scan_rows_with_mapper schema all_sources (cdr sources) default_alias needed_exprs effective_final_condition row_expr '() 0 -1 stages)))
			(define reduce_expr (list (quote lambda) (list (quote acc) (quote subrows))
				(list (quote if)
					(list (quote nil?) (quote subrows))
					(quote acc)
					(list (quote merge) (quote acc) (quote subrows)))))
			(if (not (nil? row_number_stage_filter))
				(build_join_row_number_scan_rows schema all_sources sources default_alias needed_exprs effective_final_condition row_expr row_number_stage_filter)
				(if (and (empty_list? order_items) (not (query_limit_active? offset_value limit_value)))
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
						(cons (quote list) (if (empty_list? order_items) '() (scan_order_sort_columns_for_alias src order_items)))
						(cons (quote list) (if (empty_list? order_items) '() (order_dirs order_items)))
						0
						(coalesceNil offset_value 0)
						(coalesceNil limit_value -1)
						(cons (quote list) mapcols)
						map_expr
						reduce_expr
						(list (quote list))
						(source_outer? src))))))))

(define build_join_scan_rows (lambda (schema sources default_alias needed_exprs final_condition fields order_items offset_value limit_value stages)
	(build_join_scan_rows_with_mapper
		schema
		sources
		sources
		default_alias
		needed_exprs
		final_condition
		(list (quote resultrow)
			(cons (quote list) (lower_join_result_fields sources default_alias fields)))
		order_items
		offset_value
		limit_value
		stages)))

(define lower_query_block_as_dataset_rows (lambda (block fields)
	(begin
		(if (empty_list? (qb_sources block))
			(neumann_fail "build_queryplan" "dataset query-block lowering requires a FROM source")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(neumann_fail "build_queryplan" "dataset query-block lowering cannot consume grouped input")
			true)
		(if (or (not (empty_list? (qb_order block))) (or (not (nil? (qb_limit block))) (not (nil? (qb_offset block)))))
			(neumann_fail "build_queryplan" "dataset query-block lowering cannot preserve ORDER/LIMIT yet")
			true)
		(define sources (qb_sources block))
		(define first_alias (source_alias (car sources)))
		(define where_expr (coalesceNil (qb_where block) true))
		(define push_first_where (and (or (empty_list? (qb_stages block)) (query_block_has_only_window_stages? block))
			(and (not (equal? where_expr true))
				(and (not (expr_contains_orc_column? where_expr))
					(empty_list? (external_column_refs_for_alias first_alias where_expr))))))
		(define scan_sources (if push_first_where
			(cons (source_with_join_expr (car sources) (combine_where (source_join_expr (car sources)) where_expr)) (cdr sources))
			sources))
		(define final_condition (if push_first_where true where_expr))
		(define needed_exprs (merge (list
			(extract_assoc fields (lambda (_title expr) expr))
			(list final_condition)
			(source_join_exprs scan_sources))))
		(build_join_scan_rows_with_mapper
			(qb_schema block)
			scan_sources
			scan_sources
			first_alias
			needed_exprs
			final_condition
			(lower_dataset_row_assoc scan_sources first_alias fields)
			'()
			0
			-1
			(qb_stages block)))))

(define lower_multi_source_query_block (lambda (block)
	(begin
		(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(lower_group_stage (make_group_stage_for_query_block block))
			(begin
				(define sources (qb_sources block))
				(define first_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
				(define order_items (coalesceNil (qb_order block) '()))
				(define direct_order (and (equal? first_alias (source_alias (car sources)))
					(order_items_belong_to_source? (car sources) order_items)))
				(define needed_exprs (merge (list
					(extract_assoc fields (lambda (_title expr) expr))
					(list (coalesceNil (qb_where block) true))
					(order_exprs order_items)
					(source_join_exprs sources))))
				(if direct_order
					(build_join_scan_rows (qb_schema block) sources first_alias needed_exprs (coalesceNil (qb_where block) true) fields order_items (qb_offset block) (qb_limit block) (qb_stages block))
					(join_sorted_rows_plan
						(build_join_scan_rows_with_mapper
							(qb_schema block)
							sources
							sources
							first_alias
							needed_exprs
							(coalesceNil (qb_where block) true)
							(lower_join_result_row_assoc sources first_alias fields order_items)
							'()
							0
							-1
							(qb_stages block))
						fields
						order_items
						(qb_offset block)
						(qb_limit block)))))
)))

(define lower_zero_source_query_block (lambda (block)
	(if (equal? (coalesceNil (qb_where block) true) true)
		(list (quote resultrow) (cons (quote list) (map_assoc (qb_fields block) (lambda (_title expr) (lower_scalar_marker_expr expr)))))
		(list (quote if)
			(lower_scalar_marker_expr (qb_where block))
			(list (quote resultrow) (cons (quote list) (map_assoc (qb_fields block) (lambda (_title expr) (lower_scalar_marker_expr expr)))))
			(list (quote list))))))

(define dml_assignment_exprs (lambda (cols)
	(extract_assoc (coalesceNil cols '()) (lambda (_title expr) expr))))

(define lower_dml_update_values (lambda (src cols)
	(map_assoc (coalesceNil cols '()) (lambda (_title expr)
		(lower_column_expr_for_alias src expr)))))

(define dml_target_source (lambda (sources target_schema target_tbl)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (and (source_is_base_table? src)
				(and (equal? (source_schema src) target_schema)
					(equal? (source_relation src) target_tbl)))
				src
				nil)))
		nil)))

(define lower_multi_source_dml_query_block (lambda (block target_schema target_tbl)
	(begin
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(neumann_fail "build_queryplan" "DML over grouped query-block is not implemented yet")
			true)
		(if (query_limit_active? (qb_offset block) (qb_limit block))
			(neumann_fail "build_queryplan" "multi-source DML with ORDER/LIMIT is not implemented yet")
			true)
		(define sources (qb_sources block))
		(define first_alias (source_alias (car sources)))
		(define target_src (dml_target_source sources target_schema target_tbl))
		(if (nil? target_src)
			(neumann_fail "build_queryplan" "DML target relation mismatch")
			true)
		(define target_alias (source_alias target_src))
		(define cols (qb_fields block))
		(define delete_mode (empty_list? (coalesceNil cols '())))
		(define update_ref (list (quote get_column) target_alias false "$update" false))
		(define cond (dml_preserve_driver_membership_probe target_schema (coalesceNil (qb_where block) true)))
		(define needed_exprs (merge (list
			(dml_assignment_exprs cols)
			(list cond)
			(list update_ref)
			(source_join_exprs sources))))
		(define update_fn (lower_column_expr_for_join sources first_alias update_ref))
		(define update_values (if delete_mode
			nil
			(cons (quote list) (map_assoc (coalesceNil cols '()) (lambda (_title expr)
				(lower_column_expr_for_join sources first_alias expr))))))
		(define row_expr (if delete_mode
			(list (quote if) (list update_fn) 1 0)
			(list (quote if) (list update_fn update_values) 1 0)))
		(list (quote reduce)
			(build_join_scan_rows_with_mapper
				(qb_schema block)
				sources
				sources
				first_alias
				needed_exprs
				cond
				row_expr
				'()
				0
				-1
				(qb_stages block))
			(quote +)
			0))))

(define lower_single_source_dml_query_block (lambda (block target_schema target_tbl)
	(begin
		(define src (car (qb_sources block)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "DML target must be a base table")
			true)
		(if (or (not (equal? (source_schema src) target_schema)) (not (equal? (source_relation src) target_tbl)))
			(neumann_fail "build_queryplan" "DML target relation mismatch")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(neumann_fail "build_queryplan" "DML over grouped query-block is not implemented yet")
			true)
		(define alias (source_alias src))
		(define cols (qb_fields block))
		(define delete_mode (empty_list? (coalesceNil cols '())))
		(define cond (dml_preserve_driver_membership_probe target_schema (coalesceNil (qb_where block) true)))
		(define order_items (coalesceNil (qb_order block) '()))
		(define bounded (query_limit_active? (qb_offset block) (qb_limit block)))
		(define filtercols (extract_columns_for_alias src cond))
		(define ordercols (if (empty_list? order_items) '() (order_cols_for_alias src order_items)))
		(define valuecols (merge_unique (map (dml_assignment_exprs cols) (lambda (expr)
			(extract_columns_for_alias src expr)))))
		(define mapcols (cons "$update" (merge_unique (list filtercols valuecols))))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (symbol (concat alias "." col))))
			(list (quote optimize) (lower_column_expr_for_alias src cond))))
		(define update_values (if delete_mode
			nil
			(cons (quote list) (lower_dml_update_values src cols))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col)
				(if (equal? col "$update") (symbol "$update") (symbol (concat alias "." col)))))
			(if delete_mode
				(list (quote if) (list (quote $update)) 1 0)
				(list (quote if) (list (quote $update) update_values) 1 0))))
		(if (and (empty_list? order_items) (not bounded))
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) target_schema target_tbl)
				(cons (quote list) filtercols)
				filter_expr
				(cons (quote list) mapcols)
				map_expr
				(quote +)
				0
				nil
				false)
			(list (quote scan_order)
				'(session "__memcp_tx")
				(list (quote table) target_schema target_tbl)
				(cons (quote list) filtercols)
				filter_expr
				(cons (quote list) ordercols)
				(cons (quote list) (if (empty_list? order_items) '() (order_dirs order_items)))
				0
				(coalesceNil (qb_offset block) 0)
				(coalesceNil (qb_limit block) -1)
				(cons (quote list) mapcols)
				map_expr
				(quote +)
				0
				false)))))

(define lower_dml_query_block_core (lambda (block target_schema target_tbl)
	(if (single_source? (qb_sources block))
		(lower_single_source_dml_query_block block target_schema target_tbl)
		(lower_multi_source_dml_query_block block target_schema target_tbl))))

(define lower_dml_query_block_with_stages (lambda (block target_schema target_tbl)
	(if (empty_list? (qb_stages block))
		(lower_dml_query_block_core block target_schema target_tbl)
		(cons (quote begin)
			(merge (list
				(map (qb_stages block) (lambda (stage)
					(lower_stage_prepare_using (qb_stages block) stage)))
				(list (lower_dml_query_block_core (query_block_without_stages_after_prepare_using (qb_stages block) block) target_schema target_tbl))))))))

(define lower_dml_union_block_with_stages (lambda (block target_schema target_tbl)
	(cons (quote begin)
		(map (union_branches block) (lambda (branch)
			(if (not (query_block? branch))
				(neumann_fail "build_queryplan" "DML UNION branch lowering expects query-block branches")
				(lower_dml_query_block_with_stages branch target_schema target_tbl)))))))

(define projection_titles (lambda (fields)
	(match (coalesceNil fields '())
		(cons title (cons _expr rest)) (cons title (projection_titles rest))
		_ '())))

(define projection_exprs (lambda (fields)
	(match (coalesceNil fields '())
		(cons _title (cons expr rest)) (cons expr (projection_exprs rest))
		_ '())))

(define projection_with_titles (lambda (titles exprs)
	(match (coalesceNil titles '())
		(cons title title_rest) (match (coalesceNil exprs '())
			(cons expr expr_rest) (cons title (cons expr (projection_with_titles title_rest expr_rest)))
			_ (neumann_fail "build_queryplan" "UNION branch column count mismatch"))
		_ (if (empty_list? (coalesceNil exprs '()))
			'()
			(neumann_fail "build_queryplan" "UNION branch column count mismatch")))))

(define union_align_branch_fields (lambda (branch titles width)
	(begin
		(if (not (query_block? branch))
			(neumann_fail "build_queryplan" "UNION branch lowering expects query-block branches")
			true)
		(if (not (equal? (count (qb_fields branch)) width))
			(neumann_fail "build_queryplan" "UNION branch column count mismatch")
			true)
		(make_query_block
			(qb_schema branch)
			(qb_sources branch)
			(projection_with_titles titles (projection_exprs (qb_fields branch)))
			(qb_where branch)
			(qb_group branch)
			(qb_having branch)
			(qb_order branch)
			(qb_limit branch)
			(qb_offset branch)
			(qb_hidden branch)
			(qb_stages branch)
			(qb_facts branch)))))

(define projection_index_by_title (lambda (titles title)
	(reduce (produceN (count titles)) (lambda (found i)
		(if (not (nil? found))
			found
			(if (equal?? (nth titles i) title) i nil)))
		nil)))

(define union_order_position (lambda (titles expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase col _col_ignorecase) (projection_index_by_title titles col)
		((quote get_column) _tblvar _tbl_ignorecase col _col_ignorecase) (projection_index_by_title titles col)
		_ (if (and (number? expr) (and (> expr 0) (<= expr (count titles))))
			(- expr 1)
			nil))))

(define union_order_positions (lambda (titles order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (begin
				(define pos (union_order_position titles expr))
				(if (nil? pos)
					(neumann_fail "build_queryplan" "UNION ALL ORDER BY column not found")
					pos))
			_ (neumann_fail "build_queryplan" "malformed UNION ORDER BY item"))))))

(define union_order_dirs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) dir
			_ (neumann_fail "build_queryplan" "malformed UNION ORDER BY item"))))))

(define union_ordered_branch_supported? (lambda (branch)
	(and (query_block? branch)
		(and (single_source? (qb_sources branch))
			(and (empty_list? (qb_stages branch))
				(and (empty_list? (qb_group branch))
					(and (nil? (qb_having branch))
						(and (not (query_block_has_aggregates? branch))
							(and (empty_list? (qb_order branch))
								(and (nil? (qb_limit branch))
									(nil? (qb_offset branch))))))))))))

(define grouped_union_branch? (lambda (branch)
	(and (query_block? branch)
		(and (single_source? (qb_sources branch))
			(and (empty_list? (qb_stages branch))
				(and (empty_list? (qb_order branch))
					(and (nil? (qb_limit branch))
						(and (nil? (qb_offset branch))
							(or (not (empty_list? (qb_group branch)))
								(or (not (nil? (qb_having branch)))
									(query_block_has_aggregates? branch)))))))))))

(define grouped_union_branch_stream_plan (lambda (branch)
	(begin
		(define src (car (qb_sources branch)))
		(if (not (source_is_base_table? src))
			nil
			(begin
				(define stage (make_group_stage_for_block branch src))
				(define final_block (group_stage_final_block stage (group_stage_final_extra_sources stage)))
				(if (union_ordered_branch_supported? final_block)
					(list (list (lower_group_stage_prepare_using (list stage) stage)) final_block)
					nil))))))

(define union_ordered_branch_stream_plan (lambda (branch)
	(if (not (query_block? branch))
		nil
		(if (union_ordered_branch_supported? branch)
			(list '() branch)
			(if (grouped_union_branch? branch)
				(grouped_union_branch_stream_plan branch)
				nil)))))

(define union_ordered_branch_streamable? (lambda (branch)
	(not (nil? (union_ordered_branch_stream_plan branch)))))

(define union_sort_column_for_alias (lambda (src expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) (order_column_for_alias src expr)
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) (order_column_for_alias src expr)
		_ (begin
			(define cols (extract_columns_for_alias src expr))
			(list (quote lambda)
				(map cols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(lower_column_expr_for_alias src expr))))))

(define union_ordered_scan_spec (lambda (branch titles order_positions)
	(begin
		(if (not (union_ordered_branch_supported? branch))
			(neumann_fail "build_queryplan" "UNION ALL ORDER BY requires simple single-source branches")
			true)
		(define src (car (qb_sources branch)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "UNION ALL ORDER BY requires base table branches")
			true)
		(define alias (source_alias src))
		(define fields (qb_fields branch))
		(define exprs (projection_exprs fields))
		(define condition (combine_where (qb_where branch) (source_join_expr src)))
		(define order_exprs (map order_positions (lambda (pos) (nth exprs pos))))
		(define filtercols (extract_columns_for_alias src condition))
		(define outputcols (merge_unique (map exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define sortcols (map order_exprs (lambda (expr) (union_sort_column_for_alias src expr))))
		(define sort_input_cols (merge_unique (map order_exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define mapcols (merge_unique (list outputcols sort_input_cols)))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (symbol (concat alias "." col))))
			(list (quote optimize) (lower_column_expr_for_alias src condition))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (symbol (concat alias "." col))))
			(list (quote resultrow)
				(cons (quote list) (merge (map (produceN (count titles)) (lambda (i)
					(list (nth titles i) (lower_column_expr_for_alias src (nth exprs i))))))))))
		(list
			(source_table_expr src)
			filtercols
			filter_expr
			sortcols
			mapcols
			map_expr))))

(define lower_union_all_ordered (lambda (block titles width)
	(begin
		(define branches (union_branches block))
		(define aligned (map branches (lambda (branch) (union_align_branch_fields branch titles width))))
		(define order_positions (union_order_positions titles (union_order block)))
		(define plans (map aligned union_ordered_branch_stream_plan))
		(if (reduce plans (lambda (ok plan) (and ok (not (nil? plan)))) true)
			(begin
				(define prepared (map plans (lambda (plan) (nth plan 1))))
				(define prepares (merge (map plans (lambda (plan) (nth plan 0)))))
				(define specs (map prepared (lambda (branch) (union_ordered_scan_spec branch titles order_positions))))
				(define scan_plan (list (quote scan_order_multi)
					'(session "__memcp_tx")
					(cons (quote list) (map specs (lambda (spec) (nth spec 0))))
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 1)))))
					(cons (quote list) (map specs (lambda (spec) (nth spec 2))))
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 3)))))
					(cons (quote list) (union_order_dirs (union_order block)))
					nil
					nil
					0
					(coalesceNil (union_offset block) 0)
					(coalesceNil (union_limit block) -1)
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 4)))))
					(cons (quote list) (map specs (lambda (spec) (nth spec 5))))))
				(if (empty_list? prepares)
					scan_plan
					(cons (quote begin) (merge (list prepares (list scan_plan))))))
			(neumann_fail "build_queryplan" "UNION ALL ORDER BY requires streamable branches")))))

(define union_materialized_order_fields (lambda (branch order_positions)
	(begin
		(define exprs (projection_exprs (qb_fields branch)))
		(merge (list
			(qb_fields branch)
			(merge (map (produceN (count order_positions)) (lambda (i)
				(list (concat "__order_" i) (nth exprs (nth order_positions i)))))))))))

(define query_block_with_fields (lambda (block fields)
	(make_query_block
		(qb_schema block)
		(qb_sources block)
		fields
		(qb_where block)
		(qb_group block)
		(qb_having block)
		(qb_order block)
		(qb_limit block)
		(qb_offset block)
		(qb_hidden block)
		(qb_stages block)
		(qb_facts block))))

(define lower_query_block_capture_rows (lambda (block)
	(begin
		(define state (symbol "__union_rows"))
		(define emit (symbol "__union_emit"))
		(define row (symbol "__union_row"))
		(list
			(list (quote lambda) (list state emit)
				(list (quote begin)
					(list state "rows" (list (quote list)))
					(list (quote define) (quote resultrow)
						(list (quote lambda) (list row)
							(list state "rows"
								(list (quote append) (list state "rows") (list (quote cons) row (list (quote list)))))))
					(lower_query_block_with_stages block)
					(list state "rows")))
			(list (quote newsession))
			(quote resultrow)))))

(define lower_union_all_materialized_ordered (lambda (block titles width)
	(begin
		(define branches (union_branches block))
		(define aligned (map branches (lambda (branch) (union_align_branch_fields branch titles width))))
		(define order_positions (union_order_positions titles (union_order block)))
		(define rows_plan (cons (quote merge) (map aligned (lambda (branch)
			(lower_query_block_capture_rows
				(query_block_with_fields branch (union_materialized_order_fields branch order_positions)))))))
		(join_sorted_rows_plan
			rows_plan
			(qb_fields (car aligned))
			(union_order block)
			(union_offset block)
			(union_limit block)))))

(define union_direct_order_supported? (lambda (block)
	(begin
		(define branches (union_branches block))
		(if (empty_list? branches)
			true
			(begin
				(define first_branch (car branches))
				(if (not (query_block? first_branch))
					false
					(begin
						(define titles (projection_titles (qb_fields first_branch)))
						(define order_positions (union_order_positions titles (union_order block)))
						(reduce branches (lambda (ok branch)
							(if (not ok)
								false
								(if (not (query_block? branch))
									false
									(begin
										(define exprs (projection_exprs (qb_fields branch)))
										(reduce order_positions (lambda (inner_ok pos)
											(and inner_ok (direct_column_ref? (nth exprs pos))))
											true)))))
							true))))))))

(define lower_union_all_successive (lambda (block)
	(begin
		(define branches (union_branches block))
		(if (empty_list? branches)
			nil
			(begin
				(define first_branch (car branches))
				(if (not (query_block? first_branch))
					(neumann_fail "build_queryplan" "UNION branch lowering expects query-block branches")
					true)
				(define titles (projection_titles (qb_fields first_branch)))
				(define width (count (qb_fields first_branch)))
				(if (not (empty_list? (union_order block)))
					(if (reduce branches (lambda (ok branch)
						(and ok (union_ordered_branch_streamable? (union_align_branch_fields branch titles width))))
						true)
						(lower_union_all_ordered block titles width)
						(lower_union_all_materialized_ordered block titles width))
					(if (or (not (nil? (union_limit block))) (not (nil? (union_offset block))))
						(lower_union_all_ordered block titles width)
						(cons (quote begin)
							(map branches (lambda (branch)
								(lower_query_block_with_stages (union_align_branch_fields branch titles width))))))))))))

(define wrap_plan_with_distinct_resultrow (lambda (plan)
	(begin
		(define seen (symbol "__union_distinct_seen"))
		(define emit (symbol "__union_distinct_resultrow"))
		(define row (symbol "__union_distinct_row"))
		(define key (symbol "__union_distinct_key"))
		(list (quote begin)
			(list (quote define) seen (list (quote newsession)))
			(list (quote define) emit (quote resultrow))
			(list (quote define) (quote resultrow)
				(list (quote lambda) (list row)
					(list (quote begin)
						(list (quote define) key (list (quote serialize) row))
						(list (quote if)
							(list seen key)
							nil
							(list (quote begin)
								(list seen key true)
								(list emit row))))))
			plan))))

(define lower_union_block (lambda (block)
	(if (equal? (union_mode block) (quote all))
		(lower_union_all_successive block)
		(if (or (equal? (union_mode block) (quote distinct)) (equal? (union_mode block) (quote union_distinct)))
			(if (and (not (empty_list? (union_order block))) (not (union_direct_order_supported? block)))
				(wrap_plan_with_distinct_resultrow
					(lower_union_all_successive (make_union_block (quote all) (union_branches block) '() nil nil (union_facts block))))
				(wrap_plan_with_distinct_resultrow (lower_union_all_successive block)))
			(neumann_fail "build_queryplan" "unknown UNION mode")))))

(define build_queryplan (lambda (ir)
	(begin
		(require_unnested_node "build_queryplan input" (ir_root ir))
		(match (ir_return ir)
			(symbol rows) (match (logical_op (ir_root ir))
				(symbol query-block) (lower_query_block_with_stages (ir_root ir))
				(symbol union-block) (lower_union_block (ir_root ir))
				_ (neumann_fail "build_queryplan" "unknown logical root"))
			((symbol dml) target_schema target_tbl) (match (logical_op (ir_root ir))
				(symbol query-block) (lower_dml_query_block_with_stages (ir_root ir) target_schema target_tbl)
				(symbol union-block) (lower_dml_union_block_with_stages (ir_root ir) target_schema target_tbl)
				_ (neumann_fail "build_queryplan" "DML lowering expects a query-block root"))
			_ (neumann_fail "build_queryplan" "DML lowering is intentionally not scaffolded yet")))))

(define neumann_compile_pipeline (lambda (ast)
	(build_queryplan
		(join_reorder
			(untangle_query_term ast nil)))))

(define neumann_compile_ir_pipeline (lambda (ir)
	(build_queryplan
		(join_reorder
			(require_flat_stage_dependencies "compile_ir" (normalize_stage_dependencies ir))))))

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

(define build_dml_plan (lambda (schema tbl _tblalias all_defs cols condition order limit offset)
	(begin
		(define query (make_query_block
			schema
			all_defs
			cols
			(coalesceNil condition true)
			'() nil
			(coalesceNil order '())
			limit
			offset
			'() '()
			(list (list (quote dml) true))))
		(neumann_compile_ir_pipeline
			(ir_with_return (untangle_query_term query nil) (list (quote dml) schema tbl))))))

(define sql_truncate (lambda (schema tbl)
	(build_dml_plan schema tbl nil
		(list (list tbl schema tbl false nil))
		nil true nil nil nil)))

(define explain_union_ir_metadata (lambda (ir)
	(begin
		(define root (ir_root ir))
		(if (union_block? root)
			(concat "\nbranches " (count (union_branches root)) " " (union_mode root))
			""))))

(define explain_queryplan_ir (lambda (query)
	(begin
		(define ir (untangle_query_term query nil))
		(list (quote resultrow)
			(list (quote list)
				"ir"
				(concat
					(pretty_print ir (settings "ExplainWidth"))
					(explain_union_ir_metadata ir)))))))

(define explain_queryplan_reorder (lambda (query)
	(list (quote resultrow)
		(list (quote list)
			"reorder"
			(pretty_print
				(join_reorder (untangle_query_term query nil))
				(settings "ExplainWidth"))))))
