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

parser AST -> normalize_sql_syntax -> decorrelate_logical_query
-> optimize_logical_query -> build_queryplan

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
	(if (equal?? schema "information_schema")
		/* sql-metadata.scm owns only the virtual catalog. This is the sole
		public dispatcher, so module load order cannot replace its semantics. */
		(information_schema_column_catalog schema tbl)
		(try
			(lambda ()
				(begin
					(define handle (table schema tbl))
					(if handle (show handle) (show schema tbl))))
			(lambda (_e) '())))))

(define qassoc_get (lambda (xs key default)
	(get_assoc_pairlist (coalesceNil xs '()) key default)))

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

/* Cost-model decisions register the condition which keeps their chosen branch
valid in a compile-local newsession. cached_parse installs the accumulator;
direct planner users deliberately remain unaffected. Structural string keys
deduplicate identical conditions without mutable Scheme lists. */
(define planner_record_guard_condition (lambda (condition)
	(begin
		(define condition_accumulator (try
			(lambda () ((context "session") "__memcp_queryplan_guard_conditions"))
			(lambda (_e) nil)))
		(if (nil? condition_accumulator)
			condition
			(begin
				(condition_accumulator (string condition) condition)
				(define covered_bindings (try
					(lambda () ((context "session") "__memcp_queryplan_guarded_session_keys"))
					(lambda (_e) nil)))
				(if (nil? covered_bindings)
					nil
					(reduce (query_expr_session_reads condition) (lambda (_ expr)
						(covered_bindings (string expr) true)) nil))
				condition)))))

(define planner_guarded_choice (lambda (chosen condition)
	(begin
		(planner_record_guard_condition
			(if chosen condition (list (quote not) condition)))
		chosen)))

/* EXPLAIN PHYSICAL installs this compile-local event sink. Physical lowering
records only decisions it actually makes; ordinary compilation has no sink and
therefore keeps the same plan and cache behavior. Numeric cost records retain
their calibrated nanosecond/memory components so rejected alternatives can be
compared without reverse-engineering the generated Scheme plan. */
(define planner_physical_explain_accumulator (lambda ()
	(try
		(lambda () ((context "session") "__memcp_explain_physical"))
		(lambda (_e) nil))))

(define planner_record_physical_decision (lambda (decision)
	(begin
		(define accumulator (planner_physical_explain_accumulator))
		(if (nil? accumulator)
			decision
			(begin
				(define decision_id (qassoc_get decision "decision_id" nil))
				(define seen_key (if (nil? decision_id) nil (concat "seen:" decision_id)))
				(if (and (not (nil? seen_key)) (accumulator seen_key))
					nil
					(begin
						(define count (coalesceNil (accumulator "count") 0))
						(accumulator (concat "decision:" count) decision)
						(accumulator "count" (+ count 1))
						(if (nil? seen_key) nil (accumulator seen_key true))))
				decision)))))

(define planner_physical_explain_decisions (lambda (accumulator)
	(map (produceN (coalesceNil (accumulator "count") 0)) (lambda (idx)
		(accumulator (concat "decision:" idx))))))

/* CALIBRATE compiles the same logical query repeatedly with one explicitly
forced physical alternative. Overrides are compile-local and keyed by the
stable logical stage identity, so nested decisions cannot accidentally consume
an override intended for a sibling edge. */
(define planner_physical_override (lambda (decision_id)
	(begin
		(define overrides (try
			(lambda () ((context "session") "__memcp_physical_overrides"))
			(lambda (_e) nil)))
		(if (nil? overrides) nil (qassoc_get overrides decision_id nil)))))

(define planner_physical_choice (lambda (decision_id normal_choice alternatives)
	(begin
		(define forced (planner_physical_override decision_id))
		(if (nil? forced)
			normal_choice
			(if (contains? alternatives forced)
				forced
				(error (concat "Unknown physical alternative `" forced "` for " decision_id)))))))

(define planner_cost_explain (lambda (cost)
	(list
		(list "startup_ns" (qassoc_get cost (quote startup_ns) 0))
		(list "row_ns" (qassoc_get cost (quote row_ns) 0))
		(list "probe_ns" (qassoc_get cost (quote probe_ns) 0))
		(list "batch_startup_ns" (qassoc_get cost (quote batch_startup_ns) 0))
		(list "batch_row_ns" (qassoc_get cost (quote batch_row_ns) 0))
		(list "build_ns" (qassoc_get cost (quote build_ns) 0))
		(list "memory_bytes" (qassoc_get cost (quote memory_bytes) 0))
		(list "compile_ns" (qassoc_get cost (quote compile_ns) 0))
		(list "expected_rows" (qassoc_get cost (quote expected_rows) nil))
		(list "confidence" (qassoc_get cost (quote confidence) nil))
		(list "execution_ns" (qassoc_get cost (quote execution_ns) 0))
		(list "total_ns" (qassoc_get cost (quote total_ns) 0)))))

/* Runtime statistic expressions recur throughout dynamic-programming costs.
Bind each unique expression once in the final guard instead of rereading the
catalog for every comparison. The binding catalog is compile-local as well. */
(define planner_guard_runtime_binding (lambda (expr)
	(begin
		(define bindings (try
			(lambda () ((context "session") "__memcp_queryplan_guard_bindings"))
			(lambda (_e) nil)))
		(if (nil? bindings)
			expr
			(begin
				(define key (string expr))
				(define existing (bindings key))
				(if (nil? existing)
					(begin
						(define binding (list
							(symbol (concat "__queryplan_guard_value_" (fnv_hash key)))
							expr))
						(bindings key binding)
						(car binding))
					(car existing)))))))

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

/* A scalar subquery's projection is a value demand on its relational scope,
not part of that scope's identity. Nested dependent stages use this identity as
their parent handle, so independently written projections over the same FROM /
WHERE / grouping / ordering backbone can share those dependencies before the
parent stages are merged into a multi-output stage. */
(define dependent_subquery_scope_backbone (lambda (kind subquery)
	(begin
		(define normalized (normalize_query_ast subquery))
		(if (and (equal? kind (quote scalar)) (query_block? normalized))
			(begin
				(define backbone (make_query_block
					(qb_schema normalized)
					(qb_sources normalized)
					'()
					(qb_where normalized)
					(qb_group normalized)
					(qb_having normalized)
					(qb_order normalized)
					(qb_limit normalized)
					(qb_offset normalized)
					(qb_hidden normalized)
					(qb_stages normalized)
					(qb_facts normalized)))
				(define alias_map (stage_semantic_alias_entries
					(source_aliases (qb_sources normalized)) "__subquery_local_"))
				(stage_semantic_canonical_node alias_map '() backbone))
			normalized))))

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
	(if (and (symbol? head) (contains? (list
		(quote inner_select)
		(quote inner_select_in)
		(quote inner_select_exists)
		(quote neumann_scalar)
		(quote neumann_exists)
		(quote neumann_in)
		(quote dependent-subquery)
		(quote exists_probe)) head))
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

/* Collect bound source aliases once. Join pruning must not rescan a wide
projection for every source; that turns read-model queries into O(N^2) planner
work before decorrelation has even started. */
(define query_expr_alias_set (lambda (default_alias expr aliases)
	(match expr
		((symbol get_column) tblvar _ _ _)
		(set_assoc aliases (resolve_column_alias tblvar default_alias) true)
		((quote get_column) tblvar _ _ _)
		(set_assoc aliases (resolve_column_alias tblvar default_alias) true)
		(cons head tail) (reduce tail (lambda (found item)
			(query_expr_alias_set default_alias item found))
			(query_expr_alias_set default_alias head aliases))
		_ aliases)))

(define query_exprs_alias_set (lambda (default_alias exprs)
	(reduce (coalesceNil exprs '()) (lambda (aliases expr)
		(query_expr_alias_set default_alias expr aliases)) '())))

(define source_primary_key_columns (lambda (src)
	(if (not (source_is_base_table? src))
		'()
		(map (filter (get_schema (source_schema src) (source_relation src)) (lambda (col)
			(equal?? (col "Key") "PRI"))) (lambda (col) (col "Field"))))))

(define source_unique_key_sets (lambda (src)
	(if (not (source_is_base_table? src))
		'()
		(try
			(lambda ()
				(begin
					(define info (show (source_schema src) (source_relation src) true))
					(map ((info "meta") "Unique") (lambda (unique_key)
						(unique_key "Cols")))))
			(lambda (_e) '())))))

(define unique_lookup_column_name (lambda (src expr)
	(match expr
		((symbol if) ((symbol nil?) probe) fallback value)
		(if (and (nil? fallback) (equal? probe value))
			(direct_column_name_for_alias src probe)
			nil)
		((quote if) ((quote nil?) probe) fallback value)
		(if (and (nil? fallback) (equal? probe value))
			(direct_column_name_for_alias src probe)
			nil)
		_ (direct_column_name_for_alias src expr))))

(define unique_left_join_key_term? (lambda (default_alias src key_col term)
	(match term
		'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
			(or
				(and (equal? (unique_lookup_column_name src left) key_col)
					(not (expr_refs_alias? default_alias (source_alias src) right)))
				(and (equal? (unique_lookup_column_name src right) key_col)
					(not (expr_refs_alias? default_alias (source_alias src) left))))
			false)
		_ false)))

(define unused_unique_left_join? (lambda (default_alias referenced_aliases src)
	(begin
		(define primary_key (source_primary_key_columns src))
		(define unique_keys (merge_unique (list
			(source_unique_key_sets src)
			(if (empty_list? primary_key) '() (list primary_key)))))
		(define terms (split_and_terms (coalesceNil (source_join_expr src) true)))
		(and (source_outer? src)
			(and (source_is_base_table? src)
				(and (not (has_assoc? referenced_aliases (source_alias src)))
					(reduce unique_keys (lambda (found key_cols)
						(or found
							(and (not (empty_list? key_cols))
								(reduce key_cols (lambda (bound key_col)
									(and bound (reduce terms (lambda (matched term)
										(or matched (unique_left_join_key_term?
											default_alias src key_col term))) false)))
									true))))
						false)))))))

(define prune_unused_unique_left_joins_reversed (lambda (reversed_sources default_alias referenced_aliases)
	(match (coalesceNil reversed_sources '())
		(cons src rest) (if (unused_unique_left_join? default_alias referenced_aliases src)
			(prune_unused_unique_left_joins_reversed rest default_alias referenced_aliases)
			(begin
				(define tail (prune_unused_unique_left_joins_reversed rest default_alias
					(query_expr_alias_set default_alias (source_join_expr src) referenced_aliases)))
				(cons src tail)))
		_ '())))

(define prune_unused_unique_left_joins (lambda (sources default_alias consumers)
	(reverse (prune_unused_unique_left_joins_reversed
		(reverse (coalesceNil sources '()))
		default_alias
		(query_exprs_alias_set default_alias consumers)))))


/* Walk an expression tree and collect every column name referenced from a
specific derived-table alias. The result is a flat assoc-list col_name->true. */
(define collect_alias_col_refs_deep (lambda (alias expr acc)
	(match expr
		((symbol get_column) tblvar _ col _) (if (equal? tblvar alias)
			(set_assoc acc col true)
			acc)
		((quote get_column) tblvar _ col _) (if (equal? tblvar alias)
			(set_assoc acc col true)
			acc)
		(cons head tail) (reduce tail
			(lambda (acc2 item) (collect_alias_col_refs_deep alias item acc2))
			(collect_alias_col_refs_deep alias head acc))
		_ acc)))

(define collect_alias_col_refs_from_exprs (lambda (alias exprs)
	(reduce (coalesceNil exprs '())
		(lambda (acc expr) (collect_alias_col_refs_deep alias expr acc))
		'())))

/* True if expr is or contains t.* or * for the given alias. */
(define expr_derived_star_ref (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		((quote get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		(cons head tail) (or (expr_derived_star_ref alias head)
			(reduce tail
				(lambda (found item) (or found (expr_derived_star_ref alias item)))
				false))
		_ false)))

(define has_derived_star_ref (lambda (alias exprs)
	(reduce (coalesceNil exprs '())
		(lambda (found expr) (or found (expr_derived_star_ref alias expr)))
		false)))

/* Filter a stride-2 field list (title1 expr1 ...) to entries in needed. */
(define filter_fields_by_needed (lambda (fields needed)
	(match (coalesceNil fields '())
		(cons title (cons expr rest))
		(begin
			(define tail (filter_fields_by_needed rest needed))
			(if (has_assoc? needed title)
				(cons title (cons expr tail))
				tail))
		_ '())))

/* Build pruned replacement source when inner projection can be trimmed. */
(define prune_derived_src_apply (lambda (src relation alias needed)
	(begin
		(define orig_fields (coalesceNil (qb_fields relation) '()))
		(define pruned_fields (filter_fields_by_needed orig_fields needed))
		(if (equal? (count pruned_fields) (count orig_fields))
			src
			(list
				alias
				(source_schema src)
				(make_query_block
					(qb_schema relation)
					(qb_sources relation)
					pruned_fields
					(qb_where relation)
					(qb_group relation)
					(qb_having relation)
					(qb_order relation)
					(qb_limit relation)
					(qb_offset relation)
					(qb_hidden relation)
					(qb_stages relation)
					(qb_facts relation))
				(source_outer? src)
				(source_join_expr src))))))

/* Pre-prune one derived-table source: drop inner fields not in needed.
outer_direct_consumers already includes all outer ON conditions. */
(define prune_derived_src (lambda (src outer_direct_consumers)
	(begin
		(define relation (source_relation src))
		(if (or (string? relation) (union_block? relation))
			src
			(begin
				(define alias (source_alias src))
				(if (has_derived_star_ref alias outer_direct_consumers)
					src
					(begin
						(define needed (collect_alias_col_refs_from_exprs alias outer_direct_consumers))
						(if (empty_list? needed)
							src
							(prune_derived_src_apply src relation alias needed)))))))))

/* For each derived-table source, drop inner projected columns not referenced
by the outer block's direct consumers (fields/where/group/having/order/hidden). */
(define prune_unreferenced_derived_fields (lambda (sources outer_direct_consumers)
	(map (coalesceNil sources '())
		(lambda (src) (prune_derived_src src outer_direct_consumers)))))

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
			/* SQL name resolution binds the innermost source first. A local
			alias that repeats an outer alias therefore shadows it rather than
			turning its qualified column references into correlations. */
			(define local_aliases (source_aliases (qb_sources block)))
			(define outer_aliases (filter (source_aliases outer_sources) (lambda (outer_alias)
				(not (reduce local_aliases (lambda (shadowed local_alias)
					(or shadowed (equal?? local_alias outer_alias))) false)))))
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

(define correlation_pair_using (lambda (inner_default inner_sources outer_sources term include_session)
	(match term
		((symbol equal??) a b) (begin
			(define a_inner (expr_refs_sources? inner_default inner_sources a))
			(define b_inner (expr_refs_sources? inner_default inner_sources b))
			(define a_outer (and (not a_inner) (or
				(expr_refs_sources? nil outer_sources a)
				(and include_session (session_dependency_expr? a)))))
			(define b_outer (and (not b_inner) (or
				(expr_refs_sources? nil outer_sources b)
				(and include_session (session_dependency_expr? b)))))
			(if (and a_inner b_outer)
				(list a b)
				(if (and b_inner a_outer)
					(list b a)
					nil)))
		((quote equal??) a b) (correlation_pair_using inner_default inner_sources outer_sources (list (quote equal??) a b) include_session)
		((symbol equal?) a b) (correlation_pair_using inner_default inner_sources outer_sources (list (quote equal??) a b) include_session)
		((quote equal?) a b) (correlation_pair_using inner_default inner_sources outer_sources (list (quote equal??) a b) include_session)
		_ nil)))

(define exists_correlation_pair (lambda (inner_default inner_sources outer_sources term)
	(correlation_pair_using inner_default inner_sources outer_sources term false)))

(define presence_correlation_pair (lambda (inner_default inner_sources outer_sources term)
	(correlation_pair_using inner_default inner_sources outer_sources term true)))

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

/* Equality decorrelation makes each outer lookup expression equivalent to its
inner key. Rewrite later scalar expressions to that local representative before
they enter a group-stage, so physical lowering never receives a free outer
binding. The structural index hashes the complete root once; each AST node is
then matched without repeated deep comparisons. */
(define decorrelate_expr_with_pairs (lambda (inner_default pairs expr)
	(begin
		(define pair_list (coalesceNil pairs '()))
		(if (empty_list? pair_list)
			expr
			(begin
				(define replacements (correlation_inner_keys inner_default pair_list))
				(define index (make_structural_index (correlation_lookup_keys pair_list) (list expr)))
				(define rewrite (lambda (node)
					(begin
						(define idx (index node))
						(if (nil? idx)
							(match node
								(cons head tail) (cons head (map tail rewrite))
								_ node)
							(nth replacements idx)))))
				(rewrite expr))))))

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
(define btw2025_info_accessing (lambda (info) (nth info 6)))
(define btw2025_info_accessing_after_simple (lambda (info) (nth info 7)))
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

/* Compute the Neumann dependent-join inputs once. EXISTS, IN and scalar
builders differ in result semantics, not in how they partition correlated and
local predicates. Keeping this analysis central prevents the six builders from
drifting on source-join pairs or residual outer references. */
(define analyze_query_correlations (lambda (inner outer_sources pair_fn extra_pairs include_source_pairs)
	(begin
		(define inner_sources (qb_sources inner))
		(define inner_src (if (empty_list? inner_sources) nil (car inner_sources)))
		(define inner_default (if (nil? inner_src) nil (source_alias inner_src)))
		(define terms (split_and_terms (coalesceNil (qb_where inner) true)))
		(define term_pairs (filter (map terms (lambda (term)
			(pair_fn inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define source_pairs (if include_source_pairs
			(source_join_correlation_pairs inner_default inner_sources outer_sources inner_sources)
			'()))
		(define lookup_pairs (domain_correlation_pairs
			(merge (list term_pairs source_pairs (coalesceNil extra_pairs '())))))
		(define local_terms (filter terms (lambda (term)
			(nil? (pair_fn inner_default inner_sources outer_sources term)))))
		(define local_sources
			(sources_without_outer_join_terms inner_default inner_sources outer_sources inner_sources))
		(define residual_outer_refs (merge_unique (list
			(btw2025_terms_outer_column_refs local_terms inner_sources outer_sources)
			(btw2025_sources_outer_column_refs local_sources inner_sources outer_sources))))
		(list
			(list (quote inner_src) inner_src)
			(list (quote inner_sources) inner_sources)
			(list (quote inner_default) inner_default)
			(list (quote lookup_pairs) lookup_pairs)
			(list (quote local_terms) local_terms)
			(list (quote local_sources) local_sources)
			(list (quote residual_outer_refs) residual_outer_refs)))))

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

(define make_positive_in_join_condition (lambda (input stage_alias key_names lookup_keys probe match_ag)
	(combine_where_terms
		(list
			(make_exists_stage_join_condition stage_alias key_names lookup_keys)
			(list (quote not) (list (quote nil?) probe))
			(list (quote >) (membership_count_expr input stage_alias match_ag) 0))
		true)))

/* One aggregate encodes empty=0, nonempty=1, and contains-NULL=2. */
(define in_rhs_state_descriptor (lambda (rhs_expr)
	(list (list (quote if) (list (quote nil?) rhs_expr) 2 1) (quote max) 0)))

(define membership_count_expr (lambda (input stage_alias ag)
	(list (quote coalesceNil)
		(list (quote get_column) stage_alias false (aggregate_col_name_using input ag) false)
		0)))

/* Canonical positive-WHERE membership primitive. Its stage-output source
provides the lookup relation; UNKNOWN and FALSE are intentionally equivalent in
this truth context. Reordering may later replace this semantic primitive with a
physical membership probe. */
(define in_membership_truth_expr (lambda (input probe stage_alias match_ag)
	(list (quote membership_truth) probe stage_alias (aggregate_col_name_using input match_ag))))

(define expand_in_membership_truth_expr (lambda (probe stage_alias count_col)
	(list (quote and)
		(list (quote not) (list (quote nil?) probe))
		(list (quote >)
			(list (quote coalesceNil)
				(list (quote get_column) stage_alias false count_col false)
				0)
			0))))

(define in_membership_expr (lambda (match_input null_input probe match_alias match_ag null_alias rhs_state_ag)
	(begin
		(define match_count (membership_count_expr match_input match_alias match_ag))
		(define rhs_state (membership_count_expr null_input null_alias rhs_state_ag))
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

(define not_in_membership_expr (lambda (match_input null_input probe match_alias match_ag null_alias rhs_state_ag)
	(begin
		(define match_count (membership_count_expr match_input match_alias match_ag))
		(define rhs_state (membership_count_expr null_input null_alias rhs_state_ag))
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

(define scalar_once_ordered_payload? (lambda (ag)
	(match ag
		'(((symbol scalar_first_payload) _order_expr _value_expr) _reduce _neutral) true
		'(((quote scalar_first_payload) _order_expr _value_expr) _reduce _neutral) true
		_ false)))

(define scalar_once_value_expr (lambda (input stage_alias ag)
	(begin
		(define stored (list (quote get_column) stage_alias false (aggregate_col_name_using input ag) false))
		(if (scalar_once_ordered_payload? ag)
			(list (quote if)
				(list (quote or)
					(list (quote nil?) stored)
					(list (quote nil?) (list (quote car) stored)))
				nil
				(list (quote cadr) stored))
			stored))))

(define scalar_single_value_expr (lambda (input stage_alias value_ag count_ag)
	(begin
		(define count_expr (list (quote coalesceNil)
			(list (quote get_column) stage_alias false (aggregate_col_name_using input count_ag) false)
			0))
		(define value_expr (list (quote get_column) stage_alias false (aggregate_col_name_using input value_ag) false))
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

/* Scalar row bounds belong to the decorrelated LEFT JOIN helper. Physical
lowering consumes this contract instead of reconstructing scalar semantics from
the original SQL shape after join reorder and stage rewrites. */
(define bounded_probe_physical_max_rows (lambda (stage)
	(begin
		(define facts (gs_facts stage))
		(define mode (qassoc_get facts (quote cardinality_mode) nil))
		(define max_rows (qassoc_get facts (quote physical_max_rows) nil))
		(define on_overflow (qassoc_get facts (quote on_overflow) nil))
		(match mode
			(symbol first) (if (and (equal? max_rows 1) (equal? on_overflow (quote ignore)))
				max_rows
				(neumann_fail "build_queryplan" "scalar first stage requires physical_max_rows=1 and on_overflow=ignore"))
			(symbol single_or_error) (if (and (equal? max_rows 2) (equal? on_overflow (quote error)))
				max_rows
				(neumann_fail "build_queryplan" "scalar single_or_error stage requires physical_max_rows=2 and on_overflow=error"))
			(symbol many) (if (and (equal? (qassoc_get facts (quote presence_only) false) true)
				(and (equal? (qassoc_get facts (quote max_needed_per_domain) nil) 1)
					(and (equal? max_rows 1) (equal? on_overflow (quote ignore)))))
				max_rows
				(neumann_fail "build_queryplan" "presence probe stage requires physical_max_rows=1 and on_overflow=ignore"))
			_ (neumann_fail "build_queryplan" "bounded probe stage is missing its cardinality contract")))))

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

/* graph is the assoc built by stage_dependency_graph (logical_stage_key ->
list of direct-dependency stages); a stage with no entry has none. */
(define stage_direct_deps (lambda (graph stage)
	(if (has_assoc? graph (logical_stage_key stage)) (graph (logical_stage_key stage)) '())))

(define boolean_expr_comparison_head? (lambda (head)
	(reduce (list
		(quote equal??) (symbol "equal??")
		(quote equal?) (symbol "equal?")
		(quote <) (symbol "<")
		(quote >) (symbol ">")
		(quote <=) (symbol "<=")
		(quote >=) (symbol ">=")
		(quote nil?) (symbol "nil?"))
		(lambda (found candidate) (or found (equal? head candidate)))
		false)))

/* CASE is encoded as (if condition result ... else). Conditions determine
which branch wins; only the result slots determine whether its value is
boolean-shaped. A missing ELSE is the implicit nullable boolean result. */
(define boolean_case_results_shaped? (lambda (graph stage items)
	(if (empty_list? items)
		true
		(if (empty_list? (cdr items))
			(boolean_typed_expr_shaped? graph stage (car items))
			(and (boolean_typed_expr_shaped? graph stage (nth items 1))
				(boolean_case_results_shaped? graph stage (cdr (cdr items))))))))

(define stage_dependency_for_output_alias (lambda (graph stage alias)
	(begin
		(define src (source_for_alias (stage_value_sources stage) nil alias false))
		(define target_id (if (and (not (nil? src))
			(stage_output_relation? (source_relation src)))
			(stage_output_relation_id (source_relation src))
			nil))
		(define indexed (if (nil? target_id)
			nil
			(reduce (stage_direct_deps graph stage) (lambda (found dep)
				(if (or (not (nil? found)) (not (equal? target_id (gs_id dep)))) found dep))
				nil)))
		(if (not (nil? indexed))
			indexed
			(reduce (stage_direct_deps graph stage) (lambda (found dep)
				(if (not (nil? found))
					found
					(if (or (and (not (nil? target_id)) (equal? target_id (gs_id dep)))
						(equal? alias (exists_stage_alias (gs_id dep))))
						dep
						nil)))
				nil)))))

(define stage_value_sources (lambda (stage)
	(if (query_block? (gs_input stage))
		(qb_sources (gs_input stage))
		(if (source_is_base_table? (gs_input stage))
			(list (gs_input stage))
			'()))))

(define source_boolean_column? (lambda (src col)
	(if (not (source_is_base_table? src))
		false
		(begin
			(define info (find (get_schema (source_schema src) (source_relation src))
				(lambda (candidate) (equal?? (candidate "Field") col)) nil))
			(define raw_type (toLower
				(if (nil? info) "" (coalesceNil (info "RawType") ""))))
			(or (equal? raw_type "bool") (equal? raw_type "boolean"))))))

(define stage_base_boolean_column? (lambda (stage tblvar col)
	(begin
		(define src (source_for_alias (stage_value_sources stage) nil tblvar false))
		(and (not (nil? src)) (source_boolean_column? src col)))))

(define boolean_stage_dependency_leaf? (lambda (graph stage expr)
	(match expr
		((symbol get_column) tblvar _ignorecase col _col_ignorecase) (begin
			(define dep (stage_dependency_for_output_alias graph stage tblvar))
			(if (nil? dep)
				(stage_base_boolean_column? stage tblvar col)
				(stage_boolean_shaped? graph dep col)))
		((quote get_column) tblvar _ignorecase col _col_ignorecase)
		(boolean_stage_dependency_leaf? graph stage
			(list (symbol "get_column") tblvar false col false))
		_ false)))

/* Recognize boolean values structurally, independent of the SQL spelling
which produced them. Nested stage-output columns remain conservative leaves:
their generated alias selects the exact direct dependency and their column
selects that dependency's exact aggregate. Base-table leaves use the declared
column type instead of borrowing a sibling dependency's shape. */
(define boolean_typed_expr_shaped? (lambda (graph stage expr)
	(if (or (equal?? expr true) (equal?? expr false))
		true
		(if (presence_bool_stage_output_expr? expr)
			true
			(match expr
				(cons head tail)
				(if (or (equal? head (quote and))
					(or (equal? head (symbol "and"))
						(or (equal? head (quote or)) (equal? head (symbol "or")))))
					(reduce tail (lambda (shaped item)
						(and shaped (boolean_typed_expr_shaped? graph stage item)))
						true)
					(if (or (equal? head (quote not))
						(or (equal? head (symbol "not"))
							(or (equal? head (quote sql_not)) (equal? head (symbol "sql_not")))))
						(and (equal? (count tail) 1)
							(boolean_typed_expr_shaped? graph stage (car tail)))
						(if (or (equal? head (quote if)) (equal? head (symbol "if")))
							(boolean_case_results_shaped? graph stage tail)
							(if (or (equal? head (quote coalesceNil)) (equal? head (symbol "coalesceNil")))
								(and (equal? (count tail) 2)
									(and (or (equal?? (nth tail 1) false) (equal?? (nth tail 1) true))
										(boolean_typed_expr_shaped? graph stage (car tail))))
								(if (boolean_expr_comparison_head? head)
									true
									(boolean_stage_dependency_leaf? graph stage expr))))))
				_ false)))))

/* Whether stage's own realized value is boolean-shaped, considering both the
count>0 encoding (presence_probe_stage? itself, or a passthrough of one) and a
plain/coalesced passthrough of another boolean-shaped nested stage --
recursively, so a chain of "just forwards the inner stage's boolean" wrappers
(exactly what COALESCE-wrapped nested EXISTS chains compile into) is still
recognized as boolean many levels down, not just at the first one.

requested_col picks which of stage's (possibly several) aggregates to inspect
at the top of the recursion, matching scalar_first_probe_aggregate's lookup;
nil deliberately selects the stage's primary aggregate for callers which
validate the stage as a whole rather than a particular output column. */
(define stage_boolean_shaped? (lambda (graph stage requested_col)
	(or (presence_probe_stage? stage)
		(begin
			(define ag (if (nil? requested_col)
				(if (empty_list? (gs_aggregates stage)) nil (car (gs_aggregates stage)))
				(scalar_first_probe_aggregate stage requested_col)))
			(and (not (nil? ag))
				(begin
					(define value_expr (car (scalar_first_probe_parts ag)))
					(boolean_typed_expr_shaped? graph stage value_expr)))))))

(define driver_membership_boolean_passthrough? (lambda (graph stage expr)
	(if (presence_bool_stage_output_expr? expr)
		true
		(match expr
			((symbol coalesceNil) inner default)
			(and (or (equal?? default false) (equal?? default true))
				(driver_membership_boolean_passthrough? graph stage inner))
			((quote coalesceNil) inner default)
			(driver_membership_boolean_passthrough? graph stage
				(list (symbol "coalesceNil") inner default))
			((symbol get_column) tblvar _ignorecase col _col_ignorecase) (begin
				(define dep (stage_dependency_for_output_alias graph stage tblvar))
				(and (not (nil? dep))
					(driver_membership_stage_boolean_shaped? graph dep col)))
			((quote get_column) tblvar _ignorecase col _col_ignorecase)
			(driver_membership_boolean_passthrough? graph stage
				(list (symbol "get_column") tblvar false col false))
			_ false))))

/* Driver-membership RecSets have a separate, deliberately conservative
consumer whose value-embedding fallback is not valid for every join shape.
Keep its historic passthrough-only proof while scalar probe lowering uses the
general recursive boolean proof above. */
(define driver_membership_stage_boolean_shaped? (lambda (graph stage requested_col)
	(or (presence_probe_stage? stage)
		(begin
			(define ag (if (nil? requested_col)
				(if (empty_list? (gs_aggregates stage)) nil (car (gs_aggregates stage)))
				(scalar_first_probe_aggregate stage requested_col)))
			(and (not (nil? ag))
				(begin
					(define value_expr (car (scalar_first_probe_parts ag)))
					(driver_membership_boolean_passthrough?
						graph stage value_expr)))))))

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
		(define order_ags (dedupe_aggregates_by_col
			(merge (map (order_item_exprs (qb_order inner)) extract_aggregates))))
		(define ags (dedupe_aggregates_by_col
			(merge (list visible_ags order_ags (list aggregate_count_descriptor)))))
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
		(define analysis (analyze_query_correlations inner outer_sources
			presence_correlation_pair (presence_session_domain_pairs inner) true))
		(define inner_src (qassoc_get analysis (quote inner_src) nil))
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define lookup_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
		(define residual_outer_refs (qassoc_get analysis (quote residual_outer_refs) '()))
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
					(list (quote physical_max_rows) 1)
					(list (quote on_overflow) (quote ignore))
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
		(define analysis (analyze_query_correlations inner outer_sources
			presence_correlation_pair (presence_session_domain_pairs inner) true))
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define lookup_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
		(define explicit_keys (map (qb_group inner) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define keys (group_keys_for_correlations inner_default lookup_pairs explicit_keys))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define ags (dedupe_aggregates_by_col (merge
			(list (extract_aggregates (coalesceNil (qb_having inner) true))
				(list aggregate_count_descriptor)))))
		(define stage_input (make_query_block
			(qb_schema inner)
			local_sources
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
					(list (quote max_needed_per_domain) 1)
					(list (quote physical_max_rows) 1)
					(list (quote on_overflow) (quote ignore))
					(list (quote domain) outer_domain)
					(list (quote lookup-keys) lookup_keys)
					(list (quote preserve_empty_domain) (not (empty_list? outer_domain)))
					(list (quote null_semantics) (quote exists))
					(list (quote cardinality_mode) (quote many)))
				(btw2025_stage_facts inner outer_sources lookup_pairs '() pending_info)))))
		(define stage_alias (exists_stage_alias stage_id))
		(define key_names (group_key_cols keys))
		(define having_expr (replace_group_expr
			(gs_input stage) inner_default stage_alias keys key_names ags (coalesceNil (qb_having inner) true)))
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
				(list (quote physical_max_rows) 1)
				(list (quote on_overflow) (quote ignore))
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
				(make_positive_in_join_condition (gs_input stage) stage_alias key_names (list probe) probe aggregate_count_descriptor)
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
				(in_membership_expr (gs_input stage) (gs_input null_stage)
					probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
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
		(define analysis (analyze_query_correlations membership_inner outer_sources
			exists_correlation_pair (session_domain_pairs membership_inner) false))
		(define inner_src (qassoc_get analysis (quote inner_src) nil))
		(define inner_sources (qassoc_get analysis (quote inner_sources) '()))
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define lookup_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
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
				local_sources
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
				(make_positive_in_join_condition (gs_input stage) stage_alias key_names lookup_keys probe aggregate_count_descriptor)
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
					(in_membership_truth_expr (gs_input stage) probe stage_alias aggregate_count_descriptor)
					(list stage)
					(list source))
				(list
					(if negate
						(not_in_membership_expr (gs_input stage) (gs_input null_stage)
							probe stage_alias aggregate_count_descriptor null_stage_alias null_ag)
						(in_membership_expr (gs_input stage) (gs_input null_stage)
							probe stage_alias aggregate_count_descriptor null_stage_alias null_ag))
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
			(gs_input stage) input_alias stage_alias keys key_names ags (coalesceNil (gs_having stage) true)))
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
		(define analysis (analyze_query_correlations inner outer_sources
			exists_correlation_pair (session_domain_pairs inner) true))
		(define inner_src (qassoc_get analysis (quote inner_src) nil))
		(define inner_sources (qassoc_get analysis (quote inner_sources) '()))
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define where_corr_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
		(define having_terms (split_and_terms (coalesceNil (qb_having inner) true)))
		(define having_corr_pairs (filter (map having_terms (lambda (term)
			(exists_correlation_pair inner_default inner_sources outer_sources term)))
			(lambda (pair) (not (nil? pair)))))
		(define local_having_terms (filter having_terms (lambda (term)
			(nil? (exists_correlation_pair inner_default inner_sources outer_sources term)))))
		(define all_corr_pairs (domain_correlation_pairs (merge (list
			where_corr_pairs having_corr_pairs))))
		(define local_value_expr (decorrelate_expr_with_pairs inner_default all_corr_pairs value_expr))
		(define ags (dedupe_aggregates_by_col (merge (list (extract_aggregates local_value_expr) (list aggregate_count_descriptor)))))
		(if (empty_list? ags)
			(neumann_fail "untangle_query" "table-backed scalar subquery without aggregate needs cardinality_mode single_or_error lowering")
			true)
		(define explicit_group_keys (map (coalesceNil (qb_group inner) '()) (lambda (expr)
			(canonical_column_expr_for_alias inner_default expr))))
		(define keys (group_keys_for_correlations inner_default all_corr_pairs explicit_group_keys))
		(define outer_domain (correlation_domain all_corr_pairs))
		(define lookup_keys (correlation_lookup_keys all_corr_pairs))
		(define condition (combine_where_terms local_terms true))
		(define local_having (decorrelate_expr_with_pairs inner_default all_corr_pairs
			(combine_where_terms local_having_terms true)))
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
		(define post_condition (replace_group_expr stage_input inner_default stage_alias keys key_names ags local_having))
		(define source (list
			stage_alias
			(group_stage_schema stage)
			(make_stage_output_relation stage_id)
			(or (stage_source_outer? outer_sources) (not (equal? local_having true)))
			(make_stage_lookup_condition stage_alias key_names lookup_keys post_condition)))
		(define presence_expr (list (quote get_column) stage_alias false (aggregate_col_name aggregate_count_descriptor) false))
		(define scalar_value_expr (replace_group_expr stage_input inner_default stage_alias keys key_names ags local_value_expr))
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
		(define output_index (if (>= (count args) 4) (nth args 3) 0))
		(if (not (scalar_once_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
			true)
		(define value_exprs (extract_assoc (qb_fields inner) (lambda (_title expr) expr)))
		(if (>= output_index (count value_exprs))
			(neumann_fail "untangle_query" "scalar output index exceeds the bundled projection")
			true)
		(define analysis (analyze_query_correlations inner outer_sources
			exists_correlation_pair (session_domain_pairs inner) true))
		(define inner_src (qassoc_get analysis (quote inner_src) nil))
		(if (not (source_is_base_table? inner_src))
			(neumann_fail "untangle_query" "scalar once_limit stage requires a base inner source after FROM flattening")
			true)
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define lookup_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(scalar_stage_inner_keys_for_correlations inner_default (qb_stages inner) (qb_sources inner) lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define values_for_inner (map value_exprs (lambda (value_expr)
			(canonical_column_expr_for_alias inner_default
				(decorrelate_expr_with_pairs inner_default lookup_pairs value_expr)))))
		(define order_for_inner (map (coalesceNil (qb_order inner) '()) (lambda (item)
			(match item '(expr dir) (list (canonical_column_expr_for_alias inner_default
				(decorrelate_expr_with_pairs inner_default lookup_pairs expr)) dir)))))
		(define ags (dedupe_aggregates_by_col (map values_for_inner (lambda (value_for_inner)
			(scalar_once_descriptor value_for_inner order_for_inner (qb_offset inner))))))
		(define stage_input (if (empty_list? (qb_stages inner))
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
		(define stage_id (concat "scalar-once:" (fnv_hash (string (list subquery keys outer_domain condition ags)))))
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
		(define output_exprs (map values_for_inner (lambda (value_for_inner)
			(scalar_once_value_expr stage_input stage_alias
				(scalar_once_descriptor value_for_inner order_for_inner (qb_offset inner))))))
		(list
			(nth output_exprs output_index)
			(list stage)
			(list source)
			output_exprs))))

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
				(scalar_once_value_expr stage_input alias ag)))))
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
			(replace_group_order_expr (gs_input stage) input_alias alias keys key_names ags expr))))))

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
					(rewrite_source_for_group_domain (gs_input stage) input_alias alias keys key_names ags extra)))
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
		/* LIMIT/OFFSET only bound readers of this complete row-number column. They
		do not alter the stored window and therefore must not fragment its cache. */
		(define rn_col (canonical_orc_column_name "derived_row_number" derived_src
			(list sortcols sortdirs (canonical_helper_expr_using derived_src inner_where))))
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
					(lower_column_expr_for_alias derived_src inner_where)))
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
				(merge (map (gs_aggregates stage) (lambda (ag)
					(begin
						(define rewritten_input (rewrite_stage_graph_expr base_alias_map id_map (gs_input stage)))
						(define rewritten_ag (rewrite_stage_graph_expr base_alias_map id_map ag))
						/* Logical probes from older rewrites may still carry raw descriptor
						hashes. Accept both forms while every physical writer uses the
						canonical source-aware name. */
						(list
							(list old_alias
								(aggregate_col_name ag)
								(aggregate_col_name rewritten_ag))
							(list old_alias
								(aggregate_col_name_using (gs_input stage) ag)
								(aggregate_col_name_using rewritten_input rewritten_ag))))))))))))
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
		(define output_index (if (>= (count args) 4) (nth args 3) 0))
		(if (not (scalar_single_supported? inner))
			(neumann_fail "untangle_query" "table-backed scalar subquery without explicit LIMIT 1 needs cardinality_mode single_or_error lowering")
			true)
		(define value_exprs (extract_assoc (qb_fields inner) (lambda (_title expr) expr)))
		(if (>= output_index (count value_exprs))
			(neumann_fail "untangle_query" "scalar output index exceeds the bundled projection")
			true)
		(define analysis (analyze_query_correlations inner outer_sources
			exists_correlation_pair (session_domain_pairs inner) true))
		(define inner_src (qassoc_get analysis (quote inner_src) nil))
		(define inner_default (qassoc_get analysis (quote inner_default) nil))
		(define lookup_pairs (qassoc_get analysis (quote lookup_pairs) '()))
		(define local_terms (qassoc_get analysis (quote local_terms) '()))
		(define local_sources (qassoc_get analysis (quote local_sources) '()))
		(define keys (if (empty_list? lookup_pairs)
			'(1)
			(correlation_inner_keys inner_default lookup_pairs)))
		(define outer_domain (correlation_domain lookup_pairs))
		(define lookup_keys (correlation_lookup_keys lookup_pairs))
		(define condition (combine_where_terms local_terms true))
		(define values_for_inner (map value_exprs (lambda (value_expr)
			(canonical_column_expr_for_alias inner_default
				(decorrelate_expr_with_pairs inner_default lookup_pairs value_expr)))))
		(define value_ags (map values_for_inner (lambda (value_for_inner)
			(car (scalar_single_aggregates value_for_inner)))))
		(define count_ag aggregate_count_descriptor)
		(define ags (dedupe_aggregates_by_col (merge (list value_ags (list count_ag)))))
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
		(define output_exprs (map value_ags (lambda (output_ag)
			(scalar_single_value_expr stage_input stage_alias output_ag count_ag))))
		(list
			(nth output_exprs output_index)
			(list stage)
			(list source)
			output_exprs))))

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

(define window_aggregate_value_expr (lambda (input fn args ags stage_alias)
	(match fn
		"COUNT" (list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
		"SUM" (list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
		"AVG" (list (quote sql_avg_divide)
			(list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
			(list (quote get_column) stage_alias false (aggregate_col_name_using input (cadr ags)) false))
		"MAX" (list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
		"MIN" (list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
		"GROUP_CONCAT" (list (quote get_column) stage_alias false (aggregate_col_name_using input (car ags)) false)
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
						(define col (canonical_orc_column_name (toLower fn) src
							(list value_col sortcols sortdirs offset (count partition_exprs))))
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
						(define col (canonical_orc_column_name (toLower fn) src
							(list sortcols sortdirs (count partition_exprs))))
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
						(define col (canonical_orc_column_name "row_number" src
							(list sortcols sortdirs (count partition_exprs))))
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
			(window_aggregate_value_expr src fn canonical_args ags stage_alias)
			(list stage)
			(list source)))))

(define combine_stage_rewrite_results (lambda (head rewritten_args)
	(begin
		(define expr (cons head (map rewritten_args (lambda (item) (nth item 0)))))
		(define stages (unique_stages_by_id (merge (map rewritten_args (lambda (item) (nth item 1))))))
		(define sources (merge_unique (map rewritten_args (lambda (item) (nth item 2)))))
		(list expr stages sources))))

(define untangle_scalar_subquery_with_stages (lambda (subquery outer_sources ctx output_index)
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
						(make_scalar_once_stage_rewrite inner (list outer_sources subquery pending_info output_index))
						(make_scalar_single_stage_rewrite inner (list outer_sources subquery pending_info output_index)))
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
			(untangle_scalar_subquery_with_stages subquery outer_sources ctx 0))
		((quote inner_select) subquery)
		(if (btw2025_defer_subquery_rewrite? subquery outer_sources ctx)
			(list (make_dependent_subquery_marker (quote scalar) nil subquery outer_sources) '() '())
			(untangle_scalar_subquery_with_stages subquery outer_sources ctx 0))
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

/* SQL must repeat a scalar subquery when callers need several values from the
same correlated row. Preserve that syntax at the boundary, but represent the
shared relational work once: equivalent scalar LIMIT 1 backbones become one
logical stage with one aggregate output per demanded projection. Physical
carrier selection remains a later, per-stage cost decision. */
(define btw2025_scalar_bundle_candidate? (lambda (marker)
	(and (dependent_subquery_marker? marker)
		(and (equal? (dep_subquery_kind marker) (quote scalar))
			(begin
				(define subquery (normalize_query_ast (dep_subquery_query marker)))
				(and (query_block? subquery)
					(and (not (empty_list? (qb_sources subquery)))
						(and (equal? (count (qb_fields subquery)) 2)
							(and (equal? (qb_limit subquery) 1)
								(empty_list? (extract_aggregates (query_block_first_expr subquery))))))))))))

(define btw2025_scalar_bundle_key (lambda (marker)
	(concat "scalar-bundle:"
		(serialize (list
			(dependent_subquery_scope_backbone (quote scalar) (dep_subquery_query marker))
			(dep_subquery_outer_sources marker))))))

(define btw2025_scalar_bundle_output_budget 16)

(define btw2025_scalar_markers_in_expr (lambda (expr)
	(match expr
		((symbol dependent-subquery) _kind _probe _subquery _outer_sources)
		(if (btw2025_scalar_bundle_candidate? expr) (list expr) '())
		((quote dependent-subquery) _kind _probe _subquery _outer_sources)
		(if (btw2025_scalar_bundle_candidate? expr) (list expr) '())
		(cons _head tail) (merge (map tail btw2025_scalar_markers_in_expr))
		_ '())))

(define btw2025_scalar_bundle_groups (lambda (markers)
	(reduce markers (lambda (groups marker)
		(begin
			(define key (btw2025_scalar_bundle_key marker))
			(define entries (qassoc_get groups key '()))
			(qassoc_set groups key
				(if (contains? entries marker) entries (merge entries (list marker))))))
		'())))

(define btw2025_scalar_bundle_alias_map (lambda (target candidate)
	(begin
		(define target_aliases (source_aliases (qb_sources target)))
		(define candidate_aliases (source_aliases (qb_sources candidate)))
		(if (not (equal? (count target_aliases) (count candidate_aliases)))
			(neumann_fail "untangle_query" "scalar bundle backbones expose different source arity")
			(map (produceN (count candidate_aliases)) (lambda (i)
				(list (nth candidate_aliases i) (nth target_aliases i))))))))

(define btw2025_scalar_bundle_fields (lambda (markers)
	(begin
		(define target (normalize_query_ast (dep_subquery_query (car markers))))
		(merge (mapIndex markers (lambda (i marker)
			(begin
				(define candidate (normalize_query_ast (dep_subquery_query marker)))
				(list
					(concat "__scalar_bundle_" i)
					(rewrite_stage_graph_expr
						(btw2025_scalar_bundle_alias_map target candidate)
						'()
						(query_block_first_expr candidate))))))))))

(define btw2025_scalar_bundle_query (lambda (markers)
	(begin
		(define target (normalize_query_ast (dep_subquery_query (car markers))))
		(make_query_block
			(qb_schema target)
			(qb_sources target)
			(btw2025_scalar_bundle_fields markers)
			(qb_where target)
			(qb_group target)
			(qb_having target)
			(qb_order target)
			(qb_limit target)
			(qb_offset target)
			(qb_hidden target)
			(qb_stages target)
			(qb_facts target)))))

(define btw2025_scalar_bundle_marker_index (lambda (markers marker index)
	(match markers
		(cons current rest) (if (equal? current marker)
			index
			(btw2025_scalar_bundle_marker_index rest marker (+ index 1)))
		_ 0)))

(define btw2025_bundle_scalar_marker (lambda (marker groups)
	(if (not (btw2025_scalar_bundle_candidate? marker))
		marker
		(begin
			(define markers (qassoc_get groups (btw2025_scalar_bundle_key marker) '()))
			/* Very wide demand sets are cheaper through the existing canonical
			stage merger: it deduplicates equivalent outputs before recipe emission.
			Keep bundling focused on the compact multi-column row lookups for which it
			removes relational work without inflating the generated program. */
			(if (or (<= (count markers) 1)
				(> (count markers) btw2025_scalar_bundle_output_budget))
				marker
				(make_dependent_subquery_marker
					(quote scalar-output)
					(btw2025_scalar_bundle_marker_index markers marker 0)
					(btw2025_scalar_bundle_query markers)
					(dep_subquery_outer_sources marker)))))))

(define btw2025_bundle_scalar_markers_in_expr (lambda (expr groups)
	(match expr
		((symbol dependent-subquery) _kind _probe _subquery _outer_sources)
		(btw2025_bundle_scalar_marker expr groups)
		((quote dependent-subquery) _kind _probe _subquery _outer_sources)
		(btw2025_bundle_scalar_marker expr groups)
		(cons head tail) (cons head
			(map tail (lambda (item) (btw2025_bundle_scalar_markers_in_expr item groups))))
		_ expr)))

(define btw2025_bundle_scalar_markers_in_block (lambda (block)
	(begin
		(define marker_scope (list
			(map (qb_sources block) source_join_expr)
			(qb_fields block)
			(qb_where block)
			(qb_group block)
			(qb_having block)
			(qb_order block)
			(qb_hidden block)))
		(define groups (btw2025_scalar_bundle_groups
			(btw2025_scalar_markers_in_expr marker_scope)))
		(define rewrite (lambda (expr) (btw2025_bundle_scalar_markers_in_expr expr groups)))
		(define rewritten_sources (map (qb_sources block) (lambda (src)
			(source_with_join_expr src (rewrite (source_join_expr src))))))
		(define rewritten_fields (rewrite (qb_fields block)))
		(make_query_block
			(qb_schema block)
			rewritten_sources
			rewritten_fields
			(rewrite (qb_where block))
			(rewrite (qb_group block))
			(rewrite (qb_having block))
			(rewrite (qb_order block))
			(qb_limit block)
			(qb_offset block)
			(rewrite (qb_hidden block))
			(qb_stages block)
			(qb_facts block)))))

(define btw2025_resolve_dependent_subquery (lambda (marker ctx)
	(begin
		(define kind (dep_subquery_kind marker))
		(define probe (dep_subquery_probe marker))
		(define subquery (dep_subquery_query marker))
		(define outer_sources (dep_subquery_outer_sources marker))
		(define parent (uctx_get ctx (quote btw2025-current-handle) nil))
		(define parent_ancestors (uctx_get ctx (quote btw2025-ancestor-handles) '()))
		(define current_ancestors (if (nil? parent) '() (cons parent parent_ancestors)))
		(define current_handle (concat "djoin:" (fnv_hash (string (list
			kind
			(dependent_subquery_scope_backbone kind subquery)
			outer_sources)))))
		(define current_info (btw2025_pending_unnesting_info subquery outer_sources current_handle parent current_ancestors))
		(define resolve_ctx (make_uctx ctx (list
			(list (quote defer-subquery-rewrites) true)
			(list (quote btw2025-current-handle) current_handle)
			(list (quote btw2025-ancestor-handles) current_ancestors)
			(list (quote btw2025-current-info) current_info))))
		(match kind
			(symbol scalar)
			(untangle_scalar_subquery_with_stages subquery outer_sources resolve_ctx 0)
			(symbol scalar-output)
			(untangle_scalar_subquery_with_stages subquery outer_sources resolve_ctx probe)
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
		(if (equal? (dep_subquery_kind expr) (quote scalar-output))
			(concat "dependent-bundle:" (fnv_hash (string (list
				(dep_subquery_kind expr)
				(dep_subquery_query expr)
				(dep_subquery_outer_sources expr)))))
			(concat "dependent:" (fnv_hash (string expr))))
		nil)))

(define btw2025_cached_dependent_rewrite (lambda (marker rewritten)
	(if (and (equal? (dep_subquery_kind marker) (quote scalar-output))
		(>= (count rewritten) 4))
		(list
			(nth (nth rewritten 3) (dep_subquery_probe marker))
			(nth rewritten 1)
			(nth rewritten 2)
			(nth rewritten 3))
		rewritten)))

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
			(list (btw2025_cached_dependent_rewrite expr (resolved key)) resolved)
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

(define btw2025_decorrelate_order_with_stages_using (lambda (order_items ctx resolved)
	(match (coalesceNil order_items '())
		(cons item rest) (begin
			(define rewritten_item (match item
				'(expr dir) (begin
					(define current (btw2025_decorrelate_expr_using expr ctx resolved))
					(define rewritten_expr (nth current 0))
					(list (list (nth rewritten_expr 0) dir)
						(nth rewritten_expr 1) (nth rewritten_expr 2) (nth current 1)))
				_ (list item '() '() resolved)))
			(define tail (btw2025_decorrelate_order_with_stages_using
				rest ctx (nth rewritten_item 3)))
			(list
				(cons (nth rewritten_item 0) (nth tail 0))
				(unique_stages_by_id (merge (list (nth rewritten_item 1) (nth tail 1))))
				(merge_unique (list (nth rewritten_item 2) (nth tail 2)))
				(nth tail 3)))
		_ (list '() '() '() resolved))))

(define btw2025_decorrelate_expr_list_with_stages_using (lambda (exprs ctx resolved)
	(match (coalesceNil exprs '())
		(cons expr rest) (begin
			(define current (btw2025_decorrelate_expr_using expr ctx resolved))
			(define rewritten (nth current 0))
			(define tail (btw2025_decorrelate_expr_list_with_stages_using rest ctx (nth current 1)))
			(list
				(cons (nth rewritten 0) (nth tail 0))
				(unique_stages_by_id (merge (list (nth rewritten 1) (nth tail 1))))
				(merge_unique (list (nth rewritten 2) (nth tail 2)))
				(nth tail 3)))
		_ (list '() '() '() resolved))))

(define btw2025_decorrelate_source_with_stages_using (lambda (src ctx resolved)
	(begin
		(define current (btw2025_decorrelate_expr_using (source_join_expr src) ctx resolved))
		(define rewritten_join (nth current 0))
		(list
			(merge_unique (list (nth rewritten_join 2) (list (source_with_join_expr src (nth rewritten_join 0)))))
			(nth rewritten_join 1)
			'()
			(nth current 1)))))

(define btw2025_decorrelate_sources_with_stages_using (lambda (sources ctx resolved)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define rewritten_src (btw2025_decorrelate_source_with_stages_using src ctx resolved))
			(define tail (btw2025_decorrelate_sources_with_stages_using rest ctx (nth rewritten_src 3)))
			(list
				(merge_unique (list (nth rewritten_src 0) (nth tail 0)))
				(unique_stages_by_id (merge (list (nth rewritten_src 1) (nth tail 1))))
				(merge_unique (list (nth rewritten_src 2) (nth tail 2)))
				(nth tail 3)))
		_ (list '() '() '() resolved))))

(define btw2025_decorrelate_query_block (lambda (block ctx)
	(begin
		(define bundled (btw2025_bundle_scalar_markers_in_block block))
		(define source_result (btw2025_decorrelate_sources_with_stages_using (qb_sources bundled) ctx '()))
		(define sources (nth source_result 0))
		(define source_stages (nth source_result 1))
		(define source_stage_sources (nth source_result 2))
		(define where_state (btw2025_decorrelate_expr_using
			(qb_where bundled) ctx (nth source_result 3)))
		(define where_result (nth where_state 0))
		(define field_result (btw2025_decorrelate_fields_with_stages_using
			(qb_fields bundled) ctx (nth where_state 1)))
		(define group_result (btw2025_decorrelate_expr_list_with_stages_using
			(qb_group bundled) ctx (nth field_result 3)))
		(define having_state (btw2025_decorrelate_expr_using
			(qb_having bundled) ctx (nth group_result 3)))
		(define having_result (nth having_state 0))
		(define order_result (btw2025_decorrelate_order_with_stages_using
			(qb_order bundled) ctx (nth having_state 1)))
		(define hidden_result (btw2025_decorrelate_fields_with_stages_using
			(qb_hidden bundled) ctx (nth order_result 3)))
		(make_query_block
			(qb_schema bundled)
			(merge_unique (list sources source_stage_sources (nth where_result 2) (nth field_result 2) (nth group_result 2) (nth having_result 2) (nth order_result 2) (nth hidden_result 2)))
			(nth field_result 0)
			(nth where_result 0)
			(nth group_result 0)
			(nth having_result 0)
			(nth order_result 0)
			(qb_limit bundled)
			(qb_offset bundled)
			(nth hidden_result 0)
			(unique_stages_by_id (merge (list (qb_stages bundled) source_stages (nth where_result 1) (nth field_result 1) (nth group_result 1) (nth having_result 1) (nth order_result 1) (nth hidden_result 1))))
			(qb_facts bundled)))))

(define field_expr_by_title (lambda (fields title ignorecase)
	(match (coalesceNil fields '())
		(cons current_title (cons expr rest)) (if (if ignorecase (equal?? current_title title) (equal? current_title title))
			expr
			(field_expr_by_title rest title ignorecase))
		_ nil)))

/* Derived relations expose their projection, not every column of the base
sources that replace them during logical flattening. Keep that SQL scope
available while nested queries are decorrelated. */
(define logical_relation_fields (lambda (relation)
	(begin
		(define normalized (normalize_query_ast relation))
		(if (query_block? normalized)
			(qb_fields normalized)
			(if (union_block? normalized)
				(match (union_branches normalized)
					(cons branch _rest) (logical_relation_fields branch)
					_ '())
				'())))))

/* Bind SQL names while query-block scopes and derived projections are still
logical relations. Later rewrites may flatten those relations, but must no
longer decide which scope an unqualified name belongs to. Scope lists are
ordered from the innermost query block to the outermost one.

Each entry stores the SQL-visible alias, its alpha-renamed logical source, and
a precomputed export catalog. The SQL alias is only a scoped spelling; the
source alias is the stable identity consumed by all later planner phases. */
(define binding_field_titles (lambda (fields)
	(extract_assoc (coalesceNil fields '()) (lambda (title _expr) title))))

(define binding_source_columns (lambda (src)
	(begin
		(define relation (normalize_query_ast (source_relation src)))
		(if (string? relation)
			(map (get_schema (source_schema src) relation) (lambda (column) (column "Field")))
			(binding_field_titles (logical_relation_fields relation))))))

(define binding_column_name (lambda (columns col col_ignorecase)
	(reduce (coalesceNil columns '()) (lambda (found candidate)
		(if (not (nil? found))
			found
			(if (if col_ignorecase (equal?? candidate col) (equal? candidate col)) candidate nil))) nil)))

(define binding_column_catalog (lambda (columns)
	(reduce (coalesceNil columns '()) (lambda (catalog column)
		(list
			(if (has_assoc? (nth catalog 0) column)
				(nth catalog 0)
				(set_assoc (nth catalog 0) column column))
			(begin
				(define folded (toLower column))
				(if (has_assoc? (nth catalog 1) folded)
					(nth catalog 1)
					(set_assoc (nth catalog 1) folded column)))))
		(list '() '()))))

(define binding_catalog_column_name (lambda (catalog col col_ignorecase)
	(if col_ignorecase
		(get_assoc (nth catalog 1) (toLower col) nil)
		(get_assoc (nth catalog 0) col nil))))

(define binding_entry_name (lambda (entry) (nth entry 0)))
(define binding_entry_source (lambda (entry) (nth entry 1)))
(define binding_entry_columns (lambda (entry) (nth entry 2)))
(define binding_entry_catalog_known? (lambda (entry) (nth entry 3)))

(define binding_entries_for_alias (lambda (entries tblvar tbl_ignorecase)
	(filter (coalesceNil entries '()) (lambda (entry)
		(if tbl_ignorecase
			(equal?? (binding_entry_name entry) tblvar)
			(or
				(equal? (binding_entry_name entry) tblvar)
				/* Expanded output aliases already contain bound references. */
				(equal? (source_alias (binding_entry_source entry)) tblvar)))))))

(define binding_entries_for_column (lambda (entries col col_ignorecase)
	(filter (coalesceNil entries '()) (lambda (entry)
		(or
			(not (binding_entry_catalog_known? entry))
			(not (nil? (binding_catalog_column_name (binding_entry_columns entry) col col_ignorecase))))))))

(define bind_qualified_column_in_scopes (lambda (scopes tblvar tbl_ignorecase col col_ignorecase)
	(match (coalesceNil scopes '())
		(cons entries outer_scopes) (begin
			(define matches (binding_entries_for_alias entries tblvar tbl_ignorecase))
			(if (empty_list? matches)
				(bind_qualified_column_in_scopes outer_scopes tblvar tbl_ignorecase col col_ignorecase)
				(if (single_source? matches)
					(begin
						(define entry (car matches))
						(define src (binding_entry_source entry))
						(if (equal? col "*")
							(list (quote get_column) (source_alias src) false "*" false)
							(begin
								(define resolved_col (binding_catalog_column_name (binding_entry_columns entry) col col_ignorecase))
								(if (nil? resolved_col)
									(if (binding_entry_catalog_known? entry)
										(neumann_fail "bind_query_names" (concat "Column does not exist: " (concat tblvar (concat "." col))))
										(list (quote get_column) (source_alias src) false col col_ignorecase))
									(list (quote get_column) (source_alias src) false resolved_col false)))))
					(neumann_fail "bind_query_names" (concat "ambiguous relation alias: " tblvar)))))
		_ (neumann_fail "bind_query_names" (concat "unknown relation alias: " tblvar)))))

(define bind_unqualified_column_in_scopes (lambda (scopes col col_ignorecase)
	(match (coalesceNil scopes '())
		(cons entries outer_scopes) (begin
			(define matches (binding_entries_for_column entries col col_ignorecase))
			(if (empty_list? matches)
				(bind_unqualified_column_in_scopes outer_scopes col col_ignorecase)
				(if (single_source? matches)
					(begin
						(define entry (car matches))
						(list (quote get_column)
							(source_alias (binding_entry_source entry))
							false
							(coalesceNil (binding_catalog_column_name (binding_entry_columns entry) col col_ignorecase) col)
							(if (binding_entry_catalog_known? entry) false col_ignorecase)))
					(neumann_fail "bind_query_names" (concat "ambiguous unqualified column: " col)))))
		_ (neumann_fail "bind_query_names" (concat "Column does not exist: " col)))))

(define try_bind_unqualified_column_in_scopes (lambda (scopes col col_ignorecase)
	(match (coalesceNil scopes '())
		(cons entries outer_scopes) (begin
			(define matches (binding_entries_for_column entries col col_ignorecase))
			(if (empty_list? matches)
				(try_bind_unqualified_column_in_scopes outer_scopes col col_ignorecase)
				(if (single_source? matches)
					(begin
						(define entry (car matches))
						(list (quote get_column)
							(source_alias (binding_entry_source entry)) false
							(coalesceNil (binding_catalog_column_name (binding_entry_columns entry) col col_ignorecase) col)
							(if (binding_entry_catalog_known? entry) false col_ignorecase)))
					(neumann_fail "bind_query_names" (concat "ambiguous unqualified column: " col)))))
		_ nil)))

(define binding_path (lambda (parent kind index)
	(concat parent (concat "/" (concat kind (concat ":" (string index)))))))

(define binding_fresh_alias (lambda (string_catalog candidate)
	(if (has_assoc? string_catalog candidate)
		(binding_fresh_alias string_catalog (concat candidate ":"))
		candidate)))

(define binding_collect_string_catalog (lambda (value catalog)
	(if (string? value)
		(set_assoc catalog value true)
		(match value
			(cons head tail) (reduce tail (lambda (current item)
				(binding_collect_string_catalog item current))
				(binding_collect_string_catalog head catalog))
			_ catalog))))

(define binding_fresh_query_alias (lambda (query candidate)
	(binding_fresh_alias (binding_collect_string_catalog query '()) candidate)))

(define bind_query_expr_tail (lambda (scopes items path all_strings index)
	(match (coalesceNil items '())
		(cons item rest) (cons
			(bind_query_expr scopes item (binding_path path "expr" index) all_strings)
			(bind_query_expr_tail scopes rest path all_strings (+ index 1)))
		_ '())))

(define bind_query_expr (lambda (scopes expr path all_strings)
	(match expr
		((symbol inner_select) subquery)
		(list (quote inner_select) (bind_query_names_at subquery scopes (binding_path path "subquery" 0) all_strings))
		((quote inner_select) subquery)
		(list (quote inner_select) (bind_query_names_at subquery scopes (binding_path path "subquery" 0) all_strings))
		((symbol inner_select_exists) subquery)
		(list (quote inner_select_exists) (bind_query_names_at subquery scopes (binding_path path "exists" 0) all_strings))
		((quote inner_select_exists) subquery)
		(list (quote inner_select_exists) (bind_query_names_at subquery scopes (binding_path path "exists" 0) all_strings))
		((symbol inner_select_in) probe subquery)
		(list (quote inner_select_in)
			(bind_query_expr scopes probe (binding_path path "probe" 0) all_strings)
			(bind_query_names_at subquery scopes (binding_path path "in" 0) all_strings))
		((quote inner_select_in) probe subquery)
		(list (quote inner_select_in)
			(bind_query_expr scopes probe (binding_path path "probe" 0) all_strings)
			(bind_query_names_at subquery scopes (binding_path path "in" 0) all_strings))
		((symbol get_column) tblvar tbl_ignorecase "*" col_ignorecase)
		(if (nil? tblvar)
			expr
			(bind_qualified_column_in_scopes scopes tblvar tbl_ignorecase "*" col_ignorecase))
		((quote get_column) tblvar tbl_ignorecase "*" col_ignorecase)
		(bind_query_expr scopes (list (quote get_column) tblvar tbl_ignorecase "*" col_ignorecase) path all_strings)
		((symbol get_column) tblvar tbl_ignorecase col col_ignorecase)
		(if (nil? tblvar)
			(bind_unqualified_column_in_scopes scopes col col_ignorecase)
			(bind_qualified_column_in_scopes scopes tblvar tbl_ignorecase col col_ignorecase))
		((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
		(bind_query_expr scopes (list (quote get_column) tblvar tbl_ignorecase col col_ignorecase) path all_strings)
		(cons head tail) (cons head (bind_query_expr_tail scopes tail path all_strings 0))
		_ expr)))

(define bind_group_expr_tail (lambda (scopes fields items path all_strings index)
	(match (coalesceNil items '())
		(cons item rest) (cons
			(bind_group_expr scopes fields item (binding_path path "expr" index) all_strings)
			(bind_group_expr_tail scopes fields rest path all_strings (+ index 1)))
		_ '())))

(define bind_group_expr (lambda (scopes fields expr path all_strings)
	(match expr
		((symbol inner_select) _subquery) (bind_query_expr scopes expr path all_strings)
		((quote inner_select) _subquery) (bind_query_expr scopes expr path all_strings)
		((symbol inner_select_exists) _subquery) (bind_query_expr scopes expr path all_strings)
		((quote inner_select_exists) _subquery) (bind_query_expr scopes expr path all_strings)
		((symbol inner_select_in) _probe _subquery) (bind_query_expr scopes expr path all_strings)
		((quote inner_select_in) _probe _subquery) (bind_query_expr scopes expr path all_strings)
		((symbol get_column) nil _tbl_ignorecase "*" _col_ignorecase) expr
		((quote get_column) nil _tbl_ignorecase "*" _col_ignorecase) expr
		((symbol get_column) nil _tbl_ignorecase col col_ignorecase) (begin
			(define input_ref (try_bind_unqualified_column_in_scopes scopes col col_ignorecase))
			(if (nil? input_ref)
				(begin
					(define output_ref (field_expr_by_title fields col col_ignorecase))
					(if (nil? output_ref)
						(neumann_fail "bind_query_names" (concat "Column does not exist: " col))
						output_ref))
				input_ref))
		((quote get_column) nil _tbl_ignorecase col col_ignorecase)
		(bind_group_expr scopes fields (list (quote get_column) nil false col col_ignorecase) path all_strings)
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(bind_query_expr scopes expr path all_strings)
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(bind_query_expr scopes expr path all_strings)
		(cons head tail) (cons head (bind_group_expr_tail scopes fields tail path all_strings 0))
		_ expr)))

(define bind_query_source_relations (lambda (sources path all_strings index)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define relation (normalize_query_ast (source_relation src)))
			(cons
				(source_with_relation src
					(if (or (query_block? relation) (union_block? relation))
						/* MariaDB FROM subqueries are non-lateral. */
						(bind_query_names_at relation '() (binding_path path "derived" index) all_strings)
						relation))
				(bind_query_source_relations rest path all_strings (+ index 1))))
		_ '())))

(define bind_query_source_joins (lambda (entries outer_scopes visible_entries path all_strings index)
	(match (coalesceNil entries '())
		(cons entry rest) (begin
			(define src (binding_entry_source entry))
			(define visible (merge (list visible_entries (list entry))))
			(cons
				(source_with_join_expr src
					(bind_query_expr (cons visible outer_scopes) (source_join_expr src)
						(binding_path path "join" index) all_strings))
				(bind_query_source_joins rest outer_scopes visible path all_strings (+ index 1))))
		_ '())))

(define binding_scope_has_alias? (lambda (entries alias)
	(reduce (coalesceNil entries '()) (lambda (found entry)
		(or found (equal? (binding_entry_name entry) alias))) false)))

(define binding_scopes_have_alias? (lambda (scopes alias)
	(reduce (coalesceNil scopes '()) (lambda (found entries)
		(or found (binding_scope_has_alias? entries alias))) false)))

(define binding_scope_entries (lambda (sources outer_scopes path all_strings index)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define sql_alias (source_alias src))
			(define internal_alias (if (binding_scopes_have_alias? outer_scopes sql_alias)
				(binding_fresh_query_alias all_strings
					(concat "__binding:" (concat (binding_path path "source" index) (concat ":" sql_alias))))
				sql_alias))
			(define internal_source (list
				internal_alias
				(source_schema src)
				(source_relation src)
				(source_outer? src)
				(source_join_expr src)))
			(cons
				(begin
					(define columns (binding_source_columns internal_source))
					(list sql_alias internal_source (binding_column_catalog columns) (or
						(query_block? (normalize_query_ast (source_relation internal_source)))
						(or
							(union_block? (normalize_query_ast (source_relation internal_source)))
							(not (empty_list? columns))))))
				(binding_scope_entries rest outer_scopes path all_strings (+ index 1))))
		_ '())))

(define duplicate_source_alias (lambda (sources)
	(match (coalesceNil sources '())
		(cons src rest) (if (reduce rest (lambda (found candidate)
			(or found (equal? (source_alias src) (source_alias candidate)))) false)
			(source_alias src)
			(duplicate_source_alias rest))
		_ nil)))

(define bind_output_alias_refs (lambda (fields expr)
	(match expr
		((symbol inner_select) _subquery) expr
		((quote inner_select) _subquery) expr
		((symbol inner_select_exists) _subquery) expr
		((quote inner_select_exists) _subquery) expr
		((symbol inner_select_in) probe subquery)
		(list (quote inner_select_in) (bind_output_alias_refs fields probe) subquery)
		((quote inner_select_in) probe subquery)
		(list (quote inner_select_in) (bind_output_alias_refs fields probe) subquery)
		((symbol get_column) nil _tbl_ignorecase col col_ignorecase)
		(coalesceNil (field_expr_by_title fields col col_ignorecase) expr)
		((quote get_column) nil _tbl_ignorecase col col_ignorecase)
		(coalesceNil (field_expr_by_title fields col col_ignorecase) expr)
		(cons head tail) (cons head (map tail (lambda (item) (bind_output_alias_refs fields item))))
		_ expr)))

(define bind_query_fields (lambda (fields scopes path all_strings index)
	(match (coalesceNil fields '())
		(cons title (cons expr rest)) (cons title (cons
			(bind_query_expr scopes expr (binding_path path "field" index) all_strings)
			(bind_query_fields rest scopes path all_strings (+ index 1))))
		_ '())))

(define bind_query_order (lambda (items scopes fields alias_first path all_strings index)
	(match (coalesceNil items '())
		(cons item rest) (cons
			(match item
				'(expr dir) (begin
					(define prepared (if alias_first (bind_output_alias_refs fields expr) expr))
					(list (bind_query_expr scopes prepared (binding_path path "order" index) all_strings) dir))
				_ item)
			(bind_query_order rest scopes fields alias_first path all_strings (+ index 1)))
		_ '())))

(define bind_union_output_expr (lambda (fields carrier_alias expr)
	(match expr
		((symbol get_column) nil _tbl_ignorecase col col_ignorecase) (begin
			(define title (binding_column_name (binding_field_titles fields) col col_ignorecase))
			(if (nil? title)
				(neumann_fail "bind_query_names" (concat "UNION ORDER BY column does not exist: " col))
				(list (quote get_column) carrier_alias false title false)))
		((quote get_column) nil _tbl_ignorecase col col_ignorecase)
		(bind_union_output_expr fields carrier_alias (list (quote get_column) nil false col col_ignorecase))
		((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(neumann_fail "bind_query_names" (concat "UNION ORDER BY cannot reference relation alias: " tblvar))
		((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(neumann_fail "bind_query_names" (concat "UNION ORDER BY cannot reference relation alias: " tblvar))
		(cons head tail) (cons head (map tail (lambda (item)
			(bind_union_output_expr fields carrier_alias item))))
		_ expr)))

(define bind_union_order (lambda (items fields carrier_alias)
	(map (coalesceNil items '()) (lambda (item)
		(match item
			'(expr dir) (begin
				(if (and (number? expr) (or (<= expr 0) (> expr (/ (count fields) 2))))
					(neumann_fail "bind_query_names" "UNION ORDER BY position is out of range")
					true)
				(list (bind_union_output_expr fields carrier_alias expr) dir))
			_ (neumann_fail "bind_query_names" "malformed UNION ORDER BY item"))))))

(define bind_query_block_names (lambda (block outer_scopes path all_strings)
	(begin
		(define relation_bound_sources (bind_query_source_relations (qb_sources block) path all_strings 0))
		(define duplicate_alias (duplicate_source_alias relation_bound_sources))
		(if (nil? duplicate_alias)
			true
			(neumann_fail "bind_query_names" (concat "duplicate relation alias: " duplicate_alias)))
		(define entries (binding_scope_entries relation_bound_sources outer_scopes path all_strings 0))
		(define sources (bind_query_source_joins entries outer_scopes '() path all_strings 0))
		(define scopes (cons entries outer_scopes))
		(define fields (bind_query_fields (qb_fields block) scopes path all_strings 0))
		(define bind_alias_first_expr (lambda (expr)
			(bind_query_expr scopes (bind_output_alias_refs fields expr)
				(binding_path path "output" 0) all_strings)))
		(make_query_block
			(qb_schema block)
			sources
			fields
			(bind_query_expr scopes (qb_where block) (binding_path path "where" 0) all_strings)
			/* MariaDB resolves GROUP BY against input columns first, while
			HAVING and ORDER BY prefer SELECT-list aliases. */
			(map (coalesceNil (qb_group block) '()) (lambda (expr)
				(bind_group_expr scopes fields expr (binding_path path "group" 0) all_strings)))
			(bind_alias_first_expr (qb_having block))
			(bind_query_order (qb_order block) scopes fields true path all_strings 0)
			(qb_limit block)
			(qb_offset block)
			(bind_query_fields (qb_hidden block) scopes (binding_path path "hidden" 0) all_strings 0)
			(qb_stages block)
			(qb_facts block)))))

(define bind_query_branches (lambda (branches outer_scopes path all_strings index)
	(match (coalesceNil branches '())
		(cons branch rest) (cons
			(bind_query_names_at branch outer_scopes (binding_path path "branch" index) all_strings)
			(bind_query_branches rest outer_scopes path all_strings (+ index 1)))
		_ '())))

(define bind_query_names_at (lambda (query outer_scopes path all_strings)
	(begin
		(define normalized (normalize_query_ast query))
		(if (query_block? normalized)
			(bind_query_block_names normalized outer_scopes path all_strings)
			(if (union_block? normalized)
				(begin
					(define branches (bind_query_branches (union_branches normalized) outer_scopes path all_strings 0))
					(define fields (match branches
						(cons branch _rest) (logical_relation_fields branch)
						_ '()))
					(make_union_block
						(union_mode normalized)
						branches
						(bind_union_order (union_order normalized) fields
							(concat "__binding:" (concat path ":union-output")))
						(union_limit normalized)
						(union_offset normalized)
						(union_facts normalized)))
				normalized)))))

(define bind_query_names (lambda (query outer_scopes)
	(bind_query_names_at query outer_scopes "query" query)))

(define derived_star_ref? (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		((quote get_column) tblvar _ "*" _) (or (nil? tblvar) (equal? tblvar alias))
		_ false)))

(define rewrite_derived_ref (lambda (alias projection expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col col_ignorecase) (if (equal? tblvar alias)
			(coalesceNil (field_expr_by_title projection col col_ignorecase) expr)
			expr)
		((quote get_column) tblvar _tbl_ignorecase col col_ignorecase) (if (equal? tblvar alias)
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

(define literal_false? (lambda (expr)
	(match expr
		false true
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
								/* Drop projected columns from derived tables that are not
								referenced by this block's own fields/where/group/having/order
								or by any outer source ON condition. The outer ON conditions
								must be included so that join-key columns of non-pruneable
								sources are never dropped. prune_unused_unique_left_joins
								handles removing sorter-style joins whose output is unused. */
								(define outer_direct_consumers (reduce
									(coalesceNil (qb_sources block) '())
									(lambda (acc src) (if (nil? (source_join_expr src))
										acc
										(cons (source_join_expr src) acc)))
									(filter
										(list (qb_fields block) (qb_where block) (qb_group block)
											(qb_having block) (qb_order block) (qb_hidden block))
										(lambda (x) (not (nil? x))))))
								(define prepruned_sources (prune_unreferenced_derived_fields
									(qb_sources block) outer_direct_consumers))
								(define flattened_sources (flatten_source_list prepruned_sources child_ctx))
								(define flattened_source_list (nth flattened_sources 0))
								(define rewrites (nth flattened_sources 1))
								(define source_where_terms (nth flattened_sources 2))
								(define source_stages (nth flattened_sources 3))
								/* Derived references are already bound. Rewrite them once, then
								prune unused row-preserving lookups before their join expressions
								can create decorrelation stages. */
								(define rewritten_where (combine_where_terms source_where_terms
									(rewrite_derived_ref_chain rewrites (qb_where block))))
								(define rewritten_fields (rewrite_derived_fields_chain rewrites (qb_fields block)))
								(define rewritten_group (map (coalesceNil (qb_group block) '()) (lambda (item)
									(rewrite_derived_ref_chain rewrites item))))
								(define rewritten_having (rewrite_derived_ref_chain rewrites (qb_having block)))
								(define rewritten_order (rewrite_derived_order_chain rewrites (qb_order block)))
								(define rewritten_hidden (rewrite_derived_fields_chain rewrites (qb_hidden block)))
								(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
									(if (empty_list? flattened_source_list) nil (source_alias (car flattened_source_list)))))
								(define sources (prune_unused_unique_left_joins flattened_source_list default_alias
									(list rewritten_fields rewritten_where rewritten_group rewritten_having rewritten_order rewritten_hidden)))
								(define inherited_outer_sources (uctx_get child_ctx (quote outer-sources) '()))
								(define inherited_outer_resolution_sources (uctx_get child_ctx (quote outer-resolution-sources) '()))
								(define nested_outer_resolution_sources (merge (list inherited_outer_resolution_sources (qb_sources block))))
								(define expr_outer_sources (merge (list inherited_outer_sources sources)))
								(define expr_ctx (make_uctx child_ctx (list
									(list (quote outer-sources) expr_outer_sources)
									(list (quote outer-resolution-sources) nested_outer_resolution_sources)
									(list (quote local-sources) sources))))
								(define source_join_result (untangle_source_join_exprs_with_stages sources expr_outer_sources expr_ctx))
								(define untangled_sources (nth source_join_result 0))
								(define source_join_stage_sources (nth source_join_result 2))
								(define joined_expr_outer_sources (merge (list inherited_outer_sources untangled_sources source_join_stage_sources)))
								(define joined_expr_ctx (make_uctx child_ctx (list
									(list (quote outer-sources) joined_expr_outer_sources)
									(list (quote outer-resolution-sources) nested_outer_resolution_sources)
									(list (quote local-sources) (merge_unique (list untangled_sources source_join_stage_sources))))))
								/* SQL name ownership ends in bind_query_names. Derived flattening
								only rewrites references carrying an explicit bound alias. */
								(if (expr_contains_window? rewritten_where)
									(neumann_fail "untangle_query" "window function is not allowed in WHERE")
									true)
								(define where_result (untangle_where_with_stages rewritten_where joined_expr_outer_sources joined_expr_ctx))
								(define field_result (untangle_fields_with_stages
									rewritten_fields
									joined_expr_outer_sources joined_expr_ctx))
								(define having_result (untangle_expr_with_stages
									rewritten_having
									joined_expr_outer_sources joined_expr_ctx))
								(define stage_sources (merge_unique (list (nth where_result 2) (nth field_result 2) (nth having_result 2))))
								(define group_result (untangle_expr_list_with_stages
									rewritten_group
									joined_expr_outer_sources
									joined_expr_ctx))
								(define order_result (untangle_order_with_stages
									rewritten_order
									joined_expr_outer_sources
									joined_expr_ctx))
								(define hidden_result (untangle_fields_with_stages
									rewritten_hidden
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
								(btw2025_decorrelate_query_block delayed_block
									(make_uctx child_ctx (list
										(list (quote outer-resolution-sources) nested_outer_resolution_sources)))))))))))))

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
		(define bound_query (bind_query_names query '()))
		(define root (require_unnested_node "untangle_query" (untangle_query bound_query ctx)))
		(define ir (make_ir (if (union_block? root) (quote union) (quote select))
			root
			(if (query_block? root) (qb_stages root) '())
			(make_uctx ctx (list
				(list (quote compile-budget-ms) 1000)
				(list (quote operator-model) (quote combined))))
			(quote rows)))
		(require_flat_stage_dependencies "untangle_query" (normalize_stage_dependencies ir)))))

/* ------------------------------------------------------------------------- */
