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

/* Reorder/optimise scaffold                                                  */

/* Join reordering can consume a logical hypergraph view of the query block. The
extractor deliberately does not rewrite sources or move predicates: WHERE and
ON terms remain owned by the query-block until physical lowering. */
(define join_hypergraph_alias_index (lambda (aliases)
	(reduce (mapIndex (coalesceNil aliases '()) (lambda (position alias)
		(list alias position))) (lambda (index entry)
			(set_assoc index (toLower (car entry)) entry)) '())))

(define join_hypergraph_expr_aliases_using (lambda (default_alias alias_index expr)
	(map (sort (filter
		(extract_assoc (query_expr_alias_set default_alias expr '()) (lambda (alias _present)
			(get_assoc alias_index (toLower alias) nil)))
		(lambda (entry) (not (nil? entry))))
		(lambda (left right) (< (cadr left) (cadr right)))) car)))

(define join_hypergraph_expr_aliases (lambda (default_alias aliases expr)
	(join_hypergraph_expr_aliases_using default_alias
		(join_hypergraph_alias_index aliases) expr)))

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

(define join_hypergraph_where_predicates (lambda (block default_alias alias_index)
	(if (equal? (coalesceNil (qb_where block) true) true)
		'()
		(map (split_and_terms (qb_where block)) (lambda (predicate)
			(make_join_hypergraph_predicate
				(join_hypergraph_expr_aliases_using default_alias alias_index predicate)
				(quote where)
				nil
				predicate))))))

(define join_hypergraph_source_predicates (lambda (sources default_alias alias_index)
	(merge (map (coalesceNil sources '()) (lambda (src)
		(begin
			(define join_expr (coalesceNil (source_join_expr src) true))
			(if (equal? join_expr true)
				'()
				(map (split_and_terms join_expr) (lambda (predicate)
					(make_join_hypergraph_predicate
						(merge_unique (list
							(join_hypergraph_expr_aliases_using default_alias alias_index predicate)
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

(define join_hypergraph_outer_barriers_acc (lambda (stages relation_units all_sources sources preceding_aliases default_alias alias_index)
	(match (coalesceNil sources '())
		(cons src rest) (begin
			(define alias (source_alias src))
			(define next_preceding (cons alias preceding_aliases))
			(define remaining (join_hypergraph_outer_barriers_acc
				stages relation_units all_sources rest next_preceding default_alias alias_index))
			(if (source_outer? src)
				(cons
					(list
						(list (quote kind) (quote left-outer))
						(list (quote owner) alias)
						(list (quote preserved)
							(join_optimizer_outer_requirements
								stages relation_units all_sources default_alias alias_index src))
						(list (quote references)
							(join_hypergraph_expr_aliases_using default_alias alias_index
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
		(define alias_index (join_hypergraph_alias_index aliases))
		(define default_alias (if (empty_list? aliases) nil (car aliases)))
		(define predicates (merge (list
			(join_hypergraph_where_predicates block default_alias alias_index)
			(join_hypergraph_source_predicates sources default_alias alias_index))))
		(list
			(list (quote nodes) (join_hypergraph_nodes sources))
			(list (quote locals) (join_hypergraph_predicates_of_kind predicates (quote local)))
			(list (quote edges) (join_hypergraph_predicates_of_kind predicates (quote edge)))
			(list (quote hyperedges) (join_hypergraph_predicates_of_kind predicates (quote hyperedge)))
			(list (quote residuals) (join_hypergraph_predicates_of_kind predicates (quote residual)))
			(list (quote barriers)
				(join_hypergraph_outer_barriers_acc
					(qb_stages block)
					(qassoc_get (qb_facts block) (quote join_relation_units) '())
					sources sources '() default_alias alias_index))))))

(define join_optimizer_source_stage (lambda (stages src)
	(coalesceNil
		(if (stage_output_relation? (source_relation src))
			(stage_for_output_relation stages (source_relation src)) nil)
		(stage_for_group_cache_source stages src))))

(define join_optimizer_at_most_one_unbound_source? (lambda (stages src)
	(begin
		(define stage (join_optimizer_source_stage stages src))
		(and (group_stage? stage)
			(and (equal? (stage_result_max_rows_per_partition stage) 1)
				(begin
					(define partition_by (qassoc_get (gs_facts stage) (quote partition_by) '()))
					(or (empty_list? partition_by) (equal? partition_by '(1)))))))))

(define join_optimizer_guaranteed_singleton_source? (lambda (stages src)
	(and (join_optimizer_at_most_one_unbound_source? stages src)
		(begin
			(define stage (join_optimizer_source_stage stages src))
			(define partition_by (qassoc_get (gs_facts stage) (quote partition_by) '()))
			(and (equal? (qassoc_get (gs_facts stage) (quote preserve_empty_domain) false) true)
				(or (equal? partition_by '(1))
					(not (stage_has_residual_outer_refs? stage))))))))

(define join_optimizer_inner_source? (lambda (stages src)
	(or (not (source_outer? src))
		(join_optimizer_guaranteed_singleton_source? stages src))))

(define join_optimizer_normalize_inner_joins (lambda (stages block)
	(begin
		(define inner_join_terms (merge (map (qb_sources block) (lambda (src)
			(if (and
				(join_optimizer_inner_source? stages src)
				(not (or (nil? (source_join_expr src)) (equal? (source_join_expr src) true))))
				(split_and_terms (source_join_expr src))
				'())))))
		(make_query_block
			(qb_schema block)
			(map (qb_sources block) (lambda (src)
				(if (join_optimizer_inner_source? stages src) (source_with_join_expr src nil) src)))
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
		((symbol strlike) _value pattern)
		(begin
			(define prior (text_pattern_selectivity_prior
				(planner_string_expr_value pattern)))
			(if (number? prior)
				(planner_estimate prior 0.25 (quote text_pattern_prior) true)
				(planner_unknown_selectivity_estimate)))
		((symbol strlike) _value pattern _collation)
		(begin
			(define prior (text_pattern_selectivity_prior
				(planner_string_expr_value pattern)))
			(if (number? prior)
				(planner_estimate prior 0.25 (quote text_pattern_prior) true)
				(planner_unknown_selectivity_estimate)))
		((quote strlike) value pattern)
		(join_optimizer_expr_selectivity_estimate sources default_alias
			(list (symbol "strlike") value pattern))
		((quote strlike) value pattern collation)
		(join_optimizer_expr_selectivity_estimate sources default_alias
			(list (symbol "strlike") value pattern collation))
		_ (planner_unknown_selectivity_estimate))))

(define join_optimizer_expr_selectivity (lambda (sources default_alias expr)
	(begin
		(define estimate (join_optimizer_expr_selectivity_estimate sources default_alias expr))
		(planner_estimate_planning_value estimate
			(if (equal? (qassoc_get estimate (quote source) nil) (quote range_unknown))
				0.3333333333333333 0.1)))))

(define join_optimizer_product (lambda (values)
	(reduce (coalesceNil values '()) (lambda (product value) (* product value)) 1)))

(define join_optimizer_source_rows_from_base (lambda (base_rows_value sources default_alias local_predicates src)
	(begin
		(define base_rows (coalesceNil base_rows_value 1000000))
		(define local_selectivity (join_optimizer_product
			(map (filter local_predicates
				(lambda (entry)
					/* A WHERE predicate on the nullable side observes the row after
					null extension. Cost it at the owning LEFT node, never as a scan
					filter on the stage/base leaf. */
					(not (and (source_outer? src)
						(equal? (qassoc_get entry (quote origin) nil) (quote where))))))
				(lambda (entry)
					(join_optimizer_expr_selectivity sources default_alias
						(qassoc_get entry (quote predicate) true))))))
		(max 1 (* base_rows local_selectivity)))))

(define join_optimizer_source_rows (lambda (stages sources default_alias graph src)
	(join_optimizer_source_rows_from_base
		(planner_estimate_planning_value
			(planner_source_row_estimate_using_stages stages src) 1000000)
		sources default_alias
		(join_optimizer_local_predicates graph (source_alias src)) src)))

(define planner_quoted_value (lambda (value)
	(list (quote quote) value)))

(define join_optimizer_source_rows_expr (lambda (stages sources default_alias graph src)
	(begin
		(define local_predicates
			(join_optimizer_local_predicates graph (source_alias src)))
		(define local_aliases (merge_unique (list
			(list (source_alias src))
			(map local_predicates (lambda (entry)
				(qassoc_get entry (quote aliases) '()))))))
		(define local_sources (filter sources (lambda (source)
			(contains? local_aliases (source_alias source)))))
		(define source_rows_expr (if (stage_output_relation? (source_relation src))
			(begin
				(define stage (stage_by_id stages
					(stage_output_relation_id (source_relation src))))
				(define singleton (and (group_stage? stage)
					(and (equal? (stage_result_max_rows_per_partition stage) 1)
						(and (empty_list? (qassoc_get (gs_facts stage) (quote partition_by) '()))
							(not (stage_has_residual_outer_refs? stage))))))
				(if singleton 1
					(if (group_stage? stage)
						(list (quote planner_stage_input_rows)
							(planner_quoted_value (gs_input stage))) nil)))
			(list (quote planner_source_row_count) (planner_quoted_value src))))
		(planner_guard_runtime_binding
			(list (quote join_optimizer_source_rows_from_base)
				source_rows_expr
				(planner_quoted_value local_sources)
				default_alias
				(planner_quoted_value local_predicates)
				(planner_quoted_value src))))))

(define join_optimizer_selectivity_expr (lambda (sources default_alias predicate)
	(begin
		(define aliases (join_hypergraph_expr_aliases
			default_alias (source_aliases sources) predicate))
		(define local_sources (filter sources (lambda (src)
			(contains? aliases (source_alias src)))))
		(planner_guard_runtime_binding
			(list (quote join_optimizer_expr_selectivity)
				(planner_quoted_value local_sources)
				default_alias
				(planner_quoted_value predicate))))))

(define join_optimizer_alias_subset? (lambda (required available)
	(reduce (coalesceNil required '()) (lambda (ok alias)
		(and ok (contains? available alias))) true)))

(define join_optimizer_preceding_aliases (lambda (sources alias)
	(match (coalesceNil sources '())
		(cons src rest) (if (equal? (source_alias src) alias)
			'()
			(cons (source_alias src) (join_optimizer_preceding_aliases rest alias)))
		_ '())))

(define join_optimizer_relation_unit (lambda (relation_units parent)
	(find (coalesceNil relation_units '()) (lambda (unit)
		(equal? (qassoc_get unit (quote parent) nil) parent)) nil)))

(define join_optimizer_relation_children (lambda (relation_units parent)
	(begin
		(define unit (join_optimizer_relation_unit relation_units parent))
		(if (nil? unit) '() (qassoc_get unit (quote children) '())))))

(define join_optimizer_relation_parent (lambda (relation_units child)
	(reduce (coalesceNil relation_units '()) (lambda (found unit)
		(if (not (nil? found)) found
			(if (contains? (qassoc_get unit (quote children) '()) child)
				(qassoc_get unit (quote parent) nil)
				nil))) nil)))

(define join_optimizer_last_external_predecessor (lambda (relation_units sources src_alias)
	(begin
		(define children (join_optimizer_relation_children relation_units src_alias))
		(define preceding (filter (join_optimizer_preceding_aliases sources src_alias)
			(lambda (alias) (not (contains? children alias)))))
		(if (empty_list? preceding) nil (nth preceding (- (count preceding) 1))))))

/* Every LEFT JOIN has the same NULL-extension semantics. Dependencies come
from its ON expression; children introduced while rewriting that ON remain
inside the nullable relation unit and therefore are not preserved inputs of
their parent. An ON TRUE join still needs one preserved-side anchor, for which
the nearest external predecessor is sufficient; unrelated inner joins may be
cost-reordered around the complete relation unit. */
(define join_optimizer_outer_requirements (lambda (stages relation_units sources default_alias alias_index src)
	(if (join_optimizer_inner_source? stages src)
		'()
		(begin
			(define alias (source_alias src))
			(define children (join_optimizer_relation_children relation_units alias))
			(define references (filter
				(join_hypergraph_expr_aliases_using default_alias alias_index
					(coalesceNil (source_join_expr src) true))
				(lambda (required_alias) (and
					(not (equal? required_alias alias))
					(not (contains? children required_alias))))))
			(define parent (join_optimizer_relation_parent relation_units alias))
			(define required (merge_unique (list references (if (nil? parent) '() (list parent)))))
			/* Finding the nearest fallback anchor walks the source prefix. Most
			LEFT arms already name their preserved input, so do not pay that
			quadratic prefix cost merely to discard the answer. */
			(if (not (empty_list? required))
				required
				(begin
					(define anchor (join_optimizer_last_external_predecessor
						relation_units sources alias))
					(if (nil? anchor) required (list anchor)))))))))

(define join_optimizer_metadata_nodes (lambda (stages relation_units sources default_alias alias_index graph fixed_cardinality)
	(map sources (lambda (src)
		(begin
			(define stage (join_optimizer_source_stage stages src))
			(define singleton (join_optimizer_guaranteed_singleton_source? stages src))
			(list
				(source_alias src)
				(if (or singleton fixed_cardinality) 1
					(join_optimizer_source_rows stages sources default_alias graph src))
				(if (or singleton fixed_cardinality) 1
					(join_optimizer_source_rows_expr stages sources default_alias graph src))
				(if (join_optimizer_inner_source? stages src) (quote inner) (quote left-outer))
				(join_optimizer_outer_requirements stages relation_units sources default_alias alias_index src)
				(if (group_stage? stage) (stage_result_max_rows_per_partition stage) nil)
				singleton))))))

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

/* Cost confidence defines a one-sided uncertainty interval for fuzzy physical
choices. A candidate is a clear winner only when its confidence-adjusted upper
estimate remains below the other alternative's point estimate. Callers with an
exact bounded fallback may prefer it while those intervals overlap. */
(define planner_cost_clear_winner? (lambda (candidate current)
	(begin
		(define candidate_confidence (max 0.01
			(qassoc_get candidate (quote confidence) 0.01)))
		(< (/ (qassoc_get candidate (quote total_ns) 0) candidate_confidence)
			(qassoc_get current (quote total_ns) 0)))))

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

/* The final three cost fields mirror the numeric values as cheap runtime ASTs.
They let the compile-local condition accumulator retain the exact inequalities
which selected a cached join plan without rerunning join enumeration. */
(define join_order_node_kind (lambda (node)
	(if (> (count node) 3) (nth node 3) (quote inner))))
(define join_order_node_requirements (lambda (node)
	(if (> (count node) 4) (nth node 4) '())))
(define join_order_node_max_rows_per_driver (lambda (node)
	(if (> (count node) 5) (nth node 5) nil)))
(define join_order_node_guaranteed_singleton? (lambda (node)
	(if (> (count node) 6) (nth node 6) false)))

/* A nullable leaf is pending until a plan containing all aliases of its D/left
input attaches it on the right. Composite plans have consumed the requirement.
plan = (tree aliases cardinality cost size atomic driver-cardinality left right cardinality-expr cost-expr driver-expr cost-domain pending-kind pending-requirements) */
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
(define join_order_plan_pending_kind (lambda (plan)
	(if (> (count plan) 13) (nth plan 13) (quote inner))))
(define join_order_plan_pending_requirements (lambda (plan)
	(if (> (count plan) 14) (nth plan 14) '())))
(define join_order_plan_pending_outer? (lambda (plan)
	(equal? (join_order_plan_pending_kind plan) (quote left-outer))))

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
			(planner_scan_cost rows 0.5)
			(join_order_node_kind node)
			(join_order_node_requirements node)))))

(define join_order_product_expr (lambda (items)
	(match items
		(cons item '()) item
		(cons _head _tail) (cons (quote *) items)
		_ 1)))

(define join_order_cap_cardinality (lambda (value)
	(max 1 (min 1e300 value))))

(define join_order_join_shape (lambda (left right)
	/* This is the NULL-extension rule for every LEFT JOIN. A pending nullable
	relation may be attached at any costed point above its complete preserved
	input, but it must remain on the right. Emitting an ordinary inner join would
	erase unmatched rows.

	The recursive case is general: `root LEFT (node LEFT child)` consumes the
	child boundary while deliberately propagating node's still-pending outer
	requirement. Attaching that composite relation to root performs the second
	NULL extension. No intermediate relation is materialized. */
	(if (join_order_plan_pending_outer? left)
		(if (and (join_order_plan_pending_outer? right)
			(join_order_set_subset?
				(join_order_plan_pending_requirements right)
				(join_order_plan_aliases left)))
			(list (quote left-outer)
				(join_order_plan_pending_kind left)
				(join_order_plan_pending_requirements left))
			nil)
		(if (join_order_plan_pending_outer? right)
			(if (join_order_set_subset?
				(join_order_plan_pending_requirements right)
				(join_order_plan_aliases left))
				(list (quote left-outer) (quote inner) '())
				nil)
			(list (quote inner) (quote inner) '())))))

(define join_order_outer_post_predicate? (lambda (predicate kind right)
	(and (equal? kind (quote left-outer))
		(equal? (join_order_pred_barrier_owner predicate)
			(car (join_order_plan_aliases right))))))

(define join_order_join_cardinality (lambda (predicates kind left right)
	(begin
		(define combined (merge_unique (list
			(join_order_plan_aliases left) (join_order_plan_aliases right))))
		(define join_selectivity (reduce predicates (lambda (value predicate)
			(if (join_order_predicate_crosses_in? predicate
				(join_order_plan_aliases left) (join_order_plan_aliases right) combined)
				(* value (join_order_pred_selectivity predicate))
				value)) 1))
		(define joined (*
			(join_order_plan_cardinality left)
			(join_order_plan_cardinality right)
			join_selectivity))
		(define extended (if (equal? kind (quote left-outer))
			(max (join_order_plan_cardinality left) joined)
			joined))
		(define post_selectivity (reduce predicates (lambda (value predicate)
			(if (join_order_outer_post_predicate? predicate kind right)
				(* value (join_order_pred_selectivity predicate))
				value)) 1))
		(join_order_cap_cardinality (* extended post_selectivity)))))

(define join_order_join_cardinality_expr (lambda (predicates kind left right)
	(begin
		(define combined (merge_unique (list
			(join_order_plan_aliases left) (join_order_plan_aliases right))))
		(define join_selectivities (map (filter predicates (lambda (predicate)
			(join_order_predicate_crosses_in? predicate
				(join_order_plan_aliases left) (join_order_plan_aliases right) combined)))
			join_order_pred_selectivity_expr))
		(define joined_expr (list (quote *)
			(join_order_plan_cardinality_expr left)
			(join_order_plan_cardinality_expr right)
			(join_order_product_expr join_selectivities)))
		(define extended_expr (if (equal? kind (quote left-outer))
			(list (quote max) (join_order_plan_cardinality_expr left) joined_expr)
			joined_expr))
		(define post_selectivities (map (filter predicates (lambda (predicate)
			(join_order_outer_post_predicate? predicate kind right)))
			join_order_pred_selectivity_expr))
		(list (quote join_order_cap_cardinality)
			(list (quote *) extended_expr (join_order_product_expr post_selectivities))))))

(define join_order_join_plan (lambda (universe predicates left right)
	(if (or (nil? left) (nil? right))
		nil
		(begin
			(define shape (join_order_join_shape left right))
			(if (nil? shape)
				nil
				(begin
					(define kind (nth shape 0))
					(define cardinality (join_order_join_cardinality predicates kind left right))
					(define cardinality_expr (join_order_join_cardinality_expr predicates kind left right))
					(define combined (join_order_set_union universe
						(join_order_plan_aliases left) (join_order_plan_aliases right)))
					(define children_cost (planner_cost_add
						(join_order_plan_cost_domain left) (join_order_plan_cost_domain right)
						cardinality 0.5))
					(define cost_domain (planner_cost_add children_cost
						(planner_join_work_cost cardinality 0.5) cardinality 0.5))
					(list
						(list (quote join-node) kind (join_order_plan_tree left) (join_order_plan_tree right) '())
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
						cost_domain
						(nth shape 1)
						(nth shape 2))))))))

(define join_order_candidate_better? (lambda (current candidate)
	(or (< (join_order_plan_cost candidate) (join_order_plan_cost current))
		(and (equal? (join_order_plan_cost candidate) (join_order_plan_cost current))
			(< (join_order_driver_cardinality candidate) (join_order_driver_cardinality current))))))

(define join_order_plan_driver_alias (lambda (plan)
	(if (nil? plan) nil (join_optimizer_tree_first_alias (join_order_plan_tree plan)))))

(define join_order_required_drivers (lambda (required_property)
	(filter (coalesceNil required_property '()) string?)))

(define join_order_required_aliases (lambda (required_property)
	(qassoc_get (coalesceNil required_property '()) (quote ordered_aliases) '())))

(define join_order_alias_subsequence? (lambda (required actual)
	(match required
		(cons expected rest) (if (empty_list? actual)
			false
			(if (equal? expected (car actual))
				(join_order_alias_subsequence? rest (cdr actual))
				false))
		_ true)))

(define join_order_plan_satisfies_driver_property? (lambda (plan required_drivers)
	(begin
		(define drivers (join_order_required_drivers required_drivers))
		(define ordered_aliases (join_order_required_aliases required_drivers))
		(define relevant_order (filter ordered_aliases (lambda (alias)
			(contains? (join_order_plan_aliases plan) alias))))
		(define contains_order_driver (and (not (empty_list? ordered_aliases))
			(contains? (join_order_plan_aliases plan) (car ordered_aliases))))
		(and
			(or (empty_list? drivers)
				(not (join_order_set_intersects? (join_order_plan_aliases plan) drivers))
				(contains? drivers (join_order_plan_driver_alias plan)))
			(or (not contains_order_driver)
				(equal? (join_order_plan_driver_alias plan) (car ordered_aliases)))
			(or (empty_list? relevant_order)
				(join_order_alias_subsequence? relevant_order
					(join_optimizer_tree_aliases (join_order_plan_tree plan))))))))

(define join_order_better_plan (lambda (current candidate required_drivers)
	(if (nil? candidate)
		current
		(if (nil? current)
			candidate
			(begin
				(define current_valid (join_order_plan_satisfies_driver_property? current required_drivers))
				(define candidate_valid (join_order_plan_satisfies_driver_property? candidate required_drivers))
				(if (and candidate_valid (not current_valid))
					candidate
					(if (and current_valid (not candidate_valid))
						current
						(if (join_order_candidate_better? current candidate)
							candidate current))))))))

(define join_order_best_orientation (lambda (universe predicates left right required_drivers)
	(join_order_better_plan
		(join_order_join_plan universe predicates left right)
		(join_order_join_plan universe predicates right left)
		required_drivers)))


(define join_order_hypergraph? (lambda (predicates)
	(reduce predicates (lambda (found predicate)
		(or found (> (count (join_order_pred_aliases predicate)) 2))) false)))

(define join_order_has_outer_barriers? (lambda (nodes)
	(reduce nodes (lambda (found node)
		(or found (equal? (join_order_node_kind node) (quote left-outer)))) false)))

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

(define join_order_dphyp_set_plan (lambda (universe predicates connected_by_first plans aliases required_drivers)
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
						(join_order_best_orientation universe predicates left right required_drivers)
						required_drivers)
					best))
			best)) nil)))

(define join_order_dphyp_fill (lambda (nodes universe predicates connected_by_first remaining plans entries required_drivers)
	(if (empty_list? remaining)
		(list (get_assoc plans (join_order_set_key universe) nil) entries)
		(begin
			(define aliases (car remaining))
			(define plan (if (single_source? aliases)
				(join_order_leaf_plan (join_order_find_node nodes (car aliases)))
				(join_order_dphyp_set_plan universe predicates connected_by_first plans aliases required_drivers)))
			(join_order_dphyp_fill nodes universe predicates connected_by_first (cdr remaining)
				(if (nil? plan) plans (set_assoc plans (join_order_set_key aliases) plan))
				(if (nil? plan) entries (+ entries 1)) required_drivers)))))

(define join_order_dphyp_connected (lambda (nodes aliases predicates connected required_drivers)
	(begin
		(define sorted_connected (join_order_sort_sets connected))
		(define connected_by_first (group_assoc connected car
			(lambda (subsets subset) (append subsets subset)) '()))
		(join_order_dphyp_fill nodes aliases predicates connected_by_first sorted_connected '() 0 required_drivers))))

(define join_order_dphyp_budgeted (lambda (nodes aliases predicates budget required_drivers)
	(begin
		(define connected (join_order_enumerate_connected aliases predicates budget))
		(if (cadr connected)
			(list nil 0)
			(join_order_dphyp_connected nodes aliases predicates (car connected) required_drivers))))))

/* A fused join reducer needs a left-deep logical tree, but that requirement is
not permission for the physical lowerer to reorder a bushy winner. Keep a
second DP state family for this physical property and compare its calibrated
whole-pipeline cost with the unrestricted logical winner. */
(define join_order_pipeline_width (lambda (required_property)
	(qassoc_get (coalesceNil required_property '()) (quote pipeline_reduce_width) nil)))

(define join_order_pipeline_probe_rows (lambda (plan)
	(if (nil? plan)
		nil
		(match (join_order_plan_tree plan)
			((symbol join-leaf) _alias _predicates) 0
			((quote join-leaf) _alias _predicates) 0
			((symbol join-leaf) _alias) 0
			((quote join-leaf) _alias) 0
			((symbol join-node) kind _left right _predicates)
			(if (and (equal? kind (quote inner))
				(single_source? (join_optimizer_tree_aliases right)))
				(begin
					(define left (join_order_plan_left plan))
					(define prior (join_order_pipeline_probe_rows left))
					(if (number? prior)
						(+ prior (join_order_plan_cardinality left)) nil))
				nil)
			((quote join-node) kind _left right _predicates)
			(if (and (equal? kind (quote inner))
				(single_source? (join_optimizer_tree_aliases right)))
				(begin
					(define left (join_order_plan_left plan))
					(define prior (join_order_pipeline_probe_rows left))
					(if (number? prior)
						(+ prior (join_order_plan_cardinality left)) nil))
				nil)
			_ nil))))

(define join_order_pipeline_cost (lambda (nodes plan width)
	(begin
		(define probe_rows (join_order_pipeline_probe_rows plan))
		(if (or (nil? plan) (not (number? probe_rows)))
			nil
			(begin
				(define input_rows (reduce (join_order_plan_aliases plan)
					(lambda (rows alias)
						(+ rows (cadr (join_order_find_node nodes alias)))) 0))
				(planner_scan_join_order_cost input_rows probe_rows
					(join_order_plan_cardinality plan)
					(count (join_order_plan_aliases plan))
					(join_order_plan_cardinality plan) width))))))

(define join_order_pipeline_plan_better? (lambda (nodes width candidate current)
	(if (nil? current)
		true
		(planner_cost_better?
			(join_order_pipeline_cost nodes candidate width)
			(join_order_pipeline_cost nodes current width)))))

(define join_order_pipeline_set_plan (lambda (nodes universe predicates plans aliases width)
	(reduce aliases (lambda (best right_alias)
		(begin
			(define left_aliases (join_order_set_difference universe aliases (list right_alias)))
			(define left (get_assoc plans (join_order_set_key left_aliases) nil))
			(define right (get_assoc plans (join_order_set_key (list right_alias)) nil))
			(define candidate (if (and (not (nil? left))
				(and (not (nil? right))
					(join_order_connected? predicates left_aliases (list right_alias))))
				(join_order_join_plan universe predicates left right) nil))
			(if (and (not (nil? candidate))
				(join_order_pipeline_plan_better? nodes width candidate best))
				candidate best))) nil)))

(define join_order_pipeline_fill (lambda (nodes universe predicates remaining plans entries width)
	(if (empty_list? remaining)
		(list (get_assoc plans (join_order_set_key universe) nil) entries)
		(begin
			(define aliases (car remaining))
			(define plan (if (single_source? aliases)
				(join_order_leaf_plan (join_order_find_node nodes (car aliases)))
				(join_order_pipeline_set_plan nodes universe predicates plans aliases width)))
			(join_order_pipeline_fill nodes universe predicates (cdr remaining)
				(if (nil? plan) plans (set_assoc plans (join_order_set_key aliases) plan))
				(if (nil? plan) entries (+ entries 1)) width)))))

(define join_order_pipeline_candidate (lambda (nodes aliases predicates connected required_property)
	(begin
		(define width (join_order_pipeline_width required_property))
		(if (not (number? width))
			(list nil 0)
			(join_order_pipeline_fill nodes aliases predicates
				(join_order_sort_sets connected) '() 0 width)))))

(define join_order_plan_with_cost (lambda (plan cost)
	(list
		(join_order_plan_tree plan)
		(join_order_plan_aliases plan)
		(join_order_plan_cardinality plan)
		(qassoc_get cost (quote total_ns) 0)
		(join_order_plan_size plan)
		(join_order_plan_atomic? plan)
		(join_order_driver_cardinality plan)
		(join_order_plan_left plan)
		(join_order_plan_right plan)
		(join_order_plan_cardinality_expr plan)
		(qassoc_get cost (quote total_ns) 0)
		(join_order_plan_driver_expr plan)
		cost
		(join_order_plan_pending_kind plan)
		(join_order_plan_pending_requirements plan))))

(define join_order_alias_position (lambda (aliases alias)
	(reduce (produceN (count aliases)) (lambda (found i)
		(if (not (nil? found)) found
			(if (equal? (nth aliases i) alias) i nil))) nil)))

(define join_order_alias_position_index (lambda (aliases)
	(reduce (mapIndex aliases (lambda (position alias) (list alias position)))
		(lambda (index entry) (set_assoc index (car entry) (cadr entry))) '())))

(define join_order_regular_edges (lambda (aliases predicates)
	(begin
		(define positions (join_order_alias_position_index aliases))
		(define edge_dict (reduce predicates (lambda (dict predicate)
			(begin
				(define refs (join_order_pred_aliases predicate))
				(if (not (equal? (count refs) 2))
					dict
					(begin
						(define left (car refs))
						(define right (cadr refs))
						(define ordered (if (< (get_assoc positions left)
							(get_assoc positions right))
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

(define join_order_linearized_splits (lambda (universe predicates table start end split best required_drivers)
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
						(join_order_join_plan universe predicates left right) required_drivers)
					best) required_drivers))))))

(define join_order_linearized_starts (lambda (universe predicates table length start entries required_drivers)
	(if (> (+ start length) (count universe))
		(list table entries)
		(begin
			(define end (- (+ start length) 1))
			(define best (join_order_linearized_splits universe predicates table start end start nil required_drivers))
			(join_order_linearized_starts universe predicates
				(if (nil? best) table (set_assoc table (join_order_interval_key start end) best))
				length (+ start 1) (if (nil? best) entries (+ entries 1)) required_drivers)))))

(define join_order_linearized_lengths (lambda (universe predicates table length entries required_drivers)
	(if (> length (count universe))
		(list (get_assoc table (join_order_interval_key 0 (- (count universe) 1)) nil) entries)
		(begin
			(define filled (join_order_linearized_starts universe predicates table length 0 entries required_drivers))
			(join_order_linearized_lengths universe predicates (car filled) (+ length 1) (cadr filled) required_drivers))))))

(define join_order_linearized_dp (lambda (nodes aliases predicates required_drivers)
	(begin
		(define cost_order (join_order_ikkbz_order nodes aliases predicates))
		/* Linearized DP cannot recover a driver that IKKBZ placed inside the
		chain. Prefix the cheapest order-capable alias before interval planning;
		the remaining chain still follows IKKBZ rank. */
		(define required_driver (find cost_order (lambda (alias)
			(contains? (join_order_required_drivers required_drivers) alias)) nil))
		(define order (if (nil? required_driver) cost_order
			(cons required_driver (filter cost_order (lambda (alias)
				(not (equal? alias required_driver)))))))
		(define table (reduce (produceN (count order)) (lambda (dict i)
			(set_assoc dict (join_order_interval_key i i)
				(join_order_leaf_plan (join_order_find_node nodes (nth order i))))) '()))
		(join_order_linearized_lengths aliases predicates
			table 2 (count order) required_drivers))))

(define join_order_goo_better_pair (lambda (best left_index right_index candidate)
	(if (nil? candidate)
		best
		(if (or (nil? best)
			(< (join_order_plan_cardinality candidate)
				(join_order_plan_cardinality (nth best 2))))
			(list left_index right_index candidate)
			best))))

(define join_order_goo_best_right (lambda (universe predicates required_drivers
	require_connection left_index left right_index remaining best)
	(match remaining
		(cons right rest) (begin
			(define connected (or (not require_connection)
				(join_order_connected? predicates
					(join_order_plan_aliases left) (join_order_plan_aliases right))))
			(define candidate (if connected
				(join_order_best_orientation universe predicates left right required_drivers)
				nil))
			(join_order_goo_best_right universe predicates required_drivers
				require_connection left_index left (+ right_index 1) rest
				(join_order_goo_better_pair best left_index right_index candidate)))
		_ best)))

/* GOO used to materialize an n*n list of pair records and complete candidate
plans, filter it twice, and then rebuild the winning join once more. Keep only
the current best candidate while scanning pairs and reuse its plan in the
merge step. This retains functional semantics while bounding temporary live
memory independently of the number of pairs. */
(define join_order_goo_best_pair_scan (lambda (universe predicates required_drivers
	require_connection left_index remaining best)
	(match remaining
		(cons left rest) (begin
			(define next_best (join_order_goo_best_right
				universe predicates required_drivers require_connection
				left_index left (+ left_index 1) rest best))
			(join_order_goo_best_pair_scan universe predicates required_drivers
				require_connection (+ left_index 1) rest next_best))
		_ best)))

(define join_order_goo_indexes_acc (lambda (remaining position alias_index plan_index)
	(match remaining
		(cons plan rest) (begin
			(define next_alias_index (reduce (join_order_plan_aliases plan)
				(lambda (index alias) (set_assoc index alias position)) alias_index))
			(join_order_goo_indexes_acc rest (+ position 1) next_alias_index
				(set_assoc plan_index position plan)))
		_ (list alias_index plan_index))))

(define join_order_goo_indexes (lambda (plans)
	(join_order_goo_indexes_acc plans 0 '() '())))

(define join_order_goo_predicate_components (lambda (predicate alias_index)
	(reduce (join_order_pred_aliases predicate) (lambda (components alias)
		(begin
			(define component (get_assoc alias_index alias nil))
			(if (or (nil? component) (contains? components component))
				components
				(append components component)))) '())))

(define join_order_goo_connected_pairs (lambda (predicates alias_index)
	(extract_assoc
		(reduce predicates (lambda (pairs predicate)
			(begin
				(define components (join_order_goo_predicate_components predicate alias_index))
				(if (not (equal? (count components) 2))
					pairs
					(begin
						(define left (min (car components) (cadr components)))
						(define right (max (car components) (cadr components)))
						(set_assoc pairs (concat left ":" right) (list left right)))))) '())
		(lambda (_key pair) pair))))

(define join_order_goo_best_connected_pair (lambda (universe predicates required_drivers plans)
	(begin
		(define indexes (join_order_goo_indexes plans))
		(define alias_index (car indexes))
		(define plan_index (cadr indexes))
		(reduce (join_order_goo_connected_pairs predicates alias_index)
			(lambda (best pair)
				(begin
					(define left_index (car pair))
					(define right_index (cadr pair))
					(define candidate (join_order_best_orientation universe predicates
						(get_assoc plan_index left_index nil)
						(get_assoc plan_index right_index nil)
						required_drivers))
					(join_order_goo_better_pair best left_index right_index candidate))) nil))))

(define join_order_goo_best_pair (lambda (universe plans predicates required_drivers)
	(begin
		/* A join lookup over predicate endpoints avoids scanning every component
		pair and then rescanning every predicate merely to discover connectivity.
		The all-pairs pass remains only as the correctness fallback for an
		explicitly disconnected graph. */
		(define connected (join_order_goo_best_connected_pair
			universe predicates required_drivers plans))
		(if (nil? connected)
			(join_order_goo_best_pair_scan
				universe predicates required_drivers false 0 plans nil)
			connected))))

(define join_order_goo_loop (lambda (universe predicates plans required_drivers)
	(if (single_source? plans)
		(car plans)
		(begin
			(define pair (join_order_goo_best_pair universe plans predicates required_drivers))
			(define left_index (nth pair 0))
			(define right_index (nth pair 1))
			(define joined (nth pair 2))
			(join_order_goo_loop universe predicates
				(append (filter (mapIndex plans (lambda (i plan)
					(if (or (equal? i left_index) (equal? i right_index)) nil plan)))
					(lambda (plan) (not (nil? plan)))) joined) required_drivers))))))

(define join_order_goo (lambda (nodes aliases predicates required_drivers)
	(join_order_goo_loop aliases predicates
		(map aliases (lambda (alias)
			(join_order_leaf_plan (join_order_find_node nodes alias)))) required_drivers)))

/* Over-budget regular join graphs are rooted once and decomposed at every
branch. Each recursive result is a complete independent arm; siblings are
never considered as a join candidate because they have no predicate between
them. This is the relational counterpart of collapsing the paper's
independently optimizable subqueries into meta-nodes, without recognizing a
particular star shape. */
(define join_order_arm_adjacency (lambda (aliases regular_edges)
	(reduce regular_edges (lambda (adjacency edge)
		(begin
			(define left (nth edge 0))
			(define right (nth edge 1))
			(set_assoc
				(set_assoc adjacency left (append (get_assoc adjacency left '()) right))
				right (append (get_assoc adjacency right '()) left))))
		(reduce aliases (lambda (adjacency alias)
			(set_assoc adjacency alias '())) '()))))

(define join_order_arm_root (lambda (nodes aliases adjacency required_drivers)
	(begin
		(define ordered (join_order_required_aliases required_drivers))
		(define drivers (join_order_required_drivers required_drivers))
		(define required (if (empty_list? ordered) drivers ordered))
		/* A pending nullable relation cannot drive its preserved input. Root the
		dependency tree at an unconstrained inner relation; NULL-extending arms
		then remain directed parent-to-child without any shape-specific rule. */
		(define roots (filter aliases (lambda (alias)
			(begin
				(define node (join_order_find_node nodes alias))
				(and (equal? (join_order_node_kind node) (quote inner))
					(empty_list? (join_order_node_requirements node)))))))
		(define candidates (if (empty_list? roots) aliases roots))
		(define required_root (find candidates (lambda (alias)
			(contains? required alias)) nil))
		(if (not (nil? required_root))
			required_root
			(car (reduce candidates (lambda (best alias)
				(begin
					(define degree (count (get_assoc adjacency alias '())))
					(define rows (cadr (join_order_find_node nodes alias)))
					(if (or (nil? best)
						(> degree (cadr best))
						(and (equal? degree (cadr best)) (< rows (nth best 2))))
						(list alias degree rows) best))) nil))))))

(define join_order_arm_plan_from (lambda (nodes universe predicates adjacency alias visited required_drivers)
	(begin
		(define with_root (set_assoc visited alias true))
		(define children_state (reduce (get_assoc adjacency alias '()) (lambda (state child)
			(if (get_assoc (cadr state) child false)
				state
				(begin
					(define child_result (join_order_arm_plan_from
						nodes universe predicates adjacency child (cadr state) required_drivers))
					(list (append (car state) (car child_result)) (cadr child_result)))))
			(list '() with_root)))
		/* A child plan is already a complete arm. Sorting those meta-nodes once
		keeps the branch composition O(k log k), rather than reconsidering all
		pairs after every attachment. */
		(define children (sort (car children_state) (lambda (left right)
			(if (or (nil? left) (nil? right))
				false
				(< (join_order_plan_cardinality left) (join_order_plan_cardinality right))))))
		/* A nullable child may depend on aliases which live in another arm. Such a
		local subtree has no legal orientation yet. Propagate that fact to the
		adaptive caller, which already owns the general source/property completion,
		instead of passing a partial plan into the next cardinality comparison. */
		(define invalid_child (reduce children (lambda (invalid child)
			(or invalid (nil? child))) false))
		(if invalid_child
			(list nil (cadr children_state))
			(begin
				(define plan (reduce children (lambda (parent child)
					/* Independent arms contribute one global choice: attach the arm above
					the parent or below it. LEFT dependencies naturally invalidate the
					reversed orientation in join_order_join_shape. */
					(if (nil? parent)
						nil
						(join_order_best_orientation universe predicates parent child required_drivers)))
					(join_order_leaf_plan (join_order_find_node nodes alias))))
				(list plan (cadr children_state)))))))

(define join_order_arm_plan (lambda (nodes aliases predicates regular_edges required_drivers)
	(begin
		(define adjacency (join_order_arm_adjacency aliases regular_edges))
		(define root (join_order_arm_root nodes aliases adjacency required_drivers))
		(car (join_order_arm_plan_from
			nodes aliases predicates adjacency root '() required_drivers)))))

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
		(join_order_plan_cost_domain plan)
		(join_order_plan_pending_kind plan)
		(join_order_plan_pending_requirements plan))))

(define join_order_subtree_has_regular_cycle? (lambda (plan regular_edges)
	(begin
		(define aliases (join_order_plan_aliases plan))
		(define internal_edges (filter regular_edges (lambda (edge)
			(and (contains? aliases (nth edge 0)) (contains? aliases (nth edge 1))))))
		(>= (count internal_edges) (count aliases)))))

(define join_order_subtree_has_hyperedge? (lambda (plan predicates)
	(begin
		(define aliases (join_order_plan_aliases plan))
		(reduce predicates (lambda (found predicate)
			(or found
				(and (> (count (join_order_pred_aliases predicate)) 2)
					(join_order_set_subset? (join_order_pred_aliases predicate) aliases)))) false))))

(define join_order_subtree_needs_exact_dp? (lambda (plan regular_edges predicates)
	(or (join_order_subtree_has_regular_cycle? plan regular_edges)
		(join_order_subtree_has_hyperedge? plan predicates))))

(define join_order_expensive_subtree (lambda (plan parent_size limit regular_edges predicates)
	(if (or (nil? plan) (join_order_plan_atomic? plan)
		(equal? (join_order_plan_size plan) 1))
		nil
		(begin
			(define own (if (and (<= (join_order_plan_size plan) limit)
				(and (> parent_size limit)
					(join_order_subtree_needs_exact_dp? plan regular_edges predicates))) plan nil))
			(define left (join_order_expensive_subtree
				(join_order_plan_left plan) (join_order_plan_size plan) limit regular_edges predicates))
			(define right (join_order_expensive_subtree
				(join_order_plan_right plan) (join_order_plan_size plan) limit regular_edges predicates))
			(reduce (list own left right) (lambda (best candidate)
				(if (and (not (nil? candidate))
					(or (nil? best) (> (join_order_plan_cost candidate) (join_order_plan_cost best))))
					candidate best)) nil)))))

(define join_order_replace_subtree (lambda (universe predicates plan target replacement required_drivers)
	(if (equal? plan target)
		replacement
		(if (nil? (join_order_plan_left plan))
			plan
			(join_order_join_plan universe predicates
				(join_order_replace_subtree universe predicates (join_order_plan_left plan) target replacement required_drivers)
				(join_order_replace_subtree universe predicates (join_order_plan_right plan) target replacement required_drivers))))))

(define join_order_goo_dp_loop (lambda (nodes aliases predicates regular_edges plan budget used required_drivers)
	(if (<= budget 0)
		(list plan used)
		(begin
			/* The global greedy tree composes independent connected arms. Exact
			DP is deliberately confined to small connected subtrees, which are
			then marked atomic and behave as the paper's meta-nodes. */
			(define limit 10)
			(define target (join_order_expensive_subtree plan
				(+ (join_order_plan_size plan) 1) limit regular_edges predicates))
			(if (nil? target)
				(list plan used)
				(begin
					(define optimized (join_order_dphyp_budgeted nodes
						(join_order_plan_aliases target) predicates budget required_drivers))
					(define replacement (car optimized))
					(define entries (cadr optimized))
					(define accepted (and (not (nil? replacement))
						(< (join_order_plan_cost replacement) (join_order_plan_cost target))))
					(define final_replacement (join_order_plan_with_atomic
						(if accepted replacement target) true))
					(join_order_goo_dp_loop nodes aliases predicates regular_edges
						(join_order_replace_subtree aliases predicates plan target final_replacement required_drivers)
						(- budget entries) (+ used entries) required_drivers))))))))

(define join_order_goo_dp (lambda (nodes aliases predicates _hypergraph required_drivers)
	(join_order_goo_dp_loop nodes aliases predicates
		(join_order_regular_edges aliases predicates)
		(join_order_goo nodes aliases predicates required_drivers) 10000 0 required_drivers)))

(define join_order_arm_dp (lambda (nodes aliases predicates required_drivers)
	(begin
		(define regular_edges (join_order_regular_edges aliases predicates))
		(define plan (join_order_arm_plan
			nodes aliases predicates regular_edges required_drivers))
		/* Multi-parent outer dependencies cannot be represented as independent
		arms. The ordinary hypergraph-aware composer remains the correctness path
		when the rooted construction cannot consume every pending boundary. */
		(if (or (nil? plan) (not (equal? (join_order_plan_size plan) (count aliases))))
			(join_order_goo_dp nodes aliases predicates true required_drivers)
			/* A connected regular graph with fewer edges than vertices is acyclic.
			There is no coupled subset for exact DP to improve, so avoid even walking
			the plan and rescanning its edges. */
			(if (< (count regular_edges) (count aliases))
				(list plan 0)
				(join_order_goo_dp_loop nodes aliases predicates regular_edges
					plan 10000 0 required_drivers))))))

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
		(if hypergraph (quote goo-dphyp) (quote decomposed-dphyp)))))

(define planner_record_table_statistics_guards (lambda (sources planning_session)
	(reduce (filter sources source_is_base_table?) (lambda (_ src)
		(begin
			(define token (table_planner_statistics_token
				(table (source_schema src) (source_relation src))))
			(planner_record_guard_condition
				(list (quote equal?)
					(list (quote table_planner_statistics_token)
						(list (quote table) (source_schema src) (source_relation src)))
					token) planning_session))) nil)))

(define planner_table_statistics_aliases (lambda (sources)
	(map (filter sources source_is_base_table?) source_alias)))

/* Equality and join selectivity depend only on immutable table statistics.
Text-pattern priors additionally depend on the current pattern value, so keep
that value in the guard while replacing repeated statistic derivation with one
O(1) dependency token per base table. */
(define planner_selectivity_value_dependent? (lambda (expr)
	(match expr
		((symbol strlike) _value _pattern) true
		((quote strlike) _value _pattern) true
		((symbol strlike) _value _pattern _collation) true
		((quote strlike) _value _pattern _collation) true
		_ false)))

(define join_order_record_cost_dependencies (lambda (sources nodes predicates planning_session)
	(begin
		(planner_record_table_statistics_guards sources planning_session)
		(define table_aliases (planner_table_statistics_aliases sources))
		/* Derived and synthetic nodes have no table snapshot whose token can cover
		their runtime cardinality. Preserve the exact guard for those nodes. */
		(reduce nodes (lambda (_ node)
			(if (contains? table_aliases (car node))
				nil
				(planner_record_guard_condition
					(list (quote equal?)
						(if (> (count node) 2) (nth node 2) (cadr node))
						(cadr node)) planning_session))) nil)
		(reduce predicates (lambda (_ predicate)
			(begin
				(define expr (join_order_pred_expr predicate))
				(if (planner_selectivity_value_dependent? expr)
					(planner_record_session_value_guards expr planning_session) nil))) nil))))

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

(define join_order_degree_index (lambda (edges)
	(reduce edges (lambda (degrees edge)
		(set_assoc
			(set_assoc degrees (nth edge 0) (+ (get_assoc degrees (nth edge 0) 0) 1))
			(nth edge 1) (+ (get_assoc degrees (nth edge 1) 0) 1))) '())))

(define join_order_degree_proves_budget_overflow? (lambda (aliases edges budget)
	(begin
		(define degrees (join_order_degree_index edges))
		(reduce aliases (lambda (proven alias)
			(or proven (join_order_degree_exceeds_budget?
				(get_assoc degrees alias 0) budget))) false))))

(define join_order_ordered_completion (lambda (predicates ordered remaining)
	(if (empty_list? remaining)
		ordered
		(begin
			(define next (coalesceNil
				(find remaining (lambda (alias)
					(join_order_connected? predicates ordered (list alias))) nil)
				(car remaining)))
			(join_order_ordered_completion predicates
				(append ordered next)
				(filter remaining (lambda (alias) (not (equal? alias next)))))))))

(define join_order_plan_for_alias_order (lambda (nodes universe predicates aliases)
	(match aliases
		(cons alias rest) (reduce rest (lambda (plan next_alias)
			(join_order_join_plan universe predicates plan
				(join_order_leaf_plan (join_order_find_node nodes next_alias))))
			(join_order_leaf_plan (join_order_find_node nodes alias)))
		_ nil)))

(define join_order_required_property_plan (lambda (nodes aliases predicates required_property)
	(begin
		(define ordered_aliases (join_order_required_aliases required_property))
		(if (empty_list? ordered_aliases)
			nil
			(begin
				(define remaining (filter aliases (lambda (alias)
					(not (contains? ordered_aliases alias)))))
				(join_order_plan_for_alias_order nodes aliases predicates
					(join_order_ordered_completion predicates ordered_aliases remaining)))))))

/* Connected subsets are the dominant retained state of DPHyp. Keeping this
budget below the measured compile-time cliff makes the exact search predictable;
the fallback still evaluates the same execution-cost function. */
(define join_order_dp_state_budget (lambda ()
	(begin
		(define configured (settings "JoinReorderDPBudget"))
		(if (> configured 0) configured 256))))

/* Independent guaranteed singletons, and 0:1 nullable lookups without a
post-extension filter, have a cost-equivalent relative order: neither shape can
multiply its preserved input. Build those proven trees directly instead of
running DP over a wide projection-only graph. */
(define join_order_functional_relations_plan (lambda (nodes aliases raw_predicates predicates required_drivers)
	(begin
		(define all_singletons (and (not (empty_list? nodes))
			(and (empty_list? raw_predicates)
				(reduce nodes (lambda (valid node)
					(and valid
						(and (equal? (join_order_node_kind node) (quote inner))
							(join_order_node_guaranteed_singleton? node)))) true))))
		(define inner_nodes (filter nodes (lambda (node)
			(equal? (join_order_node_kind node) (quote inner)))))
		(define driver (if (single_source? inner_nodes) (car inner_nodes) nil))
		(define driver_aliases (if (nil? driver) '() (list (car driver))))
		(define nullable_nodes (filter nodes (lambda (node)
			(equal? (join_order_node_kind node) (quote left-outer)))))
		(define no_nullable_post_filter (reduce raw_predicates (lambda (valid predicate)
			(and valid (nil? (join_order_pred_barrier_owner predicate)))) true))
		(define functional (and (not (nil? driver))
			(and (equal? (+ 1 (count nullable_nodes)) (count nodes))
				(and (or no_nullable_post_filter (equal? (count nodes) 2))
					(reduce nullable_nodes (lambda (valid node)
						(and valid
							(and (equal? (join_order_node_max_rows_per_driver node) 1)
								(join_order_set_subset?
									(join_order_node_requirements node)
									driver_aliases)))) true)))))
		(if all_singletons
			(begin
				(define plan (join_order_plan_for_alias_order nodes aliases predicates aliases))
				(if (join_order_plan_satisfies_driver_property? plan required_drivers) plan nil))
			(if functional
				(begin
					(define ordered_aliases (cons (car driver)
						(filter aliases (lambda (alias) (not (equal? alias (car driver)))))))
					(define plan (join_order_plan_for_alias_order nodes aliases predicates ordered_aliases))
					(if (join_order_plan_satisfies_driver_property? plan required_drivers) plan nil))
				nil)))))

(define join_order_adaptive (lambda (sources nodes raw_predicates required_drivers planning_session)
	(begin
		(define aliases (map nodes car))
		/* Non-inner join constraints are hypergraph constraints even when their ON
		predicate happens to mention only two aliases. Linearized DP cannot encode
		the pending preserved-side requirement, so route an over-budget outer-join
		graph through the paper's hypergraph-aware GOO-DPHyp path. */
		(define predicate_hypergraph (join_order_hypergraph? raw_predicates))
		(define hypergraph (or predicate_hypergraph
			(join_order_has_outer_barriers? nodes)))
		(define predicates (join_order_prepare_predicates aliases raw_predicates))
		(define functional_relation_plan
			(join_order_functional_relations_plan
				nodes aliases raw_predicates predicates required_drivers))
		(define regular_edges (join_order_regular_edges aliases predicates))
		(define state_budget (join_order_dp_state_budget))
		(define connected_count (if (not (nil? functional_relation_plan))
			(list '() false)
			(if (<= (count aliases) 100)
				(if (join_order_degree_proves_budget_overflow? aliases regular_edges state_budget)
					(list '() true)
					(join_order_enumerate_connected aliases predicates state_budget))
				(list '() true))))
		(define strategy (join_order_choose_strategy
			(count aliases) predicate_hypergraph (cadr connected_count)))
		(define exact (equal? strategy (quote dphyp)))
		/* Cached plans depend on the sampled cardinalities even when DPHyp keeps a
		fixed physical driver. Guard every cost input once. Pairwise DP inequalities
		are deterministic consequences of these inputs and retaining all of them
		would duplicate complete intermediate plan trees. */
		(join_order_record_cost_dependencies sources nodes predicates planning_session)
		(define result (if (not (nil? functional_relation_plan))
			(list functional_relation_plan (- (count aliases) 1))
			(if exact
				(join_order_dphyp_connected nodes aliases predicates (car connected_count) required_drivers)
				(if (equal? strategy (quote decomposed-dphyp))
					(join_order_arm_dp nodes aliases predicates required_drivers)
					(join_order_goo_dp nodes aliases predicates hypergraph required_drivers)))))
		(define result_plan (car result))
		/* One cheapest state per alias set is sufficient for inner joins, but a
		LEFT boundary adds a pending-null-extension property. The cheapest state
		can therefore be a dead end even though the original logical source order
		is valid. Keep that order as the general correctness completion; operator
		selection and all successfully reorderable plans remain cost based. */
		(define source_order_plan (if (nil? result_plan)
			(join_order_plan_for_alias_order nodes aliases predicates aliases)
			nil))
		(define forced_order_plan (if (or (nil? result_plan)
			(not (join_order_plan_satisfies_driver_property? result_plan required_drivers)))
			(join_order_required_property_plan nodes aliases predicates required_drivers)
			nil))
		(define completion_plan (if (and (not (nil? forced_order_plan))
			(join_order_plan_satisfies_driver_property? forced_order_plan required_drivers))
			forced_order_plan
			(if (and (not (nil? source_order_plan))
				(join_order_plan_satisfies_driver_property? source_order_plan required_drivers))
				source_order_plan nil)))
		(define selected (if (nil? completion_plan) result
			(list completion_plan (+ (cadr result) 1))))
		(define pipeline_result (if (or (not (number? (join_order_pipeline_width required_drivers)))
			(cadr connected_count))
			(list nil 0)
			(join_order_pipeline_candidate nodes aliases predicates
				(car connected_count) required_drivers)))
		(define pipeline_plan (car pipeline_result))
		(define pipeline_cost (if (nil? pipeline_plan) nil
			(join_order_pipeline_cost nodes pipeline_plan
				(join_order_pipeline_width required_drivers))))
		(define pipeline_wins (and (not (nil? pipeline_plan))
			(and (join_order_plan_satisfies_driver_property? pipeline_plan required_drivers)
				(or (nil? (car selected))
					(planner_cost_better? pipeline_cost
						(join_order_plan_cost_domain (car selected)))))))
		(define chosen_plan (if pipeline_wins
			(join_order_plan_with_cost pipeline_plan pipeline_cost) (car selected)))
		(define chosen_entries (+ (cadr selected)
			(if pipeline_wins (cadr pipeline_result) 0)))
		(if (nil? chosen_plan)
			(neumann_fail "join_reorder" (concat "SCM join ordering could not construct a connected plan: " (string (list nodes predicates result))))
			(if (not (join_order_plan_satisfies_driver_property? chosen_plan required_drivers))
				(neumann_fail "join_reorder" "costed join plan cannot preserve the required ORDER BY driver")
				(join_order_result (if (not (nil? functional_relation_plan))
					(quote functional-relations)
					(if pipeline_wins (quote pipeline-left-deep)
						(if (nil? completion_plan) strategy
							(if (equal? completion_plan forced_order_plan)
								(quote ordered-property) (quote source-order-completion)))))
					chosen_plan chosen_entries predicates)))))))

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
					(neumann_fail "build_queryplan" (concat
						"logical join plan does not cover the query-block sources exactly once: plan="
						(serialize aliases) " sources=" (serialize source_alias_list))))
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
					(list (quote membership_candidate_matching_rows) (qassoc_get facts (quote membership_candidate_matching_rows) nil))
					(list (quote membership_candidate_estimate_capped) (qassoc_get facts (quote membership_candidate_estimate_capped) false))
					(list (quote membership_candidate_estimate_input) (qassoc_get facts (quote membership_candidate_estimate_input) nil))
					(list (quote membership_candidate_estimate_sampled) (qassoc_get facts (quote membership_candidate_estimate_sampled) nil))
					(list (quote membership_candidate_estimate_population) (qassoc_get facts (quote membership_candidate_estimate_population) (quote table_rows)))
					(list (quote membership_candidate_estimate_coverage) (qassoc_get facts (quote membership_candidate_estimate_coverage) (quote sampled)))
					(list (quote membership_candidate_scan_invocations) (qassoc_get facts (quote membership_candidate_scan_invocations) 1))
					(list (quote membership_candidate_filter_columns) (qassoc_get facts (quote membership_candidate_filter_columns) 0))
					(list (quote membership_candidate_map_columns) (qassoc_get facts (quote membership_candidate_map_columns) 1))
					(list (quote membership_candidate_cache_map_columns) (qassoc_get facts (quote membership_candidate_cache_map_columns) 2))
					(list (quote membership_candidate_cache_backed) (qassoc_get facts (quote membership_candidate_cache_backed) false))
					(list (quote membership_candidate_expression_operations) (qassoc_get facts (quote membership_candidate_expression_operations) 0))
					(list (quote membership_candidate_expression_depth) (qassoc_get facts (quote membership_candidate_expression_depth) 0))
					(list (quote membership_candidate_index_filter_rows) (qassoc_get facts (quote membership_candidate_index_filter_rows) nil))
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
			(define physical_planned (query_block_with_physical_requirement_facts planned))
			(define physical_stages (map (qb_stages physical_planned) apply_join_optimizer_plan_stage))
			(define requirement (qassoc_get (qb_facts physical_planned) (quote membership_requirement) nil))
			/* Reorder attaches the requirement to the query block which owns every
			membership_requirement_probe. Nested blocks and stages recurse separately.
			Keep the ordinary query expressions structurally shared instead of walking
			and rebuilding the complete block when there is no physical choice here. */
			(if (nil? requirement)
				(make_query_block
					(qb_schema physical_planned) (qb_sources physical_planned)
					(qb_fields physical_planned) (qb_where physical_planned)
					(qb_group physical_planned) (qb_having physical_planned)
					(qb_order physical_planned) (qb_limit physical_planned)
					(qb_offset physical_planned) (qb_hidden physical_planned)
					physical_stages (qb_facts physical_planned))
				(begin
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
						physical_stages
						(physicalize_membership_requirement_expr_using physical_facts physical_facts)))))
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

(define join_optimizer_plan_segment (lambda (stages relation_units all_sources segment default_alias graph required_drivers planning_session)
	(begin
		(define aliases (map segment source_alias))
		(define alias_index (join_hypergraph_alias_index aliases))
		(define predicates (join_optimizer_metadata_costed_predicates
			all_sources default_alias graph aliases))
		(define fixed_functional_pair (and (equal? (count segment) 2)
			(and (empty_list? required_drivers)
				(begin
					(define driver (nth segment 0))
					(define lookup (nth segment 1))
					(define lookup_stage (join_optimizer_source_stage stages lookup))
					(and (join_optimizer_inner_source? stages driver)
						(and (not (join_optimizer_inner_source? stages lookup))
							(and (group_stage? lookup_stage)
								(and (equal? (stage_result_max_rows_per_partition lookup_stage) 1)
									(join_order_set_subset?
										(join_optimizer_outer_requirements stages relation_units
											segment default_alias alias_index lookup)
										(list (source_alias driver)))))))))))
		(define planned (join_order_adaptive segment
			(join_optimizer_metadata_nodes stages relation_units
				segment default_alias alias_index graph fixed_functional_pair)
			predicates
			required_drivers
			planning_session))
		(qassoc_set planned (quote tree)
			(join_order_tree_with_predicates
				(qassoc_get planned (quote tree) nil)
				predicates)))))

(define join_optimizer_reorder_result (lambda (tree strategy dp_entries cost cardinality cost_components properties)
	(list tree strategy dp_entries cost cardinality cost_components properties)))

(define join_optimizer_required_order_aliases (lambda (sources default_alias order_items)
	(reduce (order_exprs order_items) (lambda (aliases expr)
		(merge_unique (list aliases
			(join_hypergraph_expr_aliases default_alias (source_aliases sources) expr)))) '())))

(define join_optimizer_source_column_constant_bound? (lambda (sources default_alias src col condition)
	(reduce (split_and_terms (coalesceNil condition true)) (lambda (found term)
		(if found
			true
			(match term
				'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
					(or
						(and (equal?? (direct_column_name_for_alias src left) col)
							(empty_list? (join_hypergraph_expr_aliases
								default_alias (source_aliases sources) right)))
						(and (equal?? (direct_column_name_for_alias src right) col)
							(empty_list? (join_hypergraph_expr_aliases
								default_alias (source_aliases sources) left))))
					false)
				_ false))) false)))

(define join_optimizer_source_constant_unique_point? (lambda (sources default_alias src condition)
	(if (not (source_is_base_table? src))
		false
		(reduce (source_unique_key_sets src) (lambda (found key_cols)
			(or found (reduce key_cols (lambda (complete col)
				(and complete (join_optimizer_source_column_constant_bound?
					sources default_alias src col condition))) true))) false))))

(define join_optimizer_source_local_condition (lambda (graph src)
	(combine_where
		(source_join_expr src)
		(combine_where_terms
			(map (join_optimizer_local_predicates graph (source_alias src))
				(lambda (entry) (qassoc_get entry (quote predicate) true)))
			true))))

(define join_optimizer_constant_unique_order_prefix_aliases (lambda (sources default_alias graph stages)
	(map (filter sources (lambda (src)
		(and (join_optimizer_inner_source? stages src)
			(join_optimizer_source_constant_unique_point?
				sources default_alias src
				(join_optimizer_source_local_condition graph src)))))
		source_alias)))

(define join_optimizer_required_order_prefix (lambda (sources required_aliases condition stages prefix)
	(if (or (empty_list? sources)
		(or (empty_list? required_aliases)
			(equal? (source_alias (car sources)) (car required_aliases))))
		prefix
		(if (or
			(constant_scalar_or_presence_stage_output_source? stages (car sources))
			(join_optimizer_source_constant_unique_point? sources
				(source_alias (car sources)) (car sources)
				(combine_where condition (source_join_expr (car sources)))))
			(join_optimizer_required_order_prefix (cdr sources) required_aliases condition stages
				(append prefix (source_alias (car sources))))
			prefix))))

/* Keep one complete DPHyp result for every base-table driver which can realize
the required order. Combined ordered joins have directional work: the driver
brakes at LIMIT while every other input is prepared for joined lookup. A single
memo winner for a set of aliases would discard that physical property before
the lowerer can cost it. */
(define join_optimizer_ordered_source_rows (lambda (stage_catalog sources default_alias graph src planning_session tx)
	(begin
		(define fallback (join_optimizer_source_rows
			stage_catalog sources default_alias graph src))
		(define base_rows (planner_source_row_count src))
		(define condition (join_optimizer_source_local_condition graph src))
		(if (or (not (number? base_rows)) (equal? condition true))
			fallback
			(begin
				(define estimate (planner_source_filter_estimate src condition 512 tx planning_session))
				(max 1 (planner_estimated_matching_rows estimate base_rows fallback)))))))

(define join_optimizer_ordered_driver_work (lambda (sources base_row_catalog filtered_row_catalog planned target)
	(begin
		(define ordered_sources (join_optimizer_sources_for_order sources
			(join_optimizer_tree_aliases (qassoc_get planned (quote tree) nil))))
		(define base_rows (map ordered_sources (lambda (src)
			(qassoc_get base_row_catalog (source_alias src) 1))))
		(define driver_filtered_rows
			(qassoc_get filtered_row_catalog (source_alias (car ordered_sources)) (car base_rows)))
		(define driver_rows (planner_ordered_driver_rows_visited
			(car base_rows) driver_filtered_rows target))
		(define inner_rows (reduce (cdr base_rows) + 0))
		(define joined_rows (qassoc_get planned (quote cardinality) 1))
		(list (planner_scan_join_order_orientation_cost driver_rows inner_rows
			(count ordered_sources) target)
			driver_rows inner_rows joined_rows))))

(define join_optimizer_ordered_driver_candidate_better? (lambda (current candidate)
	(if (nil? current)
		true
		(planner_cost_better? (cadr candidate) (cadr current)))))

(define join_optimizer_plan_ordered_drivers (lambda (stage_catalog relation_units sources default_alias graph ordered_drivers target planning_session tx)
	(begin
		(define base_row_catalog (map sources (lambda (src)
			(list (source_alias src)
				(coalesceNil (planner_source_row_count src)
					(join_optimizer_source_rows stage_catalog sources default_alias graph src))))))
		(define filtered_row_catalog (map sources (lambda (src)
			(list (source_alias src)
				(join_optimizer_ordered_source_rows
					stage_catalog sources default_alias graph src planning_session tx)))))
		(reduce ordered_drivers (lambda (state driver)
			(begin
				(define planned (join_optimizer_plan_segment stage_catalog relation_units
					sources sources default_alias graph (list driver) planning_session))
				(define work (join_optimizer_ordered_driver_work
					sources base_row_catalog filtered_row_catalog planned target))
				(define candidate (list planned (car work)
					driver))
				(list (if (join_optimizer_ordered_driver_candidate_better? (car state) candidate)
					candidate (car state))
					(append (cadr state) (list (list driver
						(qassoc_get (car work) (quote total_ns) nil)
						(nth work 1) (nth work 2) (nth work 3)))))))
			(list nil '())))))

(define join_optimizer_reorder_sources (lambda (stage_catalog block graph planning_session tx)
	(begin
		(define sources (qb_sources block))
		(define segment sources)
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		/* GROUP BY/DISTINCT materializes its own ordered result. Its input join is
		unordered, so do not impose the downstream ORDER BY on the join driver. */
		(define join_order_items (if (and (empty_list? (qb_group block))
			(not (query_block_has_aggregates? block)))
			(qb_order block) '()))
		/* ORDER BY is a required physical property, not a reason to retain SQL
		source order. Any source whose ordered scan can evaluate the complete
		expression -- including unique lookup/scalar-subscan columns -- is an
		eligible driver. Keeping that property in DP dominance prevents the later
		lowerer from materializing and sorting an intermediate relation. */
		(define ordered_drivers (if (empty_list? join_order_items)
			'()
			(map (filter sources (lambda (src)
				(and (join_optimizer_inner_source? stage_catalog src)
					(order_items_supported_by_join_driver?
						sources default_alias src join_order_items stage_catalog
						(coalesceNil (qb_where block) true)))))
				source_alias)))
		/* If no single scan can provide the complete ORDER expression, retain the
		lexicographic source prefix as a required physical property. DP still costs
		all valid trees; it merely stops discarding the best tree whose nested scan
		order can produce the requested suffix without a sorting relation. */
		(define raw_required_order_aliases (if (or (empty_list? join_order_items)
			(not (empty_list? ordered_drivers)))
			'()
			(join_optimizer_required_order_aliases sources default_alias join_order_items)))
		(define required_order_aliases (if (empty_list? raw_required_order_aliases)
			'()
			(merge_unique (list
				(join_optimizer_required_order_prefix sources raw_required_order_aliases
					(coalesceNil (qb_where block) true) stage_catalog '())
				raw_required_order_aliases))))
		/* A constant unique lookup emits at most one row and therefore cannot alter
		the order of a following ordered scan. Putting every such inner source before
		the ordered driver strictly dominates repeating the same point lookup from
		every driver row. Express that proof through the existing ordered-prefix
		property so DP and the lowerer keep one shared notion of valid order. */
		(define constant_unique_aliases
			(join_optimizer_constant_unique_order_prefix_aliases
				sources default_alias graph stage_catalog))
		/* A singleton can technically be reported as an ordered driver because its
		downstream scan supplies the order. It must still remain in the prefix: using
		it as the repeated ordered root would throw away the singleton proof. Prefer
		the first non-singleton ordered source as the actual order producer. */
		(define non_singleton_ordered_drivers (filter ordered_drivers (lambda (alias)
			(not (contains? constant_unique_aliases alias)))))
		(define selected_ordered_driver (if (not (empty_list? non_singleton_ordered_drivers))
			(car non_singleton_ordered_drivers)
			(if (empty_list? ordered_drivers) nil (car ordered_drivers))))
		(define dominant_singleton_prefix (if (nil? selected_ordered_driver)
			'()
			(filter constant_unique_aliases (lambda (alias)
				(not (equal? alias selected_ordered_driver))))))
		(define base_required_order_property (if (empty_list? required_order_aliases)
			(if (empty_list? dominant_singleton_prefix)
				ordered_drivers
				(list (list (quote ordered_aliases)
					(merge (list dominant_singleton_prefix (list selected_ordered_driver))))))
			(list (list (quote ordered_aliases) required_order_aliases))))
		(define pipeline_reduce_width (if (and (empty_list? (coalesceNil (qb_order block) '()))
			(and (empty_list? (coalesceNil (qb_stages block) '()))
				(or (not (empty_list? (coalesceNil (qb_group block) '())))
					(query_block_has_aggregates? block))))
			(+ (count (coalesceNil (qb_group block) '()))
				(count (coalesceNil (qb_fields block) '()))) nil))
		(define required_order_property (if (number? pipeline_reduce_width)
			(append base_required_order_property
				(list (quote pipeline_reduce_width) pipeline_reduce_width))
			base_required_order_property))
		(define preserve_row_number_driver (and
			(not (empty_list? segment))
			(query_limit_active? (qb_offset block) (qb_limit block))
			(empty_list? (qb_group block))
			(not (query_block_has_aggregates? block))
			(source_row_number_limit_driver? (qb_stages block) (car segment))))
		(if (or (< (count segment) 2) preserve_row_number_driver)
			(join_optimizer_reorder_result
				(join_optimizer_left_deep_tree sources)
				(if preserve_row_number_driver (quote preserve-row-number-limit) (quote fixed))
				0 nil nil nil '())
			(begin
				(define relation_units
					(qassoc_get (qb_facts block) (quote join_relation_units) '()))
				(define literal_limit (planner_literal_value (qb_limit block) planning_session))
				(define literal_offset (coalesceNil
					(planner_literal_value (qb_offset block) planning_session) 0))
				(define enumerate_ordered_drivers (and
					(number? literal_limit)
					(and (>= literal_limit 0)
						(and (number? literal_offset)
							(and (empty_list? required_order_aliases)
								(and (> (count ordered_drivers) 1)
									(and (empty_list? (qb_stages block))
										(reduce sources (lambda (supported src)
											(and supported (and (source_is_base_table? src)
												(join_optimizer_inner_source? stage_catalog src)))) true))))))))
				(define ordered_driver_plans (if enumerate_ordered_drivers
					(join_optimizer_plan_ordered_drivers stage_catalog relation_units
						sources default_alias graph ordered_drivers (+ literal_offset literal_limit)
						planning_session tx) nil))
				(define ordered_choice (if (nil? ordered_driver_plans) nil
					(car ordered_driver_plans)))
				(define planned (if (nil? ordered_choice)
					(join_optimizer_plan_segment stage_catalog relation_units
						sources segment default_alias graph required_order_property planning_session)
					(car ordered_choice)))
				(join_optimizer_reorder_result
					(qassoc_get planned (quote tree) nil)
					(qassoc_get planned (quote strategy) (quote fixed))
					(qassoc_get planned (quote dp_entries) 0)
					(qassoc_get planned (quote cost) nil)
					(qassoc_get planned (quote cardinality) nil)
					(qassoc_get planned (quote cost_components) nil)
					(list
						(list (quote ordered_driver_candidates) ordered_drivers)
						(list (quote selected_ordered_driver)
							(if (nil? ordered_choice) nil (nth ordered_choice 2)))
						(list (quote ordered_driver_costs)
							(if (nil? ordered_driver_plans) '() (cadr ordered_driver_plans))))))))))

(define join_optimizer_telemetry (lambda (graph reordered)
	(list
		(list (quote join_reorder_strategy) (nth reordered 1))
		(list (quote join_plan) (nth reordered 0))
		(list (quote join_driver) (car (join_optimizer_tree_aliases (nth reordered 0))))
		(list (quote join_dp_entries) (nth reordered 2))
		(list (quote join_estimated_cost) (nth reordered 3))
		(list (quote join_estimated_rows) (nth reordered 4))
		(list (quote join_cost) (nth reordered 5))
		(list (quote join_order_properties) (nth reordered 6))
		(list (quote join_dp_state_budget) (join_order_dp_state_budget))
		(list (quote join_graph_nodes) (count (qassoc_get graph (quote nodes) '())))
		(list (quote join_graph_edges) (count (qassoc_get graph (quote edges) '())))
		(list (quote join_graph_hyperedges) (count (qassoc_get graph (quote hyperedges) '())))
		(list (quote join_graph_barriers) (count (qassoc_get graph (quote barriers) '()))))))

(define hybrid_reorder_query_block_using (lambda (stage_catalog block planning_session tx)
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
			(map (qb_stages block) (lambda (stage) (join_reorder_stage_using stage_catalog stage planning_session tx)))
			(qb_facts block))
		(begin
			(define provenance_graph (extract_join_hypergraph block))
			(define normalized (join_optimizer_normalize_inner_joins stage_catalog block))
			(define graph (join_hypergraph_with_provenance
				(extract_join_hypergraph normalized) provenance_graph))
			(define reordered (join_optimizer_reorder_sources stage_catalog normalized graph planning_session tx))
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
					(map (qb_stages normalized) (lambda (stage) (join_reorder_stage_using stage_catalog stage planning_session tx)))
					(qb_facts normalized))
				(merge (list
					(query_block_reorder_telemetry normalized planning_session)
					(list
						(list (quote join_order_planning_offset)
							(coalesceNil (planner_literal_value (qb_offset normalized) planning_session) 0))
						(list (quote join_order_planning_limit)
							(coalesceNil (planner_literal_value (qb_limit normalized) planning_session) -1)))
					(join_optimizer_telemetry graph reordered))))))))

(define hybrid_reorder_query_block (lambda (block planning_session tx)
	(hybrid_reorder_query_block_using (qb_stages block) block planning_session tx)))
(define planner_literal_value (lambda (expr planning_session)
	(begin
		(define planning_session (planner_effective_session planning_session))
		(match expr
			((symbol session) key) (try
				(lambda () (begin
					(define compile_bindings (if (nil? planning_session) nil (planning_session "__memcp_queryplan_compile_bindings")))
					(define observed (if (nil? planning_session) nil (planning_session "__memcp_queryplan_observed_session_keys")))
					(if (and (not (nil? compile_bindings)) (not (nil? observed)))
						(observed key true)
						nil)
					(define direct (if (nil? planning_session) nil (planning_session key)))
					(if (not (nil? direct))
						direct
						(if (nil? compile_bindings) nil (compile_bindings key)))))
				(lambda (_e) nil))
			((quote session) key) (planner_literal_value (list (quote session) key) planning_session)
			_ expr))))

/* Sampling callbacks run outside the generated query closure. Replace only
the session reads observed in the isolated compile scope; the corresponding
runtime guards keep the cached choice valid for those exact values. */
(define planner_bind_session_values (lambda (expr planning_session)
	(if (nil? planning_session)
		expr
		(match expr
			((symbol session) _key)
			(planner_quoted_value (planner_literal_value expr planning_session))
			((quote session) _key)
			(planner_quoted_value (planner_literal_value expr planning_session))
			(cons head tail) (cons
				(planner_bind_session_values head planning_session)
				(map tail (lambda (item)
					(planner_bind_session_values item planning_session))))
			_ expr))))

/* Physical selectivity decisions over session-dependent predicates must be
guarded by the values observed while compiling the cached plan. */
(define planner_record_session_value_guards (lambda (node planning_session)
	(reduce (query_expr_session_reads node) (lambda (_ expr)
		(begin
			(define value (planner_literal_value expr planning_session))
			(planner_record_guard_condition
				(list (quote equal?) expr
					(if (list? value) (list (quote quote) value) value)) planning_session))) nil)))

(define planner_concat_expr_value (lambda (items planning_session)
	(match items
		(cons item rest) (begin
			(define value (planner_string_expr_value item planning_session))
			(define tail (planner_concat_expr_value rest planning_session))
			(if (and (string? value) (string? tail)) (concat value tail) nil))
		_ "")))

(define planner_string_expr_value (lambda (expr planning_session)
	(begin
		(define value (planner_literal_value expr planning_session))
		(if (string? value)
			value
			(match expr
				(cons head args) (if (or (equal? head (symbol "concat")) (equal? head (quote concat)))
					(planner_concat_expr_value args planning_session) nil)
				_ nil)))))

(define like_pattern_core (lambda (pattern)
	(replace (replace (coalesceNil pattern "") "%" "") "_" "")))

(define text_pattern_selectivity_prior_for_length (lambda (length)
	(if (<= length 0)
		1
		(if (equal? length 1)
			0.7
			(max 0.01 (* 0.35
				(text_pattern_selectivity_prior_for_length (- length 1))))))))

/* Before an index has a measured candidate density, text length is still
useful statistical evidence. Single-character searches are usually broad;
each additional literal character rapidly narrows the candidate set. The
floor avoids pretending that an unseen word is impossible. */
(define text_pattern_selectivity_prior (lambda (pattern)
	(if (not (string? pattern))
		nil
		(text_pattern_selectivity_prior_for_length
			(strlen (like_pattern_core pattern))))))

(define expr_text_selectivity_prior (lambda (expr)
	(match expr
		((symbol strlike) _value pattern _collation)
		(text_pattern_selectivity_prior (planner_string_expr_value pattern))
		((quote strlike) _value pattern _collation)
		(text_pattern_selectivity_prior (planner_string_expr_value pattern))
		(cons head tail) (reduce tail (lambda (found item)
			(if (number? found) found (expr_text_selectivity_prior item)))
			(expr_text_selectivity_prior head))
		_ nil)))

(define expr_text_pattern_expr (lambda (expr)
	(match expr
		((symbol strlike) _value pattern _collation) pattern
		((quote strlike) _value pattern _collation) pattern
		(cons head tail) (reduce tail (lambda (found item)
			(if (nil? found) (expr_text_pattern_expr item) found))
			(expr_text_pattern_expr head))
		_ nil)))

(define broad_like_pattern? (lambda (pattern)
	(if (not (string? pattern))
		false
		(begin
			(define core (like_pattern_core pattern))
			(<= (strlen core) 2)))))

(define planner_broad_like_expr? (lambda (pattern planning_session)
	(begin
		(define broad (broad_like_pattern? (planner_string_expr_value pattern planning_session)))
		(if (expr_contains_session_dependency? pattern)
			(planner_guarded_choice broad
				(list (quote broad_like_pattern?) pattern) planning_session)
			broad))))

(define expr_contains_broad_text_match? (lambda (expr planning_session)
	(match expr
		((symbol strlike) _value pattern _collation)
		(planner_broad_like_expr? pattern planning_session)
		((quote strlike) _value pattern _collation)
		(expr_contains_broad_text_match? (list (quote strlike) _value pattern _collation) planning_session)
		(cons head tail)
		(reduce tail (lambda (found item)
			(or found (expr_contains_broad_text_match? item planning_session)))
			(expr_contains_broad_text_match? head planning_session))
		_ false)))

(define expr_contains_text_match? (lambda (expr)
	(match expr
		((symbol strlike) _value _pattern _collation) true
		((quote strlike) _value _pattern _collation) true
		_ false)))

(define stage_input_contains_broad_membership_filter? (lambda (input planning_session)
	(if (union_block? input)
		(reduce (union_branches input) (lambda (found branch)
			(or found (stage_input_contains_broad_membership_filter? branch planning_session)))
			false)
		(if (query_block? input)
			(expr_contains_broad_text_match? (qb_where input) planning_session)
			false))))

(define candidate_stage_broad? (lambda (stage planning_session)
	(or (equal? (qassoc_get (gs_facts stage) (quote selectivity_class) nil) (quote broad))
		(stage_input_contains_broad_membership_filter? (gs_input stage) planning_session))))

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
			/* Storage already publishes one immutable, lock-free statistics
			snapshot per table. Reading it directly keeps this metadata lookup
			independent of request/session scope. */
			(table_planner_statistics (table schema relation)))
		(lambda (_e) nil))))

(define planner_table_row_count (lambda (schema relation)
	(begin
		(define stats (planner_table_statistics schema relation))
		(if (nil? stats) nil (stats "row_count")))))

(define planner_source_filter_estimate (lambda (src condition max_rows tx planning_session)
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
						(planner_bind_session_values
							(lower_column_expr_for_alias src condition) planning_session))))
					(define estimate (scan_selectivity_estimate
						tx
						(table (source_schema src) (source_relation src))
						filtercols
						(eval filter_expr)
						max_rows))
					(define text_prior (expr_text_selectivity_prior condition))
					(define enriched (if (number? text_prior)
						(qassoc_set estimate (quote fallback_selectivity) text_prior)
						estimate))
					(define source_rows (planner_source_row_count src))
					(if (number? source_rows)
						(qassoc_set enriched (quote estimated_rows)
							(planner_estimated_matching_rows enriched source_rows source_rows))
						enriched)))
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
	(if (stage_output_relation? (source_relation src))
		(begin
			(define stage (stage_for_output_relation stages (source_relation src)))
			(define singleton (and (group_stage? stage)
				(and (equal? (stage_result_max_rows_per_partition stage) 1)
					(and (empty_list? (qassoc_get (gs_facts stage) (quote partition_by) '()))
						(not (stage_has_residual_outer_refs? stage))))))
			(define input_rows (if (group_stage? stage)
				(planner_stage_input_rows (gs_input stage)) nil))
			(if singleton
				(planner_estimate 1 1 (quote semantic_singleton) true)
				(if (number? input_rows)
					(planner_estimate (max 1 input_rows) 0.5 (quote stage_input_upper_bound) true)
					(planner_estimate nil 0 (quote unknown) false))))
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

(define planner_column_average_value_bytes_from_statistics (lambda (statistics)
	(begin
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

(define planner_runtime_column_average_value_bytes (lambda (schema relation column)
	(begin
		(define stats (table_planner_statistics (table schema relation)))
		(define columns (if (nil? stats) nil (stats "columns")))
		(define column_stats (if (nil? columns) nil (columns column)))
		(planner_column_average_value_bytes_from_statistics column_stats))))

(define planner_column_average_value_bytes (lambda (src column)
	(begin
		(define measured (planner_column_average_value_bytes_from_statistics
			(planner_column_statistics src column)))
		(if (source_is_base_table? src)
			/* Average width is a physical cost input refined by REBUILD. Keep the
			cached plan while it is unchanged; a different width recompiles the full
			cost inequality instead of making every guard walk the logical stage AST. */
			(planner_record_guard_condition
				(list (quote equal?)
					(list (quote planner_runtime_column_average_value_bytes)
						(source_schema src) (source_relation src) column)
					measured))
			nil)
		measured)))

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

(define explain_reorder_selectivities? (lambda (planning_session)
	(if (nil? planning_session) false
		(coalesceNil (planning_session "__memcp_explain_reorder_selectivities") false))))

(define query_block_reorder_telemetry (lambda (block planning_session)
	(merge (list
		(list (list (quote source_estimates) (map (qb_sources block) source_reorder_estimate)))
		(if (explain_reorder_selectivities? planning_session)
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
	(if (and (not (empty_list? estimates))
		(reduce estimates (lambda (all_hooks estimate)
			(and all_hooks (equal? (planner_estimate_population estimate)
				(quote index_hook_candidates)))) true))
		(quote index_hook_candidates)
		(if (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_population estimate) (quote index_candidates)))) false)
			(quote index_candidates)
			(if (reduce estimates (lambda (found estimate)
				(or found (equal? (planner_estimate_population estimate) (quote recset_candidates)))) false)
				(quote recset_candidates)
				(quote table_rows))))))

(define planner_merge_estimate_coverage (lambda (estimates)
	(begin
		(define has_lower (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_coverage estimate) (quote lower_bound)))) false))
		(define has_upper (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_coverage estimate) (quote upper_bound)))) false))
		(define has_sampled (reduce estimates (lambda (found estimate)
			(or found (equal? (planner_estimate_coverage estimate) (quote sampled)))) false))
		/* Adding UNION branch cardinalities preserves a one-sided bound only when
		every inexact branch points in the same direction. Mixed bounds describe an
		interval, while a sampled branch remains a sample of the complete sum. */
		(if (or has_sampled (and has_lower has_upper))
			(quote sampled)
			(if has_lower (quote lower_bound)
				(if has_upper (quote upper_bound) (quote exact)))))))

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

/* Equality bindings for every column of a unique key are a cardinality proof,
not a selectivity guess: the branch can return at most one row for any runtime
binding. Runtime sampling may still report the table-wide fallback before the
corresponding autoindex exists. Cap that estimate here so UNION membership
planning preserves the proven bound without consulting index availability. */
(define planner_unique_point_filter_estimate (lambda (src condition estimate)
	(if (not (join_optimizer_source_constant_unique_point?
		(list src) (source_alias src) src condition))
		estimate
		(begin
			(define input_rows (planner_source_row_count src))
			(define base (coalesceNil estimate (list
				(list (quote input) input_rows)
				(list (quote population) (quote table_rows))
				(list (quote coverage) (quote exact)))))
			(qassoc_set
				(qassoc_set
					(qassoc_set base (quote rows)
						(min 1 (coalesceNil (qassoc_get base (quote rows) nil) 1)))
					(quote estimated_rows)
					(min 1 (coalesceNil (qassoc_get base (quote estimated_rows) nil) 1)))
				(quote capped) false)))))

(define planner_stage_filter_estimate (lambda (input max_rows tx planning_session)
	(if (union_block? input)
		(begin
			(define branch_estimates (map (union_branches input) (lambda (branch)
				(planner_stage_filter_estimate branch max_rows tx planning_session))))
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
					(define estimated_rows (planner_add_estimates (map available (lambda (estimate)
						(qassoc_get estimate (quote estimated_rows)
							(qassoc_get estimate (quote rows) nil))))))
					(define capped (or (>= rows max_rows)
						(reduce available (lambda (found estimate)
							(or found (qassoc_get estimate (quote capped) false)))
							false)))
					(list
						(list (quote rows) rows)
						(list (quote capped) capped)
						(list (quote sampled) sampled_rows)
						(list (quote input) input_rows)
						(list (quote estimated_rows) estimated_rows)
						(list (quote population) (planner_merge_estimate_population available))
						(list (quote coverage) (planner_merge_estimate_coverage available))))))
		(if (query_block? input)
			(if (single_source? (qb_sources input))
				(begin
					(define src (car (qb_sources input)))
					(define condition (combine_where (qb_where input) (source_join_expr src)))
					(planner_unique_point_filter_estimate src condition
						(planner_source_filter_estimate src condition max_rows tx planning_session)))
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
	(if (and (query_block? branch) (candidate_recset_branch_supported? branch))
		(begin
			(define sources (qb_sources branch))
			(define tree (query_block_join_plan branch sources))
			(define ordered_sources (join_optimizer_sources_for_order
				sources (join_optimizer_tree_aliases tree)))
			(define default_alias (qassoc_get (qb_facts branch)
				(quote default_alias) (source_alias (car sources))))
			(define terms (candidate_recset_branch_terms branch tree))
			(membership_merge_candidate_work_profiles
				(map ordered_sources (lambda (src)
					(begin
						(define alias (source_alias src))
						(define local_condition (combine_where_terms
							(filter terms (lambda (term)
								(begin
									(define aliases (join_hypergraph_expr_aliases
										default_alias (source_aliases sources) term))
									(or (and (equal? alias (source_alias (car ordered_sources)))
										(empty_list? aliases))
										(and (single_source? aliases) (equal? (car aliases) alias))))))
							true))
						/* Join-key columns are map work: they feed the next RecSet
						projection even though they are not visible SQL output. */
						(membership_source_work_profile src local_condition
							(list terms (query_block_first_expr branch))))))))
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
			/* UNION candidates project RecSets directly from their branches. Every
			other supported candidate carrier reads a prepared group-stage cache. */
			(list (quote membership_candidate_cache_backed)
				(not (union_block? (gs_input stage))))
			(list (quote membership_candidate_scan_invocations) (qassoc_get work (quote scan_invocations) 1))
			(list (quote membership_candidate_filter_columns) (qassoc_get work (quote filter_columns) 0))
			(list (quote membership_candidate_map_columns) (qassoc_get work (quote map_columns) (count (gs_keys stage))))
			(list (quote membership_candidate_expression_operations) (qassoc_get work (quote expression_operations) 0))
			(list (quote membership_candidate_expression_depth) (qassoc_get work (quote expression_depth) 0))
			(list (quote membership_candidate_index_filter_rows) (qassoc_get work (quote input_rows) nil))
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

/* Count row-rejecting relations which remain after the membership carrier.
This fact must be captured while the complete logical source set is still
available: physical membership rewriting removes the candidate relation and
probe promotion may replace a correlated LEFT JOIN by an expression marker.
Each surviving non-membership relation is one downstream probe unit; nested
work remains part of that relation's calibrated per-probe cost. Sibling
membership stages are alternatives within the same canonical predicate and
are already represented by membership_candidate_probe_branches. Counting them
again here would make one carrier choice recursively tax the others.
Projection-only LEFT JOINs do not reject rows and therefore do not affect the
carrier crossover. */
(define membership_downstream_probe_branches (lambda (stage driver sources block)
	(if (nil? driver)
		0
		(begin
			(define default_alias (qassoc_get (qb_facts block)
				(quote default_alias) (source_alias driver)))
			(count (filter (coalesceNil sources '()) (lambda (src)
				(begin
					(define relation (source_relation src))
					(define downstream_stage (if (stage_output_relation? relation)
						(stage_by_id (qb_stages block) (stage_output_relation_id relation)) nil))
					(define downstream_purpose (if (group_stage? downstream_stage)
						(qassoc_get (gs_facts downstream_stage) (quote purpose) nil) nil))
					(and (not (equal? (source_alias src) (source_alias driver)))
						(and (not (and (stage_output_relation? relation)
							(equal? (stage_output_relation_id relation) (gs_id stage))))
							(and (not (contains? (list (quote in_membership) (quote in_candidate))
								downstream_purpose))
								(or (not (source_outer? src))
									(expr_refs_alias? default_alias (source_alias src)
										(qb_where block)))))))))))))))

(define candidate_reorder_telemetry (lambda (stage sources block planning_session tx)
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
				(membership_driver_local_filter driver sources block) 512 tx planning_session)))
		(define driver_rows (membership_estimated_work_rows driver_estimate driver_input_rows))
		(define candidate_rows (planner_stage_input_rows (gs_input stage)))
		(define candidate_probe_branches (if (union_block? (gs_input stage))
			(count (union_branches (gs_input stage)))
			1))
		(define candidate_estimate (planner_stage_filter_estimate
			(gs_input stage) 512 tx planning_session))
		(define estimate_matching_rows (qassoc_get candidate_estimate (quote rows) nil))
		(define estimate_rows (qassoc_get candidate_estimate (quote estimated_rows)
			estimate_matching_rows))
		(define estimate_capped (qassoc_get candidate_estimate (quote capped) false))
		(define estimate_input (qassoc_get candidate_estimate (quote input) nil))
		(define estimate_sampled (qassoc_get candidate_estimate (quote sampled) nil))
		(define estimate_population (planner_estimate_population candidate_estimate))
		(define estimate_coverage (planner_estimate_coverage candidate_estimate))
		/* estimated_rows has already been scaled to the full candidate input.
		Classify it against that same population; comparing it with sampled rows
		mixes units and labels most capped selective estimates as broad. */
		(define estimate_ratio_broad (and
			(and (number? estimate_rows) (and (number? candidate_rows) (> candidate_rows 0)))
			(>= (* estimate_rows 4) candidate_rows)))
		(define driver_alternative (membership_expr_has_driver_alternative? (qb_where block)))
		(define class (if (or estimate_ratio_broad
			(if driver_alternative (candidate_stage_broad? stage planning_session) false)) (quote broad) (quote selective)))
		/* Preserve the exact logical decision previously made from the planning
		session. Physical preparation no longer has implicit access to that scope. */
		(define base_sources (filter (qb_sources block) source_is_base_table?))
		(define broad_driver_probe_preferred (and (source_is_base_table? (gs_input stage))
			(and (single_source? base_sources)
				(and (query_limit_active? (qb_offset block) (qb_limit block))
					(and (order_items_belong_to_source? (car base_sources) (qb_order block))
						(expr_contains_broad_text_match?
							(coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true)
							planning_session))))))
		(define ordered_driver (if (nil? driver) false
			(or (source_order_limit_driver? driver (qb_order block) (qb_limit block))
				(source_row_number_limit_driver? (qb_stages block) driver))))
		(define consumer (if (query_block_has_aggregates? block) (quote aggregate)
			(if ordered_driver (quote order_limit) (quote filter))))
		(define candidate_work (membership_candidate_work_profile stage))
		/* Index hooks narrow the rows which reach the residual predicate but do not
		replace it. Cost filter callbacks and text bytes over that conservative
		candidate bound, not over the complete source. Keep the logical cardinality
		separate: an approximate hook upper bound is physical work, not proof that
		those rows satisfy LIKE/MATCH. */
		(define index_filter_rows (if (and
			(equal? estimate_population (quote index_hook_candidates))
			(and (number? estimate_rows) (number? candidate_rows)))
			(min candidate_rows estimate_rows)
			candidate_rows))
		(define index_filter_fraction (if (and (number? candidate_rows) (> candidate_rows 0))
			(min 1 (/ index_filter_rows candidate_rows)) 1))
		(define driver_work (membership_driver_work_profile driver sources block))
		(list
			(list (quote membership_stage_id) (gs_id stage))
			(list (quote membership_selectivity_class) class)
			(list (quote membership_broad_driver_probe_preferred) broad_driver_probe_preferred)
			(list (quote membership_driver_rows) driver_rows)
			(list (quote membership_local_filter_rows) driver_rows)
			(list (quote membership_driver_input_rows) driver_input_rows)
			(list (quote membership_driver_estimate_capped)
				(qassoc_get driver_estimate (quote capped) false))
			(list (quote membership_candidate_input_rows) candidate_rows)
			(list (quote membership_candidate_estimated_rows) estimate_rows)
			(list (quote membership_candidate_matching_rows) estimate_matching_rows)
			(list (quote membership_candidate_estimate_capped) estimate_capped)
			(list (quote membership_candidate_estimate_input) estimate_input)
			(list (quote membership_candidate_estimate_sampled) estimate_sampled)
			(list (quote membership_candidate_estimate_population) estimate_population)
			(list (quote membership_candidate_estimate_coverage) estimate_coverage)
			(list (quote membership_candidate_probe_branches) candidate_probe_branches)
			(list (quote membership_downstream_probe_branches)
				(membership_downstream_probe_branches stage driver sources block))
			(list (quote membership_candidate_scan_invocations) (qassoc_get candidate_work (quote scan_invocations) candidate_probe_branches))
			(list (quote membership_candidate_filter_columns) (qassoc_get candidate_work (quote filter_columns) 0))
			(list (quote membership_candidate_map_columns) (qassoc_get candidate_work (quote map_columns) (count (gs_keys stage))))
			(list (quote membership_candidate_cache_map_columns) (+ (count (gs_keys stage)) (count (gs_aggregates stage))))
			(list (quote membership_candidate_cache_backed) (not (union_block? (gs_input stage))))
			(list (quote membership_candidate_expression_operations) (qassoc_get candidate_work (quote expression_operations) 0))
			(list (quote membership_candidate_expression_depth) (qassoc_get candidate_work (quote expression_depth) 0))
			(list (quote membership_candidate_index_filter_rows) index_filter_rows)
			(list (quote membership_candidate_broad_text_match_rows)
				(* (qassoc_get candidate_work (quote broad_text_match_rows) 0) index_filter_fraction))
			(list (quote membership_candidate_broad_text_match_bytes)
				(* (qassoc_get candidate_work (quote broad_text_match_bytes) 0) index_filter_fraction))
			(list (quote membership_candidate_filter_value_rows)
				(* index_filter_fraction (coalesceNil (qassoc_get candidate_work (quote filter_value_rows) nil)
					(if (number? candidate_rows)
						(* candidate_rows (qassoc_get candidate_work (quote filter_columns) 0)) 0))))
			(list (quote membership_candidate_expression_operation_rows)
				(* index_filter_fraction (coalesceNil (qassoc_get candidate_work (quote expression_operation_rows) nil)
					(if (number? candidate_rows)
						(* candidate_rows (qassoc_get candidate_work (quote expression_operations) 0)) 0))))
			(list (quote membership_driver_scan_invocations) (qassoc_get driver_work (quote scan_invocations) (if (nil? driver) 0 1)))
			(list (quote membership_driver_filter_columns) (qassoc_get driver_work (quote filter_columns) 0))
			(list (quote membership_driver_map_columns) (qassoc_get driver_work (quote map_columns) 0))
			(list (quote membership_driver_expression_operations) (qassoc_get driver_work (quote expression_operations) 0))
			(list (quote membership_driver_expression_depth) (qassoc_get driver_work (quote expression_depth) 0))
			(list (quote membership_driver_alternative) driver_alternative)
			(list (quote membership_order_limit) (planner_literal_value (qb_limit block) planning_session))
			(list (quote membership_order_offset)
				(coalesceNil (planner_literal_value (qb_offset block) planning_session) 0))
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

/* Probe lowering can consume a stage far from the query block that discovered
it. Stamping the discovering block's catalog onto the immutable stage lets
lower_group_stage_prepare_using resolve nested dependencies without threading
a second global catalog through every physical expression helper. */
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

/* A driver-side subscan can evaluate UNION branches lazily only when every
branch is one base-table probe. Joined branches instead expose a complete
relational RecSet carrier; choosing a subscan for them would make the later
consumer rebuild that relation once per driver row. */
(define membership_driver_subscan_supported? (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(reduce (union_branches input) (lambda (supported branch)
				(and supported
					(and (query_block? branch)
						(and (single_real_source? (qb_sources branch))
							(source_is_base_table? (single_real_source (qb_sources branch))))))) true)
			(if (query_block? input)
				(and (single_real_source? (qb_sources input))
					(source_is_base_table? (single_real_source (qb_sources input))))
				(source_is_base_table? input))))))

(define driver_membership_probe_expr_for_strategy (lambda (stage probe strategy)
	(if (and (equal? strategy (quote driver_filter_join_probe))
		(membership_driver_subscan_supported? stage))
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
(define planner_estimated_matching_rows (lambda (estimate input_rows fallback)
	(if (nil? estimate)
		fallback
		(begin
			(define rows (qassoc_get estimate (quote rows) nil))
			(define sampled_input (qassoc_get estimate (quote sampled) nil))
			(define coverage (planner_estimate_coverage estimate))
			/* A random-shard estimate is a population sample even when the shard was
			read to completion. Scale that sample exactly once here. Zero observed
			matches retain the expression prior because one shard cannot prove that
			the complete table is empty for the predicate. */
			(if (and (equal? coverage (quote sampled))
				(and (number? input_rows)
					(and (number? rows) (and (number? sampled_input) (> sampled_input 0)))))
				(if (and (equal? rows 0)
					(number? (qassoc_get estimate (quote fallback_selectivity) nil)))
					(* input_rows (qassoc_get estimate (quote fallback_selectivity) 1))
					(min input_rows (* input_rows (/ rows sampled_input))))
				(if (and (qassoc_get estimate (quote capped) false)
					(and (number? input_rows)
						(and (number? rows) (and (number? sampled_input) (> sampled_input 0)))))
					(if (equal? coverage (quote lower_bound))
						(begin
							(define fallback_selectivity
								(qassoc_get estimate (quote fallback_selectivity) nil))
							(if (number? fallback_selectivity)
								(min input_rows (max rows (* input_rows fallback_selectivity)))
								(coalesceNil fallback input_rows)))
						(min input_rows (* input_rows (/ rows sampled_input))))
					(coalesceNil rows fallback)))))))

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
				(define local_filter_rows
					(membership_work_value work (quote membership_local_filter_rows) driver_input_rows))
				(define local_filter_density (if (> driver_input_rows 0)
					(min 1 (/ local_filter_rows driver_input_rows)) 0))
				(define density (*
					(membership_candidate_density candidate_input_rows candidate_rows work)
					local_filter_density))
				(define requested_rows (+ driver_rows
					(membership_work_value work (quote membership_order_offset) 0)))
				(if (> density 0)
					(min driver_input_rows (/ requested_rows density))
					driver_input_rows))))))

(define membership_common_scan_cost (lambda (candidate_input_rows candidate_rows driver_rows candidate_map_columns ordered_scan_invocations work)
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
			(+ (* scan_invocations planner_membership_scan_invocation_ns)
				(* ordered_scan_invocations
					planner_membership_ordered_scan_invocation_ns))
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

(define membership_integer_log2_ceil_from (lambda (rows power bits)
	(if (>= power rows)
		bits
		(membership_integer_log2_ceil_from rows (* power 2) (+ bits 1)))))

(define membership_ordered_recset_sort_work (lambda (rows)
	(if (<= rows 1)
		rows
		(* rows (membership_integer_log2_ceil_from rows 1 0)))))

/* Once an exact target RecSet exists, storage adaptively chooses between
sorting its inverse base-index positions and walking the ordered base-index
prefix with membership checks. These are equivalent kernels of one scan
operator, not planner alternatives. The carrier planner still needs their
minimum estimated cost when comparing carrier construction with RHS probes or
batching. A statistics-dependent change of that outer inequality is already
owned by the membership-carrier guard; do not create another consumer guard. */
(define membership_adaptive_ordered_consumer (lambda (candidate_input_rows candidate_rows driver_rows projected_rows work)
	(if (not (membership_work_value work (quote membership_order_limit_driver) false))
		(list projected_rows 0 0 (quote unordered_recset))
		(begin
			(define visited_rows (membership_expected_driver_rows_visited
				candidate_input_rows candidate_rows driver_rows work))
			(define sort_work (membership_ordered_recset_sort_work projected_rows))
			(define inverse_cost (* sort_work
				planner_membership_ordered_recset_sort_unit_ns))
			(define base_cost (* visited_rows (+
				planner_membership_scan_row_ns
				planner_membership_recset_probe_row_ns)))
			(if (< inverse_cost base_cost)
				(list projected_rows sort_work 0 (quote ordered_inverse_recset))
				(list visited_rows 0 visited_rows (quote ordered_base_membership)))))))

(define membership_projection_cost (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		/* FK projection must visit the target relation even when the downstream
		ordered consumer accepts only a small LIMIT window. */
		(define projection_rows (membership_driver_input_rows driver_rows work))
		(define projected_rows (membership_projected_driver_rows
			candidate_input_rows candidate_rows projection_rows work))
		(define candidate_cache_cost (if
			(membership_work_value work (quote membership_candidate_cache_backed) false)
			(planner_cost planner_membership_group_cache_startup_ns 0 0 0 0
				(* candidate_rows planner_membership_group_cache_build_row_ns)
				(* candidate_rows 8) 0 candidate_rows 0.65)
			(planner_zero_cost candidate_rows 0.65)))
		(define adaptive_consumer (membership_adaptive_ordered_consumer
			candidate_input_rows candidate_rows driver_rows projected_rows work))
		(define consumer_work_rows (nth adaptive_consumer 0))
		(define consumer_sort_work (nth adaptive_consumer 1))
		(define consumer_probe_rows (nth adaptive_consumer 2))
		(define adaptive_consumer_cost (planner_cost 0
			(* consumer_sort_work planner_membership_ordered_recset_sort_unit_ns)
			(* consumer_probe_rows planner_membership_recset_probe_row_ns)
			0 0 0 0 0 consumer_work_rows 0.75))
		(define base_cost (planner_cost_add (planner_cost_add
			(planner_cost_add
				(membership_common_scan_cost candidate_input_rows candidate_rows consumer_work_rows
					(membership_work_value work (quote membership_candidate_map_columns) 1)
					(membership_work_value work (quote membership_ordered_scan_invocations)
						(if (membership_work_value work (quote membership_order_limit_driver) false) 1 0))
					work)
				(planner_cost 0
					(+
						(* (membership_work_value work (quote membership_candidate_broad_text_match_rows) 0)
							planner_membership_broad_text_match_row_ns)
						(* (membership_work_value work (quote membership_candidate_broad_text_match_bytes) 0)
							planner_membership_broad_text_match_byte_ns))
					0 0 0 0 0 0 candidate_input_rows 0.55)
				candidate_input_rows 0.55)
			(planner_cost planner_membership_recset_startup_ns 0 0
				0 0 (* (+ candidate_rows projected_rows) planner_membership_recset_build_row_ns)
				(* (+ candidate_rows projected_rows) 8) 0 projection_rows 0.65)
			projection_rows 0.65)
			candidate_cache_cost projection_rows 0.65))
		(define downstream_cost (planner_membership_downstream_probe_cost
			(* projected_rows
				(membership_work_value work (quote membership_downstream_probe_branches) 0))))
		(define carrier_cost (planner_cost_add
			(planner_cost_add base_cost adaptive_consumer_cost projected_rows 0.65)
			downstream_cost projected_rows 0.65))
		(if (equal? (membership_work_value work (quote membership_consumer) (quote filter))
			(quote aggregate))
			(planner_cost_add carrier_cost
				(planner_cost 0 (* projection_rows planner_membership_recset_aggregate_row_ns)
					0 0 0 0 0 0 projection_rows 0.65)
				projection_rows 0.65)
			carrier_cost))))

/* A selective, source-local driver predicate can restrict both sides of the
membership edge before an expensive candidate predicate runs. The emitted
tree scans the driver once, projects its exact RecSet to the candidate source,
filters only that subset, projects matches back, and intersects with the
original driver RecSet. Independent downstream predicates are planned for the
complete local driver subset before that intersection, so candidate density
must not discount their work. Cost the exact emitted order from the same
calibrated components used by the other membership carriers. */
(define membership_prefiltered_candidate_cost (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		(define driver_input_rows (membership_driver_input_rows driver_rows work))
		(define branches (max 1
			(membership_work_value work (quote membership_candidate_probe_branches) 1)))
		(define candidate_domain_rows (/ candidate_input_rows branches))
		(define projected_candidate_rows (min driver_rows candidate_domain_rows))
		(define candidate_work_rows (* projected_candidate_rows branches))
		(define candidate_density (membership_candidate_density
			candidate_input_rows candidate_rows work))
		(define candidate_match_rows (* candidate_work_rows candidate_density))
		(define candidate_fraction (if (> candidate_input_rows 0)
			(min 1 (/ candidate_work_rows candidate_input_rows)) 0))
		(define projection_rows (+
			(* driver_rows 2)
			candidate_work_rows
			candidate_match_rows))
		(planner_cost_add (planner_cost
			(+ planner_membership_recset_startup_ns
				(* (+
					(membership_work_value work (quote membership_driver_scan_invocations) 1)
					(membership_work_value work (quote membership_candidate_scan_invocations) branches))
					planner_membership_scan_invocation_ns)
				(if (membership_work_value work (quote membership_order_limit_driver) false)
					planner_membership_ordered_scan_invocation_ns 0))
			(+
				(* (+ driver_input_rows candidate_work_rows projection_rows)
					planner_membership_scan_row_ns)
				(* driver_input_rows
					(membership_work_value work (quote membership_driver_filter_columns) 0)
					planner_membership_filter_column_row_ns)
				(* driver_input_rows
					(membership_work_value work (quote membership_driver_expression_operations) 0)
					planner_membership_expression_operation_row_ns)
				(* candidate_work_rows
					(membership_work_value work (quote membership_candidate_filter_columns) 0)
					planner_membership_filter_column_row_ns)
				(* candidate_work_rows
					(membership_work_value work (quote membership_candidate_expression_operations) 0)
					planner_membership_expression_operation_row_ns)
				(* projection_rows planner_membership_map_column_row_ns)
				(* (membership_work_value work (quote membership_candidate_broad_text_match_rows) 0)
					candidate_fraction planner_membership_broad_text_match_row_ns)
				(* (membership_work_value work (quote membership_candidate_broad_text_match_bytes) 0)
					candidate_fraction planner_membership_broad_text_match_byte_ns))
			0 0 0
			(* projection_rows planner_membership_recset_build_row_ns)
			(* projection_rows 8)
			0 driver_rows 0.65)
			(planner_membership_downstream_probe_cost
				(* driver_rows
					(membership_work_value work (quote membership_downstream_probe_branches) 0)))
			driver_rows 0.65))))

(define membership_driver_probe_cost (lambda (driver_rows probe_branches downstream_probe_branches)
	(begin
		(define probes (* driver_rows probe_branches))
		/* A driver membership check lowers each candidate branch to an indexed
		point-presence probe. Keep that storage subscan distinct from the ordered
		candidate-key index calibrated below. */
		(planner_cost_add
			(planner_membership_direct_probe_cost probes)
			(planner_membership_downstream_probe_cost
				(* driver_rows downstream_probe_branches))
			driver_rows 0.75))))

(define membership_ordered_driver_probe_cost (lambda (candidate_input_rows candidate_rows driver_rows work)
	(begin
		(define visited_rows (membership_expected_driver_rows_visited
			candidate_input_rows candidate_rows driver_rows work))
		(define driver_input_rows (membership_driver_input_rows driver_rows work))
		(define cache_backed
			(membership_work_value work (quote membership_candidate_cache_backed) false))
		(define base_cost (planner_cost_add
			(membership_common_scan_cost candidate_input_rows candidate_rows visited_rows
				(membership_work_value work
					(if cache_backed (quote membership_candidate_cache_map_columns)
						(quote membership_candidate_map_columns))
					(if cache_backed 2 1))
				(if (membership_work_value work (quote membership_order_limit_driver) false) 1 0)
				work)
			(if cache_backed
				(planner_cost planner_membership_group_cache_startup_ns 0
					(* visited_rows planner_membership_group_cache_probe_row_ns)
					0 0 (* candidate_rows planner_membership_group_cache_build_row_ns)
					(* candidate_rows 8) 0 (+ candidate_rows visited_rows) 0.65)
				(planner_cost planner_membership_recset_startup_ns 0
					(* visited_rows planner_membership_recset_probe_row_ns)
					0 0 (* candidate_rows planner_membership_recset_build_row_ns)
					(* candidate_rows 8) 0 (+ candidate_rows visited_rows) 0.65))
			visited_rows 0.65))
		(define carrier_cost (planner_cost_add base_cost
			(planner_membership_downstream_probe_cost
				(* visited_rows
					(membership_work_value work (quote membership_downstream_probe_branches) 0)))
			visited_rows 0.65))
		(if (and (membership_work_value work (quote membership_order_limit_driver) false)
			(not (membership_work_value work (quote membership_driver_alternative) false)))
			(planner_cost_add carrier_cost
				(planner_cost 0
					(* (/ (* driver_input_rows driver_input_rows) 1000000)
						planner_membership_ordered_driver_input_row_ns)
					0 0 0 0 0 0 driver_input_rows 0.65)
				visited_rows 0.65)
			carrier_cost))))

(define membership_cost_options (lambda (candidate_input_rows candidate_rows driver_rows probe_branches driver_strategy work)
	(if (or (nil? candidate_input_rows) (or (nil? candidate_rows) (nil? driver_rows)))
		'()
		(list
			(list (quote candidate_keyset) (membership_projection_cost candidate_input_rows candidate_rows driver_rows work))
			(list driver_strategy
				(if (equal? driver_strategy (quote driver_order_membership_probe))
					(membership_ordered_driver_probe_cost candidate_input_rows candidate_rows driver_rows work)
					(membership_driver_probe_cost driver_rows probe_branches
						(membership_work_value work (quote membership_downstream_probe_branches) 0))))))))

(define membership_cost_options_for_telemetry (lambda (telemetry)
	(begin
		/* candidate_estimated_rows is the shared cardinality result. Reapplying
		planner_estimated_matching_rows here would treat a UNION's merged raw
		sample as a fresh table sample and can inflate a selective path to 100%. */
		(define candidate_rows (qassoc_get telemetry
			(quote membership_candidate_estimated_rows)
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

/* The reorder phase carries only an abstract membership requirement. This
preparation step attaches its estimates to the stage, but deliberately does
not choose a carrier: only the consuming physical tree edge knows whether an
ordered batch is executable and what its actual driver workload is. */
(define query_block_with_physical_requirement_facts (lambda (block)
	(begin
		(define requirement (qassoc_get (qb_facts block) (quote membership_requirement) nil))
		(if (nil? requirement)
			block
			(begin
				(define candidates (membership_cost_options_for_telemetry requirement))
				(define physical_facts (merge (list
					(list
						(list (quote membership_plan_strategy)
							(quote projected_membership_alternatives))
						(list (quote membership_cost_candidates) candidates))
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
						(or (equal? (qassoc_get (gs_facts stage)
							(quote membership_broad_driver_probe_preferred) false) true)
							(expr_contains_broad_text_match? condition nil)))))))))

(define ordered_batch_stage_supported? (lambda (stage)
	(begin
		(define input (gs_input stage))
		(if (union_block? input)
			(reduce (union_branches input) (lambda (supported branch)
				(and supported (not (nil? (recset_project_join_branch_parts branch))))) true)
			(not (nil? (membership_keyset_descriptor (list stage nil "__target" nil))))))))

(define membership_truth_cost_facts (lambda (block stage)
	(begin
		(define input (gs_input stage))
		(define sources (filter (qb_sources block) source_is_base_table?))
		(define driver (if (empty_list? sources) nil (car sources)))
		(define condition (coalesceNil (qassoc_get (gs_facts stage) (quote condition) true) true))
		(define candidate_estimate (if (source_is_base_table? input)
			(planner_source_filter_estimate input condition 512) nil))
		(define candidate_input_rows (if (source_is_base_table? input)
			(planner_source_row_count input) nil))
		(define candidate_rows (planner_estimated_matching_rows
			candidate_estimate candidate_input_rows candidate_input_rows))
		(define driver_input_rows (if (nil? driver) nil (planner_source_row_count driver)))
		(define driver_condition (membership_driver_filter (qb_where block)))
		(define driver_estimate (if (nil? driver) nil
			(planner_source_filter_estimate driver
				driver_condition 512)))
		(define driver_rows (membership_estimated_work_rows driver_estimate driver_input_rows))
		(merge (list
			(list
				(list (quote membership_candidate_input_rows) candidate_input_rows)
				(list (quote membership_candidate_estimated_rows) candidate_rows)
				(list (quote membership_candidate_matching_rows)
					(qassoc_get candidate_estimate (quote rows) nil))
				(list (quote membership_candidate_estimate_capped)
					(qassoc_get candidate_estimate (quote capped) false))
				(list (quote membership_candidate_estimate_input)
					(qassoc_get candidate_estimate (quote input) nil))
				(list (quote membership_candidate_estimate_sampled)
					(qassoc_get candidate_estimate (quote sampled) nil))
				(list (quote membership_candidate_estimate_population)
					(planner_estimate_population candidate_estimate))
				(list (quote membership_candidate_estimate_coverage)
					(planner_estimate_coverage candidate_estimate))
				(list (quote membership_driver_input_rows) driver_input_rows)
				(list (quote membership_driver_condition) driver_condition)
				(list (quote membership_driver_rows) driver_rows))
			(membership_candidate_work_facts stage)
			/* merge is right-biased: stage telemetry is authoritative over
			the reconstructed late-consumer fallback. */
			(gs_facts stage))))))

(define membership_truth_projection_preferred? (lambda (block stage _guarded_alternative)
	(begin
		(define input (gs_input stage))
		(define base_sources (filter (qb_sources block) source_is_base_table?))
		/* This predicate proves only that the abstract membership marker has a
		physical consumer. It must not inspect cardinality or choose a carrier;
		those facts are meaningful only at the consuming scan-tree edge. */
		(and (source_is_base_table? input)
			(and (single_source? base_sources)
				(empty_list? (group_stage_session_domain_keys stage)))))))

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
						/* Cost facts travel with the abstract marker; carrier selection
						remains deferred to its physical consumer. */
						(define costed_stage (group_stage_with_facts stage
							(membership_truth_cost_facts block stage)))
						/* A membership branch below OR is not implied by the complete
						predicate. Preserve that semantic placement fact for unknown-cost
						fallbacks; it is not a physical carrier decision. */
						(define guarded_stage (if guarded_alternative
							(group_stage_with_facts costed_stage
								(qassoc_set
									(if (membership_broad_driver_probe_preferred? block stage)
										(qassoc_set (gs_facts costed_stage)
											(quote guarded_broad_order_driver) true)
										(gs_facts costed_stage))
									(quote membership_branch_not_implied) true))
							costed_stage))
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

/* The join tree owns predicate placement, not a second semantic copy of WHERE.
Whenever physical choice consumes a logical membership marker, rewrite the
placed predicate expression at the same phase boundary. Keeping the stale
marker here would make the leaf lowerer execute a logical operator even though
qb_where already contains the selected physical alternative. */
(define membership_choice_rewrite_join_predicate (lambda (block predicate)
	(begin
		(define sources (qb_sources block))
		(define default_alias (qassoc_get (qb_facts block) (quote default_alias)
			(if (empty_list? sources) nil (source_alias (car sources)))))
		(define origin (join_order_pred_origin predicate))
		(define owner (join_order_pred_owner predicate))
		(define chosen_expr
			(car (choose_membership_truth_expr block (join_order_pred_expr predicate))))
		(define expr_aliases (join_hypergraph_expr_aliases
			default_alias (source_aliases sources) chosen_expr))
		(define aliases (if (and (not (equal? origin (quote where))) (not (nil? owner)))
			(merge_unique (list expr_aliases (list owner)))
			expr_aliases))
		(define local_source (if (single_source? aliases)
			(join_optimizer_source_by_alias sources (car aliases)) nil))
		(define barrier_owner (if (and (equal? origin (quote where))
			(and (not (nil? local_source)) (source_outer? local_source)))
			(source_alias local_source) nil))
		(list
			aliases
			(join_order_pred_selectivity predicate)
			origin
			owner
			chosen_expr
			barrier_owner
			(join_order_pred_selectivity_expr predicate)))))

(define membership_choice_rewrite_join_facts (lambda (block facts removed_aliases)
	(begin
		(define reduced (join_optimizer_facts_without_aliases facts removed_aliases))
		(define tree (qassoc_get reduced (quote join_plan) nil))
		(if (nil? tree)
			reduced
			(begin
				(define predicates (map (join_optimizer_tree_predicates tree)
					(lambda (predicate)
						(membership_choice_rewrite_join_predicate block predicate))))
				(qassoc_set reduced (quote join_plan)
					(join_order_tree_with_predicates tree predicates)))))))

(define query_block_with_physical_membership_choices (lambda (block)
	(if (not (expr_contains_membership_truth? (qb_where block)))
		/* Keep unrelated query blocks byte-for-byte unchanged. A canonical
		membership marker itself is the capability signal; a preceding physical
		choice may already have rewritten its stage catalog, so purpose metadata
		must never gate the final marker-elimination walk. */
		block
		(begin
			(define chosen (choose_membership_truth_expr block (qb_where block)))
			(define removed_aliases (nth chosen 1))
			(define chosen_where (nth chosen 0))
			/* This pass replaces logical membership truth with an abstract driver
			marker and removes the redundant relational stage-output source. It must
			not choose or delete the stage for a physical carrier: derived row-number,
			join-leaf, and ordered consumers have different executable alternatives. */
			(define strategy_facts (qassoc_set (qb_facts block)
				(quote membership_plan_strategy) (quote projected_membership_alternatives)))
			(define chosen_facts
				(membership_choice_rewrite_join_facts block strategy_facts removed_aliases))
			(if (empty_list? removed_aliases)
				(make_query_block
					(qb_schema block) (qb_sources block) (qb_fields block) chosen_where
					(qb_group block) (qb_having block) (qb_order block) (qb_limit block)
					(qb_offset block) (qb_hidden block) (qb_stages block) chosen_facts)
				(make_query_block
					(qb_schema block)
					(filter (qb_sources block) (lambda (src) (not (contains? removed_aliases (source_alias src)))))
					(qb_fields block) chosen_where (qb_group block) (qb_having block)
					(qb_order block) (qb_limit block) (qb_offset block) (qb_hidden block)
					(qb_stages block)
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

/* This reorder requirement represents relational existence only. Scalar
boolean values, even in a truth context, retain their scalar cardinality and
are costed by scalar_first_probe_physical_operator at their consuming node. */
(define stage_recset_domain_eligible? (lambda (graph stage requested_col)
	(if (not (group_stage? stage))
		false
		(begin
			(define lookup_keys (qassoc_get (gs_facts stage) (quote lookup-keys) '()))
			(define input (gs_input stage))
			(and
				(not (stage_has_residual_outer_refs? stage))
				(presence_probe_stage? stage)
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

(define candidate_recset_inner_tree? (lambda (tree)
	(match tree
		((symbol join-leaf) _alias) true
		((quote join-leaf) _alias) true
		((symbol join-leaf) _alias _predicates) true
		((quote join-leaf) _alias _predicates) true
		((symbol join-node) kind left right _predicates)
		(and (equal? kind (quote inner))
			(and (candidate_recset_inner_tree? left) (candidate_recset_inner_tree? right)))
		((quote join-node) kind left right predicates)
		(candidate_recset_inner_tree? (make_join_optimizer_node kind left right predicates))
		_ false)))

(define candidate_recset_branch_terms (lambda (branch tree)
	(merge_unique (list
		(split_and_terms (coalesceNil (qb_where branch) true))
		(merge (map (qb_sources branch) (lambda (src)
			(split_and_terms (coalesceNil (source_join_expr src) true)))))
		(map (join_optimizer_tree_predicates tree) join_order_pred_expr)))))

(define candidate_recset_aliases_adjacent? (lambda (ordered_aliases left right)
	(match ordered_aliases
		(cons alias (cons next rest))
		(or
			(and (equal? alias left) (equal? next right))
			(or (and (equal? alias right) (equal? next left))
				(candidate_recset_aliases_adjacent? (cons next rest) left right)))
		_ false)))

(define candidate_recset_direct_edge_term? (lambda (sources left_alias right_alias term)
	(begin
		(define left_src (join_optimizer_source_by_alias sources left_alias))
		(define right_src (join_optimizer_source_by_alias sources right_alias))
		(match term
			'(op left right) (if (or (equal? op (quote equal?)) (equal? op (quote equal??)))
				(or
					(and (not (nil? (direct_column_name_for_alias left_src left)))
						(not (nil? (direct_column_name_for_alias right_src right))))
					(and (not (nil? (direct_column_name_for_alias left_src right)))
						(not (nil? (direct_column_name_for_alias right_src left)))))
				false)
			_ false))))

(define candidate_recset_branch_terms_supported? (lambda (sources default_alias ordered_aliases terms)
	(reduce terms (lambda (supported term)
		(if (not supported)
			false
			(begin
				(define aliases (join_hypergraph_expr_aliases
					default_alias (source_aliases sources) term))
				(if (<= (count aliases) 1)
					true
					(and (equal? (count aliases) 2)
						(and (candidate_recset_aliases_adjacent?
							ordered_aliases (car aliases) (cadr aliases))
							(candidate_recset_direct_edge_term?
								sources (car aliases) (cadr aliases) term))))))) true)))

(define candidate_recset_branch_edges_complete? (lambda (sources default_alias ordered_aliases terms)
	(match ordered_aliases
		(cons left (cons right rest))
		(and
			(reduce terms (lambda (found term)
				(or found
					(begin
						(define aliases (join_hypergraph_expr_aliases
							default_alias (source_aliases sources) term))
						(and (equal? (count aliases) 2)
							(and (contains? aliases left)
								(and (contains? aliases right)
									(candidate_recset_direct_edge_term?
										sources left right term))))))) false)
			(candidate_recset_branch_edges_complete?
				sources default_alias (cons right rest) terms))
		_ true)))

/* A candidate relation may be a complete inner-join chain. Its logical join
tree determines the RecSet carrier order; physical lowering scans the first
relation, projects through every equality edge, and applies each relation's
local predicates before continuing. No predicate kind (text or otherwise) is
special here. */
(define candidate_recset_branch_supported? (lambda (branch)
	(and (query_block? branch)
		(and (not (empty_list? (qb_sources branch)))
			(and (empty_list? (qb_stages branch))
				(begin
					(define sources (qb_sources branch))
					(define tree (query_block_join_plan branch sources))
					(define ordered_aliases (join_optimizer_tree_aliases tree))
					(define ordered_sources (join_optimizer_sources_for_order sources ordered_aliases))
					(define default_alias (qassoc_get (qb_facts branch)
						(quote default_alias) (source_alias (car sources))))
					(define terms (candidate_recset_branch_terms branch tree))
					(and (candidate_recset_inner_tree? tree)
						(and (reduce sources (lambda (supported src)
							(and supported (source_is_base_table? src))) true)
							(and (not (nil? (direct_column_name_for_alias
								(car (reverse ordered_sources))
								(query_block_first_expr branch))))
								(and (candidate_recset_branch_terms_supported?
									sources default_alias ordered_aliases terms)
									(candidate_recset_branch_edges_complete?
										sources default_alias ordered_aliases terms))))))))))))

(define candidate_stage_recset_supported? (lambda (stage)
	(and (group_stage? stage)
		(and (union_block? (gs_input stage))
			(reduce (union_branches (gs_input stage)) (lambda (supported branch)
				(and supported (candidate_recset_branch_supported? branch)))
				true)))))

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
		(define physical_block (query_block_with_physical_requirement_facts block))
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
(define without_source_alias (lambda (sources alias)
	(filter (coalesceNil sources '()) (lambda (src)
		(not (equal? (source_alias src) alias))))))

(define stage_output_column_for_alias (lambda (alias expr)
	(match expr
		((symbol get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(if (equal? tblvar alias) col nil)
		((quote get_column) tblvar _tbl_ignorecase col _col_ignorecase)
		(if (equal? tblvar alias) col nil)
		(cons _head tail) (reduce tail (lambda (found item)
			(if (not (nil? found)) found (stage_output_column_for_alias alias item))) nil)
		_ nil)))

(define stage_output_boolean_probe_term? (lambda (alias term)
	(match term
		((symbol coalesceNil) value false)
		(not (nil? (stage_output_column_for_alias alias value)))
		((quote coalesceNil) value false)
		(not (nil? (stage_output_column_for_alias alias value)))
		((symbol get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(equal? tblvar alias)
		((quote get_column) tblvar _tbl_ignorecase _col _col_ignorecase)
		(equal? tblvar alias)
		_ false)))

(define exists_recset_probe_term? (lambda (alias term)
	(or (stage_output_boolean_probe_term? alias term)
		(and (presence_bool_stage_output_expr? term)
			(not (nil? (stage_output_column_for_alias alias term)))))))

(define exists_recset_truth_container? (lambda (expr)
	(match expr
		(cons head _tail) (or (expr_head_and? head)
			(or (expr_head_or? head)
				(or (expr_head_not? head)
					(or (expr_head_if? head)
						(or (equal? head (quote optimize))
							(equal? head (symbol "optimize")))))))
		_ false)))

(define exists_recset_probe_column (lambda (alias expr)
	(if (exists_recset_probe_term? alias expr)
		(stage_output_column_for_alias alias expr)
		(if (exists_recset_truth_container? expr)
			(reduce (cdr expr) (lambda (found item)
				(if (not (nil? found)) found
					(exists_recset_probe_column alias item))) nil)
			nil))))

(define stage_with_primary_aggregate (lambda (stage requested_col)
	(begin
		(define selected (scalar_first_probe_aggregate stage requested_col))
		(if (nil? selected)
			stage
			(make_group_stage
				(gs_id stage) (gs_input stage) (gs_domain stage) (gs_keys stage)
				(cons selected (filter (gs_aggregates stage)
					(lambda (ag) (not (equal? ag selected)))))
				(gs_having stage) (gs_output stage) (gs_order stage)
				(gs_limit stage) (gs_offset stage) (gs_facts stage))))))

(define rewrite_exists_recset_probe_refs (lambda (alias stage probe expr)
	(if (exists_recset_probe_term? alias expr)
		(logical_membership_probe_expr
			(stage_with_primary_aggregate stage
				(stage_output_column_for_alias alias expr))
			probe)
		(match expr
			(cons head tail) (cons head (map tail (lambda (item)
				(rewrite_exists_recset_probe_refs alias stage probe item))))
			_ expr))))

(define first_driver_lookup_key (lambda (stage sources)
	(reduce (qassoc_get (gs_facts stage) (quote lookup-keys) '()) (lambda (found key)
		(if (not (nil? found))
			found
			(reduce (coalesceNil sources '()) (lambda (resolved src)
				(if (or (not (nil? resolved))
					(nil? (direct_column_name_for_alias src key)))
					resolved key)) nil))) nil)))

(define first_exists_recset_source (lambda (stages block condition)
	(begin
		(define sources (qb_sources block))
		(reduce (coalesceNil sources '()) (lambda (found src)
			(if (not (nil? found))
				found
				(begin
					(define requested_col
						(exists_recset_probe_column (source_alias src) condition))
					(if (and (not (nil? requested_col))
						(recset_domain_stage_output_source? stages src requested_col))
						(begin
							(define stage (stage_by_id stages
								(stage_output_relation_id (source_relation src))))
							(if (not (nil? (first_driver_lookup_key stage
								(without_source_alias sources (source_alias src)))))
								src nil))
						nil)))) nil))))

(define query_block_with_exists_membership_requirement (lambda (stage_catalog block planning_session tx)
	(begin
		(define sources (qb_sources block))
		(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
		(define exists_src (first_exists_recset_source
			(qb_stages block) block (qb_where block)))
		(if (nil? exists_src)
			nil
			(begin
				(define stage (group_stage_with_stage_catalog
					(stage_by_id stage_catalog
						(stage_output_relation_id (source_relation exists_src)))
					(lowering_catalog_stages stage_catalog)))
				(define probe_sources (list exists_src))
				(define rewritten_sources (rewrite_scalar_first_probe_sources_using
					(qb_stages block) sources probe_sources default_alias))
				(define driver_sources
					(without_source_alias rewritten_sources (source_alias exists_src)))
				(define candidate_telemetry
					(candidate_reorder_telemetry stage driver_sources block planning_session tx))
				(define costed_stage (group_stage_with_facts stage
					(merge (list candidate_telemetry (gs_facts stage)))))
				(define probe (first_driver_lookup_key costed_stage driver_sources))
				(define rewrite_consumer (lambda (expr)
					(rewrite_scalar_first_probe_expr
						(qb_stages block) probe_sources default_alias expr)))
				(hybrid_reorder_query_block_using stage_catalog
					(make_query_block
						(qb_schema block)
						driver_sources
						(rewrite_consumer (qb_fields block))
						(rewrite_exists_recset_probe_refs
							(source_alias exists_src) costed_stage probe (qb_where block))
						(rewrite_consumer (qb_group block))
						(rewrite_consumer (qb_having block))
						(rewrite_consumer (qb_order block))
						(qb_limit block)
						(qb_offset block)
						(rewrite_consumer (qb_hidden block))
						(candidate_stage_without_source
							(qb_stages block) (gs_id stage))
						(merge (list
							(query_block_reorder_telemetry block planning_session)
							candidate_telemetry
							(list (list (quote membership_requirement) (list
								(list (quote access) (quote membership))
								(list (quote reuse) 1))))
							(qb_facts block)))) planning_session tx))))))

(define reorder_query_block_with_candidate_strategy_using (lambda (stage_catalog block planning_session tx)
	(begin
		(define sources (qb_sources block))
		(if (and (empty_list? sources) (empty_list? (qb_stages block)))
			(hybrid_reorder_query_block_using stage_catalog block planning_session tx)
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
					(map (qb_stages block) (lambda (stage) (join_reorder_stage_using stage_catalog stage planning_session tx)))
					(qb_facts block))
				(begin
					(define candidate (first_candidate_source (qb_stages block) sources))
					(if (nil? candidate)
						(begin
							(define exists_block
								(query_block_with_exists_membership_requirement stage_catalog block planning_session tx))
							(if (nil? exists_block)
								(hybrid_reorder_query_block_using stage_catalog block planning_session tx)
								exists_block))
						(begin
							(define stage (stage_by_id stage_catalog (stage_output_relation_id (source_relation candidate))))
							(define candidate_telemetry (candidate_reorder_telemetry stage sources block planning_session tx))
							(define facts (merge (list
								(query_block_reorder_telemetry block planning_session)
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
												current_stage) planning_session tx)))
									(qb_facts block))
								facts)))))))))

(define join_reorder_node_using (lambda (stage_catalog node planning_session tx)
	(if (query_block? node)
		(reorder_query_block_with_candidate_strategy_using stage_catalog node planning_session tx)
		(if (union_block? node)
			(make_union_block
				(union_mode node)
				(map (union_branches node) (lambda (branch) (join_reorder_node_using stage_catalog branch planning_session tx)))
				(union_order node)
				(union_limit node)
				(union_offset node)
				(union_facts node))
			node))))

(define join_reorder_stage_using (lambda (stage_catalog stage planning_session tx)
	(if (group_stage? stage)
		(group_stage_with_reorder_facts
			(make_group_stage
				(gs_id stage)
				(join_reorder_node_using stage_catalog (gs_input stage) planning_session tx)
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

/* Rewriting a dependency column can in turn rename an aggregate which reads
that column. Propagate those positional interface renames through the stage DAG
until both producers and all consumers agree. */
(define propagate_stage_output_aggregate_columns (lambda (block original_stages rewritten_stages)
	(begin
		(define aggregate_col_map (stage_output_aggregate_fold_map
			block original_stages rewritten_stages))
		(if (empty_list? aggregate_col_map)
			(list block rewritten_stages)
			(begin
				(define next_stages (rewrite_stage_graph_stages
					aggregate_col_map '() rewritten_stages))
				(define block_with_stages (make_query_block
					(qb_schema block) (qb_sources block) (qb_fields block)
					(qb_where block) (qb_group block) (qb_having block)
					(qb_order block) (qb_limit block) (qb_offset block) (qb_hidden block)
					next_stages (qb_facts block)))
				(define next_block (rewrite_stage_graph_expr
					aggregate_col_map '() block_with_stages))
				(propagate_stage_output_aggregate_columns
					next_block rewritten_stages next_stages))))))

(define fold_boolean_tautologies_ir (lambda (ir)
	(begin
		(define root (ir_root ir))
		(if (not (query_block? root))
			ir
			(begin
				(define graph (stage_dependency_graph (qb_stages root)))
				(define folded_stages (map (qb_stages root) (lambda (stage)
					(fold_boolean_tautologies_stage graph stage))))
				(define folded_block (make_query_block
					(qb_schema root) (qb_sources root) (qb_fields root)
					(boolean_fold_maybe_expr (qb_where root))
					(qb_group root)
					(boolean_fold_maybe_expr (qb_having root))
					(qb_order root) (qb_limit root) (qb_offset root) (qb_hidden root)
					folded_stages (qb_facts root)))
				(define propagated (propagate_stage_output_aggregate_columns
					folded_block (qb_stages root) folded_stages))
				(make_ir (ir_kind ir) (nth propagated 0) (nth propagated 1)
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
		'(_value_expr reduce neutral finalize) (list (quote aggregate) reduce neutral finalize)
		_ (stage_semantic_rewrite_expr alias_map signatures ag))))

(define stage_semantic_facts (lambda (alias_map signatures facts)
	(map (list
		(quote purpose)
		(quote presence_only)
		(quote max_needed_per_domain)
		(quote preserve_empty_domain)
		(quote null_semantics)
		(quote partition_by)
		(quote partition_limit)
		(quote result_max_rows_per_partition)
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
			(concat "stage-backbone:" (stable_structural_hash payload true))))))

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
			(concat "stage-semantic:" (stable_structural_hash payload true))))))

(define stage_semantic_signature_index (lambda (stages)
	(reduce (coalesceNil stages '()) (lambda (index stage)
		(set_assoc index (gs_id stage) (stage_semantic_signature index stage)))
		'())))

(define stage_output_left_join_stage_key (lambda (signature_index stage)
	(if (not (group_stage? stage))
		nil
		(begin
			(define null_semantics (qassoc_get (gs_facts stage) (quote null_semantics) nil))
			(define signature (get_assoc signature_index (gs_id stage)))
			(if (and (equal? (count (gs_aggregates stage)) 1)
				(or (equal? null_semantics (quote scalar)) (equal? null_semantics (quote exists))))
				(if (equal? null_semantics (quote exists))
					/* EXISTS stages from different decorrelation parents can have the
					same value signature but live in different domain scopes. Derived
					rebinding records the containing query scope in the stage ID. */
					(concat signature ":scope:" (serialize (list
						(cdr (split (gs_id stage) ":derived:"))
						(qassoc_get (gs_facts stage) (quote btw2025_parent) nil))))
					signature)
				/* Base-table scalar stages with the same normalized domain, keys,
				condition, order, and LEFT edge share one carrier even when correlated.
				The source-join key below still retains the concrete outer alias, so
				different correlation scopes cannot collapse accidentally. */
				(if (source_is_base_table? (gs_input stage))
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
					(concat "stage-output-left-join:" (stable_structural_hash (list
						stage_key
						(source_schema src)
						(normalize_stage_output_left_join_expr (source_alias src) (source_join_expr src))) true))))))))

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
	(begin
		/* Most wide projection blocks contain many distinct scalar stages. Keep a
		key index while collecting them so the no-merge case stays linear; only an
		actual duplicate needs the aggregate-alignment walk over existing entries. */
		(define state (reduce (coalesceNil sources '()) (lambda (state src)
			(begin
				(define entries (nth state 0))
				(define keys (nth state 1))
				(define key (stage_output_left_join_key stages signature_index src))
				(if (nil? key)
					state
					(begin
						(define stage (stage_by_id stages
							(stage_output_relation_id (source_relation src))))
						(if (has_assoc? keys key)
							(list
								(map entries (lambda (entry)
									(if (equal? (stage_output_left_join_entry_key entry) key)
										(stage_output_left_join_entry_add_source entry src stage)
										entry)))
								keys)
							(list
								(append entries (stage_output_left_join_entry_for_source key src stage))
								(set_assoc keys key true)))))))
			(list '() '())))
		(nth state 0))))

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
			(define stage_lookup (make_lowering_catalog (qb_stages block)))
			(define signature_index (stage_semantic_signature_index (qb_stages block)))
			(define entries (stage_output_left_join_entries stage_lookup signature_index (qb_sources block)))
			(if (not (stage_output_left_join_entries_have_duplicates? entries))
				block
				(begin
					(define dependency_id_map (merge (map entries (lambda (entry)
						(stage_output_left_join_dependency_id_maps_for_entry stage_lookup entry)))))
					(define id_map (merge (list
						(merge (map entries stage_output_left_join_id_map_for_entry))
						dependency_id_map)))
					(define alias_map (merge (map entries stage_output_left_join_alias_map_for_entry)))
					(define column_maps (merge (map entries (lambda (entry)
						(stage_output_left_join_column_maps_for_entry stage_lookup entry)))))
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
		(define partition_id (concat "aggregate-partition:" (stable_structural_hash (list
			(source_schema driver) (source_relation driver) keys residual) true)))
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
				(list (quote partition_by) keys)
				(list (quote result_max_rows_per_partition) 1))))
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

(define join_reorder_node_stage_catalog (lambda (node)
	(if (query_block? node)
		(qb_stages node)
		(if (union_block? node)
			(unique_stages_by_id
				(merge (map (union_branches node) join_reorder_node_stage_catalog)))
			'()))))

(define join_reorder (lambda (ir planning_session tx)
	(begin
		/* A flattened decorrelation catalog grows with every independent scalar or
		UNION branch. Repeated linear stage_by_id scans made reordering those
		branches quadratic even though every individual join graph is small. Keep
		the logical list in the IR, but use the compile-local indexed view for all
		reorder lookups. */
		(define stages (ir_stages ir))
		(define stage_catalog (if (<= (count stages) 4)
			stages
			(make_indexed_lowering_catalog stages nil)))
		(define reordered_root
			(join_reorder_node_using stage_catalog (ir_root ir) planning_session tx))
		(make_ir
			(ir_kind ir)
			reordered_root
			(join_reorder_node_stage_catalog reordered_root)
			(ir_context_of ir)
			(ir_return ir)))))

/* ------------------------------------------------------------------------- */
