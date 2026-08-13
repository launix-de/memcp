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
build_queryplan resolves a stage-output through a group-cache decision.  The
default group cache is a generated keytable; later optimizers may choose an existing
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
		(lambda ()
			(begin
				(define handle (table schema tbl))
				(if handle (show handle) (show schema tbl))))
		(lambda (_e) '()))))

(define qassoc_get (lambda (xs key default)
	(match (coalesceNil xs '())
		(cons entry rest) (match entry
			(cons k values) (if (equal? k key)
				(if (equal? (count values) 1) (car values) values)
				(qassoc_get rest key default))
			_ (qassoc_get rest key default))
		_ default)))

(define qassoc_set (lambda (xs key value)
	(cons (list key value)
		(filter (coalesceNil xs '()) (lambda (entry) (match entry
			(cons k _) (not (equal? k key))
			true))))))

(define qassoc_set_without (lambda (xs key value removed_key)
	(cons (list key value)
		(filter (coalesceNil xs '()) (lambda (entry) (match entry
			(cons k _) (and (not (equal? k key)) (not (equal? k removed_key)))
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

(define logical_stage_key (lambda (stage)
	(if (group_stage? stage)
		(concat "group:" (gs_id stage))
		(if (orc_stage? stage)
			(concat "orc:" (nth stage 1))
			(if (window_stage? stage)
				(concat "window:" (nth stage 1))
				nil)))))

(define unique_stages_by_id_acc (lambda (stages seen)
	(match (coalesceNil stages '())
		(cons stage rest) (begin
			(define key (logical_stage_key stage))
			(if (and (not (nil? key)) (has_assoc? seen key))
				(unique_stages_by_id_acc rest seen)
				(cons stage
					(unique_stages_by_id_acc rest
						(if (nil? key) seen (set_assoc seen key true))))))
		_ '())))

(define unique_stages_by_id (lambda (stages)
	(unique_stages_by_id_acc stages '())))

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
		(cons head tail) (if (or (equal? head (quote and)) (equal? head (symbol "and")))
			(merge (map tail split_and_terms))
			(list expr))
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

(define expr_only_refs_alias? (lambda (default_alias alias expr)
	(match expr
		((symbol get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		((quote get_column) tblvar _ _ _) (equal?? (resolve_column_alias tblvar default_alias) alias)
		(cons _head tail) (reduce tail (lambda (ok item) (and ok (expr_only_refs_alias? default_alias alias item))) true)
		_ true)))

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

(define btw2025_expr_outer_column_refs (lambda (expr inner_sources outer_sources)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (if (nil? tblvar)
				(if (nil? (source_for_unqualified_column inner_sources nil col col_ignorecase))
					(source_for_unqualified_column outer_sources nil col col_ignorecase)
					nil)
				(source_for_alias outer_sources nil tblvar tbl_ignorecase)))
			(if (nil? src)
				'()
				(list (list
					(quote get_column)
					(if (nil? tblvar) (source_alias src) tblvar)
					(if (nil? tblvar) false tbl_ignorecase)
					col
					col_ignorecase))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(btw2025_expr_outer_column_refs
			(list (quote get_column) tblvar tbl_ignorecase col col_ignorecase)
			inner_sources
			outer_sources)
		(cons _head tail) (merge_unique (map tail (lambda (item)
			(btw2025_expr_outer_column_refs item inner_sources outer_sources))))
		_ '())))

(define btw2025_terms_outer_column_refs (lambda (terms inner_sources outer_sources)
	(merge_unique (map (coalesceNil terms '()) (lambda (term)
		(btw2025_expr_outer_column_refs term inner_sources outer_sources))))))

(define btw2025_sources_outer_column_refs (lambda (sources inner_sources outer_sources)
	(merge_unique (map (coalesceNil sources '()) (lambda (src)
		(btw2025_expr_outer_column_refs
			(coalesceNil (source_join_expr src) true)
			inner_sources
			outer_sources))))))

/* Session reads affect a dependent helper like outer-column reads do. Keep
their expressions in Domain D so reusable group caches are keyed by the binding
instead of capturing whichever session populated the cache first. */
(define query_expr_session_reads (lambda (expr)
	(match expr
		((symbol session) "__memcp_tx") '()
		((quote session) "__memcp_tx") '()
		((symbol session) key) (list (list (quote session) key))
		((quote session) key) (list (list (quote session) key))
		(cons head tail) (merge_unique (cons
			(query_expr_session_reads head)
			(map tail query_expr_session_reads)))
		_ '())))

(define query_session_read? (lambda (expr)
	(match expr
		((symbol session) "__memcp_tx") false
		((quote session) "__memcp_tx") false
		((symbol session) _key) true
		((quote session) _key) true
		_ false)))

(define session_domain_pairs (lambda (node)
	(map (query_expr_session_reads node) (lambda (expr) (list expr expr)))))

(define presence_session_domain_pairs (lambda (block)
	(session_domain_pairs (list
		(qb_sources block)
		(qb_where block)
		(qb_group block)
		(qb_having block)
		(qb_order block)
		(qb_limit block)
		(qb_offset block)
		(qb_stages block)))))

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

(define expr_refs_sources? (lambda (default_alias sources expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (nil? tblvar)
			(not (nil? (source_for_unqualified_column sources default_alias col col_ignorecase)))
			(not (nil? (source_for_alias sources default_alias tblvar tbl_ignorecase))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(expr_refs_sources? default_alias sources (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons _head tail) (reduce tail (lambda (found item)
			(or found (expr_refs_sources? default_alias sources item)))
			false)
		_ false)))

(define qualify_unqualified_column_for_sources (lambda (sources expr)
	(match expr
		((symbol inner_select) _subquery) expr
		((quote inner_select) _subquery) expr
		((symbol inner_select_exists) _subquery) expr
		((quote inner_select_exists) _subquery) expr
		((symbol inner_select_in) probe subquery)
		(list (quote inner_select_in) (qualify_unqualified_column_for_sources sources probe) subquery)
		((quote inner_select_in) probe subquery)
		(list (quote inner_select_in) (qualify_unqualified_column_for_sources sources probe) subquery)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (nil? tblvar)
			(begin
				(define src (source_for_unqualified_column sources nil col col_ignorecase))
				(if (nil? src)
					expr
					(list (quote get_column) (source_alias src) false col col_ignorecase)))
			expr)
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(qualify_unqualified_column_for_sources sources (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons (qualify_unqualified_column_for_sources sources head) (map tail (lambda (item)
			(qualify_unqualified_column_for_sources sources item))))
		_ expr)))

(define session_dependency_expr? (lambda (expr)
	(match expr
		((symbol session) key) (not (equal? key "__memcp_tx"))
		((quote session) key) (not (equal? key "__memcp_tx"))
		_ false)))

(define expr_contains_session_dependency? (lambda (expr)
	(if (session_dependency_expr? expr)
		true
		(match expr
			(cons head tail) (or
				(expr_contains_session_dependency? head)
				(reduce tail (lambda (found item)
					(or found (expr_contains_session_dependency? item)))
					false))
			_ false))))

(define exists_correlation_pair (lambda (inner_default inner_sources outer_sources term)
	(match term
		((symbol equal??) a b) (begin
			(define a_inner (expr_refs_sources? inner_default inner_sources a))
			(define b_inner (expr_refs_sources? inner_default inner_sources b))
			(define a_outer (and (not a_inner) (expr_refs_sources? nil outer_sources a)))
			(define b_outer (and (not b_inner) (expr_refs_sources? nil outer_sources b)))
			(if (and a_inner b_outer)
				(list a (qualify_unqualified_column_for_sources outer_sources b))
				(if (and b_inner a_outer)
					(list b (qualify_unqualified_column_for_sources outer_sources a))
					nil)))
		((quote equal??) a b) (exists_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		((symbol equal?) a b) (exists_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		((quote equal?) a b) (exists_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		_ nil)))

(define presence_correlation_pair (lambda (inner_default inner_sources outer_sources term)
	(match term
		((symbol equal??) a b) (begin
			(define a_inner (expr_refs_sources? inner_default inner_sources a))
			(define b_inner (expr_refs_sources? inner_default inner_sources b))
			(define a_outer (and (not a_inner) (or
				(expr_refs_sources? nil outer_sources a)
				(session_dependency_expr? a))))
			(define b_outer (and (not b_inner) (or
				(expr_refs_sources? nil outer_sources b)
				(session_dependency_expr? b))))
			(if (and a_inner b_outer)
				(list a (qualify_unqualified_column_for_sources outer_sources b))
				(if (and b_inner a_outer)
					(list b (qualify_unqualified_column_for_sources outer_sources a))
					nil)))
		((quote equal??) a b) (presence_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		((symbol equal?) a b) (presence_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		((quote equal?) a b) (presence_correlation_pair inner_default inner_sources outer_sources (list (quote equal??) a b))
		_ nil)))

(define unique_correlation_pairs (lambda (pairs)
	(reduce (coalesceNil pairs '()) (lambda (acc pair)
		(if (contains? acc pair)
			acc
			(merge acc (list pair))))
		'())))

(define correlation_pair_domain_key (lambda (pair)
	(nth pair 1)))

(define correlation_pair_preferred? (lambda (current candidate)
	(and
		(expr_refs_stage_output_alias? (nth current 0))
		(not (expr_refs_stage_output_alias? (nth candidate 0))))))

(define upsert_domain_correlation_pair (lambda (pairs pair)
	(begin
		(define domain_key (correlation_pair_domain_key pair))
		(define result (reduce pairs (lambda (acc current)
			(begin
				(define replaced (nth acc 0))
				(define rewritten (nth acc 1))
				(if (equal? (correlation_pair_domain_key current) domain_key)
					(list true (merge rewritten (list
						(if (correlation_pair_preferred? current pair) pair current))))
					(list replaced (merge rewritten (list current))))))
			(list false '())))
		(if (nth result 0)
			(nth result 1)
			(merge (nth result 1) (list pair))))))

(define domain_correlation_pairs (lambda (pairs)
	(reduce (unique_correlation_pairs pairs) upsert_domain_correlation_pair '())))

(define correlation_inner_keys (lambda (inner_default pairs)
	(map (coalesceNil pairs '()) (lambda (pair)
		(canonical_column_expr_for_alias inner_default (nth pair 0))))))

(define simple_column_expr? (lambda (expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) true
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) true
		_ false)))

(define membership_stage_source_alias? (lambda (stages sources alias)
	(begin
		(define src (source_for_alias sources nil alias false))
		(if (or (nil? src) (not (stage_output_relation? (source_relation src))))
			false
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(or
					(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote in_membership))
					(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote in_rhs_nulls))))))))

(define scalar_stage_inner_keys_for_correlations (lambda (inner_default stages sources pairs)
	(map (coalesceNil pairs '()) (lambda (pair)
		(begin
			(define inner_expr (nth pair 0))
			(define domain_expr (nth pair 1))
			(define inner_alias (expr_source_alias inner_expr))
			(if (and
				(simple_column_expr? domain_expr)
				(and (not (nil? inner_alias))
					(membership_stage_source_alias? stages sources inner_alias)))
				domain_expr
				(canonical_column_expr_for_alias inner_default inner_expr)))))))

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

(define make_btw2025_unnesting_info (lambda (handle parent ancestors outer_refs domain accessing accessing_after_simple cclasses repr)
	(list
		(quote btw2025-unnesting-info)
		handle
		parent
		(coalesceNil ancestors '())
		(coalesceNil outer_refs '())
		(coalesceNil domain '())
		(coalesceNil accessing '())
		(coalesceNil accessing_after_simple '())
		(coalesceNil cclasses '())
		(coalesceNil repr '()))))

(define btw2025_info_handle (lambda (info) (nth info 1)))
(define btw2025_info_parent (lambda (info) (nth info 2)))
(define btw2025_info_ancestors (lambda (info) (nth info 3)))
(define btw2025_info_outer_refs (lambda (info) (nth info 4)))
(define btw2025_info_domain (lambda (info) (nth info 5)))
(define btw2025_info_accessing (lambda (info) (nth info 6)))
(define btw2025_info_accessing_after_simple (lambda (info) (nth info 7)))
(define btw2025_info_cclasses (lambda (info) (nth info 8)))
(define btw2025_info_repr (lambda (info) (nth info 9)))

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

(define source_join_correlation_pairs (lambda (inner_default inner_sources outer_sources sources)
	(merge
		(map
			(coalesceNil sources '())
			(lambda (src)
				(filter
					(map
						(split_and_terms (coalesceNil (source_join_expr src) true))
						(lambda (term)
							(exists_correlation_pair inner_default inner_sources outer_sources term)))
					(lambda (pair)
						(not (nil? pair)))))))))

(define source_without_outer_join_terms (lambda (inner_default inner_sources outer_sources src)
	(begin
		(define terms (split_and_terms (coalesceNil (source_join_expr src) true)))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(list
			(source_alias src)
			(source_schema src)
			(source_relation src)
			(source_outer? src)
			(combine_where_terms local_terms true)))))

(define sources_without_outer_join_terms (lambda (inner_default inner_sources outer_sources sources)
	(map (coalesceNil sources '()) (lambda (src)
		(source_without_outer_join_terms inner_default inner_sources outer_sources src)))))

(define btw2025_local_where_terms_after_simple (lambda (inner_default inner_sources outer_sources block)
	(filter
		(split_and_terms (coalesceNil (qb_where block) true))
		(lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term))))))

(define btw2025_accessing_after_simple (lambda (block outer_sources)
	(if (not (query_block? block))
		'()
		(begin
			(define inner_default (if (empty_list? (qb_sources block)) nil (source_alias (car (qb_sources block)))))
			(define inner_sources (qb_sources block))
			(define outer_aliases (source_aliases outer_sources))
			(define local_terms (btw2025_local_where_terms_after_simple inner_default inner_sources outer_sources block))
			(define local_sources (sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources block)))
			(merge_unique (list
				(btw2025_sources_accessing_aliases local_sources outer_aliases)
				(btw2025_fields_accessing_aliases (qb_fields block) outer_aliases)
				(btw2025_fields_accessing_aliases (qb_hidden block) outer_aliases)
				(btw2025_terms_accessing_aliases local_terms outer_aliases)
				(btw2025_expr_accessing_aliases (coalesceNil (qb_having block) true) outer_aliases)
				(btw2025_order_accessing_aliases (qb_order block) outer_aliases)))))))

(define btw2025_stage_facts (lambda (block outer_sources lookup_pairs residual_outer_refs pending_info)
	(begin
		(define inner_default (if (or (not (query_block? block)) (empty_list? (qb_sources block))) nil (source_alias (car (qb_sources block)))))
		(define outer_aliases (source_aliases outer_sources))
		(define accessing (btw2025_query_block_accessing_aliases block outer_sources))
		(define accessing_after_simple (btw2025_accessing_after_simple block outer_sources))
		(define domain (merge_unique (list
			(correlation_domain lookup_pairs)
			residual_outer_refs)))
		(define key_accessing_after_simple (merge_unique (map (correlation_inner_keys inner_default lookup_pairs) (lambda (key)
			(btw2025_expr_accessing_aliases key outer_aliases)))))
		(define residual_accessing_after_simple (merge_unique (list accessing_after_simple key_accessing_after_simple)))
		(define cclasses (btw2025_cclasses_for_pairs lookup_pairs))
		(define repr (btw2025_repr_for_pairs inner_default lookup_pairs))
		(define handle (if (nil? pending_info) nil (btw2025_info_handle pending_info)))
		(define parent (if (nil? pending_info) nil (btw2025_info_parent pending_info)))
		(define ancestors (if (nil? pending_info) '() (btw2025_info_ancestors pending_info)))
		(define info (make_btw2025_unnesting_info handle parent ancestors domain domain accessing residual_accessing_after_simple cclasses repr))
		(list
			(list (quote btw2025_accessing) accessing)
			(list (quote btw2025_accessing_after_simple) residual_accessing_after_simple)
			(list (quote btw2025_simple_d_eliminated) (and (not (empty_list? accessing)) (empty_list? residual_accessing_after_simple)))
			(list (quote btw2025_domain) domain)
			(list (quote btw2025_cclasses) cclasses)
			(list (quote btw2025_repr) repr)
			(list (quote btw2025_handle) handle)
			(list (quote btw2025_parent) parent)
			(list (quote btw2025_ancestors) ancestors)
			(list (quote btw2025_info) info)
			(list (quote btw2025_lookup_keys) (merge_unique (list
				(correlation_lookup_keys lookup_pairs)
				residual_outer_refs)))))))

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

/* One aggregate encodes empty=0, nonempty=1, and contains-NULL=2. */
(define in_rhs_state_descriptor (lambda (rhs_expr)
	(list (list (quote if) (list (quote nil?) rhs_expr) 2 1) (quote max) 0)))

(define membership_count_expr (lambda (stage_alias ag)
	(list (quote coalesceNil)
		(list (quote get_column) stage_alias false (aggregate_col_name ag) false)
		0)))

/* Canonical positive-WHERE membership primitive. Its stage-output source
provides the lookup relation; UNKNOWN and FALSE are intentionally equivalent in
this truth context. Reordering may later replace this semantic primitive with a
physical membership probe. */
(define in_membership_truth_expr (lambda (probe stage_alias match_ag)
	(list (quote membership_truth) probe stage_alias (aggregate_col_name match_ag))))

(define expand_in_membership_truth_expr (lambda (probe stage_alias count_col)
	(list (quote and)
		(list (quote not) (list (quote nil?) probe))
		(list (quote >)
			(list (quote coalesceNil)
				(list (quote get_column) stage_alias false count_col false)
				0)
			0))))

(define in_membership_expr (lambda (probe match_alias match_ag null_alias rhs_state_ag)
	(begin
		(define match_count (membership_count_expr match_alias match_ag))
		(define rhs_state (membership_count_expr null_alias rhs_state_ag))
		(list (quote if)
			(list (quote nil?) probe)
			(list (quote if) (list (quote >) rhs_state 0) nil false)
			(list (quote if)
				(list (quote >) match_count 0)
				true
				(list (quote if)
					(list (quote equal??) rhs_state 2)
					nil
					false))))))

(define not_in_membership_expr (lambda (probe match_alias match_ag null_alias rhs_state_ag)
	(begin
		(define match_count (membership_count_expr match_alias match_ag))
		(define rhs_state (membership_count_expr null_alias rhs_state_ag))
		(list (quote if)
			(list (quote nil?) probe)
			(list (quote if) (list (quote >) rhs_state 0) nil true)
			(list (quote if)
				(list (quote >) match_count 0)
				false
				(list (quote if)
					(list (quote equal??) rhs_state 2)
					nil
					true))))))

(define scalar_first_probe_expr (lambda (stage col stages)
	(list (quote scalar_first_probe) stage col stages)))

(define scalar_aggregate_probe_expr (lambda (stage col)
	(list (quote scalar_aggregate_probe) stage col)))

(define scalar_cardinality_probe_expr (lambda (stage col)
	(list (quote scalar_cardinality_probe) stage col)))

(define scalar_aggregate_probe_outer_exprs (lambda (stage)
	(merge (list
		(qassoc_get (gs_facts stage) (quote lookup-keys) '())
		(list (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))))))

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

(define grouped_exists_inner_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (not (empty_list? (qb_group inner)))
				(and (empty_list? (qb_order inner))
					(and (nil? (qb_limit inner))
						(nil? (qb_offset inner)))))))))

(define normalize_membership_query_block (lambda (inner)
	(if (and (query_block? inner)
		(and (not (empty_list? (qb_order inner)))
			(and (nil? (qb_limit inner)) (nil? (qb_offset inner)))))
		(make_query_block
			(qb_schema inner)
			(qb_sources inner)
			(qb_fields inner)
			(qb_where inner)
			(qb_group inner)
			(qb_having inner)
			'()
			(qb_limit inner)
			(qb_offset inner)
			(qb_hidden inner)
			(qb_stages inner)
			(qb_facts inner))
		inner)))

(define membership_inner_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_group inner))
				(nil? (qb_having inner)))))))

(define scalar_once_supported? (lambda (inner)
	(and (query_block? inner)
		(and (scalar_source_shape_supported? (qb_sources inner))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (or (nil? (qb_offset inner)) (not (empty_list? (qb_order inner))))
						(equal? (qb_limit inner) 1))))))))

(define single_row_derived_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_group inner))
				(and (nil? (qb_having inner))
					(and (not (query_block_has_aggregates? inner))
						(and (or (nil? (qb_offset inner)) (not (empty_list? (qb_order inner))))
							(equal? (qb_limit inner) 1)))))))))

(define scalar_aggregate_supported? (lambda (inner)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (empty_list? (qb_order inner))
				(and (or (nil? (qb_limit inner))
					(or (equal? (qb_limit inner) 1)
						(and (empty_list? (qb_group inner))
							(query_block_has_aggregates? inner))))
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

(define scalar_query_probe_empty (quote __scalar_query_probe_empty))
(define scalar_query_probe_reduce_first (lambda ()
	(list (quote lambda)
		(list (quote a) (quote b))
		(list (quote if)
			(list (quote and)
				(list (quote symbol?) (quote a))
				(list (quote equal?) (quote a) (list (quote quote) scalar_query_probe_empty)))
			(quote b)
			(quote a)))))

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

(define expr_refs_one_of_aliases? (lambda (expr aliases)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) (contains? aliases tblvar)
		((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) (contains? aliases tblvar)
		(cons _head tail) (reduce tail (lambda (found item)
			(or found (expr_refs_one_of_aliases? item aliases)))
			false)
		_ false)))

(define query_block_base_aliases (lambda (block)
	(map
		(filter (qb_sources block) (lambda (src) (not (source_is_stage_output? src))))
		source_alias)))

(define scalar_first_query_stage_output_projection? (lambda (stage value_expr)
	(and (query_block? (gs_input stage))
		(and (expr_refs_stage_output_alias? value_expr)
			(not (expr_refs_one_of_aliases? value_expr (query_block_base_aliases (gs_input stage))))))))

(define scalar_first_query_stage_output_key? (lambda (stage)
	(and (query_block? (gs_input stage))
		(reduce (gs_keys stage) (lambda (found key)
			(or found (expr_refs_stage_output_alias? key)))
			false))))

(define scalar_first_probe_stage? (lambda (stage)
	(and (group_stage? stage)
		(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single))
			(and (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote first))
				(and (not (empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
					(and (empty_list? (qassoc_get (gs_facts stage) (quote btw2025_accessing_after_simple) '()))
						(and (not (reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (found key)
							(or found (expr_refs_stage_output_alias? key)))
							false))
							(and (or
								(source_is_base_table? (gs_input stage))
								(and (query_block? (gs_input stage))
									(or
										(expr_refs_stage_output_alias? (nth (car (gs_aggregates stage)) 0))
										(scalar_first_query_stage_output_key? stage))))
								(and (equal? (count (gs_aggregates stage)) 1)
									(and (equal? (nth (car (gs_aggregates stage)) 1) (scalar_once_reduce_first))
										(not (scalar_once_ordered_payload? (car (gs_aggregates stage)))))))))))))))

(define scalar_aggregate_probe_stage? (lambda (stage)
	(if (not (group_stage? stage))
		false
		(begin
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define keys (if (empty_list? lookup_keys) '() (gs_keys stage)))
			(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_aggregate))
				(and (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote many))
					(and (equal? (count keys) (count lookup_keys))
						(and (or
							(not (empty_list? lookup_keys))
							(and (empty_list? (gs_domain stage))
								(not (stage_has_residual_outer_refs? stage))))
							(and (equal? (coalesceNil (gs_having stage) true) true)
								(source_is_base_table? (gs_input stage)))))))))))

(define scalar_cardinality_probe_stage? (lambda (stage)
	(if (not (group_stage? stage))
		false
		(begin
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define ags (gs_aggregates stage))
			(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single))
				(and (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote single_or_error))
					(and (not (empty_list? lookup_keys))
						(and (equal? (count (gs_keys stage)) (count lookup_keys))
							(and (empty_list? (qassoc_get (gs_facts stage) (quote btw2025_accessing_after_simple) '()))
								(and (equal? (coalesceNil (gs_having stage) true) true)
									(and (source_is_base_table? (gs_input stage))
										(and (equal? (count ags) 2)
											(equal? (nth ags 1) aggregate_count_descriptor)))))))))))))

(define presence_probe_stage? (lambda (stage)
	(and (group_stage? stage)
		(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote exists))
			(and (not (empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
				(and (equal? (qassoc_get (gs_facts stage) (quote presence_only) false) true)
					(equal? (coalesceNil (gs_having stage) true) true)))))))

(define stage_has_residual_outer_refs? (lambda (stage)
	(not (empty_list? (qassoc_get (gs_facts stage) (quote btw2025_accessing_after_simple) '())))))

(define stage_keys_are_input_local? (lambda (stage)
	(begin
		(define alias (group_stage_input_alias stage))
		(reduce (gs_keys stage) (lambda (ok key)
			(and ok (expr_only_refs_alias? alias alias key)))
			true))))

(define scalar_or_presence_probe_stage? (lambda (stage)
	(or (scalar_first_probe_stage? stage) (presence_probe_stage? stage))))

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
		(define session_keys (query_expr_session_reads inner))
		(define explicit_keys (map (coalesceNil (qb_group inner) '()) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define keys (merge (list explicit_keys (filter session_keys (lambda (expr)
			(not (contains? explicit_keys expr)))))))
		(define stage_id (concat "scalar-group-top:" (fnv_hash (serialize (list subquery keys ags (qb_order inner))))))
		(define stage (make_group_stage
			stage_id
			inner_src
			session_keys
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
				(list (quote domain) session_keys)
				(list (quote lookup-keys) session_keys)
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote scalar))
				(list (quote cardinality_mode) (quote first)))))
		(list
			(list (quote grouped_scalar_top) stage)
			(list stage)
			'()))))

(define make_plain_exists_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(define pending_info (if (>= (count args) 3) (nth args 2) nil))
		(if (not (exists_inner_supported? inner))
			(neumann_fail "untangle_query" "EXISTS group-stage(D) currently supports one plain inner query-block")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_sources (qb_sources inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(presence_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_sources outer_sources (qb_sources inner)))
		(define lookup_pairs (domain_correlation_pairs (merge (list
			corr_pairs source_corr_pairs (presence_session_domain_pairs inner)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (presence_correlation_pair inner_default inner_sources outer_sources term)))))
		(define local_sources
			(sources_without_outer_join_terms inner_default inner_sources outer_sources inner_sources))
		(define residual_outer_refs (merge_unique (list
			(btw2025_terms_outer_column_refs local_terms inner_sources outer_sources)
			(btw2025_sources_outer_column_refs local_sources inner_sources outer_sources))))
		(define keys (if (and (empty_list? lookup_pairs) (empty_list? residual_outer_refs))
			'(1)
			(merge_unique (list
				(correlation_inner_keys inner_default lookup_pairs)
				residual_outer_refs))))
		(define outer_domain (merge_unique (list
			(correlation_domain lookup_pairs)
			residual_outer_refs)))
		(define lookup_keys (merge_unique (list
			(correlation_lookup_keys lookup_pairs)
			residual_outer_refs)))
		(define condition (combine_where_terms local_terms true))
		(define stage_input (if (and (single_source? (qb_sources inner)) (empty_list? (qb_stages inner)))
			inner_src
			(make_query_block
				(qb_schema inner)
				local_sources
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
					(list (quote preserve_empty_domain) (not (empty_list? outer_domain)))
					(list (quote null_semantics) (quote exists))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs residual_outer_refs pending_info)))))
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

(define make_grouped_exists_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(define pending_info (if (>= (count args) 3) (nth args 2) nil))
		(if (not (grouped_exists_inner_supported? inner))
			(neumann_fail "untangle_query" "grouped EXISTS requires an unordered, unlimited query-block")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_sources (qb_sources inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(presence_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_sources outer_sources (qb_sources inner)))
		(define lookup_pairs (domain_correlation_pairs (merge (list
			corr_pairs source_corr_pairs (presence_session_domain_pairs inner)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (presence_correlation_pair inner_default inner_sources outer_sources term)))))
		(define explicit_keys (map (qb_group inner) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define keys (group_keys_for_correlations inner_default lookup_pairs explicit_keys))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define ags (dedupe_aggregates_by_col (merge_unique
			(list (extract_aggregates (coalesceNil (qb_having inner) true))
				(list aggregate_count_descriptor)))))
		(define stage_input (make_query_block
			(qb_schema inner)
			(sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources inner))
			'()
			condition
			'() nil '() nil nil
			'()
			(qb_stages inner)
			(qb_facts inner)))
		(define stage_id (concat "exists-group:" (fnv_hash
			(string (list subquery keys outer_domain condition (qb_having inner))))))
		(define stage (make_group_stage
			stage_id
			stage_input
			outer_domain
			keys
			ags
			(qb_having inner)
			'()
			'()
			nil nil
			(merge (list
				(list
					(list (quote condition) true)
					(list (quote purpose) (quote exists))
					(list (quote presence_only) true)
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) (not (empty_list? outer_domain)))
					(list (quote null_semantics) (quote exists))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs '() pending_info)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define having_expr (replace_group_expr
			inner_default stage_alias keys key_names ags (coalesceNil (qb_having inner) true)))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(stage_source_outer? outer_sources)
			(make_stage_lookup_condition stage_alias key_names lookup_keys having_expr)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(list
			(list (quote >)
				(list (quote coalesceNil)
					(list (quote get_column) stage_alias false count_col false)
					0)
				0)
			(list stage)
			(list source)))))

(define make_exists_stage_rewrite (lambda (inner args)
	(if (empty_list? (qb_group inner))
		(make_plain_exists_stage_rewrite inner args)
		(make_grouped_exists_stage_rewrite inner args))))

(define combine_exists_union_results (lambda (results)
	(begin
		(define items (coalesceNil results '()))
		(if (empty_list? items)
			(list false '() '())
			(list
				(cons (quote or) (map items (lambda (item) (nth item 0))))
				(merge (map items (lambda (item) (nth item 1))))
				(merge (map items (lambda (item) (nth item 2)))))))))

(define exists_union_branch_stage (lambda (result)
	(match (nth result 1)
		'(stage) (if (and (group_stage? stage)
			(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote exists)))
			stage
			nil)
		_ nil)))

(define exists_union_stage_contract (lambda (stage)
	(if (nil? stage)
		nil
		(begin
			(define keys (gs_keys stage))
			(define input (gs_input stage))
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(if (or (not (equal? (count keys) 1))
				(or (not (equal? (count lookup_keys) 1))
					(or (not (if (query_block? input)
						(scalar_source_shape_supported? (qb_sources input))
						(source_is_base_table? input)))
						(or (stage_has_residual_outer_refs? stage)
							(not (stage_keys_are_input_local? stage))))))
				nil
				(list
					1
					(gs_domain stage)
					lookup_keys
					(qassoc_get (gs_facts stage) (quote null_semantics) nil)
					(qassoc_get (gs_facts stage) (quote cardinality_mode) nil)))))))

(define exists_union_results_compatible? (lambda (results)
	(match (coalesceNil results '())
		(cons first rest) (begin
			(define first_contract (exists_union_stage_contract (exists_union_branch_stage first)))
			(and (not (nil? first_contract))
				(reduce rest (lambda (compatible result)
					(and compatible
						(equal? first_contract
							(exists_union_stage_contract (exists_union_branch_stage result)))))
					true)))
		_ false)))

(define exists_union_candidate_fields (lambda (keys)
	(merge (mapIndex keys (lambda (i key)
		(list (concat "k" (string i)) key))))))

(define exists_union_candidate_branch (lambda (stage)
	(begin
		(define input (gs_input stage))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define fields (exists_union_candidate_fields (gs_keys stage)))
		(if (query_block? input)
			(make_query_block
				(qb_schema input)
				(qb_sources input)
				fields
				(combine_where (qb_where input) condition)
				'() nil '() nil nil '()
				(qb_stages input)
				(qb_facts input))
			(make_query_block
				(source_schema input)
				(list input)
				fields
				condition
				'() nil '() nil nil '() '() '())))))

(define make_combined_exists_union_stage_rewrite (lambda (inner results args)
	(begin
		(define first_stage (exists_union_branch_stage (car results)))
		(define outer_sources (nth args 0))
		(define lookup_keys (qassoc_get (gs_facts first_stage) (quote lookup-keys) '()))
		(define outer_domain (gs_domain first_stage))
		/* The UNION subtree can contain many already-decorrelated stages. Hash its
		serialization once and reuse the compact identity below; serializing that
		large immutable tree twice made canonicalization scale with an avoidable
		second full traversal. */
		(define inner_hash (fnv_hash (serialize inner)))
		(define candidate_alias (concat "__exists_union_" inner_hash))
		(define union_input (make_union_block
			(union_mode inner)
			(map results (lambda (result)
				(exists_union_candidate_branch (exists_union_branch_stage result))))
			'() nil nil
			(list
				(list (quote purpose) (quote exists_candidate_union))
				(list (quote alias) candidate_alias))))
		(define keys (mapIndex (gs_keys first_stage) (lambda (i _key)
			(list (quote get_column) candidate_alias false (concat "k" (string i)) false))))
		(define stage_id (concat "exists-union:"
			(fnv_hash (serialize (list inner_hash outer_domain lookup_keys)))))
		(define stage (make_group_stage
			stage_id
			union_input
			outer_domain
			keys
			(list aggregate_count_descriptor)
			nil '() '() nil nil
			(list
				(list (quote condition) true)
				(list (quote purpose) (quote exists))
				(list (quote presence_only) true)
				(list (quote max_needed_per_domain) 1)
				(list (quote domain) outer_domain)
				(list (quote lookup-keys) lookup_keys)
				(list (quote preserve_empty_domain) (not (empty_list? outer_domain)))
				(list (quote null_semantics) (quote exists))
				(list (quote cardinality_mode) (quote many)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(stage_source_outer? outer_sources)
			(make_exists_stage_join_condition stage_alias (group_key_cols keys) lookup_keys)))
		(list
			(list (quote >)
				(list (quote coalesceNil)
					(list (quote get_column) stage_alias false
						(aggregate_col_name aggregate_count_descriptor) false)
					0)
				0)
			(list stage)
			(list source)))))

(define make_exists_union_stage_rewrite (lambda (inner args)
	(if (not (union_block? inner))
		(neumann_fail "untangle_query" "EXISTS UNION lowering expects union-block")
		(if (or (not (empty_list? (union_order inner)))
			(or (not (nil? (union_limit inner))) (not (nil? (union_offset inner)))))
			(neumann_fail "untangle_query" "EXISTS over UNION currently supports plain unordered branches")
			(begin
				(define results (map (union_branches inner) (lambda (branch)
					(make_exists_stage_rewrite branch (list
						(nth args 0)
						branch
						(if (>= (count args) 3) (nth args 2) nil))))))
				(if (and (equal? (union_mode inner) (quote all))
					(exists_union_results_compatible? results))
					(make_combined_exists_union_stage_rewrite inner results args)
					(combine_exists_union_results results)))))))

/* Canonicalization is the only layer that should know IN was written over a
UNION. In positive WHERE truth context, each branch becomes one membership
alternative and the result is a flat n-ary OR. This gives naive lowering the
short-circuit behavior of OR(EXISTS ...) while exposing one application of the
general RecSet algebra documented in INVARIANTS.md:

T_R(p OR q) = T_R(p) union T_R(q).

The useful invariant is the general algebra and its proof obligations, not an
IN/UNION special-case identity. A later optimizer may associate, distribute,
factor, project, or fuse RecSet formulas and cost the equivalent plans without
recovering the original SQL spelling. This particular canonicalizer merely
makes the membership branches visible to that common machinery. NOT IN and
scalar/CASE consumers stay on the 3VL path and cannot use complement laws
without separately proving two-valued semantics. */
(define combine_in_union_results (lambda (results negate)
	(begin
		(define items (coalesceNil results '()))
		(if (empty_list? items)
			(list false '() '())
			(list
				(if (single_source? items)
					(nth (car items) 0)
					(cons (if negate (quote and) (quote or))
						(map items (lambda (item) (nth item 0)))))
				(merge_unique (map items (lambda (item) (nth item 1))))
				(merge_unique (map items (lambda (item) (nth item 2)))))))))

(define make_in_union_stage_rewrite (lambda (probe inner args)
	(if (not (union_block? inner))
		(neumann_fail "untangle_query" "IN UNION lowering expects union-block")
		(if (or (not (empty_list? (union_order inner)))
			(or (not (nil? (union_limit inner))) (not (nil? (union_offset inner)))))
			(neumann_fail "untangle_query" "IN over UNION currently supports plain unordered branches")
			(if (not (in_union_single_column? inner))
				(neumann_fail "untangle_query" "IN UNION subquery branches must expose exactly one column")
				(begin
					(define truth_mode (string? (if (>= (count args) 4) (nth args 3) false)))
					(define candidate_supported (in_union_candidate_supported? inner (nth args 0)))
					(define canonical_supported (in_union_truth_canonical_supported? inner (nth args 0)))
					(if (and (not (if (>= (count args) 3) (nth args 2) false))
						(and candidate_supported (or (not truth_mode) (not canonical_supported))))
						(make_in_union_candidate_stage_rewrite probe inner
							(if truth_mode
								(list (nth args 0) (nth args 1) false true
									(if (>= (count args) 5) (nth args 4) nil))
								args))
						(begin
							/* Unsupported complex branches keep their semantic barrier.
							Only simple membership relations become the OR cloud. */
							(define branch_args (if (and truth_mode (not canonical_supported))
								(list (nth args 0) (nth args 1) false true
									(if (>= (count args) 5) (nth args 4) nil))
								args))
							(combine_in_union_results
								(map (union_branches inner) (lambda (branch)
									(make_in_stage_rewrite probe branch branch_args)))
								(if (>= (count args) 3) (nth args 2) false))))))))))

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

(define in_union_truth_canonical_supported? (lambda (inner outer_sources)
	(and (in_union_candidate_supported? inner outer_sources)
		(reduce (union_branches inner) (lambda (ok branch)
			(and ok
				(and (single_source? (qb_sources branch))
					(and (source_is_base_table? (car (qb_sources branch)))
						(empty_list? (qb_stages branch))))))
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
		(define null_ag (in_rhs_state_descriptor candidate_key))
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
			(not semijoin_where)
			(if semijoin_where
				(make_positive_in_join_condition stage_alias key_names (list probe) probe aggregate_count_descriptor)
				(make_exists_stage_join_condition stage_alias key_names (list probe)))))
		(define null_source (list
			null_stage_alias
			(group_stage_schema null_stage)
			(make_stage_output_relation null_stage_id)
			true
			true))
		(if semijoin_where
			(list true (list stage) (list source))
			(list
				(in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
				(list stage null_stage)
				(list source null_source))))))

(define make_plain_in_stage_rewrite (lambda (probe inner args)
	(begin
		(define outer_sources (nth args 0))
		(define negate (if (>= (count args) 3) (nth args 2) false))
		(define where_mode (if (>= (count args) 4) (nth args 3) false))
		(define truth_membership (and (not negate) (string? where_mode)))
		(define semijoin_where (and (not negate) (and (not (string? where_mode)) where_mode)))
		(define pending_info (if (>= (count args) 5) (nth args 4) nil))
		(define membership_inner (normalize_membership_query_block inner))
		(if (not (membership_inner_supported? membership_inner))
			(neumann_fail "untangle_query" "IN group-stage(D) requires an ungrouped membership query-block")
			true)
		(if (not (equal? (count (qb_fields membership_inner)) 2))
			(neumann_fail "untangle_query" "IN subquery must expose exactly one column")
			true)
		(define inner_src (car (qb_sources membership_inner)))
		(define inner_sources (qb_sources membership_inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where membership_inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define lookup_pairs (domain_correlation_pairs (merge (list
			corr_pairs (session_domain_pairs membership_inner)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(define rhs_expr (canonical_column_expr_for_alias inner_default (query_block_first_expr membership_inner)))
		(define keys (cons rhs_expr
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define null_keys (if (empty_list? (correlation_inner_keys inner_default lookup_pairs))
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define domain_lookup_keys (correlation_lookup_keys lookup_pairs))
		(define lookup_keys (cons probe domain_lookup_keys))
		(define condition (combine_where_terms local_terms true))
		(define stage_input (if (and (equal? (count inner_sources) 1)
			(and (source_is_base_table? inner_src)
				(and (empty_list? (qb_stages membership_inner))
					(and (empty_list? (qb_order membership_inner))
						(nil? (qb_limit membership_inner))))))
			inner_src
			(make_query_block
				(qb_schema membership_inner)
				(sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources membership_inner))
				'()
				condition
				'() nil
				(qb_order membership_inner)
				(qb_limit membership_inner)
				(qb_offset membership_inner)
				'()
				(qb_stages membership_inner)
				(qb_facts membership_inner))))
		(define stage_condition (if (query_block? stage_input) true condition))
		(define stage_id (concat "in:" (fnv_hash (string (list probe keys lookup_keys condition)))))
		(define null_ag (in_rhs_state_descriptor rhs_expr))
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
				(btw2025_stage_facts membership_inner outer_sources lookup_pairs '() pending_info)))))
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
				(btw2025_stage_facts membership_inner outer_sources lookup_pairs '() pending_info)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define null_stage_alias (exists_stage_alias null_stage_id))
		(define key_names (group_key_cols keys))
		(define null_key_names (group_key_cols null_keys))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(or truth_membership (not semijoin_where))
			(if semijoin_where
				(make_positive_in_join_condition stage_alias key_names lookup_keys probe aggregate_count_descriptor)
				(make_exists_stage_join_condition stage_alias key_names lookup_keys))))
		(define null_source (list
			null_stage_alias
			(group_stage_schema null_stage)
			(make_stage_output_relation null_stage_id)
			true
			(make_exists_stage_join_condition null_stage_alias null_key_names domain_lookup_keys)))
		(define count_col (aggregate_col_name aggregate_count_descriptor))
		(if semijoin_where
			(list true (list stage) (list source))
			(if truth_membership
				(list
					(in_membership_truth_expr probe stage_alias aggregate_count_descriptor)
					(list stage)
					(list source))
				(list
					(if negate
						(not_in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
						(in_membership_expr probe stage_alias aggregate_count_descriptor null_stage_alias null_ag))
					(list stage null_stage)
					(list source null_source)))))))

(define grouped_membership_inner_supported? (lambda (inner outer_sources)
	(and (query_block? inner)
		(and (not (empty_list? (qb_sources inner)))
			(and (not (empty_list? (qb_group inner)))
				(and (empty_list? (qb_order inner))
					(and (nil? (qb_limit inner))
						(and (nil? (qb_offset inner))
							(empty_list? (btw2025_query_block_accessing_aliases inner outer_sources))))))))))

(define make_grouped_in_semijoin_rewrite (lambda (probe inner outer_sources)
	(begin
		(if (not (grouped_membership_inner_supported? inner outer_sources))
			(neumann_fail "untangle_query" "grouped IN currently requires an uncorrelated unordered query-block")
			true)
		(if (not (equal? (count (qb_fields inner)) 2))
			(neumann_fail "untangle_query" "IN subquery must expose exactly one column")
			true)
		(define stage (make_group_stage_for_query_block inner))
		(define input_alias (group_stage_input_alias stage))
		(define keys (gs_keys stage))
		(define ags (gs_aggregates stage))
		(define rhs_expr (canonical_column_expr_for_alias input_alias (query_block_first_expr inner)))
		(define key_idx (group_key_index input_alias keys rhs_expr))
		(if (nil? key_idx)
			(neumann_fail "untangle_query" "grouped IN currently requires the selected value to be a GROUP BY key")
			true)
		(define stage_alias (exists_stage_alias (gs_id stage)))
		(define key_names (group_key_cols keys))
		(define having_expr (replace_group_expr
			input_alias stage_alias keys key_names ags (coalesceNil (gs_having stage) true)))
		(define join_condition (combine_where_terms
			(list
				(make_exists_stage_join_condition stage_alias (list (nth key_names key_idx)) (list probe))
				(list (quote not) (list (quote nil?) probe))
				having_expr)
			true))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation (gs_id stage))
			false
			join_condition))
		(list true (list stage) (list source)))))

(define make_in_stage_rewrite (lambda (probe inner args)
	(begin
		(define outer_sources (nth args 0))
		(define negate (if (>= (count args) 3) (nth args 2) false))
		(define semijoin_where (and (not negate) (if (>= (count args) 4) (nth args 3) false)))
		(define membership_inner (normalize_membership_query_block inner))
		(if (empty_list? (qb_group membership_inner))
			(make_plain_in_stage_rewrite probe membership_inner args)
			(if semijoin_where
				(make_grouped_in_semijoin_rewrite probe membership_inner outer_sources)
				(neumann_fail "untangle_query" "grouped IN is currently supported only as a positive WHERE predicate"))))))

(define make_scalar_aggregate_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(define pending_info (if (>= (count args) 3) (nth args 2) nil))
		(if (not (scalar_aggregate_supported? inner))
			(neumann_fail "untangle_query" "scalar aggregate group-stage(D) currently supports one plain inner query-block")
			true)
		(define value_expr (query_block_first_expr inner))
		(define ags (dedupe_aggregates_by_col (merge (list (extract_aggregates value_expr) (list aggregate_count_descriptor)))))
		(if (empty_list? ags)
			(neumann_fail "untangle_query" "table-backed scalar subquery without aggregate needs cardinality_mode single_or_error lowering")
			true)
		(define inner_src (car (qb_sources inner)))
		(define inner_sources (qb_sources inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(define having_terms (split_and_terms (coalesceNil (qb_having inner) true)))
		(define having_corr_pairs (filter (map having_terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_having_terms (filter having_terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_sources outer_sources (qb_sources inner)))
		(define all_corr_pairs (domain_correlation_pairs (merge (list
			corr_pairs having_corr_pairs source_corr_pairs (session_domain_pairs inner)))))
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
				(sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources inner))
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
				(btw2025_stage_facts inner outer_sources all_corr_pairs '() pending_info)))))
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
		(define pending_info (if (>= (count args) 3) (nth args 2) nil))
		(if (not (scalar_once_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
			true)
		(define value_expr (query_block_first_expr inner))
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "scalar once_limit stage requires a base inner source after FROM flattening")
			true)
		(define inner_sources (qb_sources inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_sources outer_sources (qb_sources inner)))
		(define lookup_pairs (domain_correlation_pairs (merge (list
			corr_pairs source_corr_pairs (session_domain_pairs inner)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(scalar_stage_inner_keys_for_correlations inner_default (qb_stages inner) (qb_sources inner) lookup_pairs)))
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
				(sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources inner))
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
				(btw2025_stage_facts inner outer_sources lookup_pairs '() pending_info)))))
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
		(if (not (single_row_derived_supported? inner))
			(neumann_fail "untangle_query" "derived relation stage currently supports a non-grouped LIMIT 1 relation")
			true)
		(define inner_src (car (qb_sources inner)))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "derived relation stage requires a base inner source")
			true)
		(define inner_default (source_alias inner_src))
		(define keys '(1))
		(define ags (dedupe_aggregates_by_col
			(merge_unique (extract_assoc (qb_fields inner) (lambda (_title expr)
				(list (list
					(canonical_column_expr_for_alias inner_default expr)
					(scalar_once_reduce_first)
					nil)))))))
		(define stage_input (make_query_block
			(qb_schema inner)
			(qb_sources inner)
			'()
			(coalesceNil (qb_where inner) true)
			'() nil
			(qb_order inner)
			(qb_limit inner)
			(qb_offset inner)
			'()
			(qb_stages inner)
			(qb_facts inner)))
		(define stage_id (concat "derived-once:" (fnv_hash (string (list original_relation alias ags)))))
		(define stage (make_group_stage
			stage_id
			stage_input
			'()
			keys
			ags
			nil
			'()
			'()
			nil nil
			(list
				(list (quote condition) true)
				(list (quote purpose) (quote scalar_single))
				(list (quote domain) '())
				(list (quote lookup-keys) '())
				(list (quote preserve_empty_domain) true)
				(list (quote null_semantics) (quote scalar))
				(list (quote cardinality_mode) (quote first))
				(list (quote partition_by) '())
				(list (quote physical_max_rows) 1)
				(list (quote on_overflow) (quote ignore)))))
		(define projection (map_assoc (qb_fields inner) (lambda (_title expr)
			(begin
				(define value_expr (canonical_column_expr_for_alias inner_default expr))
				(define ag (list value_expr (scalar_once_reduce_first) nil))
				(scalar_once_value_expr alias ag)))))
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
		(define inner_where (requalify_single_source_expr inner_alias alias (qb_where inner)))
		(define filtercols (extract_columns_for_alias derived_src inner_where))
		(define rn_col (concat "__derived_row_number_" (fnv_hash (serialize (list
			alias sortcols sortdirs inner_where (qb_limit inner) (qb_offset inner))))))
		(define rn_stage (make_window_stage
			(concat "window-row-number:" rn_col)
			derived_src
			rn_col
			sortcols
			sortdirs
			0
			filtercols
			(list (quote lambda)
				(cons (quote $set) (map filtercols (lambda (col) (symbol (concat alias "." col)))))
				(list (quote list)
					(symbol "$set")
					(list (quote optimize) (lower_column_expr_for_alias derived_src inner_where))))
			(list (quote lambda) (list (quote acc) (quote mapped))
				(list (quote if) (list (quote cadr) (quote mapped))
					(list (quote begin)
						(list (list (quote car) (quote mapped)) (list (quote +) (quote acc) 1))
						(list (quote +) (quote acc) 1))
					(list (quote begin)
						(list (list (quote car) (quote mapped)) 0)
						(quote acc))))
			0
			(list (list (quote kind) (quote ordered-window)))))
		(define rn_expr (list (quote get_column) alias false rn_col false))
		(define offset_value (coalesceNil (qb_offset inner) 0))
		(define upper_bound (+ offset_value (qb_limit inner)))
		(define lower_filter (if (> offset_value 0) (list (quote >) rn_expr offset_value) true))
		(define limit_filter (combine_where lower_filter (list (quote <=) rn_expr upper_bound)))
		(define derived_filter (combine_where inner_where limit_filter))
		(list
			(cons (source_with_join_expr derived_src derived_filter) helper_sources)
			(requalify_single_source_fields inner_alias alias (qb_fields inner))
			true
			rn_stage))))

(define requalify_stage_fact_for_derived (lambda (old_alias new_alias fact)
	(match fact
		((symbol condition) expr)
		(list (quote condition) (requalify_single_source_expr old_alias new_alias expr))
		((quote condition) expr)
		(list (quote condition) (requalify_single_source_expr old_alias new_alias expr))
		((symbol domain) exprs)
		(list (quote domain) (map (coalesceNil exprs '()) (lambda (expr)
			(requalify_single_source_expr old_alias new_alias expr))))
		((quote domain) exprs)
		(list (quote domain) (map (coalesceNil exprs '()) (lambda (expr)
			(requalify_single_source_expr old_alias new_alias expr))))
		((symbol lookup-keys) exprs)
		(list (quote lookup-keys) (map (coalesceNil exprs '()) (lambda (expr)
			(requalify_single_source_expr old_alias new_alias expr))))
		((quote lookup-keys) exprs)
		(list (quote lookup-keys) (map (coalesceNil exprs '()) (lambda (expr)
			(requalify_single_source_expr old_alias new_alias expr))))
		_ fact)))

(define requalify_stage_for_derived (lambda (old_alias new_alias stage)
	(if (group_stage? stage)
		(make_group_stage
			(gs_id stage)
			(gs_input stage)
			(map (gs_domain stage) (lambda (expr)
				(requalify_single_source_expr old_alias new_alias expr)))
			(map (gs_keys stage) (lambda (expr)
				(requalify_single_source_expr old_alias new_alias expr)))
			(gs_aggregates stage)
			(requalify_single_source_expr old_alias new_alias (gs_having stage))
			(gs_output stage)
			(gs_order stage)
			(gs_limit stage)
			(gs_offset stage)
			(map (gs_facts stage) (lambda (fact)
				(requalify_stage_fact_for_derived old_alias new_alias fact))))
		stage)))

(define requalify_stages_for_derived (lambda (old_alias new_alias stages)
	(map (coalesceNil stages '()) (lambda (stage)
		(requalify_stage_for_derived old_alias new_alias stage)))))

(define derived_stage_rebind_id_map (lambda (alias stages)
	(map (coalesceNil stages '()) (lambda (stage)
		(list (gs_id stage) (concat (gs_id stage) ":derived:" (fnv_hash (string alias))))))))

(define derived_stage_rebind_alias_map (lambda (id_map stages)
	(begin
		(define base_alias_map (map (coalesceNil id_map '()) (lambda (entry)
			(list (exists_stage_alias (nth entry 0)) (exists_stage_alias (nth entry 1))))))
		(define aggregate_col_map (merge (map (coalesceNil stages '()) (lambda (stage)
			(begin
				(define old_alias (exists_stage_alias (gs_id stage)))
				(map (gs_aggregates stage) (lambda (ag)
					(list
						old_alias
						(aggregate_col_name ag)
						(aggregate_col_name
							(rewrite_stage_graph_expr base_alias_map id_map ag))))))))))
		(merge (list base_alias_map aggregate_col_map)))))

/* Derived clones rebind their root ID. Stage merges keep the surviving root ID,
while both modes rebind every reference and nested stage descriptor below it. */
(define rewrite_stage_graph_stage (lambda (alias_map id_map rebind_id stage)
	(if (not (group_stage? stage))
		stage
		(make_group_stage
			(if rebind_id (stage_merge_lookup id_map (gs_id stage) (gs_id stage)) (gs_id stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_input stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_domain stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_keys stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_aggregates stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_having stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_output stage))
			(rewrite_stage_graph_expr alias_map id_map (gs_order stage))
			(gs_limit stage)
			(gs_offset stage)
			(rewrite_stage_graph_expr alias_map id_map (gs_facts stage))))))

(define rewrite_stage_graph_stages (lambda (alias_map id_map stages)
	(map (coalesceNil stages '()) (lambda (stage)
		(rewrite_stage_graph_stage alias_map id_map true stage)))))

/* Independently flattened instances may start with identical generated stage
IDs. Give each instance its own IDs and source aliases before their plans meet. */
(define rebind_derived_stages (lambda (alias stages)
	(begin
		(define id_map (derived_stage_rebind_id_map alias stages))
		(define alias_map (derived_stage_rebind_alias_map id_map stages))
		(list
			(map (coalesceNil stages '()) (lambda (stage)
				(rewrite_stage_graph_stage alias_map id_map true stage)))
			alias_map
			id_map))))

(define rebind_derived_stage_expr (lambda (rebinding expr)
	(rewrite_stage_graph_expr (nth rebinding 1) (nth rebinding 2) expr)))

(define rebind_derived_stage_sources (lambda (rebinding sources)
	(map (coalesceNil sources '()) (lambda (src)
		(rewrite_stage_graph_source (nth rebinding 1) (nth rebinding 2) src)))))

(define make_scalar_single_stage_rewrite (lambda (inner args)
	(begin
		(define outer_sources (nth args 0))
		(define subquery (nth args 1))
		(define pending_info (if (>= (count args) 3) (nth args 2) nil))
		(if (not (scalar_single_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
			true)
		(define value_expr (query_block_first_expr inner))
		(define inner_src (car (qb_sources inner)))
		(define inner_sources (qb_sources inner))
		(define inner_default (source_alias inner_src))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define corr_pairs (filter (map terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_corr_pairs (source_join_correlation_pairs inner_default inner_sources outer_sources (qb_sources inner)))
		(define lookup_pairs (domain_correlation_pairs (merge (list
			corr_pairs source_corr_pairs (session_domain_pairs inner)))))
		(define local_terms (filter terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
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
				(sources_without_outer_join_terms inner_default inner_sources outer_sources (qb_sources inner))
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
				(btw2025_stage_facts inner outer_sources lookup_pairs '() pending_info)))))
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
		"COUNT" (if (empty_list? args)
			(list aggregate_count_descriptor)
			(list (list (sql_nonnull_count_expr (car args)) (quote +) 0)))
		"SUM" (match (car args)
			((symbol aggregate) agg_expr agg_reduce agg_neutral)
			(list (list agg_expr agg_reduce agg_neutral))
			((quote aggregate) agg_expr agg_reduce agg_neutral)
			(list (list agg_expr agg_reduce agg_neutral))
			_ (begin
				(define descriptor (sql_aggregates "SUM"))
				(list (list (car args) (car descriptor) (cadr descriptor)))))
		"AVG" (begin
			(define sum_descriptor (sql_aggregates "SUM"))
			(list
				(list (car args) (car sum_descriptor) (cadr sum_descriptor))
				(list (sql_nonnull_count_expr (car args)) (quote +) 0)))
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
		"COUNT" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"SUM" (list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
		"AVG" (list (quote sql_avg_divide)
			(list (quote get_column) stage_alias false (aggregate_col_name (car ags)) false)
			(list (quote get_column) stage_alias false (aggregate_col_name (cadr ags)) false))
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
		(define stages (unique_stages_by_id (merge (map rewritten_args (lambda (item) (nth item 1))))))
		(define sources (merge_unique (map rewritten_args (lambda (item) (nth item 2)))))
		(list expr stages sources))))

(define untangle_scalar_subquery_with_stages (lambda (subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define pending_info (uctx_get ctx (quote btw2025-current-info) nil))
		(define inner (untangle_query normalized sub_ctx))
		(if (query_block_no_from? inner)
			(list (untangle_zero_domain_subquery (quote inner_select) nil subquery ctx) '() '())
			(if (grouped_scalar_top_supported? inner outer_sources)
				(make_grouped_scalar_top_rewrite inner subquery)
				(if (empty_list? (extract_aggregates (query_block_first_expr inner)))
					(if (scalar_once_supported? inner)
						(make_scalar_once_stage_rewrite inner (list outer_sources subquery pending_info))
						(make_scalar_single_stage_rewrite inner (list outer_sources subquery pending_info)))
					(make_scalar_aggregate_stage_rewrite inner (list outer_sources subquery pending_info))))))))

(define untangle_exists_subquery_with_stages (lambda (subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define pending_info (uctx_get ctx (quote btw2025-current-info) nil))
		(define inner (untangle_query normalized sub_ctx))
		(if (union_block? inner)
			(make_exists_union_stage_rewrite inner (list outer_sources subquery pending_info))
			(if (query_block_no_from? inner)
				(list (untangle_zero_domain_subquery (quote inner_select_exists) nil subquery ctx) '() '())
				(make_exists_stage_rewrite inner (list outer_sources subquery pending_info)))))))

(define untangle_not_exists_subquery_with_stages (lambda (subquery outer_sources ctx)
	(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
		(list
			(list (quote not) (make_dependent_subquery_marker (quote exists) nil subquery outer_sources))
			'()
			'())
		(begin
			(define rewritten (untangle_exists_subquery_with_stages subquery outer_sources ctx))
			(list
				(list (quote not) (nth rewritten 0))
				(nth rewritten 1)
				(nth rewritten 2))))))

(define untangle_in_subquery_with_stages (lambda (probe subquery outer_sources ctx)
	(begin
		(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
		(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx false))))

(define resolve_in_subquery_with_stages (lambda (rewritten_probe subquery outer_sources ctx semijoin_where)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define pending_info (uctx_get ctx (quote btw2025-current-info) nil))
		(define inner (untangle_query normalized sub_ctx))
		(if (union_block? inner)
			(make_in_union_stage_rewrite rewritten_probe inner (list outer_sources subquery false semijoin_where pending_info))
			(if (query_block_no_from? inner)
				(list (untangle_zero_domain_subquery (quote inner_select_in) rewritten_probe subquery ctx) '() '())
				(make_in_stage_rewrite rewritten_probe inner (list outer_sources subquery false semijoin_where pending_info)))))))

(define resolve_not_in_subquery_with_stages (lambda (probe subquery outer_sources ctx)
	(begin
		(define normalized (normalize_query_ast subquery))
		(define sub_ctx (make_uctx ctx (list (list (quote outer-sources) outer_sources))))
		(define pending_info (uctx_get ctx (quote btw2025-current-info) nil))
		(define inner (untangle_query normalized sub_ctx))
		(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
		(if (union_block? inner)
			(make_in_union_stage_rewrite rewritten_probe inner (list outer_sources subquery true false pending_info))
			(if (query_block_no_from? inner)
				(list (untangle_zero_domain_not_in_subquery rewritten_probe subquery ctx) '() '())
				(make_in_stage_rewrite rewritten_probe inner (list outer_sources subquery true false pending_info)))))))

(define untangle_not_in_subquery_with_stages (lambda (probe subquery outer_sources ctx)
	(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
		(list (make_dependent_subquery_marker (quote not-in) probe subquery outer_sources) '() '())
		(resolve_not_in_subquery_with_stages probe subquery outer_sources ctx))))

(define direct_positive_in_term_rewrite_using (lambda (expr outer_sources ctx mode)
	(match expr
		((symbol inner_select_in) probe subquery)
		(if (uctx_get ctx (quote defer-subquery-rewrites) false)
			(list (make_dependent_subquery_marker
				(if (string? mode) (quote in-where-truth) (quote in-where))
				probe subquery outer_sources) '() '())
			(begin
				(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
				(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx mode)))
		((quote inner_select_in) probe subquery)
		(if (uctx_get ctx (quote defer-subquery-rewrites) false)
			(list (make_dependent_subquery_marker
				(if (string? mode) (quote in-where-truth) (quote in-where))
				probe subquery outer_sources) '() '())
			(begin
				(define rewritten_probe (nth (untangle_expr_with_stages probe outer_sources ctx) 0))
				(resolve_in_subquery_with_stages rewritten_probe subquery outer_sources ctx mode)))
		_ nil)))

(define direct_positive_in_term_rewrite (lambda (expr outer_sources ctx)
	(direct_positive_in_term_rewrite_using expr outer_sources ctx true)))

(define where_conjunct_with_stages (lambda (expr outer_sources ctx)
	(begin
		(define direct_in (direct_positive_in_term_rewrite expr outer_sources ctx))
		(if (nil? direct_in)
			(if (and (list? expr)
				(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or"))))
				(positive_where_expr_with_stages expr outer_sources ctx)
				(untangle_expr_with_stages expr outer_sources ctx))
			direct_in))))

/* AND and OR preserve positive WHERE truth context: SQL UNKNOWN rejects a row
just like FALSE. Descending only through this boolean cloud ensures NOT, CASE,
projection, and arbitrary scalar functions retain full IN/NOT IN null
semantics. New syntactic membership shapes should canonicalize here and reuse
membership_truth rather than adding another physical lowering path. */
(define positive_where_items_with_stages (lambda (items outer_sources ctx)
	(match items
		(cons item rest) (begin
			(define rewritten (positive_where_expr_with_stages item outer_sources ctx))
			(define tail (positive_where_items_with_stages rest outer_sources ctx))
			(list
				(cons (nth rewritten 0) (nth tail 0))
				(merge_unique (list (nth rewritten 1) (nth tail 1)))
				(merge_unique (list (nth rewritten 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define positive_where_expr_with_stages (lambda (expr outer_sources ctx)
	(begin
		(define direct_in (direct_positive_in_term_rewrite_using
			expr outer_sources ctx "truth"))
		(if (not (nil? direct_in))
			direct_in
			(if (and (list? expr)
				(or
					(or (equal? (car expr) (quote and)) (equal? (car expr) (symbol "and")))
					(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or")))))
				(begin
					(define rewritten (positive_where_items_with_stages (cdr expr) outer_sources ctx))
					(list
						(cons (car expr) (nth rewritten 0))
						(nth rewritten 1)
						(nth rewritten 2)))
				(untangle_expr_with_stages expr outer_sources ctx))))))

(define untangle_where_with_stages (lambda (expr outer_sources ctx)
	(begin
		(define condition (coalesceNil expr true))
		(define terms (split_and_terms condition))
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

(define btw2025_defer_subquery_rewrite? (lambda (_subquery _outer_sources ctx)
	(uctx_get ctx (quote defer-subquery-rewrites) false)))

(define btw2025_pending_unnesting_info (lambda (subquery outer_sources handle parent ancestors)
	(begin
		(define normalized (if (or (query_block? subquery) (union_block? subquery))
			subquery
			(normalize_query_ast subquery)))
		(define accessing (btw2025_normalized_subquery_accessing_aliases normalized outer_sources))
		(make_btw2025_unnesting_info handle parent ancestors accessing '() accessing accessing '() '()))))

(define untangle_if_expr_with_stages (lambda (head args outer_sources ctx)
	(match args
		(cons condition (cons branch rest))
		(if (equal? condition true)
			(untangle_expr_with_stages branch outer_sources ctx)
			(if (or (equal? condition false) (nil? condition))
				(match rest
					(cons fallback '()) (untangle_expr_with_stages fallback outer_sources ctx)
					_ (untangle_if_expr_with_stages head rest outer_sources ctx))
				(combine_stage_rewrite_results head
					(map args (lambda (item) (untangle_expr_with_stages item outer_sources ctx))))))
		_ (combine_stage_rewrite_results head
			(map args (lambda (item) (untangle_expr_with_stages item outer_sources ctx)))))))

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
		(cons head tail) (if (equal? head (quote if))
			(untangle_if_expr_with_stages head tail outer_sources ctx)
			(combine_stage_rewrite_results head (map tail (lambda (item) (untangle_expr_with_stages item outer_sources ctx)))))
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
		(define parent (uctx_get ctx (quote btw2025-current-handle) nil))
		(define parent_ancestors (uctx_get ctx (quote btw2025-ancestor-handles) '()))
		(define current_ancestors (if (nil? parent) '() (cons parent parent_ancestors)))
		(define current_handle (concat "djoin:" (fnv_hash (string (list kind subquery outer_sources)))))
		(define current_info (btw2025_pending_unnesting_info subquery outer_sources current_handle parent current_ancestors))
		(define resolve_ctx (make_uctx ctx (list
			(list (quote defer-subquery-rewrites) true)
			(list (quote btw2025-current-handle) current_handle)
			(list (quote btw2025-ancestor-handles) current_ancestors)
			(list (quote btw2025-current-info) current_info))))
		(match kind
			(symbol scalar)
			(untangle_scalar_subquery_with_stages subquery outer_sources resolve_ctx)
			(symbol exists)
			(untangle_exists_subquery_with_stages subquery outer_sources resolve_ctx)
			(symbol in)
			(begin
				(define probe_result (btw2025_decorrelate_expr_with_stages probe ctx))
				(define in_result (resolve_in_subquery_with_stages (nth probe_result 0) subquery outer_sources resolve_ctx false))
				(list
					(nth in_result 0)
					(unique_stages_by_id (merge (list (nth probe_result 1) (nth in_result 1))))
					(merge_unique (list (nth probe_result 2) (nth in_result 2)))))
			(symbol in-where)
			(begin
				(define probe_result (btw2025_decorrelate_expr_with_stages probe ctx))
				(define in_result (resolve_in_subquery_with_stages (nth probe_result 0) subquery outer_sources resolve_ctx true))
				(list
					(nth in_result 0)
					(unique_stages_by_id (merge (list (nth probe_result 1) (nth in_result 1))))
					(merge_unique (list (nth probe_result 2) (nth in_result 2)))))
			(symbol in-where-truth)
			(begin
				(define probe_result (btw2025_decorrelate_expr_with_stages probe ctx))
				(define in_result (resolve_in_subquery_with_stages (nth probe_result 0) subquery outer_sources resolve_ctx "truth"))
				(list
					(nth in_result 0)
					(unique_stages_by_id (merge (list (nth probe_result 1) (nth in_result 1))))
					(merge_unique (list (nth probe_result 2) (nth in_result 2)))))
			(symbol not-in)
			(begin
				(define probe_result (btw2025_decorrelate_expr_with_stages probe ctx))
				(define in_result (resolve_not_in_subquery_with_stages (nth probe_result 0) subquery outer_sources resolve_ctx))
				(list
					(nth in_result 0)
					(unique_stages_by_id (merge (list (nth probe_result 1) (nth in_result 1))))
					(merge_unique (list (nth probe_result 2) (nth in_result 2)))))
			_ (neumann_fail "untangle_query" "unknown dependent subquery marker")))))

(define btw2025_dependent_marker_key (lambda (expr)
	(if (dependent_subquery_marker? expr)
		(concat "dependent:" (fnv_hash (string expr)))
		nil)))

(define btw2025_decorrelate_exprs_using (lambda (exprs ctx resolved)
	(match exprs
		(cons expr rest) (begin
			(define current (btw2025_decorrelate_expr_using expr ctx resolved))
			(define tail (btw2025_decorrelate_exprs_using rest ctx (nth current 1)))
			(list (cons (nth current 0) (nth tail 0)) (nth tail 1)))
		_ (list '() resolved))))

(define btw2025_decorrelate_expr_using (lambda (expr ctx resolved)
	(begin
		(define key (btw2025_dependent_marker_key expr))
		(if (and (not (nil? key)) (has_assoc? resolved key))
			(list (resolved key) resolved)
			(match expr
				((symbol dependent-subquery) _kind _probe _subquery _outer_sources) (begin
					(define rewritten (btw2025_resolve_dependent_subquery expr ctx))
					(list rewritten (set_assoc resolved key rewritten)))
				((quote dependent-subquery) _kind _probe _subquery _outer_sources) (begin
					(define rewritten (btw2025_resolve_dependent_subquery expr ctx))
					(list rewritten (set_assoc resolved key rewritten)))
				(cons head tail) (begin
					(define rewritten_tail (btw2025_decorrelate_exprs_using tail ctx resolved))
					(list
						(combine_stage_rewrite_results head (nth rewritten_tail 0))
						(nth rewritten_tail 1)))
				_ (list (list expr '() '()) resolved))))))

(define btw2025_decorrelate_expr_with_stages (lambda (expr ctx)
	(nth (btw2025_decorrelate_expr_using expr ctx '()) 0)))

(define btw2025_decorrelate_field_expr_using (lambda (expr ctx resolved)
	(btw2025_decorrelate_expr_using expr ctx resolved)))

(define btw2025_decorrelate_fields_with_stages_using (lambda (fields ctx resolved)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (begin
			(define current (btw2025_decorrelate_field_expr_using expr ctx resolved))
			(define rewritten (nth current 0))
			(define tail (btw2025_decorrelate_fields_with_stages_using rest ctx (nth current 1)))
			(list
				(cons title (cons (nth rewritten 0) (nth tail 0)))
				(unique_stages_by_id (merge (list (nth rewritten 1) (nth tail 1))))
				(merge_unique (list (nth rewritten 2) (nth tail 2)))
				(nth tail 3)))
		_ (list '() '() '() resolved))))

(define btw2025_decorrelate_fields_with_stages (lambda (fields ctx)
	(btw2025_decorrelate_fields_with_stages_using fields ctx '())))

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
				(unique_stages_by_id (merge (list (nth rewritten_item 1) (nth tail 1))))
				(merge_unique (list (nth rewritten_item 2) (nth tail 2)))))
		_ (list '() '() '()))))

(define btw2025_decorrelate_expr_list_with_stages (lambda (exprs ctx)
	(match (coalesceNil exprs '())
		(cons expr rest) (begin
			(define rewritten (btw2025_decorrelate_expr_with_stages expr ctx))
			(define tail (btw2025_decorrelate_expr_list_with_stages rest ctx))
			(list
				(cons (nth rewritten 0) (nth tail 0))
				(unique_stages_by_id (merge (list (nth rewritten 1) (nth tail 1))))
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
				(unique_stages_by_id (merge (list (nth rewritten_src 1) (nth tail 1))))
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
			(unique_stages_by_id (merge (list (qb_stages block) source_stages (nth where_result 1) (nth field_result 1) (nth group_result 1) (nth having_result 1) (nth order_result 1) (nth hidden_result 1))))
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
		((symbol get_column) tblvar _tbl_ignorecase col col_ignorecase) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col col_ignorecase) expr)
			expr)
		((quote get_column) tblvar _tbl_ignorecase col col_ignorecase) (if (or (equal? tblvar alias) (nil? tblvar))
			(coalesceNil (field_expr_by_title projection col col_ignorecase) expr)
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

(define rewrite_order_output_alias (lambda (fields expr)
	(match expr
		((symbol get_column) nil _tbl_ignorecase col col_ignorecase)
		(coalesceNil (field_expr_by_title fields col col_ignorecase) expr)
		((quote get_column) nil _tbl_ignorecase col col_ignorecase)
		(coalesceNil (field_expr_by_title fields col col_ignorecase) expr)
		_ expr)))

(define rewrite_order_output_aliases (lambda (fields order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (list (rewrite_order_output_alias fields expr) dir)
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

(define literal_true? (lambda (expr)
	(match expr
		true true
		_ false)))

(define build_and_terms (lambda (terms)
	(begin
		(define filtered (filter (coalesceNil terms '()) (lambda (term) (not (literal_true? (coalesceNil term true))))))
		(if (empty_list? filtered)
			true
			(if (single_source? filtered)
				(car filtered)
				(cons (quote and) filtered))))))

(define combine_where (lambda (a b)
	(combine_where_terms (list a b) true)))

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
										(define requalified_inner_stages (requalify_stages_for_derived (source_alias (car (qb_sources inner))) alias (qb_stages inner)))
										(list
											(merge (list (nth staged 0) tail_sources))
											(cons (list alias (nth staged 1)) tail_rewrites)
											(cons (nth staged 2) tail_wheres)
											(cons (nth staged 3) (merge (list requalified_inner_stages tail_stages)))))
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
											(define requalified_inner_stages (requalify_stages_for_derived inner_alias alias (qb_stages inner)))
											(define stage_rebinding (rebind_derived_stages alias requalified_inner_stages))
											(define projection (rebind_derived_stage_expr stage_rebinding
												(requalify_single_source_fields inner_alias alias (qb_fields inner))))
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
												(merge (list (nth stage_rebinding 0) tail_stages))))
										(if (or (source_outer? src) (not (nil? (source_join_expr src))))
											(if (scalar_source_shape_supported? inner_sources)
												(begin
													(define only_inner (car inner_sources))
													(define inner_alias (source_alias only_inner))
													(define requalified_inner_stages (requalify_stages_for_derived inner_alias alias (qb_stages inner)))
													(define stage_rebinding (rebind_derived_stages alias requalified_inner_stages))
													(define projection (rebind_derived_stage_expr stage_rebinding
														(requalify_single_source_fields inner_alias alias (qb_fields inner))))
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
													(define derived_stage_sources (rebind_derived_stage_sources stage_rebinding
														(requalify_source_join_exprs inner_alias alias (cdr inner_sources))))
													(list
														(merge (list
															(list derived_base_source)
															derived_stage_sources
															(rewrite_sources_join_for_derived alias effective_projection tail_sources)))
														(cons (list alias effective_projection) tail_rewrites)
														tail_wheres
														(merge (list (nth stage_rebinding 0) tail_stages))))
												(neumann_fail "untangle_query" "multi-source derived JOIN needs relation-unit lowering"))
											(list
												(merge (list inner_sources (rewrite_sources_join_for_derived alias (qb_fields inner) tail_sources)))
												(cons (list alias (qb_fields inner)) tail_rewrites)
												(cons (qb_where inner) tail_wheres)
												(merge (list (qb_stages inner) tail_stages)))))))))
))))))

(define combine_where_terms (lambda (terms seed)
	(build_and_terms (merge (list
		(split_and_terms (coalesceNil seed true))
		(merge (map (coalesceNil terms '()) (lambda (term) (split_and_terms (coalesceNil term true))))))))))

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
				(list (quote operator-model) (quote combined))
				(list (quote defer-subquery-rewrites) true))))
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
								(define local_resolution_sources (merge_unique (list untangled_sources source_join_stage_sources)))
								(define qualify_join_expr (lambda (expr)
									(if (single_source? local_resolution_sources)
										expr
										(qualify_unqualified_column_for_sources local_resolution_sources expr))))
								(define rewritten_where (qualify_unqualified_column_for_sources
									local_resolution_sources
									(combine_where_terms source_where_terms (rewrite_derived_ref_chain rewrites (qb_where block)))))
								(if (expr_contains_window? rewritten_where)
									(neumann_fail "untangle_query" "window function is not allowed in WHERE")
									true)
								(define where_result (untangle_where_with_stages rewritten_where joined_expr_outer_sources joined_expr_ctx))
								(define rewritten_fields (rewrite_derived_fields_chain rewrites (qb_fields block)))
								(define field_result (untangle_fields_with_stages
									(qualify_join_expr rewritten_fields)
									joined_expr_outer_sources joined_expr_ctx))
								(define having_result (untangle_expr_with_stages
									(qualify_join_expr (rewrite_derived_ref_chain rewrites (qb_having block)))
									joined_expr_outer_sources joined_expr_ctx))
								(define stage_sources (merge_unique (list (nth where_result 2) (nth field_result 2) (nth having_result 2))))
								(define group_result (untangle_expr_list_with_stages
									(qualify_unqualified_column_for_sources local_resolution_sources
										(map (coalesceNil (qb_group block) '()) (lambda (item) (rewrite_derived_ref_chain rewrites item))))
									joined_expr_outer_sources
									joined_expr_ctx))
								(define order_result (untangle_order_with_stages
									(qualify_join_expr (rewrite_order_output_aliases
										rewritten_fields
										(rewrite_derived_order_chain rewrites (qb_order block))))
									joined_expr_outer_sources
									joined_expr_ctx))
								(define hidden_result (untangle_fields_with_stages
									(qualify_join_expr (rewrite_derived_fields_chain rewrites (qb_hidden block)))
									joined_expr_outer_sources joined_expr_ctx))
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

/* Join reordering can consume a logical hypergraph view of the query block. The
extractor deliberately does not rewrite sources or move predicates: WHERE and
ON terms remain owned by the query-block until physical lowering. */
(define join_hypergraph_expr_aliases (lambda (default_alias aliases expr)
	(filter (coalesceNil aliases '()) (lambda (alias)
		(expr_refs_alias? default_alias alias expr)))))

(define make_join_hypergraph_predicate (lambda (aliases origin owner predicate)
	(list
		(list (quote aliases) (coalesceNil aliases '()))
		(list (quote origin) origin)
		(list (quote owner) owner)
		(list (quote predicate) predicate))))

(define join_hypergraph_predicate_kind (lambda (entry)
	(match (count (qassoc_get entry (quote aliases) '()))
		0 (quote residual)
		1 (quote local)
		2 (quote edge)
		_ (quote hyperedge))))

(define join_hypergraph_where_predicates (lambda (block default_alias aliases)
	(if (equal? (coalesceNil (qb_where block) true) true)
		'()
		(map (split_and_terms (qb_where block)) (lambda (predicate)
			(make_join_hypergraph_predicate
				(join_hypergraph_expr_aliases default_alias aliases predicate)
				(quote where)
				nil
				predicate))))))

(define join_hypergraph_source_predicates (lambda (sources default_alias aliases)
	(merge (map (coalesceNil sources '()) (lambda (src)
		(begin
			(define join_expr (coalesceNil (source_join_expr src) true))
			(if (equal? join_expr true)
				'()
				(map (split_and_terms join_expr) (lambda (predicate)
					(make_join_hypergraph_predicate
						(merge_unique (list
							(join_hypergraph_expr_aliases default_alias aliases predicate)
							(list (source_alias src))))
						(if (source_outer? src) (quote outer-on) (quote inner-on))
						(source_alias src)
						predicate))))))))))

(define join_hypergraph_node_kind (lambda (src)
	(if (stage_output_relation? (source_relation src))
		(quote stage-output)
		(if (query_block? (source_relation src))
			(quote derived)
			(quote base)))))

(define join_hypergraph_nodes (lambda (sources)
	(mapIndex (coalesceNil sources '()) (lambda (position src)
		(list
			(list (quote alias) (source_alias src))
			(list (quote position) position)
			(list (quote kind) (join_hypergraph_node_kind src))
			(list (quote outer_join) (source_outer? src)))))))

(define join_hypergraph_outer_barriers_acc (lambda (sources preceding_aliases default_alias aliases)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define alias (source_alias src))
			(define next_preceding (append preceding_aliases alias))
			(define remaining (join_hypergraph_outer_barriers_acc
				rest next_preceding default_alias aliases))
			(if (source_outer? src)
				(cons
					(list
						(list (quote kind) (quote left-outer))
						(list (quote owner) alias)
						(list (quote preserved) preceding_aliases)
						(list (quote references)
							(join_hypergraph_expr_aliases default_alias aliases
								(coalesceNil (source_join_expr src) true)))
						(list (quote predicate) (source_join_expr src)))
					remaining)
				remaining))
		_ '())))

(define join_hypergraph_predicates_of_kind (lambda (predicates kind)
	(filter predicates (lambda (entry)
		(equal? (join_hypergraph_predicate_kind entry) kind)))))

(define extract_join_hypergraph (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define aliases (source_aliases sources))
		(define default_alias (if (empty_list? aliases) nil (car aliases)))
		(define predicates (merge (list
			(join_hypergraph_where_predicates block default_alias aliases)
			(join_hypergraph_source_predicates sources default_alias aliases))))
		(list
			(list (quote nodes) (join_hypergraph_nodes sources))
			(list (quote locals) (join_hypergraph_predicates_of_kind predicates (quote local)))
			(list (quote edges) (join_hypergraph_predicates_of_kind predicates (quote edge)))
			(list (quote hyperedges) (join_hypergraph_predicates_of_kind predicates (quote hyperedge)))
			(list (quote residuals) (join_hypergraph_predicates_of_kind predicates (quote residual)))
			(list (quote barriers)
				(join_hypergraph_outer_barriers_acc sources '() default_alias aliases))))))

(define join_optimizer_inner_source? (lambda (src)
	(and
		(source_is_base_table? src)
		(not (source_outer? src)))))

(define join_optimizer_normalize_inner_joins (lambda (block)
	(begin
		(define inner_join_terms (merge (map (qb_sources block) (lambda (src)
			(if (and
				(join_optimizer_inner_source? src)
				(not (or (nil? (source_join_expr src)) (equal? (source_join_expr src) true))))
				(split_and_terms (source_join_expr src))
				'())))))
		(make_query_block
			(qb_schema block)
			(map (qb_sources block) (lambda (src)
				(if (join_optimizer_inner_source? src) (source_with_join_expr src nil) src)))
			(qb_fields block)
			(combine_where_terms (merge (list
				(split_and_terms (coalesceNil (qb_where block) true))
				inner_join_terms)) true)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(qb_hidden block)
			(qb_stages block)
			(qb_facts block)))))

(define join_optimizer_predicates (lambda (graph)
	(merge (list
		(qassoc_get graph (quote edges) '())
		(qassoc_get graph (quote hyperedges) '())))))

(define join_optimizer_costed_predicates (lambda (graph)
	(merge (list
		(qassoc_get graph (quote locals) '())
		(join_optimizer_predicates graph)))))

(define join_hypergraph_all_predicates (lambda (graph)
	(merge (list
		(qassoc_get graph (quote locals) '())
		(qassoc_get graph (quote edges) '())
		(qassoc_get graph (quote hyperedges) '())
		(qassoc_get graph (quote residuals) '())))))

(define join_hypergraph_predicate_with_provenance (lambda (entry provenance_predicates)
	(begin
		(define predicate (qassoc_get entry (quote predicate) true))
		(define aliases (qassoc_get entry (quote aliases) '()))
		(define provenance (find provenance_predicates (lambda (candidate)
			(and (equal? (qassoc_get candidate (quote predicate) true) predicate)
				(equal? (qassoc_get candidate (quote aliases) '()) aliases))) nil))
		(if (nil? provenance)
			entry
			(qassoc_set
				(qassoc_set entry (quote origin) (qassoc_get provenance (quote origin) nil))
				(quote owner) (qassoc_get provenance (quote owner) nil))))))

/* Reordering consumes the normalized predicate cloud, while predicate origin
and barrier ownership come from the pre-normalization query block. */
(define join_hypergraph_with_provenance (lambda (graph provenance_graph)
	(begin
		(define provenance_predicates (join_hypergraph_all_predicates provenance_graph))
		(reduce '(locals edges hyperedges residuals)
			(lambda (enriched key)
				(qassoc_set enriched key
					(map (qassoc_get graph key '()) (lambda (entry)
						(join_hypergraph_predicate_with_provenance entry provenance_predicates)))))
			graph))))

(define join_optimizer_local_predicates (lambda (graph alias)
	(filter (qassoc_get graph (quote locals) '()) (lambda (entry)
		(equal? (qassoc_get entry (quote aliases) '()) (list alias))))))

(define join_optimizer_column_ref (lambda (sources default_alias expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase)
		(begin
			(define src (source_for_alias sources default_alias tblvar tbl_ignorecase))
			(if (nil? src) nil (list src (resolve_physical_column_name src col col_ignorecase))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(join_optimizer_column_ref sources default_alias
			(list (symbol "get_column") tblvar tbl_ignorecase col col_ignorecase))
		_ nil)))

(define join_optimizer_column_selectivity (lambda (column_ref)
	(if (nil? column_ref)
		0.1
		(begin
			(define distinct (planner_column_distinct_estimate (car column_ref) (cadr column_ref)))
			(if (or (nil? distinct) (<= distinct 0)) 0.1 (/ 1 (max 1 distinct)))))))

(define join_optimizer_equality_selectivity (lambda (sources default_alias left right)
	(begin
		(define left_ref (join_optimizer_column_ref sources default_alias left))
		(define right_ref (join_optimizer_column_ref sources default_alias right))
		(if (and (not (nil? left_ref)) (not (nil? right_ref)))
			(begin
				(define left_distinct (planner_column_distinct_estimate (car left_ref) (cadr left_ref)))
				(define right_distinct (planner_column_distinct_estimate (car right_ref) (cadr right_ref)))
				(if (or (nil? left_distinct) (nil? right_distinct))
					0.1
					(/ 1 (max 1 (max left_distinct right_distinct)))))
			(if (not (nil? left_ref))
				(join_optimizer_column_selectivity left_ref)
				(if (not (nil? right_ref)) (join_optimizer_column_selectivity right_ref) 0.1))))))

(define join_optimizer_expr_selectivity (lambda (sources default_alias expr)
	(match expr
		((symbol equal?) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((quote equal?) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((symbol equal??) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((quote equal??) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((symbol =) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((quote =) left right) (join_optimizer_equality_selectivity sources default_alias left right)
		((symbol <) _left _right) 0.3333333333333333
		((quote <) _left _right) 0.3333333333333333
		((symbol <=) _left _right) 0.3333333333333333
		((quote <=) _left _right) 0.3333333333333333
		((symbol >) _left _right) 0.3333333333333333
		((quote >) _left _right) 0.3333333333333333
		((symbol >=) _left _right) 0.3333333333333333
		((quote >=) _left _right) 0.3333333333333333
		_ 0.1)))

(define join_optimizer_product (lambda (values)
	(reduce (coalesceNil values '()) (lambda (product value) (* product value)) 1)))

(define join_optimizer_guaranteed_singleton_stage_source? (lambda (stages src)
	(if (not (stage_output_relation? (source_relation src)))
		false
		(begin
			(define stage (stage_for_output_relation stages (source_relation src)))
			(define purpose (if (group_stage? stage)
				(qassoc_get (gs_facts stage) (quote purpose) nil) nil))
			(and (group_stage? stage)
				(and (or (equal? purpose (quote scalar_single)) (equal? purpose (quote exists)))
					(and (empty_list? (gs_domain stage))
						(not (stage_has_residual_outer_refs? stage)))))))))

(define planner_source_row_count_using_stages (lambda (stages src)
	(if (join_optimizer_guaranteed_singleton_stage_source? stages src)
		1
		(planner_source_row_count src))))

(define join_optimizer_source_rows (lambda (stages sources default_alias graph src)
	(begin
		(define base_rows (coalesceNil (planner_source_row_count_using_stages stages src) 1000000))
		(define local_selectivity (join_optimizer_product
			(map (join_optimizer_local_predicates graph (source_alias src)) (lambda (entry)
				(join_optimizer_expr_selectivity sources default_alias
					(qassoc_get entry (quote predicate) true))))))
		(max 1 (* base_rows local_selectivity)))))

(define join_optimizer_alias_subset? (lambda (required available)
	(reduce (coalesceNil required '()) (lambda (ok alias)
		(and ok (contains? available alias))) true)))

(define join_optimizer_metadata_nodes (lambda (stages sources default_alias graph)
	(map sources (lambda (src)
		(list (source_alias src) (join_optimizer_source_rows stages sources default_alias graph src))))))

(define join_optimizer_metadata_predicates_from (lambda (sources default_alias entries aliases)
	(map (filter entries (lambda (entry)
		(join_optimizer_alias_subset? (qassoc_get entry (quote aliases) '()) aliases)))
		(lambda (entry)
			(begin
				(define predicate_aliases (qassoc_get entry (quote aliases) '()))
				(define origin (qassoc_get entry (quote origin) nil))
				(define local_source (if (single_source? predicate_aliases)
					(join_optimizer_source_by_alias sources (car predicate_aliases)) nil))
				(define barrier_owner (if (and (equal? origin (quote where))
					(and (not (nil? local_source)) (source_outer? local_source)))
					(source_alias local_source) nil))
				(define metadata (list
					predicate_aliases
					(join_optimizer_expr_selectivity sources default_alias
						(qassoc_get entry (quote predicate) true))
					origin
					(qassoc_get entry (quote owner) nil)
					(qassoc_get entry (quote predicate) true)))
				(if (nil? barrier_owner) metadata (append metadata barrier_owner)))))))

(define join_optimizer_metadata_predicates (lambda (sources default_alias graph aliases)
	(join_optimizer_metadata_predicates_from
		sources default_alias (join_optimizer_predicates graph) aliases)))

(define join_optimizer_metadata_costed_predicates (lambda (sources default_alias graph aliases)
	(join_optimizer_metadata_predicates_from
		sources default_alias (join_optimizer_costed_predicates graph) aliases)))

(define join_optimizer_source_by_alias (lambda (sources alias)
	(reduce sources (lambda (found src)
		(if (not (nil? found)) found
			(if (equal? (source_alias src) alias) src nil))) nil)))

(define join_optimizer_sources_for_order (lambda (sources aliases)
	(map aliases (lambda (alias)
		(begin
			(define src (join_optimizer_source_by_alias sources alias))
			(if (nil? src)
				(neumann_fail "join_reorder" (concat "join plan references unknown alias " alias))
				src))))))

/* Neumann/Radke join ordering lives in the SQL planner. None of these values
are runtime relations: they are compile-time graph sets, cost records and
logical trees consumed before physical scan lowering. */
(define join_order_set_subset? (lambda (required available)
	(reduce (coalesceNil required '()) (lambda (ok alias)
		(and ok (contains? available alias))) true)))

(define join_order_set_intersects? (lambda (left right)
	(reduce (coalesceNil left '()) (lambda (found alias)
		(or found (contains? right alias))) false)))

(define join_order_set_union (lambda (universe left right)
	(filter universe (lambda (alias)
		(or (contains? left alias) (contains? right alias))))))

(define join_order_set_difference (lambda (universe left right)
	(filter universe (lambda (alias)
		(and (contains? left alias) (not (contains? right alias)))))))

(define join_order_set_key (lambda (aliases) (string aliases)))

(define join_order_pred_aliases (lambda (predicate) (car predicate)))
(define join_order_pred_selectivity (lambda (predicate) (cadr predicate)))
(define join_order_pred_origin (lambda (predicate)
	(if (> (count predicate) 2) (nth predicate 2) nil)))
(define join_order_pred_owner (lambda (predicate)
	(if (> (count predicate) 3) (nth predicate 3) nil)))
(define join_order_pred_expr (lambda (predicate)
	(if (> (count predicate) 4) (nth predicate 4) true)))
(define join_order_pred_barrier_owner (lambda (predicate)
	(if (> (count predicate) 5) (nth predicate 5) nil)))

(define join_order_predicate_crosses? (lambda (predicate left right)
	(begin
		(define refs (join_order_pred_aliases predicate))
		(and (join_order_set_subset? refs (merge_unique (list left right)))
			(and (join_order_set_intersects? refs left)
				(join_order_set_intersects? refs right))))))

(define join_order_predicate_crosses_in? (lambda (predicate left right combined)
	(begin
		(define refs (join_order_pred_aliases predicate))
		(and (join_order_set_subset? refs combined)
			(and (join_order_set_intersects? refs left)
				(join_order_set_intersects? refs right))))))

(define join_order_connected? (lambda (predicates left right)
	(begin
		(define combined (merge_unique (list left right)))
		(reduce predicates (lambda (connected predicate)
			(or connected (join_order_predicate_crosses_in? predicate left right combined))) false))))

/* plan = (tree aliases cardinality cost size atomic driver-cardinality left right) */
(define join_order_plan_tree (lambda (plan) (nth plan 0)))
(define join_order_plan_aliases (lambda (plan) (nth plan 1)))
(define join_order_plan_cardinality (lambda (plan) (nth plan 2)))
(define join_order_plan_cost (lambda (plan) (nth plan 3)))
(define join_order_plan_size (lambda (plan) (nth plan 4)))
(define join_order_plan_atomic? (lambda (plan) (nth plan 5)))
(define join_order_driver_cardinality (lambda (plan) (nth plan 6)))
(define join_order_plan_left (lambda (plan) (nth plan 7)))
(define join_order_plan_right (lambda (plan) (nth plan 8)))

(define join_order_leaf_plan (lambda (node)
	(list
		(list (quote join-leaf) (car node) '())
		(list (car node))
		(max 1 (cadr node))
		0
		1
		false
		(max 1 (cadr node))
		nil nil)))

(define join_order_join_cardinality (lambda (predicates left right)
	(begin
		(define combined (merge_unique (list
			(join_order_plan_aliases left) (join_order_plan_aliases right))))
		(define selectivity (reduce predicates (lambda (value predicate)
			(if (join_order_predicate_crosses_in? predicate
				(join_order_plan_aliases left) (join_order_plan_aliases right) combined)
				(* value (join_order_pred_selectivity predicate))
				value)) 1))
		(max 1 (min 1e300 (*
			(join_order_plan_cardinality left)
			(join_order_plan_cardinality right)
			selectivity))))))

(define join_order_join_plan (lambda (universe predicates left right)
	(begin
		(define cardinality (join_order_join_cardinality predicates left right))
		(define combined (join_order_set_union universe
			(join_order_plan_aliases left) (join_order_plan_aliases right)))
		(list
			(list (quote join-node) (quote inner) (join_order_plan_tree left) (join_order_plan_tree right) '())
			combined
			cardinality
			(+ (join_order_plan_cost left) (join_order_plan_cost right) cardinality)
			(+ (join_order_plan_size left) (join_order_plan_size right))
			false
			(join_order_driver_cardinality left)
			left right))))

(define join_order_better_plan (lambda (current candidate)
	(if (nil? current)
		candidate
		(if (< (join_order_plan_cost candidate) (join_order_plan_cost current))
			candidate
			(if (and (equal? (join_order_plan_cost candidate) (join_order_plan_cost current))
				(< (join_order_driver_cardinality candidate) (join_order_driver_cardinality current)))
				candidate current)))))

(define join_order_best_orientation (lambda (universe predicates left right)
	(join_order_better_plan
		(join_order_join_plan universe predicates left right)
		(join_order_join_plan universe predicates right left))))

(define join_order_hypergraph? (lambda (predicates)
	(reduce predicates (lambda (found predicate)
		(or found (> (count (join_order_pred_aliases predicate)) 2))) false)))

(define join_order_hyperedge_support (lambda (predicates)
	(merge (map predicates (lambda (predicate)
		(begin
			(define refs (join_order_pred_aliases predicate))
			(if (> (count refs) 2)
				(map (produceN (- (count refs) 1)) (lambda (i)
					(list (list (nth refs i) (nth refs (+ i 1))) 1)))
				'())))))))

(define join_order_component_expand (lambda (current predicates)
	(begin
		(define expanded (reduce predicates (lambda (aliases predicate)
			(if (join_order_set_intersects? aliases (join_order_pred_aliases predicate))
				(merge_unique (list aliases (join_order_pred_aliases predicate)))
				aliases)) current))
		(if (equal? expanded current) current
			(join_order_component_expand expanded predicates)))))

(define join_order_components (lambda (remaining predicates)
	(if (empty_list? remaining)
		'()
		(begin
			(define component (join_order_component_expand (list (car remaining)) predicates))
			(cons component (join_order_components
				(filter remaining (lambda (alias) (not (contains? component alias))))
				predicates))))))

(define join_order_cross_product_support (lambda (aliases predicates)
	(begin
		(define components (join_order_components aliases predicates))
		(if (< (count components) 2)
			'()
			(map (produceN (- (count components) 1)) (lambda (i)
				(list (list (car (nth components i)) (car (nth components (+ i 1)))) 1)))))))

(define join_order_prepare_predicates (lambda (aliases predicates)
	(begin
		(define with_hyper_support (merge (list predicates (join_order_hyperedge_support predicates))))
		(merge (list with_hyper_support
			(join_order_cross_product_support aliases with_hyper_support))))))

(define join_order_connected_expand (lambda (universe predicates current)
	(filter (map predicates (lambda (predicate)
		(begin
			(define refs (join_order_pred_aliases predicate))
			(if (and (join_order_set_intersects? current refs)
				(not (join_order_set_subset? refs current)))
				(join_order_set_union universe current refs)
				nil)))) (lambda (candidate) (not (nil? candidate))))))

(define join_order_new_connected_candidates (lambda (candidates seen added)
	(if (empty_list? candidates)
		(list seen (reverse added))
		(begin
			(define candidate (car candidates))
			(define key (join_order_set_key candidate))
			(if (nil? (get_assoc seen key nil))
				(join_order_new_connected_candidates (cdr candidates)
					(set_assoc seen key true) (cons candidate added))
				(join_order_new_connected_candidates (cdr candidates) seen added))))))

(define join_order_enumerate_connected_loop (lambda (universe predicates frontier seen sets set_count budget)
	(if (empty_list? frontier)
		(list sets false)
		(begin
			(define current (car frontier))
			(define candidates (join_order_connected_expand universe predicates current))
			(define new_candidates (join_order_new_connected_candidates candidates seen '()))
			(define next_seen (car new_candidates))
			(define added (cadr new_candidates))
			(define next_sets (merge (list sets added)))
			(define next_count (+ set_count (count added)))
			(if (and (> budget 0) (> next_count budget))
				(list next_sets true)
				(join_order_enumerate_connected_loop universe predicates
					(merge (list (cdr frontier) added)) next_seen next_sets next_count budget))))))

(define join_order_enumerate_connected (lambda (universe predicates budget)
	(begin
		(define singletons (map universe list))
		(define seen (reduce singletons (lambda (dict singleton)
			(set_assoc dict (join_order_set_key singleton) true)) '()))
		(join_order_enumerate_connected_loop universe predicates singletons seen singletons (count singletons) budget))))

(define join_order_find_node (lambda (nodes alias)
	(find nodes (lambda (node) (equal? (car node) alias)) nil)))

(define join_order_sort_sets (lambda (sets)
	(sort sets (lambda (left right)
		(< (count left) (count right))))))

(define join_order_dphyp_set_plan (lambda (universe predicates connected_by_first plans aliases)
	(reduce (get_assoc connected_by_first (car aliases) '()) (lambda (best left_aliases)
		(if (and (< (count left_aliases) (count aliases))
			(and (equal? (car aliases) (car left_aliases))
				(join_order_set_subset? left_aliases aliases)))
			(begin
				(define right_aliases (join_order_set_difference universe aliases left_aliases))
				(define left (get_assoc plans (join_order_set_key left_aliases) nil))
				(define right (get_assoc plans (join_order_set_key right_aliases) nil))
				(if (and (not (nil? left)) (not (nil? right))
					(join_order_connected? predicates left_aliases right_aliases))
					(join_order_better_plan best
						(join_order_best_orientation universe predicates left right))
					best))
			best)) nil)))

(define join_order_dphyp_fill (lambda (nodes universe predicates connected_by_first remaining plans entries)
	(if (empty_list? remaining)
		(list (get_assoc plans (join_order_set_key universe) nil) entries)
		(begin
			(define aliases (car remaining))
			(define plan (if (single_source? aliases)
				(join_order_leaf_plan (join_order_find_node nodes (car aliases)))
				(join_order_dphyp_set_plan universe predicates connected_by_first plans aliases)))
			(join_order_dphyp_fill nodes universe predicates connected_by_first (cdr remaining)
				(if (nil? plan) plans (set_assoc plans (join_order_set_key aliases) plan))
				(if (nil? plan) entries (+ entries 1)))))))

(define join_order_dphyp (lambda (nodes aliases predicates)
	(begin
		(define connected (join_order_sort_sets (car (join_order_enumerate_connected aliases predicates 0))))
		(define connected_by_first (reduce connected (lambda (dict subset)
			(begin
				(define first (car subset))
				(set_assoc dict first (append (get_assoc dict first '()) subset)))) '()))
		(join_order_dphyp_fill nodes aliases predicates connected_by_first connected '() 0))))

(define join_order_alias_position (lambda (aliases alias)
	(reduce (produceN (count aliases)) (lambda (found i)
		(if (not (nil? found)) found
			(if (equal? (nth aliases i) alias) i nil))) nil)))

(define join_order_regular_edges (lambda (aliases predicates)
	(begin
		(define edge_dict (reduce predicates (lambda (dict predicate)
			(begin
				(define refs (join_order_pred_aliases predicate))
				(if (not (equal? (count refs) 2))
					dict
					(begin
						(define left (car refs))
						(define right (cadr refs))
						(define ordered (if (< (join_order_alias_position aliases left)
							(join_order_alias_position aliases right))
							(list left right) (list right left)))
						(define key (string ordered))
						(define old (get_assoc dict key nil))
						(set_assoc dict key (list (car ordered) (cadr ordered)
							(* (if (nil? old) 1 (nth old 2))
								(join_order_pred_selectivity predicate)))))))) '()))
		(extract_assoc edge_dict (lambda (_key edge) edge)))))

(define join_order_component_for_alias (lambda (forest alias)
	(find forest (lambda (component) (contains? component alias)) nil)))

(define join_order_mst_fold (lambda (state edge)
	(begin
		(define forest (car state))
		(define selected (cadr state))
		(define left_component (join_order_component_for_alias forest (nth edge 0)))
		(define right_component (join_order_component_for_alias forest (nth edge 1)))
		(if (equal? left_component right_component)
			state
			(list
				(cons (merge_unique (list left_component right_component))
					(filter forest (lambda (component)
						(and (not (equal? component left_component))
							(not (equal? component right_component))))))
				(append selected edge))))))

(define join_order_minimum_spanning_tree (lambda (aliases predicates)
	(begin
		(define edges (sort (join_order_regular_edges aliases predicates)
			(lambda (left right) (< (nth left 2) (nth right 2)))))
		(cadr (reduce edges join_order_mst_fold
			(list (map aliases list) '()))))))

/* IKKBZ item = (node-aliases factor cost). */
(define join_order_ikkbz_rank (lambda (item)
	(if (equal? (nth item 2) 0) 1e300
		(/ (- (nth item 1) 1) (nth item 2)))))

(define join_order_ikkbz_merge_items (lambda (left right)
	(list
		(merge (list (nth left 0) (nth right 0)))
		(* (nth left 1) (nth right 1))
		(+ (nth left 2) (* (nth left 1) (nth right 2))))))

(define join_order_ikkbz_normalize_once (lambda (chain prefix)
	(match chain
		(cons left (cons right rest))
		(if (> (join_order_ikkbz_rank left) (join_order_ikkbz_rank right))
			(list true (merge (list (reverse prefix)
				(list (join_order_ikkbz_merge_items left right)) rest)))
			(join_order_ikkbz_normalize_once (cons right rest) (cons left prefix)))
		_ (list false chain))))

(define join_order_ikkbz_normalize (lambda (chain)
	(begin
		(define normalized (join_order_ikkbz_normalize_once chain '()))
		(if (car normalized)
			(join_order_ikkbz_normalize (cadr normalized))
			chain))))

(define join_order_ikkbz_merge_chains (lambda (chains result)
	(begin
		(define available (filter chains (lambda (chain) (not (empty_list? chain)))))
		(if (empty_list? available)
			result
			(begin
				(define best (reduce available (lambda (current chain)
					(if (or (nil? current)
						(< (join_order_ikkbz_rank (car chain))
							(join_order_ikkbz_rank (car current))))
						chain current)) nil))
				(join_order_ikkbz_merge_chains
					(cons (cdr best) (filter available (lambda (chain) (not (equal? chain best)))))
					(append result (car best))))))))

(define join_order_mst_edges_for_alias (lambda (edges alias parent)
	(filter edges (lambda (edge)
		(and (or (equal? (nth edge 0) alias) (equal? (nth edge 1) alias))
			(not (or (equal? (nth edge 0) parent) (equal? (nth edge 1) parent))))))))

(define join_order_edge_other_alias (lambda (edge alias)
	(if (equal? (nth edge 0) alias) (nth edge 1) (nth edge 0))))

(define join_order_ikkbz_chain (lambda (nodes edges alias parent parent_selectivity)
	(begin
		(define child_chains (map (join_order_mst_edges_for_alias edges alias parent) (lambda (edge)
			(join_order_ikkbz_normalize
				(join_order_ikkbz_chain nodes edges
					(join_order_edge_other_alias edge alias) alias (nth edge 2))))))
		(define node (join_order_find_node nodes alias))
		(cons (list (list alias)
			(if (nil? parent) 1 (* (cadr node) parent_selectivity))
			(if (nil? parent) 0 (cadr node)))
			(join_order_ikkbz_merge_chains child_chains '())))))

(define join_order_ikkbz_order (lambda (nodes aliases predicates)
	(begin
		(define edges (join_order_minimum_spanning_tree aliases predicates))
		(define best (reduce aliases (lambda (current root)
			(begin
				(define chain (join_order_ikkbz_chain nodes edges root nil 1))
				(define sequence (reduce (cdr chain) join_order_ikkbz_merge_items (car chain)))
				(if (or (nil? current) (< (nth sequence 2) (cadr current)))
					(list (merge (map chain (lambda (item) (nth item 0)))) (nth sequence 2))
					current))) nil))
		(car best))))

(define join_order_interval_key (lambda (start end) (concat start ":" end)))

(define join_order_linearized_splits (lambda (universe predicates table start end split best)
	(if (>= split end)
		best
		(begin
			(define left (get_assoc table (join_order_interval_key start split) nil))
			(define right (get_assoc table (join_order_interval_key (+ split 1) end) nil))
			(join_order_linearized_splits universe predicates table start end (+ split 1)
				(if (and (not (nil? left)) (not (nil? right))
					(join_order_connected? predicates
						(join_order_plan_aliases left) (join_order_plan_aliases right)))
					(join_order_better_plan best
						(join_order_join_plan universe predicates left right))
					best))))))

(define join_order_linearized_starts (lambda (universe predicates table length start entries)
	(if (> (+ start length) (count universe))
		(list table entries)
		(begin
			(define end (- (+ start length) 1))
			(define best (join_order_linearized_splits universe predicates table start end start nil))
			(join_order_linearized_starts universe predicates
				(if (nil? best) table (set_assoc table (join_order_interval_key start end) best))
				length (+ start 1) (if (nil? best) entries (+ entries 1)))))))

(define join_order_linearized_lengths (lambda (universe predicates table length entries)
	(if (> length (count universe))
		(list (get_assoc table (join_order_interval_key 0 (- (count universe) 1)) nil) entries)
		(begin
			(define filled (join_order_linearized_starts universe predicates table length 0 entries))
			(join_order_linearized_lengths universe predicates (car filled) (+ length 1) (cadr filled))))))

(define join_order_linearized_dp (lambda (nodes aliases predicates)
	(begin
		(define order (join_order_ikkbz_order nodes aliases predicates))
		(define table (reduce (produceN (count order)) (lambda (dict i)
			(set_assoc dict (join_order_interval_key i i)
				(join_order_leaf_plan (join_order_find_node nodes (nth order i))))) '()))
		(join_order_linearized_lengths aliases predicates table 2 (count order)))))

(define join_order_goo_pairs (lambda (plans predicates require_connection)
	(merge (mapIndex plans (lambda (left_index left)
		(mapIndex plans (lambda (right_index right)
			(if (and (< left_index right_index)
				(or (not require_connection)
					(join_order_connected? predicates
						(join_order_plan_aliases left) (join_order_plan_aliases right))))
				(list left_index right_index
					(join_order_join_cardinality predicates left right))
				nil))))))))

(define join_order_goo_best_pair (lambda (plans predicates)
	(begin
		(define connected (filter (join_order_goo_pairs plans predicates true)
			(lambda (pair) (not (nil? pair)))))
		(define candidates (if (empty_list? connected)
			(filter (join_order_goo_pairs plans predicates false) (lambda (pair) (not (nil? pair))))
			connected))
		(reduce candidates (lambda (best pair)
			(if (or (nil? best) (< (nth pair 2) (nth best 2))) pair best)) nil))))

(define join_order_goo_loop (lambda (universe predicates plans)
	(if (single_source? plans)
		(car plans)
		(begin
			(define pair (join_order_goo_best_pair plans predicates))
			(define left_index (nth pair 0))
			(define right_index (nth pair 1))
			(define joined (join_order_best_orientation universe predicates
				(nth plans left_index) (nth plans right_index)))
			(join_order_goo_loop universe predicates
				(append (filter (mapIndex plans (lambda (i plan)
					(if (or (equal? i left_index) (equal? i right_index)) nil plan)))
					(lambda (plan) (not (nil? plan)))) joined))))))

(define join_order_goo (lambda (nodes aliases predicates)
	(join_order_goo_loop aliases predicates
		(map aliases (lambda (alias)
			(join_order_leaf_plan (join_order_find_node nodes alias)))))))

(define join_order_plan_with_atomic (lambda (plan atomic)
	(list
		(join_order_plan_tree plan)
		(join_order_plan_aliases plan)
		(join_order_plan_cardinality plan)
		(join_order_plan_cost plan)
		(join_order_plan_size plan)
		atomic
		(join_order_driver_cardinality plan)
		(join_order_plan_left plan)
		(join_order_plan_right plan))))

(define join_order_expensive_subtree (lambda (plan parent_size limit)
	(if (or (nil? plan) (join_order_plan_atomic? plan)
		(equal? (join_order_plan_size plan) 1))
		nil
		(begin
			(define own (if (and (<= (join_order_plan_size plan) limit) (> parent_size limit)) plan nil))
			(define left (join_order_expensive_subtree
				(join_order_plan_left plan) (join_order_plan_size plan) limit))
			(define right (join_order_expensive_subtree
				(join_order_plan_right plan) (join_order_plan_size plan) limit))
			(reduce (list own left right) (lambda (best candidate)
				(if (and (not (nil? candidate))
					(or (nil? best) (> (join_order_plan_cost candidate) (join_order_plan_cost best))))
					candidate best)) nil)))))

(define join_order_replace_subtree (lambda (universe predicates plan target replacement)
	(if (equal? plan target)
		replacement
		(if (nil? (join_order_plan_left plan))
			plan
			(join_order_join_plan universe predicates
				(join_order_replace_subtree universe predicates (join_order_plan_left plan) target replacement)
				(join_order_replace_subtree universe predicates (join_order_plan_right plan) target replacement))))))

(define join_order_goo_dp_loop (lambda (nodes aliases predicates hypergraph plan budget used)
	(if (<= budget 0)
		(list plan used)
		(begin
			(define limit (if hypergraph 10 100))
			(define target (join_order_expensive_subtree plan (+ (join_order_plan_size plan) 1) limit))
			(if (nil? target)
				(list plan used)
				(begin
					(define optimized (if hypergraph
						(join_order_dphyp nodes (join_order_plan_aliases target) predicates)
						(join_order_linearized_dp nodes (join_order_plan_aliases target) predicates)))
					(define replacement (car optimized))
					(define entries (cadr optimized))
					(define accepted (and (not (nil? replacement))
						(< (join_order_plan_cost replacement) (join_order_plan_cost target))))
					(define final_replacement (join_order_plan_with_atomic
						(if accepted replacement target) true))
					(join_order_goo_dp_loop nodes aliases predicates hypergraph
						(join_order_replace_subtree aliases predicates plan target final_replacement)
						(- budget entries) (+ used entries))))))))

(define join_order_goo_dp (lambda (nodes aliases predicates hypergraph)
	(join_order_goo_dp_loop nodes aliases predicates hypergraph
		(join_order_goo nodes aliases predicates) 10000 0)))

(define join_order_local_predicates_for_alias (lambda (predicates alias)
	(filter predicates (lambda (predicate)
		(and (not (nil? (join_order_pred_origin predicate)))
			(and (not (equal? (join_order_pred_origin predicate) (quote outer-on)))
				(and (nil? (join_order_pred_barrier_owner predicate))
					(equal? (join_order_pred_aliases predicate) (list alias)))))))))

(define join_order_predicate_owned_by_barrier? (lambda (predicate kind right)
	(begin
		(define owner (join_order_pred_barrier_owner predicate))
		(and (not (nil? owner))
			(and (equal? kind (quote left-outer))
				(equal? owner (join_optimizer_tree_first_alias right)))))))

(define join_order_tree_with_predicates (lambda (tree predicates)
	(match tree
		((symbol join-leaf) alias) (list (quote join-leaf) alias
			(join_order_local_predicates_for_alias predicates alias))
		((quote join-leaf) alias) (list (quote join-leaf) alias
			(join_order_local_predicates_for_alias predicates alias))
		((symbol join-leaf) alias _predicates) (list (quote join-leaf) alias
			(join_order_local_predicates_for_alias predicates alias))
		((quote join-leaf) alias _predicates) (list (quote join-leaf) alias
			(join_order_local_predicates_for_alias predicates alias))
		((symbol join-node) kind left right _predicates)
		(begin
			(define left_aliases (join_optimizer_tree_aliases left))
			(define right_aliases (join_optimizer_tree_aliases right))
			(define combined (merge_unique (list left_aliases right_aliases)))
			(list (quote join-node) kind
				(join_order_tree_with_predicates left predicates)
				(join_order_tree_with_predicates right predicates)
				(filter predicates (lambda (predicate)
					(and (not (nil? (join_order_pred_origin predicate)))
						(or (join_order_predicate_crosses_in? predicate
							left_aliases right_aliases combined)
							(join_order_predicate_owned_by_barrier? predicate kind right)))))))
		((quote join-node) kind left right _predicates)
		(join_order_tree_with_predicates
			(list (quote join-node) kind left right '()) predicates)
		_ (neumann_fail "join_reorder" "malformed optimized join tree"))))

(define join_order_result (lambda (strategy plan entries predicates)
	(list
		(list (quote strategy) strategy)
		(list (quote tree) (join_order_tree_with_predicates (join_order_plan_tree plan) predicates))
		(list (quote order) (join_order_plan_aliases plan))
		(list (quote cost) (join_order_plan_cost plan))
		(list (quote cardinality) (join_order_plan_cardinality plan))
		(list (quote dp_entries) entries))))

(define join_order_choose_strategy (lambda (alias_count hypergraph connected_over_budget)
	(if (or (< alias_count 14)
		(and (<= alias_count 100) (not connected_over_budget)))
		(quote dphyp)
		(if hypergraph
			(quote goo-dphyp)
			(if (<= alias_count 100)
				(quote linearized-dp)
				(quote goo-linearized-dp))))))

/* Every subset containing one vertex and any selection of its regular-edge
neighbors is connected. A degree d therefore proves at least 2^d connected
subsets without materializing them. */
(define join_order_degree_exceeds_budget? (lambda (degree budget)
	(if (<= degree 0)
		false
		(if (< budget 2)
			true
			(join_order_degree_exceeds_budget? (- degree 1) (/ budget 2))))))

(define join_order_regular_neighbors (lambda (edges alias)
	(map
		(filter edges (lambda (edge)
			(or (equal? (nth edge 0) alias) (equal? (nth edge 1) alias))))
		(lambda (edge)
			(if (equal? (nth edge 0) alias) (nth edge 1) (nth edge 0))))))

(define join_order_degree_proves_budget_overflow? (lambda (aliases edges budget)
	(reduce aliases (lambda (proven alias)
		(or proven (join_order_degree_exceeds_budget?
			(count (join_order_regular_neighbors edges alias)) budget))) false)))

(define join_order_adaptive (lambda (nodes raw_predicates)
	(begin
		(define aliases (map nodes car))
		(define hypergraph (join_order_hypergraph? raw_predicates))
		(define predicates (join_order_prepare_predicates aliases raw_predicates))
		(define regular_edges (join_order_regular_edges aliases predicates))
		(define regular_edge_count (count regular_edges))
		(define complete_regular_graph (equal? regular_edge_count
			(/ (* (count aliases) (- (count aliases) 1)) 2)))
		(define connected_count (if (and (>= (count aliases) 14) (<= (count aliases) 100))
			(if (or complete_regular_graph
				(join_order_degree_proves_budget_overflow? aliases regular_edges 10000))
				(list '() true)
				(join_order_enumerate_connected aliases predicates 10000))
			(list '() true)))
		(define strategy (join_order_choose_strategy
			(count aliases) hypergraph (cadr connected_count)))
		(define exact (equal? strategy (quote dphyp)))
		(define result (if exact
			(join_order_dphyp nodes aliases predicates)
			(if (equal? strategy (quote linearized-dp))
				(join_order_linearized_dp nodes aliases predicates)
				(join_order_goo_dp nodes aliases predicates hypergraph))))
		(if (nil? (car result))
			(neumann_fail "join_reorder" (concat "SCM join ordering could not construct a connected plan: " (string (list nodes predicates result))))
			(join_order_result strategy (car result) (cadr result) predicates))))))

(define make_join_optimizer_leaf (lambda (alias)
	(list (quote join-leaf) alias '())))

(define make_join_optimizer_node (lambda (kind left right predicates)
	(list (quote join-node) kind left right (coalesceNil predicates '()))))

(define join_optimizer_source_predicates (lambda (aliases src)
	(begin
		(define predicate (coalesceNil (source_join_expr src) true))
		(if (equal? predicate true)
			'()
			(map (split_and_terms predicate) (lambda (term)
				(list
					(merge_unique (list
						(join_hypergraph_expr_aliases (car aliases) aliases term)
						(list (source_alias src))))
					1
					(if (source_outer? src) (quote outer-on) (quote inner-on))
					(source_alias src)
					term)))))))

(define join_optimizer_source_join_kind (lambda (src)
	(if (source_outer? src) (quote left-outer) (quote inner))))

(define join_optimizer_left_deep_tree (lambda (sources)
	(match sources
		(cons src rest)
		(reduce rest (lambda (tree next_src)
			(make_join_optimizer_node (join_optimizer_source_join_kind next_src)
				tree (make_join_optimizer_leaf (source_alias next_src))
				(join_optimizer_source_predicates (source_aliases sources) next_src)))
			(make_join_optimizer_leaf (source_alias src)))
		_ nil)))

(define join_optimizer_append_tree (lambda (tree all_aliases src)
	(if (nil? tree)
		(make_join_optimizer_leaf (source_alias src))
		(make_join_optimizer_node (join_optimizer_source_join_kind src)
			tree (make_join_optimizer_leaf (source_alias src))
			(join_optimizer_source_predicates all_aliases src)))))

(define join_optimizer_append_sources_tree (lambda (tree sources)
	(begin
		(define all_aliases (merge (list (join_optimizer_tree_aliases tree) (source_aliases sources))))
		(reduce sources (lambda (current src)
			(join_optimizer_append_tree current all_aliases src)) tree))))

(define join_optimizer_tree_aliases (lambda (tree)
	(match tree
		((symbol join-leaf) alias) (list alias)
		((quote join-leaf) alias) (list alias)
		((symbol join-leaf) alias _predicates) (list alias)
		((quote join-leaf) alias _predicates) (list alias)
		((symbol join-node) _kind left right _predicates) (merge (list
			(join_optimizer_tree_aliases left)
			(join_optimizer_tree_aliases right)))
		((quote join-node) _kind left right _predicates) (merge (list
			(join_optimizer_tree_aliases left)
			(join_optimizer_tree_aliases right)))
		_ (neumann_fail "join_reorder" "malformed logical join plan"))))

(define join_optimizer_tree_first_alias (lambda (tree)
	(match tree
		((symbol join-leaf) alias) alias
		((quote join-leaf) alias) alias
		((symbol join-leaf) alias _predicates) alias
		((quote join-leaf) alias _predicates) alias
		((symbol join-node) _kind left _right _predicates) (join_optimizer_tree_first_alias left)
		((quote join-node) _kind left _right _predicates) (join_optimizer_tree_first_alias left)
		_ (neumann_fail "join_reorder" "malformed logical join plan"))))

(define join_optimizer_tree_predicates (lambda (tree)
	(match tree
		((symbol join-leaf) _alias) '()
		((quote join-leaf) _alias) '()
		((symbol join-leaf) _alias predicates) predicates
		((quote join-leaf) _alias predicates) predicates
		((symbol join-node) _kind left right predicates) (merge (list
			(join_optimizer_tree_predicates left)
			(join_optimizer_tree_predicates right)
			predicates))
		((quote join-node) kind left right predicates)
		(join_optimizer_tree_predicates
			(make_join_optimizer_node kind left right predicates))
		_ (neumann_fail "join_reorder" "malformed logical join plan"))))

(define join_optimizer_leaf_predicates (lambda (leaf)
	(match leaf
		((symbol join-leaf) _alias) '()
		((quote join-leaf) _alias) '()
		((symbol join-leaf) _alias predicates) predicates
		((quote join-leaf) _alias predicates) predicates
		_ (neumann_fail "build_queryplan" "expected a logical join leaf"))))

(define join_optimizer_node_condition (lambda (predicates)
	(combine_where_terms
		(map (filter (coalesceNil predicates '()) (lambda (predicate)
			(not (equal? (join_order_pred_origin predicate) (quote outer-on)))))
			join_order_pred_expr)
		true)))

(define condition_without_join_tree_predicates (lambda (condition tree)
	(begin
		(define owned_exprs (map (filter (join_optimizer_tree_predicates tree)
			(lambda (predicate)
				(not (equal? (join_order_pred_origin predicate) (quote outer-on)))))
			join_order_pred_expr))
		(combine_where_terms
			(filter (split_and_terms (coalesceNil condition true))
				(lambda (term) (not (contains? owned_exprs term))))
			true))))

(define join_optimizer_tree_without_aliases (lambda (tree removed_aliases)
	(match tree
		((symbol join-leaf) alias) (if (contains? removed_aliases alias) nil tree)
		((quote join-leaf) alias) (if (contains? removed_aliases alias) nil tree)
		((symbol join-leaf) alias _predicates) (if (contains? removed_aliases alias) nil tree)
		((quote join-leaf) alias _predicates) (if (contains? removed_aliases alias) nil tree)
		((symbol join-node) kind left right predicates)
		(begin
			(define kept_left (join_optimizer_tree_without_aliases left removed_aliases))
			(define kept_right (join_optimizer_tree_without_aliases right removed_aliases))
			(if (nil? kept_left) kept_right
				(if (nil? kept_right) kept_left
					(begin
						(define kept_aliases (merge (list
							(join_optimizer_tree_aliases kept_left)
							(join_optimizer_tree_aliases kept_right))))
						(make_join_optimizer_node kind kept_left kept_right
							(filter predicates (lambda (predicate)
								(join_optimizer_alias_subset?
									(join_order_pred_aliases predicate) kept_aliases))))))))
		((quote join-node) kind left right predicates)
		(join_optimizer_tree_without_aliases
			(make_join_optimizer_node kind left right predicates) removed_aliases)
		_ (neumann_fail "build_queryplan" "malformed logical join plan while removing consumed sources"))))

(define join_optimizer_facts_without_aliases (lambda (facts removed_aliases)
	(begin
		(define tree (qassoc_get facts (quote join_plan) nil))
		(if (or (nil? tree) (empty_list? removed_aliases))
			facts
			(qassoc_set facts (quote join_plan)
				(join_optimizer_tree_without_aliases tree removed_aliases))))))

/* Physical lowering consumes the logical tree structurally. The query-block
source list remains the semantic catalog and is used only for alias resolution. */
(define physical_join_tree? (lambda (tree)
	(match tree
		((symbol join-leaf) _alias) true
		((quote join-leaf) _alias) true
		((symbol join-leaf) _alias _predicates) true
		((quote join-leaf) _alias _predicates) true
		((symbol join-node) _kind _left _right _predicates) true
		((quote join-node) _kind _left _right _predicates) true
		_ false)))

(define physical_join_leaf_source (lambda (catalog leaf)
	(begin
		(define alias (match leaf
			((symbol join-leaf) value) value
			((quote join-leaf) value) value
			((symbol join-leaf) value _predicates) value
			((quote join-leaf) value _predicates) value
			_ (neumann_fail "build_queryplan" "expected a logical join leaf")))
		(define src (join_optimizer_source_by_alias catalog alias))
		(if (nil? src)
			(neumann_fail "build_queryplan" "join tree leaf references an unknown source")
			src))))

(define physical_join_plan_for_sources (lambda (sources)
	(if (physical_join_tree? sources) sources (join_optimizer_left_deep_tree sources))))

(define query_block_join_plan (lambda (block sources)
	(begin
		(define tree (qassoc_get (qb_facts block) (quote join_plan) nil))
		(if (and (not (nil? tree))
			(and (join_optimizer_alias_subset? (join_optimizer_tree_aliases tree) (map sources source_alias))
				(join_optimizer_alias_subset? (map sources source_alias) (join_optimizer_tree_aliases tree))))
			tree (physical_join_plan_for_sources sources)))))

/* Physical preparation validates tree coverage but never rewrites the semantic
source catalog. join_plan remains the single owner of physical join order. */
(define apply_join_optimizer_plan (lambda (block)
	(begin
		(define tree (qassoc_get (qb_facts block) (quote join_plan) nil))
		(if (nil? tree)
			block
			(begin
				(define aliases (join_optimizer_tree_aliases tree))
				(define source_alias_list (map (qb_sources block) source_alias))
				(if (and
					(equal? (count aliases) (count source_alias_list))
					(and
						(join_optimizer_alias_subset? aliases source_alias_list)
						(join_optimizer_alias_subset? source_alias_list aliases)))
					true
					(neumann_fail "build_queryplan" "logical join plan does not cover the query-block sources exactly once"))
				block)))))

(define apply_join_optimizer_plan_node (lambda (node)
	(if (query_block? node)
		(begin
			(define planned (apply_join_optimizer_plan node))
			(make_query_block
				(qb_schema planned)
				(qb_sources planned)
				(qb_fields planned)
				(qb_where planned)
				(qb_group planned)
				(qb_having planned)
				(qb_order planned)
				(qb_limit planned)
				(qb_offset planned)
				(qb_hidden planned)
				(map (qb_stages planned) apply_join_optimizer_plan_stage)
				(qb_facts planned)))
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) apply_join_optimizer_plan_node)
				(union_order node)
				(union_limit node)
				(union_offset node)
				(union_facts node))
			node))))

(define apply_join_optimizer_plan_stage (lambda (stage)
	(if (group_stage? stage)
		(make_group_stage
			(gs_id stage)
			(apply_join_optimizer_plan_node (gs_input stage))
			(gs_domain stage)
			(gs_keys stage)
			(gs_aggregates stage)
			(gs_having stage)
			(gs_output stage)
			(gs_order stage)
			(gs_limit stage)
			(gs_offset stage)
			(gs_facts stage))
		stage)))

(define join_optimizer_plan_segment (lambda (stages all_sources segment default_alias graph)
	(begin
		(define aliases (map segment source_alias))
		(define planned (join_order_adaptive
			(join_optimizer_metadata_nodes stages segment default_alias graph)
			(join_optimizer_metadata_predicates all_sources default_alias graph aliases)))
		(qassoc_set planned (quote tree)
			(join_order_tree_with_predicates
				(qassoc_get planned (quote tree) nil)
				(join_optimizer_metadata_costed_predicates
					all_sources default_alias graph aliases))))))

(define join_optimizer_primary_key_columns (lambda (src)
	(if (source_is_base_table? src)
		(map (filter (get_schema (source_schema src) (source_relation src)) (lambda (col)
			(equal?? (col "Key") "PRI"))) (lambda (col) (col "Field")))
		'())))

(define join_optimizer_ref_matches? (lambda (ref src col)
	(and (not (nil? ref))
		(and (equal? (source_alias (car ref)) (source_alias src))
			(equal? (cadr ref) col)))))

(define join_optimizer_key_bound_to_driver? (lambda (sources default_alias graph driver lookup key_col)
	(reduce (join_optimizer_predicates graph) (lambda (found entry)
		(or found
			(match (qassoc_get entry (quote predicate) true)
				'(op left right) (if (or
					(equal? op (quote equal?))
					(equal? op (quote equal??))
					(equal? op (quote =)))
					(begin
						(define left_ref (join_optimizer_column_ref sources default_alias left))
						(define right_ref (join_optimizer_column_ref sources default_alias right))
						(or
							(and (join_optimizer_ref_matches? left_ref lookup key_col)
								(and (not (nil? right_ref))
									(equal? (source_alias (car right_ref)) (source_alias driver))))
							(and (join_optimizer_ref_matches? right_ref lookup key_col)
								(and (not (nil? left_ref))
									(equal? (source_alias (car left_ref)) (source_alias driver))))))
					false)
				_ false)))
		false)))

(define join_optimizer_unique_lookup_from_driver? (lambda (sources default_alias graph driver lookup)
	(begin
		(define key_cols (join_optimizer_primary_key_columns lookup))
		(and (not (empty_list? key_cols))
			(reduce key_cols (lambda (unique key_col)
				(and unique (join_optimizer_key_bound_to_driver?
					sources default_alias graph driver lookup key_col)))
				true)))))

(define join_optimizer_order_expr_available_from_driver? (lambda (sources default_alias graph driver expr)
	(begin
		(define aliases (join_hypergraph_expr_aliases default_alias (source_aliases sources) expr))
		(reduce aliases (lambda (available alias)
			(and available
				(or (equal? alias (source_alias driver))
					(begin
						(define lookup (join_optimizer_source_by_alias sources alias))
						(and (not (nil? lookup))
							(join_optimizer_unique_lookup_from_driver?
								sources default_alias graph driver lookup))))))
			true))))

(define join_optimizer_bounded_order_driver? (lambda (sources segment default_alias graph driver order_items)
	(and
		(reduce segment (lambda (safe src)
			(and safe
				(or (equal? (source_alias src) (source_alias driver))
					(join_optimizer_unique_lookup_from_driver?
						sources default_alias graph driver src))))
			true)
		(reduce (order_exprs order_items) (lambda (available expr)
			(and available (join_optimizer_order_expr_available_from_driver?
				sources default_alias graph driver expr)))
			true))))

(define join_optimizer_bounded_order_driver (lambda (sources segment default_alias graph order_items)
	(reduce segment (lambda (found candidate)
		(if (not (nil? found))
			found
			(if (join_optimizer_bounded_order_driver?
				sources segment default_alias graph candidate order_items)
				candidate
				nil)))
		nil)))

(define join_optimizer_promote_driver (lambda (sources driver)
	(cons driver (filter sources (lambda (src)
		(not (equal? (source_alias src) (source_alias driver))))))))

(define join_optimizer_order_alias_prefix_loop (lambda (sources segment_aliases default_alias exprs prefix)
	(match exprs
		(cons expr rest)
		(begin
			(define aliases (join_hypergraph_expr_aliases default_alias (source_aliases sources) expr))
			(if (empty_list? aliases)
				(join_optimizer_order_alias_prefix_loop sources segment_aliases default_alias rest prefix)
				(if (and (single_source? aliases) (contains? segment_aliases (car aliases)))
					(join_optimizer_order_alias_prefix_loop sources segment_aliases default_alias rest
						(append_unique prefix (car aliases)))
					(list false '()))))
		_ (list true prefix))))

(define join_optimizer_order_alias_prefix (lambda (sources segment default_alias order_items)
	(join_optimizer_order_alias_prefix_loop
		sources (map segment source_alias) default_alias (order_exprs order_items) '())))

(define join_optimizer_apply_order_prefix (lambda (planned all_sources segment default_alias graph prefix)
	(if (empty_list? prefix)
		planned
		(begin
			(define planned_aliases (qassoc_get planned (quote order) '()))
			(define ordered_aliases (merge (list prefix
				(filter planned_aliases (lambda (alias) (not (contains? prefix alias)))))))
			(define ordered_sources (join_optimizer_sources_for_order segment ordered_aliases))
			(define predicates (join_optimizer_metadata_costed_predicates
				all_sources default_alias graph ordered_aliases))
			(qassoc_set
				(qassoc_set
					(qassoc_set planned (quote tree)
						(join_order_tree_with_predicates
							(join_optimizer_left_deep_tree ordered_sources) predicates))
					(quote order) ordered_aliases)
				(quote strategy) (quote order-prefix))))))

(define join_optimizer_reorder_result (lambda (tree strategy dp_entries)
	(list tree strategy dp_entries)))

(define join_optimizer_reorder_sources (lambda (stage_catalog block graph)
	(begin
		(define sources (qb_sources block))
		(define segment (leading_reorderable_inner_sources stage_catalog sources))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(define preserve_order_driver (and
			(not (empty_list? segment))
			(query_limit_active? (qb_offset block) (qb_limit block))
			(empty_list? (qb_group block))
			(not (query_block_has_aggregates? block))
			(or
				(source_order_limit_driver? (car segment) (qb_order block) (qb_limit block))
				(source_row_number_limit_driver? (qb_stages block) (car segment)))))
		(define order_prefix_result (if (query_limit_active? (qb_offset block) (qb_limit block))
			(list false '())
			(join_optimizer_order_alias_prefix sources segment default_alias (qb_order block))))
		(if (or (< (count segment) 2) preserve_order_driver)
			(join_optimizer_reorder_result
				(join_optimizer_left_deep_tree sources)
				(if preserve_order_driver (quote preserve-order-limit) (quote fixed))
				0)
			(begin
				(define cost_planned (join_optimizer_plan_segment
					stage_catalog sources segment default_alias graph))
				(define planned (if (car order_prefix_result)
					(join_optimizer_apply_order_prefix cost_planned sources segment default_alias graph (cadr order_prefix_result))
					cost_planned))
				(define remaining (join_optimizer_drop_sources sources (count segment)))
				(join_optimizer_reorder_result
					(join_optimizer_append_sources_tree
						(qassoc_get planned (quote tree) nil) remaining)
					(qassoc_get planned (quote strategy) (quote fixed))
					(qassoc_get planned (quote dp_entries) 0)))))))

(define join_optimizer_drop_sources (lambda (sources amount)
	(if (or (<= amount 0) (empty_list? sources))
		sources
		(join_optimizer_drop_sources (cdr sources) (- amount 1)))))

(define join_optimizer_telemetry (lambda (graph reordered)
	(list
		(list (quote join_reorder_strategy) (nth reordered 1))
		(list (quote join_plan) (nth reordered 0))
		(list (quote join_driver) (car (join_optimizer_tree_aliases (nth reordered 0))))
		(list (quote join_dp_entries) (nth reordered 2))
		(list (quote join_graph_nodes) (count (qassoc_get graph (quote nodes) '())))
		(list (quote join_graph_edges) (count (qassoc_get graph (quote edges) '())))
		(list (quote join_graph_hyperedges) (count (qassoc_get graph (quote hyperedges) '())))
		(list (quote join_graph_barriers) (count (qassoc_get graph (quote barriers) '()))))))

(define hybrid_reorder_query_block_using (lambda (stage_catalog block)
	(if (or (empty_list? (qb_sources block)) (single_source? (qb_sources block)))
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
			(map (qb_stages block) (lambda (stage) (join_reorder_stage_using stage_catalog stage)))
			(qb_facts block))
		(begin
			(define provenance_graph (extract_join_hypergraph block))
			(define normalized (join_optimizer_normalize_inner_joins block))
			(define graph (join_hypergraph_with_provenance
				(extract_join_hypergraph normalized) provenance_graph))
			(define reordered (join_optimizer_reorder_sources stage_catalog normalized graph))
			(query_block_with_reorder_facts
				(make_query_block
					(qb_schema normalized)
					(qb_sources normalized)
					(qb_fields normalized)
					(qb_where normalized)
					(qb_group normalized)
					(qb_having normalized)
					(qb_order normalized)
					(qb_limit normalized)
					(qb_offset normalized)
					(qb_hidden normalized)
					(map (qb_stages normalized) (lambda (stage) (join_reorder_stage_using stage_catalog stage)))
					(qb_facts normalized))
				(merge (list
					(query_block_reorder_telemetry normalized)
					(join_optimizer_telemetry graph reordered))))))))

(define hybrid_reorder_query_block (lambda (block)
	(hybrid_reorder_query_block_using (qb_stages block) block)))
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

(define source_row_number_limit_driver? (lambda (stages src)
	(reduce (coalesceNil stages '()) (lambda (found stage)
		(or found
			(and (row_number_stage? stage)
				(and (source_alias_matches? (os_source stage) (source_alias src) (source_alias src) false)
					(not (nil? (extract_row_number_filter src (os_column stage) (source_join_expr src))))))))
		false)))

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
				(match (get_schema schema relation)
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

(define planner_column_distinct_estimate (lambda (src column)
	(if (or (not (source_is_base_table? src)) (nil? column))
		nil
		(try
			(lambda ()
				(reduce (get_schema (source_schema src) (source_relation src)) (lambda (found row)
					(if (not (nil? found))
						found
						(if (equal?? (row "Field") column) (row "DistinctEstimate") nil)))
					nil))
			(lambda (_e) nil)))))

(define planner_group_distinct_estimate (lambda (src keys row_count)
	(reduce keys (lambda (estimate key)
		(if (nil? estimate)
			nil
			(begin
				(define column (direct_column_name_for_alias src key))
				(define distinct (planner_column_distinct_estimate src column))
				(if (nil? distinct) nil (min row_count (* estimate distinct))))))
		1)))

(define direct_base_group_plan_preferred? (lambda (stage)
	(begin
		(define src (gs_input stage))
		(define keys (gs_keys stage))
		(define rows (planner_source_row_count src))
		(define distinct (if (or (nil? rows) (empty_list? keys))
			nil
			(planner_group_distinct_estimate src keys rows)))
		(and (source_is_base_table? src)
			(and (empty_list? (gs_order stage))
				(and (empty_list? (group_stage_session_domain_keys stage))
					(and (not (nil? rows))
						(and (>= rows 1024)
							(and (not (nil? distinct))
								(>= (* distinct 4) rows))))))))))

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

/* Identify the leading inner-join cloud that the logical optimizer may
reorder. Outer and stage-backed sources remain barriers and are appended to the
resulting tree in semantic order. */
(define reorderable_inner_driver_source? (lambda (stages src)
	(or
		(and (source_is_base_table? src)
			(and (not (source_outer? src))
				(or (nil? (source_join_expr src)) (equal? (source_join_expr src) true))))
		(join_optimizer_guaranteed_singleton_stage_source? stages src))))

(define leading_reorderable_inner_sources (lambda (stages sources)
	(match (coalesceNil sources '())
		(cons src rest) (if (reorderable_inner_driver_source? stages src)
			(cons src (leading_reorderable_inner_sources stages rest))
			'())
		_ '())))

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
			(if (source_is_base_table? input)
				(planner_source_row_count input)
				nil)))))

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
		(define class (if (or estimate_ratio_broad (and (nil? candidate_estimate) (candidate_stage_broad? stage))) (quote broad) (quote selective)))
		(define ordered_driver (if (nil? driver) false
			(or (source_order_limit_driver? driver (qb_order block) (qb_limit block))
				(source_row_number_limit_driver? (qb_stages block) driver))))
		(list
			(list (quote membership_selectivity_class) class)
			(list (quote membership_driver_rows) driver_rows)
			(list (quote membership_candidate_input_rows) candidate_rows)
			(list (quote membership_candidate_estimated_rows) estimate_rows)
			(list (quote membership_candidate_estimate_capped) estimate_capped)
			(list (quote membership_candidate_estimate_input) estimate_input)
			(list (quote membership_order_limit) (qb_limit block))
			(list (quote membership_order_limit_driver) ordered_driver)))))

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

(define candidate_reorder_strategy (lambda (telemetry)
	(begin
		(define estimated_rows (qassoc_get telemetry (quote membership_candidate_estimated_rows) nil))
		(define candidate_rows (if (qassoc_get telemetry (quote membership_candidate_estimate_capped) false)
			(qassoc_get telemetry (quote membership_candidate_estimate_input) nil)
			estimated_rows))
		(define driver_rows (qassoc_get telemetry (quote membership_driver_rows) nil))
		/* UNION membership is already in a canonical semantic form here. Cost
		the two supported implementations for every query shape: project the
		candidate keys to a driver RecSet, or retain the driver scan and let its
		per-row membership probes use autoindex. ORDER/LIMIT is an input to future
		braking refinements, not permission for syntax to force either plan. */
		(if (membership_projection_cost_preferred? candidate_rows driver_rows)
			(quote candidate_keyset)
			(quote driver_order_membership_probe)))))

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

(define membership_truth_parts (lambda (expr)
	(match expr
		((symbol membership_truth) probe stage_alias count_col)
		(list probe stage_alias count_col)
		((quote membership_truth) probe stage_alias count_col)
		(list probe stage_alias count_col)
		_ nil)))

/* Physical membership selection operates only on the canonical primitive. It
must not inspect parser expressions or table names. Extend its capability and
cost inputs when a new physical strategy is added; keep SQL-shape recognition
in the canonicalization functions above. */

(define membership_truth_stage (lambda (stages sources stage_alias)
	(begin
		(define src (source_for_alias sources nil stage_alias false))
		(if (or (nil? src) (not (stage_output_relation? (source_relation src))))
			nil
			(stage_by_id stages (stage_output_relation_id (source_relation src)))))))

(define expr_contains_membership_truth? (lambda (expr)
	(if (not (nil? (membership_truth_parts expr)))
		true
		(match expr
			(cons _head tail) (reduce tail (lambda (found item)
				(or found (expr_contains_membership_truth? item))) false)
			_ false))))

(define query_block_has_membership_truth_stage? (lambda (block)
	(reduce (qb_stages block) (lambda (found stage)
		(or found
			(and (group_stage? stage)
				(equal? (qassoc_get (gs_facts stage) (quote purpose) nil)
					(quote in_membership)))))
		false)))

(define membership_estimated_work_rows (lambda (estimate fallback)
	(if (nil? estimate)
		fallback
		(if (qassoc_get estimate (quote capped) false)
			(coalesceNil (qassoc_get estimate (quote input) nil) fallback)
			(coalesceNil (qassoc_get estimate (quote rows) nil) fallback)))))

(define membership_driver_filter (lambda (condition)
	/* Only top-level conjuncts which do not contain membership are guaranteed
	to restrict every evaluation of the membership formula. Predicates inside
	an OR branch cannot reduce the driver cost: a sibling may still admit the
	row. Keeping this estimate conservative is preferable to choosing a RecSet
	from an invalid branch-local selectivity assumption. */
	(combine_where_terms
		(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
			(not (expr_contains_membership_truth? term))))
		true)))

(define membership_projection_cost_preferred? (lambda (candidate_rows driver_rows)
	(if (or (nil? candidate_rows) (nil? driver_rows))
		false
		/* Projection has fixed setup and per-key projection costs. Driver probing
		uses the ordinary scan/autoindex path. These units are deliberately
		simple and shared by every canonical membership shape; syntax recognition
		must never select the physical representation. */
		(< (+ 16 (* candidate_rows 4)) driver_rows))))

(define membership_truth_projection_preferred? (lambda (block stage _guarded_alternative)
	(begin
		(define input (gs_input stage))
		/* Projection from the canonical group cache currently guarantees a
		direct key map only for base-table membership stages. Derived inputs keep
		the same primitive and use indexed existence probing until their cache-key
		lineage is represented explicitly; do not add a shape-specific fallback. */
		(define base_sources (filter (qb_sources block) source_is_base_table?))
		(if (or (not (source_is_base_table? input)) (not (single_source? base_sources)))
			false
			(begin
				(define candidate_estimate (planner_source_filter_estimate input
					(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)
					512))
				(define candidate_rows (membership_estimated_work_rows candidate_estimate
					(planner_source_row_count input)))
				(define driver (reduce (qb_sources block) (lambda (found src)
					(if (not (nil? found)) found (if (source_is_base_table? src) src nil))) nil))
				(define driver_total_rows (if (nil? driver) nil (planner_source_row_count driver)))
				(define driver_estimate (if (nil? driver)
					nil
					(planner_source_filter_estimate driver
						(membership_driver_filter (qb_where block)) 512)))
				(define driver_rows (membership_estimated_work_rows driver_estimate driver_total_rows))
				(membership_projection_cost_preferred? candidate_rows driver_rows)))))))

(define choose_membership_truth_items (lambda (block items guarded_alternative)
	(match items
		(cons item rest) (begin
			(define chosen (choose_membership_truth_expr_using block item guarded_alternative))
			(define tail (choose_membership_truth_items block rest guarded_alternative))
			(list
				(cons (nth chosen 0) (nth tail 0))
				(merge_unique (list (nth chosen 1) (nth tail 1)))))
		_ (list '() '()))))

(define choose_membership_truth_expr_using (lambda (block expr guarded_alternative)
	(begin
		(define parts (membership_truth_parts expr))
		(if (not (nil? parts))
			(begin
				(define stage (membership_truth_stage (qb_stages block) (qb_sources block) (nth parts 1)))
				(if (and (not (nil? stage))
					(membership_truth_projection_preferred? block stage guarded_alternative))
					(list (driver_membership_probe_expr stage (nth parts 0)) (list (nth parts 1)))
					(list (expand_in_membership_truth_expr (nth parts 0) (nth parts 1) (nth parts 2)) '())))
			(if (and (list? expr)
				(or
					(or (equal? (car expr) (quote and)) (equal? (car expr) (symbol "and")))
					(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or")))))
				(begin
					(define below_or (or guarded_alternative
						(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or")))))
					(define chosen (choose_membership_truth_items block (cdr expr) below_or))
					(list (cons (car expr) (nth chosen 0)) (nth chosen 1)))
				(list expr '()))))))

(define choose_membership_truth_expr (lambda (block expr)
	(choose_membership_truth_expr_using block expr false)))

(define query_block_with_physical_membership_choices (lambda (block)
	(if (or (not (query_block_has_membership_truth_stage? block))
		(not (expr_contains_membership_truth? (qb_where block))))
		/* Keep unrelated query blocks byte-for-byte unchanged. Besides making
		this phase boundary explicit, the stage-purpose check avoids recursively
		walking large EXISTS/scalar expressions which cannot contain the
		canonical membership primitive. */
		block
		(begin
			(define chosen (choose_membership_truth_expr block (qb_where block)))
			(define removed_aliases (nth chosen 1))
			(if (empty_list? removed_aliases)
				(make_query_block
					(qb_schema block) (qb_sources block) (qb_fields block) (nth chosen 0)
					(qb_group block) (qb_having block) (qb_order block) (qb_limit block)
					(qb_offset block) (qb_hidden block) (qb_stages block) (qb_facts block))
				(make_query_block
					(qb_schema block)
					(filter (qb_sources block) (lambda (src) (not (contains? removed_aliases (source_alias src)))))
					(qb_fields block) (nth chosen 0) (qb_group block) (qb_having block)
					(qb_order block) (qb_limit block) (qb_offset block) (qb_hidden block)
					(qb_stages block)
					(qassoc_set (qb_facts block) (quote membership_plan_strategy)
						(quote projected_membership_alternatives))))))))

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
	(if (not (group_stage? stage))
		false
		(begin
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define input (gs_input stage))
			(and
				(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote exists))
				(equal? (qassoc_get (gs_facts stage) (quote presence_only) false) true)
				(not (empty_list? lookup_keys))
				(equal? (count lookup_keys) (count (gs_keys stage)))
				(or (source_is_base_table? input)
					(and (union_block? input)
						(reduce (union_branches input) (lambda (supported branch)
							(and supported (candidate_recset_branch_supported? branch)))
							true))))))))

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

(define driver_order_membership_strategy? (lambda (facts)
	(equal? (qassoc_get facts (quote membership_plan_strategy) nil) (quote driver_order_membership_probe))))

(define membership_strategy? (lambda (facts)
	(or (driver_order_membership_strategy? facts)
		(equal? (qassoc_get facts (quote membership_plan_strategy) nil) (quote candidate_keyset)))))

(define candidate_recset_branch_supported? (lambda (branch)
	(and (query_block? branch)
		(and (single_source? (qb_sources branch))
			(and (source_is_base_table? (car (qb_sources branch)))
				(and (empty_list? (qb_stages branch))
					(not (nil? (direct_column_name_for_alias
						(car (qb_sources branch))
						(query_block_first_expr branch))))))))))

(define candidate_stage_recset_supported? (lambda (stage)
	(and (group_stage? stage)
		(and (union_block? (gs_input stage))
			(reduce (union_branches (gs_input stage)) (lambda (supported branch)
				(and supported (candidate_recset_branch_supported? branch)))
				true)))))

(define broad_driver_order_membership_probe? (lambda (facts)
	(begin
		(define estimated_rows (qassoc_get facts (quote membership_candidate_estimated_rows) nil))
		(define estimate_input (qassoc_get facts (quote membership_candidate_estimate_input) nil))
		(define capped (qassoc_get facts (quote membership_candidate_estimate_capped) false))
		(define broad_by_count (and
			(and (not (nil? estimated_rows)) (and (not (nil? estimate_input)) (> estimate_input 0)))
			(>= (* estimated_rows 4) estimate_input)))
		(and
			(driver_order_membership_strategy? facts)
			(or
				capped
				(or
					broad_by_count
					(equal? (qassoc_get facts (quote membership_selectivity_class) nil) (quote broad))))))))

(define stage_consumed_by_membership_source? (lambda (stage stages sources facts)
	(if (not (membership_strategy? facts))
		false
		(begin
			(define candidate (first_candidate_source stages sources))
			(and
				(not (nil? candidate))
				(and (stage_output_relation? (source_relation candidate))
					(and (group_stage? stage)
						(and (candidate_stage_recset_supported? stage)
							(equal? (stage_output_relation_id (source_relation candidate)) (gs_id stage))))))))))

(define query_block_with_physical_membership_using (lambda (stages block)
	(if (not (membership_strategy? (qb_facts block)))
		block
		(begin
			(define sources (qb_sources block))
			(define candidate (first_candidate_source stages sources))
			(if (nil? candidate)
				block
				(begin
					(define stage (stage_by_id stages (stage_output_relation_id (source_relation candidate))))
					(if (not (candidate_stage_recset_supported? stage))
						block
						(begin
							(define probe (car (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
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
								(candidate_stage_without_source (qb_stages block) (gs_id stage))
								(join_optimizer_facts_without_aliases
									(qb_facts block) (list (source_alias candidate))))))))))))

(define negated_term? (lambda (term)
	(match term
		((symbol not) _expr) true
		((quote not) _expr) true
		_ false)))

(define exists_recset_probe_term? (lambda (alias term)
	(match term
		((symbol >) ((symbol coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((symbol >) ((quote coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((symbol >) ((symbol coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((symbol >) ((quote coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((quote >) ((symbol coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((quote >) ((quote coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((quote >) ((symbol coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		((quote >) ((quote coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) 0) 0)
		(equal? tblvar alias)
		_ false)))

(define condition_requires_exists_recset_probe? (lambda (alias condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(or found (exists_recset_probe_term? alias term))) false)))

(define condition_has_exists_recset_probe? (lambda (alias condition)
	(match condition
		(cons head tail) (or
			(exists_recset_probe_term? alias condition)
			(reduce tail (lambda (found item) (or found (condition_has_exists_recset_probe? alias item))) false))
		_ false)))

(define rewrite_required_exists_recset_probe_refs (lambda (alias expr)
	(if (exists_recset_probe_term? alias expr)
		true
		(match expr
			(cons head tail) (cons head (map tail (lambda (item) (rewrite_required_exists_recset_probe_refs alias item))))
			_ expr))))

(define rewrite_exists_recset_probe_refs (lambda (alias stage probe expr)
	(if (exists_recset_probe_term? alias expr)
		(driver_membership_probe_expr stage probe)
		(match expr
			(cons head tail) (cons head (map tail (lambda (item) (rewrite_exists_recset_probe_refs alias stage probe item))))
			_ expr))))

(define first_driver_lookup_key (lambda (stage sources)
	(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (found key)
		(if (not (nil? found))
			found
			(reduce (coalesceNil sources '()) (lambda (resolved src)
				(if (or (not (nil? resolved)) (nil? (direct_column_name_for_alias src key)))
					resolved
					key))
				nil)))
		nil)))

(define first_exists_recset_source (lambda (stages sources default_alias condition)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (and (exists_recset_stage_output_source? stages src)
				(and (not (nil? (first_driver_lookup_key
					(stage_by_id stages (stage_output_relation_id (source_relation src)))
					(without_source_alias sources (source_alias src)))))
					(condition_has_exists_recset_probe? (source_alias src) condition)))
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

(define reorder_query_block_with_candidate_strategy_using (lambda (stage_catalog block)
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
				(map (qb_stages block) (lambda (stage) (join_reorder_stage_using stage_catalog stage)))
				(qb_facts block))
			(begin
				(define sources (qb_sources block))
				(define candidate (first_candidate_source (qb_stages block) sources))
				(if (nil? candidate)
					(begin
						(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
						(define exists_src (first_exists_recset_source (qb_stages block) sources default_alias (qb_where block)))
						(if (nil? exists_src)
							(hybrid_reorder_query_block_using stage_catalog block)
							(begin
								(define stage (stage_by_id (qb_stages block) (stage_output_relation_id (source_relation exists_src))))
								(define stage_id (gs_id stage))
								(define probe_sources (list exists_src))
								(define probe_required (condition_requires_exists_recset_probe?
									(source_alias exists_src) (qb_where block)))
								(define rewritten_sources (rewrite_scalar_first_probe_sources_using
									(qb_stages block) sources probe_sources default_alias))
								(define driver_sources (without_source_alias rewritten_sources (source_alias exists_src)))
								(define probe (first_driver_lookup_key stage driver_sources))
								(define rewrite_consumer (lambda (expr)
									(if probe_required
										(rewrite_required_exists_recset_probe_refs (source_alias exists_src) expr)
										(rewrite_scalar_first_probe_expr
											(qb_stages block) probe_sources default_alias expr))))
								(define base_where (rewrite_exists_recset_probe_refs
									(source_alias exists_src)
									stage
									probe
									(qb_where block)))
								(hybrid_reorder_query_block_using stage_catalog
									(make_query_block
										(qb_schema block)
										driver_sources
										(rewrite_consumer (qb_fields block))
										base_where
										(rewrite_consumer (qb_group block))
										(rewrite_consumer (qb_having block))
										(rewrite_consumer (qb_order block))
										(qb_limit block)
										(qb_offset block)
										(rewrite_consumer (qb_hidden block))
										(candidate_stage_without_source (qb_stages block) stage_id)
										(merge (list
											(query_block_reorder_telemetry block)
											(list (list (quote exists_reorder_strategy) (quote project_driver)))
											(qb_facts block))))))))
					(begin
						(define stage (stage_by_id (qb_stages block) (stage_output_relation_id (source_relation candidate))))
						(define candidate_telemetry (candidate_reorder_telemetry stage sources block))
						(define strategy (candidate_reorder_strategy candidate_telemetry))
						(define costed_telemetry (qassoc_set candidate_telemetry
							(quote membership_cost_reason)
							(if (equal? strategy (quote candidate_keyset))
								(quote projected_membership_cost)
								(quote indexed_driver_probe_cost))))
						(define facts (merge (list
							(query_block_reorder_telemetry block)
							(cons
								(list (quote membership_plan_strategy) strategy)
								costed_telemetry))))
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
								(map (qb_stages block) (lambda (stage) (join_reorder_stage_using stage_catalog stage)))
								(qb_facts block))
							facts))))))))

(define reorder_query_block_with_candidate_strategy (lambda (block)
	(reorder_query_block_with_candidate_strategy_using (qb_stages block) block)))

(define join_reorder_node_using (lambda (stage_catalog node)
	(if (query_block? node)
		(reorder_query_block_with_candidate_strategy_using stage_catalog node)
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) (lambda (branch) (join_reorder_node_using stage_catalog branch)))
				(union_order node)
				(union_limit node)
				(union_offset node)
				(union_facts node))
			node))))

(define join_reorder_node (lambda (node)
	(join_reorder_node_using (if (query_block? node) (qb_stages node) '()) node)))

(define join_reorder_stage_using (lambda (stage_catalog stage)
	(if (group_stage? stage)
		(group_stage_with_reorder_facts
			(make_group_stage
				(gs_id stage)
				(join_reorder_node_using stage_catalog (gs_input stage))
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

(define join_reorder_stage (lambda (stage)
	(join_reorder_stage_using (list stage) stage)))

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

/* Aggregate column names are derived from their expressions. A decorrelation
rewrite may therefore rename a single-column stage output after a consumer was
created. Canonicalize that unambiguous interface before physical lowering. */
(define stage_output_stage_index (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (index stage)
		(if (group_stage? stage) (set_assoc index (gs_id stage) stage) index)) '())))

(define stage_output_single_aggregate_columns (lambda (stage_index sources)
	(filter (map (coalesceNil sources '()) (lambda (src)
		(begin
			(define relation (source_relation src))
			(define stage (if (stage_output_relation? relation)
				(get_assoc stage_index (stage_output_relation_id relation))
				nil))
			(if (and (group_stage? stage) (equal? (count (gs_aggregates stage)) 1))
				(list (source_alias src) (quote aggregate-column)
					(aggregate_col_name (car (gs_aggregates stage))))
				nil))))
		(lambda (entry) (not (nil? entry))))))

(define aggregate_column_name? (lambda (col)
	(and (string? col) (and (>= (strlen col) 4) (equal? (substr col 0 4) "agg_")))))

(define canonicalize_stage_output_stage (lambda (stage_index stage)
	(if (not (group_stage? stage))
		stage
		(begin
			(define input (gs_input stage))
			(define sources (if (query_block? input) (qb_sources input) (list input)))
			(rewrite_stage_graph_stage
				(stage_output_single_aggregate_columns stage_index sources)
				'() false stage)))))

(define canonicalize_stage_output_stages_acc (lambda (remaining rewritten stage_index)
	(match (coalesceNil remaining '())
		(cons stage rest) (begin
			(define canonical (canonicalize_stage_output_stage stage_index stage))
			(canonicalize_stage_output_stages_acc rest
				(merge rewritten (list canonical))
				(if (group_stage? canonical)
					(set_assoc stage_index (gs_id canonical) canonical)
					stage_index)))
		_ (list rewritten stage_index))))

(define canonicalize_stage_output_stages (lambda (stages)
	(nth (canonicalize_stage_output_stages_acc
		stages '() (stage_output_stage_index stages)) 0)))

(define canonicalize_stage_output_interfaces (lambda (ir)
	(begin
		(define root (ir_root ir))
		(if (not (query_block? root))
			ir
			(begin
				(define canonical (canonicalize_stage_output_stages_acc
					(qb_stages root) '() (stage_output_stage_index (qb_stages root))))
				(define stages (nth canonical 0))
				(define block_with_stages (make_query_block
					(qb_schema root) (qb_sources root) (qb_fields root) (qb_where root)
					(qb_group root) (qb_having root) (qb_order root) (qb_limit root)
					(qb_offset root) (qb_hidden root) stages (qb_facts root)))
				(define block (rewrite_stage_graph_expr
					(stage_output_single_aggregate_columns (nth canonical 1) (qb_sources root))
					'() block_with_stages))
				(make_ir (ir_kind ir) block stages (ir_context_of ir) (ir_return ir)))))))

(define normalize_stage_dependencies (lambda (ir)
	(begin
		(define root_result (normalize_stage_dependencies_node (ir_root ir)))
		(canonicalize_stage_output_interfaces
			(merge_compatible_stage_output_left_joins_ir (make_ir
				(ir_kind ir)
				(nth root_result 0)
				(if (query_block? (nth root_result 0)) (qb_stages (nth root_result 0)) (nth root_result 1))
				(ir_context_of ir)
				(ir_return ir)))))))

/* Scalar stages are merged after decorrelation. Build their signatures bottom-up
so generated aliases and dependency IDs do not hide equivalent stage graphs. */
(define stage_semantic_expr_aliases (lambda (expr)
	(match expr
		((symbol get_column) tblvar _ignorecase _col _col_ignorecase) (if (nil? tblvar) '() (list tblvar))
		((quote get_column) tblvar _ignorecase _col _col_ignorecase) (if (nil? tblvar) '() (list tblvar))
		(cons head tail) (merge_unique (list
			(stage_semantic_expr_aliases head)
			(map tail stage_semantic_expr_aliases)))
		_ '())))

(define stage_semantic_alias_entries (lambda (aliases prefix)
	(map (produceN (count aliases)) (lambda (i)
		(list (nth aliases i) (concat prefix i))))))

(define stage_semantic_input_sources (lambda (input)
	(if (query_block? input)
		(qb_sources input)
		(if (source_is_base_table? input) (list input) '()))))

(define stage_semantic_alias_map (lambda (stage)
	(begin
		(define local_aliases (source_aliases (stage_semantic_input_sources (gs_input stage))))
		(define referenced_aliases (merge_unique (list
			(stage_semantic_expr_aliases (gs_input stage))
			(stage_semantic_expr_aliases (gs_domain stage))
			(stage_semantic_expr_aliases (gs_keys stage))
			(stage_semantic_expr_aliases (gs_aggregates stage))
			(stage_semantic_expr_aliases (qassoc_get (gs_facts stage) (quote condition) true)))))
		(define outer_aliases (filter referenced_aliases (lambda (alias) (not (contains? local_aliases alias)))))
		(merge (list
			(stage_semantic_alias_entries local_aliases "__stage_local_")
			(stage_semantic_alias_entries outer_aliases "__stage_outer_"))))))

(define stage_semantic_rewrite_expr (lambda (alias_map signatures expr)
	(match expr
		((symbol get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(stage_merge_lookup alias_map tblvar tblvar)
			ignorecase col col_ignorecase)
		((quote get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(stage_merge_lookup alias_map tblvar tblvar)
			ignorecase col col_ignorecase)
		((symbol stage-output) stage_id)
		(list (quote stage-output) (coalesceNil (get_assoc signatures stage_id) stage_id))
		((quote stage-output) stage_id)
		(list (quote stage-output) (coalesceNil (get_assoc signatures stage_id) stage_id))
		(cons head tail) (cons (stage_semantic_rewrite_expr alias_map signatures head)
			(map tail (lambda (item) (stage_semantic_rewrite_expr alias_map signatures item))))
		_ expr)))

(define stage_semantic_rewrite_source (lambda (alias_map signatures src)
	(match src
		'(alias schema relation outer join)
		(list
			(stage_merge_lookup alias_map alias alias)
			schema
			(stage_semantic_rewrite_expr alias_map signatures relation)
			outer
			(stage_semantic_rewrite_expr alias_map signatures join))
		_ src)))

(define stage_semantic_canonical_node (lambda (alias_map signatures node)
	(if (query_block? node)
		(make_query_block
			(qb_schema node)
			(map (qb_sources node) (lambda (src) (stage_semantic_rewrite_source alias_map signatures src)))
			(stage_semantic_rewrite_expr alias_map signatures (qb_fields node))
			(stage_semantic_rewrite_expr alias_map signatures (qb_where node))
			(stage_semantic_rewrite_expr alias_map signatures (qb_group node))
			(stage_semantic_rewrite_expr alias_map signatures (qb_having node))
			(stage_semantic_rewrite_expr alias_map signatures (qb_order node))
			(qb_limit node)
			(qb_offset node)
			(stage_semantic_rewrite_expr alias_map signatures (qb_hidden node))
			'()
			'())
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) (lambda (branch) (stage_semantic_canonical_node alias_map signatures branch)))
				(stage_semantic_rewrite_expr alias_map signatures (union_order node))
				(union_limit node)
				(union_offset node)
				'())
			(stage_semantic_rewrite_source alias_map signatures node)))))

(define stage_semantic_aggregate_shape (lambda (alias_map signatures ag)
	(match ag
		'(((symbol scalar_order_value) _value_expr order_exprs dirs offset_value) reduce neutral)
		(list (quote scalar_order_value)
			(stage_semantic_rewrite_expr alias_map signatures order_exprs)
			dirs offset_value reduce neutral)
		'(((quote scalar_order_value) _value_expr order_exprs dirs offset_value) reduce neutral)
		(list (quote scalar_order_value)
			(stage_semantic_rewrite_expr alias_map signatures order_exprs)
			dirs offset_value reduce neutral)
		'(_value_expr reduce neutral) (list (quote aggregate) reduce neutral)
		_ (stage_semantic_rewrite_expr alias_map signatures ag))))

(define stage_semantic_facts (lambda (alias_map signatures facts)
	(map (list
		(quote purpose)
		(quote presence_only)
		(quote max_needed_per_domain)
		(quote preserve_empty_domain)
		(quote null_semantics)
		(quote cardinality_mode)
		(quote partition_by)
		(quote physical_max_rows)
		(quote on_overflow))
		(lambda (key) (list key (stage_semantic_rewrite_expr alias_map signatures (qassoc_get facts key nil)))))))

(define stage_semantic_signature (lambda (signatures stage)
	(if (not (group_stage? stage))
		(logical_stage_key stage)
		(begin
			(define alias_map (stage_semantic_alias_map stage))
			(define payload (list
				(stage_semantic_canonical_node alias_map signatures (gs_input stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_domain stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_keys stage))
				(map (gs_aggregates stage) (lambda (ag) (stage_semantic_aggregate_shape alias_map signatures ag)))
				(stage_semantic_rewrite_expr alias_map signatures (qassoc_get (gs_facts stage) (quote condition) true))
				(stage_semantic_rewrite_expr alias_map signatures (gs_having stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_output stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_order stage))
				(gs_limit stage)
				(gs_offset stage)
				(stage_semantic_facts alias_map signatures (gs_facts stage))))
			(concat "stage-semantic:" (fnv_hash (serialize payload)))))))

(define stage_semantic_signature_index (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (index stage)
		(set_assoc index (gs_id stage) (stage_semantic_signature index stage)))
		'())))

(define stage_output_left_join_stage_key (lambda (signature_index stage)
	(if (not (and (group_stage? stage)
		(and (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single))
			(equal? (count (gs_aggregates stage)) 1))))
		nil
		(get_assoc signature_index (gs_id stage)))))

(define normalize_stage_output_left_join_expr (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(if (equal? tblvar alias) "__stage_output_join__" tblvar)
			ignorecase
			col
			col_ignorecase)
		((quote get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(if (equal? tblvar alias) "__stage_output_join__" tblvar)
			ignorecase
			col
			col_ignorecase)
		(cons head tail) (cons (normalize_stage_output_left_join_expr alias head)
			(map tail (lambda (item) (normalize_stage_output_left_join_expr alias item))))
		_ expr)))

(define stage_output_left_join_key (lambda (stages signature_index src)
	(begin
		(define relation (source_relation src))
		(if (not (and (source_outer? src) (stage_output_relation? relation)))
			nil
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id relation)))
				(define stage_key (stage_output_left_join_stage_key signature_index stage))
				(if (nil? stage_key)
					nil
					(concat "stage-output-left-join:" (fnv_hash (serialize (list
						stage_key
						(source_schema src)
						(normalize_stage_output_left_join_expr (source_alias src) (source_join_expr src))))))))))))

(define stage_output_left_join_stage_with_aggregates (lambda (stage ags)
	(make_group_stage
		(gs_id stage)
		(gs_input stage)
		(gs_domain stage)
		(gs_keys stage)
		(dedupe_aggregates_by_col ags)
		(gs_having stage)
		(gs_output stage)
		(gs_order stage)
		(gs_limit stage)
		(gs_offset stage)
		(gs_facts stage))))

(define stage_output_left_join_entry_key (lambda (entry) (nth entry 0)))
(define stage_output_left_join_entry_source (lambda (entry) (nth entry 1)))
(define stage_output_left_join_entry_stage (lambda (entry) (nth entry 2)))
(define stage_output_left_join_entry_ids (lambda (entry) (nth entry 3)))
(define stage_output_left_join_entry_aliases (lambda (entry) (nth entry 4)))
(define stage_output_left_join_entry_ags (lambda (entry) (nth entry 5)))

(define stage_output_left_join_entry_for_source (lambda (key src stage)
	(list key src stage (list (gs_id stage)) (list (source_alias src)) (gs_aggregates stage))))

(define stage_output_left_join_aligned_aggregates (lambda (target stage)
	(begin
		(define target_sources (stage_semantic_input_sources (gs_input target)))
		(define source_sources (stage_semantic_input_sources (gs_input stage)))
		(define alias_map (map (produceN (count source_sources)) (lambda (i)
			(list (source_alias (nth source_sources i)) (source_alias (nth target_sources i))))))
		(define id_map (map (produceN (count source_sources)) (lambda (i)
			(list
				(stage_output_relation_id (source_relation (nth source_sources i)))
				(stage_output_relation_id (source_relation (nth target_sources i)))))))
		(rewrite_stage_graph_expr alias_map id_map (gs_aggregates stage)))))

(define stage_output_left_join_entry_add_source (lambda (entry src stage)
	(list
		(stage_output_left_join_entry_key entry)
		(stage_output_left_join_entry_source entry)
		(stage_output_left_join_entry_stage entry)
		(merge_unique (list (stage_output_left_join_entry_ids entry) (list (gs_id stage))))
		(merge_unique (list (stage_output_left_join_entry_aliases entry) (list (source_alias src))))
		(dedupe_aggregates_by_col (merge (list
			(stage_output_left_join_entry_ags entry)
			(stage_output_left_join_aligned_aggregates (stage_output_left_join_entry_stage entry) stage)))))))

(define stage_output_left_join_column_map_for_stage (lambda (target_alias target stage_alias stage)
	(begin
		(define source_ags (gs_aggregates stage))
		(define aligned_ags (stage_output_left_join_aligned_aggregates target stage))
		(map (produceN (count source_ags)) (lambda (i)
			(list
				stage_alias
				(aggregate_col_name (nth source_ags i))
				target_alias
				(aggregate_col_name (nth aligned_ags i))))))))

(define stage_output_left_join_column_maps_for_entry (lambda (stages entry)
	(begin
		(define target (stage_output_left_join_entry_stage entry))
		(define target_alias (source_alias (stage_output_left_join_entry_source entry)))
		(define ids (stage_output_left_join_entry_ids entry))
		(define aliases (stage_output_left_join_entry_aliases entry))
		(merge (map (produceN (count ids)) (lambda (i)
			(stage_output_left_join_column_map_for_stage
				target_alias target (nth aliases i) (stage_by_id stages (nth ids i)))))))))

(define stage_output_left_join_column_mapping (lambda (mappings alias col)
	(reduce (coalesceNil mappings '()) (lambda (found mapping)
		(if (not (nil? found))
			found
			(match mapping
				'(old_alias old_col new_alias new_col)
				(if (and (equal? alias old_alias) (equal? col old_col))
					(list new_alias new_col)
					nil)
				_ nil)))
		nil)))

(define rewrite_stage_output_left_join_columns (lambda (mappings expr)
	(match expr
		((symbol get_column) tblvar ignorecase col col_ignorecase) (begin
			(define mapped (stage_output_left_join_column_mapping mappings tblvar col))
			(if (nil? mapped)
				expr
				(list (quote get_column) (nth mapped 0) ignorecase (nth mapped 1) col_ignorecase)))
		((quote get_column) tblvar ignorecase col col_ignorecase) (begin
			(define mapped (stage_output_left_join_column_mapping mappings tblvar col))
			(if (nil? mapped)
				expr
				(list (quote get_column) (nth mapped 0) ignorecase (nth mapped 1) col_ignorecase)))
		(cons head tail) (cons (rewrite_stage_output_left_join_columns mappings head)
			(map tail (lambda (item) (rewrite_stage_output_left_join_columns mappings item))))
		_ expr)))

(define rewrite_stage_output_left_join_source_columns (lambda (mappings src)
	(source_with_join_expr src (rewrite_stage_output_left_join_columns mappings (source_join_expr src)))))

(define stage_output_left_join_upsert_entry (lambda (stages signature_index entries src)
	(begin
		(define key (stage_output_left_join_key stages signature_index src))
		(if (nil? key)
			entries
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(define matched (reduce entries (lambda (found entry)
					(or found (equal? (stage_output_left_join_entry_key entry) key)))
					false))
				(if matched
					(map entries (lambda (entry)
						(if (equal? (stage_output_left_join_entry_key entry) key)
							(stage_output_left_join_entry_add_source entry src stage)
							entry)))
					(merge entries (list (stage_output_left_join_entry_for_source key src stage)))))))))

(define stage_output_left_join_entries (lambda (stages signature_index sources)
	(reduce (coalesceNil sources '()) (lambda (entries src)
		(stage_output_left_join_upsert_entry stages signature_index entries src))
		'())))

(define stage_output_left_join_entries_have_duplicates? (lambda (entries)
	(reduce (coalesceNil entries '()) (lambda (found entry)
		(or found (> (count (stage_output_left_join_entry_ids entry)) 1)))
		false)))

(define stage_output_left_join_id_map_for_entry (lambda (entry)
	(begin
		(define primary (gs_id (stage_output_left_join_entry_stage entry)))
		(map (stage_output_left_join_entry_ids entry) (lambda (id) (list id primary))))))

(define stage_output_left_join_dependency_id_map (lambda (target stage)
	(begin
		(define target_sources (stage_semantic_input_sources (gs_input target)))
		(define source_sources (stage_semantic_input_sources (gs_input stage)))
		(filter (map (produceN (count source_sources)) (lambda (i)
			(begin
				(define source_id (stage_output_relation_id (source_relation (nth source_sources i))))
				(define target_id (stage_output_relation_id (source_relation (nth target_sources i))))
				(if (or (nil? source_id) (nil? target_id)) nil (list source_id target_id)))))
			(lambda (mapping) (not (nil? mapping)))))))

(define stage_output_left_join_dependency_id_maps_for_entry (lambda (stages entry)
	(begin
		(define target (stage_output_left_join_entry_stage entry))
		(merge (map (stage_output_left_join_entry_ids entry) (lambda (id)
			(stage_output_left_join_dependency_id_map target (stage_by_id stages id))))))))

(define stage_output_left_join_alias_map_for_entry (lambda (entry)
	(begin
		(define primary (source_alias (stage_output_left_join_entry_source entry)))
		(map (stage_output_left_join_entry_aliases entry) (lambda (alias) (list alias primary))))))

(define stage_output_left_join_candidate_ids (lambda (entries)
	(merge_unique (map (coalesceNil entries '()) stage_output_left_join_entry_ids))))

(define stage_output_left_join_removed_dependency_ids (lambda (dependency_id_map)
	(filter (map (coalesceNil dependency_id_map '()) (lambda (mapping)
		(match mapping
			'(old_id new_id) (if (equal? old_id new_id) nil old_id)
			_ nil)))
		(lambda (id) (not (nil? id))))))

(define stage_merge_lookup (lambda (mapping key default)
	(if (empty_list? mapping)
		default
		(coalesceNil
			(reduce mapping (lambda (found entry)
				(if (not (nil? found))
					found
					(match entry
						'(k v) (if (equal? k key) v nil)
						_ nil)))
				nil)
			default))))

(define stage_column_merge_lookup (lambda (mapping alias col default)
	(if (empty_list? mapping)
		default
		(coalesceNil
			(reduce mapping (lambda (found entry)
				(if (not (nil? found))
					found
					(match entry
						'(entry_alias (symbol aggregate-column) replacement)
						(if (and (equal? entry_alias alias) (aggregate_column_name? col))
							replacement
							nil)
						'(entry_alias entry_col replacement)
						(if (and (equal? entry_alias alias) (equal? entry_col col))
							replacement
							nil)
						_ nil)))
				nil)
			default))))

(define rewrite_stage_graph_probe_column (lambda (alias_map id_map stage requested_col)
	(begin
		(define ag (scalar_first_probe_aggregate stage requested_col))
		(if (nil? ag)
			requested_col
			(aggregate_col_name
				(rewrite_stage_graph_expr alias_map id_map ag))))))

(define rewrite_stage_graph_expr (lambda (alias_map id_map expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col stages)
		(list (quote scalar_first_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col)
			(rewrite_stage_graph_stages alias_map id_map stages))
		((quote scalar_first_probe) stage requested_col stages)
		(list (quote scalar_first_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col)
			(rewrite_stage_graph_stages alias_map id_map stages))
		((symbol scalar_first_probe) stage requested_col)
		(list (quote scalar_first_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col))
		((quote scalar_first_probe) stage requested_col)
		(list (quote scalar_first_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col))
		((symbol scalar_aggregate_probe) stage requested_col)
		(list (quote scalar_aggregate_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col))
		((quote scalar_aggregate_probe) stage requested_col)
		(list (quote scalar_aggregate_probe)
			(rewrite_stage_graph_stage alias_map id_map true stage)
			(rewrite_stage_graph_probe_column alias_map id_map stage requested_col))
		((symbol get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(stage_merge_lookup alias_map tblvar tblvar)
			ignorecase
			(stage_column_merge_lookup alias_map tblvar col col)
			col_ignorecase)
		((quote get_column) tblvar ignorecase col col_ignorecase)
		(list (quote get_column)
			(stage_merge_lookup alias_map tblvar tblvar)
			ignorecase
			(stage_column_merge_lookup alias_map tblvar col col)
			col_ignorecase)
		((symbol stage-output) stage_id)
		(list (quote stage-output) (stage_merge_lookup id_map stage_id stage_id))
		((quote stage-output) stage_id)
		(list (quote stage-output) (stage_merge_lookup id_map stage_id stage_id))
		(cons head tail) (cons (rewrite_stage_graph_expr alias_map id_map head)
			(map tail (lambda (item) (rewrite_stage_graph_expr alias_map id_map item))))
		_ expr)))

(define rewrite_stage_graph_source (lambda (alias_map id_map src)
	(match src
		'(alias schema relation outer join)
		(list
			(stage_merge_lookup alias_map alias alias)
			schema
			(rewrite_stage_graph_expr alias_map id_map relation)
			outer
			(rewrite_stage_graph_expr alias_map id_map join))
		_ src)))

(define stage_id_in_list? (lambda (ids stage_id)
	(reduce (coalesceNil ids '()) (lambda (found id)
		(or found (equal? id stage_id)))
		false)))

(define stage_not_in_id_list? (lambda (ids stage)
	(not (and (group_stage? stage) (stage_id_in_list? ids (gs_id stage))))))

(define merge_compatible_stage_output_left_joins_block (lambda (block)
	(if (or (empty_list? (qb_stages block)) (empty_list? (qb_sources block)))
		block
		(begin
			(define signature_index (stage_semantic_signature_index (qb_stages block)))
			(define entries (stage_output_left_join_entries (qb_stages block) signature_index (qb_sources block)))
			(if (not (stage_output_left_join_entries_have_duplicates? entries))
				block
				(begin
					(define dependency_id_map (merge (map entries (lambda (entry)
						(stage_output_left_join_dependency_id_maps_for_entry (qb_stages block) entry)))))
					(define id_map (merge (list
						(merge (map entries stage_output_left_join_id_map_for_entry))
						dependency_id_map)))
					(define alias_map (merge (map entries stage_output_left_join_alias_map_for_entry)))
					(define column_maps (merge (map entries (lambda (entry)
						(stage_output_left_join_column_maps_for_entry (qb_stages block) entry)))))
					(define candidate_ids (merge_unique (list
						(stage_output_left_join_candidate_ids entries)
						(stage_output_left_join_removed_dependency_ids dependency_id_map))))
					(define untouched_stages (filter (qb_stages block) (lambda (stage)
						(stage_not_in_id_list? candidate_ids stage))))
					(define merged_stages (map entries (lambda (entry)
						(rewrite_stage_graph_stage alias_map id_map false
							(stage_output_left_join_stage_with_aggregates
								(stage_output_left_join_entry_stage entry)
								(stage_output_left_join_entry_ags entry))))))
					(make_query_block
						(qb_schema block)
						(merge_unique (list (map (qb_sources block) (lambda (src)
							(rewrite_stage_graph_source alias_map id_map
								(rewrite_stage_output_left_join_source_columns column_maps src))))))
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_fields block)))
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_where block)))
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_group block)))
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_having block)))
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_order block)))
						(qb_limit block)
						(qb_offset block)
						(rewrite_stage_graph_expr alias_map id_map (rewrite_stage_output_left_join_columns column_maps (qb_hidden block)))
						(merge_unique (list
							(map untouched_stages (lambda (stage) (rewrite_stage_graph_stage '() id_map false stage)))
							merged_stages))
						(rewrite_stage_graph_expr alias_map id_map (qb_facts block)))))))))

(define merge_compatible_stage_output_left_joins_node (lambda (node)
	(if (query_block? node)
		(merge_compatible_stage_output_left_joins_block node)
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) merge_compatible_stage_output_left_joins_node)
				(union_order node)
				(union_limit node)
				(union_offset node)
				(union_facts node))
			node))))

(define merge_compatible_stage_output_left_joins_ir (lambda (ir)
	(begin
		(define root (merge_compatible_stage_output_left_joins_node (ir_root ir)))
		(make_ir
			(ir_kind ir)
			root
			(if (query_block? root) (qb_stages root) (ir_stages ir))
			(ir_context_of ir)
			(ir_return ir)))))

(define group_stage_cache_owner_key (lambda (stage)
	(if (group_stage? stage)
		(concat (group_stage_cache_schema stage) "\n" (group_stage_cache_relation stage))
		(concat "stage\n" (logical_stage_key stage)))))

(define group_stage_with_initializer_owner (lambda (stage owner)
	(if (not (group_stage? stage))
		stage
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
			(qassoc_set (gs_facts stage) (quote keytable_initializer_owner) owner)))))

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
	(begin
		(define stage_catalog (ir_stages ir))
		(make_ir
			(ir_kind ir)
			(join_reorder_node_using stage_catalog (ir_root ir))
			(map stage_catalog (lambda (stage) (join_reorder_stage_using stage_catalog stage)))
			(ir_context_of ir)
			(ir_return ir)))))

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

(define source_column_name (lambda (src col col_ignorecase)
	(if (not (string? col))
		col
		(if (not (source_is_base_table? src))
			nil
			(resolve_column_name
				(source_schema src)
				(source_relation src)
				col
				col_ignorecase)))))

(define source_has_column? (lambda (src col col_ignorecase)
	(not (nil? (source_column_name src col col_ignorecase)))))

(define source_for_unqualified_column (lambda (sources default_alias col col_ignorecase)
	(begin
		(define matches (filter (coalesceNil sources '()) (lambda (src)
			(source_has_column? src col col_ignorecase))))
		(if (equal? (count matches) 1)
			(car matches)
			(if (nil? default_alias)
				nil
				(begin
					(define default_src (source_for_alias matches default_alias nil false))
					(if (nil? default_src) nil default_src)))))))

(define resolve_physical_column_name (lambda (src col col_ignorecase)
	(if (not (string? col))
		col
		(if (not (source_is_base_table? src))
			col
			(coalesceNil (source_column_name src col true) col)))))

(define extract_columns_for_alias (lambda (src expr)
	(match expr
		((symbol driver_membership_probe) _stage probe)
		(extract_columns_for_alias src probe)
		((quote driver_membership_probe) _stage probe)
		(extract_columns_for_alias src probe)
		((symbol dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_alias src probe)
		((quote dml_driver_membership_probe) _fallback_schema _stage probe)
		(extract_columns_for_alias src probe)
		((symbol scalar_first_probe) stage _requested_col)
		(merge_unique (map (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (key)
			(extract_columns_for_alias src key))))
		((symbol scalar_first_probe) stage _requested_col _stages)
		(merge_unique (map (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (key)
			(extract_columns_for_alias src key))))
		((quote scalar_first_probe) stage requested_col)
		(extract_columns_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((quote scalar_first_probe) stage requested_col _stages)
		(extract_columns_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((symbol scalar_aggregate_probe) stage _requested_col)
		(merge_unique (map (scalar_aggregate_probe_outer_exprs stage) (lambda (expr)
			(extract_columns_for_alias src expr))))
		((quote scalar_aggregate_probe) stage requested_col)
		(extract_columns_for_alias src (list (quote scalar_aggregate_probe) stage requested_col))
		((symbol scalar_cardinality_probe) stage _requested_col)
		(merge_unique (map (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (key)
			(extract_columns_for_alias src key))))
		((quote scalar_cardinality_probe) stage requested_col)
		(extract_columns_for_alias src (list (quote scalar_cardinality_probe) stage requested_col))
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase) (list (resolve_physical_column_name src col col_ignorecase)) '())
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase) (list (resolve_physical_column_name src col col_ignorecase)) '())
		(cons head tail) (merge_unique (map tail (lambda (item) (extract_columns_for_alias src item))))
		_ '())))

(define lower_column_expr_for_alias_in_context (lambda (src expr probe_work_rows)
	(match expr
		((symbol driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr (list src) (source_alias src) stage probe)
		((quote driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr (list src) (source_alias src) stage probe)
		((symbol dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((quote dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col (list stage) probe_work_rows)
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col stages probe_work_rows)
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col (list stage) probe_work_rows)
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col stages probe_work_rows)
		((symbol scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr (list src) (source_alias src) stage requested_col)
		((quote scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr (list src) (source_alias src) stage requested_col)
		((symbol scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr (list src) (source_alias src) stage requested_col)
		((quote scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr (list src) (source_alias src) stage requested_col)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (symbol (concat (resolve_column_alias tblvar (source_alias src)) "." (resolve_physical_column_name src col col_ignorecase)))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (symbol (concat (resolve_column_alias tblvar (source_alias src)) "." (resolve_physical_column_name src col col_ignorecase)))
		(cons head tail) (cons head (map tail (lambda (item)
			(lower_column_expr_for_alias_in_context src item probe_work_rows))))
		_ expr)))

(define lower_column_expr_for_alias (lambda (src expr)
	(lower_column_expr_for_alias_in_context src expr nil)))

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

(define scalar_first_probe_query_key_terms (lambda (keys lookup_keys)
	(map (produceN (count keys)) (lambda (i)
		(list (quote equal??) (nth keys i) (nth lookup_keys i))))))

(define expr_source_alias (lambda (expr)
	(match expr
		((symbol get_column) tblvar _ignorecase _col _col_ignorecase) tblvar
		((quote get_column) tblvar _ignorecase _col _col_ignorecase) tblvar
		_ nil)))

(define source_alias_exists? (lambda (sources alias)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(or found (equal? (source_alias src) alias)))
		false)))

(define query_key_term_alias (lambda (sources key_expr)
	(begin
		(define alias (expr_source_alias key_expr))
		(if (and (not (nil? alias)) (source_alias_exists? sources alias))
			alias
			nil))))

(define query_sources_with_key_terms (lambda (sources terms)
	(map (coalesceNil sources '()) (lambda (src)
		(begin
			(define alias (source_alias src))
			(define source_terms (filter terms (lambda (term)
				(equal? (nth term 0) alias))))
			(list
				(source_alias src)
				(source_schema src)
				(source_relation src)
				(source_outer? src)
				(combine_where_terms
					(cons (source_join_expr src) (map source_terms (lambda (term) (nth term 1))))
					true)))))))

(define presence_bool_stage_output_expr? (lambda (expr)
	(match expr
		((symbol >) ((symbol coalesceNil) ((symbol get_column) tblvar _ignorecase col _col_ignorecase) 0) 0)
		(and (expr_refs_stage_output_alias? (list (quote get_column) tblvar false col false)) true)
		((quote >) ((quote coalesceNil) ((quote get_column) tblvar _ignorecase col _col_ignorecase) 0) 0)
		(and (expr_refs_stage_output_alias? (list (quote get_column) tblvar false col false)) true)
		((symbol >) ((quote coalesceNil) ((quote get_column) tblvar _ignorecase col _col_ignorecase) 0) 0)
		(and (expr_refs_stage_output_alias? (list (quote get_column) tblvar false col false)) true)
		((quote >) ((symbol coalesceNil) ((symbol get_column) tblvar _ignorecase col _col_ignorecase) 0) 0)
		(and (expr_refs_stage_output_alias? (list (quote get_column) tblvar false col false)) true)
		_ false)))

(define scalar_first_query_probe_direct_nested_stages (lambda (all_stages stage)
	(begin
		(define input (gs_input stage))
		(unique_stages_by_id (merge (list
			(qb_stages input)
			(stage_outputs_from_sources_using all_stages (qb_sources input))))))))

(define scalar_first_query_probe_nested_stages_using_index (lambda (direct_stages closure_index)
	(unique_stages_by_id (merge (list
		direct_stages
		(merge (map direct_stages (lambda (nested_stage)
			(get_assoc closure_index (logical_stage_key nested_stage))))))))))

(define lower_direct_scalar_query_probe (lambda (input value_expr)
	(begin
		(define sources (qb_sources input))
		(if (and (single_source? sources)
			(and (source_is_base_table? (car sources))
				(and (empty_list? (qb_stages input))
					(and (empty_list? (qb_group input))
						(and (nil? (qb_having input))
							(and (empty_list? (qb_order input))
								(and (nil? (qb_limit input)) (nil? (qb_offset input)))))))))
			(begin
				(define src (car sources))
				(define condition (combine_where (source_join_expr src) (coalesceNil (qb_where input) true)))
				(define filtercols (extract_columns_for_alias src condition))
				(define mapcols (extract_columns_for_alias src value_expr))
				(list (quote scan_order)
					'(session "__memcp_tx")
					(source_table_expr src)
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
						(list (quote optimize) (lower_column_expr_for_alias src condition)))
					(quoted_runtime_list '())
					(quoted_runtime_list '())
					0
					0
					1
					(cons (quote list) mapcols)
					(list (quote lambda)
						(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
						(lower_column_expr_for_alias src value_expr))
					(scalar_once_reduce_first)
					nil
					false))
			nil))))

(define lower_scalar_first_query_probe_expr_using (lambda (stage value_expr keys lookup_keys nested_stages prepare_stages inline_presence_stages)
	(begin
		(define input (gs_input stage))
		(define keyed_terms (map (produceN (count keys)) (lambda (i)
			(list (query_key_term_alias (qb_sources input) (nth keys i))
				(list (quote equal??) (nth keys i) (nth lookup_keys i))))))
		(define where_key_terms (map
			(filter keyed_terms (lambda (term) (nil? (nth term 0))))
			(lambda (term) (nth term 1))))
		(define keyed_input (make_query_block
			(qb_schema input)
			(query_sources_with_key_terms (qb_sources input) keyed_terms)
			(qb_fields input)
			(combine_where_terms (cons (qb_where input) where_key_terms) true)
			(qb_group input)
			(qb_having input)
			(qb_order input)
			(qb_limit input)
			(qb_offset input)
			(qb_hidden input)
			(qb_stages input)
			(qb_facts input)))
		(define inline_presence_ids (stage_id_set inline_presence_stages))
		(define inline_presence_sources (filter (qb_sources keyed_input) (lambda (src)
			(and (stage_output_relation? (source_relation src))
				(has_assoc? inline_presence_ids (stage_output_relation_id (source_relation src)))))))
		(define probe_input (if (empty_list? inline_presence_sources)
			keyed_input
			(query_block_with_presence_probe_sources_using
				inline_presence_stages inline_presence_sources keyed_input)))
		(define probe_value_expr (if (empty_list? inline_presence_sources)
			value_expr
			(begin
				(define default_alias (qassoc_get (qb_facts keyed_input) (quote default_alias)
					(if (empty_list? (qb_sources keyed_input)) nil (source_alias (car (qb_sources keyed_input))))))
				(rewrite_scalar_first_probe_expr
					inline_presence_stages inline_presence_sources default_alias value_expr))))
		(define raw_prepared_input
			(query_block_without_stages_after_eager_prepare_using nested_stages probe_input))
		(define prepared_input (if (empty_list? prepare_stages)
			(query_block_with_stage_catalog raw_prepared_input '())
			raw_prepared_input))
		(define direct_probe (if (empty_list? prepare_stages)
			(lower_direct_scalar_query_probe prepared_input probe_value_expr)
			nil))
		(define probe_expr (if (nil? direct_probe)
			(begin
				(define reduced (lower_query_block_as_dataset_reduce
					prepared_input
					(list "__value" probe_value_expr)
					(list (quote lambda) (list (quote __value)) (quote __value))
					(scalar_query_probe_reduce_first)
					(list (quote quote) scalar_query_probe_empty)
					nil))
				(list
					(list (quote lambda) (list (quote __scalar_probe_result))
						(list (quote if)
							(list (quote and)
								(list (quote symbol?) (quote __scalar_probe_result))
								(list (quote equal?) (quote __scalar_probe_result) (list (quote quote) scalar_query_probe_empty)))
							nil
							(quote __scalar_probe_result)))
					reduced))
			direct_probe))
		(if (or (not (empty_list? (qb_order input)))
			(or (not (nil? (qb_limit input))) (not (nil? (qb_offset input)))))
			(neumann_fail "build_queryplan" "scalar-first query probe cannot preserve nested ORDER/LIMIT yet")
			(begin
				(define raw_probe (if (not (empty_list? prepare_stages))
					(cons (quote !begin)
						(merge (list
							(lazy_stage_prepare_bindings nested_stages (filter nested_stages group_stage?))
							(lower_unique_stage_prepares_using nested_stages nested_stages nested_stages)
							(lower_stage_materialize_all nested_stages)
							(list probe_expr))))
					probe_expr))
				(if (presence_bool_stage_output_expr? value_expr)
					(list (quote coalesceNil) raw_probe false)
					raw_probe))))))

(define bounded_scalar_query_probe_inline_presence_stages (lambda (direct_stages probe_work_rows)
	(if (not (equal? (count direct_stages) 1))
		'()
		(filter direct_stages (lambda (nested_stage)
			(if (not (group_stage? nested_stage))
				false
				(begin
					(define stage_rows (planner_stage_input_rows (gs_input nested_stage)))
					(define cache_preferred (and (number? stage_rows)
						(and (number? probe_work_rows)
							(< (+ 16 stage_rows) (* probe_work_rows 4)))))
					(and (not cache_preferred)
						(and (equal? (qassoc_get (gs_facts nested_stage) (quote purpose) nil) (quote exists))
							(and (equal? (qassoc_get (gs_facts nested_stage) (quote presence_only) false) true)
								(and (source_is_base_table? (gs_input nested_stage))
									(not (stage_has_residual_outer_refs? nested_stage)))))))))))))

(define lower_scalar_first_query_probe_expr (lambda (all_stages stage value_expr keys lookup_keys probe_work_rows)
	(begin
		(define direct_stages (scalar_first_query_probe_direct_nested_stages all_stages stage))
		(define dependency_graph (stage_dependency_graph all_stages))
		(define closure_index (stage_dependency_closure_index_using_graph dependency_graph direct_stages))
		(define nested_stages
			(scalar_first_query_probe_nested_stages_using_index direct_stages closure_index))
		/* A bounded parent probe evaluates this subtree only for rows that survived
		root braking. Compare those expected probe calls with the dependent stage's
		input size; retain the group cache when repeated probes amortize its build. */
		(define inline_presence_stages (if (number? probe_work_rows)
			(bounded_scalar_query_probe_inline_presence_stages direct_stages probe_work_rows)
			'()))
		(define inline_ids (stage_id_set inline_presence_stages))
		(define prepare_stages (filter nested_stages (lambda (nested_stage)
			(not (has_assoc? inline_ids (gs_id nested_stage))))))
		(lower_scalar_first_query_probe_expr_using
			stage
			value_expr
			keys
			lookup_keys
			nested_stages
			prepare_stages
			inline_presence_stages))))

/* Query-input scalar probes can occur in many projected fields after their
logical stages have merged. Emit the physical probe recipe once per block and
pass correlation keys to it instead of copying the complete recipe per field. */
(define scalar_query_probe_recipe_key (lambda (stage requested_col)
	(concat "__scalar_query_probe_" (fnv_hash (concat (gs_id stage) "\n" requested_col)))))

(define scalar_query_probe_recipe_entry_add (lambda (state stage requested_col)
	(if (not (query_block? (gs_input stage)))
		state
		(begin
			(define key (scalar_query_probe_recipe_key stage requested_col))
			(if (has_assoc? (nth state 1) key)
				state
				(list
					(cons (list stage requested_col) (nth state 0))
					(set_assoc (nth state 1) key true)))))))

(define collect_scalar_query_probe_recipe_entries_acc (lambda (expr state)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(scalar_query_probe_recipe_entry_add state stage requested_col)
		((quote scalar_first_probe) stage requested_col)
		(scalar_query_probe_recipe_entry_add state stage requested_col)
		((symbol scalar_first_probe) stage requested_col _stages)
		(scalar_query_probe_recipe_entry_add state stage requested_col)
		((quote scalar_first_probe) stage requested_col _stages)
		(scalar_query_probe_recipe_entry_add state stage requested_col)
		(cons _head tail) (reduce tail (lambda (acc item)
			(collect_scalar_query_probe_recipe_entries_acc item acc)) state)
		_ state)))

(define query_block_scalar_query_probe_recipe_entries (lambda (block)
	(nth (collect_scalar_query_probe_recipe_entries_acc
		(list
			(qb_sources block)
			(qb_fields block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_hidden block))
		(list '() '())) 0)))

(define query_block_prelimit_scalar_query_probe_recipe_entries (lambda (block)
	(nth (collect_scalar_query_probe_recipe_entries_acc
		(list
			(qb_sources block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_hidden block))
		(list '() '())) 0)))

(define scalar_query_probe_recipe_keys (lambda (entries)
	(reduce entries (lambda (keys entry)
		(match entry
			'(stage requested_col) (set_assoc keys
				(scalar_query_probe_recipe_key stage requested_col) true)
			_ keys)) '())))

(define rewrite_scalar_query_probe_recipe_expr (lambda (expr recipe_keys)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(if (has_assoc? recipe_keys (scalar_query_probe_recipe_key stage requested_col))
			(cons (symbol (scalar_query_probe_recipe_key stage requested_col))
				(qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			expr)
		((quote scalar_first_probe) stage requested_col)
		(rewrite_scalar_query_probe_recipe_expr
			(list (symbol "scalar_first_probe") stage requested_col) recipe_keys)
		((symbol scalar_first_probe) stage requested_col _stages)
		(rewrite_scalar_query_probe_recipe_expr
			(list (symbol "scalar_first_probe") stage requested_col) recipe_keys)
		((quote scalar_first_probe) stage requested_col _stages)
		(rewrite_scalar_query_probe_recipe_expr
			(list (symbol "scalar_first_probe") stage requested_col) recipe_keys)
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_scalar_query_probe_recipe_expr item recipe_keys))))
		_ expr)))

(define rewrite_scalar_query_probe_recipe_source (lambda (src recipe_keys)
	(source_with_join_expr src
		(rewrite_scalar_query_probe_recipe_expr (source_join_expr src) recipe_keys))))

(define query_block_with_scalar_query_probe_recipes (lambda (block entries)
	(begin
		(define recipe_keys (scalar_query_probe_recipe_keys entries))
		(make_query_block
			(qb_schema block)
			(map (qb_sources block) (lambda (src)
				(rewrite_scalar_query_probe_recipe_source src recipe_keys)))
			(rewrite_scalar_query_probe_recipe_expr (qb_fields block) recipe_keys)
			(rewrite_scalar_query_probe_recipe_expr (qb_where block) recipe_keys)
			(rewrite_scalar_query_probe_recipe_expr (qb_group block) recipe_keys)
			(rewrite_scalar_query_probe_recipe_expr (qb_having block) recipe_keys)
			(rewrite_scalar_query_probe_recipe_expr (qb_order block) recipe_keys)
			(qb_limit block)
			(qb_offset block)
			(rewrite_scalar_query_probe_recipe_expr (qb_hidden block) recipe_keys)
			(qb_stages block)
			(qb_facts block)))))

(define scalar_query_probe_recipe_seed (lambda (all_stages entry)
	(match entry
		'(stage requested_col) (list
			stage
			requested_col
			(scalar_first_query_probe_direct_nested_stages all_stages stage))
		_ (neumann_fail "build_queryplan" "malformed scalar query probe recipe entry"))))

(define scalar_query_probe_recipe_plan_using_index (lambda (closure_index bounded_recipe_keys seed)
	(match seed
		'(stage requested_col direct_stages) (begin
			(define bounded_consumer
				(has_assoc? bounded_recipe_keys (scalar_query_probe_recipe_key stage requested_col)))
			(define nested_stages
				(scalar_first_query_probe_nested_stages_using_index direct_stages closure_index))
			/* Only a directly consumed presence output can be replaced inside this
			recipe. Attached and transitive stages belong to their nested consumer. */
			(define direct_source_ids (stage_id_set
				(stage_outputs_from_sources_using direct_stages (qb_sources (gs_input stage)))))
			(define inline_candidates (if (equal? (count direct_stages) 1)
				(filter direct_stages (lambda (nested_stage)
					(and (has_assoc? direct_source_ids (gs_id nested_stage))
						(and (group_stage? nested_stage)
							(equal? (qassoc_get (gs_facts nested_stage) (quote purpose) nil) (quote exists))
							(equal? (qassoc_get (gs_facts nested_stage) (quote presence_only) false) true)
							(source_is_base_table? (gs_input nested_stage))
							(not (stage_has_residual_outer_refs? nested_stage))))))
				'()))
			/* Bounded consumers execute presence checks after root braking. Broad
			consumers and complex stage graphs retain the persistent group cache so
			repeated keys are shared with the canonical preparation path. */
			(define inline_presence_stages (if bounded_consumer inline_candidates '()))
			(define hoisted_stages (if (empty_list? inline_presence_stages)
				(filter nested_stages (lambda (nested_stage)
					(and (group_stage? nested_stage)
						(equal? (qassoc_get (gs_facts nested_stage) (quote purpose) nil) (quote exists))
						(equal? (qassoc_get (gs_facts nested_stage) (quote presence_only) false) true)
						(source_is_base_table? (gs_input nested_stage))
						(not (stage_has_residual_outer_refs? nested_stage)))))
				'()))
			(define consumed_ids (stage_id_set (merge (list hoisted_stages inline_presence_stages))))
			(define prepare_stages (filter nested_stages (lambda (nested_stage)
				(not (has_assoc? consumed_ids (gs_id nested_stage))))))
			(list
				stage
				requested_col
				nested_stages
				hoisted_stages
				prepare_stages
				inline_presence_stages))
		_ (neumann_fail "build_queryplan" "malformed scalar query probe recipe seed"))))

(define scalar_query_probe_recipe_plans_using_graph (lambda (all_stages dependency_graph entries bounded_recipe_keys)
	(begin
		(define seeds (map (coalesceNil entries '()) (lambda (entry)
			(scalar_query_probe_recipe_seed all_stages entry))))
		(define direct_stages (unique_stages_by_id (merge (map seeds (lambda (seed) (nth seed 2))))))
		(define closure_index (stage_dependency_closure_index_using_graph dependency_graph direct_stages))
		(map seeds (lambda (seed)
			(scalar_query_probe_recipe_plan_using_index closure_index bounded_recipe_keys seed))))))

(define scalar_query_probe_recipe_plans (lambda (all_stages entries bounded_recipe_keys)
	(scalar_query_probe_recipe_plans_using_graph
		all_stages
		(stage_dependency_graph all_stages)
		entries
		bounded_recipe_keys)))

(define scalar_query_probe_param_index (lambda (lookup_keys params)
	(reduce (produceN (count lookup_keys)) (lambda (index i)
		(match (nth lookup_keys i)
			((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
			(set_assoc index (nth lookup_keys i) (nth params i))
			((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
			(set_assoc index (nth lookup_keys i) (nth params i))
			_ index)) '())))

/* A query-probe lambda evaluates its outer lookup keys before entering nested
stages. Replace inherited direct column references with those parameters so
dependency preparation does not emit free outer-row symbols. */
(define rewrite_scalar_query_probe_params (lambda (index expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(coalesceNil (get_assoc index expr) expr)
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(rewrite_scalar_query_probe_params index
			(list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons (rewrite_scalar_query_probe_params index head) (map tail (lambda (item)
			(rewrite_scalar_query_probe_params index item))))
		_ expr)))

(define scalar_query_probe_recipe_binding (lambda (plan)
	(match plan
		'(stage requested_col nested_stages _hoisted_stages prepare_stages inline_presence_stages) (begin
			(define raw_keys (gs_keys stage))
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define keys (if (empty_list? lookup_keys) '() raw_keys))
			(if (not (equal? (count keys) (count lookup_keys)))
				(neumann_fail "build_queryplan" "scalar query probe recipe key/domain mismatch")
				true)
			(define ag (scalar_first_probe_aggregate stage requested_col))
			(if (nil? ag)
				(neumann_fail "build_queryplan" "scalar query probe recipe references unknown aggregate column")
				true)
			(define value_expr (nth (scalar_first_probe_parts ag) 0))
			(define params (map (produceN (count keys)) (lambda (i)
				(symbol (concat "__probe_key_" i)))))
			(define param_index (scalar_query_probe_param_index lookup_keys params))
			(define bound_stage (rewrite_scalar_query_probe_params param_index stage))
			(define bound_keys (gs_keys bound_stage))
			(define bound_value_expr (rewrite_scalar_query_probe_params param_index value_expr))
			(define bound_nested_stages (map nested_stages (lambda (nested_stage)
				(rewrite_scalar_query_probe_params param_index nested_stage))))
			(define bound_prepare_stages (map prepare_stages (lambda (prepare_stage)
				(rewrite_scalar_query_probe_params param_index prepare_stage))))
			(define bound_inline_presence_stages (map inline_presence_stages (lambda (presence_stage)
				(rewrite_scalar_query_probe_params param_index presence_stage))))
			(list
				(quote define)
				(symbol (scalar_query_probe_recipe_key stage requested_col))
				(list (quote lambda) params
					(lower_scalar_first_query_probe_expr_using bound_stage bound_value_expr bound_keys params
						bound_nested_stages bound_prepare_stages bound_inline_presence_stages))))
		_ (neumann_fail "build_queryplan" "malformed scalar query probe recipe plan"))))

(define scalar_query_probe_recipe_bindings (lambda (plans)
	(map (coalesceNil plans '()) scalar_query_probe_recipe_binding)))

(define scalar_query_probe_recipe_hoisted_stages (lambda (plans)
	(unique_stages_by_id (merge (map (coalesceNil plans '()) (lambda (plan)
		(match plan
			'(_stage _requested_col _nested_stages hoisted_stages _prepare_stages _inline_presence_stages) hoisted_stages
			_ '())))))))

(define scalar_query_probe_recipe_prepare_exprs (lambda (plans)
	(begin
		(define stages (scalar_query_probe_recipe_hoisted_stages plans))
		(merge (list
			(lower_unique_stage_prepares_using stages stages stages)
			(lower_stage_materialize_all stages))))))

(define lower_exists_union_probe_branch (lambda (sources default_alias branch probe all_stages dependency_graph)
	(begin
		(define prepared
			(query_block_with_prepared_sources_using_graph all_stages dependency_graph branch))
		(if (not (and (query_block? prepared) (single_source? (qb_sources prepared))))
			(neumann_fail "build_queryplan" "EXISTS UNION point probe requires one prepared branch source")
			true)
		(define src (car (qb_sources prepared)))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "EXISTS UNION point probe requires base-table branches")
			true)
		(define key_expr (query_block_first_expr prepared))
		(define condition (combine_where (qb_where prepared) (source_join_expr src)))
		(define source_rows (planner_source_row_count src))
		(define probe_term (list (quote equal??) key_expr probe))
		(define probe_work_rows (if (number? source_rows)
			(max 1 (* source_rows
				(join_optimizer_expr_selectivity (list src) (source_alias src) probe_term)))
			1))
		(define filtercols (merge_unique (list
			(extract_columns_for_alias src condition)
			(extract_columns_for_alias src key_expr))))
		(list (quote scan_exists)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(list (quote optimize)
					(list (quote and)
						(lower_column_expr_for_alias_in_context src condition probe_work_rows)
						(list (quote equal??)
							(lower_column_expr_for_alias src key_expr)
							(lower_column_expr_for_join sources default_alias probe)))))))))

/* EXISTS is short-circuiting, so selective independent UNION branches remain
one n-ary physical OR of bounded probes. This preserves the logical union tree
until lowering without creating a depth-proportional binary OR chain. */
(define lower_exists_union_probe_expr (lambda (sources default_alias branches probe all_stages)
	(begin
		/* All branches resolve against the same immutable decorrelation catalog.
		Build its indexes once instead of rediscovering the complete stage set for
		every arm of a wide UNION. */
		(define branch_stages (merge (map (coalesceNil branches '()) (lambda (branch)
			(if (query_block? branch) (qb_stages branch) '())))))
		(define stage_lookup
			(make_lowering_catalog (unique_stages_by_id (merge (list all_stages branch_stages)))))
		(define dependency_graph (stage_dependency_graph stage_lookup))
		(cons (quote or)
			(map (coalesceNil branches '()) (lambda (branch)
				(lower_exists_union_probe_branch
					sources default_alias branch probe stage_lookup dependency_graph)))))))

(define lower_scalar_first_probe_expr (lambda (sources default_alias stage requested_col all_stages probe_work_rows)
	(begin
		(if (not (scalar_or_presence_probe_stage? stage))
			(neumann_fail "build_queryplan" "stage probe requires scalar_single first or presence stage")
			true)
		(define src (gs_input stage))
		(define raw_keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(define keys (if (empty_list? lookup_keys) '() raw_keys))
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
		(if (union_block? src)
			(list (quote if)
				(lower_exists_union_probe_expr
					sources default_alias (union_branches src)
					(car lookup_keys) all_stages)
				1
				nil)
			(if (query_block? src)
				(lower_scalar_first_query_probe_expr
					all_stages
					stage
					value_expr
					keys
					(map lookup_keys (lambda (key) (lower_column_expr_for_join sources default_alias key)))
					probe_work_rows)
				(begin
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
						false))))))
)

(define lower_scalar_aggregate_probe_expr (lambda (sources default_alias stage requested_col)
	(begin
		(if (not (scalar_aggregate_probe_stage? stage))
			(neumann_fail "build_queryplan" "scalar aggregate probe requires scalar_aggregate base stage")
			true)
		(define src (gs_input stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(define keys (if (empty_list? lookup_keys) '() (gs_keys stage)))
		(if (not (equal? (count keys) (count lookup_keys)))
			(neumann_fail "build_queryplan" "scalar aggregate probe key/domain mismatch")
			true)
		(define ag (scalar_first_probe_aggregate stage requested_col))
		(if (nil? ag)
			(neumann_fail "build_queryplan" "scalar aggregate probe references unknown aggregate column")
			true)
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define key_terms (scalar_first_probe_key_terms sources default_alias src keys lookup_keys))
		(define value_expr (nth ag 0))
		(define reduce_expr (nth ag 1))
		(define neutral_expr (nth ag 2))
		(define condition_cols (extract_columns_for_alias src condition))
		(define key_cols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define value_cols (extract_columns_for_alias src value_expr))
		(define filtercols (merge_unique (list condition_cols key_cols)))
		(list (quote scan)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(list (quote optimize)
					(cons (quote and)
						(cons (lower_column_expr_for_alias src condition) key_terms))))
			(cons (quote list) value_cols)
			(list (quote lambda)
				(map value_cols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(lower_column_expr_for_alias src value_expr))
			reduce_expr
			neutral_expr
			nil
			false))))

(define lower_scalar_cardinality_probe_expr (lambda (sources default_alias stage requested_col)
	(begin
		(if (not (scalar_cardinality_probe_stage? stage))
			(neumann_fail "build_queryplan" "scalar cardinality probe requires scalar_single base stage")
			true)
		(define src (gs_input stage))
		(define keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(if (not (equal? (count keys) (count lookup_keys)))
			(neumann_fail "build_queryplan" "scalar cardinality probe key/domain mismatch")
			true)
		(define ag (scalar_first_probe_aggregate stage requested_col))
		(if (nil? ag)
			(neumann_fail "build_queryplan" "scalar cardinality probe references unknown aggregate column")
			true)
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define value_expr (nth ag 0))
		(define condition_cols (extract_columns_for_alias src condition))
		(define key_cols (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define value_cols (extract_columns_for_alias src value_expr))
		(define filtercols (merge_unique (list condition_cols key_cols)))
		(define unset (list (quote quote) (quote __scalar_cardinality_unset)))
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
			'(list)
			'(list)
			0
			0
			2
			(cons (quote list) value_cols)
			(list (quote lambda)
				(map value_cols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(lower_column_expr_for_alias src value_expr))
			(list (quote lambda) '((quote acc) (quote value))
				(list (quote if)
					(list (quote equal?) (quote acc) unset)
					(quote value)
					(list (quote error) "scalar subselect returned more than one row")))
			unset
			false
			nil))))

(define collect_join_columns_acc (lambda (sources default_alias target_alias expr columns_by_alias)
	(match expr
		((symbol driver_membership_probe) _stage probe)
		(collect_join_columns_acc sources default_alias target_alias probe columns_by_alias)
		((quote driver_membership_probe) _stage probe)
		(collect_join_columns_acc sources default_alias target_alias probe columns_by_alias)
		((symbol dml_driver_membership_probe) _fallback_schema _stage probe)
		(collect_join_columns_acc sources default_alias target_alias probe columns_by_alias)
		((quote dml_driver_membership_probe) _fallback_schema _stage probe)
		(collect_join_columns_acc sources default_alias target_alias probe columns_by_alias)
		((symbol scalar_first_probe) stage _requested_col)
		(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (acc key)
			(collect_join_columns_acc sources default_alias target_alias key acc)) columns_by_alias)
		((symbol scalar_first_probe) stage _requested_col _stages)
		(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (acc key)
			(collect_join_columns_acc sources default_alias target_alias key acc)) columns_by_alias)
		((quote scalar_first_probe) stage requested_col)
		(collect_join_columns_acc sources default_alias target_alias (list (quote scalar_first_probe) stage requested_col) columns_by_alias)
		((quote scalar_first_probe) stage requested_col _stages)
		(collect_join_columns_acc sources default_alias target_alias (list (quote scalar_first_probe) stage requested_col) columns_by_alias)
		((symbol scalar_aggregate_probe) stage _requested_col)
		(reduce (scalar_aggregate_probe_outer_exprs stage) (lambda (acc expr)
			(collect_join_columns_acc sources default_alias target_alias expr acc)) columns_by_alias)
		((quote scalar_aggregate_probe) stage requested_col)
		(collect_join_columns_acc sources default_alias target_alias (list (quote scalar_aggregate_probe) stage requested_col) columns_by_alias)
		((symbol scalar_cardinality_probe) stage _requested_col)
		(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (acc key)
			(collect_join_columns_acc sources default_alias target_alias key acc)) columns_by_alias)
		((quote scalar_cardinality_probe) stage requested_col)
		(collect_join_columns_acc sources default_alias target_alias (list (quote scalar_cardinality_probe) stage requested_col) columns_by_alias)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (if (nil? tblvar)
				(source_for_unqualified_column sources default_alias col col_ignorecase)
				(source_for_alias sources default_alias tblvar tbl_ignorecase)))
			(if (or (nil? src) (and (not (nil? target_alias)) (not (equal?? (source_alias src) target_alias))))
				columns_by_alias
				(begin
					(define alias (source_alias src))
					(define physical_col (resolve_physical_column_name src col col_ignorecase))
					(qassoc_set columns_by_alias alias
						(merge_unique (list (qassoc_get columns_by_alias alias '()) (list physical_col)))))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(collect_join_columns_acc sources default_alias target_alias
			(list (symbol "get_column") tblvar tbl_ignorecase col col_ignorecase) columns_by_alias)
		(cons _head tail) (reduce tail (lambda (acc item)
			(collect_join_columns_acc sources default_alias target_alias item acc)) columns_by_alias)
		_ columns_by_alias)))

(define extract_columns_for_join_alias (lambda (sources default_alias alias expr)
	(qassoc_get (collect_join_columns_acc sources default_alias alias expr '()) alias '())))

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
		(lower_scalar_first_probe_expr sources default_alias stage requested_col (list stage) nil)
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col stages nil)
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col (list stage) nil)
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col stages nil)
		((symbol scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr sources default_alias stage requested_col)
		((quote scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr sources default_alias stage requested_col)
		((symbol scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr sources default_alias stage requested_col)
		((quote scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr sources default_alias stage requested_col)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (if (nil? tblvar)
				(source_for_unqualified_column sources default_alias col col_ignorecase)
				(source_for_alias sources default_alias tblvar tbl_ignorecase)))
			(if (nil? src)
				(symbol (concat (resolve_column_alias tblvar default_alias) "." col))
				(symbol (concat (source_alias src) "." (resolve_physical_column_name src col col_ignorecase)))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
			(define src (if (nil? tblvar)
				(source_for_unqualified_column sources default_alias col col_ignorecase)
				(source_for_alias sources default_alias tblvar tbl_ignorecase)))
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

(define scan_order_sort_callback_symbols (lambda (driver_cols bound_symbols expr)
	(match expr
		((symbol session) "__memcp_tx") (symbol "$tx")
		((quote session) "__memcp_tx") (symbol "$tx")
		((symbol quote) _value) expr
		((symbol lambda) params body) (list (quote lambda) params
			(scan_order_sort_callback_symbols driver_cols (merge_unique (list bound_symbols params)) body))
		((symbol lambda) params body numvars) (list (quote lambda) params
			(scan_order_sort_callback_symbols driver_cols (merge_unique (list bound_symbols params)) body)
			numvars)
		(cons head tail) (cons
			(scan_order_sort_callback_symbols driver_cols bound_symbols head)
			(map tail (lambda (item) (scan_order_sort_callback_symbols driver_cols bound_symbols item))))
		_ (if (or (string? expr) (contains? bound_symbols expr))
			expr
			(begin
				(define text (string expr))
				(define col (find driver_cols (lambda (candidate)
					(begin
						(define suffix (concat "." candidate))
						(and (> (strlen text) (strlen suffix))
							(equal? (substr text (- (strlen text) (strlen suffix)) (strlen suffix)) suffix)))) nil))
				(if (nil? col) expr (symbol col)))))))

(define bind_nested_scan_tx (lambda (expr tx_expr)
	(match expr
		((symbol session) "__memcp_tx") tx_expr
		((quote session) "__memcp_tx") tx_expr
		((symbol quote) _value) expr
		((quote quote) _value) expr
		(cons head tail) (cons head (map tail (lambda (item)
			(bind_nested_scan_tx item tx_expr))))
		_ expr)))

(define lower_scan_order_sort_expr_for_alias (lambda (src driver_cols expr)
	(scan_order_sort_callback_symbols driver_cols '()
		(bind_nested_scan_tx
			(lower_column_expr_for_alias src expr)
			(symbol "$tx")))))

(define scan_order_sort_column_for_alias (lambda (src expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(begin
			(define cols (merge_unique (list (extract_columns_for_alias src expr) (list "$tx"))))
			(list (quote lambda)
				(map cols (lambda (col) (symbol col)))
				(lower_scan_order_sort_expr_for_alias src cols expr)))
		((symbol scalar_first_probe) stage requested_col _stages)
		(scan_order_sort_column_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((quote scalar_first_probe) stage requested_col)
		(scan_order_sort_column_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((quote scalar_first_probe) stage requested_col _stages)
		(scan_order_sort_column_for_alias src (list (quote scalar_first_probe) stage requested_col))
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(resolve_physical_column_name src col col_ignorecase)
			(neumann_fail "build_queryplan" "ORDER BY references a different source"))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(scan_order_sort_column_for_alias src (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		_ (begin
			(define cols (extract_columns_for_alias src expr))
			(list (quote lambda)
				(map cols (lambda (col) (symbol col)))
				(lower_scan_order_sort_expr_for_alias src cols expr))))))

(define scan_order_sort_columns_for_alias (lambda (src order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (scan_order_sort_column_for_alias src expr)
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define unique_lookup_join_term? (lambda (default_alias src key_col term)
	(match term
		'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
			(or
				(and (equal? (direct_column_name_for_alias src left) key_col)
					(not (expr_refs_alias? default_alias (source_alias src) right)))
				(and (equal? (direct_column_name_for_alias src right) key_col)
					(not (expr_refs_alias? default_alias (source_alias src) left))))
			false)
		_ false)))

(define unique_lookup_key_columns (lambda (src stages)
	(begin
		(define group_cache_stage (stage_for_group_cache_source stages src))
		(if (group_stage? group_cache_stage)
			(group_key_cols (gs_keys group_cache_stage))
			(if (source_is_base_table? src)
				(prejoin_primary_key_columns src)
				(if (stage_output_relation? (source_relation src))
					(begin
						(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
						(if (group_stage? stage) (group_key_cols (gs_keys stage)) '()))
					'()))))))

(define lookup_probe_term_from_sources? (lambda (sources default_alias bound_sources lookup term)
	(begin
		(define aliases (join_hypergraph_expr_aliases default_alias (source_aliases sources) term))
		(and (expr_refs_alias? default_alias (source_alias lookup) term)
			(reduce aliases (lambda (valid alias)
				(and valid (or
					(equal? alias (source_alias lookup))
					(contains? (source_aliases bound_sources) alias)))) true)))))

(define lookup_probe_condition_from_sources (lambda (sources default_alias bound_sources lookup condition)
	(combine_where_terms
		(filter (split_and_terms (combine_where (source_join_expr lookup) condition)) (lambda (term)
			(lookup_probe_term_from_sources? sources default_alias bound_sources lookup term)))
		true)))

(define lookup_probe_condition (lambda (sources default_alias driver lookup condition)
	(lookup_probe_condition_from_sources sources default_alias (list driver) lookup condition)))

(define source_is_unique_lookup_from_sources? (lambda (sources default_alias bound_sources src stages condition)
	(begin
		(define stage (coalesceNil
			(source_stage_output_stage stages src)
			(stage_for_group_cache_source stages src)))
		(define key_cols (unique_lookup_key_columns src stages))
		(define lookup_condition (lookup_probe_condition_from_sources
			sources default_alias bound_sources src condition))
		(or
			(and (scalar_aggregate_probe_stage? stage)
				(empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
			(and (not (empty_list? key_cols))
				(reduce key_cols (lambda (unique key_col)
					(and unique (reduce (split_and_terms lookup_condition)
						(lambda (found term) (or found (unique_lookup_join_term? default_alias src key_col term))) false)))
					true))))))

(define source_is_unique_lookup? (lambda (sources default_alias driver src stages condition)
	(source_is_unique_lookup_from_sources?
		sources default_alias (list driver) src stages condition)))

(define order_expr_unique_lookup_source (lambda (sources driver default_alias expr stages condition)
	(begin
		(define columns (join_column_recipe sources default_alias (list expr)))
		(define referenced (filter (cdr sources) (lambda (src)
			(not (empty_list? (qassoc_get columns (source_alias src) '()))))))
		(if (and (equal? (count referenced) 1) (source_is_unique_lookup? sources default_alias driver (car referenced) stages condition))
			(car referenced)
			nil))))

(define join_term_driver_equivalent (lambda (driver expr term)
	(match term
		'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
			(if (and (equal? expr left) (order_expr_belongs_to_source? driver right))
				right
				(if (and (equal? expr right) (order_expr_belongs_to_source? driver left)) left nil))
			nil)
		_ nil)))

(define order_expr_driver_equivalent (lambda (sources driver expr)
	(reduce (cdr sources) (lambda (found src)
		(if (not (nil? found))
			found
			(reduce (split_and_terms (coalesceNil (source_join_expr src) true))
				(lambda (equivalent term)
					(coalesceNil equivalent (join_term_driver_equivalent driver expr term))) nil)))
		nil)))

(define scan_order_unique_lookup_sort_column (lambda (sources default_alias driver lookup expr stages condition)
	(begin
		(define driver_alias (source_alias driver))
		(define lookup_alias (source_alias lookup))
		(define probe_condition (lookup_probe_condition sources default_alias driver lookup condition))
		(define driver_cols (merge_unique (list
			(extract_columns_for_join_alias sources default_alias driver_alias probe_condition)
			(extract_columns_for_join_alias sources default_alias driver_alias expr)
			(list "$tx"))))
		(define lookup_filter_cols (extract_columns_for_join_alias sources default_alias lookup_alias probe_condition))
		(define lookup_map_cols (extract_columns_for_join_alias sources default_alias lookup_alias expr))
		(define filter_expr (list (quote lambda)
			(map lookup_filter_cols (lambda (col) (symbol (concat lookup_alias "." col))))
			(list (quote optimize) (lower_column_expr_for_join sources default_alias probe_condition))))
		(define map_expr (list (quote lambda)
			(map lookup_map_cols (lambda (col) (symbol (concat lookup_alias "." col))))
			(lower_column_expr_for_join sources default_alias expr)))
		(define probe (list (quote scan_order)
			'(session "__memcp_tx")
			(source_table_expr_using stages lookup)
			(cons (quote list) lookup_filter_cols)
			filter_expr
			(quoted_runtime_list '())
			(quoted_runtime_list '())
			0 0 1
			(cons (quote list) lookup_map_cols)
			map_expr
			(list (quote lambda) (list (quote _old) (quote value)) (quote value))
			nil true nil))
		(list (quote lambda)
			(map driver_cols symbol)
			(scan_order_sort_callback_symbols driver_cols '() probe)))))

(define scan_order_sort_column_for_join_driver (lambda (sources default_alias driver expr stages condition)
	(if (order_expr_belongs_to_source? driver expr)
		(scan_order_sort_column_for_alias driver expr)
		(begin
			(define equivalent (order_expr_driver_equivalent sources driver expr))
			(if (not (nil? equivalent))
				(scan_order_sort_column_for_alias driver equivalent)
				(begin
					(define lookup (order_expr_unique_lookup_source sources driver default_alias expr stages condition))
					(if (nil? lookup)
						(neumann_fail "build_queryplan" "ORDER BY requires a storage carrier")
						(scan_order_unique_lookup_sort_column sources default_alias driver lookup expr stages condition))))))))

(define scan_order_sort_columns_for_join_driver (lambda (sources default_alias driver order_items stages condition)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr _dir) (scan_order_sort_column_for_join_driver sources default_alias driver expr stages condition)
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define order_items_supported_by_join_driver? (lambda (sources default_alias driver order_items stages condition)
	(reduce (order_exprs order_items) (lambda (supported expr)
		(and supported (or
			(order_expr_belongs_to_source? driver expr)
			(not (nil? (order_expr_driver_equivalent sources driver expr)))
			(not (nil? (order_expr_unique_lookup_source sources driver default_alias expr stages condition)))))) true)))

(define split_order_items_for_join_driver (lambda (sources default_alias driver order_items stages condition accepted)
	(match (coalesceNil order_items '())
		(cons item rest) (if (order_items_supported_by_join_driver?
			sources default_alias driver (list item) stages condition)
			(split_order_items_for_join_driver sources default_alias driver rest stages condition (merge accepted (list item)))
			(list accepted order_items))
		_ (list accepted '()))))

(define order_items_follow_join_tree? (lambda (sources default_alias order_items stages condition)
	(if (empty_list? order_items)
		true
		(if (empty_list? sources)
			false
			(begin
				(define parts (split_order_items_for_join_driver
					sources default_alias (car sources) order_items stages condition '()))
				(define current (nth parts 0))
				(define remaining (nth parts 1))
				(if (empty_list? current)
					(and (constant_scalar_or_presence_stage_output_source? stages (car sources))
						(order_items_follow_join_tree? (cdr sources) default_alias order_items stages condition))
					(order_items_follow_join_tree? (cdr sources) default_alias remaining stages condition)))))))

(define order_dirs (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) dir
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

(define order_expr_belongs_to_source? (lambda (src expr)
	(match expr
		((symbol scalar_first_probe) stage _requested_col)
		(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (local key)
			(and local (order_expr_belongs_to_source? src key))) true)
		((symbol scalar_first_probe) stage _requested_col _stages)
		(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (local key)
			(and local (order_expr_belongs_to_source? src key))) true)
		((quote scalar_first_probe) stage requested_col)
		(order_expr_belongs_to_source? src (list (quote scalar_first_probe) stage requested_col))
		((quote scalar_first_probe) stage requested_col stages)
		(order_expr_belongs_to_source? src (list (quote scalar_first_probe) stage requested_col stages))
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
		(define cache (group_stage_cache stage))
		(define schema (group_cache_schema cache))
		(define grouptbl (group_cache_relation cache))
		(define group_src (list grouptbl schema grouptbl false nil))
		(define value_expr (replace_group_expr alias grouptbl keys key_names ags (first_projection_expr (gs_output stage))))
		(define replaced_order (map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr dir) (list (replace_group_order_expr alias grouptbl keys key_names ags expr) dir)))))
		(define session_filter (group_stage_session_filter_expr stage grouptbl keys key_names))
		(define filtercols (extract_columns_for_alias group_src session_filter))
		(define ordercols (order_cols_for_alias group_src replaced_order))
		(define valuecols (extract_columns_for_alias group_src value_expr))
		(list (quote scan_order)
			'(session "__memcp_tx")
			(list (quote table) schema grouptbl)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat grouptbl "." col))))
				(list (quote optimize) (lower_column_expr_for_alias group_src session_filter)))
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
			(list (quote list))
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
			(list (quote list))
			(list (quote lambda) '() 1)
			(quote +)
			0
			nil
			false))))

(define lower_driver_membership_probe_expr (lambda (sources default_alias stage probe)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(list (quote >)
				(cons (quote +) (map (union_branches input) (lambda (branch)
					(driver_membership_probe_branch_expr sources default_alias branch probe))))
				0)
			(begin
				(define input_src (if (query_block? input)
					(if (single_source? (qb_sources input)) (car (qb_sources input)) nil)
					input))
				(if (or (nil? input_src) (not (source_is_base_table? input_src)))
					(neumann_fail "build_queryplan" "driver membership probe expects UNION or simple base-table input")
					true)
				(define keys (gs_keys stage))
				(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
				(if (not (equal? (count keys) (count lookup_keys)))
					(neumann_fail "build_queryplan" "driver membership probe key/domain mismatch")
					true)
				(define condition (if (query_block? input)
					(combine_where (qb_where input) (source_join_expr input_src))
					(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)))
				(define key_terms (map (produceN (count keys)) (lambda (i)
					(list (quote equal??)
						(lower_column_expr_for_alias input_src (nth keys i))
						(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))
				(define filtercols (merge_unique (list
					(extract_columns_for_alias input_src condition)
					(merge_unique (map keys (lambda (key) (extract_columns_for_alias input_src key)))))))
				(list (quote scan_exists)
					'(session "__memcp_tx")
					(source_table_expr input_src)
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat (source_alias input_src) "." col))))
						(list (quote optimize)
							(cons (quote and)
								(cons (lower_column_expr_for_alias input_src condition) key_terms)))))))))))

(define lower_dml_driver_membership_probe_expr (lambda (sources default_alias fallback_schema stage _probe)
	(begin
		(define keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(if (< (count keys) (count lookup_keys))
			(neumann_fail "build_queryplan" "DML membership probe key/domain mismatch")
			true)
		(define key_names (group_key_cols keys))
		(define filter_key_names (map (produceN (count lookup_keys)) (lambda (i) (nth key_names i))))
		(define cache_schema (coalesceNil (group_stage_cache_schema stage) fallback_schema))
		(define key_terms (map (produceN (count lookup_keys)) (lambda (i)
			(list (quote equal??)
				(symbol (nth key_names i))
				(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))
		(list (quote scan_exists)
			'(session "__memcp_tx")
			(list (quote table) cache_schema (group_stage_cache_relation stage))
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

(define recset_project_join_branch_parts (lambda (branch)
	(if (or (not (and (query_block? branch) (single_source? (qb_sources branch))))
		(not (empty_list? (qb_stages branch))))
		nil
		(begin
			(define src (car (qb_sources branch)))
			(define source_col (if (source_is_base_table? src)
				(direct_column_name_for_alias src (query_block_first_expr branch))
				nil))
			(if (nil? source_col) nil (list src source_col))))))

(define recset_project_join_branch_expr (lambda (target_src branch target_col)
	(match (recset_project_join_branch_parts branch)
		'(src source_col) (begin
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
				(quoted_runtime_list (list target_col))))
		_ nil)))

(define exists_recset_projection_parts_acc (lambda (input_src target_src keys lookup_keys source_cols target_cols constant_terms)
	(match keys
		(cons inner_expr inner_rest) (match lookup_keys
			(cons lookup_expr lookup_rest) (begin
				(define source_col (direct_column_name_for_alias input_src inner_expr))
				(define target_col (direct_column_name_for_alias target_src lookup_expr))
				(if (nil? source_col)
					nil
					(if (not (nil? target_col))
						(exists_recset_projection_parts_acc input_src target_src inner_rest lookup_rest
							(cons source_col source_cols)
							(cons target_col target_cols)
							constant_terms)
						(if (expr_refs_sources? nil (list target_src) lookup_expr)
							nil
							(exists_recset_projection_parts_acc input_src target_src inner_rest lookup_rest
								source_cols target_cols
								(cons (list (quote equal??) inner_expr lookup_expr) constant_terms))))))
			_ nil)
		_ (if (empty_list? lookup_keys)
			(list (reverse source_cols) (reverse target_cols) (reverse constant_terms))
			nil))))

(define exists_recset_projection_parts (lambda (stage target_src)
	(exists_recset_projection_parts_acc
		(gs_input stage)
		target_src
		(gs_keys stage)
		(qassoc_get (gs_facts stage) (quote lookup-keys) '())
		'() '() '())))

(define exists_recset_project_join_expr (lambda (target_src stage)
	(begin
		(define input_src (gs_input stage))
		(define parts (exists_recset_projection_parts stage target_src))
		(if (or (nil? parts) (empty_list? (nth parts 0)))
			nil
			(begin
				(define alias (source_alias input_src))
				(define condition (combine_where_terms
					(cons
						(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)
						(nth parts 2))
					true))
				(define filtercols (extract_columns_for_alias input_src condition))
				(list (quote recset_project_join)
					'(session "__memcp_tx")
					(list (quote scan_recset)
						'(session "__memcp_tx")
						(source_table_expr input_src)
						(cons (quote list) filtercols)
						(list (quote lambda)
							(map filtercols (lambda (col) (symbol (concat alias "." col))))
							(list (quote optimize) (lower_column_expr_for_alias input_src condition))))
					(quoted_runtime_list (nth parts 0))
					(source_table_expr target_src)
					(quoted_runtime_list (nth parts 1))))))))

/* The canonical membership stage has already computed arbitrary RHS key
expressions into cache column k0. Project that compact key set to the driver;
the auto-index chooses the concrete access path on both tables. */
(define membership_cache_recset_project_join_expr (lambda (target_src stage target_col)
	(if (or (not (equal? (count (gs_keys stage)) 1))
		(not (equal? (count (qassoc_get (gs_facts stage) (quote lookup-keys) '())) 1)))
		nil
		(begin
			(define cache (group_stage_cache stage))
			(list (quote recset_project_join)
				'(session "__memcp_tx")
				(list (quote scan_recset)
					'(session "__memcp_tx")
					(list (quote table) (group_cache_schema cache) (group_cache_relation cache))
					(list (quote list))
					(list (quote lambda) '() true))
				(quoted_runtime_list (list "k0"))
				(source_table_expr target_src)
				(quoted_runtime_list (list target_col)))))))

(define recset_project_join_expr_for_membership (lambda (src membership)
	(begin
		(define stage (nth membership 0))
		(define target_col (nth membership 2))
		(define input (gs_input stage))
		(if (union_block? input)
			(begin
				(define projected (map (union_branches input) (lambda (branch)
					(recset_project_join_branch_expr src branch target_col))))
				(if (reduce projected (lambda (unsupported item) (or unsupported (nil? item))) false)
					nil
					(if (single_source? projected)
						(car projected)
						(list (quote recset_union) (cons (quote list) projected)))))
			(if (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote in_membership))
				(membership_cache_recset_project_join_expr src stage target_col)
				(if (exists_recset_stage? stage)
					(if (> (count (gs_keys stage)) 1)
						(exists_recset_project_join_expr src stage)
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
										(quoted_runtime_list (list target_col)))))))
					nil))))))

(define lower_scalar_marker_expr (lambda (expr)
	(match expr
		((symbol grouped_scalar_top) stage) (lower_grouped_scalar_top_expr stage)
		((quote grouped_scalar_top) stage) (lower_grouped_scalar_top_expr stage)
		((symbol scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr '() nil stage requested_col)
		((quote scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr '() nil stage requested_col)
		((symbol scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr '() nil stage requested_col)
		((quote scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr '() nil stage requested_col)
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
		((symbol scalar_aggregate_probe) stage requested_col)
		(if (qassoc_get (gs_facts stage) (quote direct_group_probe) false)
			'()
			(merge_unique (map (list stage requested_col) extract_aggregates)))
		((quote scalar_aggregate_probe) stage requested_col)
		(if (qassoc_get (gs_facts stage) (quote direct_group_probe) false)
			'()
			(merge_unique (map (list stage requested_col) extract_aggregates)))
		((symbol count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((quote count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((symbol aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		((quote aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		(cons head tail) (merge_unique (map tail extract_aggregates))
		_ '())))

/* DECIMAL is currently represented as float64 at runtime. Keep calculations
unmodified, but normalize visible aggregate results to the declared scale of
their DECIMAL inputs so binary floating-point residue does not leak through the
SQL output boundary. This annotation is attached while base-column provenance
is still available and remains an ordinary scalar expression through untangle. */
(define decimal_column_scale (lambda (src col)
	(if (or (not (string? (source_schema src))) (not (string? (source_relation src))))
		nil
		(begin
			(define info (find (get_schema (source_schema src) (source_relation src))
				(lambda (candidate) (equal?? (candidate "Field") col)) nil))
			(define dimensions (if (nil? info) '() (coalesceNil (info "Dimensions") '())))
			(if (and (not (nil? info))
				(equal? (toLower (coalesceNil (info "RawType") "")) "decimal")
				(> (count dimensions) 1))
				(nth dimensions 1)
				nil)))))

(define decimal_derived_column_scale (lambda (relation col)
	(begin
		(define normalized (normalize_query_ast relation))
		(if (query_block? normalized)
			(reduce_assoc (qb_fields normalized) (lambda (best title expr)
				(if (and (string? title) (string? col)
					(equal? (toLower title) (toLower col)))
					(decimal_expr_scale (qb_sources normalized) expr)
					best)) nil)
			nil))))

(define decimal_source_column_scale (lambda (src col)
	(begin
		(define derived_scale (decimal_derived_column_scale (source_relation src) col))
		(if (nil? derived_scale) (decimal_column_scale src col) derived_scale))))

(define decimal_column_scale_for_sources (lambda (sources tblvar col)
	(reduce (coalesceNil sources '()) (lambda (best src)
		(if (and (or (nil? tblvar) (equal?? tblvar (source_alias src)))
			(not (stage_output_relation? (source_relation src))))
			(begin
				(define scale (decimal_source_column_scale src col))
				(if (nil? scale) best (if (nil? best) scale (max best scale))))
			best)) nil)))

(define decimal_expr_scale (lambda (sources expr)
	(match expr
		((symbol get_column) tblvar _ignorecase col _col_ignorecase)
		(decimal_column_scale_for_sources sources tblvar col)
		((quote get_column) tblvar _ignorecase col _col_ignorecase)
		(decimal_column_scale_for_sources sources tblvar col)
		(cons _head tail) (reduce tail (lambda (best item)
			(begin
				(define scale (decimal_expr_scale sources item))
				(if (nil? scale) best (if (nil? best) scale (max best scale))))) nil)
		_ nil)))

(define decimal_aggregate_output_scale (lambda (sources expr)
	(reduce (extract_aggregates expr) (lambda (best aggregate_descriptor)
		(begin
			(define scale (decimal_expr_scale sources (nth aggregate_descriptor 0)))
			(if (nil? scale) best (if (nil? best) scale (max best scale))))) nil)))

(define sanitize_decimal_aggregate_fields (lambda (sources fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(begin
			(define scale (decimal_aggregate_output_scale sources expr))
			(if (nil? scale) expr (list (quote sql_decimal_output) expr scale)))))))

(define sanitize_decimal_aggregate_outputs (lambda (query)
	(begin
		(define normalized (normalize_query_ast query))
		(if (query_block? normalized)
			(make_query_block
				(qb_schema normalized)
				(map (qb_sources normalized) (lambda (src)
					(source_with_relation src
						(sanitize_decimal_aggregate_outputs (source_relation src)))))
				(sanitize_decimal_aggregate_fields (qb_sources normalized) (qb_fields normalized))
				(qb_where normalized)
				(qb_group normalized)
				(qb_having normalized)
				(qb_order normalized)
				(qb_limit normalized)
				(qb_offset normalized)
				(qb_hidden normalized)
				(qb_stages normalized)
				(qb_facts normalized))
			(if (union_block? normalized)
				(make_union_block
					(union_mode normalized)
					(map (union_branches normalized) sanitize_decimal_aggregate_outputs)
					(union_order normalized)
					(union_limit normalized)
					(union_offset normalized)
					(union_facts normalized))
				normalized)))))

/* Scmer and the storage engine intentionally keep temporal values generic.
The SQL compiler retains the declared DATE/DATETIME/TIMESTAMP provenance and
formats only visible result fields, after all relational operations have kept
working on the original values. */
(define temporal_column_type (lambda (src col)
	(if (or (not (string? (source_schema src))) (not (string? (source_relation src))))
		nil
		(begin
			(define info (find (get_schema (source_schema src) (source_relation src))
				(lambda (candidate) (equal?? (candidate "Field") col)) nil))
			(define raw_type (toLower (if (nil? info) "" (coalesceNil (info "RawType") ""))))
			(match raw_type
				"date" "DATE"
				"datetime" "DATETIME"
				"timestamp" "TIMESTAMP"
				_ nil)))))

(define temporal_derived_column_type (lambda (relation col)
	(begin
		(define normalized (normalize_query_ast relation))
		(if (query_block? normalized)
			(reduce_assoc (qb_fields normalized) (lambda (best title expr)
				(if (and (nil? best) (string? title) (string? col)
					(equal? (toLower title) (toLower col)))
					(temporal_expr_type (qb_sources normalized) expr)
					best)) nil)
			nil))))

(define temporal_source_column_type (lambda (src col)
	(begin
		(define derived_type (temporal_derived_column_type (source_relation src) col))
		(if (nil? derived_type) (temporal_column_type src col) derived_type))))

(define temporal_column_type_for_sources (lambda (sources tblvar col)
	(reduce (coalesceNil sources '()) (lambda (best src)
		(if (and (nil? best)
			(or (nil? tblvar) (equal?? tblvar (source_alias src)))
			(not (stage_output_relation? (source_relation src))))
			(temporal_source_column_type src col)
			best)) nil)))

(define temporal_date_arithmetic_type (lambda (sources value unit)
	(begin
		(define value_type (temporal_expr_type sources value))
		(if (and (equal? value_type "DATE") (string? unit))
			(match (toLower unit)
				"hour" "DATETIME"
				"minute" "DATETIME"
				"second" "DATETIME"
				"microsecond" "DATETIME"
				_ "DATE")
			value_type))))

(define temporal_expr_type (lambda (sources expr)
	(match expr
		((symbol get_column) tblvar _ignorecase col _col_ignorecase)
		(temporal_column_type_for_sources sources tblvar col)
		((quote get_column) tblvar _ignorecase col _col_ignorecase)
		(temporal_column_type_for_sources sources tblvar col)
		((symbol date_trunc_day) _value) "DATE"
		((quote date_trunc_day) _value) "DATE"
		((symbol current_date)) "DATE"
		((quote current_date)) "DATE"
		((symbol date_add) value _amount unit) (temporal_date_arithmetic_type sources value unit)
		((quote date_add) value _amount unit) (temporal_date_arithmetic_type sources value unit)
		((symbol date_sub) value _amount unit) (temporal_date_arithmetic_type sources value unit)
		((quote date_sub) value _amount unit) (temporal_date_arithmetic_type sources value unit)
		((symbol aggregate) value reducer _neutral)
		(if (or (equal? reducer min) (equal? reducer max)
			(equal? reducer (quote min)) (equal? reducer (quote max)))
			(temporal_expr_type sources value)
			nil)
		((quote aggregate) value reducer _neutral)
		(if (or (equal? reducer min) (equal? reducer max)
			(equal? reducer (quote min)) (equal? reducer (quote max)))
			(temporal_expr_type sources value)
			nil)
		((symbol now)) "TIMESTAMP"
		((quote now)) "TIMESTAMP"
		((symbol utc_timestamp)) "TIMESTAMP"
		((quote utc_timestamp)) "TIMESTAMP"
		((symbol from_unixtime) _value) "DATETIME"
		((quote from_unixtime) _value) "DATETIME"
		((symbol sql_temporal_output) _value sql_type) sql_type
		((quote sql_temporal_output) _value sql_type) sql_type
		_ nil)))

(define sanitize_temporal_output_fields (lambda (sources fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(begin
			(define sql_type (temporal_expr_type sources expr))
			(if (nil? sql_type) expr (list (quote sql_temporal_output) expr sql_type)))))))

(define sanitize_temporal_outputs (lambda (query)
	(begin
		(define normalized (normalize_query_ast query))
		(if (query_block? normalized)
			(make_query_block
				(qb_schema normalized)
				(qb_sources normalized)
				(sanitize_temporal_output_fields (qb_sources normalized) (qb_fields normalized))
				(qb_where normalized)
				(qb_group normalized)
				(qb_having normalized)
				(qb_order normalized)
				(qb_limit normalized)
				(qb_offset normalized)
				(qb_hidden normalized)
				(qb_stages normalized)
				(qb_facts normalized))
			(if (union_block? normalized)
				(make_union_block
					(union_mode normalized)
					(map (union_branches normalized) sanitize_temporal_outputs)
					(union_order normalized)
					(union_limit normalized)
					(union_offset normalized)
					(union_facts normalized))
				normalized)))))

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

(define group_table_name (lambda (schema tbl alias keys condition)
	(concat ".grp:" tbl ":" (fnv_hash (serialize (list "neumann-clean-groups-v4" schema tbl alias keys condition))))))

(define canonical_group_stage_alias "__grp")

(define canonicalize_group_stage_local_expr (lambda (alias expr)
	(requalify_single_source_expr alias canonical_group_stage_alias expr)))

(define canonicalize_group_stage_local_exprs (lambda (alias exprs)
	(map (coalesceNil exprs '()) (lambda (expr) (canonicalize_group_stage_local_expr alias expr)))))

(define make_group_keytable_cache (lambda (schema relation)
	(list (quote group-keytable) schema relation)))

(define group_cache_kind (lambda (cache)
	(if (list? cache) (nth cache 0) nil)))

(define group_cache_schema (lambda (cache)
	(if (list? cache) (nth cache 1) nil)))

(define group_cache_relation (lambda (cache)
	(if (list? cache) (nth cache 2) nil)))

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

(define group_stage_default_cache (lambda (stage)
	(begin
		(define schema (group_stage_schema stage))
		(define tbl (group_stage_input_name stage))
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(make_group_keytable_cache schema (group_table_name
			schema
			tbl
			canonical_group_stage_alias
			(canonicalize_group_stage_local_exprs alias keys)
			(canonicalize_group_stage_local_expr alias condition))))))

(define group_stage_cache (lambda (stage)
	(coalesceNil
		(qassoc_get (gs_facts stage) (quote group_cache) nil)
		(group_stage_default_cache stage))))

(define group_stage_cache_relation (lambda (stage)
	(group_cache_relation (group_stage_cache stage))))

(define group_stage_cache_schema (lambda (stage)
	(group_cache_schema (group_stage_cache stage))))

(define group_key_cols (lambda (keys)
	(map (produceN (count keys)) group_key_col_name)))

(define group_stage_session_domain_keys (lambda (stage)
	(query_expr_session_reads (gs_domain stage))))

(define group_key_expr_index (lambda (keys expr)
	(reduce (produceN (count keys)) (lambda (found i)
		(if (not (nil? found)) found (if (equal? (nth keys i) expr) i nil)))
		nil)))

(define group_stage_session_key_pairs (lambda (stage keys key_names)
	(map (group_stage_session_domain_keys stage) (lambda (expr)
		(begin
			(define direct_idx (group_key_expr_index keys expr))
			(define lookup_idx (group_key_expr_index
				(coalesceNil (qassoc_get (gs_facts stage) (quote lookup-keys) '()) '())
				expr))
			(define idx (if (not (nil? direct_idx)) direct_idx lookup_idx))
			(if (nil? idx)
				(neumann_fail "build_queryplan" (concat "session domain expression is not a group key: "
					(serialize expr) " in " (serialize keys)))
				(if (>= idx (count key_names))
					(neumann_fail "build_queryplan" (concat "session lookup key has no matching group key: "
						(serialize expr) " in " (serialize keys)))
					(list expr (nth key_names idx)))))))))

(define replace_group_session_expr (lambda (stage keys key_names expr)
	(begin
		(define pair (reduce (group_stage_session_key_pairs stage keys key_names)
			(lambda (found candidate)
				(if (not (nil? found)) found (if (equal? expr (nth candidate 0)) candidate nil)))
			nil))
		(if (not (nil? pair))
			(list (quote outer) (symbol (nth pair 1)))
			(match expr
				(cons head tail) (cons head (map tail (lambda (item)
					(replace_group_session_expr stage keys key_names item))))
				_ expr)))))

(define group_stage_session_filter_expr (lambda (stage grouptbl keys key_names)
	(begin
		(define pairs (group_stage_session_key_pairs stage keys key_names))
		(if (empty_list? pairs)
			true
			(combine_where_terms (map pairs (lambda (pair)
				(list (quote equal??)
					(list (quote get_column) grouptbl false (nth pair 1) false)
					(nth pair 0))))
				true)))))

(define group_stage_session_binding_missing_expr (lambda (stage schema grouptbl keys key_names)
	(begin
		(define pairs (group_stage_session_key_pairs stage keys key_names))
		(if (empty_list? pairs)
			(list (quote table_empty?) (list (quote table) schema grouptbl))
			(begin
				(define cols (map pairs (lambda (pair) (nth pair 1))))
				(define params (map cols (lambda (col) (symbol (concat grouptbl "." col)))))
				(list (quote not)
					(list (quote scan_exists)
						'(session "__memcp_tx")
						(list (quote table) schema grouptbl)
						(cons (quote list) cols)
						(list (quote lambda)
							params
							(list (quote optimize)
								(combine_where_terms (map (produceN (count pairs)) (lambda (i)
									(list (quote equal??) (nth params i) (nth (nth pairs i) 0))))
									true))))))))))

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
		(define session_keys (query_expr_session_reads block))
		(define explicit_keys (map (coalesceNil (qb_group block) '()) (lambda (expr)
			(canonical_column_expr_for_alias alias expr))))
		(define keys (merge (list explicit_keys (filter session_keys (lambda (expr)
			(not (contains? explicit_keys expr)))))))
		(make_group_stage
			(concat "group:" (source_relation src) ":" (fnv_hash (string (list keys ags))))
			src
			session_keys
			keys
			ags
			(qb_having block)
			(qb_fields block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(list
				(list (quote condition) (coalesceNil (qb_where block) true))
				(list (quote domain) session_keys)
				(list (quote lookup-keys) session_keys))))))

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
		(define session_keys (query_expr_session_reads block))
		(define keys (merge_unique (list group_keys field_passthrough_keys passthrough_keys session_keys)))
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
			(concat "group:query:" (fnv_hash (serialize (list
				(qb_sources block) (qb_where block) keys ags))))
			input
			session_keys
			keys
			ags
			(qb_having block)
			(qb_fields block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(list
				(list (quote condition) true)
				(list (quote domain) session_keys)
				(list (quote lookup-keys) session_keys)
				(list (quote stage_catalog) (query_block_stage_catalog block)))))))

/* Group rewriters walk original and canonical trees in lockstep. Prehash every
canonical root once so descendant lookup does not rescan keys or recanonicalize
ever-larger subtrees. */
(define make_group_key_index (lambda (keys roots)
	(make_structural_index keys roots)))

(define lookup_group_key_index (lambda (index resolved)
	(if (or (nil? resolved) (or (equal? resolved true) (equal? resolved false)))
		nil
		(index resolved))))

(define group_key_index (lambda (alias keys expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(lookup_group_key_index
			(make_group_key_index keys (list resolved))
			resolved))))

(define aggregate_count_like? (lambda (ag)
	(match ag
		'(expr (symbol +) 0) true
		'(expr (quote +) 0) true
		_ false)))

(define scalar_order_aggregate_parts (lambda (ag)
	(match ag
		'(((symbol scalar_order_value) value_expr order_exprs dirs offset_value) agg_reduce agg_neutral)
		(list value_expr order_exprs dirs (coalesceNil offset_value 0) agg_reduce agg_neutral)
		'(((quote scalar_order_value) value_expr order_exprs dirs offset_value) agg_reduce agg_neutral)
		(list value_expr order_exprs dirs (coalesceNil offset_value 0) agg_reduce agg_neutral)
		'(((symbol scalar_order_value) value_expr order_expr dir) agg_reduce agg_neutral)
		(list value_expr (list order_expr) (list dir) 0 agg_reduce agg_neutral)
		'(((quote scalar_order_value) value_expr order_expr dir) agg_reduce agg_neutral)
		(list value_expr (list order_expr) (list dir) 0 agg_reduce agg_neutral)
		_ nil)))

(define scalar_order_signature (lambda (parts)
	(list (nth parts 1) (nth parts 2) (nth parts 3) (nth parts 4) (nth parts 5))))

(define compatible_scalar_order_aggregates? (lambda (ags)
	(if (< (count ags) 2)
		false
		(begin
			(define first_parts (scalar_order_aggregate_parts (car ags)))
			(if (nil? first_parts)
				false
				(begin
					(define signature (scalar_order_signature first_parts))
					(reduce ags (lambda (ok ag)
						(and ok
							(begin
								(define parts (scalar_order_aggregate_parts ag))
								(and (not (nil? parts))
									(equal? (scalar_order_signature parts) signature)))))
						true)))))))

(define non_scalar_order_aggregates (lambda (ags)
	(filter ags (lambda (ag) (nil? (scalar_order_aggregate_parts ag))))))

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

(define replace_group_probe_stage_lookup_keys (lambda (alias grouptbl keys key_names ags key_index stage)
	(if (not (group_stage? stage))
		stage
		(begin
			(define facts (gs_facts stage))
			(define lookup_keys (qassoc_get facts (quote lookup-keys) '()))
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
				(qassoc_set facts (quote lookup-keys)
					(map lookup_keys (lambda (expr)
						(begin
							(define resolved (canonical_column_expr_for_alias alias expr))
							(replace_group_expr_indexed alias grouptbl keys key_names ags
								(make_group_key_index keys (list resolved)) expr resolved))))))))))

(define replace_group_probe_stages_lookup_keys (lambda (alias grouptbl keys key_names ags key_index stages)
	(map (coalesceNil stages '()) (lambda (stage)
		(replace_group_probe_stage_lookup_keys alias grouptbl keys key_names ags key_index stage)))))

(define replace_group_expr_tail_indexed (lambda (alias grouptbl keys key_names ags key_index items resolved_items)
	(match items
		(cons item rest)
		(cons
			(replace_group_expr_indexed alias grouptbl keys key_names ags key_index item (car resolved_items))
			(replace_group_expr_tail_indexed alias grouptbl keys key_names ags key_index rest (cdr resolved_items)))
		_ '())))

(define replace_group_expr_indexed (lambda (alias grouptbl keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol scalar_first_probe) stage requested_col stages)
				(list (quote scalar_first_probe)
					(replace_group_probe_stage_lookup_keys alias grouptbl keys key_names ags key_index stage)
					requested_col
					(replace_group_probe_stages_lookup_keys alias grouptbl keys key_names ags key_index stages))
				((quote scalar_first_probe) stage requested_col stages)
				(list (quote scalar_first_probe)
					(replace_group_probe_stage_lookup_keys alias grouptbl keys key_names ags key_index stage)
					requested_col
					(replace_group_probe_stages_lookup_keys alias grouptbl keys key_names ags key_index stages))
				((symbol scalar_aggregate_probe) stage requested_col)
				(list (quote scalar_aggregate_probe)
					(replace_group_probe_stage_lookup_keys alias grouptbl keys key_names ags key_index stage)
					requested_col)
				((quote scalar_aggregate_probe) stage requested_col)
				(list (quote scalar_aggregate_probe)
					(replace_group_probe_stage_lookup_keys alias grouptbl keys key_names ags key_index stage)
					requested_col)
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
				(cons head tail) (cons head
					(replace_group_expr_tail_indexed alias grouptbl keys key_names ags key_index tail (cdr resolved)))
				_ expr)))))

(define replace_group_expr (lambda (alias grouptbl keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(replace_group_expr_indexed
			alias grouptbl keys key_names ags (make_group_key_index keys (list resolved))
			expr resolved))))

(define replace_group_fields_indexed (lambda (alias grouptbl keys key_names ags key_index fields resolved_fields)
	(match fields
		(cons title (cons expr rest))
		(cons title (cons
			(replace_group_expr_indexed alias grouptbl keys key_names ags key_index expr (cadr resolved_fields))
			(replace_group_fields_indexed alias grouptbl keys key_names ags key_index rest (cdr (cdr resolved_fields)))))
		_ '())))

(define replace_group_order_expr_tail_indexed (lambda (alias grouptbl keys key_names ags key_index items resolved_items)
	(match items
		(cons item rest)
		(cons
			(replace_group_order_expr_indexed alias grouptbl keys key_names ags key_index item (car resolved_items))
			(replace_group_order_expr_tail_indexed alias grouptbl keys key_names ags key_index rest (cdr resolved_items)))
		_ '())))

(define replace_group_order_expr_indexed (lambda (alias grouptbl keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
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
				(cons head tail) (cons head
					(replace_group_order_expr_tail_indexed alias grouptbl keys key_names ags key_index tail (cdr resolved)))
				_ expr)))))

(define replace_group_order_expr (lambda (alias grouptbl keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(replace_group_order_expr_indexed
			alias grouptbl keys key_names ags (make_group_key_index keys (list resolved))
			expr resolved))))

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
			(if (query_session_read? (nth keys i))
				true
				(list (quote equal?)
					(lower_column_expr_for_alias src (nth keys i))
					(list (quote outer) (symbol (nth key_names i))))))))))

(define build_group_constant_key_insert_plan (lambda (schema grouptbl)
	(list (quote insert)
		(list (quote table) schema grouptbl)
		(quoted_runtime_list (list "k0"))
		(list (quote list) (list (quote list) 1))
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		true)))

(define build_group_session_key_insert_plan (lambda (schema grouptbl key_names keys)
	(list (quote insert)
		(list (quote table) schema grouptbl)
		(cons (quote list) key_names)
		(list (quote list) (cons (quote list) keys))
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		true)))

(define grouped_state_merge_expr (lambda (merge_payload)
	(list (quote lambda) (list (quote acc) (quote grouped))
		(list (quote merge_assoc) (quote acc) (quote grouped) merge_payload))))

(define build_query_group_collect_plan (lambda (input grouptbl keys key_names)
	(begin
		(define schema (qb_schema input))
		(define key_fields (merge (map (produceN (count keys)) (lambda (i)
			(list (nth key_names i) (nth keys i))))))
		(define keep_old (list (quote lambda) (list (quote old) (quote _new)) (quote old)))
		(define combine_grouped (grouped_state_merge_expr keep_old))
		(list
			(list (quote lambda) (list (quote grouped))
				(group_insert_batches_expr schema grouptbl key_names '() (quote grouped)))
			(lower_query_block_as_dataset_reduce
				input
				key_fields
				(list (quote lambda)
					(map key_names (lambda (col) (symbol col)))
					(runtime_cons_list_expr (map key_names (lambda (col) (symbol col)))))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc) (quote acc) (quote rowvals) (list (quote list))))
				(list (quote list))
				combine_grouped)))))

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

(define build_group_ordered_scalar_columns_insert_plan (lambda (schema tbl alias grouptbl keys key_names condition ags)
	(begin
		(define src (list alias schema tbl false nil))
		(define parts (map ags scalar_order_aggregate_parts))
		(define first_parts (car parts))
		(define value_exprs (map parts (lambda (part) (nth part 0))))
		(define order_exprs (nth first_parts 1))
		(define dirs (nth first_parts 2))
		(define offset_value (nth first_parts 3))
		(define agg_cols (map ags aggregate_col_name))
		(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
		(define condition_cols (extract_columns_for_alias src condition))
		(define order_cols (map order_exprs (lambda (order_expr) (order_column_for_alias src order_expr))))
		(define valuecols (merge_unique (map value_exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols order_cols)))
		(define mapcols (merge_unique (list group_key_cols_for_scan valuecols)))
		(define key_expr (runtime_cons_list_expr (map keys (lambda (expr) (lower_column_expr_for_alias src expr)))))
		(define payload_expr (runtime_cons_list_expr (map value_exprs (lambda (expr) (lower_column_expr_for_alias src expr)))))
		(define physical_keys (filter keys (lambda (expr) (not (query_session_read? expr)))))
		(define key_sortcols (map physical_keys (lambda (expr) (order_column_for_alias src expr))))
		(define key_dirs (map physical_keys (lambda (_key) (quote <))))
		(define keep_first (list (quote lambda) (list (quote old) (quote new)) (quote old)))
		(list
			(list (quote lambda) (list (quote grouped))
				(group_insert_finish_expr schema grouptbl key_names agg_cols))
			(list (quote scan_order)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols (lambda (col) (symbol (concat alias "." col))))
					(list (quote optimize) (lower_column_expr_for_alias src condition)))
				(cons (quote list) (merge (list key_sortcols order_cols)))
				(cons (quote list) (merge (list key_dirs dirs)))
				0
				(coalesceNil offset_value 0)
				-1
				(cons (quote list) mapcols)
				(list (quote lambda)
					(map mapcols (lambda (col) (symbol (concat alias "." col))))
					(runtime_cons_list_expr (list key_expr payload_expr)))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc)
						(quote acc)
						(list (quote car) (quote rowvals))
						(list (quote cadr) (quote rowvals))
						keep_first))
				(list (quote list))
				false)))))

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
				(list (quote list))
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
		(list (quote createcolumn)
			(list (quote table) schema grouptbl)
			agg_col
			"any"
			(quoted_runtime_list '())
			(quoted_runtime_list '("temp" true))
			(cons (quote list) key_names)
			(list (quote lambda)
				(map key_names (lambda (col) (symbol col)))
				(lower_query_block_as_dataset_reduce
					input
					row_fields
					(list (quote lambda)
						(extract_assoc row_fields (lambda (title _expr) (symbol title)))
						(list (quote if)
							(list (quote optimize)
								(cons (quote and) (map (produceN (count key_names)) (lambda (i)
									(list (quote equal?) (symbol (nth row_key_names i)) (list (quote outer) (symbol (nth key_names i))))))))
							(aggregate_map_value_expr ag (symbol value_col))
							agg_neutral))
					agg_reduce
					agg_neutral
					(query_aggregate_shard_combine ag))))))))

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
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define value_symbol (symbol value_col))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define mapped_value (aggregate_map_value_expr ag value_symbol))
		(define payload_expr (runtime_cons_list_expr (list mapped_value)))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(runtime_cons_list_expr (list (list agg_reduce (list (quote car) (quote old)) (list (quote car) (quote new)))))))
		(define combine_grouped (grouped_state_merge_expr merge_payload))
		(list
			(list (quote lambda) (list (quote grouped))
				(group_insert_finish_expr schema grouptbl key_names (list agg_col)))
			(lower_query_block_as_dataset_reduce
				input
				row_fields
				(list (quote lambda)
					(extract_assoc row_fields (lambda (title _expr) (symbol title)))
					(runtime_cons_list_expr (list key_expr payload_expr)))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc)
						(quote acc)
						(list (quote car) (quote rowvals))
						(list (quote cadr) (quote rowvals))
						merge_payload))
				(list (quote list))
				combine_grouped))
		_ (neumann_fail "build_queryplan" "query-input aggregate insert expects aggregate descriptor")))))

(define direct_group_assoc_expr (lambda (key_names ags)
	(begin
		(define key_pairs (merge (map (produceN (count key_names)) (lambda (i)
			(list (nth key_names i) (list (quote nth) (quote row) i))))))
		(define agg_pairs (merge (map (produceN (count ags)) (lambda (i)
			(list (aggregate_col_name (nth ags i)) (list (quote nth) (quote row) (+ (count key_names) i)))))))
		(runtime_cons_list_expr (merge (list key_pairs agg_pairs))))))

(define direct_group_aggregate_read_expr (lambda (ag)
	(begin
		(define read_expr (list (quote get_assoc) (quote rowassoc) (aggregate_col_name ag)))
		(if (aggregate_count_like? ag)
			(list (quote coalesceNil) read_expr 0)
			read_expr))))

(define replace_direct_group_expr_tail_indexed (lambda (alias keys key_names ags key_index items resolved_items)
	(match items
		(cons item rest)
		(cons
			(replace_direct_group_expr_indexed alias keys key_names ags key_index item (car resolved_items))
			(replace_direct_group_expr_tail_indexed alias keys key_names ags key_index rest (cdr resolved_items)))
		_ '())))

(define replace_direct_group_expr_indexed (lambda (alias keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
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
				(cons head tail) (cons head
					(replace_direct_group_expr_tail_indexed alias keys key_names ags key_index tail (cdr resolved)))
				_ expr))))))

(define replace_direct_group_expr (lambda (alias keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(replace_direct_group_expr_indexed
			alias keys key_names ags (make_group_key_index keys (list resolved))
			expr resolved))))

(define direct_group_result_assoc_expr_indexed (lambda (alias keys key_names ags key_index fields resolved_fields)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
		(list (quote cons)
			(string title)
			(list (quote cons)
				(replace_direct_group_expr_indexed alias keys key_names ags key_index expr (cadr resolved_fields))
				(direct_group_result_assoc_expr_indexed alias keys key_names ags key_index rest
					(cdr (cdr resolved_fields)))))
		_ (list (quote list)))))

(define direct_group_result_assoc_expr (lambda (alias keys key_names ags fields)
	(begin
		(define items (coalesceNil fields '()))
		(define resolved_fields (map_assoc items (lambda (_title expr)
			(canonical_column_expr_for_alias alias expr))))
		(define key_index (make_group_key_index keys
			(extract_assoc resolved_fields (lambda (_title expr) expr))))
		(direct_group_result_assoc_expr_indexed alias keys key_names ags key_index items resolved_fields))))

(define direct_group_agg_index (lambda (ags expr)
	(reduce (produceN (count ags)) (lambda (found i)
		(if (not (nil? found))
			found
			(if (equal? expr (nth ags i)) i nil)))
		nil)))

(define direct_group_order_column_indexed (lambda (alias keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
		(if (not (nil? key_idx))
			(nth key_names key_idx)
			(match expr
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(begin
					(define agg_idx (direct_group_agg_index ags (list agg_expr agg_reduce agg_neutral)))
					(if (nil? agg_idx) nil (aggregate_col_name (nth ags agg_idx))))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(begin
					(define agg_idx (direct_group_agg_index ags (list agg_expr agg_reduce agg_neutral)))
					(if (nil? agg_idx) nil (aggregate_col_name (nth ags agg_idx))))
				_ nil)))))

(define direct_group_order_column (lambda (alias keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(direct_group_order_column_indexed alias keys key_names ags
			(make_group_key_index keys (list resolved)) expr resolved))))

(define direct_group_order_columns (lambda (alias keys key_names ags order_items)
	(begin
		(define items (coalesceNil order_items '()))
		(define resolved_items (map items (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias alias expr) dir)))))
		(define key_index (make_group_key_index keys (map resolved_items car)))
		(map (produceN (count items)) (lambda (i)
			(match (nth items i)
				'(expr _dir)
				(begin
					(define col (direct_group_order_column_indexed alias keys key_names ags key_index expr
						(car (nth resolved_items i))))
					(if (nil? col)
						(neumann_fail "build_queryplan" (concat "direct GROUP BY cannot order by " (serialize expr)))
						col))
				_ (neumann_fail "build_queryplan" "malformed ORDER BY item")))))))

(define direct_group_order_supported? (lambda (alias keys key_names ags order_items)
	(begin
		(define items (coalesceNil order_items '()))
		(define resolved_items (map items (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias alias expr) dir)))))
		(define key_index (make_group_key_index keys (map resolved_items car)))
		(reduce (produceN (count items)) (lambda (ok i)
			(and ok (match (nth items i)
				'(expr _dir) (not (nil? (direct_group_order_column_indexed
					alias keys key_names ags key_index expr (car (nth resolved_items i)))))
				_ false)))
			true))))

(define build_base_group_scan_assoc_plan (lambda (schema tbl alias table_expr keys condition ags)
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
			(list (quote merge_assoc) (quote acc) (quote grouped) merge_payload)))
		(list (quote scan)
			'(session "__memcp_tx")
			table_expr
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat alias "." col))))
				(list (quote optimize) (lower_column_expr_for_alias src condition)))
			(cons (quote list) mapcols)
			(list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat alias "." col))))
				(list (quote list) key_expr payload_expr))
			(list (quote lambda) (list (quote acc) (quote rowvals))
				(list (quote set_assoc)
					(quote acc)
					(list (quote car) (quote rowvals))
					(list (quote cadr) (quote rowvals))
					merge_payload))
			(list (quote list))
			merge_groups
			false))))

(define base_group_membership_parts (lambda (src condition)
	(begin
		(define membership (driver_membership_for_source src condition))
		(define table_expr (if (nil? membership)
			nil
			(recset_project_join_expr_for_membership src membership)))
		(list table_expr
			(strip_driver_membership_for_source src condition (if (nil? table_expr) nil membership))))))

(define build_base_group_into_plan (lambda (schema tbl alias src grouptbl keys key_names condition ags)
	(begin
		(define membership_parts (base_group_membership_parts src condition))
		(define membership_expr (car membership_parts))
		(define effective_condition (cadr membership_parts))
		(define membership_var (symbol "__group_membership_recset"))
		(define source_expr (if (nil? membership_expr) (source_table_expr src) membership_var))
		(define grouped_scan (build_base_group_scan_assoc_plan schema tbl alias source_expr keys effective_condition ags))
		(define grouped_expr (if (equal? keys '(1))
			(list (quote if)
				(list (quote equal?) (list (quote count) (quote grouped)) 0)
				(list (quote set_assoc)
					(quote grouped)
					(runtime_cons_list_expr keys)
					(runtime_cons_list_expr (map ags (lambda (ag) (nth ag 2)))))
				(quote grouped))
			(quote grouped)))
		(define plan (list
			(list (quote lambda) (list (quote grouped))
				(list
					(list (quote lambda) (list (quote grouped))
						(group_insert_finish_expr schema grouptbl key_names (map ags aggregate_col_name)))
					grouped_expr))
			grouped_scan))
		(if (nil? membership_expr)
			plan
			(list
				(list (quote lambda) (list membership_var) plan)
				membership_expr)))))

(define direct_group_rowassoc_from_params_expr (lambda (key_names ags)
	(begin
		(define key_pairs (merge (map key_names (lambda (col) (list col (symbol col))))))
		(define agg_pairs (merge (map ags (lambda (ag)
			(begin
				(define col (aggregate_col_name ag))
				(list col (symbol col)))))))
		(runtime_cons_list_expr (merge (list key_pairs agg_pairs))))))

(define direct_group_map_params (lambda (key_names ags)
	(map (merge (list key_names (map ags aggregate_col_name))) symbol)))

(define direct_group_assoc_from_key_payload_expr (lambda (key_names ags)
	(begin
		(define key_pairs (merge (map (produceN (count key_names)) (lambda (i)
			(list (nth key_names i) (list (quote nth) (quote __group_key) i))))))
		(define agg_pairs (merge (map (produceN (count ags)) (lambda (i)
			(list (aggregate_col_name (nth ags i)) (list (quote nth) (quote __group_payload) i))))))
		(runtime_cons_list_expr (merge (list key_pairs agg_pairs))))))

(define lower_direct_base_group_stage (lambda (stage fields order_items offset_value limit_value)
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
		(define normalized_order (coalesceNil order_items '()))
		(if (not (empty_list? normalized_order))
			(neumann_fail "build_queryplan" "direct GROUP BY requires unordered output")
			true)
		(define candidate_membership (driver_membership_for_source src condition))
		(define membership_stage (if (nil? candidate_membership) nil (nth candidate_membership 0)))
		(define membership (if (and (list? membership_stage)
			(and (group_stage? membership_stage) (reduce
				(qassoc_get (gs_facts membership_stage) (quote lookup-keys) '())
				(lambda (found key) (or found (expr_contains_session_dependency? key)))
				false)))
			candidate_membership
			nil))
		(define membership_table_expr (if (nil? membership)
			nil
			(recset_project_join_expr_for_membership src membership)))
		(define effective_membership (if (nil? membership_table_expr) nil membership))
		(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
		(define membership_var (symbol "__membership_recset"))
		(define table_expr (if (nil? membership_table_expr)
			(source_table_expr src)
			membership_var))
		(define grouped_scan (build_base_group_scan_assoc_plan schema tbl alias table_expr keys effective_condition ags))
		(define rowassoc_expr (direct_group_assoc_from_key_payload_expr key_names ags))
		(define having_expr (replace_direct_group_expr alias keys key_names ags (coalesceNil (gs_having stage) true)))
		(define emit_expr (list (quote resultrow)
			(direct_group_result_assoc_expr alias keys key_names ags fields)))
		(define group_reduce (list (quote lambda)
			(list (quote __accepted) (quote __group_key) (quote __group_payload))
			(list (quote begin)
				(list (quote define) (quote rowassoc) rowassoc_expr)
				(list (quote if) having_expr
					(list (quote if) (list (quote <) (quote __accepted) offset_expr)
						(list (quote +) (quote __accepted) 1)
						(list (quote if)
							(list (quote or)
								(list (quote equal?) limit_expr -1)
								(list (quote <) (list (quote -) (quote __accepted) offset_expr) limit_expr))
							(list (quote begin) emit_expr (list (quote +) (quote __accepted) 1))
							(quote __accepted)))
					(quote __accepted)))))
		(list
			(list (quote lambda) (list (quote grouped))
				(list (quote begin)
					(list (quote reduce_assoc) (quote grouped) group_reduce 0)
					nil))
			(if (nil? membership_table_expr)
				grouped_scan
				(list
					(list (quote lambda) (list membership_var) grouped_scan)
					membership_table_expr))))))

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
	(if (empty_list? value_cols)
		'(list "$update")
		(cons (quote list) (merge (list
			(map value_cols (lambda (col) (concat "$set:" col)))
			(map value_cols (lambda (col) (concat "NEW." col)))))))))

(define group_upsert_collision_lambda (lambda (value_cols)
	(if (empty_list? value_cols)
		(list (quote lambda)
			(list (quote $update))
			(list (quote $update) (cons (quote list) '())))
		(begin
			(define setter_params (map (produceN (count value_cols)) (lambda (i) (symbol (concat "__set_group_value_" i)))))
			(define new_params (map (produceN (count value_cols)) (lambda (i) (symbol (concat "__new_group_value_" i)))))
			(list (quote lambda)
				(merge (list setter_params new_params))
				(cons (quote begin) (map (produceN (count value_cols)) (lambda (i)
					(list (nth setter_params i) (nth new_params i))))))))))

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

(define group_insert_batch_size 4096)

(define group_insert_batches (lambda (target columns collision_cols collision_fn grouped)
	((lambda (state)
		(if (equal? (car state) 0)
			0
			(insert target columns (cadr state) collision_cols collision_fn true)))
		(reduce_assoc grouped
			(lambda (state key payload)
				(begin
					(define count (car state))
					(define rows (cons (merge (list key payload)) (cadr state)))
					(if (>= (+ count 1) group_insert_batch_size)
						(begin
							(insert target columns rows collision_cols collision_fn true)
							(list 0 (list)))
						(list (+ count 1) rows))))
			(list 0 (list))))))

(define group_insert_batches_expr (lambda (schema grouptbl key_names value_cols grouped_expr)
	(list (quote group_insert_batches)
		(list (quote table) schema grouptbl)
		(cons (quote list) (merge (list key_names value_cols)))
		(group_upsert_collision_cols value_cols)
		(group_upsert_collision_lambda value_cols)
		grouped_expr)))

(define group_insert_finish_expr (lambda (schema grouptbl key_names value_cols)
	(list (quote !begin)
		(group_cleanup_missing_keys_plan schema grouptbl key_names)
		(group_insert_batches_expr schema grouptbl key_names value_cols (quote grouped)))))

(define build_query_group_aggregates_insert_plan_using (lambda (input grouptbl keys key_names ags output_ags)
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
		(define key_symbols (map row_key_names (lambda (col) (symbol col))))
		(define value_symbols (map value_cols (lambda (col) (symbol col))))
		(define key_expr (runtime_cons_list_expr key_symbols))
		(define payload_expr (runtime_cons_list_expr (map (produceN (count ags)) (lambda (i)
			(if (equal? (nth ags i) aggregate_count_descriptor)
				1
				(aggregate_map_value_expr (nth ags i) (nth value_symbols i)))))))
		(define merge_payload (list (quote lambda) (list (quote old) (quote new))
			(aggregate_payload_merge_expr ags 0)))
		(define finish_expr (group_insert_finish_expr schema grouptbl key_names (map output_ags aggregate_col_name)))
		(define combine_grouped (grouped_state_merge_expr merge_payload))
		(list
			(list (quote lambda) (list (quote grouped)) finish_expr)
			(lower_query_block_as_dataset_reduce
				input
				row_fields
				(list (quote lambda)
					(extract_assoc row_fields (lambda (title _expr) (symbol title)))
					(runtime_cons_list_expr (list key_expr payload_expr)))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc)
						(quote acc)
						(list (quote car) (quote rowvals))
						(list (quote cadr) (quote rowvals))
						merge_payload))
				(list (quote list))
				combine_grouped)))))

(define build_query_group_aggregates_insert_plan (lambda (input grouptbl keys key_names ags)
	(build_query_group_aggregates_insert_plan_using input grouptbl keys key_names ags ags)))

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

(define lower_union_block_as_dataset_reduce (lambda (block keys key_names ags value_cols row_mapper reduce_expr neutral_expr shard_reduce_expr)
	(begin
		(define candidate_alias (qassoc_get (union_facts block) (quote alias) "__union"))
		(define branches (union_branches block))
		(reduce branches (lambda (acc branch)
			(list shard_reduce_expr
				acc
				(lower_query_block_as_dataset_reduce
					branch
					(union_branch_group_row_fields candidate_alias branch keys key_names ags value_cols)
					row_mapper reduce_expr neutral_expr shard_reduce_expr)))
			neutral_expr))))

(define build_union_group_aggregates_insert_plan (lambda (input grouptbl keys key_names ags)
	(begin
		(define schema (qb_schema (car (union_branches input))))
		(define row_key_names key_names)
		(define value_cols (map (produceN (count ags)) (lambda (i) (concat "__agg" i))))
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
		(define row_mapper (list (quote lambda)
			(merge (list key_symbols value_symbols))
			(runtime_cons_list_expr (list key_expr payload_expr))))
		(define reduce_expr (list (quote lambda) (list (quote acc) (quote rowvals))
			(list (quote set_assoc)
				(quote acc)
				(list (quote car) (quote rowvals))
				(list (quote cadr) (quote rowvals))
				merge_payload)))
		(define neutral_expr (list (quote list)))
		(define combine_grouped (grouped_state_merge_expr merge_payload))
		(list
			(list (quote lambda) (list (quote grouped)) finish_expr)
			(lower_union_block_as_dataset_reduce
				input keys row_key_names ags value_cols row_mapper reduce_expr neutral_expr combine_grouped)))))

(define build_scalar_single_query_stage_fill_plan (lambda (input grouptbl keys key_names value_ag count_ag)
	(match value_ag '(value_expr _value_reduce _value_neutral) (begin
		(define prepared_input (if (and (query_block? input) (not (empty_list? (qb_stages input))))
			(query_block_without_stages_after_eager_prepare_using (qb_stages input) input)
			input))
		(define schema (qb_schema prepared_input))
		(define value_col (aggregate_col_name value_ag))
		(define count_col (aggregate_col_name count_ag))
		(define payload_col "__agg")
		(define row_key_names (map key_names (lambda (col) (concat "__row_" col))))
		(define row_fields (merge (list
			(merge (map (produceN (count keys)) (lambda (i)
				(list (nth row_key_names i) (nth keys i)))))
			(list payload_col value_expr))))
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
		(define stage_catalog (if (query_block? prepared_input) (query_block_stage_catalog prepared_input) '()))
		(define combine_grouped (grouped_state_merge_expr merge_payload))
		(define scan_plan (list
			(list (quote lambda) (list (quote grouped))
				(group_insert_finish_expr schema grouptbl key_names (list value_col count_col)))
			(lower_query_block_as_dataset_reduce
				prepared_input
				row_fields
				(list (quote lambda)
					(extract_assoc row_fields (lambda (title _expr) (symbol title)))
					(runtime_cons_list_expr (list key_expr payload_expr)))
				(list (quote lambda) (list (quote acc) (quote rowvals))
					(list (quote set_assoc)
						(quote acc)
						(list (quote car) (quote rowvals))
						(list (quote cadr) (quote rowvals))
						merge_payload))
				(list (quote list))
				combine_grouped)))
		(if (empty_list? stage_catalog)
			scan_plan
			(cons (quote !begin)
				(merge (list
					(lazy_stage_prepare_bindings stage_catalog (filter stage_catalog group_stage?))
					(list scan_plan))))))
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
	(begin
		(define alias (group_stage_input_alias stage))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define order_items (coalesceNil (gs_order stage) '()))
		(define resolved_order_exprs (map order_items (lambda (item)
			(match item '(expr _dir) (canonical_column_expr_for_alias alias expr)))))
		(define key_index (make_group_key_index keys resolved_order_exprs))
		(make_query_block
			(group_stage_schema stage)
			(list (list grouptbl (group_stage_schema stage) grouptbl false nil))
			(gs_output stage)
			having_expr
			nil nil
			(map (produceN (count order_items)) (lambda (i)
				(match (nth order_items i) '(expr dir) (list
					(replace_group_expr_indexed alias grouptbl keys key_names ags key_index expr
						(nth resolved_order_exprs i))
					dir))))
			(gs_limit stage)
			(gs_offset stage)
			'() '() '()))))

(define rewrite_source_for_group_domain (lambda (alias grouptbl keys key_names ags src)
	(begin
		(define resolved (canonical_column_expr_for_alias alias (source_join_expr src)))
		(rewrite_source_for_group_domain_indexed alias grouptbl keys key_names ags
			(make_group_key_index keys (list resolved)) src resolved))))

(define rewrite_source_for_group_domain_indexed (lambda (alias grouptbl keys key_names ags key_index src resolved_join)
	(list
		(source_alias src)
		(source_schema src)
		(source_relation src)
		(source_outer? src)
		(replace_group_expr_indexed alias grouptbl keys key_names ags key_index
			(source_join_expr src)
			resolved_join))))

(define lowering_catalog? (lambda (catalog)
	(match catalog
		((symbol lowering-catalog) _stages _id_index _group_cache_index _parent) true
		((quote lowering-catalog) _stages _id_index _group_cache_index _parent) true
		_ false)))

(define lowering_catalog_stages (lambda (catalog)
	(if (not (lowering_catalog? catalog))
		catalog
		(begin
			(define local_stages (nth catalog 1))
			(define parent (nth catalog 4))
			(if (lowering_catalog? parent)
				(unique_stages_by_id (merge (list local_stages (lowering_catalog_stages parent))))
				local_stages)))))

(define lowering_catalog_id_index (lambda (catalog) (nth catalog 2)))
(define lowering_catalog_group_cache_index (lambda (catalog) (nth catalog 3)))
(define lowering_catalog_parent (lambda (catalog) (nth catalog 4)))

(define make_indexed_lowering_catalog (lambda (stages parent)
	(list
		(quote lowering-catalog)
		stages
		(stage_dependency_id_index stages)
		(stage_dependency_group_cache_index stages)
		parent)))

(define make_lowering_catalog (lambda (stages)
	(begin
		(define available_stages (coalesceNil stages '()))
		/* Below this measured crossover, sequential list lookup costs less than
		building and hashing two immutable indexes. */
		(if (<= (count available_stages) 24)
			available_stages
			(make_indexed_lowering_catalog available_stages nil)))))

(define lowering_catalog_with_local_stages (lambda (catalog stages)
	(begin
		(define local_stages (if (lowering_catalog? catalog)
			(filter (coalesceNil stages '()) (lambda (stage)
				(if (group_stage? stage)
					(nil? (stage_by_id catalog (gs_id stage)))
					true)))
			(coalesceNil stages '())))
		(if (empty_list? local_stages)
			catalog
			(if (lowering_catalog? catalog)
				(make_indexed_lowering_catalog local_stages catalog)
				(unique_stages_by_id (merge (list local_stages catalog))))))))

(define stage_by_id (lambda (stages stage_id)
	(if (lowering_catalog? stages)
		(begin
			(define local (get_assoc (lowering_catalog_id_index stages) stage_id))
			(if (not (nil? local))
				local
				(begin
					(define parent (lowering_catalog_parent stages))
					(if (lowering_catalog? parent) (stage_by_id parent stage_id) nil))))
		(reduce (coalesceNil stages '()) (lambda (found stage)
			(if (not (nil? found))
				found
				(if (and (group_stage? stage) (equal? (gs_id stage) stage_id))
					stage
					nil)))
			nil))))

(define stage_for_output_relation (lambda (stages relation)
	(if (not (stage_output_relation? relation))
		nil
		(stage_by_id stages (stage_output_relation_id relation)))))

(define physicalize_stage_output_source (lambda (stages src)
	(begin
		(define relation (source_relation src))
		(if (not (stage_output_relation? relation))
			src
			(begin
				(define id (stage_output_relation_id relation))
				(define stage (stage_for_output_relation stages relation))
				(if (nil? stage)
					(neumann_fail "build_queryplan" (concat "physicalize stage-output source references unknown stage " id))
					true)
				(source_with_schema_relation
					src
					(group_stage_cache_schema stage)
					(group_stage_cache_relation stage)))))))

(define physicalize_stage_output_sources (lambda (stages sources)
	(map (coalesceNil sources '()) (lambda (src)
		(physicalize_stage_output_source stages src)))))

(define stage_outputs_from_sources_using (lambda (stages sources)
	(unique_stages_by_id (filter (map (coalesceNil sources '()) (lambda (src)
		(match src
			'(_alias _schema relation _outer _join_expr)
			(if (not (stage_output_relation? relation))
				nil
				(begin
					(define id (stage_output_relation_id relation))
					(define stage (stage_for_output_relation stages relation))
					(if (nil? stage)
						(neumann_fail "build_queryplan" (concat "dependency stage-output source references unknown stage " id))
						stage)))
			_ nil)))
		(lambda (stage) (not (nil? stage)))))))

(define available_stage_outputs_from_sources_using (lambda (stages sources)
	(filter (map (coalesceNil sources '()) (lambda (src)
		(if (stage_output_relation? (source_relation src))
			(stage_for_output_relation stages (source_relation src))
			nil)))
		(lambda (stage) (not (nil? stage))))))

(define source_stage_output_stage (lambda (stages src)
	(if (not (stage_output_relation? (source_relation src)))
		nil
		(stage_for_output_relation stages (source_relation src)))))

(define constant_scalar_or_presence_stage_output_source? (lambda (stages src)
	(begin
		(define stage (if (stage_output_relation? (source_relation src))
			(source_stage_output_stage stages src)
			(stage_for_group_cache_source stages src)))
		(define purpose (if (group_stage? stage)
			(qassoc_get (gs_facts stage) (quote purpose) nil)
			nil))
		(and (group_stage? stage)
			(and (or (equal? purpose (quote scalar_single)) (equal? purpose (quote exists)))
				(and (empty_list? (gs_domain stage))
					(not (stage_has_residual_outer_refs? stage))))))))

(define scalar_first_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(scalar_first_probe_stage? stage)))))

(define scalar_aggregate_probe_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(scalar_aggregate_probe_stage? stage)))))

(define scalar_cardinality_probe_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(scalar_cardinality_probe_stage? stage)))))

/* A selective LEFT JOIN may evaluate a grouped base-table source as a keyed
aggregate scan. Keep every ambiguous outer-join shape on the shared group cache. */
(define direct_group_probe_input (lambda (stage)
	(begin
		(define input (if (group_stage? stage) (gs_input stage) nil))
		(if (and (query_block? input)
			(and (single_source? (qb_sources input))
				(and (source_is_base_table? (car (qb_sources input)))
					(and (empty_list? (qb_stages input))
						(and (empty_list? (qb_group input))
							(and (nil? (qb_having input))
								(and (empty_list? (qb_order input))
									(and (nil? (qb_limit input)) (nil? (qb_offset input))))))))))
			input
			nil))))

(define direct_group_probe_key_ref? (lambda (stage_alias key_name expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(and (equal?? tblvar stage_alias) (equal?? col key_name))
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(and (equal?? tblvar stage_alias) (equal?? col key_name))
		_ false)))

(define direct_group_probe_lookup_from_term (lambda (stage_alias key_name term)
	(match term
		'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
			(if (direct_group_probe_key_ref? stage_alias key_name left)
				(if (expr_refs_alias? nil stage_alias right) nil right)
				(if (direct_group_probe_key_ref? stage_alias key_name right)
					(if (expr_refs_alias? nil stage_alias left) nil left)
					nil))
			nil)
		_ nil)))

(define direct_group_probe_lookup_keys (lambda (stage src)
	(begin
		(define terms (split_and_terms (coalesceNil (source_join_expr src) true)))
		(define key_names (group_key_cols (gs_keys stage)))
		(define lookups (map key_names (lambda (key_name)
			(reduce terms (lambda (found term)
				(coalesceNil found (direct_group_probe_lookup_from_term (source_alias src) key_name term)))
				nil))))
		(if (and (equal? (count terms) (count key_names))
			(reduce lookups (lambda (complete lookup) (and complete (not (nil? lookup)))) true))
			lookups
			nil))))

(define direct_group_probe_expr_refs_key? (lambda (stage_alias key_names expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(and (equal?? tblvar stage_alias) (contains? key_names col))
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(and (equal?? tblvar stage_alias) (contains? key_names col))
		(cons head tail) (or
			(direct_group_probe_expr_refs_key? stage_alias key_names head)
			(reduce tail (lambda (found item)
				(or found (direct_group_probe_expr_refs_key? stage_alias key_names item))) false))
		_ false)))

(define direct_group_probe_consumers_safe? (lambda (stage src consumers)
	(not (direct_group_probe_expr_refs_key?
		(source_alias src)
		(group_key_cols (gs_keys stage))
		consumers))))

(define source_join_consumers_except (lambda (sources excluded_src)
	(map
		(filter (coalesceNil sources '()) (lambda (src)
			(not (equal? (source_alias src) (source_alias excluded_src)))))
		source_join_expr)))

(define direct_group_probe_stage_for_block_source (lambda (stages sources src consumers)
	(if (expr_refs_alias? nil (source_alias src) (source_join_consumers_except sources src))
		nil
		(direct_group_probe_stage_for_source stages src consumers))))

(define direct_group_probe_aggregates_safe? (lambda (stage)
	(reduce (stage_aggregates_for_fields (gs_output stage)) (lambda (safe ag)
		(and safe (nil? (nth ag 2)))) true)))

(define direct_group_probe_stage_for_source (lambda (stages src consumers)
	(if (or (nil? consumers)
		(not (and (source_outer? src) (stage_output_relation? (source_relation src)))))
		nil
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(define input (direct_group_probe_input stage))
			(define lookups (if (nil? input) nil (direct_group_probe_lookup_keys stage src)))
			(if (or (not (nil? (qassoc_get (gs_facts stage) (quote purpose) nil)))
				(or (nil? input)
					(or (nil? lookups)
						(or (not (equal? (coalesceNil (gs_having stage) true) true))
							(or (not (direct_group_probe_aggregates_safe? stage))
								(not (direct_group_probe_consumers_safe? stage src consumers)))))))
				nil
				(begin
					(define input_src (car (qb_sources input)))
					(define condition (combine_where (qb_where input) (source_join_expr input_src)))
					(define facts (qassoc_set
						(qassoc_set
							(qassoc_set
								(qassoc_set
									(gs_facts stage)
									(quote purpose)
									(quote scalar_aggregate))
								(quote cardinality_mode)
								(quote many))
							(quote lookup-keys) lookups)
						(quote direct_group_probe) true))
					(make_group_stage
						(gs_id stage)
						input_src
						(gs_domain stage)
						(gs_keys stage)
						(gs_aggregates stage)
						(gs_having stage)
						(gs_output stage)
						(gs_order stage)
						(gs_limit stage)
						(gs_offset stage)
						(qassoc_set facts (quote condition) condition)))))))))

(define presence_stage_output_source? (lambda (stages src)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(presence_probe_stage? stage)))))

(define probeable_stage_output_source? (lambda (stages src)
	(or
		(scalar_first_stage_output_source? stages src)
		(presence_stage_output_source? stages src))))

(define list_index_of_from (lambda (items value offset)
	(begin
		(define rest (coalesceNil items '()))
		(if (empty_list? rest)
			-1
			(if (equal? (car rest) value)
				offset
				(list_index_of_from (cdr rest) value (+ offset 1)))))))

(define list_index_of (lambda (items value)
	(list_index_of_from items value 0)))

(define scalar_first_stage_key_lookup_expr (lambda (stage col)
	(begin
		(define key_names (group_key_cols (gs_keys stage)))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(define idx (list_index_of key_names col))
		(if (and (>= idx 0) (< idx (count lookup_keys)))
			(nth lookup_keys idx)
			nil))))

(define probe_stage_alias_key (lambda (alias insensitive)
	(list
		(if insensitive (quote insensitive) (quote exact))
		(if (and insensitive (string? alias)) (toLower alias) alias))))

(define probe_stage_alias_index_using_graph (lambda (stages dependency_graph sources consumers)
	(begin
		(define entries (filter (map (coalesceNil sources '()) (lambda (src)
			(begin
				(define original (source_stage_output_stage stages src))
				(define direct (direct_group_probe_stage_for_block_source stages sources src consumers))
				(define stage (coalesceNil direct original))
				(if (or (scalar_or_presence_probe_stage? stage)
					(or (scalar_aggregate_probe_stage? stage)
						(scalar_cardinality_probe_stage? stage)))
					(list src stage)
					nil))))
			(lambda (entry) (not (nil? entry)))))
		(define probe_stages (unique_stages_by_id (map entries (lambda (entry) (nth entry 1)))))
		(define closures (stage_dependency_closure_index_using_graph dependency_graph probe_stages))
		(reduce entries (lambda (index entry)
			(begin
				(define src (nth entry 0))
				(define stage (nth entry 1))
				(define direct (qassoc_get (gs_facts stage) (quote direct_group_probe) false))
				(begin
					(define index_entry (list stage (if direct
						(list stage)
						(get_assoc closures (logical_stage_key stage)))))
					(define exact_key (probe_stage_alias_key (source_alias src) false))
					(define insensitive_key (probe_stage_alias_key (source_alias src) true))
					(define with_exact (if (has_assoc? index exact_key)
						index
						(set_assoc index exact_key index_entry)))
					(if (has_assoc? with_exact insensitive_key)
						with_exact
						(set_assoc with_exact insensitive_key index_entry))))) '()))))

(define probe_stage_alias_index (lambda (stages sources consumers)
	(probe_stage_alias_index_using_graph
		stages (stage_dependency_graph stages) sources consumers)))

(define probe_stage_entry_for_alias_using_index (lambda (index default_alias tblvar tbl_ignorecase)
	(get_assoc index
		(probe_stage_alias_key (resolve_column_alias tblvar default_alias) tbl_ignorecase))))

(define rewrite_scalar_first_probe_expr_using_index (lambda (stages index default_alias expr)
	(match expr
		((symbol if)
			((symbol >) ((symbol coalesceNil) ((symbol get_column) count_alias count_tbl_ic count_col count_col_ic) 0) 1)
			((symbol error) message)
			((symbol get_column) value_alias value_tbl_ic value_col value_col_ic)) (begin
			(define entry (probe_stage_entry_for_alias_using_index index default_alias count_alias count_tbl_ic))
			(if (or (nil? entry)
				(or (not (equal?? count_alias value_alias))
					(or (not (equal? message "scalar subselect returned more than one row"))
						(not (equal? count_col (aggregate_col_name aggregate_count_descriptor))))))
				expr
				(begin
					(define stage (nth entry 0))
					(if (and (scalar_cardinality_probe_stage? stage)
						(not (nil? (scalar_first_probe_aggregate stage value_col))))
						(scalar_cardinality_probe_expr stage value_col)
						expr))))
		((symbol get_column) tblvar tbl_ignorecase col _col_ignorecase) (begin
			(define entry (probe_stage_entry_for_alias_using_index index default_alias tblvar tbl_ignorecase))
			(if (nil? entry)
				expr
				(begin
					(define stage (nth entry 0))
					(define dependencies (nth entry 1))
					(if (scalar_aggregate_probe_stage? stage)
						(coalesceNil
							(scalar_first_stage_key_lookup_expr stage col)
							(scalar_aggregate_probe_expr stage col))
						(coalesceNil
							(scalar_first_stage_key_lookup_expr stage col)
							(scalar_first_probe_expr stage col dependencies))))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(rewrite_scalar_first_probe_expr_using_index stages index default_alias
			(list (quote get_column) tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons head (map tail (lambda (item)
			(rewrite_scalar_first_probe_expr_using_index stages index default_alias item))))
		_ expr)))

(define rewrite_scalar_first_probe_expr (lambda (stages sources default_alias expr)
	(rewrite_scalar_first_probe_expr_using_index stages
		(probe_stage_alias_index stages sources nil) default_alias expr)))

(define rewrite_scalar_first_probe_fields_using_index (lambda (stages index default_alias fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(rewrite_scalar_first_probe_expr_using_index stages index default_alias expr)))))

(define rewrite_scalar_first_probe_fields (lambda (stages sources default_alias fields)
	(map_assoc (coalesceNil fields '()) (lambda (_title expr)
		(rewrite_scalar_first_probe_expr stages sources default_alias expr)))))

(define rewrite_scalar_first_probe_order (lambda (stages sources default_alias order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (list (rewrite_scalar_first_probe_expr stages sources default_alias expr) dir)
			_ item)))))

(define rewrite_scalar_first_probe_order_using_index (lambda (stages index default_alias order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (list (rewrite_scalar_first_probe_expr_using_index stages index default_alias expr) dir)
			_ item)))))

(define rewrite_scalar_first_probe_aggregate (lambda (stages sources default_alias ag)
	(match ag
		'(((symbol scalar_order_value) value_expr order_exprs dirs offset_value) reduce neutral)
		(list
			(list (quote scalar_order_value)
				(rewrite_scalar_first_probe_expr stages sources default_alias value_expr)
				(map order_exprs (lambda (expr) (rewrite_scalar_first_probe_expr stages sources default_alias expr)))
				dirs
				offset_value)
			reduce
			neutral)
		'(((quote scalar_order_value) value_expr order_exprs dirs offset_value) reduce neutral)
		(rewrite_scalar_first_probe_aggregate stages sources default_alias (list
			(list (quote scalar_order_value) value_expr order_exprs dirs offset_value)
			reduce
			neutral))
		'(((symbol scalar_order_value) value_expr order_expr dir) reduce neutral)
		(list
			(list (quote scalar_order_value)
				(rewrite_scalar_first_probe_expr stages sources default_alias value_expr)
				(list (rewrite_scalar_first_probe_expr stages sources default_alias order_expr))
				(list dir)
				0)
			reduce
			neutral)
		'(((quote scalar_order_value) value_expr order_expr dir) reduce neutral)
		(rewrite_scalar_first_probe_aggregate stages sources default_alias (list
			(list (quote scalar_order_value) value_expr order_expr dir)
			reduce
			neutral))
		'(expr reduce neutral)
		(list (rewrite_scalar_first_probe_expr stages sources default_alias expr) reduce neutral)
		_ ag)))

(define rewrite_scalar_first_probe_aggregates (lambda (stages sources default_alias ags)
	(map (coalesceNil ags '()) (lambda (ag)
		(rewrite_scalar_first_probe_aggregate stages sources default_alias ag)))))

(define rewrite_scalar_first_probe_sources (lambda (stages sources default_alias)
	(map (coalesceNil sources '()) (lambda (src)
		(list
			(source_alias src)
			(source_schema src)
			(source_relation src)
			(source_outer? src)
			(rewrite_scalar_first_probe_expr stages sources default_alias (source_join_expr src)))))))

(define rewrite_scalar_first_probe_sources_using (lambda (stages sources rewrite_sources default_alias)
	(map (coalesceNil sources '()) (lambda (src)
		(list
			(source_alias src)
			(source_schema src)
			(source_relation src)
			(source_outer? src)
			(rewrite_scalar_first_probe_expr stages rewrite_sources default_alias (source_join_expr src)))))))

(define rewrite_scalar_first_probe_sources_using_index (lambda (stages sources index default_alias)
	(map (coalesceNil sources '()) (lambda (src)
		(list
			(source_alias src)
			(source_schema src)
			(source_relation src)
			(source_outer? src)
			(rewrite_scalar_first_probe_expr_using_index stages index default_alias (source_join_expr src)))))))

(define sources_without_scalar_first_outputs (lambda (stages sources)
	(filter (coalesceNil sources '()) (lambda (src)
		(and
			(not (probeable_stage_output_source? stages src))
			(not (scalar_aggregate_probe_stage_output_source? stages src)))))))

(define stage_lookup_expr_resolves_in_sources? (lambda (sources default_alias expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase _col _col_ignorecase)
		(not (nil? (source_for_alias sources default_alias tblvar tbl_ignorecase)))
		((quote get_column) tblvar tbl_ignorecase _col _col_ignorecase)
		(not (nil? (source_for_alias sources default_alias tblvar tbl_ignorecase)))
		(cons _head tail) (reduce tail (lambda (ok item)
			(and ok (stage_lookup_expr_resolves_in_sources? sources default_alias item)))
			true)
		_ true)))

(define stage_lookup_keys_resolve_in_sources? (lambda (stage sources default_alias)
	(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (ok key)
		(and ok (stage_lookup_expr_resolves_in_sources? sources default_alias key)))
		true)))

(define probe_context_row_count (lambda (sources)
	(planner_add_estimates (map (coalesceNil sources '()) planner_source_row_count))))

(define probe_context_small_enough? (lambda (sources)
	(begin
		(define rows (probe_context_row_count sources))
		(or (nil? rows) (< rows 1000)))))

(define probe_limit_small_enough? (lambda (limit_value)
	(and (number? limit_value)
		(and (> limit_value 0)
			(<= limit_value 512)))))

(define source_column_bound_by_equality? (lambda (src col condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(if found
			true
			(match term
				'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
					(or
						(and (equal?? (direct_column_name_for_alias src left) col)
							(not (expr_refs_alias? (source_alias src) (source_alias src) right)))
						(and (equal?? (direct_column_name_for_alias src right) col)
							(not (expr_refs_alias? (source_alias src) (source_alias src) left))))
					false)
				_ false)))
		false)))

(define source_unique_point_condition? (lambda (src condition)
	(if (not (source_is_base_table? src))
		false
		(try
			(lambda ()
				(begin
					(define info (show (source_schema src) (source_relation src) true))
					(define uniques ((info "meta") "Unique"))
					(reduce uniques (lambda (found unique_key)
						(or found
							(reduce (unique_key "Cols") (lambda (complete col)
								(and complete (source_column_bound_by_equality? src col condition)))
								true)))
						false)))
			(lambda (_e) false)))))

(define probe_context_unique_point? (lambda (sources default_alias condition)
	(if (not (single_source? sources))
		false
		(begin
			(define src (car sources))
			(define combined (combine_where condition (source_join_expr src)))
			(and
				(stage_lookup_expr_resolves_in_sources? sources default_alias combined)
				(source_unique_point_condition? src combined))))))

(define scalar_aggregate_probe_aggregate_safe? (lambda (ag)
	(or
		(aggregate_count_like? ag)
		(nil? (nth ag 2)))))

(define scalar_aggregate_probe_stage_safe? (lambda (stage)
	(and (empty_list? (expr_probe_stages (list
		(qassoc_get (gs_facts stage) (quote condition) true)
		(gs_aggregates stage))))
		(reduce (gs_aggregates stage) (lambda (ok ag)
			(and ok (scalar_aggregate_probe_aggregate_safe? ag)))
			true))))

(define constant_scalar_aggregate_probe_sources? (lambda (stages sources)
	(and (not (empty_list? sources))
		(reduce sources (lambda (constant_only src)
			(if (not constant_only)
				false
				(begin
					(define stage (source_stage_output_stage stages src))
					(and (scalar_aggregate_probe_stage? stage)
						(and (empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
							(scalar_aggregate_probe_stage_safe? stage))))))
			true))))

(define stage_input_small_enough? (lambda (stage)
	(begin
		(define rows (planner_stage_input_rows (gs_input stage)))
		(and (not (nil? rows)) (< rows 1000)))))

(define stage_probe_allowed_in_context? (lambda (stage sources)
	(if (stage_has_residual_outer_refs? stage)
		true
		(if (not (stage_keys_are_input_local? stage))
			true
			(if (presence_probe_stage? stage)
				(or
					(stage_input_small_enough? stage)
					(probe_context_small_enough? sources))
				(if (query_block? (gs_input stage))
					(probe_context_small_enough? sources)
					true))))))

(define probeable_stage_output_source_for_block? (lambda (stages sources default_alias limit_value src)
	(if (scalar_first_stage_output_source? stages src)
		(or
			(probe_limit_small_enough? limit_value)
			(stage_probe_allowed_in_context?
				(stage_by_id stages (stage_output_relation_id (source_relation src)))
				(filter sources (lambda (candidate) (not (equal? (source_alias candidate) (source_alias src)))))))
		(if (not (presence_stage_output_source? stages src))
			false
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(define probe_sources (filter sources (lambda (candidate) (not (equal? (source_alias candidate) (source_alias src))))))
				(and
					(stage_probe_allowed_in_context? stage probe_sources)
					(or
						(stage_has_residual_outer_refs? stage)
						(or
							(not (stage_keys_are_input_local? stage))
							(stage_lookup_keys_resolve_in_sources?
								stage
								probe_sources
								default_alias)))))))))

(define scalar_aggregate_probe_output_source_for_block? (lambda (stages sources default_alias limit_value src)
	(if (not (scalar_aggregate_probe_stage_output_source? stages src))
		false
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(define probe_sources (filter sources (lambda (candidate) (not (equal? (source_alias candidate) (source_alias src))))))
			(and
				(scalar_aggregate_probe_stage_safe? stage)
				(and
					(if (empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
						(constant_scalar_aggregate_probe_sources? stages sources)
						(or
							(stage_has_residual_outer_refs? stage)
							(or
								(probe_limit_small_enough? limit_value)
								(probe_context_small_enough? probe_sources))))
					(stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias)))))))

(define scalar_cardinality_probe_output_source_for_block? (lambda (stages sources default_alias limit_value driver_condition src)
	(if (not (scalar_cardinality_probe_stage_output_source? stages src))
		false
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(define probe_sources (filter sources (lambda (candidate)
				(not (equal? (source_alias candidate) (source_alias src))))))
			(and (stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias)
				(or (probe_limit_small_enough? limit_value)
					(or (probe_context_small_enough? probe_sources)
						(probe_context_unique_point? probe_sources default_alias driver_condition))))))))

(define direct_group_probe_output_source_for_block? (lambda (stages sources default_alias limit_value driver_condition consumers src)
	(begin
		(define stage (direct_group_probe_stage_for_block_source stages sources src consumers))
		(if (nil? stage)
			false
			(begin
				(define probe_sources (filter sources (lambda (candidate)
					(not (equal? (source_alias candidate) (source_alias src))))))
				(and
					(stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias)
					(or (probe_limit_small_enough? limit_value)
						(or (probe_context_small_enough? probe_sources)
							(probe_context_unique_point? probe_sources default_alias driver_condition)))))))))

(define probe_output_sources_for_block (lambda (stages sources default_alias limit_value driver_condition consumers)
	(filter (coalesceNil sources '()) (lambda (src)
		(or
			(probeable_stage_output_source_for_block? stages sources default_alias limit_value src)
			(or
				(scalar_aggregate_probe_output_source_for_block? stages sources default_alias limit_value src)
				(or
					(scalar_cardinality_probe_output_source_for_block?
						stages sources default_alias limit_value driver_condition src)
					(direct_group_probe_output_source_for_block?
						stages sources default_alias limit_value driver_condition consumers src))))))))

(define sources_without_probe_outputs (lambda (sources probe_sources)
	(begin
		(define probe_aliases (reduce (coalesceNil probe_sources '()) (lambda (aliases src)
			(set_assoc aliases (source_alias src) true)) '()))
		(filter (coalesceNil sources '()) (lambda (src)
			(not (has_assoc? probe_aliases (source_alias src))))))))

(define presence_probe_output_sources (lambda (stages sources default_alias)
	(filter (coalesceNil sources '()) (lambda (src)
		(if (not (presence_stage_output_source? stages src))
			false
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(define probe_sources (filter sources (lambda (candidate) (not (equal? (source_alias candidate) (source_alias src))))))
				(and
					(stage_probe_allowed_in_context? stage probe_sources)
					(or
						(stage_has_residual_outer_refs? stage)
						(or
							(literal_true? (coalesceNil (source_join_expr src) true))
							(or
								(not (stage_keys_are_input_local? stage))
								(stage_lookup_keys_resolve_in_sources?
									stage
									probe_sources
									default_alias)))))))))))

(define stage_output_source_ids (lambda (sources)
	(filter (map (coalesceNil sources '()) (lambda (src)
		(if (stage_output_relation? (source_relation src))
			(stage_output_relation_id (source_relation src))
			nil)))
		(lambda (stage_id) (not (nil? stage_id))))))

(define sources_without_presence_probe_outputs (lambda (stages sources default_alias)
	(filter (coalesceNil sources '()) (lambda (src)
		(not (reduce (presence_probe_output_sources stages sources default_alias) (lambda (found probe_src)
			(or found (equal? (source_alias src) (source_alias probe_src))))
			false))))))

(define stage_for_group_cache_source (lambda (stages src)
	(if (lowering_catalog? stages)
		(begin
			(define index (lowering_catalog_group_cache_index stages))
			(define key (stage_dependency_group_cache_key (source_schema src) (source_relation src)))
			(define local (get_assoc index key))
			(if (not (nil? local))
				local
				(begin
					(define parent (lowering_catalog_parent stages))
					(if (lowering_catalog? parent) (stage_for_group_cache_source parent src) nil))))
		(reduce (coalesceNil stages '()) (lambda (found stage)
			(if (not (nil? found))
				found
				(if (and (group_stage? stage)
					(and (equal? (group_stage_cache_schema stage) (source_schema src))
						(equal? (group_stage_cache_relation stage) (source_relation src))))
					stage
					nil)))
			nil))))

(define group_cache_stages_from_sources (lambda (stages sources)
	(filter (map (coalesceNil sources '()) (lambda (src)
		(stage_for_group_cache_source stages src)))
		(lambda (stage) (not (nil? stage))))))

(define stage_dependency_id_index (lambda (stages)
	(if (lowering_catalog? stages)
		(if (lowering_catalog? (lowering_catalog_parent stages))
			(stage_dependency_id_index (lowering_catalog_stages stages))
			(lowering_catalog_id_index stages))
		(reduce (coalesceNil stages '()) (lambda (index stage)
			(if (group_stage? stage)
				(set_assoc index (gs_id stage) stage)
				index)) '()))))

(define stage_dependency_group_cache_key (lambda (schema relation)
	(list schema relation)))

(define stage_dependency_group_cache_index (lambda (stages)
	(if (lowering_catalog? stages)
		(if (lowering_catalog? (lowering_catalog_parent stages))
			(stage_dependency_group_cache_index (lowering_catalog_stages stages))
			(lowering_catalog_group_cache_index stages))
		(reduce (coalesceNil stages '()) (lambda (index stage)
			(if (not (group_stage? stage))
				index
				(set_assoc index
					(stage_dependency_group_cache_key
						(group_stage_cache_schema stage)
						(group_stage_cache_relation stage))
					stage))) '()))))

(define stage_dependencies_from_output_sources (lambda (id_index sources)
	(unique_stages_by_id (filter (map (coalesceNil sources '()) (lambda (src)
		(begin
			(define relation (source_relation src))
			(if (stage_output_relation? relation)
				(get_assoc id_index (stage_output_relation_id relation))
				nil))))
		(lambda (stage) (not (nil? stage)))))))

(define group_stage_direct_dependencies_using_indexes (lambda (id_index group_cache_index stage)
	(begin
		(define input (gs_input stage))
		(if (query_block? input)
			(unique_stages_by_id (merge (list
				(qb_stages input)
				(stage_dependencies_from_output_sources id_index (qb_sources input)))))
			(if (source_is_base_table? input)
				(begin
					(define group_cache_key (stage_dependency_group_cache_key
						(source_schema input)
						(source_relation input)))
					(define group_cache_stage (get_assoc group_cache_index group_cache_key))
					(if (not (nil? group_cache_stage))
						(list group_cache_stage)
						'()))
				'())))))

(define stage_dependency_graph (lambda (stages)
	(begin
		(define available_stages (unique_stages_by_id (lowering_catalog_stages stages)))
		(define id_index (stage_dependency_id_index stages))
		(define group_cache_index (stage_dependency_group_cache_index stages))
		(reduce available_stages (lambda (graph stage)
			(set_assoc graph (logical_stage_key stage)
				(if (group_stage? stage)
					(group_stage_direct_dependencies_using_indexes id_index group_cache_index stage)
					'()))) '()))))

(define stage_dependency_closure_index_visit (lambda (graph stage states closures)
	(begin
		(define key (logical_stage_key stage))
		(define state (if (has_assoc? states key) (states key) nil))
		(if (equal? state (quote visiting))
			(neumann_fail "build_queryplan" (concat "cyclic physical stage dependency " key))
			true)
		(if (has_assoc? closures key)
			(list (closures key) states closures)
			(begin
				(define nested (reduce (if (has_assoc? graph key) (graph key) '()) (lambda (acc dependency)
					(begin
						(define result (stage_dependency_closure_index_visit
							graph dependency (nth acc 1) (nth acc 2)))
						(list
							(merge (list (nth acc 0) (nth result 0)))
							(nth result 1)
							(nth result 2))))
					(list '() (set_assoc states key (quote visiting)) closures)))
				(define closure (unique_stages_by_id (cons stage (nth nested 0))))
				(list
					closure
					(set_assoc (nth nested 1) key (quote done))
					(set_assoc (nth nested 2) key closure)))))))

(define stage_dependency_closure_index_using_graph (lambda (graph stages)
	(nth (reduce (coalesceNil stages '()) (lambda (acc stage)
		(begin
			(define result (stage_dependency_closure_index_visit graph stage (nth acc 0) (nth acc 1)))
			(list (nth result 1) (nth result 2))))
		(list '() '())) 1)))

(define stage_dependency_closure_using_graph (lambda (graph stage)
	(nth (stage_dependency_closure_index_visit graph stage '() '()) 0)))

(define stage_dependency_closure_many_using_graph (lambda (graph stages)
	(begin
		(define roots (coalesceNil stages '()))
		(define closures (stage_dependency_closure_index_using_graph graph roots))
		(unique_stages_by_id (merge (map roots (lambda (stage)
			(get_assoc closures (logical_stage_key stage)))))))))

(define expr_probe_stages (lambda (expr)
	(match expr
		((symbol scalar_first_probe) stage _requested_col)
		(list stage)
		((quote scalar_first_probe) stage _requested_col)
		(list stage)
		((symbol scalar_first_probe) stage _requested_col stages)
		(merge_unique (list (list stage) stages))
		((quote scalar_first_probe) stage _requested_col stages)
		(merge_unique (list (list stage) stages))
		((symbol scalar_aggregate_probe) stage _requested_col)
		(list stage)
		((quote scalar_aggregate_probe) stage _requested_col)
		(list stage)
		((symbol scalar_cardinality_probe) stage _requested_col)
		(list stage)
		((quote scalar_cardinality_probe) stage _requested_col)
		(list stage)
		((symbol grouped_scalar_top) stage)
		(list stage)
		((quote grouped_scalar_top) stage)
		(list stage)
		(cons _head tail)
		(merge_unique (map tail expr_probe_stages))
		_ '())))

(define query_block_probe_expr_stages (lambda (block)
	(merge_unique (list
		(expr_probe_stages (qb_sources block))
		(expr_probe_stages (qb_fields block))
		(expr_probe_stages (qb_where block))
		(expr_probe_stages (qb_having block))
		(expr_probe_stages (qb_order block))
		(expr_probe_stages (qb_hidden block))))))

(define query_block_stage_catalog (lambda (block)
	(qassoc_get (qb_facts block) (quote stage_catalog) (qb_stages block))))

(define query_block_lowering_catalog (lambda (block)
	(qassoc_get (qb_facts block) (quote lowering_catalog) nil)))

(define query_block_stage_lookup (lambda (block)
	(coalesceNil (query_block_lowering_catalog block) (query_block_stage_catalog block))))

(define query_block_facts_with_stage_catalog (lambda (block stages)
	(qassoc_set (qb_facts block) (quote stage_catalog) stages)))

(define query_block_facts_with_lowering_catalog (lambda (block stages catalog)
	(if (lowering_catalog? catalog)
		(qassoc_set
			(query_block_facts_with_stage_catalog block stages)
			(quote lowering_catalog)
			catalog)
		(query_block_facts_with_stage_catalog block stages))))

(define query_block_with_stage_catalog (lambda (block stages)
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
		(query_block_facts_with_stage_catalog block stages))))

(define group_stage_with_lowering_catalog (lambda (stage catalog)
	(if (not (group_stage? stage))
		stage
		(begin
			(define facts (gs_facts stage))
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
				(if (lowering_catalog? catalog)
					(qassoc_set_without facts (quote lowering_catalog) catalog (quote stage_catalog))
					(qassoc_set facts (quote stage_catalog) catalog)))))))

(define group_stage_lowering_catalog (lambda (stage)
	(match (gs_facts stage)
		(cons entry _rest) (match entry
			((symbol lowering_catalog) catalog) catalog
			((quote lowering_catalog) catalog) catalog
			_ nil)
		_ nil)))

(define query_block_with_full_stage_catalog (lambda (block)
	(begin
		(define stages (stage_catalog_with_nested (query_block_stage_catalog block)))
		(define catalog (make_lowering_catalog stages))
		(define cataloged_stages (map (qb_stages block) (lambda (stage)
			(group_stage_with_lowering_catalog stage catalog))))
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
			cataloged_stages
			(query_block_facts_with_lowering_catalog block stages catalog)))))

(define stage_ids (lambda (stages)
	(map (coalesceNil stages '()) gs_id)))

(define stages_without_ids (lambda (stages ids)
	(filter (coalesceNil stages '()) (lambda (stage)
		(not (contains? ids (gs_id stage)))))))

(define scalar_probe_consumed_stages_many_using_graph (lambda (graph stages)
	(unique_stages_by_id (merge (list
		(filter (coalesceNil stages '()) (lambda (stage)
			(or (scalar_aggregate_probe_stage? stage) (scalar_cardinality_probe_stage? stage))))
		(stage_dependency_closure_many_using_graph graph
			(filter (coalesceNil stages '()) scalar_or_presence_probe_stage?)))))))

(define stage_id_set (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (ids stage)
		(set_assoc ids (gs_id stage) true)) '())))

(define stages_without_consumed_probes_using_graph (lambda (graph stages probe_stages)
	(begin
		(define consumed_ids (stage_id_set
			(scalar_probe_consumed_stages_many_using_graph graph probe_stages)))
		(filter (coalesceNil stages '()) (lambda (stage)
			(not (has_assoc? consumed_ids (gs_id stage))))))))

(define stages_without_scalar_first_probes (lambda (stages)
	(stages_without_consumed_probes_using_graph
		(stage_dependency_graph stages) stages stages)))

(define stages_without_probe_sources_using_graph (lambda (dependency_graph stage_lookup stages probe_sources)
	(begin
		(define consumed (stages_without_consumed_probes_using_graph dependency_graph stages
			(stage_outputs_from_sources_using stage_lookup probe_sources)))
		(define direct_ids (stage_output_source_ids (filter probe_sources (lambda (src)
			(not (nil? (direct_group_probe_stage_for_source stage_lookup src (list true))))))))
		(stages_without_ids consumed direct_ids))))

(define stages_without_probe_sources (lambda (stages probe_sources)
	(stages_without_probe_sources_using_graph
		(stage_dependency_graph stages) stages stages probe_sources)))

(define query_block_probe_consumers (lambda (block)
	(list
		(qb_fields block)
		(qb_where block)
		(qb_group block)
		(qb_having block)
		(qb_order block)
		(qb_hidden block))))

(define query_block_with_scalar_first_probes_using_graph (lambda (stages dependency_graph block)
	(begin
		(define stage_list (lowering_catalog_stages stages))
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define consumers (query_block_probe_consumers block))
		(define prelimit_sources (prelimit_sources_for sources default_alias
			(coalesceNil (qb_where block) true) (coalesceNil (qb_order block) '())))
		(define prelimit_aliases (map prelimit_sources source_alias))
		(define unbounded_probe_candidates (probe_output_sources_for_block
			stages sources default_alias nil (qb_where block) consumers))
		(define unbounded_probe_aliases (map unbounded_probe_candidates source_alias))
		(define probe_candidates (filter
			(probe_output_sources_for_block stages sources default_alias
				(qb_limit block) (qb_where block) consumers)
			(lambda (src)
				(or (not (contains? prelimit_aliases (source_alias src)))
					(contains? unbounded_probe_aliases (source_alias src))))))
		(define order_lookup (if (late_projection_candidate_block? block)
			(scalar_order_lookup_source stages sources default_alias (coalesceNil (qb_order block) '()))
			nil))
		(define retained_order_alias (if (or (nil? order_lookup) (nil? (nth order_lookup 0)))
			nil
			(source_alias (nth order_lookup 0))))
		(define probe_sources (if (nil? retained_order_alias)
			probe_candidates
			(merge_unique (list probe_candidates (list (nth order_lookup 0))))))
		(define probe_index
			(probe_stage_alias_index_using_graph stages dependency_graph probe_sources consumers))
		(define rewritten_sources (rewrite_scalar_first_probe_sources_using_index stages sources probe_index default_alias))
		(make_query_block
			(qb_schema block)
			(sources_without_probe_outputs rewritten_sources probe_sources)
			(rewrite_scalar_first_probe_fields_using_index stages probe_index default_alias (qb_fields block))
			(rewrite_scalar_first_probe_expr_using_index stages probe_index default_alias (qb_where block))
			(qb_group block)
			(rewrite_scalar_first_probe_expr_using_index stages probe_index default_alias (qb_having block))
			(rewrite_scalar_first_probe_order_using_index stages probe_index default_alias (qb_order block))
			(qb_limit block)
			(qb_offset block)
			(rewrite_scalar_first_probe_fields_using_index stages probe_index default_alias (qb_hidden block))
			(stages_without_probe_sources_using_graph
				dependency_graph stages (qb_stages block) probe_sources)
			(join_optimizer_facts_without_aliases
				(query_block_facts_with_stage_catalog block stage_list)
				(map probe_sources source_alias))))))

(define query_block_with_scalar_first_probes_using (lambda (stages block)
	(query_block_with_scalar_first_probes_using_graph
		stages (stage_dependency_graph stages) block)))

(define query_block_with_presence_probe_sources_using (lambda (stages probe_sources block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define rewritten_sources (rewrite_scalar_first_probe_sources_using stages sources probe_sources default_alias))
		(make_query_block
			(qb_schema block)
			(sources_without_presence_probe_outputs stages rewritten_sources default_alias)
			(rewrite_scalar_first_probe_fields stages probe_sources default_alias (qb_fields block))
			(rewrite_scalar_first_probe_expr stages probe_sources default_alias (qb_where block))
			(qb_group block)
			(rewrite_scalar_first_probe_expr stages probe_sources default_alias (qb_having block))
			(rewrite_scalar_first_probe_order stages probe_sources default_alias (qb_order block))
			(qb_limit block)
			(qb_offset block)
			(rewrite_scalar_first_probe_fields stages probe_sources default_alias (qb_hidden block))
			(qb_stages block)
			(join_optimizer_facts_without_aliases
				(qassoc_set
					(qb_facts block)
					(quote consumed_presence_probe_stage_ids)
					(merge_unique (list
						(qassoc_get (qb_facts block) (quote consumed_presence_probe_stage_ids) '())
						(stage_output_source_ids probe_sources))))
				(map probe_sources source_alias))))))

(define query_block_with_presence_probes_using (lambda (stages block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(query_block_with_presence_probe_sources_using
			stages
			(presence_probe_output_sources stages sources default_alias)
			block))))

(define query_block_without_stages_after_prepare_using (lambda (stages block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(define rewritten (query_block_with_scalar_first_probes_using stage_lookup block))
		(make_query_block
			(qb_schema rewritten)
			(physicalize_stage_output_sources stage_lookup (qb_sources rewritten))
			(qb_fields rewritten)
			(qb_where rewritten)
			(qb_group rewritten)
			(qb_having rewritten)
			(qb_order rewritten)
			(qb_limit rewritten)
			(qb_offset rewritten)
			(qb_hidden rewritten)
			'()
			(query_block_facts_with_stage_catalog rewritten available_stages)))))

(define query_block_without_stages_after_prepare (lambda (block)
	(query_block_without_stages_after_prepare_using (qb_stages block) block)))

(define query_block_without_stages_after_eager_prepare_using (lambda (stages block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(make_query_block
			(qb_schema block)
			(physicalize_stage_output_sources stage_lookup (qb_sources block))
			(qb_fields block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(qb_hidden block)
			'()
			(query_block_facts_with_stage_catalog block '())))))

(define query_block_without_stages_after_eager_prepare_with_constant_scalars_first (lambda (stages block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(make_query_block
			(qb_schema block)
			(physicalize_stage_output_sources stage_lookup (qb_sources block))
			(qb_fields block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_limit block)
			(qb_offset block)
			(qb_hidden block)
			'()
			(qb_facts block)))))

(define query_block_with_prepared_sources_using_graph (lambda (stages dependency_graph block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(define scalar_rewritten
			(query_block_with_scalar_first_probes_using_graph stage_lookup dependency_graph block))
		(define membership_rewritten (query_block_with_physical_membership_using stage_lookup scalar_rewritten))
		(define sources (qb_sources membership_rewritten))
		(define default_alias (qassoc_get (qb_facts membership_rewritten) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define presence_probe_sources (presence_probe_output_sources stage_lookup sources default_alias))
		(define rewritten (query_block_with_presence_probes_using stage_lookup membership_rewritten))
		(make_query_block
			(qb_schema rewritten)
			(physicalize_stage_output_sources stage_lookup (qb_sources rewritten))
			(qb_fields rewritten)
			(qb_where rewritten)
			(qb_group rewritten)
			(qb_having rewritten)
			(qb_order rewritten)
			(qb_limit rewritten)
			(qb_offset rewritten)
			(qb_hidden rewritten)
			(stages_without_probe_sources_using_graph
				dependency_graph stage_lookup (qb_stages rewritten) presence_probe_sources)
			(query_block_facts_with_stage_catalog rewritten available_stages)))))

(define query_block_with_prepared_sources_using (lambda (stages block)
	(query_block_with_prepared_sources_using_graph
		stages (stage_dependency_graph stages) block)))

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
		(define cache (group_stage_cache stage))
		(define schema (group_cache_schema cache))
		(define grouptbl (group_cache_relation cache))
		(define original_output (gs_output stage))
		(define resolved_output (map_assoc original_output (lambda (_title expr)
			(canonical_column_expr_for_alias alias expr))))
		(define original_having (coalesceNil (gs_having stage) true))
		(define resolved_having (canonical_column_expr_for_alias alias original_having))
		(define order_items (coalesceNil (gs_order stage) '()))
		(define resolved_order_exprs (map order_items (lambda (item)
			(match item '(expr _dir) (canonical_column_expr_for_alias alias expr)))))
		(define extras (coalesceNil extra_sources '()))
		(define resolved_extra_joins (map extras (lambda (extra)
			(canonical_column_expr_for_alias alias (source_join_expr extra)))))
		(define key_index (make_group_key_index keys (merge (list
			(extract_assoc resolved_output (lambda (_title expr) expr))
			(list resolved_having)
			resolved_order_exprs
			resolved_extra_joins))))
		(define output_fields (replace_group_fields_indexed
			alias grouptbl keys key_names ags key_index original_output resolved_output))
		(define replaced_having (replace_group_expr_indexed alias grouptbl keys key_names ags key_index
			original_having resolved_having))
		(define count_col_name (aggregate_col_name aggregate_count_descriptor))
		(define count_check (list (quote >) (list (quote get_column) grouptbl false count_col_name false) 0))
		(define needs_count_filter (and (not (equal? keys '(1))) (not (equal? condition true))))
		(define aggregate_having_expr (if (not needs_count_filter)
			replaced_having
			(if (or (nil? replaced_having) (equal? replaced_having true))
				count_check
				(list (quote and) replaced_having count_check))))
		(define session_filter (group_stage_session_filter_expr stage grouptbl keys key_names))
		(define having_expr (combine_where_terms (list aggregate_having_expr session_filter) true))
		(make_query_block
			schema
			(cons
				(list grouptbl schema grouptbl false nil)
				(map (produceN (count extras)) (lambda (i)
					(rewrite_source_for_group_domain_indexed alias grouptbl keys key_names ags key_index
						(nth extras i) (nth resolved_extra_joins i)))))
			output_fields
			having_expr
			nil nil
			(map (produceN (count order_items)) (lambda (i)
				(match (nth order_items i) '(expr dir) (begin
					(define replaced_order_expr (replace_group_order_expr_indexed
						alias grouptbl keys key_names ags key_index expr
						(nth resolved_order_exprs i)))
					(list (group_order_physical_expr grouptbl replaced_order_expr) dir)))))
			(gs_limit stage)
			(gs_offset stage)
			'() '() '()))))

(define group_stage_final_extra_source_refs (lambda (stage)
	(begin
		(define src (gs_input stage))
		(if (query_block? src)
			(filter (cdr (qb_sources src)) (lambda (extra)
				(source_needed_after_group_stage? (group_stage_input_alias stage) stage extra)))
			'()))))

(define group_stage_final_extra_sources_using (lambda (stages stage)
	(physicalize_stage_output_sources stages (group_stage_final_extra_source_refs stage))))

(define group_stage_final_extra_sources (lambda (stage)
	(group_stage_final_extra_sources_using (if (query_block? (gs_input stage)) (qb_stages (gs_input stage)) '()) stage)))

(define stage_prepare_identity (lambda (stage)
	(if (group_stage? stage)
		(begin
			(define cache (group_stage_cache stage))
			(list
				(quote group)
				(group_cache_schema cache)
				(group_cache_relation cache)
				(map (gs_aggregates stage) aggregate_col_name)
				(map (gs_order stage) (lambda (item) (fnv_hash (serialize item))))
				(if (source_is_base_table? (gs_input stage))
					nil
					(list
						(qassoc_get (gs_facts stage) (quote purpose) nil)
						(qassoc_get (gs_facts stage) (quote cardinality_mode) nil)))))
		(logical_stage_key stage))))

(define lower_unique_stage_prepares_acc (lambda (stages seen initialized_group_caches prepare)
	(match (coalesceNil stages '())
		(cons stage rest) (begin
			(define shared_group_cache (and (group_stage? stage) (source_is_base_table? (gs_input stage))))
			(define group_cache_key (if shared_group_cache (group_stage_cache_owner_key stage) nil))
			(define initializer_owner (or (not shared_group_cache) (not (has_assoc? initialized_group_caches group_cache_key))))
			(define prepared_stage (if shared_group_cache
				(group_stage_with_initializer_owner stage initializer_owner)
				stage))
			(define plan (prepare prepared_stage))
			(define identity (stage_prepare_identity stage))
			(define next_group_caches (if (and shared_group_cache initializer_owner)
				(set_assoc initialized_group_caches group_cache_key true)
				initialized_group_caches))
			(if (has_assoc? seen identity)
				(lower_unique_stage_prepares_acc rest seen next_group_caches prepare)
				(cons plan
					(lower_unique_stage_prepares_acc rest (set_assoc seen identity true) next_group_caches prepare))))
		_ '())))

(define lower_unique_stage_prepares (lambda (stages prepare)
	(lower_unique_stage_prepares_acc stages '() '() prepare)))

(define lower_unique_stage_prepares_using (lambda (all_stages lookup_stages stages)
	(lower_unique_stage_prepares stages (lambda (stage)
		(lower_stage_prepare_using all_stages lookup_stages stage)))))

(define lower_unique_stage_prepares_with_graph (lambda (dependency_graph stage_lookup stages)
	(lower_unique_stage_prepares stages (lambda (stage)
		(lower_stage_prepare_using
			(stage_dependency_closure_using_graph dependency_graph stage)
			stage_lookup
			stage)))))

(define lower_group_stage_prepare_using (lambda (all_stages lookup_stages stage)
	(begin
		(define src (gs_input stage))
		(define prepare_catalog (unique_stages_by_id (merge (list (list stage) all_stages))))
		(define fact_lookup (group_stage_lowering_catalog stage))
		(define stage_lookup (if (lowering_catalog? lookup_stages)
			lookup_stages
			(if (lowering_catalog? fact_lookup)
				fact_lookup
				(unique_stages_by_id
					(merge (list
						prepare_catalog
						lookup_stages
						(qassoc_get (gs_facts stage) (quote stage_catalog) '())))))))
		(if (and (not (union_block? src)) (and (not (query_block? src)) (not (source_is_base_table? src))))
			(neumann_fail "build_queryplan" "group-stage lowering expects a base table, query-block, or union-block input")
			true)
		(define cache (group_stage_cache stage))
		(if (not (equal? (group_cache_kind cache) (quote group-keytable)))
			(neumann_fail "build_queryplan" "foreign-key-backed group caches are not lowered yet")
			true)
		(define schema (group_cache_schema cache))
		(define tbl (group_stage_input_name stage))
		(define alias (group_stage_input_alias stage))
		(define query_input (or (query_block? src) (union_block? src)))
		(define rewritten_src (if (query_block? src)
			(query_block_with_presence_probes_using stage_lookup src)
			src))
		(define rewrite_sources (if (query_block? src) (qb_sources src) '()))
		(define rewrite_default_alias (if (query_block? src)
			(qassoc_get (qb_facts src) (quote default_alias) (if (empty_list? rewrite_sources) nil (source_alias (car rewrite_sources))))
			nil))
		(define presence_probe_sources_for_rewrite (if (query_block? src)
			(presence_probe_output_sources stage_lookup rewrite_sources rewrite_default_alias)
			'()))
		(define keys (if (empty_list? (gs_keys stage))
			'(1)
			(if (query_block? src)
				(map (gs_keys stage) (lambda (key)
					(rewrite_scalar_first_probe_expr stage_lookup presence_probe_sources_for_rewrite rewrite_default_alias key)))
				(gs_keys stage))))
		(define prepare_order_items (coalesceNil (gs_order stage) '()))
		(define prepare_resolved_order_exprs (map prepare_order_items (lambda (item)
			(match item '(expr _dir) (canonical_column_expr_for_alias alias expr)))))
		(define key_index (make_group_key_index keys prepare_resolved_order_exprs))
		(define ags (gs_aggregates stage))
		(define lowering_ags (if (query_block? src)
			(rewrite_scalar_first_probe_aggregates stage_lookup presence_probe_sources_for_rewrite rewrite_default_alias ags)
			ags))
		(define condition (if (query_block? src)
			(rewrite_scalar_first_probe_expr stage_lookup presence_probe_sources_for_rewrite rewrite_default_alias (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
			(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)))
		(define key_names (group_key_cols keys))
		(define aggregate_condition (replace_group_session_expr stage keys key_names condition))
		(define grouptbl (group_cache_relation cache))
		(define initializer_owner (qassoc_get (gs_facts stage) (quote keytable_initializer_owner) true))
		(define scalar_single_stage (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single)))
		(define scalar_query_stage (and (query_block? src)
			(and scalar_single_stage
				(and (equal? (qassoc_get (gs_facts stage) (quote cardinality_mode) nil) (quote single_or_error))
					(and (equal? (count ags) 2)
						(equal? (cadr ags) aggregate_count_descriptor))))))
		(define scalar_order_base_stage (and (not query_input)
			(compatible_scalar_order_aggregates? ags)))
		(define scalar_aggregate_stage (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_aggregate)))
		(define prepared_src (if (query_block? rewritten_src)
			(if scalar_aggregate_stage
				(begin
					(define constant_reorder_stages (if (lowering_catalog? stage_lookup)
						stage_lookup
						(unique_stages_by_id (merge (list stage_lookup (qb_stages rewritten_src))))))
					(if (not (empty_list? (filter (qb_sources rewritten_src) (lambda (src)
						(constant_scalar_or_presence_stage_output_source? constant_reorder_stages src)))))
						(query_block_without_stages_after_eager_prepare_with_constant_scalars_first constant_reorder_stages rewritten_src)
						(query_block_without_stages_after_eager_prepare_using stage_lookup rewritten_src)))
				(query_block_without_stages_after_eager_prepare_using stage_lookup rewritten_src))
			rewritten_src))
		(define direct_nested_stages (if (query_block? rewritten_src)
			(merge_unique (list
				(query_block_stages_to_prepare_using prepare_catalog rewritten_src)
				(available_stage_outputs_from_sources_using prepare_catalog (qb_sources rewritten_src))
				(available_stage_outputs_from_sources_using prepare_catalog (group_stage_final_extra_source_refs stage))
				(group_cache_stages_from_sources prepare_catalog (qb_sources rewritten_src))
				(group_cache_stages_from_sources prepare_catalog (group_stage_final_extra_source_refs stage))
				(query_block_probe_expr_stages rewritten_src)))
			'()))
		(define owner_handle (qassoc_get (gs_facts stage) (quote btw2025_handle) nil))
		(define owner_ancestors (qassoc_get (gs_facts stage) (quote btw2025_ancestors) '()))
		(define nested_stages (if (nil? owner_handle)
			direct_nested_stages
			(filter direct_nested_stages (lambda (candidate)
				(begin
					(define candidate_handle (qassoc_get (gs_facts candidate) (quote btw2025_handle) nil))
					(define candidate_parent (qassoc_get (gs_facts candidate) (quote btw2025_parent) nil))
					(or
						(nil? candidate_handle)
						(equal? candidate_parent owner_handle)
						(and
							(nil? candidate_parent)
							(and
								(not (equal? candidate_handle owner_handle))
								(not (contains? owner_ancestors candidate_handle))))))))))
		(define nested_prepare (if (query_block? rewritten_src)
			(lower_unique_stage_prepares_using prepare_catalog stage_lookup nested_stages)
			'()))
		(define nested_materialize (if (query_block? rewritten_src) (lower_stage_materialize_all nested_stages) '()))
		(define nested_prepare_expr (if (empty_list? nested_prepare)
			nil
			(cons (quote !begin) (merge (list nested_prepare nested_materialize)))))
		(define key_columns (map key_names (lambda (col) (list (quote list) "column" col "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))
		(define create_cols (cons (quote list)
			(cons (cons (quote list) (cons "unique" (cons "group" (list (cons (quote list) key_names)))))
				key_columns)))
		(define ensure_agg_columns (if (or query_input scalar_order_base_stage)
			(map ags (lambda (ag)
				(list (quote createcolumn)
					(list (quote table) schema grouptbl)
					(aggregate_col_name ag)
					"any"
					(quoted_runtime_list '())
					(quoted_runtime_list '()))))
			'()))
		(define collect_plan (if (not query_input)
			nil
			(if (union_block? src)
				(build_union_group_aggregates_insert_plan prepared_src grouptbl keys key_names (list aggregate_count_descriptor))
				(build_query_group_collect_plan prepared_src grouptbl keys key_names))))
		(define base_group_into_plan (if (or query_input scalar_order_base_stage)
			nil
			(build_base_group_into_plan schema tbl alias src grouptbl keys key_names condition
				(non_scalar_order_aggregates ags))))
		(define cleanup_plan (if (query_block? src)
			nil
			(build_group_keytable_cleanup schema tbl alias grouptbl keys key_names)))
		(define agg_plans (if query_input
			(if (empty_list? ags)
				'()
				(list (if (union_block? src)
					(build_union_group_aggregates_insert_plan prepared_src grouptbl keys key_names ags)
					(build_query_group_aggregates_insert_plan_using prepared_src grouptbl keys key_names lowering_ags ags))))
			(if scalar_order_base_stage
				(list (build_group_ordered_scalar_columns_insert_plan schema tbl alias grouptbl keys key_names condition ags))
				(map ags (lambda (ag) (build_group_aggregate_column schema tbl alias grouptbl keys key_names aggregate_condition ag))))))
		(define empty_aggregate_seed_plans (if (and query_input
			(and initializer_owner
				(and (not scalar_single_stage)
					(not (empty_list? ags)))))
			(if (equal? keys '(1))
				(list (build_group_constant_key_insert_plan schema grouptbl))
				(if (reduce keys (lambda (session_only key)
					(and session_only (query_session_read? key))) true)
					(list (build_group_session_key_insert_plan schema grouptbl key_names keys))
					'()))
			'()))
		(define computed_order_exprs (merge_unique (map (produceN (count prepare_order_items)) (lambda (i)
			(match (nth prepare_order_items i) '(expr _dir) (begin
				(define replaced_order_expr (replace_group_order_expr_indexed
					alias grouptbl keys key_names ags key_index expr
					(nth prepare_resolved_order_exprs i)))
				(if (direct_group_order_expr? replaced_order_expr) '() (list replaced_order_expr))))))))
		(define computed_order_plans (map computed_order_exprs (lambda (expr)
			(build_group_computed_order_column schema grouptbl expr))))
		(define ensure_agg_expr (if (empty_list? ensure_agg_columns)
			nil
			(cons (quote !begin) ensure_agg_columns)))
		(define aggregate_prepare_expr (cons
			(quote !begin)
			(merge (list ensure_agg_columns agg_plans computed_order_plans))))
		(define base_group_fill (symbol "__group_base_fill"))
		(define base_group_fill_call (list base_group_fill))
		(define finalize_group_fill (list (quote rebuild) (list (quote table) schema grouptbl) true false))
		(define initial_fill_expr (if (nil? base_group_into_plan)
			nil
			(list (quote initialize_cache_table)
				(list (quote table) schema grouptbl)
				(list (quote list) (source_table_expr src))
				(list (quote lambda) '()
					(cons (quote !begin) (filter (list aggregate_prepare_expr cleanup_plan) (lambda (expr) (not (nil? expr))))))
				base_group_fill
				(list (quote lambda) '() finalize_group_fill))))
		(define create_options (if (nil? initial_fill_expr)
			(quoted_runtime_list '("engine" "memory"))
			(list (quote list)
				"engine" "memory"
				"oninit" (list (quote lambda) '() initial_fill_expr))))
		(define group_cache_created (symbol "__group_cache_created"))
		(define keytable_init (list
			(list (quote lambda) (list group_cache_created)
				(list (quote !begin)
					(list (quote touch_keytable) (list (quote table) schema grouptbl))
					group_cache_created))
			(list (quote createtable) schema grouptbl create_cols create_options true)))
		(define lowered_plan (if scalar_query_stage
			(list (quote !begin)
				nested_prepare_expr
				(if initializer_owner keytable_init nil)
				ensure_agg_expr
				(build_scalar_single_query_stage_fill_plan prepared_src grouptbl keys key_names (car lowering_ags) (cadr lowering_ags)))
			(if query_input
				(cons (quote !begin)
					(merge (list
						nested_prepare
						nested_materialize
						(if initializer_owner (list keytable_init) '())
						ensure_agg_columns
						computed_order_plans
						(if (and initializer_owner (empty_list? ags)) (list collect_plan) '())
						agg_plans
						empty_aggregate_seed_plans)))
				(if scalar_order_base_stage
					(list (quote !begin)
						nested_prepare_expr
						(if initializer_owner keytable_init nil)
						aggregate_prepare_expr)
					(list (quote !begin)
						nested_prepare_expr
						(if initializer_owner
							(list
								(list (quote lambda) (list group_cache_created)
									(list (quote if) group_cache_created
										nil
										(list (quote if) (group_stage_session_binding_missing_expr stage schema grouptbl keys key_names)
											(list (quote !begin)
												aggregate_prepare_expr
												base_group_fill_call)
											aggregate_prepare_expr)))
								keytable_init)
							aggregate_prepare_expr))))))
		(if (nil? base_group_into_plan)
			lowered_plan
			(list
				(list (quote lambda) (list base_group_fill) lowered_plan)
				(list (quote lambda) '() base_group_into_plan))))))

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

(define lower_stage_prepare_using (lambda (all_stages lookup_stages stage)
	(if (group_stage? stage)
		(lower_group_stage_prepare_using all_stages lookup_stages stage)
		(if (orc_stage? stage)
			(lower_orc_stage_prepare stage)
			(if (window_stage? stage)
				(lower_window_stage_prepare stage)
				(neumann_fail "build_queryplan" "unknown logical stage"))))))
(define lower_stage_prepare (lambda (stage)
	(lower_stage_prepare_using (list stage) (list stage) stage)))

(define nested_stage_catalog (lambda (stage)
	(begin
		(define input (if (group_stage? stage) (gs_input stage) nil))
		(define catalog (if (query_block? input)
			(merge_unique (list
				(qassoc_get (gs_facts stage) (quote stage_catalog) '())
				(query_block_stage_catalog input)))
			'()))
		(define nested (if (query_block? input)
			(merge_unique (list
				(qb_stages input)
				(query_block_probe_expr_stages input)))
			'()))
		(unique_stages_by_id
			(merge (list
				(list stage)
				catalog
				(merge (map nested nested_stage_catalog))))))))

(define stage_catalog_with_nested (lambda (stages)
	(unique_stages_by_id
		(merge (map (coalesceNil stages '()) nested_stage_catalog)))))

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
		(define stage_catalog (stage_catalog_with_nested
			(merge (list
				(nested_stage_catalog stage)
				(if (query_block? src) (query_block_stage_catalog src) '())))))
		(list (quote begin)
			(lower_group_stage_prepare_using stage_catalog stage_catalog stage)
			(lower_query_block_core
				(group_stage_final_block stage (group_stage_final_extra_sources_using stage_catalog stage)))))))

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
								(if (expr_contains_driver_membership? condition)
									nil
									(if (not (row_number_order_compatible? src (coalesceNil (qb_order block) '()) sortcols sortdirs))
										nil
										(begin
											(define pre_condition (if (nil? row_number_filter) condition (nth row_number_filter 2)))
											(define membership (driver_membership_for_source src pre_condition))
											(define membership_table_expr (if (nil? membership) nil (recset_project_join_expr_for_membership src membership)))
											(define effective_membership (if (nil? membership_table_expr) nil membership))
											(define effective_pre_condition (strip_driver_membership_for_source src pre_condition effective_membership))
											(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
											(define membership_var (symbol "__membership_recset"))
											(define membership_filter_expr (if (nil? membership_table_expr)
												true
												(recset_contains_call_expr membership_var)))
											(define filter_condition (combine_where membership_filter_expr effective_pre_condition))
											(define rewritten_condition (replace_row_number_expr src col effective_condition))
											(define filtercols (merge_unique (list
												(if (nil? membership_table_expr) '() (list "$recset_contains"))
												(extract_columns_for_alias src effective_pre_condition))))
											(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
												(extract_columns_for_alias src expr)))))
											(define scan_mapcols (merge_unique (list (without_col fieldcols col) mapcols)))
											(define filter_expr (list (quote lambda)
												(map filtercols (lambda (filter_col) (scan_callback_symbol_for_alias (source_alias src) filter_col)))
												(list (quote optimize) (lower_column_expr_for_alias src filter_condition))))
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
											(define scan_expr (list (quote scan_order)
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
												(source_outer? src)))
											(if (nil? membership_table_expr)
												scan_expr
												(list
													(list (quote lambda) (list membership_var) scan_expr)
													membership_table_expr)))))))
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
				(define stage_lookup (query_block_stage_lookup block))
				(define rewritten (query_block_with_scalar_first_probes_using stage_lookup block))
				(define probe_rewritten (if (expr_contains_session_dependency? (qb_where rewritten))
					(query_block_with_prepared_sources_using stage_lookup rewritten)
					rewritten))
				(define sources (qb_sources probe_rewritten))
				(define stage_catalog (query_block_stage_catalog rewritten))
				(define dependency_graph (stage_dependency_graph stage_lookup))
				(define outer_prepare_stages (filter (qb_stages rewritten) (lambda (stage)
					(not (group_stage? stage)))))
				(define physical_sources (physicalize_stage_output_sources stage_lookup sources))
				(define driver_src (car physical_sources))
				(if (not (source_is_base_table? driver_src))
					(neumann_fail "build_queryplan" "group-stage source did not lower to a physical driver")
					true)
				(define stage_sources (cdr physical_sources))
				(define direct_probe_group (and
					(empty_list? stage_sources)
					(empty_list? (qb_stages probe_rewritten))
					(source_is_base_table? driver_src)
					(expr_contains_driver_membership? (qb_where probe_rewritten))))
				(define final_stage_sources (physicalize_stage_output_sources stage_lookup
					(filter (cdr sources) (lambda (src)
						(source_needed_after_group? (source_alias driver_src) probe_rewritten src)))))
				(define grouped_input_block (make_query_block
					(qb_schema probe_rewritten)
					(cons driver_src stage_sources)
					(qb_fields probe_rewritten)
					(qb_where probe_rewritten)
					(qb_group probe_rewritten)
					(qb_having probe_rewritten)
					(qb_order probe_rewritten)
					(qb_limit probe_rewritten)
					(qb_offset probe_rewritten)
					(qb_hidden probe_rewritten)
					'()
					(qb_facts probe_rewritten)))
				(define main_stage (make_group_stage_for_query_block grouped_input_block))
				(define main_stage_lookup (lowering_catalog_with_local_stages stage_lookup (list main_stage)))
				(if direct_probe_group
					(lower_query_block_core grouped_input_block)
					(cons (quote !begin)
						(merge (list
							/* The main group prepare owns nested group stages used by its input.
							Window and ORC stages still require their explicit materialization. */
							(lower_unique_stage_prepares_with_graph dependency_graph stage_lookup outer_prepare_stages)
							(lower_stage_materialize_all outer_prepare_stages)
							(list (lower_group_stage_prepare_using (cons main_stage stage_catalog) main_stage_lookup main_stage))
							(list (lower_query_block_core (group_stage_final_block main_stage final_stage_sources))))))))))))

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

(define expr_contains_driver_membership? (lambda (expr)
	(match expr
		((symbol driver_membership_probe) _stage _probe) true
		((quote driver_membership_probe) _stage _probe) true
		(cons head tail) (or
			(expr_contains_driver_membership? head)
			(reduce tail (lambda (found item)
				(or found (expr_contains_driver_membership? item)))
				false))
		_ false)))

(define stage_consumed_by_probe_source? (lambda (stage stages sources default_alias limit_value)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(or found
			(and (stage_output_relation? (source_relation src))
				(and (equal? (stage_output_relation_id (source_relation src)) (gs_id stage))
					(and
						(probeable_stage_output_source? stages src)
						(probeable_stage_output_source_for_block? stages sources default_alias limit_value src))))))
		false)))

(define scalar_first_inline_only_stage? (lambda (stage)
	(and
		(scalar_first_probe_stage? stage)
		(not (query_block? (gs_input stage))))))

(define stages_consumed_by_sources_with_closure_using_graph (lambda (dependency_graph stages sources)
	(merge_unique (map (stage_outputs_from_sources_using stages sources) (lambda (stage)
		(stage_dependency_closure_using_graph dependency_graph stage))))))

(define stages_consumed_by_sources_with_closure (lambda (stages sources)
	(stages_consumed_by_sources_with_closure_using_graph (stage_dependency_graph stages) stages sources)))

(define stage_id_in? (lambda (stage ids)
	(contains? (coalesceNil ids '()) (gs_id stage))))

(define stage_ids_for_sources_with_closure (lambda (stages sources)
	(map (stages_consumed_by_sources_with_closure stages sources) gs_id)))

(define stage_ids_for_sources_with_closure_using_graph (lambda (dependency_graph stages sources)
	(map (stages_consumed_by_sources_with_closure_using_graph dependency_graph stages sources) gs_id)))

(define prelimit_sources_for (lambda (sources default_alias where_expr order_items)
	(begin
		(define required_columns (join_column_recipe sources default_alias
			(merge (list (list where_expr) (order_exprs order_items)))))
		(if (empty_list? sources)
			'()
			(cons (car sources)
				(filter (cdr sources) (lambda (src)
					(not (empty_list? (qassoc_get required_columns (source_alias src) '()))))))))))

(define query_block_prelimit_sources (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define first_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define where_expr (coalesceNil (qb_where block) true))
		(prelimit_sources_for sources first_alias where_expr (coalesceNil (qb_order block) '())))))

(define late_projection_candidate_block? (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define prelimit_sources (query_block_prelimit_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(define order_items (coalesceNil (qb_order block) '()))
		(define membership_candidate (and
			(equal? (qassoc_get (qb_facts block) (quote membership_plan_strategy) nil) (quote candidate_keyset))
			(and (not (empty_list? sources))
				(order_items_belong_to_source? (car sources) order_items))))
		(define scalar_lookup_candidate (and
			(query_limit_active? (qb_offset block) (qb_limit block))
			(not (nil? (scalar_order_lookup_source
				(query_block_stage_lookup block) sources default_alias order_items)))))
		(and (not (single_source? sources))
			(and (not (empty_list? order_items))
				(and (or membership_candidate scalar_lookup_candidate)
					(late_projection_sources_preserve_rows? sources prelimit_sources)))))))

(define stage_direct_prepare_source_visible? (lambda (block sources default_alias stage)
	(if (single_source? sources)
		true
		(not (or
			(row_number_stage_consumed_by_join? stage sources)
			(stage_consumed_by_membership_source? stage (qb_stages block) sources (qb_facts block))
			(stage_consumed_by_probe_source? stage (qb_stages block) sources default_alias (qb_limit block)))))))

(define stage_direct_prepare_semantic_candidate? (lambda (consumed_probe_ids consumed_source_probe_ids stage_output_ids stage)
	(and
		(nil? (qassoc_get (gs_facts stage) (quote btw2025_parent) nil))
		(and
			(not (contains? consumed_probe_ids (gs_id stage)))
			(and (not (contains? consumed_source_probe_ids (gs_id stage)))
				(and
					(not (scalar_first_inline_only_stage? stage))
					(and
						(not (scalar_first_probe_stage? stage))
						(or
							(not (scalar_aggregate_probe_stage? stage))
							(contains? stage_output_ids (gs_id stage))))))))))

(define query_block_stages_to_prepare_base_using (lambda (all_stages block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (if (empty_list? sources) nil (source_alias (car sources)))))
		(define consumed_probe_ids (qassoc_get (qb_facts block) (quote consumed_presence_probe_stage_ids) '()))
		(define consumed_source_probe_ids (stage_output_source_ids (probe_output_sources_for_block
			all_stages sources default_alias (qb_limit block) (qb_where block) (query_block_probe_consumers block))))
		(define stage_output_ids (stage_output_source_ids sources))
		(define direct (filter (qb_stages block) (lambda (stage)
			(and
				(stage_direct_prepare_source_visible? block sources default_alias stage)
				(stage_direct_prepare_semantic_candidate? consumed_probe_ids consumed_source_probe_ids stage_output_ids stage)))))
		direct)))

(define query_block_stages_to_prepare_base (lambda (block)
	(query_block_stages_to_prepare_base_using (qb_stages block) block)))

(define selected_group_cache_keys (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (keys stage)
		(if (and (group_stage? stage) (source_is_base_table? (gs_input stage)))
			(set_assoc keys (group_stage_cache_owner_key stage) true)
			keys))
		'())))

(define include_shared_group_cache_stages (lambda (block_stages selected)
	(begin
		(define selected_group_caches (selected_group_cache_keys selected))
		(define selected_ids (reduce (coalesceNil selected '()) (lambda (ids stage)
			(set_assoc ids (gs_id stage) true)) '()))
		(if (empty_list? selected_group_caches)
			selected
			(filter (coalesceNil block_stages '()) (lambda (stage)
				(or
					(has_assoc? selected_ids (gs_id stage))
					(and
						(group_stage? stage)
						(and
							(source_is_base_table? (gs_input stage))
							(has_assoc? selected_group_caches (group_stage_cache_owner_key stage)))))))))))

(define query_block_stages_to_prepare_using (lambda (all_stages block)
	(begin
		(define prelimit_stage_ids (if (late_projection_candidate_block? block)
			(stage_ids_for_sources_with_closure all_stages (query_block_prelimit_sources block))
			nil))
		(include_shared_group_cache_stages
			(qb_stages block)
			(filter (query_block_stages_to_prepare_base_using all_stages block) (lambda (stage)
				(or
					(nil? prelimit_stage_ids)
					(stage_id_in? stage prelimit_stage_ids))))))))

(define query_block_stages_to_prepare (lambda (block)
	(query_block_stages_to_prepare_using (qb_stages block) block)))

(define query_block_bounded_scalar_probe_recipe_keys (lambda (block entries)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(define bounded_context (or
			(probe_limit_small_enough? (qb_limit block))
			(probe_context_unique_point? sources default_alias (qb_where block))))
		(if (not bounded_context)
			'()
			(begin
				(define prelimit_keys (scalar_query_probe_recipe_keys
					(query_block_prelimit_scalar_query_probe_recipe_entries block)))
				(reduce entries (lambda (keys entry)
					(match entry
						'(stage requested_col) (begin
							(define key (scalar_query_probe_recipe_key stage requested_col))
							(if (has_assoc? prelimit_keys key) keys (set_assoc keys key true)))
						_ keys)) '()))))))

(define prepare_simple_query_block_physical_core_chosen (lambda (block)
	(begin
		(define stage_lookup (query_block_stage_lookup block))
		(if (empty_list? (qb_stages block))
			(begin
				(define probe_recipe_entries (query_block_scalar_query_probe_recipe_entries block))
				(define bounded_probe_recipe_keys
					(query_block_bounded_scalar_probe_recipe_keys block probe_recipe_entries))
				(define probe_recipe_plans
					(scalar_query_probe_recipe_plans stage_lookup probe_recipe_entries bounded_probe_recipe_keys))
				(define recipe_block (query_block_with_scalar_query_probe_recipes block probe_recipe_entries))
				(define probe_recipe_bindings
					(scalar_query_probe_recipe_bindings probe_recipe_plans))
				(define probe_recipe_prepares
					(scalar_query_probe_recipe_prepare_exprs probe_recipe_plans))
				(list
					(merge (list probe_recipe_prepares probe_recipe_bindings))
					recipe_block))
			(begin
				(define stage_catalog (query_block_stage_catalog block))
				(define eager_stages (query_block_stages_to_prepare_using stage_lookup block))
				(define dependency_graph (stage_dependency_graph stage_lookup))
				(define raw_prepared_block (if (single_source? (qb_sources block))
					(query_block_without_stages_after_prepare_using stage_lookup block)
					(query_block_with_prepared_sources_using stage_lookup block)))
				(define probe_recipe_entries
					(query_block_scalar_query_probe_recipe_entries raw_prepared_block))
				(define bounded_probe_recipe_keys
					(query_block_bounded_scalar_probe_recipe_keys raw_prepared_block probe_recipe_entries))
				(define probe_recipe_plans
					(scalar_query_probe_recipe_plans_using_graph
						stage_lookup dependency_graph probe_recipe_entries bounded_probe_recipe_keys))
				(define prepared_block
					(query_block_with_scalar_query_probe_recipes raw_prepared_block probe_recipe_entries))
				(define probe_recipe_bindings
					(scalar_query_probe_recipe_bindings probe_recipe_plans))
				(define probe_recipe_prepares
					(scalar_query_probe_recipe_prepare_exprs probe_recipe_plans))
				(define lazy_catalog (stages_without_ids stage_catalog (stage_ids eager_stages)))
				(define core_block (query_block_without_stages
					(query_block_with_stage_catalog prepared_block stage_catalog)))
				(define lazy_stages (group_cache_stages_from_sources lazy_catalog (qb_sources core_block)))
				(list
					(merge (list
						probe_recipe_prepares
						probe_recipe_bindings
						(lazy_stage_prepare_bindings stage_catalog lazy_stages)
						(prepared_stage_bindings eager_stages)
						(lower_unique_stage_prepares_with_graph dependency_graph stage_lookup eager_stages)
						(lower_stage_materialize_all eager_stages)))
					core_block))))))

(define prepare_simple_query_block_physical_core (lambda (block)
	(prepare_simple_query_block_physical_core_chosen
		(query_block_with_physical_membership_choices block))))

(define lower_simple_query_block_with_cataloged_stages (lambda (block)
	(begin
		(define prepared (prepare_simple_query_block_physical_core block))
		(define prelude (nth prepared 0))
		(define core_block (nth prepared 1))
		(if (empty_list? prelude)
			(lower_query_block_core core_block)
			(cons (quote !begin) (merge (list
				prelude
				(list (lower_query_block_core core_block)))))))))

(define lower_query_block_with_cataloged_stages (lambda (block)
	(if (empty_list? (qb_stages block))
		(lower_simple_query_block_with_cataloged_stages block)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(lower_grouped_query_block_with_stages block)
			(begin
				(define fused_row_number (lower_fused_row_number_block block))
				(if (not (nil? fused_row_number))
					fused_row_number
					(lower_simple_query_block_with_cataloged_stages block)))))))

(define lower_query_block_with_stages (lambda (block)
	(lower_query_block_with_cataloged_stages (query_block_with_full_stage_catalog block))))

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

(define source_table_expr_using (lambda (stages src)
	(begin
		(define table_expr (source_table_expr src))
		(define group_cache_stage (stage_for_group_cache_source stages src))
		(if (nil? group_cache_stage)
			table_expr
			(list (quote !begin)
				(stage_prepare_call_expr group_cache_stage)
				table_expr)))))

(define stage_prepare_key (lambda (stage)
	(concat "__prepare_stage_" (fnv_hash (gs_id stage)))))

(define stage_prepare_call_expr (lambda (stage)
	(list (quote apply)
		(list (list (quote context) "session") (stage_prepare_key stage))
		(quoted_runtime_list '()))))

(define lazy_stage_prepare_binding (lambda (dependency_graph stage stage_catalog)
	(begin
		(define dependencies (stage_dependency_closure_using_graph dependency_graph stage))
		(list
			(list (quote context) "session")
			(stage_prepare_key stage)
			(list (quote once)
				(list (quote lambda)
					'()
					(list (quote !begin)
						(lower_stage_prepare_using dependencies stage_catalog stage)
						true)))))))

(define lazy_stage_prepare_bindings (lambda (stages selected)
	(begin
		(define dependency_graph (stage_dependency_graph stages))
		(map (coalesceNil selected '()) (lambda (stage)
			(lazy_stage_prepare_binding dependency_graph stage stages))))))

(define prepared_stage_binding (lambda (stage)
	(list
		(list (quote context) "session")
		(stage_prepare_key stage)
		(list (quote once) (list (quote lambda) '() true)))))

(define prepared_stage_bindings (lambda (stages)
	(map (coalesceNil stages '()) prepared_stage_binding)))

(define lower_query_block_core_with_lazy_prepares (lambda (stage_catalog block)
	(begin
		(define lazy_stages (group_cache_stages_from_sources stage_catalog (qb_sources block)))
		(define core_plan (lower_query_block_core block))
		(if (empty_list? lazy_stages)
			core_plan
			(cons (quote !begin)
				(merge (list
					(lazy_stage_prepare_bindings stage_catalog lazy_stages)
					(list core_plan))))))))

(define query_block_has_aggregates? (lambda (block)
	(not (empty_list? (stage_aggregates_for_fields (qb_fields block))))))

(define direct_base_group_stage_supported? (lambda (block stage)
	(begin
		(define src (gs_input stage))
		(define alias (source_alias src))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define key_names (group_key_cols keys))
		(define ags (gs_aggregates stage))
		(define order_items (coalesceNil (qb_order block) '()))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(and (not (empty_list? (qb_group block)))
			(and (nil? (qb_having block))
				(and (or (empty_list? order_items)
					(and (or
						(query_limit_active? (qb_offset block) (qb_limit block))
						(expr_contains_session_dependency? condition))
						(direct_group_order_supported? alias keys key_names ags order_items)))
					(and (query_block_has_aggregates? block)
						(not (reduce ags
							(lambda (found ag) (or found (count_distinct_descriptor? ag)))
							false)))))))))

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

(define downstream_sources_preserve_driver_rows? (lambda (sources default_alias final_condition)
	(begin
		(define rest_sources (cdr sources))
		(and
			(reduce rest_sources (lambda (ok src) (and ok (source_outer? src))) true)
			(not (expr_refs_any_alias? default_alias (source_aliases rest_sources) final_condition))))))

(define late_projection_downstream_safe? (lambda (stages sources default_alias final_condition)
	(begin
		(define rest_sources (cdr sources))
		(and
			(reduce rest_sources (lambda (ok src)
				(and ok (or
					(source_outer? src)
					(and (stage_output_relation? (source_relation src))
						(nil? (stage_by_id stages (stage_output_relation_id (source_relation src))))))))
				true)
			(not (expr_refs_any_alias? default_alias (source_aliases rest_sources) final_condition))))))

(define source_alias_in_sources? (lambda (alias sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(or found (equal? alias (source_alias src))))
		false)))

(define symbol_expr? (lambda (expr)
	(equal? expr (symbol (string expr)))))

(define join_outer_symbol? (lambda (sources current_alias expr)
	(if (not (symbol_expr? expr))
		false
		(match (string expr)
			(concat alias "." _col)
			(and
				(not (equal? alias current_alias))
				(source_alias_in_sources? alias sources))
			_ false))))

(define mark_outer_join_symbols (lambda (sources current_alias expr)
	(if (join_outer_symbol? sources current_alias expr)
		(list (quote outer) expr)
		(match expr
			((symbol quote) _value) expr
			((symbol lambda) _params _body) expr
			((symbol lambda) _params _body _numvars) expr
			(cons head tail) (cons head (map tail (lambda (item)
				(mark_outer_join_symbols sources current_alias item))))
			_ expr))))

(define late_projection_sources_preserve_rows? (lambda (sources prelimit_sources)
	(reduce (coalesceNil sources '()) (lambda (ok src)
		(and ok
			(or
				(source_alias_in_sources? (source_alias src) prelimit_sources)
				(source_outer? src))))
		true)))

(define ordered_join_limit_requires_complete_rows? (lambda (sources default_alias final_condition offset_value limit_value)
	(and
		(query_limit_active? offset_value limit_value)
		(and (not (empty_list? (cdr sources)))
			(not (downstream_sources_preserve_driver_rows? sources default_alias final_condition))))))

(define driver_memberships_for_source (lambda (src expr)
	(begin
		(define marker (driver_membership_probe_term expr))
		(if (not (nil? marker))
			(begin
				(define probe_col (direct_column_name_for_alias src (nth marker 1)))
				(if (nil? probe_col)
					'()
					(list (list (nth marker 0) (nth marker 1) probe_col expr))))
			(match expr
				(cons _head tail) (merge_unique (map tail (lambda (item)
					(driver_memberships_for_source src item))))
				_ '())))))

/* A scan may contain several branch-local membership predicates. Each RecSet is
bound once, but every marker remains at its original AND/OR position and calls
the shared $recset_contains row callback with its own RecSet argument. Do not
collapse guarded alternatives into one driver RecSet: that would discard rows
accepted by non-membership branches. The storage analyzer may use a RecSet as a
scan boundary only when the complete predicate implies it. */

(define membership_recset_var (lambda (membership)
	(symbol (concat "__membership_recset_" (fnv_hash (gs_id (nth membership 0)))))))

(define replace_driver_membership_markers (lambda (expr memberships)
	(begin
		/* Match the marker structurally, but never recurse into its embedded
		group-stage. A stage is IR data containing booleans and expressions of its
		own; treating that payload as part of the surrounding predicate would
		corrupt stage facts while replacing a row-level membership marker. */
		(define marker (driver_membership_probe_term expr))
		(define membership (if (nil? marker)
			nil
			(reduce memberships (lambda (found item)
				(if (not (nil? found))
					found
					(if (and (equal? (gs_id (nth marker 0)) (gs_id (nth item 0)))
						(equal? (nth marker 1) (nth item 1)))
						item
						nil))) nil)))
		(if (not (nil? membership))
			(recset_contains_call_expr (membership_recset_var membership))
			(if (not (nil? marker))
				expr
				(match expr
					(cons head tail) (cons head (map tail (lambda (item)
						(replace_driver_membership_markers item memberships))))
					_ expr))))))

(define membership_recset_bindings (lambda (src memberships)
	(filter (map memberships (lambda (membership)
		(begin
			(define expr (recset_project_join_expr_for_membership src membership))
			(if (nil? expr) nil (list membership (membership_recset_var membership) expr)))))
		(lambda (binding) (not (nil? binding))))))

(define wrap_membership_recset_bindings (lambda (bindings body)
	(if (empty_list? bindings)
		body
		/* The projections are independent. Bind them in one application instead
		of emitting a depth-proportional lambda tower; this keeps compilation and
		generated-plan size linear with a small constant for large OR clouds. */
		(cons
			(list (quote lambda)
				(map bindings (lambda (binding) (nth binding 1)))
				body)
			(map bindings (lambda (binding) (nth binding 2)))))))

(define expr_contains_driver_membership? (lambda (expr)
	(if (not (nil? (driver_membership_probe_term expr)))
		true
		(match expr
			(cons _head tail) (reduce tail (lambda (found item)
				(or found (expr_contains_driver_membership? item))) false)
			_ false))))

(define membership_guard_or_expr (lambda (expr)
	(if (not (nil? (driver_membership_probe_term expr)))
		nil
		(match expr
			(cons head tail)
			(if (or (equal? head (quote or)) (equal? head (symbol "or")))
				(if (reduce tail (lambda (found item)
					(or found (expr_contains_driver_membership? item))) false)
					expr
					(reduce tail (lambda (found item)
						(if (not (nil? found)) found (membership_guard_or_expr item))) nil))
				(reduce tail (lambda (found item)
					(if (not (nil? found)) found (membership_guard_or_expr item))) nil))
			_ nil))))

(define membership_binding_for_marker (lambda (bindings marker)
	(reduce bindings (lambda (found binding)
		(if (not (nil? found))
			found
			(begin
				(define membership (nth binding 0))
				(if (and (equal? (gs_id (nth marker 0)) (gs_id (nth membership 0)))
					(equal? (nth marker 1) (nth membership 1)))
					binding
					nil)))) nil)))

(define membership_branch_candidate_recset (lambda (src source_table branch bindings)
	(begin
		(define markers (driver_memberships_for_source src branch))
		(if (empty_list? markers)
			(begin
				(define cols (extract_columns_for_alias src branch))
				(list (quote scan_recset)
					'(session "__memcp_tx")
					source_table
					(cons (quote list) cols)
					(list (quote lambda)
						(map cols (lambda (col) (scan_callback_symbol_for_alias (source_alias src) col)))
						(list (quote optimize) (lower_column_expr_for_alias src branch)))))
			(begin
				(define branch_bindings (filter (map markers (lambda (marker)
					(membership_binding_for_marker bindings marker)))
					(lambda (binding) (not (nil? binding)))))
				(if (not (equal? (count branch_bindings) (count markers)))
					nil
					(if (single_source? branch_bindings)
						(nth (car branch_bindings) 1)
						(list (quote recset_union)
							(cons (quote list) (map branch_bindings (lambda (binding) (nth binding 1))))))))))))

/* When every OR alternative has a target-table candidate RecSet, the general
truth-set law T_R(p OR q) = T_R(p) union T_R(q) provides a safe scan boundary.
Membership alternatives contribute projected supersets and ordinary
alternatives contribute filtered base RecSets. Because some inputs may be
candidate supersets rather than exact truth sets, the unchanged full predicate
still runs on the union and enforces branch-local guards. Keep this code phrased
in terms of boolean RecSet formulas: future optimizers can add intersection,
factoring, or other proven set transformations without adding SQL-shape cases. */
(define membership_or_candidate_recset (lambda (src source_table condition bindings)
	(begin
		(define or_expr (membership_guard_or_expr condition))
		(if (nil? or_expr)
			nil
			(begin
				(define candidates (map (cdr or_expr) (lambda (branch)
					(membership_branch_candidate_recset src source_table branch bindings))))
				(if (reduce candidates (lambda (missing candidate)
					(or missing (nil? candidate))) false)
					nil
					(list (quote recset_union) (cons (quote list) candidates))))))))

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
				(if (direct_base_group_plan_preferred? group_stage)
					(lower_direct_base_group_stage group_stage fields (qb_order block) (qb_offset block) (qb_limit block))
					(lower_group_stage group_stage)))
			(begin
				(define alias (source_alias src))
				(define condition (combine_where (qb_where block) (source_join_expr src)))
				(define order_items (coalesceNil (qb_order block) '()))
				(define scan_order_supported (order_items_belong_to_source? src order_items))
				(define bounded (query_limit_active? (qb_offset block) (qb_limit block)))
				(define memberships (driver_memberships_for_source src condition))
				(define membership_bindings (membership_recset_bindings src memberships))
				(define bound_memberships (map membership_bindings (lambda (binding) (nth binding 0))))
				/* A membership predicate which is implied by the whole WHERE clause is
				eligible to become the scan driver. A branch-local predicate below OR is
				not: its RecSet must remain a probe or rows accepted by sibling branches
				would disappear. This distinction also preserves the established fast
				single-IN path while allowing several guarded RecSets in one scan. */
				(define direct_membership (driver_membership_for_source src condition))
				(define prefer_membership_filter (and scan_order_supported
					(and bounded
						(broad_driver_order_membership_probe? (qb_facts block)))))
				(define membership_driver (and
					(not (nil? direct_membership))
					(and (not prefer_membership_filter)
						(and (not (empty_list? membership_bindings))
							(and (empty_list? (cdr membership_bindings))
								(equal? direct_membership (car bound_memberships)))))))
				(define membership_filter (and
					(not membership_driver)
					(not (empty_list? membership_bindings))))
				(define filter_condition (if membership_driver
					(strip_driver_membership_for_source src condition direct_membership)
					(replace_driver_membership_markers condition bound_memberships)))
				(define filtercols (merge_unique (list
					(if membership_filter (list "$recset_contains") '())
					(extract_columns_for_alias src filter_condition))))
				(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
					(extract_columns_for_alias src expr)))))
				(define ordercols (if (empty_list? order_items) '() (scan_order_sort_columns_for_alias src order_items)))
				(define mapcols fieldcols)
				(define source_table (source_table_expr_using (query_block_stage_catalog block) src))
				(define membership_candidates (if membership_filter
					(membership_or_candidate_recset src source_table condition membership_bindings)
					nil))
				(define table_expr (if membership_driver
					(nth (car membership_bindings) 2)
					(coalesceNil membership_candidates source_table)))
				(define filter_expr (list (quote lambda)
					(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
					(list (quote optimize) (lower_column_expr_for_alias src filter_condition))))
				(define map_expr (list (quote lambda)
					(map mapcols (lambda (col) (symbol (concat alias "." col))))
					(list (quote resultrow)
						(cons (quote list) (map_assoc fields (lambda (title expr)
							(lower_column_expr_for_alias src expr)))))))
				(define scan_plan (if (and (empty_list? order_items) (not bounded))
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
						(begin
							(define scan_expr (list (quote scan_order)
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
								(source_outer? src)))
							scan_expr)
						(neumann_fail "build_queryplan" "single-source ORDER BY requires a storage carrier"))))
				(if membership_filter
					(wrap_membership_recset_bindings membership_bindings scan_plan)
					scan_plan))))))

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

(define join_column_recipe (lambda (sources default_alias needed_exprs)
	(reduce (coalesceNil needed_exprs '()) (lambda (columns_by_alias expr)
		(collect_join_columns_acc sources default_alias nil expr columns_by_alias)) '())))

(define join_recipe_mapcols (lambda (recipe alias)
	(if (nil? recipe) nil (qassoc_get recipe alias '()))))

(define join_filter_cols_for_alias (lambda (all_sources default_alias alias condition)
	(extract_columns_for_join_alias all_sources default_alias alias condition)))

/* Partition WHERE conjuncts without changing the logical join order. Terms
that touch the current nullable source run in its map callback, after scan has
either bound a real row or supplied the synthetic NULL row. */
(define physical_partition_condition_terms (lambda (default_alias current_source future_sources terms scan_ready post_outer pending)
	(match (coalesceNil terms '())
		(cons term rest) (if (or
			(expr_contains_orc_column? term)
			(expr_refs_any_alias? default_alias (source_aliases future_sources) term))
			(physical_partition_condition_terms default_alias current_source future_sources rest scan_ready post_outer (cons term pending))
			(if (and
				(source_outer? current_source)
				(expr_refs_alias? default_alias (source_alias current_source) term))
				(physical_partition_condition_terms default_alias current_source future_sources rest scan_ready (cons term post_outer) pending)
				(physical_partition_condition_terms default_alias current_source future_sources rest (cons term scan_ready) post_outer pending)))
		_ (list
			(combine_where_terms (reverse scan_ready) true)
			(combine_where_terms (reverse post_outer) true)
			(combine_where_terms (reverse pending) true)))))

(define physical_partition_condition (lambda (default_alias current_source future_sources condition)
	(physical_partition_condition_terms
		default_alias
		current_source
		future_sources
		(split_and_terms (coalesceNil condition true))
		'()
		'()
		'())))

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

(define recset_contains_callback_symbol (symbol "__recset_contains"))

(define scan_callback_symbol_for_alias (lambda (alias col)
	(if (or (equal? col "$break") (equal? col "$tx"))
		(symbol col)
		(if (equal? col "$recset_contains")
			recset_contains_callback_symbol
			(symbol (concat alias "." col))))))

(define recset_contains_call_expr (lambda (recset_expr)
	(list recset_contains_callback_symbol recset_expr)))

(define special_scan_col_expr? (lambda (expr)
	(or (equal? expr recset_contains_callback_symbol)
		(equal? expr (symbol "$recset_contains")))))

(define lower_special_scan_col_exprs (lambda (expr)
	(match expr
		(cons head tail) (cons
			(if (special_scan_col_expr? head) recset_contains_callback_symbol head)
			(map tail lower_special_scan_col_exprs))
		_ (if (special_scan_col_expr? expr)
			recset_contains_callback_symbol
			expr))))

(define join_ordered_stream_plan (lambda (schema all_sources plan default_alias needed_exprs final_condition fields order_items offset_value limit_value stages)
	(begin
		(define ordered_aliases (join_optimizer_tree_aliases plan))
		(define ordered_sources (join_optimizer_sources_for_order all_sources ordered_aliases))
		(define src (car ordered_sources))
		(define remaining_sources (cdr ordered_sources))
		(define remaining_plan (physical_join_plan_for_sources remaining_sources))
		(define alias (source_alias src))
		(define condition_parts (physical_partition_condition default_alias src remaining_sources final_condition))
		(define local_condition (nth condition_parts 0))
		(define remaining_condition (combine_where
			(nth condition_parts 2)
			(join_optimizer_node_condition (join_optimizer_tree_predicates plan))))
		(define condition (combine_where (source_join_expr src) local_condition))
		(define order_parts (split_order_items_for_join_driver
			ordered_sources default_alias src order_items stages final_condition '()))
		(define driver_order_items (nth order_parts 0))
		(define remaining_order_items (nth order_parts 1))
		(define membership (driver_membership_for_source src condition))
		(define membership_table_expr (if (nil? membership) nil
			(recset_project_join_expr_for_membership src membership)))
		(define effective_membership (if (nil? membership_table_expr) nil membership))
		(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
		(define filtercols (join_cols_for_alias all_sources default_alias alias (list effective_condition)))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			(list (quote optimize) (lower_column_expr_for_join all_sources default_alias effective_condition))))
		(define row_expr (cons (quote list) (lower_join_result_fields all_sources default_alias fields)))
		(define offset (coalesceNil offset_value 0))
		(define limit (coalesceNil limit_value -1))
		(define finite_limit (not (equal? limit -1)))
		(define mapcols (merge_unique (list
			(join_cols_for_alias all_sources default_alias alias needed_exprs)
			(if finite_limit (list "$break") '()))))
		(define end (list (quote +) offset limit))
		(define projection_reduce (list (quote lambda) (list (quote __matched) (quote __row))
			(list (quote begin)
				(list (quote define) (quote __position) (list (quote +) (quote __accepted) (quote __matched)))
				(list (quote if)
					(list (quote and)
						(list (quote >=) (quote __position) offset)
						(list (quote or)
							(list (quote equal?) limit -1)
							(list (quote <) (quote __position) end)))
					(list (quote resultrow) (quote __row))
					nil)
				(list (quote +) (quote __matched) 1))))
		(define projection (build_join_scan_reduce_using_recipe
			schema all_sources remaining_plan default_alias needed_exprs remaining_condition row_expr
			remaining_order_items 0 -1 true nil stages
			projection_reduce
			0
			(list (quote lambda) (list (quote count) (quote matched))
				(list (quote +) (quote count) (quote matched)))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			(list (quote lambda) (list (quote __accepted))
				(if finite_limit
					(list (quote if)
						(list (quote >=) (quote __accepted) (list (quote +) offset limit))
						(list (symbol "$break"))
						projection)
					projection))))
		(define reduce_expr (list (quote lambda)
			(list (quote __accepted) (quote __continuation))
			(list
				(list (quote lambda) (list (quote __matched))
					(list (quote +) (quote __accepted) (quote __matched)))
				(list (quote __continuation) (quote __accepted)))))
		(define scan_expr (list (quote scan_order)
			'(session "__memcp_tx")
			(coalesceNil membership_table_expr (source_table_expr_using stages src))
			(cons (quote list) filtercols)
			filter_expr
			(cons (quote list) (scan_order_sort_columns_for_join_driver
				ordered_sources default_alias src driver_order_items stages final_condition))
			(cons (quote list) (order_dirs driver_order_items))
			0 0 -1
			(cons (quote list) mapcols)
			map_expr reduce_expr 0 false))
		(list
			(list (quote lambda) (list (quote __accepted)) nil)
			scan_expr))))

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

(define build_join_row_number_scan_pipeline (lambda (schema all_sources src default_alias needed_exprs remaining_condition row_expr stage_filter membership_var membership_filter column_recipe stages result_mode combines_state continuation outer_scan)
	(begin
		(define alias (source_alias src))
		(define stage (nth stage_filter 0))
		(define filter (nth stage_filter 1))
		(define mode (nth filter 0))
		(define limit (nth filter 1))
		(define stripped_condition (nth filter 2))
		(match stage
			'(_ _id _stage_src col sortcols sortdirs _partitioncount mapcols _mapfn _reducefn _reduceinit _facts)
			(begin
				(define membership_filter_expr (if membership_filter
					(recset_contains_call_expr membership_var)
					true))
				(define filter_condition (combine_where membership_filter_expr stripped_condition))
				(define filtercols (merge_unique (list
					(if membership_filter (list "$recset_contains") '())
					(join_filter_cols_for_alias all_sources default_alias alias stripped_condition))))
				(define raw_mapcols (if (nil? column_recipe)
					(join_cols_for_alias all_sources default_alias alias needed_exprs)
					(join_recipe_mapcols column_recipe alias)))
				(define scan_mapcols (merge_unique (list (without_col raw_mapcols col) mapcols)))
				(define table_expr (source_table_expr_using stages src))
				(define rewritten_row_expr (replace_lowered_row_number_symbol alias col row_expr))
				(define filter_expr (list (quote lambda)
					(map filtercols (lambda (filter_col) (scan_callback_symbol_for_alias alias filter_col)))
					(list (quote optimize) (lower_column_expr_for_join all_sources default_alias filter_condition))))
				(define continuation_expr (list (quote lambda) (list (quote __row_number))
					(continuation remaining_condition rewritten_row_expr '())))
				(define map_expr (list (quote lambda)
					(map scan_mapcols (lambda (map_col) (symbol (concat alias "." map_col))))
					(list (quote list)
						(row_number_scan_partition_expr alias mapcols)
						continuation_expr)))
				(define row_number_reduce_expr (list (quote lambda) (list (quote state) (quote mapped))
					(list (quote begin)
						(list (quote define) (quote prev_partition) (list (quote car) (quote state)))
						(list (quote define) (quote prev_rownum) (list (quote cadr) (quote state)))
						(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
						(list (quote define) (quote continuation) (list (quote cadr) (quote mapped)))
						(list (quote define) (quote same_partition) (list (quote and)
							(list (quote not) (list (quote equal?) (quote prev_rownum) 0))
							(list (quote equal?) (quote row_partition) (quote prev_partition))))
						(list (quote define) (quote next_rownum) (list (quote if) (quote same_partition) (list (quote +) (quote prev_rownum) 1) 1))
						(list (quote if)
							(row_number_count_match_expr mode (quote next_rownum) limit)
							(list (quote !begin)
								(list (quote continuation) (quote next_rownum))
								(list (quote list) (quote row_partition) (quote next_rownum)))
							(list (quote list) (quote row_partition) (quote next_rownum))))))
				(define dataset_reduce_expr (if (join_scan_reduce? result_mode)
					(begin
						(define reduce_value (join_scan_reduce_expr result_mode combines_state))
						(list (quote lambda) (list (quote state) (quote mapped))
							(list (quote begin)
								(list (quote define) (quote prev_partition) (list (quote car) (quote state)))
								(list (quote define) (quote prev_rownum) (list (quote cadr) (quote state)))
								(list (quote define) (quote aggregate) (list (quote car) (list (quote cdr) (list (quote cdr) (quote state)))))
								(list (quote define) (quote row_partition) (list (quote car) (quote mapped)))
								(list (quote define) (quote continuation) (list (quote cadr) (quote mapped)))
								(list (quote define) (quote same_partition) (list (quote and)
									(list (quote not) (list (quote equal?) (quote prev_rownum) 0))
									(list (quote equal?) (quote row_partition) (quote prev_partition))))
								(list (quote define) (quote next_rownum) (list (quote if) (quote same_partition) (list (quote +) (quote prev_rownum) 1) 1))
								(list (quote define) (quote next_aggregate)
									(list (quote if)
										(row_number_count_match_expr mode (quote next_rownum) limit)
										(list reduce_value (quote aggregate) (list (quote continuation) (quote next_rownum)))
										(quote aggregate)))
								(list (quote list) (quote row_partition) (quote next_rownum) (quote next_aggregate)))))
					row_number_reduce_expr))
				(define scan_expr (list (quote scan_order)
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
					dataset_reduce_expr
					(if (join_scan_reduce? result_mode)
						(list (quote list) nil 0 (join_scan_neutral_expr result_mode))
						(list (quote list) nil 0))
					(or outer_scan (source_outer? src))))
				(if (join_scan_reduce? result_mode)
					(list (quote car) (list (quote cdr) (list (quote cdr) scan_expr)))
					scan_expr))
			_ (neumann_fail "build_queryplan" "malformed ROW_NUMBER stage")))))

(define join_scan_reduce? (lambda (result_mode)
	(and (list? result_mode)
		(and (not (empty_list? result_mode))
			(equal? (car result_mode) (quote reduce))))))

(define join_scan_reduce_skip (quote __join_scan_reduce_skip))

(define join_scan_skip_expr (lambda (result_mode)
	(if (join_scan_reduce? result_mode)
		(list (quote quote) join_scan_reduce_skip)
		nil)))

(define join_scan_reduce_expr (lambda (result_mode combines_state)
	(if (join_scan_reduce? result_mode)
		(begin
			(define raw_reduce (if combines_state
				(coalesceNil (nth result_mode 3) (nth result_mode 1))
				(nth result_mode 1)))
			(list (quote lambda) (list (quote acc) (quote value))
				(list (quote if)
					(list (quote and)
						(list (quote symbol?) (quote value))
						(list (quote equal?) (quote value) (list (quote quote) join_scan_reduce_skip)))
					(quote acc)
					(list raw_reduce (quote acc) (quote value)))))
		nil)))

(define join_scan_neutral_expr (lambda (result_mode)
	(if (join_scan_reduce? result_mode) (nth result_mode 2) nil)))

(define join_scan_shard_reduce_expr (lambda (result_mode)
	(if (join_scan_reduce? result_mode) (join_scan_reduce_expr result_mode true) nil)))

(define build_join_scan_leaf_using_recipe (lambda (schema all_sources leaf future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode continuation outer_scan)
	(begin
		(define src (physical_join_leaf_source all_sources leaf))
		(define future_sources (join_optimizer_sources_for_order all_sources future_aliases))
		(if (not (source_is_base_table? src))
			(neumann_fail "build_queryplan" "multi-source query-block lowering only supports base tables after untangle")
			true)
		(define alias (source_alias src))
		(define condition_parts (physical_partition_condition default_alias src future_sources final_condition))
		(define local_condition (nth condition_parts 0))
		(define post_outer_condition (nth condition_parts 1))
		(define remaining_condition (nth condition_parts 2))
		(define owned_condition (join_optimizer_node_condition
			(join_optimizer_leaf_predicates leaf)))
		(define condition (combine_where
			(source_join_expr src)
			(combine_where owned_condition local_condition)))
		(define ordered_sources (cons src future_sources))
		(define order_parts (split_order_items_for_join_driver
			ordered_sources default_alias src order_items stages final_condition '()))
		(define current_order_items (nth order_parts 0))
		(define remaining_order_items (nth order_parts 1))
		(define membership (driver_membership_for_source src condition))
		(define delay_limit_after_join (ordered_join_limit_requires_complete_rows?
			(join_optimizer_sources_for_order all_sources (cons alias future_aliases))
			default_alias final_condition offset_value limit_value))
		(define membership_table_expr (if (or (nil? membership) (or delay_limit_after_join (not allow_membership_recset)))
			nil
			(recset_project_join_expr_for_membership src membership)))
		(define membership_driver (and (not (nil? membership_table_expr)) (empty_list? future_aliases)))
		(define membership_filter (and (not (nil? membership_table_expr)) (not membership_driver)))
		(define effective_membership (if (nil? membership_table_expr) nil membership))
		(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
		(define membership_var (symbol "__membership_recset"))
		(define membership_filter_expr (if membership_filter
			(recset_contains_call_expr membership_var)
			true))
		(define filter_condition (combine_where membership_filter_expr effective_condition))
		(define row_number_stage_filter (row_number_stage_for_source stages src effective_condition))
		(define filtercols (merge_unique (list
			(if membership_filter (list "$recset_contains") '())
			(join_filter_cols_for_alias all_sources default_alias alias effective_condition))))
		(define recipe_mapcols (join_recipe_mapcols column_recipe alias))
		(define raw_mapcols (if (nil? column_recipe)
			(join_cols_for_alias all_sources default_alias alias needed_exprs)
			recipe_mapcols))
		(define mapcols raw_mapcols)
		(define table_expr (if membership_driver membership_table_expr (source_table_expr_using stages src)))
		(define lowered_filter_condition (mark_outer_join_symbols
			all_sources
			alias
			(lower_column_expr_for_join all_sources default_alias filter_condition)))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			(list (quote optimize) lowered_filter_condition)))
		(define continuation_expr (continuation remaining_condition row_expr remaining_order_items))
		(define map_body (if (equal? post_outer_condition true)
			continuation_expr
			(list (quote if)
				(list (quote optimize) (lower_column_expr_for_join all_sources default_alias post_outer_condition))
				continuation_expr
				(join_scan_skip_expr result_mode))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			map_body))
		(define combines_state (not (empty_list? future_aliases)))
		(define reduce_expr (join_scan_reduce_expr result_mode combines_state))
		(define scan_expr
			(if (not (nil? row_number_stage_filter))
				(build_join_row_number_scan_pipeline schema all_sources src default_alias needed_exprs remaining_condition row_expr row_number_stage_filter membership_var membership_filter column_recipe stages result_mode combines_state continuation outer_scan)
				(if (and (empty_list? current_order_items) (not (query_limit_active? offset_value limit_value)))
					(list (quote scan)
						'(session "__memcp_tx")
						table_expr
						(cons (quote list) filtercols)
						filter_expr
						(cons (quote list) mapcols)
						map_expr
						reduce_expr
						(join_scan_neutral_expr result_mode)
						(join_scan_shard_reduce_expr result_mode)
						(or outer_scan (source_outer? src)))
					(list (quote scan_order)
						'(session "__memcp_tx")
						table_expr
						(cons (quote list) filtercols)
						filter_expr
						(cons (quote list) (if (empty_list? current_order_items) '()
							(scan_order_sort_columns_for_join_driver ordered_sources default_alias src current_order_items stages final_condition)))
						(cons (quote list) (if (empty_list? current_order_items) '() (order_dirs current_order_items)))
						0
						(coalesceNil offset_value 0)
						(coalesceNil limit_value -1)
						(cons (quote list) mapcols)
						map_expr
						reduce_expr
						(join_scan_neutral_expr result_mode)
						(or outer_scan (source_outer? src))))))
		(if membership_filter
			(list
				(list (quote lambda) (list membership_var) scan_expr)
				membership_table_expr)
			scan_expr))))

/* Consume the logical join tree recursively. The right subtree is lowered as
the continuation of the left subtree, so join-node boundaries and outer-join
ownership remain available until the physical scans are emitted. */
(define build_join_tree_scan_using_recipe (lambda (schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode continuation outer_scan)
	(match tree
		((symbol join-leaf) _alias)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode continuation outer_scan)
		((quote join-leaf) alias)
		(build_join_tree_scan_using_recipe schema all_sources (make_join_optimizer_leaf alias) future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode continuation outer_scan)
		((symbol join-leaf) _alias _predicates)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode continuation outer_scan)
		((quote join-leaf) alias predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(list (quote join-leaf) alias predicates) future_aliases default_alias needed_exprs
			final_condition row_expr order_items offset_value limit_value allow_membership_recset
			column_recipe stages result_mode continuation outer_scan)
		((symbol join-node) kind left right predicates)
		(begin
			(if (and (equal? kind (quote left-outer))
				(not (equal? (car right) (quote join-leaf))))
				(neumann_fail "build_queryplan" "LEFT JOIN with a composite nullable subtree is not implemented")
				true)
			(define right_aliases (join_optimizer_tree_aliases right))
			(define node_condition (join_optimizer_node_condition predicates))
			(build_join_tree_scan_using_recipe
				schema all_sources left (merge_unique (list right_aliases future_aliases))
				default_alias needed_exprs final_condition row_expr
				order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode
				(lambda (left_condition left_row_expr left_order_items)
					(build_join_tree_scan_using_recipe
						schema all_sources right future_aliases
						default_alias needed_exprs (combine_where left_condition node_condition) left_row_expr
						left_order_items 0 -1 allow_membership_recset column_recipe stages result_mode continuation
						(equal? kind (quote left-outer))))
				outer_scan))
		((quote join-node) kind left right predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(make_join_optimizer_node kind left right predicates) future_aliases
			default_alias needed_exprs final_condition row_expr order_items offset_value limit_value
			allow_membership_recset column_recipe stages result_mode continuation outer_scan)
		_ (neumann_fail "build_queryplan" "malformed logical join tree"))))

(define build_join_scan_with_mapper_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode)
	(begin
		(define tree (physical_join_plan_for_sources sources))
		(define residual_condition (if (nil? tree) final_condition
			(condition_without_join_tree_predicates final_condition tree)))
		(define terminal (lambda (remaining_condition final_row_expr remaining_order_items)
			(begin
				(if (not (empty_list? remaining_order_items))
					(neumann_fail "build_queryplan" "ORDER BY requires a storage carrier")
					true)
				(if (equal? (coalesceNil remaining_condition true) true)
					final_row_expr
					(list (quote if)
						(list (quote optimize) (lower_column_expr_for_join all_sources default_alias remaining_condition))
						final_row_expr
						(join_scan_skip_expr result_mode))))))
		(if (nil? tree)
			(terminal residual_condition row_expr order_items)
			(build_join_tree_scan_using_recipe
				schema all_sources tree '() default_alias needed_exprs residual_condition row_expr
				order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode
				terminal false)))))

(define build_join_scan_pipeline_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier column_recipe stages)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages (quote pipeline))))

(define build_join_scan_reduce_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier column_recipe stages reduce_expr neutral_expr shard_reduce_expr)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages
		(list (quote reduce) reduce_expr neutral_expr shard_reduce_expr))))

(define build_join_scan_sink (lambda (schema sources default_alias needed_exprs final_condition sink_expr stages)
	(build_join_scan_pipeline_using_recipe
		schema sources sources default_alias needed_exprs final_condition sink_expr
		'() 0 -1 false nil stages)))

/* A total order that cannot be factored along the logical join tree needs a
storage-backed intermediate relation.  Materializing joined rows in a Scheme
list would violate the relation-materialization invariant and scale with heap
objects.  This last-resort intermediate relation stays in the storage engine, where
scan_order can build the appropriate auto-index and apply OFFSET/LIMIT. */
(define join_order_intermediate_names (lambda (prefix amount)
	(map (produceN amount) (lambda (idx) (concat prefix idx)))))

(define join_order_intermediate_columns (lambda (names)
	(cons (quote list) (map names (lambda (name)
		(list (quote list) "column" name "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))))

(define join_order_intermediate_insert (lambda (table_expr names values)
	(list (quote insert)
		table_expr
		(cons (quote list) names)
		(list (quote list) (cons (quote list) values))
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		true)))

(define join_order_intermediate_result_fields (lambda (fields value_names)
	(match fields
		(cons title (cons _expr rest))
		(cons title (cons (symbol (car value_names))
			(join_order_intermediate_result_fields rest (cdr value_names))))
		_ '())))

(define join_order_intermediate_scan (lambda (table_expr fields value_names order_names order_items offset_value limit_value)
	(list (quote scan_order)
		'(session "__memcp_tx")
		table_expr
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		(cons (quote list) order_names)
		(cons (quote list) (order_dirs order_items))
		0
		(coalesceNil offset_value 0)
		(coalesceNil limit_value -1)
		(cons (quote list) value_names)
		(list (quote lambda) (map value_names symbol)
			(list (quote resultrow)
				(cons (quote list) (join_order_intermediate_result_fields fields value_names))))
		nil nil false)))

(define lower_join_order_through_intermediate (lambda (block fields scan_sources scan_plan default_alias needed_exprs final_condition order_items stage_catalog)
	(begin
		(define value_exprs (extract_assoc fields (lambda (_title expr) expr)))
		(define order_values (order_exprs order_items))
		(define value_names (join_order_intermediate_names "v" (count value_exprs)))
		(define order_names (join_order_intermediate_names "o" (count order_values)))
		(define column_names (merge (list value_names order_names)))
		(define lowered_values (map (merge (list value_exprs order_values)) (lambda (expr)
			(lower_column_expr_for_join scan_sources default_alias expr))))
		(define table_key (concat "__join_order_intermediate:" (uuid)))
		(define table_name (list (quote session) table_key))
		(define table_expr (list (quote table) (qb_schema block) table_name))
		(define fill_plan (build_join_scan_pipeline_using_recipe
			(qb_schema block) scan_sources scan_plan default_alias needed_exprs final_condition
			(join_order_intermediate_insert table_expr column_names lowered_values)
			'() 0 -1 false nil stage_catalog))
		(define scan_plan_expr (join_order_intermediate_scan table_expr fields value_names order_names order_items
			(qb_offset block) (qb_limit block)))
		(define drop_plan (list (quote droptable) (qb_schema block) table_name true))
		(list (quote !begin)
			(list (quote session) table_key (list (quote concat) ".join-order:" (list (quote uuid))))
			(list (quote createtable) (qb_schema block) table_name
				(join_order_intermediate_columns column_names)
				(quoted_runtime_list '("engine" "memory")) true)
			(list (quote try)
				(list (quote lambda) '() (list (quote !begin) fill_plan scan_plan_expr))
				(list (quote lambda) (list (quote __intermediate_error))
					(list (quote !begin) drop_plan (list (quote error) (quote __intermediate_error)))))
			drop_plan))))

/* ------------------------------------------------------------------------- */
/* Canonical physical prejoin relations                                      */

(define prejoin_primary_key_columns (lambda (src)
	(map (filter (get_schema (source_schema src) (source_relation src)) (lambda (col)
		(equal?? (col "Key") "PRI"))) (lambda (col) (col "Field")))))

(define prejoin_source_position (lambda (sources alias)
	(reduce (produceN (count sources)) (lambda (found i)
		(if (not (nil? found))
			found
			(if (equal?? (source_alias (nth sources i)) alias) i nil)))
		nil)))

(define prejoin_column_name (lambda (sources alias col)
	(begin
		(define pos (prejoin_source_position sources alias))
		(if (nil? pos)
			(neumann_fail "build_queryplan" (concat "prejoin source alias not found: " alias))
			(concat "s" pos "_" (fnv_hash (string col)))))))

(define prejoin_source_table_key (lambda (src)
	(list (source_schema src) (source_relation src))))

(define prejoin_sources_distinct? (lambda (sources)
	(equal? (count sources)
		(count (reduce sources (lambda (keys src)
			(append_unique keys (serialize (prejoin_source_table_key src)))) '())))))

(define prejoin_sources_supported? (lambda (sources)
	(and (not (empty_list? sources))
		(and (prejoin_sources_distinct? sources)
			(reduce sources (lambda (supported src)
				(and supported
					(and (source_is_base_table? src)
						(and (not (source_outer? src))
							(not (empty_list? (prejoin_primary_key_columns src)))))))
				true)))))

(define physical_prejoin_supported? (lambda (block)
	(and (query_block? block)
		(and (empty_list? (qb_stages block))
			(and (> (count (qb_sources block)) 1)
				(and (prejoin_sources_supported? (qb_sources block))
					(and (not (equal? (prejoin_join_condition block) true))
						(not (expr_contains_session_dependency? (prejoin_join_condition block))))))))))

(define prejoin_query_exprs (lambda (block fields)
	(merge (list
		(extract_assoc fields (lambda (_title expr) expr))
		(list (coalesceNil (qb_where block) true))
		(qb_group block)
		(if (nil? (qb_having block)) '() (list (qb_having block)))
		(order_exprs (qb_order block))
		(source_join_exprs (qb_sources block))
		(extract_assoc (qb_hidden block) (lambda (_title expr) expr))))))

(define prejoin_columns_by_alias (lambda (block fields)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(define referenced (reduce (prejoin_query_exprs block fields) (lambda (columns expr)
			(collect_join_columns_acc sources default_alias nil expr columns)) '()))
		(reduce sources (lambda (columns src)
			(qassoc_set columns (source_alias src)
				(merge_unique (list
					(qassoc_get columns (source_alias src) '())
					(prejoin_primary_key_columns src)))))
			referenced))))

(define prejoin_source_for_expr (lambda (sources default_alias tblvar tbl_ignorecase col col_ignorecase)
	(if (nil? tblvar)
		(source_for_unqualified_column sources default_alias col col_ignorecase)
		(source_for_alias sources default_alias tblvar tbl_ignorecase))))

(define prejoin_rewrite_expr (lambda (sources default_alias prejoin_alias expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase)
		(begin
			(define src (prejoin_source_for_expr sources default_alias tblvar tbl_ignorecase col col_ignorecase))
			(if (nil? src)
				expr
				(list (quote get_column) prejoin_alias false
					(prejoin_column_name sources (source_alias src) (resolve_physical_column_name src col col_ignorecase)) false)))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(prejoin_rewrite_expr sources default_alias prejoin_alias
			(list (symbol "get_column") tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons head (map tail (lambda (item)
			(prejoin_rewrite_expr sources default_alias prejoin_alias item))))
		_ expr)))

(define prejoin_rewrite_fields (lambda (sources default_alias prejoin_alias fields)
	(map_assoc fields (lambda (_title expr)
		(prejoin_rewrite_expr sources default_alias prejoin_alias expr)))))

(define prejoin_where_join_term? (lambda (block term)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(> (count (join_hypergraph_expr_aliases default_alias (source_aliases sources) term)) 1))))

/* Inner-join predicates remain in the logical query-block through reordering.
The physical prejoin extracts only multi-source conjuncts; local WHERE filters
remain query-specific and are evaluated over the cached intermediate relation. */
(define prejoin_where_join_terms (lambda (block)
	(filter (split_and_terms (coalesceNil (qb_where block) true)) (lambda (term)
		(prejoin_where_join_term? block term)))))

(define prejoin_join_condition (lambda (block)
	(combine_where_terms (merge (list
		(source_join_exprs (qb_sources block))
		(prejoin_where_join_terms block))) true)))

(define prejoin_residual_condition (lambda (block)
	(combine_where_terms
		(filter (split_and_terms (coalesceNil (qb_where block) true)) (lambda (term)
			(not (prejoin_where_join_term? block term))))
		true)))

(define prejoin_table_name (lambda (block default_alias)
	(begin
		(define sources (qb_sources block))
		(define canonical_alias "__prejoin")
		(define signature (list
			"physical-prejoin-v2"
			(map sources prejoin_source_table_key)
			(prejoin_rewrite_expr sources default_alias canonical_alias (prejoin_join_condition block))))
		(concat ".prejoin:" (fnv_hash (serialize signature))))))

(define prejoin_primary_key_exprs (lambda (sources)
	(merge (map sources (lambda (src)
		(map (prejoin_primary_key_columns src) (lambda (col)
			(list (quote get_column) (source_alias src) false col false))))))))

(define prejoin_primary_key_names (lambda (sources)
	(merge (map sources (lambda (src)
		(map (prejoin_primary_key_columns src) (lambda (col)
			(prejoin_column_name sources (source_alias src) col))))))))

(define prejoin_create_columns (lambda (sources)
	(begin
		(define key_names (prejoin_primary_key_names sources))
		(cons (quote list)
			(cons (list (quote list) "unique" "rows" (cons (quote list) key_names))
				(map key_names (lambda (col)
					(list (quote list) "column" col "any" (quoted_runtime_list '()) (quoted_runtime_list '())))))))))

(define prejoin_sources_without_join_conditions (lambda (sources)
	(map sources (lambda (src) (source_with_join_expr src true)))))

(define prejoin_insert_expr (lambda (schema table_name column_names values)
	(list (quote insert)
		(list (quote table) schema table_name)
		(cons (quote list) column_names)
		(list (quote list) (cons (quote list) values))
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		true)))

(define prejoin_initial_fill_plan (lambda (block table_name)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(define key_exprs (prejoin_primary_key_exprs sources))
		(define condition (prejoin_join_condition block))
		(define values (map key_exprs (lambda (expr)
			(lower_column_expr_for_join sources default_alias expr))))
		(build_join_scan_sink
			(qb_schema block)
			(prejoin_sources_without_join_conditions sources)
			default_alias
			(merge (list key_exprs (list condition)))
			condition
			(prejoin_insert_expr (qb_schema block) table_name (prejoin_primary_key_names sources) values)
			'()))))

(define prejoin_replace_trigger_expr (lambda (sources default_alias trigger_alias dict_symbol expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase)
		(begin
			(define src (prejoin_source_for_expr sources default_alias tblvar tbl_ignorecase col col_ignorecase))
			(if (nil? src)
				expr
				(begin
					(define physical_col (resolve_physical_column_name src col col_ignorecase))
					(if (equal?? (source_alias src) trigger_alias)
						(list (quote get_assoc) dict_symbol physical_col)
						(list (quote get_column) (source_alias src) false physical_col false)))))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(prejoin_replace_trigger_expr sources default_alias trigger_alias dict_symbol
			(list (symbol "get_column") tblvar tbl_ignorecase col col_ignorecase))
		(cons head tail) (cons head (map tail (lambda (item)
			(prejoin_replace_trigger_expr sources default_alias trigger_alias dict_symbol item))))
		_ expr)))

(define prejoin_trigger_insert_plan (lambda (block table_name trigger_src dict_symbol)
	(begin
		(define sources (qb_sources block))
		(define trigger_alias (source_alias trigger_src))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(define remaining (prejoin_sources_without_join_conditions
			(filter sources (lambda (src) (not (equal?? (source_alias src) trigger_alias))))))
		(define remaining_default (source_alias (car remaining)))
		(define condition (prejoin_replace_trigger_expr sources default_alias trigger_alias dict_symbol
			(prejoin_join_condition block)))
		(define key_exprs (map (prejoin_primary_key_exprs sources) (lambda (expr)
			(prejoin_replace_trigger_expr sources default_alias trigger_alias dict_symbol expr))))
		(define values (map key_exprs (lambda (expr)
			(lower_column_expr_for_join remaining remaining_default expr))))
		(build_join_scan_sink
			(qb_schema block)
			remaining
			remaining_default
			(merge (list key_exprs (list condition)))
			condition
			(prejoin_insert_expr (qb_schema block) table_name (prejoin_primary_key_names sources) values)
			'()))))

(define prejoin_trigger_delete_plan (lambda (block table_name trigger_src dict_symbol)
	(begin
		(define sources (qb_sources block))
		(define alias (source_alias trigger_src))
		(define key_cols (prejoin_primary_key_columns trigger_src))
		(define cache_cols (map key_cols (lambda (col) (prejoin_column_name sources alias col))))
		(define params (map cache_cols symbol))
		(define terms (map (produceN (count key_cols)) (lambda (i)
			(list (quote equal?) (nth params i) (list (quote get_assoc) dict_symbol (nth key_cols i))))))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) (qb_schema block) table_name)
			(cons (quote list) cache_cols)
			(list (quote lambda) params (combine_where_terms terms true))
			(quoted_runtime_list (list "$update"))
			(list (quote lambda) (list (symbol "$update")) (list (symbol "$update")))
			(quote +) 0 nil false))))

(define prejoin_deferred_trigger (lambda (body)
	(list (quote quote)
		(list (quote deferred_trigger)
			(list (quote lambda) (list (quote OLD) (quote NEW)) body)))))

(define prejoin_create_trigger_plan (lambda (src name timing body)
	(list (quote createtrigger)
		(list (quote table) (source_schema src) (source_relation src))
		name timing "" (prejoin_deferred_trigger body) false)))

(define prejoin_trigger_registration_plans (lambda (block table_name)
	(merge (map (qb_sources block) (lambda (src)
		(begin
			(define prefix (concat ".prejoin:" table_name "|" (source_relation src) "|"))
			(define delete_body (prejoin_trigger_delete_plan block table_name src (quote OLD)))
			(define insert_body (prejoin_trigger_insert_plan block table_name src (quote NEW)))
			(list
				(prejoin_create_trigger_plan src (concat prefix "after_delete") "after_delete" delete_body)
				(prejoin_create_trigger_plan src (concat prefix "after_insert") "after_insert" insert_body)
				(prejoin_create_trigger_plan src (concat prefix "after_update") "after_update"
					(list (quote !begin) delete_body insert_body))
				(prejoin_create_trigger_plan src (concat prefix "after_drop_table") "after_drop_table"
					(list (quote droptable) (qb_schema block) table_name true))
				(prejoin_create_trigger_plan src (concat prefix "after_drop_column") "after_drop_column"
					(list (quote droptable) (qb_schema block) table_name true)))))))))

(define prejoin_lookup_computed_column_plan (lambda (block table_name sources src col)
	(begin
		(define alias (source_alias src))
		(define key_cols (prejoin_primary_key_columns src))
		(define input_cols (map key_cols (lambda (key_col)
			(prejoin_column_name sources alias key_col))))
		(define filter_params (map key_cols (lambda (key_col) (symbol (concat alias "." key_col)))))
		(define filter_terms (map (produceN (count key_cols)) (lambda (i)
			(list (quote equal?) (nth filter_params i) (list (quote outer) (symbol (nth input_cols i)))))))
		(define value_symbol (symbol (concat alias "." col)))
		(define reduce_expr (list (quote lambda) (list (quote _old) (quote new)) (quote new)))
		(define computor (list (quote lambda)
			(map input_cols symbol)
			(list (quote scan)
				'(session "__memcp_tx")
				(source_table_expr src)
				(cons (quote list) key_cols)
				(list (quote lambda) filter_params (combine_where_terms filter_terms true))
				(quoted_runtime_list (list col))
				(list (quote lambda) (list value_symbol) value_symbol)
				reduce_expr nil reduce_expr false)))
		(list (quote createcolumn)
			(list (quote table) (qb_schema block) table_name)
			(prejoin_column_name sources alias col)
			"any"
			(quoted_runtime_list '())
			(quoted_runtime_list '("temp" true))
			(cons (quote list) input_cols)
			computor))))

(define prejoin_computed_column_plans (lambda (block fields table_name)
	(begin
		(define sources (qb_sources block))
		(define columns_by_alias (prejoin_columns_by_alias block fields))
		(merge (map sources (lambda (src)
			(begin
				(define key_cols (prejoin_primary_key_columns src))
				(define requested (qassoc_get columns_by_alias (source_alias src) '()))
				(map (filter requested (lambda (col) (not (contains? key_cols col)))) (lambda (col)
					(prejoin_lookup_computed_column_plan block table_name sources src col))))))))))

(define prejoin_rewritten_block (lambda (block fields table_name)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(define alias table_name)
		(make_query_block
			(qb_schema block)
			(list (list alias (qb_schema block) table_name false true))
			(prejoin_rewrite_fields sources default_alias alias fields)
			(prejoin_rewrite_expr sources default_alias alias (prejoin_residual_condition block))
			(map (qb_group block) (lambda (expr) (prejoin_rewrite_expr sources default_alias alias expr)))
			(if (nil? (qb_having block)) nil (prejoin_rewrite_expr sources default_alias alias (qb_having block)))
			(map (qb_order block) (lambda (item) (match item
				'(expr dir) (list (prejoin_rewrite_expr sources default_alias alias expr) dir)
				_ (neumann_fail "build_queryplan" "malformed prejoin ORDER BY item"))))
			(qb_limit block)
			(qb_offset block)
			(prejoin_rewrite_fields sources default_alias alias (qb_hidden block))
			'()
			(qassoc_set (qb_facts block) (quote default_alias) alias)))))

(define physical_prejoin_plan (lambda (block)
	(begin
		(if (not (physical_prejoin_supported? block))
			(neumann_fail "build_queryplan" "physical prejoin requires distinct inner base tables with primary keys")
			true)
		(define sources (qb_sources block))
		(define fields (expand_query_block_fields sources (qb_fields block)))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
		(define table_name (prejoin_table_name block default_alias))
		(define table_expr (list (quote table) (qb_schema block) table_name))
		(define prepare_plan (list (quote !begin)
			(list (quote createtable) (qb_schema block) table_name
				(prejoin_create_columns sources) (quoted_runtime_list '("engine" "cache")) true)
			(list (quote initialize_cache_table)
				table_expr
				(cons (quote list) (map sources source_table_expr))
				(list (quote lambda) '() (cons (quote !begin) (prejoin_trigger_registration_plans block table_name)))
				(list (quote lambda) '() (prejoin_initial_fill_plan block table_name)))
			(cons (quote !begin) (prejoin_computed_column_plans block fields table_name))
			(list (quote touch_keytable) table_expr)))
		(list prepare_plan (prejoin_rewritten_block block fields table_name)))))

(define lower_query_block_through_prejoin (lambda (block)
	(begin
		(define prejoin (physical_prejoin_plan block))
		(list (quote !begin)
			(nth prejoin 0)
			(lower_single_source_query_block (nth prejoin 1))))))

(define build_join_scan_rows (lambda (schema sources plan default_alias needed_exprs final_condition fields order_items offset_value limit_value stages)
	(begin
		(define row_expr (list (quote resultrow)
			(cons (quote list) (lower_join_result_fields sources default_alias fields))))
		(build_join_scan_pipeline_using_recipe
			schema sources plan default_alias needed_exprs final_condition row_expr
			order_items offset_value limit_value true nil stages))))

(define lower_query_block_as_dataset_reduce (lambda (block fields row_mapper reduce_expr neutral_expr shard_reduce_expr)
	(begin
		(if (empty_list? (qb_sources block))
			(neumann_fail "build_queryplan" "dataset reducer requires a FROM source")
			true)
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(neumann_fail "build_queryplan" "dataset reducer cannot consume grouped input")
			true)
		(define sources (qb_sources block))
		(define first_alias (source_alias (car sources)))
		(define scan_sources sources)
		(define scan_plan (query_block_join_plan block scan_sources))
		(define driver_source (join_optimizer_source_by_alias scan_sources
			(join_optimizer_tree_first_alias scan_plan)))
		(define final_condition (coalesceNil (qb_where block) true))
		(define order_items (coalesceNil (qb_order block) '()))
		(define direct_order (order_items_supported_by_join_driver?
			scan_sources first_alias driver_source order_items
			(query_block_stage_catalog block) final_condition))
		(define direct_order_safe (and direct_order
			(not (ordered_join_limit_requires_complete_rows?
				scan_sources first_alias final_condition (qb_offset block) (qb_limit block)))))
		(if (not direct_order_safe)
			(neumann_fail "build_queryplan" "dataset ORDER BY requires a storage carrier")
			true)
		(define field_exprs (extract_assoc fields (lambda (_title expr) expr)))
		(define needed_exprs (merge (list
			field_exprs
			(list final_condition)
			(order_exprs order_items)
			(source_join_exprs scan_sources))))
		(define row_expr (cons row_mapper (map field_exprs (lambda (expr)
			(lower_column_expr_for_join scan_sources first_alias expr)))))
		(build_join_scan_reduce_using_recipe
			(qb_schema block)
			scan_sources
			scan_plan
			first_alias
			needed_exprs
			final_condition
			row_expr
			order_items
			(coalesceNil (qb_offset block) 0)
			(coalesceNil (qb_limit block) -1)
			true nil (query_block_stage_catalog block)
			reduce_expr neutral_expr shard_reduce_expr))))

(define scalar_order_lookup_input_keys (lambda (stage)
	(begin
		(define repr (qassoc_get (gs_facts stage) (quote btw2025_repr) '()))
		(if (equal? (count repr) (count (gs_keys stage)))
			(map repr (lambda (pair)
				(match pair
					'(_outer local) local
					_ nil)))
			'()))))

(define scalar_order_lookup_stage? (lambda (stage requested_col)
	(if (or (nil? requested_col)
		(or (not (group_stage? stage))
			(not (equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote scalar_single)))))
		false
		(begin
			(define ag (scalar_first_probe_aggregate stage requested_col))
			(define parts (if (nil? ag) nil (scalar_first_probe_parts ag)))
			(define input_rows (planner_stage_input_rows (gs_input stage)))
			(define local_keys (scalar_order_lookup_input_keys stage))
			(define input_keys (if (not (source_is_base_table? (gs_input stage)))
				false
				(reduce local_keys (lambda (local key)
					(and local (equal? (expr_source_alias key) (source_alias (gs_input stage))))) true)))
			(and (not (nil? input_rows))
				(and (>= input_rows 1000)
					(and (source_is_base_table? (gs_input stage))
						(and input_keys
							(and (equal? (count local_keys) 1)
								(and (not (nil? parts))
									(and (empty_list? (nth parts 1)) (equal? (nth parts 3) 0))))))))))))

(define scalar_order_lookup_marker (lambda (expr found)
	(if (not (nil? found))
		found
		(match expr
			((symbol scalar_first_probe) stage requested_col)
			(if (scalar_order_lookup_stage? stage requested_col) (list nil stage requested_col) nil)
			((quote scalar_first_probe) stage requested_col)
			(if (scalar_order_lookup_stage? stage requested_col) (list nil stage requested_col) nil)
			((symbol scalar_first_probe) stage requested_col _stages)
			(if (scalar_order_lookup_stage? stage requested_col) (list nil stage requested_col) nil)
			((quote scalar_first_probe) stage requested_col _stages)
			(if (scalar_order_lookup_stage? stage requested_col) (list nil stage requested_col) nil)
			(cons _head tail) (reduce tail (lambda (nested item)
				(scalar_order_lookup_marker item nested)) nil)
			_ nil))))

(define scalar_order_lookup_source (lambda (stages sources default_alias order_items)
	(begin
		(define order_columns (join_column_recipe sources default_alias (order_exprs order_items)))
		(define source_lookup (reduce sources (lambda (found src)
			(if (not (nil? found))
				found
				(begin
					(define stage (if (stage_output_relation? (source_relation src))
						(stage_for_output_relation stages (source_relation src))
						(stage_for_group_cache_source stages src)))
					(define requested_cols (qassoc_get order_columns (source_alias src) '()))
					(define requested_col (if (empty_list? requested_cols) nil (car requested_cols)))
					(if (and (scalar_order_lookup_stage? stage requested_col)
						(stage_lookup_keys_resolve_in_sources? stage sources default_alias))
						(list src stage requested_col)
						nil))))
			nil))
		(if (not (nil? source_lookup))
			source_lookup
			(scalar_order_lookup_marker (order_exprs order_items) nil)))))

(define lower_multi_source_query_block (lambda (block)
	(begin
		(define fields (expand_query_block_fields (qb_sources block) (qb_fields block)))
		(if (or (not (empty_list? (qb_group block))) (or (not (nil? (qb_having block))) (query_block_has_aggregates? block)))
			(if (physical_prejoin_supported? block)
				(lower_query_block_through_prejoin block)
				(lower_group_stage (make_group_stage_for_query_block block)))
			(begin
				(define sources (qb_sources block))
				(define first_alias (qassoc_get (qb_facts block) (quote default_alias) (source_alias (car sources))))
				(define final_condition (coalesceNil (qb_where block) true))
				(define stage_catalog (query_block_stage_catalog block))
				(define scan_sources sources)
				(define scan_plan (query_block_join_plan block scan_sources))
				(define driver_alias (join_optimizer_tree_first_alias scan_plan))
				(define driver_source (join_optimizer_source_by_alias scan_sources driver_alias))
				(define ordered_sources (join_optimizer_sources_for_order scan_sources
					(join_optimizer_tree_aliases scan_plan)))
				(define order_items (coalesceNil (qb_order block) '()))
				(define direct_order
					(order_items_supported_by_join_driver?
						scan_sources first_alias driver_source order_items stage_catalog final_condition))
				(define direct_order_safe (and direct_order
					(not (ordered_join_limit_requires_complete_rows? scan_sources first_alias final_condition (qb_offset block) (qb_limit block)))))
				(define hierarchical_order
					(order_items_follow_join_tree? ordered_sources first_alias order_items stage_catalog final_condition))
				(define needed_exprs (merge (list
					(extract_assoc fields (lambda (_title expr) expr))
					(list final_condition)
					(order_exprs order_items)
					(source_join_exprs scan_sources))))
				(if (or direct_order_safe
					(and hierarchical_order (not (query_limit_active? (qb_offset block) (qb_limit block)))))
					(build_join_scan_rows
						(qb_schema block) scan_sources scan_plan first_alias needed_exprs
						final_condition fields order_items (qb_offset block) (qb_limit block) stage_catalog)
					(if hierarchical_order
						(join_ordered_stream_plan
							(qb_schema block) scan_sources scan_plan first_alias needed_exprs
							final_condition fields order_items (qb_offset block) (qb_limit block) stage_catalog)
						(if (physical_prejoin_supported? block)
							(lower_query_block_through_prejoin block)
							(lower_join_order_through_intermediate
								block fields scan_sources scan_plan first_alias needed_exprs
								final_condition order_items stage_catalog)))
))))))

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
		(define scan_plan (query_block_join_plan block sources))
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
		(build_join_scan_reduce_using_recipe
			(qb_schema block)
			sources
			scan_plan
			first_alias
			needed_exprs
			cond
			row_expr
			'()
			0
			-1
			true nil (qb_stages block)
			(quote +) 0 (quote +)))))

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
		(begin
			(define stage_catalog (query_block_stage_catalog block))
			(define stage_lookup (query_block_stage_lookup block))
			(define dependency_graph (stage_dependency_graph stage_lookup))
			(cons (quote begin)
				(merge (list
					(lower_unique_stage_prepares_with_graph dependency_graph stage_lookup (qb_stages block))
					(list (lower_dml_query_block_core (query_block_without_stages_after_prepare_using stage_lookup block) target_schema target_tbl)))))))))

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
					(list (list (lower_group_stage_prepare_using (list stage) (list stage) stage)) final_block nil)
					nil))))))

(define union_semijoin_equal_parts (lambda (driver lookup expr)
	(match expr
		((symbol equal??) left right) (begin
			(define lookup_left (direct_column_name_for_alias lookup left))
			(define driver_right (direct_column_name_for_alias driver right))
			(if (and (not (nil? lookup_left)) (not (nil? driver_right)))
				(list lookup_left driver_right)
				(begin
					(define lookup_right (direct_column_name_for_alias lookup right))
					(define driver_left (direct_column_name_for_alias driver left))
					(if (and (not (nil? lookup_right)) (not (nil? driver_left)))
						(list lookup_right driver_left)
						nil))))
		((quote equal??) left right) (union_semijoin_equal_parts driver lookup
			(list (symbol "equal??") left right))
		_ nil)))

(define union_semijoin_join_entry (lambda (driver lookup branch)
	(reduce (merge (list
		(split_and_terms (coalesceNil (source_join_expr lookup) true))
		(split_and_terms (coalesceNil (qb_where branch) true)))) (lambda (found term)
			(if (not (nil? found))
				found
				(begin
					(define parts (union_semijoin_equal_parts driver lookup term))
					(if (nil? parts) nil (list parts term)))))
		nil)))

(define union_semijoin_branch_stream_plan_ordered (lambda (branch)
	(begin
		(define sources (qb_sources branch))
		(if (not (equal? (count sources) 2))
			nil
			(begin
				(define driver (car sources))
				(define lookup (cadr sources))
				(define default_alias (qassoc_get (qb_facts branch) (quote default_alias) (source_alias driver)))
				(define join_entry (union_semijoin_join_entry driver lookup branch))
				(define join_parts (if (nil? join_entry) nil (nth join_entry 0)))
				(define join_predicate (if (nil? join_entry) nil (nth join_entry 1)))
				(define remaining_where (combine_where_terms
					(filter (split_and_terms (coalesceNil (qb_where branch) true)) (lambda (term)
						(not (equal? term join_predicate))))
					true))
				(define visible_exprs (merge (list
					(extract_assoc (qb_fields branch) (lambda (_title expr) expr))
					(list remaining_where)
					(extract_assoc (qb_hidden branch) (lambda (_title expr) expr)))))
				(define lookup_unused (not (reduce visible_exprs (lambda (used expr)
					(or used (expr_refs_sources? default_alias (list lookup) expr))) false)))
				(define lookup_keys (prejoin_primary_key_columns lookup))
				(if (not (and
					lookup_unused
					(not (source_outer? driver))
					(not (source_outer? lookup))
					(equal? (coalesceNil (source_join_expr driver) true) true)
					(not (nil? join_parts))
					(equal? (count lookup_keys) 1)
					(equal? (car lookup_keys) (car join_parts))))
					nil
					(begin
						(define rewritten (make_query_block
							(qb_schema branch)
							(list (source_with_join_expr driver nil))
							(qb_fields branch)
							remaining_where
							(qb_group branch)
							(qb_having branch)
							(qb_order branch)
							(qb_limit branch)
							(qb_offset branch)
							(qb_hidden branch)
							(qb_stages branch)
							(qb_facts branch)))
						(define lookup_recset (list (quote scan_recset)
							'(session "__memcp_tx")
							(source_table_expr lookup)
							(quoted_runtime_list '())
							(list (quote lambda) '() true)))
						(define driver_recset (list (quote recset_project_join)
							'(session "__memcp_tx")
							lookup_recset
							(quoted_runtime_list (list (car join_parts)))
							(source_table_expr driver)
							(quoted_runtime_list (list (cadr join_parts)))))
						(if (union_ordered_branch_supported? rewritten)
							(list '() rewritten driver_recset)
							nil))))))))

(define union_semijoin_branch_with_sources (lambda (branch sources)
	(make_query_block
		(qb_schema branch)
		sources
		(qb_fields branch)
		(qb_where branch)
		(qb_group branch)
		(qb_having branch)
		(qb_order branch)
		(qb_limit branch)
		(qb_offset branch)
		(qb_hidden branch)
		(qb_stages branch)
		(qb_facts branch))))

(define union_semijoin_branch_stream_plan (lambda (branch)
	(begin
		(define sources (qb_sources branch))
		(if (not (equal? (count sources) 2))
			nil
			(begin
				(define ordered (union_semijoin_branch_stream_plan_ordered branch))
				(if (not (nil? ordered))
					ordered
					(union_semijoin_branch_stream_plan_ordered
						(union_semijoin_branch_with_sources branch
							(list (cadr sources) (car sources))))))))))

(define prejoined_union_branch_stream_plan (lambda (branch)
	(begin
		(define prejoin (physical_prejoin_plan branch))
		(define prepare (nth prejoin 0))
		(define rewritten (nth prejoin 1))
		(define stream (if (union_ordered_branch_supported? rewritten)
			(list '() rewritten nil)
			(if (grouped_union_branch? rewritten)
				(grouped_union_branch_stream_plan rewritten)
				nil)))
		(if (nil? stream)
			nil
			(list (cons prepare (nth stream 0)) (nth stream 1) (nth stream 2))))))

(define row_number_union_branch_stream_plan (lambda (branch)
	(begin
		(define cataloged (query_block_with_full_stage_catalog branch))
		(define stages (qb_stages cataloged))
		(if (not (and (equal? (count stages) 1) (row_number_stage? (car stages))))
			nil
			(begin
				(define stage_lookup (query_block_stage_lookup cataloged))
				(define dependency_graph (stage_dependency_graph stage_lookup))
				(define prepared (query_block_without_stages_after_prepare_using stage_lookup cataloged))
				(if (union_ordered_branch_supported? prepared)
					(list
						(merge (list
							(lower_unique_stage_prepares_with_graph dependency_graph stage_lookup stages)
							(lower_stage_materialize_all stages)))
						prepared
						nil)
					nil))))))

(define prepared_union_branch_stream_plan (lambda (branch)
	(begin
		(define prepared (prepare_simple_query_block_physical_core
			(query_block_with_full_stage_catalog branch)))
		(define core_block (nth prepared 1))
		(if (union_ordered_branch_supported? core_block)
			(list (nth prepared 0) core_block nil)
			nil))))

(define union_ordered_branch_stream_plan (lambda (branch)
	(if (not (query_block? branch))
		nil
		(if (union_ordered_branch_supported? branch)
			(prepared_union_branch_stream_plan branch)
			(begin
				(define semijoin (union_semijoin_branch_stream_plan branch))
				(if (not (nil? semijoin))
					semijoin
					(if (physical_prejoin_supported? branch)
						(prejoined_union_branch_stream_plan branch)
						(if (grouped_union_branch? branch)
							(grouped_union_branch_stream_plan branch)
							(begin
								(define row_number (row_number_union_branch_stream_plan branch))
								(if (not (nil? row_number))
									row_number
									(prepared_union_branch_stream_plan branch)))))))))))

(define union_sort_column_for_alias (lambda (src expr)
	(scan_order_sort_column_for_alias src expr)))

(define union_ordered_scan_spec (lambda (branch titles order_positions table_override)
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
			(coalesceNil table_override (source_table_expr src))
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
				(define specs (map plans (lambda (plan)
					(union_ordered_scan_spec (nth plan 1) titles order_positions (nth plan 2)))))
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
					(lower_union_all_ordered block titles width)
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

/* Physical preparation expands and attaches the canonical stage catalog once,
without emitting runtime operators. Keeping this boundary explicit makes
analysis and emission independently measurable while preserving the normal
build_queryplan contract. */
(define prepare_physical_queryplan (lambda (ir)
	(begin
		(require_unnested_node "build_queryplan input" (ir_root ir))
		(define planned_root (apply_join_optimizer_plan_node (ir_root ir)))
		(make_ir
			(ir_kind ir)
			(if (query_block? planned_root)
				(query_block_with_full_stage_catalog planned_root)
				planned_root)
			(map (ir_stages ir) apply_join_optimizer_plan_stage)
			(ir_context_of ir)
			(ir_return ir)))))

(define physical_relational_list_collector? (lambda (expr)
	(match expr
		((symbol sort) _input _compare) true
		((quote sort) _input _compare) true
		((symbol slice) _input _start _end) true
		((quote slice) _input _start _end) true
		((symbol merge) ((symbol map) _input _mapper)) true
		((quote merge) ((quote map) _input _mapper)) true
		(cons head tail) (or
			(physical_relational_list_collector? head)
			(reduce tail (lambda (found item)
				(or found (physical_relational_list_collector? item))) false))
		_ false)))

(define require_physical_scan_relations (lambda (plan)
	(if (physical_relational_list_collector? plan)
		(neumann_fail "build_queryplan" "relational results must remain in storage scans")
		plan)))

(define emit_physical_queryplan (lambda (ir)
	(begin
		(define plan (match (ir_return ir)
			(symbol rows) (match (logical_op (ir_root ir))
				(symbol query-block) (lower_query_block_with_cataloged_stages (ir_root ir))
				(symbol union-block) (lower_union_block (ir_root ir))
				_ (neumann_fail "build_queryplan" "unknown logical root"))
			((symbol dml) target_schema target_tbl) (match (logical_op (ir_root ir))
				(symbol query-block) (lower_dml_query_block_with_stages (ir_root ir) target_schema target_tbl)
				(symbol union-block) (lower_dml_union_block_with_stages (ir_root ir) target_schema target_tbl)
				_ (neumann_fail "build_queryplan" "DML lowering expects a query-block root"))
			_ (neumann_fail "build_queryplan" "DML lowering is intentionally not scaffolded yet")))
		(require_physical_scan_relations plan))))

(define build_queryplan (lambda (ir)
	(emit_physical_queryplan (prepare_physical_queryplan ir))))

(define neumann_compile_pipeline (lambda (ast)
	(build_queryplan
		(join_reorder
			(untangle_query_term (sanitize_temporal_outputs (sanitize_decimal_aggregate_outputs ast)) nil)))))

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

(define explain_queryplan_compile (lambda (query parse_started_ns sql_bytes)
	(begin
		(define logical_count (lambda (node target)
			(begin
				(define own (if (equal? (logical_op node) target) 1 0))
				(if (query_block? node)
					(+ own (reduce (qb_stages node) (lambda (total stage)
						(+ total (logical_count stage target))) 0))
					(if (group_stage? node)
						(+ own (logical_count (gs_input node) target))
						(if (union_block? node)
							(+ own (reduce (union_branches node) (lambda (total branch)
								(+ total (logical_count branch target))) 0))
							own))))))
		(define plan_count (lambda (node target)
			(match node
				(cons head tail) (+
					(if (equal? head target) 1 0)
					(plan_count head target)
					(reduce tail (lambda (total item)
						(+ total (plan_count item target))) 0))
				_ 0)))
		(define tree_count (lambda (node)
			(match node
				(cons head tail) (+ 1
					(tree_count head)
					(reduce tail (lambda (total item)
						(+ total (tree_count item))) 0))
				_ 1)))
		(define started_ns (nanotime))
		(define ir (untangle_query_term query nil))
		(define untangled_ns (nanotime))
		(define reordered (join_reorder ir))
		(define reordered_ns (nanotime))
		(define prepared (prepare_physical_queryplan reordered))
		(define prepared_ns (nanotime))
		(define plan (emit_physical_queryplan prepared))
		(define emitted_ns (nanotime))
		(define plan_text (pretty_print plan (settings "ExplainWidth")))
		(define serialized_ns (nanotime))
		(list (quote resultrow)
			(list (quote list)
				"parse_ns" (- started_ns parse_started_ns)
				"untangle_ns" (- untangled_ns started_ns)
				"reorder_ns" (- reordered_ns untangled_ns)
				"physical_prepare_ns" (- prepared_ns reordered_ns)
				"recipe_emit_ns" (- emitted_ns prepared_ns)
				"lower_ns" (- emitted_ns reordered_ns)
				"serialize_ns" (- serialized_ns emitted_ns)
				"planner_total_ns" (- emitted_ns started_ns)
				"compile_total_ns" (- emitted_ns parse_started_ns)
				"measured_total_ns" (- serialized_ns parse_started_ns)
				"sql_bytes" sql_bytes
				"ast_nodes" (tree_count query)
				"logical_nodes" (tree_count reordered)
				"plan_nodes" (tree_count plan)
				"plan_bytes" (strlen plan_text)
				"query_blocks" (logical_count (ir_root reordered) (quote query-block))
				"group_stages" (logical_count (ir_root reordered) (quote group-stage))
				"union_blocks" (logical_count (ir_root reordered) (quote union-block))
				"scans" (plan_count plan (quote scan))
				"ordered_scans" (+
					(plan_count plan (quote scan_order))
					(plan_count plan (quote scan_order_multi)))
				"exists_scans" (plan_count plan (quote scan_exists))
				"group_caches" (plan_count plan (quote touch_keytable))
				/* Backward-compatible telemetry alias; use group_caches in new integrations. */
				"group_carriers" (plan_count plan (quote touch_keytable)))))))
