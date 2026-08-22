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

(define query_block_base_aliases (lambda (block)
	(map
		(filter (qb_sources block) (lambda (src) (not (source_is_stage_output? src))))
		source_alias)))

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

(define btw2025_decorrelate_fields_with_stages (lambda (fields ctx)
	(btw2025_decorrelate_fields_with_stages_using fields ctx '())))

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

(define planner_estimate (lambda (value confidence source known)
	(list
		(list (quote value) value)
		(list (quote confidence) confidence)
		(list (quote source) source)
		(list (quote known) known))))

(define planner_estimate_planning_value (lambda (estimate fallback)
	(coalesceNil (qassoc_get estimate (quote value) nil) fallback)))

(define planner_unknown_selectivity_estimate (lambda ()
	(planner_estimate nil 0 (quote unknown) false)))

(define join_optimizer_column_selectivity_estimate (lambda (column_ref)
	(if (nil? column_ref)
		(planner_unknown_selectivity_estimate)
		(begin
			(define stats (planner_column_statistics (car column_ref) (cadr column_ref)))
			(define distinct (qassoc_get stats (quote distinct) nil))
			(if (or (nil? distinct) (<= distinct 0))
				(planner_unknown_selectivity_estimate)
				(planner_estimate (/ 1 (max 1 distinct))
					(qassoc_get stats (quote distinct_confidence) 0)
					(qassoc_get stats (quote distinct_source) (quote unknown)) true))))))

(define join_optimizer_equality_selectivity_estimate (lambda (sources default_alias left right)
	(begin
		(define left_ref (join_optimizer_column_ref sources default_alias left))
		(define right_ref (join_optimizer_column_ref sources default_alias right))
		(if (and (not (nil? left_ref)) (not (nil? right_ref)))
			(begin
				(define left_stats (planner_column_statistics (car left_ref) (cadr left_ref)))
				(define right_stats (planner_column_statistics (car right_ref) (cadr right_ref)))
				(define left_distinct (qassoc_get left_stats (quote distinct) nil))
				(define right_distinct (qassoc_get right_stats (quote distinct) nil))
				(if (or (nil? left_distinct) (nil? right_distinct))
					(planner_unknown_selectivity_estimate)
					(planner_estimate (/ 1 (max 1 (max left_distinct right_distinct)))
						(min (qassoc_get left_stats (quote distinct_confidence) 0)
							(qassoc_get right_stats (quote distinct_confidence) 0))
						(quote distinct_join) true)))
			(if (not (nil? left_ref))
				(join_optimizer_column_selectivity_estimate left_ref)
				(if (not (nil? right_ref))
					(join_optimizer_column_selectivity_estimate right_ref)
					(planner_unknown_selectivity_estimate)))))))

(define join_optimizer_expr_selectivity_estimate (lambda (sources default_alias expr)
	(match expr
		((symbol equal?) left right) (join_optimizer_equality_selectivity_estimate sources default_alias left right)
		((quote equal?) left right) (join_optimizer_expr_selectivity_estimate sources default_alias (list (symbol "equal?") left right))
		((symbol equal??) left right) (join_optimizer_equality_selectivity_estimate sources default_alias left right)
		((quote equal??) left right) (join_optimizer_expr_selectivity_estimate sources default_alias (list (symbol "equal??") left right))
		((symbol =) left right) (join_optimizer_equality_selectivity_estimate sources default_alias left right)
		((quote =) left right) (join_optimizer_expr_selectivity_estimate sources default_alias (list (symbol "=") left right))
		((symbol <) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((quote <) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((symbol <=) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((quote <=) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((symbol >) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((quote >) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((symbol >=) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		((quote >=) _left _right) (planner_estimate nil 0 (quote range_unknown) false)
		_ (planner_unknown_selectivity_estimate))))

(define join_optimizer_expr_selectivity (lambda (sources default_alias expr)
	(begin
		(define estimate (join_optimizer_expr_selectivity_estimate sources default_alias expr))
		(planner_estimate_planning_value estimate
			(if (equal? (qassoc_get estimate (quote source) nil) (quote range_unknown))
				0.3333333333333333 0.1)))))

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

(define join_optimizer_source_rows (lambda (stages sources default_alias graph src)
	(begin
		(define base_rows (planner_estimate_planning_value
			(planner_source_row_estimate_using_stages stages src) 1000000))
		(define local_selectivity (join_optimizer_product
			(map (join_optimizer_local_predicates graph (source_alias src)) (lambda (entry)
				(join_optimizer_expr_selectivity sources default_alias
					(qassoc_get entry (quote predicate) true))))))
		(max 1 (* base_rows local_selectivity)))))

(define planner_quoted_value (lambda (value)
	(list (quote quote) value)))

(define join_optimizer_source_rows_expr (lambda (stages sources default_alias graph src)
	(planner_guard_runtime_binding
		(list (quote join_optimizer_source_rows)
			(planner_quoted_value stages)
			(planner_quoted_value sources)
			default_alias
			(planner_quoted_value graph)
			(planner_quoted_value src)))))

(define join_optimizer_selectivity_expr (lambda (sources default_alias predicate)
	(planner_guard_runtime_binding
		(list (quote join_optimizer_expr_selectivity)
			(planner_quoted_value sources)
			default_alias
			(planner_quoted_value predicate)))))

(define join_optimizer_alias_subset? (lambda (required available)
	(reduce (coalesceNil required '()) (lambda (ok alias)
		(and ok (contains? available alias))) true)))

(define join_optimizer_metadata_nodes (lambda (stages sources default_alias graph)
	(map sources (lambda (src)
		(list
			(source_alias src)
			(join_optimizer_source_rows stages sources default_alias graph src)
			(join_optimizer_source_rows_expr stages sources default_alias graph src))))))

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
				(define predicate (qassoc_get entry (quote predicate) true))
				(define metadata (list
					predicate_aliases
					(join_optimizer_expr_selectivity sources default_alias
						predicate)
					origin
					(qassoc_get entry (quote owner) nil)
					predicate
					barrier_owner
					(join_optimizer_selectivity_expr sources default_alias predicate)))
				metadata)))))

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
(define planner_cost (lambda (startup_ns row_ns probe_ns batch_startup_ns batch_row_ns
	build_ns memory_bytes compile_ns expected_rows confidence)
	(begin
		(define execution_ns (+ startup_ns row_ns probe_ns batch_startup_ns batch_row_ns build_ns))
		(define total_ns (+ execution_ns compile_ns))
		(list
			(list (quote startup_ns) startup_ns)
			(list (quote row_ns) row_ns)
			(list (quote probe_ns) probe_ns)
			(list (quote batch_startup_ns) batch_startup_ns)
			(list (quote batch_row_ns) batch_row_ns)
			(list (quote build_ns) build_ns)
			(list (quote memory_bytes) memory_bytes)
			(list (quote compile_ns) compile_ns)
			(list (quote expected_rows) expected_rows)
			(list (quote confidence) confidence)
			(list (quote execution_ns) execution_ns)
			(list (quote total_ns) total_ns)))))

(define planner_zero_cost (lambda (expected_rows confidence)
	(planner_cost 0 0 0 0 0 0 0 0 expected_rows confidence)))

(define planner_cost_add (lambda (left right expected_rows confidence)
	(planner_cost
		(+ (qassoc_get left (quote startup_ns) 0) (qassoc_get right (quote startup_ns) 0))
		(+ (qassoc_get left (quote row_ns) 0) (qassoc_get right (quote row_ns) 0))
		(+ (qassoc_get left (quote probe_ns) 0) (qassoc_get right (quote probe_ns) 0))
		(+ (qassoc_get left (quote batch_startup_ns) 0) (qassoc_get right (quote batch_startup_ns) 0))
		(+ (qassoc_get left (quote batch_row_ns) 0) (qassoc_get right (quote batch_row_ns) 0))
		(+ (qassoc_get left (quote build_ns) 0) (qassoc_get right (quote build_ns) 0))
		(+ (qassoc_get left (quote memory_bytes) 0) (qassoc_get right (quote memory_bytes) 0))
		(+ (qassoc_get left (quote compile_ns) 0) (qassoc_get right (quote compile_ns) 0))
		expected_rows confidence)))

(define planner_cost_better? (lambda (candidate current)
	(begin
		(define candidate_total (qassoc_get candidate (quote total_ns) 0))
		(define current_total (qassoc_get current (quote total_ns) 0))
		(or (< candidate_total current_total)
			(and (equal? candidate_total current_total)
				(< (qassoc_get candidate (quote memory_bytes) 0)
					(qassoc_get current (quote memory_bytes) 0)))))))

/* Calibrated generic storage facts. They are deliberately free of SQL
semantics; SCM combines them according to the candidate being evaluated. */
(define planner_scan_cost (lambda (rows confidence)
	(planner_cost 4000 (* rows 54) 0 0 0 0 0 0 rows confidence)))

(define planner_join_work_cost (lambda (rows confidence)
	(planner_cost 0 0 (* rows 1240) 0 0 0 0 0 rows confidence)))

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
(define join_order_pred_selectivity_expr (lambda (predicate)
	(if (> (count predicate) 6) (nth predicate 6)
		(join_order_pred_selectivity predicate))))

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

/* The final three fields mirror the numeric cost values as cheap runtime ASTs.
They let the compile-local condition accumulator retain the exact inequalities
which selected a cached join plan without rerunning join enumeration. */
/* plan = (tree aliases cardinality cost size atomic driver-cardinality left right cardinality-expr cost-expr driver-expr cost-domain) */
(define join_order_plan_tree (lambda (plan) (nth plan 0)))
(define join_order_plan_aliases (lambda (plan) (nth plan 1)))
(define join_order_plan_cardinality (lambda (plan) (nth plan 2)))
(define join_order_plan_cost (lambda (plan) (nth plan 3)))
(define join_order_plan_size (lambda (plan) (nth plan 4)))
(define join_order_plan_atomic? (lambda (plan) (nth plan 5)))
(define join_order_driver_cardinality (lambda (plan) (nth plan 6)))
(define join_order_plan_left (lambda (plan) (nth plan 7)))
(define join_order_plan_right (lambda (plan) (nth plan 8)))
(define join_order_plan_cardinality_expr (lambda (plan)
	(if (> (count plan) 9) (nth plan 9) (join_order_plan_cardinality plan))))
(define join_order_plan_cost_expr (lambda (plan)
	(if (> (count plan) 10) (nth plan 10) (join_order_plan_cost plan))))
(define join_order_plan_driver_expr (lambda (plan)
	(if (> (count plan) 11) (nth plan 11) (join_order_driver_cardinality plan))))
(define join_order_plan_cost_domain (lambda (plan)
	(if (> (count plan) 12) (nth plan 12)
		(planner_zero_cost (join_order_plan_cardinality plan) 0.5))))

(define planner_scan_cost_expr (lambda (rows)
	(list (quote +) 4000 (list (quote *) rows 54))))

(define join_order_leaf_plan (lambda (node)
	(begin
		(define row_expr (if (> (count node) 2) (nth node 2) (cadr node)))
		(define rows (max 1 (cadr node)))
		(list
			(list (quote join-leaf) (car node) '())
			(list (car node))
			rows
			(qassoc_get (planner_scan_cost rows 0.5) (quote total_ns) 0)
			1
			false
			rows
			nil nil
			(list (quote max) 1 row_expr)
			(planner_scan_cost_expr (list (quote max) 1 row_expr))
			(list (quote max) 1 row_expr)
			(planner_scan_cost rows 0.5)))))

(define join_order_product_expr (lambda (items)
	(match items
		(cons item '()) item
		(cons _head _tail) (cons (quote *) items)
		_ 1)))

(define join_order_cap_cardinality (lambda (value)
	(max 1 (min 1e300 value))))

(define join_order_join_cardinality (lambda (predicates left right)
	(begin
		(define combined (merge_unique (list
			(join_order_plan_aliases left) (join_order_plan_aliases right))))
		(define selectivity (reduce predicates (lambda (value predicate)
			(if (join_order_predicate_crosses_in? predicate
				(join_order_plan_aliases left) (join_order_plan_aliases right) combined)
				(* value (join_order_pred_selectivity predicate))
				value)) 1))
		(join_order_cap_cardinality (*
			(join_order_plan_cardinality left)
			(join_order_plan_cardinality right)
			selectivity)))))

(define join_order_join_cardinality_expr (lambda (predicates left right)
	(begin
		(define combined (merge_unique (list
			(join_order_plan_aliases left) (join_order_plan_aliases right))))
		(define selectivities (map (filter predicates (lambda (predicate)
			(join_order_predicate_crosses_in? predicate
				(join_order_plan_aliases left) (join_order_plan_aliases right) combined)))
			join_order_pred_selectivity_expr))
		(list (quote join_order_cap_cardinality)
			(list (quote *)
				(join_order_plan_cardinality_expr left)
				(join_order_plan_cardinality_expr right)
				(join_order_product_expr selectivities))))))

(define join_order_join_plan (lambda (universe predicates left right)
	(begin
		(define cardinality (join_order_join_cardinality predicates left right))
		(define cardinality_expr (join_order_join_cardinality_expr predicates left right))
		(define combined (join_order_set_union universe
			(join_order_plan_aliases left) (join_order_plan_aliases right)))
		(define children_cost (planner_cost_add
			(join_order_plan_cost_domain left) (join_order_plan_cost_domain right)
			cardinality 0.5))
		(define cost_domain (planner_cost_add children_cost
			(planner_join_work_cost cardinality 0.5) cardinality 0.5))
		(list
			(list (quote join-node) (quote inner) (join_order_plan_tree left) (join_order_plan_tree right) '())
			combined
			cardinality
			(qassoc_get cost_domain (quote total_ns) 0)
			(+ (join_order_plan_size left) (join_order_plan_size right))
			false
			(join_order_driver_cardinality left)
			left right
			cardinality_expr
			(list (quote +)
				(join_order_plan_cost_expr left)
				(join_order_plan_cost_expr right)
				(list (quote *) cardinality_expr 1240))
			(join_order_plan_driver_expr left)
			cost_domain))))

(define join_order_candidate_better? (lambda (current candidate)
	(or (< (join_order_plan_cost candidate) (join_order_plan_cost current))
		(and (equal? (join_order_plan_cost candidate) (join_order_plan_cost current))
			(< (join_order_driver_cardinality candidate) (join_order_driver_cardinality current))))))

(define join_order_candidate_better_expr (lambda (current candidate)
	(list (quote or)
		(list (quote <)
			(join_order_plan_cost_expr candidate)
			(join_order_plan_cost_expr current))
		(list (quote and)
			(list (quote equal?)
				(join_order_plan_cost_expr candidate)
				(join_order_plan_cost_expr current))
			(list (quote <)
				(join_order_plan_driver_expr candidate)
				(join_order_plan_driver_expr current))))))

(define join_order_better_plan (lambda (current candidate)
	(if (nil? current)
		candidate
		(if (planner_guarded_choice
			(join_order_candidate_better? current candidate)
			(join_order_candidate_better_expr current candidate))
			candidate current))))

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

(define join_order_dphyp_connected (lambda (nodes aliases predicates connected)
	(begin
		(define sorted_connected (join_order_sort_sets connected))
		(define connected_by_first (reduce connected (lambda (dict subset)
			(begin
				(define first (car subset))
				(set_assoc dict first (append (get_assoc dict first '()) subset)))) '()))
		(join_order_dphyp_fill nodes aliases predicates connected_by_first sorted_connected '() 0))))

(define join_order_dphyp_budgeted (lambda (nodes aliases predicates budget)
	(begin
		(define connected (join_order_enumerate_connected aliases predicates budget))
		(if (cadr connected)
			(list nil 0)
			(join_order_dphyp_connected nodes aliases predicates (car connected))))))

(define join_order_dphyp (lambda (nodes aliases predicates)
	(join_order_dphyp_budgeted nodes aliases predicates (join_order_dp_state_budget))))

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
		(join_order_plan_right plan)
		(join_order_plan_cardinality_expr plan)
		(join_order_plan_cost_expr plan)
		(join_order_plan_driver_expr plan)
		(join_order_plan_cost_domain plan))))

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
						(join_order_dphyp_budgeted nodes (join_order_plan_aliases target)
							predicates (join_order_dp_state_budget))
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
	(begin
		/* Compile work is reported separately from execution work but participates
		in the displayed total. The state calibration is intentionally generic and
		does not affect execution-plan comparison. */
		(define cost_components (planner_cost_add
			(join_order_plan_cost_domain plan)
			(planner_cost 0 0 0 0 0 0 0 (* entries 50000) 0 0.5)
			(join_order_plan_cardinality plan) 0.5))
		(list
			(list (quote strategy) strategy)
			(list (quote tree) (join_order_tree_with_predicates (join_order_plan_tree plan) predicates))
			(list (quote order) (join_order_plan_aliases plan))
			(list (quote cost) (join_order_plan_cost plan))
			(list (quote cardinality) (join_order_plan_cardinality plan))
			(list (quote cost_components) cost_components)
			(list (quote dp_entries) entries)))))

(define join_order_choose_strategy (lambda (alias_count hypergraph connected_over_budget)
	(if (and (<= alias_count 100) (not connected_over_budget))
		(quote dphyp)
		(if hypergraph
			(quote goo-dphyp)
			(if (<= alias_count 100)
				(quote linearized-dp)
				(quote goo-linearized-dp))))))

(define join_order_record_exact_cost_inputs (lambda (nodes predicates)
	(begin
		(reduce nodes (lambda (_ node)
			(planner_record_guard_condition
				(list (quote equal?)
					(if (> (count node) 2) (nth node 2) (cadr node))
					(cadr node)))) nil)
		(reduce predicates (lambda (_ predicate)
			(planner_record_guard_condition
				(list (quote equal?)
					(join_order_pred_selectivity_expr predicate)
					(join_order_pred_selectivity predicate)))) nil))))

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

/* Connected subsets are the dominant retained state of DPHyp. Keeping this
budget below the measured compile-time cliff makes the exact search predictable;
the fallback still evaluates the same execution-cost function. */
(define join_order_dp_state_budget (lambda ()
	(begin
		(define configured (settings "JoinReorderDPBudget"))
		(if (> configured 0) configured 256))))

(define join_order_adaptive (lambda (nodes raw_predicates)
	(begin
		(define aliases (map nodes car))
		(define hypergraph (join_order_hypergraph? raw_predicates))
		(define predicates (join_order_prepare_predicates aliases raw_predicates))
		(define regular_edges (join_order_regular_edges aliases predicates))
		(define state_budget (join_order_dp_state_budget))
		(define connected_count (if (<= (count aliases) 100)
			(if (join_order_degree_proves_budget_overflow? aliases regular_edges state_budget)
				(list '() true)
				(join_order_enumerate_connected aliases predicates state_budget))
			(list '() true)))
		(define strategy (join_order_choose_strategy
			(count aliases) hypergraph (cadr connected_count)))
		(define exact (equal? strategy (quote dphyp)))
		(if exact nil (join_order_record_exact_cost_inputs nodes predicates))
		(define result (if exact
			(join_order_dphyp_connected nodes aliases predicates (car connected_count))
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

(define join_optimizer_tree_without_aliases_with_aliases (lambda (tree removed_aliases)
	(match tree
		((symbol join-leaf) alias) (if (contains? removed_aliases alias) (list nil '()) (list tree (list alias)))
		((quote join-leaf) alias) (if (contains? removed_aliases alias) (list nil '()) (list tree (list alias)))
		((symbol join-leaf) alias _predicates) (if (contains? removed_aliases alias) (list nil '()) (list tree (list alias)))
		((quote join-leaf) alias _predicates) (if (contains? removed_aliases alias) (list nil '()) (list tree (list alias)))
		((symbol join-node) kind left right predicates)
		(begin
			(define left_result (join_optimizer_tree_without_aliases_with_aliases left removed_aliases))
			(define right_result (join_optimizer_tree_without_aliases_with_aliases right removed_aliases))
			(define kept_left (car left_result))
			(define kept_right (car right_result))
			(if (nil? kept_left) right_result
				(if (nil? kept_right) left_result
					(begin
						(define kept_aliases (merge (list (cadr left_result) (cadr right_result))))
						(list
							(make_join_optimizer_node kind kept_left kept_right
								(filter predicates (lambda (predicate)
									(join_optimizer_alias_subset?
										(join_order_pred_aliases predicate) kept_aliases))))
							kept_aliases)))))
		((quote join-node) kind left right predicates)
		(join_optimizer_tree_without_aliases_with_aliases
			(make_join_optimizer_node kind left right predicates) removed_aliases)
		_ (neumann_fail "build_queryplan" "malformed logical join plan while removing consumed sources"))))

(define join_optimizer_tree_without_aliases (lambda (tree removed_aliases)
	(car (join_optimizer_tree_without_aliases_with_aliases tree removed_aliases))))

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

(define physical_membership_requirement_stage (lambda (stage facts)
	(if (not (group_stage? stage))
		stage
		(make_group_stage
			(gs_id stage) (gs_input stage) (gs_domain stage) (gs_keys stage)
			(gs_aggregates stage) (gs_having stage) (gs_output stage) (gs_order stage)
			(gs_limit stage) (gs_offset stage)
			(merge (list
				(list
					(list (quote membership_stage_id)
						(qassoc_get facts (quote membership_stage_id) (gs_id stage)))
					(list (quote physical_membership_requirement) true)
					(list (quote membership_consumer) (qassoc_get facts (quote membership_consumer) (quote filter)))
					(list (quote membership_candidate_input_rows) (qassoc_get facts (quote membership_candidate_input_rows) nil))
					(list (quote membership_candidate_estimated_rows) (qassoc_get facts (quote membership_candidate_estimated_rows) nil))
					(list (quote membership_candidate_estimate_capped) (qassoc_get facts (quote membership_candidate_estimate_capped) false))
					(list (quote membership_candidate_estimate_input) (qassoc_get facts (quote membership_candidate_estimate_input) nil))
					(list (quote membership_candidate_estimate_sampled) (qassoc_get facts (quote membership_candidate_estimate_sampled) nil))
					(list (quote membership_candidate_estimate_population) (qassoc_get facts (quote membership_candidate_estimate_population) (quote table_rows)))
					(list (quote membership_candidate_estimate_coverage) (qassoc_get facts (quote membership_candidate_estimate_coverage) (quote sampled)))
					(list (quote membership_candidate_scan_invocations) (qassoc_get facts (quote membership_candidate_scan_invocations) 1))
					(list (quote membership_candidate_filter_columns) (qassoc_get facts (quote membership_candidate_filter_columns) 0))
					(list (quote membership_candidate_map_columns) (qassoc_get facts (quote membership_candidate_map_columns) 1))
					(list (quote membership_candidate_cache_map_columns) (qassoc_get facts (quote membership_candidate_cache_map_columns) 2))
					(list (quote membership_candidate_expression_operations) (qassoc_get facts (quote membership_candidate_expression_operations) 0))
					(list (quote membership_candidate_expression_depth) (qassoc_get facts (quote membership_candidate_expression_depth) 0))
					(list (quote membership_candidate_broad_text_match_rows) (qassoc_get facts (quote membership_candidate_broad_text_match_rows) 0))
					(list (quote membership_candidate_broad_text_match_bytes) (qassoc_get facts (quote membership_candidate_broad_text_match_bytes) 0))
					(list (quote membership_candidate_filter_value_rows) (qassoc_get facts (quote membership_candidate_filter_value_rows) nil))
					(list (quote membership_candidate_expression_operation_rows) (qassoc_get facts (quote membership_candidate_expression_operation_rows) nil))
					(list (quote membership_driver_rows) (qassoc_get facts (quote membership_driver_rows) nil))
					(list (quote membership_driver_input_rows) (qassoc_get facts (quote membership_driver_input_rows) nil))
					(list (quote membership_driver_scan_invocations) (qassoc_get facts (quote membership_driver_scan_invocations) 1))
					(list (quote membership_driver_filter_columns) (qassoc_get facts (quote membership_driver_filter_columns) 0))
					(list (quote membership_driver_map_columns) (qassoc_get facts (quote membership_driver_map_columns) 0))
					(list (quote membership_driver_expression_operations) (qassoc_get facts (quote membership_driver_expression_operations) 0))
					(list (quote membership_driver_expression_depth) (qassoc_get facts (quote membership_driver_expression_depth) 0))
					(list (quote membership_selectivity_class) (qassoc_get facts (quote membership_selectivity_class) (quote unknown)))
					(list (quote membership_driver_alternative) (qassoc_get facts (quote membership_driver_alternative) false))
					(list (quote membership_order_limit_driver) (qassoc_get facts (quote membership_order_limit_driver) false))
					(list (quote membership_order_limit) (qassoc_get facts (quote membership_order_limit) nil))
					(list (quote membership_order_offset) (qassoc_get facts (quote membership_order_offset) 0))
					(list (quote reuse) (qassoc_get facts (quote reuse) 1)))
				(gs_facts stage)))))))

(define physicalize_membership_requirement_expr_using (lambda (expr facts)
	(match expr
		((symbol membership_requirement_probe) stage probe)
		(list (quote driver_membership_probe)
			(physical_membership_requirement_stage stage facts) probe)
		((quote membership_requirement_probe) stage probe)
		(list (quote driver_membership_probe)
			(physical_membership_requirement_stage stage facts) probe)
		(cons head tail) (cons
			(physicalize_membership_requirement_expr_using head facts)
			(map tail (lambda (item)
				(physicalize_membership_requirement_expr_using item facts))))
		_ expr)))

(define physicalize_membership_requirement_expr (lambda (expr)
	(physicalize_membership_requirement_expr_using expr '())))

(define physicalize_membership_requirement_source_using (lambda (src facts)
	(source_with_join_expr src
		(physicalize_membership_requirement_expr_using (source_join_expr src) facts))))

(define apply_join_optimizer_plan_node (lambda (node)
	(if (query_block? node)
		(begin
			(define planned (apply_join_optimizer_plan node))
			(define physical_planned (query_block_with_physical_requirement_choices planned))
			(define physical_consumer (if (query_block_has_aggregates? physical_planned)
				(quote aggregate)
				(if (query_limit_active? (qb_offset physical_planned) (qb_limit physical_planned))
					(quote order_limit)
					(quote filter))))
			(define local_driver_rows (if (equal? physical_consumer (quote order_limit))
				(probe_limit_work_rows (qb_limit physical_planned)) nil))
			(define physical_facts (merge (list
				(list
					(list (quote membership_consumer) physical_consumer)
					(list (quote membership_driver_rows) (coalesceNil local_driver_rows
						(qassoc_get (qb_facts physical_planned) (quote membership_driver_rows) nil))))
				(qb_facts physical_planned))))
			(make_query_block
				(qb_schema physical_planned)
				(map (qb_sources physical_planned) (lambda (src)
					(physicalize_membership_requirement_source_using src physical_facts)))
				(physicalize_membership_requirement_expr_using (qb_fields physical_planned) physical_facts)
				(physicalize_membership_requirement_expr_using (qb_where physical_planned) physical_facts)
				(physicalize_membership_requirement_expr_using (qb_group physical_planned) physical_facts)
				(physicalize_membership_requirement_expr_using (qb_having physical_planned) physical_facts)
				(physicalize_membership_requirement_expr_using (qb_order physical_planned) physical_facts)
				(qb_limit physical_planned)
				(qb_offset physical_planned)
				(physicalize_membership_requirement_expr_using (qb_hidden physical_planned) physical_facts)
				(map (qb_stages physical_planned) apply_join_optimizer_plan_stage)
				(physicalize_membership_requirement_expr_using physical_facts physical_facts)))
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
			(physicalize_membership_requirement_expr (gs_domain stage))
			(physicalize_membership_requirement_expr (gs_keys stage))
			(physicalize_membership_requirement_expr (gs_aggregates stage))
			(physicalize_membership_requirement_expr (gs_having stage))
			(physicalize_membership_requirement_expr (gs_output stage))
			(physicalize_membership_requirement_expr (gs_order stage))
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
		(define key_sets (source_unique_key_sets lookup))
		(reduce key_sets (lambda (unique key_cols)
			(or unique
				(and (not (empty_list? key_cols))
					(reduce key_cols (lambda (complete key_col)
						(and complete (join_optimizer_key_bound_to_driver?
							sources default_alias graph driver lookup key_col)))
						true))))
			false))))

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

(define join_optimizer_reorder_result (lambda (tree strategy dp_entries cost cardinality cost_components)
	(list tree strategy dp_entries cost cardinality cost_components)))

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
				0 nil nil nil)
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
					(qassoc_get planned (quote dp_entries) 0)
					(qassoc_get planned (quote cost) nil)
					(qassoc_get planned (quote cardinality) nil)
					(qassoc_get planned (quote cost_components) nil)))))))

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
		(list (quote join_estimated_cost) (nth reordered 3))
		(list (quote join_estimated_rows) (nth reordered 4))
		(list (quote join_cost) (nth reordered 5))
		(list (quote join_dp_state_budget) (join_order_dp_state_budget))
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
			(lambda () (begin
				(define compile_bindings ((context "session") "__memcp_queryplan_compile_bindings"))
				(define observed ((context "session") "__memcp_queryplan_observed_session_keys"))
				(if (and (not (nil? compile_bindings)) (not (nil? observed)))
					(observed key true)
					nil)
				(define direct ((context "session") key))
				(if (not (nil? direct))
					direct
					(if (nil? compile_bindings) nil (compile_bindings key)))))
			(lambda (_e) nil))
		((quote session) key) (planner_literal_value (list (quote session) key))
		_ expr)))

(define planner_concat_expr_value (lambda (items)
	(match items
		(cons item rest) (begin
			(define value (planner_string_expr_value item))
			(define tail (planner_concat_expr_value rest))
			(if (and (string? value) (string? tail)) (concat value tail) nil))
		_ "")))

(define planner_string_expr_value (lambda (expr)
	(begin
		(define value (planner_literal_value expr))
		(if (string? value)
			value
			(match expr
				(cons head args) (if (or (equal? head (symbol "concat")) (equal? head (quote concat)))
					(planner_concat_expr_value args) nil)
				_ nil)))))

(define like_pattern_core (lambda (pattern)
	(replace (replace (coalesceNil pattern "") "%" "") "_" "")))

(define broad_like_pattern? (lambda (pattern)
	(if (not (string? pattern))
		false
		(begin
			(define core (like_pattern_core pattern))
			(<= (strlen core) 2)))))

(define planner_broad_like_expr? (lambda (pattern)
	(begin
		(define broad (broad_like_pattern? (planner_string_expr_value pattern)))
		(if (expr_contains_session_dependency? pattern)
			(planner_guarded_choice broad
				(list (quote broad_like_pattern?) pattern))
			broad))))

(define expr_contains_broad_text_match? (lambda (expr)
	(match expr
		((symbol strlike) _value pattern _collation)
		(planner_broad_like_expr? pattern)
		((quote strlike) _value pattern _collation)
		(expr_contains_broad_text_match? (list (quote strlike) _value pattern _collation))
		(cons head tail)
		(reduce tail (lambda (found item)
			(or found (expr_contains_broad_text_match? item)))
			(expr_contains_broad_text_match? head))
		_ false)))

(define expr_contains_text_match? (lambda (expr)
	(match expr
		((symbol strlike) _value _pattern _collation) true
		((quote strlike) _value _pattern _collation) true
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

(define planner_table_statistics (lambda (schema relation)
	(try
		(lambda ()
			(begin
				(define cache (session "__memcp_queryplan_statistics"))
				(define key (concat schema "." relation))
				(define cached (if (nil? cache) nil (cache key)))
				(if (nil? cached)
					(begin
						(define snapshot (table_planner_statistics (table schema relation)))
						(if (nil? cache) nil (cache key snapshot))
						snapshot)
					cached)))
		(lambda (_e) nil))))

(define planner_table_row_count (lambda (schema relation)
	(begin
		(define stats (planner_table_statistics schema relation))
		(if (nil? stats) nil (stats "row_count")))))

(define planner_source_filter_estimate (lambda (src condition max_rows)
	(if (not (source_is_base_table? src))
		nil
		(try
			(lambda ()
				(begin
					(define alias (source_alias src))
					(define filtercols (extract_columns_for_alias src condition))
					/* This callback is evaluated during costing, before the enclosing
					physical plan reaches its one recursive optimization pass. Compile it
					here; callbacks emitted into the final plan must remain unwrapped. */
					(define filter_expr (optimize (list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(lower_column_expr_for_alias src condition))))
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

/* A driving source's own row count, scaled by how selective the residual
condition is against it. Used wherever a nested probe's call count needs a
real estimate instead of an unconditional "unknown": an unbounded scan still
has a knowable multiplicity (its source's row count), it just isn't capped
by a LIMIT or a unique point lookup. Falls back to `fallback` only when the
source itself has no known row count (not a base table). */
(define planner_row_count_after_selectivity (lambda (src sources default_alias condition fallback)
	(begin
		(define source_rows (planner_source_row_count src))
		(if (number? source_rows)
			(max 1 (* source_rows (join_optimizer_expr_selectivity sources default_alias condition)))
			fallback))))

(define planner_source_row_estimate (lambda (src)
	(begin
		(define rows (planner_source_row_count src))
		(if (nil? rows)
			(planner_estimate nil 0 (quote unknown) false)
			(planner_estimate rows 0.9 (quote live_or_rebuild_row_count) true)))))

(define planner_source_row_estimate_using_stages (lambda (stages src)
	(if (join_optimizer_guaranteed_singleton_stage_source? stages src)
		(planner_estimate 1 1 (quote semantic_singleton) true)
		(planner_source_row_estimate src))))

(define planner_column_statistics (lambda (src column)
	(if (or (not (source_is_base_table? src)) (nil? column))
		(list
			(list (quote known) false)
			(list (quote confidence) 0)
			(list (quote source) (quote unknown))
			(list (quote distinct) nil)
			(list (quote distinct_confidence) 0)
			(list (quote distinct_source) (quote unknown))
			(list (quote null_fraction) nil)
			(list (quote min) nil)
			(list (quote max) nil)
			(list (quote raw_type) nil)
			(list (quote average_value_bytes) nil))
		(try
			(lambda ()
				(begin
					(define stats (planner_table_statistics (source_schema src) (source_relation src)))
					(define columns (if (nil? stats) nil (stats "columns")))
					(define column_stats (if (nil? columns) nil (columns column)))
					/* The immutable column catalog deliberately does not get rebuilt
					for every INSERT/DELETE. Resolve its row-count fallback against
					the current two-entry table snapshot in constant time instead. */
					(if (or (nil? column_stats)
						(not (equal? (qassoc_get column_stats (quote distinct_source) (quote unknown))
							(quote fallback_row_count))))
						column_stats
						(qassoc_set column_stats (quote distinct) (stats "row_count")))))
			(lambda (_e) (list
				(list (quote known) false)
				(list (quote confidence) 0)
				(list (quote source) (quote unknown))
				(list (quote distinct) nil)
				(list (quote distinct_confidence) 0)
				(list (quote distinct_source) (quote unknown))
				(list (quote raw_type) nil)
				(list (quote average_value_bytes) nil)))))))

(define planner_column_distinct_estimate (lambda (src column)
	(qassoc_get (planner_column_statistics src column) (quote distinct) nil)))

(define planner_column_average_value_bytes (lambda (src column)
	(begin
		(define statistics (planner_column_statistics src column))
		(define measured (qassoc_get statistics (quote average_value_bytes) nil))
		(if (not (nil? measured))
			measured
			/* Persisted generations created before planner statistics existed do
			not expose an average width until their next REBUILD. An unbounded text
			column must not look free in that interval: its scan still has to load
			and decode every value. This conservative width is an ordinary cost
			input, so all physical alternatives remain selected by the same model. */
			(if (contains? '("text" "bytea" "blob")
				(toLower (string (qassoc_get statistics (quote raw_type) ""))))
				4096
				0)))))

(define planner_group_distinct_estimate (lambda (src keys row_count)
	(reduce keys (lambda (estimate key)
		(if (nil? estimate)
			nil
			(begin
				(define column (direct_column_name_for_alias src key))
				(define distinct (planner_column_distinct_estimate src column))
				(if (nil? distinct) nil (min row_count (* estimate distinct))))))
		1)))

/* An enforced foreign key gives a useful hard upper bound for a grouped
driver: grouping the referencing columns cannot produce more non-NULL keys
than the referenced unique relation has rows. Keep one extra bucket for NULL.
The ordinary NDV estimate remains preferable when it is tighter. */
(define planner_columns_unique? (lambda (src columns)
	(or (equal? (source_primary_key_columns src) columns)
		(contains? (source_unique_key_sets src) columns))))

(define planner_foreign_key_group_bounds (lambda (src columns)
	(if (not (source_is_base_table? src))
		'()
		(try
			(lambda ()
				(begin
					(define info (show (source_schema src) (source_relation src) true))
					(define foreign_keys (coalesceNil ((info "meta") "ForeignKeys") '()))
					(filter (map foreign_keys (lambda (foreign_key)
						(if (and (equal? (foreign_key "Role") "referencing")
							(equal? (foreign_key "LocalColumns") columns))
							(begin
								(define target (list "__aggregate_pushdown_fk_target"
									(source_schema src) (foreign_key "OtherTable") false nil))
								(define target_columns (foreign_key "OtherColumns"))
								(define target_rows (planner_source_row_count target))
								(if (and (number? target_rows)
									(planner_columns_unique? target target_columns))
									(+ target_rows 1)
									nil))
							nil)))
						(lambda (bound) (number? bound)))))
			(lambda (_e) '())))))

(define planner_foreign_key_group_bound (lambda (src columns)
	(reduce (planner_foreign_key_group_bounds src columns) (lambda (best bound)
		(if (or (nil? best) (< bound best)) bound best)) nil)))

/* Independent single-column foreign keys also bound a multi-dimensional
partition before rebuilt NDV statistics exist. Composite foreign keys keep
their tighter direct bound; this product is only an additional candidate. */
(define planner_foreign_key_product_bound (lambda (src columns row_count)
	(if (or (not (number? row_count)) (empty_list? columns))
		nil
		(begin
			(define factors (map columns (lambda (column)
				(planner_foreign_key_group_bound src (list column)))))
			(if (reduce factors (lambda (missing factor)
				(or missing (nil? factor))) false)
				nil
				(min row_count (reduce factors (lambda (product factor)
					(* product factor)) 1)))))))

(define planner_aggregate_pushdown_group_estimate (lambda (src keys row_count)
	(if (not (number? row_count))
		nil
		(begin
			(define distinct_estimate (planner_group_distinct_estimate src keys row_count))
			(define columns (map keys (lambda (key) (direct_column_name_for_alias src key))))
			(define foreign_key_bound (if (reduce columns (lambda (missing column)
				(or missing (nil? column))) false)
				nil
				(planner_foreign_key_group_bound src columns)))
			(define foreign_key_product_bound
				(planner_foreign_key_product_bound src columns row_count))
			(define estimate (reduce
				(list distinct_estimate foreign_key_bound foreign_key_product_bound)
				(lambda (best candidate)
					(if (nil? candidate) best
						(if (nil? best) candidate (min best candidate))))
				nil))
			(if (nil? estimate) nil (min row_count estimate))))))

(define aggregate_pushdown_cost_preferred? (lambda (driver_rows group_rows)
	(and (number? driver_rows)
		(and (>= driver_rows 1024)
			(and (number? group_rows)
				(< (* group_rows 4) driver_rows))))))

(define planner_aggregate_pushdown_driver_rows (lambda (src residual)
	(begin
		(define total_rows (planner_source_row_count src))
		(if (not (number? total_rows))
			nil
			(if (literal_true? residual)
				total_rows
				(planner_row_count_after_selectivity src (list src)
					(source_alias src) residual total_rows))))))

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
	(begin
		(define row_estimate (planner_source_row_estimate src))
		(list
			(source_alias src)
			(list (quote relation) (source_relation src))
			(list (quote row_count) (qassoc_get row_estimate (quote value) nil))
			(list (quote row_estimate) row_estimate)
			(list (quote outer_join) (source_outer? src))
			(list (quote join_filter) (source_join_present? src))))))

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

(define left_join_requirement (lambda (src)
	(if (not (source_outer? src))
		nil
		(begin
			(define rows (planner_source_row_count src))
			(list
				(source_alias src)
				(list (quote access) (quote nullable_lookup))
				(list (quote estimated_rows) rows)
				(list (quote null_extension_barrier) true)
				(list (quote reuse) 1))))))

(define query_block_selectivity_estimates (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(define predicates (merge (list
			(split_and_terms (coalesceNil (qb_where block) true))
			(merge (map sources (lambda (src)
				(split_and_terms (coalesceNil (source_join_expr src) true))))))))
		(map (filter predicates (lambda (predicate) (not (equal? predicate true))))
			(lambda (predicate)
				(begin
					(define estimate (join_optimizer_expr_selectivity_estimate
						sources default_alias predicate))
					(list
						(list (quote predicate) predicate)
						(list (quote estimate) estimate)
						(list (quote planning_value)
							(planner_estimate_planning_value estimate
								(if (equal? (qassoc_get estimate (quote source) nil) (quote range_unknown))
									0.3333333333333333 0.1))))))))))

(define explain_reorder_selectivities? (lambda ()
	(try
		(lambda () (coalesceNil ((context "session") "__memcp_explain_reorder_selectivities") false))
		(lambda (_e) false))))

(define query_block_reorder_telemetry (lambda (block)
	(merge (list
		(list (list (quote source_estimates) (map (qb_sources block) source_reorder_estimate)))
		(if (explain_reorder_selectivities?)
			(list (list (quote selectivity_estimates) (query_block_selectivity_estimates block)))
			'())
		(list (list (quote left_join_requirements) (filter
			(map (qb_sources block) left_join_requirement)
			(lambda (item) (not (nil? item))))))))))

(define planner_add_estimates (lambda (values)
	(reduce (coalesceNil values '()) (lambda (acc value)
		(if (nil? value) acc (+ acc value)))
		0)))

(define planner_estimate_population (lambda (estimate)
	(qassoc_get estimate (quote population) (quote table_rows))))

(define planner_estimate_coverage (lambda (estimate)
	(qassoc_get estimate (quote coverage)
		(if (qassoc_get estimate (quote capped) false) (quote sampled) (quote exact)))))

(define planner_merge_estimate_population (lambda (estimates)
	(if (reduce estimates (lambda (found estimate)
		(or found (equal? (planner_estimate_population estimate) (quote index_candidates)))) false)
		(quote index_candidates)
		(if (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_population estimate) (quote recset_candidates)))) false)
			(quote recset_candidates)
			(quote table_rows)))))

(define planner_merge_estimate_coverage (lambda (estimates)
	(if (reduce estimates (lambda (found estimate)
		(or found (equal? (planner_estimate_coverage estimate) (quote lower_bound)))) false)
		(quote lower_bound)
		(if (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_coverage estimate) (quote sampled)))) false)
			(quote sampled)
			(quote exact)))))

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
					(define sampled_rows (planner_add_estimates (map available (lambda (estimate)
						(qassoc_get estimate (quote sampled) nil)))))
					(define capped (or (>= rows max_rows)
						(reduce available (lambda (found estimate)
							(or found (qassoc_get estimate (quote capped) false)))
							false)))
					(list
						(list (quote rows) rows)
						(list (quote capped) capped)
						(list (quote sampled) sampled_rows)
						(list (quote input) input_rows)
						(list (quote population) (planner_merge_estimate_population available))
						(list (quote coverage) (planner_merge_estimate_coverage available))))))
		(if (query_block? input)
			(if (single_source? (qb_sources input))
				(begin
					(define src (car (qb_sources input)))
					(planner_source_filter_estimate src (combine_where (qb_where input) (source_join_expr src)) max_rows))
				nil)
			nil))))

(define membership_expr_has_driver_alternative? (lambda (expr)
	(match expr
		(cons head tail) (begin
			(define is_or (or (equal? head (quote or)) (equal? head (symbol "or"))))
			(define has_membership (and is_or (reduce tail (lambda (found item)
				(or found (expr_contains_membership_truth? item))) false)))
			(define has_driver (and is_or (reduce tail (lambda (found item)
				(or found (not (expr_contains_membership_truth? item)))) false)))
			(or (and has_membership has_driver)
				(reduce tail (lambda (found item)
					(or found (membership_expr_has_driver_alternative? item))) false)))
		_ false)))

(define membership_driver_local_filter (lambda (driver sources block)
	(begin
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(source_alias driver)))
		(define other_aliases (map (filter (coalesceNil sources '()) (lambda (src)
			(not (equal? (source_alias src) (source_alias driver))))) source_alias))
		(combine_where_terms
			(filter (split_and_terms (membership_driver_filter (qb_where block))) (lambda (term)
				(not (expr_refs_any_alias? default_alias other_aliases term))))
			true))))

(define membership_aggregate_pushdown_driver_rows (lambda (block)
	(begin
		(define pushdown (qassoc_get (qb_facts block) (quote aggregate_pushdown) nil))
		(if (nil? pushdown)
			nil
			(qassoc_get pushdown (quote estimated_driver_rows) nil)))))

/* Physical costing needs the amount of scalar work, not a speculative copy of
each alternative plan. This walker visits the already canonical expression
once; get_column is a leaf because column reads are costed independently. */
(define physical_text_scan_operation? (lambda (expr)
	(match expr
		((symbol strlike) _value _pattern _collation) true
		((quote strlike) _value _pattern _collation) true
		((symbol strlike_cs) _value _pattern _collation) true
		((quote strlike_cs) _value _pattern _collation) true
		_ false)))

/* Every text predicate must decode its input regardless of result selectivity.
REBUILD already walks every base value, so its exact average byte width lets
costing distinguish short labels from large text documents without building
either physical alternative or sampling the table again. */
(define physical_expression_work_profile (lambda (src expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(list (list (quote operations) 0) (list (quote depth) 0)
			(list (quote broad_text_matches) 0) (list (quote broad_text_average_bytes) 0))
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(list (list (quote operations) 0) (list (quote depth) 0)
			(list (quote broad_text_matches) 0) (list (quote broad_text_average_bytes) 0))
		(cons head tail) (begin
			(define children (map tail (lambda (child) (physical_expression_work_profile src child))))
			(define own (if (symbol? head) 1 0))
			(list
				(list (quote operations) (+ own (reduce children (lambda (total child)
					(+ total (qassoc_get child (quote operations) 0))) 0)))
				(list (quote depth) (+ own (reduce children (lambda (depth child)
					(max depth (qassoc_get child (quote depth) 0))) 0)))
				(list (quote broad_text_matches) (+
					(if (physical_text_scan_operation? expr) 1 0)
					(reduce children (lambda (total child)
						(+ total (qassoc_get child (quote broad_text_matches) 0))) 0)))
				(list (quote broad_text_average_bytes) (+
					(if (physical_text_scan_operation? expr)
						(reduce (extract_columns_for_alias src (car tail)) (lambda (total column)
							(+ total (coalesceNil (planner_column_average_value_bytes src column) 0))) 0)
						0)
					(reduce children (lambda (total child)
						(+ total (qassoc_get child (quote broad_text_average_bytes) 0))) 0)))))
		_ (list (list (quote operations) 0) (list (quote depth) 0)
			(list (quote broad_text_matches) 0) (list (quote broad_text_average_bytes) 0)))))

(define membership_source_work_profile (lambda (src condition map_expr)
	(begin
		(define input_rows (planner_source_row_count src))
		(define filter_columns (extract_columns_for_alias src condition))
		(define map_columns (extract_columns_for_alias src map_expr))
		(define expression_work (physical_expression_work_profile src condition))
		(define broad_text_average_bytes (qassoc_get expression_work (quote broad_text_average_bytes) 0))
		(list
			(list (quote scan_invocations) 1)
			(list (quote input_rows) input_rows)
			(list (quote filter_columns) (count filter_columns))
			(list (quote map_columns) (count map_columns))
			(list (quote expression_operations) (qassoc_get expression_work (quote operations) 0))
			(list (quote expression_depth) (qassoc_get expression_work (quote depth) 0))
			(list (quote broad_text_match_rows)
				(if (number? input_rows)
					(* input_rows (qassoc_get expression_work (quote broad_text_matches) 0)) nil))
			(list (quote broad_text_match_bytes)
				(if (number? input_rows) (* input_rows broad_text_average_bytes) nil))
			(list (quote filter_value_rows)
				(if (number? input_rows) (* input_rows (count filter_columns)) nil))
			(list (quote expression_operation_rows)
				(if (number? input_rows)
					(* input_rows (qassoc_get expression_work (quote operations) 0)) nil))))))

(define membership_candidate_branch_work_profile (lambda (branch)
	(if (and (query_block? branch) (single_real_source? (qb_sources branch)))
		(begin
			(define src (single_real_source (qb_sources branch)))
			(membership_source_work_profile src
				(combine_where (qb_where branch) (source_join_expr src))
				(query_block_first_expr branch)))
		nil)))

(define membership_merge_candidate_work_profiles (lambda (profiles)
	(reduce profiles (lambda (combined profile)
		(if (nil? profile)
			combined
			(list
				(list (quote scan_invocations) (+
					(qassoc_get combined (quote scan_invocations) 0)
					(qassoc_get profile (quote scan_invocations) 0)))
				(list (quote input_rows) (+
					(coalesceNil (qassoc_get combined (quote input_rows) nil) 0)
					(coalesceNil (qassoc_get profile (quote input_rows) nil) 0)))
				(list (quote filter_columns) (max
					(qassoc_get combined (quote filter_columns) 0)
					(qassoc_get profile (quote filter_columns) 0)))
				(list (quote map_columns) (max
					(qassoc_get combined (quote map_columns) 0)
					(qassoc_get profile (quote map_columns) 0)))
				(list (quote expression_operations) (max
					(qassoc_get combined (quote expression_operations) 0)
					(qassoc_get profile (quote expression_operations) 0)))
				(list (quote expression_depth) (max
					(qassoc_get combined (quote expression_depth) 0)
					(qassoc_get profile (quote expression_depth) 0)))
				(list (quote broad_text_match_rows) (+
					(coalesceNil (qassoc_get combined (quote broad_text_match_rows) nil) 0)
					(coalesceNil (qassoc_get profile (quote broad_text_match_rows) nil) 0)))
				(list (quote broad_text_match_bytes) (+
					(coalesceNil (qassoc_get combined (quote broad_text_match_bytes) nil) 0)
					(coalesceNil (qassoc_get profile (quote broad_text_match_bytes) nil) 0)))
				(list (quote filter_value_rows) (+
					(coalesceNil (qassoc_get combined (quote filter_value_rows) nil) 0)
					(coalesceNil (qassoc_get profile (quote filter_value_rows) nil) 0)))
				(list (quote expression_operation_rows) (+
					(coalesceNil (qassoc_get combined (quote expression_operation_rows) nil) 0)
					(coalesceNil (qassoc_get profile (quote expression_operation_rows) nil) 0))))))
		'())))

(define membership_candidate_work_profile (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(membership_merge_candidate_work_profiles
				(map (union_branches input) membership_candidate_branch_work_profile))
			(if (query_block? input)
				(membership_candidate_branch_work_profile input)
				(if (source_is_base_table? input)
					(membership_source_work_profile input
						(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)
						(gs_keys stage))
					'()))))))

(define membership_candidate_work_facts (lambda (stage)
	(begin
		(define work (membership_candidate_work_profile stage))
		(list
			(list (quote membership_candidate_scan_invocations) (qassoc_get work (quote scan_invocations) 1))
			(list (quote membership_candidate_filter_columns) (qassoc_get work (quote filter_columns) 0))
			(list (quote membership_candidate_map_columns) (qassoc_get work (quote map_columns) (count (gs_keys stage))))
			(list (quote membership_candidate_expression_operations) (qassoc_get work (quote expression_operations) 0))
			(list (quote membership_candidate_expression_depth) (qassoc_get work (quote expression_depth) 0))
			(list (quote membership_candidate_broad_text_match_rows) (qassoc_get work (quote broad_text_match_rows) 0))
			(list (quote membership_candidate_broad_text_match_bytes) (qassoc_get work (quote broad_text_match_bytes) 0))
			(list (quote membership_candidate_filter_value_rows) (qassoc_get work (quote filter_value_rows) nil))
			(list (quote membership_candidate_expression_operation_rows) (qassoc_get work (quote expression_operation_rows) nil))))))

(define membership_driver_work_profile (lambda (driver sources block)
	(if (nil? driver)
		'()
		(begin
			(define condition (membership_driver_local_filter driver sources block))
			(define profile (membership_source_work_profile driver condition
				(list (qb_fields block) (qb_group block) (qb_having block)
					(qb_order block) (qb_hidden block))))
			/* Stage rebinding may replace base get_column leaves before this point.
			The output arity is still a cheap lower bound for values the scan mapper
			must produce, and preserves the distinction between narrow and wide
			consumers without constructing either physical alternative. */
			(qassoc_set profile (quote map_columns)
				(max (qassoc_get profile (quote map_columns) 0) (count (qb_fields block))))))))

(define candidate_reorder_telemetry (lambda (stage sources block)
	(begin
		(define driver (reduce (coalesceNil sources '()) (lambda (found src)
			(if (not (nil? found)) found
				(if (source_is_base_table? src) src nil))) nil))
		/* COUNT pushdown can replace the base driver with an aggregate-partition
		stage before membership reorder. Preserve the already costed base-row work
		from that logical fact instead of turning the physical choice into an
		unknown-cost fallback. */
		(define driver_input_rows (if (nil? driver)
			(membership_aggregate_pushdown_driver_rows block)
			(planner_source_row_count driver)))
		(define driver_estimate (if (nil? driver)
			nil
			(planner_source_filter_estimate driver
				(membership_driver_local_filter driver sources block) 512)))
		(define driver_rows (membership_estimated_work_rows driver_estimate driver_input_rows))
		(define candidate_rows (planner_stage_input_rows (gs_input stage)))
		(define candidate_probe_branches (if (union_block? (gs_input stage))
			(count (union_branches (gs_input stage)))
			1))
		(define candidate_estimate (planner_stage_filter_estimate (gs_input stage) 512))
		(define estimate_rows (qassoc_get candidate_estimate (quote rows) nil))
		(define estimate_capped (qassoc_get candidate_estimate (quote capped) false))
		(define estimate_input (qassoc_get candidate_estimate (quote input) nil))
		(define estimate_sampled (qassoc_get candidate_estimate (quote sampled) nil))
		(define estimate_population (planner_estimate_population candidate_estimate))
		(define estimate_coverage (planner_estimate_coverage candidate_estimate))
		(define estimate_ratio_broad (and
			(and (not (nil? estimate_rows)) (and (not (nil? estimate_sampled)) (> estimate_sampled 0)))
			(>= (* estimate_rows 4) estimate_sampled)))
		(define driver_alternative (membership_expr_has_driver_alternative? (qb_where block)))
		(define class (if (or estimate_ratio_broad
			(if driver_alternative (candidate_stage_broad? stage) false)) (quote broad) (quote selective)))
		(define ordered_driver (if (nil? driver) false
			(or (source_order_limit_driver? driver (qb_order block) (qb_limit block))
				(source_row_number_limit_driver? (qb_stages block) driver))))
		(define consumer (if (query_block_has_aggregates? block) (quote aggregate)
			(if ordered_driver (quote order_limit) (quote filter))))
		(define candidate_work (membership_candidate_work_profile stage))
		(define driver_work (membership_driver_work_profile driver sources block))
		(list
			(list (quote membership_stage_id) (gs_id stage))
			(list (quote membership_selectivity_class) class)
			(list (quote membership_driver_rows) driver_rows)
			(list (quote membership_driver_input_rows) driver_input_rows)
			(list (quote membership_driver_estimate_capped)
				(qassoc_get driver_estimate (quote capped) false))
			(list (quote membership_candidate_input_rows) candidate_rows)
			(list (quote membership_candidate_estimated_rows) estimate_rows)
			(list (quote membership_candidate_estimate_capped) estimate_capped)
			(list (quote membership_candidate_estimate_input) estimate_input)
			(list (quote membership_candidate_estimate_sampled) estimate_sampled)
			(list (quote membership_candidate_estimate_population) estimate_population)
			(list (quote membership_candidate_estimate_coverage) estimate_coverage)
			(list (quote membership_candidate_probe_branches) candidate_probe_branches)
			(list (quote membership_candidate_scan_invocations) (qassoc_get candidate_work (quote scan_invocations) candidate_probe_branches))
			(list (quote membership_candidate_filter_columns) (qassoc_get candidate_work (quote filter_columns) 0))
			(list (quote membership_candidate_map_columns) (qassoc_get candidate_work (quote map_columns) (count (gs_keys stage))))
			(list (quote membership_candidate_cache_map_columns) (+ (count (gs_keys stage)) (count (gs_aggregates stage))))
			(list (quote membership_candidate_expression_operations) (qassoc_get candidate_work (quote expression_operations) 0))
			(list (quote membership_candidate_expression_depth) (qassoc_get candidate_work (quote expression_depth) 0))
			(list (quote membership_candidate_broad_text_match_rows) (qassoc_get candidate_work (quote broad_text_match_rows) 0))
			(list (quote membership_candidate_broad_text_match_bytes) (qassoc_get candidate_work (quote broad_text_match_bytes) 0))
			(list (quote membership_candidate_filter_value_rows)
				(coalesceNil (qassoc_get candidate_work (quote filter_value_rows) nil)
					(if (number? candidate_rows)
						(* candidate_rows (qassoc_get candidate_work (quote filter_columns) 0)) nil)))
			(list (quote membership_candidate_expression_operation_rows)
				(coalesceNil (qassoc_get candidate_work (quote expression_operation_rows) nil)
					(if (number? candidate_rows)
						(* candidate_rows (qassoc_get candidate_work (quote expression_operations) 0)) nil)))
			(list (quote membership_driver_scan_invocations) (qassoc_get driver_work (quote scan_invocations) (if (nil? driver) 0 1)))
			(list (quote membership_driver_filter_columns) (qassoc_get driver_work (quote filter_columns) 0))
			(list (quote membership_driver_map_columns) (qassoc_get driver_work (quote map_columns) 0))
			(list (quote membership_driver_expression_operations) (qassoc_get driver_work (quote expression_operations) 0))
			(list (quote membership_driver_expression_depth) (qassoc_get driver_work (quote expression_depth) 0))
			(list (quote membership_driver_alternative) driver_alternative)
			(list (quote membership_order_limit) (qb_limit block))
			(list (quote membership_order_offset) (coalesceNil (qb_offset block) 0))
			(list (quote membership_order_limit_driver) ordered_driver)
			(list (quote membership_consumer) consumer)))))

(define stage_reorder_telemetry (lambda (stage)
	(list
		(list (quote group_requirement) (list
			(list (quote purpose) (qassoc_get (gs_facts stage) (quote purpose) nil))
			(list (quote input_rows) (planner_stage_input_rows (gs_input stage)))
			(list (quote domain_count) (count (gs_domain stage)))
			(list (quote key_count) (count (gs_keys stage)))
			(list (quote reuse) 1))))))

/* A stage embedded directly into a driver_membership_probe marker (see
rewrite_exists_recset_probe_refs) is later consumed far from the query block
that discovered it -- e.g. exists_recset_project_join_expr, building a recset
from the stage's own group-cache, has no access to the global stages list a
qb_sources-based nested dependency inside the stage's own domain would need
to resolve. Stamping the discovering block's own stages list onto the
stage's facts here lets lower_group_stage_prepare_using's existing
gs_facts['stage_catalog'] fallback find it later, without threading a global
catalog through every downstream caller. */
(define group_stage_with_stage_catalog (lambda (stage catalog)
	(if (nil? stage)
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
			(qassoc_set (gs_facts stage) (quote stage_catalog) catalog)))))

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

(define physical_candidate_membership_strategy (lambda (telemetry)
	(begin
		(define candidates (membership_cost_options_for_telemetry telemetry))
		(if (empty_list? candidates)
			(membership_driver_strategy_for_telemetry telemetry)
			(car (reduce (cdr candidates) (lambda (chosen candidate)
				(if (planner_cost_better? (cadr candidate) (cadr chosen)) candidate chosen))
				(car candidates)))))))

(define membership_driver_strategy_for_telemetry (lambda (telemetry)
	(if (qassoc_get telemetry (quote membership_order_limit_driver) false)
		(quote driver_order_membership_probe)
		(quote driver_filter_join_probe))))

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

(define driver_membership_probe_expr_for_strategy (lambda (stage probe strategy)
	(if (equal? strategy (quote driver_filter_join_probe))
		(list (quote and)
			(list (quote not) (list (quote nil?) probe))
			(list (quote driver_membership_subscan_probe) stage probe))
		(driver_membership_probe_expr stage probe))))

(define logical_membership_probe_expr (lambda (stage probe)
	(list (quote and)
		(list (quote not) (list (quote nil?) probe))
		(list (quote membership_requirement_probe) stage probe))))

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

/* Only rows examined independently of the predicate's access index form a
selectivity sample. Once an index or RecSet has restricted the iterator,
sampled means "index candidates examined", not "table rows sampled". A capped
restricted estimate is therefore merely a lower bound and must not be
extrapolated as though the iterator were an unbiased table sample. */
(define membership_estimated_matching_rows (lambda (estimate input_rows fallback)
	(if (nil? estimate)
		fallback
		(begin
			(define rows (qassoc_get estimate (quote rows) nil))
			(define sampled_input (qassoc_get estimate (quote sampled) nil))
			(if (and (qassoc_get estimate (quote capped) false)
				(and (number? input_rows)
					(and (number? rows) (and (number? sampled_input) (> sampled_input 0)))))
				(if (equal? (planner_estimate_coverage estimate) (quote lower_bound))
					(coalesceNil fallback input_rows)
					(min input_rows (* input_rows (/ rows sampled_input))))
				(coalesceNil rows fallback))))))

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

(define membership_work_value (lambda (work key fallback)
	(coalesceNil (qassoc_get work key nil) fallback)))

(define membership_candidate_filter_value_rows (lambda (candidate_input_rows work)
	(membership_work_value work (quote membership_candidate_filter_value_rows)
		(* candidate_input_rows
			(membership_work_value work (quote membership_candidate_filter_columns) 0)))))

(define membership_candidate_expression_operation_rows (lambda (candidate_input_rows work)
	(membership_work_value work (quote membership_candidate_expression_operation_rows)
		(* candidate_input_rows
			(membership_work_value work (quote membership_candidate_expression_operations) 0)))))

(define membership_projected_driver_rows (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		(define branches (max 1 (membership_work_value work (quote membership_candidate_probe_branches) 1)))
		(define candidate_domain_rows (/ candidate_input_rows branches))
		(if (> candidate_domain_rows 0)
			(* driver_rows (min 1 (/ candidate_rows candidate_domain_rows)))
			0))))

(define membership_candidate_density (lambda (candidate_input_rows candidate_rows work)
	(begin
		(define branches (max 1 (membership_work_value work (quote membership_candidate_probe_branches) 1)))
		(define candidate_domain_rows (/ candidate_input_rows branches))
		(if (> candidate_domain_rows 0)
			(min 1 (/ candidate_rows candidate_domain_rows))
			0))))

(define membership_driver_input_rows (lambda (driver_rows work)
	(coalesceNil (membership_work_value work (quote membership_driver_input_rows) nil)
		driver_rows)))

/* An ordered scan can stop only after it has visited enough rows to obtain the
requested number of membership hits. This estimate uses logical cardinalities
and the already available predicate profile; it does not construct either
physical alternative. */
(define membership_expected_driver_rows_visited (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		(define driver_input_rows (membership_driver_input_rows driver_rows work))
		(if (not (membership_work_value work (quote membership_order_limit_driver) false))
			driver_rows
			(begin
				(define density (membership_candidate_density candidate_input_rows candidate_rows work))
				(define requested_rows (+ driver_rows
					(membership_work_value work (quote membership_order_offset) 0)))
				(if (> density 0)
					(min driver_input_rows (/ requested_rows density))
					driver_input_rows))))))

(define membership_common_scan_cost (lambda (candidate_input_rows candidate_rows driver_rows candidate_map_columns work)
	(begin
		(define scan_invocations (+
			(membership_work_value work (quote membership_candidate_scan_invocations) 1)
			(membership_work_value work (quote membership_driver_scan_invocations) 1)))
		(define filter_value_rows (+
			(membership_candidate_filter_value_rows candidate_input_rows work)
			(* driver_rows (membership_work_value work (quote membership_driver_filter_columns) 0))))
		(define map_value_rows (+
			(* candidate_rows candidate_map_columns)
			(* (membership_projected_driver_rows candidate_input_rows candidate_rows driver_rows work)
				(membership_work_value work (quote membership_driver_map_columns) 0))))
		(define expression_operation_rows (+
			(membership_candidate_expression_operation_rows candidate_input_rows work)
			(* driver_rows (membership_work_value work (quote membership_driver_expression_operations) 0))))
		(planner_cost
			(* scan_invocations planner_membership_scan_invocation_ns)
			(+
				(* (+ candidate_input_rows driver_rows) planner_membership_scan_row_ns)
				(* filter_value_rows planner_membership_filter_column_row_ns)
				(* map_value_rows planner_membership_map_column_row_ns)
				(* expression_operation_rows planner_membership_expression_operation_row_ns))
			0 0 0 0 0 0 driver_rows 0.75))))

(define membership_projection_cost_preferred? (lambda (candidate_input_rows candidate_rows driver_rows work)
	(if (or (nil? candidate_input_rows) (or (nil? candidate_rows) (nil? driver_rows)))
		false
		(planner_cost_better?
			(membership_projection_cost candidate_input_rows candidate_rows driver_rows work)
			(membership_ordered_driver_probe_cost candidate_input_rows candidate_rows driver_rows work)))))

(define membership_projection_cost (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		/* FK projection must visit the target relation even when the downstream
		ordered consumer accepts only a small LIMIT window. */
		(define projection_rows (membership_driver_input_rows driver_rows work))
		(define projected_rows (membership_projected_driver_rows
			candidate_input_rows candidate_rows projection_rows work))
		(define base_cost (planner_cost_add
			(planner_cost_add
				(membership_common_scan_cost candidate_input_rows candidate_rows projection_rows
					(membership_work_value work (quote membership_candidate_map_columns) 1) work)
				(planner_cost 0
					(+
						(* (membership_work_value work (quote membership_candidate_broad_text_match_rows) 0)
							planner_membership_broad_text_match_row_ns)
						(* (membership_work_value work (quote membership_candidate_broad_text_match_bytes) 0)
							planner_membership_broad_text_match_byte_ns))
					0 0 0 0 0 0 candidate_input_rows 0.55)
				candidate_input_rows 0.55)
			(planner_cost planner_membership_recset_startup_ns 0 (* driver_rows planner_membership_recset_probe_row_ns)
				0 0 (* (+ candidate_rows projected_rows) planner_membership_recset_build_row_ns)
				(* (+ candidate_rows projected_rows) 8) 0 projection_rows 0.65)
			projection_rows 0.65))
		(if (equal? (membership_work_value work (quote membership_consumer) (quote filter))
			(quote aggregate))
			(planner_cost_add base_cost
				(planner_cost 0 (* projection_rows planner_membership_recset_aggregate_row_ns)
					0 0 0 0 0 0 projection_rows 0.65)
				projection_rows 0.65)
			base_cost))))

(define membership_cached_driver_probe_cost (lambda (driver_rows)
	(planner_cost 0 0 (* driver_rows 54) 0 0 0 0 0 driver_rows 0.5)))

(define membership_driver_probe_cost (lambda (driver_rows probe_branches)
	(begin
		(define probes (* driver_rows probe_branches))
		/* A driver membership check lowers each candidate branch to an indexed
		point-presence probe. Keep that storage subscan distinct from the ordered
		candidate-key index calibrated below. */
		(planner_direct_presence_probe_cost probes))))

(define membership_ordered_driver_probe_cost (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		(define visited_rows (membership_expected_driver_rows_visited
			candidate_input_rows candidate_rows driver_rows work))
		(define driver_input_rows (membership_driver_input_rows driver_rows work))
		(define base_cost (planner_cost_add
			(membership_common_scan_cost candidate_input_rows candidate_rows visited_rows
				(membership_work_value work (quote membership_candidate_cache_map_columns) 2) work)
			(planner_cost planner_membership_group_cache_startup_ns 0 (* visited_rows planner_membership_group_cache_probe_row_ns)
				0 0 (* candidate_rows planner_membership_group_cache_build_row_ns)
				(* candidate_rows 8) 0 (+ candidate_rows visited_rows) 0.65)
			visited_rows 0.65))
		(if (and (membership_work_value work (quote membership_order_limit_driver) false)
			(not (membership_work_value work (quote membership_driver_alternative) false)))
			(planner_cost_add base_cost
				(planner_cost 0
					(* (/ (* driver_input_rows driver_input_rows) 1000000)
						planner_membership_ordered_driver_input_row_ns)
					0 0 0 0 0 0 driver_input_rows 0.65)
				visited_rows 0.65)
			base_cost))))

(define membership_cost_options (lambda (candidate_input_rows candidate_rows driver_rows probe_branches driver_strategy work)
	(if (or (nil? candidate_input_rows) (or (nil? candidate_rows) (nil? driver_rows)))
		'()
		(list
			(list (quote candidate_keyset) (membership_projection_cost candidate_input_rows candidate_rows driver_rows work))
			(list driver_strategy
				(if (equal? driver_strategy (quote driver_order_membership_probe))
					(membership_ordered_driver_probe_cost candidate_input_rows candidate_rows driver_rows work)
					(membership_driver_probe_cost driver_rows probe_branches)))))))

(define membership_cost_options_for_telemetry (lambda (telemetry)
	(begin
		(define candidate_rows (membership_estimated_matching_rows
			(list
				(list (quote rows) (qassoc_get telemetry (quote membership_candidate_estimated_rows) nil))
				(list (quote input) (qassoc_get telemetry (quote membership_candidate_estimate_input) nil))
				(list (quote sampled) (qassoc_get telemetry (quote membership_candidate_estimate_sampled) nil))
				(list (quote capped) (qassoc_get telemetry (quote membership_candidate_estimate_capped) false))
				(list (quote population) (qassoc_get telemetry (quote membership_candidate_estimate_population) (quote table_rows)))
				(list (quote coverage) (qassoc_get telemetry (quote membership_candidate_estimate_coverage) (quote sampled))))
			(qassoc_get telemetry (quote membership_candidate_input_rows) nil)
			(qassoc_get telemetry (quote membership_candidate_input_rows) nil)))
		(define driver_strategy (membership_driver_strategy_for_telemetry telemetry))
		(define driver_rows (qassoc_get telemetry (quote membership_driver_rows) nil))
		(define costed_driver_rows (if (equal? driver_strategy (quote driver_order_membership_probe))
			(coalesceNil (probe_limit_work_rows
				(qassoc_get telemetry (quote membership_order_limit) nil)) driver_rows)
			driver_rows))
		(membership_cost_options
			(qassoc_get telemetry (quote membership_candidate_input_rows) nil)
			candidate_rows
			costed_driver_rows
			(qassoc_get telemetry (quote membership_candidate_probe_branches) 1)
			driver_strategy
			telemetry))))

(define record_membership_physical_choice (lambda (requirement strategy reason candidates)
	(begin
		(define decision_id (concat "membership_carrier:"
			(qassoc_get requirement (quote membership_stage_id) "unknown")))
		(define alternatives (map candidates (lambda (candidate) (string (car candidate)))))
		(define forced (planner_physical_override decision_id))
		(define chosen (planner_physical_choice decision_id (string strategy) alternatives))
		(define candidate_input_rows (qassoc_get requirement (quote membership_candidate_input_rows) nil))
		(define candidate_rows (membership_estimated_matching_rows
			(list
				(list (quote rows) (qassoc_get requirement (quote membership_candidate_estimated_rows) nil))
				(list (quote input) (qassoc_get requirement (quote membership_candidate_estimate_input) nil))
				(list (quote sampled) (qassoc_get requirement (quote membership_candidate_estimate_sampled) nil))
				(list (quote capped) (qassoc_get requirement (quote membership_candidate_estimate_capped) false))
				(list (quote population) (qassoc_get requirement (quote membership_candidate_estimate_population) (quote table_rows)))
				(list (quote coverage) (qassoc_get requirement (quote membership_candidate_estimate_coverage) (quote sampled))))
			candidate_input_rows candidate_input_rows))
		(define driver_input_rows (qassoc_get requirement (quote membership_driver_input_rows) nil))
		(define driver_rows (if (qassoc_get requirement (quote membership_order_limit_driver) false)
			(coalesceNil (probe_limit_work_rows (qassoc_get requirement (quote membership_order_limit) nil))
				(qassoc_get requirement (quote membership_driver_rows) nil))
			(qassoc_get requirement (quote membership_driver_rows) nil)))
		(planner_record_physical_decision
			(list
				(list "decision_id" decision_id)
				(list "decision" "membership_carrier")
				(list "decision_site" "physical_requirement")
				(list "stage" (qassoc_get requirement (quote membership_stage_id) nil))
				(list "consumer" (string
					(qassoc_get requirement (quote membership_consumer) (quote filter))))
				(list "chosen" chosen)
				(list "normally_chosen" (string strategy))
				(list "selection" (if (nil? forced) "cost" "forced"))
				(list "reason" (if (nil? forced) (string reason) "calibration_override"))
				(list "inputs" (list
					(list "candidate_rows" candidate_rows)
					(list "candidate_input_rows" candidate_input_rows)
					(list "candidate_density" (if (and (number? candidate_input_rows) (number? candidate_rows))
						(membership_candidate_density candidate_input_rows candidate_rows requirement) nil))
					(list "projected_driver_rows" (if (and (number? candidate_input_rows)
						(and (number? candidate_rows) (number? driver_input_rows)))
						(membership_projected_driver_rows candidate_input_rows candidate_rows driver_input_rows requirement) nil))
					(list "driver_rows" driver_rows)
					(list "driver_input_rows" driver_input_rows)
					(list "expected_driver_rows_visited" (if (and (number? candidate_input_rows)
						(and (number? candidate_rows) (number? driver_rows)))
						(membership_expected_driver_rows_visited candidate_input_rows candidate_rows driver_rows requirement) nil))
					(list "limit" (qassoc_get requirement (quote membership_order_limit) nil))
					(list "offset" (qassoc_get requirement (quote membership_order_offset) 0))
					(list "driver_estimate_capped" (qassoc_get requirement (quote membership_driver_estimate_capped) false))
					(list "selectivity_class" (string (qassoc_get requirement (quote membership_selectivity_class) nil)))
					(list "estimate_capped" (qassoc_get requirement (quote membership_candidate_estimate_capped) false))
					(list "estimate_sampled_rows" (qassoc_get requirement (quote membership_candidate_estimate_sampled) nil))
					(list "estimate_population" (string (qassoc_get requirement (quote membership_candidate_estimate_population) (quote table_rows))))
					(list "estimate_coverage" (string (qassoc_get requirement (quote membership_candidate_estimate_coverage) (quote sampled))))
					(list "probe_branches" (qassoc_get requirement (quote membership_candidate_probe_branches) 1))
					(list "candidate_scan_invocations" (qassoc_get requirement (quote membership_candidate_scan_invocations) 1))
					(list "candidate_filter_columns" (qassoc_get requirement (quote membership_candidate_filter_columns) 0))
					(list "candidate_map_columns" (qassoc_get requirement (quote membership_candidate_map_columns) 1))
					(list "candidate_cache_map_columns" (qassoc_get requirement (quote membership_candidate_cache_map_columns) 2))
					(list "candidate_expression_operations" (qassoc_get requirement (quote membership_candidate_expression_operations) 0))
					(list "candidate_expression_depth" (qassoc_get requirement (quote membership_candidate_expression_depth) 0))
					(list "candidate_broad_text_match_rows" (qassoc_get requirement (quote membership_candidate_broad_text_match_rows) 0))
					(list "candidate_broad_text_match_bytes" (qassoc_get requirement (quote membership_candidate_broad_text_match_bytes) 0))
					(list "candidate_filter_value_rows" (qassoc_get requirement (quote membership_candidate_filter_value_rows) nil))
					(list "candidate_expression_operation_rows" (qassoc_get requirement (quote membership_candidate_expression_operation_rows) nil))
					(list "driver_scan_invocations" (qassoc_get requirement (quote membership_driver_scan_invocations) 1))
					(list "driver_filter_columns" (qassoc_get requirement (quote membership_driver_filter_columns) 0))
					(list "driver_map_columns" (qassoc_get requirement (quote membership_driver_map_columns) 0))
					(list "driver_expression_operations" (qassoc_get requirement (quote membership_driver_expression_operations) 0))
					(list "driver_expression_depth" (qassoc_get requirement (quote membership_driver_expression_depth) 0))
					(list "reuse" (qassoc_get requirement (quote reuse) 1))))
				(list "alternatives" (map candidates (lambda (candidate)
					(list
						(list "plan" (string (car candidate)))
						(list "status" (if (equal? (string (car candidate)) chosen) "chosen" "rejected"))
						(list "reason" (if (equal? (string (car candidate)) chosen) "selected" "higher_total_ns_or_forced_alternative"))
						(list "cost" (planner_cost_explain (cadr candidate)))))))))
		(if (qassoc_get requirement (quote membership_order_limit_driver) false)
			(begin
				(define recset_source (equal? chosen "candidate_keyset"))
				(planner_record_physical_decision
					(list
						(list "decision" "ordered_recset_iterator")
						(list "chosen" (if recset_source "scan_order_recset_part" "recset_contains_probe"))
						(list "reason" (if recset_source "projected_recset_scan_source" "base_table_scan_with_membership_filter"))
						(list "inputs" (list
							(list "limit" (qassoc_get requirement (quote membership_order_limit) nil))
							(list "candidate_rows" (qassoc_get requirement (quote membership_candidate_estimated_rows) nil))
							(list "driver_rows" (qassoc_get requirement (quote membership_driver_rows) nil))))
						(list "alternatives" (list
							(list
								(list "plan" "scan_order_recset_part")
								(list "status" (if recset_source "chosen" "rejected"))
								(list "reason" (if recset_source "projected_recset_scan_source" "source_is_base_table")))
							(list
								(list "plan" "recset_contains_probe")
								(list "status" (if recset_source "rejected" "chosen"))
								(list "reason" (if recset_source "direct_intersection_available" "membership_is_filter"))))))))
			nil))))
/* The reorder phase carries only an abstract membership requirement. Concrete
keyset/probe names enter the IR here, at the physical preparation boundary. */
(define query_block_with_physical_requirement_choices (lambda (block)
	(begin
		(define requirement (qassoc_get (qb_facts block) (quote membership_requirement) nil))
		(if (nil? requirement)
			block
			(begin
				(define strategy (physical_candidate_membership_strategy requirement))
				(define candidates (membership_cost_options_for_telemetry requirement))
				(define reason (if (equal? strategy (quote candidate_keyset))
					(quote projected_membership_cost)
					(quote indexed_driver_probe_cost)))
				(if (equal? strategy (quote candidate_keyset))
					nil
					(record_membership_physical_choice requirement strategy reason candidates))
				(define physical_facts (merge (list
					(list
						(list (quote membership_plan_strategy) strategy)
						(list (quote membership_cost_candidates) candidates)
						(list (quote membership_cost_reason) reason))
					requirement
					(qb_facts block))))
				(make_query_block
					(qb_schema block) (qb_sources block) (qb_fields block) (qb_where block)
					(qb_group block) (qb_having block) (qb_order block) (qb_limit block)
					(qb_offset block) (qb_hidden block)
					(map (qb_stages block) (lambda (stage)
						(if (equal? (gs_id stage) (qassoc_get requirement (quote membership_stage_id) nil))
							(group_stage_with_facts stage (merge (list requirement (gs_facts stage))))
							stage)))
					physical_facts))))))

(define membership_broad_driver_probe_preferred? (lambda (block stage)
	(begin
		(define base_sources (filter (qb_sources block) source_is_base_table?))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(and (source_is_base_table? (gs_input stage))
			(and (single_source? base_sources)
				(and (query_limit_active? (qb_offset block) (qb_limit block))
					(and (order_items_belong_to_source? (car base_sources) (qb_order block))
						(expr_contains_broad_text_match? condition))))))))

(define membership_truth_projection_preferred? (lambda (block stage _guarded_alternative)
	(begin
		(define input (gs_input stage))
		(define stage_facts (merge (list (membership_candidate_work_facts stage) (gs_facts stage))))
		(define base_sources (filter (qb_sources block) source_is_base_table?))
		/* Projection from the canonical group cache currently guarantees a
		direct key map only for base-table membership stages. Derived inputs keep
		the same primitive and use indexed existence probing until their cache-key
		lineage is represented explicitly. A canonical simple UNION is also valid
		for the driver probe because each branch lowers independently against the
		current lookup key; this does not project or materialize the UNION. */
		(if (or
			(or (not (source_is_base_table? input)) (not (single_source? base_sources)))
			(not (empty_list? (group_stage_session_domain_keys stage))))
			false
			(begin
				(define candidate_estimate (planner_source_filter_estimate input
					(coalesceNil (qassoc_get stage_facts (quote condition) true) true)
					512))
				(define capped (if (nil? candidate_estimate) false
					(qassoc_get candidate_estimate (quote capped) false)))
				(define candidate_total_rows (planner_source_row_count input))
				(define candidate_rows (membership_estimated_matching_rows
					candidate_estimate candidate_total_rows candidate_total_rows))
				(define driver (car base_sources))
				(define driver_total_rows (planner_source_row_count driver))
				(define driver_estimate (planner_source_filter_estimate driver
					(membership_driver_filter (qb_where block)) 512))
				(define local_driver_rows (if (and (query_limit_active? (qb_offset block) (qb_limit block))
					(order_items_belong_to_source? driver (qb_order block)))
					(probe_limit_work_rows (qb_limit block)) nil))
				(define driver_rows (coalesceNil local_driver_rows
					(membership_estimated_work_rows driver_estimate driver_total_rows)))
				(define guarded_small_driver (and _guarded_alternative
					(and (not (query_limit_active? (qb_offset block) (qb_limit block)))
						(and (number? driver_rows) (<= driver_rows 512)))))
				(define broad_text_driver (and _guarded_alternative
					(membership_broad_driver_probe_preferred? block stage)))
				(define broad_driver (if _guarded_alternative
					(or guarded_small_driver
						(or broad_text_driver
							(and (query_limit_active? (qb_offset block) (qb_limit block))
								(and (order_items_belong_to_source? driver (qb_order block))
									(or capped
										(and (and (number? candidate_total_rows) (> candidate_total_rows 512))
											(and (number? candidate_rows)
												(and (number? candidate_total_rows)
													(>= (* candidate_rows 4) candidate_total_rows)))))))))
					false))
				/* The rewrite target depends on the surrounding shape: below a
				broad ordered OR it is the lazy driver marker used to build one
				source-key index; otherwise it is the projected candidate RecSet. */
				(define projected_choice (if broad_text_driver
					"driver_order_membership_probe"
					"candidate_keyset"))
				(define decision_id (concat "membership_carrier:" (gs_id stage)))
				(define alternatives (list "candidate_keyset" "driver_order_membership_probe"))
				(define normal_choice (if broad_driver
					"driver_order_membership_probe"
					(if (membership_projection_cost_preferred? candidate_total_rows candidate_rows driver_rows stage_facts)
						"candidate_keyset"
						"driver_order_membership_probe")))
				(define normal_projection (equal? normal_choice projected_choice))
				(define chosen_choice (planner_physical_choice decision_id normal_choice alternatives))
				(define chosen_projection (equal? chosen_choice projected_choice))
				(define forced_choice (planner_physical_override decision_id))
				(define candidate_cost (if (number? candidate_rows)
					(membership_projection_cost candidate_total_rows candidate_rows driver_rows stage_facts)
					nil))
				(define driver_cost (if (number? driver_rows)
					(membership_ordered_driver_probe_cost candidate_total_rows candidate_rows driver_rows stage_facts)
					nil))
				(planner_record_physical_decision
					(list
						(list "decision_id" decision_id)
						(list "decision" "membership_carrier")
						(list "decision_site" "truth_projection")
						(list "stage" (gs_id stage))
						(list "consumer" (if (query_block_has_aggregates? block) "aggregate" (if (query_limit_active? (qb_offset block) (qb_limit block)) "order_limit" "filter")))
						(list "chosen" chosen_choice)
						(list "selection" (if (nil? forced_choice) (if broad_driver "constraint" "cost") "forced"))
						(list "normally_chosen" normal_choice)
						(list "reason" (if (not (nil? forced_choice)) "calibration_override"
							(if guarded_small_driver "guarded_small_local_driver"
								(if broad_driver "guarded_broad_order_limit_driver"
									(if capped "capped_estimate_uses_input_lower_bound" "lowest_total_ns")))))
						(list "inputs" (list
							(list "candidate_input_rows" candidate_total_rows)
							(list "candidate_matching_rows" (if (nil? candidate_estimate) nil (qassoc_get candidate_estimate (quote rows) nil)))
							(list "candidate_rows" candidate_rows)
							(list "candidate_density" (membership_candidate_density
								candidate_total_rows candidate_rows stage_facts))
							(list "projected_driver_rows"
								(membership_projected_driver_rows candidate_total_rows candidate_rows driver_total_rows stage_facts))
							(list "driver_input_rows" driver_total_rows)
							(list "driver_rows" driver_rows)
							(list "expected_driver_rows_visited" (membership_expected_driver_rows_visited
								candidate_total_rows candidate_rows driver_rows stage_facts))
							(list "limit" (qb_limit block))
							(list "offset" (coalesceNil (qb_offset block) 0))
							(list "estimate_capped" capped)
							(list "estimate_sampled_rows" (qassoc_get candidate_estimate (quote sampled) nil))
							(list "estimate_population" (string (planner_estimate_population candidate_estimate)))
							(list "estimate_coverage" (string (planner_estimate_coverage candidate_estimate)))
							(list "selectivity_class" (if capped "unknown_or_broad" (string (qassoc_get stage_facts (quote selectivity_class) (quote unknown)))))
							(list "candidate_scan_invocations" (qassoc_get stage_facts (quote membership_candidate_scan_invocations) 1))
							(list "candidate_filter_columns" (qassoc_get stage_facts (quote membership_candidate_filter_columns) 0))
							(list "candidate_map_columns" (qassoc_get stage_facts (quote membership_candidate_map_columns) 1))
							(list "candidate_cache_map_columns" (qassoc_get stage_facts (quote membership_candidate_cache_map_columns) 2))
							(list "candidate_expression_operations" (qassoc_get stage_facts (quote membership_candidate_expression_operations) 0))
							(list "candidate_expression_depth" (qassoc_get stage_facts (quote membership_candidate_expression_depth) 0))
							(list "candidate_broad_text_match_rows" (qassoc_get stage_facts (quote membership_candidate_broad_text_match_rows) 0))
							(list "candidate_broad_text_match_bytes" (qassoc_get stage_facts (quote membership_candidate_broad_text_match_bytes) 0))
							(list "candidate_filter_value_rows" (qassoc_get stage_facts (quote membership_candidate_filter_value_rows) nil))
							(list "candidate_expression_operation_rows" (qassoc_get stage_facts (quote membership_candidate_expression_operation_rows) nil))
							(list "driver_scan_invocations" (qassoc_get stage_facts (quote membership_driver_scan_invocations) 1))
							(list "driver_filter_columns" (qassoc_get stage_facts (quote membership_driver_filter_columns) 0))
							(list "driver_map_columns" (qassoc_get stage_facts (quote membership_driver_map_columns) 0))
							(list "driver_expression_operations" (qassoc_get stage_facts (quote membership_driver_expression_operations) 0))
							(list "driver_expression_depth" (qassoc_get stage_facts (quote membership_driver_expression_depth) 0))
							(list "reuse" (qassoc_get stage_facts (quote reuse) 1))))
						(list "alternatives" (list
							(list
								(list "plan" "candidate_keyset")
								(list "status" (if (equal? chosen_choice "candidate_keyset") "chosen" "rejected"))
								(list "reason" (if (equal? chosen_choice "candidate_keyset") "selected" "higher_total_ns_or_forced_alternative"))
								(list "cost" (if (nil? candidate_cost) '() (planner_cost_explain candidate_cost))))
							(list
								(list "plan" "driver_order_membership_probe")
								(list "status" (if (equal? chosen_choice "driver_order_membership_probe") "chosen" "rejected"))
								(list "reason" (if (equal? chosen_choice "driver_order_membership_probe") "selected" "higher_total_ns_or_forced_alternative"))
								(list "cost" (if (nil? driver_cost) '() (planner_cost_explain driver_cost))))))))
				chosen_projection)))))

(define choose_membership_truth_items (lambda (block items guarded_alternative)
	(match items
		(cons item rest) (begin
			(define chosen (choose_membership_truth_expr_using block item guarded_alternative))
			(define tail (choose_membership_truth_items block rest guarded_alternative))
			(list
				(cons (nth chosen 0) (nth tail 0))
				(merge_unique (list (nth chosen 1) (nth tail 1)))))
		_ (list '() '()))))

(define membership_or_has_driver_alternative? (lambda (items)
	(reduce (coalesceNil items '()) (lambda (found item)
		(or found (not (expr_contains_membership_truth? item)))) false)))

(define choose_membership_truth_expr_using (lambda (block expr guarded_alternative)
	(begin
		(define parts (membership_truth_parts expr))
		(if (not (nil? parts))
			(begin
				(define stage (membership_truth_stage (qb_stages block) (qb_sources block) (nth parts 1)))
				(if (and (not (nil? stage))
					(membership_truth_projection_preferred? block stage guarded_alternative))
					(begin
						(define guarded_stage (if (and guarded_alternative
							(membership_broad_driver_probe_preferred? block stage))
							(group_stage_with_facts stage
								(qassoc_set (gs_facts stage) (quote guarded_broad_order_driver) true))
							stage))
						(list (driver_membership_probe_expr guarded_stage (nth parts 0)) (list (nth parts 1))))
					(list (expand_in_membership_truth_expr (nth parts 0) (nth parts 1) (nth parts 2)) '())))
			(if (and (list? expr)
				(or
					(or (equal? (car expr) (quote and)) (equal? (car expr) (symbol "and")))
					(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or")))))
				(begin
					(define is_or
						(or (equal? (car expr) (quote or)) (equal? (car expr) (symbol "or"))))
					(define below_or (or guarded_alternative
						(and is_or (membership_or_has_driver_alternative? (cdr expr)))))
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
			(define chosen_where (nth chosen 0))
			(define base_sources (filter (qb_sources block) source_is_base_table?))
			(define ordered_memberships (if (single_source? base_sources)
				(driver_memberships_for_source (car base_sources) chosen_where)
				'()))
			/* Capability checks deliberately inspect only logical source/key
			signatures. Physical RecSet scan ASTs are emitted once, during lowering. */
			(define keysets_supported (membership_keysets_supported? ordered_memberships))
			(define ordered_driver_probe (and
				(not (empty_list? removed_aliases))
				(and keysets_supported
					(and (membership_expr_has_driver_alternative? (qb_where block))
						(and (query_limit_active? (qb_offset block) (qb_limit block))
							(and (single_source? base_sources)
								(order_items_belong_to_source? (car base_sources) (qb_order block))))))))
			(define chosen_facts (if ordered_driver_probe
				(merge (list
					(list
						(list (quote membership_plan_strategy) (quote driver_order_membership_probe))
						(list (quote membership_driver_alternative) true)
						(list (quote membership_order_limit_driver) true))
					(qb_facts block)))
				(qassoc_set (qb_facts block) (quote membership_plan_strategy)
					(quote projected_membership_alternatives))))
			(define removed_stage_ids (if ordered_driver_probe
				(filter (map (filter (qb_sources block) (lambda (src)
					(contains? removed_aliases (source_alias src))))
					(lambda (src) (if (stage_output_relation? (source_relation src))
						(stage_output_relation_id (source_relation src)) nil)))
					(lambda (stage_id) (not (nil? stage_id))))
				'()))
			(define chosen_stages (if ordered_driver_probe
				(filter (qb_stages block) (lambda (stage)
					(not (contains? removed_stage_ids (gs_id stage)))))
				(qb_stages block)))
			(if (empty_list? removed_aliases)
				(make_query_block
					(qb_schema block) (qb_sources block) (qb_fields block) chosen_where
					(qb_group block) (qb_having block) (qb_order block) (qb_limit block)
					(qb_offset block) (qb_hidden block) (qb_stages block) (qb_facts block))
				(make_query_block
					(qb_schema block)
					(filter (qb_sources block) (lambda (src) (not (contains? removed_aliases (source_alias src)))))
					(qb_fields block) chosen_where (qb_group block) (qb_having block)
					(qb_order block) (qb_limit block) (qb_offset block) (qb_hidden block)
					chosen_stages
					chosen_facts))))))

(define dml_preserve_driver_membership_probe (lambda (fallback_schema expr)
	(match expr
		((symbol driver_membership_probe) stage probe)
		(list (quote dml_driver_membership_probe) fallback_schema stage probe)
		((quote driver_membership_probe) stage probe)
		(list (quote dml_driver_membership_probe) fallback_schema stage probe)
		((symbol driver_membership_subscan_probe) stage probe)
		(list (quote dml_driver_membership_probe) fallback_schema stage probe)
		((quote driver_membership_subscan_probe) stage probe)
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

/* The local, no-graph-needed shape check shared by both the reorder-time
eligibility decision below (which additionally proves the recursive boolean
shape via stage_boolean_shaped?) and the later physical consumption site in
recset_project_join_expr_for_membership, where a driver_membership_probe
marker has already survived from reorder time -- meaning the deeper
recursive proof happened once, upstream, and does not need re-deriving. */
(define recset_probe_stage_shape? (lambda (stage)
	(or (presence_probe_stage? stage) (scalar_first_probe_stage? stage))))

/* A stage's domain (gs_input) may be a plain base table, or a single-source
query-block wrapping one (the shape a correlated scalar subselect's FROM
clause unnests into, e.g. `FROM standort s WHERE s.ID = doc.standort`) --
either way there is exactly one real physical table backing the domain.
Returns that base-table source, or nil otherwise: multi-source, itself
sourced from another stage output, or -- critically -- a wrapping
query-block whose qb_sources also carries a stage-output pseudo-source for a
nested dependency (the extra entry group_stage_direct_dependencies_using_indexes
adds so that dependency's own graph edge is discoverable, e.g. a WHERE clause
with its own nested correlated EXISTS). single_source? (not single_real_source?)
is deliberate here: lower_driver_membership_probe_expr's scan_exists fallback
reconstructs that wrapping block's raw WHERE clause by hand and has no
mechanism to join in such a pseudo-source's cache, so unlike most other
single-real-source checks in this file, a nested dependency here disqualifies
the domain rather than being transparently ignored (the WHERE-term-stripped
recset path has no such restriction, since it scans the stage's own
already-prepared cache instead). */
(define recset_domain_source (lambda (input)
	(if (source_is_base_table? input)
		input
		(if (and (query_block? input) (single_source? (qb_sources input)))
			(begin
				(define real_src (car (qb_sources input)))
				(if (source_is_base_table? real_src) real_src nil))
			nil))))

/* The prepared-cache projection can also consume the common nested-stage
shape: one real base-table domain plus stage-output pseudo-sources. Keep this
separate from recset_domain_source because the raw-domain fallback cannot
reconstruct those pseudo-source joins safely. */
(define recset_cache_domain_supported? (lambda (input)
	(or (not (nil? (recset_domain_source input)))
		(and (query_block? input)
			(begin
				(define real_src (single_real_source (qb_sources input)))
				(and (not (nil? real_src)) (source_is_base_table? real_src))))
		(and (union_block? input)
			(reduce (union_branches input) (lambda (supported branch)
				(and supported (candidate_recset_branch_supported? branch)))
				true)))))

/* This eligibility check is consumed only by the WHERE-term projection
path. exists_recset_project_join_expr builds the recset from the stage's own
prepared group-cache rather than reconstructing its raw domain, so the full
recursive boolean proof is valid here even when the scalar value contains
nested stages. The separate value-embedding path remains deliberately
conservative: lower_driver_membership_probe_expr may have to rebuild a raw
domain scan and therefore must not inherit this broader proof.

A stage with residual outer refs (btw2025_accessing_after_simple) reaches
further out than its own domain for a value that isn't local to it or its
declared lookup-keys. exists_recset_project_join_expr's stage_catalog is
scoped to the discovering block's own stages and cannot resolve that, so
those stages remain excluded rather than crashing during preparation. */
(define stage_recset_domain_eligible? (lambda (graph stage requested_col)
	(if (not (group_stage? stage))
		false
		(begin
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define input (gs_input stage))
			(and
				(not (stage_has_residual_outer_refs? stage))
				(recset_probe_stage_shape? stage)
				(stage_boolean_shaped? graph stage requested_col)
				(not (empty_list? lookup_keys))
				(equal? (count lookup_keys) (count (gs_keys stage)))
				(recset_cache_domain_supported? input))))))

(define recset_domain_stage_output_source? (lambda (stages src requested_col)
	(and (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(stage_recset_domain_eligible? (stage_dependency_graph stages) stage requested_col)))))

(define first_candidate_source (lambda (stages sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (candidate_stage_output_source? stages src) src nil)))
		nil)))

(define driver_order_membership_strategy? (lambda (facts)
	(or
		(equal? (qassoc_get facts (quote membership_plan_strategy) nil) (quote driver_order_membership_probe))
		(not (nil? (qassoc_get facts (quote membership_requirement) nil))))))

(define membership_strategy? (lambda (facts)
	(or (driver_order_membership_strategy? facts)
		(or (equal? (qassoc_get facts (quote membership_plan_strategy) nil) (quote candidate_keyset))
			(equal? (qassoc_get facts (quote membership_plan_strategy) nil) (quote driver_filter_join_probe))))))

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

(define membership_estimate_broad? (lambda (facts)
	(begin
		(define estimated_rows (qassoc_get facts (quote membership_candidate_estimated_rows) nil))
		(define estimate_input (qassoc_get facts (quote membership_candidate_estimate_input) nil))
		(define capped (qassoc_get facts (quote membership_candidate_estimate_capped) false))
		(define broad_by_count (and
			(and (not (nil? estimated_rows)) (and (not (nil? estimate_input)) (> estimate_input 0)))
			(>= (* estimated_rows 4) estimate_input)))
		(or
			capped
			(or
				broad_by_count
				(equal? (qassoc_get facts (quote membership_selectivity_class) nil) (quote broad)))))))

(define broad_driver_order_membership_probe? (lambda (facts)
	(and
		(driver_order_membership_strategy? facts)
		(membership_estimate_broad? facts))))

(define ordered_driver_membership_keyset? (lambda (facts)
	(and
		(qassoc_get facts (quote membership_driver_alternative) false)
		(and
			(driver_order_membership_strategy? facts)
			(qassoc_get facts (quote membership_order_limit_driver) false)))))

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

(define membership_candidate_stage_for_source (lambda (stages local_stages src)
	(begin
		(define stage (if (stage_output_relation? (source_relation src))
			(begin
				(define stage_id (stage_output_relation_id (source_relation src)))
				(coalesceNil (stage_by_id local_stages stage_id) (stage_by_id stages stage_id)))
			(stage_for_group_cache_source stages src)))
		(if (and (group_stage? stage)
			(equal? (qassoc_get (gs_facts stage) (quote purpose) nil) (quote in_candidate)))
			stage
			nil))))

(define membership_candidate_source (lambda (stages local_stages sources)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(if (not (nil? found))
			found
			(if (nil? (membership_candidate_stage_for_source stages local_stages src)) nil src)))
		nil)))

/* Stage facts describe the lookup expression at the point where the stage was
created. Derived-table flattening may subsequently rewrite the source edge to a
base-table expression. Physical consumers must use that current edge binding;
otherwise a valid cost choice can turn back into a per-row probe merely because
the logical lookup still carries an alias which no longer exists. */
(define stage_output_key_expr? (lambda (src key_name expr)
	(match expr
		((symbol get_column) tblvar tbl_ignorecase col _col_ignorecase)
		(and (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(equal? col key_name))
		((quote get_column) tblvar tbl_ignorecase col _col_ignorecase)
		(and (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
			(equal? col key_name))
		_ false)))

(define stage_output_source_lookup_expr (lambda (src key_name)
	(reduce (split_and_terms (source_join_expr src)) (lambda (found term)
		(if (not (nil? found))
			found
			(match term
				'(op left right) (if (or (equal? op (quote equal?))
					(equal? op (quote equal??)))
					(if (stage_output_key_expr? src key_name left)
						right
						(if (stage_output_key_expr? src key_name right) left nil))
					nil)
				_ nil)))
		nil)))

(define membership_truth_probe_for_alias (lambda (alias expr)
	(begin
		(define parts (membership_truth_parts expr))
		(if (not (nil? parts))
			(if (equal? (nth parts 1) alias) (nth parts 0) nil)
			(match expr
				(cons _head tail) (reduce tail (lambda (found item)
					(if (not (nil? found)) found
						(membership_truth_probe_for_alias alias item))) nil)
				_ nil)))))

(define replace_membership_truth_for_alias (lambda (alias replacement expr)
	(begin
		(define parts (membership_truth_parts expr))
		(if (and (not (nil? parts)) (equal? (nth parts 1) alias))
			replacement
			(match expr
				(cons head tail) (cons head (map tail (lambda (item)
					(replace_membership_truth_for_alias alias replacement item))))
				_ expr)))))

(define aggregate_pushdown_lineage_expr (lambda (block expr)
	(begin
		(define pushdown (qassoc_get (qb_facts block) (quote aggregate_pushdown) nil))
		(if (nil? pushdown)
			nil
			(reduce (qassoc_get pushdown (quote key_lineage) '()) (lambda (found pair)
				(if (not (nil? found)) found
					(if (equal? (car pair) expr) (cadr pair) nil))) nil)))))

(define driver_membership_probe_for_stage (lambda (stage expr)
	(begin
		(define marker (match expr
			((symbol driver_membership_subscan_probe) marker_stage probe) (list marker_stage probe)
			((quote driver_membership_subscan_probe) marker_stage probe) (list marker_stage probe)
			_ (driver_membership_probe_term expr)))
		(if (not (nil? marker))
			(if (equal? (gs_id (nth marker 0)) (gs_id stage)) (nth marker 1) nil)
			(match expr
				(cons _head tail) (reduce tail (lambda (found item)
					(if (not (nil? found)) found
						(driver_membership_probe_for_stage stage item))) nil)
				_ nil)))))

(define rebind_driver_membership_probe (lambda (stage old_probe new_probe strategy expr)
	(begin
		(define marker (match expr
			((symbol driver_membership_subscan_probe) marker_stage probe) (list marker_stage probe)
			((quote driver_membership_subscan_probe) marker_stage probe) (list marker_stage probe)
			_ (driver_membership_probe_term expr)))
		(if (not (nil? marker))
			(if (equal? (gs_id (nth marker 0)) (gs_id stage))
				(if (equal? strategy (quote driver_filter_join_probe))
					(list (quote driver_membership_subscan_probe) stage new_probe)
					(list (quote driver_membership_probe) stage new_probe))
				expr)
			(if (equal? expr old_probe)
				new_probe
				(match expr
					(cons head tail) (cons head (map tail (lambda (item)
						(rebind_driver_membership_probe stage old_probe new_probe strategy item))))
					_ expr))))))

(define query_block_with_physical_membership_using (lambda (stages block)
	(begin
		(define physical_block (query_block_with_physical_requirement_choices block))
		(if (not (membership_strategy? (qb_facts physical_block)))
			physical_block
			(begin
				(define sources (qb_sources physical_block))
				(define local_stages (qb_stages physical_block))
				(define candidate (membership_candidate_source stages local_stages sources))
				(define stage (if (nil? candidate) nil
					(membership_candidate_stage_for_source stages local_stages candidate)))
				(planner_record_physical_decision
					(list
						(list "decision" "membership_consumer")
						(list "chosen" (if (nil? candidate) "joined_stage_fallback"
							(if (candidate_stage_recset_supported? stage) "candidate_stage_rewrite"
								"unsupported_candidate_shape")))
						(list "inputs" (list
							(list "source_count" (count sources))
							(list "candidate_source_found" (not (nil? candidate)))
							(list "candidate_stage_supported"
								(if (nil? stage) false (candidate_stage_recset_supported? stage)))))))
				(if (nil? candidate)
					physical_block
					(begin
						(if (not (candidate_stage_recset_supported? stage))
							physical_block
							(begin
								(define canonical_probe
									(membership_truth_probe_for_alias (source_alias candidate) (qb_where physical_block)))
								(define stage_probe (car (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
								(define existing_probe
									(driver_membership_probe_for_stage stage (qb_where physical_block)))
								(define raw_probe (coalesceNil canonical_probe
									(coalesceNil existing_probe stage_probe)))
								(define probe (coalesceNil
									(aggregate_pushdown_lineage_expr physical_block raw_probe)
									(coalesceNil
										(stage_output_source_lookup_expr candidate (car (group_key_cols (gs_keys stage))))
										raw_probe)))
								(planner_record_physical_decision (list
									(list "decision" "membership_probe_lineage")
									(list "chosen" (if (equal? probe raw_probe) "unchanged" "rebound"))
									(list "inputs" (list
										(list "canonical_probe_found" (not (nil? canonical_probe)))
										(list "existing_probe_found" (not (nil? existing_probe)))
										(list "pushdown_facts_found" (not (nil? (qassoc_get
											(qb_facts physical_block) (quote aggregate_pushdown) nil))))
										(list "lineage_found" (not (nil?
											(aggregate_pushdown_lineage_expr physical_block raw_probe))))))))
								(define strategy (qassoc_get (qb_facts physical_block)
									(quote membership_plan_strategy) nil))
								(define marker (driver_membership_probe_expr_for_strategy stage probe strategy))
								(define physical_where (if (not (nil? canonical_probe))
									(replace_membership_truth_for_alias
										(source_alias candidate) marker (qb_where physical_block))
									(if (not (nil? existing_probe))
										(rebind_driver_membership_probe
											stage existing_probe probe strategy (qb_where physical_block))
										(combine_where (qb_where physical_block) marker))))
								(make_query_block
									(qb_schema physical_block)
									(without_source_alias sources (source_alias candidate))
									(qb_fields physical_block)
									physical_where
									(qb_group physical_block)
									(qb_having physical_block)
									(qb_order physical_block)
									(qb_limit physical_block)
									(qb_offset physical_block)
									(qb_hidden physical_block)
									(candidate_stage_without_source (qb_stages physical_block) (gs_id stage))
									(join_optimizer_facts_without_aliases
										(qb_facts physical_block) (list (source_alias candidate)))))))))))))
/* The other probe shape a group-stage-output alias can appear as: a plain or
COALESCE-wrapped bare boolean passthrough (no ">0" count-encoding), the shape
`COALESCE((SELECT bool_expr ...) ...)` compiles into for a scalar_single
boolean probe, matched against the specific logical output alias. */
(define stage_output_boolean_probe_term? (lambda (alias term)
	(match term
		((symbol coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) _default)
		(and (equal? tblvar alias) (or (equal?? _default false) (equal?? _default true)))
		((symbol coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) _default)
		(and (equal? tblvar alias) (or (equal?? _default false) (equal?? _default true)))
		((quote coalesceNil) ((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase) _default)
		(and (equal? tblvar alias) (or (equal?? _default false) (equal?? _default true)))
		((quote coalesceNil) ((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase) _default)
		(and (equal? tblvar alias) (or (equal?? _default false) (equal?? _default true)))
		((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(equal? tblvar alias)
		((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(equal? tblvar alias)
		_ false)))

(define exists_recset_probe_term? (lambda (alias term)
	(if (stage_output_boolean_probe_term? alias term)
		true
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
			_ false))))

(define condition_has_exists_recset_probe? (lambda (alias condition)
	(match condition
		(cons head tail) (or
			(exists_recset_probe_term? alias condition)
			(reduce tail (lambda (found item) (or found (condition_has_exists_recset_probe? alias item))) false))
		_ false)))

(define stage_output_column_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(if (equal? tblvar alias) col nil)
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(if (equal? tblvar alias) col nil)
		(cons _head tail) (reduce tail (lambda (found item)
			(if (not (nil? found)) found (stage_output_column_for_alias alias item))) nil)
		_ nil)))

(define exists_recset_probe_column (lambda (alias expr)
	(if (exists_recset_probe_term? alias expr)
		(stage_output_column_for_alias alias expr)
		(match expr
			(cons _head tail) (reduce tail (lambda (found item)
				(if (not (nil? found)) found (exists_recset_probe_column alias item))) nil)
			_ nil))))

(define stage_with_primary_aggregate (lambda (stage requested_col)
	(begin
		(define selected (scalar_first_probe_aggregate stage requested_col))
		(if (nil? selected)
			stage
			(make_group_stage
				(gs_id stage)
				(gs_input stage)
				(gs_domain stage)
				(gs_keys stage)
				(cons selected (filter (gs_aggregates stage) (lambda (ag) (not (equal? ag selected)))))
				(gs_having stage)
				(gs_output stage)
				(gs_order stage)
				(gs_limit stage)
				(gs_offset stage)
				(gs_facts stage))))))

(define rewrite_required_exists_recset_probe_refs (lambda (alias expr)
	(if (exists_recset_probe_term? alias expr)
		true
		(match expr
			(cons head tail) (cons head (map tail (lambda (item) (rewrite_required_exists_recset_probe_refs alias item))))
			_ expr))))

(define rewrite_exists_recset_probe_refs (lambda (alias stage probe expr)
	(if (exists_recset_probe_term? alias expr)
		(logical_membership_probe_expr
			(stage_with_primary_aggregate stage (stage_output_column_for_alias alias expr))
			probe)
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

(define first_exists_recset_source (lambda (stages block default_alias condition)
	(begin
		(define sources (qb_sources block))
		(reduce (coalesceNil sources '()) (lambda (found src)
			(if (not (nil? found))
				found
				(begin
					(define requested_col (exists_recset_probe_column (source_alias src) condition))
					(if (and (not (nil? requested_col))
						(recset_domain_stage_output_source? stages src requested_col))
						(begin
							(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
							(if (not (nil? (first_driver_lookup_key
								stage
								(without_source_alias sources (source_alias src)))))
								src
								nil))
						nil))))
			nil))))

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
						(define exists_src (first_exists_recset_source (qb_stages block) block default_alias (qb_where block)))
						(if (nil? exists_src)
							(hybrid_reorder_query_block_using stage_catalog block)
							(begin
								(define stage (group_stage_with_stage_catalog
									(stage_by_id (qb_stages block) (stage_output_relation_id (source_relation exists_src)))
									(qb_stages block)))
								(define stage_id (gs_id stage))
								(define probe_sources (list exists_src))
								(define rewritten_sources (rewrite_scalar_first_probe_sources_using
									(qb_stages block) sources probe_sources default_alias))
								(define driver_sources (without_source_alias rewritten_sources (source_alias exists_src)))
								(define candidate_telemetry (candidate_reorder_telemetry stage driver_sources block))
								(define costed_stage (group_stage_with_facts stage
									(merge (list candidate_telemetry (gs_facts stage)))))
								(define probe (first_driver_lookup_key costed_stage driver_sources))
								(define rewrite_consumer (lambda (expr)
									(rewrite_scalar_first_probe_expr
										(qb_stages block) probe_sources default_alias expr)))
								(define base_where (rewrite_exists_recset_probe_refs
									(source_alias exists_src)
									costed_stage
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
											candidate_telemetry
											(list (list (quote membership_requirement) (list
												(list (quote access) (quote membership))
												(list (quote reuse) 1))))
											(qb_facts block))))))))
					(begin
						(define stage (stage_by_id (qb_stages block) (stage_output_relation_id (source_relation candidate))))
						(define candidate_telemetry (candidate_reorder_telemetry stage sources block))
						(define facts (merge (list
							(query_block_reorder_telemetry block)
							(list (list (quote membership_requirement)
								(qassoc_set candidate_telemetry (quote reuse) 1))))))
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
								(map (qb_stages block) (lambda (current_stage)
									(join_reorder_stage_using stage_catalog
										(if (equal? (gs_id current_stage) (gs_id stage))
											(group_stage_with_facts current_stage
												(merge (list candidate_telemetry (gs_facts current_stage))))
											current_stage))))
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
					(physicalize_membership_requirement_expr (gs_facts stage)))
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
					(aggregate_col_name_using (gs_input stage) (car (gs_aggregates stage))))
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

/* ------------------------------------------------------------------------- */
/* Boolean constant-folding on the normalized IR (todos/boolean-tautology-folding.md).
Runs on the already-decorrelated, already-merged IR (after
merge_compatible_stage_output_left_joins_ir), so two occurrences of the same
correlated EXISTS check are already the same stage-output alias and match
structurally without any further alias-normalization here. Plain 2-valued
boolean algebra is safe in this truth context: every predicate this pass
folds already reached its final IR shape via coalesceNil/comparison
encodings (e.g. EXISTS lowers to `(> (coalesceNil (get_column ...) 0) 0)`)
that are never SQL-NULL by construction. */

(define expr_head_and? (lambda (head) (or (equal? head (quote and)) (equal? head (symbol "and")))))
(define expr_head_or? (lambda (head) (or (equal? head (quote or)) (equal? head (symbol "or")))))
(define expr_head_not? (lambda (head) (or (equal? head (quote not)) (equal? head (symbol "not")))))
(define expr_head_sql_not? (lambda (head) (or (equal? head (quote sql_not)) (equal? head (symbol "sql_not")))))
(define expr_head_if? (lambda (head) (or (equal? head (quote if)) (equal? head (symbol "if")))))

/* SQL parser literals retain their spelling. Within an explicit boolean
operator, "0" and "1" have the same truth values as false and true; keep this
local to the boolean fold so string-valued expressions elsewhere remain
untouched. */
(define boolean_fold_true_literal? (lambda (expr)
	(or (literal_true? expr) (or (equal? expr "1") (equal? expr 1)))))

(define boolean_fold_false_literal? (lambda (expr)
	(or (literal_false? expr) (or (equal? expr "0") (equal? expr 0)))))

(define boolean_fold_not (lambda (folded_inner)
	(if (boolean_fold_true_literal? folded_inner)
		false
		(if (boolean_fold_false_literal? folded_inner)
			true
			(match folded_inner
				(cons head tail) (if (and (expr_head_not? head) (equal? (count tail) 1))
					(car tail)
					(list (quote not) folded_inner))
				_ (list (quote not) folded_inner))))))

/* True when some folded term's negation is also present among the folded
terms -- the (A OR NOT A) / (A AND NOT A) shape. Both sides were normalized
by this same fold pass, so a correlated EXISTS check appearing verbatim on
both sides of the connective always matches here, even when one side is
wrapped in an extra NOT. */
(define terms_have_negation_pair? (lambda (folded_terms)
	(reduce folded_terms (lambda (found term)
		(or found (contains? folded_terms (boolean_fold_not term))))
		false)))

(define boolean_fold_and_terms (lambda (folded_terms)
	(begin
		(define has_false (reduce folded_terms (lambda (found term) (or found (boolean_fold_false_literal? term))) false))
		(if has_false
			false
			(begin
				(define kept (reduce folded_terms (lambda (acc term)
					(if (boolean_fold_true_literal? term) acc (append_unique acc term)))
					'()))
				(if (terms_have_negation_pair? kept)
					false
					(match kept
						'() true
						(cons single '()) single
						_ (cons (quote and) kept))))))))

(define boolean_fold_or_terms (lambda (folded_terms)
	(begin
		(define has_true (reduce folded_terms (lambda (found term) (or found (boolean_fold_true_literal? term))) false))
		(if has_true
			true
			(begin
				(define kept (reduce folded_terms (lambda (acc term)
					(if (boolean_fold_false_literal? term) acc (append_unique acc term)))
					'()))
				(if (terms_have_negation_pair? kept)
					true
					(match kept
						'() false
						(cons single '()) single
						_ (cons (quote or) kept))))))))

(define boolean_fold_if (lambda (folded_cond folded_then folded_else)
	(if (and (boolean_fold_true_literal? folded_then) (boolean_fold_false_literal? folded_else))
		folded_cond
		(if (and (boolean_fold_false_literal? folded_then) (boolean_fold_true_literal? folded_else))
			(boolean_fold_not folded_cond)
			(list (quote if) folded_cond folded_then folded_else)))))

(define boolean_fold_expr (lambda (expr)
	(match expr
		(cons head tail) (begin
			(define folded_tail (map tail boolean_fold_expr))
			(if (and (expr_head_and? head) (>= (count folded_tail) 1))
				(boolean_fold_and_terms folded_tail)
				(if (and (expr_head_or? head) (>= (count folded_tail) 1))
					(boolean_fold_or_terms folded_tail)
					(if (and (expr_head_not? head) (equal? (count folded_tail) 1))
						(boolean_fold_not (car folded_tail))
						(if (and (expr_head_sql_not? head) (equal? (count folded_tail) 1))
							(if (or (boolean_fold_true_literal? (car folded_tail))
								(boolean_fold_false_literal? (car folded_tail)))
								(boolean_fold_not (car folded_tail))
								(list (quote sql_not) (car folded_tail)))
							(if (and (expr_head_if? head) (equal? (count folded_tail) 3))
								(boolean_fold_if (nth folded_tail 0) (nth folded_tail 1) (nth folded_tail 2))
								(cons (boolean_fold_expr head) folded_tail)))))))
		_ expr)))

(define boolean_fold_maybe_expr (lambda (expr)
	(if (nil? expr) nil (boolean_fold_expr expr))))

(define fold_stage_facts_condition (lambda (facts)
	(begin
		(define existing (qassoc_get facts (quote condition) nil))
		(if (nil? existing)
			facts
			(qassoc_set facts (quote condition) (boolean_fold_expr existing))))))

(define two_valued_boolean_expr? (lambda (graph stage expr)
	/* Numeric 0/1 have truth values inside AND/OR/NOT, but remain numeric
	values in CASE results and aggregate inputs. Only actual boolean literals
	prove that the surrounding expression has a boolean result type. */
	(if (or (literal_true? expr) (literal_false? expr))
		true
		(if (presence_bool_stage_output_expr? expr)
			true
			(match expr
				(cons head tail) (if (or (expr_head_and? head) (expr_head_or? head))
					(reduce tail (lambda (two_valued item)
						(and two_valued (two_valued_boolean_expr? graph stage item)))
						true)
					(if (expr_head_not? head)
						(and (equal? (count tail) 1)
							(two_valued_boolean_expr? graph stage (car tail)))
						(if (expr_head_if? head)
							(and (equal? (count tail) 3)
								(and (two_valued_boolean_expr? graph stage (nth tail 1))
									(two_valued_boolean_expr? graph stage (nth tail 2))))
							(if (or (equal? head (quote coalesceNil)) (equal? head (symbol "coalesceNil")))
								(and (equal? (count tail) 2)
									(and (or (literal_true? (nth tail 1))
										(literal_false? (nth tail 1)))
										(boolean_typed_expr_shaped? graph stage (car tail))))
								false))))
				_ false)))))

(define fold_boolean_stage_aggregate (lambda (graph stage ag)
	(match ag
		'(expr reduce_fn neutral) (if (two_valued_boolean_expr? graph stage expr)
			(list (boolean_fold_expr expr) reduce_fn neutral)
			ag)
		_ ag)))

(define fold_boolean_tautologies_stage (lambda (graph stage)
	(if (group_stage? stage)
		(make_group_stage
			(gs_id stage)
			(gs_input stage)
			(gs_domain stage)
			(gs_keys stage)
			(map (gs_aggregates stage) (lambda (ag)
				(fold_boolean_stage_aggregate graph stage ag)))
			(boolean_fold_maybe_expr (gs_having stage))
			(gs_output stage)
			(gs_order stage)
			(gs_limit stage)
			(gs_offset stage)
			(fold_stage_facts_condition (gs_facts stage)))
		stage)))

(define stage_output_aggregate_fold_map_for_source (lambda (original_index folded_index src)
	(begin
		(define relation (source_relation src))
		(if (not (stage_output_relation? relation))
			'()
			(begin
				(define stage_id (stage_output_relation_id relation))
				(define original (get_assoc original_index stage_id))
				(define folded (get_assoc folded_index stage_id))
				(if (or (nil? original) (nil? folded))
					'()
					(filter (map (produceN (min (count (gs_aggregates original)) (count (gs_aggregates folded))))
						(lambda (i)
							(begin
								(define original_col (aggregate_col_name_using
									(gs_input original) (nth (gs_aggregates original) i)))
								(define folded_col (aggregate_col_name_using
									(gs_input folded) (nth (gs_aggregates folded) i)))
								(if (equal? original_col folded_col)
									'()
									(list (source_alias src) original_col folded_col)))))
						(lambda (entry) (not (empty_list? entry))))))))))

/* Boolean folding can change an aggregate descriptor's physical column hash.
Keep every consumer synchronized, including stages with the implicit
cardinality aggregate where the older single-output canonicalizer cannot infer
which output was renamed. */
(define stage_output_aggregate_fold_map (lambda (block original_stages folded_stages)
	(begin
		(define original_index (stage_output_stage_index original_stages))
		(define folded_index (stage_output_stage_index folded_stages))
		(define sources (merge (list
			(qb_sources block)
			(merge (map original_stages (lambda (stage)
				(stage_semantic_input_sources (gs_input stage))))))))
		(merge (map sources (lambda (src)
			(stage_output_aggregate_fold_map_for_source original_index folded_index src)))))))

(define fold_boolean_tautologies_ir (lambda (ir)
	(begin
		(define root (ir_root ir))
		(if (not (query_block? root))
			ir
			(begin
				(define graph (stage_dependency_graph (qb_stages root)))
				(define folded_stages (map (qb_stages root) (lambda (stage)
					(fold_boolean_tautologies_stage graph stage))))
				(define aggregate_col_map (stage_output_aggregate_fold_map
					root (qb_stages root) folded_stages))
				/* Most queries contain no foldable stage aggregate. Avoid copying the
				whole stage graph when every aggregate keeps its canonical column. */
				(define rewritten_stages (if (empty_list? aggregate_col_map)
					folded_stages
					(rewrite_stage_graph_stages aggregate_col_map '() folded_stages)))
				(define folded_block (make_query_block
					(qb_schema root) (qb_sources root) (qb_fields root)
					(boolean_fold_maybe_expr (qb_where root))
					(qb_group root)
					(boolean_fold_maybe_expr (qb_having root))
					(qb_order root) (qb_limit root) (qb_offset root) (qb_hidden root)
					rewritten_stages (qb_facts root)))
				(define rewritten_block (if (empty_list? aggregate_col_map)
					folded_block
					(rewrite_stage_graph_expr aggregate_col_map '() folded_block)))
				(make_ir (ir_kind ir) rewritten_block rewritten_stages
					(ir_context_of ir) (ir_return ir)))))))

(define normalize_stage_dependencies (lambda (ir)
	(begin
		(define root_result (normalize_stage_dependencies_node (ir_root ir)))
		(canonicalize_stage_output_interfaces
			(fold_boolean_tautologies_ir
				(merge_compatible_stage_output_left_joins_ir (make_ir
					(ir_kind ir)
					(nth root_result 0)
					(if (query_block? (nth root_result 0)) (qb_stages (nth root_result 0)) (nth root_result 1))
					(ir_context_of ir)
					(ir_return ir))))))))

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

(define stage_semantic_outer_aliases (lambda (stage)
	(begin
		(define local_aliases (source_aliases (stage_semantic_input_sources (gs_input stage))))
		(define referenced_aliases (merge_unique (list
			(stage_semantic_expr_aliases (gs_input stage))
			(stage_semantic_expr_aliases (gs_domain stage))
			(stage_semantic_expr_aliases (gs_keys stage))
			(stage_semantic_expr_aliases (gs_aggregates stage))
			(stage_semantic_expr_aliases (qassoc_get (gs_facts stage) (quote condition) true)))))
		(filter referenced_aliases (lambda (alias) (not (contains? local_aliases alias)))))))

(define stage_semantic_alias_map (lambda (stage)
	(begin
		(define local_aliases (source_aliases (stage_semantic_input_sources (gs_input stage))))
		(define outer_aliases (stage_semantic_outer_aliases stage))
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

(define stage_semantic_backbone_signature (lambda (signatures stage)
	(if (not (group_stage? stage))
		(logical_stage_key stage)
		(begin
			(define alias_map (stage_semantic_alias_map stage))
			(define payload (list
				(stage_semantic_canonical_node alias_map signatures (gs_input stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_domain stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_keys stage))
				(stage_semantic_rewrite_expr alias_map signatures (qassoc_get (gs_facts stage) (quote condition) true))
				(stage_semantic_rewrite_expr alias_map signatures (gs_having stage))
				(stage_semantic_rewrite_expr alias_map signatures (gs_order stage))
				(gs_limit stage)
				(gs_offset stage)
				(stage_semantic_facts alias_map signatures (gs_facts stage))))
			(concat "stage-backbone:" (fnv_hash (serialize payload)))))))

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
	(if (not (group_stage? stage))
		nil
		(begin
			(define purpose (qassoc_get (gs_facts stage) (quote purpose) nil))
			(define signature (get_assoc signature_index (gs_id stage)))
			(if (and (equal? (count (gs_aggregates stage)) 1)
				(or (equal? purpose (quote scalar_single)) (equal? purpose (quote exists))))
				(if (equal? purpose (quote exists))
					/* EXISTS stages from different decorrelation parents can have the
					same value signature but live in different domain scopes. Derived
					rebinding records the containing query scope in the stage ID. */
					(concat signature ":scope:" (serialize (list
						(cdr (split (gs_id stage) ":derived:"))
						(qassoc_get (gs_facts stage) (quote btw2025_parent) nil))))
					signature)
				/* Alias-normalized interning is safe for base-table groups. Query-input
				stages retain distinct correlation scopes until scope equivalence can
				be proven independently of their canonical carrier name. */
				(if (and (source_is_base_table? (gs_input stage))
					(empty_list? (stage_semantic_outer_aliases stage)))
					(stage_semantic_backbone_signature signature_index stage)
					nil))))))

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
				(aggregate_col_name_using (gs_input stage) (nth source_ags i))
				target_alias
				(aggregate_col_name_using (gs_input target) (nth aligned_ags i))))))))

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
			(begin
				(define rewritten_stage (rewrite_stage_graph_stage alias_map id_map true stage))
				(aggregate_col_name_using
					(gs_input rewritten_stage)
					(rewrite_stage_graph_expr alias_map id_map ag)))))))

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

(define group_stage_with_facts (lambda (stage facts)
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
			facts))))

(define group_stage_with_initializer_owner (lambda (stage owner cache)
	(group_stage_with_facts stage
		(qassoc_set
			(qassoc_set (gs_facts stage) (quote group_cache) cache)
			(quote keytable_initializer_owner) owner))))

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

/* ------------------------------------------------------------------------- */
/* Aggregate partition pushdown                                               */

/* COUNT(*) and COUNT(non-null literal) are distributive row counts: either can
become SUM of counts over any partition of the driver. Keep the proof shared
with the physical RecSet cardinality operator so logical and physical choices
accept exactly the same aggregate descriptors. */
(define non_nil_scalar_literal? (lambda (expr)
	(or (string? expr)
		(or (number? expr)
			(or (equal? expr true) (equal? expr false))))))

(define nil_test_of_non_nil_literal? (lambda (expr)
	(match expr
		(cons nil_head (cons value '())) (and
			(or (equal? nil_head (quote nil?)) (equal? nil_head (symbol "nil?")))
			(non_nil_scalar_literal? value))
		_ false)))

(define count_every_input_row_map_expr? (lambda (expr)
	(if (equal? expr 1)
		true
		(match expr
			(cons head tail) (if (expr_head_not? head)
				(and (equal? (count tail) 1) (nil_test_of_non_nil_literal? (car tail)))
				(if (expr_head_if? head)
					(and (equal? (count tail) 3)
						(and (nil_test_of_non_nil_literal? (nth tail 0))
							(and (equal? (nth tail 1) 0) (equal? (nth tail 2) 1))))
					false))
			_ false))))

(define aggregate_counts_every_input_row? (lambda (ag)
	(match ag
		'(map_expr reduce_fn neutral) (and (equal? reduce_fn (quote +))
			(and (equal? neutral 0) (count_every_input_row_map_expr? map_expr)))
		_ false)))

(define aggregate_pushdown_count_expr? (lambda (expr)
	(if (not (aggregate_expr? expr))
		false
		(begin
			(define aggregates (extract_aggregates expr))
			(and (single_source? aggregates)
				(aggregate_counts_every_input_row? (car aggregates)))))))

(define aggregate_pushdown_count_fields? (lambda (fields)
	(and (not (empty_list? fields))
		(reduce (extract_assoc fields (lambda (_title expr) expr)) (lambda (ok expr)
			(and ok (aggregate_pushdown_count_expr? expr))) true))))

(define aggregate_pushdown_stage_sources (lambda (block)
	(filter (qb_sources block) source_is_stage_output?)))

(define aggregate_pushdown_base_sources (lambda (block)
	(filter (qb_sources block) source_is_base_table?)))

(define aggregate_pushdown_stage_source_safe? (lambda (stages src)
	(if (not (source_is_stage_output? src))
		false
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(and (group_stage? stage)
				(and (not (stage_has_residual_outer_refs? stage))
					(equal? (count (gs_keys stage))
						(count (qassoc_get (gs_facts stage) (quote lookup-keys) '())))))))))

(define aggregate_pushdown_root_shape? (lambda (block)
	(begin
		(define base_sources (aggregate_pushdown_base_sources block))
		(define stage_sources (aggregate_pushdown_stage_sources block))
		(and (single_source? base_sources)
			(and (equal? (count (qb_sources block)) (+ 1 (count stage_sources)))
				(and (not (source_outer? (car base_sources)))
					(and (or (nil? (source_join_expr (car base_sources)))
						(equal? (source_join_expr (car base_sources)) true))
						(and (aggregate_pushdown_count_fields? (qb_fields block))
							(and (empty_list? (qb_group block))
								(and (nil? (qb_having block))
									(and (empty_list? (qb_order block))
										(and (nil? (qb_limit block))
											(and (nil? (qb_offset block))
												(and (empty_list? (qb_hidden block))
													(reduce stage_sources (lambda (safe src)
														(and safe (aggregate_pushdown_stage_source_safe? (qb_stages block) src)))
														true)))))))))))))))

(define aggregate_pushdown_stage_filter_term? (lambda (stage_aliases term)
	(expr_refs_one_of_aliases? term stage_aliases)))

(define aggregate_pushdown_terms (lambda (block stage_filters)
	(begin
		(define stage_aliases (map (aggregate_pushdown_stage_sources block) source_alias))
		(filter (split_and_terms (qb_where block)) (lambda (term)
			(equal? (aggregate_pushdown_stage_filter_term? stage_aliases term) stage_filters))))))

(define aggregate_pushdown_filtered_stage_sources (lambda (block filter_terms)
	(filter (aggregate_pushdown_stage_sources block) (lambda (src)
		(reduce filter_terms (lambda (filtered term)
			(or filtered (expr_refs_alias? nil (source_alias src) term))) false)))))

(define aggregate_pushdown_key_columns (lambda (block driver filter_terms)
	(begin
		/* Every stage source remains above the aggregate partition, even when its
		boolean value is not an explicit WHERE term. Preserve all driver columns
		needed to rebind those source edges after grouping. The additional key
		cardinality is visible to the pushdown cost model below. */
		(define stage_sources (aggregate_pushdown_stage_sources block))
		(merge_unique (list
			(merge (map filter_terms (lambda (term)
				(extract_columns_for_alias driver term))))
			(merge (map stage_sources (lambda (src)
				(extract_columns_for_alias driver (source_join_expr src)))))
			(merge (map stage_sources (lambda (src)
				(extract_columns_for_alias driver
					(stage_by_id (qb_stages block)
						(stage_output_relation_id (source_relation src))))))))))))

(define aggregate_pushdown_keys (lambda (driver columns)
	(map columns (lambda (column)
		(list (quote get_column) (source_alias driver) false column false)))))

(define aggregate_pushdown_key_index (lambda (driver columns expr)
	(begin
		(define column (direct_column_name_for_alias driver expr))
		(if (nil? column)
			nil
			(reduce (produceN (count columns)) (lambda (found i)
				(if (not (nil? found)) found
					(if (equal?? (nth columns i) column) i nil))) nil)))))

(define aggregate_pushdown_rewrite_expr (lambda (driver partition_alias columns expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(begin
			(define index (aggregate_pushdown_key_index driver columns expr))
			(if (nil? index) expr
				(list (quote get_column) partition_alias false (group_key_col_name index) false)))
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(aggregate_pushdown_rewrite_expr driver partition_alias columns
			(list (quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase))
		(cons head tail) (cons
			(aggregate_pushdown_rewrite_expr driver partition_alias columns head)
			(map tail (lambda (item)
				(aggregate_pushdown_rewrite_expr driver partition_alias columns item))))
		_ expr)))

(define aggregate_pushdown_rewrite_source (lambda (driver partition_alias columns src)
	(source_with_join_expr src
		(aggregate_pushdown_rewrite_expr driver partition_alias columns (source_join_expr src)))))

(define aggregate_pushdown_rewrite_stage (lambda (driver partition_alias columns stage)
	(if (not (group_stage? stage))
		stage
		(make_group_stage
			(gs_id stage)
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_input stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_domain stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_keys stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_aggregates stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_having stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_output stage))
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_order stage))
			(gs_limit stage)
			(gs_offset stage)
			(aggregate_pushdown_rewrite_expr driver partition_alias columns (gs_facts stage))))))

(define aggregate_pushdown_rewrite_count_fields (lambda (fields partition_alias input aggregate)
	(match fields
		(cons title (cons _expr rest))
		(cons title (cons
			(list (quote aggregate)
				(list (quote get_column) partition_alias false
					(aggregate_col_name_using input aggregate) false)
				(quote +) 0)
			(aggregate_pushdown_rewrite_count_fields rest partition_alias input aggregate)))
		_ '())))

(define aggregate_pushdown_build_rewrite (lambda (ir block driver movable_terms residual_terms columns driver_rows group_rows)
	(begin
		(define keys (aggregate_pushdown_keys driver columns))
		(define residual (combine_where_terms residual_terms true))
		(define movable (combine_where_terms movable_terms true))
		(define filtered_stage_aliases
			(map (aggregate_pushdown_filtered_stage_sources block movable_terms) source_alias))
		(define partition_id (concat "aggregate-partition:" (fnv_hash (serialize (list
			(source_schema driver) (source_relation driver) keys residual)))))
		(define partition_alias (concat "__aggregate_partition_" (fnv_hash partition_id)))
		(define partition_stage (make_group_stage
			partition_id
			driver
			'()
			keys
			(list aggregate_count_descriptor)
			nil '() '() nil nil
			(list
				(list (quote condition) residual)
				(list (quote purpose) (quote aggregate_partition))
				(list (quote domain) '())
				(list (quote lookup-keys) '())
				(list (quote preserve_empty_domain) false)
				(list (quote null_semantics) (quote aggregate))
				(list (quote cardinality_mode) (quote many)))))
		(define rewritten_stages (map (qb_stages block) (lambda (stage)
			(aggregate_pushdown_rewrite_stage driver partition_alias columns stage))))
		(define rewritten_stage_sources (map (aggregate_pushdown_stage_sources block) (lambda (src)
			(aggregate_pushdown_rewrite_source driver partition_alias columns src))))
		(define all_stages (merge (list (list partition_stage) rewritten_stages)))
		(define partition_source (list partition_alias (qb_schema block)
			(make_stage_output_relation partition_id) false nil))
		(define outer_block (make_query_block
			(qb_schema block)
			(cons partition_source rewritten_stage_sources)
			(aggregate_pushdown_rewrite_count_fields
				(qb_fields block) partition_alias driver aggregate_count_descriptor)
			(aggregate_pushdown_rewrite_expr driver partition_alias columns movable)
			'() nil '() nil nil '()
			all_stages
			(qassoc_set
				(qassoc_set (qb_facts block) (quote default_alias) partition_alias)
				(quote aggregate_pushdown)
				(list
					(list (quote driver) (list (source_schema driver) (source_relation driver)))
					(list (quote keys) columns)
					(list (quote key_lineage)
						(map (produceN (count keys)) (lambda (i)
							(list (nth keys i)
								(list (quote get_column) partition_alias false
									(group_key_col_name i) false)))))
					(list (quote filtered_stage_aliases) filtered_stage_aliases)
					(list (quote estimated_driver_rows) driver_rows)
					(list (quote estimated_group_rows) group_rows)
					(list (quote estimated_compression)
						(if (and (number? group_rows) (> group_rows 0)) (/ driver_rows group_rows) nil))))))
		(if (expr_refs_alias? (source_alias driver) (source_alias driver)
			(list (qb_sources outer_block) (qb_fields outer_block) (qb_where outer_block) rewritten_stages))
			ir
			(make_ir (ir_kind ir) outer_block all_stages (ir_context_of ir) (ir_return ir))))))

(define aggregate_pushdown_logical (lambda (ir)
	(begin
		(define block (ir_root ir))
		(if (or (not (equal? (ir_return ir) (quote rows)))
			(or (not (query_block? block))
				(or (qassoc_get (qb_facts block) (quote aggregate_pushdown) false)
					(not (aggregate_pushdown_root_shape? block)))))
			ir
			(begin
				(define movable_terms (aggregate_pushdown_terms block true))
				(define residual_terms (aggregate_pushdown_terms block false))
				(define driver (car (aggregate_pushdown_base_sources block)))
				(define columns
					(aggregate_pushdown_key_columns block driver movable_terms))
				(if (or (empty_list? movable_terms) (empty_list? columns))
					ir
					(begin
						(define residual (combine_where_terms residual_terms true))
						(define driver_rows (planner_aggregate_pushdown_driver_rows driver residual))
						(define keys (aggregate_pushdown_keys driver columns))
						(define group_rows (planner_aggregate_pushdown_group_estimate driver keys driver_rows))
						(define chosen (aggregate_pushdown_cost_preferred? driver_rows group_rows))
						(define driver_rows_expr (planner_guard_runtime_binding
							(list (quote planner_aggregate_pushdown_driver_rows)
								(planner_quoted_value driver)
								(planner_quoted_value residual))))
						(define group_rows_expr (planner_guard_runtime_binding
							(list (quote planner_aggregate_pushdown_group_estimate)
								(planner_quoted_value driver)
								(planner_quoted_value keys)
								driver_rows_expr)))
						(if (planner_guarded_choice chosen
							(list (quote aggregate_pushdown_cost_preferred?)
								driver_rows_expr group_rows_expr))
							(aggregate_pushdown_build_rewrite ir block driver movable_terms residual_terms
								columns driver_rows group_rows)
							ir))))))))

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

/* Like single_source?, but stage-output pseudo-sources (the extra qb_sources
entries group_stage_direct_dependencies_using_indexes adds so a nested
dependency's own dependency-graph edge is discoverable) don't count -- a
domain query block that references one nested presence stage alongside its
own single real table is still a single-base-table domain for lowering
purposes, the nested reference is a dependency to prepare, not a second join
partner. */
(define single_real_source? (lambda (sources)
	(single_source? (filter (coalesceNil sources '()) (lambda (src) (not (source_is_stage_output? src)))))))

(define single_real_source (lambda (sources)
	(car (filter (coalesceNil sources '()) (lambda (src) (not (source_is_stage_output? src)))))))

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
		((symbol driver_membership_subscan_probe) _stage probe)
		(extract_columns_for_alias src probe)
		((quote driver_membership_subscan_probe) _stage probe)
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
		((symbol driver_membership_subscan_probe) stage probe)
		(lower_driver_membership_probe_expr (list src) (source_alias src) stage probe)
		((quote driver_membership_subscan_probe) stage probe)
		(lower_driver_membership_probe_expr (list src) (source_alias src) stage probe)
		((symbol dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((quote dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr (list src) (source_alias src) fallback_schema stage probe)
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col (list stage) probe_work_rows (quote value))
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col stages probe_work_rows (quote value))
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col (list stage) probe_work_rows (quote value))
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr (list src) (source_alias src) stage requested_col stages probe_work_rows (quote value))
		((symbol scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr (list src) (source_alias src) stage requested_col)
		((quote scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr (list src) (source_alias src) stage requested_col)
		((symbol scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr (list src) (source_alias src) stage requested_col)
		((quote scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr (list src) (source_alias src) stage requested_col)
		((symbol coalesceNil) inner default)
		(lower_column_expr_for_join_in_context
			(list src) (source_alias src)
			(list (symbol "coalesceNil") inner default) probe_work_rows)
		((quote coalesceNil) inner default)
		(lower_column_expr_for_join_in_context (list src) (source_alias src)
			(list (symbol "coalesceNil") inner default) probe_work_rows)
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
			(if (or
				(equal? (aggregate_col_name ag) requested_col)
				(equal? (aggregate_col_name_using (gs_input stage) ag) requested_col))
				ag nil)))
		nil)))

(define scalar_first_probe_key_terms (lambda (sources default_alias src keys lookup_keys)
	(begin
		(define alias (source_alias src))
		(map (produceN (count keys)) (lambda (i)
			(list (quote equal??)
				(lower_column_expr_for_alias src (nth keys i))
				(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))))

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
		(define fact_catalog (group_stage_lowering_catalog stage))
		(define probe_catalog (qassoc_get (gs_facts stage) (quote probe_catalog) '()))
		(define stage_lookup (stage_catalog_with_nested
			(merge (list all_stages
				(if (lowering_catalog? fact_catalog)
					(lowering_catalog_stages fact_catalog)
					(coalesceNil fact_catalog '()))
				probe_catalog
				(nested_stage_catalog stage)))))
		(define direct_stages (unique_stages_by_id (merge (list
			(qb_stages input)
			(stage_outputs_from_sources_using stage_lookup (qb_sources input))))))
		(if (qassoc_get (gs_facts stage) (quote promoted_probe) false)
			(map direct_stages (lambda (nested_stage)
				(if (group_stage? nested_stage)
					(group_stage_with_stage_catalog nested_stage stage_lookup)
					nested_stage)))
			direct_stages))))

(define scalar_first_query_probe_nested_stages_using_index (lambda (direct_stages closure_index)
	(unique_stages_by_id (merge (list
		direct_stages
		(merge (map direct_stages (lambda (nested_stage)
			(get_assoc closure_index (logical_stage_key nested_stage))))))))))

(define lower_direct_scalar_query_probe (lambda (input value_expr physical_max_rows)
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
						(lower_column_expr_for_alias src condition))
					(quoted_runtime_list '())
					(quoted_runtime_list '())
					0
					0
					physical_max_rows
					(cons (quote list) mapcols)
					(list (quote lambda)
						(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
						(lower_column_expr_for_alias src value_expr))
					(scalar_once_reduce_first)
					nil
					false))
			nil))))

(define lower_scalar_first_query_probe_expr_using (lambda (stage value_expr keys lookup_keys nested_stages prepare_stages inline_presence_stages physical_max_rows)
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
				nested_stages inline_presence_sources keyed_input)))
		(define probe_value_expr (if (empty_list? inline_presence_sources)
			value_expr
			(begin
				(define default_alias (qassoc_get (qb_facts keyed_input) (quote default_alias)
					(if (empty_list? (qb_sources keyed_input)) nil (source_alias (car (qb_sources keyed_input))))))
				(rewrite_scalar_first_probe_expr
					nested_stages inline_presence_sources default_alias value_expr))))
		(define raw_prepared_input
			(query_block_without_stages_after_eager_prepare_using nested_stages probe_input))
		(define prepared_input (if (empty_list? prepare_stages)
			(query_block_with_stage_catalog raw_prepared_input '())
			raw_prepared_input))
		(define bounded_prepared_input (make_query_block
			(qb_schema prepared_input)
			(qb_sources prepared_input)
			(qb_fields prepared_input)
			(qb_where prepared_input)
			(qb_group prepared_input)
			(qb_having prepared_input)
			(qb_order prepared_input)
			physical_max_rows
			0
			(qb_hidden prepared_input)
			(qb_stages prepared_input)
			(qb_facts prepared_input)))
		(define direct_probe (if (empty_list? prepare_stages)
			(lower_direct_scalar_query_probe prepared_input probe_value_expr physical_max_rows)
			nil))
		(define probe_expr (if (nil? direct_probe)
			(begin
				(define reduced (lower_query_block_as_dataset_reduce
					bounded_prepared_input
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
							(map nested_stages (lambda (nested_stage)
								(if (group_stage? nested_stage)
									(stage_prepare_call_expr nested_stage)
									(lower_stage_prepare_using nested_stages nested_stages nested_stage))))
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
			(and (group_stage? nested_stage)
				(and (or (presence_probe_stage? nested_stage)
					(not (stage_shared_prepare? nested_stage)))
					(and (scalar_or_presence_probe_stage? nested_stage)
						(and (not (stage_has_residual_outer_refs? nested_stage))
							(stage_direct_probe_cost_preferred? nested_stage probe_work_rows))))))))))

(define lower_scalar_first_query_probe_expr (lambda (all_stages stage value_expr keys lookup_keys probe_work_rows)
	(begin
		(define direct_stages (scalar_first_query_probe_direct_nested_stages all_stages stage))
		(define probe_catalog (stage_catalog_with_nested
			(merge (list all_stages direct_stages))))
		(define dependency_graph (stage_dependency_graph probe_catalog))
		(define closure_index (stage_dependency_closure_index_using_graph dependency_graph direct_stages))
		(define nested_stages
			(scalar_first_query_probe_nested_stages_using_index direct_stages closure_index))
		/* A bounded parent probe evaluates this subtree only for rows that survived
		root braking. Compare those expected probe calls with the dependent stage's
		input size; retain the group cache when repeated probes amortize its build. */
		(define inline_presence_stages (if (number? probe_work_rows)
			(bounded_scalar_query_probe_inline_presence_stages direct_stages probe_work_rows)
			'()))
		/* Once a parent is selected for direct probing, its complete dependency
		closure is owned by that probe. Preparing children separately would pay
		the carrier build cost in addition to the selected direct path. */
		(define closure_index (stage_dependency_closure_index_using_graph
			dependency_graph inline_presence_stages))
		(define inline_owned_stages
			(scalar_first_query_probe_nested_stages_using_index
				inline_presence_stages closure_index))
		(define inline_ids (stage_id_set inline_owned_stages))
		(define prepare_stages (filter nested_stages (lambda (nested_stage)
			(not (has_assoc? inline_ids (gs_id nested_stage))))))
		(lower_scalar_first_query_probe_expr_using
			stage
			value_expr
			keys
			lookup_keys
			nested_stages
			prepare_stages
			inline_presence_stages
			(bounded_probe_physical_max_rows stage)))))

/* Query-input scalar probes can occur in many projected fields after their
logical stages have merged. Emit the physical probe recipe once per block and
pass correlation keys to it instead of copying the complete recipe per field. */
(define scalar_query_probe_recipe_key (lambda (stage requested_col)
	(concat "__scalar_query_probe_" (fnv_hash (concat (gs_id stage) "\n" requested_col)))))

(define expr_contains_column_ref? (lambda (expr)
	(match expr
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) true
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase) true
		(cons head tail) (or
			(expr_contains_column_ref? head)
			(reduce tail (lambda (found item)
				(or found (expr_contains_column_ref? item))) false))
		_ false)))

/* A base-table presence stage whose lookup domain contains only literals,
parameters, or session values is invariant for one query execution. It may be
bound once outside row callbacks; a column reference would make that unsafe. */
(define query_invariant_presence_stage? (lambda (stage)
	(and (presence_probe_stage? stage)
		(and (source_is_base_table? (gs_input stage))
			(and (not (stage_has_residual_outer_refs? stage))
				(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '())
					(lambda (invariant key)
						(and invariant (not (expr_contains_column_ref? key))))
					true))))))

(define query_invariant_probe_requested_col (lambda (_stage)
	(aggregate_col_name aggregate_count_descriptor)))

(define query_invariant_probe_binding_key (lambda (stage requested_col)
	(concat "__query_invariant_probe_" (fnv_hash (concat
		(stage_prepare_backbone_signature stage) "\n" requested_col)))))

(define query_invariant_probe_binding_for_col (lambda (stage requested_col)
	(get_assoc
		(qassoc_get (gs_facts stage) (quote query_invariant_probe_bindings) '())
		requested_col)))

(define group_stage_with_query_invariant_probe_binding (lambda (stage requested_col)
	(group_stage_with_facts stage
		(qassoc_set (gs_facts stage) (quote query_invariant_probe_bindings)
			(set_assoc
				(qassoc_get (gs_facts stage) (quote query_invariant_probe_bindings) '())
				requested_col
				(symbol (query_invariant_probe_binding_key stage requested_col)))))))

(define query_invariant_probe_entries_for_stages (lambda (stages)
	(nth (reduce (filter (lowering_catalog_stages stages) query_invariant_presence_stage?)
		(lambda (state stage)
			(begin
				(define requested_col (query_invariant_probe_requested_col stage))
				(define key (query_invariant_probe_binding_key stage requested_col))
				(if (has_assoc? (nth state 1) key)
					state
					(list
						(cons (list stage requested_col) (nth state 0))
						(set_assoc (nth state 1) key true)))))
		(list '() '())) 0)))

(define query_invariant_probe_key_set (lambda (entries)
	(reduce entries (lambda (keys entry)
		(match entry
			'(stage requested_col)
			(set_assoc keys (query_invariant_probe_binding_key stage requested_col) true)
			_ keys)) '())))

(define stage_with_query_invariant_probe_binding_using (lambda (binding_keys stage)
	(if (not (query_invariant_presence_stage? stage))
		stage
		(begin
			(define requested_col (query_invariant_probe_requested_col stage))
			(if (has_assoc? binding_keys (query_invariant_probe_binding_key stage requested_col))
				(group_stage_with_query_invariant_probe_binding stage requested_col)
				stage)))))

(define stage_lookup_with_query_invariant_probe_bindings (lambda (stages entries)
	(begin
		(define binding_keys (query_invariant_probe_key_set entries))
		(make_lowering_catalog
			(map (lowering_catalog_stages stages) (lambda (stage)
				(stage_with_query_invariant_probe_binding_using binding_keys stage)))))))

(define query_invariant_probe_sources (lambda (stages sources)
	(filter (coalesceNil sources '()) (lambda (src)
		(begin
			(define stage (coalesceNil
				(source_stage_output_stage stages src)
				(stage_for_group_cache_source stages src)))
			(and (not (nil? stage)) (query_invariant_presence_stage? stage)))))))

(define query_invariant_probe_binding (lambda (entry)
	(match entry
		'(stage requested_col) (begin
			(define annotated_stage
				(group_stage_with_query_invariant_probe_binding stage requested_col))
			(list
				(quote define)
				(symbol (query_invariant_probe_binding_key stage requested_col))
				(lower_scalar_first_probe_expr '() nil annotated_stage requested_col
					(list annotated_stage) 1 (quote value))))
		_ (neumann_fail "build_queryplan" "malformed query-invariant probe binding"))))

(define query_invariant_probe_bindings (lambda (entries)
	(map (reverse entries) query_invariant_probe_binding)))

(define rewrite_query_invariant_probe_expr (lambda (expr)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(if (query_invariant_presence_stage? stage)
			(symbol (query_invariant_probe_binding_key stage requested_col))
			expr)
		((quote scalar_first_probe) stage requested_col)
		(rewrite_query_invariant_probe_expr
			(list (symbol "scalar_first_probe") stage requested_col))
		((symbol scalar_first_probe) stage requested_col _dependencies)
		(if (query_invariant_presence_stage? stage)
			(symbol (query_invariant_probe_binding_key stage requested_col))
			expr)
		((quote scalar_first_probe) stage requested_col dependencies)
		(rewrite_query_invariant_probe_expr
			(list (symbol "scalar_first_probe") stage requested_col dependencies))
		(cons head tail) (cons head (map tail rewrite_query_invariant_probe_expr))
		_ expr)))

(define query_invariant_probe_symbol_index (lambda (stages sources)
	(begin
		(define stage_index (reduce (filter (lowering_catalog_stages stages)
			query_invariant_presence_stage?) (lambda (index stage)
				(begin
					(define requested_col (query_invariant_probe_requested_col stage))
					(set_assoc index
						(concat (exists_stage_alias (gs_id stage)) "." requested_col)
						(symbol (query_invariant_probe_binding_key stage requested_col))))) '()))
		(reduce (query_invariant_probe_sources stages sources) (lambda (index src)
			(begin
				(define stage (coalesceNil
					(source_stage_output_stage stages src)
					(stage_for_group_cache_source stages src)))
				(define requested_col (query_invariant_probe_requested_col stage))
				(set_assoc index
					(concat (source_alias src) "." requested_col)
					(symbol (query_invariant_probe_binding_key stage requested_col)))))
			stage_index))))

(define rewrite_query_invariant_probe_symbols_using (lambda (index bound expr)
	(if (symbol? expr)
		(if (contains? bound expr)
			expr
			(coalesceNil (get_assoc index (string expr)) expr))
		(match expr
			((symbol get_column) alias _alias_ignorecase col _col_ignorecase)
			(coalesceNil (get_assoc index (concat alias "." col)) expr)
			((quote get_column) alias alias_ignorecase col col_ignorecase)
			(rewrite_query_invariant_probe_symbols_using index bound
				(list (symbol "get_column") alias alias_ignorecase col col_ignorecase))
			((symbol lambda) params body) (list
				(quote lambda)
				params
				(rewrite_query_invariant_probe_symbols_using index (merge (list bound params)) body))
			((symbol lambda) params body arity) (list
				(quote lambda)
				params
				(rewrite_query_invariant_probe_symbols_using index (merge (list bound params)) body)
				arity)
			((symbol quote) _value) expr
			(cons head tail) (cons
				(rewrite_query_invariant_probe_symbols_using index bound head)
				(map tail (lambda (item)
					(rewrite_query_invariant_probe_symbols_using index bound item))))
			_ expr))))

(define rewrite_query_invariant_probe_symbols (lambda (index expr)
	(rewrite_query_invariant_probe_symbols_using index '() expr)))

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
			(define inline_candidates (filter direct_stages (lambda (nested_stage)
				(and (has_assoc? direct_source_ids (gs_id nested_stage))
					(and (or (presence_probe_stage? nested_stage)
						(not (stage_shared_prepare? nested_stage)))
						(and (scalar_or_presence_probe_stage? nested_stage)
							(not (stage_has_residual_outer_refs? nested_stage))))))))
			/* Bounded consumers execute selected presence checks after root braking.
			Every other base-table group without residual outer references has a
			closed initializer. Hoist those initializers to the shared recipe scope,
			where canonical carrier collection can merge duplicate key fills and
			aggregate-column extensions before code emission. */
			(define inline_presence_stages (if bounded_consumer inline_candidates '()))
			(define inline_owned_stages
				(scalar_first_query_probe_nested_stages_using_index inline_presence_stages closure_index))
			(define inline_owned_ids (stage_id_set inline_owned_stages))
			(define hoisted_stages (filter nested_stages (lambda (nested_stage)
				(and (group_stage? nested_stage)
					(and (source_is_base_table? (gs_input nested_stage))
						(and (not (stage_has_residual_outer_refs? nested_stage))
							(not (has_assoc? inline_owned_ids (gs_id nested_stage)))))))))
			(define consumed_ids (stage_id_set (merge (list hoisted_stages inline_owned_stages))))
			(define prepare_stages (filter nested_stages (lambda (nested_stage)
				(not (has_assoc? consumed_ids (gs_id nested_stage))))))
			(list
				stage
				requested_col
				nested_stages
				hoisted_stages
				prepare_stages
				inline_presence_stages
				bounded_consumer))
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

(define scalar_query_probe_param_index_add (lambda (index key param)
	(match key
		((symbol get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(set_assoc index key param)
		((quote get_column) _tblvar _tbl_ignorecase _col _col_ignorecase)
		(set_assoc index key param)
		_ index)))

(define scalar_query_probe_param_index (lambda (lookup_keys logical_lookup_keys params)
	(begin
		(define logical_keys (if (equal? (count lookup_keys) (count logical_lookup_keys))
			logical_lookup_keys
			lookup_keys))
		(reduce (produceN (count lookup_keys)) (lambda (index i)
			(scalar_query_probe_param_index_add
				(scalar_query_probe_param_index_add index (nth lookup_keys i) (nth params i))
				(nth logical_keys i)
				(nth params i))) '()))))

/* A query-probe lambda evaluates its outer lookup keys before entering nested
stages. Replace inherited direct column references with those parameters so
dependency preparation does not emit free outer-row symbols. Derived flattening
may rename the live lookup while nested stages retain its logical predecessor;
both names therefore bind to the same parameter. */
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
		'(stage requested_col nested_stages _hoisted_stages prepare_stages inline_presence_stages bounded_consumer) (begin
			(define raw_keys (gs_keys stage))
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define logical_lookup_keys
				(qassoc_get (gs_facts stage) (quote btw2025_lookup_keys) lookup_keys))
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
			(define param_index (scalar_query_probe_param_index lookup_keys logical_lookup_keys params))
			(define invariant_entries (query_invariant_probe_entries_for_stages nested_stages))
			(define annotated_nested_lookup
				(stage_lookup_with_query_invariant_probe_bindings nested_stages invariant_entries))
			(define annotated_nested_stages (lowering_catalog_stages annotated_nested_lookup))
			(define input_sources (qb_sources (gs_input stage)))
			(define input_default_alias (qassoc_get (qb_facts (gs_input stage)) (quote default_alias)
				(if (empty_list? input_sources) nil (source_alias (car input_sources)))))
			(define invariant_sources
				(query_invariant_probe_sources annotated_nested_lookup input_sources))
			(define invariant_symbol_index
				(query_invariant_probe_symbol_index annotated_nested_lookup input_sources))
			(define rewrite_invariants (lambda (expr)
				(rewrite_query_invariant_probe_symbols invariant_symbol_index
					(rewrite_query_invariant_probe_expr
						(rewrite_scalar_first_probe_expr
							annotated_nested_lookup invariant_sources input_default_alias expr)))))
			(define rewrite_bound (lambda (expr)
				(rewrite_invariants
					(rewrite_scalar_query_probe_params param_index expr))))
			/* Stage descriptors retain their aggregate columns. Only the scalar
			recipe expression may replace a stage output with its query binding. */
			(define rewrite_bound_stage (lambda (expr)
				(rewrite_query_invariant_probe_expr
					(rewrite_scalar_query_probe_params param_index expr))))
			(define bound_stage (rewrite_bound_stage stage))
			(define bound_keys (gs_keys bound_stage))
			(define bound_value_expr (rewrite_bound value_expr))
			(define bound_nested_stages (map annotated_nested_stages (lambda (nested_stage)
				(rewrite_bound_stage nested_stage))))
			(define bound_prepare_stages (map prepare_stages (lambda (prepare_stage)
				(rewrite_bound_stage
					(coalesceNil (stage_by_id annotated_nested_lookup (gs_id prepare_stage)) prepare_stage)))))
			(define bound_inline_presence_stages (map inline_presence_stages (lambda (presence_stage)
				(rewrite_bound_stage
					(coalesceNil (stage_by_id annotated_nested_lookup (gs_id presence_stage)) presence_stage)))))
			(define probe_expr
				(rewrite_query_invariant_probe_symbols invariant_symbol_index
					(lower_scalar_first_query_probe_expr_using bound_stage bound_value_expr bound_keys params
						bound_nested_stages bound_prepare_stages bound_inline_presence_stages
						(bounded_probe_physical_max_rows bound_stage))))
			(define memoized_probe_expr (if bounded_consumer
				(list
					(physical_query_session_symbol)
					"get_or_compute_scoped"
					(physical_query_scope_symbol)
					(list (quote concat)
						(concat (scalar_query_probe_recipe_key stage requested_col) ":")
						(list (quote serialize) (cons (quote list) params)))
					(list (quote lambda) '() probe_expr))
				probe_expr))
			(list
				(quote define)
				(symbol (scalar_query_probe_recipe_key stage requested_col))
				(list (quote lambda) params memoized_probe_expr)))
		_ (neumann_fail "build_queryplan" "malformed scalar query probe recipe plan"))))

(define scalar_query_probe_recipe_bindings (lambda (plans)
	(map (coalesceNil plans '()) scalar_query_probe_recipe_binding)))

(define scalar_query_probe_recipe_hoisted_stages (lambda (plans)
	(unique_stages_by_id (merge (map (coalesceNil plans '()) (lambda (plan)
		(match plan
			'(_stage _requested_col _nested_stages hoisted_stages _prepare_stages _inline_presence_stages _bounded_consumer) hoisted_stages
			_ '())))))))

(define scalar_query_probe_recipe_prepare_exprs (lambda (plans)
	(begin
		(define stages (scalar_query_probe_recipe_hoisted_stages plans))
		(define shared_stages (filter stages stage_shared_prepare?))
		(define direct_stages (filter stages (lambda (stage) (not (stage_shared_prepare? stage)))))
		(merge (list
			(lazy_stage_prepare_bindings stages shared_stages)
			(map shared_stages stage_prepare_call_expr)
			(lower_unique_stage_prepares_using direct_stages direct_stages direct_stages)
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
		(define probe_term (list (quote equal??) key_expr probe))
		(define probe_work_rows (planner_row_count_after_selectivity
			src (list src) (source_alias src) probe_term 1))
		(define filtercols (merge_unique (list
			(extract_columns_for_alias src condition)
			(extract_columns_for_alias src key_expr))))
		(list (quote scan_exists)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(list (quote and)
					(lower_column_expr_for_alias_in_context src condition probe_work_rows)
					(list (quote equal??)
						(lower_column_expr_for_alias src key_expr)
						(lower_column_expr_for_join sources default_alias probe))))))))

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

/* A scalar-first probe whose input is a derived table wrapping exactly one
base table, keyed by one of that table's own unique-key columns, is a
foreign-key point lookup: the same (small) set of key values recurs across
many outer rows (e.g. a dokument row's mandant/standort/... FK). Direct
per-row probing (lower_scalar_first_query_probe_expr) recomputes the value
fresh for every outer row even when many rows share the same key. When the
cost model prefers a reusable carrier over N direct probes -- the same
comparison already used to decide whether a nested presence stage should be
inlined -- lower this as a group keytable instead, built once and read via
O(1) point lookups. Multi-column and non-unique-key domains keep using the
direct probe; only the single-unique-column case is eligible here.

The stage's key domain may include session reads (e.g. a permission check
correlated to the current user) alongside the true per-outer-row key. Those
are constant for the whole query execution and are not part of what makes a
row-to-row carrier worth building; group_stage_session_domain_keys already
identifies them for other group-stage lowering, so the same rows/columns
carrier semantics apply here. */
/* Locate the remaining, single non-session source key once; reusable scalar
and RecSet carriers additionally verify its uniqueness below. */
(define scalar_first_probe_row_key_index (lambda (stage src keys)
	(begin
		(define session_indices (filter
			(map (group_stage_session_domain_keys stage) (lambda (expr) (group_key_expr_index keys expr)))
			(lambda (idx) (not (nil? idx)))))
		(define row_indices (filter (produceN (count keys)) (lambda (i) (not (contains? session_indices i)))))
		(if (not (equal? (count row_indices) 1))
			nil
			(begin
				(define idx (car row_indices))
				(define col (direct_column_name_for_alias src (nth keys idx)))
				(if (nil? col) nil idx))))))

(define scalar_first_probe_keytable_key_index (lambda (stage src keys)
	(begin
		(define idx (scalar_first_probe_row_key_index stage src keys))
		(if (nil? idx)
			nil
			(begin
				(define col (direct_column_name_for_alias src (nth keys idx)))
				(define primary_key (source_primary_key_columns src))
				(define unique_keys (merge (list
					(source_unique_key_sets src)
					(if (empty_list? primary_key) '() (list primary_key)))))
				(if (reduce unique_keys (lambda (found key_cols)
					(or found (and (equal? (count key_cols) 1) (equal?? (car key_cols) col))))
					false)
					idx
					nil))))))

/* stage_direct_probe_cost_preferred? only weighs the two costs against each
other when probe_work_rows is a known number; planner_direct_presence_probe_preferred?
short-circuits to false otherwise, purely because (number? probe_rows) fails,
not because the comparison came out in the carrier's favor. Treating "not
known to be direct-preferred" as "carrier preferred" is wrong: with no row
estimate, the carrier's own cost (input_rows, e.g. a large base table) was
never checked either, and a keytable over the whole source can be far more
expensive than any number of direct probes. Only activate this path when we
actually have a probe-count estimate to compare against; otherwise keep the
existing, already-safe direct probe. */
(define scalar_first_probe_keytable_cost_preferred? (lambda (stage probe_work_rows)
	(and (number? (planner_literal_value probe_work_rows))
		(not (stage_direct_probe_cost_preferred? stage probe_work_rows)))))

(define scalar_first_probe_keytable_eligible? (lambda (stage src keys probe_work_rows)
	/* Unlike lower_direct_scalar_query_probe, this path builds the keytable via
	lower_group_stage_prepare_using, the same general machinery any group-stage
	(including ones with nested dependent stages, like a nested EXISTS check)
	already uses. So qb_stages on src need not be empty here -- only the shape
	of src's own group/having/order/limit and its join-key's uniqueness matter. */
	(and (single_source? (qb_sources src))
		(and (source_is_base_table? (car (qb_sources src)))
			(and (empty_list? (qb_group src))
				(and (nil? (qb_having src))
					(and (empty_list? (qb_order src))
						(and (nil? (qb_limit src)) (nil? (qb_offset src))
							(and (not (nil? (scalar_first_probe_keytable_key_index stage (car (qb_sources src)) keys)))
								(scalar_first_probe_keytable_cost_preferred? stage probe_work_rows)))))))))))

/* untangle_query's derived-table inlining commonly collapses a simple
`FROM (SELECT ... FROM t WHERE t.pk = domain) alias` down to gs_input being
t itself: there is no wrapping query-block left once the derived table's
single source has been inlined. That shape is eligible for the same keytable
treatment as the query-block case; there is just no (qb_sources src) to look
through to reach the base table -- src already is it. */
(define scalar_first_probe_keytable_eligible_base? (lambda (stage src keys probe_work_rows)
	(and (not (nil? (scalar_first_probe_keytable_key_index stage src keys)))
		(scalar_first_probe_keytable_cost_preferred? stage probe_work_rows))))

(define lower_keytable_scalar_first_probe_expr (lambda (all_stages stage requested_col resolved_lookup_key physical_max_rows)
	(begin
		(define probe_catalog (qassoc_get (gs_facts stage) (quote probe_catalog) '()))
		(define stage_catalog (stage_catalog_with_nested
			(merge (list all_stages probe_catalog (nested_stage_catalog stage)))))
		(define physical_stage (if (empty_list? probe_catalog)
			stage
			(group_stage_with_stage_catalog stage stage_catalog)))
		(define cache (group_stage_cache physical_stage))
		(define cache_schema (group_cache_schema cache))
		(define cache_relation (group_cache_relation cache))
		(list (quote begin)
			(lower_group_stage_prepare_using stage_catalog stage_catalog physical_stage)
			(list (quote scan_order)
				'(session "__memcp_tx")
				(list (quote table) cache_schema cache_relation)
				(cons (quote list) (list "k0"))
				(list (quote lambda) (list (quote __kt_k0))
					(list (quote equal??) (quote __kt_k0) resolved_lookup_key))
				(cons (quote list) '())
				(cons (quote list) '())
				0
				0
				physical_max_rows
				(cons (quote list) (list requested_col))
				(list (quote lambda) (list (symbol requested_col)) (symbol requested_col))
				(scalar_once_reduce_first)
				nil
				false)))))

/* A scalar-first probe whose own value is presence-shaped (the same shape
presence_bool_stage_output_expr? already recognizes for the group-stage-prepare
path) can additionally be served by a RecSet: build the same group-cache
keytable once, then materialize a RecSet of the domain rows where the cached
value is true, and turn every driving row's check into an O(1) closure call
via recset_key_index instead of a per-row scan_order point-read. Only
presence-shaped values qualify -- recset_key_index answers membership, it
cannot return an arbitrary scalar the way the keytable point-read can. */
/* Calibrated cost source: tools/costgen (see planner_recset_carrier_cost).
RecSet's per-driving-row cost is folded entirely into its one-pass build --
unlike the keytable carrier, there is no separate per-row read term because
recset_project_join already visits every driving row once while building the
membership set. */
(define scalar_first_probe_recset_cost_preferred? (lambda (stage probe_work_rows)
	(and (number? (planner_literal_value probe_work_rows))
		(begin
			(define probe_rows (planner_literal_value probe_work_rows))
			(define input_rows (planner_stage_input_rows (gs_input stage)))
			(and (number? input_rows)
				(begin
					(define recset_cost (planner_recset_carrier_cost input_rows probe_rows))
					(define direct_cost (planner_direct_presence_probe_cost probe_rows))
					(define keytable_cost (planner_presence_carrier_cost input_rows probe_rows))
					(define chosen (if (planner_cost_better? recset_cost direct_cost)
						(if (planner_cost_better? recset_cost keytable_cost)
							(quote recset_carrier)
							(quote group_keytable))
						(if (planner_cost_better? direct_cost keytable_cost)
							(quote direct_subscan)
							(quote group_keytable))))
					(planner_record_physical_decision
						(list
							(list "decision_id" (concat "scalar_presence_carrier:" (gs_id stage)))
							(list "decision" "scalar_presence_carrier")
							(list "stage" (gs_id stage))
							(list "chosen" (string chosen))
							(list "reason" "lowest_total_ns")
							(list "inputs" (list
								(list "domain_rows" input_rows)
								(list "probe_rows" probe_rows)))
							(list "alternatives" (map
								(list
									(list (quote recset_carrier) recset_cost)
									(list (quote group_keytable) keytable_cost)
									(list (quote direct_subscan) direct_cost))
								(lambda (candidate)
									(list
										(list "plan" (string (car candidate)))
										(list "status" (if (equal? (car candidate) chosen) "chosen" "rejected"))
										(list "reason" (if (equal? (car candidate) chosen) "lowest_total_ns" "higher_total_ns_or_memory_tiebreak"))
										(list "cost" (planner_cost_explain (cadr candidate)))))))))
					(and
						(planner_cost_better? recset_cost (planner_direct_presence_probe_cost probe_rows))
						(planner_cost_better? recset_cost (planner_presence_carrier_cost input_rows probe_rows)))))))))

(define scalar_first_probe_recset_eligible? (lambda (graph stage src keys probe_work_rows requested_col)
	(and (single_real_source? (qb_sources src))
		(and (source_is_base_table? (single_real_source (qb_sources src)))
			(and (empty_list? (qb_group src))
				(and (nil? (qb_having src))
					(and (empty_list? (qb_order src))
						(and (nil? (qb_limit src)) (nil? (qb_offset src))
							(and (not (nil? (scalar_first_probe_keytable_key_index stage (single_real_source (qb_sources src)) keys)))
								(and (stage_boolean_shaped? graph stage requested_col)
									(scalar_first_probe_recset_cost_preferred? stage probe_work_rows)))))))))))

(define scalar_first_probe_recset_eligible_base? (lambda (graph stage src keys probe_work_rows requested_col)
	(and (not (nil? (scalar_first_probe_keytable_key_index stage src keys)))
		(and (stage_boolean_shaped? graph stage requested_col)
			(scalar_first_probe_recset_cost_preferred? stage probe_work_rows)))))

(define recset_scalar_first_probe_lookup_key (lambda (stage)
	(concat "__recset_probe_" (fnv_hash (gs_id stage)))))

(define physical_query_session_symbol (lambda ()
	(symbol "__physical_query_session")))

(define physical_query_scope_symbol (lambda ()
	(symbol "__physical_query_scope")))

(define physical_query_tx_symbol (lambda ()
	(symbol "__physical_query_tx")))

(define lower_recset_stage_prepare_once_expr (lambda (stage_catalog stage)
	(list
		(physical_query_session_symbol)
		"get_or_compute_scoped"
		(physical_query_scope_symbol)
		(stage_prepare_key stage)
		(list (quote lambda) '()
			(list (quote !begin)
				(lower_group_stage_prepare_using stage_catalog stage_catalog stage)
				true)))))

(define lower_recset_scalar_first_probe_expr (lambda (all_stages stage requested_col resolved_lookup_key)
	(begin
		(define probe_catalog (qassoc_get (gs_facts stage) (quote probe_catalog) '()))
		(define stage_catalog (stage_catalog_with_nested
			(merge (list all_stages probe_catalog (nested_stage_catalog stage)))))
		(define physical_stage (if (empty_list? probe_catalog)
			stage
			(group_stage_with_stage_catalog stage stage_catalog)))
		(define cache (group_stage_cache physical_stage))
		(define cache_schema (group_cache_schema cache))
		(define cache_relation (group_cache_relation cache))
		(define lookup_key (recset_scalar_first_probe_lookup_key stage))
		(list (quote begin)
			(lower_recset_stage_prepare_once_expr stage_catalog physical_stage)
			(list (quote apply)
				(list
					(physical_query_session_symbol)
					"get_or_compute_scoped"
					(physical_query_scope_symbol)
					lookup_key
					(list (quote lambda) '()
						(list (quote recset_key_index)
							(list (quote session) "__memcp_tx")
							(list (quote scan_recset)
								(list (quote session) "__memcp_tx")
								(list (quote table) cache_schema cache_relation)
								(quoted_runtime_list (list requested_col))
								(list (quote lambda) (list (symbol requested_col))
									(list (quote equal??) (symbol requested_col) true)))
							(quoted_runtime_list (list "k0")))))
				(list (quote list) resolved_lookup_key))))))

/* Select one physical realization at the consumer which owns this probe.
Logical decorrelation contributes the stage shape; the current scan node
contributes its work estimate. Emitters below only implement the selected
operator and do not repeat policy gates, so future carriers join this one
choice table instead of adding another promotion path. */
(define scalar_first_probe_physical_operator (lambda (graph stage src keys probe_work_rows requested_col probe_semantics)
	(if (union_block? src)
		(quote union-probe)
		(if (query_block? src)
			(if (and (equal? probe_semantics (quote truth))
				(scalar_first_probe_recset_eligible?
					graph stage src keys probe_work_rows requested_col))
				(quote recset)
				(if (scalar_first_probe_keytable_eligible? stage src keys probe_work_rows)
					(quote keytable)
					(quote query-scan)))
			(if (source_is_base_table? src)
				(if (and (equal? probe_semantics (quote truth))
					(scalar_first_probe_recset_eligible_base?
						graph stage src keys probe_work_rows requested_col))
					(quote recset)
					(if (scalar_first_probe_keytable_eligible_base? stage src keys probe_work_rows)
						(quote keytable)
						(quote table-scan)))
				(quote unsupported))))))

(define scalar_first_probe_carrier_source (lambda (src)
	(if (query_block? src)
		(single_real_source (qb_sources src))
		src)))

(define scalar_first_probe_query_invariant? (lambda (stage requested_col)
	(and (presence_probe_stage? stage)
		(not (nil? (query_invariant_probe_binding_for_col stage requested_col))))))

(define query_invariant_scalar_first_probe_key (lambda (stage requested_col)
	(concat
		(if (presence_probe_stage? stage) "__query_presence_probe_" "__query_scalar_probe_")
		(fnv_hash (serialize (list (gs_id stage) requested_col))))))

(define lower_query_invariant_scalar_first_probe_expr (lambda (stage requested_col expr)
	(list
		(physical_query_session_symbol)
		"get_or_compute_scoped"
		(physical_query_scope_symbol)
		(query_invariant_scalar_first_probe_key stage requested_col)
		(list (quote lambda) '() expr))))

(define lower_table_scalar_first_probe_expr (lambda (sources default_alias src stage value_expr keys lookup_keys order_exprs dirs offset_value physical_max_rows)
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
				(cons (quote and)
					(cons
						(lower_column_expr_for_alias src condition)
						(scalar_first_probe_key_terms sources default_alias src keys lookup_keys))))
			(cons (quote list) order_cols)
			(cons (quote list) dirs)
			0
			(coalesceNil offset_value 0)
			physical_max_rows
			(cons (quote list) mapcols)
			(list (quote lambda)
				(map mapcols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(lower_column_expr_for_alias src value_expr))
			(scalar_once_reduce_first)
			nil
			false))))

(define lower_scalar_first_probe_expr (lambda (sources default_alias stage requested_col all_stages probe_work_rows probe_semantics)
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
		(define physical_max_rows (bounded_probe_physical_max_rows stage))
		/* Callers with a more precise pre/post-limit estimate supply it directly.
		Older lowering paths still provide a real fallback from their visible driver
		sources instead of silently disabling every cost-based carrier with nil. */
		(define effective_probe_work_rows (if (number? (planner_literal_value probe_work_rows))
			probe_work_rows
			(probe_context_row_count sources)))
		(define probe_catalog (qassoc_get (gs_facts stage) (quote probe_catalog) '()))
		(define lowering_catalog (group_stage_lowering_catalog stage))
		(define probe_stages (stage_catalog_with_nested
			(merge (list
				all_stages
				probe_catalog
				(if (lowering_catalog? lowering_catalog)
					(lowering_catalog_stages lowering_catalog)
					'())
				(list stage)))))
		(define graph (stage_dependency_graph probe_stages))
		(define operator (scalar_first_probe_physical_operator
			graph stage src keys effective_probe_work_rows requested_col probe_semantics))
		(define lowered (match operator
			(symbol union-probe)
			(list (quote if)
				(lower_exists_union_probe_expr
					sources default_alias (union_branches src)
					(car lookup_keys) probe_stages)
				1
				nil)
			(symbol recset) (begin
				(define carrier_src (scalar_first_probe_carrier_source src))
				(define key_index (scalar_first_probe_keytable_key_index stage carrier_src keys))
				(lower_recset_scalar_first_probe_expr
					probe_stages stage requested_col
					(lower_column_expr_for_join sources default_alias
						(nth lookup_keys key_index))))
			(symbol keytable) (begin
				(define carrier_src (scalar_first_probe_carrier_source src))
				(define key_index (scalar_first_probe_keytable_key_index stage carrier_src keys))
				(lower_keytable_scalar_first_probe_expr
					probe_stages
					stage
					requested_col
					(lower_column_expr_for_join sources default_alias
						(nth lookup_keys key_index))
					physical_max_rows))
			(symbol query-scan)
			(lower_scalar_first_query_probe_expr
				probe_stages
				stage
				value_expr
				keys
				(map lookup_keys (lambda (key) (lower_column_expr_for_join sources default_alias key)))
				probe_work_rows)
			(symbol table-scan)
			(lower_table_scalar_first_probe_expr
				sources default_alias src stage value_expr keys lookup_keys
				order_exprs dirs offset_value physical_max_rows)
			_ (neumann_fail "build_queryplan" "scalar-first probe has no physical operator")))
		(if (scalar_first_probe_query_invariant? stage requested_col)
			(lower_query_invariant_scalar_first_probe_expr stage requested_col lowered)
			lowered)))
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
				(cons (quote and)
					(cons (lower_column_expr_for_alias src condition) key_terms)))
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
		(define physical_max_rows (bounded_probe_physical_max_rows stage))
		(list (quote scan_order)
			'(session "__memcp_tx")
			(source_table_expr src)
			(cons (quote list) filtercols)
			(list (quote lambda)
				(map filtercols (lambda (col) (symbol (concat (source_alias src) "." col))))
				(cons (quote and)
					(cons
						(lower_column_expr_for_alias src condition)
						(scalar_first_probe_key_terms sources default_alias src keys lookup_keys))))
			'(list)
			'(list)
			0
			0
			physical_max_rows
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
		((symbol driver_membership_subscan_probe) _stage probe)
		(collect_join_columns_acc sources default_alias target_alias probe columns_by_alias)
		((quote driver_membership_subscan_probe) _stage probe)
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

(define lower_column_expr_for_join_in_context (lambda (sources default_alias expr probe_work_rows)
	(match expr
		((symbol driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((quote driver_membership_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((symbol driver_membership_subscan_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((quote driver_membership_subscan_probe) stage probe)
		(lower_driver_membership_probe_expr sources default_alias stage probe)
		((symbol dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr sources default_alias fallback_schema stage probe)
		((quote dml_driver_membership_probe) fallback_schema stage probe)
		(lower_dml_driver_membership_probe_expr sources default_alias fallback_schema stage probe)
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col (list stage) probe_work_rows (quote value))
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col stages probe_work_rows (quote value))
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col (list stage) probe_work_rows (quote value))
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col stages probe_work_rows (quote value))
		((symbol scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr sources default_alias stage requested_col)
		((quote scalar_aggregate_probe) stage requested_col)
		(lower_scalar_aggregate_probe_expr sources default_alias stage requested_col)
		((symbol scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr sources default_alias stage requested_col)
		((quote scalar_cardinality_probe) stage requested_col)
		(lower_scalar_cardinality_probe_expr sources default_alias stage requested_col)
		((symbol coalesceNil) inner default)
		(if (equal?? default false)
			(list (quote coalesceNil)
				(lower_column_expr_for_join_truth_context
					sources default_alias inner probe_work_rows)
				default)
			(cons (quote coalesceNil)
				(map (list inner default) (lambda (item)
					(lower_column_expr_for_join_in_context
						sources default_alias item probe_work_rows)))))
		((quote coalesceNil) inner default)
		(lower_column_expr_for_join_in_context sources default_alias
			(list (symbol "coalesceNil") inner default) probe_work_rows)
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
		(cons head tail) (cons head (map tail (lambda (item)
			(lower_column_expr_for_join_in_context sources default_alias item probe_work_rows))))
		_ expr)))

/* A RecSet represents only the rows for which a predicate is SQL TRUE. It may
replace a scalar probe only where FALSE and UNKNOWN are observationally equal:
the positive truth context of WHERE/ON, AND/OR, CASE selection, or
COALESCE(predicate, FALSE). Every other expression is a value context and must
retain the scalar's complete value, including SQL NULL. */
(define truth_context_compositional_head? (lambda (head)
	(or (equal? head (quote and))
		(or (equal? head (symbol "and"))
			(or (equal? head (quote or))
				(or (equal? head (symbol "or"))
					(or (equal? head (quote if))
						(or (equal? head (symbol "if"))
							(or (equal? head (quote optimize))
								(equal? head (symbol "optimize")))))))))))

(define lower_column_expr_for_join_truth_context (lambda (sources default_alias expr probe_work_rows)
	(match expr
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col
			(list stage) probe_work_rows (quote truth))
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col
			stages probe_work_rows (quote truth))
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col
			(list stage) probe_work_rows (quote truth))
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr sources default_alias stage requested_col
			stages probe_work_rows (quote truth))
		((symbol coalesceNil) inner default)
		(if (equal?? default false)
			(list (quote coalesceNil)
				(lower_column_expr_for_join_truth_context
					sources default_alias inner probe_work_rows)
				default)
			(lower_column_expr_for_join_in_context
				sources default_alias expr probe_work_rows))
		((quote coalesceNil) inner default)
		(lower_column_expr_for_join_truth_context sources default_alias
			(list (symbol "coalesceNil") inner default) probe_work_rows)
		(cons head tail)
		(if (truth_context_compositional_head? head)
			(cons head (map tail (lambda (item)
				(lower_column_expr_for_join_truth_context
					sources default_alias item probe_work_rows))))
			(lower_column_expr_for_join_in_context
				sources default_alias expr probe_work_rows))
		_ (lower_column_expr_for_join_in_context
			sources default_alias expr probe_work_rows))))

(define lower_column_expr_for_join (lambda (sources default_alias expr)
	(lower_column_expr_for_join_in_context sources default_alias expr nil)))

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
	(unique_left_join_key_term? default_alias src key_col term)))

(define unique_lookup_key_sets (lambda (src stages)
	(begin
		(define group_cache_stage (stage_for_group_cache_source stages src))
		(if (group_stage? group_cache_stage)
			(list (group_key_cols (gs_keys group_cache_stage)))
			(if (source_is_base_table? src)
				(source_unique_key_sets src)
				(if (stage_output_relation? (source_relation src))
					(begin
						(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
						(if (group_stage? stage) (list (group_key_cols (gs_keys stage))) '()))
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
		(define key_sets (unique_lookup_key_sets src stages))
		(define lookup_condition (lookup_probe_condition_from_sources
			sources default_alias bound_sources src condition))
		(or
			(and (scalar_aggregate_probe_stage? stage)
				(empty_list? (qassoc_get (gs_facts stage) (quote lookup-keys) '())))
			(reduce key_sets (lambda (unique key_cols)
				(or unique
					(and (not (empty_list? key_cols))
						(reduce key_cols (lambda (complete key_col)
							(and complete (reduce (split_and_terms lookup_condition)
								(lambda (found term) (or found (unique_lookup_join_term? default_alias src key_col term))) false)))
							true))))
				false)))))

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
			(lower_column_expr_for_join sources default_alias probe_condition)))
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

(define source_column_order_collation (lambda (src col)
	(if (not (source_is_base_table? src))
		"bin"
		(begin
			(define meta (find (get_schema (source_schema src) (source_relation src))
				(lambda (candidate) (equal?? (candidate "Field") col)) nil))
			(if (nil? meta)
				"bin"
				(begin
					(define collation (meta "Collation"))
					(if (or (nil? collation) (equal? collation "")) "bin" collation)))))))

(define canonical_order_relation (lambda (dir collation)
	(if (or (equal? dir <) (equal? dir >))
		(collate collation (equal? dir >))
		dir)))

(define order_relations_default (lambda (order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(_expr dir) (canonical_order_relation dir "bin")
			_ (neumann_fail "build_queryplan" "malformed ORDER BY item"))))))

/* Resolve implicit SQL directions to canonical callbacks while source metadata
is still available. Explicit COLLATE and user callbacks pass through intact. */
(define order_relations_for_source (lambda (src order_items)
	(map (coalesceNil order_items '()) (lambda (item)
		(match item
			'(expr dir) (canonical_order_relation dir
				(match expr
					((symbol get_column) tblvar tbl_ignorecase _col _col_ignorecase)
					(if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
						(source_column_order_collation src (order_column_for_alias src expr))
						"bin")
					((quote get_column) tblvar tbl_ignorecase _col _col_ignorecase)
					(if (source_alias_matches? src (source_alias src) tblvar tbl_ignorecase)
						(source_column_order_collation src (order_column_for_alias src expr))
						"bin")
					_ "bin"))
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
		(define value_expr (replace_group_expr (gs_input stage) alias grouptbl keys key_names ags (first_projection_expr (gs_output stage))))
		(define replaced_order (map (coalesceNil (gs_order stage) '()) (lambda (item)
			(match item '(expr dir) (list (replace_group_order_expr (gs_input stage) alias grouptbl keys key_names ags expr) dir)))))
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
				(lower_column_expr_for_alias group_src session_filter))
			(cons (quote list) ordercols)
			(cons (quote list) (order_relations_for_source group_src replaced_order))
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
				(lower_column_expr_for_alias src condition))
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
		(if (not (and (query_block? branch) (single_real_source? (qb_sources branch))))
			(neumann_fail "build_queryplan" "driver membership probe expects simple query-block branches")
			true)
		(define src (single_real_source (qb_sources branch)))
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
				lowered_condition)
			(list (quote list))
			(list (quote lambda) '() 1)
			(quote +)
			0
			nil
			false))))

/* A complex decorrelated membership domain cannot be reconstructed as one raw
scan, but its canonical group cache already represents the complete query-block
semantics. Probe that carrier by its canonical keys and boolean aggregate value.
This adds a consumer for the existing cache; it does not introduce a second
cache identity or a logical fallback. */
(define lower_driver_membership_cache_probe_expr (lambda (sources default_alias stage)
	(begin
		(define keys (gs_keys stage))
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(if (not (equal? (count keys) (count lookup_keys)))
			(neumann_fail "build_queryplan" "cached driver membership probe key/domain mismatch")
			true)
		(define key_names (group_key_cols keys))
		(define value_col (aggregate_col_name_using (gs_input stage) (car (gs_aggregates stage))))
		(define filtercols (merge_unique (list key_names (list value_col))))
		(define key_terms (map (produceN (count keys)) (lambda (i)
			(list (quote equal??)
				(symbol (nth key_names i))
				(lower_column_expr_for_join sources default_alias (nth lookup_keys i))))))
		(define filter_condition (combine_where_terms
			(cons (stage_recset_value_filter_term stage value_col) key_terms)
			true))
		(define raw_input (gs_input stage))
		(define stamped_catalog (qassoc_get (gs_facts stage) (quote stage_catalog) '()))
		(define stage_catalog (stage_catalog_with_nested
			(merge (list stamped_catalog (nested_stage_catalog stage)
				(if (query_block? raw_input) (query_block_stage_catalog raw_input) '())))))
		(list (quote begin)
			(lower_group_stage_prepare_using stage_catalog stage_catalog stage)
			(list (quote scan_exists)
				'(session "__memcp_tx")
				(list (quote table) (group_stage_cache_schema stage) (group_stage_cache_relation stage))
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols symbol)
					filter_condition))))))

/* driver_membership_probe markers reach physical lowering through two
routes: as a WHERE-term (stripped by driver_membership_for_source, built via
recset_project_join_expr_for_membership) and, when the probed value feeds
directly into a projected column or a further expression (e.g. a derived
table's own boolean field), as a value embedded anywhere in an expression
tree -- dispatched here, straight from lower_column_expr_for_alias_in_context
/lower_column_expr_for_join_in_context. A simple raw domain keeps the cheaper
direct scan_exists path. A complex decorrelated domain probes its prepared
canonical cache instead, because reconstructing only part of its query tree
would lose nested-stage and join semantics. */
(define lower_driver_membership_probe_expr (lambda (sources default_alias stage probe)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(list (quote >)
				(cons (quote +) (map (union_branches input) (lambda (branch)
					(driver_membership_probe_branch_expr sources default_alias branch probe))))
				0)
			(begin
				(define input_src (recset_domain_source input))
				(if (nil? input_src)
					(lower_driver_membership_cache_probe_expr sources default_alias stage)
					(begin
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
								(cons (quote and)
									(cons (lower_column_expr_for_alias input_src condition) key_terms))))))))))))

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
				(if (empty_list? key_terms) true (cons (quote and) key_terms)))))))

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

/* EXISTS is two-valued, so a top-level NOT around its physical membership
marker is an exact set complement. Do not recognize membership/NOT IN stages
here: their UNKNOWN rows are in neither SQL truth set. */
(define find_single_driver_membership_probe_term (lambda (expr)
	(begin
		(define direct (driver_membership_probe_term expr))
		(if (not (nil? direct))
			direct
			(match expr
				(cons _head tail) (begin
					(define found (filter (map tail find_single_driver_membership_probe_term)
						(lambda (item) (not (nil? item)))))
					(if (single_source? found) (car found) nil))
				_ nil)))))

(define driver_membership_positive_guard_expr? (lambda (marker expr)
	(if (equal? (driver_membership_probe_term expr) marker)
		true
		(if (driver_membership_nil_guard? (nth marker 1) expr)
			true
			(match expr
				(cons head tail) (if (or (equal? head (quote and)) (equal? head (symbol "and")))
					(reduce tail (lambda (valid item)
						(and valid (driver_membership_positive_guard_expr? marker item))) true)
					false)
				_ false)))))

(define negated_driver_membership_probe_term (lambda (expr)
	(match expr
		((symbol not) inner) (begin
			(define marker (find_single_driver_membership_probe_term inner))
			(if (and (not (nil? marker))
				(and (presence_probe_stage? (car marker))
					(driver_membership_positive_guard_expr? marker inner))) marker nil))
		((quote not) inner) (begin
			(define marker (find_single_driver_membership_probe_term inner))
			(if (and (not (nil? marker))
				(and (presence_probe_stage? (car marker))
					(driver_membership_positive_guard_expr? marker inner))) marker nil))
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

(define driver_membership_formula_terms_for_source (lambda (src condition)
	(filter (map (split_and_terms (coalesceNil condition true)) (lambda (term)
		(begin
			(define positive (driver_membership_probe_term term))
			(define negative (negated_driver_membership_probe_term term))
			(define marker (coalesceNil positive negative))
			(if (or (nil? marker) (not (presence_probe_stage? (car marker))))
				nil
				(begin
					(define probe_col (direct_column_name_for_alias src (nth marker 1)))
					(if (nil? probe_col)
						nil
						(list (nth marker 0) (nth marker 1) probe_col term (not (nil? negative)))))))))
		(lambda (item) (not (nil? item))))))

(define strip_driver_membership_formula_terms (lambda (condition terms)
	(combine_where_terms
		(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
			(not (reduce terms (lambda (matched formula_term)
				(or matched (equal? term (nth formula_term 3)))) false))))
		true)))

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
						(lower_column_expr_for_alias src condition)))
				(quoted_runtime_list (list source_col))
				(source_table_expr target_src)
				(quoted_runtime_list (list target_col))))
		_ nil)))

/* A stage's own group-cache is keyed positionally by its gs_keys, named
k0, k1, ... regardless of what those key expressions look like -- so the
recset can be built by scanning the cache directly (already correctly
handling any nested dependency the stage's own preparation needs) instead
of re-deriving the stage's raw condition against its domain table. A key
projects as a real join column when the matching lookup-key resolves to a
plain column on target_src. Otherwise -- e.g. a session-variable key,
repeated identically on both sides, that resolves as a column on neither --
it folds into the filter condition as a constant equality term against the
cache's own key symbol, as long as it doesn't reach into target_src some
other way (which would mean it can't be evaluated against the cache alone). */
(define cache_recset_projection_parts_acc (lambda (key_names lookup_keys target_src source_cols target_cols constant_terms)
	(match key_names
		(cons key_name key_name_rest) (match lookup_keys
			(cons lookup_expr lookup_rest) (begin
				(define target_col (direct_column_name_for_alias target_src lookup_expr))
				(if (not (nil? target_col))
					(cache_recset_projection_parts_acc key_name_rest lookup_rest target_src
						(cons key_name source_cols)
						(cons target_col target_cols)
						constant_terms)
					(if (expr_refs_sources? nil (list target_src) lookup_expr)
						nil
						(cache_recset_projection_parts_acc key_name_rest lookup_rest target_src
							source_cols target_cols
							(cons (list (quote equal??) (symbol key_name) lookup_expr) constant_terms)))))
			_ nil)
		_ (if (empty_list? lookup_keys)
			(list (reverse source_cols) (reverse target_cols) (reverse constant_terms))
			nil))))

/* purpose='exists' presence stages cache a raw count (consumers wrap it in
">0" themselves); purpose='scalar_single' stages cache the literal boolean
value. stage_boolean_shaped? already told the caller this stage's value IS
boolean-shaped one way or the other -- this just picks the matching test. */
(define stage_recset_value_filter_term (lambda (stage value_col)
	(if (presence_probe_stage? stage)
		(list (quote >) (symbol value_col) 0)
		(list (quote equal??) (symbol value_col) true))))

(define exists_recset_project_join_expr (lambda (target_src stage)
	(begin
		(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
		(define key_names (group_key_cols (gs_keys stage)))
		(define parts (cache_recset_projection_parts_acc key_names lookup_keys target_src '() '() '()))
		(if (or (nil? parts) (empty_list? (nth parts 0)))
			nil
			(begin
				(define raw_input (gs_input stage))
				(define cache (group_stage_cache stage))
				(define cache_schema (group_cache_schema cache))
				(define cache_relation (group_cache_relation cache))
				(define value_col (aggregate_col_name_using raw_input (car (gs_aggregates stage))))
				(define filtercols (merge_unique (list (list value_col) key_names)))
				(define filter_condition (combine_where_terms
					(cons (stage_recset_value_filter_term stage value_col) (nth parts 2))
					true))
				(define stamped_catalog (qassoc_get (gs_facts stage) (quote stage_catalog) '()))
				(define stage_catalog (stage_catalog_with_nested
					(merge (list stamped_catalog (nested_stage_catalog stage)
						(if (query_block? raw_input) (query_block_stage_catalog raw_input) '())))))
				(list (quote begin)
					(lower_group_stage_prepare_using stage_catalog stage_catalog stage)
					(list (quote recset_project_join)
						'(session "__memcp_tx")
						(list (quote scan_recset)
							'(session "__memcp_tx")
							(list (quote table) cache_schema cache_relation)
							(cons (quote list) filtercols)
							(list (quote lambda)
								(map filtercols symbol)
								filter_condition))
						(quoted_runtime_list (nth parts 0))
						(source_table_expr target_src)
						(quoted_runtime_list (nth parts 1)))))))))

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

(define recset_project_join_expr_for_membership_raw (lambda (src membership)
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
				(if (recset_probe_stage_shape? stage)
					(exists_recset_project_join_expr src stage)
					nil))))))

/* Cost one physical tree edge once and return (strategy RecSet-expression).
Consumers decide whether that RecSet is their scan carrier or a membership
filter; they must not reconstruct the choice from enclosing block facts. */
(define recset_project_join_plan_for_membership_using (lambda (src membership consumer driver_rows_override)
	(begin
		(define stage (nth membership 0))
		/* Some late RecSet consumers are introduced after reorder telemetry was
		attached. Reconstruct only the candidate's scalar work from the existing
		logical stage; this is one formula walk, never an alternative plan build. */
		(define facts (merge (list (membership_candidate_work_facts stage) (gs_facts stage))))
		(define cost_facts (if (equal? consumer (quote aggregate))
			(qassoc_set facts (quote membership_consumer) (quote aggregate))
			facts))
		(define raw_expr (recset_project_join_expr_for_membership_raw src membership))
		(if (or (nil? raw_expr)
			(not (or
				(equal? (qassoc_get facts (quote purpose) nil) (quote in_membership))
				(equal? (qassoc_get facts (quote purpose) nil) (quote in_candidate)))))
			(if (nil? raw_expr) nil (list "candidate_keyset" raw_expr))
			(begin
				(define measured_estimate (planner_stage_filter_estimate (gs_input stage) 512))
				(define estimate_population (coalesceNil
					(qassoc_get facts (quote membership_candidate_estimate_population) nil)
					(planner_estimate_population measured_estimate)))
				(define estimate_coverage (coalesceNil
					(qassoc_get facts (quote membership_candidate_estimate_coverage) nil)
					(planner_estimate_coverage measured_estimate)))
				(define capped (or
					(qassoc_get facts (quote membership_candidate_estimate_capped) false)
					(qassoc_get measured_estimate (quote capped) false)))
				(define candidate_input_rows (coalesceNil
					(qassoc_get facts (quote membership_candidate_input_rows) nil)
					(planner_stage_input_rows (gs_input stage))))
				(define candidate_matching_rows (coalesceNil
					(qassoc_get facts (quote membership_candidate_estimated_rows) nil)
					(qassoc_get measured_estimate (quote rows) nil)))
				(define candidate_rows (membership_estimated_matching_rows
					(list
						(list (quote rows) candidate_matching_rows)
						(list (quote input) (coalesceNil
							(qassoc_get facts (quote membership_candidate_estimate_input) nil)
							(qassoc_get measured_estimate (quote input) nil)))
						(list (quote sampled) (coalesceNil
							(qassoc_get facts (quote membership_candidate_estimate_sampled) nil)
							(qassoc_get measured_estimate (quote sampled) nil)))
						(list (quote capped) capped)
						(list (quote population) estimate_population)
						(list (quote coverage) estimate_coverage))
					candidate_input_rows candidate_input_rows))
				/* Ordered LIMIT consumes only its local window before downstream
				operators. Cost this edge with that workload instead of a global
				driver cardinality; it is the relevant side of the plan inequality. */
				(define driver_rows (coalesceNil driver_rows_override
					(coalesceNil
						(qassoc_get facts (quote membership_driver_rows) nil)
						(planner_source_row_count src))))
				(define decision_id (concat "membership_carrier:" (gs_id stage)))
				(define known (and (number? candidate_rows) (number? driver_rows)))
				(define owns_requirement (qassoc_get facts (quote physical_membership_requirement) false))
				/* A membership below OR cannot become the scan driver without
				dropping sibling-accepted rows. For a broad ordered window retain
				the lazy probe carrier; the ordinary cost inequality owns every
				feasible carrier pair outside this semantic/Top-K constraint. */
				(define guarded_broad_order_driver
					(qassoc_get facts (quote guarded_broad_order_driver) false))
				(define normal_choice (if guarded_broad_order_driver
					"driver_order_membership_probe"
					(if known
						(if (membership_projection_cost_preferred? candidate_input_rows candidate_rows driver_rows cost_facts)
							"candidate_keyset"
							"driver_order_membership_probe")
						(if owns_requirement "driver_order_membership_probe" "candidate_keyset"))))
				(define alternatives (list "candidate_keyset" "driver_order_membership_probe"))
				(define chosen (planner_physical_choice decision_id normal_choice alternatives))
				(define forced (planner_physical_override decision_id))
				(define candidate_cost (if known (membership_projection_cost candidate_input_rows candidate_rows driver_rows cost_facts) nil))
				(define driver_cost (if known
					(membership_ordered_driver_probe_cost candidate_input_rows candidate_rows driver_rows cost_facts)
					nil))
				(planner_record_physical_decision
					(list
						(list "decision_id" decision_id)
						(list "decision" "membership_carrier")
						(list "decision_site" "recset_lowering")
						(list "stage" (gs_id stage))
						(list "consumer" (string (coalesceNil consumer
							(qassoc_get facts (quote membership_consumer) (quote filter)))))
						(list "chosen" chosen)
						(list "selection" (if (nil? forced) (if known "cost" "fallback") "forced"))
						(list "normally_chosen" normal_choice)
						(list "reason" (if (not (nil? forced)) "calibration_override"
							(if guarded_broad_order_driver "guarded_broad_order_limit_driver"
								(if known (if capped "capped_estimate_uses_input_lower_bound" "lowest_total_ns")
									(if owns_requirement "unknown_statistics_driver_fallback"
										"unknown_statistics_projection_fallback")))))
						(list "inputs" (list
							(list "candidate_input_rows" candidate_input_rows)
							(list "candidate_matching_rows" candidate_matching_rows)
							(list "candidate_rows" candidate_rows)
							(list "candidate_density" (membership_candidate_density
								candidate_input_rows candidate_rows facts))
							(list "projected_driver_rows"
								(membership_projected_driver_rows candidate_input_rows candidate_rows
									(membership_driver_input_rows driver_rows facts) facts))
							(list "driver_input_rows" (planner_source_row_count src))
							(list "driver_rows" driver_rows)
							(list "expected_driver_rows_visited" (membership_expected_driver_rows_visited
								candidate_input_rows candidate_rows driver_rows facts))
							(list "probe_rows" driver_rows)
							(list "limit" (qassoc_get facts (quote membership_order_limit) nil))
							(list "offset" (qassoc_get facts (quote membership_order_offset) 0))
							(list "estimate_capped" capped)
							(list "estimate_sampled_rows" (coalesceNil
								(qassoc_get facts (quote membership_candidate_estimate_sampled) nil)
								(qassoc_get measured_estimate (quote sampled) nil)))
							(list "estimate_population" (string estimate_population))
							(list "estimate_coverage" (string estimate_coverage))
							(list "probe_branches" (qassoc_get facts (quote membership_candidate_probe_branches) 1))
							(list "selectivity_class" (string (qassoc_get facts (quote membership_selectivity_class) (quote unknown))))
							(list "candidate_scan_invocations" (qassoc_get facts (quote membership_candidate_scan_invocations) 1))
							(list "candidate_filter_columns" (qassoc_get facts (quote membership_candidate_filter_columns) 0))
							(list "candidate_map_columns" (qassoc_get facts (quote membership_candidate_map_columns) 1))
							(list "candidate_cache_map_columns" (qassoc_get facts (quote membership_candidate_cache_map_columns) 2))
							(list "candidate_expression_operations" (qassoc_get facts (quote membership_candidate_expression_operations) 0))
							(list "candidate_expression_depth" (qassoc_get facts (quote membership_candidate_expression_depth) 0))
							(list "candidate_broad_text_match_rows" (qassoc_get facts (quote membership_candidate_broad_text_match_rows) 0))
							(list "candidate_broad_text_match_bytes" (qassoc_get facts (quote membership_candidate_broad_text_match_bytes) 0))
							(list "candidate_filter_value_rows" (qassoc_get facts (quote membership_candidate_filter_value_rows) nil))
							(list "candidate_expression_operation_rows" (qassoc_get facts (quote membership_candidate_expression_operation_rows) nil))
							(list "driver_scan_invocations" (qassoc_get facts (quote membership_driver_scan_invocations) 1))
							(list "driver_filter_columns" (qassoc_get facts (quote membership_driver_filter_columns) 0))
							(list "driver_map_columns" (qassoc_get facts (quote membership_driver_map_columns) 0))
							(list "driver_expression_operations" (qassoc_get facts (quote membership_driver_expression_operations) 0))
							(list "driver_expression_depth" (qassoc_get facts (quote membership_driver_expression_depth) 0))
							(list "reuse" (qassoc_get facts (quote reuse) 1))))
						(list "alternatives" (list
							(list
								(list "plan" "candidate_keyset")
								(list "status" (if (equal? chosen "candidate_keyset") "chosen" "rejected"))
								(list "reason" (if (equal? chosen "candidate_keyset") "selected" (if known "higher_total_ns_or_forced_alternative" "unknown_statistics_fallback")))
								(list "cost" (if (nil? candidate_cost) '() (planner_cost_explain candidate_cost))))
							(list
								(list "plan" "driver_order_membership_probe")
								(list "status" (if (equal? chosen "driver_order_membership_probe") "chosen" "rejected"))
								(list "reason" (if (equal? chosen "driver_order_membership_probe") "selected" "higher_total_ns_or_forced_alternative"))
								(list "cost" (if (nil? driver_cost) '() (planner_cost_explain driver_cost))))))))
				(list chosen raw_expr))))))

/* General expression callers preserve the established contract: only a
candidate-keyset choice replaces the marker with a projected RecSet carrier. */
(define recset_project_join_expr_for_membership_using (lambda (src membership consumer driver_rows_override)
	(begin
		(define plan (recset_project_join_plan_for_membership_using
			src membership consumer driver_rows_override))
		(if (and (not (nil? plan)) (equal? (car plan) "candidate_keyset"))
			(cadr plan)
			nil))))

(define recset_project_join_expr_for_membership (lambda (src membership)
	(recset_project_join_expr_for_membership_using src membership nil nil)))

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
		((symbol scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr '() nil stage requested_col (list stage) 1 (quote value))
		((symbol scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr '() nil stage requested_col stages 1 (quote value))
		((quote scalar_first_probe) stage requested_col)
		(lower_scalar_first_probe_expr '() nil stage requested_col (list stage) 1 (quote value))
		((quote scalar_first_probe) stage requested_col stages)
		(lower_scalar_first_probe_expr '() nil stage requested_col stages 1 (quote value))
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
			(dedupe_aggregates_by_col (merge (map (list stage requested_col) extract_aggregates))))
		((quote scalar_aggregate_probe) stage requested_col)
		(if (qassoc_get (gs_facts stage) (quote direct_group_probe) false)
			'()
			(dedupe_aggregates_by_col (merge (map (list stage requested_col) extract_aggregates))))
		((symbol count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((quote count_distinct) agg_expr) (list (count_distinct_descriptor agg_expr))
		((symbol aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		((quote aggregate) agg_expr agg_reduce agg_neutral) (list (list agg_expr agg_reduce agg_neutral))
		(cons head tail) (dedupe_aggregates_by_col (merge (map tail extract_aggregates)))
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
	(dedupe_aggregates_by_col
		(merge (extract_assoc fields (lambda (_title expr) (extract_aggregates expr)))))))

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

(define canonical_aggregate_probe_reference (lambda (stage requested_col)
	(begin
		(define ag (scalar_first_probe_aggregate stage requested_col))
		(define input (gs_input stage))
		(define stage_identity (if (source_is_base_table? input)
			(list
				(quote base-probe)
				(source_schema input)
				(source_relation input)
				(canonical_helper_expr_using input (gs_keys stage))
				(canonical_helper_expr_using input
					(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)))
			(group_stage_cache_relation stage)))
		(list
			stage_identity
			(if (nil? ag) requested_col
				(aggregate_col_name_using (gs_input stage) ag))))))

/* Aggregate descriptors still carry aliases because they must execute against
the current logical input. Persistent keytable columns must not. Resolve every
column to (source role, schema, base relation, physical column) for the name;
the enclosing carrier identity supplies the remaining query context. */
(define aggregate_col_name_using (lambda (input ag)
	/* The carrier already identifies the complete input graph, keys, and filter.
	The column therefore identifies only the aggregate recipe. Normalize source
	aliases by role so equivalent SQL aliases converge while self-join sides stay
	distinct, and do not let eager physical preparation rename the recipe. */
	(if (equal? ag aggregate_count_descriptor)
		(aggregate_col_name ag)
		(begin
			(define local_aliases (source_aliases (canonical_helper_sources input)))
			(define referenced_aliases (stage_semantic_expr_aliases ag))
			(define outer_aliases (filter referenced_aliases (lambda (alias)
				(not (contains? local_aliases alias)))))
			(define alias_map (merge (list
				(stage_semantic_alias_entries local_aliases "__aggregate_local_")
				(stage_semantic_alias_entries outer_aliases "__aggregate_outer_"))))
			(concat "agg_" (fnv_hash (serialize (list
				"canonical-aggregate-v5"
				(stage_semantic_rewrite_expr alias_map '() ag)))))))))

(define dedupe_aggregates_by_col (lambda (ags)
	(begin
		/* Aggregate descriptors are immutable planner ASTs. Keep the exact
		structural identity here instead of repeatedly serializing every prior
		descriptor into its canonical column hash. */
		(define seen (make_structural_catalog))
		(reduce (coalesceNil ags '()) (lambda (acc ag)
			(if (seen ag)
				acc
				(begin
					(seen ag true)
					(merge acc (list ag)))))
			'()))))

(define group_table_name (lambda (schema label input_identity keys condition)
	/* The readable label is not an identity. The hash covers the canonical input
	graph, source-role-aware keys, and complete filter, so equivalent aliases
	converge while self-join roles and different predicates remain separated. */
	(concat ".grp:" label ":" (fnv_hash (serialize (list
		"canonical-group-keytable-v5" schema input_identity keys condition))))))

/* Persistent helper objects must be named by the physical data they represent,
not by disposable SQL aliases. Source position remains part of the identity so
self-joins of the same base table still describe two distinct row roles. */
(define canonical_helper_sources (lambda (input)
	(if (group_stage? input)
		(canonical_helper_sources (gs_input input))
		(if (query_block? input)
			(qb_sources input)
			(if (union_block? input)
				(if (empty_list? (union_branches input)) '()
					(qb_sources (car (union_branches input))))
				(if (source_is_base_table? input) (list input) '()))))))

(define canonical_helper_source_role_from (lambda (sources ref_alias ignorecase pos)
	(match sources
		(cons src rest)
		(if (if ignorecase
			(equal?? ref_alias (source_alias src))
			(equal? ref_alias (source_alias src)))
			(list pos src)
			(canonical_helper_source_role_from rest ref_alias ignorecase (+ pos 1)))
		_ nil)))

(define canonical_helper_source_role (lambda (sources default_alias tblvar tbl_ignorecase)
	/* Resolve role and source in one pass. Calling source_for_alias and then
	searching its position performed two linear source walks per column node. */
	(canonical_helper_source_role_from sources
		(resolve_column_alias tblvar default_alias)
		tbl_ignorecase
		0)))

(define canonical_helper_expr_using (lambda (input expr)
	(begin
		(define sources (canonical_helper_sources input))
		(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
		(define sole_source (if (single_source? sources) (car sources) nil))
		/* Parser flags already define identifier semantics. Normalize insensitive
		references directly instead of consulting table metadata for every repeated
		aggregate reader; sensitive identifiers retain their exact physical name. */
		(define canonical_col (lambda (col ignorecase)
			(if (and (string? col) ignorecase) (toLower col) col)))
		(define rewrite (lambda (node)
			(if (group_stage? node)
				(list (quote aggregate-stage) (group_stage_cache_relation node))
				(match node
					((symbol scalar_first_probe) stage requested_col)
					(list (quote scalar_first_probe) (canonical_aggregate_probe_reference stage requested_col))
					((quote scalar_first_probe) stage requested_col)
					(list (quote scalar_first_probe) (canonical_aggregate_probe_reference stage requested_col))
					((symbol scalar_first_probe) stage requested_col _stages)
					(list (quote scalar_first_probe) (canonical_aggregate_probe_reference stage requested_col))
					((quote scalar_first_probe) stage requested_col _stages)
					(list (quote scalar_first_probe) (canonical_aggregate_probe_reference stage requested_col))
					((symbol scalar_aggregate_probe) stage requested_col)
					(list (quote scalar_aggregate_probe) (canonical_aggregate_probe_reference stage requested_col))
					((quote scalar_aggregate_probe) stage requested_col)
					(list (quote scalar_aggregate_probe) (canonical_aggregate_probe_reference stage requested_col))
					((symbol scalar_cardinality_probe) stage requested_col)
					(list (quote scalar_cardinality_probe) (canonical_aggregate_probe_reference stage requested_col))
					((quote scalar_cardinality_probe) stage requested_col)
					(list (quote scalar_cardinality_probe) (canonical_aggregate_probe_reference stage requested_col))
					((symbol get_column) tblvar tbl_ignorecase col col_ignorecase) (begin
						/* Most helper expressions address one base source. Avoid rebuilding
						a role lookup for that overwhelmingly common planner hot path. */
						(define role (if (and (not (nil? sole_source))
							(source_alias_matches? sole_source default_alias tblvar tbl_ignorecase))
							(list 0 sole_source)
							(canonical_helper_source_role sources default_alias tblvar tbl_ignorecase)))
						(if (nil? role)
							(list (quote outer-column) tblvar col)
							(begin
								(define src (nth role 1))
								(list (quote base-column) (nth role 0)
									(source_schema src) (source_relation src)
									(canonical_col col col_ignorecase)))))
					((quote get_column) tblvar tbl_ignorecase col col_ignorecase)
					(rewrite (list (symbol "get_column") tblvar tbl_ignorecase col col_ignorecase))
					(cons head tail) (cons (rewrite head) (map tail rewrite))
					_ (if (symbol? node)
						(match (string node)
							(concat alias "." col) (begin
								(define role (canonical_helper_source_role sources default_alias alias false))
								(if (nil? role)
									node
									(begin
										(define src (nth role 1))
										(list (quote base-column) (nth role 0)
											(source_schema src) (source_relation src) col))))
							_ node)
						node)))))
		(rewrite expr))))

(define canonical_group_stage_alias_map (lambda (stage)
	(begin
		(define input (gs_input stage))
		(define local_aliases (source_aliases (stage_semantic_input_sources input)))
		/* Carrier identity deliberately excludes aggregates, output, HAVING, and
		ORDER. Adding another aggregate column to the same keys and condition must
		reuse the existing keytable rather than create a second carrier. Neumann's
		domain is the complete ordered interface of outer references, so it also
		avoids rescanning the full keys and condition for aliases here. */
		(define outer_aliases (filter
			(merge_unique (map (gs_domain stage) stage_semantic_expr_aliases))
			(lambda (alias)
				(not (contains? local_aliases alias)))))
		(merge (list
			(stage_semantic_alias_entries local_aliases "__carrier_local_")
			(stage_semantic_alias_entries outer_aliases "__carrier_outer_"))))))

(define canonical_group_input_identity (lambda (alias_map signatures input)
	/* A carrier stores the input row domain, not its SELECT projection. Omitting
	fields, hidden expressions, and the stage catalog prevents nested scalar
	projection trees from being serialized again for every enclosing aggregate. */
	(if (query_block? input)
		(list
			(quote query-domain)
			(qb_schema input)
			(map (qb_sources input) (lambda (src)
				(stage_semantic_rewrite_source alias_map signatures src)))
			(stage_semantic_rewrite_expr alias_map signatures (qb_where input))
			(stage_semantic_rewrite_expr alias_map signatures (qb_group input))
			(stage_semantic_rewrite_expr alias_map signatures (qb_having input))
			/* ORDER only changes the represented row domain when a bound consumes it. */
			(if (and (nil? (qb_limit input)) (nil? (qb_offset input)))
				'()
				(stage_semantic_rewrite_expr alias_map signatures (qb_order input)))
			(qb_limit input)
			(qb_offset input))
		/* UNION projections define the rows exposed to the grouping input. Keep
		the complete canonical union until it gets a dedicated domain form. */
		(stage_semantic_canonical_node alias_map signatures input))))

(define canonical_orc_column_name (lambda (kind src payload)
	/* ORCs live on one base table. Their identity is the table plus the complete
	physical window recipe; SELECT aliases and read-time LIMIT/OFFSET are absent. */
	(concat "__orc_" kind "_" (fnv_hash (serialize (list
		"canonical-orc-v2"
		(list (source_schema src) (source_relation src))
		payload))))))

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

(define group_stage_default_cache (lambda (stage signatures)
	(begin
		(define schema (group_stage_schema stage))
		(define input (gs_input stage))
		(define label (if (source_is_base_table? input) (source_relation input)
			(if (union_block? input) "union" "query")))
		(define keys (if (empty_list? (gs_keys stage)) '(1) (gs_keys stage)))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define alias_map (canonical_group_stage_alias_map stage))
		(make_group_keytable_cache schema (group_table_name
			schema
			label
			(canonical_group_input_identity alias_map signatures input)
			(stage_semantic_rewrite_expr alias_map signatures keys)
			(stage_semantic_rewrite_expr alias_map signatures condition))))))

(define group_stage_cache (lambda (stage)
	(begin
		(define cached (qassoc_get (gs_facts stage) (quote group_cache) nil))
		/* Analysis and cost probes have no semantic index yet and deliberately use
		the cheap provisional identity. Physical preparation passes its shared index. */
		(if (nil? cached) (group_stage_default_cache stage '()) cached))))

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
							(combine_where_terms (map (produceN (count pairs)) (lambda (i)
								(list (quote equal??) (nth params i) (nth (nth pairs i) 0))))
								true)))))))))

(define runtime_cons_list_expr (lambda (exprs)
	(if (empty_list? exprs)
		(quoted_runtime_list '())
		(list (quote cons) (car exprs) (runtime_cons_list_expr (cdr exprs))))))

(define make_group_stage_for_block (lambda (block src)
	(begin
		(define visible_ags (stage_aggregates_for_fields (qb_fields block)))
		(define having_ags (extract_aggregates (coalesceNil (qb_having block) true)))
		(define ags (dedupe_aggregates_by_col (if (empty_list? (qb_group block))
			(merge (list visible_ags having_ags))
			(merge (list visible_ags having_ags (list aggregate_count_descriptor))))))
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
			(merge (list visible_ags having_ags))
			(merge (list visible_ags having_ags (list aggregate_count_descriptor))))))
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

(define group_aggregate_read_expr (lambda (input grouptbl ag)
	(begin
		(define col (aggregate_col_name_using input ag))
		(define read_expr (list (quote get_column) grouptbl false col false))
		(if (aggregate_count_like? ag)
			(list (quote coalesceNil) read_expr 0)
			read_expr))))

(define group_aggregate_order_read_expr (lambda (input grouptbl ag)
	(list (quote get_column) grouptbl false (aggregate_col_name_using input ag) false)))

(define count_distinct_read_expr (lambda (input grouptbl agg_expr)
	(begin
		(define read_expr (list (quote get_column) grouptbl false (aggregate_col_name_using input (count_distinct_descriptor agg_expr)) false))
		(list (quote if)
			(list (quote list?) read_expr)
			(list (quote count) read_expr)
			(list (quote coalesceNil) read_expr 0)))))

(define replace_group_probe_stage_lookup_keys (lambda (input alias grouptbl keys key_names ags key_index stage)
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
							(replace_group_expr_indexed input alias grouptbl keys key_names ags
								(make_group_key_index keys (list resolved)) expr resolved))))))))))

(define replace_group_probe_stages_lookup_keys (lambda (input alias grouptbl keys key_names ags key_index stages)
	(map (coalesceNil stages '()) (lambda (stage)
		(replace_group_probe_stage_lookup_keys input alias grouptbl keys key_names ags key_index stage)))))

(define replace_group_expr_tail_indexed (lambda (input alias grouptbl keys key_names ags key_index items resolved_items)
	(match items
		(cons item rest)
		(cons
			(replace_group_expr_indexed input alias grouptbl keys key_names ags key_index item (car resolved_items))
			(replace_group_expr_tail_indexed input alias grouptbl keys key_names ags key_index rest (cdr resolved_items)))
		_ '())))

(define replace_group_expr_indexed (lambda (input alias grouptbl keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol scalar_first_probe) stage requested_col stages)
				(list (quote scalar_first_probe)
					(replace_group_probe_stage_lookup_keys input alias grouptbl keys key_names ags key_index stage)
					requested_col
					(replace_group_probe_stages_lookup_keys input alias grouptbl keys key_names ags key_index stages))
				((quote scalar_first_probe) stage requested_col stages)
				(list (quote scalar_first_probe)
					(replace_group_probe_stage_lookup_keys input alias grouptbl keys key_names ags key_index stage)
					requested_col
					(replace_group_probe_stages_lookup_keys input alias grouptbl keys key_names ags key_index stages))
				((symbol scalar_aggregate_probe) stage requested_col)
				(list (quote scalar_aggregate_probe)
					(replace_group_probe_stage_lookup_keys input alias grouptbl keys key_names ags key_index stage)
					requested_col)
				((quote scalar_aggregate_probe) stage requested_col)
				(list (quote scalar_aggregate_probe)
					(replace_group_probe_stage_lookup_keys input alias grouptbl keys key_names ags key_index stage)
					requested_col)
				((symbol count_distinct) agg_expr)
				(count_distinct_read_expr input grouptbl agg_expr)
				((quote count_distinct) agg_expr)
				(count_distinct_read_expr input grouptbl agg_expr)
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_read_expr input grouptbl (list agg_expr agg_reduce agg_neutral))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_read_expr input grouptbl (list agg_expr agg_reduce agg_neutral))
				((symbol get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				((quote get_column) _tblvar _ _col _)
				(if (equal?? (resolve_column_alias _tblvar alias) alias)
					(neumann_fail "build_queryplan" (concat "non-aggregate output must be a GROUP BY key: " (serialize expr)))
					expr)
				(cons head tail) (cons head
					(replace_group_expr_tail_indexed input alias grouptbl keys key_names ags key_index tail (cdr resolved)))
				_ expr)))))

(define replace_group_expr (lambda (input alias grouptbl keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(replace_group_expr_indexed
			input alias grouptbl keys key_names ags (make_group_key_index keys (list resolved))
			expr resolved))))

(define replace_group_fields_indexed (lambda (input alias grouptbl keys key_names ags key_index fields resolved_fields)
	(match fields
		(cons title (cons expr rest))
		(cons title (cons
			(replace_group_expr_indexed input alias grouptbl keys key_names ags key_index expr (cadr resolved_fields))
			(replace_group_fields_indexed input alias grouptbl keys key_names ags key_index rest (cdr (cdr resolved_fields)))))
		_ '())))

(define replace_group_order_expr_tail_indexed (lambda (input alias grouptbl keys key_names ags key_index items resolved_items)
	(match items
		(cons item rest)
		(cons
			(replace_group_order_expr_indexed input alias grouptbl keys key_names ags key_index item (car resolved_items))
			(replace_group_order_expr_tail_indexed input alias grouptbl keys key_names ags key_index rest (cdr resolved_items)))
		_ '())))

(define replace_group_order_expr_indexed (lambda (input alias grouptbl keys key_names ags key_index expr resolved)
	(begin
		(define key_idx (lookup_group_key_index key_index resolved))
		(if (not (nil? key_idx))
			(list (quote get_column) grouptbl false (nth key_names key_idx) false)
			(match expr
				((symbol count_distinct) agg_expr)
				(count_distinct_read_expr input grouptbl agg_expr)
				((quote count_distinct) agg_expr)
				(count_distinct_read_expr input grouptbl agg_expr)
				((symbol aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_order_read_expr input grouptbl (list agg_expr agg_reduce agg_neutral))
				((quote aggregate) agg_expr agg_reduce agg_neutral)
				(group_aggregate_order_read_expr input grouptbl (list agg_expr agg_reduce agg_neutral))
				(cons head tail) (cons head
					(replace_group_order_expr_tail_indexed input alias grouptbl keys key_names ags key_index tail (cdr resolved)))
				_ expr)))))

(define replace_group_order_expr (lambda (input alias grouptbl keys key_names ags expr)
	(begin
		(define resolved (canonical_column_expr_for_alias alias expr))
		(replace_group_order_expr_indexed
			input alias grouptbl keys key_names ags (make_group_key_index keys (list resolved))
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
				(group_insert_batches_expr schema grouptbl key_names '() false (quote grouped)))
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
		(define agg_col (aggregate_col_name_using src ag))
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
						(cons (quote and) (cons
							(lower_column_expr_for_alias src condition)
							(group_key_equality_terms alias key_names keys))))
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
		(define agg_cols (map ags (lambda (ag) (aggregate_col_name_using src ag))))
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
				(group_insert_finish_expr schema grouptbl key_names agg_cols false))
			(list (quote scan_order)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda)
					(map filtercols (lambda (col) (symbol (concat alias "." col))))
					(lower_column_expr_for_alias src condition))
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
			(define agg_col (aggregate_col_name_using src ag))
			(define membership_parts (base_group_membership_parts src condition))
			(define membership_expr (car membership_parts))
			(define effective_condition (cadr membership_parts))
			(if (expr_contains_driver_membership? condition)
				(planner_record_physical_decision
					(list
						(list "decision" "base_group_membership_consumer")
						(list "chosen" (if (nil? membership_expr)
							"per_row_probe" "projected_recset"))
						(list "inputs" (list
							(list "marker_present" true)
							(list "marker_resolved_to_driver"
								(not (nil? (driver_membership_for_source src condition))))
							(list "projection_built" (not (nil? membership_expr)))))))
				nil)
			(define group_key_cols_for_scan (merge_unique (map keys (lambda (expr) (extract_columns_for_alias src expr)))))
			/* The ordinary per-group scan still lowers condition in full. Its
			membership marker is a semantic leaf whose driver probe can require
			columns removed from effective_condition, so collect dependencies from
			the same expression the callback evaluates. */
			(define condition_cols (extract_columns_for_alias src condition))
			(define filtercols (merge_unique (list group_key_cols_for_scan condition_cols)))
			(define aggcols (extract_columns_for_alias src agg_expr))
			(define direct_recset_count (and (not (nil? membership_expr))
				(and (equal? effective_condition true)
					(and (not (reduce keys (lambda (has_columns key)
						(or has_columns (expr_contains_column_ref? key))) false))
						(aggregate_counts_every_input_row? ag)))))
			(define aggregate_value_expr (if direct_recset_count
				(list (quote recset_count)
					(list
						(physical_query_session_symbol)
						"get_or_compute_scoped"
						(physical_query_scope_symbol)
						(concat "__group_count_recset_" (fnv_hash (serialize membership_expr)))
						(list (quote lambda) '() membership_expr)))
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda)
						(map filtercols (lambda (col) (symbol (concat alias "." col))))
						(cons (quote and) (cons
							(lower_column_expr_for_alias src condition)
							(group_key_equality_terms alias key_names keys))))
					(cons (quote list) aggcols)
					(list (quote lambda)
						(map aggcols (lambda (col) (symbol (concat alias "." col))))
						(aggregate_map_value_expr ag (lower_column_expr_for_alias src agg_expr)))
					agg_reduce
					agg_neutral
					(aggregate_shard_combine ag)
					false)))
			(list (quote createcolumn)
				(list (quote table) schema grouptbl)
				agg_col
				"any"
				(list (quote list))
				(quoted_runtime_list '("temp" true))
				(cons (quote list) key_names)
				(list (quote lambda)
					(map key_names (lambda (col) (symbol col)))
					aggregate_value_expr))))))

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
				(lower_column_expr_for_alias src condition))
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
						(group_insert_finish_expr schema grouptbl key_names
							(map ags (lambda (ag) (aggregate_col_name_using src ag))) true))
					grouped_expr))
			grouped_scan))
		(if (nil? membership_expr)
			plan
			(list
				(list (quote lambda) (list membership_var) plan)
				membership_expr)))))

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

(define group_upsert_collision_cols (lambda (value_cols computed_values)
	(if (empty_list? value_cols)
		'(list "$update")
		(cons (quote list) (merge (list
			(if computed_values
				(map value_cols (lambda (col) (concat "$set:" col)))
				(list "$update"))
			(map value_cols (lambda (col) (concat "NEW." col)))))))))

(define group_upsert_collision_lambda (lambda (value_cols computed_values)
	(if (empty_list? value_cols)
		(list (quote lambda)
			(list (quote $update))
			(list (quote $update) (cons (quote list) '())))
		(begin
			(define new_params (map (produceN (count value_cols)) (lambda (i) (symbol (concat "__new_group_value_" i)))))
			(if computed_values
				(begin
					(define setter_params (map (produceN (count value_cols)) (lambda (i) (symbol (concat "__set_group_value_" i)))))
					(list (quote lambda)
						(merge (list setter_params new_params))
						(cons (quote begin) (map (produceN (count value_cols)) (lambda (i)
							(list (nth setter_params i) (nth new_params i)))))))
				/* Ordinary aggregate columns are one row payload. A single $update
				keeps every value on the same replacement row and avoids reusing the
				stale recid after the first column update. */
				(list (quote lambda)
					(cons (quote $update) new_params)
					(list (quote $update) (runtime_cons_list_expr
						(merge (map (produceN (count value_cols)) (lambda (i)
							(list (nth value_cols i) (nth new_params i)))))))))))))

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

(define group_insert_batches_expr (lambda (schema grouptbl key_names value_cols computed_values grouped_expr)
	(list (quote group_insert_batches)
		(list (quote table) schema grouptbl)
		(cons (quote list) (merge (list key_names value_cols)))
		(group_upsert_collision_cols value_cols computed_values)
		(group_upsert_collision_lambda value_cols computed_values)
		grouped_expr)))

(define group_insert_finish_expr (lambda (schema grouptbl key_names value_cols computed_values)
	(list (quote !begin)
		(group_cleanup_missing_keys_plan schema grouptbl key_names)
		(group_insert_batches_expr schema grouptbl key_names value_cols computed_values (quote grouped)))))

/* Eager preparation may rewrite nested stage-output sources and aggregate
expressions. Persistent column identity was selected from the original logical
stage before that rewrite; thread it through unchanged to keep create and fill
on the same columns. */
(define build_query_group_aggregates_insert_plan (lambda (input grouptbl keys key_names ags aggregate_cols)
	(begin
		(if (not (equal? (count ags) (count aggregate_cols)))
			(neumann_fail "build_queryplan" "query-input aggregate columns do not match aggregate descriptors")
			true)
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
		(define finish_expr (group_insert_finish_expr schema grouptbl key_names aggregate_cols false))
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

(define build_union_group_aggregates_insert_plan (lambda (input naming_input grouptbl keys key_names ags)
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
		(define finish_expr (group_insert_finish_expr schema grouptbl key_names
			(map ags (lambda (ag) (aggregate_col_name_using naming_input ag))) false))
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

(define build_scalar_single_query_stage_fill_plan (lambda (input grouptbl keys key_names value_ag count_ag value_col count_col)
	(match value_ag '(value_expr _value_reduce _value_neutral) (begin
		(define prepared_input (if (and (query_block? input) (not (empty_list? (qb_stages input))))
			(query_block_without_stages_after_eager_prepare_using (qb_stages input) input)
			input))
		(define schema (qb_schema prepared_input))
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
				(group_insert_finish_expr schema grouptbl key_names (list value_col count_col) false))
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

(define rewrite_source_for_group_domain (lambda (input alias grouptbl keys key_names ags src)
	(begin
		(define resolved (canonical_column_expr_for_alias alias (source_join_expr src)))
		(rewrite_source_for_group_domain_indexed input alias grouptbl keys key_names ags
			(make_group_key_index keys (list resolved)) src resolved))))

(define rewrite_source_for_group_domain_indexed (lambda (input alias grouptbl keys key_names ags key_index src resolved_join)
	(list
		(source_alias src)
		(source_schema src)
		(source_relation src)
		(source_outer? src)
		(replace_group_expr_indexed input alias grouptbl keys key_names ags key_index
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
				/* Eager preparation may already have replaced stage-output with the
				canonical group-cache relation. Both names resolve to the same stage
				handle and must therefore expose the same probe rewrite. */
				(define original (coalesceNil
					(source_stage_output_stage stages src)
					(stage_for_group_cache_source stages src)))
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
					/* A marker only needs its proper dependency closure. Embedding the
					whole query catalog in every independent marker makes emitted IR
					quadratic in the number of scalar subqueries. */
					(define probe_catalog
						(cdr (get_assoc closures (logical_stage_key stage))))
					(define marker_stage (if direct
						stage
						(group_stage_with_facts stage
							(qassoc_set
								(qassoc_set (gs_facts stage) (quote probe_catalog) probe_catalog)
								(quote promoted_probe) true))))
					(define index_entry (list marker_stage (if direct
						(list marker_stage)
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
					(define invariant_binding (query_invariant_probe_binding_for_col stage col))
					(if (not (nil? invariant_binding))
						invariant_binding
						(if (scalar_aggregate_probe_stage? stage)
							(coalesceNil
								(scalar_first_stage_key_lookup_expr stage col)
								(scalar_aggregate_probe_expr stage col))
							(coalesceNil
								(scalar_first_stage_key_lookup_expr stage col)
								(scalar_first_probe_expr stage col dependencies)))))))
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

(define probe_limit_work_rows (lambda (limit_value)
	(begin
		(define planning_limit (planner_literal_value limit_value))
		(if (and (number? planning_limit) (> planning_limit 0))
			planning_limit
			nil))))

(define probe_limit_bounded? (lambda (limit_value)
	(not (nil? (probe_limit_work_rows limit_value)))))

/* BEGIN GENERATED COST CONSTANTS. DO NEVER MANUALLY EDIT THIS SECTION. RUN make costgen TO UPDATE.
Calibrated by tools/costgen from tests/**/*.yaml workloads tagged with
metadata.physical_calibration. Each observation is an executed, forced
EXPLAIN PHYSICAL CALIBRATE alternative with result and operator validation. */
(define planner_direct_presence_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows 48685) 0 0 0 0 0 probe_rows 0.75)))

(define planner_presence_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost 1421611 (* probe_rows 136938) 0 0 0 0
		(* domain_rows 8) 0 domain_rows 0.65)))

(define planner_recset_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost 365607 0 0 0 0 (* probe_rows 17681)
		(* probe_rows 1) 0 probe_rows 0.6)))

(define planner_membership_scan_invocation_ns 1)
(define planner_membership_scan_row_ns 178)
(define planner_membership_filter_column_row_ns 1)
(define planner_membership_map_column_row_ns 28)
(define planner_membership_expression_operation_row_ns 98)
(define planner_membership_broad_text_match_row_ns 1)
(define planner_membership_broad_text_match_byte_ns 4)
(define planner_membership_recset_startup_ns 1)
(define planner_membership_recset_build_row_ns 1)
(define planner_membership_recset_probe_row_ns 1)
(define planner_membership_recset_aggregate_row_ns 215)
(define planner_membership_group_cache_startup_ns 17538872)
(define planner_membership_group_cache_build_row_ns 11590)
(define planner_membership_group_cache_probe_row_ns 132886)
(define planner_membership_ordered_driver_input_row_ns 255)
/* END GENERATED COST CONSTANTS */

(define planner_direct_presence_probe_preferred? (lambda (probe_rows input_rows)
	(and (number? probe_rows)
		(and (> probe_rows 0)
			(and (number? input_rows)
				(planner_cost_better?
					(planner_direct_presence_probe_cost probe_rows)
					(planner_presence_carrier_cost input_rows probe_rows)))))))

(define stage_direct_probe_cost_preferred? (lambda (stage probe_rows_value)
	(begin
		(define probe_rows (planner_literal_value probe_rows_value))
		(define input_rows (planner_stage_input_rows (gs_input stage)))
		(define chosen (planner_direct_presence_probe_preferred? probe_rows input_rows))
		(if (and (number? probe_rows) (number? input_rows))
			(planner_guarded_choice chosen
				(list (quote planner_direct_presence_probe_preferred?)
					probe_rows_value
					(list (quote planner_stage_input_rows)
						(list (quote quote) (gs_input stage)))))
			chosen))))

(define stage_direct_probe_cost_preferred_for_limit? (lambda (stage limit_value)
	(begin
		(define work_rows (probe_limit_work_rows limit_value))
		(and (not (nil? work_rows))
			(stage_direct_probe_cost_preferred? stage limit_value)))))

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
		(reduce (source_unique_key_sets src) (lambda (found key_cols)
			(or found
				(reduce key_cols (lambda (complete col)
					(and complete (source_column_bound_by_equality? src col condition)))
					true)))
			false))))

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

(define stage_probe_dependencies_resolve_in_catalog? (lambda (stages stage)
	(begin
		(define input (gs_input stage))
		(if (not (query_block? input))
			true
			(reduce (qb_sources input) (lambda (resolved dependency_source)
				(and resolved
					(or (not (stage_output_relation? (source_relation dependency_source)))
						(not (nil? (source_stage_output_stage stages dependency_source))))))
				true)))))

(define query_block_structurally_refs_stage_id? (lambda (block target_id)
	(or
		(reduce (qb_sources block) (lambda (found src)
			(or found
				(and (stage_output_relation? (source_relation src))
					(equal? (stage_output_relation_id (source_relation src)) target_id))))
			false)
		(reduce (qb_stages block) (lambda (found nested_stage)
			(or found
				(or (equal? (gs_id nested_stage) target_id)
					(and (query_block? (gs_input nested_stage))
						(query_block_structurally_refs_stage_id? (gs_input nested_stage) target_id)))))
			false))))

(define stage_consumed_by_presence_stage? (lambda (stages target_stage)
	(reduce (lowering_catalog_stages stages) (lambda (found candidate)
		(or found
			(and (presence_probe_stage? candidate)
				(and (query_block? (gs_input candidate))
					(query_block_structurally_refs_stage_id?
						(gs_input candidate) (gs_id target_stage))))))
		false)))

/* Bare presence terms still feed the older driver-membership RecSet route,
whose nested-join correctness is deliberately narrower than scalar-first
carrier lowering. Preserve its established bounded-context gate until that
separate route can represent arbitrary dependency shapes safely. */
(define presence_stage_probe_allowed_in_context? (lambda (stage sources)
	(if (or (stage_has_residual_outer_refs? stage)
		(not (stage_keys_are_input_local? stage)))
		true
		(begin
			(define input_rows (planner_stage_input_rows (gs_input stage)))
			(or
				(and (number? input_rows) (< input_rows 1000))
				(probe_context_small_enough? sources))))))

/* Scalar-first stages enter the generic cost-based carrier chain whenever
their lookup is valid. Presence stages retain their established bounded/small
context gates because bare EXISTS also has a separate membership lowerer. */
(define probeable_stage_output_source_for_block? (lambda (stages sources default_alias limit_value driver_condition src)
	(if (scalar_first_stage_output_source? stages src)
		(begin
			(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
			(define probe_sources (filter sources (lambda (candidate)
				(not (equal? (source_alias candidate) (source_alias src))))))
			(and (not (stage_consumed_by_presence_stage? stages stage))
				(and (stage_probe_dependencies_resolve_in_catalog? stages stage)
					(stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias))))
		(if (not (presence_stage_output_source? stages src))
			false
			(begin
				(define stage (stage_by_id stages (stage_output_relation_id (source_relation src))))
				(define probe_sources (filter sources (lambda (candidate)
					(not (equal? (source_alias candidate) (source_alias src))))))
				(define cardinality_sources (filter probe_sources (lambda (candidate)
					(not (presence_stage_output_source? stages candidate)))))
				(and (stage_probe_dependencies_resolve_in_catalog? stages stage)
					(and (stage_lookup_keys_resolve_in_sources? stage probe_sources default_alias)
						(or (stage_direct_probe_cost_preferred_for_limit? stage limit_value)
							(or (presence_stage_probe_allowed_in_context? stage probe_sources)
								(probe_context_unique_point?
									cardinality_sources default_alias driver_condition))))))))))

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
								(stage_direct_probe_cost_preferred_for_limit? stage limit_value)
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
				(or (stage_direct_probe_cost_preferred_for_limit? stage limit_value)
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
					(or (stage_direct_probe_cost_preferred_for_limit? stage limit_value)
						(or (probe_context_small_enough? probe_sources)
							(probe_context_unique_point? probe_sources default_alias driver_condition)))))))))

(define probe_output_sources_for_block (lambda (stages sources default_alias limit_value driver_condition consumers)
	(filter (coalesceNil sources '()) (lambda (src)
		(or
			(probeable_stage_output_source_for_block?
				stages sources default_alias limit_value driver_condition src)
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
					(presence_stage_probe_allowed_in_context? stage probe_sources)
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
			/* Do not derive a carrier while merely cataloging logical alternatives.
			The cost model may lower only a small subset to group keytables; their
			canonical physical names are computed lazily by group_stage_cache. */
			(group_stage_with_facts stage
				(if (lowering_catalog? catalog)
					(qassoc_set_without (gs_facts stage) (quote lowering_catalog) catalog (quote stage_catalog))
					(qassoc_set (gs_facts stage) (quote stage_catalog) catalog)))))))

(define stages_with_canonical_group_caches_acc (lambda (stages signatures cache_index)
	(match (coalesceNil stages '())
		(cons stage rest) (begin
			(if (not (group_stage? stage))
				(begin
					(define tail (stages_with_canonical_group_caches_acc rest signatures cache_index))
					(list (cons stage (nth tail 0)) (nth tail 1)))
				(begin
					(define signature (get_assoc signatures (gs_id stage)))
					(define cache (if (has_assoc? cache_index signature)
						(cache_index signature)
						(group_stage_default_cache stage signatures)))
					(define next_index (if (has_assoc? cache_index signature)
						cache_index
						(set_assoc cache_index signature cache)))
					(define cached_stage (group_stage_with_facts stage
						(qassoc_set (gs_facts stage) (quote group_cache) cache)))
					(define tail (stages_with_canonical_group_caches_acc rest signatures next_index))
					(list (cons cached_stage (nth tail 0)) (nth tail 1)))))
		_ (list '() cache_index))))

(define stages_with_canonical_group_caches (lambda (stages signatures)
	(nth (stages_with_canonical_group_caches_acc stages signatures '()) 0)))

(define stage_shared_prepare? (lambda (stage)
	(and (group_stage? stage)
		(qassoc_get (gs_facts stage) (quote shared_prepare) false))))

(define stages_with_shared_prepare_facts (lambda (stages)
	(begin
		(define dependency_graph (stage_dependency_graph stages))
		(define counts (reduce stages (lambda (found stage)
			(if (closed_group_prepare_stage? dependency_graph stage)
				(begin
					(define key (stage_prepare_backbone_signature stage))
					(set_assoc found key (+ 1 (coalesceNil (get_assoc found key) 0))))
				found)) '()))
		(map stages (lambda (stage)
			(if (and (closed_group_prepare_stage? dependency_graph stage)
				(> (coalesceNil (get_assoc counts (stage_prepare_backbone_signature stage)) 0) 1))
				(group_stage_with_facts stage
					(qassoc_set (gs_facts stage) (quote shared_prepare) true))
				stage))))))

(define group_stage_lowering_catalog (lambda (stage)
	(match (gs_facts stage)
		(cons entry _rest) (match entry
			((symbol lowering_catalog) catalog) catalog
			((quote lowering_catalog) catalog) catalog
			_ nil)
		_ nil)))

(define query_block_with_full_stage_catalog_using (lambda (block stages)
	(begin
		(define signatures (stage_semantic_signature_index stages))
		/* Catalog lookups must return the same annotated immutable stage instances
		that root lowering sees; otherwise nested probe copies derive old names. */
		(define signature_stages (stages_with_shared_prepare_facts
			(stages_with_canonical_group_caches stages signatures)))
		(define catalog (make_lowering_catalog signature_stages))
		(define cataloged_stages (map (qb_stages block) (lambda (stage)
			(begin
				(define cached (if (group_stage? stage) (stage_by_id catalog (gs_id stage)) nil))
				(group_stage_with_lowering_catalog
					(if (nil? cached) stage cached)
					catalog)))))
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
			(query_block_facts_with_lowering_catalog block signature_stages catalog)))))

(define query_block_with_full_stage_catalog (lambda (block)
	(query_block_with_full_stage_catalog_using block
		(stage_catalog_with_nested (query_block_stage_catalog block)))))

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

(define stages_without_probe_sources_using_graph (lambda (dependency_graph stage_lookup stages probe_sources)
	(begin
		(define consumed (stages_without_consumed_probes_using_graph dependency_graph stages
			(stage_outputs_from_sources_using stage_lookup probe_sources)))
		(define direct_ids (stage_output_source_ids (filter probe_sources (lambda (src)
			(not (nil? (direct_group_probe_stage_for_source stage_lookup src (list true))))))))
		(stages_without_ids consumed direct_ids))))

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
		(define dml_block (qassoc_get (qb_facts block) (quote dml) false))
		(define eligible_probe_sources (lambda (limit_value)
			(filter (probe_output_sources_for_block
				stages sources default_alias limit_value (qb_where block) consumers)
				(lambda (src)
					(or (not dml_block)
						(not (scalar_first_stage_output_source? stages src)))))))
		(define unbounded_probe_candidates (eligible_probe_sources nil))
		(define unbounded_probe_aliases (map unbounded_probe_candidates source_alias))
		(define probe_candidates (filter
			(eligible_probe_sources (qb_limit block))
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
		(define probe_index (probe_stage_alias_index_using_graph
			stages dependency_graph probe_sources consumers))
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
			/* probe_sources is the caller's already costed and semantics-checked
			selection. Do not rerun the unbounded cache heuristic while removing it. */
			(sources_without_probe_outputs rewritten_sources probe_sources)
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

(define group_stage_with_eager_prepared_metadata (lambda (stage)
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
			(qassoc_set (gs_facts stage) (quote eager_prepared) true)))))

(define query_block_without_stages_after_eager_prepare_using (lambda (stages block)
	(begin
		(define available_stages (if (lowering_catalog? stages)
			(lowering_catalog_stages stages)
			(unique_stages_by_id (merge (list stages (qb_stages block))))))
		(define stage_lookup (if (lowering_catalog? stages) stages available_stages))
		(define source_stages (unique_stages_by_id (merge (list
			(available_stage_outputs_from_sources_using stage_lookup (qb_sources block))
			(group_cache_stages_from_sources stage_lookup (qb_sources block))))))
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
			(query_block_facts_with_stage_catalog block
				(map source_stages group_stage_with_eager_prepared_metadata))))))

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
			src alias grouptbl keys key_names ags key_index original_output resolved_output))
		(define replaced_having (replace_group_expr_indexed src alias grouptbl keys key_names ags key_index
			original_having resolved_having))
		(define count_col_name (aggregate_col_name_using src aggregate_count_descriptor))
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
					(rewrite_source_for_group_domain_indexed src alias grouptbl keys key_names ags key_index
						(nth extras i) (nth resolved_extra_joins i)))))
			output_fields
			having_expr
			nil nil
			(map (produceN (count order_items)) (lambda (i)
				(match (nth order_items i) '(expr dir) (begin
					(define replaced_order_expr (replace_group_order_expr_indexed
						src alias grouptbl keys key_names ags key_index expr
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

/* A canonical carrier describes keys and input semantics, while aggregate
columns are independent extensions of that carrier. Collect all compatible
extensions before emitting DDL/fill code so one query has one creator and one
key-fill recipe per carrier. */
(define stage_prepare_backbone_signature (lambda (stage)
	(if (not (group_stage? stage))
		(logical_stage_key stage)
		(begin
			(define cache (group_stage_cache stage))
			/* Canonical carrier naming already includes the complete key and
			condition semantics. It is therefore the physical identity used to
			collect all aggregate columns and emit exactly one initializer. */
			(concat "stage-prepare:"
				(fnv_hash (serialize (list
					(group_cache_schema cache)
					(group_cache_relation cache)))))))))

(define group_prepare_stage_with_aggregates (lambda (stage aggregates)
	(make_group_stage
		(gs_id stage)
		(gs_input stage)
		(gs_domain stage)
		(gs_keys stage)
		(dedupe_aggregates_by_col aggregates)
		(gs_having stage)
		(gs_output stage)
		(gs_order stage)
		(gs_limit stage)
		(gs_offset stage)
		(gs_facts stage))))

(define merge_group_prepare_stage (lambda (target stage)
	(group_prepare_stage_with_aggregates target
		(merge (list
			(gs_aggregates target)
			(stage_output_left_join_aligned_aggregates target stage))))))

(define collect_stage_prepares (lambda (stages)
	(begin
		(define collected (reduce (coalesceNil stages '()) (lambda (state stage)
			(begin
				(define index (nth state 0))
				(define keys (nth state 1))
				(define key (if (group_stage? stage)
					(stage_prepare_backbone_signature stage)
					(concat "logical-prepare:" (logical_stage_key stage))))
				(define previous (if (has_assoc? index key) (index key) nil))
				(list
					(set_assoc index key (if (nil? previous)
						stage
						(if (and (group_stage? previous) (group_stage? stage))
							(merge_group_prepare_stage previous stage)
							previous)))
					(if (nil? previous) (cons key keys) keys))))
			(list '() '())))
		(define index (nth collected 0))
		(map (reverse (nth collected 1)) (lambda (key) (index key))))))

(define stage_prepare_backbone_set (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (keys stage)
		(set_assoc keys (stage_prepare_backbone_signature stage) true)) '())))

(define stages_without_prepare_backbones (lambda (stages prepared)
	(begin
		(define prepared_backbones (stage_prepare_backbone_set prepared))
		(filter (coalesceNil stages '()) (lambda (stage)
			(not (has_assoc? prepared_backbones (stage_prepare_backbone_signature stage))))))))

(define stage_prepare_identity (lambda (stage)
	(if (group_stage? stage)
		(begin
			(define cache (group_stage_cache stage))
			(list
				(quote group)
				(group_cache_schema cache)
				(group_cache_relation cache)
				(map (gs_aggregates stage) (lambda (ag)
					(aggregate_col_name_using (gs_input stage) ag)))
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
			/* This is a selected physical prepare, not a catalog alternative. Keep
			the canonical carrier on its immutable stage copy so all lowering readers
			share one derivation without introducing global planner state. */
			(define cache (if (group_stage? stage) (group_stage_cache stage) nil))
			(define group_cache_key (if shared_group_cache
				(concat (group_cache_schema cache) "\n" (group_cache_relation cache))
				nil))
			(define initializer_owner (or (not shared_group_cache) (not (has_assoc? initialized_group_caches group_cache_key))))
			(define prepared_stage (if (group_stage? stage)
				(group_stage_with_initializer_owner stage initializer_owner cache)
				stage))
			(define identity (stage_prepare_identity prepared_stage))
			(define next_group_caches (if (and shared_group_cache initializer_owner)
				(set_assoc initialized_group_caches group_cache_key true)
				initialized_group_caches))
			(if (has_assoc? seen identity)
				(lower_unique_stage_prepares_acc rest seen next_group_caches prepare)
				(cons (prepare prepared_stage)
					(lower_unique_stage_prepares_acc rest (set_assoc seen identity true) next_group_caches prepare))))
		_ '())))

(define lower_unique_stage_prepares (lambda (stages prepare)
	(lower_unique_stage_prepares_acc (collect_stage_prepares stages) '() '() prepare)))

(define lower_unique_stage_prepares_using (lambda (all_stages lookup_stages stages)
	(lower_unique_stage_prepares stages (lambda (stage)
		(lower_stage_prepare_using all_stages lookup_stages stage)))))

(define lower_unique_stage_prepares_with_graph (lambda (dependency_graph stage_lookup stages)
	(lower_unique_stage_prepares stages (lambda (stage)
		(lower_stage_prepare_using
			(stage_dependency_closure_using_graph dependency_graph stage)
			stage_lookup
			stage)))))

/* Restores the group-cache partitioning hint that existed in the pre-Neumann
planner (see df66a5831, "Queryplan: add table repartitioning hint in GROUP
BY") and was dropped during the logical-operator rewrite. Real pivots are
sampled from the live source column via the existing shardcolumn/
partitiontable primitives -- no new core storage logic, just wiring them back
up. shardcolumn's automatic partition count already scales off real row count
(t.Count()/ShardSize), so small source tables naturally collapse to a single
partition and partitiontable skips them; only direct column references to the
scanned base table qualify, so session/constant-key groups (never plain
get_column refs to the source) are excluded automatically too. */
(define group_cache_partition_hint (lambda (schema tbl alias key col_name)
	(match key
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase) (if (equal? tblvar alias)
			(list col_name (list (quote shardcolumn) (list (quote table) schema tbl) col))
			'())
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase) (if (equal? tblvar alias)
			(list col_name (list (quote shardcolumn) (list (quote table) schema tbl) col))
			'())
		_ '())))

(define group_cache_partition_plan (lambda (schema tbl alias src grouptbl keys key_names)
	(begin
		(define hints (merge (map (produceN (count keys)) (lambda (i)
			(group_cache_partition_hint schema tbl alias (nth keys i) (nth key_names i))))))
		(if (or (not (source_is_base_table? src)) (empty_list? hints))
			nil
			(list (quote partitiontable) (list (quote table) schema grouptbl) (cons (quote list) hints))))))

(define lower_group_stage_prepare_using (lambda (all_stages lookup_stages stage)
	(begin
		(define src (gs_input stage))
		(define prepare_catalog (unique_stages_by_id (merge (list (list stage) all_stages))))
		(define fact_lookup (group_stage_lowering_catalog stage))
		(define raw_stage_lookup (if (lowering_catalog? lookup_stages)
			lookup_stages
			(if (lowering_catalog? fact_lookup)
				fact_lookup
				(unique_stages_by_id
					(merge (list
						prepare_catalog
						lookup_stages
						(qassoc_get (gs_facts stage) (quote stage_catalog) '())))))))
		/* A presence/scalar-first-probe source whose lookup domain is invariant
		for the whole query (only literals, parameters, or session values -- no
		outer-row column) is evaluated exactly once regardless of how many rows
		this stage's own input scan accepts. Bind it before rewriting so
		rewrite_scalar_first_probe_expr_using_index (via query_invariant_probe_
		binding_for_col) replaces every reference with that one bound value
		instead of a fresh per-row probe. Mirrors
		prepare_simple_query_block_physical_core_chosen. */
		(define invariant_probe_entries (query_invariant_probe_entries_for_stages raw_stage_lookup))
		(define stage_lookup (stage_lookup_with_query_invariant_probe_bindings
			raw_stage_lookup invariant_probe_entries))
		(define invariant_probe_bindings (query_invariant_probe_bindings invariant_probe_entries))
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
		/* Aggregate stages own their input query block. Apply the physical
		membership choice here too; otherwise the top-level preparation pass never
		reaches this nested block and eagerly materializes the candidate cache. */
		(define membership_requirement (if (query_block? src)
			(qassoc_get (qb_facts src) (quote membership_requirement) nil)
			nil))
		(define membership_src (if (and (query_block? src)
			(not (nil? membership_requirement)))
			(query_block_with_physical_membership_using stage_lookup src)
			src))
		(define rewritten_src (if (query_block? membership_src)
			(query_block_with_presence_probes_using stage_lookup membership_src)
			membership_src))
		(define rewrite_sources (if (query_block? membership_src) (qb_sources membership_src) '()))
		(define rewrite_default_alias (if (query_block? src)
			(qassoc_get (qb_facts membership_src) (quote default_alias) (if (empty_list? rewrite_sources) nil (source_alias (car rewrite_sources))))
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
		/* Aggregate column names belong to the immutable logical stage. Prepared
		input and rewritten descriptors below affect execution only. Base aggregate
		columns use their direct physical builder and need no canonical list here. */
		(define aggregate_cols (if (or query_input scalar_order_base_stage)
			(map ags (lambda (ag) (aggregate_col_name_using src ag)))
			'()))
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
			(map (produceN (count ags)) (lambda (i)
				(list (quote createcolumn)
					(list (quote table) schema grouptbl)
					(nth aggregate_cols i)
					"any"
					(quoted_runtime_list '())
					(quoted_runtime_list '()))))
			'()))
		(define collect_plan (if (not query_input)
			nil
			(if (union_block? src)
				(build_union_group_aggregates_insert_plan prepared_src src grouptbl keys key_names (list aggregate_count_descriptor))
				(build_query_group_collect_plan prepared_src grouptbl keys key_names))))
		/* A base-table bulk fill cannot evaluate a dependent stage value by
		itself. Those conditions are realized by the per-aggregate physical
		operator below (including projected RecSets), after nested stages have
		been prepared. */
		(define base_group_into_plan (if (or query_input
			(or scalar_order_base_stage (expr_refs_stage_output_alias? condition)))
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
					(build_union_group_aggregates_insert_plan prepared_src src grouptbl keys key_names ags)
					(build_query_group_aggregates_insert_plan prepared_src grouptbl keys key_names lowering_ags aggregate_cols))))
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
					src alias grouptbl keys key_names ags key_index expr
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
		(define partition_plan (group_cache_partition_plan schema tbl alias src grouptbl keys key_names))
		(define initial_fill_expr (if (nil? base_group_into_plan)
			nil
			(list (quote initialize_cache_table)
				'(session "__memcp_tx")
				(list (quote table) schema grouptbl)
				(list (quote list) (source_table_expr src))
				(list (quote lambda) '()
					(cons (quote !begin) (filter (list partition_plan aggregate_prepare_expr cleanup_plan) (lambda (expr) (not (nil? expr))))))
				base_group_fill
				(list (quote lambda) '() finalize_group_fill))))
		(define create_options (if (nil? initial_fill_expr)
			(quoted_runtime_list '("engine" "cache"))
			(list (quote list)
				"engine" "cache"
				"oninit" (list (quote lambda) '() initial_fill_expr))))
		(define group_cache_created (symbol "__group_cache_created"))
		(define keytable_init (list
			(list (quote lambda) (list group_cache_created)
				(list (quote !begin)
					(list (quote touch_keytable) (list (quote table) schema grouptbl))
					group_cache_created))
			(list (quote createtable) schema grouptbl create_cols create_options true)))
		(define lowered_plan_core (if scalar_query_stage
			(list (quote !begin)
				nested_prepare_expr
				(if initializer_owner keytable_init nil)
				ensure_agg_expr
				(build_scalar_single_query_stage_fill_plan
					prepared_src grouptbl keys key_names
					(car lowering_ags) (cadr lowering_ags)
					(car aggregate_cols) (cadr aggregate_cols)))
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
		/* A query-invariant presence/scalar-first probe (see the comment at
		raw_stage_lookup above) is bound exactly once here, ahead of whatever
		this stage's own prepare plan does, so every rewritten reference below
		reads that one binding instead of re-probing per row. */
		(define lowered_plan (if (empty_list? invariant_probe_bindings)
			lowered_plan_core
			(cons (quote !begin) (merge (list invariant_probe_bindings (list lowered_plan_core))))))
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
												(lower_column_expr_for_alias src filter_condition)))
											(define row_expr (list (quote resultrow)
												(cons (quote list) (lower_row_number_result_fields src col fields))))
											(define continuation_expr (list (quote lambda) (list (quote __row_number))
												(list (quote if)
													(lower_column_expr_for_alias src rewritten_condition)
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
					/* Session-dependent memberships are query-local domains. A base-table
					group cache uses persistent computed columns and cannot represent that
					domain as an immutable column recipe; keep the general query-group
					carrier, whose grouped insert is evaluated inside the query session. */
					(empty_list? (query_expr_session_reads probe_rewritten))
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
		((symbol driver_membership_subscan_probe) _stage _probe) true
		((quote driver_membership_subscan_probe) _stage _probe) true
		(cons head tail) (or
			(expr_contains_driver_membership? head)
			(reduce tail (lambda (found item)
				(or found (expr_contains_driver_membership? item)))
				false))
		_ false)))

(define stage_consumed_by_probe_source? (lambda (stage stages sources default_alias limit_value driver_condition)
	(reduce (coalesceNil sources '()) (lambda (found src)
		(or found
			(and (stage_output_relation? (source_relation src))
				(and (equal? (stage_output_relation_id (source_relation src)) (gs_id stage))
					(and
						(probeable_stage_output_source? stages src)
						(probeable_stage_output_source_for_block?
							stages sources default_alias limit_value driver_condition src))))))
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
		(define unordered_limit_candidate (and
			(empty_list? order_items)
			(query_limit_active? (qb_offset block) (qb_limit block))))
		(and (not (single_source? sources))
			(and (or unordered_limit_candidate
				(and (not (empty_list? order_items)) (or membership_candidate scalar_lookup_candidate)))
				(late_projection_sources_preserve_rows?
					(query_block_stage_lookup block) sources prelimit_sources default_alias
					(coalesceNil (qb_where block) true)))))))

(define stage_direct_prepare_source_visible? (lambda (block sources default_alias stage)
	(if (single_source? sources)
		true
		(not (or
			(row_number_stage_consumed_by_join? stage sources)
			(stage_consumed_by_membership_source? stage (qb_stages block) sources (qb_facts block))
			(stage_consumed_by_probe_source?
				stage (qb_stages block) sources default_alias (qb_limit block) (qb_where block)))))))

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
		(define late_projection (late_projection_candidate_block? block))
		(define prelimit_stage_ids (if late_projection
			(stage_ids_for_sources_with_closure all_stages (query_block_prelimit_sources block))
			'()))
		(include_shared_group_cache_stages
			(qb_stages block)
			(filter (query_block_stages_to_prepare_base_using all_stages block) (lambda (stage)
				(or
					(not late_projection)
					(stage_id_in? stage prelimit_stage_ids))))))))

(define query_block_stages_to_prepare (lambda (block)
	(query_block_stages_to_prepare_using (qb_stages block) block)))

(define query_block_bounded_scalar_probe_recipe_context? (lambda (block)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(or
			(probe_limit_bounded? (qb_limit block))
			(probe_context_unique_point? sources default_alias (qb_where block))))))

(define query_block_bounded_scalar_probe_recipe_keys (lambda (block entries)
	(if (not (query_block_bounded_scalar_probe_recipe_context? block))
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

/* A filter probe must retain its marker until scan lowering knows the number
of pre-limit rows it will evaluate. Turning it into a bounded direct recipe
here would incorrectly apply the output LIMIT to filter work. Projection-only
probes remain recipes and still execute after root braking. */
(define query_block_bounded_scalar_probe_recipe_entries (lambda (block entries)
	(begin
		(define prelimit_keys (scalar_query_probe_recipe_keys
			(query_block_prelimit_scalar_query_probe_recipe_entries block)))
		(filter entries (lambda (entry)
			(match entry
				'(stage requested_col)
				(not (has_assoc? prelimit_keys
					(scalar_query_probe_recipe_key stage requested_col)))
				_ false))))))

(define prepare_simple_query_block_physical_core_chosen (lambda (block)
	(begin
		(define raw_stage_lookup (query_block_stage_lookup block))
		(define invariant_probe_entries
			(query_invariant_probe_entries_for_stages raw_stage_lookup))
		(define stage_lookup
			(stage_lookup_with_query_invariant_probe_bindings
				raw_stage_lookup invariant_probe_entries))
		(define invariant_probe_bindings
			(query_invariant_probe_bindings invariant_probe_entries))
		(if (empty_list? (qb_stages block))
			(begin
				(define raw_probe_recipe_entries (query_block_scalar_query_probe_recipe_entries block))
				(define probe_recipe_entries (if (query_block_bounded_scalar_probe_recipe_context? block)
					(query_block_bounded_scalar_probe_recipe_entries block raw_probe_recipe_entries)
					'()))
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
					(merge (list
						invariant_probe_bindings
						probe_recipe_prepares
						probe_recipe_bindings))
					recipe_block))
			(begin
				(define stage_catalog (query_block_stage_catalog block))
				(define candidate_eager_stages (filter
					(query_block_stages_to_prepare_using stage_lookup block)
					(lambda (stage) (not (stage_shared_prepare? stage)))))
				(define dependency_graph (stage_dependency_graph stage_lookup))
				(define raw_prepared_block (if (single_source? (qb_sources block))
					(query_block_without_stages_after_prepare_using stage_lookup block)
					(query_block_with_prepared_sources_using stage_lookup block)))
				(define raw_probe_recipe_entries
					(query_block_scalar_query_probe_recipe_entries raw_prepared_block))
				(define probe_recipe_entries
					(if (query_block_bounded_scalar_probe_recipe_context? raw_prepared_block)
						(query_block_bounded_scalar_probe_recipe_entries
							raw_prepared_block raw_probe_recipe_entries)
						'()))
				/* Every promoted scalar probe owns its physical realization. Eager
				preparation here would make a second operator decision before the
				consumer's join-node cardinality is known. */
				(define probe_marker_stage_ids (stage_id_set
					(map raw_probe_recipe_entries (lambda (entry) (car entry)))))
				(define eager_stages (filter candidate_eager_stages (lambda (stage)
					(not (has_assoc? probe_marker_stage_ids (gs_id stage))))))
				(define eager_stage_lookup (stages_without_ids
					(lowering_catalog_stages stage_lookup)
					(extract_assoc probe_marker_stage_ids (lambda (id _owned) id))))
				(define eager_dependency_graph (stage_dependency_graph eager_stage_lookup))
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
				/* Different logical stage IDs may resolve to one physical carrier.
				Once that carrier is eager, no alias-equivalent stage may emit a
				second lazy initializer for the same session key. */
				(define lazy_catalog (stages_without_prepare_backbones stage_catalog eager_stages))
				(define core_block (query_block_without_stages
					(query_block_with_stage_catalog prepared_block stage_catalog)))
				(define lazy_stages (group_cache_stages_from_sources lazy_catalog (qb_sources core_block)))
				(list
					(merge (list
						invariant_probe_bindings
						probe_recipe_prepares
						probe_recipe_bindings
						(lazy_stage_prepare_bindings stage_catalog lazy_stages)
						(prepared_stage_bindings eager_stages)
						(lower_unique_stage_prepares_with_graph eager_dependency_graph eager_stage_lookup eager_stages)
						(lower_stage_materialize_all eager_stages)))
					core_block))))))

(define prepare_simple_query_block_physical_core (lambda (block)
	(prepare_simple_query_block_physical_core_chosen
		(query_block_with_physical_membership_choices
			(query_block_with_physical_membership_using
				(query_block_stage_lookup block)
				(query_block_with_physical_requirement_choices block))))))

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
		/* Eager preparation removes executable stages but retains their metadata so
		physical lowering can still prove group-key uniqueness and cardinality. */
		(if (or (nil? group_cache_stage)
			(qassoc_get (gs_facts group_cache_stage) (quote eager_prepared) false))
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
		(map (collect_stage_prepares selected) (lambda (stage)
			(lazy_stage_prepare_binding dependency_graph stage stages))))))

(define prepared_stage_binding (lambda (stage)
	(list
		(list (quote context) "session")
		(stage_prepare_key stage)
		(list (quote once) (list (quote lambda) '() true)))))

(define prepared_stage_bindings (lambda (stages)
	(map (collect_stage_prepares stages) prepared_stage_binding)))

(define query_block_has_aggregates? (lambda (block)
	(not (empty_list? (stage_aggregates_for_fields (qb_fields block))))))

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

(define late_projection_sources_preserve_rows_acc (lambda (stages sources remaining prelimit_sources default_alias final_condition bound_sources)
	(match (coalesceNil remaining '())
		(cons src rest) (if (source_alias_in_sources? (source_alias src) prelimit_sources)
			(late_projection_sources_preserve_rows_acc stages sources rest prelimit_sources default_alias final_condition
				(cons src bound_sources))
			(if (and (source_outer? src)
				(source_is_unique_lookup_from_sources? sources default_alias bound_sources src stages final_condition))
				(late_projection_sources_preserve_rows_acc stages sources rest prelimit_sources default_alias final_condition
					(cons src bound_sources))
				false))
		_ true)))

(define late_projection_sources_preserve_rows? (lambda (stages sources prelimit_sources default_alias final_condition)
	(late_projection_sources_preserve_rows_acc
		stages sources sources prelimit_sources default_alias final_condition '())))

(define ordered_join_sources_are_unique_lookups_acc (lambda (sources default_alias stages condition bound_sources remaining_sources)
	(match (coalesceNil remaining_sources '())
		(cons src rest) (and
			(source_is_unique_lookup_from_sources? sources default_alias bound_sources src stages condition)
			(ordered_join_sources_are_unique_lookups_acc
				sources default_alias stages condition (cons src bound_sources) rest))
		_ true)))

(define ordered_join_native_limit_supported? (lambda (sources plan default_alias order_items stages final_condition)
	(begin
		(define ordered_sources (join_optimizer_sources_for_order sources
			(join_optimizer_tree_aliases plan)))
		(define driver (car ordered_sources))
		(define order_parts (split_order_items_for_join_driver
			ordered_sources default_alias driver order_items stages final_condition '()))
		(define proof_condition (combine_where final_condition
			(join_optimizer_node_condition (join_optimizer_tree_predicates plan))))
		(and
			(empty_list? (nth order_parts 1))
			(ordered_join_sources_are_unique_lookups_acc
				sources default_alias stages proof_condition (list driver) (cdr ordered_sources))))))

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

(define membership_recset_var (lambda (src membership)
	(symbol (concat "__membership_recset_" (fnv_hash (serialize (list
		(source_schema src)
		(source_relation src)
		(gs_id (nth membership 0))
		(nth membership 1)
		(nth membership 2))))))))

(define replace_driver_membership_markers (lambda (src expr memberships)
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
			(recset_contains_call_expr (membership_recset_var src membership))
			(if (not (nil? marker))
				expr
				(match expr
					(cons head tail) (cons head (map tail (lambda (item)
						(replace_driver_membership_markers src item memberships))))
					_ expr))))))

(define membership_recset_bindings_using (lambda (src memberships consumer driver_rows_override)
	(filter (map memberships (lambda (membership)
		(begin
			(define expr (recset_project_join_expr_for_membership_using
				src membership consumer driver_rows_override))
			(if (nil? expr) nil (list membership (membership_recset_var src membership) expr)))))
		(lambda (binding) (not (nil? binding))))))

(define membership_recset_bindings (lambda (src memberships)
	(membership_recset_bindings_using src memberships nil nil)))

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

/* Build an exact RecSet formula only when every membership marker in the
predicate is a top-level AND term and at least one is a two-valued NOT EXISTS.
This lets NOT EXISTS drive a complement and EXISTS ... AND NOT EXISTS drive a
difference, while guarded OR branches keep the established residual-filter
path. */
(define driver_membership_recset_formula (lambda (src condition bindings)
	(begin
		(define terms (driver_membership_formula_terms_for_source src condition))
		(define has_negative (reduce terms (lambda (found term)
			(or found (nth term 4))) false))
		(define term_bindings (map terms (lambda (term)
			(membership_binding_for_marker bindings (list (nth term 0) (nth term 1))))))
		(if (or (not has_negative)
			(or (not (equal? (count terms) (count bindings)))
				(reduce term_bindings (lambda (missing binding)
					(or missing (nil? binding))) false)))
			nil
			(begin
				(define positive_exprs (merge (map (produceN (count terms)) (lambda (i)
					(if (nth (nth terms i) 4) '() (list (nth (nth term_bindings i) 2)))))))
				(define negative_exprs (merge (map (produceN (count terms)) (lambda (i)
					(if (nth (nth terms i) 4) (list (nth (nth term_bindings i) 2)) '())))))
				(define negative_union (if (single_source? negative_exprs)
					(car negative_exprs)
					(list (quote recset_union) (cons (quote list) negative_exprs))))
				(define formula (if (empty_list? positive_exprs)
					(list (quote recset_not) negative_union)
					(begin
						(define positive_intersection (if (single_source? positive_exprs)
							(car positive_exprs)
							(list (quote recset_intersect) (cons (quote list) positive_exprs))))
						(list (quote recset_difference)
							(cons (quote list) (cons positive_intersection negative_exprs))))))
				(list terms formula negative_exprs positive_exprs))))))

(define membership_keyset_descriptor (lambda (membership)
	(begin
		(define stage (nth membership 0))
		(define input (gs_input stage))
		(define input_src (if (query_block? input)
			(if (single_source? (qb_sources input)) (car (qb_sources input)) nil)
			input))
		(define keys (gs_keys stage))
		(define condition (if (query_block? input)
			(combine_where (qb_where input) (source_join_expr input_src))
			(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)))
		(define source_col (if (or (nil? input_src) (empty_list? keys))
			nil
			(direct_column_name_for_alias input_src (car keys))))
		(if (or (nil? source_col) (not (source_is_base_table? input_src)))
			nil
			(list input_src source_col condition)))))

(define membership_keysets_supported? (lambda (memberships)
	(begin
		(define descriptors (map memberships membership_keyset_descriptor))
		(define supported (and (not (empty_list? descriptors))
			(not (reduce descriptors (lambda (missing descriptor)
				(or missing (nil? descriptor))) false))))
		(if (not supported)
			false
			(begin
				(define first_descriptor (car descriptors))
				(reduce (cdr descriptors) (lambda (same descriptor)
					(and same
						(and (equal? (source_table_expr (nth descriptor 0))
							(source_table_expr (nth first_descriptor 0)))
							(equal? (nth descriptor 1) (nth first_descriptor 1))))) true))))))

(define membership_keyset_parts (lambda (membership)
	(begin
		(define descriptor (membership_keyset_descriptor membership))
		(if (nil? descriptor)
			nil
			(begin
				(define input_src (nth descriptor 0))
				(define source_col (nth descriptor 1))
				(define condition (nth descriptor 2))
				(define alias (source_alias input_src))
				(define filtercols (extract_columns_for_alias input_src condition))
				(list
					(source_table_expr input_src)
					source_col
					(list (quote scan_recset)
						'(session "__memcp_tx")
						(source_table_expr input_src)
						(cons (quote list) filtercols)
						(list (quote lambda)
							(map filtercols (lambda (col) (symbol (concat alias "." col))))
							(lower_column_expr_for_alias input_src condition)))))))))

(define membership_keyset_bindings (lambda (memberships)
	(begin
		(define parts (map memberships membership_keyset_parts))
		(define supported (and (not (empty_list? parts))
			(not (reduce parts (lambda (missing item) (or missing (nil? item))) false))))
		(define compatible (and supported
			(reduce (cdr parts) (lambda (same item)
				(and same
					(and (equal? (nth item 0) (nth (car parts) 0))
						(equal? (nth item 1) (nth (car parts) 1))))) true)))
		(if (not compatible)
			'()
			(begin
				(define keyset_var (symbol "__membership_keyset"))
				(define candidate_recsets (map parts (lambda (item) (nth item 2))))
				(define candidate_recset (if (single_source? candidate_recsets)
					(car candidate_recsets)
					(list (quote recset_union) (cons (quote list) candidate_recsets))))
				(define keyset_expr (list (quote recset_key_index)
					'(session "__memcp_tx")
					candidate_recset
					(quoted_runtime_list (list (nth (car parts) 1)))))
				(map memberships (lambda (membership)
					(list membership keyset_var keyset_expr))))))))

(define wrap_membership_keyset_bindings (lambda (bindings body)
	(if (empty_list? bindings)
		body
		(list
			(list (quote lambda) (list (nth (car bindings) 1)) body)
			(nth (car bindings) 2)))))

(define replace_driver_membership_keyset_markers (lambda (expr bindings)
	(begin
		(define marker (driver_membership_probe_term expr))
		(define binding (if (nil? marker) nil (membership_binding_for_marker bindings marker)))
		(if (not (nil? binding))
			(list (nth binding 1) (nth marker 1))
			(if (not (nil? marker))
				expr
				(match expr
					(cons head tail) (cons head (map tail (lambda (item)
						(replace_driver_membership_keyset_markers item bindings))))
					_ expr))))))

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
						(lower_column_expr_for_alias src branch))))
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
				(define source_table (source_table_expr_using (query_block_stage_catalog block) src))
				(define order_items (coalesceNil (qb_order block) '()))
				(define scan_order_supported (order_items_belong_to_source? src order_items))
				(define bounded (query_limit_active? (qb_offset block) (qb_limit block)))
				(define memberships (driver_memberships_for_source src condition))
				/* For a bounded ordered driver, union branch-local source candidates
				before building one immutable key index. The driver stays a scan_order
				and probes that index in its filter, so Top-K braking remains effective
				without projecting every candidate into a target RecSet. */
				(define keyset_membership_probe (ordered_driver_membership_keyset? (qb_facts block)))
				(define membership_keysets (if keyset_membership_probe
					(membership_keyset_bindings memberships)
					'()))
				(define use_membership_keysets (and keyset_membership_probe
					(equal? (count membership_keysets) (count memberships))))
				(define forced_candidate_membership (reduce memberships (lambda (forced membership)
					(or forced (equal?
						(planner_physical_override (concat "membership_carrier:" (gs_id (nth membership 0))))
						"candidate_keyset"))) false))
				(define prefer_membership_filter (and scan_order_supported
					(and bounded
						(and (not forced_candidate_membership)
							(and (qassoc_get (qb_facts block) (quote membership_driver_alternative) false)
								(membership_estimate_broad? (qb_facts block)))))))
				(define membership_bindings (if (or keyset_membership_probe prefer_membership_filter)
					'()
					(membership_recset_bindings_using src memberships
						(if bounded (quote order_limit) (quote filter))
						(if (and scan_order_supported bounded)
							(probe_limit_work_rows (qb_limit block)) nil))))
				(define bound_memberships (map membership_bindings (lambda (binding) (nth binding 0))))
				(define membership_formula (if keyset_membership_probe
					nil
					(driver_membership_recset_formula src condition membership_bindings)))
				/* A membership predicate which is implied by the whole WHERE clause is
				eligible to become the scan driver. A branch-local predicate below OR is
				not: its RecSet must remain a probe or rows accepted by sibling branches
				would disappear. This distinction also preserves the established fast
				single-IN path while allowing several guarded RecSets in one scan. */
				(define direct_membership (driver_membership_for_source src condition))
				(define membership_formula_driver (and
					(not (nil? membership_formula))
					(not prefer_membership_filter)))
				(define membership_formula_residual (if membership_formula_driver
					(strip_driver_membership_formula_terms condition (car membership_formula))
					condition))
				/* If ordinary conjuncts already define a narrower exact base set,
				subtract NOT EXISTS matches from that set instead of complementing the
				whole visible table. This is both exact and the useful Difference case. */
				(define membership_formula_difference_driver (and membership_formula_driver
					(and (empty_list? (nth membership_formula 3))
						(not (equal? membership_formula_residual true)))))
				(define membership_formula_expr (if membership_formula_difference_driver
					(begin
						(define base_cols (extract_columns_for_alias src membership_formula_residual))
						(define base_recset (list (quote scan_recset)
							'(session "__memcp_tx")
							source_table
							(cons (quote list) base_cols)
							(list (quote lambda)
								(map base_cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
								(lower_column_expr_for_alias src membership_formula_residual))))
						(list (quote recset_difference)
							(cons (quote list) (cons base_recset (nth membership_formula 2)))))
					(if membership_formula_driver (nth membership_formula 1) nil)))
				(define membership_driver (and
					(not membership_formula_driver)
					(and (not (nil? direct_membership))
						(and (not prefer_membership_filter)
							(and (not (empty_list? membership_bindings))
								(and (empty_list? (cdr membership_bindings))
									(equal? direct_membership (car bound_memberships))))))))
				(define membership_filter (and
					(not (or membership_driver membership_formula_driver))
					(not (empty_list? membership_bindings))))
				(define filter_condition (if use_membership_keysets
					(replace_driver_membership_keyset_markers condition membership_keysets)
					(if membership_formula_driver
						(if membership_formula_difference_driver true membership_formula_residual)
						(if membership_driver
							(strip_driver_membership_for_source src condition direct_membership)
							(replace_driver_membership_markers src condition bound_memberships)))))
				(define filtercols (merge_unique (list
					(if membership_filter (list "$recset_contains") '())
					(extract_columns_for_alias src filter_condition))))
				(define fieldcols (merge_unique (extract_assoc fields (lambda (_title expr)
					(extract_columns_for_alias src expr)))))
				(define ordercols (if (empty_list? order_items) '() (scan_order_sort_columns_for_alias src order_items)))
				(define mapcols fieldcols)
				(define membership_candidates (if membership_filter
					(membership_or_candidate_recset src source_table condition membership_bindings)
					nil))
				(define table_expr (if membership_formula_driver
					membership_formula_expr
					(if membership_driver
						(nth (car membership_bindings) 2)
						(coalesceNil membership_candidates source_table))))
				(define filter_expr (list (quote lambda)
					(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
					(lower_column_expr_for_alias src filter_condition)))
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
								(cons (quote list) (if (empty_list? order_items) '() (order_relations_for_source src order_items)))
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
				(if use_membership_keysets
					(wrap_membership_keyset_bindings membership_keysets scan_plan)
					(if membership_filter
						(wrap_membership_recset_bindings membership_bindings scan_plan)
						scan_plan)))))))

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

(define lower_join_result_fields (lambda (all_sources default_alias fields probe_work_rows)
	(map_assoc fields (lambda (_title expr)
		(lower_column_expr_for_join_in_context
			all_sources default_alias expr probe_work_rows)))))

(define recset_contains_callback_symbol (symbol "__recset_contains"))

(define scan_callback_symbol_for_alias (lambda (alias col)
	(if (equal? col "$tx")
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

(define acceptance_expr_source_aliases (lambda (sources default_alias expr)
	(map
		(filter (coalesceNil sources '()) (lambda (src)
			(expr_refs_alias? default_alias (source_alias src) expr)))
		source_alias)))

(define acceptance_required_aliases_expand (lambda (sources default_alias required)
	(merge_unique (list required
		(merge (map (filter (coalesceNil sources '()) (lambda (src)
			(contains? required (source_alias src)))) (lambda (src)
				(acceptance_expr_source_aliases sources default_alias
					(coalesceNil (source_join_expr src) true)))))))))

(define acceptance_required_aliases_closure (lambda (sources default_alias required)
	(begin
		(define expanded (acceptance_required_aliases_expand sources default_alias required))
		(if (equal? (count expanded) (count required))
			expanded
			(acceptance_required_aliases_closure sources default_alias expanded)))))

(define acceptance_required_sources (lambda (sources default_alias condition)
	(begin
		(define seeds (map (filter (coalesceNil sources '()) (lambda (src)
			(or
				(not (source_outer? src))
				(expr_refs_alias? default_alias (source_alias src) condition))))
			source_alias))
		(define required (acceptance_required_aliases_closure sources default_alias seeds))
		(filter (coalesceNil sources '()) (lambda (src)
			(contains? required (source_alias src)))))))

/* A bounded ordered join may use the logical driver directly only when every
remaining source is a proven unique lookup. The ordinary filter stays driver-local
and indexable. The nested lookup acceptance runs in scan_order's globally ordered
stream before its native OFFSET/LIMIT counters; projection only sees the window. */
(define join_ordered_native_limit_plan (lambda (schema all_sources plan default_alias needed_exprs final_condition fields order_items offset_value limit_value stages facts)
	(begin
		(define ordered_aliases (join_optimizer_tree_aliases plan))
		(define ordered_sources (join_optimizer_sources_for_order all_sources ordered_aliases))
		(define src (car ordered_sources))
		(define remaining_sources (cdr ordered_sources))
		/* The ordered driver is consumed here. Preserve the optimizer's remaining
		subtree instead of rebuilding a left-deep tree from the source catalog. */
		(define remaining_plan (join_optimizer_tree_without_aliases
			plan (list (source_alias src))))
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
		(define membership_plan (if (nil? membership) nil
			(recset_project_join_plan_for_membership_using src membership
				(if (query_limit_active? offset_value limit_value) (quote order_limit) (quote filter))
				(if (query_limit_active? offset_value limit_value)
					(probe_limit_work_rows limit_value) nil))))
		(define membership_strategy (if (nil? membership_plan) nil (car membership_plan)))
		(define membership_table_expr (if (and
			(not (nil? membership_plan))
			(equal? membership_strategy "candidate_keyset"))
			(cadr membership_plan)
			nil))
		(define effective_membership (if (nil? membership_table_expr) nil membership))
		(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
		/* The node-local physical choice is final. A candidate keyset becomes the
		ordered carrier; a driver-probe choice leaves the marker in the base-table
		filter so its established direct probe lowering remains in charge. */
		(define filter_condition effective_condition)
		/* Acceptance runs before OFFSET/LIMIT and therefore contains only joins
		which can reject a driver row. Projection-only nullable lookups belong to
		the map callback after the native window and must not be probed twice. */
		(define acceptance_sources (acceptance_required_sources
			remaining_sources default_alias remaining_condition))
		(define acceptance_aliases (map acceptance_sources source_alias))
		(define removed_acceptance_aliases (filter (map remaining_sources source_alias)
			(lambda (candidate) (not (contains? acceptance_aliases candidate)))))
		(define acceptance_plan (join_optimizer_tree_without_aliases
			remaining_plan removed_acceptance_aliases))
		(define acceptance_needed_exprs (merge (list
			(list remaining_condition)
			(source_join_exprs acceptance_sources))))
		(define acceptance_probe (if (empty_list? acceptance_sources)
			(lower_column_expr_for_join_truth_context
				all_sources default_alias remaining_condition acceptance_probe_work_rows)
			(build_join_scan_reduce_using_recipe
				schema all_sources acceptance_plan default_alias acceptance_needed_exprs remaining_condition true
				'() 0 1 true acceptance_probe_work_rows nil stages
				(list (quote lambda) (list (quote _accepted) (quote _row)) true)
				false
				(list (quote lambda) (list (quote accepted) (quote shard_accepted))
					(list (quote or) (quote accepted) (quote shard_accepted))))))
		(define filtercols (merge_unique (list
			(join_cols_for_alias all_sources default_alias alias (list effective_condition)))))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			(lower_column_expr_for_join_truth_context
				all_sources default_alias filter_condition acceptance_probe_work_rows)))
		(define acceptance_cols (join_cols_for_alias all_sources default_alias alias acceptance_needed_exprs))
		(define acceptance_expr (list (quote lambda)
			(map acceptance_cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			acceptance_probe))
		(define offset (coalesceNil offset_value 0))
		(define limit (coalesceNil limit_value -1))
		(define projection_probe_work_rows (coalesceNil
			(probe_limit_work_rows limit_value)
			acceptance_probe_work_rows))
		(define row_expr (list (quote resultrow)
			(cons (quote list) (lower_join_result_fields
				all_sources default_alias fields projection_probe_work_rows))))
		(define mapcols (join_cols_for_alias all_sources default_alias alias needed_exprs))
		(define projection (build_join_scan_pipeline_using_recipe
			schema all_sources remaining_plan default_alias needed_exprs remaining_condition row_expr
			remaining_order_items 0 -1 true projection_probe_work_rows nil stages
		))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			projection))
		(define scan_expr (list (quote scan_order)
			'(session "__memcp_tx")
			(coalesceNil membership_table_expr (source_table_expr_using stages src))
			(cons (quote list) filtercols)
			filter_expr
			(cons (quote list) (scan_order_sort_columns_for_join_driver
				ordered_sources default_alias src driver_order_items stages final_condition))
			(cons (quote list) (order_relations_for_source src driver_order_items))
			0 offset limit
			(cons (quote list) mapcols)
			map_expr nil nil false nil
			(cons (quote list) acceptance_cols)
			acceptance_expr))
		scan_expr)))

(define without_col (lambda (cols col)
	(filter (coalesceNil cols '()) (lambda (item) (not (equal? item col))))))

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

(define build_join_row_number_scan_pipeline (lambda (schema all_sources src default_alias needed_exprs remaining_condition row_expr stage_filter membership_var membership_filter column_recipe stages result_mode probe_context combines_state continuation outer_scan)
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
					(lower_column_expr_for_join_truth_context
						all_sources default_alias filter_condition
						(probe_work_context_rows_for_alias probe_context alias))))
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

(define probe_work_context? (lambda (value)
	(match value
		(cons ((symbol probe_work_context) true) _rest) true
		(cons ((quote probe_work_context) true) _rest) true
		_ false)))

(define probe_work_context_default_rows (lambda (context)
	(if (probe_work_context? context)
		(qassoc_get context (quote default) nil)
		nil)))

(define probe_work_context_driver_alias (lambda (context)
	(if (probe_work_context? context)
		(qassoc_get context (quote driver_alias) nil)
		nil)))

(define probe_work_context_rows_for_alias (lambda (context alias)
	(if (probe_work_context? context)
		(qassoc_get (qassoc_get context (quote by_alias) '()) alias
			(qassoc_get context (quote default) nil))
		nil)))

(define physical_probe_predicate_selectivity (lambda (predicates)
	(reduce (coalesceNil predicates '()) (lambda (product predicate)
		(* product (join_order_pred_selectivity predicate))) 1)))

(define physical_probe_work_index_scaled (lambda (index factor)
	(map (coalesceNil index '()) (lambda (entry)
		(list (car entry)
			(if (and (number? (cadr entry)) (number? factor))
				(* (cadr entry) factor)
				nil))))))

/* Return (rows-per-invocation, alias->scan-work) for a fixed physical join
tree. Right subtrees are continuations of every surviving left row, so their
probe count must be scaled at that node rather than inherited from the root
driver. This is physical costing metadata only; it never enters logical IR. */
(define physical_join_tree_probe_work (lambda (tree sources)
	(match tree
		((symbol join-leaf) alias predicates) (begin
			(define src (join_optimizer_source_by_alias sources alias))
			(define source_rows (if (nil? src) nil (planner_source_row_count src)))
			(define rows (if (number? source_rows)
				(max 1 (* source_rows (physical_probe_predicate_selectivity predicates)))
				nil))
			(list rows (list (list alias rows))))
		((quote join-leaf) alias predicates)
		(physical_join_tree_probe_work
			(list (symbol "join-leaf") alias predicates) sources)
		((symbol join-leaf) alias)
		(physical_join_tree_probe_work
			(list (symbol "join-leaf") alias '()) sources)
		((quote join-leaf) alias)
		(physical_join_tree_probe_work
			(list (symbol "join-leaf") alias '()) sources)
		((symbol join-node) kind left right predicates) (begin
			(define left_work (physical_join_tree_probe_work left sources))
			(define right_work (physical_join_tree_probe_work right sources))
			(define left_rows (car left_work))
			(define right_rows (car right_work))
			(define selectivity (physical_probe_predicate_selectivity predicates))
			(define right_invocations (if (number? left_rows)
				(* left_rows selectivity)
				nil))
			(define joined_rows (if (and (number? left_rows) (number? right_rows))
				(max 1 (* left_rows right_rows selectivity))
				nil))
			(define output_rows (if (and (equal? kind (quote left-outer)) (number? left_rows))
				(max left_rows (coalesceNil joined_rows left_rows))
				joined_rows))
			(list output_rows (merge (list
				(cadr left_work)
				(physical_probe_work_index_scaled (cadr right_work) right_invocations)))))
		((quote join-node) kind left right predicates)
		(physical_join_tree_probe_work
			(list (symbol "join-node") kind left right predicates) sources)
		_ (list nil '()))))

(define join_scan_probe_context (lambda (tree sources default_rows)
	(begin
		(define work (if (nil? tree) (list nil '())
			(physical_join_tree_probe_work tree sources)))
		(list
			(list (quote probe_work_context) true)
			(list (quote driver_alias) (if (nil? tree) nil
				(join_optimizer_tree_first_alias tree)))
			(list (quote default) (coalesceNil (car work) default_rows))
			(list (quote by_alias) (cadr work))))))

/* A source-local text predicate is already a canonical logical predicate by
the time it reaches this boundary. For a selective join driver, enumerate an
exact RecSet carrier so expensive downstream ACL/join work runs only for the
matching record IDs. This is a physical choice over the normalized predicate;
it does not recover parser spelling or leak a scan into logical IR. */
(define selective_text_filter_candidate (lambda (src all_sources default_alias condition)
	(begin
		(define alias (source_alias src))
		(define aliases (source_aliases all_sources))
		(define input_rows (planner_source_row_count src))
		(if (or (not (number? input_rows)) (< input_rows 1024))
			nil
			(reduce (split_and_terms (coalesceNil condition true)) (lambda (best term)
				(if (or (not (expr_contains_text_match? term))
					(not (equal? (join_hypergraph_expr_aliases default_alias aliases term)
						(list alias))))
					best
					(begin
						(planner_record_session_value_guards term)
						(define estimate (planner_source_filter_estimate src term 512))
						(define rows (membership_estimated_matching_rows estimate input_rows nil))
						(define work (membership_source_work_profile src term true))
						(define candidate (if (number? rows)
							(list
								(list (quote predicate) term)
								(list (quote rows) rows)
								(list (quote input_rows) input_rows)
								(list (quote work) work)
								(list (quote estimate) estimate))
							nil))
						(if (or (nil? candidate)
							(and (not (nil? best))
								(<= (qassoc_get best (quote rows) input_rows) rows)))
							best candidate))))
				nil)))))

/* The text predicate itself is common work. The carrier decision determines
whether the downstream join continuation is entered for every driver row or
only for exact matches after a parallel RecSet-producing pass. Reuse the
calibrated physical primitives; do not hide another selectivity threshold in
this lowering boundary. */
(define selective_text_filter_scan_cost (lambda (src candidate)
	(begin
		(define input_rows (qassoc_get candidate (quote input_rows) 0))
		(define work (qassoc_get candidate (quote work) '()))
		(planner_cost
			(* (qassoc_get work (quote scan_invocations) 1)
				planner_membership_scan_invocation_ns)
			(+
				(* input_rows planner_membership_scan_row_ns)
				(* (qassoc_get work (quote filter_value_rows) 0)
					planner_membership_filter_column_row_ns)
				(* (qassoc_get work (quote expression_operation_rows) 0)
					planner_membership_expression_operation_row_ns)
				(* (qassoc_get work (quote broad_text_match_rows) 0)
					planner_membership_broad_text_match_row_ns)
				(* (qassoc_get work (quote broad_text_match_bytes) 0)
					planner_membership_broad_text_match_byte_ns))
			0 0 0 0 0 0 input_rows 0.75))))

(define selective_text_filter_base_cost (lambda (src candidate)
	(begin
		(define input_rows (qassoc_get candidate (quote input_rows) 0))
		(planner_cost_add
			(selective_text_filter_scan_cost src candidate)
			(planner_join_work_cost input_rows 0.65)
			(qassoc_get candidate (quote rows) input_rows) 0.65))))

(define selective_text_filter_recset_cost (lambda (src candidate)
	(begin
		(define input_rows (qassoc_get candidate (quote input_rows) 0))
		(define rows (qassoc_get candidate (quote rows) input_rows))
		(planner_cost_add
			(planner_cost_add
				(selective_text_filter_scan_cost src candidate)
				(planner_cost planner_membership_recset_startup_ns
					(* rows planner_membership_scan_row_ns) 0 0 0
					(* rows planner_membership_recset_build_row_ns)
					(* rows 8) 0 rows 0.65)
				rows 0.65)
			(planner_join_work_cost rows 0.65)
			rows 0.65))))

(define choose_selective_text_filter_carrier (lambda (src candidate)
	(if (nil? candidate)
		(list "fused_base_scan" nil)
		(begin
			(define alias (source_alias src))
			(define decision_id (concat "selective_text_filter_carrier:" alias))
			(define recset_cost (selective_text_filter_recset_cost src candidate))
			(define base_cost (selective_text_filter_base_cost src candidate))
			(define work (qassoc_get candidate (quote work) '()))
			(define normal_choice (if (planner_cost_better? recset_cost base_cost)
				"scan_recset" "fused_base_scan"))
			(define alternatives (list "scan_recset" "fused_base_scan"))
			(define chosen (planner_physical_choice decision_id normal_choice alternatives))
			(define forced (planner_physical_override decision_id))
			(planner_record_physical_decision (list
				(list "decision_id" decision_id)
				(list "decision" "selective_text_filter_carrier")
				(list "decision_site" "join_scan_leaf")
				(list "source" alias)
				(list "chosen" chosen)
				(list "normally_chosen" normal_choice)
				(list "selection" (if (nil? forced) "cost" "forced"))
				(list "reason" (if (nil? forced) "lowest_total_ns" "calibration_override"))
				(list "inputs" (list
					(list "candidate_rows" (qassoc_get candidate (quote rows) nil))
					(list "candidate_input_rows" (qassoc_get candidate (quote input_rows) nil))
					(list "candidate_scan_invocations" (qassoc_get work (quote scan_invocations) 1))
					(list "candidate_filter_columns" (qassoc_get work (quote filter_columns) 0))
					(list "candidate_expression_operations" (qassoc_get work (quote expression_operations) 0))
					(list "candidate_expression_depth" (qassoc_get work (quote expression_depth) 0))
					(list "candidate_broad_text_match_rows" (qassoc_get work (quote broad_text_match_rows) 0))
					(list "candidate_broad_text_match_bytes" (qassoc_get work (quote broad_text_match_bytes) 0))
					(list "candidate_filter_value_rows" (qassoc_get work (quote filter_value_rows) 0))
					(list "candidate_expression_operation_rows" (qassoc_get work (quote expression_operation_rows) 0))
					(list "driver_rows" (qassoc_get candidate (quote rows) nil))
					(list "driver_input_rows" (qassoc_get candidate (quote input_rows) nil))
					(list "density" (/ (qassoc_get candidate (quote rows) 0)
						(qassoc_get candidate (quote input_rows) 1)))))
				(list "alternatives" (list
					(list
						(list "plan" "scan_recset")
						(list "status" (if (equal? chosen "scan_recset") "chosen" "rejected"))
						(list "reason" (if (equal? chosen "scan_recset") "selected" "higher_total_ns_or_forced_alternative"))
						(list "cost" (planner_cost_explain recset_cost)))
					(list
						(list "plan" "fused_base_scan")
						(list "status" (if (equal? chosen "fused_base_scan") "chosen" "rejected"))
						(list "reason" (if (equal? chosen "fused_base_scan") "selected" "higher_total_ns_or_forced_alternative"))
						(list "cost" (planner_cost_explain base_cost)))))))
			(list chosen candidate)))))

(define selective_text_filter_recset_expr (lambda (stages src candidate)
	(begin
		(define alias (source_alias src))
		(define predicate (qassoc_get candidate (quote predicate) true))
		(define cols (extract_columns_for_alias src predicate))
		(list (quote scan_recset)
			'(session "__memcp_tx")
			(source_table_expr_using stages src)
			(cons (quote list) cols)
			(list (quote lambda)
				(map cols (lambda (col) (scan_callback_symbol_for_alias alias col)))
				(lower_column_expr_for_alias src predicate))))))

(define strip_selective_text_filter_candidate (lambda (condition candidate)
	(if (nil? candidate)
		condition
		(begin
			(define predicate (qassoc_get candidate (quote predicate) true))
			(combine_where_terms
				(filter (split_and_terms (coalesceNil condition true)) (lambda (term)
					(not (equal? term predicate))))
				true)))))

(define build_join_scan_leaf_using_recipe (lambda (schema all_sources leaf future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
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
		(define membership_plan (if (or (nil? membership) (or delay_limit_after_join (not allow_membership_recset)))
			nil
			(recset_project_join_plan_for_membership_using src membership
				(if (join_scan_reduce? result_mode) (quote aggregate)
					(if (query_limit_active? offset_value limit_value) (quote order_limit) (quote filter)))
				(if (and (not (empty_list? current_order_items))
					(query_limit_active? offset_value limit_value))
					(probe_limit_work_rows limit_value) nil))))
		(define membership_strategy (if (nil? membership_plan) nil (car membership_plan)))
		(define membership_table_expr (if (and
			(not (nil? membership_plan))
			(equal? membership_strategy "candidate_keyset"))
			(cadr membership_plan)
			nil))
		(if (expr_contains_driver_membership? condition)
			(planner_record_physical_decision (list
				(list "decision" "join_leaf_membership_consumer")
				(list "chosen" (if (nil? membership_table_expr)
					"per_row_probe" "projected_recset"))
				(list "inputs" (list
					(list "marker_resolved_to_driver" (not (nil? membership)))
					(list "marker_resolved_to_any_source"
						(reduce all_sources (lambda (found candidate_src)
							(or found (not (nil? (driver_membership_for_source candidate_src condition)))))
							false))
					(list "carrier_allowed" allow_membership_recset)
					(list "limit_delayed" delay_limit_after_join)
					(list "projection_built" (not (nil? membership_table_expr)))))))
			nil)
		(define effective_membership (if (nil? membership_table_expr) nil membership))
		(define effective_condition (strip_driver_membership_for_source src condition effective_membership))
		(define row_number_stage_filter (row_number_stage_for_source stages src effective_condition))
		/* A candidate-keyset choice makes the projected row positions this leaf's
		scan carrier even when later join leaves are continuations. A driver-probe
		choice leaves the logical marker in the established direct probe path. The
		row-number pipeline still owns a base-table carrier, so it consumes a chosen
		candidate RecSet as a membership filter until it accepts an explicit source. */
		(define membership_driver (and
			(not (nil? membership_table_expr))
			(nil? row_number_stage_filter)))
		(define membership_filter (and (not (nil? membership_table_expr)) (not membership_driver)))
		(define text_filter_eligible (and
			(not membership_driver)
			(nil? row_number_stage_filter)
			(> (count all_sources) 1)
			(equal? alias (probe_work_context_driver_alias probe_context))))
		(define text_filter_candidate_before_point_check (if text_filter_eligible
			(selective_text_filter_candidate src all_sources default_alias effective_condition)
			nil))
		/* Most join leaves have no text predicate. Avoid walking their full
		condition a second time to prove point access before the cheap candidate
		classifier has established that this decision is relevant at all. */
		(define text_filter_candidate (if (and
			(not (nil? text_filter_candidate_before_point_check))
			(source_unique_point_condition? src effective_condition))
			nil
			text_filter_candidate_before_point_check))
		(define text_filter_plan (choose_selective_text_filter_carrier src text_filter_candidate))
		(define text_filter_recset (and (not (nil? text_filter_candidate))
			(equal? (car text_filter_plan) "scan_recset")))
		(define residual_condition
			(strip_selective_text_filter_candidate effective_condition
				(if text_filter_recset text_filter_candidate nil)))
		(define membership_var (symbol "__membership_recset"))
		(define membership_filter_expr (if membership_filter
			(recset_contains_call_expr membership_var)
			true))
		(define filter_condition (combine_where membership_filter_expr residual_condition))
		(define filtercols (merge_unique (list
			(if membership_filter (list "$recset_contains") '())
			(join_filter_cols_for_alias all_sources default_alias alias residual_condition))))
		(define recipe_mapcols (join_recipe_mapcols column_recipe alias))
		(define raw_mapcols (if (nil? column_recipe)
			(join_cols_for_alias all_sources default_alias alias needed_exprs)
			recipe_mapcols))
		(define mapcols raw_mapcols)
		(define table_expr (if membership_driver membership_table_expr
			(if (not text_filter_recset)
				(source_table_expr_using stages src)
				(selective_text_filter_recset_expr stages src text_filter_candidate))))
		(define lowered_filter_condition (mark_outer_join_symbols
			all_sources
			alias
			(lower_column_expr_for_join_truth_context
				all_sources default_alias filter_condition
				(probe_work_context_rows_for_alias probe_context alias))))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			lowered_filter_condition))
		(define continuation_expr (continuation remaining_condition row_expr remaining_order_items))
		(define map_body (if (equal? post_outer_condition true)
			continuation_expr
			(list (quote if)
				(lower_column_expr_for_join_truth_context
					all_sources default_alias post_outer_condition
					(probe_work_context_rows_for_alias probe_context alias))
				continuation_expr
				(join_scan_skip_expr result_mode))))
		(define map_expr (list (quote lambda)
			(map mapcols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			map_body))
		(define combines_state (not (empty_list? future_aliases)))
		(define reduce_expr (join_scan_reduce_expr result_mode combines_state))
		(define scan_expr
			(if (not (nil? row_number_stage_filter))
				(build_join_row_number_scan_pipeline schema all_sources src default_alias needed_exprs remaining_condition row_expr row_number_stage_filter membership_var membership_filter column_recipe stages result_mode probe_context combines_state continuation outer_scan)
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
						(cons (quote list) (if (empty_list? current_order_items) '() (order_relations_for_source src current_order_items)))
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
(define build_join_tree_scan_using_recipe (lambda (schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
	(match tree
		((symbol join-leaf) _alias)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
		((quote join-leaf) alias)
		(build_join_tree_scan_using_recipe schema all_sources (make_join_optimizer_leaf alias) future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
		((symbol join-leaf) _alias _predicates)
		(build_join_scan_leaf_using_recipe schema all_sources tree future_aliases default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
		((quote join-leaf) alias predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(list (quote join-leaf) alias predicates) future_aliases default_alias needed_exprs
			final_condition row_expr order_items offset_value limit_value allow_membership_recset
			column_recipe stages result_mode probe_context continuation outer_scan)
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
				order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context
				(lambda (left_condition left_row_expr left_order_items)
					(build_join_tree_scan_using_recipe
						schema all_sources right future_aliases
						default_alias needed_exprs (combine_where left_condition node_condition) left_row_expr
						left_order_items 0 -1 allow_membership_recset column_recipe stages result_mode probe_context continuation
						(equal? kind (quote left-outer))))
				outer_scan))
		((quote join-node) kind left right predicates)
		(build_join_tree_scan_using_recipe schema all_sources
			(make_join_optimizer_node kind left right predicates) future_aliases
			default_alias needed_exprs final_condition row_expr order_items offset_value limit_value
			allow_membership_recset column_recipe stages result_mode probe_context continuation outer_scan)
		_ (neumann_fail "build_queryplan" "malformed logical join tree"))))

(define build_join_scan_with_mapper_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_work_rows)
	(begin
		(define tree (physical_join_plan_for_sources sources))
		(define probe_context (join_scan_probe_context tree all_sources probe_work_rows))
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
						(lower_column_expr_for_join_truth_context
							all_sources default_alias remaining_condition
							(probe_work_context_default_rows probe_context))
						final_row_expr
						(join_scan_skip_expr result_mode))))))
		(if (nil? tree)
			(terminal residual_condition row_expr order_items)
			(build_join_tree_scan_using_recipe
				schema all_sources tree '() default_alias needed_exprs residual_condition row_expr
				order_items offset_value limit_value allow_membership_recset column_recipe stages result_mode probe_context
				terminal false)))))

(define build_join_scan_pipeline_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier probe_work_rows column_recipe stages)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages
		(list (quote pipeline)) probe_work_rows)))

(define build_join_scan_reduce_using_recipe (lambda (schema all_sources sources default_alias needed_exprs final_condition row_expr order_items offset_value limit_value allow_membership_carrier probe_work_rows column_recipe stages reduce_expr neutral_expr shard_reduce_expr)
	(build_join_scan_with_mapper_using_recipe
		schema all_sources sources default_alias needed_exprs final_condition row_expr
		order_items offset_value limit_value allow_membership_carrier column_recipe stages
		(list (quote reduce) reduce_expr neutral_expr shard_reduce_expr) probe_work_rows)))

(define build_join_scan_sink (lambda (schema sources default_alias needed_exprs final_condition sink_expr stages)
	(build_join_scan_pipeline_using_recipe
		schema sources sources default_alias needed_exprs final_condition sink_expr
		'() 0 -1 false nil nil stages)))

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

(define join_order_intermediate_has_union_membership? (lambda (sources final_condition)
	(reduce sources (lambda (found src)
		(if found
			true
			(begin
				(define membership (driver_membership_for_source src
					(combine_where (source_join_expr src) final_condition)))
				(and (not (nil? membership))
					(union_block? (gs_input (nth membership 0))))))) false)))

(define join_order_intermediate_reduce_scan (lambda (table_expr value_names order_names order_items offset_value limit_value map_expr reduce_expr neutral_expr)
	(list (quote scan_order)
		'(session "__memcp_tx")
		table_expr
		(quoted_runtime_list '())
		(list (quote lambda) '() true)
		(cons (quote list) order_names)
		(cons (quote list) (order_relations_default order_items))
		0
		(coalesceNil offset_value 0)
		(coalesceNil limit_value -1)
		(cons (quote list) value_names)
		map_expr
		reduce_expr neutral_expr false)))

(define lower_join_order_to_intermediate (lambda (block value_exprs scan_sources scan_plan default_alias needed_exprs final_condition order_items stage_catalog probe_work_rows scan_builder)
	(begin
		(define order_values (order_exprs order_items))
		(define value_names (join_order_intermediate_names "v" (count value_exprs)))
		(define order_names (join_order_intermediate_names "o" (count order_values)))
		(define column_names (merge (list value_names order_names)))
		(define lowered_values (map (merge (list value_exprs order_values)) (lambda (expr)
			(lower_column_expr_for_join_in_context
				scan_sources default_alias expr probe_work_rows))))
		(define table_key (concat "__join_order_intermediate:" (uuid)))
		(define table_name (list (quote session) table_key))
		(define table_expr (list (quote table) (qb_schema block) table_name))
		/* Keep a logically implied membership as the scan source while filling
		the storage carrier. Other joins retain their established base-table scan
		path; enabling RecSet carriers globally regresses scalar ORDER pipelines. */
		(define allow_membership_carrier
			(join_order_intermediate_has_union_membership? scan_sources final_condition))
		(define fill_plan (build_join_scan_pipeline_using_recipe
			(qb_schema block) scan_sources scan_plan default_alias needed_exprs final_condition
			(join_order_intermediate_insert table_expr column_names lowered_values)
			'() 0 -1 allow_membership_carrier probe_work_rows nil stage_catalog))
		(define scan_plan_expr (scan_builder table_expr value_names order_names))
		(define drop_plan (list (quote droptable) (qb_schema block) table_name true))
		(list (quote !begin)
			(list (quote session) table_key (list (quote concat) ".join-order:" (list (quote uuid))))
			(list (quote createtable) (qb_schema block) table_name
				(join_order_intermediate_columns column_names)
				(quoted_runtime_list '("engine" "memory")) true)
			(list
				(list (quote lambda) (list (quote __intermediate_result))
					(list (quote !begin) drop_plan (quote __intermediate_result)))
				(list (quote try)
					(list (quote lambda) '() (list (quote !begin) fill_plan scan_plan_expr))
					(list (quote lambda) (list (quote __intermediate_error))
						(list (quote !begin) drop_plan (list (quote error) (quote __intermediate_error))))))))))

(define lower_join_order_through_intermediate (lambda (block fields scan_sources scan_plan default_alias needed_exprs final_condition order_items stage_catalog)
	(begin
		(define value_exprs (extract_assoc fields (lambda (_title expr) expr)))
		(lower_join_order_to_intermediate
			block value_exprs scan_sources scan_plan default_alias needed_exprs final_condition order_items stage_catalog nil
			(lambda (table_expr value_names order_names)
				(join_order_intermediate_reduce_scan
					table_expr value_names order_names order_items (qb_offset block) (qb_limit block)
					(list (quote lambda) (map value_names symbol)
						(list (quote resultrow)
							(cons (quote list) (join_order_intermediate_result_fields fields value_names))))
					nil nil))))))

/* ------------------------------------------------------------------------- */
/* Canonical physical prejoin relations                                      */

(define prejoin_primary_key_columns (lambda (src)
	(source_primary_key_columns src)))

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
				'(session "__memcp_tx")
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

(define membership_probe_work_rows (lambda (facts condition fallback)
	(if (expr_contains_driver_membership? condition)
		(coalesceNil (qassoc_get facts (quote membership_driver_rows) nil) fallback)
		fallback)))

(define build_join_scan_rows (lambda (schema sources plan default_alias needed_exprs final_condition fields order_items offset_value limit_value stages facts)
	(begin
		(define driver_source (join_optimizer_source_by_alias sources
			(join_optimizer_tree_first_alias plan)))
		(define filter_probe_work_rows (membership_probe_work_rows facts final_condition
			(planner_row_count_after_selectivity
				driver_source sources default_alias final_condition nil)))
		(define projection_probe_work_rows (if (query_limit_active? offset_value limit_value)
			(coalesceNil (probe_limit_work_rows limit_value) 0)
			(if (probe_context_unique_point? sources default_alias final_condition)
				1
				filter_probe_work_rows)))
		(define row_expr (list (quote resultrow)
			(cons (quote list) (lower_join_result_fields
				sources default_alias fields projection_probe_work_rows))))
		(build_join_scan_pipeline_using_recipe
			schema sources plan default_alias needed_exprs final_condition row_expr
			order_items offset_value limit_value true filter_probe_work_rows nil stages))))

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
		(define field_exprs (extract_assoc fields (lambda (_title expr) expr)))
		(define needed_exprs (merge (list
			field_exprs
			(list final_condition)
			(order_exprs order_items)
			(source_join_exprs scan_sources))))
		/* Dataset fields are mapped only after the native window accepts a row.
		Propagate that bound so nested scalar probes can compare direct work with
		the cost of preparing their complete dependent carriers. An unlimited
		reduce (e.g. a bare COUNT(*)) has no LIMIT window and no unique point
		lookup to bound it, but its call count is still knowable: the driving
		source's own row count, scaled by the residual condition's selectivity. */
		(define unbounded_probe_work_rows (membership_probe_work_rows (qb_facts block) final_condition
			(planner_row_count_after_selectivity
				driver_source scan_sources first_alias final_condition nil)))
		(define projection_probe_work_rows (if (query_limit_active? (qb_offset block) (qb_limit block))
			(coalesceNil (probe_limit_work_rows (qb_limit block)) 0)
			(if (probe_context_unique_point? scan_sources first_alias final_condition) 1
				unbounded_probe_work_rows)))
		(if direct_order_safe
			(begin
				(define row_expr (cons row_mapper (map field_exprs (lambda (expr)
					(lower_column_expr_for_join_in_context
						scan_sources first_alias expr projection_probe_work_rows)))))
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
					true projection_probe_work_rows nil (query_block_stage_catalog block)
					reduce_expr neutral_expr shard_reduce_expr))
			(lower_join_order_to_intermediate
				block field_exprs scan_sources scan_plan first_alias needed_exprs final_condition order_items
				(query_block_stage_catalog block) projection_probe_work_rows
				(lambda (table_expr value_names order_names)
					(join_order_intermediate_reduce_scan
						table_expr value_names order_names order_items (qb_offset block) (qb_limit block)
						(list (quote lambda) (map value_names symbol)
							(cons row_mapper (map value_names symbol)))
						reduce_expr neutral_expr)))))))

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
						final_condition fields order_items (qb_offset block) (qb_limit block)
						stage_catalog (qb_facts block))
					(if hierarchical_order
						(if (ordered_join_native_limit_supported?
							scan_sources scan_plan first_alias order_items stage_catalog final_condition)
							(join_ordered_native_limit_plan
								(qb_schema block) scan_sources scan_plan first_alias needed_exprs
								final_condition fields order_items (qb_offset block) (qb_limit block) stage_catalog
								(qb_facts block))
							(lower_join_order_through_intermediate
								block fields scan_sources scan_plan first_alias needed_exprs
								final_condition order_items stage_catalog))
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
			true nil nil (qb_stages block)
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
			(lower_column_expr_for_alias src cond)))
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
				(cons (quote list) (if (empty_list? order_items) '() (order_relations_for_source src order_items)))
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

(define union_order_relations (lambda (order_items)
	/* UNION branches share one merge relation. Until the logical UNION model
	carries result-column collation metadata, use the canonical binary factory
	relation rather than letting individual storage scans infer incompatible
	per-branch callbacks. */
	(order_relations_default order_items)))

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
		(define memberships (driver_memberships_for_source src condition))
		(define membership_bindings (membership_recset_bindings src memberships))
		(if (not (equal? (count memberships) (count membership_bindings)))
			(neumann_fail "build_queryplan" "streamed UNION membership requires a RecSet carrier")
			true)
		(define bound_memberships (map membership_bindings (lambda (binding) (nth binding 0))))
		(define filter_condition (replace_driver_membership_markers src condition bound_memberships))
		(define order_exprs (map order_positions (lambda (pos) (nth exprs pos))))
		(define filtercols (merge_unique (list
			(if (empty_list? membership_bindings) '() (list "$recset_contains"))
			(extract_columns_for_alias src filter_condition))))
		(define outputcols (merge_unique (map exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define sortcols (map order_exprs (lambda (expr) (union_sort_column_for_alias src expr))))
		(define sort_input_cols (merge_unique (map order_exprs (lambda (expr) (extract_columns_for_alias src expr)))))
		(define mapcols (merge_unique (list outputcols sort_input_cols)))
		(define filter_expr (list (quote lambda)
			(map filtercols (lambda (col) (scan_callback_symbol_for_alias alias col)))
			(lower_column_expr_for_alias src filter_condition)))
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
			map_expr
			membership_bindings))))

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
				(define membership_bindings (merge_unique (map specs (lambda (spec) (nth spec 6)))))
				(define scan_plan (list (quote scan_order_multi)
					'(session "__memcp_tx")
					(cons (quote list) (map specs (lambda (spec) (nth spec 0))))
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 1)))))
					(cons (quote list) (map specs (lambda (spec) (nth spec 2))))
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 3)))))
					(cons (quote list) (union_order_relations (union_order block)))
					nil
					nil
					0
					(coalesceNil (union_offset block) 0)
					(coalesceNil (union_limit block) -1)
					(cons (quote list) (map specs (lambda (spec) (cons (quote list) (nth spec 4)))))
					(cons (quote list) (map specs (lambda (spec) (nth spec 5))))))
				(define bound_scan_plan (wrap_membership_recset_bindings membership_bindings scan_plan))
				(if (empty_list? prepares)
					bound_scan_plan
					(cons (quote begin) (merge (list prepares (list bound_scan_plan))))))
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
		(define stages (if (query_block? planned_root)
			(stage_catalog_with_nested (query_block_stage_catalog planned_root))
			'()))
		(make_ir
			(ir_kind ir)
			(if (query_block? planned_root)
				(query_block_with_full_stage_catalog_using planned_root stages)
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

(define prepare_binding_key (lambda (expr)
	(match expr
		'(((symbol context) "session") key ((symbol once) ((symbol lambda) _params true))) nil
		'(((quote context) "session") key ((quote once) ((quote lambda) _params true))) nil
		'(((symbol context) "session") key ((symbol once) _initializer)) key
		'(((quote context) "session") key ((quote once) _initializer)) key
		_ nil)))

(define emitted_prepare_binding_keys (lambda (expr keys)
	(begin
		(define key (prepare_binding_key expr))
		(if (not (nil? key))
			(set_assoc keys key true)
			(match expr
				(cons head tail) (reduce tail (lambda (found item)
					(emitted_prepare_binding_keys item found))
					(emitted_prepare_binding_keys head keys))
				_ keys)))))

(define without_prepare_bindings (lambda (expr keys)
	(begin
		(define key (prepare_binding_key expr))
		(if (and (not (nil? key)) (has_assoc? keys key))
			true
			(match expr
				(cons head tail) (cons
					(without_prepare_bindings head keys)
					(map tail (lambda (item) (without_prepare_bindings item keys))))
				_ expr)))))

(define physical_string_set (lambda (expr strings)
	(if (string? expr)
		(set_assoc strings expr true)
		(match expr
			(cons head tail) (reduce tail (lambda (found item)
				(physical_string_set item found))
				(physical_string_set head strings))
			_ strings))))

(define stage_aggregate_referenced? (lambda (strings stage)
	(reduce (gs_aggregates stage) (lambda (found aggregate)
		(or found (has_assoc? strings (aggregate_col_name_using (gs_input stage) aggregate))))
		false)))

(define closed_group_prepare_stage? (lambda (dependency_graph stage)
	(and (group_stage? stage)
		(and (source_is_base_table? (gs_input stage))
			(and (not (stage_has_residual_outer_refs? stage))
				(reduce (stage_dependency_closure_using_graph dependency_graph stage)
					(lambda (closed dependency)
						(and closed
							(and (group_stage? dependency)
								(not (stage_has_residual_outer_refs? dependency)))))
					true))))))

(define shared_prepare_owner_key (lambda (stage)
	(concat "__prepare_carrier_" (fnv_hash (stage_prepare_backbone_signature stage)))))

(define shared_prepare_owner_binding (lambda (dependency_graph catalog stage)
	(list
		(list (quote context) "session")
		(shared_prepare_owner_key stage)
		(list (quote once)
			(list (quote lambda) '()
				(list (quote !begin)
					(lower_stage_prepare_using
						(stage_dependency_closure_using_graph dependency_graph stage)
						catalog stage)
					true))))))

(define shared_prepare_alias_binding (lambda (stage)
	(list
		(list (quote context) "session")
		(stage_prepare_key stage)
		(list (quote once)
			(list (quote lambda) '()
				(list (quote apply)
					(list (list (quote context) "session") (shared_prepare_owner_key stage))
					(quoted_runtime_list '())))))))

/* Recipe emission is a two-step physical pass: normal lowering records which
lazy stage keys are actually reachable, then this collector emits one closed
initializer owner per canonical carrier and replaces local copies with aliases.
Both AST walks are linear; no pairwise recipe comparison is performed. */
(define consolidate_closed_group_prepares (lambda (ir plan)
	(if (not (query_block? (ir_root ir)))
		plan
		(begin
			(define catalog (query_block_stage_catalog (ir_root ir)))
			(if (empty_list? catalog)
				plan
				(begin
					(define emitted_keys (emitted_prepare_binding_keys plan '()))
					(define emitted_strings (physical_string_set plan '()))
					(define dependency_graph (stage_dependency_graph catalog))
					(define consumers (filter catalog (lambda (stage)
						(and (closed_group_prepare_stage? dependency_graph stage)
							(has_assoc? emitted_keys (stage_prepare_key stage))))))
					(define selected_backbones (stage_prepare_backbone_set consumers))
					/* A consumer can read an aggregate column without owning a prepare
					binding. Once a carrier is reachable, collect every compatible column
					requirement for it from the canonical catalog. */
					(define selected (filter catalog (lambda (stage)
						(and (closed_group_prepare_stage? dependency_graph stage)
							(and (has_assoc? selected_backbones (stage_prepare_backbone_signature stage))
								(or (has_assoc? emitted_keys (stage_prepare_key stage))
									(stage_aggregate_referenced? emitted_strings stage)))))))
					(if (empty_list? selected)
						plan
						(begin
							(define carriers (collect_stage_prepares selected))
							(define selected_keys (reduce selected (lambda (keys stage)
								(set_assoc keys (stage_prepare_key stage) true)) '()))
							(cons (quote !begin)
								(merge (list
									(map carriers (lambda (stage)
										(shared_prepare_owner_binding dependency_graph catalog stage)))
									(map selected shared_prepare_alias_binding)
									(list (without_prepare_bindings plan selected_keys)))))))))))))

(define physical_plan_uses_query_scope? (lambda (expr)
	(if (or (equal? expr (physical_query_session_symbol))
		(or (equal? expr (physical_query_scope_symbol))
			(equal? expr (physical_query_tx_symbol))))
		true
		(match expr
			((symbol quote) _value) false
			((quote quote) _value) false
			(cons head tail) (or (physical_plan_uses_query_scope? head)
				(reduce tail (lambda (found item)
					(or found (physical_plan_uses_query_scope? item))) false))
			_ false))))

/* `session` is optimized to its helper closure while planner code is loaded,
whereas dynamically-built fragments can retain the symbol. Normalize both
representations without depending on their printed form. */
(define physical_session_call_parts (lambda (expr)
	(match expr
		((symbol quote) _value) nil
		((quote quote) _value) nil
		(cons head tail) (if (or (equal? head session) (equal? head (quote session)))
			(list tail)
			nil)
		_ nil)))

/* Physical operators share one transaction for the complete query. Keep
ordinary session reads intact: the scan optimizer recognizes their lack of
column dependencies and can lift invariant CASE dispatch out of row callbacks.
Replacing them with lexical symbols here would hide that dependency fact. */
(define rewrite_physical_transaction_reads (lambda (expr)
	(begin
		(define parts (physical_session_call_parts expr))
		(if (not (nil? parts))
			(begin
				(define args (car parts))
				(if (and (equal? (count args) 1) (equal? (car args) "__memcp_tx"))
					(physical_query_tx_symbol)
					expr))
			(match expr
				((symbol quote) _value) expr
				((quote quote) _value) expr
				(cons head tail) (cons
					(rewrite_physical_transaction_reads head)
					(map tail (lambda (item)
						(rewrite_physical_transaction_reads item))))
				_ expr)))))

(define query_invariant_presence_memo_parts (lambda (expr)
	(match expr
		'(session_symbol "get_or_compute_scoped" scope_symbol key producer)
		(if (and (equal? session_symbol (physical_query_session_symbol))
			(and (equal? scope_symbol (physical_query_scope_symbol))
				(and (string? key) (strlike key "__query_presence_probe_%"))))
			(list key producer)
			nil)
		_ nil)))

(define collect_query_invariant_presence_memos (lambda (expr entries)
	(begin
		(define parts (query_invariant_presence_memo_parts expr))
		(define found (if (nil? parts) entries (set_assoc entries (nth parts 0) expr)))
		(match expr
			((symbol quote) _value) found
			((quote quote) _value) found
			(cons head tail) (reduce tail (lambda (acc item)
				(collect_query_invariant_presence_memos item acc))
				(collect_query_invariant_presence_memos head found))
			_ found))))

(define query_invariant_presence_value_symbol (lambda (key)
	(symbol (concat "__query_presence_value_" (fnv_hash key)))))

(define rewrite_query_invariant_presence_memos (lambda (expr)
	(begin
		(define parts (query_invariant_presence_memo_parts expr))
		(if (not (nil? parts))
			(query_invariant_presence_value_symbol (nth parts 0))
			(match expr
				((symbol quote) _value) expr
				((quote quote) _value) expr
				(cons head tail) (cons
					(rewrite_query_invariant_presence_memos head)
					(map tail rewrite_query_invariant_presence_memos))
				_ expr)))))

/* Presence probes have bounded boolean semantics, so evaluating their
query-invariant producers at the physical boundary cannot introduce scalar
cardinality errors. Share one lexical value across every lowered consumer;
this also prevents optimizer inlining from moving the producer back into a
row callback. */
(define consolidate_query_invariant_presence_memos (lambda (plan)
	(begin
		(define entries (collect_query_invariant_presence_memos plan '()))
		(if (empty_list? entries)
			plan
			(cons (quote !begin) (merge (list
				(extract_assoc entries (lambda (key expr)
					(list (quote define)
						(query_invariant_presence_value_symbol key)
						expr)))
				(list (rewrite_query_invariant_presence_memos plan)))))))))

/* Resolve request-local state once at the physical-plan boundary. RecSet
lookups execute inside deeply nested row callbacks; resolving `(context
"session")` there would repeat the goroutine-local lookup for every row even
though both handles are constant for the complete query generation. */
(define with_physical_query_context (lambda (plan)
	(begin
		(define rewritten (rewrite_physical_transaction_reads plan))
		(if (not (physical_plan_uses_query_scope? rewritten))
			rewritten
			(cons (quote !begin) (merge (list
				(list (list (quote define) (physical_query_session_symbol)
					(list (quote context) "session")))
				(list (list (quote define) (physical_query_scope_symbol)
					(list (quote context) "query")))
				(list (list (quote define) (physical_query_tx_symbol)
					(list (physical_query_session_symbol) "__memcp_tx")))
				(list rewritten))))))))

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
		(define consolidated_plan (consolidate_closed_group_prepares ir plan))
		(define memoized_plan (if (empty_list? (ir_stages ir))
			consolidated_plan
			(consolidate_query_invariant_presence_memos consolidated_plan)))
		(require_physical_scan_relations
			(with_physical_query_context memoized_plan)))))

(define build_queryplan (lambda (ir)
	(emit_physical_queryplan (prepare_physical_queryplan ir))))

/* Keep phase ownership explicit. normalize_query_ast inside untangle_query
removes parser-specific spelling before dependent joins are identified; this
wrapper owns the output-shape normalizers which must precede binding.
Decorrelation owns D and all subquery elimination; only then may logical join
ordering run. Storage artifacts begin in build_queryplan. */
(define normalize_sql_syntax (lambda (ast)
	(sanitize_temporal_outputs (sanitize_decimal_aggregate_outputs ast))))

(define decorrelate_logical_query (lambda (ast)
	(untangle_query_term (normalize_sql_syntax ast) nil)))

(define optimize_logical_query (lambda (ir)
	(join_reorder (aggregate_pushdown_logical ir))))

(define neumann_compile_pipeline (lambda (ast)
	(build_queryplan
		(optimize_logical_query
			(decorrelate_logical_query ast)))))

(define neumann_compile_ir_pipeline (lambda (ir)
	(build_queryplan
		(optimize_logical_query
			(require_flat_stage_dependencies "compile_ir" (normalize_stage_dependencies ir))))))

/* ------------------------------------------------------------------------- */
/* Parser-facing adapters                                                     */

(define build_queryplan_term (lambda (query)
	(neumann_compile_pipeline query)))

(define build_queryplan_term_with_sink (lambda (query sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with_return (decorrelate_logical_query query) sink_mode))))

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
			(ir_with_return (decorrelate_logical_query query) (list (quote dml) schema tbl))))))

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
		(define ir (decorrelate_logical_query query))
		(list (quote resultrow)
			(list (quote list)
				"ir"
				(concat
					(pretty_print ir (settings "ExplainWidth"))
					(explain_union_ir_metadata ir)))))))

(define explain_queryplan_reorder (lambda (query)
	(begin
		(define planning_session (context "session"))
		(planning_session "__memcp_explain_reorder_selectivities" true)
		(define reordered (optimize_logical_query (decorrelate_logical_query query)))
		(planning_session "__memcp_explain_reorder_selectivities" nil)
		(list (quote resultrow)
			(list (quote list)
				"reorder"
				(pretty_print reordered (settings "ExplainWidth")))))))

(define physical_recset_source_expr? (lambda (expr)
	(match expr
		(cons head tail) (or
			(equal? (string head) "scan_recset")
			(equal? (string head) "filter_recset")
			(equal? (string head) "recset_project_join")
			(equal? (string head) "recset_union")
			(equal? (string head) "recset_intersect")
			(equal? (string head) "recset_difference")
			(equal? (string head) "recset_not")
			(reduce tail (lambda (found item) (or found (physical_recset_source_expr? item))) false))
		_ false)))

(define physical_recset_contains_expr? (lambda (expr)
	(match expr
		(cons head tail) (or
			(physical_recset_contains_expr? head)
			(reduce tail (lambda (found item) (or found (physical_recset_contains_expr? item))) false))
		_ (or
			(equal? (string expr) "$recset_contains")
			(equal? (string expr) "__recset_contains")
			(equal? (string expr) "recset_contains")))))

(define physical_ordered_recset_decision (lambda (scan_expr)
	(begin
		(define table_expr (nth scan_expr 2))
		(define recset_source (physical_recset_source_expr? table_expr))
		(define contains_probe (or
			(physical_recset_contains_expr? (nth scan_expr 3))
			(physical_recset_contains_expr? (nth scan_expr 4))))
		(if (not (or recset_source contains_probe))
			nil
			(begin
				(define chosen (if recset_source
					(quote scan_order_recset_part)
					(quote recset_contains_probe)))
				(list
					(list "decision" "ordered_recset_iterator")
					(list "chosen" (string chosen))
					(list "reason" (if recset_source
						"recset_scan_source"
						"base_table_scan_with_membership_filter"))
					(list "inputs" (list
						(list "source" (pretty_print table_expr (settings "ExplainWidth")))
						(list "sort_columns" (pretty_print (nth scan_expr 5) (settings "ExplainWidth")))
						(list "offset" (nth scan_expr 8))
						(list "limit" (nth scan_expr 9))))
					(list "alternatives" (list
						(list
							(list "plan" "scan_order_recset_part")
							(list "status" (if recset_source "chosen" "rejected"))
							(list "reason" (if recset_source "recset_scan_source" "source_is_base_table")))
						(list
							(list "plan" "recset_contains_probe")
							(list "status" (if recset_source "rejected" "chosen"))
							(list "reason" (if recset_source "direct_intersection_available" "membership_is_filter")))))))))))

(define physical_ordered_recset_decisions (lambda (expr)
	(match expr
		(cons head tail) (begin
			(define own (if (and
				(equal? head (quote scan_order))
				(> (count expr) 9))
				(begin
					(define decision (physical_ordered_recset_decision expr))
					(if (nil? decision) '() (list decision)))
				'()))
			(reduce tail (lambda (decisions item)
				(merge (list decisions (physical_ordered_recset_decisions item)))) own))
		_ '())))

(define physical_head_matches? (lambda (head target)
	(or (equal? head target)
		(try
			(lambda () (equal? head (eval target)))
			(lambda (_e) false)))))

(define physical_expr_has_head? (lambda (expr target)
	(match expr
		(cons head tail) (or
			(physical_head_matches? head target)
			(reduce tail (lambda (found item)
				(or found (physical_expr_has_head? item target))) false))
		_ false)))

(define physical_membership_operator_family (lambda (plan)
	(if (physical_expr_has_head? plan (quote recset_project_join))
		"candidate_keyset"
		/* A forced membership variant reaches this function only after
		require_physical_scan_relations has rejected every surviving logical
		marker. Without a projected RecSet its emitted carrier is therefore the
		driver-side probe family, independent of the concrete group-cache/index
		primitive selected inside that family. */
		"driver_order_membership_probe")))

/* recset_project_join is adaptive only inside the physical operator: actual
source-key and per-shard target cardinalities are available there, while the
logical plan often has only a weak selectivity sample for text predicates.
Expose that bounded cost decision rather than presenting the carrier as an
opaque implementation detail in EXPLAIN PHYSICAL. */
(define physical_recset_project_join_decisions (lambda (expr)
	(match expr
		(cons head tail) (begin
			(define own (if (equal? (string head) "recset_project_join")
				(list (list
					(list "decision" "recset_project_join_access")
					(list "chosen" "runtime_cost_minimum")
					(list "reason" "actual_key_and_target_shard_cardinality")
					(list "alternatives" (list
						"indexed_key_probes"
						"dense_numeric_membership_scan"
						"dense_generic_membership_scan"))))
				'()))
			(reduce tail (lambda (decisions item)
				(merge (list decisions (physical_recset_project_join_decisions item)))) own))
		_ '())))

(define compile_physical_explain_variant (lambda (reordered overrides)
	(begin
		(define planning_session (context "session"))
		(define accumulator (newsession))
		(accumulator "count" 0)
		(planning_session "__memcp_explain_physical" accumulator)
		(planning_session "__memcp_physical_overrides" overrides)
		(define prepared (prepare_physical_queryplan reordered))
		(define plan (emit_physical_queryplan prepared))
		(define operator_family (physical_membership_operator_family plan))
		(define optimized_plan (optimize plan))
		(define decisions (merge (list
			(planner_physical_explain_decisions accumulator)
			(physical_ordered_recset_decisions optimized_plan)
			(physical_recset_project_join_decisions optimized_plan))))
		(planning_session "__memcp_explain_physical" nil)
		(planning_session "__memcp_physical_overrides" nil)
		(list optimized_plan decisions operator_family))))

(define physical_decision_by_id (lambda (decisions decision_id)
	(reduce decisions (lambda (found decision)
		(if (not (nil? found))
			found
			(if (equal? (qassoc_get decision "decision_id" nil) decision_id)
				decision
				nil))) nil)))

(define physical_decision_alternative (lambda (decision plan_name)
	(reduce (qassoc_get decision "alternatives" '()) (lambda (found alternative)
		(if (not (nil? found))
			found
			(if (equal? (qassoc_get alternative "plan" nil) plan_name)
				alternative
				nil))) nil)))

(define physical_calibration_input (lambda (decision name)
	(qassoc_get (qassoc_get decision "inputs" '()) name nil)))

/* Execute one forced plan while shadowing resultrow with a bounded digest
sink. The outer protocol callback receives only the calibration row, never the
potentially large calibrated SELECT result. */
(define physical_calibration_runtime_plan (lambda (plan decision variant estimated_ns operator_family suite_var)
	(begin
		(define decision_id (qassoc_get decision "decision_id" "unknown"))
		(define expected_family (if (equal? (qassoc_get decision "decision" nil) "membership_carrier")
			variant
			operator_family))
		(define consistent (or (not (equal? (qassoc_get decision "decision" nil) "membership_carrier"))
			(equal? operator_family expected_family)))
		(define baseline_hash_key (concat "hash:" decision_id))
		(define baseline_count_key (concat "count:" decision_id))
		(define shared_suite (equal? suite_var (quote __physical_calibration_suite)))
		(list
			(list (quote lambda) (list (quote __calibration_emit))
				(list (quote !begin)
					(list (quote define) (quote __calibration_capture) (list (quote newsession)))
					(list (quote __calibration_capture) "count" 0)
					(list (quote __calibration_capture) "hash" "")
					(list (quote define) (quote resultrow)
						(list (quote lambda) (list (quote __calibration_row))
							(list (quote !begin)
								(list (quote __calibration_capture) "count"
									(list (quote +) (list (quote __calibration_capture) "count") 1))
								(list (quote __calibration_capture) "hash"
									(list (quote sha256) (list (quote concat)
										(list (quote __calibration_capture) "hash")
										(list (quote serialize) (quote __calibration_row)))))
								nil)))
					(list (quote define) (quote __calibration_started_ns) (list (quote nanotime)))
					plan
					(list (quote define) (quote __calibration_whole_query_execution_ns)
						(list (quote -) (list (quote nanotime)) (quote __calibration_started_ns)))
					(list (quote define) (quote __calibration_rows) (list (quote __calibration_capture) "count"))
					(list (quote define) (quote __calibration_hash) (list (quote __calibration_capture) "hash"))
					(list (quote define) (quote __calibration_first_hash) (list suite_var baseline_hash_key))
					(list (quote define) (quote __calibration_equal)
						(list (quote if) (list (quote nil?) (quote __calibration_first_hash))
							(list (quote !begin)
								(list suite_var baseline_hash_key (quote __calibration_hash))
								(list suite_var baseline_count_key (quote __calibration_rows))
								true)
							(list (quote and)
								(list (quote equal?) (quote __calibration_first_hash) (quote __calibration_hash))
								(list (quote equal?) (list suite_var baseline_count_key) (quote __calibration_rows)))))
					(list (quote __calibration_emit)
						(list (quote list)
							"decision_id" decision_id
							"decision" (qassoc_get decision "decision" "unknown")
							"consumer" (qassoc_get decision "consumer" "unknown")
							"plan" variant
							"operator_family" operator_family
							"operator_consistent" consistent
							"estimated_ns" estimated_ns
							"whole_query_execution_ns" (quote __calibration_whole_query_execution_ns)
							"operator_ns" nil
							"measurement_scope" (if shared_suite "shared_sequential_diagnostic" "isolated_variant_request")
							"fit_eligible" (not shared_suite)
							"candidate_input_rows" (physical_calibration_input decision "candidate_input_rows")
							"candidate_rows" (physical_calibration_input decision "candidate_rows")
							"candidate_density" (physical_calibration_input decision "candidate_density")
							"estimate_population" (physical_calibration_input decision "estimate_population")
							"estimate_coverage" (physical_calibration_input decision "estimate_coverage")
							"projected_driver_rows" (physical_calibration_input decision "projected_driver_rows")
							"driver_input_rows" (physical_calibration_input decision "driver_input_rows")
							"driver_rows" (physical_calibration_input decision "driver_rows")
							"expected_driver_rows_visited" (physical_calibration_input decision "expected_driver_rows_visited")
							"limit" (physical_calibration_input decision "limit")
							"offset" (physical_calibration_input decision "offset")
							"probe_branches" (physical_calibration_input decision "probe_branches")
							"candidate_scan_invocations" (physical_calibration_input decision "candidate_scan_invocations")
							"candidate_filter_columns" (physical_calibration_input decision "candidate_filter_columns")
							"candidate_map_columns" (physical_calibration_input decision "candidate_map_columns")
							"candidate_cache_map_columns" (physical_calibration_input decision "candidate_cache_map_columns")
							"candidate_expression_operations" (physical_calibration_input decision "candidate_expression_operations")
							"candidate_expression_depth" (physical_calibration_input decision "candidate_expression_depth")
							"candidate_broad_text_match_rows" (physical_calibration_input decision "candidate_broad_text_match_rows")
							"candidate_broad_text_match_bytes" (physical_calibration_input decision "candidate_broad_text_match_bytes")
							"candidate_filter_value_rows" (physical_calibration_input decision "candidate_filter_value_rows")
							"candidate_expression_operation_rows" (physical_calibration_input decision "candidate_expression_operation_rows")
							"driver_scan_invocations" (physical_calibration_input decision "driver_scan_invocations")
							"driver_filter_columns" (physical_calibration_input decision "driver_filter_columns")
							"driver_map_columns" (physical_calibration_input decision "driver_map_columns")
							"driver_expression_operations" (physical_calibration_input decision "driver_expression_operations")
							"driver_expression_depth" (physical_calibration_input decision "driver_expression_depth")
							"rows" (quote __calibration_rows)
							"result_hash" (quote __calibration_hash)
							"result_equal" (quote __calibration_equal)))))
			(quote resultrow)))))

(define physical_calibration_variants_for_decision (lambda (reordered decision suite_var)
	(begin
		(define decision_id (qassoc_get decision "decision_id" nil))
		(map (qassoc_get decision "alternatives" '()) (lambda (alternative)
			(begin
				(define variant (qassoc_get alternative "plan" nil))
				(define compilation (compile_physical_explain_variant reordered
					(list (list decision_id variant))))
				(define variant_decision (physical_decision_by_id (nth compilation 1) decision_id))
				(define variant_alternative (physical_decision_alternative variant_decision variant))
				(define variant_cost (qassoc_get variant_alternative "cost" '()))
				(physical_calibration_runtime_plan
					(nth compilation 0)
					variant_decision
					variant
					(qassoc_get variant_cost "total_ns" nil)
					(nth compilation 2)
					suite_var)))))))

(define physical_decision_has_costed_alternatives? (lambda (decision)
	(begin
		(define alternatives (qassoc_get decision "alternatives" '()))
		(and (not (nil? (qassoc_get decision "decision_id" nil)))
			(and (> (count alternatives) 1)
				(reduce alternatives (lambda (valid alternative)
					(and valid (number? (qassoc_get
						(qassoc_get alternative "cost" '()) "total_ns" nil)))) true))))))

(define physical_decision_calibratable? (lambda (decision)
	(if (not (physical_decision_has_costed_alternatives? decision))
		false
		(if (not (equal? (qassoc_get decision "decision" nil) "membership_carrier"))
			true
			(begin
				(define inputs (qassoc_get decision "inputs" '()))
				(and (number? (qassoc_get inputs "candidate_input_rows" nil))
					(and (number? (qassoc_get inputs "candidate_rows" nil))
						(and (number? (qassoc_get inputs "driver_input_rows" nil))
							(number? (qassoc_get inputs "driver_rows" nil))))))))))

(define explain_queryplan_physical_calibrate_discover (lambda (query)
	(begin
		(define reordered (optimize_logical_query (decorrelate_logical_query query)))
		(define compilation (compile_physical_explain_variant reordered nil))
		(define decisions (filter (nth compilation 1) (lambda (decision)
			(physical_decision_calibratable? decision))))
		(if (empty_list? decisions)
			(list (quote resultrow) (list (quote list)
				"error" "no calibratable physical decision"))
			(cons (quote !begin) (map decisions (lambda (decision)
				(list (quote resultrow) (list (quote list)
					"decision_id" (qassoc_get decision "decision_id" nil)
					"decision" (qassoc_get decision "decision" nil)
					"alternatives" (cons (quote list)
						(map (qassoc_get decision "alternatives" '()) (lambda (alternative)
							(qassoc_get alternative "plan" nil))))
					"estimated_ns" (cons (quote list)
						(map (qassoc_get decision "alternatives" '()) (lambda (alternative)
							(qassoc_get (qassoc_get alternative "cost" '()) "total_ns" nil)))))))))))))

(define explain_queryplan_physical_calibrate_variant (lambda (query decision_id variant)
	(begin
		(define reordered (optimize_logical_query (decorrelate_logical_query query)))
		(define compilation (compile_physical_explain_variant reordered
			(list (list decision_id variant))))
		(define decision (physical_decision_by_id (nth compilation 1) decision_id))
		(define alternative (if (nil? decision) nil
			(physical_decision_alternative decision variant)))
		(if (or (nil? decision) (nil? alternative))
			(list (quote resultrow) (list (quote list)
				"error" "unknown physical calibration decision or variant"))
			(begin
				(define suite_var (quote __physical_calibration_variant_suite))
				(list (quote !begin)
					(list (quote define) suite_var (list (quote newsession)))
					(physical_calibration_runtime_plan
						(nth compilation 0)
						decision
						variant
						(qassoc_get (qassoc_get alternative "cost" '()) "total_ns" nil)
						(nth compilation 2)
						suite_var)))))))

(define explain_queryplan_physical_calibrate (lambda (query)
	(begin
		(define reordered (optimize_logical_query (decorrelate_logical_query query)))
		(define default_compilation (compile_physical_explain_variant reordered nil))
		(define uncalibratable (filter (nth default_compilation 1) (lambda (decision)
			(and (equal? (qassoc_get decision "decision" nil) "membership_carrier")
				(not (physical_decision_calibratable? decision))))))
		(define decisions (filter (nth default_compilation 1) (lambda (decision)
			(physical_decision_calibratable? decision))))
		(if (not (empty_list? uncalibratable))
			(list (quote resultrow) (list (quote list)
				"error" "uncalibratable physical membership decision"
				"decision" (string (car uncalibratable))))
			(if (empty_list? decisions)
				(list (quote resultrow) (list (quote list)
					"error" "no calibratable physical decision"))
				(begin
					(define suite_var (quote __physical_calibration_suite))
					(define variants (merge (map decisions (lambda (decision)
						(physical_calibration_variants_for_decision reordered decision suite_var)))))
					(cons (quote !begin) (cons
						(list (quote define) suite_var (list (quote newsession)))
						variants))))))))

(define explain_queryplan_physical (lambda (query)
	(begin
		(define reordered (optimize_logical_query (decorrelate_logical_query query)))
		(define compilation (compile_physical_explain_variant reordered nil))
		(define optimized_plan (nth compilation 0))
		(define decisions (nth compilation 1))
		(list (quote resultrow)
			(list (quote list)
				"physical"
				(string decisions)
				"code"
				(pretty_print optimized_plan (settings "ExplainWidth")))))))

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
		(define ir (decorrelate_logical_query query))
		(define untangled_ns (nanotime))
		(define reordered (optimize_logical_query ir))
		(define reordered_ns (nanotime))
		(define prepared (prepare_physical_queryplan reordered))
		(define prepared_ns (nanotime))
		(define plan (emit_physical_queryplan prepared))
		(define emitted_ns (nanotime))
		/* Read every raw-plan metric before transferring plan ownership to optimize. */
		(define plan_text (pretty_print plan (settings "ExplainWidth")))
		(define raw_plan_nodes (tree_count plan))
		(define raw_scans (plan_count plan (quote scan)))
		(define raw_ordered_scans (+
			(plan_count plan (quote scan_order))
			(plan_count plan (quote scan_order_multi))))
		(define raw_exists_scans (plan_count plan (quote scan_exists)))
		(define raw_group_caches (plan_count plan (quote touch_keytable)))
		(define serialized_ns (nanotime))
		(define optimizer_telemetry (newsession))
		(define optimized_plan (optimize plan (lambda (stats)
			(optimizer_telemetry "stats" stats))))
		(define optimized_ns (nanotime))
		(define optimizer_stats (optimizer_telemetry "stats"))
		(list (quote resultrow)
			(list (quote list)
				"parse_ns" (- started_ns parse_started_ns)
				"untangle_ns" (- untangled_ns started_ns)
				"reorder_ns" (- reordered_ns untangled_ns)
				"physical_prepare_ns" (- prepared_ns reordered_ns)
				"recipe_emit_ns" (- emitted_ns prepared_ns)
				"lower_ns" (- emitted_ns reordered_ns)
				"optimizer_ns" (optimizer_stats "compile_ns")
				"optimizer_wall_ns" (- optimized_ns serialized_ns)
				"serialize_ns" (- serialized_ns emitted_ns)
				"planner_total_ns" (- emitted_ns started_ns)
				"compile_total_ns" (- optimized_ns parse_started_ns)
				"measured_total_ns" (- optimized_ns parse_started_ns)
				"sql_bytes" sql_bytes
				"ast_nodes" (tree_count query)
				"logical_nodes" (tree_count reordered)
				"plan_nodes" raw_plan_nodes
				"plan_bytes" (strlen plan_text)
				"optimizer_input_nodes" (optimizer_stats "input_nodes")
				"optimizer_output_nodes" (tree_count optimized_plan)
				"optimizer_rewrites" (optimizer_stats "rewrites")
				"optimizer_rejected_rewrites" (optimizer_stats "rejected_rewrites")
				"optimizer_budget_remaining" (optimizer_stats "budget_remaining")
				"optimizer_callback_analyses" (optimizer_stats "callback_analyses")
				"optimizer_callback_clones" (optimizer_stats "callback_clones")
				"query_blocks" (logical_count (ir_root reordered) (quote query-block))
				"group_stages" (logical_count (ir_root reordered) (quote group-stage))
				"union_blocks" (logical_count (ir_root reordered) (quote union-block))
				"scans" raw_scans
				"ordered_scans" raw_ordered_scans
				"exists_scans" raw_exists_scans
				"group_caches" raw_group_caches
				/* Backward-compatible telemetry alias; use group_caches in new integrations. */
				"group_carriers" raw_group_caches)))))
